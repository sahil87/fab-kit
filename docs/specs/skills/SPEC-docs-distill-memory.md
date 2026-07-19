# docs-distill-memory

## Summary

Rewrites an existing `docs/memory/` domain's topic files to the **FKF present-truth style** (`$(fab kit-path)/reference/fkf.md` §3.2, §3.3). The FKF present-truth rules govern what memory writers produce *going forward*; a corpus authored before them accumulates transition narration, superseded-state prose, and over-cap / change-id-carrying `description:` frontmatter. This skill is the remediation counterpart that cleans that **existing** corpus. **`<domain>` is optional**: named explicitly, it forces a full read of that one domain and runs the one-domain flow once (no loop); **omitted, it runs no-arg survey mode** (Behavior Step 0) — a cheap heuristic scan across all domains that reports per-domain candidate counts, builds a flagged-domain worklist, and then **loops every flagged domain sequentially** (Behavior Step 6) in `docs/memory/index.md` domain-table order, running the one-domain flow as the loop body per domain (or reports the terminal all-distilled case when nothing is flagged). The loop runs **in the main session** — the approval prompt is interactive — with **no per-domain subagent dispatch**. **One domain per approval/apply unit, propose-then-apply**: read-only analysis over one domain → a per-file proposed-rewrite report → apply **only on explicit user approval** (the `/docs-reorg-memory` posture). "One domain" is a property of the analysis+apply/approval unit, not the invocation — a no-arg invocation iterates that unit over every flagged domain; an explicit `<domain>` runs it once. Not an autonomous bulk rewriter — these files encode load-bearing behavioral contracts, so a human approves **per domain** seeing per-file diffs (bulk approval across all domains in one prompt is deliberately not offered).

Introduced as the deliberately-deferred **step 3 of the present-truth effort**: steps 1–2 (the FKF §3.2 change-id ban + §3.3 present-truth body-style rule, and the forward-looking memory writers) shipped in `260717-3plm`; this skill cleans the corpus those rules did not retroactively fix.

## Niche vs. the sibling doc skills

- `/docs-reorg-memory` reorganizes **structure** (splits/merges/moves + bundle-relative link rewrites) — it never rewrites body prose to a style.
- `/docs-hydrate-memory` **backfill** mode is body-preserving (adds frontmatter only); its ingest/generate modes author *new* content from sources/code.
- `/fab-continue` **hydrate** writes each change's delta as current truth but only touches the sections its change affects.
- `/docs-distill-memory` is the only skill that rewrites **existing** body prose to the present-truth style across a whole domain. Chosen as a **discoverable user-invocable skill**, not a `/docs-reorg-memory` mode (a mode is not discoverable).

## Rewrite semantics (FKF §3.2 / §3.3)

The normative source is the **shipped FKF extract** `$(fab kit-path)/reference/fkf.md` — the skill cites the extract (reachable in every user repo), not the dev-repo `docs/specs/fkf.md` (absent there). A rewrite:

- **Removes transition narration** — "renamed X→Y in {id}", "supersedes/inverts {id}", "was `old.value`", "superseding the historical …".
- **Removes superseded-state descriptions** — the body carries only what IS; previous states live in the per-folder generated `log.md`, git history, and archived change folders.
- **Strips change-id heading suffixes** (§3.3) — a heading is `## Dispatch States`, never `### Dispatch States (xu0k)` or `## xu0k — dispatch states`; the token is removed, kept as a trailing body citation when provenance matters. Recognition is **registry-gated** (a full `YYMMDD-XXXX-slug` token always; a bare 4-char id only when registry-plausible — the Step 3 human gate covers residual false positives).
- **Dedupes byte-identical duplicate headings/blocks** (§3.3) — the later of a **byte-identical** duplicated block is removed. **Near-duplicates are flagged for manual review, never auto-merged** — content judgment stays with the human gate (cross-file duplication belongs to `/docs-reorg-memory`'s duplicate-coverage pass).
- **Rewrites Design-Decisions changelog bullets** (§3.3) — a `- **{change-id} — retired X**`-shaped bullet inside `## Design Decisions` (the banned shape) is rewritten to the four-field entry (**Decision** / **Why** / **Rejected** / *Introduced by* — the change-id moves into *Introduced by* or a trailing citation) when it encodes a durable decision, or removed when it is pure change history already in `log.md`/git. **Never fabricates rationale** — an entry with no derivable Why/Rejected carries only Decision + *Introduced by*.
- **Relocates operational TODOs → `fab/backlog.md`** (§3.3) — follow-up work items (TODOs, "still needs X", next-step checklists) belong in the project backlog, not a memory body. They are **relocated, never deleted**: the TODO is removed from the body and appended to `fab/backlog.md` as `- [ ] [{fresh-4char-id}] {YYYY-MM-DD}: {text} (relocated from docs/memory/{domain}/{file}.md by /docs-distill-memory)` (creating `fab/backlog.md` with a `# Backlog` header when absent). Relocation honors the Step 3 per-file approval unit — a skipped/cherry-picked-away file keeps its TODOs.
- **Keeps allowed provenance** — trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions (§3.3: a 6-char `(id)` cheaply defends deliberate behavior). **Bare 4-char ids are treated identically to dated ids** — kept in trailing-citation position, removed when woven into narration.
- **Fixes `description:` frontmatter** — strips change-ids (§3.2 ban — no `— xu0k`-style suffix, no `(d9rs)`-style citation) and compresses an over-cap value to the **≤500-character** routing-signal shape, moving displaced routing-irrelevant detail into the body where it isn't already present. Stamps the `type: memory` constant when an edited legacy file lacks it (§2/§3.1).

These four new removal classes (change-id heading suffixes, byte-identical duplicate blocks, DD changelog bullets, operational-TODO relocation) join the taxonomy in Behavior Step 1 (identify), Step 2 (per-file report), and Step 4 (apply), each citing `$(fab kit-path)/reference/fkf.md` §3.3 (shipped by the `[wrct]` present-truth writer contract). `fab/backlog.md` is the **one** file outside `docs/memory/` the skill writes (class-9 relocation target).

## Rationale-preservation guard (the critical constraint)

**Token savings come from dropping narration, NEVER rationale** (FKF §3.3 verbatim: *"'Don't re-break this' content lives in Design Decisions' `Why` / `Rejected` as durable, present-tense design intent — a rejected alternative is a design fact, not transition narration."*).

- Deliberate-behavior / "don't re-break this" content is **RELOCATED** into `## Design Decisions` (`Why` / `Rejected`) as present-tense intent — never deleted.
- **Deletion is safe only** for narration whose content is **already recorded elsewhere** (per-folder `log.md`, git history, archived change folders). Content recorded nowhere else and carrying intent is relocated, not dropped. When ambiguous, relocate — the safe default preserves.

## Generated files & the tombstone exemption

- **Generated files are never hand-edited** — `index.md` (root/domain/sub-domain) and `log.md` are written solely by `fab memory-index` (FKF §5, §6). The skill regenerates via `fab memory-index` after applying rewrites and never edits their rows or hand-merges a generated conflict.
- **`log.seed.md` is a curated read-only SEED INPUT, not a generated file** — `fab memory-index` *reads* it during the seed-merge (like `description:` frontmatter) but never *writes* it; the generator stays the sole writer of `log.md`. It is nonetheless **excluded from distillation** — its body *is* a citation-carrying seed ledger of pre-FKF history in the §6.2 entry format, the same exclusion posture as `removed-domains.md`. The skill skips it entirely and never rewrites it.
- **Refuse-before-regen guard** — before regenerating, the skill consults `fab memory-index --check` (`_cli-fab` § fab memory-index): exit 0/1 → regenerate; **exit 2** (destructive loss) → **refuse** and surface the `→ run /docs-reorg-memory to remediate …` pointer. This is a **no-op on born-compatible fab-kit trees** (always exit 0/1, never 2 — defense-in-depth, not dead code).
- **`docs/memory/_shared/removed-domains.md` is EXEMPT** from rewrite — the §3.3 tombstone carve-out (its body *is* a citation-carrying removal ledger, not transition narration). fab-kit's own tree has no such file; the exemption matters in user projects, where `/docs-reorg-memory` authors it.

## Context Loading

Skill-file override of the always-load layer (the `_preamble` §1 contract keys on this section): **no active change, config, or constitution** required. For each target domain (named, or reached in turn as the no-arg loop iterates the survey worklist), reads only `docs/memory/index.md`, the target domain's `index.md` (+ sub-domain indexes), every topic file in the target domain, and `$(fab kit-path)/reference/fkf.md`. **Survey mode reads the machine surface up front**: on a no-arg invocation, before any domain's full read the survey runs a single `fab memory-index --check --json` and reads its JSON `malformed[]`/`warnings[]` arrays to count flagged files per domain — it does **not** read the corpus. Only the **older-binary fallback** reverts to the legacy all-domains read-only grep scan (each domain's `index.md` + enough of every topic file's `description:` frontmatter and body to run the narration-marker grep, recursing sub-domains, honoring the exclusion set). Either way the survey is a cheap heuristic ranking, not a full read; the full read is confined to each domain as the loop reaches it. The `fab memory-index --check --json` shape (the aggregated kinds), exit tiers, and refuse-before-regen pointer are consulted via an in-body `_cli-fab` § fab memory-index pointer (not pre-loaded). Declares no `helpers:`.

## Flow

```
User invokes /docs-distill-memory [<domain>]
│
├─ <domain> OMITTED → Step 0 Survey mode (no-arg):
│     ONE fab memory-index --check --json call (the canonical machine surface), NOT a full read:
│       count flagged files per domain by aggregating 4 finding kinds — malformed[] description-change-id
│       + description-over-cap (blocking) / warnings[] description-length (501–1000 advisory) + narration-density.
│       a file with multiple findings counts ONCE; a sub-domain file rolls up to its domain (first path
│       segment under docs/memory/). exit code does NOT gate the survey (exit 1/2 still surveys).
│       missing type: memory is NOT a survey signal. RE-APPLY the exclusion set to the JSON paths — drop
│       index.md + _shared/removed-domains.md findings before counting (the primitive scans index.md stubs
│       for the description-tier kinds and removed-domains.md as a topic file; log.md/log.seed.md never appear).
│       OLDER-BINARY FALLBACK (no --json / no warnings key): legacy agent-side grep of the 3 classes
│         (description: over 500-char cap / change-ids in description: / body narration markers) + "upgrade fab" warning.
│     ├─ report per-domain candidate counts + heuristic CAVEAT
│     ├─ [no domain flagged] → "all domains distilled (survey heuristic)" + caveat → STOP (no read, no mutation)
│     └─ else build the flagged-domain WORKLIST (index-table order) → Step 6 ALL-DOMAINS LOOP:
│          iterate every flagged domain sequentially (SURVEY ONCE — no re-survey between domains),
│          running the one-domain flow below as the loop body per domain. MAIN SESSION (no per-domain dispatch).
│          per-domain approval is the unit; a skipped domain stays untouched + loop continues; an
│          already-distilled worklist domain reports "already distilled" + continues; an exit-2 within one
│          domain follows per-domain handling + does NOT swallow the rest. terminal: all-distilled or skipped/remaining.
│  <domain> GIVEN → override: skip survey, force a full read of that ONE domain (one-domain flow once, NO loop)
│
├─ Pre-flight: docs/memory/index.md exists; the resolved/named docs/memory/{domain}/ exists with ≥1 topic file
├─ [one-domain flow — the approval/apply unit; loop body on no-arg, runs once on explicit <domain>]
├─ Read (read-only): domain index + every topic file (recursing sub-domains) + $(fab kit-path)/reference/fkf.md
│     skip index.md / log.md (generated), log.seed.md (curated read-only seed input, never generated), and _shared/removed-domains.md (tombstone exempt)
├─ Classify per file: transition narration / superseded-state prose / description: defects (over-cap, change-ids)
│     / change-id heading suffixes (STRIP, registry-gated) / byte-identical duplicate blocks (DEDUP; near-dup → FLAG only)
│     / DD changelog bullets (REWRITE to four-field, or REMOVE if pure history; never fabricate rationale)
│     / embedded operational TODOs (RELOCATE → fab/backlog.md, never delete)
│     / rationale-carrying narration (RELOCATE) / allowed provenance (KEEP)
│     removal candidate carries durable intent? → relocate; else intent-free + recorded elsewhere (log.md/git/archive) → delete
├─ Report: per-file proposed rewrites (before/after for the non-obvious; every relocation shown; near-dups flagged not auto-merged)
├─ (present report, ask for approval: apply all / cherry-pick / skip)
│  └─ [if declined or already-distilled] report, stop — no mutation
│
├─ [if approved]
│  ├─ per approved file — Edit: rewrite body to present truth (remove approved narration; strip change-id
│  │        heading suffixes; dedup byte-identical blocks; rewrite/remove DD changelog bullets; relocate
│  │        rationale → Design Decisions Why/Rejected; preserve trailing (change-id) + *Introduced by*); fix
│  │        description: (strip change-ids, cap ≤500 chars, displaced detail → body); stamp type: memory if legacy file lacks it
│  ├─ relocate approved operational TODOs → append to fab/backlog.md (## Open; create with # Backlog header if absent) — never delete
│  │
│  └─ once, after ALL approved rewrites (Step 5 — not per file):
│     ├─ Bash: fab memory-index --check  → exit 0/1 regenerate; exit 2 REFUSE (→ /docs-reorg-memory pointer)
│     └─ Bash: fab memory-index          (index tiers from folder contents + description: frontmatter; each
│              log.md from git history + per-change summaries, freeze-on-write append-only; byte-stable)
│
└─ Dynamic Next: line — reports surveyed SKIPPED/REMAINING domains (with flagged-file counts, index.md order)
      as a follow-up targeted-run pointer, or "all domains distilled" when none remain. It reports surveyed
      truth; it no longer drives per-domain re-invocation (the no-arg loop already processes every flagged domain).
      no-arg: initial Step 0 survey minus every domain fully distilled this run (a skipped/partially-cherry-picked
      domain stays listed while still flagged); explicit <domain>: run the survey at completion to populate it.
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | The target domain's index + topic files + `$(fab kit-path)/reference/fkf.md` (read-only analysis). **Survey mode (no-arg)**: reads the JSON `malformed[]`/`warnings[]` output of one `fab memory-index --check --json` call — NOT the corpus (only the older-binary fallback reads every topic file's frontmatter and body) |
| Edit/Write | Rewritten topic-file bodies + `description:` frontmatter to FKF present-truth style (strip change-id heading suffixes, dedup byte-identical blocks, rewrite/remove DD changelog bullets), only with approval; never `index.md`/`log.md`/`log.seed.md`, never `_shared/removed-domains.md`. **`fab/backlog.md`** — the one file outside `docs/memory/` written (operational-TODO relocation; created with a `# Backlog` header when absent) |
| Bash | **Survey mode**: `fab memory-index --check --json` (the canonical signal source — aggregate `malformed[]` `description-change-id`/`description-over-cap` + `warnings[]` `description-length`/`narration-density`); older-binary fallback ⇒ `grep` for narration markers across all domains' topic-file bodies + "upgrade fab" warning. After approved rewrites: `fab memory-index --check` (refuse-before-regen: exit 2 → refuse + reorg pointer) and `fab memory-index` to regenerate indexes/logs |

### Sub-agents

None — the skill runs inline, including the no-arg all-domains loop (main session, no per-domain dispatch — the per-domain approval prompt is interactive and must reach the user).

### Bookkeeping

None — no `fab status` transition; the skill advances no pipeline stage and requires no active change.

---

*Mirror of `src/kit/skills/docs-distill-memory.md`. Introduced by 260717-dgp8-docs-distill-memory-skill (260717-dgp8).*
