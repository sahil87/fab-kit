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
	res, err := Compute(dir, "base", "HEAD", nil, nil)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Excluding != nil {
		t.Errorf("expected no Excluding when excludes is empty, got %+v", res.Excluding)
	}
	if res.Added <= 0 {
		t.Errorf("expected non-zero Added, got %d", res.Added)
	}
}

func TestCompute_ExcludesEmitsExcluding(t *testing.T) {
	dir := setupRepo(t)
	res, err := Compute(dir, "base", "HEAD", []string{"docs/"}, nil)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Excluding == nil {
		t.Fatal("expected Excluding block when excludes is non-empty")
	}
	// The raw pass counts everything (fab/ + docs/ included) and stays the
	// base measurement; the excluding pass strips docs/. Since the fixture
	// changes a docs/ file, raw MUST be strictly larger than excluding —
	// guarding against the raw pass accidentally applying the excludes (which
	// would collapse raw == excluding and corrupt the base measurement).
	if res.Added <= res.Excluding.Added {
		t.Errorf("raw.added (%d) must exceed excluding.added (%d) when an excluded path changed", res.Added, res.Excluding.Added)
	}
}

func TestCompute_BaseEmptyError(t *testing.T) {
	_, err := Compute("", "", "HEAD", nil, nil)
	if err == nil {
		t.Fatal("expected error when base is empty")
	}
}

func TestCompute_BadBaseError(t *testing.T) {
	dir := setupRepo(t)
	_, err := Compute(dir, "nonexistent-ref-xyzzy", "HEAD", nil, nil)
	if err == nil {
		t.Fatal("expected error when base is unresolvable")
	}
}

// TestCompute_EmptyTestPathsNoTestsBlock verifies that an empty test_paths
// leaves Result.Tests nil (no extra git pass, single-number behavior).
func TestCompute_EmptyTestPathsNoTestsBlock(t *testing.T) {
	dir := setupRepo(t)
	res, err := Compute(dir, "base", "HEAD", []string{"docs/"}, nil)
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Tests != nil {
		t.Errorf("expected no Tests when testPaths is empty, got %+v", res.Tests)
	}
}

// TestCompute_TestPathsEmitsTests verifies that a non-empty test_paths produces
// a Tests pass counting only the test files, within the scaffolding-excluded
// universe (the test fixture under docs/ is NOT counted because docs/ is
// excluded).
func TestCompute_TestPathsEmitsTests(t *testing.T) {
	dir := setupRepo(t)
	res, err := Compute(dir, "base", "HEAD", []string{"docs/"}, []string{"**/*_test.go"})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Tests == nil {
		t.Fatal("expected Tests block when testPaths is non-empty")
	}
	// foo_test.go has 2 added lines (see setupRepo); docs/foo_test.go is
	// excluded by the docs/ exclude, so it must NOT be counted.
	if res.Tests.Added != 2 {
		t.Errorf("tests.added = %d, want 2 (root test file only; docs/ test fixture excluded)", res.Tests.Added)
	}
	if res.Tests.Net != 2 {
		t.Errorf("tests.net = %d, want 2", res.Tests.Net)
	}
}

// TestCompute_TestPathsWithoutExcludesRawUniverse verifies the edge case where
// true_impact_exclude is empty but test_paths is set: the test pass runs with
// only the include pathspec (no :(exclude) args), attributing tests within the
// raw universe. Excluding stays nil.
func TestCompute_TestPathsWithoutExcludesRawUniverse(t *testing.T) {
	dir := setupRepo(t)
	res, err := Compute(dir, "base", "HEAD", nil, []string{"**/*_test.go"})
	if err != nil {
		t.Fatalf("Compute: %v", err)
	}
	if res.Excluding != nil {
		t.Errorf("expected no Excluding when excludes is empty, got %+v", res.Excluding)
	}
	if res.Tests == nil {
		t.Fatal("expected Tests block when testPaths is non-empty")
	}
	// No excludes → docs/foo_test.go (2 lines) is counted too: 2 + 2 = 4.
	if res.Tests.Added != 4 {
		t.Errorf("tests.added = %d, want 4 (both test files counted within raw universe)", res.Tests.Added)
	}
}

// setupRepo creates a tiny git repo with two commits and tags the first as
// "base", returning the repo path. Adds files in root and docs/ so pathspec
// exclusion tests are meaningful, plus a test file in each location
// (foo_test.go) so test-path attribution (and its interaction with excludes)
// can be exercised.
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
	// Add a root test file (2 lines) — counted by the test pass within the
	// scaffolding-excluded universe.
	if err := os.WriteFile(filepath.Join(dir, "foo_test.go"), []byte("package foo\n// test\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	// Add a docs/ test file (2 lines) — matches the test glob BUT lives under
	// the docs/ exclude, so it must NOT be counted when docs/ is excluded.
	if err := os.WriteFile(filepath.Join(dir, "docs", "foo_test.go"), []byte("package foo\n// doctest\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("git", "add", "-A")
	run("git", "commit", "-q", "-m", "second")

	return dir
}
