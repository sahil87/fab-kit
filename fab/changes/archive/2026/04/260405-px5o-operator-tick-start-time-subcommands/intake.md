# Intake: Operator tick-start and time subcommands for fab-go

**Change**: 260405-px5o-operator-tick-start-time-subcommands
**Created**: 2026-04-05
**Status**: Draft

## Origin

> Add fab operator tick-start and fab operator time subcommands to fab-go.
>
> Background: the fab-operator skill spec requires the operator agent to display an idle message showing current time and next-tick time (HH:MM format). The agent currently outputs 'Time: now' because it has no native clock. Using shell `date` is cross-platform unreliable (GNU vs BSD syntax differs between Linux and macOS). Two new subcommands solve this:
>
> fab operator tick-start:
> - Called at the start of each operator tick (step 1 of tick behavior in fab-operator.md)
> - Side effects: reads .fab-operator.yaml, increments tick_count by 1, writes last_tick_at as ISO 8601 UTC timestamp, writes back to .fab-operator.yaml
> - Outputs to stdout: tick: N (incremented value) and now: HH:MM (local time, 24h)
> - No --interval flag — does NOT output 'next' (loop continuation is decided later in step 7)
>
> fab operator time [--interval Nm]:
> - Pure query, no side effects, no file writes
> - Outputs: now: HH:MM (always), next: HH:MM (only when --interval is given)
> - --interval accepts Go duration strings: 3m, 5m, 2m etc.
> - Called at idle message display time (after step 7 confirms loop continues)
> - Both commands use Go time package for cross-platform portability
>
> operator command refactor: currently `fab operator` is a leaf cobra command with cobra.NoArgs. Adding subcommands requires making it a parent command. The existing launch behavior (open tmux tab) remains as the default RunE when no subcommand is given.
>
> Also update:
> - src/kit/skills/fab-operator.md: step 1 snapshot uses fab operator tick-start; idle message uses fab operator time --interval {interval}
> - src/kit/skills/_cli-fab.md: add tick-start and time subcommands under the fab operator section

Interaction mode: one-shot description with full specification detail — all key behaviors, flags, and side effects provided in the description.

## Why

The `fab-operator` skill displays an idle message between ticks showing current time and next-tick time:

```
Waiting for next tick. Time: 08:26 · next tick: 08:29
```

The agent currently outputs `Time: now` as a placeholder because it has no reliable way to read the actual clock. The straightforward shell approach (`date +%H:%M`) is cross-platform unreliable: GNU `date` (Linux) and BSD `date` (macOS) have different syntax for time formatting, and agents cannot reliably know which variant is available. This breaks the idle message on one platform or the other.

The same problem applies to the tick snapshot at the start of each tick: the operator needs to atomically increment `tick_count`, write `last_tick_at`, and capture the current time — all in one operation that doesn't require platform-specific shell gymnastics.

Using Go's `time` package in `fab-go` is the correct solution: Go is already the runtime for `fab-go`, it has a portable standard library `time` package that works identically on Linux and macOS, and `fab-go` is already distributed as a compiled binary. Two subcommands under `fab operator` give the agent shell-callable clock access with exactly the semantics required: `tick-start` for the start-of-tick atomic update, and `time` for the pure idle-message query.

Without this change:
- The idle message remains broken (`Time: now`) on all platforms.
- Tick snapshots cannot reliably record `last_tick_at` or expose `tick_count` without ad hoc shell scripting.
- Cross-platform portability is compromised for anyone running the operator on macOS vs Linux.

## What Changes

### 1. `fab operator` command refactor (`src/go/fab/cmd/fab/operator.go`)

Currently `operatorCmd()` returns a leaf command with `cobra.NoArgs` and `RunE: runOperator`. To support subcommands, it must become a parent command:

- Remove `Args: cobra.NoArgs`
- Keep `RunE: runOperator` as the default (cobra calls the parent's RunE when no subcommand is given)
- Add `tick-start` and `time` as subcommands via `cmd.AddCommand(...)`

The existing `runOperator` function (opens tmux tab or switches to existing) is unchanged.

### 2. `fab operator tick-start` subcommand

New subcommand registered on the operator parent command.

**Behavior**:

1. Resolve repo root via `gitRepoRoot()`
2. Read `.fab-operator.yaml` from repo root. If missing or `tick_count` field absent, treat `tick_count` as 0.
3. Increment `tick_count` by 1
4. Write `last_tick_at` as ISO 8601 UTC timestamp (e.g. `2026-04-05T10:42:00Z`) using `time.Now().UTC().Format(time.RFC3339)`
5. Write the updated fields back to `.fab-operator.yaml` (preserve all other fields — read full file, update only these two keys, write back)
6. Output to stdout:
   ```
   tick: 48
   now: 10:42
   ```
   - `tick:` is the incremented value
   - `now:` is `time.Now().Format("15:04")` (local time, 24-hour HH:MM)
7. No `--interval` flag. Does NOT output `next:`.

**File I/O**: Uses `yq`-compatible YAML manipulation or direct Go YAML library to read/write `.fab-operator.yaml`. Preference: use Go's `gopkg.in/yaml.v3` (already a transitive dependency in the module) for consistency with the rest of `fab-go`. Read the full file into a `map[string]interface{}`, update `tick_count` and `last_tick_at`, write back.

**Error handling**: If `.fab-operator.yaml` cannot be read (missing: treat as empty, start from 0), or cannot be written: exit 1 with error on stderr.

### 3. `fab operator time` subcommand

New subcommand registered on the operator parent command.

**Behavior**:

1. Output `now: HH:MM` using `time.Now().Format("15:04")` (local time, 24-hour)
2. If `--interval <duration>` is given:
   - Parse the duration string using `time.ParseDuration(interval)` — accepts `3m`, `5m`, `2m`, etc.
   - Compute `next = time.Now().Add(parsed_duration)`
   - Output `next: HH:MM` using `next.Format("15:04")`
3. No file reads, no file writes — pure stdout query.

**Output format**:

Without `--interval`:
```
now: 08:26
```

With `--interval 3m`:
```
now: 08:26
next: 08:29
```

**Flag**: `--interval` accepts a Go duration string (e.g., `3m`, `5m`, `2m`). Invalid duration strings produce an error to stderr and exit 1.

**Error handling**: Invalid `--interval` value → stderr error + exit 1. No other failure modes (pure computation).

### 4. `fab operator tick-start` and `fab operator time` tests (`src/go/fab/cmd/fab/operator_test.go`)

Add tests alongside the existing `operator_test.go`:

- `TestOperatorTickStart_IncrementsCount`: creates a temp `.fab-operator.yaml`, runs tick-start, verifies tick_count incremented and stdout format
- `TestOperatorTickStart_MissingFile`: runs tick-start with no `.fab-operator.yaml`, verifies it creates file with tick_count=1
- `TestOperatorTime_NoInterval`: verifies stdout contains `now: ` and matches `HH:MM` pattern
- `TestOperatorTime_WithInterval`: verifies stdout contains both `now:` and `next:` in correct format
- `TestOperatorTime_InvalidInterval`: verifies exit 1 on bad duration string

### 5. `src/kit/skills/fab-operator.md` — tick behavior updates

Update **Section 4: The Loop → Tick Behavior**, Step 1 (Snapshot):

**Before** (implied current state — agent manually increments tick_count, uses shell `date`):
```
1. **Snapshot** — increment `tick_count`, run `fab pane map`, read `.fab-operator.yaml`...
```

**After**:
```
1. **Snapshot** — run `fab operator tick-start` (increments `tick_count`, writes `last_tick_at`, outputs `tick: N` and `now: HH:MM`). Parse stdout for the tick number and current time. Then run `fab pane map` and read `.fab-operator.yaml`.
```

Update the **Idle Message** paragraph (currently says "The time is HH:MM in the operator's local timezone"):

Add the command to use:
```
Between ticks, the operator displays an idle message with the current time and next-tick time:

  Waiting for next tick. Time: 08:26 · next tick: 08:29

Run `fab operator time --interval {interval}` (where `{interval}` is the current loop interval, e.g. `3m`) to get the `now:` and `next:` values to fill in the message.
```

### 6. `src/kit/skills/_cli-fab.md` — fab operator section update

Add documentation for the new subcommands under the existing `fab operator` section:

```markdown
## fab operator subcommands

### fab operator tick-start

Called at the start of each operator tick. Increments `tick_count` by 1, writes `last_tick_at` (ISO 8601 UTC) to `.fab-operator.yaml`, and outputs current time.

```
fab operator tick-start
```

**Output** (stdout):
```
tick: N
now: HH:MM
```

No flags. Side effects: updates `.fab-operator.yaml` (tick_count, last_tick_at).

### fab operator time

Pure time query — no side effects, no file writes.

```
fab operator time [--interval <duration>]
```

| Flag | Type | Description |
|------|------|-------------|
| `--interval` | duration string | If given, also outputs `next: HH:MM` = now + interval |

**Output** without `--interval`:
```
now: HH:MM
```

**Output** with `--interval 3m`:
```
now: HH:MM
next: HH:MM
```

Duration format: Go duration strings (`3m`, `5m`, `2m`). Invalid duration → exit 1.
```

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update the operator section to document `fab operator tick-start` and `fab operator time` subcommands and their use in tick behavior and idle message
- `fab-workflow/kit-architecture`: (modify) Update `fab operator` entry in Command Reference to reflect new subcommands

## Impact

- `src/go/fab/cmd/fab/operator.go` — main change: refactor to parent command, add two subcommands
- `src/go/fab/cmd/fab/operator_test.go` — add tests for both new subcommands
- `src/kit/skills/fab-operator.md` — update tick step 1 and idle message paragraph
- `src/kit/skills/_cli-fab.md` — add `fab operator tick-start` and `fab operator time` entries
- `docs/memory/fab-workflow/execution-skills.md` — update operator documentation
- `docs/memory/fab-workflow/kit-architecture.md` — update command reference

The cobra refactor (leaf → parent with default RunE) is backwards compatible: `fab operator` with no subcommand continues to work identically. No other commands or internal packages are affected.

Dependencies: `gopkg.in/yaml.v3` (already in the Go module), `github.com/spf13/cobra` (already used), Go standard `time` package.

## Open Questions

None — the description is fully specified with exact output formats, flag names, file I/O semantics, and update targets for both skill files.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use Go `time` package for all clock operations (not shell `date`) | Explicitly stated in description; aligns with constitution's portability principle (§V) | S:95 R:95 A:95 D:95 |
| 2 | Certain | `fab operator tick-start` outputs `tick: N\nnow: HH:MM` format | Exact format given in description | S:95 R:90 A:95 D:95 |
| 3 | Certain | `fab operator time --interval` outputs `now: HH:MM\nnext: HH:MM` | Exact format given in description | S:95 R:90 A:95 D:95 |
| 4 | Certain | `--interval` accepts Go duration strings (`3m`, `5m`, etc.) | Explicitly stated | S:95 R:90 A:95 D:95 |
| 5 | Certain | Operator command refactored from leaf to parent, keeping existing `runOperator` as default RunE | Explicitly stated in description; cobra supports parent commands with default RunE | S:95 R:90 A:95 D:95 |
| 6 | Confident | Use `gopkg.in/yaml.v3` for `.fab-operator.yaml` read/write in `tick-start` | Already a transitive dependency in the Go module (used elsewhere in fab-go); consistent with codebase patterns | S:80 R:85 A:80 D:85 |
| 7 | Confident | `tick_count` starts at 0 if `.fab-operator.yaml` is missing or field absent | Standard defensive default; description says "increments by 1" implying starting state is 0 | S:80 R:85 A:80 D:80 |
| 8 | Confident | Tests added to `operator_test.go` alongside existing tests (not a new file) | Constitution: "Changes to the fab CLI MUST include corresponding test updates"; code quality: follow existing project patterns | S:85 R:90 A:90 D:85 |
| 9 | Confident | `last_tick_at` written as RFC3339 UTC string (`time.RFC3339`) | Description says "ISO 8601 UTC timestamp"; Go's `time.RFC3339` is the standard ISO 8601 format | S:85 R:85 A:90 D:85 |
| 10 | Confident | Memory updates scoped to `execution-skills.md` and `kit-architecture.md` (not a new memory file) | Both files already document operator behavior; constitution: "what actually shipped" goes in memory | S:75 R:80 A:80 D:80 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved).
