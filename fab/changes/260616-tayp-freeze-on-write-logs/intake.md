# Intake: Freeze-on-write log.md generation

**Change**: 260616-tayp-freeze-on-write-logs
**Created**: 2026-06-16

## Origin

Initiated from a `/fab-discuss` session investigating non-deterministic `log.md`
regeneration, prompted by wvrdz/loom PR #1610 — an unrelated content-safety change
(`260605-tqgd`) that, merely by touching `docs/memory/`, dragged in churn across **36
`log.md` files**. The user's framing: "the way the dates are being generated isn't very
deterministic … it's overwriting the [entries] every time the command is run. Person A
runs the log generation command and then person B runs it, only the diffs at most should
get regenerated not the whole files."

> Person A runs the log generation command and then person B runs it, only the diffs at
> most should get regenerated not the whole files, like we are seeing in this PR.

**Interaction mode**: conversational. The session diagnosed the root cause, prototyped two
candidate fix directions against loom's *real* data, and resolved every design decision
before this intake was created. Key milestones:

1. **Symptom corrected.** The user's initial read ("dates being overwritten") was refined:
   the dates are stable; what thrashes is the **per-entry descriptive text**, which is
   sourced from the live git commit *subject* (`gatherLogEntries`,
   `src/go/fab/internal/memoryindex/memoryindex.go:833-839` — the unattributable branch sets
   `summary = touch.Subject`).
2. **Root cause confirmed by reproduction.** On loom's `main`, `fab memory-index` (v2.5.5,
   the installed binary) was run against a clean checkout: it rewrote 36 `log.md` files.
   The migration commits `Part 2a`/`Part 2b` had been **squash-merged** into a single commit
   `#1721 (3ae63af)`; `git log` no longer sees the pre-squash commits, so the regenerated log
   collapsed/reworded every entry derived from them. 50 of 146 entries in `wd-web-canvas`
   thrash; all 50 are **unattributable** (carry no `(change-id)` token).
3. **Fix directions prototyped, not assumed.** Two candidate keys were tested on loom's actual
   before/after data (committed log vs. post-squash projection):
   - **Change-id keying alone (Option 2)** cannot fix the loom case — the thrashing entries have
     no change-id, so any key degrades to content-keying, which the squash broke. Dropping
     unattributable entries would delete 50/146 entries (all real history-of-tooling).
   - **Naive append-only** grew the log 146 → 161 (additive churn — the squashed `#1721` looks
     like a brand-new entry alongside the frozen `Part 2a/2b` lines).
   - **Freeze-on-write + change-id key + stop-projecting-unattributable (strategy "S6")** produced
     **0 churn across all 42 loom log folders** and was idempotent (146 → 146 → 146). This is the
     chosen design.
4. **Commit-id-as-key explicitly rejected.** The user asked whether git commit IDs (`%H`) would
   be easier than change IDs. They are trivially available (one-line `--format` change) and total
   (every commit has one), but **squash + branch-delete makes the commit hash unreachable** — the
   exact operation we are fixing. The change-id survives in the change folder name and the
   registry, independent of git. Decision: **change-id is the key.**

## Why

**Problem.** `fab memory-index` regenerates each `log.md` as a *pure function of live git state*
(`RenderLog` over `LogData`, where `LogData` is assembled from `git log --name-status` subjects
joined with `.status.yaml` summaries). The render is byte-stable for a *fixed* git history — but
git history is **not** fixed: squash-merge rewrites commit subjects and counts, and branch deletion
makes the original commits unreachable. So the generated content is a function of *whatever history
happens to be reachable at run time*, which differs between contributors and across time.

**Consequence if unfixed.** Every contributor whose PR touches `docs/memory/` (or who merely runs
`fab memory-index`) regenerates and re-commits dozens of unrelated `log.md` files — review noise,
merge-conflict surface, and a permanently-red `fab memory-index --check` (it byte-compares against
a fresh projection that no longer matches the committed-but-now-squashed history). This is a direct
violation of Constitution III (Idempotent Operations) in practice: re-running the skill on a
different machine/time produces a different result.

**Why this approach over alternatives.** The current design's central premise — *"log.md is a pure
function of git+status; regenerate freely"* — is the bug. The fix inverts it to *"the existing log
is authoritative; never re-derive what's already written."* This generalizes the **existing**
`log.seed.md` mechanism (`seed.go`), which is already a frozen, git-independent, read-but-never-written
entry store: freeze-on-write makes the *whole* log behave like the seed after first write. The change-id
key (not commit-id) is chosen because it is the only entry identity that survives squash + branch-delete.
The "stop projecting unattributable commits" rule is what closes the gap for entries that have no
change-id at all — and the prototype proved it is exactly the rule that yields zero churn.

## What Changes

### 1. Freeze-on-write generation (the core architecture)

`fab memory-index` stops rebuilding each `log.md` from scratch. New flow:

1. **Read the existing `log.md`** for the folder (parse it back into `[]LogEntry` — the inverse of
   `RenderLog`, the same parse∘render-identity discipline `parseSeedLog` already implements in
   `seed.go:28-31`).
2. **Treat existing entries as immutable and authoritative.** They are never rewritten, reworded,
   re-dated, or dropped.
3. **Project current git history** as today (`gatherLogEntries`), but use the projection only to
   discover **new** entries to *append*.
4. **Append only** entries whose identity is not already recorded (key defined in §2).
5. **Re-render** via `RenderLog` (unchanged — it stays the pure date-grouped renderer) over the
   merged `existing ∪ appended` set.

The `log.seed.md` seed-merge (`buildLogTarget` → `mergeSeedEntries`) is preserved: at first write (or
`--rebuild`), seed entries still merge beneath the git projection. After first write, the on-disk
`log.md` IS the frozen store and the seed is a no-op for already-present entries (its entries are
already in the file).

### 2. Append/dedup key = change-id (NOT commit-id)

The append guard keys on **`(file-base, change-id)`**:

- An attributable projected entry (one whose commit resolves via `attributeCommit`,
  `memoryindex.go:662`, to a registry change-id) is appended **only if** no existing entry already
  has that `(file-base, change-id)` pair. Re-running, or re-projecting after a squash that *preserved*
  the change token, is a no-op.
- **Commit-id (`%H`) is explicitly NOT the key.** Rationale recorded in Origin #4: squash + branch
  delete makes the hash unreachable, so it fails on the exact operation being fixed.

```
existing log.md (frozen):           projection (live git):            result:
  [foo](…) — summary (a1b2)           [foo] commit X → change a1b2       no-op (a1b2 present)
                                       [foo] commit Y → change c3d4       APPEND (c3d4 new)
```

### 3. Unattributable commits are frozen, not re-projected

A commit with no registry-matching token (migrations, docs-reorgs, direct-main edits) has **no
change-id to key on**. The rule (strategy S6, proven to yield 0 churn on loom):

- Unattributable entries **already present** in `log.md` at the time of the run stay **verbatim**
  (frozen — they were written at bootstrap and are never touched again).
- **New** unattributable commits are **NOT projected into the log** at all after first write.

**Accepted tradeoff** (decided with the user): future migration/reorg commits leave no log trace.
This is intentional — those are tooling commits, not memory-domain history. On loom, *all* 50
unattributable entries are migration/reorg commits; none represent content history a reader wants.

> Without this rule, a squashed unattributable commit (whose subject text changed) would be seen as a
> *new* entry and appended alongside the frozen old line → the 146 → 161 additive churn the prototype
> measured. Freezing-and-not-reprojecting is what produces the 0-churn result.

### 4. `--rebuild` flag (the deliberate destructive escape hatch)

```
fab memory-index --rebuild
```

Discards the accumulated frozen state and re-projects every `log.md` from current git (today's
behavior, made explicit and opt-in). Use cases: a corrupted frozen log, or a deliberate re-baseline.
It is **destructive** (it can rewrite/drop frozen lines) and must be loud — never the default path.

**Explicitly NOT a `--first-generation` flag.** Bootstrap is not a special mode: it is simply the
first append into an empty log, plus the pre-existing `log.seed.md` seeding. A `--first-generation`
flag would invite re-running it on run #2 and re-introducing the churn. The first run on a project
with no `log.md` projects-and-freezes through the same code path as every later run.

### 5. `--check` redesign (byte-equality → subset/superset)

Today `--check` regenerates and byte-compares (`cmd/fab/memory_index.go:94-130`, classified via
`internal/memoryindex/loss.go` `Classify`/`LossReport`). Under freeze-on-write, byte-equality is the
*wrong* check — a valid frozen log legitimately contains lines not derivable from current git (squashed
-away commits). New semantics:

- **FAIL** when an attributable entry that the projection says *should* exist is **missing** from the
  committed log (a genuine gap — someone forgot to regenerate-and-commit).
- **FAIL** when an existing line was **hand-edited** (the single-writer discipline was violated).
- **PASS** when the committed log is a valid superset of the projection (it has frozen lines the live
  history no longer shows) — this is the case that false-fails today.

This reworks `loss.go`'s tier machinery and `emitCheckReport` in `cmd/fab/memory_index.go`.

### 6. Re-baseline migration (existing projects)

Ship a migration in `src/kit/migrations/` (next version after 2.5.5) that transitions existing
projects (loom et al.) to the freeze-on-write model:

1. Run `fab memory-index --rebuild` once to produce a clean frozen baseline from current git.
2. Commit that baseline as the starting point.
3. From there, freeze-on-write keeps every subsequent run append-only stable.

**This migration IS the fix for existing repos like loom** — there is no separate manual step. The
upgrade ordering is the standard one and MUST be respected: the new **binary** lands first
(`brew upgrade fab-kit`), *then* `/fab-setup migrations` applies this migration. Applying it with an
older binary would fail on the unknown `--rebuild` flag — the migration's pre-check SHALL verify the
running binary understands `--rebuild` (e.g., probe `fab memory-index --help` or the kit VERSION)
and abort with a clear "upgrade the binary first" message if not.

The re-baseline commit is itself a **one-time, intentional churn** (it rewrites the currently-stale
`log.md` files — 36 on loom — into the clean frozen form). That commit is the *last* churn the repo
sees from this issue; every run afterward is append-only stable. The migration is idempotent
(re-running `--rebuild` + commit on an already-clean tree is a no-op diff). It follows the existing
migration format (`docs/memory/distribution/migrations.md`, `src/kit/migrations/2.4.2-to-2.5.0.md`
as the FKF-cutover precedent).

### 7. Docs + tests (constitution-mandated)

- **Spec**: update `docs/specs/fkf.md` §6 (the C-lite log spec) to describe freeze-on-write,
  the change-id key, the unattributable-freeze rule, `--rebuild`, and the new `--check` semantics.
- **CLI reference**: update `src/kit/skills/_cli-fab.md` § `fab memory-index` with the `--rebuild`
  flag and the changed `--check` contract (constitution: "Changes to the `fab` CLI … MUST update
  `src/kit/skills/_cli-fab.md`").
- **Tests**: see the explicit test matrix in §8 below — this is a first-class deliverable, not a
  follow-up. Every behavior rule in §1–§5 MUST have a locking test, alongside the existing
  `log_test.go` / `loss_test.go` / `seed_test.go` suites (`src/go/fab/internal/memoryindex/`).

### 8. Test matrix (REQUIRED — every row must ship as a test)

Per Constitution VII (Test Integrity) and the project rule that CLI changes ship with test updates,
the following cases MUST be implemented as Go tests (`*_test.go` under
`src/go/fab/internal/memoryindex/` and `src/go/fab/cmd/fab/`). They are enumerated here so apply and
review can verify coverage line-by-line; each maps to a requirement (`R#`) generated at planning.

| # | Behavior under test | Expectation |
|---|---------------------|-------------|
| TC1 | **Idempotence** — run freeze-on-write twice on the same git state | Second run is a byte-for-byte no-op (Constitution III) |
| TC2 | **Append on new change-id** — existing frozen log + projection containing a commit for a *new* `(file, change-id)` | Exactly one entry appended; no existing line touched |
| TC3 | **No-op on squashed-but-attributable commit** — frozen entry for change `abcd`; projection now shows `abcd`'s work under a single squashed commit that still carries the `abcd` token | No append (the `(file, change-id)` pair is already present) |
| TC4 | **Freeze of unattributable lines** — frozen log has unattributable lines; a re-run projects different (squash-reworded) unattributable subjects | Frozen lines unchanged; new unattributable commit NOT projected (the §3 rule) |
| TC5 | **`parseLog` round-trip** — `parseLog(RenderLog(entries)) == entries` for verb / bundle-rel path / summary / `(id)` token (mirrors the `parseSeedLog` round-trip in `seed_test.go`) | Faithful inverse; malformed lines degrade gracefully (no panic) |
| TC6 | **`--rebuild`** — frozen log with squash-stale lines + `--rebuild` | Re-projects from current git, dropping the now-unreachable lines (destructive, as designed) |
| TC7 | **`--check` PASS on valid superset** — committed log has frozen lines not in the live projection | Exit 0 (the case that false-fails today) |
| TC8 | **`--check` FAIL on missing attributable entry** — projection has a `(file, change-id)` the committed log lacks | Non-zero exit; report names the gap |
| TC9 | **`--check` FAIL on hand-edit** — an existing line was manually altered | Non-zero exit (single-writer discipline) |
| TC10 | **Loom regression fixture** — the squash collapses `Part 2a`/`Part 2b` → `#1721`; assert **0 churn** across the folder set (per memory `loom-runkit-memory-shape-evidence`, the canonical fixture). Build the fixture from synthesized git history (no live loom dependency in CI). | Merged log identical to the frozen input; 0 appended, 0 destroyed |
| TC11 | **Migration pre-check** — applying the re-baseline migration against an old binary that lacks `--rebuild` | Aborts with the "upgrade the binary first" message; no partial rewrite |
| TC12 | **Seed-merge preserved** — a folder with `log.seed.md` at first write / `--rebuild` | Seed entries still merge beneath the projection (no regression to `mergeSeedEntries`) |

## Affected Memory

- `pipeline/memory-index`: (modify) the `fab memory-index` behavior — add freeze-on-write generation,
  change-id append key, unattributable-freeze rule, `--rebuild`, `--check` redesign.
  <!-- assumed: the memory-index behavior lives in the `pipeline` domain (its index lists "schemas,
       preflight" and the index/log generator); exact file name to be confirmed against
       docs/memory/pipeline/index.md at hydrate. -->
- `distribution/migrations`: (modify) note the new re-baseline migration in the migrations catalog.

> Memory is hydrated post-implementation by `/fab-continue` (hydrate); the exact target files are
> resolved then. This section scopes intent, not the final file list.

## Impact

**Code (Go binary):**
- `src/go/fab/internal/memoryindex/memoryindex.go` — `gatherLogEntries`, `buildLogTarget`, `GatherLogs`:
  add read-existing-log + append-only merge; gate the unattributable branch behind a "bootstrap/--rebuild
  only" condition.
- `src/go/fab/internal/memoryindex/log.go` — add a `parseLog` (inverse of `RenderLog`) so the existing
  file can be read back into `[]LogEntry`; `RenderLog` itself stays pure.
- `src/go/fab/internal/memoryindex/seed.go` — `parseSeedLog` is the parse∘render-identity precedent;
  `parseLog` should share its entry-line grammar.
- `src/go/fab/internal/memoryindex/loss.go` — `Classify`/`LossReport`: redesign for subset/superset.
- `src/go/fab/cmd/fab/memory_index.go` — add `--rebuild` flag; rework the write/`--check` loop and
  `emitCheckReport`.

**Distribution:**
- `src/kit/migrations/{2.5.5-to-NEXT}.md` — new re-baseline migration.

**Docs:**
- `docs/specs/fkf.md` §6; `src/kit/skills/_cli-fab.md` § `fab memory-index`.

**Behavioral / compatibility:**
- One-time churn at migration (the `--rebuild` baseline commit) — expected and bounded.
- After migration, `fab memory-index` and `--check` become stable across contributors and history
  rewrites.
- Risk area: the `parseLog` round-trip must be a faithful inverse of `RenderLog` (verb, bundle-rel
  path, summary, `(id)` token) or existing frozen lines could be misread on the next run. The
  `parseSeedLog` precedent de-risks this (it already round-trips the identical grammar).

## Open Questions

- Exact dedup behavior for an **attributable** change whose `.status.yaml` `summary:` is *reworded*
  after its entry was frozen: keep the frozen text (pure append-only, never update in place) or
  update-in-place for a still-live change? Leaning keep-frozen (simplest, matches the immutability
  rule) — to be settled at planning. This does not affect the loom regression (those entries are
  unattributable).
- Whether `--check` should distinguish "missing because not-yet-regenerated" (benign tier-1, as today
  for index drift per memory `memory-index-date-drift-after-ship`) from "missing because hand-deleted"
  (destructive). Resolve during the `loss.go` redesign.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Freeze-on-write: existing log.md is authoritative & write-once; regen appends only | User-confirmed; prototyped to 0 churn across 42 loom folders; matches the existing log.seed.md frozen-store discipline | S:95 R:80 A:90 D:95 |
| 2 | Certain | Append/dedup key = change-id `(file-base, change-id)`, NOT commit-id | User explicitly chose change-id after commit-id rejected (squash+branch-delete makes hash unreachable) | S:95 R:75 A:95 D:100 |
| 3 | Certain | Unattributable commits frozen-not-reprojected (strategy S6); future migration/reorg commits leave no log trace | User-selected option with the tradeoff stated; the exact rule that produced 0 churn in the prototype | S:90 R:70 A:90 D:90 |
| 4 | Certain | Add `--rebuild` (destructive re-project) flag; do NOT add `--first-generation` | User-confirmed decision; bootstrap is the first append into an empty log, no special mode | S:90 R:85 A:90 D:95 |
| 5 | Certain | Redesign `--check` from byte-equality to subset/superset (no-missing, no-hand-edit) | Required by freeze-on-write — byte-equality false-fails on legitimately-frozen logs | S:85 R:65 A:85 D:80 |
| 6 | Certain | Ship a re-baseline migration (2.5.5 → next): `fab memory-index --rebuild` + commit | User-selected migration approach; FKF cutover (2.4.2-to-2.5.0) is the format precedent | S:90 R:80 A:90 D:90 |
| 7 | Certain | Full fix in one change (generation + key + flag + check + migration + docs/tests) | User-selected scope; coherent unit, constitution requires the docs/test updates alongside the CLI change | S:90 R:70 A:85 D:90 |
| 8 | Confident | `parseLog` (inverse of RenderLog) shares parseSeedLog's entry grammar | seed.go already implements the identical round-trip; strong codebase signal | S:80 R:70 A:90 D:85 |
| 9 | Tentative | Reworded summary on an already-frozen attributable change keeps the frozen text (no in-place update) | Simplest, matches immutability; multiple valid options — settle at planning | S:60 R:60 A:65 D:55 |
| 10 | Tentative | Affected memory lives in `pipeline/memory-index` (exact filename TBD at hydrate) | pipeline domain index points to it; filename unverified against docs/memory/pipeline/index.md | S:65 R:75 A:60 D:60 |

10 assumptions (8 certain, 1 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
