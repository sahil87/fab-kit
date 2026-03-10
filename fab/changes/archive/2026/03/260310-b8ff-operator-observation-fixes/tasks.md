# Tasks: Operator Observation Fixes

**Change**: 260310-b8ff-operator-observation-fixes
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Scope `discoverPanes()` to current session and add tab field — in `src/go/fab/cmd/fab/panemap.go`, change `tmux list-panes -a -F "#{pane_id} #{pane_current_path}"` to `tmux list-panes -s -F "#{pane_id} #{window_name} #{pane_current_path}"`. Update `paneEntry` struct to include a `tab` field. Update parsing in `discoverPanes()` to split 3 fields instead of 2. This change propagates to `send-keys` since it calls `discoverPanes()`.

- [x] T002 [P] Add Tab column to pane-map output — in `src/go/fab/cmd/fab/panemap.go`, add `tab` field to `paneRow` struct. Update `resolvePane()` to populate `tab` from the `paneEntry`. Update `printPaneTable()` to include Tab as the second column (between Pane and Worktree): headers become 6-element array `[Pane, Tab, Worktree, Change, Stage, Agent]` with matching format string and width calculation.

- [x] T003 [P] Add `runtime is-idle` subcommand — in `src/go/fab/cmd/fab/runtime.go`, add `runtimeIsIdleCmd()` function returning a `*cobra.Command` with `Use: "is-idle <change>"`, `Args: cobra.ExactArgs(1)`. Implementation: resolve change via `resolve.ToFolder`, load runtime file via `loadRuntimeFile`, check for `agent.idle_since` entry. Output `idle {duration}` (reuse `formatIdleDuration` from panemap.go), `active`, or `unknown`. Always exit 0. Register the command in `runtimeCmd()` via `cmd.AddCommand(runtimeIsIdleCmd())`.

## Phase 2: Skill & Spec Updates

- [x] T004 [P] Update operator skill — in `fab/.kit/skills/fab-operator1.md`: (a) Replace `fab status show --all` with `fab pane-map` in § Purpose, § Orientation on Start, § State Re-derivation, and § UC5 Status dashboard. (b) Keep `fab status show --all` in the § Outside tmux fallback section. (c) Update § Pre-Send Validation to reference `fab runtime is-idle` for idle checks.

- [x] T005 [P] Update operator spec — in `docs/specs/skills/SPEC-fab-operator1.md`: (a) Update Summary to reference `fab pane-map` instead of `fab status show --all`. (b) Update Primitives table: replace `fab status show --all` row with `fab pane-map` row, add `fab runtime is-idle` row. (c) Update Discovery section pane map structure to include Tab column. (d) Update Use Cases 1, 2, 5 to reference `fab pane-map` instead of `fab status show --all`. (e) Update Guardrails § Always re-derive state. (f) Update Relationship table.

- [x] T006 [P] Update `_scripts.md` — in `fab/.kit/skills/_scripts.md`: (a) Update `fab pane-map` section: add Tab column to the column table and example output. (b) Update `fab runtime` subcommand table to include `is-idle`. (c) Update `fab send-keys` Pane resolution description to note `-s` (session scope) instead of `-a`.

## Phase 3: Build Verification

- [x] T007 Build Go binary and verify — run `cd src/go/fab && go build ./cmd/fab/` to verify compilation. Run `go vet ./cmd/fab/` for static analysis.

---

## Execution Order

- T001 and T002 are tightly coupled (T002 depends on T001's `tab` field in `paneEntry`), so T001 should be done first, then T002
- T003 is independent of T001/T002
- T004, T005, T006 are independent of each other and can run in parallel
- T007 depends on T001, T002, T003 completing
