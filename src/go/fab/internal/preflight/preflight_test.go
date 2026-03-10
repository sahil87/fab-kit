package preflight

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testStatusYAML = `id: abcd
name: 260310-abcd-my-change
created: "2026-03-10T12:00:00Z"
created_by: test-user
change_type: feat
issues: []
progress:
  intake: done
  spec: active
  tasks: pending
  apply: pending
  review: pending
  hydrate: pending
  ship: pending
  review-pr: pending
checklist:
  generated: false
  path: checklist.md
  completed: 0
  total: 0
confidence:
  certain: 3
  confident: 1
  tentative: 0
  unresolved: 0
  score: 4.7
stage_metrics: {}
prs: []
last_updated: "2026-03-10T12:00:00Z"
`

const minimalConfigYAML = `project:
  name: test-project
`

// setupPreflightFixture creates a complete project fixture and returns fabRoot.
func setupPreflightFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")

	// Project files
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0755)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte(minimalConfigYAML), 0644)
	os.WriteFile(filepath.Join(fabRoot, "project", "constitution.md"), []byte("# Constitution\n"), 0644)

	// Change directory
	changeDir := filepath.Join(fabRoot, "changes", "260310-abcd-my-change")
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte(testStatusYAML), 0644)

	// Active symlink
	symlinkPath := filepath.Join(dir, ".fab-status.yaml")
	os.Symlink("fab/changes/260310-abcd-my-change/.status.yaml", symlinkPath)

	return fabRoot
}

func TestRun_ValidRepo(t *testing.T) {
	fabRoot := setupPreflightFixture(t)

	result, err := Run(fabRoot, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.ID != "abcd" {
		t.Errorf("ID = %q, want %q", result.ID, "abcd")
	}
	if result.Name != "260310-abcd-my-change" {
		t.Errorf("Name = %q, want %q", result.Name, "260310-abcd-my-change")
	}
	if result.ChangeDir != "fab/changes/260310-abcd-my-change" {
		t.Errorf("ChangeDir = %q, want %q", result.ChangeDir, "fab/changes/260310-abcd-my-change")
	}
	if result.Stage != "spec" {
		t.Errorf("Stage = %q, want %q", result.Stage, "spec")
	}
	if len(result.Progress) != 8 {
		t.Errorf("Progress has %d stages, want 8", len(result.Progress))
	}
}

func TestRun_MissingConfig(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0755)
	// No project/config.yaml

	_, err := Run(fabRoot, "")
	if err == nil {
		t.Fatal("expected error for missing config.yaml")
	}
	if !strings.Contains(err.Error(), "config.yaml") {
		t.Errorf("error should mention config.yaml, got: %v", err)
	}
}

func TestRun_MissingConstitution(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0755)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte(minimalConfigYAML), 0644)
	// No constitution.md

	_, err := Run(fabRoot, "")
	if err == nil {
		t.Fatal("expected error for missing constitution.md")
	}
	if !strings.Contains(err.Error(), "constitution.md") {
		t.Errorf("error should mention constitution.md, got: %v", err)
	}
}

func TestRun_MissingActiveChange(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "project"), 0755)
	os.MkdirAll(filepath.Join(fabRoot, "changes"), 0755)
	os.WriteFile(filepath.Join(fabRoot, "project", "config.yaml"), []byte(minimalConfigYAML), 0644)
	os.WriteFile(filepath.Join(fabRoot, "project", "constitution.md"), []byte("# Constitution\n"), 0644)
	// No .fab-status.yaml symlink, no changes

	_, err := Run(fabRoot, "")
	if err == nil {
		t.Fatal("expected error for missing active change")
	}
}

func TestRun_OverrideResolution(t *testing.T) {
	fabRoot := setupPreflightFixture(t)

	// Remove the symlink to force override-only resolution
	repoRoot := filepath.Dir(fabRoot)
	os.Remove(filepath.Join(repoRoot, ".fab-status.yaml"))

	result, err := Run(fabRoot, "abcd")
	if err != nil {
		t.Fatalf("Run with override failed: %v", err)
	}

	if result.Name != "260310-abcd-my-change" {
		t.Errorf("Name = %q, want %q", result.Name, "260310-abcd-my-change")
	}
}

func TestFormatYAML(t *testing.T) {
	fabRoot := setupPreflightFixture(t)

	result, err := Run(fabRoot, "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	output := FormatYAML(result)

	requiredFields := []string{"id:", "name:", "change_dir:", "stage:", "progress:", "checklist:", "confidence:"}
	for _, field := range requiredFields {
		if !strings.Contains(output, field) {
			t.Errorf("FormatYAML output missing %q", field)
		}
	}

	if !strings.Contains(output, "abcd") {
		t.Error("FormatYAML output should contain the change ID")
	}
	if !strings.Contains(output, "260310-abcd-my-change") {
		t.Error("FormatYAML output should contain the change name")
	}
}
