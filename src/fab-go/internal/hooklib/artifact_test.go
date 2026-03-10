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

func TestMatchArtifactPath_RelativeSpec(t *testing.T) {
	match, ok := MatchArtifactPath("fab/changes/260310-bvc6-test/spec.md")
	if !ok {
		t.Fatal("expected match")
	}
	if match.ChangeFolder != "260310-bvc6-test" {
		t.Errorf("ChangeFolder = %q, want %q", match.ChangeFolder, "260310-bvc6-test")
	}
	if match.Artifact != "spec.md" {
		t.Errorf("Artifact = %q, want %q", match.Artifact, "spec.md")
	}
}

func TestMatchArtifactPath_Tasks(t *testing.T) {
	match, ok := MatchArtifactPath("fab/changes/my-change/tasks.md")
	if !ok {
		t.Fatal("expected match")
	}
	if match.Artifact != "tasks.md" {
		t.Errorf("Artifact = %q, want %q", match.Artifact, "tasks.md")
	}
}

func TestMatchArtifactPath_Checklist(t *testing.T) {
	match, ok := MatchArtifactPath("fab/changes/my-change/checklist.md")
	if !ok {
		t.Fatal("expected match")
	}
	if match.Artifact != "checklist.md" {
		t.Errorf("Artifact = %q, want %q", match.Artifact, "checklist.md")
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

func TestCountUncheckedTasks(t *testing.T) {
	content := `# Tasks
- [ ] T001 First task
- [x] T002 Done task
- [ ] T003 Another task
- [ ] T004 Third task
Some other text
- [x] T005 Also done
`
	got := CountUncheckedTasks(content)
	if got != 3 {
		t.Errorf("got %d, want 3", got)
	}
}

func TestCountUncheckedTasks_Empty(t *testing.T) {
	got := CountUncheckedTasks("")
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestCountUncheckedTasks_AllChecked(t *testing.T) {
	content := `- [x] T001 Done
- [x] T002 Done
`
	got := CountUncheckedTasks(content)
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestCountChecklistItems(t *testing.T) {
	content := `# Checklist
- [ ] CHK-001 First check
- [x] CHK-002 Done check
- [ ] CHK-003 Another check
- [x] CHK-004 Also done
Some text
`
	got := CountChecklistItems(content)
	if got != 4 {
		t.Errorf("got %d, want 4", got)
	}
}

func TestCountChecklistItems_Empty(t *testing.T) {
	got := CountChecklistItems("")
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}

func TestCountChecklistItems_NoItems(t *testing.T) {
	content := `# Checklist
No items here
Just text
`
	got := CountChecklistItems(content)
	if got != 0 {
		t.Errorf("got %d, want 0", got)
	}
}
