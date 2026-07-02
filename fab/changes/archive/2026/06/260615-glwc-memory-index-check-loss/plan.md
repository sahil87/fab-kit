# Plan: `fab memory-index --check` severity exit-code tiers + mechanical destructive-loss detection

**Change**: 260615-glwc-memory-index-check-loss
**Intake**: [intake.md](intake.md)

---

## Requirements

### Go: `fab memory-index --check` severity tiers

- **R1**: `fab memory-index --check` MUST classify the rendered-vs-existing drift it already computes into three severity tiers encoded in the process exit code.
  - GIVEN a memory tree, WHEN `--check` runs, THEN it MUST exit `0` if every index file is byte-identical to its regenerated form (clean), `1` if drift exists but is benign (regen changes content without destroying curated/historical content), and `2` if regen would destructively wipe curated/historical content.
- **R2**: Loss MUST be a strict subset of drift — exit 2 implies drift exists, and a tree with no drift is always exit 0.
  - GIVEN a clean tree, WHEN `--check` runs, THEN exit code is 0 and no loss is reported.
- **R3** (Curated description → `—`): A destructive loss MUST be reported when an existing index row renders a non-empty, non-`—` description but the regenerated row would render `—` (the file/domain lacks `description:` frontmatter).
  - GIVEN a domain index row `| [login](login.md) | Login flow | ... |` and `login.md` has no `description:` frontmatter, WHEN `--check` runs, THEN tier is 2 and the loss names the file and the lost description text.
- **R4** (Tombstone row dropped): A destructive loss MUST be reported when an existing index row's `docs/memory/`-relative link target is absent on disk (the generator lists only on-disk folders/files, so it drops the row).
  - GIVEN a root index row `| [lib-bdash](lib-bdash/index.md) | ... |` and `docs/memory/lib-bdash/` does not exist, WHEN `--check` runs, THEN tier is 2 and the loss names the dropped row's link target. AND GIVEN an external/absolute link target (`https://...`, `/abs/path`), THEN it MUST NOT be reported as a tombstone (no false positive).
- **R5** (Custom grouping flattened): A destructive loss MUST be reported when the existing root `docs/memory/index.md` contains structural markdown headings (`## `/`### ` beyond the generated boilerplate) that the domains-only `RenderRoot` output omits.
  - GIVEN a root index containing `### Apps` / `### Packages` sections, WHEN `--check` runs, THEN tier is 2 and the loss names the flattened heading(s).
- **R6** (benign drift → exit 1): Drift that destroys nothing MUST classify as tier 1.
  - GIVEN an index whose only difference from regenerated output is an improved description or a refreshed `Last Updated` date (no description→`—`, no dropped row, no flattened grouping), WHEN `--check` runs, THEN exit code is 1 and no destructive loss is reported.
- **R7** (`--json`): `fab memory-index --check --json` MUST emit the loss/drift report as machine-readable JSON on stdout, mirroring the `fab pane`/`migrations-status` `--json` convention (snake_case fields).
  - GIVEN any tree, WHEN `--check --json` runs, THEN stdout is a single JSON object carrying the tier and the per-category loss enumeration; human-readable text is suppressed on stdout.
- **R8** (exit-2 pointer): On tier 2, the human-readable stderr output MUST end with a pointer to `/docs-reorg-memory` — the remediation orchestrator for all three tier-2 categories (it relocates removal-history/tombstone rows itself and dispatches `/docs-hydrate-memory` backfill mode for descriptions; backfill alone does not relocate tombstones).
  - GIVEN a tier-2 tree (no `--json`), WHEN `--check` runs, THEN stderr ends with `→ run /docs-reorg-memory to remediate (it relocates removal-history rows to _shared/removed-domains.md and backfills description: frontmatter via /docs-hydrate-memory) before regenerating.`
- **R9** (backward compatibility / born-compatible no-op): Bare `memory-index` (no `--check`) behavior MUST be unchanged, and existing `--check` consumers treating "non-zero = out of date" MUST keep working (any drift is still non-zero). A born-compatible fab-kit tree MUST never be tier 2.
  - GIVEN a born-compatible tree (frontmatter present, no tombstones, domains-only root index), WHEN `--check` runs, THEN exit code is 0 or 1, never 2.
- **R10** (pure, unit-testable classifier): The loss classification and the existing-index-row parsing MUST live as pure functions in `internal/memoryindex` (like `RenderRoot`/`Gather`), surfaced via the cmd. The cmd MUST reuse the existing rendered-vs-existing comparison rather than duplicating it.

### Skills: rewire reorg detection + refuse-before-regen guards

- **R11** (reorg rewire): `/docs-reorg-memory`'s prose-based compatibility detection MUST be replaced by invoking `fab memory-index --check --json` and consuming its structured loss output (tier 2 = surface findings; tier 0/1 = nothing to relocate/backfill). The findings-report (Step 3) and on-approval orchestration (Step 5) behavior MUST be unchanged — they now consume the primitive's output. The older-binary fallback MUST retain the prose detection as a legacy path.
  - GIVEN a pre-fab-kit tree and a `--check`-loss-capable binary, WHEN `/docs-reorg-memory` runs, THEN it derives compatibility findings from `fab memory-index --check --json` (tier 2). GIVEN an older binary, THEN it falls back to the prose detection.
- **R12** (regen-site guards): Every site that runs `fab memory-index` to regenerate MUST first consult `fab memory-index --check` and refuse to regenerate on exit 2, surfacing the `/docs-reorg-memory` remediation pointer (the orchestrator that relocates tombstone rows and dispatches `/docs-hydrate-memory` backfill for descriptions). Sites: `/docs-hydrate-memory` regen step(s), `/docs-reorg-memory` (via R11), `/fab-continue` hydrate stage. Each guard MUST carry a brief annotation that it is a no-op for born-compatible trees.
  - GIVEN a born-compatible tree, WHEN any regen site runs, THEN the guard is a no-op (exit 0/1) and regeneration proceeds. GIVEN a pre-fab-kit tree, THEN the guard refuses (exit 2) and points to `/docs-reorg-memory` for remediation.

### Docs: SPEC mirrors + CLI reference

- **R13** (SPEC mirrors): Each changed `src/kit/skills/*.md` MUST update its `docs/specs/skills/SPEC-*.md` mirror — `SPEC-docs-reorg-memory.md` (rewire), `SPEC-docs-hydrate-memory.md` (guard), `SPEC-fab-continue.md` (pipeline hydrate guard).
- **R14** (CLI doc): `src/kit/skills/_cli-fab.md` MUST document the tiered `--check` exit codes (0/1/2) and the new `--json` flag on the `fab memory-index` entry (constitution CLI-doc rule; `_cli-fab.md` has no SPEC mirror per uliv policy).

### Non-Goals

- No separate `--check-loss` flag (resolved: single `--check` flag, severity in exit codes).
- No relocation/backfill performed by the primitive itself — it only *detects and classifies*; remediation stays in the skills.
- No change to bare `memory-index` write behavior or to the warnings/idempotency contract.

### Design Decisions

- **Exit-2 via in-handler `os.Exit(2)`**: `main()` exits 1 on any RunE error, so a tiered exit 2 cannot ride a returned error. Mirror the established `pane_capture.go`/`pane_send.go` pattern — print the human report then `os.Exit(2)` in the handler. Tier 1 rides a returned error (cobra → exit 1); tier 0 returns nil. The *classifier* is a pure function unit-tested independently of the exit mechanism (so the os.Exit branch needs no process-spawn test).
- **Existing-index-row parser scope** (the intake's deferred inline SRAD assumption): bound the parser to exactly the three categories — (a) markdown table rows of the form `| [text](target) | desc | ... |` (capture link text, link target, first description cell), and (b) `## `/`### ` ATX headings in the root index. Tolerate the generated table shapes (root `| Domain | Description |`, domain `| File | Description | Last Updated |`) and leading/trailing pipes; ignore separator rows (`|---|`) and the generated boilerplate headings (`# Memory Index`, `## Sub-Domains`). Do not attempt to parse arbitrary hand-authored markdown beyond these.
- **JSON shape**: `{ "tier": 0|1|2, "drift": bool, "losses": [ { "category": "description|tombstone|grouping", "path": "<repo-rel index>", "detail": "<row/heading>" } ] }` — snake_case, mirrors the `--json` convention.

---

## Tasks

### Phase 1: Core loss-classification primitive (internal/memoryindex)

- [x] T001 Add the existing-index-row parser to `internal/memoryindex` — pure functions `parseIndexRows(content string) []indexRow` (link text, link target, first description cell) and `parseRootHeadings(content string) []string` (ATX `##`/`###` headings excluding generated boilerplate). <!-- R10 R3 R4 R5 -->
- [x] T002 Add the three pure loss detectors over (existing content, regenerated content, on-disk lookups) in `internal/memoryindex`: description→`—`, tombstone (relative-target-absent, external/absolute excluded), grouping-flatten. <!-- R3 R4 R5 -->
- [x] T003 Add `Loss` + `LossReport` types and a `Classify(...)` entry point that, given the per-target (path, existing, rendered) plus a memRoot for on-disk tombstone checks, returns the report with `Tier` (0/1/2), `Drift` bool, and `Losses []Loss`. <!-- R1 R2 R6 -->

### Phase 2: Wire the classifier into the cmd

- [x] T004 Extend `cmd/fab/memory_index.go`: add `--json` flag; in the `--check` branch, build the report via the classifier (reusing the existing rendered-vs-existing comparison + `memRoot`), then dispatch on tier — 0 → nil; 1 → return drift error (exit 1); 2 → print per-category losses + the backfill pointer to stderr and `os.Exit(2)`. With `--json`, emit the report JSON to stdout and use the same exit dispatch (no human text on stdout). <!-- R1 R6 R7 R8 R9 -->
- [x] T005 Update the cmd `Long`/flag help to describe the tiered exit codes and `--json`. <!-- R14 -->

### Phase 3: Tests (test-alongside)

- [x] T006 `internal/memoryindex` unit tests: each loss category → tier 2; benign drift → tier 1; no change → tier 0; external/absolute link never a tombstone; parser round-trips the generated row/heading shapes. <!-- R3 R4 R5 R6 R2 -->
- [x] T007 cmd-level tests in `memory_index_test.go`: `--check` clean → nil (exit 0); benign drift → error (exit 1); `--json` clean emits a parseable `{tier:0}` object; `--json` flag registered. (Tier-2 os.Exit branch is covered by the pure classifier test, not a process spawn.) <!-- R1 R7 R9 -->

### Phase 4: Skills + SPEC mirrors + CLI doc

- [x] T008 Rewire `src/kit/skills/docs-reorg-memory.md`: replace the prose compatibility-detection bullets (Step 1) with invoking `fab memory-index --check --json` and consuming tier-2 output; keep findings-report + orchestration behavior; extend the older-binary fallback to the prose legacy path. <!-- R11 -->
- [x] T009 Add the refuse-before-regen guard (consult `fab memory-index --check`, refuse on exit 2, point to backfill) to `src/kit/skills/docs-hydrate-memory.md` regen step(s) and `src/kit/skills/fab-continue.md` hydrate step — each with a born-compatible-no-op annotation. <!-- R12 -->
- [x] T010 Update `src/kit/skills/_cli-fab.md` `fab memory-index` entry: tiered `--check` exit codes (0/1/2) + `--json` flag. <!-- R14 -->
- [x] T011 Update SPEC mirrors: `SPEC-docs-reorg-memory.md` (rewire), `SPEC-docs-hydrate-memory.md` (guard), `SPEC-fab-continue.md` (pipeline hydrate guard). <!-- R13 -->

### Phase 5: Verify

- [x] T012 Run `cd src/go/fab && go test ./...` (or `just test`); all green. <!-- R1-R10 -->

## Execution Order

T001 → T002 → T003 (classifier built bottom-up) → T004 → T005 (cmd wiring) → T006, T007 [P] (tests) → T008, T009, T010, T011 [P] (docs/skills) → T012 (verify).

---

## Acceptance

### Functional Completeness

- [ ] A-001 R1: `fab memory-index --check` exits 0 (clean) / 1 (benign drift) / 2 (destructive loss) per the tier classification.
- [ ] A-002 R2: Loss is a strict subset of drift — a no-drift tree is always exit 0 with no loss reported.
- [ ] A-003 R3: A curated description that would become `—` on regen is detected and reported as a tier-2 loss naming the file and lost text.
- [ ] A-004 R4: A row whose `docs/memory/`-relative link target is absent on disk is reported as a tier-2 tombstone; external/absolute targets are never reported.
- [ ] A-005 R5: Structural headings in the root index beyond the generated table are reported as a tier-2 grouping-flatten loss.
- [ ] A-006 R6: Benign drift (improved description / refreshed date only) classifies as tier 1 with no destructive loss.
- [ ] A-007 R7: `--check --json` emits a single snake_case JSON object with `tier`, `drift`, and per-category `losses` on stdout.
- [ ] A-008 R8: Tier-2 human-readable stderr ends with the `/docs-hydrate-memory (backfill mode)` pointer.
- [ ] A-009 R9: Bare `memory-index` is unchanged; a born-compatible tree is never tier 2; existing non-zero=stale consumers still work.
- [ ] A-010 R10: Loss classification + index-row parsing are pure functions in `internal/memoryindex`; the cmd reuses (does not duplicate) the rendered-vs-existing comparison.
- [ ] A-011 R11: `/docs-reorg-memory` detection calls `fab memory-index --check --json`; findings-report + orchestration behavior unchanged; older-binary prose fallback retained.
- [ ] A-012 R12: Every regen site guards on `fab memory-index --check` exit 2 and refuses with the backfill pointer; each guard is annotated as a born-compatible no-op.
- [ ] A-013 R13: `SPEC-docs-reorg-memory.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-fab-continue.md` mirror the rewire/guards.
- [ ] A-014 R14: `_cli-fab.md` documents the tiered exit codes and `--json` on the `fab memory-index` entry.

### Behavioral Correctness

- [ ] A-015 R6 R2: The classifier never reports tier 2 without at least one concrete loss in `Losses`, and never tier 1 without drift.

### Edge Cases & Error Handling

- [ ] A-016 R4: A tombstone link target that is external (`http(s)://`) or absolute (`/...`) is excluded from tombstone detection (no false positive).
- [ ] A-017 R7: `--json` suppresses the human-readable stderr/stdout text (machine-readable only on stdout).

### Code Quality

- [ ] A-018 Pattern consistency: new Go matches the package's pure-render + Gather-I/O style and the cmd's existing `--check` branch; no duplicated comparison logic.
- [ ] A-019 No unnecessary duplication: the existing-index-row parser is bounded to the three loss categories (not a general markdown parser).

### Documentation Accuracy (checklist.extra_categories)

- [ ] A-020 The `_cli-fab.md` entry and the three SPEC mirrors accurately describe the shipped exit-code tiers, `--json` shape, and guard semantics.

### Cross-References (checklist.extra_categories)

- [ ] A-021 Skill guard annotations and the reorg rewire cross-reference the primitive consistently (no stale prose re-stating Go semantics in reorg's detection path).

---

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | This is a `feat` touching Go: new classifier + `--json` flag + tiered exit codes in `cmd/fab/memory_index.go` and `internal/memoryindex`, with tests | Carried from intake #1; new capability over the existing `--check`. | S:95 R:80 A:100 D:95 |
| 2 | Certain | Worktree is on 2.4.0 (== installed binary); Go edits target the running source | Carried from intake #2; verified the `--check` branch + `Gather`/`RenderRoot` in 2.4.0 source. | S:100 R:85 A:100 D:100 |
| 3 | Certain | Single `--check` flag, severity in exit codes (0 clean / 1 benign drift / 2 destructive loss); no separate `--check-loss` | Carried from intake #11 (resolved clarification). | S:95 R:55 A:60 D:90 |
| 4 | Certain | Three loss categories = mechanical form of 5ewp's prose signals (description→`—`, tombstone drop, grouping flatten), verified against the generator logic | Carried from intake #7; `RenderRoot`/`Gather`/`RenderDomain` confirm what regen drops. | S:95 R:80 A:95 D:80 |
| 5 | Certain | Born-compatible trees are provably never tier 2 (frontmatter present, no off-disk rows, domains-only root) | Carried from intake #9; factual property of the three categories. | S:95 R:85 A:95 D:85 |
| 6 | Confident | Tier-2 surfaced via in-handler `os.Exit(2)` (the established `pane_capture`/`pane_send` non-1-code pattern); tier 1 via returned error (cobra→exit 1); tier 0 via nil. The pure classifier is unit-tested; the os.Exit branch is not process-spawn-tested | `main()` always exits 1 on RunE error, so a returned error cannot encode exit 2. The codebase already uses in-handler os.Exit for genuine non-1 codes. Keeping the decision pure keeps it testable. | S:80 R:65 A:90 D:80 |
| 7 | Confident | Existing-index-row parser scope bounded to: markdown table rows `\| [text](target) \| desc \| ... \|` (text, target, first desc cell) + root `##`/`###` ATX headings, excluding generated boilerplate/separators | The intake's deferred inline SRAD assumption (Open Questions). The three categories need exactly these; a general markdown parser would be over-engineering (code-quality anti-pattern). | S:85 R:70 A:85 D:75 |
| 8 | Confident | `--json` shape: `{tier, drift, losses:[{category, path, detail}]}` (snake_case), mirroring `fab pane`/`migrations-status` `--json`; emitted to stdout, suppressing human text | No exact `migrations-status` loss-report precedent; modeled on the repo's snake_case `--json` convention. Reorg consumes `tier` + `losses`. | S:80 R:70 A:80 D:80 |
| 9 | Confident | Tombstone primary signal = unresolved `docs/memory/`-relative link target on disk; strikethrough is a non-required corroborating hint; external/absolute links excluded | Carried from intake #2 category + 5ewp assumption #10. The on-disk check is the deterministic signal; strikethrough parsing adds complexity for no additional precision, so detector keys on disk-resolution only. | S:90 R:75 A:85 D:80 |
| 10 | Confident | Guard sites add a one-line `fab memory-index --check` consult refusing on exit 2; reorg consumes `--json` directly (not a second guard); each annotated born-compatible no-op | Carried from intake #12 (resolved). Defense-in-depth; logic stays in Go. | S:95 R:60 A:60 D:90 |

10 assumptions (5 certain, 5 confident, 0 tentative).
</content>
</invoke>
