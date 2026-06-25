---
type: memory
description: "`/fab-setup` skill — structural bootstrap (sync-first order since szxd: doctor → config → constitution → `fab sync`), subcommand architecture (config, constitution, migrations — version handling delegated to a single `fab migrations-status --json` run), delegation pattern with `fab-kit sync`; stage_directives editor removed, semver one-liner restored at the three-way branch, fab_version fresh-create fallback, Next Steps re-aligned to the State Table (c5tr); scaffold fragment merge fails loudly (jznd) and its `.gitignore` dedup is gitignore-aware — variant coverage + negation hard-stop, `.envrc` stays literal (mqiq)"
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

### Subcommands

`/fab-setup` accepts three subcommands: `config [section]`, `constitution`, and `migrations [file]`. These provide ongoing management of initialization artifacts and version migrations without requiring separate commands. Validation is built into the `config` and `constitution` flows rather than exposed as a standalone subcommand.

### Migrations Version Handling Delegated to the Binary (szxd)

`/fab-setup migrations` no longer reads, parses, or compares the version files itself. The former triplicated version handling — pre-flight existence checks on `fab/.kit-migration-version` and the engine `VERSION`, a "Compare Versions" step, and a standalone Semver Comparison section — is deleted from `fab-setup.md`, along with the corresponding Context Loading item. The skill runs **`fab migrations-status --json` exactly once** (Step 1) and branches on its returned `local`/`engine` fields to pick the equal / local-ahead / no-op output; the binary owns version read/parse/compare as well as discovery (scan/validate/sort — see [migrations.md](/distribution/migrations.md)), and exits non-zero with remediation hints on a missing version file, whose stderr the skill surfaces before stopping. Behavior is unchanged — only the duplicated hand-rolled checks are gone. One casualty of that dedup was restored in c5tr: the Step 1.3 local/engine three-way branch again carries the **one-line semver-comparison rule** the branch needs (compare MAJOR, then MINOR, then PATCH as integers — `2.10.0` > `2.9.7`; never compare lexicographically) — a single parenthetical, not a resurrected Semver Comparison section.

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

- Step 1a: If `config.yaml` is missing, is a raw template (contains `{PROJECT_NAME}`), OR is missing the required fields `project.name`/`project.description` → invokes `/fab-setup config` in create mode. The required-fields clause is load-bearing for the canonical install path: `fab init` writes a `fab_version`-only `config.yaml` before sync's copy-if-absent runs, so an existence-only trigger never fired there and project configuration was silently skipped. The Config Pre-flight create-mode definition uses the same three-part condition
- Step 1b: If `constitution.md` doesn't exist or is a raw template (contains `{Project Name}`) → invokes `/fab-setup constitution` in create mode

**Config Create Mode preserves `fab_version`**: whether reached from bootstrap step 1a or invoked directly, create mode carries an existing `fab_version` key into the newly written `config.yaml` unchanged — the scaffold template lacks the key and the fab router errors without it (config.go). **Fresh-create fallback (c5tr)**: when no prior `config.yaml` exists at all (so there is no key to carry), create mode stamps the engine version into the new file — `fab_version: "$(cat "$(fab kit-path)/VERSION")"` — closing the one path on which the "step 1a guarantees `fab_version`" claim (which step 1c's sync relies on) was false.

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
| Hook registration | `fab-kit sync` (step 4) | Registers `{cache}/kit/hooks/on-*.sh` into `.claude/settings.local.json` hooks; supports tool-name matchers for PostToolUse events; idempotent merge (hooklib replicated in fab-kit) |
| `config.yaml` | `/fab-setup config` (delegated by `/fab-setup`) | Reads `scaffold/config.yaml` template, substitutes placeholders with user-provided values |
| `constitution.md` | `/fab-setup constitution` (delegated by `/fab-setup`) | Reads `scaffold/constitution.md` skeleton, generates principles from project context |

As of szxd, `/fab-setup` invokes `fab sync` as bootstrap step **1c — immediately after the interactive config (1a) and constitution (1b) steps** (sync requires `config.yaml`'s `fab_version`, which 1a guarantees), with a **sync-failure guard**: non-zero exit → STOP and surface sync's output, do not continue the bootstrap. The former hand-scaffolding steps are deleted — old 1c–1g (context.md / code-quality.md / code-review.md skeletons + both doc indexes), old 1i (`fab/changes/` + archive + `.gitkeep`), and old 1k (the `.gitignore` append) — because sync's `scaffoldTreeWalk` copy-if-absent installs, `scaffoldDirectories`, and the `.gitignore` fragment line-ensure merge (`.fab-*`, which subsumes `.fab-status.yaml`) already own all of them; the migration-version note (old 1h) is renumbered to 1d and its "step 1j" references repointed. Bootstrap order: doctor → 1a config → 1b constitution → 1c `fab sync` → 1d version note; the Bootstrap Output section was rewritten to surface sync's report. The resulting file tree is identical to the old sync-last order via idempotency — this reorder was the one explicit behavior-ORDER change in the szxd batch (f077).

**Scaffold writes fail loudly (jznd).** As of 260615-jznd the line-ensuring merge (`lineEnsureMerge` in `src/go/fab-kit/internal/scaffold.go`, behind the `.envrc`/`.gitignore` fragment rows above) **propagates its `os.WriteFile` errors** up the `scaffoldTreeWalk` chain instead of discarding them — a failed fragment write (disk full, read-only mount, permissions) now surfaces as a non-zero sync rather than a silent half-scaffold that looks successful. The `scaffoldDirectories` doc comment that falsely claimed "Write failures are propagated" for a sibling that swallowed them was corrected to match the now-true behavior. No observable behavior change on the success path; the difference is honest failure surfacing.

**`.gitignore` dedup is gitignore-aware (mqiq).** As of 260625-mqiq the "already present?" check in `lineEnsureMerge` is no longer literal string equality for a *directory-style* fragment entry merged into a `.gitignore` destination. The gitignore-aware path is **double-gated**: the destination basename must be `.gitignore` **and** the fragment entry must be a directory token (`gitignoreIsDirectoryToken` — anchored with a leading `/`, or in trailing-slash directory form, and carrying no `*` glob). Two helpers then add gitignore semantics: (1) **variant coverage** — a directory-style entry like `/.claude` counts as already present when any existing line normalizes to the same directory token, across the set `{/.claude, /.claude/, /.claude/*, .claude, .claude/, .claude/*}` (leading slash optional, trailing `/` or `/*` stripped); a *deeper* path such as `/.claude/commands/` does **not** reduce to the token and so does not count as covering; and (2) a **negation hard-stop** — if any `!.../.claude/...` line is present, the broader ignore is never appended (regardless of a preceding `/.claude/*` exclusion), so a user's re-inclusion block survives every sync. Everything else keeps strict literal equality: non-`.gitignore` destinations (notably `.envrc`, Guardrail A), **and** the non-directory patterns shipped in the same fragment (`.fab-*`, `.status.yaml.lock`). The non-directory scoping (Guardrail C) is what stops an anchored `/.status.yaml.lock` (root-only) from being mistaken as covering the unanchored, at-any-depth fragment `.status.yaml.lock`, and stops a `!/.status.yaml.lock` negation from hard-stopping it — either would suppress the broader ignore and let nested `fab/changes/**/.status.yaml.lock` files be committed. The shipped fragment default (`fragment-.gitignore`'s `/.claude`) is unchanged; the fix is the dedup recognizing equivalent existing forms for directory tokens, not changing what is emitted into a fresh file.

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
