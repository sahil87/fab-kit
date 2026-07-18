# Plan: Help Examples — cobra `Example:` blocks on user-facing commands

**Change**: 260717-b91h-help-examples
**Intake**: `intake.md`

## Requirements

### CLI Help: Example Invocations

#### R1: `batch archive` carries example invocations
The `fab batch archive` command definition (`src/go/fab/cmd/fab/batch_archive.go`) SHALL populate cobra's `Example:` field with runnable invocations covering the `--dry-run`, `--yes`, and explicit-args flag combinations, formatted with a two-space indent per line and a `#` comment line above each invocation.

- **GIVEN** a user runs `fab batch archive -h`
- **WHEN** cobra renders the help
- **THEN** an `Examples:` section appears showing the `--dry-run` preview, the `--yes` archive-all, and the explicit-change-args invocations
- **AND** no signature, flag, or behavior of the command changes

#### R2: `batch switch` carries example invocations
The `fab batch switch` command definition (`src/go/fab/cmd/fab/batch_switch.go`) SHALL populate `Example:` with invocations covering `--list`, explicit-args, and `--all`, in the two-space-indent + `#`-comment format.

- **GIVEN** a user runs `fab batch switch -h`
- **WHEN** cobra renders the help
- **THEN** an `Examples:` section shows the `--list`, explicit-args, and `--all` invocations
- **AND** no signature, flag, or behavior of the command changes

#### R3: `config init` carries example invocations
The `fab config init` command definition (`src/go/fab/cmd/fab/config.go`, `configInitCmd`) SHALL populate `Example:` with invocations covering both mutually-exclusive modes: `--system` and `--project` (including the repeatable `--source-path`/`--test-path` seed form).

- **GIVEN** a user runs `fab config init -h`
- **WHEN** cobra renders the help
- **THEN** an `Examples:` section shows the `--system` scaffold, the `--project` registry-generation, and the repeatable-flag seed invocations
- **AND** no signature, flag, or behavior of the command changes

#### R4: `config show` carries example invocations
The `fab config show` command definition (`src/go/fab/cmd/fab/config.go`, `configShowCmd`) SHALL populate `Example:` with invocations covering the bare effective-config print and the `--origin` provenance annotation.

- **GIVEN** a user runs `fab config show -h`
- **WHEN** cobra renders the help
- **THEN** an `Examples:` section shows the bare and `--origin` invocations
- **AND** no signature, flag, or behavior of the command changes

#### R5: `resolve` carries example invocations
The `fab resolve` command definition (`src/go/fab/cmd/fab/resolve.go`) SHALL populate `Example:` with invocations covering the default `--id` output, `--folder`, `--status`, and the `--pane` output with a `-L`/`--server` socket selector.

- **GIVEN** a user runs `fab resolve -h`
- **WHEN** cobra renders the help
- **THEN** an `Examples:` section shows the default, `--folder`, `--status`, and `--pane -L` invocations
- **AND** no signature, flag, or behavior of the command changes

#### R6: `score` carries example invocations
The `fab score` command definition (`src/go/fab/cmd/fab/score.go`) SHALL populate `Example:` with invocations covering the compute-and-persist call and the read-only `--check-gate --stage intake` gate check.

- **GIVEN** a user runs `fab score -h`
- **WHEN** cobra renders the help
- **THEN** an `Examples:` section shows the compute and the `--check-gate` invocations
- **AND** no signature, flag, or behavior of the command changes

#### R7: `dispatch start` carries example invocations
The `fab dispatch start` command definition (`src/go/fab/cmd/fab/dispatch_start.go`) SHALL populate `Example:` with invocations that make the stdin-fed prompt shape explicit (`< prompt.md`), covering both the default launch and the `--timeout` form.

- **GIVEN** a user runs `fab dispatch start -h`
- **WHEN** cobra renders the help
- **THEN** an `Examples:` section shows the stdin-fed launch and the `--timeout` invocations, both conveying that the prompt is read from stdin
- **AND** no signature, flag, or behavior of the command changes

### CLI Help: Conformance Test

#### R8: A test pins non-empty `Example` on the 7 target commands
A new Go test (in `src/go/fab/cmd/fab/`) SHALL walk the real assembled command tree (`newRootCmd()`) and assert that each of the 7 target commands (`batch archive`, `batch switch`, `config init`, `config show`, `resolve`, `score`, `dispatch start`) has a non-empty `Example` field. It MAY additionally assert the two-space-indent formatting on non-blank example lines.

- **GIVEN** the assembled command tree from `newRootCmd()`
- **WHEN** the test resolves each of the 7 target command paths
- **THEN** each resolves to a real command and its `Example` field is non-empty
- **AND** the test fails if any target's `Example` is empty or the path no longer resolves

### Non-Goals

- No changes to any command signature, flag set, argument spec, or runtime behavior — help text only.
- No `_cli-fab.md` update — that constraint keys on command *signatures*, which are untouched (disposition recorded in the intake).
- No custom help template — the stock cobra template renders `Examples:` before the Flags section; that placement is accepted (intake § Explicit non-changes).
- No sweep beyond the 7 audit-named commands; single-flag and internal/hidden commands stay example-free.
- No manual `docs/site` or shll.ai work — `help-dump` byte-preserves the `-h` text and the examples propagate on the next release automatically.

### Design Decisions

1. **Populate cobra's native `Example:` field, not a bespoke help template**: `Example:` renders in `-h` automatically and flows into `help-dump` output (each node's `Text` is the raw `UsageString`) with zero extra plumbing — *Why*: satisfies principle №3's layered-help obligation with the minimum surface, and principle №7 (compose, don't reinvent) favors the native field — *Rejected*: a custom help template to force literal "after the flags" placement (no toolkit tool sets a placement precedent; №3's enforcement receipt is help-dump conformance, not placement).
2. **Conformance test walks the real `newRootCmd()` tree via `root.Find`, not the `help-dump` JSON**: `buildNode` does not serialize the `Example` field, so the JSON dump cannot observe it — *Why*: `root.Find([]string{...})` resolves each target on the actual `*cobra.Command` object where `.Example` lives, mirroring the `lifecycle_collision_test.go` real-tree-walk pattern — *Rejected*: extending `Node`/`buildNode` to carry `Example` (out of scope — the help-dump contract JSON envelope is frozen and this change ships zero help-dump behavior changes).

## Tasks

### Phase 2: Core Implementation

- [x] T001 [P] Populate `Example:` on `batchArchiveCmd()` in `src/go/fab/cmd/fab/batch_archive.go` (`--dry-run` / `--yes` / explicit-args; two-space indent, `#` comment above each) <!-- R1 -->
- [x] T002 [P] Populate `Example:` on `batchSwitchCmd()` in `src/go/fab/cmd/fab/batch_switch.go` (`--list` / explicit-args / `--all`) <!-- R2 -->
- [x] T003 [P] Populate `Example:` on `configInitCmd()` in `src/go/fab/cmd/fab/config.go` (`--system` / `--project` / repeatable seed flags — both mutually-exclusive modes shown) <!-- R3 -->
- [x] T004 [P] Populate `Example:` on `configShowCmd()` in `src/go/fab/cmd/fab/config.go` (bare / `--origin`) <!-- R4 -->
- [x] T005 [P] Populate `Example:` on `resolveCmd()` in `src/go/fab/cmd/fab/resolve.go` (default `--id` / `--folder` / `--status` / `--pane -L`) <!-- R5 -->
- [x] T006 [P] Populate `Example:` on `scoreCmd()` in `src/go/fab/cmd/fab/score.go` (compute / `--check-gate --stage intake`) <!-- R6 -->
- [x] T007 [P] Populate `Example:` on `dispatchStartCmd()` in `src/go/fab/cmd/fab/dispatch_start.go` (stdin-fed launch `< prompt.md` / `--timeout`) <!-- R7 -->

### Phase 3: Integration & Edge Cases

- [x] T008 Add conformance test `src/go/fab/cmd/fab/examples_test.go` walking `newRootCmd()` and asserting non-empty `Example` on the 7 target commands (plus optional two-space-indent formatting assertion) <!-- R8 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab batch archive`'s `Example:` field is populated with `--dry-run`, `--yes`, and explicit-args invocations in the two-space-indent + `#`-comment format
- [x] A-002 R2: `fab batch switch`'s `Example:` field is populated with `--list`, explicit-args, and `--all` invocations
- [x] A-003 R3: `fab config init`'s `Example:` field is populated and shows both `--system` and `--project` modes (including repeatable seed flags)
- [x] A-004 R4: `fab config show`'s `Example:` field is populated with the bare and `--origin` invocations
- [x] A-005 R5: `fab resolve`'s `Example:` field is populated with the default, `--folder`, `--status`, and `--pane -L` invocations
- [x] A-006 R6: `fab score`'s `Example:` field is populated with the compute and `--check-gate` invocations
- [x] A-007 R7: `fab dispatch start`'s `Example:` field is populated and conveys the stdin-fed prompt shape (`< prompt.md`) plus the `--timeout` form
- [x] A-008 R8: a test walks `newRootCmd()` and asserts non-empty `Example` on all 7 target commands; it passes with the change and would fail if any target's `Example` were empty

### Behavioral Correctness

- [x] A-009 R1: `fab batch archive -h` renders an `Examples:` section and the command's flags/args/runtime behavior are unchanged
- [x] A-010 R7: `fab dispatch start -h` renders an `Examples:` section; no flag, arg, or the stdin-reading behavior changes

### Scenario Coverage

- [x] A-011 R8: the conformance test resolves each target command path against the real assembled tree (no path fails to resolve)

### Edge Cases & Error Handling

- [x] A-012 R3: the `config init` examples show both mutually-exclusive modes (`--system` and `--project`), not just one — a reader learns both invocation shapes

### Code Quality

- [x] A-013 Pattern consistency: the `Example:` blocks and the new test follow the surrounding cobra command-definition style and the existing `*_test.go` real-tree-walk pattern (`lifecycle_collision_test.go`)
- [x] A-014 No unnecessary duplication: the test reuses `newRootCmd()` (and cobra's `Find`) rather than reimplementing tree assembly or a bespoke command registry
- [x] A-015 Magic strings/numbers: the 7 target command paths are named as clear data (e.g. a slice of path slices), not scattered literals
- [x] A-016 Go changes ship tests: the Go changes are accompanied by the conformance test in the same change (project review rule / Constitution VII)
- [x] A-017 Canonical source only: all edits are under `src/go/` — none under `.claude/skills/`

### Documentation Accuracy

- [x] A-018 No `_cli-fab.md` change is required (zero signature changes) and none is made; the disposition is recorded in the intake

### Cross References

- [x] A-019 The change stays scoped to the 7 audit-named commands; no example is added to any other command (matching the intake's deferred-item scope)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. (Help-text-only additions: the `Example:` fields are net-new content on 7 existing command definitions plus one new conformance test; no existing code path, symbol, or config is superseded.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the 7 audit-named commands; no example added elsewhere | Backlog entry + ptwh conformance report both enumerate them; intake Assumption 1 (Certain) | S:85 R:80 A:85 D:80 |
| 2 | Certain | Use cobra's native `Example:` field with stock template placement (no custom help template) | Intake Assumption 2 + verified live that no toolkit tool sets a placement precedent; principle №7 favors the native field | S:75 R:85 A:80 D:75 |
| 3 | Confident | Example block format: two-space indent, `#` comment above each invocation, 2–4 examples per command | Cobra's two-space indent is the ecosystem convention; intake Assumption 3; wording may be tightened but flag combinations preserved | S:60 R:90 A:70 D:65 |
| 4 | Confident | Conformance test walks the real `newRootCmd()` tree via `root.Find` and asserts non-empty `Example`, not the help-dump JSON (which drops `Example`) | `buildNode` does not serialize `Example`; `Find` resolves the real `*cobra.Command`; mirrors `lifecycle_collision_test.go`'s real-tree-walk pattern; extending the frozen help-dump JSON envelope is out of scope | S:65 R:80 A:80 D:70 |
| 5 | Confident | Test lives in `src/go/fab/cmd/fab/examples_test.go` (new file), following the per-surface test-file naming | Intake names this file; matches the existing `*_test.go` naming (e.g. `helpdump_test.go`, `resolve_test.go`) | S:70 R:90 A:80 D:75 |
| 6 | Confident | No `_cli-fab.md` change (zero signature changes); ship the conformance test | "Go changes ship tests" is a project review rule; the `_cli-fab.md` constraint keys on command *signatures*, untouched here; intake Assumption 4 | S:65 R:85 A:80 D:70 |

6 assumptions (2 certain, 4 confident, 0 tentative).
