# Intake: Scrub numbered-constitution citations from shipped skills

**Change**: 260616-u4du-scrub-numbered-constitution-citations
**Created**: 2026-06-16

## Origin

> scrub numbered-constitution citations from shipped skills

Initiated conversationally during a `/fab-discuss` session. The user asked how the shipped
skills under `src/kit/skills/` handle their many references to "the constitution," given that a
**consuming** repo's constitution will differ from fab-kit's own.

Investigation found that the ~79 constitution references split into two categories:

1. **Path / role references** (the large majority) — these refer to `fab/project/constitution.md`
   as *a file at a known path playing a known role* (e.g. `_preamble.md`'s always-load layer,
   `_srad.md`'s Agent-Competence dimension, `fab-setup.md` creating the consumer's constitution).
   These are **correct in any consuming repo** — the skill loads whatever that repo's constitution
   says. No coupling to fab-kit's own constitution text. **Out of scope — do not touch.**

2. **Numbered-principle citations** — a handful of references cite fab-kit's constitution **by
   Roman numeral** (`Constitution I`, `Constitution III`, `Constitution VI`). These bake *this*
   repo's principle numbering into shipped skills. A consuming repo's constitution has entirely
   different numbering, so a citation like "specs stay human-curated (Constitution VI)" is dangling
   or misleading once the skill is deployed elsewhere. **This is the leak to fix.**

Key decisions reached in the conversation (both user-confirmed):
- **Scope**: fix both the shipped skills (`src/kit/skills/`) AND their spec mirrors
  (`docs/specs/skills/SPEC-*.md`) — honoring the constitution's "skill changes MUST update the
  SPEC mirror" rule and avoiding skill↔SPEC drift.
- **Rewrite style**: *name the principle inline* — keep the rationale, drop only the unstable
  numeral pointer. E.g. `(provider neutrality, Constitution I)` → `(provider neutrality — a fab-kit
  design principle)`.

## Why

1. **Problem (the pain point)**: Skills are deployed verbatim into consuming repos via `fab sync`.
   A citation like "specs stay human-curated (Constitution VI)" points at *fab-kit's* principle VI.
   In any other repo, principle VI is something else entirely (or doesn't exist) — so the citation
   is a dead pointer at best and actively misleading at worst. The cited *rule* is sound (it
   describes fab-kit's own design rationale), but the *pointer* breaks on deployment.

2. **Consequence if unfixed**: Consuming-repo agents reading these deployed skills are nudged toward
   fab-kit-specific governance the consumer never authored. Worse, some of these citations frame a
   **fab-kit-internal design invariant** (e.g. "no specs-index generator," "verbatim pass-through")
   as if it were the consumer's constitution — wrong framing regardless of numbering. The reference
   silently rots; nothing catches it because it's prose, not a checked link.

3. **Why this approach over alternatives**: The rule each citation explains is real and worth
   keeping — these are fab-kit explaining *its own* design choices, which happen to be correct in
   this repo. So the fix is **not** to delete the rationale (that loses the "this is a governing
   principle" signal) nor to genericize it to "the project constitution" (still implies the
   consumer's constitution governs a fab-internal choice, and is vaguer). Instead, make each
   citation **self-contained**: name the principle by its concept and attribute it as a *fab-kit
   design principle*, so it survives deployment to any repo unchanged and reads correctly in-context.

## What Changes

A pure documentation-correctness sweep. **No behavioral logic changes** — only the wording of
parenthetical/inline rationale notes that currently cite a constitution by Roman numeral.

The transformation pattern: `Constitution <Numeral>` → the named principle, attributed as a fab-kit
design principle. The named concept already sits adjacent to most citations (e.g. "idempotency,"
"provider neutrality," "Pure Prompt Play," "human-curated"), so the rewrite keeps that concept and
swaps the numeral pointer for a self-contained attribution.

Suggested canonical phrasings (the apply agent MAY tune wording per local sentence flow, but MUST
remove the numeral and MUST keep the named rationale):
- `Constitution I` → `a fab-kit design principle (Pure Prompt Play)` / `(provider neutrality — a
  fab-kit design principle)`
- `Constitution III` → `a fab-kit design principle (idempotency)` / `(idempotency — a fab-kit
  design principle)`
- `Constitution VI` → `a fab-kit design principle (specs stay human-curated)` / `(specs are
  human-curated — a fab-kit design principle)`

### 1. Shipped skills — `src/kit/skills/` (10 citations across 5 files)

- `_cli-fab.md:255` — "(provider neutrality, Constitution I)" → inline-named (Pure Prompt Play /
  provider neutrality is a fab-kit principle).
- `fab-continue.md:206` — "(re-running a reset is a state-wise no-op — Constitution III)" →
  inline-named (idempotency).
- `docs-hydrate-memory.md:191` — "(idempotency, Constitution III)" → inline-named.
- `docs-reorg-memory.md:161` — "(idempotency, Constitution III)" → inline-named.
- `docs-hydrate-specs.md:73` — "specs stay human-curated (Constitution VI)" → inline-named.
- `docs-reorg-specs.md` — **5 occurrences** (lines 18, 20 ×3, 83, 123, 124) all citing
  Constitution VI re: specs human-curated / out of FKF scope. Each rewritten inline; the dense
  line-20 block-quote note carries three and needs careful per-occurrence editing (it also contains
  the phrase "the constitution rejects" and "per **Constitution VI**" — both rewritten).

### 2. Spec mirrors — `docs/specs/skills/SPEC-*.md` (12 citations across 6 files)

Constitution requires skill changes update the SPEC mirror; the mirrors carry their own numbered
citations (some documenting re-run/idempotency contracts the skill bodies phrase differently):

- `SPEC-fab-setup.md:91` — "per Constitution I." → inline-named (Pure Prompt Play).
- `SPEC-git-pr.md:9` — "**Re-run contract** (Constitution III, ...)" → inline-named (idempotency).
- `SPEC-fab-draft.md:9` — "**Re-run contract** (Constitution III)" → inline-named.
- `SPEC-fab-new.md:9` — "**Re-run contract** (Constitution III)" → inline-named.
- `SPEC-fab-continue.md:9` — Constitution III embedded **inside a long contract sentence**
  ("a state-wise no-op — Constitution III"); reword inline without breaking the surrounding clause.
- `SPEC-docs-reorg-memory.md` — **2 occurrences** (line 30 "Idempotency (Constitution III)"; line 46
  "(Constitution I, Pure Prompt Play)").
- `SPEC-docs-reorg-specs.md` — **3 occurrences** (lines 7, 9 ×2 [includes "per **Constitution VI**"],
  22) re: specs human-curated / no FKF frontmatter.

### 3. Other deployable kit content — scope expansion (post-review)

Review passed on sections 1–2, but surfaced the **same leak class in other deployable kit content**.
Per user decision, scope was expanded to cover all *deployable* / *forward-looking* citation sites,
keeping historical records out:

- `src/kit/reference/fkf.md:21` — deployed to the kit cache (`$(fab kit-path)/reference/fkf.md`) and
  read by deployed skills; carried `Constitution VI`.
- `docs/specs/fkf.md:13,349` — the dev-repo single-source that `src/kit/reference/fkf.md` is
  extracted from; edited in **lockstep** to preserve single-sourcing.
- `docs/specs/index.md:26` — the specs-index row for fkf cited `Constitution VI`.
- `src/kit/migrations/2.4.2-to-2.5.0.md:131` — a deployed migration verification note cited
  `Constitution III`.

### Non-Goals

- **Path/role references untouched** — the ~69 references that name `constitution.md` as a file or
  describe its role are correct in any repo and MUST NOT be changed.
- **`docs/memory/**` left as historical record** — 18 memory files cite the constitution numbering
  *as it was when each change shipped*. Memory is post-implementation history (not deployable skill
  text, so no consumer-repo leak), and several files (`log.md`/`log.seed.md`) are **generated**
  (`fab memory-index` — do not hand-edit). Rewriting history is out of this change's intent.
- **`docs/specs/findings/*` left as point-in-time artifacts** — dated analysis docs that record the
  numbering at the time of writing; treated as historical record, not deployable docs.
- **No edits to fab-kit's actual constitution** (`fab/project/constitution.md`) — its own numbering
  is legitimate; only *citations of it from deployable skill/spec text* are the problem.
- **No `.claude/skills/` edits** — that's the gitignored deployed copy; `src/kit/skills/` is canonical.

> **Correction (post-review)**: an earlier draft of this list claimed "a grep confirmed zero numbered
> citations in top-level `docs/specs/*.md`." That was inaccurate — `docs/specs/fkf.md` and
> `docs/specs/index.md` did carry `Constitution VI` (now fixed in §3 above), and
> `docs/specs/findings/*` still do (intentionally left as historical artifacts).

## Affected Memory

No memory updates expected. This is a documentation-wording correctness fix in skill source and
spec mirrors; it changes no system behavior, no command signature, no schema, and no design that the
`docs/memory/` domains record. (If hydrate later finds a memory note that itself cites a numbered
constitution principle, that would be a separate follow-up — none is known at intake time.)

## Impact

- **Files touched**: 5 skill files under `src/kit/skills/` + 6 spec-mirror files under
  `docs/specs/skills/` = **11 files**, 22 citation sites total.
- **APIs / commands / schemas**: none.
- **Deployment**: changes ride into consuming repos on the next `fab sync` — no migration needed
  (pure prose edit, no user data restructure).
- **Verification**: after the change, `grep -rnE "Constitution [IVX]+" src/kit/skills/ docs/specs/skills/`
  MUST return zero matches; the named rationale concepts (idempotency / provider neutrality / Pure
  Prompt Play / human-curated specs) MUST still be present at each former citation site.
- **Risk**: very low — no logic paths, no tests exercise this prose. The only care points are the
  multi-occurrence lines (`docs-reorg-specs.md:20`, the `SPEC-docs-reorg-specs.md`/
  `SPEC-docs-reorg-memory.md` clusters) and the one citation embedded mid-sentence
  (`SPEC-fab-continue.md:9`), which need per-occurrence editing rather than a blunt find-replace.

## Open Questions

None — scope and rewrite style were both confirmed with the user during the discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope = shipped skills (`src/kit/skills/`) + spec mirrors (`docs/specs/skills/`); leave top-level specs and the actual constitution untouched | User explicitly chose "Skills + spec mirrors"; grep confirmed top-level specs have zero numbered citations; constitution self-citation is legitimate | S:95 R:80 A:95 D:95 |
| 2 | Certain | Rewrite style = name the principle inline + attribute as a fab-kit design principle; keep the rationale, drop only the numeral | User explicitly chose "Name the principle inline" over drop-entirely and generic-constitution | S:95 R:75 A:95 D:95 |
| 3 | Certain | Path/role references (the ~69 that name constitution.md as a file/role) are out of scope and stay verbatim | They are correct in any consuming repo — established in the discussion; the leak is numbered citations only | S:90 R:70 A:95 D:90 |
| 4 | Confident | No memory updates — pure prose correctness fix, no behavior/schema/command change | Constitution II ties memory to shipped behavior; nothing ships behaviorally here. Reversible via hydrate if a citing memory note surfaces later | S:80 R:75 A:85 D:80 |
| 5 | Confident | Multi-occurrence lines and the mid-sentence citation (`SPEC-fab-continue.md:9`) get per-occurrence inline edits, not a blanket sed replace | A blunt replace would mangle the dense block-quote in `docs-reorg-specs.md:20` and the long contract clause; inventory already enumerates every site | S:85 R:80 A:80 D:75 |

5 assumptions (3 certain, 2 confident, 0 tentative, 0 unresolved).
