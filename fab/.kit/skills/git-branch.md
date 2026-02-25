---
name: git-branch
description: "Create or switch to the git branch matching the active (or specified) change."
model_tier: fast
allowed-tools: Bash(git:*), Bash(yq:*)
---

# /git-branch [change-name]

Create or check out a git branch named `{branch_prefix}{change-name}` for the active or specified change. Does not modify fab state.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change. If omitted, uses the active change from `fab/current`. Supports full folder names, partial slug matches, or any substring (resolved via `changeman.sh resolve`).

---

## Behavior

### Step 1: Read Config

Read `fab/project/config.yaml` for `git.enabled` and `git.branch_prefix`.

If `git.enabled` is `false`:

```
Git integration is disabled (git.enabled: false in config.yaml)
```

STOP — do not execute any git commands.

### Step 2: Check Git Repo

Verify inside a git repository:

```bash
git rev-parse --is-inside-work-tree >/dev/null 2>&1
```

If not in a git repo:

```
Not inside a git repository.
```

STOP.

### Step 3: Resolve Change Name

If `<change-name>` provided:

```bash
bash fab/.kit/scripts/lib/changeman.sh resolve "<change-name>"
```

If not provided, resolve from `fab/current`:

```bash
bash fab/.kit/scripts/lib/changeman.sh resolve
```

If resolution fails: display changeman's stderr and STOP.

### Step 4: Derive Branch Name

```
branch_name = {git.branch_prefix}{resolved_change_name}
```

### Step 5: Context-Dependent Action

Get the current branch:

```bash
git branch --show-current
```

**If already on the target branch**: No git operation.

```
Branch: {branch_name} (already active)
```

STOP.

**If on `main` or `master`**: Auto-create the change branch without prompting.

```bash
git checkout -b "{branch_name}"
```

**If on any other branch**: Present options:

1. **Create new branch** — `git checkout -b "{branch_name}"`
2. **Adopt this branch** — no git operation, acknowledge the current branch as the working branch (default)
3. **Skip** — cancel, no action taken

### Step 6: Report

```
Branch: {branch_name} (created|checked out|adopted|already active)
```

For "Skip": `No branch change.`

---

## Output

```
Branch: {branch_name} (created|checked out|adopted|already active)
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `git.enabled` is `false` | Report and stop |
| Not in a git repo | Report and stop |
| Change name resolution fails | Display changeman's error and stop |
| `git checkout` fails (e.g., uncommitted conflicts) | Report the git error. No fab state modified. |
| `fab/project/config.yaml` not found | Assume `git.enabled: true`, `branch_prefix: ""` |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Idempotent? | Yes — checking out an already-active branch is a no-op |
| Modifies `fab/current`? | No |
| Modifies `.status.yaml`? | No |
| Modifies git state? | Yes — may create or checkout a branch |
| Requires config/constitution? | Config only (`git.enabled`, `git.branch_prefix`) |
