# Quality Checklist: Window Prefix Primitives and Done-Marker on Removal

**Change**: 260423-rxu3-window-prefix-primitives
**Generated**: 2026-04-23
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 Parent-Group Wiring: `fab pane --help` lists `window-name` alongside the existing four subcommands. `fab pane window-name --help` lists exactly `ensure-prefix` and `replace-prefix`.
- [ ] CHK-002 ensure-prefix Behavior: `ensure-prefix <pane> <char>` prepends `<char>` when absent, no-ops when already present (literal prefix check), uses `tmux display-message` + `tmux rename-window` via the `WithServer` argv builder.
- [ ] CHK-003 replace-prefix Behavior: `replace-prefix <pane> <from> <to>` renames when `<from>` is a literal prefix, no-ops (exit 0) when `<from>` is absent, accepts empty `<to>` for prefix removal, rejects empty `<from>` with exit 3.
- [ ] CHK-004 Exit-Code Scheme: both subcommands exit 0 on success/no-op, 2 on pane missing (tmux stderr "can't find pane" / "no such pane" / "pane not found"), 3 on any other tmux error (tmux not running, socket unreachable, rename failed) or usage error (empty `<char>` or `<from>`). No `$TMUX` gating — tmux's exec failure surfaces as exit 3.
- [ ] CHK-005 Output Modes: plain output is `renamed: <old> -> <new>` on rename and empty on no-op. `--json` flag emits `{"pane","old","new","action"}` where `action` is `"renamed"` or `"noop"`.
- [ ] CHK-006 Inherited `--server` Flag: both subcommands honor the persistent `--server` / `-L` flag; every internal `tmux` call runs with `-L <server>` when non-empty. Verified via argv-builder tests.
- [ ] CHK-007 Source Layout: implementation lives in `src/go/fab/cmd/fab/pane_window_name.go`; shared read helper lives in `internal/pane/pane.go`; tests in `pane_window_name_test.go` follow the argv-capture pattern.
- [ ] CHK-008 Enrollment Uses ensure-prefix: `src/kit/skills/fab-operator.md` §4 Enrollment invokes `fab pane window-name ensure-prefix <pane> »` (single line; no inline `tmux display-message` / `case` / `rename-window` block).
- [ ] CHK-009 Removal Swaps to Done-Marker: `src/kit/skills/fab-operator.md` §4 Removal invokes `fab pane window-name replace-prefix <pane> » ›` on every removal path and treats exit 2 as successful removal.
- [ ] CHK-010 Removal Paths Uniform: the same `replace-prefix » ›` call is issued for terminal stage, `stop_stage` reached, pane death, and explicit stop — no per-cause branching at the invocation site.
- [ ] CHK-011 Done-Marker Character: `›` (U+203A) is the hardcoded literal at the removal site in `fab-operator.md`; no config.yaml field is introduced for it.
- [ ] CHK-012 Spec File Sync: `docs/specs/skills/SPEC-fab-operator.md` Section Structure item 4 is updated to reference both primitives.

## Behavioral Correctness

- [ ] CHK-013 Removal No Longer Persists `»`: verify the prior rule "the `»` prefix persists forever after removal" is gone — both from the operator skill prose and from any associated cross-references in the file. Removal now swaps to `›`, never leaves `»`.
- [ ] CHK-014 User-Rename Guard Is Primitive-Enforced: the operator file no longer claims "a user-renamed window is protected by the no-restore rule"; the new text attributes the protection to `replace-prefix`'s literal prefix guard.

## Removal Verification

- [ ] CHK-015 Deprecated Rule Deleted: the exact sentence "The window name is **not** restored on removal — the `»` prefix persists. Users who want it removed rename the window manually (`Ctrl-b ,`)." is removed from `src/kit/skills/fab-operator.md` §4.
- [ ] CHK-016 Inline Shell Deleted: the three-line `tmux display-message` / `case` / `tmux rename-window` block in `src/kit/skills/fab-operator.md` §4 is removed. No other skill or script retains a copy of the inline algorithm.

## Scenario Coverage

- [ ] CHK-017 Scenario — Prepends when prefix absent: covered by unit test on `ensure-prefix` argv or integration fake.
- [ ] CHK-018 Scenario — No-op when prefix present: covered by unit test asserting no rename argv is produced.
- [ ] CHK-019 Scenario — Multi-codepoint prefix accepted: covered by a test passing a multi-char `<char>` argument and asserting no width validation error.
- [ ] CHK-020 Scenario — Swap when `<from>` present: covered by test on `replace-prefix` argv generation.
- [ ] CHK-021 Scenario — No-op (user-rename guard) when `<from>` absent: covered by test exercising the guard branch.
- [ ] CHK-022 Scenario — Empty `<to>` removes prefix: covered by test with empty `<to>`.
- [ ] CHK-023 Scenario — Empty `<from>` rejected (exit 3): covered by test asserting the usage-error path.
- [ ] CHK-024 Scenario — Empty `<char>` / `<from>` rejected with exit 3: covered by the early usage-error check in both RunE implementations; `$TMUX` is intentionally NOT gated (Copilot PR feedback — lets the primitives work from non-tmux contexts via `--server`). Tmux-not-running surfaces as exit 3 via tmux's own exec failure.
- [ ] CHK-025 Scenario — Non-existent pane exits 2: covered by test on the `ValidatePane` error path (mockable via argv capture or error-injection).
- [ ] CHK-026 Scenario — `--server` scopes tmux calls: covered by argv-builder test asserting `-L <server>` is prepended.

## Edge Cases & Error Handling

- [ ] CHK-027 Pane death between read and rename: the rename call's error path maps to exit 2 or 3 appropriately; operator removal path treats exit 2 as success.
- [ ] CHK-028 Argument count mismatches: cobra `ExactArgs(2)` (ensure) and `ExactArgs(3)` (replace) produce clear usage errors for wrong arg counts.
- [ ] CHK-029 Concurrent rename safety: the command performs one read + one write per invocation with no cross-command locking; two concurrent invocations on the same pane may race harmlessly (tmux's last-writer-wins is acceptable; documented via `replace-prefix`'s guard being the intended atomicity boundary).
- [ ] CHK-030 `--json` object keys stable: encoding/json marshal of the result struct produces a deterministic key order matching the spec shape.

## Code Quality

- [ ] CHK-031 Pattern consistency: `paneWindowNameCmd()` and subcommand functions follow the same structure as `paneSendCmd()` and `paneCaptureCmd()` (cobra command factory, `RunE` style, `WithServer` argv builder, separate test file).
- [ ] CHK-032 No unnecessary duplication: the current-name read is factored into `internal/pane.ReadWindowName`; the output emission is factored into a shared `emitResult` helper rather than inlined twice.
- [ ] CHK-033 Principle: Readability over cleverness — each subcommand's `RunE` reads top-to-bottom with minimal branching; exit-code mapping is a single helper call.
- [ ] CHK-034 Principle: Follow existing patterns — tests follow the argv-capture pattern from `pane_send_test.go` and `pane_capture_test.go`.
- [ ] CHK-035 Anti-pattern avoided — no god functions (each RunE under ~50 lines, helpers factored out).
- [ ] CHK-036 Anti-pattern avoided — no magic strings: `»` and `›` appear in `fab-operator.md` as intentional skill constants, not duplicated across Go source.

## Documentation Accuracy

- [ ] CHK-037 Help text accuracy: both subcommands have `Short` descriptions that match the behavior in the spec (idempotent prepend; atomic guarded swap).
- [ ] CHK-038 Skill text accuracy: the updated `fab-operator.md` §4 paragraphs describe the actual commands the operator issues (exact `fab pane window-name …` strings).

## Cross-References

- [ ] CHK-039 `SPEC-fab-operator.md` item 4 references both `ensure-prefix` and `replace-prefix`.
- [ ] CHK-040 `fab-operator.md` §6 step 4 parenthetical references `ensure-prefix` explicitly (no stale "inline shell" mention).

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
