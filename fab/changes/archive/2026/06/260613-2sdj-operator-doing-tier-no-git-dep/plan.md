# Plan: Operator runs on the doing-tier model and drops its hard git-repo dependency

**Change**: 260613-2sdj-operator-doing-tier-no-git-dep
**Intake**: `intake.md`

## Requirements

### Operator: Launch from a neutral directory

#### R1: Operator launch no longer requires a git repo
`runOperator` (`src/go/fab/cmd/fab/operator.go`) SHALL resolve the new tmux window's working directory by trying `gitRepoRoot()` first and falling back to `os.Getwd()` when that fails. It MUST NOT hard-fail with `cannot determine repo root` when launched outside a git repo. It SHALL error only when BOTH `gitRepoRoot()` and `os.Getwd()` fail (a genuinely broken environment). `tmux new-window` MUST always receive a `-c <dir>` argument.

- **GIVEN** the operator is launched from inside a git repo
- **WHEN** `runOperator` resolves the window directory
- **THEN** it uses the git repo root (today's behavior, unchanged)

- **GIVEN** the operator is launched from a neutral directory outside any git repo
- **WHEN** `gitRepoRoot()` fails
- **THEN** it falls back to `os.Getwd()` and the window is created with `-c <cwd>` (no hard error)

- **GIVEN** both `gitRepoRoot()` and `os.Getwd()` fail
- **WHEN** `runOperator` resolves the window directory
- **THEN** it returns an error (broken environment)

#### R1b: Operator launch no longer requires a resolvable `fab/` project
`runOperator` SHALL treat `resolve.FabRoot()` failure as non-fatal. When a `fab/` project is resolvable, it reads the spawn command from that project's `fab/project/config.yaml` (today's behavior, via `spawn.Command`). When `FabRoot()` fails (the operator is launched from a directory with no `fab/` project anywhere up the tree — its natural cross-repo home, e.g. `~/code`), it SHALL fall back to `spawn.DefaultSpawnCommand` rather than erroring. This is the same "stop forcing the operator into one project" principle as R1: the operator is a per-tmux-server cross-repo coordinator with no single owning project. The doing-tier resolution (R2) already degrades to the built-in default when no project is resolvable, so the no-`fab/` launch composes a fully-defaulted command (default spawn command + doing default `{model, effort}`).

- **GIVEN** the operator is launched from a directory under a resolvable `fab/` project
- **WHEN** `runOperator` reads the spawn command
- **THEN** it uses that project's `agent.spawn_command` via `spawn.Command(configPath)` (today's behavior, unchanged)

- **GIVEN** the operator is launched from a directory with no `fab/` project up the tree
- **WHEN** `resolve.FabRoot()` fails
- **THEN** `runOperator` falls back to `spawn.DefaultSpawnCommand` and proceeds (no hard error)

### Operator: Doing-tier model selection

#### R2: Operator launches its agent on the doing tier
`runOperator` SHALL resolve the `doing`-tier `{model, effort}` profile by shelling out to `fab resolve-agent apply` (where `apply` is the canonical member of the fab-owned, fixed `doing` tier) and parsing its byte-stable stdout (`model=<id>` and optional `effort=<level>`). On ANY failure — the command erroring (e.g. an installed binary that lacks the subcommand, or no resolvable fab project) OR empty/unparseable output — it MUST fall back to the in-process built-in doing default `agent.DefaultTier(agent.TierDoing)` = `{claude-opus-4-8, high}`. The call site MUST carry a prominent comment documenting WHY `apply` is probed, to flag the coupling to the fab-owned stage→tier mapping.

- **GIVEN** `fab resolve-agent apply` succeeds with well-formed `model=X\neffort=Y\n` stdout
- **WHEN** the profile is resolved
- **THEN** the profile is `{X, Y}`

- **GIVEN** `fab resolve-agent apply` emits `model=X\n` with no effort line
- **WHEN** the profile is resolved
- **THEN** the profile is `{X, ""}`

- **GIVEN** `fab resolve-agent apply` errors OR produces empty/garbage output
- **WHEN** the profile is resolved
- **THEN** the profile falls back to the built-in doing default `{claude-opus-4-8, high}`

#### R3: Parse-or-default logic is a pure, testable function
The parse + fallback logic MUST be extracted into a pure function `resolveDoingProfile(stdout string) agent.Profile` in `src/go/fab/cmd/fab/operator.go`. The live shell-out (exec) stays in `runOperator`. `resolveDoingProfile("")` MUST return the built-in doing default, so the caller passes `""` on command error.

- **GIVEN** `resolveDoingProfile` receives an empty string (command failed)
- **WHEN** it parses
- **THEN** it returns the built-in doing default `{claude-opus-4-8, high}`

### Operator: Spawn-command profile injection

#### R4: A reusable spawn.WithProfile helper injects --model/--effort
`src/go/fab/internal/spawn/spawn.go` SHALL gain `func WithProfile(spawnCmd, model, effort string) string` that appends `--model <model>` and `--effort <effort>` to the END of `spawnCmd` (last-wins), in that order (model then effort), OMITTING each flag entirely when its value is empty. `runOperator` SHALL wrap the configured spawn command via `spawnCmd = spawn.WithProfile(spawnCmd, profile.Model, profile.Effort)` before composing the final `shellCmd`.

- **GIVEN** both model and effort are non-empty
- **WHEN** `WithProfile` is called
- **THEN** both `--model <m>` and `--effort <e>` are appended at the end, model before effort

- **GIVEN** the model is empty and effort is non-empty
- **WHEN** `WithProfile` is called
- **THEN** only `--effort <e>` is appended

- **GIVEN** the effort is empty and model is non-empty
- **WHEN** `WithProfile` is called
- **THEN** only `--model <m>` is appended

- **GIVEN** both model and effort are empty
- **WHEN** `WithProfile` is called
- **THEN** `spawnCmd` is returned unchanged

### Documentation

#### R5: Operator docs reflect the doing-tier model and the neutral-directory launch
`src/kit/skills/fab-operator.md`, `docs/specs/skills/SPEC-fab-operator.md`, and `src/kit/skills/_cli-fab.md` (`## fab operator` section) MUST document that the operator (a) launches its coordinating agent on the doing-tier `{model, effort}` (injected as `--model`/`--effort`) and (b) no longer requires either a git repo OR a resolvable `fab/` project. The docs MUST accurately state the degraded behavior: outside a git repo the window cwd falls back to `os.Getwd()`; with no `fab/` project the spawn command falls back to `spawn.DefaultSpawnCommand` (no project `agent.spawn_command`/`agent.tiers` is read) and the doing tier resolves to its built-in default. The docs MUST NOT overclaim — earlier draft prose said the natural launch point is "a neutral parent directory with no `fab/` project" while the code still required `fab/`; the corrected prose now genuinely matches the (now `fab/`-optional) implementation.

- **GIVEN** the operator behavior changed (doing-tier model + neutral-directory launch: git-optional AND `fab/`-optional)
- **WHEN** the docs are read
- **THEN** all three files describe the launch-cwd contract, the `fab/`-optional spawn-command fallback, and the doing-tier injection consistently with the implementation — with no overclaim about what a `fab/`-less launch provides

### Non-Goals

- No state-file path or config-schema change (state stays socket-keyed; doing-tier consumption reads existing config/defaults). No migration.
- No change to the installed `fab` binary — the fallback exists precisely because the binary on PATH may predate `resolve-agent`.
- No change to the other 3 `spawn.Command` call sites (batch_switch, batch_new, spawn_command) — `WithProfile` is reusable but only wired into the operator here.

### Design Decisions

1. **Probe `apply`, not `doing` directly**: `fab resolve-agent` takes a STAGE, not a tier. `apply` is the canonical doing-tier stage in the fixed stage→tier mapping — *Why*: reuses the existing canonical resolution surface and picks up project `agent.tiers.doing` overrides for free — *Rejected*: a new `fab resolve-tier <tier>` command (unneeded new surface; the stage probe already resolves the tier).
2. **Caller passes `""` to `resolveDoingProfile` on command error**: the pure function treats empty/garbage as "use default" — *Why*: keeps the exec out of the unit-testable seam while ensuring a single default path — *Rejected*: returning an error from the pure fn (forces caller branching for no benefit).
3. **Shared `spawn.WithProfile` over inline concat**: `spawn.Command` has 4 call sites — *Why*: reusable + unit-testable, matches existing factoring — *Rejected*: inline string-building in `runOperator` (not testable, not reusable).

## Tasks

### Phase 1: Core helpers (parallelizable — independent files)

- [x] T001 [P] Add `func WithProfile(spawnCmd, model, effort string) string` to `src/go/fab/internal/spawn/spawn.go` — appends `--model`/`--effort` at the end (model then effort), omitting each flag when its value is empty. <!-- R4 -->
- [x] T002 [P] Add the pure `func resolveDoingProfile(stdout string) agent.Profile` to `src/go/fab/cmd/fab/operator.go` — parses `model=`/`effort=` lines; empty/unparseable ⇒ `agent.DefaultTier(agent.TierDoing)`. Import `internal/agent`. <!-- R2, R3 -->

### Phase 2: Wire into runOperator

- [x] T003 Replace the hard `gitRepoRoot()` error in `runOperator` (`src/go/fab/cmd/fab/operator.go`) with a `windowDir` resolution: try `gitRepoRoot()`, fall back to `os.Getwd()`, error only if both fail; use `windowDir` as the `tmux new-window -c` argument. <!-- R1 -->
- [x] T003b Make `resolve.FabRoot()` failure non-fatal in `runOperator`: when it succeeds, read `spawn.Command(filepath.Join(fabRoot, "project", "config.yaml"))` (today's behavior); when it fails, fall back to `spawn.DefaultSpawnCommand`. A comment SHALL explain that the operator is a cross-repo coordinator that may launch outside any `fab/` project. <!-- R1b -->
- [x] T004 In `runOperator`, after resolving `spawnCmd`, shell out to `fab resolve-agent apply` (with a prominent WHY-apply comment), pass its stdout (or `""` on command error) to `resolveDoingProfile`, and wrap the spawn command via `spawnCmd = spawn.WithProfile(spawnCmd, profile.Model, profile.Effort)` before composing `shellCmd`. <!-- R2, R4 -->

### Phase 3: Tests (test-alongside)

- [x] T005 [P] Add `WithProfile` table tests to `src/go/fab/internal/spawn/spawn_test.go`: both flags present (order model→effort at end), empty model only, empty effort only, both empty (unchanged). <!-- R4 -->
- [x] T006 [P] Add `resolveDoingProfile` table tests to `src/go/fab/cmd/fab/operator_test.go` (match the existing table-test style): well-formed `model=X\neffort=Y\n` → {X,Y}; `model=X\n` only → {X,""}; `""` → doing default; garbage → doing default. <!-- R2, R3 -->

### Phase 4: Docs

- [x] T007 [P] Update `src/kit/skills/fab-operator.md` — operator launches on the doing-tier model and no longer requires a git repo OR a resolvable `fab/` project (window cwd = repo root inside a repo, else `os.Getwd()`; spawn command from project config when `fab/` resolvable, else `spawn.DefaultSpawnCommand`). Remove the overclaim corrected per R5. <!-- R5 --> <!-- rework: scope expanded to fab/-optional; correct should-fix overclaim -->
- [x] T008 [P] Update `docs/specs/skills/SPEC-fab-operator.md` — mirror the behavior update (git-optional AND `fab/`-optional, with accurate degraded-behavior description). <!-- R5 --> <!-- rework: scope expanded to fab/-optional -->
- [x] T009 [P] Update `src/kit/skills/_cli-fab.md` `## fab operator` section — launch-cwd contract (repo root when inside a repo, else cwd), `fab/`-optional spawn-command fallback, and the doing-tier `--model`/`--effort` injection. <!-- R5 --> <!-- rework: scope expanded to fab/-optional -->

## Execution Order

- T001, T002 (Phase 1) precede T004 (which calls both).
- T003 and T004 both edit `runOperator` — sequence them (T003 then T004) to avoid edit conflicts.
- T005 depends on T001; T006 depends on T002.
- Phase 4 docs are independent of code and of each other.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `runOperator` resolves the window dir via `gitRepoRoot()` with an `os.Getwd()` fallback and errors only when both fail; the hard `cannot determine repo root` error is gone.
- [x] A-001b R1b: `runOperator` treats `resolve.FabRoot()` failure as non-fatal — reads the project spawn command when `fab/` is resolvable, else falls back to `spawn.DefaultSpawnCommand`; the operator no longer errors when launched from a directory with no `fab/` project.
- [x] A-002 R2: `runOperator` shells `fab resolve-agent apply`, parses the profile, and falls back to `agent.DefaultTier(agent.TierDoing)` on any failure; a prominent WHY-apply comment is present at the call site.
- [x] A-003 R3: `resolveDoingProfile(stdout string) agent.Profile` exists as a pure function; `resolveDoingProfile("")` returns the built-in doing default.
- [x] A-004 R4: `spawn.WithProfile(spawnCmd, model, effort)` exists and is wired into `runOperator` so the doing-tier `--model`/`--effort` are appended last to the spawn command.
- [x] A-005 R5: `fab-operator.md`, `SPEC-fab-operator.md`, and `_cli-fab.md` `## fab operator` document the doing-tier model and the no-git-repo launch-cwd contract.

### Behavioral Correctness

- [x] A-006 R1: Inside a git repo, the window cwd is unchanged (still the repo root); outside a repo it degrades to `os.Getwd()`.
- [x] A-006b R1b: Launched from a directory under a `fab/` project, the spawn command is unchanged (project `agent.spawn_command`); launched with no `fab/` project, it degrades to `spawn.DefaultSpawnCommand`.
- [x] A-007 R4: Both flags are appended at the END (last-wins) in order model→effort; each flag is omitted entirely when its value is empty.

### Scenario Coverage

- [x] A-008 R4: `spawn_test.go` covers all four `WithProfile` cases (both present, empty model, empty effort, both empty).
- [x] A-009 R2: `operator_test.go` covers all four `resolveDoingProfile` cases (well-formed, model-only, empty, garbage).

### Edge Cases & Error Handling

- [x] A-010 R2: A `fab resolve-agent apply` that errors (installed binary lacks the subcommand) routes the operator to the built-in doing default without crashing.

### Code Quality

- [x] A-011 Pattern consistency: New code follows the surrounding naming, error-handling (`pane.StderrError` where relevant), and comment-density conventions of `operator.go` / `spawn.go`.
- [x] A-012 No unnecessary duplication: Reuses `agent.TierDoing`/`agent.DefaultTier` and `spawn.Command` rather than reimplementing; no magic strings (tier/stage names come from the `agent` package or are the documented canonical `apply` probe).

### Documentation Accuracy (checklist.extra_categories)

- [x] A-013: The three doc updates accurately match the shipped behavior (no drift between docs and `operator.go`/`spawn.go`).

### Cross References (checklist.extra_categories)

- [x] A-014: Doc cross-references stay consistent — `_cli-fab.md` `## fab operator`, `fab-operator.md`, and `SPEC-fab-operator.md` agree on the launch-cwd contract and doing-tier injection.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Window cwd: try `gitRepoRoot()`, fall back to `os.Getwd()`, error only if both fail. | Verbatim intake assumption #1 (settled in design conversation); single reversible call site; constitution/codebase support graceful degradation. | S:90 R:80 A:85 D:90 |
| 2 | Confident | Doing-model source: shell `fab resolve-agent apply`; on ANY failure fall back to `agent.DefaultTier(agent.TierDoing)`. | Verbatim intake assumption #2; reuses just-merged l3ja surface; byte-stable contract verified. | S:90 R:75 A:85 D:85 |
| 3 | Certain | Prominent call-site comment documents WHY `apply` is probed (canonical doing-tier stage). | Verbatim intake assumption #3 + constitution Code Quality (document non-obvious coupling). | S:95 R:80 A:90 D:95 |
| 4 | Confident | Inject `--model`/`--effort` at the END (model then effort), last-wins, each omitted when empty. | Verbatim intake assumption #4; empirically verified; mirrors `_preamble` empty⇒omit convention. | S:90 R:80 A:85 D:90 |
| 5 | Confident | Implement injection as a reusable `spawn.WithProfile` helper in `internal/spawn`. | Verbatim intake assumption #5; `spawn.Command` has 4 call sites; matches existing factoring. | S:90 R:85 A:80 D:85 |
| 6 | Confident | Extract parse-or-default into pure `resolveDoingProfile`; live shell-out stays in `runOperator`. | Verbatim intake assumption #6 + constitution Test Integrity; `operator_test.go` already tests pure helpers. | S:90 R:85 A:85 D:90 |
| 7 | Certain | Docs required: `fab-operator.md`, `SPEC-fab-operator.md`, `_cli-fab.md` `## fab operator`. | Verbatim intake assumption #7 + constitution (CLI→`_cli-fab.md`, skill→`SPEC-*.md`). | S:95 R:85 A:95 D:95 |
| 8 | Confident | No migration required (no state-path/config-schema change). | Verbatim intake assumption #8; state socket-keyed; doing-tier reads existing config/defaults. | S:90 R:80 A:85 D:85 |
| 9 | Confident | `resolveDoingProfile` lives in `cmd/fab` (package `main`, alongside `operator.go`), not `internal/agent` — keeps `agent` free of stdout-parsing and matches where the existing pure helpers (`findWindowExact`, `slugify`) live. | Not explicitly stated in intake but implied by "extract into `operator.go`" and the existing test file location; one obvious placement. | S:85 R:85 A:85 D:85 |

9 assumptions (2 certain, 7 confident, 0 tentative).
