---
name: fab-apply
description: "Execute implementation tasks from tasks.md in dependency order, running tests after each. Resumable."
---

# /fab-apply

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Purpose

Execute implementation tasks from `tasks.md`. Parses unchecked items, executes them in dependency order (respecting parallel markers), runs tests after each task, marks tasks complete immediately upon finishing, and updates `.status.yaml` progress throughout. Inherently resumable — re-invoking picks up from the first unchecked item.

---

## Pre-flight Check

Before doing anything else, run the preflight script:

1. Execute `fab/.kit/scripts/fab-preflight.sh` via Bash
2. If the script exits non-zero, **STOP** and surface the stderr message to the user
3. Parse the stdout YAML to get `name`, `change_dir`, `stage`, `progress`, `checklist`, and `confidence`

Then verify stage-specific preconditions using the preflight output:

4. Verify that `progress.tasks` is `done` (tasks stage must be complete before implementation can begin)
5. Verify that `fab/changes/{name}/tasks.md` exists

**If `progress.tasks` is not `done`, STOP.** Output:

> `Tasks stage is not complete. Run /fab-continue or /fab-ff to generate tasks first.`

**If `tasks.md` does not exist, STOP.** Output:

> `No tasks.md found for this change. Run /fab-continue or /fab-ff to generate tasks first.`

---

## Context Loading

Load all context needed for implementation:

1. **`fab/config.yaml`** — project config, tech stack, conventions
2. **`fab/constitution.md`** — project principles and constraints
3. **`fab/design/index.md`** — specifications landscape (pre-implementation design intent, human-curated)
4. **`fab/changes/{name}/tasks.md`** — the task list to execute
5. **`fab/changes/{name}/spec.md`** — requirements and scenarios (the "what" and "why")
6. **`fab/changes/{name}/brief.md`** — original intent (for reference)
7. **Relevant source code** — read files referenced in task descriptions. Scope to files actually touched — do not load the entire codebase.

---

## Behavior

### Step 1: Parse Tasks

Read `fab/changes/{name}/tasks.md` and extract all task items. Tasks follow the format:

```
- [ ] T{NNN} [{markers}] {description with file paths}
- [x] T{NNN} [{markers}] {description with file paths}  (already completed)
```

Build a task list:
- **Unchecked items** (`- [ ]`): tasks remaining to execute
- **Checked items** (`- [x]`): tasks already completed (skip these)
- **Phase headers** (`## Phase N: ...`): group boundaries for execution order

**If all tasks are already checked**, the implementation is complete. Output:

> `All tasks are already complete. Nothing to do.`
>
> `Next: /fab-review`

And update `.status.yaml`:
- Set `stage` to `apply`
- Set `progress.apply` to `done`
- Update `last_updated`

Then stop.

### Step 2: Determine Execution Order

Tasks execute in the order they appear in `tasks.md`, with these rules:

1. **Phases are sequential**: All tasks in Phase 1 must complete before Phase 2 begins, and so on.
2. **Within a phase, non-`[P]` tasks are sequential**: Execute in listed order — each task depends on the one before it.
3. **Within a phase, `[P]` tasks are parallelizable**: Tasks marked `[P]` within the same phase can be executed in any order or simultaneously, as they touch different files and have no dependencies on each other.
4. **The Execution Order section** at the bottom of `tasks.md` documents non-obvious dependencies. Respect any explicit dependency constraints listed there (e.g., "T004 blocks T005").

**Resumability**: Start from the **first unchecked item** (`- [ ]`). All checked items are assumed complete. This means re-invoking `/fab-apply` after an interruption picks up exactly where it left off.

### Step 3: Execute Each Task

For each unchecked task, in execution order:

#### 3a. Read the Task

Parse the task line to extract:
- **ID**: e.g., `T003`
- **Markers**: e.g., `[P]` (optional)
- **Description**: The implementation instruction, including file paths

#### 3b. Load Relevant Source Code

Before implementing, read the source files referenced in the task description. Also consult:
- `spec.md` for the relevant requirements and scenarios
- Any existing code in files being modified

**Do not guess** — read the actual code before making changes.

#### 3c. Implement the Task

Execute the implementation described in the task. Follow these principles:

- **Follow the spec**: Implement exactly what the requirements describe. Refer to GIVEN/WHEN/THEN scenarios for expected behavior.
- **Follow the constitution**: Respect project principles and constraints from `fab/constitution.md`.
- **Follow existing patterns**: Match the codebase's existing style, conventions, and patterns.
- **Be precise**: Create, modify, or delete exactly the files described. Do not make unrelated changes.

#### 3d. Run Relevant Tests

After completing the task, run tests relevant to the code just changed:

- If the task modified a module with an associated test file, run that test file
- If the project has a test runner configured in `fab/config.yaml`, use it
- If the task description mentions specific tests to run, run those
- If no specific tests apply, run a broader test suite to check for regressions

**If tests fail:**
1. Analyze the failure
2. Fix the implementation (not the test, unless the test is wrong per the spec)
3. Re-run the tests
4. Repeat until tests pass
5. Only proceed to the next task after tests pass

**If no tests exist** for the changed code, note this and proceed — do not block implementation.

#### 3e. Mark Task Complete

Immediately after the task passes verification:

1. Edit `fab/changes/{name}/tasks.md` — change `- [ ] T{NNN}` to `- [x] T{NNN}` for the completed task
2. Do NOT batch these updates — mark each task complete as soon as it is done

#### 3f. Update `.status.yaml` Progress

After each task completion, update `.status.yaml`:

1. Set `stage` to `apply` (if not already)
2. Set `progress.apply` to `active` (if not already)
3. Update `last_updated` to the current ISO 8601 timestamp

This provides a running progress signal even if the agent is interrupted mid-run.

### Step 4: Completion

After all tasks are executed and marked `[x]`:

1. Update `.status.yaml`:
   - Set `progress.apply` to `done`
   - Update `last_updated` to the current ISO 8601 timestamp
2. Output a summary of what was implemented

---

## Output

### Starting Fresh (no tasks completed yet)

```
Starting implementation. {N} tasks remaining.

Executing T001: {description}...
✓ T001 complete. Tests passed.

Executing T002: {description}...
✓ T002 complete. Tests passed.

...

Executing T{NNN}: {description}...
✓ T{NNN} complete. Tests passed.

All {N} tasks complete. Implementation finished.

Next: /fab-review
```

### Resuming (some tasks already done)

```
Resuming implementation. {M} of {N} tasks already complete, {R} remaining.

Executing T005: {description}...
✓ T005 complete. Tests passed.

...

All {N} tasks complete. Implementation finished.

Next: /fab-review
```

### Test Failure During Task

```
Executing T003: {description}...
✗ T003: test failure in {test file}
  - {failure description}

Fixing: {description of fix}...

Re-running tests...
✓ T003 complete. Tests passed after fix.
```

### All Tasks Already Complete

```
All tasks are already complete. Nothing to do.

Next: /fab-review
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Preflight script exits non-zero | Abort with the stderr message from `fab-preflight.sh` |
| `progress.tasks` is not `done` | Abort with: "Tasks stage is not complete. Run /fab-continue or /fab-ff to generate tasks first." |
| `tasks.md` missing | Abort with: "No tasks.md found. Run /fab-continue or /fab-ff to generate tasks first." |
| Template/spec file missing during implementation | Report which file is missing; do not guess content |
| Test failure | Fix implementation, re-run tests, repeat until passing |
| All tasks already `[x]` | Report completion, set progress.apply to done, output Next line |
| Task references a file that doesn't exist | Create it if the task says "create"; otherwise report the issue |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **Yes** — sets `stage` to `apply`, progresses to `done` when all tasks complete |
| Idempotent? | **Yes** — safe to re-invoke; picks up from first unchecked task |
| Modifies tasks.md? | **Yes** — marks tasks `[x]` as they complete |
| Modifies source code? | **Yes** — implements the actual changes described in tasks |
| Updates `.status.yaml`? | **Yes** — sets stage, progress, and last_updated after each task |

---

## Next Steps Reference

After `/fab-apply` completes:

`Next: /fab-review`
