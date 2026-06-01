# Intake: Fix stale `fab status` CLI invocations in fab-new and fab-draft

**Change**: 260601-w5sr-fix-stale-status-cli-args
**Created**: 2026-06-01
**Status**: Draft

## Origin

> Fix stale `fab status` CLI invocations in the fab-new and fab-draft skill sources.
> Both skills (`src/kit/skills/fab-new.md` lines 56, 89, 113 and `src/kit/skills/fab-draft.md`
> lines 58, 91, 115) pass a `.status.yaml` file path to `fab status add-issue`,
> `set-change-type`, and `advance`. The installed fab binary (1.9.3+) expects a change
> reference (4-char ID, folder substring, or full folder name) — not a path — so the path
> form errors at runtime.

Surfaced during a `/fab-discuss` session. A user running `/fab-new` on fab 1.9.3 hit a
runtime error: the skill text instructed `fab status set-change-type` with a `.status.yaml`
path, but the binary rejected it; the user recovered by retrying with the 4-char change ID
(`l6lo`). They flagged it as a likely-shared stale invocation worth sweeping across skills.

A sweep of `src/kit/skills/*.md` confirmed the scope: **only** `fab-new.md` and
`fab-draft.md` carry the stale path form. The rest of the suite (`fab-continue.md`,
`fab-ff.md`, `fab-fff.md`) and the canonical `_cli-fab.md` reference already use the correct
`<change>` form.

## Why

1. **The pain point**: `/fab-new` and `/fab-draft` instruct the agent to call
   `fab status {add-issue,set-change-type,advance}` with a `.status.yaml` *path* as the first
   positional argument. The installed binary's command signatures are:
   - `fab status set-change-type <change> <type>`
   - `fab status advance <change> <stage> [driver]`
   - `fab status add-issue <change> <id>`

   where `<change>` is a change reference (4-char ID, folder substring, or full folder name).
   Passing a path errors at runtime, breaking the two most common entry points into the
   workflow.

2. **The consequence if unfixed**: Every `/fab-new` and `/fab-draft` invocation forces the
   agent to error, diagnose, and retry — wasting a turn and undermining trust in the very
   first command new users run. `set-change-type` and `advance` are not optional steps;
   they run on every invocation, so the failure is not edge-case.

3. **Why this approach**: The fix is a mechanical signature correction — replace the path
   form with the change reference already in scope. `/fab-new` Step 3 captures the folder
   name from `fab change new` stdout (`{name}`), and the ID is derivable; both are valid
   `<change>` references. This brings the two stragglers in line with the rest of the suite
   and the authoritative `_cli-fab.md`, with no behavior change beyond making the documented
   calls actually execute. No alternative is warranted — this is a doc/skill correctness bug,
   not a design choice.

## What Changes

### `src/kit/skills/fab-new.md` (3 lines)

Replace the `.status.yaml` path argument with the change reference (`{name}`, the folder
name already captured in Step 3):

| Line | Current (stale) | Corrected |
|------|-----------------|-----------|
| 56 | `fab status add-issue fab/changes/{name}/.status.yaml DEV-988` | `fab status add-issue {name} DEV-988` |
| 89 | `fab status set-change-type fab/changes/{name}/.status.yaml <type>` | `fab status set-change-type {name} <type>` |
| 113 | `fab status advance fab/changes/{name}/.status.yaml intake` | `fab status advance {name} intake` |

### `src/kit/skills/fab-draft.md` (3 lines)

Identical correction, same three subcommands:

| Line | Current (stale) | Corrected |
|------|-----------------|-----------|
| 58 | `fab status add-issue fab/changes/{name}/.status.yaml DEV-988` | `fab status add-issue {name} DEV-988` |
| 91 | `fab status set-change-type fab/changes/{name}/.status.yaml <type>` | `fab status set-change-type {name} <type>` |
| 115 | `fab status advance fab/changes/{name}/.status.yaml intake` | `fab status advance {name} intake` |

### Change reference: `{name}` vs `{id}`

Use `{name}` (the full folder name captured from `fab change new` stdout in Step 3 / Step 4
of each skill). It is unambiguous, already in scope at each call site, and a valid `<change>`
reference. This matches how `fab-continue.md`, `fab-ff.md`, and `fab-fff.md` already pass
`<change>` (they use the resolved change reference, not a path). Using `{name}` keeps the
six edits uniform and requires no new variable capture.

### SPEC files — verification only (likely no edit)

The constitution requires that skill-file changes update the corresponding
`docs/specs/skills/SPEC-*.md`. On inspection, **`SPEC-fab-new.md` and `SPEC-fab-draft.md`
already document the correct `<change>` form** (e.g., `fab status set-change-type <change>
<type>`, `fab status advance <change> intake`). The specs were already correct; only the
skill *sources* drifted. The spec stage should verify alignment and add an edit only if a
discrepancy is found — no proactive spec rewrite is expected.

## Affected Memory

<!-- This is an implementation-only correctness fix to skill source text. It changes no
     spec-level behavior — the documented behavior was always "pass a change reference";
     the skill text simply didn't match it. No memory file create/modify/remove. -->

- No memory changes. This corrects skill source text to match already-documented behavior;
  it introduces no new or changed spec-level behavior. `docs/memory/fab-workflow/` already
  describes the correct `<change>`-reference contract.

## Impact

- **Code areas**: `src/kit/skills/fab-new.md`, `src/kit/skills/fab-draft.md` (6 lines total).
- **Specs**: `docs/specs/skills/SPEC-fab-new.md`, `docs/specs/skills/SPEC-fab-draft.md` —
  verify alignment (expected already-correct).
- **Deployed copies**: `.claude/skills/fab-new/SKILL.md` and `.claude/skills/fab-draft/SKILL.md`
  are gitignored generated copies produced by `fab sync`; they regenerate from source and
  must NOT be hand-edited (constitution §V, context.md).
- **CLI / binary**: None. No Go code changes; the binary signatures are already correct and
  are what we're aligning to.
- **Tests**: No Go test changes (binary unchanged). If skill-text fixtures or doc-consistency
  checks exist, they should pass post-fix; verify none assert the stale path form.
- **Blast radius**: Tiny and fully reversible — six text substitutions in two markdown files.

## Open Questions

- None blocking. The `{name}` vs `{id}` choice is resolved in favor of `{name}` (folder name,
  already in scope). The SPEC files appear already-correct, to be confirmed at spec stage.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Installed `fab status {add-issue,set-change-type,advance}` expect a `<change>` reference, not a `.status.yaml` path | Verified directly against `fab status <sub> --help` on the installed binary (1.9.6) and against `_cli-fab.md`; reproduced as the user's runtime error on 1.9.3 | S:98 R:90 A:95 D:95 |
| 2 | Certain | Scope is exactly two files / six lines: `fab-new.md` (56, 89, 113) and `fab-draft.md` (58, 91, 115) | Full `grep` sweep of `src/kit/skills/*.md` for path-form `fab status` calls returned only these six; rest of suite already correct | S:98 R:88 A:95 D:92 |
| 3 | Confident | Replace path arg with `{name}` (full folder name) rather than `{id}` | `{name}` is captured in Step 3/4 of both skills, already in scope, and is a valid `<change>` reference per `_cli-fab.md`; keeps all six edits uniform with no new variable capture | S:80 R:85 A:85 D:78 |
| 4 | Confident | No memory changes required | Fix corrects skill text to match already-documented behavior; introduces no new or changed spec-level behavior | S:78 R:80 A:88 D:82 |
| 5 | Confident | SPEC-fab-new.md and SPEC-fab-draft.md need no edit (verify only) | Inspection shows both specs already document the correct `<change>` form; constitution's "update SPEC-*" obligation is already satisfied | S:82 R:80 A:88 D:80 |
| 6 | Certain | Edit source under `src/kit/skills/`, never the deployed `.claude/skills/` copies | Constitution §V and context.md: `.claude/skills/` is gitignored, regenerated by `fab sync`; editing it is overwritten on next sync | S:98 R:85 A:95 D:95 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
