// Package memoryindex implements deterministic navigation for configured
// documentation roots. GatherRoot supplies the root-aware docs-index command;
// RenderRoot and RenderDomain are pure renderers, and Classify guards destructive
// navigation loss. FKF logs remain an opt-in, freeze-on-write projection.
// Gather and GatherLogs retain the memory-shaped API over the same walker.
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

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
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
	// in the body that fall OUTSIDE the §3.3-sanctioned citation positions — the
	// distillation-debt meter). A file reaching this count triggers an advisory
	// warning. Hardcoded shape-bound const, NOT config.
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
	// stems + registry-gated change-id tokens in the body OUTSIDE the sanctioned
	// citation positions — a parenthesized `(id)` citation, an `*Introduced by*:`
	// field line) reaches NarrationMarkerWarnThreshold. ADVISORY — the standing
	// distillation-debt meter. Sanctioned citations are KEPT by distillation, so
	// they are not scored as debt (a distilled file with only allowed citations
	// clears the flag); an id woven into prose still counts.
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
	Label string
	Link  string
	// Base is the file name without the .md extension (the link target stem).
	Base string
	// Title is the file H1, used as the row label when description is missing.
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
	Generic bool // generic roots escape table cells; memory preserves its byte contract
	Link    string
	Note    string
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
	// SubDomains are content-bearing child folders, recursively populated and
	// sorted lexicographically by Name. Empty for a leaf folder.
	SubDomains []DomainData
}

// DomainRow is one row of the root (domains-only) index.
type DomainRow struct {
	Link        string
	Name        string // folder name; link target is {Name}/index.md
	Description string // curated one-liner; "" → missing-cell fallback
}

// RootData is everything RenderRoot needs. Plain value → RenderRoot is pure.
type RootData struct {
	Domains []DomainRow // sorted lexicographically by Name
	// NavNote carries the root's configured docs_index.roots[].nav_note:
	// empty renders nothing, non-empty renders verbatim followed by a blank
	// line (trailing newlines from YAML block scalars are normalized so
	// exactly one blank line follows).
	NavNote string
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
	Limit  int    // per-root advisory depth limit; zero uses the memory default
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
	case KindMissingDescription:
		return fmt.Sprintf("⚠ %s has no description: — using H1 and placeholder", w.Path)
	case KindWidth:
		return fmt.Sprintf("⚠ %s has %d topic files (soft bound: ~%d) — consider splitting into sub-domains",
			w.Path, w.Count, WidthWarnThreshold)
	case KindDepth:
		limit := w.Limit
		if limit == 0 {
			limit = MaxDepth
		}
		return fmt.Sprintf("⚠ %s is nested %d levels deep (max: %d) — consider flattening", w.Path, w.Depth, limit)
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
	b.WriteString("> **Generated by `fab docs-index`** — do not hand-edit. Re-run after any memory write;\n")
	b.WriteString("> the output is byte-stable. Per-file descriptions live in each file's `description:` frontmatter.\n\n")
	if d.NavNote != "" {
		b.WriteString(strings.TrimRight(d.NavNote, "\n") + "\n\n")
	}
	b.WriteString("| Domain | Description |\n")
	b.WriteString("|--------|-------------|\n")
	for _, dr := range d.Domains {
		desc := dr.Description
		if desc == "" {
			desc = missingCell
		}
		link := dr.Link
		if link == "" {
			link = dr.Name + "/index.md"
		}
		fmt.Fprintf(&b, "| [%s](%s) | %s |\n", dr.Name, link, desc)
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
	b.WriteString("> **Generated by `fab docs-index`** — do not hand-edit. Descriptions come from each file's `description:` frontmatter.\n\n")
	b.WriteString("| File | Description |\n")
	b.WriteString("|------|-------------|\n")
	for _, f := range d.Files {
		desc := f.Description
		if desc == "" {
			desc = missingCell
		}
		label := f.Label
		if label == "" {
			label = f.Base
		}
		link := f.Link
		if link == "" {
			link = f.Base + ".md"
		}
		fmt.Fprintf(&b, "| [%s](%s) | %s |\n", renderDescription(label, d.Generic), link, renderDescription(desc, d.Generic))
	}
	if d.Note != "" {
		fmt.Fprintf(&b, "\n%s\n", d.Note)
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
			link := sd.Link
			if link == "" {
				link = sd.Name + "/index.md"
			}
			fmt.Fprintf(&b, "| [%s](%s) | %s |\n", sd.Name, link, renderDescription(desc, d.Generic))
		}
	}
	return b.String()
}

// Gather is the memory-shaped compatibility API over the root-aware walker.
func Gather(repoRoot string) (RootData, []DomainData, []Warning, error) {
	cfg := config.DefaultDocsIndexRoots()[0]
	w := docWalk{repo: repoRoot, root: filepath.Join(repoRoot, "docs", "memory"), cfg: cfg, reg: gatherChangeRegistry(filepath.Join(repoRoot, "fab"))}
	tree, err := w.walk(w.root, false, 0)
	if err != nil {
		return RootData{}, nil, nil, fmt.Errorf("docs/memory not found under %s: %w", repoRoot, err)
	}
	root := RootData{}
	var domains []DomainData
	for _, child := range tree.children {
		domains = append(domains, child.data)
		root.Domains = append(root.Domains, DomainRow{Name: child.data.Name, Description: child.data.Description})
	}
	sortWarnings(w.warnings)
	return root, domains, w.warnings, nil
}

// domainTitle reads the existing domain index.md H1 if present (preserving a
// curated heading), else synthesizes a Title-Case heading from the folder name.
func domainTitle(domainDir, domainName string) string {
	if h1 := readH1(filepath.Join(domainDir, "index.md")); h1 != "" {
		return h1
	}
	return titleCase(domainName) + " Documentation"
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
func loadGitDates(repoRoot string) *gitDates { return loadGitDatesForRoot(repoRoot, "docs/memory") }

func loadGitDatesForRoot(repoRoot, rootPath string) *gitDates {
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
		"--name-status", "--", rootPath)
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

// parenCitationPattern matches a parenthesized `(change-id)` citation — a single
// candidate token wrapped in parentheses with no other content (optional inner
// spaces tolerated, e.g. `( xu0k )`). The captured group is the inner token; the
// caller resolves it against the registry so only a genuine change-id counts as a
// sanctioned citation (a `(see the docs)` group whose inner text has a space is
// not a lone token and never matches). This is the §3.3-sanctioned trailing
// citation form the narration-density meter must NOT count.
var parenCitationPattern = regexp.MustCompile(`\(\s*([^()\s]+)\s*\)`)

// introducedByLinePattern matches a Design-Decisions `*Introduced by*:` field
// line (leading whitespace tolerated). Every change-id on such a line is
// sanctioned provenance (§3.3), so the meter skips the whole line.
var introducedByLinePattern = regexp.MustCompile(`(?i)^\s*\*introduced by\*\s*:`)

// countNonSanctionedChangeIDs returns the number of registry-gated change-id
// token OCCURRENCES that appear OUTSIDE the two §3.3-sanctioned positions:
//   - a parenthesized `(change-id)` trailing citation (both a full
//     YYMMDD-XXXX-slug token and a bare registered 4-char id), and
//   - any change-id on an `*Introduced by*:` Design-Decisions field line.
//
// A change-id woven into prose (outside those positions) still counts — an id
// embedded in narration is a genuine density signal. This inverts the former
// count-everything rule (which counted sanctioned citations too, so a
// fully-distilled file that kept its allowed citations could never clear the
// flag): distillation is told to KEEP exactly these citations, so they must not
// be scored as debt. Occurrences are NOT deduplicated — three prose mentions of
// one id are three markers.
func countNonSanctionedChangeIDs(body string, reg map[string]changeMeta) int {
	n := 0
	for _, line := range strings.Split(body, "\n") {
		// (b) An *Introduced by*: field line — every id on it is sanctioned.
		if introducedByLinePattern.MatchString(line) {
			continue
		}
		// Total resolved change-id occurrences on the line.
		total := 0
		for _, tok := range strings.FieldsFunc(line, changeIDTokenSep) {
			if changeIDTokenID(tok, reg) != "" {
				total++
			}
		}
		// (a) Subtract each lone-parenthesized `(change-id)` citation — the
		// inner token resolves and was the sole content of its parens group.
		sanctioned := 0
		for _, m := range parenCitationPattern.FindAllStringSubmatch(line, -1) {
			if changeIDTokenID(m[1], reg) != "" {
				sanctioned++
			}
		}
		n += total - sanctioned
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
// `fab docs-index`) reads each existing log.md and appends-only; true
// (`fab docs-index --rebuild`) discards the frozen state and re-projects every
// log.md from current git (destructive). --check passes rebuild=false so the
// rendered content is the freeze-on-write merge the classifier byte-compares
// against (R7–R9).
func GatherLogs(repoRoot, fabRoot string, rebuild bool) ([]LogTarget, error) {
	targets, _, err := GatherRoot(repoRoot, fabRoot, config.DefaultDocsIndexRoots()[0], rebuild)
	if err != nil {
		return nil, err
	}
	var logs []LogTarget
	for _, t := range targets {
		if t.IsLog {
			logs = append(logs, LogTarget{Path: t.Path, Content: t.Content})
		}
	}
	sort.Slice(logs, func(i, j int) bool { return logs[i].Path < logs[j].Path })
	return logs, nil
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
func buildLogTarget(repoRoot string, dates *gitDates, reg map[string]changeMeta, folderDir, bundleRel, titleOverride string, rebuild bool, landings ...string) (LogTarget, bool) {
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
	projected := gatherLogEntries(repoRoot, dates, reg, folderDir, bundleRel, bootstrap || rebuild, landings...)

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
func gatherLogEntries(repoRoot string, dates *gitDates, reg map[string]changeMeta, folderDir, bundleRel string, projectUnattributable bool, landings ...string) []LogEntry {
	dirEntries, err := os.ReadDir(folderDir)
	if err != nil {
		return nil
	}
	if len(landings) == 0 {
		landings = []string{"index.md"}
	}
	var entries []LogEntry
	for _, de := range dirEntries {
		if de.IsDir() {
			continue
		}
		name := de.Name()
		if !strings.HasSuffix(name, ".md") || name == "log.md" || name == seedFileName {
			continue
		}
		landing := false
		for _, n := range landings {
			if name == n {
				landing = true
				break
			}
		}
		if landing {
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

// topicBodyWarnings returns the advisory body findings for one topic file:
// narration-marker density (transition stems + registry-gated change-id tokens
// outside the §3.3-sanctioned citation positions, fires at ≥
// NarrationMarkerWarnThreshold), size (> line OR > byte cap), and
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
	// registry-gated change-id token occurrences in the body that fall OUTSIDE
	// the two §3.3-sanctioned positions (a parenthesized `(id)` citation, an
	// `*Introduced by*:` field line). The sanctioned citations are exactly what
	// distillation is told to KEEP, so they are not counted as debt — a
	// fully-distilled file that retains only its allowed citations clears the
	// flag. An id woven into prose still counts (density signal for narrated ids).
	markers := countNarrationStems(body) + countNonSanctionedChangeIDs(body, reg)
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

func sourceWarnings(memRoot, p, relPath string, reg map[string]changeMeta, fkf, isIndex bool) []Warning {
	var out []Warning
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
			if ids := scanChangeIDs(desc, reg); fkf && len(ids) > 0 {
				out = append(out, Warning{Path: relPath, Kind: KindDescriptionChangeID, Detail: strings.Join(ids, ", ")})
			}
			// Length: gross over-cap (> 1000) BLOCKS; 501–1000 stays advisory —
			// mutually exclusive so a >1000 description is not double-reported.
			if n := utf8.RuneCountInString(desc); fkf && n > DescriptionBlockingLenThreshold {
				out = append(out, Warning{Path: relPath, Kind: KindDescriptionOverCap, Count: n})
			} else if n > DescriptionLenWarnThreshold {
				out = append(out, Warning{Path: relPath, Kind: KindDescriptionLength, Count: n})
			}
		}
	}

	// Topic-file BODY findings — index.md is a generated stub, never scanned.
	if !isIndex {
		for _, w := range topicBodyWarnings(memRoot, p, relPath, reg) {
			// Narration and bundle-root link semantics belong to FKF only.
			if fkf || w.Kind != KindNarrationDensity && w.Kind != KindBrokenLink {
				out = append(out, w)
			}
		}
	}

	return out
}

func escapeCell(s string) string {
	return strings.ReplaceAll(strings.ReplaceAll(s, "\n", " "), "|", "\\|")
}

func renderDescription(s string, generic bool) string {
	if generic {
		return escapeCell(s)
	}
	return s
}
