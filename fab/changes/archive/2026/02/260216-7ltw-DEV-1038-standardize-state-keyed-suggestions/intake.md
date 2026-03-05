# Intake: Standardize State-Keyed Next-Step Suggestions

**Change**: 260216-7ltw-DEV-1038-standardize-state-keyed-suggestions
**Created**: 2026-02-16
**Status**: Draft

## Origin

> Standardize next-step suggestions across all fab skills using a single state-keyed table in _context.md. Replace the current skill-keyed lookup table and all per-skill hardcoded Next: lines with one canonical state table where the key is the state reached (not the skill that ran). Drop fab-new's --switch flag and auto-switch logic to eliminate the conditional suggestion branch. Remove fab-switch's private stage→suggestion table. Make /fab-ff available only from spec (not intake), /fab-fff only from intake (not spec). Every skill ends with {per state table} instead of its own suggestion. This also makes /fab-status suggestions deterministic with zero custom logic.

This originated from a discussion analyzing the current suggestion system. The user opened `docs/specs/glossary.md` and asked to understand what suggestions are given after each workflow step. Analysis revealed three layers of duplication and several inconsistencies, leading to this standardization change.

**Interaction mode**: Conversational — extensive analysis preceded this intake. Key decisions were reached collaboratively.

## Why

1. **Duplication creates drift**: Suggestions are currently defined in 3 places — the `_context.md` lookup table, per-skill hardcoded `Next:` lines, and `fab-switch`'s private stage→suggestion table. When one is updated, the others go stale.

2. **Skill-keyed table can't handle convergence**: Multiple skills can land you in the same state (e.g., both `/fab-continue` and `/fab-ff` reach `hydrate`), but the current table is keyed by skill, so it has redundant rows that can diverge. A state-keyed table eliminates this — same state, same suggestions, regardless of how you got there.

3. **Non-deterministic suggestions**: `/fab-new` has two suggestion paths depending on `--switch` vs no-switch. `/fab-clarify` currently suggests `/fab-fff` from any planning stage, but `/fab-fff` is only meaningful from intake. Post-spec, the table shows both `/fab-ff` and `/fab-fff` as options, which is confusing since `/fab-fff` is redundant once past intake.

4. **No single source of truth for `/fab-status`**: Status currently needs its own suggestion logic rather than looking up the same table every other skill uses.

## What Changes

### 1. Replace skill-keyed lookup table with state-keyed table in `_context.md`

Replace the current "After skill | Stage reached | Next line" table (lines 68-86) with a state-keyed table:

```markdown
| State           | Available commands                              | Default          |
|-----------------|------------------------------------------------|------------------|
| (none)          | /fab-init                                       | —                |
| initialized     | /fab-new, /docs-hydrate-memory                  | /fab-new         |
| intake          | /fab-continue, /fab-fff, /fab-clarify            | /fab-continue    |
| spec            | /fab-continue, /fab-ff, /fab-clarify             | /fab-continue    |
| tasks           | /fab-continue, /fab-ff                           | /fab-continue    |
| apply           | /fab-continue                                    | /fab-continue    |
| review (pass)   | /fab-continue                                    | /fab-continue    |
| review (fail)   | (rework menu)                                    | —                |
| hydrate         | /fab-archive                                     | /fab-archive     |
| archived        | /fab-new                                         | /fab-new         |
```

Key design decisions embedded in this table:
- `/fab-fff` is only available from `intake` (full pipeline from the start)
- `/fab-ff` is only available from `spec` and `tasks` (needs spec to exist)
- `/fab-clarify` is available from `intake`, `spec`, and `tasks` (the planning stages only — not apply/review/hydrate)
- Each state has exactly one default command (or none for error states)

### 2. Update convention text in `_context.md`

Update the "Next Steps Convention" section to instruct skills to:
1. Determine the state they landed in
2. Look up that state in the table
3. Output `Next:` with available commands, bolding the default

Replace the current instruction:
> Every skill MUST end its output with a `Next:` line suggesting the available follow-up commands.

With something like:
> Every skill MUST end its output with a `Next:` line derived from the State Table below. Look up the state reached (not the skill name) and list the available commands. The default command SHOULD be listed first.

### 3. Remove `--switch` flag and auto-switch from `/fab-new`

In `fab/.kit/skills/fab-new.md`:
- Remove `--switch` from the Arguments section (line 22)
- Remove Step 6 "Activate Change (Conditional)" (line 65)
- Remove the conditional output `{if switched: "Branch: {name} (created)\n"}` (line 75)
- Remove the two-path suggestion at lines 104-106
- Replace with single suggestion derived from state table: state=`intake` → `Next: /fab-continue, /fab-fff, or /fab-clarify`

### 4. Remove `fab-switch`'s private stage→suggestion table

In `fab/.kit/skills/fab-switch.md`:
- Remove the inline table at lines 61-68
- Replace step 5 "Suggest next command based on stage" with: "Suggest next command per state table in `_context.md`"
- Update the output template at line 96 to reference the state table

### 5. Update all skills with hardcoded `Next:` lines

Each of these skills has hardcoded suggestions that should be replaced with `{per state table}`:

| File | Lines | Current hardcoded value |
|------|-------|------------------------|
| `fab-ff.md` | 118 | `Next: /fab-archive` |
| `fab-fff.md` | 146 | `Next: /fab-archive` |
| `fab-archive.md` | 104, 188 | `Next: /fab-new <description>`, `Next: /fab-switch {name}` |
| `fab-init.md` | 158 | `Next: /fab-new <description> or /docs-hydrate-memory <sources>` |
| `fab-clarify.md` | 90 | `Next: /fab-clarify or /fab-continue or /fab-ff` |
| `fab-continue.md` | 150 | `Next: /fab-continue` (review pass) |
| `fab-switch.md` | 96 | `Next: /fab-continue` |
| `docs-hydrate-memory.md` | 183 | `Next: /fab-new <description>, ...` |

### 6. Align `/fab-clarify` available commands

Currently suggests `/fab-fff` from any planning stage. After this change, `/fab-clarify` reads the current state from `.status.yaml` and uses the state table — so from `spec`, it would suggest `/fab-continue, /fab-ff, /fab-clarify` (not `/fab-fff`).

### 7. Ensure `/fab-status` uses the same table

`fab-status.md` should reference the state table for its "suggested next command" output, eliminating any custom suggestion logic.

## Affected Memory

- `fab-workflow/planning-skills`: (modify) Update to reflect state-keyed suggestion convention, removal of --switch from fab-new
- `fab-workflow/execution-skills`: (modify) Update to reflect state-keyed suggestion convention
- `fab-workflow/change-lifecycle`: (modify) Update stage transition documentation to reference state table
- `fab-workflow/clarify`: (modify) Update available commands per stage

## Impact

- **`fab/.kit/skills/_context.md`**: Primary change — new state table replaces skill-keyed table
- **`fab/.kit/skills/fab-new.md`**: Remove --switch, simplify to single suggestion path
- **`fab/.kit/skills/fab-switch.md`**: Remove private table, reference canonical table
- **`fab/.kit/skills/fab-ff.md`**: Replace hardcoded Next line
- **`fab/.kit/skills/fab-fff.md`**: Replace hardcoded Next line
- **`fab/.kit/skills/fab-archive.md`**: Replace hardcoded Next lines (2 paths: archive and restore)
- **`fab/.kit/skills/fab-init.md`**: Replace hardcoded Next line
- **`fab/.kit/skills/fab-clarify.md`**: Replace hardcoded Next line, make state-aware
- **`fab/.kit/skills/fab-continue.md`**: Replace hardcoded Next line
- **`fab/.kit/skills/fab-status.md`**: Reference state table for suggestions
- **`fab/.kit/skills/docs-hydrate-memory.md`**: Replace hardcoded Next line
- **`docs/specs/skills.md`**: Update if it documents suggestion behavior
- **`docs/specs/user-flow.md`**: Update command map if it shows suggestion paths

## Open Questions

- Should `/fab-archive --restore` map to a distinct state (e.g., `restored`) in the table, or should it just use the state of the restored change?
- Should the `(none)` state be explicit in the table or handled separately (since it's a pre-init condition)?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | State table keyed by state reached, not by skill | Direct outcome of the discussion — user confirmed Option B | S:95 R:85 A:90 D:95 |
| 2 | Certain | Remove --switch from fab-new | User explicitly asked "should we completely remove the ability to auto switch" and the analysis supports it — eliminates conditional branch | S:90 R:80 A:85 D:90 |
| 3 | Certain | /fab-fff only from intake, /fab-ff only from spec+ | Discussed and confirmed — eliminates the confusing overlap | S:90 R:75 A:90 D:90 |
| 4 | Confident | /fab-clarify available from intake, spec, tasks only | Planning stages are where clarification makes sense — apply/review/hydrate are execution stages. Not explicitly discussed but follows logically | S:70 R:85 A:80 D:75 |
| 5 | Confident | Each state has a single "default" command listed first | Discussed as part of Option B proposal — user didn't object but didn't explicitly confirm the bold/first convention | S:75 R:90 A:80 D:80 |
| 6 | Tentative | /fab-archive --restore uses restored change's state | Makes sense (restored change has a real stage), but wasn't discussed | S:50 R:70 A:65 D:55 |
<!-- assumed: archive restore uses the state of the restored change rather than a dedicated "restored" state — follows the principle of same state = same suggestions -->

6 assumptions (2 certain, 2 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
