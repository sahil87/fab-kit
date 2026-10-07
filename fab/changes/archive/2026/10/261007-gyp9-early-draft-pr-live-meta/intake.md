# Intake: Early draft PR + live, refreshable `## Meta` block

**Change**: 261007-gyp9-early-draft-pr-live-meta
**Created**: 2026-10-07

## Origin

Synthesized from a design conversation about PR/CI latency in the fab pipeline. The user's framing, verbatim in substance:

> Today nothing is committed or pushed until the `ship` stage. `apply`, `review`, and `hydrate` are all pre-commit. The PR only exists at the very end, so `ci-gate` (a required check) only starts running post-ship. The decision: open the draft PR at **apply exit** and push at each subsequent stage boundary, so CI runs concurrently with `review` and `hydrate` and surfaces failures to the agent that can still fix them in context.

Interaction mode: conversational design session, then a promptless `/fab-proceed`-style dispatch (`{questioning-mode} = promptless-defer`) — no interactive questions were asked during intake.

Decisions reached in that conversation and carried verbatim into this intake:

- The PR opens at **apply exit**, not at ship.
- The `## Meta` block becomes **marker-delimited** (`<!-- fab pr-meta:start -->` / `<!-- fab pr-meta:end -->`) and refresh becomes a pure **replace-between-markers** splice — mirroring the `fab docs-index` manual-block convention already in use.
- Refresh is **orchestrator-owned**, not a dispatched-worker obligation, so stage workers never need `gh` on PATH.
- Intermediate CI runs are **accepted signal**, not noise — do not design around suppressing them.
- The `review-pr` stage, `/git-pr-review`, and the Copilot request/poll machinery are **explicitly out of scope** — a separate follow-on change.

## Why

### The problem

`/git-pr` (the `ship` stage) is the first and only moment anything reaches the remote. Everything before it — `apply`, `review`, `hydrate` — is purely local and uncommitted. Two measured consequences in this repo:

1. **CI latency sits entirely on the critical path to merge.** `ci-gate` is a required check on `main`, so the user cannot merge until it passes. Because the PR is created at the very end of the pipeline, `ci-gate` only *starts* after ship. Measured: PR create→merge median **37 min**, with **33%** of PRs merging in under 20 min — CI is a large fraction of that window. Nothing about the pipeline's own work prevents CI from having run already.
2. **A broken intermediate state is not caught until after ship**, when the agent that wrote the code has already finished and lost its context. The feedback lands on the operator or the user instead of the author.

### What happens if we don't fix it

Every change keeps paying the full CI wall-clock serially after ship, and every CI failure keeps arriving after the authoring context is gone — which is exactly the cost structure `/git-pr` Step 3a-ter's rebase-at-ship rationale already names ("the last moment the change's author context is present"). The pipeline already believes author-context-at-push is valuable; it just applies that belief once, at the end.

### Why this approach over alternatives

- **Open the PR early and push at stage boundaries** (chosen): CI overlaps `review` and `hydrate` for free, and a failure surfaces while the apply worker (or the orchestrator, in the light lane) can still act on it. No new infrastructure — the commit/rebase/push/create machinery already exists in `/git-pr`.
- **Run CI locally before ship** (rejected): duplicates the CI definition, diverges from the real `ci-gate`, and does not shorten the required-check clock on the PR at all.
- **Keep ship-only PRs but speed up CI** (rejected): orthogonal, and does not address the lost-author-context half of the problem.

A PR opened at apply exit immediately exposes a second defect: the `## Meta` block is **write-once**. `git-pr.md` Step 3c writes it on PR create, and Step 3d prepends it to an existing OPEN PR *guarded on its absence* — so re-running never updates it. An early PR would therefore carry a permanently stale Meta block: `0/N` tasks, review pending, no impact numbers, Pipeline line showing only `intake ✓`. Making Meta refreshable is not a nice-to-have alongside the early PR; it is a prerequisite for it.

## What Changes

### 1. The PR opens at apply exit

The draft PR is created when `apply` completes, instead of at `ship`. Subsequent stage boundaries — `review` (pass), `hydrate`, `ship` — commit and push, keeping the PR current as subagent workers complete stages.

Concretely, the pipeline's git model moves from:

```
intake → apply → review → hydrate → [commit, rebase, push, create PR] ship → review-pr
```

to:

```
intake → apply → [commit, rebase, push, create draft PR] →
         review → [commit, rebase, push, sync Meta] →
         hydrate → [commit, rebase, push, sync Meta] →
         ship → [commit, rebase, push, sync Meta, finalize] → review-pr
```

Implications that must be handled explicitly:

- **More than one push per change.** `git-pr.md` Step 3a-ter's rebase-onto-base now runs more than once, so **force-push-with-lease happens repeatedly** rather than once. The existing explicit pre-fetch lease OID logic is load-bearing and must keep holding across repeated pushes: the lease MUST be the OID captured **before** `git fetch origin` (`lease_oid=$(git rev-parse -q --verify '@{u}')`), never the bare `--force-with-lease` flag — after the fetch, `origin/<branch>` already holds whatever the remote has now, so a bare lease would accept a stranger's push and overwrite it. Each boundary's push repeats that capture independently.
- **The remote-branch divergence guard (3a-ter step 2) now fires at every boundary**, not once. Its STOP semantics are unchanged.
- **Rework cycles do not push individually.** The push happens at the *review-pass* boundary, not per auto-rework cycle — otherwise a 3-cycle rework triples the CI runs for no added signal.
- **Who commits the apply output in the full lane.** `apply` runs in a dispatched worker (native, headless, or pane). The worker writes files; the **orchestrator** commits and pushes at the boundary, consistent with the orchestrator-owned rule in change 4 below. Stage workers do not gain a commit obligation.
- **The apply-exit PR open is a non-stage orchestrator step.** It reuses `/git-pr`'s commit/rebase/push/create mechanics but does **not** touch `progress.ship`; the `ship` stage keeps running `/git-pr` verbatim, which then finds an OPEN PR, rebases, pushes, syncs Meta, and finalizes (`fab status add-pr`, `finish ship`). The state machine, `stage_metrics`, and the operator's PR tracking are untouched.
  <!-- assumed: apply-exit PR open is a non-stage step and `ship` keeps its /git-pr transitions — the user decided WHEN the PR opens, not whether that re-homes the ship stage's fab status transitions; a plan could instead re-scope `ship` -->

### 2. `## Meta` becomes marker-delimited and refreshable

Wrap the rendered block in HTML-comment markers:

```markdown
<!-- fab pr-meta:start -->
## Meta

| Change ID | Type | Confidence | Plan | Review |
|-----------|------|-----------|------|--------|
| `gyp9` | feat | 3.8 / 5.0 | 7/7 tasks, 12/12 acceptance ✓ | ✓ 2 cycles |

...Impact table + caption, optional **Issues**, **Pipeline:** line...
<!-- fab pr-meta:end -->
```

Refresh is a **pure replace-between-markers** operation: find `start`, find `end`, replace everything between them with the freshly rendered block, leave the rest of the body byte-identical. This leaves the agent-written `## Summary` and `## Changes` sections — and any human edit to the body — untouched. Those two sections are therefore authored **once**, at PR create (apply exit, from the intake), and are never refreshed by the sync.

This mirrors the convention already used by `fab docs-index` in `docs/memory/index.md` and `docs/specs/index.md` (`<!-- fab docs-index:manual:start -->` / `:manual:end`; implemented at `src/go/fab/internal/memoryindex/adoption.go`), so it is an **established repo pattern, not a new one**. Reuse its shape (and, where sensible, its splice helper) rather than inventing a second marker dialect.

**Migration case — an unmarked `## Meta` block.** Every PR created before this change has a bare `## Meta` heading with no markers. The first refresh against such a body MUST **adopt** the existing block rather than duplicate it:

- Locate the `## Meta` heading and treat the section as running to the next top-level `## ` heading (or end of body).
- Replace that whole span with the marker-wrapped freshly rendered block.
- A body with **no** `## Meta` at all keeps today's Step 3d behavior: prepend the (now marker-wrapped) block ahead of the existing body.
- A body that already has markers takes the plain splice path.

All three paths MUST be idempotent (Constitution III): running the sync twice with unchanged inputs produces a byte-identical body and performs no edit the second time.

### 3. A PR-sync operation

Render Meta → splice between markers → apply via `gh pr edit --body-file -`. Idempotent: a no-op when the spliced body equals the current body (compare before calling `gh pr edit`; do not issue an edit that changes nothing).

**Where it lives is an implementation decision for the plan — evaluate and pick.** The tradeoff, stated by the user:

| Option | For | Against |
|--------|-----|---------|
| New `fab` subcommand (e.g. `fab pr-sync <change> --type <t> [--issues …]`) | `fab pr-meta` already renders the block and the splice is mechanical — the Go binary is its natural home; testable byte-stably like `prmeta.Render` | `gh pr edit` invocation currently lives entirely in the skill layer; moving it into Go puts a network/`gh` dependency inside the binary |
| New mode/flag on `/git-pr` (e.g. `/git-pr <change> --sync-meta`) | Keeps `gh` in the skill layer where it is today; no new CLI surface | `/git-pr` is a full ship pipeline; a mode that does only Meta sync strains its contract and its `allowed-tools` framing |
| New standalone skill | Clean separation | A third skill for a mechanical splice; more prose to keep in sync |

Whichever is chosen, the **render** half stays `internal/prmeta` and the block stays byte-stable.

**If a new or changed `fab` command signature results**, the owning CLI reference partial MUST be updated — `src/kit/skills/_cli-fab.md` § `fab pr-meta` is the owning section for this family — and Go test updates MUST ship in the same change (Constitution Additional Constraints; `fab/project/code-review.md` § Project-Specific Review Rules).

### 4. The orchestrator calls the sync at each stage boundary

Keep it **orchestrator-owned** — the skill that sequences stages — rather than making it a dispatched-worker obligation, so stage workers never need `gh` available. Sites:

- `src/kit/skills/_pipeline.md` — the shared ff/fff bracket: Step 1 (apply exit → open PR), Step 2 (review pass → push + sync), Step 3 (hydrate → push + sync).
- `src/kit/skills/fab-fff.md` — Steps 4–5 (ship, review-pr) already dispatch `/git-pr`; ship's own 3c/3d now find an OPEN PR and finalize.
- `src/kit/skills/fab-ff.md` — the twin skill (terminal `hydrate`). It gets the **same** behavior: the PR opens at apply exit and the review/hydrate boundaries push, so an ff run ends with an OPEN draft PR that a later `/git-pr` finalizes. `fab/project/code-quality.md` § Sibling Sweeps makes `fab-ff` ↔ `fab-fff` a mandatory sweep pair.
- `src/kit/skills/fab-continue.md` — the single-stage advance path; it sequences one stage at a time and is named as an orchestrator for this purpose.

It MUST work in **both lanes**:

- **Light lane** — everything but review runs inline in the orchestrator's own context. The sync is just another inline step.
- **Full lane** — apply/review/hydrate run in dispatched workers (native / headless / pane). The orchestrator performs the commit, push, and sync *after* reading the worker's result, before the next stage's dispatch.

### 5. Reconcile `git-pr.md` Step 3a-bis with multiple commit points

Step 3a-bis (memory-index refresh via `fab docs-index docs/memory`, then a separate `docs: refresh memory indexes` commit) is currently gated on **ship** and on "3a just committed this invocation". With commits now happening at apply, review, and hydrate boundaries too, this gate must be reconciled.

`fab docs-index docs/memory` is byte-stable and a no-op when nothing drifted, so running it at more than one commit point is safe. The substantive question is *which* boundaries should run it. Hydrate is the stage that writes `docs/memory/`, so the hydrate boundary is where the refresh actually has work to do; the ship-time run then stays as the idempotent backstop it already is. Whichever rule the plan adopts, it MUST preserve:

- the never-hand-merge rule for generated `docs/memory/**/index.md` and `log.md` on a rebase conflict (resolve topic files, re-run `fab docs-index docs/memory`, take its output wholesale — FKF §5, `$(fab kit-path)/reference/fkf.md`);
- the separate-commit discipline (never `--amend` the content commit);
- the suppress-empty-commit guard (`git diff --quiet -- docs/memory`).

### 6. Sibling sweep (do this up front, not reactively)

Per `fab/project/code-quality.md` § Sibling Sweeps, sweep the whole twin/sibling class before finishing apply:

- `src/kit/skills/fab-ff.md` ↔ `src/kit/skills/fab-fff.md`
- The aggregate specs that restate these facts: `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md`
- The memory file documenting `/git-pr` behavior: `docs/memory/pipeline/execution-skills.md`
- `src/kit/skills/fab-adopt.md` — it is a partial consumer of the `_pipeline.md` bracket and `git-pr.md` Step 3d is documented as "the ship-stage Meta retrofit `/fab-adopt` relies on"; the retrofit's semantics change here.

Grep the old claims repo-wide (`## Meta`, `pr-meta`, "retrofit", "created at ship", "Step 3d") and update every occurrence in the class.

## Affected Memory

- `pipeline/execution-skills`: (modify) `/git-pr` ship behavior (Steps 3a-bis / 3a-ter / 3b / 3c / 3d), the `_pipeline.md` two-lane bracket's stage boundaries, and the `fab-fff` Steps 4–5 ship/review-pr wiring — all restate the "PR is created at ship" model and the write-once Meta retrofit.
- `pipeline/schemas`: (modify) § `fab pr-meta` subcommand — the marker contract, the new/changed command surface if the sync lands in Go, and the refresh (rather than create-only) consumer relationship with `/git-pr`.
- `pipeline/change-lifecycle`: (modify) git integration — the single-push-at-ship model and the ship-time branch↔change guard are documented here; the early-push model and repeated force-push-with-lease change that description.

## Impact

**Kit skills (canonical sources under `src/kit/` — never edit the deployed copies under `.agents/skills/` or `.claude/skills/`, Constitution V):**

- `src/kit/skills/git-pr.md` — Steps 3a-bis, 3a-ter, 3b, 3c, 3d, 4a–4c
- `src/kit/skills/_pipeline.md` — Steps 1–3 stage boundaries, Light Lane, Auto-Rework Loop (push timing)
- `src/kit/skills/fab-fff.md` — Steps 4–5
- `src/kit/skills/fab-ff.md` — sibling sweep
- `src/kit/skills/fab-continue.md` — single-stage advance path
- `src/kit/skills/fab-adopt.md` — Meta-retrofit consumer
- `src/kit/skills/_cli-fab.md` — § `fab pr-meta` (owning CLI reference partial for this family)

**Go:**

- `src/go/fab/internal/prmeta/prmeta.go` — renders the Meta block; gains (or is wrapped by) the marker emission and splice
- `src/go/fab/internal/prmeta/prmeta_test.go` — byte-stable render goldens + new splice/adoption/idempotency cases
- `src/go/fab/cmd/fab/pr_meta.go` (+ `pr_meta_test.go`) — the command surface, if the sync lands here
- `src/go/fab/internal/memoryindex/adoption.go` — existing marker-splice precedent (`manualStart`/`manualEnd`); reuse rather than re-implement where the shapes match

**Specs (human-curated, Constitution VI — update by hand, not by tooling):**

- `docs/specs/architecture.md`, `docs/specs/skills.md`, `docs/specs/glossary.md`

**Constraints that bind this change:**

- **Constitution V (canonical sources)** — edit `src/kit/`, never a deployed copy.
- **Constitution V (citation rule)** — deployed `src/kit/**` content MUST NOT cite fab-kit-only paths (`docs/specs/*`, `docs/memory/*`, `docs/site/*`, `src/go/*`) as an authority. A Go guard test in `src/go/fab-kit/cmd/fab/` fails on this. Restate the operative rule in the skill instead.
- **Constitution III (idempotency)** — the sync is a no-op when nothing changed; re-running any boundary is safe.
- **CLI ⇒ docs + tests** — any `fab` command-signature change updates `src/kit/skills/_cli-fab.md` and ships Go tests.
- **Test-alongside** — Go changes ship their tests in the same change; run the affected packages before considering apply done.

**Explicit non-goals (do not touch, do not reference as a dependency):**

- The `review-pr` stage, `/git-pr-review`, and the Copilot request/poll machinery. A separate follow-on change immediately after this one owns them. Do not pre-emptively remove anything related to them.
- Suppressing CI on intermediate pushes. Intermediate CI failures are deliberate, useful signal.

## Open Questions

None blocking. Two decisions are delegated to the plan by the user's own framing and are recorded as graded assumptions below rather than as questions:

- Which home the PR-sync operation takes (new `fab` subcommand vs. a `/git-pr` mode vs. a new skill) — see What Changes § 3.
- Which commit boundaries run the Step 3a-bis memory-index refresh — see What Changes § 5.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The draft PR is created at **apply exit**; `review`, `hydrate`, and `ship` boundaries commit and push | Stated flatly as "the decision" in the originating conversation, with the measured motivation (37 min median create→merge) attached | S:95 R:70 A:85 D:90 |
| 2 | Certain | Meta refresh is a pure replace-between-markers splice using HTML-comment markers `<!-- fab pr-meta:start -->` / `<!-- fab pr-meta:end -->` | User named the mechanism and the marker shape; it mirrors the shipped `fab docs-index` manual-block convention (`internal/memoryindex/adoption.go`), so it is an established repo pattern, not a new dialect | S:80 R:80 A:90 D:85 |
| 3 | Certain | The sync is **orchestrator-owned** (`_pipeline.md` / `fab-fff.md` / `fab-ff.md` / `fab-continue.md`), never a dispatched-worker obligation | Explicitly decided, with the stated reason: stage workers must not need `gh` on PATH. Directly constrains where the calls go | S:90 R:70 A:85 D:85 |
| 4 | Certain | `review-pr`, `/git-pr-review`, and the Copilot machinery are untouched; intermediate CI failures are accepted signal and are not suppressed | Both stated as explicit non-goals with reasons (separate follow-on change; failure is useful signal) | S:95 R:75 A:90 D:95 |
| 5 | Certain | The pre-fetch lease OID discipline (3a-ter step 1 → 3b step 3) holds unchanged across every repeated push; the bare `--force-with-lease` flag is never used | Named in scope by the user and already encoded in `git-pr.md` with its rationale; repetition changes the frequency, not the rule | S:85 R:65 A:95 D:90 |
| 6 | Confident | The PR-sync lands as a **`fab` Go subcommand** (render + splice + `gh pr edit`), with `/git-pr` and the orchestrators calling it | User delegated the pick but supplied the full tradeoff and named the front-runner reasoning: `fab pr-meta` already renders the block and the splice is mechanical. The codebase answers it directly — `internal/memoryindex/adoption.go` is a shipped splice precedent, and byte-stable Go goldens are the strongest guard against Meta drift. The splice helper is the durable asset; its invocation site is a localized skill + CLI-reference edit | S:70 R:60 A:80 D:60 |
| 7 | Confident | First refresh of an **unmarked** body adopts the existing `## Meta` section (heading → next top-level `##`) by replacing it with the marker-wrapped block; a body with no `## Meta` prepends; a marked body splices | The requirement is stated outright — "decide and implement a sensible upgrade path so the first refresh adopts the markers rather than duplicating the block" — only the mechanism is delegated, and adoption-by-span-replacement is the one interpretation that satisfies it literally while staying idempotent | S:75 R:70 A:80 D:65 |
| 8 | Confident | Pushes happen at **stage boundaries only** — auto-rework cycles inside `review` do not each push; the push fires when review passes | The user enumerated the boundaries explicitly as "review, hydrate, ship" — stage boundaries, not rework cycles. Per-cycle pushes would multiply CI runs without adding signal the boundary push does not already carry, and the push site is a one-line skill-prose placement either way | S:75 R:70 A:75 D:70 |
| 9 | Confident | Step 3a-bis's memory-index refresh runs at the **hydrate** boundary (where `docs/memory/` is actually written), with the existing ship-time run retained as the idempotent backstop | The user named the reconciliation precisely as in-scope without prescribing the answer, and the codebase answers it: `fab docs-index docs/memory` is byte-stable and no-ops when nothing drifted, so extra runs are safe, and hydrate is the only stage that mutates memory. The never-hand-merge, separate-commit, and empty-commit-guard rules are preserved either way; the gate is skill prose, cheap to re-tune | S:55 R:70 A:75 D:60 |
| 10 | Confident | In the full lane the **orchestrator** commits and pushes the dispatched worker's output after reading its result; workers gain no commit/push/`gh` obligation | Follows directly from assumption 3's user-stated orchestrator-ownership rule and from `_preamble.md` § Dispatch-Prompt Obligations, which already fixes a worker's obligations at result-file + terminal `fab status refresh` — adding a commit obligation would be a contract change nobody asked for | S:65 R:70 A:85 D:75 |
| 11 | Tentative | The apply-exit PR open is a **non-stage orchestrator step** that reuses `/git-pr`'s commit/rebase/push/create mechanics without touching `progress.ship`; the `ship` stage keeps `/git-pr` verbatim (it finds an OPEN PR, rebases, pushes, syncs Meta, and finalizes) | The user decided *when* the PR opens, not whether that re-homes the `ship` stage's `fab status` transitions — the description is silent here. Getting it wrong cascades into `.status.yaml` semantics, the `pr-meta` Pipeline line, and the operator's PR tracking, so it is expensive to undo mid-apply. The conservative default (do not touch the state machine unless told) is a strong engineering front-runner, but a plan could legitimately re-scope `ship`. Marked for `/fab-clarify` | S:45 R:40 A:55 D:55 |
| 12 | Certain | `/fab-ff` (terminal `hydrate`) also opens the PR at apply exit and pushes at the review/hydrate boundaries, leaving an OPEN draft PR that a later `/git-pr` finalizes | **User-decided 2026-10-07**, asked directly and answered "fab-ff opens it too" with the tradeoff stated (its documented "stopping before PR stages" purpose gets rewritten; every ff run pushes a public draft PR). Rewriting `fab-ff.md`'s Purpose line is therefore an in-scope deliverable, not an optional sweep | S:95 R:70 A:90 D:95 |
| 13 | Confident | The agent-written `## Summary` / `## Changes` are authored once at PR create (apply exit, from the intake) and never refreshed by the sync | Stated outright by the user: the splice "leaves the agent-written `## Summary` and `## Changes` sections, and any human edit to the body, untouched". It follows mechanically from that contract, the intake those sections derive from is stable by apply exit, and an opt-in re-author remains a cheap later addition | S:80 R:70 A:80 D:75 |

13 assumptions (6 certain, 6 confident, 1 tentative, 0 unresolved).
