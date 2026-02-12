---
name: fab-status
description: "Show current change state at a glance — name, branch, stage, checklist status, and suggested next command."
model_tier: fast
---

# /fab-status

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Show the current change state at a glance — change name, branch, stage progress, checklist status, kit version, and suggested next command. Provides a quick orientation for where you are in the workflow without modifying anything.

---

## Context Loading

This skill uses **minimal context** — it does not need to load `fab/config.yaml` or `fab/constitution.md` (as noted in `_context.md`, status is exempt from the "Always Load" requirement).

---

## Behavior

Run the shell script and present its output:

```bash
bash fab/.kit/scripts/fab-status.sh
```

The script handles all validation, parsing, and formatting:

- Reads `fab/.kit/VERSION`, `fab/current`, `fab/changes/{name}/.status.yaml`, and `fab/config.yaml` (for `git.enabled`)
- Queries live branch via `git branch --show-current` when git is enabled (instead of reading a static `branch:` field from `.status.yaml`)
- Renders the full status block: version header, change name, branch (when git enabled), stage number, progress table with symbols (`✓` done, `●` active, `○` pending, `—` skipped, `✗` failed), checklist counts, confidence score, and next command suggestion
- Handles all error cases (no active change, missing `.status.yaml`, missing fields)
- Defaults missing progress fields to `○` (pending), missing checklist to "not yet generated", and missing confidence to "not yet scored"
- Confidence display: `Confidence: {score}/5.0 ({N} certain, {N} confident, {N} tentative)` — appends `, {N} unresolved` only when unresolved > 0; shows `Confidence: not yet scored` when the confidence block is absent

**On exit 0**: Present the stdout output to the user as-is (it is pre-formatted).

**On non-zero exit**: Present the stdout output — it contains the user-facing error message.

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
