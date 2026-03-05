# Intake: Decouple Git from Fab Switch

**Change**: 260224-vx4k-decouple-git-from-fab-switch
**Created**: 2026-02-24
**Status**: Draft

## Origin

> Decouple git branch operations from fab-switch. Extract branch create/checkout into a new /git-branch command. fab-switch should only write fab/current (no git ops). /git-branch becomes a standalone command that creates or checks out a branch matching the active change name.

Interaction mode: Conversational — preceded by `/fab-discuss` session with deep analysis of git coupling across the entire codebase. User reviewed a comprehensive coupling map of all git touchpoints in skills, scripts, specs, and templates. Four proposals were presented; user selected Proposals 1 and 2 (split changeman.sh switch, extract branch prompts into a new command) and explicitly rejected Proposals 3 and 4 (pipeline isolation abstraction, config-based user identity).

Key decisions from discussion:
- Both `/git-branch` and `/git-pr` should remain commands (skills), not shell scripts — commands are invocable mid-conversation in Claude chat sessions, preserving the interactive affordance.
- `/git-branch` can be called at any point in the pipeline — it's not tied to change activation.
- Branch switching can now happen just before `/git-pr`, decoupling it from the `fab-switch` moment.

## Why

`fab-switch` currently does two conceptually independent things: (1) set the active change pointer (`fab/current`), and (2) create or checkout a git branch. This coupling creates several problems:

1. **Conceptual conflation** — activating a change and switching git branches are separate concerns. A user may want to activate a change without touching git (e.g., reviewing artifacts, running `/fab-status`, or working in a non-git context).
2. **Interactive friction** — `fab-switch` presents branch prompts (create/adopt/skip) that interrupt the change activation flow, even when the user only wants to point at a different change.
3. **Flag complexity** — `--no-branch-change` exists as a negative flag specifically to opt out of git, which is a code smell indicating the operations should be separate.
4. **Pipeline workaround** — `dispatch.sh` already passes `--no-branch-change` to avoid git interference in worktree-based pipelines, confirming that the coupling is a liability.

If left as-is, every new feature touching change activation must also consider git branch side effects, and the `--no-branch-change` escape hatch proliferates.

## What Changes

### 1. `fab-switch` becomes git-free

Remove all git branch operations from `fab-switch.md` and from `changeman.sh switch`:

- **`fab-switch.md`**: Remove the `## Branch Integration` section (lines 68-79), the `--branch <name>` argument, and the `--no-branch-change` argument. The skill becomes: resolve change name → write `fab/current` → display stage summary.
- **`changeman.sh cmd_switch()`**: Remove lines 205-231 (git branch integration block). Remove the `branch_status` variable and the `Branch:` output line. The function becomes: resolve → write pointer → derive stage → output summary.
- **Output format** changes — the `Branch:` line is removed from the canonical output:

```
fab/current → {name}

Stage:  {display_stage} ({N}/6) — {state}
Next:   {routing_stage} (via {default_command})
```

- **Key Properties table**: `Modifies git state?` changes from `Yes` to `No`.
- **Error Handling table**: Remove "Git branch creation/checkout fails" row.

### 2. New `/git-branch` command

A new skill file `fab/.kit/skills/git-branch.md` that handles branch create/checkout:

**Arguments**:
- `[change-name]` *(optional)* — if omitted, uses active change from `fab/current`

**Behavior**:
1. Read `config.yaml` for `git.enabled` and `git.branch_prefix`
2. If `git.enabled` is false → report and stop
3. Resolve change name (via `changeman.sh resolve` or from `fab/current`)
4. Derive branch name: `{branch_prefix}{change-name}`
5. If branch exists → `git checkout {branch}`
6. If branch doesn't exist → `git checkout -b {branch}`
7. Report result

**Interactive prompts** — the context-dependent prompts currently in `fab-switch` (on main → auto-create, on wt/* → prompt, on feature → prompt) move here. `/git-branch` is the right place for "what should I do with the current branch?" decisions because it's explicitly about git.

**Output format**:
```
Branch: {branch_name} (created|checked out)
```

**Properties**: Read-only on fab state (never writes `fab/current`). Git-mutating. Idempotent (checking out an already-active branch is a no-op).

### 3. `/git-pr` awareness

Update `/git-pr` to check whether the current git branch matches the active change's expected branch. If on `main`/`master` and there's an active change, `/git-pr` could suggest running `/git-branch` first (or call it internally). This is a soft integration — `/git-pr` already has a branch guard that stops on main/master.

### 4. `dispatch.sh` cleanup

`dispatch.sh` already passes `--no-branch-change` to `/fab-switch` (line 217). After this change, the flag no longer exists and can be removed from the `tmux send-keys` call. The dispatch script handles its own branch creation via `create_worktree()` which is independent of fab-switch.

### 5. Spec and architecture doc updates

- `docs/specs/architecture.md` Git Integration section (lines 324-347): Update to reflect that `/fab-switch` no longer handles branches. Note that `/git-branch` is the branch management command.
- `docs/specs/skills.md`: Add `/git-branch` skill documentation.

## Affected Memory

- `fab-workflow/change-lifecycle`: (modify) Change activation no longer includes git branch operations
- `fab-workflow/execution-skills`: (modify) New `/git-branch` command, updated `/fab-switch` behavior

## Impact

- **`fab/.kit/skills/fab-switch.md`** — Remove branch integration section, `--branch` and `--no-branch-change` flags
- **`fab/.kit/scripts/lib/changeman.sh`** — Remove git logic from `cmd_switch()` (lines 192-231), remove `Branch:` output line
- **`fab/.kit/skills/git-branch.md`** — New file
- **`fab/.kit/skills/git-pr.md`** — Minor update: suggest `/git-branch` when on main with active change
- **`fab/.kit/scripts/pipeline/dispatch.sh`** — Remove `--no-branch-change` from send-keys call (line 217)
- **`docs/specs/architecture.md`** — Update Git Integration section
- **`docs/specs/skills.md`** — Add `/git-branch` skill entry

No breaking changes to user workflow — `/git-branch` replaces implicit branch switching with explicit invocation. Users who relied on automatic branch creation during `/fab-switch` will need to call `/git-branch` separately (or it can be suggested in `/fab-switch` output).

## Open Questions

- Should `/fab-switch` output suggest `/git-branch` as a follow-up when `git.enabled` is true? (e.g., `Tip: run /git-branch to create or switch to the matching branch`)
- Should `/git-pr` auto-invoke `/git-branch` when it detects the user is on main with an active change, or just warn and stop?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | fab-switch writes only fab/current, no git ops | Discussed — user explicitly stated "the only thing that fab-switch changes is fab/current (and not the branch)" | S:95 R:90 A:95 D:95 |
| 2 | Certain | New /git-branch is a command (skill), not a shell script | Discussed — user chose commands over scripts: "keep everything a command. That way we can use this while being in a Claude chat session" | S:95 R:85 A:90 D:90 |
| 3 | Certain | /git-pr remains a command | Discussed — user explicitly confirmed commands for both git-branch and git-pr | S:95 R:85 A:90 D:90 |
| 4 | Certain | Skip pipeline isolation abstraction (Proposal 3) | Discussed — user explicitly said "Don't do 3 and 4" | S:95 R:95 A:95 D:95 |
| 5 | Certain | Skip config-based created_by (Proposal 4) | Discussed — user explicitly said "Don't do 3 and 4" | S:95 R:95 A:95 D:95 |
| 6 | Confident | Interactive branch prompts move from fab-switch to git-branch | Inferred — the prompts are about git branch decisions, so they belong in the git-focused command | S:70 R:80 A:85 D:75 |
| 7 | Confident | /git-pr suggests /git-branch rather than auto-invoking it | Inferred — keeping git-pr autonomous (no prompts) is a stated design rule; auto-invocation would add a dependency | S:60 R:75 A:80 D:70 |
| 8 | Confident | dispatch.sh just removes --no-branch-change flag, no other changes | dispatch.sh handles its own branching via create_worktree(), independent of fab-switch | S:75 R:85 A:80 D:80 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
