# Plan: fab sync: distinguishable "not a fab-managed repo" exit code

**Change**: 260705-52i9-sync-distinguishable-unmanaged-exit
**Intake**: `intake.md`

## Requirements

### Exit Codes: `fab sync` not-a-managed-repo signal

#### R1: Distinct exit code for the unmanaged-repo outcome
When `fab sync` runs outside a fab-managed repo (i.e. `internal.ResolveConfig()` walks to the filesystem root and returns `(nil, nil)`), the `fab-kit` binary SHALL exit with a distinct, documented, non-1 exit code — `3` — after printing its existing `not in a fab-managed repo. Run 'fab init' to set one up` message to stderr. It SHALL NOT collapse this outcome to the generic exit 1 that `main()` returns for any other `RunE` error. This SHALL hold **regardless of whether the current directory is inside a git repository** — the unmanaged-repo check is a `fab/project/config.yaml` walk-up, independent of git.

- **GIVEN** a directory that has no `fab/project/config.yaml` on any ancestor path
- **WHEN** `fab sync` is invoked there
- **THEN** the process prints `not in a fab-managed repo. Run 'fab init' to set one up` to stderr
- **AND** exits with code `3` (not `0`, not `1`)

<!-- rework (cycle 2, fix-code): review found Sync() ran gitRepoRoot() BEFORE
     RequireManagedRepo(), so a non-git unmanaged directory exited 1, not 3 — silently
     violating this scenario and diverging from migrations-status (which has no git
     precondition and correctly exits 3 in the same case). Added as an explicit scenario
     below to pin the fix and prevent regression. -->

- **GIVEN** a directory that has no `fab/project/config.yaml` on any ancestor path AND is not
  inside a git repository
- **WHEN** `fab sync` is invoked there
- **THEN** the process still prints the unmanaged message and exits `3` — NOT `1` — matching
  `fab-kit migrations-status`'s behavior in the same directory

#### R2: Real sync failures remain exit 1 (unchanged)
Genuine sync failures — a corrupt/unparseable `config.yaml`, a missing `fab_version`, a failed scaffold/deploy write, a version-guard trip — SHALL continue to return a normal `error` from `Sync()` that falls through to `main()`'s blanket `os.Exit(1)`. Their behavior is unchanged by this change.

- **GIVEN** a fab-managed repo whose `config.yaml` cannot be parsed (or any other genuine failure)
- **WHEN** `fab sync` is invoked
- **THEN** the process exits with code `1` (the pre-existing generic-failure path), NOT `3`

#### R3: Exit code is a named constant, not a magic number
The numeric value `3` SHALL be defined as a single exported named constant in `internal` (so documentation and any future in-repo consumer code reference the symbol, not a literal). It SHALL carry a doc comment identifying it as the "not a fab-managed repo" exit code.

- **GIVEN** the codebase after this change
- **WHEN** the unmanaged-repo exit code is referenced (in the handler, in tests, in docs)
- **THEN** it is referenced via the named constant, and the literal `3` appears at exactly one definition site
- **AND** a bare-number anti-pattern (Code Quality) is avoided

### Consolidation: shared unmanaged-repo guard

#### R4: Single shared helper for the `cfg == nil` guard
The duplicated `ResolveConfig()` + `if cfg == nil { return fmt.Errorf("not in a fab-managed repo...") }` block SHALL be consolidated into one shared helper in `internal` (e.g. `requireManagedRepo() (*ConfigResult, error)`) that both `internal.Sync()` and `cmd/fab-kit`'s `runMigrationsStatus` call. The helper SHALL apply the R1 distinct-exit-code behavior on the nil case and propagate a genuine `ResolveConfig` error unchanged (R2).

- **GIVEN** `internal.Sync()` (its `kitVersion == ""` branch) and `cmd/fab-kit`'s `runMigrationsStatus`
- **WHEN** each needs to require a managed repo
- **THEN** both call the same shared helper rather than re-implementing the check
- **AND** the copy-pasted `fmt.Errorf("not in a fab-managed repo...")` literal no longer appears in `sync.go` or `migrations_status.go`

#### R5: `upgrade.go` is left unchanged
`internal.Upgrade`'s more elaborate variant (which tolerates a `config.yaml` missing its `fab_version` field — a partially-managed, not fully-unmanaged, semantic) SHALL NOT be modified by this change. Its two `not in a fab-managed repo` returns stay as-is.

- **GIVEN** `internal/upgrade.go`
- **WHEN** this change is applied
- **THEN** `upgrade.go` is byte-for-byte unchanged (its distinct missing-version handling is out of scope, noted as a future follow-up)

### Documentation: `_cli-fab.md` exit-code contract

#### R6: `_cli-fab.md`'s sync exit-code contract is updated (apply-stage, not hydrate)
`src/kit/skills/_cli-fab.md` currently documents `fab sync`'s exit behavior as a plain
"non-zero is failure" contract (the prose at its General Conventions section and the `sync` row
in its command table). This is now stale — exit `3` is a distinguishable "not applicable" signal,
not a failure — and `_cli-fab.md` is **kit source**, not `docs/memory/` (the hydrate-owned
free-form memory tier). Per the constitution's Additional Constraints ("Changes to the `fab`
CLI… MUST update `src/kit/skills/_cli-fab.md`") and code-review.md's "CLI ⇒ docs + tests" rule,
this update is an **apply-stage obligation that R1–R5 missed**, not a hydrate deferral.

The update SHALL:
- Add an exit-code note to the `fab sync` row (or its surrounding prose) documenting `3` =
  "not a fab-managed repo" (distinct from the generic `1` = failure), mirroring the existing
  branchable-exit-code precedent already in the file for the pane family (2/3 scheme).
- Add the same exit-`3` note to the `fab-kit migrations-status` row/section, since
  `RequireManagedRepo()` is shared by both commands (R4).
- Clarify that `fab upgrade-repo` outside a managed repo is **unaffected** by this change — it
  still exits `1` with the same stderr message (`internal/upgrade.go`, R5's untouched code path)
  — so a reader does not over-generalize the new exit-3 contract to every "unmanaged repo" case.
- Carry the matching update into `docs/specs/skills/SPEC-_cli-fab.md` in the same change (the
  constitution-required SPEC mirror — code-quality.md § Sibling & Mirror Sweeps treats a skill
  change without its SPEC mirror as must-fix).

- **GIVEN** `src/kit/skills/_cli-fab.md` after this change
- **WHEN** a reader looks up `fab sync`'s or `fab-kit migrations-status`'s exit-code behavior
- **THEN** the distinct exit-`3` ("not a fab-managed repo") contract is documented, distinguished
  from the generic exit-`1` failure path
- **AND** `fab upgrade-repo`'s unaffected exit-1 behavior in the same scenario is noted so the two
  are not conflated
- **AND** `docs/specs/skills/SPEC-_cli-fab.md` reflects the same update (SPEC-mirror sync)

### Non-Goals

- No new `--if-managed` flag or any new CLI flag surface (Assumption 1) — the distinct exit code fully satisfies the stated need.
- No re-tiering of other `fab-kit` failure exit codes (Assumption 4).
- No change to `internal.Upgrade` (R5).
- No `docs/memory/` edits during apply — memory documentation of the new exit-code contract is a **hydrate**-stage concern per the fab-continue skill (Key Properties: "Modifies `docs/memory/`? Yes — during hydrate") and the constitution (Docs Are Source of Truth, hydrated post-implementation). The intake's Affected Memory entries (`distribution/kit-architecture`, `distribution/distribution`, and the candidate `distribution/exit-codes`) are carried forward for hydrate to action.

### Design Decisions

1. **Distinct exit code over an `--if-managed` flag**: reuses the proven `os.Exit(N)`-in-handler precedent (`fab` binary's `pane_window_name.go` code 3, `memory_index.go` code 2) — *Why*: idiomatic, no new API surface, and the backlog item frames the exit-code route as what lets fab "stay authoritative long-term" — *Rejected*: a `--if-managed` no-op flag (new flag to learn/maintain, no existing pattern).
2. **Exit code `3`**: verified during apply — `fab-kit`'s `cmd/fab-kit/main.go` exits `1` uniformly for `RunE` errors; no existing fixed non-1 exit-code registry in the `fab-kit` binary; `3` collides only theoretically with `doctor`'s dynamic `os.Exit(failures)` (0–7 failure count), a different command with an unambiguous diagnostic semantic — *Why*: matches the intake's proposed value and the `fab`-binary tier-3 convention — *Rejected*: `2` (reserved by the `fab` binary's pane/memory-index tier-2 "specific-recoverable" semantic; keeping `3` = "environment/precondition not met" reads consistently).
3. **Helper lives in `internal`, does the `os.Exit` itself**: `ResolveConfig`/`ConfigResult` live in `internal`; both callers (`internal.Sync`, `cmd/fab-kit.runMigrationsStatus`) can reach an exported `internal` helper. The `os.Exit(3)` sits in the thin helper (mirroring the untested thin `os.Exit` wrappers in `memory_index.go`/`doctor.go`); the *constant value* is pinned by a direct unit test (mirroring `TestTmuxExitCode`/`TestPaneValidationExitCode`) — *Why*: `Sync()` returns an `error` to `main()`, so a returned error would collapse to exit 1; the exit must happen in-handler — *Accepted trade-off (corrected on rework)*: a sentinel-error alternative (`var ErrNotManaged`, mapped once in each binary's single `Execute()`-error funnel via `errors.Is`) is not actually multi-site — the router uses `syscall.Exec` (no error-mapping in the router itself) and each binary has exactly one funnel — so it would in fact be a single mapping site per binary and would make the exit branch unit-testable. Kept the in-handler `os.Exit` anyway for this change (matches the existing untested-thin-wrapper precedent and avoids widening scope further after two rework cycles); flagged as a candidate for a future follow-up, not blocking here.

## Tasks

### Phase 1: Core Implementation

- [x] T001 Add exported `ExitNotManaged = 3` constant with a doc comment (referencing the not-a-fab-managed-repo contract) and a shared `RequireManagedRepo() (*ConfigResult, error)` helper to `src/go/fab-kit/internal/config.go` — helper calls `ResolveConfig()`, returns `(nil, err)` on a real error, prints the existing stderr message + `os.Exit(ExitNotManaged)` on `cfg == nil`, else returns `(cfg, nil)` <!-- R3 --> <!-- R4 -->
- [x] T002 Replace the inline `ResolveConfig()` + `cfg == nil` block in `internal.Sync()` (`src/go/fab-kit/internal/sync.go`, the `kitVersion == ""` branch) with a call to `RequireManagedRepo()` <!-- R1 --> <!-- R4 -->
- [x] T003 Replace the inline `ResolveConfig()` + `cfg == nil` block in `runMigrationsStatus` (`src/go/fab-kit/cmd/fab-kit/migrations_status.go`) with a call to `internal.RequireManagedRepo()` <!-- R4 -->

### Phase 2: Tests

- [x] T004 [P] Add a unit test pinning the exit-code scheme: `ExitNotManaged == 3` and that `RequireManagedRepo` returns the config unchanged when a managed repo exists and propagates a genuine `ResolveConfig` error, in `src/go/fab-kit/internal/config_test.go` (the nil→`os.Exit` path is the untested thin wrapper, per the `memory_index`/`doctor` precedent) <!-- R1 --> <!-- R2 --> <!-- R3 -->

### Phase 3: Documentation (rework — added after review must-fix)

<!-- rework: review flagged _cli-fab.md's sync exit-code contract as stale (still reads
     "non-zero = failure"); this is kit source, an apply-stage obligation R1-R5 missed, not a
     hydrate deferral. Also requires the constitution-mandated SPEC-_cli-fab.md mirror update. -->

- [x] T005 Update `src/kit/skills/_cli-fab.md`: add the exit-`3` ("not a fab-managed repo")
      note to the `fab sync` row/prose and the `fab-kit migrations-status` row, and clarify that
      `fab upgrade-repo` is unaffected (still exits `1` in the same scenario) <!-- R6 -->
- [x] T006 Mirror the same update into `docs/specs/skills/SPEC-_cli-fab.md` (constitution-required
      SPEC sync — code-quality.md § Sibling & Mirror Sweeps) <!-- R6 -->

### Phase 4: Fix code (rework cycle 2 — must-fix)

<!-- rework: review found Sync() checks gitRepoRoot() before RequireManagedRepo(), so a
     non-git unmanaged directory exits 1 instead of 3 — violating R1's new scenario and
     diverging from migrations-status (no git precondition, correctly exits 3). Also folds
     in the should-fix doc-precision gap the same reviewers raised on the same lines. -->

- [x] T007 Reorder `internal.Sync()` (`src/go/fab-kit/internal/sync.go`) so the
      `RequireManagedRepo()` check runs BEFORE `gitRepoRoot()` in the `kitVersion == ""` branch —
      the managed-repo walk-up does not depend on git, so it should gate first; a non-git,
      unmanaged directory must now exit `3`, not `1` <!-- R1 -->
- [x] T008 Correct `src/kit/skills/_cli-fab.md` (sync row + General Conventions exception note)
      and `docs/specs/skills/SPEC-_cli-fab.md` to drop the now-inaccurate implication that the
      git-repo-root check could interfere with the exit-3 contract (moot after T007 — sync and
      migrations-status are symmetric); also fix the "mirroring the pane family's 2/3 scheme"
      wording (the pane family's 2 is benign and 3 is the failure — this change inverts that, no
      2 involved) — reviewer nice-to-have folded in while touching these lines <!-- R6 -->

## Execution Order

- T001 blocks T002, T003, T004 (they all reference the new symbol)
- T002 and T003 are independent of each other once T001 lands
- T004 depends on T001
- T005 and T006 are independent of T001-T004 (documentation only); T006 depends on T005 (mirror
  follows the source update)
- T007 depends on T002 (reorders code T002 introduced); T008 depends on T007 (doc must describe
  the corrected behavior) and on T005/T006 (edits the same doc sections)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab sync` in a non-fab directory prints the unmanaged message to stderr and exits `3` (verified via the pinned constant + handler wiring), **including when the directory is not inside a git repository** (empirically probed with a built binary, not just unit-tested — this is the exact scenario the must-fix violated)
- [x] A-002 R3: The literal `3` for this outcome is defined once as `internal.ExitNotManaged` with a doc comment; no bare `3` at the call site
- [x] A-003 R4: Both `internal.Sync()` and `runMigrationsStatus` call the shared `requireManagedRepo`/`RequireManagedRepo` helper; the `not in a fab-managed repo` literal no longer appears in `sync.go` or `migrations_status.go`

### Behavioral Correctness

- [x] A-004 R2: Genuine sync failures (corrupt config, failed writes, version-guard trip) still return an `error` → exit 1, unchanged
- [x] A-005 R5: `internal/upgrade.go` is unmodified by this change

### Scenario Coverage

- [x] A-006 R3: A unit test asserts `ExitNotManaged == 3` (the scheme is pinned, mirroring `TestTmuxExitCode`); the fab-kit `internal` and `cmd/fab-kit` packages build and their tests pass

### Code Quality

- [x] A-007 Pattern consistency: New code follows the surrounding error-message style, doc-comment style, and the established `os.Exit(N)`-in-handler / testable-classifier precedent
- [x] A-008 No unnecessary duplication: The `cfg == nil` guard is reused via one helper, not re-implemented; magic-number anti-pattern avoided (R3)

### Documentation Accuracy

- [x] A-009 **N/A** (hydrate-owned): `docs/memory/distribution/kit-architecture.md` + `distribution.md` exit-code contract documentation is a hydrate-stage task, not apply — carried in intake Affected Memory
- [x] A-011 R6: `src/kit/skills/_cli-fab.md` documents the exit-`3` contract for `fab sync` and `fab-kit migrations-status` as unconditional (no git-repo caveat needed, now that T007 makes the two commands symmetric), distinguished from the generic exit-1 failure path, and notes `fab upgrade-repo` is unaffected

### Cross References

- [x] A-010 **N/A** (hydrate-owned): the canonical exit-code doc location (new `distribution/exit-codes.md` vs. a section in `kit-architecture.md`) is an open question deferred to hydrate per the intake
- [x] A-012 R6: `docs/specs/skills/SPEC-_cli-fab.md` mirrors the corrected `_cli-fab.md` exit-code update (constitution-required SPEC sync)
- [x] A-013 R1: A non-git, unmanaged directory produces exit `3` from BOTH `fab sync` and `fab-kit migrations-status` — no asymmetry between the two commands

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (The two inline `cfg == nil` guard blocks it superseded were deleted within this same diff; `upgrade.go`'s missing-version variant is a different semantic kept per R5; the only code made retirable is `wt`'s out-of-repo interim ResolveConfig-mirroring probe, the intake's intended beneficiary.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Solve via a distinct documented exit code, not a new `--if-managed` flag | Proven `os.Exit(N)`-in-handler precedent exists (`pane_window_name.go`, `memory_index.go`); a flag adds new API surface with no existing pattern; backlog frames exit-code as the "fab stays authoritative" route | S:60 R:70 A:80 D:65 |
| 2 | Confident | Use exit code `3` | Grep-verified during apply: `cmd/fab-kit/main.go` exits `1` uniformly; no fixed non-1 exit-code registry in the `fab-kit` binary; `3` matches the `fab`-binary tier-3 convention and collides only theoretically with `doctor`'s dynamic 0–7 count (different command, unambiguous semantic). Upgraded from the intake's Tentative now that the grep confirms no fixed-code collision | S:55 R:75 A:65 D:55 |
| 3 | Confident | Consolidate the `cfg == nil` check in `migrations_status.go` (not `upgrade.go`); place the shared helper + constant in `internal/config.go` alongside `ResolveConfig`/`ConfigResult` | Same duplicated logic sits in both callers; `internal` is reachable from both packages; `upgrade.go`'s missing-version variant is a different semantic and left alone | S:65 R:70 A:75 D:60 |
| 4 | Confident | Real sync failures keep returning a normal `error` → exit 1, unchanged | Backlog only asks to distinguish the "not managed" case; re-tiering all failure codes would be scope creep | S:70 R:75 A:80 D:75 |
| 5 | Confident | The `nil → os.Exit(3)` path is not unit-tested; only the constant value + non-nil/real-error paths are | Mirrors the `memory_index.go`/`doctor.go` precedent where the thin `os.Exit` wrapper is untested and the exit-code *decision* (`tmuxExitCode`, tier map) is unit-tested; `os.Exit` inside a test kills the process | S:60 R:70 A:70 D:65 |
| 6 | Confident | `docs/memory/` updates deferred to hydrate, not done in apply | fab-continue Apply Behavior never touches `docs/memory/` (Key Properties table + Hydrate Behavior Step 4 own it); constitution treats memory as post-implementation source of truth | S:70 R:80 A:80 D:75 |
| 7 | Confident | `_cli-fab.md` (+ its SPEC-_cli-fab.md mirror) is an apply-stage obligation, unlike `docs/memory/` | Review must-fix: `_cli-fab.md` is kit source (constitution: CLI changes MUST update it), not the hydrate-owned free-form `docs/memory/` tier — R6/T005/T006 added on rework, distinct from Assumption 6's memory deferral | S:65 R:75 A:80 D:70 |
| 8 | Confident | Fix the `Sync()` ordering bug (git-root check before managed-repo check) via a code reorder, not by narrowing R1's contract to "inside a git repo only" | Cycle-2 must-fix from both reviewers: `migrations-status` has no git precondition and already exits 3 correctly in a non-git unmanaged dir, so symmetry with the simpler command is the natural fix; narrowing the contract would weaken it for exactly the named consumers (wt/hop probing arbitrary directories) | S:70 R:75 A:80 D:70 |
| 9 | Confident | `docs/memory/distribution/migrations.md`'s stale "non-zero only on genuine error" claim is added to the hydrate carry-forward, not fixed at apply | Same `docs/memory/` hydrate-deferral rule as Assumption 6 — this file was simply missed from the intake's original Affected Memory list (outward review should-fix #3); corrected here by widening the carry-forward, not by touching `docs/memory/` during apply | S:60 R:75 A:75 D:70 |

9 assumptions (0 certain, 9 confident, 0 tentative).
