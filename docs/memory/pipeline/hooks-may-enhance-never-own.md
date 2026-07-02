---
type: memory
description: "The governing principle 'hooks may enhance, never own' + the pull-based artifact-state model it produced (y022): correctness-critical `.status.yaml` state — change_type, confidence, plan counts — is recomputed by `fab status refresh`/`internal/refresh.Refresh` and self-healed at the forward+orient seams (`fab status advance`/`finish`, `fab preflight`), NOT owned by a Claude PostToolUse hook; the three push-by-nature session-telemetry hooks stay; the seam set (advance/finish/preflight) and why start/reset/skip/fail are excluded; git-staging dropped to `/git-pr` at ship"
---
# Hooks May Enhance, Never Own

**Domain**: pipeline

## Overview

The governing principle established in y022 and the pull-based artifact-state model it produced: **no correctness-critical state may live behind a Claude Code hook, because a hook fires only in the Claude harness.** Runtime telemetry (agent liveness/enrollment) is push-by-nature and MAY stay in a hook; artifact-derived pipeline state (`change_type`, intake `confidence`, `plan` counts) is correctness-critical and MUST be pull-based — recomputed on read/transition rather than on write. This file records the principle, the seam set that operationalizes it, and the rationale for the seams chosen and excluded. The `.status.yaml` field-level mechanics live in [schemas.md](/pipeline/schemas.md); the skill-side contract in [planning-skills.md](/pipeline/planning-skills.md) § Change Type (Pull-Based) and [execution-skills.md](/pipeline/execution-skills.md) § Pull-based bookkeeping; the removed hook handler in [runtime-agents.md](/runtime/runtime-agents.md) § Hook Pipeline.

> The principle is **stated** in y022's intake and enforced in code here. Its first *spec-prose* landing is 6sgj (change 3c of the cross-harness-dispatch series): `docs/specs/harness-adapters.md` fixes "hooks may enhance, never own" as a **dispatch-protocol** rule — a harness hook MAY add value around dispatch (telemetry, notifications) but MUST NOT own any step of the protocol, which is complete and correct without any hook. Promotion to a **normative constitution MUST-rule** remains later work (originally scoped to 3d, the series' final change; **3d/aetz did NOT add the constitution rule** — it is wiring-only and the principle stays stated-in-spec, enforced-in-code).
>
> **The `fab status refresh` epilogue is the protocol-owned step, written into the dispatch prompts — NOT a hook (aetz/3d).** Under the dispatch-protocol rule above, the worker-side post-stage state recompute is a **protocol** step, so 3d wires it as a **terminal `fab status refresh` epilogue in the dispatch prompt itself** (part of the dispatch-prompt obligations that bind BOTH adapters — native Agent-tool and CLI `fab dispatch` — per `_preamble.md` § Dispatch-Prompt Obligations), never as a harness hook a non-Claude worker would not fire. This is the concrete application of "if a step matters, it lives in the protocol (the prompt epilogue, the result-file obligation), not in a hook a different harness won't run": the same pull-based `fab status refresh` this file's principle produced is now the dispatched worker's own closing action, keeping a cross-harness-dispatched stage's `.status.yaml` consistent with the artifacts it wrote regardless of which harness ran it. The block-contract carve-out reconciles it with "the orchestrator owns all transitions": the prompt prohibits `fab status` *transition* commands but requires this one *recompute* (refresh is not a transition). See [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership and [runtime/dispatch.md](/runtime/dispatch.md).

## Requirements

### Requirement: Correctness-Critical Artifact State Is Pull-Based, Not Hook-Owned

The four artifact-derived `.status.yaml` field groups — `change_type` (from `intake.md`), `confidence.*`/`confidence.score` (from `intake.md`), and `plan.generated`/`plan.task_count`/`plan.acceptance_count`/`plan.acceptance_completed` (from `plan.md`) — SHALL be recomputed from the on-disk artifacts by `fab status refresh <change>` (backed by the shared `internal/refresh.Refresh`), never owned by a write-time hook. Before y022 these fields were written *only* by the `artifact-write` PostToolUse hook, which fired on Claude Code `Write`/`Edit` of `fab/changes/*/intake.md` and `plan.md`; that made the hook the **sole writer**, so any edit that did not fire it (a `sed`-style edit, a direct file write, or — the blocking case for the cross-harness dispatch series — a non-Claude agent such as `codex` writing the artifact) left the fields silently stale.

`Refresh` inspects **both** artifacts on disk (not scoped to a single written file the way the hook's `match.Artifact` was), preserves the `change_type_source: explicit` guard (re-infers only when the source is absent or `inferred` — a human's `set-change-type` survives every refresh), tolerates a missing artifact as a safe no-op, and is dirty-idempotent (a second run over unchanged artifacts produces no spurious `Save`).

#### Scenario: A hook-bypassing artifact write is healed at the next transition
- **GIVEN** a change whose `plan.md` was rewritten by a tool that never fired the Claude PostToolUse hook (`sed`, a codex worker, a direct write), leaving `plan.task_count` stale
- **WHEN** the next forward transition (`fab status advance`/`finish`) or orient (`fab preflight`) runs on that change
- **THEN** `refresh.Refresh` recomputes the derived fields from the on-disk artifacts and persists them in the same locked load/Save, so no downstream reader/gate ever acts on the stale value

#### Scenario: An explicitly set change_type is never clobbered by a refresh
- **GIVEN** a change with `change_type_source: explicit` (a human ran `fab status set-change-type`) whose intake prose would infer a different type
- **WHEN** `refresh.Refresh` runs at any seam
- **THEN** it skips inference and keeps the explicit type (the jznd sticky-explicit guard, preserved verbatim in the refresh path)

### Requirement: Push-By-Nature Telemetry Hooks Stay

The three session-scoped hooks — `fab hook session-start`, `fab hook stop`, `fab hook user-prompt` — SHALL remain: they record agent runtime state (`.fab-runtime.yaml` `_agents` map) which is **push-by-nature** (the event only exists inside the Claude harness that fires it) and does **not** own correctness-critical pipeline state. Their absence degrades gracefully (an untracked agent, not a corrupt pipeline). This is the enhance side of the principle: a hook that merely *records* harness-local telemetry is legitimate; a hook that *owns* state a non-Claude reader depends on is not.

#### Scenario: The removed artifact hook vs. the retained session hooks
- **GIVEN** the `artifact-write` PostToolUse hook (owned correctness-critical `.status.yaml` state) and the three session hooks (record harness-local agent telemetry)
- **WHEN** y022 applies the principle
- **THEN** only the artifact-write hook is removed (its state made pull-based); the three session hooks stay untouched

### Requirement: Self-Healing at the Forward and Orient Seams Only

Self-healing refresh SHALL run inside `fab status advance`, `fab status finish` (the **forward** seams — a stage transition follows an artifact-generation write), and `fab preflight` (the **orient** seam — the read/orientation point every skill hits before acting). It SHALL NOT run at `fab status start`, `reset`, `skip`, or `fail`, which move stage pointers without a preceding artifact-generation write. The forward seams plus the orient seam cover every point where a just-written artifact must be reflected before the next stage reads those fields — so drift can exist only *transiently mid-stage*, where nothing reads them. `preflight`'s refresh is **best-effort**: a recompute failure must not abort preflight's orient output (advance/finish already heal on the forward path, so a swallowed preflight refresh error is safe).

#### Scenario: start/reset/skip/fail do not refresh
- **GIVEN** any of `fab status start`/`reset`/`skip`/`fail`
- **WHEN** it runs
- **THEN** no `refresh.Refresh` occurs — the seam set stays minimal (verified by `TestStart_DoesNotSelfHeal`/`TestReset_DoesNotSelfHeal` and by inspection of the skip/fail closures)

## Design Decisions

### Remove the hook rather than keep it as a redundant belt-and-braces layer
**Decision**: y022 removes the `artifact-write` PostToolUse hook outright (drops its two `DefaultMappings` rows in both binaries, ships the `2.10.1-to-2.11.0` migration to strip the settings entry) rather than keeping it as a second, redundant writer alongside `fab status refresh`.
**Why**: A hook that "owns" correctness-critical state is precisely the anti-pattern the principle exists to eliminate. Keeping it would leave **two writers** for the same fields — a drift risk and a source-of-truth ambiguity — for no benefit, since refresh already covers every reachable-stale window. The `fab hook artifact-write` subcommand survives one release as a **silent no-op shim** (exit 0, nothing on stdout) purely so an un-migrated project's still-registered PostToolUse entry does not feed Claude Code ~505 bytes of cobra help as invalid `additionalContext`; it carries no bookkeeping logic.
**Rejected**: Belt-and-braces (hook + refresh both write) — reintroduces the dual-writer drift the change exists to kill. Delete-the-shim-now — would leave un-migrated projects emitting noisy help text on every Write/Edit until they migrate.
*Introduced by*: 260702-y022-status-refresh-drop-artifact-hook

### Pull-based extends the existing `LiveAcceptance` read-time-derivation precedent
**Decision**: Derive the artifact-state fields on demand (pull), caching opportunistically in the `.status.yaml` counters, rather than pushing them on every write.
**Why**: Two of the four field groups were *already* pull-based on the read path that matters — `status.LiveAcceptance(changeDir)` derives acceptance done/total from `plan.md` at **read** time for `fab preflight`/`fab pr-meta`/`fab status plan`, falling back to the cached counter only when the section is absent (schemas.md calls that counter a "write-time cache"). The remaining fields (`change_type`, `confidence.*`) were the load-bearing gap: nothing recomputed them on read. Extending the derive-on-demand/cache-opportunistically model to all four closes the gap uniformly and fixes the pre-existing `sed`/direct-edit warts as a side effect, with no `.status.yaml` schema change (`change_type_source` already existed).
**Rejected**: A per-field push kept in some other write path (still harness-coupled if a hook, still a remember-to-call burden if a skill instruction). Self-healing at *every* transition including `start`/`reset`/`skip`/`fail` (unnecessary surface — those are not artifact-generation seams).
*Introduced by*: 260702-y022-status-refresh-drop-artifact-hook

### `refresh` uses the non-logging `score.ApplyToStatus`, not `score.ComputeWithStatus`
**Decision**: `Refresh` recomputes confidence via `score.ApplyToStatus` (in-memory mutation, **no** `.history.jsonl` append), reserving the logging `score.ComputeWithStatus` for the explicit `fab score` path.
**Why**: Refresh runs at every self-healing transition and on every `fab preflight` — far more often than an explicit `fab score`. The logging variant would spam `.history.jsonl` with a no-delta `confidence` event on each read/orient. The three self-healing callers (`statusAdvanceCmd`/`statusFinishCmd` via `selfHealRefresh`, and `refreshPreflightState`) all swallow a `Refresh` error and proceed — the same best-effort, swallow-on-error posture the removed hook had, now under the status flock instead of a Claude harness hook. See [schemas.md](/pipeline/schemas.md) § Normal-mode failure surfacing.
*Introduced by*: 260702-y022-status-refresh-drop-artifact-hook

### Hook git-staging of `.status.yaml`/`.history.jsonl` dropped, not relocated
**Decision**: The best-effort `git add` of the change's `.status.yaml`/`.history.jsonl` (done today *only* by the artifact-write hook) is dropped — not folded into `Refresh` and not folded into the transition commands.
**Why**: The hook was the sole auto-stager (verified: no `fab status` mutator git-adds); `/git-pr` already commits status/history at ship, so the auto-stage was a convenience against transient "unstaged changes block a git op" friction, not a correctness guarantee. Folding it into `Refresh` would couple a pure state-recompute to git; folding it into the transitions would add git side effects to state mutations — both anti-patterns. Reversible either way. See [change-lifecycle.md](/pipeline/change-lifecycle.md) § Git Integration.
*Introduced by*: 260702-y022-status-refresh-drop-artifact-hook

### Shared `Refresh` extracted to `internal/refresh`, not folded into `internal/status`
**Decision**: The recompute logic moved out of the `main`-package `artifactBookkeeping` (`cmd/fab/hook.go`) into a dedicated `internal/refresh` package exposing `Refresh(fabRoot, changeDir string, sf *statusfile.StatusFile) (dirty bool, err error)`.
**Why**: `artifactBookkeeping` lived in the `main` package and could not be imported by `internal/preflight` (layering). A dedicated `internal/refresh` is importable by both `cmd/fab` (the `refresh`/`advance`/`finish` commands) and `internal/preflight`, and keeps the recompute concern cohesive. Folding into `internal/status` was rejected: it composes `score` + `hooklib`, and `internal/status` importing `score` risks a cycle (`score` imports `status`). Reused helpers (`hooklib.InferChangeType`/`HasSectionHeading`/`CountSection*`, `score.*`, `status.ApplyChangeType`/`ApplyAcceptance`, the `statusfile.SourceExplicit` machinery) stay where they live — they retain other live callers.
*Introduced by*: 260702-y022-status-refresh-drop-artifact-hook
