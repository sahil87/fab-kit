# Intake: Consolidate Setup & Upgrade Flow

**Change**: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow
**Created**: 2026-02-16
**Status**: Draft

## Origin

> User analysis of the setup/update/upgrade flow identified excessive surface area: three user-facing commands (`/fab-init`, `/fab-update`, `fab-upgrade.sh`) plus a hidden internal dependency (`sync-workspace.sh`) for what is conceptually two operations (set up a project, upgrade the kit). The names "init", "update", and "upgrade" are near-synonyms causing confusion about ordering and responsibility. A manual handoff between `fab-upgrade.sh` and `/fab-update` creates a dropped-ball failure mode. The design was refined through interactive discussion, arriving at a two-entry-point architecture: `/fab-setup` (LLM skill) and `fab-upgrade.sh` (bash script), with `fab-sync.sh` as a promoted standalone script solving the chicken-and-egg problem (symlinks must exist before any LLM skill can run).

## Why

The current flow has three problems that compound:

1. **Name confusion**: "init" vs "update" vs "upgrade" are near-synonyms. Users can't intuit which to run or in what order. The distinction between "upgrade" (replaces engine) and "update" (applies migrations) requires understanding internals.

2. **Manual handoff**: After `fab-upgrade.sh` finishes, it prints a reminder to run `/fab-update`. This is a dropped ball waiting to happen — user upgrades, sees "done", returns to work, never applies migrations. The two-step flow exists because bash can't easily invoke an LLM, but from the user's perspective it's one operation ("get the latest stuff working").

3. **Overloaded init**: `/fab-init` does four unrelated jobs: bootstrap project structure, manage config.yaml, manage constitution.md, and validate both files. "Init" implies a one-time setup, yet `/fab-init config` is an ongoing management operation.

If we don't fix this, every new user hits the same confusion, and every upgrade risks skipped migrations.

## What Changes

### 1. Promote `sync-workspace.sh` to `fab-sync.sh`

Move `fab/.kit/scripts/lib/sync-workspace.sh` to `fab/.kit/scripts/fab-sync.sh`. This script becomes a first-class user-facing entry point instead of hidden internal plumbing.

- **Why standalone**: Solves the chicken-and-egg problem. `/fab-setup` is an LLM skill that requires symlinks to exist. Symlinks are created by sync. Therefore sync MUST be runnable without an LLM.
- **What it does**: Same as today — creates directories, `fab/VERSION`, skill symlinks (Claude Code, OpenCode, Codex), agent files, `.envrc`, `.gitignore` entries, scaffold files (`docs/memory/index.md`, `docs/specs/index.md`). Idempotent.
- **Internal references**: `fab-upgrade.sh` and `/fab-setup` both call it. Users can also run it directly for repair.

### 2. New `/fab-setup` skill (replaces `/fab-init` + absorbs `/fab-update`)

A single LLM skill at `fab/.kit/skills/fab-setup.md` with four subcommands:

| Subcommand | Purpose |
|------------|---------|
| *(none)* | Full bootstrap: calls `fab-sync.sh` via Bash, then config (create mode), then constitution (create mode) |
| `config [section]` | Create or edit `fab/config.yaml` interactively. Validates after each write. |
| `constitution` | Create or amend `fab/constitution.md` with semantic versioning. Validates after each write. |
| `migrations [file]` | Apply version migrations from `fab/.kit/migrations/`. Absorbs all `/fab-update` logic. Optional `[file]` argument allows targeting a specific migration file. |

**Validation folded in**: No separate `validate` subcommand. Config validates config.yaml after each create/edit (the 8 checks). Constitution validates constitution.md after each create/amend (the 6 checks). Failures reported inline with fix suggestions and revert offer.

**Model tier**: `fast` (same as `/fab-init` today — bootstrap and config management don't need capable-tier).

**Exception**: The `migrations` subcommand may need capable-tier since it reads and executes prose migration instructions. Evaluate whether fast-tier can handle migration complexity, or whether migrations should run at default tier. If fast-tier is insufficient, the skill needs a way to signal that `migrations` runs at a different tier than the other subcommands — or migrations could remain a separate skill invoked internally.

### 3. Update `fab-upgrade.sh`

Minimal changes:
- Call `fab-sync.sh` (new path) instead of `lib/sync-workspace.sh` (old path)
- Update drift message: print "Run `/fab-setup migrations`" instead of "Run `/fab-update`"
- Update missing-VERSION message: print "Run `fab-sync.sh` then `/fab-setup`" instead of "Run `/fab-init`"

### 4. Kill `/fab-init` and `/fab-update`

- Delete `fab/.kit/skills/fab-init.md`
- Delete `fab/.kit/skills/fab-update.md`
- Remove corresponding symlinks: `.claude/skills/fab-init/`, `.claude/skills/fab-update/`
- Remove agent file: `.claude/agents/fab-init.md` (regenerated as `fab-setup.md` by `fab-sync.sh`)
- Clean up OpenCode and Codex counterparts

### 5. Update cross-references

Every reference to `/fab-init` or `/fab-update` across the codebase needs updating:

**Skills** (high priority — directly affect runtime behavior):
- `_context.md` — "Always Load" exception list, Next Steps table
- `fab-new.md` — pre-flight error message ("Run /fab-init first")
- `fab-help.md` — command listing
- `fab-status.md` — version drift message
- `fab-continue.md`, `fab-ff.md`, `fab-fff.md` — any references in error messages or next-steps
- `fab-clarify.md` — if any init references exist

**Scripts**:
- `fab-help.sh` — command catalog
- `fab-upgrade.sh` — drift messages (covered in §3)
- `fab-sync.sh` — no changes needed (it doesn't reference skills by name)
- `preflight.sh` — error messages about missing config/constitution

**Templates**:
- Check all templates for `/fab-init` references in comments

**Memory files** (updated during hydrate):
- `fab-workflow/init.md` — rename to `fab-workflow/setup.md` or heavily rewrite
- `fab-workflow/migrations.md` — update `/fab-update` → `/fab-setup migrations`
- `fab-workflow/distribution.md` — update sync-workspace → fab-sync.sh, update handoff flow
- `fab-workflow/kit-architecture.md` — update directory listing, script descriptions, bootstrap sequence
- `fab-workflow/configuration.md` — update `/fab-init` → `/fab-setup` throughout
- `fab-workflow/index.md` — update file listing

**Specs**:
- `docs/specs/skills.md` — update skill listing
- `docs/specs/architecture.md` — update bootstrap sequence
- `docs/specs/user-flow.md` — update command references
- `docs/specs/overview.md` — if it references init/update

**Other**:
- `README.md` — getting started instructions
- `fab/backlog.md` — item `pr1u` references `/fab-init`
- `fab/config.yaml` — header comments reference `/fab:init`

### 6. Update README.md

Document the simplified flow prominently:

| Situation | Command |
|-----------|---------|
| New project | `fab-sync.sh` then `/fab-setup` |
| Edit config later | `/fab-setup config` |
| Edit constitution later | `/fab-setup constitution` |
| New kit version | `fab-upgrade.sh` then `/fab-setup migrations` |
| Repair broken state | `fab-sync.sh` |

## Affected Memory

- `fab-workflow/init`: (modify) Rename to `setup.md` — rewrite to document `/fab-setup` subcommands, updated delegation pattern (fab-sync.sh), removed validate subcommand, added migrations subcommand
- `fab-workflow/migrations`: (modify) Update `/fab-update` skill references → `/fab-setup migrations`, update two-step flow description, update version drift messages
- `fab-workflow/distribution`: (modify) Update `sync-workspace.sh` → `fab-sync.sh` throughout, update bootstrap instructions, update post-upgrade handoff message
- `fab-workflow/kit-architecture`: (modify) Update directory tree (sync-workspace.sh location, fab-init.md → fab-setup.md, remove fab-update.md), update script descriptions, update bootstrap sequence, update agent file listing
- `fab-workflow/configuration`: (modify) Update all `/fab-init` references → `/fab-setup`, update lifecycle management section

## Impact

**Skill files** (create/delete/modify):
- `fab/.kit/skills/fab-setup.md` — new (content from fab-init.md + fab-update.md migrations logic)
- `fab/.kit/skills/fab-init.md` — delete
- `fab/.kit/skills/fab-update.md` — delete
- `fab/.kit/skills/_context.md` — modify (exception lists, next steps table)
- `fab/.kit/skills/fab-new.md` — modify (error messages)
- `fab/.kit/skills/fab-help.md` — modify (command listing)
- Other skills — modify (grep for `/fab-init` and `/fab-update` references)

**Scripts** (move/modify):
- `fab/.kit/scripts/lib/sync-workspace.sh` → `fab/.kit/scripts/fab-sync.sh` (move + update internal paths)
- `fab/.kit/scripts/fab-upgrade.sh` — modify (call fab-sync.sh, update messages)
- `fab/.kit/scripts/fab-help.sh` — modify (command catalog)
- `fab/.kit/scripts/lib/preflight.sh` — modify (error messages)

**Agent integration** (auto-handled by fab-sync.sh):
- `.claude/skills/fab-init/` → `.claude/skills/fab-setup/` (symlink recreated)
- `.claude/skills/fab-update/` — removed
- `.claude/agents/fab-init.md` → `.claude/agents/fab-setup.md` (regenerated)
- Same pattern for OpenCode and Codex paths

**Testing infrastructure**:
- `src/lib/sync-workspace/` — rename to `src/lib/fab-sync/` (test directory tracks script name). Update symlink, README, and test files to reference `fab-sync.sh` at its new path.

**Documentation** (specs, memory, README):
- 5 memory files modified, 1 renamed
- ~4 spec files modified
- README.md rewritten (getting started section)
- Backlog item `pr1u` updated

**Migration file needed**: A new `fab/.kit/migrations/{FROM}-to-{TO}.md` so existing projects can run `/fab-setup migrations` to update any internal references (though this change primarily affects kit-level files, not project-level files — the migration may only need to update `config.yaml` header comments).

## Open Questions

*(None remaining.)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename /fab-init → /fab-setup | Explicitly agreed in discussion | S:95 R:70 A:95 D:95 |
| 2 | Certain | Absorb /fab-update into /fab-setup migrations | Explicitly agreed in discussion | S:95 R:65 A:95 D:95 |
| 3 | Certain | Promote sync-workspace.sh → fab-sync.sh | Explicitly agreed — chicken-and-egg requires standalone script | S:95 R:75 A:95 D:95 |
| 4 | Certain | Fold validation into config/constitution subcommands | Explicitly agreed — each path validates its own output | S:90 R:80 A:90 D:90 |
| 5 | Certain | fab-upgrade.sh calls fab-sync.sh and prompts for /fab-setup migrations | Explicitly agreed in discussion | S:95 R:70 A:95 D:95 |
| 6 | Confident | fab-setup model_tier stays fast for bootstrap/config/constitution | Same complexity as fab-init which was fast tier | S:70 R:85 A:80 D:75 |
| 7 | Confident | No new migration file needed for project-level files | This change primarily affects kit-level files (skills, scripts); project files (config.yaml, constitution.md) don't reference skill names in their functional content — only in comments | S:60 R:80 A:75 D:70 |
| 8 | Confident | migrations subcommand runs at fast tier along with rest of fab-setup | Existing migrations are mechanical (move dirs, rename files, find-replace strings) — structured instructions with built-in pre-check/verification guardrails. One tier per skill avoids new infrastructure. | S:70 R:80 A:75 D:75 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
