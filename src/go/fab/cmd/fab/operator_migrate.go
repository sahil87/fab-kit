package main

import (
	"errors"
	"fmt"
	"sort"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// Legacy state-file conversion (R5, intake B1 Migration). Two idempotent
// passes, both landing in the SAME atomic write as the verb's own mutation:
//
//  1. A legacy-shaped file — any of monitored/watches/autopilot/notes present
//     with `tracked` absent — converts on the first read-modify-write by any
//     `fab operator` verb (the legacy keys are deleted), emitting kind: pane
//     items directly.
//  2. A `tracked`-present file holding a retired kind: fab-change item
//     converts that item (kind: pane, scope.change = id, scope.pane_pid:
//     null) on ANY verb's read-modify-write (convertFabChangeItems).
//
// A stray legacy key in an already-converted file survives as an unknown
// top-level key (A-034). The read verbs `state` and `track list`
// convert-and-save, then read (loadOperatorStateUpgraded).

// operatorLegacyRunningRefusal is the exact one-line refusal emitted when a
// legacy file carries a running autopilot queue: the running queue's current
// entry has in-flight side effects the conversion cannot reproduce.
const operatorLegacyRunningRefusal = "operator state file has a running autopilot queue — finish or stop it (fab operator autopilot stop on fab ≤2.24) before upgrading"

// --- legacy section types (conversion-path only — nothing else may carry the
// old schemas; A-025) --------------------------------------------------------

// monitoredEntry is one legacy `monitored` entry.
type monitoredEntry struct {
	Pane           string   `yaml:"pane"`
	Repo           string   `yaml:"repo"`
	Session        string   `yaml:"session"`
	Stage          string   `yaml:"stage,omitempty"`
	Agent          string   `yaml:"agent,omitempty"`
	StopStage      *string  `yaml:"stop_stage"`
	SpawnedBy      *string  `yaml:"spawned_by"`
	DependsOn      []string `yaml:"depends_on"`
	Branch         string   `yaml:"branch"`
	EnrolledAt     string   `yaml:"enrolled_at"`
	LastTransition string   `yaml:"last_transition"`
}

// autopilotState is the legacy top-level `autopilot` block (absent →
// `autopilot: null`).
type autopilotState struct {
	Queue     []string `yaml:"queue"`
	Current   *string  `yaml:"current"`
	Completed []string `yaml:"completed"`
	State     *string  `yaml:"state"`
	Mode      string   `yaml:"mode"`
}

// watchEntry is one legacy `watches` entry.
type watchEntry struct {
	Enabled      bool                   `yaml:"enabled"`
	Source       string                 `yaml:"source"`
	Query        map[string]interface{} `yaml:"query,omitempty"`
	TargetRepo   string                 `yaml:"target_repo"`
	StopStage    *string                `yaml:"stop_stage"`
	Known        []string               `yaml:"known"`
	Completed    []string               `yaml:"completed"`
	LastChecked  *string                `yaml:"last_checked"`
	LastError    *string                `yaml:"last_error"`
	Instructions string                 `yaml:"instructions,omitempty"`
}

// noteEntry is one legacy `notes` entry.
type noteEntry struct {
	ID         string   `yaml:"id" json:"id"`
	Kind       string   `yaml:"kind" json:"kind"`
	Text       string   `yaml:"text" json:"text"`
	Refs       []string `yaml:"refs,omitempty" json:"refs,omitempty"`
	CreatedAt  string   `yaml:"created_at" json:"created_at"`
	UpdatedAt  string   `yaml:"updated_at" json:"updated_at"`
	Resolved   bool     `yaml:"resolved" json:"resolved"`
	ResolvedAt *string  `yaml:"resolved_at" json:"resolved_at"`
}

// readNotes decodes the legacy notes section into its typed creation-order
// list. A missing/null section yields an empty list.
func readNotes(data map[string]interface{}) ([]noteEntry, error) {
	notes := []noteEntry{}
	if err := operatorSection(data, "notes", &notes); err != nil {
		return nil, err
	}
	return notes, nil
}

// legacyOperatorState reports whether data is a legacy-shaped file: any of
// the four old owned sections present (non-null) with `tracked` absent.
func legacyOperatorState(data map[string]interface{}) bool {
	if t, ok := data["tracked"]; ok && t != nil {
		return false
	}
	for _, k := range []string{"monitored", "watches", "autopilot", "notes"} {
		if v, ok := data[k]; ok && v != nil {
			return true
		}
	}
	return false
}

// loadOperatorStateUpgraded is the read-path conversion hook for verbs that
// otherwise never write (state, track list): a legacy-shaped file converts
// and saves (one atomic write) before the read proceeds; a tracked-present
// file holding retired kind: fab-change items converts (kind: pane, seeded
// scope.change/pane_pid) and saves the same way. Anything else is returned
// untouched (byte-stability preserved for read-only verbs).
func loadOperatorStateUpgraded(path string) (map[string]interface{}, error) {
	data, err := loadOperatorState(path)
	if err != nil {
		return nil, err
	}
	if legacyOperatorState(data) {
		if err := convertLegacyOperatorState(data); err != nil {
			return nil, err
		}
		if err := saveOperatorState(path, data); err != nil {
			return nil, err
		}
		return data, nil
	}
	changed, err := convertFabChangeItems(data)
	if err != nil {
		return nil, err
	}
	if !changed {
		return data, nil
	}
	if err := saveOperatorState(path, data); err != nil {
		return nil, err
	}
	return data, nil
}

// kindLegacyFabChange is the retired tracked-item kind name. The string
// "fab-change" may appear in Go ONLY at the legacy-conversion sites in this
// file (and the tests exercising them) — `track add` knows no such kind.
const kindLegacyFabChange = "fab-change"

// convertFabChangeItems is the second, idempotent conversion pass: a
// tracked-present file holding an item with the retired kind: fab-change
// rewrites it to kind: pane, seeding scope.change = the item's id (a
// fab-change item's id was the change id by construction — without this the
// converted item would never complete) and scope.pane_pid = null (no live
// fingerprint ⇒ pane-id-only join, the pre-rename behavior). Fires on ANY
// verb's read-modify-write and lands in the same atomic write. data["tracked"]
// is re-marshaled ONLY when an item converted, so an already-converted file
// keeps its raw tracked section (tolerant-read/typed-write contract).
func convertFabChangeItems(data map[string]interface{}) (bool, error) {
	items, err := decodeTrackedItems(data)
	if err != nil {
		return false, err
	}
	changed := false
	for i := range items {
		if items[i].Kind != kindLegacyFabChange {
			continue
		}
		items[i].Kind = kindPane
		if items[i].Scope == nil {
			items[i].Scope = map[string]interface{}{}
		}
		items[i].Scope["change"] = items[i].ID
		items[i].Scope["pane_pid"] = nil
		changed = true
	}
	if changed {
		data["tracked"] = items
	}
	return changed, nil
}

// hasLegacyFabChangeItems reports whether a tracked-present file still holds
// a retired kind: fab-change item — the read-verb probe gating
// convert-and-save in `state` (write verbs go through mutateOperatorState,
// which converts unconditionally).
func hasLegacyFabChangeItems(data map[string]interface{}) bool {
	items, err := decodeTrackedItems(data)
	if err != nil {
		return false
	}
	for _, it := range items {
		if it.Kind == kindLegacyFabChange {
			return true
		}
	}
	return false
}

// convertLegacyOperatorState converts a legacy-shaped file in place: the
// caller's save lands the conversion in the same atomic write as the verb's
// own mutation. A running autopilot queue refuses with
// operatorLegacyRunningRefusal and nothing is written.
func convertLegacyOperatorState(data map[string]interface{}) error {
	if !legacyOperatorState(data) {
		return nil
	}
	ap := autopilotState{}
	if err := operatorSection(data, "autopilot", &ap); err != nil {
		return err
	}
	if ap.State != nil && *ap.State == "running" {
		return errors.New(operatorLegacyRunningRefusal)
	}

	now := nowRFC3339()
	items := []trackedItem{}

	// monitored.<id> → pane items (scope from the entry fields, scope.change
	// seeded from the id, checked_at = last_transition).
	monitored := map[string]monitoredEntry{}
	if err := operatorSection(data, "monitored", &monitored); err != nil {
		return err
	}
	for _, id := range sortedKeys(monitored) {
		items = append(items, convertMonitoredEntry(id, monitored[id]))
	}

	// watches.<name> → linear/slack agent items (seen = known ∪ completed,
	// then = instructions, check_every 5m, a disabled watch converts paused).
	watches := map[string]watchEntry{}
	if err := operatorSection(data, "watches", &watches); err != nil {
		return err
	}
	for _, name := range sortedKeys(watches) {
		items = append(items, convertWatchEntry(name, watches[name], now))
	}

	// autopilot queue entries not in completed → pane-less pane-kind items
	// chained by depends_on, scope.merge_mode = autopilot.mode.
	bm := map[string]branchMapEntry{}
	if err := operatorSection(data, "branch_map", &bm); err != nil {
		return err
	}
	items = convertAutopilotQueue(items, ap, bm, now)

	// Open notes → kind: note items keeping their n<N> ids; resolved dropped.
	notes, err := readNotes(data)
	if err != nil {
		return err
	}
	for _, n := range notes {
		if n.Resolved {
			continue
		}
		scope := map[string]interface{}{}
		if len(n.Refs) > 0 {
			scope["refs"] = n.Refs
		}
		items = append(items, trackedItem{
			ID:        n.ID,
			Kind:      kindNote,
			Probe:     probeSpec{Mode: probeNone},
			DependsOn: []string{},
			Scope:     scope,
			Last:      map[string]interface{}{},
			Text:      n.Text,
			AddedAt:   n.CreatedAt,
			UpdatedAt: n.UpdatedAt,
		})
	}

	for _, k := range []string{"monitored", "watches", "autopilot", "notes", "notes_seq"} {
		delete(data, k)
	}
	data["tracked"] = items
	return nil
}

// paneScope seeds the eleven pinned pane scope keys as null (the intake B1
// schema); add-time flag sugar and the migration fill them in.
func paneScope() map[string]interface{} {
	return map[string]interface{}{
		"pane": nil, "pane_pid": nil, "change": nil, "repo": nil, "session": nil,
		"branch": nil, "stage": nil, "agent": nil, "stop_stage": nil,
		"spawned_by": nil, "merge_mode": nil,
	}
}

// setScopeString sets scope[key] when s is non-empty (empty → left null).
func setScopeString(scope map[string]interface{}, key, s string) {
	if s != "" {
		scope[key] = s
	}
}

func convertMonitoredEntry(id string, e monitoredEntry) trackedItem {
	scope := paneScope()
	scope["change"] = id // a monitored entry's id IS its change id
	setScopeString(scope, "pane", e.Pane)
	setScopeString(scope, "repo", e.Repo)
	setScopeString(scope, "session", e.Session)
	setScopeString(scope, "branch", e.Branch)
	setScopeString(scope, "stage", e.Stage)
	setScopeString(scope, "agent", e.Agent)
	if e.StopStage != nil {
		scope["stop_stage"] = *e.StopStage
	}
	if e.SpawnedBy != nil {
		scope["spawned_by"] = *e.SpawnedBy
	}
	var checkedAt *string
	if e.LastTransition != "" {
		checkedAt = &e.LastTransition
	}
	dependsOn := e.DependsOn
	if dependsOn == nil {
		dependsOn = []string{}
	}
	return trackedItem{
		ID:        id,
		Kind:      kindPane,
		Probe:     probeSpec{Mode: probePane},
		DependsOn: dependsOn,
		Scope:     scope,
		Last:      map[string]interface{}{},
		CheckedAt: checkedAt,
		AddedAt:   e.EnrolledAt,
		UpdatedAt: e.LastTransition,
	}
}

func convertWatchEntry(name string, w watchEntry, now string) trackedItem {
	instruction := fmt.Sprintf("check %s for new items; report new ids not in seen", w.Source)
	if len(w.Query) > 0 {
		instruction = fmt.Sprintf("check %s for new items matching query %v; report new ids not in seen", w.Source, w.Query)
	}
	scope := map[string]interface{}{
		"repo":       w.TargetRepo,
		"stop_stage": nil,
		"query":      nil,
	}
	if w.StopStage != nil {
		scope["stop_stage"] = *w.StopStage
	}
	if len(w.Query) > 0 {
		scope["query"] = w.Query
	}
	seen := unionStrings(w.Known, w.Completed)
	if len(seen) > trackSeenCap {
		seen = seen[len(seen)-trackSeenCap:]
	}
	checkEvery := trackCheckEveryDefaultText
	item := trackedItem{
		ID:         name,
		Kind:       w.Source,
		Probe:      probeSpec{Mode: probeAgent, Instruction: instruction},
		CheckEvery: &checkEvery,
		DependsOn:  []string{},
		Scope:      scope,
		Last:       map[string]interface{}{},
		Seen:       seen,
		CheckedAt:  w.LastChecked,
		Paused:     !w.Enabled,
		AddedAt:    now,
		UpdatedAt:  now,
	}
	if w.Instructions != "" {
		item.Then = &w.Instructions
	}
	return item
}

// convertAutopilotQueue folds the not-yet-completed queue entries into items:
// an entry already converted from monitored gains merge_mode (and a chain dep
// when it has none); a queue-only entry becomes a pane-less pane item
// (pending/held). Chaining follows the nearest same-repo predecessor rule,
// with a cross-repo entry chained to its immediate predecessor.
func convertAutopilotQueue(items []trackedItem, ap autopilotState, bm map[string]branchMapEntry, now string) []trackedItem {
	if len(ap.Queue) == 0 {
		return items
	}
	completed := map[string]bool{}
	for _, id := range ap.Completed {
		completed[id] = true
	}
	mode := ap.Mode
	if mode == "" {
		mode = config.DefaultAutopilotMergeMode
	}
	repoOf := func(id string) string {
		for _, it := range items {
			if it.ID == id {
				return scopeString(it.Scope, "repo")
			}
		}
		return bm[id].Repo
	}
	var pending []string // not-completed queue entries seen so far, in order
	for _, id := range ap.Queue {
		if completed[id] {
			continue
		}
		pred := ""
		if repo := repoOf(id); repo != "" {
			for i := len(pending) - 1; i >= 0; i-- {
				if repoOf(pending[i]) == repo {
					pred = pending[i]
					break
				}
			}
		}
		if pred == "" && len(pending) > 0 {
			pred = pending[len(pending)-1]
		}
		idx := -1
		for i := range items {
			if items[i].ID == id {
				idx = i
				break
			}
		}
		if idx >= 0 {
			items[idx].Scope["merge_mode"] = mode
			if len(items[idx].DependsOn) == 0 && pred != "" {
				items[idx].DependsOn = []string{pred}
			}
		} else {
			scope := paneScope()
			scope["change"] = id // a queue entry's id IS its change id
			setScopeString(scope, "repo", repoOf(id))
			setScopeString(scope, "branch", bm[id].Branch)
			scope["merge_mode"] = mode
			deps := []string{}
			if pred != "" {
				deps = []string{pred}
			}
			items = append(items, trackedItem{
				ID:        id,
				Kind:      kindPane,
				Probe:     probeSpec{Mode: probePane},
				DependsOn: deps,
				Scope:     scope,
				Last:      map[string]interface{}{},
				AddedAt:   now,
				UpdatedAt: now,
			})
		}
		pending = append(pending, id)
	}
	return items
}

// unionStrings concatenates the lists deduped, first occurrence winning.
func unionStrings(a, b []string) []string {
	out := []string{}
	seen := map[string]bool{}
	for _, s := range append(append([]string{}, a...), b...) {
		if !seen[s] {
			seen[s] = true
			out = append(out, s)
		}
	}
	return out
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
