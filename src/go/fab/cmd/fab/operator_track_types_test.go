package main

import (
	"strings"
	"testing"
	"time"
)

func TestNormalizeCheckEvery(t *testing.T) {
	tests := []struct {
		name    string
		mode    string
		given   string
		want    *string
		wantErr string
	}{
		{"pane forces null", probePane, "2m", nil, ""},
		{"none forces null", probeNone, "2m", nil, ""},
		{"shell default 5m", probeShell, "", strPtr("5m"), ""},
		{"agent default 5m", probeAgent, "", strPtr("5m"), ""},
		{"explicit value kept verbatim", probeShell, "2m", strPtr("2m"), ""},
		{"floor value accepted", probeAgent, "1m", strPtr("1m"), ""},
		{"below floor rejected", probeShell, "30s", nil, "below the 1m0s floor"},
		{"invalid duration rejected", probeShell, "often", nil, "invalid --check-every"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := normalizeCheckEvery(tc.mode, tc.given)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("normalizeCheckEvery(%q, %q) err = %v, want %q", tc.mode, tc.given, err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("normalizeCheckEvery(%q, %q): %v", tc.mode, tc.given, err)
			}
			if (got == nil) != (tc.want == nil) || (got != nil && *got != *tc.want) {
				t.Errorf("normalizeCheckEvery(%q, %q) = %v, want %v", tc.mode, tc.given, got, tc.want)
			}
		})
	}
}

// strPtr is shared test scaffolding (pane_map_test.go).

func TestValidateTrackedItem(t *testing.T) {
	shellItem := trackedItem{
		ID: "x", Kind: kindShell,
		Probe: probeSpec{Mode: probeShell, Argv: []string{"true"}, Fields: []string{"a"}},
	}
	tests := []struct {
		name    string
		mutate  func(*trackedItem)
		wantErr string
	}{
		{"valid shell item", nil, ""},
		{"unknown kind", func(it *trackedItem) { it.Kind = "email" }, "unknown kind"},
		{"kind-forbidden probe mode", func(it *trackedItem) { it.Kind = kindNote }, "does not allow probe mode"},
		{"shell probe without argv", func(it *trackedItem) { it.Probe.Argv = nil }, "requires --argv"},
		{"shell probe without fields", func(it *trackedItem) { it.Probe.Fields = nil }, "requires --fields"},
		{"malformed done_when names the clause", func(it *trackedItem) { it.DoneWhen = strPtr(`state == MERGED`) }, "invalid --done-when"},
		{"valid done_when", func(it *trackedItem) { it.DoneWhen = strPtr(`state == "MERGED"`) }, ""},
		{"note text within cap", func(it *trackedItem) {
			it.Kind, it.Probe.Mode, it.Text = kindNote, probeNone, strings.Repeat("x", trackNoteTextCap)
			it.Probe.Argv, it.Probe.Fields = nil, nil
		}, ""},
		{"note text over cap", func(it *trackedItem) {
			it.Kind, it.Probe.Mode, it.Text = kindNote, probeNone, strings.Repeat("x", trackNoteTextCap+1)
			it.Probe.Argv, it.Probe.Fields = nil, nil
		}, "over the 500-character cap"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			it := shellItem
			if tc.mutate != nil {
				tc.mutate(&it)
			}
			err := validateTrackedItem(it)
			if tc.wantErr == "" {
				if err != nil {
					t.Errorf("validateTrackedItem = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("validateTrackedItem = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestValidateTrackedDeps(t *testing.T) {
	existing := []trackedItem{{ID: "a"}, {ID: "b"}}
	tests := []struct {
		name    string
		it      trackedItem
		wantErr string
	}{
		{"no deps", trackedItem{ID: "c"}, ""},
		{"known deps", trackedItem{ID: "c", DependsOn: []string{"a", "b"}}, ""},
		{"unknown dep", trackedItem{ID: "c", DependsOn: []string{"zz"}}, "no tracked item with that id"},
		{"self dependency", trackedItem{ID: "a", DependsOn: []string{"a"}}, "names the item itself"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := validateTrackedDeps(tc.it, existing)
			if tc.wantErr == "" && err != nil {
				t.Errorf("validateTrackedDeps = %v, want nil", err)
			}
			if tc.wantErr != "" && (err == nil || !strings.Contains(err.Error(), tc.wantErr)) {
				t.Errorf("validateTrackedDeps = %v, want %q", err, tc.wantErr)
			}
		})
	}
}

func TestTrackedItemDone(t *testing.T) {
	tests := []struct {
		name string
		it   trackedItem
		want bool
	}{
		{"null done_when is never done", trackedItem{Kind: kindPane}, false},
		{"done_when fires on last", trackedItem{
			DoneWhen: strPtr(`state == "MERGED"`),
			Last:     map[string]interface{}{"state": "MERGED"},
		}, true},
		{"done_when false on last", trackedItem{
			DoneWhen: strPtr(`state == "MERGED"`),
			Last:     map[string]interface{}{"state": "OPEN"},
		}, false},
		{"unparseable stored done_when is not done (fail-safe)", trackedItem{
			DoneWhen: strPtr(`state == MERGED`),
			Last:     map[string]interface{}{"state": "MERGED"},
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := trackedItemDone(tc.it); got != tc.want {
				t.Errorf("trackedItemDone = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTrackItemStale(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	ago := func(d time.Duration) *string {
		s := now.Add(-d).Format(time.RFC3339)
		return &s
	}
	tests := []struct {
		name string
		it   trackedItem
		want bool
	}{
		{"agent item past 2x cadence is stale", trackedItem{
			Probe: probeSpec{Mode: probeAgent}, CheckEvery: strPtr("5m"), CheckedAt: ago(11 * time.Minute),
		}, true},
		{"agent item within 2x cadence is fresh", trackedItem{
			Probe: probeSpec{Mode: probeAgent}, CheckEvery: strPtr("5m"), CheckedAt: ago(9 * time.Minute),
		}, false},
		{"null checked_at falls back to added_at", trackedItem{
			Probe: probeSpec{Mode: probeAgent}, CheckEvery: strPtr("5m"),
			AddedAt: now.Add(-11 * time.Minute).Format(time.RFC3339),
		}, true},
		{"shell items never stale (the binary probes them)", trackedItem{
			Probe: probeSpec{Mode: probeShell}, CheckEvery: strPtr("5m"), CheckedAt: ago(time.Hour),
		}, false},
		{"null check_every never stale", trackedItem{
			Probe: probeSpec{Mode: probeAgent}, CheckedAt: ago(time.Hour),
		}, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := trackItemStale(tc.it, now); got != tc.want {
				t.Errorf("trackItemStale = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestTrackHeldDep(t *testing.T) {
	done := trackedItem{ID: "a", DoneWhen: strPtr(`x == 1`), Last: map[string]interface{}{"x": 1}}
	open := trackedItem{ID: "b", DoneWhen: strPtr(`x == 1`), Last: map[string]interface{}{"x": 0}}
	items := []trackedItem{done, open}
	tests := []struct {
		name string
		it   trackedItem
		want string
	}{
		{"satisfied dep", trackedItem{ID: "c", DependsOn: []string{"a"}}, ""},
		{"unsatisfied dep", trackedItem{ID: "c", DependsOn: []string{"b"}}, "b"},
		{"missing dep is unsatisfied", trackedItem{ID: "c", DependsOn: []string{"zz"}}, "zz"},
		{"first unsatisfied dep wins", trackedItem{ID: "c", DependsOn: []string{"a", "b"}}, "b"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := trackHeldDep(items, tc.it); got != tc.want {
				t.Errorf("trackHeldDep = %q, want %q", got, tc.want)
			}
		})
	}
}
