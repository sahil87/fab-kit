---
type: memory
description: "The /fab-setup skill — structural bootstrap (doctor → config → constitution → fab sync), subcommands, config create-mode, and delegation to fab-kit sync's resolved kit source. Covers the seven-check doctor gate, fail-loud scaffold merge, gitignore-aware .gitignore dedup, the bare-fab-setup interactive wizard (probe-driven interview, --defaults/--project, surgical config-set writes), and the read-only fab setup check doctor (probe set, exit contract, coexistence with fab doctor)."
---
# Setup

**Domain**: distribution

## Overview

`/fab-setup` is the structural bootstrap skill that creates the `fab/` directory layout. It also provides subcommands for managing `config.yaml` and `constitution.md` (with built-in validation), and for running version migrations. It delegates structural setup to `fab-kit sync`, which reads from `FAB_KIT_PATH` when set and otherwise from the version cache. It does not handle memory hydration — that responsibility belongs to `/docs-hydrate-memory`. The fab-go binary separately ships the `fab setup` command family — bare `fab setup` runs the interactive setup wizard (§ Interactive Setup Wizard) and `fab setup check` is the read-only setup-state doctor (§ Setup-State Doctor) — distinct surfaces from this skill despite the shared `setup` name.

## Requirements

### Prerequisite Check (Phase 0)

`/fab-setup` (bare bootstrap only) runs `fab doctor` as an early gate before creating any project artifacts. If doctor exits non-zero, setup stops immediately and surfaces the doctor output with fix hints. `FAB_KIT_PATH` provenance in normal doctor output is informational and does not alter the seven-check result or gate. This gate does not apply to subcommands (`config`, `constitution`, `migrations`).

### Interactive Setup Wizard (bare `fab setup`)

Bare `fab setup` runs an interactive interview layered over the setup-check probe. All detection comes from one `setupcheck.Run(setupCheckInput())` call — the same call `fab setup check` makes; the wizard adds no probing of its own and never asks a capability question, only preference questions over options pre-filtered to what the probe found. Flow: non-TTY guard → scope banner → 4-question default path → opt-in advanced section → diff-before-write summary → surgical writes.

- **Scope targeting** — the banner names the write target before any question. The default target is the system tier (`~/.fab-kit/config.yaml`, machine-wide preferences); `--project` retargets to `fab/project/config.yaml` and errors outside a fab repo.
- **Default path (4 questions)** — `agent.session` and `agent.workers` offer the probe roster filtered to detected providers (`ProviderProbe.Found()` — undetected providers are dropped, not annotated as missing), each annotated with its declared capabilities (e.g. `claude (interactive, headless, native)`); `dispatch.mode` offers rungs filtered by viability (`pane` only when the probe's tmux signal is present, `native`/`headless` only when a detected provider declares the capability) and states the ladder-ceiling semantics (resolution descends from the setting and never ascends, so an unreachable setting degrades softly); Q4 is the advanced-section opt-in (default no). Every question defaults to the current effective value with its origin shown, and its footer points at `fab config explain <key>` (owner-or-pointer — the wizard never restates the reference prose).
- **Advanced section** — opting in at Q4 asks all four keys (`agent.profiles.operator.provider`, `agent.profiles.review.provider`, `dispatch.column_width`, `dispatch.reap_done`) regardless of each key's winning tier, so first-time overrides are settable through the wizard. The profile keys are never stored by default — the read model derives their built-in-defaults row from the depth knobs — so a key still at that derived default is presented as a depth-correct inherit indication over an empty baseline (`(inherit agent.session)` for operator, `(inherit agent.workers)` for review; origin suppressed): Enter keeps the inherit and writes nothing, while typing a detected provider — including the currently-inherited one — is an explicit pin that writes. The diff summary renders the inherit indication as the old side (`agent.profiles.operator.provider: (inherit agent.session) → codex`).
- **Writes** — a diff summary (one `<key>: <old> → <new>` line per changed answer plus the target tier) and a `[Y/n]` confirmation precede any write. Zero changed answers (the all-Enter run) prints "nothing to change" and exits 0 touching no file — repeated all-Enter runs are byte-identical no-ops (Constitution III). Confirmed writes go through the existing `fab config set` path in-process — `configupgrade.SetSystem` for the system tier, `configupgrade.Set` for the project tier, with the target path from `configMutationPath` — never a whole-file rewrite, never a child-process exec, with `warnIfShadowed` firing per written key so a write shadowed by a higher tier says so.
- **`--defaults` / non-TTY** — `--defaults` runs the full flow non-interactively accepting every default (banner, resolved answers, and the zero-write summary), composable with `--project`. Non-TTY stdin without `--defaults` fails with a usage hint naming the flag (stdlib-only `os.ModeCharDevice` detection, the `batch_archive.go` pattern); EOF on stdin falls back to a question's default, so the interview can never hang on an exhausted reader.

Prompts are bare stdin line reads against the command's `InOrStdin()` — no TUI dependency (Constitution I). The interview loop lives in `src/go/fab/cmd/fab/setup_wizard.go` in package main (sibling of `setup.go`), reusing that package's unexported seams — the cmd-owns-wiring / internal-owns-probing split is preserved: the wizard is wiring/rendering over `internal/setupcheck`'s probe. `fab setup check` is a sibling behavior the wizard does not touch; its no-writes invariant holds (§ Setup-State Doctor). The `/fab-setup` skill's config flow delegates the wizard-covered preference keys here (§ Delegation Pattern).

### Setup-State Doctor (`fab setup check`)

The fab-go binary ships a read-only environment doctor as the `fab setup check` subcommand. Bare `fab setup` runs the interactive setup wizard (§ Interactive Setup Wizard); `fab setup bogus` is a usage error (exit 2). The doctor's hard invariant is **no writes**: no config mutation, no trust-store seeding, no agent/pane launches, no `.fab-*` state files, no prompts — repeated runs are byte-identical (Constitution III). All probing lives in the reusable `internal/setupcheck` package, which returns structured results (`Finding`/`ProviderProbe`/`Report`) with its environment seams injected (lookPath, `$TMUX`, kit-cache dir, config layers); the cobra command owns only input wiring, rendering, and exit-code mapping, so the wizard consumes the same `Report` to filter its interview options without shelling out. Source: `src/go/fab/cmd/fab/setup.go`, `src/go/fab/cmd/fab/setup_wizard.go`, `src/go/fab/internal/setupcheck/`.

Probe set:

- **Provider roster** — every resolvable provider (the built-ins from the embedded `defaults.yaml` ∪ user-defined providers merged by the config cascade) with declared capabilities (`interactive_command`/`headless_command`/`native` — presence grammar, "here is how", never "select this mode") and binary presence on PATH. The leading executable token is resolved through nested `sh -c '...'` wrappers, so agy/kimi resolve to `agy`/`kimi`, never `sh`. A provider a role resolves to (depth knob or `agent.profiles.<role>.provider`) whose executables are missing is **failure-severity**; an unconfigured provider's absence is informational only.
- **Environment facts** — `$TMUX` presence (pane-rung viability, classified with `internal/dispatch`'s tmux signal) and PATH presence of `gh`/`yq` (warning when absent — pipeline tooling) and `rk` (informational — fail-silent optional tooling, never a problem).
- **Version triplet and skew** — the running binary's version, the kit cache's `VERSION`, and the project pin `fab/.fab-version`, with mismatches reported as warnings; plus the override-masking bottle-skew check (see [distribution.md](/distribution/distribution.md) § Version Skew Detection).
- **Config sanity** — an unviable `dispatch.mode` is reported with the exact descent-reason strings `internal/dispatch.SelectMode` produces (`pane unavailable: no tmux`, `pane unavailable: no interactive_command`, `native unavailable`): a working descent is warning-severity, no reachable rung is failure-severity. An unreadable project config is failure-severity, with the remaining probes still running against the empty config. Outside a fab repo the doctor degrades to the system+env tiers instead of erroring.

Exit contract: **0** healthy or warnings-only, **1** when any failure-severity finding exists (returned as an operational error through the existing `run()` classification, with a stderr summary line), **2** usage errors via the binary-wide `markRunReached` seam. Informational findings never affect the exit code.

**Coexistence with `fab doctor`**: the two doctors have distinct jobs; neither subsumes nor invokes the other. fab-kit's seven-check `fab doctor` covers **system prerequisites** ("is this machine good enough to use fab-kit" — git, fab, bash, yq, jq, gh, direnv) and is untouched, including its `/fab-setup` Phase 0 gate above. `fab setup check` covers **setup-state diagnostics** — config viability, provider roster, dispatch-mode viability, version skew. The gh/yq presence overlap is accepted duplication across two binaries with different jobs.

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

### Config Create-Mode Generates via `fab config init --project`

`/fab-setup config` **create mode** (j0qm) **shells out to `fab config init --project`** with the detected identity seed (`--name`, `--description`, `--source-path`, `--test-path`), which generates the file from the registry: the A-class identity fields live, the managed fence below (see [configuration.md](/_shared/configuration.md) § `fab config init --project`). Notes:

- **fab-init already seeded the file.** On the canonical install path `fab init` has already generated `config.yaml` (with a mechanically-detected name / `src/` / test-marker seed — see [kit-architecture.md](/distribution/kit-architecture.md) § fab-kit `Init`), so create-mode's job is to **refine the seeded live values and ADD the description** (which the Go detection layer does not derive — only `/fab-setup` asks for it), not to substitute placeholders into a blank template.
- **`test_paths` stays a create-mode concern**, reframed as confirm/refine: the skill may add JS/TS test dependencies the Go marker layer skips, and confirms the detected patterns. The marker→ecosystem detection table (below) is unchanged.
- **No `fab_version` step.** `fab_version` lives in `fab/.fab-version` (stamped by `fab init`) (j0qm), not `config.yaml`, so create-mode neither preserves nor stamps it.
- **Old fab-go fails closed.** When the installed fab-go predates `fab config init --project`, `fab init` exits non-zero with an error instructing the user to upgrade fab-go — no config.yaml is written, so create-mode has nothing to refine until fab-go is current.

The `test_paths` detection derives an **anchored** pattern from a marker→ecosystem table (the same table the `2.7.1-to-2.8.0` migration and the Go `detectTestPaths` use):

| Detected marker | Ecosystem | `test_paths` |
|---|---|---|
| `go.mod` | Go | `**/*_test.go` |
| `pytest.ini` / `pyproject.toml` / `setup.cfg` | Python (pytest) | `**/test_*.py`, `**/*_test.py` |
| `package.json` (jest/vitest), or `*.spec`/`*.test` `.ts`/`.js` present | JS/TS | `**/*.spec.ts`, `**/*.test.ts`, `**/*.spec.js`, `**/*.test.js` |
| `pom.xml` / `build.gradle` | Java/Kotlin | `**/src/test/**` |
| `*.csproj` (test SDK) | .NET | `**/*Tests.cs`, `**/*Test.cs` |
| `Cargo.toml` (Rust) / *(no marker)* | — | leave empty (Rust tests are inline `#[cfg(test)]`, not glob-addressable) |

The derived value is passed as `--test-path` flags to `fab config init --project` (j0qm); the registry generator writes it live above the managed fence when non-empty, else the fence advertises it. Config Output surfaces a visible note: `Detected {ecosystem} — set test_paths to {patterns}. Edit fab/project/config.yaml if wrong.` when filled, or `No test convention detected — test_paths left empty (impact breakdown will show a single total). Set it later if desired.` for Rust/unrecognized stacks. Multi-marker repos take the union of pattern sets.

**Why anchored, not a substring**: `test_paths` drives the `/git-pr` impact breakdown's test/impl split (`impl = total − tests`). A bare substring (`**/*test*`) miscounts production code like `attestation.go`/`latest.go` — a confidently-wrong number is worse than the absent (collapsed-to-single-total) breakdown, so unrecognized stacks are left empty. The `2.7.1-to-2.8.0` migration backfills the same detection for existing repos (see [migrations.md](/distribution/migrations.md)).

### Subcommands

`/fab-setup` accepts three subcommands: `config [section]`, `constitution`, and `migrations [file]`. These provide ongoing management of initialization artifacts and version migrations without requiring separate commands. Validation is built into the `config` and `constitution` flows rather than exposed as a standalone subcommand.

### Migrations Version Handling Delegated to the Binary

`/fab-setup migrations` (szxd) does not read, parse, or compare the version files itself. The skill runs **`fab migrations-status --json` exactly once** (Step 1) and branches on its returned `local`/`engine` fields to pick the equal / local-ahead / no-op output; the binary owns version read/parse/compare as well as discovery (scan/validate/sort — see [migrations.md](/distribution/migrations.md)), and exits non-zero with remediation hints on a missing version file, whose stderr the skill surfaces before stopping. The Step 1.3 local/engine three-way branch carries the **one-line semver-comparison rule** the branch needs (compare MAJOR, then MINOR, then PATCH as integers — `2.10.0` > `2.9.7`; never compare lexicographically) — a single parenthetical, not a standalone Semver Comparison section.

### Unrecognized Arguments Rejected

When arguments other than recognized subcommands are passed, setup outputs a redirect message listing the valid subcommands: `config`, `constitution`, `migrations`. No hydration occurs.

### Output

First-run output lists only structural artifacts created. `Next:` lines derive from `_preamble.md`'s State Table (c5tr): bootstrap / config create / constitution create land in the `initialized` state → `/fab-new <description>`, `/fab-proceed`, or `/docs-hydrate-memory <sources>`; config/constitution updates change no state (no `Next:` action needed); after migrations the line re-derives from the *current* state — `initialized` when no change is active, otherwise the active change's stage row.

### Bootstrap Alternative

As an alternative to manual `cp -r`, new projects can use the one-liner bootstrap:

```
curl -sL https://github.com/{repo}/releases/latest/download/kit.tar.gz | tar xz -C fab/
```

Where `{repo}` is `sahil87/fab-kit`.

After extraction, run `fab-kit sync` then `/fab-setup` as usual.

## Subcommand Architecture

The subcommands manage the lifecycle of Fab's setup artifacts and migrations:

| Subcommand | Purpose |
|---------|---------|
| `/fab-setup constitution` | Create or amend `constitution.md` with semantic versioning (see [configuration](/_shared/configuration.md#amending-constitution)) |
| `/fab-setup config` | Create or update `config.yaml` interactively, preserving comments (see [configuration](/_shared/configuration.md#updating-config)) |
| `/fab-setup migrations [file]` | Run version migrations against the current project (see [migrations](/distribution/migrations.md)) |

`/fab-setup` delegates artifact creation to the subcommands:

- Step 1a: If `config.yaml` is missing, is a raw template (contains `{PROJECT_NAME}`), OR is missing the required fields `project.name`/`project.description` → invokes `/fab-setup config` in create mode. The required-fields clause is load-bearing for the canonical install path: `fab init` **generates** a registry `config.yaml` (j0qm) (identity fields live from a mechanically-detected seed, no description — see § Config Create-Mode) before sync, so an existence-only trigger would skip create mode and the project would never get a description; the missing-`project.description` arm keeps create-mode firing to add it. The Config Pre-flight create-mode definition uses the same three-part condition
- Step 1b: If `constitution.md` doesn't exist or is a raw template (contains `{Project Name}`) → invokes `/fab-setup constitution` in create mode

**Config Create Mode does not handle `fab_version` (j0qm)**: `fab_version` lives in the plain-text sibling `fab/.fab-version` (stamped by `fab init` — see [configuration.md](/_shared/configuration.md) § `fab_version`). Create mode neither carries nor stamps it — the router reads the version from `fab/.fab-version`, generation is a `fab config init --project` shell-out that writes only registry fields, and sync's `fab_version` precondition is satisfied by `fab/.fab-version`, not a config.yaml key.

This ensures each subcommand is the single source of truth for its artifact's generation logic. `/fab-setup` retains ownership of structural orchestration (directories, symlinks, `.gitignore`).

Each subcommand operates independently — they can be invoked directly without going through `/fab-setup`. This supports two workflows:

1. **Initial setup**: `/fab-setup` orchestrates everything (delegates to subcommands internally)
2. **Ongoing management**: User invokes subcommands directly as project evolves

## Delegation Pattern

`/fab-setup` delegates structural setup to `fab-kit sync` and adds interactive configuration on top. Sync resolves `{kit-dir}` from a validated `FAB_KIT_PATH` when set, otherwise from the local-then-remote version cache. This means `fab-kit sync` can be run independently (e.g., in CI or after a bootstrap download) without requiring `/fab-setup`.

| Responsibility | Owner | Notes |
|---|---|---|
| Directories (`changes/`, `memory/`, `specs/`) | `fab-kit sync` | Non-interactive, scriptable |
| `fab/.kit-migration-version` | `fab-kit sync` | New project → engine version; existing project (has `config.yaml`) → `0.1.0`; existing file → preserved |
| Skeleton files (`memory/index.md`, `specs/index.md`) | `fab-kit sync` | Copies from `{kit-dir}/scaffold/`; idempotent — skips if file exists |
| Skill deployment (Claude Code, OpenCode, and the generic agents dir) | `fab-kit sync` | Deploys from `{kit-dir}/skills/` to three targets; conditional on agent CLI availability — see [kit-architecture.md](/distribution/kit-architecture.md) § Agent Skill Deployment |
| `.envrc` entries | `fab-kit sync` | Line-ensuring merge from `{kit-dir}/scaffold/fragment-.envrc` |
| `.gitignore` entries | `fab-kit sync` | Line-ensuring merge from `{kit-dir}/scaffold/fragment-.gitignore` |
| Hook registration | *(none)* | `fab-kit sync` registers no Claude Code hook and never touches `.claude/settings.local.json` — there is no `fab hook` command family (ioku). Agent-state is read from run-kit's `@rk_pane_agent_state` convention; artifact bookkeeping is pull-based via `fab status refresh` (y022). Cleanup of any lingering hook entries in an existing project is the migrations' job — `2.13.6-to-2.14.0` (the checkout it runs in) and `2.15.7-to-2.15.8` (every worktree, main checkout included — see [migrations.md](/distribution/migrations.md) § `2.15.7-to-2.15.8`) |
| `config.yaml` | `/fab-setup config` (delegated by `/fab-setup`) | Shells out to `fab config init --project` with the detected identity seed (j0qm) — there is no scaffold `config.yaml` template and no placeholder substitution. Refines the fab-init-seeded live values + adds the description; fails closed with an upgrade-fab-go error if the binary predates the subcommand |
| `constitution.md` | `/fab-setup constitution` (delegated by `/fab-setup`) | Reads `scaffold/constitution.md` skeleton, generates principles from project context |

`/fab-setup config` delegates the agent/dispatch preference keys — `agent.session`, `agent.workers`, `dispatch.mode`, and the advanced `agent.profiles.*` / `dispatch.*` knobs — to the bare-`fab setup` wizard (§ Interactive Setup Wizard), which interviews with detected-capability filtering and writes via the surgical `fab config set` path; the skill's own config flow covers the identity/structural fields (name, description, `source_paths`, `test_paths`) — a disjoint set.

`/fab-setup` invokes `fab sync` as bootstrap step **1c — immediately after the interactive config (1a) and constitution (1b) steps** (szxd) (sync requires the project's pinned version, read from `fab/.fab-version`, which `fab init` stamps (j0qm); on the bare `/fab-setup` path 1a's config-create is what guarantees a usable project state), with a **sync-failure guard**: non-zero exit → STOP and surface sync's output, do not continue the bootstrap. The skill hand-scaffolds nothing: sync's `scaffoldTreeWalk` copy-if-absent installs, `scaffoldDirectories`, and the `.gitignore` fragment line-ensure merge (`.fab-*`, which subsumes `.fab-status.yaml`) own all non-interactive structural setup (see the Sync-First DD below). Bootstrap order: doctor → 1a config → 1b constitution → 1c `fab sync` → 1d version note; the Bootstrap Output section surfaces sync's report.

**Scaffold writes fail loudly (jznd).** The line-ensuring merge (`lineEnsureMerge` in `src/go/fab-kit/internal/scaffold.go`, behind the `.envrc`/`.gitignore` fragment rows above) **propagates its `os.WriteFile` errors** up the `scaffoldTreeWalk` chain — a failed fragment write (disk full, read-only mount, permissions) surfaces as a non-zero sync, never a silent half-scaffold that looks successful.

**`.gitignore` dedup is gitignore-aware (mqiq).** The "already present?" check in `lineEnsureMerge` goes beyond literal string equality for a *directory-style* fragment entry merged into a `.gitignore` destination. The gitignore-aware path is **double-gated**: the destination basename must be `.gitignore` **and** the fragment entry must be a directory token (`gitignoreIsDirectoryToken` — anchored with a leading `/`, or in trailing-slash directory form, and carrying no `*` glob). Two helpers then add gitignore semantics: (1) **variant coverage** — a directory-style entry like `/.claude` counts as already present when any existing line normalizes to the same directory token, across the set `{/.claude, /.claude/, /.claude/*, .claude, .claude/, .claude/*}` (leading slash optional, trailing `/` or `/*` stripped); a *deeper* path such as `/.claude/commands/` does **not** reduce to the token and so does not count as covering; and (2) a **negation hard-stop** — if any `!.../.claude/...` line is present, the broader ignore is never appended (regardless of a preceding `/.claude/*` exclusion), so a user's re-inclusion block survives every sync. Everything else keeps strict literal equality: non-`.gitignore` destinations (notably `.envrc`, Guardrail A), **and** the non-directory patterns shipped in the same fragment (`.fab-*`, `.status.yaml.lock`, and `!fab/.fab-version` (8ken)). The non-directory scoping (Guardrail C) is what stops an anchored `/.status.yaml.lock` (root-only) from being mistaken as covering the unanchored, at-any-depth fragment `.status.yaml.lock`, and stops a `!/.status.yaml.lock` negation from hard-stopping it — either would suppress the broader ignore and let nested `fab/changes/**/.status.yaml.lock` files be committed. The shipped fragment default (`fragment-.gitignore`'s `/.claude`) is unchanged; the fix is the dedup recognizing equivalent existing forms for directory tokens, not changing what is emitted into a fresh file.

**Negation lines take the strict-literal path (8ken).** The fragment's `!fab/.fab-version` negation (un-ignoring the relocated version file — see [kit-architecture.md](/distribution/kit-architecture.md) and [configuration.md](/_shared/configuration.md) § `fab_version`) is a **non-directory token**: `gitignoreIsDirectoryToken(entry)` is false for it (no leading `/`, no trailing `/`) and false for the `.fab-*` line above it (contains `*`), so **both** take the strict-literal-dedup path — the negation is appended once if absent and re-merges idempotently. Because the Guardrail-B negation **hard-stop** (`gitignoreHasNegation`, `scaffold.go`) is itself gated on `gitignoreIsDirectoryToken`, it is **never consulted** for either non-directory line — so adding `!fab/.fab-version` to the fragment cannot suppress the `.fab-*` ensure (the two lines coexist, last-match-wins un-ignoring the file). This is the same class as `.fab-*`/`.status.yaml.lock`, not the directory-token class that Guardrail B guards.

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

### Templates in Scaffold Files
**Decision**: `config.yaml` and `constitution.md` templates live as standalone files in `$(fab kit-path)/scaffold/` rather than as inline code blocks in `fab-setup.md`. `/fab-setup` reads from these files and substitutes placeholders. Index templates (`memory-index.md`, `specs-index.md`) are also referenced from scaffold files, eliminating duplicated inline copies.
**Why**: Prevents drift between inline templates and actual schema expectations. Aligns with Constitution V (Portability) — `.kit/` owns its templates as inspectable, diffable files. Single source of truth for both `fab-kit sync` and `/fab-setup`.
**Rejected**: Keeping inline templates — two sources of truth that can diverge when the config schema evolves.
*Introduced by*: 260217-17pe-DEV-1046-scaffold-setup-templates

### Language-Neutral Bootstrap — No Language Templates, No Inference Step
**Decision**: The bootstrap flow carries no language-specific customization step: fab-kit ships no language templates and runs no language-inference step. Projects that want language-specific conventions add them manually to `fab/project/*` files.
**Why**: fab-kit stays language-neutral per Constitution §V (portability — no assumptions about the host project's language/toolchain). Language content either encodes opinions that may not match the project (templates) or produces content with no template to route it to (inference).
**Rejected**: Bundled language templates in `$(fab kit-path)/templates/` (violate neutrality, maintenance burden, judgment calls on behalf of users) (143f). Agent-inferred conventions (a stepping stone: marker-file detection + training-knowledge inference — detection has no purpose without language-specific content to produce) (143f).
*Introduced by*: 260306-6bba-redesign-hooks-strategy

### Sync-First Bootstrap Order; Hand-Scaffolding Steps Deleted
**Decision**: In the bare bootstrap, `fab sync` runs as step 1c — immediately after the interactive config (1a) and constitution (1b) steps and before anything else — guarded by a STOP on non-zero exit. The seven steps that hand-duplicated sync's scaffolding (old 1c–1g skeleton copies, old 1i directory creation, old 1k `.gitignore` append) are deleted; sync is the single owner of non-interactive structural setup. Sync cannot move before 1a because it requires `config.yaml`'s `fab_version` (the fab router errors without it).
**Why**: Every scaffold artifact was described twice — once as a skill step, once inside sync — so each scaffold change had to land in both places, and the copies had already drifted in detail. Sync's operations are copy-if-absent / line-ensure merges, so running it earlier produces an identical file tree via idempotency. This was the one explicit behavior-ORDER change in its batch (f077), flagged in the PR description.
**Rejected**: Keeping sync last with the hand-scaffolding steps as "idempotent guards" (the duplication is the maintenance cost, not the ordering). Moving sync before the interactive steps (sync hard-requires `fab_version` from 1a). Deleting the steps without a sync-failure guard (a failed sync would otherwise be partially papered over by hand-scaffolding; with single ownership, sync failure must stop the bootstrap).
*Introduced by*: 260611-szxd-skills-twins-self-duplication-refactor

### Absorbed /fab-update into /fab-setup migrations
**Decision**: `/fab-update` functionality is now available as `/fab-setup migrations`. Version migrations live under the same command namespace as the rest of project setup.
**Why**: Reduces the dropped-ball two-step flow where users had to remember a separate `/fab-update` command after upgrading the kit. Makes migrations discoverable from the same command namespace as config and constitution management.
**Rejected**: Keeping `/fab-update` as a separate top-level skill — created a discoverability gap and a two-step flow that was easy to forget.
*Introduced by*: 260216-tk7a-DEV-1037-consolidate-setup-upgrade-flow

### Setup-State Doctor as a `fab setup check` Subcommand, Coexisting with `fab doctor`
**Decision**: The setup-state doctor ships as the `fab setup check` subcommand, leaving bare `fab setup` to the interactive setup wizard. It coexists with fab-kit's seven-check `fab doctor` by distinct job — doctor checks system prerequisites, `setup check` checks fab's own setup state.
**Why**: A subcommand reads as a distinct read-only operation beside the wizard's interactive bare command; the read-only probe layer is safe on its own (it writes nothing, so it is trivially idempotent) and gives the wizard a detection layer to build its provider-filtered interview on. Splitting by job keeps each doctor in the binary that owns its domain (fab-kit system prerequisites vs fab-go setup state).
**Rejected**: A `--check` flag on `fab setup` — couples the doctor to the wizard's flag surface. Subsuming `fab doctor` into the new command or vice versa — different binaries with different jobs; the gh/yq presence overlap is cheaper than a cross-binary dependency.
*Introduced by*: 260811-pgbq-setup-check-doctor; *Updated by*: 260811-stpw-setup-interactive-wizard

### Doctor Exit Contract — Only Failures Exit 1
**Decision**: `fab setup check` exits 0 on a healthy or warnings-only report, 1 only when at least one failure-severity finding exists, and 2 on usage errors via the existing `run()`/`markRunReached` seam — no distinct in-handler warnings tier.
**Why**: Keeps the doctor CI-able as a "real problems" gate without crying wolf on advisory findings (version skew, load-bearing overrides, descending dispatch modes, absent optional tooling).
**Rejected**: A third exit code for warnings-only reports — an undocumented extra runtime code would complicate the CI contract and the binary-wide 0/1/2 classification.
*Introduced by*: 260811-pgbq-setup-check-doctor

### Wizard Consumes the Setup-Check Report — Capability Detected, Never Asked
**Decision**: All detection comes from one `setupcheck.Run` call; the interview only asks preference questions over options pre-filtered to what the probe found.
**Why**: The probe package returns its structured `Report` for exactly this consumer; options filtered to detected providers and viable rungs mean the wizard cannot configure something the machine can't run.
**Rejected**: Wizard-owned probing or shelling out to `fab setup check` — duplicated detection logic, parse coupling to human-readable output.
*Introduced by*: 260811-stpw-setup-interactive-wizard

### Wizard Lives in package main Beside the Set/Origin Seams
**Decision**: The interview loop lands in `src/go/fab/cmd/fab/setup_wizard.go` in package main (sibling of `setup.go`), not a new internal package.
**Why**: The seams it needs — `configMutationPath`, `effectiveTierFor`, `warnIfShadowed`, the stdin-TTY helper, `version` — are unexported members of package main; an internal package would force exporting write-path plumbing for one consumer. The cmd-owns-wiring / internal-owns-probing split is preserved: the wizard IS wiring/rendering.
**Rejected**: A new `internal/setupwizard` package — export churn for no reuse; nothing else consumes an interview loop.
*Introduced by*: 260811-stpw-setup-interactive-wizard

### Advanced Section Asks All Keys When Opted In
**Decision**: Answering yes at Q4 asks every advanced question regardless of winning tier. The sparse profile keys render a depth-correct inherit indication (`(inherit agent.session)` / `(inherit agent.workers)`) over an empty baseline whenever their winning tier is the derived built-in default, so Enter keeps the inherit and even typing the currently-inherited provider is an explicit pin that writes.
**Why**: Q4's opt-in (default N) already carries the consent the original skip rule guarded, and on exactly the machines whose owners want to set an advanced knob for the first time (nothing overridden yet), a review-only section was a dead end — the prompt asked whether to *configure* the options, then declined to configure them. Enter-keeps-current preserves the all-Enter zero-write invariant (Constitution III).
**Rejected**: The original skip rule (ask only overridden keys — the review-existing-customizations posture shipped by 260811-stpw) — made first-time advanced setup unreachable through the wizard; a per-key "set it now?" confirm — four extra prompts expressing what Q4's `y` already expressed.
*Introduced by*: 260811-stpw-setup-interactive-wizard; *Updated by*: 260812-nwdn-setup-wizard-advanced-ask-all

### Non-TTY Without `--defaults` Errors
**Decision**: When stdin is not a TTY and `--defaults` was not passed, the wizard fails with a usage hint naming the flag.
**Why**: Predictable failure beats hanging on a read or silently pretending to be interactive; `--defaults` is the sanctioned non-interactive path.
**Rejected**: Auto-degrading to `--defaults` — makes CI invocations silently succeed in a mode the caller didn't choose.
*Introduced by*: 260811-stpw-setup-interactive-wizard
