# Intake: Drop the "Last Updated" column from generated memory indexes

**Change**: 260625-ugde-memory-index-drop-date-column
**Created**: 2026-06-25

## Origin

Initiated from a `/fab-discuss` session about `fab memory-index` generating spurious changes when it shouldn't. The user observed (loom PR #1846) "a lot of changes that are ONLY date changes" — e.g. `docs/memory/lib-murphy/index.md` regenerating with a different `Last Updated` cell while the underlying memory content was unchanged. The user's requirement, verbatim:

> memory-index needs to get to a stable state such that for no doc changes, it really shouldn't update dates (idempotency) — it should be safe to re-run again and again.

The discussion diagnosed the root cause and weighed four options (drop the column / freeze-on-write the dates / pin to a stable ref / relax `--check`). The user explicitly chose **Option A — drop the "Last Updated" column entirely** (interactive selection in the discuss session). Mode: conversational; the design decision and full sweep class were resolved in-conversation before this intake was created.

## Why

**The problem.** The domain/sub-domain `index.md` renders `| File | Description | Last Updated |`, and the `Last Updated` cell is a **live `git log` projection** (author date `%ad`, via `loadGitDates` → `gitDates.lookup` in `src/go/fab/internal/memoryindex/memoryindex.go:316`, rendered at `:183-187`). `git log -1 -- <file>` is stable only for a *fixed HEAD*; it is HEAD/branch-relative. So:

- A branch cut from (or not rebased onto) a newer `main` doesn't contain commits that touched a file after its branch point → regenerating there projects an **older** date than `main`'s committed index → the regen *reverts* the cell.
- Concurrent PRs each branch at a different point and project a different date snapshot for the same unchanged files → the cells churn back and forth on merge. This is exactly the loom PR #1846 symptom: many files × many concurrent branches = "lots of date-only changes."

So the index is idempotent only under "same HEAD" — the one assumption that never holds across the branch/rebase/merge lifecycle. This violates Constitution III (Idempotent Operations).

**The consequence if unfixed.** Persistent date-only churn in every domain index; `fab memory-index --check` flags it as benign tier-1 drift (exit 1) at review-pr, forcing the recurring regen-and-recommit dance. An entire downstream workaround already exists for it — `/git-pr` sub-step 3a-bis (change `o203`), a post-commit/pre-push regen — and the cost keeps recurring on every change touching `docs/memory/`.

**Why this approach over alternatives.** This is the *exact* non-determinism that `fkf.md` §6.4 already recognized and fixed for `log.md` via **freeze-on-write** ("A pure projection of *live* git history is not deterministic… re-projecting from scratch on every run produces a different result per contributor and across time"). The index date column was never given the same treatment — and `fkf.md` §5 even *claims* the index render is "byte-stable / idempotent" while depending on git dates, which is false for the date half.

Of the four options:
- **A. Drop the column (chosen).** The date is the *only* non-content-derived, non-idempotent input. Remove it and the index becomes a pure function of content (file names + descriptions + structure) → genuinely branch-independent and idempotent, making §5's claim true. No capability is lost: dated, change-attributed history already lives in the per-folder freeze-on-write `log.md`. The index's job becomes pure navigation (what exists + what it's about); recency-at-a-glance is `log.md`'s job now.
- B. Freeze-on-write the date cell — stops backward drift only; still moves on any touch; re-introduces index parse/merge machinery the team kept out of this path; weaker guarantee than A.
- C. Pin dates to `origin/main`/merge-base — main still moves; new files show `—` until merged; breaks in shallow clones / no-origin / offline. Environment-dependent. Reject.
- D. Make `--check` ignore the date column — silences the gate only; the file still churns. Half-measure.

## What Changes

### 1. Go renderer (`src/go/fab/internal/memoryindex/memoryindex.go`) — the behavior change

`RenderDomain` drops the third column. The header/separator/row format change from:

```go
b.WriteString("| File | Description | Last Updated |\n")
b.WriteString("|------|-------------|-------------|\n")
// ...
fmt.Fprintf(&b, "| [%s](%s.md) | %s | %s |\n", f.Base, f.Base, desc, date)
```

to:

```go
b.WriteString("| File | Description |\n")
b.WriteString("|------|-------------|\n")
// ...
fmt.Fprintf(&b, "| [%s](%s.md) | %s |\n", f.Base, f.Base, desc)
```

(`memoryindex.go:176-188`). Also update the package doc comment (lines 1-20) and the `RenderDomain`/§5 doc references that mention `git log` dates / "stamping Last Updated".

Remove the now-dead date plumbing **on the index path only**:
- `FileEntry.LastUpdated` field (`:66-67`) and its population in `gatherFiles` (`:316`, the `dates.lookup(...)` call).
- `gitDates.byPath` (the newest-date-per-path map), `(*gitDates).lookup` (`:537-552`), and `gitLastUpdated` (`:559-569`) — the per-file date fallback. These exist **only** to serve the index date cell.
- In `parseGitLog` (`:488-529`), stop building/returning `byPath` (it has no remaining consumer).

**KEEP**: `loadGitDates`, the batched `git log` pass, and `commitsByPath` — `log.md` generation (`gatherLogEntries`) still depends on the per-path commit list. The `--name-status` projection and the `top`/`gitRelPath` machinery stay. Only the *date-map* projection is removed; the *commit-list* projection is untouched.

### 2. `--check` parser + classifier (`indexparse.go`, `loss.go`)

`fab memory-index --check` parses the existing committed `index.md` to detect destructive-loss categories (description / tombstone / grouping). The existing-index row parser must expect **2 columns** for the domain/sub-domain tier so tombstone/description/grouping detection still works after the format change. Verify `Classify` and any golden/structural assumptions about the 3-column domain table are updated. (Root index is unaffected — it was already `| Domain | Description |`.)

### 3. CLI help text (`src/go/fab/cmd/fab/memory_index.go`)

The `Long` description mentions "stamping \"Last Updated\" from git" (`memory_index.go:26-27`) and the tier-1 example "a refreshed `Last Updated`" — update both.

### 4. Tests

`internal/memoryindex/*_test.go` — golden fixtures (`golden_test.go`), render tests (`memoryindex_test.go`), and any `freeze_test.go` / `log_test.go` / `seed_test.go` / `loss_test.go` cases that assert the 3-column domain index must be updated to the 2-column form. Per `code-quality.md` Test Strategy (test-alongside) and Constitution VII (tests conform to spec), update the goldens to the new rendered output. Run the `internal/memoryindex` + `cmd/fab` package tests before considering the change done.

### 5. Spec + kit-reference mirror (constitution-pinned sync)

- `docs/specs/fkf.md` — §2 (line 59, the "stale Last Updated cell" conformance note), §5 (lines 208-209, the Domain tier `| File | Description | Last Updated |` description + the "git-stamped" sentence), §6.1 (line 244, "the same date source the index uses" — the index no longer uses dates; reword so it refers to `log.md`'s use of the batched pass only).
- `src/kit/reference/fkf.md` — the **normative mirror** (lines 38, 146-147). `fkf.md`'s own header rule: *"Any change to FKF normative rules MUST update both files."* Both move together.
- `docs/specs/templates.md` — lines 418, 429, 463, 541 (the index-hierarchy design rationale), 571 (the "never hand-edit Last Updated cells" instruction).
- `src/kit/skills/_cli-fab.md` — § fab memory-index (lines 495, 506, 573, 578, 636): the command reference's column description, the batched-pass note, and the tier-1 drift example.

### 6. Skills + their SPEC mirrors (the skill ↔ SPEC-*.md class)

Each `src/kit/skills/*.md` edit requires its `docs/specs/skills/SPEC-*.md` mirror (Constitution Additional Constraints; `code-quality.md` § Sibling & Mirror Sweeps):
- `src/kit/skills/docs-hydrate-memory.md` (lines 37, 113) + `SPEC-docs-hydrate-memory.md`
- `src/kit/skills/fab-continue.md` (line 209) + `SPEC-fab-continue.md`
- `src/kit/skills/docs-reorg-memory.md` + `SPEC-docs-reorg-memory.md` (line 41)
- `src/kit/skills/git-pr.md` (line 213, the 3a-bis rationale) + `SPEC-git-pr.md` (line 18) — **see the 3a-bis nuance below**.

Each occurrence drops/rewords the "Last Updated" column reference. Grep `Last Updated` repo-wide as the sweep anchor (the discussion enumerated the full hit list).

### 7. Migration (`src/kit/migrations/`)

The generated `index.md` files are user data; changing their column shape is a data restructuring → it MUST ship as a migration (context.md § Migrations; `code-review.md` project rule), not an ad-hoc script. The migration re-baselines every `index.md` to the 2-column form by running the new `fab memory-index`. Standard ordering: new binary first, then `/fab-setup migrations`. Pre-check that the installed binary produces the 2-column output before rewriting. That re-baseline commit is the **last** churn the repo sees from the date column; every run afterward is byte-stable.

### 8. The 3a-bis nuance (do NOT delete it)

`/git-pr` sub-step 3a-bis (change `o203`) is a post-commit/pre-push `fab memory-index` regen that was added *specifically* to close the index `Last Updated` date drift (hydrate's regen is pre-commit, so the index was born "one regen behind"). After dropping the column, **3a-bis is still required** — `log.md` (freeze-on-write) still needs that post-commit projection to capture the change's *own* entry while its commits are still reachable (pre-squash). What changes is its **rationale**: it narrows from "index dates + log.md" to "**log.md only**," and its index-regen half becomes a reliable no-op (the index no longer depends on commit timing). **Rewrite 3a-bis's prose (in `git-pr.md`, `SPEC-git-pr.md`, and `pipeline/execution-skills.md`) to reflect the narrowed rationale — do not rip the sub-step out.**

## Affected Memory

- `memory-docs/hydrate`: (modify) Index Maintenance / Generated Index sections — remove the `| File | Description | Last Updated |` column and the "git-stamped Last Updated" date-sourcing prose (lines 87, 97, 120); the index is now content-only.
- `memory-docs/hydrate-generate`: (modify) the domain-row format reference (line 89) → `| File | Description |`.
- `memory-docs/templates`: (modify) Index Hierarchy (lines 140-141, 187) and the design-rationale "Why" (line 254) — domain/sub-domain rows are now `| File | Description |`; recency lives in `log.md`.
- `distribution/kit-architecture`: (modify) the `fab memory-index` subcommand description (line 324) and the `internal/memoryindex` architecture paragraph (line 331) — drop the date-stamping description and the dead `byPath`/`lookup`/`gitLastUpdated` references; note `loadGitDates`/`commitsByPath` retained for `log.md`.
- `pipeline/execution-skills`: (modify) the hydrate "Regenerate indexes" step (line 222) and the 3a-bis design decision (lines 63, plus the index `description:` line 3) — narrow 3a-bis rationale to log.md-only.
- `pipeline/schemas`: (modify) the batched-git-pass description (line 156) — the one pass now yields only `commitsByPath` for `log.md`; the `byPath` index-date projection is removed.

> Frozen, NOT modified: every `log.md` / `log.seed.md` historical entry that mentions the old 3-column format. They accurately describe what was true when written and are append-only/frozen artifacts — leave them.

## Impact

- **Code**: `src/go/fab/internal/memoryindex/{memoryindex.go,indexparse.go,loss.go}` + tests; `src/go/fab/cmd/fab/memory_index.go`. Net effect is partly a *deletion* (dead date plumbing).
- **Generated data**: every `docs/memory/**/index.md` loses a column (one-time, via migration). Root `index.md` unaffected.
- **Skills/specs/docs**: the `Last Updated` sweep class enumerated above (specs + kit reference mirror + 4 skills + their SPEC mirrors + 6 memory files).
- **Downstream**: `/git-pr` 3a-bis retained with narrowed rationale; the review-pr benign-drift dance disappears for the date column.
- **No API/flag changes**: `fab memory-index` keeps the same flags (`--check`, `--json`, `--rebuild`); only its rendered output and help text change. `--check` exit-code contract is unchanged.

## Open Questions

- None blocking. The design (Option A), the full sweep class, the migration approach, and the 3a-bis disposition were all resolved in the discuss session. Minor execution-time judgment (exact wording of the migration pre-check; whether `parseGitLog` keeps a 2-tuple return signature or collapses to one) is apply-stage decide-and-record, not a human gate.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop the `Last Updated` column entirely (Option A), rather than freeze-on-write the dates, pin to a ref, or relax `--check` | User explicitly selected Option A in the discuss session after a four-option comparison; it is the only option that makes the index a pure function of content (true idempotency, Constitution III) | S:95 R:70 A:95 D:95 |
| 2 | Certain | Keep `loadGitDates` + `commitsByPath`; remove only the `byPath`/`lookup`/`gitLastUpdated` date-map plumbing | `log.md` generation (`gatherLogEntries`) consumes `commitsByPath` from the same batched pass; only the index date cell consumed `byPath` | S:90 R:80 A:95 D:90 |
| 3 | Certain | Ship a `src/kit/migrations/` file to re-baseline existing `index.md` files to 2-column | Generated index files are user data; column-shape restructuring MUST ship as a migration per context.md § Migrations + the code-review project rule, never an ad-hoc script | S:85 R:75 A:95 D:90 |
| 4 | Certain | Retain `/git-pr` sub-step 3a-bis; rewrite its rationale to log.md-only instead of deleting it | `log.md` freeze-on-write still needs a post-commit projection to capture the change's own entry pre-squash; only the index-date justification disappears | S:90 R:65 A:90 D:85 |
| 5 | Certain | Sweep the full `Last Updated` mirror class (specs + `src/kit/reference/fkf.md` + 4 skills + SPEC mirrors + 6 memory files) in one change | Missed sibling/mirror sweeps are this repo's #1 rework cause (`code-quality.md` § Sibling & Mirror Sweeps); reviewers treat SPEC-mirror + fkf dual-file sync as must-fix | S:90 R:60 A:90 D:85 |
| 6 | Confident | Update `indexparse.go`/`loss.go` so `--check` parses the 2-column domain table; `--check` exit-code contract unchanged | The destructive-loss detectors parse existing index rows; the column count is a structural input they must track. The classifier categories (description/tombstone/grouping) are column-shape-independent in intent | S:70 R:75 A:85 D:80 |
| 7 | Confident | Treat this as a single change (Go + specs + skills + memory + migration together), not split per surface | The Go render and its doc/spec/skill mirrors are constitution-pinned to move together; splitting would ship a skill change without its SPEC mirror (a must-fix violation) | S:75 R:70 A:85 D:80 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
