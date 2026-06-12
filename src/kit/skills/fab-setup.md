---
name: fab-setup
description: "Set up a new project, manage config/constitution, or apply version migrations. Safe to re-run."
---

# /fab-setup [subcommand]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.
> **Exception**: `/fab-setup` has subcommand-specific context loading:
> - **bare / config / constitution**: Skip the "Always Load" context layer if files don't exist (first-run). Load them only if they already exist (re-run scenario).
> - **migrations**: Load `fab/project/config.yaml` (MUST exist). Skip Change Context loading — migrations operate on project-level files, not a specific change.

---

## Arguments

- **No arguments** — full structural bootstrap (default behavior)
- **`config [section]`** — create or update `fab/project/config.yaml` interactively. Optional `[section]` skips the menu and edits that section directly. Valid sections: `project`, `source_paths`, `checklist`.
- **`constitution`** — create or amend `fab/project/constitution.md` with semantic versioning
- **`migrations [file]`** — apply version migrations to bring project files in sync with the installed kit version (absorbed from fab-update)
- **`validate`** — redirect message: "Validation is built into `/fab-setup config` and `/fab-setup constitution` — each validates after every edit."

Any unrecognized argument triggers: "Unknown subcommand: {arg}. Valid: config, constitution, migrations. Run `/fab-setup` with no arguments for full setup."

---

## Pre-flight Check

Before doing anything else, verify the kit is accessible:

1. Run `fab kit-path` and check that it exits 0
2. Check that `$(fab kit-path)/VERSION` file exists and is readable

**If either check fails, STOP immediately.** Output: `Kit not found. Run 'fab sync' or 'fab upgrade-repo' to populate the cache.` Do NOT create any files.

### Argument Classification

| First argument | Action |
|----------------|--------|
| *(none)* | Proceed to **Bootstrap Behavior** |
| `config` | Proceed to **Config Behavior** (pass remaining args as section argument) |
| `constitution` | Proceed to **Constitution Behavior** |
| `migrations` | Proceed to **Migrations Behavior** (pass remaining args as file argument) |
| `validate` | Output redirect message and STOP |
| *(anything else)* | Output unknown subcommand message and STOP |

### Command Logging

After the pre-flight check passes, log the command invocation:

```bash
fab log command "fab-setup" 2>/dev/null || true
```

This is best-effort — `fab log` resolves the active change via `.fab-status.yaml` if one exists. Failures are silently ignored.

---

## Bootstrap Behavior

When invoked with no arguments, perform the full structural bootstrap. `/fab-setup` delegates directory/skeleton/deployment creation to `fab sync` (step 1c) while handling interactive config/constitution generation itself.

> **Ordering note**: `fab sync` runs immediately after the interactive config/constitution steps (1a/1b) — it requires `config.yaml`'s `fab_version` to exist. Its scaffolding operations are copy-if-absent / line-ensure merges, so the outcome is identical to the former sync-last order via idempotency.

### Phase 0: Prerequisite Check

Run `fab doctor` as the first step. If doctor exits non-zero, STOP immediately and surface the doctor output to the user. Do NOT create any project artifacts.

This gate applies only to the bare bootstrap flow. Subcommands (`config`, `constitution`, `migrations`) skip this check.

### Phase 1: Structural Bootstrap

Each step is **idempotent** — skip if the artifact already exists and is valid. On re-run, verify and repair rather than recreate.

#### 1a. `fab/project/config.yaml`

If missing, raw template (contains `{PROJECT_NAME}`), or missing the required fields `project.name`/`project.description` (the canonical `fab init` flow writes a `fab_version`-only config.yaml before sync's copy-if-absent runs): execute **Config Behavior** (below) in create mode.
If exists with the required fields and not a raw template: report "config.yaml already exists — skipping".

#### 1b. `fab/project/constitution.md`

If missing or raw template (contains `{Project Name}`): execute **Constitution Behavior** (below) in create mode.
If exists and not a raw template: report "constitution.md already exists — skipping".

#### 1c. `fab sync` — scaffold, directories, deployment, gitignore

Run `fab sync`. The command owns all non-interactive structural setup in one idempotent pass:

- **Skeleton files** (copy-if-absent from `$(fab kit-path)/scaffold/`): `fab/project/context.md`, `fab/project/code-quality.md`, `fab/project/code-review.md`, `docs/memory/index.md`, `docs/specs/index.md` (creating `docs/memory/` and `docs/specs/` as needed)
- **Directories**: `fab/changes/`, `fab/changes/archive/`, `fab/changes/.gitkeep`
- **`fab/.kit-migration-version`** (see 1d)
- **Skill deployment**: copies skills from the cache kit to `.claude/skills/{name}/SKILL.md`
- **`.gitignore`**: line-ensure merge of the kit's fragment (adds `.fab-*`, which covers `.fab-status.yaml`)

**Sync-failure guard**: if `fab sync` exits non-zero, STOP immediately and surface its output — do not continue the bootstrap. (Sync requires `config.yaml`'s `fab_version`, which step 1a guarantees.)

Report how many skills were created, repaired, or already valid, plus the scaffold files and directories sync created.

#### 1d. `fab/.kit-migration-version`

Handled by `fab sync` (step 1c). The sync command creates `fab/.kit-migration-version` with version logic based on project state:

- **New project** (no `fab/project/config.yaml`): copies `$(fab kit-path)/VERSION` value (engine version)
- **Existing project** (has `fab/project/config.yaml`, no `fab/.kit-migration-version`): writes `0.1.0` (base version, run `/fab-setup migrations` to migrate)
- **Already exists**: preserves existing `fab/.kit-migration-version` — no overwrite

On bootstrap output:
- New project: `Created: fab/.kit-migration-version ({engine_version})`
- Existing project: `Created: fab/.kit-migration-version (0.1.0 — existing project, run "/fab-setup migrations" to migrate)`
- Re-run: `fab/.kit-migration-version` reported as part of scaffold output (no modification)

### Bootstrap Output

```
Found kit v{VERSION}. Initializing project...
{config.yaml prompts and creation}
{constitution.md generation}
Created: fab/project/config.yaml
Created: fab/project/constitution.md
{fab sync report — scaffold files (context.md, code-quality.md, code-review.md, docs/memory/index.md, docs/specs/index.md), fab/changes/ (+ archive), fab/.kit-migration-version ({version}), skills deployed to .claude/skills/, .gitignore merge (.fab-*)}
fab/ initialized successfully.

Next: {per state table — initialized}
```

On re-run, report config/constitution as OK/repaired instead of Created and surface sync's idempotent report, ending with `fab/ structure verified.`

---

## Config Behavior

Create a new `fab/project/config.yaml` interactively or update specific sections. Preserves YAML comments via targeted string replacement. Validates after each edit.

**Context loading**: Loads `fab/project/config.yaml` only (the file being edited). Does NOT load constitution, memory, or specs.

### Config Arguments

- **`[section]`** *(optional)* — section to edit directly, skipping the menu. Valid values: `project`, `source_paths`, `checklist`, `context`, `code-quality`, `code-review`.

### Config Pre-flight

- **Update mode**: `fab/project/config.yaml` must exist. If missing (direct invocation): STOP with `fab/project/config.yaml not found. Run /fab-setup to create it.`
- **Create mode** (from bootstrap): `fab/project/config.yaml` does not exist, is a raw template, or is missing the required fields `project.name`/`project.description` (e.g., a `fab init`-created, `fab_version`-only config).

### Config Create Mode

When `fab/project/config.yaml` does not exist (or exists without the required `project.name`/`project.description` fields):

1. Read the project's README, package.json, or other root-level files for context
2. Ask the user: project name, description, source paths
3. Read `$(fab kit-path)/scaffold/fab/project/config.yaml` as the starting template
4. Substitute placeholders with user-provided values: `{PROJECT_NAME}`, `{PROJECT_DESCRIPTION}`, `{SOURCE_PATHS}`
5. **Preserve `fab_version`**: if the existing config.yaml has a `fab_version` key (e.g., written by `fab init`), carry it into the new file unchanged — the scaffold template lacks it and the fab router errors without it. **Fallback (fresh create)**: if no `fab_version` key exists (no prior config.yaml at all), stamp the engine version into the new file — `fab_version: "$(cat "$(fab kit-path)/VERSION")"` — so the guarantee that `fab_version` exists after step 1a holds on every path
6. Write the result to `fab/project/config.yaml`
7. Output: `Created fab/project/config.yaml`

### Config Update Mode — Menu Flow

When invoked without a section argument:

1. Display the section menu:

```
fab/project/config.yaml sections:
1. project            — name and description
2. source_paths       — implementation code directories
3. checklist          — extra plan-acceptance categories (config key remains `checklist.extra_categories`)
4. context.md         — free-form project context
5. code-quality.md    — coding standards for apply/review
6. code-review.md     — review policy for validation sub-agent
7. Done

Which section to update? (1-7)
```

2. Process selection -> **Edit Section Flow**
3. After editing: "Update another section? (1-6 or 'done')"
4. Loop until Done

When invoked with a section argument: validate against valid sections (error if invalid), go directly to **Edit Section Flow**, then offer to update another section.

### Config Edit Section Flow

1. **Display current value** of the section
2. **Accept new value** — inline for simple values, block for multi-line
3. **Apply via string replacement** — targeted match, NOT full YAML rewrite (preserves comments)
4. **Validate** — YAML parseable, required fields present (`project.name`, `project.description`)
5. Pass -> confirm: `Updated {section}.` Fail -> report error, offer revert.

If no changes made, output: `No changes made. config.yaml unchanged.`

### Config Output

Show `Created fab/project/config.yaml` (create mode), `{N} sections updated in fab/project/config.yaml` (update mode), or `No changes made` (no-op). Next steps: `/fab-new` after create.

### Config Error Handling

| Condition | Action |
|-----------|--------|
| `fab/project/config.yaml` missing (update mode, direct invocation) | Abort with creation guidance |
| Invalid section argument | Output valid section names |
| YAML parse failure after edit | Report error, offer revert |
| Missing required field after edit | Report which field, offer revert |
| String replacement target not found | Warn about manual reformatting, fall back to section insert |

---

## Constitution Behavior

Create a new project constitution or amend an existing one with semantic versioning and structural preservation.

**Context loading**: Loads `fab/project/config.yaml` and `fab/project/constitution.md` (if it exists). Does NOT load memory or specs.

### Constitution Pre-flight

1. `fab/project/config.yaml` must exist. If missing (direct invocation): STOP with `fab/project/config.yaml not found. Run /fab-setup first.`
2. Read `fab/project/config.yaml` for project context
3. Check whether `fab/project/constitution.md` exists -> determines mode

### Constitution Create Mode

When `fab/project/constitution.md` does not exist:

1. Read project context from `fab/project/config.yaml` + README, existing docs, codebase structure
2. Read `$(fab kit-path)/scaffold/fab/project/constitution.md` as the starting skeleton
3. Generate principles based on the project's actual patterns, tech stack, and constraints — fill in the skeleton structure (replace `{Project Name}`, `{Principle Name}`, `{DATE}` placeholders; generate 3-7 principles with MUST/SHALL/SHOULD keywords)
4. Write the result to `fab/project/constitution.md`
5. Output: `Created fab/project/constitution.md (version 1.0.0) with {N} principles.`

### Constitution Update Mode

When `fab/project/constitution.md` already exists:

1. Read and display current content, read version from Governance
2. Present amendment menu:

```
Current constitution: version {X.Y.Z}, {N} principles

What would you like to change?
1. Add a new principle
2. Modify an existing principle
3. Remove a principle
4. Add or modify a constraint
5. Update governance metadata
6. Done — no changes
```

3. Process selection:
   - **Add**: Ask for name/description, insert at next Roman numeral. Bump: MINOR.
   - **Modify**: Show numbered list, accept new text. Ask: "(1) fundamental change or (2) wording clarification?" Bump: MAJOR or PATCH.
   - **Remove**: Show numbered list, re-number remaining. Bump: MAJOR.
   - **Add/modify constraint**: Show section, accept edits. Bump: MINOR (add) or PATCH (modify).
   - **Update governance**: Allow metadata edits. Bump: PATCH.
   - **Done**: Proceed to version bump.

4. After each action: "Any other changes? (yes/no)" — loop or proceed.

5. **Version bump**: Apply highest-severity bump across all amendments (MAJOR > MINOR > PATCH). Update Governance: increment version, set "Last Amended" to today.

6. **Structural preservation**: Verify heading hierarchy, sequential Roman numerals, Governance format. Re-number if needed.

7. Write updated file. If no changes: `No changes made. Constitution unchanged at version {X.Y.Z}.`

### Constitution Output

Show `Created fab/project/constitution.md (version 1.0.0) with {N} principles.` (create) or amendment summary with `Version: {old} -> {new}` (update). Next steps: `/fab-new`.

### Constitution Error Handling

| Condition | Action |
|-----------|--------|
| `fab/project/config.yaml` missing (direct invocation) | Abort with guidance |
| `fab/project/constitution.md` malformed (update mode) | Warn: "Structure appears non-standard. Proceeding with best-effort parsing." |
| Governance section missing version | Warn and start from 1.0.0 |
| Roman numeral parsing fails | Warn and proceed with sequential numbering from I |

---

## Migrations Behavior

Bring project files in sync with the installed kit version. Version reading, parsing, and comparison are owned by `fab migrations-status` — the skill runs it once and branches on its result, then applies each applicable migration file. Each migration is a markdown instruction file — the skill reads it and executes the steps as an LLM agent.

When `[file]` is provided, read and apply that specific migration file directly, bypassing version range discovery.

### Migrations Context Loading

1. Read `fab/project/config.yaml` (Always Load layer — MUST exist; if missing: STOP with `fab/project/config.yaml not found. Run /fab-setup to create it.`). Skip Change Context.
2. Migration discovery comes from the **single** `fab migrations-status --json` run in Step 1 (the binary owns version read/parse/compare and the scan/validate/sort) — no separate version read or second invocation here

### Migrations Step 1: Discover Migrations

Discovery is owned by the binary — do NOT read, parse, or compare the version files, and do NOT scan, validate, or sort the migrations directory by hand. The binary exits non-zero with remediation hints on a missing `fab/.kit-migration-version` or engine `VERSION` file — surface its stderr and stop.

1. Run `fab migrations-status --json` and parse the result. The shape is:
   `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}`.
   - `local` / `engine` — the project's `fab/.kit-migration-version` and the kit's `$(fab kit-path)/VERSION`, already read and parsed by the binary
   - `applicable` — the ordered list of migration files to apply, FROM ascending (already discovered, gap-skipped, and chained by the binary)
   - `gap_skips` — human-readable "no migration needed for X -> Y, skipping" lines to surface in output
   - `overlaps` — pairs of conflicting filenames; non-empty means the migrations directory is malformed
2. **If `overlaps` is non-empty**: STOP and report the conflict (see [Overlapping Ranges](#overlapping-ranges)). Do NOT apply anything.
3. **If `applicable` is empty** (and no overlap): nothing to do — pick the output by comparing the returned `local`/`engine` fields: equal → [Versions Already Equal](#versions-already-equal); `local` ahead of `engine` → [Local Version Ahead](#local-version-ahead); otherwise → [No Migrations Apply](#no-migrations-apply). (Semver comparison: compare MAJOR, then MINOR, then PATCH as integers — `2.10.0` > `2.9.7`; never compare lexicographically.) `fab upgrade-repo` already stamps `fab/.kit-migration-version` silently in the no-op case, so this subcommand has no version to write.

### Migrations Step 2: Apply Migrations (Loop)

Surface each `gap_skips` line, then apply each file in `applicable` IN ORDER:

1. For each `{from,to,file}` in `applicable`, apply it (see [Applying a Migration](#applying-a-migration)) — this reads the file at `$(fab kit-path)/migrations/{file}`, executes its Pre-check/Changes/Verification, and writes `to` to `fab/.kit-migration-version`.
2. Continue until every `applicable` entry is applied.

### Migrations Step 3: Finalize

- After applying the last `applicable` migration, `fab/.kit-migration-version` already holds that migration's `to` value (written per [Applying a Migration](#applying-a-migration)).
- Output completion summary

---

## Applying a Migration

For each migration file:

1. **Read** the migration file `$(fab kit-path)/migrations/{FROM}-to-{TO}.md`
2. **Execute Pre-check** section: verify each condition. If any fails -> STOP, report which pre-check failed, do NOT proceed
3. **Execute Changes** section: apply each change in order. Read referenced files, make modifications, write results
4. **Execute Verification** section: validate each condition. If any fails -> STOP, report which verification step failed
5. **Update version**: write `TO` to `fab/.kit-migration-version`

---

## Migrations Output Format

### Successful Multi-Step Migration

```
Local version:  {current}
Engine version: {target}
Migrations found: {N}

[1/{N}] Applying {FROM} -> {TO}...
{migration output}
-> fab/.kit-migration-version updated to {TO}

[2/{N}] Applying {FROM} -> {TO}...
{migration output}
-> fab/.kit-migration-version updated to {TO}

All migrations complete. fab/.kit-migration-version: {original} -> {final}
```

### Migration with Gap Skip

```
Local version:  {current}
Engine version: {target}
Migrations found: {N}

No migration needed for {current} -> {FROM}, skipping.

[1/{N}] Applying {FROM} -> {TO}...
{migration output}
-> fab/.kit-migration-version updated to {TO}

All migrations complete. fab/.kit-migration-version: {original} -> {final}
```

### Versions Already Equal

```
Already up to date ({version}).
```

### Local Version Ahead

```
Local version (fab/.kit-migration-version) is ahead of engine version ($(fab kit-path)/VERSION): {local} > {engine}.
This is unexpected — check your kit cache installation.
```

### No Migrations Apply

```
Local version:  {current}
Engine version: {target}
No migrations apply.
```

(`fab migrations-status` returned an empty `applicable` list. `fab upgrade-repo` silently stamps `fab/.kit-migration-version` to the engine version in this no-op case, so there is nothing for this subcommand to write.)

### Overlapping Ranges

```
Overlapping migration ranges detected: {file1} and {file2}. Fix the migrations directory.
```

### Mid-Chain Failure

```
[{N}/{total}] Applying {FROM} -> {TO}...
{partial output}
FAIL: Migration failed at {Pre-check|Changes|Verification} step: {description}
fab/.kit-migration-version remains at {current_version}.
Fix the issue and re-run /fab-setup migrations to continue from {current_version}.
```

---

## Idempotency

All paths are safe to re-run. Structural artifacts are created once (skipped on re-run). Symlinks are verified/repaired every run. Config/constitution edits are no-ops when unchanged. Migrations apply only remaining steps.

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No — project-level tool |
| Idempotent? | Yes |
| Modifies `fab/project/config.yaml`? | Yes (bootstrap creates, config subcommand updates, migrations may modify) |
| Modifies `fab/project/constitution.md`? | Yes (bootstrap creates, constitution subcommand updates, migrations may modify) |
| Modifies `fab/.kit-migration-version`? | Yes (migrations) |
| Modifies kit cache? | No — migrations only touch project-level files |
| Requires active change? | No |

---

## Next Steps Reference

All `Next:` lines are derived from the state table in `_preamble.md`:

- After bootstrap, config create, or constitution create: state = `initialized` → `Next: /fab-new <description>, /fab-proceed, or /docs-hydrate-memory <sources>`
- After config/constitution update: (no state change, no further action needed — validation is automatic)
- After migrations: re-derive from the current state — `initialized` (no active change) → `Next: /fab-new <description>, /fab-proceed, or /docs-hydrate-memory <sources>`; with an active change, use that change's stage row instead
