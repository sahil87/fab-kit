# Intake: Memory Tree Shape & Rebalance

**Change**: 260607-tciy-memory-tree-shape-rebalance
**Created**: 2026-06-07
**Status**: Draft

## Origin

Backlog item `[tciy]` (2026-06-01):

> Sometimes, the memory folders become too wide. What's the ideal shape? I am assuming trees that are neither too deep nor too wide. How to achieve this? Also should we add a rebalance skill? (Wide trees create central memory files that get modified on almost every change — with merge conflicts.)

Invoked via `/fab-new tciy && /fab-fff`. During intake, the user chose scope **"guidance + rebalance skill"** and, for the anti-churn mechanism, asked for a **multi-agent analysis of which approach is best** rather than picking one upfront.

That analysis was run (7 sub-agents: one proponent + one adversarial skeptic per approach, plus a prior-art scout, then a synthesis judge). Its conclusions are folded into **What Changes** and the **Assumptions** table below. The headline: the measured pain is **not tree width** — it is the **hand-edited per-row index columns** (`description` + `Last Updated`) that get rewritten on nearly every content edit. Only a **generated index** removes that hand-edit; tree-shape changes alone (splitting) merely relocate the conflict and, worse, *manufacture* a burst of new conflicts by moving files and breaking ~57 intra-domain relative links.

## Why

**1. The problem, quantified.** The current memory tree is maximally wide and shallow: a single domain (`docs/memory/fab-workflow/`) holding **21 flat topic files** plus one `index.md`. Measured over the last 100 commits:
- **65/100 commits touch a memory file under `docs/memory/`**; of those, **~63 touch a domain index**, and **~57 are pure in-place row rewrites** (symmetric `+N/-N` on the hot `description`/`Last Updated` cells) — the textbook merge-conflict generator.
- The **root** `docs/memory/index.md` is touched ~0–6/100 (it only changes on domain birth) — so the churn lives in the **per-domain** index, not the root.
- The hand-stamped dates are *already wrong* (e.g. index says `model-tiers 2026-02-19` / `hydrate 2026-05-07`; git says `2026-04-02` / `2026-05-08`), and the root inline roster already lists 18 files when 20+ exist — a hand-maintained registry that has silently drifted.

**2. What happens if we don't fix it.** As the project grows past one domain, every change continues to serialize on hand-edited index rows → recurring merge conflicts on `docs/memory/` during parallel/worktree work, plus a routing index (`_preamble` always-load layer, `_preamble.md:43`) that drifts out of sync with reality and degrades agent context selection.

**2a. Cross-repo evidence (loom & run-kit).** Two example repos confirm this is not hypothetical:
- **`loom`** (mature, ~40-domain memory tree, depth up to 3 — i.e. the future fab-kit is worried about) **already uses hierarchical sub-domains (Approach A)** and it did **not** prevent the churn: loom touches *an* index in **100/100** recent commits and the **root** index in **64/100**. A relocated the hot line; it did not remove the hand-edit.
- **loom's hand-maintained root index is already badly stale**: it inlines per-domain file rosters with counts, and **4 of 7 sampled folders are wrong** — `wd-web/canvas` claims `(12)` but has **20** files on disk; `styling` `(7)` vs **15**; `testing` `(7)` vs **13**; `multiplayer` `(4)` vs **6**. This is the "hand-maintained registry silently drifts" failure, *in production*. A generated index (Approach B) is correct by construction; A/C cannot fix it.
- **Sub-domains overflow anyway**: loom's `wd-web/canvas` holds **20 files even as a sub-domain** — over any sane fan-out bound — so a reactive split trigger (Approach C, ~12) is genuinely needed, not just a one-time reshape.
- **Neither repo uses frontmatter** (files start `# H1` then `## Overview`), confirming descriptions are not derivable (Assumption #3).
- **`run-kit`** is single-domain (`run-kit/`, 5 files) — same starting shape as fab-kit, reinforcing that the guidance must scale from 1 domain to loom-scale gracefully.
- loom also uses two reserved-domain conventions worth adopting: **`_shared/`** (cross-cutting, maps to no single package) and **`_unsorted/`** (staging area for not-yet-placed notes), and maintains its index via `/fab-archive` as well as hydrate.

**3. Why this approach over alternatives.** Three anti-churn approaches were evaluated adversarially. **Adjusted scores after skeptic critique: B (generated index) = 61, A (hierarchical sub-domains) = 48, C (fan-out bounds) = 41.**
- **B eliminates** the conflict class (no hand-edit → no conflict) and reuses the established `prmeta`/`impact`/`score` deterministic `Render`/`Gather` Go pattern already in `src/go/fab/internal/` — the same move `git-pr` made for the PR Meta block.
- **A and C only reduce** conflict *probability* (they relocate/shrink the hot line, not remove it), and both *manufacture* a one-time conflict bomb: a split moves files, breaking the **57 intra-domain relative links** and the intake `{domain}/{file-name}` contract (`src/kit/templates/intake.md:38`).
- Therefore the design is **B as foundation, C's bounds as the trigger, A's sub-domain split as the *action* of the rebalance skill** — and **B ships first** so file moves (A/C) become cheap and conflict-free.

This also satisfies the user's explicit two-part scope: (1) codify ideal-shape guidance, and (2) add a `/fab-rebalance-memory` skill.

## What Changes

### 1. `fab memory-index` — generated index command (the foundation, Approach B)

A new pure-Go subcommand modeled byte-for-byte on the existing `prmeta` package (a pure `Render(Data) string` + a `Gather` I/O orchestrator that walks `docs/memory/`). It:

- Enumerates each `{domain}/` folder and its non-`index` `.md` files.
- Reads each file's H1 title plus a machine-readable **`description:` frontmatter field** (new — see §3).
- Stamps **"Last Updated"** from `git log -1 --date=short <file>` (authoritative, free of merge pain), degrading gracefully in worktree/squash/rebase/shallow-clone/uncommitted contexts (mirror `prmeta`'s degradation).
- Writes, deterministically and idempotently (byte-stable on re-run):
  - the **root** `docs/memory/index.md` → **domain rows only** (drop the inlined per-file "Memory Files" column at `templates.md:369`), and
  - every **`docs/memory/{domain}/index.md`** (and, post-rebalance, `{domain}/{sub-domain}/index.md`) → file rows.

The indexes become **generated artifacts agents never hand-edit**, so two branches can never produce conflicting hand-edits to the same index row; any residual textual diff conflict auto-resolves by re-running `fab memory-index` post-merge.

### 2. Hydrate codifies shape guidance + calls `fab memory-index` (Approaches C + B wiring)

In the hydrate skill (`docs-hydrate-memory` / the `/fab-continue` hydrate stage) and the memory template/specs:

- **Replace** the prose "Step 4: Update Indexes" (hand-maintain rows without removing entries) with a single mechanical call to **`fab memory-index`**.
- **Codify ideal-shape bounds** as SHOULD guidance:
  - **Max ~12 topic files per folder** (soft upper bound); **lower bound ~5** before a sub-domain earns its own index.
  - **Max depth 3**: `docs/memory/{domain}/{sub-domain}/{topic}.md`.
  - **Introduce a sub-domain only reactively** — when a real cluster of **≥8 cohesive files** exists in one domain. Never pre-build hierarchy (Obsidian/monorepo/Starlight prior-art consensus: let clusters emerge).
- Add a **sequencing guard** (preflight check or hook) so a forgotten regen can't silently rot the routing index — there's no hand-edit left to catch staleness.

### 3. `description:` frontmatter on memory files (load-bearing for B)

The index `description` column is **not derivable** from file contents today (descriptions are curated one-liners; auto-extracting H1 + first `## Overview` sentence is lossy and would break tables — e.g. `hydrate.md`'s Overview contains literal `|` pipes — and degrade the always-load routing signal). So:

- Add a **`description:` frontmatter field** to the memory-file template (co-located metadata — the Starlight lesson).
- **Backfill all ~21 existing files** with their current curated descriptions (one-time migration; conflict moves from the hot index row to a far colder per-file frontmatter line).
- Every memory writer (hydrate, `/docs-hydrate-memory`, `/fab-rebalance-memory`) authors this field going forward.

### 4. Fan-out bound detection — `fab memory-index` warns when shape is violated (Approach C, detect-only)

This is the **safe half of C**, shipped now. The risky half (actually moving files into sub-domains — Approach A) is **explicitly out of scope** for this change (see §"Scope boundary").

`fab memory-index` already walks the whole tree to regenerate indexes, so computing per-folder file counts and depth is nearly free. On every run it emits **non-fatal warnings** (stderr) when a folder violates the shape bounds:

```
⚠ docs/memory/fab-workflow has 20 topic files (soft bound: ~12) — consider splitting into sub-domains
⚠ docs/memory/<domain>/<sub>/<deep> exceeds depth 3 — consider flattening
```

- Warnings are advisory; they never block, never modify files, and never auto-split (Constitution: Idempotent — a regen with warnings is still byte-stable on the index output).
- The bounds (~5 lower / ~12 upper, depth ≤3) are codified as **SHOULD guidance** in the hydrate skill + memory template/specs — directly answering the backlog item's "what's the ideal shape?".
- Reserved domains `_shared/` and `_unsorted/` are **exempt** from the bound check.
- The **existing `docs-reorg-memory` skill is the rebalancer** — no separate `/fab-rebalance-memory` skill is created. This change teaches `docs-reorg-memory` the shape vocabulary (a Shape Report in its diagnosis, split/merge/flatten in its migration map, reserved-domain exemptions) and switches its index updates to `fab memory-index`. The *file-moving apply path* it now documents (link rewriting on split) is exercised in the follow-up; `tciy` ships and tests the **detect/diagnose + index-regen** behavior.

### Scope boundary (what this change does NOT do)

- **No file moves / no sub-domain splitting.** That is Approach A and lands in a follow-up change.
- **No new `/fab-rebalance-memory` skill.** The rebalancer is folded into the existing `docs-reorg-memory` skill. This change adds its shape *vocabulary* (detect/diagnose + `fab memory-index` wiring); the file-moving *apply* path is deferred (validated in the follow-up).
- **No change to the intake `{domain}/{file-name}` contract, `_preamble.md:43` always-load, or `context-loading`'s 2-hop convention.** Because no files move, the flat lookup is preserved as-is. This is what resolves the formerly-Unresolved sub-domain-path question: **the decision is "not now" — sub-domains aren't introduced, so neither path contract changes.**
- **No relative-link rewriting machinery.** Not needed until files move.

### Phasing

1. **This change (B + C-detect)**: `fab memory-index` (generated index + bound warnings) + `description:` frontmatter + hydrate wiring + reserved `_shared`/`_unsorted` + shape bounds as guidance + the `docs-reorg-memory` Shape Report / vocabulary. Kills the conflict + drift class immediately and tells you when a folder is over-wide.
2. **Follow-up change (A + C-apply)**: exercise & harden `docs-reorg-memory`'s split/merge/flatten *apply* path — decide Internal-vs-External addressing with real evidence, rewrite links on move — cheap and conflict-free *because* B's generated index already exists.

## Affected Memory

- `fab-workflow/hydrate` — (modify) Step 4 "Update Indexes" replaced with `fab memory-index`; shape bounds added
- `fab-workflow/hydrate-generate` — (modify) index maintenance now mechanical (`fab memory-index`)
- `fab-workflow/hydrate-specs` — (modify) consistency with the generated-index flow
- `fab-workflow/templates` — (modify) memory-file template gains `description:` frontmatter; root index becomes domains-only
- `fab-workflow/configuration` — (modify) any bound/threshold config surface, if added
- `fab-workflow/kit-architecture` — (modify) new `fab memory-index` command in the kit map; note the deferred rebalance follow-up
- `fab-workflow/configuration` — (modify) shape bounds as guidance (and config surface if added)
- *(All ~21 files)* — (modify) one-time `description:` frontmatter backfill

> `context-loading` is **not** modified — the flat `{domain}/{file-name}` lookup is unchanged because no sub-domains are introduced. A `fab-workflow/rebalance` memory file will be created by the follow-up change, not this one.

## Impact

- **`src/go/fab/`** — new `internal/memoryindex` package (pattern: `internal/prmeta`) + `cmd` wiring; tests (`*_test.go`) with byte-for-byte render fixtures (Constitution: tests conform to spec). Update `src/kit/skills/_cli-fab.md` with the new command signature (constitution constraint).
- **`src/kit/skills/`** — modify the hydrate skill(s) (Step 4 → `fab memory-index`; shape bounds as guidance) **and `docs-reorg-memory.md`** (Shape Report, split/merge/flatten vocabulary, reserved-domain exemptions, `fab memory-index` regen on apply). No new skill file. Constitution: skill changes MUST update the corresponding `docs/specs/skills/SPEC-*.md`.
- **`src/kit/templates/`** — memory-file template (`description:` frontmatter); root/domain index template (domains-only root).
- **`docs/memory/` (all ~21 files)** — one-time `description:` frontmatter backfill + first `fab memory-index` regen (a large but mechanical migration diff).
- **`docs/specs/`** — update `templates.md`, `skills.md` (hydrate), **`docs/specs/skills/SPEC-docs-reorg-memory.md`** (shape behavior), and the memory-shape guidance.
- **Contracts NOT touched**: intake `{domain}/{file-name}` Affected-Memory format, `context-loading` 2-hop convention, `_preamble.md:43` always-load routing — all preserved, because no files move and no sub-domains are introduced in this change.
- **Out of scope (excluded from true-impact via config)**: `fab/`, `docs/` line counts.

## Open Questions

- Where does the regen guard live — a `fab preflight` check, a PostToolUse hook on `docs/memory/**`, or a CI check? (See Assumption #3.)
- Should the shape bounds (~5/~12, depth ≤3) be hardcoded guidance or configurable in `fab/project/config.yaml`?
- Should `fab memory-index` run automatically (hook) on every memory write, or be an explicit step the hydrate skill calls?

> **Deferred to the follow-up change** (not blocking this one): the Internal-vs-External sub-domain path contract, and the design of the rebalancing actor. The latter is expected to be an **agent-driven skill** (`/fab-rebalance-memory`) that *instructs an agent* to move files and rewrite links — propose-then-apply — **not** a deterministic Go file-mover. See Assumption #5.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope this change to **B (generated `fab memory-index`) + C-detect (bounds as guidance + warnings)**; defer A (file-moving rebalance) + C-apply to a follow-up. | **User-confirmed scope** after the A/B/C explainer. Backed by adversarial analysis (B=61 > A=48 > C=41). One obvious interpretation now. | S:95 R:75 A:88 D:92 |
| 2 | Certain | **No sub-domains are introduced in this change**, so the intake `{domain}/{file-name}` contract, `_preamble` always-load, and `context-loading` 2-hop convention are all **preserved unchanged**. | Determined by the user-confirmed scope (file-mover is out) — there is nothing to decide here anymore. Internal-vs-External moves to the follow-up. | S:92 R:85 A:88 D:92 |
| 3 | Confident | Add a **`description:` frontmatter field** to memory files + backfill all ~21; do **not** auto-derive descriptions from H1/Overview. | Auto-derivation is lossy and breaks tables (literal pipes in `hydrate.md`) and degrades always-load routing. Verified: files carry no frontmatter today. This is the spine of B, not a nudge. | S:70 R:50 A:80 D:75 |
| 4 | Confident | Add a **regen sequencing guard** (preflight check or hook) so a forgotten `fab memory-index` can't silently rot the routing index. | No hand-edit remains to catch staleness once the index is generated. Exact mechanism (preflight vs hook vs CI) is an open question but that a guard is needed is clear. | S:60 R:65 A:70 D:55 |
| 5 | Certain | The rebalancer is the **existing `docs-reorg-memory` skill, enhanced** — NOT a new `/fab-rebalance-memory` and NOT a Go file-mover. This change adds the shape *vocabulary* (Shape Report, split/merge/flatten migration kinds, reserved-domain exemptions, `fab memory-index` index regen); the file-moving *apply* path is documented but exercised/hardened in the follow-up. | **User-confirmed**: improve the skill that already exists rather than spawn a duplicate. Splits are LLM-judged (agent work → markdown skill, per Pure Prompt Play), not mechanical. The detect side is what `tciy` ships. | S:92 R:80 A:90 D:90 |
| 6 | Confident | Ideal-shape bounds codified **in this change as SHOULD guidance + warnings only**: ~12 files/folder upper, ~5 lower, depth ≤3, sub-domain worth it at ≥8-file clusters. No enforcement/splitting. | Converges with Obsidian (≤3-4 levels, 5-10 groups, "let MOCs emerge"), monorepo, Starlight prior art. Warning is non-fatal; the index walk already computes counts. | S:70 R:80 A:75 D:75 |
| 7 | Certain | Implement `fab memory-index` reusing the **`prmeta` `Render`/`Gather`** pattern with byte-for-byte unit tests; update `_cli-fab.md`. | **Verified in codebase**: `src/go/fab/internal/prmeta` exists; `git-pr` already made this exact move for the Meta block. Constitution explicitly admits deterministic Go helpers + mandates `_cli-fab.md` updates. One obvious pattern to follow. | S:90 R:65 A:92 D:88 |
| 8 | Confident | Root `docs/memory/index.md` becomes **domains-only** (drop the inlined per-file column). | Measured: root index is near-zero churn already; dropping the file column makes it change only on domain birth/death. Prior-art consensus (monorepo root-README ToC indexes areas, not leaves). | S:70 R:70 A:80 D:75 |
| 9 | Certain | The follow-up change (A + C-apply: the agent-driven rebalancer) is **explicitly out of scope here** and tracked separately. | **User-confirmed deferral.** B is independently valuable and unblocks cheap, conflict-free file moves later. | S:90 R:82 A:85 D:88 |
| 10 | Confident | Reserve **`_shared/`** (cross-cutting) and **`_unsorted/`** (staging) as special domain names exempt from the fan-out warning. | Field-proven in loom. Avoids warning on a deliberately-cross-cutting bucket. | S:70 R:75 A:80 D:75 |
| 11 | Certain | Use **loom's stale root-index counts** (canvas 12→20, styling 7→15, testing 7→13) as a canonical regression fixture proving `fab memory-index` self-heals drift. | **Measured fact** (verified against loom on disk 2026-06-07), not a guess. | S:90 R:82 A:88 D:85 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved). The formerly-Unresolved sub-domain-path question (#2) is resolved by scoping the file-mover out of this change.
