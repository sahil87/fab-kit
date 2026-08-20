# Intake: Autopilot Merge-Mode Naming Symmetry + `stacked-prs` Mode

**Change**: 260820-t6rq-autopilot-merge-modes-stacked-prs
**Created**: 2026-08-20

## Origin

Created via `/fab-proceed` promptless dispatch (`{questioning-mode} = promptless-defer`) from a synthesized user-conversation description. Key decisions were made in that conversation and are encoded verbatim below; no questions were asked at intake.

> Autopilot merge-mode naming symmetry + new `stacked-prs` mode (fab-operator). Rename the two existing autopilot merge modes to three symmetric noun names — `stack-then-review` (default), `merge-on-complete`, `stacked-prs` — dropping the misleading `--merge-on-complete` pseudo-flag spelling from skill prose; make the mode a real, persisted flag `fab operator autopilot start --mode <name>` stored in the autopilot state block; and add the new `stacked-prs` mode: stack-then-review merge timing with true stacked-PR topology (each same-repo dependent branches off its dependency's branch and PRs against it, so each PR diff shows only its own delta). Both halves ship in ONE change, per explicit user direction.

## Why

**Problem 1 — naming asymmetry / pseudo-flag.** `src/kit/skills/fab-operator.md` § Autopilot names its two merge modes asymmetrically: the default is the noun `stack-then-review`, the opt-in is spelled `--merge-on-complete` in CLI-flag notation. That flag does not exist anywhere in the Go binary (verified: zero occurrences of `merge-on-complete` under `src/go/`) — autopilot mode selection is conversational. Since change m7kq (#603) made `fab operator autopilot start --queue` a real binary verb (`src/go/fab/cmd/fab/operator_autopilot.go`), the pseudo-flag prose is actively misleading: it reads like something you could pass to that command. Side gap: the persisted autopilot state block (`autopilotState` in `src/go/fab/cmd/fab/operator_state.go` — `queue`/`current`/`completed`/`state`) has NO mode field, so the merge mode lives only in the operator's session context and is lost on `/clear` while the queue itself survives. If unfixed, an operator resuming after `/clear` cannot know whether the queue was meant to merge-as-it-goes.

**Problem 2 — no reviewer-clean stacked topology.** In stack-then-review, a chained PR's branch is created off `origin/{default_branch}` and its same-repo dependencies are cherry-picked in as a squashed `"operator: cherry-pick <dep> dependency"` commit. The PR's diff against the default branch therefore includes its dependencies' content, so reviewers re-see already-reviewed code on every dependent PR. Standard stacked-PRs practice — each PR based on its dependency's *branch*, showing only its own delta — is not available in any mode.

**Why this approach:** three flat mode names, selected by one real flag, chosen over making stacked topology a knob/variant of stack-then-review (e.g. a `pr_topology` sub-option) — rejected for user-facing simplicity. The flag+persistence fix also closes the `/clear` durability gap for the existing modes, not just the new one.

## What Changes

### 1. Real, persisted `--mode` flag on `fab operator autopilot start`

New flag on the existing verb (Go binary, `src/go/fab/cmd/fab/operator_autopilot.go`):

```
fab operator autopilot start --queue <id,id,...> [--mode <stack-then-review|merge-on-complete|stacked-prs>]
```

- Default: `stack-then-review` (matches today's default behavior).
- Validation: unknown mode value exits non-zero with a one-line error, matching the shared state-verb error posture (`_cli-fab.md` § shared state-verb mechanics).
- Persistence: `autopilotState` (`src/go/fab/cmd/fab/operator_state.go:41`) gains a `Mode string \`yaml:"mode"\`` field written by `start`, retained across `pause`/`resume`/`advance`, retained on queue exhaustion alongside `queue`/`completed`, and cleared with the whole block by `stop`. The mode thereby survives `/clear` — fixing the persistence gap.
- Back-compat: an existing state file whose `autopilot` block lacks `mode` reads as `stack-then-review` (tolerant read; owned-section re-marshal adds the field on next mutation).
- Scope boundary: the binary **stores, validates, and prints** the mode (via `fab operator state`); all merge choreography remains skill prose executed by the operator LLM. No merge logic enters the binary.
- Tests: extend `src/go/fab/cmd/fab/operator_autopilot_test.go` (flag parsing, default fill, validation error, persistence through the verb lifecycle, absent-field back-compat read).
- Docs: update `src/kit/skills/_cli-fab.md` § fab operator autopilot (signature + state-shape bullets) per the constitution's CLI ⇒ docs + tests rule.

### 2. Rename the modes symmetrically in skill prose

In `src/kit/skills/fab-operator.md` (canonical source — never `.claude/skills/`):

- Drop every `--merge-on-complete` pseudo-flag spelling; the mode is the noun `merge-on-complete`, selected via `--mode merge-on-complete` (or natural language).
- Three flat mode names throughout: `stack-then-review` (default), `merge-on-complete`, `stacked-prs`.
- Natural-language equivalents remain the conversational interface, mapping onto the flag: "merge as you go" / "merge on complete" / "merge each when done" → `merge-on-complete`; add equivalents for `stacked-prs` (e.g. "stacked PRs", "stack the PRs") mirroring the existing equivalents-list pattern.
- Confirmation prompt gains a third line for the new mode, following the existing per-mode pattern (§ Autopilot: "Default (stack-then-review): … creates PRs — merge after review." / "merge-on-complete: … merges PRs on completion."), e.g. "stacked-prs: Confirm upfront (creates stacked PRs — merge after review)."
- Sections to update: § Autopilot (mode definitions, per-change loop, confirmation prompts), § Dependency Resolution (mode-conditional same-repo strategy), the **Failures** line (rebase-conflict row is currently `merge-on-complete` only; stacked-prs adds its own — see change 3), **Interrupts**, § Queue Completion Summary and § Ordered Merge (stacked-prs shares them, with the extra merge-all choreography).

### 3. New `stacked-prs` mode

Same merge **timing** as stack-then-review — PRs created up front, merged only on explicit user request — but stacked PR **topology** for same-repo chains:

- **Branch base**: the dependent's branch is created off the dependency's *branch* instead of off `origin/{default_branch}` + cherry-pick. The squashed `"operator: cherry-pick"` commit does not exist in this mode for same-repo deps. The base-ref seam is the § 6 spawn sequence's worktree/branch creation step (`wt create` probe-and-route per `_cli-external.md` § wt); the exact mechanism (e.g. the `wt create --checkout <dep-branch>` route, or branch-from-ref at branch creation) is decided at apply — the spawn sequence already routes on existing branches.
- **PR base**: the dependent's PR base is the dependency's branch, so each PR diff shows only its own delta. `/git-pr` creates PRs with `gh pr create --draft` (no `--base` — defaults to the repo default branch) and the worker agent is mode-unaware, so the operator retargets after creation: `gh pr edit <pr> --base <dep-branch>`. `/git-pr` itself is unchanged (operator-side retarget chosen over teaching `/git-pr` a base parameter — mode is operator state; plumbing it through the spawn command adds complexity for one consumer).
- **Cross-repo dependencies**: unchanged — ordering-only barriers in every mode.
- **Non-scope**: dependency-branch drift after a dependent PR exists (a dep's review-pr rework moving its branch) is out of scope — the same exposure exists today in the cherry-pick model; conflicts surface at merge-all and escalate. <!-- assumed: drift policy out of scope for consistency with today's cherry-pick model; not discussed in the source conversation -->
- **"Merge all" choreography** (extends § Ordered Merge for this mode):
  1. Base-first merge order per repo, waiting on CI per PR — as today.
  2. After each merge of a chain's base PR, GitHub auto-retargets the dependent PR's base onto the default branch when the merged base branch is deleted — rely on this, and verify/retarget explicitly (`gh pr edit --base`) if the branch was not deleted.
  3. **Squash-merge duplicate-commit problem**: after a squash merge, the next branch in the chain still carries the dependency's original commits, which the default branch now contains only as a squashed commit — so before that next PR is clean/mergeable, rebase it onto the default branch, dropping the already-merged dependency commits (e.g. `git fetch origin && git rebase --onto origin/{default_branch} <merged-dep-branch> <next-branch>` + force-push). `{default_branch}` is resolved per Dependency Resolution step 0 — never a hardcoded `origin/main`.
  4. **Rebase conflict during merge-all → escalate** (do not silently skip), consistent with the cherry-pick-conflict policy. The Failures line therefore distinguishes: rebase conflict *mid-queue* → skip (`merge-on-complete` only, as today); rebase conflict *during stacked-prs merge-all* → escalate.

### 4. Docs/memory sweep (sibling-sweep classes per code-quality.md)

Grep-driven repo-wide sweep of `merge-on-complete` / `stack-then-review` / autopilot-mode claims before finishing apply:

- `docs/memory/runtime/operator.md` — restates the mode names, confirmation texts, per-change loop, `--merge-on-complete` opt-in paragraph, failure table, and the interrupts/state-block paragraph (verified occurrences at lines 49, 118, 229–251, 366). Present-truth prose updates to the three-mode model + `--mode` flag + persisted `mode` field.
- `docs/memory/pipeline/change-lifecycle.md:195` — mentions "`--merge-on-complete` rebases" in the default-branch-resolution convention; rephrase to the noun mode.
- `docs/specs/operator.md`, `docs/specs/skills.md` (§ fab-operator), `docs/specs/glossary.md` — aggregate specs restating autopilot facts; update where they carry present-truth mode claims (spec version-history rows and dated log files — `log.md`, `log.seed.md` — are historical records and stay untouched).
- Regenerate memory indexes via `fab memory-index` after memory writes.

### 5. Migration decision (recorded)

No `src/kit/migrations/` file. The operator state file is binary-owned, self-creating (`fab operator state` persists the empty skeleton), and lives under `$XDG_STATE_HOME/fab/operator/` — it is not project config, `.status.yaml`, or archive layout, which is the migration policy's scope (context.md § Migrations). Adding an optional field with a defined absent-value default (`stack-then-review`) is additive; the tolerant-read/typed-write posture handles old files without restructuring.

## Affected Memory

- `runtime/operator.md`: (modify) three-mode model (`stack-then-review`/`merge-on-complete`/`stacked-prs`), the real persisted `--mode` flag and state-block `mode` field, stacked-prs topology + merge-all choreography, updated failure table and confirmation texts
- `pipeline/change-lifecycle.md`: (modify) rephrase the "`--merge-on-complete` rebases" mention in the default-branch-resolution convention to the noun mode name

## Impact

- **Go binary**: `src/go/fab/cmd/fab/operator_autopilot.go` (new `--mode` flag, validation, default), `src/go/fab/cmd/fab/operator_state.go` (`autopilotState.Mode` field), `src/go/fab/cmd/fab/operator_autopilot_test.go` (tests — constitution: Go changes ship tests). Scoped test run: the `cmd/fab` package's operator tests.
- **Skills (canonical `src/kit/skills/`)**: `fab-operator.md` (§ Autopilot, § Dependency Resolution, Failures/Interrupts, Queue Completion Summary, Ordered Merge, confirmation prompts), `_cli-fab.md` (§ fab operator autopilot signature + state semantics). `/git-pr` unchanged (operator retargets PR base post-creation).
- **Specs**: `docs/specs/operator.md`, `docs/specs/skills.md`, `docs/specs/glossary.md` (present-truth autopilot claims only).
- **Memory**: per Affected Memory above + index regeneration.
- **No migration file**; no config schema change; no change to `/fab-fff`/dispatch surfaces.
- Behavior back-compat: default mode unchanged (`stack-then-review`); existing state files read correctly with the field absent.

## Open Questions

- None — promptless-defer produced no Unresolved decisions; the source conversation resolved the major forks explicitly (see Assumptions).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Both halves (rename/flag symmetry + new `stacked-prs` mode) ship in ONE change | Discussed — explicit user direction | S:95 R:70 A:90 D:95 |
| 2 | Certain | Three flat noun mode names: `stack-then-review` (default), `merge-on-complete`, `stacked-prs`; `pr_topology`-style sub-knob rejected | Discussed — user chose flat names for user-facing simplicity | S:95 R:75 A:90 D:95 |
| 3 | Certain | Mode becomes a real flag `fab operator autopilot start --mode <name>` (default `stack-then-review`), persisted in the autopilot state block so it survives `/clear` | Discussed — fixes the verified persistence gap (`autopilotState` has no mode field) | S:95 R:70 A:90 D:90 |
| 4 | Certain | `stacked-prs` = stack-then-review merge timing + stacked topology (branch off dep branch, PR base = dep branch, no cherry-pick commit for same-repo deps) | Discussed — the mode's defining contract, stated with specifics | S:95 R:70 A:85 D:90 |
| 5 | Certain | Cross-repo dependencies remain ordering-only barriers in all three modes | Discussed — explicitly unchanged | S:95 R:85 A:90 D:95 |
| 6 | Certain | Merge-all rebase conflict in `stacked-prs` → escalate, never silently skip (consistent with cherry-pick-conflict policy); mid-queue rebase-conflict-skip stays `merge-on-complete`-only | User-proposed handling with stated rationale; one prose line, trivially revisable | S:90 R:85 A:80 D:85 |
| 7 | Confident | No migration file: state file is binary-owned, self-creating, outside the migration policy's scope; absent `mode` reads as `stack-then-review` | Description asked to decide-and-record; context.md § Migrations scope + tolerant-read posture answer it | S:70 R:80 A:85 D:80 |
| 8 | Confident | Binary stores/validates/prints the mode only; all merge choreography stays skill prose | Matches the established binary-owns-state / operator-owns-choreography split (runtime/operator.md Full Mediation doctrine) | S:80 R:70 A:85 D:85 |
| 9 | Confident | PR base in `stacked-prs` set by operator post-creation (`gh pr edit --base <dep-branch>`); `/git-pr` unchanged | Worker is mode-unaware and mode is operator state; verified `/git-pr` passes no `--base` today — retarget is the minimal seam, and swapping it later is prose-only | S:45 R:75 A:65 D:55 |
| 10 | Confident | Merge-all mechanics: base-first per repo + CI wait (as today), rely-then-verify GitHub base auto-retarget, post-squash `git rebase --onto origin/{default_branch}` of the next chain branch + force-push | Constraints enumerated in discussion; exact git incantation is an implementation detail in revisable choreography prose | S:75 R:75 A:70 D:65 |
| 11 | Confident | `--mode` validates against exactly the three names; unknown value → non-zero one-line error | Matches `_cli-fab.md` shared state-verb error posture | S:55 R:80 A:85 D:80 |
| 12 | Confident | Mode is fixed for a queue's lifetime (set at `start`); changing mode = `stop` + new `start` | Obvious default of the persisted-at-start design; a mid-queue mode verb could be added later without breaking this | S:50 R:85 A:70 D:65 |
| 13 | Confident | Sweep updates present-truth prose only (skills, memory, aggregate specs); historical logs (`log.md`/`log.seed.md`) and spec version-history rows stay untouched | FKF present-truth rule + code-quality sibling-sweep classes | S:60 R:80 A:85 D:75 |
| 14 | Confident | NL equivalents remain the conversational interface mapping onto the flag; stacked-prs gets its own equivalents list | Discussed for existing modes; extension mirrors the established pattern | S:80 R:85 A:75 D:70 |
| 15 | Confident | Branch-off-dep mechanism lands in the §6 spawn sequence's worktree/branch step (e.g. `wt create --checkout <dep-branch>` route or branch-from-ref); exact seam chosen at apply | Spawn sequence already probe-and-routes on existing branches; several equivalent implementations, all apply-time revisable | S:55 R:70 A:50 D:45 |
| 16 | Tentative | Dependency-branch drift after a dependent PR exists (dep rework moves its branch) is out of scope — same exposure exists today in the cherry-pick model; conflicts surface at merge-all and escalate per row 6 | Not discussed; consistency-with-today default, recorded for /fab-clarify visibility | S:30 R:60 A:45 D:40 |

16 assumptions (6 certain, 9 confident, 1 tentative, 0 unresolved).
