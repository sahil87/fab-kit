# Intake: Fix Kit Scripts

**Change**: 260227-yobi-fix-kit-scripts
**Created**: 2026-02-27
**Status**: Draft

## Origin

> Fix issues surfaced in retrospective findings from a `/fab-ff` session (DEX-44). Original findings at `.tmp/findings.txt`. After investigation, 2 of 5 items were already resolved; remaining 3 consolidated into this change. During intake discussion, scope expanded to include a shipped script invocation guide (`_scripts.md`) to address cross-repo reliability — agents in other repos fail more because they lack the calling conventions that are only discoverable by reading script source.

## Why

Kit scripts have inconsistent argument conventions and a sequencing trap that cause agent failures during pipeline execution. The retrospective logged 5+ failed `stageman.sh` calls in a single session due to history commands expecting directory paths while all other commands accept change identifiers. Separately, `calc-score.sh --check-gate` for the spec stage reads a pre-computed score from `.status.yaml` instead of computing it, requiring callers to know a two-step invocation dance.

These issues compound in repos with sparse memory/specs context where the agent has less surrounding information to recover from script failures. The root cause isn't just buggy scripts — it's that correct calling conventions aren't in the always-loaded context. Agents discover conventions by grepping source or trial-and-error, which fails across repos.

## What Changes

### 1. Unify stageman history commands onto `resolve_change_arg`

The three history commands (`log-command`, `log-confidence`, `log-review`) currently use `resolve_change_dir()` which only understands directory paths. All other stageman commands use `resolve_change_arg()` which accepts change IDs, folder names, or `.status.yaml` paths and resolves via `changeman.sh resolve`.

**Fix**: Route history commands through `resolve_change_arg`, then derive the change directory from the resolved `.status.yaml` path (strip the filename). This simultaneously fixes the `log-review` path bug (where passing a `.status.yaml` path produced `{path}/.status.yaml/.history.jsonl`) and standardizes all commands on the same argument convention.

- Update `log_command()`, `log_confidence()`, `log_review()` to accept the resolved `.status.yaml` path and derive change dir via `dirname`
- Update CLI dispatch for `log-command`, `log-confidence`, `log-review` to call `resolve_change_arg` like every other command
- Update help text to show `<change>` instead of `<change_dir>` for history commands
- Remove `resolve_change_dir()` — no callers will remain
- Update internal callers that pass directory paths:
  - `changeman.sh` lines 398, 486: pass folder name (`$folder_name`, `$new_name`) instead of full directory path (`$changes_dir/$folder_name`)
  - `calc-score.sh` line 329: pass `$status_file` instead of `$change_dir`
  - Skill prompts (`fab-ff.md`, `fab-fff.md`, `fab-continue.md`, `fab-clarify.md`): update `<change_dir>` to `<change>` in history command examples

### 2. Make `calc-score.sh --check-gate` self-contained for spec stage

The intake gate path (line 137-171) already parses `intake.md` on the fly — it's self-contained. The spec gate path (line 174-181) reads `score:` and counts from `.status.yaml`, requiring a prior non-gate `calc-score.sh` run to have populated those values.

**Fix**: Make the spec gate path parse `spec.md` and compute the score inline, mirroring the intake gate's approach. The gate check becomes a pure read-only query that doesn't depend on prior side effects.

### 3. Create `_scripts.md` — shipped script invocation guide

Create `fab/.kit/skills/_scripts.md` as a new always-loaded skill file (alongside `_preamble.md` and `_generation.md`). This is the primary deliverable for cross-repo reliability — it puts correct calling conventions directly in every skill's context.

Contents:
- **`<change>` argument convention**: All kit scripts accept a unified `<change>` argument. Accepted forms: 4-char change ID (`yobi`), folder name substring (`fix-kit`), full folder name (`260227-yobi-fix-kit-scripts`), `.status.yaml` path. Resolution is handled by `changeman.sh resolve` internally.
- **Script summaries**: One section per script (`changeman.sh`, `stageman.sh`, `calc-score.sh`, `preflight.sh`) with key subcommands and signatures
- **Stage transition side effects**: `finish intake` auto-activates spec, `finish spec` auto-activates tasks, etc.
- **Common error patterns**: What "Status file not found" means, what "Not a directory" means, etc.

Loading: `_preamble.md` gets a one-liner in its always-load section referencing `_scripts.md`. The chain is: skill → preamble → scripts. Every skill that reads the preamble automatically picks up script conventions.

Update `_generation.md` to use `<change>` placeholder consistently (currently shows `<file>` on lines 95-97).

### 4. Create memory reference for kit scripts

Create `docs/memory/fab-workflow/kit-scripts.md` as the deep reference for agents working *on* the scripts themselves. Full implementation details: internal functions, resolution logic, schema details, design rationale. This is supplementary to `_scripts.md` — the shipped guide covers "how to call", the memory file covers "how it works".

## Affected Memory

- `fab-workflow/kit-scripts`: (new) Deep reference for kit shell script internals — resolution logic, state machine, implementation details

## Impact

- `fab/.kit/scripts/lib/stageman.sh` — history commands refactored, `resolve_change_dir` removed
- `fab/.kit/scripts/lib/calc-score.sh` — spec gate path rewritten to parse-then-check; `log-confidence` call updated
- `fab/.kit/scripts/lib/changeman.sh` — `log-command` calls updated to pass folder name instead of directory path
- `fab/.kit/skills/_scripts.md` — new file (shipped)
- `fab/.kit/skills/_preamble.md` — add one-liner to always-load section referencing `_scripts.md`
- `fab/.kit/skills/_generation.md` — update `<file>` placeholders to `<change>`
- `fab/.kit/skills/fab-ff.md`, `fab-fff.md`, `fab-continue.md`, `fab-clarify.md` — update `<change_dir>` to `<change>` in history command examples
- `docs/memory/fab-workflow/kit-scripts.md` — new file
- `docs/memory/fab-workflow/index.md` — add kit-scripts entry

## Open Questions

None — scope is well-defined from investigation and discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | History commands route through `resolve_change_arg` | Discussed — user confirmed standardize on change ID. `resolve_change_arg` already handles this for all other commands | S:90 R:85 A:95 D:95 |
| 2 | Certain | Spec gate parses spec.md inline like intake gate | Discussed — user confirmed parse-then-check in one invocation. Intake gate already demonstrates the pattern | S:85 R:80 A:90 D:95 |
| 3 | Certain | `changeman.sh rename` dropped from scope | Discussed — already implemented. Verified in source (line 406) | S:95 R:95 A:95 D:95 |
| 4 | Certain | Script guide is a new shipped file `_scripts.md` | Discussed — user confirmed separate file over adding to `_preamble.md`. Matches underscore-prefix convention | S:90 R:85 A:90 D:90 |
| 5 | Certain | `_preamble.md` references `_scripts.md` in always-load | Discussed — ensures every skill loads script conventions without changing individual skill files | S:85 R:90 A:90 D:95 |
| 6 | Certain | Keep `resolve_change_arg` strict — no directory support | Discussed — separation of concerns. Each script accepts change IDs at the boundary. Internal callers pass folder names or `.status.yaml` paths, not directory paths | S:85 R:85 A:90 D:90 |
| 7 | Certain | Internal callers pass folder names, not directory paths | Discussed — changeman.sh passes `$folder_name` (a valid change ID), calc-score.sh passes `$status_file`. Round-trip through `changeman.sh resolve` is redundant but correct | S:80 R:85 A:90 D:90 |
| 8 | Confident | Derive change dir via `dirname` on resolved `.status.yaml` | Natural approach — `resolve_change_arg` returns `.status.yaml` path, `dirname` gives the change dir | S:70 R:90 A:85 D:90 |
| 9 | Confident | Remove `resolve_change_dir` rather than deprecate | After this change, no callers remain. Dead code removal is cleaner than deprecation shim | S:65 R:85 A:80 D:85 |
| 10 | Confident | Memory file is supplementary deep reference | `_scripts.md` is the primary deliverable for agent reliability. Memory file serves agents maintaining the scripts and human reference | S:75 R:85 A:80 D:80 |
| 11 | Confident | Update `_generation.md` and skill prompts to `<change>` | Aligns with unified convention. Currently inconsistent across `_generation.md`, `fab-ff.md`, `fab-fff.md`, `fab-continue.md`, `fab-clarify.md` | S:70 R:90 A:85 D:90 |

11 assumptions (7 certain, 4 confident, 0 tentative, 0 unresolved).
