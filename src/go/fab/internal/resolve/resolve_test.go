package resolve

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupFabRoot creates a minimal fab/ structure in a temp dir and returns the fabRoot path.
func setupFabRoot(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(filepath.Join(fabRoot, "changes"), 0755)
	return fabRoot
}

// createChange creates a change directory with a .status.yaml sentinel file.
func createChange(t *testing.T, fabRoot, folderName string) string {
	t.Helper()
	changeDir := filepath.Join(fabRoot, "changes", folderName)
	os.MkdirAll(changeDir, 0755)
	os.WriteFile(filepath.Join(changeDir, ".status.yaml"), []byte("id: test\n"), 0644)
	return changeDir
}

func TestExtractID(t *testing.T) {
	tests := []struct {
		folder string
		want   string
	}{
		{"260310-abcd-my-change", "abcd"},
		{"260310-ef12-slug", "ef12"},
		{"noprefix", ""},
		{"260310-xy", "xy"},
	}
	for _, tt := range tests {
		t.Run(tt.folder, func(t *testing.T) {
			got := ExtractID(tt.folder)
			if got != tt.want {
				t.Errorf("ExtractID(%q) = %q, want %q", tt.folder, got, tt.want)
			}
		})
	}
}

func TestExtractFolderFromSymlink(t *testing.T) {
	tests := []struct {
		target string
		want   string
	}{
		{"fab/changes/260310-abcd-my-change/.status.yaml", "260310-abcd-my-change"},
		{"fab/changes/260310-ef12-other/.status.yaml", "260310-ef12-other"},
		{"fab/changes//.status.yaml", ""},      // empty name
		{"wrong/prefix/name/.status.yaml", ""}, // wrong prefix
		{"completely-unrelated-path", ""},       // no matching structure
	}
	for _, tt := range tests {
		t.Run(tt.target, func(t *testing.T) {
			got := ExtractFolderFromSymlink(tt.target)
			if got != tt.want {
				t.Errorf("ExtractFolderFromSymlink(%q) = %q, want %q", tt.target, got, tt.want)
			}
		})
	}
}

func TestToFolder_ExactMatch(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	got, err := ToFolder(fabRoot, "260310-abcd-my-change")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "260310-abcd-my-change" {
		t.Errorf("got %q, want %q", got, "260310-abcd-my-change")
	}
}

func TestToFolder_4CharID(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	got, err := ToFolder(fabRoot, "abcd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "260310-abcd-my-change" {
		t.Errorf("got %q, want %q", got, "260310-abcd-my-change")
	}
}

func TestToFolder_Substring(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	got, err := ToFolder(fabRoot, "my-change")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "260310-abcd-my-change" {
		t.Errorf("got %q, want %q", got, "260310-abcd-my-change")
	}
}

func TestToFolder_Ambiguous(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")
	createChange(t, fabRoot, "260310-efgh-my-other-change")

	_, err := ToFolder(fabRoot, "my")
	if err == nil {
		t.Fatal("expected error for ambiguous match")
	}
	if !strings.Contains(err.Error(), "Multiple changes match") {
		t.Errorf("expected 'Multiple changes match' error, got: %v", err)
	}
}

func TestToFolder_NoMatch(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	_, err := ToFolder(fabRoot, "nonexistent")
	if err == nil {
		t.Fatal("expected error for no match")
	}
	if !strings.Contains(err.Error(), "No change matches") {
		t.Errorf("expected 'No change matches' error, got: %v", err)
	}
}

func TestToFolder_Symlink(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	// Create .fab-status.yaml symlink at repo root (parent of fab/)
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	os.Symlink("fab/changes/260310-abcd-my-change/.status.yaml", symlinkPath)

	got, err := ToFolder(fabRoot, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "260310-abcd-my-change" {
		t.Errorf("got %q, want %q", got, "260310-abcd-my-change")
	}
}

func TestToDir(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	got, err := ToDir(fabRoot, "abcd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "fab/changes/260310-abcd-my-change/"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToStatus(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	got, err := ToStatus(fabRoot, "abcd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "fab/changes/260310-abcd-my-change/.status.yaml"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToAbsDir(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	got, err := ToAbsDir(fabRoot, "abcd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(fabRoot, "changes", "260310-abcd-my-change")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestToAbsStatus(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")

	got, err := ToAbsStatus(fabRoot, "abcd")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(fabRoot, "changes", "260310-abcd-my-change", ".status.yaml")
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFabRoot(t *testing.T) {
	// FabRoot walks up from cwd to find a fab/ directory.
	// Test creates a temp dir with fab/ and a nested subdir, then
	// verifies FabRoot resolves correctly from the nested location.
	dir, _ := filepath.EvalSymlinks(t.TempDir())
	fabDir := filepath.Join(dir, "fab")
	os.MkdirAll(fabDir, 0755)

	// Create a nested subdirectory
	nested := filepath.Join(dir, "a", "b", "c")
	os.MkdirAll(nested, 0755)

	// Save and restore cwd
	origDir, _ := os.Getwd()
	defer os.Chdir(origDir)

	os.Chdir(nested)
	got, err := FabRoot()
	if err != nil {
		t.Fatalf("FabRoot() from nested dir: %v", err)
	}
	if got != fabDir {
		t.Errorf("FabRoot() = %q, want %q", got, fabDir)
	}
}

func TestToFolder_NoChangesDir(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0755)
	// No changes/ directory

	_, err := ToFolder(fabRoot, "anything")
	if err == nil {
		t.Fatal("expected error when fab/changes/ does not exist")
	}
}
