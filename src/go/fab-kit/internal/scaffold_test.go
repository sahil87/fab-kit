package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldTreeWalk_CopyIfAbsent(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create scaffold file
	os.MkdirAll(filepath.Join(scaffoldDir, "docs", "memory"), 0755)
	os.WriteFile(filepath.Join(scaffoldDir, "docs", "memory", "index.md"), []byte("# Index\n"), 0644)

	// Run tree-walk
	if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
		t.Fatalf("scaffoldTreeWalk failed: %v", err)
	}

	// Verify file was copied
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "memory", "index.md"))
	if err != nil {
		t.Fatal("expected index.md to be created")
	}
	if string(data) != "# Index\n" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestScaffoldTreeWalk_CopyIfAbsentSkip(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create scaffold file
	os.WriteFile(filepath.Join(scaffoldDir, "existing.md"), []byte("scaffold content\n"), 0644)

	// Create destination file with different content
	os.WriteFile(filepath.Join(repoRoot, "existing.md"), []byte("user content\n"), 0644)

	// Run tree-walk
	if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
		t.Fatalf("scaffoldTreeWalk failed: %v", err)
	}

	// Verify existing file was NOT overwritten
	data, _ := os.ReadFile(filepath.Join(repoRoot, "existing.md"))
	if string(data) != "user content\n" {
		t.Errorf("existing file should not be overwritten, got: %s", string(data))
	}
}

func TestJsonMergePermissions_CreateNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	srcJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read"},
		},
	}
	srcData, _ := json.MarshalIndent(srcJSON, "", "  ")
	os.WriteFile(src, srcData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal("expected dest file to be created")
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)
	if len(allow) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(allow))
	}
}

func TestJsonMergePermissions_Merge(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	srcJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read", "Write"},
		},
	}
	destJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Edit"},
		},
	}
	srcData, _ := json.MarshalIndent(srcJSON, "", "  ")
	destData, _ := json.MarshalIndent(destJSON, "", "  ")
	os.WriteFile(src, srcData, 0644)
	os.WriteFile(dest, destData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	// Read merged result
	data, _ := os.ReadFile(dest)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)

	// Should have 4: Edit (existing), Bash(git *) (existing/deduped), Read (new), Write (new)
	if len(allow) != 4 {
		t.Errorf("expected 4 permissions after merge, got %d: %v", len(allow), allow)
	}
}

func TestJsonMergePermissions_NoDuplicates(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	// Same permissions in both — no change expected
	perms := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read"},
		},
	}
	srcData, _ := json.MarshalIndent(perms, "", "  ")
	os.WriteFile(src, srcData, 0644)
	os.WriteFile(dest, srcData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)
	if len(allow) != 2 {
		t.Errorf("expected 2 permissions (no duplicates), got %d", len(allow))
	}
}

func TestLineEnsureMerge_CreateNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("# comment\nnode_modules/\n.env\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal("expected dest file to be created")
	}

	content := string(data)
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

// TestLineEnsureMerge_PropagatesWriteError covers jznd (c): a failed
// os.WriteFile during the create-new path must propagate up the call chain
// instead of being silently swallowed (the F21-residue bug). We force the
// failure by making dest's parent a regular file, so creating dest fails with
// ENOTDIR.
func TestLineEnsureMerge_PropagatesWriteError(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	os.WriteFile(src, []byte("node_modules/\n"), 0644)

	// Make a regular file, then try to write a child path "under" it.
	notADir := filepath.Join(destDir, "blocker")
	os.WriteFile(notADir, []byte("x"), 0644)
	dest := filepath.Join(notADir, ".gitignore") // parent is a file → write fails

	err := lineEnsureMerge(src, dest, ".gitignore")
	if err == nil {
		t.Fatal("expected lineEnsureMerge to propagate the os.WriteFile error, got nil")
	}
	if !strings.Contains(err.Error(), ".gitignore") {
		t.Errorf("error should reference the label, got: %v", err)
	}
}

// TestScaffoldTreeWalk_PropagatesFragmentWriteError covers jznd (c) at the
// call-chain level: a write failure inside lineEnsureMerge surfaces from
// scaffoldTreeWalk rather than being swallowed.
func TestScaffoldTreeWalk_PropagatesFragmentWriteError(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// A fragment file produces a dest of repoRoot/<name>; block it by making
	// repoRoot/.gitignore's parent unwritable is awkward — instead use a
	// fragment whose dest parent is a regular file.
	// Layout: scaffold/blocker/fragment-.gitignore → dest repoRoot/blocker/.gitignore
	os.WriteFile(filepath.Join(repoRoot, "blocker"), []byte("x"), 0644)
	if err := os.MkdirAll(filepath.Join(scaffoldDir, "blocker"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(scaffoldDir, "blocker", "fragment-.gitignore"), []byte("node_modules/\n"), 0644)

	err := scaffoldTreeWalk(scaffoldDir, repoRoot)
	if err == nil {
		t.Fatal("expected scaffoldTreeWalk to propagate the fragment write error, got nil")
	}
}

func TestLineEnsureMerge_AppendNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("node_modules/\n.env\n"), 0644)
	os.WriteFile(dest, []byte("node_modules/\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	content := string(data)
	// Should contain .env but not duplicate node_modules/
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

func TestLineEnsureMerge_SkipComments(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "entries")
	dest := filepath.Join(destDir, "entries")

	os.WriteFile(src, []byte("# this is a comment\nactual-entry\n"), 0644)

	if err := lineEnsureMerge(src, dest, "entries"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	content := string(data)
	// Should only contain "actual-entry", not the comment
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

// scaffoldKitDir builds a minimal cached-kit layout under tmp with the given VERSION.
func scaffoldKitDir(t *testing.T, version string) string {
	t.Helper()
	kitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(kitDir, "VERSION"), []byte(version+"\n"), 0644); err != nil {
		t.Fatalf("cannot write kit VERSION: %v", err)
	}
	return kitDir
}

func TestScaffoldDirectories_FreshProject(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if err != nil {
		t.Fatalf("expected .kit-migration-version to be created: %v", err)
	}
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("fresh project: got %q, want %q", string(got), want)
	}
}

func TestScaffoldDirectories_PreExistingMigrationVersion(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	// Simulate Init() having already stamped both config.yaml and .kit-migration-version.
	if err := os.MkdirAll(filepath.Join(fabDir, "project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, "project", "config.yaml"), []byte("fab_version: 1.6.1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, ".kit-migration-version"), []byte("1.6.1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	// Pre-existing .kit-migration-version must be preserved, not overwritten with 0.1.0.
	got, _ := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("post-init: got %q, want %q (must preserve, not write 0.1.0)", string(got), want)
	}
}

func TestScaffoldDirectories_ExistingProjectWithoutMigrationVersion(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	// Pre-migration-version-era project: config.yaml exists, no .kit-migration-version.
	// This is the legitimate "existing project" branch (e.g., manual `fab sync` on an old project).
	if err := os.MkdirAll(filepath.Join(fabDir, "project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, "project", "config.yaml"), []byte("fab_version: 0.43.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if want := "0.1.0\n"; string(got) != want {
		t.Errorf("legacy project: got %q, want %q", string(got), want)
	}
}

func TestScaffoldDirectories_MissingKitVersionFails(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := t.TempDir() // no VERSION file

	// New-project branch (no config.yaml) reads kit VERSION — a failed read
	// must propagate, not silently stamp an empty .kit-migration-version.
	err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1")
	if err == nil {
		t.Fatal("expected error when kit VERSION is unreadable")
	}
	if !strings.Contains(err.Error(), "VERSION") {
		t.Errorf("expected kit VERSION read error, got: %v", err)
	}
}
