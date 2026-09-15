# Plan: Root-Agnostic Docs-Index Generator

**Change**: 260908-flt3-root-agnostic-docs-index
**Intake**: `intake.md`

## Requirements

### Docs index: Configuration

#### R1: Configuration
Project-scoped docs_index.roots SHALL accept path, index_file (index.md), also_accept ([]), log (false), max_depth (3), and superseded ([]); absence SHALL select docs/memory with log true. Config explain SHALL expose defaults, description, scope, and advertisement.

- **GIVEN** no docs_index config
- **WHEN** roots resolve
- **THEN** the implicit memory root is selected; explicit root fields use their documented defaults

### Docs index: Recursive navigation

#### R2: Recursive navigation
The generator SHALL recurse to arbitrary depth and link each content-bearing child landing. max_depth SHALL warn advisorially, never truncate traversal. Zero-config memory output SHALL retain existing bytes except generated-by command headers.

- **GIVEN** a depth-7 tree and max_depth 3
- **WHEN** generation runs
- **THEN** all live folders are indexed and depth warnings do not fail generation

### Docs index: Landing ownership

#### R3: Landing ownership
index_file landings SHALL use whole-file generation; existing also_accept landings SHALL receive a marker-delimited generated table with outside prose byte-preserved, without a neighboring index_file.

- **GIVEN** README.md is accepted and contains human prose
- **WHEN** generation runs twice
- **THEN** the table appears once, prose is unchanged, and index.md is absent

### Docs index: Sparse descriptions

#### R4: Sparse descriptions
Missing description in the generic root format SHALL produce an H1-labelled row with — and an advisory warning, never invented text or a generation failure.

- **GIVEN** a live file has H1 but no description
- **WHEN** generation runs
- **THEN** its row uses the H1 and placeholder and reports an advisory

### Docs index: Superseded status

#### R5: Superseded status
Superseded glob patterns SHALL match folders and files, including recursive archive, escaped bracket filename, and Z-prefix conventions. A superseded subtree SHALL have one parent summary row and one own index of immediate child folders with counts, no descendant per-file rows or topic-description reads. Version summaries SHALL use numeric version bounds; individual matches SHALL fold into the parent folder superseded-file note.

- **GIVEN** archive contains v1 through v4 and live v5 is a sibling
- **WHEN** generation runs
- **THEN** the live rows remain navigable and archive is one pointer/count plus its per-version index

### Docs index: Seed adoption

#### R6: Seed adoption
First-run curated landings SHALL seed descriptions and preserve curated navigation, grouping, and tombstones without force/adopt flags or tier-2 loss. Later regeneration SHALL retain destructive-loss detection.

- **GIVEN** a hand-curated index has descriptions, custom headings, and historical rows
- **WHEN** check then first generation run
- **THEN** check is benign and all curated navigation survives idempotent adoption

### Docs index: Safety contract

#### R7: Safety contract
All roots SHALL retain deterministic output, --check 0/1/2, worst-root aggregation, blocking floor at 1, advisory non-blocking, and JSON tier/drift/losses/malformed/warnings. Refuse-before-regen guards SHALL continue to key on exit 2.

- **GIVEN** one root is benign and another loses curated data
- **WHEN** all-root check runs
- **THEN** the aggregate exit is 2 and JSON retains every required key

### Docs index: FKF scope

#### R8: FKF scope
Only log:true roots SHALL use fkf_version, freeze-on-write logs, seeds, C-lite joins, reserved-domain exemptions and FKF description escalations; generic malformed and shape/size/length diagnostics SHALL generalize.

- **GIVEN** specs has log false and memory has log true
- **WHEN** generation and checks run
- **THEN** specs has no generated log/FKF metadata or FKF blocking escalations and memory keeps its guarantees

### Docs index: CLI rename

#### R9: CLI rename
fab docs-index [root-path] SHALL process all configured roots or one configured positional root, reject unconfigured paths naming docs_index.roots, and retain --check/--json/--rebuild. memory-index SHALL remain a memory-only deprecated alias with one stderr notice and unchanged JSON stdout, for at least one minor version.

- **GIVEN** both memory and specs are configured
- **WHEN** the alias is invoked
- **THEN** only memory is processed and stderr carries one deprecation line

### Docs index: Caller migration

#### R10: Caller migration
The eight intake-listed kit skills, template, FKF reference, scaffold and embedded CLI skill SHALL use docs-index. Memory operations SHALL select docs/memory; aggregate ship operations MAY process all roots. Alias references in kit skills SHALL occur only in older-binary fallback prose; historical migrations SHALL remain unchanged.

- **GIVEN** kit callers and sibling aggregate specs are swept
- **WHEN** migration completes
- **THEN** guards, strings, comments and references use the new command with correct scope

### Docs index: Specs adoption and governance

#### R11: Specs adoption and governance
Constitution VI SHALL carve out generated index/navigation, retain human spec ownership, bump 1.6.0 to 1.7.0 with a dated governance note; this repo SHALL configure memory and specs and regenerate specs through seed-import. The no-generator stance SHALL be rewritten in memory and docs-reorg-specs; config specs SHALL document the new key.

- **GIVEN** this repository uses curated specs
- **WHEN** the change is applied
- **THEN** its specs navigation is generated without losing curated descriptions and the governing documents agree

### Docs index: Migration note

#### R12: Migration note
A next-minor migration note SHALL explain nothing-to-do compatibility, alias duration, opt-in roots and one-time header churn; historical migration files SHALL stay frozen.

- **GIVEN** an existing memory-only project upgrades
- **WHEN** it follows the migration note
- **THEN** no config or data migration is necessary

### Docs index: Bounded warning reporting

#### R13: Bounded warning reporting
Warning output SHALL be bounded: stderr advisory reporting SHALL cap per-class output with a visible truncation count (e.g. "… and N more"), and the `--json` report SHALL either bound the `warnings` array the same way or carry an explicit total alongside the truncated list. Required JSON keys and exit semantics SHALL be preserved. <!-- added in rework cycle 2: review found 69 stderr lines / 8.5 KB JSON on this repo alone — unbounded surface violates the toolkit Principle 9 MUST incorporated by the constitution's Toolkit Standards article -->

- **GIVEN** a sparse-description root producing dozens of advisories
- **WHEN** `fab docs-index --check --json` runs
- **THEN** stderr shows a bounded list plus a truncation count and JSON keeps `tier`/`drift`/`losses`/`malformed`/`warnings` with the total discoverable

## Tasks

### Phase 1: Setup

- [x] T001 Add root config types/default resolution and registered explain metadata in src/go/fab/internal/config/docs_index.go and src/go/fab/internal/configref/configref.go, with tests. <!-- R1 --> <!-- rework: overlap validation misses `.` as an ancestor root — `path + "/"` prefix compare never matches the repo-root path `.`, so `.` and `docs/specs` are both accepted (R1 duplicate/overlap rejection violated) -->
### Phase 2: Core Implementation

- [x] T002 Implement recursive root-aware gathering/rendering in src/go/fab/internal/memoryindex/docsindex.go, preserving legacy render goldens. <!-- R2 -->
- [x] T003 Implement accepted landing block ownership in src/go/fab/internal/memoryindex/docsindex.go with README preservation tests. <!-- R3 -->
- [x] T004 Add H1 placeholder rows and missing-description advisories in src/go/fab/internal/memoryindex/docsindex.go with sparse-root tests. <!-- R4 -->
- [x] T005 Implement superseded pattern/count/version rendering in src/go/fab/internal/memoryindex/superseded.go with tree tests. <!-- R5 --> <!-- rework: live→superseded flip leaves stale generated descendant index.md/log.md on disk (docsindex.go:74 stops emission but cmd/fab/memory_index.go:141 never removes obsolete generated targets) — R5's no-descendant-output contract violated -->
- [x] T006 Implement first-run navigation seed adoption in src/go/fab/internal/memoryindex/adoption.go and test descriptions, grouping, tombstones and idempotency. <!-- R6 --> <!-- rework: table parsing uses strings.Split(line, "|") which breaks on escaped pipes (\|) — corrupts preserved curated rows (adoption.go:72, indexparse.go:68) -->
- [x] T007 Scope warnings/log generation to per-root options in src/go/fab/internal/memoryindex/memoryindex.go and docsindex.go; test generic versus FKF roots. <!-- R8 --> <!-- rework cycle 3: memoryindex.go:1215-1221 sourceWarnings filters only narration-density for log:false roots — the FKF broken-link diagnostic (KindBrokenLink) still runs on generic roots and misinterprets /... links; gate it on fkf and add a log:false regression test -->
### Phase 3: Integration & Edge Cases

- [x] T008 Generalize root-relative classification and aggregate checks in src/go/fab/internal/memoryindex/loss.go and src/go/fab/cmd/fab/memory_index.go with graded-exit tests. <!-- R7 --> <!-- rework: the escaped-pipe parser defect feeds loss classification too — escaped-pipe link labels can evade R7's tombstone/description safeguards; fix the shared parser and add a regression test -->
- [x] T009 Wire docs-index and deprecated memory-only alias in src/go/fab/cmd/fab/main.go and memory_index.go; add command/config/alias integration tests. <!-- R9 -->
### Phase 4: Polish

- [x] T010 Migrate all eight callers under src/kit/skills/, src/kit/templates/memory.md, src/kit/reference/fkf.md, src/kit/scaffold/docs/memory/index.md, src/go/fab/cmd/fab/skill.md and sibling specs; preserve fallback semantics. <!-- R10 --> <!-- rework cycle 2: git-pr-review.md:199 runs bare `fab docs-index` for a memory-only conflict-recovery operation — must be `fab docs-index docs/memory` per _cli-fab.md's memory-only scope contract (cycle-1 owner-or-pointer fix confirmed done) -->
- [x] T011 Amend fab/project/constitution.md and config.yaml, docs/specs/config.md and docs/memory/memory-docs/specs-index.md; perform seed-import regen of docs/specs/index.md. <!-- R11 -->
- [x] T012 Write src/kit/migrations/2.23.17-to-2.24.0.md and finish scoped Go tests, broader checks if needed, and final sibling sweep. <!-- R12 --> <!-- rework cycle 3: docs/memory/memory-docs/templates.md:191 still claims specs-index generation was declined / specs index hand-written (contradicts R11); docs/specs/templates.md:389-396,478-512,659-660 instructs bare `fab docs-index` in memory-specific workflows — scope those to `fab docs-index docs/memory` (generated-header EXAMPLES may keep the bare name) --> <!-- rework cycle 2: sweep still incomplete — live owner comments in the five log.seed.md files (_shared, distribution, memory-docs, pipeline, runtime — line 2 of each) still name fab memory-index (curated seed inputs, NOT generated log history), and docs/memory/distribution/setup.md:188 claims scaffold files memory-index.md/specs-index.md where the live files are src/kit/scaffold/docs/memory/index.md and src/kit/scaffold/docs/specs/index.md -->
- [x] T013 Add bounded warning reporting: per-class stderr cap with visible truncation count and a bounded-or-totaled JSON `warnings` surface in src/go/fab/cmd/fab/memory_index.go and src/go/fab/internal/memoryindex/docsindex.go, preserving JSON keys and exit semantics, with tests. <!-- R13 -->
- [x] T014 Reconcile the branch with origin/main (one commit behind): restore the landed t513 unconditional-agent-deploy state — src/go/fab-kit/internal/skills.go must not re-gate .claude/skills and .agents/skills, and fab/project/constitution.md must retain BOTH the t513 wording/governance note (.agents/skills) AND this change's VI carve-out + 1.7.0 bump. Rebase or merge, then re-run both Go module test suites. <!-- R11 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: Configuration meets its requirement and documented scenario.
- [x] A-002 R2: Recursive navigation meets its requirement and documented scenario.
- [x] A-003 R3: Landing ownership meets its requirement and documented scenario.
- [x] A-004 R4: Sparse descriptions meets its requirement and documented scenario.
- [x] A-005 R5: Superseded status meets its requirement and documented scenario.
- [x] A-006 R6: Seed adoption meets its requirement and documented scenario.
- [x] A-007 R7: Safety contract meets its requirement and documented scenario.
- [x] A-008 R8: FKF scope meets its requirement and documented scenario.
- [x] A-009 R9: CLI rename meets its requirement and documented scenario.
- [x] A-010 R10: Caller migration meets its requirement and documented scenario.
- [x] A-011 R11: Specs adoption and governance meets its requirement and documented scenario.
- [x] A-012 R12: Migration note meets its requirement and documented scenario.

### Behavioral Correctness

- [x] A-013 R2: Depth-7 and zero-config golden tests verify recursion and memory compatibility.
- [x] A-014 R3: README generated-block round trips preserve prefix/suffix bytes.
- [x] A-015 R8: FKF logs remain frozen and generic roots do not produce them.

### Removal Verification

- [x] A-016 R10: No active kit caller depends on memory-index outside version-skew fallback prose.
- [x] A-017 R11: No current specs ownership claim forbids generated navigation.

### Scenario Coverage

- [x] A-018 R5: Superseded versions, non-version folders, escaped bracket files, Z-prefix folders, and double generation are tested.
- [x] A-019 R6: First-run curated descriptions, groupings and tombstones survive check/write/check.

### Edge Cases & Error Handling

- [x] A-020 R7: Tier 2 outranks blocking findings; advisories leave a byte-clean root at exit 0.
- [x] A-021 R9: Unconfigured/invalid root and alias JSON behavior are tested.

### Code Quality

- [x] A-022: New code follows surrounding naming, error handling and structure.
- [x] A-023: Existing utilities are reused without unnecessary duplication.
- [x] A-024: Implementation favors readability and maintainability.
- [x] **N/A**: A-025: Composition is preferred over inheritance. — Go implementation introduces no inheritance hierarchy.
- [x] A-026: Functions remain focused with large operations decomposed.
- [x] A-027: Thresholds and markers use named constants.
- [x] A-028: Only canonical kit sources are edited.
- [x] A-029: CLI docs and tests accompany the surface change.
- [x] A-030: Owned skill rules are stated once or linked, never both.
- [x] A-031: Sibling skill, aggregate spec and user-facing string sweeps are complete.
- [x] A-032: Relevant Go tests conform to the intake and pass.
- [x] A-033 R13: Warning output is bounded on stderr with a visible truncation count, and the JSON warnings surface is bounded or totaled, with required keys and exit semantics preserved.
- [x] A-034 R11: The branch incorporates origin/main — the t513 unconditional-agent-deploy behavior and governance note are retained alongside this change's Constitution VI carve-out and 1.7.0 bump.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Keep internal/memoryindex; expose a root-aware API there | Existing render, classifier and FKF utilities are reusable; package rename adds no behavior | S:80 R:95 A:90 D:90 |
| 2 | Confident | Preserve imported navigation in explicit curated markers in generated landing files; reuse its rows as description seeds | Keeps adoption self-contained and human-readable, with no topic-body rewrites or hidden sidecars | S:75 R:80 A:75 D:70 |
| 3 | Confident | Use slash-separated glob semantics with ** and escaped literal brackets; match directory paths with trailing /** as subtree roots | The intake explicitly asks for recursive, Z-prefix and bracket-name coverage | S:85 R:80 A:80 D:75 |

| 4 | Confident | Keep filename-stem labels for missing descriptions in the legacy memory format; generic roots use H1 | Source inspection shows the old renderer used the stem even with an H1. The intake explicitly preserves zero-config bytes; its sparse-H1 use case is the new generic roots | S:80 R:90 A:80 D:65 |

4 assumptions (1 certain, 3 confident, 0 tentative).

## Apply Rework Cycle 1

All five must-fix findings are implemented. The regression tests cover normalized
`.` overlap in either order, escaped-pipe description replacement and loss
classification, and live-to-superseded retirement (read-only check, legacy headers,
whole-file removal, alternate-landing prose preservation, seed preservation, and
idempotency). Current command references were swept repo-wide; frozen migration
instructions, dated review evidence, historical identifiers and generated log
history retain their historical spelling. The specs reorganization skill delegates
the guard procedure to `_cli-fab`.

The optional configured-landing log exclusion is fixed and tested, and the unused
command-local `indexTarget` type is removed. Legacy package wrappers and their
log-title fallback remain to support the existing compatibility tests.

Validation: `go test ./...` in `src/go/fab` passes. The built CLI's all-root check
passes at tier 0 after regeneration, with no losses or blocking findings. Frozen
log histories are unchanged apart from generated command headers; mirrored skill
and FKF documents remain byte-identical. Acceptance checkmarks above remain the
review worker's responsibility.

## Deletion Candidates

- `src/go/fab/internal/memoryindex.Gather`, `GatherLogs`, and `LogTarget` (`memoryindex.go:354-370`, `756-792`) — production callers now use `GatherRoot`; these compatibility surfaces are referenced only by legacy package tests.
- `src/go/fab/internal/memoryindex.loadGitDates` (`memoryindex.go:448-454`) — root-aware production code calls `loadGitDatesForRoot`; only legacy tests retain the memory-specific wrapper.
- `src/go/fab/internal/memoryindex.domainTitle` and the empty-`titleOverride` branch in `buildLogTarget` (`memoryindex.go:372-379`, `850-853`) — `GatherRoot` always supplies the gathered folder title, leaving this fallback reachable only through legacy test helpers.

## Apply Rework Cycle 2

Reconciled with `origin/main` at `b0ae654a` before continuing implementation.
The t513 unconditional deployment implementation is unchanged from main; both
constitution governance notes, `.agents/skills` wording, the VI navigation
carve-out, and version 1.7.0 survive the reconciliation (A-034 evidence).

Memory conflict recovery now selects `docs/memory`. The five curated seed owner
comments and the obsolete scaffold filenames are corrected; historical seed
entries remain byte-identical (A-010/A-031 evidence). The repo-wide rename sweep
retains alias compatibility, historical migration/review evidence, change IDs,
and frozen history only.

Advisory reporting now caps details at five per kind across the whole invocation
on stderr and in JSON, with visible omitted counts and additive `warnings_total`.
Shape warnings remain stderr-only; blocking findings and losses remain complete.
Survey consumers handle sampled output without treating omitted domains as clean.
Regression tests cover cross-root caps, per-class truncation, required JSON keys,
blocking floors, and advisory/loss exit semantics (A-033 evidence).

Validation: `go test ./...` passes in both `src/go/fab` and `src/go/fab-kit`.
The built CLI checks all roots at tier 0 with no losses or blocking findings;
68 JSON-eligible advisories are represented by 16 samples and 20 stderr lines.
`git diff --check` passes; the embedded skill mirrors its source and remains
below the 150-line cap. Acceptance checkmarks remain for the review worker.

## Apply Rework Cycle 3

FKF broken-link diagnostics are gated alongside narration density: `log: false`
roots retain generic size/length warnings without interpreting site-root links
as FKF bundle paths. A paired `log: false` / `log: true` regression test confirms
both exclusion and retained FKF behavior. The CLI reference states the scope.

The memory templates documentation now records generated specs navigation and
human-owned content. Memory-specific template workflows consistently select
`fab docs-index docs/memory`; generated-header examples keep the generator name.

Validation: `go test ./internal/memoryindex ./cmd/fab` passes. The rebuilt CLI's
all-root check remains tier 0, with no losses or blocking findings, and
`git diff --check` passes. T007 and T012 are complete; acceptance checkmarks
remain for the review worker.
