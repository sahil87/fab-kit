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
func TestValidateGlobsNamesField(t *testing.T) {
	for _, tc := range []struct{ field, pattern, want string }{
		{"exclude", "/abs/**", "docs_index.roots.exclude"},
		{"exclude", "../up", "docs_index.roots.exclude"},
		{"exclude", "[bad", "docs_index.roots.exclude"},
		{"superseded", "/abs/**", "docs_index.roots.superseded"},
	} {
		err := validateGlobs(tc.field, []string{tc.pattern})
		if err == nil || !strings.Contains(err.Error(), tc.want) {
			t.Errorf("validateGlobs(%q, %q) = %v, want error naming %s", tc.field, tc.pattern, err, tc.want)
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
	// The walker's primary landings are the pure renderers' output plus the
	// always-present manual block (seeded on creation).
	if got := docContent(t, targets, "memory/index.md"); got != withManualBlock(RenderRoot(root), manualBlockSeed) {
		t.Fatalf("memory root changed: %s", got)
	}
	if got := docContent(t, targets, "auth/index.md"); got != withManualBlock(RenderDomain(domains[0]), manualBlockSeed) {
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

// TestDocsIndexConvergesWithQuotedDescriptions is the escape-runaway
// regression test: folder-index `description:` values containing escaped
// double quotes must round-trip un-doubled, and a second consecutive run over
// the unchanged tree must be a byte-for-byte no-op. The docIdempotent helper
// and TestMemoryIndex_GenerateThenGenerateIsByteStable already cover
// double-generation over benign descriptions; this fixture adds the
// adversarial quoted shapes (the intake's loom repro) that escaped them.
func TestDocsIndexConvergesWithQuotedDescriptions(t *testing.T) {
	repo := t.TempDir()
	c := config.DefaultDocsIndexRoots()[0]
	docWrite(t, repo, "docs/memory/demo/index.md", "---\ndescription: \"Demo domain\"\n---\n# Demo\n")
	docWrite(t, repo, "docs/memory/demo/topic.md", "# Topic\nbody\n")
	// The intake's reproduction shape: a folder index whose description
	// carries Go-quoted escaped double quotes.
	docWrite(t, repo, "docs/memory/demo/sub/index.md", "---\ndescription: \"Sub with an escaped \\\"quoted\\\" phrase\"\n---\n# Sub\n")
	docWrite(t, repo, "docs/memory/demo/sub/note.md", "---\ndescription: \"A <Button variant=\\\"ghost\\/> mention\"\n---\n# Note\n")

	first, _ := docGather(t, repo, c)
	// The round trip must not double the escapes: the rewritten folder index
	// keeps the seeded description line byte-for-byte.
	sub := docContent(t, first, "sub/index.md")
	if !strings.Contains(sub, "description: \"Sub with an escaped \\\"quoted\\\" phrase\"") {
		t.Fatalf("escaped quotes were not preserved:\n%s", sub)
	}
	docApply(t, first)

	// Second run over the unchanged tree: every target is byte-identical to
	// what is on disk (nothing to write — the cmd would report no updates).
	second, _ := docGather(t, repo, c)
	for _, tg := range second {
		onDisk, err := os.ReadFile(tg.Path)
		if err != nil {
			t.Fatalf("second-pass target %s missing from disk: %v", tg.Path, err)
		}
		if string(onDisk) != tg.Content {
			t.Errorf("NOT convergent: second generate of %s differs.\n--- on disk ---\n%s\n--- re-rendered ---\n%s",
				tg.Path, onDisk, tg.Content)
		}
	}
}

// TestDocsIndexNavNoteGenericRoot covers nav_note on a generic root landing:
// set inserts the note verbatim right after the "Generated by" note (and the
// result converges on regeneration); empty (or unset) renders nothing.
func TestDocsIndexNavNoteGenericRoot(t *testing.T) {
	note := "> **New here?** See the [Glossary](../glossary.md)."

	t.Run("set", func(t *testing.T) {
		repo := t.TempDir()
		c := specsRoot()
		c.NavNote = note
		docWrite(t, repo, "docs/specs/area/topic.md", "---\ndescription: A topic\n---\n# Topic\n")
		targets, _ := docGather(t, repo, c)
		root := docContent(t, targets, "specs/index.md")
		anchor := "is preserved.\n\n"
		i := strings.Index(root, anchor)
		if i < 0 || !strings.HasPrefix(root[i+len(anchor):], note+"\n\n") {
			t.Fatalf("nav_note must follow the Generated-by note:\n%s", root)
		}
		docIdempotent(t, repo, c, targets)
	})

	t.Run("empty", func(t *testing.T) {
		repo := t.TempDir()
		c := specsRoot()
		docWrite(t, repo, "docs/specs/area/topic.md", "# Topic\n")
		targets, _ := docGather(t, repo, c)
		if root := docContent(t, targets, "specs/index.md"); strings.Contains(root, "New here?") {
			t.Fatalf("empty nav_note must render nothing on a generic root:\n%s", root)
		}
	})
}

// TestDocsIndexNavNotePipeSequence covers a nav_note containing an inline
// "| " sequence: the manual block must land after the whole note, immediately
// before the first table — never mid-note.
func TestDocsIndexNavNotePipeSequence(t *testing.T) {
	note := "> Compare A | B in the table below, or use | inline."

	repo := t.TempDir()
	c := specsRoot()
	c.NavNote = note
	docWrite(t, repo, "docs/specs/area/topic.md", "---\ndescription: A topic\n---\n# Topic\n")
	targets, _ := docGather(t, repo, c)
	root := docContent(t, targets, "specs/index.md")
	anchor := note + "\n\n"
	i := strings.Index(root, anchor)
	if i < 0 || !strings.HasPrefix(root[i+len(anchor):], manualStart) {
		t.Fatalf("manual block must follow the full nav_note:\n%s", root)
	}
	docIdempotent(t, repo, c, targets)
}

// TestDocsIndexNavNoteLegacyRoot covers nav_note on the legacy memory root
// through the full gather: set renders the note verbatim, empty (or unset)
// renders no nav line, and both converge on regeneration.
func TestDocsIndexNavNoteLegacyRoot(t *testing.T) {
	note := "> **New here?** Start with the [README](../../README.md). For terminology, see the [Glossary](../glossary.md)."

	t.Run("set", func(t *testing.T) {
		repo := t.TempDir()
		c := config.DefaultDocsIndexRoots()[0]
		c.NavNote = note
		docWrite(t, repo, "docs/memory/auth/login.md", "---\ndescription: Login flow\n---\n# Login\n")
		targets, _ := docGather(t, repo, c)
		root := docContent(t, targets, "memory/index.md")
		if !strings.Contains(root, note) || strings.Contains(root, "../specs/glossary.md") {
			t.Fatalf("set nav_note must render verbatim:\n%s", root)
		}
		docIdempotent(t, repo, c, targets)
	})

	t.Run("empty", func(t *testing.T) {
		repo := t.TempDir()
		c := config.DefaultDocsIndexRoots()[0]
		docWrite(t, repo, "docs/memory/auth/login.md", "# Login\n")
		targets, _ := docGather(t, repo, c)
		root := docContent(t, targets, "memory/index.md")
		if strings.Contains(root, "New here?") || strings.Contains(root, "glossary") {
			t.Fatalf("empty nav_note must render no nav line:\n%s", root)
		}
		docIdempotent(t, repo, c, targets)
	})
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

// --- All file types are topics (R1/R4/R6/R7) --------------------------------

func TestDocsIndexAllFileTypesAreTopics(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	docWrite(t, repo, "docs/specs/a.md", "---\ndescription: Alpha\n---\n# A\n")
	docWrite(t, repo, "docs/specs/b.html", "<html><head><title>Bee</title><meta name=\"description\" content=\"Bee page\"></head><body></body></html>")
	docWrite(t, repo, "docs/specs/c.png", "not really a png")
	docWrite(t, repo, "docs/specs/.hidden", "x")
	targets, w := docGather(t, repo, c)
	body := docContent(t, targets, "specs/index.md")
	for _, want := range []string{"| [a](a.md) | Alpha |", "| [Bee](b.html) | Bee page |", "| [c.png](c.png) | — |"} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s:\n%s", want, body)
		}
	}
	if strings.Contains(body, "hidden") {
		t.Fatalf("dotfile must not be a row:\n%s", body)
	}
	for _, x := range w {
		if x.Kind == KindMissingDescription && !strings.HasSuffix(x.Path, ".md") && !strings.HasSuffix(x.Path, ".html") {
			t.Fatalf("missing-description advisory fired for a type that cannot carry one: %+v", x)
		}
	}
	docIdempotent(t, repo, c, targets)
}

func TestHTMLHeadMeta(t *testing.T) {
	repo := t.TempDir()
	write := func(name, content string) string {
		t.Helper()
		p := filepath.Join(repo, name)
		if err := os.WriteFile(p, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
		return p
	}
	bigStyle := "<style>" + strings.Repeat("x", 60*1024) + "</style>"
	for _, tc := range []struct {
		name, content, wantTitle, wantDesc string
	}{
		{"entities", "<head><title>Left &amp; Right</title><meta content=\"Panel rewrite\" name=\"description\"></head>", "Left & Right", "Panel rewrite"},
		{"single quotes", "<head><meta name='description' content='Single quoted'></head>", "", "Single quoted"},
		{"order swapped", "<head><meta content=\"Order swapped\" name=\"description\"></head>", "", "Order swapped"},
		{"uppercase", "<HEAD><TITLE>Up &amp; Away</TITLE><META NAME=\"description\" CONTENT=\"Caps\"></HEAD>", "Up & Away", "Caps"},
		{"whitespace collapsed", "<head><title>  A\n\t B  </title></head>", "A B", ""},
		{"title after big style", "<head>" + bigStyle + "<title>Late title</title></head>", "Late title", ""},
		{"title only in body", "<head></head><body><title>Nope</title></body>", "", ""},
		{"empty title", "<head><title></title></head>", "", ""},
		{"meta without description name", "<head><title>T</title><meta name=\"viewport\" content=\"width=1\"></head>", "T", ""},
		{"quoted gt in attribute", "<head><meta name=\"description\" content=\"a > b\"></head>", "", "a > b"},
		{"no head bound", "<title>Doc</title><p>hello</p>", "Doc", ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			title, desc := htmlHeadMeta(write(strings.ReplaceAll(tc.name, " ", "_")+".html", tc.content))
			if title != tc.wantTitle || desc != tc.wantDesc {
				t.Errorf("got (%q, %q), want (%q, %q)", title, desc, tc.wantTitle, tc.wantDesc)
			}
		})
	}
}

func TestDocsIndexHTMLRows(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	// A <title> is a label source only — never the description.
	docWrite(t, repo, "docs/specs/page.html", "<head><title>Left &amp; Right</title><meta content=\"Panel rewrite\" name=\"description\"></head>")
	docWrite(t, repo, "docs/specs/bare.html", "<head><title>Bare page</title></head>")
	docWrite(t, repo, "docs/specs/pipe.html", "<head><title>A | B</title></head>")
	docWrite(t, repo, "docs/specs/upper.HTM", "<head></head>")
	targets, w := docGather(t, repo, c)
	body := docContent(t, targets, "specs/index.md")
	for _, want := range []string{
		"| [Left & Right](page.html) | Panel rewrite |",
		"| [Bare page](bare.html) | — |", // title is never a description
		`| [A \| B](pipe.html) | — |`,    // table-cell escaping on an HTML label
		"| [upper.HTM](upper.HTM) | — |", // .HTM treated as HTML; empty/absent title falls back to the filename
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("missing %s:\n%s", want, body)
		}
	}
	htmlWarnings := 0
	for _, x := range w {
		if x.Kind != KindMissingDescription {
			continue
		}
		htmlWarnings++
		msg := x.String()
		if strings.Contains(msg, "H1") {
			t.Fatalf("HTML advisory must not claim H1: %s", msg)
		}
		if !strings.Contains(msg, "<meta name=\"description\">") {
			t.Fatalf("HTML advisory must name the meta tag: %s", msg)
		}
	}
	// bare.html, pipe.html and upper.HTM have no meta description; page.html does.
	if htmlWarnings != 3 {
		t.Fatalf("want 3 HTML missing-description advisories, got %d: %+v", htmlWarnings, w)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexNonMarkdownOnlyFolder(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	docWrite(t, repo, "docs/specs/only/page.html", "<head><title>Only</title></head>")
	targets, _ := docGather(t, repo, c)
	// A folder holding only non-markdown files now has count > 0: its own
	// generated landing and a Sub-Domains row whose description round-trips
	// from the landing frontmatter — starting at —, never synthesized.
	child := docContent(t, targets, "only/index.md")
	if !strings.Contains(child, "| [Only](page.html) | — |") {
		t.Fatal(child)
	}
	root := docContent(t, targets, "specs/index.md")
	if !strings.Contains(root, "| [only](only/index.md) | — |") {
		t.Fatal(root)
	}
	docIdempotent(t, repo, c, targets)
	second, _ := docGather(t, repo, c)
	if !strings.Contains(docContent(t, second, "specs/index.md"), "| [only](only/index.md) | — |") {
		t.Fatal("sub-domain description must stay — across regenerations (no synthesized text)")
	}
}

func TestDocsIndexExclude(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.Exclude = []string{"assets/**", "**/*.png"}
	docWrite(t, repo, "docs/specs/assets/cursio/x.png", "png")
	docWrite(t, repo, "docs/specs/guide/shot.png", "png")
	docWrite(t, repo, "docs/specs/guide/page.html", "<head><title>Guide</title><meta name=\"description\" content=\"Guide page\"></head>")
	targets, _ := docGather(t, repo, c)
	for _, x := range targets {
		if strings.Contains(filepath.ToSlash(x.Path), "assets/") {
			t.Fatalf("excluded folder produced output: %s", x.Path)
		}
	}
	root := docContent(t, targets, "specs/index.md")
	if strings.Contains(root, "assets") {
		t.Fatalf("excluded folder must not be a sub-domain row:\n%s", root)
	}
	guide := docContent(t, targets, "guide/index.md")
	if !strings.Contains(guide, "| [Guide](page.html) | Guide page |") || strings.Contains(guide, "shot.png") {
		t.Fatalf("excluded file must produce no row:\n%s", guide)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexExcludeBeatsSuperseded(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.Superseded = []string{"old/**"}
	c.Exclude = []string{"old/**"}
	docWrite(t, repo, "docs/specs/old/a.md", "# Old\n")
	docWrite(t, repo, "docs/specs/live/b.md", "# Live\n")
	targets, _ := docGather(t, repo, c)
	root := docContent(t, targets, "specs/index.md")
	if strings.Contains(root, "old") || strings.Contains(strings.ToLower(root), "superseded") {
		t.Fatalf("a path matching both exclude and superseded is never seen:\n%s", root)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexAllExcludedFolderDropped(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.Exclude = []string{"**/*.png"}
	docWrite(t, repo, "docs/specs/mixed/a.png", "png")
	docWrite(t, repo, "docs/specs/mixed/b.png", "png")
	docWrite(t, repo, "docs/specs/topic.md", "# Topic\n")
	targets, _ := docGather(t, repo, c)
	for _, x := range targets {
		if strings.Contains(filepath.ToSlash(x.Path), "mixed/") {
			t.Fatalf("excluded-to-empty folder must be dropped like an empty folder: %s", x.Path)
		}
	}
	if strings.Contains(docContent(t, targets, "specs/index.md"), "mixed") {
		t.Fatal("excluded-to-empty folder must not be a sub-domain row")
	}
}

func TestDocsIndexExcludeValidationNamesField(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.Exclude = []string{"/abs/**"}
	if _, _, err := GatherRoot(repo, filepath.Join(repo, "fab"), c, false); err == nil || !strings.Contains(err.Error(), "docs_index.roots.exclude") {
		t.Fatalf("want error naming docs_index.roots.exclude, got %v", err)
	}
}

func TestDocsIndexWidthCountsNonMarkdown(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	for i := 0; i < WidthWarnThreshold+1; i++ {
		docWrite(t, repo, fmt.Sprintf("docs/specs/wide/f%02d.png", i), "x")
	}
	_, w := docGather(t, repo, c)
	found := false
	for _, x := range w {
		if x.Kind == KindWidth && strings.HasSuffix(x.Path, "wide") && x.Count == WidthWarnThreshold+1 {
			found = true
		}
	}
	if !found {
		t.Fatalf("width advisory must count non-markdown topics: %+v", w)
	}
}

func TestDocsIndexFKFWarningsSkipNonMarkdown(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.Log = true
	docWrite(t, repo, "docs/specs/x.md", "# X\n")
	docWrite(t, repo, "docs/specs/y.pdf", "%PDF-1.7 fake")
	_, w := docGather(t, repo, c)
	for _, x := range w {
		if strings.HasSuffix(x.Path, ".pdf") {
			t.Fatalf("frontmatter/FKF machinery must not run on non-markdown topics: %+v", x)
		}
	}
	missingMD := false
	for _, x := range w {
		if x.Kind == KindMissingDescription && strings.HasSuffix(x.Path, "x.md") {
			missingMD = true
		}
	}
	if !missingMD {
		t.Fatalf("markdown advisory lost: %+v", w)
	}
}

// --- The manual block (R9/R10/R11/R12) ---------------------------------------

func TestDocsIndexLegacyCuratedBlockReadsAsManual(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	old := "# Specs\n\n> **Generated by `fab docs-index`** — do not hand-edit.\n\n" +
		legacyCuratedStart + "\n| [App](app.md) | Hand routing |\n| [Other](other.md) | Also hand |\n" + legacyCuratedEnd +
		"\n\n| File | Description |\n|---|---|\n| [app](app.md) | — |\n| [other](other.md) | — |\n"
	docWrite(t, repo, "docs/specs/index.md", old)
	docWrite(t, repo, "docs/specs/app.md", "# App\n")
	docWrite(t, repo, "docs/specs/other.md", "# Other\n")
	targets, _ := docGather(t, repo, c)
	body := docContent(t, targets, "specs/index.md")
	if strings.Contains(body, "docs-index:curated") {
		t.Fatalf("only the manual spelling is written:\n%s", body)
	}
	if !strings.Contains(body, manualStart) || !strings.Contains(body, "Hand routing") || !strings.Contains(body, "Also hand") {
		t.Fatalf("legacy block rows must survive the rename:\n%s", body)
	}
	// The rename alone is benign drift (tier 1), never tier 2.
	report := Classify([]CheckTarget{{Path: "docs/specs/index.md", Existing: old, Rendered: body, IsRoot: true}},
		func(p string) bool { _, e := os.Stat(filepath.Join(repo, "docs/specs", p)); return e == nil })
	if report.Tier != TierBenignDrift {
		t.Fatalf("rename-only drift must be tier 1: %+v", report)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexManualBlockAlwaysEmitted(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	c.AlsoAccept = []string{"README.md"}
	docWrite(t, repo, "docs/specs/README.md", "# Specs\n\nHuman prose.\n")
	docWrite(t, repo, "docs/specs/topic.md", "# Topic\n")
	docWrite(t, repo, "docs/specs/sub/note.md", "# Note\n")
	targets, _ := docGather(t, repo, c)
	// The alternate landing is already a hand-managed region outside its
	// generated block — no manual markers there.
	alt := docContent(t, targets, "specs/README.md")
	if strings.Contains(alt, manualStart) || strings.Contains(alt, "docs-index:manual") {
		t.Fatalf("alternate landings never carry the manual block:\n%s", alt)
	}
	// Primary landings (domain and sub-domain) carry the seeded block between
	// the header note and the first table.
	for _, suffix := range []string{"sub/index.md"} {
		body := docContent(t, targets, suffix)
		if !strings.Contains(body, manualStart) || !strings.Contains(body, manualBlockSeed) {
			t.Fatalf("primary landing must carry the seeded block:\n%s", body)
		}
		if strings.Index(body, "is preserved.\n") > strings.Index(body, manualStart) || strings.Index(body, manualEnd) > strings.Index(body, "| File |") {
			t.Fatalf("block must sit between the header note and the first table:\n%s", body)
		}
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexManualBlockLegacyMemoryRoot(t *testing.T) {
	repo := t.TempDir()
	c := config.DefaultDocsIndexRoots()[0]
	docWrite(t, repo, "docs/memory/auth/login.md", "# Login\n")
	targets, _ := docGather(t, repo, c)
	root := docContent(t, targets, "memory/index.md")
	if !strings.Contains(root, manualStart) || !strings.Contains(root, manualBlockSeed) {
		t.Fatalf("legacy memory root must carry the seeded block:\n%s", root)
	}
	// The Constitution VI sense of "human-curated" stays.
	if !strings.Contains(root, "human-curated") {
		t.Fatalf("the specs-are-human-curated preamble sentence stays:\n%s", root)
	}
	domain := docContent(t, targets, "auth/index.md")
	if !strings.Contains(domain, manualStart) || strings.Contains(domain, "curated") {
		t.Fatalf("domain landing must carry the block and drop block-sense 'curated':\n%s", domain)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexManualBlockContentPreservedVerbatim(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	block := manualStart + "\nCustom prose, no comment.\n\n| [Extra](https://example.com) | External |\n" + manualEnd
	docWrite(t, repo, "docs/specs/index.md", "# Specs\n\n"+block+"\n")
	docWrite(t, repo, "docs/specs/topic.md", "# Topic\n")
	targets, _ := docGather(t, repo, c)
	body := docContent(t, targets, "specs/index.md")
	if !strings.Contains(body, "Custom prose, no comment.") || !strings.Contains(body, "| [Extra](https://example.com) | External |") {
		t.Fatalf("existing block content must pass through verbatim:\n%s", body)
	}
	if strings.Contains(body, manualBlockSeed) {
		t.Fatalf("the comment is content, not scaffolding — never re-emitted into an existing block:\n%s", body)
	}
	docIdempotent(t, repo, c, targets)
}

func TestDocsIndexManualRowDescriptionFollowsSource(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	docWrite(t, repo, "docs/specs/index.md", "# S\n\n"+manualStart+"\n| [Arch](arch.html) | Hand-written |\n"+manualEnd+"\n")
	docWrite(t, repo, "docs/specs/arch.html", "<head><title>Arch</title></head>")
	targets, _ := docGather(t, repo, c)
	body := docContent(t, targets, "specs/index.md")
	// The generated description is —, so the hand-written one is kept and the
	// generated duplicate row is removed (the target appears once).
	if !strings.Contains(body, "| [Arch](arch.html) | Hand-written |") || strings.Count(body, "(arch.html)") != 1 {
		t.Fatalf("manual row keeps its hand-written description when the source has none:\n%s", body)
	}
	// A non-— generated description replaces the manual row's description.
	docWrite(t, repo, "docs/specs/arch.html", "<head><title>Arch</title><meta name=\"description\" content=\"From meta\"></head>")
	targets, _ = docGather(t, repo, c)
	body = docContent(t, targets, "specs/index.md")
	if !strings.Contains(body, "| [Arch](arch.html) | From meta |") {
		t.Fatalf("manual row description must follow a source that supplies one:\n%s", body)
	}
}

func TestDocsIndexHeaderProse(t *testing.T) {
	repo := t.TempDir()
	c := specsRoot()
	docWrite(t, repo, "docs/specs/area/topic.md", "# Topic\n")
	targets, _ := docGather(t, repo, c)
	root := docContent(t, targets, "specs/index.md")
	if !strings.Contains(root, "> **Generated by `fab docs-index`**") {
		t.Fatalf("the Generated-by prefix must survive (insertNavNote/supersededCleanup anchor):\n%s", root)
	}
	for _, want := range []string{"do not hand-edit", "only the manual block below is preserved"} {
		if !strings.Contains(root, want) {
			t.Fatalf("header must carry both halves of the one-line note (%s):\n%s", want, root)
		}
	}
	for _, gone := range []string{"Descriptions come from", "everything outside", "curated"} {
		if strings.Contains(root, gone) {
			t.Fatalf("stale header wording %q:\n%s", gone, root)
		}
	}
	docIdempotent(t, repo, c, targets)
}
