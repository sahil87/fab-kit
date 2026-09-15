# Plan: docs-index — Index Every File Type in Generated Navigation, and Rename the Hand-Managed Block to `manual`

**Change**: 260909-l0bf-docs-index-all-files-manual-block
**Intake**: `intake.md`

## Requirements

### Walker: All Regular Files Are Topics

#### R1: Every regular file is a topic row
`docWalk.isTopic` (`src/go/fab/internal/memoryindex/docsindex.go`) MUST accept every regular file, excluding only: the landing filenames (`index_file` and every `also_accept` name), `log.md` and `log.seed.md` on `log: true` roots, dotfiles (name starts with `.`), symlinks (already skipped in `walk`), and paths matching the root's `exclude` globs (R2). No other extension-based filtering SHALL remain.

- **GIVEN** a root folder containing `a.md`, `b.html`, `c.png`, `.hidden`, and `index.md`
- **WHEN** `fab docs-index` generates the folder's landing
- **THEN** the File table has rows for `a.md`, `b.html`, and `c.png` and no row for `.hidden` or `index.md`

- **GIVEN** a child folder holding only `page.html` (no `.md`) and no existing landing
- **WHEN** the parent is walked
- **THEN** the child has `count == 1`, receives a generated `index_file` landing, and appears in the parent's `## Sub-Domains` table (today it is dropped)

#### R2: Per-root `exclude` glob list
`config.DocsIndexRoot` MUST gain `Exclude []string` (YAML/JSON key `exclude`, `omitempty`, default empty) using the same root-relative slash-glob syntax as `superseded` (`**` spans directories; `*`/`?`/bracket classes within a segment). A file matching any pattern SHALL produce no row and not count; a folder matching any pattern SHALL not be walked (no landing, no sub-domain row, no counts, and no cleanup of stale generated output inside it). A folder whose files are all excluded SHALL be dropped like an empty folder. On a path matching both `exclude` and `superseded`, `exclude` wins. `GatherRoot` MUST validate `exclude` globs up front and the validation error MUST name the correct field (`docs_index.roots.exclude` vs `docs_index.roots.superseded`).

- **GIVEN** `exclude: ["assets/**", "**/*.png"]` and a tree with `assets/cursio/x.png`, `guide/shot.png`, `guide/page.html`
- **WHEN** the root is generated
- **THEN** `assets/` has no landing and no sub-domain row, `guide/` lists only `page.html`, and `guide`'s topic count is 1

- **GIVEN** `exclude: ["/abs/**"]`
- **WHEN** `GatherRoot` runs
- **THEN** it returns an error whose text contains `docs_index.roots.exclude`

- **GIVEN** a folder matching both `exclude` and `superseded`
- **WHEN** the root is generated
- **THEN** the folder is absent from output and contributes nothing to any superseded count

#### R3: Shared glob matcher, no duplication
The glob helpers in `superseded.go` (`matchGlob`, `supersededPath`, `validateGlobs`) MUST be reused for `exclude` — `supersededPath` generalized to a name-neutral `matchAny(patterns, name)` (or equivalent) and `validateGlobs` parametrized by field name — with no second glob implementation.

- **GIVEN** the finished change
- **WHEN** the package is grepped for glob matching logic
- **THEN** exactly one matcher exists and both `superseded` and `exclude` call it

### Walker: Per-Type Metadata

#### R4: Type-dispatched label and description
`readTopic` MUST dispatch on the lower-cased extension. `.md`: unchanged (stem as `Base`; H1 label only when description is missing in a generic root; `description:` frontmatter as description; `Link` defaults to `Base + ".md"`). `.html`/`.htm` (case-insensitive): label from `<title>` text (entities unescaped via `html.UnescapeString`, whitespace collapsed) falling back to the filename; description from `<meta name="description" content="…">` falling back to `—`. Every other type: label is the filename with extension, description `—`, and the file is NOT opened. For every non-markdown file, `FileEntry.Link` MUST be set explicitly to the filename verbatim. Sort order stays lexicographic by link target as today.

- **GIVEN** `page.html` with `<head><title>Left &amp; Right</title><meta content="Panel rewrite" name="description"></head>`
- **WHEN** the folder is generated
- **THEN** the row is `| [Left & Right](page.html) | Panel rewrite |`

- **GIVEN** `page.html` with a `<title>` but no meta description
- **WHEN** generated
- **THEN** the row's description is `—` (the title is never used as description) and the label is the title

- **GIVEN** `diagram.png` beside the above
- **WHEN** generated
- **THEN** the row is `| [diagram.png](diagram.png) | — |` and the PNG is never read

- **GIVEN** an `.HTM` file whose `<title>` is empty
- **WHEN** generated
- **THEN** the label is the filename

#### R5: Stdlib HTML head scan
HTML metadata extraction MUST use only the standard library (no `golang.org/x/net/html`): a case-insensitive scan of the document's `<head>` region — bounded by `</head>` or the first `<body` — for the first `<title>…</title>` and the first `<meta …>` tag carrying `name="description"` in either attribute order with single or double quotes. It MUST NOT rely on a fixed byte prefix.

- **GIVEN** an HTML file with a 60KB inline `<style>` before `<title>` inside `<head>`
- **WHEN** scanned
- **THEN** the title is found

- **GIVEN** `<meta name='description' content='Single quoted'>` and, separately, `<meta content="Order swapped" name="description">`
- **WHEN** scanned
- **THEN** both yield their `content` value

- **GIVEN** a `<title>` appearing only inside `<body>`
- **WHEN** scanned
- **THEN** no title is extracted (label falls back to filename)

#### R6: Advisory scoping
The `KindMissingDescription` advisory MUST fire for `.md` (message unchanged) and for `.html`/`.htm` when the meta description is absent/empty (message wording MUST NOT claim "H1" — e.g. `has no <meta name="description"> — using <title> and placeholder`). It MUST NOT fire for any other type. Advisory kind names and the `--json` `warnings[]` shape SHALL be unchanged.

- **GIVEN** a folder with `a.md` (no frontmatter), `b.html` (no meta), `c.pdf`
- **WHEN** `--check` runs
- **THEN** missing-description warnings exist for `a.md` and `b.html` only

#### R7: Non-markdown files and FKF/frontmatter machinery
Non-markdown topics MUST count toward the folder's topic count (the width advisory). FKF frontmatter validation (`sourceWarnings`) MUST run only for `.md` files. The `log.md` projection is unchanged.

- **GIVEN** a `log: true` root folder holding `x.md` and `y.pdf`
- **WHEN** generated
- **THEN** no frontmatter/FKF warning is emitted for `y.pdf`, and the folder's width counts 2

### Rendering: Sub-Domain Descriptions

#### R8: Folder description round-trip unchanged
A sub-domain row's description MUST come from the generated landing's `description:` frontmatter round-trip as today; a folder holding only non-markdown files starts with `—`. The generator MUST NOT synthesize folder descriptions.

- **GIVEN** a new folder holding only `only.html` and no landing
- **WHEN** generated twice
- **THEN** the parent's Sub-Domains row description is `—` both times and output is byte-identical

### Adoption: The `manual` Block

#### R9: Marker rename with read-both compatibility
The region markers MUST become `<!-- fab docs-index:manual:start -->` / `<!-- fab docs-index:manual:end -->`. `adoption.go` MUST rename `curatedStart`/`curatedEnd` → `manualStart`/`manualEnd`, keep `legacyCuratedStart`/`legacyCuratedEnd` with the old spelling, and rename `curatedNavigation` → `manualNavigation`, which recognizes both spellings (`manual` first, then legacy `curated`). `adoptNavigation` MUST always write the new spelling. No code removes legacy recognition in this change.

- **GIVEN** an existing landing with a `curated` block holding two rows
- **WHEN** regenerated
- **THEN** the output carries the same two rows inside `manual` markers, and no `curated` marker remains

- **GIVEN** the regenerated output from the previous scenario
- **WHEN** `--check` compares it against the pre-rename file
- **THEN** the drift is tier 1 (benign), not tier 2

#### R10: Always emit the block in primary landings
Every primary (`index_file`) landing — the legacy `docs/memory` root rendered by `RenderRoot`, generic roots, domains, sub-domains — MUST carry the manual block even when empty, placed immediately before the first table (after the "Generated by" note and any `nav_note`). Alternate landings (`also_accept`) MUST NOT receive it. When the generator creates a block (landing had none and seed import found nothing), it MUST seed the agent-facing comment:

```
<!-- fab docs-index:manual:start -->
<!-- Hand-managed; preserved verbatim on regeneration. Add rows the generator
     cannot produce: non-markdown files' descriptions (HTML, PDF), external links,
     custom groupings. A row for a file the walker also indexes keeps its place and
     label here, but its description follows that file's frontmatter. A row
     whose target no longer exists is flagged by `fab docs-index --check`. -->
<!-- fab docs-index:manual:end -->
```

When a landing already has a block (either spelling), its content MUST be preserved verbatim (markers renamed only); the comment is not re-emitted into existing blocks.

- **GIVEN** a fresh folder with no landing
- **WHEN** generated
- **THEN** the landing contains the seeded block between the header note and the File table, and generating again yields byte-identical output

- **GIVEN** a `README.md` alternate landing
- **WHEN** generated
- **THEN** its generated block contains no manual markers

- **GIVEN** an existing block containing custom prose and no comment
- **WHEN** regenerated
- **THEN** the prose is unchanged and no comment is inserted

#### R11: Adoption semantics unchanged
`adoptNavigation` row semantics MUST remain: a manual row whose target the walker indexes keeps place and label, its description replaced by the generated one only when the generated one is non-`—`; the generated duplicate row is removed; unknown/external targets pass through; missing local targets are flagged by `tombstoneLosses`; deleting a manual row whose target lacks a source description is a tier-2 loss.

- **GIVEN** a manual row `| [Arch](arch.html) | Hand-written |` and `arch.html` with no meta description
- **WHEN** regenerated
- **THEN** the manual row keeps `Hand-written` and the generated File table has no `arch.html` row

- **GIVEN** the same row but `arch.html` gains `<meta name="description" content="From meta">`
- **WHEN** regenerated
- **THEN** the manual row reads `| [Arch](arch.html) | From meta |`

### Rendering: Header Prose

#### R12: Two-sentence header
`RenderRoot` and `RenderDomain` MUST replace the "do not hand-edit … Descriptions come from each file's `description:` frontmatter" line with a note that (a) keeps the `> **Generated by \`fab docs-index\`**` prefix so `insertNavNote` and `supersededCleanup` anchors still resolve, (b) says everything outside the manual block is regenerated from the tree and each file's own metadata (frontmatter, HTML `<title>`/`<meta name="description">`) — don't edit it, re-run the command, and (c) says the manual block is the one hand-managed region — edit it, it is preserved verbatim. The word "curated" MUST NOT appear in generated output except the memory root's Constitution VI sentence "Specs … are human-curated", which stays.

- **GIVEN** any generated primary landing
- **WHEN** inspected
- **THEN** the header note starts with the `Generated by` prefix and contains both sentences

- **GIVEN** a generic root with `nav_note` configured
- **WHEN** generated
- **THEN** the note lands after the header and before the manual block

### CLI and Config Surface

#### R13: Loss-guard and help wording
`cmd/fab/memory_index.go` MUST reword the tier-2 stderr line to `destructive loss — regenerating would wipe hand-managed/historical content:`, drop "curated" from `Long` help, and list `exclude` among per-root fields. `loss.go` comments drop "curated". JSON `losses[].category` values and `--check` exit tiers (0/1/2) MUST be unchanged.

- **GIVEN** a description wipe on a drift run
- **WHEN** `--check` runs
- **THEN** exit is 2 and stderr contains `hand-managed/historical content`

#### R14: Config registry and docs
`configref.go`'s `docs_index.roots` description MUST list `exclude ([] glob patterns)` next to `superseded` and replace "First-run curated navigation is imported automatically" with manual-block wording; `docsIndexSegment` MUST carry a commented `exclude` example. `src/kit/skills/_cli-fab.md` § fab docs-index, `docs/specs/config.md` § Documentation index roots, and `docs/specs/glossary.md`'s `fab docs-index` row MUST document `exclude`, all-file-types indexing, per-type metadata, advisory scoping, and the manual block (no "curated" block wording).

- **GIVEN** `fab config explain docs_index.roots`
- **WHEN** run
- **THEN** output mentions `exclude` and does not mention "curated"

### Owner-or-Pointer Sweep

#### R15: Skill pointers, not restatements
The manual-block rule is owned by `docs/memory/memory-docs/docs-index.md` (hydrate writes it). `src/kit/skills/docs-hydrate-memory.md` § Index Ownership, `docs-reorg-memory.md`, `docs-distill-memory.md`, and `fab-continue.md` hydrate step 6 MUST each gain a one-sentence pointer that the manual block is the second hand-managed region, owned by the docs-index memory file — without restating its rules. A repo-wide grep (excluding `fab/changes/archive/**`, `.agents/`, `.claude/`) for `docs-index:curated` MUST return only the Go legacy constants and their tests.

- **GIVEN** the finished apply
- **WHEN** `grep -rn 'docs-index:curated' --exclude-dir=archive --exclude-dir=.agents --exclude-dir=.claude .` runs
- **THEN** hits are confined to `src/go/fab/internal/memoryindex/adoption.go` and `*_test.go`

### Dogfood

#### R16: Regenerate this repo's indexes
After the Go change and tests pass, `fab docs-index` MUST be run so `docs/memory/index.md`, every domain/sub-domain index, `docs/specs/index.md`, and `docs/specs/findings/index.md` carry the new header and block; the existing `docs/specs/index.md` block's 18 seed-imported rows MUST be kept, and the agent-facing comment hand-inserted once as its first line. `fab docs-index --check` MUST exit 0 afterwards.

- **GIVEN** the rebuilt binary
- **WHEN** `fab docs-index --check` runs before regeneration
- **THEN** it exits 1 (benign drift), never 2

- **GIVEN** the regenerated indexes
- **WHEN** `fab docs-index --check` runs again
- **THEN** it exits 0

### Non-Goals

- PDF metadata extraction — no library; filename label only.
- Pruning the 18 seed-imported rows in `docs/specs/index.md` — they lack frontmatter descriptions; pruning trips the loss guard. Follow-up.
- Removing legacy `curated` marker recognition — next-minor follow-up.
- Any Constitution VI change — 1.7.0 already permits generated navigation.
- Cleanup of previously generated landings inside a newly `exclude`d subtree — `superseded` owns retirement.
- A migration file for the rename — see Design Decisions.

### Design Decisions

#### Exclude Glob Over a Hardcoded Asset List
**Decision**: The only opt-out from "every file is a row" is the per-root `exclude` glob list, sharing the `superseded` matcher.
**Why**: Keeps the one-sentence mental model (the index is the tree minus what you exclude) and bakes no file-type policy into the binary.
**Rejected**: A built-in list of asset extensions to skip — silently wrong for any repo whose "assets" are documentation.
*Introduced by*: 260909-l0bf-docs-index-all-files-manual-block

#### Title Is a Label, Never a Description
**Decision**: HTML `<title>` fills only the label slot; description comes solely from `<meta name="description">`.
**Why**: Adoption replaces a manual row's description whenever the generated one is non-`—`; a title-as-description would clobber hand-written manual rows on every regeneration.
**Rejected**: Falling back to the title for the description — destroys the manual block's one remaining purpose.
*Introduced by*: 260909-l0bf-docs-index-all-files-manual-block

#### No Migration File for the Marker Rename
**Decision**: Ship read-both compatibility in the binary and no `src/kit/migrations/` entry.
**Why**: Index files are generator output, not user data; the next `fab docs-index` run rewrites them and there is nothing for a user to do.
**Rejected**: A nothing-to-do announce migration (flt3 precedent) — flagged as a judgment call; a one-file addition if review disagrees.
*Introduced by*: 260909-l0bf-docs-index-all-files-manual-block

#### Seed the Agent Comment Only on Block Creation
**Decision**: The agent-facing comment is written when the generator creates a block; existing blocks are preserved verbatim with markers renamed.
**Why**: "Preserved verbatim" is the block's contract; re-emitting would need a fragile sentinel and double-insert once wording is tuned.
**Rejected**: Re-emitting the comment every run.
*Introduced by*: 260909-l0bf-docs-index-all-files-manual-block

#### Stdlib Head Scan Over an HTML Tokenizer
**Decision**: Extract `<title>`/`<meta name="description">` with a bounded stdlib scan of `<head>`.
**Why**: Two tags do not justify `golang.org/x/net/html`; `go.mod` stays at two direct dependencies; a head-bounded scan handles self-contained pages with large inline styles.
**Rejected**: Adding the tokenizer; a fixed byte-prefix scan (misses titles after large `<style>` blocks).
*Introduced by*: 260909-l0bf-docs-index-all-files-manual-block

## Tasks

### Phase 1: Setup

- [x] T001 Add `Exclude []string` (`yaml:"exclude,omitempty" json:"exclude,omitempty"`) with doc comment to `DocsIndexRoot` in `src/go/fab/internal/config/docs_index.go`; add a `config_test`/`docs_index_test.go` case that round-trips `exclude` <!-- R2 -->
- [x] T002 [P] In `src/go/fab/internal/memoryindex/superseded.go`, generalize `supersededPath` → `matchAny(patterns, name)` (update callers) and parametrize `validateGlobs(field string, patterns []string)` so errors name `docs_index.roots.<field>`; add a test asserting the `exclude` field name appears in the error <!-- R3 -->

### Phase 2: Core Implementation

- [x] T003 In `docsindex.go`: `GatherRoot` validates `cfg.Exclude` via the parametrized validator; `walk` skips excluded folders before recursing (no landing, no counts) and skips excluded files before `f.count++`; `isTopic` accepts every regular file except landings, `log.md`/`log.seed.md` on `log: true` roots, and dotfiles <!-- R1 -->
- [x] T004 Add `htmlmeta.go` (package `memoryindex`) with `htmlHeadMeta(path) (title, description string)`: stdlib, case-insensitive, bounded by `</head>`/`<body`, first `<title>`, first `<meta>` with `name="description"` in either attribute order and either quote style, `html.UnescapeString`, whitespace collapsed <!-- R5 -->
- [x] T005 Rewrite `readTopic` in `docsindex.go` as a type dispatch: `.md` path unchanged; `.html`/`.htm` uses `htmlHeadMeta` (title→Label, meta→Description, filename fallback, explicit `Link`); other types filename Label, `—` Description, explicit `Link`, no file read; `sourceWarnings` only for `.md`; `KindMissingDescription` emitted for `.md` and HTML only, with an HTML-specific message variant in `memoryindex.go` <!-- R4 -->
- [x] T006 In `adoption.go`: rename constants to `manualStart`/`manualEnd`, add `legacyCuratedStart`/`legacyCuratedEnd`, rename `curatedNavigation` → `manualNavigation` reading both spellings, `adoptNavigation` always writes `manual` markers <!-- R9 -->
- [x] T007 In `adoption.go` (and `RenderRoot` path in `renderFolder`): when the landing is primary and no block exists after seed import, emit the seeded comment block before the first table; never emit into alternate landings; existing block content preserved verbatim <!-- R10 -->
- [x] T008 In `memoryindex.go`: rewrite the header note in `RenderRoot` and `RenderDomain` to the two-sentence form keeping the `> **Generated by \`fab docs-index\`**` prefix; update `FileEntry`/`RenderDomain` comments ("curated" → neutral); verify `insertNavNote` still anchors <!-- R12 -->

### Phase 3: Integration & Edge Cases

- [x] T009 Update `src/go/fab/cmd/fab/memory_index.go`: tier-2 stderr line → `hand-managed/historical content`, `Long` help drops "curated" and lists `exclude`; update `loss.go` comments <!-- R13 -->
- [x] T010 Update `src/go/fab/internal/configref/configref.go`: `docs_index.roots` description adds `exclude`, replaces curated-import sentence with manual-block wording; `docsIndexSegment` gains a commented `exclude` example; fix registry tests <!-- R14 -->
- [x] T011 Tests in `docsindex_test.go`: non-md topics rows; HTML title/meta extraction (entities, attribute order, quotes, head-bounded, body-title ignored, empty title fallback, title-never-description); PNG not opened; non-md-only folder becomes sub-domain with `—` and byte-stable double generation; `exclude` file/folder/exclude-beats-superseded/all-excluded-dropped; advisory scoping (md+html yes, pdf no); width counts non-md; FKF warnings skipped for non-md on `log: true` <!-- R1 -->
- [x] T012 Tests in `docsindex_test.go`/`loss_test.go`: legacy `curated` block read → `manual` written with rows intact; rename-only drift classifies tier 1; always-emit seeded block in primary landings (root incl. legacy memory root, domain, sub-domain) and absent from `README.md` alternate landing; existing block content verbatim with no re-seeded comment; manual-row description kept when generated is `—` and replaced when meta supplies one; header prose assertions and nav_note ordering <!-- R10 -->
- [x] T013 Update golden/freeze fixtures (`golden_test.go`, `freeze_test.go`, `memoryindex_test.go`, `cmd/fab/*index*_test.go` help/exit-tier tests) to the new header and block; run `go test ./internal/memoryindex/ ./internal/config/ ./internal/configref/ ./cmd/fab/` then `go test ./...` from `src/go/fab` <!-- R12 -->

### Phase 4: Polish

- [x] T014 [P] Docs sweep: `src/kit/skills/_cli-fab.md` § fab docs-index (exclude, all-types paragraph, per-type metadata, advisory scoping, manual block, drop "curated"); `docs/specs/config.md` § Documentation index roots (`exclude` row + example + wording); `docs/specs/glossary.md` `fab docs-index` row <!-- R14 -->
- [x] T015 [P] Pointer sweep: one-sentence manual-block pointer in `src/kit/skills/docs-hydrate-memory.md` § Index Ownership, `docs-reorg-memory.md`, `docs-distill-memory.md`, `fab-continue.md` hydrate step 6; then `grep -rn 'docs-index:curated\|curated region\|curated navigation\|curated rows' --exclude-dir=archive --exclude-dir=.agents --exclude-dir=.claude --exclude-dir=.git .` and fix every non-Go hit outside `docs/memory/` (memory is hydrate's) <!-- R15 -->
- [x] T016 Dogfood: build (`go build ./... && go install` or the repo's dev-binary path so `fab` on PATH is the new build — check `scripts/`), run `fab docs-index --check` (expect exit 1, never 2), run `fab docs-index`, hand-insert the agent comment as the first line inside `docs/specs/index.md`'s manual block, re-run `fab docs-index` (byte-stable) and `fab docs-index --check` (expect exit 0); run `fab sync` <!-- R16 -->

## Execution Order

- T001, T002 first (config field + shared matcher) — T003 depends on both
- T004 before T005 (helper before dispatch)
- T006 before T007 (rename before always-emit)
- T011–T013 after T003–T010; T013 last in Phase 3 (fixtures reflect final output)
- T016 last overall; it needs the built binary and the Phase 4 docs done so the regenerated indexes and `fab sync` pick up everything

## Acceptance

### Functional Completeness

- [x] A-001 R1: Every non-dotfile regular file that is not a landing/log/seed produces a File-table row; a folder holding only non-markdown files becomes a sub-domain with its own landing
- [x] A-002 R2: `exclude` is parsed from config, validated with a field-correct error, and excluded files/folders produce no rows, counts, or landings; `exclude` beats `superseded`
- [x] A-003 R3: One glob matcher serves both `superseded` and `exclude`; `validateGlobs` is parametrized by field name
- [x] A-004 R4: Markdown rows unchanged; HTML rows take label from `<title>` and description from `<meta name="description">`; other types are filename + `—` with explicit `Link` and no file read
- [x] A-005 R5: HTML extraction is stdlib-only, head-bounded, case-insensitive, attribute-order- and quote-agnostic, entity-unescaped
- [x] A-006 R6: Missing-description advisory fires for `.md` and HTML only, with an HTML-specific message; kind names and JSON shape unchanged
- [x] A-007 R7: Non-markdown files count toward width; FKF frontmatter validation runs only on `.md`
- [x] A-008 R8: Folder descriptions still round-trip from landing frontmatter; no synthesized folder text
- [x] A-009 R9: Markers renamed to `manual`; both spellings read; only `manual` written; legacy constants retained
- [x] A-010 R10: The seeded manual block appears in every primary landing (legacy memory root, generic root, domain, sub-domain), never in alternate landings; existing block content preserved verbatim
- [x] A-011 R11: Manual-row place/label/description semantics and tombstone/description loss detection behave as before
- [x] A-012 R12: Header note rewritten in both renderers with the `Generated by` prefix intact; "curated" absent from generated output except the memory root's Constitution VI sentence
- [x] A-013 R13: Tier-2 stderr says `hand-managed/historical content`; help lists `exclude`; exit tiers and `losses[].category` unchanged
- [x] A-014 R14: configref description, `_cli-fab.md`, `config.md`, `glossary.md` document `exclude`, all-types indexing, per-type metadata, advisory scoping, and the manual block
- [x] A-015 R15: Skill pointers added without restating rules; `docs-index:curated` grep hits are confined to `adoption.go` and tests — review note: one additional deliberate hit at `src/kit/skills/_cli-fab.md:1066` documenting the one-version read-both compat window (required for doc accuracy, not a stale block-sense reference); remaining hits are this change's own artifacts, dispatch prompts, the allowed historical migration, and the `dist/` build artifact
- [x] A-016 R16: This repo's indexes regenerated with new header and block; the specs block keeps its 18 rows plus the hand-inserted comment; `--check` exits 0 after regeneration

### Behavioral Correctness

- [x] A-017 R1: A folder that previously vanished (non-markdown only) now appears; a mixed folder no longer renders an empty File table beside an undocumented HTML file
- [x] A-018 R9: Regenerating a `curated`-block landing yields a `manual`-block landing with identical rows and tier-1 drift
- [x] A-019 R12: `insertNavNote` still places `nav_note` after the header and before the manual block

### Scenario Coverage

- [x] A-020 R4: Test covers `<title>` present with no meta → description `—`, label = title
- [x] A-021 R5: Test covers a title after a large inline `<style>` in `<head>` and a title only in `<body>` (ignored)
- [x] A-022 R2: Test covers `exclude` on a folder, on a file glob, on all-files-excluded (dropped), and on a double match with `superseded`
- [x] A-023 R10: Double-generation byte-stability test passes for a landing with a seeded block and for a folder holding only non-markdown files
- [x] A-024 R11: Test covers manual row kept when generated is `—` and replaced when meta supplies a description

### Edge Cases & Error Handling

- [x] A-025 R4: A `|` in an HTML `<title>` is escaped in the table cell via the existing description/label escaping path
- [x] A-026 R4: `.HTM` uppercase extension is treated as HTML; an empty `<title>` falls back to the filename
- [x] A-027 R1: Dotfiles and symlinks never produce rows
- [x] A-028 R2: An invalid `exclude` glob (absolute, `..`, bad bracket) fails `GatherRoot` with an error naming `docs_index.roots.exclude`

### Code Quality

- [x] A-029 Pattern consistency: New code follows naming and structural patterns of `docsindex.go`/`adoption.go` (small functions, table-driven tests)
- [x] A-030 No unnecessary duplication: glob matching, table-cell escaping, and frontmatter reading are reused, not reimplemented
- [x] A-031 Readability over cleverness: the HTML scan is a plain bounded string scan with named helpers, not a regex thicket
- [x] A-032 No god functions: `readTopic` dispatch and `adoptNavigation` remain under ~50 lines each or are split with clear names
- [x] A-033 No magic strings: marker spellings and the seeded comment live in named constants
- [x] A-034 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`; `fab sync` regenerates them
- [x] A-035 CLI ⇒ docs + tests: `_cli-fab.md` § fab docs-index updated and Go tests accompany every `.go` change
- [x] A-036 Owner-or-pointer: skill files point at the docs-index memory file for the manual-block rule rather than restating it
- [x] A-037 Sibling sweep: `_cli-fab.md`, `config.md`, `glossary.md`, and configref wording are all updated in the same change
- [x] A-038 Migration judgment call is visible: the no-migration decision is recorded in Design Decisions and the PR-facing plan, not silent
- [x] A-039 Test integrity: fixtures updated to match the specified output, never the implementation bent to fit a stale fixture

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Memory files (`docs/memory/**`) are hydrate's to write; apply touches them only via the `fab docs-index` regeneration in T016.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. `supersededPath` was generalized to `matchAny` in place (no orphan left behind), and the old header strings were replaced, not duplicated. One forward-looking candidate, already a recorded follow-up (intake Open Questions (b), Non-Goals):

- `src/go/fab/internal/memoryindex/adoption.go:15-16` (`legacyCuratedStart`/`legacyCuratedEnd` and the legacy branch in `manualNavigation`) — legacy `curated` marker recognition becomes deletable in the next minor after this ships; deliberately retained this version for read-both compatibility.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | New helper file `htmlmeta.go` in `internal/memoryindex` rather than inlining in `docsindex.go` | Keeps `docsindex.go` readable; file naming follows `superseded.go`/`seed.go` precedent; trivially movable | S:60 R:90 A:85 D:80 |
| 2 | Confident | `supersededPath` generalized to `matchAny`; name is a plan choice, apply may pick an equivalent neutral name | Intake requires a shared matcher; the exact identifier is cosmetic | S:60 R:95 A:90 D:85 |
| 3 | Confident | The seeded comment is emitted by the same code path (`adoptNavigation`) for `RenderRoot` and `RenderDomain` output, since both flow through `renderFolder`'s primary-landing branch | Verified `renderFolder` routes every primary landing through `adoptNavigation`; alternate landings take the `replaceGeneratedBlock` branch | S:70 R:90 A:85 D:80 |
| 4 | Confident | HTML-specific advisory text is a message variant keyed off the path extension inside `Warning.String()`, not a new `Kind` | Intake fixes kind names and JSON shape as unchanged; a variant message is the smallest additive change | S:65 R:90 A:85 D:80 |
| 5 | Confident | Dogfood binary comes from `go install` / the repo's dev-binary convention discovered in `scripts/` at apply time | The exact build step is repo-local knowledge the apply worker resolves; either path yields the new `fab` on PATH | S:55 R:85 A:80 D:75 |
| 6 | Certain | Memory files are not edited during apply except via the `fab docs-index` regeneration | Pipeline contract: hydrate owns `docs/memory/**` content; the intake's Affected Memory list feeds hydrate | S:90 R:95 A:95 D:95 |

6 assumptions (1 certain, 5 confident, 0 tentative).
