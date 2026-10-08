# Intake: Copilot Review Request at Ship + Drop the `/fab-archive` Nudge

**Change**: 261008-or8g-ship-copilot-request-drop-archive-nudge
**Created**: 2026-10-08

## Origin

Synthesized from a `/fab-discuss` session on 2026-10-08 covering two end-of-pipeline behavior
fixes. Both concern what happens *after* the automatic pipeline's terminal `ship` stage. The
session produced two user decisions, both already resolved — placement, guards, and the exact
routing-table rows were decided in the discussion, not left open here.

> Two end-of-pipeline behavior fixes, decided in a /fab-discuss session.
>
> **Decision 1 — Request a Copilot review at ship.** Add a one-line, fire-and-forget reviewer
> request at the end of the ship stage: `gh pr edit --add-reviewer copilot-pull-request-reviewer`.
> Placement (decided): a NEW step in `src/kit/skills/git-pr.md` immediately AFTER Step 4b (Finish
> Ship Stage) — NOT in Step 3c (PR create). Guards (decided): idempotent probe before requesting;
> best-effort (never fails ship); must not fire on the MERGED STOP path; no polling, no gate, no
> config knob.
>
> **Decision 2 — Stop nudging `/fab-archive` after every change.** Remove `/fab-archive` from the
> Next-Steps routing table entirely. Archiving is POST-MERGE housekeeping whose timing fab cannot
> know (the user merges by hand, sometimes days later, sometimes in a batched sweep like PR #690
> which archived 9 changes at once), so `/fab-archive` is never a correct automatic "next command".

The discussion also verified, against this repository, the empirical premise that change
`261007-4p4z` relied on when it deleted the request path — and found it false (see § Why).

## Why

### Decision 1: the "the repo will request it" premise was false

Change `261007-4p4z-retire-review-pr-stage` retired `review-pr` as a pipeline stage and deleted
every Copilot request path. Its recorded Design Decision (`docs/memory/pipeline/execution-skills.md`
§ "`/git-pr-review` Never Requests a Review, Even Manually", ~line 427–455) explicitly **rejected**
"a fire-and-forget request at ship", on the stated premise that *"post-ship cross-vendor eyes come
from the repository's own review setup (e.g. GitHub's automatic Copilot review)"*
(§ "Reviewer Diversity Lives Outside the Review Block", same file).

That premise was verified **FALSE for this repository** during the discuss session:

1. `gh api repos/sahil87/fab-kit/rulesets/14613881` carries rules `deletion`, `non_fast_forward`,
   `pull_request`, and `required_status_checks` — there is **no `copilot_code_review` rule**. The
   repository has no automatic review setup to inherit.
2. **PR #690** — the first PR created after 4p4z shipped, with no `/git-pr-review` run — has
   **ZERO reviews**. Every Copilot review in the repo's history (#688 at +6.9 min, #689 at
   +6.5 min after PR creation) arrived via the request path 4p4z deleted.
3. Measurement from change `gyp9`: **42% of PRs carry a Copilot fix**. Without a request, that
   signal is lost on every PR — a cost that was accepted on the (false) assumption that the repo
   would supply the review itself.

**Consequence if not fixed**: every PR from here on ships with no second-vendor review at all, and
the 42% fix-rate signal silently disappears. The user would have to remember to run
`/git-pr-review` manually on every PR — exactly the kind of "remember to do the thing" step the
pipeline exists to eliminate.

**Why this approach over alternatives**: a GitHub ruleset `copilot_code_review` rule was the
zero-code alternative and was **rejected**: it fires on "ready for review", but `/git-pr` creates
and leaves PRs as `--draft`, so it would not fire until the user manually un-drafts to merge — at
which point the review is useless. An explicit `gh pr edit --add-reviewer` works on a draft PR,
which is why this one line is the fix.

**Why Step 4b and not Step 3c**: `_pipeline.md` § PR Boundary Procedure's "apply-exit open" reuses
`git-pr`'s Steps 3a/3a-ter/3b/3c to open the **draft PR right after apply**. A request wired into
Step 3c would therefore request a Copilot review on half-finished, **pre-review** code — burning
the review on a diff that the pipeline's own review stage has not even seen yet. Step 4b runs only
at ship, which is the correct moment.

### Decision 2: `/fab-archive` is never the right "next command"

Agents repeatedly ask "Should I run `fab archive`?" after every completed change. Archiving is
**post-merge housekeeping**:

- `fab` cannot know when the merge happened — the user merges by hand, sometimes days later.
- Archiving is often batched: PR #690 archived **9 changes at once** in a single housekeeping PR.
- `internal/resolve/resolve.go:200,242` skips `archive/`, so an archived change makes
  `/git-pr-review <change>` (explicit argument) hard-STOP with "Cannot resolve change" — archiving
  early actively breaks the one remaining manual stage.

So a `Next: /fab-archive` line is a nudge toward an action that is wrong at the moment it is
printed. Removing the row from the routing table removes the nudge at its source, rather than
teaching each skill to suppress it.

**Why removal over suppression**: a "sometimes correct" routing entry is the thing that produces
the repeated question. The routing table's contract is "the command to run next"; archive does not
satisfy it in any state. The `/fab-archive` skill itself stays — it is still the right command when
the user decides to archive.

## What Changes

### 1. `/git-pr`: request a Copilot review at ship

**New step in `src/kit/skills/git-pr.md`, immediately after Step 4b (Finish Ship Stage)** — call it
Step 4b-bis (or renumber as the implementer prefers, provided it sits between 4b and 4c and runs
only on the ship path).

Behavior:

1. **Gate**: runs only when a PR URL is known AND the MERGED STOP path (Step 3, ~line 142–158) did
   not fire. It MUST NOT fire on the merged-PR STOP.
2. **Idempotence probe (Constitution III)**: before requesting, check whether
   `copilot-pull-request-reviewer` is already a *requested* reviewer **or** has already *submitted*
   a review. If either holds, skip the request silently. Re-running `/git-pr` on a shipped PR must
   not re-request a re-review.

   **Probe surface — correction to the drafted design.** The discuss note proposed
   `gh pr view --json reviewRequests,reviews`. `gh pr view --json` is GraphQL-backed, and this
   repo has already recorded that **GraphQL `reviewRequests` omits bot reviewers**
   (`docs/memory/runtime/operator.md:255`; `docs/memory/pipeline/log.md:59`, change `qg64`), so the
   `reviewRequests` half of that probe would always read empty and the request would fire on every
   re-run. Use the REST surface for the pending-request half:

   ```bash
   # pending request (REST — GraphQL omits bot reviewers)
   gh api "repos/{owner}/{repo}/pulls/{number}/requested_reviewers" \
     --jq '.users[].login' 2>/dev/null
   # already-submitted review
   gh pr view {number} --json reviews --jq '.reviews[].author.login' 2>/dev/null
   ```

   **Two logins, deliberately different** (recorded in `qg64` / `u1m1`):

   | Surface | Login to match |
   |---------|----------------|
   | `requested_reviewers` (REST) | `Copilot` |
   | `reviews[].author.login` | `copilot-pull-request-reviewer` |

   Match either (a prefix/`startswith` match on the reviews side is the safer form — finding f180
   notes REST and GraphQL render the bot login differently, with and without a `[bot]` suffix).
3. **The request** (only when the probe found neither):

   ```bash
   gh pr edit {pr_url_or_number} --add-reviewer copilot-pull-request-reviewer 2>/dev/null \
     || echo "  ! copilot review request skipped (no entitlement or gh error)"
   ```
4. **Best-effort, always**: a failure — no Copilot entitlement in the repo/org, a `gh` error, a
   network timeout — prints a **one-line warning** and **NEVER fails the ship**. Customer repos
   without Copilot must see zero breakage. This step never STOPs, never retries, never blocks
   Step 4c.
5. **Print on success**: one line, e.g. `  ✓ review — requested copilot-pull-request-reviewer`
   (matching the existing `  ✓ {thing} — {detail}` output style of Steps 3a/3c/4c).

**Explicit non-scope for this step**: no polling, no waiting, no gate, no timeout budget, no
`--tool` flag, no config knob. It is a request only. It does **not** revive the retired `review-pr`
pipeline stage, and it does **not** re-introduce the retired `review_tools` config block (retired
outright in 2.29.0 — `_cli-fab.md:419`). `ship` remains the terminal stage;
`/git-pr-review` remains manual-only.

**`git-pr.md` § Rules** gains a line matching the existing best-effort phrasing, e.g.
"The Copilot review request is best-effort — never blocks shipping."

### 2. The memory Design Decision must record the reversal

`docs/memory/pipeline/execution-skills.md` § "`/git-pr-review` Never Requests a Review, Even
Manually" (~line 427–434) currently lists *"a fire-and-forget request at ship"* in its **Rejected**
line. That block MUST be **rewritten to record the reversal and its evidence** — not silently
contradicted by the skill prose:

- The **Decision** stays true for `/git-pr-review` itself (it still carries no request path, no
  poll, no `--tool`); the *ship-time* request is a separate, newly-added `/git-pr` behavior.
- The **Rejected** entry for the fire-and-forget-at-ship option must be moved out of Rejected and
  recorded as adopted, with the three evidence items from § Why (no `copilot_code_review` ruleset
  rule; PR #690 zero reviews; 42% fix rate), plus the draft-PR argument against the ruleset
  alternative.
- `*Updated by*: 261008-or8g-…` is appended to the block.
- § "Reviewer Diversity Lives Outside the Review Block" carries the same false premise in its
  **Decision** line (*"post-ship cross-vendor eyes come from the repository's own review setup
  (e.g. GitHub's automatic Copilot review)"*) — reword it to name the ship-time request as the
  source instead.

### 3. Remove `/fab-archive` from the Next-Steps routing table

**`src/kit/skills/_preamble.md` § Next Steps Convention — State Table (lines 229–231)**:

```
| hydrate          | /git-pr        | /git-pr        |
| ship             | /git-pr-review | /git-pr-review |
| review-pr (pass) | —              | —              |
```

(Previously: `hydrate` → `/git-pr, /fab-archive`; `ship` → `/fab-archive, /git-pr-review` with
`/fab-archive` as DEFAULT; `review-pr (pass)` → `/fab-archive`.)

**`_preamble.md` line 245 — the Precedence paragraph** below the table currently ends
"…deterministically selects `/fab-archive`". It must end by selecting `/git-pr-review` for the
`ship: done` + `review-pr: pending` case.

**`_preamble.md` § Next Steps Convention — new rule for a no-command terminal state.** The
convention currently says skills MUST end output with a `Next:` line; the new `review-pr (pass)`
row has no available commands. Document the rendering: **omit the `Next:` line and print
`Pipeline complete.` instead.** `src/kit/skills/_pipeline.md` lines ~49, ~65, ~67 already render
`Pipeline complete.` followed by `Next: {per state table}` — those stay as-is (both `/fab-ff`'s
`hydrate` terminal and `/fab-fff`'s `ship` terminal still have a command), but the new rule must be
worded so it does not contradict them.

### 4. `src/kit/skills/fab-status.md` line ~46

```
3. Routing stage/default command as `Next: {stage} (via {command})`, or `Next: /fab-archive` when complete
```

must render the two-branch form instead: `Next: {stage} (via {command})` for a routing stage,
`Next: /git-pr-review` while `review-pr` is still `pending`, and `Pipeline complete.` (no `Next:`
line) once `review-pr` is `done`/`skipped`.

### 5. `src/go/fab/internal/change/change.go` — the non-obvious site

The nudge is **also emitted by the Go binary**, not just skill prose. Removing only the prose would
leave the binary still printing it.

**Correction to the drafted description**: the emitter is `change.Switch()` — i.e.
`fab change switch` / `fab switch` output (the `Stage: … / Confidence: … / Next: …` block at
`change.go:234–249`) — **not** `fab status`. A repo-wide grep confirms `change.go:247` is the only
non-help Go site printing `/fab-archive` (`cmd/fab/fab_help.go:159,160,179` is the flow diagram,
which is explicitly out of scope).

**Current code** (`change.go:239–249`):

```go
// Only when the automatic pipeline has run its course … does it collapse to
// the bare post-pipeline suggestion `/fab-archive` …
if allResolved {
    fmt.Fprintf(&output, "Next:        /fab-archive")
} else {
    fmt.Fprintf(&output, "Next:        %s (via %s)", routingStage, defaultCommand(routingStage))
}
```

`allStagesResolved` (`change.go:255–271`) returns true when every stage is `done`/`skipped`
**including a still-`pending` review-pr** (manual-only since 4p4z).

**Decided alignment** — the new table maps the `ship` state (ship `done` + review-pr `pending`) to
`/git-pr-review`, so the Go branch must split in two, keeping the binary's output and the
skill-side state table behaviourally identical:

| Progress state | New output |
|----------------|------------|
| Not all resolved | `Next:        {routingStage} (via {defaultCommand})` *(unchanged)* |
| All resolved, `review-pr` still `pending` | `Next:        /git-pr-review` (bare, no `(via …)`) |
| All resolved, `review-pr` `done` or `skipped` | `Pipeline complete.` — no `Next:` line |

Also update:
- the explanatory comment at `change.go:239–245` (it names `/fab-archive` twice and describes the
  old collapse),
- the `defaultCommand` doc comment at `change.go:466–469`: *"The all-done case is handled by the
  caller (allStagesResolved → /fab-archive), not by this map."*

### 6. Go tests that pin the old string (MUST be updated)

`src/go/fab/internal/change/change_test.go` — three cases assert `wantNext: "Next:        /fab-archive"`:

| Line | Case name | New expectation |
|------|-----------|-----------------|
| ~749–752 | `ship done with pending manual-only review-pr collapses to fab-archive` | `Next:        /git-pr-review` (rename the case accordingly) |
| ~759–762 | `all done collapses to fab-archive` | `Pipeline complete.` with no `Next:` line |
| ~764–767 | `hydrate done with trailing skipped collapses to fab-archive` | `Pipeline complete.` (review-pr is `skipped` here) |

The two existing negative cases (`review-pr active routes to git-pr-review, not fab-archive`,
`review-pr failed surfaces git-pr-review, not fab-archive`, lines ~744 and ~754) keep their
assertions; their names reference `fab-archive` only as the thing NOT printed — rename for clarity
if convenient, but the expectations do not change.

### 7. Sweep class — additional sites found beyond the drafted list

Per `fab/project/code-quality.md` § Sibling Sweeps, aggregate specs that restate per-skill facts and
the memory files documenting a skill's behavior are in the sweep class. A repo-wide
`grep -rn "fab-archive"` found these sites the drafted description did **not** list:

| Site | Verdict |
|------|---------|
| `src/kit/skills/fab-adopt.md:162` — `Next: /fab-archive, /git-pr-review` | **CHANGE** → `Next: /git-pr-review` |
| `src/kit/skills/fab-switch.md:108` — table row "…`Next:` shows only `/fab-archive`" | **CHANGE** → must match the new two-branch Go behavior (§5) |
| `docs/memory/_shared/context-loading.md:176–180` § Next Steps Convention (State Table) — states "The `ship` row's default is `/fab-archive`" | **CHANGE** (memory) |
| `docs/memory/pipeline/change-lifecycle.md:259` — "Next-line derivation (k4ge)": "…collapse to the bare post-pipeline suggestion `Next: /fab-archive`" | **CHANGE** (memory) |
| `docs/memory/pipeline/change-lifecycle.md:174` — "After either pipeline completes, run `/fab-archive` to move the change to archive." | **CHANGE** → reword to post-merge framing |
| `docs/specs/skills.md:200` — `/fab-continue` → hydrate row: `Next: /fab-archive` | **CHANGE** |
| `docs/specs/skills.md:692, 711` — "Land at ship done → summary + `Next: /fab-archive`" | **CHANGE** → `/git-pr-review` |
| `docs/specs/skills.md:859` — hydrate → `"Next: /fab-archive"` | **CHANGE** |
| `src/kit/skills/fab-operator.md:456` — "Maintenance: rebase, merge PR, `/fab-archive`" | **KEEP** — this *is* post-merge, the correct moment |
| `src/kit/skills/fab-continue.md:265` — "Moves change folder…? No — use `/fab-archive`" | **KEEP** — factual cross-reference, not a nudge |
| `docs/specs/{overview,glossary,architecture,templates,user-flow}.md` | **AUDIT** — most hits describe the `/fab-archive` skill itself (keep); `user-flow.md:38,55,56,119,120` and `overview.md:144` render archive as the step *after* the pipeline, which stays true. Change only lines that present it as a routed next-command |
| `README.md:256,286,433`, `docs/site/workflows.md:55`, `docs/site/skill.md:79` | **AUDIT** — same rule |
| `docs/specs/findings/*`, `docs/findings/*`, `fab/changes/archive/**`, `docs/memory/**/log*.md` | **KEEP** — historical records, never swept |

The implementer MUST re-run `grep -rn "fab-archive"` repo-wide and audit every hit before finishing.
`docs/memory/pipeline/index.md` and the specs index are generated — refresh via `fab docs-index`,
never hand-edit.

### 8. Explicitly OUT of scope

- **`fab help`'s flow diagram** (`src/go/fab/cmd/fab/fab_help.go:159,160,179`:
  `/fab-new → /fab-ff → /fab-archive`) stays **as-is**. It is the on-demand discoverability
  surface, not what nags the user. (`fab_help.go:38`'s `"fab-archive": "Completion"` category
  mapping likewise stays.)
- The `/fab-archive` **skill itself**, `fab change archive`, and `fab batch archive` are all
  unchanged.
- **Auto-running `fab batch archive` at the end of ship** — investigated and **rejected** during
  the discussion: (a) `fab change archive` moves tracked files and edits `fab/backlog.md` +
  `fab/changes/archive/index.md` with **no commit step** (see `fab-archive.md` § Dirty-tree
  disclosure), so it would leave an uncommitted rename on top of an already-pushed PR; (b) archive
  currently means "merged", and archiving at ship would put un-merged changes in the archive index;
  (c) `internal/resolve/resolve.go:200,242` skips `archive/`, so an archived change makes
  `/git-pr-review <change>` hard-STOP with "Cannot resolve change".
- **`fab switch --none` at ship** — the lighter fix for the underlying `.history.jsonl` churn was
  identified during the discussion but is **out of scope here**. It MUST NOT be implemented, and
  MUST NOT be mentioned anywhere as implemented.

## Affected Memory

- `pipeline/execution-skills`: (modify) — rewrite § "`/git-pr-review` Never Requests a Review, Even
  Manually" to record the ship-time-request reversal with its three evidence items; reword
  § "Reviewer Diversity Lives Outside the Review Block"'s "the repository's own review setup" claim;
  document the new `/git-pr` ship-time request step (placement, idempotence probe, best-effort
  contract) and the `/fab-archive` removal from the `/git-pr`/`/fab-adopt` Next lines.
- `pipeline/change-lifecycle`: (modify) — the switch Next-line derivation paragraph (k4ge, line
  ~259) gains the two-branch terminal rendering; line ~174's "After either pipeline completes, run
  `/fab-archive`" is reframed as post-merge housekeeping; line ~168's archive description stays
  factual.
- `_shared/context-loading`: (modify) — § Next Steps Convention (State Table), lines ~176–180: the
  10-state table's `ship`/`hydrate`/`review-pr (pass)` rows, and the new no-command terminal-state
  rendering rule.

## Impact

**Kit skills (canonical sources only — `src/kit/skills/*.md`)**:
`git-pr.md` (new step + Rules line), `_preamble.md` (3 table rows + Precedence paragraph + new
terminal-state rule), `fab-status.md` (line ~46), `fab-adopt.md` (line 162), `fab-switch.md`
(line 108).

**Go (`src/go/fab`)**: `internal/change/change.go` (the `allResolved` branch + two doc comments),
`internal/change/change_test.go` (3 cases updated). No command signature changes, so no
`_cli-fab.md` signature update is strictly required — `_cli-fab.md` does not document
`fab change switch`'s `Next:` output format today (only `archive`/`restore` YAML). If the reviewer
holds that the partial owns this output, add a one-line note under the `change` family table.

**Docs**: `docs/memory/pipeline/execution-skills.md`, `docs/memory/pipeline/change-lifecycle.md`,
`docs/memory/_shared/context-loading.md`, `docs/specs/skills.md`, plus the audit sweep in § 7.

**Constraints that bind this change**:
- **Constitution V**: `src/kit/**` is deployed into customer repos — the new `git-pr.md` step MUST
  NOT cite fab-kit's own `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` as an
  authority. The two-login fact and the REST-vs-GraphQL caveat must be **restated inline** in the
  skill, not pointed at.
- **Constitution III**: the idempotence probe is the governing guard for the request.
- **Constitution Additional Constraints + `code-quality.md`**: a change to the `fab` binary's output
  MUST include test updates (§ 6 covers this).
- Edit canonical sources at `src/kit/skills/*.md` **only** — never the deployed copies under
  `.agents/skills/` or `.claude/skills/`.
- **Two Go modules** in this repo; run Go tests from the module root (`src/go/fab`).

**Known environment hazard**: the deployed `.agents/skills/_preamble/SKILL.md` in this worktree is
**stale** — it still carries a pre-4p4z State Table (`ship → /git-pr-review`, `review-pr (pass) →
/fab-archive`, no Precedence paragraph) that does not match canonical `src/kit/skills/_preamble.md`.
Do not read the deployed copy as truth, and when refreshing deployed copies use a **dev** binary
built from this worktree (`FAB_KIT_PATH=$PWD/src/kit fab-kit-dev sync`), not the released `fab` —
the released-binary sync has silently reverted in-repo content before (the stale-binary trap).

**Testing**: `go test ./internal/change/...` from `src/go/fab`. The Copilot request step is skill
prose with no Go surface, so it has no automated test — its correctness is verified by the
idempotence probe reading correctly on a real PR (re-running `/git-pr` on a shipped PR must print
the skip, not re-request).

## Open Questions

*(none — both decisions and all guards were settled in the discuss session; the two corrections
below are recorded as graded assumptions rather than questions, since the repository's own memory
answers them unambiguously.)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Copilot request goes in a NEW `git-pr.md` step immediately after Step 4b, never in Step 3c | User-decided in discuss; rationale verified — `_pipeline.md` § PR Boundary Procedure reuses Step 3c at apply exit, so a 3c request would review pre-review code | S:95 R:85 A:90 D:95 |
| 2 | Certain | Request is best-effort: a failure prints one warning line and never fails ship; no polling, gate, or config knob | User-decided; matches the existing silently-ignored-failure idiom already used at `git-pr.md` Steps 4a/4b | S:95 R:90 A:90 D:95 |
| 3 | Certain | Request must NOT fire on the MERGED STOP path (git-pr.md Step 3, ~line 142–158) | User-decided; the merged path returns before any ship bookkeeping | S:95 R:85 A:90 D:90 |
| 4 | Confident | Idempotence probe uses **REST** `pulls/{n}/requested_reviewers` for the pending half, not GraphQL `reviewRequests` as drafted | Repo memory records that GraphQL `reviewRequests` omits bot reviewers (`runtime/operator.md:255`; `pipeline/log.md:59`, change qg64) — the drafted probe would read empty and re-request on every run | S:60 R:80 A:85 D:70 |
| 5 | Certain | Two distinct logins: `Copilot` under REST `requested_reviewers`, `copilot-pull-request-reviewer` on `reviews[].author.login` | Documented in this repo's memory (u1m1, qg64) and finding f180; the `--add-reviewer` argument is the latter | S:70 R:80 A:90 D:85 |
| 6 | Certain | State-table rows are exactly as specified (hydrate → `/git-pr`; ship → `/git-pr-review`; review-pr (pass) → `—`) | User-selected resolution, given verbatim | S:95 R:85 A:95 D:95 |
| 7 | Confident | A no-command terminal state omits the `Next:` line and prints `Pipeline complete.` instead | Stated as the discuss recommendation, not a hard user decision; consistent with `_pipeline.md`'s existing `Pipeline complete.` literal and reversible via `/fab-clarify` | S:70 R:85 A:75 D:65 |
| 8 | Certain | Go: bare `Next:        /git-pr-review` while review-pr is `pending`; terminal rendering only once review-pr is `done` | Decided alignment — keeps the binary byte-consistent with the new skill-side table | S:85 R:80 A:85 D:80 |
| 9 | Certain | The Go emitter is `change.Switch()` (`fab change switch`/`fab switch`), not `fab status` as drafted | Verified: `change.go:247` is the only non-help Go site printing `/fab-archive`; `/fab-status` renders its own line from `fab-status.md:46` | S:80 R:90 A:95 D:95 |
| 10 | Confident | `review-pr: skipped` is treated as terminal, same as `done` (no table row exists for it) | `allStagesResolved` already treats `skipped` as resolved; test case at change_test.go:764 pins the skipped path, so it needs a decided answer | S:40 R:80 A:70 D:60 |
| 11 | Certain | `fab help`'s flow diagram (`fab_help.go:159,160,179`) and the `fab-archive → "Completion"` category mapping stay unchanged | User-decided out of scope — discoverability surface, not a nudge | S:95 R:90 A:90 D:95 |
| 12 | Certain | No config knob, no revival of `review_tools`, no revival of `review-pr` as an automatic stage | User-decided; `review_tools` was retired outright in 2.29.0 with no destination (`_cli-fab.md:419`) | S:95 R:85 A:90 D:95 |
| 13 | Confident | Sweep rule: only lines presenting `/fab-archive` as a *routed next command* change; descriptive references to the skill and post-merge instructions (fab-operator.md:456, fab-continue.md:265) stay | Follows directly from the stated rationale (archive is correct post-merge, wrong as an automatic next step); the keep/change split for each site is listed in § 7 | S:65 R:75 A:80 D:70 |
| 14 | Confident | No `_cli-fab.md` update is strictly required (no signature change; the partial does not document `change switch`'s `Next:` output today) | Verified by grep — `_cli-fab.md:102/108` cover the `switch` signature and `archive`/`restore` YAML only; flagged in § Impact so review can overrule | S:60 R:85 A:75 D:65 |
| 15 | Certain | Two sites beyond the drafted list MUST change: `fab-adopt.md:162` and `fab-switch.md:108` | Found by repo-wide grep; both print a routed `Next:` containing `/fab-archive`, so both fall squarely under Decision 2 | S:75 R:85 A:95 D:90 |
| 16 | Confident | The existing execution-skills DD block is **rewritten in place** with `*Updated by*: 261008-or8g-…`, not replaced by a new block | Matches the file's established convention for reversals (every other block uses `*Introduced by*` + `*Updated by*`); the drafted description asks for the reversal to be recorded, not for a specific block shape | S:65 R:80 A:85 D:70 |
| 17 | Confident | `fab switch --none` at ship is NOT implemented and is mentioned nowhere as implemented | User-decided out of scope and stated explicitly; the risk is only that an eager implementer adds it | S:85 R:70 A:80 D:80 |

17 assumptions (10 certain, 7 confident, 0 tentative, 0 unresolved).
