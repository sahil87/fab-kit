package resolve

import (
	"errors"
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
		{"completely-unrelated-path", ""},      // no matching structure
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
	// jznd (d): the documented message is preserved verbatim AND classified as
	// ErrAmbiguous so archive soft-skip can branch on it.
	if !errors.Is(err, ErrAmbiguous) {
		t.Errorf("ambiguous match must be errors.Is(ErrAmbiguous), got: %v", err)
	}
	if errors.Is(err, ErrNotFound) {
		t.Error("ambiguous match must NOT be errors.Is(ErrNotFound)")
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
	// jznd (d): classified as ErrNotFound (the "maybe already archived"
	// soft-skip path), not ErrAmbiguous.
	if !errors.Is(err, ErrNotFound) {
		t.Errorf("no match must be errors.Is(ErrNotFound), got: %v", err)
	}
	if errors.Is(err, ErrAmbiguous) {
		t.Error("no match must NOT be errors.Is(ErrAmbiguous)")
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

// assertNoActiveChangeGuidance verifies the "No active change." error carries
// the recovery guidance promised by _preamble.md (preflight stderr contains
// the specific error and suggested fix).
func assertNoActiveChangeGuidance(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected 'No active change' error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "No active change.") {
		t.Errorf("error = %q, want it to contain 'No active change.'", msg)
	}
	if !strings.Contains(msg, "/fab-new") {
		t.Errorf("error = %q, want it to suggest /fab-new", msg)
	}
	if !strings.Contains(msg, "/fab-switch") {
		t.Errorf("error = %q, want it to suggest /fab-switch", msg)
	}
}

func TestToFolder_NoActiveChange_ZeroCandidates(t *testing.T) {
	// changes/ exists but holds no change with a .status.yaml
	fabRoot := setupFabRoot(t)

	_, err := ToFolder(fabRoot, "")
	assertNoActiveChangeGuidance(t, err)
}

func TestToFolder_NoActiveChange_MissingChangesDir(t *testing.T) {
	dir := t.TempDir()
	fabRoot := filepath.Join(dir, "fab")
	os.MkdirAll(fabRoot, 0755)
	// No changes/ directory, no .fab-status.yaml symlink

	_, err := ToFolder(fabRoot, "")
	assertNoActiveChangeGuidance(t, err)
}

func TestToFolder_NoActiveChange_MultipleCandidates(t *testing.T) {
	// The multi-candidate variant keeps its own /fab-switch hint.
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")
	createChange(t, fabRoot, "260310-ef12-other")

	_, err := ToFolder(fabRoot, "")
	if err == nil {
		t.Fatal("expected error for multiple candidates without a symlink")
	}
	if !strings.Contains(err.Error(), "multiple changes exist") || !strings.Contains(err.Error(), "/fab-switch") {
		t.Errorf("error = %q, want the multiple-changes /fab-switch hint", err.Error())
	}
}

// --- Dangling pointer target validation (mz4q F08): the symlink is trusted
// only when its target .status.yaml still exists; otherwise resolution falls
// through to the no-active-change / single-change logic, leaving the link in
// place (resolve is a pure query). ---

func danglingSymlink(t *testing.T, fabRoot string) {
	t.Helper()
	repoRoot := filepath.Dir(fabRoot)
	symlinkPath := filepath.Join(repoRoot, ".fab-status.yaml")
	if err := os.Symlink("fab/changes/260301-gone-archived-change/.status.yaml", symlinkPath); err != nil {
		t.Fatal(err)
	}
}

func TestToFolder_DanglingSymlinkMultipleCandidates(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")
	createChange(t, fabRoot, "260311-ef12-other-change")
	danglingSymlink(t, fabRoot)

	_, err := ToFolder(fabRoot, "")
	if err == nil {
		t.Fatal("expected error: a dangling pointer must not resolve to the stale folder")
	}
	if !strings.Contains(err.Error(), "multiple changes exist") {
		t.Errorf("expected multiple-changes guidance, got: %v", err)
	}

	// The stale link is left in place — resolve has no side effects.
	repoRoot := filepath.Dir(fabRoot)
	if _, err := os.Lstat(filepath.Join(repoRoot, ".fab-status.yaml")); err != nil {
		t.Errorf("expected stale symlink to be left in place: %v", err)
	}
}

func TestToFolder_DanglingSymlinkSingleCandidate(t *testing.T) {
	fabRoot := setupFabRoot(t)
	createChange(t, fabRoot, "260310-abcd-my-change")
	danglingSymlink(t, fabRoot)

	got, err := ToFolder(fabRoot, "")
	if err != nil {
		t.Fatalf("expected single-change fallback, got error: %v", err)
	}
	if got != "260310-abcd-my-change" {
		t.Errorf("got %q, want single-change fallback to 260310-abcd-my-change", got)
	}
}

func TestToFolder_DanglingSymlinkZeroCandidates(t *testing.T) {
	fabRoot := setupFabRoot(t)
	danglingSymlink(t, fabRoot)

	_, err := ToFolder(fabRoot, "")
	if err == nil {
		t.Fatal("expected error with zero candidates")
	}
	if !strings.Contains(err.Error(), "No active change. Run /fab-new") {
		t.Errorf("expected actionable no-active-change guidance, got: %v", err)
	}
}

func TestToFolder_SymlinkTargetMissingStatusFile(t *testing.T) {
	fabRoot := setupFabRoot(t)
	// Folder exists but its .status.yaml was deleted — still a stale pointer.
	os.MkdirAll(filepath.Join(fabRoot, "changes", "260310-abcd-my-change"), 0755)
	createChange(t, fabRoot, "260311-ef12-other-change")

	repoRoot := filepath.Dir(fabRoot)
	os.Symlink("fab/changes/260310-abcd-my-change/.status.yaml", filepath.Join(repoRoot, ".fab-status.yaml"))

	got, err := ToFolder(fabRoot, "")
	if err != nil {
		t.Fatalf("expected single-change fallback, got error: %v", err)
	}
	if got != "260311-ef12-other-change" {
		t.Errorf("got %q, want fallback to the one valid change", got)
	}
}

func TestToFolder_UnreadableSymlinkTargetClassified(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("running as root — permission bits are not enforced")
	}
	fabRoot := setupFabRoot(t)
	changeDir := createChange(t, fabRoot, "260310-abcd-my-change")
	createChange(t, fabRoot, "260311-ef12-other-change")

	repoRoot := filepath.Dir(fabRoot)
	if err := os.Symlink("fab/changes/260310-abcd-my-change/.status.yaml", filepath.Join(repoRoot, ".fab-status.yaml")); err != nil {
		t.Fatal(err)
	}

	// A non-traversable change dir makes the target stat fail with EACCES —
	// not absence. That must surface with its cause, not fall through to the
	// misleading no-active-change / multiple-changes guidance.
	if err := os.Chmod(changeDir, 0o000); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(changeDir, 0o755) })

	_, err := ToFolder(fabRoot, "")
	if err == nil {
		t.Fatal("expected error for unreadable pointer target")
	}
	if strings.Contains(err.Error(), "No active change") {
		t.Errorf("permission failure must not masquerade as no-active-change: %v", err)
	}
	if !strings.Contains(err.Error(), "stat active change target") || !strings.Contains(err.Error(), "permission denied") {
		t.Errorf("expected cause-bearing stat error, got: %v", err)
	}
}
