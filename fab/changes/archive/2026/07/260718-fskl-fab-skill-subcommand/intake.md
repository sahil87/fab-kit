# Intake: `fab skill` Subcommand + Canonical Skill Bundle

**Change**: 260718-fskl-fab-skill-subcommand
**Created**: 2026-07-18

## Origin

Backlog item `[fskl]` invoked via `/fab-new fskl` (one-shot, no prior conversation). The entry is the adoption tracker deferred by change `260717-ptwh-toolkit-standards-conformance`'s conformance report (which recorded `skill` as "deferred, not yet adopted" per the standard's own Adoption section).

> [fskl] 2026-07-18: Toolkit `skill` standard adoption — implement `fab skill` (contract: shll docs/site/standards/skill.md @ v0.0.23; phased per-repo, no tool ships it yet, so absence is "not yet in violation" — this entry is the adoption tracker, deferred per change 260717-ptwh conformance report). Author the canonical `docs/site/skill.md` usage bundle (≤150 lines, STATIC-only — when-to-use, capabilities map, composition patterns, stdout/exit-code contracts, gotchas; NO flag tables/command trees/install prose — those defer to `-h`/help-dump/README) and serve it via a visible `fab skill` subcommand printing raw markdown to stdout byte-identical to the repo file (stderr empty on success, exit 0, no pager/framing), embedded at build time via the sync + drift-guard pattern (committed embedded copy + sync script + drift-guard test — the `shll standards` mechanism). Renders free at shll.ai/tools/fab-kit/skill as part of the pulled docs/site tree. New visible subcommand ⇒ update `src/kit/skills/_cli-fab.md` + tests AND re-verify the help-dump contract (command tree changes). Cross-repo: unblocks the fab row of shll [agst] bundle seeding; sibling of consumer-side [clix] (whose version-skew fallback assumes tools may predate their `skill` subcommand). shll v0.0.23.

The live standard was re-fetched at intake time (`shll standards skill`) and matches the backlog pin — same contract, Adoption section still reads "No tool ships `skill` today" (per the recurring lesson that the shll contracts evolve; re-fetch again at apply entry if time has passed).

## Why

1. **The gap**: three existing surfaces each fall short for an agent that wants to *use* an installed `fab` from an arbitrary repo — `-h`/help-dump is flag reference (structure, not judgment); README/docs/site needs a checkout or network; `fab/project/` context and the deployed `.claude/skills/` orient a *fab-managed-repo* session, not a caller elsewhere. A `<tool> skill` bundle is offline (embedded), present wherever the binary is, and version-locked by construction — the prose ships in the same binary as the flags it describes.
2. **The obligation**: the constitution's Toolkit Standards article binds this repo to the shll-published standards. The `skill` standard is phased per-repo; fab is currently "not yet in violation" but the adoption is tracked, and this change is the adoption.
3. **Cross-repo unblock**: shll backlog `[agst]` (aggregated bundle seeding — the forward `shll agent-setup` design that concatenates every installed tool's `<tool> skill` output) needs the fab row; consumer-side `[clix]` already assumes tools may predate their `skill` subcommand, so shipping order is safe.
4. **If we don't**: fab stays the tool an agent must guess at from `-h` alone outside fab-managed repos, and the toolkit-wide aggregation effort stalls on its most complex member.

## What Changes

### 1. `docs/site/skill.md` — the canonical bundle (new file)

A **usage briefing** in agent-first language, authored fresh for this change. Hard constraints from the standard:

- **≤150 lines** (hard budget — bundles are later aggregated across all installed tools; every line is paid N times).
- **Static only** — byte-identical on every invocation/machine for a given release; no timestamps, no environment lookups, no session state (contrast `run-kit context`, whose dynamic Environment header is exactly what this genre excludes).
- **In**: when to reach for fab (and when not); capabilities map (one line per capability, keyed to the subcommand — change lifecycle, status/state machine, score, resolve, dispatch, pane, config, memory-index...); composition patterns (how fab plays with `wt`, `rk`, `gh`, the `/fab-*` skills it deploys via `fab sync`); stdout/exit-code contracts (stdout-is-data, `--json` availability, the binary-wide `0`/`1` operational/`2` usage convention, the special exits: pane `2`/`3`, memory-index `0/1/2`, sync/migrations-status `3` = not a fab-managed repo); gotchas (the non-obvious traps — e.g., `fab` routes to two binaries; `.claude/skills/` is deployed copies, never edit; `<change>` accepts ID/substring/folder anywhere; `fab skill` ≠ fab's kit-skills concept).
- **Out** (explicitly): exhaustive flag tables (defer to `-h`), full command trees (defer to help-dump / shll.ai), install prose (defer to README / `docs/site/install.md`).
- Because `docs/site/**` is already the pulled site surface (§9 ACTIVE), the file renders at `shll.ai/tools/fab-kit/skill` with **zero additional work**.

Authoring sources: `README.md`, `src/kit/skills/_cli-fab.md`, live `-h` output, `docs/memory/distribution/*`. One fab-specific content obligation: the bundle must **disambiguate vocabulary** — `fab skill` (this toolkit-standard bundle command) vs. fab's own "skills" (the `/fab-*` markdown skills deployed to `.claude/skills/`); an agent meeting both terms cold will conflate them.

### 2. `fab skill` subcommand (new, visible, fab-go)

- Registered in the rich CLI at `src/go/fab/cmd/fab/skill.go`; the `fab` router's always-route policy forwards it with no router change (verified: no collision with the `LifecycleCommands` allowlist — init/upgrade-repo/sync/update/doctor/migrations-status — and `lifecycle_collision_test.go` guards this class automatically).
- Contract (uniform across the toolkit, from the standard's Invocation section):
  - Command name exactly `skill` (not `agent`, not `context` — name rationale is settled in the standard).
  - Prints the bundle as **raw markdown to stdout, byte-identical** to `docs/site/skill.md`.
  - **stderr empty on success, exit 0.** No rendering, no pager, no added framing.
  - Takes no args/flags (`cobra.NoArgs`); an argued invocation is a usage error → exit `2` via the existing binary-wide `run()`/`markRunReached` classification (no new code needed).

### 3. Build-time embedding — the sync + drift-guard pattern (new mechanism in this repo)

Reuse the mechanism `shll standards` established (the standard names it explicitly), adapted to a single file. fab has **no `go:embed` today**, and the geometry matches shll exactly: the Go module root is `src/go/fab/`, `docs/site/` sits above it, so `//go:embed` cannot reach the canonical file directly.

- **Committed embedded copy**: `docs/site/skill.md` copied to a path under the `cmd/fab` package dir (e.g. `src/go/fab/cmd/fab/skill.md` beside `skill.go`), embedded via `//go:embed`. The copy is committed so a clean `go build ./...` compiles without running any script.
- **Sync script**: `scripts/sync-skill.sh` (mirroring shll's `scripts/sync-standards.sh` — `set -euo pipefail`, cd to repo root, `cp -f` canonical → package dir), referenced by a `//go:generate` directive in `skill.go`.
- **Drift-guard test**: a Go test (shll's `TestStandardsEmbedMatchesCanonical` shape) comparing embedded bytes against the canonical `docs/site/skill.md` byte-for-byte, failing the build when they diverge. Plus contract tests: stdout byte-identity with the embedded copy, empty stderr, exit 0, ≤150-line budget (the line-budget assertion keeps future edits honest), and a static-only sanity check where feasible.

### 4. Documentation + conformance obligations (constitution-mandated)

- **`src/kit/skills/_cli-fab.md`**: new `## fab skill` section documenting the subcommand (a new visible subcommand changes the CLI surface).
- **SPEC mirror sweep** (code-quality § Sibling & Mirror Sweeps): on a CLI-surface change, treat all of a skill's SPEC mirrors as the sweep class — at minimum the `docs/specs/skills/` mirrors of any skill file touched, plus aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) if they restate the command roster.
- **Help-dump re-verification**: the command tree changes, so re-verify the help-dump contract (the dump is generated from the live cobra tree, so `skill` flows in automatically; the check is that conformance tests still pass and the new node carries a sensible `Short`/`Usage`).
- **No cobra `Example:` block required**: the b91h audit scoped `Example:` to user-facing *multi-flag* commands; `fab skill` is zero-flag/zero-arg, so it is not added to `exampleTargetPaths` (adding a trivial example is allowed but not required).
- **No migration**: nothing restructures user data — a new binary subcommand plus repo docs files only.

## Affected Memory

- `distribution/distribution.md`: (modify) the shll.ai docs-site / toolkit-standards-conformance material currently records "skill = not yet adopted; adoption tracked at [fskl]" — update to record the shipped adoption (subcommand, bundle, mechanism)
- `distribution/kit-architecture.md`: (modify) the fab-go command inventory gains the visible `fab skill` subcommand and the repo's first `go:embed` + sync-script + drift-guard mechanism

## Impact

- **New files**: `docs/site/skill.md` (canonical bundle), `src/go/fab/cmd/fab/skill.go` (+ `skill_test.go`), the committed embedded copy under `src/go/fab/cmd/fab/`, `scripts/sync-skill.sh`.
- **Modified files**: `src/kit/skills/_cli-fab.md` (+ its SPEC mirror sweep class under `docs/specs/`), possibly aggregate specs restating the command roster.
- **Untouched**: the `fab` router and fab-kit binary (always-route covers the new command); templates/migrations; no config schema change; consumer-side version-skew is shll `[clix]`'s problem by design.
- **Tests**: new drift-guard + contract tests in `cmd/fab`; existing `lifecycle_collision_test.go` and help-dump conformance tests re-run green.
- **Release note surface**: new visible subcommand ⇒ shows up in `-h` and the shll.ai command reference on next release capture.

## Open Questions

*(none — the backlog entry plus the live standard fully determine the contract; remaining choices are graded below)*

## Assumptions

<!-- STATE TRANSFER: This table is the sole continuity mechanism between the intake-stage
     agent and the apply-entry agent (which co-generates plan.md). -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Adopt the invocation contract verbatim from the live standard: command named exactly `skill`, raw markdown to stdout byte-identical to `docs/site/skill.md`, stderr empty on success, exit 0, no pager/framing, static-only, ≤150 lines | Constitution's Toolkit Standards article binds; live standard re-fetched at intake and matches the backlog pin (v0.0.23) | S:95 R:90 A:95 D:95 |
| 2 | Certain | Embed via the sync + drift-guard pattern (committed copy in the `cmd/fab` package + `scripts/` sync script + byte-identity drift-guard test), reusing shll's `standards` mechanism | The standard names this mechanism explicitly ("reuse it"); module-root geometry is identical (module at `src/go/fab/`, `docs/site/` above it) | S:90 R:85 A:90 D:90 |
| 3 | Confident | Concrete placement/naming: embedded copy at `src/go/fab/cmd/fab/skill.md` beside `skill.go` (single-file `//go:embed skill.md`), sync script `scripts/sync-skill.sh` with a `//go:generate` pointer | Mirrors shll's layout adapted from a 4-file dir to one file; pure naming choice, trivially renameable during apply | S:70 R:90 A:80 D:65 |
| 4 | Confident | Bundle audience & content: a usage briefing for an agent operating the *installed* fab without the deployed kit-skills context; drawn from README/`_cli-fab.md`/`-h`/memory; must explicitly disambiguate `fab skill` (this bundle) from fab's kit-skills (`.claude/skills/` deployed by `fab sync`) | The standard's genre definition answers the audience question; exact prose selection is authoring judgment grounded in strong repo signal | S:75 R:75 A:70 D:60 |
| 5 | Confident | No router or fab-kit change: the always-route policy forwards `skill` to fab-go; `skill` collides with nothing in the `LifecycleCommands` allowlist (verified at intake; `lifecycle_collision_test.go` guards it) | Verified against `src/go/fab-kit/cmd/fab-kit/main.go`; the collision test makes regressions loud | S:65 R:85 A:80 D:85 |
| 6 | Certain | Ship the constitution-mandated companions in the same change: `_cli-fab.md` `## fab skill` section, SPEC-mirror sweep of touched skill files + roster-restating aggregate specs, Go tests alongside the code, help-dump conformance re-verified | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps are explicit; this is the project's #1 recurring rework cause when skipped | S:90 R:85 A:95 D:95 |
| 7 | Confident | No cobra `Example:` block / no `exampleTargetPaths` entry for `fab skill` | The b91h audit scoped `Example:` to user-facing multi-flag commands; `fab skill` takes no flags or args | S:60 R:95 A:80 D:75 |
| 8 | Certain | No migration ships with this change | Nothing restructures user data (`fab/`, `.status.yaml`, config, archive untouched) — new subcommand + repo docs only | S:85 R:90 A:95 D:95 |
| 9 | Certain | Consumer-side version-skew and site rendering need no work here: shll `[clix]` owns the predates-subcommand fallback; `docs/site/**` is already the pulled+rendered site surface | Backlog entry states both; the docs-site pull (§9 ACTIVE) is recorded in distribution memory | S:80 R:90 A:85 D:90 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
