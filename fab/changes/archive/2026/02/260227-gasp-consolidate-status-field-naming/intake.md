# Intake: Consolidate Status Field Naming

**Change**: 260227-gasp-consolidate-status-field-naming
**Created**: 2026-02-27
**Status**: Draft

## Origin

> Conversational discussion in `/fab-discuss` session. User identified two problems: (1) `issue_id` is a scalar but should be an array to support multi-issue changes, (2) `shipped`/`ship_url`/`is_shipped` naming is inconsistent — `shipped` reads as a verb/adjective while it's actually a noun (array of PR URLs). User proposed consolidating both fields with symmetric naming. Additional finding: `fab-new` writes `issue_id` with raw `yq` instead of going through a proper script API.

Key decisions from discussion:
- `shipped: []` → `prs: []` (plural noun describing contents)
- `issue_id: null` → `issues: []` (plural noun, array not scalar)
- Four symmetric stageman operations: `add-issue`, `get-issues`, `add-pr`, `get-prs`
- No dedicated `has-*` boolean — `get-*` returning an empty list is sufficient
- `fab-new` must use stageman API, not raw `yq`

## Why

1. **Naming inconsistency**: `shipped` sounds like a boolean state but is an array of URLs. `ship_url` is a verb-noun combo. `is_shipped` is a boolean query on array length. The naming doesn't self-document what the field contains or what the operations do.

2. **Scalar limitation**: `issue_id` is a single string, but a change can close multiple Linear tickets. When creating a PR, Linear picks up all issue IDs mentioned in the title — space-separated works natively.

3. **Missing API boundary**: `fab-new` writes `issue_id` directly via `yq` instead of using stageman, violating the principle that stageman owns `.status.yaml` field mutations. This creates a maintenance hazard — any schema change to the field requires hunting down raw `yq` calls.

## What Changes

### Template: `fab/.kit/templates/status.yaml`

Replace:
```yaml
issue_id: null
shipped: []
```

With:
```yaml
issues: []
prs: []
```

### Schema: `fab/.kit/schemas/workflow.yaml`

Replace the `shipped:` section with two field declarations:

```yaml
issues:
  type: "string[]"
  description: "External tracker IDs associated with this change (e.g., DEV-123). Append-only."
  initial_value: "[]"
  managed_by: "stageman.sh add-issue / get-issues"

prs:
  type: "string[]"
  description: "PR URLs associated with this change. Append-only list of bare URL strings."
  initial_value: "[]"
  managed_by: "stageman.sh add-pr / get-prs"
```

### Stageman functions: `fab/.kit/scripts/lib/stageman.sh`

**Replace** `ship_url()` and `is_shipped()` with four symmetric functions:

| Function | CLI | Behavior |
|----------|-----|----------|
| `add_issue(status_file, id)` | `stageman.sh add-issue <change> <id>` | Append issue ID to `issues[]`, deduplicate, update `last_updated` |
| `get_issues(status_file)` | `stageman.sh get-issues <change>` | Emit issue IDs one per line (empty output = no issues) |
| `add_pr(status_file, url)` | `stageman.sh add-pr <change> <url>` | Append PR URL to `prs[]`, deduplicate, update `last_updated` |
| `get_prs(status_file)` | `stageman.sh get-prs <change>` | Emit PR URLs one per line (empty output = no PRs) |

The implementation pattern follows `ship_url()` — tmpfile copy, `yq -i` append with dedup check, atomic `mv`.

**Remove** the old `ship_url()`, `is_shipped()` functions and their CLI routes (`ship`, `is-shipped`).

### Skill: `fab/.kit/skills/fab-new.md`

**Replace** the raw `yq` call:
```
yq -i '.issue_id = "DEV-988"' fab/changes/{name}/.status.yaml
```

With:
```
fab/.kit/scripts/lib/stageman.sh add-issue fab/changes/{name}/.status.yaml DEV-988
```

### Skill: `fab/.kit/skills/git-pr.md`

**Replace** `issue_id` reads with `get-issues`:

- Step 1: Instead of reading `issue_id` from `.status.yaml` directly, run `stageman.sh get-issues <change>` and capture output. The skill should never instruct the agent to read `.status.yaml` fields directly — always go through stageman.
- Step 3c: Join issue IDs with space for PR title: `feat: DEV-123 DEV-456 Add OAuth support`
- If `get-issues` returns empty output, omit the issue prefix (same as current null behavior)

**Replace** `stageman.sh ship` calls with `stageman.sh add-pr`.

**Replace** commit message `"Record shipped URL in .status.yaml"` with `"Record PR URL in .status.yaml"`.

### Principle: Skills MUST NOT touch `.status.yaml` directly

All skills that read or write `.status.yaml` fields MUST use stageman CLI commands — never raw `yq` or direct YAML reads. This ensures:
- Field renames don't break skills (only stageman internals change)
- Consistent `last_updated` tracking
- Atomic writes via tmpfile pattern

Currently only `fab-new` violates this (raw `yq` for `issue_id`). This change fixes it.

### Stageman help text

Update the help/usage block to reflect the new commands:

```
add-issue <change> <id>            Append issue ID to issues array (idempotent)
get-issues <change>                List issue IDs (one per line)
add-pr <change> <url>              Append PR URL to prs array (idempotent)
get-prs <change>                   List PR URLs (one per line)
```

Remove `ship`, `is-shipped` from help.

### Pipeline runner: `fab/.kit/scripts/pipeline/run.sh`

- Rename `.shipped` sentinel file to `.pr-done` (or similar — the sentinel signals "all git ops complete", not "shipped")
- Update the sentinel check: `local shipped_sentinel=.../.shipped` → `local pr_sentinel=.../.pr-done`
- Update log message: `"Done: $resolved_id — shipped"` → `"Done: $resolved_id — pr complete"`
- Update comment: `"Sentinel replaces stageman is-shipped"` → reference the new naming

### Skill: `fab/.kit/skills/fab-archive.md`

- Update Step 1 to reference `.pr-done` instead of `.shipped` as the temp file to clean up

### Spec: `docs/specs/naming.md`

- Update PR title pattern: `{type}: {issue_id} {title}` → `{type}: {issues} {title}` (space-joined array)
- Update backlog entry pattern: `[{issue_id}]` → `[{issues}]` (or note that multiple IDs can appear)
- Update prose references from `issue_id` to `issues`

### Documentation: `docs/memory/fab-workflow/templates.md`

Update the `.status.yaml` documentation section:
- Replace `issue_id: null` description with `issues: []` — array of external tracker IDs, managed by `stageman.sh add-issue / get-issues`
- Replace `shipped: []` description with `prs: []` — array of PR URLs, managed by `stageman.sh add-pr / get-prs`
- Note that skills access these fields via stageman CLI, not direct reads

## Affected Memory

- `fab-workflow/templates`: (modify) Update `.status.yaml` field docs — `issue_id` → `issues[]`, `shipped` → `prs[]`, document stageman API
- `fab-workflow/schemas`: (modify) Update workflow.yaml field declarations
- `fab-workflow/change-lifecycle`: (modify) Update `issue_id` references to `issues`
- `fab-workflow/kit-architecture`: (modify) Update directory tree comment (`shipped: []` → `prs: []`), stageman subcommand docs (`ship`/`is-shipped` → `add-pr`/`get-prs`), internal fn docs (`ship_url`/`is_shipped` → `add_pr`/`get_prs`)
- `fab-workflow/pipeline-orchestrator`: (modify) Update `.shipped` sentinel references to `.pr-done`
- `fab-workflow/execution-skills`: (modify) Update `stageman.sh ship` references to `stageman.sh add-pr`
- `fab-workflow/configuration`: (modify) Update `issue_id` reference in status.yaml field list

## Impact

- **stageman.sh** — replace `ship_url()`/`is_shipped()` with `add_pr()`/`get_prs()`/`add_issue()`/`get_issues()`, update CLI routes
- **workflow.yaml** — replace `shipped:` schema block with `issues:` and `prs:`
- **status.yaml template** — `issue_id: null` → `issues: []`, `shipped: []` → `prs: []`
- **fab-new skill** — replace raw `yq` with `stageman.sh add-issue`
- **git-pr skill** — read issues via `stageman.sh get-issues`, use `add-pr` instead of `ship`, update commit message
- **pipeline/run.sh** — rename `.shipped` sentinel to `.pr-done`, update references
- **fab-archive skill** — update `.shipped` cleanup to `.pr-done`
- **docs/specs/naming.md** — update `issue_id` references to `issues`
- **Memory files** — 7 files need updated references (templates, schemas, kit-architecture, pipeline-orchestrator, execution-skills, configuration, change-lifecycle)
- **Archived changes** — not migrated (old field names in archived `.status.yaml` files; no active code reads them)

## Open Questions

- None — all decisions made in discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Field names: `issues: []` and `prs: []` | Discussed — user chose plural nouns over verb forms | S:95 R:90 A:95 D:95 |
| 2 | Certain | Four symmetric operations: add-issue, get-issues, add-pr, get-prs | Discussed — user confirmed symmetric pattern, no has-* needed | S:95 R:85 A:90 D:95 |
| 3 | Certain | fab-new uses stageman API, not raw yq | Discussed — user identified raw yq as a bug | S:90 R:80 A:90 D:95 |
| 4 | Certain | PR title joins issues with space: `feat: DEV-123 DEV-456 title` | Discussed — Linear picks up space-separated IDs | S:85 R:85 A:85 D:90 |
| 5 | Certain | No migration for archived changes | Archives not read by active code for these fields | S:80 R:95 A:90 D:90 |
| 6 | Confident | get-prs replaces is-shipped (no dedicated boolean) | Discussed — caller checks empty output; slightly less ergonomic for shell `if` but simpler API | S:85 R:90 A:80 D:75 |

6 assumptions (5 certain, 1 confident, 0 tentative, 0 unresolved).
