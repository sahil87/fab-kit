// Package memoryindex deterministically (re)generates the docs/memory index
// files: the root docs/memory/index.md (domains-only) and every
// docs/memory/{domain}/index.md (file rows). It is the deterministic
// counterpart to the hand-maintained index rows that previously lived in the
// hydrate / docs-reorg-memory skill prose — reading the same inputs (each
// memory file's H1 + `description:` frontmatter, plus `git log` dates) and
// emitting the exact same markdown on every run so the indexes stop drifting
// and stop generating merge conflicts on the hot per-row cells.
//
// Rendering is split into pure functions that take structured inputs and
// return markdown (RenderRoot / RenderDomain), plus a Gather orchestrator that
// performs the I/O (directory walk, file reads, git shelling). This mirrors
// internal/prmeta and keeps the byte-for-byte render contract unit-testable
// without git fixtures.
//
// It also walks the whole tree anyway, so it cheaply computes per-folder file
// counts and depth and returns non-fatal shape-bound warnings (the "C-detect"
// half of the memory-tree-shape work). Warnings never affect the rendered
// index output — they are advisory only, keeping the index byte-stable.
package memoryindex

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/frontmatter"
	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
	"github.com/sahil87/fab-kit/src/go/fab/internal/statusfile"
)

// Shape bounds. The upper width bound and max depth are enforced as non-fatal
// warnings; the lower floor (~5) and the ≥8-file sub-domain-cluster heuristic
// are SHOULD guidance documented in the skills, not warned on here (warning on
// too-few files would be noise).
const (
	// WidthWarnThreshold is the soft upper bound on topic files per folder.
	// A folder with strictly more than this triggers a width warning.
	WidthWarnThreshold = 12
	// MaxDepth is the maximum allowed nesting under docs/memory/ before a
	// depth warning fires (docs/memory/{domain}/{sub-domain}/{topic}.md = 3).
	MaxDepth = 3
)

// reservedDomains are exempt from the width warning: cross-cutting and staging
// buckets that are deliberately broad (loom convention).
var reservedDomains = map[string]bool{
	"_shared":   true,
	"_unsorted": true,
}

// FileEntry is one non-index .md file within a domain folder.
type FileEntry struct {
	// Base is the file name without the .md extension (the link target stem).
	Base string
	// Title is the H1 of the file (first `# ` line); unused in the rendered
	// row today but gathered for parity/diagnostics.
	Title string
	// Description is the `description:` frontmatter value; "" → rendered as the
	// missing-cell fallback.
	Description string
	// LastUpdated is the `git log -1 --date=short` date (YYYY-MM-DD); "" →
	// rendered as the missing-cell fallback.
	LastUpdated string
}

// DomainData is everything RenderDomain needs to render one domain index. It is
// a plain value so RenderDomain is a pure function of it. The same struct
// represents both a top-level domain and a sub-domain (a folder one level under
// a domain dir holding its own topic files) — the file-row contract is
// identical at either tier, so RenderDomain renders both.
type DomainData struct {
	// Name is the folder name (e.g. "fab-workflow" for a domain, "runtime" for
	// a sub-domain).
	Name string
	// Title is the human heading rendered at the top of the (sub-)domain index.
	Title string
	// Description is the curated one-liner for the parent index row; it is
	// round-tripped through the generated index.md's `description:` frontmatter
	// so it survives regeneration (the (sub-)domain index is the single home for
	// this fact — the parent index reads it back here).
	Description string
	// Files are the (sub-)domain's topic files, sorted lexicographically by Base.
	Files []FileEntry
	// SubDomains are the child sub-domain folders (one level down) that hold at
	// least one topic file, sorted lexicographically by Name. Empty for a
	// sub-domain (recursion is one level only — deeper nesting is a depth
	// warning, not a generated index tier) and for a flat domain.
	SubDomains []DomainData
}

// DomainRow is one row of the root (domains-only) index.
type DomainRow struct {
	Name        string // folder name; link target is {Name}/index.md
	Description string // curated one-liner; "" → missing-cell fallback
}

// RootData is everything RenderRoot needs. Plain value → RenderRoot is pure.
type RootData struct {
	Domains []DomainRow // sorted lexicographically by Name
}

// Warning is a non-fatal shape-bound finding. String renders the stderr line.
type Warning struct {
	Path  string // repo-relative folder/path the finding is about
	Kind  string // "width" | "depth"
	Count int    // file count (width) — 0 for depth
	Depth int    // observed depth (depth) — 0 for width
}

// String formats the advisory warning line written to stderr.
func (w Warning) String() string {
	switch w.Kind {
	case "width":
		return fmt.Sprintf("⚠ %s has %d topic files (soft bound: ~%d) — consider splitting into sub-domains",
			w.Path, w.Count, WidthWarnThreshold)
	case "depth":
		return fmt.Sprintf("⚠ %s exceeds depth %d — consider flattening", w.Path, MaxDepth)
	default:
		return fmt.Sprintf("⚠ %s", w.Path)
	}
}

// missingCell is the fallback rendered for an absent description or date,
// matching internal/prmeta's "—" convention for missing data.
const missingCell = "—"

// rootFrontmatter is the FKF version block RenderRoot prepends to the root
// docs/memory/index.md — the ONLY index.md permitted frontmatter beyond the
// generator's own output (FKF §8). No domain/sub-domain index carries it.
const rootFrontmatter = "---\nfkf_version: \"0.1\"\n---\n"

// RenderRoot assembles the complete root docs/memory/index.md markdown for d.
// It is pure: identical RootData always yields identical output. The table is
// domains-only — the legacy inlined per-file "Memory Files" column is dropped.
func RenderRoot(d RootData) string {
	var b strings.Builder
	b.WriteString(rootFrontmatter)
	b.WriteString("# Memory Index\n\n")
	b.WriteString("> **Memory files are post-implementation artifacts** — what actually *happened*. They are the\n")
	b.WriteString("> authoritative source of truth for system behavior and design decisions, maintained by\n")
	b.WriteString("> `/fab-continue` (hydrate) after each change is completed.\n")
	b.WriteString(">\n")
	b.WriteString("> Contrast with [`docs/specs/index.md`](../specs/index.md): specs are *pre-implementation* —\n")
	b.WriteString("> what you planned. Specs capture conceptual design intent and are human-curated.\n\n")
	b.WriteString("> **Generated by `fab memory-index`** — do not hand-edit. Re-run after any memory write;\n")
	b.WriteString("> the output is byte-stable. Per-file descriptions live in each file's `description:` frontmatter.\n\n")
	b.WriteString("> **New here?** Start with the [README](../../README.md) for setup and a walkthrough. For terminology, see the [Glossary](../specs/glossary.md).\n\n")
	b.WriteString("| Domain | Description |\n")
	b.WriteString("|--------|-------------|\n")
	for _, dr := range d.Domains {
		desc := dr.Description
		if desc == "" {
			desc = missingCell
		}
		fmt.Fprintf(&b, "| [%s](%s/index.md) | %s |\n", dr.Name, dr.Name, desc)
	}
	return b.String()
}

// RenderDomain assembles the complete docs/memory/{domain}/index.md markdown
// for d. Pure: identical DomainData always yields identical output.
func RenderDomain(d DomainData) string {
	var b strings.Builder
	// Round-trip the curated domain description through frontmatter so the root
	// index can read it back on the next regen — keeping the whole pipeline
	// idempotent (the generated index is the home of this fact).
	if d.Description != "" {
		fmt.Fprintf(&b, "---\ndescription: %q\n---\n", d.Description)
	}
	fmt.Fprintf(&b, "# %s\n\n", d.Title)
	b.WriteString("> **Generated by `fab memory-index`** — do not hand-edit. Descriptions come from each file's `description:` frontmatter; dates from `git log`.\n\n")
	b.WriteString("| File | Description | Last Updated |\n")
	b.WriteString("|------|-------------|-------------|\n")
	for _, f := range d.Files {
		desc := f.Description
		if desc == "" {
			desc = missingCell
		}
		date := f.LastUpdated
		if date == "" {
			date = missingCell
		}
		fmt.Fprintf(&b, "| [%s](%s.md) | %s | %s |\n", f.Base, f.Base, desc, date)
	}
	// Sub-domain references — emitted only when sub-domains exist, so a flat
	// domain index renders byte-identically to the pre-recursion output. Mirrors
	// the root index's domain-link convention: [name](name/index.md).
	if len(d.SubDomains) > 0 {
		b.WriteString("\n## Sub-Domains\n\n")
		b.WriteString("| Sub-Domain | Description |\n")
		b.WriteString("|------------|-------------|\n")
		for _, sd := range d.SubDomains {
			desc := sd.Description
			if desc == "" {
				desc = missingCell
			}
			fmt.Fprintf(&b, "| [%s](%s/index.md) | %s |\n", sd.Name, sd.Name, desc)
		}
	}
	return b.String()
}

// Gather walks docs/memory/ under repoRoot and reads every input the index
// renderers need: each domain's topic files (H1 + `description:` frontmatter +
// git date) and a domain description for the root row. It also computes the
// non-fatal shape warnings. Returns (root, domains, warnings, err). A missing
// docs/memory/ directory is a hard error; everything else degrades gracefully
// (missing frontmatter / dates render as the missing-cell fallback).
//
// domains is sorted lexicographically by Name; each domain's Files are sorted
// lexicographically by Base — so the output is deterministic and byte-stable.
func Gather(repoRoot string) (RootData, []DomainData, []Warning, error) {
	memRoot := filepath.Join(repoRoot, "docs", "memory")
	entries, err := os.ReadDir(memRoot)
	if err != nil {
		return RootData{}, nil, nil, fmt.Errorf("docs/memory not found under %s: %w", repoRoot, err)
	}

	var domains []DomainData
	var root RootData
	var warnings []Warning

	// One batched git-log pass over docs/memory replaces the per-file
	// `git log -1` spawns (N files → N subprocesses, each a history walk).
	// nil on failure → the per-file fallback inside dates.lookup.
	dates := loadGitDates(repoRoot)

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		domainName := e.Name()
		domainDir := filepath.Join(memRoot, domainName)

		files := gatherFiles(repoRoot, domainDir, dates)
		subDomains := gatherSubDomains(repoRoot, domainDir, dates)
		desc := domainDescription(domainDir)
		domains = append(domains, DomainData{
			Name:        domainName,
			Title:       domainTitle(domainDir, domainName),
			Description: desc,
			Files:       files,
			SubDomains:  subDomains,
		})
		root.Domains = append(root.Domains, DomainRow{
			Name:        domainName,
			Description: desc,
		})

		// Width warnings — reserved-exempt — for the domain and each sub-domain.
		// Depth warnings walk the whole subtree once below.
		if !reservedDomains[domainName] && len(files) > WidthWarnThreshold {
			warnings = append(warnings, Warning{
				Path:  filepath.ToSlash(filepath.Join("docs", "memory", domainName)),
				Kind:  "width",
				Count: len(files),
			})
		}
		for _, sd := range subDomains {
			if len(sd.Files) > WidthWarnThreshold {
				warnings = append(warnings, Warning{
					Path:  filepath.ToSlash(filepath.Join("docs", "memory", domainName, sd.Name)),
					Kind:  "width",
					Count: len(sd.Files),
				})
			}
		}
		warnings = append(warnings, depthWarnings(memRoot, domainDir)...)
	}

	sort.Slice(domains, func(i, j int) bool { return domains[i].Name < domains[j].Name })
	sort.Slice(root.Domains, func(i, j int) bool { return root.Domains[i].Name < root.Domains[j].Name })
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Path != warnings[j].Path {
			return warnings[i].Path < warnings[j].Path
		}
		return warnings[i].Kind < warnings[j].Kind
	})

	return root, domains, warnings, nil
}

// gatherFiles reads the topic files (non-index .md) directly under domainDir,
// sorted lexicographically by base name. dates is the batched git-date map
// (nil when the batched pass failed — lookup then falls back per file).
//
// index.md and log.md are both generated, single-writer artifacts — not topic
// files — so they are skipped. (gatherLogEntries applies the identical skip;
// excluding log.md here is what keeps a freshly-generated tree idempotent: a
// second `fab memory-index` run must not read the just-written log.md back as a
// topic row and add a spurious `[log](log.md)` line to the domain index.)
func gatherFiles(repoRoot, domainDir string, dates *gitDates) []FileEntry {
	dirEntries, err := os.ReadDir(domainDir)
	if err != nil {
		return nil
	}
	var files []FileEntry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, ".md") || name == "index.md" || name == "log.md" {
			continue
		}
		path := filepath.Join(domainDir, name)
		base := strings.TrimSuffix(name, ".md")
		files = append(files, FileEntry{
			Base:        base,
			Title:       readH1(path),
			Description: frontmatter.Field(path, "description"),
			LastUpdated: dates.lookup(repoRoot, path),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Base < files[j].Base })
	return files
}

// gatherSubDomains reads the immediate child directories of domainDir that hold
// at least one non-index topic file and returns a DomainData per sub-domain,
// sorted lexicographically by Name. Recursion is one level only: a sub-domain's
// own SubDomains field is left empty — deeper nesting is surfaced as a depth
// warning, not an additional generated index tier (the depth-3 bound is
// {domain}/{sub-domain}/{topic}.md). An empty sub-folder (no .md) yields no
// entry, so it never produces a spurious index.
func gatherSubDomains(repoRoot, domainDir string, dates *gitDates) []DomainData {
	dirEntries, err := os.ReadDir(domainDir)
	if err != nil {
		return nil
	}
	var subs []DomainData
	for _, de := range dirEntries {
		if !de.IsDir() {
			continue
		}
		subName := de.Name()
		subDir := filepath.Join(domainDir, subName)
		files := gatherFiles(repoRoot, subDir, dates)
		if len(files) == 0 {
			continue // no topic files → not a sub-domain, no index to generate
		}
		subs = append(subs, DomainData{
			Name:        subName,
			Title:       domainTitle(subDir, subName),
			Description: domainDescription(subDir),
			Files:       files,
		})
	}
	sort.Slice(subs, func(i, j int) bool { return subs[i].Name < subs[j].Name })
	return subs
}

// domainTitle reads the existing domain index.md H1 if present (preserving a
// curated heading), else synthesizes a Title-Case heading from the folder name.
func domainTitle(domainDir, domainName string) string {
	if h1 := readH1(filepath.Join(domainDir, "index.md")); h1 != "" {
		return h1
	}
	return titleCase(domainName) + " Documentation"
}

// domainDescription reads the `description:` frontmatter of the domain's
// index.md if present (the curated one-liner for the root row); otherwise "".
// Because RenderDomain round-trips this value back into the generated
// index.md's frontmatter, it survives regeneration.
func domainDescription(domainDir string) string {
	return frontmatter.Field(filepath.Join(domainDir, "index.md"), "description")
}

// readH1 returns the first `# ` heading text in the file, or "".
func readH1(path string) string {
	fileLines, err := lines.ReadFileLines(path)
	if err != nil {
		return ""
	}
	for _, l := range fileLines {
		line := strings.TrimSpace(l)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(line[2:])
		}
	}
	return ""
}

// titleCase converts a kebab/snake folder name to a spaced Title Case string.
func titleCase(name string) string {
	repl := strings.NewReplacer("-", " ", "_", " ")
	parts := strings.Fields(repl.Replace(name))
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, " ")
}

// gitDates is the result of the single batched git-log pass over
// docs/memory: the most recent commit date per file, keyed by the path
// relative to the git top-level (slash-separated, as git prints it). The same
// pass also captures the full per-path commit list (commitsByPath) that the
// log.md generator joins with the change registry — both projections come from
// ONE `git log` invocation (no per-file spawns; pw3k F34).
type gitDates struct {
	top           string                // `git rev-parse --show-toplevel` for repoRoot
	byPath        map[string]string     // repo-relative path → newest YYYY-MM-DD
	commitsByPath map[string][]gitTouch // repo-relative path → commits touching it, newest-first
}

// gitTouch is one (commit, file) tuple from the batched name-status pass: the
// commit's date and message (for change-id attribution) plus this file's
// per-commit status code (for verb derivation). It is the C-lite log's raw
// git input before the registry join.
type gitTouch struct {
	Date    string // commit date, YYYY-MM-DD
	Subject string // commit subject line (first line)
	Status  string // this file's name-status code in this commit (A/D/M/R.../C...)
}

// gitLogRecordSep / gitLogFieldSep are the bytes git EMITS to delimit the
// batched `git log` stream so the commit header (date + subject) is
// unambiguously separable from the name-status path lines, even when a subject
// contains arbitrary text. NUL (record) and US (unit) are bytes git never emits
// inside a one-line subject. NOTE: the `--format` string MUST use git's own
// `%x00` / `%x1f` escapes (gitLogFormat), NOT these literal bytes — a literal
// leading-NUL format gets swallowed when combined with --name-status, dropping
// the header line entirely (the original date-only pass used %x00 for the same
// reason). The parser splits on these emitted bytes.
const (
	gitLogRecordSep = "\x00"
	gitLogFieldSep  = "\x1f"
)

// gitLogFormat is the --format argument (git escapes, not literal bytes):
// "<NUL>%ad<US>%s" — record-separated date + subject header per commit.
const gitLogFormat = "%x00%ad%x1f%s"

// loadGitDates runs ONE `git log --date=short --name-status` pass over
// docs/memory and records (a) the first (= most recent, git log is
// newest-first) date seen per path and (b) the ordered per-path commit list
// the log generator consumes. Returns nil when git fails (not a repo, git
// missing) — callers then fall back to the per-file gitLastUpdated and emit no
// log.md. Equivalence with the per-file `git log -1 -- <path>` date defaults:
// merge commits contribute no file list (no -m), renames are not followed,
// matching both defaults; core.quotepath=off keeps non-ASCII paths unquoted so
// map keys match filesystem paths. --name-status (vs the former --name-only)
// is a superset — the date projection is unchanged; the status column is new.
func loadGitDates(repoRoot string) *gitDates {
	topCmd := exec.Command("git", "rev-parse", "--show-toplevel")
	if repoRoot != "" {
		topCmd.Dir = repoRoot
	}
	topOut, err := topCmd.Output()
	if err != nil {
		return nil
	}
	top := strings.TrimSpace(string(topOut))

	logCmd := exec.Command("git", "-c", "core.quotepath=off", "log",
		"--date=short", "--format="+gitLogFormat,
		"--name-status", "--", "docs/memory")
	if repoRoot != "" {
		logCmd.Dir = repoRoot
	}
	out, err := logCmd.Output()
	if err != nil {
		return nil
	}
	byPath, commitsByPath := parseGitLog(string(out))
	return &gitDates{top: top, byPath: byPath, commitsByPath: commitsByPath}
}

// parseGitLog parses the batched `--format=<NUL>%ad<US>%s --name-status` stream
// into both projections the package needs from ONE pass:
//   - byPath: the newest date per path (FIRST date seen wins, git being
//     newest-first) — the index's "Last Updated" source, unchanged in behavior.
//   - commitsByPath: the ordered (newest-first) list of (date, subject, status)
//     tuples per path — the C-lite log's raw git input.
//
// A record begins with a NUL line carrying "<date><US><subject>"; the following
// name-status lines are "<status>\t<path>" (or "<status>\t<oldpath>\t<newpath>"
// for renames/copies — the LAST tab-field is the current path). Pure function,
// extracted for unit tests.
func parseGitLog(out string) (byPath map[string]string, commitsByPath map[string][]gitTouch) {
	byPath = make(map[string]string)
	commitsByPath = make(map[string][]gitTouch)
	curDate, curSubject := "", ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, gitLogRecordSep) {
			header := strings.TrimPrefix(line, gitLogRecordSep)
			if i := strings.Index(header, gitLogFieldSep); i >= 0 {
				curDate = strings.TrimSpace(header[:i])
				curSubject = header[i+len(gitLogFieldSep):]
			} else {
				curDate = strings.TrimSpace(header)
				curSubject = ""
			}
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		// name-status row: "<status>\t<path>[\t<newpath>]". Tab-split; the
		// status is field 0, the current path is the LAST field (handles
		// rename/copy's old→new pair).
		fields := strings.Split(line, "\t")
		if len(fields) < 2 {
			continue
		}
		status := strings.TrimSpace(fields[0])
		path := strings.TrimSpace(fields[len(fields)-1])
		if path == "" {
			continue
		}
		if _, seen := byPath[path]; !seen {
			byPath[path] = curDate
		}
		commitsByPath[path] = append(commitsByPath[path], gitTouch{
			Date:    curDate,
			Subject: curSubject,
			Status:  status,
		})
	}
	return byPath, commitsByPath
}

// lookup returns the last-updated date for path. With a populated batch map
// (non-nil receiver) it is a pure map lookup — a missing key means the file
// is uncommitted and yields "", exactly like the per-file call. With a nil
// receiver (batched pass failed), or when path cannot be expressed relative
// to the git top-level (e.g. symlinked temp dirs — git prints the resolved
// top), it falls back to the per-file gitLastUpdated spawn.
func (d *gitDates) lookup(repoRoot, path string) string {
	if d == nil {
		return gitLastUpdated(repoRoot, path)
	}
	rel, err := filepath.Rel(d.top, path)
	if err != nil || strings.HasPrefix(rel, "..") {
		// Retry with symlinks resolved (git's top-level is always resolved).
		if resolved, rerr := filepath.EvalSymlinks(path); rerr == nil {
			if r2, e2 := filepath.Rel(d.top, resolved); e2 == nil && !strings.HasPrefix(r2, "..") {
				return d.byPath[filepath.ToSlash(r2)]
			}
		}
		return gitLastUpdated(repoRoot, path)
	}
	return d.byPath[filepath.ToSlash(rel)]
}

// gitLastUpdated returns `git log -1 --date=short --format=%ad <path>` run in
// repoRoot, or "" when git produces no output (uncommitted file, worktree /
// shallow-clone / squash / rebase context, or git unavailable). Mirrors how
// internal/prmeta degrades on missing git context — never an error. Kept as
// the per-file FALLBACK for when the batched loadGitDates pass fails.
func gitLastUpdated(repoRoot, path string) string {
	cmd := exec.Command("git", "log", "-1", "--date=short", "--format=%ad", "--", path)
	if repoRoot != "" {
		cmd.Dir = repoRoot
	}
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// --- C-lite log.md: change registry + commit attribution + gathering --------

// changeMeta is one registered change's identity + log inputs, keyed by its
// 4-char id in the registry. Folder is the YYMMDD-XXXX-slug folder name; Slug
// is the trailing slug (the §6.3 summary-absent fallback); Summary is the
// `.status.yaml` summary: field ("" when unset).
type changeMeta struct {
	Folder  string
	Slug    string
	Summary string
}

// gatherChangeRegistry enumerates every change under fab/changes/* and
// fab/changes/archive/** to build the canonical {id → changeMeta} map the log
// generator joins commits against. The change owns its own identity (the folder
// IS the registry — FKF/intake assumption #12), so this is authoritative. Each
// change's `.status.yaml` summary: is read here (the C-lite "what"). A missing
// fab/changes dir yields an empty (non-nil) registry — the log then degrades to
// id-less / slug-less entries rather than erroring.
func gatherChangeRegistry(fabRoot string) map[string]changeMeta {
	reg := map[string]changeMeta{}
	if fabRoot == "" {
		return reg
	}
	changesDir := filepath.Join(fabRoot, "changes")

	// Active changes: direct children of fab/changes/ (skip the archive dir).
	if entries, err := os.ReadDir(changesDir); err == nil {
		for _, e := range entries {
			if e.IsDir() && e.Name() != "archive" {
				registerChange(reg, changesDir, e.Name())
			}
		}
	}
	// Archived changes: fab/changes/archive/{YYYY}/{MM}/{folder} (walk for any
	// folder holding a .status.yaml, so the bucketing layout is not hard-coded).
	archiveDir := filepath.Join(changesDir, "archive")
	_ = filepath.Walk(archiveDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || !info.IsDir() {
			return nil
		}
		if _, statErr := os.Stat(filepath.Join(p, ".status.yaml")); statErr == nil {
			registerChange(reg, filepath.Dir(p), filepath.Base(p))
		}
		return nil
	})
	return reg
}

// registerChange adds one folder (under parentDir) to the registry, keyed by its
// extracted id. A folder with no parseable id is skipped (it cannot be a join
// target). The `.status.yaml` summary: is read via internal/statusfile.
func registerChange(reg map[string]changeMeta, parentDir, folder string) {
	id, slug := extractIDSlug(folder)
	if id == "" {
		return
	}
	summary := ""
	if st, err := statusfile.Load(filepath.Join(parentDir, folder, ".status.yaml")); err == nil {
		summary = st.Summary
	}
	reg[id] = changeMeta{Folder: folder, Slug: slug, Summary: summary}
}

// extractIDSlug splits a YYMMDD-XXXX-slug folder name into its 4-char id and the
// trailing slug. Mirrors internal/resolve.ExtractID's SplitN(folder,"-",3)
// convention (kept local to avoid importing the cmd-oriented resolve package).
// Returns ("","") when the name does not match the change-folder shape.
func extractIDSlug(folder string) (id, slug string) {
	parts := strings.SplitN(folder, "-", 3)
	if len(parts) < 2 {
		return "", ""
	}
	id = parts[1]
	if len(parts) == 3 {
		slug = parts[2]
	}
	return id, slug
}

// attributeCommit recovers the registered change a commit belongs to by scanning
// its subject for a token that resolves to a registry id, returning ("", false)
// when the commit cannot be attributed (FKF graceful degradation: a direct edit
// on main, a pre-FKF historical commit, or a squash-merge whose branch token was
// dropped). Two token shapes are recognized, both registry-GATED (a token only
// counts when it maps to a known change — never raw prose):
//   - a full YYMMDD-XXXX-slug folder name embedded in the subject (the
//     merge-commit branch token, "Merge pull request #N from owner/<folder>");
//   - a bare 4-char id that exactly matches a registry key.
//
// Gating on the registry keeps the join authoritative and false-positive-free.
func attributeCommit(subject string, reg map[string]changeMeta) (string, bool) {
	// 1. Full folder-name token → its id (only if that id is registered AND the
	//    registered folder matches, so a coincidental slug can't mis-attribute).
	for _, tok := range strings.FieldsFunc(subject, func(r rune) bool {
		return r == ' ' || r == '/' || r == '\t' || r == '(' || r == ')' || r == ':'
	}) {
		id, _ := extractIDSlug(tok)
		if id == "" {
			continue
		}
		if meta, ok := reg[id]; ok && meta.Folder == tok {
			return id, true
		}
	}
	// 2. Bare registered id appearing as a standalone token.
	for _, tok := range strings.FieldsFunc(subject, func(r rune) bool {
		return r == ' ' || r == '/' || r == '\t' || r == '(' || r == ')' || r == ':'
	}) {
		if _, ok := reg[tok]; ok {
			return tok, true
		}
	}
	return "", false
}

// LogTarget is one folder's rendered log.md: its path and content, built by
// GatherLogs. The cmd appends these to its byte-stable write / --check loop.
type LogTarget struct {
	Path    string // absolute log.md path
	Content string // rendered RenderLog output
}

// GatherLogs builds the log.md targets for every domain and sub-domain folder
// with attributable git history under repoRoot. It reuses the single batched git
// pass (loadGitDates' commitsByPath) and the change registry (gatherChangeRegistry
// over fabRoot), so it spawns no extra git processes. A folder with zero commits
// touching its files is SKIPPED (no empty log.md — Design Decision 4). When the
// batched git pass fails (non-git dir, git missing) it returns nil targets, no
// error — the log surface degrades gracefully exactly like the index dates do.
func GatherLogs(repoRoot, fabRoot string) ([]LogTarget, error) {
	memRoot := filepath.Join(repoRoot, "docs", "memory")
	entries, err := os.ReadDir(memRoot)
	if err != nil {
		return nil, fmt.Errorf("docs/memory not found under %s: %w", repoRoot, err)
	}

	dates := loadGitDates(repoRoot)
	if dates == nil || dates.commitsByPath == nil {
		return nil, nil // no git history → no logs (graceful, not an error)
	}
	reg := gatherChangeRegistry(fabRoot)

	var targets []LogTarget
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		domainName := e.Name()
		domainDir := filepath.Join(memRoot, domainName)

		// Domain-tier log.
		if t, ok := buildLogTarget(repoRoot, dates, reg, domainDir, domainName, ""); ok {
			targets = append(targets, t)
		}
		// Sub-domain logs (one level down, mirroring the index tiers).
		for _, sd := range gatherSubDomains(repoRoot, domainDir, dates) {
			subDir := filepath.Join(domainDir, sd.Name)
			if t, ok := buildLogTarget(repoRoot, dates, reg, subDir, domainName+"/"+sd.Name, sd.Title); ok {
				targets = append(targets, t)
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Path < targets[j].Path })
	return targets, nil
}

// buildLogTarget assembles one folder's LogData → log.md target. bundleRel is
// the folder's bundle-relative base ("distribution" or "fab-workflow/runtime").
// titleOverride (when non-empty) is the gathered sub-domain Title; for a domain
// it is "" and the Title is read from the folder's index.md / synthesized.
// Returns ok=false when the folder has no attributable commits (skip, no file).
func buildLogTarget(repoRoot string, dates *gitDates, reg map[string]changeMeta, folderDir, bundleRel, titleOverride string) (LogTarget, bool) {
	entries := gatherLogEntries(repoRoot, dates, reg, folderDir, bundleRel)
	if len(entries) == 0 {
		return LogTarget{}, false
	}
	title := titleOverride
	if title == "" {
		title = domainTitle(folderDir, filepath.Base(folderDir))
	}
	return LogTarget{
		Path:    filepath.Join(folderDir, "log.md"),
		Content: RenderLog(LogData{Title: title, Entries: entries}),
	}, true
}

// gatherLogEntries projects the batched commit history for one folder's direct
// topic files into LogEntry values, attributing each commit to a registered
// change (slug/summary fallback per §6.3) and deriving the verb from the
// per-commit name-status. Entries are returned newest-commit-first (git's order),
// with a stable secondary sort (file base then change-id) so same-date entries
// are byte-stable across runs. Only direct topic files are considered — a
// sub-domain's history belongs to the sub-domain's own log.
func gatherLogEntries(repoRoot string, dates *gitDates, reg map[string]changeMeta, folderDir, bundleRel string) []LogEntry {
	dirEntries, err := os.ReadDir(folderDir)
	if err != nil {
		return nil
	}
	var entries []LogEntry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, ".md") || name == "index.md" || name == "log.md" {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		bundlePath := "/" + bundleRel + "/" + base + ".md"
		rel := gitRelPath(dates, repoRoot, filepath.Join(folderDir, name))
		for _, touch := range dates.commitsByPath[rel] {
			summary, id := "", ""
			if cid, ok := attributeCommit(touch.Subject, reg); ok {
				id = cid
				if meta := reg[cid]; meta.Summary != "" {
					summary = meta.Summary // §6.3 the curated "what"
				} else {
					summary = meta.Slug // §6.3 slug fallback (summary unset)
				}
			} else {
				// Unattributable commit (direct main edit, pre-FKF history, or a
				// squash-merge that dropped the branch token): degrade gracefully
				// per FKF §6 / R7 — omit the (change-id) and use the commit subject
				// as the descriptive line (a git projection, still conflict-free).
				// Falls through to the renderer's "—" when even the subject is empty.
				summary = strings.TrimSpace(touch.Subject)
			}
			entries = append(entries, LogEntry{
				Date:          touch.Date,
				Verb:          nameStatusVerb(touch.Status),
				FileBase:      base,
				BundleRelPath: bundlePath,
				Summary:       summary,
				ChangeID:      id,
			})
		}
	}
	// Stable deterministic order: newest date first, then file base, then id —
	// independent of os.ReadDir order so the output is byte-stable.
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Date != entries[j].Date {
			return entries[i].Date > entries[j].Date
		}
		if entries[i].FileBase != entries[j].FileBase {
			return entries[i].FileBase < entries[j].FileBase
		}
		return entries[i].ChangeID < entries[j].ChangeID
	})
	return entries
}

// gitRelPath returns path expressed relative to the git top-level in the
// slash-separated form parseGitLog keyed commitsByPath by (matching how git
// prints paths). Falls back to a docs/memory-relative guess when the top is
// unknown.
func gitRelPath(dates *gitDates, repoRoot, path string) string {
	if dates != nil && dates.top != "" {
		if rel, err := filepath.Rel(dates.top, path); err == nil && !strings.HasPrefix(rel, "..") {
			return filepath.ToSlash(rel)
		}
		if resolved, rerr := filepath.EvalSymlinks(path); rerr == nil {
			if rel, err := filepath.Rel(dates.top, resolved); err == nil && !strings.HasPrefix(rel, "..") {
				return filepath.ToSlash(rel)
			}
		}
	}
	if rel, err := filepath.Rel(repoRoot, path); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.ToSlash(path)
}

// depthWarnings walks domainDir for any .md file whose depth under docs/memory/
// exceeds MaxDepth and returns a warning per offending directory (deduped).
func depthWarnings(memRoot, domainDir string) []Warning {
	seen := map[string]bool{}
	var out []Warning
	_ = filepath.Walk(domainDir, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		rel, relErr := filepath.Rel(memRoot, p)
		if relErr != nil {
			return nil
		}
		// depth = number of path segments under docs/memory/ counting the file.
		// {domain}/{topic}.md = 2; {domain}/{sub}/{topic}.md = 3; deeper warns.
		depth := len(strings.Split(filepath.ToSlash(rel), "/"))
		if depth <= MaxDepth {
			return nil
		}
		dir := filepath.ToSlash(filepath.Join("docs", "memory", filepath.Dir(rel)))
		if seen[dir] {
			return nil
		}
		seen[dir] = true
		out = append(out, Warning{Path: dir, Kind: "depth", Depth: depth})
		return nil
	})
	return out
}
