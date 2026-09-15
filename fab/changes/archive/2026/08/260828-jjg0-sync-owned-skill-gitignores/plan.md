# Plan: Sync-Owned Skill Gitignores

**Change**: 260828-jjg0-sync-owned-skill-gitignores
**Intake**: `intake.md`

## Requirements

### Distribution: Scaffold fragment

#### R1: Fragment ships no agent-directory ignores
`src/kit/scaffold/fragment-.gitignore` MUST NOT contain any of the directory tokens `/.agents`, `/.claude`, `/.cursor`, `/.opencode`, `/.codex`, `/.kimi`, `/.gemini` (or the `# Optional - ignore agent specific folders` header), and MUST contain the non-directory line `/.claude/settings.local.json`. The existing lines `.fab-*`, `!fab/.fab-version`, `.status.yaml.lock` are unchanged.

- **GIVEN** a fresh project with no `.gitignore`
- **WHEN** `fab sync` runs
- **THEN** the created `.gitignore` contains exactly the fragment's non-comment lines (`.fab-*`, `!fab/.fab-version`, `.status.yaml.lock`, `/.claude/settings.local.json`) and no `/.claude`-style directory line

### Distribution: Generated per-target manifest

#### R2: Every fired deploy target gets a generated `.gitignore` manifest
After deploying to a target, `fab sync` MUST write `{BaseDir}/.gitignore` (mode 0644, trailing newline) whose content is: two header comment lines, the self-entry `/.gitignore`, then one anchored entry per skill **actually deployed** in this run, in `listSkills` order — `/{name}/` for `directory` format, `/{name}.md` for `flat` format. The file is whole-file owned: it is overwritten on every sync and is byte-stable across two consecutive syncs with the same kit. A target that is skipped (no candidate CLI found) MUST NOT get a manifest written or modified.

- **GIVEN** `FAB_AGENTS=claude` and a kit with skills `fab-new`, `_preamble`
- **WHEN** `fab sync` runs
- **THEN** `.claude/skills/.gitignore` exists and its non-comment lines are exactly `/.gitignore`, `/_preamble/`, `/fab-new/`
- **AND** `.opencode/commands/.gitignore` and `.agents/skills/.gitignore` do not exist

- **GIVEN** the same setup and a second `fab sync`
- **WHEN** the manifest is re-read
- **THEN** its bytes are identical to the first run

- **GIVEN** `FAB_AGENTS=opencode`
- **WHEN** `fab sync` runs
- **THEN** `.opencode/commands/.gitignore` non-comment lines are `/.gitignore`, `/_preamble.md`, `/fab-new.md`

#### R3: Manifest write failures fail the sync
A failed manifest write MUST surface as a non-zero `fab sync` (joined into `deploySkills`'s returned error), consistent with the jznd fail-loud scaffold contract.

- **GIVEN** a target `BaseDir` that is read-only after skills were deployed
- **WHEN** the manifest write is attempted
- **THEN** `deploySkills` returns a non-nil error naming the target

### Distribution: Manifest-scoped stale pruning

#### R4: Stale pruning removes only previously-manifested entries
`cleanStaleSkills` MUST read the target's existing manifest before deployment and prune an entry only when it is (a) present in that previous manifest and (b) absent from the current kit list. Entries never recorded by fab (user-added skills or commands) MUST NOT be removed. Manifest parsing skips comment lines and the `/.gitignore` self-entry and reverses the format-specific entry shape (`/x/` → `x`, `/x.md` → `x`).

- **GIVEN** `.claude/skills/` containing `fab-new/`, `my-team-skill/`, and a manifest listing only `/fab-new/` and `/old-fab-skill/`, and a kit whose only skill is `fab-new`
- **WHEN** `fab sync` runs
- **THEN** `my-team-skill/` still exists, `fab-new/` exists, and nothing else was removed (there was no `old-fab-skill/` on disk — a manifest entry with no directory is a no-op)

- **GIVEN** `.claude/skills/` containing `fab-new/` and `fab-old/`, a manifest listing both, and a kit whose only skill is `fab-new`
- **WHEN** `fab sync` runs
- **THEN** `fab-old/` is removed and the new manifest lists only `/fab-new/`

- **GIVEN** `.opencode/commands/` containing `fab-new.md`, `mine.md`, and a manifest listing `/fab-new.md` and `/gone.md`
- **WHEN** `fab sync` runs with kit `fab-new`
- **THEN** `mine.md` survives

#### R5: No previous manifest ⇒ prune nothing, print a note
When a fired target has no manifest yet, `cleanStaleSkills` MUST prune nothing. If the directory already held at least one entry other than the kit's skills before this sync, sync MUST print exactly one line: `Note: {rel} has no fab manifest yet — stale fab skills (if any) were not pruned this run; they will be from the next sync.` (`{rel}` = path relative to repo root). A fresh, empty target prints nothing.

- **GIVEN** `.agents/skills/` containing `fab-new/` and `unknown/` and no `.gitignore`
- **WHEN** `fab sync` runs with kit `fab-new`
- **THEN** `unknown/` survives, the note is printed once, and the new manifest lists `/fab-new/` only

- **GIVEN** a target directory that does not exist yet
- **WHEN** `fab sync` runs
- **THEN** no note is printed

### Distribution: Migration and version

#### R6: Migration `2.22.0-to-2.23.0` strips the historical lines and regenerates
A migration file `src/kit/migrations/2.22.0-to-2.23.0.md` MUST exist, following the shape of `2.15.1-to-2.15.2.md` (Summary / Pre-check / Changes / Verification), and `src/kit/VERSION` MUST read `2.23.0`. The migration MUST: (1) skip when not a git repo or no root `.gitignore`; (2) remove from the root `.gitignore` only lines whose directory token normalizes to one of the seven historical tokens (leading slash optional, trailing `/` or `/*` tolerated) plus the `# Optional - ignore agent specific folders` header when present, never touching `!` negation lines, printing each removed line; (3) run `fab sync`; (4) list entries in each fired target that are absent from its new manifest and ask the user which (if any) to delete — deleting nothing without confirmation; (5) verify with `git status --porcelain` that no deployed fab skill path is untracked; (6) print the worktree note and offer to run `fab sync` in each sibling from `git worktree list --porcelain`. It MUST be sentinel-guarded (a re-run after success is a no-op: no historical tokens remain and manifests exist).

- **GIVEN** a root `.gitignore` containing `/.claude`, `/.agents/`, `.opencode/*`, `!/.claude/commands/`, `/dist/`
- **WHEN** the migration's step 2 runs
- **THEN** `/.claude`, `/.agents/`, `.opencode/*` are removed; `!/.claude/commands/` and `/dist/` remain; three removal lines are printed

#### R7: fab-kit's own checkout is migrated in this change
The repo's root `.gitignore` MUST no longer contain the six agent-dir lines, the `/.gemini` line, or their comment block; the untracked `.gemini/` tree MUST be deleted from the developer checkout if present. After `fab sync` with the new binary, `git status --porcelain` MUST show no `.claude/skills/*`, `.agents/skills/*`, or `.opencode/commands/*` entries, and `.claude/settings.local.json` MUST be ignored.

- **GIVEN** this worktree after the change is applied and `just install` + `fab sync` have run
- **WHEN** `git status --porcelain` runs
- **THEN** no path under `.claude/skills/`, `.agents/skills/`, `.opencode/commands/` appears and `git check-ignore -q .claude/settings.local.json` exits 0

### Documentation

#### R8: Every behavior claim about skill-dir ignoring is updated
Memory, specs, kit skills, and project files MUST describe the new mechanism and MUST NOT claim `.claude/` (or `.claude/skills/`) "is gitignored" as a directory. Specifically: `docs/memory/distribution/kit-architecture.md` § Agent Skill Deployment (manifest + scoped prune), `docs/memory/distribution/setup.md` (`.gitignore entries` row; note the directory-token dedup class is empty in the shipped fragment; add `/.claude/settings.local.json` to the non-directory class), `docs/memory/distribution/migrations.md` (new `2.22.0-to-2.23.0` section), `docs/specs/architecture.md` (tree comment line ~36 and § `.gitignore` Guidance), `src/kit/skills/_cli-fab.md` `sync` row, `fab/project/context.md` and `fab/project/constitution.md` Principle V wording (constitution: wording-only, dated HTML comment, no version bump). A repo-wide grep for `gitignored` / `is gitignored` / `(gitignored)` / `/.claude` across `src/kit/`, `docs/`, `fab/project/`, `README.md`, and `*_test.go` comments MUST find no stale directory-ignore claim.

- **GIVEN** the change is applied
- **WHEN** `grep -rn "gitignored" src/kit docs fab/project README.md src/go` runs
- **THEN** every hit about `.claude`/`.agents`/`.opencode` describes the generated per-target manifest, not a directory ignore

### Non-Goals

- Retiring the `.opencode/commands/` target — user verified OpenCode does not surface `.agents/skills/` as commands.
- Collapsing `.claude/skills/` into `.agents/skills/` — follow-up candidate.
- Removing the mqiq directory-token dedup code in `scaffold.go` — kept, tests kept, only memory wording changes.
- Prefix-based namespacing of fab skills — rejected in discussion.

### Design Decisions

#### The generated `.gitignore` is the manifest
**Decision**: `fab sync` writes one generated `.gitignore` per fired target listing exactly the deployed entries, and `cleanStaleSkills` reads that same file as the record of what fab owns.
**Why**: One file, one truth — the ignore list and the prune scope cannot drift from each other, and the file is already ignored via its own self-entry with no root-`.gitignore` change needed.
**Rejected**: A separate `.fab-manifest` file (already covered by `.fab-*`, but two files can disagree); prefix namespacing (`git-*`/`docs-*`/`code-*` collide with user skills and bake a naming rule into an ignore rule); ignoring symlinks only (gitignore cannot match by file type).
*Introduced by*: 260828-jjg0-sync-owned-skill-gitignores

#### No-manifest first run prunes nothing
**Decision**: When a target has no manifest yet, sync prunes nothing and prints one note; the migration owns one-time cleanup with user confirmation.
**Why**: Guessing ownership from "not in the kit list" is exactly the data-loss bug being fixed; the cost is that pre-existing stale fab skills linger until the migration or a manual delete.
**Rejected**: Treating any entry matching a historical fab skill name as owned (the binary has no name history); pruning everything on first run (destroys user skills).
*Introduced by*: 260828-jjg0-sync-owned-skill-gitignores

## Tasks

### Phase 1: Setup

- [x] T001 Edit `src/kit/scaffold/fragment-.gitignore`: remove the `# Optional - ignore agent specific folders` header and the six `/.agents … /.kimi` lines; append `/.claude/settings.local.json`. Update any test asserting the old fragment content (`src/go/fab-kit/internal/sync_integration_test.go` `TestSync_FullRunProducesExpectedTree` / `TestSync_GitignoreNegationSurvivesFullSync` if they check `/.claude`). <!-- R1 -->

### Phase 2: Core Implementation

- [x] T002 In `src/go/fab-kit/internal/skills.go` add manifest helpers: `manifestEntry(format, name string) string` (`/{name}/` or `/{name}.md`), `manifestPath(baseDir) string`, `readSkillManifest(baseDir, format string) (map[string]bool, bool)` (returns owned set + whether a manifest existed; skips comments and the self-entry), `writeSkillManifest(baseDir, format string, deployed []string) error` (header two comment lines, `/.gitignore`, entries in order, trailing newline, 0644). Have `syncAgentSkills` return the list of skills it actually deployed (created+repaired+ok) so the manifest reflects reality. <!-- R2, R3 -->
- [x] T003 Rework `deploySkills`/`cleanStaleSkills` ordering per target: `prev, had := readSkillManifest(...)` → `syncAgentSkills` → `cleanStaleSkills(baseDir, format, prev, currentKitSet, had, repoRoot)` (prune only `prev − current`; when `!had` prune nothing and print the R5 note if the dir held a non-kit entry) → `writeSkillManifest` (join its error into the returned error). Keep `cleanLegacyAgents` unchanged. <!-- R4, R5, R3 -->
- [x] T004 Tests in `src/go/fab-kit/internal/skills_test.go` (and `sync_integration_test.go` where the full `Sync()` path is needed): manifest written per format with exact content; byte-stable across two runs; skipped target gets no manifest; user-added dir/file survives (`directory` + `flat`); previously-manifested skill absent from kit is pruned; manifest entry with no on-disk dir is a no-op; no-manifest run prunes nothing and prints the note exactly once; empty fresh target prints no note; manifest write failure propagates. Rewrite `TestCleanStaleSkills_Directory` / `TestCleanStaleSkills_Flat` for the new signature. Run `go test ./src/go/fab-kit/...`. <!-- R2, R3, R4, R5 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Write `src/kit/migrations/2.22.0-to-2.23.0.md` per R6 (model on `2.15.1-to-2.15.2.md` and `2.15.7-to-2.15.8.md` for the worktree sweep), and bump `src/kit/VERSION` to `2.23.0`. Include the seven historical tokens verbatim and the exact token-normalization rule. <!-- R6 -->
- [x] T006 Repo cleanup: edit the root `.gitignore` (remove `/.agents`–`/.kimi`, the `/.gemini` line and its 4-line comment); `rm -rf .gemini` if present; `just install` (or the repo's equivalent to rebuild+cache the binary) then `fab sync`; verify `git status --porcelain` shows no `.claude/skills`, `.agents/skills`, `.opencode/commands` paths and `git check-ignore -q .claude/settings.local.json` passes. Do NOT commit the generated nested `.gitignore` files (they self-ignore). <!-- R7 -->

### Phase 4: Polish

- [x] T007 Documentation updates per R8: `docs/memory/distribution/kit-architecture.md` (§ Agent Skill Deployment — new "Generated manifest and scoped pruning" paragraph + DD lift), `docs/memory/distribution/setup.md` (`.gitignore entries` row, dedup-class wording, settings.local.json line), `docs/memory/distribution/migrations.md` (new `2.22.0-to-2.23.0` section in the existing per-migration style), `docs/specs/architecture.md` (tree comment + § `.gitignore` Guidance), `src/kit/skills/_cli-fab.md` `sync` row, `fab/project/context.md`, `fab/project/constitution.md` Principle V (+ dated comment, no version bump). Run `fab memory-index` afterwards and commit the regenerated indexes with the content. <!-- R8 -->
- [x] T008 Behavior-claim sweep: `grep -rn -i "gitignored\|/\.claude\b\|/\.agents\b\|/\.opencode\b" src/kit docs fab/project README.md src/go --include=*.md --include=*.go` and fix every remaining stale directory-ignore claim, including `*_test.go` comments and any user-facing string literals. <!-- R8 -->

## Execution Order

- T002 blocks T003; T003 blocks T004 and T006 (T006 needs the new binary installed)
- T001, T005 are independent of the Go work and may run alongside T002–T004
- T007–T008 after T006 so the docs describe the verified behavior

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fragment-.gitignore` has none of the seven directory tokens and contains `/.claude/settings.local.json`; `.fab-*`, `!fab/.fab-version`, `.status.yaml.lock` intact
- [x] A-002 R2: Each fired target has a generated `.gitignore` with two header comments, `/.gitignore`, and one anchored entry per deployed skill in the correct per-format shape; skipped targets untouched
- [x] A-003 R3: A manifest write failure makes `deploySkills` return a non-nil error
- [x] A-004 R4: `cleanStaleSkills` removes only entries in (previous manifest − current kit); user-added entries survive in both formats
- [x] A-005 R5: No-manifest run prunes nothing and prints the note exactly once when non-kit entries existed; empty target prints nothing
- [x] A-006 R6: `src/kit/migrations/2.22.0-to-2.23.0.md` exists with the six steps and `src/kit/VERSION` is `2.23.0`
- [x] A-007 R7: Repo root `.gitignore` cleaned; `git status --porcelain` clean of deployed skill paths after `fab sync`; `.claude/settings.local.json` ignored
- [x] A-008 R8: All listed docs updated; grep sweep finds no stale directory-ignore claim (remaining `/.claude`-as-directory-ignore hits are point-in-time dated analyses under `docs/specs/findings/` and the deliberately retained mqiq dedup-machinery tests in `scaffold_test.go` — both historical/synthetic, not behavior claims)

### Behavioral Correctness

- [x] A-009 R4: A user skill dir placed in `.claude/skills/` before a sync still exists after two consecutive syncs
- [x] A-010 R2: Two consecutive syncs produce byte-identical manifests

### Removal Verification

- [x] A-011 R1: No code path still emits `/.claude` or sibling directory tokens into a root `.gitignore`

### Scenario Coverage

- [x] A-012 R4: Test covers pruning of a formerly-deployed skill after the kit drops it (`directory` and `flat`)
- [x] A-013 R6: Migration text includes the exact seven tokens, the normalization rule, the `!`-line carve-out, and the worktree note

### Edge Cases & Error Handling

- [x] A-014 R4: A manifest entry with no matching on-disk entry is a silent no-op (no error, no count)
- [x] A-015 R5: Corrupt/unparseable manifest lines are skipped, not fatal

### Code Quality

- [x] A-016 Pattern consistency: helpers follow `scaffold.go`'s small-helper + doc-comment style; no function exceeds ~50 lines without reason
- [x] A-017 No unnecessary duplication: entry-shape logic lives in one `manifestEntry`/inverse pair used by both writer and reader
- [x] A-018 No magic strings: manifest header lines and the note format are named constants
- [x] A-019 CLI ⇒ docs + tests: `_cli-fab.md` `sync` row updated and every `.go` change has test coverage
- [x] A-020 Canonical source only: no edits under `.claude/skills/`; migration ships as `src/kit/migrations/` markdown
- [x] A-021 Owner-or-pointer: memory states the manifest rule once (kit-architecture.md); setup.md and specs point at it rather than restating

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change replaces the prune-all `cleanStaleSkills` body in the same diff that adds the manifest-scoped one, leaving no redundant code behind. The mqiq directory-token dedup machinery (`gitignoreIsDirectoryToken` + helpers in `src/go/fab-kit/internal/scaffold.go` and its `scaffold_test.go` tests) lost its shipped-fragment consumer but is deliberately retained per plan `## Non-Goals` / Assumption 10 — not a deletion candidate.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Migration's one-time stale sweep = list non-manifest entries and ask, no byte-identity heuristic | Intake open question resolved toward simplicity; a list-and-ask step is cheap and safe, a heuristic adds code for a one-time event | S:60 R:85 A:80 D:70 |
| 2 | Confident | `syncAgentSkills` returns the deployed list rather than the manifest assuming the whole kit list | A skill missing from the cache is skipped with a WARN today; listing it in the manifest would claim ownership of nothing | S:70 R:85 A:85 D:75 |
| 3 | Confident | R5 note fires only when the dir held a non-kit entry | An all-fab directory has nothing to warn about; avoids noise on every fresh checkout | S:65 R:90 A:80 D:70 |
| 4 | Confident | Constitution Principle V change is wording-only, no version bump | The MUST rules are unchanged; only the parenthetical "gitignored" description is corrected — matches the udwv precedent | S:70 R:90 A:85 D:80 |
| 5 | Tentative | `TestSync_FullRunProducesExpectedTree` asserts the old fragment lines and needs updating | Not verified before dispatch; T001 says "if they check" | S:50 R:95 A:60 D:60 |

5 assumptions (0 certain, 4 confident, 1 tentative).
