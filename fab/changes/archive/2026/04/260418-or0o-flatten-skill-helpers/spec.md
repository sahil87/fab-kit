# Spec: Flatten Skill Helper Include Tree

**Change**: 260418-or0o-flatten-skill-helpers
**Created**: 2026-04-18
**Affected memory**: `docs/memory/fab-workflow/context-loading.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- **No binary-level enforcement of `helpers:` values.** `fab sync` will not reject skills with unknown helper names. Convention-only keeps the change in the markdown layer; enforcement is a follow-up if needed.
- **No rename of `_`-prefixed helper file/folder convention.** Backlog item 84bh explored renaming for visibility; this change addresses the root cause (unreliable 2nd-layer loads) structurally, leaving the `_` prefix convention untouched.
- **No change to `_generation`, `_review`, `_cli-external`.** These are already well-scoped inline references from specific skills. Refactoring them would regress.
- **No change to subagent dispatch convention.** Subagent standard context (5 `fab/project/**` files) remains as defined in `_preamble.md` § Subagent Dispatch.
- **No removal of fab CLI reference material.** Content moved out of `_cli-fab.md` during compression MUST land in `docs/memory/fab-workflow/` if not retained — it MUST NOT be lost.

## Helper Tree: Always-Load Layer

### Requirement: Single universal helper

The always-load layer SHALL consist of exactly one skill helper file: `_preamble.md`. All other helpers (`_generation`, `_review`, `_cli-fab`, `_cli-external`) SHALL be opt-in per skill via a `helpers:` frontmatter list.

#### Scenario: Read-only skill loads only _preamble

- **GIVEN** a read-only skill like `/fab-discuss` with no `helpers:` declared in its frontmatter
- **WHEN** the skill is invoked
- **THEN** the agent SHALL read `.claude/skills/_preamble/SKILL.md`
- **AND** SHALL NOT read `_cli-fab`, `_naming`, or `_cli-rk` (the latter two no longer exist as separate files)
- **AND** SHALL NOT read any other helper

#### Scenario: Planning skill loads declared helpers

- **GIVEN** `fab-continue` with frontmatter `helpers: [_generation, _review]`
- **WHEN** the skill is invoked
- **THEN** the agent SHALL read `.claude/skills/_preamble/SKILL.md`
- **AND** SHALL read `.claude/skills/_generation/SKILL.md`
- **AND** SHALL read `.claude/skills/_review/SKILL.md`
- **AND** SHALL NOT read any undeclared helper

### Requirement: `_preamble` contains no external "also read" directives

`_preamble.md` MUST NOT contain any "Also read the `_X`" directive that instructs the agent to load another helper file. The `helpers:` frontmatter field in each invoking skill is the sole mechanism for opting into additional helpers.

#### Scenario: Fanout directives removed

- **GIVEN** the current `_preamble.md` containing three "Also read" paragraphs (lines 48, 50, 52 in the deployed copy — pointing to `_cli-fab`, `_naming`, `_cli-rk`)
- **WHEN** this change is applied
- **THEN** all three paragraphs SHALL be removed from `src/kit/skills/_preamble.md`
- **AND** the replacement SHALL be a single paragraph referring the reader to the `Skill Helper Declaration` convention defined later in the same file

## Helper Tree: Inlined Content

### Requirement: Naming conventions inlined into `_preamble`

The body content of `src/kit/skills/_naming.md` SHALL be inlined into `_preamble.md` as a `## Naming Conventions` subsection. After inlining, `src/kit/skills/_naming.md` SHALL be deleted.

#### Scenario: _naming file no longer exists

- **GIVEN** `src/kit/skills/_naming.md` currently exists (76 lines of content)
- **WHEN** this change is applied
- **THEN** `src/kit/skills/_naming.md` SHALL NOT exist
- **AND** all its body content (excluding YAML frontmatter) SHALL appear as a `## Naming Conventions` subsection in `src/kit/skills/_preamble.md`
- **AND** after `fab sync`, `.claude/skills/_naming/` SHALL NOT exist as a deployed directory

#### Scenario: Inline references updated

- **GIVEN** `src/kit/skills/git-branch.md` and `src/kit/skills/git-pr.md` each contain the line `> Branch naming conventions are defined in _naming.md.`
- **WHEN** this change is applied
- **THEN** each occurrence SHALL be rewritten to `> Branch naming conventions are defined in _preamble.md § Naming Conventions.`

### Requirement: Run-Kit reference inlined into `_preamble`

The body content of `src/kit/skills/_cli-rk.md` SHALL be inlined into `_preamble.md` as a `## Run-Kit (rk) Reference` subsection. After inlining, `src/kit/skills/_cli-rk.md` SHALL be deleted. `docs/specs/skills/SPEC-_cli-rk.md` SHALL also be deleted.

#### Scenario: _cli-rk file and spec removed

- **GIVEN** `src/kit/skills/_cli-rk.md` (91 lines) and `docs/specs/skills/SPEC-_cli-rk.md` (31 lines) both exist
- **WHEN** this change is applied
- **THEN** neither file SHALL exist
- **AND** `_cli-rk`'s body content SHALL appear as a `## Run-Kit (rk) Reference` subsection in `src/kit/skills/_preamble.md`
- **AND** the silent-fail-when-rk-missing behavior documented in the original file SHALL be preserved verbatim in the inlined subsection

### Requirement: Common fab commands inlined into `_preamble`

The 6 most-used runtime `fab` command families (verified via grep across `src/kit/skills/`) — `status` (with subcommands: `finish`, `advance`, `set-checklist`, `fail`, `reset`), `change` (with subcommands: `new`, `switch`, `resolve`), `resolve` (pure-query alias), `score` (with `--check-gate`), `preflight`, and `log command` — SHALL be documented in a new `## Common fab Commands` subsection of `_preamble.md`. The subsection SHALL present each command as a condensed table row with purpose and canonical invocation form, and SHALL cross-reference `_cli-fab` for exhaustive flag documentation.

#### Scenario: Common commands accessible without _cli-fab

- **GIVEN** a skill that only uses commands from the top-6 families (e.g., `fab preflight`, `fab score --check-gate`, `fab log command`, `fab change resolve --folder`, `fab resolve --folder`, `fab status advance`)
- **WHEN** the skill is invoked without `_cli-fab` in its `helpers:` list
- **THEN** the `Common fab Commands` subsection of `_preamble.md` SHALL contain sufficient information for the agent to issue those commands correctly
- **AND** the skill SHALL NOT require loading `_cli-fab`

## Helper Tree: `_cli-fab` Compression

### Requirement: `_cli-fab.md` compressed to ≤300 lines

`src/kit/skills/_cli-fab.md` SHALL be compressed from its current 773 lines to no more than 300 lines while preserving all canonical command and flag documentation. Content removed MUST either be (a) redundant prose that can be replaced by tables, (b) historical rationale relocated to `docs/memory/fab-workflow/`, or (c) duplicates of the common-commands subsection in `_preamble`.

#### Scenario: Compression preserves canonical reference

- **GIVEN** the current 773-line `src/kit/skills/_cli-fab.md`
- **WHEN** compression is applied
- **THEN** the resulting file SHALL be ≤300 lines
- **AND** every command and flag documented in the original SHALL either be retained, cross-referenced from `_preamble`'s Common fab Commands, or relocated to `docs/memory/fab-workflow/` with a pointer comment in `_cli-fab`
- **AND** no canonical flag behavior documentation SHALL be lost

#### Scenario: Operator skill still works

- **GIVEN** `/fab-operator` declares `helpers: [_cli-fab, _cli-external]`
- **WHEN** the skill is invoked after compression
- **THEN** the agent SHALL have access to all fab commands it needs (via `_preamble` common commands + compressed `_cli-fab`)
- **AND** all operator-specific usage documented in `fab-operator.md` SHALL remain supported

## Helper Tree: Skill Helper Declaration Convention

### Requirement: `helpers:` frontmatter field defined in `_preamble`

`_preamble.md` MUST define a `Skill Helper Declaration` subsection documenting the `helpers:` frontmatter field, including its allowed values (`_generation`, `_review`, `_cli-fab`, `_cli-external`), its semantics (list of additional helpers to load after `_preamble`), and its default (empty — load only `_preamble`).

#### Scenario: Convention is documented

- **GIVEN** a new skill author reading `_preamble.md`
- **WHEN** they look up how to declare additional helpers
- **THEN** they SHALL find a `## Skill Helper Declaration` subsection listing the 4 allowed values and an example frontmatter block
- **AND** the subsection SHALL explicitly state that `_naming` and `_cli-rk` are not allowed values (they are inlined into `_preamble`)
- **AND** the subsection SHALL explicitly state that `_preamble` itself is implicit and never listed

### Requirement: Every skill has correct `helpers:` frontmatter

After this change, every skill file in `src/kit/skills/` SHALL have its `helpers:` frontmatter declared per the following mapping, or omitted (equivalent to empty):

| Skill | `helpers:` value |
|---|---|
| fab-new | `[_generation]` |
| fab-draft | `[_generation]` |
| fab-continue | `[_generation, _review]` |
| fab-ff | `[_generation, _review]` |
| fab-fff | `[_generation, _review]` |
| fab-operator | `[_cli-fab, _cli-external]` |
| All others (19 skills: fab-discuss, fab-status, fab-help, fab-clarify, fab-archive, fab-setup, fab-switch, fab-proceed, docs-hydrate-memory, docs-hydrate-specs, docs-reorg-memory, docs-reorg-specs, internal-consistency-check, internal-retrospect, internal-skill-optimize, git-branch, git-pr, git-pr-review) | omitted or `[]` |

#### Scenario: Audit matches the mapping

- **GIVEN** the post-change state of `src/kit/skills/`
- **WHEN** each skill's frontmatter is inspected
- **THEN** the `helpers:` field SHALL match the table above
- **AND** no skill SHALL reference a helper not listed in its frontmatter (except via inline body references, which are not auto-loaded)

## Memory and Specs Updates

### Requirement: `context-loading.md` reflects the new opt-in model

`docs/memory/fab-workflow/context-loading.md` MUST be updated so that the `### Always Load Layer` section no longer lists `_cli-fab`, `_naming`, or `_cli-rk` as part of the universal always-load set. The section SHALL instead describe the single universal helper (`_preamble`) and the per-skill `helpers:` opt-in mechanism. A new design decision entry SHALL be added explaining the flatten-tree rationale.

#### Scenario: Memory reflects new truth

- **GIVEN** the current `context-loading.md` lists 3 always-load internal helpers after the 7 project files
- **WHEN** this change is hydrated
- **THEN** the `### Always Load Layer` subsection SHALL list only `_preamble` as the universal helper
- **AND** a `### Skill Helper Declaration (Opt-In)` subsection SHALL describe the `helpers:` frontmatter convention
- **AND** the changelog SHALL record this change

### Requirement: `kit-architecture.md` updated

`docs/memory/fab-workflow/kit-architecture.md` MUST be updated: the skills directory tree SHALL no longer list `_naming.md` or `_cli-rk.md`, and the always-load/selective mapping table SHALL reflect the new single-universal-helper model.

#### Scenario: Directory listing updated

- **GIVEN** the current `kit-architecture.md` contains a skills directory tree with `_naming.md` and references `_cli-rk.md` via the Always-Load `_cli-rk` design decision
- **WHEN** this change is hydrated
- **THEN** the directory tree SHALL omit `_naming.md`
- **AND** the Always-Load `_cli-rk` design decision SHALL be marked superseded (with a pointer to this change)
- **AND** the loading-map table SHALL list `_preamble` as the sole always-load helper

### Requirement: Specs updated

`docs/specs/skills/SPEC-preamble.md` MUST be updated to document the inlined Naming Conventions, Run-Kit Reference, Common fab Commands, and Skill Helper Declaration subsections. `docs/specs/skills/SPEC-_cli-rk.md` SHALL be deleted. `docs/specs/skills.md` MUST document the `helpers:` frontmatter field and allowed values.

#### Scenario: SPEC-preamble reflects new structure

- **GIVEN** the current `SPEC-preamble.md` (84 lines) describes only the original sections
- **WHEN** this change is applied
- **THEN** `SPEC-preamble.md` SHALL include sections describing the new `## Naming Conventions`, `## Run-Kit (rk) Reference`, `## Common fab Commands`, and `## Skill Helper Declaration` subsections in `_preamble.md`
- **AND** `docs/specs/skills/SPEC-_cli-rk.md` SHALL NOT exist
- **AND** `docs/specs/skills.md` SHALL contain a subsection on the `helpers:` frontmatter field with its 4 allowed values

### Requirement: Backlog item 84bh removed

The line `[ ] [84bh] 2026-04-02: Try using references instead of _ folders in skills - other agents don't read from _ folders in skills` in `fab/backlog.md` SHALL be removed. The subsumption is cited in the hydrate output.

#### Scenario: Backlog item closed

- **GIVEN** `fab/backlog.md` currently contains backlog item `[84bh]`
- **WHEN** this change is applied
- **THEN** item `[84bh]` SHALL NOT appear in `fab/backlog.md`
- **AND** the change's intake and spec SHALL both note the subsumption

## Deprecated Requirements

### Universal `_cli-fab` Always-Load (via `_preamble` "also read")

**Reason**: Unreliable (agents frequently skip 2nd-layer includes) and wasteful (15 of 24 skills don't use it). Replaced by per-skill `helpers:` opt-in.

**Migration**: The 6 most-common `fab` commands are inlined into `_preamble` as a new `## Common fab Commands` subsection, covering the needs of ~90% of skills. Skills that genuinely need the full CLI reference declare `_cli-fab` in their `helpers:` frontmatter. The Always-Load `_cli-fab` mention in `docs/memory/fab-workflow/kit-architecture.md` and `context-loading.md` is superseded.

### Universal `_naming` Always-Load (via `_preamble` "also read")

**Reason**: Content is small (76 lines) — inlining into `_preamble` reduces file count without material bloat. Matches the project principle of preferring flat structure.

**Migration**: Content is inlined into `_preamble.md` as the `## Naming Conventions` subsection. Inline references in `git-branch.md` and `git-pr.md` are updated to point to the inlined section. File `src/kit/skills/_naming.md` is deleted.

### Optional `_cli-rk` Always-Load (via `_preamble` "also read")

**Reason**: Content is small (91 lines) and universally-relevant (rk is nice-to-have for any session) — inlining is strictly an improvement given the silent-fail-when-rk-missing design. Directly supersedes design decision "Always-Load `_cli-rk` Skill for rk Capabilities" (introduced in 260416-mgsm-add-cli-rk-skill).

**Migration**: Content is inlined into `_preamble.md` as the `## Run-Kit (rk) Reference` subsection. Files `src/kit/skills/_cli-rk.md` and `docs/specs/skills/SPEC-_cli-rk.md` are deleted.

## Design Decisions

1. **Per-skill `helpers:` frontmatter list (replaces universal "also read" fanout)**
   - *Why*: Makes the helper load set explicit, auditable, and grep-able. Agents reliably read frontmatter (it's parsed before body). Eliminates the silent-skip problem of 2nd-layer "also read" directives.
   - *Rejected*: (a) Inline body references in each skill — less discoverable, not machine-readable. (b) Deeper helper taxonomy with sub-helpers — worsens the skip-rate problem. (c) Keep status quo and rely on prompt caching — does nothing for correctness when pointers are silently skipped.

2. **Inline small helpers (`_naming`, `_cli-rk`) into `_preamble`**
   - *Why*: At 76 and 91 lines respectively, these are too small to justify file-level modularity when measured against the cost of an additional include. User principle: "No point of slimming down skill files too much."
   - *Rejected*: Keep as separate files with `helpers:` opt-in — would require adding two more allowed helper values for content that every skill that cares about naming or rk will want anyway. Worse fanout, not better.

3. **Compress `_cli-fab` rather than delete or fully inline**
   - *Why*: At 773 lines, it's 45% of total helper content but contains genuine canonical reference material for flag behavior. Compression (tables instead of prose, relocate historical rationale to memory) preserves the canonical role without inflating `_preamble` or every skill that might use a fab command.
   - *Rejected*: (a) Full inline of `_cli-fab` into `_preamble` — would add ~500 lines to the universal load, defeating the purpose. (b) Delete `_cli-fab` entirely and move everything to `_preamble` common commands — loses canonical reference for uncommon flags.

4. **Convention-only enforcement (no `fab sync` validation of `helpers:`)**
   - *Why*: Keeps the change scope markdown-only. Bad `helpers:` values surface as runtime behavior changes (agent either loads an unknown file and fails, or doesn't load a needed one and surfaces a missing-reference failure). Validation can be added later if drift becomes a problem.
   - *Rejected*: `fab sync` rejects unknown helpers — adds binary-side work for marginal benefit given the small, stable allowed-values set.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove "also read" fanout from `_preamble` | Confirmed from intake #1 — direct user agreement + cross-skill audit showing 15/24 skills don't use those helpers. No change at spec level. | S:95 R:80 A:90 D:95 |
| 2 | Certain | Leave `_generation`, `_review`, `_cli-external` alone | Confirmed from intake #2 — spec-level scope review agrees; these are well-scoped inline references. No change at spec level. | S:95 R:90 A:95 D:95 |
| 3 | Certain | Inline `_naming` + `_cli-rk` into `_preamble` | Confirmed from intake #3. Spec-level review: content combined is 167 lines — `_preamble` grows from 384 to ~550 lines, acceptable per user's "don't over-slim" principle. | S:95 R:80 A:90 D:95 |
| 4 | Certain | `helpers:` frontmatter is the opt-in mechanism | Upgraded from intake Confident. Spec-level analysis: frontmatter is YAML-parseable, grep-able, and agents load frontmatter reliably before body. Alternatives (inline body references, per-skill extra lists) all regress on auditability or discoverability. | S:90 R:70 A:90 D:85 |
| 5 | Certain | Change type is `refactor` | Confirmed from intake #7 — deterministic per taxonomy keyword rules. | S:95 R:95 A:95 D:95 |
| 6 | Certain | `SPEC-preamble.md` exists (no underscore), `SPEC-_cli-rk.md` exists | Confirmed from intake #11 — verified via `ls docs/specs/skills/`. | S:95 R:85 A:95 D:90 |
| 7 | Confident | Compress `_cli-fab` to ≤300 lines | Confirmed from intake #5. Spec-level reality check: the 300-line target is approximate; if canonical-reference content demands slightly more, overshoot slightly rather than lose fidelity. Non-Goals section pins "no lost canonical content" as the hard constraint. | S:80 R:60 A:80 D:75 |
| 8 | Confident | Inline top-6 common `fab` commands into `_preamble` | Confirmed from intake #6. The 6 chosen (`preflight`, `score`, `log command`, `change`, `resolve`, `status`) are justified by grep frequency across `src/kit/skills/`. | S:85 R:75 A:85 D:85 |
| 9 | Certain | Allowed `helpers:` values are exactly 4 (`_generation`, `_review`, `_cli-fab`, `_cli-external`) | Verified: cross-skill audit (grep across `src/kit/skills/`) shows these are the only helpers currently referenced. `_naming` and `_cli-rk` are inlined per this change; `_preamble` is implicit. | S:95 R:85 A:90 D:90 |
| 10 | Confident | No binary-level enforcement of `helpers:` needed | Confirmed from intake #10 and Design Decision #4. Pure markdown change has highest reversibility. | S:80 R:90 A:75 D:80 |
| 11 | Confident | Backlog item 84bh subsumption | Confirmed from intake #9. Spec-level commitment: this change removes item 84bh from `fab/backlog.md`. Naming-convention work remains possible as a separate follow-up. | S:80 R:85 A:80 D:80 |
| 12 | Certain | `docs/memory/fab-workflow/context-loading.md` and `kit-architecture.md` are the memory files to hydrate | Verified via grep: these two files contain the substantive references to the helpers being restructured (`also read` directives, always-load listings, helper-file inventory). No other memory file contains load-convention content. | S:90 R:90 A:90 D:90 |
| 13 | Certain | `fab-operator` reference list (lines 41–43) needs updating (remove `_naming.md` line, keep `_cli-fab.md` and `_cli-external.md`) | Verified via direct file inspection — lines 41-43 explicitly list helper file paths. | S:95 R:90 A:95 D:95 |
| 14 | Confident | `.claude/skills/_naming/` and `.claude/skills/_cli-rk/` deployed directories must be manually removed | New at spec level — `fab sync` deploys files from `src/kit/skills/` but doesn't clean up deployed directories when source files are deleted. Spec requires an explicit post-sync cleanup step in tasks. | S:85 R:80 A:80 D:80 |

14 assumptions (9 certain, 5 confident, 0 tentative, 0 unresolved).
