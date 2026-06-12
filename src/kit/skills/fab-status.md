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

Use `fab preflight` and `fab status` for validation and data retrieval. The skill handles formatting and presentation:

- Reads kit VERSION (via `fab kit-path`), `fab/.kit-migration-version` (if exists), `.fab-status.yaml`, and `fab/changes/{name}/.status.yaml`
- Queries live branch via `git branch --show-current` (instead of reading a static `branch:` field from `.status.yaml`)
- **Version drift check**: if `fab/.kit-migration-version` exists and its value is less than the kit VERSION, display a warning: `Version drift: local {local}, engine {engine} -- run /fab-setup migrations`. If versions match, no warning. If `fab/.kit-migration-version` doesn't exist, no warning (handled by `/fab-setup`)
- Uses `display_stage` and `display_state` from preflight output for the primary "Stage:" line, showing the stage with a state qualifier (e.g., `Stage: intake (1/6) — done`). The "Next:" line shows the routing stage with the default command (e.g., `Next: apply (via /fab-continue)`). When all stages are done, shows `Next: /fab-archive`
- Renders the full status block: version header, change name, branch, stage with state qualifier (out of 6 total stages), next action, progress table with symbols (`✓` done, `●` active, `◷` ready, `○` pending, `✗` failed, `⏭` skipped — the skipped glyph matches the Go renderer's `ProgressLine`), plan counts (tasks: `{plan.task_count}`, acceptance: `{plan.acceptance_completed}/{plan.acceptance_count}`), confidence score, optional Impact line (see below), optional refactor-growth warning (see below), version drift warning (if applicable)
- **Impact line** — when `.status.yaml` `true_impact` block is present, render a single line under the change summary, sourced from the block:

  ```
  Impact: +{net} (raw {added}/-{deleted}, excluding fab/docs +{excl_net} ({excl_added}/-{excl_deleted}))
  ```

  When `excluding` is absent in the block (project's `true_impact_exclude` is empty), render only the raw figures: `Impact: +{net} ({added}/-{deleted})`. When `true_impact` is absent entirely, omit the line — no "not yet computed" placeholder. The line MUST be prefixed with a warning emoji (⚠️) and rendered **bold** when EITHER `true_impact.net > 100` OR `true_impact.excluding.net > 50` (when `excluding` is present) — mirroring fab-operator's health-emoji convention. ANSI SGR escapes (e.g., `\e[33m`) MUST NOT be used: they do not survive the assistant-message render path (verified empirically in fab-operator §4); emoji + bold are the surviving channels. Both thresholds are hard-coded; they MUST NOT be project-configurable.
- **Refactor-growth soft warning** — when ALL of the following hold, emit a single line below the impact line: (a) `change_type == refactor`, (b) `true_impact.excluding.net > 50` if `excluding` is present, else `true_impact.net > 50`, (c) `true_impact` block is present (from any stage). Warning text (exact, hard-coded):

  ```
  Refactor changes typically shrink or stay flat — review whether this growth is intentional.
  ```

  The +50 threshold is hard-coded. The warning is informational only — no gate, no block.
- Handles all error cases (no active change, missing `.status.yaml`, missing fields)
- Defaults missing progress fields to `○` (pending), missing plan to "plan not yet generated", and missing confidence to "not yet scored"
- **Confidence display** — read uniformly from `.status.yaml` (via preflight output) for all stages (the `indicative` flag is retired in 1.10.0 — intake scoring is authoritative, so there is no separate "indicative" label):
  - **Score > 0.0**: `Confidence: {score} of 5.0 ({N} certain, {N} confident, {N} tentative)` — appends `, {N} unresolved` only when unresolved > 0.
  - **Score = 0.0 with all grade counts 0 (template default, pre-intake)**: `Confidence: not yet scored`

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — purely informational, read-only |
| Idempotent? | **Yes** — no side effects, safe to call any number of times |
| Modifies `.fab-status.yaml`? | **No** |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Requires config/constitution? | **No** |
