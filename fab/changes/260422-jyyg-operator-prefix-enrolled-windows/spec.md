# Spec: Operator prefix on window enrollment

**Change**: 260422-jyyg-operator-prefix-enrolled-windows
**Created**: 2026-04-22
**Affected memory**: `docs/memory/fab-workflow/execution-skills.md`

## Non-Goals

- Restoring the original window name when a change leaves the monitored set — the user explicitly wants the prefix to persist after removal.
- Renaming panes, sessions, or tmux status-line titles — only the *window* name (`#W`) is affected.
- Retroactively renaming windows that were monitored in prior operator sessions — the rename is triggered by enrollment events in the current session, not by reading back historical state (though `/clear` recovery re-enrollment is covered via the idempotent guard).
- Rewriting the stale `fab-operator4` content in `docs/specs/skills/SPEC-fab-operator.md`. The spec update in this change is a **single additive bullet**; broader spec cleanup is out of scope.

## fab-operator: Window Rename on Enrollment

### Requirement: Rename on enrollment

The operator SHALL rename the target tmux window to prefix `»` to its current name whenever it enrolls a change into the monitored set. The rename applies to all enrollment paths listed in `src/kit/skills/fab-operator.md` §4 (Enrollment bullet) — direct command dispatch, user-requested monitoring, autopilot spawns, and watch spawns.

The rename SHALL be implemented as:

```sh
name=$(tmux display-message -p -t <pane> '#W')
case "$name" in »*) ;; *) tmux rename-window -t <pane> "»${name}" ;; esac
```

`<pane>` is the pane ID already persisted on the monitored entry (e.g. `%3`). The operator SHALL NOT introduce a separate window-ID field — tmux resolves the pane to its containing window automatically when `-t <pane-id>` is used with `rename-window`.

The rename SHALL occur **after** the monitored entry has been written to `.fab-operator.yaml`, so enrollment state is durable even if the rename step fails.

#### Scenario: New enrollment, plain name

- **GIVEN** a tmux window named `dev-tab` containing pane `%4`, and no monitored entry for that change
- **WHEN** the operator enrolls the change (e.g. after sending `/fab-continue` to that pane, or receiving "watch `%4`")
- **THEN** `.fab-operator.yaml` SHALL contain the new monitored entry
- **AND** the window SHALL be renamed to `»dev-tab`

#### Scenario: Enrollment of an operator-spawned window

- **GIVEN** a tmux window created by the operator's spawn sequence with name `»swift-fox` (already prefixed per `src/kit/skills/fab-operator.md:303`)
- **WHEN** the post-spawn enrollment step runs
- **THEN** the name-prefix check SHALL find `»` already present
- **AND** no `tmux rename-window` invocation SHALL occur
- **AND** the window name SHALL remain `»swift-fox` exactly once (no `»»swift-fox`)

#### Scenario: Enrollment via autopilot spawn

- **GIVEN** autopilot is about to spawn change `cd34` with `depends_on: [ab12]`
- **WHEN** the spawn sequence reaches the enrollment step (step 4 of `src/kit/skills/fab-operator.md` §6 Spawning an Agent)
- **THEN** the enrollment SHALL apply the rename per this requirement
- **AND** no autopilot-specific branching SHALL be required (single enrollment path)

### Requirement: Idempotent prefix guard

The operator SHALL treat the rename as a no-op when the current window name already begins with `»`. The guard SHALL be a string-prefix check against the literal character `»` (U+00BB), not a broader Unicode-category test or regex over arbitrary marker characters.

This guard makes the enrollment step safe to invoke in the following situations without producing `»»…` names:

- Operator-spawned windows (already carry the prefix from `tmux new-window -n "»<wt>"`)
- Re-enrollment of the same change within a session (e.g., after a transient removal)
- `/clear` recovery, where the monitored set is re-read from `.fab-operator.yaml` and re-applied to the live pane map

#### Scenario: /clear recovery over prefixed windows

- **GIVEN** `.fab-operator.yaml` lists three monitored entries, all of whose current window names start with `»`
- **WHEN** the operator restarts via `/clear` and restores the monitored set per §2 Init
- **THEN** for each restored entry the prefix guard SHALL match
- **AND** zero `tmux rename-window` invocations SHALL occur
- **AND** no window name SHALL change

#### Scenario: Re-enrollment after transient removal

- **GIVEN** change `r3m7` was enrolled, then removed (e.g., pane death false-positive), and is now being re-enrolled, with window name currently `»auth-flow`
- **WHEN** the enrollment step runs
- **THEN** the prefix guard SHALL match
- **AND** the window name SHALL remain `»auth-flow`

### Requirement: No rename on removal

The operator MUST NOT rename, rewrite, or restore the target window's name when a change leaves the monitored set, regardless of removal cause. This applies to every removal trigger listed in `src/kit/skills/fab-operator.md` §4 (Removal bullet): reaching `stop_stage`, reaching a terminal stage (hydrate / ship / review-pr when `stop_stage` is null), pane death, or explicit user stop.

The operator SHALL NOT persist the pre-rename window name in `.fab-operator.yaml` or any other state, since it is never consumed.

#### Scenario: Change reaches terminal stage

- **GIVEN** change `ab12` is enrolled with window name `»add-oauth`, and its stage advances to `hydrate` (terminal when `stop_stage` is null)
- **WHEN** the monitoring tick removes `ab12` from the monitored set
- **THEN** the removal SHALL succeed
- **AND** no `tmux rename-window` SHALL be invoked for that window
- **AND** the window name SHALL remain `»add-oauth` until the user changes it manually

#### Scenario: Pane death

- **GIVEN** change `ef56` is enrolled with pane `%7` and window name `»ef56-tab`, and pane `%7` no longer appears in `fab pane map`
- **WHEN** the monitoring tick's pane-death detection removes `ef56`
- **THEN** no `tmux rename-window` SHALL be invoked (the window is already gone in this case, but the explicit guarantee is: the operator does not attempt to rename on removal)

#### Scenario: Explicit user stop

- **GIVEN** change `k8ds` is enrolled with window name `»k8ds-tab`, and the user says "stop monitoring k8ds"
- **WHEN** the operator processes the removal
- **THEN** the window name SHALL remain `»k8ds-tab`
- **AND** the operator's report SHALL NOT mention a rename

### Requirement: Rename failure resilience

A failure of the `tmux rename-window` invocation during enrollment (for example, the pane vanished between the pane-map refresh and the rename call) SHALL NOT abort or roll back the enrollment. The monitored entry already written to `.fab-operator.yaml` is the source of truth.

On rename failure, the operator SHALL log a single line of the form:

```
{change}: window rename skipped ({error}).
```

…and continue with the rest of the tick. The bounded-retry table in `src/kit/skills/fab-operator.md` §3 is unchanged — the rename is cosmetic, not a retriable action.

#### Scenario: Pane vanishes between refresh and rename

- **GIVEN** change `jyyg` is being enrolled with pane `%9`, and `%9` is killed after the pane-map refresh but before the `tmux rename-window` call
- **WHEN** enrollment runs
- **THEN** the monitored entry for `jyyg` SHALL be written to `.fab-operator.yaml`
- **AND** the `tmux rename-window` invocation SHALL return non-zero
- **AND** the operator SHALL emit the skipped-rename log line and continue
- **AND** the tick's subsequent pane-death detection (step 4) SHALL remove `jyyg` from the monitored set

### Requirement: Spawn path compatibility

The spawn path in `src/kit/skills/fab-operator.md` §6 (step 3 of "Spawning an Agent") — `tmux new-window -n "»<wt>" …` — SHALL remain unchanged. Deduplication between spawn naming and enrollment rename SHALL be handled exclusively by the idempotent prefix guard (see "Idempotent prefix guard" above). The spawn sequence SHALL NOT acquire an explicit "skip rename because already prefixed" branch.

#### Scenario: Spawn-then-enroll sequence

- **GIVEN** the operator is executing the spawn sequence for a new change, step 3 creates window `»swift-fox`, step 4 enrolls
- **WHEN** step 4's enrollment applies the rename requirement
- **THEN** the prefix guard SHALL match `»swift-fox`
- **AND** no duplicate rename or corruption SHALL occur

## Documentation: Skill and spec file sync

### Requirement: Skill file documents the new behavior

The skill file `src/kit/skills/fab-operator.md` SHALL be updated so the enrollment rename is discoverable from §4 (Monitored Set) and cross-referenced from §6 (Spawning an Agent). The updates SHALL include:

1. In §4's **Enrollment** bullet: a sentence describing the rename, the idempotent guard, and the exact `tmux rename-window -t <pane> "»<current-name>"` form.
2. In §4's **Removal** bullet: an explicit sentence stating that the window name is not restored on removal.
3. In §6's "Spawning an Agent" step 4 (enrollment): a parenthetical or cross-reference noting that enrollment applies the window-rename behavior defined in §4, so spawn and non-spawn enrollment share a single rename step.

The updates SHALL be additive only — no unrelated restructuring of §4 or §6 in this change.

#### Scenario: Reader follows the skill top-to-bottom

- **GIVEN** a reader encounters §4's Enrollment bullet
- **WHEN** they read through to §6's Spawning an Agent
- **THEN** they SHALL be able to answer "why does my manually-watched window also get `»`?" without consulting the spec or memory
- **AND** they SHALL find the exact tmux command used, inline in §4

### Requirement: Spec file gets an additive bullet

The spec file `docs/specs/skills/SPEC-fab-operator.md` SHALL be updated with a single additive bullet under the **Monitoring System** section (item 4 of the "Section Structure" list) noting the window-rename-on-enrollment convention. The bullet SHALL reference the `»` convention and the idempotent guard but SHALL NOT duplicate the full requirement text from this spec.

The update SHALL NOT rewrite, reorganize, or remove existing content — including the stale `fab-operator4` references, which are out of scope for this change.

#### Scenario: Spec reader scans the Monitoring System section

- **GIVEN** a reader opens `docs/specs/skills/SPEC-fab-operator.md` and scans section 4 (Monitoring System)
- **WHEN** they look for enrollment-related behaviors
- **THEN** the new additive bullet SHALL mention the window rename
- **AND** the pre-existing bullet list describing enrollment/removal triggers, `/loop` lifecycle, and monitoring-tick steps SHALL remain untouched in ordering and wording

## Design Decisions

1. **Rename target is the window name, not the pane title.** Tmux exposes three distinct labels: session (`#S`), window (`#W`), and pane (`#T`). The existing `»` convention is applied via `tmux new-window -n`, which sets `#W`. Using `#W` keeps visual parity between spawned and enrolled windows in the tab bar.
   - *Why*: Tab-bar symmetry is the sole user-visible outcome. Pane titles are not rendered in the default tmux status format.
   - *Rejected*: Using `tmux select-pane -T` (pane title) — invisible in default status line; breaks parity with spawn path.

2. **Guard on literal `»` prefix rather than a generic "already-marked" check.** We chose a narrow string comparison over a regex that would match alternative markers (e.g., `⚡`, arbitrary emoji).
   - *Why*: The `»` convention is explicitly standardized (per memory entry at `docs/memory/fab-workflow/execution-skills.md:563`). Anything else in the first byte is user content the operator MUST NOT pretend to own.
   - *Rejected*: Prefix regex like `^[»⚡]` — would silently absorb legacy or user-chosen markers, violating the user's naming sovereignty.

3. **Rename after writing `.fab-operator.yaml`, not before.** Enrollment state is the source of truth; the rename is cosmetic.
   - *Why*: If the ordering were reversed and the YAML write failed after a successful rename, the user would see a `»`-prefixed window that the operator does not track — a worse failure mode than the converse (entry exists, name unchanged).
   - *Rejected*: Rename-first ordering — produces a confusing orphaned prefix on partial-failure paths.

4. **No restoration on removal.** User explicitly requested this; the design decision is to encode it as a hard prohibition rather than a default-off option.
   - *Why*: Restoring the prior name requires storing it somewhere and maintaining that storage through `/clear` recovery. The user's mental model is "once operator-touched, stays marked until I clean it up." Matching that model removes state.
   - *Rejected*: Storing `original_name` on the monitored entry and restoring on removal — adds state and edge cases (what if the user renamed the window mid-monitoring?) for no user-requested benefit.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename target is the window name via `#W`, matching line 303's `-n` surface | Confirmed from intake #1 — tmux convention unchanged; `tmux new-window -n` and `tmux rename-window -t <pane>` both address `#W` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Skipped when name already starts with literal `»` (idempotent) | Confirmed from intake #2 — prefix check is string-level, not regex | S:95 R:90 A:95 D:95 |
| 3 | Certain | Removal leaves the name as-is — no restoration path | Confirmed from intake #3 — user explicit | S:100 R:90 A:100 D:100 |
| 4 | Certain | Canonical skill path is `src/kit/skills/fab-operator.md` | Confirmed from intake #4 — verified via filesystem | S:100 R:95 A:100 D:100 |
| 5 | Certain | Spec file updated alongside the skill per constitution | Confirmed from intake #5 — constitution "Additional Constraints" mandates | S:100 R:90 A:100 D:100 |
| 6 | Certain | Rename uses `tmux rename-window -t <pane> "»<name>"` | Clarified from intake #6 — user confirmed | S:95 R:85 A:90 D:80 |
| 7 | Certain | Applies to all enrollment paths (direct, user-request, autopilot, watch) | Clarified from intake #7 — user confirmed. Single enrollment step covers all paths | S:95 R:85 A:85 D:85 |
| 8 | Certain | Rename failure logs and continues — does not abort enrollment | Clarified from intake #8 — user confirmed | S:95 R:80 A:85 D:80 |
| 9 | Certain | Spec file gets an additive bullet, not a rewrite | Clarified from intake #9 — user confirmed. Stale `fab-operator4` content out of scope | S:95 R:90 A:90 D:85 |
| 10 | Certain | Memory update lands in `docs/memory/fab-workflow/execution-skills.md` | Clarified from intake #10 — user confirmed. Also verified via grep: this file currently documents §4 enrollment triggers (line 260) and the `»` convention (line 563) | S:100 R:85 A:95 D:95 |
| 11 | Certain | Rename timing: after `.fab-operator.yaml` write, before tick continues | New (spec stage) — state durability argument (see Design Decision 3). Reversing ordering creates orphaned-prefix failure mode | S:90 R:80 A:85 D:85 |
| 12 | Certain | Skipped-rename log format: `{change}: window rename skipped ({error}).` | New (spec stage) — follows operator's existing one-line log pattern (§4 "Actions print as plain lines below the frame") | S:80 R:90 A:85 D:80 |
| 13 | Certain | Scope of rename is window name only, not pane title or session name | New (spec stage) — Design Decision 1. Pane/session titles are invisible in the default status line, so touching them would violate the "visual parity with tab bar" goal | S:85 R:90 A:95 D:90 |
| 14 | Confident | No opt-out flag or configuration toggle | New (spec stage) — the intake described a universal rule; introducing a toggle would create two code paths with no articulated use case. Reversible if a user demands it later | S:65 R:80 A:80 D:80 |

14 assumptions (13 certain, 1 confident, 0 tentative, 0 unresolved).
