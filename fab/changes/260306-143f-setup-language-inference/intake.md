# Intake: Replace Template-Driven Language Detection with Agent-Inferred Conventions

**Change**: 260306-143f-setup-language-inference
**Created**: 2026-03-06
**Status**: Draft

## Origin

> Replace template-driven language detection in fab-setup with agent-inferred conventions. Remove step 1b-lang's dependency on fab/.kit/templates/ files.

This change emerged from a `/fab-discuss` session reviewing the outcome of three template changes (rwt1/Rust, qg80/TypeScript, 4vj0/React) generated on 2026-03-05. The discussion concluded that shipping language-specific conventions inside `fab/.kit/` violates the kit's neutrality principle (Constitution §V: Portability). The three template changes are being archived/deleted.

The discussion converged on a specific alternative: keep detection (it's neutral), replace content sourcing (templates → agent inference + project file reading), and write to user-owned `fab/project/*` files instead of kit-owned templates.

## Why

1. **Problem**: Step 1b-lang in `fab-setup.md` references `fab/.kit/templates/constitutions/{lang}.md` and `fab/.kit/templates/configs/{lang}.yaml` files. These bundle opinionated language conventions (e.g., "use thiserror for Rust errors") inside the kit, making judgment calls on behalf of all users and creating maintenance burden as ecosystems evolve.

2. **Consequence if unfixed**: The kit violates its own Constitution §V (portability — no assumptions about host project's language/toolchain). The templates directory bloats the kit with files irrelevant to most projects. Every new language/framework requires new template files to be authored and maintained.

3. **Why this approach**: The agent already knows language conventions from training data. What it needs from the constitution is project-specific deviations from standard practice. By reading the project's actual files (linter configs, `package.json`, `Cargo.toml`, etc.) at setup time, the agent can infer conventions grounded in what the project actually uses — not what a template author assumed.

## What Changes

### 1. Rewrite step 1b-lang in `fab/.kit/skills/fab-setup.md`

Replace the current template-lookup logic (lines 83–115) with a prompt-driven flow:

**Detection** (same marker-file approach, but expanded):
- Check for `Cargo.toml`, `tsconfig.json`, `package.json`, `go.mod`, `pyproject.toml`, etc.
- Read detected files for framework signals (e.g., "react" in `package.json` dependencies)
- Scan for linter/formatter configs: `.eslintrc`, `biome.json`, `clippy.toml`, `rustfmt.toml`, `.prettierrc`, `ruff.toml`

**Inference**:
- From detected files, the agent infers conventions using its training knowledge combined with the actual config values it read
- E.g., if `biome.json` has `"semicolons": "asNeeded"`, that becomes a convention; if Rust project has no `clippy.toml`, write standard Rust conventions

**Adaptive questions** (up to 3, free-form):
- Only ask about things that genuinely can't be inferred from the project files
- Questions are adaptive — if `vitest` is already in `package.json`, don't ask about test runner
- Examples of question-worthy gaps: error handling philosophy, architecture pattern, testing expectations when ambiguous
- SRAD does not apply to this flow

**Write to appropriate `fab/project/*` files**:
- MUST/SHOULD enforcement rules → `fab/project/constitution.md` (new section or fill existing scaffolds)
- Descriptive stack info ("we use React 19 with Vite") → `fab/project/context.md`
- Coding standards, anti-patterns → `fab/project/code-quality.md`
- Review policy additions → `fab/project/code-review.md`
- Source paths, checklist categories → `fab/project/config.yaml`

**Idempotency on re-run**:
- Read existing `fab/project/*` content before writing
- Merge/update without duplicating or overwriting user edits
- Detect what's already present and skip or augment

### 2. Delete template files from `fab/.kit/templates/`

Remove these directories entirely:
- `fab/.kit/templates/constitutions/` (contains `rust.md`, `node-typescript.md`, `react.md`)
- `fab/.kit/templates/configs/` (contains `rust.yaml`, `node-typescript.yaml`, `react.yaml`)

The `fab/.kit/templates/` directory itself stays — it still holds `intake.md`, `spec.md`, `tasks.md`, `checklist.md`, `status.yaml`, and other artifact templates.

### 3. Remove template advisory from `2-sync-workspace.sh`

Delete the "Language template advisory" block (lines 219–247 of `fab/.kit/sync/2-sync-workspace.sh`) which checks for `fab/.kit/templates/constitutions/` existence and suggests running `/fab-setup` when conventions are missing. This block references the template files being deleted.

## Affected Memory

- `fab-workflow/setup`: (modify) Update to reflect the new language-inference flow replacing template-driven detection

## Impact

- **`fab/.kit/skills/fab-setup.md`** — step 1b-lang rewritten (~30 lines replaced with ~30 lines of new prompt logic)
- **`fab/.kit/sync/2-sync-workspace.sh`** — language template advisory block removed (~28 lines deleted)
- **`fab/.kit/templates/constitutions/`** — directory deleted (3 files)
- **`fab/.kit/templates/configs/`** — directory deleted (3 files)
- No changes to bootstrap phases 0, 1a, 1b, 1b2–1h, config behavior, constitution behavior, or migrations behavior
- No changes to any other skills, scripts, or Go binary

## Open Questions

- None — the discussion resolved the design decisions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Kit stays language-neutral — no bundled convention templates | Discussed — Constitution §V mandates portability, user explicitly decided | S:95 R:90 A:95 D:95 |
| 2 | Certain | Detection via marker files is kept (same approach) | Discussed — detection is neutral, only content sourcing changes | S:90 R:95 A:90 D:95 |
| 3 | Certain | Convention content comes from agent inference + project file reading | Discussed — user explicitly chose this over templates | S:95 R:85 A:90 D:90 |
| 4 | Certain | Write to all `fab/project/*` files as appropriate | Discussed — user confirmed setup works on all project files, not just constitution | S:90 R:85 A:90 D:90 |
| 5 | Certain | Up to 3 adaptive free-form questions for unresolvable gaps | Discussed — user accepted adaptive questioning, rejected hard-coded questions | S:90 R:90 A:85 D:90 |
| 6 | Certain | Re-runs detect existing content and merge/update | Discussed — user agreed on idempotent merge behavior | S:85 R:80 A:85 D:90 |
| 7 | Certain | SRAD does not apply to the setup question flow | Discussed — user explicitly excluded SRAD from setup | S:90 R:90 A:85 D:95 |
| 8 | Certain | Delete `fab/.kit/templates/constitutions/` and `fab/.kit/templates/configs/` | Discussed — these are the rejected template files | S:95 R:85 A:95 D:95 |
| 9 | Confident | Remove template advisory block from `2-sync-workspace.sh` | References deleted template files — logically must be removed. User didn't explicitly discuss but said "delete convention detection logic if it references templates" | S:80 R:90 A:90 D:90 |
| 10 | Confident | Detection table in fab-setup.md expands to include Python (`pyproject.toml`) and more linter configs | Natural extension of "detect language/framework via marker files" — user mentioned these specific files | S:75 R:90 A:85 D:80 |

10 assumptions (8 certain, 2 confident, 0 tentative, 0 unresolved).
