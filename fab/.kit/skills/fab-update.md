---
name: fab-update
description: "Apply version migrations to bring project files in sync with the installed kit version."
---

# /fab-update

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Exception**: `/fab-update` skips Change Context loading (§2) — it operates on project-level files, not a specific change.

---

## Purpose

Compare `fab/VERSION` (local project version) to `fab/.kit/VERSION` (engine version), discover applicable migration files in `fab/.kit/migrations/`, and apply them sequentially. Each migration is a markdown instruction file — the skill reads it and executes the steps as an LLM agent.

---

## Arguments

None. `/fab-update` always operates on the current project.

---

## Context Loading

1. Read `fab/VERSION` and `fab/.kit/VERSION`
2. Read `fab/config.yaml` and `fab/constitution.md` (Always Load layer)
3. Scan `fab/.kit/migrations/` for migration files

---

## Pre-flight Checks

Before attempting any migration, verify:

1. **`fab/VERSION` exists** — if not: STOP with `fab/VERSION not found. Run /fab-init to create it.`
2. **`fab/.kit/VERSION` exists** — if not: STOP with `fab/.kit/VERSION not found — kit may be corrupted.`
3. Read both version strings and parse as `MAJOR.MINOR.PATCH` integers

---

## Behavior

### Step 1: Compare Versions

- Read `fab/VERSION` → `current`
- Read `fab/.kit/VERSION` → `target`
- If `current` >= `target`: report and stop (see scenarios below)

### Step 2: Discover Migrations

1. Scan `fab/.kit/migrations/` for files matching `{FROM}-to-{TO}.md`
2. Parse FROM and TO as semver from each filename
3. **Validate non-overlapping ranges**: for every pair of migration files, check that their ranges do not overlap (`A.FROM < B.TO AND B.FROM < A.TO` means overlap). If overlap detected: STOP with error listing the conflicting files
4. Sort migrations by FROM ascending

### Step 3: Apply Migrations (Loop)

Execute the migration discovery algorithm:

1. Find the first migration where `FROM <= current < TO`
2. **If found**: apply it (see [Applying a Migration](#applying-a-migration)), set `current = TO`, repeat from (1)
3. **If not found but a migration exists with `FROM > current`**: skip to that FROM — log: `No migration needed for {current} → {FROM}, skipping.` — repeat from (1)
4. **If not found and no later migrations exist**: set `fab/VERSION` to engine version, done

### Step 4: Finalize

- Write `fab/VERSION` with the engine version (should already match after migrations)
- Output completion summary

---

## Applying a Migration

For each migration file:

1. **Read** the migration file `fab/.kit/migrations/{FROM}-to-{TO}.md`
2. **Execute Pre-check** section: verify each condition. If any fails → STOP, report which pre-check failed, do NOT proceed
3. **Execute Changes** section: apply each change in order. Read referenced files, make modifications, write results
4. **Execute Verification** section: validate each condition. If any fails → STOP, report which verification step failed
5. **Update version**: write `TO` to `fab/VERSION`

---

## Output Format

### Successful Multi-Step Migration

```
Local version:  {current}
Engine version: {target}
Migrations found: {N}

[1/{N}] Applying {FROM} → {TO}...
{migration output}
✓ fab/VERSION updated to {TO}

[2/{N}] Applying {FROM} → {TO}...
{migration output}
✓ fab/VERSION updated to {TO}

All migrations complete. fab/VERSION: {original} → {final}
```

### Migration with Gap Skip

```
Local version:  {current}
Engine version: {target}
Migrations found: {N}

No migration needed for {current} → {FROM}, skipping.

[1/{N}] Applying {FROM} → {TO}...
{migration output}
✓ fab/VERSION updated to {TO}

All migrations complete. fab/VERSION: {original} → {final}
```

### Versions Already Equal

```
Already up to date ({version}).
```

### Local Version Ahead

```
Local version (fab/VERSION) is ahead of engine version (fab/.kit/VERSION): {local} > {engine}.
This is unexpected — check your fab/.kit/ installation.
```

### No Migrations Exist

```
Local version:  {current}
Engine version: {target}
No migrations found. fab/VERSION updated to {target}.
```

### Overlapping Ranges

```
Overlapping migration ranges detected: {file1} and {file2}. Fix the migrations directory.
```

### Migration Failure

```
[{N}/{total}] Applying {FROM} → {TO}...
{partial output}
✗ Migration failed at {Pre-check|Changes|Verification} step: {description}
fab/VERSION remains at {current_version}.
Fix the issue and re-run /fab-update to continue from {current_version}.
```

---

## Semver Comparison

To compare two semver strings, compare MAJOR, then MINOR, then PATCH as integers. `A >= B` means A.MAJOR > B.MAJOR, or (A.MAJOR == B.MAJOR and A.MINOR > B.MINOR), or (A.MAJOR == B.MAJOR and A.MINOR == B.MINOR and A.PATCH >= B.PATCH).

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — project-level tool |
| Idempotent? | **Yes** — re-running applies only remaining migrations |
| Modifies `fab/VERSION`? | **Yes** — updated after each successful migration |
| Modifies project files? | **Yes** — migrations may modify `config.yaml`, `constitution.md`, etc. |
| Modifies `fab/.kit/`? | **No** — migrations only touch project-level files |
| Requires active change? | **No** |
