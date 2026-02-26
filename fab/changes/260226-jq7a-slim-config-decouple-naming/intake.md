# Intake: Slim Config & Decouple Naming

**Change**: 260226-jq7a-slim-config-decouple-naming
**Created**: 2026-02-26
**Status**: Draft

## Origin

> Simplify config.yaml: remove git and naming sections, rename rules to stage_directives with all-stage placeholders, add issue_id to status.yaml template, reduce config verbosity, update scaffold template, create migration, create docs/specs/naming.md spec, update fab-new to write issue_id to status.yaml instead of embedding in slug, update git-pr to read issue_id from status.yaml.

Conversational mode — extensive discussion preceded this intake. Key design decisions were made iteratively through field-by-field analysis of config.yaml usage patterns, naming convention audit, and Linear integration research.

## Why

Config.yaml has grown to 108 lines with heavy inline comments and fields that don't earn their place. The `git` section provides configurability that doesn't match real workflows (you often don't know feat/fix type when creating a branch). The `naming` section documents conventions that are enforced by code (`changeman.sh`), not by config — making it documentation-as-config that nothing actually parses. The `rules` key is too generic for what it does (per-stage artifact generation directives).

Separately, the Linear issue ID is currently embedded in the change folder slug (e.g., `DEV-988-add-oauth`), coupling the tracker ID to the immutable folder name. This means the issue ID must be known at `/fab-new` time — but in practice it may be discovered later (during discussion, or at PR time). The ID should live in status.yaml where it can be hydrated at any point and consumed by the one place that actually needs it: the PR title.

Finally, naming conventions are scattered across skills, scripts, config comments, and memory files with no single canonical reference.

## What Changes

### Remove `git` section from config.yaml

The `git.enabled` and `git.branch_prefix` fields are removed entirely.

**Current behavior**:
- `git.enabled: false` suppresses branch prompts in `/git-branch`, `/fab-switch`, `/fab-status`
- `git.branch_prefix` prepends a string (e.g., `feat/`) to branch names

**New behavior**:
- Git integration is always enabled (if in a git repo)
- Branch name = change folder name directly, no prefix
- `/git-branch` removes its Step 1 `git.enabled` gate
- Scripts (`dispatch.sh`, `batch-fab-switch-change.sh`) already default gracefully when the field is absent — `get_branch_prefix()` returns `""`

### Remove `naming` section from config.yaml

Both `naming.format` and `naming.backlog_format` are removed.

- `naming.format` is descriptive documentation — `changeman.sh` hardcodes the `{YYMMDD}-{XXXX}-{slug}` pattern and never reads this field
- `naming.backlog_format` has zero references in any skill or script — dead config

The change folder naming convention moves to `docs/specs/naming.md` as canonical documentation for humans. Skills and scripts remain self-contained (they carry their own naming logic, not referencing the spec).

### Rename `rules` to `stage_directives`

The `rules` key becomes `stage_directives` with explicit placeholders for all six stages:

```yaml
stage_directives:
  intake: []
  spec:
    - Use GIVEN/WHEN/THEN for scenarios
    - Mark ambiguities with [NEEDS CLARIFICATION]
  tasks: []
  apply: []
  review: []
  hydrate: []
```

Empty arrays serve as visible placeholders showing what's available.

### Add `issue_id` to status.yaml template

A new `issue_id` field is added to `fab/.kit/templates/status.yaml`:

```yaml
issue_id: null
```

This decouples the tracker ID from the change folder name. Any skill that learns about the issue can write it:
- `/fab-new` (when created from a Linear ticket)
- User provides it during discussion
- `/git-pr` (user provides at PR time)

### Update `/fab-new` — write `issue_id` instead of embedding in slug

Currently Step 1 prefixes the Linear ID into the slug (e.g., `DEV-988-add-oauth`). Instead:
- Slug is generated without the Linear ID: `add-oauth`
- If a Linear ticket was detected in Step 0, write `issue_id: DEV-988` to `.status.yaml` after `changeman.sh new`

### Update `/git-pr` — read `issue_id` from status.yaml

`/git-pr` reads `issue_id` from `.status.yaml` and includes it in the PR title or description for Linear auto-linking. Linear recognizes the issue ID in branch name, PR title, or PR description — so including it in the PR title/description is sufficient.

### Reduce config.yaml verbosity

Strip multi-line `# Consumed by:` comment blocks. Keep one-liner descriptions per field. Remove the file header comment block. Target ~35 lines down from ~108.

### Update scaffold template

`fab/.kit/scaffold/fab/project/config.yaml` receives the same structural changes: remove `git`, remove `naming`, rename `rules` → `stage_directives`, reduce comments.

### Create `docs/specs/naming.md`

New spec documenting all five naming conventions with pattern, example, and where encoded:

1. **Change folder name** — `YYMMDD-XXXX-slug`, encoded in `changeman.sh`
2. **Git branch** — equals change folder name, encoded in `/git-branch` skill
3. **Worktree** — `adjective-noun`, encoded in `wt-create`
4. **PR title** — `{type}: {title}`, encoded in `/git-pr` skill
5. **Backlog entry** — `- [ ] [{ID}] {YYYY-MM-DD}: {description}`, encoded in idea command

This spec is project-level documentation (not shipped with .kit). Skills and scripts remain self-contained — naming info is duplicated at the point of use, with the spec as canonical human reference.

### Create migration `0.10.0-to-0.20.0.md`

Migration covering:
- Remove `git:` block from config.yaml
- Remove `naming:` block from config.yaml
- Rename `rules:` → `stage_directives:`
- Strip verbose comments by referring to the scaffold at fab/.kit/scaffold/fab/project/config.yaml
- Bump `fab/project/VERSION` to `0.20.0`

### Update downstream skills and scripts

Skills that reference `git.enabled`, `git.branch_prefix`, or `rules`:
- `fab/.kit/skills/git-branch.md` — remove git.enabled gate, remove prefix logic
- `fab/.kit/skills/git-pr.md` — read `issue_id` from status.yaml, remove `git.branch_prefix` read
- `fab/.kit/skills/fab-switch.md` — always show `/git-branch` tip (remove git.enabled conditional)
- `fab/.kit/skills/fab-status.md` — always show current branch (remove git.enabled conditional)
- `fab/.kit/skills/_generation.md` — if any reference to `rules`, update to `stage_directives`
- All skills referencing `config.rules` → update to `config.stage_directives`

### Update memory and specs references

- `docs/memory/fab-workflow/configuration.md` — remove `git` and `naming` schema docs, rename `rules` → `stage_directives`
- `docs/specs/architecture.md` — remove git config references
- `docs/specs/skills.md` — update `/git-branch` spec, `/git-pr` spec
- `docs/specs/index.md` — add `naming.md` row

## Affected Memory

- `fab-workflow/configuration`: (modify) Remove git and naming schema docs, rename rules → stage_directives, document issue_id in status.yaml, document change_name_format removal
- `fab-workflow/change-lifecycle`: (modify) Simplify git integration section — always enabled, no prefix config
- `fab-workflow/templates`: (modify) Document issue_id addition to status.yaml template
- `fab-workflow/schemas`: (modify) Update status.yaml schema with issue_id field

## Impact

- **Skills**: git-branch, git-pr, fab-switch, fab-status, fab-new — behavioral changes
- **Scripts**: dispatch.sh, batch-fab-switch-change.sh — already handle missing git fields gracefully, minimal changes
- **Templates**: status.yaml template, scaffold config.yaml
- **Migration**: Projects upgrading from 0.10.0+ need to run the migration
- **Specs**: architecture.md, skills.md, index.md — reference updates
- **New spec**: naming.md

## Open Questions

None — all design decisions were resolved during the preceding discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove `git` section entirely | Discussed — user explicitly requested removal. Scripts already default gracefully. | S:95 R:85 A:90 D:95 |
| 2 | Certain | Remove `naming` section entirely | Discussed — user confirmed removal. Format never parsed by code, backlog_format has zero references. | S:95 R:90 A:90 D:95 |
| 3 | Certain | Rename `rules` → `stage_directives` | Discussed — user agreed on the name after evaluating alternatives. | S:95 R:90 A:95 D:95 |
| 4 | Certain | Add empty `[]` placeholders for all six stages in `stage_directives` | Discussed — user requested templates for all stages or "it won't get used." | S:90 R:95 A:85 D:90 |
| 5 | Certain | Add `issue_id` to status.yaml, not folder name | Discussed — user proposed this design. Decouples tracker ID from immutable folder name. | S:95 R:85 A:90 D:95 |
| 6 | Certain | Key name is `issue_id` (not `tracker_id` or `external_id`) | Discussed — user implicitly agreed when proposing `issue_id` in status.yaml. | S:85 R:95 A:85 D:80 |
| 7 | Certain | naming.md covers 5 items only (no task IDs, no external identifiers) | Discussed — user explicitly scoped: change folder, branch, worktree, PR title, backlog entry. | S:95 R:90 A:90 D:95 |
| 8 | Certain | naming.md is project-level spec, not shipped with .kit | Discussed — user clarified portability constraint. Skills remain self-contained. | S:95 R:90 A:95 D:95 |
| 9 | Certain | `/git-pr` is sole consumer of `issue_id` | Discussed — user confirmed only PR title needs the issue ID for Linear auto-linking. | S:90 R:85 A:85 D:85 |
| 10 | Certain | Git integration always enabled after removal | Discussed — user chose "Always enabled, no prefix" option. | S:90 R:80 A:85 D:90 |
| 11 | Confident | Migration range is `0.10.0-to-0.20.0` | Last migration TO was 0.10.0; projects from 0.10.0 through 0.19.x need this. | S:75 R:85 A:80 D:75 |
| 12 | Confident | `/git-pr` includes `issue_id` in PR title (not just description) | Linear auto-links from title OR description; title is more visible. User didn't specify placement. | S:70 R:90 A:75 D:70 |

12 assumptions (10 certain, 2 confident, 0 tentative, 0 unresolved).
