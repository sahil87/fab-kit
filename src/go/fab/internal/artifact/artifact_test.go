package artifact

import (
	"strings"
	"testing"
)

func TestInferChangeType_Fix(t *testing.T) {
	tests := []struct {
		content string
		want    string
	}{
		{"This fixes a bug in the parser", "fix"},
		{"Fix broken regression test", "fix"},
		{"A REGRESSION in the build", "fix"},
	}
	for _, tt := range tests {
		got := InferChangeType(tt.content)
		if got != tt.want {
			t.Errorf("InferChangeType(%q) = %q, want %q", tt.content, got, tt.want)
		}
	}
}

// TestInferChangeType_MustFixNonMatch covers jznd (a): a feature intake that
// merely mentions "must-fix"/"must fix" in passing must NOT be misclassified
// as `fix` just because RE2 treats the hyphen as a word boundary.
func TestInferChangeType_MustFixNonMatch(t *testing.T) {
	tests := []struct {
		content string
		want    string
	}{
		{"Add a widget; reviewers flagged it must-fix before merge", "feat"},
		{"This is a must fix item per the rubric, but it adds a feature", "feat"},
		{"A must-fix checklist gate for the new dashboard", "feat"},
	}
	for _, tt := range tests {
		got := InferChangeType(tt.content)
		if got != tt.want {
			t.Errorf("InferChangeType(%q) = %q, want %q (must-fix should not trigger fix)", tt.content, got, tt.want)
		}
	}
}

// TestInferChangeType_HyphenatedFixCompounds covers jznd (a): fix-describing
// compounds still classify `fix`. bug-fix/bug-free match via the standalone
// `bug` token; hot-fix matches via `fix` (no `must` directive guard applies).
func TestInferChangeType_HyphenatedFixCompounds(t *testing.T) {
	tests := []string{
		"Ship the bug-fix for the parser",
		"Apply the hot-fix to production",
		"Make the allocator bug-free",
	}
	for _, content := range tests {
		if got := InferChangeType(content); got != "fix" {
			t.Errorf("InferChangeType(%q) = %q, want fix", content, got)
		}
	}
}

func TestInferChangeType_Refactor(t *testing.T) {
	tests := []struct {
		content string
		want    string
	}{
		{"Refactor the module layout", "refactor"},
		{"Restructure the internal packages", "refactor"},
		{"Consolidate duplicate code", "refactor"},
		{"Split large function", "refactor"},
		{"Rename variables for clarity", "refactor"},
		{"Redesign the API surface", "refactor"},
	}
	for _, tt := range tests {
		got := InferChangeType(tt.content)
		if got != tt.want {
			t.Errorf("InferChangeType(%q) = %q, want %q", tt.content, got, tt.want)
		}
	}
}

func TestInferChangeType_Docs(t *testing.T) {
	got := InferChangeType("Update the README guide")
	if got != "docs" {
		t.Errorf("got %q, want %q", got, "docs")
	}
}

func TestInferChangeType_Test(t *testing.T) {
	got := InferChangeType("Improve test coverage")
	if got != "test" {
		t.Errorf("got %q, want %q", got, "test")
	}
}

func TestInferChangeType_CI(t *testing.T) {
	got := InferChangeType("Fix the CI pipeline")
	// "Fix" comes first in order, so this should match "fix"
	got2 := InferChangeType("Update the deployment pipeline")
	if got2 != "ci" {
		t.Errorf("got %q, want %q", got2, "ci")
	}
	// But "fix" takes precedence
	if got != "fix" {
		t.Errorf("got %q, want %q — fix should take precedence over ci", got, "fix")
	}
}

func TestInferChangeType_Chore(t *testing.T) {
	got := InferChangeType("Housekeeping: update dependencies")
	if got != "chore" {
		t.Errorf("got %q, want %q", got, "chore")
	}
}

func TestInferChangeType_Default(t *testing.T) {
	got := InferChangeType("Add a new feature for the widget")
	if got != "feat" {
		t.Errorf("got %q, want %q", got, "feat")
	}
}

func TestInferChangeType_CaseInsensitive(t *testing.T) {
	got := InferChangeType("REFACTOR the whole thing")
	if got != "refactor" {
		t.Errorf("got %q, want %q", got, "refactor")
	}
}

func TestInferChangeType_FirstMatchWins(t *testing.T) {
	// "fix" appears before "refactor" in order
	got := InferChangeType("Fix and refactor the module")
	if got != "fix" {
		t.Errorf("got %q, want %q — first match should win", got, "fix")
	}
}

func TestHasSectionHeading_Present(t *testing.T) {
	content := `# Plan: example

## Tasks

- [ ] T001 do thing

## Acceptance

- [ ] A-001 check thing
`
	if !HasSectionHeading(content, SectionTasks) {
		t.Error("expected ## Tasks heading to be detected")
	}
	if !HasSectionHeading(content, SectionAcceptance) {
		t.Error("expected ## Acceptance heading to be detected")
	}
}

func TestHasSectionHeading_Missing(t *testing.T) {
	content := `# Plan: example

## Tasks

- [ ] T001 do thing
`
	if !HasSectionHeading(content, SectionTasks) {
		t.Error("expected ## Tasks heading to be detected")
	}
	if HasSectionHeading(content, SectionAcceptance) {
		t.Error("expected ## Acceptance heading to be absent")
	}
}

func TestHasSectionHeading_DoesNotMatchPrefix(t *testing.T) {
	content := `## TasksAndStuff

- [ ] T001 do thing
`
	if HasSectionHeading(content, SectionTasks) {
		t.Error("## TasksAndStuff should not match the SectionTasks heading")
	}
}

func TestCountSectionItemsBounded_TasksAndAcceptance(t *testing.T) {
	content := `# Plan: example

## Tasks

### Phase 1: Setup
- [ ] T001 First task
- [x] T002 Done task

### Phase 2: Core
- [ ] T003 Another task
- [ ] T004 Third task
- [x] T005 Also done

## Execution Order

- T001 blocks T003

## Acceptance

- [ ] A-001 unmet
- [x] A-002 met
- [ ] A-003 unmet
`
	tasks := CountSectionItemsBounded(content, SectionTasks)
	if tasks != 5 {
		t.Errorf("Tasks count: got %d, want 5", tasks)
	}
	acceptance := CountSectionItemsBounded(content, SectionAcceptance)
	if acceptance != 3 {
		t.Errorf("Acceptance count: got %d, want 3", acceptance)
	}
	completed := CountCompletedSectionItemsBounded(content, SectionAcceptance)
	if completed != 1 {
		t.Errorf("Acceptance completed: got %d, want 1", completed)
	}
}

func TestCountSectionItemsBounded_StopsAtNextHeading(t *testing.T) {
	// Items only under Tasks should be counted; nothing past `## Acceptance`.
	content := `## Tasks

- [ ] T001 inside tasks
- [x] T002 inside tasks

## Acceptance

- [ ] A-001 not a task
- [ ] A-002 not a task
`
	if got := CountSectionItemsBounded(content, SectionTasks); got != 2 {
		t.Errorf("Tasks count: got %d, want 2", got)
	}
}

func TestCountSectionItemsBounded_OversizedLineInsideSection(t *testing.T) {
	// The old in-memory scanner hit bufio.ErrTooLong on a >64KB line and
	// silently stopped, undercounting items — wrong counts were *persisted*
	// into .status.yaml by the artifact-write hook, not just displayed.
	long := strings.Repeat("x", 70*1024)
	content := "## Tasks\n\n" +
		"- [ ] T001 before the long line\n" +
		"- [x] T002 " + long + "\n" +
		"- [ ] T003 after the long line\n" +
		"\n## Acceptance\n\n" +
		"- [x] A-001 met\n"

	if got := CountSectionItemsBounded(content, SectionTasks); got != 3 {
		t.Errorf("Tasks count: got %d, want 3 (items after the oversized line must be counted)", got)
	}
	if got := CountCompletedSectionItemsBounded(content, SectionTasks); got != 1 {
		t.Errorf("Tasks completed: got %d, want 1 (the oversized item itself)", got)
	}
	if !HasSectionHeading(content, SectionAcceptance) {
		t.Error("Acceptance heading after the oversized line must be found")
	}
}

func TestCountSectionItemsBounded_MissingSectionReturnsZero(t *testing.T) {
	content := `# Plan: example

## Acceptance

- [ ] A-001 unmet
`
	// Tasks section absent — bounded scan returns 0; callers should use
	// HasSectionHeading to distinguish "missing" from "empty".
	if got := CountSectionItemsBounded(content, SectionTasks); got != 0 {
		t.Errorf("Tasks count when section absent: got %d, want 0", got)
	}
}
