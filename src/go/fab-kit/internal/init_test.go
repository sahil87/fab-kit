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

// writeConfig creates fab/project/config.yaml under a fresh temp dir with the
// given content and returns its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	tmp := t.TempDir()
	dir := filepath.Join(tmp, "fab", "project")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

// TestSetFabVersion_PreservesEverythingButOwnedLine is the core regression: an
// existing config.yaml with header comments, a comment-only mapping block,
// non-alphabetical keys, inline comments, and 2-space nested indentation must be
// byte-identical after the call except the single fab_version line.
func TestSetFabVersion_PreservesEverythingButOwnedLine(t *testing.T) {
	existing := "" +
		"# Providers reference: run `fab config reference`\n" +
		"# agent:\n" +
		"#     tiers: ...\n" +
		"fab_version: 2.13.1\n" +
		"project:\n" +
		"  name: my-repo   # my main repo\n" +
		"  description: FAB Kit\n" +
		"zeta: last\n"
	path := writeConfig(t, existing)

	if err := setFabVersion(path, "2.14.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	want := strings.Replace(existing, "fab_version: 2.13.1\n", "fab_version: 2.14.0\n", 1)
	if string(got) != want {
		t.Errorf("byte-for-byte mismatch\n--- got ---\n%q\n--- want ---\n%q", string(got), want)
	}

	// The commented agent: block must NOT have been collapsed to `agent: null`.
	if strings.Contains(string(got), "agent: null") {
		t.Error("comment-only mapping block was collapsed to `agent: null`")
	}
	// Key order preserved: project must still precede zeta (non-alphabetical).
	if idxProject, idxZeta := strings.Index(string(got), "project:"), strings.Index(string(got), "zeta:"); idxProject > idxZeta {
		t.Error("key order was alphabetized (project should precede zeta)")
	}
	// Indentation untouched: the 2-space nested mapping survives.
	if !strings.Contains(string(got), "  name: my-repo   # my main repo") {
		t.Error("nested 2-space indentation or inline comment was altered")
	}

	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("readback failed: %v", err)
	}
	if v != "2.14.0" {
		t.Errorf("readback = %q, want 2.14.0", v)
	}
}

func TestSetFabVersion_PreservesTrailingComment(t *testing.T) {
	path := writeConfig(t, "fab_version: 1.2.3  # pinned\nproject:\n  name: x\n")

	if err := setFabVersion(path, "2.14.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if want := "fab_version: 2.14.0  # pinned\nproject:\n  name: x\n"; string(got) != want {
		t.Errorf("trailing comment not preserved\ngot:  %q\nwant: %q", string(got), want)
	}
}

func TestSetFabVersion_QuotedValueReplaced(t *testing.T) {
	path := writeConfig(t, "project:\n  name: test-project\nfab_version: \"0.42.0\"\n")

	if err := setFabVersion(path, "0.43.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("readback failed: %v", err)
	}
	if v != "0.43.0" {
		t.Errorf("readback = %q, want 0.43.0", v)
	}
	// Everything but the fab_version line is preserved.
	got, _ := os.ReadFile(path)
	if !strings.Contains(string(got), "project:\n  name: test-project\n") {
		t.Errorf("non-owned lines altered: %q", string(got))
	}
}

func TestSetFabVersion_AppendsWhenMissing(t *testing.T) {
	// Input intentionally lacks a trailing newline to exercise the
	// exactly-one-trailing-newline guarantee.
	path := writeConfig(t, "project:\n  name: my-repo\n# a trailing comment")

	if err := setFabVersion(path, "2.14.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	want := "project:\n  name: my-repo\n# a trailing comment\nfab_version: 2.14.0\n"
	if string(got) != want {
		t.Errorf("append case mismatch\ngot:  %q\nwant: %q", string(got), want)
	}
	if strings.HasSuffix(string(got), "\n\n") {
		t.Error("file ends with a doubled trailing newline")
	}

	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("readback failed: %v", err)
	}
	if v != "2.14.0" {
		t.Errorf("readback = %q, want 2.14.0", v)
	}
}

func TestSetFabVersion_AppendPreservesTrailingNewline(t *testing.T) {
	// Input already ends with exactly one newline — the result must too.
	path := writeConfig(t, "project:\n  name: my-repo\n")

	if err := setFabVersion(path, "2.14.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	if want := "project:\n  name: my-repo\nfab_version: 2.14.0\n"; string(got) != want {
		t.Errorf("append (newline-terminated input) mismatch\ngot:  %q\nwant: %q", string(got), want)
	}
}

// TestSetFabVersion_IgnoresIndentedAndCommentedOccurrences verifies that only a
// column-0 fab_version: key is treated as top-level; an indented or commented
// occurrence must be left untouched and the top-level line appended.
func TestSetFabVersion_IgnoresIndentedAndCommentedOccurrences(t *testing.T) {
	existing := "" +
		"# fab_version: 9.9.9 (this is a comment, not the key)\n" +
		"nested:\n" +
		"  fab_version: 1.1.1\n"
	path := writeConfig(t, existing)

	if err := setFabVersion(path, "2.14.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, _ := os.ReadFile(path)
	// The commented and indented occurrences are untouched.
	if !strings.Contains(string(got), "# fab_version: 9.9.9 (this is a comment, not the key)\n") {
		t.Error("commented fab_version occurrence was altered")
	}
	if !strings.Contains(string(got), "  fab_version: 1.1.1\n") {
		t.Error("indented fab_version occurrence was altered")
	}
	// A new top-level line was appended.
	want := existing + "fab_version: 2.14.0\n"
	if string(got) != want {
		t.Errorf("expected top-level line appended\ngot:  %q\nwant: %q", string(got), want)
	}

	v, err := readFabVersion(path)
	if err != nil {
		t.Fatalf("readback failed: %v", err)
	}
	if v != "2.14.0" {
		t.Errorf("readback = %q, want 2.14.0", v)
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

func TestInit_FromSubdirectoryWritesAtRepoRoot(t *testing.T) {
	requireGit(t)
	root := t.TempDir()
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	subdir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	chdir(t, subdir)
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

	stubRunSync(t, func(systemVersion, kitVersion string, shimOnly, projectOnly bool) error {
		return nil
	})

	if err := Init("1.5.0"); err != nil {
		t.Fatalf("Init failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(subdir, "fab")); !os.IsNotExist(err) {
		t.Error("Init wrote fab/ into the subdirectory instead of the repo root")
	}
	v, err := readFabVersion(filepath.Join(root, "fab", "project", "config.yaml"))
	if err != nil {
		t.Fatalf("config.yaml not at repo root: %v", err)
	}
	if v != latest {
		t.Errorf("fab_version = %q, want %s", v, latest)
	}
	if _, err := os.Stat(filepath.Join(root, "fab", ".kit-migration-version")); err != nil {
		t.Errorf(".kit-migration-version not at repo root: %v", err)
	}
}
