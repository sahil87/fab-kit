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

func TestCachedBinary(t *testing.T) {
	bin := CachedBinary("0.43.0")
	if !strings.HasSuffix(bin, "/0.43.0/fab-go") {
		t.Errorf("CachedBinary should end with /0.43.0/fab-go, got %s", bin)
	}
}

func TestCachedKitDir(t *testing.T) {
	kit := CachedKitDir("0.43.0")
	if !strings.HasSuffix(kit, "/0.43.0/kit") {
		t.Errorf("CachedKitDir should end with /0.43.0/kit, got %s", kit)
	}
}

func TestIsCached_Exists(t *testing.T) {
	// Create a fake cached binary
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	versionDir := filepath.Join(tmp, ".fab-kit", "versions", "0.43.0")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(versionDir, "fab-go")
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if !IsCached("0.43.0") {
		t.Error("expected IsCached to return true for existing binary")
	}
}

func TestIsCached_NotExists(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	if IsCached("0.99.0") {
		t.Error("expected IsCached to return false for non-existent version")
	}
}

func TestIsCached_NotExecutable(t *testing.T) {
	tmp := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmp)
	defer os.Setenv("HOME", origHome)

	versionDir := filepath.Join(tmp, ".fab-kit", "versions", "0.43.0")
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatal(err)
	}
	binPath := filepath.Join(versionDir, "fab-go")
	// Write without executable permission
	if err := os.WriteFile(binPath, []byte("#!/bin/sh\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if IsCached("0.43.0") {
		t.Error("expected IsCached to return false for non-executable binary")
	}
}
