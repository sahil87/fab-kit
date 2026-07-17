# Intake: FKF Present-Truth Body-Style Rule + Memory-Writer Fixes

**Change**: 260717-3plm-fkf-present-truth-style-rule
**Created**: 2026-07-17

## Origin

Promptless dispatch (shared Create-Intake Procedure, `{questioning-mode} = promptless-defer`) from a synthesized user conversation. The conversation diagnosed a systemic style defect in `docs/memory/` and decided a two-part fix (spec rule + writer fixes) shipped as this single change, with a corpus-cleanup skill explicitly deferred to a follow-up change.

> Memory docs under `docs/memory/` are written as accumulated change-keyed deltas / narrated diffs rather than statements of current truth. Amend FKF §3.3 with a normative present-truth body-style rule (plus a §3.2 no-change-ids-in-description clarification) and fix the memory writers (`fab-continue` Hydrate, `docs-hydrate-memory`, the memory template) so hydrate rewrites sections to current truth instead of appending change-keyed delta entries. Provenance *citations* stay allowed; transition *narration* is banned.

All decisions below were made in that conversation; no fabrication beyond it.

## Why

**The pain point.** Memory topic files narrate their own edit history instead of stating current truth. Examples from `docs/memory/pipeline/execution-skills.md`: "renamed `spawn=`→`dispatch=` in tykw", "this inverts the w7dp claim", "supersedes szxd's form", "was `agent.spawn_command`". Measured: ~460 change-id references across topic files; the worst file (`pipeline/execution-skills.md`) has 67. Even `description:` frontmatter carries change-ids (verified in the current tree: `memory-docs/hydrate.md` "— xu0k", `memory-docs/hydrate-specs.md` "— d9rs", `memory-docs/specs-index.md` "(d9rs)", "(5ewp)", `distribution/setup.md` "since szxd … (c5tr) … j0qm: … 8ken:"), against the spirit of FKF §3.2's routing-signal rule.

**The consequence if unfixed.** The narration (a) duplicates what already exists elsewhere — the per-folder generated `log.md` records the dated *what* (FKF §6), git history records the diff, and archived change folders record the full design; (b) accumulates monotonically — nothing ever consolidates a file back to current truth; (c) forces readers to mentally apply a patch series to learn the current contract; and (d) wastes tokens on every always-load/lookup read (estimated 20–40% of body text in the worst files). Any cleanup done without fixing the writers is a treadmill: hydrate would immediately resume producing deltas.

**Root causes (located in the writer contracts, all verified in the current tree):**

1. Hydrate's "Merge without duplication" rule (`src/kit/skills/fab-continue.md` Hydrate Behavior, Steps → item 4, currently line 208) keys dedup on the **change name**: "before appending to a target memory file, check it for an existing entry referencing this change (**by change name**) and update that entry in place instead of appending a duplicate". This institutionalizes change-keyed entries — the unit of memory is the change, not the topic.
2. Sanctioned provenance markers invite narration: the memory template (`src/kit/templates/memory.md`) ships `*Introduced by*: {change-name}` and hydrate's pattern-capture step (fab-continue.md, currently line 213) says "note them … with the change name for traceability".
3. Hydrate is incremental and change-scoped; no writer step ever consolidates a file back to current truth.

**Why this approach.** Fix the writers and the normative spec first (this change), then build the corpus-cleanup skill after the fixes land (follow-up change) — otherwise cleanup is a treadmill. Keep provenance *citations* (cheap, 6 chars, proven to defend deliberate behavior against future "fixes" — e.g. the Copilot poll-predicate was repeatedly re-broken until pinned); ban only the description of superseded state.

## What Changes

### 1. FKF §3.3 — normative present-truth body-style rule (dev spec + shipped extract)

Amend `docs/specs/fkf.md` §3.3 (Body) with a normative body-style rule:

- The body states **current truth in present tense**.
- **No transition narration** — never "renamed X→Y in {id}", "this inverts/supersedes {id}'s claim", "was `old.value`".
- **Superseded behavior is never described in the body.** The previous state belongs to the per-folder generated `log.md` (§6), git history, and archived change folders — the body describes only what IS.
- **Provenance is limited to** trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions.
- **Rationale survives distillation**: "don't re-break this" content lives in Design Decisions `Why`/`Rejected` as durable present-tense intent (a rejected alternative is a design fact, not transition narration). Token savings come from dropping narration, never rationale.

### 2. FKF §3.2 — clarification: no change-ids in `description:` frontmatter

Amend §3.2: the curated `description:` one-liner MUST NOT contain change-ids (neither `— xu0k` suffixes nor `(d9rs)` citations). The description is a routing signal; provenance citations belong in the body. (Enforcement is NOT added — `fab memory-index` validation is explicitly not extended by this change.)

### 3. Dual-update rule (binding)

Per the FKF header rule in `docs/specs/fkf.md` (verbatim: "**Any change to FKF normative rules MUST update both files** so they cannot silently diverge"), both amendments land in BOTH `docs/specs/fkf.md` (dev spec, §3.2/§3.3) AND `src/kit/reference/fkf.md` (shipped normative extract — §3 is inside its shipped subset, original anchors preserved).

### 4. Memory-writer fixes (so the defect doesn't recur)

- **`src/kit/skills/fab-continue.md` (Hydrate Behavior)**:
  - Change "Merge without duplication"'s dedup key from *change name* to *topic/section*, and instruct hydrate to **rewrite the affected section as the new current truth** rather than append (or update in place) a change-keyed delta entry. Superseded statements are removed, not narrated; the change's dated *what* is already captured once via `fab status set-summary` → `log.md` (the existing C-lite step).
  - Align the pattern-capture step (item 6, "with the change name for traceability") with citation-form provenance (trailing citation / `*Introduced by*`), not narration.
- **`src/kit/skills/docs-hydrate-memory.md`**: ingest Step 3 item 4 ("If target file exists → **merge** new content…") gains the same present-truth rewrite instruction; the description-authoring lines (ingest/generate/backfill all synthesize `description:` one-liners) gain the no-change-ids rule.
- **`src/kit/skills/docs-reorg-memory.md`**: verify during apply — it moves/merges whole files rather than authoring body entries, but FKF §3.2 names it a `description:` author, so the no-change-ids-in-description rule likely applies to its description-rewrite paths. Update only what actually authors memory content.
- **`src/kit/templates/memory.md`**: guidance comments state the present-truth body style (present tense, no narration, no superseded behavior; citations allowed). The `*Introduced by*: {change-name}` line on Design Decisions is KEPT (allowed provenance).

### 5. Mirror-class sweep (constitution-required)

Per the constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`") and `fab/project/code-quality.md` § Sibling & Mirror Sweeps, sweep the whole class up front:

- `docs/specs/skills/SPEC-fab-continue.md`, `docs/specs/skills/SPEC-docs-hydrate-memory.md` (+ `SPEC-docs-reorg-memory.md` only if that skill file is edited).
- Aggregate specs restating per-skill facts: check `docs/specs/skills.md`, `docs/specs/templates.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md` for restatements of the merge contract / memory-body shape.
- Memory files documenting the writers (see Affected Memory) — in the sweep class even where under-listed.
- Grep the old claims repo-wide before finishing apply (e.g. "by change name", "with the change name for traceability").

### 6. Explicitly OUT of scope (deferred to follow-up changes)

- **The `docs-distill-memory` cleanup skill** that rewrites the existing corpus to the new style — decided to be a separate NEW skill (not a mode of `docs-reorg-memory`), built AFTER these writer fixes land.
- Rewriting existing memory files (beyond files this change's own hydrate touches, which are brought to the new style, including their `description:` frontmatter).
- Extending `fab memory-index` validation to detect violations — no Go/CLI change in this change.
- Any `fkf_version` bump (see Assumptions).

### Rejected alternatives (from the conversation, recorded for plan generation)

- **Distill as a mode inside `docs-reorg-memory`** — rejected: not discoverable.
- **Cleanup-only (a distill skill without writer fixes)** — rejected: a treadmill; hydrate would keep producing deltas.
- **Zero provenance (ban change-ids entirely)** — rejected: citations defend deliberate behavior (Copilot poll-predicate precedent).

## Affected Memory

- `pipeline/execution-skills.md`: (modify) documents `/fab-continue` hydrate behavior incl. the merge-without-duplication contract — update to the topic-keyed rewrite-as-current-truth contract
- `memory-docs/hydrate.md`: (modify) documents `/docs-hydrate-memory` hydration rules — add the present-truth merge + no-change-ids-in-description rules
- `memory-docs/templates.md`: (modify) documents the shipped memory template + memory-file format — add the §3.3 body-style rule and §3.2 description clarification
- `memory-docs/hydrate-generate.md`: (modify, verify) generate-mode placement/skeleton rules — likely a light touch only if generate-mode guidance changes

## Impact

- **Files edited (prose only, no Go/CLI)**: `docs/specs/fkf.md`, `src/kit/reference/fkf.md`, `src/kit/skills/fab-continue.md`, `src/kit/skills/docs-hydrate-memory.md`, `src/kit/templates/memory.md`, possibly `src/kit/skills/docs-reorg-memory.md`; SPEC mirrors `docs/specs/skills/SPEC-fab-continue.md`, `SPEC-docs-hydrate-memory.md` (+ `SPEC-docs-reorg-memory.md` if its source is edited); aggregate specs as the sweep finds; Affected Memory files at hydrate.
- **Kit distribution**: edits go to canonical `src/kit/` sources only (never `.claude/skills/` deployed copies); they ship via the normal release/sync path. No migration file — no user-data restructuring (existing memory files are not rewritten; the rule governs future writes).
- **No test impact**: no `.go` files change; `fab memory-index` behavior is unchanged.
- **Review focus**: `documentation_accuracy` + `cross_references` checklist categories; SPEC-mirror sync is must-fix per `fab/project/code-review.md`.

## Open Questions

None — the source conversation resolved all scope decisions (see Assumptions; no rows were deferred).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Rule semantics: body = current truth in present tense; no transition narration; superseded behavior never in the body (belongs to `log.md`/git/archives); provenance limited to trailing `(change-id)` citations + `*Introduced by*` on Design Decisions | Decided in conversation with explicit rationale and examples; the citation-vs-narration nuance was an explicit design decision to preserve | S:95 R:70 A:90 D:95 |
| 2 | Certain | Both FKF amendments update BOTH `docs/specs/fkf.md` AND `src/kit/reference/fkf.md` in this change | The FKF header carries a verbatim MUST-update-both rule; §3 is inside the shipped extract's subset | S:100 R:80 A:100 D:100 |
| 3 | Certain | §3.2 clarification is a flat ban: no change-ids anywhere in `description:` frontmatter (citations live in the body) | Decided in conversation ("no change-ids in `description:` frontmatter"); consistent with §3.2's routing-signal rationale | S:90 R:75 A:90 D:90 |
| 4 | Certain | Hydrate's merge-without-duplication dedup key changes from *change name* to *topic/section*, and hydrate rewrites the affected section as the new current truth instead of appending a change-keyed delta | Decided in conversation as the key edit; root cause #1 | S:95 R:70 A:90 D:90 |
| 5 | Certain | `docs-distill-memory` corpus cleanup is a separate follow-up NEW skill built after this lands; this change does not rewrite the existing corpus | Decided in conversation; both alternatives (reorg mode, cleanup-only) explicitly rejected with reasons | S:95 R:85 A:90 D:95 |
| 6 | Certain | Constitution-required sweep: SPEC mirrors for every edited skill file, plus aggregate-spec and writer-memory-file checks, greping old claims repo-wide before finishing apply | Constitution Additional Constraints + code-quality.md § Sibling & Mirror Sweeps mandate it; recurring-rework precedent | S:85 R:85 A:95 D:90 |
| 7 | Confident | No Go/CLI change ships: `fab memory-index` validation is NOT extended to enforce the new rules | Explicit conversation constraint ("verify during planning"); prose-only surface confirmed by file inspection | S:85 R:75 A:80 D:80 |
| 8 | Confident | No `fkf_version` bump: the style rule is a backward-compatible authoring convention within v0.1; a bump would require a Go change (`fab memory-index` writes `fkf_version`), contradicting the no-Go constraint | Not discussed in conversation — inferred from FKF §8 (minor versions add backward-compatible features; generator owns the field) + the no-Go constraint | S:55 R:85 A:80 D:80 |
| 9 | Confident | `docs-reorg-memory.md` needs at most a description-authoring touch (no body-style edit): it moves/merges whole files; grep shows no change-keyed body-authoring language; FKF §3.2 names it a `description:` author | Verified by grep during intake; full read deferred to apply — update only what actually authors memory content | S:70 R:90 A:70 D:75 |
| 10 | Certain | Pattern-capture step and template guidance align to citation-form provenance (trailing citation / `*Introduced by*`), keeping traceability while removing narration-inviting phrasing; the template's `*Introduced by*` line is kept | Direct application of decided semantics (root cause #2); conversation explicitly keeps `*Introduced by*` | S:75 R:80 A:85 D:80 |
| 11 | Certain | Memory files touched by this change's own hydrate are brought to the new style, including removing change-ids from their `description:` frontmatter; all other corpus files untouched | Conversation: "does not rewrite existing memory files (beyond any hydrate of this change itself)" | S:80 R:85 A:80 D:75 |
| 12 | Confident | Exact placement/wording of the normative rule text (e.g. a body-style block inside §3.3 preserving existing anchors in both files) is decided at apply within the decided semantics | Wording is agent-decidable; semantics fully pinned by rows 1–3; reversible via review/rework | S:65 R:85 A:85 D:70 |
| 13 | Confident | Change type stays as procedure-inferred (`fix`-or-`docs` band; both carry expected_min 3, flat 3.0 gate) | Conversation: "let the procedure's normal type detection decide"; type is fully reversible via `fab status set-change-type` | S:60 R:95 A:85 D:60 |

13 assumptions (8 certain, 5 confident, 0 tentative, 0 unresolved).
