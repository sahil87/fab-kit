# _cli-external

## Summary

External CLI tool reference — the non-fab command-line tools used for multi-agent coordination: **wt** (worktree manager), **idea** (backlog manager), **hop** (multi-repo navigator), **tmux**, **rk** (run-kit), and **/loop**. As of `260718-clix` it carries only **fab-owned** content — each tool's one-line identity plus the fab-specific integration choreography no tool's own docs carry (the operator spawning sequence, the escalation `rk notify` usage, the `fab pane`/`/loop` notes). Each owned tool's usage knowledge is delegated at use-time to `<tool> skill` (the usage briefing per `shll standards skill` — `command -v`-gated fail-silent for all four owned binaries, with a required version-skew fallback to the shll.ai bundle page `https://shll.ai/<tool>/skill`), and its exhaustive command tree to `<tool> help-dump` — the two siblings this file inlines neither of. Like `_cli-fab.md`, it is a **reference catalog**, not a procedure.

This is an internal partial (`user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`) — never invoked directly. It is **loaded by operator skills only** (not part of the always-load layer): `/fab-operator` declares `helpers: [_cli-fab, _cli-external]`. Carrying the fab-owned choreography and the tmux/`/loop` notes here means only operator skills pay for it — every other skill still carries just the inline `rk` detection/fail-silent rule from `_preamble.md` § Run-Kit. Canonical source is the flat `src/kit/skills/_cli-external.md`; `fab sync` deploys it to `.claude/skills/_cli-external/SKILL.md`.

> **No prior SPEC mirror existed** (260620): this file and `SPEC-_cli-fab.md` backfill the two missing CLI-partial mirrors so the constitution's SPEC-mirror rule holds across the whole `src/kit/skills/` tree.

## Command Inventory

One `##` section per tool (plus the framing Reference Model), mirroring the partial's section order. Each section documents only the **fab-owned** surface of its tool — the identity plus fab integration choreography — delegating tool-owned usage/flags to `<tool> skill` / `<tool> help-dump` at use-time.

| Section | Covers |
|---------|--------|
| Reference Model | The fab-owned-only convention plus the twin use-time delegations — `<tool> skill` (usage briefing, per `shll standards skill`) and `<tool> help-dump` (command tree); the required **version-skew fallback** (`<tool> skill` probe → silent `https://shll.ai/<tool>/skill` pointer on an old binary); and the one-class absent-binary discipline (all four owned binaries are separate sibling formulas that may be absent — every delegation `command -v`-gated fail-silent, while `wt`'s *functional* entry points — operator spawning, `fab batch new`/`switch` — instead stop with an install hint) governing both delegations |
| wt (Worktree Manager) | The operator's **fab-owned** spawn-in-worktree choreography: run `wt create` in the target repo's directory, `fab agent --print --repo <target-repo>`, tmux new-window, and the Operator Spawning Rules (known-change probe-and-route with the `--checkout`-vs-positional decision + `/fab-new` Step 11 rename + no-`/git-branch` rule). wt's command set, `wt create` flags, and the 260717-2af2 branch-selection contract itself are **tool-owned** — read via `wt skill`/`wt help-dump` |
| idea (Backlog Manager) | The one-line identity (backlog inbox feeding `/fab-new <id>`, backlog IDs consumed by `_intake` Step 0); verbs/flags/query/backlog-format are tool-owned (`idea skill`) |
| hop (Multi-Repo Navigator) | The optional-sibling identity + fail-silent gate, and the fab-owned "why it matters" (discovering a sibling repo's main-worktree root for spawning); discovery commands are tool-owned (`hop skill`) |
| tmux | Pane/session primitives the operator builds on (`send-keys`, session/pane addressing) layered under `fab pane` (third-party — no `skill` bundle) |
| rk (run-kit) | The one-line identity + detection/fail-silent pointer, and the **fab-owned** operator escalation send (`rk notify` with the operator's `{change}: {summary} ({repo})` / `Operator: strategic question` template). The `rk notify` contract, `rk context` (server-URL/iframe/proxy/Visual Display Recipe), and the rest are tool-owned (`rk skill`; live environment via `rk context`). Since run-kit v3.0.0 the formula/binary are named `run-kit` (`sahil87/tap/run-kit`); `rk` is a symlink alias, the invocation form throughout fab skills |
| /loop | The recurring-interval driver the operator uses for its monitoring tick (a Claude Code skill, not a binary — no `skill` bundle) |

> The inventory mirrors the file's `##` section order. `_preamble.md` § Run-Kit carries the `rk` detection/fail-silent rule plus the matching version-skew fallback (`rk skill` probe → silent `https://shll.ai/rk/skill` pointer, so the always-load layer stays self-consistent) and points here for the fab-owned escalation usage (no longer for a full command body — that is delegated to `rk skill`); a change to either side must keep the pointer accurate (cross-reference rule), and by the mirror rule, this SPEC's corresponding row.

### Tools used

None — `_cli-external.md` is a reference document consumed by operator skills (looked up, not executed). The tools it documents are run by the consuming skills via Bash; the file itself defines no flow and runs nothing.

### Sub-agents

None.
