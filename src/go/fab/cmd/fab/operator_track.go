package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/predicate"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// Track verbs (intake B1) — the single verb family over the operator state
// file's `tracked` list, replacing enroll/update/remove, watch *, autopilot *,
// and note * (removed outright, no aliases). The binary owns the schema, the
// timestamps, the caps, and the atomic write; the operator states intent
// through flags and never hand-writes the YAML.

func operatorTrackCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "track",
		Short: "Manage the operator state file's tracked items",
	}
	cmd.AddCommand(
		operatorTrackAddCmd(),
		operatorTrackUpdateCmd(),
		operatorTrackObserveCmd(),
		operatorTrackRmCmd(),
		operatorTrackListCmd(),
		operatorTrackClockCmd(),
	)
	return cmd
}

// trackPaneSugarFlags are the pane scope shortcut flags (the spawn step stays
// one command); the binary writes them into scope.
var trackPaneSugarFlags = []string{"pane", "repo", "session", "branch", "stage", "agent", "stop-stage", "spawned-by", "change"}

func operatorTrackAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add <id>",
		Short: "Add a tracked item (the kind fills its defaults; flags override)",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorTrackAdd,
	}
	f := cmd.Flags()
	f.String("kind", "", "item kind: pane | github-pr | linear | slack | shell | task | note (required)")
	f.String("probe", "", "probe mode (each kind allows only its default)")
	f.StringArray("argv", nil, "shell probe argv tokens (repeatable; never a shell string)")
	f.StringSlice("fields", nil, "comma-separated probe fields compared against and stored into last")
	f.String("instruction", "", "agent probe instruction (the LLM runs it, the binary never does)")
	f.String("check-every", "", "probe cadence (floor 1m; default 5m for shell/agent; forced null for pane/none)")
	f.String("done-when", "", "done predicate: <path> ==|!= <json-scalar> clauses joined by ' and '")
	f.String("then", "", "prose the operator runs when done/changed fires")
	f.StringSlice("depends-on", nil, "comma-separated item ids this item depends on")
	f.String("scope", "", "scope metadata as a JSON object")
	f.String("text", "", "note text (kind note only; 500-char cap)")
	f.String("mode", "", "merge mode for a chained pane item: "+strings.Join(config.ValidAutopilotMergeModes, " | ")+" (default: autopilot.merge_mode config, else "+config.DefaultAutopilotMergeMode+")")
	for _, name := range trackPaneSugarFlags {
		f.String(name, "", "pane scope sugar")
	}
	_ = cmd.MarkFlagRequired("kind")
	return cmd
}

func runOperatorTrackAdd(cmd *cobra.Command, args []string) error {
	id := args[0]
	f := cmd.Flags()
	kind, _ := f.GetString("kind")
	kd, ok := trackKinds[kind]
	if !ok {
		return fmt.Errorf("unknown --kind %q (valid: %s)", kind, trackKindNames)
	}
	mode, err := trackProbeMode(cmd, kind, kd)
	if err != nil {
		return err
	}
	scope, err := scopeFlag(cmd)
	if err != nil {
		return err
	}
	if kind == kindPane {
		base := paneScope()
		for k, v := range scope {
			base[k] = v
		}
		scope = base
		if err := applyPaneSugar(cmd, scope); err != nil {
			return err
		}
		recordPanePID(scope)
	} else {
		for _, name := range append(append([]string{}, trackPaneSugarFlags...), "mode") {
			if f.Changed(name) {
				return fmt.Errorf("--%s applies only to kind pane", name)
			}
		}
	}
	if scope == nil {
		scope = map[string]interface{}{}
	}

	probe := probeSpec{Mode: mode}
	switch mode {
	case probeShell:
		argv, _ := f.GetStringArray("argv")
		fields, _ := f.GetStringSlice("fields")
		if kind == kindGitHubPR {
			if !f.Changed("argv") {
				argv, err = defaultGitHubPRArgv(scope)
				if err != nil {
					return err
				}
			}
			if !f.Changed("fields") {
				fields = githubPRProbeFields
			}
		}
		probe.Argv, probe.Fields = argv, fields
	case probeAgent:
		probe.Instruction, _ = f.GetString("instruction")
	}

	checkEveryGiven, _ := f.GetString("check-every")
	checkEvery, err := normalizeCheckEvery(mode, checkEveryGiven)
	if err != nil {
		return err
	}
	doneWhen, err := doneWhenFlag(cmd, kd.doneWhen)
	if err != nil {
		return err
	}
	if kind == kindShell && doneWhen == nil {
		return fmt.Errorf("kind shell requires --done-when")
	}
	text, _ := f.GetString("text")
	if f.Changed("text") && kind != kindNote {
		return fmt.Errorf("--text applies only to kind note")
	}
	if kind == kindNote && text == "" {
		return fmt.Errorf("kind note requires --text")
	}
	var then *string
	if f.Changed("then") {
		if t, _ := f.GetString("then"); t != "" {
			then = &t
		}
	}
	dependsOn, _ := f.GetStringSlice("depends-on")

	// Merge mode resolves when a chain is added (--depends-on or --mode):
	// flag > autopilot.merge_mode config > built-in default.
	modeLine := ""
	if kind == kindPane && (f.Changed("mode") || len(dependsOn) > 0) {
		m, source, err := resolveMergeMode(cmd)
		if err != nil {
			return err
		}
		scope["merge_mode"] = m
		modeLine = fmt.Sprintf("mode: %s (%s)", m, source)
	}

	now := nowRFC3339()
	item := trackedItem{
		ID:         id,
		Kind:       kind,
		Probe:      probe,
		CheckEvery: checkEvery,
		DoneWhen:   doneWhen,
		Then:       then,
		DependsOn:  dependsOn,
		Scope:      scope,
		Last:       map[string]interface{}{},
		Text:       text,
		AddedAt:    now,
		UpdatedAt:  now,
	}
	if item.DependsOn == nil {
		item.DependsOn = []string{}
	}
	if err := validateTrackedItem(item); err != nil {
		return err
	}

	if err := mutateOperatorState(func(data map[string]interface{}) error {
		items, err := decodeTrackedItems(data)
		if err != nil {
			return err
		}
		for _, e := range items {
			if e.ID == id {
				return fmt.Errorf("tracked item %s already exists", id)
			}
		}
		if err := validateTrackedDeps(item, items); err != nil {
			return err
		}
		data["tracked"] = append(items, item)
		if kind == kindPane {
			branch, repo := scopeString(scope, "branch"), scopeString(scope, "repo")
			if branch != "" && repo != "" {
				bm := map[string]branchMapEntry{}
				if err := operatorSection(data, "branch_map", &bm); err != nil {
					return err
				}
				// branch_map is keyed by the change id: --change when given,
				// else the item id (a known-change spawn's id IS the change).
				key := scopeString(scope, "change")
				if key == "" {
					key = id
				}
				bm[key] = branchMapEntry{Branch: branch, Repo: repo}
				data["branch_map"] = bm
			}
		}
		return nil
	}); err != nil {
		return err
	}
	if modeLine != "" {
		fmt.Fprintln(cmd.OutOrStdout(), modeLine)
	}
	return nil
}

// trackProbeMode resolves the item's probe mode: the kind default, or the
// --probe override when the kind allows it.
func trackProbeMode(cmd *cobra.Command, kind string, kd trackKind) (string, error) {
	if !cmd.Flags().Changed("probe") {
		return kd.defaultProbe, nil
	}
	p, _ := cmd.Flags().GetString("probe")
	switch p {
	case probePane, probeShell, probeAgent, probeNone:
	default:
		return "", fmt.Errorf("unknown --probe %q (valid: pane, shell, agent, none)", p)
	}
	if p != kd.defaultProbe {
		return "", fmt.Errorf("kind %s does not allow probe mode %q", kind, p)
	}
	return p, nil
}

// applyPaneSugar writes the pane flag sugar into scope. --change pre-declares
// the expected change (validated non-empty only — the change may not exist
// yet); only the stage-valued flags are name-checked.
func applyPaneSugar(cmd *cobra.Command, scope map[string]interface{}) error {
	f := cmd.Flags()
	for _, name := range trackPaneSugarFlags {
		if !f.Changed(name) {
			continue
		}
		v, _ := f.GetString(name)
		key := strings.ReplaceAll(name, "-", "_")
		if v == "" {
			scope[key] = nil
			continue
		}
		if (key == "stage" || key == "stop_stage") && !validStage(v) {
			return fmt.Errorf("invalid --%s %q (valid: intake, apply, review, hydrate, ship, review-pr)", name, v)
		}
		scope[key] = v
	}
	return nil
}

// trackPanePIDLookup is the pane_pid fingerprint lookup at track add/update —
// pane.GetPanePID against the default server (the operator runs inside its own
// tmux). Package-level var so tests stub the seam (the rkPanesRunner /
// ghNameWithOwnerRunner precedent).
var trackPanePIDLookup = func(paneID string) (int, error) {
	return pane.GetPanePID(paneID, "")
}

// recordPanePID maintains the scope.pane_pid fingerprint alongside scope.pane:
// a non-empty pane records the pane's current shell pid; ANY lookup failure
// (tmux unqueryable, pane absent, unparseable pid) stores null and proceeds —
// a null fingerprint means the tick joins on the pane id alone. A cleared
// pane (absent/empty) nulls pane_pid.
func recordPanePID(scope map[string]interface{}) {
	paneID := scopeString(scope, "pane")
	if paneID == "" {
		scope["pane_pid"] = nil
		return
	}
	pid, err := trackPanePIDLookup(paneID)
	if err != nil {
		scope["pane_pid"] = nil
		return
	}
	scope["pane_pid"] = pid
}

// scopeFlag parses --scope as a JSON object. Absent/empty → nil; invalid JSON
// or a non-object → error.
func scopeFlag(cmd *cobra.Command) (map[string]interface{}, error) {
	if !cmd.Flags().Changed("scope") {
		return nil, nil
	}
	raw, _ := cmd.Flags().GetString("scope")
	if raw == "" {
		return nil, nil
	}
	var m map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		return nil, fmt.Errorf("invalid --scope JSON: %v", err)
	}
	if m == nil {
		return nil, fmt.Errorf("invalid --scope: must be a JSON object")
	}
	return m, nil
}

// doneWhenFlag resolves --done-when: unchanged → the kind default ("" →
// null); changed-empty → null (cleared); otherwise parse-validated.
func doneWhenFlag(cmd *cobra.Command, kindDefault string) (*string, error) {
	v := kindDefault
	if cmd.Flags().Changed("done-when") {
		v, _ = cmd.Flags().GetString("done-when")
	}
	if v == "" {
		return nil, nil
	}
	if _, err := predicate.Parse(v); err != nil {
		return nil, fmt.Errorf("invalid --done-when: %v", err)
	}
	return &v, nil
}

// ghNameWithOwnerRunner resolves a local repo path's owner/repo via
// `gh repo view` at track add time (plan Assumption 5). Injectable seam (the
// rkCronRunner precedent); LookPath-gated, argv-only, bounded by
// rkCronTimeout. A failure is an add-time error (defaultGitHubPRArgv) — the
// argv is never silently left without --repo.
var ghNameWithOwnerRunner = func(repoDir string) (string, error) {
	if _, err := exec.LookPath("gh"); err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(context.Background(), rkCronTimeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner")
	cmd.Dir = repoDir
	out, err := cmd.Output()
	if err != nil {
		return "", err
	}
	var v struct {
		NameWithOwner string `json:"nameWithOwner"`
	}
	if err := json.Unmarshal(out, &v); err != nil {
		return "", err
	}
	if v.NameWithOwner == "" {
		return "", fmt.Errorf("gh repo view returned an empty nameWithOwner")
	}
	return v.NameWithOwner, nil
}

// defaultGitHubPRArgv builds the github-pr kind's default probe argv:
// `gh pr view <n> [--repo <owner/repo>] --json state,mergedAt,mergeable`.
// The PR number comes from scope.pr; --repo is derived from scope.repo via
// ghNameWithOwnerRunner when scope carries no owner/repo override.
func defaultGitHubPRArgv(scope map[string]interface{}) ([]string, error) {
	pr := scopePRNumber(scope)
	if pr == "" {
		return nil, fmt.Errorf("kind github-pr requires scope.pr (or explicit --argv)")
	}
	argv := []string{"gh", "pr", "view", pr}
	fields := strings.Join(githubPRProbeFields, ",")
	if repoDir := scopeString(scope, "repo"); repoDir != "" {
		// A failed derivation is an add-time error, never a silent drop:
		// without --repo the probe would run from the operator's cwd (a
		// neutral directory, or a different repo in the cross-repo model)
		// and query the wrong PR number.
		nwo, err := ghNameWithOwnerRunner(repoDir)
		if err != nil {
			return nil, fmt.Errorf("cannot resolve the GitHub repo for scope.repo %s (%v) — pass --argv explicitly (gh pr view %s --repo <owner/repo> --json %s) or fix gh auth", repoDir, err, pr, fields)
		}
		argv = append(argv, "--repo", nwo)
	}
	return append(argv, "--json", fields), nil
}

// scopePRNumber renders scope.pr as the PR-number argv token (JSON numbers
// decode as float64; hand-written YAML may carry an int or string).
func scopePRNumber(scope map[string]interface{}) string {
	switch v := scope["pr"].(type) {
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	case int64:
		return strconv.FormatInt(v, 10)
	case string:
		return v
	}
	return ""
}

// resolveMergeMode resolves a chained pane item's merge mode by the
// ladder: an explicitly passed --mode flag > config (autopilot.merge_mode —
// the key keeps its historical name) > the built-in default. A config-sourced
// value outside the valid set errors naming the config key — merging is
// destructive-tier, so there is no silent fallback to a different topology
// than the one configured.
func resolveMergeMode(cmd *cobra.Command) (mode, source string, err error) {
	if cmd.Flags().Changed("mode") {
		mode, _ = cmd.Flags().GetString("mode")
		if !validMergeMode(mode) {
			return "", "", fmt.Errorf("unknown --mode %q (valid: %s)", mode, strings.Join(config.ValidAutopilotMergeModes, ", "))
		}
		return mode, "flag", nil
	}
	cfg := mergeModeConfig()
	mode = cfg.GetAutopilotMergeMode()
	source = "default"
	if cfg != nil && cfg.Autopilot.MergeMode != "" {
		source = "config"
	}
	if !validMergeMode(mode) {
		return "", "", fmt.Errorf("invalid autopilot.merge_mode %q in config (valid: %s)", mode, strings.Join(config.ValidAutopilotMergeModes, ", "))
	}
	return mode, source, nil
}

func validMergeMode(mode string) bool {
	for _, m := range config.ValidAutopilotMergeModes {
		if m == mode {
			return true
		}
	}
	return false
}

// mergeModeConfig loads the effective config for merge-mode resolution from
// the operator's natural (possibly fab-less) cwd: the project tier when a
// fab/ tree is found, else the cwd-relative path (tolerant of absence) so the
// system and env tiers still compose. A load error is fail-soft (nil → the
// built-in default).
func mergeModeConfig() *config.Config {
	if fabRoot, err := resolve.FabRoot(); err == nil {
		cfg, _ := config.Load(fabRoot) // nil on error → nil-safe accessor resolves the default
		return cfg
	}
	cfg, _ := config.LoadPath(filepath.Join("fab", "project", "config.yaml"))
	return cfg
}

func operatorTrackUpdateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Mutate a tracked item's cadence, predicate, deps, scope, or paused flag",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorTrackUpdate,
	}
	f := cmd.Flags()
	f.String("check-every", "", "probe cadence (floor 1m; null for pane/none items)")
	f.String("done-when", "", "done predicate (empty string clears to null)")
	f.String("then", "", "prose to run when done/changed fires (empty string clears)")
	f.StringSlice("depends-on", nil, "comma-separated item ids (replaces the list)")
	f.String("scope", "", "scope as a JSON object (merged per key)")
	f.String("text", "", "note text (kind note only)")
	f.Bool("pause", false, "pause the item (probes stop until --resume)")
	f.Bool("resume", false, "resume the item (clears paused and zeroes failures)")
	cmd.MarkFlagsMutuallyExclusive("pause", "resume")
	return cmd
}

func runOperatorTrackUpdate(cmd *cobra.Command, args []string) error {
	id := args[0]
	f := cmd.Flags()
	scopeMerge, err := scopeFlag(cmd)
	if err != nil {
		return err
	}
	checkEveryGiven, _ := f.GetString("check-every")
	doneWhen, err := doneWhenFlag(cmd, "")
	if err != nil {
		return err
	}
	pause, _ := f.GetBool("pause")
	resume, _ := f.GetBool("resume")

	return mutateOperatorState(func(data map[string]interface{}) error {
		items, err := decodeTrackedItems(data)
		if err != nil {
			return err
		}
		idx := -1
		for i := range items {
			if items[i].ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("no tracked item %s", id)
		}
		it := items[idx]
		if f.Changed("check-every") {
			ce, err := normalizeCheckEvery(it.Probe.Mode, checkEveryGiven)
			if err != nil {
				return err
			}
			it.CheckEvery = ce
		}
		if f.Changed("done-when") {
			it.DoneWhen = doneWhen
		}
		if f.Changed("then") {
			t, _ := f.GetString("then")
			it.Then = nil
			if t != "" {
				it.Then = &t
			}
		}
		if f.Changed("depends-on") {
			deps, _ := f.GetStringSlice("depends-on")
			if deps == nil {
				deps = []string{}
			}
			it.DependsOn = deps
			if err := validateTrackedDeps(it, items); err != nil {
				return err
			}
		}
		if scopeMerge != nil {
			if it.Scope == nil {
				it.Scope = map[string]interface{}{}
			}
			for k, v := range scopeMerge {
				it.Scope[k] = v
			}
			// A --scope carrying pane re-records the pane_pid fingerprint in
			// the same mutation (a cleared pane nulls it).
			if _, touched := scopeMerge["pane"]; touched {
				recordPanePID(it.Scope)
			}
		}
		if f.Changed("text") {
			if it.Kind != kindNote {
				return fmt.Errorf("--text applies only to kind note")
			}
			text, _ := f.GetString("text")
			if len([]rune(text)) > trackNoteTextCap {
				return fmt.Errorf("note text over the %d-character cap", trackNoteTextCap)
			}
			it.Text = text
		}
		if pause {
			it.Paused = true
		}
		if resume {
			it.Paused = false
			it.Failures = 0
		}
		it.UpdatedAt = nowRFC3339()
		items[idx] = it
		data["tracked"] = items
		return nil
	})
}

func operatorTrackRmCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "rm <id>",
		Short: "Remove a tracked item (the ack for done; a pane item's branch_map entry is retained)",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorTrackRm,
	}
}

func runOperatorTrackRm(cmd *cobra.Command, args []string) error {
	id := args[0]
	return mutateOperatorState(func(data map[string]interface{}) error {
		items, err := decodeTrackedItems(data)
		if err != nil {
			return err
		}
		idx := -1
		for i := range items {
			if items[i].ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("no tracked item %s", id)
		}
		removed := items[idx]
		items = append(items[:idx], items[idx+1:]...)
		if items == nil {
			items = []trackedItem{}
		}
		// A DONE dependency's edges are satisfied — drop it from every
		// dependent's depends_on so the chain advances after the ack (the
		// documented tick flow rms a done item, then spawns its dependents;
		// dependency satisfaction searches only `tracked`). A not-done
		// removal keeps the edges: dependents stay held and the tick names
		// the missing id, which the operator resolves with `track update
		// --depends-on`.
		if trackedItemDone(removed) {
			for i := range items {
				items[i].DependsOn = dropString(items[i].DependsOn, removed.ID)
			}
		}
		data["tracked"] = items
		// branch_map deliberately untouched — entries persist for downstream
		// dependency resolution until explicitly cleared (branch-map rm).
		return nil
	})
}

// dropString returns list without every occurrence of s (never nil — an
// emptied depends_on stays `[]`).
func dropString(list []string, s string) []string {
	out := make([]string, 0, len(list))
	for _, v := range list {
		if v != s {
			out = append(out, v)
		}
	}
	return out
}

func operatorTrackObserveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "observe <id>",
		Short: "Record an agent-probe result (--json) or failure (--error) for a tracked item",
		Args:  cobra.ExactArgs(1),
		RunE:  runOperatorTrackObserve,
	}
	cmd.Flags().String("json", "", "observed fields as a JSON object (stored per probe.fields, or whole when fields is empty)")
	cmd.Flags().String("error", "", "probe failure message (increments failures; the third pauses the item)")
	cmd.Flags().StringArray("seen", nil, "handled item id to append to seen (repeatable; 200-cap, oldest pruned)")
	return cmd
}

func runOperatorTrackObserve(cmd *cobra.Command, args []string) error {
	id := args[0]
	f := cmd.Flags()
	jsonGiven, errGiven := f.Changed("json"), f.Changed("error")
	if jsonGiven == errGiven {
		return fmt.Errorf("track observe requires exactly one of --json or --error")
	}
	var observed map[string]interface{}
	if jsonGiven {
		raw, _ := f.GetString("json")
		var decoded interface{}
		if err := json.Unmarshal([]byte(raw), &decoded); err != nil {
			return fmt.Errorf("invalid --json: %v", err)
		}
		var ok bool
		observed, ok = decoded.(map[string]interface{})
		if !ok {
			return fmt.Errorf("invalid --json: must be a JSON object")
		}
	}
	seen, _ := f.GetStringArray("seen")

	return mutateOperatorState(func(data map[string]interface{}) error {
		items, err := decodeTrackedItems(data)
		if err != nil {
			return err
		}
		idx := -1
		for i := range items {
			if items[i].ID == id {
				idx = i
				break
			}
		}
		if idx < 0 {
			return fmt.Errorf("no tracked item %s", id)
		}
		it := items[idx]
		now := nowRFC3339()
		it.CheckedAt = &now
		it.UpdatedAt = now
		if errGiven {
			it.Failures++
			if it.Failures >= trackFailurePauseCap {
				it.Paused = true
			}
		} else {
			newLast := observed
			if len(it.Probe.Fields) > 0 {
				newLast = extractProbeFields(observed, it.Probe.Fields)
			}
			normalizeJSONNumbers(newLast)
			if reflect.DeepEqual(newLast, it.Last) {
				it.Unchanged++
			} else {
				it.Unchanged = 0
				it.Last = newLast
			}
			it.Failures = 0
			for _, s := range seen {
				if !stringSliceContains(it.Seen, s) {
					it.Seen = append(it.Seen, s)
				}
			}
			if len(it.Seen) > trackSeenCap {
				it.Seen = it.Seen[len(it.Seen)-trackSeenCap:]
			}
		}
		// done_when is level-triggered and derived at read time
		// (trackedItemDone) — observe stores no verdict.
		items[idx] = it
		data["tracked"] = items
		return nil
	})
}

// extractProbeFields pulls the declared fields (top-level keys, or dotted
// paths a.b into nested objects) out of an observed object. An absent field
// stores null so a value→absent transition stays visible and `field == null`
// predicates work.
func extractProbeFields(obj map[string]interface{}, fields []string) map[string]interface{} {
	out := map[string]interface{}{}
	for _, f := range fields {
		segs := strings.Split(f, ".")
		setPathValue(out, segs, getPathValue(obj, segs))
	}
	return out
}

// setPathValue writes v at a dotted path, MERGING into nested objects that
// earlier fields already created — fields `a.b` and `a.c` both land under
// `a` instead of the second overwriting the first. A non-object value at an
// intermediate segment is replaced by an object so the declared field
// always lands.
func setPathValue(out map[string]interface{}, segs []string, v interface{}) {
	cur := out
	for _, s := range segs[:len(segs)-1] {
		next, ok := cur[s].(map[string]interface{})
		if !ok {
			next = map[string]interface{}{}
			cur[s] = next
		}
		cur = next
	}
	cur[segs[len(segs)-1]] = v
}

// normalizeJSONNumbers rewrites integral float64 values (JSON's number
// decoding) to ints so a stored `last` compares DeepEqual-stable across a
// YAML save/load roundtrip (YAML decodes "1" as int).
func normalizeJSONNumbers(v interface{}) {
	switch m := v.(type) {
	case map[string]interface{}:
		for k, val := range m {
			if f, ok := val.(float64); ok && f == float64(int(f)) {
				m[k] = int(f)
			} else {
				normalizeJSONNumbers(val)
			}
		}
	case []interface{}:
		for i, val := range m {
			if f, ok := val.(float64); ok && f == float64(int(f)) {
				m[i] = int(f)
			} else {
				normalizeJSONNumbers(val)
			}
		}
	}
}

func stringSliceContains(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func operatorTrackListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tracked items (one line per item: id · kind · state · checked age · next)",
		Args:  cobra.NoArgs,
		RunE:  runOperatorTrackList,
	}
	cmd.Flags().String("kind", "", "filter to one kind")
	cmd.Flags().Bool("json", false, "print the items array as JSON")
	return cmd
}

func runOperatorTrackList(cmd *cobra.Command, args []string) error {
	path, err := operatorStatePath()
	if err != nil {
		return err
	}
	data, err := loadOperatorStateUpgraded(path)
	if err != nil {
		return err
	}
	items, err := decodeTrackedItems(data)
	if err != nil {
		return err
	}
	shown := items
	if kindFilter, _ := cmd.Flags().GetString("kind"); kindFilter != "" {
		if _, ok := trackKinds[kindFilter]; !ok {
			return fmt.Errorf("unknown --kind %q (valid: %s)", kindFilter, trackKindNames)
		}
		shown = []trackedItem{}
		for _, it := range items {
			if it.Kind == kindFilter {
				shown = append(shown, it)
			}
		}
	}
	if asJSON, _ := cmd.Flags().GetBool("json"); asJSON {
		out, err := json.MarshalIndent(shown, "", "  ")
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(out))
		return nil
	}
	w := cmd.OutOrStdout()
	now := time.Now().UTC()
	for _, it := range shown {
		fmt.Fprintf(w, "%s · %s · %s · %s · %s\n",
			it.ID, it.Kind, trackListState(items, it, now), trackListCheckedAge(items, it, now), trackListNext(items, it))
	}
	return nil
}

// trackListState derives the item's lifecycle state (R10) without a pane
// snapshot — pane items with a pane read live (the tick's join is the
// authoritative liveness check).
func trackListState(items []trackedItem, it trackedItem, now time.Time) string {
	switch {
	case trackedItemDone(it):
		return "done"
	case it.Paused:
		return "paused"
	}
	if trackHeldDep(items, it) != "" {
		return "held"
	}
	if it.Kind == kindPane && scopeString(it.Scope, "pane") == "" {
		return "pending"
	}
	if trackItemStale(it, now) {
		return "stale"
	}
	if it.Probe.Mode == probePane {
		return "live"
	}
	return "watching"
}

// trackListCheckedAge renders the Checked column: — for held items, live for
// pane items, the age since checked_at for probed items, and the age since
// updated_at for notes.
func trackListCheckedAge(items []trackedItem, it trackedItem, now time.Time) string {
	if !trackedItemDone(it) && !it.Paused && trackHeldDep(items, it) != "" {
		return "—"
	}
	if it.Probe.Mode == probePane {
		return "live"
	}
	base := it.CheckedAt
	if it.Kind == kindNote {
		base = &it.UpdatedAt
	}
	if base == nil || *base == "" {
		return "—"
	}
	t, err := time.Parse(time.RFC3339, *base)
	if err != nil {
		return "—"
	}
	return formatTrackAge(now.Sub(t))
}

// trackListNext renders the Next column: the item's then, "spawn" for a
// pending pane item, "held: <dep-id>" for a held item.
func trackListNext(items []trackedItem, it trackedItem) string {
	if it.Then != nil && *it.Then != "" {
		return *it.Then
	}
	if dep := trackHeldDep(items, it); dep != "" && !trackedItemDone(it) {
		return "held: " + dep
	}
	if it.Kind == kindPane && scopeString(it.Scope, "pane") == "" && !trackedItemDone(it) && !it.Paused {
		return "spawn"
	}
	return "—"
}

// trackNoteStaleThreshold flags a note item stale (display-only) when its
// updated_at age exceeds it.
const trackNoteStaleThreshold = 14 * 24 * time.Hour

// formatTrackAge renders an item age with day capability (note ages span
// weeks) — the note-list age formatter carried over to items.
func formatTrackAge(d time.Duration) string {
	secs := int64(d.Seconds())
	if secs < 0 {
		secs = 0
	}
	switch {
	case secs < 60:
		return fmt.Sprintf("%ds", secs)
	case secs < 3600:
		return fmt.Sprintf("%dm", secs/60)
	case secs < 86400:
		return fmt.Sprintf("%dh", secs/3600)
	default:
		return fmt.Sprintf("%dd", secs/86400)
	}
}

// formatTrackNoteLine renders the shared one-line note shape used by the
// `state` OPEN NOTES header: id · note · age · first line of text. A stale
// note (updated_at older than trackNoteStaleThreshold) carries the
// display-only `⚠ <age>` flag.
func formatTrackNoteLine(it trackedItem, now time.Time) string {
	age := formatTrackAge(trackNoteAge(it, now))
	if trackNoteAge(it, now) > trackNoteStaleThreshold {
		age = "⚠ " + age
	}
	first, _, _ := strings.Cut(it.Text, "\n")
	return fmt.Sprintf("%s · %s · %s · %s", it.ID, kindNote, age, first)
}

// trackNoteAge is the note item's age from updated_at; an unparseable
// timestamp is 0.
func trackNoteAge(it trackedItem, now time.Time) time.Duration {
	t, err := time.Parse(time.RFC3339, it.UpdatedAt)
	if err != nil {
		return 0
	}
	return now.Sub(t)
}

func operatorTrackClockCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "clock",
		Short: "Override the derived operator-tick cadence for a bounded time (or clear with --off)",
		Args:  cobra.NoArgs,
		RunE:  runOperatorTrackClock,
	}
	cmd.Flags().String("every", "", "fixed cadence (schedule kind every)")
	cmd.Flags().String("idle-every", "", "idle-keyed cadence (schedule kind idle-every)")
	cmd.Flags().String("for", "", "override lifetime (required — unbounded overrides are forbidden)")
	cmd.Flags().Bool("off", false, "clear the override early")
	return cmd
}

func runOperatorTrackClock(cmd *cobra.Command, args []string) error {
	f := cmd.Flags()
	off, _ := f.GetBool("off")
	every, _ := f.GetString("every")
	idleEvery, _ := f.GetString("idle-every")
	if off {
		if f.Changed("every") || f.Changed("idle-every") || f.Changed("for") {
			return fmt.Errorf("track clock --off takes no other flags")
		}
		return mutateOperatorState(func(data map[string]interface{}) error {
			delete(data, "clock_override")
			return nil
		})
	}
	if every != "" && idleEvery != "" {
		return fmt.Errorf("track clock takes exactly one of --every or --idle-every")
	}
	kind, val := "every", every
	if idleEvery != "" {
		kind, val = "idle-every", idleEvery
	}
	if val == "" {
		return fmt.Errorf("track clock requires --every or --idle-every (or --off)")
	}
	d, err := time.ParseDuration(val)
	if err != nil {
		return fmt.Errorf("invalid --%s %q: %v", kind, val, err)
	}
	if d < trackCheckEveryFloor {
		return fmt.Errorf("--%s %q is below the %s floor", kind, val, trackCheckEveryFloor)
	}
	if !f.Changed("for") {
		return fmt.Errorf("track clock requires --for (unbounded overrides are forbidden; use --off to clear)")
	}
	forDur, _ := f.GetString("for")
	life, err := time.ParseDuration(forDur)
	if err != nil {
		return fmt.Errorf("invalid --for %q: %v", forDur, err)
	}
	if life <= 0 {
		return fmt.Errorf("--for must be positive")
	}
	override := clockOverride{
		Schedule: clockOverrideSchedule{Kind: kind, Every: val},
		Deliver:  operatorDeliverSkipIfBusy,
		Until:    time.Now().UTC().Add(life).Format(time.RFC3339),
	}
	return mutateOperatorState(func(data map[string]interface{}) error {
		data["clock_override"] = override
		return nil
	})
}
