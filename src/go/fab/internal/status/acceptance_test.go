package status

import (
	"os"
	"path/filepath"
	"testing"
)

const planWithAcceptance = `# Plan: example

## Tasks

- [x] T001 do thing
- [ ] T002 other thing

## Acceptance

- [x] A-001 R1: first criterion
- [x] A-002 R2: second criterion
- [ ] A-003 R3: third criterion
`

// TestLiveAcceptance_CountsCheckboxes covers jznd (b): LiveAcceptance derives
// done/total from plan.md `## Acceptance` checkboxes, ignoring the (possibly
// stale) .status.yaml counter entirely.
func TestLiveAcceptance_CountsCheckboxes(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte(planWithAcceptance), 0644); err != nil {
		t.Fatal(err)
	}
	done, total, ok := LiveAcceptance(dir)
	if !ok {
		t.Fatal("expected ok=true when plan.md has an ## Acceptance section")
	}
	if done != 2 || total != 3 {
		t.Errorf("LiveAcceptance = (%d, %d), want (2, 3)", done, total)
	}
}

// TestLiveAcceptance_ReflectsEdit covers the sed/direct-edit scenario: a
// checkbox toggled by a hook-bypassing edit is reflected immediately, since
// the count is derived at read time rather than trusting a cached counter.
func TestLiveAcceptance_ReflectsEdit(t *testing.T) {
	dir := t.TempDir()
	planPath := filepath.Join(dir, "plan.md")
	if err := os.WriteFile(planPath, []byte(planWithAcceptance), 0644); err != nil {
		t.Fatal(err)
	}
	// Simulate a sed edit that checks A-003 without firing any hook.
	toggled := []byte(`# Plan: example

## Tasks

- [x] T001 do thing
- [ ] T002 other thing

## Acceptance

- [x] A-001 R1: first criterion
- [x] A-002 R2: second criterion
- [x] A-003 R3: third criterion
`)
	if err := os.WriteFile(planPath, toggled, 0644); err != nil {
		t.Fatal(err)
	}
	done, total, ok := LiveAcceptance(dir)
	if !ok || done != 3 || total != 3 {
		t.Errorf("LiveAcceptance after edit = (%d, %d, %v), want (3, 3, true)", done, total, ok)
	}
}

func TestLiveAcceptance_NoPlanFile(t *testing.T) {
	dir := t.TempDir()
	if _, _, ok := LiveAcceptance(dir); ok {
		t.Error("expected ok=false when plan.md is absent (caller falls back to cache)")
	}
}

func TestLiveAcceptance_NoAcceptanceSection(t *testing.T) {
	dir := t.TempDir()
	planNoAccept := `# Plan: example

## Tasks

- [ ] T001 do thing
`
	if err := os.WriteFile(filepath.Join(dir, "plan.md"), []byte(planNoAccept), 0644); err != nil {
		t.Fatal(err)
	}
	if _, _, ok := LiveAcceptance(dir); ok {
		t.Error("expected ok=false when plan.md has no ## Acceptance heading")
	}
}
