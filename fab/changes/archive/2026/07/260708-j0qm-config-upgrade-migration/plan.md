# Plan: Config Upgrade Command & fab_version Migration

**Change**: 260708-j0qm-config-upgrade-migration
**Intake**: [intake.md](intake.md)

> Change 3 of the config-upgrade effort. Design authorities: `fab/plans/sahil/config-upgrade.md`
> (§ Change 3, § Resolved decisions, § The fence worked example) and `docs/specs/config.md`
> (§ Forward-looking intent Change 3). Changes 1 (ff2v) and 2 (lpb5) landed the registry
> (`internal/configref`, ordered `[]Field` with `default`/`description`/`scope`/`advertise`/
> `renamed_from`/`Segment`) and the three-layer cascade (`internal/config.LoadPath`,
> `internal/configscope`). This change completes the design: comments become regenerable
> generated output inside a managed fence, `fab config upgrade` becomes the only writer of
> config.yaml, `fab_version` leaves config.yaml for `fab/.fab-version`, and the scaffold
> config.yaml is deleted in favor of registry-driven init generation.

---

## Requirements

### R1 — HOME hermeticity for the fab-go test suite (inherited follow-up)

The Change-2 cascade made `internal/config.LoadPath` merge `~/.fab-kit/config.yaml` via the
`homeDir = os.UserHomeDir` seam. Test packages that assert exact bytes on resolved
`providers`/`agent.tiers` output can be perturbed by a real system config on the developer's
machine. Only `providers` and `agent` are `ScopeBoth` (every other key is `ScopeProject`, pruned
from the system layer), so the vulnerable surface is exactly the provider/agent-resolving tests.

- **R1.1** The fab-go module MUST isolate `$HOME` to a per-test temp dir in every test package
  that resolves `providers`/`agent.tiers` and asserts on the result: `cmd/fab`, `internal/spawn`.
  MUST land before any new test in this change writes a real system config.
  - GIVEN a developer has a real `~/.fab-kit/config.yaml` with an `agent.tiers` override
    WHEN `go test ./...` runs in `src/go/fab`
    THEN `cmd/fab/resolve_agent_test.go`, `cmd/fab/agent_test.go`, `cmd/fab/batch_new_test.go`,
    `cmd/fab/batch_switch_test.go`, `cmd/fab/dispatch_start_test.go`, and
    `internal/spawn/spawn_test.go` still pass (their resolved-command assertions see only the
    project config).

### R2 — `fab config upgrade` subcommand (fab-go)

A new `fab config upgrade` cobra subcommand rewrites `fab/project/config.yaml` mechanically over
the registry, under the A/B/C field-category model (`docs/specs/config.md` § Advertise semantics),
writing via `internal/atomicfile.WriteFile`.

- **R2.1 — Live (A) fields kept verbatim.** A live top-level key that is NOT the managed fence and
  NOT a parked-removal block MUST be preserved byte-for-byte, including the user's own comments on
  it. Presence = intent (decision 2): a live field is an override even when its value equals the
  default; it is NEVER auto-removed. B-hygiene ("equals default — remove?") is an advisory report
  line only, never a mutation.
  - GIVEN config.yaml has a user comment `# pin review to fable` above a live `agent:` block
    WHEN `fab config upgrade` runs
    THEN that comment and the `agent:` block are preserved byte-for-byte.
- **R2.2 — The managed fence (C fields).** `advertise: true` registry fields NOT currently
  overridden MUST be rendered as a fully-commented scaffold inside a managed fence:
  - Byte-exact splice anchors: a BEGIN line `# >>> fab reference (kit X.Y.Z) >>> ` padded with
    `-` to a fixed width, and an END line `# <<< end fab reference <<< ` similarly padded. Upgrade
    rewrites ONLY the region between (and including) the two anchor lines; everything outside is
    the user's.
  - The kit-version stamp `X.Y.Z` in the BEGIN line is the running binary's version.
  - Every scaffolded block MUST be fully commented including parent keys (a live `agent:` key over
    comment-only children is exactly what the old masher collapsed to `agent: null`).
  - The fence MUST omit fields already overridden as live keys above it (shows what you *could*
    override but haven't).
  - When a legacy file has no fence, upgrade MUST append one at the bottom.
  - GIVEN config.yaml has a live `agent.tiers.review` override and no fence
    WHEN `fab config upgrade` runs
    THEN a fence is appended whose commented scaffold advertises `providers`, `test_paths`,
    `true_impact_exclude`, `checklist.extra_categories`, `stage_hooks`, `branch_prefix` but NOT the
    already-overridden `agent.tiers`.
- **R2.3 — Unknown fields parked, never deleted.** A live top-level key no longer in the registry
  MUST be parked in a `# removed in X.Y.Z (parked by fab config upgrade — delete when done):`
  comment block BELOW the fence, with the user's value serialized in the comment. Parkings are
  user territory: appended exactly once, never regenerated away on a subsequent run.
  - GIVEN config.yaml has a top-level `legacy_mode: true` key absent from the registry
    WHEN `fab config upgrade` runs
    THEN a parked block below the fence carries `#   legacy_mode: true`, the live key is removed
    from the active YAML, AND a second run leaves the parked block byte-identical.
- **R2.4 — Renames carried mechanically.** A live field whose top-level key matches some registry
  row's `renamed_from` MUST be carried to the new key (value verbatim). `renamed_from` is `""` on
  every registry row today, so this path is exercised only by test fixtures with a seeded rename;
  it MUST NOT fire for any real field today.
- **R2.5 — Byte-stable and idempotent.** Running `fab config upgrade` twice MUST yield a
  byte-identical file (the `fab memory-index` discipline: golden + idempotence tests). The write
  MUST go through `internal/atomicfile.WriteFile`.
  - GIVEN any config.yaml
    WHEN `fab config upgrade` runs twice
    THEN the file after the second run is byte-identical to after the first.
- **R2.6 — Pure over the registry.** The fence/parking logic MUST read the field set, scope,
  advertise flag, `renamed_from`, and rendered `Segment` from `configref.Fields()` — no second copy
  of the schema.

### R3 — `fab_version` relocation to `fab/.fab-version`; `setFabVersion` deleted

- **R3.1 — New sibling file.** `fab/.fab-version` MUST be a one-line plain-text file (bare semver +
  `\n`), written by a `stampMigrationVersion`-shaped helper. Kept separate from
  `.kit-migration-version`.
  - GIVEN a fresh `fab init` or a `fab upgrade-repo`
    WHEN version stamping runs
    THEN `fab/.fab-version` contains `{version}\n` and no `fab_version:` key is written into
    config.yaml.
- **R3.2 — `setFabVersion` and `topLevelFabVersionValue` deleted** from fab-kit `internal/init.go`;
  their callers (`Init`, `Upgrade`) stamp `fab/.fab-version` instead. After this, no fab-kit code
  writes config.yaml.
- **R3.3 — Both reader stacks move to `.fab-version` with a config.yaml `fab_version:` fallback**
  for one compat window:
  - fab-kit `readFabVersion` (feeding `ConfigResult.FabVersion`) MUST read `fab/.fab-version`
    first, then fall back to the config.yaml `fab_version:` key.
  - fab-go `internal/config` MUST resolve `FabVersion` from `fab/.fab-version` first, then the
    config.yaml `fab_version:` key (the sole consumer is `preflight`'s staleness check).
  - GIVEN a repo with `fab/.fab-version: 2.15.0\n` and no `fab_version:` in config.yaml
    WHEN the router resolves the pinned version, or preflight checks staleness
    THEN they read `2.15.0` from `.fab-version`.
  - GIVEN a repo with a `fab_version:` in config.yaml and no `.fab-version` file (pre-migration)
    WHEN either reader runs
    THEN it falls back to the config.yaml value (no error, no behavior change).
- **R3.4 — Registry/scope cleanup.** The `fab_version` row MUST be removed from
  `configref.Fields()`, and its `fab_version` entry removed from `configscope.keyScopes`. The
  reference no longer documents `fab_version` (it is machine-managed and out of the file).

### R4 — `fab upgrade-repo` auto-runs the upgrader (fab-kit)

- **R4.1** After `runSync` and version stamping, `Upgrade` MUST shell out to the installed fab-go
  binary's `fab config upgrade` (the pinned binary from `EnsureCached`, the `execFabGo` precedent).
- **R4.2 — Fail-open.** If the installed fab predates the subcommand (non-zero exit / unknown
  command), `Upgrade` MUST print a reminder and continue — an upgrade MUST never break on the
  config step.
  - GIVEN an installed fab-go that lacks `fab config upgrade`
    WHEN `fab upgrade-repo` runs
    THEN it prints a reminder and completes the upgrade successfully.

### R5 — Scaffold config.yaml deleted; `fab init` generates from the registry

- **R5.1 — Scaffold file deleted.** `src/kit/scaffold/fab/project/config.yaml` MUST be deleted.
  The scaffold tree walk no longer carries a config.yaml; copy-if-absent semantics for all other
  scaffold files are preserved.
- **R5.2 — `fab config init` project mode.** `fab config init` MUST gain a project mode
  (distinct from `--system`) that writes an initial `fab/project/config.yaml` from the registry:
  the A-class identity fields (`project.name`, `project.description`, `source_paths`, `test_paths`)
  written live from seed values passed as flags, plus the managed fence for the advertise-flagged
  fields. It MUST refuse to overwrite an existing config.yaml.
  - GIVEN a repo with no config.yaml
    WHEN `fab config init --project --name X --description Y --source-path src/ [--test-path P]`
    runs
    THEN config.yaml is written with the seeded identity fields live and the managed fence below.
- **R5.3 — fab-kit `Init` generates config.yaml via shell-out.** fab-kit's `Init` MUST shell out
  to the installed fab-go `fab config init --project ...` (same one-brew-package skew + fail-open
  discipline as R4), passing detected project name / source paths / test paths as seed input.
- **R5.4 — Embedded stub fallback.** When the installed fab predates `fab config init --project`,
  fab-kit MUST write a minimal embedded stub config.yaml (a tiny bounded copy of the A-class
  identity fields) — NOT a printed instruction. A fresh repo MUST never fail preflight for lack of
  a config.yaml.
  - GIVEN an installed fab-go that lacks `fab config init --project`
    WHEN `fab init` runs
    THEN a minimal stub config.yaml exists and preflight passes.
- **R5.5 — new-vs-existing classification preserved.** `scaffoldDirectories` uses config.yaml
  presence to classify a repo as new vs existing. Since `Init` now generates config.yaml (before
  the scaffold walk, as `setFabVersion` did), that classification MUST continue to behave correctly
  (a fresh `fab init` is still "new project").
- **R5.6 — Registry init/seed metadata.** The registry MUST carry which fields are written live at
  init (the A-class identity fields), consumed by the project-mode generator. The scaffold's extra
  prose (multi-language `test_paths` examples, providers narrative) is folded into registry
  descriptions/segments where not already present.
- **R5.7 — `agent.tiers` pinning dies.** The generated config.yaml MUST NOT pin `agent.tiers` live
  (under presence=intent, init-pinned tiers are an accidental override). Fresh projects inherit
  the defaults; the fence advertises the override surface.

### R6 — Migration (constitution: user-data restructure)

- **R6.1** A new `src/kit/migrations/2.14.0-to-2.15.0.md` MUST move the `fab_version:` value from
  `fab/project/config.yaml` to a new `fab/.fab-version` file and delete the key (and its stale
  comment line) from config.yaml. Structure per the `2.13.1-to-2.13.2` precedent: Summary /
  Pre-check (sentinel + idempotency gates) / ordered Changes (atomic temp+rename) / Verification.
- **R6.2 — Sentinel-guarded + idempotent.** `.fab-version` already present AND the key absent from
  config.yaml ⇒ complete no-op.
- **R6.3** Historical comment-backfill migrations are left untouched; the pattern is retired going
  forward (documented, no code change).

### R7 — Test coverage (constitution VII)

- **R7.1** Every Go code change MUST ship tests alongside it. The fence generator MUST have golden
  tests (full-document byte assertions) and an idempotence test, per the `internal/memoryindex`
  precedent. The `.fab-version` readers MUST have fallback-path tests. The auto-run and stub
  fallback paths MUST have tests exercising both the present-subcommand and predates-subcommand
  branches.
- **R7.2 — Superset test replaced.** `TestConfigReferenceSupersetsScaffoldKeys`
  (`cmd/fab/config_test.go`) MUST be retired/replaced once the scaffold file is gone — its
  skill-consumed-key guard is re-anchored to a registry-internal invariant (init-seeded/stub keys
  ⊆ registry keys).

### R8 — Documentation obligations

- **R8.1** `src/kit/skills/_cli-fab.md` § fab config MUST document `upgrade` and the `init`
  project mode (the section already forward-refs Change 3).
- **R8.2** `docs/specs/config.md` § Forward-looking intent (Change 3) MUST be flipped to landed, in
  authoritative detail (same treatment Changes 1/2 got).
- **R8.3** Skill/SPEC mirrors MUST be swept: `src/kit/skills/fab-setup.md` Config Create-Mode
  (scaffold + placeholder detection → generated path) and its `docs/specs/skills/SPEC-fab-setup.md`
  mirror; `docs/specs/skills/SPEC-_cli-fab.md` (fab config subcommand facts). Grep-sweep
  `scaffold`, `fab config`, `fab_version` across `src/kit/skills/` + `docs/specs/` for other hits.

### Non-Goals

- No `--check` drift mode in this change (the version stamp merely enables it later — assumption 8).
- Historical comment-backfill migrations are NOT backfilled or rewritten (assumption 7).
- No new built-in providers; codex/gemini remain template text.
- Memory files (`docs/memory/`) are updated at hydrate, not apply.

### Design Decisions

- **Migration TO version = 2.15.0.** Installed VERSION is `2.14.0`; the fence worked example in the
  design doc stamps `kit 2.15.0`. This change bumps `src/kit/VERSION` `2.14.0 → 2.15.0` (a feature
  release adding a subcommand + user-data restructure) and names the migration `2.14.0-to-2.15.0`.
- **New upgrader package `internal/configupgrade`.** The fence splice + parking logic is new code
  with no existing home; a leaf package keeps `cmd/fab/config.go` thin and mirrors how
  `internal/memoryindex` owns its rendering. It reads `configref.Fields()` and writes via
  `internal/atomicfile`. Rejected: extending `internal/configref` (which is a pure renderer with no
  file I/O — mixing the writer in would muddy its single responsibility).
- **HOME isolation via a package-level `TestMain` per vulnerable package.** Neither `cmd/fab` nor
  `internal/spawn` has a `TestMain`; adding one that sets `os.Setenv("HOME", t.TempDir()-equivalent)`
  before `m.Run()` isolates the whole package cheaply, with no per-test edits. `internal/config`
  already isolates via `isolateSystemConfig`; leave it. Rejected: editing every helper
  (`resolveAgentTestRepo`, `agentTestRepo`, …) individually — more churn, easy to miss one.
- **`fab config init` gains `--project` (mutually exclusive with `--system`).** Grammar:
  `--project --name <s> --description <s> --source-path <s> (repeatable) --test-path <s>
  (repeatable)`. Follows the `init --system` precedent (refuse-to-overwrite, generated from the
  registry). Bare `fab config init` stays a usage error.

---

## Tasks

### Phase 1: Setup — HOME hermeticity (must land first)

- [x] T001 Add `TestMain(m *testing.M)` to package `cmd/fab` (new file `cmd/fab/main_test.go` or
  in `testhelpers_test.go`) that sets `HOME` to a fresh temp dir before `m.Run()`, so
  `os.UserHomeDir`/`~/.fab-kit/config.yaml` never sees the developer's real system config. (HOME-only:
  `os.UserHomeDir` honors `$HOME` on the unix target; no `USERPROFILE` shim is emitted.) Verify no
  existing `TestMain` in the package. <!-- R1 -->
- [x] T002 [P] Add an equivalent `TestMain` to `internal/spawn` (new file
  `internal/spawn/main_test.go`) isolating `HOME`. <!-- R1 -->
- [x] T003 Run `cd src/go/fab && go test ./cmd/fab/... ./internal/spawn/...` to confirm the exact-byte
  provider/agent assertions still pass with HOME isolated (baseline before any config-write test lands). <!-- R1 -->

### Phase 2: Core — `fab_version` relocation + registry cleanup

- [x] T004 In fab-go `internal/config/config.go`: change `FabVersion` resolution so `LoadPath`
  (and/or the `Config`/`GetFabVersion` path) reads `fab/.fab-version` first, falling back to the
  config.yaml `fab_version:` key. Add a helper `readDotFabVersion(fabRoot)` (or resolve at the
  `Load`/`LoadPath` seam where `fabRoot` is known). Preserve nil-safety of `GetFabVersion()`. <!-- R3.3 -->
- [x] T005 In fab-go `internal/configref/configref.go`: remove the `fab_version` Field row from
  `Fields()`. In `internal/configscope/configscope.go`: remove the `"fab_version"` entry from
  `keyScopes`. Update the package doc/comments that enumerate `fab_version` as a documented key. <!-- R3.4 -->
- [x] T006 In fab-kit `internal/init.go`: delete `setFabVersion` and `topLevelFabVersionValue`; add
  a `stampFabVersion(repoRoot, version)` helper (shaped like `stampMigrationVersion`, writing
  `fab/.fab-version` as `{version}\n`). Update `Init` (line ~41) to call `stampFabVersion` instead of
  `setFabVersion`. <!-- R3.1 R3.2 -->
- [x] T007 In fab-kit `internal/config.go`: update `readFabVersion` (or its caller `resolveConfigFrom`)
  to read `fab/.fab-version` first, falling back to the config.yaml `fab_version:` key. Keep
  `ConfigResult.FabVersion` semantics (empty when neither source has it). <!-- R3.3 -->
- [x] T008 In fab-kit `internal/upgrade.go`: update `Upgrade` (the `setFabVersion` call at ~line 120)
  to call `stampFabVersion(cfg.RepoRoot, targetVersion)` after `runSync`. Keep the sync-before-stamp
  ordering. <!-- R3.1 R3.2 -->
- [x] T009 Tests: fab-go `internal/config` `.fab-version`-first + config.yaml-fallback resolution;
  fab-kit `internal/config` `readFabVersion` fallback; fab-kit `internal/init` stamps `.fab-version`
  and never writes config.yaml `fab_version:`; `internal/configref` reference no longer documents
  `fab_version` (update any test asserting the `fab_version` block; the byte-stable/superset tests
  will shift). Run affected package tests. <!-- R7.1 R3 -->

### Phase 3: Core — the `fab config upgrade` engine

- [x] T010 Create `internal/configupgrade/configupgrade.go`: the fence/parking engine. <!-- rework DONE (cycle 1): (1) splitFence now returns below-fence NON-parked content separately; render hoists it above the regenerated fence (never dropped) — regression test TestRender_BelowFenceLiveOverrideHoisted + golden TestGolden_BelowFenceContentHoisted_FullDocument. (2) commentOutSegment exported as configupgrade.CommentOutSegment; cmd/fab copy deleted. Also folded: rename-carry hardening (skip when target live), interior-column0-comment block capture, YAML parse-refuse before write, B-hygiene equals-default advisory, yamlScalar backslash/control escaping. --> Public entry
  `Upgrade(configPath string, kitVersion string) (changed bool, report []string, err error)` (exact
  signature decided at implementation). Read `configref.Fields()`. Define byte-exact anchor
  constants for the BEGIN/END fence lines (`# >>> fab reference (kit %s) >>> ` + dash pad, `# <<< end
  fab reference <<< ` + dash pad). Implement: parse existing file → identify live top-level keys →
  classify each registry field A (live) / C (advertise & not-live) → build the commented fence for C
  fields (fully commented incl. parents, omitting live keys) → splice between anchors (append at
  bottom if no fence) → park unknown live keys below the fence once → carry `renamed_from` renames →
  write via `atomicfile.WriteFile`. Must be byte-stable and idempotent. <!-- R2.1 R2.2 R2.3 R2.4 R2.5 R2.6 -->
- [x] T011 [P] Golden tests `internal/configupgrade/golden_test.go`: full-document byte assertions <!-- rework DONE (cycle 1): golden_test.go rewritten to full-document literal `got != want` over a small synthetic field set (goldenFields) — mirrors memoryindex/golden_test.go. Anchors composed from the same beginPrefix/endPrefix/fenceWidth constants so a deliberate anchor-format change updates in lockstep. Behavioral shipped-registry tests moved to configupgrade_test.go (kept, not lost). -->
  for the key cases — legacy file with no fence (fence appended), file with an existing fence
  (region rewritten, outside preserved), live-override omission from the fence, a user comment on an
  A field preserved, an unknown key parked. Mirror the `memoryindex/golden_test.go` shape. <!-- R7.1 R2 -->
- [x] T012 [P] Idempotence test `internal/configupgrade/freeze_test.go`: run `Upgrade` twice, assert
  byte-identical output; assert a parked block is not regenerated/duplicated on the second run;
  assert a `renamed_from` carry (seeded fixture) fires once. Mirror `memoryindex/freeze_test.go`. <!-- R7.1 R2.5 R2.3 R2.4 -->
- [x] T013 In `cmd/fab/config.go`: add `configUpgradeCmd()` wired into `configCmd()`. It resolves the
  repo root (`resolve.FabRoot`), computes the project config path, reads the binary version for the
  kit stamp, calls `configupgrade.Upgrade`, and prints the advisory report (B-hygiene lines, parked
  notices). `cobra.NoArgs`. Add a command-level test in `cmd/fab` (uses the isolated HOME from T001). <!-- R2 R7.1 -->

### Phase 4: Core — auto-run + init generation

- [x] T014 In fab-kit `internal/upgrade.go`: after `runSync` + `stampFabVersion`, shell out to the
  pinned fab-go (`EnsureCached(targetVersion)`) `fab config upgrade` via `exec.Command`. On non-zero
  exit / unknown-command, print a fail-open reminder and continue (do NOT return the error).
  Extract a small helper so it is testable. <!-- R4.1 R4.2 -->
- [x] T015 In `cmd/fab/config.go`: extend `configInitCmd()` with a `--project` mode (mutually
  exclusive with `--system`) taking `--name`, `--description`, `--source-path` (repeatable),
  `--test-path` (repeatable). It writes `fab/project/config.yaml` from the registry: A-class identity
  fields live from the seed values, then the managed fence (reuse `configupgrade`'s fence renderer,
  or a shared render helper). Refuse to overwrite an existing config.yaml. Bare `fab config init`
  stays a usage error. Add tests (project-mode write, overwrite refusal, agent.tiers NOT pinned). <!-- R5.2 R5.7 R7.1 -->
- [x] T016 In fab-go `internal/configref/configref.go`: add init/seed metadata marking the A-class
  identity fields written live at init (`project.name`, `project.description`, `source_paths`,
  `test_paths`). Fold the scaffold's extra prose (multi-language `test_paths` examples; any providers
  narrative not already present) into the relevant registry `Segment`/`Description` text. Add a test
  asserting the init-seed field set. <!-- R5.6 -->
- [x] T017 In fab-kit `internal/init.go`: replace the (now-deleted) `setFabVersion` config.yaml write <!-- rework DONE (cycle 1): generateProjectConfig now runs mechanical detection (detectProjectSeed: repo-folder name, existing src/, marker-table test_paths via detectTestPaths) and passes --name/--source-path/--test-path to `fab config init --project`; the embedded stub (renderStubConfig) carries the same seed. E2E verified against a real fab-go build (config carries live project/source_paths/test_paths). Tests: TestGenerateProjectConfig_PassesDetectedSeed (fake-fab-go args capture, success path), TestDetectProjectSeed(_NoMarkers), stub carries detected name. -->
  with generation — shell out to pinned fab-go `fab config init --project ...` passing detected
  name/source_paths/test_paths. On predates-subcommand (non-zero/unknown), write a minimal embedded
  stub config.yaml (bounded copy of the identity fields). Runs BEFORE the scaffold walk (preserving
  the new-vs-existing classification in `scaffoldDirectories`). fab-kit's detection of project name /
  source paths / test paths becomes generator input. <!-- R5.3 R5.4 R5.5 -->
- [x] T018 In fab-kit `internal/scaffold.go`: confirm the scaffold tree walk no longer needs to carry
  config.yaml (it is generated in `Init` now); the copy-if-absent classification in
  `scaffoldDirectories` (config.yaml presence → existing project) must still hold against the
  generated file. No behavior regression for existing repos. Add/adjust tests. <!-- R5.1 R5.5 -->
- [x] T019 Delete `src/kit/scaffold/fab/project/config.yaml`. <!-- R5.1 -->
- [x] T020 Retire/replace `TestConfigReferenceSupersetsScaffoldKeys` in `cmd/fab/config_test.go`:
  its scaffold-key superset guard has no file to read once the scaffold is gone. Replace with a
  registry-anchored guard (the init-seeded/stub key set ⊆ registry key set). Remove the now-dead
  `scaffoldConfigRelPath`/`scaffoldKeyTokens` test helpers if unused elsewhere. <!-- R7.2 -->
- [x] T021 fab-kit tests: `Init` generates config.yaml via shell-out with the stub fallback on the
  predates-subcommand path; `Upgrade` auto-runs the upgrader and fails open when the subcommand is
  absent. <!-- R7.1 R4 R5 -->

### Phase 5: Migration + version bump

- [x] T022 Write `src/kit/migrations/2.14.0-to-2.15.0.md` (structure per `2.13.1-to-2.13.2`):
  Summary; Pre-check (skip if no config.yaml; sentinel: skip when `fab/.fab-version` present AND no
  `fab_version:` key ⇒ no-op); Changes (read `fab_version:` value from config.yaml → write
  `fab/.fab-version` as `{value}\n` via atomic temp+rename → delete the `fab_version:` line and its
  stale comment line from config.yaml, preserving all other keys/comments verbatim); Verification
  (`.fab-version` present with the moved value; `fab_version:` key absent; YAML still parses; re-run
  is a no-op). Note the comment-backfill pattern retired going forward. <!-- R6.1 R6.2 R6.3 -->
- [x] T023 Bump `src/kit/VERSION` `2.14.0` → `2.15.0`. <!-- R6.1 Design Decisions -->

### Phase 6: Documentation

- [x] T024 [P] `src/kit/skills/_cli-fab.md` § fab config: document `fab config upgrade` (fence
  contract, A/B/C, parked removals, byte-stable/idempotent, only writer of config.yaml) and the
  `fab config init --project` mode; add `upgrade`/`init --project` to the command summary block.
  Remove the `fab_version` key from the "Full schema coverage" enumeration (it left the file). <!-- R8.1 -->
- [x] T025 [P] `docs/specs/config.md`: flip § Forward-looking intent (Change 3) to landed —
  authoritative detail on the fence, presence=intent, parked removals, `.fab-version` relocation,
  the auto-run, scaffold deletion + init-from-registry. Update the scope-taxonomy prose that lists
  `fab_version` (it is no longer a config.yaml field). Mark Change 3 landed in the header banner. <!-- R8.2 -->
- [x] T026 [P] `src/kit/skills/fab-setup.md` Config Create Mode: rewrite steps 3-6 from <!-- rework DONE (cycle 1): step 3 rewritten from "replace placeholder identity values" to "REFINE the fab-init-seeded live values + ADD the description (not seeded)"; added a "What fab init already seeded" note naming the folder-name/src/marker-table detection; test_paths sub-step reframed as confirm/refine (skill adds JS/TS deps the Go layer skips). Verified against the landed flow. -->
  "read scaffold + substitute placeholders" to "shell out to `fab config init --project` with
  detected seed values (stub fallback if the binary predates it)"; drop the `fab_version` preserve/
  stamp step (fab_version now lives in `fab/.fab-version`, stamped by `fab init`, out of config.yaml).
  Keep the non-interactive test_paths detection (it becomes generator input). Update the Ordering
  note and the create-mode trigger prose. <!-- R8.3 -->
- [x] T027 [P] `docs/specs/skills/SPEC-fab-setup.md`: mirror the fab-setup.md Config Create-Mode <!-- rework DONE (cycle 1): both Summary paragraphs (test_paths detection + Config generation) and the Phase 1a box updated — "passing a mechanically-detected identity seed as --name/--source-path/--test-path flags", "refines + adds the description", description-not-detected caveat. Byte-consistent with fab-setup.md's corrected claims. -->
  rewrite (the Phase 1a box: Read scaffold → generate via `fab config init --project`; the
  fab_version-preserve note). <!-- R8.3 -->
- [x] T028 [P] `docs/specs/skills/SPEC-_cli-fab.md`: add `upgrade` and `init --project` to the
  `fab config` row; note `fab_version` moved to `fab/.fab-version` and is no longer a documented
  config key. <!-- R8.3 -->
- [x] T029 Sweep remaining `scaffold` / `fab config` / `fab_version` hits across `src/kit/skills/` +
  `docs/specs/` (from the grep in the survey): update any that state scaffold-drives-config-create,
  that `fab_version` lives in config.yaml, or that enumerate `fab config` subcommands, to match the
  landed design. (Migration files under `src/kit/migrations/` that reference the historical scaffold
  are left untouched — they are historical records.) <!-- R8.3 -->

### Phase 7: Verification

- [x] T030 Run `cd src/go/fab && go test ./...` and `cd src/go/fab-kit && go test ./...`; fix
  failures (up to 3 attempts each). Then `fab status refresh`. <!-- R7 --> <!-- rework DONE (cycle 1): both full suites green (fab-go all packages ok incl. cmd/fab; fab-kit all ok); `go vet ./...` clean in both modules. -->

## Execution Order

- Phase 1 (HOME hermeticity) MUST complete before any Phase 3/4 test that writes a config (T001-T003
  before T011-T013, T015, T021).
- T005 (registry `fab_version` removal) and T004/T007 (`.fab-version` readers) should land together —
  removing the row while the reader still expects the key would break the fallback semantics; do
  T004+T007 first, then T005.
- T010 (engine) before T011/T012 (its tests) and before T013 (the command) and T015 (init reuses the
  fence renderer).
- T019 (delete scaffold) after T016/T017 (init no longer needs it) and before/with T020 (superset
  test replacement, which reads the scaffold).
- T023 (VERSION bump) pairs with T022 (migration TO version).
- Phase 6 docs after the code they describe is settled.

## Acceptance

### Functional Completeness

- [x] A-001 R1: The fab-go `cmd/fab` and `internal/spawn` test packages isolate `$HOME` and pass
  with a real `~/.fab-kit/config.yaml` present.
- [x] A-002 R2: `fab config upgrade` exists, rewrites config.yaml over the registry (A verbatim, C
  fenced, unknowns parked, renames carried), and writes via `atomicfile`. *(Met after rework cycle 1:
  below-fence content is now HOISTED above the regenerated fence and classified like any other live
  key — never dropped. Regression + full-document golden cover the exact `branch_prefix` scenario the
  review confirmed.)*
- [x] A-003 R3: `fab_version` reads resolve from `fab/.fab-version` with a config.yaml fallback in
  both reader stacks; `setFabVersion`/`topLevelFabVersionValue` are deleted; the registry and
  configscope no longer carry `fab_version`.
- [x] A-004 R4: `fab upgrade-repo` auto-runs `fab config upgrade` after sync and fails open when the
  subcommand is absent.
- [x] A-005 R5: The scaffold config.yaml is deleted; `fab config init --project` generates from the
  registry; `fab init` shells out with an embedded-stub fallback; `agent.tiers` is not pinned live.
- [x] A-006 R6: `src/kit/migrations/2.14.0-to-2.15.0.md` moves `fab_version` to `fab/.fab-version`,
  deletes the key, is sentinel-guarded and idempotent; `src/kit/VERSION` is `2.15.0`.
- [x] A-007 R8: `_cli-fab.md`, `docs/specs/config.md`, `fab-setup.md` + `SPEC-fab-setup.md`,
  `SPEC-_cli-fab.md` reflect the landed design; the sibling/mirror sweep is complete.

### Behavioral Correctness

- [x] A-008 R2.1: A user comment on a live A field is preserved byte-for-byte across `fab config
  upgrade`.
- [x] A-009 R2.2: The fence omits already-overridden fields and comments parent keys fully.
- [x] A-010 R2.3: An unknown live key is parked once below the fence and not regenerated on re-run.
- [x] A-011 R2.5: `fab config upgrade` is byte-stable/idempotent (golden + idempotence tests pass).
  *(Met after rework cycle 1: `golden_test.go` now pins the COMPLETE document with literal `got !=
  want` over a small synthetic field set, per the `memoryindex/golden_test.go` precedent; the
  idempotence/freeze tests remain.)*
- [x] A-012 R3.3: `.fab-version`-absent + config.yaml `fab_version:` present ⇒ readers fall back
  correctly (no error).
- [x] A-013 R4.2 / R5.4: The predates-subcommand paths fail open (upgrade continues; init writes a
  stub) — both tested.

### Removal Verification

- [x] A-014 R3.2 / R5.1 / R7.2: `setFabVersion`, `topLevelFabVersionValue`, the scaffold config.yaml,
  and `TestConfigReferenceSupersetsScaffoldKeys` are gone; no dangling references remain (grep clean
  in `src/go/`, and no test reads the deleted scaffold).

### Scenario Coverage

- [x] A-015 R6.2: Re-running the `2.14.0-to-2.15.0` migration is a complete no-op (sentinel trips).
- [x] A-016 R5.5: A fresh `fab init` is still classified "new project" by `scaffoldDirectories`
  against the generated config.yaml.

### Edge Cases & Error Handling

- [x] A-017 R2.2: A legacy config.yaml with no fence gets one appended at the bottom.
- [x] A-018 R5.2: `fab config init --project` refuses to overwrite an existing config.yaml; bare
  `fab config init` is a usage error.

### Code Quality

- [x] A-019 Pattern consistency: the fence engine follows the `internal/memoryindex` byte-stable
  render discipline and writes via `internal/atomicfile`; the `.fab-version` stamp mirrors
  `stampMigrationVersion`; no magic strings (fence anchors are named constants).
- [x] A-020 No unnecessary duplication: the fence renderer is shared between `fab config upgrade` and
  `fab config init --project`; the schema is read only from `configref.Fields()` (no second copy).
- [x] A-021 Canonical source only: no edits under `.claude/skills/`; kit changes are in `src/kit/`.
- [x] A-022 Migrations for user-data restructuring: the fab_version relocation ships as a
  `src/kit/migrations/` file, not an ad-hoc script.
- [x] A-023 Go changes ship tests: every `.go` change in this plan has accompanying test coverage.

### documentation_accuracy

- [x] A-024 `_cli-fab.md`, `docs/specs/config.md`, and the SPEC mirrors accurately describe the
  landed `fab config upgrade` / `init --project` behavior and the `.fab-version` relocation; no
  stale claim that `fab_version` lives in config.yaml or that a scaffold drives config creation
  remains in `src/kit/skills/` or `docs/specs/` (historical records exempt: `src/kit/migrations/*`
  migration files AND `docs/specs/findings/*` transcripts, both of which document prior states as
  they were and are not swept).

### cross_references

- [x] A-025 The `src/kit/skills/*.md` ↔ `docs/specs/skills/SPEC-*.md` mirror class is in sync
  (fab-setup ↔ SPEC-fab-setup; _cli-fab ↔ SPEC-_cli-fab); `docs/specs/config.md` cross-references
  are consistent with the landed registry/cascade/upgrade design.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Migration TO version and VERSION bump = `2.15.0` (`2.14.0-to-2.15.0.md`) | Installed VERSION is 2.14.0; fence worked example stamps `kit 2.15.0`; feature+restructure ⇒ minor bump. Intake assumption 11 | S:80 R:80 A:85 D:85 |
| 2 | Confident | New leaf package `internal/configupgrade` owns the fence/parking engine; `cmd/fab/config.go` stays thin | configref is a pure renderer with no file I/O; memoryindex precedent owns its own rendering pkg; atomicfile is the write helper. Intake assumption 14 | S:70 R:75 A:85 D:70 |
| 3 | Confident | HOME isolation lands as a package-level `TestMain` in `cmd/fab` and `internal/spawn` (the two vulnerable provider/agent-resolving packages); `internal/config` already isolates | Survey confirmed no existing TestMain in either; only providers/agent are ScopeBoth so only these packages are perturbable; TestMain is the lowest-churn suite-wide seam. Intake assumption 12 | S:70 R:90 A:85 D:70 |
| 4 | Confident | `fab config init` gains `--project` (mutually exclusive with `--system`) taking `--name/--description/--source-path(repeatable)/--test-path(repeatable)` | Plan doc names the shell-out + registry init/seed metadata; grammar guided by the `init --system` precedent; low blast radius. Intake assumption 10 | S:60 R:80 A:75 D:60 |
| 5 | Confident | The fence anchor lines are `# >>> fab reference (kit X.Y.Z) >>> ` and `# <<< end fab reference <<< ` dash-padded to a fixed width, matching the worked example | Worked example in config-upgrade.md § The fence shows this exact form | S:75 R:70 A:85 D:75 |
| 6 | Confident | `TestConfigReferenceSupersetsScaffoldKeys` is replaced by a registry-anchored guard (init-seeded/stub keys ⊆ registry keys) rather than deleted outright | Plan doc: "its skill-consumed-key guard needs a new anchor once the file is gone". Intake assumption 13 | S:55 R:90 A:75 D:60 |
| 7 | Confident | Both reader stacks read `.fab-version` FIRST, config.yaml `fab_version:` as fallback (one compat window) | Intake assumption 6; the migration moves the value so post-migration `.fab-version` is authoritative | S:80 R:75 A:85 D:80 |
| 8 | Confident | fab-kit's version-stamp helper is a new `stampFabVersion(repoRoot, version)` shaped like `stampMigrationVersion`, called from `Init` and `Upgrade` | Intake assumption 9; preserves the single-writer invariant for config.yaml | S:70 R:75 A:85 D:75 |
| 9 | Confident | Init generation runs in `Init` BEFORE the scaffold walk (as `setFabVersion` did), so `scaffoldDirectories`' config.yaml-presence classification still sees a fresh repo as new | Verified `Init` calls `setFabVersion` at line 41, `stampMigrationVersion` at 50, `runSync` at 57; keeping generation at ~41 preserves ordering | S:75 R:70 A:85 D:75 |
| 10 | Confident | The `fab config init --project` generator and `fab config upgrade` share one fence-render helper (in `internal/configupgrade`) to avoid a second fence copy | Code-quality: no duplication; the fence must be byte-identical from both entry points | S:65 R:75 A:80 D:70 |

10 assumptions (1 certain, 9 confident, 0 tentative).

## Deletion Candidates

*(Re-review, rework cycle 1. Cycle-1 candidates `commentOutSegment` duplicate / `bHygieneReport` stub /
unused `version` param were all resolved in the rework — recorded in the task rework notes above.)*

- `src/go/fab/cmd/fab/config.go:278` `defaultSubtree`, `:309` `normalizeToGeneric`, `:409`
  `asGenericMap` — now shadowed by near-identical copies in `internal/configupgrade`
  (`defaultSubtreeFor`/`normalizeToGeneric`/`asGenericMap`, added in rework cycle 1 for
  `bHygieneReport`). Single-source per the `CommentOutSegment` precedent: export from
  `configupgrade` (the importable leaf) and delete the `cmd/fab` copies.
- `configref.Field.InitSeed` + `configref.InitSeedKeys()` — production symbols whose only consumer is
  `cmd/fab/config_test.go` (`TestConfigInitSeedKeysSubsetOfRegistry`); the `--project` generator takes
  seeds as flags and does not read the metadata. Wire the generator to it, or accept it as a
  test-anchored guard and re-word the "consumed by the init generator" claims (R5.6,
  `docs/specs/config.md` init-seed row, the `Field.InitSeed` doc comment).
- Compat-window fallbacks (deletion candidates for a FUTURE release, after the `2.14.0-to-2.15.0`
  migration horizon): fab-go `internal/config` `Config.FabVersion` yaml-tag parse of a config.yaml
  `fab_version:` key, the `pruneProjectScoped` `fab_version` strip line (`config.go:327`), and step 2
  of fab-kit `readFabVersion` (`internal/config.go`) — all documented one-compat-window code.
