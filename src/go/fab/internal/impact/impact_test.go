package impact

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestParseShortstat(t *testing.T) {
	cases := []struct {
		name        string
		line        string
		wantAdded   int
		wantDeleted int
	}{
		{"both", " 3 files changed, 142 insertions(+), 38 deletions(-)\n", 142, 38},
		{"only_insertions", " 1 file changed, 5 insertions(+)\n", 5, 0},
		{"only_deletions", " 1 file changed, 7 deletions(-)\n", 0, 7},
		{"singular", " 1 file changed, 1 insertion(+), 1 deletion(-)\n", 1, 1},
		{"empty", "", 0, 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			a, d := parseShortstat(tc.line)
			if a != tc.wantAdded || d != tc.wantDeleted {
				t.Errorf("parseShortstat(%q) = (%d, %d), want (%d, %d)", tc.line, a, d, tc.wantAdded, tc.wantDeleted)
			}
		})
	}
}

func TestCompute_EmptyExcludesNoExcludingBlock(t *testing.T) {
	dir := setupRepo(t)
	withCwd(t, dir, func() {
		res, err := Compute("base", "HEAD", nil)
		if err != nil {
			t.Fatalf("Compute: %v", err)
		}
		if res.Excluding != nil {
			t.Errorf("expected no Excluding when excludes is empty, got %+v", res.Excluding)
		}
		if res.Added <= 0 {
			t.Errorf("expected non-zero Added, got %d", res.Added)
		}
	})
}

func TestCompute_ExcludesEmitsExcluding(t *testing.T) {
	dir := setupRepo(t)
	withCwd(t, dir, func() {
		res, err := Compute("base", "HEAD", []string{"docs/"})
		if err != nil {
			t.Fatalf("Compute: %v", err)
		}
		if res.Excluding == nil {
			t.Fatal("expected Excluding block when excludes is non-empty")
		}
		// docs/ excluded — exclude pass should have fewer (or equal) additions
		if res.Excluding.Added > res.Added {
			t.Errorf("excluding.added (%d) should not exceed raw.added (%d)", res.Excluding.Added, res.Added)
		}
	})
}

func TestCompute_BaseEmptyError(t *testing.T) {
	_, err := Compute("", "HEAD", nil)
	if err == nil {
		t.Fatal("expected error when base is empty")
	}
}

func TestCompute_BadBaseError(t *testing.T) {
	dir := setupRepo(t)
	withCwd(t, dir, func() {
		_, err := Compute("nonexistent-ref-xyzzy", "HEAD", nil)
		if err == nil {
			t.Fatal("expected error when base is unresolvable")
		}
	})
}

// setupRepo creates a tiny git repo with two commits and tags the first as
// "base", returning the repo path. Adds files in both root and docs/ so
// pathspec exclusion tests are meaningful.
func setupRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	run := func(args ...string) {
		cmd := exec.Command(args[0], args[1:]...)
		cmd.Dir = dir
		if out, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("%v: %s", args, out)
		}
	}

	run("git", "init", "-q", "-b", "main")
	run("git", "config", "user.email", "test@example.com")
	run("git", "config", "user.name", "test")
	run("git", "config", "commit.gpgsign", "false")

	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "a.txt")
	run("git", "commit", "-q", "-m", "initial")
	run("git", "tag", "base")

	// Modify root file (counts in raw and excluding when only docs/ excluded)
	if err := os.WriteFile(filepath.Join(dir, "a.txt"), []byte("hello\nworld\nfoo\nbar\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Add a docs/ file (counts in raw, excluded in excluding)
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "b.md"), []byte("doc\nmore\nlines\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "second")

	return dir
}

// withCwd runs fn with cwd set to dir, restoring on exit. Tests using this
// MUST NOT be run in parallel — they mutate process-wide cwd.
func withCwd(t *testing.T, dir string, fn func()) {
	t.Helper()
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(orig); err != nil {
			t.Fatal(err)
		}
	}()
	fn()
}
