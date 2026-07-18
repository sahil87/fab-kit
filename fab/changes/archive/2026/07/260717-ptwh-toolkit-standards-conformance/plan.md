# Plan: Toolkit Standards Conformance

**Change**: 260717-ptwh-toolkit-standards-conformance
**Intake**: `intake.md`

## Requirements

<!-- Requirements are derived from the intake's audit-then-fix design and the four
     runtime-enumerated standards (shll v0.0.23). Each requirement is traced by
     tasks (T#) and acceptance (A#). -->

### Standards: Runtime Enumeration & Audit Baseline

#### R1: Audit against the runtime-enumerated standards, not memory
The audit set MUST be the standards enumerated at apply entry by `shll standards`, read in full via `shll standards <name>`, and the audit MUST be pinned to the installed shll version (`shll version`'s shll row). If `shll standards` is missing, run `shll update` once; if it still fails, STOP.

- **GIVEN** an apply run of this change
- **WHEN** `shll standards` is run and each entry read
- **THEN** the audit covers exactly the enumerated standards (at apply time: `principles`, `help-dump`, `readme-extraction`, `skill` @ shll v0.0.23) and the report states the audited shll version

#### R2: Audit the binary by building the worktree source
help-dump/principle audits of the binary MUST run against a freshly built worktree binary (`src/go/fab`), never the installed `fab` (version skew).

- **GIVEN** a binary-scope audit
- **WHEN** `help-dump` conformance is checked
- **THEN** it is checked against a `go build`-produced artifact of `src/go/fab/cmd/fab`, not the installed `fab`

### Standard: help-dump (mechanical, binary)

#### R2H: The help-dump envelope MUST NOT emit `captured_at`
The `fab help-dump` envelope MUST be exactly `{tool, version, schema_version, root}`, with `schema_version` the integer `1`, no `captured_at` field (owned by shll.ai's puller). The command MUST exit 0, write valid JSON to stdout only, stderr empty; `completion`/`help`/hidden commands absent; every node carries `text` + `short`/`usage`/`path`.

- **GIVEN** the built worktree binary
- **WHEN** `fab help-dump` is run
- **THEN** stdout is valid JSON whose top-level keys are exactly `tool, version, schema_version, root` (no `captured_at`), exit 0, stderr empty

#### R3: help-dump tests pin the conformant envelope
The Go tests MUST assert the absence of `captured_at` (on the encoded bytes), pin the top-level key order to `tool, version, schema_version, root`, and keep a minimal end-to-end conformance test (exit 0, valid JSON, expected `tool`/`schema_version`).

- **GIVEN** `helpdump_test.go`
- **WHEN** `go test ./cmd/fab/` runs
- **THEN** it fails if `captured_at` is ever re-introduced and passes on the conformant envelope

#### R4: help-dump output-shape docs stay accurate
Because the emitted output shape changed, `src/kit/skills/_cli-fab.md`'s documented envelope MUST match (no `captured_at`), and any now-stale surrounding prose about the delivery model MUST be corrected.

- **GIVEN** `_cli-fab.md` § fab help-dump
- **WHEN** the envelope is documented
- **THEN** it shows `{tool, version, schema_version, root}` and describes the pull model accurately

### Standard: readme-extraction (mechanical, repo)

#### R5: README slice-region relative links leaving the published set MUST be absolute
Every doc link in the README slice region (head → `## Development` tail boundary) that leaves the published set (README slice + `docs/site/**`) MUST be an absolute `https://github.com/sahil87/fab-kit/blob/main/<path>` URL. README→`docs/site/` links stay repo-relative (the sanctioned auto-rewritten form).

- **GIVEN** the README slice region
- **WHEN** grepping `](docs/` / `](CONTRIBUTING.md)` etc.
- **THEN** every relative target either points into `docs/site/` (auto-rewritten) or has been made absolute; no relative link to `docs/specs/`/`CONTRIBUTING.md` survives above the tail boundary

#### R6: The remaining readme-extraction checklist items PASS unchanged
Head structure (`#` H1 → toolkit blockquote → contiguous badges → tagline), all images absolute, no `#gh-*-mode-only` fragments, no site-worthy mermaid-only diagrams, no reserved `docs/site/` slug (`overview`/`readme`/`commands`), README cross-links its `docs/site/` pages + the absolute command-reference URL, and the `docs/site/**` closed-set rules MUST remain satisfied. These are verified but require no change.

- **GIVEN** `README.md` + `docs/site/**`
- **WHEN** the verification checklist is executed verbatim
- **THEN** every item except R5 already passes (no edit needed); the command-reference URL `https://shll.ai/tools/fab-kit/commands/` is confirmed correct against shll.ai's actual routing (`tools/fab-kit/…`)

### Standard: principles (foundation, 10)

#### R7: Small, additive principle gaps fixed here; restructuring gaps deferred with references
Each of the ten principles MUST be assessed against `fab`'s actual behavior. Any gap that is small and additive is fixed in this change; any restructuring-sized gap MUST be recorded as a `fab/backlog.md` entry (fresh 4-char ID) and referenced from the report.

- **GIVEN** the principle audit
- **WHEN** a gap is found
- **THEN** it is dispositioned as fixed-here (named files) or deferred-to-[id]; the audit found P1/P5/P6/P7/P8/P10 PASS and P2/P3/P4/P9 as deferred (systemic/multi-command) gaps, each recorded in the backlog

### Standard: skill (binary + repo)

#### R8: `fab skill` reported "deferred, not yet adopted" — never implemented
`fab skill` is confirmed absent (no subcommand, no `docs/site/skill.md`). Per the standard's Adoption section, the report MUST read "deferred, not yet adopted"; the subcommand MUST NOT be implemented in this change.

- **GIVEN** the absent `fab skill`
- **WHEN** the report is written
- **THEN** the skill section reads "deferred, not yet adopted" and no `skill` command is added

### Deliverable

#### R9: One per-standard conformance report is the PR body
A `conformance-report.md` MUST be written to the change folder with one section per runtime-enumerated standard, each PASS or listing gaps dispositioned as fixed-here (named files) or deferred-to-[ref], plus the audited shll version.

- **GIVEN** the completed audit + fixes
- **WHEN** the report is written
- **THEN** `fab/changes/260717-ptwh-toolkit-standards-conformance/conformance-report.md` exists with one section per standard and the shll version

### Non-Goals

- Implementing `fab skill` (deferred per the standard's Adoption section — not yet in violation)
- Fixing restructuring-sized principle gaps (exit-code renumbering, `--json` across the status surface, `Example:`/`--quiet` across multiple commands) — deferred to backlog `[swon]`/`[jx4w]`/`[b91h]`/`[o5f9]`
- Auditing the `fab-kit`/router/shim binaries (standards name `fab`; binary scope targets `src/go/fab`)
- Editing memory files (`docs/memory/distribution/*`) — that is hydrate's job (Affected Memory)

### Design Decisions

1. **`captured_at` removal is a conformance fix, not a schema break** — the consumer owns and stamps it; `schema_version` stays `1`. *Why*: the standard forbids emitting it verbatim. *Rejected*: bumping `schema_version` (removing a consumer-owned field is not a breaking change to the consumer).
2. **Command-reference URL kept as `https://shll.ai/tools/fab-kit/commands/`** — *Why*: verified against shll.ai's actual routing (`sites/.../content/docs/tools/fab-kit/commands`); the standard's `/<tool>/commands/` is a template with `<tool>` = `tools/fab-kit` for this repo. *Rejected*: rewriting to `/fab/commands/` (would 404).
3. **Deferred all four principle gaps (P2/P3/P4/P9)** — *Why*: P4 (exit codes) needs cross-command reconciliation with existing domain-specific exit-2 uses (memory_index/pane); P2 spans the whole status surface; P3/P9 span ~7/2 commands and each adds signature+mirror+test obligations, exceeding "a missing flag on one command." *Rejected*: fixing P3/P9 here (multi-command additive polish is a coherent deferred workstream, and the intake's strongest fix-here mandate is the mechanical contracts).

## Tasks

### Phase 1: Runtime enumeration & audit (baseline)

- [x] T001 Re-run `shll standards` + `shll standards <name>` for every entry and `shll version`; confirm the enumerated set and audited shll version (v0.0.23; four standards). <!-- R1 -->
- [x] T002 Build the worktree binary (`go build ./cmd/fab` under `src/go/fab`) and run the help-dump "Verifying conformance" checklist verbatim against the built artifact. <!-- R2 R2H -->
- [x] T003 Execute the readme-extraction "Verifying conformance" checklist verbatim against `README.md` + `docs/site/**`; confirm the command-reference tool slug against shll.ai's actual routing. <!-- R5 R6 -->
- [x] T004 Assess each of the ten principles against `fab`'s actual behavior (prompts/TTY, stream routing, `--json`/`--dry-run`/`--yes`, exit codes/error wording, idempotency, output volume). <!-- R7 -->
- [x] T005 Confirm `fab skill` + `docs/site/skill.md` absent. <!-- R8 -->

### Phase 2: help-dump fix

- [x] T006 Remove the `CapturedAt` field from the `HelpDoc` struct and its population in `dumpDoc` (`src/go/fab/cmd/fab/helpdump.go`); drop the now-unused `time` import. <!-- R2H -->
- [x] T007 Update `src/go/fab/cmd/fab/helpdump_test.go`: delete the `captured_at`-presence/RFC3339 assertions, add a `captured_at`-absence assertion on the encoded bytes, fix the key-order assertion to `tool, version, schema_version, root`, and add a minimal end-to-end conformance test (exit 0, valid JSON to stdout, expected `tool`/`schema_version`, no `captured_at`). <!-- R3 -->
- [x] T008 Rebuild the binary and re-run the help-dump verification checklist to confirm the conformant envelope. <!-- R2H R2 -->

### Phase 3: docs & readme-extraction fixes

- [x] T009 Update `src/kit/skills/_cli-fab.md` § fab help-dump: remove `captured_at` from the documented envelope, add the envelope/no-`captured_at` note, and correct the stale delivery-model prose to the pull model. Confirm the SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` needs no content change (one-line inventory row only). <!-- R4 -->
- [x] T010 Rewrite the 8 slice-region relative links in `README.md` (7 `docs/specs/*.md` in Companion tools + Learn More, and the `CONTRIBUTING.md` at line 658) to absolute `https://github.com/sahil87/fab-kit/blob/main/<path>` URLs; leave the below-`## Development` `CONTRIBUTING.md` link and the `docs/site/` links untouched. <!-- R5 -->

### Phase 4: deferrals & deliverable

- [x] T011 Record the four deferred principle gaps as `fab/backlog.md` entries with fresh 4-char IDs (`swon` P4 exit codes, `jx4w` P2 `--json`, `b91h` P3 `Example:`, `o5f9` P9 `--quiet`). <!-- R7 -->
- [x] T012 Write `fab/changes/260717-ptwh-toolkit-standards-conformance/conformance-report.md` — one section per runtime-enumerated standard, each PASS or listing gaps dispositioned as fixed-here (named files) or deferred-to-[id], the skill section "deferred, not yet adopted", and the audited shll version. <!-- R8 R9 -->

### Phase 5: tests

- [x] T013 Run `go test ./cmd/fab/` (touched package) then `go test ./...` (fab module); confirm green. Re-run the help-dump verification checklist since help-dump output changed. <!-- R3 R2H -->

### Phase 6: Review rework (cycle 1)

- [x] T014 Fix `src/kit/skills/_cli-external.md` § help-dump contract (~lines 36–63): the example envelope + Fields line still present `captured_at` as "part of the contract" with fab implicitly among the emitters. Update to reflect the shll v0.0.23 standard: the field is forbidden toolkit-wide (the puller stamps it post-capture); name fab among the tools that omit it, and note remaining emitters drop it on their own release cadence. Then confirm the SPEC mirror `docs/specs/skills/SPEC-_cli-external.md` still needs no content change (review verified it carries no envelope detail — re-verify after the edit). <!-- rework: review must-fix — repo-wide captured_at behavior-claim sweep missed this file (code-quality.md § Sibling & Mirror Sweeps) --> <!-- R4 -->
- [x] T015 Fix `src/kit/skills/_cli-fab.md` §  fab help-dump opening line (~line 892): "Hidden, CI/build-time-only command" contradicts the corrected pull-model paragraph below it. Reword to the pull model, e.g. "Hidden, machine-consumer command (invoked by shll.ai's puller on a schedule)". <!-- rework: review should-fix — residual stale delivery-model phrase in the section T009 edited --> <!-- R4 -->
- [x] T016 Fix `fab/changes/260717-ptwh-toolkit-standards-conformance/conformance-report.md` (~line 49): the rendered SVG images *precede* the mermaid fences in README (lines 31→36, 500→507), not follow them — correct the direction wording since this file becomes the PR body. <!-- rework: review nice-to-have — accepted because the report ships as the PR body --> <!-- R9 -->
- [x] T017 Re-sweep: repo-wide grep for `captured_at` and stale help-dump delivery-model claims (push/CI/release-workflow framing) across `src/kit/`, `docs/specs/`, `README.md`, `docs/site/**`; confirm every remaining occurrence is now correct. Do NOT touch `docs/memory/**` (the `kit-architecture.md:353` drift is hydrate's — already flagged for hydrate scope extension). <!-- rework: post-rework behavior-claim re-sweep — required after every behavior-changing rework --> <!-- R4 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: The audit covers exactly the runtime-enumerated standards and the report states the audited shll version (v0.0.23).
- [x] A-002 R2: help-dump/principle binary audits ran against a freshly built worktree binary, not the installed `fab`.
- [x] A-003 R2H: `fab help-dump` on the built binary emits `{tool, version, schema_version, root}` with `schema_version:1`, no `captured_at`, exit 0, valid JSON to stdout, stderr empty.
- [x] A-004 R4: `_cli-fab.md` documents the conformant envelope (no `captured_at`); the SPEC mirror needs no content change.
- [x] A-005 R5: No relative link to `docs/specs/`/`CONTRIBUTING.md` survives in the README slice region (above `## Development`); all rewritten to absolute GitHub-blob URLs.
- [x] A-006 R8: The report's skill section reads "deferred, not yet adopted"; no `fab skill` command was added.
- [x] A-007 R9: `conformance-report.md` exists with one section per standard, per-gap dispositions, and the shll version.

### Behavioral Correctness

- [x] A-008 R3: `helpdump_test.go` fails on any re-introduction of `captured_at` and passes on the conformant envelope; the minimal conformance test exercises the real command.

### Removal Verification

- [x] A-009 R2H: `CapturedAt` and its `time.Now()` population are gone from `helpdump.go`; no other `captured_at`/`CapturedAt` reference remains in `src/go/fab`.

### Scenario Coverage

- [x] A-010 R6: The full readme-extraction checklist was executed verbatim; every item except R5 passed with no change (head structure, absolute images, no gh-mode fragments, reserved slugs, command-ref URL, docs/site closed-set rules).
- [x] A-011 R7: Each principle gap is dispositioned; P2/P3/P4/P9 recorded in the backlog with fresh IDs and referenced from the report.

### Edge Cases & Error Handling

- [x] A-012 R1: If `shll standards` had failed after `shll update`, the run would STOP and report (precondition verified holding: exit 0, four standards, v0.0.23).

### Code Quality

- [x] A-013 Pattern consistency: The Go edits follow the existing struct/encoder/test patterns in `helpdump.go`/`helpdump_test.go`; the doc edits follow `_cli-fab.md` section conventions; the backlog entries follow the `[id] YYYY-MM-DD:` format (cf. `[clix]`).
- [x] A-014 No unnecessary duplication: The `captured_at`-absence pin reuses the existing `encodeDoc` helper; the conformance test reuses `newRootCmd`.
- [x] A-015 Test integrity: Tests were changed to match the standard (the spec), not the implementation bent to fixtures (Constitution VII) — the standard forbids `captured_at`, so the tests now assert its absence.

### Documentation Accuracy & Cross-References

- [x] A-016 Documentation accuracy: `_cli-fab.md`'s help-dump envelope and delivery-model prose match the shipped behavior (no `captured_at`, pull model).
- [x] A-017 Cross-references: The report references the backlog IDs for deferred gaps and names the exact files for fixed-here gaps; the README absolute URLs resolve to existing files.

## Notes

- Check items as reviewed: `- [x]`
- Constitution CLI constraint: this is an output-shape-only help-dump change (no command *signature* change), so `_cli-fab.md` was updated for accuracy and tests shipped; no SPEC-mirror content change was required (the mirror carries only a one-line row).
- `.claude/skills/` deployed copies were never edited — only the canonical `src/kit/skills/_cli-fab.md`.

## Deletion Candidates

- None — this change adds new functionality without making existing code redundant (the only redundancy it created — the unused `time` import and the `captured_at` presence/RFC3339 test assertions — was already deleted within the change itself).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Command-reference URL `https://shll.ai/tools/fab-kit/commands/` is correct (not a violation) — the standard's `/<tool>/commands/` renders under `tools/fab-kit/` for this repo | Verified against shll.ai's actual routing (`sites/.../content/docs/tools/fab-kit/commands`) + repo memory | S:90 R:85 A:95 D:90 |
| 2 | Confident | Defer principles P3 (`Example:`) and P9 (`--quiet`) rather than fix here | Each spans multiple commands and adds signature+mirror+test obligations, exceeding the intake's "a missing flag on one command" small-additive bar; the strongest fix-here mandate is the mechanical contracts | S:70 R:85 A:75 D:65 |
| 3 | Certain | Defer principle P4 (exit codes) and P2 (`--json` on status) | P4 needs reconciliation with existing domain-specific exit-2 uses (memory_index/pane); P2 spans the whole status query surface — both restructuring-sized per the intake | S:85 R:85 A:85 D:85 |
| 4 | Certain | Update `_cli-fab.md` (envelope + stale prose) even though this is output-shape-only, not a signature change | Constitution requires `_cli-fab.md` reflect the CLI; leaving a documented field the CLI no longer emits is a doc-accuracy defect; SPEC mirror unaffected (one-line row) | S:85 R:90 A:90 D:85 |
| 5 | Confident | Principle 7 (wt flag hardcoding) is PASS-with-caveat, not a fixed/deferred gap | The code degrades safely (offline ls-remote → positional → wt re-checks and errors visibly); full `--help` probing is optional, not a violation given the fallback | S:65 R:80 A:75 D:70 |

5 assumptions (3 certain, 2 confident, 0 tentative).
