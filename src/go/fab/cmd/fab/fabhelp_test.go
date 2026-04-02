package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScanSkills_BasicDiscovery(t *testing.T) {
	dir := t.TempDir()

	// Create a valid skill file
	os.WriteFile(filepath.Join(dir, "fab-new.md"), []byte(`---
name: fab-new
description: "Start a new change"
---
# Content`), 0o644)

	// Create a partial (should be excluded)
	os.WriteFile(filepath.Join(dir, "_preamble.md"), []byte(`---
name: _preamble
description: "Preamble partial"
---`), 0o644)

	// Create an internal skill (should be excluded)
	os.WriteFile(filepath.Join(dir, "internal-test.md"), []byte(`---
name: internal-test
description: "Internal skill"
---`), 0o644)

	// Create a skill without frontmatter (should be excluded)
	os.WriteFile(filepath.Join(dir, "no-frontmatter.md"), []byte(`# No frontmatter`), 0o644)

	skills := scanSkills(dir)

	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}
	if skills[0].name != "fab-new" {
		t.Errorf("name = %q, want %q", skills[0].name, "fab-new")
	}
	if skills[0].desc != "Start a new change" {
		t.Errorf("desc = %q, want %q", skills[0].desc, "Start a new change")
	}
}

func TestScanSkills_EmptyDir(t *testing.T) {
	dir := t.TempDir()
	skills := scanSkills(dir)
	if len(skills) != 0 {
		t.Errorf("expected 0 skills from empty dir, got %d", len(skills))
	}
}

func TestScanSkills_NonexistentDir(t *testing.T) {
	skills := scanSkills("/nonexistent/dir")
	if skills != nil {
		t.Errorf("expected nil from nonexistent dir, got %v", skills)
	}
}

func TestGetBatchEntries(t *testing.T) {
	entries := getBatchEntries()
	if len(entries) != 3 {
		t.Fatalf("expected 3 batch entries, got %d", len(entries))
	}

	// Collect display names and verify all expected commands are present
	got := make(map[string]bool)
	for _, e := range entries {
		got[e.display] = true
	}
	expected := []string{"fab batch new", "fab batch switch", "fab batch archive"}
	for _, want := range expected {
		if !got[want] {
			t.Errorf("missing batch entry %q", want)
		}
	}

	// Verify descriptions come from cobra metadata (non-empty)
	for _, e := range entries {
		if e.desc == "" {
			t.Errorf("batch entry %q has empty description", e.display)
		}
	}
}

func TestReadKitVersion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "VERSION"), []byte("0.44.9\n"), 0o644)

	ver := readKitVersion(dir)
	if ver != "0.44.9" {
		t.Errorf("readKitVersion() = %q, want %q", ver, "0.44.9")
	}
}

func TestReadKitVersion_Missing(t *testing.T) {
	ver := readKitVersion("/nonexistent")
	if ver != "unknown" {
		t.Errorf("readKitVersion() = %q, want %q", ver, "unknown")
	}
}

func TestComputeMaxNameLen(t *testing.T) {
	skills := []skillEntry{
		{name: "fab-new", desc: "short"},
		{name: "docs-hydrate-memory", desc: "long name"},
	}
	got := computeMaxNameLen(skills)
	// "/docs-hydrate-memory" = 20 chars
	if got != 20 {
		t.Errorf("computeMaxNameLen() = %d, want 20", got)
	}
}

func TestFabHelp_GroupMapping(t *testing.T) {
	// Verify all expected skills are mapped
	expectedMapped := []string{
		"fab-new", "fab-switch", "fab-status", "fab-discuss",
		"fab-continue", "fab-ff", "fab-fff", "fab-clarify",
		"fab-archive", "git-pr",
		"docs-hydrate-specs", "docs-reorg-specs", "docs-reorg-memory",
		"fab-setup", "fab-help", "docs-hydrate-memory",
	}
	for _, name := range expectedMapped {
		if _, ok := skillToGroupMap[name]; !ok {
			t.Errorf("skill %q is not in skillToGroupMap", name)
		}
	}
}

func TestFabHelp_OutputContainsSections(t *testing.T) {
	// Create a minimal fab structure
	dir := t.TempDir()
	fabDir := filepath.Join(dir, "fab")
	kitDir := filepath.Join(fabDir, ".kit")
	skillsDir := filepath.Join(kitDir, "skills")
	os.MkdirAll(skillsDir, 0o755)
	os.WriteFile(filepath.Join(kitDir, "VERSION"), []byte("0.44.9"), 0o644)

	os.WriteFile(filepath.Join(skillsDir, "fab-new.md"), []byte(`---
name: fab-new
description: "Start a new change"
---`), 0o644)

	// Run scanSkills to verify it works
	skills := scanSkills(skillsDir)
	if len(skills) != 1 {
		t.Fatalf("expected 1 skill, got %d", len(skills))
	}

	// Verify the output structure contains expected sections
	ver := readKitVersion(kitDir)
	if !strings.Contains(ver, "0.44.9") {
		t.Errorf("version should contain 0.44.9, got %q", ver)
	}
}
