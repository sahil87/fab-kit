package hooklib

import (
	"strings"
	"testing"
)

func TestParsePayload_Valid(t *testing.T) {
	input := `{"tool_input":{"file_path":"fab/changes/260310-bvc6-test/intake.md"}}`
	r := strings.NewReader(input)
	path, err := ParsePayload(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "fab/changes/260310-bvc6-test/intake.md" {
		t.Errorf("got %q, want %q", path, "fab/changes/260310-bvc6-test/intake.md")
	}
}

func TestParsePayload_MalformedJSON(t *testing.T) {
	input := `{invalid json}`
	r := strings.NewReader(input)
	_, err := ParsePayload(r)
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParsePayload_MissingFilePath(t *testing.T) {
	input := `{"tool_input":{}}`
	r := strings.NewReader(input)
	path, err := ParsePayload(r)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Errorf("expected empty path, got %q", path)
	}
}

func TestParsePayload_Empty(t *testing.T) {
	r := strings.NewReader("")
	_, err := ParsePayload(r)
	if err == nil {
		t.Error("expected error for empty input")
	}
}

func TestMatchArtifactPath_AbsoluteIntake(t *testing.T) {
	match, ok := MatchArtifactPath("/home/user/project/fab/changes/260310-bvc6-test/intake.md")
	if !ok {
		t.Fatal("expected match")
	}
	if match.ChangeFolder != "260310-bvc6-test" {
		t.Errorf("ChangeFolder = %q, want %q", match.ChangeFolder, "260310-bvc6-test")
	}
	if match.Artifact != "intake.md" {
		t.Errorf("Artifact = %q, want %q", match.Artifact, "intake.md")
	}
}

// TestMatchArtifactPath_RelativePlan covers a relative artifact path that
// matches. spec.md is no longer a recognized artifact (1.10.0), so plan.md
// stands in for the relative-path case.
func TestMatchArtifactPath_RelativePlan(t *testing.T) {
	match, ok := MatchArtifactPath("fab/changes/260310-bvc6-test/plan.md")
	if !ok {
		t.Fatal("expected match")
	}
	if match.ChangeFolder != "260310-bvc6-test" {
		t.Errorf("ChangeFolder = %q, want %q", match.ChangeFolder, "260310-bvc6-test")
	}
	if match.Artifact != "plan.md" {
		t.Errorf("Artifact = %q, want %q", match.Artifact, "plan.md")
	}
}

// TestMatchArtifactPath_LegacySpecRejected verifies a leftover spec.md no
// longer matches (1.10.0), so editing it cannot fire the score hook and
// overwrite the authoritative intake confidence — mirroring the tasks.md
// rejection.
func TestMatchArtifactPath_LegacySpecRejected(t *testing.T) {
	if _, ok := MatchArtifactPath("fab/changes/260310-bvc6-test/spec.md"); ok {
		t.Error("expected no match for legacy spec.md")
	}
}

func TestMatchArtifactPath_Plan(t *testing.T) {
	match, ok := MatchArtifactPath("fab/changes/my-change/plan.md")
	if !ok {
		t.Fatal("expected match")
	}
	if match.Artifact != "plan.md" {
		t.Errorf("Artifact = %q, want %q", match.Artifact, "plan.md")
	}
}

func TestMatchArtifactPath_LegacyTasksRejected(t *testing.T) {
	if _, ok := MatchArtifactPath("fab/changes/my-change/tasks.md"); ok {
		t.Error("expected no match for legacy tasks.md")
	}
}

func TestMatchArtifactPath_LegacyChecklistRejected(t *testing.T) {
	if _, ok := MatchArtifactPath("fab/changes/my-change/checklist.md"); ok {
		t.Error("expected no match for legacy checklist.md")
	}
}

func TestMatchArtifactPath_NonFabPath(t *testing.T) {
	_, ok := MatchArtifactPath("src/main.go")
	if ok {
		t.Error("expected no match for non-fab path")
	}
}

func TestMatchArtifactPath_UnknownArtifact(t *testing.T) {
	_, ok := MatchArtifactPath("fab/changes/my-change/other.md")
	if ok {
		t.Error("expected no match for unknown artifact")
	}
}

func TestMatchArtifactPath_EmptyFolder(t *testing.T) {
	_, ok := MatchArtifactPath("fab/changes//intake.md")
	if ok {
		t.Error("expected no match for empty folder")
	}
}

func TestMatchArtifactPath_NoFolder(t *testing.T) {
	_, ok := MatchArtifactPath("fab/changes/intake.md")
	if ok {
		t.Error("expected no match when no folder separator")
	}
}

func TestMatchArtifactPath_NotFabPrefix(t *testing.T) {
	_, ok := MatchArtifactPath("not-fab/changes/name/intake.md")
	if ok {
		t.Error("expected no match for non-fab prefix")
	}
}

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
