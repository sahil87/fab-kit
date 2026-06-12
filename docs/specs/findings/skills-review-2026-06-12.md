# fab-kit Skills Audit — 2026-06-12

> Multi-agent audit of all canonical skill sources (`src/kit/skills/*.md`, 31 files), SPEC mirrors (`docs/specs/skills/`), non-skill spec docs, scaffold/templates/schemas, and the Go CLI (`src/go/fab/`), run the day after the four skills-review batches (PRs #390–#393, v2.1.6) merged. 253 agents (~15.9M tokens): a prior-findings digest (known-open items from the [2026-06-11 review](skills-review-2026-06-11.md) excluded by construction), 18 per-skill review units + 7 cross-cutting lenses (CLI contract vs Go source, helper-model integrity, pipeline coherence, reference integrity, residual duplication, architecture, templates/config) + 4 critic-proposed follow-up sweeps (SPEC-hooks vs shipped Go hooks, non-skill spec docs, git-state safety, exit-code contract vs Go). Every finding adversarially verified — must-fix findings required two independent confirmations — and the headline findings were reproduced empirically against a built binary or sandbox repos.

**Totals**: 199 raw findings → **175 confirmed** (21 must-fix · 82 should-fix · 63 nice-to-have · 9 structural); 24 refuted or duplicate findings dropped before synthesis.

IDs (`a###`) are stable for cross-referencing; the backlog batch entries (k4ge, w7dp, g8st, c5tr, d9rs) reference the theme sections below. Line numbers refer to the tree at commit 1431a9c3 (v2.1.6).

---
## 1. Executive summary

The four review batches genuinely improved the corpus: the `_pipeline` bracket single-sourcing held, terminal paths are mostly routed, and the prose of the 30 skill sources is largely internally coherent. The residual risk concentrates at three boundaries the batches never crossed. First, **the documented CLI contract vs. the Go binary**: `fab score --check-gate` exits 0 on gate failure (cmd/fab/score.go:25-32), so the pipeline's single human checkpoint — the intake confidence gate that `/fab-ff` and `/fab-fff` rely on — is silently bypassable under the exit-code detection the docs prescribe; relatedly, `fab status advance <chg> ship` writes a schema-forbidden `ready` state that permanently bricks preflight for that change. Second, **orchestration seams**: `/fab-fff`'s change-name override ships and mutates the *active* change while Steps 1–3 operate on the override change, and the operator autopilot's spawn sequence double-dispatches. Third, **everything outside `src/kit/skills/`**: the scaffold still seeds the deleted `spec` stage into every new project (permanently, since `fab init` stamps past the migration), `workflow.yaml` still defines 7 stages, and SPEC-hooks describes a shell-script hook system that no longer exists. A secondary lesson: several defects were *introduced by* the batches themselves (#393's f080 dedup deleted the semver-comparison rule its own new step needs; the f049 spawn consolidation created the autopilot double-dispatch; fab-clarify's zero-gaps early exit makes its bulk-confirm primary scenario unreachable) — single-sourcing refactors need a seam audit as a follow-up step. Severity profile after de-duplication across lenses: ~15 distinct must-fix, ~40 should-fix; the top defects were independently converged on by multiple lenses, which raises confidence in this ranking.

---

## 2. Priority themes

### Theme 1 — Documented CLI contracts the Go binary does not honor (highest impact)

Skills and helper references assert exit-code/flag behavior the binary contradicts. Because `_preamble.md`'s generic failure rule keys STOP on non-zero exit, a *false* "exits non-zero" claim means failures silently pass, and a *false* "exits 0" claim means benign paths abort.

Findings:
- `src/kit/skills/_preamble.md:230` + `_cli-fab.md:91` + `_pipeline.md` Pre-flight 3 — **`fab score --check-gate` documented as non-zero on gate fail; binary always exits 0** (3 convergent must-fix findings; empirically reproduced — `gate: fail` on stdout, exit 0). The ff/fff intake gate is undetectable via the documented contract.
- `src/kit/skills/_preamble.md:232` — **canonical form `fab change resolve --folder` is an invalid command** (2 convergent must-fix findings; `ERROR: unknown flag: --folder`, exit 1). Agents copying the canonical column hit the failure rule and STOP.
- `src/go/fab/internal/status/status.go:21-61` + `_cli-fab.md:58` — **state machine permits transitions its own schema forbids**: `advance ship|review-pr` writes `ready`, `skip intake` writes `skipped`; preflight then exits 1 forever ("State 'ready' not allowed for stage ship"). Empirically reproduced.
- `src/kit/skills/fab-switch.md:84-99` / `src/go/fab/internal/change/change.go:217-223` — **`fab change switch` Next: guidance off-by-one** at post-review stages (prints `/fab-archive` where `/git-pr-review` is correct; diverges from fab-status for the same state).
- `src/kit/skills/_cli-fab.md:168` — "All hook subcommands exit 0" false for `fab hook sync`.
- `src/kit/skills/_cli-fab.md:42-46` + `fab-archive.md:110` — **documented re-archive soft skip is unreachable** (genuinely archived changes exit 1 "No change matches"); archive partial failure (YAML + non-zero) undocumented; `fab change archive` with no arg exits 0 with help text; `fab batch archive` exits 1 on empty sets with failed==0.
- `src/kit/skills/fab-archive.md:153` / `internal/archive/archive.go:171-178` — restore `--switch` swallows activation failure, rendered as "not requested" (2 findings).

**Action:** one Go conformance change with tests (constitution requires same-PR `_cli-fab.md` updates): check-gate exits non-zero on `gate: fail`; `lookupTransition` rejects targets not in AllowedStates; fix `Switch`'s Next: routing; `ExactArgs(1)` on archive; `pointer: failed`; then correct the `_preamble`/`_cli-fab` rows that remain intentionally exit-0. Ship the gate fix first — it is the only one that bypasses a safety gate.
**Slug:** `cli-exit-contract-conformance` (gate fix alone: `score-gate-exit-nonzero`)

### Theme 2 — Orchestrator dispatch seams: wrong change, unparseable commands, undefined paths

The orchestration layer (fab-fff, fab-proceed, fab-operator, _pipeline) hands off via strings and substring resolution with no target verification.

Findings:
- `src/kit/skills/fab-fff.md:39-61` — **Steps 4–5 pass `change: {id}` to /git-pr and /git-pr-review, but both self-resolve only the ACTIVE change** (must-fix): with the advertised `<change-name>` override, Steps 1–3 work the override change while ship/review-pr mutate the active change's status and push whatever branch is checked out.
- `src/kit/skills/_pipeline.md:106` — **exhaustion-stop recovery `/fab-clarify intake` is unexecutable** (must-fix): `intake` parses as a change name, clarify is stage-guard-blocked post-intake, and the promised requirements regeneration never happens (plan.md is preserved on reset).
- `src/kit/skills/fab-operator.md:489-492` vs `:391` — autopilot per-change loop **double-dispatches** (spawn step 5 embeds `'<command>'`; Gate and Dispatch come after) — #393 f049 regression (verifier: should-fix).
- `src/kit/skills/fab-operator.md:463` — initial command `/fab-switch <change> && /fab-proceed` relies on nonexistent slash-command chaining semantics.
- `src/kit/skills/fab-operator.md:475/481/511` — implicit queue chaining (`depends_on: [<prev>]`) contradicts its own worked example (queue-previous vs nearest-same-repo-predecessor produce different worktrees).
- `src/kit/skills/fab-continue.md` (4 sites) + `_pipeline.md:106` — **five callers still invoke `/fab-clarify intake`**, parsed as a change-name substring.
- `src/kit/skills/fab-proceed.md:163` — fab-new subagent dispatch has no defined behavior when SRAD must ask the user (promptless context, no [AUTO-MODE]); `src/kit/skills/_review.md:53` — inward sub-agent's change_type skip condition has no defined input.
- `src/kit/skills/fab-new.md:50` — backlog-ID collision pre-check is substring-based; a false-positive match silently skips creation.
- `src/kit/skills/fab-proceed.md:89/91` + `_cli-external.md:59-63` — /git-branch chained after fab-new is a stale no-op since #322's inline branch creation; the operator spawn flow misstates the worktree's branch.
- `src/kit/skills/fab-ff.md:32` / `fab-fff.md:32` — `{driver}` row claims it is "passed to every fab status event command", contradicting _pipeline's deliberate driver-less fail/recovery commands (history-shape divergence); `_pipeline.md:87-92` — "requirements mismatch" routes to two different rework paths.

**Action:** give /git-pr and /git-pr-review an explicit change argument plus a branch-matches-change guard (or a fab-fff Step-4 precondition STOP), then rewrite every dispatch string to a single parseable command and ID-anchor the resolution pre-checks.
**Slug:** `ship-dispatch-target-guard`

### Theme 3 — Autonomous git skills lack state guards and honest failure semantics

The no-questions-asked skills (git-pr, git-pr-review, git-branch, fab-archive) assume a clean, attached, main-defaulted, local-only world.

Findings:
- `src/kit/skills/git-pr.md:104-121, 156-163` — **detached HEAD passes the branch guard, then autonomously commits and emits a refspec-less push** (must-fix).
- `src/kit/skills/git-pr-review.md:139-149, 180` — **"(no partial state)" is false on push rejection** (must-fix + should-fix convergence): `git reset` cannot undo the commit, and the re-run path declares "No changes needed" / posts "Fixed" replies citing an unpushed SHA — fixes permanently stranded.
- `src/kit/skills/git-pr.md:143-150` — autonomous `git add -A` sweeps every untracked repo file into a pushed commit.
- `src/kit/skills/git-pr.md:70-127` — `has_pr` ignores PR state; a closed/merged PR short-circuits creation (`state`/`number` fetched, never read).
- `src/kit/skills/git-branch.md:53-62, 85-90` — ambiguous multi-match silently creates a junk standalone branch with a false message; remote-only branches get recreated divergent instead of tracked.
- `src/kit/skills/fab-new.md:154-160` / `git-branch.md:140` — dirty working tree silently rides into the new change's branch (caveat covers committed work only).
- `src/kit/skills/fab-operator.md:429-445, 485, 499` — cherry-pick/rebase hardcode `origin/main`, no fetch step — autopilot unusable on non-main-default repos.
- `src/kit/skills/fab-archive.md` — archive/restore move tracked files and edit `fab/backlog.md` with no commit step; the "safe" claim contradicts the dirty tree; the archive-ok/backlog-mark-failed exit has no recovery path (re-run can never mark the backlog).
- `src/kit/skills/git-pr.md:106` — branch guard checks literal `main`/`master`, not the actual default branch.

**Action:** one git-state hardening change: detached-HEAD STOP, push-failure split (keep commit + documented recovery + unpushed-commit check in the re-run gate), PR-state branching (OPEN/MERGED/CLOSED), `--track origin/<branch>` checkout, multi-match disambiguation STOP, default-branch resolution helper.
**Slug:** `git-state-hardening`

### Theme 4 — `review-pr: failed` is a dead end; state vocabulary drifts from status.go

Three independent lenses confirmed the post-PR failure state has no exit, and skill state enumerations disagree with the Go machine.

Findings:
- `src/kit/skills/fab-continue.md:42-59` — **no dispatch row for review-pr/failed** (3 convergent should-fix findings): preflight at that state matches "all done → Change is complete", mis-reporting a failed PR review as complete; `/fab-continue review-pr` as reset target then errors at the CLI (reset From excludes `failed`).
- `src/kit/skills/fab-continue.md:193` — Reset Flow errors when the target stage is already `active` — non-idempotent re-run (constitution Principle III violation).
- `src/kit/skills/fab-continue.md:204` — intake.md-missing error points to plain /fab-continue, an infinite pointer loop.
- `src/kit/skills/fab-continue.md:85` — reset From-set omits `skipped` (3 convergent findings); `:57-58` — ship/review-pr rows cite an unreachable `ready` state.
- `src/kit/skills/fab-switch.md:95` — display_state qualifiers omit `ready` (the standard state of every freshly switched draft) and `skipped`; `fab-status.md:46` — legend has no `skipped` glyph.
- `src/kit/skills/git-pr-review.md:227 vs 180/185` — Rules "fail fast … stop immediately" contradicts batch-1's route-through-Step-6 design; Step 6's "processing error" outcome is orphaned; phase tracking never fires on the Phase-2 Copilot path; `git-pr.md:35` — Step 0a "no-op" claim is actually a suppressed non-zero error on the canonical path.
- `docs/specs/skills/SPEC-_pipeline.md:20` — **PR-meta rationale is false**: the fail+reset choreography's cascade deletes `stage_metrics.review`, so PR meta always reports "1 cycle" — the choreography zeroes the very counter it cites as payoff (Go fix: preserve Iterations or derive from `.history.jsonl`).
- `src/kit/skills/_review.md:66/75` — Deletion Candidates replace rule is rework-scoped; plain re-runs duplicate the section; `fab-continue.md:184-185` — hydrate's pattern capture sequenced after the finish, unreachable on resume.

**Action:** one fab-continue change adding the review-pr/failed row (re-execute /git-pr-review; its Step 0 `start` already handles failed→active), an already-active reset skip, and a single pass aligning every state enumeration to status.go AllowedStates; fix the metrics cascade in Go.
**Slug:** `review-pr-failed-recovery`

### Theme 5 — Scaffold and config rot: dead keys shipped to every new project, live keys documented nowhere

Findings:
- `src/kit/scaffold/fab/project/config.yaml:31-38` — **scaffold still seeds `stage_directives.spec`** (must-fix, 2 convergent findings): new projects never run the relocating migration, so the zombie key is permanent, the GIVEN/WHEN/THEN defaults are dead, and the relocated `apply:` directive "Mark ambiguities with [NEEDS CLARIFICATION]" (also live in this repo's own `fab/project/config.yaml:19-21`) directly contradicts `src/kit/templates/plan.md:36`.
- `src/kit/skills/fab-setup.md:167` — **`stage_directives` is a dead key end-to-end**: scaffold promises it, fab-setup edits it, three migrations preserve it, zero readers exist in skills or Go. `model_tiers` likewise has zero consumers.
- `src/go/fab/internal/config/config.go:18` / `status.go:603-621` — **`stage_hooks` is live, Go-consumed (pre blocks start; post runs after save), and documented nowhere** — including the re-run trap where a failing post-hook leaves the stage done and the documented re-run hits done→done.
- `src/kit/schemas/workflow.yaml` — **still defines the 7-stage pipeline with the spec stage** (must-fix); nothing consumes it, yet `docs/specs/user-flow.md:201` calls it source of truth.
- `src/kit/scaffold/fab/project/code-review.md:44` (+ this repo's staler local copy) — escalation names the removed "revise spec" path; the "Max cycles: 3" knob is consumed by nothing (_pipeline hard-codes 3).
- `src/kit/skills/fab-setup.md:303 vs :294` — **#393's f080 dedup deleted the Semver Comparison rule the new three-way branch needs**; `:95/:153` — the fab_version "guarantee" has no create-mode fallback; `:85-97` — sync's settings/hooks/direnv side effects unenumerated; `:430-436` — Next Steps lines drift from the State Table they claim to derive from.
- `src/kit/skills/_preamble.md:40` — always-load descriptor still advertises removed `naming` and dead `model tiers`.

**Action:** a scaffold-truth change: fix the spec key (move directives under `apply:` with the marker directive removed or moved to `intake:`), decide wire-or-remove for `stage_directives` and `stage_hooks` (see Structural bets), regenerate-or-retire workflow.yaml, restore a one-line semver rule, and clean this repo's local config/code-review copies.
**Slug:** `scaffold-config-truth`

### Theme 6 — SRAD scoring layer is internally inconsistent (math, bands, and the clarify escape valve)

The scoring rubric agents pattern-match has holes that move real intake scores across the 3.0 gate.

Findings:
- `src/kit/skills/fab-clarify.md:55 vs 61-73` — **Step 1.5 zero-gaps early exit makes bulk confirm unreachable in its primary scenario** (must-fix): a Confident-only intake (no markers) dead-ends at "artifact looks solid" while the score sits below the gate.
- `docs/specs/srad.md:54-118` — **Assumptions-table contract contradicts _srad** (must-fix): Scores column "optional", Certain/Unresolved rows omitted — tables `fab score` cannot fully parse, deflating the gate inputs.
- `src/kit/skills/_srad.md:30` — closed integer bands vs continuous composite: 59.85/84.5 match no grade; boundary is worth 0.7 score per row.
- `src/kit/skills/_srad.md:71` — Worked Example 3's Certain grade is mathematically unreachable (cap 84.75 < 85).
- `src/kit/skills/_srad.md:47 vs 30` — Critical Rule has two competing numeric definitions (<25 override vs 0–39 Low band).
- `docs/specs/srad.md:246-248` — Example 1 composite arithmetic wrong for two rows.
- `src/kit/skills/fab-clarify.md:110-118` — S→95 upgrade labels rows Certain below the 85 threshold; `:122/:153` — audit-trail placement/append rules asymmetric.
- `src/kit/skills/fab-new.md:182-189` — Output ordering violates _srad's Assumptions-block-last SHALL; `_generation.md:75-78` — plan walk never emits the `## Assumptions` section it depends on, and _srad's omit-when-zero rule conflicts with the scaffolded templates.
- `src/kit/skills/_srad.md:51-57` — autonomy table covers 4 of the 6 declaring skills.

**Action:** one scoring-coherence change: half-open thresholds (≥85/≥60/≥30/else), one Critical-Rule number used everywhere, re-dimension Example 3, fix srad.md's contract + arithmetic, evaluate fab-clarify's bulk-confirm trigger before the early exit, and add the explicit Assumptions step to the plan walk.
**Slug:** `srad-scoring-coherence`

### Theme 7 — The documentation layer describes the previous system

Constitution-mandated SPEC mirrors and the docs/specs corpus lag one or more architecture generations behind. Three sub-clusters:

(a) **SPEC-hooks is fiction end-to-end** — `docs/specs/skills/SPEC-hooks.md`: Current Hooks lists two deleted shell scripts (must-fix); the "Proposed Hook Architecture" presents shipped Go behavior as future work (must-fix); the events table rates UserPromptSubmit "No" while that hook is registered (must-fix); plus superseded `fab runtime` proposal, dead yq inventory, stale phase list, outdated runtime schema (4 should-fix).

(b) **Legacy non-skill docs** — `docs/specs/assembly-line.md:121` narrates spec/tasks stages (must-fix); `architecture.md` is built on the pre-binary `.kit/` distribution model and contradicts its own Router Dispatch section (structural); `overview.md` tells a 4-stage story and omits /git-pr, /git-pr-review, /fab-proceed, /fab-operator; `glossary.md:49/115` defines auto-clarify behavior fab-ff disclaims; `user-flow.md:84/183` says failed is "review only"; `templates.md` carries the pre-1.10.0 intake template and a .status.yaml block missing `ready`/`id`/`issues`/`prs`; `operator.md:11` says "current operator (v8)" above its own v9 row; `skills.md:583` documents the pre-date-bucketing archive path.

(c) **SPEC skill mirrors** — ~18 mirrors drifted through the batches, e.g. SPEC-fab-operator (status-only-mode + rejected Decision 2 as current), SPEC-_preamble (misquoted opening instruction ×2, dead kit.conf row ×2), SPEC-fab-proceed (self-contradiction on _preamble loading), SPEC-fab-clarify (removed [target-artifact] flow, `fab score` missing `--stage`), SPEC-fab-continue (writes a removed "Spec" artifact, claims forbidden `fab score` use), SPEC-fab-archive (preflight/hydrate guard applied to both modes — #393 f087 regression), SPEC-git-pr, SPEC-git-pr-review (×2), SPEC-fab-status, SPEC-fab-help, SPEC-fab-new, SPEC-docs-hydrate-specs (phantom modify/index paths), SPEC-docs-hydrate-memory (three-way exemption contradiction), SPEC-_review (spec/plan phrasing), SPEC-docs-reorg-memory (wrong Kind tokens), SPEC-fab-discuss.

**Action:** three changes, in order: rewrite SPEC-hooks as-shipped (`spec-hooks-rewrite`); expand the deferred uliv sweep to cover architecture/assembly-line/overview/glossary/user-flow/templates/srad/workflow (`legacy-docs-truth-sweep`); a mechanical mirror resync driven by the constitution's skill→SPEC rule (`spec-mirror-resync`).
**Slug:** `spec-docs-reality-sweep` (umbrella)

### Theme 8 — Memory/specs maintenance skills: index ownership and placement rules are one-sided

Findings:
- `src/kit/skills/docs-reorg-memory.md:125-126` — **Step 5.3 instructs editing a sub-domain index.md that doesn't exist until Step 5.4 generates it — and Step 5.4 forbids the edit** (must-fix). The stub pattern it needs already exists at docs-hydrate-memory.md:69.
- `src/kit/skills/docs-hydrate-memory.md:124-149` — generate mode has no placement rules (target path, domain creation, index stub, shape bounds all live only under ingest); `:69/81-83` — sub-domain index stubs never instructed; the memory-index tier description omits sub-domain indexes in 3 locations (incl. fab-continue.md:183).
- `src/kit/skills/docs-hydrate-specs.md:64` — no branch for a gap with no suitable target spec file; Step 6 handles a "skip rest" token Step 5 never offers.
- `src/kit/skills/docs-reorg-memory.md:23/56/78-84` — depth off-by-one between the ≤3 bound (path segments) and the report's folder-depth column; dangling-link hard block has no abort/rollback escape.
- `src/kit/skills/docs-reorg-specs.md:12-35` — no reserved-path exemption for the constitution-pinned SPEC mirrors; recursion into subfolders undefined.

**Action:** define index ownership once — `description:` frontmatter is the single hand-curated field; stub is created *before* `fab memory-index` — and propagate to hydrate (both modes), reorg, and the SPECs; add the no-target and reserved-paths branches.
**Slug:** `memory-index-ownership`

---

## 3. Quick wins

Two bundles of small, isolated edits; each fits comfortably in one change.

**Bundle A — stale pointers, counts, and wording** (slug: `stale-pointer-sweep`):
- `fab-operator.md:23` — "(see §5)" → "(see §6)" (2 convergent findings; only bad §-pointer in the file).
- `fab-proceed.md:109` — "(see Output Format)" → "(see Bypass Notes)" (heading exists only in the SPEC).
- `fab-continue.md:149` — "Review Behavior" → a heading that exists in `_review.md` (the one dangling heading pointer corpus-wide).
- `fab-operator.md:226-252` — status-frame example: "7 tracked" vs 8 entries; `gmail-deploys` watch has no schema-valid source. Also `:192` — drop the stale "until the operator session ends" branch_map retention clause.
- `git-pr-review.md:105` — drop the unconsumed `node_id` from the jq projection (+ SPEC line 55); `:11/64` — reword the `--tool` header ("bypasses automatic detection" is false; "the cascade" is undefined residue).
- `docs-hydrate-specs.md:70/76` — align the yes/no/done prompt with the four-token handler.
- `fab-help.md:12` — Purpose understates the output (git-*, docs-*, batch, packages); `internal-retrospect.md` — add the missing H1; `_cli-external.md:34` — `--reuse` requires `--worktree-name`; `operator.md:11` — v8 → v9; `_generation.md:17` — drop "auto-clarify"; `fab-clarify.md:182` — protocol example cites a removed flow; `fab-discuss.md:33` — state the stage-derivation rule for `.status.yaml`; `fab-switch.md:29-33` — add the missing "run the switch after selection" step; `fab-status.md:78` — preflight does require config/constitution to exist; `fab-proceed.md:74/108` — `sort -r` on full folder names for the same-day tiebreak; `git-pr.md:220-222` — reorder the --fill fallback vs STOP branches; `_cli-fab.md:175-178` — document artifact-write's git auto-staging.
- Process hygiene: check the 9u91/uliv/zc9m/szxd backlog boxes and archive the four merged-but-unarchived change folders.

**Bundle B — enumeration completion** (slug: `enumeration-completion-sweep`):
- `_preamble.md:36` — Always-Load exceptions list misses /fab-proceed, /fab-help, /fab-archive, /docs-hydrate-specs, /docs-reorg-* (3 convergent findings); give docs-hydrate-memory the Context Loading section the rule keys on.
- `_preamble.md:301` — Subagent Dispatch orchestrator list omits /fab-proceed; `:355-361` — Confidence Scoring invokers omit /fab-draft and mis-scope clarify's recompute (2 findings).
- `internal-skill-optimize.md:15/21/86` (+ SPEC) — all partial enumerations omit `_pipeline` (2 convergent should-fix findings; this list has now gone stale twice).
- `_generation.md:3/11-13` (+ SPEC:5) — fab-continue belongs to both consumer groups (2 findings).
- `_cli-fab.md:27` — in-file index omits migrations-status and memory-index; `fab-draft.md:30/48`, `fab-new.md:223`, `fab-setup.md:432-436` — Next: lines omit /fab-proceed; `_srad.md:51-57` — one-line note for fab-draft/fab-clarify.
- Root-cause recommendation: wherever possible replace enumerations with derivation rules (e.g., "every `_*.md` file is a shared partial — reference, never target"; "derive Next: at runtime per the _preamble Lookup Procedure") so these can't drift a third time.

---

## 4. Structural bets

Each is worth a design discussion, not a drive-by fix.

1. **`fab status render` Go subcommand** (`fab-status.md:33-66`, 2 convergent findings). The skill is ~90% deterministic formatting (hard-coded thresholds, exact warning strings, version-drift compare) re-derived by an LLM per glance and mirrored in the SPEC. Precedent: pr-meta, fab-help, memory-index. Resolves known-open f091/f172/f201 at the root. *Tradeoff:* tension with constitution Principle I (logic in markdown), Go surface + tests + _cli-fab section; format iteration moves to binary releases.
2. **`fab operator frame` subcommand** (`fab-operator.md:216-287`). The status frame is fully mechanical given inputs the binary already owns (pane map JSON + server-keyed state file). Byte-stable frames, defined failure surface for the tick. *Tradeoff:* the natural-language stuck-threshold override needs a `--stuck` flag; frame changes need Go releases.
3. **Conditional loading for operator autopilot/watches** (`fab-operator.md` §6–§7, ~21KB of 49.8KB re-paid every /clear). Extract to `_operator-autopilot.md`/`_operator-watches.md` read only when the state file has a queue/watches. *Tradeoff:* two more internal partials and an agent-compliance dependency — needs backstop read lines like fab-continue's.
4. **`stage_directives` / `stage_hooks`: wire or remove** (Theme 5). Either _generation/_review consume `stage_directives.{stage}` and `_cli-fab` documents `stage_hooks`, or both leave config.go, the scaffold, and the fab-setup menu. *Tradeoff:* wiring adds prompt surface to every generation; removal needs a migration for projects that populated them.
5. **Retire `workflow.yaml`** rather than regenerate. Nothing consumes it and it has already drifted a full pipeline generation unnoticed; repoint user-flow.md and memory at the Go state machine. *Tradeoff:* loses the one declarative schema artifact; user-flow.md needs a new source-of-truth anchor.
6. **Hoist the shared ff/fff Output frame into `_pipeline`** (residual twin drift: header wording, apply annotation, duplicated resuming sentence, force-mode header underspecified). *Tradeoff:* _pipeline grows beyond behavior steps and `{driver}` substitution inside output templates is a new pattern; alternative is byte-for-byte alignment with keep-in-sync comments (the accepted f032 pattern). Same decision applies to the fab-continue/_pipeline rework-path twins.
7. **Generate-mode scan scope from `source_paths`** (`docs-hydrate-memory.md:91`) — mirror internal-consistency-check; folds into adding the skill's missing Context Loading section. *Tradeoff:* adds a config read to a previously config-free skill.
8. **`_srad` stage-conditional in fab-continue** (frontmatter → in-body read alongside _generation). ~6KB saved on hydrate/ship/review-pr/resume invocations. *Tradeoff:* touches the helper-model claims in _srad's header, the SPEC, and the preamble; one more compliance-dependent read.

---

## 5. Verified clean

So the coverage is clear — these areas were swept and held up:

- **#393's `_pipeline` bracket extraction**: the behavior steps themselves are coherently single-sourced; no contradiction was found *inside* the bracket. All residual ff/fff drift is in the per-driver framing (Purpose/Arguments/Output), severity nice-to-have.
- **Cross-reference integrity at large**: all ~25 of fab-operator's §-pointers resolve except the one §5/§6 mispointer; exactly one dangling heading pointer exists corpus-wide (fab-continue → _review "Review Behavior"). The post-#392 pointer discipline otherwise held.
- **Prior batch fixes verified in place**: f019's review/failed dispatch row, f051's complete _srad Placed-by list, f062's current intake template (its `templates.md` mirror is what lags), batch-1's fabhelp.go six-stage pipeline + group map.
- **Batch-2 naming cleanup held**: no statusman/changeman/logman residue in any `src/kit/skills/` source — the remaining script-name residue is confined to the docs files already enumerated in the deferred uliv sweep.
- **Migrations are correct**: 1.9.7→1.10.0 properly relocates `stage_directives.spec` for existing projects — the must-fix gap is the scaffold (never migrated), not the migration logic.
- **SRAD worked examples partially sound**: srad.md Example 2 and Example 1 row 1 arithmetic are correct; the errors are two composites in Example 1 and the structural Example 3 issue, with grade outcomes mostly unaffected.
- **docs/memory content**: no confirmed findings — drift findings all target skills, SPECs, docs/specs, scaffold, schemas, and the Go CLI.
- **Verification rigor**: many headline findings (gate exit code, advance bricking, resolve --folder, archive exit paths, remote-branch divergence) were reproduced empirically against a built binary or sandbox repos; refuted and duplicate raw findings were dropped before this synthesis, and the verifier downgraded one headline (operator autopilot double-dispatch: must-fix → should-fix), which is reflected above.

---

## Appendix A — Confirmed findings index

| ID | Sev | File | Location | Title |
|---|---|---|---|---|
| a001 | structural | `docs/specs/architecture.md` | § Directory Structure (lines 9-71), § Agent Integration (374-402), § Distribution & Bootstrapping (406-453), § Updating .kit/ (457-476); contradicted by the same file's § Router Dispatch (line 482) | architecture.md is built on the pre-binary fab/.kit distribution model the constitution explicitly replaced |
| a002 | must-fix | `docs/specs/assembly-line.md` | § How It Works step 2 (line 121); also intro line 5 | assembly-line.md still narrates the removed spec/tasks stages as part of the pipeline |
| a003 | should-fix | `docs/specs/glossary.md` | § Skills, /fab-ff row (line 49); repeated in § Workflow Concepts 'Fast-forward' (line 115) | glossary.md describes /fab-ff as running 'auto-clarify between planning stages' — a mechanism fab-ff explicitly disclaims |
| a004 | should-fix | `docs/specs/operator.md` | § Version History, line 11 vs table row line 23 | operator.md prose says 'current operator (v8) … eight iterations' but its own table ends at v9 (seed finding, confirmed) |
| a005 | structural | `docs/specs/overview.md` | heading at line 37; Quick Reference table lines 81–103 | overview.md tells a 4-stage story the constitution, fab-help, and glossary contradict, and its Quick Reference omits /git-pr, /git-pr-review, /fab-proceed, and /fab-operator |
| a006 | structural | `docs/specs/skills.md` | ## `/fab-archive [<change-name>]`, Behavior step 1, line 583 | skills.md /fab-archive section documents the pre-date-bucketing archive path the Go CLI no longer uses |
| a007 | should-fix | `docs/specs/skills/SPEC-_pipeline.md` | ## Per-Cycle Rework Choreography (f071), item 1 (line 20) | SPEC's PR-meta rationale is false: the fail+reset choreography wipes stage_metrics.review.iterations every cycle, so PR meta always reports 1 review cycle |
| a008 | should-fix | `docs/specs/skills/SPEC-_preamble.md` | ## Summary, closing paragraph (line 7) | SPEC mirror cites a stale opening instruction that no skill uses |
| a009 | should-fix | `docs/specs/skills/SPEC-_preamble.md` | ### Tools used table (line 105) | SPEC Tools-used table still lists kit.conf build guard, eliminated in 260402-gnx5 |
| a010 | should-fix | `docs/specs/skills/SPEC-_preamble.md` | ### Tools used, line 105 | SPEC-_preamble Tools table cites a dead "kit.conf (build guard)" read |
| a011 | nice-to-have | `docs/specs/skills/SPEC-_preamble.md` | ## Summary, line 7 | SPEC-_preamble misquotes the canonical per-skill opening instruction |
| a012 | nice-to-have | `docs/specs/skills/SPEC-_review.md` | Summary (line 5) and Sub-agents section (line 110) | SPEC-_review retains pre-1.10.0 'spec/plan' phrasing for what the inward sub-agent validates against |
| a013 | should-fix | `docs/specs/skills/SPEC-docs-hydrate-memory.md` | Flow diagram, line 12 (vs _preamble.md §1 line 36 and the skill file, which has no Context Loading section) | Three-way contradiction on docs-hydrate-memory's always-load exemption: SPEC says partial skip, preamble says entire skip, skill file is silent |
| a014 | should-fix | `docs/specs/skills/SPEC-docs-hydrate-specs.md` | Flow diagram lines 22 and 25; Tools table line 33 | SPEC-docs-hydrate-specs flow drifted from the skill: phantom 'modify' option, phantom spec-index edit and new-file creation |
| a015 | nice-to-have | `docs/specs/skills/SPEC-docs-reorg-memory.md` | ## Summary (line 5) and § Link Impact (line 19) | SPEC mirror uses Kind tokens `split`/`merge` where the skill's Migration Map enum is `split-domain`/`merge-domain` |
| a016 | should-fix | `docs/specs/skills/SPEC-fab-archive.md` | ## Flow diagram, lines 11-31 | SPEC-fab-archive flow diagram applies preflight + hydrate guard to both modes, contradicting the skill and the SPEC's own Summary |
| a017 | should-fix | `docs/specs/skills/SPEC-fab-clarify.md` | Flow diagram lines 12, 24, 30-49, 43 vs Bookkeeping table line 71 | SPEC-fab-clarify Flow diagram retains removed [target-artifact] argument, {artifact}.md placeholders, and a fab score call missing --stage |
| a018 | should-fix | `docs/specs/skills/SPEC-fab-continue.md` | ### Tools used, lines 137-139 | SPEC mirror Tools table is stale: writes a removed 'Spec' artifact and claims fab score usage the skill forbids |
| a019 | nice-to-have | `docs/specs/skills/SPEC-fab-discuss.md` | Flow (lines 9-19) and Tools used table (lines 21-26) | SPEC-fab-discuss flow and tools table omit the conditional .status.yaml read for the active change's stage |
| a020 | should-fix | `docs/specs/skills/SPEC-fab-help.md` | Flow diagram, line 14 | SPEC-fab-help flow claims the Go subcommand scans src/kit/skills, but fabhelp.go scans the kit cache sibling to the binary |
| a021 | nice-to-have | `docs/specs/skills/SPEC-fab-new.md` | ## Flow, Step 3 (line 46) | SPEC flow's Step 3 command line omits the conditional --change-id flag central to the backlog collision story |
| a022 | should-fix | `docs/specs/skills/SPEC-fab-operator.md` | Key Properties table ('Requires tmux?' row) and Section Structure item 2 ('outside-tmux degradation') vs skill §2 Tmux Gate and §9 (line 625) | SPEC claims a 'status-only mode without' tmux; the skill mandates a hard stop |
| a023 | should-fix | `docs/specs/skills/SPEC-fab-operator.md` | Resolved Design Decisions item 2; also 'operator4' residue in Primitives intro and 'loaded in the operator's own startup section' in Summary | SPEC Resolved Design Decision 2 ('All-auto-answer over two-tier classification') contradicts the current rule-4 Routine/Strategic two-tier model |
| a024 | should-fix | `docs/specs/skills/SPEC-fab-proceed.md` | Summary (line 5) vs 'Key differences from /fab-fff and /fab-ff' (line 114) | SPEC mirror contradicts itself (and the skill) on whether /fab-proceed loads _preamble |
| a025 | should-fix | `docs/specs/skills/SPEC-fab-status.md` | ## Flow line 13 + Tools used table (lines 38-41) | SPEC-fab-status cites the dev-repo path src/kit/VERSION and omits the .status.yaml read the skill depends on |
| a026 | should-fix | `docs/specs/skills/SPEC-git-pr-review.md` | ### Status-commit bookkeeping (Step 6.5), Gate row (line 125) vs flow diagram line 87 | SPEC Status-commit bookkeeping Gate row omits the timeout path, contradicting both the skill and the SPEC's own flow diagram |
| a027 | nice-to-have | `docs/specs/skills/SPEC-git-pr-review.md` | Flow diagram, phase-tracking footer (line 94) | SPEC phase chain starts with a 'waiting' phase that nothing in the kit or Go CLI ever sets |
| a028 | nice-to-have | `docs/specs/skills/SPEC-git-pr.md` | Summary (line 5) and Flow Step 0b (lines 24–27) | SPEC-git-pr omits the explicit /git-pr {type} argument from the type-resolution chain — mirror lists 3 sources, skill has 4 |
| a029 | must-fix | `docs/specs/skills/SPEC-hooks.md` | ## Current Hooks, lines 5-14 | 'Current Hooks' section describes a hook system that no longer exists (2 shell scripts vs 5 Go subcommands) |
| a030 | must-fix | `docs/specs/skills/SPEC-hooks.md` | ## Possible Events to Use, lines 91-92 | Events table rates UserPromptSubmit fab-kit fit 'No' while a UserPromptSubmit hook is shipped and registered |
| a031 | must-fix | `docs/specs/skills/SPEC-hooks.md` | ## Proposed Hook Architecture (Trimmed) + ### Registration in 5-sync-hooks.sh, lines 186-247 (also lines 31-50 framing) | 'Proposed Hook Architecture' and 'Registration in 5-sync-hooks.sh' present already-shipped behavior as future work, citing dead artifacts |
| a032 | should-fix | `docs/specs/skills/SPEC-hooks.md` | ## Hooks Embedded in Skills, line 24 | git-pr-review phase list stale: 'waiting' phase gone, 'replying' phase missing |
| a033 | should-fix | `docs/specs/skills/SPEC-hooks.md` | ## yq Dependency in Hooks, lines 153-169 | 'yq Dependency in Hooks' section inventories three src/kit files that no longer exist; hook yq usage is zero |
| a034 | should-fix | `docs/specs/skills/SPEC-hooks.md` | ### Proposal: fab runtime subcommands, lines 171-180 | 'Proposal: fab runtime subcommands' superseded — no `fab runtime` command exists; hooks call the internal runtime package directly |
| a035 | should-fix | `docs/specs/skills/SPEC-hooks.md` | ### What this changes in skills, lines 210-221 (vs line 52) | 'What this changes in skills' table contradicts shipped skill state and the SPEC's own 1.10.0 note (fab-clarify/fab-new rows) |
| a036 | nice-to-have | `docs/specs/skills/SPEC-hooks.md` | ## Current Hooks, line 12 | Runtime-file schema description outdated: 'agent.idle_since' singleton vs per-session '_agents[session_id]' map with GC |
| a037 | must-fix | `docs/specs/srad.md` | § Dimension Score Persistence (line 54, table lines 57-60) and § Assumptions Summary (lines 105, 110-118) | srad.md Assumptions-table contract contradicts _srad: Scores column 'optional' and Certain/Unresolved rows omitted |
| a038 | should-fix | `docs/specs/srad.md` | § Worked Examples, Example 1 table (lines 246-248) | srad.md Example 1 composite arithmetic is wrong for two of three rows under the documented formula |
| a039 | should-fix | `docs/specs/templates.md` | ## .status.yaml — State Vocabulary table (lines 13-19) and Template block (lines 27-54) | docs/specs/templates.md .status.yaml section drifts from the shipped template and the Go state machine |
| a040 | should-fix | `docs/specs/templates.md` | ## intake.md section, lines 91-137 | docs/specs/templates.md intake.md section is the pre-1.10.0 template — BLOCKING/DEFERRED labels and New/Modified/Removed subsections no longer exist |
| a041 | should-fix | `docs/specs/user-flow.md` | §3 intro (line 84) and §4 diagram note (lines 182-184) | user-flow.md claims failed is 'review only' and lists five stage states, contradicting the workflow schema it cites as source of truth |
| a042 | should-fix | `src/go/fab/internal/config/config.go` | Config struct, line 18; consumed at internal/status/status.go:603-621 | stage_hooks is a live Go-consumed config key documented nowhere in the kit |
| a043 | should-fix | `src/go/fab/internal/status/status.go` | AllowedStates (lines 21-28) vs defaultTransitions/stageTransitions (lines 37-61); Validate (lines 502-535) | Go state machine permits transitions into states its own schema forbids (advance ship/review-pr → ready; skip intake → skipped), bricking preflight |
| a044 | should-fix | `src/kit/scaffold/fab/project/code-review.md` | ## Rework Budget (line 44) | Scaffold code-review.md rework escalation still names the removed-era "revise spec" path (dev-repo copy says "revise tasks"/"revise spec"), and the "Max cycles" knob has no consumer |
| a045 | should-fix | `src/kit/scaffold/fab/project/code-review.md` | ## Rework Budget, lines 38-44 | Scaffold code-review.md Rework Budget names a nonexistent 'revise spec' escalation path, and its Max-cycles knob is consumed by nothing |
| a046 | must-fix | `src/kit/scaffold/fab/project/config.yaml` | stage_directives block, lines 31-38 | Scaffold config template still seeds the dead `spec:` stage in stage_directives — every new project gets a zombie key and loses the apply directives |
| a047 | must-fix | `src/kit/scaffold/fab/project/config.yaml` | stage_directives block, lines 31-38 | Scaffold config.yaml seeds a dead stage_directives.spec key whose [NEEDS CLARIFICATION] directive contradicts the apply-stage contract |
| a048 | must-fix | `src/kit/schemas/workflow.yaml` | stages list (lines 64-82) and stage_numbers (lines 212-219) | workflow.yaml still defines the 7-stage pipeline with the removed spec stage |
| a049 | should-fix | `src/kit/skills/_cli-external.md` | § Operator Spawning Rules → 'New change (from backlog)', lines 59-63 | _cli-external 'New change (from backlog)' spawn flow is stale: unconditional /git-branch step ignores fab-new's inline branch creation, and step 1 misstates the worktree's branch |
| a050 | should-fix | `src/kit/skills/_cli-external.md` | § wt — Operator Spawning Rules, "New change (from backlog)" (line 61) | _cli-external: wt new-change flow claims the fresh worktree "creates on default branch" — contradicted by both git's checkout-exclusivity rule and wt's actual behavior |
| a051 | nice-to-have | `src/kit/skills/_cli-external.md` | § `wt create` Flags table, line 34 | wt create --reuse documented without its --worktree-name requirement |
| a052 | should-fix | `src/kit/skills/_cli-fab.md` | ## fab batch, line 494 | _cli-fab claims `fab batch archive` 'exits non-zero only when failed > 0', but it also exits 1 on empty/unresolvable sets with failed == 0 |
| a053 | should-fix | `src/kit/skills/_cli-fab.md` | ## fab change (extended subcommand details), lines 42, 46 (also fab-archive.md:67 and fab-archive.md:110 'Idempotent? Yes — re-archive is a soft skip') | Documented re-archive soft skip (exit 0) is unreachable — genuinely archived changes exit 1 with 'No change matches' |
| a054 | should-fix | `src/kit/skills/_cli-fab.md` | ## fab hook, line 168 | "All hook subcommands exit 0" is false for `fab hook sync` |
| a055 | should-fix | `src/kit/skills/_cli-fab.md` | ## fab status (extended subcommand details), "Side effects of `finish`" paragraph (line 78); cross-ref § fab impact Consumers (line 291) | Undocumented config-driven stage_hooks (pre/post) run by fab status start/finish, with after-save failure semantics |
| a056 | should-fix | `src/kit/skills/_cli-fab.md` | § fab status (extended subcommand details), advance row (line 58); also fab-continue.md dispatch rows 57-58 | fab status advance on ship/review-pr writes a schema-forbidden 'ready' state that permanently bricks preflight for the change |
| a057 | nice-to-have | `src/kit/skills/_cli-fab.md` | ## fab change (extended subcommand details), archive output paragraph (line 46) | Archive partial-failure outcome (YAML on stdout + non-zero exit) undocumented |
| a058 | nice-to-have | `src/kit/skills/_cli-fab.md` | ## fab change (extended subcommand details), line 42 (Usage column) | `fab change archive` with no argument prints help and exits 0 — the documented required <change> guard is silently passable |
| a059 | nice-to-have | `src/kit/skills/_cli-fab.md` | ## fab hook, artifact-write row (line 175) and paragraph (line 178) | artifact-write's git auto-staging side effect omitted from the hook table |
| a060 | nice-to-have | `src/kit/skills/_cli-fab.md` | ### Commands covered in `_preamble` Common fab Commands (line 27) | _cli-fab's own 'remaining commands' index omits fab migrations-status and fab memory-index |
| a061 | should-fix | `src/kit/skills/_generation.md` | § Plan Generation Procedure steps 1–7 (esp. step 3 bullet, lines 75–78) vs. § Intake Generation Procedure step 4 (line 41) | Plan Generation walk never emits the plan's ## Assumptions section it depends on; Assumptions handling is asymmetric and conflicts with the scaffolded templates on the zero-assumptions case |
| a062 | nice-to-have | `src/kit/skills/_generation.md` | Header blockquote (lines 11-13) | _generation consumer routing omits fab-continue's Intake Generation paths |
| a063 | nice-to-have | `src/kit/skills/_generation.md` | header Orchestration note, line 17 | Stale 'auto-clarify' term in the orchestration carve-out — no consumer has an auto-clarify step since 1.10.0 |
| a064 | nice-to-have | `src/kit/skills/_generation.md` | header note lines 11–13 and frontmatter description (line 3) | Consumer mapping is stale again: fab-continue now also runs the Intake Generation Procedure (intake regeneration), so the 'disjoint consumer groups' claim is wrong |
| a065 | must-fix | `src/kit/skills/_pipeline.md` | ## Pre-flight, item 3 (line 31); also Shared Error Handling row 'Intake gate fails' | Intake gate is undetectable via the documented contract: `fab score --check-gate` exits 0 on gate fail, while the preamble claims it 'returns non-zero' |
| a066 | must-fix | `src/kit/skills/_pipeline.md` | ### Stop (after 3 failed cycles), closing paragraph (line 106) | Exhaustion-stop recovery guidance is unexecutable: `/fab-clarify intake` parses 'intake' as a change name, is stage-guard-blocked post-intake, and the promised requirements regeneration never happens |
| a067 | should-fix | `src/kit/skills/_pipeline.md` | § Auto-Rework Loop → Decision heuristics, lines 87-92 | _pipeline decision heuristics route 'requirements mismatch' to two different rework paths |
| a068 | must-fix | `src/kit/skills/_preamble.md` | ## Common fab Commands, `fab change <sub>` table row (line 232) | Canonical form `fab change resolve --folder` is an invalid command — the flag does not exist on `fab change resolve` |
| a069 | must-fix | `src/kit/skills/_preamble.md` | § Common fab Commands, `fab score` table row (line 230); same claim in src/kit/skills/_cli-fab.md § fab score (extended), Gate row (line 91) | fab score --check-gate documented as exiting non-zero on gate fail, but the Go binary always exits 0 — the sole ff/fff gate can be silently passed |
| a070 | must-fix | `src/kit/skills/_preamble.md` | § Common fab Commands, fab change row (line 232) | Canonical-form example `fab change resolve --folder` is a broken command — the subcommand has no --folder flag |
| a071 | must-fix | `src/kit/skills/_preamble.md` | § Common fab Commands, fab score row (line 230); also _cli-fab.md § fab score (extended), Gate mode row (line 91) | fab score --check-gate never exits non-zero on gate failure — skills document an exit-code contract the Go binary does not implement |
| a072 | should-fix | `src/kit/skills/_preamble.md` | ## Context Loading > ### 1. Always Load (line 36) | Always-Load 'Current exceptions' list omits /fab-proceed, which declares it skips all context loading |
| a073 | should-fix | `src/kit/skills/_preamble.md` | ## Context Loading > ### 1. Always Load (line 36) | _preamble §1 'Current exceptions' enumeration no longer matches the skill corpus |
| a074 | should-fix | `src/kit/skills/_preamble.md` | § Confidence Scoring → Invocation, lines 355-361 | _preamble Confidence-Scoring invocation list omits /fab-draft and mis-scopes /fab-clarify recompute to suggest mode |
| a075 | should-fix | `src/kit/skills/_preamble.md` | § Context Loading › 1. Always Load, line 36 | _preamble always-load "Current exceptions" list is incomplete — misses 4 skills whose own Context Loading sections deviate |
| a076 | nice-to-have | `src/kit/skills/_preamble.md` | ## Confidence Scoring > ### Invocation (lines 357-361) | Confidence Scoring 'invoked by' list omits /fab-draft, which scores via fab-new's inherited Steps 0–9 |
| a077 | nice-to-have | `src/kit/skills/_preamble.md` | ## Subagent Dispatch (Orchestrator Skills), opening paragraph (line 301) | Subagent Dispatch enumerates orchestrators as exactly /fab-ff and /fab-fff, but /fab-proceed dispatches prefix steps per this section |
| a078 | nice-to-have | `src/kit/skills/_preamble.md` | §1 Always Load file list, line 40 | _preamble always-load descriptor for config.yaml still advertises removed 'naming conventions' and dead 'model tiers' |
| a079 | should-fix | `src/kit/skills/_review.md` | § Inward Sub-Agent Dispatch — Validation Steps, step 7 (line 53) vs. context list (line 35) | Inward sub-agent's Step 7/8 skip condition depends on change_type, which no defined input provides |
| a080 | should-fix | `src/kit/skills/_review.md` | § Inward Sub-Agent Dispatch — step 8, lines 66 and 75 | ## Deletion Candidates replace rule is scoped to rework cycles only — a plain review re-run reads as 'append', duplicating the section |
| a081 | should-fix | `src/kit/skills/_srad.md` | ## SRAD Scoring, Aggregation (line 30) | Grade thresholds are closed integer bands but the composite is continuous — values like 59.85 or 84.5 match no grade |
| a082 | should-fix | `src/kit/skills/_srad.md` | ## Worked Examples, Example 3 (line 71) vs Aggregation (line 30) | _srad Worked Example 3's grade is mathematically unreachable: S:Low + R/A/D:High caps the composite at 84.75, below the 85 Certain threshold |
| a083 | nice-to-have | `src/kit/skills/_srad.md` | ## Critical Rule (line 47) vs Aggregation (line 30) and the dimension table (line 24) | Critical Rule's 'low Reversibility AND low Agent Competence' has two competing numeric definitions (<25 override vs the 0–39 Low band) |
| a084 | nice-to-have | `src/kit/skills/_srad.md` | ## Skill-Specific Autonomy Levels (lines 51–57) | Skill-Specific Autonomy Levels table covers only 4 of the 6 skills that declare _srad — fab-draft and fab-clarify have no column |
| a085 | should-fix | `src/kit/skills/docs-hydrate-memory.md` | ## Generate Mode Behavior → Step 3: Memory File Generation, lines 124-149 | Generate mode Step 3 has no placement rules — no target-path mapping, no domain-index/description stub, no shape bounds |
| a086 | nice-to-have | `src/kit/skills/docs-hydrate-memory.md` | ## Ingest Mode Behavior → Step 3 item 2 (line 69) and Step 4 (lines 81-83) | Sub-domain index description: stub is never instructed, and Step 4 omits sub-domain indexes from what fab memory-index regenerates |
| a087 | nice-to-have | `src/kit/skills/docs-hydrate-memory.md` | Ingest Mode Step 4 (line 83) and Generate Mode Step 4 (line 153); same omission at fab-continue.md Hydrate step 4 (line 183) | fab memory-index tier description diverges: hydrate-side copies omit the sub-domain index tier |
| a088 | structural | `src/kit/skills/docs-hydrate-memory.md` | ## Generate Mode Behavior → Step 1: Codebase Scanning, line 91 | Generate mode defines its own scan scope and ignores config source_paths, unlike internal-consistency-check |
| a089 | should-fix | `src/kit/skills/docs-hydrate-specs.md` | ## Behavior → Step 5: Present Gaps with Previews, line 64 | docs-hydrate-specs has no branch for a gap with no suitable existing target spec file |
| a090 | nice-to-have | `src/kit/skills/docs-hydrate-specs.md` | ## Behavior → Step 5 prompt (line 70) vs Step 6: Interactive Confirmation (line 76) | Step 6 handles a 'skip rest' token that Step 5's prompt never offers |
| a091 | must-fix | `src/kit/skills/docs-reorg-memory.md` | ## Behavior § Step 5: User Confirmation & Apply, items 3-4 (lines 125-126); also Key Properties line 176 | Step 5.3 tells the agent to add description: frontmatter to a sub-domain index.md that does not exist until Step 5.4 generates it — and Step 5.4 forbids the edit |
| a092 | should-fix | `src/kit/skills/docs-reorg-memory.md` | § Ideal Shape Bounds (line 23), Step 1 (line 56), Step 3 Shape Report example (lines 78-84) | Depth off-by-one: Ideal Shape Bound counts the topic file as a path segment (≤3) while the Shape Report's Depth column counts folder depth (domain=1, sub=2) |
| a093 | nice-to-have | `src/kit/skills/docs-reorg-memory.md` | § Step 5 item 5 (line 127) and Error Handling last row (line 161) | Dangling-link hard block has no abort/rollback escape — apply can loop indefinitely when a rewrite target cannot be determined |
| a094 | should-fix | `src/kit/skills/docs-reorg-specs.md` | ## Purpose (line 12), ## Pre-flight (lines 18-19), § Step 1 (line 35) | docs-reorg-specs has no reserved-path exemption for the path-pinned SPEC mirrors (and is ambiguous about subfolder recursion) |
| a095 | should-fix | `src/kit/skills/fab-archive.md` | ## Behavior Step 1 (lines 51-67) + Key Properties idempotency row (line 110) | fab-archive has no path for the archive-succeeded-but-backlog-mark-failed exit; re-run can never mark the backlog |
| a096 | should-fix | `src/kit/skills/fab-archive.md` | Purpose (line 14); Step 1 (lines 51-67); Key Properties (lines 105-116 and 180-189) | fab-archive: archive/restore move tracked files and edit fab/backlog.md with no commit step, no git-state disclosure, and a "safe" claim the dirty tree contradicts |
| a097 | nice-to-have | `src/kit/skills/fab-archive.md` | ## Restore Mode › Step 2: Format Report table, lines 144-153 | Restore report maps pointer: skipped to '— not requested' even when --switch was requested and silently failed |
| a098 | nice-to-have | `src/kit/skills/fab-archive.md` | ### Step 2: Format Report (Restore Mode), line 153; Error Handling table lines 171-178 has no failed-switch row | fab change restore --switch swallows activation failure — exits 0 with `pointer: skipped`, which fab-archive renders as 'not requested' |
| a099 | must-fix | `src/kit/skills/fab-clarify.md` | Suggest Mode § Step 1.5 (line 55) vs § Step 2 (lines 61-73) | Step 1.5 zero-gaps early exit makes bulk confirm unreachable in its primary scenario (Confident-only intake) |
| a100 | nice-to-have | `src/kit/skills/fab-clarify.md` | Auto Mode item 4 (line 214) and § Step 8 (line 176) | "update last_updated" / Step 8 imply a manual .status.yaml edit that fab score already performs |
| a101 | nice-to-have | `src/kit/skills/fab-clarify.md` | Step 2 § Artifact Update (lines 110-118) and Step 4 item 2 | S→95 upgrade labels rows Certain whose composite score stays below _srad's 85 Certain threshold |
| a102 | nice-to-have | `src/kit/skills/fab-clarify.md` | Step 2 § Audit Trail (line 122) vs § Step 5 (line 153) | Step 5 audit trail lacks the placement rule Step 2 has, and same-day bulk-confirm re-runs duplicate session headings |
| a103 | nice-to-have | `src/kit/skills/fab-clarify.md` | § Skill Invocation Protocol, line 182 vs § Currently Applicable, lines 195-199 | fab-clarify Skill Invocation Protocol opens with an example its own section disclaims as removed |
| a104 | should-fix | `src/kit/skills/fab-continue.md` | ## Error Handling, line 204 | intake.md-missing error points to plain /fab-continue, which loops back to the same error |
| a105 | should-fix | `src/kit/skills/fab-continue.md` | ## Normal Flow > Step 1 dispatch table, line 58 | No dispatch row for review-pr in failed state — /fab-continue is undefined after a failed PR review |
| a106 | should-fix | `src/kit/skills/fab-continue.md` | ## Reset Flow (with stage argument), step 3, line 193 | Reset Flow errors when the target stage is already active — non-idempotent re-run |
| a107 | should-fix | `src/kit/skills/fab-continue.md` | ## Review Behavior (line 149) | fab-continue points to a 'Review Behavior' section that does not exist in _review.md |
| a108 | should-fix | `src/kit/skills/fab-continue.md` | Arguments :24, rework-menu Revise-requirements row :163, Reset Flow :191, Error Handling :208; also _pipeline.md:106 | Five caller sites still invoke `/fab-clarify intake`, which the current argument contract parses as a change-name override |
| a109 | should-fix | `src/kit/skills/fab-continue.md` | Step 1 dispatch table (lines 42-59) and Reset Flow (lines 189-196) | fab-continue has no dispatch path for review-pr:failed, and its Reset Flow on that state errors at the CLI |
| a110 | should-fix | `src/kit/skills/fab-continue.md` | § Normal Flow Step 1, dispatch guard (line 42) and table rows 55/58-59 | fab-continue has no dispatch row for review-pr failed — preflight derivation at that state matches the 'all done / Change is complete' row |
| a111 | should-fix | `src/kit/skills/fab-continue.md` | § Review Behavior → Verdict Fail options table + triage paragraph (lines 157-165) vs _pipeline.md § Auto-Rework Loop (lines 77, 87-92) | Rework triage policy and the three rework-path actions are divergent near-twins in fab-continue and _pipeline |
| a112 | nice-to-have | `src/kit/skills/fab-continue.md` | ## Hydrate Behavior > Steps, lines 184-185 | Hydrate's optional pattern capture is sequenced after the finish, making it unreachable on resume |
| a113 | nice-to-have | `src/kit/skills/fab-continue.md` | ## Normal Flow > Step 1 dispatch table, lines 57-58 | Ship and review-pr dispatch rows cite a `ready` state the Go state machine disallows for those stages |
| a114 | nice-to-have | `src/kit/skills/fab-continue.md` | ### Step 4: Update `.status.yaml` (line 85) | fab-continue understates the reset transition's From-set (omits `skipped`) |
| a115 | nice-to-have | `src/kit/skills/fab-continue.md` | Step 4: Update .status.yaml (line 85) | fab-continue Step 4 documents reset as "done/ready → active", dropping the `skipped` source state the Go machine and _cli-fab both allow |
| a116 | nice-to-have | `src/kit/skills/fab-continue.md` | § Step 4: Update .status.yaml, event command list (line 85) | fab-continue's reset event description omits the 'skipped' from-state |
| a117 | structural | `src/kit/skills/fab-continue.md` | frontmatter line 4 + Step 3, line 67 | _srad loads unconditionally though it is needed only at the same moments as the stage-conditional _generation |
| a118 | nice-to-have | `src/kit/skills/fab-discuss.md` | Context Loading, line 33 | fab-discuss tells the agent to read .status.yaml 'for the current stage', but .status.yaml has no stage field — derivation from the progress map is left implicit |
| a119 | nice-to-have | `src/kit/skills/fab-draft.md` | ## Behavior delta 3 (line 30) and file trailer (line 48) | Hard-coded Next: command lists omit /fab-proceed from state-table-derived rows |
| a120 | should-fix | `src/kit/skills/fab-ff.md` | § Behavior parameter table, `{driver}` row (line 32); identical row in src/kit/skills/fab-fff.md line 32 | Driver parameter row claims `{driver}` is 'passed to every fab status event command', contradicting _pipeline's deliberate driver-less fail and recovery-start commands |
| a121 | nice-to-have | `src/kit/skills/fab-ff.md` | § Arguments `--force` bullet (line 22) vs § Output template header (line 42); identical pair in src/kit/skills/fab-fff.md lines 22/68 | Force-mode output header underspecified: composing the Arguments instruction with the Output template yields the contradictory 'gate passed. (force mode -- gate bypassed)' |
| a122 | nice-to-have | `src/kit/skills/fab-ff.md` | § Output (lines 41-58) vs src/kit/skills/fab-fff.md § Output (lines 65-93) | Residual twin drift in the per-driver Output blocks: header wording and apply-output annotation diverge, resuming sentence duplicated verbatim |
| a123 | nice-to-have | `src/kit/skills/fab-ff.md` | § Purpose (line 15), § Arguments (lines 21-22), § Output closing lines (line 58) — byte-identical in fab-fff.md lines 15, 21-22, 93; claim at _pipeline.md lines 22-23 | Residual fab-ff/fab-fff verbatim twin prose contradicts _pipeline's 'single authoritative source' claim |
| a124 | nice-to-have | `src/kit/skills/fab-ff.md` | § Purpose (line 15); same phrase in src/kit/skills/fab-fff.md line 15; SPEC-fab-ff.md Summary says 'The bracket owns the single intake confidence gate' while its bookkeeping table says 'Before the bracket (intake gate)' | Purpose says the intake gate is 'checked before the bracket' while Behavior and the SPEC say the bracket owns it |
| a125 | must-fix | `src/kit/skills/fab-fff.md` | § Step 4: Ship and § Step 5: Review-PR (lines 39-61), interacting with § Arguments (line 21) | fab-fff Steps 4-5 pass `change: {id}` to /git-pr and /git-pr-review, but both skills self-resolve only the ACTIVE change — the <change-name> override ships/mutates the wrong change |
| a126 | nice-to-have | `src/kit/skills/fab-help.md` | Purpose, line 12 | fab-help Purpose understates the help output: it lists git-*, docs-*, fab sync, batch commands, and packages, not only /fab-* commands |
| a127 | should-fix | `src/kit/skills/fab-new.md` | ## Output (lines 182-189) vs _srad.md § Assumptions Summary Block (line 92) | Output ordering violates _srad's SHALL: Assumptions summary must be the final block immediately before Next:, but Confidence/Activated/Branch lines intervene |
| a128 | should-fix | `src/kit/skills/fab-new.md` | Step 11 branch table rows 2/3/5 (lines 154-160) and Error Handling (line 205); mirror at git-branch.md:140 | fab-new Step 11 / git-branch: dirty working tree silently rides into the new change's branch — the documented caveat covers only committed work |
| a129 | should-fix | `src/kit/skills/fab-new.md` | Step 3: Create Change, backlog bullet (line 50) | Backlog-ID collision pre-check is substring-based, not ID-anchored — single false-positive match wrongly routes to resume and silently skips creation |
| a130 | must-fix | `src/kit/skills/fab-operator.md` | §6 Autopilot, per-change loop steps 1-4 (lines 489-492) vs Spawning an Agent step 5 (line 391) | Autopilot per-change loop double-dispatches: spawn-sequence step 5 embeds '<command>' in the tab-open, but Gate and Dispatch come after |
| a131 | should-fix | `src/kit/skills/fab-operator.md` | §1 Principles › Spawn-in-worktree, line 23 | fab-operator Spawn-in-worktree principle points to §5 for the spawn sequence, which lives in §6 |
| a132 | should-fix | `src/kit/skills/fab-operator.md` | §6 Autopilot repo-spanning paragraph (line 475), Queue ordering table (line 481), Queue Completion Summary example (line 511) | Implicit queue chaining contradicts its own worked example: depends_on:[<prev-change-id>] vs 'ef56 cherry-picks from its same-repo predecessor' / 'depends on ab12' |
| a133 | should-fix | `src/kit/skills/fab-operator.md` | §6 Dependency Resolution (lines 429-445); Pipeline Reference (line 379); Autopilot (lines 485, 499) | fab-operator: cherry-pick/rebase commands hardcode `origin/main` — guaranteed git failure on any coordinated repo whose default branch is not literally `main`, with no fetch step to keep the ref fresh |
| a134 | should-fix | `src/kit/skills/fab-operator.md` | §6 Working a Change, entry-form table, 'Existing change' row (line 463) | Initial command '/fab-switch <change> && /fab-proceed' relies on undefined slash-command chaining semantics |
| a135 | nice-to-have | `src/kit/skills/fab-operator.md` | §1 Principles, 'Spawn-in-worktree' paragraph (line 23) | §1 cross-reference points spawn flow at §5 (Auto-Nudge) instead of §6 |
| a136 | nice-to-have | `src/kit/skills/fab-operator.md` | §4 Branch Map (line 192) | branch_map retention 'until the operator session ends' is stale pre-server-keyed semantics with no clearing mechanism |
| a137 | nice-to-have | `src/kit/skills/fab-operator.md` | §4 Status Frame Format, example frame (lines 226-249) vs header rule (line 252) and §7 Schema `source` row | Status-frame example is internally inconsistent: header says '7 tracked' for 8 entries, and the 'gmail-deploys' watch has no valid source |
| a138 | structural | `src/kit/skills/fab-operator.md` | §4 Status Frame Format (lines 216–287) | Operator status frame is a fully mechanical render (emoji tables, ordering, thresholds) specified in ~3.5KB of prose — a `fab operator frame` subcommand would make it byte-stable and shrink the skill |
| a139 | structural | `src/kit/skills/fab-operator.md` | §6 Coordination Patterns (lines 365–539, ~17.0KB) and §7 Watches (lines 541–591, ~3.9KB); reload mandate at §1 line 33 | fab-operator should load autopilot and watches conditionally — 42% of its 49.8KB body is re-paid on every /clear even when no queue or watch exists |
| a140 | should-fix | `src/kit/skills/fab-proceed.md` | § Dispatch Behavior > Conversation Context Synthesis (line 163) and § Error Handling (lines 175-183) | fab-new subagent dispatch has no defined behavior when SRAD requires asking the user |
| a141 | should-fix | `src/kit/skills/fab-proceed.md` | § State Detection > Dispatch Table, rows 3 and 5 (lines 89, 91); also mirrored in SPEC-fab-proceed.md lines 79, 81 | Dispatch table chains /git-branch after /fab-new, but fab-new has created the branch inline since PR #322 |
| a142 | nice-to-have | `src/kit/skills/fab-proceed.md` | Header blockquote (line 10) vs _preamble.md §1 Always Load (line 36) | fab-proceed's context-loading opt-out has no sanctioned home — _preamble §1's exception list omits it and the skill lacks a Context Loading section |
| a143 | nice-to-have | `src/kit/skills/fab-proceed.md` | Step 1: Active Change Check (lines 32-38) and Error Handling (line 177) | fab-proceed conflates 'project not initialized' with 'no active change' — never routes to the State Table's (none) → /fab-setup |
| a144 | nice-to-have | `src/kit/skills/fab-proceed.md` | § Relevance Assessment step 4 (line 108) and Dispatch Table note (line 95); scan command at line 74 | Date-recency tiebreak is undefined and non-deterministic for same-day drafts |
| a145 | nice-to-have | `src/kit/skills/fab-proceed.md` | § Relevance Assessment step 5 (line 109) | Cross-reference '(see Output Format)' points to a heading that exists only in the SPEC |
| a146 | should-fix | `src/kit/skills/fab-setup.md` | config menu, line 167 (plus scaffold comment lines 29-30) | stage_directives is a dead config key — edited and migrated everywhere, consumed nowhere |
| a147 | should-fix | `src/kit/skills/fab-setup.md` | § Migrations Step 1: Discover Migrations, item 3 (line 303) vs line 294 | Migrations Step 1.3 now requires the agent to semver-compare local vs engine, but #393 deleted the Semver Comparison rule it needs |
| a148 | nice-to-have | `src/kit/skills/fab-setup.md` | § 1c Sync-failure guard (line 95) and § Config Create Mode step 5 (line 153) | Sync-failure guard claims step 1a 'guarantees' fab_version, but Config Create Mode has no fallback when there is no existing key to preserve |
| a149 | nice-to-have | `src/kit/skills/fab-setup.md` | § 1c. fab sync — scaffold, directories, deployment, gitignore, lines 85-97 | Step 1c's 'owns all non-interactive structural setup' enumeration omits sync's settings.local.json permissions merge, hook registration, .envrc/direnv, and project sync scripts |
| a150 | nice-to-have | `src/kit/skills/fab-setup.md` | § Next Steps Reference, lines 430-436 | Next Steps Reference omits /fab-proceed from all four 'initialized' lines it claims to derive from the State Table |
| a151 | nice-to-have | `src/kit/skills/fab-status.md` | ## Behavior, status block bullet (line 46) | fab-status progress-table symbol legend has no glyph for the `skipped` state |
| a152 | nice-to-have | `src/kit/skills/fab-status.md` | ## Key Properties, line 78 | fab-status Key Properties claims config/constitution are not required, but its mandatory preflight hard-fails without them |
| a153 | structural | `src/kit/skills/fab-status.md` | ## Behavior, lines 33–66 | fab-status is ~90% deterministic formatting prose that belongs in a Go subcommand, following the fab pr-meta / fab fab-help / fab memory-index precedent |
| a154 | structural | `src/kit/skills/fab-status.md` | ## Behavior, lines 40-66 | fab-status rendering rules (impact thresholds, refactor warning, drift check) are mechanical logic that belongs in the Go CLI |
| a155 | must-fix | `src/kit/skills/fab-switch.md` | ## Output, lines 84-99 (root cause: src/go/fab/internal/change/change.go:217-223, defaultCommand:427-440) | fab change switch Next: guidance is off-by-one at post-review stages and contradicts both the skill's gloss and fab-status |
| a156 | should-fix | `src/kit/skills/fab-switch.md` | ## Output (line 95) | fab-switch documents display_state qualifiers as done/active/pending — omitting `ready`, the standard state of every freshly switched draft, and `skipped` |
| a157 | nice-to-have | `src/kit/skills/fab-switch.md` | ## Behavior › No Argument Flow, lines 29-33 | fab-switch no-argument flow never says to run the switch after the user picks from the list |
| a158 | should-fix | `src/kit/skills/git-branch.md` | Step 2: Resolve Change Name, lines 53-62 | git-branch: ambiguous multi-match resolution silently creates a junk standalone branch with a false 'No matching change found' message |
| a159 | should-fix | `src/kit/skills/git-branch.md` | Step 4: Context-Dependent Action, lines 85-90 | git-branch: branch-existence check ignores remote-only branches, so a fresh clone/worktree recreates a divergent branch instead of tracking origin |
| a160 | nice-to-have | `src/kit/skills/git-branch.md` | Step 4, rename guard bullets, lines 122-140 | git-branch rename guard enumerates only 'resolution fails' and 'matches a different change' — same-change match and detached HEAD are undefined states |
| a161 | must-fix | `src/kit/skills/git-pr-review.md` | Step 5 Commit and Push (lines 139-149); Step 6 intro (line 180); Rules (line 229) | git-pr-review: "(no partial state)" claim is false on non-fast-forward push rejection — git reset cannot undo the commit, and the idempotent re-run permanently strands the fixes |
| a162 | should-fix | `src/kit/skills/git-pr-review.md` | ## Rules (line 227) vs Step 6 (lines 180, 185) | Rules 'Fail fast … stop immediately' contradicts the batch-1 Step-6 routing design, and Step 6's 'processing error' outcome is orphaned — no step routes processing errors there |
| a163 | should-fix | `src/kit/skills/git-pr-review.md` | Header (line 11) vs Step 2 Phase 1 (line 64) and Phase 2 forced-tool note (line 74) | --tool flag header claims it 'bypasses automatic detection' and 'only that tool is attempted', but Phase 1 detection still runs and silently overrides the flag when comments exist; 'the cascade' is an undefined leftover term |
| a164 | should-fix | `src/kit/skills/git-pr-review.md` | Step 5: Commit and Push (lines 139-149) and Step 5.5 (line 165) | Step 5 push-failure handling cannot deliver its promised 'no partial state' — git reset does not undo a successful commit, and the re-run path posts 'Fixed' replies citing an unpushed SHA |
| a165 | nice-to-have | `src/kit/skills/git-pr-review.md` | ### Phase Sub-State Tracking, table rows (lines 212-220) | Phase tracking never fires on the Phase-2 Copilot path: 'received' and 'reviewer' are defined only for a Phase-1 hit |
| a166 | nice-to-have | `src/kit/skills/git-pr-review.md` | Step 3: Fetch Comments (line 105) | Step 3 fetches node_id that nothing consumes — residue of an abandoned GraphQL thread-resolution design |
| a167 | must-fix | `src/kit/skills/git-pr.md` | Step 2 Branch Guard (line 106) + Step 3b Push (lines 156-163); Step 1 (line 73) | git-pr: detached HEAD passes the branch guard, then autonomously commits and emits a refspec-less push |
| a168 | should-fix | `src/kit/skills/git-pr.md` | Step 1: Gather State (lines 70–87) + Step 3 'If nothing to do' (line 127) + Step 3c gate (line 165) | has_pr ignores PR state — a closed/merged PR on the branch short-circuits PR creation; `state` and `number` are fetched but never consulted |
| a169 | should-fix | `src/kit/skills/git-pr.md` | Step 3a Commit (lines 143-150); Rules (line 273) | git-pr: autonomous `git add -A` sweeps every untracked/unrelated file repo-wide into a pushed commit with no inspection step |
| a170 | nice-to-have | `src/kit/skills/git-pr.md` | Step 0a: Start Ship Stage (line 35) | Step 0a claims a start on an already-active ship stage 'is a no-op' — the CLI actually rejects it with a non-zero error, and already-active is the canonical path |
| a171 | nice-to-have | `src/kit/skills/git-pr.md` | Step 2 Branch Guard (lines 104-121) | git-pr: branch guard checks the literal names main/master, not the repo's actual default branch — on a develop/trunk-default repo the autonomous commit and push land directly on the default branch before PR creation fails |
| a172 | nice-to-have | `src/kit/skills/git-pr.md` | Step 3c, sub-step 4 (lines 220–222) | Step 3c.4 failure branches contradict: 'PR creation fails → STOP' is listed before a silent --fill fallback whose trigger ('body generation fails') belongs to the previous sub-step |
| a173 | nice-to-have | `src/kit/skills/internal-retrospect.md` | Top of file, lines 1-6 (frontmatter flows straight into body prose) | internal-retrospect is the only skill file with no H1 heading |
| a174 | should-fix | `src/kit/skills/internal-skill-optimize.md` | Arguments (line 15), Pre-flight step 1 (line 21), Constraints (line 86); same omission in docs/specs/skills/SPEC-internal-skill-optimize.md Summary (line 7) and Flow (line 15) | internal-skill-optimize partial enumerations omit _pipeline — regression from the #393 twins refactor |
| a175 | should-fix | `src/kit/skills/internal-skill-optimize.md` | Arguments (line 15), Procedure step 1 (line 21), Constraints (line 86) | internal-skill-optimize's three partial enumerations omit _pipeline (created by #393) |

## Appendix B — Finding details

### `docs/specs/architecture.md`

#### `a001` [STRUCTURAL] architecture.md is built on the pre-binary fab/.kit distribution model the constitution explicitly replaced

**Location**: § Directory Structure (lines 9-71), § Agent Integration (374-402), § Distribution & Bootstrapping (406-453), § Updating .kit/ (457-476); contradicted by the same file's § Router Dispatch (line 482) · **Category**: staleness · **Found by**: non-skill-spec-docs-drift-sweep

Rewrite the directory-structure/bootstrap/update sections around the current model: brew-installed `fab` router + version cache, `fab sync` producing gitignored deployed copies in .claude/skills/ (not symlinks into src/kit/, per constitution.md:33), user projects containing only fab/ + .claude/skills/. The four .kit-era sections (cp -r bootstrap, 'When .kit/ gets its own repository', symlink agent integration, 'What to commit … src/kit/') all describe an abandoned architecture, while the appended Router Dispatch section describes the real one — the document contradicts itself. (Distinct from the deferred uliv pre-Go-CLI script-name sweep, which only covers the statusman/changeman/logman name residue.) Impact: the always-loaded specs index routes agents to a spec whose first 470 lines describe a distribution model that no longer exists.

**Verifier**: All cited evidence independently verified. docs/specs/architecture.md:428 ("no package manager, no CLI binary, no system install") directly contradicts both constitution.md:18 (kit in ~/.fab-kit cache, brew-installed fab binary) and the same file's line 482 (brew-installed thin router). The Agent In…

### `docs/specs/assembly-line.md`

#### `a002` [MUST-FIX] assembly-line.md still narrates the removed spec/tasks stages as part of the pipeline

**Location**: § How It Works step 2 (line 121); also intro line 5 · **Category**: staleness · **Found by**: non-skill-spec-docs-drift-sweep

Change line 121 to the six-stage list (apply → review → hydrate → ship → review-pr after intake) and line 5 to 'its own intake, plan, and status' — which would also make line 5 consistent with the file's own line 140 ('each change has its own intake, plan, and status'). Impact: the marketing-facing spec teaches a 7+-stage pipeline with spec/tasks artifacts that 1.10.0 removed, contradicting a constitutional MUST constraint.

**Verifier**: Confirmed verbatim: docs/specs/assembly-line.md:121 narrates "(spec, tasks, apply, review, hydrate, then ship and review-pr)" and line 5 says "its own spec, tasks, and status" — both contradict constitution.md:34 (six stages, no spec stage or spec.md artifact, amended 2026-06-01) AND the file's own…

### `docs/specs/glossary.md`

#### `a003` [SHOULD-FIX] glossary.md describes /fab-ff as running 'auto-clarify between planning stages' — a mechanism fab-ff explicitly disclaims

**Location**: § Skills, /fab-ff row (line 49); repeated in § Workflow Concepts 'Fast-forward' (line 115) · **Category**: staleness · **Found by**: non-skill-spec-docs-drift-sweep

Delete the auto-clarify clause from both the /fab-ff row and the 'Fast-forward' workflow-concept entry (line 115 'Proceeds autonomously with auto-clarify and rework loop'); 'between planning stages' is itself a fossil of the removed spec stage, since intake is now the only planning stage. Impact: the glossary — the canonical terminology reference linked from the specs index — defines a fast-forward behavior that no longer exists, implying mid-bracket interaction that 1.10.0's single-intake-gate model deliberately removed.

**Verifier**: Confirmed at stated severity. glossary.md:49 ("Auto-clarify between planning stages") and :115 ("Proceeds autonomously with auto-clarify and rework loop") quoted exactly. Ground truth disclaims it in three places: src/kit/skills/fab-ff.md:15 and _pipeline.md:59 ("No /fab-clarify runs inside the brac…

### `docs/specs/operator.md`

#### `a004` [SHOULD-FIX] operator.md prose says 'current operator (v8) … eight iterations' but its own table ends at v9 (seed finding, confirmed)

**Location**: § Version History, line 11 vs table row line 23 · **Category**: correctness · **Found by**: non-skill-spec-docs-drift-sweep

Update the prose to 'The current operator (v9) evolved through nine iterations'. While editing, note that the spec's only state-file mention is the v6 row's `.fab-operator.yaml`, which fab-operator.md:65/121 now calls abandoned ('Old repo-rooted `.fab-operator.yaml` files are not read or migrated') — add a line pointing to the current server-keyed operator state file at $XDG_STATE_HOME/fab/operator/<server-slug>.yaml. Impact: readers conclude v8 is current and that repo-rooted .fab-operator.yaml is the live state mechanism.

**Verifier**: Confirmed verbatim: /home/sahil/code/sahil87/fab-kit.worktrees/woven-vole/docs/specs/operator.md:11 reads 'The current operator (v8) evolved through eight iterations:' while the same file's Version History table ends with a v9 row at line 23 ('v9 | Spawn-in-worktree principle ...'). The v9 principle…

### `docs/specs/overview.md`

#### `a005` [STRUCTURAL] overview.md tells a 4-stage story the constitution, fab-help, and glossary contradict, and its Quick Reference omits /git-pr, /git-pr-review, /fab-proceed, and /fab-operator

**Location**: heading at line 37; Quick Reference table lines 81–103 · **Category**: onboarding · **Found by**: lens-architecture

Constitution line 34 states 'The core pipeline is six stages' and batch-1's fabhelp.go fix renders a 6-stage diagram with git-pr/git-pr-review in the Completion group — but the new-user entry doc still frames ship/review-pr as add-ons and its Quick Reference lists `fab batch` while omitting the two PR-stage skills and both orchestration entry points (/fab-proceed, /fab-operator). Retitle to 'The Six Stages', add the four missing rows (one line each, linking /fab-operator to operator.md), and align the stage-table 'Includes' column. Tradeoff: none material — this is the doc most likely read first, and it currently produces a different command inventory than /fab-help.

**Verifier**: Verified verbatim: overview.md:37 heading '## The 4 Core Stages (6 with Ship + Review-PR)' vs constitution.md:34 'The core pipeline is six stages', glossary.md:23/36-37 (Ship=Stage 5, Review-PR=Stage 6), and fabhelp.go:37-38/168-176 (six-stage pipeline, git-pr/git-pr-review in Completion). Even docs…

### `docs/specs/skills.md`

#### `a006` [STRUCTURAL] skills.md /fab-archive section documents the pre-date-bucketing archive path the Go CLI no longer uses

**Location**: ## `/fab-archive [<change-name>]`, Behavior step 1, line 583 · **Category**: spec-drift · **Found by**: lens-architecture

The skill (fab-archive.md:62) and the binary (internal/archive/archive.go: `destDir := filepath.Join(archiveDir, bucketYear, bucketMonth)`) both move to `fab/changes/archive/yyyy/mm/{name}/`. Update the skills.md step to the date-bucketed path and add a parenthetical to glossary.md's `fab/changes/archive/` row when the deferred uliv docs sweep touches that file (this drift is not in that sweep's enumerated scope, so it will otherwise survive it). Tradeoff: none — a user scripting against the documented flat path would silently find nothing.

**Verifier**: Verified: docs/specs/skills.md:583 documents the flat archive path (fab/changes/archive/{name}/) while both the canonical skill (src/kit/skills/fab-archive.md:62) and the Go binary (src/go/fab/internal/archive/archive.go:85, destDir := filepath.Join(archiveDir, bucketYear, bucketMonth), introduced i…

### `docs/specs/skills/SPEC-_pipeline.md`

#### `a007` [SHOULD-FIX] SPEC's PR-meta rationale is false: the fail+reset choreography wipes stage_metrics.review.iterations every cycle, so PR meta always reports 1 review cycle

**Location**: ## Per-Cycle Rework Choreography (f071), item 1 (line 20) · **Category**: spec-drift · **Found by**: pipeline-srad-helpers

Ground truth: status.go Reset cascades downstream stages to pending and applyMetricsSideEffect's `case "pending", "skipped": delete(statusFile.StageMetrics, stage)` deletes the review metric on every per-cycle `reset apply`; the subsequent `finish apply` auto-activate recreates it with Iterations=1. So prmeta.go's reviewCell ('✓ {N} cycle{s}') renders '1 cycle' no matter how many rework cycles ran — the choreography destroys the very counter the SPEC cites as its payoff (the f071 verifier missed the cascade delete). Fix in Go: preserve Iterations when cascading a failed/previously-active stage to pending (or derive the PR-meta cycle count from .history.jsonl review 'failed' events, which do accumulate via log.Review). At minimum delete the false parenthetical so the SPEC stops claiming telemetry the choreography zeroes.

**Verifier**: Independently confirmed end-to-end. Evidence quote is verbatim at docs/specs/skills/SPEC-_pipeline.md:20. Go ground truth verified: src/go/fab/internal/status/status.go Reset (lines 190-218) cascades downstream stages to pending and applyMetricsSideEffect (lines 542-574) deletes StageMetrics on pend…

### `docs/specs/skills/SPEC-_preamble.md`

#### `a008` [SHOULD-FIX] SPEC mirror cites a stale opening instruction that no skill uses

**Location**: ## Summary, closing paragraph (line 7) · **Category**: spec-drift · **Found by**: preamble

All 16 skills carrying the read line (and the preamble's own canonical blurb at _preamble.md:11-12) use: "Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding." The SPEC's `src/kit/skills/` path is also wrong for deployed user projects, where kit sources don't exist (constitution Principle V). Replace the quoted instruction with the actual deployed-skill wording.

**Verifier**: Confirmed. SPEC-_preamble.md:7 quotes an opening instruction ("Read `src/kit/skills/_preamble.md` first.") that no skill uses — repo-wide grep shows that string exists only in the SPEC itself. The actual wording in 15 skill files plus _preamble.md:11-12's canonical blurb is: "Read the `_preamble` sk…

#### `a009` [SHOULD-FIX] SPEC Tools-used table still lists kit.conf build guard, eliminated in 260402-gnx5

**Location**: ### Tools used table (line 105) · **Category**: spec-drift · **Found by**: preamble

`kit.conf` and the test-build guard were removed from the preamble in 260402-gnx5 (confirmed by docs/memory/_shared/context-loading.md:200: "Test-build guard removed from preamble (`kit.conf` eliminated)"); the current _preamble.md contains no kit.conf reference and `grep -rn kit.conf src/` returns nothing. Change the Read row to "all context layer files" only. This stale row survived both the batch-2 SPEC refresh (f048) and the batch-3 SPEC update for the preamble diet.

**Verifier**: CONFIRMED. (1) Evidence quote exists verbatim at docs/specs/skills/SPEC-_preamble.md:105: "| Read | kit.conf (build guard), all context layer files |". (2) Ground truth: grep -rn kit.conf src/ returns nothing; src/kit/skills/_preamble.md has zero kit.conf or build-guard references; the only kit.conf…

#### `a010` [SHOULD-FIX] SPEC-_preamble Tools table cites a dead "kit.conf (build guard)" read

**Location**: ### Tools used, line 105 · **Category**: staleness · **Found by**: lens-reference-integrity

Delete "kit.conf (build guard)" — the test-build guard and kit.conf were eliminated in 260402-gnx5 (docs/memory/_shared/context-loading.md changelog: "Test-build guard removed from preamble (`kit.conf` eliminated)"), and no kit.conf reference exists anywhere in src/kit/ today. This SPEC was rewritten in all of #391/#392/#393 and the dead row survived each pass.

**Verifier**: Independently confirmed. SPEC-_preamble.md line 105 Tools table reads "| Read | kit.conf (build guard), all context layer files |", but the canonical src/kit/skills/_preamble.md has no kit.conf or build-guard reference, and grep finds zero kit.conf hits anywhere in src/kit/ or src/go/. The cited cha…

#### `a011` [NICE-TO-HAVE] SPEC-_preamble misquotes the canonical per-skill opening instruction

**Location**: ## Summary, line 7 · **Category**: staleness · **Found by**: lens-reference-integrity

Update the quote to the actual canonical blockquote (_preamble.md:11-12 and docs/specs/skills.md New Skill Checklist item 2): "Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding." The quoted src/kit path is pre-cache-relocation wording and doesn't exist in user projects.

**Verifier**: Confirmed. docs/specs/skills/SPEC-_preamble.md:7 quotes an opening instruction ("Read `src/kit/skills/_preamble.md` first.") that no skill uses. All 16 skills with a preamble-read line (src/kit/skills/*.md) use the canonical blockquote from _preamble.md:11-12, which docs/specs/skills.md:121 (New Ski…

### `docs/specs/skills/SPEC-_review.md`

#### `a012` [NICE-TO-HAVE] SPEC-_review retains pre-1.10.0 'spec/plan' phrasing for what the inward sub-agent validates against

**Location**: Summary (line 5) and Sub-agents section (line 110) · **Category**: staleness · **Found by**: gen-review-helpers

There is no spec stage or spec.md artifact (constitution: "there is no separate `spec` stage or `spec.md` artifact"), and _review.md itself says the inward sub-agent validates against "the plan's `## Requirements`, `## Tasks`, and `## Acceptance`". Replace both "spec/plan" occurrences with "the plan's `## Requirements`/`## Tasks`/`## Acceptance`" (or just "plan.md").

**Verifier**: Confirmed: docs/specs/skills/SPEC-_review.md lines 5 and 110 both say the inward sub-agent "validates implementation against spec/plan". The canonical skill src/kit/skills/_review.md:31 says it validates against "the plan's ## Requirements, ## Tasks, and ## Acceptance", and fab/project/constitution.…

### `docs/specs/skills/SPEC-docs-hydrate-memory.md`

#### `a013` [SHOULD-FIX] Three-way contradiction on docs-hydrate-memory's always-load exemption: SPEC says partial skip, preamble says entire skip, skill file is silent

**Location**: Flow diagram, line 12 (vs _preamble.md §1 line 36 and the skill file, which has no Context Loading section) · **Category**: spec-drift · **Found by**: docs-hydrate

The SPEC says the skill does a partial always-load (skips only config/constitution), but _preamble.md line 36 says '/docs-hydrate-memory skip[s] the layer entirely', and the skill file itself has no Context Loading section at all — yet batch 3's f001 fix made the skill file the authority ('the skill file wins'). 'Entirely' is also behaviorally wrong: the skill's Pre-flight requires 'docs/memory/index.md must exist and be readable' (a layer file). Add a Context Loading section to src/kit/skills/docs-hydrate-memory.md (modeled on fab-status.md:28) stating exactly what it loads (docs/memory/index.md via pre-flight; not config/constitution/specs-index), then align the preamble exception wording and the SPEC flow line to it.

**Verifier**: Confirmed three-way inconsistency. (1) docs/specs/skills/SPEC-docs-hydrate-memory.md:12 says "always-load layer — partial: skips config/constitution"; (2) src/kit/skills/_preamble.md:36 says /docs-hydrate-memory "skip[s] the layer entirely" and makes the skill's own Context Loading section the autho…

### `docs/specs/skills/SPEC-docs-hydrate-specs.md`

#### `a014` [SHOULD-FIX] SPEC-docs-hydrate-specs flow drifted from the skill: phantom 'modify' option, phantom spec-index edit and new-file creation

**Location**: Flow diagram lines 22 and 25; Tools table line 33 · **Category**: spec-drift · **Found by**: docs-hydrate

The skill's confirmation options are '(yes / no / done)' — there is no modify path — and the skill body never creates new spec files nor touches docs/specs/index.md (Step 6 only inserts into existing spec files; Error Handling and Steps 1-7 have no index step). The Tools table's 'Edit | Spec files, spec index' repeats the phantom. Per the constitution constraint ('Changes to skill files MUST update the corresponding SPEC-*.md'), rewrite the SPEC flow to match the skill: options yes/no/done, insertion into existing spec files only — or, if new-file creation is actually intended, add it to the skill first (see the no-target-branch finding).

**Verifier**: All three cited discrepancies verified verbatim in docs/specs/skills/SPEC-docs-hydrate-specs.md (lines 22, 25, 33). The skill (src/kit/skills/docs-hydrate-specs.md) uses '(yes / no / done)' (line 70, Steps 5-6), inserts only into existing spec files, and never edits docs/specs/index.md or creates ne…

### `docs/specs/skills/SPEC-docs-reorg-memory.md`

#### `a015` [NICE-TO-HAVE] SPEC mirror uses Kind tokens `split`/`merge` where the skill's Migration Map enum is `split-domain`/`merge-domain`

**Location**: ## Summary (line 5) and § Link Impact (line 19) · **Category**: spec-drift · **Found by**: docs-reorg

The skill defines the Kind column enum as `move-section` / `split-domain` / `merge-domain` / `flatten` / `move` (docs-reorg-memory.md:100); the SPEC's shorthand `split` / `merge` (also in the Summary: 'performs the approved moves (split / merge / flatten / move)') could lead a SPEC-guided edit to emit wrong Migration Map tokens. Use the skill's exact tokens in both SPEC spots.

**Verifier**: Confirmed. SPEC-docs-reorg-memory.md:19 uses backticked `split` / `merge` in the normative Link Impact MUST sentence, while the skill's canonical Kind enum (src/kit/skills/docs-reorg-memory.md:100, reused verbatim at :102 and :123) is `split-domain` / `merge-domain` / `flatten` / `move`. The line-19…

### `docs/specs/skills/SPEC-fab-archive.md`

#### `a016` [SHOULD-FIX] SPEC-fab-archive flow diagram applies preflight + hydrate guard to both modes, contradicting the skill and the SPEC's own Summary

**Location**: ## Flow diagram, lines 11-31 · **Category**: spec-drift · **Found by**: change-mgmt

The preflight call and hydrate guard sit above the Archive/Restore branch split, implying they run for restore too — but the skill ("No standard preflight runs ... the hydrate guard is waived") and the SPEC's own Summary ("it **waives** the standard preflight and the hydrate guard") say restore skips both. Move the two lines inside the Archive Mode branch. Likely introduced by #393's f087 single-document restructure when the diagram was redrawn.

**Verifier**: Confirmed. docs/specs/skills/SPEC-fab-archive.md lines 14-18: 'fab preflight [change-name]' and 'Guard: progress.hydrate must be done' sit above the Archive/Restore branch split, implying both run for restore. Ground truth contradicts: src/kit/skills/fab-archive.md line 125 ('No standard preflight r…

### `docs/specs/skills/SPEC-fab-clarify.md`

#### `a017` [SHOULD-FIX] SPEC-fab-clarify Flow diagram retains removed [target-artifact] argument, {artifact}.md placeholders, and a fab score call missing --stage

**Location**: Flow diagram lines 12, 24, 30-49, 43 vs Bookkeeping table line 71 · **Category**: spec-drift · **Found by**: fab-clarify

The skill removed artifact targets ("Any positional argument is treated as a change name", fab-clarify.md:28), yet the Flow header still shows [target-artifact] and Steps 1/2/3-4/5 and Auto Mode still edit a generic `fab/changes/{name}/{artifact}.md`. Line 43 shows "Bash: fab score <change>" without `--stage intake`, contradicting the skill's Step 7 and the SPEC's own Bookkeeping row (`fab score --stage intake <change>`). Line 30 states the bulk-confirm trigger as only "if confident >= 3", dropping the second condition `confident > tentative + unresolved`. Rewrite the Flow to `[<change-name>]` + literal `intake.md`, add `--stage intake` at :43, and show both trigger conditions at :30 (constitution: skill changes MUST keep SPEC mirrors current).

**Verifier**: All four sub-claims confirmed against /home/sahil/code/sahil87/fab-kit.worktrees/woven-vole/docs/specs/skills/SPEC-fab-clarify.md and src/kit/skills/fab-clarify.md. (1) SPEC:12 reads "User invokes /fab-clarify [change-name] [target-artifact]" verbatim, while the skill (fab-clarify.md:7 "# /fab-clari…

### `docs/specs/skills/SPEC-fab-continue.md`

#### `a018` [SHOULD-FIX] SPEC mirror Tools table is stale: writes a removed 'Spec' artifact and claims fab score usage the skill forbids

**Location**: ### Tools used, lines 137-139 · **Category**: spec-drift · **Found by**: fab-continue

Two stale rows: (1) 'Write | Spec, plan, memory files' — spec.md was removed in 1.10.0; the skill writes intake.md (intake-active regeneration), plan.md, and memory files. (2) 'Bash | All `fab status` transitions, `fab score`, `fab preflight`, test execution' — the skill states 'No scoring at any stage `/fab-continue` runs' (fab-continue.md:75) and the SPEC's own flow note says '(no scoring here — intake score is written by /fab-new and /fab-clarify)'; drop `fab score`. While editing, also reconcile the Bookkeeping table's 'Review pass | fab status set-acceptance' row with the skill's Verdict Fail path, which also runs set-acceptance (fab-continue.md:157).

**Verifier**: CONFIRMED. Both stale rows verified verbatim in /home/sahil/code/sahil87/fab-kit.worktrees/woven-vole/docs/specs/skills/SPEC-fab-continue.md: line 137 `| Write | Spec, plan, memory files |` and line 139 `| Bash | All \`fab status\` transitions, \`fab score\`, \`fab preflight\`, test execution |`. (1…

### `docs/specs/skills/SPEC-fab-discuss.md`

#### `a019` [NICE-TO-HAVE] SPEC-fab-discuss flow and tools table omit the conditional .status.yaml read for the active change's stage

**Location**: Flow (lines 9-19) and Tools used table (lines 21-26) · **Category**: spec-drift · **Found by**: entry-points

The skill's Context Loading step 2 reads `fab/changes/{name}/.status.yaml` when a change is active (an 8th Read), but the SPEC flow stops at `fab resolve --folder` and the tools table lists only '7 project files'. Add a conditional flow line ('[if active] Read: fab/changes/{name}/.status.yaml for stage') and update the Read row, per the constitution rule that SPEC mirrors track skill behavior.

**Verifier**: Confirmed. src/kit/skills/fab-discuss.md:30-34 step 2 reads fab/changes/{name}/.status.yaml for the current stage when fab resolve --folder succeeds, and the skill's Orientation Summary (line 73) displays that stage — so the read is behavior-bearing. SPEC-fab-discuss.md Flow (lines 9-19) omits it an…

### `docs/specs/skills/SPEC-fab-help.md`

#### `a020` [SHOULD-FIX] SPEC-fab-help flow claims the Go subcommand scans src/kit/skills, but fabhelp.go scans the kit cache sibling to the binary

**Location**: Flow diagram, line 14 · **Category**: spec-drift · **Found by**: entry-points

fabhelp.go:81 runs `scanSkills(filepath.Join(kitDir, "skills"))` where kitpath.KitDir() resolves the kit/ directory sibling to the running executable (~/.fab-kit/versions/<v>/kit/) — src/kit/ does not even exist in user projects (constitution Principle V). The SPEC was last touched by #308 (Relocate Kit to System Cache) and never updated for the relocation it describes; the skill body itself correctly says the version is 'read from the cache'. Reword to '(scans skill frontmatter in the kit cache — ~/.fab-kit/versions/<v>/kit/skills/, sibling to the fab binary)'.

**Verifier**: Confirmed. SPEC-fab-help.md:14 claims `fab fab-help` "scans src/kit/skills/*.md frontmatter", but fabhelp.go:74,81 calls kitpath.KitDir() + scanSkills(filepath.Join(kitDir, "skills")), and kitpath.go:21-37 resolves the kit/ directory sibling to the resolved executable — i.e., the cache (~/.fab-kit/v…

### `docs/specs/skills/SPEC-fab-new.md`

#### `a021` [NICE-TO-HAVE] SPEC flow's Step 3 command line omits the conditional --change-id flag central to the backlog collision story

**Location**: ## Flow, Step 3 (line 46) · **Category**: spec-drift · **Found by**: fab-new

The skill passes `--change-id <4char>` when a backlog ID was detected (fab-new.md Step 3), and the SPEC's own collision note relies on it ('Linear re-runs pass no --change-id, so the scan is the only collision guard' — incoherent unless backlog runs DO pass it). Amend the flow line to `fab change new --slug <slug> [--change-id <id> if backlog] --log-args <desc>` so the diagram matches the skill.

**Verifier**: Confirmed. SPEC-fab-new.md:46 shows `fab change new --slug <slug> --log-args <desc>` while the skill (src/kit/skills/fab-new.md Step 3, lines 57-62) conditionally passes `--change-id <4char>` for backlog IDs; the flag exists in the CLI (src/go/fab/cmd/fab/change.go:52). The SPEC is internally incohe…

### `docs/specs/skills/SPEC-fab-operator.md`

#### `a022` [SHOULD-FIX] SPEC claims a 'status-only mode without' tmux; the skill mandates a hard stop

**Location**: Key Properties table ('Requires tmux?' row) and Section Structure item 2 ('outside-tmux degradation') vs skill §2 Tmux Gate and §9 (line 625) · **Category**: spec-drift · **Found by**: fab-operator

The skill says "If `$TMUX` is unset, STOP" and Key Properties says "Yes — hard stop without it"; the SPEC still describes a degraded status-only mode and "outside-tmux degradation" in §2's summary. Update the SPEC rows to the hard-stop behavior. While there, also fix the SPEC's 'Uses /loop?' row ("for proactive monitoring after every send") to match the skill's "3m heartbeat" lifecycle.

**Verifier**: Verified verbatim: SPEC-fab-operator.md:133 "Requires tmux? | Yes for pane map, resolve --pane, monitoring, auto-nudge; status-only mode without" and :22 "outside-tmux degradation" contradict the skill (src/kit/skills/fab-operator.md:55-61 Tmux Gate "If $TMUX is unset, STOP" and :625 "Yes — hard sto…

#### `a023` [SHOULD-FIX] SPEC Resolved Design Decision 2 ('All-auto-answer over two-tier classification') contradicts the current rule-4 Routine/Strategic two-tier model

**Location**: Resolved Design Decisions item 2; also 'operator4' residue in Primitives intro and 'loaded in the operator's own startup section' in Summary · **Category**: staleness · **Found by**: fab-operator

The SPEC's own Answer Model section documents exactly a two-tier model (Routine → auto-answer, Strategic → escalate, plus the 30m idle auto-default), so Decision 2 now records a rejected design as current. Rewrite it to note the supersession (all-auto-answer was later re-split into Routine/Strategic with idle auto-default as the latency mitigation). Also sweep the pre-rewrite naming residue: "operator4 does not duplicate tool tables" (the skill is just fab-operator now) and the Summary's "External tool reference (_cli-external.md) is loaded in the operator's own startup section" which predates the frontmatter `helpers: [_cli-fab, _cli-external]` mechanism.

**Verifier**: All three sub-claims independently confirmed. (1) SPEC-fab-operator.md:144 (Resolved Design Decision 2) records "All-auto-answer over two-tier classification... The two-tier model added pipeline latency without meaningful safety improvement" while the same SPEC's Answer Model (lines 77-98, plus Sect…

### `docs/specs/skills/SPEC-fab-proceed.md`

#### `a024` [SHOULD-FIX] SPEC mirror contradicts itself (and the skill) on whether /fab-proceed loads _preamble

**Location**: Summary (line 5) vs 'Key differences from /fab-fff and /fab-ff' (line 114) · **Category**: spec-drift · **Found by**: fab-proceed

Fix the line-114 bullet to match both the Summary and fab-proceed.md:8 ("Read the `_preamble` skill first"): e.g., "Reads `_preamble.md` but does NOT run preflight or the always-load context layer — delegates those to `/fab-fff`." The SPEC was last updated in PR #342 while the skill changed in batches #391/#392, violating the constitution's skill→SPEC sync constraint.

**Verifier**: Confirmed: SPEC-fab-proceed.md line 5 ("Reads `_preamble.md` (per skill convention) but skips running preflight") directly contradicts line 114 ("Does NOT load `_preamble.md` or run preflight — delegates that to `/fab-fff`"). Ground truth sides with line 5: src/kit/skills/fab-proceed.md:8 instructs…

### `docs/specs/skills/SPEC-fab-status.md`

#### `a025` [SHOULD-FIX] SPEC-fab-status cites the dev-repo path src/kit/VERSION and omits the .status.yaml read the skill depends on

**Location**: ## Flow line 13 + Tools used table (lines 38-41) · **Category**: staleness · **Found by**: change-mgmt

The skill reads the kit VERSION "via `fab kit-path`" (i.e. `$(fab kit-path)/VERSION` in the system cache); `src/kit/VERSION` exists only in the fab-kit dev repo and contradicts constitution Principle V portability. Also, the SPEC's Impact/refactor-warning boxes consume `.status.yaml` `true_impact`/`change_type` (neither is in preflight YAML per preflight.go), yet the flow and Tools table list no `.status.yaml` read. Update the flow line to `$(fab kit-path)/VERSION` and add the scoped `.status.yaml` read to the diagram and the Read row.

**Verifier**: CONFIRMED both halves. (1) VERSION path: docs/specs/skills/SPEC-fab-status.md line 13 reads "├─ Read: src/kit/VERSION, fab/.kit-migration-version" (evidence quote verbatim) and the Tools table Read row (line 40) says "VERSION, migration-version". The canonical skill src/kit/skills/fab-status.md line…

### `docs/specs/skills/SPEC-git-pr-review.md`

#### `a026` [SHOULD-FIX] SPEC Status-commit bookkeeping Gate row omits the timeout path, contradicting both the skill and the SPEC's own flow diagram

**Location**: ### Status-commit bookkeeping (Step 6.5), Gate row (line 125) vs flow diagram line 87 · **Category**: spec-drift · **Found by**: git-pr-review

The skill (git-pr-review.md line 202) skips Step 6.5 'when Step 6 took the `fail` or `timeout` path', and the SPEC's own flow diagram (line 87) says '(skip on no-change / fail / timeout path)', but the prose Gate row drops timeout. Add 'and on the `timeout` path' to the Gate row so the table matches the skill and the diagram.

**Verifier**: Confirmed. SPEC-git-pr-review.md line 125 Gate row enumerates only the fail path and no-change in its "Skipped silently on..." sentence, while the SPEC's own flow diagram (line 87: "skip on no-change / fail / timeout path") and the skill (src/kit/skills/git-pr-review.md line 202: "when Step 6 took t…

#### `a027` [NICE-TO-HAVE] SPEC phase chain starts with a 'waiting' phase that nothing in the kit or Go CLI ever sets

**Location**: Flow diagram, phase-tracking footer (line 94) · **Category**: spec-drift · **Found by**: git-pr-review

The skill's Phase Sub-State Tracking table defines only received/triaging/fixing/pushed/replying; grep confirms no skill (including git-pr.md, which writes no phase) and no Go code writes a `waiting` value to stage_metrics.review-pr.phase. Remove `waiting` from the SPEC chain, or — if a pre-review 'waiting' state is intended — add it to the skill's table with a defined setter (e.g., set by git-pr after PR creation).

**Verifier**: Confirmed spec drift. SPEC-git-pr-review.md:94 lists the phase chain 'waiting → received → triaging → fixing → pushed → replying', but the canonical skill's Phase Sub-State Tracking table (src/kit/skills/git-pr-review.md:210-216) defines only received/triaging/fixing/pushed/replying. No setter exist…

### `docs/specs/skills/SPEC-git-pr.md`

#### `a028` [NICE-TO-HAVE] SPEC-git-pr omits the explicit /git-pr {type} argument from the type-resolution chain — mirror lists 3 sources, skill has 4

**Location**: Summary (line 5) and Flow Step 0b (lines 24–27) · **Category**: spec-drift · **Found by**: git-pr

The skill's Step 0b resolution chain has four sources evaluated in order — explicit argument ('If the user invoked `/git-pr {type}`…'), status, intake, diff — and stores the source vocabulary `explicit, status, intake, diff`. The SPEC's Summary and its Step 0b flow node list only status/intake/diff, dropping the highest-precedence source. Update the Summary to 'Resolves PR type from explicit argument/status/intake/diff' and add an explicit-argument line as the first child of the SPEC's Step 0b node. (Distinct from known f199, which concerns the skill's own missing Arguments section, not the SPEC mirror.)

**Verifier**: Confirmed. SPEC-git-pr.md line 5 reads "Resolves PR type from status/intake/diff." verbatim, and its Step 0b flow node (lines 24-27) lists only three sources (status from Step 0, intake keyword match, git diff fallback). The canonical skill src/kit/skills/git-pr.md Step 0b (lines 43-66) has a four-s…

### `docs/specs/skills/SPEC-hooks.md`

#### `a029` [MUST-FIX] 'Current Hooks' section describes a hook system that no longer exists (2 shell scripts vs 5 Go subcommands)

**Location**: ## Current Hooks, lines 5-14 · **Category**: staleness · **Found by**: spec-hooks-vs-shipped-go-hooks

Rewrite the Current Hooks table to list the five `fab hook` Go subcommands and `.claude/settings.local.json` registration via `fab hook sync`; impact: a reader following the SPEC would look for two shell scripts and a sync script that do not exist.

**Verifier**: CONFIRMED. Evidence verified verbatim: docs/specs/skills/SPEC-hooks.md:5 claims "Two Claude Code hooks exist today, both registered via `src/kit/sync/5-sync-hooks.sh`" with a table (lines 9-10) citing src/kit/hooks/on-stop.sh and on-session-start.sh. Ground truth independently confirmed: src/kit/hoo…

#### `a030` [MUST-FIX] Events table rates UserPromptSubmit fab-kit fit 'No' while a UserPromptSubmit hook is shipped and registered

**Location**: ## Possible Events to Use, lines 91-92 · **Category**: correctness · **Found by**: spec-hooks-vs-shipped-go-hooks

Update the UserPromptSubmit row to 'In use — clears idle_since' and the PostToolUse row to 'In use — artifact-write bookkeeping'; impact: the assessment table actively contradicts the registered hook set.

**Verifier**: Confirmed verbatim. docs/specs/skills/SPEC-hooks.md:92 rates UserPromptSubmit fab-kit fit "No — context injection is skill-level" and line 91 rates PostToolUse "**High**" as an unrealized opportunity, while both hooks are shipped and registered: src/go/fab/cmd/fab/hook.go:154-157 (hookUserPromptCmd,…

#### `a031` [MUST-FIX] 'Proposed Hook Architecture' and 'Registration in 5-sync-hooks.sh' present already-shipped behavior as future work, citing dead artifacts

**Location**: ## Proposed Hook Architecture (Trimmed) + ### Registration in 5-sync-hooks.sh, lines 186-247 (also lines 31-50 framing) · **Category**: spec-drift · **Found by**: spec-hooks-vs-shipped-go-hooks

Convert the section from proposal to as-shipped description (or mark it historical with the shipped equivalent), and delete the 5-sync-hooks.sh registration snippet and map_event reference; impact: the SPEC tells readers fab-kit's core bookkeeping automation is unbuilt when it has shipped.

**Verifier**: All evidence independently confirmed. SPEC-hooks.md:186-247 presents as future work ("◄── NEW") exactly what shipped in 260306-6bba/260310-bvc6: hook.go:254-298 implements the intake.md (InferChangeType + score.Compute "intake") and plan.md (SetAcceptance generated/task_count/acceptance_count/accept…

#### `a032` [SHOULD-FIX] git-pr-review phase list stale: 'waiting' phase gone, 'replying' phase missing

**Location**: ## Hooks Embedded in Skills, line 24 · **Category**: staleness · **Found by**: spec-hooks-vs-shipped-go-hooks

Update the phase enumeration to received/triaging/fixing/pushed/replying; impact: anyone monitoring `.status.yaml` from the SPEC will expect a 'waiting' value that is never written and miss 'replying'.

**Verifier**: Confirmed. SPEC-hooks.md:24 enumerates phases as (waiting/received/triaging/fixing/pushed); the canonical skill src/kit/skills/git-pr-review.md:210-218 defines received/triaging/fixing/pushed/replying and never writes 'waiting' (full-file grep clean; the only 'Waiting' is a printed poll message). Gi…

#### `a033` [SHOULD-FIX] 'yq Dependency in Hooks' section inventories three src/kit files that no longer exist; hook yq usage is zero

**Location**: ## yq Dependency in Hooks, lines 153-169 · **Category**: staleness · **Found by**: spec-hooks-vs-shipped-go-hooks

Trim the section to the one surviving yq consumer (git-pr-review phase writes) and delete the dead-file inventory and silent-failure problem statement; impact: 'Problem' framing describes failure modes of scripts that were deleted.

**Verifier**: Confirmed at stated severity (should-fix). Quote at docs/specs/skills/SPEC-hooks.md:157 and the table at :163-169 are exact. src/kit/ contains no .sh files at all (only VERSION/migrations/scaffold/schemas/skills/templates); hooks/on-stop.sh, hooks/on-session-start.sh, scripts/fab-doctor.sh, and 5-sy…

#### `a034` [SHOULD-FIX] 'Proposal: fab runtime subcommands' superseded — no `fab runtime` command exists; hooks call the internal runtime package directly

**Location**: ### Proposal: fab runtime subcommands, lines 171-180 · **Category**: spec-drift · **Found by**: spec-hooks-vs-shipped-go-hooks

Mark the proposal as superseded by the inline Go hook handlers (the yq elimination happened, but via in-process calls, not a `fab runtime` subcommand); impact: readers may search for or attempt to use a nonexistent `fab runtime` command.

**Verifier**: Independently confirmed. Evidence quote matches docs/specs/skills/SPEC-hooks.md:171-180 verbatim; the Proposed Hook Architecture diagram (:192-196) repeats the nonexistent `fab runtime set-idle/clear-idle` calls, compounding the drift. Ground truth: no `fab runtime` command exists (src/go/fab/cmd/fa…

#### `a035` [SHOULD-FIX] 'What this changes in skills' table contradicts shipped skill state and the SPEC's own 1.10.0 note (fab-clarify/fab-new rows)

**Location**: ### What this changes in skills, lines 210-221 (vs line 52) · **Category**: spec-drift · **Found by**: spec-hooks-vs-shipped-go-hooks

Replace the future-tense table with the actual outcome per skill (plan-write bookkeeping dropped; fab-new keeps explicit scoring + verify-and-override for type; fab-clarify deliberately keeps recompute); impact: the table mixes shipped, partially-shipped, and rejected changes and contradicts the SPEC's own removal note.

**Verifier**: Independently confirmed. (1) All cited quotes verified: SPEC-hooks.md table rows (fab-new at line 216, fab-clarify at line 220 — finding says 221, one-line offset, content exact) and line 52's "Intake scoring is recomputed by /fab-clarify (intake-only) directly, not via a write hook". (2) Shipped st…

#### `a036` [NICE-TO-HAVE] Runtime-file schema description outdated: 'agent.idle_since' singleton vs per-session '_agents[session_id]' map with GC

**Location**: ## Current Hooks, line 12 · **Category**: staleness · **Found by**: spec-hooks-vs-shipped-go-hooks

Describe the `_agents` per-session map, the user-prompt idle-clear, and the GC sweep; impact: the singleton `agent.` schema no longer matches anything the binary reads or writes.

**Verifier**: Confirmed. SPEC-hooks.md:12 quote is verbatim; ground truth verified independently: runtime.go:31 agentsKey="_agents", runtime.go:41-70 AgentEntry per-session record (change/pid/tmux_server/tmux_pane/transcript_path), hook.go:24 gcInterval=180s, _cli-fab.md:172-178 documents stop/user-prompt/session…

### `docs/specs/srad.md`

#### `a037` [MUST-FIX] srad.md Assumptions-table contract contradicts _srad: Scores column 'optional' and Certain/Unresolved rows omitted

**Location**: § Dimension Score Persistence (line 54, table lines 57-60) and § Assumptions Summary (lines 105, 110-118) · **Category**: spec-drift · **Found by**: non-skill-spec-docs-drift-sweep

Rewrite both sections to match _srad.md: Scores column required on every row (the srad.md:110-115 example table also lacks it entirely), intake `## Assumptions` includes all four grades, Unresolved rows carry `Asked/Deferred` status context, and the Certain grade-table cell (srad.md:82 'Not mentioned — not worth noting') should read 'Noted in Assumptions summary' per _srad.md:41. Impact: an author following srad.md writes intake Assumptions tables that `fab score` cannot fully parse (no Scores → no dimension stats) and that undercount Certain decisions, deflating total_decisions/cover and making the 3.0 intake gate misfire.

**Verifier**: All evidence quotes confirmed verbatim. docs/specs/srad.md:54 says "optional `Scores` column"; :82 says Certain "Not mentioned — not worth noting"; :105 scopes ## Assumptions to "Confident or Tentative"; :110-115 example table has no Scores column; :118 says "Certain grades are omitted... Unresolved…

#### `a038` [SHOULD-FIX] srad.md Example 1 composite arithmetic is wrong for two of three rows under the documented formula

**Location**: § Worked Examples, Example 1 table (lines 246-248) · **Category**: correctness · **Found by**: non-skill-spec-docs-drift-sweep

Correct the two composites to 15.75 and 42.75 (grades are unaffected: still Unresolved-via-Critical-Rule and Tentative). While in this section, note srad.md shares both _srad defects flagged by the prior audit: the closed-integer grade bands (lines 43-48: '60–84', '30–59', '0–29' leave continuous composites like 84.5 unmapped — ironically the example annotations '(30 ≤ 42.0 < 60)' already imply the half-open-interval fix the table should state) and the Critical-Rule numeric ambiguity (line 50 'if R < 25 AND A < 25' vs the line 226 prose 'low Reversibility AND low Agent Competence', where 'Low' is rubric band 0–39). Impact: worked examples are the template agents pattern-match when scoring; wrong arithmetic plus ambiguous bands propagate into real intake scores.

**Verifier**: Confirmed by independent computation against docs/specs/srad.md:36's formula (0.25*S + 0.30*R + 0.25*A + 0.20*D): line 247 (15,10,20,20) = 15.75 not 15.5; line 248 (20,50,55,45) = 42.75 not 42.0. Row 1 (12.5) and all of Example 2 are correct (94.35 rounds to the printed 94.3). Grades are unaffected,…

### `docs/specs/templates.md`

#### `a039` [SHOULD-FIX] docs/specs/templates.md .status.yaml section drifts from the shipped template and the Go state machine

**Location**: ## .status.yaml — State Vocabulary table (lines 13-19) and Template block (lines 27-54) · **Category**: spec-drift · **Found by**: lens-templates-config

Three drifts: (1) the State Vocabulary table omits `ready` even though Go ValidStates is {pending, active, ready, done, failed, skipped} and the section's own template comments four lines later say "# pending | active | ready | done"; `skipped` is not "(reserved)" — `fab status skip` is a live transition for every stage except intake; and `failed` is "Used by: review" while status.go also allows it for review-pr. (2) The fenced template lacks `id:`, `issues: []`, and `prs: []`, all present in src/kit/templates/status.yaml and CLI-managed (`fab status add-issue`/`add-pr`), plus the lazy `true_impact` comment. (3) The field note "change_type classifies the change for dynamic confidence gate thresholds (see Change Types for per-type values)" overstates — the gate is flat 3.0 for all types per change-types.md and _preamble. Sync the table and template against status.go AllowedStates and the shipped file.

**Verifier**: All three drifts independently confirmed. (1) State Vocabulary table (docs/specs/templates.md:13-19): omits `ready` though status.go:18 ValidStates includes it and the spec's own template comments (lines 35-38) list it; `skipped` is not "(reserved)" — fab status skip is registered (cmd/fab/status.go…

#### `a040` [SHOULD-FIX] docs/specs/templates.md intake.md section is the pre-1.10.0 template — BLOCKING/DEFERRED labels and New/Modified/Removed subsections no longer exist

**Location**: ## intake.md section, lines 91-137 · **Category**: spec-drift · **Found by**: lens-templates-config

The shipped template (src/kit/templates/intake.md, fixed in batch 2 f062+g3-1) says the opposite: "SRAD handles prioritization at plan generation (apply entry) — no need for explicit blocking/deferred labels here" and replaced the `### New Files / ### Modified Files / ### Removed Files` subsections with a flat annotated list ("Mark each with (new), (modify), or (remove)"). The spec mirror was never updated, still references the deleted spec stage twice, and its `## Origin`/`## Why` guidance comments also lag the template ("1-3 sentences" vs "A single sentence is almost never enough"). Replace the section's fenced block with the current template verbatim, and prune the matching `[BLOCKING]`/`[DEFERRED]` glossary rows (docs/specs/glossary.md:133-134) in the same pass. templates.md is not part of the deferred uliv docs sweep, so this is unowned drift.

**Verifier**: CONFIRMED. Evidence verbatim at docs/specs/templates.md:113-120 ("Mark each with priority: [BLOCKING] must resolve before spec, [DEFERRED] can resolve during spec... Maximum 3 [BLOCKING] questions" plus `- [BLOCKING] {question}` / `- [DEFERRED] {question}` placeholders) and the `### New Files / ###…

### `docs/specs/user-flow.md`

#### `a041` [SHOULD-FIX] user-flow.md claims failed is 'review only' and lists five stage states, contradicting the workflow schema it cites as source of truth

**Location**: §3 intro (line 84) and §4 diagram note (lines 182-184) · **Category**: spec-drift · **Found by**: non-skill-spec-docs-drift-sweep

Change both annotations to 'review and review-pr only', and fix the state count: §4's own diagram and workflow.yaml define six states including `skipped`, which line 84's 'five states' list omits. Impact: the visual spec teaches that review-pr cannot fail, contradicting the review-pr:failed terminal path batch 4 just built into _pipeline and the State Table's 'review-pr (fail)' row.

**Verifier**: CONFIRMED. Evidence quotes verified verbatim: docs/specs/user-flow.md:84 "one of five states: `pending`, `active`, `ready`, `done`, or `failed` (review only)" and :183 "¹ Review stage only". Ground truth contradicts both: (1) src/kit/schemas/workflow.yaml:32-37 defines SIX states including `skipped`…

### `src/go/fab/internal/config/config.go`

#### `a042` [SHOULD-FIX] stage_hooks is a live Go-consumed config key documented nowhere in the kit

**Location**: Config struct, line 18; consumed at internal/status/status.go:603-621 · **Category**: undocumented-config · **Found by**: lens-templates-config

runStageHook loads `stage_hooks.{stage}.pre/post` from fab/project/config.yaml and shells them out on every stage transition, yet the key appears in no current documentation surface: not the scaffold config.yaml, not the fab-setup config menu, not _cli-fab.md, not docs/specs/templates.md or architecture.md, and not docs/memory (only archived change artifacts from the Rust-port change mention it). Users cannot discover the feature, and the config-contract audit can't tell deliberate from leftover. Add a commented `# stage_hooks:` example to the scaffold and a short row in _cli-fab.md (or docs/specs/architecture.md's config section); alternatively, if the hook mechanism is considered internal/retired, remove it from config.go and status.go with tests.

**Verifier**: Confirmed at stated severity (should-fix, undocumented-config). Evidence exact at src/go/fab/internal/config/config.go:18; consumption at internal/status/status.go:603-621 via Start (pre, status.go:116 — failing pre hook BLOCKS stage start) and Finish (post, status.go:178), executing user shell comm…

### `src/go/fab/internal/status/status.go`

#### `a043` [SHOULD-FIX] Go state machine permits transitions into states its own schema forbids (advance ship/review-pr → ready; skip intake → skipped), bricking preflight

**Location**: AllowedStates (lines 21-28) vs defaultTransitions/stageTransitions (lines 37-61); Validate (lines 502-535) · **Category**: correctness · **Found by**: lens-pipeline-coherence

A single documented CLI call — `fab status advance <change> ship` (or review-pr), advertised generically as "active → ready" in fab-continue.md Step 4 and _cli-fab.md:58 — succeeds and writes a state that AllowedStates forbids; since preflight runs status.Validate, every subsequent skill invocation then STOPs with "State 'ready' not allowed for stage ship". Same for `fab status skip <change> intake` (default skip allows active→skipped; intake's allowed set is {active, ready, done}). Either add the missing states to AllowedStates (simplest: ship/review-pr gain "ready", intake gains "skipped") or make lookupTransition reject targets not in AllowedStates with an actionable error, plus a test asserting every reachable transition target is schema-valid.

**Verifier**: CONFIRMED by source reading and empirical reproduction. Built the fab binary and reproduced all three variants against a synthetic project: (1) `fab status advance <chg> ship` exits 0 and writes ship: ready; `fab preflight` then exits 1 with "Invalid .status.yaml: State 'ready' not allowed for stage…

### `src/kit/scaffold/fab/project/code-review.md`

#### `a044` [SHOULD-FIX] Scaffold code-review.md rework escalation still names the removed-era "revise spec" path (dev-repo copy says "revise tasks"/"revise spec"), and the "Max cycles" knob has no consumer

**Location**: ## Rework Budget (line 44) · **Category**: spec-drift · **Found by**: lens-pipeline-coherence

The canonical escalation paths since the spec-stage merge are "Revise plan" / "Revise requirements" (_pipeline.md Escalation rule: "escalate to 'Revise plan' or 'Revise requirements' after 2 consecutive 'fix code' attempts"; fab-continue Verdict table matches). Update the scaffold line, and the staler dev-repo instance fab/project/config sibling fab/project/code-review.md:44 ("revise tasks" or "revise spec" — both removed stage-era terms). Also resolve the budget linkage ambiguity: the file presents "Max cycles: 3" as project policy ("Applies to /fab-fff and /fab-ff auto-rework loops") while _pipeline.md hard-codes "Auto-Rework Loop (up to 3 cycles)" and never consults code-review.md § Rework Budget — either have _pipeline read the policy value or annotate the scaffold field as informational/hard-coded like the Parsimony Pass comment does.

**Verifier**: Confirmed verbatim: src/kit/scaffold/fab/project/code-review.md:44 says escalate to "revise plan" or "revise spec"; dev-repo fab/project/code-review.md:44 says "revise tasks" or "revise spec". Canonical paths are "Revise plan"/"Revise requirements" (_pipeline.md:89-92 escalation rule; fab-continue.m…

#### `a045` [SHOULD-FIX] Scaffold code-review.md Rework Budget names a nonexistent 'revise spec' escalation path, and its Max-cycles knob is consumed by nothing

**Location**: ## Rework Budget, lines 38-44 · **Category**: spec-drift · **Found by**: lens-templates-config

The actual rework paths post-1.10.0 are "Fix code / Revise plan / Revise requirements" (fab-continue.md:161-163; _pipeline.md:92 "escalate to 'Revise plan' or 'Revise requirements'"); "revise spec" no longer exists. Replace with "revise requirements". This repo's local fab/project/code-review.md:44 is even staler ('escalate to "revise tasks" or "revise spec"') — fix both. Additionally, the section's comment promises configurability ("Max auto-rework cycles before escalating ... Applies to /fab-fff and /fab-ff auto-rework loops") but _pipeline.md hard-codes "Auto-Rework Loop (up to 3 cycles)" and never reads code-review.md's value — a project that edits `Max cycles` is silently ignored. Either have _pipeline.md read the budget (default 3) or annotate the scaffold knob as informational like the Parsimony Pass comment does ("hard-coded in the kit and NOT configurable here").

**Verifier**: Confirmed verbatim: scaffold src/kit/scaffold/fab/project/code-review.md:44 escalates to "revise plan" or "revise spec"; local fab/project/code-review.md:44 is staler ("revise tasks" or "revise spec"). Ground truth contradicts both: _pipeline.md:92 escalation paths are "Revise plan" or "Revise requi…

### `src/kit/scaffold/fab/project/config.yaml`

#### `a046` [MUST-FIX] Scaffold config template still seeds the dead `spec:` stage in stage_directives — every new project gets a zombie key and loses the apply directives

**Location**: stage_directives block, lines 31-38 · **Category**: staleness · **Found by**: fab-setup

Move the two directives under `apply:` and delete the `spec:` key. The constitution (since 260601-j6cs) says "there is no separate `spec` stage", and migration 1.9.7-to-1.10.0 explicitly "Relocates `stage_directives.spec` content ... into" apply and removes the key for existing projects — but fab-setup Config Create Mode reads this scaffold as "the starting template" (fab-setup.md:151) and only substitutes {PROJECT_NAME}/{PROJECT_DESCRIPTION}/{SOURCE_PATHS}, so every freshly bootstrapped project ships a `spec:` block no stage reads. Because `fab init` stamps new projects at the engine version, the relocating migration never runs for them, so the zombie key is permanent and the GIVEN/WHEN/THEN defaults are silently dead (the dev repo's own config.yaml has them under `apply:`). Reachable through the skill's own flow; missed by the 2026-06-11 review (scaffold wasn't in scope — only templates/intake.md spec-residue was fixed via f062).

**Verifier**: Confirmed end-to-end. Scaffold config (src/kit/scaffold/fab/project/config.yaml:31-38) seeds stage_directives.spec with the GIVEN/WHEN/THEN + [NEEDS CLARIFICATION] directives and leaves apply: [] empty — quote matches verbatim. Constitution line 34 (amended 2026-06-01) says no spec stage exists; mig…

#### `a047` [MUST-FIX] Scaffold config.yaml seeds a dead stage_directives.spec key whose [NEEDS CLARIFICATION] directive contradicts the apply-stage contract

**Location**: stage_directives block, lines 31-38 · **Category**: spec-drift · **Found by**: lens-templates-config

This answers the brief's open question: it IS a kit problem, not just this project's local config. New projects scaffolded at >=1.10.0 never run the 1.9.7-to-1.10.0 migration (which relocates spec->apply for old projects), so they get a fresh `spec:` key for a stage that no longer exists. Worse, the migration relocated these directives verbatim into this repo's local `apply:` block (fab/project/config.yaml:19-21), where "Mark ambiguities with [NEEDS CLARIFICATION]" directly contradicts the plan template's rule "NO [NEEDS CLARIFICATION] MARKERS: those are an intake-only construct" (src/kit/templates/plan.md:36) — and every apply agent reads config.yaml via the always-load layer, so the contradictory directive reaches it. Fix the scaffold to the 6-stage key set with intake-appropriate defaults (move the marker directive to `intake:` or drop it), and clean this repo's local apply directives.

**Verifier**: Confirmed end-to-end. Scaffold (src/kit/scaffold/fab/project/config.yaml:31-38) ships a stage_directives.spec key for a stage removed by j6cs; git log shows the scaffold was last touched by 096be509 (#370) — j6cs missed it. Post-j6cs schema is documented as 4 keys (intake/apply/review/hydrate) in do…

### `src/kit/schemas/workflow.yaml`

#### `a048` [MUST-FIX] workflow.yaml still defines the 7-stage pipeline with the removed spec stage

**Location**: stages list (lines 64-82) and stage_numbers (lines 212-219) · **Category**: spec-drift · **Found by**: lens-templates-config

Regenerate the schema for the 6-stage pipeline: delete the spec stage block, change apply's `requires: [spec]` to `[intake]`, renumber stage_numbers 1-6, and bump metadata.version/last_updated. The file directly contradicts the constitution's six-stage rule and docs/memory/pipeline/index.md, which already claims "`workflow.yaml` schema — 6-stage pipeline". Nothing in src/go or src/kit/skills consumes the file (its own header claim "consumed by all scripts and skills" is also false), so alternatively retire it and repoint docs/specs/user-flow.md:201 ("Source of truth: src/kit/schemas/workflow.yaml") and docs/memory/pipeline/schemas.md at the Go status state machine.

**Verifier**: Confirmed in full. src/kit/schemas/workflow.yaml still defines the 7-stage pipeline: spec stage block at lines 64-72 (generates: spec.md), apply requires: [spec] at line 78, and 7-entry stage_numbers with spec: 2 at lines 212-219; metadata.last_updated is 2026-03-04, predating the 2026-06-01 spec->a…

### `src/kit/skills/_cli-external.md`

#### `a049` [SHOULD-FIX] _cli-external 'New change (from backlog)' spawn flow is stale: unconditional /git-branch step ignores fab-new's inline branch creation, and step 1 misstates the worktree's branch

**Location**: § Operator Spawning Rules → 'New change (from backlog)', lines 59-63 · **Category**: staleness · **Found by**: misc-internal

fab-new.md Step 11 ('After activating the change, create or check out the matching git branch inline') now performs the alignment itself — its case-4 rename guard explicitly targets 'a disposable `wt create` name' — so by the time intake advances the branch already matches and this step is a no-op presented as the alignment mechanism. It also drifts from fab-operator.md:90, which makes the same send conditional ('if the tab's git branch doesn't match the change folder name'). Rewrite step 3 as a conditional backstop for fab-new's non-fatal Step 11 ('fab-new aligns the branch inline; if the branch still doesn't match after intake, send /git-branch') and align fab-operator.md:200's unconditional auto-nudge wording. Secondary inaccuracy in step 1, line 61: 'creates on default branch' — wt create with no branch argument creates a new random-named branch off the default (per `wt create --help`: 'creates an exploratory worktree with a random name'), which is precisely the local-only no-change branch fab-new's rename guard keys on; say 'creates a disposable random-named branch off the default'.

**Verifier**: Confirmed at _cli-external.md:59-63. (1) Step 3's unconditional /git-branch send is stale: fab-new.md Step 11 (lines 129-165, added in 91a2a998 #322, after the spawn-flow text existed from the #308 era) aligns the branch inline — its case-4 rename guard (line 159) explicitly targets 'a disposable `w…

#### `a050` [SHOULD-FIX] _cli-external: wt new-change flow claims the fresh worktree "creates on default branch" — contradicted by both git's checkout-exclusivity rule and wt's actual behavior

**Location**: § wt — Operator Spawning Rules, "New change (from backlog)" (line 61) · **Category**: staleness · **Found by**: git-state-safety-sweep

Correct line 61 to: "creates the worktree on a new disposable branch named after the worktree (the default branch stays checked out in the main worktree)" — which also explains why step 3's later /git-branch hits the rename-guard path. Impact: operators reasoning from the documented (wrong) branch model misjudge which git-branch table row will fire and whether a disposable branch needs cleanup.

**Verifier**: CONFIRMED. (1) Quote verified verbatim at src/kit/skills/_cli-external.md:61. (2) Ground truth verified in the wt source (/home/sahil/code/sahil87/wt): src/cmd/wt/create.go:215-221 — when the branch arg is omitted, wt calls CreateExploratoryWorktree(finalName,...) and reports `Branch: <finalName>`;…

#### `a051` [NICE-TO-HAVE] wt create --reuse documented without its --worktree-name requirement

**Location**: § `wt create` Flags table, line 34 · **Category**: correctness · **Found by**: misc-internal

The actual flag contract is '--reuse  Reuse existing worktree if name collides (requires --worktree-name)' (wt create --help). An agent composing its own respawn command from this table alone could pass --reuse without --worktree-name and get a CLI error; the trigger is a name collision, not a generic 'match'. Amend the cell to 'Reuse the existing worktree when the name collides (requires --worktree-name). Used for autopilot respawns.'

**Verifier**: Confirmed. Evidence quote appears verbatim at src/kit/skills/_cli-external.md:34. Ground truth verified two ways: (1) live `wt create --help` prints '--reuse  Reuse existing worktree if name collides (requires --worktree-name)'; (2) wt source at /home/sahil/code/sahil87/wt/src/cmd/wt/create.go:54-59…

### `src/kit/skills/_cli-fab.md`

#### `a052` [SHOULD-FIX] _cli-fab claims `fab batch archive` 'exits non-zero only when failed > 0', but it also exits 1 on empty/unresolvable sets with failed == 0

**Location**: ## fab batch, line 494 · **Category**: correctness · **Found by**: exit-code-contract-vs-go-sweep

Either document the two additional exit-1 paths (no archivable changes under --all/no-args; all named targets unresolvable or not archivable) in _cli-fab.md:494, or change the binary to treat the empty --all case as a benign exit-0 no-op. Impact: a clean-repo `fab batch archive` no-op exits 1, which _preamble.md:242's generic failure rule ('any fab command ... that exits non-zero → STOP and surface stderr') escalates into a spurious abort; this empty-set path also interacts with the unreachable 'already archived, skipping' line (finding 1): a single genuinely-archived name falls through 'could not resolve' into the exit-1 'No valid changes' path instead of the documented skip.

**Verifier**: Confirmed. _cli-fab.md:494 claims 'Exits non-zero only when failed > 0', but batch_archive.go has two os.Exit(1) paths with failed==0, both before archiveLoop (so the documented footer never prints): lines 65-68 (empty --all/no-args set, 'No archivable changes found.') and lines 92-95 (all named tar…

#### `a053` [SHOULD-FIX] Documented re-archive soft skip (exit 0) is unreachable — genuinely archived changes exit 1 with 'No change matches'

**Location**: ## fab change (extended subcommand details), lines 42, 46 (also fab-archive.md:67 and fab-archive.md:110 'Idempotent? Yes — re-archive is a soft skip') · **Category**: correctness · **Found by**: exit-code-contract-vs-go-sweep

Either make `fab change archive` resolve against archive/ too (return the soft-skip when the name matches an archived folder), or correct _cli-fab.md:42/46 + fab-archive.md:67/110 to state that re-archiving a fully archived change exits 1 'No change matches' and the soft skip applies only to the source-exists+dest-exists edge case (e.g., after a restore that left the archive copy). Note: distinct from known-open f085, which covers the skill's preflight-first flow; this is the binary-level contract being false. Impact: /fab-archive's documented idempotent re-run never sees the `already archived:` signal it is told to treat as a clean no-op.

**Verifier**: Independently confirmed, including empirically. Built fab-go and reproduced in a temp fixture: first `fab change archive abcd` exits 0 with YAML; re-running after the move exits 1 with 'ERROR: No change matches "abcd".' (same for the full folder name); the exit-0 'already archived: abcd' line fires…

#### `a054` [SHOULD-FIX] "All hook subcommands exit 0" is false for `fab hook sync`

**Location**: ## fab hook, line 168 · **Category**: correctness · **Found by**: cli-fab-ref

Scope the claim to the four event handlers: in src/go/fab/cmd/fab/hook.go the session-start/stop/user-prompt/artifact-write RunE functions all `return nil // swallow`, but hookSyncCmd (hook.go:303-326) returns errors from resolve.FabRoot() and hooklib.Sync(), so `fab hook sync` exits 1 via main.go's error path. Reword to "The four event handlers always exit 0 ... `sync` reports errors normally (non-zero exit)."

**Verifier**: Confirmed. _cli-fab.md '## fab hook' states verbatim 'All hook subcommands exit 0 — errors silently swallowed' while listing sync in the subcommand table. Go source contradicts it: the four event handlers in src/go/fab/cmd/fab/hook.go all 'return nil // swallow', but hookSyncCmd (hook.go:304-326) re…

#### `a055` [SHOULD-FIX] Undocumented config-driven stage_hooks (pre/post) run by fab status start/finish, with after-save failure semantics

**Location**: ## fab status (extended subcommand details), "Side effects of `finish`" paragraph (line 78); cross-ref § fab impact Consumers (line 291) · **Category**: correctness · **Found by**: cli-fab-ref

src/go/fab/internal/status/status.go runs project-configurable shell hooks from config.yaml `stage_hooks:` — `start` runs the stage's `pre` hook and a failure blocks the transition (status.go:116), while `finish` runs the `post` hook AFTER the transition is saved (status.go:178), so a failing post-hook yields a non-zero exit with the stage already done — re-running `finish` per _preamble's generic STOP/re-run failure rule would then hit an invalid done→done transition. `finish apply|hydrate` also best-effort writes `true_impact` (status.go:175). Document these side effects in the fab status section; today the only allusion is § fab impact's "the apply-finish + hydrate-finish hooks", a term defined nowhere (and easily confused with § fab hook). If `stage_hooks` is actually dead, flag it for removal instead.

**Verifier**: Independently confirmed against src/go/fab/internal/status/status.go: pre hook at line 116 blocks `start` on failure; `finish` saves the transition (line 170) before running the post hook (line 178), so a failing post hook exits non-zero with the stage already done — and a re-run per _preamble.md:24…

#### `a056` [SHOULD-FIX] fab status advance on ship/review-pr writes a schema-forbidden 'ready' state that permanently bricks preflight for the change

**Location**: § fab status (extended subcommand details), advance row (line 58); also fab-continue.md dispatch rows 57-58 · **Category**: correctness · **Found by**: lens-cli-contract

Go's default advance transition has no stage guard, but AllowedStates excludes 'ready' for ship and review-pr (src/go/fab/internal/status/status.go:26-27). Empirically verified: `fab status advance <id> ship` exits 0 and writes `ship: ready`, after which every `fab preflight` exits 1 with `Invalid .status.yaml: State 'ready' not allowed for stage ship` — locking all preflight-based skills out of the change. Fix the Go to reject advance for ship/review-pr (and update _cli-fab's advance row to note 'intake/apply/review/hydrate only', per the constitution's CLI-change rule). Relatedly, fab-continue.md rows 57-58 key ship and review-pr dispatch on "`active`/`ready`" — a state those stages can never legally hold; trim to `active`.

**Verifier**: Independently confirmed end-to-end. (1) Evidence verbatim at src/kit/skills/_cli-fab.md:58 (advance row, no stage restriction); fab-continue.md:57-58 confirmed keying ship/review-pr dispatch on `active`/`ready`. (2) Go ground truth: status.go AllowedStates excludes 'ready' for ship (line 26) and rev…

#### `a057` [NICE-TO-HAVE] Archive partial-failure outcome (YAML on stdout + non-zero exit) undocumented

**Location**: ## fab change (extended subcommand details), archive output paragraph (line 46) · **Category**: ergonomics · **Found by**: cli-fab-ref

There is a third stdout/exit combination: when the move succeeds but the backlog mark genuinely fails, src/go/fab/cmd/fab/archive.go:36-39 prints the full YAML report and then returns the error (non-zero exit). An agent parsing stdout sees valid `action:/move:/...` YAML yet gets a failure exit, which neither documented outcome covers. Add one sentence: "Partial failure (move ok, backlog write failed): YAML is still emitted, then the command exits non-zero — the archive itself succeeded."

**Verifier**: Confirmed. _cli-fab.md:46 documents exactly two archive stdout/exit outcomes (YAML+exit-0; plain 'already archived' line+exit-0), but src/go/fab/cmd/fab/archive.go:33-39 implements a third: when ArchiveWithBacklog (internal/archive/archive.go:128-140) returns a non-nil result with a backlog-mark err…

#### `a058` [NICE-TO-HAVE] `fab change archive` with no argument prints help and exits 0 — the documented required <change> guard is silently passable

**Location**: ## fab change (extended subcommand details), line 42 (Usage column) · **Category**: idempotency · **Found by**: exit-code-contract-vs-go-sweep

Replace the Help()-and-return-nil shim with `cobra.ExactArgs(1)` (matching `restore`) so a missing argument exits non-zero. Impact: a skill or script that interpolates an empty unquoted variable (`fab change archive $X`) gets exit 0 plus help text instead of an error, silently passing _preamble's exit-code failure rule — same class as the score --check-gate gate bug, though lower stakes.

**Verifier**: Independently confirmed by building the CLI and running it: `fab change archive` with no args exits 0 with help on stdout (src/go/fab/cmd/fab/archive.go:18-22, MaximumNArgs(1) + Help() shim), while `fab change restore` exits 1 (ExactArgs(1), archive.go:57) — exactly as the _cli-fab.md:42-43 usage co…

#### `a059` [NICE-TO-HAVE] artifact-write's git auto-staging side effect omitted from the hook table

**Location**: ## fab hook, artifact-write row (line 175) and paragraph (line 178) · **Category**: staleness · **Found by**: cli-fab-ref

src/go/fab/cmd/fab/hook.go:219-224 also runs `git add <change>/.status.yaml <change>/.history.jsonl` (best-effort) on every artifact write — this explains why status files show up pre-staged during /git-pr commits. Append "; auto-stages the change's `.status.yaml`/`.history.jsonl` (best-effort `git add`)" to the artifact-write description.

**Verifier**: Confirmed. hook.go:218-224 (src/go/fab/cmd/fab/hook.go) runs best-effort `git add` of the change's .status.yaml/.history.jsonl on every matched artifact write; the Go doc comment (hook.go:180-181) even names "git staging for status/history files" as part of the handler's job, yet _cli-fab.md's hook…

#### `a060` [NICE-TO-HAVE] _cli-fab's own 'remaining commands' index omits fab migrations-status and fab memory-index

**Location**: ### Commands covered in `_preamble` Common fab Commands (line 27) · **Category**: staleness · **Found by**: lens-helper-integrity

The file contains full `## fab migrations-status` and `## fab memory-index` sections (lines ~232 and ~331) that are absent from this in-file index of 'the remaining commands'. Add both to the parenthetical (or drop the exhaustive enumeration in favor of 'the remaining commands and extended flag details') so the index cannot silently drift as new subcommands gain sections.

**Verifier**: Confirmed. src/kit/skills/_cli-fab.md:27's parenthetical enumerates 11 'remaining commands' but omits fab migrations-status and fab memory-index, both of which have full sections in the same file (lines 232 and 331) and are not among the 6 commands covered by _preamble.md Common fab Commands (lines…

### `src/kit/skills/_generation.md`

#### `a061` [SHOULD-FIX] Plan Generation walk never emits the plan's ## Assumptions section it depends on; Assumptions handling is asymmetric and conflicts with the scaffolded templates on the zero-assumptions case

**Location**: § Plan Generation Procedure steps 1–7 (esp. step 3 bullet, lines 75–78) vs. § Intake Generation Procedure step 4 (line 41) · **Category**: ergonomics · **Found by**: gen-review-helpers

Step 3 routes under-specified requirements into "the plan's `## Assumptions` section", but no numbered step (1–7) instructs filling that section or updating its `{N} assumptions (...)` count line — unlike the intake procedure's explicit step 4. The agent must bridge via _srad's general Assumptions Summary Block rule, and _srad.md:115 ("If 0 assumptions were made, omit the Assumptions summary entirely") conflicts with both templates, which pre-scaffold the section with a placeholder row; the intake step 4 verb "Append" likewise mismatches the already-scaffolded template section. Add an explicit step to the plan walk ("Fill `## Assumptions` per _srad § Assumptions Summary Block — three grades; delete the scaffolded section when zero assumptions; update the count line"), change intake step 4 "Append" to "Fill in (or remove, when zero)", and mirror in SPEC-_generation.md's flow diagram (whose plan branch also lacks the Assumptions step).

**Verifier**: Confirmed at stated severity (should-fix, ergonomics). Evidence quote verbatim at src/kit/skills/_generation.md:77-78; Plan Generation steps 1-7 never instruct filling plan.md ## Assumptions or its count line, while Intake step 4 (line 41) does — and its verb "Append" mismatches the pre-scaffolded t…

#### `a062` [NICE-TO-HAVE] _generation consumer routing omits fab-continue's Intake Generation paths

**Location**: Header blockquote (lines 11-13) · **Category**: staleness · **Found by**: lens-helper-integrity

fab-continue also consumes the Intake Generation Procedure: its intake-`active` dispatch row (fab-continue.md:52, 'generate intake if missing (read `.claude/skills/_generation/SKILL.md` first — Intake Generation Procedure)'), its stage-conditional-helpers note (line 11, 'intake-`active` regeneration'), and the Reset Flow intake reset (line 194). Batch 2's f060 fixed this consumer list to five skills, then batch 3's f122 formalized fab-continue's intake-regeneration read without updating the bucketing. Amend the blockquote so fab-continue is listed under both procedures, keeping the header accurate for future editors of the Intake Generation Procedure.

**Verifier**: Confirmed. _generation.md:11-13 (and frontmatter description line 3) bucket fab-continue under Plan Generation only, but fab-continue consumes the Intake Generation Procedure at three verified points: fab-continue.md:52 (intake-`active` dispatch row explicitly says 'read `.claude/skills/_generation/…

#### `a063` [NICE-TO-HAVE] Stale 'auto-clarify' term in the orchestration carve-out — no consumer has an auto-clarify step since 1.10.0

**Location**: header Orchestration note, line 17 · **Category**: staleness · **Found by**: gen-review-helpers

fab-clarify.md:209 confirms "the former `/fab-ff` and `/fab-fff` auto-clarify steps were removed in 1.10.0" and no skill file contains an auto-clarify step today, so the carve-out points agents at functionality that does not exist in any consumer. Drop "auto-clarify" from the list (or replace with a current concern such as "verdict transitions").

**Verifier**: Independently confirmed. Evidence quote verified verbatim at src/kit/skills/_generation.md:17. Cross-ref verified: fab-clarify.md:209 states the former /fab-ff and /fab-fff auto-clarify steps were removed in 1.10.0, and grep shows no skill source contains an auto-clarify step today (only the stale c…

#### `a064` [NICE-TO-HAVE] Consumer mapping is stale again: fab-continue now also runs the Intake Generation Procedure (intake regeneration), so the 'disjoint consumer groups' claim is wrong

**Location**: header note lines 11–13 and frontmatter description (line 3) · **Category**: spec-drift · **Found by**: gen-review-helpers

Since batch 1/3, fab-continue.md:52 dispatches "generate intake if missing (read `.claude/skills/_generation/SKILL.md` first — Intake Generation Procedure)", so fab-continue belongs to both consumer groups — a regression of the f060 fix's mapping. Update the description and header note to add fab-continue's intake-regeneration path, and fix SPEC-_generation.md:5's now-false claim "split across two disjoint consumer groups".

**Verifier**: Confirmed. _generation.md:3 and :11-13 claim a clean partition (fab-new/fab-draft = Intake Generation; fab-continue/fab-ff/fab-fff = Plan Generation), but fab-continue.md:52 explicitly dispatches "read .claude/skills/_generation/SKILL.md first — Intake Generation Procedure" on the intake-active rege…

### `src/kit/skills/_pipeline.md`

#### `a065` [MUST-FIX] Intake gate is undetectable via the documented contract: `fab score --check-gate` exits 0 on gate fail, while the preamble claims it 'returns non-zero'

**Location**: ## Pre-flight, item 3 (line 31); also Shared Error Handling row 'Intake gate fails' · **Category**: correctness · **Found by**: pipeline-srad-helpers

Ground truth: cmd/fab/score.go:25-31 prints FormatGateYAML (including `gate: fail`) and returns nil — exit code is 0 whether the gate passes or fails (score_test.go confirms CheckGate returns err==nil on fail). But _preamble.md § Common fab Commands states '`--check-gate` returns non-zero below the single intake gate (flat 3.0 for all types)' and _cli-fab.md:91 repeats it ('non-zero below the flat 3.0 intake gate'), and _preamble's generic Failure rule keys STOP on non-zero exit — so an agent or script following the documented contract treats every gate check as a pass, silently bypassing the constitution's single confidence gate. Fix the root: make the Go command exit non-zero when `gate: fail` (matching the documented contract; per constitution this needs test updates + _cli-fab.md), or else rewrite _pipeline.md item 3 to 'parse the `gate:` field of the YAML output — the command exits 0 either way' and correct _preamble.md and _cli-fab.md. Note prior finding f131 (Appendix B) restates the wrong non-zero claim as fact — the mismatch was never caught.

**Verifier**: Independently confirmed, including empirically: built fab from src/go/fab and ran `fab score --check-gate --stage intake` on a below-gate fixture — stdout shows `gate: fail` but exit code is 0. cmd/fab/score.go:25-31 prints FormatGateYAML and returns nil unconditionally; TestCheckGate_Fail (internal…

#### `a066` [MUST-FIX] Exhaustion-stop recovery guidance is unexecutable: `/fab-clarify intake` parses 'intake' as a change name, is stage-guard-blocked post-intake, and the promised requirements regeneration never happens

**Location**: ### Stop (after 3 failed cycles), closing paragraph (line 106) · **Category**: correctness · **Found by**: pipeline-srad-helpers

Three defects: (1) fab-clarify.md:28 says 'Any positional argument is treated as a change name' — `intake` is not a valid target form and will be resolved as a change-name substring (error or wrong change). (2) At the exhaustion terminal state (apply: done, review: failed) the change is post-intake, so fab-clarify.md:37's stage guard STOPs: 'Clarification is intake-only… reset with /fab-continue intake first.' (3) The parenthetical regeneration claim is false: even after an intake reset, Step 1's plan generation runs 'unless plan.md already exists' and fab-continue.md:194 says plan.md is preserved — the user MUST delete plan.md to regenerate `## Requirements`. Rewrite to the reachable path: 'Alternatively, reset to intake (/fab-continue intake), refine via /fab-clarify, and delete plan.md to force requirements regeneration, then re-run /{driver}.' Mirror the fix in SPEC-_pipeline.md:26, which endorses 'the /fab-clarify intake alternative'.

**Verifier**: All three defects independently confirmed. (1) fab-clarify.md:28 states verbatim "Any positional argument is treated as a change name"; resolve.go:110-122 does substring matching against fab/changes/ folders, none of which contain "intake" — `/fab-clarify intake` errors `No change matches "intake".`…

#### `a067` [SHOULD-FIX] _pipeline decision heuristics route 'requirements mismatch' to two different rework paths

**Location**: § Auto-Rework Loop → Decision heuristics, lines 87-92 · **Category**: correctness · **Found by**: lens-duplication

The cycle choreography demands the agent 'select exactly one path per the decision heuristics', and the escalation rule's hard 2-consecutive-fix-code counter depends on path identity — yet the most common must-fix class (_review.md's first must-fix bullet is literally 'Requirements mismatches') matches both the Fix-code and Revise-requirements rows. Carried verbatim from the pre-#393 drivers (fab-ff.md:77/79 at a8e720dd~1) into the single-sourced helper; prior review missed it. Disambiguate: Fix code = implementation fails to satisfy a correct requirement (code-side mismatch); Revise requirements = the requirement itself is wrong/drifted vs the intake — and remove the overlapping term from one row. Mirror the wording fix in SPEC-_pipeline.md.

**Verifier**: Confirmed verbatim at _pipeline.md:88/90 — "requirements mismatches" routes to Fix code while "requirements mismatch" routes to Revise requirements, with no qualifier distinguishing them. Collision matters: _review.md:81's first must-fix tier is literally "Requirements mismatches (vs. plan.md ## Req…

### `src/kit/skills/_preamble.md`

#### `a068` [MUST-FIX] Canonical form `fab change resolve --folder` is an invalid command — the flag does not exist on `fab change resolve`

**Location**: ## Common fab Commands, `fab change <sub>` table row (line 232) · **Category**: correctness · **Found by**: preamble

Empirically verified: `go run ./cmd/fab change resolve --folder` fails with `ERROR: unknown flag: --folder` (exit 1) — changeResolveCmd in src/go/fab/cmd/fab/change.go registers no flags; `--folder` belongs only to top-level `fab resolve` (src/go/fab/cmd/fab/resolve.go:73). Since this is the canonical-form column agents copy verbatim, and the preamble's own Failure rule says non-zero exit → STOP, fix the cell to `fab change resolve` (already a passthrough to `fab resolve --folder` per `_cli-fab.md:39`), or better exemplify the row with `fab change new --slug <slug>` and leave folder-output to the `fab resolve` row directly below, which already shows the valid `fab resolve --folder 2>/dev/null`.

**Verifier**: Independently confirmed. /home/sahil/code/sahil87/fab-kit.worktrees/woven-vole/src/kit/skills/_preamble.md:232 canonical-form cell reads `fab change resolve --folder`, and running `go run ./cmd/fab change resolve --folder` fails with `ERROR: unknown flag: --folder` (exit 1). Ground truth: changeReso…

#### `a069` [MUST-FIX] fab score --check-gate documented as exiting non-zero on gate fail, but the Go binary always exits 0 — the sole ff/fff gate can be silently passed

**Location**: § Common fab Commands, `fab score` table row (line 230); same claim in src/kit/skills/_cli-fab.md § fab score (extended), Gate row (line 91) · **Category**: correctness · **Found by**: ff-fff-draft

Ground truth: cmd/fab/score.go:25-32 prints FormatGateYAML and returns nil regardless of verdict, and internal/score/score.go:111-125 encodes the verdict only as `gate: pass|fail` in the YAML — there is no non-zero exit below the gate. This is the single confidence gate /fab-ff and /fab-fff execute via _pipeline.md Pre-flight 3 ('If the gate fails → STOP', detection method unspecified); an agent relying on the documented exit code (reinforced by _preamble's generic 'non-zero → STOP' failure rule) treats a failing gate as a pass and enters the autonomous bracket on an under-resolved intake. Fix one side: either make CheckGate exit non-zero on `gate: fail` (matching the documented contract; constitution requires test + _cli-fab updates), or correct both doc rows and add 'parse the `gate:` field from the YAML output' to _pipeline.md Pre-flight 3. Prior review never verified this (Appendix B f131 even re-quotes the wrong claim).

**Verifier**: Independently confirmed. Docs at src/kit/skills/_preamble.md:230 and src/kit/skills/_cli-fab.md:91 both claim `fab score --check-gate` "returns non-zero below" the 3.0 intake gate. Ground truth: src/go/fab/cmd/fab/score.go:25-32 prints FormatGateYAML and `return nil` unconditionally in check-gate mo…

#### `a070` [MUST-FIX] Canonical-form example `fab change resolve --folder` is a broken command — the subcommand has no --folder flag

**Location**: § Common fab Commands, fab change row (line 232) · **Category**: correctness · **Found by**: lens-cli-contract

changeResolveCmd registers zero flags (src/go/fab/cmd/fab/change.go:147-169), so cobra rejects it — empirically verified: `ERROR: unknown flag: --folder`, exit 1. An agent copying the canonical form hits the failure rule and STOPs. `fab change resolve` already outputs the full folder name (change.Resolve is a direct passthrough to resolve.ToFolder), so the cell should read `fab change resolve` or `fab resolve --folder`. Predates the recent batches (introduced in PR #341) and was missed by the prior review — not in its findings or Appendix B.

**Verifier**: Confirmed on all axes. (1) Quote verbatim at src/kit/skills/_preamble.md:232 (Canonical form column of the `fab change <sub>` row). (2) changeResolveCmd (src/go/fab/cmd/fab/change.go:147-169) registers zero flags; empirically `fab change resolve --folder` fails with `ERROR: unknown flag: --folder`,…

#### `a071` [MUST-FIX] fab score --check-gate never exits non-zero on gate failure — skills document an exit-code contract the Go binary does not implement

**Location**: § Common fab Commands, fab score row (line 230); also _cli-fab.md § fab score (extended), Gate mode row (line 91) · **Category**: correctness · **Found by**: lens-cli-contract

The Go command always returns nil after printing the gate YAML (src/go/fab/cmd/fab/score.go:25-32; internal/score.CheckGate returns a GateResult with Gate:"fail" and nil error). Empirically verified: a 2.4-score change prints `gate: fail` and exits 0. Since _preamble's generic failure rule keys STOP on non-zero exit, /fab-ff//fab-fff/_pipeline Pre-flight step 3 ("If the gate fails → STOP") silently passes a failing gate under exit-code detection — bypassing the pipeline's single human checkpoint. Fix one side: either make the Go command exit non-zero when gate==fail (matches both doc rows and the failure rule), or rewrite _preamble.md:230 and _cli-fab.md:91 to "parse the `gate: pass|fail` field from the YAML output" and document the gate-mode output shape (gate/score/threshold/change_type/certain/confident/tentative/unresolved — currently documented nowhere, even though _pipeline.md:31 and fab-ff.md:42 need `{score}` from it).

**Verifier**: CONFIRMED. (1) Evidence quotes verified verbatim: src/kit/skills/_preamble.md:230 ("--check-gate returns non-zero below the single intake gate (flat 3.0 for all types)") and src/kit/skills/_cli-fab.md:91 ("non-zero below the flat 3.0 intake gate"). (2) Ground truth verified in source AND empirically…

#### `a072` [SHOULD-FIX] Always-Load 'Current exceptions' list omits /fab-proceed, which declares it skips all context loading

**Location**: ## Context Loading > ### 1. Always Load (line 36) · **Category**: staleness · **Found by**: preamble

fab-proceed.md:10 states: "`/fab-proceed` follows `_preamble.md` conventions but skips preflight/context loading itself — it delegates all pipeline context loading to `/fab-fff`" — yet it is absent from the enumeration added in PR #392 (f001 fix). An agent reading only the preamble would load all 7 files in /fab-proceed, contradicting that skill's design. Add /fab-proceed to the exceptions list (and mirror the addition in SPEC-_preamble.md's Flow note, which repeats the same four-skills-plus-operator list at lines 30-33).

**Verifier**: Confirmed. _preamble.md:36 exceptions list (added in PR #392/eca310c4, the f001 fix) omits /fab-proceed, yet fab-proceed.md:10 has declared "skips preflight/context loading itself — delegates all pipeline context loading to /fab-fff" since the skill's creation (PR #278, predating #392). SPEC-fab-pro…

#### `a073` [SHOULD-FIX] _preamble §1 'Current exceptions' enumeration no longer matches the skill corpus

**Location**: ## Context Loading > ### 1. Always Load (line 36) · **Category**: staleness · **Found by**: lens-helper-integrity

The enumeration written by the batch-3 f001 fix is incomplete and partly one-sided: fab-help.md:31 says 'This skill uses **no context**' and fab-proceed.md:10 says it 'skips preflight/context loading itself — it delegates all pipeline context loading to `/fab-fff`' (fab-archive.md:45 likewise declares 'None beyond preflight'), yet none appear in the list; conversely docs-hydrate-memory.md contains no Context Loading statement at all, so its exemption exists only on the preamble side, outside the 'skill's own Context Loading section says otherwise' rule the sentence itself states. Either complete the enumeration (add /fab-help, /fab-proceed; add a one-line Context Loading note to docs-hydrate-memory) or soften it to non-exhaustive examples ('e.g.,').

**Verifier**: Confirmed at src/kit/skills/_preamble.md:36 (quote verbatim, written by batch-3 commit eca310c4 / PR #392 f001 fix). All cited skill-side declarations verified: fab-help.md:31 'This skill uses **no context**', fab-proceed.md:10 'skips preflight/context loading itself', fab-archive.md:45 'None beyond…

#### `a074` [SHOULD-FIX] _preamble Confidence-Scoring invocation list omits /fab-draft and mis-scopes /fab-clarify recompute to suggest mode

**Location**: § Confidence Scoring → Invocation, lines 355-361 · **Category**: spec-drift · **Found by**: lens-duplication

Two divergences from the skill files this always-loaded list summarizes: (1) fab-draft.md:21 executes fab-new's 'Pre-flight, Arguments, and Steps 0–9 exactly as written' — Step 7 is the `fab score --stage intake` call, so /fab-draft also persists the intake score but is absent from the list; (2) fab-clarify.md:172 says 'Both Suggest and Auto modes recompute (Auto Mode step 4)', not suggest-mode-only. Add a /fab-draft bullet (or '(/fab-new Steps 0–9, also via /fab-draft)') and drop '(suggest mode)' or replace with '(both modes)'. Same flavor of enumeration drift as the already-fixed f051 Placed-by list.

**Verifier**: CONFIRMED both sub-claims at the cited location (_preamble.md:355-361, quote verbatim). (1) fab-draft omission: fab-draft.md:21 executes fab-new's Pre-flight/Arguments/Steps 0-9 "exactly as written"; fab-new Step 7 (fab-new.md:91-101) runs `fab score --stage intake <change>` and persists to .status.…

#### `a075` [SHOULD-FIX] _preamble always-load "Current exceptions" list is incomplete — misses 4 skills whose own Context Loading sections deviate

**Location**: § Context Loading › 1. Always Load, line 36 · **Category**: spec-drift · **Found by**: lens-reference-integrity

The enumeration (added by #392's f001 descriptive-contract rewrite) omits four more deviators: /fab-help ("uses no context — it does not load fab/project/config.yaml, fab/project/constitution.md", fab-help.md:31), /docs-hydrate-specs (docs-hydrate-specs.md:35), /docs-reorg-memory (docs-reorg-memory.md:48), and /docs-reorg-specs (docs-reorg-specs.md:27) — each of the latter three states "Does NOT require `.fab-status.yaml`, config, or constitution" and loads only memory/specs files. Either extend the list with all four or mark it non-exhaustive ("e.g.,") so the skill-file-wins contract isn't undercut by a stale enumeration.

**Verifier**: Confirmed. _preamble.md:36 quote matches verbatim, and all four cited skill deviations verify at the exact cited lines: fab-help.md:31 ("uses **no context**"), docs-hydrate-specs.md:35, docs-reorg-memory.md:48, docs-reorg-specs.md:27 (each: "Does NOT require `.fab-status.yaml`, config, or constituti…

#### `a076` [NICE-TO-HAVE] Confidence Scoring 'invoked by' list omits /fab-draft, which scores via fab-new's inherited Steps 0–9

**Location**: ## Confidence Scoring > ### Invocation (lines 357-361) · **Category**: correctness · **Found by**: preamble

fab-draft.md:21 executes fab-new's "Pre-flight, Arguments, and Steps 0–9 exactly as written", which includes fab-new.md:95 "Call `fab score --stage intake <change>`" — so drafted (unactivated) changes do persist an intake score, but an agent reading the preamble's two-item list would conclude they are unscored. Add a third bullet: "`/fab-draft` (via fab-new's Steps 0–9)". Distinct from known-open f020 (scoring timing) and f121 (unscored regenerated intakes).

**Verifier**: Confirmed. _preamble.md:357-359 lists only /fab-new and /fab-clarify as fab score invokers, and line 361 ("no scoring at any post-intake stage") makes the list read as exhaustive. But fab-draft.md:21 executes fab-new's Steps 0-9 "exactly as written" with deltas touching only Step 9's tail and Steps…

#### `a077` [NICE-TO-HAVE] Subagent Dispatch enumerates orchestrators as exactly /fab-ff and /fab-fff, but /fab-proceed dispatches prefix steps per this section

**Location**: ## Subagent Dispatch (Orchestrator Skills), opening paragraph (line 301) · **Category**: staleness · **Found by**: preamble

fab-proceed.md:130 mandates its prefix steps (fab-new, fab-switch, git-branch) "be dispatched as a subagent using the Agent tool (`subagent_type: \"general-purpose\"`) per `_preamble.md` § Subagent Dispatch", and fab-proceed's own description calls it a "Context-aware orchestrator". Reword the parenthetical to include /fab-proceed (prefix steps) or soften to an e.g.-list, so the section's consumer set stays accurate. Distinct from Appendix-B f153, which concerns the terminal Skill-tool delegation exception, not this enumeration.

**Verifier**: Confirmed. _preamble.md:301 enumerates orchestrators as exactly (`/fab-ff`, `/fab-fff`) with no e.g.-softener, while fab-proceed.md:130 mandates prefix-step dispatch "per `_preamble.md` § Subagent Dispatch" and fab-proceed.md:3 self-describes as a "Context-aware orchestrator" — so the section has a…

#### `a078` [NICE-TO-HAVE] _preamble always-load descriptor for config.yaml still advertises removed 'naming conventions' and dead 'model tiers'

**Location**: §1 Always Load file list, line 40 · **Category**: staleness · **Found by**: lens-templates-config

The `naming` section was removed from config.yaml by the 0.10.0-to-0.20.0 migration (naming now lives in _preamble's own Naming Conventions section), and `model_tiers` is absent from the scaffold and consumed by nothing in skills or Go. Every skill reads this line each invocation and is told to expect config content that doesn't exist. Reword to match the actual key set, e.g. "— project identity, source/test paths, review tooling, agent spawn command".

**Verifier**: CONFIRMED. Evidence quote appears verbatim at src/kit/skills/_preamble.md:40. Both staleness claims independently verified: (1) `naming:` was removed from config.yaml by src/kit/migrations/0.10.0-to-0.20.0.md ("remove `git` and `naming` sections"; success criterion: `yq '.naming'` returns null), and…

### `src/kit/skills/_review.md`

#### `a079` [SHOULD-FIX] Inward sub-agent's Step 7/8 skip condition depends on change_type, which no defined input provides

**Location**: § Inward Sub-Agent Dispatch — Validation Steps, step 7 (line 53) vs. context list (line 35) · **Category**: ergonomics · **Found by**: gen-review-helpers

The inward sub-agent evaluates the skip but its declared context (Standard Subagent Context + plan.md + touched sources + memory files) never includes change_type, and `fab preflight` does not emit it (fab-new.md:88 states this explicitly: "`fab preflight` does not emit this field"). Add change_type to the dispatch context contract: have the orchestrator read it via `grep '^change_type:' {change_dir}/.status.yaml` and pass it in the sub-agent prompt (or instruct the sub-agent to read it the same way). Mirror in SPEC-_review.md's step-7/8 "Skipped when" rows.

**Verifier**: Confirmed at all cited locations. _review.md:53 (step 7) and :64/:75 (step 8 + Deletion Candidates omission rule) gate on change_type, but the inward dispatch contract (_review.md:35) provides only Standard Subagent Context (_preamble.md:314-327: 5 project files) + plan.md + touched sources + memory…

#### `a080` [SHOULD-FIX] ## Deletion Candidates replace rule is scoped to rework cycles only — a plain review re-run reads as 'append', duplicating the section

**Location**: § Inward Sub-Agent Dispatch — step 8, lines 66 and 75 · **Category**: idempotency · **Found by**: gen-review-helpers

Re-runs that are not rework cycles (review re-entry after interruption via Resumability, or `fab status reset <change> review`) will find an existing section, and the literal text ("On rework cycles, an existing `## Deletion Candidates` section SHALL be replaced in place") only mandates replacement for rework. Make the rule state-based, not history-based: "If a `## Deletion Candidates` section already exists, replace it in place; otherwise insert it below `## Notes`." This satisfies constitution principle III for all re-run paths. Update the matching wording at SPEC-_review.md:97 ("Output appended (or replaced on rework)").

**Verifier**: Confirmed. _review.md:66 says "Append (or replace, on rework)" and :75 scopes the SHALL-replace rule to "rework cycles" only — the rule is history-based. Two documented non-rework re-run paths re-execute step 8 with the section already present: (1) fab-continue.md:194 Reset Flow explicitly preserves…

### `src/kit/skills/_srad.md`

#### `a081` [SHOULD-FIX] Grade thresholds are closed integer bands but the composite is continuous — values like 59.85 or 84.5 match no grade

**Location**: ## SRAD Scoring, Aggregation (line 30) · **Category**: ergonomics · **Found by**: pipeline-srad-helpers

The same paragraph mandates a 'continuous 0–100 scale' and a weighted mean (0.25/0.30/0.25/0.20), which is generally fractional — e.g., S:70 R:62 A:55 D:50 → 59.85, falling in the undefined gap between Tentative (≤59) and Confident (≥60); same for (29,30) and (84,85). Agents will round inconsistently, and the Tentative/Confident boundary alone is worth 0.7 score per row (wTentative=1.0 vs wConfident=0.3 in score.go), enough to flip the 3.0 gate. Replace with half-open thresholds: 'Certain ≥ 85, Confident ≥ 60, Tentative ≥ 30, else Unresolved' (or mandate rounding the composite to the nearest integer before mapping).

**Verifier**: Confirmed. Quote verbatim at src/kit/skills/_srad.md:30; line 21 mandates a continuous 0-100 scale and the weighted mean (0.25/0.30/0.25/0.20) is generally fractional — verified S:70 R:62 A:55 D:50 -> 59.85, undefined between Tentative (<=59) and Confident (>=60); same gaps at (29,30) and (84,85). N…

#### `a082` [SHOULD-FIX] _srad Worked Example 3's grade is mathematically unreachable: S:Low + R/A/D:High caps the composite at 84.75, below the 85 Certain threshold

**Location**: ## Worked Examples, Example 3 (line 71) vs Aggregation (line 30) · **Category**: correctness · **Found by**: pipeline-srad-helpers

With S in the Low band (≤39) and R/A/D maxed at 100, the weighted mean 0.25*S + 0.30*R + 0.25*A + 0.20*D peaks at 0.25*39 + 75 = 84.75 — strictly below Certain (85–100). The formula grades this example Confident while the text says Certain, and the difference is score-bearing: score.go uses wCertain=0.0 vs wConfident=0.3, so the disagreement shifts the intake score and can flip the 3.0 gate. Either re-dimension the example (a config-determined answer means the question is fully disambiguated — score S High since signal sufficiency comes from config, not prose length) or add an explicit qualitative override next to the Critical Rule ('decisions deterministically answered by config/constitution are Certain regardless of composite'), so the grade table's 'Determined by config' definition and the formula agree.

**Verifier**: Confirmed at src/kit/skills/_srad.md:71 vs :30. Math checks out: S Low band is 0-39 (line 23 table), so max composite = 0.25*39 + 0.30*100 + 0.25*100 + 0.20*100 = 84.75 < 85 (Certain threshold). The only documented override (line 30; docs/specs/srad.md:50) is the Critical Rule, which only forces gra…

#### `a083` [NICE-TO-HAVE] Critical Rule's 'low Reversibility AND low Agent Competence' has two competing numeric definitions (<25 override vs the 0–39 Low band)

**Location**: ## Critical Rule (line 47) vs Aggregation (line 30) and the dimension table (line 24) · **Category**: ergonomics · **Found by**: pipeline-srad-helpers

Line 30 defines the override numerically as 'R < 25 AND A < 25 → always Unresolved', but the scoring table defines 'Low' as 0–39, and Worked Example 1 just says '(Critical Rule applies: low R + low A)'. An Unresolved row with R:30/A:30 is table-Low but misses the override — one agent treats it as must-ask, another permits 'Deferred — {reason}' (allowed by the Rules bullet at line 113). Bind the Critical Rule to the explicit override ('decisions tripping the R < 25 AND A < 25 override MUST be asked, never deferred') or redefine the override to the 0–39 Low band — one number, used in both places. This is distinct from known-open f023 (skill scope) and f037 (question budget), which concern other clauses of the same paragraph.

**Verifier**: Confirmed. _srad.md:47 quote verbatim; line 30 names the numeric threshold "Critical Rule override: R < 25 AND A < 25 -> always Unresolved" while the dimension table (lines 23-28) defines Low as 0-39, and the Critical Rule section plus Worked Example 1 (line 63) use unanchored "low". Divergence is r…

#### `a084` [NICE-TO-HAVE] Skill-Specific Autonomy Levels table covers only 4 of the 6 skills that declare _srad — fab-draft and fab-clarify have no column

**Location**: ## Skill-Specific Autonomy Levels (lines 51–57) · **Category**: convention · **Found by**: pipeline-srad-helpers

The helper's own header (lines 11–13) says it is declared by six planning skills, and all six do declare `helpers: [_srad]` — but an agent executing /fab-draft or /fab-clarify loads this table and finds no posture, interruption budget, or escape-valve row for its own skill. Cheapest fix is a one-line note under the table: 'fab-draft follows the fab-new column (thin delta over /fab-new — no activation, no branch); fab-clarify's interaction model is defined in its own file (it is the escape valve, not a consumer of one).' Adding two full columns also works but duplicates content batch 4 deliberately single-sourced into fab-new.md/fab-clarify.md.

**Verifier**: Confirmed at src/kit/skills/_srad.md:51-57: table columns are fab-new/fab-continue/fab-fff/fab-ff only, while the header (lines 11-13) names six consumers and all six declare helpers:[_srad] (verified by grep across src/kit/skills/*.md). Severity nice-to-have is correct, arguably the ceiling: behavi…

### `src/kit/skills/docs-hydrate-memory.md`

#### `a085` [SHOULD-FIX] Generate mode Step 3 has no placement rules — no target-path mapping, no domain-index/description stub, no shape bounds

**Location**: ## Generate Mode Behavior → Step 3: Memory File Generation, lines 124-149 · **Category**: ergonomics · **Found by**: docs-hydrate

The target path (`docs/memory/{domain}/{topic}.md`), domain-folder creation, the domain-index stub carrying `description:` frontmatter, and the shape bounds are all defined only under 'Ingest Mode Behavior' Step 3; generate mode cross-references ingest only for Step 4 ('Same as ingest mode Step 4'). An agent in generate mode must improvise where files go, and a new domain it creates gets a '—' description in the root index because nothing instructs authoring the domain index stub (memoryindex.go domainDescription reads {domain}/index.md frontmatter, '' → missing-cell). The SPEC flow already assumes the ingest path scheme ('Write: docs/memory/{domain}/{file}.md'). Add to generate Step 3: 'Placement, domain creation, domain-index `description:` stub, and shape bounds: same as Ingest Step 3 (items 1-2 + Shape bounds).'

**Verifier**: CONFIRMED. Evidence quote verbatim at src/kit/skills/docs-hydrate-memory.md:126. Generate Step 3 (lines 124-149) has only the file-content template; target path (line 63), domain creation + domain-index description stub (lines 68-69), and shape bounds (lines 75-79) exist only under 'Ingest Mode Beha…

#### `a086` [NICE-TO-HAVE] Sub-domain index description: stub is never instructed, and Step 4 omits sub-domain indexes from what fab memory-index regenerates

**Location**: ## Ingest Mode Behavior → Step 3 item 2 (line 69) and Step 4 (lines 81-83) · **Category**: correctness · **Found by**: docs-hydrate

The skill's own shape guidance has it introduce sub-domains (≥8-file cluster), but the description-stub instruction covers only `{domain}/index.md` — so a new sub-domain renders '—' in the parent's Sub-Domains table (Go round-trips it from `{domain}/{sub}/index.md` frontmatter: memoryindex.go gatherSubDomains → domainDescription). Step 4 also says the command regenerates 'the root (domains-only) and every domain index', while the command additionally generates every `{domain}/{sub-domain}/index.md` (memory_index.go Long text). Extend item 2 to sub-domain index stubs and Step 4's wording to 'root + every domain and sub-domain index'.

**Verifier**: Confirmed both sub-claims. (1) docs-hydrate-memory.md:69 instructs a description: stub only for {domain}/index.md, while its own Shape bounds (lines 76-78) license introducing {domain}/{sub-domain}/ during placement; Go ground truth (memoryindex.go gatherSubDomains:311 -> domainDescription:351, Rend…

#### `a087` [NICE-TO-HAVE] fab memory-index tier description diverges: hydrate-side copies omit the sub-domain index tier

**Location**: Ingest Mode Step 4 (line 83) and Generate Mode Step 4 (line 153); same omission at fab-continue.md Hydrate step 4 (line 183) · **Category**: spec-drift · **Found by**: lens-duplication

_cli-fab.md § fab memory-index confirms the command regenerates 'every docs/memory/{domain}/{sub-domain}/index.md' as a third tier, and _preamble § Memory File Lookup makes sub-domained Affected Memory entries a normal hydrate target — so an agent hydrating into a sub-domain could conclude the sub-domain index is not command-owned and hand-edit it. Add 'and sub-domain' to the two under-describing copies (docs-hydrate-memory both modes, fab-continue Hydrate step 4). Distinct from refuted f104 (shape-bounds numbers): this is a wrong description of command behavior, not bound duplication.

**Verifier**: Confirmed at all three cited locations. docs-hydrate-memory.md:83 ("regenerates the root (domains-only) and every domain index") and :153, plus fab-continue.md:183, omit the sub-domain index tier; docs-reorg-memory.md:126 correctly lists "root, every domain index.md, AND every sub-domain index.md".…

#### `a088` [STRUCTURAL] Generate mode defines its own scan scope and ignores config source_paths, unlike internal-consistency-check

**Location**: ## Generate Mode Behavior → Step 1: Codebase Scanning, line 91 · **Category**: architecture · **Found by**: docs-hydrate

The kit already has a canonical 'where the implementation lives' key — internal-consistency-check.md:18 'Read fab/project/config.yaml, extract source_paths' — but generate mode scans the project root with a hard-coded exclusion list, so in this very repo it would flag fab/, docs/, and .claude/ as undocumented 'modules'. Default the no-arg scan scope to config's `source_paths` when defined (fall back to project root when absent), mirroring internal-consistency-check. Note this requires the skill to load config.yaml, which folds naturally into adding the missing Context Loading section (see the exemption-contradiction finding).

**Verifier**: Confirmed: docs-hydrate-memory.md:91 generate mode scans project root with a hard-coded ecosystem-specific exclusion list, while internal-consistency-check.md:18 establishes source_paths (fab/project/config.yaml) as the canonical implementation-scope key; memory (_shared/configuration.md:181) and th…

### `src/kit/skills/docs-hydrate-specs.md`

#### `a089` [SHOULD-FIX] docs-hydrate-specs has no branch for a gap with no suitable existing target spec file

**Location**: ## Behavior → Step 5: Present Gaps with Previews, line 64 · **Category**: ergonomics · **Found by**: docs-hydrate

Step 5 hard-codes that every gap has an existing `{spec_file}` target, but a memory domain covering a whole area absent from specs (the strongest possible gap) has no natural target — the agent must either silently drop the gap or shoehorn it into an unrelated spec. The SPEC mirror's '(if new files added)' hints creation was intended. Define the branch explicitly: either 'no suitable target → propose creating a new `docs/specs/{name}.md` plus an index row (specs index is hand-curated), with the same per-gap confirmation', or 'gaps with no plausible target file are reported in the summary as needs-new-spec, not auto-inserted'.

**Verifier**: Confirmed. src/kit/skills/docs-hydrate-specs.md Step 5 (line 65: "**Target**: `{spec_file}` -> after {section}") and Step 6 ("yes -> insert at specified location") assume every gap targets an existing spec file; no Behavior step, Error Handling row, or Key Property covers the no-suitable-target case…

#### `a090` [NICE-TO-HAVE] Step 6 handles a 'skip rest' token that Step 5's prompt never offers

**Location**: ## Behavior → Step 5 prompt (line 70) vs Step 6: Interactive Confirmation (line 76) · **Category**: ergonomics · **Found by**: docs-hydrate

The per-gap prompt offers exactly '(yes / no / done)' but the handler defines four tokens. An agent rendering the prompt literally never surfaces 'skip rest', and one parsing replies has two names for one action. Align them: either add 'skip rest' to the Step 5 prompt or drop it from Step 6 (keep 'done' only).

**Verifier**: Confirmed verbatim in src/kit/skills/docs-hydrate-specs.md: Step 5 prompt (line 70) offers '(yes / no / done)' while Step 6 (line 77) handles '**done** / **skip rest**' — 'skip rest' is never surfaced to the user and duplicates 'done'. Not intentional: the SPEC mirror (docs/specs/skills/SPEC-docs-hy…

### `src/kit/skills/docs-reorg-memory.md`

#### `a091` [MUST-FIX] Step 5.3 tells the agent to add description: frontmatter to a sub-domain index.md that does not exist until Step 5.4 generates it — and Step 5.4 forbids the edit

**Location**: ## Behavior § Step 5: User Confirmation & Apply, items 3-4 (lines 125-126); also Key Properties line 176 · **Category**: correctness · **Found by**: docs-reorg

At Step 5.3 time the sub-domain has no index.md (fab memory-index creates it in Step 5.4), and Step 5.4 says 'Do **not** hand-edit index files — they are generated' while Key Properties says 'Indexes hand-edited? | No' — so an agent either skips the step (sub-domain row renders '—' forever) or violates the rule. Fix: instruct creating a stub `{domain}/{sub-domain}/index.md` containing only `---\ndescription: "<one-liner>"\n---` BEFORE running `fab memory-index` (Go round-trips it: memoryindex.go domainDescription reads it, RenderDomain re-emits it — and docs-hydrate-memory.md:69 already uses exactly this stub pattern for new domains), and carve out the `description:` frontmatter as the single hand-curated field in the never-hand-edit rule (Step 5.4, Key Properties, and SPEC-docs-reorg-memory.md).

**Verifier**: Confirmed. Evidence verbatim at docs-reorg-memory.md:125; prohibitions at :126 and :176. Ground truth: nothing in Steps 5.1-5.3 creates a sub-domain index.md — only fab memory-index does (cmd/fab/memory_index.go:62-67), so 5.3's "new sub-domain index.md a split creates" targets a nonexistent file an…

#### `a092` [SHOULD-FIX] Depth off-by-one: Ideal Shape Bound counts the topic file as a path segment (≤3) while the Shape Report's Depth column counts folder depth (domain=1, sub=2)

**Location**: § Ideal Shape Bounds (line 23), Step 1 (line 56), Step 3 Shape Report example (lines 78-84) · **Category**: ergonomics · **Found by**: docs-reorg

The bound's exemplar path has 3 segments counting the .md file, but Step 1 records folder 'depth relative to docs/memory/' and the example rows show Depth 1 for a domain and 2 for a sub-domain. An agent comparing folder depth against '≤3' will pass a {domain}/{sub}/{subsub} folder (folder depth 3, files at segment depth 4) that the Go binary warns on (memoryindex.go: MaxDepth=3 compared against file path-segment count, so the warning fires at folder depth ≥3). Define the unit once: 'Depth = folder depth relative to docs/memory/ (domain=1, sub-domain=2); ⚠ over depth when folder depth ≥3', or switch the report to file-segment counting to match the bound. Mirror the clarification in SPEC-docs-reorg-memory.md ('over depth 3').

**Verifier**: Confirmed off-by-one unit mismatch in src/kit/skills/docs-reorg-memory.md. Line 23's bound "Tree depth ≤3 ({domain}/{sub-domain}/{topic}.md)" counts path segments including the .md file, while Step 1 (line 56) records folder depth relative to docs/memory/ and the Step 3 Shape Report example (lines 7…

#### `a093` [NICE-TO-HAVE] Dangling-link hard block has no abort/rollback escape — apply can loop indefinitely when a rewrite target cannot be determined

**Location**: § Step 5 item 5 (line 127) and Error Handling last row (line 161) · **Category**: idempotency · **Found by**: docs-reorg

The write/move-failure row has a terminal remedy ('Report error, roll back that migration, continue') but the dangling-link row only says keep fixing — if a link's correct target is ambiguous or the target was never part of the migration (Link Impact list incomplete), the agent has no defined exit. Add the escape: 'if a rewrite target cannot be determined, roll back that migration (as in the write-failure row), report the link, and continue with the remaining migrations' — so apply always terminates and the partial-failure state matches the documented rollback semantics.

**Verifier**: CONFIRMED. Evidence verified verbatim: src/kit/skills/docs-reorg-memory.md:127 ("**A remaining dangling relative link is a hard block** — do NOT finalize that migration until every broken link is rewritten.") and the Error Handling row at :161 ("**Hard block** — report the dangling link; do not fina…

### `src/kit/skills/docs-reorg-specs.md`

#### `a094` [SHOULD-FIX] docs-reorg-specs has no reserved-path exemption for the path-pinned SPEC mirrors (and is ambiguous about subfolder recursion)

**Location**: ## Purpose (line 12), ## Pre-flight (lines 18-19), § Step 1 (line 35) · **Category**: correctness · **Found by**: docs-reorg

docs/specs/ is no longer flat (the prior review's f107 analysis assumed 'specs are flat'): docs/specs/skills/ holds 30 SPEC-*.md files whose names/paths are a constitution-mandated 1:1 contract ('Changes to skill files MUST update the corresponding docs/specs/skills/SPEC-*.md file'), plus docs/specs/findings/. Nothing stops a reorg proposal from merging/renaming SPEC mirrors, which would break that contract and the New Skill Checklist tooling; it is also undefined whether Step 1 recurses into these subfolders at all (contrast docs-reorg-memory's explicit 'recursing into sub-domain folders'). Add a 'Reserved paths (exempt)' note mirroring docs-reorg-memory's Reserved Domains section — exclude subfolders whose file paths are externally pinned contracts (per-skill SPEC mirrors, findings archives) from migration proposals — and state explicitly whether Step 1 recurses.

**Verifier**: Confirmed. Evidence quote exact at src/kit/skills/docs-reorg-specs.md:35; no reserved-path or recursion language anywhere in the skill. Ground truth: docs/specs/ is non-flat (docs/specs/skills/ = 30 SPEC-*.md mirrors, docs/specs/findings/ = review archive); constitution.md:32 pins the 1:1 path contr…

### `src/kit/skills/fab-archive.md`

#### `a095` [SHOULD-FIX] fab-archive has no path for the archive-succeeded-but-backlog-mark-failed exit; re-run can never mark the backlog

**Location**: ## Behavior Step 1 (lines 51-67) + Key Properties idempotency row (line 110) · **Category**: idempotency · **Found by**: change-mgmt

Go's ArchiveWithBacklog (cmd/fab/archive.go:36-39) deliberately prints the success YAML AND exits non-zero when the move succeeded but the backlog mark genuinely failed ("archive succeeded but marking backlog item ... failed"). The skill has no archive-mode Error Handling at all, so the preamble's generic rule applies: STOP and surface stderr, "resumability handles the re-run" — but re-run hits the exit-0 `already archived:` soft skip (or preflight failure per known f085), so the backlog item is permanently left unmarked and the emitted YAML report is discarded. Add an archive-mode error row: on non-zero exit with YAML on stdout, still render the Step-2 report, surface the backlog error, and instruct manual backlog reconciliation (or re-run `fab change archive` is NOT sufficient). Also extend _cli-fab.md's `archive` exception note, which currently documents only the soft-skip non-YAML case.

**Verifier**: CONFIRMED at should-fix. Every link in the chain verified independently:

1. Evidence quote exact at src/kit/skills/fab-archive.md:110; Behavior Step 1 at lines 51-67 as cited. Archive mode has no Error Handling section — the only one in the file is under Restore Mode (lines 169-178).

2. The partia…

#### `a096` [SHOULD-FIX] fab-archive: archive/restore move tracked files and edit fab/backlog.md with no commit step, no git-state disclosure, and a "safe" claim the dirty tree contradicts

**Location**: Purpose (line 14); Step 1 (lines 51-67); Key Properties (lines 105-116 and 180-189) · **Category**: idempotency · **Found by**: git-state-safety-sweep

Add a status-commit step mirroring git-pr Step 4c (commit + push the move, index, and backlog edit), or at minimum document that both modes leave the working tree dirty, add "Modifies git state?" rows to both Key Properties tables, and warn about archiving while the change's branch/PR is still unmerged. Impact: the archive is illusory from the repo's perspective until someone else happens to commit it — usually into an unrelated change's PR.

**Verifier**: Confirmed end-to-end. (1) archive.go is pure os.Rename + in-place backlog edit, zero git calls (verified lines 86-93, 128-140; restore at 156-165 likewise). (2) fab/changes/* and fab/backlog.md are git-tracked (git ls-files), so both modes leave a mass tracked-file move + tracked-file edit uncommitt…

#### `a097` [NICE-TO-HAVE] Restore report maps pointer: skipped to '— not requested' even when --switch was requested and silently failed

**Location**: ## Restore Mode › Step 2: Format Report table, lines 144-153 · **Category**: correctness · **Found by**: change-mgmt

Go's Restore (internal/archive/archive.go:171-178) defaults pointerStatus to "skipped" and, when `--switch` is passed but change.Switch errors, swallows the error and still emits `pointer: skipped` — the skill then reports "— not requested", which is false. Either have the CLI emit a distinct `pointer: failed` (preferred; propagate or at least surface the switch error) or soften the report line to `Pointer:  — not switched` and note the failure case.

**Verifier**: Confirmed. Evidence verbatim at src/kit/skills/fab-archive.md:153 (and Output template :162). Go ground truth matches exactly: src/go/fab/internal/archive/archive.go:171-178 defaults pointerStatus to "skipped" and discards change.Switch's error (`if err == nil` only), returning nil error; FormatRest…

#### `a098` [NICE-TO-HAVE] fab change restore --switch swallows activation failure — exits 0 with `pointer: skipped`, which fab-archive renders as 'not requested'

**Location**: ### Step 2: Format Report (Restore Mode), line 153; Error Handling table lines 171-178 has no failed-switch row · **Category**: correctness · **Found by**: exit-code-contract-vs-go-sweep

Add a third pointer value (e.g., `pointer: failed`) or propagate the Switch error from Restore; alternatively amend fab-archive.md:153 so `pointer: skipped` reads 'not requested or activation failed' and add an Error Handling row. Impact: a user who asked for activation gets exit 0 and a report claiming activation was never requested — a silent failure misreported as intentional.

**Verifier**: Confirmed end to end. (1) fab-archive.md:153 maps `pointer: skipped` to 'Pointer: — not requested' verbatim; Restore Error Handling table (lines 169-178) has no failed-switch row. (2) Go evidence verified at src/go/fab/internal/archive/archive.go:171-178 — change.Switch error is discarded, pointerSt…

### `src/kit/skills/fab-clarify.md`

#### `a099` [MUST-FIX] Step 1.5 zero-gaps early exit makes bulk confirm unreachable in its primary scenario (Confident-only intake)

**Location**: Suggest Mode § Step 1.5 (line 55) vs § Step 2 (lines 61-73) · **Category**: correctness · **Found by**: fab-clarify

Per _srad.md, Confident assumptions carry no artifact marker (Marker column: "None") and are not content gaps, so an intake with e.g. 5 Confident rows and no [NEEDS CLARIFICATION]/<!-- assumed: --> markers hits the Step 1.5 early exit and stops — yet Step 2's trigger (confident >= 3 AND confident > tentative + unresolved) is trivially satisfied there, and Step 2's own display header calls these items the "primary confidence drag". fab-new's "Run /fab-clarify to review" routes exactly this case here, and the user gets a dead-end "artifact looks solid" while the score stays below the 3.0 gate. Fix: evaluate the Step 2 trigger before the early exit — change the exit condition to "If zero gaps AND the Step 2 bulk-confirm trigger is not met: stop" (or move the trigger check into Step 1.5). Update SPEC-fab-clarify.md flow accordingly.

**Verifier**: CONFIRMED as a genuine regression, not intentional design. (1) Quote verified at src/kit/skills/fab-clarify.md:55; the unconditional "If zero gaps: ... stop" precedes Step 2's bulk-confirm trigger (lines 61-66). (2) Per _srad.md grade table, Confident rows have no artifact marker, and Step 1.5 scans…

#### `a100` [NICE-TO-HAVE] "update last_updated" / Step 8 imply a manual .status.yaml edit that fab score already performs

**Location**: Auto Mode item 4 (line 214) and § Step 8 (line 176) · **Category**: ergonomics · **Found by**: fab-clarify

Ground truth: score.Compute → status.SetConfidence → StatusFile.Save, and Save() sets `sf.LastUpdated = nowISO()` (src/go/fab/internal/statusfile/statusfile.go:288), so the single `fab score` call writes both confidence and last_updated atomically. As written, Auto Mode item 4 and Step 8 ("Only update `confidence` and `last_updated` in `.status.yaml`") read as separate agent actions with no specified mechanism, inviting a redundant raw yq write that bypasses the atomic temp+rename save. Reword both to descriptive voice, e.g. "the `fab score` call persists `confidence` and refreshes `last_updated` — make no other `.status.yaml` edits".

**Verifier**: CONFIRMED at stated nice-to-have severity. Evidence verified: src/kit/skills/fab-clarify.md:214 (Auto Mode item 4) reads "recompute the intake score (`fab score --stage intake <change>`) and update `last_updated`" — two conjoined imperative actions — and Step 8 (lines 174-176) says "Only update `con…

#### `a101` [NICE-TO-HAVE] S→95 upgrade labels rows Certain whose composite score stays below _srad's 85 Certain threshold

**Location**: Step 2 § Artifact Update (lines 110-118) and Step 4 item 2 · **Category**: correctness · **Found by**: fab-clarify

_srad.md:30 maps composite = 0.25*S + 0.30*R + 0.25*A + 0.20*D to grades with Certain = 85-100. A confirmed Confident row (e.g. S:40 R:70 A:70 D:60, composite 60.5) becomes Grade=Certain with composite 74.25 — the Grade and Scores columns of the same row now contradict the declared helper's mapping (fab score counts the Grade column, so the gate is fine, but the persisted dimension means and the table become internally inconsistent). Add one sentence to Artifact Update: "User confirmation supersedes the threshold mapping — the Grade column is authoritative for clarified rows" (or bump D as well, since confirmation is direct disambiguation). Keeps f150's Appendix-B lead intact since it builds on this same table.

**Verifier**: Confirmed. fab-clarify.md:110-118 upgrades confirmed rows to Grade=Certain bumping only S→95; _srad.md:30 maps composite=0.25S+0.30R+0.25A+0.20D with Certain=85-100. Example math verified (S:40 R:70 A:70 D:60 → 60.5 pre, 74.25 post — still Confident band). Structurally S→95 adds at most 23.75 points…

#### `a102` [NICE-TO-HAVE] Step 5 audit trail lacks the placement rule Step 2 has, and same-day bulk-confirm re-runs duplicate session headings

**Location**: Step 2 § Audit Trail (line 122) vs § Step 5 (line 153) · **Category**: idempotency · **Found by**: fab-clarify

Step 2's audit trail specifies placement ("create before `## Assumptions` if it doesn't exist") but Step 5 does not — a Q&A-only session that creates `## Clarifications` by appending at end of file violates _srad.md's rule that `## Assumptions` is "a trailing ... section" (the last section, which clarify's marker scan and fab score rely on finding). Conversely, Step 2's session block has no append-to-existing rule, so a partially-confirmed bulk run repeated the same day creates a second identical `### Session {YYYY-MM-DD} (bulk confirm)` heading. Mirror the two rules: add "create before `## Assumptions`" to Step 5, and "append rows to an existing same-day (bulk confirm) session" to Step 2.

**Verifier**: Confirmed at nice-to-have severity. Evidence quote matches fab-clarify.md:153 exactly; the asymmetry is real: Step 2 (line 122) has the placement rule "create before ## Assumptions if it doesn't exist" that Step 5 lacks, and Step 2's audit block always emits a fresh "### Session {YYYY-MM-DD} (bulk c…

#### `a103` [NICE-TO-HAVE] fab-clarify Skill Invocation Protocol opens with an example its own section disclaims as removed

**Location**: § Skill Invocation Protocol, line 182 vs § Currently Applicable, lines 195-199 · **Category**: staleness · **Found by**: lens-duplication

Residue of the #392 f041 MOVE: the protocol intro moved here verbatim and kept the pre-1.10.0 ff→clarify flow as its canonical example, which the section's own 'Currently Applicable' paragraph then contradicts. Since this section is now the sole authority _preamble points to, reword the example as explicitly hypothetical (e.g., 'e.g., a future orchestrator invoking /fab-clarify') or drop the parenthetical. Not covered by f064 (Auto Mode retention / intake-only repetition).

**Verifier**: Confirmed: src/kit/skills/fab-clarify.md:182 uses "/fab-ff invoking /fab-clarify between stages" as the protocol's canonical example while lines 195-199 of the same section state that exact auto-invocation was removed in 1.10.0; fab-ff.md:15 and fab-fff.md:15 independently confirm no in-bracket clar…

### `src/kit/skills/fab-continue.md`

#### `a104` [SHOULD-FIX] intake.md-missing error points to plain /fab-continue, which loops back to the same error

**Location**: ## Error Handling, line 204 · **Category**: ergonomics · **Found by**: fab-continue

At apply entry, intake progress is necessarily `done` (apply only activates after finish intake), so plain /fab-continue re-dispatches apply and hits the identical error — an infinite pointer loop. The intake-regeneration path ('generate intake if missing', line 52) requires intake state `active`, reachable only via reset. Change the message to: 'No intake.md found. Run /fab-continue intake to reset and regenerate the intake first.' (the Reset Flow's intake reset regenerates the artifact and stops at ready).

**Verifier**: Confirmed. Evidence quote appears verbatim at src/kit/skills/fab-continue.md:204. The loop is real: _preamble.md derives stage from the .status.yaml progress map (the stage with active/ready state), not artifact presence, and the apply-entry error only fires when derived stage is apply — which requi…

#### `a105` [SHOULD-FIX] No dispatch row for review-pr in failed state — /fab-continue is undefined after a failed PR review

**Location**: ## Normal Flow > Step 1 dispatch table, line 58 · **Category**: correctness · **Found by**: fab-continue

When git-pr-review's Step 6 fail path (or this row's own fallback `fail <change> review-pr`) leaves review-pr `failed`, a subsequent /fab-continue matches no row: CurrentStage's all-done fallback returns `review-pr` (status.go:460), progress.review-pr is `failed` (not `active`/`ready`), and the 'all `done`' row doesn't apply. Batch 4's f019 added a failed row only for `review`. Add a `review-pr`/`failed` row: 'Re-execute `/git-pr-review` behavior — its Step 0 best-effort `fab status start <change> review-pr` already handles failed→active (git-pr-review.md:22-25); same only-if-still-active guards as the active row.' This also matches _preamble's State Table row 'review-pr (fail) | /git-pr-review'.

**Verifier**: Independently confirmed end-to-end. (1) src/kit/skills/fab-continue.md dispatch table (lines 49-59) has a review/failed row (line 55, f019) but no review-pr/failed row; line 58 covers only active/ready. (2) The state is reachable: git-pr-review.md Step 6 item 2 runs `fab status fail <change> review-…

#### `a106` [SHOULD-FIX] Reset Flow errors when the target stage is already active — non-idempotent re-run

**Location**: ## Reset Flow (with stage argument), step 3, line 193 · **Category**: idempotency · **Found by**: fab-continue

Go's reset transition is From {done, ready, skipped} (status.go:41), so `/fab-continue apply` while apply is already `active` (interrupted mid-apply, or right after a review-fail reset) exits non-zero and the preamble failure rule STOPs with a raw CLI error — even though the intent (re-run apply) is well-formed, and the skill's own legacy-target error ('use /fab-continue apply (regenerates plan.md and re-runs)') routes users into exactly this call. This violates constitution Principle III: re-running the same invocation succeeds once then fails. Add to step 3: 'If the target stage is already `active`, skip the reset (already at target) and proceed to step 4.' Also note the Normal Flow Step 4 signature line 'done/ready → active' omits `skipped` from the Go From-set.

**Verifier**: CONFIRMED. Evidence verbatim at fab-continue.md:193 (Reset Flow step 3, unconditional `fab status reset`). Go ground truth: status.go:41 (and :51/:58 overrides) define reset From:{done,ready,skipped} — `active` rejected by lookupTransition (status.go:64-83) with non-zero exit ("Cannot reset stage ..…

#### `a107` [SHOULD-FIX] fab-continue points to a 'Review Behavior' section that does not exist in _review.md

**Location**: ## Review Behavior (line 149) · **Category**: spec-drift · **Found by**: lens-helper-integrity

_review.md's H1 is `# Shared Review Dispatch` and its sections are Preconditions / Inward Sub-Agent Dispatch / Outward Sub-Agent Dispatch / Parallel Dispatch / Findings Merge — there is no heading named 'Review Behavior'. Reword fab-continue line 149 to e.g. 'then follow its dispatch-and-merge procedure (Preconditions through Findings Merge)', or rename _review.md's H1 to 'Review Behavior'. Under the post-#392 pointer discipline (pointed-to sections must exist with matching heading names — the standard every other diet pointer meets), this bolded pseudo-heading is the one dangling reference.

**Verifier**: Confirmed: src/kit/skills/fab-continue.md:149 says "follow its **Review Behavior**" where "its" = _review.md, but _review.md (H1 "# Shared Review Dispatch") contains no heading named "Review Behavior" — sections are Preconditions / Inward Sub-Agent Dispatch / Outward Sub-Agent Dispatch / Parallel Di…

#### `a108` [SHOULD-FIX] Five caller sites still invoke `/fab-clarify intake`, which the current argument contract parses as a change-name override

**Location**: Arguments :24, rework-menu Revise-requirements row :163, Reset Flow :191, Error Handling :208; also _pipeline.md:106 · **Category**: correctness · **Found by**: fab-clarify

fab-clarify.md:28 says "Any positional argument is treated as a change name", so `/fab-clarify intake` runs `fab preflight intake` — case-insensitive substring matching against fab/changes/ folder names — which either fails resolution mid-recovery-path with a confusing 'change not found' error, or silently targets an unrelated change whose folder slug contains 'intake'. The `intake` target token is residue from the pre-1.10.0 multi-target era (fab-clarify only lists spec/plan/tasks as removed). Either (a) drop the token at all five call sites (plain `/fab-clarify`), or (b) have fab-clarify's Arguments section special-case the literal `intake` as an accepted legacy no-op target consumed before change-name resolution — (b) is safer given _pipeline.md:106 was freshly written in #393. Update both SPEC mirrors.

**Verifier**: Confirmed at stated severity (should-fix). All five caller sites verified: fab-continue.md:24 (Arguments), :163 (rework menu, evidence quote verbatim), :191 (Reset Flow), :208 (Error Handling), and _pipeline.md:106; SPEC-fab-continue.md:5 and SPEC-_pipeline.md:26 also carry the construct, so the sug…

#### `a109` [SHOULD-FIX] fab-continue has no dispatch path for review-pr:failed, and its Reset Flow on that state errors at the CLI

**Location**: Step 1 dispatch table (lines 42-59) and Reset Flow (lines 189-196) · **Category**: correctness · **Found by**: lens-pipeline-coherence

Batch 4 added the review/failed row, but the analogous review-pr/failed state (left behind when git-pr-review's Step 6 runs `fail` — e.g., no PR found) matches nothing: the Step-1 guard keys only `progress.review`, the review-pr row keys `active`/`ready`, and Go's CurrentStage/DisplayStage yield (review-pr, done-on-ship) so no table row fires. Worse, `/fab-continue review-pr` as a reset target runs `fab status reset <change> review-pr`, which Go rejects (reset From = {done, ready, skipped}, not failed). Add a `review-pr | failed` row (keyed on `progress.review-pr == failed`) that re-executes /git-pr-review behavior — its Step 0 `start` already handles failed→active — matching the State Table's `review-pr (fail) → /git-pr-review` row; and note in Reset Flow that a failed review/review-pr is recovered via `start`, not `reset`.

**Verifier**: Independently confirmed all claims. (1) fab-continue.md line 42 guard keys only progress.review; line 58 review-pr row keys active/ready only — no row for review-pr failed. (2) State is reachable: git-pr-review.md Step 6 (line 185) runs `fab status fail <change> review-pr` on failure outcomes (no PR…

#### `a110` [SHOULD-FIX] fab-continue has no dispatch row for review-pr failed — preflight derivation at that state matches the 'all done / Change is complete' row

**Location**: § Normal Flow Step 1, dispatch guard (line 42) and table rows 55/58-59 · **Category**: correctness · **Found by**: lens-cli-contract

The batch-4 guard and the `review`/`failed` row cover only progress.review. For `review-pr: failed` (the state git-pr-review's Step 6 fail path leaves), preflight emits `stage: review-pr`, `display_stage: ship`, `display_state: done` (empirically verified in a sandbox), so no row matches — the nearest match is `all done | — | Block: "Change is complete."`, which mis-reports a failed PR review as complete. Add a `review-pr`/`failed` row (keyed on `progress.review-pr == failed`, like row 55's note) that re-runs /git-pr-review behavior — its own Step 0 `fab status start <change> review-pr ... || true` already handles the failed→active transition. Distinct from known-open f045 (_preamble State Table derivations) and f019 (review-only row).

**Verifier**: CONFIRMED. (1) Evidence quote verbatim at src/kit/skills/fab-continue.md:42 — the Review-failed dispatch guard keys only on progress.review; the dispatch table (lines 49-59) has rows for review-pr active/ready (line 58) and review/failed (line 55, explicitly keyed on progress.review) but nothing for…

#### `a111` [SHOULD-FIX] Rework triage policy and the three rework-path actions are divergent near-twins in fab-continue and _pipeline

**Location**: § Review Behavior → Verdict Fail options table + triage paragraph (lines 157-165) vs _pipeline.md § Auto-Rework Loop (lines 77, 87-92) · **Category**: duplication · **Found by**: lens-duplication

The same three rework paths (Fix code / Revise plan / Revise requirements) with the same artifact edits ('uncheck affected tasks in plan.md ## Tasks with a rework comment', 'edit plan.md ## Requirements plus the downstream ## Tasks/## Acceptance it affects') are now defined twice — fab-continue's manual menu and _pipeline's autonomous heuristics — and the triage-policy sentence has already drifted (skipped vs acknowledged-but-deferred; brace vs no-brace in the rework marker). #393 single-sourced the loop for ff/fff but left this manual/auto twin. Low-churn fix per the accepted f032 pattern: align the wording byte-for-byte and add reciprocal keep-in-sync comments; alternatively single-source the path definitions (trigger → artifact edit) in one file and have the other reference it, noting _pipeline already points exhausted users at fab-continue's menu.

**Verifier**: Confirmed. fab-continue.md:159-165 (manual rework menu + triage sentence) and _pipeline.md:77,87-92 (auto-rework decision heuristics + triage sentence) define the same three rework paths with near-identical artifact-edit text: 'Revise requirements' action is byte-identical; 'Fix code' differs only b…

#### `a112` [NICE-TO-HAVE] Hydrate's optional pattern capture is sequenced after the finish, making it unreachable on resume

**Location**: ## Hydrate Behavior > Steps, lines 184-185 · **Category**: idempotency · **Found by**: fab-continue

Step 5 marks hydrate `done` (auto-activating ship) before step 6's memory writes. An interruption between 5 and 6 silently loses pattern capture: a re-run dispatches ship, and the finish-time true_impact computation (status.go WriteTrueImpact on hydrate finish) also predates the step-6 writes. In the ff/fff subagent path the order is already effectively 6-before-finish (the subagent skips step 5 and the orchestrator finishes after it returns). Swap steps 5 and 6 so the finish is last, and update the subagent note's 'skip step 5 (the finish)' reference to the new number.

**Verifier**: Core claim verified: src/kit/skills/fab-continue.md:184-185 sequences the hydrate finish (step 5) before optional pattern capture (step 6). Interruption between them silently loses capture — once hydrate is done, the Step 1 dispatch table (line 57) routes re-runs to ship and _pipeline.md:110 skips h…

#### `a113` [NICE-TO-HAVE] Ship and review-pr dispatch rows cite a `ready` state the Go state machine disallows for those stages

**Location**: ## Normal Flow > Step 1 dispatch table, lines 57-58 · **Category**: correctness · **Found by**: fab-continue

Go AllowedStates excludes `ready` for both ship ({pending, active, done, skipped}) and review-pr ({pending, active, done, failed, skipped}) — status.go:26-27 — and preflight's Validate rejects a .status.yaml containing ship:ready. The `ready` half of these two State cells describes an unreachable state and muddies the 'only if the stage is still `active`' guards in the same rows. Change both cells to `active` (the review row's `active`/`ready` is fine — review legitimately allows ready).

**Verifier**: Confirmed. fab-continue.md:57-58 list State `active`/`ready` for ship and review-pr, but Go AllowedStates (src/go/fab/internal/status/status.go:26-27) excludes `ready` for both stages, and preflight calls status.Validate (preflight.go:69) before deriving stage/display_state — so a .status.yaml with…

#### `a114` [NICE-TO-HAVE] fab-continue understates the reset transition's From-set (omits `skipped`)

**Location**: ### Step 4: Update `.status.yaml` (line 85) · **Category**: correctness · **Found by**: lens-helper-integrity

Both the Go CLI (src/go/fab/internal/status/status.go:41 — `"reset": {From: []string{"done", "ready", "skipped"}, To: "active"}`) and _cli-fab.md's fab status table ('done/ready/skipped → active') accept resetting a `skipped` stage; only fab-continue's enumeration drops `skipped`. Since the Reset Flow can legitimately target a previously skipped stage, change the line to 'done/ready/skipped → active' to match the CLI and _cli-fab.

**Verifier**: Confirmed: fab-continue.md:85 says 'done/ready → active' while status.go:41 (plus override maps at :51/:58) and _cli-fab.md:59 both accept 'done/ready/skipped → active'. The skipped state is reachable (skip event at status.go:42, _cli-fab.md:60; batch archive treats hydrate:done|skipped as terminal)…

#### `a115` [NICE-TO-HAVE] fab-continue Step 4 documents reset as "done/ready → active", dropping the `skipped` source state the Go machine and _cli-fab both allow

**Location**: Step 4: Update .status.yaml (line 85) · **Category**: staleness · **Found by**: lens-pipeline-coherence

Go's reset transition is From {done, ready, skipped} (status.go:41), and _cli-fab.md:59 correctly says "done/ready/skipped → active". Align the fab-continue bullet to "done/ready/skipped → active" so an agent doesn't wrongly conclude a skipped stage cannot be reset (the supported path for re-entering a skipped stage).

**Verifier**: Confirmed. src/kit/skills/fab-continue.md:85 documents reset as "done/ready → active", while the Go machine (src/go/fab/internal/status/status.go:41 and the per-stage maps at :51/:58, plus cmd/fab/status.go:254) allows From {done, ready, skipped}, and src/kit/skills/_cli-fab.md:59 correctly says "do…

#### `a116` [NICE-TO-HAVE] fab-continue's reset event description omits the 'skipped' from-state

**Location**: § Step 4: Update .status.yaml, event command list (line 85) · **Category**: spec-drift · **Found by**: lens-cli-contract

Go's reset transition accepts {done, ready, skipped} → active (src/go/fab/internal/status/status.go:41), and _cli-fab.md:59 documents all three. Align the fab-continue line to "done/ready/skipped → active" so the Reset Flow is known to work on a stage previously skipped via `fab status skip` (e.g., a skip-cascaded ship/review-pr).

**Verifier**: CONFIRMED. (1) Evidence verbatim at src/kit/skills/fab-continue.md:85: "`fab status reset <change> <stage> fab-continue` — done/ready → active (cascades downstream to pending)" — omits "skipped". (2) Ground truth verified: src/go/fab/internal/status/status.go:41 has reset From:{done,ready,skipped} (…

#### `a117` [STRUCTURAL] _srad loads unconditionally though it is needed only at the same moments as the stage-conditional _generation

**Location**: frontmatter line 4 + Step 3, line 67 · **Category**: architecture · **Found by**: fab-continue

Batch 3 (f122) made _generation and _review stage-conditional for this skill, but left _srad (~6KB) in frontmatter, so hydrate/ship/review-pr invocations and apply-resumes still pay for it. SRAD is exercised exactly when _generation is: intake generation/regeneration and plan generation ('Inside apply, under-specified requirements are resolved inline as graded SRAD assumptions'). Move _srad to the same in-body conditional read instruction as _generation ('read .claude/skills/_srad/SKILL.md alongside _generation when generating an artifact'), update the Step 3 '(loaded via `helpers:`)' parenthetical, the stage-conditional note at line 11, the SPEC mirror's Helpers paragraph, and _srad.md's header claim that all six planning skills declare it via `helpers:`.

**Verifier**: Confirmed real and actionable. fab-continue.md:4 declares helpers: [_srad] unconditionally while the line-11 note makes _generation/_review stage-conditional. Verified the core claim: every SRAD touchpoint in fab-continue coincides with a _generation read — intake regeneration (Step 3 'Intake only',…

### `src/kit/skills/fab-discuss.md`

#### `a118` [NICE-TO-HAVE] fab-discuss tells the agent to read .status.yaml 'for the current stage', but .status.yaml has no stage field — derivation from the progress map is left implicit

**Location**: Context Loading, line 33 · **Category**: ergonomics · **Found by**: entry-points

src/kit/templates/status.yaml contains only a `progress:` map (no top-level `stage:` key), so an agent looking for a stage field must improvise. Either append the derivation rule inline ('derive the stage from the progress map — the stage marked active/ready, per _preamble § State derivation') or replace the raw read with `fab preflight {name}`, whose YAML output already includes `stage`/`display_stage` and is side-effect-free (adjusting the 'Runs preflight? No' Key Properties row if so).

**Verifier**: Confirmed at stated severity (nice-to-have/ergonomics). Evidence verbatim at src/kit/skills/fab-discuss.md:33. Ground truth: src/kit/templates/status.yaml has only a progress: map; zero top-level stage: keys in any real fab/changes/*/.status.yaml; Go statusfile schema has only nested confidence.comp…

### `src/kit/skills/fab-draft.md`

#### `a119` [NICE-TO-HAVE] Hard-coded Next: command lists omit /fab-proceed from state-table-derived rows

**Location**: ## Behavior delta 3 (line 30) and file trailer (line 48) · **Category**: convention · **Found by**: lens-reference-integrity

The _preamble State Table intake row lists "/fab-continue, /fab-ff, /fab-fff, /fab-proceed, /fab-clarify" and the initialized row lists "/fab-new, /fab-proceed, /docs-hydrate-memory", but the hard-coded lists in fab-draft.md:30/48, the fab-new.md:223 trailer, and fab-setup.md:432-436 (whose Next Steps Reference explicitly claims "All `Next:` lines are derived from the state table") all drop /fab-proceed. Either add /fab-proceed to these literals or remove it from the state-table rows if the omission is deliberate — currently the "derived from the state table" claim is false for these four lines (distinct from known-open f083, which covers the hard-coded 'initialized' state and missing update-path Next: lines).

**Verifier**: Confirmed at all four cited locations: src/kit/skills/fab-draft.md:30,:48 and fab-new.md:223 omit /fab-proceed from the intake-state list; fab-setup.md:430-436 omits it from the initialized-state list while explicitly claiming "All Next: lines are derived from the state table". Ground truth: _preamb…

### `src/kit/skills/fab-ff.md`

#### `a120` [SHOULD-FIX] Driver parameter row claims `{driver}` is 'passed to every fab status event command', contradicting _pipeline's deliberate driver-less fail and recovery-start commands

**Location**: § Behavior parameter table, `{driver}` row (line 32); identical row in src/kit/skills/fab-fff.md line 32 · **Category**: correctness · **Found by**: ff-fff-draft

_pipeline.md deliberately omits the driver on `fab status fail <change> review` (per-cycle item 1 and the exhaustion Stop) and on the Resumability `fab status start <change> review` recovery ('preserved verbatim from the pre-extraction drivers, passes none'), and its Behavior note promises 'every conforming run leaves the same `.status.yaml` history shape'. An agent obeying the driver file's 'every' appends `fab-ff`/`fab-fff` to those commands (status.go accepts it; Start records it in metrics), producing a different history shape than a bracket-literal run — exactly the divergence the choreography note exists to prevent. Reword the row in both drivers to match the bracket, e.g. '`fab-ff` — substituted wherever the bracket's commands show `{driver}`, and used in re-run guidance'. Introduced by #393 (new parameter-table text); not covered by g1-4/f006.

**Verifier**: Confirmed. The quote is verbatim at src/kit/skills/fab-ff.md:32 and fab-fff.md:32, and the 'every' claim is factually false: _pipeline.md has three driver-less fab status commands (Resumability start review at line 51, per-cycle fail review at line 81, exhaustion-Stop fail review at line 96), and it…

#### `a121` [NICE-TO-HAVE] Force-mode output header underspecified: composing the Arguments instruction with the Output template yields the contradictory 'gate passed. (force mode -- gate bypassed)'

**Location**: § Arguments `--force` bullet (line 22) vs § Output template header (line 42); identical pair in src/kit/skills/fab-fff.md lines 22/68 · **Category**: ergonomics · **Found by**: ff-fff-draft

The Output template hardcodes '..., gate passed.' with no force variant, so a literal agent appends the force suffix to a 'gate passed' claim that was never evaluated; the source of `{score}` in force mode (preflight YAML `confidence` field, since `--check-gate` is skipped) is also unstated. Show the force-mode header explicitly in both files, e.g. `/fab-ff — confidence {score} of 5.0 (force mode -- gate bypassed).` replacing 'gate passed.', and note the score comes from preflight's `confidence` field. Pre-existing (untouched by #393) but unreported and hit on every --force run.

**Verifier**: Confirmed at all four cited locations. fab-ff.md:22 and fab-fff.md:22 both instruct 'Output header includes "(force mode -- gate bypassed)"' while the only Output templates (fab-ff.md:42 '/fab-ff — confidence {score} of 5.0, gate passed.'; fab-fff.md:68 '/fab-fff — intake confidence {score} of 5.0,…

#### `a122` [NICE-TO-HAVE] Residual twin drift in the per-driver Output blocks: header wording and apply-output annotation diverge, resuming sentence duplicated verbatim

**Location**: § Output (lines 41-58) vs src/kit/skills/fab-fff.md § Output (lines 65-93) · **Category**: duplication · **Found by**: ff-fff-draft

fab-fff's header reads 'intake confidence {score}' where fab-ff says just 'confidence {score}', and fff annotates the apply section '(plan generation — incl. ## Requirements — + task execution)' where ff says '(plan generation + task execution)'; the 'Resuming shows `(resuming)...` header and `Skipping {stage} — already done.`' sentence is duplicated verbatim in both. (The `## Assumptions (cumulative)` presence delta is known f070 — excluded here.) This is exactly the drift pattern f007/#393 set out to kill but left in the per-driver Output blocks. Either normalize the shared wording now, or structurally: hoist the shared Output frame (header, Implementation/Review/Hydrate sections, resuming/bail sentence) into _pipeline.md with `{driver}` substitution, leaving only the fff-only Ship/Review-PR sections and timeout closing line in fab-fff.md.

**Verifier**: All three claims verified verbatim: fab-ff.md:42 "confidence {score}" vs fab-fff.md:68 "intake confidence {score}"; ff:45 "(plan generation + task execution)" vs fff:71 "(plan generation — incl. ## Requirements — + task execution)"; resuming/bail sentences at ff:58 and fff:93 duplicated word-for-wor…

#### `a123` [NICE-TO-HAVE] Residual fab-ff/fab-fff verbatim twin prose contradicts _pipeline's 'single authoritative source' claim

**Location**: § Purpose (line 15), § Arguments (lines 21-22), § Output closing lines (line 58) — byte-identical in fab-fff.md lines 15, 21-22, 93; claim at _pipeline.md lines 22-23 · **Category**: duplication · **Found by**: lens-duplication

Outside the bracket the drivers still share ~10 verbatim lines: the gate/rework/clarify-is-intake-only Purpose sentences, the entire Arguments section (including the identical '--force ... (force mode -- gate bypassed)' bullet), and the Output resuming/bail convention ('Resuming shows `(resuming)...` header and `Skipping {stage} — already done.`'). One header has already drifted ('confidence {score}' vs 'intake confidence {score}'). Either move the shared Arguments and resume-output convention into _pipeline (it already owns Resumability) and trim Purposes to scope-only one-liners, or soften _pipeline's single-source claim to cover behavior steps only.

**Verifier**: Confirmed at stated severity (nice-to-have). All cited evidence verified byte-for-byte: the gate/rework Purpose sentence is verbatim in src/kit/skills/fab-ff.md:15 and fab-fff.md:15; Arguments sections (lines 21-22) are byte-identical in both including the --force '(force mode -- gate bypassed)' bul…

#### `a124` [NICE-TO-HAVE] Purpose says the intake gate is 'checked before the bracket' while Behavior and the SPEC say the bracket owns it

**Location**: § Purpose (line 15); same phrase in src/kit/skills/fab-fff.md line 15; SPEC-fab-ff.md Summary says 'The bracket owns the single intake confidence gate' while its bookkeeping table says 'Before the bracket (intake gate)' · **Category**: spec-drift · **Found by**: ff-fff-draft

The gate is _pipeline.md's Pre-flight step 3, i.e. inside the bracket — the same skill files say two paragraphs later that 'The bracket defines ... pre-flight (intake prerequisite + intake gate)'. Worst case is a harmless doubled read-only check, but it muddies the thin-wrapper contract #393 established. Change both Purposes to 'checked in the bracket's pre-flight' and align SPEC-fab-ff.md's bookkeeping-row Trigger ('Before the bracket') with SPEC's own 'bracket owns the gate' sentence. Introduced by the #393 Purpose rewrite.

**Verifier**: Confirmed at all cited locations. fab-ff.md:15 and fab-fff.md:15 both say the gate is 'checked before the bracket', while line 35 of each file says 'The bracket defines ... pre-flight (intake prerequisite + intake gate)', and _pipeline.md (§ Pre-flight step 3) — self-declared 'single authoritative s…

### `src/kit/skills/fab-fff.md`

#### `a125` [MUST-FIX] fab-fff Steps 4-5 pass `change: {id}` to /git-pr and /git-pr-review, but both skills self-resolve only the ACTIVE change — the <change-name> override ships/mutates the wrong change

**Location**: § Step 4: Ship and § Step 5: Review-PR (lines 39-61), interacting with § Arguments (line 21) · **Category**: correctness · **Found by**: ff-fff-draft

git-pr.md Step 0 runs 'Run `fab change resolve 2>/dev/null`' (no argument — active change only) and ships the current branch; git-pr-review.md Step 0 likewise ('If an active change resolves (`fab change resolve 2>/dev/null`)'). Neither skill can consume the dispatched `{id}`. So when /fab-fff is invoked with `<change-name>` ('target a specific change instead of the active one' — the parallel-tabs workflow _preamble's Change-name override explicitly advertises), Steps 1-3 correctly operate on the override change while Steps 4-5 start/finish ship and review-pr on the ACTIVE change's .status.yaml, derive PR type/issues from the wrong intake, and commit/push whatever branch is checked out. Fix: either have git-pr/git-pr-review accept an explicit change argument (`fab change resolve {id}`) plus a branch-matches-change guard, or add a Step 4 precondition to fab-fff: if `{id}` is not the active change (or the current branch ≠ change folder name), STOP with 'ship requires the change to be active — run /fab-switch {name} and re-run'. The prior review noted the active-only resolution only as a verifier caveat inside f006; it was never filed against fff's override path.

**Verifier**: CONFIRMED. (1) Evidence verbatim: src/kit/skills/fab-fff.md:43 "Dispatch `/git-pr` as subagent — change: `{id}`" and :53 for /git-pr-review; :21 advertises the `<change-name>` override; the bracket (src/kit/skills/_pipeline.md:29) passes the override to preflight, so Steps 1-3 genuinely operate on t…

### `src/kit/skills/fab-help.md`

#### `a126` [NICE-TO-HAVE] fab-help Purpose understates the help output: it lists git-*, docs-*, fab sync, batch commands, and packages, not only /fab-* commands

**Location**: Purpose, line 12 · **Category**: correctness · **Found by**: entry-points

fabhelp.go renders /git-branch, /git-pr, /git-pr-review, the four /docs-* skills, `fab sync`, `fab batch *`, and the wt/idea PACKAGES section. Reword to match the frontmatter description, e.g., 'list every skill command (and the non-skill fab/batch entries) with a one-line description', so the body doesn't contradict the actual output of the subcommand it declares the single source of truth.

**Verifier**: Confirmed. fab-help.md:12 says the help lists "every /fab-* command", but fabhelp.go (src/go/fab/cmd/fab/fabhelp.go) renders all non-partial, non-internal skills including /git-branch, /git-pr, /git-pr-review and the four /docs-* skills (scanSkills lines 193-234, skillToGroupMap lines 24-46), plus h…

### `src/kit/skills/fab-new.md`

#### `a127` [SHOULD-FIX] Output ordering violates _srad's SHALL: Assumptions summary must be the final block immediately before Next:, but Confidence/Activated/Branch lines intervene

**Location**: ## Output (lines 182-189) vs _srad.md § Assumptions Summary Block (line 92) · **Category**: spec-drift · **Found by**: fab-new

_srad.md:92 mandates: 'Every planning skill invocation SHALL include an Assumptions summary as the final content block of its output — immediately before the closing `Next:` line'. fab-new's Output places the Assumptions block before Confidence:, Activated:, and Branch: lines, three lines ahead of Next:. The batch-3 'skill file wins' scoping (f001) covers only the always-load contract and the Next:-line ending, not helper content rules, so this is a live contradiction an agent must arbitrarily resolve. Either move the `{if assumptions: ...}` block to just before the Next: line in fab-new's Output (and check fab-draft inherits it correctly), or soften _srad's placement rule to 'unless the skill's own Output template orders it otherwise'.

**Verifier**: Confirmed. fab-new.md:182-189 places the {if assumptions: ...} block before Confidence:/Activated:/Branch:, three content lines ahead of Next:, while _srad.md:92 (loaded via fab-new's helpers: [_generation, _srad]) mandates the Assumptions summary as the final content block immediately before Next:.…

#### `a128` [SHOULD-FIX] fab-new Step 11 / git-branch: dirty working tree silently rides into the new change's branch — the documented caveat covers only committed work

**Location**: Step 11 branch table rows 2/3/5 (lines 154-160) and Error Handling (line 205); mirror at git-branch.md:140 · **Category**: correctness · **Found by**: git-state-safety-sweep

Extend the caveat in both fab-new.md row 5 and git-branch.md Step 4 (the comment at fab-new.md:150 mandates keeping them in sync) to state that uncommitted changes carry over on every checkout path, and add a `git status --porcelain` dirtiness check that warns (suggest commit/stash) before switching. Impact: one change's in-flight edits silently contaminate another change's branch and PR.

**Verifier**: Confirmed at all cited locations. fab-new.md:160 and git-branch.md:140 document only committed-HEAD inheritance for the row-5 checkout -b path; git's actual behavior also transplants uncommitted working-tree changes on every successful switch (row 2 git checkout when non-conflicting; rows 3/5 git ch…

#### `a129` [SHOULD-FIX] Backlog-ID collision pre-check is substring-based, not ID-anchored — single false-positive match wrongly routes to resume and silently skips creation

**Location**: Step 3: Create Change, backlog bullet (line 50) · **Category**: correctness · **Found by**: fab-new

`fab change resolve` resolves via case-insensitive SUBSTRING match (resolve.go resolveOverride), so a 4-char backlog ID matching a slug fragment (e.g., id `able` vs `...-enable-feature`) or a date prefix (all-digit id `2606` vs `2606xx-...`) resolves to an unrelated change; fab-new then reports 'Change {name} already exists' and STOPs — the change is never created and the anchored CLI safety net (`hasIDCollision` pattern `??????-{id}-*`) never fires. Anchor the pre-check on the ID segment: e.g., `[ "$(fab resolve \"{id}\" 2>/dev/null)" = "{id}" ]` (fab resolve outputs the matched folder's own 4-char ID, so equality proves a true ID collision); only then route to resume. (Multi-match resolve failure is already benign — the `Change ID already in use` safety net catches it.)

**Verifier**: Confirmed. fab-new.md:50 instructs `fab change resolve {id} 2>/dev/null` and line 53 treats any successful resolution as proof of an existing change (route to resume + STOP). resolveOverride (src/go/fab/internal/resolve/resolve.go:98-122) falls back to case-insensitive substring matching after an ex…

### `src/kit/skills/fab-operator.md`

#### `a130` [MUST-FIX] Autopilot per-change loop double-dispatches: spawn-sequence step 5 embeds '<command>' in the tab-open, but Gate and Dispatch come after

**Location**: §6 Autopilot, per-change loop steps 1-4 (lines 489-492) vs Spawning an Agent step 5 (line 391) · **Category**: correctness · **Found by**: fab-operator

Spawn-sequence step 5 opens the tab as `"<spawn_cmd> '<command>'"` — the agent starts running an initial command at tab-open. But autopilot's own step 3 is "Gate — check confidence score" and step 4 is "Dispatch — send /fab-fff", both AFTER step 2 executed spawn steps 3-6. So either '<command>' is undefined in the autopilot path, or the command launches before the gate and step 4 double-dispatches. This seam is a regression from #393's f049 single-sourcing (the pre-szxd autopilot step 2 read "open agent tab" with no embedded command). Fix: run the gate before the spawn sequence, define autopilot's '<command>' as the step-4 command (e.g. `/fab-fff <change>`), and delete the separate Dispatch step — or explicitly state the autopilot tab opens bare and dispatch is via send-keys.

**Verifier**: CONFIRMED with one severity disagreement (must-fix is overstated; should-fix/medium is right).

Verified facts (src/kit/skills/fab-operator.md): (1) Spawn-sequence step 5 (line 391) opens the tab as `tmux new-window ... "<spawn_cmd> '<command>'"` — the command is embedded at tab-open and step 5 offe…

#### `a131` [SHOULD-FIX] fab-operator Spawn-in-worktree principle points to §5 for the spawn sequence, which lives in §6

**Location**: §1 Principles › Spawn-in-worktree, line 23 · **Category**: cross-reference staleness · **Found by**: lens-reference-integrity

Change "(see §5)" to "(see §6)": the spawn sequence is defined in §6 Coordination Patterns › Spawning an Agent (steps 1–6, line 381); §5 is Auto-Nudge and contains no spawn/tab-opening content. Every other §-pointer in the file resolves correctly (verified all ~25). The mispointer predates the review batches but survived #391's edit of this exact line and #393's f049 spawn-sequence consolidation.

**Verifier**: Confirmed at src/kit/skills/fab-operator.md:23 — the Spawn-in-worktree principle says "then spawn the agent tab (see §5)", but §5 (line 300) is Auto-Nudge with no spawn content; the 6-step spawn sequence is §6 Coordination Patterns › Spawning an Agent (lines 381–396). The adjacent "(see §4)" pointer…

#### `a132` [SHOULD-FIX] Implicit queue chaining contradicts its own worked example: depends_on:[<prev-change-id>] vs 'ef56 cherry-picks from its same-repo predecessor' / 'depends on ab12'

**Location**: §6 Autopilot repo-spanning paragraph (line 475), Queue ordering table (line 481), Queue Completion Summary example (line 511) · **Category**: correctness · **Found by**: fab-operator

The normative rule says "every change after the first gets depends_on: [<prev-change-id>]" — for the chain ab12 → cd34(other repo) → ef56, ef56's dep is cd34, which is cross-repo and therefore ordering-only: ef56 receives NO code from ab12. Yet line 475 says ef56 "cherry-picks from its same-repo predecessor" and the completion-summary example annotates "ef56: ... depends on ab12". These produce materially different worktrees. Decide the semantics — strict queue-previous chaining, or nearest-same-repo-predecessor chaining — and fix either the rule or both example passages to match.

**Verifier**: Confirmed: genuine internal contradiction in src/kit/skills/fab-operator.md, introduced by commit 2ec69d99 (Operator Multi-Repo, PR #382). The strict queue-previous rule is stated FOUR times (line 452 'every change after the first automatically gets depends_on: [<prev-change-id>]', line 481 same in…

#### `a133` [SHOULD-FIX] fab-operator: cherry-pick/rebase commands hardcode `origin/main` — guaranteed git failure on any coordinated repo whose default branch is not literally `main`, with no fetch step to keep the ref fresh

**Location**: §6 Dependency Resolution (lines 429-445); Pipeline Reference (line 379); Autopilot (lines 485, 499) · **Category**: correctness · **Found by**: git-state-safety-sweep

Resolve the base ref at use-time — document a `git fetch origin` followed by `git symbolic-ref refs/remotes/origin/HEAD --short` (fallback: probe origin/main then origin/master) — and substitute that variable for every hardcoded `origin/main`. Impact: dependency resolution and merge-on-complete autopilot are unusable on master-default repos and conflict-prone on stale clones.

**Verifier**: Confirmed at stated severity (should-fix, correctness). All quotes verified at src/kit/skills/fab-operator.md lines 431/434/445 (cherry-pick + rationale), 379 (maintenance), 485/499 (merge-on-complete rebase), 11/21 (multi-repo claim); 'fetch' appears nowhere in the file. Hard-failure claim is solid…

#### `a134` [SHOULD-FIX] Initial command '/fab-switch <change> && /fab-proceed' relies on undefined slash-command chaining semantics

**Location**: §6 Working a Change, entry-form table, 'Existing change' row (line 463) · **Category**: ergonomics · **Found by**: fab-operator

This string is passed as the agent's single initial prompt (`"<spawn_cmd> '<command>'"`). Claude Code parses a leading slash command and passes the remainder as its arguments — `/fab-switch` would receive `<change> && /fab-proceed` as its argument (likely failing change resolution), and `/fab-proceed` never executes as a skill; `&&` is shell syntax, not prompt syntax. Replace with a single command that takes a target — `/fab-fff <change>` (fab-proceed itself documents this form for named changes) — or specify two sequential sends (switch first, then proceed once the agent is idle).

**Verifier**: Confirmed. Evidence verbatim at src/kit/skills/fab-operator.md:463; spawn step 5 (line 391) shows the chained string is passed single-quoted as the agent's single initial CLI prompt ("<spawn_cmd> '<command>'"), so `&&` reaches Claude Code unevaluated and no second send exists. All signature claims c…

#### `a135` [NICE-TO-HAVE] §1 cross-reference points spawn flow at §5 (Auto-Nudge) instead of §6

**Location**: §1 Principles, 'Spawn-in-worktree' paragraph (line 23) · **Category**: correctness · **Found by**: fab-operator

The spawn sequence lives in §6 (Coordination Patterns → Spawning an Agent); §5 is Auto-Nudge. Change "(see §5)" to "(see §6)".

**Verifier**: Confirmed: src/kit/skills/fab-operator.md:23 says "then spawn the agent tab (see §5)" but §5 (line 300) is Auto-Nudge; the spawn sequence (step 5 "Open agent tab") is in §6 Coordination Patterns → Spawning an Agent (lines 381-396). Sibling refs in the same paragraph ((see §4), (§8)) resolve correctl…

#### `a136` [NICE-TO-HAVE] branch_map retention 'until the operator session ends' is stale pre-server-keyed semantics with no clearing mechanism

**Location**: §4 Branch Map (line 192) · **Category**: staleness · **Found by**: fab-operator

The server-keyed state file survives session end and server restarts (per _cli-fab.md § fab operator tick-start), and §2 Init step 2 restores branch_map across /clear — nothing clears entries at "session end", and "operator session" is undefined (tmux session? Claude conversation?). Residue from the repo-rooted model. Define retention explicitly: entries persist in the state file until the user clears them (and optionally suggest pruning entries whose branches have merged/been deleted) and drop the session-ends clause.

**Verifier**: Confirmed. Quote matches src/kit/skills/fab-operator.md:192 exactly. The "until the operator session ends" clause is verifiably stale residue: (1) the sentence predates the server-keyed model — introduced by 97b12086 (2026-04-02, repo-rooted .fab-operator.yaml era) and untouched through 2ec69d99 (#3…

#### `a137` [NICE-TO-HAVE] Status-frame example is internally inconsistent: header says '7 tracked' for 8 entries, and the 'gmail-deploys' watch has no valid source

**Location**: §4 Status Frame Format, example frame (lines 226-249) vs header rule (line 252) and §7 Schema `source` row · **Category**: correctness · **Found by**: fab-operator

Per the stated rule ("`N tracked` is the total count of all entries (changes + watches)"), the example holds 5 changes (r3m7, ab12, k8ds, ef56, cd34) + 3 watches = 8, not 7 — an LLM learns the count rule from this example. Also, the `gmail-deploys` watch implies a `gmail` source, but §7's schema allows only "`linear` or `slack`". Fix the count to 8 and rename the watch to a supported source (or drop the row to make 7 correct).

**Verifier**: Confirmed both halves. (1) Count: src/kit/skills/fab-operator.md:226 says '**7 tracked**' but the example contains 5 changes (r3m7, ab12, k8ds, ef56, cd34) + 3 watches = 8, contradicting the rule at line 252 ('N tracked is the total count of all entries (changes + watches)'). Git history shows the d…

#### `a138` [STRUCTURAL] Operator status frame is a fully mechanical render (emoji tables, ordering, thresholds) specified in ~3.5KB of prose — a `fab operator frame` subcommand would make it byte-stable and shrink the skill

**Location**: §4 Status Frame Format (lines 216–287) · **Category**: robustness · **Found by**: lens-architecture

Every frame element is deterministic given inputs the binary already owns: `fab pane map --all-sessions --json` (repo/session/stage/idle/pr_url), the server-keyed state file (tick_count, monitored, autopilot, watches incl. last_error/last_checked), and the 15m stuck threshold. Add `fab operator frame` that reads both and prints the bare-markdown frame; the agent appends only the italic action footnotes. This removes per-tick layout drift (column order, emoji mapping, relative-timestamp math) from LLM hands and incidentally gives the binary a defined failure surface for the tick (the state file is already its to read). Tradeoff: the natural-language stuck-threshold override ('flag agents stuck for more than {N}m') needs a `--stuck <dur>` flag; frame format changes move from markdown edits to Go releases.

**Verifier**: Confirmed real and feasible. The frame spec at src/kit/skills/fab-operator.md:215-287 (~5.2KB with the example; ~3.5KB mechanical spec) is a fully deterministic render, and the binary already exposes nearly all inputs: pane map JSON rows carry session/pane/repo/change/stage/agent_state/agent_idle_du…

#### `a139` [STRUCTURAL] fab-operator should load autopilot and watches conditionally — 42% of its 49.8KB body is re-paid on every /clear even when no queue or watch exists

**Location**: §6 Coordination Patterns (lines 365–539, ~17.0KB) and §7 Watches (lines 541–591, ~3.9KB); reload mandate at §1 line 33 · **Category**: context-economy · **Found by**: lens-architecture

Extract §6's Autopilot/Dependency Resolution/Ordered Merge and §7 Watches into `_operator-autopilot.md` and `_operator-watches.md`, loaded via in-body read instructions (the stage-conditional pattern zc9m established for fab-continue's _generation/_review): read each only when the state file has a non-null autopilot queue / non-empty watches map, or when the user requests one. A plain monitoring session (the common case) drops ~21KB per startup and per /clear. Tradeoff: two more internal partials and an agent-compliance dependency — backstop with explicit read lines at Tick Behavior steps 3–4 and at the spawn-sequence entry, mirroring fab-continue's backstops.

**Verifier**: Verified: src/kit/skills/fab-operator.md is 49,807B; §6 (lines 365-539) = 17,157B, §7 (541-591) = 3,932B → 42.3% as claimed; the §1:33 /clear re-read mandate is quoted accurately, so the cost is genuinely re-paid per clear in a long-lived session. Not planned anywhere: fab/backlog.md batches (9u91/u…

### `src/kit/skills/fab-proceed.md`

#### `a140` [SHOULD-FIX] fab-new subagent dispatch has no defined behavior when SRAD requires asking the user

**Location**: § Dispatch Behavior > Conversation Context Synthesis (line 163) and § Error Handling (lines 175-183) · **Category**: ergonomics · **Found by**: fab-proceed

This is aspirational with no failure path: fab-new Step 8 mandates conversational questioning ("when 5+ Unresolved, ask one at a time") and _srad.md:47's Critical Rule says low-R/low-A Unresolved items "MUST always be asked — even in /fab-new", yet the subagent runs in a promptless Agent-tool context with no [AUTO-MODE] prefix (per _preamble, no skill currently passes it). Different agents will stall, fabricate answers, or return questions as the result — and the Error Handling table only covers "fab-new subagent fails". Add to the fab-new dispatch prompt: "subagent context — never prompt; record would-be questions as Tentative/[NEEDS CLARIFICATION] markers in the intake and return", plus an Error Handling row noting that a low-confidence intake is then stopped by /fab-fff's intake gate. (Adjacent to but distinct from Appendix-B f151, which covers description content sourcing, not the ask path.)

**Verifier**: Confirmed at stated severity (should-fix). Evidence quote verbatim at src/kit/skills/fab-proceed.md:163; Error Handling table (lines 175-181) has no row for the ask path. Cross-refs verified: fab-new.md:105 Step 8 mandates SRAD questioning incl. conversational mode for 5+ Unresolved; _srad.md:47 Cri…

#### `a141` [SHOULD-FIX] Dispatch table chains /git-branch after /fab-new, but fab-new has created the branch inline since PR #322

**Location**: § State Detection > Dispatch Table, rows 3 and 5 (lines 89, 91); also mirrored in SPEC-fab-proceed.md lines 79, 81 · **Category**: staleness · **Found by**: fab-proceed

fab-new.md Step 11 ("After activating the change, create or check out the matching git branch inline", line 131) makes the trailing /git-branch subagent a near-guaranteed 'already active' no-op on both fab-new rows — a full wasted subagent dispatch per run, dating from before #322. Either drop /git-branch from the two /fab-new rows (keep it on the /fab-switch rows — fab-switch only tips the user to run /git-branch, lines 77/92), or add a one-line annotation that the dispatch is a deliberate idempotent ensure covering fab-new's non-fatal Step 11 failure paths. Update the SPEC table to match.

**Verifier**: CONFIRMED at stated severity (should-fix, staleness). Evidence verified verbatim: src/kit/skills/fab-proceed.md:89 and :91 chain `/fab-new` -> `/git-branch` -> `/fab-fff`; docs/specs/skills/SPEC-fab-proceed.md:79,81 mirror it; fab-new.md:129-165 (Step 11, "create or check out the matching git branch…

#### `a142` [NICE-TO-HAVE] fab-proceed's context-loading opt-out has no sanctioned home — _preamble §1's exception list omits it and the skill lacks a Context Loading section

**Location**: Header blockquote (line 10) vs _preamble.md §1 Always Load (line 36) · **Category**: convention · **Found by**: fab-proceed

_preamble §1 says the layer applies "unless the skill's own Context Loading section says otherwise" and enumerates current exceptions (/fab-setup, /fab-status, /fab-switch, /docs-hydrate-memory, /fab-operator) — fab-proceed is absent, and its opt-out lives in a header blockquote rather than a Context Loading section, so an agent following §1 literally loads 7 files before state detection while one following the blockquote loads none. Either give fab-proceed a one-line '## Context Loading' section (skips the layer; /fab-fff loads it in the main context) or add fab-proceed's deferral to §1's exception enumeration.

**Verifier**: Confirmed. Evidence quote verbatim at src/kit/skills/fab-proceed.md:10; _preamble.md:36 keys the opt-out to "the skill's own Context Loading section" and enumerates five exceptions (/fab-setup, /fab-status, /fab-switch, /docs-hydrate-memory, /fab-operator) — fab-proceed is absent. grep confirms fab-…

#### `a143` [NICE-TO-HAVE] fab-proceed conflates 'project not initialized' with 'no active change' — never routes to the State Table's (none) → /fab-setup

**Location**: Step 1: Active Change Check (lines 32-38) and Error Handling (line 177) · **Category**: convention · **Found by**: lens-pipeline-coherence

`fab resolve --folder` exits non-zero both when no change is active and when the project is uninitialized (`fab/changes/ not found.`), and fab-proceed treats every non-zero identically — so on an uninitialized repo a substantive conversation dispatches /fab-new (which only then STOPs with "Run /fab-setup first"), and the empty/thin error row advises "run /fab-new (or /fab-draft)" where _preamble's State Table derives (none) → /fab-setup. Branch on the resolve stderr (or stat fab/project/config.yaml, the documented (none) derivation) and short-circuit to "Project not initialized — run /fab-setup first."

**Verifier**: Confirmed. fab-proceed.md:38 treats every non-zero `fab resolve --folder` exit as "no active change" and proceeds to Steps 3/4; resolve.go (FabRoot:23, resolveFromCurrent:139) never checks fab/project/config.yaml, so uninitialized and no-active-change are indistinguishable — especially since fab-pro…

#### `a144` [NICE-TO-HAVE] Date-recency tiebreak is undefined and non-deterministic for same-day drafts

**Location**: § Relevance Assessment step 4 (line 108) and Dispatch Table note (line 95); scan command at line 74 · **Category**: ergonomics · **Found by**: fab-proceed

`-k1,1r` keys only on the YYMMDD field; for two drafts created the same day (a plausible /fab-draft batch), GNU sort's last-resort whole-line comparison picks the lexicographically smallest random ID — ascending, arbitrary, and not 'most-recent'. The empty/thin row's "pick the most-recent by `YYMMDD` prefix" has the same hole, and the arbitrarily chosen draft gets activated and run through the full autonomous pipeline. Use plain `sort -r` on full folder names (date-desc with a deterministic ID tiebreak) in both the Step 4 scan and the tiebreak, and state the same-day rule explicitly; structurally, consider replacing the raw ls/sed/sort pipeline with a `fab change list`-based query so candidate enumeration and ordering live in the Go CLI.

**Verifier**: CONFIRMED at the stated severity (nice-to-have/ergonomics), with one wording correction. Evidence verified verbatim: src/kit/skills/fab-proceed.md:108 (tiebreak `sort -t- -k1,1r | head -1`), :74 (scan pipeline), :95 ("pick the most-recent by `YYMMDD` prefix"). Technical claim independently reproduce…

#### `a145` [NICE-TO-HAVE] Cross-reference '(see Output Format)' points to a heading that exists only in the SPEC

**Location**: § Relevance Assessment step 5 (line 109) · **Category**: correctness · **Found by**: fab-proceed

The skill's sections are titled '## Output' and '### Bypass Notes' — 'Output Format' is a heading only in SPEC-fab-proceed.md. Change the reference to '(see Bypass Notes)' to match the skill's own heading.

**Verifier**: Confirmed. fab-proceed.md:109 says "(see Output Format)" but the skill's headings are "## Output" (line 187) and "### Bypass Notes" (line 200); "Output Format" exists only as SPEC-fab-proceed.md:102 ("### Output Format"), and not in _preamble.md. Since deployed skills live standalone in user project…

### `src/kit/skills/fab-setup.md`

#### `a146` [SHOULD-FIX] stage_directives is a dead config key — edited and migrated everywhere, consumed nowhere

**Location**: config menu, line 167 (plus scaffold comment lines 29-30) · **Category**: dead-config · **Found by**: lens-templates-config

The scaffold promises "Extra instructions the agent follows when generating each stage's artifact", fab-setup offers an interactive editor for the section, and three migrations carefully preserve/relocate it — but grep over all of src/kit/skills (including _generation.md's Intake and Plan Generation Procedures) and src/go finds zero readers: no skill step says "apply stage_directives.{stage}" and config.go only parses stage_hooks/true_impact_exclude/test_paths. Either wire it in (e.g., a step in _generation.md's two procedures and _review.md: "append config.yaml stage_directives.{stage} entries to the generation instructions") or remove the key from the scaffold, the fab-setup menu, and docs/specs/architecture.md:241. Same audit also confirms the documented `model_tiers` key (architecture.md config example) has zero consumers.

**Verifier**: Confirmed end-to-end. Quote verbatim at src/kit/skills/fab-setup.md:167 (also lines 18/138 valid-sections); scaffold promise at src/kit/scaffold/fab/project/config.yaml:29-30. Zero readers: grep over src/kit/skills finds stage_directives only in fab-setup (the editor) — _generation.md's Intake/Plan…

#### `a147` [SHOULD-FIX] Migrations Step 1.3 now requires the agent to semver-compare local vs engine, but #393 deleted the Semver Comparison rule it needs

**Location**: § Migrations Step 1: Discover Migrations, item 3 (line 303) vs line 294 · **Category**: ergonomics · **Found by**: fab-setup

Regression introduced by the f080 dedup in PR #393: the same diff that added this three-way branch deleted the `## Semver Comparison` section defining how to compare (component-wise integer MAJOR/MINOR/PATCH), and the section header above it now says "do NOT read, parse, or compare the version files" (line 294). An LLM comparing e.g. 1.9.10 vs 1.9.9 lexically picks the wrong output. Either re-add a one-line rule at Step 1.3 ("compare component-wise as integers, never lexically"), or — cleaner — add a `relation` field (equal/local_ahead/behind) to `fab migrations-status --json` (the binary already has compareSemver) so the skill never compares versions at all.

**Verifier**: CONFIRMED. (1) Evidence quote matches src/kit/skills/fab-setup.md:303 exactly; the three-way branch requires the agent to order-compare the returned local/engine semver strings. (2) git show a8e720dd (PR #393) verifies the same diff that added this branch deleted the '## Semver Comparison' section (…

#### `a148` [NICE-TO-HAVE] Sync-failure guard claims step 1a 'guarantees' fab_version, but Config Create Mode has no fallback when there is no existing key to preserve

**Location**: § 1c Sync-failure guard (line 95) and § Config Create Mode step 5 (line 153) · **Category**: correctness · **Found by**: fab-setup

Step 1a only preserves fab_version conditionally: "if the existing config.yaml has a `fab_version` key ... carry it into the new file unchanged — the scaffold template lacks it" (line 153). On a repo where config.yaml is wholly absent (e.g. deleted, or skills present without `fab init`), create mode writes the fab_version-less scaffold and `fab sync` then fails (config.go: "no fab_version in config.yaml. Run 'fab init' to set one"), dead-ending the bootstrap after config/constitution were already written. Make the guarantee true: in Config Create Mode step 5, fall back to writing the engine version from `$(fab kit-path)/VERSION` (already resolved in pre-flight) when no key exists to carry over — or weaken the parenthetical and add a recovery hint (`run fab init, then re-run /fab-setup`).

**Verifier**: Confirmed end-to-end. fab-setup.md:95 claims step 1a "guarantees" fab_version, but step 1a's create-mode preserve (line 153) is conditional on an existing key, and the scaffold template (src/kit/scaffold/fab/project/config.yaml) lacks fab_version. In the "If missing" branch step 1a itself enumerates…

#### `a149` [NICE-TO-HAVE] Step 1c's 'owns all non-interactive structural setup' enumeration omits sync's settings.local.json permissions merge, hook registration, .envrc/direnv, and project sync scripts

**Location**: § 1c. fab sync — scaffold, directories, deployment, gitignore, lines 85-97 · **Category**: staleness · **Found by**: fab-setup

sync.go also merges `.claude/settings.local.json` permission rules from scaffold/.claude/fragment-settings.local.json (jsonMergePermissions), registers fab hooks there (syncHooks), line-ensures `.envrc` and runs `direnv allow`, copies `fab/sync/README.md`, and executes project `fab/sync/*.sh` scripts — none appear in the five bullets or the Bootstrap Output template, and the gitignore bullet says only "adds `.fab-*`" while the fragment also adds /.claude, /.agents etc. Since bootstrap modifies the user's Claude permissions/hook settings, add one bullet ("Agent settings: permissions + hook merge into .claude/settings.local.json; .envrc merge + direnv allow; project fab/sync/*.sh scripts") or an explicit catch-all so agents surface sync's full report rather than the abridged list.

**Verifier**: Confirmed at stated severity (nice-to-have). Evidence quote verbatim at src/kit/skills/fab-setup.md:87; the five bullets (lines 89-93), the report instruction (line 97), and the Bootstrap Output template (line 120) omit real sync behaviors verified in src/go/fab-kit/internal/sync.go: (1) jsonMergePe…

#### `a150` [NICE-TO-HAVE] Next Steps Reference omits /fab-proceed from all four 'initialized' lines it claims to derive from the State Table

**Location**: § Next Steps Reference, lines 430-436 · **Category**: convention · **Found by**: fab-setup

The section opens "All `Next:` lines are derived from the state table in `_preamble.md`", but the preamble's initialized row is "/fab-new, /fab-proceed, /docs-hydrate-memory" — all four hard-coded lines (bootstrap, config create, constitution create, migrations) drop /fab-proceed. Either add it, or replace the enumeration with a single "derive at runtime per the _preamble Lookup Procedure" pointer so the list can't drift again. Distinct from known-open f083, which covers the wrong hard-coded state after migrations and the missing update-path lines, not the command-list contents.

**Verifier**: Confirmed. fab-setup.md:430 claims "All `Next:` lines are derived from the state table in `_preamble.md`", but all four enumerated lines (432, 433, 435, 436) list only "/fab-new <description> or /docs-hydrate-memory <sources>" while _preamble.md:257's initialized row is "/fab-new, /fab-proceed, /doc…

### `src/kit/skills/fab-status.md`

#### `a151` [NICE-TO-HAVE] fab-status progress-table symbol legend has no glyph for the `skipped` state

**Location**: ## Behavior, status block bullet (line 46) · **Category**: staleness · **Found by**: lens-pipeline-coherence

`skipped` is a first-class state (`fab status skip` is in _preamble's Common fab Commands and _cli-fab's table; Go's ProgressLine renders it as `⏭`), but fab-status defines no symbol for it — a skipped stage's row rendering is undefined and "Defaults missing progress fields to `○` (pending)" doesn't cover it (the field is present, valued `skipped`). Add a skipped glyph (e.g., `⏭`) to the legend. Distinct from known-open f091 (absence of a literal output template) — this is a state-coverage gap in the enumeration itself.

**Verifier**: Confirmed. fab-status.md:46 legend enumerates exactly five glyphs (✓ done, ● active, ◷ ready, ○ pending, ✗ failed) with nothing for `skipped`, which is a first-class state: status.go:18 ValidStates includes it, it is legal in 5 of 6 stages (status.go:23-27), `fab status skip` exists (status.go:42; _…

#### `a152` [NICE-TO-HAVE] fab-status Key Properties claims config/constitution are not required, but its mandatory preflight hard-fails without them

**Location**: ## Key Properties, line 78 · **Category**: correctness · **Found by**: change-mgmt

`fab preflight` (the skill's first Behavior step) exits non-zero with "Project not initialized — fab/project/config.yaml not found. Run /fab-setup." when either file is missing (preflight.go:33-40), so /fab-status cannot run pre-init. Contrast fab-switch, where the same row is true because `fab change switch` never checks config. Reword to something like "Not loaded for content (Always-Load exempt), but preflight requires both to exist — errors pre-init".

**Verifier**: Confirmed. Evidence verbatim at src/kit/skills/fab-status.md:78. The skill's first Behavior step is unconditional `fab preflight` (fab-status.md:34-38), and preflight.go:33-40 (src/go/fab/internal/preflight/preflight.go) hard-fails with 'Project not initialized — fab/project/config.yaml not found. R…

#### `a153` [STRUCTURAL] fab-status is ~90% deterministic formatting prose that belongs in a Go subcommand, following the fab pr-meta / fab fab-help / fab memory-index precedent

**Location**: ## Behavior, lines 33–66 · **Category**: robustness · **Found by**: lens-architecture

Add `fab status render [<change>]` that emits the whole block byte-stably (the binary already owns every input: preflight YAML, progress-line, plan/confidence extraction, true_impact, kit VERSION vs .kit-migration-version) and collapse the skill to run-and-display plus the Next: line. The prose currently encodes hard-coded thresholds (net>100 / excluding.net>50), emoji/bold rules, and version-drift comparison — exactly the drift class that already bit once (the f047 ANSI mandate the render path strips). Moving it also moots the open ambiguity about which preamble §2 steps apply to a glance command. Tradeoff: constitution Principle I favors logic in markdown, but this is deterministic lookup/formatting (the carve-out the same constitution's CLI constraint already exercises); cost is Go surface + tests + a _cli-fab section per the CLI constraint.

**Verifier**: Verified end-to-end. (1) Evidence accurate: fab-status.md:33-66 is near-100% deterministic formatting prose (hard-coded thresholds net>100/excl>50/+50, exact warning text, emoji/bold rules, version-drift compare, confidence formats, missing-field defaults) — no agent judgment. (2) Precedent claim so…

#### `a154` [STRUCTURAL] fab-status rendering rules (impact thresholds, refactor warning, drift check) are mechanical logic that belongs in the Go CLI

**Location**: ## Behavior, lines 40-66 · **Category**: architecture · **Found by**: change-mgmt

The skill encodes deterministic business rules in prose: two hard-coded impact thresholds, an exact refactor-warning string with a 3-clause trigger, a version-drift comparison, and field-default rules — all re-derived by an LLM on every glance invocation and mirrored verbatim in the SPEC. Following the archive/switch mechanicalization precedent (#365, f087/f094 direction), add a `fab status report [<change>]` Go subcommand that emits the canonical status block (it already has DisplayStage/CurrentStage/StatusFile incl. true_impact and change_type), reducing the skill to display-stdout-plus-Next:. This would also resolve the f091 (no canonical output template), f172 (redundant raw reads), and f201 ('excluding fab/docs' label) known leads at the root.

**Verifier**: Verified end-to-end. (1) Evidence quote matches src/kit/skills/fab-status.md:53; lines 40-66 do encode purely deterministic rules in prose: two hard-coded thresholds (net>100, excluding.net>50, 'MUST NOT be project-configurable'), exact refactor-warning string with 3-clause trigger, version-drift co…

### `src/kit/skills/fab-switch.md`

#### `a155` [MUST-FIX] fab change switch Next: guidance is off-by-one at post-review stages and contradicts both the skill's gloss and fab-status

**Location**: ## Output, lines 84-99 (root cause: src/go/fab/internal/change/change.go:217-223, defaultCommand:427-440) · **Category**: correctness · **Found by**: change-mgmt

The binary prints `Next: NextStage(CurrentStage) (via defaultCommand(CurrentStage))`, where defaultCommand implements the preamble State Table keyed by DONE stage but is fed the next-to-EXECUTE stage. The off-by-one is masked at intake/apply/review (all map to /fab-continue) but emits wrong guidance later: at review-done/hydrate-active it prints `Next: ship (via /git-pr)` (State Table says /fab-continue must run hydrate first); at ship-done/review-pr-pending it prints `Next: /fab-archive` (should be /git-pr-review), which also falsifies the skill's claim "When all stages are done, `Next:` shows only `/fab-archive`" — that line fires whenever routing=review-pr, not only when all done. It also diverges from fab-status, which documents `Next: {routing stage} (via default)` using preflight's `stage` directly (e.g. `Next: apply (via /fab-continue)` where switch prints `Next: review (via /fab-continue)` for the same state). Fix change.go's Switch to print the routing stage with the State-Table-aligned command (matching fab-status), then correct the skill's Output template/gloss; per constitution, update _cli-fab.md alongside.

**Verifier**: Independently confirmed by code reading AND an empirical Go test run against change.Switch (src/go/fab/internal/change/change.go:217-223). Switch prints Next: NextStage(routing) (via defaultCommand(routing)) where routing=status.CurrentStage (next-to-execute stage, status.go:430-461) but defaultComm…

#### `a156` [SHOULD-FIX] fab-switch documents display_state qualifiers as done/active/pending — omitting `ready`, the standard state of every freshly switched draft, and `skipped`

**Location**: ## Output (line 95) · **Category**: staleness · **Found by**: lens-pipeline-coherence

Go's DisplayStage emits five states (tier 1 first `active`, tier 2 first `ready`, tier 3 last `done`/`skipped`, tier 4 first `pending`). /fab-new and /fab-draft both end with `fab status advance {name} intake`, so the canonical switch-to-a-draft flow displays `intake (1/6) — ready` — a qualifier the skill's own contract says cannot occur. The same sentence's gloss "`{display_stage}` is 'where you are' (last active or last done stage)" also misstates the tier order. Update the qualifier list to done/active/ready/pending/skipped and the gloss to match the Go tiers. (Distinct from known-open f088, which covers the no-arg listing mechanism.)

**Verifier**: Confirmed. fab-switch.md:95 says the {state} qualifier is done/active/pending, but status.DisplayStage (src/go/fab/internal/status/status.go:464-499) emits five states: active, ready, done, skipped, pending — and change.Switch (change.go:192-214) prints that state verbatim in the documented Stage li…

#### `a157` [NICE-TO-HAVE] fab-switch no-argument flow never says to run the switch after the user picks from the list

**Location**: ## Behavior › No Argument Flow, lines 29-33 · **Category**: ergonomics · **Found by**: change-mgmt

The flow ends at "wait for selection" with no step to execute `fab change switch "<selected>"`, while the Argument Flow's multi-match branch spells that exact follow-up out ("After selection, run `fab change switch \"<selected>\"`") and SPEC-fab-switch's diagram includes the call. Add step 4: `After selection, run fab change switch "<selected>"` (then Command Logging and the Hint Line apply). Distinct from known-open f088, which covers the ls-scan vs `fab change list` mechanism conflict.

**Verifier**: Confirmed. src/kit/skills/fab-switch.md No Argument Flow (lines 29-33) ends at "wait for selection" (line 33, evidence quote exact) with no step to run `fab change switch "<selected>"`, while the Argument Flow multi-match branch (line 45) spells that follow-up out and the SPEC mirror docs/specs/skil…

### `src/kit/skills/git-branch.md`

#### `a158` [SHOULD-FIX] git-branch: ambiguous multi-match resolution silently creates a junk standalone branch with a false 'No matching change found' message

**Location**: Step 2: Resolve Change Name, lines 53-62 · **Category**: correctness · **Found by**: entry-points

fab change resolve fails for two distinct reasons (resolve.go:119 'Multiple changes match "%s": %s.' vs :122 'No change matches "%s".'). The skill treats every explicit-arg failure as standalone fallback, so an ambiguous-but-valid reference (e.g., /git-branch auth with two auth changes) creates a literal branch named 'auth' and prints a factually wrong message. Branch on stderr: if it starts with 'Multiple changes match', surface the candidate list and STOP (ask the user to disambiguate); only enter standalone fallback on 'No change matches'. Update the Error Handling table row accordingly and mirror in SPEC-git-branch.md.

**Verifier**: Confirmed. Evidence verbatim at src/kit/skills/git-branch.md:53-62 (Error Handling row :170 same conflation). Ground truth: resolve.go:119 'Multiple changes match' vs :122 'No change matches' are distinct exit-1 failures; live repro in this repo — `fab change resolve "skills"` exits 1 listing 3 cand…

#### `a159` [SHOULD-FIX] git-branch: branch-existence check ignores remote-only branches, so a fresh clone/worktree recreates a divergent branch instead of tracking origin

**Location**: Step 4: Context-Dependent Action, lines 85-90 · **Category**: idempotency · **Found by**: entry-points

Plain rev-parse does not match refs/remotes/origin/{branch_name}, so when the change branch exists only on origin (fresh worktree via wt create, second machine, re-cloned repo), the skill falls through to the create paths and runs `git checkout -b` from current HEAD — a silently divergent second lineage whose later push is rejected non-fast-forward. Add a remote check before the create paths: if `git rev-parse --verify "refs/remotes/origin/{branch_name}"` succeeds, run `git checkout -b "{branch_name}" --track "origin/{branch_name}"` and report '(checked out, tracking origin)'. Distinct from Appendix B f178, which is about the check matching too many refs (tags), not too few.

**Verifier**: Confirmed. Evidence quote matches src/kit/skills/git-branch.md:89; all three create paths (lines 110-113, 134-137, 142-145) run `git checkout -b` from current HEAD with no remote check, bypassing git's checkout DWIM. Empirically verified in a scratch repo: with the branch existing only as refs/remot…

#### `a160` [NICE-TO-HAVE] git-branch rename guard enumerates only 'resolution fails' and 'matches a different change' — same-change match and detached HEAD are undefined states

**Location**: Step 4, rename guard bullets, lines 122-140 · **Category**: ergonomics · **Found by**: entry-points

Gap in the batch-1 f100 fix: if the current branch name resolves to the SAME change as the target (e.g., a branch named with the 4-char ID or a slug substring), neither bullet applies — the guard prose ('does not belong to another change') implies rename, but the agent must infer both the outcome and the comparison procedure (compare resolved folder vs target folder; never stated). Also, on detached HEAD `git branch --show-current` prints empty, making the guard run a no-arg resolve and `git branch -m` fail. Add a third bullet ('matches the same change → rename') with an explicit folder-name comparison, and a detached-HEAD row to Error Handling.

**Verifier**: CONFIRMED at stated severity (nice-to-have/ergonomics). Evidence quote verified verbatim at src/kit/skills/git-branch.md:134; guard structure at lines 122-140 enumerates exactly two outcomes (resolution fails -> rename; succeeds-and-matches-DIFFERENT-change -> checkout -b).

Same-change gap verified…

### `src/kit/skills/git-pr-review.md`

#### `a161` [MUST-FIX] git-pr-review: "(no partial state)" claim is false on non-fast-forward push rejection — git reset cannot undo the commit, and the idempotent re-run permanently strands the fixes

**Location**: Step 5 Commit and Push (lines 139-149); Step 6 intro (line 180); Rules (line 229) · **Category**: correctness · **Found by**: git-state-safety-sweep

Split the failure handling: commit failure → `git reset` is correct; push rejection → keep the commit and document recovery (`git pull --rebase origin <branch>` then re-push, or STOP with that guidance). Also extend the re-run gate to check `git log @{u}..HEAD` for unpushed fix commits before declaring "No changes needed", and delete the "(no partial state)" parentheticals at lines 147 and 180. Impact: review fixes are silently lost from the PR while replies and stage state claim they shipped.

**Verifier**: Confirmed. All three quotes verified verbatim (src/kit/skills/git-pr-review.md:147, :180, :229). Step 5 commits before pushing; bare `git reset` is --mixed against HEAD and cannot undo a commit, so a push rejection after a successful commit leaves an unpushed fix commit — the exact partial state lin…

#### `a162` [SHOULD-FIX] Rules 'Fail fast … stop immediately' contradicts the batch-1 Step-6 routing design, and Step 6's 'processing error' outcome is orphaned — no step routes processing errors there

**Location**: ## Rules (line 227) vs Step 6 (lines 180, 185) · **Category**: agent-ergonomics · **Found by**: git-pr-review

PR #390 (f015) rewired all terminal failures through Step 6 ('Step 6 is the exit point for every terminal path after Step 0'), but this Rules line survived unamended: a literal reading says stop without recording the stage failure. Worse, Step 6 outcome 2 lists 'processing error' ('On failure (gh missing, no PR found, processing error)') yet no step text ever routes a mid-triage/edit error to Step 6 — the only general rule covering such errors is this fail-fast line, which says stop immediately instead. Amend the Rules line to: 'Fail fast — on error, report it and exit via Step 6 with outcome failure so the stage is marked; only Step 1.5 (invalid --tool) and Step 5 (commit/push failure) STOP directly', and add a sentence to Step 4 routing processing errors to Step 6 with outcome failure.

**Verifier**: Confirmed at src/kit/skills/git-pr-review.md — Rules line 227 ("Fail fast ... stop immediately, except where a step explicitly declares best-effort handling") vs Step 6 line 180 ("exit point for every terminal path after Step 0, with two exceptions": only Step 1.5 and Step 5). For unanticipated mid-…

#### `a163` [SHOULD-FIX] --tool flag header claims it 'bypasses automatic detection' and 'only that tool is attempted', but Phase 1 detection still runs and silently overrides the flag when comments exist; 'the cascade' is an undefined leftover term

**Location**: Header (line 11) vs Step 2 Phase 1 (line 64) and Phase 2 forced-tool note (line 74) · **Category**: staleness · **Found by**: git-pr-review

The body (line 74) and SPEC (line 105: 'the config check is skipped entirely') scope the flag to skipping only the Phase-2 config check — Phase 1 still runs, and when existing comments are found 'only that tool is attempted' is false (no tool is attempted; the flag is silently ignored, including on the 'reviews exist but no inline comments' no-reviews branch). An agent reading 'bypassing automatic detection' could plausibly skip Step 2 Phase 1 entirely. Reword line 11 to 'Skips the review_tools config check in Step 2 Phase 2; existing reviews still take precedence.' Also line 64's '(skip Phase 2 — the cascade does not run …)' references 'the cascade', a term defined nowhere in the file — residue of the removed multi-tool cascade; say 'the Copilot request is not made' instead.

**Verifier**: Confirmed at stated severity (should-fix, staleness). (1) Header claim verified verbatim at src/kit/skills/git-pr-review.md:11; contradicted by line 74 ("skip the config check below") and SPEC-git-pr-review.md:105 ("the config check is skipped entirely") + :99 (Phase 2 only runs when Phase 1 finds n…

#### `a164` [SHOULD-FIX] Step 5 push-failure handling cannot deliver its promised 'no partial state' — git reset does not undo a successful commit, and the re-run path posts 'Fixed' replies citing an unpushed SHA

**Location**: Step 5: Commit and Push (lines 139-149) and Step 5.5 (line 165) · **Category**: correctness · **Found by**: git-pr-review

Split the two failure cases. Commit failure: git reset, report, STOP (working-tree edits remain — say so instead of claiming 'no partial state'). Push failure: the commit already exists; do NOT git reset (it's a no-op), report and STOP noting the local commit is retained and a re-run pushes it. Then harden the re-run path: in Step 5's 'no modifications' branch (line 140), check for unpushed commits (`git rev-list @{u}..HEAD --count`) and push them before Step 5.5; define `{sha}` for the no-new-commit case (line 165 says '{sha} is the short (7-char) commit SHA from Step 5', which is undefined when Step 5 made no commit this run — currently an agent must guess or fabricate one, and the cited commit may not exist on the remote).

**Verifier**: Confirmed at src/kit/skills/git-pr-review.md. Line 147 quote verified verbatim; the "no partial state" claim is repeated at line 180. Ground truth: bare `git reset` is a mixed reset to HEAD — it cannot undo a successful commit (push-failure case: no-op, local commit remains ahead of origin) and leav…

#### `a165` [NICE-TO-HAVE] Phase tracking never fires on the Phase-2 Copilot path: 'received' and 'reviewer' are defined only for a Phase-1 hit

**Location**: ### Phase Sub-State Tracking, table rows (lines 212-220) · **Category**: agent-ergonomics · **Found by**: git-pr-review

When a Copilot review arrives via Phase 2 polling (Step 2 → Step 3), no table row matches, so `received` is skipped and 'The `reviewer` field is set when reviews are detected' is ambiguous about whether Phase-2 arrival counts. Change the `received` trigger to 'Reviews detected (Step 2 Phase 1 hit, or Copilot review appears during the Phase 2 poll)' and state explicitly that `reviewer` is set at the same moment on either path. (Distinct from known-open f099, which covers the undefined <status_file>, section placement, and raw-yq mechanics.)

**Verifier**: Confirmed. src/kit/skills/git-pr-review.md:212 reads exactly "| `received` | Reviews detected (Step 2, Phase 1 hit) |", and on the Phase-2 Copilot path (poll succeeds at line 94 -> Step 3) no table row matches, so `received` is skipped and phase jumps from unset to `triaging`. Line 220's reviewer tr…

#### `a166` [NICE-TO-HAVE] Step 3 fetches node_id that nothing consumes — residue of an abandoned GraphQL thread-resolution design

**Location**: Step 3: Fetch Comments (line 105) · **Category**: duplication · **Found by**: git-pr-review

Replies are posted via REST using the numeric id (`-F in_reply_to={comment_id}`), the dedup fetch (line 158) doesn't request node_id, and the Step 3 note explicitly disclaims thread-resolution handling — node_id (a GraphQL identifier) has zero consumers in the skill, the kit, or the Go CLI. Drop `node_id: .node_id` from the jq projection and remove '(with id, node_id)' from SPEC-git-pr-review.md line 55, or add a comment stating what it is reserved for.

**Verifier**: Confirmed at stated severity (nice-to-have). Evidence quote verbatim at src/kit/skills/git-pr-review.md:105. node_id appears exactly twice in the repo: that line and docs/specs/skills/SPEC-git-pr-review.md:55 '(with id, node_id)'. Zero consumers anywhere: replies use REST numeric id (-F in_reply_to=…

### `src/kit/skills/git-pr.md`

#### `a167` [MUST-FIX] git-pr: detached HEAD passes the branch guard, then autonomously commits and emits a refspec-less push

**Location**: Step 2 Branch Guard (line 106) + Step 3b Push (lines 156-163); Step 1 (line 73) · **Category**: correctness · **Found by**: git-state-safety-sweep

Add an explicit detached-HEAD branch to Step 2: if `git branch --show-current` output is empty, STOP with "Cannot ship from a detached HEAD — check out or create a branch first (/git-branch)" before any staging or commit. Impact: a fully autonomous skill silently strands committed work and dies with a cryptic git error.

**Verifier**: CONFIRMED empirically, attempted to refute and could not. All citations verified in /home/sahil/code/sahil87/fab-kit.worktrees/woven-vole/src/kit/skills/git-pr.md: line 73 (`git branch --show-current` in Step 1), line 106 ("If the current branch is `main` or `master`, STOP immediately."), line 159 (…

#### `a168` [SHOULD-FIX] has_pr ignores PR state — a closed/merged PR on the branch short-circuits PR creation; `state` and `number` are fetched but never consulted

**Location**: Step 1: Gather State (lines 70–87) + Step 3 'If nothing to do' (line 127) + Step 3c gate (line 165) · **Category**: correctness · **Found by**: git-pr

`gh pr view` without a selector falls back to the most recently created PR for the branch even when it is MERGED or CLOSED, so `has_pr` becomes true and Step 3c is skipped — new commits on a branch whose previous PR was closed never get a PR, and the run can print 'already shipped' citing a dead PR URL. The skill already fetches `state` (and `number`) but no step reads either field. Branch on state: OPEN → existing-PR semantics as today; MERGED → keep the 'already shipped' stop (report '(merged)'); CLOSED → treat as no PR and proceed to Step 3c to create a fresh one. Drop the unused `number` field from the --json list. Mirror the rule in SPEC-git-pr.md.

**Verifier**: Confirmed empirically, not just by reading. (1) src/kit/skills/git-pr.md:77 fetches `gh pr view --json number,state,url` and line 86 derives has_pr; whole-file grep shows `state`/`number` are never read again — only `url` is consumed (lines 131, 223, 227). Gates at line 127 ("If nothing to do ... PR…

#### `a169` [SHOULD-FIX] git-pr: autonomous `git add -A` sweeps every untracked/unrelated file repo-wide into a pushed commit with no inspection step

**Location**: Step 3a Commit (lines 143-150); Rules (line 273) · **Category**: safety · **Found by**: git-state-safety-sweep

Scope staging to intentional paths (e.g., `git add -u` plus new files belonging to the change), or insert a guard that reviews `git status --porcelain` untracked entries and STOPs/excludes on suspicious paths before the autonomous commit. Impact: unrelated or sensitive files get irreversibly published to the remote by a no-questions-asked skill.

**Verifier**: CONFIRMED. (1) Evidence verbatim: src/kit/skills/git-pr.md:145 "1. Stage all changes: `git add -A`" (Step 3a) and :273 "Fully autonomous — never ask questions, never present options"; Step 3b/3c then push and open a PR in the same run. (2) Cross-reference verbatim: src/kit/skills/git-pr-review.md:14…

#### `a170` [NICE-TO-HAVE] Step 0a claims a start on an already-active ship stage 'is a no-op' — the CLI actually rejects it with a non-zero error, and already-active is the canonical path

**Location**: Step 0a: Start Ship Stage (line 35) · **Category**: staleness · **Found by**: git-pr

Ground truth (src/go/fab/internal/status/status.go:38, :64–83): `start` transitions only from `pending` (`{From: []string{"pending"}, To: "active"}`); from `active` it returns 'Cannot start stage … no valid transition' and exits non-zero — only the trailing `2>/dev/null || true` masks it. Since `fab status finish <change> hydrate` auto-activates ship, already-active is the normal case when /git-pr runs, so the error path is the common path. Reword to: 'If the stage is already active, the command exits non-zero (no valid transition) — harmless and suppressed by `|| true`; the stage state is unchanged.' This prevents an agent from copying the call elsewhere without the suppressor on the belief it succeeds idempotently.

**Verifier**: Confirmed end-to-end. (1) Quote verbatim at src/kit/skills/git-pr.md:35, following the suppressed call at :32. (2) Ground truth exact: src/go/fab/internal/status/status.go:38 start transitions only pending->active; ship has no override (overrides exist only for review/review-pr, :46-61); lookupTrans…

#### `a171` [NICE-TO-HAVE] git-pr: branch guard checks the literal names main/master, not the repo's actual default branch — on a develop/trunk-default repo the autonomous commit and push land directly on the default branch before PR creation fails

**Location**: Step 2 Branch Guard (lines 104-121) · **Category**: correctness · **Found by**: git-state-safety-sweep

Derive the actual default branch (`gh repo view --json defaultBranchRef -q .defaultBranchRef.name`, or `git symbolic-ref refs/remotes/origin/HEAD --short`, falling back to main/master) and guard against it in Step 2. Impact: the guard's stated protection silently does not apply to repos with non-main/master defaults, and the failure arrives only after an irreversible autonomous push.

**Verifier**: Verified verbatim at src/kit/skills/git-pr.md:106 (guard) and :111/:118 (messages). Pipeline order confirmed: Step 2 guard -> 3a git add -A/commit -> 3b git push -> 3c gh pr create, so on a develop/trunk-default repo the guard passes and the autonomous push lands on the default branch before gh pr c…

#### `a172` [NICE-TO-HAVE] Step 3c.4 failure branches contradict: 'PR creation fails → STOP' is listed before a silent --fill fallback whose trigger ('body generation fails') belongs to the previous sub-step

**Location**: Step 3c, sub-step 4 (lines 220–222) · **Category**: ergonomics · **Found by**: git-pr

Body assembly happens in sub-step 3, yet its failure handling lives under sub-step 4, after the STOP rule — and a body-induced `gh pr create` failure (e.g., malformed quoting) plausibly matches both branches, so an agent cannot tell whether to STOP or retry with --fill. Replace the two bullets with explicit ordering: 'If body assembly (sub-step 3) failed for any reason, run `gh pr create --draft --fill` instead (silent fallback). If the create command itself fails (either form), report the error and STOP.'

**Verifier**: Evidence verified verbatim at src/kit/skills/git-pr.md:220-222. Sub-step 3c.3 (body generation, lines 179-218) has no failure branch; its failure handling ('Fall back to gh pr create --draft --fill if body generation fails for any reason') sits under sub-step 4 after the 'If PR creation fails -> rep…

### `src/kit/skills/internal-retrospect.md`

#### `a173` [NICE-TO-HAVE] internal-retrospect is the only skill file with no H1 heading

**Location**: Top of file, lines 1-6 (frontmatter flows straight into body prose) · **Category**: convention · **Found by**: misc-internal

All 30 other src/kit/skills/*.md files open with an H1 (user-invocable skills use `# /skill-name [args]`); internal-retrospect has none, and internal-skill-optimize Rule 3 even mandates preserving '# /skill-name' as a structural invariant the optimizer must keep. Add `# /internal-retrospect` after the frontmatter; while there, consider normalizing the other two internal-* H1s ('# Internal Consistency Check', '# Internal Skill Optimize') to the `# /skill-name` form used by every other user-invocable skill. (Distinct from known f035, which covers only the missing _preamble-read line.)

**Verifier**: Confirmed end-to-end. (1) /home/sahil/code/sahil87/fab-kit.worktrees/woven-vole/src/kit/skills/internal-retrospect.md has no H1 anywhere; frontmatter (lines 1-4) flows straight into the body prose at line 6, where the evidence quote appears verbatim. (2) Enumerated all 31 files in src/kit/skills/: t…

### `src/kit/skills/internal-skill-optimize.md`

#### `a174` [SHOULD-FIX] internal-skill-optimize partial enumerations omit _pipeline — regression from the #393 twins refactor

**Location**: Arguments (line 15), Pre-flight step 1 (line 21), Constraints (line 86); same omission in docs/specs/skills/SPEC-internal-skill-optimize.md Summary (line 7) and Flow (line 15) · **Category**: staleness · **Found by**: misc-internal

PR #393 created the _pipeline helper (130 lines, well over the 80-line lean threshold) but all three enumerated partial lists in this skill and both lists in its SPEC mirror still name only six partials. Consequences: (a) Pre-flight never loads _pipeline as reference context, so optimizing fab-ff/fab-fff cannot apply the 'Redundant re-explanation' signal against the pipeline-bracket content #393 just single-sourced; (b) an agent following the parenthetical enumerations literally could treat _pipeline.md as a batch-mode target — exactly the f027 class of damage batch 2 fixed for _review/_cli-*. Root-cause fix: replace every enumeration with the glob rule ('every `_*.md` file in src/kit/skills/ is a shared partial — reference, never target') so future helpers are covered automatically; this list has now gone stale twice (f027, then _pipeline). Update the SPEC mirror in the same edit.

**Verifier**: CONFIRMED. /home/sahil/code/sahil87/fab-kit.worktrees/woven-vole/src/kit/skills/_pipeline.md (130 lines) was created by commit a8e720dd (PR #393), which updated _preamble.md:105 and docs/specs/skills.md:24 allowed-helpers lists to include _pipeline but never touched internal-skill-optimize.md (last…

#### `a175` [SHOULD-FIX] internal-skill-optimize's three partial enumerations omit _pipeline (created by #393)

**Location**: Arguments (line 15), Procedure step 1 (line 21), Constraints (line 86) · **Category**: staleness · **Found by**: lens-helper-integrity

Add `_pipeline` to all three enumerations (lines 15, 21, 86). Batch 3 (#392) added `_srad` to these lists when it was extracted, but batch 4 (#393) did not do the same for `_pipeline`. The line-21 omission is the operative one: the optimizer never loads the pipeline bracket as reference context, so when optimizing fab-ff/fab-fff it cannot recognize bracket content as 'fully defined in a partial' and may re-inline or mis-flag it; line 86's protected-list omission leaves `_pipeline.md` guarded only by the `_*.md` glob phrasing while the parenthetical reads as exhaustive. Note docs/specs/skills.md:24 and SPEC-_preamble.md:15 were correctly updated to 6 values including `_pipeline` — only this skill drifted.

**Verifier**: Confirmed at all three cited locations (src/kit/skills/internal-skill-optimize.md:15,21,86): each enumerates six partials omitting _pipeline. Ground truth verified: src/kit/skills/_pipeline.md exists, created by a8e720dd (#393), which touched zero lines of internal-skill-optimize.md; batch 3 (eca310…
