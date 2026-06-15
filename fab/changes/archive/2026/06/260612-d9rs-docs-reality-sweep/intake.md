# Intake: Docs Reality Sweep

**Change**: 260612-d9rs-docs-reality-sweep
**Created**: 2026-06-12

## Origin

`/fab-new d9rs` (one-shot, backlog-ID input). Backlog entry `[d9rs]` from `fab/backlog.md`, verbatim:

> Skills-audit batch 5/5 — docs reality sweep + quick-win bundles. DEPENDS: LAST of the five — mirrors/docs must describe post-batch reality. GOAL: the documentation layer describes the shipped system; burn down the quick wins. ACTIONS (report §2 Themes 7+8, §3): SPEC-HOOKS rewrite as-shipped (3 must-fix: Current Hooks lists two deleted shell scripts; "Proposed Hook Architecture" presents shipped Go behavior as future work; events table rates UserPromptSubmit "No" while that hook is registered; plus superseded fab runtime proposal, dead yq inventory, stale phase list, outdated runtime schema). LEGACY DOCS truth sweep (extends the deferred uliv doc-residue list): assembly-line.md:121 spec/tasks narrative; architecture.md pre-binary .kit/ distribution model + Router Dispatch self-contradiction (consider rewrite vs retire); overview.md 4-stage story omitting /git-pr /git-pr-review /fab-proceed /fab-operator; glossary.md:49/115 auto-clarify fab-ff disclaims; user-flow.md:84/183 "failed is review-only"; templates.md pre-1.10.0 intake template + .status.yaml block missing ready/id/issues/prs; skills.md:583 pre-date-bucketing archive path; operator.md:11 v8→v9. SPEC MIRROR resync (~18 drifted, enumerated in Theme 7c — e.g. SPEC-fab-operator status-only-mode + rejected Decision 2 recorded as current, SPEC-_preamble misquoted opening instruction + dead kit.conf row, SPEC-fab-continue writes a removed "Spec" artifact + claims forbidden fab score use, SPEC-fab-archive #393 f087 regression, SPEC-fab-clarify removed [target-artifact] flow, SPEC-fab-proceed _preamble self-contradiction). MEMORY-INDEX ownership (Theme 8): docs-reorg-memory.md:125-126 Step 5.3 edits a sub-domain index that doesn't exist until Step 5.4 generates it AND 5.4 forbids the edit (must-fix; reuse the stub pattern at docs-hydrate-memory.md:69); define ownership once — description: frontmatter is the single hand-curated field, stub created BEFORE fab memory-index — propagate to hydrate generate-mode (no placement rules today), docs-hydrate-specs no-target-spec branch + phantom skip token, docs-reorg depth off-by-one + dangling-link abort escape, docs-reorg-specs reserved-path exemption for constitution-pinned SPEC mirrors. QUICK-WIN Bundle A stale pointers (§3 has the full list): fab-operator.md:23 §5→§6, :226-252 "7 tracked" vs 8 entries + gmail-deploys schema-invalid source, :192 stale retention clause; fab-proceed.md:109 "(see Output Format)"→Bypass Notes; fab-continue.md:149 dangling "Review Behavior" heading; git-pr-review.md:105 unconsumed node_id + :11/64 --tool header; ~12 more one-liners. Bundle B enumeration completion: _preamble.md:36 exceptions list (+/fab-proceed /fab-help /fab-archive /docs-hydrate-specs /docs-reorg-*), :301 orchestrator list omits /fab-proceed, :355-361 scoring invokers omit /fab-draft; internal-skill-optimize.md:15/21/86 omits _pipeline (stale twice now); _generation.md:3/11-13 fab-continue in both consumer groups; _cli-fab.md:27 index omits migrations-status/memory-index; Next: lines omit /fab-proceed (fab-draft:30/48, fab-new:223, fab-setup:432-436). Root-cause: prefer derivation rules over enumerations ("every _*.md file is a shared partial"; "derive Next: per the Lookup Procedure") so these cannot drift a third time. HYGIENE: archive the four merged-but-unarchived 2606xx change folders. CONSTRAINTS: prose-only, no behavior changes; SPEC mirror per touched skill; src/kit is canonical. REPORT: docs/specs/findings/skills-review-2026-06-12.md §2 Themes 7+8 + §3.

Source report: `docs/specs/findings/skills-review-2026-06-12.md` — §2 Theme 7 (lines 120–131), Theme 8 (lines 133–143), §3 Quick wins (lines 147–168). §5 "Verified clean" (lines 186–197) enumerates areas swept and confirmed sound — those are explicitly **out of scope** (do not "fix" them).

## Why

1. **The pain point**: The documentation layer — `docs/specs/` legacy docs, the constitution-mandated `docs/specs/skills/SPEC-*.md` mirrors, and stale prose inside `src/kit/skills/` — describes one or more architecture generations behind the shipped system: a pre-binary `.kit/` distribution model, a 4-stage pipeline story, deleted shell hooks presented as current, removed flows (`[target-artifact]`, spec.md artifact) presented as live, and ~18 drifted SPEC mirrors that violate the constitution's "skill change ⇒ SPEC update" constraint.
2. **The consequence if unfixed**: Agents and humans consulting specs per constitution Principle VI get answers that contradict the shipped system. Enumerations keep rotting — `internal-skill-optimize.md`'s `_pipeline` omission has now gone stale **twice**, proving point-fixes don't hold. Batches 1–4 of this audit (PRs #390–#404) changed behavior; this batch is sequenced LAST precisely so the docs can describe post-batch reality — deferring it reopens the gap immediately.
3. **Why this approach**: Three motions, all enumerated by the audit with file:line precision: (a) rewrite-as-shipped for the fiction-heavy docs (SPEC-hooks, legacy docs), (b) mechanical mirror resync driven by the constitution's skill→SPEC rule, (c) burn-down of two quick-win bundles. Root-cause guard: replace enumerations with derivation rules wherever a rule can state the invariant, so the same lists cannot drift a third time.

## What Changes

All edits are markdown-only. Canonical sources: `docs/specs/**` for docs/mirrors, `src/kit/skills/*.md` for skill prose (never `.claude/skills/` — deployed copies, gitignored). Per the w7dp lesson: when a stale claim is fixed, sweep the same claim-class repo-wide rather than fixing only the cited line.

### 1. SPEC-hooks rewrite as-shipped

`docs/specs/skills/SPEC-hooks.md` is fiction end-to-end; rewrite to describe the shipped Go hook system:

- **Must-fix (3)**: Current Hooks section lists two deleted shell scripts; "Proposed Hook Architecture" presents already-shipped Go behavior as future work; the events table rates UserPromptSubmit "No" while that hook is registered.
- **Should-fix (4)**: superseded `fab runtime` proposal, dead yq inventory, stale phase list, outdated runtime schema.

### 2. Legacy docs truth sweep (extends the deferred uliv doc-residue list)

- `docs/specs/assembly-line.md:121` — spec/tasks stage narrative (must-fix; pipeline is 6 stages, no spec stage since 1.10.0).
- `docs/specs/architecture.md` — built on the pre-binary `.kit/` distribution model and self-contradicts in its Router Dispatch section (structural). Rewrite as-shipped, keeping the file and its specs-index row. <!-- clarified: rewrite-in-place over retire — user confirmed in clarify session 2026-06-12 -->
- `docs/specs/overview.md` — 4-stage story; omits `/git-pr`, `/git-pr-review`, `/fab-proceed`, `/fab-operator`.
- `docs/specs/glossary.md:49/115` — defines auto-clarify behavior that fab-ff disclaims.
- `docs/specs/user-flow.md:84/183` — says `failed` is "review only" (review-pr also fails since w7dp).
- `docs/specs/templates.md` — carries the pre-1.10.0 intake template; its `.status.yaml` block is missing `ready`/`id`/`issues`/`prs`.
- `docs/specs/skills.md:583` — pre-date-bucketing archive path.
- `docs/specs/operator.md:11` — "current operator (v8)" above its own v9 row.

### 3. SPEC mirror resync (~18 drifted mirrors, Theme 7c)

Mechanical resync of `docs/specs/skills/SPEC-*.md` against their `src/kit/skills/` sources, per the constitution's skill→SPEC rule. Enumerated drift (fix the named items, then diff each mirror against its source for same-class residue):

- SPEC-fab-operator — status-only-mode + rejected Decision 2 recorded as current
- SPEC-_preamble — misquoted opening instruction (×2), dead kit.conf row (×2)
- SPEC-fab-proceed — self-contradiction on _preamble loading
- SPEC-fab-clarify — removed `[target-artifact]` flow; `fab score` missing `--stage`
- SPEC-fab-continue — writes a removed "Spec" artifact; claims forbidden `fab score` use
- SPEC-fab-archive — preflight/hydrate guard applied to both modes (#393 f087 regression)
- SPEC-git-pr, SPEC-git-pr-review (×2), SPEC-fab-status, SPEC-fab-help, SPEC-fab-new, SPEC-docs-hydrate-specs (phantom modify/index paths), SPEC-docs-hydrate-memory (three-way exemption contradiction), SPEC-_review (spec/plan phrasing), SPEC-docs-reorg-memory (wrong Kind tokens), SPEC-fab-discuss

### 4. Memory-index ownership (Theme 8)

Define the index-ownership model **once** and propagate: `description:` frontmatter is the single hand-curated field; the sub-domain index **stub is created BEFORE `fab memory-index` runs** (reuse the stub pattern at `docs-hydrate-memory.md:69`).

- `src/kit/skills/docs-reorg-memory.md:125-126` — **must-fix**: Step 5.3 instructs editing a sub-domain index.md that doesn't exist until Step 5.4 generates it, and Step 5.4 forbids the edit. Resolve via the stub-before-index pattern.
- `src/kit/skills/docs-hydrate-memory.md:124-149` — generate mode has no placement rules (target path, domain creation, index stub, shape bounds live only under ingest); `:69/81-83` — sub-domain index stubs never instructed; memory-index tier description omits sub-domain indexes in 3 locations (incl. `fab-continue.md:183`).
- `src/kit/skills/docs-hydrate-specs.md:64` — add the missing branch for a gap with no suitable target spec file; Step 6 handles a "skip rest" token Step 5 never offers (align them).
- `src/kit/skills/docs-reorg-memory.md:23/56/78-84` — depth off-by-one between the ≤3 bound (path segments) and the report's folder-depth column; dangling-link hard block needs an abort/rollback escape.
- `src/kit/skills/docs-reorg-specs.md:12-35` — add a reserved-path exemption for the constitution-pinned SPEC mirrors; define recursion into subfolders.

These are documentation-of-correct-behavior fixes within skill markdown (resolving self-contradictions and missing branches) — no Go/binary changes.

### 5. Quick-win Bundle A — stale pointers, counts, wording (§3 full list)

- `fab-operator.md:23` — "(see §5)" → "(see §6)"; `:226-252` — status-frame example "7 tracked" vs 8 entries + `gmail-deploys` watch has no schema-valid source; `:192` — drop the stale "until the operator session ends" branch_map retention clause.
- `fab-proceed.md:109` — "(see Output Format)" → "(see Bypass Notes)"; `:74/108` — `sort -r` on full folder names for the same-day tiebreak.
- `fab-continue.md:149` — "Review Behavior" → a heading that exists in `_review.md` (the one dangling heading pointer corpus-wide).
- `git-pr-review.md:105` — drop the unconsumed `node_id` from the jq projection (+ SPEC line 55); `:11/64` — reword the `--tool` header ("bypasses automatic detection" is false; "the cascade" is undefined residue).
- `docs-hydrate-specs.md:70/76` — align the yes/no/done prompt with the four-token handler.
- `fab-help.md:12` — Purpose understates the output (git-*, docs-*, batch, packages); `internal-retrospect.md` — add the missing H1; `_cli-external.md:34` — `--reuse` requires `--worktree-name`; `_generation.md:17` — drop "auto-clarify"; `fab-clarify.md:182` — protocol example cites a removed flow; `fab-discuss.md:33` — state the stage-derivation rule for `.status.yaml`; `fab-switch.md:29-33` — add the missing "run the switch after selection" step; `fab-status.md:78` — preflight does require config/constitution to exist; `git-pr.md:220-222` — reorder the `--fill` fallback vs STOP branches; `_cli-fab.md:175-178` — document artifact-write's git auto-staging.

### 6. Quick-win Bundle B — enumeration completion + derivation rules

- `_preamble.md:36` — Always-Load exceptions list misses `/fab-proceed`, `/fab-help`, `/fab-archive`, `/docs-hydrate-specs`, `/docs-reorg-*`; give `docs-hydrate-memory` the Context Loading section the rule keys on.
- `_preamble.md:301` — Subagent Dispatch orchestrator list omits `/fab-proceed`; `:355-361` — Confidence Scoring invokers omit `/fab-draft` and mis-scope clarify's recompute.
- `internal-skill-optimize.md:15/21/86` (+ SPEC) — all partial enumerations omit `_pipeline` (stale twice now).
- `_generation.md:3/11-13` (+ SPEC:5) — `fab-continue` belongs to both consumer groups.
- `_cli-fab.md:27` — in-file index omits `migrations-status`/`memory-index`; `fab-draft.md:30/48`, `fab-new.md:223`, `fab-setup.md:432-436` — `Next:` lines omit `/fab-proceed`; `_srad.md:51-57` — one-line note for fab-draft/fab-clarify.
- **Root-cause guard**: where a rule can state the invariant, replace the enumeration with a derivation rule — e.g., "every `_*.md` file is a shared partial — reference, never target"; "derive `Next:` at runtime per the _preamble Lookup Procedure" — so these lists cannot drift a third time. Keep enumerations only where no rule captures the membership.

### 7. Hygiene

- Archive the merged-but-unarchived change folders (audit counted four at write-time; **recount at apply** — determine merge state from each folder's `.status.yaml` `prs:` entries via `gh pr view --json state,mergedAt`; archive only actually-merged changes via the `/fab-archive` mechanics: move to `fab/changes/archive/`, update the archive index, check the matching backlog box).
- Check the `9u91`/`uliv`/`zc9m`/`szxd` backlog boxes (again: only where the corresponding PR is merged).

### Constraints (from the backlog entry)

- **Prose-only, no behavior changes** — markdown edits only; no Go/binary changes; skill edits resolve documented self-contradictions and missing branches, they do not introduce new runtime mechanisms.
- **SPEC mirror per touched skill** — every `src/kit/skills/*.md` touched in bundles A/B/Theme-8 gets its `docs/specs/skills/SPEC-*.md` mirror updated in the same change (constitution constraint).
- **`src/kit/` is canonical** — never edit `.claude/skills/` directly; deployed copies regenerate via `fab sync`. Use Read/Edit tools, not `sed` (sed bypasses the PostToolUse hooks — ye8r lesson).
- **§5 Verified-clean areas are out of scope** — e.g., the `_pipeline` bracket extraction, prior batch fixes (f019/f051/f062), batch-2 naming cleanup, migration logic.

## Affected Memory

- `memory-docs/hydrate`: (modify) index-ownership model (description: frontmatter = single hand-curated field; stub before `fab memory-index`), generate-mode placement rules, sub-domain index stubs
- `memory-docs/hydrate-generate`: (modify) placement rules now defined for generate mode (target path, domain creation, index stub, shape bounds)
- `memory-docs/hydrate-specs`: (modify) no-target-spec branch; prompt/handler token alignment
- `memory-docs/specs-index`: (modify) SPEC-mirror resync outcome; reserved-path exemption for constitution-pinned mirrors in reorg-specs
- `_shared/context-loading`: (modify) corrected Always-Load exceptions (or their replacement by a derivation rule), orchestrator/scoring-invoker list completion
- `pipeline/planning-skills`: (modify) `Next:` lines derived per the Lookup Procedure rather than enumerated; `_generation` consumer-group correction

(Hydrate may additionally create a small memory file for the reorg-skill bounds/escape semantics if the existing files don't absorb them cleanly — decided at hydrate.)

## Impact

- **docs/specs/**: ~8 legacy docs (assembly-line, architecture, overview, glossary, user-flow, templates, skills, operator) + `skills/SPEC-hooks.md` + ~18 `skills/SPEC-*.md` mirrors.
- **src/kit/skills/**: ~20 skill/partial files across bundles A/B and Theme 8 (fab-operator, fab-proceed, fab-continue, git-pr-review, git-pr, docs-hydrate-specs, docs-hydrate-memory, docs-reorg-memory, docs-reorg-specs, fab-help, internal-retrospect, internal-skill-optimize, _cli-external, _cli-fab, _generation, _srad, _preamble, fab-clarify, fab-discuss, fab-switch, fab-status, fab-draft, fab-new, fab-setup).
- **fab/**: backlog checkbox updates; archive moves for merged change folders + archive index.
- **Not touched**: Go sources (`src/*.go`), templates under `src/kit/templates/`, tests, runtime behavior. `true_impact_exclude` covers `fab/` and `docs/` — true-impact surface is the `src/kit/skills/` prose only.
- **Review surface**: config `checklist.extra_categories` includes `documentation_accuracy` and `cross_references` — both directly exercised by this change.

## Open Questions

None — the backlog entry and report §2/§3 fully enumerate the work; the remaining judgment calls are graded below.

## Clarifications

### Session 2026-06-12

| Q | Question | Answer |
|---|----------|--------|
| 1 | architecture.md: rewrite as-shipped in place, or retire and fold into overview.md? | Rewrite in place — keep the file and its specs-index row; replace pre-binary content with the shipped distribution model |

### Session 2026-06-12 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 3 | Confirmed | — |
| 4 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly the d9rs backlog enumeration (report §2 Themes 7+8 + §3 bundles + hygiene); report §5 Verified-clean areas and §4 Structural bets are out of scope | Backlog ACTIONS/CONSTRAINTS + report sectioning are explicit; structural bets are flagged "worth a design discussion, not a drive-by fix" | S:95 R:90 A:95 D:95 |
| 2 | Certain | Markdown-only edits with `src/kit/` as canonical source; every touched skill gets its SPEC mirror updated in the same change | Constitution constraints (skill→SPEC rule, src/kit canonical) + backlog CONSTRAINTS state this verbatim | S:95 R:90 A:95 D:95 |
| 3 | Confident | "Prose-only, no behavior changes" means no Go/runtime changes; the Theme-8 normative additions (stub-before-index rule, no-target branch, reserved-path exemption, dangling-link abort escape) are in scope | Clarified — user confirmed | S:95 R:70 A:80 D:70 |
| 4 | Confident | Ship as a single bundled change, not the three separate changes the report's Theme 7 Action suggests | Clarified — user confirmed | S:95 R:65 A:85 D:85 |
| 5 | Certain | `architecture.md`: rewrite as-shipped in place, rather than retire | Clarified — user chose rewrite-in-place over retire | S:95 R:75 A:55 D:40 |
| 6 | Confident | Apply derivation rules over enumerations only where the rule states the invariant without changing behavior; keep enumerations elsewhere | Clarified — user confirmed | S:95 R:80 A:80 D:75 |
| 7 | Confident | Hygiene archives only changes whose PRs are actually merged at apply time (recount via `gh pr view`), treating the audit-time count of "four" as indicative, not binding | Clarified — user confirmed | S:95 R:85 A:80 D:75 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
