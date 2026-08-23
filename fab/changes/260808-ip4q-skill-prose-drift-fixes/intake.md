# Intake: Skill Prose Drift Fixes (Phase 1)

**Change**: 260808-ip4q-skill-prose-drift-fixes
**Created**: 2026-08-08

## Origin

One-shot `/fab-new` invocation picking up **Phase 1 (drift-bug fixes)** of the backlog detail doc `fab/plans/sahil/26-08-08-skill-prose-consolidation.md` (authored 2026-08-08 from a six-sub-agent audit of all 36 files in `src/kit/skills/`; merged via PR #542).

> Phase 1 (drift-bug fixes) of fab/plans/sahil/26-08-08-skill-prose-consolidation.md: fix the 10 defects in the Phase 1 table - dispatch-branch notes in fab-ff.md/fab-fff.md/fab-proceed.md; restore the --pane safety warning in fab-continue.md and fix wording; add missing dispatch invariants to fab-adopt.md; propagate the fab dispatch reap step to _pipeline.md/fab-continue.md/fab-adopt.md; fix fab-operator.md glyph violation at line 592; verify and align fab-setup.md symlink-repair claim; fix docs-hydrate-memory.md dev-repo-only path; delete dead /fab-rebalance-memory reference in docs-reorg-memory.md; add TOC to fab-dedupe.md; and verify/fix allowed-tools frontmatter on git-pr.md and git-pr-review.md. Update matching docs/specs/skills/SPEC-*.md mirrors per the constitution.

All 10 defects (plus the flagged-unconfirmed `allowed-tools` item) were **re-verified live at intake time against HEAD** (`7680ba34`, post-v2.17.0) — verification results per defect are recorded in What Changes below. The plan doc is the design authority; this intake transfers its Phase 1 table plus the intake-time verification state.

## Why

1. **Pain point**: the skill corpus restates canonical contracts instead of pointing at their owners, and the restatements have drifted from canon. The audit surfaced 10 live factual defects. The clearest one is self-demonstrating: PR #539 added `fab dispatch reap` to the canonical dispatch contract in `_preamble.md` § CLI-Adapter Dispatch — and none of the downstream restatements in `_pipeline.md`, `fab-continue.md`, or `fab-adopt.md` carry it, so any orchestrator driving from those files skips a mandatory step (pane workers leak their panes on `done`).
2. **Consequence if unfixed**: agents executing from drifted files take wrong actions — dispatch stages unconditionally native when `dispatch=` resolution says otherwise (fab-ff/fff/proceed), force `--pane` speculatively where canon says it is a hard error without tmux + `session_command` (fab-continue), skip reap (three files), render monochrome glyphs the operator's own frame rules ban (fab-operator), and follow a dev-repo-only path that does not exist in user repos (docs-hydrate-memory).
3. **Why this approach**: Phase 1 is deliberately the smallest, highest-value slice of the four-phase consolidation plan — pure factual corrections, no restructuring, each independently verifiable against the named canonical source. Shipping it first buys correctness before Phases 2–4 restructure the same text (the plan's dependency ordering: "ship first").

## What Changes

All edits go to the canonical sources `src/kit/skills/*.md` (never `.claude/skills/` — deployed copies, gitignored). Each skill edit carries its `docs/specs/skills/SPEC-*.md` mirror update in the same change (constitution, Additional Constraints). Line refs below are **HEAD-verified at intake time** (2026-08-08, `7680ba34`); re-confirm before each edit.

### 1. Dispatch-branch notes in `fab-ff.md` / `fab-fff.md` (defect #1)

**Verified at HEAD**: neither file contains the string `dispatch=` — both per-stage-model notes describe dispatch as unconditionally native (the two seams), omitting the branch that `_preamble.md` § CLI-Adapter Dispatch makes canonical.

**Fix**: rewrite both per-stage-model notes to state: resolve once with `fab resolve-agent <stage> --alias`, surface the resolved lines (compliance visibility), then **branch on `dispatch=` presence** per `_preamble.md` § CLI-Adapter Dispatch — absent ⇒ native two-seam dispatch, present ⇒ the CLI adapter. Keep the note pointer-style (Phase 2 collapses these further — do not grow them). Twin sweep: `fab-ff` ↔ `fab-fff` must stay consistent with each other.

### 2. Prefix-step dispatch bullets in `fab-proceed.md` (defect #2)

**Verified at HEAD**: lines 145, 152, 161, 169 all say "apply through the two seams" only — no instruction for when `fab resolve-agent fast --alias` (or `default --alias`) emits a `dispatch=` line. This is reachable: `agent.workers: codex` makes the `fast` role resolve a provider with a `dispatch_command`.

**Fix**: add the standard branch instruction (branch on `dispatch=` presence per `_preamble.md` § CLI-Adapter Dispatch) to the per-stage-model note at L145 and the three resolve bullets (L152/L161/L169). **Decision (per plan recommendation): follow the standard branch**, not a "prefix steps always dispatch native" carve-out — see Assumptions #2.

### 3. `--pane` safety warning + wording in `fab-continue.md` (defect #3)

**Verified at HEAD**: line 67 says "add `--pane` only to force a window from a dispatcher that is itself outside tmux" — (a) drops the canon safety warning (forcing `--pane` requires both a reachable tmux server and a `session_command` on the resolved provider, is a hard error without either, and must never be added speculatively — `_preamble.md` § CLI-Adapter Dispatch → Pane mode), and (b) says "force a window" where canon says "force a pane worker".

**Fix**: restore the warning **as a pointer to `_preamble.md`, not a restatement** (Phase 2 shrinks this line anyway), and fix "window" → "pane worker". The same shorthand blockquote at L82/L84 should be checked for the same wording drift.

### 4. Missing dispatch invariants in `fab-adopt.md` (defect #4)

**Verified at HEAD**: `fab-adopt.md`'s dispatch restatement (§ around L98) carries neither "NO fallback to a session command" nor "no STATE cleanup after `done`" (post-#539 wording where reap is pane hygiene, not state cleanup).

**Fix**: add both invariants as **one pointer clause** referencing `_preamble.md` § CLI-Adapter Dispatch — not a restatement.

### 5. Propagate `fab dispatch reap` to the three dispatch restatements (defect #5)

**Verified at HEAD**: `grep -l 'dispatch reap' src/kit/skills/_pipeline.md src/kit/skills/fab-continue.md src/kit/skills/fab-adopt.md` → no matches. Canon (`_preamble.md` § CLI-Adapter Dispatch step 3, post-#539): on `done`, read `{stage}-result.yaml`, then run **`fab dispatch reap <change> <stage>`** unconditionally (no mode check, no config check — the whole guard lives in Go), then proceed with the sequencer transition.

**Fix**: since Phase 1 ships standalone (before Phase 2's collapse-to-pointer), add the reap step to each of the three restatements now, minimally — one clause per site on the `done` path, keeping the canonical properties (unconditional, dumb, reap-is-not-kill) as a pointer rather than re-deriving them. See Assumptions #3.

### 6. Glyph violation in `fab-operator.md:592` (defect #6)

**Verified at HEAD**: L592 reads "entries with `▶` that show ✓ (green) are complete, the one showing ● (green) / ◌ (yellow) is current" — while L301 bans exactly those geometric glyphs in frames ("geometric glyphs like `●◌✗` render monochrome and are NOT used").

**Fix**: rewrite L592's queue-progress description to the health-emoji + `▶` convention that L301 and the §4 frame rules define.

### 7. `fab-setup.md` symlink-repair claim (defect #7)

**Verified at HEAD**: L401 says "Symlinks are verified/repaired every run" under Idempotency, while L106 describes copy-based deployment. One of the two misstates the real `fab sync` mechanism.

**Fix**: verify against the actual `fab sync` implementation in `src/go/` first, then align **both** lines (L106 and L401) to the true mechanism. Direction of the fix is determined by the code, not chosen in prose — see Assumptions #4.

### 8. Dev-repo-only path in `docs-hydrate-memory.md:192` (defect #8)

**Verified at HEAD**: L192 points at "the dev-repo design doc `docs/specs/fkf.md` §10 item 2" — a path that exists only in the fab-kit dev repo, not in user repos where the deployed skill runs (the deployed-skill leak class that `docs-distill-memory.md` L33 warns about).

**Fix**: point at `$(fab kit-path)/reference/fkf.md` instead, and drop the migration-trajectory narration around it (the "removing the 20 existing per-file changelogs is a separate change" aside).

### 9. Dead `/fab-rebalance-memory` reference in `docs-reorg-memory.md:320` (defect #9)

**Verified at HEAD**: the Key Properties row reads "Yes — supersedes any separate `/fab-rebalance-memory`; …". No such skill exists.

**Fix**: delete the clause (keep the rest of the row's content).

### 10. Missing TOC in `fab-dedupe.md` (defect #10)

**Verified at HEAD**: 295 lines, no `## Contents` section — the only >100-line skill violating `internal-skill-optimize`'s own TOC rule.

**Fix**: add the `## Contents` TOC listing the file's `##` sections, matching the format used by other skills (e.g., `fab-new.md`).

### 11. `allowed-tools` frontmatter on `git-pr.md` / `git-pr-review.md` (flagged, verify-then-fix)

**Verified at HEAD**: `git-pr.md` declares `allowed-tools: Bash(git:*), Bash(gh:*)`; `git-pr-review.md` declares `allowed-tools: Bash(git:*), Bash(gh:*), Bash(command:*)`. Both skills' step bodies instruct Read (and git-pr-review instructs file edits during fix application) — if `allowed-tools` is restrictive in this harness, those steps cannot execute.

**Fix**: determine whether `allowed-tools` is restrictive for Claude Code skills (check current Claude Code documentation/behavior). If restrictive → widen the frontmatter to cover the tools the steps actually use. If advisory-only → leave frontmatter as-is and record a one-line note in the two SPEC mirrors. See Assumptions #5.

### 12. SPEC mirror updates (constitution requirement)

Every touched skill gets its mirror updated in the same change: `SPEC-fab-ff.md`, `SPEC-fab-fff.md`, `SPEC-fab-proceed.md`, `SPEC-fab-continue.md`, `SPEC-fab-adopt.md`, `SPEC-_pipeline.md`, `SPEC-fab-operator.md`, `SPEC-fab-setup.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-docs-reorg-memory.md`, `SPEC-fab-dedupe.md`, `SPEC-git-pr.md`, `SPEC-git-pr-review.md` (all exist in `docs/specs/skills/`). Treat the whole mirror class as in-scope, not just files carrying the literal changed phrase (`fab/project/code-review.md` § Project-Specific Review Rules). Also grep aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) for any moved/renamed phrases.

### Execution constraints (from the plan doc, binding on apply)

- **Prose-only**: no behavioral change to any skill; every fix aligns prose to an already-canonical contract.
- **Protected set** (plan § Must NOT be compressed): command grammars, byte-stable output tokens, exit-code tables, `{stage}-result.yaml` schemas, reap-is-not-kill, anti-drift prohibitions — none of these may be shortened while editing around them.
- **Pointer over restatement**: where a fix adds text (defects #3, #4, #5), add the minimal pointer to the canonical `_preamble.md` section rather than a fresh restatement — Phase 2 collapses restatements, so new prose added here should already be pointer-shaped.
- **Verification** (plan § Verification per phase): run `fab sync` and spot-load deployed copies; grep protected-set literals before/after (byte-identical); confirm the reap step is reachable from every dispatch site; sweep SPEC mirrors against every touched skill.

## Affected Memory

None. Phase 1 aligns skill prose to contracts that canon (`_preamble.md`, `_cli-fab.md`) and memory already document correctly — no spec-level behavior changes. The plan's Non-goals explicitly exclude `docs/memory/` changes beyond the constitution-required SPEC mirrors (which are specs, not memory).

## Impact

- **Files**: ~13 skill sources under `src/kit/skills/` (fab-ff, fab-fff, fab-proceed, fab-continue, fab-adopt, _pipeline, fab-operator, fab-setup, docs-hydrate-memory, docs-reorg-memory, fab-dedupe, git-pr, git-pr-review — the last two possibly frontmatter-only or SPEC-note-only) + up to 13 SPEC mirrors under `docs/specs/skills/` + possible aggregate-spec touch-ups.
- **Code**: none (no Go changes; defect #7 *reads* `src/go/` to verify the sync mechanism but changes only prose).
- **Tests**: none required (no `.go` changes). Markdown-only change.
- **Risk**: low — each edit is small, independently verifiable against its named canonical source, and behavior-neutral by design.

## Open Questions

None — the plan doc resolves or prescribes a resolution path for every item (the two "decide at apply" items — sync mechanism direction and allowed-tools semantics — are verification-driven, not preference-driven).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is exactly Phase 1 of the plan doc (10 defects + the allowed-tools verification + SPEC mirrors); Phases 2–4 out of scope | User's invocation names Phase 1 explicitly; plan doc defines each phase as independently shippable | S:95 R:90 A:95 D:95 |
| 2 | Confident | Defect #2 (fab-proceed): add the standard `dispatch=` branch to prefix-step bullets, not an "always dispatch native" carve-out | Plan offers both, audit recommends the standard branch; fab-continue.md L67 already establishes the pattern; easily revised via rework | S:70 R:80 A:80 D:65 |
| 3 | Certain | Defect #5: add the reap step to all three restatements now (not defer to Phase 2's collapse) | Plan's own conditional: "If phase 1 ships standalone first, add the reap step to each restatement" — Phase 1 is shipping standalone first by design | S:90 R:85 A:95 D:90 |
| 4 | Certain | Defect #7: direction of the fix (symlink vs copy wording) is determined by reading `fab sync` in `src/go/` at apply, then both L106 and L401 align to it | Plan prescribes verify-then-align; the code is the ground truth and is available to the apply agent | S:75 R:85 A:85 D:70 |
| 5 | Confident | Item #11: if `allowed-tools` is restrictive, widen frontmatter; if advisory, leave it and add a one-line SPEC-mirror note | Plan prescribes exactly this verification-driven branch; both outcomes are small and reversible | S:70 R:85 A:70 D:65 |
| 6 | Certain | New text added by fixes #3/#4/#5 is pointer-shaped (references `_preamble.md` § CLI-Adapter Dispatch), not restatement | Plan states this per-defect ("as a pointer, not a restatement", "one pointer clause"); Phase 2's collapse depends on it | S:90 R:90 A:90 D:90 |
| 7 | Certain | No `docs/memory/` updates in this change | Plan Non-goals exclude memory changes beyond SPEC mirrors; fixes are behavior-neutral alignments to already-documented canon | S:85 R:90 A:90 D:90 |

7 assumptions (5 certain, 2 confident, 0 tentative, 0 unresolved).
