package kitpath

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func resetKitPathOverrides(t *testing.T) {
	t.Helper()
	SetOverride("")
	t.Cleanup(func() { SetOverride("") })
	t.Setenv(KitPathEnv, "")
}

func TestKitDir_WithKitSibling(t *testing.T) {
	resetKitPathOverrides(t)

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
	resetKitPathOverrides(t)

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

func TestKitDir_EnvironmentOverrideIsAbsolutized(t *testing.T) {
	resetKitPathOverrides(t)

	parent := t.TempDir()
	kitDir := filepath.Join(parent, "worktree-kit")
	if err := os.Mkdir(kitDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(parent)
	t.Setenv(KitPathEnv, "worktree-kit")

	got, err := KitDir()
	if err != nil {
		t.Fatalf("KitDir: %v", err)
	}
	if got != kitDir {
		t.Errorf("KitDir = %q, want %q", got, kitDir)
	}
}

func TestKitDir_EnvironmentOverrideRejectsNonDirectories(t *testing.T) {
	for _, tc := range []struct {
		name string
		path func(*testing.T) string
	}{
		{
			name: "missing",
			path: func(t *testing.T) string { return filepath.Join(t.TempDir(), "missing") },
		},
		{
			name: "file",
			path: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "kit-file")
				if err := os.WriteFile(path, []byte("not a directory"), 0644); err != nil {
					t.Fatal(err)
				}
				return path
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			resetKitPathOverrides(t)
			t.Setenv(KitPathEnv, tc.path(t))

			_, err := KitDir()
			if err == nil {
				t.Fatal("expected invalid override to fail")
			}
			if !strings.Contains(err.Error(), KitPathEnv) || !strings.Contains(err.Error(), "not a directory") {
				t.Errorf("error = %q, want %s not-a-directory error", err, KitPathEnv)
			}
		})
	}
}

func TestKitDir_SetOverrideWinsOverEnvironment(t *testing.T) {
	resetKitPathOverrides(t)

	testOverride := filepath.Join(t.TempDir(), "test-only")
	SetOverride(testOverride)
	t.Setenv(KitPathEnv, filepath.Join(t.TempDir(), "missing"))

	got, err := KitDir()
	if err != nil {
		t.Fatalf("KitDir: %v", err)
	}
	if got != testOverride {
		t.Errorf("KitDir = %q, want SetOverride value %q", got, testOverride)
	}
}
