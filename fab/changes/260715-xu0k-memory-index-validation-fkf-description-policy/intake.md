# Intake: Memory-Index Frontmatter Validation + Generated-Index Merge Policy + FKF Description-Length Policy

**Change**: 260715-xu0k-memory-index-validation-fkf-description-policy
**Created**: 2026-07-15

## Origin

Created via promptless dispatch (Create-Intake Procedure, `{questioning-mode} = promptless-defer`) from a synthesized description of a live conversation. The user hit recurring merge conflicts in generated `docs/memory/**/index.md` files in the loom repo (a fab-kit consumer). Investigation of one concrete corruption (loom change 260714-9qsj) established three findings, and the user explicitly asked for **one single fab change** covering all three remediations below. The one-time loom data fix is explicitly OUT of scope (handled separately via an operator message).

> Synthesized problem statement: (1) a hydrate edit rewrote a memory file's `description:` frontmatter line and swallowed the newline + closing fence — the file now reads `description: "…text…"---` with the closing `---` glued to the end of the description line (and the file lost its trailing newline); `fab memory-index` passed the garbage value verbatim into the generated index row with no warning and no `--check` failure. (2) Index merge conflicts carry zero information but demand manual resolution — the correct resolution is always "resolve the topic file, re-run `fab memory-index`, take its output wholesale", yet the user hand-merged the index and propagated the corrupted row. (3) The conflict pressure comes from giant descriptions — hydrate agents write ~700-word single-line `description:` values where FKF intends a curated one-liner.

## Why

**1. Silent garbage propagation from malformed frontmatter.** `internal/frontmatter.Field` (`src/go/fab/internal/frontmatter/frontmatter.go`) is a line-based parser: it requires line 1 to be `---`, scans subsequent lines for a standalone closing `---`, and on a `description:` prefix match extracts the value, strips inline comments, then strips surrounding quotes. `stripQuotes` only strips when the value both starts AND ends with the same quote character. The loom corruption — `description: "…text…"---` — fails that check (the line now ends with `-`), so the raw value (leading `"`, trailing `"---`) is returned **verbatim** and rendered into the generated index row. Nothing warns; `fab memory-index --check` exits 0, because the check is a pure drift comparison: the committed garbage row is byte-identical to what regeneration produces from the corrupted source (the index is a pure function of content). Garbage in, garbage out, silently — and `--check` at review-pr cannot catch it. If we don't fix this, every frontmatter-mangling edit in every consuming repo silently corrupts the knowledge index that the always-load context layer routes on.

**2. Generated-index merge conflicts are pure toil and a corruption vector.** The index is deliberately byte-stable and dateless (prior work) — a pure function of the memory tree. But when two branches both edit one file's `description:`, both the topic file AND the generated index row conflict on the same line. The only correct resolution is mechanical: resolve the topic-file conflict, re-run `fab memory-index`, take its output wholesale. `docs/specs/fkf.md` §5 already *hints* at this ("any residual textual conflict auto-resolves by re-running `fab memory-index` post-merge") but no skill or spec states it as a normative never-hand-merge procedure at the seams where agents actually resolve conflicts. In practice the user hand-merged the index and carried the corrupted row from main onto the branch — hand-merging generated files is exactly the failure mode.

**3. Giant descriptions bloat the conflict surface.** FKF §3.2 intends `description:` as a curated one-liner ("One-line summary used by the generated domain-index row") but sets no length bound. Hydrate agents in practice write ~700-word one-line monsters, making every index row a same-line merge hazard and degrading the routing signal the description exists to provide. Without a normative cap read at hydrate time, the pressure that produced findings 1–2 keeps regenerating.

## What Changes

### 1. Malformed-frontmatter detection in `fab memory-index` (Go)

**Detection (in the gather path).** `internal/memoryindex.Gather` already reads every topic file's frontmatter; add validation that detects, per topic file (and domain/sub-domain `index.md` stubs read for descriptions):

- **(a) Unclosed frontmatter block** — the file opens with `---` (line 1) but contains no subsequent standalone `---` line. Note the loom corruption is *also* an instance of (a): gluing the fence onto the description line removes the closing fence entirely.
- **(b) Quote-strip failure on `description:`** — the extracted value begins with `"` (or `'`) but fails `stripQuotes` (does not end with the matching quote) — the glued-fence signature, e.g. a value ending in `"---`.

Detection primitives belong in `internal/frontmatter` (alongside `Field`/`HasFrontmatter` — e.g. a `Validate`-style function returning structured findings) so the parser and its validator share one grammar; `internal/memoryindex` surfaces them as new `Warning` entries (new `Kind` values alongside the existing `"width"`/`"depth"`, with `Warning.String()` cases), sorted into the existing deterministic warning order.

**Surfacing.** Warnings print to stderr alongside the existing shape warnings in `cmd/fab/memory_index.go`. **Byte-stability is a hard constraint**: validation MUST NOT change the rendered index output — `Field`'s return value and both renderers stay byte-identical to today (a malformed value keeps rendering exactly as it does now); validation is stderr + exit-code only.

**`--check` fails on malformed frontmatter (blocking).** Unlike the advisory shape warnings, malformed frontmatter MUST make `fab memory-index --check` exit non-zero **even when every target is byte-identical to its regenerated form** — the drift comparison alone provably exits 0 on the loom case (committed garbage == regenerated garbage), so the validation must run independent of drift. This is what makes corruption block at review-pr (CI/staleness callers treat exit ≥ 1 as fail).
<!-- assumed: malformed frontmatter surfaces as a new blocking category within the existing tiered --check exit-code scheme — front-runner is a tier-2-style blocking failure with its own category + remediation message ("fix the file's frontmatter", distinct from the /docs-reorg-memory pointer), which also makes the hydrate/reorg refuse-before-regen guards (exit == 2) refuse to regenerate over corrupted input; alternative is a plain exit-1 failure independent of the loss tiers. Apply decides; either way the --json schema's category enum ({"category": "description"|"tombstone"|"grouping"}) and its consumer (/docs-reorg-memory compatibility detection) must be checked for compatibility, and the human-readable output must enumerate the offending file(s). -->

**Exit-code integration (design point for apply).** Today `--check` is tiered: 0 clean / 1 benign drift / 2 destructive loss (three index-only loss categories `description`/`tombstone`/`grouping`, `--json` report consumed by `/docs-reorg-memory`). Malformed frontmatter is a new failure class: corruption in the *source* files, not in the index targets. The chosen integration must keep existing consumer contracts working: CI fails on exit ≥ 1; the hydrate/reorg refuse-before-regen guards fire only on exit == 2; `--json` consumers branch on `tier`. Whatever tier/code is chosen, regenerating (non-`--check` runs) still writes — warnings never block the write path.

**Tests (constitution VII, test-alongside).** Ship in the same change:
- `internal/frontmatter/frontmatter_test.go` — the glued-fence regression fixture: a file whose frontmatter reads `description: "text"---` glued on one line with no closing fence and no trailing newline (the loom corruption, verbatim shape); an unclosed-fence fixture; clean quoted/unquoted values keep passing.
- `internal/memoryindex/memoryindex_test.go` (and `loss_test.go` if the classifier is extended) — warnings gathered/sorted; rendered output byte-identical with and without warnings.
- `cmd/fab/memory_index_test.go` — `--check` exits non-zero on a tree whose committed indexes are byte-clean but whose source frontmatter is malformed; `--json` shape stays parseable.

**Docs sweep (constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps).** CLI behavior change ⇒ update `src/kit/skills/_cli-fab.md` § fab memory-index (new warning lines, new `--check` failure semantics, exit-code table) and treat the skill's SPEC mirrors as in-scope (`docs/specs/skills/SPEC-_cli-fab.md` exists on disk — verify and update). Sweep the behavior claims in `docs/specs/fkf.md` (§5 index generation, §6.4 `--check` semantics) + the shipped normative extract `src/kit/reference/fkf.md` (see change 2's dual-file rule), and the memory files that document `--check` tier semantics and memory-index behavior: `docs/memory/pipeline/schemas.md`, `docs/memory/memory-docs/templates.md`, `docs/memory/memory-docs/hydrate.md` (verified — these carry the `--check` loss-tier and index-generation claims).

### 2. Generated-index merge policy: regenerate, don't hand-merge

**Policy home: the FKF spec.** Add normative prose to `docs/specs/fkf.md` §5 (Index Files) — and, because the spec header mandates it, the **shipped normative extract `src/kit/reference/fkf.md`** in the same change ("Any change to FKF normative rules MUST update both files"): on any merge conflict in a generated `docs/memory/**/index.md` or `log.md`, agents MUST NOT hand-merge. Procedure: (1) resolve the conflicts in the *topic files* (and `.status.yaml`/seed inputs) only; (2) re-run `fab memory-index`; (3) take its output wholesale as the resolution. `fab memory-index --check` at review-pr backstops staleness.

**Skill seams (verified by reading the sources).** No skill currently carries conflict-resolution prose (verified: `git-pr.md` and `git-pr-review.md` have none). Land short operational pointers where agents touch these files around merges:
- `src/kit/skills/git-pr.md` — Step 3's index-refresh sub-step (it already owns the `fab memory-index` regen + separate `docs: refresh memory indexes` follow-up commit) gains the never-hand-merge rule for conflicts encountered during ship.
- `src/kit/skills/git-pr-review.md` — the PR-feedback path (where branch↔main divergence and conflict resolution actually surface) gains the same pointer.
- The hydrate seams (`fab-continue.md` Hydrate Behavior, `docs-hydrate-memory.md`) already route all index writes through `fab memory-index`; add at most a one-line cross-reference to the FKF policy.
- SPEC mirror sweep for every touched skill file: `docs/specs/skills/SPEC-git-pr.md`, `SPEC-git-pr-review.md`, `SPEC-fab-continue.md`, `SPEC-docs-hydrate-memory.md` (whole mirror class, not just files carrying the literal phrase).

**Optional documentation-only recipe.** Document (do NOT auto-install — no tooling change, no migration) a `.gitattributes` merge-driver recipe for generated `docs/memory/**/index.md` + `log.md` files (e.g. a driver that takes either side and defers to regeneration), placed with the merge policy in `docs/specs/fkf.md` §5 as a non-normative aside.

### 3. FKF description-length policy, read at hydrate time

**Normative policy in FKF §3.2** (both `docs/specs/fkf.md` AND `src/kit/reference/fkf.md`): `description:` is a **one-line index-row summary** capped at **500 characters** — unit: characters, not words, measured on the description value (the single-line frontmatter scalar, after quote-stripping). <!-- clarified: cap resolved via /fab-clarify 2026-07-15 (user-delegated decision) — 500 chars, grounded in measured description-length distributions across both real trees; see Assumptions #14 / Open Questions --> Detail belongs in the memory file BODY (`## Overview`, `## Requirements`, `## Design Decisions`), never in the description. The description is a routing signal for the always-load layer, not a summary of record.

**Hydrate-time wiring — the policy must be read EVERY time hydrate happens** (user requirement). Verified seams where description authoring is instructed today:
- `src/kit/templates/memory.md` — the `description:` placeholder line (read on demand at every file creation by both hydrate paths); amend the placeholder/guidance to state the one-liner + cap rule.
- `src/kit/skills/fab-continue.md` § Hydrate Behavior Step 4 — the "curated `description:` one-liner, per `$(fab kit-path)/reference/fkf.md` §3.1–§3.2" bullet gains the length rule (the FKF-reference citation is how consuming-repo hydrate agents read the policy every run — which is why the shipped extract must carry it).
- `src/kit/skills/docs-hydrate-memory.md` — ingest Step 3/Step 4 (create/merge, "keep `description:` accurate"), generate mode, and backfill Step 2 (synthesize `description:`) each author descriptions; each gains the rule or the §3.2 citation.
- SPEC mirrors for both skills, per the sweep class.

**Advisory length warning in `fab memory-index`.** Alongside the malformed-frontmatter warnings (change 1) and the existing shape warnings: a description whose length exceeds the 500-character cap emits a stderr warning (new `Warning` kind, e.g. `description-length`, naming the file and the observed length). **Advisory only — it does NOT fail `--check`** (the deliberate asymmetry: corruption blocks, over-length nags), and it never changes rendered output. Threshold: a hardcoded package constant like `WidthWarnThreshold` (value: 500) — NOT config-overridable in this change (matches the existing shape-bound pattern; promoting it to config would touch the `internal/config` field registry + `docs/specs/config.md` and is not requested). Tests alongside per change 1's test plan; `_cli-fab.md` § fab memory-index documents the new warning line.

## Affected Memory

- `memory-docs/templates.md`: (modify) memory-file format — `description:` one-liner + length policy, template change, new memory-index warnings
- `memory-docs/hydrate.md`: (modify) hydrate/backfill description-authoring rules; regen sites' relationship to the new `--check` failure
- `pipeline/schemas.md`: (modify) `fab memory-index --check` exit-code/tier semantics gain the malformed-frontmatter blocking class
- `pipeline/execution-skills.md`: (modify) `/git-pr` + `/git-pr-review` generated-file merge policy; hydrate description policy wiring

## Impact

- **Go** (`src/go/fab/`): `internal/frontmatter` (validation primitives + tests), `internal/memoryindex` (new Warning kinds, gather-path validation, possible classifier/loss extension + tests incl. golden byte-stability), `cmd/fab/memory_index.go` (`--check` failure wiring, stderr output + tests). No schema/user-data restructuring ⇒ no migration file needed (validation is read-only; no `.status.yaml`/config change).
- **Kit skills** (`src/kit/skills/`): `_cli-fab.md`, `git-pr.md`, `git-pr-review.md`, `fab-continue.md`, `docs-hydrate-memory.md` — each with its `docs/specs/skills/SPEC-*.md` mirror.
- **Kit templates/reference** (`src/kit/`): `templates/memory.md`, `reference/fkf.md` (shipped normative extract — MUST move in lockstep with the spec).
- **Specs** (`docs/specs/`): `fkf.md` (§3.2 length policy, §5 merge policy + optional `.gitattributes` aside, §5/§6.4 `--check` semantics updates).
- **Memory** (`docs/memory/`): the four files above, then `fab memory-index` regen.
- **Consumers to keep compatible**: CI/staleness callers (exit ≥ 1 = fail), hydrate/reorg refuse-before-regen guards (exit == 2), `/docs-reorg-memory` `--check --json` consumer, review-pr `--check` backstop. Existing repos with long descriptions will start emitting advisory warnings (non-blocking — by design; fab-kit's own tree currently carries ~a dozen descriptions over the 500-char cap — deliberate dogfooding, a visible cleanup backlog); repos with committed malformed frontmatter will start FAILING `--check` (intended: that is the corruption this change exists to block).

## Open Questions

- **Resolved (2026-07-15, /fab-clarify)** — Exact `description:` length cap: **500 characters** (unit: characters, not words), measured on the description value (the single-line frontmatter scalar, after quote-stripping). Advisory-only — the length warning never fails `--check`, per Assumption #4. Grounding (measured 2026-07-15 across both real trees): loom `docs/memory` n=259, p50=118, p90=224, p99=1452, max=4307 chars (14 files > 500, 7 > 1000); fab-kit `docs/memory` n=23, p50=872, p90=3845, max=5859. 500 sits well clear of the healthy population (≤~250 chars) while catching every pathological description. Accepted consequence: fab-kit's own memory tree will emit ~a dozen advisory length warnings after this ships — deliberate dogfooding; the warnings are the visible, non-blocking cleanup backlog.

## Clarifications

### Session 2026-07-15 (auto — user-delegated decision)

| # | Action | Detail |
|---|--------|--------|
| 14 | Changed | "500-character cap on the `description:` value (characters, not words; measured on the quote-stripped single-line scalar); advisory-only length warning" |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Single fab change covering all three items (validation + merge policy + description policy) | User explicit in the originating conversation | S:95 R:70 A:90 D:95 |
| 2 | Certain | One-time loom data fix is OUT of scope | User explicit — handled separately via an operator message | S:95 R:90 A:95 D:95 |
| 3 | Certain | Byte-stability of rendered index output is inviolable; all validation is stderr/exit-code only, never changes rendered bytes | User-stated hard constraint; matches the package's documented advisory-warning discipline | S:90 R:75 A:95 D:90 |
| 4 | Certain | Asymmetry: malformed frontmatter FAILS `--check` (blocking); description over-length is advisory only (never fails `--check`) | Discussed recommendation adopted by the user — corruption blocks at review-pr, style nags don't | S:90 R:70 A:85 D:85 |
| 5 | Certain | Any FKF normative-rule edit updates BOTH `docs/specs/fkf.md` and the shipped extract `src/kit/reference/fkf.md` | Mandated verbatim by the fkf.md header ("MUST update both files so they cannot silently diverge") | S:85 R:70 A:95 D:95 |
| 6 | Certain | Go change ships tests in-change; `_cli-fab.md` updated for CLI behavior; full SPEC-mirror class swept up front | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps | S:90 R:70 A:100 D:95 |
| 7 | Certain | The malformed-frontmatter check must run independent of the byte-drift comparison (committed garbage row is byte-identical to its regeneration, so drift alone exits 0) | Mechanically verified against `frontmatter.Field`/`stripQuotes` and the pure-function index render | S:70 R:75 A:90 D:85 |
| 8 | Confident | Malformed detection = two signatures: (a) unclosed frontmatter fence; (b) `description:` value starting with a quote that fails quote-stripping (glued-fence, e.g. trailing `"---`) | Derived from the verified loom corruption + parser mechanics; (a) also catches the loom file, (b) is the specific diagnostic | S:80 R:70 A:85 D:70 |
| 9 | Tentative | Exit-code integration: new blocking category within the tiered `--check` scheme — front-runner tier-2-style with its own category + fix-the-file remediation message (guards then refuse to regen over corrupted input); alternative plain exit-1; apply decides, keeping CI (≥1), guards (==2), and `--json` consumers compatible | Multiple valid integrations with different consumer ripples; codebase gives partial signals only | S:45 R:50 A:50 D:40 |
| 10 | Confident | Advisory length warning = new `Warning` kind in `internal/memoryindex` beside width/depth; threshold a hardcoded package const (like `WidthWarnThreshold`), NOT config-overridable in this change | Existing shape-bound pattern is hardcoded consts; config promotion would drag in the config-registry surface un-asked | S:60 R:80 A:85 D:75 |
| 11 | Confident | Merge-policy placement: FKF §5 (+ shipped extract) as normative home; operational pointers in `git-pr.md` (owns the ship-time index-regen sub-step) and `git-pr-review.md` (PR-feedback/conflict seam); hydrate skills get at most a cross-reference | Verified by reading the sources — no conflict-handling prose exists anywhere today; these are the seams where agents meet merges | S:55 R:80 A:60 D:45 |
| 12 | Confident | `.gitattributes` merge-driver recipe is documentation only (non-normative aside in fkf.md §5), NOT auto-installed, no migration | User said "optionally document"; auto-install would be a user-data/tooling change out of scope | S:75 R:85 A:80 D:70 |
| 13 | Confident | Hydrate-time wiring points: `templates/memory.md` placeholder + `fab-continue.md` Hydrate Step 4 + `docs-hydrate-memory.md` ingest/generate/backfill authoring steps (+ SPEC mirrors) | Verified — these are the exact places description authoring is instructed today; the §3.2 citation chain makes consuming-repo hydrate agents read the policy every run | S:70 R:80 A:85 D:75 |
| 14 | Tentative | `description:` length cap = 500 characters (unit: characters, not words), measured on the quote-stripped single-line description value; advisory-only warning, never fails `--check` | Clarified — user changed to 500 chars (delegated decision, 2026-07-15; grounded in measured trees: loom n=259 p50=118 p90=224 max=4307, fab-kit n=23 p50=872 max=5859 — 500 clears the healthy ≤~250-char population and catches every pathological description; accepted: fab-kit's own tree emits ~a dozen advisory warnings, deliberate dogfooding) | S:95 R:70 A:15 D:25 |

14 assumptions (7 certain, 5 confident, 2 tentative, 0 unresolved).
