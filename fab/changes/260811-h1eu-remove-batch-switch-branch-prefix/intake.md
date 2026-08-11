# Intake: Remove batch switch branch_prefix — unify branch naming on the no-prefix convention

**Change**: 260811-h1eu-remove-batch-switch-branch-prefix
**Created**: 2026-08-11

## Origin

> Backlog `[h1eu]` 2026-08-06: (BUG) fab batch switch applies branch_prefix while /git-branch and naming.md say no prefix — the two branch-creating paths fork; decide and unify (spec f044)

One-shot autonomous invocation: the user asked for `[h1eu]` to be investigated, and if still valid, fixed end-to-end via `/fab-new` + `/fab-fff` without questions, recording ambiguous calls in this intake.

**Investigation (2026-08-11) confirmed the bug is still valid.** `src/go/fab/cmd/fab/batch_switch.go:101,119` computes `branchName := cfg.GetBranchPrefix() + match`, while `docs/specs/naming.md:32` and `_preamble.md` § Naming Conventions (Git Branch) both state "The branch name equals the change folder name directly. No prefix." — and `/fab-new` Step 11 / `/git-branch` create branches named exactly `{change-folder-name}`.

**Key decision reached during investigation (recorded per SRAD below): unify by REMOVING `branch_prefix`** — from `batch_switch.go` and from the config surface entirely — rather than teaching `/git-branch`/naming.md to honor it. Evidence: (a) the slim-config change (commit `930ab5e2`, Feb 2026) deliberately removed `git.branch_prefix` system-wide ("no prefix"); the Go migration of batch switch (`0f196ea1`, then `ceb20d75`) reintroduced it only there — a regression against a deliberate decision; (b) spec finding `f044` (docs/specs/findings/skills-review-2026-06-11.md:1365, verifier-confirmed at high confidence) recommends exactly this removal; (c) a prefixed branch breaks real behavior, see Why.

## Why

1. **The pain point**: `fab batch switch` attaches worktrees to *existing* changes, whose branches were created unprefixed by `/fab-new`/`/git-branch`. If a project sets `branch_prefix` (say `feature/`), batch switch probes for `feature/260811-xxxx-slug`, finds nothing, and creates a **new orphan branch** with the prefixed name instead of checking out the change's real branch. The worktree then sits on a branch with none of the change's commits. Conversely `/git-branch` (following the no-prefix convention) will never match — and per f044's verifier can even rename the prefixed branch away. The two branch-creating paths produce disjoint branch namespaces for the same change.
2. **If we don't fix it**: the config key is a loaded gun — it is advertised in every project's managed config fence (`# branch_prefix: ""`), so any user who uncomments it silently forks their branch naming and breaks worktree/branch reuse across the whole pipeline (operator branch-alignment checks, `/fab-switch`, PR flows all assume branch == folder name; see also finding f194).
3. **Why removal over honoring the prefix**: honoring it would require threading `branch_prefix` through `/git-branch`, `/fab-new` Step 11, naming.md, the operator's branch-alignment checks, and every skill that assumes `branch == change folder name` — significant complexity to support a key that was deliberately removed system-wide in Feb 2026, was undocumented for most of its life, defaults to `""`, and has no known users. The no-prefix convention is the documented, tested, spec-backed behavior; batch switch is the lone deviant.

## What Changes

### 1. `batch_switch.go` — stop applying the prefix

- `src/go/fab/cmd/fab/batch_switch.go`: delete `branchPrefix := cfg.GetBranchPrefix()` (line 101) and change `branchName := branchPrefix + match` (line 119) to use `match` directly (branch name = change folder name, matching `/git-branch` and naming.md). The `cfg` load stays only if still needed by other code in the function (currently it is used solely for the prefix — if nothing else consumes it, drop the `config.Load` call and import too).
- `src/go/fab/cmd/fab/batch_switch_test.go`: update/remove prefix-related tests; keep/extend a test asserting the branch name equals the resolved folder name.

### 2. Retire the `branch_prefix` config key entirely

Following the retired-key precedent (`review_tools`, `agent.spawn_command`):

- `src/go/fab/internal/config/config.go`: remove the `BranchPrefix` field (line 303) and the `GetBranchPrefix()` accessor (lines 783–789).
- `src/go/fab/internal/configref/configref.go`: remove the `branch_prefix` field row (lines 653–669); update the `referenceHeader` mention `(agent.profiles, stage_hooks, branch_prefix)` (line 685) to drop it.
- `src/go/fab/cmd/fab/config.go:140`: same header-text sweep in the explain-help prose.
- `src/go/fab/internal/configscope/configscope.go`: remove from the scope map (line 80) and the project-only list (line 110).
- Tests: `config_test.go` (GetBranchPrefix round-trips, type-error fixture, scope table at cmd/fab/config_test.go:1354, commented-out-in-reference assertion at line 97), `configscope_test.go` (lines 22, 63), `config_show_init_test.go:1012`. Add `branch_prefix` to `TestConfigReferenceRetiresLegacyKeys`'s retired list (`cmd/fab/config_test.go:1122–1127`).
- **Leave untouched**: `configupgrade` fixture usages (`golden_test.go`, `configupgrade_test.go`, `freeze_test.go`) — these use `branch_prefix` as a *synthetic key name in local field tables* (frozen historical snapshots and self-contained golden fixtures), not the live registry. Frozen version snapshots must not be rewritten.

### 3. Project config fence regeneration

- `fab/project/config.yaml` (this repo's own): regenerate the managed fence via `fab config upgrade` so the `# branch_prefix: ""` advert block disappears (lines 68–71 today). Downstream user projects get the same automatically at their next `fab config upgrade` — the fence is regenerated on every upgrade.

### 4. Migration for user projects

- A user who *set* `branch_prefix` live (above the fence) would be left with a silently-ignored unknown key. Ship a small migration in `src/kit/migrations/` (FROM = current release `2.19.4`, TO = next minor) that removes a live `branch_prefix:` key from `fab/project/config.yaml` if present (noting the retirement), and is a no-op otherwise. Follow the retired-key migration precedent (e.g., `review_tools` → code-review.md, `agent.spawn_command` → providers) for tone/shape — here the key has no destination; it is simply removed. <!-- assumed: migration ships despite ~zero expected users with the key set — the code-quality anti-pattern rule requires a migration for config restructuring; apply should confirm shape against the most recent retired-key migration -->

### 5. Documentation sweep (specs, skills, memory)

- `src/kit/skills/_cli-fab.md`: line 411 (full-schema key list — drop `branch_prefix`; note it in the retired-keys parenthetical alongside `review_tools`/`agent.spawn_command`) and line 1258 (`switch` names branches `{branch_prefix}{folder_name}` → branch name = folder name).
- `docs/specs/config.md`: lines 160–162 (reference block sample), 192/195 (project-scope key enumerations), 266 (advertised-keys list).
- `docs/specs/architecture.md:342` (`branch_prefix: ""` in the sample config).
- `docs/memory/distribution/kit-architecture.md`: the `fab batch switch` bullet (branch naming sentence — "using `branch_prefix` from the shared `internal/config` accessor" and the `branchName = branchPrefix + match` prose).
- `docs/memory/_shared/configuration.md`: lines 19/21 (Config-struct key list, opt-in override blocks list), 458 (consumed-key enumeration + accessor list — drop `GetBranchPrefix`, empty-branch-prefix default mention).
- Regenerate memory indexes (`fab memory-index`) after memory edits.
- `docs/specs/naming.md` needs **no change** — it stays authoritative as-is.
- Sweep per code-quality.md § Sibling & Mirror Sweeps: grep `branch_prefix` and `BranchPrefix` repo-wide at the end of apply; every remaining hit must be either a frozen fixture/snapshot (§2 above), a historical migration file (`src/kit/migrations/2.*.md` — historical, never rewritten), a findings archive (`docs/specs/findings/`), or `docs/memory/distribution/log.md`/`log.seed.md` (append-only history).

### 6. Backlog

- `fab/backlog.md` `[h1eu]` gets marked done at archive time by the normal `/fab-archive` flow — no manual edit in this change.

## Affected Memory

- `distribution/kit-architecture.md`: (modify) `fab batch switch` bullet — branch naming no longer uses `branch_prefix`; branch name = change folder name
- `_shared/configuration.md`: (modify) remove `branch_prefix` from the Config-struct key lists, opt-in override blocks, and the § single-config-reader decision's key/accessor enumerations; note the retirement

## Impact

- **Code**: `src/go/fab/cmd/fab/batch_switch.go` (+test), `src/go/fab/internal/config/config.go` (+test), `src/go/fab/internal/configref/configref.go`, `src/go/fab/internal/configscope/configscope.go` (+test), `src/go/fab/cmd/fab/config.go` (+`config_test.go`, `config_show_init_test.go`).
- **Kit**: new `src/kit/migrations/2.19.4-to-2.20.0.md` (or next-minor equivalent), `src/kit/skills/_cli-fab.md`.
- **Docs**: `docs/specs/config.md`, `docs/specs/architecture.md`, `docs/memory/distribution/kit-architecture.md`, `docs/memory/_shared/configuration.md`, regenerated memory indexes, this repo's `fab/project/config.yaml` fence.
- **Behavioral**: no behavior change for any project with `branch_prefix` unset (the universal case — default `""` means the concatenation was already a no-op). A project that set it stops getting prefixed branches from `fab batch switch` — which is the bug being fixed.
- **Tests**: Go test updates across the four packages above; constitution requires CLI changes to ship test + `_cli-fab.md` updates.

## Open Questions

*(none — autonomous run; all decisions recorded as graded assumptions below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Unify by REMOVING branch_prefix (from batch switch and the config surface) rather than documenting the prefix and teaching /git-branch to honor it | Slim-config (930ab5e2) deliberately removed it system-wide; f044 verifier recommends removal; zero known users (undocumented most of its life, default ""); honoring it would touch every branch-assuming skill | S:60 R:70 A:85 D:70 |
| 2 | Confident | Retire the key fully (struct field, accessor, registry row, scope tables) instead of only bypassing it in batch_switch.go | Leaving it would recreate the zombie-config state f044 flagged; retired-keys test (`TestConfigReferenceRetiresLegacyKeys`) is the established precedent for exactly this | S:55 R:75 A:80 D:75 |
| 3 | Tentative | Ship a migration that deletes a live `branch_prefix:` key from user configs (no-op when absent); fence cleanup itself needs no migration (regenerated by `fab config upgrade`) | code-quality.md anti-pattern rule requires migrations for config restructuring, but expected real-world impact is ~zero; apply should confirm shape/versioning against the latest retired-key migration precedent | S:35 R:70 A:45 D:35 |
| 4 | Confident | Leave `configupgrade` golden/freeze/behavior test fixtures using `branch_prefix` as a synthetic key untouched | They are self-contained local field tables and frozen historical snapshots, not wired to the live registry; rewriting frozen snapshots is prohibited by their purpose | S:50 R:90 A:80 D:70 |
| 5 | Certain | `src/kit/VERSION` is not bumped by this change; the migration's TO version names the next minor | Version bumps are release-owned per repo precedent (log.seed.md l3ja note: "`src/kit/VERSION` left at 2.2.0 — the bump is release-owned") | S:70 R:90 A:95 D:90 |

5 assumptions (1 certain, 3 confident, 1 tentative, 0 unresolved).
