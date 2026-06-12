package internal

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// chdir switches the process CWD for the duration of the test.
func chdir(t *testing.T, dir string) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chdir(orig) })
}

// stubRunSync replaces the runSync seam for the duration of the test.
func stubRunSync(t *testing.T, fn func(systemVersion, kitVersion string, shimOnly, projectOnly bool) error) {
	t.Helper()
	orig := runSync
	runSync = fn
	t.Cleanup(func() { runSync = orig })
}

// populateRemoteCache creates a resolvable cached version under home so
// EnsureCached never hits the network.
func populateRemoteCache(t *testing.T, home, version string) {
	t.Helper()
	dir := filepath.Join(home, ".fab-kit", "versions", version)
	if err := os.MkdirAll(filepath.Join(dir, "kit"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fab-go"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "kit", "VERSION"), []byte(version+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
}

// setupUpgradeRepo creates a repo dir with fab/project/config.yaml pinned to
// currentVersion, chdirs into it, and pre-populates the cache for targetVersion.
func setupUpgradeRepo(t *testing.T, currentVersion, targetVersion string) string {
	t.Helper()
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	populateRemoteCache(t, home, targetVersion)

	configDir := filepath.Join(repo, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf("fab_version: %q\n", currentVersion)
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte(config), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, repo)
	return repo
}

// captureStdout runs fn with os.Stdout redirected and returns what it printed.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	orig := os.Stdout
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = w
	defer func() { os.Stdout = orig }()
	fn()
	w.Close()
	data, _ := io.ReadAll(r)
	return string(data)
}

func TestUpgrade_SyncFailureExitsNonZeroWithoutStamping(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")

	var gotSystem, gotKit string
	stubRunSync(t, func(systemVersion, kitVersion string, shimOnly, projectOnly bool) error {
		gotSystem, gotKit = systemVersion, kitVersion
		return fmt.Errorf("deploy write failed")
	})

	var err error
	out := captureStdout(t, func() {
		err = Upgrade("1.5.0", "2.0.0")
	})

	if err == nil {
		t.Fatal("expected Upgrade to propagate the sync failure (non-zero exit)")
	}
	if !strings.Contains(err.Error(), "run 'fab sync' to repair") {
		t.Errorf("expected repair guidance in error, got: %v", err)
	}
	if strings.Contains(out, "Updated:") {
		t.Errorf("must not print 'Updated:' after a failed sync, output:\n%s", out)
	}

	// F22 threading: the embedded binary version and the kit version are
	// passed separately into Sync.
	if gotSystem != "1.5.0" || gotKit != "2.0.0" {
		t.Errorf("Sync called with (system=%q, kit=%q), want (1.5.0, 2.0.0)", gotSystem, gotKit)
	}

	// F18: fab_version must NOT be stamped on failure.
	v, err := readFabVersion(filepath.Join(repo, "fab", "project", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if v != "1.0.0" {
		t.Errorf("fab_version stamped to %q despite sync failure, want 1.0.0", v)
	}
}

func TestUpgrade_RerunAfterFailureRetries(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")

	// First attempt fails — stamp must not land.
	stubRunSync(t, func(string, string, bool, bool) error { return fmt.Errorf("boom") })
	if err := Upgrade("1.5.0", "2.0.0"); err == nil {
		t.Fatal("expected first upgrade attempt to fail")
	}

	// Re-run with a healthy sync: must retry (no "Already on the latest
	// version" short-circuit of the broken state) and stamp on success.
	synced := false
	stubRunSync(t, func(string, string, bool, bool) error { synced = true; return nil })
	if err := Upgrade("1.5.0", "2.0.0"); err != nil {
		t.Fatalf("re-run after failed upgrade should succeed, got: %v", err)
	}
	if !synced {
		t.Error("re-run short-circuited instead of retrying the sync")
	}

	v, _ := readFabVersion(filepath.Join(repo, "fab", "project", "config.yaml"))
	if v != "2.0.0" {
		t.Errorf("fab_version = %q after successful upgrade, want 2.0.0", v)
	}
}

func TestUpgrade_SuccessStampsAfterSync(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")

	// At sync time, the stamp must not have landed yet (stamp-after-success).
	stubRunSync(t, func(string, string, bool, bool) error {
		v, err := readFabVersion(filepath.Join(repo, "fab", "project", "config.yaml"))
		if err != nil {
			return err
		}
		if v != "1.0.0" {
			return fmt.Errorf("fab_version already stamped to %q before sync succeeded", v)
		}
		return nil
	})

	var err error
	out := captureStdout(t, func() {
		err = Upgrade("1.5.0", "2.0.0")
	})
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}
	if !strings.Contains(out, "Updated: 1.0.0 -> 2.0.0") {
		t.Errorf("expected success line, output:\n%s", out)
	}

	v, _ := readFabVersion(filepath.Join(repo, "fab", "project", "config.yaml"))
	if v != "2.0.0" {
		t.Errorf("fab_version = %q, want 2.0.0", v)
	}
}

func TestUpgrade_AlreadyOnTargetShortCircuits(t *testing.T) {
	setupUpgradeRepo(t, "2.0.0", "2.0.0")

	stubRunSync(t, func(string, string, bool, bool) error {
		t.Error("sync must not run when already on the target version")
		return nil
	})
	if err := Upgrade("1.5.0", "2.0.0"); err != nil {
		t.Fatalf("expected no-op success, got: %v", err)
	}
}

// requireGit skips the test when git is unavailable.
func requireGit(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
}
