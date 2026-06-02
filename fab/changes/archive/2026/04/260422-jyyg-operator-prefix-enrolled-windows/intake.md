# Intake: Operator prefix on window enrollment

**Change**: 260422-jyyg-operator-prefix-enrolled-windows
**Created**: 2026-04-22
**Status**: Draft

## Origin

Invoked via `/fab-new` with the following request:

> When the operator enrolls an existing tmux window into its monitored set, rename the window to prepend `»` to its name (matching the convention used for operator-spawned windows like `»worktree-name`). Only applies on enrollment; on removal, leave the name as-is. Update `src/kit/skills/fab-operator/SKILL.md` and per constitution also update the corresponding `docs/specs/skills/SPEC-fab-operator.md`.

One-shot invocation — no prior conversation. The canonical skill file is `src/kit/skills/fab-operator.md` (there is no `fab-operator/SKILL.md` in the repo). The spec file `docs/specs/skills/SPEC-fab-operator.md` exists and will be updated alongside the skill, as required by the constitution (additional constraint: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file").

## Why

**Problem.** The `»` prefix on tmux window names is the visual signal that a window is operator-monitored. Today only operator-spawned windows carry it — they are created with `tmux new-window -n "»<wt>" …` (see `src/kit/skills/fab-operator.md:303`). Windows the user created manually and then handed to the operator (via "watch this pane", automatic enrollment after a command is sent to an existing pane, or any other enrollment path in §4 of the skill) keep their original names. The operator's monitored set therefore contains two visually indistinguishable populations: prefixed (spawned) and unprefixed (enrolled). The user has to consult `fab pane map` or the operator frame to know which windows are under observation.

**Consequence if unfixed.** The `»` convention is load-bearing for at-a-glance coordination in multi-agent workflows but is only half-enforced. Users who rely on the prefix as ground truth ("`»` means the operator has this one") get a false negative on every manually-enrolled window.

**Why this approach.** Extending the existing convention is cheaper and more discoverable than inventing a second signal (e.g., a status-bar marker). `tmux rename-window` is a single, reversible shell call — no new state, no new tool dependency. Restricting the rename to enrollment (not removal) matches the user's stated preference and preserves user intent for window names they chose themselves: once a window leaves the monitored set, its original name may no longer be accurate, so silently restoring it would be presumptuous.

## What Changes

### 1. Rename on enrollment (new behavior)

When the operator enrolls a change into its monitored set (§4 of `src/kit/skills/fab-operator.md`), it MUST ensure the target tmux window's name is prefixed with `»`.

**Algorithm** (executed immediately after writing the monitored entry to `.fab-operator.yaml`):

1. Read the current window name:
   ```sh
   name=$(tmux display-message -p -t <pane> '#W')
   ```
   `<pane>` is the pane ID already stored on the monitored entry (e.g., `%3`). `#W` yields the window name for the window that contains the pane.

2. If `name` already starts with `»`, skip — idempotent no-op. This handles:
   - Operator-spawned windows (already prefixed at creation time per line 303)
   - Re-enrollment of a window previously enrolled in this session
   - `/clear` recovery where the monitored set is restored from `.fab-operator.yaml` and every window already carries the prefix

3. Otherwise, rename:
   ```sh
   tmux rename-window -t <pane> "»${name}"
   ```
   Using the pane ID (not a window ID) lets tmux resolve to the containing window automatically.

4. If the rename fails (pane vanished between the pane-map refresh and the rename — rare but possible), log the failure and continue. Enrollment itself has already succeeded; the prefix is cosmetic and not worth aborting enrollment over.

**Scope of "enrollment"** (per §4 Enrollment bullet): operator sends a command to a change, user requests monitoring, autopilot spawns a change, a watch spawns a change. The rename applies to all of these paths. For operator-spawned windows the step is a no-op (see point 2).

### 2. No rename on removal (explicit non-behavior)

When a change is removed from the monitored set (§4 Removal bullet — reaches stop stage, terminal stage, pane dies, or user explicitly stops), the operator MUST NOT rename the window back. The `»` prefix persists. Rationale:

- The window may already be gone (pane death path).
- Restoring the original name requires storing it somewhere — adds state for no user benefit.
- The user's mental model is "once operator-touched, stays marked until they clean it up themselves", matching the request.

Users who want the prefix removed can rename the window manually (`Ctrl-b ,` in tmux).

### 3. Spawn path unchanged

`tmux new-window -n "»<wt>" …` in §6 Spawning an Agent (line 303) already produces a prefixed name and remains as-is. Deduplication is handled by the "already starts with `»`" guard in the enrollment algorithm — the spawn flow enrolls after creating the window, and the guard silently skips the redundant rename.

### 4. Skill documentation updates

Update `src/kit/skills/fab-operator.md`:

- In §4 Monitored Set → Enrollment bullet, add the rename step: "On enrollment, the operator MUST rename the target window to prefix `»` to its current name (idempotent — skipped if already prefixed). The rename uses `tmux rename-window -t <pane> "»<current-name>"`. On removal, the window name is left as-is."
- In §4 Monitored Set → Removal bullet, add the explicit non-behavior note: "The window name is not restored."
- In §6 Spawning an Agent → step 4 (enrollment), cross-reference the rename behavior so readers understand spawn and enrollment share one enrollment path.

### 5. Spec file update

Update `docs/specs/skills/SPEC-fab-operator.md` to reflect the new enrollment behavior. The existing spec is a summary (not a full mirror of the skill) and currently describes enrollment only implicitly. Add a short bullet under the Monitoring System section noting the window-rename-on-enrollment convention, consistent with the skill. Do not rewrite unrelated portions of the spec in this change.

## Affected Memory

- `fab-workflow/execution-skills` or equivalent: (modify) if the execution-skills memory file documents operator enrollment semantics; otherwise no memory update is required. The hydrate stage will determine this after reading `docs/memory/fab-workflow/index.md`. Flagged here so the spec-stage agent evaluates it rather than silently skipping.

## Impact

- **Files touched**: `src/kit/skills/fab-operator.md`, `docs/specs/skills/SPEC-fab-operator.md`. Possibly one memory file under `docs/memory/fab-workflow/` during hydrate.
- **Runtime cost**: one extra `tmux display-message` and at most one `tmux rename-window` per enrollment. Negligible.
- **Backwards compatibility**: additive behavior. Existing monitored sets carry over on `/clear` recovery; the idempotent guard means restored entries do not get double-prefixed.
- **External dependencies**: none. `tmux` is already a hard requirement of the operator (§2 Tmux Gate).
- **User-visible change**: window names in tmux tab lists gain the `»` prefix the moment the operator begins monitoring them.

## Open Questions

*(none — request is self-contained)*

## Clarifications

### Session 2026-04-22 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 10 | Confirmed | User pre-confirmed at invocation ("Agreed with 10.") |
| 6 | Confirmed | all ✓ |
| 7 | Confirmed | all ✓ |
| 8 | Confirmed | all ✓ |
| 9 | Confirmed | all ✓ |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rename target is the current window name (via `#W`), not session or pane titles | `»<wt>` convention at line 303 uses `tmux new-window -n` which sets the window name — same surface | S:95 R:90 A:95 D:95 |
| 2 | Certain | Rename is skipped when the name already starts with `»` (idempotent) | User said "prepend `»`"; double-prefixing (`»»name`) would be a bug. Also required for `/clear` recovery correctness | S:85 R:90 A:95 D:95 |
| 3 | Certain | Removal does not restore the original name | User explicitly stated "on removal, leave the name as-is" | S:100 R:90 A:100 D:100 |
| 4 | Certain | Canonical skill file is `src/kit/skills/fab-operator.md` | The path in the request (`src/kit/skills/fab-operator/SKILL.md`) does not exist; `src/kit/skills/fab-operator.md` does. Constitution §V.Additional says `src/kit/` is canonical | S:90 R:95 A:100 D:100 |
| 5 | Certain | Spec file `docs/specs/skills/SPEC-fab-operator.md` is updated alongside the skill | Constitution "Additional Constraints" mandates this; user also reiterated it in the request | S:100 R:90 A:100 D:100 |
| 6 | Certain | Rename uses `tmux rename-window -t <pane> "»<name>"` rather than a window-scoped selector | Clarified — user confirmed | S:95 R:85 A:90 D:80 |
| 7 | Certain | Enrollment-triggered rename applies to all enrollment paths (direct send, autopilot spawn, watch spawn, user request) | Clarified — user confirmed | S:95 R:85 A:85 D:85 |
| 8 | Certain | Rename failure (e.g., pane vanished between refresh and rename) logs and continues — it does not abort enrollment | Clarified — user confirmed | S:95 R:80 A:85 D:80 |
| 9 | Certain | The spec file gets a short additive note, not a rewrite | Clarified — user confirmed | S:95 R:90 A:90 D:85 |
| 10 | Certain | Memory file update (if any) lives under `docs/memory/fab-workflow/` | Clarified — user confirmed | S:95 R:80 A:60 D:70 |

10 assumptions (10 certain, 0 confident, 0 tentative, 0 unresolved).
