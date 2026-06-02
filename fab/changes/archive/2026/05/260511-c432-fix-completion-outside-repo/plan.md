# Plan: Remove router config gate + add `shell-init` wrapper

**Change**: 260511-c432-fix-completion-outside-repo
**Status**: In Progress
**Intake**: `intake.md`
**Spec**: `spec.md`

## Tasks

<!-- Sequential work items for the apply stage. Checked off [x] as completed. -->

### Phase 1: Router Refactor

- [x] T001 Remove `fabGoNoConfigArgs` map, `resolveFabVersion` helper, and the "Not in a fab-managed repo" exit path from `src/go/fab-kit/cmd/fab/main.go`. Inline version selection in `execFabGo` (`if cfg != nil { v = cfg.FabVersion } else { v = version }`), matching the existing pattern in `printHelp`. Preserve the corrupted-config (parse error) hard-error path.
- [x] T002 Delete `TestFabGoNoConfigArgs` and `TestResolveFabVersion` from `src/go/fab-kit/cmd/fab/main_test.go`. Keep `TestFabKitArgs`, `TestVersion`, and `TestPrintVersion` unchanged. Remove the now-unused `internal` import if no other test needs it.
- [x] T003 Run `go test ./src/go/fab-kit/cmd/fab/...` to verify the router still builds and passes.

### Phase 2: `shell-init` Command

- [x] T004 Create `src/go/fab/cmd/fab/shellinit.go` with factory `shellInitCmd() *cobra.Command`. Accept exactly one positional arg (`bash`, `zsh`, or `fish`). Delegate to `cmd.Root().GenBashCompletion(out)` / `GenZshCompletion(out)` / `GenFishCompletion(out, true)` based on the arg (Cobra v1.8.1 signatures verified). Reject unknown shells with a clear error listing the supported set. Match the convention of sibling files (`kitpath.go`, `operator.go`, `fabhelp.go`).
- [x] T005 Register `shellInitCmd()` in the `root.AddCommand(...)` block in `src/go/fab/cmd/fab/main.go`.
- [x] T006 Create `src/go/fab/cmd/fab/shellinit_test.go` covering: bash/zsh/fish produce non-empty output, zsh output starts with `#compdef fab`, unknown shell (`powershell`) returns an error, no-arg returns an error, too-many-args returns an error.
- [x] T007 Run `go test ./src/go/fab/cmd/fab/...` to verify shell-init and existing tests pass.

### Phase 3: Documentation

- [x] T008 Update `README.md` to add the `eval "$(fab shell-init zsh)"` one-liner with bash/fish equivalents in the install/quickstart section.
- [x] T009 Update `docs/specs/architecture.md` to document the router's always-route policy and version-selection rule. If any `fabGoNoConfigArgs` mention exists, remove it.

## Execution Order

- T001 blocks T002 blocks T003 (router refactor sequence)
- T004 blocks T005 blocks T006 blocks T007 (shell-init sequence)
- T008 and T009 are independent of code phases but should run after T003 and T007 succeed

## Acceptance

### Functional Completeness

- [ ] A-001 Always-Route Policy: router execs `~/.fab-kit/versions/{cfg.FabVersion}/fab-go` when config present, `~/.fab-kit/versions/{routerVersion}/fab-go` when absent; no router-side exit on missing config.
- [ ] A-002 Removal of Router-Side Config Gate: `fabGoNoConfigArgs`, `resolveFabVersion`, and the "Not in a fab-managed repo" string are all absent from `src/go/fab-kit/cmd/fab/main.go`.
- [ ] A-003 fab-go Self-Guards Are Authoritative: router no longer pre-empts per-command guards; existing guards in fab-go remain the source of "needs config" errors.
- [ ] A-004 `fab shell-init <shell>` command exists at `src/go/fab/cmd/fab/shellinit.go`, registered in `main.go`, exposing the synopsis `fab shell-init <bash|zsh|fish>`.
- [ ] A-005 Implementation Delegation: `shell-init` invokes `cmd.Root().GenBashCompletion / GenZshCompletion / GenFishCompletion` (no re-implementation of completion logic).
- [ ] A-006 Router Test Updates: `TestFabGoNoConfigArgs` and `TestResolveFabVersion` are removed; `TestFabKitArgs`, `TestVersion`, `TestPrintVersion` remain; `go test ./src/go/fab-kit/cmd/fab/...` passes.
- [ ] A-007 `shell-init` Tests: `shellinit_test.go` exists and covers all spec scenarios; `go test ./src/go/fab/cmd/fab/...` passes.
- [ ] A-008 README Install Update: README contains an `eval "$(fab shell-init zsh)"` (or equivalent) line in an install/setup context, with bash/fish guidance.
- [ ] A-009 Architecture Spec Update: `docs/specs/architecture.md` describes the always-route policy and version-selection rule; no `fabGoNoConfigArgs` reference remains.

### Behavioral Correctness

- [ ] A-010 Corrupted-config path still hard-errors at the router (parse error from `internal.ResolveConfig`).
- [ ] A-011 `fab completion zsh`, `fab --help`, `fab kit-path` all work outside a fab repo without router-emitted errors.

### Scenario Coverage

- [ ] A-012 Spec scenarios covered: inside a fab repo, outside a fab repo, corrupted config, map/helper/message absent (grep checks).
- [ ] A-013 shell-init scenarios covered: bash/zsh/fish outputs, invalid shell, missing arg, too many args, parity with `fab completion <shell>`.

### Code Quality

- [ ] A-014 New `shellinit.go` follows the codebase's one-command-one-file factory pattern, no functions exceed the local convention's typical size, no magic strings (supported-shell list is a named slice/array).
- [ ] A-015 No duplication: `shell-init` delegates to Cobra's built-in generators rather than re-implementing completion scripts.
- [ ] A-016 Readability: inline version selection in `execFabGo` mirrors the existing pattern in `printHelp` (consistency over cleverness).

### Documentation Accuracy

- [ ] A-017 README's new snippet matches the actual command surface (`shell-init` accepts `bash`/`zsh`/`fish`).
- [ ] A-018 `docs/specs/architecture.md` text accurately reflects the post-change router behavior (project-pinned if config present, bundled otherwise; always routes).

### Cross-References

- [ ] A-019 No stale references to `fabGoNoConfigArgs` remain in `docs/specs/`. (Memory updates land at hydrate stage and are out of scope here.)
