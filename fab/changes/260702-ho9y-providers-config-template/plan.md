# Plan: Providers Config Template — Three Providers Pre-Filled

**Change**: 260702-ho9y-providers-config-template
**Intake**: `intake.md`

## Requirements

<!-- Requirements derived from intake.md § What Changes. This change is
     documentation/template text only — no resolution or dispatch BEHAVIOR
     changes (intake § Impact "Out of scope"). The "grammar" requirements below
     are about what STRINGS the template ships, verified against live CLI docs
     at apply per intake § What Changes item 3. -->

### Config Reference: Three-Provider Template

#### R1: `fab config reference` ships all three providers with both command fields as text

The generated reference `config.yaml` (`internal/configref`) SHALL present a `providers:` block that contains — **as text** — the two command fields (`session_command`, `dispatch_command`) for all three providers `claude`, `codex`, and `gemini`, so a user configuring a non-claude provider copies and adapts rather than composing command grammar from scratch.

- **GIVEN** a user runs `fab config reference`
- **WHEN** the `providers:` block is rendered
- **THEN** the output contains, as text, `session_command` and `dispatch_command` lines for `claude`, `codex`, and `gemini`

#### R2: Anything whose uncommenting changes default behavior ships commented

Any command whose *uncommenting* would change fab-kit's default runtime behavior SHALL ship commented-out (opt-in convention). Specifically: claude's `dispatch_command` (uncommenting flips claude stages from native Agent-tool dispatch to CLI dispatch) SHALL be commented; the entire `codex` and `gemini` blocks (opt-in providers) SHALL be commented. Claude's `session_command` SHALL appear live (it restates the built-in default — a harmless restatement consistent with how baseline keys show live example values, and the existing reference/scaffold already show it live).

- **GIVEN** the rendered reference `providers:` block
- **WHEN** a reader inspects which lines are live vs. commented
- **THEN** only `claude.session_command` is live; `claude.dispatch_command` and the whole `codex`/`gemini` blocks are commented-out

#### R3: No new built-in providers in Go

The Go `defaultProviders` table SHALL continue to contain `claude` as the only built-in provider. `codex` and `gemini` SHALL exist purely as commented template text in the rendered reference and the scaffold — never as Go-embedded provider defaults.

- **GIVEN** the `internal/agent` `defaultProviders` table
- **WHEN** this change ships
- **THEN** `claude` remains the only built-in provider; no `codex`/`gemini` entry is added to Go

#### R4: Gemini commands carry no `{effort}` placeholder

Gemini's command strings SHALL omit the `{effort}` placeholder entirely — the gemini CLI has no reasoning-effort flag. The empty-effort token-drop rule is not relied upon; the placeholder is simply absent.

- **GIVEN** the gemini template lines
- **WHEN** rendered
- **THEN** neither `session_command` nor `dispatch_command` for gemini contains `{effort}`

#### R5: Verified CLI grammar for each provider's command strings

The command strings SHALL use CLI grammar verified against current CLI docs at apply time (intake § What Changes item 3). The verified strings are:

- **claude**: `session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"'` (built-in default, unchanged); `dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'` (headless; `-p` reads the prompt from stdin — conforms to the `fab dispatch` stdin-delivery contract).
- **codex**: `session_command: 'codex -m {model} -c model_reasoning_effort={effort}'`; `dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'` (`codex exec` reads the prompt from stdin when no prompt argument is given — conforms).
- **gemini**: `session_command: 'gemini -m {model}'` (interactive session; `-m/--model` verified to take a model name); `dispatch_command: 'gemini -m {model}'` (headless via piped stdin — gemini enters non-interactive mode automatically in a non-TTY/piped environment and treats piped stdin as the prompt; **no `-p` flag** — see R6 and Design Decisions).

- **GIVEN** the rendered/scaffolded provider template strings
- **WHEN** compared against current claude/codex/gemini CLI grammar
- **THEN** each string uses a real, current flag grammar for its provider

#### R6: Prompt-delivery conformance is documented per provider; non-conforming grammar carries a caveat

Each provider's `dispatch_command` SHALL conform to the `fab dispatch` prompt-delivery contract (the stage prompt is piped to the command's **stdin** via `<cmd> < {prompt}`, per `internal/dispatch` and `docs/specs/harness-adapters.md`). Where a provider's headless grammar requires care to conform, the template line SHALL carry a comment stating what to adapt.

- **GIVEN** the `fab dispatch` contract pipes the stage prompt to the command's stdin
- **WHEN** each provider's `dispatch_command` is exercised
- **THEN** claude (`-p` from stdin) and codex (`codex exec` from stdin) conform directly; gemini conforms via bare piped stdin (no `-p`), with a comment noting that `-p` takes/​appends an argument and MUST NOT be used for stdin-delivered dispatch

#### R7: Codex model IDs in comment examples use current real IDs

Any codex model ID shown in a comment example SHALL be a current, real codex model ID (cosmetic; verified at apply). No literal model ID is embedded in the codex command *template* itself (the `{model}` placeholder is substituted at resolve time), so this applies only to illustrative comment prose if present.

- **GIVEN** a comment example mentioning a codex model ID
- **WHEN** the reference/scaffold is rendered
- **THEN** the ID is a current real codex model ID (e.g. `gpt-5.3-codex`) or the example omits a literal ID and relies on `{model}`

### Scaffold: Same Template on `fab init`

#### R8: New-project scaffold carries the identical three-provider template

`src/kit/scaffold/fab/project/config.yaml` SHALL gain the same `providers:` template block (claude `session_command` live; claude `dispatch_command` commented; `codex`/`gemini` blocks fully commented) so a fresh `fab init` project sees the three-provider template without running `fab config reference`.

- **GIVEN** a freshly scaffolded project (`fab init`)
- **WHEN** the user opens `fab/project/config.yaml`
- **THEN** the `providers:` block shows the three-provider template matching the reference's presentation (claude live, codex+gemini commented)

### Docs & Tests

#### R9: `_cli-fab.md` schema-coverage line reflects the three-provider template

`src/kit/skills/_cli-fab.md` § `fab config reference` "Full schema coverage" prose SHALL note that the `providers:` block ships as a three-provider (claude/codex/gemini) template. The SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` SHALL be swept for the same class (constitution: CLI-adjacent doc + SPEC-mirror sync).

- **GIVEN** the `_cli-fab.md` `fab config reference` section and its SPEC mirror
- **WHEN** this change ships
- **THEN** both describe the providers block as a three-provider template consistent with the rendered reference

#### R10: `architecture.md` config example extended to restate the three-provider block

`docs/specs/architecture.md`'s example `config.yaml` restates the `providers:` block (currently claude live + codex commented). Because it restates the block, it SHALL be extended to include the gemini commented block (and claude's commented `dispatch_command`) to stay consistent — per intake § Impact ("architecture.md config example extended only if it restates the providers block").

- **GIVEN** `architecture.md`'s example restates the `providers:` block
- **WHEN** this change ships
- **THEN** the example includes the commented codex AND gemini blocks (and claude's commented `dispatch_command`), matching the reference

#### R11: configref tests / golden output updated

The `internal/configref` / `cmd/fab` config tests SHALL be updated so the round-trip, coverage, byte-stability, providers-documentation, and placeholder guards continue to pass, and a guard SHALL assert the three-provider template is present (codex AND gemini documented, gemini carries no `{effort}`).

- **GIVEN** the config reference test suite
- **WHEN** `go test ./cmd/fab/...` runs
- **THEN** all tests pass and the three-provider presence is guarded

### Non-Goals

- New built-in provider defaults in Go (`codex`/`gemini` stay commented template text only) — R3.
- Any change to resolution/dispatch behavior (`fab resolve-agent`, `fab dispatch`, `spawn.WithProfile`) — template text only.
- Validating provider commands (the standing no-validation / provider-neutrality contract is unchanged; a misauthored template still fails at dispatch time).

### Design Decisions

1. **Gemini `dispatch_command` is `gemini -m {model}` with NO `-p` flag** — *Why*: `fab dispatch` pipes the stage prompt to the command's **stdin** (`<cmd> < {prompt}`, `internal/dispatch`). The gemini CLI's `-p/--prompt` flag takes prompt text as its *argument value* and, per the live docs, is "appended to stdin input if provided" — a bare `-p` with no argument is not a valid stdin-delivery form, and `-p "text"` would append AFTER the piped prompt. Gemini enters non-interactive mode automatically in a non-TTY/piped environment and treats piped stdin as the prompt (`echo "…" | gemini`), which is exactly what `fab dispatch` provides. So the conformant dispatch grammar omits `-p`. This CORRECTS the intake's tentative `gemini -m {model} -p`. — *Rejected*: `gemini -m {model} -p` (bare `-p` is ambiguous/invalid for stdin delivery); dropping gemini to session-only (unnecessary — bare stdin conforms cleanly); a `{prompt}` seam (out of scope — no dispatch-behavior change this release).

2. **Verified against live CLI docs at apply** — *Why*: intake § What Changes item 3 makes this an explicit apply obligation. gemini `-m`/`-p` grammar and non-TTY stdin behavior confirmed against the official gemini CLI headless-mode + cli-reference docs (July 2026); codex `codex exec` + `-c model_reasoning_effort=` confirmed against OpenAI Codex CLI docs; current codex model IDs (`gpt-5.3-codex`, etc.) noted. — *Rejected*: shipping the intake's best-guess strings unverified (the intake explicitly deferred verification to apply).

3. **Claude `session_command` ships live in BOTH reference and scaffold** — *Why*: it restates the built-in default (harmless), the existing reference and scaffold already show it live, and consistency across the two surfaces is worth more than scaffold minimalism (intake Open Question 1). — *Rejected*: fully commenting claude in the scaffold (breaks parity with the reference and the existing live-claude convention; a live example key is the established scaffold style).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite the `providers:` block in `referenceTemplate` (`src/go/fab/internal/configref/configref.go`) to the three-provider template: claude `session_command` live (`{{ .SessionCommand }}`), claude `dispatch_command` commented, and fully-commented `codex` and `gemini` blocks; gemini carries no `{effort}`; update the leading providers comment to describe the three-provider template and the gemini bare-stdin (no `-p`) dispatch caveat <!-- R1 R2 R3 R4 R5 R6 R7 --> <!-- rework: prose continuation lines are interleaved INSIDE the commented codex/gemini YAML blocks, so whole-block uncommenting yields invalid YAML — move prose above the block or onto the command lines as trailing comments, matching the stage_hooks/branch_prefix commented-block pattern (review cycle 1, should-fix) -->
- [x] T002 Add the identical three-provider `providers:` template block to `src/kit/scaffold/fab/project/config.yaml` (claude live, claude `dispatch_command` + codex + gemini commented), matching the reference presentation <!-- R8 --> <!-- rework: same prose-interleaving fix as T001 in the scaffold's commented codex/gemini blocks — whole-block uncomment must yield valid YAML (review cycle 1, should-fix) -->

### Phase 3: Integration & Docs

- [x] T003 Extend the `providers:` example in `docs/specs/architecture.md` to restate the codex AND gemini commented blocks (plus claude's commented `dispatch_command`), matching the reference. Also swept `docs/specs/stage-models.md` § "Config schema" — its illustrative `providers:` example restated the same codex-only shape, so added the gemini commented block + claude commented `dispatch_command` there too (cross_references consistency; the stage-models tier-drift guard `TestDocTablesMatchAgentMaps` still passes — it parses the tier table, not the providers example) <!-- R10 -->
- [x] T004 Update `src/kit/skills/_cli-fab.md` § `fab config reference` "Full schema coverage" prose to note the three-provider (claude/codex/gemini) template, and sweep the SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` for the same class (mirror is a summary-table row whose "fully-commented reference config.yaml (all available options)" contract remains accurate — no edit needed) <!-- R9 --> <!-- rework: the clause "the five `agent.tiers` are shown live" was displaced into the gemini parenthetical at _cli-fab.md:317 (non sequitur) — reattach it to the "Baseline keys appear live with example values" sentence (review cycle 1, should-fix, flagged by both reviewers) -->

### Phase 4: Tests

- [x] T005 Update/extend the config reference tests in `src/go/fab/cmd/fab/config_test.go`: keep round-trip / byte-stable / coverage / providers-documentation / placeholder guards green, and add a guard asserting the three-provider template is present (codex AND gemini documented; gemini carries no `{effort}`). Run `go test ./cmd/fab/... ./internal/configref/... ./internal/agent/...` <!-- R11 --> <!-- rework: (1) MUST-FIX cmd/fab/config_test.go:219 is not gofmt-clean (CI hard gate fails on gofmt -l output) — run gofmt -w; (2) add parse-side assertions to TestConfigReferenceRoundTrips: claude DispatchCommand == "" and GetProvider("codex")/GetProvider("gemini") return !ok, guarding against accidental future un-commenting (review cycle 1) -->

## Execution Order

- T001 is the primary source change; T002/T003/T004 (scaffold + docs mirrors) can follow in any order.
- T005 (tests) runs after T001–T002 (it asserts on rendered reference + scaffold content).

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab config reference` output contains, as text, `session_command` and `dispatch_command` lines for claude, codex, and gemini
- [x] A-002 R8: the scaffold `config.yaml` `providers:` block shows the same three-provider template (claude live, codex+gemini commented)
- [x] A-003 R3: `internal/agent` `defaultProviders` still contains only `claude` (no codex/gemini Go default added)
- [x] A-004 R9: `_cli-fab.md` § `fab config reference` schema-coverage prose describes a three-provider template, and `SPEC-_cli-fab.md` is consistent
- [x] A-005 R10: `architecture.md`'s example `providers:` block restates the codex AND gemini commented blocks

### Behavioral Correctness

- [x] A-006 R2: only `claude.session_command` is live in the rendered reference and scaffold; `claude.dispatch_command`, `codex`, and `gemini` blocks are commented-out
- [x] A-007 R4: gemini's command strings contain no `{effort}` placeholder
- [x] A-008 R6: gemini's `dispatch_command` is `gemini -m {model}` (no `-p`) and carries a comment explaining the bare-stdin dispatch conformance; claude (`-p` from stdin) and codex (`codex exec` from stdin) conform directly

### Scenario Coverage

- [x] A-009 R5: each provider's command strings use current, real CLI grammar (claude/codex/gemini) as verified at apply against live docs
- [x] A-010 R7: any codex model ID in a comment example is a current real ID (or the example relies on `{model}` with no literal ID)
- [x] A-011 R11: the config reference test suite passes and guards the three-provider template presence (`go test ./cmd/fab/... ./internal/configref/...`)

### Edge Cases & Error Handling

- [x] A-012 R1: the rendered reference still round-trips into `Config` (the added commented blocks do not break YAML parse) and `fab config reference` still exits 0
- [x] A-013 R11: `fab config reference` output remains byte-stable across renders (no map range-iteration introduced)

### Code Quality

- [x] A-014 Pattern consistency: new template/comment text follows the surrounding configref template conventions (comment style, quoting, injection points) and the scaffold's existing comment style
- [x] A-015 No unnecessary duplication: the three-provider block is authored once per surface; default values still injected from constants where they exist (`{{ .SessionCommand }}`), no new drift-prone hand-copied constant
- [x] A-016 Canonical source only: edits are under `src/kit/...`, `src/go/...`, and `docs/...` — no `.claude/skills/` deployed copy is touched
- [x] A-017 SPEC-mirror sweep: the `_cli-fab.md` edit carries its `SPEC-_cli-fab.md` mirror update (constitution Additional Constraints)
- [x] A-018 documentation_accuracy: the shipped command strings match verified live CLI grammar; no stale/guessed flag remains
- [x] A-019 cross_references: the reference, scaffold, and `architecture.md` example present a mutually consistent three-provider block

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Memory updates (`_shared/configuration.md`, `runtime/providers-and-tiers.md`) are hydrate's responsibility, not apply's.

## Deletion Candidates

- `src/go/fab/cmd/fab/config_test.go:176` (`TestConfigReferenceMentionsCommandPlaceholders`) — its two assertions (`{model}`/`{effort}` appear in the rendered reference) are now logically subsumed by `TestConfigReferenceDocumentsThreeProviderTemplate`, whose asserted codex command string contains both placeholders; keep only if the independent "placeholders are documented" contract should survive a future template reshape.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Gemini `dispatch_command` corrected to `gemini -m {model}` (NO `-p`), overriding the intake's tentative `gemini -m {model} -p` | Verified at apply against live gemini CLI docs: `-p` takes prompt text as an arg and is "appended to stdin input"; `fab dispatch` pipes the prompt to stdin, and gemini auto-enters headless mode in a non-TTY pipe (`echo … | gemini`). Bare `-p` is not a valid stdin-delivery form. Template text, trivially reversible | S:70 R:85 A:75 D:70 |
| 2 | Certain | Codex grammar `codex -m {model} -c model_reasoning_effort={effort}` (session) / `codex exec …` (dispatch); `codex exec` reads prompt from stdin | Established in tykw design + existing configref example; confirmed against current Codex CLI docs (`-c model_reasoning_effort=`, `codex exec` stdin). Conforms to the dispatch stdin contract | S:85 R:85 A:90 D:90 |
| 3 | Certain | Claude `dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'` ships commented; `session_command` live | Intake exact content; `-p` reads stdin (conforms); commented so uncommenting is the opt-in that flips native→CLI dispatch | S:80 R:85 A:90 D:90 |
| 4 | Confident | Claude `session_command` shown live in BOTH reference and scaffold; codex+gemini fully commented (intake Open Question 1) | Existing reference and scaffold already show claude live; live example key is the established convention; parity across the two surfaces beats scaffold minimalism. Easily changed | S:65 R:85 A:80 D:75 |
| 5 | Confident | `architecture.md` example extended to add the commented gemini block (and claude's commented `dispatch_command`) alongside the existing codex block | Intake § Impact conditions this on the example restating the block — it does (codex already shown). Keeping the three surfaces consistent is the cross_references discipline | S:70 R:85 A:80 D:80 |
| 6 | Confident | Current codex model IDs referenced as `gpt-5.3-codex` family; codex template relies on `{model}` (no literal ID embedded) | Verified current codex model IDs at apply; the command template uses the `{model}` placeholder so no literal ID ships in the template itself — the intake's `gpt-5-codex` example is superseded by the current family only in prose if used | S:60 R:90 A:70 D:75 |
| 7 | Certain | No new built-in providers in Go; no resolution/dispatch behavior change; no command validation | Intake Non-Goals + standing no-validation/provider-neutrality contract; this change is template text on three doc/scaffold surfaces plus tests | S:90 R:90 A:95 D:90 |

7 assumptions (3 certain, 4 confident, 0 tentative).
