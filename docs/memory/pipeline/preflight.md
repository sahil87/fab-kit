---
type: memory
description: "`fab preflight [change-name]` — the orient seam: init/change/`.status.yaml` validation, structured YAML output (`id`/`name`/`change_dir`/`stage`/`display_stage`/`display_state`/`progress`/`plan`/`confidence`), transient change-name override, live acceptance derivation, and the best-effort self-healing recompute that makes it a writer, not a pure reader"
---
# Preflight

**Domain**: pipeline

## Overview

`fab preflight [change-name]` validates the active change's state and prints a structured YAML document to stdout for agent consumption. It consolidates the validation every skill would otherwise inline — project initialization, change resolution, `.status.yaml` presence and schema — into one command, and is the **orient seam**: the read/orientation point every change-operating skill hits before acting. Preflight is the strict validation gate (non-zero exit on a bad state by design); `fab resolve` is the pure query that can answer "none" when asked.

The command lives in `preflightCmd` (`src/go/fab/cmd/fab/preflight.go`) as a cobra command taking at most one positional argument; the validation and derivation logic live in `Run` and `FormatYAML` (`src/go/fab/internal/preflight/preflight.go`).

## Requirements

### Structured YAML Output

`FormatYAML` writes a fixed field sequence to stdout, in this order:

- `id` — the 4-char change ID, derived from the folder name by `resolve.ExtractID`
- `name` — the resolved change folder name
- `change_dir` — the change directory as `fab/changes/{name}` (repo-relative)
- `stage` — routing stage: what `/fab-continue` will work on next (`status.CurrentStage`)
- `display_stage` — display stage: "where you are" in the pipeline (`status.DisplayStage`). Five-tier walk: first `active`, else first `failed` (parked review/review-pr failures surface instead of falling through to the last done stage), else first `ready`, else last `done`/`skipped`, else first `pending` — see [change-lifecycle.md](/pipeline/change-lifecycle.md) "Deriving display stage"
- `display_state` — the state of the display stage: `active`, `ready`, `done`, `failed`, `pending`, or `skipped`
- `progress` — nested map of all 6 stages to their state, from `GetProgressMap()`
- `plan` — nested block: `generated` (bool), `task_count`, `acceptance_count`, `acceptance_completed`
- `confidence` — nested block: `certain`, `confident`, `tentative`, `unresolved` (ints) and `score` (one decimal place)

`confidence.indicative` is not emitted; the `statusfile` struct field stays decode-tolerant so a legacy `indicative: true` key on disk round-trips harmlessly. `change_type` is **not** a preflight field — callers that need it read it from the change's `.status.yaml` directly (`_intake.md`, `_review.md`, `_pipeline.md` all note this).

Agents consume the output by running the command via Bash and parsing the stdout YAML directly, using those fields for all subsequent change context instead of re-reading `.status.yaml`.

**Live acceptance derivation**: `Run` overrides the persisted `plan.acceptance_completed`/`acceptance_count` with a live count from `plan.md`'s `## Acceptance` checkboxes when `status.LiveAcceptance(changeDir)` succeeds, so a hook-bypassing edit (`sed`, a direct file write) cannot leave preflight reporting a stale counter. The persisted counters remain the write-time cache and the fallback when `plan.md` or its `## Acceptance` section is absent.

### Validation Checks

`Run` validates in this order, returning at the first failure:

1. `fab/project/config.yaml` exists — otherwise `Project not initialized — fab/project/config.yaml not found. Run /fab-setup.`
2. `fab/project/constitution.md` exists — same message shape for the constitution
3. Sync staleness check (non-blocking) — `checkSyncStaleness` compares the kit's `VERSION` file (under `kitpath.KitDir()`) against the project's `fab_version` from `config.Load` (sourced from the plain-text `fab/.fab-version`), emitting `⚠ Skills may be out of sync — run fab sync to refresh (engine {kit}, project {project})` to stderr when they differ. Any read/parse failure or an empty value silently skips the warning — the check never blocks and never touches stdout
4. Change name resolves via `resolve.ToFolder` — from the positional override, else the `.fab-status.yaml` symlink with single-change fallback
5. The change directory `fab/changes/{folder}` exists
6. `.status.yaml` exists within it
7. `.status.yaml` loads (`statusfile.Load`) and passes `status.Validate` — catching invalid states, missing stages, and multiple active stages

Every failure returns a Go error, which cobra surfaces on stderr with a non-zero exit; skills STOP and surface that message verbatim, since it carries the specific error and suggested fix. The staleness check is the sole exception — advisory only, never blocking.

### Change-Name Override

The optional positional `[change-name]` argument is passed through to `resolve.ToFolder` as the override, replacing default resolution via the `.fab-status.yaml` symlink. Resolution is exact-match-first, then case-insensitive substring matching against non-archive folder names, so full folder names, partial slugs, and 4-char IDs all work. The override is **transient** — `.fab-status.yaml` is never modified — which is what lets parallel sessions target different changes concurrently.

### Preflight Is a Writer at the Orient Seam

Preflight is **not** read-only. Before the derivation, `preflightCmd` calls `refreshPreflightState(fabRoot, changeOverride)`, a locked load-mutate-save that persists the artifact-derived `.status.yaml` fields (`change_type`, `confidence.*`, plan counts) when they have drifted from `intake.md`/`plan.md`. It resolves the status path, takes the same `lockfile.WithLock` flock the `fab status` mutators use, loads the file, runs `refresh.Refresh`, and `Save`s exactly once when dirty.

Every failure along that path is swallowed — an unresolvable change, an unreadable status file, or a scoring hiccup all leave preflight free to continue and surface state. This best-effort posture is safe because the forward seams (`fab status advance`/`finish`) heal on the write path anyway. Preflight is the **orient** member of that seam set; the seam rationale, the excluded transitions (`start`/`reset`/`skip`/`fail`), and the governing principle live in [hooks-may-enhance-never-own.md](/pipeline/hooks-may-enhance-never-own.md) § Self-Healing at the Forward and Orient Seams Only.

The split of responsibility is deliberate: `preflight.Run` is a pure reader, and the mutation lives in `cmd/fab` so the write path routes through the same lock discipline as `fab status`.

### Skill Integration

Skills operating on a change run preflight per `_preamble.md` §2: execute `fab preflight [change-name]`, STOP and surface stderr on a non-zero exit, then parse the stdout YAML for change context. Because preflight already checks that `config.yaml` and `constitution.md` exist, those skills need no separate existence check — they only read the files for content.

After preflight, skills log the invocation with a direct `fab log command "<skill>" "<id>"` call (`_preamble.md` §2 step 4), passing the `id` from the preflight output. That command owns a best-effort contract — it always exits 0 given valid usage, printing internal failures as a stderr warning — so no shell guard is needed.

Skills that operate without an active change do not run preflight: `/fab-setup`, `/fab-new`, `/fab-switch`, `/fab-discuss`, `/fab-help`, `/fab-dedupe` (which must not resolve or disturb a change), and `/fab-operator` (a listed exception to the always-load layer). Exempt skills that still want telemetry call `fab log command` directly.

## Design Decisions

### Pure Reader Plus a Locked Writer at the Command Layer
**Decision**: `internal/preflight.Run` performs no writes; the self-healing recompute lives in `refreshPreflightState` in `cmd/fab/preflight.go`, running under `lockfile.WithLock` before `Run` is called.
**Why**: Keeping `Run` a pure reader lets it stay importable and trivially testable, while routing the write through the same flock discipline as the `fab status` mutators avoids a second, unserialized writer for `.status.yaml`. The two concerns are separable because the recompute's output is only ever observed through the subsequent read.
**Rejected**: Folding the recompute into `preflight.Run` — makes a validation-and-derivation function a mutator and puts a write path outside the status package's locking conventions.

### Best-Effort Recompute Never Blocks Orientation
**Decision**: Every error inside `refreshPreflightState` is swallowed; preflight proceeds to `Run` regardless.
**Why**: Preflight's job is to orient the agent, and it must do that even when a recompute cannot run — a swallowed error degrades to slightly stale derived fields, whereas a propagated one would abort the orient step entirely and strand the skill. The forward seams (`advance`/`finish`) heal the same fields on the write path, so the window a swallowed failure leaves open is transient.
**Rejected**: Failing preflight on a recompute error — converts an advisory self-heal into a hard stop for an unrelated concern.

### Validation Gate, Not a Queryable Probe
**Decision**: Preflight exits non-zero on any unresolvable or invalid state; there is no flag that turns absence into a success-channel answer.
**Why**: The two roles are split across two commands — preflight is the gate skills rely on for a hard stop before acting, and `fab resolve --or-none` is the probe for "is a change active?" that answers `(none)` on exit 0. Collapsing them would leave callers unable to express "this must be valid" without extra checking.
**Rejected**: An `--or-none`-style flag on preflight — duplicates `fab resolve`'s probe role and weakens the gate every skill's pre-flight step depends on.
