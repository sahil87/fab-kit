# Intake: `fab config reference` — generated reference config.yaml command

**Change**: 260702-6nke-config-reference-command
**Created**: 2026-07-02

## Origin

Drafted via `/fab-draft` out of a `/fab-discuss` session; will be handed off to another agent for execution.

> Ideally, we should be maintaining a reference config.yaml either through help files, or on the website, or via a command, so users know what the available options are. Most tools do this.

Conversation arc (key decisions, in order):

1. User initially asked for a commented-out `agent.tiers` section in a project's `config.yaml` so users discover the override surface.
2. Discussion surfaced the weaknesses of the scaffold-comment approach: it only reaches new projects, creates a third drift surface for the default tier values, and comments in live configs are fragile (fab-kit's own `fab/project/config.yaml` has been machine-rewritten — alphabetized, all original scaffold comments stripped).
3. User pivoted to the tool-standard pattern (golangci-lint reference config, `--print-config` style): a canonical reference maintained by the tool itself. **Chosen surface: a command — `fab config reference`** — because the binary owns the defaults, so the binary should emit the reference.
4. User explicitly approved this shape: generated from real Go constants (not hand-written), coverage guarded by test, scaffold shrinks to essentials + pointer line, docs point at the command instead of embedding copies.

## Why

**Problem — config options are undiscoverable.** The full config.yaml schema has no single reference surface. The scaffold (`src/kit/scaffold/fab/project/config.yaml`) shows only a subset: `stage_hooks`, `branch_prefix`, `agent.tiers`, and `project.linear_workspace` appear nowhere in it. Documentation is scattered across `docs/specs/architecture.md`, `docs/specs/stage-models.md`, and migration files. A user who wants to know "what can I put in config.yaml" has no command or file to consult.

**The prior art proves the comment approach fails.** Gap analysis found that migration `2.2.0-to-2.3.0` already shipped exactly what the user first asked for: a fully-commented `agent.tiers` reference block appended to existing configs (sentinel-guarded). Its observed weaknesses motivate this change:

- **Scope**: it documents `agent.tiers` only, not the schema.
- **Drift**: its default values (`claude-opus-4-8` etc.) are hand-written prose; when the tier defaults bump (the "Fable upgrade path" in `stage-models.md`), the block goes stale — and its idempotency sentinel *prevents* re-application, so it can never be refreshed.
- **Fragility**: fab-kit's own config.yaml contains no such block today (comments in live configs get lost to machine rewrites, or migrations get skipped).
- **New projects never see it**: the scaffold was never given the block, so post-2.3.0 initialized projects have no tiers documentation at all.

**Consequence of not fixing**: every new config key (e.g., a future per-tier `spawn_command`) widens the discoverability gap; users typo keys that yaml.v3 silently ignores; the `agent.tiers` override surface — deliberately designed for user override — stays effectively hidden.

**Why this approach**: generation from the real constants is strictly stronger than any drift-guard test on hand-written copies — values *cannot* drift because there is no second copy. A command is version-locked to the binary, works offline, and gives docs/website a stable pointer target.

## What Changes

### 1. New command: `fab config reference`

A new pure-query cobra command in the `fab` binary (`src/go/fab/cmd/fab/`): command group `config` with subcommand `reference` (group naming deliberately leaves room for a future `fab config validate`). Prints a fully-commented reference config.yaml to stdout. Exit 0; byte-stable output for a given binary version (same convention as `fab resolve` queries). No flags in v1.

**Generated, not hand-written.** The reference text lives in Go as a template (new small internal package, e.g. `internal/configref`, or colocated per the `resolve_agent.go` pattern — apply decides placement). Real values are injected from their canonical constants:

- `spawn.DefaultSpawnCommand` (`internal/spawn`)
- default tier profiles via `agent.DefaultTier` / `agent.TierNames` (`internal/agent` — accessors already exported for the existing drift-guard test)
- pipeline stage names for the `stage_hooks` section via `agent.StageNames`

**Full schema coverage — both key sets.** Critically, the Go `Config` struct (`internal/config/config.go`) models only *binary-consumed* keys. Several keys are *skill-consumed* (read by markdown skills, invisible to Go reflection). The reference MUST cover both:

| Key | Consumer |
|-----|----------|
| `project.name`, `project.description` | skills (orientation, PR bodies) |
| `project.linear_workspace` | binary (`Config.Project`) |
| `source_paths` | skills (apply context scoping) |
| `test_paths` | binary (`Config.TestPaths`) |
| `true_impact_exclude` | binary (`Config.TrueImpactExclude`) |
| `checklist.extra_categories` | skills (plan `## Acceptance` generation) |
| `review_tools.claude/codex/copilot` | skills (PR review) |
| `agent.spawn_command` | binary (`Config.Agent.SpawnCommand`) |
| `agent.tiers.{thinking,doing,fast}.{model,effort}` | binary (`Config.Agent.Tiers`) |
| `stage_hooks.<stage>.{pre,post}` | binary (`Config.StageHooks`) |
| `branch_prefix` | binary (`Config.BranchPrefix`) |
| `fab_version` | binary (`Config.FabVersion`) — document as machine-managed, do not hand-edit |

**Layout convention**: baseline keys every project sets (`project`, `source_paths`, `test_paths`, `true_impact_exclude`, `checklist`, `review_tools`, `agent.spawn_command`, `fab_version`) appear live with example/default values; opt-in override blocks (`agent.tiers`, `stage_hooks`, `branch_prefix`) appear commented-out with fab-kit's defaults shown in comments — mirroring the 2.3.0 block's style so uncommenting is opting in. The `agent.tiers` section carries the fixed stage→tier mapping, the built-in default profiles (template-injected), and the override shape, equivalent in content to the 2.3.0 migration block but generated.

Representative excerpt of intended output (values shown here are illustrative — the real ones are template-injected):

```yaml
agent:
  # Base command for spawning agent sessions in scripts and operators.
  spawn_command: 'claude --dangerously-skip-permissions --effort xhigh -n "$(basename "$(pwd)")"'

  # agent.tiers — per-stage model override (optional). fab-kit owns the FIXED
  # stage→tier mapping; you override only what each tier MEANS (model + effort).
  #   thinking: intake, review            default: { model: <injected>, effort: <injected> }
  #   doing:    apply, review-pr, hydrate  default: { model: <injected>, effort: <injected> }
  #   fast:     ship                       default: { model: <injected>, effort: <injected> }
  # tiers:
  #   doing: { model: claude-sonnet-4-6, effort: medium }
```

### 2. Tests (coverage + validity)

Three test contracts alongside the new code:

1. **Validity round-trip**: the emitted output parses via `config.LoadPath`/`yaml.Unmarshal` into `Config` without error.
2. **Binary-key coverage by reflection**: walk the `Config` struct's yaml tags (recursively, including nested structs and map value types) and assert every key path appears in the emitted reference (commented or live). A new binary-consumed config key then *forces* a reference update at test time.
3. **Skill-key coverage by scaffold superset**: parse the scaffold `src/kit/scaffold/fab/project/config.yaml` keys and assert the reference's key set is a superset — this guards the skill-consumed keys that reflection cannot see.

Injected default values need no drift test — they cannot drift by construction (no second copy).

### 3. Scaffold pointer line

`src/kit/scaffold/fab/project/config.yaml` gains one header comment line (top of file):

```yaml
# Full reference of all available options: fab config reference
```

The scaffold otherwise stays minimal — no big commented blocks added.

### 4. Migration (next version slot)

A new sentinel-guarded, config-only migration in `src/kit/migrations/` appending the same one-line pointer comment to existing projects' config.yaml, following the established 2.2.0-to-2.3.0 / 2.7.1-to-2.8.0 precedent (pre-check: skip if config.yaml absent or the pointer sentinel already present; no `.status.yaml` schema change; VERSION bump per the migration system's dual-version model).

### 5. Documentation (point, don't copy)

- `src/kit/skills/_cli-fab.md` — new `fab config reference` command entry (constitution: CLI changes MUST update `_cli-fab.md` + tests).
- `docs/specs/architecture.md` — the config documentation points to `fab config reference` as the canonical full reference.
- `README.md` — configuration section gains a pointer to the command.
- **Mirror sweep obligation** (code-quality.md § Sibling & Mirror Sweeps): any `src/kit/skills/*.md` touched requires its `docs/specs/skills/SPEC-*.md` mirror updated in the same change; on a CLI change, treat all of the touched skill's SPEC mirrors as the sweep class.

### Non-Goals

- **`fab config validate`** (unknown-key/typo linting) — natural follow-up, explicitly deferred; this change only names the command group to leave room.
- **Multi-agent spawn work** — `spawn.WithProfile` template placeholders and per-tier `spawn_command`/CLI dispatch adapter are separate changes (discussed as "change 2" and "change 3"); the reference documents the schema as it exists today.
- **No retro-edit of the 2.2.0-to-2.3.0 migration** — it stays as shipped history; projects that have its block keep it.
- **No config writing** — the command prints to stdout only; it does not create or modify any file.

## Affected Memory

- `_shared/configuration`: (modify) document `fab config reference` as the canonical schema-discovery surface, the binary-vs-skill-consumed key split, and the generated-reference (no-second-copy) drift stance
- `distribution/kit-architecture`: (modify) new `fab config` command group in the fab binary command inventory
- `distribution/migrations`: (modify) new pointer-line migration entry in the migration history

## Impact

- `src/go/fab/cmd/fab/` — new `config` command group + `reference` subcommand (+ tests)
- `src/go/fab/internal/` — new template location (e.g. `configref`) drawing on `internal/spawn` and `internal/agent` exported constants/accessors; no changes to `internal/config` parsing itself
- `src/kit/scaffold/fab/project/config.yaml` — one pointer line
- `src/kit/migrations/` — one new migration file (+ VERSION bump per migration model)
- `src/kit/skills/_cli-fab.md` + its SPEC mirror — command documentation
- `docs/specs/architecture.md`, `README.md` — pointers
- No `.status.yaml` schema change; no behavior change to any existing command

## Open Questions

*(none — all decision points were resolved in the originating discussion or graded as assumptions below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Command is `fab config reference` under a new `config` command group | User chose the name explicitly in discussion; group leaves room for future `config validate` | S:90 R:70 A:95 D:90 |
| 2 | Certain | Reference is generated from a Go template with values injected from real constants (`spawn.DefaultSpawnCommand`, `agent` default tiers) — never hand-written | User explicitly approved "generated, not hand-written"; strictly stronger than drift-guard tests on copies | S:95 R:80 A:95 D:95 |
| 3 | Confident | Coverage tests = validity round-trip + Config-struct yaml-tag reflection + scaffold key-superset | Round-trip + reflection were discussed; the scaffold-superset leg is an extension needed because skill-consumed keys are invisible to reflection | S:60 R:85 A:80 D:70 |
| 4 | Confident | Layout: baseline keys live with example values; opt-in blocks (`agent.tiers`, `stage_hooks`, `branch_prefix`) commented with defaults shown | Mirrors the shipped 2.3.0 block style (uncommenting = opting in); prevents copy-paste pinning of defaults | S:55 R:85 A:75 D:65 |
| 5 | Certain | Scaffold gains a single header pointer line, no commented reference blocks | User's explicit pivot away from scaffold comment blocks | S:85 R:90 A:90 D:85 |
| 6 | Confident | Ship a sentinel-guarded migration adding the pointer line to existing configs | Follows 2.3.0/2.8.0 precedent for surfacing new config surfaces to existing users; alternative (help-only discovery) noted but precedent favors the migration | S:40 R:90 A:70 D:50 |
| 7 | Certain | Docs point at the command; no YAML copies embedded in architecture.md/README | User approved "docs point, don't copy" in the recommendation that led to this draft | S:80 R:95 A:90 D:85 |
| 8 | Confident | Template lives in a new small internal package (e.g. `internal/configref`) or colocated per the `resolve_agent.go` pattern — apply decides exact placement | Implementation detail with clear codebase precedent; easily moved | S:50 R:90 A:85 D:70 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
