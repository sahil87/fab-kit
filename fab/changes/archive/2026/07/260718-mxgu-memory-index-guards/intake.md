# Intake: Memory-Index Guards for FKF Present-Truth Debt

**Change**: 260718-mxgu-memory-index-guards
**Created**: 2026-07-18

## Origin

Invoked via `/fab-new mxgu` (one-shot, backlog-driven). Backlog entry `[mxgu]` (fab/backlog.md, 2026-07-18), quoted verbatim:

> Change A: fab memory-index guards for FKF present-truth debt (cross-repo memory audit 2026-07-18, loom/run-kit/idea; staleness 0/18 spot-checks — the disease is accretion/form, not truth). (1) ESCALATE the description guard: change-ids in description: (mechanical regex; the §3.2 ban is currently "no enforcement is added") and gross over-cap (e.g. >2x the 500-char cap) become BLOCKING in `fab memory-index --check` — the advisory-only posture demonstrably failed (run-kit shipped a 16,519-char description and loom a 24,906-char one, 33x/50x cap, straight through the warning; bloat leaks verbatim into generated route tables — run-kit domain index costs ~7K tokens to route 7 files). Constraint: must NOT collide with the exit-2 destructive-loss tier that distill/hydrate refuse-before-regen guards key on — join the malformed-frontmatter blocking class instead. (2) NEW ADVISORY warnings: per-file narration-marker density (no longer/previously/renamed/supersed + change-id tokens in prose — a standing distillation-debt meter; idea 50 ids in prose, run-kit 1,066, loom 2,121 across 70% of files), per-file size soft cap (~400 lines / ~15KB; mega-files at every scale: idea structure.md 60KB = half corpus, run-kit ui-patterns.md 2,033 lines = 45%, loom 24 files >500 lines), and _unsorted/ non-empty (staging should trend to empty; loom holds 4 stale infra-505 session notes). (3) Broken memory-to-memory link detection: parse ](/...) targets, verify existence (loom: 1 broken of 686). Surfaces: Go memory_index + tests, `_cli-fab.md` § fab memory-index exit tiers, FKF §3.2/§4 posture update in BOTH docs/specs/fkf.md AND src/kit/reference/fkf.md (they must never diverge). PARALLEL with [wrct] (only overlap: both amend the two fkf.md files — merge seam, not ordering); [dsrx] consumes these signals.

No prior conversation context — the change is fully specified by the backlog entry, which encodes the decisions of the 2026-07-18 cross-repo memory audit (loom, run-kit, idea).

## Why

**The FKF §3.2 advisory-only posture demonstrably failed.** The 2026-07-18 cross-repo audit found run-kit shipped a 16,519-character `description:` and loom a 24,906-character one — 33× and 50× the 500-char cap — straight through the existing advisory warning (shipped in 260715-xu0k). The warning nags; nothing stops the bloat, and it leaks verbatim into the generated route tables: run-kit's domain index costs ~7K tokens to route 7 files. The `description:` field exists as a routing signal for the always-load context layer; an over-length or change-id-laden description degrades exactly the signal it exists to provide, on every single agent session in that repo.

**The disease is accretion/form, not truth** — staleness spot-checks scored 0/18 across the audited repos. Facts stay accurate; narration and bloat accumulate monotonically because no tool measures them. Without a standing meter, distillation debt (change-id tokens in prose: idea 50, run-kit 1,066, loom 2,121 across 70% of files; mega-files: idea `structure.md` 60KB = half its corpus, run-kit `ui-patterns.md` 2,033 lines = 45%, loom 24 files >500 lines) is invisible until a manual audit happens to run.

**If we don't fix it**: descriptions keep growing unbounded past the advisory nag, route tables keep bloating the always-load layer, `_unsorted/` staging keeps accumulating stale notes (loom: 4 infra-505 session notes parked since May), broken bundle-relative links go undetected (loom: 1 of 686), and the planned distill/reorg extensions (`[dsrx]`) have no canonical signal source to consume.

**Why this approach**: `fab memory-index` already walks every memory file on every run and already owns a two-class check taxonomy (advisory warnings + the blocking malformed-frontmatter class + the exit-2 destructive-loss tier). Escalating the two demonstrated-failure checks into the existing blocking class and adding the audit's debt meters as advisory warnings is the minimal mechanical extension — no new command, no new walk, no new exit-code scheme. Blocking joins the exit-1 floor (malformed-frontmatter class) precisely so the exit-2 destructive-loss tier that the hydrate/reorg refuse-before-regen guards key on stays untouched.

## What Changes

All checks below live in `src/go/fab/internal/memoryindex` + `src/go/fab/cmd/fab/memory_index.go`, extending the existing warning machinery (`Warning` kinds, `frontmatterWarnings` pass, `LossReport`). None of them changes the rendered index/log bytes — the byte-stability contract ("warnings never affect output") holds for every new check, on both the write and `--check` paths.

### 1. Escalate two description guards to BLOCKING (join the malformed-frontmatter class)

Two new blocking findings on `description:` values (validated on topic files AND domain/sub-domain `index.md` stubs, same scope as the existing malformed checks):

- **Change-id in `description:`** — the FKF §3.2 ban ("No enforcement is added" today) becomes enforced. Detection is **registry-gated** (mirroring `attributeCommit`'s false-positive-free design): a token counts as a change-id only when it resolves against the `fab/changes/*` + `fab/changes/archive/**` registry — either a full `YYMMDD-XXXX-slug` folder-name token or a bare 4-char registered id appearing in the banned §3.2 shapes (a `(d9rs)`-style parenthesized citation or a `— xu0k`-style suffix, plus a bare registered-id token). A description mentioning "code" or "yaml" never false-positives because those don't resolve in the registry.
- **Gross over-cap** — a `description:` strictly longer than **2× the cap (> 1000 runes)** becomes blocking. The 500-char advisory warning is unchanged for the 501–1000 range (trim nag); past 1000 the check fails. Measured identically to today: runes on the quote-stripped value.

**Blocking mechanics** (the hard constraint from the backlog): both join the **malformed-frontmatter blocking class** — they **floor the `--check` exit at 1** independent of index drift, enumerate the offending file(s) to stderr with a fix-the-source pointer, and ride the additive `malformed` JSON array with new `kind` values. They are **NOT tier-2 categories**: exit 2 stays reserved for the three index-only destructive-loss detectors, so the `/docs-hydrate-memory` / `/docs-reorg-memory` refuse-before-regen guards (which key on exit == 2) are unaffected. Tier-2 still wins the exit code when co-occurring; blocking findings are enumerated either way. The internal `malformedKinds` set / `IsMalformed()` generalizes to a blocking set (naming at apply's discretion) — the `--json` key stays `malformed` for consumer compatibility (`/docs-reorg-memory` branches on `tier`/`losses`, which are unchanged).

Stderr lines use the existing `✖` blocking glyph, e.g.:

```
✖ docs/memory/runtime/dispatch.md `description:` carries a change-id (registry match: xu0k) — descriptions are routing signals; move citations to the body (FKF §3.2)
✖ docs/memory/ui/ui-patterns.md has a 16519-character `description:` (blocking cap: 1000, soft cap: 500) — trim to a one-liner; detail belongs in the file body
```

### 2. New ADVISORY warnings (never affect the exit code)

Three new `⚠` advisory kinds, joining width/depth/description-length (stderr on both write and `--check` paths):

- **Narration-marker density (per topic file)** — the standing distillation-debt meter. Count, over the file body: case-insensitive substring hits of the transition stems `no longer`, `previously`, `renamed`, `supersed` (covers supersede/superseded/supersedes), plus registry-gated change-id tokens in prose. Warn when a file's total marker count reaches the threshold (proposed: **≥ 5**), reporting the count, e.g. `⚠ docs/memory/pipeline/schemas.md has 12 narration markers (threshold: 5) — distillation debt; consider /docs-distill-memory`. Note: sanctioned `(change-id)` citations do count toward the meter — it measures accumulated provenance density, not just violations, which is what makes it a debt meter; it is advisory precisely because citations are legitimate.
- **Per-file size soft cap** — warn when a topic file exceeds **~400 lines OR ~15KB** (either bound), e.g. `⚠ docs/memory/ui/ui-patterns.md is 2033 lines / 62KB (soft cap: ~400 lines / ~15KB) — consider splitting; see /docs-reorg-memory`. Mega-files are split candidates for `[dsrx]`'s reorg extension.
- **`_unsorted/` non-empty** — warn when `docs/memory/_unsorted/` holds ≥ 1 topic file (staging should trend to empty), e.g. `⚠ docs/memory/_unsorted holds 4 staged file(s) — triage into domains (staging should trend to empty)`. `_unsorted` stays width-exempt; this is a presence signal, not a shape bound.

### 3. Broken memory-to-memory link detection (advisory)

Parse bundle-relative link targets (`](/...)`) in topic-file bodies, resolve each against `docs/memory/` on disk, and warn per missing target, e.g. `⚠ docs/memory/pipeline/schemas.md links to /runtime/dispach.md — target does not exist`. **Advisory, not blocking**: FKF §7 says consumers MUST tolerate broken links (a missing target is not malformed) — this is the author-side nag that finds them, matching that posture. Only bundle-relative (`/`-prefixed) targets are checked; repo-relative and external links are out of scope (no false positives on links out of the bundle).

### 4. `--json` surface

The `--check --json` report gains the new blocking kinds in the existing additive `malformed` array, and adds an **additive `warnings` array** (`[{kind, path, count, detail}]`, empty-never-null like `losses`/`malformed`) carrying the advisory findings — the machine surface `[dsrx]`'s survey/reorg extensions consume instead of parsing stderr. `tier`/`drift`/`losses` are unchanged.

### 5. Documentation surfaces

- **`src/kit/skills/_cli-fab.md` § fab memory-index** — new warning lines, the widened blocking class, the 2×-cap escalation, the `warnings` JSON array, updated exit-code text (constitution: CLI changes MUST update `_cli-fab.md`; `_cli-fab` has no SPEC mirror by policy).
- **FKF posture update in BOTH `docs/specs/fkf.md` AND `src/kit/reference/fkf.md`** (they must never diverge — the fkf.md single-sourcing rule): §3.2 drops "No enforcement is added" (the change-id ban is now enforced blocking; the length cap is advisory to 1000, blocking past it); §4 (spec §4 / extract's shape-bounds text) documents the new advisory kinds while keeping the shape bounds themselves advisory-never-enforced. **Merge seam with `[wrct]`**: both changes amend these two files — parallel work, semantic merge at integration, no ordering dependency.
- Go cobra `Long`/flag help text on `fab memory-index` updated to match.

### Non-changes (scope fences)

- No config surface: all thresholds are hardcoded package consts (the `DescriptionLenWarnThreshold` precedent — explicitly not config-overridable).
- No `.status.yaml` schema change → no migration file needed.
- No exit-code renumbering; exit 2 semantics untouched.
- Consuming the new signals (survey heuristic switch, reorg split candidates, `_unsorted` triage) is `[dsrx]`'s scope, not this change's.

## Affected Memory

- `memory-docs/templates.md`: (modify) the Memory Tree Shape section homes the warning/blocking taxonomy — add the two blocking escalations, three advisory kinds, and broken-link detection
- `memory-docs/hydrate.md`: (modify) the refuse-before-regen guard note distinguishes exit-2 from the malformed blocking class — update to name the widened blocking class (change-id + gross over-cap join it)

## Impact

- **Go**: `src/go/fab/internal/memoryindex/memoryindex.go` (+ `memoryindex_test.go`) — new Warning kinds, registry-gated change-id scan (registry already gathered for log.md; the description pass gains access to it), body scans for narration/size/links, `_unsorted` presence check; `src/go/fab/cmd/fab/memory_index.go` (+ test) — blocking-class widening in `emitCheckReport`, `warnings` JSON array, help text. Possibly `internal/frontmatter` if the description checks need a new accessor (likely not — `frontmatter.Field` suffices).
- **Kit skills**: `src/kit/skills/_cli-fab.md` (no SPEC mirror — excluded by policy).
- **Specs**: `docs/specs/fkf.md` + `src/kit/reference/fkf.md` (both, in lockstep).
- **Performance**: the new passes add per-file body reads to a walk that already reads every file's frontmatter/H1 — negligible.
- **Consumers**: CI/pre-commit callers (fail on ≥1) now also fail on the two escalated checks — intended; hydrate/reorg guards (exit == 2) unaffected by construction. This repo's own tree is clean against the new blocking checks (born-FKF, distilled 2026-07-18), so `--check` at review-pr stays green.
- **Parallel**: `[wrct]` (shared fkf.md merge seam only); `[dsrx]` consumes the new warnings downstream.

## Open Questions

None — the backlog entry (audit-derived) resolves the material decisions; residual parameter choices are graded below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Blocking escalations join the exit-1 malformed-frontmatter floor class, never the exit-2 destructive-loss tier | Backlog states this as a hard constraint; code confirms hydrate/reorg refuse-before-regen guards key on exit == 2 | S:95 R:80 A:95 D:95 |
| 2 | Confident | Gross over-cap blocking threshold = strictly > 1000 runes (2× the 500 cap); 501–1000 stays advisory | Backlog gives ">2x the 500-char cap" as an example ("e.g."); 2× is the stated anchor and a hardcoded const is trivially tuned later | S:70 R:85 A:80 D:75 |
| 3 | Confident | Change-id detection is registry-gated (token must resolve in the fab/changes registry), matching the banned §3.2 shapes | Blocking checks need false-positive-freedom; `attributeCommit` establishes the registry-gating precedent in the same package, and the audited repos' violations resolve against their own registries | S:60 R:75 A:85 D:70 |
| 4 | Certain | Broken-link detection is advisory, never blocking | FKF §7 explicitly: consumers MUST tolerate broken links — a missing target is not malformed; author-side detection nags, matching posture | S:65 R:85 A:90 D:85 |
| 5 | Confident | Narration-density warning fires at ≥ 5 markers per file, counting the four transition stems + registry-gated change-id tokens (sanctioned citations included) | Backlog names the markers but no threshold; advisory + hardcoded const = cheap to tune; the meter deliberately counts citations (density is the signal, violations are §3.2's job) | S:40 R:85 A:50 D:35 |
| 6 | Confident | Size warning fires when either bound is exceeded (> 400 lines OR > 15KB), hardcoded consts | Backlog gives "~400 lines / ~15KB"; OR-semantics catches both the line-heavy and byte-heavy mega-file shapes seen in the audit | S:70 R:85 A:75 D:70 |
| 7 | Certain | `_unsorted/` warning fires on ≥ 1 topic file present; `_unsorted` keeps its width exemption | Backlog: "staging should trend to empty" — presence is the signal; the width exemption governs a different bound | S:80 R:85 A:90 D:85 |
| 8 | Confident | `--json` gains an additive `warnings` array for advisory findings; blocking kinds ride the existing `malformed` array; `tier`/`drift`/`losses` unchanged | `[dsrx]` consumes these signals — JSON is the machine surface (stderr parsing is brittle); additive arrays follow the 260715-xu0k `malformed` precedent and keep `/docs-reorg-memory`'s tier/losses branching intact | S:55 R:80 A:70 D:60 |
| 9 | Certain | All thresholds are hardcoded package consts, not config-overridable | Direct precedent: `DescriptionLenWarnThreshold` is documented in-code as "hardcoded (the shape-bound-const pattern) — NOT config-overridable" | S:75 R:85 A:95 D:90 |
| 10 | Certain | New per-file body checks (narration, size, links) scan topic files only (index.md / log.md / log.seed.md excluded); the description blocking checks also cover index.md stubs, matching the existing malformed-check scope | Mirrors the established `frontmatterWarnings` / `gatherFiles` skip sets; generated files are not concept documents | S:65 R:85 A:85 D:80 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved).
