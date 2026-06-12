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
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/frontmatter"
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

// RenderRoot assembles the complete root docs/memory/index.md markdown for d.
// It is pure: identical RootData always yields identical output. The table is
// domains-only — the legacy inlined per-file "Memory Files" column is dropped.
func RenderRoot(d RootData) string {
	var b strings.Builder
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
		if !strings.HasSuffix(name, ".md") || name == "index.md" {
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
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
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
// relative to the git top-level (slash-separated, as git prints it).
type gitDates struct {
	top    string            // `git rev-parse --show-toplevel` for repoRoot
	byPath map[string]string // repo-relative path → YYYY-MM-DD
}

// loadGitDates runs ONE `git log --date=short --name-only` pass over
// docs/memory and records the first (= most recent, git log is newest-first)
// date seen per path. Returns nil when git fails (not a repo, git missing) —
// callers then fall back to the per-file gitLastUpdated. Equivalence with
// the per-file `git log -1 -- <path>` defaults: --name-only skips
// merge-commit file lists and does not follow renames, matching both
// defaults; core.quotepath=off keeps non-ASCII paths unquoted so map keys
// match filesystem paths.
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
		"--date=short", "--format=%x00%ad", "--name-only", "--", "docs/memory")
	if repoRoot != "" {
		logCmd.Dir = repoRoot
	}
	out, err := logCmd.Output()
	if err != nil {
		return nil
	}
	return &gitDates{top: top, byPath: parseGitDates(string(out))}
}

// parseGitDates parses the `--format=%x00%ad --name-only` stream: a line
// starting with NUL carries the commit date; subsequent non-empty lines are
// the paths it touched. The FIRST date seen per path wins (newest-first
// ordering). Pure function, extracted for unit tests.
func parseGitDates(out string) map[string]string {
	byPath := make(map[string]string)
	current := ""
	for _, line := range strings.Split(out, "\n") {
		if strings.HasPrefix(line, "\x00") {
			current = strings.TrimSpace(line[1:])
			continue
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if _, seen := byPath[line]; !seen {
			byPath[line] = current
		}
	}
	return byPath
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
