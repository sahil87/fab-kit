# Intake: Operator Hand-over to HexoKit (Plan Step F1)

**Change**: 261113-915b-operator-hexokit-handover
**Created**: 2026-11-13

## Origin

> Plan step F1 (Hand over: deprecate fab operator, stop seeding the operator tick cron entry,
> /fab-operator becomes a pointer to HexoKit's chat skill, migration file) from the plan at
> /home/sahil/code/sahil87/hexokit.worktrees/arcing-fjord/fab/plans/sahil/26-11-10-operator-to-hexokit.md
> — this plan does not exist in fab-kit, read it at that absolute path (Rules, Decisions, and the
> F1 brief) first. HexoKit run-kit v3.22.13 (R1-R5: rk operator track/adopt/check/brief/run loop,
> rk skill operator) is released; delegate to it when present, otherwise keep today's behaviour.
> Do not run fab sync, brew upgrade, or restart anything on the live server; C1 does that with the
> user.

One-shot `/fab-new` invocation, written by the hexokit operator executing the multi-repo plan
"Operator to HexoKit — check-then-run loop, notes as memory, autopilot in Go" (2026-11-10). The plan
file lives only in the hexokit checkout at the absolute path above. Read the plan's **Rules**,
**Decisions already taken**, and the **F1** brief before planning. The plan's Decisions table is
closed: do not reopen it at intake or apply.

**F1 brief, verbatim from the plan:**

> - `fab operator track|tick-start|clock …` print a deprecation line and delegate to the matching
>   `rk operator` verb when rk ≥ R5's release is present; otherwise behave as today.
> - `fab operator clock sync` and the reconcile seed stop creating the `operator tick` entry when rk
>   ≥ R5's release is present.
> - `/fab-operator` (project skill and the `~/` pointer written by `fab sync`) becomes a short
>   pointer to HexoKit's chat skill.
> - Migration file in `src/kit/migrations/` describing the hand-over (`rk operator adopt`, cron
>   entry switch). The binary does no data moves; rk adopts.
> - Update `_cli-fab-operator.md`, memory and specs (`docs/specs/operator.md`, `skills.md`) per the
>   CLI ⇒ docs + tests rule.

**Plan rules that bind this change:**
- Rule 1: the live operator runs on the **installed** fab and rk binaries until step C1. The apply
  stage MUST NOT run `fab sync`, `brew upgrade`, or restart the rk daemon. It MUST NOT touch the
  live tmux server's cron file or operator state. Build and test only in this worktree
  (`go test` with stubbed seams).
- The plan's later step **F2** deletes fab's operator code, skill, `_cli-fab-operator.md` and the
  user-level pointer template after one release cycle. F1 keeps everything working and only
  redirects. Do not delete operator code here.
- Plan Non-goal: "Running the operator without rk." Even so, the user's instruction for F1 is that
  without a capable rk, fab behaves exactly as today.

## Why

**Problem.** The operator is one long-lived chat session that rk cron types `operator tick` into
every 3–30 min. Each tick re-reads the whole chat history. Measured: ≈ $451 over 6.2 days, ≈ 90 %
of it cached-history re-reads, and 76 % of ticks reported "no change". HexoKit (run-kit) v3.22.13
now ships the replacement:
- its own operator state engine: `rk operator track add|update|observe|rm|list|note|pause|resume`
  and `rk operator adopt`;
- a deterministic `rk operator check`;
- a run brief (`rk operator brief`);
- a daemon-owned `operator loop` cron entry (check, then a one-shot headless run only on change);
- a chat skill (`rk skill operator`).

fab still owns the old path:
- it seeds and reconciles the typed `operator tick` cron entry (`operator_clock*.go`);
- it keeps its own tracked list at `$XDG_STATE_HOME/fab/operator/<slug>.yaml`;
- it ships the ≈ 108KB `/fab-operator` skill, which is loaded into context on every operator chat.

**Consequence of not doing F1.**
1. Two operators fight over the clock. rk v3.22.13's `SeedOperatorLoop` (rk
   `internal/cron/seed.go`) mutes, never deletes, fab's legacy typed entry when rk's state file
   exists. fab's mute reconcile (`reconcileOperatorMute` in `fab operator clock sync`, and the
   edge-triggered `syncOperatorClock` after every `fab operator track` mutation) clears that mute
   whenever fab's own file has a live tracked item. The legacy tick then fires into a chat beside
   rk's loop. That is the double-fire the cutover must avoid.
2. `rk operator run` still types `/fab-operator` as the operator window's kickoff. Without F1 that
   loads the 108KB fab skill instead of HexoKit's chat skill.
3. Cutover step C1 needs a written migration (adopt, cron switch, rollback) shipped with the kit.

**Why this approach.** The plan's Ownership decision is that HexoKit owns the engine, loop, brief
and chat skill, and fab provides plain CLI facts and actions only. F1 is the reversible middle step:
- delegate when a capable rk is present;
- otherwise fall back to today's code unchanged;
- keep all code for one release so F2 can delete it.

Rejected alternatives:
- Deleting fab's operator now: that is F2, user-gated, after one release cycle.
- Having the fab binary move data into rk's file: rejected by the brief ("The binary does no data
  moves; rk adopts").
- A version-string comparison against `rk --version`: fab's existing rk gating uses capability
  probes (`rkOperatorPath`'s help-token probe in `operator.go`), and `_preamble` § HexoKit (rk)
  Reference prescribes capability-probing.

## What Changes

### 1. Capability probe: "rk ≥ R5's release is present"

Add one injectable, side-effect-free probe seam beside `rkOperatorPath` in
`src/go/fab/cmd/fab/operator.go` (or a new `operator_handover.go`), e.g.
`rkOperatorHandover func() (path string, ok bool)`:
- `exec.LookPath("rk")` first; absent → `false`, silently.
- Run `rk operator --help` bounded by a short timeout (reuse the `rkCronTimeout`-style 5s bound).
  Answer `true` only when the help lists **both** the `check` and `brief` subcommands. Match them
  as Available-Commands rows, i.e. a line whose first token is `check` or `brief`, not a bare
  substring.
- Any error or non-match → `false`, silently. Never print, never change an exit code.
- Memoize per process: one probe per fab invocation.

Why `check` and `brief`: R1 (`track`, `adopt`) shipped in rk **v3.22.12**. R2 (`check`), R3, R4
(`brief`, `rk skill operator`) and R5 (the loop, hidden `run-exec`) all shipped together in
**v3.22.13** (verified: run-kit commits a4c7fa55, efd0fa7f and bb347322 are first contained in tag
v3.22.13; 93dcbba4 in v3.22.12). So `track`/`adopt` alone would wrongly detect v3.22.12, which has
no loop. `check` + `brief` are public verbs present exactly from the R5 release. The hidden
`run-exec` verb is deliberately not probed: it is an internal wrapper.

Tests stub the seam, matching the `rkOperatorPath` / `rkCronRunner` precedent. Add one table test
of the help-text matcher against captured help text from v3.22.12 (expect false) and v3.22.13
(expect true).

### 2. `fab operator track|tick-start|clock …` deprecate and delegate

When the probe is true:
- print one deprecation line to **stderr**, e.g.
  `fab operator track is deprecated — delegating to 'rk operator track' (HexoKit owns the operator; see the 2.29.0 migration)`;
- run the matching rk verb with stdin/stdout/stderr inherited, argv-only (never a shell string);
- exit with rk's exit code.

When the probe is false: today's code path, byte-for-byte, with no deprecation line.

Verb mapping. Every pair must be verified against both `--help` outputs at apply. The current
surfaces were captured at intake: fab 2.28.16, rk v3.22.13.

| fab invocation | Delegates to | Notes |
|---|---|---|
| `fab operator track add <id> [flags]` | `rk operator track add <id> [flags]` | Flag names are identical (`--kind --probe --argv --fields --check-every --done-when --then --depends-on --scope --text --mode` + pane flags). Kinds are identical (`pane github-pr linear slack shell task note`). Pass through verbatim. |
| `fab operator track list [flags]` | `rk operator track list [flags]` | fab's output-format flag must be mapped to rk's `--json` if they differ; verify at apply. |
| `fab operator track rm <id>` | `rk operator track rm <id>` | rk also removes the item's notes. |
| `fab operator track observe <id> --json/--error/--seen` | `rk operator track observe <id> …` | Same flags. |
| `fab operator track update <id> --pause` | `rk operator track pause <id>` | fab's flags become rk verbs. |
| `fab operator track update <id> --resume` | `rk operator track resume <id>` | |
| `fab operator track update <id> --text <t>` (note kind) | `rk operator track note <id> <t>` | Semantics shift: fab replaced the note text, rk appends a note (notes are append-only). Say so in the deprecation line or the `_cli` doc. |
| `fab operator track update <id> [other flags]` | `rk operator track update <id> [other flags]` | A mixed invocation (e.g. `--pause --then x`) runs `update` first, then `pause`/`resume`/`note`; the first non-zero exit stops the sequence. |
| `fab operator track clock …` | no rk verb | Print the deprecation and a pointer: tune the loop with `rk cron edit`/`rk cron mute --for` on the `operator loop` entry. Exit non-zero (1). Write nothing. |
| `fab operator tick-start [--diff] [--quiet]` | `rk operator check` | Output shape differs: rk prints its own report, not fab's `deltas/candidates` YAML. No consumer remains once `/fab-operator` is a pointer. fab's `--quiet` is dropped (rk's global `--quiet` means something else). `tick_count`/`last_tick_at` are not written. |
| `fab operator clock sync` | no rk operator verb | See §3. |

Unchanged in F1:
- `fab operator state`, `fab operator branch-map …` and `fab operator time` keep today's behaviour
  in both arms. The deprecation line covers them too, but they are not delegated; they read/write
  fab's file only, which `rk operator adopt` reads.
- Bare `fab operator` (the launcher) already delegates to `rk operator` when capable
  (`rkOperatorPath`). Unchanged.
- `fab agent operator -o yaml [--headless]` is a fab touchpoint HexoKit keeps (plan Decisions).
  It MUST NOT be deprecated.


### 3. Stop fab's clock side effects when the probe is true

With the probe true, fab MUST perform **no write** to rk's cron file through any path:
- no `rk cron add` seed (`seedOperatorCronRow` / `ensureOperatorCronRow`);
- no `rk cron mute` / `mute --off` (`syncOperatorClock`, `muteOperatorClockIfUntracked`,
  `reconcileOperatorMute`);
- no `rk cron edit` (`reconcileOperatorSchedule`, clock-override apply/expire).

Stopping the seed alone is insufficient. A remaining unmute path re-enables the legacy entry rk
muted (see Why §1). Gate this once, at the shared entry points, so every caller is covered:
`ensureOperatorCronRow`, `syncOperatorClock`, `muteOperatorClockIfUntracked`,
`reconcileOperatorMute`, `reconcileOperatorSchedule`. The delegated verbs never reach fab's state
save anyway, but the fallback-arm verbs (`state`, `branch-map`) and any internal caller must not
either.

`fab operator clock sync` with the probe true:
- deprecation line on stderr;
- no fab-side reconcile;
- run `rk cron list` (inherited stdio), where rk's `operator loop` row (target `operator loop`) and
  the muted legacy row are visible;
- exit with rk's code.

With the probe false: unchanged. Seed-if-missing, two-way mute reconcile, schedule apply, and the
five-key YAML doc, exactly as today.

The pre-adopt window is accepted. With a capable rk but no rk state file yet, fab no longer seeds,
and rk has not seeded its loop (rk seeds only once its state file exists). A server whose legacy
entry already exists keeps it, untouched by fab. A brand-new server has no clock until the first
`rk operator track …` / `rk operator adopt` creates rk's file. That is the cutover's job (C1), and
the migration documents it.

### 4. `/fab-operator` becomes a short pointer

**Project skill `src/kit/skills/fab-operator.md`.** Replace the body with a short pointer, roughly
40–80 lines. Keep the name `fab-operator`, because `rk operator run` types `/fab-operator` as its
kickoff. Keep a trigger-worthy `description` (the user-level pointer copies it via
`skillDescription`), updated to say it hands off to HexoKit's operator. The pointer:
1. Runs the gate `command -v rk >/dev/null 2>&1`, then `rk skill operator`. Success means exit 0
   and non-empty output whose first heading names the operator topic (`# HexoKit skill: operator`).
   Older rk prints the core bundle or errors; treat any other output as a failed probe.
2. Probe succeeds: follow that output as the operator chat skill. HexoKit's chat skill owns
   everything from here. Add one hand-over hint: if fab's operator state file exists
   (`fab operator state` path) and `rk operator track list` lists nothing, suggest that the user
   run `rk operator adopt` (the cutover step). The pointer never runs adopt itself.
3. Probe fails (rk absent or older than v3.22.13): read the **legacy operator skill** and follow
   it, i.e. today's behaviour.

**Legacy body relocation.** Move today's full `fab-operator.md` body verbatim into an internal,
non-user-invocable kit partial, e.g. `src/kit/skills/_operator-legacy.md`, with frontmatter
`user-invocable: false`, `disable-model-invocation: true`, `metadata.internal: true`, and the
original `helpers:` list carried so its helpers still load. The pointer reads it from
`.agents/skills/_operator-legacy/SKILL.md`, or `$(fab kit-path)/skills/_operator-legacy.md` when the
deployed copy is absent (`_preamble` § Skill Helper Declaration's fallback rule). F2 deletes it.
Verify at apply that `fab sync` deploys `_`-prefixed partials, and check for tests or allowlists
that enumerate kit skills or helper names. Candidates: `src/go/fab-kit/internal/skills.go`, the
preamble's Allowed-values list (it governs `helpers:` declarations, not in-body reads), and any
skill-inventory test. Update them.

**User-level pointer template `src/kit/templates/user-skill-fab-operator.md`.** It is written by
`fab sync` to `~/.agents/skills/fab-operator/SKILL.md` (always) and
`~/.claude/skills/fab-operator/SKILL.md` (when `claude` is on PATH). Rewrite it so its first step is
the same rk probe:
1. `rk skill operator` succeeds → follow it.
2. Otherwise → today's two steps unchanged: the project's deployed `fab-operator` skill when inside
   a fab project, else `$(fab kit-path)/skills/fab-operator.md`, with the flat-kit-copy rule.

Both branches now resolve through the new pointer, which in turn falls back to the legacy partial.
Keep the "Generated by `fab sync` — do not edit" header. The template keeps `{DESCRIPTION}`
substitution. `setupcheck` (`ProbeUserSkills`) compares the description line, so a changed
description stays consistent automatically. Verify that no test pins the old template body.

**Do not run `fab sync` in this worktree's live environment** (plan Rule 1). Verification uses unit
tests over the template/deploy code with a temp `$HOME`.

### 5. Migration file

Add `src/kit/migrations/<from>-to-<to>.md`. The version slot is a release-owned placeholder,
following the `2.27.0-to-2.28.0.md` precedent's note. Current fab is 2.28.16. A minor bump
(`2.28.x-to-2.29.0`) fits a deprecation. Use the existing migration file shape: `# Migration`,
`## Summary`, `## Pre-check`, `## Changes`, `## Verification`. Contents:
- **Summary.** HexoKit (rk ≥ v3.22.13) owns the operator: state, check, loop, run brief, chat
  skill. With a capable rk, `fab operator track|tick-start|clock` deprecate and delegate, fab stops
  seeding/reconciling the typed `operator tick` entry, and `/fab-operator` points at
  `rk skill operator`. Without one, nothing changes. **This migration performs no file edits and
  the binary moves no data.** fab's `$XDG_STATE_HOME/fab/operator/<slug>.yaml` is never modified.
- **Pre-check.** `command -v rk` and `rk operator --help` lists `check` and `brief`. If rk is
  absent or older, this migration is a no-op.
- **Changes: hand-over steps, per tmux server, done with the user.**
  1. Stop the old operator chat, or let it idle. It still has the old skill in context, and its
     `fab operator track`/`tick-start` calls now delegate to rk.
  2. Upgrade fab and rk.
  3. `rk operator adopt --dry-run`, then `rk operator adopt`. This imports tracked items and
     branch_map, is idempotent, skips ids already present, and leaves fab's file read-only.
     Compare Operator Tasks before and after.
  4. Cron entry switch. On its next tick after rk's state file exists, the rk daemon seeds the
     target-less `operator loop` entry and **mutes, never deletes**, the legacy `operator tick`
     entry. Confirm with `rk cron list`.
  5. Open the new chat: `rk operator` (kickoff `/fab-operator` now resolves to
     `rk skill operator`).
  6. Watch one wake → check → run → note cycle (Cron Log).
- **Rollback.** Mute the `operator loop` entry and unmute the legacy entry (`rk cron mute <id>`,
  `rk cron mute <id> --off`). fab's state file was never modified. Pinning the old fab/rk versions
  restores the old skill.
- **Verification.** `rk cron list` shows exactly one unmuted operator clock (the loop).
  `fab operator clock sync` prints the deprecation line and does not unmute the legacy entry.

The migration MUST NOT cite fab-kit-only paths (`docs/specs/*`, `docs/memory/*`, `src/go/*`) or the
hexokit plan path (Constitution V: deployed content). Name rk commands and behaviour in prose.

### 6. Docs and tests (CLI ⇒ docs + tests rule)

- `src/kit/skills/_cli-fab-operator.md`:
  - document the deprecation and delegation table, the capability probe, the clock-side-effect
    gate, and the `track clock` / `clock sync` rk-arm behaviour;
  - state the fallback arm is unchanged;
  - keep `fab agent` sections untouched.
- `src/kit/skills/_preamble.md` § Context Loading names "`/fab-operator` loads a reduced 3-file
  set". Update it if the pointer's context-loading override changes. The pointer itself should load
  nothing beyond the probe. The legacy partial keeps the old reduced set.
- `src/kit/skills/fab-help.md`, `fab_help.go`, and anything that lists `/fab-operator`'s
  description: keep consistent.
- `docs/specs/operator.md` and `docs/specs/skills.md`: these are human-curated specs (Constitution
  VI). Edit only the operator hand-over intent the brief names: delegation, pointer, legacy
  fallback, F2 deletion pending.
- Tests:
  - probe matcher (v3.22.12 vs v3.22.13 help text);
  - each delegated verb builds the right rk argv, including `update` splitting and the
    `track clock` refusal;
  - clock side-effect gate: no `rk cron add/mute/edit` argv recorded through `rkCronRunner` when
    the probe is true, including the tracked-set case that previously issued `mute --off`;
  - fallback arm unchanged (existing tests stay green with the probe stubbed false);
  - user-level pointer template renders with the rk-first step.
- Constitution Principle V says the user-level pointer "points at `$(fab kit-path)/skills/` and
  carries no copied prose". Still true for the fallback branch. If the wording becomes inaccurate,
  a wording-only amendment comment (no version bump, matching the udwv/ga0v precedent) is in scope.
  Flag it rather than silently editing.

## Affected Memory

- `runtime/operator`: (modify) hand-over:
  - capability probe (`rk operator --help` lists `check` + `brief`);
  - deprecate+delegate mapping for `track|tick-start|clock`;
  - fab clock side effects (seed/mute/edit) disabled under a capable rk;
  - `/fab-operator` is a pointer to `rk skill operator` with the `_operator-legacy` fallback;
  - rk owns the loop entry and mutes the legacy typed tick.
- `distribution/kit-architecture`: (modify) the user-level `fab-operator` pointer template now
  probes `rk skill operator` first; new internal `_operator-legacy` partial.
- `distribution/migrations`: (modify) record the hand-over migration (no file edits, no data move;
  adopt + cron switch + rollback documented).
- `_shared/context-loading`: (modify) only if `/fab-operator`'s reduced context-loading override
  moves to the legacy partial.

## Impact

- **Go (`src/go/fab/cmd/fab/`)**:
  - `operator.go`: probe seam, deprecation helper;
  - `operator_track.go`: delegation in each verb's RunE, or a pre-dispatch wrapper on the `track`
    parent;
  - `operator_tick_start.go`: delegate to `rk operator check`;
  - `operator_clock.go`: gate at the shared entry points;
  - `operator_clock_sync.go`: rk arm;
  - tests in the matching `_test.go` files.
- **Go (`src/go/fab-kit/internal/skills.go`, `src/go/fab/internal/setupcheck/`)**: verify the
  user-level pointer deploy/check still works with the new template and description. Likely no
  logic change.
- **Kit content**:
  - `src/kit/skills/fab-operator.md`: rewritten as a pointer;
  - `src/kit/skills/_operator-legacy.md`: new, verbatim old body;
  - `src/kit/templates/user-skill-fab-operator.md`;
  - `src/kit/skills/_cli-fab-operator.md`;
  - possibly `_preamble.md`, `fab-help.md`;
  - new migration file.
- **Docs**: `docs/specs/operator.md`, `docs/specs/skills.md`, memory per Affected Memory.
- **External dependency (runtime only)**: rk ≥ v3.22.13's `rk operator {track,check,brief}`,
  `rk skill operator`, and `rk cron list`. All of it is optional: absent or older rk falls back.
- **Not touched**:
  - `fab agent`, `fab pane`, `fab dispatch`, `fab status set-base-branch`;
  - the operator Go code's fallback behaviour;
  - fab's state file format;
  - the live tmux server, installed binaries, and deployed skills (no `fab sync`, `brew upgrade`,
    or daemon restart).

## Open Questions

- Exact name and location of the legacy partial (`_operator-legacy`), pending what `fab sync`'s
  deploy logic and any kit-inventory tests accept.
- Whether fab's `track list` output flag maps cleanly onto rk's `--json`. Resolve by diffing both
  `--help` outputs at apply.
- Whether Constitution Principle V's pointer sentence needs a wording-only touch.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Plan Decisions are closed. F1 deletes nothing (F2 does). No `fab sync`, `brew upgrade` or rk daemon restart during apply | Plan Rules 1 and 4 plus the user's explicit instruction | S:95 R:85 A:95 D:95 |
| 2 | Certain | Without a capable rk every `fab operator` verb, the clock reconcile, and the skill behave exactly as today | User: "delegate to it when present, otherwise keep today's behaviour"; brief: "otherwise behave as today" | S:95 R:80 A:90 D:90 |
| 3 | Confident | "rk ≥ R5 present" = capability probe: `rk operator --help` lists both `check` and `brief`. Not a version-string compare | R1 shipped in v3.22.12 without the loop, while R2/R4/R5 all landed in v3.22.13 (verified by tag containment). Matches the `rkOperatorPath` help-token precedent and `_preamble`'s capability-probe rule | S:70 R:85 A:80 D:70 |
| 4 | Certain | Under a capable rk, fab performs no `rk cron add/mute/edit` through any path, not just no seed | rk's `SeedOperatorLoop` mutes the legacy entry. fab's `reconcileOperatorMute`/`syncOperatorClock` would `mute --off` it on a tracked set and cause a double fire | S:80 R:80 A:90 D:85 |
| 5 | Certain | Delegation prints one deprecation line on stderr, runs rk argv-only with inherited stdio, and exits with rk's code | Keeps stdout machine-readable for scripts. argv-only matches fab's rk-call convention | S:75 R:90 A:80 D:80 |
| 6 | Confident | Verb mapping per the §2 table: `update --pause/--resume/--text` split into rk `pause/resume/note`; `tick-start [--diff]` → `rk operator check` | Captured from both `--help` surfaces (fab 2.28.16, rk v3.22.13). rk splits these into verbs and has append-only notes | S:70 R:85 A:75 D:65 |
| 7 | Confident | `fab operator track clock` under a capable rk refuses (exit 1) with a pointer to `rk cron edit/mute --for` on the `operator loop` entry | No rk operator equivalent exists. A passthrough could silently edit the wrong entry | S:45 R:85 A:55 D:45 |
| 8 | Confident | `fab operator clock sync` under a capable rk: deprecation line plus passthrough to `rk cron list`, no fab reconcile | The brief says "delegate to the matching verb", but none exists for clock. Listing shows the loop and the muted legacy row | S:45 R:85 A:55 D:45 |
| 9 | Confident | Legacy skill body moves verbatim to an internal `_operator-legacy` partial that the pointer reads when `rk skill operator` is unavailable | Must keep today's behaviour without rk, but a 108KB skill must not load on every chat. Name and location to verify against sync deploy rules | S:55 R:75 A:55 D:45 |
| 10 | Confident | `/fab-operator` keeps its name. Both project skill and user-level pointer probe `rk skill operator` first (topic heading check), then fall back | `rk operator run` types `/fab-operator` as kickoff. Older rk prints the core bundle, hence the heading check | S:75 R:85 A:80 D:70 |
| 11 | Confident | `fab operator state`, `branch-map` and `time` get the deprecation line but are not delegated. `fab agent operator` is not deprecated | The brief names only track/tick-start/clock for delegation. `fab agent operator -o yaml` is a touchpoint HexoKit keeps | S:65 R:90 A:75 D:65 |
| 12 | Certain | Migration is a no-edit, no-data-move document in the release-owned version slot (placeholder `2.28.x-to-2.29.0`), restating the C1 steps, the rk loop seeding/mute behaviour, and rollback without citing fab-kit-only or hexokit paths | The brief says "the binary does no data moves; rk adopts". Constitution V. Follows the `2.27.0-to-2.28.0.md` precedent | S:80 R:90 A:85 D:75 |
| 13 | Confident | Pre-adopt window accepted: under a capable rk with no rk state file, fab stops seeding and leaves any existing legacy entry untouched. The migration sequences adopt immediately after upgrade | rk seeds its loop only once its state file exists. C1 is user-gated and handles the order | S:65 R:80 A:75 D:70 |

13 assumptions (5 certain, 8 confident, 0 tentative, 0 unresolved).
