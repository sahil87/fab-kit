# Intake: Operator activates change pointer at spawn for existing changes

**Change**: 260617-5xnx-operator-spawn-activate-pointer
**Created**: 2026-06-17

## Origin

> Operator spawn sequence should activate the change pointer (fab change switch) in the newly created worktree whenever the change folder/intake already exists — so an operator-spawned worktree is self-describing after the pipeline completes, instead of leaving `.fab-status.yaml` unset.

Initiated conversationally (multi-turn). The gap was diagnosed empirically while inspecting a real completed worktree:

- A worktree at `/home/sahil/code/wvrdz/loom.worktrees/rhqy-repofile-rename-cleanup` had its change `260617-rhqy-…` fully complete (`review-pr: done`, all six stages `done`), yet `fab preflight` failed with `No active change (multiple changes exist — use /fab-switch)`.
- Investigation confirmed `.fab-status.yaml` **did not exist at all** in that worktree (not dangling — never created). `.fab-status.yaml` is gitignored (`.fab-*`) and per-checkout, so a freshly-created worktree starts with no pointer.
- Root cause: the operator started that agent via `/fab-fff rhqy` (the §6 "Existing change" entry form). That path deliberately relies on (a) the worktree branch already matching the change folder name (created by `wt create`) and (b) `/fab-fff`'s `<change>` argument being a **transient** override that never writes `.fab-status.yaml`. So the pipeline ran correctly but the worktree was left pointer-less.
- The user's instinct was `/fab-switch rhqy && /fab-fff`, but `&&`-chained slash commands have no chaining semantics and MUST NOT be sent (already documented in the skill + SPEC Resolved Design Decision #5). The correct fix is to add an explicit `fab change switch` step to the spawn sequence, scoped to spawns where the change folder already exists.

Decisions reached in conversation (all carried into Assumptions below): scope to **all spawn paths where the change folder/intake already exists** (functionally just the Existing-change form today); **guard the switch on folder existence** — never attempt a switch when the folder doesn't exist yet; the raw-text and backlog forms go through `/fab-new` which creates **and activates** the change *inside the spawned agent*, so the operator must not (and cannot) switch at spawn for those.

## Why

1. **Problem (the pain point)**: After an operator-driven pipeline completes on an existing change, the spawned worktree has no active-change pointer. Any human who later `cd`s into that worktree and runs a bare `fab`/`/fab-*` command gets `No active change (multiple changes exist — use /fab-switch)` and must remember to pass the change name to every follow-up command (`/fab-archive <change>`, `/fab-status <change>`, re-runs). The worktree looks orphaned even though it cleanly finished its change.

2. **Consequence if not fixed**: Every operator-spawned worktree for an existing change is left in this state. The most natural human follow-up after a pipeline — archiving the completed change — is the highest-friction case, since it now requires naming the change explicitly. This is an ergonomic papercut that recurs on every operator spawn of an existing change.

3. **Why this approach over alternatives**:
   - **Set the pointer at spawn (chosen)**: each operator worktree is a *dedicated, single-change checkout* with its *own* `.fab-status.yaml`, so setting the pointer there carries **zero cross-tab collision risk** — the very concern the transient-override path was protecting against (parallel tabs targeting different changes via one shared `.fab-status.yaml`) does not apply within a single dedicated worktree. The override remains correct for resolution; we simply *also* set the pointer so the worktree is self-describing.
   - **Leave as-is / rely on the `<change>` arg (rejected)**: zero new writes, but pushes friction onto every human follow-up. Rejected as the inferior ergonomics the user explicitly flagged.
   - The skill already treats switching as the right move in the adjacent path: §3 Pre-Send Validation item 3 says, when sending to an *existing pane* whose active change is wrong, "send `/fab-switch <change>` first." The asymmetry — switch on the existing-pane path but not the spawn path — is exactly the gap. This change makes the spawn path consistent with that established stance.

## What Changes

### 1. §6 "Spawning an Agent" — add a pointer-activation step (existence-guarded)

The §6 spawn sequence currently runs: (1) establish target repo → (2) `wt create` → (3) resolve dependencies → (4) `fab spawn-command --repo` → (5) open agent tab → (6) enroll in monitored set. None of these sets `.fab-status.yaml`.

Add an **existence-guarded activation** between worktree creation and opening the agent tab (i.e., after step 2 / alongside dependency resolution, before step 5). The activation runs **in the new worktree's directory** and only when the change folder already exists:

```sh
# In the newly created worktree directory, only when the change already exists.
# `fab resolve --folder <change>` succeeds iff a non-archived change folder matches.
if fab resolve --folder "<change>" >/dev/null 2>&1; then
  fab change switch "<change>"   # writes this worktree's own .fab-status.yaml
fi
```

Key properties of the step:
- **Existence guard is mandatory.** When the change folder/intake does not exist yet (the raw-text and backlog entry forms, before `/fab-new` runs inside the spawned agent), the operator MUST NOT attempt a switch — there is nothing to switch to, and `/fab-new` will create+activate the change itself once the agent runs.
- **Scoped to the new worktree.** The `fab change switch` runs with the just-created worktree as CWD, so it writes *that worktree's* `.fab-status.yaml` — never the operator's own checkout or any other worktree. This is why there is no cross-tab collision: each worktree owns its own pointer file.
- **Fail-soft.** A `fab change switch` failure is non-fatal to the spawn — log one line and continue opening the agent tab. The transient `<change>` override on the embedded pipeline command still makes the pipeline resolve correctly even if the pointer write failed; the activation is an ergonomic enhancement, not a correctness prerequisite.

### 2. §6 "Working a Change" entry-form table — note the activation on the Existing-change row

The Existing-change row currently reads (paraphrased): "`/fab-fff <change>` … The change-name override targets the change directly, no `/fab-switch` needed; the worktree's branch already matches …". Update this row's note to reflect that the operator now **also activates the pointer at spawn** for self-describing worktrees, while keeping the existing statement that the override (not the pointer) is what targets the pipeline. The raw-text and backlog rows explicitly note that activation is owned by `/fab-new` inside the spawned agent (no operator switch at spawn).

The `&&`-chaining prohibition stays exactly as-is — the fix is a separate `fab change switch` step in the spawn sequence, NOT a chained slash command sent to the pane.

### 3. SPEC mirror — `docs/specs/skills/SPEC-fab-operator.md`

Per the constitution ("Changes to skill files MUST update the corresponding `docs/specs/SPEC-*.md` file"), mirror the behavior into the SPEC:
- §6 / "Coordination Patterns" section description: note the existence-guarded pointer activation in the spawn sequence.
- Add a **Resolved Design Decision** (next number, currently #12) capturing: the gap (pointer-less completed worktrees), the chosen fix (activate at spawn when the folder exists), the no-collision rationale (dedicated single-change worktree owns its own `.fab-status.yaml`), the existence guard (raw-text/backlog forms defer to `/fab-new`'s own activation), and the rejected alternative (leave-as-is / rely on the `<change>` arg). Cross-reference Decision #5 (the `/git-branch`-removal / `&&`-has-no-chaining lineage) since it is the nearest related decision.

## Affected Memory

- `runtime/operator`: (modify) The operator coordination memory file documents the spawn sequence and the Working-a-Change entry forms. Add the existence-guarded pointer-activation step (spawn-time `fab change switch` scoped to the new worktree, guarded on folder existence, fail-soft) and the no-cross-tab-collision rationale. Hydrate stage will apply this.

## Impact

- **`src/kit/skills/fab-operator.md`** — canonical skill source. §6 "Spawning an Agent" sequence (add the guarded activation step) and §6 "Working a Change" entry-form table (Existing-change row note; raw-text/backlog rows note `/fab-new`-owned activation). This is the only behavioral change.
- **`docs/specs/skills/SPEC-fab-operator.md`** — SPEC mirror (§6 description + new Resolved Design Decision #12). Required by the constitution.
- **`docs/memory/runtime/operator.md`** — updated at hydrate (post-implementation record).
- **No Go/binary change.** The fix is pure skill prose; it composes existing CLI verbs (`fab resolve --folder`, `fab change switch`) already used elsewhere in the skill. No new command signatures, so `_cli-fab.md` is unaffected.
- **No migration.** No restructuring of user data (config, `.status.yaml`, archive layout). `.fab-status.yaml` is per-worktree and already created/removed by existing verbs.
- **Deployed copy** `.claude/skills/fab-operator.md` is regenerated by `fab sync` — not hand-edited (gitignored).

## Open Questions

(None — scope, guard, and surface were all resolved in the originating conversation.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Edit canonical `src/kit/skills/fab-operator.md`, never the deployed `.claude/skills/` copy | Constitution + context.md: `src/kit/` is canonical; `.claude/skills/` is gitignored, regenerated by `fab sync` | S:95 R:90 A:100 D:100 |
| 2 | Certain | Update the SPEC mirror `docs/specs/skills/SPEC-fab-operator.md` | Constitution: skill-file changes MUST update the corresponding SPEC-*.md | S:95 R:85 A:100 D:100 |
| 3 | Certain | Scope to spawns where the change folder/intake already exists; guard the switch on existence | User directive verbatim ("you can switch to a fab-change when its folder/intake doesn't exist. Just check that") | S:95 R:80 A:95 D:95 |
| 4 | Certain | Raw-text + backlog forms do NOT get an operator switch at spawn | They route through `/fab-new` which creates+activates inside the spawned agent (fab-new Step 10); folder doesn't exist at spawn time | S:90 R:80 A:100 D:95 |
| 5 | Confident | Activation runs in the new worktree's CWD, writing that worktree's own `.fab-status.yaml` | Per-worktree pointer file is the no-collision guarantee the parallel-workflow override was protecting; established by the diagnosis | S:85 R:80 A:90 D:85 |
| 6 | Confident | `fab change switch` failure at spawn is non-fatal (log + continue) | The transient `<change>` override still resolves the pipeline; activation is ergonomic, not a correctness prerequisite; matches the skill's fail-soft pattern for window-rename steps | S:80 R:85 A:85 D:80 |
| 7 | Confident | Place the new step between `wt create` (step 2) and opening the agent tab (step 5) | The pointer must exist before the agent runs so resolution and human follow-up both see it; aligns with where dependency resolution already sits | S:80 R:75 A:85 D:80 |
| 8 | Confident | Use `fab resolve --folder <change>` as the existence check | It is the skill's existing pure-query verb for change resolution; non-zero exit cleanly signals "no such change" | S:80 R:85 A:85 D:80 |
| 9 | Confident | Add the new SPEC Resolved Design Decision as #12, cross-referencing #5 | #5 is the nearest related decision (the `&&`-no-chaining / `/git-branch`-removal lineage); SPEC currently ends at #11 | S:85 R:90 A:85 D:80 |
| 10 | Confident | Keep the `&&`-chaining prohibition unchanged | The fix is a separate spawn-sequence step, not a chained slash command; `&&` has no slash-command semantics (already documented) | S:90 R:85 A:90 D:85 |

10 assumptions (4 certain, 6 confident, 0 tentative, 0 unresolved).
