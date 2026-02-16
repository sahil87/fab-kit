---
name: fab-status
description: "Show current change state at a glance — name, branch, stage, checklist status, and suggested next command."
model_tier: fast
---

# /fab-status [<change-name>]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Show the current change state at a glance — change name, branch, stage progress, checklist status, kit version, and suggested next command. Provides a quick orientation for where you are in the workflow without modifying anything.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change instead of the active one in `fab/current`. Supports full folder names, partial slug matches, or 4-char IDs (e.g., `r3m7`). When provided, passed to the status script as `$1` for transient resolution — `fab/current` is **not** modified.

If no argument is provided, the skill displays status for the active change in `fab/current`.

---

## Context Loading

This skill uses **minimal context** — it does not need to load `fab/config.yaml` or `fab/constitution.md` (as noted in `_context.md`, status is exempt from the "Always Load" requirement).

---

## Behavior

Run the preflight script to resolve the change, then render the status display:

```bash
bash fab/.kit/scripts/lib/preflight.sh [change-name]
```

Use `lib/preflight.sh` and `lib/stageman.sh` for validation and data retrieval. The skill handles formatting and presentation:

- Reads `fab/.kit/VERSION`, `fab/VERSION` (if exists), `fab/current`, `fab/changes/{name}/.status.yaml`, and `fab/config.yaml` (for `git.enabled`)
- Queries live branch via `git branch --show-current` when git is enabled (instead of reading a static `branch:` field from `.status.yaml`)
- **Version drift check**: if `fab/VERSION` exists and its value is less than `fab/.kit/VERSION`, display a warning: `⚠ Version drift: local {local}, engine {engine} — run /fab-setup migrations`. If versions match, no warning. If `fab/VERSION` doesn't exist, no warning (handled by `/fab-setup`)
- Renders the full status block: version header, change name, branch (when git enabled), stage number, progress table with symbols (`✓` done, `●` active, `○` pending, `—` skipped, `✗` failed), checklist counts, confidence score, version drift warning (if applicable), and next command suggestion
- Handles all error cases (no active change, missing `.status.yaml`, missing fields)
- Defaults missing progress fields to `○` (pending), missing checklist to "not yet generated", and missing confidence to "not yet scored"
- Confidence display: `Confidence: {score} of 5.0 ({N} certain, {N} confident, {N} tentative)` — appends `, {N} unresolved` only when unresolved > 0; shows `Confidence: not yet scored` when the confidence block is absent

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — purely informational, read-only |
| Idempotent? | **Yes** — no side effects, safe to call any number of times |
| Modifies `fab/current`? | **No** |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Requires config/constitution? | **No** |
