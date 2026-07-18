// Package memoryindex deterministically (re)generates the docs/memory index
// files: the root docs/memory/index.md (domains-only) and every
// docs/memory/{domain}/index.md (file rows). It is the deterministic
// counterpart to the hand-maintained index rows that previously lived in the
// hydrate / docs-reorg-memory skill prose — reading the same inputs (each
// memory file's H1 + `description:` frontmatter) and emitting the exact same
// markdown on every run so the indexes stop drifting and stop generating merge
// conflicts on the hot per-row cells. The index is a pure function of content
// (file names + descriptions + structure) — no git dates — so its output is
// branch-independent and idempotent; per-folder change history (the "when")
// lives in the freeze-on-write log.md instead.
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
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

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
	// DescriptionLenWarnThreshold is the soft upper bound on a `description:`
	// value's length (in characters/runes, measured on the quote-stripped
	// single-line scalar). A description strictly longer than this triggers an
	// advisory length warning. Hardcoded (the shape-bound-const pattern, like
	// WidthWarnThreshold) — NOT config-overridable in this change. Curated
	// one-liner is FKF §3.2's intent; detail belongs in the file body.
	DescriptionLenWarnThreshold = 500
	// DescriptionBlockingLenThreshold is the BLOCKING upper bound on a
	// `description:` value's rune length — 2× the advisory soft cap
	// (DescriptionLenWarnThreshold). A description strictly longer than this
	// FAILS `--check` (joins the blocking class), not merely nags: the 501–1000
	// range keeps the advisory length warning; past 1000 the check blocks. The
	// advisory-only posture demonstrably failed (33×/50× descriptions shipped
	// straight through the nag — mxgu). Hardcoded shape-bound const, NOT config.
	DescriptionBlockingLenThreshold = 1000
	// NarrationMarkerWarnThreshold is the advisory threshold on a topic file's
	// narration-marker count (transition stems + registry-gated change-id tokens
	// in the body — the distillation-debt meter). A file reaching this count
	// triggers an advisory warning. Hardcoded shape-bound const, NOT config.
	NarrationMarkerWarnThreshold = 5
	// FileSizeLineWarnThreshold is the advisory soft cap on a topic file's line
	// count. A file strictly over this triggers an advisory size warning (the
	// mega-file split signal). Hardcoded shape-bound const, NOT config.
	FileSizeLineWarnThreshold = 400
	// FileSizeByteWarnThreshold is the advisory soft cap on a topic file's byte
	// size (15KB = 15×1024). A file strictly over this triggers the same
	// advisory size warning (either bound trips it). Hardcoded const, NOT config.
	FileSizeByteWarnThreshold = 15 * 1024
)

// Warning Kind values. The shape-bound kinds ("width"/"depth") are advisory;
// the malformed-frontmatter kinds are the blocking corruption signals surfaced
// to `--check` (see the cmd's LossReport.Malformed). "description-length" is
// advisory (never blocks). All three new kinds keep the rendered index output
// byte-identical — they are stderr/exit-code only (change 260715-xu0k).
const (
	// KindWidth: a folder holds more topic files than WidthWarnThreshold.
	KindWidth = "width"
	// KindDepth: nesting under docs/memory/ exceeds MaxDepth.
	KindDepth = "depth"
	// KindMalformedFence: a memory file's frontmatter block is unclosed (opens
	// `---` with no closing `---`) — the loom glued-fence corruption is an
	// instance. Blocking: fails `--check` independent of index drift.
	KindMalformedFence = "malformed-fence"
	// KindMalformedDescription: a `description:` value starts with a quote but
	// fails quote-stripping (the glued-fence diagnostic, e.g. trailing `"---`).
	// Blocking, like KindMalformedFence.
	KindMalformedDescription = "malformed-description"
	// KindDescriptionLength: a `description:` value exceeds
	// DescriptionLenWarnThreshold characters. Advisory only — never blocks
	// `--check` (the deliberate asymmetry: corruption blocks, over-length nags).
	KindDescriptionLength = "description-length"
	// KindDescriptionChangeID: a `description:` value carries a registry-gated
	// change-id token (a full YYMMDD-XXXX-slug folder-name token or a bare
	// registered 4-char id). BLOCKING — the FKF §3.2 change-id ban is now
	// enforced. Descriptions are routing signals; citations belong in the body.
	KindDescriptionChangeID = "description-change-id"
	// KindDescriptionOverCap: a `description:` value exceeds
	// DescriptionBlockingLenThreshold runes (2× the soft cap). BLOCKING — gross
	// over-cap fails `--check` (the 501–1000 advisory nag demonstrably failed).
	KindDescriptionOverCap = "description-over-cap"
	// KindNarrationDensity: a topic file's narration-marker count (transition
	// stems + registry-gated change-id tokens in the body) reaches
	// NarrationMarkerWarnThreshold. ADVISORY — the standing distillation-debt
	// meter (counts sanctioned citations too; density is the signal).
	KindNarrationDensity = "narration-density"
	// KindFileSize: a topic file exceeds FileSizeLineWarnThreshold lines OR
	// FileSizeByteWarnThreshold bytes. ADVISORY — the mega-file split signal.
	KindFileSize = "file-size"
	// KindUnsorted: docs/memory/_unsorted/ holds ≥1 topic file. ADVISORY —
	// staging should trend to empty (a presence signal, not a shape bound;
	// _unsorted keeps its width exemption).
	KindUnsorted = "unsorted-nonempty"
	// KindBrokenLink: a bundle-relative `](/...)` link target in a topic file
	// body does not resolve on disk under docs/memory/. ADVISORY — FKF §7 says
	// consumers tolerate broken links; this is the author-side nag.
	KindBrokenLink = "broken-link"
)

// blockingKinds is the set of Warning kinds that FAIL the cmd's `--check`
// (as distinct from the advisory shape/length/density/size warnings). It
// generalizes the former malformed-frontmatter set: the two malformed
// corruption kinds plus the two description escalations (registry-gated
// change-id, gross over-cap). All four floor `--check` at exit 1 independent of
// index drift and ride the additive `malformed` JSON array; none is a tier-2
// destructive-loss category (exit 2 stays reserved), so the hydrate/reorg
// refuse-before-regen guards (keyed on exit == 2) are unaffected. Kept here so
// producer and consumer share one list.
var blockingKinds = map[string]bool{
	KindMalformedFence:       true,
	KindMalformedDescription: true,
	KindDescriptionChangeID:  true,
	KindDescriptionOverCap:   true,
}

// IsBlocking reports whether w is a blocking finding (malformed frontmatter or a
// description escalation) as opposed to an advisory width/depth/length/density/
// size/staging/link warning. The cmd's `--check` branch uses this to floor the
// exit code at 1 independent of the drift tier.
func (w Warning) IsBlocking() bool { return blockingKinds[w.Kind] }

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

// Warning is a non-fatal finding surfaced to stderr (and, for the blocking
// kinds, to the cmd's `--check` exit gate). String renders the stderr line.
// Count is reused across kinds: file count (width), description rune length
// (description-length / description-over-cap), narration-marker count
// (narration-density), or line count (file-size). Bytes carries the byte size
// (file-size). Detail carries the offending frontmatter value
// (malformed-description), the matched change-id (description-change-id), or the
// broken link target (broken-link).
type Warning struct {
	Path   string // repo-relative folder/file path the finding is about
	Kind   string // one of the Kind* constants
	Count  int    // file count (width) | description rune length | marker count | line count
	Depth  int    // observed depth (depth) — 0 otherwise
	Bytes  int    // observed byte size (file-size) — 0 otherwise
	Detail string // offending value / matched change-id / broken link target — "" otherwise
}

// String formats the warning line written to stderr.
func (w Warning) String() string {
	switch w.Kind {
	case KindWidth:
		return fmt.Sprintf("⚠ %s has %d topic files (soft bound: ~%d) — consider splitting into sub-domains",
			w.Path, w.Count, WidthWarnThreshold)
	case KindDepth:
		return fmt.Sprintf("⚠ %s is nested %d levels deep (max: %d) — consider flattening", w.Path, w.Depth, MaxDepth)
	case KindMalformedFence:
		return fmt.Sprintf("✖ %s has malformed frontmatter — unclosed frontmatter block (no closing `---`)", w.Path)
	case KindMalformedDescription:
		return fmt.Sprintf("✖ %s has malformed frontmatter — `description:` value fails quote-stripping (unterminated quote): %s", w.Path, w.Detail)
	case KindDescriptionLength:
		return fmt.Sprintf("⚠ %s has a %d-character `description:` (soft cap: %d) — trim to a one-liner; detail belongs in the file body",
			w.Path, w.Count, DescriptionLenWarnThreshold)
	case KindDescriptionChangeID:
		return fmt.Sprintf("✖ %s `description:` carries a change-id (registry match: %s) — descriptions are routing signals; move citations to the body (FKF §3.2)",
			w.Path, w.Detail)
	case KindDescriptionOverCap:
		return fmt.Sprintf("✖ %s has a %d-character `description:` (blocking cap: %d, soft cap: %d) — trim to a one-liner; detail belongs in the file body",
			w.Path, w.Count, DescriptionBlockingLenThreshold, DescriptionLenWarnThreshold)
	case KindNarrationDensity:
		return fmt.Sprintf("⚠ %s has %d narration markers (threshold: %d) — distillation debt; consider /docs-distill-memory",
			w.Path, w.Count, NarrationMarkerWarnThreshold)
	case KindFileSize:
		return fmt.Sprintf("⚠ %s is %d lines / %dKB (soft cap: ~%d lines / ~%dKB) — consider splitting; see /docs-reorg-memory",
			w.Path, w.Count, w.Bytes/1024, FileSizeLineWarnThreshold, FileSizeByteWarnThreshold/1024)
	case KindUnsorted:
		return fmt.Sprintf("⚠ %s holds %d staged file(s) — triage into domains (staging should trend to empty)",
			w.Path, w.Count)
	case KindBrokenLink:
		return fmt.Sprintf("⚠ %s links to %s — target does not exist", w.Path, w.Detail)
	default:
		return fmt.Sprintf("⚠ %s", w.Path)
	}
}

// missingCell is the fallback rendered for an absent description,
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
	b.WriteString("> **Generated by `fab memory-index`** — do not hand-edit. Descriptions come from each file's `description:` frontmatter.\n\n")
	b.WriteString("| File | Description |\n")
	b.WriteString("|------|-------------|\n")
	for _, f := range d.Files {
		desc := f.Description
		if desc == "" {
			desc = missingCell
		}
		fmt.Fprintf(&b, "| [%s](%s.md) | %s |\n", f.Base, f.Base, desc)
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
// renderers need: each domain's topic files (H1 + `description:` frontmatter)
// and a domain description for the root row. It also computes the non-fatal
// shape warnings. Returns (root, domains, warnings, err). A missing
// docs/memory/ directory is a hard error; everything else degrades gracefully
// (missing frontmatter renders as the missing-cell fallback).
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

		files := gatherFiles(domainDir)
		subDomains := gatherSubDomains(domainDir)
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
				Kind:  KindWidth,
				Count: len(files),
			})
		}
		// _unsorted staging presence (advisory): staging should trend to empty,
		// so ANY topic file present is the signal. _unsorted keeps its width
		// exemption (above) — this is a presence signal, not a shape bound.
		if domainName == "_unsorted" && len(files) > 0 {
			warnings = append(warnings, Warning{
				Path:  filepath.ToSlash(filepath.Join("docs", "memory", domainName)),
				Kind:  KindUnsorted,
				Count: len(files),
			})
		}
		for _, sd := range subDomains {
			if len(sd.Files) > WidthWarnThreshold {
				warnings = append(warnings, Warning{
					Path:  filepath.ToSlash(filepath.Join("docs", "memory", domainName, sd.Name)),
					Kind:  KindWidth,
					Count: len(sd.Files),
				})
			}
		}
		warnings = append(warnings, depthWarnings(memRoot, domainDir)...)
	}

	// Frontmatter validation + description/body/link/staging warnings — a
	// read-only pass over every topic file and every index.md stub read for a
	// description. It never touches the rendered output (byte-stability, intake
	// #3); it only produces stderr/exit-code warnings. Walked separately from
	// the render gather so the render path stays untouched. The change registry
	// (fab/changes/* + archive/**) is gathered ONCE here and threaded into the
	// pass so the registry-gated change-id checks (description blocking + body
	// narration meter) resolve tokens without a per-file registry walk. fabRoot
	// is derived from repoRoot the same way the cmd derives repoRoot from
	// fabRoot (repoRoot = filepath.Dir(fabRoot)); a missing fab/changes yields an
	// empty registry (gatherChangeRegistry degrades gracefully — no false
	// change-id matches, exactly the born-FKF / test-tree case).
	reg := gatherChangeRegistry(filepath.Join(repoRoot, "fab"))
	warnings = append(warnings, frontmatterWarnings(memRoot, reg)...)

	sort.Slice(domains, func(i, j int) bool { return domains[i].Name < domains[j].Name })
	sort.Slice(root.Domains, func(i, j int) bool { return root.Domains[i].Name < root.Domains[j].Name })
	sort.Slice(warnings, func(i, j int) bool {
		if warnings[i].Path != warnings[j].Path {
			return warnings[i].Path < warnings[j].Path
		}
		if warnings[i].Kind != warnings[j].Kind {
			return warnings[i].Kind < warnings[j].Kind
		}
		// Detail tiebreaks same-(path,kind) warnings (e.g. multiple broken links
		// in one file) so the order is fully deterministic / byte-stable.
		return warnings[i].Detail < warnings[j].Detail
	})

	return root, domains, warnings, nil
}

// gatherFiles reads the topic files (non-index .md) directly under domainDir,
// sorted lexicographically by base name.
//
// index.md and log.md are both generated, single-writer artifacts — not topic
// files — so they are skipped. (gatherLogEntries applies the identical skip;
// excluding log.md here is what keeps a freshly-generated tree idempotent: a
// second `fab memory-index` run must not read the just-written log.md back as a
// topic row and add a spurious `[log](log.md)` line to the domain index.)
func gatherFiles(domainDir string) []FileEntry {
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
		if !strings.HasSuffix(name, ".md") || name == "index.md" || name == "log.md" || name == seedFileName {
			continue
		}
		path := filepath.Join(domainDir, name)
		base := strings.TrimSuffix(name, ".md")
		files = append(files, FileEntry{
			Base:        base,
			Title:       readH1(path),
			Description: frontmatter.Field(path, "description"),
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
func gatherSubDomains(domainDir string) []DomainData {
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
		files := gatherFiles(subDir)
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
// docs/memory: the full per-path commit list (commitsByPath) that the log.md
// generator joins with the change registry, keyed by the path relative to the
// git top-level (slash-separated, as git prints it). This is the sole git
// projection the package needs — the index is a pure function of content (no
// dates), so only log.md consumes this pass.
type gitDates struct {
	top           string                // `git rev-parse --show-toplevel` for repoRoot
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
// docs/memory and records the ordered per-path commit list the log generator
// consumes. Returns nil when git fails (not a repo, git missing) — callers then
// emit no log.md. core.quotepath=off keeps non-ASCII paths unquoted so map keys
// match filesystem paths. --name-status carries the per-commit status column
// the log's verb derivation needs.
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
	return &gitDates{top: top, commitsByPath: parseGitLog(string(out))}
}

// parseGitLog parses the batched `--format=<NUL>%ad<US>%s --name-status` stream
// into the per-path commit list the C-lite log generator consumes:
// commitsByPath maps each path to the ordered (newest-first) list of
// (date, subject, status) tuples touching it.
//
// A record begins with a NUL line carrying "<date><US><subject>"; the following
// name-status lines are "<status>\t<path>" (or "<status>\t<oldpath>\t<newpath>"
// for renames/copies — the LAST tab-field is the current path). Pure function,
// extracted for unit tests.
func parseGitLog(out string) (commitsByPath map[string][]gitTouch) {
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
		commitsByPath[path] = append(commitsByPath[path], gitTouch{
			Date:    curDate,
			Subject: curSubject,
			Status:  status,
		})
	}
	return commitsByPath
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

// changeIDTokenSep splits text into candidate change-id tokens. It starts from
// attributeCommit's delimiters (whitespace, slash, and the punctuation that
// wraps the banned §3.2 shapes — parentheses `(d9rs)`, colons, the `— xu0k`
// suffix's spaces) and adds the prose/markdown punctuation a body/description
// scan sees but a commit-subject scan does not: `,` `[` `]` newline, plus
// sentence terminators (`.` `;` `!` `?`), the ASCII quotes/backtick (`"` `'`
// and the backtick), the em-dash `—` itself (so a GLUED `—xu0k` suffix with no
// surrounding space still splits — the banned §3.2 suffix shape), and `*` (so a
// bolded `**xu0k**` markdown citation tokenizes cleanly) so a citation like
// `(d9rs).`, `see abcd;`, `'xu0k'`, `—xu0k`, or `**xu0k**` tokenizes cleanly.
// (Curly/smart quotes “ ” are NOT in the separator set — only the ASCII quotes
// above are.) It
// deliberately does NOT split on `-` — a full folder-name token
// (YYMMDD-XXXX-slug) contains hyphens and must survive as one token (the
// em-dash `—` U+2014 is a distinct rune from the ASCII hyphen `-` U+002D, so
// splitting on it does not fragment folder tokens).
func changeIDTokenSep(r rune) bool {
	switch r {
	case ' ', '/', '\t', '\n', '\r', '(', ')', ':', ',', '[', ']',
		'.', ';', '!', '?', '"', '\'', '`', '—', '*':
		return true
	}
	return false
}

// changeIDTokenID resolves a single candidate token to a registry-gated
// change-id, or "" when it does not resolve. A token counts only in
// attributeCommit's two gated shapes: a full YYMMDD-XXXX-slug folder-name token
// whose registered folder matches (so a coincidental slug cannot mis-attribute),
// or a bare registered 4-char id. False-positive-free — "code"/"yaml" and any
// unregistered 4-char word never resolve.
func changeIDTokenID(tok string, reg map[string]changeMeta) string {
	if id, _ := extractIDSlug(tok); id != "" {
		if meta, ok := reg[id]; ok && meta.Folder == tok {
			return id
		}
	}
	if _, ok := reg[tok]; ok {
		return tok
	}
	return ""
}

// scanChangeIDs returns the registry-gated change-ids appearing in text,
// DEDUPLICATED, in first-seen order — the set used by the `description:`
// blocking check (which reports the matched id(s), so uniqueness is what matters).
func scanChangeIDs(text string, reg map[string]changeMeta) []string {
	var ids []string
	seen := map[string]bool{}
	for _, tok := range strings.FieldsFunc(text, changeIDTokenSep) {
		if id := changeIDTokenID(tok, reg); id != "" && !seen[id] {
			seen[id] = true
			ids = append(ids, id)
		}
	}
	return ids
}

// countChangeIDOccurrences returns the TOTAL number of registry-gated change-id
// tokens in text (NOT deduplicated) — the density measure the narration-marker
// meter needs (three citations of the same id are three provenance markers, not
// one; the audit counts occurrences: run-kit 1,066, loom 2,121).
func countChangeIDOccurrences(text string, reg map[string]changeMeta) int {
	n := 0
	for _, tok := range strings.FieldsFunc(text, changeIDTokenSep) {
		if changeIDTokenID(tok, reg) != "" {
			n++
		}
	}
	return n
}

// LogTarget is one folder's rendered log.md: its path and content, built by
// GatherLogs. The cmd appends these to its byte-stable write / --check loop.
type LogTarget struct {
	Path    string // absolute log.md path
	Content string // rendered RenderLog output
}

// GatherLogs builds the log.md targets for every domain and sub-domain folder
// under repoRoot. It reuses the single batched git pass (loadGitDates'
// commitsByPath) and the change registry (gatherChangeRegistry over fabRoot), so
// it spawns no extra git processes. A folder is included when it has ANY entries
// after the freeze-on-write merge: an existing frozen log.md, a per-folder
// log.seed.md, and/or freshly attributable git commits. A folder that nets zero
// entries is SKIPPED (no empty log.md — Design Decision 4). When the batched git
// pass fails (non-git dir, git missing) the git-projection surface degrades to
// empty, but frozen log.md and log.seed.md entries still produce targets — so
// the result is nil only when no folder has any frozen/seed/git entry at all.
//
// rebuild selects the freeze-on-write mode (R6): false (the default,
// `fab memory-index`) reads each existing log.md and appends-only; true
// (`fab memory-index --rebuild`) discards the frozen state and re-projects every
// log.md from current git (destructive). --check passes rebuild=false so the
// rendered content is the freeze-on-write merge the classifier byte-compares
// against (R7–R9).
func GatherLogs(repoRoot, fabRoot string, rebuild bool) ([]LogTarget, error) {
	memRoot := filepath.Join(repoRoot, "docs", "memory")
	entries, err := os.ReadDir(memRoot)
	if err != nil {
		return nil, fmt.Errorf("docs/memory not found under %s: %w", repoRoot, err)
	}

	// dates may be nil when git is entirely unavailable (non-repo, git missing):
	// the git-projection surface then degrades to empty, but any per-folder
	// log.seed.md still produces a log.md (seed entries are git-independent — the
	// pre-FKF history they preserve has no live git/`.status.yaml` to project from).
	dates := loadGitDates(repoRoot)
	reg := gatherChangeRegistry(fabRoot)

	var targets []LogTarget
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		domainName := e.Name()
		domainDir := filepath.Join(memRoot, domainName)

		// Domain-tier log.
		if t, ok := buildLogTarget(repoRoot, dates, reg, domainDir, domainName, "", rebuild); ok {
			targets = append(targets, t)
		}
		// Sub-domain logs (one level down, mirroring the index tiers).
		for _, sd := range gatherSubDomains(domainDir) {
			subDir := filepath.Join(domainDir, sd.Name)
			if t, ok := buildLogTarget(repoRoot, dates, reg, subDir, domainName+"/"+sd.Name, sd.Title, rebuild); ok {
				targets = append(targets, t)
			}
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targets[i].Path < targets[j].Path })
	return targets, nil
}

// buildLogTarget assembles one folder's LogData → log.md target under the
// freeze-on-write model (R1–R4, R6). bundleRel is the folder's bundle-relative
// base ("distribution" or "fab-workflow/runtime"). titleOverride (when non-empty)
// is the gathered sub-domain Title; for a domain it is "" and the Title is read
// from the folder's index.md / synthesized. Returns ok=false when the folder has
// no entries at all (skip, no file).
//
// Freeze-on-write flow:
//  1. Read the EXISTING log.md and parse it back into the frozen, authoritative
//     entry set (R1). bootstrap is true when no existing log.md is present.
//  2. Project the live git history (gatherLogEntries). Unattributable commits are
//     projected only at bootstrap or under rebuild (R3 gate).
//  3. Append-only merge (R2): existing entries are kept verbatim; a projected
//     ATTRIBUTABLE entry is appended only when its (FileBase, ChangeID) pair is
//     absent from the existing set. Re-running, or re-projecting a squash that
//     preserved the change-id token, is a no-op.
//  4. Merge the folder's `log.seed.md` (FKF §6 seed input) beneath (R4) —
//     de-duplicated against the running set, byte-stable / idempotent.
//
// Under rebuild the existing log is DISCARDED and every entry re-projected from
// current git (the pre-freeze behavior, made explicit and destructive — R6),
// with the seed merged beneath as at bootstrap.
func buildLogTarget(repoRoot string, dates *gitDates, reg map[string]changeMeta, folderDir, bundleRel, titleOverride string, rebuild bool) (LogTarget, bool) {
	logPath := filepath.Join(folderDir, "log.md")

	// Existing frozen log — authoritative on a normal run, discarded under rebuild.
	var existing []LogEntry
	if !rebuild {
		if data, err := os.ReadFile(logPath); err == nil {
			existing = parseLog(string(data))
		}
	}
	bootstrap := len(existing) == 0

	// Project live git; the unattributable branch is gated to bootstrap / rebuild.
	projected := gatherLogEntries(repoRoot, dates, reg, folderDir, bundleRel, bootstrap || rebuild)

	// Append-only merge: existing entries are immutable; only NEW attributable
	// (FileBase, ChangeID) pairs are appended (R1/R2). At bootstrap/rebuild the
	// existing set is empty, so this is the full projection.
	entries := appendNewEntries(existing, projected)

	// Seed merge beneath (R4) — de-duplicated against the running set.
	seed := readSeedEntries(folderDir)
	entries = mergeSeedEntries(entries, seed)
	if len(entries) == 0 {
		return LogTarget{}, false
	}
	// Re-apply the stable order (date desc, file base, change-id) over the merged
	// set so existing / appended / seed entries interleave deterministically.
	// mergeSeedEntries keeps the running (existing+appended) entries ahead of seed
	// entries in slice order, and a stable sort preserves that for entries equal
	// under the comparator — so within a date the frozen + git-projected lines
	// render before the seed lines.
	sortLogEntries(entries)
	title := titleOverride
	if title == "" {
		title = domainTitle(folderDir, filepath.Base(folderDir))
	}
	return LogTarget{
		Path:    logPath,
		Content: RenderLog(LogData{Title: title, Entries: entries}),
	}, true
}

// appendNewEntries implements the freeze-on-write append-only merge (R1/R2): it
// returns the existing (frozen, authoritative) entries verbatim, followed by each
// projected entry whose append key is not already present. The append key is the
// `(FileBase, ChangeID)` pair (R2 — the only identity that survives squash +
// branch-delete, intake Origin #4).
//
// Only ATTRIBUTABLE projected entries (ChangeID != "") participate: an
// unattributable projected entry has no change-id to key on and, per R3, is only
// ever produced at bootstrap / --rebuild (when the existing set is empty), so it
// is appended unconditionally there. The existing set's keys seed the seen-map so
// a re-projection of an already-frozen change is a no-op (idempotence — R1, TC1/TC3).
func appendNewEntries(existing, projected []LogEntry) []LogEntry {
	out := make([]LogEntry, 0, len(existing)+len(projected))
	seen := make(map[string]bool, len(existing))
	for _, e := range existing {
		out = append(out, e)
		if e.ChangeID != "" {
			seen[appendKey(e)] = true
		}
	}
	for _, p := range projected {
		if p.ChangeID != "" {
			key := appendKey(p)
			if seen[key] {
				continue // (file-base, change-id) already frozen → no-op
			}
			seen[key] = true
		}
		out = append(out, p)
	}
	return out
}

// appendKey is the freeze-on-write dedup key: the `(FileBase, ChangeID)` pair
// (R2). The US byte (never present in a file base or a 4-char id) joins the two
// fields unambiguously.
func appendKey(e LogEntry) string {
	return e.FileBase + "\x1f" + e.ChangeID
}

// readSeedEntries reads and parses the folder's `log.seed.md` seed input (FKF §6
// seed-merge). A missing seed file yields no entries (the pure git-projection
// path, unchanged). The seed is read, never written — single-writer discipline.
func readSeedEntries(folderDir string) []LogEntry {
	data, err := os.ReadFile(filepath.Join(folderDir, seedFileName))
	if err != nil {
		return nil
	}
	return parseSeedLog(string(data))
}

// sortLogEntries applies the package's stable log order (newest date first, then
// file base, then change-id) in place — the same comparator gatherLogEntries uses
// for its git-only set, lifted here so the seed-merged set is ordered identically.
func sortLogEntries(entries []LogEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		if entries[i].Date != entries[j].Date {
			return entries[i].Date > entries[j].Date
		}
		if entries[i].FileBase != entries[j].FileBase {
			return entries[i].FileBase < entries[j].FileBase
		}
		return entries[i].ChangeID < entries[j].ChangeID
	})
}

// gatherLogEntries projects the batched commit history for one folder's direct
// topic files into LogEntry values, attributing each commit to a registered
// change (slug/summary fallback per §6.3) and deriving the verb from the
// per-commit name-status. Entries are returned newest-commit-first (git's order),
// with a stable secondary sort (file base then change-id) so same-date entries
// are byte-stable across runs. Only direct topic files are considered — a
// sub-domain's history belongs to the sub-domain's own log.
//
// projectUnattributable gates the unattributable branch (a commit attributeCommit
// cannot resolve to a registry change-id — a direct main edit, pre-FKF history, or
// a squash-merge whose branch token was dropped). Under freeze-on-write (R3) an
// unattributable commit has no change-id to key an append on, so it is projected
// ONLY at bootstrap (the folder has no existing log.md yet) or under --rebuild —
// when projectUnattributable is true. On a normal regeneration with an existing
// log.md it is false, and new unattributable commits are simply not projected
// (the frozen lines already on disk are preserved by the caller's append-only
// merge; re-projecting a squash-reworded subject would otherwise churn the log).
func gatherLogEntries(repoRoot string, dates *gitDates, reg map[string]changeMeta, folderDir, bundleRel string, projectUnattributable bool) []LogEntry {
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
		if !strings.HasSuffix(name, ".md") || name == "index.md" || name == "log.md" || name == seedFileName {
			continue
		}
		base := strings.TrimSuffix(name, ".md")
		bundlePath := "/" + bundleRel + "/" + base + ".md"
		if dates == nil {
			continue // no git history → no projected entries (seed merge still applies upstream)
		}
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
				// squash-merge that dropped the branch token). Under freeze-on-write
				// (R3) it is projected ONLY at bootstrap / --rebuild — when
				// projectUnattributable is true. On a normal regen (existing log.md
				// present) it is dropped: it has no change-id to key an append on,
				// and the frozen line (if any) is already preserved by the caller's
				// append-only merge. When projected, degrade gracefully per FKF §6 —
				// omit the (change-id) and use the commit subject as the descriptive
				// line (still a conflict-free git projection); falls through to the
				// renderer's "—" when even the subject is empty.
				if !projectUnattributable {
					continue
				}
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
	sortLogEntries(entries)
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
		out = append(out, Warning{Path: dir, Kind: KindDepth, Depth: depth})
		return nil
	})
	return out
}

// frontmatterWarnings walks docs/memory/ and returns, per .md file, the
// blocking + advisory findings that need a read-only content pass. The pass is
// read-only and never affects rendered output (byte-stability, intake #3):
//
//   - Description findings (both topic files AND domain/sub-domain index.md
//     stubs — a corrupted/over-cap/change-id-laden domain description mangles
//     the root row exactly as one on a topic file mangles a domain row):
//     malformed-frontmatter (via internal/frontmatter.Validate), the blocking
//     registry-gated change-id finding, and the length findings (advisory
//     501–1000 vs. blocking gross over-cap > 1000, mutually exclusive).
//   - Topic-file BODY findings (topic files only — index.md is a generated
//     stub, never a concept document): narration-marker density, size, and
//     broken bundle-relative links.
//
// log.md / log.seed.md are generated/curated log inputs, never concept
// documents, so they are skipped entirely.
func frontmatterWarnings(memRoot string, reg map[string]changeMeta) []Warning {
	var out []Warning
	_ = filepath.Walk(memRoot, func(p string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(p, ".md") {
			return nil
		}
		name := filepath.Base(p)
		if name == "log.md" || name == seedFileName {
			return nil // generated/curated log inputs — not concept documents
		}
		relPath := filepath.ToSlash(filepath.Join("docs", "memory", relOrBase(memRoot, p)))
		isIndex := name == "index.md"

		// Description findings — inspected on both topic files and index.md stubs,
		// only when the file actually opens a frontmatter block (a body-only file
		// degrades gracefully; the root index.md often carries no description stub).
		if frontmatter.HasFrontmatter(p) {
			for _, f := range frontmatter.Validate(p) {
				switch f.Kind {
				case frontmatter.KindUnclosedFence:
					out = append(out, Warning{Path: relPath, Kind: KindMalformedFence})
				case frontmatter.KindQuoteStripFailure:
					out = append(out, Warning{Path: relPath, Kind: KindMalformedDescription, Detail: f.Detail})
				}
			}
			if desc := frontmatter.Field(p, "description"); desc != "" {
				// Blocking change-id in the description (registry-gated §3.2 ban).
				if ids := scanChangeIDs(desc, reg); len(ids) > 0 {
					out = append(out, Warning{Path: relPath, Kind: KindDescriptionChangeID, Detail: strings.Join(ids, ", ")})
				}
				// Length: gross over-cap (> 1000) BLOCKS; 501–1000 stays advisory —
				// mutually exclusive so a >1000 description is not double-reported.
				if n := utf8.RuneCountInString(desc); n > DescriptionBlockingLenThreshold {
					out = append(out, Warning{Path: relPath, Kind: KindDescriptionOverCap, Count: n})
				} else if n > DescriptionLenWarnThreshold {
					out = append(out, Warning{Path: relPath, Kind: KindDescriptionLength, Count: n})
				}
			}
		}

		// Topic-file BODY findings — index.md is a generated stub, never scanned.
		if !isIndex {
			out = append(out, topicBodyWarnings(memRoot, p, relPath, reg)...)
		}
		return nil
	})
	return out
}

// topicBodyWarnings returns the advisory body findings for one topic file:
// narration-marker density (transition stems + registry-gated change-id tokens,
// fires at ≥ NarrationMarkerWarnThreshold), size (> line OR > byte cap), and
// broken bundle-relative links (`](/...)` targets absent under docs/memory/,
// skipping fenced code blocks). All advisory — none affects the exit code.
// A file that cannot be read yields no findings (graceful degradation).
func topicBodyWarnings(memRoot, p, relPath string, reg map[string]changeMeta) []Warning {
	data, err := os.ReadFile(p)
	if err != nil {
		return nil
	}
	content := string(data)
	body := stripFrontmatter(content)
	var out []Warning

	// Narration-marker density: case-insensitive transition-stem hits + the
	// registry-gated change-id token OCCURRENCES in the body (the debt meter
	// counts sanctioned citations too, and each occurrence — density is the
	// signal, not violation; three cites of one id are three markers).
	markers := countNarrationStems(body) + countChangeIDOccurrences(body, reg)
	if markers >= NarrationMarkerWarnThreshold {
		out = append(out, Warning{Path: relPath, Kind: KindNarrationDensity, Count: markers})
	}

	// Size: > line cap OR > byte cap (either bound). Line count matches `wc -l`
	// (the count of newline bytes), so the reported metric agrees with what an
	// author sees from `wc -l`; a final unterminated line adds 1 (as `wc -l`
	// omits it, but a canonical memory file ends in a trailing newline). Byte
	// size = file bytes.
	nLines := strings.Count(content, "\n")
	if len(content) > 0 && !strings.HasSuffix(content, "\n") {
		nLines++ // count a trailing line with no terminating newline
	}
	if nLines > FileSizeLineWarnThreshold || len(data) > FileSizeByteWarnThreshold {
		out = append(out, Warning{Path: relPath, Kind: KindFileSize, Count: nLines, Bytes: len(data)})
	}

	// Broken bundle-relative links: `](/...)` targets absent on disk under
	// docs/memory/. Code-fenced examples are skipped (documentation, not links).
	for _, tgt := range brokenBundleLinks(memRoot, body) {
		out = append(out, Warning{Path: relPath, Kind: KindBrokenLink, Detail: tgt})
	}
	return out
}

// narrationStems are the case-insensitive transition-narration substrings the
// density meter counts (FKF §3.3 "no transition narration"). "supersed" covers
// supersede/superseded/supersedes.
var narrationStems = []string{"no longer", "previously", "renamed", "supersed"}

// countNarrationStems returns the total case-insensitive substring-hit count of
// every narration stem in body.
func countNarrationStems(body string) int {
	lower := strings.ToLower(body)
	total := 0
	for _, s := range narrationStems {
		total += strings.Count(lower, s)
	}
	return total
}

// stripFrontmatter returns content with a leading `---`-fenced YAML frontmatter
// block removed (so body scans never count frontmatter tokens). A file without
// a leading fence is returned unchanged.
func stripFrontmatter(content string) string {
	lines := strings.Split(content, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return content
	}
	for i := 1; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) == "---" {
			return strings.Join(lines[i+1:], "\n")
		}
	}
	return content // unclosed fence → treat whole thing as body (malformed check owns it)
}

// bundleLinkPattern matches a markdown link whose target begins with `/` (a
// bundle-relative memory↔memory link, FKF §7). Group 1 is the target.
var bundleLinkPattern = regexp.MustCompile(`\]\((/[^)\s]+)\)`)

// inlineCodeSpan matches a markdown inline code span (“ `…` “). Its content is
// documentation shown verbatim (e.g. a log-line format example), never a live
// link, so it is elided before link matching.
var inlineCodeSpan = regexp.MustCompile("`[^`]*`")

// brokenBundleLinks returns the bundle-relative link targets in body that do
// NOT resolve on disk under memRoot, deduplicated in first-seen order. Only
// `/`-prefixed targets are checked (repo-relative and external links are out of
// scope — no false positives on links out of the bundle). Both FENCED code
// blocks (``` ``` ```) and INLINE code spans (“ `…` “) are skipped: this repo's
// own memory docs carry illustrative link-format examples like
// “ `[base](/{domain}[/{sub}]/base.md)` “ and “ `](/bundle/rel.md)` “ inside
// code markup that are not live links (FKF §7 says consumers tolerate broken
// links; this is the author-side nag, not a literal-example linter). A trailing
// `#anchor` is stripped before the on-disk resolve.
func brokenBundleLinks(memRoot, body string) []string {
	var out []string
	seen := map[string]bool{}
	inFence := false
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "```") {
			inFence = !inFence
			continue
		}
		if inFence {
			continue
		}
		// Elide inline code spans so a link shown as a format example inside
		// backticks is not scanned as a live cross-link.
		scan := inlineCodeSpan.ReplaceAllString(line, "")
		for _, m := range bundleLinkPattern.FindAllStringSubmatch(scan, -1) {
			target := m[1]
			if seen[target] {
				continue
			}
			seen[target] = true
			// Strip a trailing #anchor before resolving the path on disk.
			relTarget := target
			if i := strings.IndexByte(relTarget, '#'); i >= 0 {
				relTarget = relTarget[:i]
			}
			relTarget = strings.TrimPrefix(relTarget, "/")
			if relTarget == "" {
				continue // bare "/" or "/#anchor" — not a file target
			}
			if _, statErr := os.Stat(filepath.Join(memRoot, filepath.FromSlash(relTarget))); statErr != nil {
				out = append(out, target)
			}
		}
	}
	return out
}

// relOrBase returns p relative to memRoot in slash form, falling back to the
// base name when the relative computation fails (defensive — Walk always yields
// a path under memRoot in practice).
func relOrBase(memRoot, p string) string {
	if rel, err := filepath.Rel(memRoot, p); err == nil {
		return filepath.ToSlash(rel)
	}
	return filepath.Base(p)
}
