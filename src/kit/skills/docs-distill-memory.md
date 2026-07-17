---
name: docs-distill-memory
description: "Rewrite existing memory files to the FKF present-truth style — strip transition narration and superseded-state prose, cap descriptions, relocate rationale into Design Decisions. One domain per run; read-only until you approve."
---

# /docs-distill-memory <domain>

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

**One domain per run, propose-then-apply.** The skill runs **read-only analysis** over one named domain's topic files, produces a **per-file proposed-rewrite report**, and applies **only on explicit user approval** — the same posture as `/docs-reorg-memory` (its report → confirm-and-apply shape). These files encode load-bearing behavioral contracts, so a human approves per domain, seeing per-file diffs. It is **not** an autonomous bulk rewriter.

> **Distinct from the sibling doc skills.** `/docs-reorg-memory` reorganizes **structure** (splits/merges/moves + link rewrites) — it never rewrites body prose to a style. `/docs-hydrate-memory` backfill mode is **body-preserving** (adds frontmatter only); its ingest/generate modes author *new* content. `/fab-continue` hydrate writes each change's delta as current truth but only touches the sections its change affects. This skill is the only one that rewrites **existing** body prose to the present-truth style across a whole domain.

### What a rewrite does (FKF §3.2 / §3.3)

The normative source is the shipped FKF extract `$(fab kit-path)/reference/fkf.md` — cite it (deployed skills reach the extract; the dev-repo `docs/specs/fkf.md` is absent in user repos). A rewrite:

- **Removes transition narration** — no "renamed X→Y in {id}", no "this inverts/supersedes {id}'s claim", no "was `old.value`", no "superseding the historical …".
- **Removes superseded-state descriptions** — the body carries only what IS. Previous states belong to the per-folder generated `log.md`, git history, and archived change folders.
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

- **`<domain>`** *(required)* — the single memory domain to distill, named by its `docs/memory/` folder (e.g. `pipeline`, `distribution`, `runtime`, `_shared`). Case-insensitive substring match against domain folder names; an ambiguous or unknown name is handled per Error Handling. **One domain per run** — the skill rejects a multi-domain invocation (run it once per domain so each domain's diffs are approved on their own).

---

## Pre-flight

1. `docs/memory/index.md` must exist and be readable.
2. The named domain folder `docs/memory/{domain}/` must exist and contain at least one topic file (a non-`index.md`/`log.md`/`log.seed.md` `.md` file).

If either fails, STOP with the matching Error Handling message.

---

## Context Loading

This section is the skill-file override the `_preamble.md` §1 contract keys on — the skill does **NOT** load the always-load layer. It requires **no active change, config, or constitution**. Up front it reads only:

- `docs/memory/index.md` and the target domain's `docs/memory/{domain}/index.md` (and any sub-domain `index.md`) — the domain landscape.
- Every topic file in the target domain (recursing into sub-domains) — the rewrite subjects.
- `$(fab kit-path)/reference/fkf.md` — the shipped normative extract (§3.2 `description` rules incl. the 500-char cap and change-id ban; §3.3 present-truth body style incl. the tombstone carve-out). Read it so every proposed rewrite cites the deployed rule, not a remembered one.

For the `fab memory-index` exit tiers and the refuse-before-regen pointer, consult **`_cli-fab` § fab memory-index** by in-body pointer (below) — it is not pre-loaded.

---

## Behavior

### Step 1: Read the domain (read-only)

Read the target domain's `index.md` and every topic file (recursing into sub-domains). For each topic file, identify:

1. **Transition narration** — "renamed X→Y in {id}", "supersedes/inverts {id}", "was `old.value`", "superseding the historical …", and similar retrospective prose.
2. **Superseded-state descriptions** — prose describing behavior that is no longer current (what a thing *used to* do).
3. **`description:` frontmatter defects** — a value over the **500-character** cap, or one carrying change-ids (a `— xu0k`-style suffix or a `(d9rs)`-style citation) — both banned/capped by §3.2.
4. **Rationale carried inside narration** — deliberate-behavior / "don't re-break this" content woven into transition prose (candidates for **relocation**, per the guard).
5. **Allowed provenance already present** — trailing `(change-id)` citations and `*Introduced by*` fields (to be **preserved**).

Skip `index.md` / `log.md` (generated), `log.seed.md` (a curated read-only seed input — never written by the generator, and excluded from distillation like a ledger), and `_shared/removed-domains.md` (tombstone exemption) — never rewrite them.

Classify every removal candidate — intent first: does it carry durable intent (a deliberate-behavior defense, a "don't re-break this", a rejected alternative)? If yes → **relocate into Design Decisions**, do not delete — regardless of where else it is recorded. Only intent-free narration whose content is already recorded elsewhere (per-folder `log.md`, git history, an archived change folder) is **safe to delete**. When in doubt, relocate.

### Step 2: Per-file proposed-rewrite report

Emit a per-file report so the user sees the full blast radius before approving — the `/docs-reorg-memory` propose-then-apply idiom. For each file with proposed changes:

```
## Proposed rewrites — {domain}

### docs/memory/{domain}/{file}.md
- description: 5,912 → 418 chars (change-ids stripped: 260703-gvxd, +12 bare); displaced detail → body ## Overview
- remove transition narration: 3 lines (recorded in log.md — safe)
    - "renamed operator4 → operator in 260703-gvxd, superseding the historical …"
- RELOCATE to Design Decisions (Why): 1 block — the Copilot poll-predicate "do not simplify" note (recorded nowhere else — intent preserved)
- keep: trailing (change-id) citations ×7, *Introduced by* ×2

### docs/memory/{domain}/{other}.md
- no changes — already present-truth
```

Show enough of each proposed edit (before/after snippets for the non-obvious ones, especially every **relocation**) that the user can judge it. State per file whether any content is deleted vs. relocated, and name where deleted content is already recorded.

If the domain is already present-truth, report **"no rewrites proposed — {domain} is already distilled"** and stop (idempotency — Constitution III).

### Step 3: User confirmation

Options: **Apply all**, **Cherry-pick** (select specific files), **Skip** (keep analysis only). Mutate nothing until the user approves. If the user declines, report the analysis and stop — no file changed.

### Step 4: Apply approved rewrites

For each approved file:

1. **Rewrite the body to present truth** (§3.3) — remove the approved transition-narration / superseded-state lines; **relocate** each rationale block into a `## Design Decisions` entry (`Why` / `Rejected`, present-tense; add `*Introduced by*: {change-name}` when the change is known); preserve trailing `(change-id)` citations and existing `*Introduced by*` fields verbatim.
2. **Fix the `description:` frontmatter** (§3.2) — strip change-ids; compress to a ≤500-character single-line routing signal; move displaced routing-irrelevant detail into the body (`## Overview` / `## Requirements` / `## Design Decisions`) where it is not already present.
3. **Stamp the `type: memory` constant** — keep it when present; **stamp it if the legacy file lacks it** (FKF §2/§3.1 require it on every memory file, and every writer that touches a file leaves it conforming). This runs for every approved file regardless of whether its `description:` needed changing — a file with an already-conforming description must still be left with `type: memory`.
4. **Bundle-relative links** (§7) — if a rewrite touches a memory↔memory link, keep the bundle-relative `/...` form (resolved from `docs/memory/`); links out of the bundle (source, specs, URLs) stay repo-relative / absolute-URL. This skill moves no files, so it creates no new link breakage.

Never touch `index.md` / `log.md` (Step 5 regenerates them), `log.seed.md` (a curated read-only seed input — the generator reads it, never writes it), or `_shared/removed-domains.md` (exempt).

### Step 5: Regenerate indexes (refuse-before-regen guard)

After applying rewrites, regenerate the generated files — **never hand-edit them** (FKF §5).

1. **Consult `fab memory-index --check` first** (the refuse-before-regen guard `/docs-hydrate-memory` also carries; exit tiers in `_cli-fab` § fab memory-index):
   - **Exit 0** (clean) / **exit 1** (benign drift) → proceed to regenerate.
   - **Exit 2** (destructive loss) → **refuse to regenerate** and surface the pointer `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.` This is a **no-op for born-compatible fab-kit trees** (always exit 0/1, never 2 — not dead code); it is defense-in-depth for a pre-fab-kit tree reaching this skill.
2. **Regenerate** via `fab memory-index` — it rewrites the `index.md` tiers (root domains-only, domain, sub-domain) and each folder's `log.md`, from two distinct derivations: the **index tiers** are a pure function of folder contents + each file's `description:` frontmatter (content-only, no dates), while each **`log.md`** is the C-lite join of git history + per-change `.status.yaml` `summary:` fields (freeze-on-write, append-only — the existing log is authoritative; only new `(file-base, change-id)` entries are appended, and any `log.seed.md` is merged beneath). Take its output wholesale; never hand-merge a generated file (FKF §5). `fab memory-index` is byte-stable, so a no-op re-run produces no index diff.

Heed any non-fatal shape/length warnings `fab memory-index` prints (advisory only) — a still-over-cap `description:` warning is a signal to trim further.

---

## Output

```
Distilling docs/memory/{domain}/ — reading {N} topic files (read-only)...

{per-file proposed-rewrite report}

Apply these rewrites? (apply all / cherry-pick / skip)
```

After apply:

```
Distillation complete — {F} files rewritten, {D} narration lines removed, {R} rationale blocks relocated to Design Decisions, {C} descriptions capped/de-id'd. Indexes regenerated via fab memory-index; no generated file hand-edited.
```

If no changes needed: `No rewrites proposed — {domain} is already distilled (present-truth).`

If the user declined: `Analysis reported; {domain} left intact (no files mutated).`

If `fab memory-index --check` returned exit 2: report the refuse-before-regen pointer and stop before regenerating (rewrites already applied stay; regeneration is deferred to the reorg remediation).

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `docs/memory/index.md` missing | Abort: "docs/memory/ not found. Run /fab-setup first." |
| No `<domain>` argument | Abort: "Name a domain to distill, e.g. /docs-distill-memory pipeline. Run one domain per run." |
| Domain folder missing / no topic files | Abort: "Domain '{domain}' not found (or has no topic files). Available: {list domain folders}." |
| Ambiguous domain (matches >1 folder) | Abort: "'{domain}' matches {N} domains: {list}. Name one." |
| Multiple domains passed | Abort: "One domain per run — run /docs-distill-memory once per domain so each domain's diffs are approved separately." |
| `fab memory-index --check` exit 2 (destructive loss) | Refuse to regenerate; surface the `→ run /docs-reorg-memory to remediate …` pointer (no-op on born-compatible fab-kit trees — not dead code) |
| `fab memory-index` unavailable (older binary) | Warn; the rewrites are applied but indexes are not regenerated — tell the user to upgrade `fab` and re-run `fab memory-index` |

---

## Key Properties

| Property | Value |
|----------|-------|
| Advances stage? | No |
| Requires active change? | No |
| Requires config/constitution? | No |
| Scope per run | One domain, propose-then-apply (read-only until explicit approval) |
| Modifies memory files? | Yes — rewrites topic-file bodies + `description:` frontmatter to FKF present-truth style, only with explicit confirmation. `_shared/removed-domains.md` is exempt (§3.3 tombstone carve-out) |
| Preserves rationale? | Yes — deliberate-behavior/"don't re-break" content is relocated into Design Decisions (`Why`/`Rejected`), never deleted; deletion is confined to narration recorded elsewhere (log.md/git/archive) |
| Preserves provenance? | Yes — trailing `(change-id)` citations and `*Introduced by*` fields are kept; change-ids are stripped only from `description:` frontmatter (§3.2) |
| Moves files? | No — this skill rewrites in place; structural moves belong to /docs-reorg-memory |
| Idempotent? | Yes — an already-distilled domain proposes nothing; `fab memory-index` regeneration is byte-stable (Constitution III) |
| Indexes hand-edited? | No — regenerated by `fab memory-index`; honors the refuse-before-regen `--check` exit-2 guard |

---

Next: /docs-distill-memory {another-domain}, /docs-reorg-memory, or /fab-new
