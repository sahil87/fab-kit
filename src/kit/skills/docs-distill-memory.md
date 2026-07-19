---
name: docs-distill-memory
description: "Rewrite existing memory files to the FKF present-truth style — strip transition narration and superseded-state prose, cap descriptions, relocate rationale into Design Decisions. Optional <domain>: named forces a full read of that one domain; omitted surveys all domains and loops every flagged one sequentially. One domain per approval unit (per-domain gate retained); read-only until you approve."
---

# /docs-distill-memory [<domain>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

## Contents

- Purpose
- Arguments
- Pre-flight
- Context Loading
- Behavior
- Output
- Error Handling
- Key Properties

---

## Purpose

Rewrite an existing `docs/memory/` domain's topic files to the **FKF present-truth style** (`$(fab kit-path)/reference/fkf.md` §3.2, §3.3). The FKF present-truth rules govern what memory writers produce *going forward*; a corpus authored before them accumulates transition narration, superseded-state prose, and over-cap / change-id-carrying `description:` frontmatter. This skill cleans that existing corpus — it is the remediation counterpart to the forward-looking writers (hydrate, `/docs-hydrate-memory`, `/docs-reorg-memory`), which already emit current truth.

**One domain per approval/apply unit, propose-then-apply.** The skill runs **read-only analysis** over a domain's topic files — the domain named explicitly, or (with no `<domain>`) each flagged domain in turn as the heuristic **survey** (Behavior Step 0) loops all of them sequentially (Behavior Step 6) — produces a **per-file proposed-rewrite report**, and applies **only on explicit user approval** — the same posture as `/docs-reorg-memory` (its report → confirm-and-apply shape). These files encode load-bearing behavioral contracts, so a human approves **per domain**, seeing per-file diffs. It is **not** an autonomous bulk rewriter. "One domain" is a property of the analysis+apply/approval unit, not the invocation: exactly one domain is read-in-full, reported, approved, and rewritten as a unit — a no-arg invocation iterates that unit over every flagged domain, an explicit `<domain>` runs it once.

> **Distinct from the sibling doc skills.** `/docs-reorg-memory` reorganizes **structure** (splits/merges/moves + link rewrites) — it never rewrites body prose to a style. `/docs-hydrate-memory` backfill mode is **body-preserving** (adds frontmatter only); its ingest/generate modes author *new* content. `/fab-continue` hydrate writes each change's delta as current truth but only touches the sections its change affects. This skill is the only one that rewrites **existing** body prose to the present-truth style across a whole domain.

### What a rewrite does (FKF §3.2 / §3.3)

The normative source is the shipped FKF extract `$(fab kit-path)/reference/fkf.md` — cite it (deployed skills reach the extract; the dev-repo `docs/specs/fkf.md` is absent in user repos). A rewrite:

- **Removes transition narration** — no "renamed X→Y in {id}", no "this inverts/supersedes {id}'s claim", no "was `old.value`", no "superseding the historical …".
- **Removes superseded-state descriptions** — the body carries only what IS. Previous states belong to the per-folder generated `log.md`, git history, and archived change folders.
- **Strips change-id heading suffixes** — a heading is `## Dispatch States`, never `### Dispatch States (xu0k)` or `## xu0k — dispatch states`; the token is removed (kept as a trailing body citation when provenance matters). Recognition is registry-gated (full `YYMMDD-XXXX-slug` always; bare 4-char id only when registry-plausible).
- **Dedupes byte-identical duplicate headings/blocks** — the later of a byte-identical duplicated block is removed. **Near-duplicates are flagged for manual review, never auto-merged** — content judgment stays with the human gate.
- **Rewrites Design-Decisions changelog bullets** — a `- **{change-id} — retired X**`-shaped bullet inside `## Design Decisions` (the shape §3.3 bans there) is rewritten to the four-field entry (durable decision) or removed (pure change history in `log.md`/git). **Never fabricates rationale** — an entry with no derivable `Why`/`Rejected` carries only the fields that exist (Decision + *Introduced by*).
- **Relocates operational TODOs → `fab/backlog.md`** — follow-up work items (TODOs, "still needs X", next-step checklists) are never memory-body content (§3.3). They are **relocated to the backlog, never deleted**.
- **Keeps allowed provenance** — trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions. Per §3.3, a 6-char `(id)` cheaply defends a deliberate, easily-"fixed"-away behavior against future regressions. **Bare 4-char ids count the same as dated ids** — in trailing-citation position they stay; woven into narration they go with the narration.
- **Fixes `description:` frontmatter** — strips change-ids (§3.2 bans them: no trailing `— xu0k`-style suffix, no `(d9rs)`-style citation) and compresses an over-cap description to the **≤500-character** routing-signal shape, moving displaced routing-irrelevant detail into the body where it is not already present.

### Rationale-preservation guard (the critical constraint)

**Token savings come from dropping narration, NEVER rationale.** Per FKF §3.3 verbatim: *"'Don't re-break this' content lives in Design Decisions' `Why` / `Rejected` as durable, present-tense design intent — a rejected alternative is a design *fact*, not transition narration."*

- Deliberate-behavior / "don't re-break this" content is **RELOCATED** into `## Design Decisions` (`Why` / `Rejected`) as present-tense intent — it is **never deleted**. This repo's history shows agents repeatedly "fixing" deliberate behavior (e.g. the Copilot poll-predicate); the distilled file must retain those defenses.
- **Deletion is safe only** for narration whose content is **already recorded elsewhere** — the per-folder `log.md`, git history, or archived change folders. Content recorded nowhere else and carrying intent is **relocated, not dropped**.

When you cannot tell whether a narration line encodes durable intent, treat it as rationale and relocate it — the safe default preserves, it does not delete.

### Generated files & the tombstone exemption

- **Never hand-edit generated files** — `index.md` (root / domain / sub-domain tiers) and `log.md` are written solely by `fab memory-index` (FKF §5, §6). This skill regenerates them via `fab memory-index` after applying rewrites; it never edits their rows.
- **`log.seed.md` is a curated read-only SEED INPUT, not a generated file** — `fab memory-index` *reads* it during the seed-merge but never *writes* it (like `description:` frontmatter, it is a gathered input; the generator stays the sole writer of `log.md`). It is nonetheless **excluded from distillation**: its body *is* a citation-carrying seed ledger of pre-FKF history in the §6.2 entry format, not topic-file prose — the same exclusion posture as `removed-domains.md` below. Skip it entirely; never rewrite it.
- **`docs/memory/_shared/removed-domains.md` is EXEMPT** from rewrite — the §3.3 tombstone carve-out: its body *is* removal records, a citation-carrying tombstone ledger, not transition narration. Skip it entirely. (fab-kit's own tree has no such file; the exemption matters in user projects, where `/docs-reorg-memory` authors it.)

---

## Arguments

- **`<domain>`** *(optional)* — a single memory domain to distill, named by its `docs/memory/` folder (e.g. `pipeline`, `distribution`, `runtime`, `_shared`). Case-insensitive substring match against domain folder names; an ambiguous or unknown name is handled per Error Handling. The skill rejects a **multi-domain** invocation (to distill several domains, run no-arg — it loops all flagged domains — or run it once per named domain).
  - **When omitted**, the skill runs **survey mode** (Behavior Step 0): a cheap heuristic scan over all domains that reports per-domain status, builds the flagged-domain worklist, and then **loops every flagged domain sequentially** (Behavior Step 6) in `docs/memory/index.md` domain-table order — running the one-domain flow (full read → per-file report → **per-domain approval** → apply → regen) as the loop body per domain. No-arg no longer aborts and no longer stops after one domain.
  - **When given explicitly**, the domain is the **override** — the skill skips the survey heuristics, forces a full read of that domain (Behavior Step 1 onward), and runs the one-domain flow **once** (no loop).

**One domain per approval/apply unit, iterated within a single invocation.** "One domain" is a property of the approval/apply unit — one domain is read-in-full, reported, approved, and rewritten as a unit, with the human seeing that domain's per-file diffs — **not** of the invocation. A no-arg invocation iterates that unit over every flagged domain; an explicit `<domain>` runs it once.

---

## Pre-flight

1. `docs/memory/index.md` must exist and be readable.
2. **Resolve the domain(s)**:
   - **`<domain>` omitted** → run **survey mode** (Behavior Step 0). The survey builds the flagged-domain worklist the all-domains loop (Behavior Step 6) iterates; the topic-file precondition below is checked per domain as the loop reaches it. A no-arg invocation never aborts for a missing argument.
   - **`<domain>` given** → the named domain folder `docs/memory/{domain}/` must exist and contain at least one topic file (a non-`index.md`/`log.md`/`log.seed.md` `.md` file). An ambiguous or unknown name is handled per Error Handling. Runs the one-domain flow once (no loop).

If check 1 fails — or a resolved/named domain folder is missing or has no topic files — STOP with the matching Error Handling message.

---

## Context Loading

This section is the skill-file override the `_preamble.md` §1 contract keys on — the skill does **NOT** load the always-load layer. It requires **no active change, config, or constitution**. For each target domain (named explicitly, or reached in turn as the no-arg loop iterates the survey worklist), it reads only:

- `docs/memory/index.md` and the target domain's `docs/memory/{domain}/index.md` (and any sub-domain `index.md`) — the domain landscape.
- Every topic file in the target domain (recursing into sub-domains) — the rewrite subjects.
- `$(fab kit-path)/reference/fkf.md` — the shipped normative extract (§3.2 `description` rules incl. the 500-char cap and change-id ban; §3.3 present-truth body style incl. the tombstone carve-out). Read it so every proposed rewrite cites the deployed rule, not a remembered one.

**Survey mode reads the machine surface up front, not the corpus.** On a no-arg invocation (Behavior Step 0), *before* any domain's full read the survey runs a single `fab memory-index --check --json` and reads its JSON `malformed[]`/`warnings[]` arrays to count flagged files per domain — it does **not** read every topic file's frontmatter and body. This is the canonical machine-surface path; only the **older-binary fallback** (Step 0) reverts to the legacy all-domains read-only grep scan (each domain's `index.md` + enough of every topic file's `description:` and body to run the narration-marker grep, recursing sub-domains, honoring the distillation exclusion set). Either way the all-domains survey is not a full Step 1 read; the full read is confined to each domain as the loop (Behavior Step 6) reaches it. An explicit `<domain>` skips the survey and reads only the target-domain set above.

For the `fab memory-index --check --json` shape (the `malformed[]`/`warnings[]` kinds the survey aggregates), the exit tiers, and the refuse-before-regen pointer, consult **`_cli-fab` § fab memory-index** by in-body pointer (below) — it is not pre-loaded.

---

## Behavior

### Step 0: Survey mode (no-arg only)

Runs **only when `<domain>` was omitted**. An explicit `<domain>` skips this step entirely and goes straight to Step 1 (the override forces a full read regardless of survey heuristics, and does NOT loop — see Step 6).

The survey is a **cheap machine-surface read** over every domain — it does NOT do the full Step 1 read. Its job is to rank the flagged domains and drive the all-domains loop, not to classify exhaustively; the full read still runs once per domain inside the loop. It reports per domain in the order of `docs/memory/index.md`'s domain table (deterministic, matches the user-facing landscape).

**Signal source: one `fab memory-index --check --json` invocation.** The survey runs `fab memory-index --check --json` **once** and consumes its structured output — the **canonical** signal source (`_cli-fab` § fab memory-index), not an agent-side grep of frontmatter and bodies. Per-domain **flagged-file counts** aggregate from four finding kinds — the same §3.2/§3.3 defect classes distillation fixes:

1. `malformed[]` kind **`description-change-id`** — a `description:` carrying a registry-gated change-id (§3.2 ban, enforced/blocking).
2. `malformed[]` kind **`description-over-cap`** — a `description:` over the 1000-rune blocking cap (§3.2).
3. `warnings[]` kind **`description-length`** — a `description:` in the 501–1000 advisory band, over the 500-char soft cap (§3.2).
4. `warnings[]` kind **`narration-density`** — a topic file whose body carries ≥5 narration markers (§3.3 distillation-debt meter).

**Aggregation rules:** a file with **multiple findings counts once** (dedupe by `path`); a **sub-domain file rolls up to its domain** — the first path segment under `docs/memory/` (so `docs/memory/pipeline/runtime/x.md` counts under `pipeline`). The survey **re-applies the distillation exclusion set to the JSON finding paths** — it drops any finding whose path is an `index.md` or `_shared/removed-domains.md` before counting. The primitive scans neither exhaustively: it inspects `index.md` stubs for the three description-tier kinds (`description-change-id` / `description-over-cap` / `description-length`) and treats `_shared/removed-domains.md` as an ordinary topic file (its citation-dense rows trip `narration-density`), so their findings would otherwise be miscounted against a distilled domain. (`log.md` / `log.seed.md` never appear — the walker skips them.) Re-applying the exclusion set keeps a fully-distilled tree surveying clean — the worklist comes up empty and the loop reports the terminal all-distilled state.

**The check's exit code does NOT gate the survey** — the survey *consumes the report*, it is not a regen guard. Exit 1 (benign drift) and exit 2 (destructive loss) still produce a survey (the JSON is emitted on all `--check` exits). The refuse-before-regen exit-2 handling is a Step 5 concern, unrelated to surveying.

A **missing `type: memory` is NOT a survey signal** — the full read (Step 1 / Step 4) stamps it once a domain is selected, so it does not affect ranking.

**Older-binary fallback.** When `fab memory-index --check --json` is unavailable, or its output lacks the `warnings` key (an older binary that predates the machine surface), the survey **falls back to the legacy agent-side grep heuristics verbatim** — the three §3.2/§3.3 classes below — and **warns the user to upgrade `fab`** (mirroring `/docs-reorg-memory`'s Step 1 older-binary fallback posture):

1. **`description:` over the 500-char cap** — a frontmatter `description:` value longer than 500 characters (§3.2).
2. **change-ids in `description:`** — a `description:` carrying a `— xu0k`-style suffix or a `(d9rs)`-style citation (§3.2 bans them).
3. **narration markers in bodies** — a grep for the transition-narration patterns from Step 1 (e.g. `renamed`, `supersed`, `` was ` ``, `superseding the historical`, `inverts`). The list is seeded from Step 1's canonical patterns and is extensible (the "e.g." is deliberate). The fallback **survey exclusions** match distillation's: skip `index.md`, `log.md`, `log.seed.md`, and `_shared/removed-domains.md`; **recurse into sub-domains** like Step 1.

Then:

1. **Report per-domain status** — which domains have flagged files and how many (see § Output). State the heuristic **caveat**.
2. **Build the flagged-domain worklist** — every domain with ≥1 flagged file, in `docs/memory/index.md` domain-table order. This is the loop's fixed worklist: the survey runs **once** and the loop iterates this initial list (**no re-survey between domains**). Announce the worklist, then enter **Step 6 (all-domains loop)** — each domain runs the unchanged one-domain flow (full read → per-file report → Step 3 approval → apply → regen) as the loop body.
3. **If no domain has any flagged file**, report the terminal **"all domains distilled (survey heuristic)"** case with the caveat (§ Output) and stop — nothing is read in full and nothing is mutated.

> The survey is heuristic: a domain can pass the cheap scan while still carrying superseded-state prose. That is fine for ranking the worklist (the full read catches it once the loop reaches that domain); the only silent-skip risk is the terminal all-clean case, so its output MUST carry the caveat.

### Step 1: Read the domain (read-only)

> **Steps 1–5 are the one-domain flow — the *approval/apply unit*.** On an explicit `<domain>` they run once for that domain. On a no-arg invocation they are the **loop body**: Step 6 iterates them over every flagged domain in the survey worklist, one domain per approval unit. Nothing below changes between the two entry paths.

Read the target domain's `index.md` and every topic file (recursing into sub-domains). For each topic file, identify:

1. **Transition narration** — "renamed X→Y in {id}", "supersedes/inverts {id}", "was `old.value`", "superseding the historical …", and similar retrospective prose.
2. **Superseded-state descriptions** — prose describing behavior that is no longer current (what a thing *used to* do).
3. **`description:` frontmatter defects** — a value over the **500-character** cap, or one carrying change-ids (a `— xu0k`-style suffix or a `(d9rs)`-style citation) — both banned/capped by §3.2.
4. **Rationale carried inside narration** — deliberate-behavior / "don't re-break this" content woven into transition prose (candidates for **relocation**, per the guard).
5. **Allowed provenance already present** — trailing `(change-id)` citations and `*Introduced by*` fields (to be **preserved**).
6. **Change-id heading suffixes** *(§3.3 — heading text names its topic, never a change)* — a heading carrying a change-id token: `### Dispatch States (xu0k)`, `## Foo — 260718-mxgu`, `## xu0k — dispatch states`. Token recognition is **registry-gated** (the same posture the mxgu change-id checks use): a full `YYMMDD-XXXX-slug` token always matches; a bare 4-char id matches **only** when it is registry-plausible (present under `fab/changes/*` / `archive/**`) — the Step 3 human gate covers residual false positives. Candidate for **stripping the token, keeping the heading text**.
7. **Literal duplicate headings/blocks** *(§3.3 — a body states current truth once)* — a **byte-identical** duplicated heading pair or block within one file (e.g. the same `## Foo`-headed block appearing twice verbatim). Candidate for **removing the later byte-identical duplicate**. A merely *similar* (non-byte-identical) block is a **near-duplicate** — flagged for the human, never auto-removed.
8. **Design-Decisions changelog bullets** *(§3.3 — the changelog-bullet shape is banned inside `## Design Decisions`)* — a `- **{change-id} — retired X**`-shaped bullet inside a `## Design Decisions` section. Candidate for **rewrite to the four-field entry** (durable decision) or **removal** (pure change history already in `log.md`/git).
9. **Embedded operational TODOs** *(§3.3 — follow-up work items are never memory-body content)* — a TODO, "still needs X", or next-step checklist item in a memory body. Candidate for **relocation to `fab/backlog.md`** (never deletion — Step 4).

Skip `index.md` / `log.md` (generated), `log.seed.md` (a curated read-only seed input — never written by the generator, and excluded from distillation like a ledger), and `_shared/removed-domains.md` (tombstone exemption) — never rewrite them.

Classify every removal candidate — intent first: does it carry durable intent (a deliberate-behavior defense, a "don't re-break this", a rejected alternative)? If yes → **relocate into Design Decisions**, do not delete — regardless of where else it is recorded. Only intent-free narration whose content is already recorded elsewhere (per-folder `log.md`, git history, an archived change folder) is **safe to delete**. When in doubt, relocate. (The class-8 DD-bullet rewrite runs this same intent test; class 9 is a **relocation**, never a deletion.)

### Step 2: Per-file proposed-rewrite report

Emit a per-file report so the user sees the full blast radius before approving — the `/docs-reorg-memory` propose-then-apply idiom. For each file with proposed changes:

```
## Proposed rewrites — {domain}

### docs/memory/{domain}/{file}.md
- description: 5,912 → 418 chars (change-ids stripped: 260703-gvxd, +12 bare); displaced detail → body ## Overview
- remove transition narration: 3 lines (recorded in log.md — safe)
    - "renamed operator4 → operator in 260703-gvxd, superseding the historical …"
- strip change-id heading suffixes: 2 headings (kept as trailing citations where provenance matters)
    - "### Dispatch States (xu0k)" → "### Dispatch States"
- dedupe byte-identical blocks: 1 (near-duplicates flagged: 1 — see below, NOT auto-merged)
- rewrite DD changelog bullets: 1 — "- **260703-gvxd — retired the poll shim**" → four-field Design Decision entry
- RELOCATE TODOs → fab/backlog.md: 1 — "TODO: delete the stale gh secret" (relocated, never deleted)
- RELOCATE to Design Decisions (Why): 1 block — the Copilot poll-predicate "do not simplify" note (recorded nowhere else — intent preserved)
- keep: trailing (change-id) citations ×7, *Introduced by* ×2
- ⚠ near-duplicate flagged for manual review (NOT auto-merged): the two "## Retry policy" sections differ by one sentence

### docs/memory/{domain}/{other}.md
- no changes — already present-truth
```

Show enough of each proposed edit (before/after snippets for the non-obvious ones, especially every **relocation** — both the Design-Decisions relocations and every TODO → `fab/backlog.md` relocation) that the user can judge it. State per file whether any content is deleted vs. relocated, and name where deleted content is already recorded. **Every near-duplicate (class 7) is flagged for manual review, never listed as an auto-removal** — content judgment stays with the human gate.

If the domain is already present-truth, report **"no rewrites proposed — {domain} is already distilled"** and stop (idempotency — Constitution III).

### Step 3: User confirmation

Options: **Apply all**, **Cherry-pick** (select specific files), **Skip** (keep analysis only). Mutate nothing until the user approves. If the user declines, report the analysis and stop — no file changed.

### Step 4: Apply approved rewrites

For each approved file:

1. **Rewrite the body to present truth** (§3.3) — remove the approved transition-narration / superseded-state lines; **relocate** each rationale block into a `## Design Decisions` entry (`Why` / `Rejected`, present-tense; add `*Introduced by*: {change-name}` when the change is known); preserve trailing `(change-id)` citations and existing `*Introduced by*` fields verbatim.
2. **Strip change-id heading suffixes** (§3.3, class 6) — remove the change-id token from each approved heading, keeping the heading text (`### Dispatch States (xu0k)` → `### Dispatch States`). If the stripped token carried provenance worth keeping, add it back as a **trailing `(change-id)` citation in the section body** (allowed provenance, per the keep-list) — never leave it in the heading.
3. **Remove byte-identical duplicate blocks** (§3.3, class 7) — delete the later of a **byte-identical** duplicated heading pair/block. **Never auto-merge a near-duplicate** — a non-byte-identical similar block was flagged for manual review in Step 2 and is left untouched here (content judgment is the human's, not this skill's).
4. **Rewrite Design-Decisions changelog bullets** (§3.3, class 8) — a `- **{change-id} — retired X**`-shaped bullet inside `## Design Decisions`: when it encodes a **durable decision**, rewrite it to the four-field entry (**Decision** / **Why** / **Rejected** / *Introduced by* — the change-id moves into *Introduced by* or a trailing citation); when it is **pure change history** already recorded in `log.md`/git, remove it under the Step 1 deletion-safety rule. **Never fabricate rationale** — when `Why`/`Rejected` content is not derivable from the bullet or its surrounding context, the rewritten entry carries only the fields that exist (Decision + *Introduced by*).
5. **Relocate operational TODOs → `fab/backlog.md`** (§3.3, class 9) — **relocate, never delete** (follow-up work items belong in the project backlog, not a memory body). Remove the TODO from the memory body and append a standard backlog entry to `fab/backlog.md`:

   ```markdown
   - [ ] [{fresh-4char-id}] {YYYY-MM-DD}: {TODO text} (relocated from docs/memory/{domain}/{file}.md by /docs-distill-memory)
   ```

   Generate a fresh 4-char id (lowercase alphanumeric, not colliding with a registered change or an existing backlog id) and today's date. Append under the backlog's `## Open` section (create the section if absent). **If `fab/backlog.md` does not exist** (user repos), create it with a minimal `# Backlog` header first. Relocation honors the Step 3 approval unit: a file the user **skips or cherry-picks away keeps its TODOs** — no orphaned relocation is written for an un-approved file.
6. **Fix the `description:` frontmatter** (§3.2) — strip change-ids; compress to a ≤500-character single-line routing signal; move displaced routing-irrelevant detail into the body (`## Overview` / `## Requirements` / `## Design Decisions`) where it is not already present.
7. **Stamp the `type: memory` constant** — keep it when present; **stamp it if the legacy file lacks it** (FKF §2/§3.1 require it on every memory file, and every writer that touches a file leaves it conforming). This runs for every approved file regardless of whether its `description:` needed changing — a file with an already-conforming description must still be left with `type: memory`.
8. **Bundle-relative links** (§7) — if a rewrite touches a memory↔memory link, keep the bundle-relative `/...` form (resolved from `docs/memory/`); links out of the bundle (source, specs, URLs) stay repo-relative / absolute-URL. This skill moves no files, so it creates no new link breakage.

Never touch `index.md` / `log.md` (Step 5 regenerates them), `log.seed.md` (a curated read-only seed input — the generator reads it, never writes it), or `_shared/removed-domains.md` (exempt). `fab/backlog.md` (class 9 relocation target) is the **one** file outside `docs/memory/` this skill writes.

### Step 5: Regenerate indexes (refuse-before-regen guard)

After applying rewrites, regenerate the generated files — **never hand-edit them** (FKF §5).

1. **Consult `fab memory-index --check` first** (the refuse-before-regen guard `/docs-hydrate-memory` also carries; exit tiers in `_cli-fab` § fab memory-index):
   - **Exit 0** (clean) / **exit 1** (benign drift) → proceed to regenerate.
   - **Exit 2** (destructive loss) → **refuse to regenerate** and surface the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` This is a **no-op for born-compatible fab-kit trees** (always exit 0/1, never 2 — not dead code); it is defense-in-depth for a pre-fab-kit tree reaching this skill.
2. **Regenerate** via `fab memory-index` — it rewrites the `index.md` tiers (root domains-only, domain, sub-domain) and each folder's `log.md`, from two distinct derivations: the **index tiers** are a pure function of folder contents + each file's `description:` frontmatter (content-only, no dates), while each **`log.md`** is the C-lite join of git history + per-change `.status.yaml` `summary:` fields (freeze-on-write, append-only — the existing log is authoritative; only new `(file-base, change-id)` entries are appended, and any `log.seed.md` is merged beneath). Take its output wholesale; never hand-merge a generated file (FKF §5). `fab memory-index` is byte-stable, so a no-op re-run produces no index diff.

Heed any non-fatal shape/length warnings `fab memory-index` prints — a still-over-cap `description:` warning (501–1000 chars, advisory) is a signal to trim further. Note the two `description:` escalations are **blocking**, not advisory: a change-id in `description:`, or a gross over-cap value (> 1000 chars, 2× the 500 soft cap), fails `--check` (FKF §3.2) — so a distillation run that leaves either in place will not regenerate clean.

### Step 6: All-domains loop (no-arg only)

Runs **only on a no-arg invocation** (an explicit `<domain>` runs Steps 1–5 once for that domain and stops — no loop). After the Step 0 survey builds the flagged-domain worklist, iterate **every flagged domain sequentially in `docs/memory/index.md` domain-table order**, running Steps 1–5 (the one-domain flow) as the loop body for each. The loop runs **in the main session** — the Step 3 approval prompt is interactive and must reach the user, so there is **no per-domain subagent dispatch**.

Loop semantics:

- **Survey once, no re-survey between domains** — the loop iterates the *initial* Step 0 worklist. A file mutated in one domain never changes another domain's membership, so no re-survey is needed (or run).
- **Per-domain approval is the unit** — each domain gets its own Step 3 prompt (apply all / cherry-pick / skip). Bulk approval across all domains in one prompt is **deliberately NOT offered** — it would collapse the human safeguard on load-bearing memory files.
- **Skip → untouched, loop continues** — a **skipped** domain (Step 3 skip, or a cherry-pick that leaves flagged files) stays untouched and the loop moves to the next domain; it is reported in the terminal summary as skipped/remaining.
- **Already-distilled domain → report and continue** — a domain whose full read (Step 1) finds nothing reports "no rewrites proposed — {domain} is already distilled" (Step 2) and the loop continues to the next domain (the survey is heuristic, so a worklist domain can turn out clean on the full read).
- **Exit-2 within one domain → per-domain handling, then continue** — an exit-2 refuse-before-regen event (Step 5) is handled per the existing per-domain posture (report the reorg-remediation pointer, defer that domain's regen); it does **not** silently swallow the remaining domains. Continue the loop (or stop and report) per that domain's error-handling outcome — remaining domains are never dropped without a report.
- **Terminal state** — when the worklist is exhausted, report either **"all domains distilled"** (every flagged domain processed) or a summary listing the **skipped/remaining** domains (§ Output). The dynamic `Next:` line reports that same surveyed truth (§ Output → Dynamic `Next:` line) — it no longer drives per-domain re-invocation.

---

## Output

**Survey report (no-arg only, Step 0)** — emitted before the all-domains loop begins:

```
Surveying docs/memory/ — heuristic scan across {N} domains (read-only)...

  _shared        2 files flagged
  distribution   3 files flagged
  memory-docs    —
  pipeline       —
  runtime        1 file flagged

Survey is heuristic; run /docs-distill-memory <domain> to force a full read of a single domain.

Looping 3 flagged domains in index order: _shared → distribution → runtime.
Distilling _shared (1 of 3) → reading its topic files in full...
```

If the survey finds nothing anywhere (terminal all-clean case):

```
Surveying docs/memory/ — heuristic scan across {N} domains (read-only)...

All domains distilled (survey heuristic) — no candidate domains flagged.
Survey is heuristic; run /docs-distill-memory <domain> to force a full read of a specific domain.
```

**Per-domain distillation** (each loop iteration; also the whole output on an explicit `<domain>`):

```
Distilling docs/memory/{domain}/ ({i} of {M}) — reading {N} topic files (read-only)...

{per-file proposed-rewrite report}

Apply these rewrites? (apply all / cherry-pick / skip)
```

After each domain's apply:

```
{domain} distilled — {F} files rewritten, {D} narration lines removed, {H} change-id heading suffixes stripped, {B} byte-identical blocks deduped ({N} near-duplicates flagged), {DD} DD changelog bullets rewritten, {T} TODOs relocated → fab/backlog.md, {R} rationale blocks relocated to Design Decisions, {C} descriptions capped/de-id'd. Indexes regenerated via fab memory-index; no generated file hand-edited.
```

Per-domain, when no changes are needed: `No rewrites proposed — {domain} is already distilled (present-truth).` — the loop continues to the next domain.

Per-domain, when the user declines/skips: `Analysis reported; {domain} left intact (no files mutated).` — the loop continues to the next domain (the domain stays listed as skipped/remaining in the terminal summary).

Per-domain, if `fab memory-index --check` returned exit 2: report the refuse-before-regen pointer for that domain and stop that domain's regeneration (rewrites already applied stay; regeneration is deferred to the reorg remediation). Remaining domains are handled per the Step 6 exit-2 posture — never silently dropped.

**Terminal summary (no-arg, after the worklist is exhausted)** — one of:

```
All domains distilled — {M} domains processed ({F} files rewritten total). No candidates remain (survey heuristic).
```

```
Distillation loop complete — {P} of {M} domains distilled; {S} skipped/remaining: _shared (2 files flagged), runtime (1 file flagged).
```

**Dynamic `Next:` line** — the skill's closing line (below) reports the surveyed **skipped/remaining** domains (surveyed truth), or all-distilled when none remain. It **reports state; it no longer drives per-domain re-invocation**:

- On a **no-arg** invocation, the line reflects the initial Step 0 survey minus every domain fully distilled this run. A domain the user **skipped** or only **partially cherry-picked** stays listed while it still carries flagged files.
- On an **explicit-`<domain>`** invocation (no upfront survey ran), run the survey at completion to populate the line.
- Skipped/remaining domains are listed in `docs/memory/index.md` domain-table order, each with its flagged-file count, as a pointer for a follow-up **targeted** run. Example:

```
Next: all domains distilled (survey heuristic) — /docs-reorg-memory or /fab-new
```

or, when the user skipped or partially cherry-picked domains that still carry flagged files:

```
Next: skipped/remaining — /docs-distill-memory _shared (2 files flagged), /docs-distill-memory runtime (1 file flagged); or /docs-reorg-memory, /fab-new
```

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/index.md` missing | Abort: "docs/memory/ not found. Run /fab-setup first." |
| No `<domain>` argument | *(not an error — runs survey mode + the all-domains loop, Behavior Steps 0 and 6)* |
| Domain folder missing / no topic files (explicit `<domain>`) | Abort: "Domain '{domain}' not found (or has no topic files). Available: {list domain folders}." |
| Ambiguous domain (matches >1 folder) | Abort: "'{domain}' matches {N} domains: {list}. Name one." |
| Multiple domains passed | Abort: "One domain per named run — run /docs-distill-memory with no argument to loop every flagged domain (each still approved on its own), or name a single domain." |
| `fab memory-index --check --json` unavailable / no `warnings` key (older binary) — **survey mode** | Fall back to the legacy agent-side grep heuristics (the three §3.2/§3.3 classes) verbatim and warn to upgrade `fab`; the survey still runs (Behavior Step 0 older-binary fallback) |
| `fab memory-index --check` exit 2 (destructive loss) | Refuse to regenerate; surface the `→ run /docs-reorg-memory to remediate …` pointer (no-op on born-compatible fab-kit trees — not dead code). *(Survey mode does NOT gate on exit code — exit 2 still surveys.)* |
| `fab memory-index` unavailable (older binary) — **regeneration** | Warn; the rewrites are applied but indexes are not regenerated — tell the user to upgrade `fab` and re-run `fab memory-index` |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Requires config/constitution? | No |
| Argument | `[<domain>]` optional — named forces a full read of that one domain (no loop); omitted runs survey mode then the all-domains loop (heuristic scan → per-domain report → loop every flagged domain sequentially in index order, one-domain flow per domain) |
| Scope per run | One domain per **approval/apply unit**, propose-then-apply (read-only until explicit approval), iterated within a single invocation — a no-arg invocation loops the unit over every flagged domain; an explicit `<domain>` runs it once. Each domain is read-in-full, reported, approved, and rewritten as its own unit (per-domain gate; no bulk approval) |
| Modifies memory files? | Yes — rewrites topic-file bodies + `description:` frontmatter to FKF present-truth style (transition narration, superseded state, change-id heading suffixes, byte-identical duplicate blocks, DD changelog bullets), only with explicit confirmation. `_shared/removed-domains.md` is exempt (§3.3 tombstone carve-out) |
| Writes outside `docs/memory/`? | Yes — one file: `fab/backlog.md` (operational-TODO relocation target, §3.3 class 9 — created with a `# Backlog` header when absent). Never deletes a TODO |
| Preserves rationale? | Yes — deliberate-behavior/"don't re-break" content is relocated into Design Decisions (`Why`/`Rejected`), never deleted; deletion is confined to narration recorded elsewhere (log.md/git/archive). **Never fabricates rationale** — a DD-bullet rewrite with no derivable Why/Rejected carries only Decision + *Introduced by* |
| Preserves provenance? | Yes — trailing `(change-id)` citations and `*Introduced by*` fields are kept; change-ids are stripped from `description:` frontmatter (§3.2) and from heading text (§3.3 — kept as a trailing body citation when provenance matters) |
| Auto-merges near-duplicates? | No — only **byte-identical** within-file blocks are auto-removed; near-duplicates are flagged for manual review. Cross-file duplicate coverage belongs to /docs-reorg-memory |
| Moves files? | No — this skill rewrites in place; structural moves (incl. cross-file splits/merges) belong to /docs-reorg-memory |
| Idempotent? | Yes — an already-distilled domain proposes nothing; a fully-distilled tree surveys clean so the no-arg loop's worklist is empty (terminal all-distilled); `fab memory-index` regeneration is byte-stable (Constitution III) |
| Indexes hand-edited? | No — regenerated by `fab memory-index`; honors the refuse-before-regen `--check` exit-2 guard |

---

Next: {surveyed skipped/remaining domains, e.g. skipped/remaining — /docs-distill-memory runtime (1 file flagged); or /docs-reorg-memory, /fab-new}
<!-- Dynamic: populated from the survey (§ Output → Dynamic `Next:` line). Reports the surveyed SKIPPED/REMAINING domains in docs/memory/index.md domain-table order with flagged-file counts — a pointer for a follow-up targeted run, NOT per-domain re-invocation (the no-arg loop already processes every flagged domain in one invocation). When none remain, reports "all domains distilled (survey heuristic)" instead. -->
