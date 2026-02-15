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

Loads `fab/config.yaml` (for `git.enabled`, `git.branch_prefix`), `fab/changes/` folder names, and matched change's `.status.yaml`. Does NOT load constitution, memory, or specs.

---

## Behavior

### No Argument Flow

1. Scan `fab/changes/` (exclude `archive/`)
2. If no folders: `No active changes found. Run /fab-new <description> to start one.`
3. If folders exist: list with stages as numbered options, wait for selection

### Argument Flow

Match `<change-name>` against folder names (case-insensitive substring):
- **Exact/single match** → use that folder
- **Multiple matches** → list matches with stages, ask user to pick
- **No match** → list all available changes, inform user

### Deactivation Flow (`--blank`)

1. Delete `fab/current` (no-op if already absent)
2. If `--branch` also provided: attempt checkout (deactivation succeeds regardless of checkout result)
3. Display confirmation

### Switch Flow

Once a single change is identified:

1. **Write** folder name to `fab/current` (overwrites previous)
2. **Branch Integration** (see below)
3. **Read `.status.yaml`** for stage and progress
4. **Display confirmation** with stage number (intake=1, spec=2, tasks=3, apply=4, review=5, hydrate=6)
5. **Suggest next command** based on stage:

| Stage | Suggested next |
|-------|---------------|
| `intake` (active) | `/fab-continue or /fab-clarify` |
| `spec` (active/done) | `/fab-continue or /fab-ff or /fab-clarify` |
| `tasks` (done) | `/fab-continue` |
| `apply` (done) | `/fab-continue` |
| `review` (done) | `/fab-continue` |
| `review` (failed) | `/fab-continue (re-review after fixes)` |

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

Canonical format:

```
fab/current now points to {name}

Stage:  {stage} ({N}/6)
Branch: {name} (created|adopted)

Next: /fab-continue
```

Branch line omitted if skipped. Deactivation shows `No active change.` with optional `Branch:` line. Already-blank shows `No active change (already blank).`

---

## Error Handling

| Condition | Action |
|-----------|--------|
| No changes exist | "No active changes found. Run /fab-new." |
| Matched folder missing `.status.yaml` | Switch anyway, warn: "Warning: .status.yaml not found — change may be corrupted." |
| `fab/changes/` doesn't exist | "fab/changes/ not found. Run /fab-init." |
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
