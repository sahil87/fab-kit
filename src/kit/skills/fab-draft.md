---
name: fab-draft
description: "Create a change intake without activating it."
helpers: [_generation, _srad, _intake]
---

# /fab-draft <description>

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

`/fab-draft` creates a change folder and generates an intake without activating the change. This is the "queue for later" path — use `/fab-new` instead if you want to immediately start working on the change.

---

## Pre-flight

1. Verify `fab/project/config.yaml` and `fab/project/constitution.md` exist
2. **If either missing, STOP**: `fab/ is not initialized. Run /fab-setup first to bootstrap the project.`

---

## Arguments

- **`<description>`** *(required)* — natural language, Linear ticket ID (`DEV-988`), or backlog ID (`90g5`)

If no description: ask *"What change do you want to make?"*

---

## Behavior

### Steps 0–9: Create the Intake

Read `.claude/skills/_intake/SKILL.md` and execute the **Create-Intake Procedure** (Steps 0–9) with:

- **`{questioning-mode} = interactive`** — Step 8 asks the user via SRAD (no fixed cap; conversational mode when 5+ Unresolved). Same intake-creation behavior as `/fab-new`.

Then **STOP after the procedure's Step 9** (intake at `ready`):

- Do **NOT** activate the change — there is no Step 10. The `.fab-status.yaml` symlink is NOT created; the user must run `/fab-switch {name}` to make it active before proceeding.
- Do **NOT** create a git branch — there is no Step 11. Run no `fab change switch` and no `git` command.

This is the only difference between `/fab-draft` and `/fab-new`: `/fab-draft` calls the shared Create-Intake Procedure and stops at `ready`, while `/fab-new` follows the same procedure with its own Steps 10–11 tail (activate + branch). Because those steps live only in `fab-new.md`'s tail — which `/fab-draft` never reads — there is no "run activation by momentum" hazard here.

### Output

Use fab-new's Output block **minus** the `Activated: {name}` and `Branch: ...` lines, ending with the Activation Preamble `Next:` line (`_preamble.md` § Activation Preamble — `/fab-draft` always uses it):

```
Next: /fab-switch {name} to make it active, then /fab-continue, /fab-ff, /fab-fff, /fab-proceed, or /fab-clarify
```

(The command list after "then" is the state table's intake row, derived per `_preamble.md` § Lookup Procedure — default first, not hardcoded.)

### Error Handling

The Create-Intake Procedure's own error conditions apply (config/constitution missing, no description, intake template missing, `fab change new` collision/failure, Linear/backlog lookup failures). `/fab-draft` adds **no** activation/git rows — `fab change switch`, not-in-git-repo, and `git checkout`/`git branch` failures cannot occur here because those steps never run.

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Partially — re-running with the same backlog/Linear ID routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of creating a duplicate; a natural-language re-run intentionally creates a new change each run |
| Advances stage? | Yes — intake to `ready` |
| Modifies `.fab-status.yaml`? | No — change is not activated |
| Modifies git state? | No |

---

Next: `/fab-switch {name} to make it active, then` {intake-state commands per `_preamble.md` § Lookup Procedure: `/fab-continue, /fab-ff, /fab-fff, /fab-proceed, or /fab-clarify`}
