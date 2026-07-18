---
type: memory
description: "The /fab-setup skill — structural bootstrap (sync-first order: doctor → config → constitution → fab sync), its subcommand architecture (config, constitution, migrations), delegation to fab-kit sync, and Config Create-Mode (generates config.yaml via fab config init --project with a detected identity seed + test_paths marker detection). Covers the fail-loud scaffold merge and the gitignore-aware .gitignore dedup."
---
# Setup

**Domain**: distribution

## Overview

`/fab-setup` is the structural bootstrap skill that creates the `fab/` directory layout. It also provides subcommands for managing `config.yaml` and `constitution.md` (with built-in validation), and for running version migrations (absorbed from the former `/fab-update`). It delegates structural setup to `fab-kit sync` (which reads kit content from the system cache). It does not handle memory hydration — that responsibility belongs to `/docs-hydrate-memory`.

## Requirements

### Prerequisite Check (Phase 0)

`/fab-setup` (bare bootstrap only) runs `fab doctor` as an early gate before creating any project artifacts. If doctor exits non-zero, setup stops immediately and surfaces the doctor output with fix hints. This gate does not apply to subcommands (`config`, `constitution`, `migrations`).

### Structural Bootstrap Only

`/fab-setup` performs only Phase 1 (structural bootstrap). It does not accept `[sources...]` arguments and contains no source hydration logic.

- Creates `fab/project/config.yaml` (project configuration)
- Creates `fab/project/constitution.md` (project principles)
- Creates `fab/.kit-migration-version` (migration version — via `fab-kit sync`)
- Creates `docs/memory/index.md` (memory index skeleton)
- Creates `docs/specs/index.md` (specifications index skeleton — pre-implementation, human-curated)
- Creates `fab/changes/` directory
- Creates skill deployments via `fab-kit sync`
- Creates `.gitignore` entries
- Safe to re-run (idempotent — skips existing files)

### Config Create-Mode Generates via `fab config init --project` (j0qm)

`/fab-setup config` **create mode** **shells out to `fab config init --project`** with the detected identity seed (`--name`, `--description`, `--source-path`, `--test-path`), which generates the file from the registry: the A-class identity fields live, the managed fence below (see [configuration.md](/_shared/configuration.md) § `fab config init --project`). Notes:

- **fab-init already seeded the file.** On the canonical install path `fab init` has already generated `config.yaml` (with a mechanically-detected name / `src/` / test-marker seed — see [kit-architecture.md](/distribution/kit-architecture.md) § fab-kit `Init`), so create-mode's job is to **refine the seeded live values and ADD the description** (which the Go detection layer does not derive — only `/fab-setup` asks for it), not to substitute placeholders into a blank template.
- **`test_paths` stays a create-mode concern**, reframed as confirm/refine: the skill may add JS/TS test dependencies the Go marker layer skips, and confirms the detected patterns. The marker→ecosystem detection table (below) is unchanged.
- **No `fab_version` step.** `fab_version` lives in `fab/.fab-version` (stamped by `fab init`, j0qm), not `config.yaml`, so create-mode neither preserves nor stamps it.
- **Stub fallback.** When the installed fab-go predates `fab config init --project`, `fab init` writes a minimal embedded stub config.yaml instead (never a printed instruction); create-mode then refines that stub.

The `test_paths` detection derives an **anchored** pattern from a marker→ecosystem table (the same table the `2.7.1-to-2.8.0` migration and the Go `detectTestPaths` use):

| Detected marker | Ecosystem | `test_paths` |
|---|---|---|
| `go.mod` | Go | `**/*_test.go` |
| `pytest.ini` / `pyproject.toml` / `setup.cfg` | Python (pytest) | `**/test_*.py`, `**/*_test.py` |
| `package.json` (jest/vitest), or `*.spec`/`*.test` `.ts`/`.js` present | JS/TS | `**/*.spec.ts`, `**/*.test.ts`, `**/*.spec.js`, `**/*.test.js` |
| `pom.xml` / `build.gradle` | Java/Kotlin | `**/src/test/**` |
| `*.csproj` (test SDK) | .NET | `**/*Tests.cs`, `**/*Test.cs` |
| `Cargo.toml` (Rust) / *(no marker)* | — | leave empty (Rust tests are inline `#[cfg(test)]`, not glob-addressable) |

The derived value is passed as `--test-path` flags to `fab config init --project` (j0qm — was a `{TEST_PATHS}` scaffold placeholder before the scaffold config.yaml was deleted); the registry generator writes it live above the managed fence when non-empty, else the fence advertises it. Config Output surfaces a visible note: `Detected {ecosystem} — set test_paths to {patterns}. Edit fab/project/config.yaml if wrong.` when filled, or `No test convention detected — test_paths left empty (impact breakdown will show a single total). Set it later if desired.` for Rust/unrecognized stacks. Multi-marker repos take the union of pattern sets.

**Why anchored, not a substring**: `test_paths` drives the `/git-pr` impact breakdown's test/impl split (`impl = total − tests`). A bare substring (`**/*test*`) miscounts production code like `attestation.go`/`latest.go` — a confidently-wrong number is worse than the absent (collapsed-to-single-total) breakdown, so unrecognized stacks are left empty. The `2.7.1-to-2.8.0` migration backfills the same detection for existing repos (see [migrations.md](/distribution/migrations.md)).

### Subcommands

`/fab-setup` accepts three subcommands: `config [section]`, `constitution`, and `migrations [file]`. These provide ongoing management of initialization artifacts and version migrations without requiring separate commands. Validation is built into the `config` and `constitution` flows rather than exposed as a standalone subcommand.

### Migrations Version Handling Delegated to the Binary (szxd)

`/fab-setup migrations` does not read, parse, or compare the version files itself. The skill runs **`fab migrations-status --json` exactly once** (Step 1) and branches on its returned `local`/`engine` fields to pick the equal / local-ahead / no-op output; the binary owns version read/parse/compare as well as discovery (scan/validate/sort — see [migrations.md](/distribution/migrations.md)), and exits non-zero with remediation hints on a missing version file, whose stderr the skill surfaces before stopping. The Step 1.3 local/engine three-way branch carries the **one-line semver-comparison rule** the branch needs (compare MAJOR, then MINOR, then PATCH as integers — `2.10.0` > `2.9.7`; never compare lexicographically) — a single parenthetical, not a standalone Semver Comparison section.

### Unrecognized Arguments Rejected

When arguments other than recognized subcommands are passed, setup outputs a redirect message listing the valid subcommands: `config`, `constitution`, `migrations`. No hydration occurs.

### Output

First-run output lists only structural artifacts created. The "With Sources" output section has been removed. `Next:` lines derive from `_preamble.md`'s State Table (re-aligned in c5tr — the old lines had drifted from the table they claimed to derive from): bootstrap / config create / constitution create land in the `initialized` state → `/fab-new <description>`, `/fab-proceed`, or `/docs-hydrate-memory <sources>`; config/constitution updates change no state (no `Next:` action needed); after migrations the line re-derives from the *current* state — `initialized` when no change is active, otherwise the active change's stage row.

### Bootstrap Alternative

As an alternative to manual `cp -r`, new projects can use the one-liner bootstrap:

```
curl -sL https://github.com/{repo}/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

Where `{repo}` is the `repo` value from `kit.conf (removed)`.

After extraction, run `fab-kit sync` then `/fab-setup` as usual.

## Subcommand Architecture

The subcommands manage the lifecycle of Fab's setup artifacts and migrations:

| Subcommand | Purpose |
|---------|---------|
| `/fab-setup constitution` | Create or amend `constitution.md` with semantic versioning (see [configuration](/_shared/configuration.md#amending-constitution)) |
| `/fab-setup config` | Create or update `config.yaml` interactively, preserving comments (see [configuration](/_shared/configuration.md#updating-config)) |
| `/fab-setup migrations [file]` | Run version migrations against the current project (see [migrations](/distribution/migrations.md)) |

`/fab-setup` delegates artifact creation to the subcommands:

- Step 1a: If `config.yaml` is missing, is a raw template (contains `{PROJECT_NAME}`), OR is missing the required fields `project.name`/`project.description` → invokes `/fab-setup config` in create mode. The required-fields clause is load-bearing for the canonical install path: as of j0qm `fab init` **generates** a registry `config.yaml` (identity fields live from a mechanically-detected seed, no description — see § Config Create-Mode) before sync, so an existence-only trigger would skip create mode and the project would never get a description; the missing-`project.description` arm keeps create-mode firing to add it. (Before j0qm the same clause guarded against `fab init` writing a `fab_version`-only config; the file is now a full generated config, but the required-fields trigger is unchanged.) The Config Pre-flight create-mode definition uses the same three-part condition
- Step 1b: If `constitution.md` doesn't exist or is a raw template (contains `{Project Name}`) → invokes `/fab-setup constitution` in create mode

**Config Create Mode no longer handles `fab_version` (j0qm)**: `fab_version` left `config.yaml` for the plain-text sibling `fab/.fab-version` (stamped by `fab init`, out of `config.yaml` entirely — see [configuration.md](/_shared/configuration.md) § `fab_version`). Create mode neither carries nor stamps it — the router reads the version from `fab/.fab-version`, and generation is a `fab config init --project` shell-out that writes only registry fields. This retires the former "Config Create Mode preserves `fab_version`" carry-forward and the c5tr fresh-create `fab_version` stamp fallback: sync's `fab_version` precondition is now satisfied by `fab/.fab-version`, not a config.yaml key.

This ensures each subcommand is the single source of truth for its artifact's generation logic. `/fab-setup` retains ownership of structural orchestration (directories, symlinks, `.gitignore`).

Each subcommand operates independently — they can be invoked directly without going through `/fab-setup`. This supports two workflows:

1. **Initial setup**: `/fab-setup` orchestrates everything (delegates to subcommands internally)
2. **Ongoing management**: User invokes subcommands directly as project evolves

## Delegation Pattern

`/fab-setup` delegates structural setup to `fab-kit sync` (which resolves kit content from the system cache) and adds interactive configuration on top. This means `fab-kit sync` can be run independently (e.g., in CI or after a bootstrap download) without requiring `/fab-setup`.

| Responsibility | Owner | Notes |
|---|---|---|
| Directories (`changes/`, `memory/`, `specs/`) | `fab-kit sync` | Non-interactive, scriptable |
| `fab/.kit-migration-version` | `fab-kit sync` | New project → engine version; existing project (has `config.yaml`) → `0.1.0`; existing file → preserved |
| Skeleton files (`memory/index.md`, `specs/index.md`) | `fab-kit sync` | Copies from `{cache}/kit/scaffold/`; idempotent — skips if file exists |
| Skill deployment (Claude Code, OpenCode, Codex, Gemini) | `fab-kit sync` | Deploys from `{cache}/kit/skills/`; conditional on agent CLI availability |
| `.envrc` entries | `fab-kit sync` | Line-ensuring merge from `{cache}/kit/scaffold/fragment-.envrc` |
| `.gitignore` entries | `fab-kit sync` | Line-ensuring merge from `{cache}/kit/scaffold/fragment-.gitignore` |
| Hook registration | *removed in ioku (2.14.0)* | `fab-kit sync` no longer registers any Claude Code hook or touches `.claude/settings.local.json` — the `fab hook` command family (and its sync step) was removed outright. Agent-state is read from run-kit's `@rk_agent_state` convention; artifact bookkeeping is pull-based via `fab status refresh` (the `artifact-write` PostToolUse registration was removed earlier in y022). Cleanup of any lingering hook entries in an existing project is handled by the `2.13.6-to-2.14.0` migration (for the checkout it runs in) and the `2.15.7-to-2.15.8` migration (which sweeps every worktree, including the main checkout — see [migrations.md](/distribution/migrations.md) § `2.15.7-to-2.15.8`) |
| `config.yaml` | `/fab-setup config` (delegated by `/fab-setup`) | Shells out to `fab config init --project` with the detected identity seed (j0qm — the scaffold `config.yaml` template was deleted; no more placeholder substitution). Refines the fab-init-seeded live values + adds the description; stub fallback if the binary predates the subcommand |
| `constitution.md` | `/fab-setup constitution` (delegated by `/fab-setup`) | Reads `scaffold/constitution.md` skeleton, generates principles from project context |

As of szxd, `/fab-setup` invokes `fab sync` as bootstrap step **1c — immediately after the interactive config (1a) and constitution (1b) steps** (sync requires the project's pinned version — as of j0qm read from `fab/.fab-version`, which `fab init` stamps; on the bare `/fab-setup` path 1a's config-create is what guarantees a usable project state), with a **sync-failure guard**: non-zero exit → STOP and surface sync's output, do not continue the bootstrap. The former hand-scaffolding steps are deleted — old 1c–1g (context.md / code-quality.md / code-review.md skeletons + both doc indexes), old 1i (`fab/changes/` + archive + `.gitkeep`), and old 1k (the `.gitignore` append) — because sync's `scaffoldTreeWalk` copy-if-absent installs, `scaffoldDirectories`, and the `.gitignore` fragment line-ensure merge (`.fab-*`, which subsumes `.fab-status.yaml`) already own all of them; the migration-version note (old 1h) is renumbered to 1d and its "step 1j" references repointed. Bootstrap order: doctor → 1a config → 1b constitution → 1c `fab sync` → 1d version note; the Bootstrap Output section was rewritten to surface sync's report. The resulting file tree is identical to the old sync-last order via idempotency — this reorder was the one explicit behavior-ORDER change in the szxd batch (f077).

**Scaffold writes fail loudly (jznd).** As of 260615-jznd the line-ensuring merge (`lineEnsureMerge` in `src/go/fab-kit/internal/scaffold.go`, behind the `.envrc`/`.gitignore` fragment rows above) **propagates its `os.WriteFile` errors** up the `scaffoldTreeWalk` chain instead of discarding them — a failed fragment write (disk full, read-only mount, permissions) now surfaces as a non-zero sync rather than a silent half-scaffold that looks successful. The `scaffoldDirectories` doc comment that falsely claimed "Write failures are propagated" for a sibling that swallowed them was corrected to match the now-true behavior. No observable behavior change on the success path; the difference is honest failure surfacing.

**`.gitignore` dedup is gitignore-aware (mqiq).** As of 260625-mqiq the "already present?" check in `lineEnsureMerge` is no longer literal string equality for a *directory-style* fragment entry merged into a `.gitignore` destination. The gitignore-aware path is **double-gated**: the destination basename must be `.gitignore` **and** the fragment entry must be a directory token (`gitignoreIsDirectoryToken` — anchored with a leading `/`, or in trailing-slash directory form, and carrying no `*` glob). Two helpers then add gitignore semantics: (1) **variant coverage** — a directory-style entry like `/.claude` counts as already present when any existing line normalizes to the same directory token, across the set `{/.claude, /.claude/, /.claude/*, .claude, .claude/, .claude/*}` (leading slash optional, trailing `/` or `/*` stripped); a *deeper* path such as `/.claude/commands/` does **not** reduce to the token and so does not count as covering; and (2) a **negation hard-stop** — if any `!.../.claude/...` line is present, the broader ignore is never appended (regardless of a preceding `/.claude/*` exclusion), so a user's re-inclusion block survives every sync. Everything else keeps strict literal equality: non-`.gitignore` destinations (notably `.envrc`, Guardrail A), **and** the non-directory patterns shipped in the same fragment (`.fab-*`, `.status.yaml.lock`, and — as of 8ken — `!fab/.fab-version`). The non-directory scoping (Guardrail C) is what stops an anchored `/.status.yaml.lock` (root-only) from being mistaken as covering the unanchored, at-any-depth fragment `.status.yaml.lock`, and stops a `!/.status.yaml.lock` negation from hard-stopping it — either would suppress the broader ignore and let nested `fab/changes/**/.status.yaml.lock` files be committed. The shipped fragment default (`fragment-.gitignore`'s `/.claude`) is unchanged; the fix is the dedup recognizing equivalent existing forms for directory tokens, not changing what is emitted into a fresh file.

**Negation lines take the strict-literal path (8ken).** The `!fab/.fab-version` negation the fragment gained in 8ken (un-ignoring the relocated version file — see [kit-architecture.md](/distribution/kit-architecture.md) and [configuration.md](/_shared/configuration.md) § `fab_version`) is a **non-directory token**: `gitignoreIsDirectoryToken(entry)` is false for it (no leading `/`, no trailing `/`) and false for the `.fab-*` line above it (contains `*`), so **both** take the strict-literal-dedup path — the negation is appended once if absent and re-merges idempotently. Because the Guardrail-B negation **hard-stop** (`gitignoreHasNegation`, `scaffold.go`) is itself gated on `gitignoreIsDirectoryToken`, it is **never consulted** for either non-directory line — so adding `!fab/.fab-version` to the fragment cannot suppress the `.fab-*` ensure (the two lines coexist, last-match-wins un-ignoring the file). This is the same class as `.fab-*`/`.status.yaml.lock`, not the directory-token class that Guardrail B guards.

**Bootstrap path** (without `/fab-setup`): After `brew install fab-kit` and `fab init`, running `fab sync` alone creates a complete structural scaffold. `/fab-setup` is only needed to generate `config.yaml` and `constitution.md`.

## Design Decisions

### Init as Pure Structural Bootstrap
**Decision**: `/fab-setup` only creates directory structure and configuration files. Source hydration is delegated to `/docs-hydrate-memory`.
**Why**: Clean separation of concerns — bootstrap runs once per project, hydration runs whenever new sources need ingesting. Using "init" for repeated hydration was confusing.
**Rejected**: Keeping hydration in init with an optional flag — muddled the interface and made init's help text complex.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Redirect Message for Old Interface
**Decision**: When arguments are passed to `/fab-setup`, show a helpful redirect to `/docs-hydrate-memory` instead of silently ignoring.
**Why**: Better UX — users who remember the old interface get guided to the new one.
**Rejected**: Silently ignoring arguments — confusing, user would think hydration happened.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Consolidated Skill with Subcommands
**Decision**: All four commands are subcommands within a single `fab-setup.md` skill file — `config`, `constitution`, `migrations`, and a validate-redirect for backward compatibility. Each subcommand has its own behavior section, sharing the same `model_tier` and frontmatter.
*Introduced by*: 260213-3tyk-merge-fab-init-subcommands

### Config Updates Use String Replacement
**Decision**: `/fab-setup config` uses targeted string replacement rather than full YAML parse-and-rewrite. This preserves the heavily-commented `config.yaml` format at the cost of slightly less structural safety.
*Introduced by*: 260212-h9k3-fab-init-family

### Validate Is Read-Only (deprecated)
**Decision**: `/fab-init validate` only checked and reported — it never modified files. Fix suggestions were provided but the user applied them (directly or via the other subcommands).
**Deprecated**: Validation is now folded into the `config` and `constitution` subcommand flows, removing the need for a standalone validate step.
*Introduced by*: 260212-h9k3-fab-init-family
*Deprecated by*: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow

### Templates in Scaffold Files
**Decision**: `config.yaml` and `constitution.md` templates live as standalone files in `$(fab kit-path)/scaffold/` rather than as inline code blocks in `fab-setup.md`. `/fab-setup` reads from these files and substitutes placeholders. Index templates (`memory-index.md`, `specs-index.md`) are also referenced from scaffold files, eliminating duplicated inline copies.
**Why**: Prevents drift between inline templates and actual schema expectations. Aligns with Constitution V (Portability) — `.kit/` owns its templates as inspectable, diffable files. Single source of truth for both `fab-kit sync` and `/fab-setup`.
**Rejected**: Keeping inline templates — two sources of truth that can diverge when the config schema evolves.
*Introduced by*: 260217-17pe-DEV-1046-scaffold-setup-templates

### Agent-Inferred Conventions Replace Templates (superseded)
**Decision**: Step 1b-lang uses agent inference (Detection → Inference → Write) instead of bundled language templates. The agent reads project marker files (`Cargo.toml`, `tsconfig.json`, `package.json`, `go.mod`, `pyproject.toml`, etc.) and linter/formatter configs, then derives conventions from its training knowledge grounded in actual config values. Conventions are routed to the appropriate `fab/project/*` file by content type (enforcement rules → constitution, stack info → context, coding standards → code-quality, review policy → code-review, source paths → config). The skill describes the *process*, not hard-coded convention content.
**Why**: Bundling language-specific templates in `$(fab kit-path)/templates/constitutions/` and `$(fab kit-path)/templates/configs/` violated Constitution §V (portability — no assumptions about host project's language/toolchain). Templates created maintenance burden and encoded opinions that may not match the project's actual setup.
**Rejected**: Keeping language templates — violates neutrality, creates maintenance burden, makes judgment calls on behalf of users.
*Introduced by*: 260306-143f-setup-language-inference
*Superseded by*: 260306-6bba-redesign-hooks-strategy — language-specific customization rejected entirely; fab-kit stays language-neutral. Step 1b-lang removed from bootstrap flow.

### Sync-First Bootstrap Order; Hand-Scaffolding Steps Deleted (szxd)
**Decision**: In the bare bootstrap, `fab sync` runs as step 1c — immediately after the interactive config (1a) and constitution (1b) steps and before anything else — guarded by a STOP on non-zero exit. The seven steps that hand-duplicated sync's scaffolding (old 1c–1g skeleton copies, old 1i directory creation, old 1k `.gitignore` append) are deleted; sync is the single owner of non-interactive structural setup. Sync cannot move before 1a because it requires `config.yaml`'s `fab_version` (the fab router errors without it).
**Why**: Every scaffold artifact was described twice — once as a skill step, once inside sync — so each scaffold change had to land in both places, and the copies had already drifted in detail. Sync's operations are copy-if-absent / line-ensure merges, so running it earlier produces an identical file tree via idempotency. This was the szxd batch's one explicit behavior-ORDER change (f077), flagged in the PR description.
**Rejected**: Keeping sync last with the hand-scaffolding steps as "idempotent guards" (the duplication is the maintenance cost, not the ordering). Moving sync before the interactive steps (sync hard-requires `fab_version` from 1a). Deleting the steps without a sync-failure guard (a failed sync would previously have been partially papered over by the hand-scaffolding; with single ownership, sync failure must stop the bootstrap).
*Introduced by*: 260611-szxd-skills-twins-self-duplication-refactor

### Absorbed /fab-update into /fab-setup migrations
**Decision**: `/fab-update` functionality is now available as `/fab-setup migrations`. Version migrations live under the same command namespace as the rest of project setup.
**Why**: Reduces the dropped-ball two-step flow where users had to remember a separate `/fab-update` command after upgrading the kit. Makes migrations discoverable from the same command namespace as config and constitution management.
**Rejected**: Keeping `/fab-update` as a separate top-level skill — created a discoverability gap and a two-step flow that was easy to forget.
*Introduced by*: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow

## Deprecated Requirements

### Source Hydration (Phase 2)
**Deprecated by**: 260207-q7m3-separate-hydrate-smart-context (2026-02-07)
**Reason**: Source hydration extracted to dedicated `/docs-hydrate-memory` skill for better separation of concerns.
**Migration**: Use `/fab-hydrate [sources...]` instead of `/fab-setup [sources...]`.

### /fab-init validate Subcommand
**Deprecated by**: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow (2026-02-16)
**Reason**: Validation folded into the `config` and `constitution` subcommand flows. A standalone validate step was redundant — each subcommand now validates its own artifact as part of the create/update workflow.
**Migration**: Use `/fab-setup config` or `/fab-setup constitution` which include built-in validation.

### /fab-update
**Deprecated by**: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow (2026-02-16)
**Reason**: Absorbed into `/fab-setup migrations` to reduce the two-step upgrade flow and make migrations discoverable from the same command namespace.
**Migration**: Use `/fab-setup migrations [file]` instead of `/fab-update`.

### Template-Driven Language Detection (Step 1b-lang)
**Deprecated by**: 260306-143f-setup-language-inference (2026-03-06)
**Reason**: Replaced by agent-inferred conventions. Template files (`$(fab kit-path)/templates/constitutions/`, `$(fab kit-path)/templates/configs/`) deleted. Language template advisory in `src/kit/sync/2-sync-workspace.sh` (invoked by `fab-sync.sh`, section 2b) removed.
**Migration**: Step 1b-lang now uses agent inference — no user action required.

### Agent-Inferred Language Conventions (Step 1b-lang)
**Deprecated by**: 260306-6bba-redesign-hooks-strategy (2026-03-06)
**Reason**: Language-specific customization rejected entirely — fab-kit stays language-neutral per Constitution §V. Detection logic has no purpose without templates or language-specific content to produce. Agent inference (260306-143f) was a stepping stone that this change supersedes. Step 1b-lang removed from `fab-setup.md` bootstrap flow.
**Migration**: Projects that want language-specific conventions can add them manually to `fab/project/*` files.
