# Intake: Tests, CI & Toolchain

**Change**: 260612-tb6f-tests-ci-toolchain
**Created**: 2026-06-12

## Origin

One-shot `/fab-new tb6f` resolving backlog item `[tb6f]` — Binary-review batch B6/6, the **last of the six batches** filed from the 2026-06-12 adversarially-verified Go binary review (report: `docs/specs/findings/binary-review-2026-06-12.md` §B6 F39–F47, baselined against commit `1431a9c3`/v2.1.6).

> [tb6f] 2026-06-12: Binary-review batch B6/6 — tests, CI & toolchain. DEPENDS: wave 3, LAST of the six — after k4ge (state-machine tests must target the FIXED transition semantics incl. AllowedStates hardening), mz4q (locking changes Save behavior), dn2c (release.yml seam + new lifecycle exit contracts to test), ye8r (flag-API changes land before the cobra bump). GOAL: the state layer and destructive mutators are tested, releases can't ship untested, the toolchain is in support. ACTIONS: F39 table-driven exhaustive test over lookupTransition (stage × event × from-state, incl. the review/review-pr failed→active override) + Reset/Skip cascade tests. F45 test fab-kit's destructive workspace mutators. F40 release.yml runs ZERO tests before shipping binaries on a tag push — add a test step, and replace the hardcoded go-version '1.22' with go-version-file. F42 compile darwin in CI. F41 toolchain: bump go + cobra; for yaml evaluate goccy/go-yaml vs staying pinned, with golden round-trip tests on .status.yaml FIRST. F46 cover the cobra wiring RunE bodies. F44 split fab-kit/internal/sync.go. F47 remove dead exported funcs + vestigial code INCLUDING src/benchmark/statusman-go, EXCLUDING stage_hooks/model_tiers — c5tr owns that wire-or-remove decision, coordinate. F43 optional/stretch (effort large): move domain logic out of cmd/fab package main into internal packages — take only the pieces that fall out naturally while testing. CONSTRAINTS: constitution — Test Integrity (tests conform to spec, never the reverse); yaml swap must preserve byte-stable index/status output; src/kit canonical. REPORT: docs/specs/findings/binary-review-2026-06-12.md §B6 F39-F47 (vs 1431a9c3).

**Wave-3 gate status (verified at intake)**: all dependency batches are merged into main at this branch's base `47846c0b` — k4ge (#395), mz4q (#396), dn2c (#397), pw3k (#398), c5tr (#399), hv7t (#400), g8st (#401), ye8r (#402). The dn2c↔tb6f release.yml seam dissolved by merge order: dn2c's checksum publishing is already on main; this change adds the test gate on top with no rebase choreography.

**State refresh (measured at intake, this branch)**: because four sibling batches landed tests after the report's baseline, every coverage number in §B6 was re-measured today via `go test -coverprofile` + `go tool cover -func`. The refreshed numbers below are authoritative for this change; the report's baselines are listed only for context. Do not re-implement tests siblings already added.

## Why

1. **The pain**: the core pipeline state machine (`src/go/fab/internal/status`) still has genuinely untested mutators (`SetChangeType`, `AddIssue`, `ProgressMap`, `ProgressLine`, `AllStages` at 0.0%; `Skip`'s forward-cascade at 26.3%), and fab-kit's two riskiest code paths — the `Sync` orchestrator and `cleanLegacyAgents` (which deletes files in user repos) — are at 0.0%. This is exactly the state logic that, if regressed, corrupts every change's `.status.yaml` while all existing tests stay green. A live suspect already exists: `stage_metrics` review iterations were observed resetting to 1 under the fail→reset rework choreography on 2026-06-12 (PR #402's meta reported "1 cycle" for a 3-cycle review) despite #395 claiming that fix.
2. **The consequence of not fixing**: `release.yml` ships binaries to Homebrew users on any tag push with **zero** test/vet steps (the `if:` guard only restricts `workflow_dispatch`), and the documented backport flow (release.sh pushing directly to `release/*` branches) never passes through ci.yml at all — untested releases are a real, documented path. Meanwhile the shipped binaries carry the frozen Go 1.22 stdlib (out of the security window since Go 1.24, Feb 2025) and make TLS downloads with it (`download.go` against github.com); yaml.v3 (archived April 2025) parses every `.status.yaml` and `config.yaml` the toolkit touches.
3. **Why now, as the last batch**: deliberately sequenced after all five sibling batches so tests target the *fixed* semantics — k4ge's AllowedStates hardening, mz4q's locking/Save behavior, dn2c's fail-loud lifecycle exit contracts, ye8r's flag-API/LifecycleCommands changes (which had to land before the cobra bump). That gate is satisfied (see Origin).

## What Changes

### F39 — State-machine tests for `internal/status` [high/small]

Current coverage (refreshed; report baseline was package 37.9%): **package 67.8%**. Siblings closed much of the original gap (Reset 77.8%, Fail 69.2%, AddPR 80.0%, Advance 55.6%, lookupTransition 77.8%). Remaining work:

- **Exhaustive table-driven matrix over `lookupTransition`** (stage × event × from-state), asserting allowed targets and rejections against the transition tables — including the review/review-pr `failed→active` start override. lookupTransition sits at 77.8%, not exhaustive.
- **`Skip` forward-cascade** pending→skipped (26.3% — the subtle ordered-iteration logic over `sf.StageOrder` is untested).
- **Still 0.0%, untested anywhere**: `SetChangeType`, `AddIssue`, `ProgressMap`, `ProgressLine`, `AllStages`.
- **`Advance`** (55.6%) — cover the remaining branches.
- Use the existing `loadFixture` helper (already round-trips a `.status.yaml` in `t.TempDir`). Tests MUST target the post-k4ge transition semantics (AllowedStates hardening) and post-mz4q Save-under-lock behavior — the spec is current main, not the report's baseline.
- **`stage_metrics` iterations regression test**: write a test asserting `stage_metrics.review.iterations` accumulates across the fail→reset→re-finish rework choreography (the spec'd behavior #395 claimed to implement). First check #395's actual implementation; if the test exposes the observed reset-to-1 bug, fix the implementation to match the spec (constitution VII — Test Integrity).

### F45 — fab-kit destructive workspace mutator tests [medium/medium]

Current coverage (refreshed; report baseline was package 52.2%): **package 67.1%**. dn2c closed part of the gap (Init 77.3%, deploySkills 88.9%, Upgrade 46.4%). Remaining work:

- **Still 0.0%**: the `Sync` orchestrator (`sync.go:39`) and `cleanLegacyAgents` (`sync.go:803` — the file-deleting path, the riskiest code in the binary). `syncAgentSkills` at 54.7%.
- Add an integration-style test: build a temp git repo + fake cached kit dir (existing tests already construct both pieces individually), run `Sync` **twice**, assert (a) the resulting tree is correct and (b) the second run is a **content-identical no-op** — directly encoding constitution III's idempotency MUST (compare content, not mtimes — `syncAgentSkills` rewrites only on content mismatch).
- Cover the `shimOnly`/`projectOnly` branch split and `cleanLegacyAgents`' deletion scoping in the same harness.
- Known feasibility seams (from the verifier): `FAB_AGENTS` env override for deploySkills; `$HOME`-rooted cache with `os.Setenv("HOME", tmp)` precedent in `cache_test.go`; `systemVersion="dev"` bypasses versionGuard; `checkPrerequisites` requires git/bash/yq-v4/direnv on PATH — needs PATH shims or CI deps.
- Cover `Upgrade`'s remaining branches (46.4%) — dn2c's F18 stamp-after-success ordering is the new contract to test.

### F40 — Release workflow test gate + single-source Go version [high/small]

`release.yml` currently: checkout → build → package → release → push formula, with zero test/vet/gofmt steps, on `push: tags: ['v*']` from any ref.

- Add a test step before `just build-all` — either invoke `just test` directly or make the release job `needs:` a reusable test job shared with `ci.yml`.
- Replace the hardcoded `go-version: '1.22'` (release.yml:61) with `go-version-file:` pointing at a module go.mod — making ci.yml's "single source of truth" comment true (it has been false since birth).
- Update the release-workflow step list in `docs/memory/distribution/distribution.md` (~lines 267-280) per Docs Are Source of Truth.

### F42 — Compile darwin in CI [medium/small]

CI is linux-only; `proc_darwin.go` + `pane_process_darwin.go` (~131 lines) are first compiled by `just build-all` inside the release workflow — after the tag is already pushed.

- Add a CI step per module: `GOOS=darwin GOARCH=arm64 go build ./... && GOOS=darwin GOARCH=arm64 go vet ./...`. Cross-compilation type-checks the darwin files; no macOS runner needed (empirically verified to pass on linux by the report's verifier). Only the `fab` leg strictly needs it (fab-kit has no build-constrained files) but apply to both matrix legs — harmless and future-proof.
- Add `-race` to the existing linux `go test` step while touching it (nearly free at this suite size).

### F41 — Toolchain bump [medium/small]

Both `go.mod` files declare `go 1.22`, cobra `v1.8.1`, yaml.v3 `v3.0.1` (verified current on this branch).

- **Go**: bump the `go` directive in both modules to the current supported line (local toolchain is 1.26.2). CI picks it up automatically via `go-version-file`; release.yml is fixed by F40.
- **cobra**: v1.8.1 → current v1.10.x line (two *minor* versions behind, not majors — verifier correction). ye8r's flag-API changes are merged, so the bump dependency is satisfied; ye8r's contract/collision drift tests now guard the CLI surface through the bump.
- **yaml.v3** (archived April 2025, load-bearing in 11+ non-test files): write **golden round-trip tests FIRST** asserting byte-stable output for `.status.yaml` and the generated memory/archive indexes (byte-stability is a documented property of `fab memory-index`). Then evaluate goccy/go-yaml: swap only if the golden tests prove byte-identical parity; otherwise stay pinned and record the archived status + migration plan in memory.
- Update docs that document "Go 1.22": `docs/memory/distribution/distribution.md` (~:273) and `kit-architecture.md` (~:295).
- The `src/benchmark/statusman-go` module (go 1.25.0) is deleted by F47, so it needs no bump.

### F46 — Cobra wiring coverage for fab-go [low/medium]

Current coverage (refreshed; report baseline was package 35.6%): **cmd/fab 55.2%**. Remaining low RunE bodies (measured): `changeArchiveListCmd` 10.0%, `logReviewCmd` 12.5%, `changeArchiveCmd` 23.5%, `changeSwitchCmd` 23.5%, `changeRestoreCmd` 28.6%, `changeListCmd` 35.7%, `changeRenameCmd` 38.5%, `listPendingItems` 27.3%.

- Extend the in-package cobra-execution pattern — the accurate exemplars are `memory_index_test.go:23-100` (setupFabRepo t.TempDir fixture + os.Chdir + SetArgs + stdout/file assertions), `pr_meta_test.go`, `shellinit_test.go` (NOT batch_new_test.go, which only asserts command structure — verifier correction).
- Assert the exact stdout shapes skills parse: the archive command's structured YAML, the `already archived:` soft-skip line, and hv7t's new `index: failed` print-then-error contract (non-zero exit with YAML still printed).

### F44 — Split `fab-kit/internal/sync.go` [medium/small]

Now **886 lines** (grew from the report's 780 — dn2c added to it); still the bundled monolith. Mechanical split within the existing flat internal package, no API changes, tests move with their functions:

- `semver.go` — parseSemver/compareSemver; **fix the `major < "4"` lexicographic string-compare bug in `checkPrerequisites` while moving** (misorders a hypothetical yq v10; the bug is in checkPrerequisites, not compareSemver — verifier correction).
- `prereqs.go` — checkPrerequisites incl. yq version sniffing.
- `scaffold.go` — tree-walk + both merge mini-engines (JSON permissions merge, line-ensure merge).
- `skills.go` — deploySkills/listSkills/syncAgentSkills/cleanStaleSkills/cleanLegacyAgents.
- `sync.go` keeps the ~100-line orchestrator (+ versionGuard).
- Touch up `docs/memory/distribution/migrations.md` (~:62) which cites "the existing parseSemver/compareSemver helpers in sync.go".

### F47 — Dead code & vestigial removal [low/small]

All verified still present on this branch:

- `resolve.ToDir`/`resolve.ToStatus` — zero non-test callers while `cmd/fab/resolve.go:40-42` re-inlines the identical format strings. Pick one: delete them (plus their tests in resolve_test.go) or route the command through them. Default: delete.
- `change.List` — unused wrapper; all callers use `ListWithOptions`.
- Collapse the duplicate digit-only int parsers — `status.parseInt` and `statusfile.parseIntStrict` (identical logic; `status` already imports `statusfile`, no cycle).
- **Delete `src/benchmark/` implementations** (statusman-go module + node/rust/bash-opt siblings — never compiled/tested by any workflow or justfile recipe), keeping `RESULTS.md` + `README.md` as the historical decision record (the decision is also duplicated in `kit-architecture.md` §statusman decision record).
- **Absorb hv7t's recorded deletion candidates** (hv7t plan.md § Deletion Candidates): `internal/backlog/backlog.go:117` (`MarkDone`'s inline ReadFile+Split, now redundant with `internal/lines`), `internal/intake/intake.go:28` (same idiom — confirm CRLF-trim parity is acceptable when migrating), `internal/score/changetypes_doc_test.go:140` (test-local bufio.Scanner whose cited precedent is gone).
- **Exclusions — do not touch**: `stage_hooks` (c5tr/#399 deliberately *preserves* it — see the `2.1.6-to-2.2.0` migration) and `model_tiers` (already dropped defensively by that same migration). The backlog's "coordinate with c5tr" is resolved: c5tr is merged; re-verify residue at apply but do not re-litigate.

### F43 — Stretch: move domain logic out of cmd/fab [medium/large, optional]

Per the backlog, **take only the pieces that fall out naturally while testing** — this is explicitly not a committed deliverable. Natural candidates if F46/F39 testing makes them fall out: platform-split process discovery → `internal/proc`; operator state path/slugify → a new `internal/operator`; pane discovery/resolution → `internal/pane`. If nothing falls out naturally, skip entirely.

### Non-goals

- Re-implementing tests sibling batches already added (the refreshed numbers above define the actual gaps).
- Re-filing anything in the report's Refuted section (R1–R8) — notably the hooksync duplication between the two modules is a documented deliberate decision (kit-architecture.md, 260402-ktbg); do not "deduplicate" it while splitting sync.go.
- New CLI commands or signature changes (pure test/CI/structure batch; `_cli-fab.md` needs updates only if a signature unexpectedly changes).

## Affected Memory

- `distribution/distribution`: (modify) release workflow step list gains the test gate; `go-version-file` replaces the hardcoded Go version; "Go 1.22" mentions updated
- `distribution/kit-architecture`: (modify) "Go 1.22" mention updated; sync.go split reflected in the fab-kit sync description; statusman benchmark decision record becomes the sole survivor after `src/benchmark/` deletion
- `distribution/migrations`: (modify) "parseSemver/compareSemver helpers in sync.go" path reference updated after the split
- `pipeline/schemas`: (modify — conditional) only if the `stage_metrics` iterations investigation results in a behavior fix that changes documented semantics

## Impact

- `src/go/fab`: `internal/status` (new tests + possible stage_metrics fix), `cmd/fab` (new cobra-execution tests), `internal/resolve`/`internal/change`/`internal/statusfile`/`internal/backlog`/`internal/intake` (F47 deletions/consolidation), `go.mod` (go/cobra bump), possible new golden-test files for yaml byte-stability
- `src/go/fab-kit`: `internal/sync.go` split into 5 files, new Sync/cleanLegacyAgents integration tests, `go.mod` (go/cobra bump)
- `.github/workflows/ci.yml`: darwin cross-compile step, `-race`
- `.github/workflows/release.yml`: test gate, `go-version-file` (builds on dn2c's already-merged checksum changes)
- `src/benchmark/`: deleted except RESULTS.md + README.md
- `docs/memory/distribution/*`: three files per Affected Memory
- Constitution constraints in force: VII Test Integrity (tests conform to spec); changes to the Go binary MUST include test updates (this change *is* the test update); byte-stable index/status output preserved across any yaml change; `src/kit/` canonical (no skill files are touched here)

## Open Questions

*(none — all decision points have clear front-runners; see Assumptions)*

## Clarifications

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | — |
| 10 | Confirmed | — |
| 11 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Wave-3 dependency gate is satisfied; no rebase choreography needed | Verified in git log: #395–#402 all merged at branch base `47846c0b`; dn2c's release.yml seam dissolved by merge order | S:95 R:90 A:95 D:95 |
| 2 | Certain | Coverage targets are the refreshed numbers measured today (status 67.8%, fab-kit 67.1%, cmd/fab 55.2% + per-function list), not the report's 1431a9c3 baselines | Measured via `go tool cover -func` on this branch; siblings added tests after the report baseline — re-implementing them would be waste and churn | S:90 R:85 A:90 D:90 |
| 3 | Certain | F47 exclusions are resolved, not pending: `stage_hooks` stays (c5tr preserves it), `model_tiers` already dropped by c5tr's 2.1.6→2.2.0 migration | c5tr (#399) is merged; the migrations memory documents the decision — backlog's "coordinate" is satisfied by reading the outcome | S:85 R:90 A:95 D:90 |
| 4 | Certain | Go directive bumps to the current stable line (1.26.x) rather than minimum-supported | Clarified — user confirmed | S:95 R:90 A:75 D:65 |
| 5 | Certain | yaml.v3 default outcome: golden byte-stability tests land unconditionally; goccy/go-yaml swap happens only on proven byte-identical parity, else stay pinned + record archived status | Clarified — user confirmed | S:95 R:70 A:80 D:70 |
| 6 | Certain | stage_metrics iterations regression test is in F39 scope; if it exposes the observed reset-to-1 bug, the implementation fix rides in this change | Clarified — user confirmed | S:95 R:75 A:80 D:70 |
| 7 | Certain | hv7t's three recorded deletion candidates are absorbed into F47 | Clarified — user confirmed | S:95 R:90 A:85 D:75 |
| 8 | Certain | `resolve.ToDir`/`ToStatus`: delete (with their tests) rather than routing the command through them | Clarified — user confirmed | S:95 R:85 A:80 D:60 |
| 9 | Certain | `src/benchmark/`: delete all four implementations, keep RESULTS.md + README.md as the decision record | Clarified — user confirmed | S:95 R:75 A:80 D:70 |
| 10 | Certain | cobra bumps to the latest v1.10.x patch; ye8r's contract/collision drift tests are the regression guard for the CLI surface | Clarified — user confirmed | S:95 R:80 A:85 D:75 |
| 11 | Certain | F42 darwin cross-compile applies to both matrix legs, and `-race` is added to the linux test step | Clarified — user confirmed | S:95 R:95 A:85 D:80 |

11 assumptions (11 certain, 0 confident, 0 tentative, 0 unresolved).
