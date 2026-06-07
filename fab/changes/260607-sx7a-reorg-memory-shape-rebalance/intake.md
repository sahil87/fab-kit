# Intake: Memory Rebalancer Apply Path (docs-reorg-memory)

**Change**: 260607-sx7a-reorg-memory-shape-rebalance
**Created**: 2026-06-07
**Status**: Draft

## Origin

Follow-up to `tciy` (`260607-tciy-memory-tree-shape-rebalance`, merged as PR #377). Created as a draft during the `tciy` session; see this folder's `NOTES.md` and `REFERENCE-docs-reorg-memory-stab.md` (a 166-line draft of the enhanced skill).

`tciy` shipped the **foundation + detect half** (Approach B + C-detect): `fab memory-index` (deterministic generated index, kills the hand-edited-index churn/drift class), `description:` frontmatter on memory files, hydrate rewired to call `fab memory-index`, reserved `_shared`/`_unsorted` domains, shape bounds (~5/~12/depth ≤3) as SHOULD guidance, and a **read-only Shape Report** in `docs-reorg-memory` that *proposes* split/merge/flatten but does NOT move files. The merged `docs-reorg-memory.md` carries an explicit scope note: *"the file-moving apply path … is a deferred follow-up."* **That deferred apply path is this change.**

During intake the user chose **External** sub-domain addressing (see Assumptions #1).

## Why

**1. What's missing.** `tciy` can *detect* that a folder is over-wide (e.g. `fab-workflow` at 20 files > ~12) and *propose* a split, but the user must perform the moves by hand. The rebalancer is half-built: diagnosis without action. This change completes it — `docs-reorg-memory` actually performs the approved split/merge/flatten, rewrites the relative links a move breaks, and regenerates indexes via `fab memory-index`.

**2. Why now / why safe.** `tciy` made this cheap and conflict-free: because indexes are now *generated*, moving a file no longer triggers the index merge-conflict that splitting used to cause — `fab memory-index` just re-derives every affected `index.md` after the move. The only remaining hazard is **relative-link rot** (~57 intra-domain links in `fab-workflow` use bare `](file.md)` paths that break when a file moves into a sub-folder), which this change handles with a deterministic rewrite + a no-dangling-link guard.

**3. Why External addressing.** When a split creates `docs/memory/{domain}/{sub-domain}/{topic}.md`, the rest of fab must be able to *address* that file. **External** makes sub-domains first-class: the intake `Affected Memory` slot, `_preamble` always-load, and `context-loading`'s selective-load walk all gain a sub-domain level. Tradeoff considered: loom (a real multi-domain repo) uses External-style nesting and *still* churned its index — but that churn was the hand-edited-index problem, which `tciy` already eliminated (sub-domain indexes are generated too). So External's historical downside is moot post-`tciy`; what remains is its upside: explicit, navigable addressing and no "find-the-file-anywhere-under-the-domain" resolver ambiguity (which the Internal option's duplicate-truth-file failure mode stemmed from).

**4. Why not a Go file-mover.** Split boundaries are semantic/LLM-judged ("do these 8 files cluster into `hydrate/`?"), so the mover lives in the markdown skill (`docs-reorg-memory`), per Constitution I (Pure Prompt Play) — not the `fab` binary. `fab` only does the deterministic index regen (already shipped) and MAY gain a deterministic link-rewrite helper if that proves cleaner than skill-driven edits (see Assumptions #6).

## What Changes

### 1. Activate the apply path in `docs-reorg-memory`

Remove the "deferred follow-up" scope note from `src/kit/skills/docs-reorg-memory.md` (and its spec). The Step-5 apply flow (already drafted in the merged skill as proposals) becomes real:

On approval of a `split` / `merge` / `flatten` / `move` migration:
1. **Move files** to their new paths (`git mv` semantics — preserve history where possible).
2. **Rewrite relative links** broken by the move (every link in the proposal's **Link Impact** note), in BOTH directions: links *from* moved files to siblings, and links *to* moved files from elsewhere in the domain.
3. **Add `description:` frontmatter** to any new file/sub-domain index a split creates (the generated index reads it).
4. **Regenerate indexes**: run `fab memory-index` (never hand-edit).
5. **Verify**: no headings lost, no dangling relative links remain (hard guard — a remaining dangling link blocks finalizing that migration).

### 2. Link Impact analysis (proposal step)

For any move-bearing migration, the proposal MUST list every relative link that will break, with its rewrite (`](runtime-agents.md)` in `execution-skills.md` → `](runtime/runtime-agents.md)`), so the user sees the blast radius before approving.

### 3. External sub-domain addressing (the contract changes)

- **`src/kit/templates/intake.md:38`** — the `Affected Memory` format gains an optional sub-domain level: `{domain}/{sub-domain}/{file-name}` (the flat `{domain}/{file-name}` remains valid for un-split domains).
- **`context-loading`** (`_preamble.md` selective-load convention + the `context-loading` memory doc) — the selective-load walk becomes: read `docs/memory/{domain}/index.md` → if the referenced file lives in a sub-domain, read `docs/memory/{domain}/{sub-domain}/index.md` → read the file. Document the (up to) 3-hop walk.
- **`_preamble.md:43`** always-load layer — unchanged in *which* files it loads (root + domain indexes), but its description of the memory landscape acknowledges sub-domains.
- **`fab memory-index`** — confirm it already renders sub-domain index tiers (it walks the tree; verify depth-2 domains produce a `{domain}/{sub-domain}/index.md`). If it only renders depth-1 today, extend it. **This is the one place `tciy`'s implementation may need a Go change** — the Copilot review on PR #377 flagged that `gatherFiles` only enumerates files directly under a domain dir (depth-2), not depth-3 sub-domain topics. That recursion is in scope here.

### 4. Specs + tests

- Update `docs/specs/skills/SPEC-docs-reorg-memory.md` (remove deferred-scope note; document apply path + link rewriting + External addressing).
- Update `docs/specs/templates.md` and the `templates` / `context-loading` memory docs for the External path.
- Tests: the link-rewrite logic (deterministic, testable), `fab memory-index` recursion into sub-domains (Go test), and a dry-run/golden test of a split proposal's Link Impact accuracy.

## Affected Memory

- `fab-workflow/execution-skills` — (modify) `docs-reorg-memory` now performs moves + link rewriting; rebalancer apply path
- `fab-workflow/templates` — (modify) External sub-domain addressing in memory file format; `{domain}/{sub-domain}/{file}` 
- `fab-workflow/context-loading` — (modify) selective-load becomes a (up to) 3-hop walk through sub-domain indexes
- `fab-workflow/kit-architecture` — (modify) `fab memory-index` sub-domain recursion; rebalancer completion note
- `fab-workflow/schemas` — (modify) if `fab memory-index` sub-domain rendering changes its output schema

## Impact

- **`src/kit/skills/docs-reorg-memory.md`** — activate apply path; Link Impact requirement; External addressing. Constitution: skill change MUST update `SPEC-docs-reorg-memory.md`.
- **`src/kit/templates/intake.md`** — `Affected Memory` gains the sub-domain slot.
- **`src/kit/skills/_preamble.md`** + `_generation.md` if it references the path format — External addressing.
- **`src/go/fab/internal/memoryindex/`** + `cmd/fab/memory_index.go` — recurse into sub-domains (the PR #377 Copilot finding); tests. Update `_cli-fab.md` if behavior/flags change.
- **`docs/specs/`** — `SPEC-docs-reorg-memory.md`, `templates.md`, possibly `architecture.md` (sub-domain path convention).
- **`docs/memory/`** — the memory hydration of this change; NO actual split performed by this change unless the user runs `/docs-reorg-memory` afterward (this change builds the *capability*; it does not necessarily exercise it on `fab-workflow`). **Decision: do we split `fab-workflow` (20 files) as part of this change to d-foodtest the path? See Assumptions #5.**
- **Out of scope (true-impact excluded)**: `fab/`, `docs/`.

## Open Questions

- Should this change **dogfood** the new split path by actually splitting `fab-workflow` (20 files) into sub-domains, or only ship the capability and leave the split to a deliberate later `/docs-reorg-memory` run? (Assumptions #5.)
- Does the link-rewrite live in the skill (LLM-driven Edits) or as a deterministic `fab` helper (e.g. `fab memory-relink`)? (Assumptions #6.)
- For External addressing: does `git mv` history preservation matter enough to require it, or is a plain move + `fab memory-index` acceptable?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | **External** sub-domain addressing: intake template, `context-loading`, and `_preamble` gain a `{domain}/{sub-domain}/{file}` level. | **User-confirmed** this session. loom's External-style churn was the hand-edit problem `tciy` already fixed, so External's downside is moot; its upside (explicit addressing, no resolver ambiguity) stands. | S:90 R:60 A:85 D:90 |
| 2 | Certain | The rebalancer is the **enhanced `docs-reorg-memory` skill** (apply path activated), NOT a new skill or Go file-mover. Moves + link rewriting are skill-driven (LLM judgment); only index regen is in `fab`. | Locked in `tciy`; Constitution I (Pure Prompt Play). The skill already has the proposal scaffolding merged. | S:90 R:70 A:90 D:90 |
| 3 | Confident | `fab memory-index` must **recurse into sub-domains** (render `{domain}/{sub-domain}/index.md`); extend it if it only handles depth-1 today. | The PR #377 Copilot review flagged `gatherFiles` is depth-2-only. Required for External addressing to have generated indexes at the sub-domain tier. | S:75 R:55 A:85 D:80 |
| 4 | Confident | Apply enforces a **no-dangling-link guard**: a move is not finalized while any relative link it broke remains unrewritten. | Link rot is the one real hazard left (index churn is solved). Hard guard prevents shipping a broken tree. | S:70 R:65 A:80 D:75 |
| 5 | Certain | This change ships the **capability only** and does NOT auto-split `fab-workflow`; an actual split is a deliberate, separately-reviewed `/docs-reorg-memory` run. | **User-confirmed.** Keeps the PR reviewable (logic, not a 20-file-move + 57-link-rewrite diff). | S:90 R:70 A:80 D:88 |
| 6 | Certain | Link rewriting is **skill-driven** (the agent edits the links per the Link Impact list), not a new deterministic `fab` helper — unless review finds the skill approach unreliable. | Skill-driven keeps logic in markdown (Pure Prompt Play) and avoids new Go surface; a `fab memory-relink` helper is the fallback if determinism matters. | S:90 R:75 A:85 D:88 |
| 7 | Confident | Hard-depends on `tciy` (merged): reuses `fab memory-index`, the `description:` frontmatter convention, the Shape Report, and reserved domains. | `tciy` is merged to main; this branch is off latest main. | S:85 R:75 A:85 D:85 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
