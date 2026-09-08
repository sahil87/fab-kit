package memoryindex

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

func docWrite(t *testing.T, repo, path, content string) {
	t.Helper()
	p := filepath.Join(repo, path)
	if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}
func docGather(t *testing.T, repo string, c config.DocsIndexRoot) ([]Target, []Warning) {
	t.Helper()
	targets, w, err := GatherRoot(repo, filepath.Join(repo, "fab"), c, false)
	if err != nil {
		t.Fatal(err)
	}
	return targets, w
}
func docContent(t *testing.T, targets []Target, suffix string) string {
	t.Helper()
	for _, x := range targets {
		if strings.HasSuffix(filepath.ToSlash(x.Path), suffix) {
			return x.Content
		}
	}
	t.Fatalf("missing target %s", suffix)
	return ""
}
func docApply(t *testing.T, targets []Target) {
	t.Helper()
	for _, x := range targets {
		if err := os.WriteFile(x.Path, []byte(x.Content), 0644); err != nil {
			t.Fatal(err)
		}
	}
}
func docIdempotent(t *testing.T, repo string, c config.DocsIndexRoot, first []Target) {
	t.Helper()
	docApply(t, first)
	second, _ := docGather(t, repo, c)
	if len(first) != len(second) {
		t.Fatalf("target counts %d != %d", len(first), len(second))
	}
	for i := range first {
		if first[i] != second[i] {
			t.Errorf("second generation changed %s\nfirst:\n%s\nsecond:\n%s", first[i].Path, first[i].Content, second[i].Content)
		}
	}
}
func specsRoot() config.DocsIndexRoot {
	return config.DocsIndexRoot{Path: "docs/specs", IndexFile: "index.md", MaxDepth: 3}
}

func TestDocsIndexDeepSparseTree(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	docWrite(t, repo, "docs/specs/a/b/c/d/e/f/topic.md", "# Deep topic\n")
	targets, w := docGather(t, repo, c)
	if len(targets) != 7 {
		t.Fatalf("want 7 landing files, got %d", len(targets))
	}
	body := docContent(t, targets, "a/b/c/d/e/f/index.md")
	if !strings.Contains(body, "[Deep topic](topic.md) | —") {
		t.Fatal(body)
	}
	kinds := map[string]bool{}
	for _, x := range w {
		if x.IsBlocking() {
			t.Fatalf("advisory became blocking: %+v", x)
		}
		kinds[x.Kind] = true
	}
	if !kinds[KindDepth] || !kinds[KindMissingDescription] {
		t.Fatalf("missing warnings: %+v", w)
	}
	for _, x := range targets {
		if strings.Contains(x.Content, "fkf_version") || x.IsLog {
			t.Fatal("generic root acquired FKF output")
		}
	}
	docIdempotent(t, repo, c, targets)
}
func TestDocsIndexReadmeLanding(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.AlsoAccept = []string{"README.md"}
	before := "# Design\n\nHuman introduction.\n\n## Notes\n\nKeep this exactly.\n"
	docWrite(t, repo, "docs/specs/README.md", before)
	docWrite(t, repo, "docs/specs/topic.md", "# Topic\n")
	targets, _ := docGather(t, repo, c)
	body := docContent(t, targets, "README.md")
	if !strings.HasPrefix(body, before) || strings.Count(body, GeneratedStart) != 1 {
		t.Fatal(body)
	}
	if len(targets) != 1 {
		t.Fatalf("unexpected neighboring landing: %+v", targets)
	}
	docIdempotent(t, repo, c, targets)
	docWrite(t, repo, "docs/specs/README.md", "prefix\n"+GeneratedStart+"\nstale\n"+GeneratedEnd+"\nsuffix unchanged")
	targets, _ = docGather(t, repo, c)
	body = docContent(t, targets, "README.md")
	if !strings.HasPrefix(body, "prefix\n") || !strings.HasSuffix(body, "\nsuffix unchanged") {
		t.Fatal(body)
	}
}
func TestDocsIndexSuperseded(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.Superseded = []string{"**/archive/**", `**/\[archived\]-*`, "**/Z*/**"}
	docWrite(t, repo, "docs/specs/current/v5/live.md", "---\ndescription: Live design\n---\n# Live\n")
	for i, n := range []int{5, 12, 8, 8} {
		for j := 0; j < n; j++ {
			docWrite(t, repo, fmt.Sprintf("docs/specs/current/archive/v%d/file%d.md", i+1, j), "---\ndescription: \"BROKEN NEVER READ\n# Historical\n")
		}
	}
	docWrite(t, repo, "docs/specs/current/[archived]-old.md", "# Do not list\n")
	docWrite(t, repo, "docs/specs/Zlegacy/old.md", "# Do not list\n")
	targets, w := docGather(t, repo, c)
	body := docContent(t, targets, "current/index.md")
	if !strings.Contains(body, "[archive](archive/index.md) | 4 superseded versions (v1–v4), 33 files") {
		t.Fatal(body)
	}
	if strings.Contains(body, "archived]-old") {
		t.Fatal(body)
	}
	root := docContent(t, targets, "specs/index.md")
	if !strings.Contains(root, "1 superseded files") || !strings.Contains(root, "Superseded — 1 files") {
		t.Fatal(root)
	}
	archive := docContent(t, targets, "archive/index.md")
	if !strings.Contains(archive, "[v2/](v2/) | Superseded — 12 files") {
		t.Fatal(archive)
	}
	for _, x := range targets {
		if strings.Contains(x.Content, "file0") || strings.Contains(x.Path, "archive/v") {
			t.Fatalf("enumerated superseded files: %s", x.Path)
		}
	}
	for _, x := range w {
		if strings.Contains(x.Path, "archive/") {
			t.Fatalf("read historical content: %+v", x)
		}
	}
	docIdempotent(t, repo, c, targets)
}
func TestDocsIndexAdoption(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	old := "# Curated specs\n\nHuman purpose.\n\n### Apps\n\n| Spec | Description |\n|---|---|\n| [App](app.md) | Curated routing |\n| [Removed](gone.md) | Historical note |\n"
	docWrite(t, repo, "docs/specs/index.md", old)
	docWrite(t, repo, "docs/specs/app.md", "# Application\n")
	targets, _ := docGather(t, repo, c)
	body := docContent(t, targets, "specs/index.md")
	for _, want := range []string{"Curated routing", "Historical note", "### Apps", "Human purpose."} {
		if !strings.Contains(body, want) {
			t.Fatalf("lost %s: %s", want, body)
		}
	}
	report := Classify([]CheckTarget{{Path: "docs/specs/index.md", Existing: old, Rendered: body, IsRoot: true}}, func(p string) bool { _, e := os.Stat(filepath.Join(repo, "docs/specs", p)); return e == nil })
	if report.Tier != TierBenignDrift {
		t.Fatalf("adoption should be benign: %+v", report)
	}
	docIdempotent(t, repo, c, targets)
	if err := os.Remove(filepath.Join(repo, "docs/specs/app.md")); err != nil {
		t.Fatal(err)
	}
	next, _ := docGather(t, repo, c)
	report = Classify([]CheckTarget{{Path: "docs/specs/index.md", Existing: body, Rendered: docContent(t, next, "specs/index.md"), IsRoot: true}}, func(string) bool { return false })
	if report.Tier != TierDestructiveLoss {
		t.Fatalf("later live deletion must trip tier 2: %+v", report)
	}
}
func TestDocsIndexGenericVersusFKFWarnings(t *testing.T) {
	for _, fkf := range []bool{false, true} {
		t.Run(fmt.Sprint(fkf), func(t *testing.T) {
			repo := t.TempDir()
			c := specsRoot()
			c.Log = fkf
			docWrite(t, repo, "docs/specs/topic.md", "---\ndescription: "+strings.Repeat("x", 1001)+"\n---\n# Topic\n")
			_, w := docGather(t, repo, c)
			blocked := false
			for _, x := range w {
				blocked = blocked || x.IsBlocking()
			}
			if blocked != fkf {
				t.Fatalf("fkf=%v warnings=%+v", fkf, w)
			}
		})
	}
}
func TestDocsIndexZeroConfigGoldenCompatibility(t *testing.T) {
	repo := t.TempDir()
	c := config.DefaultDocsIndexRoots()[0]
	docWrite(t, repo, "docs/memory/auth/login.md", "---\ndescription: Login | access\n---\n# Login\n")
	docWrite(t, repo, "docs/memory/auth/index.md", "---\ndescription: Authentication\n---\n# Auth Documentation\n")
	docWrite(t, repo, "docs/memory/auth/login-alt.md", "---\ndescription: Alternate\n---\n# Alt\n")
	docWrite(t, repo, "docs/memory/auth/sparse.md", "# Human title\n")
	root, domains, _, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	targets, _ := docGather(t, repo, c)
	if got := docContent(t, targets, "memory/index.md"); got != RenderRoot(root) {
		t.Fatalf("memory root changed: %s", got)
	}
	if got := docContent(t, targets, "auth/index.md"); got != RenderDomain(domains[0]) {
		t.Fatalf("memory domain changed: %s", got)
	}
	if body := docContent(t, targets, "auth/index.md"); !strings.Contains(body, "[sparse](sparse.md) | —") {
		t.Fatal("zero-config memory must retain legacy filename label", body)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexAdoptionPreservesLiteralRowsAndNestedLinks(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	row := "| [Nested](area/topic.md) | A `stage\\|role` value and [context](elsewhere.md) | Extra metadata |"
	docWrite(t, repo, "docs/specs/index.md", "# Specs\n\n| Spec | Description | More |\n|---|---|---|\n"+row+"\n")
	docWrite(t, repo, "docs/specs/area/topic.md", "# Nested\n")
	targets, _ := docGather(t, repo, c)
	if body := docContent(t, targets, "specs/index.md"); !strings.Contains(body, row) {
		t.Fatal(body)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexAcceptedSupersededLandingPreservesProse(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.AlsoAccept = []string{"README.md"}
	c.Superseded = []string{"**/archive/**"}
	prose := "# Historical\n\nCurated historical prose.\n"
	docWrite(t, repo, "docs/specs/archive/README.md", prose)
	docWrite(t, repo, "docs/specs/archive/v1/topic.md", "---\ndescription: \"must not read\n")
	targets, w := docGather(t, repo, c)
	if body := docContent(t, targets, "archive/README.md"); !strings.HasPrefix(body, prose) {
		t.Fatal(body)
	}
	if len(w) != 0 {
		t.Fatal(w)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexGenericLogRootMetadataAndSeed(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.Log = true
	docWrite(t, repo, "docs/specs/index.md", "---\ndescription: Root routing\n---\n# Specs\n")
	docWrite(t, repo, "docs/specs/area/topic.md", "# Topic\n")
	docWrite(t, repo, "docs/specs/area/log.seed.md", "# Seed\n\n## 2026-01-01\n- **Update** [topic](/area/topic.md) — old history (aaaa)\n")
	targets, _ := docGather(t, repo, c)
	root := docContent(t, targets, "specs/index.md")
	if strings.Count(root, "---\n") != 2 || !strings.Contains(root, "fkf_version:") {
		t.Fatal(root)
	}
	if log := docContent(t, targets, "area/log.md"); !strings.Contains(log, "old history") {
		t.Fatal(log)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexAdoptionUpdatesEscapedPipeCells(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	docWrite(t, repo, "docs/specs/index.md", "# Specs\n\n| Topic | Description | Owner |\n| --- | --- | --- |\n| [A\\|B](topic.md) | Old\\|description | Team\\|One |\n")
	docWrite(t, repo, "docs/specs/topic.md", "---\ndescription: New | description\n---\n# Topic\n")
	targets, _ := docGather(t, repo, c)
	body := docContent(t, targets, "specs/index.md")
	if !strings.Contains(body, `| [A\|B](topic.md) | New \| description | Team\|One |`) {
		t.Fatal(body)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexLogUsesConfiguredLandings(t *testing.T) {
	repo := t.TempDir()
	folder := filepath.Join(repo, "docs/reference/area")
	docWrite(t, repo, "docs/reference/area/index.md", "# Ordinary topic\n")
	docWrite(t, repo, "docs/reference/area/nav.md", "# Landing\n")
	dates := &gitDates{top: repo, commitsByPath: map[string][]gitTouch{
		"docs/reference/area/index.md": {{Date: "2026-09-08", Subject: "Topic added", Status: "A"}},
		"docs/reference/area/nav.md":   {{Date: "2026-09-08", Subject: "Landing added", Status: "A"}},
	}}
	entries := gatherLogEntries(repo, dates, nil, folder, "area", true, "nav.md")
	if len(entries) != 1 || entries[0].FileBase != "index" {
		t.Fatalf("ordinary index.md topic missing from log: %+v", entries)
	}
}

func TestDocsIndexBundleLinkWarningsRequireFKF(t *testing.T) {
	for _, fkf := range []bool{false, true} {
		t.Run(fmt.Sprintf("log=%t", fkf), func(t *testing.T) {
			repo := t.TempDir()
			c := specsRoot()
			c.Log = fkf
			// Generic roots can contain site-root links; FKF alone interprets
			// these as bundle-relative paths under the configured docs root.
			content := "---\ndescription: " + strings.Repeat("x", 600) + "\n---\n# Topic\n\n[Guide](/guides/page.md)\n" + strings.Repeat("Previously renamed.\n", FileSizeLineWarnThreshold)
			docWrite(t, repo, "docs/specs/topic.md", content)
			_, warnings := docGather(t, repo, c)
			kinds := map[string]int{}
			for _, w := range warnings {
				kinds[w.Kind]++
				if w.Kind == KindBrokenLink && w.Detail != "/guides/page.md" {
					t.Fatalf("unexpected link diagnostic: %+v", w)
				}
			}
			for _, kind := range []string{KindBrokenLink, KindNarrationDensity} {
				want := 0
				if fkf {
					want = 1
				}
				if kinds[kind] != want {
					t.Errorf("%s count = %d, want %d", kind, kinds[kind], want)
				}
			}
			for _, kind := range []string{KindFileSize, KindDescriptionLength} {
				if kinds[kind] != 1 {
					t.Errorf("generic diagnostic %s was lost: %+v", kind, warnings)
				}
			}
		})
	}
}
