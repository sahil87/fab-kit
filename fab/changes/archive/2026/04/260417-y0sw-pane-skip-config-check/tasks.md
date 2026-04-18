# Tasks: Pane Skip Config Check

**Change**: 260417-y0sw-pane-skip-config-check
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

<!-- All production changes confined to src/go/fab-kit/cmd/fab/main.go per spec "No Changes Outside Router" requirement. -->

- [x] T001 Add `fabGoNoConfigArgs map[string]bool` allowlist in `src/go/fab-kit/cmd/fab/main.go` alongside the existing `fabKitArgs` (currently lines 17–23). Initialize with a single entry: `"pane": true`. Add a short comment explaining the allowlist's purpose (fab-go commands exempt from the `config.yaml` requirement because they resolve state from their arguments, not from the invoker's CWD).

- [x] T002 Add a pure helper `resolveFabVersion(cfg *internal.ConfigResult, arg0 string, routerVersion string) (fabVersion string, shouldExit bool)` in `src/go/fab-kit/cmd/fab/main.go`. Logic: if `cfg != nil` return `(cfg.FabVersion, false)`; else if `fabGoNoConfigArgs[arg0]` return `(routerVersion, false)`; else return `("", true)`. No I/O, no `os.Exit`, no `syscall.Exec` — strictly pure. Place near `execFabGo` so readers see both together.

- [x] T003 Update `execFabGo` (currently lines 71–93 of `src/go/fab-kit/cmd/fab/main.go`) to call `resolveFabVersion(cfg, arg0, version)` after `ResolveConfig`. On `shouldExit == true`, print the existing `"Not in a fab-managed repo. Run 'fab init' to set one up."` message to stderr and `os.Exit(1)`. On `shouldExit == false`, proceed with `internal.EnsureCached(fabVersion)` and `syscall.Exec` as today. `arg0` is `args[0]` when `len(args) > 0`, otherwise the empty string.

- [x] T004 Update `printHelp` (currently lines 96–123 of `src/go/fab-kit/cmd/fab/main.go`) to invoke the fab-go `--help` subprocess even when `cfg == nil`. Version selection: use `cfg.FabVersion` when `cfg != nil`, else the router's build-time `version` constant. Errors from `EnsureCached` or the subprocess remain silently swallowed — the help section is simply omitted on failure. Do NOT gate the section on exemption — `printHelp` shows the full fab-go help regardless of allowlist membership so all workflow commands remain discoverable.

## Phase 2: Tests

<!-- All tests live in src/go/fab-kit/cmd/fab/main_test.go, following the existing test-alongside pattern. No network, no cache state required. -->

- [x] T005 [P] Add `TestFabGoNoConfigArgs` in `src/go/fab-kit/cmd/fab/main_test.go`. Assert `fabGoNoConfigArgs["pane"] == true`. Assert `fabGoNoConfigArgs["runtime"] == false`, `fabGoNoConfigArgs["status"] == false`, `fabGoNoConfigArgs["preflight"] == false`, `fabGoNoConfigArgs["change"] == false`, `fabGoNoConfigArgs["score"] == false`. Mirror the style of the existing `TestFabKitArgs`.

- [x] T006 [P] Add `TestResolveFabVersion` as a table-driven test in `src/go/fab-kit/cmd/fab/main_test.go`. Cases:
  - `cfg = &internal.ConfigResult{FabVersion: "1.3.7"}, arg0 = "pane", routerVersion = "dev"` → expect `("1.3.7", false)`
  - `cfg = &internal.ConfigResult{FabVersion: "1.3.7"}, arg0 = "status", routerVersion = "dev"` → expect `("1.3.7", false)`
  - `cfg = nil, arg0 = "pane", routerVersion = "1.3.7"` → expect `("1.3.7", false)`
  - `cfg = nil, arg0 = "status", routerVersion = "1.3.7"` → expect `("", true)`
  - `cfg = nil, arg0 = "", routerVersion = "1.3.7"` → expect `("", true)`

## Phase 3: Verification

- [x] T007 Run `go build ./...` from `src/go/fab-kit/` to confirm compilation. Address any errors before proceeding.

- [x] T008 Run `go test ./cmd/fab/...` from `src/go/fab-kit/` to confirm all tests pass (both new and existing). Must include `TestFabKitArgs`, `TestVersion`, `TestPrintVersion`, `TestFabGoNoConfigArgs`, `TestResolveFabVersion`.

- [x] T009 Run `go vet ./...` from `src/go/fab-kit/` to confirm no static-analysis warnings on the changed file.

---

## Execution Order

- T001 → T002 → T003 (sequential: T003 uses the helper from T002, which references the allowlist from T001)
- T004 can run after T002 (uses the same version-selection logic, either via the helper or inline)
- T005 and T006 [P] are independent of each other but require T001/T002 respectively
- T007 → T008 → T009 in Phase 3 (build before test before vet)
