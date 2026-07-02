# Plan: Gitignore-aware sync dedup

**Change**: 260625-mqiq-gitignore-aware-sync-dedup
**Intake**: `intake.md`

## Requirements

### Sync: `.gitignore` dedup coverage

#### R1: Gitignore-aware "already covered" dedup
`lineEnsureMerge` SHALL, when merging into a `.gitignore`, treat a directory-style
fragment entry (e.g. `/.claude`) as already present when any destination line is a
variant covering the same core directory token. The core token is derived by stripping
a leading `/` and a single trailing `/` or `/*`. The covering set for `/.claude` is
`{ /.claude, /.claude/, /.claude/*, .claude, .claude/, .claude/* }`.

- **GIVEN** a `.gitignore` already containing one of `/.claude/`, `/.claude/*`, `.claude`, `.claude/`, `.claude/*`
- **WHEN** `fab sync` merges the fragment entry `/.claude`
- **THEN** no additional `/.claude` line is appended (the entry is already covered)

#### R2: Normalization is anchored at the directory token only
A destination line that is a DEEPER nested path (e.g. `/.claude/commands/`) SHALL NOT
normalize to the core token `.claude` and SHALL NOT count as covering the entry.

- **GIVEN** a `.gitignore` whose only `.claude`-related line is `/.claude/commands/`
- **WHEN** `fab sync` merges the fragment entry `/.claude`
- **THEN** `/.claude` is appended (a deeper path does not cover the directory token)

#### R3: Genuine miss still appends (happy path preserved)
When no destination line covers the entry, `lineEnsureMerge` SHALL append the entry,
preserving the pre-existing behavior.

- **GIVEN** a `.gitignore` containing none of the `.claude` variants
- **WHEN** `fab sync` merges the fragment entry `/.claude`
- **THEN** `/.claude` is appended

#### R4: Negation is a hard stop (Guardrail B)
When the destination `.gitignore` already contains a negation line for the entry's core
token (matching `!/.claude/...` or `!.claude/...`), `lineEnsureMerge` SHALL NOT append a
broader ignore for that entry — regardless of whether a broader `/.claude/*` exclusion
precedes the negation, and regardless of the variant-coverage check.

- **GIVEN** a `.gitignore` containing `/.claude/*` then `!/.claude/commands/`
- **WHEN** `fab sync` merges the fragment entry `/.claude`
- **THEN** the file is left unchanged (no trailing `/.claude`); the negation survives
- **AND GIVEN** a `.gitignore` containing only `!/.claude/commands/` (no preceding exclusion)
- **WHEN** `fab sync` merges the fragment entry `/.claude`
- **THEN** the append is still suppressed (a present negation suppresses regardless)

#### R5: Semantic matching is scoped to `.gitignore` only (Guardrail A)
The gitignore-aware coverage and negation logic SHALL apply only when the destination
basename is `.gitignore`. For all other destinations (notably `.envrc`), `lineEnsureMerge`
SHALL fall back to the existing strict literal `==` equality check, unchanged.

- **GIVEN** an `.envrc` containing a line literally different from a fragment entry
- **WHEN** `lineEnsureMerge` merges that fragment entry
- **THEN** the fragment entry is appended (no semantic matching leaks to `.envrc`)

### Non-Goals

- Changing the shipped scaffold default — `src/kit/scaffold/fragment-.gitignore` keeps `/.claude`.
- Full gitignore-spec parsing — only the directory-token variant set and negation hard-stop.
- Touching `src/kit/skills/_cli-fab.md` or any SPEC mirror — this is not a command-signature change.
- Any migration — `.gitignore` content is the user's, not a managed config artifact.

### Design Decisions

1. **New unexported helper `gitignoreCovers(existingLine, entry string) bool`**: encapsulates
   the variant-coverage predicate (normalize-and-compare on the core token) — *Why*: keeps
   `lineEnsureMerge` readable and the predicate unit-testable in isolation — *Rejected*:
   inlining a regex tangle into the dedup loop (harder to read, magic patterns).
2. **Negation detection via a second helper `gitignoreHasNegation(destLines []string, entry string) bool`**:
   scans destination lines once for a `!`-prefixed line whose normalized core token equals the
   entry's core token — *Why*: the negation hard-stop is independent of (and binds tighter than)
   the per-line coverage check, so it reads cleanest as its own scan — *Rejected*: folding it into
   the per-line loop (would conflate two distinct predicates).
3. **Gate on `filepath.Base(label) == ".gitignore"`**: the `label`/`dest` argument is the
   destination path; its basename distinguishes `.gitignore` from `.envrc` and any other fragment —
   *Why*: matches the intake's Guardrail A guidance and survives nested destinations — *Rejected*:
   matching on the full label string (breaks for `subdir/.gitignore`).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Add unexported helper `gitignoreNormalize(s string) string` (or inline equivalent) plus `gitignoreCovers(existingLine, entry string) bool` and `gitignoreHasNegation(destLines []string, entry string) bool` in `src/go/fab-kit/internal/scaffold.go`, implementing directory-token normalization (strip leading `/`, strip single trailing `/` or `/*`) with deeper-path rejection (normalized form must equal the core token exactly). <!-- R1 --> <!-- R2 --> <!-- R4 -->
- [x] T002 Rewire the dedup loop in `lineEnsureMerge` (`src/go/fab-kit/internal/scaffold.go` ~line 312–318): when `filepath.Base(label) == ".gitignore"`, first short-circuit on `gitignoreHasNegation` (suppress append), else mark `found` when any destination line `gitignoreCovers` the entry; for non-`.gitignore` destinations keep the literal `strings.TrimRight(dl, "\r") == entry` check unchanged. Preserve existing CR handling. <!-- R3 --> <!-- R4 --> <!-- R5 -->

### Phase 3: Tests

- [x] T003 [P] Add table-driven `TestLineEnsureMerge_GitignoreVariantCoverage` (each of `/.claude/`, `/.claude/*`, `.claude`, `.claude/`, `.claude/*` → no `/.claude` appended) and `TestLineEnsureMerge_GitignoreGenuineMissAppends` + `TestLineEnsureMerge_GitignoreDeeperPathDoesNotCover` to `src/go/fab-kit/internal/scaffold_test.go`. <!-- R1 --> <!-- R2 --> <!-- R3 -->
- [x] T004 [P] Add `TestLineEnsureMerge_GitignoreNegationSurvives` (both `/.claude/*` + `!/.claude/commands/` and the lone-negation case → unchanged) and `TestLineEnsureMerge_EnvrcStrictEquality` (literally-different `.envrc` line still appends; semantic matching does not leak) to `src/go/fab-kit/internal/scaffold_test.go`. <!-- R4 --> <!-- R5 -->
- [x] T005 Extend `src/go/fab-kit/internal/sync_integration_test.go` with `TestSync_GitignoreNegationSurvivesFullSync` — a real end-to-end `fab sync` over a `.gitignore` containing `/.claude/*` + `!/.claude/commands/` leaves it unchanged (no trailing `/.claude`). <!-- R4 -->
- [x] T006 Run `go test ./internal/...` from `src/go/fab-kit`; all tests pass. <!-- R1 --> <!-- R2 --> <!-- R3 --> <!-- R4 --> <!-- R5 -->

## Execution Order

- T001 blocks T002 (the loop calls the helpers).
- T003/T004 (`[P]`, scaffold_test.go cases) and T005 (sync_integration_test.go) depend on T001+T002.
- T006 runs last.

## Acceptance

### Functional Completeness

- [ ] A-001 R1: A `.gitignore` containing any variant in `{ /.claude/, /.claude/*, .claude, .claude/, .claude/* }` does not gain an appended `/.claude` from sync.
- [ ] A-002 R3: A `.gitignore` with none of the variants still gets `/.claude` appended (happy path unchanged).
- [ ] A-003 R5: `.envrc` retains strict literal equality — a literally-different line still appends; no semantic matching leaks.

### Behavioral Correctness

- [ ] A-004 R2: A `.gitignore` whose only `.claude` line is the deeper path `/.claude/commands/` still gets `/.claude` appended (directory-token-only normalization).
- [ ] A-005 R4: A `.gitignore` with `/.claude/*` + `!/.claude/commands/` is left unchanged by sync; the lone-negation case is also suppressed.

### Scenario Coverage

- [ ] A-006 R4: An end-to-end `fab sync` over a negation-bearing `.gitignore` leaves it unchanged (integration test passes).
- [ ] A-007 R1: Variant coverage is exercised by a table-driven unit test (one case per variant).

### Edge Cases & Error Handling

- [ ] A-008 R1: CR handling (`strings.TrimRight(dl, "\r")`) is preserved in the new coverage path.

### Code Quality

- [ ] A-009 Pattern consistency: New helpers and tests follow surrounding naming/structure (`TestLineEnsureMerge_*`, unexported helpers, error-propagation contract unchanged).
- [ ] A-010 No unnecessary duplication: Normalization logic is centralized in one helper, reused by both coverage and negation predicates.
- [ ] A-011 Documentation accuracy: No CLI/skill/SPEC mirror edits made (correctly out of scope — not a command-signature change); scaffold default unchanged.
- [ ] A-012 Cross-references: Function doc comment on `lineEnsureMerge` reflects the gitignore-aware dedup so the behavior is discoverable.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Gate semantic matching on `filepath.Base(label) == ".gitignore"` (not full-label match) | The `label` arg is the destination path and may be nested (`scaffoldTreeWalk` passes `destPath`); basename is the robust discriminator the intake calls for | S:95 R:85 A:95 D:90 |
| 2 | Certain | Negation hard-stop binds tighter than variant coverage and is checked first | Intake Guardrail B: "the negation check is the binding guardrail"; suppress on ANY `!.../.claude/...` line | S:95 R:80 A:90 D:90 |
| 3 | Confident | Deeper-path rejection via exact normalized-token equality (normalized form must equal core token with no residual `/`) | Intake clarification: `/.claude/commands/` must not cover `/.claude`; exact equality of normalized core tokens achieves this cleanly | S:85 R:80 A:90 D:85 |

3 assumptions (2 certain, 1 confident, 0 tentative).
