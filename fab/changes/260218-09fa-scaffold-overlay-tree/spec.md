# Spec: Scaffold Overlay Tree

**Change**: 260218-09fa-scaffold-overlay-tree
**Created**: 2026-02-18
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md` (modify)

## Non-Goals

- Changing how skill/agent sections (5, 6) work — they don't come from scaffold
- Changing directory creation logic (section 1) — not scaffold-derived
- Changing `fab/VERSION` logic (section 1b) — not scaffold-derived
- Adding new scaffold files — this change only restructures existing ones
- Changing fab-setup interactive prompting logic — only detection and path references

## Scaffold: Overlay Tree Structure

### Requirement: Scaffold Directory Mirrors Repo Root

The `fab/.kit/scaffold/` directory SHALL be restructured as an overlay tree where each file's path relative to `scaffold/` mirrors its destination path relative to the repo root. Files that require merge strategies (not plain copy) SHALL be prefixed with `fragment-`.

The complete mapping SHALL be:

| Current flat path | New overlay path |
|---|---|
| `scaffold/envrc` | `scaffold/fragment-.envrc` |
| `scaffold/gitignore-entries` | `scaffold/fragment-.gitignore` |
| `scaffold/settings.local.json` | `scaffold/.claude/fragment-settings.local.json` |
| `scaffold/memory-index.md` | `scaffold/docs/memory/index.md` |
| `scaffold/specs-index.md` | `scaffold/docs/specs/index.md` |
| `scaffold/sync-readme.md` | `scaffold/fab/sync/README.md` |
| `scaffold/config.yaml` | `scaffold/fab/config.yaml` |
| `scaffold/constitution.md` | `scaffold/fab/constitution.md` |
| `scaffold/context.md` | `scaffold/fab/context.md` |
| `scaffold/code-quality.md` | `scaffold/fab/code-quality.md` |
| `scaffold/code-review.md` | `scaffold/fab/code-review.md` |

#### Scenario: Verify overlay tree structure

- **GIVEN** the scaffold directory has been restructured
- **WHEN** listing all files recursively under `scaffold/`
- **THEN** exactly 11 files exist, each at its new overlay path
- **AND** the old flat paths (e.g., `scaffold/envrc`, `scaffold/memory-index.md`) no longer exist

#### Scenario: Fragment prefix identifies merge files

- **GIVEN** the scaffold directory contains `fragment-.envrc`, `fragment-.gitignore`, and `.claude/fragment-settings.local.json`
- **WHEN** inspecting the scaffold tree
- **THEN** exactly 3 files have the `fragment-` prefix
- **AND** all other files (8 remaining) have no prefix

## Scaffold: Fragment Convention

### Requirement: Fragment Prefix Semantics

Files prefixed with `fragment-` SHALL be merge sources, not complete files. The sync script SHALL strip the `fragment-` prefix from the filename to determine the target filename and dispatch to the appropriate merge strategy based on file extension.

Merge strategy dispatch:
- **`fragment-` + `.json` extension**: JSON array merge (merge `permissions.allow` arrays using jq)
- **`fragment-` + any other extension**: Line-ensuring merge (append non-duplicate, non-comment lines)

Files without the `fragment-` prefix SHALL use copy-if-absent strategy (copy only if the target does not exist).

#### Scenario: Line-ensuring merge for .envrc

- **GIVEN** `scaffold/fragment-.envrc` contains required `.envrc` entries
- **AND** the repo root has an existing `.envrc` file
- **WHEN** the sync tree-walk processes `scaffold/fragment-.envrc`
- **THEN** the `fragment-` prefix is stripped → target is `.envrc`
- **AND** each non-comment, non-empty line from the fragment is appended to `.envrc` if not already present
- **AND** existing lines in `.envrc` are preserved

#### Scenario: Line-ensuring merge creates target if absent

- **GIVEN** `scaffold/fragment-.gitignore` contains required entries
- **AND** the repo root has no `.gitignore` file
- **WHEN** the sync tree-walk processes `scaffold/fragment-.gitignore`
- **THEN** `.gitignore` is created with the fragment's entries

#### Scenario: JSON merge for settings.local.json

- **GIVEN** `scaffold/.claude/fragment-settings.local.json` contains baseline permission rules
- **AND** `.claude/settings.local.json` exists with user-added permissions
- **WHEN** the sync tree-walk processes the fragment
- **THEN** the `fragment-` prefix is stripped → target is `.claude/settings.local.json`
- **AND** scaffold `permissions.allow` entries not already present are appended to the existing array
- **AND** existing user permissions are preserved

#### Scenario: JSON merge copies when target absent

- **GIVEN** `scaffold/.claude/fragment-settings.local.json` exists
- **AND** `.claude/settings.local.json` does not exist
- **WHEN** the sync tree-walk processes the fragment
- **THEN** the scaffold file is copied as `.claude/settings.local.json` (stripping `fragment-` prefix)

#### Scenario: Copy-if-absent for non-fragment files

- **GIVEN** `scaffold/docs/memory/index.md` exists (no `fragment-` prefix)
- **AND** `docs/memory/index.md` does not exist
- **WHEN** the sync tree-walk processes the file
- **THEN** the file is copied to `docs/memory/index.md`

#### Scenario: Copy-if-absent skips existing files

- **GIVEN** `scaffold/docs/specs/index.md` exists
- **AND** `docs/specs/index.md` already exists
- **WHEN** the sync tree-walk processes the file
- **THEN** no copy occurs — the existing file is preserved

## Sync: Generic Scaffold Tree-Walk

### Requirement: Replace Bespoke Sections with Generic Walk

`3-sync-workspace.sh` SHALL replace the per-file bespoke sections (current sections 2, 3, 4, 7, 8, 9) with a single generic scaffold tree-walk section. The tree-walk SHALL:

1. Discover all files under `scaffold/` recursively
2. For each file, compute the destination path by stripping the `scaffold/` prefix and any `fragment-` filename prefix
3. Create parent directories at the destination as needed (`mkdir -p`)
4. Dispatch to the appropriate strategy based on the `fragment-` prefix and file extension

No skip-list or exclusions — all 11 scaffold files are processed generically. Template files (config.yaml, constitution.md) are copied via copy-if-absent like any other non-fragment file. `/fab-setup` detects raw templates at runtime and overwrites them interactively (see [fab-setup Template Detection](#requirement-fab-setup-template-detection)).

Sections that remain unchanged: 1 (directories), 1b (fab/VERSION), 5 (skill symlinks), 4b (model tier classification), 6 (model tier agent files).

#### Scenario: Tree-walk processes all scaffold files

- **GIVEN** the scaffold directory contains 11 files
- **WHEN** the tree-walk executes
- **THEN** all 11 files are processed (3 fragment merges + 8 copy-if-absent)

#### Scenario: Tree-walk creates intermediate directories

- **GIVEN** `scaffold/.claude/fragment-settings.local.json` exists
- **AND** the repo root has no `.claude/` directory
- **WHEN** the tree-walk processes this file
- **THEN** `.claude/` directory is created before the merge operation

#### Scenario: Empty scaffold directory

- **GIVEN** `scaffold/` exists but contains no files
- **WHEN** the tree-walk executes
- **THEN** no files are processed and no errors occur

### Requirement: Helper Functions

The tree-walk SHALL extract merge logic into reusable helper functions within `3-sync-workspace.sh`:

**`line_ensure_merge <source> <target>`**: Read non-empty, non-comment lines from `<source>`. For each line, if not present in `<target>`, append it. If `<target>` does not exist, create it. If `<target>` is a symlink, resolve it to a real file first (preserving content). Report lines added.

**`json_merge_permissions <source> <target>`**: If `<target>` does not exist, copy `<source>` as-is. If `<target>` exists, use jq to merge `permissions.allow` arrays (append entries from source not already in target). If jq is not available, warn and skip. Report entries added.

#### Scenario: line_ensure_merge with symlink target

- **GIVEN** `.envrc` is a symlink to another file
- **AND** `fragment-.envrc` has entries to ensure
- **WHEN** `line_ensure_merge` is called
- **THEN** the symlink is replaced with a real file containing the resolved content
- **AND** missing entries are appended

#### Scenario: json_merge_permissions without jq

- **GIVEN** `fragment-settings.local.json` needs to be merged
- **AND** jq is not installed
- **WHEN** `json_merge_permissions` is called
- **THEN** a warning is printed: "WARN: jq not found — skipping {target} merge"
- **AND** no file is modified

### Requirement: Output Reporting

The tree-walk SHALL produce per-file output consistent with the existing script's reporting style:

- Fragment merge with additions: `Updated: {dest_path} (added {details})`
- Fragment merge with no additions: `{dest_path}: OK`
- Copy-if-absent (copied): `Created: {dest_path}`
- Copy-if-absent (already exists): no output (silent skip)

## References: Path Updates

### Requirement: fab-setup Skill Path References

`fab/.kit/skills/fab-setup.md` SHALL update all scaffold path references to use the new overlay structure:

| Old reference | New reference |
|---|---|
| `fab/.kit/scaffold/config.yaml` | `fab/.kit/scaffold/fab/config.yaml` |
| `fab/.kit/scaffold/constitution.md` | `fab/.kit/scaffold/fab/constitution.md` |
| `fab/.kit/scaffold/context.md` | `fab/.kit/scaffold/fab/context.md` |
| `fab/.kit/scaffold/code-quality.md` | `fab/.kit/scaffold/fab/code-quality.md` |
| `fab/.kit/scaffold/code-review.md` | `fab/.kit/scaffold/fab/code-review.md` |
| `fab/.kit/scaffold/memory-index.md` | `fab/.kit/scaffold/docs/memory/index.md` |
| `fab/.kit/scaffold/specs-index.md` | `fab/.kit/scaffold/docs/specs/index.md` |

#### Scenario: fab-setup config create reads from new path

- **GIVEN** `fab/config.yaml` does not exist
- **WHEN** `/fab-setup` runs config create mode
- **THEN** it reads the template from `fab/.kit/scaffold/fab/config.yaml`

#### Scenario: fab-setup constitution create reads from new path

- **GIVEN** `fab/constitution.md` does not exist
- **WHEN** `/fab-setup` runs constitution create mode
- **THEN** it reads the skeleton from `fab/.kit/scaffold/fab/constitution.md`

### Requirement: Migration File Path Reference

`fab/.kit/migrations/0.7.0-to-0.8.0.md` SHALL update the scaffold reference from `fab/.kit/scaffold/code-quality.md` to `fab/.kit/scaffold/fab/code-quality.md`.

#### Scenario: Migration references updated path

- **GIVEN** the migration file `0.7.0-to-0.8.0.md` exists
- **WHEN** inspecting section 2 (Extract `code_quality:`)
- **THEN** the scaffold path reads `fab/.kit/scaffold/fab/code-quality.md`

## fab-setup: Template Detection

### Requirement: fab-setup Template Detection

`fab/.kit/skills/fab-setup.md` SHALL change the bootstrap detection logic for `config.yaml` and `constitution.md` from file-existence checks to raw-template checks. A file is a "raw template" if it is identical to its scaffold source (i.e., still contains unfilled placeholders).

**config.yaml** (step 1a): Treat as needing creation if the file is missing OR contains the placeholder `{PROJECT_NAME}`.

**constitution.md** (step 1b): Treat as needing creation if the file is missing OR contains the placeholder `{Project Name}`.

This enables the sync tree-walk to copy all scaffold files generically (including templates) without interfering with fab-setup's interactive generation.

#### Scenario: fab-setup detects raw config.yaml template

- **GIVEN** `fab/config.yaml` exists but contains `{PROJECT_NAME}` (copied by sync tree-walk)
- **WHEN** `/fab-setup` runs bootstrap step 1a
- **THEN** it enters config create mode (interactive prompting)
- **AND** overwrites the raw template with user-provided values

#### Scenario: fab-setup detects raw constitution.md template

- **GIVEN** `fab/constitution.md` exists but contains `{Project Name}` (copied by sync tree-walk)
- **WHEN** `/fab-setup` runs bootstrap step 1b
- **THEN** it enters constitution create mode (interactive generation)
- **AND** overwrites the raw template with generated content

#### Scenario: fab-setup skips already-configured files

- **GIVEN** `fab/config.yaml` exists and does NOT contain `{PROJECT_NAME}`
- **WHEN** `/fab-setup` runs bootstrap step 1a
- **THEN** it reports "config.yaml already exists — skipping" (unchanged behavior)

## Design Decisions

1. **Template detection over skip-list**: fab-setup detects raw templates via placeholder grep instead of the sync tree-walk maintaining a skip-list of fab-setup-owned files.
   - *Why*: Eliminates all special-casing in the tree-walk — every scaffold file is processed generically. fab-setup already knows the placeholder patterns since it fills them. The detection is a single grep per file.
   - *Rejected*: 5-entry skip-list in the tree-walk — adds maintenance burden and couples sync to fab-setup's file ownership. Also rejected: `scaffold/fab/` subtree exclusion — would incorrectly skip `scaffold/fab/sync/README.md` (sync-managed).

2. **Helper functions in 3-sync-workspace.sh**: Merge logic extracted to `line_ensure_merge` and `json_merge_permissions` functions within the same file, not a separate library.
   - *Why*: Only one consumer (the tree-walk). Extracting to a separate library would add indirection without reuse benefit. The functions are small (~20 lines each).
   - *Rejected*: Separate `lib/scaffold-merge.sh` — premature abstraction for a single consumer.

3. **Symlink migration absorbed into line_ensure_merge**: The legacy `.envrc` symlink-to-file migration is handled generically by the line-ensure function rather than as a separate pre-step.
   - *Why*: The symlink case is a 3-line check (is symlink → resolve → replace). Embedding it in `line_ensure_merge` makes the tree-walk fully generic without special-casing `.envrc`.
   - *Rejected*: Separate pre-cleanup step — adds a section for a one-time migration that fits naturally in the merge function.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `fragment-` prefix convention | Confirmed from intake #1 — explicitly agreed in conversation, prefix over manifest/suffix | S:95 R:80 A:90 D:95 |
| 2 | Certain | Three files get fragment- prefix (.envrc, .gitignore, settings.local.json) | Confirmed from intake #2 — only merge files, all others are copy-if-absent or templates | S:90 R:85 A:95 D:90 |
| 3 | Confident | Skill/agent sync sections (5, 6) remain unchanged | Confirmed from intake #3 — they operate on skills/, not scaffold | S:80 R:90 A:85 D:85 |
| 4 | Confident | No migration needed for existing projects | Confirmed from intake #4 — scaffold lives inside .kit/ which is the engine | S:75 R:90 A:80 D:85 |
| 5 | Certain | No skip-list — template detection in fab-setup instead | User-directed — fab-setup checks for placeholder strings, tree-walk is fully generic | S:95 R:85 A:90 D:95 |
| 6 | Certain | JSON merge targets permissions.allow specifically | Config/constitution, codebase confirm only one JSON fragment exists; specific merge matches current behavior | S:85 R:85 A:90 D:90 |
| 7 | Confident | Symlink migration absorbed into line_ensure_merge | Small, generic check; avoids special-casing .envrc in the tree-walk | S:70 R:85 A:75 D:80 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
