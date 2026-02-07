---
name: fab-help
description: "Show the fab workflow overview and a quick summary of all available commands."
---

# /fab:help

---

## Purpose

Orient the user in the fab workflow. Show how the stages fit together and list every `/fab:*` command with a one-line description. Read-only — no files are created or modified.

---

## Behavior

When invoked, output the following **exactly** (substitute the kit version from `fab/.kit/VERSION`, or "unknown" if the file is missing):

---

## Output

```
Fab Kit v{version} — Specification-Driven Development

WORKFLOW

  /fab:new ─→ /fab:continue (or /fab:ff) ─→ /fab:apply ─→ /fab:review ─→ /fab:archive
               ↕ /fab:clarify

  Planning stages: proposal → specs → plan (optional) → tasks
  Execution stages: apply → review → archive

COMMANDS

  Start & Navigate
    /fab:new <desc>         Start a new change from a description
    /fab:switch [name]      Switch active change (lists all if no name)
    /fab:status             Show current change state at a glance

  Planning
    /fab:continue [stage]   Advance to the next planning stage (or reset to stage)
    /fab:ff                 Fast-forward through all remaining planning stages
    /fab:clarify            Refine the current stage artifact without advancing

  Execution
    /fab:apply              Implement tasks from tasks.md in dependency order
    /fab:review             Validate implementation against specs and checklists

  Completion
    /fab:archive            Complete change — hydrate docs, move to archive

  Setup
    /fab:init [sources...]  Bootstrap fab/ directory (safe to re-run)
    /fab:help               Show this help

TYPICAL FLOW

  Quick change:  /fab:new → /fab:ff → /fab:apply → /fab:review → /fab:archive
  Careful change: /fab:new → /fab:continue (repeat) → /fab:apply → /fab:review → /fab:archive
```

---

## Context Loading

This skill uses **no context** — it does not load `fab/config.yaml`, `fab/constitution.md`, or any change artifacts. The only file it reads is `fab/.kit/VERSION` for the version string.

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — purely informational |
| Idempotent? | **Yes** — no side effects |
| Modifies any files? | **No** |
| Requires active change? | **No** |
| Requires config/constitution? | **No** |
