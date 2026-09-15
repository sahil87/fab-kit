# Intake: FAB_KIT_PATH — session-scoped kit-resolution override for kit development

**Change**: 260808-j9rb-kit-path-override
**Created**: 2026-08-08

## Origin

> Implement Change 6 (FAB_KIT_PATH: session-scoped kit-resolution override for kit development) of `fab/plans/sahil/26-08-08-config-overhaul.md` — read that full section plus the Orchestration and Resolved Decisions sections for scope, decisions, obligations, and rejected alternatives. This change is edge-free and independent of every other change in the plan per the dependency map — it touches a different seam (the shim + kit-path resolution) and mostly a different binary, with only trivial adjacency in `_cli-fab.md` and doctor output. Changes 1, 2, and 3 are running in parallel in sibling worktrees off the same base commit; proceed independently of them.

One-shot `/fab-new` invocation against a fully-resolved plan document (all decisions user-confirmed 2026-08-08 in a `/fab-discuss` session; Change 6 folded in from backlog `[kpth]` the same day). Backlog anchor for the whole plan: `[x3cf]`. The pipeline is directed to continue autonomously through `/fab-fff` after intake.

## Why

**The pain**: today *nothing* can make a kit-dev worktree run its own unshipped `src/kit/`:

1. `fab/.fab-version` correctly pins the last released kit — it is a data-migration stamp, and bumping it early would lie.
2. The shim consults `~/.fab-kit/local-versions/{v}` only for the *pinned* version — a `just install`-populated 2.17.3 is invisible while the stamp says 2.17.2.
3. The shim binary carries zero `FAB_*` env overrides (verified via `strings` 2026-08-08, and re-verified in this worktree: the only `os.Getenv` calls in `src/go/fab-kit` are `SHELL` in doctor.go and `HOME` in cache.go; `FAB_KIT_PATH` appears nowhere in `src/` or `docs/`).

**The consequence**: `fab sync` perpetually redeploys the released kit over `.claude/skills/`, and `$(fab kit-path)` serves stale templates / `reference/fkf.md` / migrations — agents working *in the fab-kit dev repo itself* exercise one-release-behind skills. Live evidence: change 260808-s2sz burned a full review cycle + a failed T007 + a user decision on exactly this (acceptance A-014 "deployed copies match sources" was unsatisfiable and had to be relaxed to sync-SOURCE equality).

**Why this approach**: honor a per-process `FAB_KIT_PATH=<dir>` at the kit-resolution seam in both binaries, so `sync`, `kit-path`, templates, reference docs, and migrations ALL follow one override. Rejected alternatives (plan-recorded, user-confirmed):

- **`fab sync --source` flag** — a sync-only override recreates the score-binary/source version-skew disease for every other kit reader.
- **In-repo autodetect** — would make dev-repo behavior diverge invisibly from user repos.
- **A persistent `kit.path` config setting** — a committed field breaks hermeticity for teammates; a machine-wide setting recreates the stale-kit disease this change diagnoses; and the shim resolves the kit *before* any config cascade exists. **Env-only is the guardrail.**

This is a sibling of the plan's Change 1 (env override layer) in philosophy but deliberately NOT part of its mechanism: Change 1's generic mapping walks *registry rows*; the kit path is *not a config field* and must never become one (plan Resolved Decision 14). It is a special-cased variable at a different seam, landed in both binaries.

## What Changes

### 1. fab-go: `internal/kitpath.KitDir()` honors `FAB_KIT_PATH`

`src/go/fab/internal/kitpath/kitpath.go` — `KitDir()` currently resolves `kit/` as an exe-sibling (after `EvalSymlinks`), with a test-only in-process `SetOverride` seam. Add the env check ahead of exe resolution:

- If `FAB_KIT_PATH` is set (non-empty): absolutize it (`filepath.Abs`), `os.Stat` it, and return it if it is an existing directory.
- If set but missing/not a directory: return a **loud error naming the variable** (e.g. `FAB_KIT_PATH is set but <dir> is not a directory`) — never fall back silently to the exe-sibling kit (guardrail: a lingering shell export must never silently mix kits).
- Precedence vs the existing test seam: keep `overrideDir` (SetOverride) winning over the env var, so existing tests stay hermetic even if the developer's shell exports `FAB_KIT_PATH`. (Alternatively neutralize the env in tests — implementer's choice, but tests must not be broken by an ambient export.)

This one choke point covers every fab-go kit reader: the `fab kit-path` command (kitpath.go), `.status.yaml` template reads (`internal/change/change.go:68`), preflight's sync-staleness compare of `$(kit)/VERSION` vs `fab/.fab-version` (`internal/preflight/preflight.go:137`), and `fab fab-help` (`cmd/fab/fabhelp.go:77`). All follow the override with no per-site work.

### 2. fab-kit: kit-directory reads honor `FAB_KIT_PATH`

`src/go/fab-kit/internal/cache.go` — `CachedKitDir(version)` currently returns local-cache kit (preferred) or remote-cache kit. The override must win here for the **reader** paths:

- **`Sync()`** (`internal/sync.go:81`) — the primary target: scaffolding, scaffold tree-walk, and `deploySkills` all read from the resolved kit dir, so under override they deploy the worktree's `src/kit/` content.
- **`fab migrations-status`** (`cmd/fab-kit/migrations_status.go:65`) — migration discovery follows the override, so unshipped migrations are visible in the dev repo.

Because the current `CachedKitDir` signature has no error return and the override must fail loud on an invalid dir, implement as an override-aware resolution (e.g. a `ResolveKitDir(version) (string, error)`-shaped seam or equivalent) that the reader call sites adopt; exact shape is an implementation decision — the contract is: env set + valid dir ⇒ that dir; env set + invalid ⇒ loud error; env unset ⇒ current local-then-remote cache behavior, byte-identical.

Under an active override, `Sync()` **skips the cache-resolution step** (`EnsureCached(fabVersion)` and the version guard) — the kit is served from the override dir, so no download should be triggered and the guard's premise (cache content == pinned version) is void. Prerequisites check, scaffolding, skill deployment, direnv allow, and project sync scripts run unchanged (sourced from the override).

### 3. Version-stamping lifecycle commands refuse under override

`fab init` and `fab upgrade-repo` both stamp `fab/.fab-version` and populate/read the version cache as their core semantic. Running them against arbitrary override content would either stamp a version that does not describe the deployed content (silent kit mixing — exactly what guardrail 1 forbids) or silently ignore the session's declared override. Neither is acceptable, so both commands **error out early when `FAB_KIT_PATH` is set**, with a clear message to unset it first (e.g. `FAB_KIT_PATH is set — unset it before running 'fab upgrade-repo' (the override must not influence version stamping)`). `fab update` (brew binary update) and `fab doctor` are unaffected as commands; sync and migrations-status follow the override as readers.

### 4. Provenance — mandatory, both surfaces (plan guardrail 1)

- **`fab sync` output**: when the override is active, print `kit: <dir> (FAB_KIT_PATH override)` (replacing/augmenting the current `Resolving kit v%s from cache...` line, which is skipped along with cache resolution).
- **`fab doctor` output**: when `FAB_KIT_PATH` is set, print the same `kit: <dir> (FAB_KIT_PATH override)` line. Doctor runs pre-config (no repo/pinned-version context), so the line appears **only when the override is active** and is informational — not one of the 7 pass/fail prerequisite checks, and it does not change the exit-code = failure-count contract.

### 5. Set-once ergonomics: the dev repo's own `.envrc` (plan guardrail 3)

Add `export FAB_KIT_PATH=$PWD/src/kit` to the fab-kit repo's own tracked `.envrc` (currently: `IDEAS_FILE`, `WORKTREE_INIT_SCRIPT`). Since `.envrc` is committed, every worktree's direnv points `FAB_KIT_PATH` at *that worktree's* `src/kit/` — set once, correct per worktree.

**Explicitly NOT** touched: `src/kit/scaffold/fragment-.envrc` (the scaffold shipped to user repos — user repos have no `src/kit/` and must see zero behavior change). This does not contradict Change 1's direnv rejection: that decision declined to document direnv as the *end-user pattern* for worker selection; here `.envrc` is the fab-kit dev repo's own tooling.

Known edge (documented, not engineered around): `wt`'s `WORKTREE_INIT_SCRIPT="fab sync"` may run before direnv has allowed/loaded the new worktree's `.envrc`, so a worktree's *first* sync can deploy the cache kit; the next `fab sync` from a direnv-loaded shell repairs it (sync is idempotent).

### 6. Docs & specs (obligations from the plan)

- `src/kit/skills/_cli-fab.md`: § fab sync + § fab kit-path gain the `FAB_KIT_PATH` env-override note (contract: per-process, both binaries, fail-loud on invalid dir, provenance line); § fab doctor gains the conditional provenance line. + `docs/specs/skills/SPEC-_cli-fab.md` mirror (constitution-required).
- `docs/specs/config.md`: an env-override note recording that `FAB_KIT_PATH` is deliberately **not** a registry field (decision 14) — kit resolution happens before the config cascade exists; env-only by design.
- Memory `docs/memory/distribution/kit-architecture.md`: the kit-resolution sections (fab-go kitpath, fab-kit cache/sync pipeline) gain the override contract.
- Memory `docs/memory/distribution/distribution.md`: sync/doctor/upgrade-repo behavior descriptions updated where they restate the affected output/refusal contracts.
- `fab/plans/sahil/26-08-08-config-overhaul.md` is a plan artifact — leave it untouched (its Change 6 section remains the historical design record).

### 7. Tests

- `kitpath_test.go`: override honored, absolutized, invalid-dir loud error, unset ⇒ unchanged exe-sibling behavior, SetOverride-vs-env precedence.
- fab-kit tests (`cache_test.go` / `sync_test.go` / doctor test): override-aware kit resolution (valid/invalid/unset), sync provenance line + skipped cache resolution under override, doctor provenance line only-when-set, init/upgrade-repo refusal.
- Guard tests against ambient `FAB_KIT_PATH` in the developer's environment (`t.Setenv` neutralization) so the suite is hermetic.

### Non-goals

- **Binary resolution unchanged**: `FAB_KIT_PATH` governs kit *content* only; the shim's fab-go binary resolution (pinned version, local-versions-first) is untouched. `just install` + local-versions remains the binary-dev path.
- **No config field, no registry row, no migration** (env-only, opt-in, no persisted state).
- **No router (shim `cmd/fab/main.go`) code change**: it execs sub-binaries with `os.Environ()`, so the variable propagates as-is (verified in source).
- **No stamp or cache mutation under override** (hermeticity guardrail 2) — enforced by the § 3 refusals.

## Affected Memory

- `distribution/kit-architecture`: (modify) kit-resolution contract — fab-go `internal/kitpath` env override, fab-kit cache/sync kit-dir resolution, sync-pipeline provenance + skipped cache step, init/upgrade-repo refusal
- `distribution/distribution`: (modify) sync/doctor/upgrade-repo surface descriptions where they restate output and refusal contracts (doctor command row, upgrade-repo behavior)
- `distribution/setup`: (verify — likely no change) doctor is described as the Phase-0 gate only; the added line is informational and does not alter the gate contract

## Impact

- **Go — fab-go binary**: `src/go/fab/internal/kitpath/kitpath.go` (+ test) — the single choke point; no call-site changes expected.
- **Go — fab-kit binary**: `src/go/fab-kit/internal/cache.go` (kit-dir resolution), `internal/sync.go` (provenance + skip cache step), `internal/init.go` + `internal/upgrade.go` (refusal guards), `cmd/fab-kit/doctor.go` (provenance line), + tests for each.
- **Dev-repo tooling**: `.envrc` (one export line).
- **Kit content**: `src/kit/skills/_cli-fab.md` (never `.claude/skills/` — deployed copies).
- **Specs**: `docs/specs/skills/SPEC-_cli-fab.md`, `docs/specs/config.md`.
- **Memory**: per Affected Memory.
- **Parallel-work adjacency**: Changes 1/2/3 run in sibling worktrees; the only shared surfaces are `_cli-fab.md` + SPEC mirror and `docs/specs/config.md`, in *different sections* — mechanical rebase only, per the plan's conflict-surface table. No semantic coupling; proceed independently.

## Open Questions

*(none — the plan section, orchestration map, and resolved-decisions list answer every scoping question; remaining choices are graded assumptions below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Honor the override at the two existing choke points: fab-go `kitpath.KitDir()` and fab-kit's kit-dir resolution (`CachedKitDir` seam) — no per-reader flags | Plan-specified ("the shim and the fab binary, wherever the kit directory is resolved"); code-verified: these two functions cover every kit read in both binaries | S:90 R:85 A:90 D:90 |
| 2 | Certain | Env-only — never a config field, registry row, or persisted state; no migration ships | Plan Resolved Decision 14, verbatim; hermeticity is the stated guardrail | S:95 R:90 A:95 D:95 |
| 3 | Certain | Provenance line format `kit: <dir> (FAB_KIT_PATH override)` on sync and doctor output | Plan guardrail 1 gives the literal string | S:90 R:85 A:90 D:90 |
| 4 | Certain | Dev-repo `.envrc` gains `export FAB_KIT_PATH=$PWD/src/kit`; scaffold `fragment-.envrc` untouched | Plan guardrail 3 names the dev repo's own existing `.envrc`; user repos must see zero change | S:80 R:90 A:85 D:80 |
| 5 | Certain | No router/shim `main.go` change — env propagates through `syscall.Exec(bin, argv, os.Environ())` | Verified in `src/go/fab-kit/cmd/fab/main.go` (both exec paths pass `os.Environ()`) | S:85 R:90 A:95 D:90 |
| 6 | Confident | Invalid override (missing / not a directory) ⇒ loud error naming `FAB_KIT_PATH`, never a silent fallback to the cache | Guardrail 1 ("can never silently mix kits") + the repo's fail-loud exit-contract convention | S:70 R:80 A:80 D:75 |
| 7 | Confident | `fab init` and `fab upgrade-repo` refuse to run while `FAB_KIT_PATH` is set (clear unset-first error); readers (sync, migrations-status, all fab-go reads) follow the override | Plan's "ALL follow" list names only readers; guardrail 2 forbids stamp mutation under override — stamping a release version while deploying override content would silently mix kits | S:55 R:75 A:70 D:60 |
| 8 | Confident | Under override, `Sync()` skips `EnsureCached` + the version guard (no download, guard premise void); all other sync steps unchanged, sourced from the override dir | Follows from serving kit content from the override; avoids surprise network fetches; trivially reversible if review prefers keeping the guard | S:50 R:80 A:70 D:65 |
| 9 | Confident | Doctor prints the provenance line only when the override is active, as an informational line outside the 7-check pass/fail + exit-code contract | Doctor runs pre-config with no pinned-version context, so an unconditional `kit:` line is unresolvable there; guardrail 1 only requires override visibility | S:55 R:85 A:75 D:65 |
| 10 | Confident | Relative `FAB_KIT_PATH` values are absolutized (`filepath.Abs`) at read time; validation requires an existing directory | A CWD-relative kit path would resolve differently per invocation dir — absolutizing keeps one process-wide meaning; `.envrc` exports an absolute path anyway | S:40 R:85 A:65 D:55 |
| 11 | Confident | The worktree-creation first-sync edge (init script may run before direnv loads `.envrc`) is documented, not engineered around — a re-run of `fab sync` repairs | Sync is idempotent (Constitution III); engineering wt/direnv ordering is out of scope and cross-tool | S:45 R:80 A:60 D:55 |
| 12 | Confident | Binary resolution stays untouched — `FAB_KIT_PATH` governs kit content only; `local-versions` remains the binary-dev mechanism | Plan scope list ("sync, kit-path, templates, reference docs, and migrations") names content readers only; binary override is a different problem with an existing mechanism | S:70 R:75 A:80 D:75 |

12 assumptions (5 certain, 7 confident, 0 tentative, 0 unresolved).
