# Plan: Un-gitignore fab/.fab-version

**Change**: 260708-8ken-fab-version-gitignore-fix
**Intake**: `intake.md`

## Requirements

### Repo: This repo's own .gitignore

- **R1** — The repo's own `.gitignore` MUST stop ignoring `fab/.fab-version` by
  adding a negation line `!fab/.fab-version` immediately after the `.fab-*` line
  (currently `.gitignore:29`). SHALL leave every other line untouched.
  - GIVEN the repo `.gitignore` has `.fab-*` at line 29 and `fab/.fab-version`
    exists on disk, WHEN the negation is added, THEN
    `git check-ignore fab/.fab-version` exits non-zero (no longer ignored) and
    the file becomes trackable (`git add fab/.fab-version` succeeds). The git
    commit itself is OUT OF SCOPE for apply (ship owns git).

### Scaffold: fragment negation for all future + syncing projects

- **R2** — `src/kit/scaffold/fragment-.gitignore` MUST carry `!fab/.fab-version`
  directly under its `.fab-*` line, so every `fab sync` self-heals a project's
  `.gitignore` (the fragment is applied via `lineEnsureMerge` on every sync, not
  only at init — `sync.go:89` → `scaffoldTreeWalk` → `lineEnsureMerge`).
  - GIVEN a project `.gitignore` still ignoring `fab/.fab-version`, WHEN
    `fab sync` runs the fragment merge, THEN the negation is appended (once) and
    `fab/.fab-version` is no longer ignored.
- **R3** — The fragment negation MUST merge cleanly under the existing
  `lineEnsureMerge` semantics: `!fab/.fab-version` is a NON-directory token
  (`gitignoreIsDirectoryToken` is false — it has no leading `/`, no trailing `/`,
  no `*`), so it uses **strict literal dedup** and NEVER consults the Guardrail-B
  negation hard-stop (`gitignoreHasNegation`), which is gated on
  `gitignoreIsDirectoryToken(entry)`. Adding the negation therefore CANNOT
  suppress the `.fab-*` ensure (also a non-directory token, also strict literal).
  - GIVEN a fresh project `.gitignore` without the negation, WHEN the fragment
    merges, THEN `!fab/.fab-version` is appended exactly once.
  - GIVEN a re-sync of a project that already has the negation, WHEN the fragment
    merges again, THEN nothing is appended (idempotent).
  - GIVEN a project `.gitignore` carrying `.fab-*` but not the negation, WHEN the
    fragment (which ships BOTH `.fab-*` and `!fab/.fab-version`) merges, THEN
    only `!fab/.fab-version` is appended (`.fab-*` already covered by strict
    literal dedup) and the `.fab-*` line survives.

### Migration: 2.15.1-to-2.15.2 (verify + commit for already-shipped repos)

- **R4** — A new migration `src/kit/migrations/2.15.1-to-2.15.2.md` MUST exist,
  following the structure of the newest migration (`2.14.0-to-2.15.0.md`): a
  `## Summary`, `## Pre-check` (sentinel/idempotency), `## Changes`, and
  `## Verification`. Because the fragment merge self-heals the negation at sync
  time (R2), the migration's job is **verification + the commit the binary cannot
  do**.
  - GIVEN a repo whose `fab/.fab-version` is not-ignored AND already committed,
    WHEN the migration Pre-check runs, THEN it SKIPS silently (idempotent no-op).
  - GIVEN a repo that is not a git repository OR has no `fab/.fab-version`, WHEN
    the migration Pre-check runs, THEN it SKIPS silently (pre-2.15 repos reach
    this migration only after `2.14.0-to-2.15.0` has run).
  - GIVEN a repo where `fab/.fab-version` is still ignored (sync predated the
    fixed fragment), WHEN the migration Changes run, THEN `!fab/.fab-version` is
    appended after `.fab-*` (or at end) and `git add .gitignore fab/.fab-version`
    + commit lands both.
  - GIVEN the migration has run once, WHEN it re-runs, THEN it is a no-op
    (`git check-ignore fab/.fab-version` non-zero AND
    `git ls-files --error-unmatch fab/.fab-version` exit 0 ⇒ Pre-check trips).

### Stamp-path hardening (fail-open warning)

- **R5** — After `stampFabVersion` (`internal/init.go`) successfully writes
  `fab/.fab-version`, its callers (`Init`, `Upgrade`) MUST, when inside a git
  work tree AND `git check-ignore -q fab/.fab-version` reports the file ignored,
  print a fail-open warning to **stderr** advising the negation + commit (e.g.
  `fab: warning: fab/.fab-version is gitignored — commit it so worktrees/clones/CI
  see the version (add '!fab/.fab-version' to .gitignore)`). It MUST be a warning
  only (never an error, never a non-zero exit) and MUST be **silent** when git is
  absent, when not in a repo, or when the path is not ignored (fail-open, the rk
  fail-silent precedent).
  - GIVEN a repo whose `.gitignore` still ignores `fab/.fab-version`, WHEN
    stampFabVersion writes and the check runs, THEN the `fab: warning:` line is
    printed to stderr and the stamp still succeeds (no error).
  - GIVEN a repo whose `.gitignore` negates `fab/.fab-version`, WHEN the check
    runs, THEN nothing is printed.
  - GIVEN a directory that is not a git repository (or git absent), WHEN the check
    runs, THEN nothing is printed and no error surfaces.

### Version + docs obligations

- **R6** — `src/kit/VERSION` MUST bump `2.15.1` → `2.15.2` (patch — pure fix;
  matches the migration name FROM=released TO=next).
  - GIVEN the current VERSION `2.15.1`, WHEN the bump lands, THEN
    `cat src/kit/VERSION` prints `2.15.2`.
- **R7** — `docs/specs/config.md`'s `.fab-version` "committed" prose MUST gain the
  gitignore-negation caveat: the design says `.fab-version` is committed; the
  mechanism (the `.fab-*` ignore class + its required negation) now guarantees it.
  - GIVEN the "committed" mentions at config.md:118-119 and :297-298, WHEN the
    caveat is added, THEN the doc states the `.fab-*` class would otherwise ignore
    it and that a `!fab/.fab-version` negation is what makes it committable.
- **R8** — No CLI command signature changes, so `_cli-fab.md` gains NO command
  change. The stamp warning line is added to `_cli-fab.md`'s `upgrade-repo`/init
  output description ONLY IF that section enumerates output lines. (Verified: the
  `upgrade-repo`/init prose describes behavior but does NOT enumerate stamp-output
  lines, so `_cli-fab.md` is left unchanged — Assumption 3.)
  - GIVEN `_cli-fab.md` does not enumerate stamp-path output, WHEN R8 is applied,
    THEN `_cli-fab.md` is unchanged and no skill↔SPEC mirror sweep is triggered.

### Non-Goals

- No renaming of `fab/.fab-version` (churn on a just-shipped design — rejected in
  intake).
- No narrowing of `.fab-*` to `/.fab-*` (would change coverage for nested
  worktree roots and the fragment's dedup semantics — rejected in intake).
- No git commit performed during apply (ship owns git, including the
  `fab/.fab-version` add — block contract).

### Design Decisions

- **Negation is a non-directory token** — `!fab/.fab-version` is deliberately
  written without a leading `/` and without a trailing `/`, so
  `gitignoreIsDirectoryToken` returns false and it takes the strict-literal-dedup
  path. This is the same class as `.fab-*` and `.status.yaml.lock` (both already
  tested), so Guardrail-B never applies and the negation append is a simple
  append-once-if-absent. Verified against `scaffold.go:296,324` and a live
  evaluation of `gitignoreIsDirectoryToken`.
- **Warning helper is git-gated and fail-open** — mirrors the existing
  `gitRepoRoot()` shell-out style (`exec.Command("git", ...)`); any git error
  (not a repo, git absent) is swallowed and the warning is skipped. Never returns
  an error to the caller.

## Tasks

### Phase 1: Kit content — negations, migration, version

- [x] T001 [P] Add `!fab/.fab-version` to the repo's own `.gitignore` immediately after the `.fab-*` line (line 29). Verify `git check-ignore fab/.fab-version` exits non-zero afterward. Do NOT `git add`/commit (ship owns git). <!-- R1 -->
- [x] T002 [P] Add `!fab/.fab-version` to `src/kit/scaffold/fragment-.gitignore` directly under its `.fab-*` line (line 2). <!-- R2 -->
- [x] T003 [P] Bump `src/kit/VERSION` from `2.15.1` to `2.15.2`. <!-- R6 -->
- [x] T004 Create `src/kit/migrations/2.15.1-to-2.15.2.md` following the `2.14.0-to-2.15.0.md` structure (`## Summary`, `## Pre-check`, `## Changes`, `## Verification`). Pre-check: skip when `git check-ignore -q fab/.fab-version` fails (not ignored) AND `git ls-files --error-unmatch fab/.fab-version` succeeds (committed); skip silently when not a git repo or `fab/.fab-version` absent. Changes: append `!fab/.fab-version` after `.fab-*` (or at end) if still ignored, then `git add .gitignore fab/.fab-version` + commit. Verification: check-ignore non-zero, ls-files exit 0, re-run no-op. Note the fragment-self-heal rationale in the Summary. <!-- R4 -->

### Phase 2: Go — stamp-path warning helper + callers

- [x] T005 Add a git-gated, fail-open warning helper in `src/go/fab-kit/internal/init.go`: after a successful `stampFabVersion` write, when inside a git work tree AND `git check-ignore -q fab/.fab-version` reports the path ignored, print the `fab: warning: fab/.fab-version is gitignored — commit it so worktrees/clones/CI see the version (add '!fab/.fab-version' to .gitignore)` line to stderr. Never an error; silent when git absent / not a repo / path not ignored. Wire the check into both callers of `stampFabVersion` (`Init` at init.go:44, `Upgrade` at upgrade.go:145) — extract a helper so both share it. <!-- R5 -->

### Phase 3: Tests (alongside every Go change — constitution VII)

- [x] T006 Add fragment-merge tests to `src/go/fab-kit/internal/scaffold_test.go` pinning R2/R3: (a) append-once on a fresh `.gitignore`; (b) idempotent re-merge when the negation is already present; (c) merging the full fragment (`.fab-*` + `!fab/.fab-version`) onto a `.gitignore` that already has `.fab-*` appends only the negation and preserves `.fab-*`; (d) no Guardrail-B interaction — a present `!fab/.fab-version` negation does not hard-stop and `.fab-*` is still ensured. Follow the existing `TestLineEnsureMerge_*` patterns. <!-- R3 -->
- [x] T007 Add warning-path tests near `internal/init_test.go` (or a focused new test) for R5: warning fires when `fab/.fab-version` is ignored (init a temp git repo with a `.gitignore` containing `.fab-*`; assert the stderr line); silent when the `.gitignore` negates it; silent outside a git repo. Capture stderr to assert presence/absence. <!-- R5 -->

### Phase 4: Docs

- [x] T008 [P] Add the gitignore-negation caveat to `docs/specs/config.md`'s `.fab-version` "committed" prose (near lines 118-119 and 297-298): the `.fab-*` ignore class would otherwise swallow `fab/.fab-version`, so a `!fab/.fab-version` negation is what makes the "committed" guarantee real. <!-- R7 -->
- [x] T009 [P] Verify `src/kit/skills/_cli-fab.md`'s `upgrade-repo`/init sections do not enumerate stamp-path output lines; leave unchanged if so (no signature change). Record the outcome. <!-- R8 -->

### Phase 5: Full-suite validation

- [x] T010 Run `cd src/go/fab-kit && go test ./... -count=1 && go vet ./...`; run the `src/go/fab` suites too (VERSION/docs sweeps can touch shared fixtures). Ensure `gofmt -l .` is empty in BOTH Go modules (the previous PR failed CI on exactly this). Fix any failure at root cause. <!-- R3 R5 R6 -->

## Execution Order

- Phase 1 tasks T001–T004 are independent `[P]` (different files).
- T005 (Go helper) precedes T007 (its tests). T006 depends on nothing but the
  fragment change T002 makes no code change, so T006 can be written against the
  existing `lineEnsureMerge` behavior directly.
- T010 (full-suite) runs LAST, after all code + test changes land.

## Acceptance

### Functional Completeness

- [x] A-001 R1: The repo's `.gitignore` carries `!fab/.fab-version` right after `.fab-*`, and `git check-ignore fab/.fab-version` exits non-zero (file trackable). No other line changed. *(review: verified empirically — `check-ignore -q` exit 1, `git add --dry-run` succeeds, diff is exactly +1 line)*
- [x] A-002 R2: `src/kit/scaffold/fragment-.gitignore` carries `!fab/.fab-version` directly under `.fab-*`.
- [x] A-003 R4: `src/kit/migrations/2.15.1-to-2.15.2.md` exists with `## Summary`/`## Pre-check`/`## Changes`/`## Verification` sections matching the newest migration's shape, encoding the verify+commit + fragment-self-heal rationale.
- [x] A-004 R5: `stampFabVersion`'s callers emit the fail-open `fab: warning:` line to stderr when `fab/.fab-version` is ignored, and are silent otherwise. *(review: `warnIfFabVersionIgnored` wired at init.go:48 + upgrade.go:148; 4 tests pass)*
- [x] A-005 R6: `src/kit/VERSION` reads `2.15.2`.
- [x] A-006 R7: `docs/specs/config.md` states the `.fab-*` ignore caveat and the `!fab/.fab-version` negation requirement alongside the "committed" prose. *(review: caveat at config.md:300-308; the plan's other cited location :118-119 has no "committed" claim — the Change-3 section is the sole one, covered)*
- [x] A-007 R8: `_cli-fab.md` is unchanged (no enumerated stamp output; no signature change), decision recorded.

### Behavioral Correctness

- [x] A-008 R3: The fragment negation appends exactly once on a fresh `.gitignore`, is idempotent on re-merge, and does not suppress the `.fab-*` ensure (Guardrail-B not consulted for either non-directory token) — pinned by tests. *(review: 4 `TestLineEnsureMerge_FabVersion*` tests pass; token claims re-verified against `gitignoreIsDirectoryToken`/`gitignoreHasNegation` in scaffold.go; end-to-end git simulation of an old-fragment repo confirms un-ignore + commit + fresh clone resolves the version)*
- [x] A-009 R5: The stamp warning is git-gated and fail-open — silent outside a git repo or when the path is negated; never returns an error or non-zero exit.

### Edge Cases & Error Handling

- [x] A-010 R4: The migration Pre-check skips silently for non-git repos and repos without `fab/.fab-version`, and is a complete no-op on re-run. *(review: walked against fresh-post-2.15.2, pre-fix-synced, and already-fixed repo states)*
- [x] A-011 R5: A `git check-ignore` failure (git absent / not a repo) is swallowed — the stamp still succeeds. *(review: also probed — a tracked-but-pattern-matched file yields exit 1, so no spurious warning on an already-committed file)*

### Code Quality

- [x] A-012 Pattern consistency: the warning helper follows the existing `internal/` shell-out style (`exec.Command("git", ...)`, `gitRepoRoot()` precedent); the new tests follow the existing `TestLineEnsureMerge_*` / `TestStampFabVersion_*` patterns.
- [x] A-013 No unnecessary duplication: the warning check is a single shared helper wired into both `Init` and `Upgrade`, not copy-pasted. *(review: `captureStderr` is the stderr sibling of the existing `captureStdout` in upgrade_test.go — different stream, not a duplication)*

### Documentation Accuracy (checklist.extra_categories)

- [x] A-014 The `docs/specs/config.md` caveat is accurate: `.fab-*` is unanchored and matches at any depth (hence swallows `fab/.fab-version`), and `!fab/.fab-version` is the negation that un-ignores it.

### Cross-References (checklist.extra_categories)

- [x] A-015 The migration `2.15.1-to-2.15.2.md`, `docs/specs/config.md`, and the fragment stay mutually consistent on the negation string `!fab/.fab-version` and the version `2.15.2`.

### Test Integrity & Build

- [x] A-016 `cd src/go/fab-kit && go test ./... -count=1 && go vet ./...` pass; `src/go/fab` suites pass; `gofmt -l .` is empty in both Go modules. *(review: both suites re-run green with -count=1; vet clean; gofmt empty in both modules)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Warning text = `fab: warning: fab/.fab-version is gitignored — commit it so worktrees/clones/CI see the version (add '!fab/.fab-version' to .gitignore)` | Intake §4 gives this exact wording as the example; `fab: warning:` prefix matches the config-cascade precedent (config.md:125) | S:85 R:85 A:90 D:85 |
| 2 | Certain | Warning helper is a single shared function wired into both `Init` and `Upgrade`, git-gated via `git check-ignore -q`, swallowing any git error | Intake §4 + constitution DRY; `gitRepoRoot()` is the existing shell-out precedent; fail-open is the stated discipline | S:85 R:80 A:90 D:85 |
| 3 | Certain | `_cli-fab.md` left unchanged (its upgrade-repo/init prose does not enumerate stamp-output lines) | Verified by grep of `_cli-fab.md` §upgrade-repo/init: behavior prose only, no per-line output enumeration; intake §5 makes the edit conditional on enumeration | S:80 R:85 A:90 D:85 |
| 4 | Certain | Migration follows `2.14.0-to-2.15.0.md` structure (Summary/Pre-check/Changes/Verification) | It is the newest existing migration and the intake §3 shape maps onto it directly | S:85 R:80 A:95 D:90 |
| 5 | Confident | Warning-path tests init a real temp git repo (via the existing `requireGit`/`git init` test helpers) to exercise `git check-ignore` | init_test.go already uses `requireGit(t)` + `exec.Command("git","init",dir)`; check-ignore needs a real repo, so a temp git repo is the faithful fixture | S:70 R:85 A:85 D:80 |
| 6 | Confident | Fragment-merge tests live in `scaffold_test.go` alongside the existing `TestLineEnsureMerge_*` cases (not a new file) | Constitution VII (tests alongside); the existing gitignore-dedup tests are the sibling cluster | S:75 R:85 A:85 D:80 |

6 assumptions (4 certain, 2 confident, 0 tentative).

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (Purely additive: negation lines, a warning helper with two live call sites, a new migration, tests, and a spec caveat. The `.fab-*` pattern, `lineEnsureMerge` semantics, and both prior migrations remain load-bearing.)
