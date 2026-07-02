# _cli-fab

## Summary

Fab CLI command reference — the exhaustive companion to the **Common fab Commands** headline table in `_preamble.md`. The preamble documents the 6 most-used command families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`); this partial documents everything else plus the full flag/argument surface, the calling convention, the extended `fab score` formula and `.status.yaml` schema details, and the common error messages. It is a **reference catalog**, not a procedure: skills look up command signatures here rather than running a defined flow.

This is an internal partial (`user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`) — never invoked directly. It is loaded selectively via a skill's `helpers: [_cli-fab]` frontmatter (per `_preamble.md` § Skill Helper Declaration); `/fab-operator` is its primary consumer (`helpers: [_cli-fab, _cli-external]`). Canonical source is the flat `src/kit/skills/_cli-fab.md`; `fab sync` deploys it to `.claude/skills/_cli-fab/SKILL.md`.

> **No prior SPEC mirror existed** (260620): the two CLI-partial mirrors (`SPEC-_cli-fab.md`, `SPEC-_cli-external.md`) were missing while every other `src/kit/skills/*.md` had a `docs/specs/skills/SPEC-*.md` twin. This file backfills the gap so the constitution's SPEC-mirror rule (every skill/partial edit pairs with its mirror) holds across the whole `src/kit/skills/` tree.

## Command Inventory

The partial is organized as one `##` section per command (or command group), plus framing sections. Each documents the command's purpose, arguments, flags, and output contract. The `_preamble.md`-covered families (`change`, `status`, `score`, `preflight`, `log`, `resolve`) appear here in **extended** form — additional subcommands and exhaustive flag detail beyond the preamble headline.

| Section | Covers |
|---------|--------|
| Calling Convention | How fab commands are invoked (paths relative to repo root), exit-code/stderr contract, the best-effort vs. fail-fast distinction |
| fab change (extended subcommand details) | Lifecycle subcommands beyond the preamble headline (`new`, `switch`, `rename`, `list`, `archive`, `restore`, `resolve`) |
| fab status (extended subcommand details) | State-machine subcommands beyond the headline (`finish`, `advance`, `start`, `reset`, `skip`, `fail`, `refresh`, `set-*`, `add-issue`, `add-pr`, …) and their state transitions. `refresh` recomputes artifact-derived fields from `intake.md`/`plan.md` (pull-based successor to the removed artifact-write hook), self-healed at `advance`/`finish`/`preflight` |
| fab score (extended) | The confidence formula, the SRAD `.status.yaml` schema, `--check-gate` / `--stage` semantics, status-template details |
| fab preflight (extended) | The structured-YAML output fields and internal validation steps |
| fab log (extended) | Append-only `.history.jsonl` logging beyond the `log command` headline |
| fab resolve (extended) | Query flags (`--id` / `--folder` / `--dir` / `--status` / `--pane`) and canonical-output forms |
| fab resolve-agent | Per-stage model/effort tier resolution (`<stage>` → tier → `{model, effort}`); `--alias` for the Agent-tool short alias |
| fab config reference | Prints the fully-commented reference config.yaml (all available options — binary- and skill-consumed keys), generated from the binary's constants; pure query, no flags, byte-stable stdout, exit 0 on success (a usage error from cobra.NoArgs exits non-zero) |
| fab hook | Claude Code hook subcommands — the three session-scoped runtime-telemetry handlers (`session-start`, `stop`, `user-prompt`) plus `sync`. Artifact bookkeeping is no longer a hook — it is pull-based via `fab status refresh` |
| fab pane | Tmux pane operations (`map`, `--all-sessions`, `--json`) used by the operator |
| fab doctor | Prerequisite validation |
| fab migrations-status | Which migrations apply between local and engine versions |
| fab kit-path | Resolve the kit directory path (`$(fab kit-path)/templates/…`) |
| fab shell-init | Emit the shell-completion script |
| fab impact | Git diff line-count math (added/deleted/net) between two refs |
| fab pr-meta | Render a fab PR's `## Meta` block as final markdown |
| fab memory-index | Deterministically (re)generate `docs/memory/` index + log files |
| fab fab-help | The workflow overview / command summary (backs `/fab-help`) |
| fab help-dump | Machine-readable command dump |
| fab operator | Launch the operator in a dedicated tmux tab (singleton); degraded-behavior contract |
| fab spawn-command | Print a repo's configured agent spawn command |
| fab batch | Multi-target batch operations |
| Common Error Messages | The shared error strings and their meanings |

> The inventory mirrors the file's `##` section order. When a command's signature changes, the constitution requires updating `_cli-fab.md` **and** its consumers' tests — and, by the mirror rule, this SPEC's corresponding row.

### Tools used

None — `_cli-fab.md` is a reference document consumed by skills (looked up, not executed). The commands it documents are run by the consuming skills via Bash; the file itself defines no flow and runs nothing.

### Sub-agents

None.
