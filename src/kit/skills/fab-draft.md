---
name: fab-draft
description: "Create a change intake without activating it."
helpers: [_generation, _srad]
---

# /fab-draft <description>

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Purpose

`/fab-draft` creates a change folder and generates an intake without activating the change. This is the "queue for later" path — use `/fab-new` instead if you want to immediately start working on the change.

---

## Behavior (delta over /fab-new)

`/fab-draft` is a **thin delta over `/fab-new`**. Read `.claude/skills/fab-new/SKILL.md` and execute its **Pre-flight**, **Arguments**, and **Steps 0–9** exactly as written there (self-name mentions read as `/fab-draft` — e.g., Step 4's "preceded this `/fab-new` invocation"), with these deltas:

1. **Step 9 tail**: after `fab status advance {name} intake`, the change is **NOT activated** — the user must run `/fab-switch {name}` to make it active before proceeding. (This replaces fab-new's Step 9 closing sentence about Step 10 activating the change.)

2. **SKIP Steps 10–11 ENTIRELY — no activation, no git branch.** Do NOT run `fab change switch`. Do NOT run any `git` command. Stop after Step 9. This is the defining difference from `/fab-new`; running activation or branch creation by momentum is the known failure mode of this delta form — before any `fab change switch` or `git` invocation, re-check that you are executing `/fab-draft`.

3. **Output**: fab-new's Output block **minus** the `Activated: {name}` and `Branch: ...` lines, ending with the Activation Preamble `Next:` line (`_preamble.md` § Activation Preamble — `/fab-draft` always uses it):

   ```
   Next: /fab-switch {name} to make it active, then /fab-continue, /fab-fff, /fab-ff, or /fab-clarify
   ```

4. **Error Handling**: fab-new's table **minus** the activation/git rows (`fab change switch` failure, not-in-git-repo, `git checkout`/`git branch` failure) — those steps never run here.

---

## Key Properties

| Property | Value |
|----------|-------|
| Idempotent? | Partially — re-running with the same backlog/Linear ID routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of creating a duplicate; a natural-language re-run intentionally creates a new change each run |
| Advances stage? | Yes — intake to `ready` |
| Modifies `.fab-status.yaml`? | No — change is not activated |
| Modifies git state? | No |

---

Next: `/fab-switch {name} to make it active, then /fab-continue, /fab-fff, /fab-ff, or /fab-clarify`
