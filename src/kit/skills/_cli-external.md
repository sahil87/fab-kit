---
name: _cli-external
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), hop (multi-repo navigator), tmux, rk (run-kit), and /loop. Carries only fab-owned content (operator spawning choreography, the escalation rk-notify usage plus pointers to the operator's startup role self-mark and the rk-mux agent-messaging and pane peek/kill/process usage, the tmux/pane and /loop notes); each owned tool's usage knowledge is delegated to `<tool> skill` at use-time (`command -v`-gated fail-silent for all four owned binaries, with a version-skew fallback to the shll.ai bundle page), and its exhaustive command tree to `<tool> help-dump`. Loaded by operator skills only."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# External CLI Tool Reference

> Loaded by operator skills only (not part of the always-load layer). Documents non-fab CLI tools used for multi-agent coordination.

## Contents

- Reference Model
- wt (Worktree Manager)
- idea (Backlog Manager)
- hop (Multi-Repo Navigator)
- tmux
- rk (run-kit)
- /loop

---

## Reference Model

This file documents only **fab-owned** content — what each tool *is* in one line,
and the fab-specific integration choreography that no tool's own documentation
carries (the operator's spawning sequence, the escalation `rk notify` usage and
the pointer to the operator's startup role self-mark, the pane-substrate
delegation notes — skill-facing capture/kill/process ride rk's `mux` twins,
with fab's copies dispatch-internal). It deliberately does **not** restate any
tool-owned usage knowledge: that is delegated to each owned tool's own bundle at
use-time, so this file never goes stale against a tool's release cadence.

Each owned tool exposes two version-locked surfaces: `<tool> skill` for its usage
briefing and `<tool> help-dump` for its exhaustive command tree and flags.

### Delegation and binary gate

`wt`, `idea`, `rk`, and `hop` are optional sibling binaries. Gate every
informational delegation on `command -v` and fail silently when the binary is
absent:

```sh
command -v wt   >/dev/null 2>&1 && wt skill        # gated, fail silently
command -v idea >/dev/null 2>&1 && idea skill
command -v rk   >/dev/null 2>&1 && rk skill
command -v hop  >/dev/null 2>&1 && hop skill
```

Per `shll standards skill`, `<tool> skill` prints a static, ≤150-line,
agent-optimized usage briefing as raw markdown to stdout (exit 0, stderr empty),
byte-identical to the tool repo's canonical `docs/site/skill.md`.

**Version-skew fallback (required).** An installed tool may predate its `skill`
subcommand. The invocation MUST **capability-probe** it — `<tool> skill` failing
(non-zero exit, or no output) is the probe — and fall back **silently** to the
shll.ai bundle-page pointer `https://shll.ai/<tool>/skill`; operator context
loading MUST NOT break or surface an error on an older binary. This composes with
the `command -v` gate: **absent** binary → skip entirely; **present but old** →
the fallback pointer. The retained fab-owned choreography already covers the
operator-critical `wt` semantics, so the fallback never needs to reproduce a tool
gist.

### The `help-dump` contract

For any specific flag or subcommand, run the gated `<tool> help-dump` (or
`<tool> <cmd> --help`) and treat that output as authoritative. The JSON envelope
contains `tool`, `version`, `schema_version`, and a recursive `root` command tree;
it does not contain `captured_at`.

```json
{
  "tool": "idea",
  "version": "v0.0.13",
  "schema_version": 1,
  "root": {
    "name": "idea",
    "path": "idea",
    "short": "Backlog idea management (current worktree; use --main for main worktree)",
    "usage": "idea [flags]",
    "text": "...full help text...",
    "commands": [
      { "name": "add", "path": "idea add", "short": "...", "usage": "idea add <text> [flags]", "text": "...", "commands": [] }
    ]
  }
}
```

### Functional entry points

The fail-silent rule governs informational delegations. `wt` remains
**functionally required** for worktree-based flows — the operator's
spawn-in-worktree sequence (`fab-operator.md` §2 wt Gate) and `fab batch
new`/`switch` (an upfront `exec.LookPath("wt")` guard in the binary). Those entry
points do NOT silently skip: they stop with an actionable install hint
(`… install it via: brew install sahil87/tap/wt`).

---

## wt (Worktree Manager)

`wt` manages git worktrees for parallel development. Install it with
`brew install sahil87/tap/wt`; required worktree entry points stop with that hint
when it is absent (per § Functional entry points).

> `wt`'s command set (`list`/`create`/`delete`/…), the `wt create` flags
> (`--non-interactive`/`--worktree-name`/`--reuse`/`--base`/`--checkout` + the
> positional `[branch]`), and its branch-selection contract (positional is
> new-branch-only, exit 2 on an existing branch; `--checkout <branch>` is the
> existing-branch opt-in and conflicts with both `--base` and the positional) are
> **tool-owned** — read them at use-time via `wt skill` (usage) / `wt help-dump`
> (flags), gated per § Delegation and binary gate. What stays below is
> **fab-owned**: how the operator drives `wt create` for spawning, and which wt
> form the fab routing rule selects when (that decision is fab's).

> **Repo-targeted spawning (operator).** `wt` operates on the **current working directory's** repo. For multi-repo coordination, the operator MUST run `wt create` **in the target repo's directory** (the agent's absolute main-worktree root), so the new worktree lands under `$(dirname <target-repo>)/<repo-name>.worktrees/` — not under the operator's own repo. Composing the session command is a separate step with its own `--repo` targeting rule — see `_cli-agents.md` § Spawn Composition (and `fab-operator.md` §6 step 5 for the operator's always-pass-`--repo` policy).

### Operator Spawning Rules

When the operator creates a worktree for an agent, the naming strategy depends on whether the change already exists:

#### Known change (already exists)

The change's branch usually already exists (created by `/fab-new` Step 11 in the original checkout), so **probe branch existence and route** — the existing branch takes `--checkout`, a missing one the positional (wt's positional is new-branch-only; the exact contract is in `wt skill`, per § Reference Model). Probe local first (`git show-ref --verify --quiet refs/heads/<change-folder-name>`), then remote (`git ls-remote --heads origin <change-folder-name>`):

```
# branch exists (the common case) → put the worktree ON the existing branch
wt create --non-interactive --worktree-name <name> --checkout <change-folder-name>

# branch missing → create it (new-branch positional)
wt create --non-interactive --worktree-name <name> <change-folder-name>
```

The worktree gets a random name; the branch matches the change. No `/git-branch` needed.

#### New change (from backlog)

The change folder doesn't exist yet, so there's no branch name to use:

1. `wt create --non-interactive` — auto-generates worktree name, creates on default branch
2. Agent runs `/fab-new` to create the change folder — its Step 11 then renames the worktree's disposable branch to the change folder name inline (the rename guard passes: the `wt create` branch resolves to no change)
3. No operator action needed — the branch already matches the change; the operator does NOT send `/git-branch`

---

## idea (Backlog Manager)

Standalone binary for backlog idea management — CRUD for `fab/backlog.md` (the
inbox that feeds `/fab-new <id>`). Install it with
`brew install sahil87/tap/idea`; `/fab-new <id>` can resolve backlog IDs directly
when it is absent.

`idea`'s verbs, flags, matching rules, and backlog line format are tool-owned;
delegate per § Delegation and binary gate.

---

## hop (Multi-Repo Navigator)

`hop` is the **repo locator** for the same space `wt` operates on: `wt`
enumerates worktrees within a repo, while `hop` enumerates registered repos.
Its discovery grammar is tool-owned; delegate per § Delegation and binary gate.

**Why it matters to the operator (fab-owned).** Multi-repo coordination needs the absolute main-worktree root of a *sibling* repo — e.g. to spawn an agent into it (see the **Repo-targeted spawning** note in the `wt` section, which requires running `wt create` in the target repo's directory and reading `fab agent --print --repo <target-repo>`). `hop` is how an agent **discovers** those locations rather than hardcoding paths; the specific discovery command is in `hop skill`.

---

## tmux

Terminal multiplexer commands used by the operator for agent observation and interaction.

### Commands

| Command | Usage | Purpose |
|---------|-------|---------|
| `new-window` | `tmux new-window -n <name> -c <dir> "<cmd>"` | Open a new tmux tab with a command running in a specific directory |

### Usage Notes

- **Pane mapping across sessions**: The operator's tick snapshots **all** sessions on its tmux server via `fab pane map --all-sessions --json` (see `_cli-fab.md`), not just the operator's own session. The `--json` output carries a per-row `repo` field (the pane's absolute main-worktree root, `null` when unresolved) used to group the status frame by repo then session.
- **Pane capture**: Prefer `rk mux capture` when rk is present (`command -v rk`-gated; substrate-enriched capture — last-N tail, `--raw`/`--json`, reconciled agent state). When rk is absent, fail open to raw `tmux capture-pane` — never an error. Usage ownership is in `_cli-agents.md` § Peek. (`fab pane capture` is dispatch-internal — kept for the rk-less pane arm; see `_cli-fab.md` § fab pane.)
- **Send keys**: Prefer `rk mux send` when rk is present (`command -v rk`-gated; it carries built-in pane-existence and agent-state validation with probe-verified delivery). When rk is absent, fail open to raw `tmux send-keys` behind the caller's own state read (`fab pane map`) plus the manual delivery probe — never an error. Usage ownership for both paths is in `_cli-agents.md` § Pre-Send Validation and `fab-operator.md` §3/§5.
- **`new-window`** is also how an agent session is spawned — the command form, quoting, and the one-prompt/no-`&&`-chaining rule are owned by `_cli-agents.md` § Spawn Composition ("Open it in a pane"); the operator's `»<wt>` window-marker name is its own policy, in `fab-operator.md` §6

---

## rk (run-kit)

run-kit is the tmux session manager with a web UI that may host the operator's
session. The Homebrew formula and primary binary are `run-kit`
(`sahil87/tap/run-kit`); `rk` is the invocation used throughout fab skills.

`rk`'s command surface is tool-owned; delegate per § Delegation and binary gate.
This includes `rk notify`, `rk context`, iframe windows, proxy URLs, and the
visual display recipe.

The **dynamic** environment (current server URL, session, pane) stays in `rk context` — run at use-time, never hardcoded. `rk skill` is the static usage briefing; `rk context` reports the live environment (the two are distinct per `shll standards skill`).

### Operator escalation send (fab-owned)

The operator's non-blocking Strategic escalation (`fab-operator.md` §5) uses `rk notify` as its default out-of-band notification send — the fab-specific usage (message/title template), gated on `command -v rk` and relying on run-kit's fail-silent-by-contract guarantee:

```sh
command -v rk >/dev/null 2>&1 && rk notify "{change}: {summary} ({repo})" --title "Operator: strategic question"
```

This is the operator's *usage* of the tool, not the `rk notify` contract itself (that is tool-owned — see `rk skill`). When `rk` is absent, the operator falls back to a documented alternative channel per `fab-operator.md` §5 Notification Send.

### Operator role self-mark (fab-owned — pointer)

The second fab-owned rk usage — the fail-silent `rk role operator` self-mark that pins the operator window in run-kit's dashboard — is owned by `fab-operator.md` §2 Startup (§ Role Mark). The `@rk_role` option contract and its one-operator-per-server radio semantics are tool-owned; see `rk skill`.

### Agent messaging (fab-owned — pointer)

The third fab-owned rk usage — agent messaging via `rk mux send`/`rk mux await`, `command -v rk`-gated and fail-open to the raw-tmux path when rk is absent — is owned by `_cli-agents.md` § Pre-Send Validation / § Await and `fab-operator.md` §3/§5. The verbs' full contract (gate matrix, probe-verified delivery, report words) is tool-owned; see `rk skill`.

### Pane peek/kill/process (fab-owned — pointer)

The fourth fab-owned rk usage — pane peek via `rk mux capture`, pane removal via the agent-state-gated `rk mux kill`, and process-tree inspection via `rk mux process`, each `command -v rk`-gated and fail-open to raw tmux when rk is absent — is owned by `_cli-agents.md` § Peek (with the operator's capture form in `fab-operator.md` §5). The verbs' full contracts are tool-owned; see `rk skill`. fab's own `fab pane capture`/`kill`/`process` remain dispatch-internal for the rk-less pane arm (`_cli-fab.md` § fab pane).

---

## /loop

Recurring check skill — invokes a prompt at a regular interval.

### Usage

```
/loop <interval> "<prompt>"
```

- **`<interval>`** — duration between ticks (e.g., `5m`, `2m`)
- **`<prompt>`** — the instruction to execute on each tick

### Constraints

- **One loop at a time** — there SHALL be at most one active `/loop` in a session
- **Start**: when the first change is enrolled in monitoring and no loop is running
- **Stop**: when the monitored set becomes empty, or on explicit user command
- **Autopilot override**: autopilot uses its own cadence (default 2m); replaces any existing monitoring loop
