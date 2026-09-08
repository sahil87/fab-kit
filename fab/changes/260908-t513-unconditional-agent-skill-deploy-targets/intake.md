# Intake: Unconditional Agent Skill Deploy Targets

**Change**: 260908-t513-unconditional-agent-skill-deploy-targets
**Created**: 2026-09-08

## Origin

Dispatched by `/fab-proceed` (promptless-defer mode) from a synthesized design conversation — decisions below are final unless marked open.

> Make the two directory-format skill deploy targets in `fab sync` unconditional — `.claude/skills/` (Claude Code) and `.agents/skills/` (cross-client Agent Skills convention) deploy on every sync, irrespective of which agent CLIs are on PATH. The `.opencode/commands/` flat target stays detection-gated on `opencode` being on PATH.

Change type: **feat** (explicit — automatic inference has repeatedly mis-typed similar changes as fix/chore; this intake mandates feat).

## Why

1. **Determinism**: today every target is gated by CLI-on-PATH detection (`agentAvailable`, `exec.LookPath`). The deployed tree therefore varies by machine — a teammate without `codex`/`agy`/`kimi` on PATH silently gets no `.agents/skills/`. Unconditional deploy makes `fab sync` produce the same tree everywhere, aligning with Constitution III (Idempotent Operations).
2. **Convention**: `.agents/skills/` is the widely-adopted cross-client convention from the Agent Skills open standard (the agentskills.io integration guide recommends every client scan it; Codex, Copilot, OpenCode, Cursor, and Gemini CLI all read it). Always deploying it makes fab projects work out of the box for those agents.
3. **Rejected alternative — `.agents/`-only switch**: deploying ONLY to `.agents/skills/` and retiring `.claude/skills/` was considered and REJECTED after live verification: Claude Code 2.1.263 does not scan `.agents/skills/` at all (verified empirically via a headless probe in a test worktree with skills only in `.agents/skills/` — zero fab skills visible — and confirmed against official Claude Code docs/changelog, which document only `.claude/skills/`, `~/.claude/skills/`, and plugin skills, with no enabling flag). An `.agents`-only deploy would break the primary harness. A follow-up idea is to retire the `.claude` target if/when Claude Code ships `.agents` support.
4. **Accepted tradeoffs**: clients that scan multiple locations (OpenCode scans `.opencode/skills/`, `.claude/skills/`, and `.agents/skills/`) will see byte-identical duplicate copies; same-name shadowing is harmless. Also accepted: two dotfolders appear in every fab project even when only one agent is used.

If we don't fix it: per-machine tree drift keeps producing "works on my machine" skill availability — teammates and CI without the gating CLIs on PATH silently miss deploy targets, and every new `.agents/skills`-reading client stays invisible to fab projects until its CLI name is added to the candidate list.

## What Changes

### 1. Always-on deploy targets (`src/go/fab-kit/internal/skills.go`)

The `agents` target table (`skills.go:52-56`, `agentConfig` struct at `:28-34` with `Label`, `CLIs`, `BaseDir`, `Format`, `Mode`):

```go
agents := []agentConfig{
    {Label: "Claude Code", CLIs: []string{"claude"}, BaseDir: ..., Format: "directory", Mode: "copy"},
    {Label: "OpenCode", CLIs: []string{"opencode"}, BaseDir: ..., Format: "flat", Mode: "copy"},
    {Label: "Agents dir", CLIs: []string{"codex", "agy", "kimi"}, BaseDir: ..., Format: "directory", Mode: "copy"},
}
```

Make the "Claude Code" (`.claude/skills`, directory, copy) and "Agents dir" (`.agents/skills`, directory, copy) rows **always-on**; the "OpenCode" (`.opencode/commands`, flat) row stays detection-gated on `opencode`. Mechanism is the implementer's choice — either an `AlwaysOn bool` field checked at the detection gate (`skills.go:60-64`), or empty `CLIs` treated as always-available in `agentAvailable` (`skills.go:116`) — prefer the smaller, clearer diff. The table's doc comment (`:47-51`, one-target-per-skill-set invariant) needs its "deploys once when any of them is installed" claim updated.

### 2. Skip/warning output

- The `Skipping {Label}: {missingCLIs}` message (`skills.go:62`, `missingCLIs` at `:142`) never fires for always-on rows; it remains for the gated OpenCode row.
- The `Warning: No agent CLIs found in PATH. Skills were not deployed to any agent.` branch (`skills.go:85-87`) becomes unreachable since `agentsFound` is now always ≥ 2 — remove it, or reword it to cover only gated targets.

### 3. `FAB_AGENTS` env override seam (open decision — see Open Questions)

`agentAvailable` (`skills.go:116-134`) honors `FAB_AGENTS` (space-separated CLI names) as the tests/CI seam replacing PATH detection. Open: should `FAB_AGENTS` be able to suppress always-on targets (keeps a test-isolation escape), or govern only gated rows (purest "unconditional")? **Recommendation from the discussion**: always-on rows bypass `agentAvailable` entirely and `FAB_AGENTS` governs only gated rows; tests assert the new unconditional tree instead of isolating targets. Note `src/kit/skills/_cli-fab.md:438`'s claim that `FAB_AGENTS` "is unrelated and has no config meaning" (config-cascade context) stays true either way.

### 4. Not changing

The per-target `.gitignore` manifest machinery (`skills.go:275-357` — read-manifest → deploy → scoped prune → write-manifest) is target-generic and needs NO changes. New target dirs self-bootstrap via the per-target manifest on the next `fab sync`, so **no migration file** is expected (nothing existing is restructured — consistent with context.md § Migrations, which triggers only on restructuring existing user data).

### 5. Tests (Constitution: CLI/Go changes ship tests)

`src/go/fab-kit/internal/` package, run scoped first:

- `sync_integration_test.go:201` `TestSync_FullRunProducesExpectedTree` — currently runs with `FAB_AGENTS=claude` and asserts `.agents/skills/.gitignore` and `.opencode/commands/.gitignore` do NOT exist; must now expect `.claude/skills/` AND `.agents/skills/` present, `.opencode/commands/` still absent.
- `skills_test.go:23` `TestAgentAvailable_FABAgentsOverride`, `:41` `TestAgentAvailable_AnyCandidate`, `:62` `TestMissingCLIs`, `:83` `TestDeploySkills_GenericDirForNonCodexCLI` (keep its assertion that `.gemini`/`.agy`/`.kimi`/`.codex` dirs are never created), `:459` `TestDeploySkills_PropagatesAgentFailure` — update for the always-on split.
- Other sync integration tests to re-verify: `:290` `TestSync_SecondRunIsContentIdenticalNoop`, `:356` `TestSync_ManifestLifecycle`, `:417` `TestSync_ProjectOnlyRunsOnlyProjectScripts`, `:437` `TestSync_ShimOnlySkipsProjectScripts`.

### 6. Docs sweep (sibling-sweep class — grep old claims repo-wide before finishing apply)

Current claims documenting three *conditional* targets + detection:

- `docs/memory/distribution/kit-architecture.md` § Agent Skill Deployment (~:104-130 — ":105 Deployment is conditional… A target whose candidates are all absent is skipped… When no target fires at all, a warning is printed… FAB_AGENTS… override PATH detection") and § Design Decisions (~:380-394 — ":380-381 Deployment is conditional on any of a target's candidate CLIs… Conditional deployment avoids creating dot folders for agents the developer doesn't use… The multi-candidate gate is the mechanism").
- `docs/memory/distribution/distribution.md` (~:194 deployed-copies list, :213 "copies refreshed on every target").
- `docs/specs/architecture.md` (~:460-470 manifest note at :467; :489 "deployed to each detected agent… Deployment is conditional — each agent's CLI is checked via PATH lookup"; target table ~:490-497 incl. :497 "deploys once when any of them is on PATH"; :539 "deploys skills to detected agents").
- `docs/site/skill.md` (~:111 "`.claude/skills/` (and `.agents/`, `.opencode/`) are deployed copies"), `docs/site/install.md` (~:118-123 deployed-copies note) — verify wording still holds; likely additive.
- `src/kit/skills/_cli-agents.md` (:209 agy, :231 kimi — "which is why `fab sync` deploys one skill set to `.agents/skills/`" — rationale mentions native reading; verify phrasing survives unconditional deploy).
- Sweep phrases: "detected", "on PATH", "three targets", "Skipping <agent>", "conditional", and user-facing string literals about detection (including `*_test.go` comments per the recurring-lessons sweep rule).
- `src/kit/skills/_cli-fab.md` — checked at intake: its `sync` row (§ Workspace Command Exit Semantics) documents manifest/exit behavior, not per-target detection; update only if apply changes output it documents.

### 7. Constitution / context wording

Constitution Principle V and context.md mention `.claude/skills/` deployment. Verify whether wording changes are needed — likely an additive mention of `.agents/skills/`, wording-only, **no version bump** (jjg0/udwv precedent: no MUST-rule change).

## Affected Memory

- `distribution/kit-architecture`: (modify) § Agent Skill Deployment + § Design Decisions — three conditional targets become two always-on + one gated; skip/warning output and `FAB_AGENTS` scope updated
- `distribution/distribution`: (modify) deployed-target enumerations (~:194, :213) reworded for the unconditional pair

## Impact

- **Code**: `src/go/fab-kit/internal/skills.go` (target table, detection gate, warning branch; ~small diff). No changes to manifest machinery, scaffolding, or the router.
- **Tests**: `src/go/fab-kit/internal/skills_test.go`, `sync_integration_test.go` (assertions flip to the unconditional tree). Scoped run: fab-kit internal package first.
- **Docs**: 2 memory files, `docs/specs/architecture.md`, `docs/site/{skill,install}.md` (verify), `src/kit/skills/_cli-agents.md` (verify), constitution/context wording (verify, no bump).
- **Behavior**: every `fab sync` in every fab project now creates `.claude/skills/` and `.agents/skills/` (with per-target `.gitignore` manifests) regardless of installed CLIs; `.opencode/commands/` unchanged. No migration.

## Open Questions

- Should `FAB_AGENTS` be able to suppress the always-on targets (test-isolation escape), or govern only the gated `.opencode` row (recommended: always-on rows bypass `agentAvailable` entirely; tests assert the unconditional tree)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `.claude/skills/` + `.agents/skills/` deploy unconditionally; `.opencode/commands/` stays gated on `opencode` | Discussed — final decision; determinism (Constitution III) + Agent Skills convention | S:95 R:70 A:90 D:95 |
| 2 | Certain | `change_type` is **feat**, set explicitly | Discussed — mandated; inference has repeatedly mis-typed similar changes as fix/chore | S:100 R:90 A:95 D:100 |
| 3 | Certain | `.agents/`-only switch (retiring `.claude/skills/`) rejected | Discussed — live-verified: Claude Code 2.1.263 does not scan `.agents/skills/`; retiring `.claude` would break the primary harness. Follow-up idea: retire when Claude Code ships `.agents` support | S:95 R:60 A:90 D:90 |
| 4 | Certain | Byte-identical duplicate copies for multi-location scanners + two dotfolders in every project are accepted tradeoffs | Discussed — same-name shadowing is harmless; explicitly accepted | S:90 R:85 A:85 D:90 |
| 5 | Confident | Always-on mechanism is implementer's choice (`AlwaysOn bool` field vs empty-`CLIs`-means-always) — prefer the smaller, clearer diff | Discussed — both named as acceptable; trivially reversible, table-local | S:75 R:85 A:80 D:60 |
| 6 | Confident | No migration file — new target dirs self-bootstrap via the per-target manifest on next sync | Nothing existing is restructured; context.md § Migrations triggers only on restructuring existing user data | S:70 R:75 A:80 D:75 |
| 7 | Confident | "No agent CLIs found in PATH" warning (`skills.go:85-87`) is removed or reworded to cover only gated targets | Becomes unreachable (`agentsFound` always ≥ 2); either resolution acceptable per discussion | S:65 R:90 A:80 D:55 |
| 8 | Confident | Constitution V / context.md get at most additive wording (`.agents/skills/` mention), no version bump | jjg0/udwv precedent: wording-only amendments bump nothing; verify during apply | S:60 R:80 A:75 D:70 |
| 9 | Unresolved | `FAB_AGENTS` scope for always-on rows — bypass entirely (recommended) vs suppression escape | Deferred — promptless dispatch | S:35 R:40 A:45 D:35 |

9 assumptions (4 certain, 4 confident, 0 tentative, 1 unresolved).
