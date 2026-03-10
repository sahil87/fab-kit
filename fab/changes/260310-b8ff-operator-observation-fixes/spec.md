# Spec: Operator Observation Fixes

**Change**: 260310-b8ff-operator-observation-fixes
**Created**: 2026-03-10
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Removing the `--all` flag from the `status show` CLI — it has legitimate non-operator uses for scripting
- Changing agent idle tracking accuracy — addressed by change `bvc6` (UserPromptSubmit hook)
- Adding persistent operator standing orders or event-driven polling

## CLI: Pane Map Session Scoping

### Requirement: Session-scoped pane discovery

`fab pane-map` SHALL list only panes belonging to the current tmux session. The tmux command MUST use `-s` (session scope) instead of `-a` (all sessions).

#### Scenario: Pane map in a multi-session environment

- **GIVEN** the user is running tmux with session "work" containing 3 panes, and a separate session "personal" containing 2 panes
- **WHEN** the user runs `fab pane-map` from session "work"
- **THEN** only the 3 panes from session "work" appear in the output
- **AND** no panes from session "personal" appear

#### Scenario: Single-session environment

- **GIVEN** only one tmux session exists with 2 panes
- **WHEN** the user runs `fab pane-map`
- **THEN** both panes appear (behavior is identical to `-a` when only one session exists)

### Requirement: Tab column in pane map output

The pane map output SHALL include a **Tab** column displaying the tmux window name (`#{window_name}`). The column MUST be positioned between Pane and Worktree.

The tmux format string MUST include `#{window_name}` alongside `#{pane_id}` and `#{pane_current_path}`:

```
tmux list-panes -s -F "#{pane_id} #{window_name} #{pane_current_path}"
```

#### Scenario: Tab column in output

- **GIVEN** a tmux session with pane `%3` in window "alpha" and pane `%7` in window "bravo"
- **WHEN** the user runs `fab pane-map`
- **THEN** the output table includes columns: `Pane  Tab  Worktree  Change  Stage  Agent`
- **AND** pane `%3` shows Tab "alpha" and pane `%7` shows Tab "bravo"

#### Scenario: Table alignment with Tab column

- **GIVEN** windows with varying name lengths (e.g., "a", "long-window-name")
- **WHEN** the user runs `fab pane-map`
- **THEN** all columns are properly aligned using the same dynamic width calculation as existing columns

## CLI: Send-Keys Session Scoping

### Requirement: Session-scoped pane resolution in send-keys

`fab send-keys` SHALL discover panes using `-s` (session scope) instead of `-a` (all sessions), consistent with pane-map. The `discoverPanes()` function is shared between pane-map and send-keys, so the change propagates to both.

#### Scenario: Send keys targets only current session panes

- **GIVEN** session "work" has pane `%3` with change "r3m7" and session "personal" has pane `%9` with change "r3m7" (same change checked out in both)
- **WHEN** the operator runs `fab send-keys r3m7 "/fab-continue"` from session "work"
- **THEN** the command sends to `%3` (current session) not `%9`

#### Scenario: Target pane in different session

- **GIVEN** the target change's pane exists only in a different tmux session
- **WHEN** the operator runs `fab send-keys <change> "<text>"`
- **THEN** the command exits 1 with `No tmux pane found for change "<folder>".`

## CLI: Runtime Is-Idle Subcommand

### Requirement: Read-only idle state query

`fab runtime` SHALL support an `is-idle <change>` subcommand that reads `.fab-runtime.yaml` and outputs the agent's idle state to stdout.

**Output contract**:

| Condition | Stdout | Exit code |
|-----------|--------|-----------|
| `agent.idle_since` exists for the change | `idle {duration}` | 0 |
| No `agent` block or no `idle_since` | `active` | 0 |
| `.fab-runtime.yaml` doesn't exist | `unknown` | 0 |

Duration format MUST match pane-map: `{N}s` (< 60s), `{N}m` (60s–59m), `{N}h` (>= 60m). Floor division.

#### Scenario: Agent is idle

- **GIVEN** `.fab-runtime.yaml` contains an `agent.idle_since` entry for change "r3m7" set 90 seconds ago
- **WHEN** the user runs `fab runtime is-idle r3m7`
- **THEN** stdout prints `idle 1m` and exit code is 0

#### Scenario: Agent is active

- **GIVEN** `.fab-runtime.yaml` exists but has no `agent.idle_since` for change "r3m7"
- **WHEN** the user runs `fab runtime is-idle r3m7`
- **THEN** stdout prints `active` and exit code is 0

#### Scenario: Runtime file missing

- **GIVEN** `.fab-runtime.yaml` does not exist
- **WHEN** the user runs `fab runtime is-idle r3m7`
- **THEN** stdout prints `unknown` and exit code is 0

#### Scenario: Standard change resolution

- **GIVEN** a valid change that can be resolved by 4-char ID, substring, or full folder name
- **WHEN** the user runs `fab runtime is-idle <any-form>`
- **THEN** the change is resolved using the same `resolve.ToFolder` mechanism as other runtime subcommands

## Operator Skill: Replace status show --all with pane-map

### Requirement: Pane-map as sole observation mechanism

The operator skill (`fab/.kit/skills/fab-operator1.md`) SHALL use `fab pane-map` as its primary observation command. All references to `fab status show --all` in the operator skill MUST be replaced with `fab pane-map`.

Specifically:
- **Orientation on Start** (§ Orientation): Replace `fab status show --all` with `fab pane-map` as the second observation step. Since pane-map already includes stage info, a single pane-map call provides the full orientation.
- **State Re-derivation** (§ State Re-derivation): Replace `fab status show --all` with `fab pane-map` in the re-query list.
- **UC5: Status dashboard**: Replace the `fab status show --all` reference with `fab pane-map`.

#### Scenario: Operator orientation on start (inside tmux)

- **GIVEN** the operator is invoked inside a tmux session with multiple fab agents running
- **WHEN** the operator performs its orientation sequence
- **THEN** it runs `fab pane-map` (which shows Pane, Tab, Worktree, Change, Stage, Agent columns)
- **AND** it does NOT run `fab status show --all`

#### Scenario: Operator state re-derivation before action

- **GIVEN** the operator is about to execute a coordination action
- **WHEN** it re-derives state
- **THEN** it runs `fab pane-map` for the pane/change/stage/agent mapping
- **AND** it runs `fab runtime` only when checking specific agents' idle state (for pre-send validation)

#### Scenario: Operator outside tmux

- **GIVEN** the operator is invoked outside a tmux session (`$TMUX` unset)
- **WHEN** it performs orientation
- **THEN** it runs `fab status show --all` as a fallback (pane-map is unavailable)
- **AND** it displays the tmux warning as before

### Requirement: Update operator spec

`docs/specs/skills/SPEC-fab-operator1.md` SHALL be updated to reflect:
- Pane map as primary observation mechanism (not `status show --all`)
- Updated pane map structure with Tab column
- `fab pane-map` replaces `status show --all` in the Primitives table, Discovery section, and Use Cases
- `fab runtime is-idle` added as a primitive for pre-send validation

#### Scenario: Spec reflects new observation model

- **GIVEN** the spec file exists
- **WHEN** the change is applied
- **THEN** the Primitives table lists `fab pane-map` (not `fab status show --all`) as the primary observation primitive
- **AND** the Discovery section references `fab pane-map` output with the Tab column
- **AND** `fab runtime is-idle` is listed as the idle check mechanism

## Documentation: Scripts Reference

### Requirement: Update _scripts.md

`fab/.kit/skills/_scripts.md` SHALL be updated to reflect:
- `fab pane-map` column list updated to include Tab column
- `fab runtime` subcommand table updated to include `is-idle`
- `fab send-keys` pane resolution description updated to note session scoping (`-s`)

#### Scenario: Scripts reference accuracy

- **GIVEN** the `_scripts.md` file documents CLI commands
- **WHEN** the change is applied
- **THEN** the pane-map section shows 6 columns: Pane, Tab, Worktree, Change, Stage, Agent
- **AND** the runtime section includes `is-idle` with its output contract
- **AND** the send-keys section notes session-scoped pane discovery

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use `-s` not `-a` for tmux pane listing in pane-map | Confirmed from intake #1 — operator should only see panes in its own session | S:95 R:90 A:90 D:95 |
| 2 | Certain | Add Tab column using `#{window_name}` between Pane and Worktree | Confirmed from intake #2 — operator needs spatial orientation beyond pane IDs | S:90 R:90 A:85 D:90 |
| 3 | Certain | `is-idle` prints `idle {duration}` / `active` / `unknown`, always exit 0 | Confirmed from intake #3 — clean single-command check for pre-send validation | S:85 R:85 A:85 D:85 |
| 4 | Certain | Replace `status show --all` with `pane-map` in operator skill only | Confirmed from intake #4 — pane-map is strictly more informative; `--all` flag stays in CLI | S:90 R:90 A:90 D:90 |
| 5 | Certain | Apply same `-s` scoping to send-keys pane discovery via shared `discoverPanes()` | Upgraded from intake Confident #5 — discoverPanes is shared, change propagates automatically | S:85 R:85 A:90 D:90 |
| 6 | Confident | Tab column placed between Pane and Worktree | Confirmed from intake #6 — natural reading order: pane ID → tab name → worktree path | S:75 R:90 A:80 D:75 |
| 7 | Certain | Go-only implementation — no Rust binary changes | Upgraded from intake Confident #7 — Go is the distributed binary per config and constitution | S:85 R:85 A:85 D:90 |
| 8 | Confident | Operator skill keeps `status show --all` as fallback when outside tmux | New — outside-tmux mode has no pane-map; status show --all is the only observation option | S:70 R:90 A:80 D:75 |
| 9 | Confident | `send-keys` pane discovery note added to `_scripts.md` but no behavior change to the command doc | New — the doc already says `tmux list-panes -a`; updating to note `-s` is a doc fix | S:75 R:90 A:85 D:80 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
