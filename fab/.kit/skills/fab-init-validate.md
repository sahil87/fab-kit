---
name: fab-init-validate
description: "Validate config.yaml and constitution.md structural correctness."
model_tier: fast
---

# /fab-init-validate

> Read and follow the instructions in `fab/.kit/skills/_context.md` before proceeding.
> **Context loading**: This skill reads `fab/config.yaml` and `fab/constitution.md` for validation. It does NOT load `fab/docs/index.md` or `fab/design/index.md`.

---

## Purpose

Validate the structural correctness of `fab/config.yaml` and `fab/constitution.md`. Reports issues with actionable fix suggestions. Useful after manual edits, before commits, or as a health check.

---

## Arguments

None.

---

## Pre-flight Check

No preflight script needed — this skill validates the files that preflight would normally check, so it must handle missing files gracefully.

---

## Behavior

### Step 1: Discover Files

Check for the existence of both files:
- `fab/config.yaml`
- `fab/constitution.md`

Track which files exist. Missing files are reported but do not block validation of other files.

### Step 2: Validate `config.yaml`

If `fab/config.yaml` exists, run all 8 structural checks in order:

| # | Check | Pass criteria | Failure message | Fix suggestion |
|---|-------|---------------|-----------------|----------------|
| 1 | YAML parseable | File parses as valid YAML | "FAIL: config.yaml is not valid YAML: {parse error}" | "Fix the YAML syntax error at the indicated location" |
| 2 | Required top-level keys | `project`, `context`, `stages`, `source_paths` all present | "FAIL: Missing required key '{key}'" | "Add `{key}:` as a top-level section" |
| 3 | `project.name` non-empty | String, length > 0 | "FAIL: project.name is missing or empty" | "Add `name: \"your-project\"` under the `project:` section" |
| 4 | `project.description` non-empty | String, length > 0 | "FAIL: project.description is missing or empty" | "Add `description: \"...\"` under the `project:` section" |
| 5 | `stages` non-empty list | Array with at least 1 entry | "FAIL: stages list is empty" | "Add at least the default stages (brief, spec, tasks, apply, review, archive)" |
| 6 | Stage `id` fields present | Every stage entry has an `id` string | "FAIL: Stage at index {N} is missing `id` field" | "Add `id: {suggested_id}` to the stage entry" |
| 7 | Stage `requires` valid | Every `requires` entry references an existing stage ID | "FAIL: Stage '{id}' requires non-existent stage '{ref}'" | "Check the `requires` list — valid stage IDs are: {list}" |
| 8 | No circular dependencies | No cycles in the stage dependency graph | "FAIL: Circular dependency detected: {cycle path}" | "Remove one of the `requires` entries to break the cycle" |

**Additional check (derived from check 6):**
- Stage IDs are unique: "FAIL: Duplicate stage ID '{id}'" / "Rename one of the duplicate stages"

If check 1 fails (YAML not parseable), **skip checks 2-8** — they depend on parsed content.

If `fab/config.yaml` does not exist:
```
config.yaml: not found — run /fab-init or /fab-init-config to create it
```

### Step 3: Validate `constitution.md`

If `fab/constitution.md` exists, run all 6 structural checks:

| # | Check | Pass criteria | Failure message | Fix suggestion |
|---|-------|---------------|-----------------|----------------|
| 1 | Non-empty | File has content (not just whitespace) | "FAIL: constitution.md is empty" | "Run /fab-init-constitution to generate content" |
| 2 | Level-1 heading | Contains `# ... Constitution` (case-insensitive match on "Constitution") | "FAIL: Missing level-1 heading with 'Constitution'" | "Add `# {Project Name} Constitution` as the first heading" |
| 3 | Core Principles section | Contains `## Core Principles` heading | "FAIL: Missing `## Core Principles` section" | "Add a `## Core Principles` section with at least one principle" |
| 4 | Roman numeral headings | At least one `### I.` or `### II.` etc. under Core Principles | "FAIL: No Roman numeral principle headings found (expected `### I.`, `### II.`, etc.)" | "Number each principle with Roman numerals: `### I. {Name}`, `### II. {Name}`, etc." |
| 5 | Governance section | Contains `## Governance` heading | "FAIL: Missing `## Governance` section" | "Add a Governance section with version, ratified date, and last amended date" |
| 6 | Version format | Governance section contains a version matching `MAJOR.MINOR.PATCH` pattern (e.g., `1.0.0`, `2.3.1`) | "FAIL: No version in MAJOR.MINOR.PATCH format found in Governance section" | "Add `**Version**: 1.0.0` to the Governance section" |

If `fab/constitution.md` does not exist:
```
constitution.md: not found — run /fab-init or /fab-init-constitution to create it
```

### Step 4: Combined Report

Present results for both files:

```
config.yaml:      {passed}/{total} checks passed {✓ or ✗}
constitution.md:  {passed}/{total} checks passed {✓ or ✗}
```

If all checks pass:
```
All validation checks passed.
```

If any checks fail:
```
{N} issue(s) found. Fix the issues above and re-run /fab-init-validate.
```

---

## Output

### All Checks Pass

```
config.yaml checks:
  ✓ YAML parseable
  ✓ Required top-level keys present
  ✓ project.name non-empty
  ✓ project.description non-empty
  ✓ stages non-empty list
  ✓ Stage id fields present
  ✓ Stage requires references valid
  ✓ No circular dependencies

constitution.md checks:
  ✓ Non-empty
  ✓ Level-1 heading with "Constitution"
  ✓ Core Principles section
  ✓ Roman numeral headings
  ✓ Governance section
  ✓ Version in MAJOR.MINOR.PATCH format

config.yaml:      8/8 checks passed ✓
constitution.md:  6/6 checks passed ✓

All validation checks passed.
```

### With Failures

```
config.yaml checks:
  ✓ YAML parseable
  ✓ Required top-level keys present
  ✗ project.name is missing or empty
    → Add `name: "your-project"` under the `project:` section
  ✓ project.description non-empty
  ✓ stages non-empty list
  ✓ Stage id fields present
  ✓ Stage requires references valid
  ✓ No circular dependencies

constitution.md checks:
  ✓ Non-empty
  ✓ Level-1 heading with "Constitution"
  ✓ Core Principles section
  ✓ Roman numeral headings
  ✗ Missing ## Governance section
    → Add a Governance section with version, ratified date, and last amended date
  — Version format (skipped — no Governance section)

config.yaml:      7/8 checks passed ✗
constitution.md:  4/6 checks passed ✗

2 issue(s) found. Fix the issues above and re-run /fab-init-validate.
```

### One File Missing

```
config.yaml checks:
  ✓ YAML parseable
  ...

config.yaml:      8/8 checks passed ✓
constitution.md:  not found — run /fab-init or /fab-init-constitution to create it

1 issue(s) found. Fix the issues above and re-run /fab-init-validate.
```

### Both Files Missing

```
config.yaml:      not found — run /fab-init or /fab-init-config to create it
constitution.md:  not found — run /fab-init or /fab-init-constitution to create it

No files to validate. Run /fab-init to bootstrap the project.
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/config.yaml` missing | Report as missing, suggest creation, continue to validate constitution |
| `fab/constitution.md` missing | Report as missing, suggest creation, continue to validate config |
| Both files missing | Report both, suggest `/fab-init` |
| YAML parse failure | Report parse error, skip remaining config checks |
| Governance section exists but version not found | Fail check 6, suggest adding version |
| Constitution check depends on failed earlier check | Mark as "skipped" with reason |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | **No** — project-level validation tool |
| Idempotent? | **Yes** — read-only, no modifications |
| Modifies `fab/config.yaml`? | **No** — read-only |
| Modifies `fab/constitution.md`? | **No** — read-only |
| Modifies `.status.yaml`? | **No** |
| Modifies source code? | **No** |
| Requires config? | **No** — this skill *validates* the config; handles missing files gracefully |

---

## Next Steps Reference

After all checks pass: `Next: /fab-new <description> or /fab-init-config (update) or /fab-init-constitution (amend)`

After failures: `Next: Fix the reported issues and re-run /fab-init-validate`
