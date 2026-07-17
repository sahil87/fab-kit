# Intake: Migrate fab batch switch to wt's Explicit --checkout Contract

**Change**: 260717-otol-batch-switch-explicit-checkout
**Created**: 2026-07-17

## Origin

One-shot `/fab-new` invocation:

> fab-kit's fab batch switch (src/go/fab/cmd/fab/batch_switch.go:98) relies on wt's old wt create dual-semantics positional (an existing branch name silently checked out in place). wt 260717-2af2-explicit-base-checkout-flags (merged) made this a hard error (exit 2) unless --checkout \<branch\> is passed explicitly -- the positional now only ever creates a new branch. Update fab-kit's batch_switch.go call site to pass --checkout explicitly when targeting an existing branch, and audit any other fab-kit call sites or docs (naming conventions, _cli-external.md, etc.) that assume the old wt create positional behavior.

Grounding performed at intake: read the wt change's own intake (`~/code/sahil87/wt/fab/changes/260717-2af2-explicit-base-checkout-flags/intake.md`) for the exact new contract, swept fab-kit repo-wide for `wt create` mentions (Go call sites, kit skills, specs, memory, README/site docs), and inspected wt's `BranchExistsLocally`/`BranchExistsRemotely` implementations to mirror the probe semantics. The wt change itself anticipated this migration: *"fab-kit migration (existence probe + `--checkout`/positional routing) is a coordinated follow-up in the fab-kit repo"* and decided **hard break, no deprecation window**.

## Why

1. **The pain point**: `fab batch switch`'s entire purpose is attaching worktrees to *existing* changes — whose branches usually already exist (created by `/fab-new` Step 11 in the original checkout). Its call `wt create --non-interactive --reuse --worktree-name <match> <branch>` (batch_switch.go:98) relied on wt silently dispatching the positional on branch existence. Under the merged wt 2af2 contract, the positional is **new-branch-only**: an existing local *or remote* branch is a hard `ExitInvalidArgs` (exit 2) error pointing at `--checkout`. Once the wt release carrying 2af2 ships (installed wt v0.0.23 still has old semantics — the change is merged but unreleased), `fab batch switch` fails for every change whose branch exists but whose worktree doesn't (worktree deleted, branch kept; fresh window after `--reuse` miss), degrading to warn-and-skip with a generic message that *discards* wt's stderr fix hint (`.Output()` drops stderr).
2. **If we don't fix it**: the moment new wt releases, the common batch-switch path silently skips every existing-branch change with an unexplained `Error: failed to create worktree`. The operator docs (`_cli-external.md` § wt, `fab-operator.md` spawn steps) still teach the dual-semantics invocation, so operator-driven spawns for known changes hit the same exit 2 with no guidance.
3. **Why this approach**: mirror wt's own dispatch in fab — probe branch existence with the same checks wt makes (`git show-ref --verify --quiet refs/heads/<b>` locally, `git ls-remote --heads origin <b>` remotely) and route: exists → `--checkout <branch>`, missing → positional (new branch). This is exactly the migration shape the wt change designed for, keeps fab's routing in permanent agreement with wt's validation, and needs no wt version detection (hard-break coordination was decided upstream; both tools share one author and release channel). Alternatives rejected: try-positional-then-retry-on-exit-2 (exit-code sniffing, two subprocess rounds, scary transient stderr); wt version detection/compat shim (contradicts the upstream hard-break decision).

## What Changes

### 1. `batch_switch.go` — probe-and-route the wt create invocation

Replace the single line-98 invocation with existence-probed routing. Sketch (exact shape left to apply):

```go
branchName := branchPrefix + match

// Route per wt's 2af2 contract: positional = new-branch-only, --checkout = existing branch.
wtArgs := []string{"create", "--non-interactive", "--reuse", "--worktree-name", match}
if branchExists(branchName) {
    wtArgs = append(wtArgs, "--checkout", branchName)
} else {
    wtArgs = append(wtArgs, branchName)
}
wtOut, wtStderr, err := pane.RunCmd("wt", wtArgs...)
if err != nil {
    fmt.Fprintf(errW, "Error: failed to create worktree for '%s' (%v), skipping\n", match, pane.StderrError(err, wtStderr))
    continue
}
```

```go
// branchExists mirrors wt's own BranchExistsLocally/BranchExistsRemotely checks
// (internal/worktree/git.go in the wt repo) so fab's routing never disagrees
// with wt's positional validation.
func branchExists(branch string) bool {
    if exec.Command("git", "show-ref", "--verify", "--quiet", "refs/heads/"+branch).Run() == nil {
        return true
    }
    out, err := exec.Command("git", "ls-remote", "--heads", "origin", branch).Output()
    return err == nil && strings.TrimSpace(string(out)) != ""
}
```

Key behaviors:

- **Probe order**: local first (no network); `ls-remote` only when the branch is not local. Remote-only existing branches are the exact shared-branch danger case 2af2 closes — passing them positionally would exit 2, so the remote check is required for routing fidelity.
- **Offline degradation**: a failed `ls-remote` counts as not-remote → positional → wt itself re-checks and errors → the item is warn-and-skipped with wt's stderr surfaced. Visible, not silent.
- **`--reuse` retained on both forms**: wt's name-collision short-circuit is unchanged and ignores branch selectors entirely, so routing is irrelevant when the worktree already exists (the reuse path keeps working with zero behavior change).
- **Stderr surfacing**: switch from `exec.Command(...).Output()` (stderr discarded) to the `pane.RunCmd` + `pane.StderrError` pattern `batch_new.go:118` already uses. wt's typed exit-2 error with its fix hint is the designed migration signal — the warn-and-skip line must carry it.

### 2. `batch_new.go` — audited, no change

`batch_new.go:118` passes **no positional** (`wt create --non-interactive --worktree-name <id>`) — the exploratory-create path, explicitly unchanged by 2af2 ("`fab batch new` (no positional) is unaffected"). No code change.

### 3. `src/kit/skills/_cli-external.md` § wt — new contract + routed examples

- **Flags table**: rewrite the `[branch]` row — *"Positional — name for a NEW branch only; exits 2 if the branch already exists locally or remotely (use `--checkout`)"*. Add a `--checkout <branch>` row — *"Put the worktree on an EXISTING local/remote branch (fetches remote-only branches); conflicts with `--base` and with the positional (both exit 2)"*.
- **Example — known change** (line ~120) and **§ Operator Spawning Rules → Known change** (line ~133): the change branch may already exist (created by `/fab-new` Step 11 in another checkout), so teach probe-and-route: branch exists → `wt create --non-interactive --worktree-name <name> --checkout <change-folder-name>`; missing → today's positional form.
- **Example — autopilot respawn** (line ~121, `--reuse --worktree-name <name> <branch> --base <prev-change>`): same routing, plus **drop `--base` on the `--checkout` arm** — `--checkout`+`--base` is a hard exit-2 conflict in new wt, and `--base` is only meaningful when creating the branch (an existing branch already embodies its start-point).
- **§ New change (from backlog)**: bare `wt create --non-interactive` (no positional) is unchanged — verify wording only.

### 4. `src/kit/skills/fab-operator.md` — invocation lines

Update the literal `wt create ... <branch>` invocations to reference the routed form (delegating detail to `_cli-external.md` § wt, which the operator already loads):

- Line ~110 (idea-lookup single-match action): `wt create --non-interactive --worktree-name <name> <branch>`.
- Line ~436 (§6 spawn sequence step 2): `wt create --non-interactive --worktree-name <wt> [<branch>]`.
- Line ~539 (entry-form table, Existing-change row): the parenthetical "created by `wt create … <change-folder-name>`" phrasing.

### 5. `src/kit/skills/_cli-fab.md` § fab batch — switch bullet

The `switch` bullet ("create worktrees with branch names (applying `branch_prefix`)…") gains the routing sentence: probes branch existence (local `show-ref`, then `ls-remote --heads origin`) and passes `--checkout <branch>` for existing branches / the positional for new ones, per wt's 2af2 contract; wt failures now surface the child stderr in the warn-and-skip line.

### 6. Specs — `companions.md` + the three SPEC mirrors

- `docs/specs/companions.md` § wt: the integration paragraph ("`fab batch switch` calls `wt create` (with `--reuse`) to attach worktrees to existing changes") gains the probe-and-route description and the minimum-wt coupling note.
- SPEC mirrors (constitution: every `src/kit/skills/*.md` edit updates its mirror; treat the whole class): `docs/specs/skills/SPEC-_cli-external.md`, `SPEC-fab-operator.md`, `SPEC-_cli-fab.md`.

### 7. Audited — confirmed unaffected (no edits)

Repo-wide `wt create` sweep, remaining mentions all describe the **bare exploratory create** (no positional — unchanged by 2af2) or are generic: `README.md:270`, `docs/site/workflows.md:101`, `fabhelp.go:160` (command list), `docs/specs/naming.md:42` (worktree naming), `docs/specs/glossary.md:108`, `fab-new.md:94` / `git-branch.md:161` / `docs/memory/pipeline/change-lifecycle.md:188` ("disposable `wt create` name" rename-guard prose — the disposable branch comes from the bare create).

## Affected Memory

- `distribution/kit-architecture`: (modify) the `fab batch switch` bullet gains the probe-and-route contract + stderr surfacing; the `wt create [flags] [branch]` description (~line 384) still documents the retired dual semantics and baseWarnings ("ignored with a warning for existing local/remote branches") — correct to the 2af2 contract. (Note: this file's wt sections also still claim wt is built from `src/go/fab/cmd/wt/main.go` — stale since the 4rtx decoupling; correct the claims touched by this change, don't restructure the section.)
- `runtime/operator`: (modify) the Operator Spawning Rules pointer/known-change naming strategy ("pass the change folder name as the branch argument to `wt create`") reflects the probe-and-route form.

## Impact

- **Source**: `src/go/fab/cmd/fab/batch_switch.go` (routing + `pane.RunCmd` stderr surfacing; `batch_new.go` untouched).
- **Tests** (constitution: Go change ships tests): `src/go/fab/cmd/fab/batch_switch_test.go` — the existing PATH-shim stub infra (`stubBatchSwitchTmuxCapture`, `wt` echo stub) extends to an argv-capturing `wt` stub + a stubbed/fixtured `git` for the probe; assert: existing local branch → `--checkout` form; missing branch → positional form; probe-fail → positional form; wt failure → stderr surfaced in the warning. Installed wt (v0.0.23) still has OLD semantics — tests must rely on stubs, never the real binary.
- **Kit skills**: `_cli-external.md`, `fab-operator.md`, `_cli-fab.md` (+ SPEC mirrors `SPEC-_cli-external.md`, `SPEC-fab-operator.md`, `SPEC-_cli-fab.md`).
- **Specs**: `docs/specs/companions.md`.
- **External coupling**: the `--checkout` path requires the wt release carrying 2af2 (> v0.0.23). Old wt + migrated fab: `--checkout` is an unknown flag → wt errors → warn-and-skip with surfaced stderr (visible, recoverable by upgrading wt). New wt + unmigrated fab: exit 2 on every existing branch (the breakage this change fixes). No version detection — hard break per the upstream decision.

## Open Questions

- None — the upstream wt change record resolves the design questions (positional fate, hard-break transition, conflict matrix), and the fab-side routing mirrors decisions wt already made.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Probe-and-route mirroring wt's own dispatch: local `git show-ref --verify --quiet refs/heads/<b>`, else `git ls-remote --heads origin <b>`; exists → `--checkout <branch>` (no positional), missing → positional | wt 2af2 intake explicitly anticipated "existence probe + `--checkout`/positional routing" in fab-kit; mirroring `BranchExistsLocally/Remotely` keeps fab's routing in agreement with wt's validation | S:85 R:85 A:90 D:88 |
| 2 | Certain | `fab batch new` needs no code change | batch_new.go:118 passes no positional (exploratory create, contract unchanged); 2af2 intake states it verbatim | S:90 R:95 A:95 D:95 |
| 3 | Certain | Keep `--reuse --worktree-name` on both routed forms | wt's reuse name-collision short-circuit is unchanged and ignores branch selectors ("branch selectors are not consulted") | S:75 R:85 A:90 D:85 |
| 4 | Confident | The documented autopilot-respawn example drops `--base` on the `--checkout` arm | `--checkout`+`--base` is a hard exit-2 conflict in new wt; `--base` is only meaningful for new branches | S:60 R:80 A:85 D:75 |
| 5 | Confident | batch_switch adopts `pane.RunCmd` + `pane.StderrError` (replacing `.Output()`'s stderr discard) | wt's typed exit-2 error is the designed migration signal; batch_new already uses this exact pattern — pattern reuse, not invention | S:55 R:85 A:85 D:70 |
| 6 | Confident | No wt version detection/fallback: migrated fab requires the wt release carrying 2af2; older wt fails the `--checkout` path loudly (unknown flag → warn-and-skip with stderr) | Hard-break-now was decided in the wt change; both tools share an author and release channel; a compat shim contradicts that decision | S:70 R:75 A:85 D:80 |
| 7 | Confident | Remote probe scoped to `origin` only; failed/offline `ls-remote` degrades to not-remote → positional → wt re-checks and errors → visible warn-and-skip | Mirrors wt's own origin-only `BranchExistsRemotely`; degradation is loud, not silent, and recoverable | S:55 R:80 A:82 D:72 |
| 8 | Certain | Docs audit scope: `_cli-external.md` (flags table + both examples + Known-change rule), `fab-operator.md` invocation lines, `_cli-fab.md` switch bullet, `companions.md`, + 3 SPEC mirrors; all other `wt create` mentions confirmed unaffected (bare exploratory create) | Grounded in a repo-wide grep sweep performed at intake, not inference | S:70 R:90 A:85 D:80 |
| 9 | Confident | `kit-architecture.md` memory correction limited to claims this change touches (batch-switch bullet + the `wt create` dual-semantics/baseWarnings description); the broader stale-since-4rtx wt section is out of scope | Minimal-blast-radius hydrate; a full wt-section restructure is a separate cleanup | S:50 R:85 A:75 D:65 |

9 assumptions (4 certain, 5 confident, 0 tentative, 0 unresolved).
