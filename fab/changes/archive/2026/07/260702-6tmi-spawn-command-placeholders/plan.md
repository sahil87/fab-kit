# Plan: Provider-forgiving spawn_command via `{model}`/`{effort}` placeholders

**Change**: 260702-6tmi-spawn-command-placeholders
**Intake**: `intake.md`

## Requirements

### Spawn: Template Mode in `WithProfile`

#### R1: Placeholder detection selects template mode vs. append fallback
`spawn.WithProfile` (`src/go/fab/internal/spawn/spawn.go`) SHALL detect whether `spawnCmd` contains the literal placeholder `{model}` or `{effort}`. When either placeholder is present, the command is a **template** and MUST be resolved by substitution (R2/R3). When **no** placeholder is present, `WithProfile` MUST fall back to today's append behavior byte-for-byte.

- **GIVEN** a `spawnCmd` containing no `{model}`/`{effort}` placeholder
- **WHEN** `WithProfile(spawnCmd, model, effort)` is called
- **THEN** the result is byte-identical to the current append implementation (` --model <model>` then ` --effort <effort>`, each omitted when empty)
- **AND** the `DefaultSpawnCommand` fallback and existing Claude configs are unaffected

#### R2: Template substitution replaces every placeholder occurrence (all-or-nothing)
When template mode is active, `WithProfile` SHALL substitute **every** occurrence of `{model}` with the resolved model value and **every** occurrence of `{effort}` with the resolved effort value. Template mode is **all-or-nothing**: the presence of any placeholder disables appending entirely — a value whose placeholder is absent is simply not injected (never appended).

- **GIVEN** a template `codex -m {model} -c model_reasoning_effort={effort}` with model `gpt-5` and effort `high`
- **WHEN** `WithProfile` is called
- **THEN** the result is `codex -m gpt-5 -c model_reasoning_effort=high`
- **AND GIVEN** a template `codex -m {model}` with a non-empty effort
- **WHEN** `WithProfile` is called
- **THEN** the effort is NOT appended (only `{model}` is substituted)
- **AND GIVEN** a template containing `{model}` twice
- **THEN** both occurrences are substituted with the same value

#### R3: Empty-value rule drops the placeholder token and a preceding flag token
When template mode substitutes an **empty** value (the documented "inherit/omit" signal, `_preamble.md` § Per-Stage Model Resolution), `WithProfile` SHALL drop the whitespace-delimited token containing that placeholder, and SHALL ALSO drop the immediately preceding whitespace-delimited token when that preceding token begins with `-`.

- **GIVEN** a template `codex -m {model}` and an empty model
- **WHEN** `WithProfile` is called
- **THEN** both `-m` and `{model}` tokens are dropped, leaving `codex`
- **AND GIVEN** `--model={model}` with an empty model
- **THEN** the single token `--model={model}` is dropped (no preceding `-`-token to drop)
- **AND GIVEN** `-c model_reasoning_effort={effort}` with an empty effort
- **THEN** the `model_reasoning_effort={effort}` token and the preceding `-c` are dropped
- **AND GIVEN** a non-empty value
- **THEN** no token is dropped — the value is substituted in place

### CLI: Raw-Output Leak Prevention (`fab spawn-command`, `fab batch`)

#### R4: `fab spawn-command` resolves templates with an empty profile before printing
`fab spawn-command` (`src/go/fab/cmd/fab/spawn_command.go`) SHALL apply the template resolution (R2/R3) with an **empty profile** (empty model and empty effort) to the configured command before printing. A templated command therefore degrades to a clean invocation (placeholders and their flag tokens stripped per R3); a non-templated command MUST print verbatim as today.

- **GIVEN** a repo whose `agent.spawn_command` is `codex -m {model} -c model_reasoning_effort={effort}`
- **WHEN** `fab spawn-command --repo <repo>` runs
- **THEN** stdout is `codex` (both placeholder tokens and their preceding flags stripped)
- **AND GIVEN** a non-templated `agent.spawn_command` (e.g. `claude --dangerously-skip-permissions`)
- **THEN** stdout is that command verbatim (byte-for-byte as today)

#### R6: All raw spawn-command consumers strip placeholders before shell interpolation
<!-- rework cycle 1: review must-fix — R4's scope named only `fab spawn-command`, but `fab batch new`/`fab batch switch` also interpolate raw `spawn.Command` output into tmux shell commands, reopening the leak Design Decision 2 rejected -->
Every fab code path that interpolates the configured `agent.spawn_command` into a shell command **without** a resolved profile SHALL first apply the empty-profile template resolution (R3 semantics). Concretely: `fab batch new` (`src/go/fab/cmd/fab/batch_new.go`) and `fab batch switch` (`src/go/fab/cmd/fab/batch_switch.go`) MUST NOT emit literal `{model}`/`{effort}` braces into their tmux `new-window` commands. A named helper `spawn.StripPlaceholders(cmd)` (thin wrapper over `WithProfile(cmd, "", "")`) SHOULD state this intent at each raw-consumer site, including the existing `spawn_command.go` site.

- **GIVEN** a repo whose `agent.spawn_command` is `codex -m {model} -c model_reasoning_effort={effort}`
- **WHEN** `fab batch new` or `fab batch switch` composes its tmux window spawn command
- **THEN** the interpolated spawn command is `codex` (placeholders and flag tokens stripped) — no literal braces reach tmux
- **AND GIVEN** a non-templated `agent.spawn_command`
- **THEN** the interpolated spawn command is unchanged from today (byte-for-byte)

### Docs: Prose Sweep (mirror class)

#### R5: Placeholder semantics are documented across the mirror class
The behavior change SHALL be reflected in every member of the documentation mirror class in the same change: the `_preamble.md` operator-launcher exception note, `fab-operator.md` Key Properties row, `_cli-fab.md` `fab spawn-command` entry, their SPEC mirrors, `docs/specs/stage-models.md`, and the scaffold `config.yaml` `spawn_command` comment. A repo-wide grep for stale "appends `--model`" / "WithProfile" / "spawn_command" claims SHALL confirm no member is left describing the old always-append behavior as unconditional.

- **GIVEN** the append behavior is now conditional (template mode vs. append fallback)
- **WHEN** the prose sweep completes
- **THEN** each member of the class describes the template/append duality (or the operator launcher's unchanged behavior) accurately
- **AND** the scaffold `config.yaml` `spawn_command` comment notes the optional `{model}`/`{effort}` placeholders

### Non-Goals

- **Per-tier `spawn_command`** (`agent.tiers.<tier>.spawn_command`) — deferred to change 3.
- **CLI dispatch adapter / headless stage dispatch** — deferred to change 3, spec-first.
- **Provider validation** — fab still never validates model/effort; placeholders only relocate where the strings land.
- **New CLI commands or flags** — none added.
- **Config schema / migration** — placeholders are plain string content; no schema change, no migration, no `.status.yaml` change.
- **Memory (`docs/memory/`) edits** — owned by the hydrate stage, not apply.

### Design Decisions

1. **Token-drop empty-value rule over raw empty substitution**: on an empty value in template mode, drop the placeholder's whitespace-delimited token plus a preceding `-`-prefixed token — *Why*: a deterministic rule covering all four common flag shapes (`-m {model}`, `--model {model}`, `--model={model}`, `-c model_reasoning_effort={effort}`) without a templating engine — *Rejected*: raw empty substitution (leaves dangling `-m` / `model_reasoning_effort=`); a full templating language (heavy, YAGNI).
2. **`fab spawn-command` resolves with an empty profile**: — *Why*: the `/fab-operator` skill spawns workers from raw `fab spawn-command` output with no profile injection, so a templated command would leak literal `{...}` braces; empty-profile resolution degrades a template to a clean invocation — *Rejected*: print verbatim + document the hazard (leaves a foot-gun). *(Rework cycle 1 extended this decision to ALL raw `spawn.Command` output consumers — the batch commands had the same foot-gun; see R6.)*
4. **Non-empty substitution preserves the raw string** *(rework cycle 1, review should-fix)*: when every substituted value is non-empty, template resolution uses plain string replacement on the raw command — tokenization (which collapses whitespace runs) applies only on the empty-value drop path — *Why*: non-empty substitution needs no token surgery; preserving author spacing is strictly safer — *Rejected*: tokenizing unconditionally (collapses multi-space/tab runs for no benefit). The token-drop grammar remains quote-blind and limited to the four documented flag shapes; a doc comment on `resolveTemplate` marks quoted or valueless-flag-adjacent placeholders as unsupported.
3. **Operator launcher (`operator.go`) call shape unchanged**: `WithProfile` remains the single seam and its signature is unchanged, so `operator.go` (the one production consumer) keeps calling `spawn.WithProfile(spawnCmd, model, effort)` with a resolved doing-tier profile — a non-templated Claude `spawn_command` still takes the append path — *Why*: the seam already exists; template mode is additive — *Rejected*: threading template awareness into the caller (needless coupling).

## Tasks

### Phase 1: Core Implementation

- [x] T001 Implement template mode in `spawn.WithProfile` (`src/go/fab/internal/spawn/spawn.go`): detect `{model}`/`{effort}`; when present, substitute every occurrence and apply the empty-value token-drop rule (drop placeholder token + preceding `-`-token); when absent, keep today's append path byte-for-byte. Extract a small tokenize/substitute helper if it keeps `WithProfile` focused. <!-- R1 R2 R3 -->
- [x] T002 Update `fab spawn-command` (`src/go/fab/cmd/fab/spawn_command.go`) to pass the resolved command through the template resolution with an empty profile before printing (non-templated commands unchanged). <!-- R4 -->

### Phase 2: Tests

- [x] T003 [P] Extend `spawn_test.go` (`src/go/fab/internal/spawn/spawn_test.go`) with table-driven cases: existing no-placeholder append cases stay green; both placeholders substituted; single placeholder (other half NOT appended); empty model / empty effort / both empty under each token shape (`-m {model}`, `--model {model}`, `--model={model}`, `-c model_reasoning_effort={effort}`); multiple occurrences of one placeholder; placeholder embedded mid-word. <!-- R1 R2 R3 -->
- [x] T004 [P] Extend `spawn_command_test.go` (`src/go/fab/cmd/fab/spawn_command_test.go`) with a templated-config case asserting the placeholder/flag stripping on print, plus a non-templated case asserting verbatim output. <!-- R4 -->

### Phase 3: Docs Prose Sweep (mirror class)

- [x] T005 Update `src/kit/skills/_preamble.md` operator-launcher exception note (§ Per-Stage Model Resolution → Harness-adapter boundary) to reflect that the launcher's non-templated Claude `spawn_command` still takes the append path, while a templated `spawn_command` is now resolved by substitution. <!-- R5 -->
- [x] T006 [P] Update `src/kit/skills/fab-operator.md` Key Properties "Coordinating-agent model" row (~line 703) to note append-vs-template behavior of `spawn.WithProfile`. <!-- R5 -->
- [x] T007 [P] Update `src/kit/skills/_cli-fab.md`: the `fab spawn-command` entry (~line 775) to document empty-profile template resolution before print; the `fab operator` doing-tier paragraph (~line 737) to note the append/template duality of `WithProfile`. <!-- R5 -->
- [x] T008 [P] Update `docs/specs/skills/SPEC-fab-operator.md` (Startup §2 and Key Properties row) and `docs/specs/stage-models.md` (§ Skill wiring / § Harness-adapter boundary operator-launcher note, ~line 246) to reflect the template/append duality. <!-- R5 -->
- [x] T009 [P] Update `src/kit/scaffold/fab/project/config.yaml` `spawn_command` comment (~line 40) to add one line noting the optional `{model}`/`{effort}` placeholders (template mode vs. append fallback). <!-- R5 -->
- [x] T010 Re-sweep: grep repo-wide (`src/kit/`, `docs/specs/`, `src/kit/scaffold/`) for "appends `--model`", "WithProfile", and unconditional-append `spawn_command` claims; update any stale occurrence not already covered by T005–T009. <!-- R5 -->

### Phase 4: Rework Cycle 1 (review findings)

- [x] T011 Add `spawn.StripPlaceholders(cmd string) string` to `src/go/fab/internal/spawn/spawn.go` (thin named wrapper over `WithProfile(cmd, "", "")`) and use it at all three raw-consumer sites: `spawn_command.go` (replacing the inline `WithProfile(..., "", "")` call), `batch_new.go:82`, and `batch_switch.go:75` — so `fab batch new`/`fab batch switch` never interpolate literal `{model}`/`{effort}` braces into their tmux commands. <!-- R6 -->
- [x] T012 Restructure `resolveTemplate` (`src/go/fab/internal/spawn/spawn.go`): non-empty substitutions use plain `strings.ReplaceAll` on the raw string (preserving whitespace runs); tokenization + token-drop applies only when at least one substituted value is empty. Add a doc comment marking quoted placeholders and valueless-flag-adjacent placeholders (e.g. `--verbose {model}` with empty model, `-- {model}`) as outside the supported grammar. <!-- R2 R3 -->
- [x] T013 Tests: extend `spawn_test.go` with whitespace-preservation (multi-space template, non-empty values), placeholder-as-first-token with empty value (exercises the `n > 0` guard), and a single token carrying both placeholders; extend `batch_new_test.go`/`batch_switch_test.go` with a templated-config case asserting the composed spawn command carries no literal braces, plus a non-templated case asserting verbatim pass-through. <!-- R6 R3 -->
- [x] T014 Update `src/kit/skills/_cli-fab.md` `fab batch new`/`batch switch` entries with a one-line placeholder-stripping note (same semantics as the `fab spawn-command` entry); sweep for a SPEC mirror or memory-adjacent spec claim describing batch spawn composition and update if present. <!-- R5 R6 -->

## Execution Order

- T001 blocks T002, T003, T004 (implementation before tests/CLI wiring)
- T003, T004 are independent of each other once T001/T002 land ([P])
- T005–T009 are independent doc edits ([P]); T010 runs after them (verification sweep)
- T011 and T012 are independent; T013 follows both; T014 follows T011

## Acceptance

### Functional Completeness

- [x] A-001 R1: `WithProfile` selects template mode when `{model}`/`{effort}` is present and the append fallback otherwise; no-placeholder output is byte-identical to the prior implementation. — `spawn.go:59-75` dispatches on `isTemplate`; the else branch is the unchanged builder. `TestWithProfile` (no-placeholder) stays green.
- [x] A-002 R2: Every occurrence of each placeholder is substituted in template mode; a placeholder-absent half is never appended (all-or-nothing). — `resolveTemplate` (`spawn.go:126-156`) substitutes via `strings.ReplaceAll` (raw on the all-non-empty path `:128-131`, per-token on the empty-value path `:150-151`); the append block in `WithProfile` is unreachable in template mode (the `isTemplate` branch returns first). Tests "single {model}…effort not appended" / "single {effort}…model not appended" cover the all-or-nothing rule. (Line-number drift from the rework restructure: was cited `:99-122`.)
- [x] A-003 R3: On an empty substituted value, the placeholder's token and a preceding `-`-prefixed token are dropped across all four flag shapes; a non-empty value drops nothing. — `spawn.go:141-146` (empty-value path) drops the token and, if `out[n-1]` begins with `-`, the preceding token; the non-empty path (`:128-131`) drops nothing (raw `ReplaceAll`). All four shapes verified against tests. (Line-number drift: was cited `:107-112`, which is now inside the doc comment.)
- [x] A-004 R4: `fab spawn-command` resolves templates with an empty profile before printing (templated → stripped; non-templated → verbatim). — `spawn_command.go:52` calls `spawn.StripPlaceholders(spawn.Command(configPath))` (the named wrapper over `WithProfile(..., "", "")` added in rework). Tests `TestSpawnCommandCmd_TemplatedConfigStripped` (→ `codex`) and `_NonTemplatedPrintsVerbatim`.
- [x] A-005 R5: The placeholder semantics are documented across the mirror class (`_preamble.md`, `fab-operator.md`, `_cli-fab.md`, their SPEC mirrors, `stage-models.md`, scaffold `config.yaml`). — All six members edited (diff confirmed).

### Behavioral Correctness

- [x] A-006 R1: Existing `spawn_test.go` no-placeholder cases remain green (the append path is unchanged), confirming back-compat. — `TestWithProfile` (4 subcases) and the `TestCommand_*` suite pass unchanged; `go test -count=1 ./internal/spawn/...` = ok.
- [x] A-007 R4: The non-templated `fab spawn-command` cases (configured command / fallback) still print exactly as today. — `TestSpawnCommandCmd_RepoWithConfiguredCommand`, `_RepoWithoutCommandFallsBack`, `_RepoMissingConfigFallsBack`, `_NonTemplatedPrintsVerbatim` all pass.

### Scenario Coverage

- [x] A-008 R2: A test exercises the `codex -m {model} -c model_reasoning_effort={effort}` substitution and a single-placeholder template. — `spawn_test.go:126-147`: "both placeholders substituted" (→ `codex -m gpt-5 -c model_reasoning_effort=high`), "single {model} placeholder", "single {effort} placeholder".
- [x] A-009 R3: A test exercises empty model / empty effort / both empty under each of the four token shapes. — `spawn_test.go:148-209`: empty model on `-m` and `--model` and `--model=`; empty effort on `-c key={effort}`; both empty; plus the empty-profile strip case.
- [x] A-010 R4: A `spawn_command_test.go` case exercises stripping on a templated config. — `TestSpawnCommandCmd_TemplatedConfigStripped` (`spawn_command_test.go:85-94`).

### Edge Cases & Error Handling

- [x] A-011 R2: Multiple occurrences of one placeholder and a placeholder embedded mid-word are covered by tests and behave correctly. — `spawn_test.go:186-201`: "multiple {model} occurrences all substituted" (`wrap {model} -- run --tag {model}` → both) and "placeholder embedded mid-word" (`--profile=pre-{model}-post` → `pre-gpt-5-post`).
- [x] A-012 R3: The `--model={model}` shape (no preceding `-`-token) drops only the single token; `-c model_reasoning_effort={effort}` drops the preceding `-c`. — `spawn_test.go:163-177`: "empty model drops single --model={model} token, no preceding flag" and "empty effort drops model_reasoning_effort token and -c".

### Code Quality

- [x] A-013 Pattern consistency: New Go code follows the surrounding `internal/spawn` style (string-builder / stdlib `strings`, focused functions, no external deps) per Constitution I. — `resolveTemplate`/`isTemplate` use only stdlib `strings`; named constants `modelPlaceholder`/`effortPlaceholder`; matches the existing builder style.
- [x] A-014 No unnecessary duplication: Template resolution logic lives once in `internal/spawn` and is reused by `fab spawn-command` rather than reimplemented. — Single `resolveTemplate` in `spawn.go`; the three raw consumers (`spawn_command.go:52`, `batch_new.go:85`, `batch_switch.go:79`) reuse it via `spawn.StripPlaceholders`, and `operator.go:103` via `WithProfile`. No parallel implementation in `cmd/fab`.
- [x] A-015 No god function: `WithProfile` stays focused (<50 lines) — helper extracted if template handling would bloat it (code-quality.md § Anti-Patterns). — `WithProfile` is 17 lines (`spawn.go:59-75`); template handling extracted to `resolveTemplate` (31 lines `spawn.go:126-156`, incl. its heavy grammar-limits doc comment — well under 50) and `isTemplate` (4 lines `spawn.go:91-94`). (Line-count drift: `resolveTemplate` was cited 24 lines pre-restructure.)
- [x] A-016 Go changes ship tests: every changed `.go` file has accompanying table-driven test updates (code-review.md § Go changes ship tests; Constitution VII). — `spawn.go` ↔ `spawn_test.go` (`TestWithProfile_Template`, 12 cases); `spawn_command.go` ↔ `spawn_command_test.go` (2 new cases).

### documentation_accuracy

- [x] A-017 R5: `_cli-fab.md` reflects the `fab spawn-command` output-behavior change (Constitution: CLI change MUST update `_cli-fab.md`); no doc still describes the append as unconditional. — `_cli-fab.md:776` adds the "Template resolution before print (leak prevention)" paragraph; `:737` operator paragraph now describes the append/template duality of `WithProfile`.
- [x] A-018 R5: Each edited `src/kit/skills/*.md` carries its matching `docs/specs/skills/SPEC-*.md` mirror update (Constitution Additional Constraints; code-quality.md § Sibling & Mirror Sweeps). — `fab-operator.md` ↔ `SPEC-fab-operator.md` (both edited). `_preamble.md`/`_cli-fab.md` mirror updates land in `stage-models.md` (their operator-launcher detail lives there, not in SPEC-_preamble/SPEC-_cli-fab, per Assumption 7). Verified `SPEC-_preamble.md`/`SPEC-_cli-fab.md` carry no operator-launcher/spawn-command append line needing edit.

### cross_references

- [x] A-019 R5: The repo-wide re-sweep (T010) leaves no stale "appends `--model`"/"WithProfile"/unconditional-`spawn_command` claim in the mirror class. — Grep of `src/kit/`+`docs/specs/` finds zero "appends `--model`" claims; all four `WithProfile` mentions describe the duality. Remaining `spawn_command` mentions (migrations, architecture.md YAML example, dated binary-review finding) are historical/example, not append-behavior claims.
- [x] A-020 R5: No edits were made under `.claude/skills/` (gitignored deployed copies); all skill edits are in `src/kit/skills/`. — `git diff --name-only` shows no `.claude/skills/` paths; all skill edits under `src/kit/skills/`.

### Rework Cycle 1 (R6 + substitution refinements)

- [x] A-021 R6: `fab batch new` and `fab batch switch` strip `{model}`/`{effort}` placeholders (empty-profile resolution) before interpolating the spawn command into tmux; non-templated commands pass through byte-for-byte. — `batch_new.go:85` and `batch_switch.go:79` both build `spawnCmd := spawn.StripPlaceholders(spawn.Command(configPath))` before composing the `tmux new-window` shell command (`batch_new.go:131`, `batch_switch.go:111`). Tests `TestRunBatchNew_SpawnCommandPlaceholderStripping` and `TestRunBatchSwitch_SpawnCommandPlaceholderStripping` capture the tmux argv and assert no literal `{model}`/`{effort}` braces reach tmux (templated → `codex '/fab-...`) and verbatim pass-through for a non-templated command.
- [x] A-022 R6: A named `spawn.StripPlaceholders` helper is used at all three raw-consumer sites (`spawn_command.go`, `batch_new.go`, `batch_switch.go`) — no site calls `WithProfile(..., "", "")` inline. — `StripPlaceholders` defined `spawn.go:85-87` (thin wrapper over `WithProfile(cmd, "", "")`); used at `spawn_command.go:52`, `batch_new.go:85`, `batch_switch.go:79`. Repo-wide grep for `WithProfile(...,"","")` finds only the helper body — no inline empty-profile call at any consumer.
- [x] A-023 R3: Non-empty substitution preserves the raw template string's whitespace (no token rejoin); tokenization occurs only on the empty-value drop path, covered by a test. — `resolveTemplate` (`spawn.go:126-156`) short-circuits at `spawn.go:128-131`: when both values are non-empty it uses plain `strings.ReplaceAll` on the raw string (no `strings.Fields`/rejoin); tokenization + `strings.Join(out, " ")` runs only on the empty-value path (`spawn.go:133-155`). Covered by `spawn_test.go:212-218` "non-empty values preserve multi-space and tab whitespace" (`codex  -m  {model}\t...` → whitespace preserved byte-for-byte).
- [x] A-024 R6: Table-driven tests in `batch_new_test.go`/`batch_switch_test.go` cover templated (stripped) and non-templated (verbatim) spawn commands. — `batch_new_test.go:338-380` and `batch_switch_test.go:167-213`: each has a templated subtest (asserts no braces + `codex '/fab-...`) and a non-templated subtest (verbatim pass-through), driven through a PATH-stubbed tmux that captures its argv.
- [x] A-025 R6: `_cli-fab.md` documents the batch commands' placeholder stripping; `resolveTemplate` carries the unsupported-grammar doc comment. — `_cli-fab.md:85` (`new`) and `:86` (`switch` in the diff) both carry a **Placeholder stripping** note naming `spawn.StripPlaceholders` and the empty-profile semantics. `resolveTemplate` doc comment (`spawn.go:119-125`) marks quoted placeholders and valueless-flag-adjacent placeholders (`--verbose {model}`, `-- {model}`) as OUTSIDE the supported grammar.

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

None — this change adds new functionality without making existing code redundant. The append path in `WithProfile` (`spawn.go:64-74`) is preserved byte-for-byte as the no-placeholder fallback (R1), and its sole production consumer `operator.go:103` keeps its unchanged `WithProfile` call shape; the new template branch and helpers (`isTemplate` `spawn.go:91-94`, `resolveTemplate` `:126-156`, `StripPlaceholders` `:85-87`) are additive and all reached. `StripPlaceholders` is the intended single seam for the three raw-output consumers (`spawn_command.go:52`, `batch_new.go:85`, `batch_switch.go:79`) — it did not obsolete any prior inline empty-profile call (the earlier `fab spawn-command` inline `WithProfile(..., "", "")` was replaced by the wrapper in the same change, so no now-dead code remains). The `fab spawn-command`/`batch` changes reuse `resolveTemplate` via the seam rather than adding a parallel resolver, so nothing was superseded.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Placeholder syntax is literal `{model}`/`{effort}`; every occurrence substituted; no-placeholder ⇒ byte-identical append fallback | User named the syntax explicitly and approved back-compat; carried verbatim from intake | S:90 R:88 A:92 D:88 |
| 2 | Confident | Any placeholder ⇒ full template mode (the placeholder-absent half is NOT appended) | Prevents cross-grammar contamination; per-half independence would append Claude flags to a non-Claude command (intake assumption 3) | S:45 R:88 A:80 D:55 |
| 3 | Confident | Empty-value rule: drop the placeholder's whitespace-delimited token + a preceding `-`-prefixed token. Tokenization (split on whitespace, rejoin with single spaces) applies **only on the empty-value drop path** — when every substituted value is non-empty, resolution uses plain `strings.ReplaceAll` on the raw string and preserves author whitespace exactly (rework cycle 1, T012; supersedes the original unconditional single-space-rejoin wording) | Deterministic rule covering all four flag shapes; whitespace tokenization matches the intake's "whitespace-delimited token" wording (shell expansions like `$(basename ...)` do not appear adjacent to placeholders in the templated grammar). The rejoin-collapses-whitespace side effect is now confined to the drop path, where a token is being removed anyway | S:45 R:82 A:75 D:50 |
| 4 | Confident | `fab spawn-command` resolves templates with an empty profile before printing; non-templated prints verbatim | Prevents literal-brace leak into the operator skill's worker spawns (intake assumption 5) | S:40 R:85 A:80 D:60 |
| 5 | Certain | Implementation confined to `internal/spawn` + `spawn_command.go`; `operator.go` call shape unchanged (append path for its non-templated Claude config) | Single production consumer confirmed by sweep; the `WithProfile` seam is unchanged (intake assumption 6) | S:75 R:90 A:90 D:85 |
| 6 | Confident | Template resolution is a single reusable function in `internal/spawn`, reused by both `WithProfile`'s template branch and `fab spawn-command` | Avoids duplicating the substitution/token-drop logic across two packages (code-quality.md No-duplication anti-pattern); alternative (reimplement in cmd) rejected | S:55 R:85 A:85 D:70 |
| 7 | Confident | Prose sweep class as enumerated (§ What Changes item 4) plus a repo-wide re-sweep at T010 | Grep-verified today (`_preamble`:325, `fab-operator`:703, `stage-models`:246); SPEC-_preamble/SPEC-_cli-fab carry no operator-launcher detail line, so those mirrors need no exception-note edit | S:60 R:85 A:85 D:75 |
| 8 | Confident | T014 batch placeholder-stripping is documented in `_cli-fab.md` prose only — no `SPEC-_cli-fab.md` row edit and no aggregate-spec (`architecture.md`/`overview.md`/`assembly-line.md`/`companions.md`) edit (rework cycle 1) | The `fab batch new`/`switch` command **surface** (flags, subcommands, error messages) is unchanged — only internal spawn-command composition changed, an implementation detail. `SPEC-_cli-fab.md` is a thin command inventory (one row per command) that already lists `fab batch`; the aggregate specs carry one-line "Worktree + tmux tab running /fab-*" inventory rows that stay accurate. Grep confirmed no spec describes batch spawn composition or the old always-append/brace-leak behavior. Mirrors the T002/T007 precedent (the earlier `fab spawn-command` leak-prevention landed in `_cli-fab.md` prose without a SPEC-_cli-fab row edit) | S:55 R:80 A:82 D:70 |

8 assumptions (2 certain, 6 confident, 0 tentative).
