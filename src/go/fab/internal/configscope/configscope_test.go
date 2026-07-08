package configscope

import "testing"

// TestScopeFor pins the decision-6 taxonomy (config-upgrade.md): agent/providers
// are preference-class (`both`); every other top-level key is semantics-class
// (`project`); an unknown key reports (_, false). The cascade loader keys on this
// table to prune project-scoped fields out of the system layer, so the taxonomy
// is single-sourced here and this test guards it.
func TestScopeFor(t *testing.T) {
	want := map[string]Scope{
		"project":             ScopeProject,
		"source_paths":        ScopeProject,
		"test_paths":          ScopeProject,
		"true_impact_exclude": ScopeProject,
		"checklist":           ScopeProject,
		"providers":           ScopeBoth,
		"agent":               ScopeBoth,
		"stage_hooks":         ScopeProject,
		"branch_prefix":       ScopeProject,
		"fab_version":         ScopeProject,
	}
	for key, wantScope := range want {
		got, ok := ScopeFor(key)
		if !ok {
			t.Errorf("ScopeFor(%q) reported unknown, want %q", key, wantScope)
			continue
		}
		if got != wantScope {
			t.Errorf("ScopeFor(%q) = %q, want %q", key, got, wantScope)
		}
	}
}

// TestScopeFor_Unknown: an unrecognized top-level key reports (_, false). The
// loader treats such a key as ignored-silently (matching project-file behavior),
// so the bool — not a default scope — is what a caller keys on.
func TestScopeFor_Unknown(t *testing.T) {
	if s, ok := ScopeFor("no_such_key"); ok {
		t.Errorf("ScopeFor(unknown) = (%q, true), want (_, false)", s)
	}
}

// TestValid: Valid accepts exactly the three known scope values and rejects
// anything else (the configref registry lint relies on this).
func TestValid(t *testing.T) {
	for _, s := range []Scope{ScopeProject, ScopeSystem, ScopeBoth} {
		if !Valid(s) {
			t.Errorf("Valid(%q) = false, want true", s)
		}
	}
	for _, s := range []Scope{"", "Project", "user", "global"} {
		if Valid(s) {
			t.Errorf("Valid(%q) = true, want false", s)
		}
	}
}
