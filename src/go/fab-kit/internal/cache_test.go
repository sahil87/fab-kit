package internal

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCacheDir(t *testing.T) {
	dir := CacheDir("0.43.0")
	if !strings.Contains(dir, ".fab-kit/versions/0.43.0") {
		t.Errorf("CacheDir should contain .fab-kit/versions/0.43.0, got %s", dir)
	}
}

func TestLocalCacheDir(t *testing.T) {
	dir := LocalCacheDir("0.43.0")
	if !strings.Contains(dir, ".fab-kit/local-versions/0.43.0") {
		t.Errorf("LocalCacheDir should contain .fab-kit/local-versions/0.43.0, got %s", dir)
	}
}

func TestResolveBinary_LocalPriority(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Create both local and remote binaries
	for _, dir := range []string{"local-versions", "versions"} {
		versionDir := filepath.Join(tmp, ".fab-kit", dir, "0.43.0")
		os.MkdirAll(versionDir, 0755)
		os.WriteFile(filepath.Join(versionDir, "fab-go"), []byte("#!/bin/sh\n"), 0755)
	}

	path, found := ResolveBinary("0.43.0")
	if !found {
		t.Fatal("expected ResolveBinary to find binary")
	}
	if !strings.Contains(path, "local-versions") {
		t.Errorf("expected local-versions to take priority, got %s", path)
	}
}

func TestResolveBinary_FallbackToRemote(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Only remote exists
	versionDir := filepath.Join(tmp, ".fab-kit", "versions", "0.43.0")
	os.MkdirAll(versionDir, 0755)
	os.WriteFile(filepath.Join(versionDir, "fab-go"), []byte("#!/bin/sh\n"), 0755)

	path, found := ResolveBinary("0.43.0")
	if !found {
		t.Fatal("expected ResolveBinary to find remote binary")
	}
	if !strings.Contains(path, "versions/0.43.0") {
		t.Errorf("expected versions/ path, got %s", path)
	}
}

func TestResolveBinary_NotFound(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	_, found := ResolveBinary("0.99.0")
	if found {
		t.Error("expected ResolveBinary to return false for non-existent version")
	}
}

func TestResolveBinary_LocalNotExecutable(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Local exists but not executable; remote is executable
	localDir := filepath.Join(tmp, ".fab-kit", "local-versions", "0.43.0")
	os.MkdirAll(localDir, 0755)
	os.WriteFile(filepath.Join(localDir, "fab-go"), []byte("#!/bin/sh\n"), 0644) // no exec bit

	remoteDir := filepath.Join(tmp, ".fab-kit", "versions", "0.43.0")
	os.MkdirAll(remoteDir, 0755)
	os.WriteFile(filepath.Join(remoteDir, "fab-go"), []byte("#!/bin/sh\n"), 0755)

	path, found := ResolveBinary("0.43.0")
	if !found {
		t.Fatal("expected ResolveBinary to find remote binary")
	}
	if !strings.Contains(path, "versions/0.43.0") {
		t.Errorf("expected fallback to remote, got %s", path)
	}
}

func TestCachedKitDir_LocalPriority(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// Create local kit dir
	localKit := filepath.Join(tmp, ".fab-kit", "local-versions", "0.43.0", "kit")
	os.MkdirAll(localKit, 0755)

	kit := CachedKitDir("0.43.0")
	if !strings.Contains(kit, "local-versions") {
		t.Errorf("expected local-versions kit, got %s", kit)
	}
}

func TestCachedKitDir_FallbackToRemote(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	// No local kit dir exists
	kit := CachedKitDir("0.43.0")
	if !strings.Contains(kit, "versions/0.43.0") {
		t.Errorf("expected remote kit path, got %s", kit)
	}
}

func TestResolveKitDir_EnvironmentOverride(t *testing.T) {
	parent := t.TempDir()
	kitDir := filepath.Join(parent, "worktree-kit")
	if err := os.Mkdir(kitDir, 0755); err != nil {
		t.Fatal(err)
	}
	t.Chdir(parent)
	t.Setenv(KitPathEnv, "worktree-kit")

	got, err := ResolveKitDir("0.43.0")
	if err != nil {
		t.Fatalf("ResolveKitDir: %v", err)
	}
	if got != kitDir {
		t.Errorf("ResolveKitDir = %q, want %q", got, kitDir)
	}
}

func TestResolveKitDir_InvalidEnvironmentOverride(t *testing.T) {
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
			t.Setenv(KitPathEnv, tc.path(t))
			_, err := ResolveKitDir("0.43.0")
			if err == nil {
				t.Fatal("expected invalid override to fail")
			}
			if !strings.Contains(err.Error(), KitPathEnv) || !strings.Contains(err.Error(), "not a directory") {
				t.Errorf("error = %q, want %s not-a-directory error", err, KitPathEnv)
			}
		})
	}
}

func TestResolveKitDir_UnsetPreservesCacheResolution(t *testing.T) {
	t.Setenv(KitPathEnv, "")
	t.Setenv("HOME", t.TempDir())

	got, err := ResolveKitDir("0.43.0")
	if err != nil {
		t.Fatalf("ResolveKitDir: %v", err)
	}
	if want := CachedKitDir("0.43.0"); got != want {
		t.Errorf("ResolveKitDir = %q, want cached path %q", got, want)
	}
}
