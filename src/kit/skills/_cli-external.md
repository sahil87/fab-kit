---
name: _cli-external
description: "External CLI tool reference — wt (worktree manager), idea (backlog manager), hop (multi-repo navigator), tmux, rk (run-kit), and /loop. Carries only fab-owned content (operator spawning choreography, the escalation rk-notify usage, the tmux/pane and /loop notes); each owned tool's usage knowledge is delegated to `<tool> skill` at use-time (bare for wt/idea, `command -v`-gated fail-silent for rk/hop, with a version-skew fallback to the shll.ai bundle page), and its exhaustive command tree to `<tool> help-dump`. Loaded by operator skills only."
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
carries (the operator's spawning sequence, the escalation `rk notify` usage, the
`fab pane` internalization notes). It deliberately does **not** restate any
tool-owned usage knowledge: that is delegated to each owned tool's own bundle at
use-time, so this file never goes stale against a tool's release cadence.

Each owned tool serves two use-time surfaces, and this file delegates to both:

- **`<tool> skill`** — the tool's **usage briefing** (when to reach for it, its
  capabilities map, composition patterns, gotchas). Use it for a tool's usage
  knowledge beyond the fab-owned content retained here.
- **`<tool> help-dump`** — the tool's **exhaustive command tree** as JSON. Use it
  for any specific flag or subcommand.

Both surfaces are **version-locked by construction** (embedded in the same binary
as the flags they describe), so neither can document a capability the installed
binary lacks.

### The `skill` delegation (usage knowledge)

For any owned tool's usage knowledge beyond the fab-owned content retained in this
file, run `<tool> skill` at use-time — **bare** for `wt`/`idea` (assumed-present),
**`command -v`-gated fail-silent** for `rk`/`hop` (genuinely-optional), per the
absent-binary discipline below:

```sh
wt skill                                          # wt/idea: assumed present, bare
idea skill
command -v rk  >/dev/null 2>&1 && rk skill        # rk/hop: gated, fail silently
command -v hop >/dev/null 2>&1 && hop skill
```

Per `shll standards skill`, `<tool> skill` prints a static, ≤150-line,
agent-optimized usage briefing as raw markdown to stdout (exit 0, stderr empty),
byte-identical to the tool repo's canonical `docs/site/skill.md`.

**Version-skew fallback (required).** An installed tool may predate its `skill`
subcommand. The invocation MUST **capability-probe** it — `<tool> skill` failing
(non-zero exit, or no output) is the probe — and fall back **silently** to the
shll.ai bundle-page pointer `https://shll.ai/<tool>/skill`; operator context
loading MUST NOT break or surface an error on an older binary. For `rk`/`hop` this
composes with the `command -v` gate: **absent** binary → skip entirely; **present
but old** → the fallback pointer. The retained fab-owned choreography already
covers the operator-critical `wt` semantics, so the fallback never needs to
reproduce a tool gist.

> The `skill` delegation scopes to the **four owned binaries** (`wt`, `idea`,
> `rk`, `hop`) — the same scope as `help-dump` below. `tmux` is third-party and
> has no `skill` bundle; `/loop` is a Claude Code skill, not a binary — neither
> is covered.

### The `help-dump` contract

Each owned tool — `wt`, `idea`, `rk`, `hop` (and `fab` itself) — exposes a **hidden
but stable** `help-dump` subcommand that emits its entire command tree as JSON. It
is not listed in `--help`, but the shape is stable across all of them:

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

Fields: `tool`, `version`, `schema_version` (currently `1`), and a recursive
`root` → `commands[]` tree where each node carries `name`, `path`, `short`,
`usage`, and `text`. The tree is recursive — `commands[]` nests for subcommand
groups. Per the shll v0.0.23 help-dump standard the envelope carries **no
`captured_at`**: the capture timestamp is owned by shll.ai's puller (a tool
cannot know its own capture time — it is stamped after capture), so emitting it
is forbidden toolkit-wide. `fab` and `wt` already omit it; any peer tool still
emitting `captured_at` (empty or otherwise) drops it on its own release cadence.

**Use it at use-time.** For any specific flag or subcommand, run `<tool> help-dump`
(or `<tool> <cmd> --help`) and treat *that*, not this file, as authoritative for
the exhaustive surface. `help-dump` is the command-tree sibling of `skill` above
(usage knowledge) — this file inlines neither; both are delegated at use-time.

> The `help-dump` delegation scopes to the same **four owned binaries** (`wt`,
> `idea`, `rk`, `hop`) as the `skill` delegation. `tmux` is third-party and has no
> `help-dump`; `/loop` is a Claude Code skill, not a binary — neither is covered.

### Absent-binary discipline (two install classes)

The four owned binaries fall into two classes by install guarantee, and the
fail-silent rule applies **asymmetrically**:

- **Assumed-present — `wt`, `idea`.** These are Homebrew `depends_on` of `fab-kit`
  (they land together via `brew install fab-kit`). Invoke them **bare**; no
  `command -v` gate is required.
- **Genuinely-optional — `rk`, `hop`.** Each is a separate sibling formula the user
  may or may not have installed (`rk` is run-kit — formula `sahil87/tap/run-kit`
  since run-kit v3.0.0, with `rk` kept as a symlink alias; `hop` is the multi-repo
  navigator). **Every `rk`/`hop` invocation — including `skill` and `help-dump` —
  MUST be `command -v`-gated and fail silently** (never surface `command not found`
  or any error/warning when the tool is absent). Do NOT generalize this gate to
  `wt`/`idea`.

This discipline governs **both** use-time delegations (`skill` and `help-dump`)
identically — each is a per-tool invocation:

```sh
command -v hop >/dev/null 2>&1 && hop skill        # rk/hop: gated, fail silently
command -v rk  >/dev/null 2>&1 && rk help-dump     # rk/hop: gated, fail silently
wt skill                                            # wt/idea: assumed present, bare
idea help-dump                                      # wt/idea: assumed present, bare
```

---

## wt (Worktree Manager)

`wt` manages git worktrees for parallel development. Installed system-wide via `brew install fab-kit`.

> `wt`'s command set (`list`/`create`/`delete`/…), the `wt create` flags
> (`--non-interactive`/`--worktree-name`/`--reuse`/`--base`/`--checkout` + the
> positional `[branch]`), and its branch-selection contract (positional is
> new-branch-only, exit 2 on an existing branch; `--checkout <branch>` is the
> existing-branch opt-in and conflicts with both `--base` and the positional) are
> **tool-owned** — read them at use-time via `wt skill` (usage) / `wt help-dump`
> (flags), assumed present — bare, per § Reference Model. What stays below is
> **fab-owned**: how the operator drives `wt create` for spawning, and which wt
> form the fab routing rule selects when (that decision is fab's).

> **Repo-targeted spawning (operator).** `wt` operates on the **current working directory's** repo. For multi-repo coordination, the operator MUST run `wt create` **in the target repo's directory** (the agent's absolute main-worktree root), so the new worktree lands under `$(dirname <target-repo>)/<repo-name>.worktrees/` — not under the operator's own repo. The operator reads that target repo's session command separately via `fab agent --print --repo <target-repo>` (see `_cli-fab.md`), never its own `config.yaml`.

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
3. No operator action needed — the branch already matches the change; the operator does NOT send `/git-branch` (the former post-intake send predates fab-new's inline branch creation)

---

## idea (Backlog Manager)

Standalone binary for backlog idea management — CRUD for `fab/backlog.md` (the inbox that feeds `/fab-new <id>`). Installed system-wide via `brew install fab-kit` (not a `fab` subcommand).

`idea`'s verbs (`add`/`list`/`show`/`done`/`reopen`/`edit`/`rm` + bare-text shorthand), its persistent flags (`--file`/`--main` + worktree-vs-main-backlog resolution), its query-matching rule, and the `fab/backlog.md` line format are all **tool-owned** — read them at use-time via `idea skill` (usage) / `idea help-dump` (flags), assumed present — bare, per § Reference Model.

---

## hop (Multi-Repo Navigator)

`hop` is a **genuinely-optional** binary — a separate sibling formula, not a `fab-kit` Homebrew dependency, so it can legitimately be absent. Every `hop` invocation MUST be `command -v hop`-gated and skip silently when absent (per § Reference Model — never surface `command not found` or any error/warning). This mirrors the `rk` fail-silent discipline.

`hop` is the **repo locator** — the discovery front-end to the same repo/worktree space `wt` operates on: where `wt` enumerates the worktrees *within* a repo, `hop` enumerates the *repos* themselves (registered in `~/.config/hop/hop.yaml`). Its discovery commands (`ls`/`ls --trees`/`where`) and grammar are **tool-owned** — read them at use-time via the gated delegation:

```sh
command -v hop >/dev/null 2>&1 && hop skill   # usage briefing; gated, fail silently
```

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
- **Pane capture**: Use `fab pane capture` instead of raw `tmux capture-pane`. It provides fab context enrichment, validation, and structured output.
- **Send keys**: Use `fab pane send` instead of raw `tmux send-keys`. It includes built-in pane existence and agent idle validation.
- **`new-window`** is used for spawning new agent sessions: `tmux new-window -n "»<wt>" -c <worktree> "$SPAWN_CMD '<command>'"` where `<wt>` is the worktree name and `$SPAWN_CMD` is the target repo's session command (see the repo-targeted spawning note in the wt section above)

---

## rk (run-kit)

run-kit is the tmux session manager with a web UI that may host the operator's session. Since run-kit v3.0.0 the Homebrew formula and primary binary are named `run-kit` (`sahil87/tap/run-kit`); `rk` is kept as a symlink alias and remains the invocation form used throughout fab skills. All `rk` usage is subject to the **detection / fail-silent rule** stated once in `_preamble.md` § Run-Kit (rk) Reference — check `command -v rk` first and skip silently when rk is absent (never error, never warn).

`rk`'s command surface is **tool-owned** — the `rk notify` contract (Web Push delivery, fail-silent-by-contract guarantee), `rk context` (server-URL discovery, iframe windows via `@rk_type`/`@rk_url`, the `/proxy/{port}/` pattern, the Visual Display Recipe + visual-explainer integration), and the rest (`daemon`/`doctor`/`serve`/…). Read it at use-time via the gated delegation:

```sh
command -v rk >/dev/null 2>&1 && rk skill   # usage briefing; gated, fail silently
```

The **dynamic** environment (current server URL, session, pane) stays in `rk context` — run at use-time, never hardcoded. `rk skill` is the static usage briefing; `rk context` reports the live environment (the two are distinct per `shll standards skill`).

### Operator escalation send (fab-owned)

The operator's non-blocking Strategic escalation (`fab-operator.md` §5) uses `rk notify` as its default out-of-band notification send — the fab-specific usage (message/title template), gated on `command -v rk` and relying on run-kit's fail-silent-by-contract guarantee:

```sh
command -v rk >/dev/null 2>&1 && rk notify "{change}: {summary} ({repo})" --title "Operator: strategic question"
```

This is the operator's *usage* of the tool, not the `rk notify` contract itself (that is tool-owned — see `rk skill`). When `rk` is absent, the operator falls back to a documented alternative channel per `fab-operator.md` §5 Notification Send.

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
