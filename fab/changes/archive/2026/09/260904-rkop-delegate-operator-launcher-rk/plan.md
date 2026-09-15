# Plan: Delegate Operator Launcher to rk

**Change**: 260904-rkop-delegate-operator-launcher-rk
**Intake**: `intake.md`

## Requirements

### Operator Launcher: rk delegation

#### R1: Bare `fab operator` delegates to `rk operator` when capable
When `rk` is on PATH and its `operator` subcommand is capable (R2's probe), bare `fab operator` MUST hand the entire launch to `rk operator`, replacing the fab process (exec), passing `--workers <v>` through as an argv flag when (and only when) the flag was supplied. The `--workers` value rides **argv, not env** on the delegated path — rk composes `FAB_AGENT_WORKERS` for the launched agent itself.

- **GIVEN** an rk on PATH whose `rk operator --help` exits 0 and documents `--workers`
- **WHEN** `fab operator` (bare, no subcommand) runs
- **THEN** fab execs `rk operator` (argv appends `--workers <v>` iff the flag was set) and rk's own output, preconditions, and exit status are the user's outcome
- **AND** the fab-side `$TMUX` precheck is skipped on this path (rk enforces its own preconditions; double-owning the check duplicates the error surface)

#### R2: Capability probe — side-effect-free, fail-open on absence
The probe MUST NOT create windows or touch tmux state, and MUST fail open: `exec.LookPath("rk")` succeeds AND `rk operator --help` exits 0 AND its output contains the token `--workers` (the flag fab passes through is the capability discriminant — the binary is probed, never the version string, per the `gate_rk.go` `rkSentinelToken` precedent; bare exit-0 could mask an rk whose `operator` predates the flag).

- **GIVEN** rk absent from PATH, or present without a capable `operator` subcommand
- **WHEN** `fab operator` runs
- **THEN** the probe answers false **silently** (no warning, no error) and today's launcher runs unchanged (cli-layering delegation rule 2: absence degrades, never errors)

#### R3: No fallback after a passing probe
Once the probe passes, fab MUST NOT fall back to its own launcher. With a true exec the only fab-visible failure is the exec syscall itself; that error is returned, not swallowed.

- **GIVEN** the probe passed
- **WHEN** the exec of `rk operator` fails (syscall error)
- **THEN** fab surfaces the error and stops — no fab-launcher retry (a partial rk launch may have mutated tmux state; retrying risks a duplicate operator window)

#### R4: Fallback path and subcommand family unchanged
The rk-absent path stays byte-identical: `$TMUX` check, exact server-wide singleton match, cwd resolution, `operatorSpawnCommand` profile resolution, `withWorkersEnv` prefix, `spawn.WithShellFallback`. The operator subcommand family (`tick-start`/`time`/`state`/`enroll`/`update`/`remove`/`note`/`watch`/`autopilot`/`branch-map`) is untouched — only the bare `RunE` grows the delegation branch.

- **GIVEN** rk absent
- **WHEN** `fab operator` / `fab operator tick-start` / any subcommand runs
- **THEN** behavior and output are exactly today's (existing tests pass unmodified)

#### R5: Help text points at rk operator
The cobra command help MUST state the delegation: when run-kit is installed and capable, the launch is handed to `rk operator` (role-marked window, provider-agnostic typed kickoff); today's launcher is the rk-absent fallback.

- **GIVEN** `fab operator --help`
- **WHEN** rendered
- **THEN** the text names `rk operator` as the delegated launcher and the built-in launcher as the fallback

#### R6: Docs updated (CLI ⇒ docs + tests)
`src/kit/skills/_cli-fab.md` § fab operator MUST document the delegation branch first (probe, argv pass-through and rk's charset validation delta on `--workers`, no-fallback-after-probe), then the existing launcher text explicitly labeled as the rk-absent fallback. `src/kit/skills/fab-operator.md`'s launcher sentence and §9 Key Properties launcher rows note the delegation as pointers to `_cli-fab.md` § fab operator (owner-or-pointer — no restating).

- **GIVEN** the docs sweep class (`_cli-fab.md` owner, `fab-operator.md` pointers)
- **WHEN** the change ships
- **THEN** every launcher-behavior claim site reflects delegation-first, fallback-second

#### R7: Tests cover the delegation seams
Probe and exec are injectable package-level vars (the `execAgent` / `rkPanesRunner` precedent). Tests MUST cover: probe-positive delegates with correct argv (with and without `--workers`), probe-negative falls through to the existing launcher, the pure help-token decision helper, and exec-error propagation.

- **GIVEN** stubbed probe/exec seams
- **WHEN** `go test ./cmd/fab` runs in `src/go/fab`
- **THEN** all delegation cases pass without a live rk or tmux

### Non-Goals

- No porting of rk's role-option probe, `@rk_win_role` stamping, or typed-kickoff composite into fab (rule 1 — that would reimplement rk's layer).
- No config knob to disable the delegation (capability presence IS the policy, matching every other fab→rk delegation).
- No change to `fab batch new`/`switch` spawn paths (they open worker windows, not the operator).

### Design Decisions

#### Probe discriminant is the `--workers` token in `rk operator --help`
**Decision**: capability = LookPath + `rk operator --help` exit 0 + output containing `--workers`.
**Why**: the flag fab passes through is exactly the capability fab needs; probing the binary (not the version string) follows `gate_rk.go`'s `rkSentinelToken` precedent and a bottle can predate a same-version source change.
**Rejected**: attempt-is-the-probe (pane-map style) — the launcher has side effects, so a failed attempt could leave a half-created window; bare exit-0 — could mask an early `rk operator` lacking `--workers`.
*Introduced by*: 260904-rkop-delegate-operator-launcher-rk

#### True process replacement via `syscall.Exec` behind an injectable seam
**Decision**: the delegated launch execs rk directly (resolved path, argv `["rk", "operator", ...]`, `os.Environ()` untouched) through a package-level var seam.
**Why**: matches `agent.go`'s `var execAgent = syscall.Exec` precedent; the user's terminal talks to rk with no fab middleman, exit codes propagate by construction, and the seam keeps it unit-testable.
**Rejected**: subprocess with stdio inheritance — adds a wait/propagate layer for zero benefit here; there is no post-delegation fab work.
*Introduced by*: 260904-rkop-delegate-operator-launcher-rk

## Tasks

### Phase 2: Core Implementation

- [x] T001 In `src/go/fab/cmd/fab/operator.go`: add the capability probe (injectable `rkOperatorProbe` var: `exec.LookPath("rk")` + `rk operator --help` via `pane.RunCmd` + pure token-check helper on `--workers`) and the delegation branch at the top of `runOperator` (before the `$TMUX` check): on probe pass, build argv (`operator`, plus `--workers <v>` iff `workersOverride` says set) and exec rk via an injectable `execOperator = syscall.Exec` seam with `os.Environ()`; return the exec error on failure. Update the cobra `Short`/`Long` to name the delegation and fallback. <!-- R1, R2, R3, R5 -->
- [x] T002 In `src/go/fab/cmd/fab/operator_test.go`: tests for the pure token helper, a real-probe test against stubbed `rk` scripts on an isolated PATH (capable / token-missing / absent), probe-positive exec argv (with/without `--workers`), probe-negative fall-through (existing launcher path entered — assert via the `$TMUX` error when unset), and exec-error propagation. Add a `stubNoRK(t)` helper and apply it to the three pre-existing bare-launcher tests — they run on the host PATH, where a capable installed rk would otherwise fire the delegation branch and `syscall.Exec` would replace the test process. Run `go test ./cmd/fab` in `src/go/fab`. <!-- R7 -->

### Phase 4: Polish

- [x] T003 [P] `src/kit/skills/_cli-fab.md` § fab operator: prepend the delegation paragraph (probe, argv pass-through + rk charset-validation delta, no-fallback-after-probe, skipped fab-side `$TMUX` check), label the existing launcher text as the rk-absent fallback. <!-- R6 -->
- [x] T004 [P] `src/kit/skills/fab-operator.md`: launcher sentence (~line 25) and §9 Key Properties launcher rows gain the delegation note as pointers to `_cli-fab.md` § fab operator. <!-- R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: With a capable rk stubbed, bare `fab operator` execs `rk operator` — argv carries `--workers <v>` exactly when the flag was supplied, env is `os.Environ()` untouched
- [x] A-002 R2: Probe is side-effect-free and answers false silently on rk absent / incapable; fallback launcher then runs unchanged
- [x] A-003 R5: `fab operator --help` names `rk operator` delegation and the built-in fallback

### Behavioral Correctness

- [x] A-004 R3: A failing exec after a passing probe surfaces the error; no fab-launcher retry occurs
- [x] A-005 R4: All pre-existing operator tests pass, modified only by the `stubNoRK(t)` seam pin (which forces the fallback arm they exercise — required because the host machine may carry a capable rk); subcommand family and fallback launch behavior untouched

### Scenario Coverage

- [x] A-006 R7: Tests cover probe-positive (both argv shapes), probe-negative, token-helper, and exec-error cases; `go test ./cmd/fab` green in `src/go/fab`

### Edge Cases & Error Handling

- [x] A-007 R2: An rk whose `operator --help` exits 0 but lacks the `--workers` token is treated as incapable (fallback), not delegated to

### Code Quality

- [x] A-008 Pattern consistency: seams and probe match the `execAgent` / `rkPanesRunner` / `gate_rk.go` precedents; comments explain constraints, not narration
- [x] A-009 No unnecessary duplication: reuses `workersOverride`, `pane.RunCmd`, `exec.LookPath` — no parallel probe machinery duplicated from `gate_rk.go` (different capability, deliberately separate: that probe is pane-gate-specific)
- [x] A-010 CLI ⇒ docs + tests: `_cli-fab.md` updated in the same change as the Go signature behavior; tests ship alongside (Constitution + code-review.md rules)
- [x] A-011 Canonical source only: all skill edits under `src/kit/skills/`, none under `.claude/skills/`

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the built-in launcher is deliberately retained as the rk-absent fallback per R4)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Probe discriminant = `--workers` token in `rk operator --help` (not bare exit 0) | gate_rk.go token-probe precedent; guards against a pre-flag `rk operator` | S:75 R:85 A:85 D:75 |
| 2 | Confident | True `syscall.Exec` behind an injectable seam, env untouched | execAgent precedent; no post-delegation fab work exists | S:75 R:85 A:85 D:70 |
| 3 | Certain | `--workers` rides argv on the delegated path, never the env prefix | rk owns FAB_AGENT_WORKERS composition for the agent it launches | S:85 R:90 A:90 D:90 |
| 4 | Confident | Probe lives unexported in `cmd/fab/operator.go`, not `internal/pane` | gate_rk.go's probe is pane-gate-specific; this one is launcher-specific — no shared consumer yet | S:70 R:90 A:80 D:75 |

4 assumptions (1 certain, 3 confident, 0 tentative).
