---
name: fab-status
description: "Show current change state at a glance — name, branch, stage, checklist status, and suggested next command."
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

- Reads `fab/.kit/VERSION`, `fab/current`, and `fab/changes/{name}/.status.yaml`
- Renders the full status block: version header, change name, branch, stage number, progress table with symbols (`✓` done, `●` active, `○` pending, `—` skipped, `✗` failed), checklist counts, and next command suggestion
- Handles all error cases (no active change, missing `.status.yaml`, missing fields)
- Defaults missing progress fields to `○` (pending) and missing checklist to "not yet generated"

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
