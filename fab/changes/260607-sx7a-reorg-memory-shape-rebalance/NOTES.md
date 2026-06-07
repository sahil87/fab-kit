# Draft notes — reorg-memory-shape-rebalance (follow-up to tciy)

**Status**: DRAFT — not activated. Pick up *after* `260607-tciy-memory-tree-shape-rebalance` ships.

## What this change is

The **second half** of the memory-shape work. `tciy` ships the foundation:
`fab memory-index` (generated index, kills churn+drift), `description:` frontmatter,
hydrate wiring, reserved `_shared`/`_unsorted`, shape bounds as guidance, and the
**detect/diagnose** Shape Report in `docs-reorg-memory`.

This change adds the **apply** half: enhance the existing `docs-reorg-memory` skill to
actually *rebalance* — split over-width domains into sub-domains, merge under-floor
siblings, flatten over-depth trees — and on apply, **rewrite relative links** + regen
indexes via `fab memory-index`.

> Decision locked in tciy: the rebalancer is the **existing `docs-reorg-memory` skill,
> enhanced** — NOT a new `/fab-rebalance-memory`, NOT a Go file-mover. Splits are
> LLM-judged (agent work → markdown skill, per Pure Prompt Play).

## The reference stab

`REFERENCE-docs-reorg-memory-stab.md` is a full draft of the enhanced skill (166 lines)
written during the tciy session. It already contains: Ideal Shape Bounds table, Reserved
Domains, the Shape Report (Step 3), the `Kind` migration column + Link Impact note
(Step 4), and the `fab memory-index` + link-rewrite apply path (Step 5). Use it as the
starting point — most of the skill body is done; this change's real work is:

1. **The open decision**: Internal-vs-External sub-domain addressing. Does the intake
   `{domain}/{file-name}` contract (`src/kit/templates/intake.md:38`), `_preamble.md:43`
   always-load, and `context-loading`'s 2-hop convention gain a sub-domain slot, or do
   sub-domains stay internal to a domain (flat lookup preserved)? Decide with post-tciy
   evidence. (loom uses External and still churns — lean toward Internal or measure first.)
2. **Link-rewriting machinery**: ~57 intra-domain relative links in fab-workflow break on
   any split (verified). Needs a deterministic rewrite step + a no-dangling-link check.
3. **Update `docs/specs/skills/SPEC-docs-reorg-memory.md`** to match (constitution).
4. **Tests** for the split/merge/flatten apply path.

## Dependency

Hard-depends on `tciy` (needs `fab memory-index` to exist). Do not start until tciy is merged.

## To resume

`/fab-switch sx7a` (or `/fab-new`-style intake generation), then proceed. The intake was
intentionally NOT generated — generate it fresh when picking this up, incorporating the
reference stab and whatever tciy actually shipped.
