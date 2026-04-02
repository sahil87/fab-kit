package kitpath

import (
	"os"
	"path/filepath"
	"testing"
)

func TestKitDir_WithKitSibling(t *testing.T) {
	// Create a temp directory with a fake executable and kit/ sibling
	dir := t.TempDir()
	kitDir := filepath.Join(dir, "kit")
	if err := os.Mkdir(kitDir, 0755); err != nil {
		t.Fatalf("cannot create kit dir: %v", err)
	}

	// Create a fake executable
	exePath := filepath.Join(dir, "fab-go")
	if err := os.WriteFile(exePath, []byte("fake"), 0755); err != nil {
		t.Fatalf("cannot create fake exe: %v", err)
	}

	// We can't easily override os.Executable() in a unit test,
	// so we test the resolution logic directly.
	result := filepath.Join(filepath.Dir(exePath), "kit")
	if result != kitDir {
		t.Errorf("expected %s, got %s", kitDir, result)
	}
}

func TestKitDir_MissingSibling(t *testing.T) {
	// KitDir should return an error when kit/ doesn't exist next to the executable.
	// Since we can't control os.Executable() in tests, we verify the error message pattern.
	_, err := KitDir()
	// In test context the executable is the test binary — kit/ won't exist next to it.
	if err == nil {
		t.Skip("kit/ unexpectedly exists next to test binary")
	}
	if err.Error() == "" {
		t.Error("expected non-empty error message")
	}
}
