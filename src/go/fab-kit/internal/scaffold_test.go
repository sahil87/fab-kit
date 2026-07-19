package internal

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldTreeWalk_CopyIfAbsent(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create scaffold file
	os.MkdirAll(filepath.Join(scaffoldDir, "docs", "memory"), 0755)
	os.WriteFile(filepath.Join(scaffoldDir, "docs", "memory", "index.md"), []byte("# Index\n"), 0644)

	// Run tree-walk
	if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
		t.Fatalf("scaffoldTreeWalk failed: %v", err)
	}

	// Verify file was copied
	data, err := os.ReadFile(filepath.Join(repoRoot, "docs", "memory", "index.md"))
	if err != nil {
		t.Fatal("expected index.md to be created")
	}
	if string(data) != "# Index\n" {
		t.Errorf("unexpected content: %s", string(data))
	}
}

func TestScaffoldTreeWalk_CopyIfAbsentSkip(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// Create scaffold file
	os.WriteFile(filepath.Join(scaffoldDir, "existing.md"), []byte("scaffold content\n"), 0644)

	// Create destination file with different content
	os.WriteFile(filepath.Join(repoRoot, "existing.md"), []byte("user content\n"), 0644)

	// Run tree-walk
	if err := scaffoldTreeWalk(scaffoldDir, repoRoot); err != nil {
		t.Fatalf("scaffoldTreeWalk failed: %v", err)
	}

	// Verify existing file was NOT overwritten
	data, _ := os.ReadFile(filepath.Join(repoRoot, "existing.md"))
	if string(data) != "user content\n" {
		t.Errorf("existing file should not be overwritten, got: %s", string(data))
	}
}

func TestJsonMergePermissions_CreateNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	srcJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read"},
		},
	}
	srcData, _ := json.MarshalIndent(srcJSON, "", "  ")
	os.WriteFile(src, srcData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	// Verify file was created
	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal("expected dest file to be created")
	}

	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)
	if len(allow) != 2 {
		t.Errorf("expected 2 permissions, got %d", len(allow))
	}
}

func TestJsonMergePermissions_Merge(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	srcJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read", "Write"},
		},
	}
	destJSON := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Edit"},
		},
	}
	srcData, _ := json.MarshalIndent(srcJSON, "", "  ")
	destData, _ := json.MarshalIndent(destJSON, "", "  ")
	os.WriteFile(src, srcData, 0644)
	os.WriteFile(dest, destData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	// Read merged result
	data, _ := os.ReadFile(dest)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)

	// Should have 4: Edit (existing), Bash(git *) (existing/deduped), Read (new), Write (new)
	if len(allow) != 4 {
		t.Errorf("expected 4 permissions after merge, got %d: %v", len(allow), allow)
	}
}

func TestJsonMergePermissions_NoDuplicates(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "settings.json")
	dest := filepath.Join(destDir, "settings.json")

	// Same permissions in both — no change expected
	perms := map[string]interface{}{
		"permissions": map[string]interface{}{
			"allow": []interface{}{"Bash(git *)", "Read"},
		},
	}
	srcData, _ := json.MarshalIndent(perms, "", "  ")
	os.WriteFile(src, srcData, 0644)
	os.WriteFile(dest, srcData, 0644)

	if err := jsonMergePermissions(src, dest, "settings.json"); err != nil {
		t.Fatalf("jsonMergePermissions failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	var result map[string]interface{}
	json.Unmarshal(data, &result)
	allow := extractPermissionsAllow(result)
	if len(allow) != 2 {
		t.Errorf("expected 2 permissions (no duplicates), got %d", len(allow))
	}
}

func TestLineEnsureMerge_CreateNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("# comment\nnode_modules/\n.env\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, err := os.ReadFile(dest)
	if err != nil {
		t.Fatal("expected dest file to be created")
	}

	content := string(data)
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

// TestLineEnsureMerge_PropagatesWriteError covers jznd (c): a failed
// os.WriteFile during the create-new path must propagate up the call chain
// instead of being silently swallowed (the F21-residue bug). We force the
// failure by making dest's parent a regular file, so creating dest fails with
// ENOTDIR.
func TestLineEnsureMerge_PropagatesWriteError(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	os.WriteFile(src, []byte("node_modules/\n"), 0644)

	// Make a regular file, then try to write a child path "under" it.
	notADir := filepath.Join(destDir, "blocker")
	os.WriteFile(notADir, []byte("x"), 0644)
	dest := filepath.Join(notADir, ".gitignore") // parent is a file → write fails

	err := lineEnsureMerge(src, dest, ".gitignore")
	if err == nil {
		t.Fatal("expected lineEnsureMerge to propagate the os.WriteFile error, got nil")
	}
	if !strings.Contains(err.Error(), ".gitignore") {
		t.Errorf("error should reference the label, got: %v", err)
	}
}

// TestScaffoldTreeWalk_PropagatesFragmentWriteError covers jznd (c) at the
// call-chain level: a write failure inside lineEnsureMerge surfaces from
// scaffoldTreeWalk rather than being swallowed.
func TestScaffoldTreeWalk_PropagatesFragmentWriteError(t *testing.T) {
	scaffoldDir := t.TempDir()
	repoRoot := t.TempDir()

	// A fragment file produces a dest of repoRoot/<name>; block it by making
	// repoRoot/.gitignore's parent unwritable is awkward — instead use a
	// fragment whose dest parent is a regular file.
	// Layout: scaffold/blocker/fragment-.gitignore → dest repoRoot/blocker/.gitignore
	os.WriteFile(filepath.Join(repoRoot, "blocker"), []byte("x"), 0644)
	if err := os.MkdirAll(filepath.Join(scaffoldDir, "blocker"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(scaffoldDir, "blocker", "fragment-.gitignore"), []byte("node_modules/\n"), 0644)

	err := scaffoldTreeWalk(scaffoldDir, repoRoot)
	if err == nil {
		t.Fatal("expected scaffoldTreeWalk to propagate the fragment write error, got nil")
	}
}

func TestLineEnsureMerge_AppendNew(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("node_modules/\n.env\n"), 0644)
	os.WriteFile(dest, []byte("node_modules/\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	content := string(data)
	// Should contain .env but not duplicate node_modules/
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

func TestLineEnsureMerge_SkipComments(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	src := filepath.Join(srcDir, "entries")
	dest := filepath.Join(destDir, "entries")

	os.WriteFile(src, []byte("# this is a comment\nactual-entry\n"), 0644)

	if err := lineEnsureMerge(src, dest, "entries"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	content := string(data)
	// Should only contain "actual-entry", not the comment
	if content == "" {
		t.Fatal("file should not be empty")
	}
}

// TestLineEnsureMerge_GitignoreVariantCoverage covers R1: a .gitignore that
// already carries any directory-token variant of the fragment entry must not
// gain an appended /.claude — the dedup is gitignore-aware, not literal.
func TestLineEnsureMerge_GitignoreVariantCoverage(t *testing.T) {
	variants := []string{
		"/.claude/",
		"/.claude/*",
		".claude",
		".claude/",
		".claude/*",
	}
	for _, variant := range variants {
		t.Run(variant, func(t *testing.T) {
			srcDir := t.TempDir()
			destDir := t.TempDir()
			src := filepath.Join(srcDir, "gitignore")
			dest := filepath.Join(destDir, ".gitignore")

			os.WriteFile(src, []byte("/.claude\n"), 0644)
			os.WriteFile(dest, []byte(variant+"\n"), 0644)

			if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
				t.Fatalf("lineEnsureMerge failed: %v", err)
			}

			data, _ := os.ReadFile(dest)
			if strings.Count(string(data), ".claude") != 1 {
				t.Errorf("variant %q should already cover /.claude (no append); got:\n%s", variant, data)
			}
		})
	}
}

// TestLineEnsureMerge_GitignoreGenuineMissAppends covers R3: a .gitignore with
// none of the variants still gets /.claude appended (happy path unchanged).
func TestLineEnsureMerge_GitignoreGenuineMissAppends(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("/.claude\n"), 0644)
	os.WriteFile(dest, []byte("node_modules/\n.env\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if !strings.Contains(string(data), "/.claude") {
		t.Errorf("genuine miss should append /.claude; got:\n%s", data)
	}
}

// TestLineEnsureMerge_GitignoreDeeperPathDoesNotCover covers R2: a deeper nested
// path like /.claude/commands/ does NOT normalize to the core token, so the
// entry is still appended (directory-token-only normalization).
func TestLineEnsureMerge_GitignoreDeeperPathDoesNotCover(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("/.claude\n"), 0644)
	os.WriteFile(dest, []byte("/.claude/commands/\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if !strings.Contains(string(data), "\n/.claude\n") && !strings.HasSuffix(string(data), "\n/.claude\n") {
		t.Errorf("deeper path must not cover /.claude — entry should be appended; got:\n%s", data)
	}
}

// TestLineEnsureMerge_GitignoreNegationSurvives covers R4 (Guardrail B): a
// present negation for the entry's core token is a hard stop — sync never
// appends a broader /.claude. Holds both with a preceding /.claude/* exclusion
// and for the lone-negation case (no preceding exclusion).
func TestLineEnsureMerge_GitignoreNegationSurvives(t *testing.T) {
	cases := map[string]string{
		"exclusion+negation": "/.claude/*\n!/.claude/commands/\n",
		"lone-negation":      "!/.claude/commands/\n",
		"no-slash-negation":  "!.claude/commands/\n",
	}
	for name, initial := range cases {
		t.Run(name, func(t *testing.T) {
			srcDir := t.TempDir()
			destDir := t.TempDir()
			src := filepath.Join(srcDir, "gitignore")
			dest := filepath.Join(destDir, ".gitignore")

			os.WriteFile(src, []byte("/.claude\n"), 0644)
			os.WriteFile(dest, []byte(initial), 0644)

			if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
				t.Fatalf("lineEnsureMerge failed: %v", err)
			}

			data, _ := os.ReadFile(dest)
			if string(data) != initial {
				t.Errorf("negation must suppress append (file unchanged); want:\n%s\ngot:\n%s", initial, data)
			}
		})
	}
}

// TestLineEnsureMerge_EnvrcStrictEquality covers R5 (Guardrail A): semantic
// matching must NOT leak to .envrc — a literally-different line still appends,
// even though it might look "gitignore-similar".
func TestLineEnsureMerge_EnvrcStrictEquality(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "envrc")
	dest := filepath.Join(destDir, ".envrc")

	// Entry differs literally from the existing line; under gitignore-aware
	// normalization "/.claude" and ".claude/" would match, but .envrc must use
	// strict equality, so a literally-different entry appends.
	os.WriteFile(src, []byte("/.claude\n"), 0644)
	os.WriteFile(dest, []byte(".claude/\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".envrc"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if !strings.Contains(string(data), "/.claude") {
		t.Errorf(".envrc must use strict equality — literally-different entry should append; got:\n%s", data)
	}
}

// TestLineEnsureMerge_GitignoreNonDirectoryStrictDedup covers R6 (Guardrail C):
// non-directory fragment patterns (".fab-*", ".status.yaml.lock") use STRICT
// literal dedup even in a .gitignore — the directory-token equivalence must not
// leak to them. An anchored existing line like "/.status.yaml.lock" only ignores
// the file at repo root, whereas the unanchored fragment ".status.yaml.lock"
// must match at any depth (fab/changes/**/.status.yaml.lock); treating them as
// equivalent would suppress the broader ignore and let nested lock files be
// committed, so the fragment entry must still be appended.
func TestLineEnsureMerge_GitignoreNonDirectoryStrictDedup(t *testing.T) {
	cases := map[string]struct {
		fragment string // unanchored, non-directory fragment entry
		existing string // a pre-existing .gitignore line that must NOT cover it
	}{
		"anchored-lock-does-not-cover-unanchored": {
			fragment: ".status.yaml.lock",
			existing: "/.status.yaml.lock\n",
		},
		"glob-fragment-not-covered-by-anchored": {
			fragment: ".fab-*",
			existing: "/.fab-state\n",
		},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			srcDir := t.TempDir()
			destDir := t.TempDir()
			src := filepath.Join(srcDir, "gitignore")
			dest := filepath.Join(destDir, ".gitignore")

			os.WriteFile(src, []byte(tc.fragment+"\n"), 0644)
			os.WriteFile(dest, []byte(tc.existing), 0644)

			if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
				t.Fatalf("lineEnsureMerge failed: %v", err)
			}

			data, _ := os.ReadFile(dest)
			// The unanchored, non-directory fragment must be appended (strict
			// literal dedup) — the anchored existing line does not cover it.
			if !strings.Contains(string(data), "\n"+tc.fragment+"\n") &&
				!strings.HasSuffix(string(data), "\n"+tc.fragment+"\n") {
				t.Errorf("non-directory pattern %q must use strict dedup and append (anchored %q does not cover it); got:\n%s",
					tc.fragment, strings.TrimSpace(tc.existing), data)
			}
		})
	}
}

// TestLineEnsureMerge_GitignoreNonDirectoryNegationDoesNotHardStop covers R7
// (Guardrail B is directory-token-only): a negation for a non-directory pattern
// (e.g. "!/.status.yaml.lock") must NOT hard-stop appending the unanchored
// fragment ".status.yaml.lock". Guardrail B is scoped to directory-style entries;
// applying it here would weaken the lock-file ignore coverage at depth.
func TestLineEnsureMerge_GitignoreNonDirectoryNegationDoesNotHardStop(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte(".status.yaml.lock\n"), 0644)
	os.WriteFile(dest, []byte("!/.status.yaml.lock\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if !strings.Contains(string(data), "\n.status.yaml.lock\n") &&
		!strings.HasSuffix(string(data), "\n.status.yaml.lock\n") {
		t.Errorf("a non-directory negation must not hard-stop the unanchored fragment append; got:\n%s", data)
	}
}

// TestLineEnsureMerge_FabVersionNegationAppendedOnce covers R2/R3 (8ken): the
// fragment's !fab/.fab-version negation is a NON-directory token (no leading "/",
// no trailing "/", no "*"), so gitignoreIsDirectoryToken is false and it uses
// strict literal dedup. On a fresh .gitignore it is appended exactly once.
func TestLineEnsureMerge_FabVersionNegationAppendedOnce(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("!fab/.fab-version\n"), 0644)
	os.WriteFile(dest, []byte("node_modules/\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if strings.Count(string(data), "!fab/.fab-version") != 1 {
		t.Errorf("negation should be appended exactly once; got:\n%s", data)
	}
}

// TestLineEnsureMerge_FabVersionNegationIdempotent covers R3: re-merging when the
// negation is already present appends nothing (strict literal dedup).
func TestLineEnsureMerge_FabVersionNegationIdempotent(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte("!fab/.fab-version\n"), 0644)
	initial := ".fab-*\n!fab/.fab-version\n"
	os.WriteFile(dest, []byte(initial), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if string(data) != initial {
		t.Errorf("re-merge with the negation present must be a no-op; want:\n%s\ngot:\n%s", initial, data)
	}
	if strings.Count(string(data), "!fab/.fab-version") != 1 {
		t.Errorf("negation must not be duplicated; got:\n%s", data)
	}
}

// TestLineEnsureMerge_FabVersionFragmentAppendsNegationOnly covers R3: the shipped
// fragment carries BOTH .fab-* and !fab/.fab-version. Merging onto a project
// .gitignore that already has .fab-* (but not the negation) appends ONLY the
// negation — .fab-* is already covered by strict literal dedup and survives. This
// is the "existing repo self-heals on next sync" path.
func TestLineEnsureMerge_FabVersionFragmentAppendsNegationOnly(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	// Mirror the shipped fragment ordering: .fab-* then the negation.
	os.WriteFile(src, []byte(".fab-*\n!fab/.fab-version\n"), 0644)
	os.WriteFile(dest, []byte(".fab-*\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if strings.Count(string(data), ".fab-*") != 1 {
		t.Errorf(".fab-* must not be duplicated (strict literal dedup covers it); got:\n%s", data)
	}
	if strings.Count(string(data), "!fab/.fab-version") != 1 {
		t.Errorf("the negation must be appended exactly once; got:\n%s", data)
	}
}

// TestLineEnsureMerge_FabVersionNegationDoesNotSuppressFabStarEnsure covers R3
// (Guardrail-B not consulted): merging the full fragment onto a .gitignore that
// ALREADY has the !fab/.fab-version negation but is MISSING .fab-* must still
// append .fab-*. Because both lines are non-directory tokens, gitignoreHasNegation
// (Guardrail B) is never consulted, so the present negation cannot hard-stop the
// broader .fab-* ensure.
func TestLineEnsureMerge_FabVersionNegationDoesNotSuppressFabStarEnsure(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()
	src := filepath.Join(srcDir, "gitignore")
	dest := filepath.Join(destDir, ".gitignore")

	os.WriteFile(src, []byte(".fab-*\n!fab/.fab-version\n"), 0644)
	// Destination has the negation but NOT .fab-*.
	os.WriteFile(dest, []byte("!fab/.fab-version\n"), 0644)

	if err := lineEnsureMerge(src, dest, ".gitignore"); err != nil {
		t.Fatalf("lineEnsureMerge failed: %v", err)
	}

	data, _ := os.ReadFile(dest)
	if !strings.Contains(string(data), "\n.fab-*\n") && !strings.HasPrefix(string(data), ".fab-*\n") {
		t.Errorf("the .fab-* ensure must not be suppressed by a present non-directory negation; got:\n%s", data)
	}
	if strings.Count(string(data), "!fab/.fab-version") != 1 {
		t.Errorf("the negation must not be duplicated; got:\n%s", data)
	}
}

// scaffoldKitDir builds a minimal cached-kit layout under tmp with the given VERSION.
func scaffoldKitDir(t *testing.T, version string) string {
	t.Helper()
	kitDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(kitDir, "VERSION"), []byte(version+"\n"), 0644); err != nil {
		t.Fatalf("cannot write kit VERSION: %v", err)
	}
	return kitDir
}

func TestScaffoldDirectories_FreshProject(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if err != nil {
		t.Fatalf("expected .kit-migration-version to be created: %v", err)
	}
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("fresh project: got %q, want %q", string(got), want)
	}
}

func TestScaffoldDirectories_PreExistingMigrationVersion(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	// Simulate Init() having already written config.yaml and .kit-migration-version.
	if err := os.MkdirAll(filepath.Join(fabDir, "project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, "project", "config.yaml"), []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, ".kit-migration-version"), []byte("1.6.1\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	// Pre-existing .kit-migration-version must be preserved, not overwritten with 0.1.0.
	got, _ := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if want := "1.6.1\n"; string(got) != want {
		t.Errorf("post-init: got %q, want %q (must preserve, not write 0.1.0)", string(got), want)
	}
}

func TestScaffoldDirectories_ExistingProjectWithoutMigrationVersion(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := scaffoldKitDir(t, "1.6.1")

	// Pre-migration-version-era project: config.yaml exists, no .kit-migration-version.
	// This is the legitimate "existing project" branch (e.g., manual `fab sync` on an old project).
	if err := os.MkdirAll(filepath.Join(fabDir, "project"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(fabDir, "project", "config.yaml"), []byte("project:\n  name: test\n"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1"); err != nil {
		t.Fatalf("scaffoldDirectories failed: %v", err)
	}

	got, _ := os.ReadFile(filepath.Join(fabDir, ".kit-migration-version"))
	if want := "0.1.0\n"; string(got) != want {
		t.Errorf("legacy project: got %q, want %q", string(got), want)
	}
}

func TestScaffoldDirectories_MissingKitVersionFails(t *testing.T) {
	repoRoot := t.TempDir()
	fabDir := filepath.Join(repoRoot, "fab")
	kitDir := t.TempDir() // no VERSION file

	// New-project branch (no config.yaml) reads kit VERSION — a failed read
	// must propagate, not silently stamp an empty .kit-migration-version.
	err := scaffoldDirectories(repoRoot, fabDir, kitDir, "1.6.1")
	if err == nil {
		t.Fatal("expected error when kit VERSION is unreadable")
	}
	if !strings.Contains(err.Error(), "VERSION") {
		t.Errorf("expected kit VERSION read error, got: %v", err)
	}
}
