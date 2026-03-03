# Spec: Update Underscore Skill References

**Change**: 260303-6b7c-update-underscore-skill-references
**Created**: 2026-03-03
**Affected memory**: `docs/memory/fab-workflow/context-loading.md`, `docs/memory/fab-workflow/kit-architecture.md`, `docs/memory/fab-workflow/planning-skills.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Changing how the sync script deploys underscore files — that change is already done and working
- Updating archived changes in `fab/changes/archive/` — these are historical artifacts
- Modifying the underscore files' internal logic or content (only updating references *to* them)
- Changing how `references/speckit/commands.md` works — the `agent_scripts:` references there are unrelated to underscore skill files

## Design Decisions

1. **Keep `fab/.kit/skills/_preamble.md` as the canonical reference form (drop `./` prefix)**
   - *Why*: The `./` prefix adds no information and can confuse agents that try to resolve it relative to their skill's base directory (e.g., `.claude/skills/fab-new/`). The repo-root-relative path `fab/.kit/skills/_preamble.md` is unambiguous and already used by `fab-switch.md`. This is the minimal change that fixes the path confusion.
   - *Rejected*: Bare `_preamble.md` in the top-of-file instruction — not a valid Read tool path, would require the agent to guess where to find it. Skill tool invocation — `_preamble.md` has `user-invocable: false` and `disable-model-invocation: true`, so it can't be invoked.

2. **Leave Pattern B inline shorthand references unchanged**
   - *Why*: Inline references like `` `_preamble.md` §2 `` and `` (`_generation.md`) `` are human-readable shorthand within skill bodies, not Read tool paths. They already work naturally as agents can locate the co-located skill files. Changing them would be churn with no functional benefit.
   - *Rejected*: Full paths everywhere — clutters the prose and adds no operational value for inline references.

## Skills: Pattern A Top-of-File Instruction

### Requirement: Standardize top-of-file preamble references

All skill files in `fab/.kit/skills/` that reference `_preamble.md` in their top-of-file instruction SHALL use the form `fab/.kit/skills/_preamble.md` (no leading `./`).

The standard instruction line SHALL be:
```
> Read and follow the instructions in `fab/.kit/skills/_preamble.md` before proceeding.
```

The `fab-switch.md` variant SHALL remain:
```
> Read `fab/.kit/skills/_preamble.md` first. Only after that Read completes, proceed with any Bash calls.
```

#### Scenario: Agent reads a skill and resolves preamble path
- **GIVEN** an agent loads any skill from `.claude/skills/{name}/SKILL.md`
- **WHEN** the agent encounters the top-of-file instruction
- **THEN** the instruction references `fab/.kit/skills/_preamble.md` (no `./` prefix)
- **AND** the agent can resolve this path from the repo root CWD

#### Scenario: All 11 skill files with the standard instruction are updated
- **GIVEN** the files: `fab-ff.md`, `fab-archive.md`, `fab-setup.md`, `fab-clarify.md`, `fab-status.md`, `fab-new.md`, `fab-continue.md`, `fab-fff.md`, `docs-hydrate-memory.md`, `fab-discuss.md`, `docs-hydrate-specs.md`
- **WHEN** the `./` prefix is removed from the top-of-file instruction
- **THEN** each file's line 8 reads: `> Read and follow the instructions in \`fab/.kit/skills/_preamble.md\` before proceeding.`

#### Scenario: fab-switch.md variant is updated
- **GIVEN** `fab-switch.md` uses a different instruction format (line 8)
- **WHEN** the `./` prefix is removed (if present) or the path is standardized
- **THEN** line 8 reads: `> Read \`fab/.kit/skills/_preamble.md\` first. Only after that Read completes, proceed with any Bash calls.`

## Skills: _preamble.md Self-Reference

### Requirement: Update _preamble.md's own reference instruction

`_preamble.md` SHALL reference itself using `fab/.kit/skills/_preamble.md` (no `./` prefix) in the example instruction it shows skills should use.

#### Scenario: _preamble.md example instruction
- **GIVEN** `_preamble.md` line 12 contains the example instruction that skills should follow
- **WHEN** the example instruction text is read
- **THEN** it shows: `` `Read and follow the instructions in fab/.kit/skills/_preamble.md before proceeding.` ``

### Requirement: Update _preamble.md's _scripts.md reference

The "Also read" instruction in `_preamble.md` §1 (Always Load) that references `_scripts.md` SHALL use `fab/.kit/skills/_scripts.md` (no `./` prefix).

#### Scenario: _scripts.md reference in Always Load section
- **GIVEN** `_preamble.md` §1 contains an instruction to also read `_scripts.md`
- **WHEN** the instruction is read
- **THEN** it references `fab/.kit/skills/_scripts.md` (no `./`)

## Skills: internal-skill-optimize.md Reference

### Requirement: Update internal-skill-optimize.md preamble path

`internal-skill-optimize.md` references `fab/.kit/skills/_preamble.md` in its instruction list. This SHALL use the form without `./` prefix.

#### Scenario: internal-skill-optimize step 1
- **GIVEN** `internal-skill-optimize.md` line 21 references `_preamble.md` for context loading
- **WHEN** the reference is read
- **THEN** it uses `fab/.kit/skills/_preamble.md` (no `./`)

## Tests: Sync Workspace Test Assertions

### Requirement: Update stale test asserting underscore files are NOT deployed

The test in `src/lib/sync-workspace/test.bats` at line 288 (`"skips _preamble.md partial (not deployed as skill)"`) SHALL be updated to assert that underscore files ARE deployed, since the sync script now includes them.

#### Scenario: Test asserts underscore files are deployed
- **GIVEN** `2-sync-workspace.sh` deploys all `*.md` files from `fab/.kit/skills/` including underscore files
- **WHEN** the test suite runs
- **THEN** the test asserts `_preamble`, `_generation`, and `_scripts` directories exist in `.claude/skills/`
- **AND** the test asserts the corresponding `SKILL.md` files exist within each directory

#### Scenario: Test validates underscore file content
- **GIVEN** underscore files are deployed as skills
- **WHEN** the test checks the deployed copies
- **THEN** the deployed `SKILL.md` files are byte-accurate copies of the source `fab/.kit/skills/_*.md` files

## Memory: Update path references

### Requirement: Update memory file references to underscore skill files

Memory files in `docs/memory/fab-workflow/` that reference underscore files by their `fab/.kit/skills/` path SHALL use the form without `./` prefix. Bare shorthand references (e.g., `` `_preamble.md` ``) SHALL remain unchanged.

#### Scenario: context-loading.md references
- **GIVEN** `context-loading.md` references `_preamble.md` by full path in multiple locations
- **WHEN** the references are updated
- **THEN** all full-path references use `fab/.kit/skills/_preamble.md` (no `./`)
- **AND** bare shorthand references like `` `_preamble.md` `` are unchanged

#### Scenario: kit-architecture.md references
- **GIVEN** `kit-architecture.md` references underscore files in the directory structure listing and elsewhere
- **WHEN** the references are updated
- **THEN** any full-path references use `fab/.kit/skills/` without `./` prefix

#### Scenario: planning-skills.md references
- **GIVEN** `planning-skills.md` references `_generation.md` and `_preamble.md` extensively
- **WHEN** the references are updated
- **THEN** full-path references use `fab/.kit/skills/` without `./` prefix
- **AND** bare shorthand references are unchanged

#### Scenario: execution-skills.md references
- **GIVEN** `execution-skills.md` references `_preamble.md` in several changelog entries
- **WHEN** the references are updated
- **THEN** full-path references use `fab/.kit/skills/` without `./` prefix

## Specs: Update path references

### Requirement: Update spec file references to underscore skill files

Spec files in `docs/specs/` that reference underscore files SHALL use bare shorthand where currently used (no path change needed for `` `_preamble.md` `` form).

#### Scenario: glossary.md reference
- **GIVEN** `glossary.md` references `` `_preamble.md` `` (bare shorthand)
- **WHEN** the change is applied
- **THEN** the reference is unchanged (already using shorthand form)

#### Scenario: skills.md reference
- **GIVEN** `skills.md` references `` `_preamble.md` §1 `` (bare shorthand)
- **WHEN** the change is applied
- **THEN** the reference is unchanged (already using shorthand form)

## Kit Architecture Memory: Document underscore deployment

### Requirement: Update kit-architecture.md to reflect underscore file deployment

The kit-architecture memory file SHALL document that underscore files (`_preamble.md`, `_generation.md`, `_scripts.md`) are now deployed alongside regular skills by the sync script, with `user-invocable: false` frontmatter to prevent direct invocation.

#### Scenario: Directory structure listing includes underscore files in deployed skills
- **GIVEN** `kit-architecture.md` documents the skills directory structure
- **WHEN** the reader checks the deployment section
- **THEN** underscore files are listed as deployed skills alongside regular skills

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Underscore files are now deployed co-located | Confirmed from intake #1 — sync script verified, `ls .claude/skills/` shows `_preamble`, `_generation`, `_scripts` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Deployed copies auto-update on sync | Confirmed from intake #3 — only canonical sources in `fab/.kit/skills/` need editing | S:95 R:95 A:95 D:95 |
| 3 | Certain | Use `fab/.kit/skills/_preamble.md` (no `./`) as canonical form | Resolved — `fab-switch.md` already uses this form, it's unambiguous, minimal change | S:90 R:85 A:90 D:90 |
| 4 | Certain | Pattern B inline shorthand references stay unchanged | Resolved — shorthand like `_preamble.md` §2 is human-readable, not a Read tool path, already works | S:90 R:90 A:85 D:90 |
| 5 | Confident | Archive files should NOT be updated | Confirmed from intake #4 — archived changes are historical artifacts; updating them has no operational value | S:80 R:90 A:85 D:80 |
| 6 | Certain | Test file needs updating from "skips" to "deploys" | Verified — test at line 288 contradicts the sync script behavior (deploys all `*.md` including `_*.md`) | S:95 R:90 A:95 D:95 |
| 7 | Certain | Spec files need no changes | Verified — `glossary.md` and `skills.md` already use bare shorthand references | S:90 R:95 A:90 D:95 |
| 8 | Confident | Memory files: only full-path refs need `./` removal | Many memory references use bare shorthand already; only a few use full paths with `./` prefix | S:80 R:85 A:80 D:80 |

8 assumptions (6 certain, 2 confident, 0 tentative, 0 unresolved).
