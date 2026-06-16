package memoryindex

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- parseSeedLog: the inverse of RenderLog (FKF §6.2) ---------------------

// TestParseSeedLog_RoundTripsRenderLog pins the parse∘render identity: parsing
// RenderLog's own output back yields the same LogEntry set, so a generated log.md
// fed back as a seed is a fixed point (the basis of idempotent preservation).
func TestParseSeedLog_RoundTripsRenderLog(t *testing.T) {
	entries := []LogEntry{
		{Date: "2026-06-13", Verb: verbUpdate, FileBase: "migrations", BundleRelPath: "/distribution/migrations.md",
			Summary: "surfaces the optional agent.tiers override as a commented config block", ChangeID: "l3ja"},
		{Date: "2026-06-12", Verb: verbCreation, FileBase: "setup", BundleRelPath: "/distribution/setup.md",
			Summary: "adds the setup doc", ChangeID: "tb6f"},
		{Date: "2026-06-12", Verb: "", FileBase: "orphan", BundleRelPath: "/distribution/orphan.md",
			Summary: "", ChangeID: ""}, // no verb / no id / empty summary → renders "—"
	}
	rendered := RenderLog(LogData{Title: "Distribution", Entries: entries})
	got := parseSeedLog(rendered)
	if len(got) != len(entries) {
		t.Fatalf("round-trip count mismatch: parsed %d, want %d\nrendered:\n%s", len(got), len(entries), rendered)
	}
	// RenderLog sorts newest-first; build a by-key map to compare set-wise.
	want := map[string]LogEntry{}
	for _, e := range entries {
		want[e.Date+"|"+e.FileBase] = e
	}
	for _, e := range got {
		w, ok := want[e.Date+"|"+e.FileBase]
		if !ok {
			t.Fatalf("parsed an entry with no counterpart: %+v", e)
		}
		if e != w {
			t.Errorf("round-trip entry mismatch:\n got  %+v\n want %+v", e, w)
		}
	}
}

// TestParseSeedLog_PreservesSeedDateAndLinkInSummary covers the real conversion
// shape: a seed entry under a pre-FKF authored date whose summary itself contains
// a bundle-relative link and bold markup. The whole summary is preserved verbatim.
func TestParseSeedLog_PreservesSeedDateAndLinkInSummary(t *testing.T) {
	seed := "# Log — Memory Docs\n" +
		"<!-- Generated ... -->\n" +
		"\n" +
		"## 2026-02-09\n" +
		"- **Update** [hydrate-specs](/memory-docs/hydrate-specs.md) — Initial creation — see [templates](/memory-docs/templates.md) for the shape (h3v7)\n"
	got := parseSeedLog(seed)
	if len(got) != 1 {
		t.Fatalf("expected 1 seed entry, got %d: %+v", len(got), got)
	}
	e := got[0]
	if e.Date != "2026-02-09" {
		t.Errorf("seed date heading not preserved: got %q", e.Date)
	}
	if e.FileBase != "hydrate-specs" || e.BundleRelPath != "/memory-docs/hydrate-specs.md" {
		t.Errorf("link cell mis-parsed: base=%q path=%q", e.FileBase, e.BundleRelPath)
	}
	if e.ChangeID != "h3v7" {
		t.Errorf("trailing (id) not peeled: got %q", e.ChangeID)
	}
	wantSummary := "Initial creation — see [templates](/memory-docs/templates.md) for the shape"
	if e.Summary != wantSummary {
		t.Errorf("summary not preserved verbatim:\n got  %q\n want %q", e.Summary, wantSummary)
	}
	if e.Verb != verbUpdate {
		t.Errorf("verb mis-parsed: got %q", e.Verb)
	}
}

// TestSplitTrailingID_NotMisledByInProseParens guards that a PR ref or a prose
// aside in parens is not mistaken for the (change-id) token.
func TestSplitTrailingID_NotMisledByInProseParens(t *testing.T) {
	cases := []struct{ desc, wantID, wantSummary string }{
		{"a normal line (d9rs)", "d9rs", "a normal line"},
		{"fixed the thing (#420)", "", "fixed the thing (#420)"}, // a `#NNN` PR ref is NOT a change-id (spec §6.2; matches attributeCommit)
		{"a line with (multi word) aside", "", "a line with (multi word) aside"},
		{"no parens at all", "", "no parens at all"},
		{missingCell, "", ""},
	}
	for _, c := range cases {
		id, summary := splitTrailingID(c.desc)
		if id != c.wantID || summary != c.wantSummary {
			t.Errorf("splitTrailingID(%q) = (%q,%q), want (%q,%q)", c.desc, id, summary, c.wantID, c.wantSummary)
		}
	}
}

// --- mergeSeedEntries: union + de-dup -------------------------------------

func TestMergeSeedEntries_DedupsExactMatch(t *testing.T) {
	shared := LogEntry{Date: "2026-06-12", Verb: verbUpdate, FileBase: "a", BundleRelPath: "/d/a.md",
		Summary: "same", ChangeID: "aaaa"}
	projected := []LogEntry{shared}
	seed := []LogEntry{
		shared, // byte-equal → must be de-duplicated
		{Date: "2026-02-01", Verb: verbCreation, FileBase: "a", BundleRelPath: "/d/a.md", Summary: "old history", ChangeID: "bbbb"},
	}
	got := mergeSeedEntries(projected, seed)
	if len(got) != 2 {
		t.Fatalf("expected 2 unique entries (1 deduped), got %d: %+v", len(got), got)
	}
	// The shared entry must appear exactly once.
	count := 0
	for _, e := range got {
		if e == shared {
			count++
		}
	}
	if count != 1 {
		t.Errorf("the byte-equal entry must appear once, appeared %d times", count)
	}
}

func TestMergeSeedEntries_KeepsProjectedAheadOfSeed(t *testing.T) {
	projected := []LogEntry{{Date: "2026-06-12", FileBase: "p", BundleRelPath: "/d/p.md", Summary: "proj"}}
	seed := []LogEntry{{Date: "2026-06-12", FileBase: "s", BundleRelPath: "/d/s.md", Summary: "seed"}}
	got := mergeSeedEntries(projected, seed)
	if len(got) != 2 || got[0].Summary != "proj" || got[1].Summary != "seed" {
		t.Errorf("projected must precede seed in slice order, got %+v", got)
	}
}

// --- GatherLogs with a seed file: full merge + idempotence -----------------

// TestGatherLogs_SeedMerge exercises the cutover crux end-to-end: a folder with a
// committed topic file (git-projected entry) AND a log.seed.md carrying pre-FKF
// history under its own authored dates. The generated log.md must merge both, and
// a second pass must be byte-identical (idempotent — no doubled seed entries).
func TestGatherLogs_SeedMerge(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	writeFile(t, repo, "fab/changes/260601-aaaa-recent/.status.yaml",
		"id: aaaa\nname: 260601-aaaa-recent\nsummary: \"the recent change\"\n")
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	// Seed: pre-FKF history with an authored date well before any git commit.
	writeFile(t, repo, "docs/memory/d/log.seed.md",
		"# Log — D\n<!-- seed -->\n\n## 2026-02-09\n- **Creation** [topic](/d/topic.md) — initial pre-FKF creation (h3v7)\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "Merge pull request #1 from o/260601-aaaa-recent",
		"--date", "2026-06-01T12:00:00 +0000")

	fabRoot := filepath.Join(repo, "fab")
	targets, err := GatherLogs(repo, fabRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	var dLog *LogTarget
	for i := range targets {
		if strings.HasSuffix(filepath.ToSlash(targets[i].Path), "docs/memory/d/log.md") {
			dLog = &targets[i]
		}
	}
	if dLog == nil {
		t.Fatalf("expected a d/log.md target, got %d: %+v", len(targets), targets)
	}
	content := dLog.Content
	// Git-projected (recent) entry present.
	if !strings.Contains(content, "## 2026-06-01") || !strings.Contains(content, "the recent change (aaaa)") {
		t.Errorf("git-projected entry missing:\n%s", content)
	}
	// Seed (pre-FKF) entry preserved under its OWN date.
	if !strings.Contains(content, "## 2026-02-09") ||
		!strings.Contains(content, "- **Creation** [topic](/d/topic.md) — initial pre-FKF creation (h3v7)\n") {
		t.Errorf("seed entry not preserved under its authored date:\n%s", content)
	}
	// Newest date first: 2026-06-01 must precede 2026-02-09.
	if strings.Index(content, "## 2026-06-01") > strings.Index(content, "## 2026-02-09") {
		t.Errorf("dates not newest-first:\n%s", content)
	}

	// Idempotence: write log.md, re-gather, assert byte-identical (no dupes).
	logPath := dLog.Path
	if err := os.WriteFile(logPath, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := GatherLogs(repo, fabRoot, false)
	if err != nil {
		t.Fatal(err)
	}
	for _, tg := range second {
		if tg.Path == logPath && tg.Content != content {
			t.Errorf("NOT idempotent: second seed-merge differs.\n--- first ---\n%s\n--- second ---\n%s", content, tg.Content)
		}
	}
}

// TestGatherLogs_SeedOnlyFolderEmitsLog covers a folder whose topic file has NO
// attributable git history but DOES carry a log.seed.md: it must still emit a
// log.md (the pre-FKF history is git-independent). Before the seed-merge a folder
// with no git commits was skipped entirely.
func TestGatherLogs_SeedOnlyFolderEmitsLog(t *testing.T) {
	repo := t.TempDir()
	gitDateRun(t, repo, "init")
	// Commit one folder so the batched git pass succeeds (dates != nil).
	writeFile(t, repo, "docs/memory/committed/a.md", "# A\n")
	gitDateRun(t, repo, "add", ".")
	gitDateRun(t, repo, "commit", "-m", "init", "--date", "2026-01-01T12:00:00 +0000")
	// Seed-only folder: present on disk, NOT committed, but carries a seed.
	writeFile(t, repo, "docs/memory/seeded/b.md", "# B\n")
	writeFile(t, repo, "docs/memory/seeded/log.seed.md",
		"## 2026-03-03\n- **Update** [b](/seeded/b.md) — seeded history (zzzz)\n")

	targets, err := GatherLogs(repo, filepath.Join(repo, "fab"), false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, tg := range targets {
		if strings.HasSuffix(filepath.ToSlash(tg.Path), "docs/memory/seeded/log.md") {
			found = true
			if !strings.Contains(tg.Content, "seeded history (zzzz)") {
				t.Errorf("seed-only log.md missing its seed entry:\n%s", tg.Content)
			}
		}
	}
	if !found {
		t.Errorf("a seed-only folder must emit a log.md, targets:\n%+v", targets)
	}
}

// TestGather_ExcludesSeedFile pins that log.seed.md is never gathered as a topic
// file (no [log.seed] row in the index) — the same single-writer exclusion as
// index.md / log.md.
func TestGather_ExcludesSeedFile(t *testing.T) {
	repo := t.TempDir()
	writeFile(t, repo, "docs/memory/d/topic.md", "# Topic\n")
	writeFile(t, repo, "docs/memory/d/log.seed.md", "## 2026-01-01\n- [topic](/d/topic.md) — x (aaaa)\n")
	_, domains, _, err := Gather(repo)
	if err != nil {
		t.Fatal(err)
	}
	if len(domains) != 1 {
		t.Fatalf("expected one domain, got %d", len(domains))
	}
	for _, f := range domains[0].Files {
		if f.Base == "log.seed" || f.Base == "log" {
			t.Errorf("log.seed.md must not be gathered as a topic file, got %q", f.Base)
		}
	}
	if len(domains[0].Files) != 1 || domains[0].Files[0].Base != "topic" {
		t.Errorf("expected only the real topic file, got %+v", domains[0].Files)
	}
}
