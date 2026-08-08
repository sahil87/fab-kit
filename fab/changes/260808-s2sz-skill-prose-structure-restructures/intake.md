# Intake: Skill Prose→Structure Restructures (Phase 3)

**Change**: 260808-s2sz-skill-prose-structure-restructures
**Created**: 2026-08-08

## Origin

> Phase 3 (prose to structure restructures) of fab/plans/sahil/skill-prose-consolidation.md: cover both sub-phase 3a (_preamble.md CLI-Adapter Dispatch section restructure to branch table plus 4-step procedure plus recovery-policy tables plus pane-mode bullets; Per-Stage Model Resolution restructure to seam table plus override table; delete re-derivations of watchable semantics, --alias rationale, override-binds-native, pane placement, fill precedence from _preamble.md since these now live in _cli-fab.md; and _cli-fab.md restructures for fab memory-index taxonomy tables, fab dispatch pane-prose trim, fab batch subcommand table, fab config provenance-to-spec-pointer, pane map --json field table, pr-meta Impact taxonomy, Confidence Scoring gate table) and sub-phase 3b (fab-operator.md strategic-prompt and frame-prose tables, fab-setup.md config-create-mode table, _intake.md header trim, _review.md dedup, _generation.md consumer table, _srad.md carve-out, _cli-agents.md caveat trim, git/docs family restructures per the plan). Note that Phase 1 and Phase 2 already shipped and merged to main, so re-verify every line reference against current HEAD before editing rather than trusting the plans baseline refs. Every constraint must survive; only the packaging changes. Preserve the plans protected set verbatim, including the runtime no-fence blockquote and frame example in fab-operator.md. Update matching docs/specs/skills/SPEC-*.md mirrors per the constitution, and sweep the aggregate specs (skills.md, glossary.md, architecture.md) for stale references to restructured sections.

One-shot invocation via `/fab-new`. The governing document is `fab/plans/sahil/skill-prose-consolidation.md` (committed at `1c702ef5`-era, § Phase 3), written from a 2026-08-08 six-sub-agent audit of all 36 files in `src/kit/skills/`. Phase 1 (drift fixes, change `ip4q`, PR #544) and Phase 2 (mechanical deletion, change `mcxv`, PR #547) are merged to main and present in this worktree's history — this change executes **Phase 3 only**, both sub-phases (3a + 3b) in a single change (the plan suggested two; the user directed one covering both).

## Why

1. **Token cost on every load**: the skill corpus writes multi-hundred-word paragraphs where a table or numbered list carries the same constraints in a fraction of the words. `_preamble.md` § CLI-Adapter Dispatch (~2,900w) and § Per-Stage Model Resolution (~1,640w, one 356-word sentence-paragraph) are loaded by *every* fab skill; `_cli-fab.md` § fab memory-index states its blocking/advisory taxonomy three separate times. Phase 3 targets ~7,000 words (net ~5,500 for 3a after Phase 2 overlap + ~3,500 for 3b).
2. **Drift risk from restated prose**: long prose re-derivations are where drift bugs breed — the plan documents PR #539 updating the canonical dispatch contract while every downstream restatement silently missed the reap step. Structure (tables with one row per case) makes omissions visible and diffs reviewable.
3. **If not done**: the corpus keeps paying ~20% context overhead per skill load, and the next contract change repeats the #539 drift pattern inside the prose walls.
4. **Why restructure rather than delete**: unlike Phase 2, this content is all load-bearing — every constraint must survive. Only the *packaging* changes (prose → tables/lists/pointers). The plan's rule: any instruction over ~80 words becomes a list or table.

## What Changes

All edits in `src/kit/skills/*.md` (canonical source — never `.claude/skills/`), each with its `docs/specs/skills/SPEC-*.md` mirror updated in the same change. **The plan's line refs are baseline (`6f74e0c3`/`b05ab998`) and are stale after Phases 1–2 merged; anchors below were re-verified against this worktree's HEAD (`da6f4f2b`), and apply MUST re-verify by section heading, not line number, before each edit.**

### 3a-1. `_preamble.md` § CLI-Adapter Dispatch (HEAD lines 325–379)

Restructure to:
1. A **2-row branch table** (`dispatch=` absent → native Agent-tool dispatch, two seams, default for built-in roles; `dispatch=` present → CLI adapter `fab dispatch`, no session-command fallback, per-stage mixing allowed) with a one-line watchable note: `dispatch.watchable` is a second *reason* for presence (tmux presence decides pane vs native), never a third branch.
2. The **4-step procedure** (start with prompt on stdin, no `--timeout` by default / wait `--timeout 300` push-not-poll, background preferred / handle+reap / no-state-cleanup-after-done) with five-state handling as a `state | meaning | action` table (`running` / `done` / `failed` / `failed (no-result)` / `orphaned`).
3. **Recovery policy** as a `state → automatic restart? → budget` table plus the (a/b/c) peek-classification table (visibly progressing → re-arm; parked at error → kill+restart within budget; awaiting human input → escalate without killing). One-restart-per-stage budget, tracked in orchestrator context only.
4. **Pane mode cut to ~4 bullets**: auto-mode default (tmux decides, forced flags' preconditions stay as pointer); 3-state subset (`failed`/`failed (no-result)` unreachable) + reap inheritance; peek via `fab pane capture` with the socket-included command printed by `fab dispatch logs`; steering is contract-neutral. **Placement mechanics** (split-window flags, column carve at `dispatch.column_width`, sibling detection via record pane-IDs, two-tier hierarchy) live in `_cli-fab.md` § fab dispatch only — delete the re-derivation here, keep a pointer.

Must survive (protected): the five state names byte-identical; "NO fallback to a session command"; reap-is-not-kill; "the pipeline NEVER sends keys" and its verb set (peek/kill/restart/notify/stop/reap); "orchestrator owns all transitions"; the `dispatched …` output-token forms incl. `auto:` suffixes; wait-timeout-exits-0 semantics; the `--timeout` start-vs-wait distinction. § Dispatch-Prompt Obligations (380–425: the three obligations, `{stage}-result.yaml` schemas, status-vs-verdict split, block-contract carve-out) is **not in scope** — untouched.

### 3a-2. `_preamble.md` § Per-Stage Model Resolution (HEAD lines 297–324)

Restructure to:
- A **2-row seam table**: model → Agent tool `model` param (alias enum, via `--alias`); effort → imperative prompt instruction (advisory on the native arm — session effort dominates, GitHub #64033/#39220 — binding only where `--effort` rides a composed CLI command). Both rows: empty ⇒ omit/inherit.
- A **3-row override table**: `override kind | binds where | executable path` — within-claude `--model`/`--effort` (native seam, executable); cross-provider `--provider` (native arm only, CANNOT move a stage onto CLI dispatch — `dispatch start` re-resolves from config); config override (`agent.workers`/`agent.session`/`agent.profiles.<role>.provider` — the sole executable cross-provider path). Kills the current 356-word paragraph.
- One compliance-visibility sentence (surface resolved `model=`/`effort=`/`provider=`/`dispatch=` lines; all-empty resolution is a flag-not-dispatch-blind signal) and one `--alias` sentence (emits the Agent-tool-valid short alias; operator launcher is the deliberate no-`--alias` exception).
- The role/knob resolution ladder prose → **pointer to `_cli-fab.md` § fab resolve-agent's fill-precedence** block.
- Delete `_preamble.md` re-derivations now owned by `_cli-fab.md`: watchable semantics (×3 across the two sections), `--alias` rationale (×4), override-binds-native (×3), pane placement, fill precedence.
- "Review is unexceptional" and residual "(today's behavior)"-class narration inside these sections is absorbed by the wholesale rewrite (2a leftovers — see Assumption 8).

Must survive: resolve-agent output line order (`model=` then optional `effort=`/`provider=`/`dispatch=`) + omit-when-empty rules; the operator-launcher `WithProfile` template-vs-append exception (may compress to a short note + `stage-models.md` pointer, but the fact survives); "no validation / verbatim pass-through"; the effort-asymmetry fact.

### 3a-3. `_cli-fab.md` § fab memory-index (HEAD lines 846–1244)

The blocking/advisory taxonomy is stated three times → **one** `severity | marker | fires when | affects --check exit` table + **one** 3-row tier table; delete the third restatement keeping exactly two footnotes: (1) exit-2 destructive-loss beats blocking, (2) the `--json` key stays `malformed`. FKF §6.4 freeze rationale (~200w) → pointer to `$(fab kit-path)/reference/fkf.md`. Protected: `--check` tier semantics and the blocking-vs-tier-2 precedence, exit codes, all marker literals.

### 3a-4. `_cli-fab.md` § fab dispatch (HEAD lines 519–744)

Keep the shape table and the mode-ladder table verbatim; cut the ~10 pane-prose bullets to ~4; `restart`'s 5-bullet diff-vs-`start` → one line + the two unique behaviors (mode re-derivation from current environment; relaunch from persisted `{stage}-prompt.md`). Protected: every fenced usage grammar, `dispatched …` output forms, the reap section's fires-only-on-done/no-file-removal contract.

### 3a-5. `_cli-fab.md` § fab batch (HEAD lines 1245–1277)

Prose → **subcommand × flags × guards table**; keep the archive consent matrix as-is.

### 3a-6. `_cli-fab.md` § fab config (HEAD lines 380–469)

Provenance/history rationale → pointer to `docs/specs/config.md` (already cited); keep modes, flags, exit codes.

### 3a-7. `_cli-fab.md` small items

- `fab pane map --json` 255-word cell (HEAD ~line 480–491) → field table.
- `fab pr-meta` Impact 262-word paragraph (HEAD 810–845) → 5-row taxonomy table, surfacing `raw = true + excluded` as a checkable row.
- Exit-code § Go-implementation notes (HEAD 48–55) trimmed; keep the code-2/3 overload notes verbatim.

### 3a-8. `_preamble.md` § Confidence Scoring (HEAD lines 432–460)

371 words for ~4 facts → **gate table** (one gate, intake, flat 3.0 all types, `--force` bypasses, data-only future divergence) + **invocation list** (who scores when; `/fab-continue` never scores post-intake).

### 3b. Remaining skills (per plan § 3b, all anchors re-verified at apply)

- **`fab-operator.md`** (720 lines): §5 strategic-prompt cluster → 4-row classification/action/notify/watchdog decision table + 3-line watchdog spec; §4 frame prose → `Element | Format | Notes` table — **keep the health-emoji table, the runtime no-fence blockquote, and the frame example verbatim** (protected); the long table cell (baseline L555) → 3-item numbered list; §1 principles → 1–2 sentences each + § pointers; spawn step 3 sub-bullets → code block + one guard line. Protected: the REQUIRED cross-repo caveat (baseline :485).
- **`fab-setup.md`** (423 lines): Config Create Mode → trigger bullets + `Field | Seeded by init | Your job` table; Arguments merged with Argument Classification into one table; Migrations "binary-owned" ×3 → one procedure + branch table; unify the three config-missing remediation strings into one. Protected: migrations output literals, constitution version-bump severity mapping, ecosystem→`test_paths` table + its anchoring counter-example.
- **`_intake.md`** (143 lines): header blockquote → keep the `{questioning-mode}` binding table + call-site bullet, drop design rationale; Step 7 → 3 lines; Step 8's verbatim `_srad` carve-out quote → pointer + the one operational instruction not in `_srad` (return deferred rows in the subagent result).
- **`_review.md`** (164 lines): single-review-agent fact ×4 → once in § Review Agent Dispatch; drop § Review Mode bullets that re-narrate the table above them; mode-gating ×4 → the two heading annotations only.
- **`_generation.md`** (234 lines): header blockquote → 4-column consumer table; traceability `<!-- R# -->` rule ×4 → once with the two format literals; Assumptions rule → pointer to `_srad` (×2). Protected: parser-contract heading literals, ID formats `T{NNN}`/`A-{NNN}`/`R#`/`<!-- R# -->`.
- **`_srad.md`** (131 lines): § Critical Rule 3 paragraphs → rule + carve-out (~130w); L73 prose (the remaining-three-skills paragraph) → table rows. **Keep the Worked Examples verbatim** (calibration anchors, protected). Protected: composite formula, half-open 80/50/20 thresholds and their rationale, marker literals, Assumptions table shape + footer strings.
- **`_cli-agents.md`** (181 lines): `--provider` caveat blockquote → 1 sentence + pointer; delete the `claude`-entry back-reference to § Peek; split the Dictionary Discipline closing paragraph (grew post-#540 — re-read at HEAD) into bullets.
- **`git-pr.md`** (391 lines): Step 3c `pr-meta` internals paragraph → 2 lines (output used verbatim); Key Properties `Idempotent?` 9-line cell → 1 line + pointers; Step 0b ladder → `Rung | Source | Condition | Result` table. Protected: the `✓ commit/push/pr/meta/status` literals (grep targets), `git add -u` + NOT-`-A` rationale.
- **`git-pr-review.md`** (290 lines): two-logins blockquote → 4-row table + the GraphQL-omits-bots trap line — **restructure only, never shorten the fact** (protected); Step 6/6.5 outcome classes → one table with a commit-status column. Protected: `Fixed —`/`Deferred —`/`Skipped —` prefixes, the synchronous-poll directive + its "(prior efforts stalled mid-poll…)" credibility anchor.
- **`docs-distill-memory.md`** (343 lines): Output § six blocks → two. Protected: approval-gate semantics (per-domain gate, per-file delete confirmation, yes/no/done tokens).
- **`docs-reorg-memory.md`** (333 lines): Key Properties cells capped at one clause + § pointer.
- **`fab-switch.md`** (125 lines): baseline :100 prose → two small tables.
- **`fab-status.md`** (78 lines): baseline :46,:53 → render-order list + glyph table + condition table.
- **`fab-dedupe.md`** (301 lines): rationale blocks (baseline 19, 89, 161, 192) → one clause each. Protected: "bulk approval deliberately NOT offered".

### Mirrors and aggregates (same change)

- Every touched `src/kit/skills/X.md` → matching `docs/specs/skills/SPEC-X.md` update (constitution, Additional Constraints; whole mirror class in scope per `fab/project/code-quality.md` § Sibling & Mirror Sweeps).
- Aggregate specs `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md` grepped for references to restructured/renamed section shapes and moved phrases; also `docs/specs/harness-adapters.md` and `docs/specs/stage-models.md` if they cite `_preamble` section wording.
- Memory files that quote restructured section names (see Affected Memory) swept for staleness — content changes expected nil since behavior is unchanged.

### Verification (from plan § Verification per phase)

1. `fab sync`, then spot-load deployed copies.
2. Diff the `##`/`###` heading sets before/after — restructures must not rename parser-contract headings; heading renames otherwise are allowed only with a full reference sweep.
3. Grep the protected-set literals before and after — byte-identical.
4. `ls docs/specs/skills/SPEC-*.md` against every touched skill; aggregate specs grepped for moved/renamed phrases.
5. Confirm each dispatch site still reaches the (restructured) canon via exactly one pointer.

## Affected Memory

Prose-only repackaging — no skill behavior changes, so no memory content changes are expected. The rows below are **staleness-sweep candidates** (files documenting the restructured skills that may quote section shapes or wording); hydrate should verify references and otherwise no-op.

- `runtime/dispatch.md`: (modify) verify references to `_preamble.md` § CLI-Adapter Dispatch / § Per-Stage Model Resolution wording survive the restructure; no behavioral content change
- `runtime/operator.md`: (modify) verify `fab-operator.md` §4/§5 references (frame format, strategic-prompt cluster) still match the restructured shape
- `runtime/providers-and-profiles.md`: (modify) verify resolve-agent / override-path wording references
- `pipeline/planning-skills.md`: (modify) verify `_intake.md`/`_srad.md`/`_generation.md` structural references
- `pipeline/execution-skills.md`: (modify) verify `_review.md` / dispatch-wiring references
- `_shared/context-loading.md`: (modify) verify `_preamble.md` section-name references (Contents list changed shape)

## Impact

- **Files**: ~20 skill sources under `src/kit/skills/` (`_preamble.md`, `_cli-fab.md`, `fab-operator.md`, `fab-setup.md`, `_intake.md`, `_review.md`, `_generation.md`, `_srad.md`, `_cli-agents.md`, `git-pr.md`, `git-pr-review.md`, `docs-distill-memory.md`, `docs-reorg-memory.md`, `fab-switch.md`, `fab-status.md`, `fab-dedupe.md`) + their SPEC mirrors + up to 5 aggregate/related specs + ~6 memory staleness sweeps.
- **No Go changes, no CLI-surface changes, no template changes, no migrations** — pure markdown repackaging. No test updates required (no `.go` files touched).
- **Size**: ~9,000 words removed across the corpus (net of Phase 2 overlap); the two highest-traffic files (`_preamble.md` loaded by every skill, `_cli-fab.md` by most) shrink the most.
- **Risk profile**: highest-review-care phase of the plan — the failure mode is silently dropping a constraint during prose→table conversion. Mitigated by the protected-set grep discipline and heading-set diffs (Verification above).

## Open Questions

*(none — the plan plus the user's directive resolve all decision points; see Assumptions)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One change covers both sub-phases 3a and 3b (plan recommended two changes) | User's directive explicitly says "cover both" — overrides the plan's suggestion | S:95 R:70 A:95 D:95 |
| 2 | Certain | Re-verify every anchor against HEAD by section heading, never trust plan baseline line refs | User-directed; verified stale already (CLI-Adapter Dispatch moved 333→325, memory-index 827→846) | S:95 R:90 A:100 D:100 |
| 3 | Certain | Zero contract loss — every constraint survives, only packaging changes; >~80-word instructions become lists/tables | The plan's governing rule for Phase 3, restated by the user | S:90 R:60 A:90 D:90 |
| 4 | Certain | Protected set preserved verbatim (plan § Must NOT be compressed), incl. fab-operator no-fence blockquote + frame example, SRAD Worked Examples, Copilot two-login fact (table ok, never shorten), result-yaml schemas, byte-stable output tokens | Explicit in both plan and user directive | S:95 R:50 A:90 D:95 |
| 5 | Certain | SPEC-*.md mirrors updated in the same change; aggregate specs (skills.md, glossary.md, architecture.md) swept | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps + user directive | S:95 R:80 A:100 D:100 |
| 6 | Certain | change_type = refactor (override the inferred `feat`) | Prose repackaging with zero behavior change; Phase 2 shipped as `refactor:` (#547) | S:80 R:90 A:95 D:90 |
| 7 | Confident | Parser-contract headings never renamed; other heading renames only with full reference sweep; heading-set diff is an acceptance check | Plan § Verification; external files reference these sections by name | S:75 R:60 A:85 D:80 |
| 8 | Confident | Phase-2 leftovers inside 3a sections (e.g. "Review is unexceptional", the Skill Invocation Protocol pointer, "(today's behavior)" narration — still present at HEAD) are absorbed by the wholesale section rewrites; apply works from actual HEAD state, not the plan's assumption of full 2a execution | Verified at HEAD that several 2a-listed items survived Phase 2; the 3a rewrite subsumes them | S:65 R:70 A:80 D:75 |
| 9 | Confident | Word-savings targets (~5,500 + ~3,500) are estimates, not acceptance gates; acceptance = zero contract loss + protected-set byte-identity + heading discipline + mirror sync | Plan quotes them as estimates ("post-verification estimates") | S:70 R:85 A:80 D:80 |
| 10 | Confident | Memory content changes expected nil; Affected Memory rows are staleness-sweep candidates only | Behavior unchanged ⇒ memory (post-implementation truth) unchanged; only section-name quotes could go stale | S:65 R:85 A:80 D:75 |
| 11 | Certain | Verification per plan: fab sync + spot-load, heading-set diff, protected-literal grep before/after, SPEC mirror ls-sweep, aggregate grep, one-pointer-per-dispatch-site check | Plan § Verification per phase, verbatim | S:80 R:90 A:90 D:85 |

11 assumptions (7 certain, 4 confident, 0 tentative, 0 unresolved).
