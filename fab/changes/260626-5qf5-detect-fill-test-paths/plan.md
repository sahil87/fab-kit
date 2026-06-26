# Plan: Detect & Fill test_paths at Setup

**Change**: 260626-5qf5-detect-fill-test-paths
**Intake**: `intake.md`

## Requirements

### Scaffold: Standing test_paths Example Comment Block

#### R1: Persistent annotated example comment above the active key
The scaffold `src/kit/scaffold/fab/project/config.yaml` SHALL present `test_paths` examples as a **standing comment block above the active key** — one example per line in YAML-list form, annotated by ecosystem and convention — plus a `:(glob)` magic-pathspec note and an anti-substring warning. The block SHALL persist whether the key is filled or left empty. A `{TEST_PATHS}` placeholder token SHALL be introduced so create-mode setup can substitute a detected value while preserving the comment block.

- **GIVEN** a fresh project scaffolded from the kit template
- **WHEN** the user opens `fab/project/config.yaml`
- **THEN** the `test_paths` section shows the annotated examples (Go/Python/JS-TS/Java-Kotlin), the `:(glob)`/anti-substring note, and an active line carrying the `{TEST_PATHS}` placeholder

#### R2: Filling preserves the comment block
When create-mode substitutes a value for `{TEST_PATHS}`, the standing example comment block MUST remain intact and only the active key line is replaced.

- **GIVEN** create-mode has detected an ecosystem
- **WHEN** it writes the detected `test_paths` value
- **THEN** the example comment block above the key is preserved verbatim and only the active key line carries the detected patterns

### Skill: Non-Interactive Detection in fab-setup Create Mode

#### R3: Non-interactive ecosystem detection derives test_paths
`src/kit/skills/fab-setup.md` Config Create-Mode SHALL gain a detection sub-step (in step 2) that reads on-disk marker files and derives an anchored `test_paths` pattern from a language→pattern table, WITHOUT prompting the user. Multi-marker repos take the union of patterns.

- **GIVEN** a project being set up that has a `go.mod` on disk
- **WHEN** create-mode runs detection
- **THEN** `{TEST_PATHS}` is derived as `**/*_test.go` with no prompt to the user
- **AND** a project with no recognized marker leaves `test_paths` empty

#### R4: {TEST_PATHS} placeholder substitution
`src/kit/skills/fab-setup.md` step 4 (placeholder substitution) SHALL include `{TEST_PATHS}` alongside `{PROJECT_NAME}`, `{PROJECT_DESCRIPTION}`, `{SOURCE_PATHS}`. When a value is filled, substitution MUST preserve the standing comment block and replace only the active key line.

- **GIVEN** detection has derived a pattern set
- **WHEN** step 4 substitutes placeholders
- **THEN** `{TEST_PATHS}` is replaced by the detected patterns and the example comment block is preserved

#### R5: Visible detection note in Config Output
Detection SHALL surface a visible note in step 7 / Config Output: when filled, `Detected {ecosystem} — set test_paths to {patterns}. Edit via /fab-setup config source_paths if wrong.`; when no convention is recognized, leave the key commented and note `No test convention detected — test_paths left empty (impact breakdown will show a single total). Set it later if desired.`

- **GIVEN** create-mode detected Python (pytest)
- **WHEN** Config Output is shown
- **THEN** it includes the detected-ecosystem note with the patterns
- **AND** an unrecognized stack shows the "no convention detected" note

### Migration: Backfill Existing Repos (2.7.1 → 2.8.0)

#### R6: New migration file refreshes comment block and fills test_paths
A new migration `src/kit/migrations/2.7.1-to-2.8.0.md` SHALL (a) refresh the scaffold comment block in the user's existing `fab/project/config.yaml` to match R1, and (b) detect + fill `test_paths` using the same detection table — ONLY when the key is absent or empty. It MUST follow the established migration shape (Summary / Pre-check / Changes / Verification) per `2.2.0-to-2.3.0.md`.

- **GIVEN** an existing fab-kit project with `test_paths` absent or empty and a `pom.xml` on disk
- **WHEN** `/fab-setup migrations` applies `2.7.1-to-2.8.0`
- **THEN** the comment block is refreshed and `test_paths` is filled with `**/src/test/**`

#### R7: Migration idempotency and value preservation (Constitution III)
The migration MUST be idempotent and sentinel-guarded: skip entirely when `config.yaml` is absent; never overwrite a non-empty user `test_paths` (still refresh the comment); use a sentinel comment marker to make the comment refresh a re-run no-op; leave `test_paths` empty when no ecosystem is recognized. Report lines mirror the create-mode notes.

- **GIVEN** a project that already has a hand-set non-empty `test_paths`
- **WHEN** the migration runs
- **THEN** the user's value is preserved unchanged and only the comment block is refreshed
- **AND** re-running the migration is a complete no-op on the comment refresh (sentinel detected)

#### R8: VERSION bump to 2.8.0
`src/kit/VERSION` SHALL be bumped from `2.7.1` to `2.8.0` (next minor) to match the new migration's target version.

- **GIVEN** the current VERSION is `2.7.1`
- **WHEN** this change ships
- **THEN** `src/kit/VERSION` reads `2.8.0` and a `2.7.1-to-2.8.0.md` migration targets it

### Doc Mirrors (Sweep Class — Constitution-Required)

#### R9: SPEC mirror reflects detection + placeholder
`docs/specs/skills/SPEC-fab-setup.md` SHALL mirror the create-mode detection sub-step and the `{TEST_PATHS}` placeholder (Constitution Additional Constraints: a skill change MUST carry its SPEC mirror).

- **GIVEN** the fab-setup skill gained create-mode detection
- **WHEN** the SPEC mirror is read
- **THEN** the create-mode flow documents reading marker files → deriving `{TEST_PATHS}` (non-interactive) and the `{TEST_PATHS}` placeholder in the substitution list

#### R10: Memory documents the new behavior and migration
`docs/memory/distribution/setup.md` SHALL document that create-mode now detects/fills `test_paths` (non-interactive, with a visible note; unrecognized stacks left empty). `docs/memory/distribution/migrations.md` SHALL record the new `2.7.1-to-2.8.0` migration (it enumerates individual migrations — confirmed at apply). After any `docs/memory/` write, the byte-stable index SHALL be regenerated via `fab memory-index` and verified with `fab memory-index --check`.

- **GIVEN** the change touched create-mode and added a migration
- **WHEN** the distribution memory files are read
- **THEN** `setup.md` describes the detection/fill behavior and `migrations.md` lists the `2.7.1-to-2.8.0` migration
- **AND** `fab memory-index --check` passes

### Non-Goals

- **No Go binary change** — detection is skill/migration prompt logic (Constitution I); `impact.go` already consumes any non-empty `test_paths` verbatim. No `_cli-fab.md` update and no Go-test obligation are triggered.
- **No interactive prompt for test_paths** — detection is non-interactive by design (user explicitly requested this).
- **No hardcoded default glob** — an unanchored substring (`**/*test*`) was rejected; unrecognized stacks leave the value empty rather than guess.

### Design Decisions

1. **Anchored language detection over a default glob**: derive the pattern from on-disk markers anchored to each language's test convention — *Why*: an unanchored substring produces a confidently-wrong impact number (worse than absent); anchoring is what makes the classification reliable — *Rejected*: a case-insensitive `**/*test*` default (miscounts `attestation.go`, `latest.go`).
2. **Standing comment block survives a filled value**: examples live as a comment above the active key, not inline on the key line — *Why*: the user keeps an editing reference even after detection writes a value — *Rejected*: inline-only examples on the key line (lost once the value is written).
3. **Rust / unrecognized → empty, not guessed**: Rust tests are inline `#[cfg(test)]`, not glob-addressable — *Why*: a guess would be doubly wrong (matches `attestation.rs`, misses the real inline tests) — *Rejected*: a substring fallback for unrecognized stacks.

## Tasks

### Phase 1: Scaffold + Skill (independent files)

- [x] T001 [P] Replace the `test_paths` block (lines ~14-18) in `src/kit/scaffold/fab/project/config.yaml` with the standing annotated comment block (verbatim from intake §1) above an active key line carrying the `{TEST_PATHS}` placeholder <!-- R1 -->
- [x] T002 [P] In `src/kit/skills/fab-setup.md` Config Create-Mode: add the non-interactive detection sub-step to step 2 (reproduce the detection table marker→ecosystem→patterns from intake §2), add `{TEST_PATHS}` to step 4's substitution list with the preserve-comment-block note, and add the detected/empty note to step 7 / Config Output <!-- R3 --> <!-- R4 --> <!-- R5 -->

### Phase 2: Migration + VERSION

- [x] T003 Create `src/kit/migrations/2.7.1-to-2.8.0.md` following the `2.2.0-to-2.3.0.md` shape (Summary / Pre-check / Changes / Verification): refresh the scaffold comment block (sentinel-guarded on the `# Examples (uncomment/adapt the line for your stack):` line), detect+fill `test_paths` via the §2 table only when absent/empty, preserve a non-empty user value, idempotent, report lines mirroring create-mode <!-- R6 --> <!-- R7 -->
- [x] T004 Bump `src/kit/VERSION` from `2.7.1` to `2.8.0` <!-- R8 -->

### Phase 3: Doc Mirrors (sweep class)

- [x] T005 [P] Update `docs/specs/skills/SPEC-fab-setup.md` to mirror the create-mode detection sub-step (read marker files → derive `{TEST_PATHS}`, non-interactive) and the `{TEST_PATHS}` placeholder in the substitution list <!-- R9 -->
- [x] T006 [P] Update `docs/memory/distribution/setup.md` to document create-mode auto-detects/fills `test_paths` (non-interactive, visible note, unrecognized → empty) <!-- R10 -->
- [x] T007 [P] Update `docs/memory/distribution/migrations.md` to record the new `2.7.1-to-2.8.0` migration (enumerated catalog: add the description-frontmatter entry + a dedicated subsection) <!-- R10 -->

### Phase 4: Index Regeneration

- [x] T008 Run `fab memory-index` then `fab memory-index --check` to regenerate and verify the byte-stable memory index <!-- R10 -->

## Execution Order

- T001 and T002 are independent (different files) — parallelizable.
- T003 depends conceptually on T001 (the migration's comment-block refresh must match the scaffold block) — write T001 first.
- T005, T006, T007 are independent files; T008 must run after T006 and T007 (it regenerates the index over the memory writes).

## Acceptance

### Functional Completeness

- [ ] A-001 R1: The scaffold's `test_paths` section is a standing annotated comment block (Go/Python/JS-TS/Java-Kotlin examples, `:(glob)`/anti-substring note) above an active key line with the `{TEST_PATHS}` placeholder
- [ ] A-002 R2: Substituting `{TEST_PATHS}` preserves the comment block and replaces only the active key line
- [ ] A-003 R3: fab-setup create-mode step 2 has a non-interactive detection sub-step with the marker→ecosystem→pattern table; no test_paths prompt is added
- [ ] A-004 R4: `{TEST_PATHS}` is listed in step 4's placeholder substitution alongside the existing three placeholders, with the preserve-comment-block note
- [ ] A-005 R5: step 7 / Config Output has the detected-ecosystem note and the "no convention detected → empty" note
- [ ] A-006 R6: `src/kit/migrations/2.7.1-to-2.8.0.md` exists with Summary/Pre-check/Changes/Verification, refreshing the comment block and filling test_paths via the detection table
- [ ] A-007 R8: `src/kit/VERSION` reads `2.8.0`
- [ ] A-008 R9: `docs/specs/skills/SPEC-fab-setup.md` mirrors the detection sub-step and `{TEST_PATHS}` placeholder
- [ ] A-009 R10: `docs/memory/distribution/setup.md` and `docs/memory/distribution/migrations.md` document the new behavior and migration

### Behavioral Correctness

- [ ] A-010 R3: Detection is non-interactive — the skill derives the value from markers and never prompts for test_paths
- [ ] A-011 R7: The migration preserves a non-empty user test_paths and only refreshes the comment block in that case

### Scenario Coverage

- [ ] A-012 R3: Each ecosystem row in the detection table maps a concrete marker to an anchored pattern set (Go→`**/*_test.go`, Python→`**/test_*.py`+`**/*_test.py`, JS/TS→spec/test infixes, Java/Kotlin→`**/src/test/**`, .NET→`**/*Tests.cs`+`**/*Test.cs`, Rust/unknown→empty)
- [ ] A-013 R6: A repo with a recognized marker and absent/empty test_paths gets the pattern filled by the migration

### Edge Cases & Error Handling

- [ ] A-014 R7: Migration is idempotent — absent config.yaml → skip; sentinel present → comment-refresh no-op; Rust/unrecognized → empty + note
- [ ] A-015 R3: Multi-marker repos take the union of patterns; Rust (inline tests) is deliberately left empty with a note explaining why

### Code Quality

- [ ] A-016 Pattern consistency: The migration follows the existing migration file shape and the scaffold comment matches intake §1 verbatim; canonical sources only (`src/kit/`), never `.claude/skills/`
- [ ] A-017 No unnecessary duplication: The detection table is the single reference reproduced in the skill and referenced by the migration; the scaffold comment block is authored once
- [ ] A-018 Mirror sweep: The skill change carries its SPEC mirror and the two distribution memory files in the same change (Constitution Additional Constraints; code-quality § Sibling & Mirror Sweeps)
- [ ] A-019 Markdown-only / CommonMark: all artifacts are plain markdown/YAML; no Go change, so no `_cli-fab.md`/Go-test obligation

### Documentation Accuracy

- [ ] A-020 R9: The SPEC mirror and memory files accurately describe the shipped create-mode detection and migration behavior (no drift from the skill/migration)

### Cross References

- [ ] A-021 R10: `fab memory-index --check` passes after the memory writes; memory↔memory links use the bundle-relative `/...` form

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Detection is non-interactive (auto-fill + visible note), not a prompt | User explicitly asked "non-interactively if possible"; marker signal is strong; value trivially editable and migration re-runnable | S:90 R:80 A:80 D:80 |
| 2 | Confident | Anchored language→pattern detection table (Go/Python/JS-TS/Java-Kotlin/.NET; Rust & unknown → empty) | Direction approved in discussion; anchoring is what makes classification reliable | S:80 R:75 A:75 D:75 |
| 3 | Certain | Migration skips non-empty test_paths and is sentinel-guarded for the comment refresh | Constitution III; established `2.2.0-to-2.3.0` / `1.9.1-to-1.9.2` pre-check shape | S:85 R:85 A:95 D:90 |
| 4 | Confident | New migration `2.7.1-to-2.8.0.md` + VERSION bump to 2.8.0 (next minor) | Current VERSION is 2.7.1; additive config-only change is a minor bump per the migration cadence | S:75 R:80 A:85 D:80 |
| 5 | Confident | No Go binary change — detection is skill/migration prompt logic | Constitution I; `impact.go` already handles any non-empty `test_paths` | S:80 R:75 A:90 D:85 |
| 6 | Certain | `migrations.md` enumerates individual migrations → the new entry is required (Open Question resolved) | Read at apply: the file's `description:` lists each migration by range and carries dedicated per-migration subsections (e.g. `### 2.6.6-to-2.7.0`) | S:95 R:80 A:95 D:90 |
| 7 | Tentative | Unrecognized/inline-test stacks (e.g. Rust) leave test_paths empty rather than guessing | Rust tests aren't glob-addressable; a guess would be doubly wrong; empty = today's safe collapsed breakdown | S:70 R:80 A:70 D:55 |

7 assumptions (2 certain, 4 confident, 1 tentative).
