# Intake: Kit Upgrade 2.16.3 → 2.16.4

**Change**: 260719-6kcj-kit-upgrade-2-16-4
**Created**: 2026-07-19

## Origin

Promptless intake dispatch (`{questioning-mode} = promptless-defer`) from a synthesized description — no interactive conversation. The source material:

> Adopt the fab-kit **2.16.3 → 2.16.4** repo upgrade produced by running `fab upgrade-repo` — the diff is ALREADY APPLIED to the working tree (uncommitted). The change must NOT re-run or alter the upgrade; its job is to carry the already-applied mechanical diff through the pipeline: verify the diff is exactly the expected version stamp and nothing else, keep it intact, and ship it as a PR.

The upgrade was run before this intake was created; this change exists to shepherd the resulting diff through the pipeline (verify → keep intact → PR), not to produce it.

## Why

1. **Problem**: the repo's kit metadata was one release behind the installed `fab` binary (binary 2.16.4, repo stamped 2.16.3). `fab upgrade-repo` was run and stamped the repo to 2.16.4; that diff now sits uncommitted in the working tree and must land on `main`.
2. **Consequence if not shipped**: the working tree stays permanently dirty with version-stamp noise that pollutes every subsequent change's diff, and the repo's recorded kit version (`fab/.fab-version`, `fab/.kit-migration-version`, the config reference-fence header) misreports the kit actually in use — version-drift detection and future migrations key off these files.
3. **Why through the pipeline**: the previous kit upgrade (2.16.2 → 2.16.3) was commit `521719c5` "Upgrading fab-kit" with the identical 3-file shape, committed directly to `main`. This time the same mechanical bump goes through the full pipeline for traceability (change record + PR to sahil87/fab-kit) instead of a direct commit.

## What Changes

### The already-applied diff (to be carried, not created)

The working-tree diff is exactly three files, all mechanical version stamps — verified at intake time with `git status --porcelain` / `git diff`:

1. `fab/.fab-version`: `2.16.3` → `2.16.4`
2. `fab/.kit-migration-version`: `2.16.3` → `2.16.4`
3. `fab/project/config.yaml`: a single comment line — the regenerated reference-fence header:

   ```diff
   -# >>> fab reference (kit 2.16.3) >>> ---------------------------------------
   +# >>> fab reference (kit 2.16.4) >>> ---------------------------------------
   ```

   No field values changed; all comments preserved.

### What the pipeline must do

- **Verify, don't produce**: apply's job is verification that the working-tree diff is exactly the 3-file stamp above and nothing else (`git status --porcelain` shows only those three modified files; `git diff` content matches). Do NOT re-run `fab upgrade-repo`, do NOT edit the three files, do NOT "fix" anything in them.
- **Keep intact**: no task may revert, regenerate, or amend the stamped files. The only additional files this change introduces are its own pipeline artifacts under `fab/changes/260719-6kcj-kit-upgrade-2-16-4/`.
- **Ship as PR**: commit the 3-file diff (plus change artifacts) and open a PR against `main` on sahil87/fab-kit (via `/git-pr` at ship).

### `fab upgrade-repo` output facts (context for verification)

- Current version 2.16.3, target 2.16.4; the installed fab binary is 2.16.4.
- Sync validated all 34 Claude Code skills already valid (created 0, repaired 0); `.claude/settings.local.json`, `.envrc`, `.gitignore` all OK — hence no diff beyond the three stamps.
- `fab/.kit-migration-version` was OK at 2.16.3 and stamped to 2.16.4 — no kit migration steps were needed for 2.16.3 → 2.16.4.
- Project sync script `1-worktree-backlog.sh` ran normally.
- Config field `agent` equals the current default and was deliberately kept as-is ("presence=intent") — no action required, no config field diff expected.

## Affected Memory

None. The upgrade *mechanism* — `fab upgrade-repo`, version stamping, the migration dual-version model — is documented in the `distribution` domain (`distribution/distribution.md`, `distribution/migrations.md`) and did not change; this change is merely an application of that mechanism to this repo. Per the intake rule, only spec-level behavior changes warrant memory updates — a mechanical version stamp changes no behavior.

## Impact

- **Files**: exactly 3 modified files (`fab/.fab-version`, `fab/.kit-migration-version`, `fab/project/config.yaml` — one comment line), plus this change's own artifacts under `fab/changes/260719-6kcj-kit-upgrade-2-16-4/`.
- **Code**: no `src/` changes (neither Go nor kit content); no behavior change in this repo's sources.
- **Tests**: none affected — no `.go` files touched, so no test runs are required (per code-quality test strategy, test scope follows changed `.go` files).
- **Change type**: chore (mechanical version bump).
- **Risk**: minimal — the diff is already applied and validated; the residual risk is accidental mutation of the diff during the pipeline, which the verify-and-keep-intact tasks guard against.

## Open Questions

None — the synthesized description resolves scope, verification method, shipping route, and memory impact explicitly.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change type is `chore` | Explicit in the dispatch description; matches the taxonomy (mechanical version bump, no behavior change) | S:95 R:90 A:95 D:95 |
| 2 | Certain | Carry-only scope — do NOT re-run or alter `fab upgrade-repo`; apply verifies the diff and keeps it intact | Explicit in the dispatch description ("must NOT re-run or alter the upgrade") | S:95 R:85 A:90 D:95 |
| 3 | Certain | Affected Memory: none | Explicit in the dispatch; confirmed against `docs/memory/index.md` — the mechanism lives in the `distribution` domain and is unchanged; stamp application changes no spec-level behavior | S:90 R:90 A:95 D:90 |
| 4 | Certain | Ship via the full pipeline as a PR to sahil87/fab-kit (departing from the direct-to-main precedent `521719c5`) | Explicit in the dispatch ("this time it goes through the full pipeline") | S:90 R:85 A:90 D:90 |
| 5 | Certain | Acceptance = diff exactness: `git status --porcelain` shows only the 3 expected files; `git diff` matches the stamps verbatim (config.yaml delta limited to the fence-header comment) | Explicit verification instruction in the dispatch; already confirmed true at intake time | S:90 R:90 A:95 D:90 |
| 6 | Confident | Change artifacts (`fab/changes/260719-6kcj-kit-upgrade-2-16-4/`) ride in the same PR alongside the 3-file stamp | Standard fab pipeline behavior; "keep the diff intact" governs the three stamped files, not a prohibition on the change's own artifacts | S:70 R:80 A:85 D:80 |
| 7 | Confident | The uncommitted diff survives change activation and branch creation (git preserves the working tree across branch switches from the same HEAD) | Standard git behavior; the pipeline stops at intake `ready` here, so activation/branching happens later under the dispatcher's control | S:75 R:75 A:85 D:80 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
