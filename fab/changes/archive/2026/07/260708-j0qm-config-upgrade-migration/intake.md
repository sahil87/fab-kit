# Intake: Config Upgrade Command & fab_version Migration

**Change**: 260708-j0qm-config-upgrade-migration
**Created**: 2026-07-08

## Origin

One-shot `/fab-new` invocation. This is **Change 3 of the config-upgrade effort** — the design
authority is `fab/plans/sahil/config-upgrade.md` (all six decisions user-confirmed in the
2026-07-08 `/fab-discuss` session) plus `docs/specs/config.md` § Forward-looking intent (Change 3)
(`docs/specs/config.md:242-263`). Changes 1 (260708-ff2v, registry + `--json`) and 2 (260708-lpb5,
cascade + visibility commands) are landed at review-pr:done. The stopgap for the triggering bug
shipped separately (260708-yogn, PR #473 — `setFabVersion` line splice).

> Implement fab config upgrade plus the config.yaml migration (the risky one — user-data
> restructure). See fab/plans/sahil/config-upgrade.md Change 3 for full context, all decisions
> resolved there including today folded scope: 1) The upgrader keeps live fields verbatim, parks
> unknown fields in a "# removed in X.Y.Z, your value was:" comment block (never silently
> deleted), carries renames via renamed_from, regenerates the managed fence byte-stable and
> idempotent (same discipline as fab memory-index). 2) Move fab_version out of config.yaml to
> fab/.fab-version (one-line plain text, committed, sibling to .kit-migration-version); delete
> setFabVersion; ship the migration; after this fab config upgrade is the only writer of
> config.yaml ever. 3) fab upgrade-repo auto-runs the upgrader after sync; if the installed fab
> predates the subcommand, fail-open with a printed reminder. 4) Scaffold config.yaml is deleted
> (folded in today) — init generates from the registry instead: fab init writes the initial
> config.yaml from the registry (shells out to installed fab-go); when the installed fab predates
> fab config init, fall back to a minimal embedded stub config.yaml (NOT a printed instruction —
> user-confirmed decision, a fresh repo must never fail preflight for lack of a config.yaml). The
> registry gains init/seed metadata for which fields are written live at init (project.*,
> source_paths, test_paths) versus fence territory. Fold the scaffold's extra prose into registry
> descriptions and retire/replace TestConfigReferenceSupersetsScaffoldKeys. 5) Also address the
> HOME-hermeticity follow-up inherited from Change 2's review (only 3 test files isolate $HOME; a
> real ~/.fab-kit/config.yaml with a both-scoped override can break exact-output tests in
> resolve_agent, preflight, status, impact, spawn, prmeta) before this change starts writing real
> system configs in tests. Obligations: migration file per constitution user-data-restructure
> rule, _cli-fab.md update, tests.

## Why

1. **The pain point**: `config.yaml` upgrades are not mechanical. Comments are hand-maintained
   user state that every whole-file writer destroys — `setFabVersion` (fab-kit
   `src/go/fab-kit/internal/init.go:99-130`) mashed comments/order/indentation on every
   `fab init`/`fab upgrade-repo` until PR #473's line-splice stopgap, and the comment-backfill
   migration pattern (e.g. `2.13.1-to-2.13.2`) exists only to repair that damage after the fact.
   New/renamed/removed config fields today each need a hand-written migration.
2. **The consequence of not fixing**: the masher bug class recurs (observed 2026-07-03 and
   2026-07-08); users lose the advertised-override scaffolding permanently (migrations never
   re-apply once `.kit-migration-version` passes them); every future field rename/removal costs
   another bespoke migration.
3. **Why this approach**: Changes 1+2 made the registry (`src/go/fab/internal/configref/`,
   ordered `[]Field` with `default`/`description`/`scope`/`advertise`/`renamed_from`) the single
   schema source and landed the three-layer cascade. Change 3 completes the design: comments
   become **regenerable generated output** inside a managed fence, `fab config upgrade` becomes
   the **only writer of config.yaml ever** (single-writer by construction — `setFabVersion` is
   deleted, `fab_version` leaves the file), and renames/removals become mechanical registry data
   (`renamed_from`, parked-removals block) instead of hand-written migrations.

## What Changes

### 1. New subcommand: `fab config upgrade` (fab-go)

New cobra subcommand under `configCmd()` (`src/go/fab/cmd/fab/config.go:24-39`), implemented over
the registry (`configref.Fields()`, `src/go/fab/internal/configref/configref.go:258-399`) and the
cascade scope/advertise metadata. It rewrites `fab/project/config.yaml` under the field-category
model (`docs/specs/config.md:126-143`):

- **A) live (user-overridden) fields — kept verbatim, byte-for-byte**, including the user's own
  comments on them. Presence = intent (decision 2): a live field is an override even when its
  value equals the default; never auto-removed. B-hygiene ("these fields equal current defaults —
  remove?") is an advisory report line only.
- **C) advertise-flagged fields not currently overridden** — regenerated as a commented scaffold
  inside the **managed fence**, per the worked example in `fab/plans/sahil/config-upgrade.md`
  § The fence:
  - Byte-exact splice anchors: `# >>> fab reference (kit X.Y.Z) >>> ---…` /
    `# <<< end fab reference <<< ---…`. Upgrade rewrites ONLY between the markers; everything
    outside is the user's. No fence in a legacy file → append one at the bottom.
  - The kit-version stamp in the BEGIN line makes staleness visible (enables a *later* `--check`
    drift mode — NOT in this change).
  - Every scaffolded block is **fully commented including parent keys** (a live `agent:` key over
    comment-only children is exactly what the old masher collapsed to `agent: null`).
  - The fence **omits fields already overridden above it** — it shows what you *could* override
    but haven't.
- **Unknown fields** (live keys no longer in the registry) — parked in a
  `# removed in X.Y.Z (parked by fab config upgrade — delete when done):` comment block **below**
  the fence, with the user's value serialized in the comment. Never silently deleted. Parkings
  are user territory: appended exactly once, never regenerated away.
- **Renames** — a live field matching some registry row's `renamed_from` is carried to the new
  key mechanically (value verbatim), replacing the per-rename hand-written-migration pattern.
- **Output discipline**: byte-stable and idempotent — running `fab config upgrade` twice yields
  byte-identical output (`fab memory-index` precedent: pure renders, golden + idempotence tests —
  `src/go/fab/internal/memoryindex/`, `golden_test.go`, `freeze_test.go:47`). Note the
  `>>>`/`<<<` splice-anchor pattern itself is NEW code (memoryindex is a whole-file owner, not a
  fence splicer); use `internal/atomicfile.WriteFile` (`atomicfile.go:20`) for the write.

### 2. `fab_version` moves out of config.yaml → `fab/.fab-version`; `setFabVersion` deleted

- **New file** `fab/.fab-version` (decision 1): one-line plain text (bare semver + `\n`),
  committed, sibling to `fab/.kit-migration-version` — exact precedent
  `stampMigrationVersion(repoRoot, version)` (`src/go/fab-kit/internal/init.go:69-78`). Kept
  separate from `.kit-migration-version`: deployed-kit version vs migration baseline diverge
  exactly when migrations are pending, which is when both are needed distinctly.
- **Delete `setFabVersion`** (`src/go/fab-kit/internal/init.go:99-130`, incl.
  `topLevelFabVersionValue` `:137-150`); its two callers (`Init` `init.go:41`, `Upgrade`
  `upgrade.go:120`) stamp `fab/.fab-version` instead. After this, `fab config upgrade` is the
  only writer of config.yaml, ever.
- **Reader updates — two independent reader stacks, both move to `.fab-version` with a
  config.yaml `fab_version:` fallback for one compat window**:
  - fab-kit: its own `readFabVersion` (`src/go/fab-kit/internal/config.go:88-105`) feeding
    `ConfigResult.FabVersion` — consumed by the router (`cmd/fab/main.go:46,81,123` — pinned
    version resolution), `internal/sync.go:47` (version guard), `internal/upgrade.go:32-49`
    (`ResolveConfig`), and `cmd/fab-kit/migrations_status.go:65`.
  - fab-go: `internal/config` `Config.FabVersion` (`config.go:122`) / `GetFabVersion()`
    (`:367-372`) — sole consumer `internal/preflight/preflight.go:148-155` (staleness check).
- **Registry/scope cleanup**: remove the `fab_version` row (`configref.go:383-392`) and its
  `internal/configscope` entry; `fab config reference` output and its verbatim-block tests
  update accordingly.
- The migration (§5) moves the existing value and deletes the key from user configs.

### 3. `fab upgrade-repo` auto-runs the upgrader (fab-kit)

In `Upgrade` (`src/go/fab-kit/internal/upgrade.go:29-171`), after `runSync` (`:115`) and version
stamping (now `.fab-version`), shell out to the **installed** fab-go binary's `fab config upgrade`
(decision 4) — `EnsureCached(version)` already returns the pinned fab-go binary path (the router's
`execFabGo` precedent, `cmd/fab/main.go:72-95`; `exec.Command` precedent at `main.go:128`). Both
binaries ship in one brew package, so binary/kit skew occurs only on explicit-version upgrades
(acceptable; the fence stamp shows which kit rendered it). **Fail-open**: if the installed fab
predates the subcommand (non-zero exit / unknown command), print a reminder and continue — an
upgrade must never break on the config step.

### 4. Scaffold config.yaml deleted — init generates from the registry

`src/kit/scaffold/fab/project/config.yaml` (90-line template with `{PROJECT_NAME}`/
`{SOURCE_PATHS}`/`{TEST_PATHS}` placeholders) is the last hand-maintained copy of defaults/comment
prose; value/prose drift is unguarded (`TestConfigReferenceSupersetsScaffoldKeys`,
`src/go/fab/cmd/fab/config_test.go:113-130`, checks key *presence* only). Delete it — nothing in
it is irreducibly scaffold-only:

- **`fab init` (fab-kit) generates the initial config.yaml** by shelling out to the installed
  fab-go (`fab config init` project mode — the same one-brew-package skew + fail-open discipline
  as §3). The scaffold tree walk (`src/go/fab-kit/internal/scaffold.go:100-162`, copy-if-absent)
  simply no longer carries a config.yaml; generation is an explicit init step. Existing repos are
  untouched (copy-if-absent semantics preserved: never overwrite an existing config.yaml; note
  `scaffoldDirectories:77` uses config.yaml presence for new-vs-existing classification — keep
  that behavior against the generated file).
- **Fallback when the installed fab predates `fab config init` project mode**: write a minimal
  **embedded stub config.yaml** (a tiny, bounded second copy of the A-class identity fields) —
  NOT a printed instruction. User-confirmed: a fresh repo must never fail preflight for lack of a
  config.yaml.
- **The registry gains init/seed metadata**: which fields are written live at init (the A-class
  identity fields: `project.*`, `source_paths`, `test_paths`) and their value slots. fab-kit's
  detection of project name / source paths / test paths (today template substitution of
  `{PROJECT_NAME}`/`{SOURCE_PATHS}`/`{TEST_PATHS}`; `/fab-setup`'s on-disk `test_paths` detection,
  5qf5) becomes generator **input** (passed to `fab config init`). Everything else is fence
  territory from day one.
- **The scaffold's live `agent.tiers` pinning dies with it**: under presence=intent, tiers pinned
  at init are an accidental override that stops tracking fab-kit's defaults. Fresh projects
  inherit; the fence advertises.
- **Fold the scaffold's extra prose** (multi-language `test_paths` examples, providers narrative)
  into registry descriptions/segments where not already present.
- **Retire/replace `TestConfigReferenceSupersetsScaffoldKeys`**: its skill-consumed-key guard
  needs a new anchor once the file is gone (e.g. init-seeded/stub keys ⊆ registry keys).
- **Skill updates**: `/fab-setup`'s Config Create-Mode (`src/kit/skills/fab-setup.md`) references
  the scaffold + placeholder detection — update it to the generated path (and its
  `docs/specs/skills/SPEC-fab-setup.md` mirror, per the constitution).

### 5. Migration file (constitution: user-data restructure)

New `src/kit/migrations/{release}-to-{next}.md` (naming per `DiscoverMigrations`,
`src/go/fab-kit/internal/migrations.go:36-52`; structure per the `2.13.1-to-2.13.2` precedent:
Summary / Pre-check with sentinel+idempotency gates / ordered Changes with atomic temp+rename
write / Verification):

- Move the `fab_version:` value from `fab/project/config.yaml` to new `fab/.fab-version`; delete
  the key (and its stale comment line) from config.yaml.
- Sentinel-guarded and idempotent (`.fab-version` already present + key absent ⇒ no-op).
- Historical comment-backfill migrations are left alone; the pattern is retired going forward.
- The fence itself needs no migration step — it appears on the first `fab config upgrade` run
  (auto-run by the next `upgrade-repo`, §3).

### 6. HOME hermeticity for the fab-go test suite (inherited follow-up, do FIRST)

Change 2's cascade made `internal/config.LoadPath` merge `~/.fab-kit/config.yaml` (via the
`homeDir = os.UserHomeDir` seam, `config.go:18`), but only 3 test files isolate `$HOME`
(`internal/config/config_test.go` `isolateSystemConfig:17-22`, `cmd/fab/config_test.go:25`,
`cmd/fab/config_show_init_test.go` `setupConfigRepo:16-30`). A real `~/.fab-kit/config.yaml` with
a both-scoped override breaks exact-output tests: **HIGH** `cmd/fab/resolve_agent_test.go`
(exact-bytes assertions `:43,:306,:333`), `cmd/fab/agent_test.go`; **MED**
`internal/preflight/preflight_test.go`, `internal/status/*_test.go`,
`internal/impact/impact_test.go`, `internal/spawn/spawn_test.go`,
`cmd/fab/batch_new_test.go`/`batch_switch_test.go`, prmeta tests. Neither module has a `TestMain`.
Add suite-wide `$HOME` isolation (temp-HOME via `TestMain` per affected package and/or the shared
helpers, e.g. `chdirTestEnv`, `cmd/fab/testhelpers_test.go:11`) **before** this change's own tests
start writing real system configs.

### 7. Documentation obligations

- `src/kit/skills/_cli-fab.md` § fab config (`:304-366`): add `upgrade` (and the `init` project
  mode); the section already forward-refs Change 3 at `:329`.
- `docs/specs/config.md`: flip § Forward-looking intent (Change 3) to landed, in authoritative
  detail (same treatment Changes 1/2 got).
- Skill/SPEC mirrors per the Sibling & Mirror Sweeps class (fab-setup.md ↔ SPEC-fab-setup.md; any
  other skill restating scaffold/config-creation or `fab config` subcommand facts — grep-sweep
  `scaffold`, `fab config`, `fab_version` across `src/kit/skills/` + `docs/specs/`).
- Go tests alongside every code change (constitution VII); memory updates at hydrate.

## Affected Memory

- `_shared/configuration`: (modify) `fab config upgrade` semantics (A/B/C, fence contract,
  presence=intent, parked removals, renamed_from carry), `fab_version` relocation to
  `fab/.fab-version`, single-writer invariant
- `distribution/kit-architecture`: (modify) `setFabVersion` deletion, scaffold config.yaml
  removal + init-from-registry generation (shell-out + embedded stub fallback), `.fab-version`
  sibling file
- `distribution/distribution`: (modify) `fab upgrade-repo` flow gains the post-sync auto-run of
  `fab config upgrade` (fail-open reminder path)
- `distribution/migrations`: (modify) new migration entry (fab_version extraction), note the
  comment-backfill pattern retired going forward
- `distribution/setup`: (modify) `/fab-setup` config Create-Mode now generates via
  `fab config init` instead of scaffold placeholder substitution

## Impact

- **fab-go** (`src/go/fab/`): `cmd/fab/config.go` (new `upgrade` subcommand + `init` project
  mode), new upgrader engine package (or extension of `internal/configref`), `internal/configref`
  (init/seed metadata, `fab_version` row removal, prose folding), `internal/configscope`
  (fab_version entry), `internal/config` (`.fab-version` reader + fallback),
  `internal/preflight` (staleness check source), test suites across `cmd/fab` +
  `internal/{config,preflight,status,impact,spawn}` (HOME hermeticity + new tests)
- **fab-kit** (`src/go/fab-kit/`): `internal/init.go` (delete `setFabVersion`, stamp
  `.fab-version`, config generation shell-out + stub fallback), `internal/upgrade.go` (auto-run),
  `internal/config.go` (`readFabVersion` → `.fab-version` + fallback), `internal/sync.go` /
  `cmd/fab/main.go` / `cmd/fab-kit/migrations_status.go` (reader call sites), `internal/scaffold.go`
  (config.yaml leaves the walk), tests
- **Kit content** (`src/kit/`): delete `scaffold/fab/project/config.yaml`, new
  `migrations/{release}-to-{next}.md`, `skills/_cli-fab.md`, `skills/fab-setup.md`
- **Docs**: `docs/specs/config.md`, `docs/specs/skills/SPEC-fab-setup.md` (+ mirror-sweep hits),
  memory files at hydrate
- **User-facing**: every fab-managed repo gets the fence + `.fab-version` on its next
  upgrade-repo + `/fab-setup migrations` pass — the user-data-restructure risk this change is
  named for

## Open Questions

*(none — all six effort decisions plus the two folded-scope decisions were user-confirmed
2026-07-08; see Assumptions)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab_version` → `fab/.fab-version`: one-line plain text, committed, sibling to (never merged with) `.kit-migration-version` | Decision 1, user-confirmed in plan doc | S:95 R:60 A:95 D:95 |
| 2 | Certain | Presence = intent: upgrader never auto-removes live fields; B-hygiene advisory-only report line | Decision 2, user-confirmed; spec §247-252 | S:95 R:70 A:95 D:95 |
| 3 | Certain | Fence contract: byte-exact `>>>`/`<<<` anchors w/ kit-version stamp, rewrite only between, fully-commented blocks incl. parents, omit overridden fields, parked removals below fence appended once, append fence at bottom of legacy files | Decision 3 + worked example in plan doc; spec §254-263 | S:95 R:65 A:90 D:90 |
| 4 | Certain | `fab upgrade-repo` shells out to installed fab-go `fab config upgrade` after sync; predates-subcommand ⇒ printed reminder, never a failed upgrade | Decision 4, user-confirmed | S:95 R:75 A:90 D:95 |
| 5 | Certain | Scaffold config.yaml deleted; `fab init` generates from registry via installed fab-go; embedded minimal stub fallback (not a printed instruction) when installed fab predates the subcommand | Folded scope, user-confirmed 2026-07-08 (plan doc § Change 3 last bullet) | S:95 R:60 A:85 D:90 |
| 6 | Certain | Both fab_version reader stacks (fab-kit `readFabVersion` + fab-go `config`/preflight) get a config.yaml fallback for one compat window | Stated in plan doc § Obligations | S:90 R:70 A:90 D:90 |
| 7 | Certain | Historical comment-backfill migrations left untouched; pattern retired going forward only | Plan doc explicit | S:95 R:90 A:95 D:95 |
| 8 | Certain | No `--check` drift mode in this change — the version stamp merely enables it later | Plan doc: "enables a later `--check` drift mode" | S:80 R:95 A:85 D:85 |
| 9 | Confident | `setFabVersion`'s version-stamp duty is replaced by a `stampMigrationVersion`-shaped plain-text write of `fab/.fab-version` in fab-kit, called from `Init` and `Upgrade` | Inferred: fab-kit must still record the deployed version; a plain-text sibling write preserves the single-writer invariant for config.yaml | S:70 R:70 A:85 D:75 |
| 10 | Confident | `fab config init` gains a project mode taking seed values (name/description/source_paths/test_paths) as flags from fab-kit's detection; exact flag grammar decided at apply | Plan doc names the shell-out + registry init/seed metadata; grammar itself unspecified but low-blast-radius and convention-guided (`init --system` precedent) | S:60 R:80 A:75 D:55 |
| 11 | Confident | Migration ships as `2.14.0-to-{next-release}.md` following the sentinel-guarded config-only precedent (2.13.1-to-2.13.2 shape), plus `.fab-version` creation; exact TO version fixed at release | Naming rule in `migrations.go:36-52`; installed fab is 2.14.0 | S:75 R:80 A:80 D:80 |
| 12 | Confident | HOME hermeticity lands as suite-wide temp-HOME isolation (TestMain per affected package and/or HOME injection in shared setup helpers) covering resolve_agent, agent, preflight, status, impact, spawn, prmeta, batch tests | Inherited Change 2 review follow-up; mechanism choice is apply-level (homeDir seam already honors $HOME) | S:70 R:90 A:85 D:65 |
| 13 | Confident | `TestConfigReferenceSupersetsScaffoldKeys` replaced by a registry-anchored guard (init-seeded/stub keys ⊆ registry keys) | Plan doc: "needs a new anchor once the file is gone"; exact shape decided at apply | S:55 R:90 A:75 D:60 |
| 14 | Certain | The `>>>`/`<<<` splice fence is new code (no existing fence splicer); byte-stability/idempotency discipline copied from memoryindex (golden + idempotence tests), writes via `internal/atomicfile` | Code-mapper verification: memoryindex is a whole-file owner, no marker constants exist; atomicfile is the established safe-write helper | S:80 R:80 A:90 D:80 |

14 assumptions (9 certain, 5 confident, 0 tentative, 0 unresolved).
