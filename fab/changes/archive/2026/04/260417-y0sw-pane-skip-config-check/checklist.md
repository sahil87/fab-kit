# Quality Checklist: Pane Skip Config Check

**Change**: 260417-y0sw-pane-skip-config-check
**Generated**: 2026-04-17
**Spec**: `spec.md`

## Functional Completeness

- [x] CHK-001 fabGoNoConfigArgs allowlist: Package-level variable declared in `src/go/fab-kit/cmd/fab/main.go` with exactly one entry, `"pane": true`. No other entries.
- [x] CHK-002 Exemption check placement: In `execFabGo`, the exemption-vs-error decision happens after `internal.ResolveConfig` returns without error and before any "Not in a fab-managed repo" exit.
- [x] CHK-003 Pure helper exists: `resolveFabVersion(cfg *internal.ConfigResult, arg0 string, routerVersion string) (string, bool)` declared in `main.go`, with no I/O, no `os.Exit`, no `syscall.Exec`.
- [x] CHK-004 execFabGo uses helper: `execFabGo` calls `resolveFabVersion` and branches on its `shouldExit` return.
- [x] CHK-005 printHelp no-config path: `printHelp` invokes the fab-go `--help` subprocess regardless of whether `cfg` is nil, selecting version via `cfg.FabVersion` when present else the router's `version` constant.

## Behavioral Correctness

- [x] CHK-006 In-repo behavior unchanged: Inside a fab repo, every command (exempt or not) resolves to `cfg.FabVersion`. `fab pane map`, `fab status`, `fab runtime ...` all dispatch to `fab-go` at the project-pinned version.
- [x] CHK-007 Outside-repo exempt path: `fab pane ...` invoked outside any fab repo resolves to the router's `version` constant via `EnsureCached(version)` and successfully execs `fab-go` — no "Not in a fab-managed repo" error.
- [x] CHK-008 Outside-repo non-exempt path unchanged: Every non-exempt command (`status`, `runtime`, `preflight`, `change`, `score`, `log`, `hook`, `operator`, `batch`, `kit-path`, `fab-help`) still exits 1 with the existing "Not in a fab-managed repo" error when invoked outside a fab repo.
- [x] CHK-009 ResolveConfig error path unchanged: When `ResolveConfig` returns an error (malformed `config.yaml`, missing `fab_version`), the process still exits 1 with `"ERROR: {err}\n"` regardless of exemption.
- [x] CHK-010 printHelp in-repo unchanged: `fab --help` inside a fab repo continues to show the workflow commands section using the project-pinned `FabVersion`.
- [x] CHK-011 printHelp outside-repo expanded: `fab --help` outside a fab repo now shows the workflow commands section (using bundled version); previously the section was absent.

## Scenario Coverage

- [x] CHK-012 TestFabGoNoConfigArgs exists: Test in `main_test.go` asserts `fabGoNoConfigArgs["pane"] == true` and that `"runtime"`, `"status"`, `"preflight"`, `"change"`, `"score"` are not in the allowlist.
- [x] CHK-013 TestResolveFabVersion table covers five cases: (cfg non-nil, arg=pane), (cfg non-nil, arg=status), (cfg nil, arg=pane), (cfg nil, arg=status), (cfg nil, arg=""). Expected outputs match spec Requirement: Pure Helper scenarios.
- [x] CHK-014 Verification passes: `go build ./...`, `go test ./cmd/fab/...`, `go vet ./...` all succeed from `src/go/fab-kit/`.

## Edge Cases & Error Handling

- [x] CHK-015 Empty args case: `resolveFabVersion(nil, "", routerVersion)` returns `("", true)` — no panic on empty `arg0`.
- [x] CHK-016 Help best-effort under failure: When `EnsureCached(version)` fails during `printHelp` (e.g., network down, empty cache), the help section is silently omitted and `fab --help` still exits 0 with the workspace-commands section.
- [x] CHK-017 pane send idle guard unchanged: No file under `src/go/fab/` is modified; `pane send` without `--force` still rejects non-idle target panes regardless of invoker CWD.

## Code Quality

- [x] CHK-018 Pattern consistency: New code (allowlist declaration, helper function, updated `execFabGo` and `printHelp`) follows the style of existing code in `main.go` (same comment style, same Go idioms, consistent naming).
- [x] CHK-019 No unnecessary duplication: The version-selection logic lives in one place — `resolveFabVersion`. If `printHelp` reuses the helper, do so; if it inlines the 3-line branch, the inline version must match the helper's semantics exactly.
- [x] CHK-020 No god functions: `execFabGo` and `printHelp` remain under 50 lines each after changes (current `execFabGo` is ~22 lines, `printHelp` is ~27 lines; adding the helper call and branch should not materially increase either).
- [x] CHK-021 No magic strings: The error message `"Not in a fab-managed repo. Run 'fab init' to set one up."` is preserved verbatim (no new duplicated copies introduced if a constant is warranted — but either unchanged inline or extracted as a constant is acceptable).

## Documentation Accuracy

<!-- From fab/project/config.yaml checklist.extra_categories -->

- [x] CHK-022 Memory file cross-references: The updates planned for `docs/memory/fab-workflow/kit-architecture.md` and `docs/memory/fab-workflow/distribution.md` accurately reflect what shipped (exemption allowlist name, bundled-version fallback semantics, which commands are/aren't exempt). These will be applied during hydrate stage.

## Cross References

<!-- From fab/project/config.yaml checklist.extra_categories -->

- [x] CHK-023 Spec-to-code mapping: Every requirement in `spec.md` maps to at least one task in `tasks.md` and to at least one implementation change or test. Verified via auto-clarify at tasks stage.
- [x] CHK-024 Scope boundary: `git diff --name-only main` after implementation lists only `src/go/fab-kit/cmd/fab/main.go` and `src/go/fab-kit/cmd/fab/main_test.go` under `src/` (plus fab/changes/y0sw and docs/memory changes outside `src/`). No files under `src/go/fab/` or `src/go/fab-kit/internal/`.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
