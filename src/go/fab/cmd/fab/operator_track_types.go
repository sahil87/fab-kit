package main

import (
	"fmt"
	"math"
	"time"

	"github.com/sahil87/fab-kit/src/go/fab/internal/predicate"
)

// Tracked-item model (intake B1): the operator state file's one owned list.
// The binary owns the schema, the timestamps, the caps, and the atomic write;
// the tolerant-read/typed-write posture is unchanged — unknown TOP-LEVEL keys
// survive a mutation, an invented field inside `tracked` does not (every
// mutation re-marshals the section from the typed structs below).

const (
	// trackProbeTimeout bounds each shell probe subprocess (the T007 tick
	// probe runner).
	trackProbeTimeout = 10 * time.Second
	// trackTickBudget bounds total shell-probe wall time per tick (T007).
	trackTickBudget = 60 * time.Second
	// trackFailurePauseCap is the consecutive-probe-failure count that trips
	// an item to paused: true.
	trackFailurePauseCap = 3
	// trackSeenCap is the binary-enforced cap on a linear/slack item's `seen`
	// list (oldest pruned first) — succeeds the watch known-list cap.
	trackSeenCap = 200
	// trackCheckEveryFloor is the minimum check_every cadence (R3).
	trackCheckEveryFloor = time.Minute
	// trackCheckEveryDefault is the check_every filled at add time for
	// shell/agent items that omit it (R3).
	trackCheckEveryDefault = 5 * time.Minute
	// trackCheckEveryDefaultText is the default's stored form ("5m", not
	// Duration.String's "5m0s") — the state file carries terse durations.
	trackCheckEveryDefaultText = "5m"
	// trackStaleMultiplier defines staleness: an agent item is stale when
	// now - checked_at exceeds stale-multiplier × check_every (T007/T008).
	trackStaleMultiplier = 2
	// trackNoteTextCap is the binary-enforced cap on a note item's text (in
	// runes) — succeeds the notes-section cap.
	trackNoteTextCap = 500
)

// Item kinds and probe modes (R1).
const (
	kindPane     = "pane"
	kindGitHubPR = "github-pr"
	kindLinear   = "linear"
	kindSlack    = "slack"
	kindShell    = "shell"
	kindTask     = "task"
	kindNote     = "note"
)

const (
	probePane  = "pane"
	probeShell = "shell"
	probeAgent = "agent"
	probeNone  = "none"
)

// trackKind carries one kind's defaults from the intake B1 kinds table. Each
// kind allows exactly its default probe mode.
type trackKind struct {
	defaultProbe string
	doneWhen     string // "" = null default
}

var trackKinds = map[string]trackKind{
	kindPane:     {defaultProbe: probePane},
	kindGitHubPR: {defaultProbe: probeShell, doneWhen: `state == "MERGED"`},
	kindLinear:   {defaultProbe: probeAgent},
	kindSlack:    {defaultProbe: probeAgent},
	kindShell:    {defaultProbe: probeShell},
	kindTask:     {defaultProbe: probeNone},
	kindNote:     {defaultProbe: probeNone},
}

// trackKindNames is the valid-kind list for error messages, in table order.
const trackKindNames = "pane, github-pr, linear, slack, shell, task, note"

// githubPRProbeFields is the github-pr kind's default declared-field set.
var githubPRProbeFields = []string{"state", "mergedAt", "mergeable"}

// probeSpec declares how an item is probed. argv/fields belong to shell
// probes (argv is an argv list, never a shell string); instruction belongs to
// agent probes (the LLM runs it, the binary never does).
type probeSpec struct {
	Mode        string   `yaml:"mode" json:"mode"`
	Argv        []string `yaml:"argv,omitempty" json:"argv,omitempty"`
	Fields      []string `yaml:"fields,omitempty" json:"fields,omitempty"`
	Instruction string   `yaml:"instruction,omitempty" json:"instruction,omitempty"`
}

// trackedItem is one `tracked` list entry — the intake B1 schema. Field order
// pins the YAML key order. seen is linear/slack-only, text note-only (both
// omitempty).
type trackedItem struct {
	ID         string                 `yaml:"id" json:"id"`
	Kind       string                 `yaml:"kind" json:"kind"`
	Probe      probeSpec              `yaml:"probe" json:"probe"`
	CheckEvery *string                `yaml:"check_every" json:"check_every"`
	DoneWhen   *string                `yaml:"done_when" json:"done_when"`
	Then       *string                `yaml:"then" json:"then"`
	DependsOn  []string               `yaml:"depends_on" json:"depends_on"`
	Scope      map[string]interface{} `yaml:"scope" json:"scope"`
	Last       map[string]interface{} `yaml:"last" json:"last"`
	Seen       []string               `yaml:"seen,omitempty" json:"seen,omitempty"`
	Text       string                 `yaml:"text,omitempty" json:"text,omitempty"`
	CheckedAt  *string                `yaml:"checked_at" json:"checked_at"`
	Unchanged  int                    `yaml:"unchanged" json:"unchanged"`
	Failures   int                    `yaml:"failures" json:"failures"`
	Paused     bool                   `yaml:"paused" json:"paused"`
	DoneAt     *string                `yaml:"done_at" json:"done_at"` // set by tick-start when a pane item's built-in completion fires; durable across pane death
	AddedAt    string                 `yaml:"added_at" json:"added_at"`
	UpdatedAt  string                 `yaml:"updated_at" json:"updated_at"`
}

// decodeTrackedItems decodes the tracked section into its typed list (the
// typed-write half of the posture: in-section drift is dropped). A
// missing/null section yields an empty list.
func decodeTrackedItems(data map[string]interface{}) ([]trackedItem, error) {
	items := []trackedItem{}
	if err := operatorSection(data, "tracked", &items); err != nil {
		return nil, err
	}
	return items, nil
}

// normalizeCheckEvery applies the R3 rules: forced null for pane/none items
// (silently), the 5m default for shell/agent items when omitted, and the 1m
// floor (below → error).
func normalizeCheckEvery(mode, given string) (*string, error) {
	if mode == probePane || mode == probeNone {
		return nil, nil
	}
	if given == "" {
		def := trackCheckEveryDefaultText
		return &def, nil
	}
	dur, err := time.ParseDuration(given)
	if err != nil {
		return nil, fmt.Errorf("invalid --check-every %q: %v", given, err)
	}
	if dur < trackCheckEveryFloor {
		return nil, fmt.Errorf("--check-every %q is below the %s floor", given, trackCheckEveryFloor)
	}
	return &given, nil
}

// validateTrackedItem runs the add/update validation rules that need no other
// items: known kind, kind-allowed probe mode, shell probe argv+fields, a
// parseable done_when, and the note text cap. The duplicate-id and
// depends-on checks need the existing list and live in validateTrackedDeps.
func validateTrackedItem(it trackedItem) error {
	kd, ok := trackKinds[it.Kind]
	if !ok {
		return fmt.Errorf("unknown kind %q (valid: %s)", it.Kind, trackKindNames)
	}
	if it.Probe.Mode != kd.defaultProbe {
		return fmt.Errorf("kind %s does not allow probe mode %q", it.Kind, it.Probe.Mode)
	}
	if it.Probe.Mode == probeShell {
		if len(it.Probe.Argv) == 0 {
			return fmt.Errorf("shell probe requires --argv")
		}
		if len(it.Probe.Fields) == 0 {
			return fmt.Errorf("shell probe requires --fields")
		}
	}
	if it.DoneWhen != nil && *it.DoneWhen != "" {
		if _, err := predicate.Parse(*it.DoneWhen); err != nil {
			return fmt.Errorf("invalid --done-when: %v", err)
		}
	}
	if len([]rune(it.Text)) > trackNoteTextCap {
		return fmt.Errorf("note text over the %d-character cap", trackNoteTextCap)
	}
	return nil
}

// validateTrackedDeps checks the item's depends_on against the existing list:
// every named id must exist and no item may depend on itself.
func validateTrackedDeps(it trackedItem, existing []trackedItem) error {
	for _, dep := range it.DependsOn {
		if dep == it.ID {
			return fmt.Errorf("--depends-on %q names the item itself", dep)
		}
		known := false
		for _, e := range existing {
			if e.ID == dep {
				known = true
				break
			}
		}
		if !known {
			return fmt.Errorf("--depends-on %q: no tracked item with that id", dep)
		}
	}
	return nil
}

// trackedItemDone reports whether the item's done_when fires against its last
// observed fields. A null done_when (the pane built-in, standing linear/slack
// items) is never done HERE — the pane built-in predicate (review-pr
// done/skipped, or at/past stop_stage, gated on a non-null scope.change) is
// the tick's to evaluate (T007).
func trackedItemDone(it trackedItem) bool {
	// A persisted built-in completion (pane items) is durable evidence: the
	// item stays done whatever its pane does afterwards.
	if it.DoneAt != nil && *it.DoneAt != "" {
		return true
	}
	if it.DoneWhen == nil || *it.DoneWhen == "" {
		return false
	}
	p, err := predicate.Parse(*it.DoneWhen)
	if err != nil {
		return false
	}
	return p.Eval(it.Last)
}

// scopeString reads a string scope value ("" for absent/null/non-string).
func scopeString(scope map[string]interface{}, key string) string {
	s, _ := scope[key].(string)
	return s
}

// scopeInt reads an integer scope value (ok=false for absent/null/non-numeric
// — the fingerprint rule treats those as "no recorded fingerprint"). JSON
// numbers decode as float64; hand-written YAML may carry an int. A non-integral
// float64 (48213.5) is NOT truncated — it reports ok=false rather than
// silently joining a pane whose PID merely shares the floor.
func scopeInt(scope map[string]interface{}, key string) (int, bool) {
	switch v := scope[key].(type) {
	case int:
		return v, true
	case int64:
		return int(v), true
	case float64:
		if v != math.Trunc(v) {
			return 0, false
		}
		return int(v), true
	}
	return 0, false
}

// checkEveryDuration parses the item's check_every; a null/unparseable value
// reports ok=false.
func checkEveryDuration(it trackedItem) (time.Duration, bool) {
	if it.CheckEvery == nil {
		return 0, false
	}
	d, err := time.ParseDuration(*it.CheckEvery)
	return d, err == nil
}

// trackItemStale applies the R8 staleness rule: an agent item with
// now - checked_at > 2 × check_every (or checked_at null and added_at older
// than 2×).
func trackItemStale(it trackedItem, now time.Time) bool {
	if it.Probe.Mode != probeAgent {
		return false
	}
	ce, ok := checkEveryDuration(it)
	if !ok {
		return false
	}
	base := it.AddedAt
	if it.CheckedAt != nil {
		base = *it.CheckedAt
	}
	t, err := time.Parse(time.RFC3339, base)
	if err != nil {
		return false
	}
	return now.Sub(t) > trackStaleMultiplier*ce
}

// trackHeldDep returns the first depends_on id whose item is not done (""
// when every dep is satisfied). A dep id missing from the list counts as
// unsatisfied (the tick names it in a probe_error — R10).
func trackHeldDep(items []trackedItem, it trackedItem) string {
	for _, dep := range it.DependsOn {
		satisfied := false
		for _, e := range items {
			if e.ID == dep && trackedItemDone(e) {
				satisfied = true
				break
			}
		}
		if !satisfied {
			return dep
		}
	}
	return ""
}
