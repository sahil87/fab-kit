# Intake: Repoint Skill Read-Paths to .agents/skills and Gate .claude on Claude

**Change**: 260908-yd9s-repoint-agents-skills-gate-claude
**Created**: 2026-09-08

## Origin

Promptless dispatch (`/fab-proceed` create-new, `{questioning-mode} = promptless-defer`) from a completed design conversation. Decisions below are final unless explicitly marked open.

> Re-tier the skill deploy targets to the clean taxonomy — unconditional = the portable Agent Skills standard (`.agents/skills/`); gated = brand-specific surfaces. Repoint fab-internal read-path references from `.claude/skills/` to `.agents/skills/` across `src/kit/skills/*.md`; flip the `.claude` deploy-target row in `src/go/fab-kit/internal/skills.go` from `AlwaysOn: true` back to gated on the `claude` CLI; gate the `.claude/settings.local.json` scaffold write on the same gate; sweep present-truth docs; update Go tests. No migration file. Designed follow-up to merged PR #647 (`260908-t513-unconditional-agent-skill-deploy-targets`, commit b0ae654a).

Key facts established in the design conversation (verified there, re-verified at intake where noted):

- Post-#647, the `.agents/skills/` copy is guaranteed present on every machine (`AlwaysOn` row in the `agentConfig` table) and is byte-identical to the `.claude` copy (`diff -r` exit 0; a live worker followed `.agents/skills/` paths successfully).
- Claude Code cannot read `.agents/skills/` (verified against docs and a live probe on Claude Code 2.1.263), so `.claude/skills/` remains the Claude channel — but only for machines that have Claude. Claude Code continues to LOAD user-invoked skills from `.claude/skills/` (its native scan); the repoint changes only in-body read references, so Claude Code UX is unchanged on machines with claude.
- If you run Claude Code, `claude` is on PATH — the CLI gate is near-perfect for that brand.
- `fab sync` currently writes `.claude/settings.local.json` unconditionally (seen as `Created: .claude/settings.local.json (15 permission rules)` in sync output).
- Measured repoint surface (re-verified at intake): **37 references across 23 files** in `src/kit/skills/*.md`.

## Why

1. **De-brands fab's internals**: dispatch prompts and helper-load instructions should target the harness-neutral, guaranteed-present copy (`.agents/skills/` — the portable Agent Skills standard), not a brand folder. This is the stepping stone to retiring `.claude/skills/` entirely if/when Claude Code ships `.agents/skills/` support (recorded follow-up idea from #647).
2. **Litter**: non-Claude projects (e.g. codex-only teams) currently get a `.claude/` tree (skills + settings.local.json) they never read. Post-#647 the `.claude` row is `AlwaysOn`, so every `fab sync` recreates it everywhere.
3. **Taxonomy purity**: #647 left `.claude` unconditional as a transitional shape. The clean tiering is: unconditional = the cross-client standard directory; gated = brand-specific surfaces (`.claude` on `claude`, `.opencode` on `opencode`). Now safely improvable because the `.agents` copy is guaranteed and byte-identical.

If not fixed: fab's own skill files keep hard-coding a brand path as the canonical read channel, every non-Claude user keeps accumulating an unread `.claude/` tree, and the eventual `.claude` retirement has no stepping stone.

**Risk note (recorded from the conversation)**: the repoint is a same-wording sibling sweep across 23+ files — this repo's #1 rework cause. The sweep-class distinctions in § What Changes are the mitigation and MUST carry into the plan.

**Rejected alternatives (recorded)**:
- **Keeping `.claude` unconditional** (the #647 shape) — taxonomy-impure, litter for non-Claude users; now safely improvable since `.agents` is guaranteed.
- **Repointing to `$(fab kit-path)/skills/` instead of `.agents/skills/`** — kit-cache paths are version-resolved and not workspace-visible to all harnesses; the deployed workspace copy is the contract.

## What Changes

### 1. Repoint fab-internal read-path references (`src/kit/skills/*.md`)

Change every fab-internal READ-PATH reference from `.claude/skills/…` to `.agents/skills/…` across `src/kit/skills/*.md`. Measured surface: **37 references across 23 files**, of which:

- **19 are the byte-identical boilerplate line** (one per skill file header):

  ```
  > Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.
  ```

  One of the 19 is `_preamble.md` itself quoting that line as the canonical sentence skills must carry ("Each skill file should begin with: …") — **update the quoted canonical form too**, or the sweep re-diverges on the next skill authored.

- **The rest are helper/skill read instructions**, e.g.:
  - `.claude/skills/_intake/SKILL.md` (fab-new/fab-draft/fab-proceed pointing at the shared procedure)
  - `.claude/skills/{skill}/SKILL.md` in dispatch-prompt sections (`_preamble.md` § Subagent Dispatch item 1, `_pipeline.md`, skill dispatch templates)
  - `_preamble.md` § Skill Helper Declaration semantics: "the agent MUST read `.claude/skills/{helper}/SKILL.md`" and the stage-conditional in-body read examples ("read `.claude/skills/_review/SKILL.md` …")

All become `.agents/skills/…`. Safe because post-#647 the `.agents/skills/` copy is AlwaysOn (guaranteed present) and byte-identical.

### 2. Re-gate the `.claude` deploy-target row (`src/go/fab-kit/internal/skills.go`)

Current `agentConfig` table (skills.go lines 53–56):

```go
agents := []agentConfig{
    {Label: "Claude Code", BaseDir: filepath.Join(repoRoot, ".claude", "skills"), Format: "directory", Mode: "copy", AlwaysOn: true},
    {Label: "OpenCode", CLIs: []string{"opencode"}, BaseDir: filepath.Join(repoRoot, ".opencode", "commands"), Format: "flat", Mode: "copy"},
    {Label: "Agents dir", BaseDir: filepath.Join(repoRoot, ".agents", "skills"), Format: "directory", Mode: "copy", AlwaysOn: true},
}
```

Change: the **Claude Code row** drops `AlwaysOn: true` and restores `CLIs: []string{"claude"}` (the pre-#647 candidate list). The **Agents dir row stays `AlwaysOn: true`**. The **OpenCode row is unchanged** (gated on `opencode`).

`FAB_AGENTS` semantics for gated rows are unchanged from #647: always-on rows bypass `agentAvailable`; the `.claude` row simply re-joins the gated set that `agentAvailable(clis...)` (skills.go, `FAB_AGENTS` env override then PATH lookup) governs.

### 3. Gate the `.claude/settings.local.json` scaffold write on the same `claude` gate

The writer is **not** in `deploySkills`: it is the generic scaffold tree-walk — `scaffoldTreeWalk` in `src/go/fab-kit/internal/scaffold.go` (called from `sync.go` ~line 98–100), which walks `$(kit)/scaffold/` and merges `src/kit/scaffold/.claude/fragment-settings.local.json` into the repo via `jsonMergePermissions`. Gate this write on the same `claude` availability check, otherwise a `.claude/` dir still appears for non-Claude users and the litter win is void.

**OPEN SUB-DECISION (recorded honestly, implementer's choice)**: the exact mechanism for sharing the gate between `deploySkills` and the scaffold writer — e.g. an exported/package-level helper, or a precomputed `claudeAvailable` bool passed in; and how the generic `scaffoldTreeWalk` learns which scaffold entries are claude-gated (e.g. skip entries whose dest path is under `.claude/` when the gate is closed). Front-runner: path-prefix skip in the walk keyed on one shared `agentAvailable("claude")` result. See Assumptions row 4.

### 4. Docs sweep (present-truth docs only)

Update for the new tiering:

- `docs/memory/distribution/kit-architecture.md` — § Agent Skill Deployment + § Design Decisions: the always-on-pair decision from #647 gets **re-scoped** (not deleted): `.agents` unconditional as the cross-client standard, `.claude` gated as brand surface; record the reversal rationale.
- `docs/memory/distribution/distribution.md`, `docs/memory/distribution/setup.md` (its deployment row).
- `docs/specs/architecture.md` (deploy-target table); `docs/specs/overview.md` + `docs/specs/skills.md` wherever they carry `.claude/skills/` read-path instructions.
- `docs/site/skill.md` + `docs/site/install.md` (verify; update only if they carry stale claims). Constitution § Toolkit Standards: any docs/site/ edit MUST be checked against the governing shll standards (re-fetch the live spec).
- `fab/project/constitution.md` Principle V + canonical-source constraint, and `fab/project/context.md` — **wording-only, no constitution version bump** (jjg0/udwv precedent; add the dated HTML comment as those did).
- `docs/memory/_shared/context-loading.md`, `docs/memory/runtime/agent-primitives.md`, `docs/memory/memory-docs/templates.md`, `docs/memory/pipeline/planning-skills.md` + `execution-skills.md` — wherever they carry `.claude/skills/` READ-PATH instructions.

**Critical sweep distinctions (MUST carry into plan tasks and review acceptance)**:

| Class | Rule |
|-------|------|
| (a) Read-path instructions ("read `.claude/skills/X/SKILL.md`") | Repoint to `.agents/skills/` |
| (b) Deploy-target descriptions ("skills are deployed to `.claude/skills/` …") | Stay `.claude`, wording updated to the gated contract ("when `claude` is present") |
| (c) Historical artifacts | **NEVER touched**: `src/kit/migrations/2.22.0-to-2.23.0.md`, all `docs/memory/**/log.md` + `log.seed.md`, `docs/specs/findings/*` |

### 5. Tests (Constitution: Go changes ship tests)

Update `src/go/fab-kit/internal/skills_test.go` + `sync_integration_test.go` for the re-tiered gates:

- `.agents/skills/` always present; `.claude/skills/` present iff `claude` ∈ `FAB_AGENTS`/PATH; `.opencode/commands/` unchanged; `settings.local.json` gated.
- `TestSync_FullRunProducesExpectedTree` runs under `FAB_AGENTS=claude` → should now expect `.claude/skills/` AND `.agents/skills/` AND `.claude/settings.local.json` present.
- Add/adjust a case with `FAB_AGENTS` lacking `claude` (e.g. `FAB_AGENTS=codex`) asserting `.claude/` **absent entirely** while `.agents/skills/` is fully deployed.

Constitution constraint: if any `fab` CLI command signature changes (none expected), `src/kit/skills/_cli-fab.md` must be updated.

### 6. No migration file

Nothing existing is restructured: a machine with claude keeps its `.claude/skills/`; a machine without claude simply stops receiving fresh copies (existing dirs are preserved by sync's skip path; the generated per-target `.gitignore` manifest means a user CAN safely delete a stale `.claude/skills/` by hand). **Verified at intake against `fab/project/context.md` § Migrations**: a migration is required only when a change "requires restructuring existing user data" — this change restructures nothing, so no migration ships.

## Affected Memory

- `distribution/kit-architecture`: (modify) § Agent Skill Deployment + § Design Decisions — re-scope the #647 always-on-pair decision; record reversal rationale
- `distribution/distribution`: (modify) deploy-target tiering description
- `distribution/setup`: (modify) deployment row
- `_shared/context-loading`: (modify) read-path references `.claude/skills/` → `.agents/skills/`
- `runtime/agent-primitives`: (modify) verify/repoint read-path references
- `memory-docs/templates`: (modify) verify; repoint only read-path references if present
- `pipeline/planning-skills`: (modify) verify/repoint read-path references
- `pipeline/execution-skills`: (modify) verify/repoint read-path references

(Historical artifacts excluded per sweep class (c): all `log.md`/`log.seed.md` untouched.)

## Impact

- **Go**: `src/go/fab-kit/internal/skills.go` (agentConfig table), `scaffold.go` and/or `sync.go` (scaffold gate), `skills_test.go`, `sync_integration_test.go`.
- **Kit skills**: 23 files under `src/kit/skills/` (37 read-path references; 19 identical boilerplate lines + helper-load/dispatch-prompt references).
- **Docs**: ~12 present-truth files across `docs/memory/` (5 domains), `docs/specs/` (architecture, overview, skills), `docs/site/` (verify skill.md, install.md), `fab/project/` (constitution.md wording-only, context.md).
- **Behavior contract**: `fab sync` output changes on machines without `claude` (no `.claude/` writes); unchanged on machines with `claude`.
- **Release tier**: feat → next release MINOR.
- **Not touched**: `.opencode/commands/` gating, `src/kit/migrations/*`, all logs/findings, deployed-copy directories (regenerated by sync).

## Open Questions

- (none — the one open sub-decision is recorded as Assumptions row 4, delegated to the implementer by the design conversation)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change type is `feat`, set explicitly (not left to inference) | Mandated by the design conversation; inference has repeatedly mis-typed similar changes | S:95 R:90 A:95 D:95 |
| 2 | Certain | Repoint ALL 37 read-path references (23 files) in `src/kit/skills/*.md` to `.agents/skills/`, including `_preamble.md`'s quoted canonical boilerplate sentence | Discussed — measured surface re-verified at intake (grep: 23 files, 37 refs); `.agents` copy is AlwaysOn and byte-identical post-#647 | S:90 R:75 A:90 D:90 |
| 3 | Certain | `.claude` row → `CLIs: ["claude"]` gated; `.agents` row stays `AlwaysOn`; `.opencode` row unchanged | Discussed — final decision; verified current table at skills.go:53-56 matches the described #647 shape | S:95 R:80 A:90 D:95 |
| 4 | Tentative | Gate-sharing mechanism for the settings.local.json scaffold: skip scaffold entries destined under `.claude/` in `scaffoldTreeWalk` when a shared `agentAvailable("claude")` result is false (exported helper vs precomputed bool = implementer's choice) | Marked OPEN in the conversation, explicitly delegated to implementer; writer located at intake (scaffoldTreeWalk, scaffold.go, fed by `src/kit/scaffold/.claude/fragment-settings.local.json`); 2–3 mechanisms, clear front-runner, trivially reversible | S:60 R:85 A:85 D:55 |
| 5 | Certain | Three-class sweep discipline: (a) read-paths repoint, (b) deploy-target descriptions stay `.claude` with gated wording, (c) historical artifacts (migrations, log.md/log.seed.md, findings) never touched | Discussed — mandated mitigation for the same-wording sibling-sweep risk (this repo's #1 rework cause) | S:90 R:70 A:90 D:90 |
| 6 | Certain | Constitution Principle V / context.md updates are wording-only — no constitution version bump, dated HTML comment appended | Discussed — jjg0/udwv precedent cited in the conversation; no MUST-rule changes | S:90 R:85 A:90 D:90 |
| 7 | Certain | No migration file ships | Discussed and verified at intake against context.md § Migrations: nothing existing is restructured; sync's skip path preserves existing dirs; manifest enables safe manual deletion | S:90 R:80 A:90 D:90 |
| 8 | Certain | Test surface: skills_test.go + sync_integration_test.go; `TestSync_FullRunProducesExpectedTree` (FAB_AGENTS=claude) expects `.claude` + `.agents` + settings.local.json; new/adjusted case with FAB_AGENTS lacking claude asserts `.claude/` absent, `.agents/skills/` full | Discussed — specific test expectations given verbatim; FAB_AGENTS gate semantics unchanged from #647 (verified agentAvailable at skills.go:107+) | S:90 R:85 A:90 D:90 |
| 9 | Confident | docs/site/skill.md + install.md are verify-first (update only stale claims), and any docs/site edit is checked against the live shll standards | Conversation says "verify"; Constitution § Toolkit Standards binds docs/site changes; standards re-fetch is a standing lesson | S:70 R:85 A:85 D:80 |

9 assumptions (7 certain, 1 confident, 1 tentative, 0 unresolved).
