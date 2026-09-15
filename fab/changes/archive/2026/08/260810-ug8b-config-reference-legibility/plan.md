# Plan: Config Reference Legibility — Flatten Provider Blocks + Reorder Dispatch Above Agent

**Change**: 260810-ug8b-config-reference-legibility
**Intake**: `intake.md`

## Requirements

### Config Reference: Provider Block Presentation

#### R1: Uniform single-hash provider rendering
The providers segment (`providersSegment`/`providersYAML`/`profilesLines`/`providersShortSegment` in `src/go/fab/internal/configref/configref.go`) MUST render all four built-in provider blocks (claude, codex, agy, kimi) uniformly LIVE in the segment's canonical form — same indentation family as claude's block today — with no pre-commented content lines. `profilesLines` MUST emit plain YAML lines (no `# `-prefixing) for non-claude fills.

- **GIVEN** the field registry builds the providers segment
- **WHEN** `Render()` or a file-bound renderer (`CommentOutSegment` over the short form) produces output
- **THEN** every provider line in the rendered fence / `init --system --print` carries exactly ONE `#`
- **AND** in the bare `fab config explain` long form, all four provider blocks appear live at the same indentation

#### R2: Pins-on-hoist warning prose
The prose immediately above the provider blocks in `providersSegment` MUST carry a prominent warning with two required elements: (a) uncommenting a whole provider block makes its fills a live override that PINS the shown values (kit releases refresh the built-in fills, but a pinned copy shadows every refresh), and (b) the preferred action is overriding a single field (`providers.<name>.profiles.<role>.model`).

- **GIVEN** a reader of `fab config explain` or the managed fence
- **WHEN** they consider hoisting a provider block
- **THEN** the warning's two elements are stated in short prose near the blocks, not buried mid-essay

#### R3: Strip-to-valid-YAML guarantee survives
Stripping the leading `# ` from a whole fence providers block MUST still yield valid YAML containing all four built-ins (the `YAMLSingleQuoted` escaping of the nested-shell agy/kimi commands remains load-bearing).

- **GIVEN** the rendered fence with the flattened providers block
- **WHEN** a reader strips one `# ` layer from every line of the block
- **THEN** the result parses as a valid `providers:` mapping with all four built-in providers

### Config Reference: Registry Row Order

#### R4: dispatch rows above agent rows
The three `dispatch.*` registry rows (`dispatch.mode`, `dispatch.column_width`, `dispatch.reap_done`) MUST be ordered immediately BEFORE the `agent.session` row in the `Fields()` table. The `providers` row MUST remain adjacent to (after) the `agent` rows.

- **GIVEN** the `Fields()` registry
- **WHEN** `Render()`, `RenderJSON()`, the managed fence, or `fab config init --system` walk it
- **THEN** the rendered order is `… consolidate → dispatch → agent → providers → stage_hooks …`
- **AND** no key semantics, defaults, or segment contents change (order only)

#### R5: Directional prose corrected for the new order
Prose cross-references inside `providersSegment`/`dispatchSegment`/`agentSegment` that use directional words ("above"/"below") about the agent knobs or dispatch block MUST read correctly for the new dispatch → agent → providers order.

- **GIVEN** the reordered registry
- **WHEN** a reader follows a cross-reference like "agent.session / agent.workers, above"
- **THEN** the direction named matches the actual rendered position

### Docs & Tests: Sweep and Test Integrity

#### R6: Old-presentation prose swept repo-wide
Every claim describing the old presentation MUST be updated: the `configref.go` function-comment paragraph ("Presentation (260805-j3cm)…", "DELIBERATELY-COMMENTED content lines"), the segment prose line "Every non-claude block below is commented because it merely restates a built-in default…", the per-provider comment-layer wording in the `YAMLSingleQuoted` doc comment, plus every occurrence in `docs/specs/config.md`, `docs/memory/` files describing the presentation, and `src/kit/` (grep `# #`, "commented reference", "DELIBERATELY-COMMENTED").

- **GIVEN** the flattened rendering ships
- **WHEN** the repo is grepped for the old claims
- **THEN** no prose still asserts non-claude blocks render commented or that comment depth encodes semantics

#### R7: Tests and fixtures updated, suites green
Byte-stable rendering assertions in `src/go/fab/cmd/fab/` (config tests) and `src/go/fab/internal/configupgrade/` (fence-fixture tests) MUST be regenerated to the new single-hash, reordered form, and `go test` MUST pass for `internal/configref`, `internal/configupgrade`, and `cmd/fab`.

- **GIVEN** the implementation change
- **WHEN** `go test ./internal/configref/... ./internal/configupgrade/... ./cmd/fab/...` runs from `src/go/fab`
- **THEN** all tests pass with assertions matching the new rendering

### Non-Goals

- No migration ships — the managed fence self-heals on the next `fab config upgrade`; no key is renamed or restructured (presence=intent untouched).
- The `providers` row does not move — only `dispatch.*` moves above `agent.*`.
- No skill files change unless one restates the presentation (grep-verified in R6).

### Design Decisions

#### Flatten over keep-with-legend
**Decision**: Render all four provider blocks uniformly live; carry the pinning warning as prose.
**Why**: The doubled `# #` marker's semantics were illegible to the project's own author; a legend preserves the visual irregularity. Uniform single-hash is decodable at a glance.
**Rejected**: Keep two-tier comment depth with a legend line — the user weighed both and chose flatten, accepting that a whole-block hoist now yields four live provider overrides (mitigated by the R2 warning).
*Introduced by*: 260810-ug8b-config-reference-legibility

## Tasks

### Phase 2: Core Implementation

- [x] T001 Flatten `providersYAML` + `profilesLines` in `src/go/fab/internal/configref/configref.go`: emit codex/agy/kimi blocks live at claude's indentation, drop `# `-prefixing on fill lines, update the kimi TrimRight comment; update `providersSegment` prose (remove "Every non-claude block below is commented…", add the R2 pins-on-hoist warning above the blocks); rewrite the `providersSegment` function-comment paragraph (drop "DELIBERATELY-COMMENTED", describe the flattened presentation) and the `YAMLSingleQuoted` doc comment's per-provider comment-layer wording. <!-- R1, R2, R3 -->
- [x] T002 Move the `dispatch.mode` / `dispatch.column_width` / `dispatch.reap_done` registry rows in `src/go/fab/internal/configref/configref.go` `Fields()` to immediately before the `agent.session` row; keep the `providers` row after the `agent` rows. <!-- R4 -->
- [x] T003 Fix directional prose in `providersSegment`/`dispatchSegment`/`agentSegment` for the new order (verified: "agent.session / agent.workers, above" in `providersSegment` stays correct — agent rows remain immediately above providers; no other directional reference affected). <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T004 Update `src/go/fab/cmd/fab` config tests to the flattened rendering: `config_test.go` (`TestConfigReferenceRoundTrips` inverted to live-block assertions, `TestConfigReferenceDocumentsBuiltInProviders` block-shape assertions, `TestConfigReferenceUncommentedProviderBlocksParse` reshaped to `TestConfigReferenceProviderBlocksParse` parsing the live blocks — `uncommentProviderBlock` deleted, fill-line expectations). `config_show_init_test.go` needed no changes (scaffold stays fully commented via `CommentOutSegment`). <!-- R3, R7 -->
- [x] T005 Update `src/go/fab/internal/configupgrade` references to the doubled-comment form (`configupgrade_test.go` CommentOutSegment test comments, block-strip test comment; `configupgrade.go` `CommentOutSegment` doc comment — now cite the agent block's `# profiles:` example lines). Also reordered `internal/configscope/configscope.go` `dottedKeys` (+ its test) to match the new registry order (parity lint). <!-- R7 -->
- [x] T006 Run `go test ./...` scoped to `internal/configref`, `internal/configupgrade`, `internal/agent`, and `cmd/fab` from `src/go/fab`; fix every failure. <!-- R7 -->

### Phase 4: Polish

- [x] T007 Sweep docs: grep repo-wide for `# #`, "commented reference", "DELIBERATELY-COMMENTED", "Every non-claude block", directional "above"/"below" near the moved blocks; update `docs/specs/config.md`, `docs/memory/_shared/configuration.md`, any other `docs/memory/` file describing the presentation, and any `src/kit/` occurrence. (Updated: `docs/specs/config.md`, `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `docs/memory/_shared/configuration.md`, `docs/memory/runtime/providers-and-profiles.md`. `src/kit/` hits were historical migration records only — left untouched. No order claims on agent-vs-dispatch found in docs.) <!-- R6 -->
- [x] T008 Run `gofmt -l` over every touched `.go` file and fix any differences. (`gofmt -l .` over the whole module prints nothing.) <!-- R7 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `Render()` shows all four provider blocks live at uniform indentation; `CommentOutSegment` over the providers short segment yields exactly one `#` per line (no `# #` anywhere in a rendered fence or `init --system --print`). (verified: `config explain` shows all four live at one indentation; `init --system --print` carries exactly one `#` per provider line; grep finds no `# #` in rendered output)
- [x] A-002 R2: The prose above the provider blocks states both required warning elements: pins-on-hoist (a hoisted copy shadows kit-release fill refreshes) and prefer per-field override (`providers.<name>.profiles.<role>.model`). (verified: `# NOTE:` lines immediately above the `providers:` payload carry both elements)
- [x] A-003 R4: `Render()` (and fence / `init --system`) order is `consolidate → dispatch → agent → providers → stage_hooks`; registry row order matches. (verified in rendered output and in the `Fields()` diff; `configscope.dottedKeys` reordered in parity)
- [x] A-004 R6: Repo-wide grep finds no surviving claim that non-claude provider blocks render commented, no `# #` presentation references, no stale directional "above"/"below" near the moved blocks. (verified: remaining hits are historical archives/migrations/old change folders or generic fence-mechanism prose that stays accurate)
- [x] A-005 R7: `go test` passes for `internal/configref`, `internal/configupgrade`, `cmd/fab` (and `internal/agent`). (verified: all green; the only failures — `TestDispatchDeliver_RefusesAMidStageWorker`, `TestDispatchReady_RefusesAMidStageWorker` — fail identically on base 33a9aed, environmental tmux flakes unrelated to this change)

### Behavioral Correctness

- [x] A-006 R3: Stripping one `# ` layer from the fence providers block yields valid YAML parsing to all four built-ins; the agy/kimi nested-shell single-quote doubling is preserved (asserted by test). (verified independently: stripped block parsed with yaml.v3 → all four providers; `TestCommentOutSegment_BlockStripRestoresSegment` green)

### Scenario Coverage

- [x] A-007 R4: Keyed surfaces resolve after the reorder: `fab config explain dispatch.mode` and `fab config explain agent.session` return their owning segments; `ResolveKey("dispatch.column_width")` still resolves to the shared dispatch segment. (verified: all three keyed explains exit 0, `providers` too)
- [x] A-008 R1: Edge renderings preserved — kimi still renders no `profiles:` key (grammar-only built-in), agy/kimi blocks still open on `headless_command` with no `interactive_command`, sparse fill maps stay sparse. (verified in rendered output; asserted by `TestConfigReferenceDocumentsBuiltInProviders`)

### Code Quality

- [x] A-009 Pattern consistency: New/edited prose and Go code follow the surrounding file's conventions (string-concat segments, doc-comment density).
- [x] A-010 No unnecessary duplication: `providersYAML` remains the single payload source shared by long and short segments; no second copy of provider grammar introduced.
- [x] A-011 Sibling & mirror sweep: `docs/specs/config.md` and the memory file documenting the presentation updated in the same change; `gofmt -l` clean over touched files. (verified: `docs/specs/config.md`, `stage-models.md`, `architecture.md`, `docs/memory/_shared/configuration.md`, `runtime/providers-and-profiles.md` all updated; `gofmt -l .` prints nothing)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- **Collision note** (from intake): change `260810-ki9v-kimi-pane-enablement` adds kimi's `interactive_command` to the built-in defaults — whichever lands second takes a mechanical fixture rebase.

## Deletion Candidates

- `src/go/fab/cmd/fab/config_test.go` `uncommentProviderBlock` (test helper) — made redundant by the flattened rendering (no commented provider blocks left to extract); already removed by this change, no surviving references.
- None remaining — beyond the helper above (deleted in-change), no existing file, function, branch, or config was made redundant or left unused by this change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Exact warning wording and placement: a short `# NOTE:` pair of lines immediately above the `providers:` payload in `providersSegment` | Intake fixes the two required elements (pins-on-hoist, prefer per-field override) and leaves phrasing to apply | S:55 R:90 A:70 D:60 |
| 2 | Certain | kimi keeps its no-fills rendering; the whole-block TrimRight in `providersYAML` stays (only indentation/comments change) | kimi ships no fills by design; the trim rationale is unchanged by flattening | S:90 R:85 A:90 D:90 |
| 3 | Confident | `TestConfigReferenceUncommentedProviderBlocksParse` is reshaped to parse the now-LIVE provider blocks out of `Render()` directly (same valid-YAML guarantee, no comment strip needed); the strip-one-layer guarantee is exercised against `CommentOutSegment` output instead | The segment no longer carries commented provider blocks, so the old extraction helper has nothing to find; the R3 guarantee moves to the fence form | S:65 R:80 A:75 D:65 |
| 4 | Confident | `docs/memory/` presentation updates ship in this apply run (per the dispatch prompt's hard constraint), not deferred to hydrate | The apply prompt explicitly orders the memory sweep; hydrate will still do its own merge pass | S:70 R:85 A:80 D:65 |

4 assumptions (1 certain, 3 confident, 0 tentative).
