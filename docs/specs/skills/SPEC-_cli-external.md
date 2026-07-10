# _cli-external

## Summary

External CLI tool reference — the non-fab command-line tools used for multi-agent coordination: **wt** (worktree manager), **idea** (backlog manager), **hop** (multi-repo navigator), **tmux**, **rk** (run-kit: context/iframe/proxy/visual-display + notify), and **/loop**. It is a hand-authored *gist* per tool — the operator-critical commands, flags, and integration semantics — and deliberately delegates each tool's exhaustive command/flag surface to that tool's own `help-dump` at use-time. Like `_cli-fab.md`, it is a **reference catalog**, not a procedure.

This is an internal partial (`user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`) — never invoked directly. It is **loaded by operator skills only** (not part of the always-load layer): `/fab-operator` declares `helpers: [_cli-fab, _cli-external]`. Carrying the `rk` and tmux detail here means only operator skills pay for it — every other skill still carries just the inline `rk` detection/fail-silent rule from `_preamble.md` § Run-Kit. Canonical source is the flat `src/kit/skills/_cli-external.md`; `fab sync` deploys it to `.claude/skills/_cli-external/SKILL.md`.

> **No prior SPEC mirror existed** (260620): this file and `SPEC-_cli-fab.md` backfill the two missing CLI-partial mirrors so the constitution's SPEC-mirror rule holds across the whole `src/kit/skills/` tree.

## Command Inventory

One `##` section per tool (plus the framing Reference Model), mirroring the partial's section order. Each section documents the operator-critical surface of its tool, not the full command set.

| Section | Covers |
|---------|--------|
| Reference Model | The hand-authored-gist-plus-`help-dump`-at-use-time convention; the universal silent-fail detection rule (`command -v <tool>`, skip silently when absent) shared by every external tool |
| wt (Worktree Manager) | Worktree create/list/remove; worktree directory naming (`{adjective}-{noun}`); the operator's spawn-in-worktree rules |
| idea (Backlog Manager) | Backlog entry management feeding the change pipeline (backlog IDs consumed by `_intake` Step 0) |
| hop (Multi-Repo Navigator) | Cross-repo navigation for the operator spanning multiple repos on one tmux server |
| tmux | Pane/session primitives the operator builds on (`send-keys`, session/pane addressing) layered under `fab pane` |
| rk (run-kit) | `rk context` (server-URL discovery, iframe windows, the proxy URL pattern, the Visual Display Recipe) and `rk notify` (the operator's default notification send) — the full body the `_preamble.md` § Run-Kit pointer forwards to. Since run-kit v3.0.0 the Homebrew formula and primary binary are named `run-kit` (`sahil87/tap/run-kit`); `rk` is kept as a symlink alias and remains the invocation form throughout fab skills |
| /loop | The recurring-interval driver the operator uses for its monitoring tick |

> The inventory mirrors the file's `##` section order. `_preamble.md` § Run-Kit carries only the `rk` detection/fail-silent rule and points here for the command body; a change to either side must keep the pointer accurate (cross-reference rule), and by the mirror rule, this SPEC's corresponding row.

### Tools used

None — `_cli-external.md` is a reference document consumed by operator skills (looked up, not executed). The tools it documents are run by the consuming skills via Bash; the file itself defines no flow and runs nothing.

### Sub-agents

None.
