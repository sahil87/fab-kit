---
name: fab-status
description: "Show current change state at a glance — name, branch, stage, plan progress, and suggested next command."
---

# /fab-status [<change-name>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

Show the current change state at a glance — change name, branch, stage progress, plan progress (tasks + acceptance counts), kit version, and suggested next command. Provides a quick orientation for where you are in the workflow without modifying anything.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one resolved via `.fab-status.yaml`. Supports full folder names, partial slug matches, or 4-char IDs (e.g., `r3m7`). When provided, passed to the status script as `$1` for transient resolution — `.fab-status.yaml` is **not** modified.

If no argument is provided, the skill displays status for the active change resolved via `.fab-status.yaml`.

---

## Context Loading

This skill uses **minimal context** — it does not need to load `fab/project/config.yaml` or `fab/project/constitution.md` (as noted in `_preamble.md`, status is exempt from the "Always Load" requirement).

---

## Behavior

Run the preflight script to resolve the change, then render the status display:

```bash
fab preflight [change-name]
```

Use `fab preflight` and `fab status` for validation/data. Read kit VERSION via `fab kit-path`, optional `fab/.kit-migration-version`, `.fab-status.yaml`, and `fab/changes/{name}/.status.yaml`; query the live branch with `git branch --show-current` rather than a static status field.

Render in this order:

1. Version header, change name, and live branch
2. `display_stage` + `display_state` as `Stage: {stage} ({N}/6) — {state}`
3. Routing stage/default command as `Next: {stage} (via {command})`, or `Next: /fab-archive` when complete
4. Progress table, then plan task/acceptance counts
5. Confidence
6. Optional Impact and refactor-growth lines
7. Optional version-drift warning

| State | Glyph |
|-------|-------|
| done | `✓` |
| active | `●` |
| ready | `◷` |
| pending / missing progress | `○` |
| failed | `✗` |
| skipped | `⏭` (matches Go `ProgressLine`) |

| Condition | Rendering |
|-----------|-----------|
| Migration version exists and is below kit VERSION | `Version drift: local {local}, engine {engine} -- run /fab-setup migrations`; otherwise omit |
| `true_impact` with `excluding` | `Impact: +{net} (raw {added}/-{deleted}, excluding fab/docs +{excl_net} ({excl_added}/-{excl_deleted}))` |
| `true_impact` without `excluding` | `Impact: +{net} ({added}/-{deleted})` |
| No `true_impact` | Omit Impact; no placeholder |
| `true_impact.net > 100`, or present `excluding.net > 50` | Prefix Impact with ⚠️ and render **bold**; thresholds are hard-coded and not configurable |
| `change_type == refactor` and effective non-excluded net > 50 | Below Impact, emit exact hard-coded advisory `Refactor changes typically shrink or stay flat — review whether this growth is intentional.`; informational only |
| Confidence score > 0.0 | `Confidence: {score} of 5.0 ({N} certain, {N} confident, {N} tentative)`, appending `, {N} unresolved` only when nonzero |
| Score/counts all zero | `Confidence: not yet scored` |
| Missing plan | `plan not yet generated` |

Never use ANSI SGR escapes such as `\e[33m`; emoji + bold are the surviving assistant-message channels. Handle no active change, missing `.status.yaml`, and missing fields as errors.

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — purely informational, read-only |
| Idempotent? | **Yes** — no side effects, safe to call any number of times |
| Modifies `.fab-status.yaml`? | **No** |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Requires config/constitution? | **Exists-only** — `fab preflight` validates both files exist (it exits non-zero without them); their *content* is never loaded |
