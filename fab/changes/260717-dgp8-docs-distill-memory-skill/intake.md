# Intake: docs-distill-memory Skill

**Change**: 260717-dgp8-docs-distill-memory-skill
**Created**: 2026-07-17

## Origin

Promptless dispatch (Create-Intake Procedure, `{questioning-mode} = promptless-defer`) from a synthesized user-conversation description. No questions were asked; every would-be-asked decision is a deferred Unresolved row in `## Assumptions`.

> Create the new user-invocable `docs-distill-memory` skill — the deliberately-deferred step 3 of the present-truth effort. Steps 1–2 shipped as change `260717-3plm-fkf-present-truth-style-rule` (PR #485, merged to main as `a4511d39`): FKF §3.3 now carries the normative present-truth body-style rule, §3.2 bans change-ids in `description:` frontmatter, and the memory writers (hydrate, docs-hydrate-memory, docs-reorg-memory, memory template) write topic-keyed current truth. Those rules stop NEW deltas; this skill cleans the EXISTING corpus.

Verified against the current tree (this worktree's `docs/`, `src/kit/`, and `src/go/` content is diff-identical to `origin/main` at `a4511d39`):

- FKF §3.2 (`description` rules incl. the 500-char cap and the no-change-ids ban) and §3.3 (present-truth body style incl. the tombstone carve-out) exist in **both** `docs/specs/fkf.md` (§3.2 at line 87, §3.3 at line 133) and `src/kit/reference/fkf.md` (§3.2 at line 59, §3.3 at line 103).
- The corpus baseline was re-measured on this tree (numbers below in Why) — the conversation's approximate figures (~460 references, worst file ~67) were re-grounded to measured values.

## Why

**Problem.** The FKF present-truth rules (shipped in 3plm) are forward-looking: they govern what memory writers produce from now on. The existing `docs/memory/` corpus predates them and violates them extensively. Measured on the current tree (dated-form regex `2[0-9]{5}-[a-z0-9]{4}`, excluding generated `index.md`/`log.md`/`log.seed.md`):

- **~404 dated change-id occurrences across 23 topic files.** Worst: `pipeline/planning-skills.md` (55), `distribution/kit-architecture.md` (55), `runtime/operator.md` (41), `pipeline/execution-skills.md` (40), `distribution/distribution.md` (34), `_shared/context-loading.md` (25), `pipeline/change-lifecycle.md` (24). Bare 4-char id references (e.g. `zc9m`, `ioku`, `szxd` woven into prose) are additional and uncounted by that regex.
- **≥12 `description:` frontmatter values exceed the §3.2 500-character cap** — worst `distribution/kit-architecture.md` (~5.9k chars), then `_shared/configuration.md` (~4.2k), `distribution/migrations.md` (~3.9k), `runtime/operator.md` (~2.8k), `pipeline/planning-skills.md` (~2.8k).
- **Several descriptions carry change-ids**, banned outright by §3.2: `runtime/operator.md` (a dated `260703-gvxd` plus ~12 bare ids), `memory-docs/specs-index.md` (`uliv`, `d9rs`, `5ewp`), `memory-docs/hydrate-specs.md` (`d9rs`).
- Bodies are dense with **transition narration and superseded-state descriptions** ("superseding the historical operator4", "previously the doing tier via …, 2sdj", "renamed X→Y in {id}") — exactly what §3.3 now prohibits.

**Consequence if not fixed.** The always-load and selective-load context layers pay the token cost of accumulated narration on every skill invocation; agents reading transition narration can mistake superseded behavior for current contract; and the corpus itself contradicts the normative format it is supposed to exemplify — every future hydrate merges current truth into files that are not current truth.

**Why this approach.** No existing mechanism covers this (gap analysis):

- `/docs-reorg-memory` reorganizes **structure** (splits/merges/moves, link rewrites) — it never rewrites body prose to a style.
- `/docs-hydrate-memory` backfill mode is explicitly **body-preserving** (adds frontmatter only); ingest/generate author new content, they don't clean existing files.
- `/fab-continue` hydrate writes each change's delta as current truth going forward but only touches sections its change affects.
- `internal-skill-optimize` condenses **skills**, not memory.

A dedicated, user-invocable skill was chosen in conversation; making it a `docs-reorg-memory` mode was explicitly rejected ("If its a mode it wont be discoverable").

## What Changes

### 1. New skill: `src/kit/skills/docs-distill-memory.md`

A user-invocable skill that rewrites existing `docs/memory/` topic files to the FKF §3.2/§3.3 present-truth style. Core behavior (all decided in conversation):

**Scope of a run — one domain per run, propose-then-apply.** The user names a domain (e.g. `/docs-distill-memory pipeline`); the skill runs read-only analysis over that domain's topic files, produces a per-file report of proposed rewrites (per-file diffs/summaries), and applies only on explicit user approval — the same propose-then-apply posture as `/docs-reorg-memory` (its Step 4 report → Step 5 confirmation & apply shape is the sibling pattern). NOT fully autonomous bulk rewriting: these files encode load-bearing behavioral contracts, so a human approves per domain seeing per-file diffs.

**What a rewrite does** (the normative source is FKF §3.3 "Body style: state current truth in present tense" and §3.2, quoted here for state transfer):

- Removes **transition narration** — no "renamed X→Y in {id}", no "this inverts/supersedes {id}'s claim", no "was `old.value`", no "superseding the historical …".
- Removes **superseded-state descriptions** — the body carries only what IS; previous states belong to the per-folder generated `log.md`, git history, and archived change folders.
- Keeps **allowed provenance**: trailing `(change-id)` citations and the `*Introduced by*: {change-name}` field on Design Decisions. Per §3.3: "Citations are deliberately preserved — a 6-char `(id)` cheaply defends a deliberate, easily-'fixed'-away behavior against future regressions." Bare 4-char ids count the same as dated ids: in trailing-citation position they stay; woven into narration they go with the narration.
- Strips change-ids from `description:` frontmatter (§3.2: "The description MUST NOT carry change-ids — neither a trailing `— xu0k`-style suffix nor a `(d9rs)`-style citation") and compresses over-cap descriptions to the ≤500-character routing-signal shape, moving displaced detail into the body where it isn't already there.

**Rationale-preservation guard (the critical constraint).** Token savings come from dropping narration, NEVER rationale — §3.3 verbatim: "'Don't re-break this' content lives in Design Decisions' `Why` / `Rejected` as durable, present-tense design intent — a rejected alternative is a design *fact*, not transition narration." The skill RELOCATES don't-re-break/deliberate-behavior content into Design Decisions (`Why`/`Rejected`) rather than deleting it. This repo's history shows agents repeatedly "fixing" deliberate behavior (e.g. the Copilot poll-predicate) — the distilled file must retain those defenses. Deletion is safe only for narration whose content is already recorded elsewhere (per-folder `log.md`, git history, archived change folders); content recorded nowhere else and carrying intent is relocated, not dropped.

**Generated files are never hand-edited.** `index.md` (root/domain/sub-domain tiers), `log.md`, and `log.seed.md` are untouched except by regenerating via `fab memory-index` after applying rewrites. The skill heeds the refuse-before-regen convention: consult `fab memory-index --check` first and refuse to regenerate on exit 2 (destructive loss), surfacing the existing `→ run /docs-reorg-memory to remediate …` pointer (same guard `/docs-hydrate-memory` carries; `_cli-fab` § fab memory-index documents the exit tiers).

**Exemption.** `docs/memory/_shared/removed-domains.md` is exempt — the §3.3 tombstone carve-out ("whose body *is* removal records — a citation-carrying tombstone ledger, not transition narration"). Note: fab-kit's own tree currently has no such file (`_shared/` holds only `configuration.md`, `context-loading.md`, and generated files); the exemption matters because the skill ships to user projects where `/docs-reorg-memory` authors that file.

**Idempotent (Constitution III).** Re-running on an already-distilled domain finds nothing to do and reports that; `fab memory-index` regeneration is byte-stable.

**Skill-file conventions** — follow the New Skill Checklist (`docs/specs/skills.md` § New Skill Checklist, line 117: eight integration points):

1. Frontmatter: `name: docs-distill-memory` + a behavior-naming `description` (one-liner; itself change-id-free).
2. Standard preamble-read blockquote (as `docs-hydrate-memory.md` carries).
3. `helpers:` — none expected; reference `_cli-fab` § fab memory-index by in-body pointer, the `docs-reorg-memory` sibling pattern. An explicit `## Context Loading` section defines the skill's reduced load (memory indexes + the target domain's files + `$(fab kit-path)/reference/fkf.md`; no active change required) — the skill file wins over the always-load layer per `_preamble` §1.
4. Output ends with a `Next:` line.
5. Body closes with Error Handling + Key Properties tables (mirror `docs-reorg-memory.md`'s, e.g. "Advances stage? No", "Requires active change? No", "Idempotent? Yes", "Indexes hand-edited? No — regenerated by `fab memory-index`").
6–8. SPEC mirror, skills.md row, help grouping — sections 2–3 below.

### 2. SPEC mirror + aggregate/inventory surfaces (the sweep class)

Verified per-surface on the current tree — follow the `docs-reorg-memory`/`docs-hydrate-memory` sibling pattern:

| Surface | Action | Verified detail |
|---------|--------|-----------------|
| `docs/specs/skills/SPEC-docs-distill-memory.md` | **new** | Constitution-required mirror; naming policy is mechanical `SPEC-{source-filename}.md` (per `docs/memory/memory-docs/specs-index.md` and the New Skill Checklist item 6) |
| `docs/specs/skills.md` | **modify** | Add the skill's own section (sibling: `## /docs-reorg-memory` at ~line 741); no § Skill Helpers row needed if no `helpers:` declared |
| `docs/specs/glossary.md` | **modify** | Add a command row (sibling: `/docs-reorg-memory` row at line 57) |
| `README.md` | **modify** | Add a row to the command table (the `/docs-*` block at lines 459–462) |
| `src/go/fab/cmd/fab/fabhelp.go` | **modify (deferred — see Assumptions #11)** | Checklist item 8: add `"docs-distill-memory": "Maintenance"` to `skillToGroupMap` (line ~42; unmapped skills fall into the "Other" bucket), plus the hardcoded `Maintain docs:` TYPICAL FLOW line (line ~156), plus `fabhelp_test.go` expectations (line ~124). No command signature changes ⇒ no `_cli-fab.md` edit |
| `src/kit/skills/fab-help.md` | **no edit** | Verified: `/fab-help` runs `fab fab-help`, which scans deployed skill frontmatter dynamically — the new skill auto-appears; only its *grouping* comes from `skillToGroupMap` |
| `docs/specs/user-flow.md` | **no edit** | Verified: contains no `docs-*` command inventory (pipeline `/fab-*` diagrams only; points to README for command coverage) |
| sync/distribution manifest | **none exists** | Verified: `fab sync` deploys via `listSkills` = `os.ReadDir` of the kit `skills/` dir (`src/go/fab-kit/internal/skills.go`) — presence in `src/kit/skills/` suffices; ships via normal release/sync. Kit-canonical placement only, never `.claude/skills/` (gitignored deployed copies) |

### 3. What does NOT change

- **No Go/CLI behavior change**: `fab memory-index` unchanged; no new subcommands; no migration (`src/kit/migrations/` untouched — no user data is restructured). The only candidate Go touch is the fabhelp.go registration above (deferred decision).
- **No template change**: `src/kit/templates/memory.md` already carries the FKF shape (shipped in 3plm).
- **FKF spec unchanged**: §3.2/§3.3 already state the rules this skill enforces.

## Affected Memory

- `memory-docs/distill`: (new) — the skill's behavior doc, authored at hydrate; sibling of `memory-docs/hydrate` (which documents `/docs-hydrate-memory`)
- `memory-docs/templates`: (modify) — its Memory Tree Shape / format section homes the docs-reorg-memory rebalancer today; gains the cross-pointer that corpus-style remediation is `/docs-distill-memory` (small, additive)
- `_shared/context-loading`: (modify) — its § Exception Skills enumerates the shipped Context-Loading-override skill set; `/docs-distill-memory` joins that set (added during review rework — the documented cross-cutting-`_shared/`-prose under-coverage class)

## Impact

- **Prose**: 1 new skill file (`src/kit/skills/docs-distill-memory.md`), 1 new SPEC mirror, one-row/section edits to `docs/specs/skills.md`, `docs/specs/glossary.md`, `README.md`.
- **Go** (pending the deferred fabhelp decision): ~2 lines in `src/go/fab/cmd/fab/fabhelp.go` + a `fabhelp_test.go` expectation update. Constitution's CLI constraint requires the test update; `_cli-fab.md` is unaffected (no command-signature change).
- **Runtime surface**: none — the skill is markdown driven by an agent; the only binaries it invokes already exist (`fab memory-index`, `fab kit-path`).
- **Out of scope** (recorded from conversation): actually RUNNING the distillation on fab-kit's own corpus (a later invocation of the shipped skill); extending `fab memory-index` validation; any other Go change; rewriting specs (FKF governs `docs/memory/` only — specs are human-curated and out of FKF scope per Constitution VI and the fkf.md scope note).

**Rejected alternatives (recorded from conversation, binding)**:
- A `docs-reorg-memory` mode — rejected for discoverability ("If its a mode it wont be discoverable").
- Autonomous full-corpus bulk rewrite in one run — too risky for load-bearing contract docs; per-domain approval chosen.
- Zero-provenance stripping — citations stay allowed per FKF §3.3 (a trailing `(id)` cheaply defends deliberate behavior).

## Open Questions

- Should the new skill's registration include the small Go edit the New Skill Checklist item 8 requires (`skillToGroupMap` entry + test), given the conversation's "prose-only, NO Go/CLI change" decision? Without it the skill still appears in `/fab-help` but under the "Other" bucket, and the checklist says to work through all eight points. (Deferred — see Assumptions #11.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Separate NEW user-invocable skill named `docs-distill-memory`, not a `docs-reorg-memory` mode | Discussed — user explicitly rejected the mode for discoverability | S:95 R:60 A:90 D:95 |
| 2 | Certain | Kit-canonical placement `src/kit/skills/docs-distill-memory.md`; ships via normal release/sync; no manifest registration needed | Discussed + verified: `fab sync` enumerates the kit skills dir via `os.ReadDir` (`fab-kit/internal/skills.go`) | S:90 R:85 A:95 D:95 |
| 3 | Certain | One domain per run with per-file diff review: read-only analysis → report → user approval → apply (the `docs-reorg-memory` propose-then-apply posture); never autonomous bulk rewrite | Discussed — binding; load-bearing contract docs need a human gate | S:95 R:70 A:85 D:90 |
| 4 | Certain | Rationale-preservation guard: don't-re-break/deliberate-behavior content is RELOCATED into Design Decisions (`Why`/`Rejected`), never deleted; trailing `(change-id)` citations and `*Introduced by*` kept; deletion only for narration recorded elsewhere (log.md/git/archive) | Discussed — the critical constraint; verbatim normative text in FKF §3.3 | S:95 R:75 A:95 D:95 |
| 5 | Certain | Generated files (`index.md`, `log.md`, `log.seed.md`) never hand-edited — regen via `fab memory-index` after rewrites, honoring the refuse-before-regen `--check` exit-2 guard; `_shared/removed-domains.md` exempt (§3.3 tombstone carve-out; absent in fab-kit's own tree, relevant for user projects); idempotent per Constitution III | Discussed + verified against FKF §3.3, `docs-hydrate-memory.md`'s guard text, and `memory_index.go` exit tiers | S:90 R:85 A:95 D:90 |
| 6 | Confident | `/fab-help` group = "Maintenance" (with `docs-reorg-memory`/`docs-reorg-specs`/`docs-hydrate-specs`), not "Setup" (where `docs-hydrate-memory` sits) — applies only if the fabhelp registration (row 11) is approved | Sibling pattern: maintenance/cleanup semantics match the reorg skills | S:60 R:95 A:80 D:70 |
| 7 | Confident | Skill declares no `helpers:`; carries the standard preamble-read line + an explicit `## Context Loading` override (memory indexes + target-domain files + `$(fab kit-path)/reference/fkf.md`; no active change required); references `_cli-fab` § fab memory-index by in-body pointer | Sibling pattern (`docs-reorg-memory` pointer style, `docs-hydrate-memory` preamble line); skill file wins over always-load per `_preamble` §1 | S:55 R:90 A:80 D:70 |
| 8 | Confident | Bare 4-char change-id references are treated identically to dated ids: trailing-citation position stays, narration-woven occurrences go with the narration | FKF §3.3 defines allowed provenance by *position/form* (trailing citation), not by id format | S:60 R:85 A:80 D:70 |
| 9 | Confident | Compressing over-cap `description:` values to ≤500 chars is in scope for every file the run touches (≥12 currently over; worst ~5.9k chars), with displaced routing-irrelevant detail moved into the body where not already present | Discussed ("keeping descriptions ≤500 chars") + §3.2 cap is advisory-but-normative ("routing signal, not a summary of record") | S:70 R:85 A:85 D:75 |
| 10 | Certain | `change_type` = `feat` (new capability) | Verified: `fab change new` inferred `feat`; matches a new user-invocable skill | S:85 R:95 A:95 D:90 |
| 11 | Unresolved | Include the minimal fabhelp.go registration (checklist item 8: `skillToGroupMap` entry + `Maintain docs:` flow line + `fabhelp_test.go` update) despite the conversation's "prose-only, NO Go/CLI change" decision — recommended default: include it (registration, not behavior; no command-signature change, so no `_cli-fab.md` edit; skipping it lands the skill in the "Other" bucket and violates the tree's own eight-point checklist) | Deferred — promptless dispatch | S:45 R:90 A:70 D:50 |

11 assumptions (6 certain, 4 confident, 0 tentative, 1 unresolved).
