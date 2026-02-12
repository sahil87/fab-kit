---
name: fab-archive
description: "Complete a change — validate review passed, hydrate learnings into centralized docs, and move to archive."
---

# /fab-archive

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Complete a change and hydrate its learnings into centralized docs. Performs a final validation that review has passed, checks for concurrent changes touching the same docs, hydrates `spec.md` into `fab/docs/`, updates status, moves the change folder to the archive, and clears the active change pointer. After archiving, the project's centralized documentation reflects all new requirements and design decisions from the change.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Then verify stage-specific preconditions using the preflight output:

4. Verify that `progress.review` is `done` (review must have passed before archiving)
5. Verify that all tasks in `fab/changes/{name}/tasks.md` are checked (`[x]`)
6. Verify that all checklist items in `fab/changes/{name}/checklists/quality.md` are checked (`[x]`), including items marked **N/A**

**If `progress.review` is not `done`, STOP.** Output:

> `Review has not passed. Run /fab-review to validate implementation first.`

**If any tasks are unchecked in `tasks.md`, STOP.** Output:

> `{N} of {total} tasks are incomplete. All tasks must be complete before archiving. Run /fab-apply to finish, then /fab-review.`

**If any checklist items are unchecked in `checklists/quality.md`, STOP.** Output:

> `{N} of {total} checklist items are not verified. All items must be checked (including N/A items). Run /fab-review to complete the checklist.`

---

## Context Loading

Load all context needed for archiving:

1. **`fab/config.yaml`** — project config, tech stack, conventions
2. **`fab/constitution.md`** — project principles and constraints
3. **`fab/design/index.md`** — specifications landscape (pre-implementation design intent, human-curated)
4. **`fab/changes/{name}/spec.md`** — requirements and scenarios to hydrate into centralized docs
5. **`fab/changes/{name}/brief.md`** — original intent, including Affected Docs section listing target centralized docs
7. **`fab/docs/index.md`** — top-level documentation index
8. **Target centralized doc(s)** — read the specific docs referenced by the brief's Affected Docs section (New, Modified, Removed entries). For each doc path listed, read `fab/docs/{domain}/{name}.md` if it exists. Also read the domain index `fab/docs/{domain}/index.md` if it exists.

---

## Behavior

### Step 1: Final Validation

Verify that the change is ready to archive:

1. Read `fab/changes/{name}/tasks.md` — confirm every task is `[x]`
2. Read `fab/changes/{name}/checklists/quality.md` — confirm every `CHK-*` item is `[x]` (including items marked **N/A**: these must still be `[x]`)

If any item is unchecked, STOP and direct the user to the appropriate skill (see Pre-flight Check messages above).

Report:

> `✓ Final validation passed — all tasks and checklist items complete`

### Step 2: Concurrent Change Check

Scan `fab/changes/` for **other** active change folders (exclude the current change and the `archive/` subfolder). For each active change found:

1. Read its `spec.md` (if it exists)
2. Check if it references any of the same centralized doc paths listed in the current change's brief Affected Docs section
3. If overlap is found, warn the user:

> `⚠ Change "{other-name}" also modifies {doc-path}. After this archive, that change's spec was written against a now-stale base. Re-review with /fab-review after switching to it.`

If no overlapping changes are found:

> `✓ No concurrent changes reference the same docs`

This is a **warning only** — it does not block archiving. Proceed to Step 3 regardless.

### Step 3: Hydrate into `fab/docs/`

Read `spec.md` and the current centralized doc(s), then rewrite the centralized docs to incorporate the changes.

For **each** centralized doc referenced in the brief's Affected Docs section:

#### 3a. Determine if Doc Exists

- Check if `fab/docs/{domain}/{name}.md` exists
- Check if the domain folder `fab/docs/{domain}/` exists

#### 3b. New Doc (doc does not exist)

If the centralized doc does not exist yet:

1. **Create domain folder** if `fab/docs/{domain}/` does not exist:
   - Create the directory `fab/docs/{domain}/`
   - Create `fab/docs/{domain}/index.md` with the domain index template:
     ```markdown
     # {Domain} Documentation

     | Doc | Description | Last Updated |
     |-----|-------------|-------------|
     ```
   - Add a row to `fab/docs/index.md` for the new domain:
     ```
     | [{domain}]({domain}/index.md) | {domain description} | {doc-name} |
     ```

2. **Create the doc file** from the individual doc template:
   - Create `fab/docs/{domain}/{name}.md` with the standard structure:
     ```markdown
     # {Doc Name}

     **Domain**: {domain}

     ## Overview

     <!-- 1-2 sentences describing what this doc covers. -->

     ## Requirements

     <!-- Requirements hydrated from spec.md -->

     ## Design Decisions

     <!-- Durable architectural decisions extracted from spec.md during hydration. -->

     ## Changelog

     | Change | Date | Summary |
     |--------|------|---------|
     ```
   - Populate the **Overview** section from the spec's context
   - Populate the **Requirements** section with requirements and scenarios from `spec.md` relevant to this doc
   - Populate the **Design Decisions** section with durable decisions from `spec.md` — see Step 3d
   - Add the first **Changelog** row — see Step 3e

3. **Update domain index** — add a row to `fab/docs/{domain}/index.md`:
   ```
   | [{name}]({name}.md) | {brief description} | {DATE} |
   ```

4. **Update top-level index** — if this is a new domain, the row was already added in step 1. If this is a new doc in an existing domain, update the doc-list column in `fab/docs/index.md` to include the new doc name (comma-separated list of all docs in the domain).

#### 3c. Existing Doc (doc already exists)

If the centralized doc already exists:

1. **Read the current doc** in full
2. **Update the Requirements section** semantically:
   - **From `spec.md`** → integrate new/changed requirements and scenarios into the Requirements section
   - Compare each requirement in `spec.md` against the existing doc to determine what's **new** (not present in current doc), **changed** (present but different), or **removed** (explicitly deprecated in spec's Deprecated Requirements section)
   - Add new requirements with their scenarios
   - Update changed requirements in place — modify the text and scenarios to reflect the new behavior
   - Remove deprecated requirements (delete the requirement and its scenarios from the doc). Add a note in the Changelog about what was removed.
   - **Minimize edits to unchanged sections** — do not rewrite or reformat parts of the doc that are not affected by this change. This prevents drift over successive archives.
3. **Update Design Decisions** — see Step 3d
4. **Add Changelog row** — see Step 3e
5. **Update domain index** — update the "Last Updated" column for this doc in `fab/docs/{domain}/index.md`
6. **Update top-level index** — update the doc-list column in `fab/docs/index.md` for this domain to reflect the current set of docs (in case a new doc was added to the domain by this archive)

#### 3d. Extract Design Decisions (from `spec.md`)

1. Read the **Decisions** section of `spec.md` (if it exists)
2. For each decision, evaluate whether it is **durable** (will remain relevant beyond this change):
   - **Include**: Architectural choices, API contracts, data model decisions, pattern selections, security boundaries
   - **Skip**: Tactical details (specific file paths, library install commands, setup steps, one-time migration scripts)
3. Add durable decisions to the centralized doc's **Design Decisions** section:
   ```markdown
   ### {Decision Title}
   **Decision**: {chosen approach}
   **Why**: {rationale}
   **Rejected**: {alternative and why it was worse}
   *Introduced by*: {change-name}
   ```
4. If the doc already has Design Decisions, append new decisions below existing ones. Do not remove or modify existing decisions from previous changes.

#### 3e. Add Changelog Row

Append a row to the doc's **Changelog** table (most recent first — new row goes at the top of the table body):

```
| {change-name} | {DATE} | {one-line summary of what changed} |
```

The summary should describe what this change added/modified/removed for this specific doc — not a generic change description.

#### 3f. Handle Removed Docs

If the brief's Affected Docs lists docs under **Removed Docs**:

1. Do NOT delete the file — deprecation is handled via the spec's Deprecated Requirements section
2. Add a Changelog row noting the deprecation
3. Add a notice at the top of the doc: `> **Deprecated**: This document was deprecated by change {change-name} on {DATE}.`

### Step 4: Update `.status.yaml`

Update `fab/changes/{name}/.status.yaml`:

- Set `stage` to `archive`
- Set `progress.archive` to `done`
- Update `last_updated` to the current ISO 8601 timestamp

### Step 5: Move Change Folder to Archive

1. Move the entire change folder: `fab/changes/{name}/` → `fab/changes/archive/{name}/`
   - The `fab/changes/archive/` directory is created by `/fab-init` and should already exist. If it doesn't, create it first.
2. Do NOT rename the folder — the date is already in the folder name
3. **Use relative paths** in all Bash commands — never expand to absolute paths (they break permission allow-patterns and are harder to review).

### Step 6: Update Archive Index

Maintain `fab/changes/archive/index.md` — a searchable summary of all archived changes.

**If `fab/changes/archive/index.md` does not exist:**

1. Create the file with a `# Archive Index` heading followed by a blank line
2. **Backfill** all existing archived change folders (sorted most-recent-first by folder name):
   - For each folder in `fab/changes/archive/` (excluding `index.md`):
     - Read `fab/changes/archive/{folder}/brief.md`
     - Extract the first 1-2 sentences from the **Why** section as the description
     - If `brief.md` does not exist, use the slug portion of the folder name as a fallback description
   - Write all entries as a bullet list, most-recent-first
3. The newly archived change is included in the backfill (it was moved in Step 5)

**If `fab/changes/archive/index.md` already exists:**

1. Read the existing file
2. Prepend a new entry for the archived change immediately after the `# Archive Index` heading (most-recent-first ordering)
3. Do NOT modify existing entries

**Entry format:**

```markdown
- **{folder-name}** — {1-2 sentence description from brief Why section}
```

### Step 7: Mark Backlog Item Done

If the brief contains a backlog ID reference, mark it as done in `fab/backlog.md`:

1. Read `fab/changes/{name}/brief.md` (or from archive if already moved)
2. Extract the backlog ID from the Origin section — look for the line `**Backlog**: [ID]` (e.g., `[0r8e]`, `[emcb]`)
3. If no backlog ID is found, skip this step
4. If a backlog ID is found:
   - Read `fab/backlog.md`
   - Search for a line in the **Backlog** section (unchecked items) that starts with `- [ ] [ID]` where ID matches the extracted backlog ID
   - If found:
     - Mark the item as done by changing `[ ]` to `[x]`
     - Move the entire line to the **Done** section (prepend it at the top of the Done list, most-recent-first ordering)
   - If not found in Backlog section, check if it's already in the Done section (already marked). If so, no action needed.
   - If not found in either section, skip silently (backlog ID may reference a deleted or external item)

Report:

> `✓ Backlog item [ID] marked done` (if found and updated)
> `✓ Backlog item [ID] already done` (if already in Done section)
> `— No backlog ID to update` (if brief contains no backlog reference)

### Step 8: Clear Pointer

Delete `fab/current` to indicate there is no active change.

---

## Order of Operations (Fail-Safe)

Steps 4–8 are executed in this specific order for safety:

1. **Status update first** (Step 4) — if the process is interrupted after this point, the change is marked archived but still in `changes/`. The agent can detect this state and complete the remaining steps on next invocation.
2. **Folder move second** (Step 5) — moves the change to the archive directory.
3. **Index update third** (Step 6) — updates the archive index after the folder is in place.
4. **Backlog update fourth** (Step 7) — marks the corresponding backlog item as done. Non-blocking — if this fails, the archive still succeeds.
5. **Pointer clear last** (Step 8) — `fab/current` is deleted only after the folder is safely archived and indexed. This means that during the archive process, `/fab-status` still reports the active change rather than "no active change" with a half-hydrated state.

**Recovery from interruption**: If the agent detects that `.status.yaml` has `progress.archive: done` but the folder is still in `fab/changes/` (not in `archive/`), it should complete the move, update the index, and clear the pointer.

---

## Output

### Successful Archive

```
Archive: {change name}

Validation: ✓ All tasks and checklist items complete
Concurrent: ✓ No conflicts (or ⚠ warnings listed)

Hydrated docs:
  - fab/docs/{domain}/{name}.md (new|updated)
  - fab/docs/{domain}/{name}.md (new|updated)

Status:   ✓ archive: done
Archived: ✓ fab/changes/archive/{name}/
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  ✓ [ID] marked done (or — No backlog ID to update)
Pointer:  ✓ fab/current cleared

Archive complete.

Next: /fab-new <description> (start next change)
```

### Successful Archive with Concurrent Warnings

```
Archive: {change name}

Validation: ✓ All tasks and checklist items complete
Concurrent: ⚠ 1 conflict(s)
  - Change "{other-name}" also modifies {doc-path}. Re-review with /fab-review after switching.

Hydrated docs:
  - fab/docs/{domain}/{name}.md (updated)

Status:   ✓ archive: done
Archived: ✓ fab/changes/archive/{name}/
Index:    ✓ fab/changes/archive/index.md updated
Backlog:  ⚠ 1 conflict(s) noted above (if concurrent warnings exist)
Backlog:  ✓ [ID] marked done (or — No backlog ID to update)
Pointer:  ✓ fab/current cleared

Archive complete.

Next: /fab-new <description> (start next change)
```

### Review Not Passed

```
Review has not passed. Run /fab-review to validate implementation first.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| `progress.review` is not `done` | Abort with: "Review has not passed. Run /fab-review to validate implementation first." |
| Any tasks unchecked in `tasks.md` | Abort with incomplete task count — user must run /fab-apply then /fab-review |
| Any checklist items unchecked | Abort with incomplete item count — user must run /fab-review |
| Concurrent change references same doc | Warn but proceed — not blocking |
| Target centralized doc does not exist | Create from template (new doc flow) |
| Target domain folder does not exist | Create folder + domain index (new domain flow) |
| No Decisions section in spec.md | Skip Design Decisions extraction — hydrate only requirements |
| `archive/` directory does not exist | Create it before moving |
| `archive/index.md` does not exist | Create with backfill of all existing archived changes |
| Archived change missing `brief.md` during backfill | Use folder slug as fallback description |
| Interrupted mid-archive (status=done but not moved) | Complete the move, update index, and clear pointer |
| No backlog ID in brief | Skip backlog update step — report "No backlog ID to update" |
| Backlog ID not found in backlog file | Skip silently — may be deleted or external reference |
| Backlog ID already in Done section | Skip update — report "Backlog item [ID] already done" |
| Hydration produces garbled output | Recovery: `git checkout` on affected doc files. Recommend reviewing diff before pushing. |
| All checks pass | Complete archive, output Next line |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **Yes** — sets `stage` to `archive`, progresses to `done` |
| Idempotent? | **Partially** — safe to re-invoke if interrupted mid-archive (detects and completes remaining steps). Hydration is NOT idempotent — re-running on an already-archived change would duplicate content. The pre-flight check guards against this by requiring `fab/current` to exist. |
| Modifies `fab/docs/`? | **Yes** — hydrates requirements and design decisions from the change into centralized docs |
| Modifies tasks.md? | **No** — archive only reads tasks for validation |
| Modifies source code? | **No** — archive only modifies documentation files |
| Updates `.status.yaml`? | **Yes** — sets stage to `archive`, progress to `done`, updates last_updated |
| Moves change folder? | **Yes** — from `fab/changes/{name}/` to `fab/changes/archive/{name}/` |
| Updates archive index? | **Yes** — creates or updates `fab/changes/archive/index.md` with entry for the archived change |
| Updates backlog? | **Yes** — marks the corresponding backlog item as done if backlog ID is present in brief |
| Clears `fab/current`? | **Yes** — deletes the pointer file |

---

## Next Steps Reference

After `/fab-archive` completes:

`Next: /fab-new <description> (start next change)`
