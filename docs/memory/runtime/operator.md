---
type: memory
description: "What fab still owns of the operator after its handover to HexoKit: the `operator` agent role resolved by `fab agent operator -o yaml`, the kept spawn-contract surfaces (`fab pane`, `fab dispatch`, `fab status set-base-branch`, `/fab-new` / `/fab-proceed` prompts), and fab's operator state file left on disk for `rk operator adopt`. The `fab operator` verbs, the `/fab-operator` skill (project and user level) and `autopilot.merge_mode` are gone."
---
# Operator

**Domain**: runtime

## Overview

The operator — the long-lived coordination session that tracks agents across tmux panes and repos, answers their prompts, spawns dependency-ordered work, merges, and recovers stuck PRs — is **owned by HexoKit (rk)**: `rk operator` launches it, `rk skill operator` serves its chat skill, `rk operator autopilot` runs the unattended loop, and the tracked-item state, tick, and cron cadence live in rk. fab ships **no operator surface**: there is no `fab operator` command family, no `/fab-operator` skill at project or user level, and no operator configuration key. What fab keeps is the small contract HexoKit's operator builds on.

## Requirements

### Kept Contract Surfaces

| Surface | What the operator uses it for |
|---------|-------------------------------|
| `fab agent operator -o yaml [--headless]` | Resolves the **operator** role (Tier 1, session depth) — provider, model, effort, composed command — so `rk operator` launches the session with the project's configured profile. The role is one of six fixed slots; see [providers-and-profiles.md](/runtime/providers-and-profiles.md) |
| `fab pane map`, `fab pane capture`, `fab pane open/ready/deliver` | Pane observation and provider-generic spawn/deliver primitives; see [pane-commands.md](/runtime/pane-commands.md) and [agent-primitives.md](/runtime/agent-primitives.md) |
| `fab dispatch *` | The stage-dispatch machinery the operator's spawned agents run on; see [dispatch.md](/runtime/dispatch.md) |
| `fab status set-base-branch <change> <branch>` | Records a change's dependency branch (stacked-PR topology) in `.status.yaml`; see [pipeline/schemas.md](/pipeline/schemas.md) |
| `.status.yaml` reads, `/fab-new` and `/fab-proceed` spawn prompts | The per-change state and the prompts the operator hands the agents it spawns |

`fab agent` is documented in the kit's `_cli-fab.md` § fab agent.

### Removed Surfaces

- **`fab operator`** (bare launcher, `track`, `tick-start`, `clock`, `state`, `branch-map`, `time`) is an unknown command. Replacements: `rk operator`, `rk operator track|check`, `rk cron …`, `rk operator autopilot`.
- **`/fab-operator`** is not shipped at any level. `rk skill operator` is the only operator entry point. The `fab-operator` project skill and the user-level pointer template are deleted, so `fab sync` deploys neither; it removes a previously written fab-generated user-level `fab-operator` copy (see [distribution/setup.md](/distribution/setup.md)).
- **`autopilot.merge_mode`** is not a fab config key; rk reads `operator.merge_mode` from its own settings. A leftover `autopilot:` block in a project or system config is ignored silently (an unknown key). The migration note spells out the manual `rk settings set operator.merge_mode <value>` step (see [distribution/migrations.md](/distribution/migrations.md)).

### Operator State File

fab's former operator state file, `$XDG_STATE_HOME/fab/operator/<slug>.yaml`, is **never read, written, or deleted** by any fab command. It stays on disk so `rk operator adopt` can import items it still holds.

## Design Decisions

### Operator Ownership Lives in HexoKit; fab Keeps Only the Contract Surfaces

**Decision**: fab deletes the operator command family, the operator skill, and the operator-only configuration, keeping the `operator` agent role plus the pane / dispatch / base-branch primitives HexoKit's operator calls.
**Why**: Two repos owning one operator made the seam the historical source of breakage; HexoKit shipped the whole replacement (state engine, check, Operator Tasks, chat skill, loop, Go autopilot). `rk operator` launches via `fab agent operator -o yaml`, so the role is a live dependency, while the verbs and skill were dead weight.
**Rejected**: Keeping fab's operator indefinitely as a fallback (two operators alive, the seam the move exists to remove; HexoKit is a hard prerequisite of the operator anyway); keeping a deprecation shim for `fab operator` (a removed command should fail as unknown, not silently delegate).
*Introduced by*: 261117-egu4-delete-fab-operator

### No `/fab-operator` Skill at Any Level

**Decision**: fab ships no `/fab-operator` skill — project or user level. `rk skill operator` is the sole entry point, and `fab sync` removes stale fab-generated user-level copies (marker-matched; anything else at that path is left alone).
**Why**: HexoKit stopped typing `/fab-operator` from `rk operator`, so a pointer skill had no caller left; a stale user-level copy would keep resolving the retired name silently.
**Rejected**: A project-level rk-only pointer (keeps a seam with no consumer); deleting the user-level copies without cleanup (stale pointers keep resolving).
*Introduced by*: 261117-egu4-delete-fab-operator

### `autopilot.merge_mode` Is Retired, Not Migrated by the Binary

**Decision**: The key is removed from every fab surface (config struct, defaults, `fab config explain` / scope tables, validation) and a migration note tells users to run `rk settings set operator.merge_mode <value>` by hand.
**Why**: fab cannot write rk's settings, and a key nothing honours is a trap — users set it and silently get `cherry-pick-ladder`. Existing blocks stay harmless because unknown keys are dropped at load.
**Rejected**: Keeping the key as an ignored alias (the trap persists); a load-time error on a leftover block (breaks every project that ever set it for no behavioural gain).
*Introduced by*: 261117-egu4-delete-fab-operator

### The State File Stays on Disk

**Decision**: No fab code path reads, writes, or deletes the old operator state file.
**Why**: `rk operator adopt` reads it on its own path logic, and leaving it untouched keeps an adoption or rollback possible after the verbs are gone.
**Rejected**: Deleting it at upgrade (loses un-imported items irrecoverably); keeping a read path in fab (retains code whose only consumer left).
*Introduced by*: 261117-egu4-delete-fab-operator
