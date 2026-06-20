---
name: git-branch
description: "Create or switch to the git branch matching the active (or specified) change. Unmatched explicit names fall back to a standalone branch with that literal name."
allowed-tools: Bash(git:*)
---

# /git-branch [change-name]

> Branch naming conventions are defined in `_preamble.md` § Naming Conventions.

## Contents

- Arguments
- Behavior
- Output
- Error Handling
- Key Properties

Create or check out a git branch named `{change-name}` for the active or specified change. When an explicit argument doesn't match any change, falls back to creating a standalone branch with the literal name. Does not modify fab state.

---

## Arguments

- **`<change-name>`** *(optional)* — target a specific change. If omitted, uses the active change resolved via `.fab-status.yaml`. Supports full folder names, partial slug matches, or any substring (resolved via `fab change resolve`).

---

## Behavior

### Step 1: Check Git Repo

Verify inside a git repository:

```bash
git rev-parse --is-inside-work-tree >/dev/null 2>&1
```

If not in a git repo:

```
Not inside a git repository.
```

STOP.

### Step 2: Resolve Change Name

If `<change-name>` provided:

```bash
fab change resolve "<change-name>"
```

If not provided, resolve from `.fab-status.yaml`:

```bash
fab change resolve
```

If resolution fails:

- **If no argument was provided**: display `fab change resolve`'s stderr and STOP.
- **If an explicit argument was provided**, distinguish the two failure modes by stderr (both exit 1):
  - **Ambiguous multi-match** — stderr contains `Multiple changes match`: matches exist, the reference was just ambiguous. STOP with the candidate list from stderr — do NOT create any branch:

```
Ambiguous change reference '{argument}' — multiple changes match:
  {candidate list from stderr}
Re-run /git-branch with a more specific name. No branch created.
```

  - **True no-match** — stderr contains `No change matches` (or any other resolution failure): enter **standalone fallback** — use the raw argument as a literal branch name. Print:

```
No matching change found — using standalone branch '{name}'
```

Set `standalone = true` and proceed to Step 3.

### Step 3: Derive Branch Name

**If standalone**: use the raw argument as-is — no prefix, no transformation:

```
branch_name = {raw_argument}
```

**Otherwise** (change resolved): use the change name directly:

```
branch_name = {resolved_change_name}
```

### Step 4: Context-Dependent Action

<!-- Keep these cases in sync with fab-new.md Step 11 — same cases, same commands, same report strings (incl. the rename guard, the remote-only --track case, and the dirty-tree note). One deliberate divergence lives OUTSIDE the shared cases: fab-new derives {dirty_count} excluding fab/changes/{name}/ (its own just-created artifacts); git-branch counts the full porcelain output. -->

Get the current branch and the dirty-tree count:

```bash
git branch --show-current
git status --porcelain | wc -l    # {dirty_count} — used for the non-blocking note below
```

Check if the target branch already exists locally, and whether it exists on the remote:

```bash
git rev-parse --verify "{branch_name}" >/dev/null 2>&1
git rev-parse --verify "origin/{branch_name}" >/dev/null 2>&1
```

> **Dirty-tree note** (non-blocking — never prompt, never stash): when `{dirty_count}` > 0 AND the action below creates or renames a branch (`git checkout -b` / `git branch -m`), the uncommitted work rides onto the new branch. Append to the Step 5 report line: ` — note: {dirty_count} uncommitted change(s) carried over from {old_branch}`.

**If already on the target branch**: No git operation.

```
Branch: {branch_name} (already active)
```

STOP.

**If the target branch exists locally but is not current**: Switch to it.

```bash
git checkout "{branch_name}"
```

Report: `Branch: {branch_name} (checked out)`

STOP.

**If the target branch exists only on the remote** (local verify fails, `origin/{branch_name}` verify succeeds): check it out tracking the remote branch — do NOT recreate a divergent local with `checkout -b`.

```bash
git checkout --track "origin/{branch_name}"
```

Report: `Branch: {branch_name} (checked out, tracking origin/{branch_name})`

STOP.

**If on `main` or `master`**: Auto-create the branch without prompting.

```bash
git checkout -b "{branch_name}"
```

**If on any other branch**: Check upstream tracking to decide action:

```bash
upstream=$(git config "branch.$(git branch --show-current).remote" 2>/dev/null || true)
```

- **No upstream** (local-only branch) — **rename guard**: rename only when the current branch does not belong to a *different* change. Check what the current branch name resolves to:

```bash
fab change resolve "$(git branch --show-current)" 2>/dev/null
```

  - **Resolution fails** (current branch matches no change — e.g., a disposable `wt create` name) **or resolves to the SAME change being branched** (e.g., a worktree placeholder named with the change's own ID): rename the current branch:

```bash
git branch -m "{branch_name}"
```

  - **Resolution succeeds and matches a different change** (the current branch is another change's local-only branch — e.g., after `/fab-switch`): do NOT rename it away. Create a new branch, leaving the current one intact:

```bash
git checkout -b "{branch_name}"
```

  > Known caveat: the `checkout -b` fallback inherits the old change's HEAD — unpushed commits from the previous change carry over onto the new branch.

- **Has upstream** (branch has been pushed) — create a new branch, leaving the current one intact:

```bash
git checkout -b "{branch_name}"
```

### Step 5: Report

```
Branch: {branch_name} (created|checked out|checked out, tracking origin/{branch_name}|renamed from {old_branch}|created, leaving {old_branch} intact|already active)[ — note: {dirty_count} uncommitted change(s) carried over from {old_branch}]
```

The trailing note appears only on the create/rename actions with a dirty tree (see Step 4's dirty-tree note).

---

## Output

```
Branch: {branch_name} (created|checked out|checked out, tracking origin/{branch_name}|renamed from {old_branch}|created, leaving {old_branch} intact|already active)[ — note: {dirty_count} uncommitted change(s) carried over from {old_branch}]
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Not in a git repo | Report and stop |
| Change name resolution fails (no argument) | Display `fab change resolve`'s error and stop |
| Resolution ambiguous (explicit argument, stderr `Multiple changes match`) | STOP with the candidate list — no branch created |
| Change name resolution fails (explicit argument, no match) | Standalone fallback — use literal argument as branch name |
| `git checkout` fails (e.g., uncommitted conflicts) | Report the git error. No fab state modified. |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Idempotent? | Yes — checking out an already-active branch is a no-op |
| Modifies `.fab-status.yaml`? | No |
| Modifies `.status.yaml`? | No |
| Modifies git state? | Yes — may create, checkout, or rename a branch |
| Requires config/constitution? | No |
