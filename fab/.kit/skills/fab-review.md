---
name: fab-review
description: "Validate implementation against specs and checklists. On pass, advances to archive. On failure, presents rework options."
---

# /fab-review

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Validate implementation against specs and checklists. Inspects every task and checklist item, runs affected tests, spot-checks features against spec requirements, and checks for doc drift. On pass, advances to archive. On failure, presents rework options so the user can choose where to loop back.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Then verify stage-specific preconditions using the preflight output:

4. Verify that `progress.apply` is `done` (implementation must be complete before review)
5. Verify that `fab/changes/{name}/tasks.md` exists
6. Verify that `fab/changes/{name}/checklists/quality.md` exists

**If `progress.apply` is not `done`, STOP.** Output:

> `Apply stage is not complete. Run /fab-apply to finish implementation first.`

**If `tasks.md` does not exist, STOP.** Output:

> `No tasks.md found for this change. Run /fab-continue or /fab-ff to generate tasks first.`

**If `checklists/quality.md` does not exist, STOP.** Output:

> `No quality checklist found. Run /fab-continue or /fab-ff to generate the checklist first.`

---

## Context Loading

Load all context needed for review:

1. **`fab/config.yaml`** — project config, tech stack, conventions
2. **`fab/constitution.md`** — project principles and constraints
3. **`fab/design/index.md`** — specifications landscape (pre-implementation design intent, human-curated)
4. **`fab/changes/{name}/tasks.md`** — the task list (all should be `[x]`)
5. **`fab/changes/{name}/checklists/quality.md`** — the quality checklist to verify
6. **`fab/changes/{name}/spec.md`** — requirements and scenarios (the "what" to validate against)
7. **`fab/changes/{name}/brief.md`** — original intent (for reference)
8. **Centralized docs** — read `fab/docs/index.md` and the specific docs referenced by the brief's Affected Docs section, to check for doc drift
9. **Relevant source code** — read files touched by the change (referenced in tasks or task descriptions). Scope to files actually modified — do not load the entire codebase.

---

## Behavior

### Step 1: Verify All Tasks Complete

Read `fab/changes/{name}/tasks.md` and check every task item:

- Count total tasks and checked tasks (`- [x]`)
- If **any task is unchecked** (`- [ ]`), the review cannot proceed

**If unchecked tasks exist, STOP.** Output:

> `{N} of {total} tasks are incomplete. Run /fab-apply to finish implementation first.`
>
> `Incomplete tasks: {list of unchecked task IDs}`

If all tasks are checked, report:

> `✓ {total}/{total} tasks complete`

### Step 2: Verify Quality Checklist

Read `fab/changes/{name}/checklists/quality.md` and process each `CHK-*` item:

For **each checklist item**:

1. **Read the item**: Parse the CHK ID, category, and specific verifiable criterion
2. **Inspect relevant code/tests**: Read the source files and test files that relate to this checklist item. Cross-reference against `spec.md` requirements.
3. **Evaluate**:
   - If the criterion is met by the implementation, mark the item `[x]` in `checklists/quality.md`
   - If the item is not applicable to this change, mark as `[x]` and prefix with **N/A**: `- [x] CHK-{NNN} **N/A**: {reason}`
   - If the criterion is **not met**, leave unchecked (`- [ ]`) and record the failure with a specific explanation

After processing all items, report:

> `✓ {passed}/{total} checklist items passed` (if all pass)

or:

> `✗ {passed}/{total} checklist items passed`
> `Failed: {list of failed CHK IDs with brief reasons}`

### Step 3: Run Affected Tests

Run tests scoped to the modules and files touched by the change:

1. Identify test files associated with modified source files
2. If the project has a test runner configured in `fab/config.yaml`, use it
3. Run the scoped tests (not the full test suite unless the change is pervasive)
4. Report results:

> `✓ Tests passed ({N} test files, {M} assertions)` (adapt format to test runner output)

or:

> `✗ Tests failed: {summary of failures}`

### Step 4: Spot-Check Spec Requirements

Compare the implementation against key requirements from `spec.md`:

1. Read each requirement section in `spec.md`
2. For requirements with GIVEN/WHEN/THEN scenarios, verify the implementation handles the described behavior
3. Spot-check that the code matches the specified behavior — read the relevant source files and confirm
4. Report:

> `✓ Spec requirements verified` (if all match)

or:

> `✗ Spec drift detected: {list of mismatches between spec and implementation}`

### Step 5: Check for Doc Drift

Compare the implementation against centralized docs referenced in the brief:

1. Read the centralized docs listed in the brief's **Affected Docs** section
2. Verify the implementation doesn't contradict what those docs describe
3. Note any docs that will need updating during `/fab-archive` (this is informational, not a failure — `/fab-archive` handles the hydration)
4. Report:

> `✓ No doc drift detected`

or:

> `⚠ Doc updates needed during archive: {list of docs that diverged}`

(Doc drift is a warning, not a failure — it signals work for `/fab-archive` but does not block the review.)

---

## Review Verdict

After all five checks, determine the overall result:

### Pass — All Checks Green

All of these must be true:
- All tasks `[x]`
- All checklist items `[x]` (including N/A items)
- Tests pass
- Spec requirements match implementation
- No blocking issues found

**On pass**, update `.status.yaml`:
- Set `progress.review` to `done`
- Set `progress.archive` to `active` (two-write transition: review → done, archive → active)
- Update `checklist.completed` to match the count of checked items
- Update `last_updated` to the current ISO 8601 timestamp

Output the full review report, then:

> `Next: /fab-archive`

### Fail — One or More Checks Failed

Any checklist item unchecked, tests failing, or spec drift detected.

**On failure**, update `.status.yaml`:
- Set `progress.review` to `failed`
- Set `progress.apply` to `active` (backward movement: review → failed, apply → active)
- Update `checklist.completed` to match the current count of checked items
- Update `last_updated` to the current ISO 8601 timestamp

Output the full review report with specific failure details (CHK IDs, test names, spec mismatches), then present the rework options:

---

## Rework Options (On Failure)

Present all applicable options and let the user choose:

### Option 1: Fix Code → `/fab-apply`

**When to use**: Implementation bug — the code doesn't match what the tasks describe.

**What happens**:
1. Identify which tasks need rework based on the failed checklist items
2. Uncheck those tasks in `tasks.md`: change `- [x] T{NNN}` back to `- [ ] T{NNN}`
3. Add a rework comment after the unchecked task: `<!-- rework: {reason from failed CHK item} -->`
4. The user then runs `/fab-apply`, which picks up the unchecked items and re-implements them

**Output**:
> `Option 1: Fix code — uncheck {N} tasks for rework, then run /fab-apply`

### Option 2: Revise Tasks → edit `tasks.md`, then `/fab-apply`

**When to use**: Tasks are missing or wrong — the task list doesn't cover what the spec requires.

**What happens**:
1. Add new tasks to `tasks.md` with the next sequential ID (e.g., if last task is T012, new tasks start at T013)
2. Or modify existing task descriptions to correct them (uncheck modified tasks)
3. Completed tasks that are unaffected stay `[x]` — only new or revised tasks are executed
4. The user then runs `/fab-apply`

**Output**:
> `Option 2: Revise tasks — add/modify tasks in tasks.md, then run /fab-apply`

### Option 3: Revise Spec → `/fab-continue spec`

**When to use**: Requirements were wrong or incomplete.

**What happens**:
1. Resets to spec stage via `/fab-continue spec`
2. `spec.md` is updated in place
3. Tasks are subsequently regenerated — all downstream artifacts are reset
4. After spec revision, the user works through `/fab-continue` stages again

**Output**:
> `Option 3: Revise spec — run /fab-continue spec to reset and regenerate all downstream`

---

## Output

### Full Pass

```
Review: {change name}

Tasks:     ✓ {total}/{total} complete
Checklist: ✓ {total}/{total} passed
Tests:     ✓ Passed
Spec:      ✓ Requirements verified
Docs:      ✓ No drift detected (or ⚠ updates needed during archive)

Review PASSED. All checks green.

Next: /fab-archive
```

### Partial Failure

```
Review: {change name}

Tasks:     ✓ {total}/{total} complete
Checklist: ✗ {passed}/{total} passed
  - CHK-007: {failure reason}
  - CHK-011: {failure reason}
Tests:     ✓ Passed (or ✗ {failure summary})
Spec:      ✓ Requirements verified (or ✗ {drift details})
Docs:      ✓ No drift detected

Review FAILED. {N} issue(s) found.

Rework options:
  1. Fix code — uncheck {N} tasks for rework, then /fab-apply
  2. Revise tasks — add/modify tasks in tasks.md, then /fab-apply
  3. Revise spec — /fab-continue spec (resets all downstream)

Which option? (1-3)
```

### Apply Not Complete

```
Apply stage is not complete. Run /fab-apply to finish implementation first.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| `progress.apply` is not `done` | Abort with: "Apply stage is not complete. Run /fab-apply to finish implementation first." |
| `tasks.md` missing | Abort with: "No tasks.md found. Run /fab-continue or /fab-ff to generate tasks first." |
| `checklists/quality.md` missing | Abort with: "No quality checklist found. Run /fab-continue or /fab-ff to generate the checklist first." |
| Unchecked tasks found | Abort with incomplete task list — user must run /fab-apply first |
| Checklist item fails | Record failure with CHK ID and reason; include in final report |
| Tests fail | Record failure; include in final report |
| Spec drift detected | Record mismatch; include in final report |
| Doc drift detected | Report as warning (not blocking) — handled by /fab-archive |
| All checks pass | Set `review: done`, `archive: active`, output Next: /fab-archive |
| Any check fails | Set `review: failed`, `apply: active`, present rework options |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **Yes** — sets `progress.review` to `done` on pass (with `archive: active`) or `failed` on failure (with `apply: active`) |
| Idempotent? | **Yes** — safe to re-invoke; re-evaluates all checks from scratch |
| Modifies tasks.md? | **Only on rework** — unchecks tasks and adds `<!-- rework: reason -->` comments when user chooses Option 1 |
| Modifies checklists/quality.md? | **Yes** — marks checklist items `[x]` as they pass verification |
| Modifies source code? | **No** — review only reads and validates, does not change implementation |
| Updates `.status.yaml`? | **Yes** — sets progress (done/failed), checklist counts, and last_updated |

---

## Next Steps Reference

After `/fab-review` passes:

`Next: /fab-archive`

After `/fab-review` fails:

*(contextual — see rework options above)*
