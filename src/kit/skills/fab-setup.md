---
name: fab-setup
description: "Set up a new project, manage config/constitution, or apply version migrations. Safe to re-run."
---

# /fab-setup [subcommand]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.
> **Exception**: `/fab-setup` has subcommand-specific context loading:
> - **bare / config / constitution**: Skip the "Always Load" context layer if files don't exist (first-run). Load them only if they already exist (re-run scenario).
> - **migrations**: Load `fab/project/config.yaml` (MUST exist). Skip Change Context loading — migrations operate on project-level files, not a specific change.

## Contents

- [Arguments](#arguments)
- [Pre-flight Check](#pre-flight-check)
- [Bootstrap Behavior](#bootstrap-behavior)
- [Config Behavior](#config-behavior)
- [Constitution Behavior](#constitution-behavior)
- [Migrations Behavior](#migrations-behavior)
- [Applying a Migration](#applying-a-migration)
- [Migrations Output Format](#migrations-output-format)
- [Idempotency](#idempotency)
- [Key Properties](#key-properties)
- [Next Steps Reference](#next-steps-reference)

---

## Arguments

- **No arguments** — full structural bootstrap
- **`config [section]`** — create/update `fab/project/config.yaml` interactively; optional `[section]` edits one section directly (valid: `project`, `source_paths`, `checklist`)
- **`constitution`** — create/amend `fab/project/constitution.md` with semantic versioning
- **`migrations [file]`** — apply version migrations to sync project files with the installed kit (absorbed from fab-update)
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
fab log command "fab-setup"
```

This is best-effort — the command always exits 0 (failures surface only as a stderr warning) and resolves the active change via `.fab-status.yaml` if one exists.

---

## Bootstrap Behavior

When invoked with no arguments, perform the full structural bootstrap. `/fab-setup` delegates directory/skeleton/deployment creation to `fab sync` (step 1c) while handling interactive config/constitution generation itself.

> **Ordering note**: `fab sync` runs immediately after the interactive config/constitution steps (1a/1b) — it requires a resolvable pinned version (`fab/.fab-version`, stamped by `fab init`). Its scaffolding operations are copy-if-absent / line-ensure merges, so the outcome is identical to the former sync-last order via idempotency.

### Phase 0: Prerequisite Check

Run `fab doctor` as the first step. If doctor exits non-zero, STOP immediately and surface the doctor output to the user. Do NOT create any project artifacts.

This gate applies only to the bare bootstrap flow. Subcommands (`config`, `constitution`, `migrations`) skip this check.

### Phase 1: Structural Bootstrap

Each step is **idempotent** — skip if the artifact already exists and is valid. On re-run, verify and repair rather than recreate.

#### 1a. `fab/project/config.yaml`

If the create-mode trigger holds (see [Config Create Mode](#config-create-mode)): execute **Config Behavior** (below) in create mode.
If exists with the required fields and not a placeholder generation: report "config.yaml already exists — skipping".

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

**Sync-failure guard**: if `fab sync` exits non-zero, STOP immediately and surface its output — do not continue the bootstrap. (Sync requires a resolvable pinned version in `fab/.fab-version`, which `fab init` stamps.)

Report how many skills were created, repaired, or already valid, plus the scaffold files and directories sync created.

#### 1d. `fab/.kit-migration-version`

Handled by `fab sync` (step 1c) — version logic by project state, with the matching bootstrap output line:

- **New project** (no `fab/project/config.yaml`): copies `$(fab kit-path)/VERSION` (engine version) → `Created: fab/.kit-migration-version ({engine_version})`
- **Existing project** (has `fab/project/config.yaml`, no `fab/.kit-migration-version`): writes `0.1.0` (base; run `/fab-setup migrations` to migrate) → `Created: fab/.kit-migration-version (0.1.0 — existing project, run "/fab-setup migrations" to migrate)`
- **Already exists**: preserves existing value, no overwrite → reported as part of scaffold output (no modification)

### Bootstrap Output

```
Found kit v{VERSION}. Initializing project...
{config.yaml + constitution.md interactive creation}
Created: fab/project/config.yaml
Created: fab/project/constitution.md
{fab sync report — scaffold files, fab/changes/ (+ archive), fab/.kit-migration-version ({version}), skills to .claude/skills/, .gitignore merge (.fab-*)}
fab/ initialized successfully.

Next: {per state table — initialized}
```

Re-run variant: report config/constitution as OK/repaired instead of `Created`, surface sync's idempotent report, and end with `fab/ structure verified.`

---

## Config Behavior

Create a new `fab/project/config.yaml` interactively or update specific sections. Preserves YAML comments via targeted string replacement. Validates after each edit.

**Context loading**: Loads `fab/project/config.yaml` only (the file being edited). Does NOT load constitution, memory, or specs.

### Config Arguments

- **`[section]`** *(optional)* — section to edit directly, skipping the menu. Valid values: `project`, `source_paths`, `checklist`, `context`, `code-quality`, `code-review`.

### Config Pre-flight

- **Update mode**: `fab/project/config.yaml` must exist. If missing (direct invocation): STOP with `fab/project/config.yaml not found. Run /fab-setup to create it.`
- **Create mode** (from bootstrap): the create-mode trigger holds (see [Config Create Mode](#config-create-mode)).

### Config Create Mode

**Create-mode trigger** (canonical): `fab/project/config.yaml` is missing, is a placeholder generation (contains the example identity value `My Project` — the embedded-stub fallback's default name), OR is missing the required fields `project.name`/`project.description` — e.g. the canonical `fab init` flow generates `config.yaml` from the registry (via `fab config init --project`, or a minimal embedded stub if the installed fab-go predates it) before sync's copy-if-absent runs.

> **What `fab init` already seeded.** `fab init` runs a mechanical, non-interactive detection at the Go layer and passes it to `fab config init --project`, so the generated file already carries **live** identity fields where detection was confident: `project.name` from the repo folder name, `source_paths` from an existing `src/` directory, and `test_paths` from the ecosystem marker table below. `project.description` is never detected mechanically (there is no reliable source) and is absent from the generated file. Your job in create mode is to **refine** these seeded values to what the user actually wants and to **add the description** — not to fill an empty template.

When that trigger holds:

1. Read the project's README, package.json, or other root-level files for context
2. Ask the user: project name, description, source paths (showing the seeded values `fab init` detected as defaults the user can accept or override). Then **confirm/refine `test_paths` non-interactively** (do NOT prompt for it — `fab init` already detected it from on-disk marker files; re-derive only to write the detection note in step 6):
   - **Detection sub-step**: the on-disk marker files map to an anchored `test_paths` pattern via the table below — the SAME table `fab init` used to seed the file. Multi-marker repos take the **union** of matched pattern sets. The anchoring (suffix/prefix/infix/source-root) is what makes the test/impl classification reliable — never substitute a bare substring like `**/*test*` (it miscounts production code such as `attestation.go` or `latest.go`).

     | Detected marker | Ecosystem | `test_paths` |
     |---|---|---|
     | `go.mod` | Go | `**/*_test.go` |
     | `pytest.ini` / `pyproject.toml` / `setup.cfg` | Python (pytest) | `**/test_*.py`, `**/*_test.py` |
     | `package.json` with jest/vitest dep, or `*.spec.ts`/`*.test.ts`/`*.spec.js`/`*.test.js` present | JS/TS | `**/*.spec.ts`, `**/*.test.ts`, `**/*.spec.js`, `**/*.test.js` |
     | `pom.xml` / `build.gradle` | Java/Kotlin (Maven/Gradle) | `**/src/test/**` |
     | `*.csproj` referencing a test SDK | .NET | `**/*Tests.cs`, `**/*Test.cs` |
     | `Cargo.toml` | Rust | *(none — Rust tests are inline `#[cfg(test)]`; not glob-addressable)* → leave empty, note why |
     | *(no marker / unrecognized)* | — | leave empty; standing examples remain the reference |

     Record the detected ecosystem + pattern set (or "no convention detected") for the note in step 6. Note the Go-layer detection is intentionally conservative (folder name; `src/`; single-file markers only) — JS/TS package.json-dep detection and any non-obvious call are your job here, so the seeded file may lack a `test_paths` your inspection can now add.
3. **Refine the registry-generated `config.yaml`** (the scaffold template was retired in 2.15.0 — `config.yaml` is generated from the registry, not substituted from a template). `fab init` already generated the file with the detected identity fields live above the managed reference fence (or a minimal embedded stub carrying the same detected seed if the installed fab-go predated `fab config init --project`). Apply the user's refinements in place via **targeted string replacement** (the same comment-preserving edit update-mode uses — NOT a full rewrite; the managed fence and every comment stay intact):
   - `project.name` → the user's name (the seeded folder-name default is often right — replace only if the user chose differently); **add `project.description`** (the generated file has no description key — insert one under `project:`)
   - `source_paths` → the user's source paths (replacing the seeded `src/` if different)
   - `test_paths` → the detected patterns. **For `test_paths`**: if the generated file has a live `test_paths:` key (detection seeded one), replace its value only if your richer inspection found a better set; if `test_paths` sits only inside the commented fence (no marker was detected at init), add a live `test_paths:` above the fence when your inspection now finds a convention, else leave the fence untouched — the field stays inherited/advertised.
   ```yaml
   test_paths:
     - "**/*_test.go"
   ```
   When no ecosystem was recognized (or the stack uses inline tests like Rust), leave `test_paths` unset (it stays advertised in the fence); the impact breakdown collapses to a single total (today's behavior). Do **not** hand-add or remove the managed reference fence — `fab config upgrade` (auto-run by `fab upgrade-repo`) owns it.
4. **Do NOT touch the pinned version**: the engine version lives in `fab/.fab-version` (stamped by `fab init`), NOT in `config.yaml` — there is no `fab_version:` key to preserve or stamp (relocated in 2.15.0). Leave `fab/.fab-version` alone.
5. Validate the edited `config.yaml` (YAML parses; `project.name`/`project.description` present).
6. Output: `Updated fab/project/config.yaml`, then a **test_paths detection note**:
   - **Detected**: `Detected {ecosystem} — set test_paths to {patterns}. Edit fab/project/config.yaml if wrong.`
   - **Not detected**: `No test convention detected — test_paths left empty (impact breakdown will show a single total). Set it later if desired.`

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
3. After editing: "Update another section? (1-7 or 'done')"
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

Show `Updated fab/project/config.yaml` (create mode — the registry-generated file is populated in place), `{N} sections updated in fab/project/config.yaml` (update mode), or `No changes made` (no-op). In create mode, follow the update line with the **test_paths detection note** (per Config Create Mode step 6 — detected ecosystem + patterns, or "no test convention detected → left empty"). Next steps: `/fab-new` after create.

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

Discovery is binary-owned (per the Migrations Behavior intro and Context Loading) — do nothing by hand. The binary exits non-zero with remediation hints on a missing `fab/.kit-migration-version` or engine `VERSION` file — surface its stderr and stop.

1. Run `fab migrations-status --json` and parse the result. Shape: `{local, engine, applicable:[{from,to,file}], gap_skips, overlaps}` — `local`/`engine` are the parsed project + kit versions; `applicable` is the ordered (FROM-ascending, gap-skipped, chained) list of files to apply; `gap_skips` are human-readable "no migration needed for X -> Y, skipping" lines to surface; `overlaps` are conflicting-filename pairs (non-empty = malformed migrations directory).
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

Canonical happy path (successful multi-step migration). The header scaffolding (`Local version:` / `Engine version:` / `Migrations found:`) and per-step block below are reused by the variants noted after it:

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

**Variants** (same header scaffolding unless noted; every literal below is exact):

- **Gap skip**: before the first `[1/{N}]` block (and after the header), insert: `No migration needed for {current} -> {FROM}, skipping.`
- **Versions already equal**: `Already up to date ({version}).`
- **Local version ahead**:
  `Local version (fab/.kit-migration-version) is ahead of engine version ($(fab kit-path)/VERSION): {local} > {engine}.`
  `This is unexpected — check your kit cache installation.`
- **No migrations apply**: header scaffolding (just `Local version:` / `Engine version:`, no `Migrations found:`) followed by `No migrations apply.` (`fab migrations-status` returned an empty `applicable` list. `fab upgrade-repo` silently stamps `fab/.kit-migration-version` to the engine version in this no-op case, so there is nothing for this subcommand to write.)
- **Overlapping ranges**: `Overlapping migration ranges detected: {file1} and {file2}. Fix the migrations directory.`
- **Mid-chain failure** (replaces the per-step block from the failing step onward):
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
