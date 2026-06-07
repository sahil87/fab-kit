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
// a plain value so RenderDomain is a pure function of it.
type DomainData struct {
	// Name is the domain folder name (e.g. "fab-workflow").
	Name string
	// Title is the human heading rendered at the top of the domain index.
	Title string
	// Description is the curated one-liner for the root index row; it is
	// round-tripped through the generated domain index.md's `description:`
	// frontmatter so it survives regeneration (the domain index is the single
	// home for this fact — the root index reads it back here).
	Description string
	// Files are the domain's topic files, sorted lexicographically by Base.
	Files []FileEntry
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

	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		domainName := e.Name()
		domainDir := filepath.Join(memRoot, domainName)

		files := gatherFiles(repoRoot, domainDir)
		desc := domainDescription(domainDir)
		domains = append(domains, DomainData{
			Name:        domainName,
			Title:       domainTitle(domainDir, domainName),
			Description: desc,
			Files:       files,
		})
		root.Domains = append(root.Domains, DomainRow{
			Name:        domainName,
			Description: desc,
		})

		// Shape warnings — width (reserved-exempt) + depth.
		if !reservedDomains[domainName] && len(files) > WidthWarnThreshold {
			warnings = append(warnings, Warning{
				Path:  filepath.ToSlash(filepath.Join("docs", "memory", domainName)),
				Kind:  "width",
				Count: len(files),
			})
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
// sorted lexicographically by base name.
func gatherFiles(repoRoot, domainDir string) []FileEntry {
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
			LastUpdated: gitLastUpdated(repoRoot, path),
		})
	}
	sort.Slice(files, func(i, j int) bool { return files[i].Base < files[j].Base })
	return files
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

// gitLastUpdated returns `git log -1 --date=short --format=%ad <path>` run in
// repoRoot, or "" when git produces no output (uncommitted file, worktree /
// shallow-clone / squash / rebase context, or git unavailable). Mirrors how
// internal/prmeta degrades on missing git context — never an error.
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
