# Intake: Add fab-proceed to operator skill Pipeline References

**Change**: 260326-4r2p-add-proceed-to-cli-fab
**Created**: 2026-03-26
**Status**: Draft

## Origin

> Add fab-proceed to the operator skills — the new context-aware orchestrator skill was implemented in PR 278 but its entry was missed from the Pipeline Reference sections in fab-operator6 and fab-operator7. The operators need to know about `/fab-proceed` so they can use it instead of manually chaining `/fab-new` → `/fab-switch` → `/git-branch` → `/fab-fff`.

Course correction after initial misunderstanding: the target is the operator skill files, not `_cli-fab.md`. The operators learn about fab skills through a hardcoded Pipeline Reference section — `/fab-proceed` is absent from both.

## Why

`/fab-proceed` is a context-aware orchestrator that replaces the manual chain of `/fab-new` → `/fab-switch` → `/git-branch` → `/fab-fff`. Both `fab-operator6.md` and `fab-operator7.md` have a "Pipeline Reference" section (§6 Coordination Patterns) that lists setup, pipeline, and maintenance commands. This is how operators decide which command to send to an agent. Without `/fab-proceed` in this list, operators will never use it — they'll always fall back to the manual multi-step chain.

## What Changes

### Add `/fab-proceed` to Pipeline Reference in `fab/.kit/skills/fab-operator7.md`

The Pipeline Reference (line ~267) currently lists:

```
Setup commands: /fab-new (create change), /fab-switch (activate), /git-branch (align branch)
Pipeline commands: /fab-continue (one stage), /fab-fff (full pipeline), /fab-ff (fast-forward to hydrate)
```

Add `/fab-proceed` as either a setup or pipeline command with a brief description indicating it auto-detects state and chains the needed prefix steps before `/fab-fff`. It belongs logically between setup and pipeline commands since it orchestrates both.

### Add `/fab-proceed` to Pipeline Reference in `fab/.kit/skills/fab-operator6.md`

Same change — operator6 has an identical Pipeline Reference section (line ~247).

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Note that operator skills' Pipeline References now include `/fab-proceed`

## Impact

- `fab/.kit/skills/fab-operator7.md` — Pipeline Reference section
- `fab/.kit/skills/fab-operator6.md` — Pipeline Reference section
- No CLI or Go binary changes
- No other skills affected

## Open Questions

None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Target is operator skill files, not _cli-fab.md | User corrected — operators learn skills from their hardcoded Pipeline Reference | S:95 R:95 A:95 D:95 |
| 2 | Certain | Both operator6 and operator7 need the same update | Both have identical Pipeline Reference sections missing fab-proceed | S:95 R:90 A:95 D:95 |
| 3 | Confident | fab-proceed listed as a pipeline/orchestrator command | It orchestrates setup + pipeline steps; fits between setup and pipeline categories | S:80 R:90 A:80 D:75 |
| 4 | Certain | This is a docs change type | Only markdown skill files are modified | S:95 R:95 A:95 D:95 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
