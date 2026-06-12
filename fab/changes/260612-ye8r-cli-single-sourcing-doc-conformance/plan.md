# Plan: CLI Single-Sourcing & Doc Conformance

**Change**: 260612-ye8r-cli-single-sourcing-doc-conformance
**Intake**: `intake.md`

## Requirements

### fab-kit Go: Lifecycle-command single-sourcing (F23)

#### R1: Shared LifecycleCommands table
The fab-kit module SHALL define a single canonical table `internal.LifecycleCommands` (`[]LifecycleCommand{Name, Short}`) listing the 6 workspace commands (`init`, `upgrade-repo`, `sync`, `update`, `doctor`, `migrations-status`). The router's `fabKitArgs` map (`cmd/fab/main.go`) and `cmd/fab-kit`'s `fabKitCommands` map MUST be derived from this table — no hand-maintained copies of the 6 names remain in either `cmd/` entry.

- **GIVEN** a new lifecycle command is added to the table
- **WHEN** both binaries are rebuilt
- **THEN** the router allowlist, the router help section, and the fab-kit test cross-check all pick it up from the single table with no other Go edit

#### R2: Router workspace help derived from the table in-process
`printHelp` in `cmd/fab/main.go` MUST derive its "Workspace commands" section from `LifecycleCommands` (names + Shorts) in-process — it MUST NOT exec `fab-kit --help`. The help section SHALL render even when the fab-kit binary is absent, and the rendered Short for each command SHALL equal the cobra `Short` registered in `cmd/fab-kit` (fixing the already-diverged `migrations-status` line).

- **GIVEN** the fab-kit binary is not installed
- **WHEN** `fab --help` runs
- **THEN** the workspace-commands section still renders, with Shorts identical to the cobra registrations

#### R3: Contract + collision tests replace the tautologies
The two tautological tests MUST be replaced by tests that can catch real skew: (a) a `cmd/fab-kit` test asserting the set of registered cobra commands equals the table's names AND each registered command's `Short` equals the table entry; (b) a doc contract test (modeled on `changetypes_doc_test.go`) parsing the `_cli-fab.md` router line's parenthesized command list and asserting set equality with the table; (c) a collision test in the fab module asserting no top-level fab-go command name (sourced from the in-process `fab help-dump` tree of the assembled root command) appears in the documented router allowlist parsed from the same `_cli-fab.md` line.

- **GIVEN** a future fab-go command is added whose name matches a router-allowlisted workspace command
- **WHEN** `go test ./...` runs in the fab module
- **THEN** the collision test fails naming the shadowed command
- **AND** when the `_cli-fab.md` router line drifts from the Go table, the fab-kit contract test fails

### Kit docs: undocumented CLI surface (F24)

#### R4: `_cli-fab.md` documents the full surface
`src/kit/skills/_cli-fab.md` MUST document: (1) `fab pane window-name ensure-prefix|replace-prefix` including the `--json` flag and the 2/3 exit-code scheme; (2) `fab shell-init <bash|zsh|fish>`; (3) `fab change list --show-stats` (both the `_cli-fab.md` change table and `_preamble.md`'s `fab change` row); (4) the router line at the top MUST include `migrations-status`. The same pass SHALL fix `docs/memory/runtime/pane-commands.md`'s "1 (no tmux) / 2 / 3" changelog inconsistency (code routes tmux-not-running → 3). This consolidates findings f008 + f136 from `docs/specs/findings/skills-review-2026-06-11.md`.

- **GIVEN** an agent loads `_cli-fab.md`
- **WHEN** it needs window-name, shell-init, or `--show-stats`
- **THEN** the reference documents the signature, flags, and exit codes matching the code

### fab Go: resolve output flags (F25)

#### R5: Mutually exclusive output flags; `--id` wired
`resolveCmd` MUST call `cmd.MarkFlagsMutuallyExclusive("id", "folder", "dir", "status", "pane")` so conflicting flags fail loudly, and `--id` MUST be read and wired into the output-mode selection (a real explicit-default flag, not dead surface). The `-o/--output` enum consolidation is deferred (Non-Goals).

- **GIVEN** `fab resolve --status --folder <change>`
- **WHEN** the command executes
- **THEN** it exits non-zero with cobra's mutual-exclusion error instead of silently printing the folder
- **AND GIVEN** `fab resolve --id <change>`, the 4-char ID prints (explicit default)

### fab Go: config.yaml parser consolidation (F26)

#### R6: Five fab-module parsers behind `internal/config`
`internal/config.Config` MUST be widened to model `branch_prefix`, `fab_version`, `agent.spawn_command`, and `project.linear_workspace`, and the four satellite parsers (`spawn.Command`, `getBranchPrefix` in `batch_switch.go`, `readLinearWorkspace` in `prmeta.go`, the anonymous struct in `preflight.checkSyncStaleness`) MUST be converted to accessors over a single `config.Load`/`config.LoadPath` result, following the nil-safe `GetStageHook` pattern. Per-caller fallback semantics MUST survive: spawn's `DefaultSpawnCommand`, empty branch prefix, prmeta's empty workspace, preflight's silent skip. `prmeta.buildDerivation`'s double parse of the same file MUST collapse to one `config.Load`. The fab-kit module's `readFabVersion` stays (cross-module internal visibility). Known recorded caveat: a single Unmarshal couples failure modes — a yaml type error on any modeled key sends all accessors to their documented fallbacks.

- **GIVEN** a config.yaml with `branch_prefix`, `agent.spawn_command`, `project.linear_workspace`, `fab_version`
- **WHEN** any consumer reads its key
- **THEN** the value comes from the shared `internal/config` struct via a nil-safe accessor
- **AND GIVEN** the file is missing or malformed, each consumer observes its previous fallback (default spawn command / empty prefix / empty workspace / silent staleness skip)

### fab Go: resolve surface dedup (F27)

#### R7: `change resolve` is a thin wrapper over the shared resolve implementation
`fab change resolve [override]` MUST execute the same shared implementation as `fab resolve --folder` (a thin cobra wrapper that sets the folder output mode), so help/flag surface can never drift again. Deprecation is rejected — skills depend on `change resolve`. The now-unused one-line passthrough `internal/change.Resolve` SHALL be removed.

- **GIVEN** `fab change resolve <override>` and `fab resolve --folder <override>`
- **WHEN** both run against the same change
- **THEN** they produce identical stdout and identical error strings via the same code path

#### R8: `--server`/`-L` plumbed through `fab resolve` for `--pane`
`resolveCmd` MUST gain a `--server`/`-L` string flag (same help text as the pane family's persistent flag). In pane mode with `--server` set, discovery MUST target that socket server-wide and the `$TMUX` guard is skipped; with `--server` empty, behavior is unchanged (current-session discovery, `$TMUX` required).

- **GIVEN** a change's pane lives on tmux socket `runKit` and the caller is outside that server
- **WHEN** `fab resolve <change> --pane --server runKit` runs
- **THEN** the pane ID on that server is returned without requiring `$TMUX`

### fab Go: `fab log command` best-effort contract (F28)

#### R9: `fab log command` always exits 0; boilerplate retired
`fab log command` (the telemetry event ONLY — `log review`/`log confidence`/`log transition` unchanged) MUST always exit 0, printing a one-line `Warning: fab log command: …` to stderr on any internal failure (FabRoot failure, explicit-change resolve failure, unwritable `.history.jsonl`). The `2>/dev/null || true` boilerplate MUST then be deleted from: `_preamble.md` (Common-fab-Commands row, key-behaviors bullet, §2 step-4 template — coordinated wording with the failure rule), the 5 skill files with explicit guarded calls (`fab-help.md`, `fab-switch.md`, `fab-setup.md`, `fab-operator.md`, `fab-discuss.md`), `_cli-fab.md`'s fab-log section (exit-1 asymmetry note), and each touched skill's SPEC mirror.

- **GIVEN** `.history.jsonl` is unwritable or no fab root exists
- **WHEN** `fab log command "skill" "id"` runs unguarded
- **THEN** it prints a stderr warning and exits 0 — a forgotten shell guard can never escalate telemetry failure into a pipeline STOP

### fab Go: batch family (F29)

#### R10: `batch switch` resolves in-process
`batch_switch.go` MUST replace the `exec.Command("fab", "change", "resolve", …)` PATH self-exec with a direct `resolve.ToFolder(fabRoot, change)` call (mirroring `batch_archive.go`), keeping warn-and-skip but surfacing the resolver's specific error (e.g. "Multiple changes match…") in the warning.

- **GIVEN** an ambiguous change argument
- **WHEN** `fab batch switch <arg>` runs
- **THEN** the warning names the specific resolution error, no `fab` subprocess is spawned, and remaining changes still process

#### R11: `batch archive` no-arg defaults to `--list`
`fab batch archive` with no arguments MUST default to `--list` (aligned with `batch new`/`batch switch`); the bulk action requires explicit `--all`.

- **GIVEN** `fab batch archive` with no args and archivable changes present
- **WHEN** it runs
- **THEN** it lists archivable changes and archives nothing; `fab batch archive --all` performs the bulk action

### fab Go: exit-code semantics & error formatting (F30)

#### R12: capture/send adopt the 2/3 pane exit-code scheme
`fab pane capture` and `fab pane send` MUST map a `ValidatePane` failure to exit 2 (pane missing) or exit 3 (any other tmux failure), matching `window-name`'s scheme. `internal/pane` SHALL expose the classification (a `PaneNotFoundError` type detectable via `errors.As`) with the error message `pane <id> not found` unchanged. The operator contract is preserved: `window-name` exit-2-as-successful-removal semantics are untouched (R14).

- **GIVEN** pane `%99` does not exist
- **WHEN** `fab pane capture %99` or `fab pane send %99 hi` runs
- **THEN** stderr carries `Error: pane %99 not found` and the exit code is 2
- **AND GIVEN** the tmux server is dead, the exit code is 3

#### R13: Plain exit-1 paths funneled through RunE
Handlers whose failure paths exit 1 MUST return errors from `RunE` instead of in-handler `os.Exit(1)`, so all plain failures flow through the single `main.go` `ERROR: %s` formatter. Sites: `resolve.go` ($TMUX guard, no-matching-pane), `panemap.go` ($TMUX guard), `pane_capture.go` (`--lines` validation), `pane_send.go` (agent-not-idle), `pane_process.go` (ValidatePane failure), `batch_switch.go` ($TMUX guard, no-changes), `batch_archive.go` (no-valid-changes, failed>0). In-handler `os.Exit` remains ONLY where non-1 codes are genuinely needed (`window-name`, capture/send pane validation, `doctor`'s failure-count exit).

- **GIVEN** any of the converted failure paths fires
- **WHEN** the command exits
- **THEN** stderr shows the single `ERROR: <message>` format and the exit code is 1

#### R14: Operator window-name contract preserved
`fab pane window-name`'s exit-code scheme and output semantics MUST be byte-identical after this change — `/fab-operator` treats exit 2 as successful removal.

- **GIVEN** the operator removal path hits a vanished pane
- **WHEN** `fab pane window-name replace-prefix <pane> » ›` runs
- **THEN** it still exits 2 with tmux's stderr propagated

### Docs: memory + spec mirrors

#### R15: Affected memory files updated with the new behavior
`docs/memory/distribution/kit-architecture.md` (router allowlist → shared `LifecycleCommands` table + table-derived help replacing the static-help asymmetry note; resolve PreRunE priority chain → mutual exclusivity + `--server`; batch switch resolution in-process — extend the batch-archive note to the family; `change resolve` thin wrapper), `docs/memory/runtime/pane-commands.md` (capture/send 2/3 scheme, updated documented error strings, corrected changelog "1 (no tmux)" note), `docs/memory/pipeline/schemas.md` (`fab log command` fully owns the best-effort contract), and `docs/memory/pipeline/preflight.md` (staleness check reads `fab_version` via the shared accessor — wording only) MUST reflect the implemented behavior.

- **GIVEN** an agent consults these memory files after this change
- **WHEN** it reads the documented schemes
- **THEN** they match the shipped code (no stale exit codes, parsers, or allowlist descriptions)

#### R16: SPEC mirrors updated for every touched skill file
Every `src/kit/skills/*.md` file changed by this plan MUST have its `docs/specs/skills/SPEC-*.md` mirror reconciled in the same change (constitution.md:32). Touched: `SPEC-_preamble.md`, `SPEC-fab-help.md`, `SPEC-fab-switch.md`, `SPEC-fab-setup.md`, `SPEC-fab-operator.md`, `SPEC-fab-discuss.md`, plus `SPEC-hooks.md`'s guarded example line. (`_cli-fab.md` has no SPEC mirror — none exists by precedent.)

- **GIVEN** a skill file edit retiring the log-command guard
- **WHEN** its SPEC mirror is read
- **THEN** the mirror carries no stale `2>/dev/null || true` guidance

### Non-Goals

- `-o/--output id|folder|dir|status|pane` enum consolidation for `fab resolve` — deferred (clarified; touches ≥4 doc files, exceeds batch budget)
- Generic CI diff of `fab help-dump` output vs `_cli-fab.md` — scoped out (clarified; F23 contract+collision tests cover the drift class)
- x8c9 change-types drift test — already shipped on main; no work here
- fab-kit module's `readFabVersion` consolidation — Go internal-package visibility forbids cross-module reuse
- `batch_switch.go` wt-create/tmux-new-window stderr surfacing — F31 (B5) territory, not in this intake's scope
- k4ge-owned items (`change resolve --folder` doc fix, `fab hook sync` exit-0 overclaim) — already merged as #395

### Design Decisions

1. **Collision test lives in the fab module and meets the table through `_cli-fab.md`**: the fab module cannot import the fab-kit module's `internal/` (Go visibility), so the collision test parses the same `_cli-fab.md` router line the fab-kit contract test pins to the Go table — giving transitive code↔code coverage with no cross-module import and no subprocess. — *Why*: in-process, hermetic, uses the intake-mandated help-dump tree (via `dumpDoc` on the assembled root). — *Rejected*: exec'ing `fab help-dump` from the fab-kit module's tests (requires a built binary on PATH — version-skew-prone); a checked-in snapshot (one more hand-maintained copy).
2. **`PaneNotFoundError` typed error in `internal/pane`**: classification for the 2/3 split travels on the error value (`errors.As`), keeping the `pane <id> not found` message byte-identical. — *Why*: call sites must not re-parse error strings; window-name's stderr-bytes mapping (`tmuxExitCode`) is unavailable at `ValidatePane` call sites. — *Rejected*: sentinel `errors.New` (message composition gets awkward); exporting stderr bytes from ValidatePane (breaks every caller's signature).
3. **`config.LoadPath` alongside `Load`**: `spawn.Command(configPath)` keeps its path-based signature (used by `fab spawn-command --repo` for arbitrary repo roots) and delegates to `config.LoadPath(configPath)`; `Load(fabRoot)` becomes a thin join over it. — *Why*: zero churn at spawn's call sites while the parse itself is single-sourced. — *Rejected*: changing spawn.Command to take fabRoot (touches 4 call sites and the --repo path construction for no gain).
4. **`resolve --pane --server` implies server-wide discovery**: with `--server` set, discovery runs in all-sessions mode on that socket and the `$TMUX` guard is skipped; default behavior (no flag) is unchanged. — *Why*: "current session" is undefined on a foreign socket; the motivating callers (daemons) are outside that server — mirrors `pane map`'s `--session`/`--all-sessions` guard-skip semantics. — *Rejected*: keeping the `$TMUX` guard with `--server` (defeats the cross-socket purpose); plumbing `--session` targeting too (no caller needs it; latent surface).

## Tasks

### Phase 1: fab-kit module single-sourcing (F23)

- [x] T001 Create `src/go/fab-kit/internal/lifecycle.go`: `LifecycleCommand` struct, `LifecycleCommands` table (6 entries, Shorts = cobra Shorts incl. migrations-status's full text), `LifecycleCommandSet() map[string]bool` <!-- R1 -->
- [x] T002 Derive `fabKitArgs` from the table and render `printHelp`'s workspace section from it in-process in `src/go/fab-kit/cmd/fab/main.go` <!-- R1, R2 -->
- [x] T003 Derive `fabKitCommands` from the table and extract a `rootCmd()` constructor in `src/go/fab-kit/cmd/fab-kit/main.go` <!-- R1, R3 -->
- [x] T004 Replace tautological tests: `cmd/fab/main_test.go` asserts `fabKitArgs` == table set; `cmd/fab-kit/main_test.go` asserts registered cobra commands == table (names both directions + Short equality) <!-- R3 -->
- [x] T005 [P] Add `src/go/fab-kit/cmd/fab/clifab_doc_test.go`: walk-up locate `src/kit/skills/_cli-fab.md`, parse the router line's backticked command list, assert set equality with the table <!-- R3 -->

### Phase 2: fab module Go changes

- [x] T006 `src/go/fab/cmd/fab/resolve.go`: `MarkFlagsMutuallyExclusive("id","folder","dir","status","pane")`; wire `--id` into the selection chain <!-- R5 -->
- [x] T007 `src/go/fab/cmd/fab/resolve.go`: extract shared `runResolve` implementation; add `--server`/`-L` flag; pane mode uses server-wide discovery + skips `$TMUX` guard when `--server` set; convert the two `os.Exit(1)` paths to returned errors <!-- R7, R8, R13 -->
- [x] T008 `src/go/fab/cmd/fab/change.go`: `changeResolveCmd` becomes a thin wrapper invoking the shared resolve implementation in folder mode; delete `internal/change.Resolve` passthrough in `src/go/fab/internal/change/change.go` <!-- R7 -->
- [x] T009 `src/go/fab/internal/config/config.go`: widen `Config` (`branch_prefix`, `fab_version`, `agent.spawn_command`, `project.linear_workspace`), add `LoadPath`, add nil-safe accessors (`GetBranchPrefix`, `GetFabVersion`, `GetSpawnCommand`, `GetLinearWorkspace`) <!-- R6 -->
- [x] T010 Convert satellite parsers to the shared config: `src/go/fab/internal/spawn/spawn.go` (delegate to `config.LoadPath`, keep `DefaultSpawnCommand` fallback), `src/go/fab/cmd/fab/batch_switch.go` (delete `getBranchPrefix`), `src/go/fab/internal/prmeta/prmeta.go` (delete `readLinearWorkspace`, reuse the existing `config.Load` result), `src/go/fab/internal/preflight/preflight.go` (staleness via `config.Load`, silent-skip preserved) <!-- R6 -->
- [x] T011 `src/go/fab/cmd/fab/log.go`: `logCommandCmd` always exits 0 — extract `runLogCommand`, wrap with stderr `Warning:` on failure; `log review`/`confidence`/`transition` untouched <!-- R9 -->
- [x] T012 `src/go/fab/cmd/fab/batch_switch.go`: replace `exec.Command("fab","change","resolve",…)` with `resolve.ToFolder(fabRoot, change)` surfacing the specific error in the warning; convert `os.Exit(1)` sites (lines 59/68) to returned errors <!-- R10, R13 -->
- [x] T013 `src/go/fab/cmd/fab/batch_archive.go`: no-arg default flips to `--list` (explicit `--all` required); convert `os.Exit(1)` sites (no-valid-changes, failed>0) to returned errors <!-- R11, R13 -->
- [x] T014 `src/go/fab/internal/pane/pane.go`: add `PaneNotFoundError` type; `validatePaneResult` returns it on both missing-pane branches (message `pane <id> not found` unchanged) <!-- R12 -->
- [x] T015 Pane exit-code conformance: shared `paneValidationExitCode(err)` helper in `src/go/fab/cmd/fab/pane.go`; `pane_capture.go` + `pane_send.go` ValidatePane failures exit 2/3; convert plain exit-1 paths to RunE errors — `pane_capture.go` (`--lines`), `pane_send.go` (not-idle), `pane_process.go` (ValidatePane), `panemap.go` (`$TMUX` guard); `pane_window_name.go` untouched <!-- R12, R13, R14 -->

### Phase 3: Tests & verification

- [x] T016 Extract `newRootCmd()` in `src/go/fab/cmd/fab/main.go`; add `src/go/fab/cmd/fab/lifecycle_collision_test.go` asserting no top-level command of the help-dump tree appears in the `_cli-fab.md` router allowlist <!-- R3 -->
- [x] T017 [P] Resolve tests in `src/go/fab/cmd/fab/resolve_test.go`: mutual-exclusion error on two flags; `--id` explicit wiring; `change resolve` ↔ `resolve --folder` output parity; `--server` flag presence <!-- R5, R7, R8 -->
- [x] T018 [P] Update/add module tests: `internal/config` accessor + fallback tests (incl. malformed-yaml coupled-fallback caveat); move `getBranchPrefix` tests to config; `internal/spawn` tests still green; `cmd/fab/log_test.go` always-exit-0 + stderr warning; `batch_archive_test.go` no-arg→list default; `batch_switch_test.go` in-process resolution warn-and-skip; `paneValidationExitCode` mapping; `PaneNotFoundError` via `errors.As` in `internal/pane` <!-- R6, R9, R10, R11, R12 -->
- [x] T019 Run `go test ./...` in BOTH `src/go/fab` and `src/go/fab-kit`; fix all failures <!-- R1, R3, R5, R6, R7, R9, R12, R13 -->

### Phase 4: Docs (kit skills, SPEC mirrors, memory)

- [x] T020 `src/kit/skills/_cli-fab.md`: router line + `migrations-status`; pane family line + `window-name` section (verbs, `--json`, exit 2/3); new `fab shell-init` section; `change list [--archive] [--show-stats]`; resolve section (flags mutually exclusive, `--server` for `--pane`); fab-log section (always exit 0, warning on stderr); capture/send rows exit 2/3 + process/map ERROR-format notes; batch rows (archive no-arg → `--list`, switch in-process resolution, ERROR-format messages) <!-- R4, R5, R8, R9, R11, R12, R13 -->
- [x] T021 `src/kit/skills/_preamble.md`: log-command row guard removal + reworded purpose; key-behaviors bullet rewrite; §2 step-4 template guard removal; failure-rule wording coordinated; `fab change` row gains `--show-stats` <!-- R4, R9 -->
- [x] T022 [P] Remove `2>/dev/null || true` + stale best-effort prose from `src/kit/skills/fab-help.md`, `fab-switch.md`, `fab-setup.md`, `fab-operator.md`, `fab-discuss.md` <!-- R9 -->
- [x] T023 Reconcile SPEC mirrors: `docs/specs/skills/SPEC-_preamble.md` (guard/failure-rule wording), `SPEC-fab-help.md`, `SPEC-fab-switch.md`, `SPEC-fab-setup.md`, `SPEC-fab-operator.md`, `SPEC-fab-discuss.md`, `SPEC-hooks.md` (guarded example) <!-- R16 -->
- [x] T024 [P] `docs/memory/runtime/pane-commands.md`: capture/send error-behavior rows → 2/3 scheme + updated error strings; map error string ERROR-format; send not-idle string; correct the 260423-rxu3 changelog "1 (no tmux)" note; changelog entry for this change <!-- R4, R12, R13, R15 -->
- [x] T025 [P] `docs/memory/distribution/kit-architecture.md`: router allowlist → shared table + derived help; `fab resolve --pane` PreRunE chain note → mutual exclusivity + `--server`; batch switch in-process resolution (family-wide note); `change resolve` thin wrapper; changelog entry <!-- R15 -->
- [x] T026 [P] `docs/memory/pipeline/schemas.md` (log-command posture now literally true; guard boilerplate retired) and `docs/memory/pipeline/preflight.md` (staleness reads `fab_version` via shared accessor — wording only); changelog entries <!-- R15 -->

### Phase 5: Review rework (cycle 1) <!-- rework: review must-fix — memory doc contradicting new batch-archive default; plus low-effort should-fix items in theme -->

- [x] T027 `docs/memory/pipeline/change-lifecycle.md:144`: the benign-no-op claim still covers "/no-args" — drop that half; no-args now lists archivable changes (`--list` default), explicit `--all` required for bulk (empty `--all` set remains the benign no-op) <!-- R11, R15 -->
- [x] T028 [P] gofmt `src/go/fab/cmd/fab/batch_switch.go` (trailing blank line at EOF from the `getBranchPrefix` deletion); verify `gofmt -l` is clean across both Go modules <!-- R10 -->
- [x] T029 [P] `docs/specs/architecture.md:472`: workspace-command allowlist omits `migrations-status` — same 260610-9733 drift class fixed in `_cli-fab.md:17`; add it <!-- R4 -->
- [x] T030 [P] Qualify the unconditional "always exits 0" claim for `fab log command` with "(given valid usage — cobra arg-count errors exit non-zero before RunE)" wherever it is stated: `src/kit/skills/_cli-fab.md` (fab-log section), `src/kit/skills/_preamble.md` (key-behaviors bullet), `docs/memory/pipeline/schemas.md`; reconcile `SPEC-_preamble.md` if its wording mirrors the bullet <!-- R9, R15, R16 -->

### Phase 6: Review rework (cycle 2) <!-- rework: re-review must-fix — another stale memory body-text instance (context-loading.md failure rule); escalate spot-fixes to a class-sweep -->

- [x] T031 `docs/memory/_shared/context-loading.md:65` (`### Generic fab-Command Failure Rule`): body text still teaches the retired guard-marked rule ("not explicitly marked best-effort (`2>/dev/null || true`)") — reword to the new contract mirroring `_preamble.md`'s failure rule: unconditional non-zero → STOP, with the `fab log command` always-exit-0 carve-out (given valid usage). Leave the dated changelog row at :197 untouched <!-- R9, R15 -->
- [x] T032 [P] `docs/memory/_shared/configuration.md:43` (`spawn_command` body text): remove the `lib/spawn.sh` / `fab_spawn_cmd` ghost (helper deleted in 260402-41gc); describe the current read path (`spawn.Command` → `internal/config` `LoadPath`/`GetSpawnCommand`). Leave the :289 changelog row untouched <!-- R6, R15 -->
- [x] T033 Class-sweep so no third instance survives: grep ALL of `docs/memory/` and `docs/specs/` (body text; dated changelog rows exempt) for stale claims of every class this change touched — `2>/dev/null || true` guard taught as current contract, old batch-archive no-arg bulk default, unqualified `fab log command` "always exits 0"/exit-1 claims, `lib/spawn.sh`, the resolve PreRunE priority-chain as current behavior, pane capture/send "exit 1" claims, the 5-command router allowlist — and fix any body-text hits in the same style as the declared-file edits. Report the grep patterns used and hits fixed <!-- R15 -->

## Execution Order

- T001 blocks T002–T005 (table must exist)
- T007 blocks T008 (shared implementation before the wrapper)
- T009 blocks T010 (widened config before conversions)
- T014 blocks T015 (typed error before exit-code mapping)
- T016 depends on T020's router-line fix landing in the same change (the collision test parses the corrected line); run T020 before T019's final test pass or fix the line first
- Phase 4 doc tasks are independent of each other ([P]) but T020/T021 must precede T019's final verification because T005/T016 parse `_cli-fab.md`

## Acceptance

### Functional Completeness

- [x] A-001 R1: `internal.LifecycleCommands` is the only place the 6 workspace-command names+Shorts are declared in Go; `fabKitArgs` and `fabKitCommands` are derived
- [x] A-002 R2: `printHelp` renders the workspace section from the table with no `fab-kit --help` subprocess; migrations-status help line matches the cobra Short
- [x] A-003 R3: contract test (doc router line ↔ table), registration test (cobra ↔ table incl. Shorts), and collision test (fab-go top-level ↔ allowlist) all exist and pass
- [x] A-004 R4: `_cli-fab.md` documents window-name (with exit 2/3 + `--json`), shell-init, and `change list --show-stats`; router line includes migrations-status; `_preamble.md` fab-change row updated
- [x] A-005 R5: conflicting resolve output flags error via `MarkFlagsMutuallyExclusive`; `--id` is read and selects ID mode
- [x] A-006 R6: the four satellite parsers are gone; all four keys are modeled on `internal/config.Config` with nil-safe accessors; fab-kit's `readFabVersion` untouched
- [x] A-007 R7: `fab change resolve` executes the shared resolve implementation (folder mode); `internal/change.Resolve` removed
- [x] A-008 R8: `fab resolve --pane --server <s>` targets socket `<s>` server-wide without `$TMUX`; empty `--server` behavior unchanged
- [x] A-009 R9: `fab log command` exits 0 on all failure paths with a stderr warning; `log review`/`confidence`/`transition` semantics unchanged
- [x] A-010 R10: `batch switch` resolves via in-process `resolve.ToFolder`; warning carries the specific resolver error; no self-exec remains in either module
- [x] A-011 R11: `fab batch archive` (no args) lists; `--all` required for bulk archive
- [x] A-012 R12: capture/send exit 2 on missing pane, 3 on other tmux validation failure; error message `Error: pane <id> not found` preserved

### Behavioral Correctness

- [x] A-013 R13: every converted exit-1 path emits `ERROR: <msg>` via main.go's single formatter and exits 1; no plain-exit-1 `os.Exit` remains in the converted handlers
- [x] A-014 R14: `pane window-name` exit codes/output are byte-identical (operator exit-2 contract intact)
- [x] A-015 R6: malformed-config behavior is the documented fallbacks (recorded coupled-Unmarshal caveat), verified by test

### Scenario Coverage

- [x] A-016 R3: collision scenario — injecting a colliding command name into the test's allowlist set fails the test (verified during development); drift scenario — editing the router doc line breaks the contract test
- [x] A-017 R12: dead-server scenario verified by unit test on the exit-code mapping (2 vs 3 classification via `errors.As`)

### Edge Cases & Error Handling

- [x] A-018 R9: unwritable `.history.jsonl`, missing fab root, and bad explicit change arg each produce exit 0 + one-line stderr warning
- [x] A-019 R10: unresolvable and ambiguous change names in `batch switch` warn-and-skip with the specific message; loop continues

### Code Quality

- [x] A-020 Pattern consistency: new code follows existing conventions (cobra RunE error returns, nil-safe accessors, walk-up doc-test pattern, `WithServer` argv building)
- [x] A-021 No unnecessary duplication: no hand-maintained copies of the lifecycle set, config keys, or resolve logic remain; existing utilities (`resolve.ToFolder`, `pane.RunCmd`, `dumpDoc`) reused
- [x] A-022 No god functions: extracted helpers (`runResolve`, `runLogCommand`, `rootCmd`, `paneValidationExitCode`) keep handlers focused
- [x] A-023 No magic strings: lifecycle names/Shorts come from the named table; exit codes documented at their mapping sites

### Documentation Accuracy

- [x] A-024: `_cli-fab.md`, `_preamble.md`, and the four memory files describe exactly the shipped behavior (exit codes, defaults, allowlist, parser ownership); no stale guard guidance survives anywhere under `src/kit/skills/`
- [x] A-025: SPEC mirrors for every touched skill file reconciled (constitution.md:32); `_cli-fab.md` mirror-less status noted

### Cross References

- [x] A-026: pane-commands.md ↔ `_cli-fab.md` ↔ code agree on the pane-family exit-code scheme; kit-architecture.md ↔ shim code agree on the allowlist mechanism; f008/f136 consolidation referenced in the F24 doc updates

### Review Rework (cycle 1)

- [x] A-027 R11: `change-lifecycle.md` no longer claims no-arg `batch archive` is a bulk/benign no-op; it describes the `--list` default (empty `--all` set stays the benign no-op)
- [x] A-028: `gofmt -l` reports zero files in both `src/go/fab` and `src/go/fab-kit`
- [x] A-029 R4: `docs/specs/architecture.md` workspace allowlist includes `migrations-status`
- [x] A-030 R9: every "always exits 0" statement about `fab log command` is qualified for cobra usage errors, consistently across `_cli-fab.md`, `_preamble.md`, `schemas.md` (and `SPEC-_preamble.md` if applicable)

### Review Rework (cycle 2)

- [x] A-031 R9: `context-loading.md` failure-rule body text states the new unconditional contract with the log-command carve-out; no memory body text teaches the retired guard mechanism
- [x] A-032 R6: `configuration.md` `spawn_command` body text describes the `internal/config` read path; no `lib/spawn.sh` reference survives outside dated changelog rows
- [x] A-033 R15: a documented class-sweep of `docs/memory/` + `docs/specs/` body text found zero remaining stale claims in the classes this change touched (guard rule, batch-archive default, log exit semantics, spawn helper, resolve priority chain, pane exit-1, router allowlist)

## Notes

- Apply-execution notes (for review): (1) The SPEC mirrors for `fab-help`, `fab-switch`, `fab-setup`, `fab-operator`, and `fab-discuss` already showed unguarded `fab log command` calls — no content change was needed there; `SPEC-_preamble.md` (failure rule) and `SPEC-hooks.md` (guarded example, line 61) carried the stale guard and were updated. (2) `docs/memory` domain indexes regenerated via `fab memory-index` after the `description:` frontmatter updates (byte-stable, idempotent). (3) `fab doctor`'s `os.Exit(failures)` is untouched — exit code = failure count is a genuinely-needed non-1 contract (R13's carve-out). (4) `batch_switch.go`'s discarded `wt create`/`tmux new-window` stderr remains as-is — F31 (B5) territory, recorded in Non-Goals.
- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Collision test lives in the fab module, sources top-level names from the in-process help-dump tree (`dumpDoc`), and reads the allowlist by parsing the `_cli-fab.md` router line (transitive code↔code via the doc) | Cross-module `internal` import is impossible; exec'ing a built binary is version-skew-prone; intake mandates help-dump as the source | S:80 R:85 A:85 D:70 |
| 2 | Confident | `resolve --pane --server <s>` searches all sessions on `<s>` and skips the `$TMUX` guard; no-flag behavior unchanged | "Current session" is undefined on a foreign socket; mirrors pane map's guard-skip for explicit targeting; intake notes the session-scoping fork | S:75 R:80 A:80 D:65 |
| 3 | Confident | 2/3 exit-code extension applies to capture/send only; `pane process` ValidatePane failure funnels through RunE (exit 1, `ERROR:` format) | Intake scopes the scheme extension to capture/send verbatim and lists pane_process under the RunE-funneling sites | S:85 R:75 A:80 D:70 |
| 4 | Certain | `PaneNotFoundError` typed error carries the 2-vs-3 classification; message string `pane <id> not found` byte-identical | Smallest API that avoids string matching; preserves every documented error string | S:85 R:85 A:90 D:85 |
| 5 | Certain | `internal/change.Resolve` is deleted with the thin-wrapper conversion (sole caller was the cobra command) | Grep shows one caller; keeping a dead passthrough contradicts the change's single-sourcing goal | S:90 R:90 A:95 D:90 |
| 6 | Confident | Converted error messages keep their existing text but gain the `ERROR:` prefix via main.go (e.g. `ERROR: not inside a tmux session`, `ERROR: No valid changes to archive.`); docs updated to match | F30's stated goal is the single formatter; verifier confirmed no skill pattern-matches the old `Error:` strings on these paths | S:80 R:80 A:85 D:75 |
| 7 | Certain | `config.LoadPath(configPath)` added alongside `Load(fabRoot)`; `spawn.Command` keeps its path signature and delegates | `fab spawn-command --repo` needs path-based reads from arbitrary repo roots (intake-verified) | S:85 R:85 A:90 D:85 |
| 8 | Confident | `fabKitCommands` kept (derived from the table, per intake wording) even though only tests consume it | Intake explicitly says "cmd/fab-kit derives fabKitCommands from the table"; deleting it would deviate from the stated design | S:75 R:90 A:80 D:70 |
| 9 | Certain | No SPEC-_cli-fab.md mirror is created — `_cli-fab.md` has no mirror by precedent (verifier-confirmed) | Constitution mirrors exist per skill file; none exists for `_cli-fab.md` and prior changes did not create one | S:85 R:90 A:90 D:85 |
| 10 | Confident | `batch switch`/`batch archive` stderr text changes (specific resolver errors, `ERROR:` prefixes) are acceptable — no skill invokes `fab batch` | F29 verifier: zero skills invoke any `fab batch` command; only `_cli-fab.md` documents them (updated same-PR) | S:80 R:85 A:85 D:75 |

10 assumptions (4 certain, 6 confident, 0 tentative).
