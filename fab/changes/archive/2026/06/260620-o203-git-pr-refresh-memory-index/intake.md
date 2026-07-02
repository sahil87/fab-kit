# Intake: Refresh memory indexes post-commit in /git-pr (fix index date drift)

**Change**: 260620-o203-git-pr-refresh-memory-index
**Created**: 2026-06-20

## Origin

This change was synthesized from a design conversation that diagnosed and decided a fix for a
benign-but-noisy index date-drift symptom in the fab pipeline. It is dispatched **promptless**
(`/fab-proceed`-style, `{questioning-mode} = promptless-defer`) — no questions were asked; any
decision SRAD would normally surface is recorded as a deferred Unresolved row in `## Assumptions`.

> Refresh memory indexes post-commit in `/git-pr` (fix index date drift). `fab memory-index`
> stamps each `index.md` row's "Last Updated" cell from `git log` (committed dates only). During the
> hydrate stage, memory files are still uncommitted, so the regenerated index stamps each touched
> file's PREVIOUS commit date — the index is born "one regen behind" on every file the change
> edited. The real date only exists once the content commit lands (at ship). A subsequent
> `fab memory-index --check` then flags benign tier-1 drift (exit 1) at review-pr until a later regen
> catches up. THE FIX (decided): add a new sub-step **3a-bis** to `/git-pr`, between step 3a (Commit)
> and step 3b (Push) — the only position where `git log` knows the real commit date without coupling.

**Empirically verified this session.** Two confirmations were established alongside the diagnosis:
(1) the drift is **`index.md`-ONLY** — `log.md` is freeze-on-write / append-only (keyed on
`(file-base, change-id)`; existing entries are never re-dated) and does NOT drift; (2) squash-per-PR
does NOT flatten dates on `main` (a file keeps one real date per PR that touched it), so the drift is
the only real symptom and it is benign + self-healing.

**Alternatives considered and rejected** (captured here for traceability; see `## Impact` →
Design Decisions for the full rationale):

1. **Regen at the end of hydrate** — rejected: hydrate is entirely pre-commit, so `git log` still
   can't see the change's own commit; no position inside hydrate fixes it.
2. **Frontmatter timestamp field stamped into memory files** — rejected: loses the
   `fab memory-index --check` git oracle (the date becomes an un-validatable stored assertion), and
   overkill since squash isn't flattening dates.
3. **Unconditional regen in `/git-pr`** — rejected: would couple the general-purpose standalone
   `/git-pr` tool (used outside fab-kit) to fab; hence the `{has_fab}` gate, following git-pr's
   existing conditional-fab pattern (Steps 0a / 4a / 4c).
4. **`stage_hooks.ship.post`** — a viable alternative that keeps it in project config, but the user
   chose the `{has_fab}`-gated in-skill approach for now.

## Why

**The problem.** `fab memory-index` derives every `index.md` row's "Last Updated" cell from
`git log` — i.e. from **committed** dates only. The pipeline regenerates indexes at hydrate Step 5
(`docs/memory/pipeline/execution-skills.md` Hydrate Behavior: *"Regenerate indexes — run
`fab memory-index`"*), but at hydrate time the change's own memory edits are still **uncommitted**.
So for every file the change touched, `git log` returns that file's PREVIOUS commit date, and the
freshly regenerated index is born "one regen behind." The real date for these edits does not exist in
`git log` until the content commit lands — which happens at **ship** (in `/git-pr` step 3a).

**The consequence if unfixed.** A later `fab memory-index --check` (the refuse-before-regen oracle,
glwc) sees the stale "Last Updated" cells and flags **benign tier-1 drift** (exit 1) at the review-pr
stage. The change is self-healing — the next regen on a tree where the commit has landed corrects the
cell — but until that happens, every fab change leaves a spurious exit-1 drift signal that has to be
recognized as benign and worked around (regen + commit + push the indexes on the PR branch
post-ship). This is a recurring, known annoyance (the operator's recorded "memory-index date drift
after ship" lesson). It is noise, not corruption: there is no data loss and the dates are correct
once a post-commit regen runs.

**Why this approach over the alternatives.** The drift exists because the index is regenerated at a
moment when `git log` cannot yet see the relevant commit. The **only** position in the entire
pipeline where `git log` knows the real commit date — without coupling to anything else — is
**immediately after step 3a commits the content** in `/git-pr` (ship), and **before** the push in
step 3b. A regen there stamps the just-landed real date. Every other candidate position either still
runs pre-commit (hydrate, alternative 1), trades the validatable git oracle for a stored assertion
(frontmatter timestamps, alternative 2), or couples the standalone general-purpose `/git-pr` to fab
(unconditional regen, alternative 3). The `stage_hooks.ship.post` route (alternative 4) is viable but
the user chose the in-skill `{has_fab}`-gated approach for now.

## What Changes

This is a **skill-prose + spec-mirror** change. **No Go code change, no new CLI command, no
migration.** The canonical skill source is `src/kit/skills/git-pr.md` (NOT the gitignored deployed
copy at `.claude/skills/`). The constitution requires the corresponding
`docs/specs/skills/SPEC-git-pr.md` mirror to be updated in the same change.

### 1. New sub-step 3a-bis in `src/kit/skills/git-pr.md` (between 3a Commit and 3b Push)

Add a new sub-step **3a-bis: Refresh Memory Indexes** positioned BETWEEN step `#### 3a. Commit` and
step `#### 3b. Push` in the `### Step 3: Execute Pipeline` section. This is the only position where
`git log` knows the real commit date without coupling. Behavior:

- **Gating** — gated on BOTH conditions; skip the entire sub-step otherwise:
  - `{has_fab}` is true (the Step 0 variable), AND
  - step 3a **just committed this invocation** — i.e. the `has_uncommitted` path ran in 3a. It is
    NOT reached on the "already shipped" / no-changes re-run paths (where 3a did not commit).
- **Run the regen** — `fab memory-index` (byte-stable; writes only `docs/memory/` index + log
  files; a no-op when nothing drifted).
- **Conditional follow-up commit** — if `docs/memory/` changed
  (`git diff --quiet -- docs/memory` exits non-zero): stage `git add docs/memory` and make a
  **SEPARATE follow-up commit**:

  ```bash
  fab memory-index
  if ! git diff --quiet -- docs/memory; then
    git add docs/memory
    git commit -m "docs: refresh memory indexes"
  fi
  ```

  Do **NOT** use `--amend` — keep 3a's authored content commit intact; squash collapses the pair on
  merge anyway. If `git diff --quiet -- docs/memory` exits 0 (nothing drifted), make **no** commit
  (the guard suppresses an empty follow-up commit — Constitution III idempotency).
- **Failure handling** — if the regen OR the commit fails: **report the error and STOP**. The 3a
  content commit is already made and intact; a failed refresh leaves a benign stale-date index,
  recoverable by re-running `fab memory-index` — never a torn state.
- **Output** — print (ONLY when a follow-up commit was actually made):

  ```
    ✓ commit — "docs: refresh memory indexes"
  ```
- **Rationale note in the skill prose** — state that: this is the first moment `git log` reports the
  real commit date; the step lives in **ship** (not hydrate) because hydrate is entirely pre-commit;
  it is a **silent no-op** when `/git-pr` runs standalone outside a fab project (`{has_fab}` false),
  so the general-purpose standalone use of `/git-pr` is unaffected.

### 2. Update the Key Properties "Idempotent?" row in `src/kit/skills/git-pr.md`

Amend the existing `| Idempotent? | ... |` row in the `## Key Properties` table to note that 3a-bis
is gated on 3a-having-just-committed, so a re-run (the no-commit path) skips it; and even if reached,
`fab memory-index` is byte-stable and the `git diff --quiet -- docs/memory` guard suppresses an empty
follow-up commit.

### 3. Mirror the 3a-bis node in `docs/specs/skills/SPEC-git-pr.md` (constitution-required)

Add the 3a-bis node to the SPEC's Flow tree, positioned **between the `3a. Commit` node and the
`3b. Push` node**, with the rationale. The current Flow tree reads (excerpt):

```
│  ├─ 3a. Commit (if uncommitted)
│  │  ├─ ...
│  │  └─ Bash: git commit -m "<message>"
│  ├─ 3b. Push (if unpushed)
│  │  └─ Bash: git push [-u origin <branch>]
```

The new node goes between the `3a. Commit` subtree and the `3b. Push` subtree, e.g.:

```
│  ├─ 3a-bis. Refresh Memory Indexes (if {has_fab} AND 3a just committed)
│  │  ├─ Bash: fab memory-index  (byte-stable; writes only docs/memory/)
│  │  └─ Bash: if ! git diff --quiet -- docs/memory; then
│  │           git add docs/memory && git commit -m "docs: refresh memory indexes"
│  │     (no --amend — keeps 3a's content commit; squash collapses on merge;
│  │      first moment git log knows the real commit date; lives in ship not
│  │      hydrate because hydrate is entirely pre-commit; silent no-op when
│  │      {has_fab} false → standalone /git-pr unaffected;
│  │      regen/commit failure → report + STOP, 3a commit intact, no torn state)
```

A matching prose paragraph SHOULD be added to the SPEC's summary section describing the 3a-bis
behavior and its date-drift-fix rationale, consistent with how the SPEC documents other `/git-pr`
hardening (g8st, w7dp, etc.).

## Affected Memory

- `pipeline/execution-skills.md`: (modify) — it owns the `/git-pr` ship-behavior prose (the
  **PR shipping** section and the **Step 4** description). Hydrate will add a description of the new
  3a-bis sub-step: `fab memory-index` runs post-commit / pre-push in ship to stamp the real commit
  date, with a separate `docs: refresh memory indexes` follow-up commit, gated on `{has_fab}` AND
  3a-having-just-committed, and the date-drift-fix rationale (why ship and not hydrate). This is the
  same memory file whose long `description:` frontmatter already enumerates the g8st / w7dp / glwc
  `/git-pr` hardening — the 3a-bis note belongs in the same lineage.

## Impact

**Affected files** (the apply stage will edit these):

- `src/kit/skills/git-pr.md` — canonical skill source; add sub-step 3a-bis between 3a and 3b; amend
  the Key Properties Idempotent? row. (NEVER edit `.claude/skills/git-pr.md` — gitignored deployed
  copy, regenerated by `fab sync`.)
- `docs/specs/skills/SPEC-git-pr.md` — constitution-required SPEC mirror; add the 3a-bis Flow-tree
  node + a rationale paragraph.

**Out of scope / Non-Goals:**

- No Go code change (`cmd/fab`, `internal/`), no new/changed CLI command, no `_cli-fab.md` update —
  `fab memory-index` already exists and is unchanged; this change only adds a new *caller* of it in
  skill prose.
- No migration (skills redeploy via `fab sync`; no user-data restructuring).
- Does not change `fab memory-index` behavior, the hydrate-stage regen at Step 5, or the
  refuse-before-regen guard (glwc). The hydrate regen stays as-is; 3a-bis is an additional
  post-commit regen in ship.
- Does not touch `log.md` behavior — `log.md` is freeze-on-write / append-only and does not drift;
  this fix targets the `index.md` "Last Updated" drift only.
- Does not alter the standalone (`{has_fab}` false) behavior of `/git-pr` — the gate makes 3a-bis a
  silent no-op outside a fab project.

**Design Decisions** (with rejected alternatives):

- **Decision: post-commit / pre-push in ship (3a-bis), not in hydrate.** This is the only pipeline
  position where `git log` can see the change's own content commit. Rejected: regen at end of hydrate
  (hydrate is entirely pre-commit — no in-hydrate position fixes it).
- **Decision: separate follow-up commit, not `git commit --amend`.** Keeps 3a's authored content
  commit intact and reviewable; squash collapses the index-refresh commit into the content commit on
  merge anyway. Avoids rewriting an already-made commit.
- **Decision: `{has_fab}` gate, following git-pr's existing conditional-fab pattern (Steps 0a / 4a /
  4c).** Rejected: unconditional regen (would couple the general-purpose standalone `/git-pr` to fab).
- **Decision: `git diff --quiet -- docs/memory` guard before the follow-up commit.** Suppresses an
  empty commit when nothing drifted (Constitution III idempotency); `fab memory-index` itself is
  byte-stable, so a no-drift regen produces no diff and no commit.
- **Decision: fail → report + STOP, leaving the 3a content commit intact.** A failed refresh degrades
  to a benign stale-date index recoverable by re-running `fab memory-index`; it is never a torn state
  (the content commit is already durable).
- **Decision: keep the validatable `fab memory-index --check` git oracle.** Rejected: a frontmatter
  timestamp field (a stored assertion the `--check` oracle could no longer validate; also overkill
  since squash isn't flattening dates).
- **Considered but not chosen now: `stage_hooks.ship.post`.** A viable alternative that keeps the
  refresh in project config rather than skill prose; the user chose the in-skill `{has_fab}`-gated
  approach for this change.

**Constraints (from constitution / context):**

- `src/kit/` is canonical; `.claude/skills/` is gitignored deployed copies — edit only
  `src/kit/skills/git-pr.md`, never the deployed copy (Constitution V; context.md).
- Skill changes MUST update the corresponding `docs/specs/skills/SPEC-*.md` (Constitution Additional
  Constraints).
- Markdown-only artifacts; no build step (Constitution I / IV).

## Open Questions

- Should the 3a-bis follow-up commit also push within 3a-bis, or rely on step 3b to push it? The
  decided behavior places 3a-bis BEFORE 3b (Push), so step 3b naturally pushes both the 3a content
  commit and the 3a-bis index-refresh commit together — no separate push is described in 3a-bis.
  This intake assumes 3b handles the push (see Assumptions); flagged in case the apply stage finds a
  push-ordering subtlety in the existing 3b "if has_unpushed or just committed" condition.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add sub-step 3a-bis in `src/kit/skills/git-pr.md` positioned between `#### 3a. Commit` and `#### 3b. Push`. | Position explicitly decided in the change description and grounded in the constraint that this is the only point where `git log` sees the content commit; the canonical source path is fixed by Constitution V + context.md. | S:98 R:80 A:95 D:95 |
| 2 | Certain | Gate 3a-bis on BOTH `{has_fab}` (Step 0) AND 3a-having-just-committed (the `has_uncommitted` path ran); skip on the "already shipped" / no-change re-run paths. | Explicitly decided; mirrors git-pr's existing conditional-fab pattern (Steps 0a/4a/4c) and the established `{has_fab}` variable from Step 0. | S:97 R:78 A:92 D:92 |
| 3 | Certain | Make a SEPARATE follow-up commit `git commit -m "docs: refresh memory indexes"` (not `--amend`) only when `git diff --quiet -- docs/memory` exits non-zero. | Commit message, separateness, the no-amend rule, and the diff guard are all stated verbatim in the decided fix, with rationale (keep 3a content intact; squash collapses on merge; suppress empty commit). | S:98 R:75 A:90 D:95 |
| 4 | Certain | On regen-or-commit failure, report the error and STOP; the 3a content commit stays intact (no torn state, recoverable by re-running `fab memory-index`). | Failure semantics explicitly decided; consistent with git-pr's existing fail-fast rule and the durability of the already-made 3a commit. | S:96 R:80 A:90 D:92 |
| 5 | Certain | Print `  ✓ commit — "docs: refresh memory indexes"` only when a follow-up commit was actually made. | Output line and its condition stated verbatim; matches git-pr's existing `✓ <step>` progress-line convention. | S:97 R:90 A:90 D:95 |
| 6 | Certain | Add the 3a-bis node to `docs/specs/skills/SPEC-git-pr.md` Flow tree between the 3a Commit node and the 3b Push node, with rationale. | Constitution Additional Constraints require the SPEC mirror; placement explicitly specified; the SPEC's existing Flow tree shape is known (read this session). | S:95 R:85 A:95 D:90 |
| 7 | Certain | Update the Key Properties "Idempotent?" row in `git-pr.md` to note 3a-bis is gated on 3a-just-committed (re-run skips it) and is byte-stable + diff-guarded even if reached. | Explicitly decided; the exact Idempotent? row already exists in the skill and is the natural home for this note. | S:96 R:88 A:92 D:92 |
| 8 | Certain | Change type is `fix` (corrects the index date-drift symptom). | Stated explicitly as `fix`; the PostToolUse intake-write hook independently infers `fix` from "fix"/"drift" keywords — verified `change_type: fix` in `.status.yaml`, no override needed. | S:90 R:95 A:85 D:90 |
| 9 | Certain | Affected memory is `pipeline/execution-skills.md` (modify) — hydrate adds the 3a-bis ship-behavior description to the file that owns `/git-pr` ship prose. | Stated explicitly; verified that this file's `description:` frontmatter already enumerates the g8st/w7dp/glwc git-pr hardening, so the 3a-bis note belongs in the same lineage. | S:90 R:80 A:85 D:88 |
| 10 | Confident | Step 3b (Push) pushes both the 3a content commit and the 3a-bis index-refresh commit together; 3a-bis does NOT push on its own. | 3a-bis is placed BEFORE 3b and the decided behavior describes no push inside 3a-bis; 3b's existing "if has_unpushed or just committed" trigger naturally covers both commits. Low-risk ordering inference, easily reversible, flagged as an Open Question for apply to confirm. | S:78 R:80 A:80 D:75 |

10 assumptions (9 certain, 1 confident, 0 tentative, 0 unresolved).
