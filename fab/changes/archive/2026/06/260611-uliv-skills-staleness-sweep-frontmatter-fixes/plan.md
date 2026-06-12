# Plan: Skills Staleness Sweep + Frontmatter Description Accuracy (Skills-Review Batch 2/4)

**Change**: 260611-uliv-skills-staleness-sweep-frontmatter-fixes
**Intake**: `intake.md`

## Requirements

> Finding IDs (`f###`, `g#-#`) reference `docs/specs/findings/skills-review-2026-06-11.md`.
> Global constraints: all edits in `src/kit/` (never `.claude/skills/`); templates in
> `src/kit/templates/`; every skill edit updates its `docs/specs/skills/SPEC-*.md` mirror;
> no Go code changes; no behavior changes except R7 (D), R20 (J), R24 (M.2).

### Skills: Stale Command Names (A — f030)

#### R1: Pre-Go-CLI command names removed
All occurrences of `statusman`, `changeman`, and `logman` in `src/kit/skills/*.md` SHALL be replaced with the current Go CLI command families (`fab status`, `fab change resolve`, `fab log`), with each replacement's argument form matching the current CLI signature. `fab-archive.md`'s restore-mode references to "the script" SHALL become "the command". `git-pr.md` Step 4c's two sentences SHALL be rewritten to name the real commands (`fab change resolve` succeeded / `fab status add-pr` ran).

- **GIVEN** the 9 affected skill files (fab-new, fab-draft, fab-fff, fab-discuss, fab-setup, git-pr, git-pr-review, git-branch, plus fab-archive's "the script")
- **WHEN** `grep -rn 'statusman\|changeman\|logman' src/kit/` runs after the sweep
- **THEN** it returns zero hits, and every rewritten sentence names a command that exists in the Go CLI

### Operator: State-File Terminology (B — f114; M.3)

#### R2: "Operator state file" defined once
`fab-operator.md` SHALL define the term **operator state file** once in §4 (the section currently headed `### .fab-operator.yaml`) with the real server-keyed path (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`, fallback `~/.local/state/...`, keyed by tmux socket path, derived by the binary), and SHALL refer to the live file by that defined term everywhere else (lines ~23, 29, 33, 277, 344, 551, 565, 638, 645). Mentions of the **legacy repo-rooted** `.fab-operator.yaml` in not-read/not-migrated context (lines ~65, ~121) are deliberate and SHALL be kept.

- **GIVEN** the reworded skill
- **WHEN** grepping `fab-operator.md` for `.fab-operator.yaml`
- **THEN** the only remaining hits describe the abandoned legacy repo-rooted file, and tick step 6 says "write updated state to the operator state file"

#### R3: Tick summary dedupe matches §7
The §4 tick-loop watch step ("compare against `known`") SHALL be aligned with §7 step 2's corrected rule: dedupe against `known` **plus** `completed`.

- **GIVEN** §7 step 2 already dedupes against known + completed
- **WHEN** the §4 tick step 3 line is read
- **THEN** it says "compare against `known` + `completed`" (or equivalent), with no contradiction between the two sections

### Templates: Spec-Stage Residue + Dead Fields (C — f062+g3-1, g3-2, g3-5)

#### R4: Intake template uses plan/apply-entry vocabulary
`src/kit/templates/intake.md` SHALL contain no spec-stage vocabulary: "primary input for spec generation" → "primary input for plan generation (apply entry)"; "Helps scope the spec" → "Helps scope the plan"; "SRAD handles prioritization at spec generation time" → "at plan generation (apply entry)"; "between the intake-stage agent and the spec-stage agent" → "...and the apply-entry agent (which co-generates plan.md)".

- **GIVEN** the regenerated template
- **WHEN** grepping it for `spec generation|spec-stage|scope the spec`
- **THEN** zero hits remain, and the wording matches `_generation.md`'s canonical phrasing

#### R5: SRAD grade scope corrected (three at apply, four at intake)
`src/kit/templates/plan.md`'s `## Assumptions` comment SHALL say **three grades only** (Certain/Confident/Tentative — Unresolved is intake-only; apply decides and records). `_preamble.md`'s Assumptions Summary Block rule "Include all four grades" SHALL be scoped to **intake artifacts**, stating that plan.md `## Assumptions` excludes Unresolved. The intake template keeps all four grades.

- **GIVEN** an apply-entry agent generating plan.md `## Assumptions`
- **WHEN** it reads the template comment and `_preamble.md`'s rules
- **THEN** both consistently allow only Certain/Confident/Tentative in plan.md while intake artifacts record all four

#### R6: Dead `**Status**:` header lines deleted
The `**Status**: Draft` line in `src/kit/templates/intake.md` and `**Status**: In Progress` line in `src/kit/templates/plan.md` SHALL be deleted (no skill reads or flips them; `.status.yaml` is the state of record).

- **GIVEN** the two templates
- **WHEN** grepping `src/kit/templates/` for `^\*\*Status\*\*:`
- **THEN** zero hits remain

### Generation: Acceptance R# Exemption (D — f061+g3-3) — *behavior-flagged*

#### R7: Non-requirement-derived acceptance categories exempt from R#
`_generation.md` (step 4 traceability bullet + step 6) and `src/kit/templates/plan.md` (header TRACEABILITY comment + `## Acceptance` section comment) SHALL scope the "each `## Acceptance` item MUST name the requirement it accepts" rule to requirement-derived categories (Functional Completeness, Behavioral Correctness, Removal Verification, Scenario Coverage, Edge Cases & Error Handling, Security). Code Quality and `checklist.extra_categories` items SHALL use `A-{NNN}: {outcome}` (optionally `A-{NNN} {label}: {outcome}`) without an R# reference — matching the template's own A-007/A-008 shape.

- **GIVEN** a plan with Code Quality baseline items and `checklist.extra_categories` items
- **WHEN** the generation procedure is followed literally
- **THEN** those items are valid without inventing a fake R#, while requirement-derived items still carry the mandatory `A-{NNN} R#:` form

### Preamble + CLI Reference Staleness (E — f050, f054, f036, f038, f052, f053)

#### R8: Autonomy table fab-continue column updated to post-1.10.0
`_preamble.md` Skill-Specific Autonomy Levels: the fab-continue column SHALL read — Posture: SRAD at intake only (apply decides-and-records); Interruption budget: 1-2 at intake, 0 at apply and later; Output: drop the "[NEEDS CLARIFICATION] count".

- **GIVEN** `_generation.md` bans [NEEDS CLARIFICATION] markers at plan generation and fab-continue.md:64 budgets 1-2 questions at intake only
- **WHEN** the autonomy table is read
- **THEN** the fab-continue column no longer contradicts either

#### R9: Duplicate gate rows collapsed
`_cli-fab.md`'s `fab score` modes table SHALL present a single Gate row (`fab score --check-gate [--stage intake] <change>` — `--stage` defaults to `intake`; flat 3.0 threshold, the single gate), removing the spec-stage-residue "Gate" vs "Intake gate" split. The Normal row's unique details are kept.

- **GIVEN** score.go has one gate and `--stage` defaults to intake
- **WHEN** the table is read
- **THEN** exactly one gate row exists and no second gate is implied

#### R10: `fab status fail` documented as review/review-pr only
"(review only)" SHALL become "(review/review-pr only)" at `_preamble.md` § Common fab Commands, `_cli-fab.md`'s `fail` row, and `fab-continue.md`'s Step 4 event list (same stale string, file already in scope).

- **GIVEN** status.go permits fail on review and review-pr
- **WHEN** an agent cross-checks `fab status fail <change> review-pr` against the helper docs
- **THEN** the docs confirm the call is valid

#### R11: Preflight field lists complete
The preflight output field lists SHALL include `id`, `display_stage`, and `display_state`: `_preamble.md` §2 step 3 (add display_stage/display_state — id already present), `_preamble.md` § Common fab Commands preflight row, and `_cli-fab.md` `fab preflight (extended)` (add all three).

- **GIVEN** preflight.go FormatYAML emits nine fields
- **WHEN** the three documented field lists are read
- **THEN** all nine fields are declared

#### R12: Common Error Messages table regenerated from resolve.go
`_cli-fab.md`'s Common Error Messages table SHALL be regenerated from the actual strings in `internal/resolve/resolve.go`: `No change matches "{arg}".`, `Multiple changes match "{arg}": {list}.`, `No active changes found.` (override given, zero change folders), `No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.` (no override, symlink absent, 0 candidates), `No active change (multiple changes exist — use /fab-switch).`, `fab/changes/ not found.` — with accurate Cause/Fix columns. Strings that exist nowhere in the Go source (`Status file not found: {path}`, `Cannot resolve change '{arg}'`) SHALL be removed.

- **GIVEN** the regenerated table
- **WHEN** each Error cell is grepped against `src/go/fab/internal/resolve/resolve.go`
- **THEN** every documented string matches a real error verbatim (modulo `%s`→`{arg}` placeholders)

#### R13: Six missing `fab status` query subcommands documented
`_cli-fab.md`'s "Full subcommand table" SHALL gain rows for `all-stages`, `progress-map <change>`, `display-stage <change>`, `plan <change>`, `confidence <change>`, and `validate-status-file <change>` (all visible in status.go), keeping the "Full" charter accurate.

- **GIVEN** status.go registers these six non-hidden subcommands
- **WHEN** the table is read
- **THEN** every visible subcommand has a row

### Generation: Consumer List (F — f060+g4-8)

#### R14: `_generation.md` names all five consumers
The frontmatter description and intro blockquote SHALL name all five consumers with their procedure split: fab-new/fab-draft (Intake Generation Procedure) and fab-continue/fab-ff/fab-fff (Plan Generation Procedure).

- **GIVEN** five skills declare `helpers: [_generation]`
- **WHEN** `_generation.md` lines 3 and 11 are read
- **THEN** both name all five consumers and which procedure each uses

### Internal Skills (G — f029, f027)

#### R15: internal-retrospect exemplars name real commands
The `/meta:scriptify` and `/meta:review` exemplars SHALL be replaced: the review case points to `/internal-skill-optimize {skill-file}`; the scriptify bullet is dropped (no real mechanism exists).

- **GIVEN** the Suggested Actions exemplar list
- **WHEN** grepping the repo's `src/kit/` for `/meta:`
- **THEN** zero hits remain and every exemplar command exists

#### R16: internal-skill-optimize excludes all `_*.md` partials
The batch-mode exclusion (Arguments + Constraints) SHALL cover **all** `_*.md` partials (`_preamble`, `_generation`, `_review`, `_cli-fab`, `_cli-external`), and Pre-flight SHALL read the partials as reference context, not targets.

- **GIVEN** a batch run over `src/kit/skills/`
- **WHEN** targets are enumerated
- **THEN** no `_*.md` file is a rewrite target

### Specs: SPEC Coverage + Naming (H — f048)

#### R17: Four new SPEC files created
`docs/specs/skills/` SHALL gain `SPEC-internal-consistency-check.md`, `SPEC-internal-retrospect.md`, `SPEC-internal-skill-optimize.md`, and `SPEC-_generation.md`, following the existing house style (Summary + Flow + supporting tables, cross-referencing the skill source as canonical).

- **GIVEN** the hybrid coverage decision (intake assumption #9)
- **WHEN** `ls docs/specs/skills/` runs
- **THEN** the four files exist and accurately summarize their skill sources

#### R18: Exclusion policy documented + underscore convention normalized
An explicit SPEC exclusion policy for the pure-reference partials (`_cli-fab`, `_cli-external`) SHALL be documented in `docs/specs/skills.md` (their content mirrors the CLI — the constitution's CLI rule already forces `_cli-fab.md` updates; a SPEC would be a third copy). The SPEC partial naming convention SHALL keep the leading underscore (`SPEC-_review.md` form): `SPEC-preamble.md` is renamed to `SPEC-_preamble.md`, and the live reference in `docs/memory/_shared/context-loading.md` (§ prose, line ~72) is updated; historical changelog rows are left untouched.

- **GIVEN** the rename
- **WHEN** grepping live docs for `SPEC-preamble.md`
- **THEN** only historical changelog/archive rows reference the old name, and the policy text names which files are excluded and why

### Docs: CONTRIBUTING Modernization (I — f126)

#### R19: CONTRIBUTING.md teaches the current architecture
The "Stage Manager" section SHALL be rewritten for the Go CLI (`fab status`, `fab preflight`) and the 6-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`); `fab/.kit` reading-path descriptions SHALL be updated to the system-cache model (`~/.fab-kit/versions/<version>/kit/`).

- **GIVEN** the rewritten file
- **WHEN** grepping it for `stageman|fab/.kit|get_stage_number`
- **THEN** zero hits remain and every documented command exists in the current CLI

### Planning Skills: Change-Type Hook Ownership (J — g3-4) — *behavior-flagged*

#### R20: Hook owns change_type; skills verify and override only if wrong
`fab-new.md` and `fab-draft.md` Step 6 SHALL drop the manual keyword-matching + unconditional `set-change-type` write. Replacement contract: the PostToolUse intake-write hook infers and writes `change_type` on every `intake.md` write (word-boundary regexes incl. `redesign`, first match wins, default `feat` — artifact.go); the skill verifies the hook's value by reading `change_type` from the change's `.status.yaml` (preflight does not emit it), and overrides via `fab status set-change-type` **only if wrong**, noting that any later intake write re-fires the hook and overwrites a manual value.

- **GIVEN** an intake.md write fires the hook
- **WHEN** Step 6 is followed
- **THEN** no unconditional `set-change-type` runs, the double-write race is gone, and the wrong-type recovery path is documented

### Skills: Frontmatter Descriptions (K — g4-2, g4-6, g4-9, g4-7, g4-4, g4-10, g4-5)

#### R21: Descriptions match actual behavior
Frontmatter `description:` one-liners SHALL be corrected: `fab-continue.md` (canonical six stage names incl. ship/review-pr), `fab-proceed.md` (takes no arguments; `/fab-fff <change>` for targeting), `git-pr.md` (creates a **draft** PR), `fab-switch.md` (`--none` deactivates), `docs-reorg-memory.md` (also the memory rebalancer — shape diagnosis, split/merge/flatten), `git-branch.md` (unmatched explicit names fall back to a literal standalone branch). g4-5 (`fab-fff.md` body text on git-pr-review's no-reviews behavior) is verified already fixed by batch 1 — no edit unless residue is found.

- **GIVEN** an agent selecting skills by description
- **WHEN** it reads the six updated one-liners
- **THEN** each names the previously hidden behavior, and `fab-fff.md` already names the Copilot request + 10-minute poll

### Specs: New Skill Checklist (L — f125)

#### R22: New Skill Checklist added to skills.md
`docs/specs/skills.md` SHALL gain a "New Skill Checklist" section enumerating: frontmatter fields (name/description, optional `user-invocable`/`disable-model-invocation`/`metadata.internal`), the preamble-read line, `helpers:` declaration, `Next:` line convention, Error Handling + Key Properties tables, the SPEC mirror file (with the R18 exclusion policy), the skills.md mapping row, and the fabhelp.go `skillToGroupMap` help grouping.

- **GIVEN** a kit developer adding a new skill
- **WHEN** they follow the checklist
- **THEN** all eight integration points are covered in one place

### Batch-1 Residuals (M)

#### R23: git-pr-review exit-point claim names its exceptions (M.1 + M.5)
The "single exit point" sentence in `git-pr-review.md` SHALL be reworded to name the two direct-STOP exceptions (Step 1.5 invalid `--tool`; Step 5 commit/push failure after `git reset`). The `docs/specs/skills.md` git-pr-review rollup SHALL be updated to match actual behavior: `--tool` accepts only `copilot`; no Codex/Claude cascade (Copilot request + 30s×20 poll); terminal outcomes route through Step 6 (success/no-reviews → finish, failure → fail, timeout → leave active) with the two named STOP exceptions.

- **GIVEN** the skill body's Steps 1.5 and 5
- **WHEN** the single-exit sentence and the skills.md rollup are read
- **THEN** neither overclaims — both name the exceptions and the real cascade-less Copilot flow

#### R24: fab-continue review-pr Fail branch guarded (M.2) — *behavior-flagged*
The `review-pr` dispatch row's Fail branch in `fab-continue.md` SHALL carry the same only-if-still-active guard its Pass branch got in batch 1: git-pr-review's Step 6 runs its own `fail` transition, so fab-continue runs `fail <change> review-pr` only if the stage is still `active` after the behavior returns.

- **GIVEN** git-pr-review already failed the stage internally
- **WHEN** fab-continue's review-pr row processes a Fail outcome
- **THEN** no second (CLI-rejected) `fail` call is mandated

#### R25: Issue-ID collision grep anchored (M.4)
The Linear-ID collision check in `fab-new.md` and `fab-draft.md` SHALL use a word-boundary grep (`grep -lw "{ISSUE_ID}" fab/changes/*/.status.yaml`) so `DEV-123` no longer matches `DEV-1234`.

- **GIVEN** changes exist for DEV-1234 but not DEV-123
- **WHEN** the collision check runs for DEV-123
- **THEN** no false collision is reported (word-boundary semantics: the char after the match must be a non-word char)

#### R26: fab-setup "first run only" claim fixed (M.6)
`docs/specs/skills.md`'s fab-setup **Creates** block SHALL drop the "first run only" claim in favor of idempotent wording (re-runs skip whatever already exists).

- **GIVEN** setup is re-runnable
- **WHEN** the Creates block is read
- **THEN** it no longer claims setup runs once

### Cross-Cutting: SPEC Mirror Discipline

#### R27: Every edited skill's SPEC mirror updated
For every `src/kit/skills/*.md` edit above, the corresponding `docs/specs/skills/SPEC-*.md` SHALL be checked and updated where it mirrors the changed text — known targets: SPEC-fab-operator.md (state-file term at line ~94), SPEC-git-pr-review.md (single-exit line ~72), SPEC-fab-new.md/SPEC-fab-draft.md (summary "infers change type", flow `set-change-type` step, collision grep), SPEC-fab-continue.md (summary stage list + review-pr guard), SPEC-git-pr.md (draft), SPEC-(_)preamble.md (autonomy/fields if mirrored). Template edits update `docs/specs/templates.md` (Status lines, spec-stage vocabulary, acceptance R# mandate, plus its `lib/statusman.sh` line — same staleness class).

- **GIVEN** the constitution's mirror rule
- **WHEN** each SPEC mirror is grepped for the stale text its skill shed
- **THEN** no mirror still asserts the pre-change wording

### Non-Goals

- Renaming `statusman`/`changeman`/`logman` in non-skill specs (`docs/specs/naming.md`, `architecture.md`, `glossary.md`, `user-flow.md`, `change-types.md`) — human-curated design docs outside this batch's skill+mirror scope; noted for a future sweep
- `docs/specs/srad.md` autonomy-table residue — same reason
- Any Go code change (`resolve.go`, `artifact.go`, `status.go`, `fabhelp.go` are read-only references)
- Memory file updates (`pipeline/*`, `runtime/operator`, `memory-docs/*`) — hydrate-stage work, except the mechanical link fix in `_shared/context-loading.md` required by the R18 rename
- fab-help grouping gaps and other f125 drift beyond documenting the checklist

## Tasks

### Phase 1: Skill-file edits (src/kit/skills/)

- [x] T001 Rename `statusman`→`fab status`, `changeman`→`fab change resolve`, `logman`→`fab log` across `src/kit/skills/{fab-new,fab-draft,fab-fff,fab-discuss,fab-setup,git-pr,git-pr-review,git-branch}.md`; rewrite `git-pr.md` Step 4c sentences; "the script"→"the command" in `src/kit/skills/fab-archive.md` <!-- R1 -->
- [x] T002 `src/kit/skills/fab-operator.md`: retitle §4 state-file heading, define "operator state file" once, replace the ~9 live-file `.fab-operator.yaml` mentions with the term (keep legacy-migration mentions) <!-- R2 -->
- [x] T003 `src/kit/skills/fab-operator.md` §4 tick step 3: dedupe summary "against `known` + `completed`" <!-- R3 -->
- [x] T004 `src/kit/skills/_preamble.md`: autonomy-table fab-continue column (posture/budget/output) <!-- R8 -->
- [x] T005 "(review only)"→"(review/review-pr only)" in `src/kit/skills/_preamble.md` (Common fab Commands), `src/kit/skills/_cli-fab.md` (fail row), `src/kit/skills/fab-continue.md` (Step 4 list) <!-- R10 -->
- [x] T006 Preflight field lists: `src/kit/skills/_preamble.md` §2 step 3 + Common-Commands row; `src/kit/skills/_cli-fab.md` preflight (extended) <!-- R11 -->
- [x] T007 `src/kit/skills/_preamble.md` Assumptions Summary Block: scope four-grades rule to intake artifacts; plan.md excludes Unresolved <!-- R5 -->
- [x] T008 `src/kit/skills/_cli-fab.md`: collapse Gate/Intake-gate rows to one <!-- R9 -->
- [x] T009 `src/kit/skills/_cli-fab.md`: regenerate Common Error Messages table from `src/go/fab/internal/resolve/resolve.go` <!-- R12 -->
- [x] T010 `src/kit/skills/_cli-fab.md`: add 6 query-subcommand rows to the fab status table <!-- R13 -->
- [x] T011 `src/kit/skills/_generation.md`: consumer list (description + intro) names all 5 with procedure split <!-- R14 -->
- [x] T012 `src/kit/skills/_generation.md`: step 4 + step 6 acceptance R# exemption for Code Quality / extra categories <!-- R7 -->
- [x] T013 `src/kit/skills/internal-retrospect.md`: replace `/meta:*` exemplars with `/internal-skill-optimize` <!-- R15 -->
- [x] T014 `src/kit/skills/internal-skill-optimize.md`: exclude all `_*.md` partials (Arguments, Pre-flight, Constraints) <!-- R16 -->
- [x] T015 `src/kit/skills/fab-new.md` + `src/kit/skills/fab-draft.md` Step 6: hook-owned change_type (verify via `.status.yaml`, override only if wrong) <!-- R20 -->
- [x] T016 `src/kit/skills/fab-new.md` + `src/kit/skills/fab-draft.md`: word-anchor the Linear-ID collision grep (`grep -lw`) <!-- R25 -->
- [x] T017 Frontmatter descriptions: `fab-continue.md`, `fab-proceed.md`, `git-pr.md`, `fab-switch.md`, `docs-reorg-memory.md`, `git-branch.md`; verify g4-5 already fixed in `fab-fff.md` <!-- R21 -->
- [x] T018 `src/kit/skills/git-pr-review.md`: reword single-exit sentence to name Step 1.5 + Step 5 exceptions <!-- R23 -->
- [x] T019 `src/kit/skills/fab-continue.md`: add only-if-still-active guard to review-pr Fail branch <!-- R24 -->

### Phase 2: Templates (src/kit/templates/)

- [x] T020 `src/kit/templates/intake.md`: rewrite 4 spec-stage comments; delete `**Status**: Draft` line <!-- R4 --> <!-- R6 -->
- [x] T021 `src/kit/templates/plan.md`: three-grades comment; delete `**Status**: In Progress` line; scope TRACEABILITY/ACCEPTANCE-FORMAT comments + `## Acceptance` section comment per the R# exemption <!-- R5 --> <!-- R6 --> <!-- R7 -->

### Phase 3: Specs & docs

- [x] T022 SPEC mirror sweep: `docs/specs/skills/SPEC-fab-operator.md` (~:94), `SPEC-git-pr-review.md` (~:72), `SPEC-fab-new.md` + `SPEC-fab-draft.md` (summary, flow step, collision grep), `SPEC-fab-continue.md` (summary + review-pr guard), `SPEC-git-pr.md` (draft), `SPEC-preamble.md` content (autonomy/fields if mirrored); verify remaining mirrors of edited skills have no stale text <!-- R27 -->
- [x] T023 `docs/specs/templates.md`: Status lines (×3), spec-stage vocabulary, acceptance R# mandate scoping, `lib/statusman.sh` line <!-- R27 -->
- [x] T024 `git mv docs/specs/skills/SPEC-preamble.md docs/specs/skills/SPEC-_preamble.md`; update live ref in `docs/memory/_shared/context-loading.md` (~:72) <!-- R18 -->
- [x] T025 [P] Create `docs/specs/skills/SPEC-internal-consistency-check.md` <!-- R17 -->
- [x] T026 [P] Create `docs/specs/skills/SPEC-internal-retrospect.md` <!-- R17 -->
- [x] T027 [P] Create `docs/specs/skills/SPEC-internal-skill-optimize.md` <!-- R17 -->
- [x] T028 [P] Create `docs/specs/skills/SPEC-_generation.md` <!-- R17 -->
- [x] T029 `docs/specs/skills.md`: add "New Skill Checklist" section incl. SPEC-mirror item with `_cli-*` exclusion policy + underscore convention <!-- R22 --> <!-- R18 -->
- [x] T030 `docs/specs/skills.md`: git-pr-review rollup (M.5: `--tool copilot` only, no cascade, Step-6 routing + exceptions), fab-setup Creates wording (M.6), fab-continue purpose stage list (g4-2 mirror) <!-- R23 --> <!-- R26 --> <!-- R21 -->
- [x] T031 `CONTRIBUTING.md`: rewrite Stage Manager section for Go CLI + 6-stage pipeline; fix `.kit/`-era reading-path descriptions <!-- R19 -->

### Phase 4: Verification

- [x] T032 Residual sweep (`grep -rn 'statusman\|changeman\|logman' src/kit/`, `grep -rn '\.fab-operator\.yaml' src/kit/` legacy-only, `grep -rn '/meta:' src/kit/`, `grep -rn 'spec generation\|spec-stage' src/kit/templates/`); `go test ./...`; confirm no `.claude/skills/` file modified <!-- R1 --> <!-- R2 --> <!-- R15 --> <!-- R4 -->

## Execution Order

- T001–T021 are independent of each other (different files or different regions); T022/T023 (mirrors) run after their skill/template edits; T024–T031 are independent; T032 last.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `grep -rn 'statusman\|changeman\|logman' src/kit/` returns zero hits; rewritten sentences name real CLI commands; fab-archive says "the command"
- [x] A-002 R2: fab-operator.md defines "operator state file" once with the server-keyed path; live-file references use the term; legacy mentions remain only in not-migrated context
- [x] A-003 R3: §4 tick watch step dedupes against `known` + `completed`, matching §7 step 2
- [x] A-004 R4: intake template has zero spec-stage vocabulary; phrasing matches `_generation.md`
- [x] A-005 R5: plan template says three grades (Unresolved intake-only); `_preamble.md` scopes the four-grades rule to intake artifacts
- [x] A-006 R6: no `**Status**:` line in either template
- [x] A-007 R7: `_generation.md` + plan template scope the R# mandate to requirement-derived categories; Code Quality/extra categories documented as `A-{NNN}: {outcome}` without R#
- [x] A-008 R8: autonomy table fab-continue column shows intake-only questions (1-2; 0 at apply+) and no [NEEDS CLARIFICATION] output
- [x] A-009 R9: exactly one gate row in the fab score modes table
- [x] A-010 R10: all three `fail` doc sites say "(review/review-pr only)"
- [x] A-011 R11: the three preflight field lists declare id/display_stage/display_state (nine fields total)
- [x] A-012 R12: every error string in the regenerated table greps verbatim (placeholder-adjusted) in resolve.go; removed strings absent from Go source
- [x] A-013 R13: the fab status table includes all-stages, progress-map, display-stage, plan, confidence, validate-status-file
- [x] A-014 R14: `_generation.md` description + intro name all five consumers with the Intake/Plan procedure split
- [x] A-015 R15: zero `/meta:` references in src/kit/; exemplar commands exist
- [x] A-016 R16: internal-skill-optimize excludes all `_*.md` partials in Arguments, Pre-flight, and Constraints
- [x] A-017 R17: the four new SPEC files exist, follow house style, and match their skill sources
- [x] A-018 R18: exclusion policy for `_cli-fab`/`_cli-external` documented; `SPEC-_preamble.md` exists; no live doc references `SPEC-preamble.md` (changelog/archive rows exempt)
- [x] A-019 R19: CONTRIBUTING.md has zero stageman/`fab/.kit`/dead-stage instructions; documented commands exist in the CLI
- [x] A-020 R20: fab-new/fab-draft Step 6 has no manual keyword list; documents hook ownership, `.status.yaml` verification, override-only-if-wrong, and re-fire overwrite caveat
- [x] A-021 R21: six frontmatter descriptions updated as specified; fab-fff.md verified to already name the Copilot request + poll (g4-5 no-op)
- [x] A-022 R22: New Skill Checklist section enumerates all eight integration points
- [x] A-023 R23: git-pr-review names its two STOP exceptions; skills.md rollup matches actual behavior (`--tool copilot` only, no cascade, Step-6 routing)
- [x] A-024 R24: fab-continue review-pr Fail branch carries the only-if-still-active guard
- [x] A-025 R25: both collision greps use `grep -lw`; SPEC mirrors updated
- [x] A-026 R26: fab-setup Creates block uses idempotent wording
- [x] A-027 R27: every edited skill's SPEC mirror checked; no mirror retains the pre-change wording

### Behavioral Correctness

- [x] A-028 R7: following the new generation rule produces valid Code Quality/extra-category items without fake R#s (this plan's own `## Acceptance` demonstrates it)
- [x] A-029 R20: no unconditional `set-change-type` remains; the hook/skill double-write race is removed at its root
- [x] A-030 R24: a git-pr-review-internal `fail` no longer triggers a second CLI-rejected `fail` from fab-continue

### Removal Verification

- [x] A-031 R6: `**Status**:` lines gone from both templates (and mirror examples in templates.md)
- [x] A-032 R15: `/meta:scriptify` and `/meta:review` exemplars gone
- [x] A-033 R20: the 7-line keyword-matching list is gone from both fab-new.md and fab-draft.md

### Scenario Coverage

- [x] A-034 R12: each table row was verified against resolve.go by grep before writing
- [x] A-035 R25: DEV-123/DEV-1234 word-boundary scenario documented in the skill text or verified by the grep semantics

### Edge Cases & Error Handling

- [x] A-036 R2: legacy `.fab-operator.yaml` mentions intentionally kept (not-migrated context) are still present and unambiguous
- [x] A-037 R20: wrong-hook-inference recovery path (override + re-fire caveat) is documented

### Code Quality

- [x] A-038: Pattern consistency — edits follow surrounding prose/table style of each file
- [x] A-039: No unnecessary duplication — definitions stated once and referenced (operator state file, exclusion policy)

### Documentation Accuracy

- [x] A-040: Every regenerated doc claim verified against Go source (resolve.go, status.go, preflight.go, artifact.go) — no new false claims introduced (notably: change_type verification does NOT claim preflight emits it)

### Cross References

- [x] A-041: No dangling links/references after the SPEC-_preamble rename; new SPEC files consistent with skills.md and the exclusion policy; `go test ./...` passes and `.claude/skills/` untouched

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Out-of-scope staleness observed (for a future sweep): `statusman/changeman/logman` in `docs/specs/{naming,architecture,glossary,user-flow,change-types}.md`; `docs/specs/srad.md` autonomy-table residue; `fab-sync.sh` mentions in `docs/specs/skills.md` fab-setup section; duplicate index row in CONTRIBUTING.md

## Deletion Candidates

- None — the sweep deleted its own dead text inline (the 7-line keyword list in fab-new/fab-draft, `**Status**:` template lines, the duplicate Gate row, the two phantom error-table rows, the `/meta:*` exemplars); no remaining code or docs were made redundant by these edits. The known out-of-scope staleness is already recorded in `## Notes` for a future sweep.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Honor all 9 intake assumptions verbatim (esp. #6 hook-owns-change_type, #7 add-the-6-subcommands, #9 hybrid SPEC coverage) | Intake is the state-transfer document; decisions already made there | S:95 R:85 A:95 D:95 |
| 2 | Confident | Underscore convention: keep the underscore (`SPEC-_review.md` form) — rename `SPEC-preamble.md` → `SPEC-_preamble.md`; mapping becomes mechanical `SPEC-{source-filename}.md` | Two of three existing partial SPECs would keep their name; mechanical mapping beats a strip-rule with two exceptions; rename is one `git mv` + one live ref | S:70 R:85 A:85 D:75 |
| 3 | Confident | J verification: skill text verifies `change_type` by reading `.status.yaml` (e.g. `grep '^change_type:'`), not "via `fab preflight`" — preflight.go FormatYAML does not emit change_type | Verified in Go source; writing the backlog's literal phrasing would create a new false doc claim, the exact defect class this change fixes | S:80 R:90 A:90 D:85 |
| 4 | Confident | Extend f036 to `fab-continue.md:81` and f038 to `_preamble.md` Common-Commands row — same stale strings, files already in scope | Identical staleness class at point-of-use; leaving them would re-introduce the contradiction the findings fixed | S:75 R:90 A:90 D:85 |
| 5 | Confident | A-rename scope: `src/kit/` only, plus `docs/specs/templates.md:63` (`lib/statusman.sh`) because that file is already being edited as a template mirror; other non-skill specs left for a future sweep (recorded in Non-Goals) | Intake section A scopes the rename to skill files; constitution mirror rule pulls in only the specs being edited | S:80 R:90 A:85 D:80 |
| 6 | Confident | M.4 anchor mechanism: `grep -lw` (word-boundary) — hyphen-safe since boundaries are checked only at match edges; `DEV-123` no longer matches `DEV-1234` | Intake suggests "word-boundary or exact-bracket"; -lw is the smallest portable change to the existing one-liner | S:70 R:90 A:90 D:80 |
| 7 | Confident | g4-5 is a no-op: `fab-fff.md` (line ~111) already names the Copilot request + 10-minute poll; repo grep for the stale "prints a stop message" phrase is empty | Batch 1's timeout-outcome work already covered it; verified in the current tree | S:85 R:95 A:90 D:90 |
| 8 | Tentative | H placement/style: exclusion policy + underscore convention live inside the skills.md "New Skill Checklist" section (not a separate file); new SPECs follow SPEC-_review.md's Summary/Flow house style | Checklist's SPEC-mirror item is the natural single home (avoids a second policy location); alternative homes (constitution amendment, skills/ README) are defensible | S:55 R:85 A:70 D:55 |
| 9 | Confident | R2 keeps legacy `.fab-operator.yaml` mentions at fab-operator.md ~:65/:121 and SPEC-fab-operator.md :7/:154 — they describe the abandoned repo-rooted file, not the live one | Replacing them would falsify the migration note; finding f114 targets only live-file references | S:80 R:90 A:90 D:85 |
| 10 | Confident | CONTRIBUTING rewrite stays minimal: replace the Stage Manager section with a short fab-CLI stage-queries section; fix only the `.kit/`-era descriptions; other drift (duplicate index row) noted, untouched | f126 names exactly these two fixes; broader rewrite risks scope creep in a mechanical batch | S:75 R:85 A:85 D:80 |

10 assumptions (1 certain, 8 confident, 1 tentative).
