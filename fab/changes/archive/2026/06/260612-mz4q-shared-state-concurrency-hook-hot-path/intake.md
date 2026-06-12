# Intake: Shared-State Concurrency & Hook Hot Path

**Change**: 260612-mz4q-shared-state-concurrency-hook-hot-path
**Created**: 2026-06-12

## Origin

> /fab-new mz4q

One-shot invocation from backlog ID `[mz4q]` — **Binary-review batch B1/6** (wave 1), filed 2026-06-12 from the adversarially-verified Go binary review. The backlog entry encodes findings **F01–F08** of `docs/specs/findings/binary-review-2026-06-12.md` §B1 (line numbers vs commit `1431a9c3`). This intake incorporates the report's adversarial-verifier corrections verbatim — they are binding design constraints, not optional commentary.

**Parallelism context**: wave 1 runs alongside `k4ge`, `dn2c`, `pw3k`. Seam with `k4ge` on `src/go/fab/internal/status/status.go`: k4ge owns `lookupTransition`/`AllowedStates`; **this change owns the Save/SetAcceptance paths** — same-file-different-function; whichever merges second rebases.

## Why

1. **The pain point**: fab's three shared state files are all mutated via *unlocked load-modify-save cycles* from processes that by design run concurrently (Claude Code hooks firing from multiple sessions in one worktree, operator-driven `fab status` in other panes, parallel review sub-agents editing `plan.md`):
   - `.fab-runtime.yaml` — all four mutators (`WriteAgent`, `ClearAgent`, `ClearAgentIdle`, `GCIfDue`, runtime.go:183–345) race: A loads `{}`, B loads `{}`, A saves `{A}`, B saves `{B}` — A's agent entry is silently lost. The temp+rename in `SaveFile` prevents torn files but **not lost updates** (the docs at `runtime/runtime-agents.md:75` conflate the two).
   - `.status.yaml` — the artifact-write hook holds a stale snapshot across up to **4 separate full-document Saves** (hook.go:281–296) while `fab status start/finish` may run in another pane; last-writer-wins over the whole document including the progress map. It also never `Sync()`s before rename, so a crash can leave an empty `.status.yaml` — the pipeline state machine's source of truth.
   - `.fab-status.yaml` — the active-pointer swap is `os.Remove` + `os.Symlink` (non-atomic; a concurrent reader in the gap sees no pointer), and a dangling pointer is trusted without validating the target, producing wrong answers (`fab resolve` prints the stale folder) or misleading errors.
2. **The consequence of not fixing**: lost agent entries block `fab pane send` or let the operator inject keystrokes mid-turn (stale `idle_since`); silently reverted stage transitions corrupt unattended `fab-fff` pipelines; per-edit hook overhead (2 YAML parses + 1–2 fsync'd writes of the runtime file, plus 4× serialize-and-rename of `.status.yaml`) sits on every interaction's latency path; misclassified read errors ("not found" for permission/parse failures) route agents down wrong recovery paths.
3. **Why this approach**: one small advisory-flock helper wrapping every load-mutate-save cycle fixes the lost-update class at its root (rather than per-call-site patches); splitting mutation from persistence fixes the write amplification at its root (one Save per hook event); temp-symlink+rename matches the project's established atomic-write convention; truthful error classification and no-silent-drop writes follow constitution Principle III ("MUST NOT corrupt or lose data").

## What Changes

All paths relative to `src/go/fab/` unless noted. Line numbers vs commit `1431a9c3`.

### F01 — Cross-process locking for `.fab-runtime.yaml` (correctness, medium effort)

Add one advisory-lock helper: open a sibling lock file (`.fab-runtime.yaml.lock`) with `os.OpenFile(O_CREATE)`, `syscall.Flock(fd, LOCK_EX)`, then load/mutate/save/unlock. `flock` works on both supported GOOS (linux + darwin, per the `proc` build tags); no Windows concern (`pipeline/change-lifecycle.md:252` records that explicitly).

- Wrap all four `.fab-runtime.yaml` mutators: `WriteAgent` (runtime.go:183–199), `ClearAgent` (:203–231), `ClearAgentIdle` (:239–273), `GCIfDue` (:304–345).
- Hook call sites invoking them: hook.go:122–123, 147–148, 170–171.
- **Verifier corrections (binding)**: the realistic race surface is multiple Claude sessions in the *same* worktree (operator-spawned agents each get their own worktree and never contend) plus the cross-process GC clobber; concrete failure modes are a lost `ClearAgentIdle` (operator injects keystrokes into a busy agent) and a lost `WriteAgent` (idle agent unmatched, `pane send` blocked until next hook fire). The new `.lock` sibling needs a `.gitignore` entry. `runtime/runtime-agents.md:75` must be updated per docs-are-source-of-truth (it currently claims atomicity "even under concurrent hook invocations" — true only for torn writes, not lost updates).

### F02 — Batch the artifact-write hook's redundant saves and reads (performance, small effort)

`fab hook artifact-write` fires on **every** PostToolUse Write/Edit of `plan.md` or `intake.md` (registered in hooklib/sync.go:24–25 — every checkbox tick during apply).

- **plan.md branch**: `status.SetAcceptance` is called up to 4 times (hook.go:281, 287, 295, 296), each ending in a full `yaml.Marshal` + `CreateTemp` + rename (status.go:325). Split mutation from persistence: make `SetAcceptance`/`SetChangeType` mutate the in-memory `StatusFile` only (or add non-saving variants), and have `artifactBookkeeping` call `statusFile.Save` **exactly once** at the end.
- **intake.md branch**: the change folder is resolved 3× (hook.go:204 `resolve.ToFolder`, score.go:130 `resolve.ToAbsDir`, log.go:53 `ConfidenceLog`→`ToAbsDir` — each an `os.ReadDir` over `fab/changes`), `.status.yaml` is loaded twice (hook.go:211, score.go:145) and saved twice, `intake.md` read twice (hook.go:256, score.go:228–233). Add a `score.ComputeWithStatus(fabRoot, changeDir, statusFile, statusPath)` entry point that reuses the already-resolved folder and already-loaded `StatusFile`; pass the resolved folder to `log.ConfidenceLog` instead of re-resolving (`filepath.Join` suffices for known-exact folder names).
- **Verifier corrections (binding)**: non-saving variants must keep validate-before-mutate (`SetChangeType` validates at status.go:280–291). External contract preserved: `additionalContext` JSON shape, final `.status.yaml` state, and the `fab status set-acceptance` CLI signature are all unchanged.

### F03 — `.status.yaml` write races + durability (correctness, medium effort)

- Reuse the same flock helper for `.status.yaml` load-mutate-save cycles: take the lock in `loadStatus`-style entry points (cmd/fab/status.go:48–62) so hook bookkeeping and `fab status start/finish` in other panes serialize instead of last-writer-wins over the whole document.
- Add `tmp.Sync()` before Close in `statusfile.Save` (statusfile.go:303–313) — `.status.yaml` is the pipeline's source of truth; runtime.SaveFile already syncs (runtime.go:124).
- **Verifier corrections (binding)**: no flock helper exists anywhere today — this is *new* infrastructure (F01 builds it; F03 reuses it). Within a single session, tool calls and hooks are serialized — the realistic exposure is parallel review sub-agent dispatch (`_review.md:140–142`, editing `plan.md` in place) and cross-pane writers. The hook's 4-Save window (F02's fix) is the other half of this finding.

### F04 — Fold GC into the agent-entry mutation (performance, small effort)

Every Stop/UserPromptSubmit/SessionStart hook event does two independent load-modify-save cycles: the entry mutation, then `GCIfDue` re-stats and re-parses the whole file (runtime.go:306–310) *before* checking the 180s `last_run_gc` throttle (:316–320).

- Add `runtime.UpdateAgent(fabRoot, sessionID, mutate func(...), gcInterval)`: load once, apply the entry mutation, run the GC sweep inline when due (`pidAlive` is just `kill(pid, 0)` — no extra I/O), save once. This makes the code match the documented design (`runtime-agents.md:87`: "the GC sweep piggybacks on the same file read + write round-trip when it's actually due").
- Drop the per-write `fsync` in runtime `SaveFile` (runtime.go:124) — the file is ephemeral and fully re-derivable ("state re-populates on next hook event"), and the fsync sits on every hook event's latency path. The report frames this as "consider"; it is included here because it directly serves the backlog GOAL (minimal per-edit hook work) and is a one-line revert (Assumption 5).
- **Verifier corrections (binding)**: the merged `UpdateAgent` must keep GC-on-no-op semantics (GC runs even when the mutation half is a no-op, e.g. `ClearAgent` with no entry) and must **skip the save entirely** when neither the entry nor GC changed anything — today's write-free paths (`ClearAgent` early-return at :223–224, `ClearAgentIdle` at :264–265) stay write-free.

### F05 — Atomic `.fab-status.yaml` pointer swap (correctness, small effort)

`change.Switch` does `os.Remove` + `os.Symlink` (change.go:179–181); `Rename` does the same with the `os.Symlink` error *discarded* (change.go:159–160 — a race there permanently leaves no pointer).

- Create the symlink at a temp name in the repo root and `os.Rename(tmp, symlinkPath)` — rename atomically replaces the old link on POSIX, eliminating both the empty-pointer window and the concurrent-Switch EEXIST race. Extract a shared `setActivePointer` helper used by `Switch` and `Rename`.
- Matches established convention: `.status.yaml` (statusfile.go:286–319) and the runtime file (runtime.go:98–133) already use temp+rename.

### F06 — Classify `.status.yaml` read errors (errors-ux, small effort)

`statusfile.Load` maps **any** `os.ReadFile` failure to `status file not found: <path>` (statusfile.go:114–118) — permission-denied, is-a-directory, and I/O errors all masquerade as absence, routing agents down wrong recovery paths. `change.ListWithOptions` (change.go:276–285) repeats the conflation: YAML parse errors print `Warning: .status.yaml not found for <name>`.

- Wrap the real error (`fmt.Errorf("read status file %s: %w", path, err)`), keeping an `os.IsNotExist` special case with the current friendly text; make the change-list warning echo the actual load error so corruption (e.g. git merge-conflict markers — `.status.yaml` is git-tracked) is distinguishable from absence.
- **Verifier notes**: no skill or test depends on the current string (grep-verified). Preflight currently emits the self-contradictory `Failed to load .status.yaml: status file not found` for unreadable-but-existing files. `_cli-fab.md` needs updating only if its documented error strings change (its Common Error Messages table documents resolve.go strings, not this one).

### F07 — No silent dropped `.status.yaml` writes (correctness, medium effort)

`syncToRaw` (statusfile.go:363–411) only writes struct fields into keys that *already exist* in the parsed document — only `true_impact` has insertion logic (:408–410). On a file missing `prs:`/`stage_metrics:`/`confidence:` (restored pre-0.24.0 archives, hand-edited files — migrations never touch `fab/changes/archive/**`), `fab status add-pr` exits 0 and persists nothing. `SetProgress` (:335–345) silently no-ops when the stage key or `progress:` mapping is absent — `fab status start` then persists no transition while `stage_metrics` and `.history.jsonl` (status.go:563) *do* persist: an inconsistent state, worse than a clean no-op.

- Generalize the `insertTrueImpact`-style insertion so absent keys are created on write; make `SetProgress` create a missing stage key; return an error when the document shape is malformed (`progress:` absent or not a mapping) so callers can't report success on a dropped write.
- **Chosen variant**: write-time insertion, NOT validate-at-Load refusal — the verifier shows Load-refusal conflicts with the documented tolerance posture for un-migrated/archived files (1.9.7-to-1.10.0.md:22–27), while insertion matches established project posture (`TestLegacyChecklistFileSavesPlanBlock`, statusfile_test.go:256–259, fixed this exact bug class for `plan:`). The backlog's "error/warn" phrasing is satisfied by erroring on *malformed shape*; well-formed-but-sparse files get the write persisted via insertion.
- Skills rely on exit-0 semantics (git-pr-review.md:184–193 commits bookkeeping assuming `finish` persisted), so silent drops propagate into autonomous pipelines — constitution Principle III supports the fix.

### F08 — Validate the `.fab-status.yaml` target before trusting it (correctness, small effort)

`resolveFromCurrent` (resolve.go:125–136) returns whatever folder the symlink encodes without checking `fab/changes/<folder>/.status.yaml` exists. Dangling pointers (folder archived from another worktree, branch switch, manual delete — the symlink is gitignored and never updates with the tree) produce per-command inconsistency: `fab resolve` **succeeds with the stale folder** (silent wrong answer to agent callers), `fab preflight`/`fab score` fail with misleading messages, `fab hook` records the stale folder into runtime agent entries. `log.Command` already guards this exact case (log.go:34–36).

- In `resolveFromCurrent`, after extracting the folder name, `os.Stat` the target `.status.yaml`; on failure treat the pointer as stale (optionally remove it) and fall through to the existing no-active-change/single-change-fallback logic so the user gets the actionable `/fab-switch` guidance.
- **Verifier correction (binding, prerequisite)**: `Archive` renames the change folder into `archive/` *and then* resolves the now-dangling symlink to detect that the archived change was active and clear the pointer (archive.go:91, :102–108). The stat-and-fall-through fix MUST be preceded by reordering `Archive` to capture/clear the pointer **before** the rename (or read the link directly), else the pointer is never cleared and `fab-archive`'s `pointer:` output regresses from "cleared" to "skipped". `_cli-fab.md`'s error table documents only the symlink-absent case and needs the routine constitution-mandated update.

## Affected Memory

- `runtime/runtime-agents`: (modify) — line 75's atomicity claim ("even under concurrent hook invocations") conflates rename-atomicity with race safety; document the flock posture and the merged single load/save (F01, F04). The GC-piggyback description at :87 becomes accurate once F04 lands.
- `pipeline/change-lifecycle`: (modify) — `.fab-status.yaml` pointer semantics: atomic temp+rename swap, dangling-target validation with fall-through, archive's capture-before-rename ordering (F05, F08); `.status.yaml` locking + fsync durability posture (F03, F07).

## Impact

- **Code** (all in `src/go/fab/`): `internal/runtime/runtime.go` (lock wrap, UpdateAgent, fsync drop), new shared lock helper (small new package, e.g. `internal/lockfile`), `internal/statusfile/statusfile.go` (Sync, syncToRaw insertion, SetProgress, Load error wrapping), `internal/status/status.go` (**Save/SetAcceptance paths only — k4ge seam**), `internal/change/change.go` (setActivePointer, list warning), the archive flow (capture-pointer-before-rename reorder), `internal/resolve/resolve.go` (target validation), `internal/score/score.go` (ComputeWithStatus), `internal/log/log.go` (accept resolved folder), `cmd/fab/hook.go` (single-Save bookkeeping, single resolution).
- **Tests**: constitution requires test updates for any Go change; the lost-update, single-Save, atomic-swap, insertion, and dangling-pointer behaviors each need coverage (test paths: `**/*_test.go`).
- **Docs**: `src/kit/skills/_cli-fab.md` only where documented messages change (F06 wording, F08 error-table row). Two memory files (above) at hydrate.
- **Repo plumbing**: `.gitignore` entries for the new `.lock` siblings (`.fab-runtime.yaml.lock`; the `.status.yaml` lock sibling pattern under `fab/changes/`).
- **External contracts unchanged**: hook `additionalContext` JSON shape, all CLI command signatures, exit-0 hook semantics, final `.status.yaml` state shape.
- **Merge choreography**: second-to-merge (vs `k4ge`) rebases; their functions don't overlap.

## Open Questions

*(none — the backlog entry and verified report resolve all design decisions; remaining choices are graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly F01–F08 as filed in §B1, with all adversarial-verifier corrections treated as binding constraints | Backlog ACTIONS enumerate the findings; the report was verified against the code at `1431a9c3` this same day | S:95 R:90 A:95 D:95 |
| 2 | Certain | Go changes ship with test updates; `_cli-fab.md` touched only where documented messages change | Constitution line 31 + backlog CONSTRAINTS state both explicitly | S:95 R:90 A:100 D:100 |
| 3 | Confident | One shared flock helper in a new small package (e.g. `internal/lockfile`), sibling `.lock` files, `syscall.Flock` LOCK_EX, no build-tag split | F01 sketches `withRuntimeLock`; F03 says "reuse the same helper" — a shared package is the only home serving both runtime and statusfile; linux+darwin both support flock | S:85 R:75 A:85 D:70 |
| 4 | Confident | F07 resolved as write-time insertion (generalize `insertTrueImpact`) + error on malformed shape; NOT validate-at-Load refusal | Backlog says "error/warn", report's primary fix says insert; verifier shows insertion matches established posture (`TestLegacyChecklistFileSavesPlanBlock`) while Load-refusal conflicts with documented legacy tolerance — synthesis: persist sparse-file writes, error only on malformed shape | S:75 R:70 A:85 D:60 |
| 5 | Confident | Drop the per-write fsync in runtime `SaveFile` while adding `Sync()` to `statusfile.Save` — durability follows criticality (source-of-truth durable, ephemeral re-derivable state fast) | Report says "consider dropping"; file is documented as self-healing/re-derivable; serves the backlog GOAL (minimal per-edit hook work); one-line revert if wrong | S:55 R:90 A:85 D:60 |
| 6 | Certain | Merged `UpdateAgent` keeps GC-on-no-op semantics and skips the save when nothing changed (today's write-free paths stay write-free, preserving idempotency) | Verifier correction (ii) on F04 — explicit implementation mandate | S:90 R:80 A:90 D:90 |
| 7 | Certain | F08 lands with the Archive reorder (capture/clear pointer before folder rename) as a prerequisite in the same change | Verifier correction: without it the fix regresses `fab-archive`'s pointer-clearing | S:90 R:75 A:90 D:90 |
| 8 | Confident | Lock siblings get `.gitignore` entries; lock files live next to the file they guard | F01 verifier correction (c) mandates the gitignore entry; sibling placement is the report's stated design | S:70 R:85 A:80 D:75 |

8 assumptions (4 certain, 4 confident, 0 tentative, 0 unresolved).
