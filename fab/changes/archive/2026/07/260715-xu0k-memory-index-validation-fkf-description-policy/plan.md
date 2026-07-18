# Plan: Memory-Index Frontmatter Validation + Generated-Index Merge Policy + FKF Description-Length Policy

**Change**: 260715-xu0k-memory-index-validation-fkf-description-policy
**Intake**: `intake.md`

## Requirements

### Frontmatter: Malformed-Frontmatter Detection Primitives

#### R1: Structured frontmatter validation in `internal/frontmatter`
The `internal/frontmatter` package MUST expose a validation primitive (a `Validate`-style
function) that reads a file's leading YAML frontmatter and returns structured findings for the two
malformed signatures, sharing the same line-based grammar as `Field`/`HasFrontmatter` so the parser
and its validator cannot diverge. `Field`'s return value MUST remain byte-identical to today (a
malformed value keeps rendering exactly as it does now) — validation is a separate read, never a
mutation of extraction behavior.

- **GIVEN** a memory file that opens with `---` on line 1 but has no subsequent standalone `---`
  line (an unclosed frontmatter block)
- **WHEN** the validator runs
- **THEN** it reports an "unclosed frontmatter fence" finding for that file
- **AND** `Field(path, "description")` returns exactly what it returns today (unchanged)

#### R2: Quote-strip-failure detection (the glued-fence signature)
The validator MUST detect a `description:` value that begins with a quote character (`"` or `'`) but
fails `stripQuotes` (does not end with the matching quote) — the loom glued-fence signature, e.g. a
value ending in `"---`.

- **GIVEN** a memory file whose frontmatter reads `description: "text"---` glued on one line with no
  closing fence and no trailing newline (the loom corruption, verbatim shape)
- **WHEN** the validator runs
- **THEN** it reports BOTH the unclosed-fence finding (the glued fence removed the closing `---`)
  AND the quote-strip-failure finding on `description:`
- **AND** clean quoted/unquoted/empty `description:` values report no finding

### MemoryIndex: New Warning Kinds + Gather-Path Validation

#### R3: Malformed-frontmatter warnings surfaced by `internal/memoryindex`
`internal/memoryindex.Gather` MUST run the R1/R2 validator over every topic file and every
domain/sub-domain `index.md` stub it reads for descriptions, and surface each finding as a new
`Warning` (new `Kind` values with `Warning.String()` cases) sorted into the existing deterministic
warning order. Rendered index output MUST stay byte-identical to today with and without warnings
(the byte-stability hard constraint, intake Assumption #3).

- **GIVEN** a `docs/memory/` tree with one topic file carrying malformed frontmatter
- **WHEN** `Gather` runs
- **THEN** the returned warnings slice includes a malformed-frontmatter warning naming that file
- **AND** the rendered `RenderRoot`/`RenderDomain` output is byte-identical to the same tree without
  the malformed value in place (the corrupted value still renders verbatim into the row)

#### R4: Advisory over-length description warning
`internal/memoryindex` MUST emit an advisory `Warning` (new `Kind`, e.g. `description-length`) for
any `description:` value whose length exceeds a hardcoded package constant (500 characters),
naming the file and the observed length. The threshold is a package const in the shape-bound-const
pattern (like `WidthWarnThreshold`) — NOT config-overridable in this change. Length is measured in
characters (Unicode runes) on the quote-stripped value.

- **GIVEN** a topic file whose `description:` value is 501 characters after quote-stripping
- **WHEN** `Gather` runs
- **THEN** the warnings slice includes a `description-length` warning naming the file and length 501
- **AND** a 500-character description emits no length warning (boundary: strictly greater than 500)
- **AND** the length warning is advisory only — it does not, on its own, affect `--check` exit codes

### CmdFab: `--check` Blocking + Stderr Surfacing

#### R5: Malformed frontmatter blocks `--check` independent of drift
`fab memory-index --check` MUST exit non-zero when any target file carries malformed frontmatter
(R1/R2), **even when every rendered index is byte-identical to its committed form** (the drift
comparison alone provably exits 0 on the loom case). The check MUST run independent of the byte-drift
tier comparison. The human-readable output MUST enumerate the offending file(s).

- **GIVEN** a tree whose committed indexes are byte-clean (drift tier 0) but whose source frontmatter
  is malformed on ≥1 file
- **WHEN** `fab memory-index --check` runs
- **THEN** the process exits with a non-zero (blocking) code and stderr enumerates the offending
  file(s) with a fix-the-frontmatter remediation
- **AND** `--check --json` remains a parseable object whose existing keys (`tier`, `drift`,
  `losses`) are unchanged for existing consumers

#### R6: Exit-code integration keeps all existing consumer contracts
The chosen integration MUST keep: CI/staleness callers (exit ≥ 1 = fail); the hydrate/reorg
refuse-before-regen guards (fire only on exit == 2 destructive loss); `--json` consumers (branch on
`tier` / read the `losses` category enum `description|tombstone|grouping`); and non-`--check` (write)
runs never blocked by warnings. Malformed frontmatter is a corruption in *source* files, orthogonal
to the index-drift tier scheme.

- **GIVEN** a born-FKF fab-kit tree with no malformed frontmatter and only benign index drift
- **WHEN** `fab memory-index --check` runs
- **THEN** it exits 1 (benign drift) exactly as today — malformed detection did not perturb the tier
- **AND** a tree with destructive loss (tier 2) still exits 2 and still fires the guards
- **AND** the `losses[]` category enum is NOT extended with a malformed category (guards keyed on
  `tier == 2` never fire on mere source corruption)

#### R7: Advisory length warning never fails `--check`
An over-length `description:` (R4) MUST NOT, on its own, cause `--check` to exit non-zero (the
deliberate asymmetry: corruption blocks, over-length nags). It prints to stderr on both write and
`--check` runs like the other advisory shape warnings.

- **GIVEN** a tree that is drift-clean (tier 0) and free of malformed frontmatter but carries a
  description over the 500-char cap
- **WHEN** `fab memory-index --check` runs
- **THEN** it exits 0 and prints the advisory length warning to stderr

### Docs: FKF Normative Policy (Both fkf.md Files)

#### R8: FKF §3.2 description-length policy
`docs/specs/fkf.md` §3.2 AND `src/kit/reference/fkf.md` §3.2 MUST both state: `description:` is a
one-line index-row summary capped at 500 characters (unit: characters, measured on the quote-stripped
single-line scalar); detail belongs in the memory-file body, not the description. Both files MUST be
updated in the same change (the fkf.md header's dual-file rule, intake Assumption #5).

- **GIVEN** the FKF spec and its shipped extract
- **WHEN** the description-length policy lands
- **THEN** §3.2 in both files carries the 500-char one-liner cap as normative prose
- **AND** `grep -c` of the cap statement finds it in both files

#### R9: FKF §5 generated-index merge policy (regenerate, never hand-merge)
`docs/specs/fkf.md` §5 AND `src/kit/reference/fkf.md` §5 MUST both carry normative prose: on any
merge conflict in a generated `docs/memory/**/index.md` or `log.md`, agents MUST NOT hand-merge;
the procedure is (1) resolve conflicts in the *topic files* (and `.status.yaml`/seed inputs) only,
(2) re-run `fab memory-index`, (3) take its output wholesale. A non-normative `.gitattributes`
merge-driver recipe is documented as an aside (documentation only — NOT auto-installed, no migration).

- **GIVEN** a merge conflict in a generated index/log file
- **WHEN** an agent consults the FKF policy
- **THEN** §5 in both files states the never-hand-merge / resolve-topic-file / regenerate procedure
- **AND** the `.gitattributes` recipe is present as a clearly non-normative aside

### KitSkills: `_cli-fab.md` + Operational Merge/Length Pointers

#### R10: `_cli-fab.md` documents the new CLI behavior
`src/kit/skills/_cli-fab.md` § fab memory-index MUST document: the new malformed-frontmatter stderr
warning line(s), the new advisory `description-length` warning line, and the new `--check` blocking
semantics for malformed frontmatter (its exit code + how it relates to the existing tier scheme).
`docs/specs/skills/SPEC-_cli-fab.md` MUST be updated per the mirror rule.

- **GIVEN** the CLI behavior change
- **WHEN** `_cli-fab.md` is read
- **THEN** its § fab memory-index describes the malformed-frontmatter and length warnings and the
  malformed-frontmatter `--check` blocking behavior with its exit code
- **AND** SPEC-_cli-fab.md's memory-index coverage reflects the change

#### R11: Merge-policy operational pointers at the skill seams
The skills where agents meet generated-file merges MUST carry a short never-hand-merge pointer to
the FKF §5 policy: `src/kit/skills/git-pr.md` (its Step 3 index-refresh sub-step), and
`src/kit/skills/git-pr-review.md` (its PR-feedback conflict-resolution path). The hydrate seams
(`src/kit/skills/fab-continue.md` Hydrate Behavior, `src/kit/skills/docs-hydrate-memory.md`) MUST
carry at most a one-line cross-reference to the FKF policy (they already route all index writes
through `fab memory-index`). Every touched skill's `docs/specs/skills/SPEC-*.md` mirror MUST be
updated (whole mirror class, intake Assumption #6 / #11).

- **GIVEN** a merge conflict encountered during ship or PR-review
- **WHEN** the agent follows the skill
- **THEN** git-pr.md / git-pr-review.md point to the never-hand-merge FKF §5 procedure
- **AND** SPEC-git-pr.md and SPEC-git-pr-review.md mirror the pointers

#### R12: Description-length policy wired into every hydrate-time authoring seam
The 500-char cap MUST be read every time hydrate authors a description. The seams:
`src/kit/templates/memory.md` (the `description:` placeholder/guidance), `src/kit/skills/fab-continue.md`
Hydrate Behavior Step 4 (the curated-`description:`-one-liner bullet), and
`src/kit/skills/docs-hydrate-memory.md` (ingest Step 3/4, generate Step 3, backfill Step 2 authoring
steps) MUST each gain the cap rule or the §3.2 citation. Every touched skill's SPEC mirror MUST be
updated.

- **GIVEN** a hydrate run authoring a new/edited memory-file description
- **WHEN** the agent reads the template / fab-continue Hydrate Step 4 / docs-hydrate-memory authoring
  steps
- **THEN** each states or cites the 500-char one-liner cap
- **AND** SPEC-fab-continue.md and SPEC-docs-hydrate-memory.md mirror the wiring

### Non-Goals

- The one-time loom data fix is OUT of scope (handled separately via an operator message —
  intake Assumption #2).
- No `.status.yaml`/config schema restructuring ⇒ **no migration file** (validation is read-only;
  the length threshold is a hardcoded const, not a config field — intake Assumption #10 / Impact).
- The `.gitattributes` merge-driver is documented only, never auto-installed.
- `docs/memory/**` content edits (the four Affected-Memory files) are a **hydrate-stage** concern,
  not apply — apply does not hydrate memory (pipeline: apply → review → hydrate).

### Design Decisions

1. **Exit-code integration: malformed frontmatter is a distinct blocking signal, NOT a new
   destructive-loss (tier-2) category, and NOT folded into the drift tier.** (Resolves intake
   Assumption #9 — the delegated design point.) The `LossReport` gains a separate `Malformed []MalformedFinding`
   field (additive to the existing `tier`/`drift`/`losses` JSON). `emitCheckReport` blocks (returns a
   drift-style error → exit 1) when malformed findings exist, regardless of `Tier` — so a tier-0
   drift-clean tree with corruption exits 1, and a tier-2 tree with corruption still exits 2 (highest
   blocking code wins; malformed floors the exit at 1). *Why not tier 2*: tier 2 means "regen would
   *wipe* curated/historical content" with the `→ /docs-reorg-memory` remediation and it fires the
   hydrate/reorg refuse-before-regen guards; malformed frontmatter is a *different* problem (fix the
   source file) with a *different* remediation, and folding it into tier 2 would (a) pollute the
   `losses[]` category enum `/docs-reorg-memory` consumes, routing corruption to a reorg that cannot
   fix it, and (b) break the load-bearing "born-FKF tree is provably never tier 2 / guards are no-ops"
   invariant. *Why not silently a tier-1 alias*: benign drift auto-heals on regen; corruption does
   not — the whole point is that `--check` must FAIL on it independent of drift. So a distinct
   `malformed` signal with its own remediation string, floor-1 exit, and additive `--json` field is the
   integration that keeps CI (≥1 ✓), guards (==2, index-only ✓), and `--json` consumers (unchanged
   keys ✓) all compatible. — *Rejected*: plain exit-1 via a new tier value, or a fourth
   `LossCategory`.
2. **Length measured in runes, cap = 500.** `utf8.RuneCountInString` on the quote-stripped value;
   `>` (strictly greater) trips the warning, matching the `> WidthWarnThreshold` shape-bound
   convention. — *Why*: intake fixes the unit as characters (Clarification #14); runes are the honest
   character count for non-ASCII descriptions.
3. **Validation runs inside `Gather`, findings flow as `Warning` values; the cmd separates the
   blocking subset.** The malformed warnings are `Warning`s (uniform stderr surfacing with the shape
   warnings) but the cmd's `--check` branch also feeds the malformed findings into the `LossReport`
   so the exit code can block on them — one gather pass, two consumers (stderr line + exit gate),
   mirroring how drift already serves both the write path and `--check`. — *Rejected*: a second
   filesystem walk purely for the blocking check.

## Tasks

### Phase 1: Frontmatter validation primitives (Go)

- [x] T001 Add a `Validate(filePath) []Finding` (or equivalent structured-findings) primitive to `src/go/fab/internal/frontmatter/frontmatter.go`: reuse the `lines.ReadFileLines` + line-1-`---` + closing-`---`-scan grammar of `Field`; detect (a) unclosed fence (opens `---`, no subsequent standalone `---`) and (b) `description:` value that starts with a quote but fails the `stripQuotes` matching-quote check. Return a typed finding per signature (kind + optional detail). Do NOT change `Field`/`HasFrontmatter`/`stripQuotes` behavior. <!-- R1 R2 -->
- [x] T002 Add `src/go/fab/internal/frontmatter/frontmatter_test.go` cases: the glued-fence regression fixture (`description: "text"---` on one line, no closing fence, no trailing newline — the loom corruption verbatim); an unclosed-fence fixture; clean quoted/unquoted/single-quoted/empty `description:` produce no finding; assert `Field` still returns unchanged values on all fixtures (byte-identical extraction). <!-- R1 R2 -->

### Phase 2: MemoryIndex warning kinds + gather validation (Go)

- [x] T003 In `src/go/fab/internal/memoryindex/memoryindex.go`: add the new `Warning.Kind` values (malformed-fence, malformed-description-quote, and `description-length`) with `Warning.String()` cases; add a `DescriptionLenWarnThreshold` package const (500) beside `WidthWarnThreshold`; extend the `Warning` struct only as needed (reuse `Count` for the observed length, or add a field) keeping existing width/depth rendering unchanged. <!-- R3 R4 -->
- [x] T004 In `Gather` (and a small helper), run the frontmatter validator over every topic file and every domain/sub-domain `index.md` read for descriptions; append malformed-frontmatter warnings; measure the quote-stripped `description:` length (runes) and append a `description-length` warning when `> DescriptionLenWarnThreshold`. Sort into the existing deterministic warning order (extend the sort comparator for the new kinds). Rendered output MUST stay byte-identical. <!-- R3 R4 -->
- [x] T005 Add `src/go/fab/internal/memoryindex/memoryindex_test.go` cases: malformed frontmatter on a topic file yields a warning naming the file; the length warning fires at 501 runes and not at 500 (boundary); the rendered `RenderRoot`/`RenderDomain` output is byte-identical with and without the malformed/over-length value (golden byte-stability); warnings are gathered in deterministic sorted order. <!-- R3 R4 -->

### Phase 3: cmd `--check` blocking + surfacing (Go)

- [x] T006 In `src/go/fab/internal/memoryindex/loss.go`: add a `Malformed []MalformedFinding` field to `LossReport` (json tag `malformed`, initialized non-nil so `--json` is always `[]`), a `MalformedFinding{Path, Kind, Detail}` type, and a way for the cmd to populate it from the gathered malformed warnings. Do NOT extend `LossCategory` / the `losses` enum. Keep `Tier`/`Drift`/`Losses` semantics unchanged. <!-- R5 R6 -->
- [x] T007 In `src/go/fab/cmd/fab/memory_index.go`: (a) always print every gathered warning to stderr on both write and `--check` paths (unchanged for width/depth; now includes malformed + length); (b) in the `--check` branch, populate `report.Malformed` from the gathered malformed warnings; (c) in `emitCheckReport`, when `report.Malformed` is non-empty, enumerate the offending files to stderr (non-`--json`) with a fix-the-frontmatter remediation distinct from the `→ /docs-reorg-memory` pointer, and floor the exit at 1 (block) even when `Tier == 0`; tier-2 still exits 2. Update the cobra `Long` help + `--check` flag descriptions for the new blocking class. <!-- R5 R6 R7 R10 -->
- [x] T008 Add `src/go/fab/cmd/fab/memory_index_test.go` cases: `--check` exits non-zero on a tree whose committed indexes are byte-clean (tier 0) but whose source frontmatter is malformed; `--check --json` stays a parseable object with unchanged `tier`/`drift`/`losses` keys plus the new `malformed` array; an over-length-only tree (`--check`) exits 0 with a stderr length warning (advisory asymmetry); a clean born-FKF tree `--check` still exits 0. <!-- R5 R6 R7 -->

### Phase 4: FKF spec + shipped extract (docs)

- [x] T009 Update `docs/specs/fkf.md` §3.2 AND `src/kit/reference/fkf.md` §3.2 with the 500-char one-liner cap (unit: characters, measured on the quote-stripped single-line scalar; detail belongs in the body; advisory-only warning). Keep the two files' §3.2 normative content in lockstep. <!-- R8 -->
- [x] T010 Update `docs/specs/fkf.md` §5 AND `src/kit/reference/fkf.md` §5 with the never-hand-merge normative procedure (resolve topic files → re-run `fab memory-index` → take output wholesale) covering both `index.md` and `log.md`; add the non-normative `.gitattributes` merge-driver recipe as a documentation-only aside (not auto-installed). Keep both files in lockstep. <!-- R9 -->

### Phase 5: Kit skills + SPEC mirrors (docs)

- [x] T011 Update `src/kit/skills/_cli-fab.md` § fab memory-index: add the malformed-frontmatter stderr warning line(s), the advisory `description-length` warning line, and the malformed-frontmatter `--check` blocking semantics (its exit code + relationship to the tier scheme). Update `docs/specs/skills/SPEC-_cli-fab.md`'s memory-index coverage. <!-- R10 -->
- [x] T012 Add the never-hand-merge FKF §5 pointer to `src/kit/skills/git-pr.md` (Step 3 / 3a-bis index-refresh) and `src/kit/skills/git-pr-review.md` (PR-feedback conflict path); update `docs/specs/skills/SPEC-git-pr.md` and `SPEC-git-pr-review.md`. <!-- R11 -->
- [x] T013 Add the description-length cap wiring to `src/kit/templates/memory.md` (the `description:` placeholder/guidance), `src/kit/skills/fab-continue.md` Hydrate Behavior Step 4, and `src/kit/skills/docs-hydrate-memory.md` (ingest Step 3/4, generate Step 3, backfill Step 2); add a one-line FKF-policy cross-reference for merge at the hydrate seams (fab-continue Hydrate + docs-hydrate-memory). Update `docs/specs/skills/SPEC-fab-continue.md` and `SPEC-docs-hydrate-memory.md`. <!-- R11 R12 -->

### Phase 6: Verify + sweep

- [x] T014 Run `go build ./...` and `go test ./...` scoped to `internal/frontmatter`, `internal/memoryindex`, `cmd/fab` (widen to the full Go module if cross-cutting); confirm green. <!-- R1 R2 R3 R4 R5 R6 R7 -->
- [x] T015 Mirror-class sweep: grep the repo for every `docs/specs/skills/SPEC-*.md` mirror of a touched skill and confirm each touched skill (`_cli-fab`, `git-pr`, `git-pr-review`, `fab-continue`, `docs-hydrate-memory`) has its SPEC updated; confirm fkf.md ↔ reference/fkf.md §3.2/§5 parity by grep. <!-- R8 R9 R10 R11 R12 -->

## Execution Order

- Phase 1 (T001) blocks Phase 2 (T003/T004 call the validator).
- Phase 2 blocks Phase 3 (the cmd consumes the gathered warnings + `LossReport`).
- Phases 4 and 5 (docs) are independent of the Go phases and of each other, but T011–T013 depend on
  the exit-code decision (Design Decision 1) being final — which it is (recorded in ## Assumptions).
- T014 runs after Phases 1–3; T015 runs after Phases 4–5.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal/frontmatter` exposes a structured validation primitive sharing `Field`'s grammar; `Field`/`HasFrontmatter`/`stripQuotes` behavior is unchanged.
- [x] A-002 R2: the validator detects both the unclosed-fence and the quote-strip-failure (glued-fence) signatures; the loom `description: "text"---` fixture reports both.
- [x] A-003 R3: `Gather` surfaces malformed-frontmatter findings as sorted `Warning`s; rendered index output is byte-identical with/without the malformed value.
- [x] A-004 R4: an advisory `description-length` warning (threshold const 500, runes) fires for over-length descriptions, naming file + length; it does not affect exit codes on its own.
- [x] A-005 R5: `fab memory-index --check` exits non-zero on a byte-clean tree with malformed source frontmatter, and stderr enumerates the offending file(s).
- [x] A-006 R8: FKF §3.2 in BOTH `docs/specs/fkf.md` and `src/kit/reference/fkf.md` carries the 500-char one-liner cap.
- [x] A-007 R9: FKF §5 in BOTH files carries the never-hand-merge / regenerate procedure and the non-normative `.gitattributes` aside.
- [x] A-008 R10: `_cli-fab.md` § fab memory-index documents the two new warnings + the `--check` blocking behavior; SPEC-_cli-fab.md mirrors it.
- [x] A-009 R11: git-pr.md and git-pr-review.md carry the never-hand-merge pointer; their SPEC mirrors are updated.
- [x] A-010 R12: templates/memory.md, fab-continue.md Hydrate Step 4, and docs-hydrate-memory.md authoring steps carry the 500-char cap; their SPEC mirrors are updated.

### Behavioral Correctness

- [x] A-011 R6: a benign-drift tree still exits 1, a destructive-loss tree still exits 2 and still fires the refuse-before-regen guards, and the `losses[]` category enum is NOT extended — malformed detection does not perturb the drift tier.
- [x] A-012 R7: an over-length-only, drift-clean, malformed-free tree exits 0 under `--check` (the corruption-blocks / over-length-nags asymmetry holds).
- [x] A-013 R5: `--check --json` stays a parseable object; existing `tier`/`drift`/`losses` keys unchanged; new `malformed` array additive.

### Scenario Coverage

- [x] A-014 R1 R2 R3 R4 R5 R6 R7: `go test ./...` for `internal/frontmatter`, `internal/memoryindex`, `cmd/fab` passes, including the loom glued-fence regression fixture, the length boundary (500 vs 501), and the `--check` byte-clean-but-corrupt integration test.

### Edge Cases & Error Handling

- [x] A-015 R2: clean quoted/unquoted/single-quoted/empty `description:` values produce no malformed finding (no false positives).
- [x] A-016 R4: the 500-char boundary is exact (`>` 500 warns, `== 500` does not); non-ASCII descriptions count runes, not bytes.

### Code Quality

- [x] A-017 Pattern consistency: new code follows the package's existing patterns — pure functions in `internal/*` (RenderRoot/Gather/Classify split), stderr warnings via `Warning.String()`, the `os.Exit` non-1 pattern in the cmd, the shape-bound-const convention for the threshold.
- [x] A-018 No unnecessary duplication: the validator reuses `lines.ReadFileLines` + the `Field` grammar rather than reimplementing frontmatter scanning; the `--check` blocking reuses the single gather pass rather than a second filesystem walk.
- [x] A-019 Canonical source only: all kit edits are under `src/kit/` (never `.claude/skills/`); every `src/kit/skills/*.md` change carries its `docs/specs/skills/SPEC-*.md` mirror update.
- [x] A-020 CLI ⇒ docs + tests: the `fab memory-index` behavior change updates `_cli-fab.md` and ships tests in-change.

### Documentation Accuracy

- [x] A-021 Any FKF normative edit updated BOTH `docs/specs/fkf.md` and `src/kit/reference/fkf.md` in lockstep (§3.2 and §5); section anchors preserved.

### Cross References

- [x] A-022 The whole SPEC-mirror class for every touched skill is swept (not only files carrying the literal changed phrase); fkf.md ↔ reference/fkf.md parity verified by grep.

## Notes

- Check items as you review: `- [x]`
- Memory-file updates (`memory-docs/templates.md`, `memory-docs/hydrate.md`, `pipeline/schemas.md`, `pipeline/execution-skills.md`) happen during **hydrate**, not apply — they are listed in the intake's Affected Memory for the hydrate stage.

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The validation/warning surfaces are additive: the drift-tier classifier, the shape warnings, and the `Field` extraction path all remain load-bearing; the `"width"`/`"depth"` string literals were converted to named constants in place, leaving no orphaned code.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Malformed frontmatter is a **distinct blocking signal** (a `Malformed []` field on `LossReport`, floor-1 exit even at tier 0), NOT a new tier-2 destructive-loss category and NOT a `losses[]` category — this is the delegated exit-code integration decision (intake #9) | Keeps CI (≥1), the refuse-before-regen guards (==2, index-only), and `--json` consumers (unchanged `tier`/`losses` enum) all compatible; tier 2's remediation (`→ /docs-reorg-memory`) and its "born-FKF never tier 2 / guards no-op" invariant would both break if corruption were folded in; benign-drift (tier 1) auto-heals on regen whereas corruption must FAIL independent of drift | S:60 R:55 A:65 D:55 |
| 2 | Certain | Length cap = 500 characters, measured as runes on the quote-stripped value, `>` trips the warning | Fixed by intake Clarification #14 / Assumption #14 (delegated user decision); runes are the honest char count; `>` matches the `> WidthWarnThreshold` convention | S:95 R:75 A:90 D:85 |
| 3 | Certain | Byte-stability of rendered index output is inviolable; validation is stderr + exit-code only; `Field` extraction unchanged | Intake Assumption #3 (user-stated hard constraint) + verified against the package's advisory-warning discipline | S:90 R:75 A:95 D:90 |
| 4 | Certain | Advisory length warning never fails `--check`; malformed frontmatter always fails it (the asymmetry) | Intake Assumption #4 (adopted recommendation) | S:90 R:70 A:90 D:85 |
| 5 | Certain | No migration file (read-only validation; hardcoded threshold const, not a config field) | Intake Impact / Assumption #10 — no `.status.yaml`/config restructuring | S:90 R:80 A:95 D:90 |
| 6 | Certain | Memory-file content edits are a hydrate-stage concern, excluded from apply | Pipeline design (apply → review → hydrate); intake's Affected Memory feeds hydrate | S:90 R:70 A:95 D:90 |
| 7 | Confident | Malformed detection = two signatures: (a) unclosed fence; (b) `description:` starting with a quote that fails quote-stripping (glued-fence, e.g. trailing `"---`) | Intake Assumption #8, derived from the verified loom corruption + parser mechanics | S:80 R:70 A:85 D:70 |
| 8 | Confident | The validator lives in `internal/frontmatter` (shares `Field`'s grammar) and findings flow through `internal/memoryindex` as new `Warning` kinds; the cmd separates the blocking subset into `LossReport.Malformed` | Intake "detection primitives belong in internal/frontmatter … memoryindex surfaces them as new Warning entries"; reuses the single gather pass | S:70 R:70 A:85 D:70 |
| 9 | Confident | Threshold is a hardcoded package const (`DescriptionLenWarnThreshold = 500`) beside `WidthWarnThreshold`, not config-overridable | Intake Assumption #10 — matches the existing shape-bound-const pattern; config promotion would drag in the config registry un-asked | S:70 R:80 A:85 D:75 |
| 10 | Confident | Merge-policy home = FKF §5 (+ shipped extract); operational pointers in git-pr.md (ship index-refresh) + git-pr-review.md (PR-feedback conflict); hydrate skills get a one-line cross-reference | Intake Assumption #11 — verified no conflict-handling prose exists today; these are the seams where agents meet merges | S:60 R:80 A:65 D:55 |
| 11 | Confident | `.gitattributes` merge-driver recipe is documentation-only (non-normative aside in fkf.md §5), not auto-installed, no migration | Intake Assumption #12 — user said "optionally document"; auto-install would be out-of-scope tooling/user-data change | S:75 R:85 A:80 D:70 |
| 12 | Confident | Hydrate-time wiring points = templates/memory.md placeholder + fab-continue.md Hydrate Step 4 + docs-hydrate-memory.md ingest/generate/backfill authoring steps (+ SPEC mirrors) | Intake Assumption #13 — the exact places description authoring is instructed today; the §3.2 citation chain makes consuming-repo hydrate agents read the policy every run | S:70 R:80 A:85 D:75 |

12 assumptions (6 certain, 6 confident, 0 tentative).
