# Intake: Operator runs on the doing-tier model and drops its hard git-repo dependency

**Change**: 260613-2sdj-operator-doing-tier-no-git-dep
**Created**: 2026-06-13

## Origin

> Operator runs on the doing-tier model and drops its hard git-repo dependency.

Discovered while running `fab operator` from a neutral parent directory (`~/code/sahil87`,
outside any git repo). It failed with:

```
cannot determine repo root: fatal: not a git repository
```

Investigation showed the git dependency is **incidental**, not essential. A separate
just-merged change (`260613-l3ja`, PR #406) added a per-stage model **tier** system
(`internal/agent`, `fab resolve-agent <stage>`, `agent.tiers` config) that the operator
should consume so it launches its coordinating agent on a deliberately-chosen model.

This intake was created by `/fab-proceed` in **promptless autonomous dispatch**. All design
decisions were settled in a live design conversation and are recorded below as Confident/
Certain assumptions — there were no must-ask (Unresolved) decisions to defer.

## Why

**Problem 1 — Incidental git dependency.** `fab operator` (`src/go/fab/cmd/fab/operator.go`,
`runOperator`) hard-fails when not inside a git repo. The repo root it resolves via
`gitRepoRoot()` (`git rev-parse --show-toplevel`) is used in **exactly one place**: as the
`-c` (working directory) argument to `tmux new-window` (`operator.go` line ~78). It is **not**
part of the operator's state-file path — state is keyed by the tmux **socket path** under
`XDG_STATE_HOME` (`StatePath`/`serverSlug`), not by repo. The operator is a per-tmux-server
**singleton** designed for **cross-repo** coordination (per the multi-repo design), so its
natural launch point is a neutral parent directory with no git and no `fab/` project. Forcing
it into one repo root contradicts that intent, and the consequence is that the operator simply
cannot be launched from its natural home directory today.

**Problem 2 — Operator should run on a deliberately-chosen model.** The just-merged `l3ja`
change introduced three model tiers — `thinking` / `doing` / `ship` — each a `{model, effort}`
profile. Resolution is two-layer, per-field merged: (1) project override via
`agent.tiers.<tier>` in `fab/project/config.yaml`, falling back to (2) fab-kit's built-in
default. The `doing` tier default is `{claude-opus-4-8, high}`. The operator is
**execution-that-coordinates** work, so it should launch its agent on the **doing** tier
`{model, effort}`. Today it inherits whatever model the spawn command happens to specify, with
no deliberate tier choice.

**Why this approach.** Both fixes are minimal and align the operator with existing, just-shipped
infrastructure rather than inventing new mechanism. The git fix degrades gracefully
(`os.Getwd()` fallback) so behavior inside a repo is unchanged. The model fix reuses
`fab resolve-agent` (the canonical tier-resolution surface) plus the in-process
`agent.DefaultTier` fallback, so the operator picks up project overrides for free.

## What Changes

### 1. Git-repo dependency → graceful `os.Getwd()` fallback (`operator.go`)

In `runOperator`, the working-directory resolution becomes non-fatal. Try `gitRepoRoot()`;
**on failure, fall back to `os.Getwd()` instead of erroring.** The new tmux window always gets a
sensible `-c <dir>`:

- Inside a repo → preserves today's "start in repo root" behavior.
- Outside a repo → degrades to "start where I am" (the neutral parent directory).

`tmux new-window` still receives a `-c <dir>` argument in both cases. The hard error
`cannot determine repo root: ...` is removed.

```go
// before (operator.go ~line 63):
repoRoot, err := gitRepoRoot()
if err != nil {
    return fmt.Errorf("cannot determine repo root: %w", err)
}
// ... used only at line ~78:
pane.RunCmd("tmux", "new-window", "-c", repoRoot, "-n", tabName, shellCmd)

// after: try gitRepoRoot(), fall back to os.Getwd() on failure (no hard error).
windowDir, err := gitRepoRoot()
if err != nil {
    windowDir, err = os.Getwd()
    // handle os.Getwd error path appropriately
}
// ... -c windowDir
```

### 2. Operator launches its agent on the `doing` tier (`operator.go`)

The operator obtains the `doing`-tier `{model, effort}` by shelling
`fab resolve-agent apply`. The `apply` stage is the **canonical member of the doing tier**
(`apply → doing` in the fab-owned, fixed stage→tier mapping). Parse the byte-stable stdout
lines `model=<id>` and optional `effort=<level>`.

**On ANY failure** — no fab project resolvable (`fab resolve-agent` itself fails outside a fab
project with "fab/ directory not found"), or a parse error — **fall back to fab-kit's built-in
doing default**, available in-process via `agent.DefaultTier(agent.TierDoing)` =
`{claude-opus-4-8, high}`.

A **prominent code comment at the call site MUST document WHY `apply` is probed** (it is the
canonical doing-tier stage) to flag the coupling to the internal, fab-owned stage→tier mapping,
so a future remapping surfaces this dependency.

The parse-or-default logic MUST be extracted into a **pure, testable function** (the live
shell-out is not unit-testable, but the parse + fallback is). Example shape:

```go
// resolveDoingProfile parses `fab resolve-agent apply` stdout (lines `model=<id>` and
// optional `effort=<level>`) into a Profile, falling back to the built-in doing default
// on any parse failure or empty output. Pure function — caller does the shell-out.
func resolveDoingProfile(stdout string) agent.Profile { ... }
```

### 3. Inject `--model`/`--effort` into the spawn command via a new `spawn.WithProfile` helper

The operator spawns a fresh `claude` CLI in the new tmux window via the configured
`agent.spawn_command` (e.g. `claude --dangerously-skip-permissions --effort xhigh -n "..."`).
**Append BOTH `--model <doing-model>` and `--effort <doing-effort>` to the END of the spawn
command**, so they are the **last** occurrences of those flags (verified: the `claude` CLI
accepts duplicate `--effort` without a parse error, and last-wins).

Each flag is **OMITTED when its resolved value is empty** (mirrors the `empty ⇒ omit` convention
in `_preamble.md` § Per-Stage Model Resolution: empty model ⇒ omit `--model`/inherit; empty
effort ⇒ omit `--effort`).

Implement as a new reusable, unit-testable helper in `internal/spawn`:

```go
// WithProfile appends --model/--effort to the END of spawnCmd (last-wins), omitting
// each flag when its value is empty.
func WithProfile(spawnCmd, model, effort string) string { ... }
```

Rationale for a shared helper over inline concatenation: `spawn.Command` is consumed at **4 call
sites** (operator, batch_switch, batch_new, spawn_command), and a shared helper keeps the pattern
reusable and testable.

### 4. Tests (constitution Test Integrity + test-alongside)

- `src/go/fab/internal/spawn/spawn_test.go` — `WithProfile` coverage: both flags present,
  empty model only, empty effort only, both empty.
- `src/go/fab/cmd/fab/operator_test.go` (already exists for `findWindowExact`/`slugify`) —
  coverage for the pure `resolveDoingProfile` parse-or-default function: well-formed
  `model=`+`effort=`, `model=` only (no effort line), empty/garbage output ⇒ built-in doing
  default.

### 5. Docs (constitution REQUIRES these for CLI + skill changes)

- `src/kit/skills/fab-operator.md` — document that the operator launches on the **doing-tier
  model** and **no longer requires a git repo** (can be launched from a neutral parent dir;
  window cwd falls back to the launch dir).
- `docs/specs/skills/SPEC-fab-operator.md` — behavior update mirroring the skill doc.
- `src/kit/skills/_cli-fab.md` — the `## fab operator` section currently says "create one in
  the **repo root**"; update the launch-cwd contract (repo root when inside a repo, else the
  current working directory) and note the doing-tier model injection.

## Affected Memory

- `runtime/operator`: (modify) Operator launch preconditions (no longer requires a git repo;
  window cwd = repo root or `os.Getwd()` fallback) and model selection (launches its
  coordinating agent on the doing tier via `fab resolve-agent apply` with built-in fallback).
- `_shared/configuration`: (modify) The doing-tier consumption by a non-pipeline command — the
  operator is the **first non-orchestrator consumer** of `resolve-agent` / the agent-tier system.

## Impact

**Code:**
- `src/go/fab/internal/spawn/spawn.go` — new `WithProfile(spawnCmd, model, effort) string` helper.
- `src/go/fab/cmd/fab/operator.go` — `runOperator`: (a) git-root → `os.Getwd()` fallback;
  (b) resolve doing model via `fab resolve-agent apply` with `agent.DefaultTier(TierDoing)`
  fallback; (c) inject via `spawn.WithProfile`; (d) extract pure `resolveDoingProfile` parse helper.

**Dependencies (existing, just-merged via `l3ja`/PR #406):**
- `internal/agent`: `agent.DefaultTier`, `agent.TierDoing`, `agent.Profile` (`{Model, Effort}`).
- `cmd/fab/resolve_agent.go`: `fab resolve-agent <stage>` byte-stable `model=`/`effort=` stdout.

**Tests:** `spawn_test.go` (new helper), `operator_test.go` (pure parse-or-default fn).

**Docs:** `src/kit/skills/fab-operator.md`, `docs/specs/skills/SPEC-fab-operator.md`,
`src/kit/skills/_cli-fab.md`.

**Behavioral compatibility:** Inside a git repo, the window cwd is unchanged (still repo root).
The doing-tier flags are appended last-wins, so a spawn_command that already pins a model/effort
is overridden by the deliberate doing-tier choice (intended). No state-file or config-schema
changes — no migration required.

## Open Questions

(none — all design points were settled in the live design conversation and recorded as
Confident/Certain assumptions below.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Window cwd: in `runOperator`, try `gitRepoRoot()`; on failure fall back to `os.Getwd()` instead of erroring (`tmux new-window` still gets `-c <dir>`). | Settled in design conversation; reversible single call site; constitution/codebase support graceful degradation; one obvious interpretation. | S:90 R:80 A:85 D:90 |
| 2 | Confident | Doing-model source: shell `fab resolve-agent apply` (apply = canonical doing-tier stage), parse `model=`/`effort=` lines; on ANY failure fall back to in-process `agent.DefaultTier(agent.TierDoing)` = `{claude-opus-4-8, high}`. | Settled in conversation; reuses just-merged l3ja surface; verified byte-stable contract; clear single approach. | S:90 R:75 A:85 D:85 |
| 3 | Certain | A prominent call-site code comment MUST document WHY `apply` is probed (canonical doing-tier stage) to flag the coupling to the fab-owned stage→tier mapping. | Explicit conversation requirement + constitution Code Quality (no magic strings; document non-obvious coupling). Determined by the design contract. | S:95 R:80 A:90 D:95 |
| 4 | Confident | Inject both `--model <m>` and `--effort <e>` at the END of the spawn command (last-wins; duplicate `--effort` accepted by claude CLI, verified), each OMITTED when its value is empty (mirrors `empty ⇒ omit` convention). | Settled in conversation, empirically verified; reversible; mirrors documented `_preamble` convention; one obvious mapping. | S:90 R:80 A:85 D:90 |
| 5 | Confident | Implement injection as a new reusable `spawn.WithProfile(spawnCmd, model, effort) string` helper in `internal/spawn` (not inline concat). | Conversation decision; `spawn.Command` has 4 call sites so a shared, unit-testable helper matches existing factoring; reversible. | S:90 R:85 A:80 D:85 |
| 6 | Confident | Extract the parse-or-default logic into a pure, testable function (`resolveDoingProfile`); the live shell-out stays in `runOperator` (not unit-testable) but the parse+fallback is. | Conversation decision + constitution Test Integrity / test-alongside; existing `operator_test.go` already tests pure helpers (`findWindowExact`/`slugify`). | S:90 R:85 A:85 D:90 |
| 7 | Certain | Docs required: update `src/kit/skills/fab-operator.md`, `docs/specs/skills/SPEC-fab-operator.md`, and `src/kit/skills/_cli-fab.md` (operator launch-cwd contract is documented there at the `## fab operator` section). | Constitution: CLI changes MUST update `_cli-fab.md`; skill changes MUST update the corresponding `SPEC-*.md`. `_cli-fab.md` operator section confirmed present (says "repo root"). | S:95 R:85 A:95 D:95 |
| 8 | Confident | No migration required: no state-file path or config-schema change; the fix is behavioral within `runOperator` + a new helper. | State is socket-keyed (unchanged); doing-tier consumption reads existing config/defaults. Reversible; codebase confirms no schema touch. | S:90 R:80 A:85 D:85 |

8 assumptions (2 certain, 6 confident, 0 tentative, 0 unresolved).
