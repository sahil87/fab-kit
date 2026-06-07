# Plan: Memory Rebalancer Apply Path (docs-reorg-memory)

**Change**: 260607-sx7a-reorg-memory-shape-rebalance
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

### Skill: docs-reorg-memory Apply Path

#### R1: Activate the file-moving apply path
The `docs-reorg-memory` skill SHALL actually perform approved `split` / `merge` / `flatten` / `move` migrations — moving files to their new paths, rewriting the relative links a move breaks, adding `description:` frontmatter to any new file or sub-domain index a split creates, regenerating indexes via `fab memory-index`, and enforcing a no-dangling-link guard. The "deferred follow-up" scope note SHALL be removed.

- **GIVEN** a Shape Report proposing a `split` of an over-wide domain into sub-domains
- **WHEN** the user approves the migration
- **THEN** the skill moves the clustered files into `docs/memory/{domain}/{sub-domain}/`, rewrites every relative link the move breaks, adds `description:` frontmatter to the new files, runs `fab memory-index`, and verifies no dangling relative link remains before finalizing

#### R2: Link Impact list is mandatory before approval
For any move-bearing migration (`split` / `merge` / `flatten` / `move`), the proposal SHALL produce a **Link Impact** list enumerating every relative link that will break, each paired with its rewrite, so the blast radius is visible before approval.

- **GIVEN** a proposed migration that moves `runtime-agents.md` into a `runtime/` sub-domain
- **WHEN** the proposal is presented
- **THEN** it lists every link such as `](runtime-agents.md)` in `execution-skills.md` → `](runtime/runtime-agents.md)`, covering both links *from* moved files and links *to* moved files from elsewhere in the domain

#### R3: No-dangling-link guard (hard)
Apply SHALL NOT finalize a migration while any relative link broken by the move remains unrewritten. A residual dangling link blocks finalizing that migration and is reported.

- **GIVEN** an approved migration whose moves have been executed
- **WHEN** a relative link broken by the move was not rewritten
- **THEN** the skill reports the dangling link and does not finalize that migration until it is fixed

### Templates: External Sub-Domain Addressing

#### R4: Affected Memory gains an optional sub-domain level
The intake template's `Affected Memory` format SHALL accept an optional sub-domain level — `{domain}/{sub-domain}/{file-name}` — while the flat `{domain}/{file-name}` form SHALL remain valid for un-split domains.

- **GIVEN** a memory file that lives in a sub-domain (`fab-workflow/runtime/runtime-agents`)
- **WHEN** it is listed under Affected Memory
- **THEN** the `{domain}/{sub-domain}/{file-name}` form is accepted and documented as valid alongside the flat form

#### R5: Selective-load convention becomes an up-to-3-hop walk
The `_preamble` selective-load convention SHALL describe an up-to-3-hop walk: domain index → (if the file is in a sub-domain) sub-domain index → file. The always-load layer description SHALL acknowledge that domains may contain sub-domains. The `context-loading` memory doc SHALL be updated to match.

- **GIVEN** an Affected Memory entry referencing a file in a sub-domain
- **WHEN** a skill selectively loads memory
- **THEN** it reads `docs/memory/{domain}/index.md`, then `docs/memory/{domain}/{sub-domain}/index.md`, then the file

### CLI: fab memory-index Sub-Domain Recursion

#### R6: Generate a sub-domain index per sub-domain
`fab memory-index` SHALL recurse into sub-domains: a directory under a domain dir that contains at least one non-index `.md` file SHALL get its own generated `docs/memory/{domain}/{sub-domain}/index.md`, rendered with the same file-row contract used for domain indexes.

- **GIVEN** `docs/memory/fab-workflow/runtime/` holding two topic files
- **WHEN** `fab memory-index` runs
- **THEN** it writes `docs/memory/fab-workflow/runtime/index.md` with a row per topic file, and the output is byte-stable on re-run

#### R7: Parent domain index references its sub-domains
A domain index whose folder contains sub-domains SHALL reference each sub-domain (linking to `{sub-domain}/index.md`). A domain with no sub-domains SHALL render byte-identically to the pre-change output (no spurious section).

- **GIVEN** `docs/memory/fab-workflow/` containing topic files plus a `runtime/` sub-domain
- **WHEN** `fab memory-index` runs
- **THEN** `docs/memory/fab-workflow/index.md` lists the sub-domain linking to `runtime/index.md`, while a sub-domain-free domain index is unchanged byte-for-byte

#### R8: Recursion stays deterministic, idempotent, and depth-bounded
Sub-domain enumeration SHALL be lexicographically ordered and byte-stable across runs. Depth and width warnings SHALL continue to fire on the recursive tree (depth-3 sub-domain topics counted; over-depth still warns). The recursion SHALL NOT recurse past the sub-domain tier into the index output (only domain and sub-domain index tiers are generated).

- **GIVEN** a nested tree with multiple sub-domains
- **WHEN** `fab memory-index` runs twice
- **THEN** the second run produces no diff and `--check` exits zero

### Non-Goals

- Performing an actual split of `fab-workflow` (or any existing domain). This change ships the capability only; the real split is a later deliberate `/docs-reorg-memory` run (intake Assumption #5).
- A `fab memory-relink` Go subcommand. Link rewriting is skill-driven per the Link Impact list (intake Assumption #6).
- Reorganizing `docs/memory/` content. Only skill/spec/template/Go capability changes plus this change's own hydration.

### Design Decisions

1. **Sub-domains are an additional index tier, not a new renderer**: `Gather` recurses one level (domain → sub-domain) and reuses `RenderDomain` for sub-domain indexes. — *Why*: the file-row contract is identical at any tier; relative `[file](file.md)` links are correct from a sub-domain index too. — *Rejected*: a bespoke `RenderSubDomain` — needless duplication.
2. **Parent domain index gains a `## Sub-Domains` section only when sub-domains exist**: byte-identical output for sub-domain-free domains. — *Why*: preserves every existing fixture/test and keeps the common case unchanged. — *Rejected*: always emitting the section (would churn every existing domain index).
3. **One level of recursion only** (`{domain}/{sub-domain}/`): matches the depth-3 bound (`{domain}/{sub-domain}/{topic}.md`). Deeper nesting is a depth warning, not a generated tier.

## Tasks

### Phase 1: Go — memory-index sub-domain recursion

- [x] T001 Add a `SubDomains []DomainData` field to `DomainData` in `src/go/fab/internal/memoryindex/memoryindex.go` <!-- R6 -->
- [x] T002 Extract sub-domain discovery in `Gather`: for each domain dir, enumerate child directories that hold ≥1 non-index `.md`, build a `DomainData` per sub-domain via the existing file-gather, attach lexicographically sorted to the parent's `SubDomains`; keep width/depth warnings firing across the recursive tree in `src/go/fab/internal/memoryindex/memoryindex.go` <!-- R6 R8 -->
- [x] T003 Extend `RenderDomain` in `src/go/fab/internal/memoryindex/memoryindex.go` to append a `## Sub-Domains` table (`| Sub-Domain | Description |` linking to `{sub}/index.md`) only when `len(SubDomains) > 0`; sub-domain-free output unchanged <!-- R7 -->
- [x] T004 Flatten domains + their sub-domains into `indexTarget`s in `src/go/fab/cmd/fab/memory_index.go` so every sub-domain `index.md` is written/checked <!-- R6 R7 -->

### Phase 2: Go — tests

- [x] T005 [P] Add byte-for-byte fixture tests for the nested-tree render (`RenderDomain` with sub-domains; sub-domain index via `RenderDomain`) and assert sub-domain-free `RenderDomain` is byte-identical to the existing expectation in `src/go/fab/internal/memoryindex/memoryindex_test.go` <!-- R7 R8 -->
- [x] T006 [P] Add `Gather` recursion tests: depth-3 sub-domain discovery, deterministic ordering, depth warning on depth-4, idempotency, in `src/go/fab/internal/memoryindex/memoryindex_test.go` <!-- R6 R8 -->
- [x] T007 [P] Add a cmd-level nested-tree test in `src/go/fab/cmd/fab/memory_index_test.go`: a `{domain}/{sub}/topic.md` tree regenerates a sub-domain index, the parent references it, and a second `--check` run is clean <!-- R6 R7 R8 -->

### Phase 3: Skill + template + preamble contract changes

- [x] T008 Rewrite `src/kit/skills/docs-reorg-memory.md` from the REFERENCE stab: remove the deferred-scope note, activate the Step-5 apply path (move → rewrite links → add frontmatter → `fab memory-index` → no-dangling-link guard), require the Link Impact list in the proposal <!-- R1 R2 R3 -->
- [x] T009 Update `src/kit/templates/intake.md` Affected Memory line to document the optional `{domain}/{sub-domain}/{file-name}` form alongside the flat form <!-- R4 -->
- [x] T010 Update `src/kit/skills/_preamble.md`: §3 Memory File Lookup becomes the up-to-3-hop walk; §1 Always Load description acknowledges sub-domains <!-- R5 -->

### Phase 4: Specs + docs

- [x] T011 [P] Update `docs/specs/skills/SPEC-docs-reorg-memory.md`: remove deferred-scope note; document apply path, link rewriting, Link Impact, no-dangling-link guard, External addressing <!-- R1 R2 R3 R4 -->
- [x] T012 [P] Update `docs/specs/templates.md`: Affected Memory sub-domain slot; memory directory structure shows a sub-domain tier; sub-domain index in the Generated Indexes section <!-- R4 R6 R7 -->
- [x] T013 [P] Update `docs/specs/architecture.md` memory tree illustration to show a `{domain}/{sub-domain}/` tier <!-- R6 -->
- [x] T014 [P] Update `src/kit/skills/_cli-fab.md` `## fab memory-index` to document sub-domain index generation + parent reference <!-- R6 R7 -->

### Phase 5: Verification

- [x] T015 `gofmt` + `go vet` clean; `go build ./...`; run `go test ./internal/memoryindex/... ./cmd/fab/...` then full `go test ./...`; verify `fab memory-index` idempotent on the real tree (run twice → no diff) <!-- R8 -->

## Execution Order

- T001 blocks T002, T003, T004 (the struct field is the foundation).
- T002–T004 block the tests T005–T007.
- Phase 3/4 (skill/spec/template) are independent of the Go phases and may proceed in parallel.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `docs-reorg-memory.md` performs approved moves, rewrites links, adds frontmatter, runs `fab memory-index`, and the deferred-scope note is gone — verified: Step 5 apply path (move → rewrite → frontmatter → memory-index → guard) present; deferred note removed
- [x] A-002 R2: the skill requires a Link Impact list for every move-bearing migration before approval — verified: "the proposal MUST also list … every relative link that would break" with both-directions requirement
- [x] A-003 R3: the skill enforces a hard no-dangling-link guard that blocks finalizing a migration — verified: Step 5.5 + Edge-Cases row both state "Hard block — do not finalize that migration until it is rewritten"
- [x] A-004 R4: intake template documents the optional `{domain}/{sub-domain}/{file-name}` Affected Memory form — verified in `src/kit/templates/intake.md` (comment + 3-part example row)
- [x] A-005 R5: `_preamble` §3 describes the up-to-3-hop walk and §1 acknowledges sub-domains; `context-loading` memory doc matches — `_preamble` §3 (3-hop walk) + §1 (sub-domain note) verified. **Memory-doc half deferred to hydrate**: `context-loading.md` lives in `docs/memory/` (hydrate-owned, post-implementation); updating it at apply would violate the capability-only scope boundary. It is updated by the hydrate stage that runs after this review (status: hydrate pending). Contract surfaces are done.
- [x] A-006 R6: `fab memory-index` generates a `{domain}/{sub-domain}/index.md` per sub-domain — verified end-to-end with the built binary on a nested temp fixture (sub-domain index generated with file rows)
- [x] A-007 R7: the parent domain index references each sub-domain; sub-domain-free domain indexes are byte-identical to pre-change output — verified: parent emits `## Sub-Domains` linking `runtime/index.md`; flat `flatdom` emits no section; structural diff of old-vs-new binary on the real tree is byte-identical (date column is a git-env artifact, `gitLastUpdated` untouched)
- [x] A-008 R8: recursion is deterministic, idempotent (`--check` clean on re-run), and depth-bounded — verified: run 2 = "already up to date", `--check` exit 0, depth-4 warns, no index past sub-domain tier

### Behavioral Correctness

- [x] A-009 R6: `gatherFiles`/`Gather` now enumerate depth-3 sub-domain topics (the PR #377 Copilot finding), not only depth-2 — verified: `gatherSubDomains` recurses one level and reuses `gatherFiles` for the sub-domain tier

### Scenario Coverage

- [x] A-010 R7: nested-tree byte-for-byte fixture test passes — `TestRenderDomain_WithSubDomains` + `TestRenderDomain_NoSubDomainsByteIdentical` pass
- [x] A-011 R8: idempotency test (two runs → no diff) and depth-warning test pass — `TestGather_SubDomainRenderIdempotent`, `TestGather_DepthWarningStillFiresUnderRecursion` pass
- [x] A-012 R6: cmd-level nested-tree test writes a sub-domain index and `--check` is clean afterward — `TestMemoryIndexCmd_GeneratesSubDomainIndex` passes

### Edge Cases & Error Handling

- [x] A-013 R6: an empty sub-domain dir (no `.md`) does NOT produce a spurious index; reserved domains stay width-exempt — `TestGather_EmptySubDirIsNotASubDomain` + reserved exemption preserved (only domain-tier reserved check; sub-domain width still warns per `TestGather_OverWideSubDomainWarns`)
- [x] A-014 R8: a depth-4 file still triggers the depth warning under recursion — `TestGather_DepthWarningStillFiresUnderRecursion` passes; verified live (`deeper exceeds depth 3`)

### Code Quality

- [x] A-015 Pattern consistency: Go changes mirror the existing `internal/memoryindex`/`internal/prmeta` pure-render + Gather-I/O split; skill/spec prose matches surrounding style — verified: `gatherSubDomains` is I/O in Gather; `RenderDomain` stays pure; prose matches surrounding tone
- [x] A-016 No unnecessary duplication: sub-domain rendering reuses `RenderDomain` rather than a new renderer — verified: no `RenderSubDomain`; sub-domain index uses `RenderDomain(sd)`

### Documentation Accuracy

- [x] A-017 R4 R6 R7: `SPEC-docs-reorg-memory.md`, `templates.md`, `architecture.md`, `_cli-fab.md` all reflect the apply path + sub-domain recursion accurately — all four diffs reviewed and accurate
- [x] A-018 R1: skill change is mirrored in its SPEC file (constitution rule) — `SPEC-docs-reorg-memory.md` gains Apply Path / Link Impact / External Addressing sections mirroring the skill

### Cross-References

- [x] A-019 R5: the up-to-3-hop walk is consistent across `_preamble.md`, the `context-loading` memory doc, and the intake template's Affected Memory format — `_preamble.md` and intake template are consistent (flat + 3-part forms, same walk). **`context-loading` memory doc deferred to hydrate** (same rationale as A-005); cross-ref will close at hydrate.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- This change ships capability only — it does NOT split `fab-workflow` (intake Assumption #5).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Sub-domains are rendered by reusing `RenderDomain` (file-row contract is tier-agnostic), not a new renderer. | Constitution code-quality (no duplication); the relative `[file](file.md)` link is correct from any tier. Locked by intake design. | S:90 R:75 A:90 D:88 |
| 2 | Certain | Parent domain index gains a `## Sub-Domains` section ONLY when sub-domains exist, keeping sub-domain-free output byte-identical. | Preserves every existing test/fixture; byte-stability is a hard invariant of `fab memory-index`. | S:85 R:70 A:90 D:85 |
| 3 | Confident | A "sub-domain" is a dir directly under a domain dir containing ≥1 non-index `.md`; recursion is one level only (matches depth-3 bound). | The Copilot finding is specifically about depth-3 topics; deeper nesting is a depth warning, not a generated tier. Intake Assumption #3. | S:80 R:65 A:85 D:80 |
| 4 | Confident | Link rewriting + the no-dangling-link guard are skill-driven Edits, not a Go helper. | Intake Assumptions #4/#6; Constitution I (Pure Prompt Play). | S:85 R:75 A:88 D:85 |
| 5 | Certain | `_generation.md` needs no change — it does not reference the `{domain}/{file-name}` format (grep-confirmed). | Verified by grep; the format lives in the intake template + `_preamble` + `context-loading`. | S:95 R:80 A:95 D:92 |
| 6 | Confident | Sub-domain index rows use the same `| File | Description | Last Updated |` table; the parent's sub-domain reference uses `[sub](sub/index.md)` mirroring the root index's domain links. | Consistent with the existing root/domain link convention; lowest-surprise rendering. | S:80 R:70 A:85 D:82 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).

## Deletion Candidates

None — this change adds new functionality (sub-domain recursion + the activated apply path) without making existing code redundant. `gatherSubDomains` reuses `gatherFiles`/`domainTitle`/`domainDescription` rather than introducing a parallel implementation, and sub-domain rendering reuses `RenderDomain` (no `RenderSubDomain` to retire). The deferred-scope prose in the skill/spec was rewritten in place, not left dangling. (Note for housekeeping, not source deletion: `NOTES.md` and `REFERENCE-docs-reorg-memory-stab.md` in this change folder were `tciy`-session scaffolding now fully realized by this change — archivable when the change ships, but they are change artifacts, not repo source.)
