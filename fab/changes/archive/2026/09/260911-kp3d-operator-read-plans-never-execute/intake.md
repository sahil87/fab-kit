# Intake: Operator principles — read plans, never execute, always spawn

**Change**: 260911-kp3d-operator-read-plans-never-execute
**Created**: 2026-09-11

## Origin

Conversational — synthesized from a user conversation about two observed failures of the running `/fab-operator` session on 2026-09-11, then dispatched promptless via `/fab-proceed` (create-new). The user's decision, verbatim intent:

> Instead of adding more conditions I would rather remove conditions. It's the job of the operator to keep reading plans and executing them until the end. The only condition we should add: it should not be executing stuff on its own. It should get it done via agents and worktrees.

**Observed failures that motivated it** (both on 2026-09-11, live operator session):

1. **Inline execution.** When the user gave the operator a task or reported a bug, the operator investigated and fixed it inline in its own pane instead of spawning a worktree agent. The §1 row "Coordinate, don't execute — Route implementation to agents; ask when ambiguous. Perform only maintenance-level actions such as merge, archive, and worktree deletion directly (§6)" has no trigger clause: nothing says a user bug report *is* a work request, and "ask when ambiguous" plus the vague "maintenance-level" carve-out read as outs. Diagnosis was treated as not-execution.
2. **Refusal to read a plan.** When the user asked the operator to "check the rk-mcp plan" (a multi-phase project roadmap: W0 merged, W1 merged, W2/W3/W4 not yet spawned), the operator refused to read the plan document, citing "operator doesn't read plan artifacts", and asked the user what the next phase should cover. The §1 row "Keep context lean — Never read intake/spec/plan artifacts; retain only pane maps, snapshots, and operator state (§2, §4)" blocked the operator from doing its core job: sequencing the next phase of tracked work.

Interaction mode: the description handed to this intake was already a settled decision — no further questions were asked (promptless dispatch; would-be questions, if any, are recorded as deferred Unresolved rows in `## Assumptions`).

## Why

**Problem.** Two §1 principle rows in `src/kit/skills/fab-operator.md` work against the operator's job:

- "Keep context lean" was written for context-window budget (the PR #550 restructure era, when the operator re-read its whole state file each tick and paid the full always-load layer after every `/clear`). Since then the operator's durable state moved into the server-keyed state file (`fab operator state` / `fab operator track` verbs), the tick payload became the bare `operator tick`, and §4 Post-Compaction Reload makes the operator survive context loss losslessly. The rule's *benefit* (a smaller context) is now marginal; its *cost* is that the operator cannot read the one document that tells it what to spawn next, so it asks the user instead — the exact failure observed with the rk-mcp roadmap.
- "Coordinate, don't execute" has no trigger and two escape hatches. Without a trigger, a bug report reads as a question to answer rather than work to route; "ask when ambiguous" invites the operator to stall on a decision it should default; and "maintenance-level actions such as …" is an open-ended list the operator stretched to cover "reproduce and fix a small bug".

**Consequence of not fixing.** The operator keeps doing the two things it must never do — authoring changes in its own pane (bypassing intake, review, PR, and the worktree isolation every other agent gets) and stalling on multi-phase plans it is supposed to drive to completion. Every such incident costs a human intervention and, in the inline-fix case, produces untracked work on whatever branch the operator happened to be on.

**Why this approach.** The user explicitly chose *removing* conditions over adding them. The one condition that stays — and gets sharpened — is "never execute in the operator pane": everything is done through agents in worktrees. Reading plans is not execution; it is coordination input. So: delete the read-prohibition row outright (no replacement condition such as "on demand only" or an allowlist of document kinds), and rewrite the execution row into a single work rule with an explicit trigger, an enumerated maintenance allowlist, and no "ask when ambiguous" hatch. Alternatives rejected: (a) adding a "bug report = work request" clause *alongside* the existing rows (more conditions, the opposite of the user's direction); (b) relaxing "Keep context lean" to "read plans on demand" (a new condition; the operator would again have to judge when a read is "on demand"); (c) an allowlist of readable document kinds (same problem — the rk-mcp roadmap was not a fab artifact at all).

## What Changes

Skill-text + docs only. **No Go code**, no `_cli-fab-operator.md` CLI-reference changes. Canonical source is `src/kit/skills/fab-operator.md`; `.agents/skills/` and `.claude/skills/` are deployed copies and are never edited (Constitution V; code-quality.md anti-pattern "Editing deployed skills directly").

### 1. `src/kit/skills/fab-operator.md` §1 Principles — delete one row, rewrite one row

**Delete** the row (currently line 41):

```
| Keep context lean | Never read intake/spec/plan artifacts; retain only pane maps, snapshots, and operator state (§2, §4). |
```

outright. No replacement row and no replacement condition. The operator MAY read any plan, roadmap, intake, or task document it needs to drive tracked work to completion. The "retain only pane maps, snapshots, and operator state" half is not carried anywhere — the §1 "Survive compaction" row and §4 Post-Compaction Reload already own the durable-state story.

**Rewrite** the row (currently line 37):

```
| Coordinate, don't execute | Route implementation to agents; ask when ambiguous. Perform only maintenance-level actions such as merge, archive, and worktree deletion directly (§6). |
```

into the single **work rule**. Proposed wording (apply may tighten phrasing; the five semantic elements below are required):

```
| Coordinate, don't execute | Every task, bug report, or idea the user hands the operator is a **work request** — it enters through §6 Working a Change (a fresh report is the raw-text form: a `/fab-new` spawn in a fresh worktree; a report naming a live tracked item is a send to that item's agent). Reading code to reproduce or diagnose, or editing files, in the operator pane **is** executing and is prohibited. The operator reads whatever plan, roadmap, intake, or task document the tracked work needs and keeps handing the next unit to an agent until the plan is done. Direct actions are exactly the maintenance allowlist: merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution (§6), `fab operator track` verbs (§4), and pane sends/answers/nudges (§5). The only thing the operator asks about a work request is **which repo** (plus §6 step 2's spawn-target-session tie-break) — never whether to spawn. |
```

Required semantic elements:

1. **Trigger clause** — any task, bug report, or idea handed to the operator *is* a work request. No "ask when ambiguous".
2. **Entry point** — §6 Working a Change (note: the originating description said "§8 Working a Change"; in the current file `### Working a Change` is under **§6 Coordination Patterns** — §8 is Configuration. Apply targets §6). Raw-text form → `/fab-new` spawn in a fresh worktree for a fresh report; the existing-change form for a report that names a live tracked item.
3. **Diagnosis is execution** — reading code to reproduce/diagnose, or editing files, in the operator pane counts as executing and is prohibited.
4. **Read-plans permission** — the operator reads any plan/roadmap/intake/task document it needs and drives multi-phase work to completion (the positive replacement for the deleted row, stated as permission, not condition).
5. **Enumerated maintenance allowlist** (closed list, replaces "maintenance-level actions such as …"): merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution, `fab operator track` verbs, pane sends/answers/nudges. Only asks: which repo; §6 step 2's spawn-target tie-break (already defined).

The row title stays "Coordinate, don't execute" so the spec/glossary phrasings ("Coordinates, never executes" / "Coordinates but never implements") remain recognisable cross-references.

### 2. `src/kit/skills/fab-operator.md` — other prose citing the deleted row

- **§2 Context Loading** (line 51): the sentence "code-quality, code-review, and the doc indexes serve artifact generation and review, which the operator never does (§1 Context discipline)" cites the deleted row by its old name. Re-anchor the rationale to the work rule: the operator never *authors or reviews* artifacts (§1 Coordinate, don't execute), so those files are dead weight in a long-lived session. **Delete** the trailing sentence "Do not load change artifacts." — it restates the deleted prohibition. The three-file always-load set (`config.yaml`, `constitution.md`, `context.md`) and "Do not run `fab preflight`" are **unchanged** — that exception concerns project boilerplate files at startup, not plans.
- **§6 The fab-change Kind** — the "Pipeline-first" bullet ends "Orchestration maintenance (merge, archive, worktree deletion) remains direct." This is a partial restatement of the allowlist; under the owner-or-pointer rule it becomes a pointer: "Direct actions are the §1 maintenance allowlist." (§1 owns the list.)
- **§9 Key Properties** (line 815): `| Loads change artifacts? | No — orchestration context only |` → a descriptive row, e.g. `| Reads change/plan artifacts? | As the work needs — any plan, roadmap, intake, or task document required to drive tracked work (§1); none are startup always-loads (§2) |`. Descriptive, not a new condition.
- Verify no other prose in the file cites "Context discipline", "Keep context lean", "maintenance-level", or "ask when ambiguous" (repo-wide grep in the sweep below found only lines 37, 41, 51, 815 plus the §6 bullet).

### 3. Sibling sweep — restatements outside the skill (code-quality.md § Sibling Sweeps; do up front, not after review)

Grep evidence (2026-09-11, `grep -rn -i` over `src/kit`, `docs`, `README.md` for the old phrases):

| File:line | Current text (abridged) | Edit |
|-----------|-------------------------|------|
| `docs/memory/runtime/operator.md:21` | "**Coordinate, don't execute.** The operator routes user instructions to the right agent — it never implements work directly. If the target is ambiguous, ask." | Rewrite to mirror the new work rule: trigger clause, diagnosis-is-execution, read-plans permission, closed allowlist, only-asks-which-repo. Drop "If the target is ambiguous, ask." |
| `docs/memory/runtime/operator.md:25` | "**Context discipline.** The operator never reads change artifacts (intakes, specs, tasks). Its context window is reserved for coordination state — pane maps, stage snapshots, tracked items." | **Delete** the principle paragraph. |
| `docs/memory/runtime/operator.md:31` | "…which the operator never does (per its own §1 Context discipline principle)…" and the trailing "It does NOT load change-specific artifacts." | Re-anchor the clause to the work rule (never authors/reviews); delete the trailing sentence. Three-file load text unchanged. |
| `docs/memory/runtime/operator.md:260` | "- **No change artifacts**: Never reads intakes, specs, or tasks — context window reserved for coordination state" | **Delete** the Design Constraints bullet. |
| `docs/memory/runtime/operator.md:384–388` | Design Decision "Operator Context Loading Trimmed to Three Files (zc9m)" — **Why** cites "against its own §1 'Context discipline' principle" | **Stays as history** (the decision was correct when made). Append one clause so its forward-looking claim does not contradict the new rule, e.g. "(the §1 read-prohibition row was later removed — see *Operator Reads Plans, Never Executes* below; the three-file startup load stands on the re-pay cost alone)". Do not rewrite the decision body. |
| `docs/specs/skills.md:1101` | "**Context**: … It runs no `fab preflight` and never reads change artifacts, keeping a long-lived context window reserved for coordination state." | Rewrite: runs no `fab preflight`; loads no change artifacts at startup; reads plan/roadmap/intake/task documents as the tracked work needs (§1 work rule). |
| `docs/specs/skills.md:1104` | "- **Coordinates, never executes** — all pipeline work is spawned into a freshly created worktree agent (`wt create --non-interactive`), never run in the operator's own pane. Operational maintenance (merge PR, archive, delete worktree) is the one direct-execution exception." | Rewrite to carry the trigger clause, diagnosis-is-execution, and the closed allowlist (merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution, `fab operator track` verbs, pane sends/answers/nudges). |
| `docs/specs/operator.md:23` | v9 row: "Spawn-in-worktree principle — operator pane reserved for coordination state; all pipeline work runs in freshly spawned agent tabs, never in the operator pane itself" | **Verify only** — consistent with the new rule (it is about *work*, not *reading*). No new version-history row; this is a principle sharpening within v11, not a skill iteration. |
| `docs/specs/glossary.md:77` | "Coordinates but never implements — all pipeline work is spawned into a fresh worktree agent." | **Verify only** — consistent. No edit unless apply finds a contradiction. |

**Historical text explicitly left alone**: `docs/memory/pipeline/log.md:298` and `log.seed.md:194` (the qkov log entry "Context discipline — loads always-load layer only, never change artifacts" — a dated hydrate log line), and every `docs/specs/findings/*` review finding (f117 etc.). History is not swept.

### 4. `docs/memory/runtime/operator.md` — new Design Decision

Append to `## Design Decisions` (four-field shape, FKF §3.3, matching the file's existing entries):

```
### Operator Reads Plans, Never Executes (kp3d)
**Decision**: The §1 "Keep context lean" row (never read intake/spec/plan artifacts) is deleted outright — the operator reads any plan, roadmap, intake, or task document it needs to drive tracked work to completion, with no replacement condition. The §1 "Coordinate, don't execute" row is the single work rule: every user task, bug report, or idea is a work request entering via §6 Working a Change (`/fab-new` spawn in a fresh worktree; a send for a live tracked item); reading code to diagnose or editing files in the operator pane counts as executing; direct actions are a closed maintenance allowlist (merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution, `fab operator track` verbs, pane sends/answers/nudges); the only question about a work request is which repo.
**Why**: Two 2026-09-11 incidents. (1) Handed a bug, the operator reproduced and fixed it inline — the old row had no trigger clause, and "ask when ambiguous" + "maintenance-level actions such as…" read as outs; diagnosis was treated as not-execution. (2) Asked to "check the rk-mcp plan" (W0/W1 merged, W2–W4 unspawned), the operator refused to read the roadmap, citing the read-prohibition, and asked the user what the next phase should cover. The read rule existed for context budget (PR #550 era); the server-keyed state file + §4 Post-Compaction Reload already make the operator survive context loss, so the rule's cost (cannot sequence a roadmap) exceeded its benefit. User direction: remove conditions rather than add them; the one condition that stays is "never execute in the operator pane".
**Rejected**: Adding a "bug report = work request" clause beside the existing rows (adds conditions). Relaxing the read rule to "on demand only" or an allowlist of document kinds (new conditions; the rk-mcp roadmap was not a fab artifact). Keeping "ask when ambiguous" (the stall hatch the incident exercised).
*Introduced by*: 260911-kp3d-operator-read-plans-never-execute
```

### 5. Unchanged by design

- §2 Context Loading's three-file always-load exception and `_preamble.md` §1's "`/fab-operator` loads a reduced 3-file set" note — untouched (project boilerplate, not plans).
- §3 Confirmation Tiers — untouched; it owns *how* each direct action is confirmed, §1 owns *which* actions are direct.
- §6 Working a Change's three forms and the spawn sequence — untouched; §1 points at them.
- Constitution — no amendment (no MUST rule added/changed; skill prose only).
- `docs/memory/runtime/operator.md` `description:` frontmatter — unchanged (it never mentioned the read rule), so no `fab docs-index` regeneration is required unless hydrate changes the description.

### Non-goals (deferred by the user to a later change — do NOT include)

- (a) Changing the every-10th-tick full-frame constant in the Go `tick-start` verb to a time-based 10-minute rule.
- (b) Having `tick-start` write an HTML fleet table beside the operator state file for a run-kit quake-terminal tab.
- No Go changes, no tests, no `_cli-fab-operator.md` changes.
- Backlog `hf6x` (route a review-pr CI failure back to the authoring agent before escalating) is adjacent — the new trigger clause covers *user*-handed work, not worker-reported failures. Not folded in.

## Affected Memory

- `runtime/operator`: (modify) rewrite the "Coordinate, don't execute" principle paragraph; delete the "Context discipline" principle paragraph and the "No change artifacts" design constraint; re-anchor the Context Loading clause; annotate the zc9m design decision; add the "Operator Reads Plans, Never Executes (kp3d)" design decision.

## Impact

- **Files**: `src/kit/skills/fab-operator.md` (§1 two rows, §2 one paragraph, §6 one bullet, §9 one row), `docs/memory/runtime/operator.md` (5 edits + 1 new DD), `docs/specs/skills.md` (2 bullets in the `/fab-operator` section), `docs/specs/operator.md` and `docs/specs/glossary.md` (verify-only, expected no edit).
- **Change type**: docs (skill prose + memory/specs). No Go, no tests, no CLI reference, no migration, no constitution amendment.
- **Expected lane**: light (a handful of tasks — one skill file, one memory file, one spec file, two verify-only files).
- **Behavioral effect**: a running operator, after its next `/fab-operator` reload, will (a) read plan/roadmap documents when asked to check or sequence them and spawn the next phase instead of asking the user, and (b) treat any user-handed task/bug/idea as a spawn, never a pane-local fix. No binary behavior changes.
- **Constraints honored**: Constitution V (edit `src/kit/` only; the skill cites no fab-kit-only paths — the new row references §-internal sections and kit helpers only); owner-or-pointer (the §1 row owns the allowlist; §6's bullet and §9's row point/describe, they do not re-enumerate with different wording); sibling sweep done up front per code-quality.md.
- **Deploy**: after merge, `fab sync` refreshes `.agents/skills/` / `.claude/skills/`; the live operator picks it up on its next compaction reload or restart.

## Open Questions

- None. Every decision was either taken in the originating conversation (recorded as Certain/Confident below) or is a low-stakes apply-time wording choice graded in `## Assumptions`. No question was deferred under the promptless dispatch.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Delete the §1 "Keep context lean" row outright; no replacement row and no replacement condition ("on demand only", document-kind allowlist all rejected) | Discussed — user's verbatim direction: remove conditions rather than add; "the job of the operator [is] to keep reading plans and executing them until the end" | S:95 R:90 A:95 D:95 |
| 2 | Certain | Rewrite "Coordinate, don't execute" into the single work rule with an explicit trigger: any user task, bug report, or idea is a work request entering via Working a Change | Discussed — the description names the trigger clause as the fix for the inline-fix incident | S:95 R:90 A:90 D:90 |
| 3 | Certain | Reading code to reproduce/diagnose, or editing files, in the operator pane counts as executing and is prohibited | Discussed — description states it explicitly ("Diagnosis was treated as not-execution" is the named failure) | S:95 R:90 A:90 D:95 |
| 4 | Certain | Remove "ask when ambiguous"; the only question about a work request is which repo (plus §6 step 2's existing spawn-target tie-break) — never whether to spawn | Discussed — description states the exact ask set | S:95 R:90 A:90 D:90 |
| 5 | Certain | Closed maintenance allowlist: merge PR, archive, worktree deletion, rebase/cherry-pick for dependency resolution, `fab operator track` verbs, pane sends/answers/nudges | Discussed — the enumerated list is given in the description; replaces the open "such as …" phrasing | S:90 R:90 A:85 D:85 |
| 6 | Certain | §2 three-file always-load exception (config/constitution/context; no code-quality/code-review/doc indexes) and `_preamble.md`'s 3-file note are unchanged | Discussed — description marks it UNCHANGED; it concerns startup boilerplate, not plans | S:95 R:95 A:95 D:95 |
| 7 | Certain | Non-goals (a) time-based full-frame rule in Go `tick-start`, (b) HTML fleet table — excluded; no Go, no `_cli-fab-operator.md` changes | Discussed — user explicitly deferred both to a later change | S:95 R:95 A:95 D:95 |
| 8 | Certain | Add a memory Design Decision in `docs/memory/runtime/operator.md` recording the removal and both incidents; four-field shape with `*Introduced by*: 260911-kp3d-…` | Discussed — requested in the description; shape follows the file's existing entries and `_generation.md` § Plan Generation step 3 | S:90 R:95 A:95 D:95 |
| 9 | Certain | Sweep all restatements up front: memory operator.md lines 21/25/31/260/386; specs skills.md 1101/1104; verify operator.md:23 and glossary.md:77 | Discussed + code-quality.md § Sibling Sweeps (aggregate specs + the skill's memory file are named classes); grep evidence in § What Changes 3 | S:90 R:90 A:95 D:95 |
| 10 | Certain | Historical text stays: the zc9m DD body, `docs/memory/pipeline/log*.md` qkov entries, `docs/specs/findings/*` — only the zc9m DD gets a one-clause forward-looking annotation | Discussed — description: "stays as history but its forward-looking claim must not contradict the new rule"; log/findings are dated records | S:85 R:95 A:90 D:90 |
| 11 | Certain | Change type `docs` (skill prose + memory/specs; no Go) | Determined by change-types keyword taxonomy and the description's "Change type: docs/skill-text" | S:90 R:95 A:95 D:95 |
| 12 | Certain | Edit `src/kit/skills/fab-operator.md` only; never `.agents/skills/` or `.claude/skills/` | Constitution V + code-quality.md anti-pattern; deterministic | S:95 R:95 A:100 D:100 |
| 13 | Certain | The description's "§8 Working a Change" means the current file's §6 `### Working a Change` (§8 is Configuration); apply targets §6 and the row cites §6 | Verified against `fab-operator.md` outline — the heading exists only under §6; a stale section number in a synthesized description, not a request to move the section | S:70 R:90 A:90 D:85 |
| 14 | Confident | Row title stays "Coordinate, don't execute" (content rewritten) rather than a new title such as "Spawn, never execute" | Keeps the spec/glossary cross-references ("Coordinates, never executes", "Coordinates but never implements") recognisable; low-stakes wording, reversible in one edit | S:55 R:95 A:75 D:65 |
| 15 | Confident | A user report that names a *live tracked item* is routed as a send to that item's agent (Working a Change existing-change form); a fresh report is the raw-text `/fab-new` spawn. Both are "never in the operator pane" | The description emphasises the raw-text form; §6 Working a Change already defines the existing-change form, and routing a live item's bug to its own agent is the pipeline-first default. Two options with a clear front-runner | S:55 R:85 A:70 D:60 |
| 16 | Certain | §6 "The fab-change Kind" bullet's trailing "Orchestration maintenance (merge, archive, worktree deletion) remains direct" becomes a pointer to the §1 allowlist; §9 "Loads change artifacts?" row becomes descriptive ("as the work needs … none are startup always-loads") | code-quality.md owner-or-pointer rule — §1 owns the allowlist; a second enumeration is the drift mechanism the rule exists to kill. The §9 row is a properties table (descriptive), not a condition | S:65 R:90 A:85 D:75 |
| 17 | Confident | No new version-history row (v12) in `docs/specs/operator.md`; v9 row verified consistent and left as-is | The table records skill iterations (rewrites, new mechanisms); this change sharpens two principle rows within v11. Reversible — a row can be added at hydrate if the reviewer disagrees | S:60 R:90 A:65 D:55 |
| 18 | Certain | Re-anchor §2 Context Loading's rationale to "the operator never authors or reviews artifacts (§1 Coordinate, don't execute)" and delete "Do not load change artifacts."; do not add any replacement guard sentence | Follows from #1 (no replacement condition) and #6 (startup set unchanged); the exclusion of code-quality/code-review/doc indexes still has a valid rationale (never authors/reviews + per-reload re-pay) | S:70 R:90 A:85 D:75 |
| 19 | Certain | No constitution amendment, no `fab docs-index` regeneration expected (operator.md `description:` unchanged), no migration | No MUST rule touched; frontmatter unchanged; no user-data restructuring | S:75 R:95 A:90 D:90 |
| 20 | Certain | Expected lane: light — the plan should land at a handful of tasks (skill file, memory file, spec file, verify-only pair) | The description's expectation; the lane fork is decided by plan task count at apply entry, so this is informational | S:70 R:95 A:80 D:80 |

20 assumptions (17 certain, 3 confident, 0 tentative, 0 unresolved).
