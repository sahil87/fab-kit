# Intake: Fab Draft Auto Activate

**Change**: 260404-rzvq-fab-draft-auto-activate
**Created**: 2026-04-04
**Status**: Draft

## Origin

> Review feedback on README.md: "/fab-switch after /fab-new is confusing (line 151) — A newcomer with one change will ask 'switch to what?' If /fab-new auto-activates, remove this step. If not, explain why."
>
> Discussion concluded with Option D: rename the behaviors — `/fab-new` becomes create+activate (what newcomers expect), new `/fab-draft` for create-only (power user path). Separate skill files, not a flag.

## Why

The current `/fab-new` → `/fab-switch` two-step is confusing for newcomers. When a user creates their first change and the next instruction says "switch to it," they reasonably ask "switch to what? I only have one change." In practice, 90%+ of `/fab-new` invocations are immediately followed by `/fab-switch` to the same change. Making `/fab-new` auto-activate removes a stumbling block from the primary onboarding path.

The power-user scenario — "queue a second change without switching context" — is real but rare, and deserves its own explicit command (`/fab-draft`) rather than being the default behavior.

## What Changes

### 1. `/fab-new` — add auto-activation (Step 10)

After Step 9 (advance intake to ready), add a new Step 10:

```bash
fab change switch "{name}"
```

Display the switch confirmation. Update the `Next:` line to remove the activation preamble — now goes straight to `Next: /fab-continue, /fab-fff, /fab-ff, or /fab-clarify`.

Update the skill description from "Start a new change from a natural language description. Creates the change folder and generates the intake." to "Start a new change — creates the intake and activates it."

Remove the `--switch` flag references (it existed in docs/specs but was never in the skill source — now the behavior is always-on).

### 2. `/fab-draft` — new skill (create-only, no activation)

New file `src/kit/skills/fab-draft.md`. Behavior is identical to the current `/fab-new` (Steps 0-9), but does NOT auto-activate. The `Next:` line retains the activation preamble: `Next: /fab-switch {name} to make it active, then /fab-continue or /fab-fff or /fab-clarify`.

Skill description: "Create a change intake without activating it."

### 3. `/fab-switch` — update error messages

Change "No active changes found. Run /fab-new." to "No active changes found. Run /fab-new or /fab-draft."

### 4. `/fab-proceed` — simplify prefix chain

Since `/fab-new` now auto-activates, the dispatch table row for "Conversation context (no intake)" changes from `/fab-new → /fab-switch → /git-branch → /fab-fff` to `/fab-new → /git-branch → /fab-fff`. The `/fab-switch` step after `/fab-new` is no longer needed.

Similarly, "Unactivated intake (no active change)" still needs `/fab-switch` — those intakes were created by `/fab-draft`, not `/fab-new`.

### 5. `_preamble.md` — update Activation Preamble

The Activation Preamble section says "This applies to `/fab-new` (always)". Update to say "This applies to `/fab-draft` (always)" since `/fab-new` now self-activates.

### 6. `README.md` — simplify walkthrough

Remove the `/fab-switch` line from the "Your first change" walkthrough. The sequence becomes:
```
/fab-new Add a loading spinner to the submit button
/fab-continue   # spec
/fab-continue   # tasks
...
```

### 7. `docs/specs/skills.md` — add `/fab-draft`, update `/fab-new`

- Update `/fab-new` section: remove `--switch` flag, update description and behavior to reflect auto-activation
- Add new `/fab-draft` section documenting the create-only behavior
- Update the Next Steps lookup table for `/fab-new` (no activation preamble)

### 8. `docs/specs/overview.md` — update references

Remove `--switch` from the `/fab-new` command table entry. Update the quick reference walkthrough.

### 9. Other cross-references

Update error messages and references in `fab-ff.md`, `fab-fff.md`, `fab-operator.md`, `fab-setup.md`, `_cli-fab.md`, and `_naming.md` where they mention `/fab-new` — most are fine as-is since they just say "run /fab-new", but operator routing tables and batch new logic may need awareness of the activation change.

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Add `/fab-draft` to the skill inventory, note `/fab-new` auto-activation behavior
- `fab-workflow/change-lifecycle`: (modify) Update the "create a change" flow to reflect auto-activation

## Impact

- **Skills**: `fab-new.md`, `fab-draft.md` (new), `fab-switch.md`, `fab-proceed.md`, `_preamble.md`
- **Docs**: `README.md`, `docs/specs/skills.md`, `docs/specs/overview.md`
- **Operator**: `fab-operator.md` dispatch table references may need updating
- **CLI**: No Go code changes — `fab change switch` already exists as a CLI command

## Open Questions

None — all design decisions resolved during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `/fab-draft` is a separate skill file, not a `--draft` flag on `/fab-new` | Discussed — user explicitly chose separate files for single-purpose clarity | S:95 R:85 A:95 D:95 |
| 2 | Certain | `/fab-new` always auto-activates (no opt-out flag) | Discussed — the opt-out path is `/fab-draft` | S:95 R:80 A:90 D:95 |
| 3 | Certain | No data migration needed | This is a skill behavior change, not a data format change | S:90 R:95 A:95 D:95 |
| 4 | Confident | `/fab-draft` shares all Steps 0-9 with `/fab-new` via copy, not shared partial | Constitution says "pure prompt play" — no build step or include mechanism; duplication is acceptable for two separate skill files | S:70 R:90 A:80 D:75 |
| 5 | Confident | `/fab-proceed` keeps `/fab-switch` in the "unactivated intake" path | Unactivated intakes exist because `/fab-draft` created them — they still need explicit activation | S:75 R:80 A:85 D:80 |
| 6 | Certain | Remove the never-implemented `--switch` flag from specs | The flag existed in `docs/specs/skills.md` but was never in the skill source — now superseded by always-on activation | S:90 R:90 A:90 D:95 |
| 7 | Confident | Operator routing unchanged for batch new | `fab batch new` spawns agents with `/fab-new`, which now auto-activates — this is correct behavior since each agent works on its own change | S:70 R:75 A:80 D:80 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
