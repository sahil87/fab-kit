# Intake: Pull-based artifact state — `fab status refresh` + self-healing transitions, drop the artifact-write hook

**Change**: 260702-y022-status-refresh-drop-artifact-hook
**Created**: 2026-07-02

## Origin

This change was designed in a planning discussion (user-approved decisions, encoded below as
Certain/Confident assumptions). It is **change 3a of a four-part series (3a–3d)** that together
enable **cross-harness stage dispatch** — running a post-intake pipeline stage (e.g. apply) on a
non-Claude agent CLI such as `codex`. The series:

- **3a (this change)** — make hook-owned `.status.yaml` state **pull-based**: add `fab status
  refresh`, wire self-healing refresh into the transition commands, and remove the `artifact-write`
  PostToolUse hook. This is the prerequisite: a non-Claude worker will write `intake.md`/`plan.md`
  without firing Claude Code's PostToolUse hook, so the artifact-derived state must be recomputed on
  read/transition rather than on write.
- **3b** — per-tier `spawn_command` in `agent.tiers` (each tier can name its own agent launcher).
- **3c** — a `fab dispatch` process-manager command family writing to a repo-root
  `.fab-dispatch/{change-id}/` directory.
- **3d** — a harness-adapters spec plus the CLI dispatch branch in the skill dispatch seam, and the
  place to state the governing **"hooks may enhance, never own"** principle in spec/constitution prose.

Recently merged related work this builds on: **PR #455** (`fab config reference`, shipped as the
`2.9.2-to-2.10.0` migration) and **PR #456** (`spawn_command` `{model}`/`{effort}` placeholders).

Governing principle established in this change (stated here, promoted to spec/constitution prose in
3d): **hooks may enhance, never own** — no correctness-critical state may live behind a Claude Code
hook, because a hook fires only in the Claude harness. Runtime telemetry (liveness/enrollment) is
push-by-nature and MAY stay in a hook; artifact-derived pipeline state (change type, intake
confidence, plan counts) is correctness-critical and MUST be pull-based.

## Why

**Problem.** Four `.status.yaml` fields are today written *only* by the `artifact-write` PostToolUse
hook (`src/go/fab/cmd/fab/hook.go`, `artifactBookkeeping`), which fires on Write/Edit of
`fab/changes/*/intake.md` and `plan.md`:

1. `change_type` — inferred from `intake.md` keywords (unless `change_type_source: explicit`).
2. `confidence.*` + `confidence.score` — recomputed from `intake.md` via `score.ComputeWithStatus`.
3. `plan.generated`, `plan.task_count` — counted from `plan.md` `## Tasks`.
4. `plan.acceptance_count`, `plan.acceptance_completed` — counted from `plan.md` `## Acceptance`.

Because the hook is the sole writer, any edit that does **not** fire the Claude Code PostToolUse hook
leaves these fields stale:

- a `sed`-style edit or a direct file write (the hook only fires on the Claude Code `Write`/`Edit`
  tools) — a documented wart today;
- **a non-Claude agent (codex, etc.) writing the artifact** — the blocking prerequisite for the 3a–3d
  cross-harness dispatch series. A codex worker that writes `plan.md` will never fire the hook, so
  `plan.task_count`/`plan.generated` would silently stay at their pre-write values.

**Consequence if unfixed.** Cross-harness dispatch (3b–3d) cannot land safely: a stage run on
another CLI would produce artifacts whose derived `.status.yaml` state is wrong, and downstream
readers/gates would act on stale data. Even within Claude today, sed/direct edits produce silent
drift.

**Why this approach (pull-based) over alternatives.** Two of the four field groups are *already*
pull-based on the read path that matters: `status.LiveAcceptance(changeDir)`
(`src/go/fab/internal/status/acceptance.go`) derives acceptance done/total from `plan.md`
`## Acceptance` checkboxes at **read** time for `fab preflight`, `fab pr-meta`, and `fab status
plan`, falling back to the cached counter only when `plan.md`/its section is absent. That precedent
(schemas.md calls the counter a "write-time cache") is exactly the model to extend: derive on demand,
cache opportunistically. The remaining fields — `change_type` and `confidence.*` — are **not**
read-time-derived today (nothing recomputes them on read); they are the load-bearing gap. A pull at
transition seams (preflight, advance, finish) recomputes all four from on-disk artifacts, so drift
can exist only *transiently mid-stage*, where nothing reads those fields. Self-healing at the seams
means no skill ever has to remember to call refresh, and it fixes the existing sed/non-Claude-editor
warts as a side effect.

Removing the hook (rather than keeping it as a redundant belt-and-braces layer) is deliberate: a hook
that "owns" correctness-critical state is precisely the anti-pattern 3a exists to eliminate, and
keeping it would leave two writers for the same fields (drift risk + confusion about the source of
truth).

## What Changes

### 1. New `fab status refresh <change>` command

Recomputes the artifact-derived `.status.yaml` fields from on-disk artifacts. It **MUST reuse the
existing `artifactBookkeeping` logic** in `src/go/fab/cmd/fab/hook.go` — extract it to a shared
internal package (see §2), do **not** reimplement. Behavior:

- Resolve the change (4-char ID / folder substring / full folder name, like every `fab status`
  subcommand), acquire the `.status.yaml` flock via `lockfile.WithLock` (same as `withStatusLock` in
  `src/go/fab/cmd/fab/status.go`), load the `StatusFile` once, mutate in memory, Save exactly once if
  dirty — the single-load/single-Save discipline the hook already follows (the `mz4q F02` pattern).
- **intake.md branch**: if `intake.md` exists, recompute `change_type` (via
  `hooklib.InferChangeType`) **only when `change_type_source` is absent or `inferred`** — when it is
  `explicit` (a human ran `fab status set-change-type`), keep the explicit type and do NOT overwrite
  (this is the existing jznd guard — see §5). Recompute `confidence.*`/`score` via
  `score.ComputeWithStatus`.
- **plan.md branch**: if `plan.md` exists, set `plan.generated=true` when `## Tasks` is present, and
  recount `task_count`, `acceptance_count`, `acceptance_completed` (section-bounded checkbox counts),
  each guarded by `HasSectionHeading` exactly as the hook does today (missing section ⇒ leave that
  field untouched, do not zero a valid value).
- **Missing-artifact tolerance**: refresh MUST be a safe no-op when an artifact is absent (e.g. no
  `plan.md` yet at intake stage) — it recomputes only what it can read. It never errors on a missing
  artifact; it exits non-zero only on a genuine failure (unresolvable change, unreadable/corrupt
  `.status.yaml`).
- **Idempotent** (Constitution III): running refresh twice with unchanged artifacts produces the same
  `.status.yaml` and no spurious `dirty` write.

Signature to document in `_cli-fab.md`:

```
fab status refresh <change>   # recompute change_type + confidence (from intake.md) and
                              # plan.generated/task_count/acceptance_count/acceptance_completed
                              # (from plan.md) from on-disk artifacts; respects change_type_source: explicit
```

### 2. Extract `artifactBookkeeping` to a shared internal package

`artifactBookkeeping(fabRoot, filePath, match, statusFile) ([]string, bool)` currently lives in
`src/go/fab/cmd/fab/hook.go` (the `main` package) and is called only by `hookArtifactWriteCmd`. Move
the recompute logic into a shared internal package so both the (to-be-removed) hook path — during the
one-release-no-op window if we keep it (see Open Questions) — and the new `refresh` command and the
self-healing transition wiring can call it. Candidate home: a new `internal/refresh` package (or fold
into `internal/status`), exposing something like:

```go
// Refresh recomputes artifact-derived fields on the in-memory StatusFile from
// on-disk intake.md/plan.md under the change dir. Returns whether anything was
// mutated (caller owns the single Save under the held lock). Respects
// change_type_source: explicit.
func Refresh(fabRoot, changeDir string, sf *statusfile.StatusFile) (dirty bool, err error)
```

The current `artifactBookkeeping` keys off `match.Artifact` (which single file was just written); the
shared `Refresh` instead inspects **both** artifacts on disk (intake.md AND plan.md) since a
transition-time refresh is not scoped to a single write. The per-write `additionalContext`
"Bookkeeping: ..." string is a hook-only affordance and is dropped (no agent consumes it at a
transition seam).

Reused helpers stay where they are (they have other live callers — do NOT delete):
`hooklib.InferChangeType`, `hooklib.HasSectionHeading`, `hooklib.CountSectionItemsBounded`,
`hooklib.CountCompletedSectionItemsBounded` (the last three are also used by
`status.LiveAcceptance`), `score.ComputeWithStatus` (used by `score.Compute`),
`status.ApplyChangeType`/`ApplyAcceptance`, and the `statusfile` `SourceExplicit` machinery.

### 3. Self-healing transitions

`fab preflight`, `fab status advance`, and `fab status finish` internally run the same `Refresh`, so
no skill has to remember to call it.

- **`fab status advance` / `fab status finish`** (`src/go/fab/cmd/fab/status.go`): both route through
  `withStatusLock`, which already holds the flock and hands the loaded `StatusFile` + `statusPath` +
  `fabRoot` to a closure. Call `Refresh(fabRoot, changeDir, st)` inside that closure **before** the
  `status.Advance`/`status.Finish` mutation, so the recompute and the transition persist in the same
  locked load/Save. Note: `advance`'s closure currently discards `fabRoot` (`_ string`) — it must
  capture it (and derive `changeDir`) to call Refresh. `finish`'s closure already receives `fabRoot`.
- **`fab preflight`** (`src/go/fab/cmd/fab/preflight.go` → `internal/preflight`): preflight is
  currently a **pure reader** (loads `.status.yaml`, never mutates/Saves; derives live acceptance via
  `LiveAcceptance`). Self-healing means preflight must now recompute + persist. Route the refresh
  through the same locked load-mutate-save (via `lockfile.WithLock`) — either in the `preflight`
  RunE before calling the read-only `preflight.Run`, or by threading a refresh into `preflight.Run`.
  This is a deliberate change to preflight's read-only posture; keep the read-only *derivation* (the
  YAML formatting) unchanged and add only the pre-read refresh. Because `LiveAcceptance` already makes
  acceptance counts correct-on-read for preflight's output, the load-bearing part preflight gains from
  refresh is `change_type` + `confidence` self-healing (and persisting the plan counts as a cache).

**Which transitions do NOT get refresh**, and why: `start`, `reset`, `skip`, `fail` are not artifact
generation seams — they move stage pointers without an artifact write preceding them. Refresh at
`advance`/`finish` (the forward seams) + `preflight` (read/orient seam) covers every point where a
just-written artifact must be reflected before the next stage reads it. (Decide during apply whether
`start` also warrants refresh for symmetry — graded below; the recommendation is no, to keep the
surface minimal and because `advance`/`preflight` already cover the reachable-stale windows.)

### 4. Remove the `artifact-write` hook

Two mechanisms deploy/own the hook registration:

- **`fab hook sync`** (`hookSyncCmd` in `hook.go` → `hooklib.Sync` in
  `src/go/fab/internal/hooklib/sync.go`) builds registrations from `hooklib.DefaultMappings`, which
  contains the two `artifact-write` PostToolUse rows (one `Write` matcher, one `Edit` matcher). Remove
  those two rows from `DefaultMappings` so sync stops (re-)registering them. Also drop the
  `on-artifact-write.sh` entry from `oldScriptToSubcommand` (the legacy-script migration map). The
  same registration table is **replicated in the `fab-kit` binary** (`src/go/fab-kit/internal/hooksync.go`) — sweep it in lockstep.
- **The `fab hook artifact-write` subcommand** itself (`hookArtifactWriteCmd` + its `AddCommand`
  registration in `hookCmd`) — see Open Questions for delete-now vs. keep-one-release-as-no-op.

- **Migration** (`src/kit/migrations/2.10.0-to-2.11.0.md`, next slot after the shipped
  `2.9.2-to-2.10.0`; bumps `src/kit/VERSION` 2.10.0 → 2.11.0): sentinel-guarded, idempotent, removes
  the existing PostToolUse `fab hook artifact-write` entries from a project's
  `.claude/settings.local.json`. This restructures user-owned data, so it MUST ship as a migration
  (constitution + code-quality rule), not an ad-hoc script. Follow the settings.local.json-editing
  precedent in `src/kit/migrations/0.46.0-to-1.1.0.md` §1 (read → parse `hooks.PostToolUse` array →
  drop entries whose `command` is `fab hook artifact-write` for both matchers → write back; print a
  `Skipped: ...already up to date` line when none present). Keep the three session-hook entries
  untouched. Match the `2.9.2-to-2.10.0` file format: plain markdown, `## Summary` / `## Pre-check` /
  `## Changes` / `## Verification`; atomic write (temp file + rename); complete no-op on re-run.

### 5. `change_type` re-trip gotcha — already closed; preserve in the refresh path

The originally-flagged gotcha (the hook re-inferring `change_type` on every intake write and
clobbering a manual `set-change-type`) is **already fixed in code** by the jznd change (260615):
`statusfile` has a `change_type_source` field (`SourceExplicit = "explicit"`), `status.SetChangeType`
marks it `explicit`, and the hook re-infers **only** when the source is absent or `inferred`. So
**this change does not introduce a new marker** — it must **preserve the existing `explicit` guard**
in the new `Refresh` path: `Refresh` recomputes `change_type` only when `change_type_source` is absent
or `inferred`, and skips inference (keeping the explicit type) when it is `explicit`. This is a
behavior-parity requirement on the extracted code, not a new field. The stale skill caveat in
`_intake.md` Step 6 line 108 ("any subsequent intake edit re-fires the hook and overwrites the
override, so re-verify") is doubly wrong post-change (no hook fires, and the explicit guard already
held) and is swept in §7.

### 6. Git staging of status/history files — find a home

The artifact-write hook also does a best-effort `git add` of the change's `.status.yaml` and
`.history.jsonl` (`hook.go` after bookkeeping) so hook-driven writes never block a later git
operation. **No `fab status` mutator git-adds today** (verified: only `true_impact.go` shells out to
git in `internal/status`, unrelated) — the hook is the *sole* auto-stager of these files. Removing the
hook drops that behavior. Decide during apply where it lands (graded below):

- **(a) Drop it** — status/history files get committed at ship time by `/git-pr` regardless; the hook
  auto-stage was a convenience against transient "unstaged changes block a git op" friction, not a
  correctness guarantee. Simplest; lowest surface. **Recommended** unless apply finds a concrete
  workflow that relies on continuous staging.
- **(b) Fold into `Refresh`** — refresh already runs at the transition seams under the lock; add the
  `git add` there. Keeps parity with today's behavior but couples a pure state-recompute to git.
- **(c) Fold into the transition commands** (`advance`/`finish`) — narrower than (b), but still adds
  git side effects to state transitions.

Grade honestly via SRAD and pick one in the plan; the recommendation is (a) with a one-line rationale,
falling back to (b) if apply surfaces a real dependency.

### 7. Documentation & mirror sweep (must-fix class)

Every load-bearing claim about the artifact-write / PostToolUse hook owning `change_type`, confidence,
or plan counts, and about hook git-staging, must be swept — canonical skill + its SPEC mirror + any
aggregate spec + the memory files — in this same change (Constitution Additional Constraints;
code-quality § Sibling & Mirror Sweeps; reviewers read the "skill change ⇒ SPEC mirror" rule
strictly). Enumerated surface (grep-verified):

**Canonical skills (`src/kit/skills/`)** and their SPEC mirrors:

- `_intake.md` Step 6 (lines 105/107/108: "The PostToolUse intake-write hook owns `change_type`…";
  the stale re-fire caveat) ⇒ mirror `docs/specs/skills/SPEC-_intake.md` (lines 5/61/94:
  "verify hook-owned `change_type`", "intake-write hook set it in Step 5", "the intake-write hook owns
  `change_type`"). Rewrite Step 6 to: change_type is recomputed by `fab status refresh` (self-healed
  at transition seams); verify from `.status.yaml`; override via `set-change-type` (sticky `explicit`).
- `_generation.md` Plan Generation step 8 (lines 152–157: "The PostToolUse hook updates `.status.yaml`
  `plan.generated`…") and line 231 ("the PostToolUse counters") ⇒ mirror `SPEC-_generation.md`
  (lines 65/108). Rewrite to: plan counts are recomputed by `fab status refresh`, self-healed at the
  next `advance`/`finish`/`preflight`; skills need not call `set-acceptance` at generation time.
- `fab-continue.md` line 123 ("The PostToolUse hook updates `plan.generated`…") ⇒ mirror
  `SPEC-fab-continue.md` line 198.
- `_cli-fab.md` (canonical CLI reference) — **CLI change ⇒ this file + tests are mandatory**:
  - line 115 (`set-change-type` row referencing "the PostToolUse intake-write hook stops re-inferring")
  - line 313 ("the four event handlers") ⇒ three
  - line 320 (hook table `artifact-write` row) ⇒ remove
  - line 323 + lines 325–327 (the `artifact-write` bookkeeping sub-section + git-staging sentence) ⇒
    remove/relocate; **add the new `fab status refresh` entry** to the `fab status` command family.
  - mirror `SPEC-_cli-fab.md` line 26 ("PostToolUse artifact hooks, etc.").
- `fab-new.md` — mirror `SPEC-fab-new.md` line 5 ("the PostToolUse intake-write hook owns
  `change_type`; the skill overrides via `set-change-type` only if wrong") ⇒ reword (verify Step 6 in
  the source `fab-new.md`/`_intake.md` and sweep whatever carries the claim).
- Bookkeeping-table rows in SPEC mirrors that name the PostToolUse hook: `SPEC-_pipeline.md` line 90,
  `SPEC-fab-ff.md` line 34 ("PostToolUse hook recomputes plan counts…"). Check whether the canonical
  `_pipeline.md`/`fab-ff.md` carry the phrase and sweep in lockstep (SPEC mirror rule; whole class).

**Aggregate specs:**

- `docs/specs/architecture.md` line 178 ("…or the `artifact-write` hook — skills never hand-edit").
- `docs/specs/templates.md` lines 30/66/70/71 (status.yaml header comment; `change_type` field;
  plan-count field; confidence field — all attribute writes to the artifact-write hook).
- `docs/specs/change-types.md` line 84 ("the `artifact-write` hook does this automatically on every
  `intake.md` write").
- `docs/specs/skills/SPEC-hooks.md` — the primary spec to rewrite substantially: `## Summary`
  ("four event handlers"), the Registered-Hooks table (`artifact-write` PostToolUse rows), the entire
  `## artifact-write Bookkeeping` section, the git-staging side-effect line, "the hook owns them
  mechanically", the `## Event Coverage` PostToolUse row, and the four→three count (lines 5/74).

**Memory files (Affected Memory, all `(modify)`):** see next section.

### 8. Tests (ship in the same change — test-alongside, Constitution VII)

- **New**: `fab status refresh` command tests (idempotency; respects `change_type_source: explicit`;
  intake-only present; plan-only present; both present; missing-artifact no-op; missing-section leaves
  fields untouched). Self-healing tests for `advance`/`finish`/`preflight` (a sed-simulated stale
  `.status.yaml` is healed by the transition).
- **Move/retire**: the `artifact-write bookkeeping` block in `src/go/fab/cmd/fab/hook_test.go`
  (`runArtifactWriteHook` + `TestHookArtifactWrite_*`) migrates to test the new `refresh`/shared
  package (behavior is the same; the entry point changes).
- **Update**: `src/go/fab/internal/hooklib/sync_test.go` and
  `src/go/fab-kit/internal/hooksync_test.go` — the "expect 2 PostToolUse entries" assertions and the
  `on-artifact-write.sh` migration/double-mapping tests (`TestSyncHooks_ArtifactWriteDoubleMapping`)
  drop or invert (expect the artifact-write rows to be absent). `internal/hooklib/artifact_test.go` —
  `MatchArtifactPath`/`InferChangeType` tests follow the function to its new caller; `HasSectionHeading`/
  `CountSection*` tests stay (functions stay live). `internal/status/status_test.go`
  `TestSetChangeType_MarksExplicit` stays (behavior unchanged) but its doc comment mentioning the hook
  is reworded.

## Affected Memory

Every hit below documents hook-owned state that becomes pull-based; all `(modify)`. Per-domain index
lines are updated in lockstep with their domain files (the `fab memory-index` generator flags drift
otherwise).

- `pipeline/change-lifecycle.md`: (modify) drop "Counts are maintained by the PostToolUse
  `artifactBookkeeping` hook…" (line 70); update the flock/serialized-writers list and the Single-Save
  hook-bookkeeping bullet (lines 78/79/355) to describe `fab status refresh` + the transition seams.
- `pipeline/schemas.md`: (modify) the write-time-cache framing (line 120), the "PostToolUse
  intake-write hook applies inference…" `change_type_source` guard (line 133), the failure-surfacing
  hook-caller note (line 200) — reattribute recompute to `fab status refresh`; the `change_type_source`
  guard *behavior* stays, now enforced by refresh.
- `pipeline/planning-skills.md`: (modify) the hook-owned `change_type` decision/why/introduced-by block
  (lines 84/86/386–389), the Hook-backed-bookkeeping paragraph (line 25), and the counts-flow note
  (line 20) — reattribute to refresh; the jznd sticky-explicit behavior is preserved.
- `pipeline/execution-skills.md`: (modify) the Hook-backed-bookkeeping paragraph (line 15) and the
  "PostToolUse hook updates `.status.yaml` plan block" lines (109/130/161).
- `pipeline/index.md`: (modify) the change-lifecycle / planning-skills / schemas description rows
  (lines 10/13/15) in lockstep with the domain files.
- `runtime/runtime-agents.md`: (modify) "The fourth hook, `fab hook artifact-write`…" (line 57) and the
  table row (line 66) — now three session hooks only.
- `distribution/kit-architecture.md`: (modify) the kit-tree `on-artifact-write.sh` line (83), the
  `fab hook artifact-write` bullet (319), and the `internal/hooklib` capability/testing paragraphs
  (332/336).
- `distribution/setup.md`: (modify) the hook-registration row (line 114) if it enumerates the
  artifact-write hook.
- `memory-docs/templates.md`: (modify) the PostToolUse `artifactBookkeeping` counts paragraph (line 88).
- `pipeline/hooks-may-enhance-never-own.md` **or a new memory file**: (new) capture the governing
  principle + the pull-based state model (decide domain/name during hydrate; likely `pipeline/`).

## Impact

- **Go binary** (`src/go/fab/`): new `internal/refresh` (or `internal/status`) `Refresh`;
  `cmd/fab/status.go` (`refresh` subcommand + advance/finish self-heal); `cmd/fab/hook.go` (remove
  `hookArtifactWriteCmd` + its registration, or no-op it one release; `artifactBookkeeping` moves out);
  `cmd/fab/preflight.go` + `internal/preflight` (self-heal); `internal/hooklib/sync.go`
  (`DefaultMappings`, `oldScriptToSubcommand`). `src/go/fab-kit/internal/hooksync.go` (replicated
  registration table). Tests across all touched packages.
- **Kit content** (`src/kit/`): new migration `2.10.0-to-2.11.0.md`; bump `src/kit/VERSION` → 2.11.0;
  `skills/_cli-fab.md` + the swept skill files.
- **Specs** (`docs/specs/`): SPEC mirrors of every swept skill + `SPEC-hooks.md` (substantial rewrite)
  + `architecture.md`, `templates.md`, `change-types.md`.
- **Memory** (`docs/memory/`): the Affected-Memory set above (hydrate stage).
- **User data**: existing projects' `.claude/settings.local.json` (migration removes the two
  artifact-write PostToolUse entries; three session hooks untouched).
- **No `.status.yaml` schema change** — `change_type_source` already exists; refresh only changes *who*
  writes the existing derived fields, not the schema.

## Open Questions

- Delete `fab hook artifact-write` from the binary immediately, or keep it one release as a no-op for
  stale deployed settings that still reference it? (Graded below — recommendation: delete now; a
  registered command that does nothing is harmless if invoked, and the migration removes the
  registration, so a "no-op shim" guards only a hand-edited settings file that the migration would
  also fix. Confirm during apply that an unknown `fab hook <x>` subcommand fails gracefully — cobra
  errors non-zero, but the hook contract is exit-0/swallow; if an unregistered subcommand could noisily
  fail a PostToolUse event on an un-migrated project, keep the no-op shim one release.)
- Should `fab status start` also self-heal for symmetry, or is `advance`/`finish`/`preflight` the
  complete seam set? (Graded below — recommendation: no; start is not an artifact-generation seam.)
- Where does the git-staging of `.status.yaml`/`.history.jsonl` go — drop / fold into refresh / fold
  into transitions? (§6; graded below — recommendation: drop.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add `fab status refresh <change>` recomputing `change_type`+confidence (from intake.md) and plan.generated/task_count/acceptance counts (from plan.md) from on-disk artifacts | Discussed — user approved the new command and its exact recompute surface; mirrors existing `artifactBookkeeping` fields verbatim | S:95 R:80 A:95 D:90 |
| 2 | Certain | `refresh` MUST reuse `artifactBookkeeping` by extracting it to a shared internal package — no reimplementation | Discussed — user mandated reuse-not-reimplement; code-quality forbids duplicating existing utilities | S:95 R:75 A:95 D:90 |
| 3 | Certain | Self-heal by running refresh inside `fab preflight`, `fab status advance`, `fab status finish` | Discussed — user approved these three seams so no skill must remember to call refresh | S:95 R:75 A:90 D:90 |
| 4 | Certain | Remove the `artifact-write` hook: drop its `DefaultMappings` rows (both binaries) so `fab hook sync` stops registering it, and ship a migration removing existing settings.local.json entries | Discussed — user approved removal + migration; user-data restructuring MUST be a migration (constitution/code-quality) | S:95 R:60 A:95 D:90 |
| 5 | Certain | Keep the three session hooks (session-start/stop/user-prompt) untouched — push-by-nature runtime telemetry | Discussed — user approved; they degrade gracefully and don't own correctness-critical state | S:95 R:90 A:95 D:95 |
| 6 | Certain | State the "hooks may enhance, never own" principle in this intake (promoted to spec/constitution prose in 3d) | Discussed — user approved as the governing principle for the 3a–3d series | S:90 R:90 A:90 D:90 |
| 7 | Certain | Migration is the next slot `2.10.0-to-2.11.0` and bumps `src/kit/VERSION` 2.10.0 → 2.11.0 | Verified `src/kit/VERSION`=2.10.0, last migration `2.9.2-to-2.10.0`; migrations.md range/versioning model is deterministic | S:90 R:80 A:100 D:95 |
| 8 | Certain | Follow `0.46.0-to-1.1.0.md` §1 (parse hooks section, match `command`, rewrite) as the settings.local.json-editing migration precedent; match `2.9.2-to-2.10.0` file format (sentinel-guarded, idempotent, atomic write) | Verified both precedents; the repo's established migration shape | S:90 R:80 A:100 D:95 |
| 9 | Certain | Sweep the full mirror class: every swept `src/kit/skills/*.md` + its `SPEC-*.md`, aggregate specs (architecture/templates/change-types/SPEC-hooks), `_cli-fab.md`+tests, and the enumerated memory files | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps (recurring must-fix); grep-verified the enumerated surface | S:90 R:65 A:95 D:90 |
| 10 | Certain | The `change_type` re-trip gotcha is already fixed by the jznd `change_type_source: explicit` guard; this change preserves that guard in the refresh path (no new field) | Verified in code: statusfile `SourceExplicit`, `SetChangeType` marks explicit, hook skips inference when explicit. The originally-flagged "add a marker" is already done | S:95 R:75 A:100 D:90 |
| 11 | Certain | Ship all Go changes with tests in the same change; migrate the `artifact-write` hook_test bookkeeping tests to cover the shared `refresh` path; update sync_test/hooksync_test PostToolUse-count assertions | Constitution VII + code-quality test-alongside (mandated); the test surface is grep-enumerated and mechanically follows the code move | S:80 R:70 A:90 D:85 |
| 12 | Confident | The extracted shared `Refresh` inspects BOTH intake.md and plan.md on disk (not scoped to one written file like the hook's `match.Artifact`), since a transition-time refresh is not tied to a single write; drop the hook-only `additionalContext` output | Design follows from the transition-seam use; no agent consumes additionalContext at a transition; low blast radius, one obvious shape | S:70 R:75 A:80 D:75 |
| 13 | Confident | `preflight`'s self-heal is a deliberate move from pure-reader to load-mutate-save under the flock; keep the read-only YAML derivation unchanged and add only the pre-read refresh | Verified preflight is currently a pure reader; adding a locked refresh is the minimal structural change; `LiveAcceptance` already covers acceptance-on-read so refresh's load-bearing gain is change_type+confidence | S:75 R:70 A:80 D:75 |
| 14 | Confident | Do NOT add self-heal to `start`/`reset`/`skip`/`fail` — only `advance`/`finish`/`preflight` (forward + orient seams) | These are not artifact-generation seams; advance/preflight already cover every reachable-stale window; minimal-surface principle | S:70 R:75 A:80 D:70 |
| 15 | Tentative | Git-staging of `.status.yaml`/`.history.jsonl` (today done only by the hook): DROP it — status/history are committed at ship by `/git-pr`; the auto-stage was convenience not correctness | Verified the hook is the sole auto-stager, but the choice among drop / fold-into-refresh / fold-into-transitions is a genuine open design call (three valid options, front-runner not locked) — decide in the plan; low Signal (design not yet fixed) and low Disambiguation (multiple valid options), reversible either way | S:40 R:55 A:50 D:35 |
| 16 | Tentative | Delete `fab hook artifact-write` from the binary immediately rather than keeping a one-release no-op shim | Genuinely undecided pending an apply-time check: whether an unregistered `fab hook` subcommand fails gracefully on an un-migrated project (cobra errors non-zero, but the hook contract is exit-0/swallow). If it could noisily fail a PostToolUse event, keep the shim one release. Low Signal/Disambiguation until that check runs; reversible | S:40 R:55 A:45 D:35 |

16 assumptions (11 certain, 3 confident, 2 tentative, 0 unresolved). Run /fab-clarify to review.
