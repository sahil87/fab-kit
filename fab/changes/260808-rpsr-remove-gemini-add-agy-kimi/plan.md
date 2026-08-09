# Plan: Remove Gemini, Add Antigravity (agy) and Kimi Providers

**Change**: 260808-rpsr-remove-gemini-add-agy-kimi
**Intake**: `intake.md`

## Requirements

### Providers: Built-in Roster

#### R1: The `gemini` built-in provider SHALL be removed outright
The built-in provider table MUST NOT define `gemini`. The Go symbols `providerGemini`,
`DefaultGeminiSessionCommand`, and `DefaultGeminiDispatchCommand` MUST be deleted. No compatibility
migration SHALL ship: an existing user config naming `gemini` resolves through the normal
unknown-provider path.

- **GIVEN** a fresh project with no `providers:` block
- **WHEN** `fab config explain` / `agent.ProviderNames(nil)` is consulted
- **THEN** the resolvable built-in set is exactly `agy, claude, codex, kimi`
- **AND** no exported Go identifier named `DefaultGemini*` remains

- **GIVEN** a user config with `agent.workers: gemini` and no `providers.gemini` block
- **WHEN** a stage resolves
- **THEN** the provider name passes through verbatim (no validation) and the lookup surfaces the
  ordinary unknown-provider error at the point of use — deliberately, with no migration

#### R2: The `agy` (Antigravity) built-in provider SHALL ship dispatch grammar and sparse fills — NO session_command
`providers.agy` MUST carry the dispatch command and a sparse two-role fill map, verbatim, and MUST NOT
carry a `session_command`:

```yaml
agy:
  dispatch_command: 'sh -c ''agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'''
  profiles:
    default: { model: gemini-3.1-pro-high }
    fast:    { model: gemini-3.6-flash-low }
```

The command SHALL NOT carry an `{effort}` placeholder, and no fill SHALL set `effort` — agy's model
IDs embed the reasoning level (`gemini-3.1-pro-high`), so a separate `--effort` flag would fight the
suffix.

**Why no session_command** (revised in rework cycle 2; applies to R3 identically): pane-mode dispatch
composes the provider's `session_command` and appends the prompt-file pointer as a single positional
argument. Verified live against agy v1.1.11 in a real tmux pane: `agy … "<pointer prompt>"` starts the
interactive TUI at an EMPTY prompt — the positional is silently dropped — and a fresh workspace is
additionally gated behind an interactive trust prompt even with `--dangerously-skip-permissions`, so
an unattended pane worker parks before the agent ever sees the prompt. A shipped `session_command`
would make tmux (`auto: tmux`) dispatch select pane mode and orphan every stage. With NO
`session_command`, auto-mode dispatch takes the documented soft-fallback to headless
(`auto: no session_command`) and explicit `--pane` hard-errors actionably. Users who want an
interactive agy session may add `providers.agy.session_command` in their own config; the pane
limitation MUST be documented in `_cli-agents.md`'s `### agy` entry.

- **GIVEN** `agent.workers: agy`
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=gemini-3.1-pro-high`, `provider=agy`, and a `dispatch=` line are emitted
- **AND** no `effort=` line is emitted

- **GIVEN** `agent.workers: agy`
- **WHEN** `fab resolve-agent ship` runs (the `fast` role)
- **THEN** `model=gemini-3.6-flash-low` — role differentiation survives the swap

#### R3: The `kimi` built-in provider SHALL ship dispatch grammar only — NO per-role fills, NO session_command
`providers.kimi` MUST carry only the dispatch command and MUST define no `profiles` map and no
`session_command`:

```yaml
kimi:
  dispatch_command: 'sh -c ''kimi -m {model} -p "$(cat)"'''
```

`-m` takes a *user-config* model alias, so a pinned value would break non-managed installs. The empty
resolved `{model}` MUST drop the `-m` flag via `spawn.WithProfile`'s empty-value token-drop, letting
kimi fall back to the user's own `default_model`. The dispatch form MUST carry no approval flag
(`kimi -p` already auto-approves and rejects `--yolo`/`--auto`).

**Why no session_command**: same pane-adapter contract violation as R2, failing louder — kimi parses a
bare positional as a SUBCOMMAND (`kimi --yolo "<pointer prompt>"` exits non-zero with
`unknown command '…'`, verified against kimi-code v0.34.0), and kimi has no
interactive-initial-prompt flag at all (`-p` is non-interactive; upstream issue #2240 tracks the gap).
The pane limitation MUST be documented in `_cli-agents.md`'s `### kimi` entry.

- **GIVEN** `agent.workers: kimi` and no `providers.kimi` config
- **WHEN** `fab resolve-agent apply` runs
- **THEN** the resolved model is EMPTY (no `model=` value), `provider=kimi`, and
  `dispatch=sh -c 'kimi -p "$(cat)"'` — the `-m {model}` pair dropped, the quoted segment left
  syntactically intact

- **GIVEN** `agent.workers: agy` (or `kimi`) inside tmux
- **WHEN** `fab dispatch start` runs in auto mode
- **THEN** the dispatch soft-falls back to headless with the `auto: no session_command` selection
  suffix — no pane worker is spawned, nothing orphans

#### R4: The rendered config reference SHALL describe the four-provider roster
`internal/configref` MUST render `agy` and `kimi` reference blocks in place of `gemini`, every command
string interpolated from its canonical `internal/agent` symbol (no literal copy). Every "THREE built-in
providers" / "claude, codex and gemini" phrase in the reference text and the managed-fence text MUST
name the four-provider roster instead. The reference MUST state that `kimi` deliberately ships no
fills.

- **GIVEN** a rendered `fab config explain`
- **WHEN** the `providers:` segment is read
- **THEN** it shows `claude` live plus commented `# codex:` / `# agy:` / `# kimi:` blocks
- **AND** `kimi`'s block renders no `profiles:` key at all (`profilesLines` returns "" for an empty map)
- **AND** the rendered text parses with `GetProvider("codex"/"agy"/"kimi")` reporting `!ok` (commented
  blocks register no project override)

### Distribution: Skill Sync Targets

#### R5: The Gemini deploy target SHALL be removed and the generic `.agents/skills` gate widened
`deploySkills` MUST NOT define a `.gemini/skills` target. The generic-directory target
(`.agents/skills`) MUST deploy when **any** of `codex`, `agy`, `kimi` is on PATH, and its label MUST
reflect the generic nature rather than naming one CLI. `agentAvailable` MUST accept multiple candidate
binaries, matching `FAB_AGENTS` against any of them. No per-brand target SHALL be added for `agy` or
`kimi` — both read the generic workspace directory natively, so duplicate-skill conflict warnings are
impossible by construction.

- **GIVEN** only `agy` on PATH (via `FAB_AGENTS=agy`)
- **WHEN** `fab-kit sync` deploys skills
- **THEN** `.agents/skills/{name}/SKILL.md` is written
- **AND** no `.gemini/skills` directory is created

- **GIVEN** `FAB_AGENTS` naming none of `codex`/`agy`/`kimi`
- **WHEN** deployment runs
- **THEN** the generic target is skipped with a message naming all three candidates

#### R6: The scaffold gitignore fragment SHALL drop `/.gemini` and add `/.kimi`
`src/kit/scaffold/fragment-.gitignore` MUST no longer ignore `/.gemini` and MUST ignore `/.kimi`.
`/.agents` is already present and stays. The dev repo's own `.gitignore` gains `/.kimi` but KEEPS
`/.gemini` (revised in rework cycle 2): the main checkout carries an untracked stale `.gemini/skills/`
tree, and un-ignoring it would surface 36 stray files in `git status`. User projects are unaffected —
the fragment merges via `lineEnsureMerge`, which only ever adds lines.

- **GIVEN** a project scaffolded from the fragment
- **WHEN** a kimi session creates `.kimi/`
- **THEN** it is gitignored, and no `/.gemini` entry remains

### Quality: Drift Guards and Documentation

#### R7: Every drift-guarded test SHALL be updated in the same change
The pinned/guard tests named in `defaults.yaml`'s header MUST pass against the new roster:
`TestDefaultsFileIsWellFormed`, `TestDefaultsFileProviders`, `TestDefaultRoleProfilesArePinned`,
`TestNonClaudeProviderFillsArePinned`, `TestDocTablesMatchAgentMaps`,
`TestMirrorDocsMatchDefaultProfiles`, `TestCLIFabReferenceListsDefaultRoles`. Assertions that assume
every built-in carries a `profiles.default.model` MUST be relaxed for `kimi` and re-expressed as a
deliberate no-fills assertion, not deleted. `internal/spawn` MUST gain coverage for the nested-shell
quoted-placeholder shape both new dispatch commands use.

- **GIVEN** the new `defaults.yaml`
- **WHEN** `go test ./...` runs for `internal/agent`, `internal/configref`, `internal/spawn`,
  `internal/config`, `internal/configupgrade`, `cmd/fab`, and `fab-kit/internal`
- **THEN** all pass

- **GIVEN** `sh -c 'kimi -m {model} -p "$(cat)"'` and an empty model
- **WHEN** `spawn.WithProfile` resolves it
- **THEN** the `-m {model}` pair is dropped and the result is `sh -c 'kimi -p "$(cat)"'`

#### R8: The whole skill/SPEC/spec mirror class SHALL be swept
Every `src/kit/skills/*.md` file whose provider dictionary or roster enumeration names `gemini` MUST be
updated together with its `docs/specs/skills/SPEC-*.md` mirror, plus the aggregate specs that restate
the roster (`stage-models.md`, `config.md`, `architecture.md`, `glossary.md`).

- **GIVEN** the change is complete
- **WHEN** `grep -ri gemini src/kit/skills docs/specs` runs
- **THEN** the only surviving hits are run-kit `rk agent-setup` harness-coverage statements and the
  Superpowers platform-shim comparison — statements about *other* projects, not fab's provider roster

### Non-Goals

- **No compat migration for existing gemini users** — user decision ("Just remove support"). Existing
  `.gemini/skills/` directories in user checkouts are left behind (gitignored, harmless).
- **No per-brand skill deploy target for agy or kimi** — the generic `.agents/skills/` covers both.
- **Historical artifacts are not rewritten** — `src/kit/migrations/*.md`, `docs/memory/*/log.md`,
  `log.seed.md`, and archived `fab/changes/**` keep their gemini mentions verbatim; they are records,
  not present-truth claims.
- **run-kit harness-coverage statements are not rewritten** — "`rk agent-setup` covers Claude Code,
  Codex, Copilot, Gemini, OpenCode" is a fact about run-kit's instrumentation, unaffected by fab's
  provider roster. Same for `superpowers-comparison.md`'s note that Superpowers ships a Gemini CLI
  shim. `src/kit/skills/fab-operator.md` and `SPEC-fab-operator.md` carry only such mentions, so they
  are out of scope despite the intake listing them.
- **`docs/memory/` is not touched by apply** — memory updates are hydrate's stage; the intake's
  Affected Memory list feeds that stage.
- **This repo's own `fab/project/config.yaml` managed fence is not hand-edited** — its text source is
  `configref.go`, and it regenerates via `fab config upgrade` once a binary carrying this change is
  installed.

### Design Decisions

#### Nested-shell `"$(cat)"` idiom for both new dispatch commands
**Decision**: Both new `dispatch_command`s wrap the CLI in `sh -c '... -p "$(cat)"'`.
**Why**: Both CLIs take the prompt as an *argument* to `-p` and ignore stdin, while `fab dispatch`
delivers the prompt on stdin. POSIX expands `$(cat)` **before** applying the `< file` redirect, so the
un-nested form reads the *outer* stdin; nesting a shell makes the inner `sh`'s stdin the redirected
prompt file. Verified end-to-end in dispatch shape for both CLIs.
**Rejected**: `-p` with no argument (errors: `flag needs an argument: -p`); relying on stdin as the
prompt (never read by either CLI).
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi

#### kimi ships no per-role fills
**Decision**: `providers.kimi` defines no `profiles` map at all.
**Why**: kimi's `-m` takes a *user-config* model alias (managed installs use `kimi-code`/`k3`; custom
providers differ), so any pinned value breaks non-managed setups. An empty `{model}` drops the `-m`
pair cleanly through the existing token-drop, and kimi falls back to the user's `default_model` — the
correct answer for every install shape.
**Rejected**: Pinning `k3` (breaks custom-provider installs); a sparse `default` fill (same problem,
one row smaller).
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi

#### One generic skill-deploy target, gated on any of three CLIs
**Decision**: The `.agents/skills` target deploys when `codex`, `agy`, **or** `kimi` is on PATH; no
per-brand directory is deployed for the two new providers.
**Why**: The gemini skill-conflict warning class existed because the same skill set was deployed to
both `.agents/skills/` and `.gemini/skills/`, and gemini read both. agy discovers
`<workspace>/.agents/skills/<skill>/SKILL.md` natively, and kimi's discovery merges the generic group
with its brand group by priority. One deployment target per skill set makes duplication impossible by
construction rather than by warning suppression.
**Rejected**: Per-brand `.agy/skills` + `.kimi/skills` targets (reintroduces the duplicate-discovery
class the change exists to kill).
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi

### Deprecated Requirements

#### gemini built-in provider
**Reason**: No longer used; its per-brand skill deployment produced recurring conflict warnings, and
its config rotted at release cadence.
**Migration**: N/A — removed outright by user decision. Users wanting it back write a
`providers.gemini` block in their own config (the provider table is user-extensible).

#### `.gemini/skills` deploy target
**Reason**: Root cause of the duplicate-skill conflict warnings.
**Migration**: N/A — existing `.gemini/skills/` directories are gitignored and left in place.

## Tasks

### Phase 1: Provider data and Go symbols

- [x] T001 <!-- rework c2 (revise requirements): R2/R3 revised — agy and kimi ship NO session_command (pane pointer-prompt delivery fails on both CLIs, verified live; agy also trust-prompts fresh workspaces). Update defaults.yaml blocks + comments to dispatch_command-only per the revised R2/R3 YAML --> Rewrite the `providers:` block of `src/go/fab/internal/agent/defaults.yaml` — delete the `gemini:` block, add the `agy:` and `kimi:` blocks verbatim per R2/R3 (with their explanatory comments), and update the file header's "ALL THREE ship fills" / "codex and gemini carry one" prose and the drift-guard test roster to the four-provider reality <!-- R1 R2 R3 -->
- [x] T002 Update `src/go/fab/internal/agent/agent.go` — replace `providerGemini` with `providerAgy` and `providerKimi`, replace the `DefaultGemini*` vars with `DefaultAgy*` and `DefaultKimi*`, and revise the package doc / `defaultProviders` / const-block comments that say "three providers" or name gemini <!-- R1 R2 R3 -->

### Phase 2: Rendered reference and comment sweeps

- [x] T003 Update `src/go/fab/internal/configref/configref.go` — `providersSegment` renders `# agy:` and `# kimi:` in place of `# gemini:` (interpolating the new agent vars), the per-provider notes describe agy's effort-suffixed IDs and kimi's deliberate no-fills, `agentSegment`'s "claude, codex and gemini" becomes the four-name roster, and the `providers` registry row `Description` says four built-ins <!-- R4 -->
- [x] T004 [P] Replace the `fab config set agent.session gemini --system` example in `src/go/fab/cmd/fab/config.go` <!-- R4 -->
- [x] T005 [P] Comment-only sweeps: `src/go/fab/internal/config/config.go:144`, `src/go/fab/internal/configupgrade/configupgrade.go` (fence-comment references to the `# gemini:` block). `src/go/fab/internal/pane/pane.go:220` was verified OUT of scope at apply — its `(codex/copilot/gemini)` list enumerates the harnesses run-kit's `rk agent-setup` instruments, not fab's provider roster, so it falls under the run-kit non-goal <!-- R1 -->

### Phase 3: Skill sync targets and scaffold

- [x] T006 Update `src/go/fab-kit/internal/skills.go` — `agentConfig.CLI string` becomes `CLIs []string`, `agentAvailable` becomes variadic over candidates (FAB_AGENTS matches any), delete the Gemini target, and relabel the `.agents/skills` target to reflect its `codex`/`agy`/`kimi` gate (skip message names all candidates) <!-- R5 -->
- [x] T007 Add `src/go/fab-kit/internal/skills_test.go` coverage — `agentAvailable` multi-candidate matching under `FAB_AGENTS`, and a `deploySkills` case proving the `.agents/skills` target fires for `agy`-only and no `.gemini/skills` is created <!-- R5 -->
- [x] T008 [P] `src/kit/scaffold/fragment-.gitignore`: drop `/.gemini`, add `/.kimi`; sweep the dev repo's own `.gitignore` to match <!-- R6 -->

### Phase 4: Test updates (drift guards and fixtures)

- [x] T009 <!-- rework c2: session_command assertions for agy/kimi must flip to absence assertions per revised R2/R3 --> `src/go/fab/internal/agent/defaults_test.go` — roster becomes the four built-ins; the no-`{effort}` assertion moves from gemini to agy; the `profiles.default.model` and `dispatch_command` assertions are re-expressed so `kimi`'s deliberate empty fill map is asserted rather than failing; wire the `DefaultAgy*`/`DefaultKimi*` vars into `TestPackageTablesMatchDefaultsFile` <!-- R7 -->
- [x] T010 <!-- rework c2: grammar assertions for agy/kimi flip to dispatch-only (no session_command); assert the no-session_command resolution/fallback behavior where a seam exists --> `src/go/fab/internal/agent/agent_test.go` — `TestNonClaudeProviderFillsArePinned` pins agy's two fills and kimi's empty map; `TestWorkersKnobResolvesBuiltInFills` runs over codex/agy and gains a kimi case asserting an EMPTY model resolves deliberately; `TestResolveProvider_BuiltInCodexAndGemini` becomes the four-built-in equivalent with agy/kimi grammar assertions; swap remaining gemini fixtures <!-- R7 -->
- [x] T011 [P] `src/go/fab/cmd/fab/` test sweep — `agent_test.go`, `config_test.go`, `config_show_init_test.go`, `resolve_agent_test.go`: replace gemini fixtures/expectations with agy or kimi (or a neutral user-defined provider where the fixture only needed *some* provider) <!-- R7 -->
- [x] T012 [P] `src/go/fab/internal/config/config_test.go` and `src/go/fab/internal/configupgrade/configupgrade_test.go` — swap gemini fixtures and update the fence-comment expectations that name the `# gemini:` block <!-- R7 -->
- [x] T013 [P] Add `internal/spawn` coverage for the nested-shell quoted-placeholder shape: `sh -c 'kimi -m {model} -p "$(cat)"'` with an empty model drops the `-m` pair and keeps the quoted segment intact; the agy template substitutes a non-empty model without disturbing `"$(cat)"` <!-- R3 R7 -->

### Phase 5: Kit content and spec mirrors

- [x] T014 <!-- rework c2: ### agy and ### kimi entries must document the no-session_command posture and WHY (pane pointer-prompt delivery fails: kimi positional=subcommand error, agy silently drops positional + first-run trust prompt); interactive sessions need a user-added session_command with the pane caveat --> `src/kit/skills/_cli-agents.md` — frontmatter description and `helpers`-adjacent provider list become four; replace the `### gemini` dictionary entry with `### agy` and `### kimi` entries (grammar, `-p` semantics, profile flags, discovery recipes); update the "built-ins" enumerations. Leave the two `rk agent-setup` harness-coverage sentences verbatim <!-- R8 -->
- [x] T015 `src/kit/skills/_cli-fab.md` — § fab config `providers:` paragraph and § fab agent provider-addressed bullet name the four built-ins, describe agy's suffixed-effort IDs and kimi's no-fills/`default_model` inheritance. Leave the § fab pane `@rk_agent_state` sentence verbatim <!-- R8 -->
- [x] T016 [P] <!-- rework c2: mirror the T014/T015 session_command revisions in lockstep --> SPEC mirrors for the two edited skills plus the roster claim in `docs/specs/skills/SPEC-_preamble.md` — `SPEC-_cli-agents.md`, `SPEC-_cli-fab.md`, `SPEC-_preamble.md` (its "all three carry fills" claim is now false: kimi carries none) <!-- R8 -->
- [x] T017 <!-- rework c2: inline-YAML sample + prose must show agy/kimi as dispatch_command-only; sweep any both-commands claims --> `docs/specs/stage-models.md` — § Three built-in providers becomes four (inline-YAML sample carries agy's fills and kimi's fill-less block), the two gemini-specific "load-bearing shapes" bullets become agy/kimi ones, the consequences list, the § Refreshing prose, the § Config schema sample, the harness-adapter and drift-guard paragraphs, and the `agent.workers:` examples <!-- R8 -->
- [x] T018 [P] <!-- rework c2: sweep config.md/architecture.md/glossary.md for agy/kimi session_command claims --> `docs/specs/architecture.md` (providers reference block + the Agent Integration deploy-target table), `docs/specs/config.md`, `docs/specs/glossary.md` — four-provider roster, `.gemini/skills` row removed, `.agents/skills` row re-described as the generic multi-CLI target <!-- R5 R8 -->

- [x] T020 [P] `docs/specs/skills.md:53` — the fab-operator helpers row still calls `_cli-agents` the "three-provider grammar/discovery dictionary"; make it four-provider. Then sweep the roster-COUNT phrasings repo-wide (`three.provider`, `three built-in`, `all three`) in src/kit/skills and docs/specs — counts escape the gemini-word grep <!-- R8 -->
- [x] T021 [P] Dev repo `.gitignore` — keep `/.gemini` alongside `/.kimi` (the main checkout has an untracked stale `.gemini/skills/` tree; user projects merge via lineEnsureMerge and keep their entry, but this repo was hand-edited) <!-- R6 -->
- [x] T022 [P] `src/go/fab-kit/internal/skills.go:48` — singular skip message for single-candidate targets (`Skipping Claude Code: claude not found in PATH`), "none of" phrasing only for the multi-candidate `.agents/skills` target <!-- R5 -->

### Phase 6: Verification

- [x] T019 Run the affected package tests (`internal/agent`, `internal/configref`, `internal/config`, `internal/configupgrade`, `internal/spawn`, `internal/pane`, `cmd/fab`, `fab-kit/internal`), then `go build ./...` and a repo-wide `grep -ri gemini` to confirm only the allow-listed historical/other-project mentions survive <!-- R7 R8 -->

## Execution Order

- T001 blocks T002 (the Go vars read the file's keys), which blocks T003 (the reference interpolates
  those vars).
- T001–T003 block Phase 4 (every drift guard derives expectations from the new table).
- T017 must land before T019: `TestMirrorDocsMatchDefaultProfiles` reads `docs/specs/stage-models.md`,
  and `TestCLIFabReferenceListsDefaultRoles` reads `src/kit/skills/_cli-fab.md` (T015) — the doc sweep
  is *test-enforced*, not cosmetic.
- T006 blocks T007. T008, T014–T018 are otherwise independent.

## Acceptance

### Functional Completeness

- [x] A-001 R1: No `gemini` key exists in `defaults.yaml`, and no `providerGemini` / `DefaultGemini*` identifier remains anywhere in `src/go/`
- [x] A-002 R2: `providers.agy` ships the dispatch command only (no session_command) and the two pinned fills (`default: gemini-3.1-pro-high`, `fast: gemini-3.6-flash-low`) exactly as specified
- [x] A-003 R3: `providers.kimi` ships the dispatch command only — no session_command, no `profiles` map
- [x] A-026 R2/R3: With `agent.workers: agy` (or `kimi`) inside tmux, `fab dispatch start` auto-mode soft-falls back to headless with the `auto: no session_command` suffix — no pane worker, nothing orphans (assert at whatever seam the existing fallback tests use)
- [x] A-004 R4: `fab config explain` renders `agy` and `kimi` blocks with every command string interpolated from an `internal/agent` symbol (no literal copy in `configref.go`)
- [x] A-005 R5: `deploySkills` defines no `.gemini/skills` target, and the `.agents/skills` target is gated on any of `codex`/`agy`/`kimi`
- [x] A-006 R6: `fragment-.gitignore` ignores `/.kimi` and not `/.gemini`
- [x] A-007 R8: The two edited skills and their SPEC mirrors, plus `stage-models.md`/`config.md`/`architecture.md`/`glossary.md`, describe the four-provider roster — including kimi's deliberate no-fills posture wherever a fills claim is made, and the `session_command`-reachability claim recounted to four built-ins

### Behavioral Correctness

- [x] A-008 R2: `agent.workers: agy` resolves `gemini-3.1-pro-high` for apply and `gemini-3.6-flash-low` for ship, with no `effort` on either
- [x] A-009 R3: `agent.workers: kimi` resolves an EMPTY model, and the composed `dispatch=` is `sh -c 'kimi -p "$(cat)"'` — the `-m` pair dropped, quoting intact
- [x] A-010 R5: With only `agy` on PATH, skills deploy to `.agents/skills/` and no `.gemini/skills` directory is created
- [x] A-011 R1: A config naming `gemini` gets the ordinary unknown-provider treatment — no special-case code path, no migration file added

### Removal Verification

- [x] A-012 R1: `grep -ri gemini src/go src/kit/skills src/kit/scaffold docs/specs` returns only run-kit `rk agent-setup` harness-coverage statements and the Superpowers comparison line — no fab provider-roster claim survives. Exclude the two legitimate non-claim classes before reading the output: `| grep -viE 'gemini-3\.|rk agent-setup|superpowers|regression'` — agy's own model IDs are `gemini-3.*` strings, and `skills_test.go`'s regression comments name the retired `.gemini/skills` target on purpose
- [x] A-013 R1: `src/kit/migrations/*.md`, `docs/memory/**/log.md`, `log.seed.md`, and archived `fab/changes/**` are unmodified — history is not rewritten

### Scenario Coverage

- [x] A-014 R7: Every drift-guard test named in `defaults.yaml`'s header passes, and each was *updated* (not disabled or loosened into vacuity) where the roster changed
- [x] A-015 R7: `internal/spawn` has a test for the nested-shell quoted-placeholder shape used by both new dispatch commands
- [x] A-016 R5: `skills_test.go` covers multi-candidate `agentAvailable` matching under `FAB_AGENTS`

### Edge Cases & Error Handling

- [x] A-017 R3: `TestDefaultsFileProviders`' "every built-in resolves a model" assertion is re-expressed so kimi's deliberate absence is *asserted*, not silently skipped
- [x] A-018 R4: `profilesLines` returning "" for kimi's empty fill map renders no stray `# profiles:` key, and the rendered reference still round-trips (commented blocks parse as absent)
- [x] A-019 R5: The generic target's skip message names all three candidate CLIs, so a user with none installed can tell what would enable it

### Code Quality

- [x] A-020 Pattern consistency: New provider blocks follow the existing `defaults.yaml` shape (grammar comment above, sparse fills below); new Go symbols follow the `Default{Provider}{Session,Dispatch}Command` naming
- [x] A-021 No unnecessary duplication: Command strings live once in `defaults.yaml` and reach the reference by interpolation; no literal copy is introduced in `configref.go` or any doc that a drift guard covers
- [x] A-022 Canonical source only: No file under `.claude/skills/` is edited — every skill change lands in `src/kit/skills/`
- [x] A-023 SPEC-mirror sync: Every edited `src/kit/skills/*.md` carries its `docs/specs/skills/SPEC-*.md` update in this change
- [x] A-024 Go changes ship tests: Every touched Go package has its tests updated in the same change, and they conform to the spec rather than the reverse
- [x] A-025 No migration was invented for user-data restructuring that the change deliberately does not perform (removal is by user decision, not a data reshape)

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [ ] A-NNN **N/A**: {reason}`

## Deletion Candidates

- `src/go/fab-kit/internal/skills.go:19` (`agentConfig.Mode`) and its `symlink` branches at `skills.go:161`/`skills.go:198` — all three remaining deploy targets ship `Mode: "copy"`, so the symlink path has zero PRODUCTION producers (only `skills_test.go:229` still constructs one). **Pre-existing** (gemini was `copy` too), so this change did not create the redundancy — surfaced only because the target table was rewritten here.
- ~~The three-copy YAML single-quote-doubling rule~~ — **resolved in rework cycle 1**: the rule now has one owner, the exported `configref.YAMLSingleQuoted`, called by `configref.providersSegment`, `cmd/fab/config_test.go:418`, and `configupgrade_test.go:367`. Re-verified this cycle — no copies remain.
- No other existing code was made redundant: the change swaps provider rows in a data file and its projections, and the deleted `providerGemini` / `DefaultGemini*` symbols and `.gemini/skills` target were removed in the diff itself.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Go symbol names are `providerAgy`/`providerKimi` and `DefaultAgy{Session,Dispatch}Command` / `DefaultKimi{Session,Dispatch}Command` | Mechanical application of the existing `DefaultCodex*`/`DefaultGemini*` naming pattern to the new keys | S:90 R:90 A:95 D:90 |
| 2 | Certain | `agentConfig.CLI string` becomes `CLIs []string` with a variadic `agentAvailable(clis ...string)` | Intake requires a multi-candidate probe; variadic keeps the existing single-arg test call sites compiling, minimizing churn | S:85 R:85 A:90 D:85 |
| 3 | Confident | `src/kit/skills/fab-operator.md` and `SPEC-fab-operator.md` need no edit | Their only gemini mentions describe run-kit's `rk agent-setup` harness coverage, not fab's provider roster — unaffected by this change. Intake listed them speculatively ("verify at apply") | S:75 R:85 A:85 D:80 |
| 4 | Confident | `docs/specs/hooks.md` and `docs/specs/superpowers-comparison.md` need no edit | Same class: run-kit instrumentation coverage, and a factual note about what Superpowers ships | S:75 R:90 A:85 D:80 |
| 5 | Confident | This repo's own `fab/project/config.yaml` managed fence is left stale | Its text source is `configref.go`; it regenerates via `fab config upgrade` once a binary carrying this change is installed. Hand-splicing fence bytes risks a ragged fence for zero durable gain | S:70 R:85 A:80 D:75 |
| 6 | Confident | `docs/memory/` is untouched during apply | Memory is hydrate's stage; the intake's Affected Memory list is that stage's input. Apply writing memory would duplicate hydrate's present-truth merge | S:80 R:85 A:85 D:85 |
| 7 | Confident | The `.agents/skills` target label becomes `Agents dir` (candidates named in the skip message rather than baked into the label) | Keeps the `%-12s` column alignment in `syncAgentSkills`' summary line readable while still telling a user with no CLI installed what would enable the target | S:60 R:90 A:80 D:70 |
| 8 | Confident | `TestResolveProvider_BuiltInCodexAndGemini` is renamed to a roster-neutral name rather than kept and extended | The name encodes a two-provider roster that no longer holds; a roster-neutral name survives the next provider change | S:65 R:90 A:85 D:75 |

8 assumptions (2 certain, 6 confident, 0 tentative).
