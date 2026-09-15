# Plan: Deployed Kit Content Must Not Cite fab-kit-Only Paths

**Change**: 260910-d5tk-deployed-skills-no-fabkit-paths
**Intake**: `intake.md`

## Requirements

### Governance: Constitution citation rule

#### R1: Principle V gains a deployed-content citation rule
`fab/project/constitution.md` § V. Portability MUST gain a new paragraph stating that deployed kit content (`src/kit/skills/`, `src/kit/templates/`, `src/kit/migrations/`, `src/kit/scaffold/`, `src/kit/reference/`) MUST NOT reference files that exist only in the fab-kit repository; MAY cite another kit skill or helper, a `fab` command, a `fab/project/` file, a `$(fab kit-path)/…` deployed asset, or a documented host-project convention path; MUST NOT cite fab-kit's own `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` as an authority or rule owner; and that owned rules are carried into the deployed file by restatement, after which the fab-kit doc points at the skill. External repositories' doc paths are covered by the same prohibition — an external standard or spec is named in prose or by URL, never by a bare relative path. The Governance block MUST read `**Version**: 1.8.0` and `**Last Amended**: 2026-09-10`, and a dated `<!-- 2026-09-10 (260910-d5tk): … -->` note MUST be appended in the existing comment style.

- **GIVEN** the constitution at 1.7.0
- **WHEN** the amendment is applied
- **THEN** § V contains the citation paragraph, Version reads 1.8.0, Last Amended reads 2026-09-10, and a dated governance comment explains the bump

#### R2: code-quality and code-review carry the boundary
`fab/project/code-quality.md` MUST (a) extend the "Stating an owned rule AND pointing at its owner" anti-pattern with the fab-kit-only-owner exception (the deployed skill restates and becomes the owner for deployed purposes; the fab-kit doc points at the skill), (b) add a sibling anti-pattern "Citing fab-kit-only paths from deployed content" naming the allowed citation targets and the host convention paths, and (c) qualify the § Sibling Sweeps closing sentence with "within the deployed set; across the deploy boundary the skill is the owner". `fab/project/code-review.md` § Project-Specific Review Rules MUST gain a must-fix row "Deployed content cites fab-kit-only paths". `src/kit/skills/internal-skill-optimize.md` line 56's Sibling-duplication row and `docs/specs/skills.md` line 1557's Ownership-rule signal MUST carry the same boundary clause.

- **GIVEN** a reviewer reading `code-review.md`
- **WHEN** a `src/kit/**` file names fab-kit's `docs/specs/*` as an authority
- **THEN** the review rules classify it must-fix and point at restating the rule in the skill

### Kit skills: Category A sweep

#### R3: No `src/kit/skills/*.md` line cites a fab-kit-only or external-repo doc path
Every Category A site enumerated in `intake.md` § What Changes § 3 MUST be rewritten per its Disposition column: restate the operative rule inline, drop the citation, or repoint to a deployed owner (`_preamble.md` § CLI-Adapter Dispatch / § Dispatch-Prompt Obligations / § Naming Conventions, `_cli-fab.md` § fab agent / § fab config / § fab docs-index / § fab score). Specifically: the manual-block rule MUST be restated ONCE in `_cli-fab.md` § fab docs-index and the six skill sites (`fab-continue.md:217`, `docs-hydrate-memory.md:37`, `docs-distill-memory.md:61,146,205`, `docs-reorg-memory.md:244`) MUST point there; `_cli-fab.md:442` MUST restate the four-tier cascade one-liner (environment > system `~/.fab-kit/config.yaml` > project > built-in defaults; per-leaf deep merge, empty-skip); `_cli-fab.md:939–941` MUST keep only the `fab skill` contract (raw markdown, byte-stable per release, stderr empty, exit 0) and name "the shll toolkit-wide `skill` standard, published at shll.ai" without a path; `fab-operator.md:158` and `_cli-fab.md:1267` MUST name run-kit's operator-cron spec in prose; `_cli-external.md:54` MUST name "the shll toolkit `skill` standard" in prose; `_preamble.md:481`'s example summary MUST use a placeholder-shaped path (`docs/memory/{domain}/{file}.md`). Category B convention paths (`docs/memory/index.md`, `docs/specs/index.md`, `docs/memory/_shared/removed-domains.md`, `docs/memory/_shared/utilities.md`, the illustrative `docs/memory/pipeline/runtime/x.md`) MUST remain untouched. Only `src/kit/skills/` is edited — never `.agents/skills/` or `.claude/skills/`.

- **GIVEN** the swept tree
- **WHEN** `grep -rnoE '\bdocs/(specs|memory|site)/[A-Za-z0-9_./-]+\.md\b|\bsrc/go/[A-Za-z0-9_./-]+' src/kit/skills/` runs
- **THEN** every match is a Category B convention path, a `docs/memory/**/index.md` or `docs/specs/**/index.md` tier index, or a placeholder-shaped example

### Kit migrations: retroactive sweep

#### R4: Historical migrations cite no fab-kit doc path
The seven migration files `2.2.0-to-2.3.0`, `2.4.2-to-2.5.0`, `2.5.5-to-2.6.0`, `2.6.6-to-2.7.0`, `2.11.0-to-2.12.0`, `2.12.1-to-2.13.0`, `2.19.4-to-2.20.0` under `src/kit/migrations/` MUST be edited so that `docs/specs/fkf.md` citations become `$(fab kit-path)/reference/fkf.md` (section references preserved) and `docs/specs/stage-models.md` / `docs/specs/naming.md` citations are dropped (the surrounding sentence keeps its operative content; a "See …" sentence that carries nothing else is removed). The `2.6.6-to-2.7.0` probe snippet's `docs/memory/probe/x.md` and `docs/memory/probe/index.md` paths are host-shaped test fixtures and MUST remain. Migration semantics (the instructions a customer executes) MUST NOT change.

- **GIVEN** a customer applying `/fab-setup migrations` across 2.2.0 → 2.20.0
- **WHEN** they follow a "See …" pointer in any migration
- **THEN** it resolves to a deployed asset (`$(fab kit-path)/…`) or is absent — never to fab-kit's `docs/specs/`

### Go: portability guard

#### R5: A fab-kit module test fails on any deploy-boundary violation
A new test file `src/go/fab-kit/cmd/fab/kit_portability_test.go` (package `main`, reusing `findRepoFile`) MUST walk every regular file under `src/kit/` except `src/kit/VERSION` and `src/kit/reference/fkf.md`, match each line against `\bdocs/(specs|memory|site)/[A-Za-z0-9_./-]+\.md\b` and `\bsrc/go/[A-Za-z0-9_./-]+`, and fail — with a `file:line — cites a repo-local doc path <p>; restate the rule, or name the external repo/standard in prose` message per hit — on any match that is not exempt. Exempt: (i) the named allowlist constant `hostConventionPaths` = {`docs/memory/index.md`, `docs/specs/index.md`, `docs/memory/_shared/removed-domains.md`, `docs/memory/_shared/utilities.md`}; (ii) any `docs/memory/**/index.md` or `docs/specs/**/index.md` (tier indexes are generated in the host at every depth); (iii) placeholder-shaped paths — a segment containing `{`…`}` or a bare `x.md` leaf. There is NO marker or attribution-token escape hatch. The matcher MUST be exercised by a fixture test over a `t.TempDir()` tree containing one violating file, one allowlisted file, one tier-index file, and one placeholder-shaped file, asserting exactly the violating hit is reported. `go test ./cmd/fab/... ./internal/...` in `src/go/fab-kit` MUST pass on the swept tree, and `gofmt -l` MUST print nothing for the new file. No `fab` CLI command is added; `_cli-fab.md` receives no signature change.

- **GIVEN** the pre-sweep tree
- **WHEN** the new test runs
- **THEN** it fails naming each Category A site
- **GIVEN** the swept tree
- **WHEN** the new test runs
- **THEN** it passes, and the fixture test reports exactly the violating fixture

### Docs: direction-of-ownership flips

#### R6: fab-kit docs point at the deployed owner
Where a spec or memory file says the skills "point here" for a rule the skill now restates, the doc MUST be updated to point at the skill (or state both carry it with the skill canonical), keeping its own design content intact (Constitution VI): `docs/specs/harness-adapters.md` § Dispatch-prompt obligations, `docs/specs/stage-models.md` § Skill wiring, `docs/specs/config.md` (cascade), `docs/specs/change-types.md` (taxonomy pointer). `docs/specs/skills.md` § New Skill Checklist MUST gain item 9 (portability of citations). `docs/specs/architecture.md` § Directory Structure MUST gain one sentence stating `docs/specs/` and `docs/memory/` are dev-repo-only and must not be cited from `src/kit/`. Memory files are hydrate's responsibility (intake § Affected Memory) and are NOT edited at apply, except `docs/memory/memory-docs/docs-index.md` where the manual-block ownership sentence would otherwise be false after T005 — that one sentence MAY be flipped at apply.

- **GIVEN** a fab-kit maintainer reading `docs/specs/harness-adapters.md` § Dispatch-prompt obligations
- **WHEN** they look for the deployed rule text
- **THEN** the spec names `_preamble.md` § Dispatch-Prompt Obligations as the deployed owner

### Non-Goals

- Deriving Category B convention paths from `docs_index.roots` — separate, softer issue
- Any `fab sync` behavior change, any CLI signature change, any `fab` subcommand for the guard
- Editing deployed copies under `.agents/skills/` or `.claude/skills/` (Constitution V) — do NOT run `fab sync` in this worktree either: the released binary deploys the released kit, not this branch
- Rewriting `src/kit/reference/fkf.md` or `docs/site/fkf.md` — the standard's URL-attributed provenance stays; the guard excludes the byte-copy by name
- Memory hydration (Affected Memory list) — hydrate stage

### Design Decisions

#### Deployed Skills Own Their Rules Across the Deploy Boundary
**Decision**: Within `src/kit/`, the owner-or-pointer convention is bounded by the deploy boundary — a deployed file may point only at something that also deploys (kit skills, `fab` commands, `fab/project/` files, `$(fab kit-path)/…` assets, host convention paths). When the genuine owner is a fab-kit-only doc, the skill restates the operative rule and becomes the owner for deployed purposes; the doc points back at the skill.
**Why**: Following owner-or-pointer in good faith produced thirteen dead pointers across four PRs, because inside fab-kit every path resolves. The rule that prevents drift had no notion of where the file would be read.
**Rejected**: Shipping the specs into customer repos (Constitution V: the kit content tree is never copied); an attribution marker allowing bare external paths (a bare relative path is equally dead in a customer repo, and the customer report flagged exactly such an attributed path).
*Introduced by*: 260910-d5tk-deployed-skills-no-fabkit-paths

#### The Guard Has No Escape Hatch
**Decision**: The portability test recognizes only a named allowlist, tier-index paths, and placeholder shapes. No `portability-ignore` marker, no attribution-token recognition.
**Why**: Every exemption mechanism is a second place to be wrong. Naming a repo or standard in prose costs nothing and reads correctly from any repo.
**Rejected**: `<!-- portability-ignore: <repo> -->` same-line marker (adds prose noise to skills and a second exemption surface to audit).
*Introduced by*: 260910-d5tk-deployed-skills-no-fabkit-paths

## Tasks

### Phase 1: Setup

- [x] T001 Write the guard first (red before green): create `src/go/fab-kit/cmd/fab/kit_portability_test.go` in package `main` — walk `src/kit/` via `findRepoFile(t, "src/kit")` skipping `VERSION` and `reference/fkf.md`; per-line regexes for `docs/(specs|memory|site)/….md` and `src/go/…`; `hostConventionPaths` allowlist constant; tier-index and placeholder exemptions; per-hit `file:line — cites a repo-local doc path <p>; restate the rule, or name the external repo/standard in prose`; plus a `t.TempDir()` fixture test (violating / allowlisted / tier-index / placeholder files → exactly one hit). Run `cd src/go/fab-kit && gofmt -l ./cmd/fab && go test ./cmd/fab/ -run 'Portab' -v` and confirm the live-tree test FAILS naming the Category A sites while the fixture test passes. <!-- R5 -->

### Phase 2: Core Implementation

- [x] T002 Amend `fab/project/constitution.md`: append the citation paragraph to § V. Portability (intake § What Changes § 1 wording, external-repo paths included in the prohibition per Assumption 3), set `**Version**: 1.8.0` and `**Last Amended**: 2026-09-10`, append the dated `<!-- 2026-09-10 (260910-d5tk): … 1.7.0 → 1.8.0 -->` governance comment after the flt3 note. <!-- R1 -->
- [x] T003 [P] Edit `fab/project/code-quality.md` (extend the owner-or-pointer bullet with the fab-kit-only-owner exception; add the "Citing fab-kit-only paths from deployed content" bullet; qualify the § Sibling Sweeps closing sentence) and `fab/project/code-review.md` (add the "Deployed content cites fab-kit-only paths" must-fix row). Also add the boundary clause to `src/kit/skills/internal-skill-optimize.md:56` and `docs/specs/skills.md:1557`. <!-- R2 -->
- [x] T004 [P] Sweep `src/kit/skills/_preamble.md` lines 305, 320, 356, 450, 481, 534 per the intake Disposition table (drop stage-models clauses; rephrase § CLI-Adapter Dispatch opener so `_preamble.md` is the canonical procedure and `_cli-fab.md` § fab dispatch owns runtime details; drop the "Per docs/specs/harness-adapters.md" attribution; placeholder path in the example summary; repoint the change-types sentence to `_cli-fab.md` § fab score / `fab status set-change-type`). <!-- R3 -->
- [x] T005 Sweep `src/kit/skills/_cli-fab.md` lines 212, 285 (×2), 381, 415, 419, 442, 637, 939, 941, 1267, 1445 per the Disposition table, AND restate the manual-block rule once in § fab docs-index (the block between `<!-- fab docs-index:manual:start -->`/`:end -->` is the one hand-managed region of a generated primary landing, preserved verbatim by the generator, never rewritten by skills) so T006 can point at it. Keep the `fab skill` contract facts at 939–941; trim embed/sync-script/drift-guard-test provenance; name the shll `skill` standard in prose. <!-- R3 -->
- [x] T006 Sweep the remaining skill sites: `_cli-agents.md:97` (repoint to `_preamble.md` § CLI-Adapter Dispatch), `_cli-external.md:54` (shll `skill` standard in prose), `fab-operator.md:158` (run-kit's operator-cron spec in prose), and the six manual-block sites `fab-continue.md:217`, `docs-hydrate-memory.md:37`, `docs-distill-memory.md:61,146,205`, `docs-reorg-memory.md:244` (point at `_cli-fab.md` § fab docs-index). Leave every Category B path untouched. <!-- R3 -->
- [x] T007 [P] Sweep `src/kit/migrations/{2.2.0-to-2.3.0,2.4.2-to-2.5.0,2.5.5-to-2.6.0,2.6.6-to-2.7.0,2.11.0-to-2.12.0,2.12.1-to-2.13.0,2.19.4-to-2.20.0}.md`: `docs/specs/fkf.md` → `$(fab kit-path)/reference/fkf.md` (keep § refs); drop `docs/specs/stage-models.md` / `docs/specs/naming.md` citations without changing migration instructions; keep the 2.6.6 probe fixture paths. <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Flip direction of ownership in `docs/specs/harness-adapters.md` § Dispatch-prompt obligations, `docs/specs/stage-models.md` § Skill wiring, `docs/specs/config.md` (cascade), `docs/specs/change-types.md` (taxonomy pointer) — one pointer sentence each toward the deployed owner, design content untouched; add item 9 to `docs/specs/skills.md` § New Skill Checklist; add the dev-repo-only sentence to `docs/specs/architecture.md` § Directory Structure; flip the one manual-block ownership sentence in `docs/memory/memory-docs/docs-index.md` if T005 made it false. <!-- R6 -->
- [x] T009 Verify green: `cd src/go/fab-kit && gofmt -l . && go test ./cmd/fab/... ./internal/...` passes (guard included); `grep -rnoE '\bdocs/(specs|memory|site)/[A-Za-z0-9_./-]+\.md\b|\bsrc/go/[A-Za-z0-9_./-]+' src/kit/` prints only Category B paths, tier indexes, and placeholder shapes; `git status` shows no changes under `.agents/` or `.claude/`. <!-- R5 -->

## Execution Order

- T001 first (the guard must be observed red on the pre-sweep tree)
- T003, T004, T007 are independent of each other and of T005
- T005 blocks T006 (the six manual-block pointers target the § T005 writes) and T008 (the docs-index memory flip)
- T009 last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab/project/constitution.md` § V contains the deployed-content citation paragraph (allowed targets, forbidden targets, restate-then-point rule, external-repo paths covered); Version 1.8.0; Last Amended 2026-09-10; dated d5tk governance comment present
- [x] A-002 R2: `code-quality.md` has the owner-or-pointer exception, the new anti-pattern bullet, and the qualified Sibling Sweeps sentence; `code-review.md` has the must-fix row; `internal-skill-optimize.md:56` and `docs/specs/skills.md:1557` carry the boundary clause
- [x] A-003 R3: every Category A site in the intake Disposition table is rewritten as specified; the manual-block rule appears once in `_cli-fab.md` § fab docs-index and the six sites point there
- [x] A-004 R4: the seven migration files cite no `docs/specs/*` path; fkf citations use `$(fab kit-path)/reference/fkf.md` with section numbers preserved
- [x] A-005 R5: `src/go/fab-kit/cmd/fab/kit_portability_test.go` exists with the live-tree walk, `hostConventionPaths`, tier-index and placeholder exemptions, the loud per-hit message, and the fixture test
- [x] A-006 R6: the four specs point at the deployed owner; `skills.md` checklist item 9 and the `architecture.md` sentence are present

### Behavioral Correctness

- [x] A-007 R5: `go test ./cmd/fab/... ./internal/...` in `src/go/fab-kit` passes on the swept tree and `gofmt -l` prints nothing for the new file
- [x] A-008 R5: the fixture test asserts that only the violating fixture reports hits (one per matched path — a docs/specs path and a src/go path) and none for the allowlisted, tier-index, or placeholder fixtures
- [x] A-009 R4: migration instructions are semantically unchanged — only citation sentences differ in the diff

### Scenario Coverage

- [x] A-010 R3: the repo-wide grep over `src/kit/` returns only Category B paths, `**/index.md` tier indexes, and placeholder shapes
- [x] A-011 R3: `fab-operator.md:158`, `_cli-fab.md:1267`, `_cli-fab.md:941`, `_cli-external.md:54` contain no `docs/…` path and name run-kit / the shll `skill` standard in prose

### Edge Cases & Error Handling

- [x] A-012 R5: `src/kit/reference/fkf.md` and `src/kit/VERSION` are skipped by the walk (the guard passes although fkf.md carries URL-attributed fab-kit paths)
- [x] A-013 R3: Category B paths (`docs/memory/index.md`, `docs/specs/index.md`, `_shared/removed-domains.md`, `_shared/utilities.md`, `pipeline/runtime/x.md`) are byte-identical to `main`

### Code Quality

- [x] A-014 Pattern consistency: the new test follows `clifab_doc_test.go` conventions (package `main`, `findRepoFile`, regexp constants, table-style assertions)
- [x] A-015 No unnecessary duplication: `findRepoFile` is reused, not re-implemented
- [x] A-016 Canonical source only: no edits under `.agents/skills/` or `.claude/skills/`; no `fab sync` run
- [x] A-017 Owner-or-pointer within the deployed set: the manual-block rule is stated exactly once in `src/kit/skills/` (in `_cli-fab.md`), pointed at elsewhere
- [x] A-018 Go changes ship tests / CLI unchanged: the guard is a plain test, `_cli-fab.md` has no new command signature, no production Go code changed
- [x] A-019 Sibling sweep complete: every statement of the owner-or-pointer convention in `src/kit/skills/`, `fab/project/`, and `docs/specs/` carries the boundary clause

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `None — this change adds a new guard test and prose-rule amendments without making existing code redundant`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Tier indexes at any depth (`docs/memory/**/index.md`, `docs/specs/**/index.md`) are exempt from the guard, not only the two root indexes | `fab docs-index` generates an `index.md` in every domain and sub-domain of the host; the 2.6.6 migration's probe snippet cites `docs/memory/probe/index.md`, which is host-shaped, not fab-kit-only | S:80 R:90 A:85 D:80 |
| 2 | Confident | The guard is written first and observed failing on the pre-sweep tree (T001), rather than only trusting the fixture test | Constitution VII / code-quality test-alongside; a guard that was never seen red proves nothing about the live matcher | S:75 R:95 A:90 D:80 |
| 3 | Confident | `_cli-fab.md:212`'s `expected_min` reference is restated as "documentation-only, not on the score path" with the path dropped | The intake Disposition table names this; `fab score` output already omits `expected_min` | S:75 R:90 A:85 D:80 |
| 4 | Confident | `docs/memory/memory-docs/docs-index.md`'s single ownership sentence MAY be flipped at apply; all other memory edits wait for hydrate | Leaving a sentence that is false after T005 until hydrate would make review's `documentation_accuracy` check fail on a known-stale claim; hydrate still owns the rest of the Affected Memory list | S:70 R:90 A:80 D:70 |
| 5 | Confident | Migration `2.4.2-to-2.5.0.md:73` "lands the full FKF tooling (`docs/specs/fkf.md`)" keeps its sentence with the path swapped to `$(fab kit-path)/reference/fkf.md` | The deployed reference is a byte-copy of the same standard, so the pointer stays accurate | S:80 R:90 A:85 D:85 |

5 assumptions (0 certain, 5 confident, 0 tentative).
