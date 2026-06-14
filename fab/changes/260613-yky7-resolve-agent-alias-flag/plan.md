# Plan: resolve-agent --alias flag (Claude-Code model alias adapter)

**Change**: 260613-yky7-resolve-agent-alias-flag
**Intake**: `intake.md`

## Requirements

<!-- Derived from the intake-authoritative design. RFC 2119 keywords; stable R# IDs;
     each requirement carries at least one GIVEN/WHEN/THEN scenario. -->

### Alias mapping: `internal/agent`

#### R1: ModelAlias prefix-based family mapping
The `agent` package MUST expose an exported `ModelAlias(model string) string` that maps a full
Claude model ID to its Claude-Code short alias (`opus` / `sonnet` / `haiku` / `fable`) using a
**prefix** match (`claude-opus-` → `opus`, `claude-sonnet-` → `sonnet`, `claude-haiku-` → `haiku`,
`claude-fable-` → `fable`), so dated/versioned variants resolve.

- **GIVEN** a full ID `claude-opus-4-8`
- **WHEN** `ModelAlias` is called
- **THEN** it returns `opus`
- **AND** `claude-sonnet-4-6` → `sonnet`, `claude-haiku-4-5` → `haiku`, `claude-fable-1` → `fable`
- **AND** the dated variant `claude-haiku-4-5-20251001` → `haiku` (prefix match absorbs the date suffix)

#### R2: ModelAlias verbatim pass-through for empty and unmapped inputs
`ModelAlias` MUST return its input VERBATIM when no mapping applies — an empty string stays empty
(preserving the "inherit the session model" signal), and an unrecognized / non-Claude ID is returned
unchanged (so `--alias` stays a best-effort Claude-Code adapter, NOT a validator that rejects other
providers' models).

- **GIVEN** an empty model string `""`
- **WHEN** `ModelAlias("")` is called
- **THEN** it returns `""` (empty in, empty out)
- **GIVEN** a non-Claude ID `gpt-5`
- **WHEN** `ModelAlias("gpt-5")` is called
- **THEN** it returns `gpt-5` verbatim (no mapping, pass-through)

### CLI: `fab resolve-agent --alias`

#### R3: --alias flag emits the short alias on the model= line
`fab resolve-agent <stage>` MUST accept a boolean `--alias` flag. When set, the resolved
`profile.Model` SHALL be passed through `ModelAlias` before formatting, so the `model=` line emits the
short alias. The `effort=` line MUST be unaffected by `--alias`.

- **GIVEN** the default config (no `agent.tiers` override)
- **WHEN** `fab resolve-agent apply --alias` runs
- **THEN** stdout is `model=opus\neffort=high\n`
- **AND** the `effort=` line is identical to the non-`--alias` output

#### R4: default (no --alias) is byte-identical to today
With `--alias` absent, `fab resolve-agent <stage>` MUST emit exactly today's output (the full model
ID). The CLI/operator path and the byte-stable two-line contract are unchanged.

- **GIVEN** the default config
- **WHEN** `fab resolve-agent apply` runs (no flag)
- **THEN** stdout is `model=claude-opus-4-8\neffort=high\n` (regression guard — full ID preserved)

#### R5: empty-model tier under --alias still emits an empty model= line
When the resolved tier has an empty model (the "inherit" signal), `--alias` MUST leave the `model=`
line empty (since `ModelAlias("")` → `""`).

- **GIVEN** a tier whose resolved model is empty
- **WHEN** `fab resolve-agent <stage> --alias` runs
- **THEN** the `model=` line is empty (`model=\n`) — the inherit signal is preserved under `--alias`

### Documentation & skill wiring

#### R6: CLI reference documents --alias
`src/kit/skills/_cli-fab.md` § fab resolve-agent MUST document the `--alias` flag: that it emits the
short alias on the `model=` line, that it is the Claude-Code Agent-tool adapter, that the default
(absent) behavior is unchanged (full ID), that the `effort=` line is unaffected, and that
empty/non-Claude models pass through verbatim.

- **GIVEN** the CLI reference for `fab resolve-agent`
- **WHEN** a reader looks up the command signature
- **THEN** the `--alias` flag and its semantics are documented

#### R7: dispatch prose repointed from hand-map to --alias
The post-#413 adapter prose that currently instructs the orchestrator to "map the resolved id to the
alias at the dispatch seam" (the hand-map instruction) MUST be repointed to resolve the model half with
`fab resolve-agent <stage> --alias` (which emits an Agent-tool-valid alias directly). This applies to
the canonical `_preamble.md` § Harness-adapter boundary paragraph and every sibling site that carries
the same hand-map / Agent-tool-model-half dispatch instruction: `fab-ff.md`, `fab-fff.md`,
`fab-continue.md`, the shared bracket `_pipeline.md`, and the spec `docs/specs/stage-models.md`. No
sibling SHALL be left on the stale "maps id → alias by hand" / "short alias" phrasing.

- **GIVEN** the `_preamble.md` Harness-adapter boundary paragraph
- **WHEN** it is read after this change
- **THEN** it instructs resolving the model half via `fab resolve-agent <stage> --alias`, not a manual id→alias map
- **AND** a tree-wide grep for the old "short alias … orchestrator maps … id … alias" hand-map phrasing returns no skill/spec prose still on the manual instruction

#### R8: operator launcher path keeps resolving WITHOUT --alias
The operator launcher (`fab operator` / `_cli-fab.md` operator section) appends `--model <full-id>` to a
`claude` CLI invocation, which accepts full IDs. It MUST continue to resolve WITHOUT `--alias` — the
CLI and Agent-tool paths deliberately diverge.

- **GIVEN** the operator launcher resolving the doing-tier model
- **WHEN** this change ships
- **THEN** it still runs `fab resolve-agent apply` (no `--alias`) and appends the full model ID

#### R9: SPEC mirrors updated where they reference the hand-map
Per the Constitution (skill behavior changes update the corresponding `SPEC-*.md`), the SPEC mirrors
that reference the id→alias hand-mapping (`docs/specs/skills/SPEC-_preamble.md`) MUST be updated to the
`--alias` mechanism. SPEC mirrors that only mention the model-param seam generically (no hand-map
phrasing) need no edit.

- **GIVEN** `SPEC-_preamble.md` describing the dispatch seam with the "short alias … maps id→alias" phrasing
- **WHEN** read after this change
- **THEN** it describes the `--alias` resolver flag as the deterministic Agent-tool model-half mechanism

### Non-Goals

- Switching tier defaults to aliases — explicitly rejected (full IDs stay canonical in `defaultTiers`
  and the two drift-guarded spec tables; provider-neutrality + Fable version-pin preserved).
- Touching the two drift-guarded tables in `stage-models.md` (default tier profiles, stage→tier mapping)
  or the `defaultTiers`/`stageTiers` Go maps — `TestDocTablesMatchAgentMaps` must stay unaffected.
- Adding an effort param/flag — `--alias` touches only the `model=` line; the #413 effort-prompt seam
  is left exactly as shipped.
- Validating models — `--alias` is a best-effort adapter, not a Claude-only validator.

### Design Decisions

1. **Apply `ModelAlias` to `profile.Model` before formatting (in the RunE), not a format variant**:
   the cleanest seam — `formatAgentProfile` stays a pure formatter with the unchanged byte contract;
   the `--alias` transform is a one-line pre-format mutation in `resolveAgentCmd`. — *Why*: keeps the
   omit-when-empty / inherit branches of the formatter untouched and independently testable. —
   *Rejected*: threading a bool into `formatAgentProfile` (couples the formatter to a flag it doesn't
   need; the empty-model branch already does the right thing because `ModelAlias("")` → `""`).
2. **Prefix match over exact map**: absorbs dated variants (`claude-haiku-4-5-20251001` → `haiku`)
   without enumerating every version. — *Why*: the Agent enum is family-level; full IDs carry
   version/date suffixes. — *Rejected*: exact-string map (brittle; breaks on the next dated release).
3. **Repoint `_pipeline.md` too** (the shared bracket), though the intake's explicit list names only
   `_preamble/ff/fff/continue/stage-models`: the thin wrappers `fab-ff.md`/`fab-fff.md` delegate their
   Steps 1–3 dispatch to the `_pipeline.md` bracket, whose per-stage-model note carries the same
   Agent-tool-model-half dispatch instruction. Leaving the canonical bracket un-repointed while the
   wrappers say `--alias` would be an inconsistent stale sibling. — *Why*: the intake's own directive is
   "sweep ALL siblings; do not leave a stale sibling." — *Rejected*: editing only the literal list (would
   contradict the sweep directive).

## Tasks

### Phase 1: Core Implementation (Go)

- [x] T001 Add exported `ModelAlias(model string) string` to `src/go/fab/internal/agent/agent.go` — prefix-based family map (`claude-opus-`→`opus`, `claude-sonnet-`→`sonnet`, `claude-haiku-`→`haiku`, `claude-fable-`→`fable`), empty→empty, unmapped/non-Claude→verbatim; placed alongside the tier tables / `Resolve` <!-- R1 R2 -->
- [x] T002 Add a `--alias` bool flag to `resolveAgentCmd` in `src/go/fab/cmd/fab/resolve_agent.go`; when set, apply `agent.ModelAlias` to `profile.Model` before `formatAgentProfile`. Default path unchanged (full ID). Update the command doc comment to note the flag. <!-- R3 R4 R5 -->

### Phase 2: Go Test Coverage (test-alongside)

- [x] T003 [P] Unit-test `ModelAlias` in `src/go/fab/internal/agent/agent_test.go`: all four families → alias; dated `claude-haiku-4-5-20251001` → `haiku`; empty → empty; `gpt-5` → verbatim <!-- R1 R2 -->
- [x] T004 [P] Command-level test in `src/go/fab/cmd/fab/resolve_agent_test.go`: `apply --alias` → `model=opus\neffort=high\n`; `apply` (no flag) → `model=claude-opus-4-8\neffort=high\n` (regression guard); empty-model under `--alias` → empty `model=` line (asserted at alias+formatter level — no config resolves to an empty model) <!-- R3 R4 R5 -->

### Phase 3: Documentation & Skill-Prose Repoint

- [x] T005 [P] `src/kit/skills/_cli-fab.md` § fab resolve-agent (~line 217): document the `--alias` flag (short alias on `model=`; Claude-Code Agent-tool adapter; default unchanged = full ID; `effort=` line unaffected; empty/non-Claude pass through verbatim) <!-- R6 -->
- [x] T006 [P] `src/kit/skills/_preamble.md` § Harness-adapter boundary (~line 353) + the two-seam model bullet (~line 348): repoint the hand-map sentence ("the orchestrator maps the resolved id to the alias at the dispatch seam") to resolving the model half with `fab resolve-agent <stage> --alias` <!-- R7 -->
- [x] T007 [P] `src/kit/skills/fab-ff.md` (~line 37) and `src/kit/skills/fab-fff.md` (~lines 37, 47, 57): repoint each Agent-tool-dispatch resolve call to `fab resolve-agent <stage> --alias` for the model half <!-- R7 -->
- [x] T008 [P] `src/kit/skills/fab-continue.md` (~lines 19, 52, 161 + dispatch-table rows 56/58/59/61): repoint the one-stage-sequencer note, the sub-agent dispatch contract, the nested-reviewers note, and the table resolve calls to `--alias` <!-- R7 -->
- [x] T009 [P] `src/kit/skills/_pipeline.md` (per-stage-model note + apply/review/hydrate/rework resolve calls): repoint each Agent-tool-dispatch resolve call to `--alias` for the model half (the shared bracket — keep it consistent with the wrappers) <!-- R7 -->
- [x] T010 [P] `docs/specs/stage-models.md` § Skill wiring + § Harness-adapter boundary: repoint the "orchestrator maps id → alias at the seam" prose to describe `--alias` as the deterministic Agent-tool model-half mechanism. Two drift-guarded tables NOT touched (confirmed via diff). <!-- R7 -->
- [x] T011 [P] `docs/specs/skills/SPEC-_preamble.md` (~line 5 description + ~lines 96–97 ASCII box): repoint the "short alias … orchestrator maps id→alias" phrasing to the `--alias` resolver flag <!-- R9 -->

### Phase 4: Verification

- [x] T012 Build + full `go test ./...` under `src/go/fab/` (all packages pass); `ModelAlias` tests, `resolve-agent --alias` command tests, and the drift-guard `TestDocTablesMatchAgentMaps` all pass; locally-built binary exercised (`apply --alias`→`model=opus`, no-flag→`model=claude-opus-4-8`). Tree-wide grep confirms no skill/spec prose left on the stale hand-map instruction; operator launcher path confirmed still resolving WITHOUT `--alias`. <!-- R1 R2 R3 R4 R5 R7 R8 R9 -->

## Execution Order

- T001 blocks T002, T003, T004 (the function must exist first)
- T003, T004 depend on T001/T002 but are independent of each other and of the doc tasks
- T005–T011 are independent docs/prose edits (parallelizable)
- T012 runs last (verification gate)

## Acceptance

### Functional Completeness

- [x] A-001 R1: `ModelAlias` maps all four Claude families by prefix; a dated variant resolves to its family alias
- [x] A-002 R2: `ModelAlias` returns empty for empty input and passes an unmapped/non-Claude ID through verbatim
- [x] A-003 R3: `fab resolve-agent apply --alias` emits `model=opus` with the `effort=` line unaffected
- [x] A-004 R6: `_cli-fab.md` § fab resolve-agent documents the `--alias` flag and its semantics
- [x] A-005 R7: the `_preamble.md` Harness-adapter boundary paragraph and all sibling dispatch sites (fab-ff/fff/continue, _pipeline, stage-models) resolve the model half via `--alias` instead of a hand-map
- [x] A-006 R9: `SPEC-_preamble.md` describes `--alias` as the Agent-tool model-half mechanism

### Behavioral Correctness

- [x] A-007 R4: default (no `--alias`) output is byte-identical to today (`model=claude-opus-4-8\neffort=high\n`) — regression guard test present and passing
- [x] A-008 R8: the operator launcher path still resolves WITHOUT `--alias` (full ID appended to the `claude` CLI)

### Edge Cases & Error Handling

- [x] A-009 R5: an empty-model tier under `--alias` still emits an empty `model=` line (inherit signal preserved)
- [x] A-010 R2: a non-Claude override (`gpt-5`) under `--alias` flows through unchanged (adapter, not validator)

### Scenario Coverage

- [x] A-011 R3 R4: command-level tests cover both `--alias` (alias) and no-flag (full ID) on the same stage

### Code Quality

- [x] A-012 Pattern consistency: `ModelAlias` follows the package's existing exported-helper style; the flag wiring follows the existing cobra command pattern in `resolve_agent.go`
- [x] A-013 No unnecessary duplication: reuses `formatAgentProfile` (no parallel formatter); `ModelAlias` is the single mapping site

### Documentation Accuracy (checklist.extra_categories)

- [x] A-014 Doc prose accurately describes `--alias` (no claim the default changed; effort line explicitly noted as unaffected)

### Cross References (checklist.extra_categories)

- [x] A-015 The drift-guarded tables in `stage-models.md` and the `defaultTiers`/`stageTiers` Go maps are untouched; `TestDocTablesMatchAgentMaps` still passes

## Notes

- Check items as you review: `- [x]`
- The PATH `fab` is the installed 2.3.1 release; the new flag must be exercised via the locally-built
  binary (`go run ./cmd/fab resolve-agent apply --alias` from `src/go/fab/`), not the PATH `fab`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix via a Go-side `--alias` flag on `fab resolve-agent`, replacing #413's prompt-side hand-mapping (carried from intake) | User chose this explicitly; encoding the map in Go removes the live failure mode | S:95 R:80 A:92 D:95 |
| 2 | Certain | Default (no `--alias`) byte-identical to today; CLI/operator path + #413 effort-prompt seam unchanged (carried from intake) | The `claude` CLI accepts full IDs; only the Agent-tool enum rejects them; `--alias` touches only `model=` | S:95 R:85 A:95 D:95 |
| 3 | Certain | Do NOT switch tier defaults to aliases; full IDs stay canonical in `defaultTiers` + drift-guarded tables (carried from intake) | Preserves provider-neutrality + Fable version-pin; avoids a coordinated multi-file edit; keeps the drift-guard unaffected | S:90 R:75 A:90 D:90 |
| 4 | Confident | Mapping is prefix-based so dated variants resolve (carried from intake) | Agent enum is family-level; full IDs carry date/version suffixes; prefix match is the robust mapping | S:75 R:80 A:85 D:80 |
| 5 | Confident | Unmapped/non-Claude under `--alias` passes through verbatim, not an error (carried from intake) | `--alias` is a Claude-Code adapter, not a validator; a non-Claude override still flows; low-risk, reversible | S:65 R:80 A:75 D:70 |
| 6 | Confident | Apply `ModelAlias` to `profile.Model` in the RunE before `formatAgentProfile` (not a format variant) | Keeps the formatter a pure, byte-stable function with its omit-when-empty branches intact; one-line pre-format transform is the cleanest seam | S:80 R:85 A:85 D:80 |
| 7 | Confident | Repoint `_pipeline.md` (the shared bracket) too, beyond the intake's literal file list | fab-ff/fff delegate Steps 1–3 dispatch to the `_pipeline.md` bracket, which carries the same model-half dispatch instruction; the intake directs "sweep ALL siblings — do not leave a stale sibling," so consistency requires it | S:70 R:85 A:80 D:75 |

7 assumptions (3 certain, 4 confident, 0 tentative).
