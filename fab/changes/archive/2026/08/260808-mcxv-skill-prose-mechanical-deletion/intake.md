# Intake: Skill Prose Mechanical Deletion (Phase 2)

**Change**: 260808-mcxv-skill-prose-mechanical-deletion
**Created**: 2026-08-08

## Origin

One-shot `/fab-new` invocation with a detailed directive. Raw input:

> Phase 2 (mechanical deletion: narration + duplication) of fab/plans/sahil/26-08-08-skill-prose-consolidation.md: cover both sub-phase 2a (historical/transition narration sweep, ~2,700 words - delete or rewrite-as-present-truth per the plans known-sites list) and sub-phase 2b (duplication collapse, ~3,300 words - dispatch-seam restatements to pointers, fab log command caveat consolidation, command -v gate discipline merge, fab-operator internal duplication cleanup, fab-ff/fab-fff shared framing extraction to _pipeline.md, docs family triples consolidation, small boilerplate dedup). Follow the sequencing note: do the dispatch-seam collapse in 2b against current canon (do not restructure _preamble.md itself, that is phase 3). Preserve everything in the plans Must NOT be compressed protected set verbatim. Update matching docs/specs/skills/SPEC-*.md mirrors per the constitution.

**Authoritative source document**: `fab/plans/sahil/26-08-08-skill-prose-consolidation.md` — read it in full at apply entry. This intake condenses its Phase 2 section; the plan carries the complete per-site lists, the § Execution constraints, the § Must NOT be compressed protected set, and the § Verification per phase checklist. The plan is the four-phase backlog doc from the 2026-08-08 `/fab-discuss` audit of all 36 files in `src/kit/skills/` (~108k words). Phase 1 (the 10 drift-bug fixes) already shipped as change `260808-ip4q` / PR #544 (merged). This change is Phase 2 only; phases 3 (prose→structure restructures) and 4 (policy + extraction) are out of scope.

Key decisions from the invocation: one fab change covers both sub-phases 2a and 2b (the plan offered "one or two"; the user chose one); the dispatch-seam collapse targets the *current* canon in `_preamble.md` (no restructuring of `_preamble.md` itself — that is phase 3a); the protected set is preserved verbatim; SPEC mirrors update in the same change.

## Why

1. **The pain point**: ~6,000 words of historical narration and cross-file duplication in `src/kit/skills/` cost tokens on every skill load and — worse — are the active drift mechanism. The dominant disease is restating a contract instead of pointing at its owner. Proof it bites: PR #539 (`fab dispatch reap`) updated the canonical dispatch contract in `_preamble.md` and none of the downstream restatements in `_pipeline.md`, `fab-continue.md`, `fab-adopt.md` — every copy silently omitted the reap step, one day after the audit measured the pattern. Phase 1 fixed the resulting factual defects; this phase removes the structural cause at the consumer sites.
2. **If we don't**: every future canon edit must fan out to N restatements, and history shows the fan-out gets missed — five real drift bugs so far. The corpus also keeps paying ~6,000 words of load cost per session for zero information.
3. **Why this approach**: pure deletion and collapse-to-pointer, no rewriting judgment — the lowest-risk cut in the plan. Doing 2b before phase 3's `_preamble.md` restructure is the plan's own sequencing recommendation: collapse consumers to pointers against the *current* canon, then restructure the canon in place without re-touching consumers.

## What Changes

All edits are to canonical sources `src/kit/skills/*.md` (never `.claude/skills/` — deployed, gitignored) plus the matching `docs/specs/skills/SPEC-*.md` mirrors. Prose-only; zero behavioral/contract change is a hard requirement. The governing editorial convention to apply throughout: **a file may state a rule it owns, or point at the file that owns it — never both** (applied here as practice; *stating* it in `_preamble.md` is phase 4b, not this change).

> **Line-ref caveat**: the plan's line refs are analysis-baseline refs (commit `6f74e0c3`, re-verified at `b05ab998`). Since then PRs #539/#540/#541 and Phase 1 (#544) landed — Phase 1 touched many of the same files. Treat every ref below as a content anchor and re-verify against HEAD before editing (plan § Execution constraints, item 6).

### 2a. Historical/transition narration sweep (~2,700 words, ~15 files)

Delete or rewrite-as-present-truth. Known sites (plan § 2a — grep for more: `no longer`, `used to`, `former`, `removed in`, `retired`, `predates`, `supersedes`, `unchanged`, change-id citations in prose):

- `_cli-fab.md` § fab hook (~310 words on a command removed in 2.14.0) → delete the section (the one-line successor note near the top of the file already covers it).
- `fab-clarify.md` § Skill Invocation Protocol + § Auto Mode (32 lines, both self-declared unused) → collapse to a 3-line note or relocate to a spec; also delete `_preamble.md`'s pointer section whose only content is that nothing uses the protocol (§ Skill Invocation Protocol (pointer)).
- The `indicative`-flag archaeology ×3: `fab-switch.md`, `fab-status.md`, `_cli-fab.md`.
- The "removed in 1.10.0 / spec stage is gone" cluster: `_preamble.md` (×2), `fab-clarify.md`, `_generation.md` (×4-ish) — keep ONE one-release back-compat note for legacy `spec.md` ingestion (`_generation.md`'s Legacy spec.md ingestion step), delete the rest; record a tracked follow-up to delete the whole back-compat window when it closes.
- `_preamble.md` narration: the § Adoption note, "Review is unexceptional", "(today's behavior)" / "Unchanged; byte-preserving" asides, "former every-10th-poll" (×2), the "two exit-code states stay unreachable, not newly handled" meta-commentary. (Deleting these asides is 2a-scope; restructuring the sections they sit in is phase 3.)
- `_cli-fab.md` narration: `fab spawn-command` removal note, shll.ai push-vs-pull history, pane-title post-mortem, "previously lived in…" ×3, inline change-id citations woven into prose (~15 sites — keep provenance ONLY where it names a still-live regression test or a don't-re-break-this force).
- `fab-operator.md`: the ×3 "old repo-rooted `.fab-operator.yaml` not migrated" (keep one), "schema is unchanged" ×2, plus the remaining narration sites the audit's § 4 table lists.
- `fab-setup.md`: "(absorbed from fab-update)", "former sync-last order", the 2.15.0 template-retirement archaeology (×3).
- git/docs family: `git-pr.md` "why 3a-bis lives in ship" history → one line; `git-pr-review.md` — keep the don't-re-break force, drop change-id archaeology; `docs-distill-memory.md` ×3; `docs-reorg-memory.md` ×2.
- **Keep-list** (explicitly NOT deleted): `git-pr-review.md`'s "(prior efforts stalled mid-poll…)" credibility anchor; every anti-drift prohibition in the protected set.

### 2b. Duplication collapse (~3,300 words)

- **Dispatch-seam restatements → pointers** (the big one). Collapse each consumer to "run the Dispatch Contract for `<stage>` per `_preamble.md` § CLI-Adapter Dispatch" + that site's specific delta only:
  - `_pipeline.md`: the Behavior blockquote (~600w, explicitly violating `_preamble.md`'s own no-restatement rule) and the five per-step repeats (~450w) → extract ONE `### Stage Dispatch Procedure` (≈5 numbered lines) inside `_pipeline.md` and reference it per step.
  - `fab-continue.md`: the ~354-word single-line dispatch paragraph → a 5-step numbered contract; delete the two "Dispatch shorthand" blockquotes (~230w) that exist only to re-explain it.
  - `fab-adopt.md` (~280w), `fab-ff.md` / `fab-fff.md` (~390w), `fab-proceed.md` (~300w) — each keeps only its delta (adopt's `mode: diff-only`; fff's ship/review-pr exception; proceed's role table).
  - Verification (plan § Verification): after the collapse, `_pipeline.md`, `fab-continue.md`, `fab-adopt.md`, `fab-ff/fff.md`, `fab-proceed.md` each contain exactly one dispatch pointer plus their delta, and the `fab dispatch reap` step is reachable from every dispatch site via the pointer. (Phase 1 shipped first and added interim reap mentions to restatements per its item 5 fallback — this collapse supersedes those; verify the actual Phase 1 diff at apply entry.)
- **`fab log command` best-effort caveat**: stated 4× in `_preamble.md` (keep the Common-fab-Commands table row) and restated in 6 skills' `## Command Logging` sections (`fab-discuss`, `fab-setup`, `fab-switch`, `fab-dedupe`, `fab-operator`, `fab-help`) → reduce each to the bare command + when-to-log (model: `fab-dedupe.md`'s form). ~250w.
- **`command -v` gate discipline**: `_preamble.md` self-repeats (cut the middle one); `_cli-external.md` §§ Reference Model + Absent-binary discipline state one rule for ~120 lines with the same snippet block twice and the same scope note twice, plus per-tool restatements ×5 → merge to one gate + one snippet + the fail-silent-vs-stop-with-hint distinction (that distinction is load-bearing — keep it). ~850w.
- **`fab-operator.md` internal cleanup**: the verbatim `rk notify` command duplicated byte-for-byte with `_cli-external.md` (keep the `_cli-external` copy — operator-owned section); the 242-word `&&`-chaining re-derivation (owner: `_cli-agents.md`; the file already points there); `wt` probe-and-route ×3 (owner: `_cli-external.md`); adaptive cadence ×5 → keep § 4's spec; the internal duplicate table — auto-enroll ×3, `»`/`›` prefix ×4, `branch_map` ×5, nearest-predecessor ×4, `stop_stage` ×4. (~110 lines total across 2a+2b for this file.)
- **fab-ff/fab-fff shared framing → `_pipeline.md`**: byte-identical Arguments, ~90%-identical Purpose sentence, byte-identical `{driver}` row, ~95%-identical per-stage-model blockquote, shared Output skeleton — both already load `_pipeline` via `helpers:`. fab-ff shrinks ~30 lines; closes a live twin-drift seam. ~280w.
- **Docs family triples**: FKF authoring rules ×3 in `docs-hydrate-memory.md` → one `### FKF authoring rules` block; the nine defect classes ×3 in `docs-distill-memory.md` → one Detect/Action table; "one domain per approval unit" ×4 → once in Arguments; `docs-reorg-memory.md`'s aggregation rule → pointer to distill's section (it already names distill as owner); older-binary fallback ×3 and `split-file` rules ×4 in `docs-reorg-memory.md`.
- **Small boilerplate**: duplicate trailing `Next:` footers in `fab-new`, `fab-draft`, `fab-dedupe`, `fab-adopt` (keep the Output-block copy); `git-branch.md`'s `## Output` byte-identical to its Step 5 (delete); `fab-archive.md`'s two Key Properties tables sharing 6/8 rows (merge with Archive/Restore columns); the Activation Preamble written out verbatim in `fab-draft.md` + `fab-dedupe.md` (owner: `_preamble.md` § Activation Preamble, which already names both consumers).

### SPEC mirrors + sweep classes

Every touched `src/kit/skills/*.md` gets its `docs/specs/skills/SPEC-*.md` mirror updated in this change (constitution, Additional Constraints; review treats the whole mirror class as in-scope). Sweep classes per `fab/project/code-quality.md` § Sibling & Mirror Sweeps: fab-ff ↔ fab-fff twins; aggregate specs (`docs/specs/skills.md`, `glossary.md`, `architecture.md`) grepped for moved/renamed phrases before finishing apply.

### Protected set (MUST NOT be compressed — verbatim survival)

The plan § Must NOT be compressed applies in full: exact command grammars/flags and fenced usage blocks; byte-stable output tokens (`(none)`, `dispatched …` + `auto:` suffixes, `✓` literals, `Fixed —`/`Deferred —`/`Skipped —` prefixes, resolve-agent line order + omit-when-empty); exit-code tables; `{stage}-result.yaml` schemas + status-vs-verdict split; the three dispatch-prompt obligations; the block-contract carve-out; "orchestrator owns all transitions"; "pipeline never sends keys"; reap-is-not-kill; SRAD formula/thresholds/markers/Worked Examples; parser contracts (`## Requirements`/`## Tasks`/`## Acceptance`, ID formats); the Copilot two-login fact + GraphQL-omits-bots trap; approval-gate semantics; anti-drift prohibitions; mode-selection/fill-precedence ladders; the ecosystem→`test_paths` table. Grep the protected-set literals before and after — byte-identical.

### Verification (from plan § Verification per phase)

1. `fab sync`, then spot-load deployed copies; diff `##` heading sets before/after (no parser-contract heading renamed).
2. Protected-set literal greps byte-identical before/after.
3. Dispatch-pointer check: each of the five dispatch sites has exactly one pointer + its delta; reap reachable from every site.
4. SPEC mirror sweep: `ls docs/specs/skills/SPEC-*.md` against every touched skill; aggregate specs grepped.

## Affected Memory

None. Phase 2 is prose-only deletion/collapse with zero contract loss — no spec-level behavior changes (same rationale as Phase 1, `260808-ip4q`). The plan's Non-goals exclude `docs/memory/` changes beyond the constitution-required SPEC mirrors (which are specs, not memory). If hydrate finds a memory claim citing a deleted restatement's location, correcting the pointer is in scope; content changes are not.

## Impact

- **Files**: ~25–30 of the 36 files under `src/kit/skills/` (2a touches ~15; 2b's heaviest are `_pipeline.md`, `fab-continue.md`, `fab-adopt.md`, `fab-ff.md`, `fab-fff.md`, `fab-proceed.md`, `_cli-external.md`, `fab-operator.md`, `docs-hydrate-memory.md`, `docs-distill-memory.md`, `docs-reorg-memory.md`) plus their `docs/specs/skills/SPEC-*.md` mirrors and the three aggregate specs if phrases move.
- **Size**: ~6,000 words deleted (~2,700 narration + ~3,300 duplication); net-negative diff.
- **No Go changes, no test changes** — prose-only. No migrations (no user-data restructuring). `src/kit/skills/` is under `source_paths` (`src/`), so true-impact counts it.
- **Risk surface**: accidental deletion of a protected-set literal or a load-bearing delta during collapse — mitigated by the before/after greps and the dispatch-pointer verification.

## Open Questions

None — the plan document plus the invocation directive resolve scope, sequencing, and protection rules.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One fab change covers both sub-phases 2a and 2b | User directive; the plan offered "one or two" and the user chose one | S:95 R:70 A:95 D:95 |
| 2 | Certain | 2b's dispatch-seam collapse targets the current `_preamble.md` canon; `_preamble.md` itself is not restructured (phase-3 boundary) | User directive + the plan's sequencing note verbatim | S:95 R:80 A:95 D:95 |
| 3 | Certain | Protected set (plan § Must NOT be compressed) survives byte-identical, verified by before/after greps | User directive + plan requirement; verification procedure specified | S:95 R:60 A:95 D:100 |
| 4 | Certain | SPEC mirrors update in the same change, whole mirror class + twins + aggregate specs swept | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps | S:90 R:75 A:100 D:100 |
| 5 | Certain | Plan line refs are content anchors only — every ref re-verified against HEAD before editing | Plan constraint 6; baseline `6f74e0c3` predates #539–#541 and Phase 1 (#544), which touched the same files | S:85 R:90 A:90 D:90 |
| 6 | Confident | Phase 1 added interim reap mentions to dispatch restatements (its item-5 fallback); this collapse supersedes them with pointers — actual #544 diff verified at apply entry | Phase 1 shipped standalone first, so its fallback path applied; exact wording unverified until apply | S:80 R:80 A:70 D:80 |
| 7 | Confident | 2a scope includes grep-discovered narration sites beyond the known list, under the same delete-or-rewrite rule and keep-list | Plan says "grep for more" with the marker list; keep-list + protected set bound the risk | S:70 R:85 A:80 D:80 |
| 8 | Confident | The ownership convention is applied as editorial practice but NOT stated in `_preamble.md` (that adoption is phase 4b) | Plan wording: "to state in `_preamble.md` once adopted" — adoption is listed under Phase 4b | S:70 R:90 A:75 D:75 |
| 9 | Confident | `change_type: refactor` — prose restructuring with zero behavior change; skills are kit source, not user docs | Consolidation/deletion of source prose fits refactor better than docs; low stakes either way | S:70 R:95 A:80 D:60 |
| 10 | Certain | Affected Memory: none | Template rule (memory updates only for spec-level behavior change) + Phase 1 precedent + plan Non-goals | S:85 R:90 A:90 D:85 |

10 assumptions (6 certain, 4 confident, 0 tentative, 0 unresolved).
