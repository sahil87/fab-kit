# Plan: Retire the Codex→Claude Review Cascade

**Change**: 260808-rvcs-retire-codex-claude-review-cascade
**Intake**: `intake.md`

## Requirements

### Review Skill: Cascade Retirement

#### R1: `_review.md` carries no external-tool cascade
`src/kit/skills/_review.md` SHALL NOT contain a `### Codex→Claude Cascade` section, nor any procedure that shells out to a `codex` or `claude` CLI as part of the holistic-diff review. The dispatched review worker MUST be the sole holistic reviewer.

- **GIVEN** the review worker reads `_review.md` at review entry
- **WHEN** it executes the Shared Review Dispatch end-to-end
- **THEN** it performs the holistic-diff focus areas itself with full repository access
- **AND** it invokes no `command -v codex` / `command -v claude` probe and spawns no external reviewer subprocess

#### R2: `_review.md` frontmatter description states present truth
The `description:` frontmatter of `src/kit/skills/_review.md` SHALL describe the holistic-diff focus areas without the parenthetical "(Codex→Claude cascade with full repo access)". Full repo access remains stated as a property of the single worker.

- **GIVEN** a reader (or a skill-index consumer) reads `_review.md`'s frontmatter
- **WHEN** the description is rendered
- **THEN** it names the two checklists and the `mode` parameter, and attributes full repo access to the single review worker
- **AND** it mentions no external-tool cascade

#### R3: The zero-findings pass rule survives without the retired toggle
The Findings & Verdict zero-findings example in `src/kit/skills/_review.md` SHALL NOT reference `code-review.md` § Review Tools or tool availability. The rule itself — no must-fix findings (including zero findings) passes, so an empty `diff-only` result passes best-effort and adoption is never hard-blocked — MUST be preserved verbatim in meaning.

- **GIVEN** a `diff-only` review of an adopted change that surfaces no findings
- **WHEN** the verdict is computed
- **THEN** the review passes best-effort
- **AND** the prose justifying that outcome cites the reviewer's own judgment, not a disabled/unavailable external tool

#### R4: Review semantics are otherwise byte-unchanged in meaning
The single-worker dispatch model, `mode: full | diff-only` gating, Preconditions, the Plan-Conformance Steps (including the parsimony pass and deletion-candidate prompt), the Holistic-Diff Focus Areas, the three findings tiers, the deterministic pass/fail rule, and the `{stage}-result.yaml` review schema SHALL be unchanged.

- **GIVEN** the retirement is applied
- **WHEN** a `full`-mode review runs and produces one must-fix finding
- **THEN** the verdict is `fail`, exactly as before
- **AND** `/fab-adopt`'s `diff-only` path still skips Preconditions and the plan-conformance steps

### Mirror Class: Spec and Restatement Sweep

#### R5: The SPEC-`_review.md` mirror matches the retired skill
`docs/specs/skills/SPEC-_review.md` SHALL carry no Codex→Claude cascade in its Summary, its mode paragraph, its single-dispatch paragraph, its Flow diagram, its Tools-used table, or its Sub-agents note. The single-dispatch paragraph's reviewer-diversity claim SHALL be reworded: diversity is delegated to `agent.profiles.review` (pre-ship) and Copilot at review-pr (post-ship).

- **GIVEN** the constitution requires a skill edit to carry its SPEC mirror
- **WHEN** `grep -n 'Codex' docs/specs/skills/SPEC-_review.md` runs after apply
- **THEN** no cascade reference remains
- **AND** the Flow diagram's `Codex→Claude Cascade [both modes]` node is gone rather than renamed

#### R6: `fab-continue.md` and its SPEC restate the merged procedure without the cascade
`src/kit/skills/fab-continue.md`'s Review Behavior parenthetical SHALL list the merged procedure as plan-conformance steps + holistic-diff focus areas only. `docs/specs/skills/SPEC-fab-continue.md`'s Flow diagram and sub-agent table SHALL do the same.

- **GIVEN** a reader follows `/fab-continue`'s Review Behavior to `_review.md`
- **WHEN** the parenthetical enumerates what `_review.md` defines
- **THEN** it names two checklists, matching `_review.md`'s actual contents

#### R7: `skills.md`'s git-pr-review rollup drops the dead contrast
`docs/specs/skills.md` Phase-2 bullet SHALL NOT contrast against a Codex/Claude cascade that no longer exists anywhere. The Copilot-toggle clause (`code-review.md` § Review Tools, absent = enabled) SHALL be preserved.

- **GIVEN** the cascade is retired repo-wide
- **WHEN** a reader reaches the git-pr-review Phase 2 description
- **THEN** it states Copilot is the only automated reviewer without referencing a retired mechanism

#### R8: The mirror-class sweep is verified, not assumed
Before apply completes, a repo-wide grep for `cascade` / `Codex` SHALL be run over the living-doc class (`src/kit/`, `docs/specs/`, `docs/site/`, `README.md`), and every remaining hit SHALL be one of the known unrelated uses: the config four-layer cascade, the dispatch descent ladder, the status-transition cascade, `codex` as a provider name, or a frozen historical record (migrations, dated findings docs, `log.md`/`log.seed.md` entries, archived changes).

- **GIVEN** apply has edited the enumerated files
- **WHEN** the sweep grep runs
- **THEN** every surviving hit is classifiable into the unrelated set
- **AND** any unclassifiable hit is fixed before apply returns

### Scaffold: Review Tools Section

#### R9: The scaffolded `code-review.md` § Review Tools documents only `copilot`
`src/kit/scaffold/fab/project/code-review.md` § Review Tools SHALL remove the `codex / claude` explanation bullet and the `- codex: false` / `- claude: false` example lines. The section, its absent-means-enabled semantics, the `copilot` bullet, and the `- copilot: false` example SHALL remain (`/git-pr-review` Phase 2 reads the `copilot` entry; `--tool copilot` force-overrides it).

- **GIVEN** a new project scaffolded by `/fab-setup`
- **WHEN** the operator opens `fab/project/code-review.md` § Review Tools
- **THEN** the only documented toggle is `copilot`
- **AND** setting `- copilot: false` still skips the `/git-pr-review` Phase 2 request

### Non-Goals

- **No provider-resolved second-opinion knob** — explicitly deferred by the backlog entry ("do NOT build it in this change"); it is an optional follow-up only if a second opinion is missed later.
- **No Go changes and no Go test changes** — the sole Go-side mention (`src/go/fab/cmd/fab/config_test.go:723`, "review_tools (retired to code-review.md § Review Tools)") stays accurate because the section survives with the `copilot` entry.
- **No `_cli-fab.md` change** — no CLI surface is touched; its § Review Tools pointer still resolves to a live section.
- **No migration file** — after the deletion nothing reads `codex`/`claude` bullets, so stale entries in an existing user project's optional, user-owned `code-review.md` are inert prose.
- **No memory edits during apply** — `docs/memory/pipeline/execution-skills.md` and `docs/memory/_shared/configuration.md` are hydrate's targets (intake § What Changes 4), not apply's.
- **No edits to frozen historical records** — `src/kit/migrations/2.12.1-to-2.13.0.md`, `docs/specs/findings/skills-review-2026-06-1*.md`, `log.md`/`log.seed.md`, and archived change folders stay verbatim.

### Design Decisions

#### Cascade retired outright rather than reimplemented over the providers table
**Decision**: Delete the Codex→Claude external-tool cascade from the review skill; reviewer diversity is delegated to `agent.profiles.review.provider` (plus `FAB_AGENT_PROFILES` per session) pre-ship and to Copilot at review-pr post-ship.
**Why**: The cascade predates the provider system and duplicates a decision the provider system now owns first-class — it hardcodes a `command -v codex` shell-out, carries its own parallel toggle surface, and bypasses `fab resolve-agent`, the `providers:` grammar, per-role model/effort fills, and the dispatch-mode ladder. It also composes badly: an opus-at-high-effort worker babysits a ~10-minute codex subprocess poll, and `review: { provider: codex }` would spawn codex-inside-codex. Empirically (2026-08-09, fp02 review) the Codex arm only confirmed the Claude worker's top findings, discovered nothing new, and cost ~15 minutes of wall-clock.
**Rejected**: Porting the cascade onto the providers table as a "second opinion" knob run headless — deferred as an optional follow-up rather than built here, because no evidence yet shows a second opinion is missed once Copilot covers the post-ship cross-vendor read.
*Introduced by*: 260808-rvcs-retire-codex-claude-review-cascade

#### `copilot` entry survives the § Review Tools trim
**Decision**: Keep `code-review.md` § Review Tools with only its `copilot` entry instead of retiring the whole section.
**Why**: `/git-pr-review` Phase 2 reads the `copilot` toggle at review-pr, so the section has a live consumer independent of the cascade; retiring it would break that toggle and invalidate the `_cli-fab.md` pointer and the `2.12.1-to-2.13.0` migration's seeding target.
**Rejected**: Deleting § Review Tools entirely — it would strand `/git-pr-review`'s only opt-out and require a Go-side and migration sweep the backlog explicitly excluded.
*Introduced by*: 260808-rvcs-retire-codex-claude-review-cascade

### Deprecated Requirements

#### Codex→Claude external-tool cascade in the holistic-diff review
**Reason**: Superseded by the provider system — `agent.profiles.review.provider` gives reviewer choice a first-class home; the cascade is a second, inferior mechanism for the same decision, and empirically yields no marginal findings for ~15 minutes of wall-clock.
**Migration**: Pre-ship reviewer choice → `agent.profiles.review.provider` (per-session via `FAB_AGENT_PROFILES`). Post-ship cross-vendor eyes → Copilot at review-pr (`/git-pr-review`). Stale `- codex: false` / `- claude: false` bullets in an existing project's `code-review.md` are inert; no migration ships.

#### `codex` and `claude` entries in `code-review.md` § Review Tools
**Reason**: Their sole reader was the cascade's 5-step check-config procedure, which this change deletes.
**Migration**: N/A — the entries were opt-out toggles for a mechanism that no longer exists. The `copilot` entry in the same section is unaffected.

## Tasks

### Phase 1: Canonical Skill Sources

- [x] T001 Delete the `### Codex→Claude Cascade` section (the cascade description, the Review Tools gating prose, and the 5-step check-config/attempt-Codex/check-config/attempt-Claude/graceful-no-op procedure) from `src/kit/skills/_review.md`; leave `### Holistic-Diff Focus Areas` verbatim <!-- R1 -->
- [x] T002 Rewrite the `description:` frontmatter of `src/kit/skills/_review.md` to drop "(Codex→Claude cascade with full repo access)" while keeping full repo access attributed to the single worker <!-- R2 -->
- [x] T003 Reword the zero-findings example in `src/kit/skills/_review.md` § Findings & Verdict so it no longer cites `code-review.md` § Review Tools or tool availability, preserving the best-effort `diff-only` pass rule <!-- R3 -->
- [x] T004 [P] Drop "+ Codex→Claude cascade" from the merged-procedure parenthetical in `src/kit/skills/fab-continue.md` Review Behavior <!-- R6 -->

### Phase 2: Scaffold

- [x] T005 In `src/kit/scaffold/fab/project/code-review.md` § Review Tools, remove the `- codex / claude — the review-stage Codex → Claude cascade …` explanation bullet and the `- codex: false` / `- claude: false` example lines; keep the section, the absent-means-enabled semantics, the `copilot` bullet, and `- copilot: false` <!-- R9 -->

### Phase 3: SPEC Mirrors & Aggregate Specs

- [x] T006 Update `docs/specs/skills/SPEC-_review.md`: Summary paragraph, the mode paragraph's "identical for both modes" clause, the single-dispatch paragraph's reviewer-diversity claim (reword to `agent.profiles.review` + post-ship Copilot), the prose-packaging note's cascade mention, the Flow diagram's `Codex→Claude Cascade [both modes]` node, the Bash row in Tools used, and the Sub-agents note <!-- R5 -->
- [x] T007 [P] Update `docs/specs/skills/SPEC-fab-continue.md`: the Flow diagram's cascade line and the sub-agent table's "holistic full-repo diff review via Codex→Claude cascade" cell <!-- R6 -->
- [x] T008 [P] Simplify the git-pr-review Phase 2 bullet in `docs/specs/skills.md` so it no longer contrasts against the retired cascade, preserving the Copilot-toggle clause <!-- R7 -->

### Phase 4: Sweep

- [x] T009 Run the repo-wide `cascade`/`Codex` sweep over the living-doc class (`src/kit/`, `docs/specs/`, `docs/site/`, `README.md`), classify every surviving hit against the unrelated set (config four-layer cascade, dispatch descent ladder, status-transition cascade, `codex` as a provider name, frozen historical records), and fix any unclassifiable hit <!-- R8 -->
- [x] T010 Verify the non-goals hold: `git diff --name-only` contains no `.go` file, no `src/kit/migrations/` file, no `docs/memory/` file, and no `src/kit/skills/_cli-fab.md` <!-- R4 -->

## Execution Order

- T001–T003 touch the same file; run sequentially in that order.
- T006 must follow T001–T003 (the SPEC mirrors the final skill text); T007 must follow T004.
- T009 must run last among content edits; T010 is the final gate.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/kit/skills/_review.md` contains no `Codex→Claude Cascade` heading and no `command -v codex` / `command -v claude` probe
- [x] A-002 R2: `_review.md`'s `description:` frontmatter names the two checklists and the `mode` parameter with no cascade parenthetical
- [x] A-003 R3: `_review.md` § Findings & Verdict states the zero-findings/best-effort rule without referencing § Review Tools or tool availability
- [x] A-004 R5: `docs/specs/skills/SPEC-_review.md` has zero `Codex` occurrences and its single-dispatch paragraph attributes diversity to `agent.profiles.review` + post-ship Copilot — *the six live claim sites named by R5 (Summary, mode paragraph, single-dispatch paragraph, Flow diagram, Tools-used table, Sub-agents note) are clean; the only surviving `Codex` strings are inside the new dated `**Cascade retirement** (260808-rvcs)` lineage note, which names the deleted section as a record of the retirement (same convention as the skop/s2sz/pag2 notes above it)*
- [x] A-005 R6: `src/kit/skills/fab-continue.md` and `docs/specs/skills/SPEC-fab-continue.md` describe the merged procedure as two checklists
- [x] A-006 R7: `docs/specs/skills.md`'s Phase 2 bullet states Copilot is the only automated reviewer without a cascade contrast, and retains the § Review Tools toggle clause
- [x] A-007 R9: `src/kit/scaffold/fab/project/code-review.md` § Review Tools documents `copilot` only, with the section and its absent-means-enabled semantics intact

### Behavioral Correctness

- [x] A-008 R1: The review worker is described everywhere as performing the holistic-diff review itself with full repository access — no delegation to an external CLI
- [x] A-009 R4: `mode: full | diff-only` gating, Preconditions, the seven plan-conformance steps, the three findings tiers, the "any must-fix → fail" rule, and the `{stage}-result.yaml` review schema are unchanged
- [x] A-010 R9: The `copilot` toggle path (`/git-pr-review` Phase 2 reads it; `--tool copilot` force-overrides) still resolves to a live scaffolded section

### Removal Verification

- [x] A-011 R1: The 5-step cascade procedure, its Review Tools gating prose, and the `- codex: false` example are gone from `_review.md` with no dead pointer left behind
- [x] A-012 R5: The `└─ Codex→Claude Cascade [both modes]` node is removed from SPEC-`_review.md`'s Flow diagram rather than renamed, and the diagram still parses as a coherent tree
- [x] A-013 R9: No `codex` or `claude` reviewer-toggle entry survives in the scaffold template

### Scenario Coverage

- [x] A-014 R8: A repo-wide `cascade`/`Codex` grep over `src/kit/`, `docs/specs/`, `docs/site/`, and `README.md` returns only classifiable unrelated hits
- [x] A-015 R3: The `/fab-adopt` `diff-only` zero-findings path is still documented as passing best-effort in both `_review.md` and its SPEC

### Edge Cases & Error Handling

- [x] A-016 R8: Frozen historical records (`src/kit/migrations/2.12.1-to-2.13.0.md`, `docs/specs/findings/skills-review-2026-06-1*.md`, `log.md`/`log.seed.md`, archived change folders) are untouched
- [x] A-017 R4: No `.go`, `src/kit/migrations/`, `docs/memory/`, or `_cli-fab.md` file appears in the apply-stage diff

### Code Quality

- [x] A-018 Pattern consistency: Edited prose matches surrounding skill/SPEC voice, heading structure, and diagram conventions
- [x] A-019 No unnecessary duplication: No rule is restated in a file that does not own it — pointers stay pointers (code-quality.md § "Stating an owned rule AND pointing at its owner")
- [x] A-020 Canonical source only: No file under `.claude/skills/` is edited (code-quality.md § Anti-Patterns, Constitution V)
- [x] A-021 SPEC-mirror sync: Every edited `src/kit/skills/*.md` carries its `docs/specs/skills/SPEC-*.md` update in this change (Constitution Additional Constraints; code-quality.md § Sibling & Mirror Sweeps)
- [x] A-022 Markdown-only artifacts: All edits are standard CommonMark; no binary or generated formats introduced

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Hydrate targets (out of apply scope, intake § What Changes 4): `docs/memory/pipeline/execution-skills.md` cascade paragraphs + a superseding design-decision entry, and `docs/memory/_shared/configuration.md` § `code-review.md` Review Tools bullet → copilot-only.

## Deletion Candidates

- `docs/memory/pipeline/execution-skills.md` (cascade paragraphs at the review-stage and git-pr-review Phase 2 sections, plus the cascade-leaning Why/Rejected text in the Copilot-only Phase 2 and Single Review Agent design decisions) — describes a mechanism this change deletes; already claimed by hydrate (plan § Notes), listed here so the redundancy is on the record
- `docs/memory/_shared/configuration.md` § `code-review.md` — the `codex`/`claude` half of the § Review Tools bullet is now dead prose; hydrate's declared target
- `src/kit/migrations/2.12.1-to-2.13.0.md:19,133` — the seed text still describes `codex`/`claude` as "the review-stage outward Codex→Claude cascade" and seeds those bullets into projects upgrading from 2.12.1. Deliberately frozen per intake Assumption #5 (migrations are historical records); surfaced for the human only — no action taken
- `src/kit/skills/git-pr-review.md:101` "Only the `copilot` entry is honored here." — its contrast partners (`codex`/`claude`) no longer exist in the kit, so the clause now only tells readers that stale entries in an existing project's `code-review.md` are inert. Still useful under Assumption #6 (no migration strips them); trim only if that reading is not wanted

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `docs/memory/` edits are hydrate's, not apply's — the plan's tasks stop at skills/scaffold/specs | Intake § What Changes segments 1–3 as the change surface and labels 4 "Hydrate"; the pipeline gives hydrate its own stage | S:95 R:90 A:95 D:95 |
| 2 | Certain | `docs/specs/skills/SPEC-git-pr-review.md`, `src/kit/skills/git-pr-review.md`, `docs/specs/architecture.md`, and `docs/specs/glossary.md` need no edit | Their § Review Tools references are all copilot-scoped or toggle-location pointers; the section and the `copilot` entry both survive | S:90 R:85 A:90 D:85 |
| 3 | Certain | `_pipeline.md`, `fab-adopt.md`, `user-flow.md`, `templates.md`, and the SPEC mirrors' "cascade" hits are the status-transition cascade and stay | Verified during gap analysis and re-verified by grep at plan time — they describe `reset`/`skip` downstream cascading, unrelated to review tooling | S:95 R:90 A:95 D:95 |
| 4 | Certain | `README.md`, `docs/site/workflows.md`, and `docs/specs/harness-adapters.md` "Codex" hits stay | They name Codex as a supported harness/provider — the provider system is explicitly the surviving mechanism | S:95 R:90 A:95 D:95 |
| 5 | Confident | SPEC-`_review.md`'s dated prose-packaging note (260808-s2sz) is edited in place to drop "cascade behavior" rather than left verbatim | It is a living SPEC paragraph making a present-tense claim about what is unchanged, not a frozen record like a migration or a dated findings doc — leaving it would assert the cascade still exists | S:70 R:85 A:80 D:75 |
| 6 | Confident | No Go tests are run: the change touches zero `.go` files | code-quality.md § Test Strategy scopes test runs to `.go` changes; the intake forbids Go edits and T010 verifies the diff | S:80 R:85 A:85 D:80 |

6 assumptions (4 certain, 2 confident, 0 tentative).
