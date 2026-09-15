# Intake: Remove fab_version Config Fallback

**Change**: 260719-kq7v-remove-fab-version-fallback
**Created**: 2026-07-19

## Origin

Backlog item `[kq7v]` (2026-07-19), invoked one-shot via `/fab-new kq7v`:

> Delete the one-compat-window `fab_version` fallback code after the migration horizon: the fab-go `internal/config` `fab_version` yaml-tag parse, the `pruneProjectScoped` strip line, and fab-kit `readFabVersion`'s step 2 (config.yaml `fab_version:` fallback — the pin lives in `fab/.fab-version` since j0qm). (relocated from docs/memory/_shared/configuration.md by /docs-distill-memory)

## Why

The 260708-j0qm change (config-upgrade Change 3, shipped in 2.15.1 on 2026-07-08) relocated the project-pinned engine version out of `fab/project/config.yaml` into the plain-text sibling `fab/.fab-version`, with migration `2.14.0-to-2.15.0` moving the value for existing repos. Both reader stacks kept a **deliberately one-compat-window** fallback to the legacy `config.yaml` `fab_version:` key, and every site carries a comment promising its removal (e.g. `config.go:324` "Removed once the compat-window Config.FabVersion field goes"; `configscope.go:70` "a one-off in the loader, removed with the compat window").

The horizon has passed: current release is 2.16.4, several releases beyond 2.15.x, and any repo that upgraded through `fab upgrade-repo` has run the `2.14.0-to-2.15.0` migration (the chain is sequential). Keeping the fallback now costs:

1. **Dead-weight complexity** — three code sites plus a named scope-taxonomy exception (`pruneProjectScoped`'s one-off strip outside the `keyScopes` table) that every reader of `internal/config` and `internal/configscope` must reason around.
2. **Stale-truth drift risk** — memory and spec files describe the fallback as live present-truth; the longer the "temporary" window stays open, the more docs accrete on top of it.
3. **A broken promise** — the compat window was scoped to one window by design; leaving it indefinitely makes the next compat-window commitment less credible.

If we don't remove it, nothing breaks — but the single-writer/single-source design of j0qm stays permanently half-landed.

## What Changes

### 1. fab-go `internal/config`: stop parsing `fab_version:` from config.yaml

`src/go/fab/internal/config/config.go`:

- `Config.FabVersion` field (line 123): change the tag `yaml:"fab_version"` → `yaml:"-"`. The **field itself stays** — it is the carrier `Load` populates from `fab/.fab-version` (`readDotFabVersion` overlay, lines 145–147), consumed by `GetFabVersion()` → `internal/preflight.checkSyncStaleness` (preflight.go:155). Only the config.yaml *parse* is deleted. An explicit `yaml:"-"` (not a bare untagged field) is required so yaml.v3 cannot match the lowercased field name.
  <!-- assumed: keep FabVersion as a non-yaml carrier field rather than removing it from Config — preflight still needs the .fab-version value through GetFabVersion -->
- `Load` doc comment (lines 132–139): rewrite — `.fab-version` is now the sole source; drop the "for one compat window, falls back to a config.yaml `fab_version:` key" sentence.
- `readDotFabVersion` comment (lines 151–154): drop "defers to the config.yaml fallback in Load" — a missing `.fab-version` now simply leaves `FabVersion` empty (preflight's staleness check already silent-skips an empty value).

### 2. fab-go: delete the `pruneProjectScoped` strip line

`src/go/fab/internal/config/config.go` (lines 310–327):

- Delete `delete(m, "fab_version")` and the whole "fab_version is a NAMED compat-window exception" comment block (lines 317–325). With the yaml tag gone, a stale system-file `fab_version:` is an inert unknown key — nothing unmarshals it, so it can never reach a repo's resolved version. The only residual surface is cosmetic: plain `fab config show` renders the raw merged map, so a stale *system-file* key would display — accepted, because `fab_version` never legitimately lived in the system file (`~/.fab-kit/config.yaml` postdates the relocation; the strip was purely defensive), and `--origin` walks the registry, which carries no `fab_version` row.
- `src/go/fab/internal/configscope/configscope.go` (comment, lines 60–70): rewrite the "fab_version is intentionally ABSENT" paragraph — keep the fact that `fab_version` is not a config key (it lives in `fab/.fab-version`), drop all compat-window/fallback/strip prose.

### 3. fab-kit: delete `readFabVersion` step 2

`src/go/fab-kit/internal/config.go`:

- Delete step 2 of `readFabVersion` (lines 108–122): the config.yaml read, the anonymous-struct `yaml:"fab_version"` unmarshal, and the non-empty check. `fab/.fab-version` becomes the sole source; an absent/empty file is the error case.
- Error message: currently `"no fab version found in fab/.fab-version or config.yaml. Run 'fab init' to set one"`. New message drops the config.yaml mention and names the existing-repo recovery path, e.g.: `"no fab version found in fab/.fab-version. Run 'fab init' (new repo) or 'fab upgrade-repo' (existing repo) to set one"`. <!-- assumed: exact final wording is an apply decision; the two fixed requirements are dropping "or config.yaml" and pointing at upgrade-repo as the unmigrated-repo recovery -->
- The `configPath` parameter becomes unused inside `readFabVersion` — shrink the signature to `readFabVersion(repoRoot string)` (the caller `resolveConfigFrom` keeps `candidate` for `ConfigResult.ConfigPath`).
- Drop the now-unused `gopkg.in/yaml.v3` import if nothing else in the file uses it (nothing else does today).
- Update the `dotFabVersionRelPath` comment (lines 14–17): drop "config.yaml's fab_version: key is read only as a one-compat-window fallback".
- `Upgrade`'s missing-version tolerance (`upgrade.go` lines 51–71) is **untouched** — it is the recovery path for a repo whose pin is unresolvable, and it never reads the legacy key itself (it proceeds with `FabVersion: ""`).

### 4. Tests (both modules)

fab module — `src/go/fab/internal/config/config_test.go`:

- `TestLoad_FabVersionFallbackToConfig` (line ~173): the pinned behavior is being deleted. Replace with a test pinning the NEW behavior: config.yaml `fab_version:` present + no `.fab-version` ⇒ `GetFabVersion() == ""` (the key is ignored, no error).
- The `.fab-version`-wins test (line ~152): keep — still valid (stale key in config.yaml is now simply ignored rather than out-competed; assert the same winning value).
- The `pruneProjectScoped` strip tests (lines ~697–760, including the silent-strip assertion and the "system fab_version must not become the resolved version" cascade test): rework — the strip is gone, so assert instead that a system-file `fab_version` never reaches `Config.FabVersion` (inert unknown key) and produces no warning.
- `TestConfigReferenceOmitsRelocatedFabVersion` (cmd/fab/config_test.go:118): unchanged — the registry still carries no row.

fab-kit module:

- `upgrade_test.go` fixtures (lines 65, 263) and `sync_integration_test.go` (line 81) create repos by writing `fab_version:` into config.yaml — these rely on the fallback for version resolution. Repoint fixtures to write `fab/.fab-version` instead.
- Add/adjust a `readFabVersion` test pinning: `.fab-version` absent ⇒ the new error (config.yaml `fab_version:` no longer consulted).
- Run `go test ./...` in both `src/go/fab` and `src/go/fab-kit`.

### 5. Kit-doc and spec sweeps (constitution-mandated)

- `src/kit/skills/_cli-fab.md` — three sites claim the fallback as live: the router description (line 44, "for a not-yet-migrated repo the reader falls back to a legacy `fab_version:` key … for one compat window"), the `sync` version-guard row (line 65, "with the legacy config.yaml `fab_version:` fallback"), and the `migrations-status` section (line 541, same phrase). Drop the fallback clauses; `fab/.fab-version` is the sole source.
- `docs/specs/skills/SPEC-_cli-fab.md` — sweep the mirrored router/sync/migrations-status prose for the same fallback phrasing and align (constitution: skill edits update their SPEC mirror).
- `docs/specs/config.md` — lines ~121–122 and ~309–313 describe the fallback + strip as the landed design; update to record the window as closed (this change).
- Grep-sweep the whole class before finishing apply: `grep -rn "fallback" --include="*.md" src/kit docs/specs | grep -i "fab_version\|fab-version"` plus the literal phrase "compat window" — per code-quality § Sibling & Mirror Sweeps.

### Non-Goals

- **No new migration file.** No user data is restructured — `2.14.0-to-2.15.0` already owns the relocation and remains in the chain for pre-2.15 repos.
- The `.fab-version` gitignore-negation seams (8ken, 2.15.1-to-2.15.2 verify+commit migration) are untouched.
- No change to `fab config upgrade`'s parked-removal block (`config.yaml`'s trailing "removed in an earlier release" comment for `fab_version` is fence-engine territory, not this change).

## Affected Memory

- `_shared/configuration.md`: (modify) § `fab_version` — lines 104–108 claim "Both reader stacks read `.fab-version` first, with a one-compat-window config.yaml `fab_version:` fallback" and name the three fallback code pieces; rewrite to sole-source present-truth
- `distribution/kit-architecture.md`: (modify) line ~198 — same both-reader-stacks fallback claim on the `fab/.fab-version` entry
- `distribution/distribution.md`: (modify) line ~180 — preflight staleness "falling back to `fab_version:` in `config.yaml` for one compat window (j0qm)"
- `distribution/migrations.md`: (modify) line ~177 — the `2.14.0-to-2.15.0` entry's present-tense "Both reader stacks … read `.fab-version` first with a one-compat-window fallback; this migration closes that window" — reword to reflect the window is now closed in code as well

## Impact

- **Code**: `src/go/fab/internal/config/config.go`, `src/go/fab/internal/configscope/configscope.go` (comment only), `src/go/fab-kit/internal/config.go`; tests in `src/go/fab/internal/config/config_test.go`, `src/go/fab-kit/internal/upgrade_test.go`, `src/go/fab-kit/internal/sync_integration_test.go` (+ any router-resolution fixtures found during apply).
- **Behavior**: a repo never migrated past 2.15.0 (pin still only in `config.yaml`) hard-fails fab-kit router resolution with the actionable error; recovery is `fab upgrade-repo` (or `fab init` for a fresh repo). Migrated repos see zero behavior change. fab-go preflight staleness for such a repo silently skips (empty version) instead of warning — advisory-only check, acceptable.
- **Kit content**: `src/kit/skills/_cli-fab.md` (3 sites) + `docs/specs/skills/SPEC-_cli-fab.md` mirror + `docs/specs/config.md`.
- **Scale**: small, surgical deletion — ~3 code files, ~5 test files, ~3 doc files.

## Open Questions

None — the backlog entry names the exact three deletion targets, and the code comments pre-authorize the removal.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Keep `Config.FabVersion` as a non-yaml carrier field (`yaml:"-"`) rather than deleting the field | Backlog names the *yaml-tag parse* as the target; `Load`'s `.fab-version` overlay and preflight's `GetFabVersion` consumer still need the carrier; removing the field forces re-plumbing for zero benefit | S:75 R:80 A:80 D:65 |
| 2 | Confident | Migration horizon has passed — breaking never-migrated repos is acceptable | Backlog filed 2026-07-19 explicitly "after the migration horizon"; j0qm shipped 2.15.1 (2026-07-08), current 2.16.4; every repo upgraded since ran the migration chain; recovery is one command (`fab upgrade-repo`) | S:80 R:45 A:65 D:75 |
| 3 | Certain | No new migration file | No user data is restructured — binary behavior + docs only; `2.14.0-to-2.15.0` already owns the relocation and stays in the chain for stragglers | S:85 R:85 A:90 D:85 |
| 4 | Confident | `readFabVersion` error message drops "or config.yaml" and adds `fab upgrade-repo` as the existing-repo recovery pointer | Message must stop naming a source that is no longer read; `upgrade-repo` is the actual recovery path for an unmigrated repo (`Upgrade` tolerates a missing pin) | S:55 R:95 A:80 D:70 |
| 5 | Confident | Deleting the strip line leaves a stale system-file `fab_version` as an inert unknown key; cosmetic plain-`show` bleed accepted | No yaml tag ⇒ can't reach the resolved config; `show --origin` walks the registry (no `fab_version` row); the system file postdates the relocation so the key ~never exists there | S:70 R:85 A:80 D:70 |

5 assumptions (1 certain, 4 confident, 0 tentative, 0 unresolved).
