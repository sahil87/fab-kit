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

func TestStampFabVersion_FreshDir(t *testing.T) {
	repoRoot := t.TempDir()

	if err := stampFabVersion(repoRoot, "2.15.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(repoRoot, "fab", ".fab-version"))
	if err != nil {
		t.Fatalf("expected fab/.fab-version to be created: %v", err)
	}
	if want := "2.15.0\n"; string(got) != want {
		t.Errorf("got %q, want %q", string(got), want)
	}
}

func TestStampFabVersion_OverwritesExisting(t *testing.T) {
	repoRoot := t.TempDir()
	path := filepath.Join(repoRoot, "fab", ".fab-version")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("2.14.0\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := stampFabVersion(repoRoot, "2.15.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	got, _ := os.ReadFile(path)
	if want := "2.15.0\n"; string(got) != want {
		t.Errorf("got %q, want %q (expected overwrite)", string(got), want)
	}
}

// TestStampFabVersion_DoesNotWriteConfig confirms the relocation: stampFabVersion
// writes only fab/.fab-version and never touches config.yaml (config.yaml is now
// written solely by `fab config init/upgrade`).
func TestStampFabVersion_DoesNotWriteConfig(t *testing.T) {
	repoRoot := t.TempDir()
	if err := stampFabVersion(repoRoot, "2.15.0"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, err := os.Stat(filepath.Join(repoRoot, "fab", "project", "config.yaml")); !os.IsNotExist(err) {
		t.Error("stampFabVersion must not create config.yaml — it owns only fab/.fab-version")
	}
}

// TestGenerateProjectConfig_StubFallback: when the pinned fab-go writes no
// config.yaml (a predates-subcommand stub binary), generateProjectConfig falls
// open to the embedded stub so a fresh repo always has a config.yaml — carrying
// the detected identity seed (the repo folder name).
func TestGenerateProjectConfig_StubFallback(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "my-cool-repo")
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatal(err)
	}
	// A stub fab-go that exits 0 but writes nothing (mimics a predates-subcommand
	// binary that ignores the unknown `config init --project`).
	binDir := t.TempDir()
	fabGoBin := filepath.Join(binDir, "fab-go")
	if err := os.WriteFile(fabGoBin, []byte("#!/bin/sh\nexit 0\n"), 0755); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(repoRoot, "fab", "project", "config.yaml")

	if err := generateProjectConfig(fabGoBin, repoRoot, configPath); err != nil {
		t.Fatalf("generateProjectConfig: %v", err)
	}
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("stub fallback did not write a config.yaml: %v", err)
	}
	// The stub carries the A-class identity fields so preflight passes.
	for _, want := range []string{"project:", "name:", "source_paths:"} {
		if !strings.Contains(string(data), want) {
			t.Errorf("stub config missing %q:\n%s", want, string(data))
		}
	}
	// The detected repo folder name is seeded into the stub (not the placeholder).
	if !strings.Contains(string(data), `name: "my-cool-repo"`) {
		t.Errorf("stub must carry the detected repo name, got:\n%s", string(data))
	}
}

// TestGenerateProjectConfig_PassesDetectedSeed: on the success path (a fab-go that
// honors `config init --project`), generateProjectConfig passes the detected
// name/source_paths/test_paths as flags. A fake fab-go records its args so we can
// assert the seed flags were passed; it then writes a config.yaml so the success
// branch is taken.
func TestGenerateProjectConfig_PassesDetectedSeed(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "widget-service")
	if err := os.MkdirAll(filepath.Join(repoRoot, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	// A Go marker so test_paths detection fires.
	if err := os.WriteFile(filepath.Join(repoRoot, "go.mod"), []byte("module x\n"), 0644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(repoRoot, "fab", "project", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatal(err)
	}

	// A fake fab-go: records "$@" to an args file, then writes the config so the
	// success branch (exit 0 AND file present) is taken.
	binDir := t.TempDir()
	fabGoBin := filepath.Join(binDir, "fab-go")
	argsFile := filepath.Join(binDir, "args.txt")
	script := "#!/bin/sh\nprintf '%s\\n' \"$@\" > " + argsFile + "\ncat > " + configPath + " <<'YAML'\nproject:\n  name: from-fab-go\nYAML\nexit 0\n"
	if err := os.WriteFile(fabGoBin, []byte(script), 0755); err != nil {
		t.Fatal(err)
	}

	if err := generateProjectConfig(fabGoBin, repoRoot, configPath); err != nil {
		t.Fatalf("generateProjectConfig: %v", err)
	}
	recorded, err := os.ReadFile(argsFile)
	if err != nil {
		t.Fatalf("fake fab-go did not record args: %v", err)
	}
	args := string(recorded)
	for _, want := range []string{"--project", "--name", "widget-service", "--source-path", "src/", "--test-path", "**/*_test.go"} {
		if !strings.Contains(args, want) {
			t.Errorf("expected the detected seed flag %q to be passed, got args:\n%s", want, args)
		}
	}
}

// TestDetectProjectSeed covers the mechanical detection: repo-folder name,
// existing src/ dir, and ecosystem test_paths from marker files.
func TestDetectProjectSeed(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "acme-app")
	if err := os.MkdirAll(filepath.Join(repoRoot, "src"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repoRoot, "pyproject.toml"), []byte("[tool]\n"), 0644); err != nil {
		t.Fatal(err)
	}
	seed := detectProjectSeed(repoRoot)
	if seed.name != "acme-app" {
		t.Errorf("name = %q, want acme-app", seed.name)
	}
	if len(seed.sourcePaths) != 1 || seed.sourcePaths[0] != "src/" {
		t.Errorf("sourcePaths = %v, want [src/]", seed.sourcePaths)
	}
	if len(seed.testPaths) != 2 || seed.testPaths[0] != "**/test_*.py" || seed.testPaths[1] != "**/*_test.py" {
		t.Errorf("testPaths = %v, want [**/test_*.py **/*_test.py]", seed.testPaths)
	}
}

// TestDetectProjectSeed_NoMarkers: an empty repo yields the folder name only (no
// source_paths, no test_paths — those stay fence-advertised).
func TestDetectProjectSeed_NoMarkers(t *testing.T) {
	repoRoot := filepath.Join(t.TempDir(), "bare")
	if err := os.MkdirAll(repoRoot, 0755); err != nil {
		t.Fatal(err)
	}
	seed := detectProjectSeed(repoRoot)
	if seed.name != "bare" {
		t.Errorf("name = %q, want bare", seed.name)
	}
	if len(seed.sourcePaths) != 0 {
		t.Errorf("sourcePaths = %v, want empty", seed.sourcePaths)
	}
	if len(seed.testPaths) != 0 {
		t.Errorf("testPaths = %v, want empty", seed.testPaths)
	}
}

// TestGenerateProjectConfig_NeverOverwrites: an existing config.yaml is left
// untouched (generateProjectConfig is copy-if-absent for the file).
func TestGenerateProjectConfig_NeverOverwrites(t *testing.T) {
	repoRoot := t.TempDir()
	configPath := filepath.Join(repoRoot, "fab", "project", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		t.Fatal(err)
	}
	existing := "project:\n  name: keep-me\n"
	if err := os.WriteFile(configPath, []byte(existing), 0644); err != nil {
		t.Fatal(err)
	}

	if err := generateProjectConfig("/nonexistent/fab-go", repoRoot, configPath); err != nil {
		t.Fatalf("generateProjectConfig: %v", err)
	}
	got, _ := os.ReadFile(configPath)
	if string(got) != existing {
		t.Errorf("existing config.yaml must be preserved, got:\n%s", string(got))
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
	// The version is stamped into fab/.fab-version (not config.yaml).
	repoRoot := dir
	v, err := readFabVersion(repoRoot)
	if err != nil {
		t.Fatal(err)
	}
	if v != latest {
		t.Errorf("resolved fab version = %q, want %s", v, latest)
	}
	dotVer, err := os.ReadFile(filepath.Join(dir, "fab", ".fab-version"))
	if err != nil {
		t.Fatalf("fab/.fab-version not written: %v", err)
	}
	if strings.TrimSpace(string(dotVer)) != latest {
		t.Errorf("fab/.fab-version = %q, want %s", strings.TrimSpace(string(dotVer)), latest)
	}
	// config.yaml exists (the stub fab-go writes nothing, so the embedded stub
	// fallback fires — a fresh repo must always have a config.yaml).
	if _, err := os.Stat(filepath.Join(dir, "fab", "project", "config.yaml")); err != nil {
		t.Errorf("Init must leave a config.yaml (stub fallback): %v", err)
	}
}

// captureStderr redirects os.Stderr to a pipe for the duration of fn and returns
// everything written to it. warnIfFabVersionIgnored writes directly to os.Stderr,
// so this is how the warning-path tests observe (or confirm the absence of) the line.
func captureStderr(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	orig := os.Stderr
	os.Stderr = w
	defer func() { os.Stderr = orig }()
	done := make(chan string, 1)
	go func() {
		data, _ := io.ReadAll(r)
		done <- string(data)
	}()
	// Wrap fn so the writer is closed even if fn panics or calls runtime.Goexit
	// (e.g. t.Fatal from a future callback); otherwise the drain goroutine blocks
	// on <-done forever and os.Stderr stays redirected into later tests.
	func() {
		defer w.Close()
		fn()
	}()
	out := <-done
	r.Close()
	return out
}

// gitInitRepo creates a temp git repo with the given .gitignore content and a
// fab/.fab-version file on disk. Returns the repo root. Skips when git is absent.
func gitInitRepo(t *testing.T, gitignore string) string {
	t.Helper()
	requireGit(t)
	root := t.TempDir()
	if out, err := exec.Command("git", "init", root).CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(root, ".gitignore"), []byte(gitignore), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(root, "fab"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "fab", ".fab-version"), []byte("2.15.2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestWarnIfFabVersionIgnored_FiresWhenIgnored: a .gitignore whose .fab-* line
// swallows fab/.fab-version triggers the fail-open stderr warning.
func TestWarnIfFabVersionIgnored_FiresWhenIgnored(t *testing.T) {
	root := gitInitRepo(t, ".fab-*\n")
	out := captureStderr(t, func() { warnIfFabVersionIgnored(root) })
	if !strings.Contains(out, "fab: warning:") || !strings.Contains(out, "fab/.fab-version is gitignored") {
		t.Errorf("expected the gitignored warning on stderr, got: %q", out)
	}
	if !strings.Contains(out, "!fab/.fab-version") {
		t.Errorf("warning should advise the negation '!fab/.fab-version', got: %q", out)
	}
}

// TestWarnIfFabVersionIgnored_SilentWhenNegated: a .gitignore that negates the
// path (the fix) produces no warning.
func TestWarnIfFabVersionIgnored_SilentWhenNegated(t *testing.T) {
	root := gitInitRepo(t, ".fab-*\n!fab/.fab-version\n")
	out := captureStderr(t, func() { warnIfFabVersionIgnored(root) })
	if out != "" {
		t.Errorf("expected no warning when the path is negated, got: %q", out)
	}
}

// TestWarnIfFabVersionIgnored_SilentWhenNotIgnored: a .gitignore that never
// ignores the path produces no warning.
func TestWarnIfFabVersionIgnored_SilentWhenNotIgnored(t *testing.T) {
	root := gitInitRepo(t, "node_modules/\n")
	out := captureStderr(t, func() { warnIfFabVersionIgnored(root) })
	if out != "" {
		t.Errorf("expected no warning when the path is not ignored, got: %q", out)
	}
}

// TestWarnIfFabVersionIgnored_SilentOutsideGitRepo: a plain (non-git) directory
// produces no warning and no error — git check-ignore fails and is swallowed
// (fail-open).
func TestWarnIfFabVersionIgnored_SilentOutsideGitRepo(t *testing.T) {
	requireGit(t)
	dir := t.TempDir()                                     // not a git repo
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir)) // never walk up into an outer repo
	if err := os.WriteFile(filepath.Join(dir, ".gitignore"), []byte(".fab-*\n"), 0644); err != nil {
		t.Fatal(err)
	}
	out := captureStderr(t, func() { warnIfFabVersionIgnored(dir) })
	if out != "" {
		t.Errorf("expected no warning outside a git repo, got: %q", out)
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
	v, err := readFabVersion(root)
	if err != nil {
		t.Fatalf("version not resolvable at repo root: %v", err)
	}
	if v != latest {
		t.Errorf("resolved fab version = %q, want %s", v, latest)
	}
	if _, err := os.Stat(filepath.Join(root, "fab", ".fab-version")); err != nil {
		t.Errorf("fab/.fab-version not at repo root: %v", err)
	}
	if _, err := os.Stat(filepath.Join(root, "fab", ".kit-migration-version")); err != nil {
		t.Errorf(".kit-migration-version not at repo root: %v", err)
	}
}
