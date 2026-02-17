---
name: fab-switch
description: "Switch the active change to a different one. Lists available changes when called with no argument. Handles branch integration."
model_tier: fast
---

# /fab-switch [change-name] [--blank] [--branch <name>] [--no-branch-change]

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.

---

## Arguments

- **`<change-name>`** *(optional)* — full or partial name of the change to switch to. Supports full folder names, partial slug matches, or any substring. Case-insensitive.
- **`--blank`** — deactivate the current change by deleting `fab/current`. Can combine with `--branch` to also switch branches. Mutually exclusive with `<change-name>` (if both given, prefer `--blank`).
- **`--branch <name>`** — explicit branch name. Creates if new, checks out if existing. Skips interactive branch prompt.
- **`--no-branch-change`** — skip all branch integration.

If no argument (and no `--blank`): list all active changes and ask user to pick.

---

## Context Loading

Loads `fab/config.yaml` (for `git.enabled`, `git.branch_prefix`) and matched change's `.status.yaml`. Name resolution and switch logic are delegated to `fab/.kit/scripts/lib/changeman.sh`. Does NOT load constitution, memory, or specs.

---

## Behavior

### No Argument Flow

1. Scan `fab/changes/` (exclude `archive/`)
2. If no folders: `No active changes found. Run /fab-new <description> to start one.`
3. If folders exist: list with stages as numbered options, wait for selection

### Argument Flow

Delegate to `changeman.sh switch` via a single Bash call:

```bash
bash fab/.kit/scripts/lib/changeman.sh switch "<change-name>"
```

If changeman exits 0: display the stdout output (contains name, stage, branch, next command).

If changeman exits 1 and stderr contains "Multiple changes match": parse the comma-separated folder names from stderr, list them with stages as numbered options, ask user to pick. After selection, run `changeman.sh switch "<selected>"`.

If changeman exits 1 and stderr contains "No change matches": list all available changes, inform user.

### Deactivation Flow (`--blank`)

Run `changeman.sh switch --blank`. If `--branch` also provided: attempt checkout after deactivation (deactivation succeeds regardless of checkout result). Display changeman's stdout output.

### Switch Flow

`changeman.sh switch` handles the full flow internally:
1. Resolves the change name
2. Writes `fab/current`
3. Performs git branch integration (respecting `config.yaml`)
4. Outputs structured summary with stage and next command

The skill displays changeman's stdout directly.

---

## Branch Integration

**Skip entirely if**: `--no-branch-change`, or `git.enabled` is `false`, or not inside a git repo.

**If `--branch <name>` provided**: Use name directly. Check out if exists, `git checkout -b` if not. Skip interactive prompt.

**If no `--branch`** (interactive):

- **On `main`/`master`**: Auto-create `{branch_prefix}{change-name}` — no prompt (Certain grade per SRAD: high R, A, D).
- **On `wt/*` branch**: Prompt with options: **Create new branch** (default), **Adopt this branch**, **Skip**.
- **On feature branch**: Prompt with options: **Adopt this branch** (default), **Create new branch**, **Skip**.

---

## Output

Output is passthrough from `changeman.sh switch`. Canonical format:

```
fab/current → {name}

Stage:  {stage} ({N}/6)
Branch: {name} (created|checked out)

Next: {per state table}
```

Branch line omitted if git disabled or not in a repo. Deactivation shows `No active change.`. Already-blank shows `No active change (already blank).`

---

## Error Handling

| Condition | Action |
|-----------|--------|
| No changes exist | "No active changes found. Run /fab-new." |
| Matched folder missing `.status.yaml` | Switch anyway, warn: "Warning: .status.yaml not found — change may be corrupted." |
| `fab/changes/` doesn't exist | "fab/changes/ not found. Run /fab-setup." |
| `fab/config.yaml` not found | Skip branch integration |
| Git branch creation/checkout fails | Report error, continue without branch change. Switch still completes. |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — changes only the active pointer |
| Idempotent? | Yes |
| Modifies `fab/current`? | Yes (writes name, or deletes with `--blank`) |
| Modifies `.status.yaml`? | No |
| Modifies git state? | Yes — may create/checkout branch |
| Requires config/constitution? | Config only (`git.enabled`, `git.branch_prefix`) |
