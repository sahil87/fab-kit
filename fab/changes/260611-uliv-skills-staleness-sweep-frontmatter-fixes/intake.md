# Intake: Skills Staleness Sweep + Frontmatter Description Accuracy (Skills-Review Batch 2/4)

**Change**: 260611-uliv-skills-staleness-sweep-frontmatter-fixes
**Created**: 2026-06-11

## Origin

> /fab-new uliv

One-shot invocation from backlog ID `uliv` (`fab/backlog.md:19`). The backlog entry is batch 2 of the 4-batch remediation of the 2026-06-11 skills review (`docs/specs/findings/skills-review-2026-06-11.md`, line numbers vs commit `ae79e04c`). Batch 1 (`9u91`, correctness + idempotency) shipped as draft PR #390; batches 3 (`zc9m`, _preamble context diet) and 4 (`szxd`, twins refactor) follow later. Per the batch-1 session's handoff note, four residual non-blocking findings from PR #390's "Known follow-ups" plus two reviewer-flagged staleness items are folded into this batch's scope (section M below) — they are not in the backlog text itself.

## Why

The skills review found widespread *stale references* across skill files, templates, and docs: pre-Go-CLI command names (`statusman`, `changeman`, `logman`), pre-1.10.0 spec-stage vocabulary (the spec stage was merged into apply in change `j6cs`), an operator state-file path that changed during the multi-repo redesign, CLI reference tables that no longer match the Go binary's actual behavior, and frontmatter `description:` lines that misdescribe what skills do (hurting model-invocation accuracy — the agent picks skills based on those one-liners).

If left unfixed, agents following these skills execute commands that don't exist, describe artifacts that were removed, and route users to the wrong skills. Staleness compounds: every new skill copies the nearest existing pattern, so wrong text propagates. The review batched the remediation deliberately — this batch is the *mechanical* sweep (rename, rewrite, regenerate, delete dead lines) with **no behavior changes except where explicitly flagged** (sections D, J, and M.2 change normative skill behavior; everything else is text accuracy).

A dedicated batch (rather than ad-hoc fixes) is the right approach because the findings report pins every edit to file:line at a known commit, making the work verifiable and reviewable as one coherent diff.

## What Changes

> Finding IDs (`f###`, `g#-#`) reference `docs/specs/findings/skills-review-2026-06-11.md`. All line numbers are vs commit `ae79e04c` — verify content at each location before editing (treat line refs as locators, not gospel).
>
> **Constraints (apply to every section):**
> - All edits in `src/kit/` (canonical source) — **never** `.claude/skills/` (deployed copies, gitignored)
> - Templates live in `src/kit/templates/`
> - Each skill edit updates its `docs/specs/skills/SPEC-*.md` mirror (constitution rule)

### A. Pre-Go-CLI command rename (f030)

Global rename across ~9 skill files: `statusman` → `fab status`, `changeman` → `fab change resolve`, `logman` → `fab log`. Worst occurrences: `git-pr.md:243` and `git-pr.md:252`. Mechanical find-and-replace, but verify each replacement's argument form matches the current `fab` CLI signature (see `_cli-fab.md`).

### B. Operator state-file references (f114)

`fab-operator.md`: define "operator state file" **once** in section 4 with the real server-keyed path (state file keyed by tmux socket path under `XDG_STATE_HOME` — the multi-repo operator design), then replace the 9 stale `.fab-operator.yaml` mentions at lines 23, 29, 33, 117, 277, 344, 551, 565, 645 with the defined term.

### C. Template spec-stage residue (f062+g3-1, g3-2, g3-5)

- **f062+g3-1** — `src/kit/templates/intake.md`: rewrite the 4 HTML-comment lines still referencing the removed spec stage (lines 29, 46, 51, 58–59) to plan/apply-entry vocabulary per `_generation.md:22-23` (the canonical phrasing: the downstream *apply-entry agent co-generates `plan.md`*; "primary input for spec generation" → "primary input for plan generation"; the Assumptions STATE TRANSFER comment's "spec-stage agent" → "apply-entry agent").
- **g3-2** — `src/kit/templates/plan.md:214`: fix "all four SRAD grades may appear" to **three grades** (Unresolved is intake-only — apply *decides and records*, it never leaves Unresolved). Correspondingly scope the `_preamble.md:477` "include all four grades" rule to **intake artifacts only**.
- **g3-5** — delete the dead `**Status**:` header lines from `src/kit/templates/intake.md:5` and `src/kit/templates/plan.md:4`. No skill ever reads or flips them; `.status.yaml` is the state of record.

### D. Acceptance R# trace exemption (f061+g3-3) — *behavior-flagged*

`_generation.md` step 6 + `src/kit/templates/plan.md:155`: exempt non-requirement-derived acceptance categories (**Code Quality**, `checklist.extra_categories`) from the "each `## Acceptance` item MUST name the requirement it accepts" rule. The templates' own A-007/A-008 examples already violate the MUST as written. Requirement-derived categories keep the mandatory `A-{NNN} R#:` form; exempted categories use `A-{NNN}: {outcome}` without an R# reference.

### E. `_preamble` + `_cli-fab` staleness (f050, f054, f036, f038, f052, f053)

- **f050** — `_preamble.md:405-413` autonomy table: update the `fab-continue` column to post-1.10.0 reality (SRAD scoring at intake only; 1–2 questions at intake, 0 at apply and later; drop the "[NEEDS CLARIFICATION] count" from its Output row).
- **f054** — `_cli-fab.md:78-87`: collapse the duplicate "Gate" / "Intake gate" rows (identical invocation — spec-stage residue from when there were two gates).
- **f036** — `(review only)` → `(review/review-pr only)` at `_preamble.md:252` and `_cli-fab.md:61` (the `fab status fail` constraint).
- **f038** — add `id`, `display_stage`, `display_state` to the preflight output field lists at `_preamble.md:56` and `_cli-fab.md:92` (the binary already emits them; docs lag).
- **f052** — `_cli-fab.md:463-470`: regenerate the Common Error Messages table from the actual strings in `internal/resolve/resolve.go` (3 of 4 rows are wrong).
- **f053** — `_cli-fab.md` `fab status` table: add the 6 missing visible query subcommands (preferred — `_cli-fab` is chartered as the *full* reference) or, failing that, retitle the table to declare partial coverage.

### F. `_generation` consumer list (f060+g4-8)

`_generation.md:3,11`: fix the stale consumer list — 5 consumers (fab-new, fab-draft, fab-continue, fab-ff, fab-fff), not 2 (the text currently names only fab-continue and fab-ff).

### G. Internal-skill exemplars (f029, f027)

- **f029** — `internal-retrospect.md`: replace nonexistent `/meta:*` command exemplars with `/internal-skill-optimize`.
- **f027** — `internal-skill-optimize.md:15,87`: change the exclusion from 2 named files to **all `_*.md` partials**.

### H. SPEC coverage + naming (f048)

Six skill sources have no `docs/specs/skills/SPEC-*.md` mirror: `internal-consistency-check`, `internal-retrospect`, `internal-skill-optimize`, `_generation`, `_cli-fab`, `_cli-external`. Either create the 6 missing SPEC files or document an explicit exclusion policy for internal-*/helper files. Recommended split: create SPECs for the 3 user-invocable `internal-*` skills and the behavioral `_generation` partial (precedent: `SPEC-_review.md` and `SPEC-preamble.md` already exist for behavioral partials); document an exclusion policy for the 2 pure-reference partials (`_cli-fab`, `_cli-external`) whose content mirrors the CLI rather than defining behavior.
<!-- assumed: hybrid SPEC coverage — 4 new SPEC files (internal-* ×3 + _generation) + documented exclusion for _cli-* reference partials; backlog leaves create-vs-exclude open -->
Also normalize the underscore convention: `SPEC-_review.md` vs `SPEC-preamble.md` disagree on whether the partial's leading underscore is kept — pick one form and rename the outlier.

### I. CONTRIBUTING.md modernization (f126)

Rewrite the Stage Manager section for the Go CLI + 6-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`), and fix the `fab/.kit` reading-path links (kit content now lives in the system cache at `~/.fab-kit/versions/<version>/kit/`, not in-repo `.kit/`).

### J. Change-type inference alignment (g3-4) — *behavior-flagged*

`fab-new.md:77-90` + `fab-draft.md:77-93`: the skills' manual keyword-matching step conflicts with the PostToolUse hook (`artifact.go:90-111`), which uses word-boundary regexes (including `redesign`) and **re-fires on every intake write, silently overwriting the skill-set value**. Per the backlog's recommendation: replace the manual keyword step with — *the intake-write hook sets `change_type`; verify via `fab preflight`; override with `fab status set-change-type` only if wrong*. This removes the skill/hook double-write race instead of documenting around it.

### K. Frontmatter description fixes (model-invocation accuracy, one-liners each)

| ID | File | Fix |
|----|------|-----|
| g4-2 | `fab-continue.md` | Use canonical stage names; add ship/review-pr |
| g4-6 | `fab-proceed.md` | Takes no arguments — point to `/fab-fff <change>` for targeting a named change |
| g4-9 | `git-pr.md` | Creates a **DRAFT** PR |
| g4-7 | `fab-switch.md` | `--none` deactivates |
| g4-4 | `docs-reorg-memory.md` | Also the memory rebalancer — shape diagnosis, split/merge/flatten |
| g4-10 | `git-branch.md` | Unmatched explicit names fall back to a literal standalone branch |
| g4-5 | `fab-fff.md:109` | Subagent requests a Copilot review and polls up to 10 min — not prints-stop-and-completes |

### L. New Skill Checklist (f125)

Add a "New Skill Checklist" section to `docs/specs/skills.md`: frontmatter fields, preamble-read line, `helpers:` declaration, `Next:` line, Error Handling + Key Properties tables, SPEC mirror file, skills.md row, help grouping.

### M. Batch-1 residuals (PR #390 "Known follow-ups" + reviewer notes)

1. **git-pr-review.md** — the "single exit point" sentence overclaims: Step 1.5 (invalid `--tool`) and Step 5 (commit failure) still STOP directly. Reword to name the exceptions.
2. **fab-continue.md** — *behavior-flagged*: the review-pr row's Fail branch lacks the only-if-still-active guard its Pass branch got in batch 1. Add the same guard.
3. **fab-operator.md** (~line 274) — the tick-loop summary still says dedupe "against known"; §7 step 2 is the corrected rule (**known + completed**). Align the summary.
4. **fab-new.md / fab-draft.md** — collision checks are substring/unanchored (`DEV-123` matches `DEV-1234`). Anchor the grep (e.g., word-boundary or exact-bracket match) in both skills' issue-ID collision checks.
5. **docs/specs/skills.md:792** — git-pr-review rollup drift (rollup text no longer matches the skill's actual exit behavior); fix alongside M.1.
6. **docs/specs/skills.md:128** — fab-setup "first run only" claim is stale (setup is re-runnable/idempotent); fix the wording.

## Affected Memory

- `pipeline/planning-skills`: (modify) change-type inference becomes hook-owned (J); fab-new/fab-draft collision-check anchoring (M.4)
- `pipeline/execution-skills`: (modify) git-pr-review exit-point exceptions (M.1); fab-continue review-pr fail-branch guard (M.2)
- `runtime/operator`: (modify) operator state-file term + server-keyed path (B); tick-loop dedupe summary (M.3)
- `memory-docs/templates`: (modify) intake/plan template comment rewrites, three-grades rule, Status-line removal (C); acceptance R# exemption (D)
- `memory-docs/specs-index`: (modify) SPEC coverage/exclusion policy + underscore normalization (H), only if the exclusion-policy half lands

## Impact

- **Skill sources** (`src/kit/skills/`): ~15 files touched — the ~9 rename targets (A), `fab-operator.md` (B, M.3), `_preamble.md` (C, E), `_cli-fab.md` (E), `_generation.md` (D, F), `internal-retrospect.md`, `internal-skill-optimize.md` (G), `fab-new.md`, `fab-draft.md` (J, M.4), frontmatter one-liners in 7 files (K), `git-pr-review.md` (M.1), `fab-continue.md` (M.2)
- **Templates** (`src/kit/templates/`): `intake.md`, `plan.md` (C, D)
- **Specs** (`docs/specs/`): `skills/SPEC-*.md` mirrors for every edited skill; up to 4 new SPEC files + 1 rename (H); `skills.md` (L, M.5, M.6)
- **Docs**: `CONTRIBUTING.md` (I)
- **Go code**: none — `internal/resolve/resolve.go` and `internal/status/artifact.go` (hook) are read-only references for regenerating doc tables (E) and aligning skill text (J)
- **Deploy**: after merge, `fab sync` redeploys `.claude/skills/` copies; no migration needed (no user-data restructuring)

## Open Questions

- None — the single open design choice (H: create-vs-exclude for missing SPECs) is carried as a Tentative assumption with a recommended hybrid; revisit via `/fab-clarify` if the recommendation is wrong.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is a mechanical staleness sweep — no behavior changes except the flagged items (D, J, M.2) | Stated verbatim in the backlog entry ("no behavior changes except where flagged") | S:95 R:85 A:90 D:90 |
| 2 | Certain | Fold the 4 PR #390 "Known follow-ups" residuals + 2 reviewer notes (skills.md:792, skills.md:128) into this batch (section M) | Explicit handoff instruction from the batch-1 session ("merge these 4 items into its action list when starting uliv") | S:90 R:80 A:90 D:85 |
| 3 | Certain | All edits in `src/kit/` (canonical); each skill edit updates its SPEC mirror; templates in `src/kit/templates/` | Constitution constraints + restated verbatim in the backlog CONSTRAINTS clause | S:95 R:80 A:95 D:95 |
| 4 | Certain | Report line numbers (vs `ae79e04c`) are locators only — verify content at each location before editing, since the worktree base includes the later cherry-pick `994e8100` | Only sane interpretation; blind line-number edits after a base-advancing commit would corrupt files | S:80 R:90 A:90 D:90 |
| 5 | Certain | Change type = `fix` | Deterministic keyword rule (content contains "fix"); the intake-write hook independently derives the same | S:85 R:90 A:95 D:90 |
| 6 | Confident | J: adopt the backlog's recommended resolution — hook owns `change_type`, skills verify via preflight and override only if wrong | Backlog says "recommended:"; hook behavior verified in `artifact.go:90-111`; removes the double-write race at its root | S:90 R:70 A:90 D:90 |
| 7 | Confident | f053: add the 6 missing `fab status` query subcommands rather than retitling the table | `_cli-fab` is chartered as the full reference ("every subcommand" per `_preamble.md`); retitling would codify the gap | S:60 R:90 A:90 D:75 |
| 8 | Confident | Build on the current `uliv` worktree base (main + `994e8100` cherry-pick); overlap conflicts with unmerged PR #390 are resolved at ship time | User pre-staged the dependency cherry-pick on this branch; batch ordering (2 after 1) was the review's explicit design | S:80 R:65 A:80 D:80 |
| 9 | Tentative | H: hybrid SPEC coverage — create 4 SPECs (internal-* ×3, `_generation`), document exclusion for `_cli-fab`/`_cli-external` | Backlog leaves create-vs-exclude open; precedent (SPEC-_review, SPEC-preamble exist for behavioral partials) supports the split, but full-create and full-exclude are both defensible | S:50 R:85 A:65 D:45 |

9 assumptions (5 certain, 3 confident, 1 tentative, 0 unresolved).
