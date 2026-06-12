package status

import (
	"fmt"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/hooks"
	"github.com/sahil87/fab-kit/src/go/fab/internal/log"
	sf "github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// Valid change types.
var ValidChangeTypes = []string{"feat", "fix", "refactor", "docs", "test", "ci", "chore"}

// Valid states.
var ValidStates = []string{"pending", "active", "ready", "done", "failed", "skipped"}

// Allowed states per stage.
var AllowedStates = map[string][]string{
	"intake":    {"active", "ready", "done"},
	"apply":     {"pending", "active", "ready", "done", "skipped"},
	"review":    {"pending", "active", "ready", "done", "failed", "skipped"},
	"hydrate":   {"pending", "active", "ready", "done", "skipped"},
	"ship":      {"pending", "active", "done", "skipped"},
	"review-pr": {"pending", "active", "done", "failed", "skipped"},
}

// Transition defines a state machine transition.
type Transition struct {
	From []string
	To   string
}

// Default transitions (applicable to all stages unless overridden).
var defaultTransitions = map[string]Transition{
	"start":   {From: []string{"pending"}, To: "active"},
	"advance": {From: []string{"active"}, To: "ready"},
	"finish":  {From: []string{"active", "ready"}, To: "done"},
	"reset":   {From: []string{"done", "ready", "skipped"}, To: "active"},
	"skip":    {From: []string{"pending", "active"}, To: "skipped"},
}

// Stage-specific overrides (review and review-pr add fail and start-from-failed).
var stageTransitions = map[string]map[string]Transition{
	"review": {
		"start":   {From: []string{"pending", "failed"}, To: "active"},
		"advance": {From: []string{"active"}, To: "ready"},
		"finish":  {From: []string{"active", "ready"}, To: "done"},
		"reset":   {From: []string{"done", "ready", "skipped"}, To: "active"},
		"fail":    {From: []string{"active"}, To: "failed"},
	},
	"review-pr": {
		"start":   {From: []string{"pending", "failed"}, To: "active"},
		"advance": {From: []string{"active"}, To: "ready"},
		"finish":  {From: []string{"active", "ready"}, To: "done"},
		"reset":   {From: []string{"done", "ready", "skipped"}, To: "active"},
		"fail":    {From: []string{"active"}, To: "failed"},
	},
}

// lookupTransition finds the target state for an event on a stage with a given current state.
// The resolved target is validated against AllowedStates for the stage: an
// event whose target state the schema forbids (e.g. `advance ship` → ready,
// `skip intake` → skipped) errors cleanly instead of writing a state that
// permanently fails validation (and bricks `fab preflight`).
func lookupTransition(event, stage, currentState string) (string, error) {
	// Check stage-specific overrides first
	if stageT, ok := stageTransitions[stage]; ok {
		if t, ok := stageT[event]; ok {
			if contains(t.From, currentState) {
				return validateTarget(event, stage, t.To)
			}
			return "", fmt.Errorf("Cannot %s stage '%s' — current state is '%s', no valid transition", event, stage, currentState)
		}
	}

	// Check default transitions
	if t, ok := defaultTransitions[event]; ok {
		if contains(t.From, currentState) {
			return validateTarget(event, stage, t.To)
		}
	}

	return "", fmt.Errorf("Cannot %s stage '%s' — current state is '%s', no valid transition", event, stage, currentState)
}

// validateTarget rejects transitions whose target state is not in the
// stage's AllowedStates.
func validateTarget(event, stage, target string) (string, error) {
	if allowed, ok := AllowedStates[stage]; ok && !contains(allowed, target) {
		return "", fmt.Errorf("Cannot %s stage '%s' — target state '%s' is not allowed for this stage", event, stage, target)
	}
	return target, nil
}

// validateStage returns nil if the stage is part of the current pipeline,
// the strict-error message for the removed `tasks` stage, or a generic
// "Invalid stage" error for everything else.
func validateStage(event, stage string) error {
	if isValidStage(stage) {
		return nil
	}
	if stage == "tasks" {
		return fmt.Errorf("\"tasks\" stage was removed — run \"fab status %s <change> apply\" instead. plan.md is now generated at apply entry.", event)
	}
	if stage == "spec" {
		return fmt.Errorf("\"spec\" stage was removed — spec.md is now generated at apply entry. Use \"apply\".")
	}
	return fmt.Errorf("Invalid stage '%s'", stage)
}

// Start transitions a stage from {pending,failed} to active.
// If a pre hook is configured for the stage, it runs before the transition.
// A failing pre hook blocks the stage from starting.
func Start(statusFile *sf.StatusFile, statusPath, fabRoot, stage, driver, from, reason string) error {
	if err := validateStage("start", stage); err != nil {
		return err
	}

	currentState := statusFile.GetProgress(stage)
	targetState, err := lookupTransition("start", stage, currentState)
	if err != nil {
		return err
	}

	// Run pre hook before transitioning
	if err := runStageHook(fabRoot, stage, "pre"); err != nil {
		return err
	}

	if err := statusFile.SetProgress(stage, targetState); err != nil {
		return err
	}
	applyMetricsSideEffect(statusFile, fabRoot, stage, targetState, driver, from, reason)

	return statusFile.Save(statusPath)
}

// Advance transitions a stage from active to ready.
func Advance(statusFile *sf.StatusFile, statusPath, stage, driver string) error {
	if err := validateStage("advance", stage); err != nil {
		return err
	}

	currentState := statusFile.GetProgress(stage)
	targetState, err := lookupTransition("advance", stage, currentState)
	if err != nil {
		return err
	}

	if err := statusFile.SetProgress(stage, targetState); err != nil {
		return err
	}

	return statusFile.Save(statusPath)
}

// Finish transitions a stage to done and auto-activates the next pending stage.
// If a post hook is configured for the stage, it runs after the transition.
// A failing post hook causes the stage to fail.
func Finish(statusFile *sf.StatusFile, statusPath, fabRoot, stage, driver string) error {
	if err := validateStage("finish", stage); err != nil {
		return err
	}

	currentState := statusFile.GetProgress(stage)
	targetState, err := lookupTransition("finish", stage, currentState)
	if err != nil {
		return err
	}

	if err := statusFile.SetProgress(stage, targetState); err != nil {
		return err
	}
	applyMetricsSideEffect(statusFile, fabRoot, stage, targetState, "", "", "")

	// Auto-activate next pending stage
	nextStage := sf.NextStage(stage)
	if nextStage != "" {
		nextState := statusFile.GetProgress(nextStage)
		if nextState == "pending" {
			if err := statusFile.SetProgress(nextStage, "active"); err != nil {
				return err
			}
			applyMetricsSideEffect(statusFile, fabRoot, nextStage, "active", driver, "", "")
		}
	}

	if err := statusFile.Save(statusPath); err != nil {
		return err
	}

	// Compute and write true_impact for apply/hydrate finish (best-effort).
	_ = WriteTrueImpact(statusFile, statusPath, fabRoot, stage)

	// Run post hook after transition is saved
	if err := runStageHook(fabRoot, stage, "post"); err != nil {
		return err
	}

	// Auto-log review/review-pr pass
	if stage == "review" || stage == "review-pr" {
		_ = log.Review(fabRoot, statusFile.Name, "passed", "")
	}

	return nil
}

// Reset transitions a stage to active and cascades downstream to pending.
func Reset(statusFile *sf.StatusFile, statusPath, fabRoot, stage, driver, from, reason string) error {
	if err := validateStage("reset", stage); err != nil {
		return err
	}

	currentState := statusFile.GetProgress(stage)
	targetState, err := lookupTransition("reset", stage, currentState)
	if err != nil {
		return err
	}

	if err := statusFile.SetProgress(stage, targetState); err != nil {
		return err
	}
	applyMetricsSideEffect(statusFile, fabRoot, stage, targetState, driver, from, reason)

	// Cascade downstream to pending
	foundTarget := false
	for _, s := range sf.StageOrder {
		if foundTarget {
			if err := statusFile.SetProgress(s, "pending"); err != nil {
				return err
			}
			applyMetricsSideEffect(statusFile, fabRoot, s, "pending", "", "", "")
		}
		if s == stage {
			foundTarget = true
		}
	}

	return statusFile.Save(statusPath)
}

// Skip transitions a stage to skipped and cascades downstream pending to skipped.
func Skip(statusFile *sf.StatusFile, statusPath, fabRoot, stage, driver string) error {
	if err := validateStage("skip", stage); err != nil {
		return err
	}

	currentState := statusFile.GetProgress(stage)
	targetState, err := lookupTransition("skip", stage, currentState)
	if err != nil {
		return err
	}

	if err := statusFile.SetProgress(stage, targetState); err != nil {
		return err
	}
	applyMetricsSideEffect(statusFile, fabRoot, stage, targetState, "", "", "")

	// Forward cascade: downstream pending → skipped
	foundTarget := false
	for _, s := range sf.StageOrder {
		if foundTarget {
			if statusFile.GetProgress(s) == "pending" {
				if err := statusFile.SetProgress(s, "skipped"); err != nil {
					return err
				}
				applyMetricsSideEffect(statusFile, fabRoot, s, "skipped", "", "", "")
			}
		}
		if s == stage {
			foundTarget = true
		}
	}

	return statusFile.Save(statusPath)
}

// Fail transitions a stage to failed (review/review-pr only).
func Fail(statusFile *sf.StatusFile, statusPath, fabRoot, stage, driver, rework string) error {
	if err := validateStage("fail", stage); err != nil {
		return err
	}

	currentState := statusFile.GetProgress(stage)
	targetState, err := lookupTransition("fail", stage, currentState)
	if err != nil {
		return err
	}

	if err := statusFile.SetProgress(stage, targetState); err != nil {
		return err
	}

	if err := statusFile.Save(statusPath); err != nil {
		return err
	}

	// Auto-log review/review-pr failure
	if stage == "review" || stage == "review-pr" {
		_ = log.Review(fabRoot, statusFile.Name, "failed", rework)
	}

	return nil
}

// ApplyChangeType validates and sets the change_type field on the in-memory
// StatusFile without saving. Validation happens before any mutation. Callers
// own persistence — the artifact-write hook batches several Apply* mutations
// into a single Save (mz4q F02).
func ApplyChangeType(statusFile *sf.StatusFile, changeType string) error {
	valid := false
	for _, t := range ValidChangeTypes {
		if t == changeType {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("Invalid change type '%s' (valid: %s)", changeType, strings.Join(ValidChangeTypes, ", "))
	}
	statusFile.ChangeType = changeType
	return nil
}

// SetChangeType sets the change_type field and persists.
func SetChangeType(statusFile *sf.StatusFile, statusPath, changeType string) error {
	if err := ApplyChangeType(statusFile, changeType); err != nil {
		return err
	}
	return statusFile.Save(statusPath)
}

// ApplyAcceptance updates a field on the in-memory plan: block without
// saving. Valid fields: generated (bool), task_count (int), acceptance_count
// (int), acceptance_completed (int). Validation happens before any mutation.
// Callers own persistence (see ApplyChangeType).
func ApplyAcceptance(statusFile *sf.StatusFile, field, value string) error {
	switch field {
	case "generated":
		if value != "true" && value != "false" {
			return fmt.Errorf("Invalid value '%s' for field 'generated' (expected true/false)", value)
		}
		statusFile.Plan.Generated = value == "true"
	case "task_count":
		n, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("Invalid value '%s' for field 'task_count' (expected non-negative integer)", value)
		}
		statusFile.Plan.TaskCount = n
	case "acceptance_count":
		n, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("Invalid value '%s' for field 'acceptance_count' (expected non-negative integer)", value)
		}
		statusFile.Plan.AcceptanceCount = n
	case "acceptance_completed":
		n, err := parseInt(value)
		if err != nil {
			return fmt.Errorf("Invalid value '%s' for field 'acceptance_completed' (expected non-negative integer)", value)
		}
		statusFile.Plan.AcceptanceCompleted = n
	default:
		return fmt.Errorf("Invalid plan field '%s' (expected: generated, task_count, acceptance_count, acceptance_completed)", field)
	}
	return nil
}

// SetAcceptance updates a field on the plan: block of .status.yaml and
// persists. Valid fields: generated (bool), task_count (int),
// acceptance_count (int), acceptance_completed (int).
func SetAcceptance(statusFile *sf.StatusFile, statusPath, field, value string) error {
	if err := ApplyAcceptance(statusFile, field, value); err != nil {
		return err
	}
	return statusFile.Save(statusPath)
}

// SetChecklistRemovedError returns the strict-error message for the removed
// `set-checklist` command. The command is no longer supported; callers
// should use `set-acceptance` instead. This is exposed so the Cobra layer
// can surface the same message regardless of how it is reached.
func SetChecklistRemovedError() error {
	return fmt.Errorf("\"set-checklist\" is now \"set-acceptance\" — run fab status set-acceptance instead.")
}

// ApplyConfidence replaces the in-memory confidence block without saving.
// Callers own persistence (see ApplyChangeType).
func ApplyConfidence(statusFile *sf.StatusFile, certain, confident, tentative, unresolved int, score float64) {
	statusFile.Confidence = sf.Confidence{
		Certain:    certain,
		Confident:  confident,
		Tentative:  tentative,
		Unresolved: unresolved,
		Score:      score,
	}
}

// SetConfidence replaces the confidence block and persists.
func SetConfidence(statusFile *sf.StatusFile, statusPath string, certain, confident, tentative, unresolved int, score float64) error {
	ApplyConfidence(statusFile, certain, confident, tentative, unresolved, score)
	return statusFile.Save(statusPath)
}

// ApplyConfidenceFuzzy replaces the in-memory confidence block with dimension
// data without saving. Callers own persistence (see ApplyChangeType).
func ApplyConfidenceFuzzy(statusFile *sf.StatusFile, certain, confident, tentative, unresolved int, score, meanS, meanR, meanA, meanD float64) {
	statusFile.Confidence = sf.Confidence{
		Certain:    certain,
		Confident:  confident,
		Tentative:  tentative,
		Unresolved: unresolved,
		Score:      score,
		Fuzzy:      sf.BoolPtr(true),
		Dimensions: &sf.Dimensions{
			Signal:         meanS,
			Reversibility:  meanR,
			Competence:     meanA,
			Disambiguation: meanD,
		},
	}
}

// SetConfidenceFuzzy replaces the confidence block with dimension data and
// persists.
func SetConfidenceFuzzy(statusFile *sf.StatusFile, statusPath string, certain, confident, tentative, unresolved int, score, meanS, meanR, meanA, meanD float64) error {
	ApplyConfidenceFuzzy(statusFile, certain, confident, tentative, unresolved, score, meanS, meanR, meanA, meanD)
	return statusFile.Save(statusPath)
}

// AddIssue appends an issue ID (idempotent).
func AddIssue(statusFile *sf.StatusFile, statusPath, id string) error {
	for _, existing := range statusFile.Issues {
		if existing == id {
			return statusFile.Save(statusPath) // refresh last_updated
		}
	}
	statusFile.Issues = append(statusFile.Issues, id)
	return statusFile.Save(statusPath)
}

// AddPR appends a PR URL (idempotent).
func AddPR(statusFile *sf.StatusFile, statusPath, url string) error {
	for _, existing := range statusFile.PRs {
		if existing == url {
			return statusFile.Save(statusPath)
		}
	}
	statusFile.PRs = append(statusFile.PRs, url)
	return statusFile.Save(statusPath)
}

// ProgressMap returns stage:state pairs in pipeline order.
func ProgressMap(statusFile *sf.StatusFile) []sf.StageState {
	return statusFile.GetProgressMap()
}

// ProgressLine returns a single-line visual progress string.
func ProgressLine(statusFile *sf.StatusFile) string {
	var parts []string
	hasActive := false
	hasPending := false

	for _, ss := range statusFile.GetProgressMap() {
		switch ss.State {
		case "done":
			parts = append(parts, ss.Stage)
		case "active":
			parts = append(parts, ss.Stage+" ⏳")
			hasActive = true
		case "ready":
			parts = append(parts, ss.Stage+" ◷")
		case "failed":
			parts = append(parts, ss.Stage+" ✗")
		case "skipped":
			parts = append(parts, ss.Stage+" ⏭")
		case "pending":
			hasPending = true
		}
	}

	if len(parts) == 0 {
		return ""
	}

	line := strings.Join(parts, " → ")
	if !hasActive && !hasPending {
		line += " ✓"
	}
	return line
}

// CurrentStage determines the active/next stage.
func CurrentStage(statusFile *sf.StatusFile) string {
	pm := statusFile.GetProgressMap()

	// First active or ready
	for _, ss := range pm {
		if ss.State == "active" || ss.State == "ready" {
			return ss.Stage
		}
	}

	// Fallback: first pending after last done/skipped
	lastDone := ""
	for _, ss := range pm {
		if ss.State == "done" || ss.State == "skipped" {
			lastDone = ss.Stage
		}
	}

	if lastDone != "" {
		foundLast := false
		for _, ss := range pm {
			if foundLast && ss.State == "pending" {
				return ss.Stage
			}
			if ss.Stage == lastDone {
				foundLast = true
			}
		}
	}

	return "review-pr" // all done
}

// DisplayStage returns the display stage and state as "stage:state".
func DisplayStage(statusFile *sf.StatusFile) (string, string) {
	pm := statusFile.GetProgressMap()

	// Tier 1: first active — in-progress work supersedes a parked failure.
	for _, ss := range pm {
		if ss.State == "active" {
			return ss.Stage, "active"
		}
	}

	// Tier 2: first failed — a parked failure outranks ready/done so it
	// surfaces instead of being masked by the last-done fallback. Only
	// review and review-pr can hold "failed" (AllowedStates).
	for _, ss := range pm {
		if ss.State == "failed" {
			return ss.Stage, "failed"
		}
	}

	// Tier 3: first ready
	for _, ss := range pm {
		if ss.State == "ready" {
			return ss.Stage, "ready"
		}
	}

	// Tier 4: last done/skipped
	lastDone := ""
	lastDoneState := ""
	for _, ss := range pm {
		if ss.State == "done" || ss.State == "skipped" {
			lastDone = ss.Stage
			lastDoneState = ss.State
		}
	}
	if lastDone != "" {
		return lastDone, lastDoneState
	}

	// Tier 5: first pending
	if len(sf.StageOrder) > 0 {
		return sf.StageOrder[0], "pending"
	}
	return "intake", "pending"
}

// Validate validates a .status.yaml against the schema.
func Validate(statusFile *sf.StatusFile) error {
	activeCount := 0
	var errors []string

	for _, stage := range sf.StageOrder {
		state := statusFile.GetProgress(stage)
		if state == "" {
			state = "pending"
		}

		if !isValidState(state) {
			errors = append(errors, fmt.Sprintf("Invalid state '%s' for stage %s", state, stage))
			continue
		}

		allowed, ok := AllowedStates[stage]
		if ok && !contains(allowed, state) {
			errors = append(errors, fmt.Sprintf("State '%s' not allowed for stage %s", state, stage))
		}

		if state == "active" {
			activeCount++
		}
	}

	if activeCount > 1 {
		errors = append(errors, "Multiple stages are active (expected 0 or 1)")
	}

	if len(errors) > 0 {
		return fmt.Errorf("%s", strings.Join(errors, "; "))
	}
	return nil
}

// AllStages returns all stage IDs in pipeline order.
func AllStages() []string {
	return sf.StageOrder
}

func applyMetricsSideEffect(statusFile *sf.StatusFile, fabRoot, stage, state, driver, from, reason string) {
	now := time.Now().UTC().Format(time.RFC3339)

	switch state {
	case "active":
		sm, ok := statusFile.StageMetrics[stage]
		if !ok {
			sm = &sf.StageMetric{}
			statusFile.StageMetrics[stage] = sm
		}
		sm.Iterations++
		sm.StartedAt = now
		sm.Driver = driver
		sm.CompletedAt = ""

		// Log transition (best-effort)
		folder := statusFile.Name
		action := "enter"
		if sm.Iterations > 1 {
			action = "re-entry"
		}
		_ = log.Transition(fabRoot, folder, stage, action, from, reason, driver)

	case "done":
		sm, ok := statusFile.StageMetrics[stage]
		if ok {
			sm.CompletedAt = now
		}

	case "pending", "skipped":
		// Preserve the iterations counter across reset/skip cascades: the
		// rework choreography's `fail review` + `reset apply` cascades
		// review → pending, and deleting the entry would zero the cycle
		// counter PR meta reports (stage_metrics.review.iterations,
		// "incremented, not reset — tracks rework cycles"). Timing fields
		// are cleared — the next activation rewrites them.
		if sm, ok := statusFile.StageMetrics[stage]; ok && sm.Iterations > 0 {
			sm.StartedAt = ""
			sm.Driver = ""
			sm.CompletedAt = ""
		} else {
			delete(statusFile.StageMetrics, stage)
		}
	}
}

func isValidStage(stage string) bool {
	for _, s := range sf.StageOrder {
		if s == stage {
			return true
		}
	}
	return false
}

func isValidState(state string) bool {
	for _, s := range ValidStates {
		if s == state {
			return true
		}
	}
	return false
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// runStageHook loads the project config and runs the pre or post hook for the given stage.
// Returns nil if no hook is configured or if the hook succeeds.
func runStageHook(fabRoot, stage, phase string) error {
	cfg, err := config.Load(fabRoot)
	if err != nil {
		return fmt.Errorf("failed to load config for stage hooks: %w", err)
	}

	hook := cfg.GetStageHook(stage)
	var command string
	switch phase {
	case "pre":
		command = hook.Pre
	case "post":
		command = hook.Post
	}

	return hooks.Run(fabRoot, command)
}

func parseInt(s string) (int, error) {
	if len(s) == 0 {
		return 0, fmt.Errorf("empty string")
	}
	n := 0
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0, fmt.Errorf("not a number")
		}
		n = n*10 + int(c-'0')
	}
	return n, nil
}
