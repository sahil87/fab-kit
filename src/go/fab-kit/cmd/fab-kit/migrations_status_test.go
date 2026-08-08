package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

func setupMigrationsStatusRepo(t *testing.T) string {
	t.Helper()
	t.Setenv(internal.KitPathEnv, "")

	repo := t.TempDir()
	if err := os.MkdirAll(filepath.Join(repo, "fab", "project"), 0755); err != nil {
		t.Fatal(err)
	}
	for path, content := range map[string]string{
		filepath.Join(repo, "fab", "project", "config.yaml"): "project:\n  name: test\n",
		filepath.Join(repo, "fab", ".fab-version"):           "1.0.0\n",
		filepath.Join(repo, "fab", ".kit-migration-version"): "1.0.0\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}
	t.Chdir(repo)
	return repo
}

func TestMigrationsStatus_UsesEnvironmentOverride(t *testing.T) {
	setupMigrationsStatusRepo(t)
	overrideKit := t.TempDir()
	if err := os.MkdirAll(filepath.Join(overrideKit, "migrations"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(overrideKit, "VERSION"), []byte("1.1.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv(internal.KitPathEnv, overrideKit)

	cmd := migrationsStatusCmd()
	var out bytes.Buffer
	cmd.SetOut(&out)
	if err := runMigrationsStatus(cmd, false); err != nil {
		t.Fatalf("runMigrationsStatus: %v", err)
	}
	if !strings.Contains(out.String(), "Engine version: 1.1.0") {
		t.Errorf("migrations-status did not read override VERSION:\n%s", out.String())
	}
}

func TestMigrationsStatus_InvalidEnvironmentOverrideFailsLoudly(t *testing.T) {
	setupMigrationsStatusRepo(t)
	t.Setenv(internal.KitPathEnv, filepath.Join(t.TempDir(), "missing"))

	err := runMigrationsStatus(migrationsStatusCmd(), false)
	if err == nil {
		t.Fatal("expected invalid override to fail")
	}
	if !strings.Contains(err.Error(), internal.KitPathEnv) || !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("error = %q, want loud %s not-a-directory error", err, internal.KitPathEnv)
	}
}

func TestMigrationsStatus_MissingVersionHintFollowsKitSource(t *testing.T) {
	t.Run("override", func(t *testing.T) {
		setupMigrationsStatusRepo(t)
		t.Setenv(internal.KitPathEnv, t.TempDir()) // valid dir, no VERSION

		err := runMigrationsStatus(migrationsStatusCmd(), false)
		if err == nil {
			t.Fatal("expected missing VERSION to fail")
		}
		if !strings.Contains(err.Error(), internal.KitPathEnv) {
			t.Errorf("error = %q, want the hint to name %s", err, internal.KitPathEnv)
		}
		if strings.Contains(err.Error(), "kit cache") {
			t.Errorf("error = %q, want no cache hint under the override", err)
		}
	})

	t.Run("cache", func(t *testing.T) {
		setupMigrationsStatusRepo(t)
		t.Setenv("HOME", t.TempDir()) // empty cache, no VERSION

		err := runMigrationsStatus(migrationsStatusCmd(), false)
		if err == nil {
			t.Fatal("expected missing VERSION to fail")
		}
		if !strings.Contains(err.Error(), "kit cache") {
			t.Errorf("error = %q, want the cache hint when no override is set", err)
		}
	})
}
