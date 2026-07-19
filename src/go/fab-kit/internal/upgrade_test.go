package internal

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
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

// setupUpgradeRepo creates a repo dir with fab/.fab-version pinned to
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
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// The starting version pin is the plain-text sibling (the sole source since
	// 260719-kq7v closed the config.yaml fallback). A successful Upgrade overwrites
	// it; a failed one leaves it, which the mid-flow/failure-path assertions read.
	if err := os.WriteFile(filepath.Join(repo, "fab", ".fab-version"), []byte(currentVersion+"\n"), 0644); err != nil {
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
		err = Upgrade("1.5.0", "2.0.0", false)
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

	// F18: the version must NOT be re-stamped on failure — the fixture's starting
	// pin (1.0.0) is left intact in fab/.fab-version.
	v, err := readFabVersion(repo)
	if err != nil {
		t.Fatal(err)
	}
	if v != "1.0.0" {
		t.Errorf("version stamped to %q despite sync failure, want 1.0.0", v)
	}
}

func TestUpgrade_RerunAfterFailureRetries(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")

	// First attempt fails — stamp must not land.
	stubRunSync(t, func(string, string, bool, bool) error { return fmt.Errorf("boom") })
	if err := Upgrade("1.5.0", "2.0.0", false); err == nil {
		t.Fatal("expected first upgrade attempt to fail")
	}

	// Re-run with a healthy sync: must retry (no "Already on the latest
	// version" short-circuit of the broken state) and stamp on success.
	synced := false
	stubRunSync(t, func(string, string, bool, bool) error { synced = true; return nil })
	if err := Upgrade("1.5.0", "2.0.0", false); err != nil {
		t.Fatalf("re-run after failed upgrade should succeed, got: %v", err)
	}
	if !synced {
		t.Error("re-run short-circuited instead of retrying the sync")
	}

	v, _ := readFabVersion(repo)
	if v != "2.0.0" {
		t.Errorf("fab_version = %q after successful upgrade, want 2.0.0", v)
	}
}

func TestUpgrade_SuccessStampsAfterSync(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")

	// At sync time, the target stamp must not have landed yet (stamp-after-success):
	// fab/.fab-version still reads the fixture's starting pin (1.0.0), not 2.0.0.
	stubRunSync(t, func(string, string, bool, bool) error {
		v, err := readFabVersion(repo)
		if err != nil {
			return err
		}
		if v != "1.0.0" {
			return fmt.Errorf("version already stamped to %q before sync succeeded", v)
		}
		return nil
	})

	var err error
	out := captureStdout(t, func() {
		err = Upgrade("1.5.0", "2.0.0", false)
	})
	if err != nil {
		t.Fatalf("Upgrade failed: %v", err)
	}
	if !strings.Contains(out, "Updated: 1.0.0 -> 2.0.0") {
		t.Errorf("expected success line, output:\n%s", out)
	}

	v, _ := readFabVersion(repo)
	if v != "2.0.0" {
		t.Errorf("fab_version = %q, want 2.0.0", v)
	}
}

// TestUpgrade_ConfigUpgradeFailsOpen: when the pinned fab-go's `fab config upgrade`
// exits non-zero (a binary that predates the subcommand), the upgrade must still
// succeed and stamp — an upgrade may never break on the config step (decision 4).
func TestUpgrade_ConfigUpgradeFailsOpen(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")
	// Replace the cached fab-go with one that ALWAYS exits non-zero (mimics an
	// unknown `config upgrade` subcommand on a predates-subcommand binary).
	fabGo := filepath.Join(os.Getenv("HOME"), ".fab-kit", "versions", "2.0.0", "fab-go")
	if err := os.WriteFile(fabGo, []byte("#!/bin/sh\necho 'unknown command \"upgrade\"' >&2\nexit 1\n"), 0755); err != nil {
		t.Fatal(err)
	}
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	var err error
	out := captureStdout(t, func() { err = Upgrade("1.5.0", "2.0.0", false) })
	if err != nil {
		t.Fatalf("Upgrade must fail open on a config-upgrade failure, got: %v", err)
	}
	if !strings.Contains(out, "could not auto-run") {
		t.Errorf("expected a fail-open reminder for the config upgrade step, output:\n%s", out)
	}
	// The version stamp still landed (the config step failing does not roll it back).
	v, _ := readFabVersion(repo)
	if v != "2.0.0" {
		t.Errorf("version = %q, want 2.0.0 (stamp lands even when config upgrade fails open)", v)
	}
}

func TestUpgrade_AlreadyOnTargetShortCircuits(t *testing.T) {
	setupUpgradeRepo(t, "2.0.0", "2.0.0")

	stubRunSync(t, func(string, string, bool, bool) error {
		t.Error("sync must not run when already on the target version")
		return nil
	})
	if err := Upgrade("1.5.0", "2.0.0", false); err != nil {
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

// --- Remaining Upgrade branches (260612-tb6f, F45) ---

func TestUpgrade_MissingKitVersionFileFails(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)

	// Cache resolvable (fab-go present) but the kit carries no VERSION file.
	dir := filepath.Join(home, ".fab-kit", "versions", "2.0.0")
	if err := os.MkdirAll(filepath.Join(dir, "kit"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "fab-go"), []byte("#!/bin/sh\n"), 0755); err != nil {
		t.Fatal(err)
	}

	configDir := filepath.Join(repo, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, "fab", ".fab-version"), []byte("1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, repo)

	stubRunSync(t, func(string, string, bool, bool) error {
		t.Error("sync must not run when the cached kit has no VERSION file")
		return nil
	})

	err := Upgrade("1.5.0", "2.0.0", false)
	if err == nil {
		t.Fatal("expected Upgrade to fail on a VERSION-less cached kit")
	}
	if !strings.Contains(err.Error(), "missing VERSION file") {
		t.Errorf("expected VERSION-file error, got: %v", err)
	}
}

// migrationsDirFor returns the cached kit migrations dir for a version under
// the current (test) HOME.
func migrationsDirFor(t *testing.T, version string) string {
	t.Helper()
	return filepath.Join(os.Getenv("HOME"), ".fab-kit", "versions", version, "kit", "migrations")
}

func TestUpgrade_MigrationReminderWhenApplicable(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	// Local migration version 1.0.0 + an applicable 1.0.0-to-2.0.0 migration.
	if err := os.WriteFile(filepath.Join(repo, "fab", ".kit-migration-version"), []byte("1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	migDir := migrationsDirFor(t, "2.0.0")
	if err := os.MkdirAll(migDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "1.0.0-to-2.0.0.md"), []byte("# migration\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var err error
	out := captureStdout(t, func() { err = Upgrade("1.5.0", "2.0.0", false) })
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !strings.Contains(out, "/fab-setup migrations") {
		t.Errorf("expected migration reminder, output:\n%s", out)
	}

	// The skill owns the stamp — upgrade must NOT have stamped it.
	got, _ := os.ReadFile(filepath.Join(repo, "fab", ".kit-migration-version"))
	if string(got) != "1.0.0\n" {
		t.Errorf(".kit-migration-version = %q, want 1.0.0 (unstamped — migrations apply)", got)
	}
}

func TestUpgrade_SilentStampWhenNoMigrationsApply(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	if err := os.WriteFile(filepath.Join(repo, "fab", ".kit-migration-version"), []byte("1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Migrations dir exists but nothing applies.
	if err := os.MkdirAll(migrationsDirFor(t, "2.0.0"), 0755); err != nil {
		t.Fatal(err)
	}

	var err error
	out := captureStdout(t, func() { err = Upgrade("1.5.0", "2.0.0", false) })
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if strings.Contains(out, "/fab-setup migrations") {
		t.Errorf("no reminder expected when nothing applies, output:\n%s", out)
	}

	// Silent self-stamp to the target stops version drift.
	got, _ := os.ReadFile(filepath.Join(repo, "fab", ".kit-migration-version"))
	if string(got) != "2.0.0\n" {
		t.Errorf(".kit-migration-version = %q, want 2.0.0 (silently stamped)", got)
	}
}

func TestUpgrade_DiscoveryFailureWarnsWithoutStamp(t *testing.T) {
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	// .kit-migration-version present but the cached kit has NO migrations dir
	// — discovery errors; the upgrade itself must still succeed, and the
	// stamp must be skipped (cannot confirm nothing applies).
	if err := os.WriteFile(filepath.Join(repo, "fab", ".kit-migration-version"), []byte("1.0.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	var err error
	captureStdout(t, func() { err = Upgrade("1.5.0", "2.0.0", false) })
	if err != nil {
		t.Fatalf("Upgrade must not fail on discovery failure: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(repo, "fab", ".kit-migration-version"))
	if string(got) != "1.0.0\n" {
		t.Errorf(".kit-migration-version = %q, want 1.0.0 (no stamp on discovery failure)", got)
	}
}

func TestUpgrade_NoFabVersionInstallPath(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	populateRemoteCache(t, home, "2.0.0")

	// config.yaml exists but has no fab_version — the recovery branch
	// proceeds and prints "Installed:" instead of "Updated:".
	configDir := filepath.Join(repo, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("project:\n    name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	chdir(t, repo)

	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	var err error
	out := captureStdout(t, func() { err = Upgrade("1.5.0", "2.0.0", false) })
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !strings.Contains(out, "Installed: 2.0.0") {
		t.Errorf("expected 'Installed: 2.0.0' line, output:\n%s", out)
	}

	// The version is stamped into fab/.fab-version (config.yaml is no longer
	// version-stamped), so readFabVersion resolves it from there.
	v, err := readFabVersion(repo)
	if err != nil {
		t.Fatalf("readFabVersion: %v", err)
	}
	if v != "2.0.0" {
		t.Errorf("resolved fab version = %q, want 2.0.0", v)
	}
}

// TestUpgrade_NoFabVersionFromSubdirectory: with the config.yaml fallback closed
// (260719-kq7v), an unmigrated repo (no fab/.fab-version) makes ResolveConfig
// error, so Upgrade hits its recovery branch. Run from a SUBDIRECTORY, that
// branch must still walk up to locate fab/project/config.yaml and proceed —
// rather than falsely reporting "not in a fab-managed repo" (Copilot #506).
func TestUpgrade_NoFabVersionFromSubdirectory(t *testing.T) {
	repo := t.TempDir()
	home := t.TempDir()
	t.Setenv("HOME", home)
	populateRemoteCache(t, home, "2.0.0")

	// config.yaml exists but there is no fab/.fab-version pin.
	configDir := filepath.Join(repo, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), []byte("project:\n    name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// Run from a nested subdirectory, not the repo root.
	subDir := filepath.Join(repo, "src", "go", "fab-kit")
	if err := os.MkdirAll(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	chdir(t, subDir)

	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	var err error
	out := captureStdout(t, func() { err = Upgrade("1.5.0", "2.0.0", false) })
	if err != nil {
		t.Fatalf("Upgrade from subdirectory: %v", err)
	}
	if !strings.Contains(out, "Installed: 2.0.0") {
		t.Errorf("expected 'Installed: 2.0.0' line, output:\n%s", out)
	}

	// The pin is stamped at the repo root (walked-up), not under the CWD.
	v, err := readFabVersion(repo)
	if err != nil {
		t.Fatalf("readFabVersion: %v", err)
	}
	if v != "2.0.0" {
		t.Errorf("resolved fab version = %q, want 2.0.0", v)
	}
	if _, statErr := os.Stat(filepath.Join(subDir, "fab", ".fab-version")); statErr == nil {
		t.Errorf("pin must be stamped at repo root, not under the CWD subdirectory")
	}
}

// --- Target resolution precedence (260613-1hmj, offline-first default) ---

// stubGitHubAPINever points githubAPIURL at a server that fails the test if the
// releases/latest endpoint is hit, proving an offline resolution path.
func stubGitHubAPINever(t *testing.T) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("GitHub API must not be called for this resolution path, got request: %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	orig := githubAPIURL
	githubAPIURL = srv.URL
	t.Cleanup(func() {
		githubAPIURL = orig
		srv.Close()
	})
}

// stubGitHubAPILatest points githubAPIURL at a server that returns the given tag
// from releases/latest and records whether it was hit.
func stubGitHubAPILatest(t *testing.T, tag string, hit *bool) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/repos/"+githubRepo+"/releases/latest" {
			http.NotFound(w, r)
			return
		}
		if hit != nil {
			*hit = true
		}
		io.WriteString(w, fmt.Sprintf(`{"tag_name": %q}`, tag))
	}))
	orig := githubAPIURL
	githubAPIURL = srv.URL
	t.Cleanup(func() {
		githubAPIURL = orig
		srv.Close()
	})
}

func TestUpgrade_DefaultResolvesToSystemVersionNoNetwork(t *testing.T) {
	// systemVersion is a real release tag → resolve offline to it, never
	// touching the GitHub API. Cache the systemVersion (2.3.1) as the target so
	// EnsureCached is satisfied without networking.
	repo := setupUpgradeRepo(t, "1.0.0", "2.3.1")
	stubGitHubAPINever(t)
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	var err error
	out := captureStdout(t, func() { err = Upgrade("2.3.1", "", false) })
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if strings.Contains(out, "Resolving latest version...") {
		t.Errorf("default path must not resolve via the network, output:\n%s", out)
	}

	// The repo must have been upgraded to the systemVersion.
	v, _ := readFabVersion(repo)
	if v != "2.3.1" {
		t.Errorf("fab_version = %q, want 2.3.1 (resolved to systemVersion)", v)
	}
}

func TestUpgrade_LatestFlagCallsAPI(t *testing.T) {
	// --latest resolves via LatestVersion(). The cache is populated for the tag
	// the stubbed API returns (2.0.0) so EnsureCached is satisfied.
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")
	hit := false
	stubGitHubAPILatest(t, "v2.0.0", &hit)
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	var err error
	captureStdout(t, func() { err = Upgrade("2.3.1", "", true) })
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !hit {
		t.Error("--latest must resolve via the GitHub API, but it was not called")
	}

	v, _ := readFabVersion(repo)
	if v != "2.0.0" {
		t.Errorf("fab_version = %q, want 2.0.0 (resolved via --latest)", v)
	}
}

func TestUpgrade_DevBinaryFallsBackToAPI(t *testing.T) {
	// A "dev" systemVersion has no real release tag → fall back to the API.
	repo := setupUpgradeRepo(t, "1.0.0", "2.0.0")
	hit := false
	stubGitHubAPILatest(t, "v2.0.0", &hit)
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	var err error
	captureStdout(t, func() { err = Upgrade("dev", "", false) })
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if !hit {
		t.Error("a dev binary must fall back to the GitHub API, but it was not called")
	}

	v, _ := readFabVersion(repo)
	if v != "2.0.0" {
		t.Errorf("fab_version = %q, want 2.0.0 (dev fallback to API)", v)
	}
}

func TestUpgrade_ExplicitArgIgnoresLatest(t *testing.T) {
	// An explicit arg wins; --latest is ignored and the API is never called.
	repo := setupUpgradeRepo(t, "1.0.0", "2.2.0")
	stubGitHubAPINever(t)
	stubRunSync(t, func(string, string, bool, bool) error { return nil })

	var err error
	captureStdout(t, func() { err = Upgrade("2.3.1", "2.2.0", true) })
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}

	v, _ := readFabVersion(repo)
	if v != "2.2.0" {
		t.Errorf("fab_version = %q, want 2.2.0 (explicit arg wins over --latest)", v)
	}
}
