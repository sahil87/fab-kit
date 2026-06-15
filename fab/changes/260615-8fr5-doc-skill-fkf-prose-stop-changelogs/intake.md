# Intake: FKF Change 3/4 — Doc-Skill Prose → Author FKF, Stop Writing Per-File Changelogs

**Change**: 260615-8fr5-doc-skill-fkf-prose-stop-changelogs
**Created**: 2026-06-15

## Origin

> Backlog `[8fr5]`: *FKF Change 3/4 (refactor, depends on Change 2): Doc-skill prose → author FKF, stop writing per-file changelogs. Skills can now truthfully say 'history lives in generated log.md' because Change 2 emits it.* Tasks (a) `fab-continue.md` hydrate behavior — author `type:`+`description:` frontmatter, CALL `fab status set-summary` instead of appending a `## Changelog` row, bundle-relative memory↔memory links; (b) `docs-hydrate-memory.md` (all 3 modes) — stamp `type: memory`, drop `## Changelog` from the file template, bundle-relative links; (c) `_generation.md` memory template; (d) `_review.md` if it touches changelogs. SYNCHRONOUS SPEC mirrors per Constitution rule. Spec: `docs/specs/fkf.md` §3.3, §7. Depends-on: Change 2. Context PR #419.

Invoked via `/fab-new 8fr5` (one-shot, backlog-ID input). This is the third implementation change of the FKF migration tracked in `docs/specs/fkf.md` §10:

- **Change 1/4** (`5943`, shipped) — added the `.status.yaml` `summary:` field + `fab status set-summary`/`get-summary` verbs + migration `2.4.2-to-2.5.0`.
- **Change 2/4** (`bmzo`, shipped — commit `02f3ab28` / PR #422) — taught `fab memory-index` to emit per-folder `log.md` (C-lite git-history + `summary` join) and stamp FKF frontmatter on the root index.
- **Change 3/4** (this change, `8fr5`) — updates the **doc-skill prose** (the canonical `src/kit/skills/` sources) so skills author FKF frontmatter, stop *writing new* `## Changelog` sections, call `fab status set-summary`, and use bundle-relative memory↔memory links.
- **Change 4/4** (not yet drafted) — strips the `## Changelog` sections from the 20 *existing* memory files (this change only stops *new* writes; the 20 existing files are untouched here).

> **Source vs. installed-binary note**: the `fab status set-summary` verb that task (a) wires the prose to call exists in **source** (`src/go/fab/cmd/fab/status.go`, `_cli-fab.md`, the specs, and memory — shipped in `5943`/`bmzo`), but is **not** in the locally-installed `fab 2.4.2` binary (which predates those merges). This is a known source-vs-binary skew and does **not** block this change: this change edits *prose that instructs an agent to call the verb*; it does not itself invoke `set-summary`. The prose is correct against the canonical source contract.

## Why

1. **The problem.** Today the doc skills (`fab-continue` hydrate, `docs-hydrate-memory`) instruct the agent to *write a `## Changelog` table into each memory file*. FKF (`docs/specs/fkf.md` §3.3) **removes** per-file changelog tables — change history now lives in the per-folder generated `log.md` (§6, C-lite). After Change 2 shipped, `fab memory-index` actually emits that `log.md`, so the skills can now *truthfully* say "history lives in the generated `log.md`." Until the prose is updated, skills keep instructing agents to hand-write changelogs that FKF has retired — directly contradicting the shipped format.

2. **The consequence if unfixed.** New hydrate runs would keep appending `## Changelog` rows to memory files (re-introducing the very same-day merge-conflict surface FKF's C-lite model eliminates — `docs/specs/fkf.md` §6.1), and would keep writing relative memory↔memory links that break when `docs-reorg-memory` moves files between domains (§7). The generated `log.md` and the hand-written `## Changelog` would coexist and disagree. The prose would also fail to stamp the now-**required** `type: memory` frontmatter (§3.1) and would not author the `summary:` field that `log.md` generation reads (§6.3).

3. **Why this approach.** Prose-only change to the canonical skill sources is the minimal, correct unit: FKF is the contract (already specced), Changes 1–2 supplied the mechanism (the field, verbs, and generator), and this change aligns the *instructions agents follow* with that mechanism. It is split from Change 4 (strip existing changelogs) deliberately — stopping new writes and removing old content are independently reviewable, and Change 4 is a bulk data edit with a different risk profile.

## What Changes

All edits are to canonical **`src/kit/skills/`** sources (never the gitignored `.claude/skills/` deployed copies — Constitution V). Each skill-file edit pairs with a **synchronous** `docs/specs/skills/SPEC-*.md` mirror update (Constitution: "Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md` file").

### (a) `src/kit/skills/fab-continue.md` — hydrate behavior

The hydrate-step prose (≈ line 195) currently reads, in part:

> "...update existing (Requirements, Design Decisions, **Changelog**, keep `description:` accurate)..."

Three edits to the hydrate behavior:

1. **Drop `Changelog` from the "update existing" section list.** Memory files no longer carry a `## Changelog` section. The merge-without-duplication contract (check for an existing entry by change name, update in place) stays for Requirements/Design Decisions.
2. **Author FKF frontmatter on new/updated files.** New memory files must carry both `type: memory` (constant, §3.1) and a curated `description:` one-liner (§3.2). Today the prose mentions only `description:`; add `type: memory`.
3. **Call `fab status set-summary` instead of appending a changelog row.** Replace the per-file changelog write with a single `fab status set-summary {change} "<one-line what-changed>"` call — the C-lite source line (§6.3) that `fab memory-index` joins with git history when generating `log.md`. The summary is authored **once** during the change at hydrate (the §6.3 "authored at hydrate" path).
4. **Bundle-relative memory↔memory links.** Any memory↔memory cross-link the hydrate prose instructs writing must use the bundle-relative `/...` form (§7), not relative paths. Links *out* of the bundle (to source, specs, URLs) stay repo-relative/absolute-URL.

### (b) `src/kit/skills/docs-hydrate-memory.md` — all 3 modes (ingest / generate / backfill)

1. **Ingest mode** (Step 3, ≈ line 87): the "create with ... Overview, Requirements, Design Decisions, **Changelog** sections" instruction drops `Changelog`; add `type: memory` to the authored frontmatter (alongside the existing `description:` authoring at ≈ line 90); memory↔memory links bundle-relative.
2. **Generate mode** (the file template, ≈ lines 150–165): remove the literal `## Changelog` block from the generated-file template:
   ```markdown
   ## Changelog
   | Date | Change |
   |------|--------|
   | {DATE} | Generated from code analysis |
   ```
   and stamp `type: memory` into the template's frontmatter.
3. **Backfill mode**: backfill is **body-preserving** (it only prepends/edits `description:` frontmatter on files missing it). It must now also stamp `type: memory` when adding frontmatter, so a backfilled file is FKF-conforming (§2 item 2 — `type: memory` is required). It must NOT strip existing `## Changelog` bodies (that is Change 4's job; backfill stays body-preserving).
4. **Index Ownership / `description:` authoring prose** stays as-is — only `type:` stamping, `## Changelog` removal, and bundle-relative links are added.

### (c) `src/kit/skills/_generation.md` — memory template

**Investigation result (to be confirmed in plan):** `_generation.md` contains *only* the Intake Generation Procedure and the Plan Generation Procedure. It has **no memory-file template and no `## Changelog` reference** (its only `memory` mentions are "config/constitution/memory" context and the intake "Affected Memory" section). The memory-file *template* the backlog cites is not in `_generation.md`; the canonical generated-file template lives in `docs-hydrate-memory.md` generate mode (task b). There is **no separate memory template file** in `src/kit/templates/` either (only `intake.md`, `plan.md`, `status.yaml`).

→ Expected outcome: task (c) is a **no-op** (nothing to change in `_generation.md`), to be confirmed by a close read at apply. If a memory-template fragment is found, apply it the same way as (b). Consequently `SPEC-_generation.md` likely needs **no** mirror edit.

### (d) `src/kit/skills/_review.md` — only if it touches changelogs

**Investigation result (to be confirmed in plan):** `_review.md` has **no `## Changelog` writing**. It *reads* memory files for consistency checks ("inconsistencies with documented patterns") and writes a `## Deletion Candidates` section (unrelated to changelogs). The backlog hedges this task with "if it touches changelogs" — it does not.

→ Expected outcome: task (d) is a **verify/no-op**. Consequently `SPEC-_review.md` likely needs **no** mirror edit. (If review prose elsewhere references writing a `## Changelog`, update it to point at the generated `log.md`.)

### SPEC mirrors (Constitution rule — synchronous)

Update the SPEC mirror for **every** skill file actually changed:
- `docs/specs/skills/SPEC-fab-continue.md` — mirror the hydrate-behavior edits (a).
- `docs/specs/skills/SPEC-docs-hydrate-memory.md` — mirror the 3-mode edits (b).
- `docs/specs/skills/SPEC-_generation.md` — only if (c) turns out non-empty.
- `docs/specs/skills/SPEC-_review.md` — only if (d) turns out non-empty.

> Out of scope here: `docs/specs/fkf.md` itself (the contract — already written and authoritative; this change conforms prose *to* it, not edits it).

## Affected Memory

Per the backlog (`Memory: memory-docs/hydrate.md, memory-docs/templates.md, _shared/context-loading.md`):

- `memory-docs/hydrate.md`: (modify) — the hydrate skills' documented behavior: stop writing `## Changelog`, author `type: memory` + `description:`, call `fab status set-summary`, bundle-relative links.
- `memory-docs/templates.md`: (modify) — the memory-file template description: drop `## Changelog` from the documented template shape, add `type: memory` to the frontmatter contract. (`summary:`/`log.md` C-lite detail already documented here from `5943`/`bmzo`.)
- `_shared/context-loading.md`: (modify) — if the always-load / memory-authoring context-loading prose references the `## Changelog` section or relative links, align it to the FKF format.

> NOTE: All three are `(modify)` — they document behavior this change updates. Per Constitution II, hydrate (the `/fab-continue` hydrate stage of *this* change) authors these memory updates; the changes here stop writing `## Changelog` and use bundle-relative links — so this change's *own* hydrate must already follow the new FKF prose (eat-your-own-dogfood). These memory files **carry `## Changelog` sections today** (part of the 20) — this change does not strip them (Change 4); it only stops adding new ones and updates their *content* to describe the new behavior.

## Impact

- **Code areas**: `src/kit/skills/{fab-continue,docs-hydrate-memory,_generation,_review}.md` (canonical sources) + their `docs/specs/skills/SPEC-*.md` mirrors. Pure prose/markdown — **no Go code, no CLI signature changes** (Change 1 already added the verb; the `_cli-fab.md` reference is already accurate).
- **Dependencies**: depends on Change 2 (`bmzo`, shipped) for the `log.md` emitter, and Change 1 (`5943`, shipped) for `set-summary`. Both are merged to `main`; the dependency is satisfied. (Installed binary `fab 2.4.2` lags source on `set-summary`, but this change writes prose, not a live `set-summary` call — see Origin note.)
- **Downstream**: Change 4/4 (strip existing `## Changelog` from the 20 memory files) is the follow-on; this change is a clean prerequisite — once new writes stop, Change 4 is a one-time bulk strip.
- **Verification**: after `fab sync`, the deployed skills no longer instruct writing `## Changelog`; a hydrate run authors `type: memory` + `set-summary` + bundle-relative links. SPEC mirrors match their sources.
- **Risk / edge cases**: (1) the (c)/(d) no-op finding must be confirmed by a close read at apply — if either skill *does* touch changelogs in prose the grep missed, scope expands to include its SPEC mirror; (2) ensure backfill mode stays body-preserving (does not strip existing changelog bodies) — only adds `type:`; (3) keep the merge-without-duplication contract intact when removing the `Changelog` mention from `fab-continue` hydrate.

## Open Questions

- Confirm at apply (close read) that `_generation.md` and `_review.md` carry no changelog-writing or memory-template prose — i.e., that tasks (c) and (d) are genuine no-ops. (Strong evidence they are; flagged as Confident, not blocking.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Target memory format is dictated by `docs/specs/fkf.md` (drop `## Changelog`, add `type: memory`, `description:` required, bundle-relative `/...` links, `summary:` as `log.md` source) | The spec is the authoritative, already-shipped contract; this change conforms prose to it, no design latitude | S:95 R:80 A:100 D:95 |
| 2 | Certain | Edits target canonical `src/kit/skills/` sources; `.claude/skills/` is gitignored deployed output, never hand-edited | Constitution V is explicit | S:100 R:90 A:100 D:100 |
| 3 | Certain | Every changed skill file pairs with a synchronous `docs/specs/skills/SPEC-*.md` mirror update | Constitution "Changes to skill files MUST update the corresponding SPEC-*.md" | S:100 R:75 A:100 D:95 |
| 4 | Confident | Change type is `refactor` (re-aligning prose to an already-shipped format; no new user-facing capability) | Backlog labels it "(refactor)"; the keyword hook should infer `refactor` — will verify and override if wrong | S:85 R:90 A:90 D:85 |
| 5 | Confident | Task (c) `_generation.md` is a **no-op** — it has no memory template and no `## Changelog`; the generated-file template lives in `docs-hydrate-memory.md` (task b) | Full case-insensitive grep of `_generation.md` found only intake/plan generation; `templates/` has no memory template | S:80 R:70 A:90 D:75 |
| 6 | Confident | Task (d) `_review.md` is a **verify/no-op** — it reads memory and writes `## Deletion Candidates`, but writes no `## Changelog` | Backlog hedges "if it touches changelogs"; grep found no changelog-writing prose | S:80 R:70 A:90 D:80 |
| 7 | Confident | `fab status set-summary` is the correct verb to wire into `fab-continue` hydrate prose despite its absence from the installed `fab 2.4.2` binary | Verb exists in source (`status.go`, `_cli-fab.md`, specs, memory — shipped `5943`/`bmzo`); prose is correct against canonical contract; this change writes prose, not a live call | S:90 R:85 A:95 D:90 |
| 8 | Confident | `summary:` is authored **once at hydrate** by `fab-continue` (the §6.3 "authored at hydrate" path), not at intake | §6.3 lists both, but hydrate is where this change wires the call (task a is explicitly hydrate behavior) | S:75 R:80 A:85 D:80 |
| 9 | Confident | This change's affected-memory set is exactly the 3 files the backlog names (`hydrate.md`, `templates.md`, `_shared/context-loading.md`), all `(modify)` | Backlog enumerates them; matches the behavior actually changing | S:85 R:75 A:90 D:85 |
| 10 | Tentative | `_shared/context-loading.md` needs editing only if it references `## Changelog` or relative links; otherwise it stays untouched | Listed by backlog, but its relevance to *this* change is conditional on its current content — confirm at hydrate | S:60 R:75 A:70 D:65 |

10 assumptions (3 certain, 6 confident, 1 tentative, 0 unresolved). Run /fab-clarify to review.
