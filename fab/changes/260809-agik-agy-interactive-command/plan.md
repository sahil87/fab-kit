# Plan: Give agy a Built-in `interactive_command`

**Change**: 260809-agik-agy-interactive-command
**Intake**: `intake.md`

## Requirements

### Provider data: agy's pane capability

#### R1: agy's built-in row SHALL carry an `interactive_command`
`src/go/fab/internal/agent/defaults.yaml` MUST add `interactive_command: 'agy --dangerously-skip-permissions --model {model} -i {prompt}'` to the `agy` provider row. kimi's row MUST remain interactive-command-free.

- **GIVEN** a project with no `providers:` config
- **WHEN** `fab config explain providers` renders the built-in table, or `agent.ResolveProvider(nil, "agy")` is called
- **THEN** agy's `interactive_command` is non-empty and carries the shipped grammar
- **AND** kimi's `interactive_command` is still empty

#### R2: The agy interactive grammar SHALL be shape-constrained
The shipped string MUST carry `--dangerously-skip-permissions`, MUST substitute `{model}`, MUST NOT carry an `{effort}` placeholder (agy's model IDs embed the reasoning level as a suffix), and MUST END with the `-i {prompt}` pair — `-i`/`--prompt-interactive` is a **value-taking** flag (probe P1, agy 1.1.11: bare `-i` exits 2), so the pointer must arrive as its value, never as a bare positional.

- **GIVEN** the built-in agy provider
- **WHEN** a test inspects `agent.DefaultAgyInteractiveCommand`
- **THEN** it contains `{model}` and `{prompt}`, does not contain `{effort}`, and its final two tokens are `-i {prompt}`
- **AND** `spawn.WithProfile` substitution leaves that pair in final position

#### R3: `internal/agent` Go prose and exported symbols SHALL describe kimi as the sole dispatch-only built-in
`src/go/fab/internal/agent/agent.go` MUST expose `DefaultAgyInteractiveCommand` sourced from `defaultProviders[providerAgy].InteractiveCommand` (the same no-duplicate-literal convention `DefaultCodexInteractiveCommand` follows), and every comment block asserting "agy and kimi deliberately ship none" MUST be rewritten to kimi-only. `defaults.yaml`'s own providers-block note MUST be rewritten the same way, recording the `{prompt}` mechanism and the trust-wall caveat.

- **GIVEN** a reader of `internal/agent`
- **WHEN** they read the providers-block note in `defaults.yaml` or the command-var block in `agent.go`
- **THEN** agy is described as pane-capable via its `-i {prompt}` grammar, with the trust-wall caveat
- **AND** kimi remains described as the deliberate no-interactive-command built-in

### Spawn composition: the `{prompt}` placeholder

#### R4: `{prompt}` SHALL be a recognized `interactive_command` placeholder with substitute-or-drop semantics
`src/go/fab/internal/spawn` MUST recognize a third placeholder token `{prompt}` and expose it through two exported entry points: a predicate reporting whether a command carries it, and a resolver substituting a supplied value for it. A non-empty value substitutes every occurrence, preserving the command's raw whitespace; an EMPTY value triggers the same token-drop rule `{model}`/`{effort}` already use — the whitespace-delimited token containing the placeholder is dropped along with an immediately preceding token beginning with `-`.

- **GIVEN** the command `agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i {prompt}`
- **WHEN** the prompt value is `'Read x.md and execute it.'`
- **THEN** the result is that command with the quoted value in place of `{prompt}`
- **AND** with an empty prompt value, both `-i` and `{prompt}` are dropped, leaving `agy --dangerously-skip-permissions --model gemini-3.1-pro-high`

#### R5: `{prompt}` SHALL NOT change `WithProfile`'s template-vs-append mode
`isTemplate` MUST continue to key on `{model}`/`{effort}` alone. A command carrying only `{prompt}` MUST still take append mode, receiving `--model`/`--effort` appended at the end exactly as a placeholder-free command does. `{prompt}` resolution MUST therefore be a SEPARATE pass over the profile-resolved command, not a third parameter of `WithProfile`.

- **GIVEN** the command `mycli --flag -i {prompt}` with model `m1` and effort `high`
- **WHEN** `spawn.WithProfile` resolves it
- **THEN** the result is `mycli --flag -i {prompt} --model m1 --effort high` (append mode, `{prompt}` untouched)
- **AND** the subsequent prompt pass resolves `{prompt}` independently

#### R6: Launch seams SHALL deliver — or drop — the initial prompt by placeholder presence
<!-- rework cycle 1: R6 originally classified operator/batch as promptless seams; they carry initial prompts, and WithPrompt(cmd, "") + positional append silently loses them on a {prompt}-carrying provider (a regression vs the pre-change claude fallthrough) -->
Bare `fab agent` (`cmd/fab/agent.go` `runAgent`) is the **only** seam that composes an `interactive_command` with no initial content: it MUST run the prompt pass with an empty value after `spawn.WithProfile`, so a `{prompt}`-carrying command launches a plain session with the flag+placeholder pair dropped.

Every seam that composes an `interactive_command` **with** an initial prompt — the operator launcher (`cmd/fab/operator.go`, `'/fab-operator'`), `fab batch new` (`cmd/fab/batch_new.go`, `/fab-new …`) and `fab batch switch` (`cmd/fab/batch_switch.go`, `/fab-switch …`) — MUST deliver it by the same two-shape branch pane dispatch uses: substitute the shell-quoted prompt at `{prompt}` when the resolved command carries the placeholder, append it positionally (shell-quoted) otherwise. The branch MUST exist as **one shared helper** used by all consumers including `internal/dispatch.WindowCommand` — a single implementation of the delivery grammar. These seams MUST NOT run the empty-value prompt pass.

- **GIVEN** the built-in agy provider and no `providers:` config
- **WHEN** `fab agent --provider agy --print` runs
- **THEN** it prints `agy --dangerously-skip-permissions --model <resolved-id>` with no `-i` and no `{prompt}`
- **AND** `fab agent --provider agy` (no `--print`) execs that same command

- **GIVEN** `agent.session: agy` (or any provider whose `interactive_command` carries `{prompt}`)
- **WHEN** the operator launcher or a `fab batch new`/`switch` worker spawn composes its window command
- **THEN** the initial prompt (`/fab-operator`, `/fab-new …`, `/fab-switch …`) appears shell-quoted as the `{prompt}` substitution (agy: as `-i`'s value) and no `{prompt}` token survives
- **AND** for a placeholder-free provider (claude/codex) the composed string is byte-identical to the pre-change positional-append form

#### R7: Pane dispatch SHALL substitute the shell-quoted pointer at `{prompt}`, preserving positional append otherwise
`internal/dispatch.WindowCommand` MUST substitute the shell-quoted pointer at `{prompt}` when the resolved command carries the placeholder. When it does NOT, `WindowCommand` MUST keep today's behavior byte-for-byte: `resolvedCmd + " " + shellQuote(pointer)`. The pointer MUST be shell-quoted on BOTH paths (a repo path containing a single quote must not break out of the embedded argument).

- **GIVEN** the resolved pane command `agy … --model <id> -i {prompt}` and the pointer `Read .fab-dispatch/agik/apply-prompt.md and execute it.`
- **WHEN** `WindowCommand` composes the tmux command
- **THEN** the quoted pointer appears as `-i`'s value and no `{prompt}` token survives
- **AND** for `claude …`/`codex …` (no `{prompt}`) the composed string is byte-identical to the pre-change positional-append form

### Rendered reference: `fab config explain`

#### R8: The rendered providers reference SHALL show agy's `interactive_command`
`src/go/fab/internal/configref/configref.go`'s commented `# agy:` block MUST open on `interactive_command` (interpolated from `agent.DefaultAgyInteractiveCommand`, never a literal copy) followed by `headless_command`, and its surrounding prose MUST stop asserting agy is dispatch-only.

- **GIVEN** `fab config explain providers` (or `fab config init`'s managed fence)
- **WHEN** the `# agy:` block renders
- **THEN** its first key is `interactive_command` carrying the shipped grammar, and the "dispatch only" annotation is gone from agy's `headless_command` line
- **AND** stripping the leading `# ` from the block still yields valid YAML

### Documentation sweep: skills and their SPEC mirrors

#### R9: `_cli-agents.md` SHALL describe agy as pane-capable and document `{prompt}`
`src/kit/skills/_cli-agents.md` MUST update the session-form roster note and the built-ins roster line, document the `{prompt}` placeholder in § Spawn Composition alongside the `{model}`/`{effort}` token-drop rule, and rewrite § agy's Interactive row + Dispatch-only section. The "add one in your own config" recipe for agy MUST be removed (it is now shipped). kimi's § Dispatch-only MUST keep its own rationale and drop the "Like agy" framing. The agy Approvals row MUST read true now that agy genuinely has two forms.

- **GIVEN** an agent reading `_cli-agents.md` § agy
- **WHEN** it looks for agy's interactive form
- **THEN** it finds the shipped `interactive_command` grammar and the trust-wall caveat, not an "add one yourself" recipe
- **AND** no roster phrasing claims only claude and codex ship an interactive command

#### R10: `_cli-fab.md` SHALL describe agy as pane-capable and document `{prompt}`
`src/kit/skills/_cli-fab.md` § `providers:` (the four-built-ins paragraph and its dispatch-only-posture rationale) and § `fab agent`'s error-reachability note MUST flip to kimi-only, and the `{prompt}` placeholder MUST be documented where the composition grammar is described (§ `providers:` and § `fab dispatch`'s pane composition).

- **GIVEN** the `_cli-fab.md` providers reference
- **WHEN** a reader checks which built-ins have pane capability
- **THEN** claude, codex and agy are listed and kimi is the sole dispatch-only built-in
- **AND** the `{prompt}` substitute-at-placeholder / positional-append-otherwise rule is stated

#### R11: The SPEC mirrors of every edited skill SHALL be updated
`docs/specs/skills/SPEC-_cli-agents.md` and `docs/specs/skills/SPEC-_cli-fab.md` MUST be swept for the same claims (Constitution Additional Constraints; code-quality.md § Sibling & Mirror Sweeps).

- **GIVEN** a change touching `src/kit/skills/_cli-agents.md` and `src/kit/skills/_cli-fab.md`
- **WHEN** review checks mirror sync
- **THEN** both `docs/specs/skills/SPEC-*.md` mirrors carry the matching updates

### Documentation sweep: specs

#### R12: The specs SHALL stop asserting agy is dispatch-only
`docs/specs/stage-models.md`, `docs/specs/config.md`, `docs/specs/architecture.md`, `docs/specs/glossary.md` and `docs/specs/harness-adapters.md` MUST be swept for dispatch-only phrasings, roster counts ("Two of the four are dispatch-only", "Only claude and codex ship an interactive_command"), and commented provider examples.

- **GIVEN** a grep for `dispatch-only`, `interactive_command`, and roster-count phrasings across `docs/specs/`
- **WHEN** the sweep is complete
- **THEN** every hit that named agy as dispatch-only now names kimi alone
- **AND** the commented provider examples show agy's `interactive_command`

#### R13: The pointer-delivery contract SHALL document the placeholder path
`docs/specs/harness-adapters.md` § 3 (interactive pane) MUST record that the pointer is delivered EITHER at a `{prompt}` placeholder inside the resolved `interactive_command` or, absent one, appended as the single quoted positional — shell-quoted on both paths.

- **GIVEN** a reader of the three-adapter contract
- **WHEN** they read how a pane worker receives its prompt
- **THEN** both delivery shapes are described, with the placeholder path named as the one a value-taking flag (agy's `-i`) requires

### Trust wall

#### R14: The agy trust wall SHALL be documented wherever agy's pane capability is described
Every site describing agy's pane capability MUST carry the caveat: a pane-dispatched agy worker in a not-yet-trusted worktree parks at agy's interactive trust dialog once per worktree until a human answers it (the wait-timeout peek classifies this as "waiting on genuine human input" and escalates without killing). The trust store is `~/.gemini/antigravity-cli/settings.json` `trustedWorkspaces`, EXACT-match absolute paths. Seeding it is explicitly out of scope.

- **GIVEN** a user enabling `dispatch.mode: pane` with `agent.workers: agy` in a fresh worktree
- **WHEN** they read the agy provider documentation
- **THEN** they learn the first dispatch parks at the trust dialog and that a human answers it once per worktree

### Tests

#### R15: The test suite SHALL pin agy's new capability, kimi's continued absence, and the `{prompt}` contract
`internal/agent/defaults_test.go` MUST assert agy carries an `interactive_command` (non-empty, `{model}`, no `{effort}`, trailing `-i {prompt}`) and kimi does not; `internal/spawn/spawn_test.go` MUST cover substitution, empty-drop, and the mode-interaction rule (R4/R5); `internal/dispatch`'s composition tests MUST cover both `WindowCommand` paths (R7); `internal/agent/agent_test.go`, `defaultprofiles_mirrors_test.go`, `cmd/fab/agent_test.go`, `cmd/fab/config_test.go`, `cmd/fab/config_show_init_test.go` and `cmd/fab/dispatch_start_test.go` MUST be updated wherever they pin agy's capability set or the rendered reference. The rpsr ride-along at `cmd/fab/config_test.go` ("three names known today" → four) MUST be corrected.

- **GIVEN** the repository test suite
- **WHEN** `go test ./...` runs under `src/go/fab`
- **THEN** every package passes and no assertion still claims agy ships no interactive command

### Pre-ship verification

#### R16: The residual live probes SHALL run before ship
P1 is RESOLVED (see § Probe results — bare `-i` hard-errors; the `{prompt}` placeholder is the user-chosen resolution). The residual probes MUST run against the installed `agy` (1.1.11) in a real tmux pane: **P1'** — the token-dropped session form `agy --dangerously-skip-permissions --model gemini-3.1-pro-high` (no `-i`) opens a usable plain session; **P2 remainder** — an `-i '<prompt>'` launch in this untrusted worktree still lands the prompt as the first user message after the trust dialog is answered. Both are observable pre-response, so agy's exhausted Starter quota (resets ~2026-08-16) does not block them; the result summary MUST record exactly what was and was not observable.

- **GIVEN** an installed agy 1.1.11 and a tmux server
- **WHEN** the token-dropped session form runs
- **THEN** a usable interactive session opens (P1' pass)
- **AND** with `-i '<prompt>'`, the prompt appears as the session's first user message after the trust dialog is accepted (P2 remainder)

### Follow-up bookkeeping

#### R17: The deferred kimi flip SHALL be recorded in the backlog
`fab/backlog.md` MUST carry the kimi `interactive_command` follow-up (gated on change `3oz7`'s readiness-gate/send-keys delivery redesign) so archiving `[agik]` after an agy-only ship does not silently drop it.

- **GIVEN** `[agik]` is archived after this agy-only change ships
- **WHEN** a reader scans the backlog for provider work
- **THEN** the kimi flip is still present as its own item with its `3oz7` gate recorded

### Non-Goals

- kimi's `interactive_command` — kimi has no interactive-initial-prompt flag at all, so even with `{prompt}` shipped there is no grammar to substitute into. Gated on change `3oz7`.
- `trustedWorkspaces` seeding at dispatch time — delivery choreography, belongs to `3oz7`.
- Changing `dispatch.mode` defaults or the descent ladder — under the default `native`, agy still resolves headless.
- A `{prompt}` placeholder in `headless_command` — headless delivery is stdin, not argv; the placeholder is an `interactive_command` construct only.
- kimi echo-behavior probing as a gate — opportunistic recording only.

### Design Decisions

#### `{prompt}` is a third placeholder resolved in a SEPARATE pass
**Decision**: `{prompt}` is recognized by `internal/spawn` alongside `{model}`/`{effort}`, but is resolved by its own exported pass (`spawn.WithPrompt` + `spawn.HasPrompt`) running AFTER `WithProfile`, not by extending `WithProfile`'s signature. `isTemplate` keeps keying on `{model}`/`{effort}` alone. The three placeholders share one internal substitute-or-drop helper, so the token-drop grammar has exactly one implementation.
**Why**: The two placeholders and the third have different lifetimes — model/effort are known at profile resolution, the pointer only at pane dispatch — and intake assumption #9 requires that a `{prompt}`-only command still receive appended `--model`/`--effort`. Folding `{prompt}` into `isTemplate` would silently suppress that append for such a command. Running the prompt pass last also means a pointer whose text happens to contain `{model}` is never re-substituted.
**Rejected**: A third parameter on `WithProfile` — it would force every one of the five call sites to name a prompt value they mostly do not have, and would couple the mode decision to the placeholder that must not influence it.
*Introduced by*: 260809-agik-agy-interactive-command

#### `-i {prompt}` is load-bearing, not cosmetic
**Decision**: The agy `interactive_command` ends with the `-i {prompt}` pair.
**Why**: `-i`/`--prompt-interactive` is a value-taking flag (probe P1: bare `-i` exits 2 with `flag needs an argument`). Pane dispatch substitutes the shell-quoted pointer at `{prompt}`, so the pointer becomes the flag's value and is delivered as the session's first user message. Session launches substitute empty, and the empty-value token-drop removes the pair, leaving a plain session. One grammar, both seams.
**Rejected**: A grammar without `-i` (the pre-existing rpsr posture) — nominally pane-capable while orphaning every tmux-dispatched stage. A trailing bare `-i` — hard-errors every session launch (P1).
*Introduced by*: 260809-agik-agy-interactive-command

#### `fab resolve-agent`'s `dispatch=` line leaves `{prompt}` unsubstituted
**Decision**: The `dispatch=` line for a pane rung prints the profile-resolved command with `{prompt}` still visible.
**Why**: The pointer does not exist at resolve time (`fab dispatch start` writes the prompt file and derives the pointer itself), and dispatch sites never execute the `dispatch=` value. A visible `{prompt}` honestly marks where the pointer goes.
**Rejected**: Token-dropping it at resolve time — that would print a session-shaped command that is NOT what `fab dispatch` runs, which is actively misleading.
*Introduced by*: 260809-agik-agy-interactive-command

#### The trust wall is documented, not seeded
**Decision**: agy's once-per-worktree interactive trust dialog is accepted and caveated in the docs; fab does not write `~/.gemini/antigravity-cli/settings.json`.
**Why**: Seeding a third-party CLI's settings file at dispatch time is delivery choreography, and the pipeline's wait-timeout peek already classifies a parked trust dialog as "waiting on genuine human input" and escalates without killing.
**Rejected**: Read-modify-write of agy's trust store at dispatch time — it belongs to the `3oz7` readiness-gate change, where the pane delivery path is being redesigned anyway.
*Introduced by*: 260809-agik-agy-interactive-command

## Tasks

### Phase 1: Spawn composition — the `{prompt}` placeholder

- [x] T001 In `src/go/fab/internal/spawn/spawn.go`, extract the existing `{model}`/`{effort}` substitute-or-drop body of `resolveTemplate` into a shared token-walking helper parameterized by (placeholder, value) pairs, leaving `resolveTemplate` a thin caller. No behavior change. <!-- R4 -->
- [x] T002 Add `promptPlaceholder = "{prompt}"` plus exported `HasPrompt(cmd string) bool` and `WithPrompt(cmd, prompt string) string` built on that helper; document that `{prompt}` is deliberately absent from `isTemplate` (R5) and that the pass runs AFTER `WithProfile`. <!-- R4 R5 -->
- [x] T003 Resolve `{prompt}` to empty at the sole promptless session seam: `cmd/fab/agent.go` (`runAgent`). <!-- R6 --> <!-- rework: originally also applied WithPrompt(…, "") to operator.go and batch.go — those are prompt-CARRYING seams; T027 reverses that and rewires them onto the shared delivery helper -->
- [x] T004 Update `internal/dispatch/dispatch.go`'s `WindowCommand` to substitute the shell-quoted pointer at `{prompt}` when present and keep the positional append byte-for-byte otherwise; update its doc comment to describe both delivery shapes. <!-- R7 -->

### Phase 2: Provider data and Go

- [x] T005 Add `interactive_command: 'agy --dangerously-skip-permissions --model {model} -i {prompt}'` to the `agy` row in `src/go/fab/internal/agent/defaults.yaml`, and rewrite that file's providers-block note so kimi is the sole no-interactive-command built-in (with agy's `-i {prompt}` rationale and the trust-wall caveat). <!-- R1 R2 -->
- [x] T006 Add the exported `DefaultAgyInteractiveCommand` var in `src/go/fab/internal/agent/agent.go`, sourced from `defaultProviders[providerAgy].InteractiveCommand`, and rewrite the surrounding comment blocks (the non-claude command block and the `defaultProviders` doc comment) to kimi-only dispatch-only phrasing. <!-- R3 -->
- [x] T007 Update `src/go/fab/internal/configref/configref.go`: render agy's `# interactive_command:` line from `agent.DefaultAgyInteractiveCommand` above its `headless_command`, drop the `# dispatch only;` annotation from agy's headless line, and sweep the block's surrounding prose (the providers `Description`, the built-ins narrative, and the `agy silently DROPS a positional prompt` passage) for dispatch-only claims about agy. <!-- R8 -->

### Phase 3: Documentation sweep

- [x] T008 [P] Sweep `src/kit/skills/_cli-agents.md`: the session-form roster note, the built-ins roster line, § Spawn Composition's placeholder/token-drop paragraph (add `{prompt}`), § agy's Interactive row + Approvals row + Dispatch-only section (replace the "add one in your own config" recipe with the shipped grammar and the trust-wall caveat), and kimi's § Dispatch-only (drop "Like agy"). <!-- R9 R14 -->
- [x] T009 [P] Sweep `src/kit/skills/_cli-fab.md`: the `providers:` four-built-ins paragraph and dispatch-only-posture rationale, the `fab agent` error-reachability note, and the pane-composition description in § `fab dispatch` (the `{prompt}` path). <!-- R10 R14 -->
- [x] T010 Update the SPEC mirrors `docs/specs/skills/SPEC-_cli-agents.md` and `docs/specs/skills/SPEC-_cli-fab.md` for every claim changed in T008–T009. <!-- R11 -->
- [x] T011 [P] Sweep `docs/specs/stage-models.md`: the inline provider YAML, the "Two of the four are dispatch-only" paragraph, the capability-summary line, and the commented provider examples. <!-- R12 -->
- [x] T012 [P] Sweep `docs/specs/config.md`, `docs/specs/architecture.md` and `docs/specs/glossary.md` for dispatch-only phrasings and commented provider examples naming agy. <!-- R12 -->
- [x] T013 Update `docs/specs/harness-adapters.md` § 3 / § pane composition to record the two pointer-delivery shapes (`{prompt}` substitution vs positional append), both shell-quoted. <!-- R13 -->
- [x] T014 Verify the trust-wall caveat is present at every site describing agy's pane capability (skills, SPEC mirrors, specs, Go comments) after T005–T013, and add it where missing. <!-- R14 -->
- [x] T015 Repo-wide re-grep for `dispatch-only`, `dispatch only`, `interactive_command`, and roster-count phrasings (`only claude and codex`, `two of the four`, `two dispatch`) across `src/`, `docs/specs/`, and `docs/memory/`; update any surviving agy claim outside `fab/changes/` (memory files are hydrate's, but note them). <!-- R12 --> <!-- rework cycle 2: the grep class was too literal — it missed "ship a headless_command only" (SPEC-_cli-agents.md:59) and the superseded session-vs-pane axis phrasings ("dropped … at a session launch": glossary.md:86, stage-models.md:474-475). Re-sweep including capability paraphrases and the OLD AXIS wordings; the correct axis is promptless (bare fab agent) vs prompt-carrying (operator, batch new/switch, pane dispatch) -->

### Phase 4: Tests

- [x] T016 Extend `src/go/fab/internal/spawn/spawn_test.go`: `{prompt}` substitution (whitespace preserved), empty-value token-drop of the `-i {prompt}` pair, the mode-interaction rule (a `{prompt}`-only command still takes append mode), and `HasPrompt` on both shapes. <!-- R15 R4 R5 -->
- [x] T017 Extend the `internal/dispatch` composition tests: `WindowCommand` substitutes at `{prompt}` (shell-quoted) and appends positionally byte-for-byte when the placeholder is absent. <!-- R15 R7 -->
- [x] T018 Update `src/go/fab/internal/agent/defaults_test.go`: flip the interactive-command assertions to kimi-only and add agy's positive assertions (non-empty, carries `{model}` and `{prompt}`, no `{effort}`, ends in `-i {prompt}`); add `DefaultAgyInteractiveCommand` to the exported-var mirror table. <!-- R15 R2 -->
- [x] T019 Update `src/go/fab/internal/agent/agent_test.go` (the session-command absence table) and `defaultprofiles_mirrors_test.go` wherever they pin agy's capability set. <!-- R15 -->
- [x] T020 Update `src/go/fab/cmd/fab/agent_test.go` so the dispatch-only built-ins error test covers kimi alone, and add coverage that `fab agent --provider agy --print` composes a session command with the `-i {prompt}` pair dropped. <!-- R15 R6 -->
- [x] T021 Update `src/go/fab/cmd/fab/config_test.go`: the rendered-reference assertions for agy's new `interactive_command` line and block ordering, plus the rpsr ride-along ("three names known today" → four). <!-- R15 -->
- [x] T022 [P] Update `src/go/fab/cmd/fab/config_show_init_test.go` and `src/go/fab/cmd/fab/dispatch_start_test.go` wherever they pin agy's capability set or the rendered reference. <!-- R15 -->
- [x] T023 Run `go test ./...` from `src/go/fab` (and `src/go/fab-kit` if touched); fix failures until green. <!-- R15 -->

### Phase 5: Live probes and bookkeeping

- [x] T024 Run probe P1' in a real tmux pane: `agy --dangerously-skip-permissions --model gemini-3.1-pro-high` (no `-i`) opens a usable plain session. Record exactly what was observable. <!-- R16 -->
- [x] T025 Run the P2 remainder in a real tmux pane: `agy … -i '<test prompt>'` in this untrusted worktree; answer the trust dialog and verify the prompt lands as the first user message. Record echo-on-typed-input / Enter-submit observations for `3oz7` as a free rider. <!-- R16 -->
- [x] T026 Add a backlog item to `fab/backlog.md` for the deferred kimi `interactive_command` flip, recording its `3oz7` gate and the rpsr never-ship-a-parking-pane-capability rationale. <!-- R17 -->

### Phase 6: Rework cycle 1 — prompt-carrying launch seams (review must-fix + should-fixes)

- [x] T027 Extract the two-shape prompt-delivery branch into ONE shared exported helper in `src/go/fab/internal/spawn` (e.g. `spawn.DeliverPrompt(cmd, prompt string) string`: substitute the shell-quoted prompt at `{prompt}` when `HasPrompt(cmd)`, else `cmd + " " + <shell-quoted prompt>`); rewire `internal/dispatch.WindowCommand` (dispatch.go:483) onto it, and rewire the three prompt-carrying seams — `cmd/fab/operator.go:92` (`'/fab-operator'`), `cmd/fab/batch_new.go:139` (`/fab-new …`), `cmd/fab/batch_switch.go:140` (`/fab-switch …`) — removing their now-wrong `spawn.WithPrompt(…, "")` empty-drop calls (operator.go:148, batch.go:36). Decide and record the helper's exact home/signature as a graded assumption. <!-- R6 R7 -->
- [x] T028 Ship the seam tests (A-027): operator/batch composition tests pinning both shapes — a `{prompt}`-carrying provider gets its `/fab-operator` / `/fab-new …` / `/fab-switch …` prompt substituted shell-quoted at the placeholder; a placeholder-free provider composes byte-identical to the pre-change positional form; plus shared-helper unit tests (quote-safety on both paths). <!-- R6 R15 -->
- [x] T029 `src/kit/skills/fab-operator.md` step 6 (~line 434): alongside the one-prompt/no-`&&`-chaining sub-rule pointer, name the `{prompt}` placeholder caveat (pointer to the owning section, not a restatement — owner-or-pointer convention); update the step-5 `fab agent --print` recipe if the shared helper changes what the operator must do with the printed command; ride the SPEC mirror `docs/specs/skills/SPEC-fab-operator.md` if present. <!-- R9 R11 -->
- [x] T030 `src/go/fab/internal/configref/configref.go:653-660` — fix the `interactive_command` field bullet: name `{prompt}`, state that only `{model}`/`{effort}` select template mode (a `{prompt}`-only command still takes append mode), and align it with the field's `Description` string (configref.go:516) so the two renderings agree. <!-- R8 -->
- [x] T031 (optional nice-to-have) `src/kit/skills/_cli-agents.md:187` — reword the agy Approvals row parenthetical to present-truth (drop the "fab now ships both" transition narration). <!-- R9 -->

### Phase 7: Rework cycle 2 — sweep stragglers (review must-fix + should-fixes)

- [x] T032 Fix the four must-fix doc stragglers: `docs/specs/skills/SPEC-_cli-agents.md:59` (agy ships both command fields; kimi alone is headless-only — match that file's own lines 37/55/56), `docs/specs/glossary.md:86` and `docs/specs/stage-models.md:474-475` (re-state `{prompt}` on the promptless vs prompt-carrying axis — stage-models currently contradicts its own lines 277-281), and `src/go/fab/internal/configref/configref.go:734-737` (align the agy narrative with the corrected field bullet at :663-667 and Description at :516 — all three renderings must agree). Re-check A-008/A-009/A-025/A-031 after. <!-- R11 R12 R8 -->
- [x] T033 Fix the superseded-axis wording in test comments/messages: `internal/agent/defaults_test.go:82-83` (kimi-only exception) and `:185` (token-drop happens at the PROMPTLESS launch), `cmd/fab/config_test.go:406-408` (kimi alone is dispatch-only), `internal/spawn/spawn_test.go:349-351` (rename the empty-drop case to the promptless seam; drop operator/batch from the comment). <!-- R15 -->
- [x] T034 Close the operator step-6 remedy gap: state PLAINLY in `src/kit/skills/fab-operator.md` step 6 (and the owning `_cli-agents.md:86` bullet) that a `{prompt}`-carrying session provider is UNSUPPORTED for operator per-change worker spawns until the `3oz7` readiness-gate/send-keys delivery change ships (which removes spawn-time delivery entirely) — do NOT add a new `fab agent --print` mode (out of scope). SPEC mirrors ride along. <!-- R9 R11 -->
- [x] T035 (optional) `internal/dispatch/dispatch.go:226` — replace the private `shellQuote` duplicate with `internal/shellquote.Single` where safe (dispatch already imports spawn now), and add the one-line comment at `cmd/fab/dispatch_start.go:329` noting `spawn_cmd` persists the profile-resolved command, not the delivered one. <!-- R15 -->

## Execution Order

- T001 blocks T002; T002 blocks T003, T004, T016, T017.
- T005 blocks T006, T007 (the exported var and rendered reference read the parsed data).
- T008–T013 are parallelizable across distinct files; T010 depends on T008–T009; T014–T015 run after them as verification sweeps.
- T016–T022 follow their subject edits; T023 runs last in Phase 4.
- T024–T025 are live probes with no code dependency — they may run at any point, but their outcomes are reported in the apply result, so run them after the code is green.
- Phase 6 (rework cycle 1): T027 blocks T028; T029–T031 are parallelizable doc edits; re-run T023 (full `go test ./...`) after T027–T028.

## Acceptance

### Functional Completeness

- [x] A-001 R1: `src/go/fab/internal/agent/defaults.yaml` gives agy an `interactive_command` and kimi still has none.
- [x] A-002 R2: The shipped agy interactive grammar carries `--dangerously-skip-permissions`, `{model}` and `{prompt}`, carries no `{effort}`, and ends in `-i {prompt}`.
- [x] A-003 R3: `agent.DefaultAgyInteractiveCommand` exists, is sourced from the parsed defaults (no literal duplicate), and `internal/agent` prose names kimi as the sole dispatch-only built-in.
- [x] A-004 R4: `internal/spawn` recognizes `{prompt}` with substitute-or-drop semantics through an exported predicate and resolver.
- [x] A-005 R8: `fab config explain providers` renders agy's `# interactive_command:` line above its `headless_command`, interpolated from the agent var. *(Verified by running the built binary.)*
- [x] A-006 R9: `src/kit/skills/_cli-agents.md` § agy documents the shipped interactive grammar instead of an "add one yourself" recipe, and § Spawn Composition documents `{prompt}`.
- [x] A-007 R10: `src/kit/skills/_cli-fab.md` names kimi as the sole dispatch-only built-in in both swept sites and states the `{prompt}` rule.
- [x] A-008 R11: `docs/specs/skills/SPEC-_cli-agents.md` and `SPEC-_cli-fab.md` carry the matching updates. *(cycle 2: :59 now reads "codex and agy ship both command fields … kimi alone ships a `headless_command` only"; the § Spawn Composition row also carries the operator UNSUPPORTED remedy, and `SPEC-fab-operator.md` rides along.)*
- [x] A-009 R12: No file under `docs/specs/` asserts agy is dispatch-only or that only claude and codex ship an `interactive_command`. *(cycle 2: re-swept with the widened class — capability paraphrases plus the old session-vs-pane axis. Remaining `docs/specs/` hits for "dispatch-only" are kimi-scoped or ladder-reason strings.)*
- [x] A-010 R13: `docs/specs/harness-adapters.md` records both pointer-delivery shapes.
- [x] A-011 R14: The trust-wall caveat (once-per-worktree dialog, exact-match `trustedWorkspaces`, seeding out of scope) appears wherever agy's pane capability is described.
- [x] A-012 R17: `fab/backlog.md` carries the deferred kimi flip with its `3oz7` gate (`[ki9v]`).

### Behavioral Correctness

- [x] A-013 R6: `fab agent --provider agy --print` composes `agy --dangerously-skip-permissions --model <id>` — the `-i {prompt}` pair is dropped, no placeholder survives.
- [x] A-014 R7: A pane dispatch of an agy stage composes `… -i '<shell-quoted pointer>'`, and a claude/codex pane dispatch composes the byte-identical positional-append form it did before this change.
- [x] A-015 R5: A command carrying only `{prompt}` still takes `WithProfile`'s append mode (`--model`/`--effort` appended).
- [x] A-016 R1: With no `providers:` config, `dispatch.mode: pane` inside tmux selects the pane rung for an agy-resolved stage rather than descending to headless.

### Scenario Coverage

- [x] A-017 R15: `go test ./...` passes under `src/go/fab` with the updated assertions. *(All 32 packages `ok`; `gofmt -l` clean, `go vet` clean.)*
- [x] A-018 R16: Probes P1' and the P2 remainder were run live against agy 1.1.11 in a tmux pane and their outcomes are recorded in the apply result.

### Edge Cases & Error Handling

- [x] A-019 R4: An empty `{prompt}` value drops the placeholder token AND its preceding `-`-flag token, matching the documented `{model}`/`{effort}` grammar limits.
- [x] A-020 R15: `fab agent --provider kimi` still errors with the config-key hint — kimi's dispatch-only posture is untouched.

### Code Quality

- [x] A-021 Pattern consistency: New Go symbols and comments follow the surrounding `internal/agent` / `internal/spawn` conventions (data in `defaults.yaml`, exported vars sourced from the parsed table, one token-drop implementation).
- [x] A-022 No unnecessary duplication: The agy interactive command string appears as a literal in exactly one place (`defaults.yaml`); every other site interpolates it. The token-drop walk exists once.
- [x] A-023 Canonical source only: All skill edits are under `src/kit/skills/`; nothing under `.claude/skills/` was touched (the deployed copies still differ from source, i.e. they carry the pre-change kit).
- [x] A-024 SPEC-mirror sync: Every edited `src/kit/skills/*.md` has its `docs/specs/skills/SPEC-*.md` mirror updated in this change.
- [x] A-025 Sibling sweep: The dispatch-only/roster-count class is now fully swept repo-wide. *(Honestly: it took two corrective cycles, not one up-front pass — cycle 2 closed `SPEC-_cli-agents.md:59`, `glossary.md:86`, `stage-models.md:475`, `configref.go:737`, `defaults_test.go:82,185`, `config_test.go:407`, `spawn_test.go:349`, plus two the re-review had not listed: `agent_test.go:1133,1136` and `spawn.go:147-152`. The cycle-2 grep class was widened to capability paraphrases ("ship a headless_command only", "no interactive_command") and old-axis wordings ("at a session launch"). `docs/memory/` stragglers are hydrate's — listed in the apply result.)*
- [x] A-026 CLI ⇒ docs + tests: the composition-behavior change carries `_cli-fab.md` updates and test updates.
- [x] A-027 Go changes ship tests: operator/batch seam composition tests exist and pin both delivery shapes (T028) — every changed `.go` seam carries accompanying test updates.
- [x] A-028 No migration needed: The change is additive built-in provider data plus a backward-compatible composition rule, and restructures no user data.
- [x] A-029 R6: With a `{prompt}`-carrying session provider, the operator launcher and `fab batch new`/`switch` worker spawns deliver their initial prompts as the `{prompt}` substitution (shell-quoted; agy: as `-i`'s value); with a placeholder-free provider the composed strings are byte-identical to the pre-change positional-append forms.
- [x] A-030 R6/R7: The two-shape delivery grammar exists in exactly ONE implementation (the shared helper), consumed by `WindowCommand`, the operator launcher, and both batch spawns — no site hand-rolls the branch.
- [x] A-031 R8: `configref.go`'s `interactive_command` field bullet and its `Description` string agree — both name `{prompt}` and state that only `{model}`/`{effort}` select template mode. *(cycle 2: the agy narrative in the rendered block now agrees with both, on the same promptless-vs-prompt-carrying axis — all three renderings match.)*

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- Memory files (`docs/memory/runtime/providers-and-profiles.md`, `runtime/dispatch.md`, `_shared/configuration.md`) are hydrate's, not apply's — the intake's Affected Memory list carries them.

### Probe results — 2026-08-09, agy 1.1.11, tmux 3.6a, worktree `fab-kit.worktrees/agik` (untrusted)

**P1 (session seam) — FAILED. This is the intake's blocking contingency.**

`agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i` with no trailing value does NOT open a session. It exits **2** immediately with:

```
flag needs an argument: -i
Usage of agy:
  ...
```

`-i` / `--prompt-interactive` is a **value-taking flag**, not a boolean. No session, no TUI, no trust dialog — a usage dump and a non-zero exit.

**Consequence**: the session seam and the pane seam genuinely conflict in one grammar. `fab agent --provider agy` and `fab operator` agy spawns compose `interactive_command` with **no** appended positional, so a shipped trailing `-i` would make every agy session launch hard-error at startup. All tasks stopped at T001 per the intake's "do not silently pick a fallback grammar" instruction.

**P2 (pane seam) — precondition confirmed, verification not completed.**

The same command **with** a pointer-style positional (`… -i 'Read the file at <path> and follow it'`) is accepted and launches the interactive TUI, which parks on agy's workspace-trust dialog:

```
Accessing workspace:
/home/sahil/code/sahil87/fab-kit.worktrees/agik
Do you trust the contents of this project?
Antigravity CLI requires permission to read, edit, and execute files here.
> Yes, I trust this folder
  No, exit
```

This reproduces the documented once-per-worktree trust wall exactly. The dialog was **not answered** (answering writes a permanent `trustedWorkspaces` entry, and the change is blocked), so "the `-i` prompt survives the trust dialog" is **still unverified** — as is the echo-on-typed-input / Enter-submit free-rider observation for `3oz7`. The trust store was left unmodified.

**P3 (quota) — not reached.** Both probes resolved before any model call, so the exhausted Starter quota did not affect either outcome. P1's verdict is quota-independent (argument parsing) and stands regardless of the ~2026-08-16 reset.

**Resolution (2026-08-09)**: the user chose the `{prompt}` placeholder (intake § 1a). P1 is closed; the residual probes are P1' (token-dropped session form opens a usable session) and the P2 remainder (prompt survives the trust dialog) — see R16 / T024–T025.

### Residual probe results — 2026-08-09, agy 1.1.11, tmux 3.6a, worktree `fab-kit.worktrees/agik`

Run on a dedicated tmux socket (`-L fabprobe-agik`) so no user pane was disturbed. The worktree was **untrusted** at the start; the trust store was snapshotted before and **restored byte-identical afterwards**, so the machine is as found (`trustedWorkspaces` carries no `agik` entry).

**Composition, verified against the built worktree binary** (not the installed one):

```
$ fab agent --provider agy --print
agy --dangerously-skip-permissions
$ fab agent --provider agy --model gemini-3.1-pro-high --print
agy --dangerously-skip-permissions --model gemini-3.1-pro-high
$ FAB_DISPATCH_MODE=pane TMUX=x fab resolve-agent doing --provider agy
model=gemini-3.1-pro-high
provider=agy
dispatch=agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i {prompt}
```

The session seam drops the `-i {prompt}` pair as designed; the `dispatch=` line keeps `{prompt}` verbatim per the Design Decision above.

**P2 remainder — PASS. The `-i` prompt survives the trust dialog.**

Launched untrusted with `-i 'XYZTEST-AGIK-PROBE: reply with the single word PONG and nothing else'`. The workspace-trust dialog appeared exactly as before; `Enter` on `> Yes, I trust this folder` cleared it, and the session opened with the prompt already delivered as its **first user message**:

```
> XYZTEST-AGIK-PROBE: reply with the single word PONG and nothing else

⚠ Individual quota reached. Please upgrade your subscription to increase your limits. Resets in 149h45m54s.
```

The trust wall **defers, does not eat**, the prompt. The quota banner appears *after* the prompt visibly lands, so delivery is confirmed and the exhausted Starter quota (resets ~2026-08-16) did not obstruct the observation. Not observable (quota-blocked): the model's actual response.

**P1' — PASS. The token-dropped session form opens a usable plain session.**

`agy --dangerously-skip-permissions --model gemini-3.1-pro-high` (no `-i`) opened the TUI directly: banner reads `Gemini 3.1 Pro (High)`, cwd correct, live input box at `>`. Typed text (`P1PRIME-USABLE`) echoed into the buffer, the pane stayed alive (`dead=0`), and **no quota banner appeared** — no model call is made until a turn is submitted, so this verdict is quota-independent.

**`3oz7` free-rider — echo-on-typed-input and Enter-submit both confirmed.** A sentinel sent with **no** Enter (`ECHOPROBE123`) appeared immediately inside agy's bordered input box, so the buffer is live and the printed-prompt trap does not apply here. A subsequent `Enter` submitted it (`⣻ Generating...` + `esc to cancel`). Note for `3oz7`: agy's input box is **bordered and full-width**, and long input **wraps inside it** — the same wrapping shape that broke `fab dispatch deliver`'s echo-verify on kimi.

## Deletion Candidates

- `src/go/fab/internal/spawn/spawn.go` `resolveTemplate` — after T001 extracted the walk into `substitute`, this is a 4-line pass-through with exactly one call site (`WithProfile`); inlining the two `placeholderSub` literals there would remove the indirection without touching the grammar. (Carried from cycle 2; deliberately not taken — the named function documents the profile-placeholder pair.)
- `src/go/fab/internal/dispatch/dispatch.go` `WindowCommand` — now a one-line delegate to `spawn.DeliverPrompt` with a single call site (`cmd/fab/dispatch_start.go:471`). It survives as the pane adapter's named seam (and carries the pane-specific quoting rationale), but it adds no behavior of its own.
- *(Already deleted by this change, recorded for completeness: `internal/dispatch`'s private `shellQuote` — the cycle-2 candidate, taken by T035 in favour of `internal/shellquote.Single`; agy's "add a `providers.agy.interactive_command` in your own config" recipe in `_cli-agents.md`; and the `# dispatch only;` annotation on agy's rendered `headless_command` line.)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `{prompt}` is resolved by a SEPARATE exported pass (`spawn.WithPrompt`/`HasPrompt`) running after `WithProfile`, sharing one internal token-drop helper with `{model}`/`{effort}`; `isTemplate` is untouched | Intake assumption 9 requires a `{prompt}`-only command still take append mode; a third `WithProfile` parameter would couple the mode decision to the placeholder that must not influence it and force five call sites to name a value they lack | S:85 R:75 A:90 D:85 |
| 2 | Certain | `WindowCommand` branches on placeholder presence: substitute the shell-quoted pointer at `{prompt}`, else today's positional append byte-for-byte | Intake § 1a states the back-compat requirement as hard; the branch is the minimal form that satisfies it | S:90 R:80 A:95 D:90 |
| 3 | Confident | `fab resolve-agent`'s `dispatch=` line leaves `{prompt}` unsubstituted rather than token-dropping it | The pointer does not exist at resolve time and dispatch sites never execute the value; dropping it would print a command that is not what `fab dispatch` runs | S:65 R:85 A:80 D:70 |
| 4 | Certain | `fab agent` is the ONLY seam taking the empty-prompt pass; the operator launcher and `fab batch new`/`switch` are prompt-CARRYING seams that deliver via the shared two-shape helper (rework cycle 1 — the original classification of all three as promptless was the review's must-fix regression) | Review verified against the composed output: operator/batch append `/fab-operator`//`/fab-new …`//`/fab-switch …` positionally, which a `{prompt}`-carrying provider discards | S:90 R:80 A:90 D:90 |
| 5 | Confident | `{prompt}` is an `interactive_command` construct only — no `headless_command` support, no validation that a pane-capable command carries one | Headless delivery is stdin; the resolver's document-don't-validate philosophy forbids rejecting a `{prompt}`-free interactive command (claude/codex have none and must keep working) | S:75 R:80 A:85 D:75 |
| 6 | Certain | Probes are the LAST phase, not the first: P1 is resolved, and the residuals verify a shipped grammar rather than gate its choice | The intake's P1 hard gate closed when the user chose the `{prompt}` resolution; the residual probes cannot change the grammar, only confirm it | S:85 R:80 A:90 D:85 |
| 7 | Confident | The kimi follow-up is recorded in `fab/backlog.md` during apply rather than deferred to ship/archive | Intake assumption 7 requires it be recorded so it is not lost; doing it now is idempotent and cannot be dropped by an interrupted ship | S:70 R:90 A:80 D:75 |
| 8 | Confident | `docs/specs/glossary.md` and `docs/specs/harness-adapters.md` are in the sweep class beyond the intake's named sites | Intake says its list is not exhaustive and instructs a repo-wide grep; glossary restates the `providers` capability grammar and harness-adapters owns the pointer-delivery contract | S:75 R:85 A:85 D:80 |
| 9 | Confident | Memory files are left to hydrate | Intake lists them under Affected Memory; the pipeline's hydrate stage owns `docs/memory/` writes | S:80 R:85 A:90 D:85 |
| 10 | Certain | The shared two-shape helper is `spawn.DeliverPrompt(spawnCmd, prompt string) string` in `internal/spawn` — it shell-quotes the prompt itself (via `internal/shellquote.Single`) and branches on `HasPrompt`; `dispatch.WindowCommand` becomes a one-line delegate and the three prompt-carrying seams call it directly | `internal/spawn` already owns the placeholder grammar (`HasPrompt`/`WithPrompt`) and both consumer packages already import it, so no new dependency edge or cycle; putting the quoting INSIDE the helper is what makes the two paths provably consistent (a caller that quoted only one path is the bug class this rework exists to kill), and `shellquote.Single` is byte-identical to `dispatch`'s private `shellQuote`, so the positional form stays byte-for-byte | S:85 R:85 A:90 D:90 |
| 11 | Confident | The empty-prompt case stays on `WithPrompt`, NOT on `DeliverPrompt` — the helper takes only real prompts | A promptless seam must DROP the flag pair, not deliver `''`; folding both into one entry point would make `DeliverPrompt(cmd, "")` silently mean "drop", one keystroke away from the delivery bug just fixed. `fab agent` is the sole caller of the empty path | S:75 R:80 A:85 D:80 |
| 12 | Confident | The `{prompt}` axis is re-documented as promptless-vs-prompt-carrying (not session-vs-pane) across skills, SPEC mirrors, specs and Go prose | The old axis is now factually wrong: `fab operator`/`fab batch` are session launches that DO carry a prompt. The class sweep covered `_cli-agents.md`, `_cli-fab.md`, both SPEC mirrors, `harness-adapters.md`, `stage-models.md`, `architecture.md`, `defaults.yaml`, `agent.go`, `defaults_test.go`, `configref.go` | S:80 R:85 A:85 D:80 |

12 assumptions (5 certain, 7 confident, 0 tentative).
