---
name: fab-init
description: "Bootstrap fab/ directory structure. Safe to re-run."
model_tier: fast
---

# /fab-init

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Exception**: `/fab-init` skips the "Always Load" context layer (config and constitution don't exist yet on first run). Load them only if they already exist (re-run scenario).

---

## Purpose

Bootstrap `fab/` in an existing project. Safe to run repeatedly — structural artifacts are created once (skipped if they already exist) and symlinks are repaired if broken.

---

## Pre-flight Check

Before doing anything else, verify the kit exists:

1. Check that `fab/.kit/` directory exists
2. Check that `fab/.kit/VERSION` file exists and is readable

**If either check fails, STOP immediately.** Output this message and do nothing else:

> `fab/.kit/ not found. Copy the kit directory into fab/.kit/ first — see the Getting Started guide.`

Do NOT create partial structure. Do NOT create `fab/config.yaml`, `fab/constitution.md`, or any other file. The kit must be in place before init can run.

**If arguments are provided** (e.g., URLs or file paths), STOP and output:

> `Did you mean /fab-hydrate? /fab-init no longer accepts source arguments.`

Do NOT proceed with structural bootstrap when arguments are passed — this prevents confusion with the old interface.

---

## Behavior

### Delegation Pattern

`/fab-init` delegates structural setup to `fab/.kit/scripts/fab-setup.sh` (invoked in step 1f) and only adds interactive/configuration artifacts on top. This separation keeps the script automatable for CI and bootstrap workflows while the skill handles project-specific configuration that requires user input.

| Responsibility | Owner | Why |
|---|---|---|
| Directories, skeleton files, symlinks, .gitignore, .envrc | `fab-setup.sh` | Scriptable, automatable, no user input needed |
| `config.yaml` (interactive) | `/fab-init-config` (delegated by `/fab-init`) | Single source of truth for config generation and updates |
| `constitution.md` (interactive) | `/fab-init-constitution` (delegated by `/fab-init`) | Single source of truth for constitution generation and amendments |
| Invoking `fab-setup.sh` | `/fab-init` (step 1f) | Ensures structural setup runs as part of init |

Steps 1c–1e below have idempotent guards (`if not exists`) so they gracefully skip when `fab-setup.sh` has already created the structural artifacts.

### Phase 1: Structural Bootstrap

Each step is **idempotent** — skip if the artifact already exists and is valid. On re-run, verify and repair rather than recreate.

#### 1a. `fab/config.yaml`

If `fab/config.yaml` does **not** exist:

**Delegate to `/fab-init-config`** in create mode. This ensures a single source of truth for config generation logic — see `/fab-init-config` for the full interactive flow and template.

If `fab/config.yaml` **already exists**: report "config.yaml already exists — skipping" and move on.

#### 1b. `fab/constitution.md`

If `fab/constitution.md` does **not** exist:

**Delegate to `/fab-init-constitution`** in create mode. This ensures a single source of truth for constitution generation logic — see `/fab-init-constitution` for the full interactive flow and template.

If `fab/constitution.md` **already exists**: report "constitution.md already exists — skipping" and move on.

#### 1c. `fab/docs/index.md`

If `fab/docs/index.md` does **not** exist:

1. Create `fab/docs/` directory if needed
2. Create `fab/docs/index.md` with an empty index:

```markdown
# Documentation Index

<!-- This index is maintained by /fab-continue (archive) when changes are completed. -->
<!-- Each domain gets a row linking to its docs. -->

| Domain | Description | Docs |
|--------|-------------|------|
```

If `fab/docs/index.md` **already exists**: report "docs/index.md already exists — skipping" and move on.

#### 1d. `fab/design/index.md`

If `fab/design/index.md` does **not** exist:

1. Create `fab/design/` directory if needed
2. Create `fab/design/index.md` with an empty index:

```markdown
# Specifications Index

> **Specs are pre-implementation artifacts** — what you *planned*. They capture conceptual design
> intent, high-level decisions, and the "why" behind features. Specs are human-curated,
> flat in structure, and deliberately size-controlled for quick reading.
>
> Contrast with [`fab/docs/index.md`](../docs/index.md): docs are *post-implementation* —
> what actually happened. Docs are the authoritative source of truth for system behavior,
> maintained by `/fab-continue` (archive) hydration.
>
> **Ownership**: Specs are written and maintained by humans. No automated tooling creates or
> enforces structure here — organize files however makes sense for your project.

| Spec | Description |
|------|-------------|
```

If `fab/design/index.md` **already exists**: report "design/index.md already exists — skipping" and move on.

#### 1e. `fab/changes/`

If `fab/changes/` directory does **not** exist:

1. Create `fab/changes/` directory
2. Create `fab/changes/archive/` subdirectory (pre-created so archive behavior doesn't need a separate `mkdir`)
3. Create `fab/changes/.gitkeep` to ensure git tracks the empty directory

If `fab/changes/` **already exists**: ensure `fab/changes/archive/` exists (create if missing), then report "changes/ already exists — skipping" and move on.

#### 1f. `.claude/skills/` Symlinks

Run `fab/.kit/scripts/fab-setup.sh` to create or repair all skill symlinks and directories. This script is the **single source of truth** for the structural bootstrap — it handles directories, symlinks, docs index, and `.gitignore`.

The script discovers skills dynamically by globbing `fab/.kit/skills/fab-*.md` — no hardcoded list to maintain. Each discovered skill gets a subdirectory symlink:

```
.claude/skills/fab-{name}/SKILL.md → ../../../fab/.kit/skills/fab-{name}.md
```

If the script cannot be executed (e.g., Windows without bash), perform the equivalent manually:

1. For each `fab-*.md` file in `fab/.kit/skills/`, create `.claude/skills/fab-{name}/SKILL.md` as a relative symlink to `../../../fab/.kit/skills/fab-{name}.md`
2. Skip `_context.md` (internal, not a skill)
3. If a symlink already exists and resolves correctly (`test -e` passes), skip it
4. If a symlink is broken (dangling), remove and recreate it

**Important**: Use relative paths so symlinks work after cloning the repo. Do NOT use absolute paths.

**Important**: Do NOT modify or remove any existing content in `.claude/skills/` (e.g., `commit/`, `dev-browser/`, `prd/`).

Report how many symlinks were created, repaired, or already valid.

#### 1g. `.gitignore` — append `fab/current`

1. Read `.gitignore` at the project root (create it if it doesn't exist)
2. Check if `fab/current` is already listed (exact line match or with trailing whitespace/comment)
3. If not present, append `fab/current` on a new line
4. If already present, skip

---

## Output

### First Run (fresh bootstrap)

```
Found fab/.kit/ (v{VERSION}). Initializing project...
{config.yaml prompts and creation}
{constitution.md generation}
Created: fab/config.yaml
Created: fab/constitution.md
Created: fab/docs/index.md
Created: fab/design/index.md
Created: fab/changes/
Created: 11 symlinks in .claude/skills/
Updated: .gitignore (added fab/current)
fab/ initialized successfully.

Next: /fab-new <description> or /fab-hydrate <sources>
```

### Re-run (structural health check)

```
Found fab/.kit/ (v{VERSION}). Verifying structure...
config.yaml — OK
constitution.md — OK
docs/index.md — OK
design/index.md — OK
changes/ — OK
Symlinks: 11/11 valid (repaired 1)
.gitignore: fab/current present
fab/ structure verified.
```

---

## Idempotency Guarantee

This skill is safe to run any number of times:

- **Config and constitution**: Created once, never overwritten on re-run
- **Docs index**: Created once, never touched on re-run
- **Design index**: Created once, never touched on re-run
- **Changes directory**: Created once, never touched on re-run
- **Symlinks**: Verified and repaired on every run — broken symlinks are fixed, valid ones are left alone
- **`.gitignore`**: Entry is appended only if not already present

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/.kit/` missing | Abort immediately with guidance message. Do NOT create any files. |
| `fab/.kit/VERSION` unreadable | Abort with: "fab/.kit/VERSION not found or unreadable — kit may be corrupted." |
| Arguments provided | Abort with: "Did you mean /fab-hydrate? /fab-init no longer accepts source arguments." |
| Symlink target missing | Report which skill file is missing in `fab/.kit/skills/` — do NOT create a broken symlink |

---

## Related Commands

| Command | Description |
|---------|-------------|
| `/fab-init-constitution` | Create or amend the project constitution with semantic versioning |
| `/fab-init-config` | Create or update config.yaml interactively, preserving comments |
| `/fab-init-validate` | Validate config.yaml and constitution.md structural correctness |

---

Next: `/fab-new <description>` or `/fab-hydrate <sources>`
