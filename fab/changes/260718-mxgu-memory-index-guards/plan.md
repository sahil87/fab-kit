# Plan: Memory-Index Guards for FKF Present-Truth Debt

**Change**: 260718-mxgu-memory-index-guards
**Intake**: `intake.md`

## Requirements

### memory-index: Blocking description guards

#### R1: Registry-gated change-id in `description:` is BLOCKING
`fab memory-index` MUST treat a `description:` frontmatter value that carries a change-id token as a **blocking** finding — joining the existing malformed-frontmatter blocking class (floors the `--check` exit at 1 independent of index drift, NEVER the exit-2 destructive-loss tier). A token counts as a change-id only when it **resolves against the `fab/changes/*` + `fab/changes/archive/**` registry** (mirroring `attributeCommit`'s false-positive-free gating): either a full `YYMMDD-XXXX-slug` folder-name token whose registered folder matches, or a bare 4-char token that is a registered id. The scan MUST cover topic files AND domain/sub-domain `index.md` stubs (the same scope as the existing malformed checks). Validation is stderr/exit-code only — it MUST NOT change the rendered index bytes.

- **GIVEN** a topic file whose `description:` is `"Dispatch runtime — see (xu0k)"` and `xu0k` is a registered change id
- **WHEN** `fab memory-index` runs (write or `--check`)
- **THEN** a `✖` blocking finding is emitted naming the file and the matched id, and `--check` exit is floored at 1 (never 2)
- **AND** the rendered index rows are byte-identical to a run over the same tree

- **GIVEN** a `description:` mentioning the word `code` or `yaml` (not registered ids)
- **WHEN** `fab memory-index` runs
- **THEN** no change-id blocking finding fires (registry gating gives false-positive-freedom)

#### R2: Gross over-cap `description:` (> 1000 runes) is BLOCKING
`fab memory-index` MUST treat a `description:` value **strictly longer than 1000 runes** (2× the 500 cap, measured in runes on the quote-stripped value) as a **blocking** finding joining the malformed-frontmatter class. The existing 501–1000 range MUST keep emitting today's advisory over-length warning (unchanged). Both bounds are hardcoded package consts, not config.

- **GIVEN** a `description:` of 1001 runes
- **WHEN** `fab memory-index --check` runs
- **THEN** a `✖` blocking finding fires and the exit is floored at 1 (never 2)

- **GIVEN** a `description:` of 600 runes
- **WHEN** `fab memory-index --check` runs
- **THEN** only the existing advisory `⚠` length warning fires and `--check` still exits 0 (over-cap boundary is strictly > 1000)

#### R3: Blocking mechanics preserve the exit-2 tier and the `malformed` JSON contract
Both new blocking findings MUST ride the existing additive `malformed` JSON array with new `kind` values, MUST floor the `--check` exit at 1, and MUST NOT enter the exit-2 destructive-loss tier. The `tier`/`drift`/`losses` keys MUST be unchanged. Exit 2 MUST still win when a tier-2 loss co-occurs (blocking findings enumerated either way), so the hydrate/reorg refuse-before-regen guards (keyed on exit == 2) stay unaffected.

- **GIVEN** a tree with a change-id-in-description finding but no index drift (tier 0)
- **WHEN** `fab memory-index --check` runs
- **THEN** exit is 1 and the `--json` `malformed` array carries the finding with `tier: 0`

- **GIVEN** a tree with both a tier-2 destructive loss and a blocking description finding
- **WHEN** `fab memory-index --check` runs
- **THEN** exit is 2 and the malformed finding is still enumerated

### memory-index: Advisory density/size/staging warnings

#### R4: Per-topic-file narration-marker density warning (advisory)
`fab memory-index` MUST emit an advisory `⚠` warning for a topic file whose narration-marker count reaches **5**, reporting the count. A marker is a case-insensitive substring hit of the stems `no longer`, `previously`, `renamed`, `supersed` OR a registry-gated change-id token in the file body (same gating as R1). The warning MUST NOT affect the exit code. It applies to topic files only (index.md / log.md / log.seed.md excluded).

- **GIVEN** a topic file body containing 3 stem hits and 2 registry-gated change-id tokens
- **WHEN** `fab memory-index` runs
- **THEN** an advisory `⚠` narration warning fires reporting count 5, and the exit code is unaffected

#### R5: Per-topic-file size warning (advisory)
`fab memory-index` MUST emit an advisory `⚠` warning for a topic file exceeding **400 lines OR 15360 bytes (15KB)** (either bound). Advisory only; topic files only.

- **GIVEN** a 401-line topic file
- **WHEN** `fab memory-index` runs
- **THEN** an advisory size warning fires reporting the line/byte counts, exit code unaffected

- **GIVEN** a 300-line, 20000-byte topic file
- **WHEN** `fab memory-index` runs
- **THEN** the size warning still fires (byte bound exceeded)

#### R6: `_unsorted/` non-empty warning (advisory)
`fab memory-index` MUST emit an advisory `⚠` warning when `docs/memory/_unsorted/` holds ≥ 1 topic file, reporting the count. `_unsorted` keeps its width exemption (this is a presence signal, not a shape bound). Advisory only.

- **GIVEN** `docs/memory/_unsorted/` holds 4 topic files
- **WHEN** `fab memory-index` runs
- **THEN** one advisory warning fires naming `docs/memory/_unsorted` and the count 4, exit code unaffected

### memory-index: Broken memory-to-memory link detection

#### R7: Broken bundle-relative link warning (advisory)
`fab memory-index` MUST parse bundle-relative `](/...)` link targets in topic-file bodies and emit an advisory `⚠` warning per target that does not resolve on disk under `docs/memory/`. Only `/`-prefixed targets are in scope; repo-relative and external links are out of scope. Targets inside fenced code blocks MUST be skipped (they are documentation examples, not live links — FKF §7 says consumers tolerate broken links; this is the author-side nag). Advisory, never blocking.

- **GIVEN** a topic-file body linking to `/runtime/dispach.md` (typo, no such file)
- **WHEN** `fab memory-index` runs
- **THEN** an advisory `⚠` broken-link warning fires naming the source file and target, exit code unaffected

- **GIVEN** a fenced code block containing an example link `/{domain}/base.md`
- **WHEN** `fab memory-index` runs
- **THEN** no broken-link warning fires for it (code fences are skipped)

### memory-index: `--check --json` warnings surface

#### R8: Additive `warnings` array in `--check --json`
The `--check --json` report MUST gain an additive `warnings` array carrying the advisory findings (`{kind, path, count, detail}`-shaped, empty-never-null like `losses`/`malformed`). Blocking kinds MUST continue to ride the `malformed` array. `tier`/`drift`/`losses` MUST be unchanged.

- **GIVEN** a tree with a size warning and a narration warning
- **WHEN** `fab memory-index --check --json` runs
- **THEN** stdout carries a `warnings` array with both findings and the `losses`/`malformed`/`tier`/`drift` keys are unchanged

### memory-index: Documentation

#### R9: `_cli-fab.md` documents the new guards
`src/kit/skills/_cli-fab.md` § fab memory-index MUST document the new advisory warning lines, the widened blocking class (change-id + gross over-cap), the 2×-cap escalation, the `warnings` JSON array, and updated exit-code text. Its SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md`'s `fab memory-index` inventory row MUST be updated in lockstep (constitution SPEC-mirror rule).

- **GIVEN** the `_cli-fab.md` change
- **WHEN** a reader consults the `fab memory-index` section
- **THEN** the blocking/advisory taxonomy, `warnings` array, and exit-code semantics match the shipped behavior, and SPEC-_cli-fab.md's row agrees

#### R10: FKF §3.2/§4 posture updated in BOTH fkf.md files
`docs/specs/fkf.md` AND `src/kit/reference/fkf.md` MUST update the FKF §3.2 posture: drop "No enforcement is added" (the change-id ban is now enforced/blocking; the length cap is advisory to 1000, blocking past it), document the two escalations and the three new advisory kinds (§4 shape-bounds text), keeping the shape bounds themselves advisory-never-enforced. The two files MUST NOT diverge (single-sourcing rule).

- **GIVEN** a reader consulting FKF §3.2 in either file
- **WHEN** they read the change-id and length rules
- **THEN** both files state the change-id ban is enforced (blocking), the length cap is advisory ≤1000 / blocking >1000, and the new advisory kinds are documented — with identical normative content

#### R11: Cobra help text updated
The `fab memory-index` cobra `Long` description and flag help text MUST be updated to describe the new blocking findings, advisory kinds, and the `warnings` JSON array.

- **GIVEN** `fab memory-index --help`
- **WHEN** a user reads it
- **THEN** the new guards are described consistently with `_cli-fab.md`

### Affected Memory (hydrate-stage, not apply)

#### R12: Memory docs reflect the widened taxonomy
`docs/memory/memory-docs/templates.md` (Memory Tree Shape) and `docs/memory/memory-docs/hydrate.md` (refuse-before-regen guard note) MUST be updated during **hydrate** to name the two blocking escalations, three advisory kinds, and broken-link detection, and to distinguish exit-2 from the widened malformed blocking class. (Memory edits are a hydrate-stage concern — apply does not write `docs/memory/`.)

- **GIVEN** the hydrate stage runs after this change
- **WHEN** memory is updated
- **THEN** templates.md and hydrate.md describe the new guards

### Non-Goals

- No config surface for any threshold — all are hardcoded package consts (the `DescriptionLenWarnThreshold` precedent).
- No `.status.yaml` schema change and no migration file.
- No exit-code renumbering; exit-2 semantics untouched.
- Consuming the new signals (survey heuristic, reorg split candidates, `_unsorted` triage) is `[dsrx]`'s scope, not this change.
- No `docs/memory/` edits in apply (those land in hydrate per R12).

### Design Decisions

1. **Blocking set generalizes, JSON key stays `malformed`**: rename the internal `malformedKinds`/`IsMalformed()` concept to a blocking set (`blockingKinds`/`IsBlocking()`) covering the two malformed kinds plus the two new escalations, but keep the `--json` array key `malformed` for consumer compatibility (`/docs-reorg-memory` branches on `tier`/`losses`). — *Why*: the intake mandates the JSON key stays `malformed`; the internal naming is at apply's discretion. — *Rejected*: a separate `blocking` array (breaks the additive contract and forces consumers to learn a second array).
2. **Broken-link scan skips both fenced code blocks AND inline code spans**: the scan tracks ```` ``` ```` fences and elides `` `…` `` inline spans before matching link tokens. — *Why*: this repo's own memory docs carry illustrative link-format examples like `` `[base](/{domain}[/{sub}]/base.md)` `` and `` `](/bundle/rel.md)` `` inside code markup; scanning them produced four false-positive advisory warnings on the born-FKF tree (verified). Skipping both fence types is the principled rule (a link shown inside code markup is documentation, not a live cross-link) and eliminates all four. — *Rejected*: scanning everything (four false positives); metacharacter-filtering only (fragile; misses the `/bundle/rel.md` example which has no metacharacters).
3. **SPEC-_cli-fab.md IS updated** despite the intake's "no SPEC mirror by policy" note: `docs/specs/skills/SPEC-_cli-fab.md` exists (backfilled 260620) and carries a `fab memory-index` inventory row describing the exact behavior changed here; the constitution's SPEC-mirror rule and code-quality.md § Sibling & Mirror Sweeps require updating it. — *Why*: repo reality overrides the stale intake note; leaving the row stale is a documented must-fix rework cause. — *Rejected*: honoring the intake's exclusion (the mirror demonstrably exists and describes the changed command).
4. **Registry gathered once, shared by all registry-gated scans**: `gatherChangeRegistry(fabRoot)` (already used by log.md) is called once in `Gather` and threaded into the frontmatter/body pass, matching the existing single-registry-pass pattern. — *Why*: avoids a second registry walk; mirrors `attributeCommit`'s consumer. — *Rejected*: a fresh registry gather per file (O(files × changes) I/O).

## Tasks

### Phase 1: Core detection primitives (internal/memoryindex)

- [x] T001 Add new Warning `Kind` consts to `src/go/fab/internal/memoryindex/memoryindex.go`: `KindDescriptionChangeID`, `KindDescriptionOverCap` (blocking); `KindNarrationDensity`, `KindFileSize`, `KindUnsorted`, `KindBrokenLink` (advisory). Add package consts `DescriptionBlockingLenThreshold = 1000`, `NarrationMarkerWarnThreshold = 5`, `FileSizeLineWarnThreshold = 400`, `FileSizeByteWarnThreshold = 15360`. Document each as a hardcoded shape-bound const (NOT config). <!-- R1 R2 R4 R5 R6 R7 -->
- [x] T002 Generalize the blocking set in `memoryindex.go`: rename `malformedKinds` → `blockingKinds` and `IsMalformed()` → `IsBlocking()`, adding `KindDescriptionChangeID` and `KindDescriptionOverCap` to the set; update the `Warning` struct doc and add `Count`/`Detail` usage notes for the new kinds. Update all in-package references. <!-- R1 R2 R3 -->
- [x] T003 Extend `Warning.String()` in `memoryindex.go` with stderr lines for all six new kinds (`✖` for the two blocking, `⚠` for the four advisory), matching the intake's example wording (change-id: registry match + FKF §3.2 pointer; over-cap: blocking cap 1000 / soft cap 500; narration: count + threshold + /docs-distill-memory; size: lines/bytes + soft cap + /docs-reorg-memory; unsorted: staged file count; broken-link: names target + "target does not exist"). <!-- R1 R2 R4 R5 R6 R7 -->
- [x] T004 Add a registry-gated change-id token scanner in `memoryindex.go` (a helper reusing `extractIDSlug` + the `attributeCommit` tokenization/gating shape) usable on both a `description:` value and a file body; return the matched id(s). <!-- R1 R4 -->

### Phase 2: Wire the scans into Gather

- [x] T005 Thread the change registry into the frontmatter/body pass: gather it once via `gatherChangeRegistry(fabRoot)` in `Gather` (add a `fabRoot` parameter to `Gather`, or resolve it inside) and pass it to `frontmatterWarnings`. Update the sole caller in `cmd/fab/memory_index.go` to pass `fabRoot`. <!-- R1 R4 -->
- [x] T006 Extend `frontmatterWarnings` (the existing read-only walk) in `memoryindex.go` to emit, per file: the change-id-in-description blocking finding (R1) and the > 1000-rune over-cap blocking finding (R2), keeping the existing 501–1000 advisory length warning unchanged. Preserve the topic-file + index.md-stub scope. <!-- R1 R2 -->
- [x] T007 Add per-topic-file body scans in `memoryindex.go` (topic files only — skip index.md/log.md/log.seed.md): narration-marker density (R4, stems + registry-gated change-id tokens, fires at ≥5), size (R5, >400 lines OR >15360 bytes), and broken bundle-relative links (R7, `](/...)` targets resolved under docs/memory/, skipping fenced code blocks). Add the `_unsorted/` non-empty presence check (R6). Keep all read-only and byte-stable. <!-- R4 R5 R6 R7 -->
- [x] T008 Ensure the deterministic warning sort in `Gather` still holds across the new kinds (sort by Path, then Kind) so output is byte-stable. <!-- R4 R5 R6 R7 -->

### Phase 3: cmd wiring — blocking floor + JSON warnings

- [x] T009 In `src/go/fab/cmd/fab/memory_index.go`, update the `--check` report build so the generalized blocking set (`IsBlocking()`) feeds `report.Malformed` (the two new blocking kinds join the existing two). Confirm the exit floor logic in `emitCheckReport` blocks on any `report.Malformed` entry independent of tier and that exit 2 still wins. <!-- R1 R2 R3 -->
- [x] T010 Add an additive `Warnings []WarningFinding` field to `LossReport` in `src/go/fab/internal/memoryindex/loss.go` (JSON key `warnings`, initialized non-nil/empty-never-null) with a `WarningFinding{Kind, Path, Count, Detail}` struct; populate it in `cmd/fab/memory_index.go` from the gathered advisory warnings (the non-blocking kinds). Keep `tier`/`drift`/`losses`/`malformed` unchanged. <!-- R8 -->
- [x] T011 Update the cobra `Long` description and flag help text on `memoryIndexCmd()` in `cmd/fab/memory_index.go` to describe the two blocking description findings, the four advisory kinds, and the `warnings` JSON array. <!-- R11 -->

### Phase 4: Tests

- [x] T012 [P] Add package tests in `src/go/fab/internal/memoryindex/memoryindex_test.go`: change-id-in-description blocking (registry match + false-positive-free on non-registered tokens), over-cap boundary (1000 no-block advisory / 1001 blocks), narration density boundary (4 no-warn / 5 warns; stems + registry-gated ids), size boundary (line and byte bounds), `_unsorted` presence, broken-link (fires on missing target, skips code fences), byte-stability of rendered output under all new findings, deterministic warning order. Add a registry fixture (fab/changes folder) so registry-gated scans resolve. <!-- R1 R2 R3 R4 R5 R6 R7 -->
- [x] T013 [P] Add cmd tests in `src/go/fab/cmd/fab/memory_index_test.go`: change-id/over-cap blocking floors `--check` at 1 on a byte-clean tree; blocking does not reach exit 2; `--check --json` carries the additive `warnings` array (empty-never-null on a clean tree, populated when advisory findings exist) with `tier`/`drift`/`losses`/`malformed` unchanged; advisory findings alone do not affect the exit code; help text exposes the new guards. <!-- R3 R8 R11 -->

### Phase 5: Documentation

- [x] T014 Update `src/kit/skills/_cli-fab.md` § fab memory-index: new advisory warning lines (narration, size, `_unsorted`, broken-link), widened blocking class (change-id + gross over-cap join the two malformed kinds), 2×-cap escalation wording, the `warnings` JSON array in the `--json` shape, and exit-code text. <!-- R9 -->
- [x] T015 Update `docs/specs/skills/SPEC-_cli-fab.md` `fab memory-index` inventory row in lockstep with T014 (the SPEC mirror). <!-- R9 -->
- [x] T016 Update FKF §3.2/§4 posture in BOTH `docs/specs/fkf.md` AND `src/kit/reference/fkf.md`: drop "No enforcement is added", document the change-id ban as enforced (blocking), the length cap advisory ≤1000 / blocking >1000, and the three new advisory kinds + broken-link detection under the shape-bounds text (shape bounds stay advisory-never-enforced). Keep the two files' normative content identical. <!-- R10 -->

### Phase 6: Rework (cycle 1 — review findings)

- [x] T017 Fix the stale advisory-only posture in `docs/specs/templates.md`: the "`description:` ≤ 500 characters … fab memory-index warns (advisory only; it never fails --check)" claim (~line 483) and the "Blocking" blockquote (~line 488) that enumerates only the two malformed-frontmatter signatures — update both to the four-kind blocking class (change-id + >1000 gross over-cap joined) and the split cap posture (501–1000 advisory nag, >1000 blocking). <!-- R10 -->
- [x] T018 Sweep the remaining advisory-only description-cap claims in kit sources: `src/kit/skills/docs-hydrate-memory.md` (~line 113) and `src/kit/skills/docs-distill-memory.md` (~line 178) and `src/kit/templates/memory.md` (~line 7) — rephrase each to the split posture (advisory ≤1000 / blocking >1000; change-id in description now blocking); update `docs/specs/skills/SPEC-docs-hydrate-memory.md` ("warns over the cap") and verify `SPEC-docs-distill-memory.md` in lockstep (SPEC-mirror rule). Finish with a repo-wide grep for remaining stale advisory-only claims about the description cap (exclude `docs/memory/` — that sweep is hydrate's). <!-- R10 R9 -->
- [x] T018b Complete the T018 sweep — two missed occurrences (cycle-2 re-review must-fix; use these EXACT replacements): <!-- rework: A-026 sweep gap — fab-continue.md:205 + SPEC-fab-continue.md:20 still claim advisory-only -->
  (a) `src/kit/skills/fab-continue.md` Hydrate Behavior Step 4 (~line 205): replace the parenthetical `` (`fab memory-index` emits an advisory over-length warning) `` with `` (`fab memory-index` warns over the 500 cap — advisory to 1000 runes, blocking `--check` past it; a change-id in the description is likewise a blocking finding) ``.
  (b) `docs/specs/skills/SPEC-fab-continue.md` (~line 20): replace `` detail belongs in the body, `fab memory-index` warns over the cap `` with `` detail belongs in the body, `fab memory-index` warns over the cap — and, as of 260718-mxgu, blocks `--check` past 1000 runes or on a change-id in the description ``. Then re-grep repo-wide (excluding `docs/memory/`) to confirm zero remaining unqualified advisory-only cap claims. <!-- R10 R9 -->
- [x] T019 Fix the cobra `Long` string-concatenation hyphenation bug in `src/go/fab/cmd/fab/memory_index.go` (~line 52): renders "narration- marker density" — should be "narration-marker density"; verify via the built binary's `--help`. <!-- R11 -->
- [x] T020 Fix the line-count off-by-one in the file-size scan (`strings.Count(content, "\n") + 1` overcounts trailing-newline files by 1; count actual lines so the reported metric matches `wc -l`); adjust the boundary test accordingly. <!-- R5 -->
- [x] T021 Add an additive `bytes` field to `WarningFinding` in `src/go/fab/internal/memoryindex/loss.go`, populated for file-size findings (a byte-bound-only trip is currently inexplicable in JSON); update the `--json` shape documentation in `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md` in lockstep; extend the cmd JSON test. <!-- R8 -->
- [x] T021b Complete T021's doc half (cycle-2 re-review must-fix): in `src/kit/skills/_cli-fab.md` § fab memory-index, the documented `--json` shape string's `warnings` entry — currently `{"kind": ..., "path": "<repo-rel file/folder>", "count": <N>, "detail": "<broken link target, omitted otherwise>"}` — must gain the `bytes` key the binary emits: insert `"bytes": <N — file-size findings only, omitted otherwise>, ` between the `count` and `detail` fields. SPEC-_cli-fab.md's row does not enumerate JSON fields, so only the `_cli-fab.md` shape string changes. <!-- rework: T021 code+test landed, shape string missed (CLI ⇒ docs rule) --> <!-- R8 R9 -->
- [x] T022 Add a cmd test asserting a POPULATED `warnings` array in `--check --json` (an advisory-only tier-0 tree returns nil — no os.Exit — so stdout is capturable in-process). <!-- R8 -->
- [x] T023 Tighten the `--json` flag help wording in `cmd/fab/memory_index.go` to enumerate the four advisory kinds the `warnings` array carries (the 501–1000 description-length nag is deliberately excluded from the array). <!-- R11 -->
- [x] T024 Add `—` (em-dash) and `*` to the change-id token separator set so a glued `—xu0k` suffix or bolded `**xu0k**` citation cannot escape the blocking description check (registry gating unchanged — still false-positive-free); extend the scanner test. <!-- R1 -->

## Execution Order

- Phase 1 (T001–T004) before Phase 2 (consts/kinds/helpers are prerequisites for the scans).
- T005 before T006/T007 (registry must be threaded before the scans consume it).
- Phase 2 before Phase 3 (cmd wiring consumes the gathered warnings).
- T010 before T013 (the JSON field must exist before the cmd test asserts it).
- Phase 4 tests after the code they cover (test-alongside).
- Phase 5 docs may run in parallel with Phase 4 but after Phase 1–3 fix the shipped behavior.

## Acceptance

### Functional Completeness

- [x] A-001 R1: A registry-matched change-id in a `description:` produces a `✖` blocking finding (on topic files and index.md stubs) that floors `--check` at 1 and never at 2.
- [x] A-002 R1: A `description:` with a non-registered 4-char word (e.g. `code`/`yaml`) produces no change-id finding (registry-gated false-positive-freedom).
- [x] A-003 R2: A `description:` strictly > 1000 runes produces a `✖` blocking finding; 501–1000 keeps the advisory `⚠` length warning.
- [x] A-004 R3: Both blocking kinds ride the `malformed` JSON array; `tier`/`drift`/`losses` are unchanged; exit 2 still wins when a tier-2 loss co-occurs.
- [x] A-005 R4: A topic file reaching 5 narration markers (stems + registry-gated change-id tokens) produces an advisory `⚠` warning reporting the count; exit code unaffected.
- [x] A-006 R5: A topic file over 400 lines OR over 15360 bytes produces an advisory `⚠` size warning; exit code unaffected.
- [x] A-007 R6: A non-empty `docs/memory/_unsorted/` produces one advisory `⚠` warning reporting the topic-file count; `_unsorted` keeps its width exemption.
- [x] A-008 R7: A broken bundle-relative `](/...)` target produces an advisory `⚠` warning naming source + target; targets inside code fences are skipped; external/repo-relative links are out of scope.
- [x] A-009 R8: `--check --json` carries an additive `warnings` array (empty-never-null) with the advisory findings; blocking kinds ride `malformed`; `tier`/`drift`/`losses` unchanged.

### Behavioral Correctness

- [x] A-010 R1 R2 R3 R4 R5 R6 R7: None of the new checks change the rendered index/log bytes on either the write or `--check` path (byte-stability holds).
- [x] A-011 R3: The exit-2 destructive-loss tier is unaffected — the hydrate/reorg refuse-before-regen guards (keyed on exit == 2) still fire only on true tier-2 loss.

### Scenario Coverage

- [x] A-012 R1 R2 R4 R5 R6 R7 R8: Package + cmd tests exercise each new finding's boundary (fires / does-not-fire) and the JSON surface.
- [x] A-013 R1 R4: The registry-gated scanner matches both a full folder-name token and a bare registered id, and rejects unregistered tokens.

### Edge Cases & Error Handling

- [x] A-014 R2: The over-cap threshold is strictly > 1000 (exactly 1000 does not block).
- [x] A-015 R4: The narration threshold fires at exactly 5, not at 4.
- [x] A-016 R7: The broken-link scan resolves anchors (`#...`) and only `/`-prefixed targets; a fenced-code example does not false-positive.
- [x] A-017 R1 R2 R4 R5 R6 R7: Running the built binary's `--check` against THIS repo's own `docs/memory/` fires no new BLOCKING findings (advisory warnings are acceptable and reported).

### Code Quality

- [x] A-018 Pattern consistency: New code follows the existing `internal/memoryindex` conventions (pure Warning kinds + `String()`, read-only `frontmatterWarnings`-style walk, registry gating mirroring `attributeCommit`).
- [x] A-019 No unnecessary duplication: The registry is gathered once and threaded; the change-id scanner reuses `extractIDSlug`/the `attributeCommit` gating shape rather than reimplementing token resolution.
- [x] A-020 Magic numbers: All thresholds are named package consts (no bare literals in the scan logic).
- [x] A-021 Go changes ship tests: every new detection path has an accompanying test (Constitution VII, test-alongside).

### documentation_accuracy

- [x] A-022 R9 R11: `_cli-fab.md` § fab memory-index and the cobra `Long`/flag help describe the two blocking findings, four advisory kinds, and the `warnings` JSON array consistently with the shipped behavior.
- [x] A-023 R10: FKF §3.2/§4 in BOTH `docs/specs/fkf.md` and `src/kit/reference/fkf.md` drop "No enforcement is added", document the escalations + advisory kinds, keep shape bounds advisory, and carry identical normative content.

### cross_references

- [x] A-024 R9: `docs/specs/skills/SPEC-_cli-fab.md`'s `fab memory-index` inventory row is updated in lockstep with `_cli-fab.md` (SPEC-mirror rule).
- [x] A-025 R10: The two fkf.md files' shared §3.2/§4 edits do not diverge (single-sourcing rule); section anchors resolve identically.

### Rework (cycle 1)

- [x] A-026 R10 R9: No kit source or spec still claims the description over-cap warning is advisory-only without the >1000 blocking qualification (`docs/specs/templates.md`, `docs-hydrate-memory.md` + SPEC mirror, `docs-distill-memory.md` + SPEC mirror verified, `src/kit/templates/memory.md` all swept; `docs/memory/` deferred to hydrate). <!-- MET (re-review cycle 2): T018b landed both pinned splices verbatim (fab-continue.md Hydrate Step 4 + SPEC-fab-continue.md:20); repo-wide re-grep (excl. docs/memory/, fab/changes/) finds only properly qualified split-posture claims -->
- [x] A-027 R8: File-size `warnings` entries carry an additive `bytes` key; a populated `warnings` array (not just the empty case) is asserted in the cmd JSON test.
- [x] A-028 R11 R5: The built binary's `--help` renders "narration-marker density" (no split hyphenation); reported line counts match `wc -l`.
- [x] A-029 R1: A glued `—xu0k` suffix and a bolded `**xu0k**` citation (registered id) are detected by the blocking description check.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- A-017 is the intake's explicit self-check: no new blocking finding on this repo's born-FKF tree; advisory warnings here are acceptable and reported.
- Rework cycle 1 triage (review verdict fail, 2 must-fix / 4 should-fix / 4 nice-to-have): must-fix sweep gaps → T017/T018; should-fixes taken → T019 (help hyphenation), T020 (line-count off-by-one), T021 (`bytes` JSON field), T022 (populated-warnings test); nice-to-haves taken → T023 (help wording), T024 (separator set). **Skipped**: single-line-fence tracker desync + `~~~` fences (advisory-only, false-negative direction, edge case — deferred); exit-2-wins-over-blocking subprocess regression test (verified empirically during review; in-process testing blocked by `os.Exit(2)`, a pre-existing limitation shared by all exit-2 paths). **Routed to hydrate**: `docs/memory/pipeline/schemas.md` (`IsMalformed()` → `IsBlocking()`, blocking-class widening) and `docs/memory/distribution/kit-architecture.md` (`--json` shape: `malformed` + `warnings` arrays) — memory edits are hydrate-stage; outside the intake's Affected Memory list but in hydrate's sweep class.

## Deletion Candidates

- None — this change adds new detection paths (blocking description guards, advisory body meters, broken-link scan, `warnings` JSON array) without making existing code redundant. The one internal rename (`malformedKinds`→`blockingKinds`, `IsMalformed()`→`IsBlocking()`) replaced the old symbols in place; `attributeCommit`'s own tokenizer deliberately keeps its narrower commit-subject separator set and is not subsumed by `changeIDTokenSep`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Blocking escalations join the exit-1 malformed-frontmatter floor class, never the exit-2 destructive-loss tier | Intake hard constraint; code confirms hydrate/reorg guards key on exit == 2 | S:95 R:80 A:95 D:95 |
| 2 | Confident | Gross over-cap blocking threshold = strictly > 1000 runes; 501–1000 stays advisory | Intake states 2× the 500 cap as the anchor; hardcoded const trivially tuned later | S:75 R:85 A:80 D:80 |
| 3 | Confident | Change-id detection is registry-gated, mirroring `attributeCommit`'s full-folder-token / bare-registered-id gating | Blocking checks need false-positive-freedom; `attributeCommit` is the in-package precedent; intake specifies it | S:70 R:75 A:85 D:75 |
| 4 | Certain | Broken-link detection is advisory, never blocking | FKF §7: consumers tolerate broken links — a missing target is not malformed | S:70 R:85 A:90 D:85 |
| 5 | Confident | Narration density fires at ≥ 5 markers (four stems + registry-gated change-id tokens, sanctioned citations included) | Intake proposes ≥5; advisory + hardcoded const = cheap to tune; density (not violation) is the signal | S:55 R:85 A:60 D:50 |
| 6 | Confident | Size warning fires on > 400 lines OR > 15360 bytes (15KB), hardcoded consts | Intake gives ~400 lines / ~15KB; OR-semantics catches both mega-file shapes; 15KB = 15×1024 bytes | S:70 R:85 A:75 D:70 |
| 7 | Certain | `_unsorted/` warning fires on ≥ 1 topic file present; keeps its width exemption | Intake: staging should trend to empty — presence is the signal; width exemption governs a different bound | S:80 R:85 A:90 D:85 |
| 8 | Confident | `--check --json` gains an additive `warnings` array; blocking kinds ride `malformed`; `tier`/`drift`/`losses` unchanged | Intake specifies the machine surface for [dsrx]; additive arrays follow the 260715-xu0k `malformed` precedent | S:70 R:80 A:80 D:70 |
| 9 | Certain | All thresholds are hardcoded package consts, not config | Direct precedent: `DescriptionLenWarnThreshold` is in-code documented as NOT config-overridable | S:80 R:85 A:95 D:90 |
| 10 | Certain | New per-file body checks scan topic files only; description blocking checks also cover index.md stubs | Mirrors the established `frontmatterWarnings`/`gatherFiles` skip sets; matches existing malformed-check scope | S:70 R:85 A:85 D:80 |
| 11 | Confident | The broken-link scan skips both fenced code blocks AND inline code spans | This repo's own memory docs carry link-format examples in code markup (fenced + inline) that would false-positive; skipping both eliminates all four on the born-FKF tree (verified — 0 broken-link warnings after) | S:65 R:80 A:80 D:65 |
| 12 | Confident | SPEC-_cli-fab.md IS updated (its `fab memory-index` row), contra the intake's "no SPEC mirror" note | The mirror demonstrably exists (backfilled 260620) and carries a memory-index row describing the changed command; constitution + code-quality.md require lockstep SPEC updates | S:80 R:75 A:90 D:80 |
| 13 | Confident | The internal blocking set/predicate is renamed (`malformedKinds`→`blockingKinds`, `IsMalformed`→`IsBlocking`) while the `--json` key stays `malformed` | Intake sanctions internal renaming at apply's discretion and pins the JSON key for consumer compatibility | S:70 R:85 A:80 D:70 |

13 assumptions (6 certain, 7 confident, 0 tentative).
