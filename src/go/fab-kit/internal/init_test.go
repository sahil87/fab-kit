package internal

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetFabVersion_NewFile(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "fab", "project", "config.yaml")

	if err := setFabVersion(path, "0.43.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("cannot read file: %v", err)
	}

	content := string(data)
	if content == "" {
		t.Fatal("file is empty")
	}

	// Verify the version can be read back
	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("cannot read back fab_version: %v", err)
	}
	if v != "0.43.0" {
		t.Errorf("expected 0.43.0, got %s", v)
	}
}

func TestSetFabVersion_ExistingFile(t *testing.T) {
	tmp := t.TempDir()
	configDir := filepath.Join(tmp, "fab", "project")
	if err := os.MkdirAll(configDir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(configDir, "config.yaml")

	// Write existing content
	existing := "project:\n  name: test-project\nfab_version: \"0.42.0\"\n"
	if err := os.WriteFile(path, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := setFabVersion(path, "0.43.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify version updated
	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("cannot read back fab_version: %v", err)
	}
	if v != "0.43.0" {
		t.Errorf("expected 0.43.0, got %s", v)
	}
}

func TestStampMigrationVersion_FreshDir(t *testing.T) {
	repoRoot := t.TempDir()

	if err := stampMigrationVersion(repoRoot, "1.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, "fab", ".kit-migration-version"))
	if err != nil {
		t.Fatalf("expected file to be created: %v", err)
	}
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("got %q, want %q", string(got), want)
	}
}

func TestStampMigrationVersion_OverwritesExisting(t *testing.T) {
	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, "fab", ".kit-migration-version")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("0.1.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := stampMigrationVersion(repoRoot, "1.6.1"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("got %q, want %q (expected overwrite)", string(got), want)
	}
}

func TestCopyDir(t *testing.T) {
	// Create source structure
	src := t.TempDir()
	if err := os.MkdirAll(filepath.Join(src, "skills"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "VERSION"), []byte("0.43.0\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "skills", "test.md"), []byte("# Test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Copy to destination
	dst := filepath.Join(t.TempDir(), "kit")
	if err := copyDir(src, dst); err != nil {
		t.Fatalf("copyDir failed: %v", err)
	}

	// Verify files
	data, err := os.ReadFile(filepath.Join(dst, "VERSION"))
	if err != nil {
		t.Fatalf("VERSION not found: %v", err)
	}
	if string(data) != "0.43.0\n" {
		t.Errorf("unexpected VERSION content: %s", string(data))
	}

	data, err = os.ReadFile(filepath.Join(dst, "skills", "test.md"))
	if err != nil {
		t.Fatalf("skills/test.md not found: %v", err)
	}
	if string(data) != "# Test\n" {
		t.Errorf("unexpected skill content: %s", string(data))
	}
}

func TestInit_RequiresGitRepoBeforeAnyWork(t *testing.T) {
	requireGit(t)
	dir := t.TempDir() // not a git repository
	chdir(t, dir)
	t.Setenv("HOME", t.TempDir())
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir)) // never walk up into an outer repo

	// Any network call would mean the precondition ran too late.
	var apiHits int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiHits++
		io.WriteString(w, `{"tag_name": "v9.9.9"}`)
	}))
	defer srv.Close()
	origAPI := githubAPIURL
	githubAPIURL = srv.URL
	defer func() { githubAPIURL = origAPI }()

	err := Init("1.5.0")
	if err == nil {
		t.Fatal("expected Init to fail outside a git repository")
	}
	if !strings.Contains(err.Error(), "git repository") {
		t.Errorf("expected git-repository precondition error, got: %v", err)
	}
	if apiHits != 0 {
		t.Errorf("Init performed %d network call(s) before the git check", apiHits)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "fab")); !os.IsNotExist(statErr) {
		t.Error("Init left fab/ artifacts behind despite failing the precondition")
	}
}

func TestInit_ThreadsVersionsIntoSync(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()
	if out, err := exec.Command("git", "init", dir).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	chdir(t, dir)
	home := t.TempDir()
	t.Setenv("HOME", home)

	const latest = "0.51.0"
	populateRemoteCache(t, home, latest)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, `{"tag_name": "v`+latest+`"}`)
	}))
	defer srv.Close()
	origAPI := githubAPIURL
	githubAPIURL = srv.URL
	defer func() { githubAPIURL = origAPI }()

	var gotSystem, gotKit string
	stubRunSync(t, func(systemVersion, kitVersion string, shimOnly, projectOnly bool) error {
		gotSystem, gotKit = systemVersion, kitVersion
		return nil
	})

	if err := Init("1.5.0"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if gotSystem != "1.5.0" || gotKit != latest {
		t.Errorf("Sync called with (system=%q, kit=%q), want (1.5.0, %s)", gotSystem, gotKit, latest)
	}
	v, err := readFabVersion(filepath.Join(dir, "fab", "project", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	if v != latest {
		t.Errorf("fab_version = %q, want %s", v, latest)
	}
}
