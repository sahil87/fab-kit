# Spec: Window Prefix Primitives and Done-Marker on Removal

**Change**: 260423-rxu3-window-prefix-primitives
**Created**: 2026-04-23
**Affected memory**: `docs/memory/fab-workflow/pane-commands.md`, `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- **Width / BMP / codepoint validation inside the primitives.** Single-width-BMP remains operator-skill guidance, not a primitive-level contract. Avoids coupling the mechanical verb to tmux tab-rendering rules.
- **A `fab operator window-name …` or top-level `fab window-name …` surface.** The new verbs live under the existing `fab pane` group only.
- **Config surface for the done-marker character.** The marker is a skill constant in `src/kit/skills/fab-operator.md`; no `config.yaml: operator.done_marker` field is added.
- **Storing an `original_name` on the monitored entry.** The `replace-prefix` guard on the current name is sufficient to protect user-renamed windows without schema state.
- **Splitting infra (subcommands) and behavior (operator consumption + replace-on-removal) into two PRs.** All three land in one PR so the primitive ships with two use sites on day one.
- **Add / remove / four-verb set.** The two-verb set (`ensure-prefix`, `replace-prefix`) subsumes add (= ensure with a new char) and remove (= replace with empty `<to>`).

## fab pane: New `window-name` Subcommand Group

### Requirement: Parent-Group Wiring

`fab pane` SHALL register a `window-name` subcommand group alongside the existing `map`, `capture`, `send`, and `process` subcommands. The group SHALL contain exactly two subcommands: `ensure-prefix` and `replace-prefix`. Invoking `fab pane window-name` with no subcommand SHALL print standard cobra help listing both subcommands.

#### Scenario: Help output lists new group
- **GIVEN** the `fab` binary is built with this change
- **WHEN** the user runs `fab pane --help`
- **THEN** the output lists five subcommands: `map`, `capture`, `send`, `process`, `window-name`
- **AND** each is accompanied by its short description

#### Scenario: Group help lists two verbs
- **GIVEN** the `fab` binary is built with this change
- **WHEN** the user runs `fab pane window-name --help`
- **THEN** the output lists exactly two subcommands: `ensure-prefix`, `replace-prefix`

### Requirement: `ensure-prefix` Behavior

`fab pane window-name ensure-prefix <pane> <char>` SHALL read the target pane's current window name via `tmux display-message -p -t <pane> '#W'` and idempotently prepend `<char>` to it. If the name already begins with the literal string `<char>` (prefix check, not regex), the command SHALL be a no-op. Otherwise the command SHALL run `tmux rename-window -t <pane> "<char><current-name>"` and exit 0.

The `<pane>` argument SHALL accept any tmux pane ID (`%3`, `%abc`) in the same format used by existing `fab pane send` / `fab pane capture`. The `<char>` argument SHALL be any non-empty string; the command SHALL NOT restrict `<char>` to single characters or validate width/BMP/codepoint.

#### Scenario: Prepends when prefix is absent
- **GIVEN** a tmux pane `%3` whose window name is `spec-work`
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 »`
- **THEN** tmux is asked to rename window `%3` to `»spec-work`
- **AND** the command exits 0 with stdout `renamed: spec-work -> »spec-work`

#### Scenario: No-op when prefix already present
- **GIVEN** a tmux pane `%3` whose window name is `»spec-work`
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 »`
- **THEN** no `tmux rename-window` call is made
- **AND** the command exits 0 with empty stdout

#### Scenario: Multi-codepoint prefix is accepted
- **GIVEN** a tmux pane `%3` whose window name is `work`
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 '** '`
- **THEN** tmux is asked to rename window `%3` to `** work`
- **AND** the command exits 0 (no width validation)

#### Scenario: Empty `<char>` is rejected
- **GIVEN** a running tmux session
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 ''`
- **THEN** the command exits 3 with `Error: <char> must be non-empty` on stderr
- **AND** no `tmux` subprocess is invoked

### Requirement: `replace-prefix` Behavior

`fab pane window-name replace-prefix <pane> <from> <to>` SHALL perform an atomic guarded prefix swap. The command SHALL read the current window name; if it begins with the literal string `<from>`, it SHALL run `tmux rename-window -t <pane> "<to><name-without-from-prefix>"`. If the current name does NOT begin with `<from>`, the command SHALL be a no-op and SHALL exit 0 without error. The `<to>` argument MAY be empty — an empty `<to>` performs a prefix removal.

The `<from>` argument SHALL be non-empty; supplying an empty `<from>` SHALL be treated as a usage error (exit 3 with a message on stderr). `<to>` has no content restrictions (mirroring `ensure-prefix`).

#### Scenario: Swaps prefix when `<from>` is present
- **GIVEN** a tmux pane `%3` whose window name is `»spec-work`
- **WHEN** the user runs `fab pane window-name replace-prefix %3 » ›`
- **THEN** tmux is asked to rename window `%3` to `›spec-work`
- **AND** the command exits 0 with stdout `renamed: »spec-work -> ›spec-work`

#### Scenario: No-op when `<from>` is absent (user-rename guard)
- **GIVEN** a tmux pane `%3` whose window name is `my-custom-name` (user renamed mid-monitoring)
- **WHEN** the user runs `fab pane window-name replace-prefix %3 » ›`
- **THEN** no `tmux rename-window` call is made
- **AND** the command exits 0 with empty stdout

#### Scenario: Empty `<to>` removes the prefix
- **GIVEN** a tmux pane `%3` whose window name is `»spec-work`
- **WHEN** the user runs `fab pane window-name replace-prefix %3 » ''`
- **THEN** tmux is asked to rename window `%3` to `spec-work`
- **AND** the command exits 0 with stdout `renamed: »spec-work -> spec-work`

#### Scenario: Empty `<from>` is rejected
- **GIVEN** a running tmux session
- **WHEN** the user runs `fab pane window-name replace-prefix %3 '' ›`
- **THEN** the command exits 3 with a usage error on stderr
- **AND** no `tmux rename-window` call is made

### Requirement: Exit-Code Scheme

Both subcommands SHALL use a two-code tmux-error scheme (plus 0 for success):

| Exit | Meaning |
|------|---------|
| 0 | Rename succeeded OR operation was a no-op |
| 2 | Pane does not exist — tmux stderr contains `can't find pane` / `no such pane` / `pane ... not found` (case-insensitive) |
| 3 | Any other tmux error: tmux not running / socket unreachable / rename failed / permission denied / argument usage error (e.g., empty `<char>` or `<from>`) |

The primitives SHALL NOT gate on `$TMUX` — tmux's own exec failure surfaces as exit 3 when tmux is not running or the socket is unreachable, which lets callers invoke the primitives via `--server` targeting from non-tmux processes (e.g., daemons). Exit 2 SHALL propagate tmux's stderr verbatim. Exit 3 SHALL propagate tmux's stderr when tmux supplied it; for argument usage errors (empty `<char>` / `<from>`), stderr is a fab-generated usage message.

#### Scenario: Non-existent pane exits 2
- **GIVEN** a tmux session with no pane `%99`
- **WHEN** the user runs `fab pane window-name ensure-prefix %99 »`
- **THEN** the command exits 2 with stderr containing `pane %99 not found` or tmux's equivalent message

#### Scenario: Other tmux failure exits 3
- **GIVEN** a tmux pane `%3` that exists but `tmux rename-window` fails for another reason
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 »`
- **THEN** the command exits 3 with tmux's stderr propagated

### Requirement: Output Modes

Both subcommands SHALL default to plain-text output and SHALL accept a `--json` flag for structured output. The plain-text format SHALL be `renamed: <old> -> <new>` on a successful rename and empty stdout on a no-op. The JSON format SHALL emit a single object with the shape:

```json
{"pane": "%3", "old": "spec-work", "new": "»spec-work", "action": "renamed"}
```

…and `{"pane": "%3", "old": "»spec-work", "new": "»spec-work", "action": "noop"}` for a no-op. `--json` SHALL be the only non-positional flag on these subcommands (no `--server` collision beyond the inherited persistent flag).

#### Scenario: Default plain output on rename
- **GIVEN** a pane `%3` named `work`
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 »`
- **THEN** stdout is exactly `renamed: work -> »work\n`

#### Scenario: `--json` output on no-op
- **GIVEN** a pane `%3` named `»work`
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 » --json`
- **THEN** stdout is a single JSON object `{"pane":"%3","old":"»work","new":"»work","action":"noop"}`
- **AND** exit is 0

### Requirement: Inherited `--server` / `-L` Flag

Both subcommands SHALL honor the persistent `--server <name>` / `-L <name>` flag registered on `paneCmd`. When non-empty, every `tmux` invocation inside the subcommand SHALL be prepended with `-L <name>` via the existing `internal/pane.WithServer` helper. This matches the behavior already implemented for `map`, `capture`, `send`, `process`.

#### Scenario: `--server` flag scopes tmux calls
- **GIVEN** two tmux servers running (default and `runKit`)
- **AND** pane `%3` on the `runKit` server is named `work`
- **WHEN** the user runs `fab pane window-name ensure-prefix %3 » --server runKit`
- **THEN** every internal `tmux` call runs with `-L runKit`
- **AND** pane `%3` on the `runKit` server is renamed to `»work`

### Requirement: Source Layout

The two subcommands SHALL be implemented in a single new file `src/go/fab/cmd/fab/pane_window_name.go` exposing a `paneWindowNameCmd()` cobra group that `paneCmd()` wires into `src/go/fab/cmd/fab/pane.go` alongside the existing four. Shared tmux read-and-rename logic (current-name read, prefix check, rename) MAY be factored into a helper in `src/go/fab/internal/pane/` if the refactor keeps net lines down; otherwise it SHALL live in the new command file. A test file `pane_window_name_test.go` SHALL accompany the command file, following the argv-capture test pattern used by `pane_capture_test.go` and `pane_send_test.go`.

## /fab-operator Skill: Site Updates

### Requirement: Enrollment Uses `ensure-prefix`

`src/kit/skills/fab-operator.md` §4 ("Monitored Set") SHALL replace the current inline three-line shell block

```sh
name=$(tmux display-message -p -t <pane> '#W')
case "$name" in »*) ;; *) tmux rename-window -t <pane> "»${name}" ;; esac
```

…with a single call to the new primitive:

```sh
fab pane window-name ensure-prefix <pane> »
```

The behavior on non-zero exit SHALL be preserved: log `"{change}: window rename skipped ({error})."` and continue (enrollment itself is already durable from the preceding `.fab-operator.yaml` write). The §6 step 4 parenthetical that references the inline-shell approach SHALL be updated to reference the `ensure-prefix` primitive.

#### Scenario: Enrollment on an unprefixed window
- **GIVEN** the operator has just written a monitored entry for change `r3m7` in pane `%3`
- **AND** `%3`'s window name is `work`
- **WHEN** the operator runs the enrollment rename step
- **THEN** the operator invokes `fab pane window-name ensure-prefix %3 »`
- **AND** the pane's window name becomes `»work`
- **AND** the monitored entry is retained regardless of the primitive's exit code

#### Scenario: Enrollment on an already-prefixed window
- **GIVEN** the operator has just written a monitored entry for change `r3m7` in pane `%3`
- **AND** `%3`'s window name is `»work` (spawned-window case)
- **WHEN** the operator runs the enrollment rename step
- **THEN** the operator invokes `fab pane window-name ensure-prefix %3 »`
- **AND** the primitive no-ops
- **AND** the pane's window name remains `»work`

### Requirement: Removal Swaps to Done-Marker

`src/kit/skills/fab-operator.md` §4 ("Monitored Set", Removal paragraph) SHALL REMOVE the current rule "The window name is **not** restored on removal — the `»` prefix persists" and REPLACE it with the following rule:

> On every removal path (change reaches its stop stage or a terminal stage if `stop_stage` is null, pane dies, user explicitly stops), the operator SHALL invoke
>
> ```sh
> fab pane window-name replace-prefix <pane> » ›
> ```
>
> …to swap the active-monitoring `»` prefix for the done-marker `›`. The swap leaves user-renamed windows untouched (the `replace-prefix` guard) and is a safe no-op when the pane has already died (exit 2 is logged and treated as successful removal).

The done-marker `›` (U+203A, SINGLE RIGHT-POINTING ANGLE QUOTATION MARK) SHALL be hardcoded as a literal in the skill file; no `config.yaml` field is introduced.

#### Scenario: Removal on an actively-prefixed window
- **GIVEN** a monitored change `r3m7` in pane `%3` reaches a terminal stage
- **AND** `%3`'s window name is `»spec-work`
- **WHEN** the operator runs the removal step for `r3m7`
- **THEN** the operator invokes `fab pane window-name replace-prefix %3 » ›`
- **AND** the pane's window name becomes `›spec-work`
- **AND** the monitored entry is removed

#### Scenario: Removal on a user-renamed window
- **GIVEN** a monitored change `r3m7` in pane `%3` reaches a terminal stage
- **AND** the user has renamed the window to `my-custom-name` (no longer starts with `»`)
- **WHEN** the operator runs the removal step for `r3m7`
- **THEN** the operator invokes `fab pane window-name replace-prefix %3 » ›`
- **AND** the primitive no-ops (guard fails)
- **AND** the pane's window name remains `my-custom-name`
- **AND** the monitored entry is removed

#### Scenario: Removal on a dead pane
- **GIVEN** a monitored change `r3m7` in pane `%3`
- **AND** pane `%3` no longer exists
- **WHEN** the operator runs the removal step for `r3m7`
- **THEN** the operator invokes `fab pane window-name replace-prefix %3 » ›`
- **AND** the primitive exits 2 (pane not found)
- **AND** the operator treats this as a successful removal
- **AND** the monitored entry is removed

### Requirement: Removal Paths Are Uniform

The removal swap SHALL apply to every removal trigger uniformly. The same `replace-prefix » ›` call SHALL be issued for: (a) change reaching its `stop_stage`, (b) change reaching a terminal stage when `stop_stage` is null, (c) pane death, (d) user-requested stop. The operator SHALL NOT branch on removal cause when invoking the swap.

## Done-Marker Character

### Requirement: `›` (U+203A)

The done-marker character SHALL be `›` (U+203A, SINGLE RIGHT-POINTING ANGLE QUOTATION MARK). The character is hardcoded as a literal in `src/kit/skills/fab-operator.md` at the removal site only. The enrollment-site prefix remains `»` (U+00BB). Together they preserve the guillemet visual family (`»` "active" → `›` "trail preserved").

The character selection rationale is:

- Single-width BMP codepoint (per the 260328 and 260416 decisions) — tmux tab rendering is consistent.
- Direct parity with `»` — `›` is the single-guillemet counterpart of the double-guillemet.
- Avoids visual collision with `✓`, which already appears in the operator status frame as the stage-done signal (`● apply → review ✓`). Reusing `✓` on window names would create a light semantic overlap.

## Spec File Sync

### Requirement: SPEC-fab-operator.md Update

`docs/specs/skills/SPEC-fab-operator.md` Section Structure item 4 ("Monitoring System") SHALL update the one-line window-name bullet to reflect the new behavior. The current bullet says:

> Window-name rename on enrollment: prefix `»` to the tmux window name (idempotent — skipped if already prefixed). Removal does not restore the original name.

It SHALL be replaced with:

> Window-name rename on enrollment: prefix `»` to the tmux window name via `fab pane window-name ensure-prefix` (idempotent). Removal replaces `»` with `›` via `fab pane window-name replace-prefix`, guarded to skip user-renamed windows.

No other changes to SPEC-fab-operator.md are required.

## Hydrate: Memory Updates

### Requirement: pane-commands.md Gains `window-name` Section

`docs/memory/fab-workflow/pane-commands.md` SHALL add a new section `### Subcommand: fab pane window-name` placed after `### Subcommand: fab pane process` and before `### --server / -L Flag`. The section SHALL mirror the structure used by the existing four subcommand sections:

- Command form line (`fab pane window-name <verb> [flags]`)
- Source path line
- Verb-specific behavior (one paragraph per verb)
- Exit codes table or prose
- Output formats (plain + `--json`)

The parent-command overview paragraph SHALL be updated from "four tmux-pane operations" to "five tmux-pane operations" and from `map, capture, send, process` to `map, capture, send, process, window-name` (preserving the existing ordering convention — new entries appended at the end).

The Parent Command `fab pane` Requirement paragraph SHALL be updated to mention five subcommands.

The Shared Pane Package section SHALL gain an entry for any new helper introduced in `internal/pane/` (if one is factored); if no helper is factored, no change to that section is required.

A new changelog entry SHALL be appended for 260423-rxu3.

### Requirement: execution-skills.md §4 Revised

`docs/memory/fab-workflow/execution-skills.md` §4 ("Monitored Set") paragraphs describing enrollment and removal SHALL be rewritten to describe the new primitive-backed approach:

- The enrollment paragraph SHALL reference `fab pane window-name ensure-prefix <pane> »` instead of the inline three-line shell.
- The removal paragraph SHALL describe the replace-prefix swap to `›` as the new behavior, replacing the previous "window name is not restored" rule.

The "`»` Prefix Extends to Enrolled Windows" Design Decision entry SHALL be revised (or a successor entry added) to capture: (a) the primitive extraction rationale, (b) the replace-on-removal reversal of the prior "no restore" rule, (c) the done-marker character `›` choice with BMP/collision-avoidance reasoning. The "Rejected" bullet listing "Restoring the original name on removal" SHALL be updated — the new behavior does not restore the original name; it substitutes a done-marker while the user-rename-mid-monitoring guard is now enforced by `replace-prefix`'s literal prefix check.

A new changelog entry SHALL be appended for 260423-rxu3.

### Requirement: Index Date Bumps

`docs/memory/fab-workflow/index.md` SHALL date-bump both `execution-skills` and `pane-commands` rows to `2026-04-23`.

## Deprecated Requirements

### "Removal Does Not Restore Original Name"

**Reason**: Leaving `»` on removed windows made the tab bar lie about what the operator is currently tracking — the entire point of the prefix is at-a-glance coordination.
**Migration**: Removal now replaces `»` with `›` (done-marker) via `replace-prefix`. The user-rename guard is enforced by the primitive's literal prefix check rather than by a "no restore" policy.

## Design Decisions

1. **Two-verb primitive set (`ensure-prefix`, `replace-prefix`)**
   - *Why*: `ensure-prefix` collapses idempotent add; `replace-prefix` with empty `<to>` collapses remove. Add without idempotency is almost never useful in practice (double-prefixing), so making it idempotent by default removes a footgun.
   - *Rejected*: Four-verb set (`add`, `remove`, `replace`, `ensure`) — redundant; adds surface without covering new cases.

2. **Guard on current prefix instead of stored `original_name`**
   - *Why*: A literal prefix check is stateless and inherently tolerant of user-rename-mid-monitoring (the guard simply doesn't match). Storing `original_name` introduces schema state and an ambiguity — "which original is authoritative if the user renamed at t+5m?" — with no user-requested benefit.
   - *Rejected*: Store `original_name` on monitored entries; restore on removal. Rejected in both 260422-jyyg (for a different reason — schema churn) and here.

3. **Done-marker `›` (U+203A), not `✓`**
   - *Why*: `›` preserves the guillemet family (`»` → `›`) with a visually-lighter "reduced" reading. `✓` already signals stage-done in the operator status frame and reusing it on window names would create a semantic collision.
   - *Rejected*: `✓` (U+2713) — strong "done" semantics but collides with status-frame usage; `·` (U+00B7) — too quiet to read as "completed-but-trail-preserved".

4. **Done-marker as skill constant, not config option**
   - *Why*: No current demand for per-project customization. A config surface (`operator.done_marker`) would add default-handling logic and docs cost for zero current benefit. Re-open only if demand surfaces.
   - *Rejected*: Config option with skill-level default — middle ground, still speculative.

5. **No width/BMP/codepoint validation in the primitives**
   - *Why*: The primitive is a mechanical shell-replacement tool. Coupling it to tmux tab-rendering rules (single-width BMP) would make it an opinionated style enforcer with a responsibility it doesn't own. The single-width requirement lives in the operator-skill guidance that dictates which characters to pass — that's where the taste belongs.
   - *Rejected*: Validate by default; `--strict` opt-in flag.

6. **Distinct exit codes 1 / 2 / 3**
   - *Why*: The operator's removal path wants to treat "pane gone" as a successful removal (the window is gone anyway) and "pane alive but rename failed" as a warning. Distinct codes let the caller discriminate without parsing stderr.
   - *Rejected*: Generic non-zero on any failure — simpler but forces callers to parse stderr.

7. **`--json` output alongside plain text**
   - *Why*: Consistent with the other `fab pane` subcommands. Small extra surface now; removes the "I need it later" cost of adding it after release.
   - *Rejected*: Plain-only with deferred `--json` — minor but repeatedly paying a consistency cost.

8. **Single PR for infra + behavior**
   - *Why*: Shipping the primitive with zero users would fail the "build the minimum" test. Bundling gives the primitive two call sites on day one (enrollment + removal), which validates the abstraction.
   - *Rejected*: Two PRs (infra-then-behavior) — doubles review surface and leaves the primitive momentarily unused.

## Assumptions

<!-- SCORING SOURCE: fab score reads only this table. Carries forward intake assumptions; spec-stage analysis has upgraded tentatives to certain via /fab-clarify and confirmed/refined the remaining set. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope covers three deliverables in one PR: `fab pane window-name` subcommands, operator skill consumption at two sites, replace-on-removal behavior | Confirmed from intake #1. User explicitly scoped "one PR that (a) adds the subcommands, (b) updates /fab-operator to call them, (c) introduces the replace-on-removal rule" | S:95 R:85 A:90 D:90 |
| 2 | Certain | Minimum verb set is two: `ensure-prefix` and `replace-prefix`; add/remove subsumed | Confirmed from intake #2. Discussed explicitly; add-without-idempotency is a footgun, remove = replace with empty `<to>` | S:95 R:85 A:90 D:90 |
| 3 | Certain | Subcommands live under `fab pane`, not `fab operator` or top-level | Confirmed from intake #3. Consistent with existing `fab pane {map, capture, send, process}` group | S:95 R:90 A:95 D:95 |
| 4 | Certain | Both verbs use literal string-prefix check, not regex | Confirmed from intake #4. Consistent with 260422-jyyg Design Decision 2 (literal `»` guard) | S:95 R:85 A:90 D:95 |
| 5 | Certain | `<pane>` argument format matches existing `fab pane send` / `capture` (tmux pane ID like `%3`) | Confirmed from intake #5 | S:95 R:95 A:100 D:95 |
| 6 | Certain | Replace-on-removal wholesale replaces the prior "no rename on removal" rule | Confirmed from intake #6. The new rule is strictly more informative — honest tab-bar signal | S:95 R:80 A:90 D:90 |
| 7 | Certain | Guard-on-current-prefix protects user-rename-mid-monitoring; no `original_name` schema field | Confirmed from intake #7. Stateless; avoids authoritative-original ambiguity | S:95 R:80 A:85 D:90 |
| 8 | Certain | Done-marker character is `›` (U+203A, single guillemet) | Upgraded from intake Tentative #17 via /fab-clarify Q1. Single-width BMP; guillemet-family parity; avoids `✓` collision with operator status frame | S:95 R:75 A:70 D:50 |
| 9 | Certain | Done-marker stored as skill constant in `src/kit/skills/fab-operator.md`, not a config option | Upgraded from intake Tentative #13 via /fab-clarify Q2. No current demand for per-project customization | S:95 R:70 A:70 D:60 |
| 10 | Certain | Distinct exit codes 1 / 2 / 3 (no tmux / pane missing / other tmux error) | Upgraded from intake Tentative #14 via /fab-clarify Q3. Operator removal path discriminates "pane gone" (successful removal) from other failures | S:95 R:75 A:75 D:65 |
| 11 | Certain | Primitives do NOT validate width/BMP/codepoint of `<char>`, `<from>`, `<to>` | Upgraded from intake Tentative #15 via /fab-clarify Q4. Mechanical verb; single-width is operator-skill guidance | S:95 R:80 A:70 D:60 |
| 12 | Certain | Primitives expose `--json` alongside plain default | Upgraded from intake Tentative #16 via /fab-clarify Q5. Consistent with existing `fab pane` subcommands | S:95 R:85 A:75 D:70 |
| 13 | Certain | Single-width BMP family maintained (per 260328/260416 decisions); both `»` and `›` fit | Upgraded from intake Confident #8 via bulk confirm. No SMP, no double-width | S:95 R:85 A:85 D:75 |
| 14 | Certain | All changes land in one PR, not split infra-then-behavior | Upgraded from intake Confident #9 via bulk confirm. Splitting loses validation — primitive ships with zero users | S:95 R:80 A:85 D:85 |
| 15 | Certain | No `original_name` or equivalent state on monitored entries | Upgraded from intake Confident #10 via bulk confirm. Implied by #7; no edge case forces schema addition | S:95 R:80 A:85 D:85 |
| 16 | Certain | Enrollment call shape is `fab pane window-name ensure-prefix <pane> »` (literal `»`, pane ID first) | Upgraded from intake Confident #11 via bulk confirm. Matches existing `fab pane` ergonomics | S:95 R:90 A:85 D:80 |
| 17 | Certain | Removal swap applies uniformly to every removal trigger (terminal stage, stop_stage, pane death, explicit stop) | Upgraded from intake Confident #12 via bulk confirm. 260422-jyyg established a single Removal-triggers set; this change does not split it | S:95 R:85 A:85 D:85 |
| 18 | Confident | `paneWindowNameCmd()` is a cobra command group with two subcommand children (`ensure-prefix`, `replace-prefix`), wired into `paneCmd()` alongside the existing four | Matches the existing pattern in `pane.go`: `cmd.AddCommand(paneMapCmd(), paneCaptureCmd(), paneSendCmd(), paneProcessCmd())`. Spec-level decision | S:80 R:90 A:90 D:85 |
| 19 | Confident | JSON output object keys are `pane`, `old`, `new`, `action` — lowercase, `action` value is `"renamed"` or `"noop"` | No existing `fab pane` subcommand emits this exact shape; following snake_case key convention used elsewhere (`panemap.go`). Keys chosen for minimal ambiguity | S:75 R:85 A:80 D:80 |
| 20 | Confident | Usage error (e.g., empty `<from>`) exits 3 rather than a separate code | Exit 3 is already "other tmux error" which covers argument-level failures too. Avoids inventing a fourth code for a rare case | S:70 R:85 A:80 D:75 |
| 21 | Confident | Shared read-and-rename helper is factored into `internal/pane/` only if net lines decrease; otherwise kept in the command file | Mirrors the 260417 decision on `WithServer` helper scope — factor when there is a second caller, otherwise keep local | S:75 R:90 A:80 D:85 |
| 22 | Confident | Test file `pane_window_name_test.go` follows the argv-capture pattern used by `pane_capture_test.go` and `pane_send_test.go` rather than spinning up real tmux | Existing pattern — tmux subprocesses are not exercised in unit tests; argv builders are extracted and tested as pure functions | S:80 R:85 A:85 D:80 |

22 assumptions (17 certain, 5 confident, 0 tentative, 0 unresolved).
