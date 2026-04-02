---
name: fab-help
description: "Show the fab workflow overview and a quick summary of all available commands."
---

# /fab-help

---

## Purpose

Orient the user in the fab workflow. Show how the stages fit together and list every `/fab-*` command with a one-line description. Read-only — no files are created or modified.

---

## Behavior

When invoked, log the command and execute the help subcommand:

```bash
fab log command "fab-help" 2>/dev/null || true
fab fab-help
```

The subcommand reads the kit version from `fab/.kit/VERSION` (falling back to "unknown" if missing), scans `fab/.kit/skills/*.md` frontmatter for command descriptions, and prints the complete help text. The subcommand is the single source of truth for help content.

---

## Context Loading

This skill uses **no context** — it does not load `fab/project/config.yaml`, `fab/project/constitution.md`, or any change artifacts. The only file read is `fab/.kit/VERSION` (by the script).

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — purely informational |
| Idempotent? | **Yes** — no side effects |
| Modifies any files? | **No** |
| Requires active change? | **No** |
| Requires config/constitution? | **No** |
