# Intake: Ship Memory-File Template in Kit Cache

**Change**: 260616-2fm8-ship-memory-file-template
**Created**: 2026-06-16

## Origin

> Backlog `[2fm8]` (2026-06-16): "Shipped memory-file template in kit cache (refactor; depends on 260616-frlo FKF-contract-to-cache, ideally after it)."

One-shot invocation via `/fab-new 2fm8`. The backlog item is itself a fully-specified design (PROBLEM / FIX / TASKS / MEMORY / NON-GOALS / Depends-on), so the intake is a faithful transcription of that design grounded against the current repo state, not a fresh derivation.

**Pre-flight grounding performed during intake** (recorded so the apply-entry agent inherits it):

1. **Dependency `260616-frlo` has already shipped.** `src/kit/reference/fkf.md` exists in this worktree (commit `f34ab40e fix: Ship FKF normative contract to kit cache`). The `$(fab kit-path)/reference/fkf.md` §3.3 / §7 cites the template body needs are therefore **reachable**. The "ideally after it" sequencing constraint is satisfied.
2. **No `memory.md` template ships today.** `src/kit/templates/` contains only `intake.md`, `plan.md`, `status.yaml` — confirmed by directory listing. The PROBLEM statement is accurate.
3. **`_cli-fab.md` admission confirmed** at `src/kit/skills/_cli-fab.md:514`: "...there is **no** memory-file template carrying `type: memory` yet".
4. **Inlined memory shape confirmed** in `docs-hydrate-memory.md` (the literal code block at lines ~143–158, plus generate-mode and backfill-mode prose), `fab-continue.md` hydrate behavior (line 195), and both reorg skills.
5. **Task (c) design-Q premise corrected — see Design Decision below.** The backlog says "today the generator hardcodes it [`type: memory`]". This is **inaccurate**: `fab memory-index` (`src/go/fab/internal/memoryindex/memoryindex.go`) does **NOT** stamp `type: memory` onto topic files. It only **preserves it when present** on round-trip (per `memory-docs/templates.md:128`: "provides only the **preserve-when-present round-trip** ... does **not** bulk-stamp"). Topic-file `type:` is authored by the doc *skills* (the memory writers), per FKF §3.1. This corrects the design question's framing and resolves it (see Design Decisions).
6. **Related item `8fr5` is DONE** (PR #423, merged 2026-06-15). Its task (c) was "`_generation.md` memory template" — it did **NOT** inline a template; `_generation.md` references only the `intake.md`/`plan.md` templates and has no memory-shape block. So 2fm8 task (b)'s "confirm 8fr5 didn't already inline it" check passes: there is nothing to supersede, only the new template to wire in.

## Why

**Problem.** FKF §3.1 names "the memory-file template" as a load-bearing tooling stamper of the `type: memory` constant, and §10.4 references a memory-file template — but **none ships**. The kit ships only `templates/{intake,plan,status}`. As a workaround, the conventional memory-file shape (FKF frontmatter `type: memory` + `description:`, plus the `## Overview` / `## Requirements` / `### Scenario` / `## Design Decisions` body per `fkf.md` §3.3) is **inlined and duplicated** across `docs-hydrate-memory.md`, `fab-continue.md` hydrate behavior, and the reorg skills. `_cli-fab.md:514` explicitly admits the gap.

**Consequence if unfixed.** This is exactly the multi-source-of-truth drift FKF was designed to prevent: the memory-file shape lives in 3+ places, so any future change to the conventional shape must be hand-propagated to every inlined copy or they silently diverge. The contract (`fkf.md` §3) and its alleged "template" stamper are out of sync with reality.

**Why this approach.** Ship a single canonical template at `src/kit/templates/memory.md` and have the doc skills *read it on demand* via `$(fab kit-path)/templates/memory.md` — the exact pattern `_generation.md` / `_intake.md` already use for `$(fab kit-path)/templates/intake.md`. This:
- Collapses the duplicated shape to one source of truth (FKF's whole point).
- Realizes FKF §3.1's "the memory-file template" as a real artifact, closing the `_cli-fab.md` admission.
- Ships with **zero packaging/Go change** — `src/kit/` is copied verbatim by `just install` (rsync) and `just dist-kit` (`cp -a`), the same mechanism `260616-frlo` already verified for `reference/fkf.md`.
- Adds **no migration** — skills re-deploy via `fab sync`; the template is read from the version-pinned cache, so existing projects pick it up on their next sync.

## What Changes

### A. New canonical template — `src/kit/templates/memory.md`

Create the single source-of-truth memory-file template. Structure (from `fkf.md` §3.3, the conventional shape):

```markdown
---
type: memory
description: "{One-line summary used by the generated domain-index row.}"
---
# {File Name}

**Domain**: {domain}

## Overview
<!-- 1-2 sentences describing what this file covers. -->

## Requirements
### Requirement: {Name}
{RFC 2119 text: MUST / SHALL / SHOULD / MAY}

#### Scenario: {Name}
- **GIVEN** {precondition}
- **WHEN** {action}
- **THEN** {expected outcome}

## Design Decisions
### {Decision Title}
**Decision**: {chosen approach}
**Why**: {rationale}
**Rejected**: {alternative and why it was worse}
*Introduced by*: {change-name}

<!-- Cross-links to other memory files use the bundle-relative form (FKF §7):
     See [migrations](/distribution/migrations.md). Links OUT of the bundle
     (source, specs, URLs) stay repo-relative/absolute-URL. -->
```

Constraints (all from `fkf.md`, the now-reachable reference):
- **Leading FKF frontmatter** — `type: memory` as a constant + `description:` placeholder (§3.1–§3.2).
- **Conventional body skeleton** — Overview / Requirements (+ Scenario) / Design Decisions (§3.3).
- **NO `## Changelog` section** — the single biggest FKF divergence; change history lives in the per-folder generated `log.md` (§3.3, §6).
- **Bundle-relative cross-link example** — demonstrate the `](/{domain}/{file}.md)` form (§7).
- The body **SHOULD cite** `$(fab kit-path)/reference/fkf.md` §3.3 for the conventional headings (in a guidance comment) rather than re-describing the rules — the template is a scaffold, the reference is the contract. Mirrors how `intake.md`/`plan.md` templates use guidance comments that the agent strips on fill.

> **Headings are SHOULD, not MUST** (`fkf.md` §3.3): a conforming memory file need not have every section. The template scaffolds the full shape; a small reference-pointer file legitimately omits a GIVEN/WHEN/THEN scenario. The guidance comment must preserve this nuance, not imply the sections are mandatory.

### B. Repoint the doc skills to READ the template

Replace each inlined shape with an instruction to read `$(fab kit-path)/templates/memory.md` on demand (the `$(fab kit-path)/templates/intake.md` pattern). Three skill files:

1. **`src/kit/skills/_generation.md`** — the Intake/Plan generation procedures. There is no inlined memory shape today (confirmed); the wiring here is to have any memory-authoring path reference the template. Treat this as the lightest touch of the three: confirm 8fr5 left nothing to supersede, then point memory authoring at the template. *(If `_generation.md` has no memory-authoring section to repoint, this reduces to a no-op note — the real consumers are docs-hydrate-memory and fab-continue. To be decided at apply against the actual file content.)*

2. **`src/kit/skills/docs-hydrate-memory.md`** — **all three modes**:
   - **generate** (Step 3, the literal ```markdown block at ~lines 143–158) — replace the inlined block with "read the shape from `$(fab kit-path)/templates/memory.md`".
   - **ingest** — same: author from the template rather than the inlined description.
   - **backfill** — backfill is a *pure frontmatter operation* (it only adds `type: memory` + `description:` to existing files, body-preserving). It should reference the template for the **frontmatter shape** specifically, NOT impose the body skeleton on existing files. Preserve the backfill scope contract (`docs-hydrate-memory.md:178/190/191`) — do not let the repoint widen backfill into a body rewrite.

3. **`src/kit/skills/fab-continue.md`** hydrate behavior (line 195) — repoint the "create new files (each carrying FKF frontmatter...)" prose to read the template for the new-file shape. Preserve all the surrounding hydrate contracts (set-summary for log.md, bundle-relative links, merge-without-duplication, refuse-before-regen guard, shape SHOULD guidance).

> **Reorg skills** (`docs-reorg-memory.md`) — the backlog PROBLEM names them as a duplication site, but reorg **moves** files (preserving FKF frontmatter byte-for-byte) and only *stamps* `type: memory` + authors `description:` on genuinely-new split files; it does not author full memory bodies. The reorg's frontmatter handling is already correct and minimal. **Tentative scope call**: repoint reorg's new-file frontmatter authoring to cite the template for consistency *only if* it reduces duplication without expanding reorg's responsibilities; otherwise leave reorg as-is (it authors frontmatter, not bodies). Decide at apply against the actual prose. The backlog's TASKS list (b) names only `_generation`/`docs-hydrate-memory`/`fab-continue` — reorg is named in PROBLEM but NOT in the task list, so it is out of the mandatory repoint set.

### C. Resolve task (c) — `fab memory-index` stamping (design decision)

See Design Decisions below. **Resolution: keep `fab memory-index`'s existing preserve-when-present round-trip; the template + doc skills are the stampers.** No Go change. This is the lowest-risk reading and aligns with FKF §3.1 (writers stamp; the generator round-trips). The backlog's "today the generator hardcodes it" premise was inaccurate (the generator never bulk-stamps topic-file `type:`).

### D. SPEC mirrors (Constitution rule)

Per Constitution ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`"), mirror every repointed skill:
- `docs/specs/skills/SPEC-_generation.md`
- `docs/specs/skills/SPEC-docs-hydrate-memory.md`
- `docs/specs/skills/SPEC-fab-continue.md`

(`_cli-fab.md` is **excluded** from SPEC mirrors — there is no `SPEC-_cli-fab.md`; confirmed by the backlog note. If the reorg skills end up touched per the Tentative call in B, add `SPEC-docs-reorg-memory.md` then.)

### E. Close the `_cli-fab.md` admission

Update `src/kit/skills/_cli-fab.md` (~line 514) to remove the "there is **no** memory-file template carrying `type: memory` yet" admission — the template now exists. (`_cli-fab.md` is exempt from SPEC mirrors.)

## Affected Memory

- `memory-docs/templates.md`: (modify) The file's `description:` currently claims to cover "the memory file format"; its body (line 11) says "This doc covers the two artifact templates (intake, plan) and the memory file format". Update to record that the memory file format is now a **shipped template** (`templates/memory.md`), the third artifact template, read on demand by the doc skills via `$(fab kit-path)/templates/memory.md`.
- `distribution/kit-architecture.md`: (modify) The `templates/` tree enumeration (lines 56–59) lists `intake.md`, `plan.md`, `status.yaml` — add `memory.md`. Also update the `templates/` "Replaced" note (line 249) if it enumerates contents.

## Impact

**Files touched (apply scope):**
- `src/kit/templates/memory.md` — **new** (the canonical template)
- `src/kit/skills/_generation.md` — repoint (possibly no-op; see A/B note)
- `src/kit/skills/docs-hydrate-memory.md` — repoint all 3 modes
- `src/kit/skills/fab-continue.md` — repoint hydrate new-file shape
- `src/kit/skills/_cli-fab.md` — remove the no-template admission (~line 514)
- `src/kit/skills/docs-reorg-memory.md` — **conditional** (Tentative; only if it reduces duplication cleanly)
- `docs/specs/skills/SPEC-_generation.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-fab-continue.md` — synchronous SPEC mirrors
- `docs/memory/memory-docs/templates.md`, `docs/memory/distribution/kit-architecture.md` — hydrate-stage memory updates

**No code/packaging change:** zero Go, zero migration. `src/kit/` ships verbatim via `just install` rsync + `just dist-kit cp -a` (verified by `260616-frlo`). Existing projects receive the template on their next `fab sync` from the version-pinned cache.

**Dependencies:** `260616-frlo` (shipped) for the reachable `$(fab kit-path)/reference/fkf.md` §3.3 cite. Related: `8fr5` (done) — its task (c) is realized/superseded here.

**Constitution touchpoints:** II (Docs are source of truth — collapses duplication), III (idempotent — template read is pure), V (portability — kit content in cache), plus the skill→SPEC-mirror constraint (drives D).

## Open Questions

- None blocking. The one genuine design question (task c, memory-index stamping) was resolved during intake grounding (see Design Decisions). The `_generation.md` repoint scope (A/B) and the reorg-skills repoint scope (B) are Tentative apply-time judgments against actual file content, not blocking Unresolved decisions — apply decides-and-records them.

## Assumptions

<!-- STATE TRANSFER: feeds the apply-entry agent. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship the template at `src/kit/templates/memory.md`; doc skills read it via `$(fab kit-path)/templates/memory.md` | Backlog FIX states this verbatim; mirrors the existing `intake.md` template-read pattern in `_generation.md`/`_intake.md` | S:100 R:80 A:95 D:95 |
| 2 | Certain | No Go / packaging / migration change | Backlog NON-GOALS + verified `src/kit/` ships verbatim (rsync/cp -a) per `260616-frlo`; skills re-deploy via `fab sync` | S:95 R:85 A:95 D:90 |
| 3 | Certain | Template body excludes `## Changelog`; carries `type: memory` + `description:` + Overview/Requirements(+Scenario)/Design Decisions + bundle-relative link example | Directly dictated by `fkf.md` §3.1/§3.3/§7 (the now-reachable reference) | S:95 R:85 A:100 D:95 |
| 4 | Certain | SPEC mirrors for `_generation`, `docs-hydrate-memory`, `fab-continue`; `_cli-fab.md` exempt | Constitution skill→SPEC-mirror rule + backlog task (d) note (`_cli-fab.md` excluded); all three SPEC files exist | S:95 R:75 A:100 D:95 |
| 5 | Confident | Task (c) resolved: keep `fab memory-index`'s preserve-when-present round-trip; template + doc skills are the stampers — no Go change | Codebase check: generator does NOT bulk-stamp topic-file `type:` (only round-trips when present, per `memory-docs/templates.md:128`); backlog's "generator hardcodes it" premise was inaccurate; FKF §3.1 assigns stamping to writers | S:70 R:80 A:90 D:85 |
| 6 | Confident | Template body cites `$(fab kit-path)/reference/fkf.md` §3.3 in a guidance comment rather than re-describing the heading rules | Backlog FIX says the body SHOULD cite §3.3; the cite target is reachable (frlo shipped); mirrors intake/plan templates' guidance-comment style | S:80 R:90 A:85 D:80 |
| 7 | Tentative | `_generation.md` repoint may be a near-no-op — its real memory consumers are docs-hydrate-memory and fab-continue; `_generation.md` has no inlined memory-body block today | Confirmed `_generation.md` references only intake/plan templates and 8fr5 left no inlined memory shape; exact wiring decided at apply against file content | S:65 R:80 A:75 D:60 |
| 8 | Tentative | Reorg skills (`docs-reorg-memory.md`) repointed only if it reduces duplication without widening reorg's responsibilities; else left as-is | Backlog names reorg in PROBLEM but NOT in the task (b) list; reorg moves files (preserving frontmatter) and stamps only new split files — its frontmatter handling is already minimal/correct | S:60 R:75 A:70 D:55 |
| 9 | Tentative | Backfill mode references the template for the **frontmatter shape only**, not the body skeleton — preserving backfill's pure-frontmatter-operation contract | `docs-hydrate-memory.md:178/190/191` defines backfill as body-preserving; the repoint must not widen it into a body rewrite | S:70 R:70 A:80 D:65 |

9 assumptions (4 certain, 2 confident, 3 tentative, 0 unresolved).
