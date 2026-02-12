---
name: fab-init-constitution
description: "Create or amend the project constitution with semantic versioning."
model_tier: fast
---

# /fab-init-constitution

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Context loading**: This skill loads `fab/config.yaml` (required for project context) and `fab/constitution.md` (if it exists). It does NOT load `fab/docs/index.md` or `fab/design/index.md`.

---

## Purpose

Create a new project constitution or amend an existing one. Manages the governance lifecycle of `fab/constitution.md` with semantic versioning, structural preservation, and audit trail output.

---

## Arguments

None. Mode is determined automatically by file existence.

---

## Pre-flight Check

1. Check that `fab/config.yaml` exists
   - If missing, **STOP**: `fab/config.yaml not found. Run /fab-init first.`
2. Read `fab/config.yaml` for project context
3. Check whether `fab/constitution.md` exists — this determines the mode

---

## Behavior

### Mode Selection

| `fab/constitution.md` exists? | Mode |
|-------------------------------|------|
| No | **Create mode** — generate a new constitution |
| Yes | **Update mode** — guided amendment |

### Create Mode

When `fab/constitution.md` does not exist:

1. Read project context from `fab/config.yaml` (project name, description, context/tech stack)
2. Examine additional project context: README, existing documentation, codebase structure, and conversation history
3. Generate `fab/constitution.md` with this structure:

```markdown
# {Project Name} Constitution

## Core Principles

### I. {Principle Name}
{Description using MUST/SHALL/SHOULD keywords. Include rationale.}

### II. {Principle Name}
{Description}

<!-- Generate 3-7 principles based on the project's actual patterns, tech stack, and constraints -->

## Additional Constraints
<!-- Project-specific: security, performance, testing, etc. -->

## Governance

**Version**: 1.0.0 | **Ratified**: {TODAY'S DATE} | **Last Amended**: {TODAY'S DATE}
```

4. Output confirmation:

```
Created fab/constitution.md (version 1.0.0) with {N} principles.
```

### Update Mode

When `fab/constitution.md` already exists:

1. Read and display the current constitution content
2. Read the current version from the Governance section
3. Present the amendment menu:

```
Current constitution: version {X.Y.Z}, {N} principles

What would you like to change?
1. Add a new principle
2. Modify an existing principle
3. Remove a principle
4. Add or modify a constraint
5. Update governance metadata
6. Done — no changes
```

4. Process the user's selection:
   - **Add principle**: Ask for the principle name and description. Insert at the next Roman numeral position. Record bump level: MINOR.
   - **Modify principle**: Show numbered list of principles, ask which to modify, accept new text. Record bump level: MAJOR if meaning changes, PATCH if clarification only. Ask the user which: "Is this a (1) fundamental change or (2) wording clarification?"
   - **Remove principle**: Show numbered list, ask which to remove. Re-number remaining principles with sequential Roman numerals. Record bump level: MAJOR.
   - **Add/modify constraint**: Show the Additional Constraints section, accept edits. Record bump level: MINOR for additions, PATCH for modifications.
   - **Update governance**: Allow editing the Ratified date or other governance metadata. Record bump level: PATCH.
   - **Done**: Proceed to version bump (if any changes were made).

5. After each action, ask: **"Any other changes? (yes/no)"**
   - If yes, return to the amendment menu (step 3)
   - If no, proceed to step 6

6. **Apply version bump**: Determine the final version based on the highest-severity bump encountered across all amendments in this session:
   - **MAJOR** takes precedence over MINOR and PATCH
   - **MINOR** takes precedence over PATCH
   - Update the Governance section: increment the appropriate version component, update "Last Amended" to today's date

7. **Structural preservation**: After all edits:
   - Verify heading hierarchy is intact (h1 > h2 > h3)
   - Verify Roman numeral numbering is sequential (I, II, III, ...)
   - Verify Governance section format is correct
   - Re-number principles if any were added or removed

8. Write the updated `fab/constitution.md`

9. **Output amendment summary**:

```
Amended fab/constitution.md:
- Added: III. {Principle Name} (MINOR)
- Removed: V. {Old Principle Name} (MAJOR)
- Modified: I. {Principle Name} (PATCH — clarification)

Version: {old} → {new}
```

### No-Op Handling

If the user selects "Done" without making any changes (or answers "no" to the first "Any other changes?" prompt after selecting "Done" from the menu), output:

```
No changes made. Constitution unchanged at version {X.Y.Z}.
```

Do NOT modify the file or bump the version.

---

## Output

### Create Mode

```
Created fab/constitution.md (version 1.0.0) with {N} principles.

Next: /fab-init-validate (verify structure) or /fab-new <description>
```

### Update Mode — Changes Applied

```
Amended fab/constitution.md:
- Added: VII. {Principle Name} (MINOR)

Version: 1.2.0 → 1.3.0

Next: /fab-init-validate (verify structure)
```

### Update Mode — No Changes

```
No changes made. Constitution unchanged at version 1.2.0.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing | Abort: "fab/config.yaml not found. Run /fab-init first." |
| `fab/constitution.md` malformed (update mode) | Warn: "Constitution structure appears non-standard. Proceeding with best-effort parsing." |
| Governance section missing version | Warn: "No version found in Governance section. Starting from 1.0.0." |
| Roman numeral parsing fails | Warn and proceed with sequential numbering from I |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — this is a project-level tool, not tied to the change pipeline |
| Idempotent? | **Yes** — create mode skips if file exists (use update mode); update mode with no changes is a no-op |
| Modifies `fab/constitution.md`? | **Yes** — creates or updates |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Requires config? | **Yes** — `fab/config.yaml` must exist |

---

## Next Steps Reference

After create mode: `Next: /fab-init-validate (verify structure) or /fab-new <description>`

After update mode: `Next: /fab-init-validate (verify structure)`
