# Plan: Publish FKF Standard to docs/site

**Change**: 260720-g538-publish-fkf-standard-site
**Intake**: `intake.md`

## Requirements

### Publishing: Authoritative FKF standard at `docs/site/fkf.md`

#### R1: Canonical published standard
A new `docs/site/fkf.md` SHALL be the canonical, authoritative FKF standard. Its body MUST be the normative content of the current `src/kit/reference/fkf.md` (§2 Conformance / §3 Concept Documents / §5 Index Files / §6 Log Files / §7 Cross-links / §8 Versioning — original section numbers preserved, gaps intentional), and its shared header MUST replace the extract's "Single-sourcing note" with framing that reads correctly in BOTH homes (the published page at `https://shll.ai/fab-kit/fkf` and the kit-cache copy at `$(fab kit-path)/reference/fkf.md`). The header MUST state: what FKF is (retaining the "What this is" + "Scope: `docs/memory/` only" blockquotes), that this file is the canonical standard (maintained at `docs/site/fkf.md`, published at `https://shll.ai/fab-kit/fkf`, shipped verbatim as `$(fab kit-path)/reference/fkf.md`), that rationale/history live in the non-normative companion (linked absolutely: `https://github.com/sahil87/fab-kit/blob/main/docs/specs/fkf.md`), that section numbering is preserved from the original design doc, and the edit workflow (edit `docs/site/fkf.md` → `scripts/sync-fkf.sh` → CI test fails on divergence).

- **GIVEN** the repo after this change
- **WHEN** `docs/site/fkf.md` is read as the shll.ai page or as the kit-cache copy
- **THEN** every existing "FKF §N" citation (§2/§3/§5/§6/§7/§8 and their sub-sections) resolves identically to today's extract
- **AND** the header makes the file's canonical status, its two homes, and the sync workflow explicit

#### R2: Closed-set link audit
Every link in `docs/site/fkf.md` that leaves `docs/site/**` MUST be an absolute `https://…` URL (readme-extraction standard, docs/site closed-set rules 1–2). No relative images (none exist).

- **GIVEN** the completed `docs/site/fkf.md`
- **WHEN** grepping for relative link targets (`](./`, `](../`, `](docs/`) outside fenced code examples
- **THEN** no rendered link leaves `docs/site/**` via a relative path

#### R3: Shipped kit copy is a byte-copy
`src/kit/reference/fkf.md` SHALL be byte-identical to `docs/site/fkf.md`. Its path and section anchors do not change, so all deployed-skill citations of `$(fab kit-path)/reference/fkf.md` §N remain valid with zero skill-file edits.

- **GIVEN** the completed change
- **WHEN** `cmp docs/site/fkf.md src/kit/reference/fkf.md` runs
- **THEN** the files are byte-identical

### Sync Enforcement: script + drift-guard test

#### R4: `scripts/sync-fkf.sh`
A new `scripts/sync-fkf.sh` MUST mirror `scripts/sync-skill.sh` exactly (header comment explaining the canonical file / shipped copy / drift-guard, `set -euo pipefail`, `cd "$(dirname "$0")/.."`, `SRC`/`DEST` vars, `cp -f`, one echo) with `SRC="docs/site/fkf.md"`, `DEST="src/kit/reference/fkf.md"`. It MUST be executable.

- **GIVEN** an edit to `docs/site/fkf.md`
- **WHEN** `scripts/sync-fkf.sh` runs from any CWD
- **THEN** `src/kit/reference/fkf.md` is refreshed to a byte-copy and one `synced FKF standard: …` line is echoed

#### R5: Go drift-guard test
A test in the `fab` module MUST assert byte-equality of `docs/site/fkf.md` and `src/kit/reference/fkf.md`. Since neither file is embedded (both live outside the module root `src/go/fab/`), the test MUST read both via repo-relative paths resolved by walking up from the test's working directory (the `findRepoFile` pattern shared by `skill_test.go` / `lifecycle_collision_test.go`; same approach as `internal/score/changetypes_doc_test.go`'s `findDocFile`). On divergence the failure message MUST tell the user to run `scripts/sync-fkf.sh`. The test runs on every `go test ./...` (CI's `ci.yml` invokes it).

- **GIVEN** `docs/site/fkf.md` edited without running the sync script
- **WHEN** `go test ./src/go/fab/cmd/fab/` (or `go test ./...`) runs
- **THEN** the drift-guard test fails, naming both files and instructing `run scripts/sync-fkf.sh`
- **GIVEN** the two files byte-identical
- **WHEN** the same test runs
- **THEN** it passes

### Companion: `docs/specs/fkf.md` slims to non-normative rationale

#### R6: Slimmed design companion
`docs/specs/fkf.md` SHALL become the non-normative design companion: rewritten header (points at the standard's two homes and the new sync contract, replacing the "Shipped normative extract / update BOTH files" note); KEEPS §1 Relationship to OKF, §4 Bundle Organization, §9 Non-Scope, §10 Adoption/Migration, §11 Glossary with original numbering; REPLACES each normative section §2/§3/§5/§6/§7/§8 with a pointer stub at the original top-level heading (heading text unchanged so GitHub anchors keep resolving) whose body is a one-line redirect to the standard. Unique non-normative design rationale that lived only inside the replaced sections (the "Why a constant" / "Why recommended" / "Why present-truth" / "Why C-lite" asides, the Starlight lesson, the cap-escalation incident history, and §6.4's freeze-on-write design rationale) is retained beneath the corresponding stub as clearly-labeled rationale notes — zero normative rule text remains anywhere in the file. The existing citation `docs/specs/fkf.md` §10 (from `docs-hydrate-memory.md:192`) stays valid because §10 remains.

- **GIVEN** the slimmed `docs/specs/fkf.md`
- **WHEN** a reader follows an inbound `§2`/`§3`/`§5`/`§6`/`§7`/`§8` citation
- **THEN** the original heading exists and redirects to the standard in one hop
- **AND** no RFC-2119 rule text, conformance list, format contract, or enforcement semantics remain in the file
- **GIVEN** a reader looking for FKF design rationale (OKF lineage, rejected alternatives, migration history)
- **WHEN** they read the companion
- **THEN** that content is still present (nothing rationale-only was deleted from the repo)

### Reference Sweep: restatements of the arrangement

#### R7: Repo-wide arrangement sweep
Every restatement of the OLD arrangement ("shipped normative extract of docs/specs/fkf.md", "update BOTH files") MUST be updated to the new one; citations of `$(fab kit-path)/reference/fkf.md` §N and of `docs/specs/fkf.md` §N stay unchanged (stubs keep the latter one-hop resolvable). In scope for apply: `docs/specs/index.md` fkf row (now: design rationale + history companion; normative standard at `docs/site/fkf.md` → shll.ai/fab-kit/fkf) and `docs/memory/distribution/kit-architecture.md` line ~62 provenance comment (now: byte-copy of `docs/site/fkf.md`, synced by `scripts/sync-fkf.sh`, drift-guarded). `src/kit/skills/docs-distill-memory.md:33`'s parenthetical stays true and needs no edit (verify). Other memory-file mentions are hydrate-stage work, not apply's. If any `src/kit/skills/*.md` file is edited, its `docs/specs/skills/SPEC-*.md` mirror MUST be updated in the same change.

- **GIVEN** the completed apply
- **WHEN** grepping repo-wide for `specs/fkf`, `reference/fkf`, `update BOTH`, `normative extract`
- **THEN** no non-memory, non-change-folder file still describes reference/fkf.md as an extract of docs/specs/fkf.md or carries the "update BOTH files" duty (except `docs/memory/**` lines deferred to hydrate)

### README: hub cross-link

#### R8: README links the FKF standard
`README.md` SHALL gain a cross-link to the FKF standard written naturally as `docs/site/fkf.md` (the site rewrites it to `/fab-kit/fkf`; GitHub resolves it) — readme-extraction rule 8 (the README is the hub), placed in the deeper-docs section alongside the existing docs links.

- **GIVEN** the updated README
- **WHEN** the shll.ai pull rewrites README → docs/site links
- **THEN** the FKF link renders as `/fab-kit/fkf` on the site and resolves on GitHub

### Non-Goals

- Changing the normative section set (§6.4 freeze-on-write and other spec-only enforcement semantics are NOT added to the standard; shipped semantics remain documented in `docs/memory/pipeline/schemas.md`)
- Renaming the "extract" vocabulary across skills/SPECs where it does not restate the old sync arrangement
- Any fab CLI, skill-behavior, kit-packaging, or shll.ai-side change
- Full memory hydration (only the kit-architecture.md provenance line is apply-scope; the rest is the hydrate stage)

### Design Decisions

#### Stub headings keep original text; pointer lives in the body
**Decision**: Replaced sections keep their exact original heading text (e.g. `## 2. Conformance`); the one-line "moved" pointer is the section body.
**Why**: GitHub derives anchors from heading text — a `— moved` suffix (the intake's inline example) would change the anchor and defeat the intake's stated goal ("so inbound anchors don't silently 404").
**Rejected**: Suffixing headings with "— moved" — breaks `#2-conformance`-style anchors.
*Introduced by*: 260720-g538-publish-fkf-standard-site

#### Drift-guard test lives in `cmd/fab` beside the skill.md guard
**Decision**: `src/go/fab/cmd/fab/fkf_sync_test.go`, reusing the existing `findRepoFile` helper.
**Why**: `cmd/fab` already hosts the sibling sync-drift guard (`skill_test.go`) and the repo-root-walk helper; co-locating keeps one pattern and zero new helpers.
**Rejected**: A new package near `internal/score/changetypes_doc_test.go` — would duplicate `findRepoFile`.
*Introduced by*: 260720-g538-publish-fkf-standard-site

## Tasks

### Phase 1: Setup

- [x] T001 Create `scripts/sync-fkf.sh` mirroring `scripts/sync-skill.sh` (repo-root cd, `set -euo pipefail`, `cp -f docs/site/fkf.md src/kit/reference/fkf.md`, one echo, explanatory header comment); `chmod +x` <!-- R4 -->

### Phase 2: Core Implementation

- [x] T002 Create `docs/site/fkf.md`: rewritten shared header (What-this-is + Scope blockquotes, canonical-standard statement, absolute companion link, numbering note, edit workflow) + the current extract's §2/§3/§5/§6/§7/§8 body verbatim; make the OKF and companion references absolute `https://` links <!-- R1 -->
- [x] T003 Run `scripts/sync-fkf.sh` to overwrite `src/kit/reference/fkf.md`; verify byte-equality with `cmp` <!-- R3 -->
- [x] T004 Add `src/go/fab/cmd/fab/fkf_sync_test.go` (byte-equality drift guard via `findRepoFile`, failure message names `scripts/sync-fkf.sh`); run `go test ./cmd/fab/` in `src/go/fab` <!-- R5 -->
- [x] T005 Slim `docs/specs/fkf.md`: new companion header; keep §1/§4/§9/§10/§11 verbatim; replace §2/§3/§5/§6/§7/§8 with pointer stubs at original headings + retained rationale notes (Why-constant, Starlight lesson + cap incident, Why-recommended, Why-present-truth, hand-merge failure mode, Why-C-lite, freeze-on-write rationale); zero normative text <!-- R6 -->

### Phase 3: Integration & Edge Cases

- [x] T006 [P] Update `docs/specs/index.md` fkf row: rationale+history companion; normative standard at `docs/site/fkf.md` → shll.ai/fab-kit/fkf, shipped as `$(fab kit-path)/reference/fkf.md` <!-- R7 -->
- [x] T007 [P] Update `docs/memory/distribution/kit-architecture.md` line ~62 tree-diagram provenance comment: byte-copy of `docs/site/fkf.md`, synced by `scripts/sync-fkf.sh`, drift-guarded <!-- R7 -->
- [x] T008 [P] Add README cross-link to the FKF standard as `docs/site/fkf.md` in the deeper-docs list (§ "Read more" block, lines ~660) <!-- R8 -->
- [x] T009 Reference sweep verification: grep repo-wide for `specs/fkf`, `reference/fkf`, `update BOTH`, `normative extract`; confirm no stale restatement of the old arrangement outside `docs/memory/**` (hydrate scope) and change folders; confirm `src/kit/skills/docs-distill-memory.md:33` needs no edit; run closed-set link audit on `docs/site/fkf.md` <!-- R7, R2 -->

### Phase 4: Polish

- [x] T010 Run `go test ./...` for the `src/go/fab` module (full-module pass incl. the new drift guard) <!-- R5 -->

## Execution Order

- T001 blocks T003 (script must exist to sync)
- T002 blocks T003, T004 (canonical file must exist)
- T006–T008 are independent [P] after T005

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `docs/site/fkf.md` exists with the rewritten dual-home header and the extract's §2/§3/§5/§6/§7/§8 body, original numbering preserved
- [ ] A-002 R3: `src/kit/reference/fkf.md` is byte-identical to `docs/site/fkf.md` (`cmp` clean)
- [ ] A-003 R4: `scripts/sync-fkf.sh` exists, is executable, and mirrors `sync-skill.sh` structure with the fkf SRC/DEST
- [ ] A-004 R5: A drift-guard test asserting byte-equality of the two files exists in the `fab` module and passes
- [ ] A-005 R6: `docs/specs/fkf.md` carries the new companion header, keeps §1/§4/§9/§10/§11, and stubs §2/§3/§5/§6/§7/§8 at their original headings
- [ ] A-006 R7: `docs/specs/index.md` fkf row and `docs/memory/distribution/kit-architecture.md` provenance line describe the new arrangement
- [ ] A-007 R8: README cross-links the FKF standard as `docs/site/fkf.md`

### Behavioral Correctness

- [ ] A-008 R5: With the two files divergent, the drift-guard test fails and its message instructs running `scripts/sync-fkf.sh`
- [ ] A-009 R6: Zero normative rule text remains in `docs/specs/fkf.md` (no conformance list, format contracts, or enforcement semantics — pointer stubs + rationale only)
- [ ] A-010 R1: Every existing "FKF §N" citation target (§2/§3/§5/§6/§7/§8 headings and sub-headings) resolves identically in `docs/site/fkf.md` as in today's extract

### Scenario Coverage

- [ ] A-011 R2: No rendered link in `docs/site/fkf.md` leaves `docs/site/**` via a relative path (closed-set audit clean; code-fence examples exempt)
- [ ] A-012 R7: Repo-wide grep shows no remaining "update BOTH files" duty or "extract of docs/specs/fkf.md" restatement outside `docs/memory/**` (hydrate scope) and archived change folders
- [ ] A-013 R5: `go test ./...` passes for the `src/go/fab` module

### Edge Cases & Error Handling

- [ ] A-014 R6: Citations `docs/specs/fkf.md §10` (docs-hydrate-memory.md:192) and inbound §2–§8 textual citations still resolve (kept section / one-hop stub redirect)
- [ ] A-015 R7: No `src/kit/skills/*.md` file was edited — or, if one was, its `docs/specs/skills/SPEC-*.md` mirror was updated in the same change

### Code Quality

- [ ] A-016 Pattern consistency: new script/test follow the `sync-skill.sh` / `skill_test.go` patterns (naming, structure, failure-message shape)
- [ ] A-017 No unnecessary duplication: the drift-guard test reuses `findRepoFile` instead of re-declaring a repo-root walker
- [ ] A-018 No `.claude/skills/` edits (deployed copies untouched; canonical sources only)
- [ ] A-019 No user-data restructuring introduced (no migration required — content-only change; kit packaging ships `src/kit/` verbatim)
- [ ] A-020 Go change ships its test (the new test IS the Go change; no magic strings — paths are named constants)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Stub headings keep original text exactly; the "moved" pointer is the body line (deviates from the intake's `## 2. Conformance — moved` inline example) | The intake's stated goal — inbound anchors must not 404 — is only met by unchanged heading text; the example contradicts its own goal | S:70 R:90 A:85 D:80 |
| 2 | Confident | Rationale-only asides inside the replaced sections (Why-* blockquotes, Starlight lesson, cap-incident history, §6.4 freeze-on-write why) are retained beneath the stubs rather than deleted | Intake: "Rationale/history moves nowhere — it stays in docs/specs/fkf.md"; the ~250-line target only fits with retention; "zero normative text" still holds — only rule text is removed | S:60 R:85 A:80 D:70 |
| 3 | Confident | §6.4's normative freeze-on-write semantics are not added to the standard and their rule text leaves the spec; shipped semantics remain documented in `docs/memory/pipeline/schemas.md` | Intake assumption 3 fixes the normative set to the extract's (§6.4 was never in it); memory already carries the full shipped contract, so nothing is lost | S:65 R:75 A:85 D:75 |
| 4 | Certain | Textual citations of `docs/specs/fkf.md §N` elsewhere (Go comments, migrations, SPEC files) stay unchanged | Intake explicitly scopes the sweep to arrangement restatements, "not the citations, which stay valid" — the stubs keep them one-hop resolvable | S:85 R:95 A:90 D:85 |
| 5 | Confident | Drift-guard test placed at `src/go/fab/cmd/fab/fkf_sync_test.go` reusing `findRepoFile` | Intake says implementer's choice following the existing pattern; `cmd/fab` hosts both the sibling guard and the helper | S:70 R:95 A:90 D:80 |
| 6 | Confident | README link goes in the deeper-docs "Read more" list (not the line-11 onboarding hub row) | Line 11 is user-onboarding (install/workflows/glossary); the FKF standard is a deep reference like the specs links already listed there; intake allows "line 11 and/or the docs section" | S:60 R:95 A:80 D:70 |

6 assumptions (1 certain, 5 confident, 0 tentative).
