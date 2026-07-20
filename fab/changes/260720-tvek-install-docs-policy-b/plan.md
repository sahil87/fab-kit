# Plan: Install Docs Policy B Conformance

**Change**: 260720-tvek-install-docs-policy-b
**Intake**: `intake.md`

## Requirements

### Docs: docs/site/install.md — "Install the CLI" section

#### R1: CLI install section points to the shll.ai bootstrap
The "Install the CLI" section of `docs/site/install.md` MUST NOT carry the per-formula Homebrew instructions (`brew tap sahil87/tap` / `brew install fab-kit`). It SHALL instead present the shll.ai curl bootstrap in both forms — the fab-kit subset (`curl -fsSL https://shll.ai/install | sh -s -- fab-kit`) and the full-toolkit variant (`curl -fsSL https://shll.ai/install | sh`) — with prose noting the bootstrap installs fab-kit (plus the shll meta-CLI) via Homebrew with tap trust handled automatically, and an absolute `https://shll.ai` link as the canonical install reference. The two-CLI role table (`fab` router / `fab-kit` lifecycle) MUST be kept unchanged (feature content, not install instruction).

- **GIVEN** `docs/site/install.md` currently instructs `brew tap sahil87/tap && brew install fab-kit`
- **WHEN** the section is rewritten
- **THEN** the section carries the `curl -fsSL https://shll.ai/install | sh -s -- fab-kit` bootstrap, the full-toolkit variant, and an absolute https://shll.ai link
- **AND** the `fab`/`fab-kit` role table remains intact

### Docs: docs/site/install.md — companions block

#### R2: Companion install instruction becomes `shll install`, hint example kept verbatim
In the companions block of `docs/site/install.md`, the code block `brew install sahil87/tap/wt sahil87/tap/idea` MUST be replaced with the shll subset-install form `shll install wt idea` (with an absolute https://shll.ai pointer), and the immediately-preceding lead-in sentence ("install from their own formulas") MUST be adjusted to say they install via `shll install` / shll.ai while retaining the independent-projects / own-release-cadences fact. The parenthetical error-hint example on line 39 — ``(`wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt`)`` — MUST be kept byte-for-byte verbatim (Policy A-mandated hint quoted in degradation prose; matches `src/go/fab/cmd/fab/batch_new.go:66`). The wt/idea role table and the graceful-degradation prose MUST be kept.

- **GIVEN** the companions block instructs `brew install sahil87/tap/wt sahil87/tap/idea`
- **WHEN** the block is rewritten
- **THEN** the code block reads `shll install wt idea` and the lead-in points install steps at shll.ai
- **AND** line 39's hint example, the role table, and the degradation prose are unchanged

### Docs: README.md — Prerequisites table row

#### R3: Prerequisites row parenthetical uses the shll form
The wt/idea row of README.md's Prerequisites table (line ~111) MUST replace the parenthetical ``(`brew install sahil87/tap/wt sahil87/tap/idea`)`` with the shll form: ``(`shll install wt idea` — see [shll.ai](https://shll.ai))``. The rest of the cell (worktree isolation / idea backlog description, the Companion-tools anchor link) stays unchanged.

- **GIVEN** the Prerequisites table row carries the per-formula parenthetical
- **WHEN** the row is edited
- **THEN** the cell reads ``Recommended companions (`shll install wt idea` — see [shll.ai](https://shll.ai)) — worktree isolation and the idea backlog; see [Companion tools](#companion-tools)``

### Docs: README.md — Companion tools section

#### R4: Companion tools section points install at shll.ai
README.md's `## Companion tools` section (lines ~643–649) MUST replace the code block `brew install sahil87/tap/wt sahil87/tap/idea` with `shll install wt idea`, and adjust the lead sentence ("installed from their own formulas") to point install steps at an absolute https://shll.ai link while keeping the independent-projects / own-release-cadences fact. All graceful-degradation prose and the role table below it MUST be kept unchanged.

- **GIVEN** the Companion tools section instructs `brew install sahil87/tap/wt sahil87/tap/idea`
- **WHEN** the section is rewritten
- **THEN** the code block reads `shll install wt idea` and the lead sentence links https://shll.ai
- **AND** the degradation prose and role table are unchanged

### Docs: Conformance verification

#### R5: No toolkit per-formula install instructions remain; keeps are untouched
After the edits, a grep of README.md and docs/site/ for `brew install sahil87/tap/` MUST return exactly one occurrence: the mandated hint example in `docs/site/install.md` (the quoted `wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt` line). The third-party prerequisite blocks (`brew install yq jq gh direnv`, `brew install go just` in both files) MUST remain unchanged, README's `## Install` section MUST remain unchanged (already conformant), and the apply-stage diff touches no files outside README.md/docs/site/ (no Go source, no docs/specs, no docs/memory — memory is hydrate's job; later pipeline stages add their own files by design: hydrate updates docs/memory, and the change record under fab/changes/ evolves throughout, so this scope check applies to the apply stage only, not the full PR).

- **GIVEN** all four replacements (R1–R4) are applied
- **WHEN** `grep -rn 'brew install sahil87/tap/' README.md docs/site/` runs
- **THEN** the only hit is the install.md error-hint example
- **AND** `git status` at apply completion (before hydrate/ship) shows only README.md and docs/site/install.md modified

### Non-Goals

- Go source hint strings (`batch_new.go`, `batch_switch.go`, `update.go`, `sync.go`, `doctor.go`, `prereqs.go`) — Policy A's mandated binary half, not violations
- The tap README (`sahil87/homebrew-tap`) — different repo
- `docs/specs/` and `docs/memory/` mentions of the formula reality — internal/historical docs, not site pull surfaces; the memory update is hydrate-stage work, not apply
- Any change to usage/feature content (role tables, graceful-degradation prose)

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite the "Install the CLI" section of `docs/site/install.md` (lines ~10–25): replace the Homebrew framing + `brew tap`/`brew install fab-kit` code block with the shll.ai curl bootstrap pair (subset `sh -s -- fab-kit` + full toolkit) and canonical https://shll.ai link, keeping the two-CLI role table <!-- R1 -->
- [x] T002 In `docs/site/install.md` companions block (lines ~26–30): replace the `brew install sahil87/tap/wt sahil87/tap/idea` code block with `shll install wt idea` and adjust the lead-in sentence to point at `shll install` / https://shll.ai (keeping the independent-projects fact); leave line 39's hint example, the role table, and degradation prose byte-identical <!-- R2 -->
- [x] T003 [P] In `README.md` Prerequisites table (line ~111): replace the wt/idea row's parenthetical with ``(`shll install wt idea` — see [shll.ai](https://shll.ai))`` <!-- R3 -->
- [x] T004 [P] In `README.md` `## Companion tools` (lines ~643–649): replace the code block with `shll install wt idea` and repoint the lead sentence's install steps at https://shll.ai, keeping the release-cadence fact, degradation prose, and role table <!-- R4 -->

### Phase 3: Integration & Edge Cases

- [x] T005 Verify conformance: `grep -rn 'brew install sahil87/tap/\|brew tap sahil87/tap' README.md docs/site/` returns only the install.md hint example; confirm third-party blocks (`yq jq gh direnv`, `go just`) and README `## Install` are untouched; `git status` shows only the two files modified <!-- R5 -->

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `docs/site/install.md` "Install the CLI" carries both shll.ai bootstrap forms and an absolute https://shll.ai link; no `brew tap sahil87/tap` / `brew install fab-kit` instruction remains
- [ ] A-002 R2: install.md companions block instructs `shll install wt idea` with a shll.ai pointer; lead-in no longer says "install from their own formulas"
- [ ] A-003 R3: README Prerequisites wt/idea row parenthetical reads `shll install wt idea` with a shll.ai link
- [ ] A-004 R4: README Companion tools code block reads `shll install wt idea` and the lead sentence links https://shll.ai

### Behavioral Correctness

- [ ] A-005 R1: the two-CLI role table (`fab` / `fab-kit`) in install.md is unchanged
- [ ] A-006 R2: install.md line-39 hint example is byte-identical to before (and still matches `batch_new.go:66`); wt/idea role table and degradation prose unchanged
- [ ] A-007 R4: README Companion tools keeps the independent-release-cadence fact, all degradation prose, and the role table

### Removal Verification

- [ ] A-008 R5: `grep -rn 'brew install sahil87/tap/' README.md docs/site/` returns exactly one hit — the install.md error-hint example

### Scenario Coverage

- [ ] A-009 R5: third-party prerequisite blocks (`brew install yq jq gh direnv` at README:~95 + install.md:~50; `brew install go just` at README:~122 + install.md:~88) are unchanged; README `## Install` is unchanged

### Edge Cases & Error Handling

- [ ] A-010 R5: readme-extraction conformance — all replacement shll.ai links are absolute `https://` (rule 5), no structural changes to the README H1/blockquote/badges head or footer headings, no new relative links introduced

### Code Quality

- [ ] A-011 Pattern consistency: replacement prose mirrors README's already-conformant `## Install` section (wording and `sh` code-fence style)
- [ ] A-012 No unnecessary duplication: install.md does not restate README Quick-Start content; only the install pointer changed
- [ ] A-013 Markdown-only artifacts: all edits are standard CommonMark; no files outside README.md/docs/site/install.md touched (no Go, no `.claude/skills/`, no specs/memory)

## Notes

- Check items as you review: `- [ ]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [ ] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | README Prerequisites parenthetical uses the intake's given replacement text verbatim | Intake supplies the exact cell text ("e.g." form); pinning it avoids wording drift | S:90 R:90 A:95 D:90 |
| 2 | Confident | install.md CLI-section prose mirrors README `## Install` ("Installs fab-kit (plus the shll meta-CLI) via Homebrew, handling tap trust automatically") plus a canonical-reference shll.ai sentence, keeping the "installs two CLIs" → role-table flow | Intake names README's Install section as the model to mirror and lists the prose points; exact sentence flow is mine | S:80 R:85 A:85 D:80 |
| 3 | Confident | Replacement code fences use `sh` (not the section's previous `bash`) | Matches the mirrored README `## Install` fences and the intake's replacement snippets; purely presentational | S:70 R:90 A:85 D:80 |
| 4 | Confident | Companion lead-ins keep the independent-projects/release-cadence fact in-sentence while repointing install at `shll install` / https://shll.ai | Intake instructs exactly this adjustment for both files; sentence construction is mine | S:80 R:90 A:85 D:80 |

4 assumptions (1 certain, 3 confident, 0 tentative).
