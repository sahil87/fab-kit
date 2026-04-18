# Spec: Operator Spawn Add Fab Sync

**Change**: 260405-xh08-operator-spawn-add-fab-sync
**Created**: 2026-04-06
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Changing `--reuse` semantics in any way beyond adding the init call — branch resolution, collision detection, and the printed path remain unchanged
- Making init failure fatal on `--reuse` — the worktree already exists and may have working skills; a transient `fab sync` error must not abort autopilot respawns
- Changing the non-reuse init path (lines 222–240) — it already runs correctly
- Adding a new flag or environment variable to control reuse-init behavior separately from `--worktree-init`

---

## `wt` Package: Default Init Script

### Requirement: Canonical Default in `InitScriptPath()`

`InitScriptPath()` in `src/go/wt/internal/worktree/context.go` SHALL return `"fab sync"` when the `WORKTREE_INIT_SCRIPT` environment variable is not set.

**Rationale**: `"fab-kit sync"` was the correct invocation before the three-binary consolidation. After consolidation, `fab sync` is the canonical routing command — users type `fab sync`, not `fab-kit sync`. Any environment that relies on the default (no `WORKTREE_INIT_SCRIPT` override) currently calls the wrong binary command. `fab sync` routes to `fab-kit sync` internally, so the behavior is identical; only the entry point changes.

#### Scenario: Default returned when env var absent

- **GIVEN** `WORKTREE_INIT_SCRIPT` is not set (empty string)
- **WHEN** `InitScriptPath()` is called
- **THEN** it returns `"fab sync"`

#### Scenario: Custom script respected when env var set

- **GIVEN** `WORKTREE_INIT_SCRIPT` is set to `"custom/init.sh"`
- **WHEN** `InitScriptPath()` is called
- **THEN** it returns `"custom/init.sh"`
- **AND** the default `"fab sync"` is not used

### Requirement: Test Assertion Updated

`TestInitScriptPath_Default` in `src/go/wt/internal/worktree/context_test.go` SHALL assert `"fab sync"` instead of `"fab-kit sync"`.

#### Scenario: Updated test passes

- **GIVEN** the test sets `WORKTREE_INIT_SCRIPT` to `""`
- **WHEN** `go test ./internal/worktree/...` is run
- **THEN** `TestInitScriptPath_Default` passes asserting `"fab sync"`

---

## `wt create`: Init on `--reuse`

### Requirement: Init Script Runs in the `--reuse` Early-Return Block

When `wt create --reuse` finds a name collision and reuses the existing worktree, AND `worktreeInit` is `"true"`, the init script SHALL be executed on the existing worktree path before the command returns.

**Rationale**: The current code returns immediately at line 183 (`return nil`) without reaching the init block at line 222. This silently skips `fab sync`, leaving `.claude/skills/` stale if the worktree was created before the last sync. The fix inserts the same init call pattern inside the `--reuse` collision block.

#### Scenario: Init runs on reuse when `--worktree-init` is default

- **GIVEN** a worktree named `"alpha"` already exists
- **AND** `WORKTREE_INIT_SCRIPT` is set to a script that writes `.init-script-ran` in the worktree
- **AND** `--worktree-init` is not passed (defaults to `"true"`)
- **WHEN** `wt create --non-interactive --reuse --worktree-name alpha` is executed
- **THEN** the init script is executed with `cmd.Dir` set to the existing worktree path
- **AND** `.init-script-ran` is present in the existing worktree
- **AND** the command exits 0 and prints the worktree path to stdout

#### Scenario: Init skipped on reuse when `--worktree-init false`

- **GIVEN** a worktree named `"beta"` already exists
- **AND** `WORKTREE_INIT_SCRIPT` is set to a script that writes `.init-script-ran`
- **WHEN** `wt create --non-interactive --reuse --worktree-name beta --worktree-init false` is executed
- **THEN** the init script is NOT executed
- **AND** `.init-script-ran` is absent from the worktree
- **AND** the command exits 0 and prints the worktree path to stdout

#### Scenario: Init failure on reuse is non-fatal

- **GIVEN** a worktree named `"gamma"` already exists
- **AND** `WORKTREE_INIT_SCRIPT` is set to a command that is not on PATH
- **WHEN** `wt create --non-interactive --reuse --worktree-name gamma` is executed
- **THEN** `RunWorktreeSetup` is called and silently skips (returns nil for unavailable commands per existing `RunWorktreeSetup` behavior)
- **AND** the command still exits 0 and prints the worktree path to stdout
<!-- assumed: non-fatal — RunWorktreeSetup already silently skips unavailable commands; init error return is discarded with `_ =` in the reuse block consistent with the intake proposal -->

#### Scenario: No init regression on fresh create (non-reuse)

- **GIVEN** no worktree named `"delta"` exists
- **WHEN** `wt create --non-interactive --worktree-name delta --worktree-init false` is executed
- **THEN** the init block at line 222 is not reached (worktree-init is false)
- **AND** no behavioural change occurs for the non-reuse path

### Requirement: New Test for Reuse Init Behavior

`src/go/wt/cmd/create_test.go` SHALL include `TestCreate_ReuseRunsInitScript` verifying that `--reuse` on a pre-existing worktree executes the init script when `--worktree-init` is default (true).

The test SHALL follow the same pattern as `TestCreate_InitScriptRuns`:
1. Create a test repo
2. Create a named init script and commit it
3. Create a worktree first (so the collision will trigger reuse)
4. Run `wt create --non-interactive --reuse --worktree-name <name>` with `WORKTREE_INIT_SCRIPT` set
5. Assert that `.init-script-ran` exists in the reused worktree

#### Scenario: Test validates reuse init

- **GIVEN** `TestCreate_ReuseRunsInitScript` exists in `create_test.go`
- **WHEN** `go test ./cmd/...` is run
- **THEN** `TestCreate_ReuseRunsInitScript` passes confirming init script ran
- **AND** `TestCreate_ReuseExisting` still passes (stdout path unchanged)

---

## Memory: kit-architecture Update

### Requirement: `wt create --reuse` Description Updated

The `wt create` description in `docs/memory/fab-workflow/kit-architecture.md` (the `### wt Binary` section) SHALL document that `--reuse` now also runs the init script on the existing worktree when `--worktree-init` is `"true"` (the default).

The update is a hydrate-stage artifact — not required for the apply stage.

#### Scenario: Memory reflects the fixed behavior

- **GIVEN** `kit-architecture.md` currently describes `wt create --reuse` without mentioning init script execution
- **WHEN** the hydrate stage runs
- **THEN** the updated entry includes a note that `--reuse` runs the init script (via `RunWorktreeSetup` in force mode) before returning, and that init failure is non-fatal

---

## Design Decisions

1. **Init failure is non-fatal on `--reuse`**
   - *Why*: The reuse path is typically used by operator autopilot respawns. A transient `fab sync` failure (e.g., network unavailable, binary not found) must not abort the respawn — the existing worktree may have perfectly functional skills from the prior session. A hard exit here would introduce a new failure mode that did not exist before the fix.
   - *Rejected*: Fatal init failure on `--reuse`. While more defensive, it would break autopilot respawns every time `fab` is temporarily unavailable, which is a worse tradeoff than occasionally missing a sync.

2. **`worktreeInit == "true"` gate preserved in the reuse block**
   - *Why*: Consistent with the non-reuse path (line 222). `--worktree-init false` is a valid suppression flag for automation scripts that pre-configure the worktree; it should suppress init on reuse too.
   - *Rejected*: Always run init on reuse regardless of `--worktree-init`. Would break existing callers that pass `--worktree-init false` specifically to skip init.

3. **`RunWorktreeSetup` called with `"force"` mode on `--reuse`**
   - *Why*: The `--reuse` path is already inside the non-interactive code path (used by operator and batch commands). `"force"` mode skips the `ConfirmYesNo` prompt, consistent with how the non-interactive fresh-create path calls `RunWorktreeSetup`. Prompting during a reuse would be surprising and unusable in automation.
   - *Rejected*: Using `""` (prompt mode). Would stall operator autopilot with a confirmation prompt.

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Change type is `fix` | Confirmed from intake #1 — keyword "fix" in description; regression in `--reuse` path and stale default | S:90 R:90 A:95 D:90 |
| 2 | Certain | Default `WORKTREE_INIT_SCRIPT` changes to `"fab sync"` in `context.go` | Confirmed from intake #2 — `fab sync` is canonical post-consolidation; source code verified at line 180 returning `"fab-kit sync"` | S:95 R:90 A:95 D:95 |
| 3 | Certain | `--reuse` path in `create.go` must run init before `return nil` at line 183 | Confirmed from intake #3 — source code verified: early return at lines 182-184 bypasses init block at line 222 | S:95 R:90 A:95 D:95 |
| 4 | Confident | Init failure on `--reuse` is non-fatal (error discarded with `_ =`) | Confirmed from intake #4 — `RunWorktreeSetup` already silently skips unavailable commands; existing worktree may have functional skills; hard failure would break autopilot respawns | S:70 R:65 A:75 D:70 |
| 5 | Confident | `--worktree-init false` suppresses init even on `--reuse` | Upgraded from intake Tentative — spec-level analysis confirms: the `worktreeInit == "true"` gate is the natural insertion point, consistent with non-reuse path design; no ambiguity about which gate to use | S:80 R:70 A:80 D:75 |
| 6 | Certain | `RunWorktreeSetup` called with `"force"` mode (no confirmation prompt) | Derived from spec-level analysis — the `--reuse` path is always non-interactive (used by operator, batch); prompt mode would stall automation; consistent with fresh-create non-interactive path | S:90 R:85 A:90 D:90 |

6 assumptions (4 certain, 2 confident, 0 tentative, 0 unresolved).
