# fab skill bundle

Usage briefing for an agent operating an installed `fab` from any repo. `fab` drives a
six-stage, spec-first change pipeline (intake → apply → review → hydrate → ship → review-pr)
plus workspace, batch, and multi-agent coordination tooling. Plain-markdown prompts, no SDK.

This is the offline, version-locked companion to `fab -h` (flag reference) and the
[shll.ai command tree](https://shll.ai/tools/fab-kit/commands/): `-h` tells you a command's
shape; this tells you *which* command to reach for and how the pieces compose.

## When to use fab

- Reach for fab when a repo has a `fab/` directory (a fab-managed project) and you are
  moving a change through the pipeline, querying its state, or coordinating agents.
- Reach for it for the query commands (`resolve`, `preflight`, the `status` read-only
  queries, `impact`, `pane map --json`) whenever you need machine-readable state.
- Do **not** use fab to author code directly — it orchestrates *you* (the agent) through a
  workflow; the actual editing is still your job, driven by the skills fab deploys.
- Do **not** invoke fab in a repo with no `fab/` directory for project-state commands: they
  fail closed with `ERROR: fab/ directory not found`. Config-free commands still work.

## Capabilities map

One line per capability, keyed to its command:

- **Change lifecycle** — `fab change {new,switch,rename,list,archive,restore,…}` creates and
  moves changes; `fab preflight [<change>]` validates init and resolves the active change to
  a YAML state block (id, stage, progress, plan, confidence).
- **State machine** — `fab status {finish,advance,start,reset,skip,fail,refresh,set-*,…}`
  drives `.status.yaml`; its read-only queries (`progress-map`, `confidence`, `plan`, …)
  take `--json`. `refresh` recomputes artifact-derived fields (pull-based, no hooks).
- **Confidence** — `fab score [--check-gate] [--stage intake] <change>` computes the SRAD
  gate score from `intake.md`.
- **Resolution** — `fab resolve [--id|--folder|--dir|--status|--pane] [<change>]` converts a
  change reference to canonical output; `fab resolve-agent <stage>` resolves the per-stage
  model/effort/dispatch profile.
- **Dispatch** — `fab dispatch {start,status,logs,kill,clean}` runs a stage as a detached,
  tmux-independent worker (the cross-harness CLI adapter).
- **Panes / operator** — `fab pane {map,capture,send,process,window-name}` inspects and
  drives tmux panes; `fab operator` launches the coordination tab.
- **Config** — `fab config {reference,show,init,upgrade}` reads and reconciles
  `config.yaml`; `show --origin` gives per-field provenance.
- **Memory** — `fab memory-index [--check [--json]]` regenerates `docs/memory/` indexes and
  per-folder logs deterministically (never hand-edit them).
- **Batch** — `fab batch {new,switch,archive}` fans changes out across worktree tmux tabs.
- **Workspace lifecycle** — `fab {init,sync,upgrade-repo,update,doctor,migrations-status}`
  (these route to the `fab-kit` binary — see Gotchas).
- **Introspection** — `fab kit-path`, `fab impact <base> <head>`, `fab fab-help`,
  `fab shell-init <shell>`, and this bundle via `fab skill`.

## Composition patterns

fab is one member of the [shll toolkit](https://shll.ai) and composes with its siblings
(toolkit principle: tools shell out to tools):

- **`wt`** (worktrees) — `fab batch new`/`switch` shell out to `wt create` to spin up a
  worktree per change before opening its tmux tab. Add `eval "$(wt shell-init)"` to your shell
  profile once so `cd`-on-open works.
- **`idea`** (backlog) — `fab batch new` reads backlog items; `fab batch archive` (and the
  `/fab-archive` skill) mark the matching backlog entry done.
- **`gh`** (GitHub CLI) — the ship/review-pr stages use `gh` for PR creation and comment
  triage. Authenticate with `gh auth login` first.
- **`rk` (run-kit)** — fab is a pure *consumer* of run-kit's `@rk_agent_state` tmux
  pane-option convention (written by `rk agent-setup`); `fab pane` reads it to gate `send`.
- **The `/fab-*` skills** — `fab sync` deploys markdown skills (`/fab-new`, `/fab-continue`,
  `/fab-ff`, `/fab-fff`, `/git-pr`, …) into the repo's agent directories. Those skills are
  how a harness actually drives the pipeline; the `fab` binary is their engine.

Typical flow: `/fab-new <desc>` → `/fab-continue` (repeat per stage) or `/fab-fff` (run the
whole pipeline gated on the single intake confidence gate).

## Output & exit-code contracts

- **stdout is data.** Command output on stdout is meant to be consumed (parsed, piped);
  diagnostics and progress go to stderr. A command that succeeds writes only its data.
- **`--json`** is available on the machine-readable query surfaces — the `fab status`
  read-only queries (all but `progress-line`), `fab pane map`, `fab config reference`,
  `fab dispatch status`, and `fab memory-index --check` — with additive, stable keys.
  `fab preflight` and `fab resolve` emit YAML/plain text and take no JSON flag.
- **Exit codes** follow the toolkit convention for `fab-go` commands: **`0`** success,
  **`1`** operational failure (missing change, failed preflight, below-gate, tmux/gh/fs
  error), **`2`** usage error (unknown/malformed flag, arg-count violation, unknown
  subcommand, mutually-exclusive flags). Classification is by execution phase, not message
  text — a malformed invocation never reaches its handler.
- **Special in-handler codes coexist** (not renumbered): `fab pane` verbs use `2` = pane
  missing, `3` = other tmux failure; `fab memory-index --check` uses `0`/`1`/`2` with
  `2` = destructive loss; `fab sync` and `fab migrations-status` use `3` = "not a
  fab-managed repo" (a branchable "not applicable here", distinct from the generic `1`).

## Gotchas

- **`fab` is two binaries.** The router forwards `init`, `upgrade-repo`, `sync`, `update`,
  `doctor`, and `migrations-status` to the `fab-kit` workspace binary; everything else goes
  to the version-pinned `fab-go` engine. You never call `fab-kit`/`fab-go` directly.
- **`.claude/skills/` (and `.agents/`, `.opencode/`) are deployed copies.** `fab sync`
  overwrites them from the kit. Never hand-edit a deployed skill — your edit is lost on the
  next sync. In the fab-kit repo itself, the canonical source is `src/kit/skills/`.
- **`fab skill` ≠ fab's kit-skills.** *This* command (the toolkit-standard bundle you are
  reading) is unrelated to fab's own "skills" — the `/fab-*` markdown prompts `fab sync`
  deploys. Same word, two concepts: `fab skill` prints this one static page; the kit-skills
  are the many pipeline prompts.
- **`<change>` is flexible everywhere.** Any command taking a change accepts a 4-char ID
  (`yobi`), a folder substring (`fix-kit`), or the full folder name — not a bare path.
- **Skills go stale after an upgrade.** After `brew upgrade fab-kit`, run `fab sync` (or
  `fab upgrade-repo`) so deployed copies match the new engine; preflight warns on skew.
- **This bundle is static.** `fab skill` prints byte-identical bytes on every machine for a
  release — no environment or session state. For live, environment-derived context use the
  dedicated query commands above (`fab preflight`, `fab pane map`), not this page.
