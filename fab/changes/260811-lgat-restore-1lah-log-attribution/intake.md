# Intake: Restore 1lah's Missing Log Attribution (lgat)

> **Title-token note (deliberate)**: this heading feeds `/git-pr`'s PR title, which becomes the squash-merge subject. It must carry the standalone token `lgat` so `attributeCommit` credits THIS change's `pipeline/schemas.md` edit, and must NOT contain `1lah` as a standalone token (the possessive `1lah's` does not tokenize to a change id) — otherwise the squash commit would be mis-attributed to change 1lah. Do not "clean up" this heading.

**Change**: 260811-lgat-restore-1lah-log-attribution
**Created**: 2026-08-11

## Origin

Backlog item `[lgat]` (2026-08-11), taken up autonomously after an investigation confirmed it valid:

> [lgat] 2026-08-11: (BUG-ish, main-side fix) `docs/memory/runtime/log.md` 2026-08-10 entries miss change 1lah's memory updates — `memoryindex.attributeCommit` needs the change-folder token or bare 4-char id in the commit SUBJECT, and neither 1lah's ship commit nor PR #568's squash subject carried `1lah`. Fix on main: a commit touching those memory files with `1lah` in the subject + `fab memory-index`, or a `log.seed.md` entry. Never hand-edit the generated log (FKF §5).

Interaction mode: one-shot autonomous (user instructed: confirm the bug, then run the pipeline end to end; record ambiguous calls in the intake). The investigation preceding this intake verified every claim against the codebase — see Why.

## Why

1. **The pain point**: `docs/memory/runtime/log.md` is the dated change-history ledger for the runtime memory domain, but its `## 2026-08-10` section credits only change `ki9v` (PR #566). Change `1lah` (PR #568, "Provider-Generic Pane Verbs") substantively updated `docs/memory/runtime/agent-primitives.md`, `dispatch.md`, and `pane-commands.md` (70 lines in pane-commands alone) on the same day, yet the log carries **no row for any of them**. Anyone reading the log to trace when/why pane-commands.md changed finds a hole.

2. **Root cause (verified)**: `attributeCommit` (`src/go/fab/internal/memoryindex/memoryindex.go:767`) recovers a change from a commit **subject** only — either the full `YYMMDD-XXXX-slug` folder token or a bare registered 4-char id. PR #568's squash subject is `feat: Provider-Generic Pane Verbs — Extract open/ready/deliver Primitives from Dispatch (#568)` — no token. Under freeze-on-write (FKF §6.4; `gatherLogEntries`, memoryindex.go:1154) an unattributable commit is **silently dropped** on every normal regen (projected only at bootstrap/`--rebuild`). So no future `fab memory-index` run can ever recover these rows from git. `fab memory-index --check` exits 0 — the gap is invisible to the drift checker (a frozen log that is a valid superset/equal of the merge passes; a *missing unattributable* projection is not drift).

3. **If we don't fix it**: the runtime domain's history permanently misattributes 2026-08-10 — the ki9v rows (frozen into the log by 1lah's own PR, whose stacked branch history still contained ki9v-attributable commits) stand alone, implying kimi enablement was the only runtime-memory activity that day. The 1lah design history (provider-generic pane verbs — a significant extraction) is exactly the kind of dated *what* the log exists to preserve.

4. **Why the seed-entry approach over the alternatives** (the backlog offers two fix vehicles):
   - **Chosen — `log.seed.md` recovery entries**: `log.seed.md` is the FKF §6-designated *curated input* for rows the git projection cannot produce (`pipeline/schemas.md` § log.seed.md Seed-Merge; `seed.go`). Entries carry their **own authored date**, so the rows land under `## 2026-08-10` where they belong; `mergeSeedEntries` dedupes byte-equal rows so regen stays idempotent (Constitution III); the generator remains the sole writer of `log.md` (FKF §5).
   - **Rejected — a new commit touching those memory files with `1lah` in the subject + regen**: the projection dates a row by the *commit* date, so the rows would land under 2026-08-11 (wrong day), and it requires artificial no-op edits to three memory files just to make git list them — history pollution to trick the projector.
   - **Rejected — hand-editing `log.md`**: violates the single-writer discipline (FKF §5/§6.4); the drift checker would then flag hand-edited lines the merge cannot reproduce; explicitly forbidden by the backlog note.

## What Changes

### 1. `docs/memory/runtime/log.seed.md` — three recovery entries (the fix)

Add a `## 2026-08-10` section (newest-first, so **above** the existing `## 2026-06-13` section, below the header comment) with one FKF §6.2-shaped row per 1lah-edited topic file, using 1lah's curated `.status.yaml` `summary:` verbatim and the bare `(1lah)` id (matching the bare-id style of the generated ki9v rows already in this log):

```markdown
## 2026-08-10
- **Update** [agent-primitives](/runtime/agent-primitives.md) — New provider-generic fab pane open/ready/deliver primitives in internal/pane; dispatch open/ready/deliver become thin record-keeping bindings; fab pane send warns and proceeds on unknown agent state (1lah)
- **Update** [dispatch](/runtime/dispatch.md) — New provider-generic fab pane open/ready/deliver primitives in internal/pane; dispatch open/ready/deliver become thin record-keeping bindings; fab pane send warns and proceeds on unknown agent state (1lah)
- **Update** [pane-commands](/runtime/pane-commands.md) — New provider-generic fab pane open/ready/deliver primitives in internal/pane; dispatch open/ready/deliver become thin record-keeping bindings; fab pane send warns and proceeds on unknown agent state (1lah)
```

Exactly these three files and no others:
- PR #568's memory-file footprint is `runtime/{agent-primitives,dispatch,index,pane-commands}.md` plus the two regenerated logs (`runtime/log.md`, `_shared/log.md`).
- `index.md` and `log.md` are excluded from log projection by the generator (`gatherLogEntries` skips them), so they get no rows by design.
- `_shared/log.md`'s #568 diff added only ki9v rows — 1lah edited no `_shared` topic file, so no `_shared/log.seed.md` entry is warranted.
- Verb is `**Update**` for all three (files modified, not created/removed).

Precede the new section with a short HTML comment explaining what these rows are, e.g.:

```markdown
<!-- 2026-08-10 (backlog lgat): recovery entries for change 1lah (PR #568) — the squash
     subject carried no change token, so the git projection can never attribute these
     memory edits (FKF §6.4 unattributable-drop). Summary text is 1lah's .status.yaml
     summary verbatim. -->
```

The file's existing header comment ("Edit only to correct a preserved historical entry") is left **verbatim** — it describes the oovf-cutover seed rows; the in-place comment documents this second, exceptional use without rewording shared boilerplate that other domains' seed files also carry. (No comment line may begin with `- ` or `## ` — `parseLogBody` treats such lines as entries/date headings.)

### 2. `docs/memory/runtime/log.md` — regenerated (not hand-edited)

Run the installed `fab memory-index` (2.19.4, matches project version) after editing the seed. Expected diff: exactly the three rows appear in the existing `## 2026-08-10` section, interleaved per the generator's deterministic file-base → change-id sort (each 1lah row immediately before its file's ki9v row, since "1lah" < "ki9v"). No other log or index file should change; if the regen produces unrelated churn, inspect before committing.

### 3. `docs/memory/pipeline/schemas.md` — record the recovery-entry usage (hydrate)

The § log.seed.md Seed-Merge section currently frames the seed only as the pre-FKF cutover ledger. Add a brief note that seed entries are also the designated recovery path for a real change whose squash subject dropped the change token (the FKF §6.4 unattributable class), with 260811-lgat as the precedent — so the next occurrence (a known recurring failure mode) has a documented playbook.

## Affected Memory

- `pipeline/schemas`: (modify) add the seed-recovery-entry usage note to § log.seed.md Seed-Merge
- `runtime/log.seed.md` and `runtime/log.md` are **implementation targets** of this change (curated-input edit + regen), not hydrate subjects — listed here only so hydrate does not double-document them

## Impact

- **Files**: `docs/memory/runtime/log.seed.md` (hand-curated edit), `docs/memory/runtime/log.md` (regenerated), `docs/memory/pipeline/schemas.md` (hydrate note). Zero Go/source changes — `attributeCommit`'s subject-only contract and the freeze-on-write drop are working as designed (FKF §6.4 documents the squash-dropped-token class explicitly); this is a data restoration, not a code fix.
- **Tests**: none — no code changes. Verification is behavioral: regen is idempotent (second `fab memory-index` run is a byte-for-byte no-op) and `fab memory-index --check` stays green.
- **Risk**: low; every edit is reversible text, and the seed-merge dedup guarantees re-runs are no-ops.

## Open Questions

- None — the backlog entry prescribed the fix space, and the investigation resolved the vehicle choice.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Bug is valid: log.md 2026-08-10 misses 1lah's rows; root cause is #568's token-less squash subject + freeze-on-write unattributable-drop | Verified directly: #568 diff/subject, attributeCommit + gatherLogEntries code, current log.md contents | S:95 R:90 A:95 D:95 |
| 2 | Confident | Fix vehicle: `log.seed.md` recovery entries, not an artificial `1lah`-subject commit | Backlog offers both; seed wins on correct 2026-08-10 dating, no artificial file touches, idempotent merge (FKF §6, Constitution III) | S:80 R:85 A:85 D:70 |
| 3 | Certain | Exactly 3 rows — agent-primitives, dispatch, pane-commands; bare `(1lah)` id; `**Update**` verb; summary = 1lah's `.status.yaml` `summary:` verbatim | #568 memory footprint verified; index.md/log.md excluded by generator; matches §6.2 shape and the neighboring ki9v rows' bare-id style | S:90 R:90 A:90 D:85 |
| 4 | Tentative | Leave the seed header boilerplate verbatim; document the exception via an in-place HTML comment above the new section | Amending the shared oovf boilerplate in one domain's seed would diverge from the other seed files; an adjacent comment is honest and local. Reasonable people could instead amend the header | S:60 R:85 A:70 D:45 |
| 5 | Confident | Record the recovery-entry precedent in `pipeline/schemas.md` at hydrate; no spec (`docs/site/fkf.md`) change | The normative FKF spec already defines seed-as-curated-input and the unattributable class; only the memory-side usage note is missing. Squash-token-drop is a known recurring failure mode worth a playbook | S:70 R:80 A:80 D:70 |
| 6 | Confident | Leave the `[lgat]` backlog checkbox unticked; archive-time marking handles it | Existing convention: `/fab-archive` marks backlog items when the change is archived | S:65 R:90 A:75 D:75 |

6 assumptions (2 certain, 3 confident, 1 tentative, 0 unresolved).
