# Plan: Conform Repo to Standardized Toolkit Name — shll toolkit

**Change**: 260718-udwv-shll-toolkit-rename
**Intake**: `intake.md`

## Requirements

<!-- Prose-only rename "sahil87 toolkit" → "shll toolkit" (sahil87/shll#56). The
     intake's What-Changes map enumerates every in-scope occurrence with exact line
     numbers and byte-exact replacement text. Requirements below restate that map as
     RFC-2119 statements, one domain section per surface. -->

### README: Toolkit Name Conformance

#### R1: README blockquote matches the readme-extraction standard's canonical line
The README toolkit blockquote (`README.md:3`) MUST be replaced, byte-identical, with the standard's canonical line — the head order (H1 → blockquote → badges) MUST be preserved (single-line content replacement, no reordering).

- **GIVEN** `README.md:3` reads `> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.`
- **WHEN** the conformance edit is applied
- **THEN** line 3 reads exactly `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`
- **AND** line 1 remains the `# Fab Kit` H1 and line 5 remains the badges row (order preserved)

#### R2: README install-prose old-name reference is renamed
The single README prose occurrence of the old name (`README.md:21`) MUST be updated `sahil87 toolkit` → `shll toolkit`; nothing else on the line changes.

- **GIVEN** `README.md:21` ends `To install the entire sahil87 toolkit instead:`
- **WHEN** the sweep is applied
- **THEN** the line ends `To install the entire shll toolkit instead:`

### Skill Bundle: Canonical + Embedded Copy

#### R3: Canonical skill bundle composition line is renamed
`docs/site/skill.md:53` MUST have its link text `[@sahil87 toolkit]` → `[shll toolkit]` — the `@` sigil drops with the old-name styling; the URL and surrounding prose are unchanged.

- **GIVEN** `docs/site/skill.md:53` reads `fab is one member of the [@sahil87 toolkit](https://shll.ai) and composes with its siblings`
- **WHEN** the sweep is applied
- **THEN** the line reads `fab is one member of the [shll toolkit](https://shll.ai) and composes with its siblings`
- **AND** the `(https://shll.ai)` URL is unchanged and the line count of the bundle is unchanged (≤150-line budget test unaffected)

#### R4: Embedded skill bundle copy is re-synced to match the canonical
After editing `docs/site/skill.md`, the committed embed copy `src/go/fab/cmd/fab/skill.md` MUST be regenerated via `bash scripts/sync-skill.sh` so the byte-equality drift guard `TestSkillEmbedMatchesCanonical` passes. The copy MUST NOT be hand-edited.

- **GIVEN** `docs/site/skill.md` has been edited (R3)
- **WHEN** `bash scripts/sync-skill.sh` is run
- **THEN** `src/go/fab/cmd/fab/skill.md` is byte-identical to `docs/site/skill.md`
- **AND** `go test ./cmd/fab/ -run 'TestSkill' -count=1` (run from the `src/go/fab` module root) passes, and the package's full test suite stays green

### Kit Skill + SPEC Mirror

#### R5: Kit skill prose and its constitution-mandated SPEC mirror are renamed together
The phrase `sahil87 toolkit-wide` in `src/kit/skills/_cli-fab.md:583` MUST be updated to `shll toolkit-wide`, and its mirror `docs/specs/skills/SPEC-_cli-fab.md:33` MUST receive the identical update in the same change (Constitution Additional Constraints: skill-file edits MUST update the corresponding SPEC-*.md). The surrounding `shll docs/site/standards/skill.md` path references stay verbatim (identifiers). Deployed copies under `.claude/skills/` are gitignored and MUST NOT be edited.

- **GIVEN** both files carry `the sahil87 toolkit-wide \`skill\` standard`
- **WHEN** the sweep is applied
- **THEN** both `src/kit/skills/_cli-fab.md:583` and `docs/specs/skills/SPEC-_cli-fab.md:33` read `shll toolkit-wide` and the `shll docs/site/standards/skill.md` references are unchanged

### Constitution: Cosmetic Article Wording

#### R6: Toolkit Standards article wording is renamed with a dated amendment comment
In `fab/project/constitution.md:38`, `This tool is part of the sahil87 toolkit` MUST become `This tool is part of the shll toolkit`. Nothing else in the article body changes — in particular the `sahil87/shll repository's docs/site/standards/ tree` canonical-source reference stays verbatim (identifier). `Last Amended` stays `2026-07-18` (bump target equals today's current value), and `Version` stays `1.4.0` (cosmetic wording, no normative rule change). A new dated HTML amendment comment MUST be appended after the existing amendment comments, noting the cosmetic rename.

- **GIVEN** `constitution.md:38` opens `This tool is part of the sahil87 toolkit and MUST conform ...`
- **WHEN** the edit is applied
- **THEN** it opens `This tool is part of the shll toolkit and MUST conform ...`, the `sahil87/shll` canonical-source reference on the same line is unchanged, the governance line still reads `**Version**: 1.4.0 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-07-18`
- **AND** a new `<!-- 2026-07-18 (260718-udwv): ... -->` amendment comment is appended after the existing ones, and the existing historical amendment comments (including 260717-y8it's at lines 56–61, which contain old-name wording) are untouched

### Non-Goals

- **Identifiers untouched** — `sahil87/tap` formula strings, `github.com/sahil87` and `raw.githubusercontent.com/sahil87` URLs, `img.shields.io/.../sahil87/...` badge URLs, the `githubRepo = "sahil87/fab-kit"` / `REPO="sahil87/fab-kit"` constants, and the `sahil87/shll` canonical-source references (constitution, `scripts/sync-skill.sh` comment, `helpdump.go` comment) all stay verbatim.
- **No memory edits** — `docs/memory/distribution/distribution.md`'s two old-name passages are hydrate's job, not apply's (Affected Memory).
- **No `.claude/skills/` edits** — gitignored deployed copies regenerated by `fab sync`.
- **No historical-record edits** — `fab/changes/` archives and the constitution's existing dated amendment comments stay as-is.
- **No behavior change / no test goldens / no `schema_version` bump** — zero user-visible Go strings or CLI help text contain the old name (verified by grep at intake); the only `src/go/` diff is the embedded markdown asset.

### Design Decisions

1. **`@` sigil dropped in the skill-bundle link text**: `[@sahil87 toolkit]` → `[shll toolkit]`, not `[@shll toolkit]` — *Why*: the mapping is name-level (`sahil87 toolkit` → `shll toolkit`) and the `@` was old-name styling; the standard's own phrasing is `the [shll toolkit](https://shll.ai)` — *Rejected*: a literal splice preserving `@`, which matches neither the new name nor the standard.
2. **Constitution version/date frozen**: cosmetic wording bumps no version and today equals the current `Last Amended` value, so both stay byte-identical; a dated amendment comment records the change — *Why*: the file's own 260601/260611 amendment convention bumps version only on a normative rule change — *Rejected*: a version bump (there is no new MUST-rule).

## Tasks

<!-- Prose-only edits. Phase 2 edits are independent files ([P]); Phase 3 is the
     sync-then-verify sequence that depends on the docs/site/skill.md edit. -->

### Phase 2: Core Implementation

- [x] T001 [P] Replace the README blockquote at `README.md:3` with `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` (byte-exact); preserve head order H1→blockquote→badges <!-- R1 -->
- [x] T002 [P] Update `README.md:21` install prose `sahil87 toolkit` → `shll toolkit` <!-- R2 -->
- [x] T003 [P] Update `docs/site/skill.md:53` link text `[@sahil87 toolkit]` → `[shll toolkit]` (drop the `@`; URL and surrounding prose unchanged) <!-- R3 -->
- [x] T004 [P] Update `src/kit/skills/_cli-fab.md:583` `sahil87 toolkit-wide` → `shll toolkit-wide` (leave the `shll docs/site/standards/skill.md` path reference verbatim) <!-- R5 -->
- [x] T005 [P] Update `docs/specs/skills/SPEC-_cli-fab.md:33` `sahil87 toolkit-wide` → `shll toolkit-wide` (SPEC mirror of T004) <!-- R5 -->
- [x] T006 [P] Update `fab/project/constitution.md:38` article body `This tool is part of the sahil87 toolkit` → `This tool is part of the shll toolkit` (leave the `sahil87/shll` canonical-source reference and the governance line verbatim), then append a dated `<!-- 2026-07-18 (260718-udwv): ... -->` amendment comment after the existing ones noting the cosmetic rename (per sahil87/shll#56, no version bump) <!-- R6 -->

### Phase 3: Integration & Verification

- [x] T007 Re-run `bash scripts/sync-skill.sh` to regenerate the embedded copy `src/go/fab/cmd/fab/skill.md` from the edited canonical `docs/site/skill.md` (depends on T003) <!-- R4 -->
- [x] T008 Run `cd src/go/fab && go test ./cmd/fab/ -run 'TestSkill' -count=1`, then the package's full tests, to confirm `TestSkillEmbedMatchesCanonical` and the bundle line-budget test pass (depends on T007) <!-- R4 -->
- [x] T009 Repo-wide sweep verification: grep for remaining in-scope `sahil87 toolkit` / `sahil87 tool` / `@sahil87` prose (excluding `fab/changes/` archives, `.claude/`, `docs/memory/`, and identifier URLs) to confirm the class is fully swept <!-- R1 R2 R3 R5 R6 -->

## Execution Order

- T001–T006 are independent (`[P]`) — different files, no ordering between them.
- T007 depends on T003 (canonical edit must land before the embed re-sync).
- T008 depends on T007 (test runs against the re-synced copy).
- T009 runs last (verifies all prose edits are complete).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `README.md:3` reads exactly `> Part of the [shll toolkit](https://shll.ai) — see all projects there.` (byte-identical) with head order H1→blockquote→badges preserved
- [x] A-002 R2: `README.md:21` reads `To install the entire shll toolkit instead:`
- [x] A-003 R3: `docs/site/skill.md:53` reads `fab is one member of the [shll toolkit](https://shll.ai) and composes with its siblings`
- [x] A-004 R4: `src/go/fab/cmd/fab/skill.md` is byte-identical to `docs/site/skill.md` after `scripts/sync-skill.sh`
- [x] A-005 R5: both `src/kit/skills/_cli-fab.md:583` and `docs/specs/skills/SPEC-_cli-fab.md:33` read `shll toolkit-wide`, with `shll docs/site/standards/skill.md` references unchanged
- [x] A-006 R6: `constitution.md:38` article body reads `This tool is part of the shll toolkit`; version `1.4.0` and `Last Amended` `2026-07-18` unchanged; a new dated amendment comment is appended

### Behavioral Correctness

- [x] A-007 R4: `TestSkillEmbedMatchesCanonical` passes and the bundle ≤150-line budget test passes; no Go logic changed (only the embedded markdown asset differs)
- [x] A-008 R6: the `sahil87/shll` canonical-source reference on `constitution.md:38` and the existing 260717-y8it amendment comment (lines 56–61) remain verbatim

### Scenario Coverage

- [x] A-009 R4: `cd src/go/fab && go test ./cmd/fab/ -count=1` (full package, from the `src/go/fab` module root) is green <!-- there is no go.mod/go.work at src/go, so src/go/fab is the module root; the package (all 6 TestSkill* tests incl. TestSkillEmbedMatchesCanonical + TestSkillBundle_LineBudget) is green -->

### Edge Cases & Error Handling

- [x] A-010 R1: no identifier surfaces were touched — `sahil87/tap`, `github.com/sahil87`, `raw.githubusercontent.com/sahil87`, `img.shields.io/.../sahil87/...`, `githubRepo`/`REPO` constants, and `sahil87/shll` references remain verbatim (verified by post-edit grep)

### Code Quality

- [x] A-011 Pattern consistency: replacement text matches the existing surrounding prose style (link markdown, sentence flow) exactly as prescribed by the intake's byte-exact map
- [x] A-012 No unnecessary duplication: no new files or utilities created; edits are in-place content replacements
- [x] A-013 Canonical source only: no edits under `.claude/skills/` (gitignored deployed copies); the kit change lives in `src/kit/` (code-quality.md anti-pattern)
- [x] A-014 SPEC-mirror sync: the `src/kit/skills/_cli-fab.md` edit carries its `docs/specs/skills/SPEC-_cli-fab.md` mirror update in the same change (Constitution Additional Constraints; code-quality.md § Sibling & Mirror Sweeps)

### Documentation Accuracy (checklist.extra_categories)

- [x] A-015 Every in-scope prose occurrence in the intake's occurrence map is updated; no in-scope `sahil87 toolkit`/`sahil87 tool`/`@sahil87` prose remains outside archives/.claude/docs-memory/identifiers (grep-verified)

### Cross References (checklist.extra_categories)

- [x] A-016 The canonical (`docs/site/skill.md`) and embedded (`src/go/fab/cmd/fab/skill.md`) skill bundles stay in sync; the kit skill and its SPEC mirror stay in sync

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

<!-- Apply-agent record of graded decisions made while co-generating ## Requirements.
     These carry forward the intake's assumptions that bear on apply scope. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | README blockquote replaced with the byte-exact standard line, head order preserved | Text given verbatim in the intake and verified against the live `shll standards readme-extraction` output at intake time | S:95 R:90 A:100 D:100 |
| 2 | Confident | `[@sahil87 toolkit]` → `[shll toolkit]` in `docs/site/skill.md:53` (the `@` sigil drops with the old name) | Mapping is name-level; a literal `@`-preserving splice would match neither the new name nor the standard's own `the [shll toolkit](https://shll.ai)` phrasing | S:60 R:90 A:80 D:75 |
| 3 | Confident | Sweep includes `src/kit/skills/_cli-fab.md:583` and its constitution-mandated mirror `docs/specs/skills/SPEC-_cli-fab.md:33` | Sweep principle is "wherever the phrase appears as prose"; the constitution requires SPEC mirrors to track skill-file edits (code-quality.md § Sibling & Mirror Sweeps) | S:55 R:85 A:80 D:70 |
| 4 | Confident | Constitution `Last Amended` stays `2026-07-18`, version stays `1.4.0`, dated amendment comment appended | File's own convention: non-normative amendments bump no version but always leave a dated comment; today equals the current `Last Amended` value so the bump is byte-identical | S:60 R:90 A:85 D:70 |
| 5 | Certain | Historical records keep old wording (`fab/changes/` archives and the existing 260717-y8it amendment comment) | Intake freezes archives; dated amendment comments are the same class of historical record | S:55 R:90 A:80 D:70 |

5 assumptions (2 certain, 3 confident, 0 tentative).
