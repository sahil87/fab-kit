package main

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/sahil87/fab-kit/src/go/fab/internal/memoryindex"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func runDocs(t *testing.T, args ...string) (error, string, string) {
	t.Helper()
	c := docsIndexCmd()
	c.SetArgs(args)
	c.SilenceUsage = true
	var out, diag bytes.Buffer
	c.SetOut(&out)
	c.SetErr(&diag)
	err := c.Execute()
	return err, out.String(), diag.String()
}
func configureDocs(t *testing.T, repo string) {
	t.Helper()
	mustWrite(t, filepath.Join(repo, "fab/project/config.yaml"), "docs_index:\n  roots:\n    - path: docs/memory\n      log: true\n    - path: docs/specs\n      also_accept: [README.md]\n")
	mustWrite(t, filepath.Join(repo, "docs/specs/topic.md"), "# Spec topic\n")
}
func TestDocsIndexRootSelectionAndAlias(t *testing.T) {
	repo := setupFabRepo(t)
	configureDocs(t, repo)
	if err, _, diag := runMemoryIndex(t); err != nil || strings.Count(diag, "is deprecated") != 1 {
		t.Fatalf("alias error=%v stderr=%s", err, diag)
	}
	if _, err := os.Stat(filepath.Join(repo, "docs/specs/index.md")); !os.IsNotExist(err) {
		t.Fatal("alias processed specs")
	}
	if err, out, diag := runMemoryIndex(t, "--check", "--json"); err != nil || !json.Valid([]byte(out)) || strings.Count(diag, "is deprecated") != 1 {
		t.Fatalf("alias JSON error=%v stdout=%s stderr=%s", err, out, diag)
	}
	if err, _, diag := runDocs(t, "docs/specs"); err != nil || strings.Contains(diag, "deprecated") {
		t.Fatalf("docs-index: %v %s", err, diag)
	}
	if err, out, _ := runDocs(t, "--check", "--json"); err != nil || !strings.Contains(out, `"tier": 0`) {
		t.Fatalf("aggregate check: %v %s", err, out)
	}
	if err, _, _ := runDocs(t, "docs/unknown"); err == nil || !strings.Contains(err.Error(), "docs_index.roots") {
		t.Fatalf("unconfigured root: %v", err)
	}
	if docsIndexCmd().Flags().Lookup("root") != nil {
		t.Fatal("positional-only contract acquired --root")
	}
}
func TestDocsIndexAliasWithoutConfiguredMemory(t *testing.T) {
	repo := setupFabRepo(t)
	mustWrite(t, filepath.Join(repo, "fab/project/config.yaml"), "docs_index: {roots: [{path: docs/specs}]}\n")
	if err, _, _ := runMemoryIndex(t); err != nil {
		t.Fatal(err)
	}
}

// A subprocess captures the existing explicit os.Exit contract for tier 2 and
// JSON tier 1, as the memory-index check integration tests do.
func docsExit(t *testing.T, repo, args string) (int, string, string) {
	t.Helper()
	cmd := exec.Command(os.Args[0], "-test.run=^TestDocsIndexMachineHelper$")
	cmd.Dir = repo
	cmd.Env = append(os.Environ(), "FAB_DOCS_INDEX_TEST_CHILD=1", "FAB_DOCS_INDEX_TEST_ARGS="+args)
	var out, diag bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &diag
	err := cmd.Run()
	code := 0
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			code = e.ExitCode()
		} else {
			t.Fatal(err)
		}
	}
	return code, out.String(), diag.String()
}
func TestDocsIndexMachineHelper(t *testing.T) {
	if os.Getenv("FAB_DOCS_INDEX_TEST_CHILD") != "1" {
		return
	}
	c := docsIndexCmd()
	c.SetArgs(strings.Fields(os.Getenv("FAB_DOCS_INDEX_TEST_ARGS")))
	c.SetOut(os.Stdout)
	c.SetErr(os.Stderr)
	if err := c.Execute(); err != nil {
		os.Exit(1)
	}
	os.Exit(0)
}
func TestDocsIndexCheckExitTiersPerRoot(t *testing.T) {
	repo := setupFabRepo(t)
	configureDocs(t, repo)
	if err, _, _ := runDocs(t); err != nil {
		t.Fatal(err)
	}
	for _, root := range []string{"docs/memory", "docs/specs"} {
		code, out, _ := docsExit(t, repo, root+" --check --json")
		if code != 0 || !json.Valid([]byte(out)) {
			t.Fatalf("%s clean: %d %s", root, code, out)
		}
	}
	mustWrite(t, filepath.Join(repo, "docs/specs/new.md"), "# New\n")
	if code, out, _ := docsExit(t, repo, "docs/specs --check --json"); code != 1 || !strings.Contains(out, `"tier": 1`) {
		t.Fatalf("benign: %d %s", code, out)
	}
	if err, _, _ := runDocs(t, "docs/specs"); err != nil {
		t.Fatal(err)
	}
	// A source deletion after generation must trip tier 2 and dominate the
	// independent benign memory drift in the aggregate result.
	if err := os.Remove(filepath.Join(repo, "docs/specs/new.md")); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(repo, "docs/memory/auth/login.md"), "---\ndescription: Improved\n---\n# Login\n")
	code, out, _ := docsExit(t, repo, "--check --json")
	if code != 2 {
		t.Fatalf("aggregate: %d %s", code, out)
	}
	var report map[string]any
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"tier", "drift", "losses", "malformed", "warnings"} {
		if _, ok := report[key]; !ok {
			t.Errorf("missing %s", key)
		}
	}
	if report["tier"] != float64(2) {
		t.Fatal(report)
	}
	if code, out, _ := docsExit(t, repo, "docs/memory --check --json"); code != 1 {
		t.Fatalf("memory benign: %d %s", code, out)
	}
}

func TestDocsIndexLastNestedDocumentDeletionIsDestructive(t *testing.T) {
	repo := setupFabRepo(t)
	configureDocs(t, repo)
	topic := filepath.Join(repo, "docs/specs/area/topic.md")
	mustWrite(t, topic, "---\ndescription: Nested design\n---\n# Topic\n")
	if err, _, _ := runDocs(t, "docs/specs"); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(topic); err != nil {
		t.Fatal(err)
	}
	code, out, _ := docsExit(t, repo, "docs/specs --check --json")
	if code != 2 || !strings.Contains(out, "area/index.md") {
		t.Fatalf("nested loss: %d %s", code, out)
	}
	code, _, diag := docsExit(t, repo, "docs/specs --check")
	if code != 2 || !strings.Contains(diag, "configured root") || strings.Contains(diag, "_shared/removed-domains") {
		t.Fatalf("generic remediation: %d %s", code, diag)
	}
	if !strings.Contains(diag, "hand-managed/historical content") {
		t.Fatalf("tier-2 stderr must name hand-managed/historical content: %s", diag)
	}
}

func TestDocsIndexHelpDocumentsExcludeNotCurated(t *testing.T) {
	long := docsIndexCmd().Long
	if !strings.Contains(long, "exclude") {
		t.Fatal("help must list the per-root exclude field")
	}
	if strings.Contains(long, "curated") {
		t.Fatal("help must drop the curated block wording")
	}
}

func TestDocsIndexCurrentToSupersededCleanup(t *testing.T) {
	repo := setupFabRepo(t)
	configPath := filepath.Join(repo, "fab/project/config.yaml")
	config := "docs_index:\n  roots:\n    - path: docs/specs\n      log: true\n      also_accept: [README.md]\n"
	mustWrite(t, configPath, config)
	prefix := "docs/specs/archive/"
	seed := "# Seed\n\n## 2026-09-08\n- **Update** [Topic](/archive/v1/topic.md) — Existing history\n"
	for _, folder := range []string{"", "v1/", "v1/deep/", "v2/"} {
		mustWrite(t, filepath.Join(repo, prefix+folder+"topic.md"), "---\ndescription: Topic description\n---\n# Topic\n")
		mustWrite(t, filepath.Join(repo, prefix+folder+"log.seed.md"), seed)
	}
	readme := "# Version two\n\nHuman introduction.\n"
	mustWrite(t, filepath.Join(repo, prefix+"v2/README.md"), readme)
	if err, _, _ := runDocs(t); err != nil {
		t.Fatal(err)
	}
	// Capture alternate prose surrounding the block, including the generator's
	// separator newlines. Cleanup must preserve these bytes exactly.
	readmePath := filepath.Join(repo, prefix+"v2/README.md")
	before, err := os.ReadFile(readmePath)
	if err != nil {
		t.Fatal(err)
	}
	start := strings.Index(string(before), "<!-- fab docs-index:generated:start -->")
	endMarker := "<!-- fab docs-index:generated:end -->"
	end := strings.Index(string(before), endMarker)
	if start < 0 || end < 0 {
		t.Fatalf("missing generated block: %s", before)
	}
	wantReadme := string(before[:start]) + string(before[end+len(endMarker):])
	removed := []string{"log.md", "v1/index.md", "v1/log.md", "v1/deep/index.md", "v1/deep/log.md", "v2/log.md"}
	for _, path := range removed {
		if _, err := os.Stat(filepath.Join(repo, prefix+path)); err != nil {
			t.Fatalf("fixture %s: %v", path, err)
		}
	}
	// Retirement also recognizes artifacts generated before the command rename.
	for _, name := range []string{"v1/deep/index.md", "v1/deep/log.md"} {
		p := filepath.Join(repo, prefix+name)
		data, err := os.ReadFile(p)
		if err != nil {
			t.Fatal(err)
		}
		mustWrite(t, p, strings.ReplaceAll(string(data), "fab docs-index", "fab memory-index"))
	}
	// Human landings and logs were never generated and must remain untouched.
	mustWrite(t, filepath.Join(repo, prefix+"manual/index.md"), "# Human index\n")
	mustWrite(t, filepath.Join(repo, prefix+"manual/log.md"), "# Human history\n")
	mustWrite(t, configPath, config+"      superseded: ['archive/**']\n")
	if code, out, _ := docsExit(t, repo, "--check --json"); code != 1 || !strings.Contains(out, `"tier": 1`) {
		t.Fatalf("cleanup check: %d %s", code, out)
	}
	for _, path := range removed {
		if _, err := os.Stat(filepath.Join(repo, prefix+path)); err != nil {
			t.Fatalf("--check removed %s", path)
		}
	}
	if err, _, _ := runDocs(t); err != nil {
		t.Fatal(err)
	}
	for _, path := range removed {
		if _, err := os.Stat(filepath.Join(repo, prefix+path)); !os.IsNotExist(err) {
			t.Fatalf("obsolete output remains %s: %v", path, err)
		}
	}
	for path, want := range map[string]string{
		"v2/README.md":    wantReadme,
		"manual/index.md": "# Human index\n",
		"manual/log.md":   "# Human history\n",
		"v1/log.seed.md":  seed,
		"v1/topic.md":     "---\ndescription: Topic description\n---\n# Topic\n",
	} {
		got, err := os.ReadFile(filepath.Join(repo, prefix+path))
		if err != nil || string(got) != want {
			t.Fatalf("preserved %s: %v %q", path, err, got)
		}
	}
	archive, err := os.ReadFile(filepath.Join(repo, prefix+"index.md"))
	if err != nil || !strings.Contains(string(archive), "[v1/](v1/)") || strings.Contains(string(archive), "topic.md") {
		t.Fatalf("boundary index: %v %s", err, archive)
	}
	if code, out, _ := docsExit(t, repo, "--check --json"); code != 0 {
		t.Fatalf("after cleanup: %d %s", code, out)
	}
	if err, out, _ := runDocs(t); err != nil || !strings.Contains(out, "already up to date") {
		t.Fatalf("second write: %v %s", err, out)
	}
}

func TestDocsIndexWarningLimitAcrossRoots(t *testing.T) {
	repo := setupFabRepo(t)
	mustWrite(t, filepath.Join(repo, "fab/project/config.yaml"), "docs_index: {roots: [{path: docs/specs}, {path: docs/reference}]}\n")
	for _, root := range []string{"specs", "reference"} {
		for i := 0; i < 20; i++ {
			mustWrite(t, filepath.Join(repo, "docs", root, fmt.Sprintf("topic%02d.md", i)), fmt.Sprintf("# Topic %02d\n", i))
		}
	}
	check := func(wantExit int) {
		t.Helper()
		code, out, diag := docsExit(t, repo, "--check --json")
		if code != wantExit {
			t.Fatalf("exit %d, want %d: %s %s", code, wantExit, out, diag)
		}
		var report memoryindex.LossReport
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatal(err)
		}
		if report.WarningsTotal != 40 || len(report.Warnings) != docsWarningLimit {
			t.Fatalf("warning sampling: %+v", report)
		}
		if !strings.Contains(diag, "[missing-description] … and 35 more (40 total)") {
			t.Fatalf("missing truncation count: %s", diag)
		}
		// Five topic warnings, two width warnings, one truncation summary.
		if strings.Count(strings.TrimSpace(diag), "\n")+1 != docsWarningLimit+3 {
			t.Fatalf("unbounded or missing stderr details: %s", diag)
		}
		var keys map[string]any
		if err := json.Unmarshal([]byte(out), &keys); err != nil {
			t.Fatal(err)
		}
		for _, key := range []string{"tier", "drift", "losses", "malformed", "warnings", "warnings_total"} {
			if _, ok := keys[key]; !ok {
				t.Errorf("missing %s", key)
			}
		}
	}
	check(1)
	if err, _, diag := runDocs(t); err != nil || !strings.Contains(diag, "and 35 more") {
		t.Fatalf("write warnings: %v %s", err, diag)
	}
	check(0) // Advisory truncation cannot turn a byte-clean root into a failure.
	// Losses still outrank a large advisory sample.
	if err := os.Remove(filepath.Join(repo, "docs/specs/topic00.md")); err != nil {
		t.Fatal(err)
	}
	if code, out, _ := docsExit(t, repo, "--check --json"); code != 2 {
		t.Fatalf("lost severity: %d %s", code, out)
	}
}

func TestDocsWarningLimitPerClassPreservesBlocking(t *testing.T) {
	cmd := docsIndexCmd()
	var diag bytes.Buffer
	cmd.SetErr(&diag)
	var warnings []memoryindex.Warning
	for i := 0; i < docsWarningLimit+4; i++ {
		for _, kind := range []string{memoryindex.KindMissingDescription, memoryindex.KindFileSize, memoryindex.KindDepth, memoryindex.KindMalformedFence} {
			warnings = append(warnings, memoryindex.Warning{Kind: kind, Path: fmt.Sprintf("topic%d.md", i)})
		}
	}
	report := memoryindex.Classify(nil, nil)
	reportDocsWarnings(cmd, &report, warnings)
	if report.WarningsTotal != 18 || len(report.Warnings) != 2*docsWarningLimit || len(report.Malformed) != 9 {
		t.Fatalf("incorrect class limits or truncated blocking findings: %+v", report)
	}
	for _, kind := range []string{memoryindex.KindMissingDescription, memoryindex.KindFileSize, memoryindex.KindDepth} {
		if !strings.Contains(diag.String(), "["+kind+"] … and 4 more (9 total)") {
			t.Errorf("missing %s summary: %s", kind, diag.String())
		}
	}
	if report.Tier != memoryindex.TierClean {
		t.Fatalf("advisories changed tier: %+v", report)
	}
	if err := emitCheckReport(cmd, report, false); err == nil || !strings.Contains(err.Error(), "malformed frontmatter") {
		t.Fatalf("blocking findings lost their exit floor: %v", err)
	}
}
