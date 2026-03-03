# Intake: Update Underscore Skill References

**Change**: 260303-6b7c-update-underscore-skill-references
**Created**: 2026-03-03
**Status**: Draft

## Origin

> Now that underscore skill files (`_preamble.md`, `_scripts.md`, `_generation.md`) are deployed alongside regular skills (via the sync script change), update all references in skill bodies, documentation, and tests to use the simpler co-located paths instead of the old `./fab/.kit/skills/` paths.

Conversational mode. This change follows directly from the sync script fix that removed the `_*.md` filter in `2-sync-workspace.sh` and added `user-invocable: false` frontmatter to underscore files. The deployment is done; now the references need to match.

## Why

1. **Path confusion is the original problem**: Agents frequently fail to find `_preamble.md` because they see their skill's "Base directory" (e.g., `.claude/skills/fab-new/`) and try to read `_preamble.md` relative to *that* directory, not the repo root. The old path `./fab/.kit/skills/_preamble.md` only works from repo root CWD.

2. **If we don't fix it**: The sync change deployed underscore files co-located with skills, but skills still reference the old `./fab/.kit/skills/` path. Agents will continue trying the old path first, hitting the same confusion. The deployment change is inert without updating the references.

3. **Simpler paths are better**: Now that `_preamble.md` is a sibling skill (e.g., `.claude/skills/_preamble/SKILL.md`), skills can reference it by short name or a consistent co-located path. This is more robust across agent types (Claude Code, OpenCode, Codex, Gemini) regardless of their deployment format.

## What Changes

### 1. Skill file reference updates (~15 files in `fab/.kit/skills/`)

Two reference patterns exist and both need updating:

**Pattern A — Top-of-file instruction** (appears in every skill except `_preamble.md` itself):
```markdown
# Before
> Read and follow the instructions in `./fab/.kit/skills/_preamble.md` before proceeding.

# After — TBD at spec stage: either short name or a standardized co-located reference
```

**Pattern B — Inline shorthand** (within skill bodies):
```markdown
# These references like:
per `_preamble.md` §2
(`_generation.md`)
(`_scripts.md`)
```

Pattern B may already work naturally since agents can now find the co-located file. The spec stage should determine whether these need updating or are already sufficient.

### 2. Memory file updates (~10 files in `docs/memory/fab-workflow/`)

Files like `context-loading.md`, `kit-architecture.md`, `planning-skills.md`, etc. reference `_preamble.md` and `_scripts.md` by their `fab/.kit/skills/` path. Update to match whatever canonical reference form is chosen.

### 3. Spec file updates (2 files in `docs/specs/`)

`skills.md` and `glossary.md` reference underscore files. Update path references.

### 4. Test file updates (1 file)

`src/lib/sync-workspace/test.bats` — may need assertions updated since underscore files are now included in the deployment count (was 21, now 24).

### 5. _preamble.md self-reference update

`_preamble.md` itself instructs skills to reference it. The "Also read `fab/.kit/skills/_scripts.md`" instruction in §1 needs updating to match the new convention.

## Affected Memory

- `fab-workflow/context-loading`: (modify) Update path references to underscore skill files
- `fab-workflow/kit-architecture`: (modify) Document that underscore files are now deployed alongside skills
- `fab-workflow/planning-skills`: (modify) Update references to `_preamble.md` and `_generation.md`
- `fab-workflow/execution-skills`: (modify) Update references to `_preamble.md` and `_generation.md`

## Impact

- `fab/.kit/skills/*.md` — all skill files with underscore references (~15)
- `docs/memory/fab-workflow/` — memory files (~10)
- `docs/specs/` — spec files (2)
- `src/lib/sync-workspace/test.bats` — test assertions
- `references/speckit/commands.md` — reference doc (1)
- Deployed copies in `.claude/`, `.agents/`, `.opencode/` auto-update on next sync run

## Open Questions

- What should the canonical reference form be? Options: (a) short name `_preamble.md` everywhere, (b) keep `fab/.kit/skills/_preamble.md` for the initial "read this" instruction but use short names inline, (c) use the Skill tool invocation pattern instead of explicit file reads. Needs decision at spec stage.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Underscore files are now deployed co-located | Discussed — sync script change already applied and verified (24/24) | S:95 R:90 A:95 D:95 |
| 2 | Certain | Pattern B (inline shorthand) references may already work | Agents can find co-located `_preamble` as a sibling skill directory | S:85 R:90 A:85 D:90 |
| 3 | Certain | Deployed copies auto-update on sync | By design — only canonical sources in `fab/.kit/skills/` need editing | S:95 R:95 A:95 D:95 |
| 4 | Confident | Archive files should NOT be updated | Archived changes are historical artifacts; updating them has no operational value | S:80 R:90 A:85 D:80 |
| 5 | Tentative | The canonical reference form for Pattern A | Multiple valid options — short name, repo-root path, or Skill tool invocation | S:60 R:75 A:55 D:45 |
<!-- assumed: Reference form — needs spec-stage decision; all options are viable with different tradeoffs -->

5 assumptions (3 certain, 1 confident, 1 tentative).
