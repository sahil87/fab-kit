# Intake: Reorg-Memory Orchestrates Frontmatter Backfill for Pre-fab-kit Trees

**Change**: 260614-5ewp-reorg-memory-backfill-orchestration
**Created**: 2026-06-14

## Origin

> reorg-memory orchestrates frontmatter backfill for pre-fab-kit memory trees

Initiated conversationally after diagnosing a real-world failure: a repo that adopted fab-kit over a **pre-existing, hand-curated** `docs/memory/` tree found that `fab memory-index` was net-destructive when run as-is. The tree had **zero `description:` frontmatter** on its 196 content files, so the generator (which reads descriptions exclusively from frontmatter — `src/go/fab/internal/memoryindex/memoryindex.go:303` `frontmatter.Field(path, "description")`) would emit `—` for every row, wiping all curated descriptions.

Key facts established during the discussion (verified against the codebase):

- `fab memory-index` and the hydrate skill already agree on a single source of truth: **descriptions live in each file's `description:` frontmatter**; index files are generated, "do not hand-edit" artifacts (`docs-hydrate-memory.md` § Index Ownership; `memoryindex.go:142,168` banners). A *fresh* fab-kit repo is therefore born compatible — every file hydrate creates leads with a `description:` line (`docs-hydrate-memory.md:80,83,138-140`).
- The legacy repo's failure is **not a fab-kit bug** — it's an un-migrated pre-fab-kit memory layout. The fix is a one-time **frontmatter backfill**.
- **Backfill = hydrate's existing "merge into existing file" path applied to files missing frontmatter** — the synthesis muscle (summarize content → write a one-liner) is hydrate's, not reorg's.
- **Detection of the gap belongs in reorg** — it already reads every memory file, diagnoses tree shape, and produces an approve-before-mutate findings report. "Files lack the frontmatter the index depends on" is a shape diagnosis.
- **Two losses are irreversible and backfill cannot recover them**: (a) *tombstone rows* — removed-domain history + the change IDs explaining each removal (the generator only walks folders that exist on disk — `memoryindex.go:225-266`); (b) *custom structural groupings* (Apps/Packages/Cross-cutting) — the root index is domains-only by design (`memoryindex.go:132`).
- **Specs are explicitly out of scope** (verified): there is **no specs-index generator** (no counterpart to `internal/memoryindex`), and `docs-reorg-specs` rewrites `docs/specs/index.md` *by hand* (`docs-reorg-specs.md:79`). A spec missing frontmatter breaks nothing downstream — there is no compatibility contract to violate. Constitution VI keeps specs human-curated; adding spec-frontmatter backfill would invent a non-problem and push specs toward a generated-index model the constitution rejects.

Decided interactively in this session (two SRAD-Unresolved points asked):
- **Tombstone handling → auto-relocate, then proceed**: reorg moves tombstone rows into a generated `docs/memory/_shared/removed-domains.md` (a real topic file that round-trips through frontmatter), then continues. One command, no manual hand-off.
- **Trigger → auto-detect, prompt on approval**: reorg scans for missing frontmatter as part of its normal diagnosis (no new flag); if found, it proposes backfill in the findings report and runs it only on user approval — consistent with reorg's existing posture.

## Why

**Problem.** A repo adopting fab-kit over an existing hand-curated `docs/memory/` tree has no safe path to the fab-kit convention. Running `fab memory-index` (which the hydrate skills invoke routinely) silently destroys curated descriptions, removal-history tombstones, and structural groupings. The user only discovers this *after* the damage — or, if cautious, abandons `fab memory-index` entirely, losing the anti-drift / `--check` CI guard the command exists to provide.

**Consequence if unfixed.** fab-kit is effectively unusable on any repo with a pre-existing memory tree without manual, undocumented, error-prone surgery. Adoption friction is high precisely at the moment of adoption. The most valuable loss (tombstone history — decision context derivable from nothing else) happens silently.

**Why this approach.**
1. **reorg as the single front door** delivers true one-command DX (`/docs-reorg-memory`): the user types one skill and reorg sequences detect → backfill → tombstone-relocate → rebalance → regenerate. This is why detect-and-point (hand the user off to hydrate) was rejected — it's three invocations of two skills, not one command.
2. **Competence seam preserved.** Per-file *content synthesis* (writing a description from file content) stays in hydrate, which already owns memory-content authoring. reorg **calls** hydrate's backfill as a sub-agent rather than absorbing it — mirroring how `/fab-proceed` orchestrates sub-skills without absorbing their logic. reorg's own job stays structural (detect, relocate mechanical rows, rebalance).
3. **Tombstone auto-relocate over hard-block** was chosen for DX (one command). The tradeoff — reorg authors one mechanical content file — is acceptable because relocating existing rows is mechanical movement, not prose synthesis; the genuine synthesis (per-file descriptions) still routes to hydrate.

**Alternatives rejected** (from the discussion):
- *New top-level skill* (`/docs-make-compatible`, `/docs-refresh`): duplicates ~80% of hydrate's logic, splits "get memory into the convention" across two skills, and names the implementation rather than a user intent. A once-per-repo-lifetime task is poor standing-command surface.
- *Backfill mode lives in reorg* (synthesis in reorg): crosses the restructure/author seam — would make two skills both author memory content.
- *Migration file* (`src/kit/migrations/`, applied via `/fab-setup migrations`): heavier; only pays off if many repos adopt fab-kit over existing trees. Not mutually exclusive with this change — a future migration could *orchestrate* the same two skills. Deferred.
- *Warn-only on tombstones*: irreversible loss behind an advisory warning = silent data loss. Rejected.
- *Specs parity* (`/docs-reorg-specs` backfill mode): no compatibility gap exists on the specs side (verified). Out of scope.

## What Changes

Two skill files change. No Go changes (the `fab memory-index` generator and `--check` flag already exist and are correct — `memory_index.go:104`).

### 1. `docs-reorg-memory.md` — add backfill orchestration + tombstone guard

reorg gains a **compatibility-detection** step folded into its existing diagnosis (Step 1 reads all memory files anyway), and an **orchestration** step that runs before its normal rebalance.

**Detection (additive to the findings report).** During the read-all-files pass, detect:
- **Missing frontmatter**: topic files (non-`index.md` `.md` files) lacking a `description:` frontmatter field. Reuse `frontmatter.Field` semantics — a file with no frontmatter or no `description:` key counts as missing.
- **Tombstone rows**: rows in the *existing* hand-curated index files whose `docs/memory/` relative link target is **absent on disk** (the primary signal, assumption #10). Strikethrough syntax (`~~lib-bdash~~`) is a corroborating hint that raises confidence but is not required — un-struck tombstones are still caught. Scoping the signal to `docs/memory/`-relative paths avoids false positives on intentional external links. Candidates are **surfaced for user confirmation** before any relocation. These are the removal-history rows the generator will drop.
- **Custom groupings**: structural headings in the existing root `index.md` beyond the generated domains-only table (e.g., `### Apps`, `### Packages`, `### Cross-cutting`) — content that the domains-only regen will flatten.

**Findings report.** When any of the above is found, surface it in reorg's existing approve-before-mutate report, e.g.:

```
## Compatibility (pre-fab-kit memory tree detected)

- 12 topic files lack `description:` frontmatter (will render as — on regen)
- 6 tombstone rows reference removed folders (will be dropped by fab memory-index)
- Grouped layout (Apps / Packages / Cross-cutting) will flatten to a domains-only table

Proposed remediation (on approval):
  1. Relocate tombstone rows → docs/memory/_shared/removed-domains.md
  2. Backfill description: frontmatter (12 files, via docs-hydrate-memory)
  3. Rebalance + regenerate indexes (fab memory-index)
```

**Orchestration (on approval), in strict order:**

1. **Relocate tombstones → `docs/memory/_shared/removed-domains.md`.** reorg authors this single mechanical file: a `description:` frontmatter one-liner (so it round-trips), an H1, and the tombstone rows lifted verbatim from the old index (preserving the change IDs that explain each removal). If the file already exists, **merge** new tombstone rows without duplicating existing ones (idempotency). This is the **one** content-authoring action reorg performs — bounded to mechanical row relocation, explicitly NOT per-file description synthesis.
2. **Dispatch backfill to hydrate** as a general-purpose sub-agent (per `_preamble.md` § Subagent Dispatch — standard subagent context, the 5 project files). The sub-agent runs `/docs-hydrate-memory`'s **backfill mode** (see change 2) with the instruction to backfill the tree — it **re-scans `docs/memory/` independently** to find files missing `description:` (no manifest is passed from reorg; the seam stays loose, see assumption #9). Synthesis lives there.
3. **Rebalance + regenerate** as reorg already does — its existing split/merge/flatten logic, then `fab memory-index`.

**Approval gate.** Backfill + relocation run only on explicit user approval (reorg's existing posture). If the user declines, reorg reports the compatibility findings and stops without mutating — the user keeps their hand-curated tree intact.

**No specs symmetry.** Add an explicit note to `docs-reorg-specs.md` (one line) stating that no backfill/compatibility step applies to specs, with the rationale (no generator; hand-curated index). This prevents a future contributor from "fixing the asymmetry."

### 2. `docs-hydrate-memory.md` — add backfill mode

A third mode alongside ingest and generate, callable both directly by a user and as the sub-agent reorg dispatches.

**Trigger.** Explicit invocation over an existing tree where topic files lack `description:` frontmatter. (Direct user form and the reorg-dispatched form both target already-existing files.) Distinguish from generate mode: generate **creates** files from source-code gaps; backfill **adds frontmatter to existing** memory files without changing their body.

**File discovery.** Backfill **re-scans `docs/memory/` itself** to find every topic file (non-`index.md` `.md`) lacking a `description:` field — it does not receive a file list from its caller (assumption #9). This holds for both the direct-user form and the reorg-dispatched form: reorg's dispatch prompt names the operation ("backfill this tree"), not the files. Scanning is idempotent — files that already have `description:` are skipped, so a second pass is a no-op.

**Behavior.**
1. For each discovered topic file missing `description:`, read the file's **own content** (Overview / first section / H1) and synthesize a one-line `description:`. Where an existing curated index row maps to the file (they line up file-by-file, as observed in the legacy repo), prefer the curated text as the source — it's higher quality than re-synthesis.
2. Write the `description:` frontmatter as the leading block of the file. **Body is preserved byte-for-byte** — backfill only prepends/edits frontmatter, never the content. Files that already have `description:` are skipped (idempotent).
3. Also create any missing domain/sub-domain `index.md` **stub** (description-only) the same way ingest/generate modes do, so `fab memory-index` has the domain description to read.
4. Do **not** run `fab memory-index` when invoked as reorg's sub-agent (reorg runs it once at the end of orchestration). When invoked directly by a user, run it as the final step like the other modes. (The mode needs to know its caller — pass a flag/prefix in the dispatch prompt, or have backfill always defer regen to the caller and have the direct-user path regen explicitly.)

**Mode routing.** Update the Argument Classification table / mode-routing prose so backfill is reachable and unambiguous relative to ingest/generate.

### 3. SPEC mirrors (constitution requirement)

Per constitution: "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file." Update:
- `docs/specs/skills/SPEC-docs-reorg-memory.md` — orchestration + tombstone guard + detection
- `docs/specs/skills/SPEC-docs-hydrate-memory.md` — backfill mode
- `docs/specs/skills/SPEC-docs-reorg-specs.md` — the explicit no-symmetry note (if the file exists / if the one-line note lands in the skill)

### Canonical-source note

All skill edits target `src/kit/skills/*.md` (the canonical source), **never** `.claude/skills/` (gitignored deployed copies — `fab/project/context.md:9`). Verify the deployed copy via `fab sync` after editing if testing locally.

## Affected Memory

- `memory-docs/hydrate`: (modify) `/docs-hydrate-memory` gains a backfill mode (third mode beside ingest/generate) — adds `description:` frontmatter to existing files, body-preserving, caller-aware regen deferral
- `memory-docs/specs-index`: (modify) record the verified no-generator / hand-curated-index asymmetry that makes specs out of scope for compatibility backfill — and the explicit no-symmetry note added to `docs-reorg-specs`
- `memory-docs/hydrate-generate`: (modify, possibly) if backfill is documented near generate-mode placement rules, note the create-vs-add-frontmatter distinction
- Note: `docs-reorg-memory` has no dedicated memory file in the current index (reorg behavior is covered under `memory-docs`); hydrate's index entry is the primary home. The hydrate authoring memory should capture reorg's new orchestration role (reorg calls hydrate's backfill).

## Impact

- **Skill files** (canonical source): `src/kit/skills/docs-reorg-memory.md`, `src/kit/skills/docs-hydrate-memory.md`, one-line note in `src/kit/skills/docs-reorg-specs.md`.
- **SPEC mirrors**: `docs/specs/skills/SPEC-docs-reorg-memory.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-docs-reorg-specs.md`.
- **Memory** (hydrate stage): `docs/memory/memory-docs/hydrate.md`, `specs-index.md`, possibly `hydrate-generate.md`.
- **No Go changes.** The generator, `--check`, and `frontmatter.Field` already exist and are correct. This change is purely skill-prose orchestration.
- **No new fab subcommand, no migration** (deferred — see rejected alternatives).
- **Cross-skill dependency**: reorg now dispatches hydrate as a sub-agent. Both skills must land together; the dispatch contract (which files to backfill, regen-deferral signal) is the integration seam to get right.
- **Idempotency** (constitution III): re-running reorg on an already-converted tree must be a no-op — no duplicate tombstone rows, frontmatter-present files skipped, byte-stable index. This is a primary acceptance concern.

## Open Questions

*(Resolved via /fab-clarify 2026-06-14 — see ## Clarifications and ## Assumptions #9, #10.)*

- ~~How does backfill receive the list of files to process?~~ **Resolved**: backfill re-scans `docs/memory/` independently; no manifest passed (assumption #9).
- ~~Tombstone detection heuristic?~~ **Resolved**: unresolved `docs/memory/`-relative link target (primary) + strikethrough hint + user confirmation (assumption #10).
- Should the direct-user backfill mode also detect/relocate tombstones, or is that reorg-only? **Leaning reorg-only** (not blocking): backfill stays a pure frontmatter operation; tombstone detection/relocation is reorg's structural concern. To be finalized at apply as an inline SRAD assumption if still open.

## Clarifications

### Session 2026-06-14 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | Migration not required |

### Session 2026-06-14

| # | Q | A |
|---|---|---|
| 9 | How does hydrate's backfill mode learn which files to process (re-scan vs. dispatched manifest)? | Re-scan independently — reorg dispatches "backfill this tree"; hydrate walks `docs/memory/` itself. Loose seam, idempotent. |
| 10 | How are tombstone rows detected (unresolved-link vs. strikethrough)? | Both — unresolved `docs/memory/` link target is the primary signal, strikethrough a corroborating hint, always user-confirmed before relocation. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Specs are out of scope — no backfill/compatibility step for `docs-reorg-specs` | Verified: no specs-index generator exists; `docs-reorg-specs.md:79` hand-rewrites the index; constitution VI keeps specs human-curated. No compatibility contract to violate. | S:95 R:90 A:100 D:95 |
| 2 | Certain | No Go changes — generator, `--check`, `frontmatter.Field` already exist and are correct | Read `memoryindex.go` + `memory_index.go`; the generation/check machinery is complete. Change is skill-prose only. | S:90 R:85 A:100 D:95 |
| 3 | Certain | All skill edits target `src/kit/skills/`, never `.claude/skills/` | `fab/project/context.md:9` + constitution: canonical source vs. gitignored deployed copies. | S:100 R:80 A:100 D:100 |
| 4 | Certain | SPEC mirrors must be updated for every skill file changed | Constitution: "Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`." | S:100 R:75 A:100 D:100 |
| 5 | Confident | reorg orchestrates (detect + relocate + dispatch); hydrate owns per-file synthesis | Clarified — user confirmed. Preserves the restructure/author competence seam; mirrors `/fab-proceed` orchestration. | S:95 R:65 A:80 D:80 |
| 6 | Confident | Backfill is a mode of `/docs-hydrate-memory`, not a new top-level skill | Clarified — user confirmed. Avoids ~80% logic duplication; once-per-repo task is poor standing-command surface. | S:95 R:70 A:85 D:85 |
| 7 | Certain | Backfill is body-preserving — only prepends/edits `description:` frontmatter | Clarified — user confirmed. Idempotency (constitution III) + the convention: descriptions live in frontmatter, body is the user's content. | S:95 R:75 A:90 D:85 |
| 8 | Confident | Migration file deferred — not part of this change | Clarified — user confirmed migration is not required. Heavier, only pays off at scale; a future migration could orchestrate these skills. | S:95 R:80 A:75 D:80 |
| 9 | Confident | Backfill re-scans `docs/memory/` independently to find files missing `description:` — reorg dispatches "backfill this tree", no manifest passed | Clarified — user chose re-scan over manifest. Loose seam between two independently-invocable skills; idempotent; trivial double-scan cost; robust to drift. | S:95 R:65 A:55 D:85 |
| 10 | Confident | Tombstone detection = index row whose `docs/memory/` relative link target is absent on disk (primary signal), strikethrough `~~...~~` as a corroborating hint, always user-confirmed before relocation | Clarified — user chose the combined rule. Catches struck and un-struck tombstones; relative-path scope avoids external-link false positives; confirmation gates relocation. | S:95 R:60 A:60 D:85 |

10 assumptions (5 certain, 5 confident, 0 tentative).
