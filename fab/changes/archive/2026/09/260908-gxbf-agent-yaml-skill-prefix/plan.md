# Plan: Fold skill prefix into `fab agent` resolution; retire `fab skill-prompt`

**Change**: 260908-gxbf-agent-yaml-skill-prefix
**Intake**: [intake.md](intake.md)

## Requirements

### Runtime: Skill prefix ownership

#### R1: One Go owner for the prefix rule
`internal/agent.SkillPrefix(provider)` SHALL return `$` for the exact provider name `codex` and `/` for every other name, including built-in, custom, unknown and empty. `internal/agent.SkillPrompt` SHALL compose `<prefix><skill>[ <arguments>]` through `SkillPrefix` and MUST NOT carry its own copy of the rule. Launcher behaviour from `260908-tfzn` (batch new/switch, built-in operator launcher, Claude-fallback provider) SHALL be unchanged.

- **GIVEN** provider `codex` **WHEN** `SkillPrefix` is called **THEN** it returns `$`
- **GIVEN** provider `claude`, `agy`, `kimi`, `custom` or `""` **WHEN** `SkillPrefix` is called **THEN** it returns `/`
- **GIVEN** `agent.session: codex` **WHEN** `fab batch new`/`switch` or the fallback operator launcher opens a window **THEN** the final argv prompt still starts with `$` (existing launcher regression stays green)

#### R2: `fab agent -o yaml` exposes `skill_prefix`
The `-o yaml` document SHALL carry a `skill_prefix` key whose value is `SkillPrefix(<resolved provider>)`. The key SHALL always be present (never `omitempty`), SHALL be the last key of the document whether or not `dispatch:` is present, and SHALL reflect the post-override, post-fallback provider (so `--provider codex` yields `$`). The `fab resolve-agent` line protocol (`Resolution.Lines`) MUST NOT gain a prefix line.

- **GIVEN** a resolution whose provider is `codex` **WHEN** `fab agent <selector> -o yaml` (or `--provider codex -o yaml`) runs **THEN** the document ends with `skill_prefix: $`
- **GIVEN** a resolution whose provider is `oracle` with `dispatch:` present **WHEN** `-o yaml` runs **THEN** `skill_prefix: /` follows the `dispatch:` block as the final key
- **GIVEN** `Resolution{Provider: "codex", SkillPrefix: "$"}` **WHEN** `Lines(false)` renders **THEN** no `skill_prefix` line is emitted

### CLI: Retire `fab skill-prompt`

#### R3: The subcommand is removed
`fab skill-prompt` SHALL no longer exist: `cmd/fab/skill_prompt.go` and its CLI tests are deleted, the `main.go` registration is removed, and `fab skill-prompt …` returns cobra's unknown-command error. The launcher regression `TestSkillPromptLaunchers` SHALL be retained in a launcher-scoped test file. No alias, shim or migration is added (the command never shipped).

- **GIVEN** the built binary **WHEN** `fab skill-prompt --provider codex fab-fff` runs **THEN** it exits non-zero with an unknown-command error
- **GIVEN** `go test ./...` under `src/go/fab` **WHEN** it runs **THEN** the launcher regression still exercises `new`/`switch`/`operator` for codex and non-codex providers and passes

### Skills: Prose owner and pointers

#### R4: `_cli-fab.md` reflects the CLI surface
`src/kit/skills/_cli-fab.md` SHALL drop the `## fab skill-prompt` section and its Contents entry, SHALL document `skill_prefix` in the § fab agent `-o yaml` key table, and SHALL reword the § fab batch and § fab operator sentences that name `fab skill-prompt` to name `internal/agent.SkillPrompt`. The help bundle (`docs/site/skill.md` canonical → `src/go/fab/cmd/fab/skill.md` via `scripts/sync-skill.sh`) SHALL drop the Skill-prompts bullet and stay byte-identical.

- **GIVEN** the edited `_cli-fab.md` **WHEN** grepped for `skill-prompt` **THEN** there are no hits
- **GIVEN** the edited help bundle **WHEN** `skill_test.go`'s drift guard runs **THEN** it passes

#### R5: `_cli-agents.md` § Skill Prompts is the short prose owner
`src/kit/skills/_cli-agents.md` § Skill Prompts SHALL state: the prefix belongs to the receiving harness (`codex` ⇒ `$`, else `/`; owner `agent.SkillPrefix`, exposed as `skill_prefix`); receiver identity comes from `fab agent default -o yaml --repo <target-repo>` for fresh default-role spawns, from that role's resolution for other fresh roles, and from § Peek's process tree for existing panes (unknown ⇒ `/`); the composed prompt is shell-quoted once as one token per § Spawn Composition; only explicit skill invocations are transformed. The `fab_skill_token`/`eval` recipe and the trailing-newline paragraph SHALL be deleted. `src/kit/skills/fab-operator.md` steps 6–7 SHALL point at that section and the tmux example SHALL use a one-token quoted prompt variable instead of `$fab_skill_token`; its other § Skill Prompts pointers stay.

- **GIVEN** the edited `_cli-agents.md` **WHEN** read **THEN** § Skill Prompts contains no `fab skill-prompt`, `eval`, or `fab_skill_token` text and still contains the message-types-distinct rule
- **GIVEN** the edited `fab-operator.md` **WHEN** grepped for `fab_skill_token` or `--repo` mode of the renderer **THEN** there are no hits

### Specs: Sibling sweep

#### R6: Specs name the resolver key, not the retired command
`docs/specs/glossary.md`, `architecture.md`, `skills.md` and `stage-models.md` SHALL no longer name `fab skill-prompt`; `architecture.md` and `skills.md` SHALL describe the prefix as owned by `internal/agent.SkillPrompt`/`SkillPrefix` and exposed as `skill_prefix`; `stage-models.md`'s YAML key list SHALL include `skill_prefix`. Sentence-level edits only (Constitution VI).

- **GIVEN** the repo outside `.claude/`, `.agents/` and `fab/changes/` **WHEN** grepped for `skill-prompt`, `fab_skill_token`, `fab_skill_prompt` **THEN** the only remaining hits are in `docs/memory/` (rewritten by hydrate)

### Non-Goals

- `change.go` `defaultCommand` slash-form `Next:` suggestions — human-facing; follow-up candidate
- run-kit / delegated `rk operator` kickoff; provider config keys; launcher wiring from `260908-tfzn`; migrations or a deprecation shim

### Design Decisions

#### Skill Prefix Rides the Existing Resolver
**Decision**: The receiving-harness skill prefix is a `skill_prefix` key on `fab agent -o yaml`, computed by `internal/agent.SkillPrefix`; there is no standalone `fab skill-prompt` CLI.
**Why**: The only consumer that needed a CLI hop is the markdown operator, which already resolves the target repo through `fab agent … -o yaml`; one resolution supplies both the session command and the prefix, so no second resolver or shell round-trip exists to drift.
**Rejected**: Keeping `fab skill-prompt` with fewer flags (still a second entry point and a shell hop for a one-character decision); a `providers.<name>.skill_prefix` config knob (rule is fixed by decision in `260908-tfzn`); a prose-only rule (the Go launchers need the renderer).
*Introduced by*: 260908-gxbf-agent-yaml-skill-prefix

### Deprecated Requirements

#### `fab skill-prompt` read-only renderer CLI
**Reason**: Duplicated default-role provider resolution and added a three-step shell recipe for a binary decision; never released (PR #649 unmerged).
**Migration**: N/A — consumers read `skill_prefix` from `fab agent -o yaml` or identify the receiver from process evidence.

## Tasks

### Phase 1: Core Implementation

- [x] T001 Go: add `SkillPrefix` to `src/go/fab/internal/agent/skill_prompt.go` (used by `SkillPrompt`) with a table test; add `SkillPrefix string \`yaml:"skill_prefix"\`` after `Dispatch` in `internal/agent/resolution.go` and populate it in `cmd/fab/resolution.go`; add a `Lines` guard in `internal/agent/resolution_test.go`; update `cmd/fab/agent_surface_test.go` key counts (11→12, 12→13), golden fixtures (append `skill_prefix`), and add a codex case; delete `cmd/fab/skill_prompt.go`, relocate `TestSkillPromptLaunchers` into a new `cmd/fab/launcher_prompt_test.go`, delete `cmd/fab/skill_prompt_test.go`, remove the `main.go` registration, reword `operator.go` `Long` text; run `cd src/go/fab && go test ./...`. <!-- R1 R2 R3 -->

### Phase 2: Skill prose

- [x] T002 [P] `src/kit/skills/_cli-fab.md`: delete `## fab skill-prompt` + Contents entry; add `skill_prefix` row to § fab agent `-o yaml` key table; reword § fab batch and § fab operator built-in-launcher sentences. Help bundle: edit `docs/site/skill.md` (drop the Skill-prompts bullet) and run `scripts/sync-skill.sh`. <!-- R4 -->
- [x] T003 [P] `src/kit/skills/_cli-agents.md`: replace § Skill Prompts with the short owner section (receiver identity, one-token quoting, message types distinct); `src/kit/skills/fab-operator.md`: reword step 6, step 7 tmux example + parenthetical (`$skill_prompt_quoted`). <!-- R5 -->

### Phase 3: Specs sweep

- [x] T004 [P] Specs: `docs/specs/glossary.md:104` (drop `fab skill-prompt` from the CLI list), `architecture.md:566` (renderer/`skill_prefix` sentence), `skills.md:1095` (prefix from `skill_prefix`/receiver identity), `stage-models.md:572–574` (add `skill_prefix` to the YAML key list). <!-- R6 -->

### Phase 4: Verification

- [x] T005 Sweep: grep `skill-prompt|skill_prompt|fab_skill_token|fab_skill_prompt` outside `.claude/`, `.agents/`, `fab/changes/`, `docs/memory/` and confirm zero hits; `cd src/go/fab && go test ./...`; `go vet ./...`. <!-- R3 R4 R5 R6 -->

## Acceptance

### Functional Completeness

- [x] A-001 R1: `agent.SkillPrefix` exists, returns `$` only for `codex`, and `SkillPrompt` delegates to it; a table test covers `codex`, `claude`, `agy`, `kimi`, `custom`, `""`
- [x] A-002 R2: `fab agent -o yaml` emits `skill_prefix` as the final, always-present key with the resolved provider's prefix; the golden fixtures and key-count tests assert it, including a codex case
- [x] A-003 R3: `fab skill-prompt` is gone (source, tests, `main.go` registration, `operator.go` help text) and `TestSkillPromptLaunchers` survives in a launcher-scoped test file
- [x] A-004 R4: `_cli-fab.md` has no `skill-prompt` section or mentions, documents `skill_prefix`, and the help bundle drift guard passes
- [x] A-005 R5: `_cli-agents.md` § Skill Prompts is the short owner section; `fab-operator.md` steps 6–7 point at it with a one-token quoted prompt variable
- [x] A-006 R6: the four specs no longer name `fab skill-prompt`; `stage-models.md` lists `skill_prefix`

### Behavioral Correctness

- [x] A-007 R2: `Resolution.Lines()` output is unchanged (no `skill_prefix` line) — guarded by a test

### Removal Verification

- [x] A-008 R3: repo-wide grep for `skill-prompt`, `fab_skill_token`, `fab_skill_prompt` (excluding `.claude/`, `.agents/`, `fab/changes/`, `docs/memory/`) returns nothing

### Scenario Coverage

- [x] A-009 R1: launcher regression passes for `codex`, `claude`, `agy`, `kimi`, `custom`, `missing` across `new`/`switch`/`operator`

### Code Quality

- [x] A-010 Pattern consistency: new field/test code follows the surrounding `Resolution`/`agent_surface_test.go` patterns; edits touch canonical `src/kit/skills/` only
- [x] A-011 No unnecessary duplication: the prefix rule has exactly one Go owner and one prose owner; every other file points
- [x] A-012 CLI ⇒ docs + tests: the `fab agent` surface change is reflected in `_cli-fab.md` and covered by tests; `go test ./...` passes

## Notes

- Memory files (`runtime/agent-primitives`, `runtime/operator`, `runtime/providers-and-profiles`) are rewritten at hydrate, not apply.

## Deletion Candidates

None — this change is itself a deletion: it removed the planned targets (`cmd/fab/skill_prompt.go`, `cmd/fab/skill_prompt_test.go` minus the relocated launcher regression, the `main.go` registration, the `_cli-fab.md` section, and the help-bundle bullet) and orphaned nothing (`loadRepoConfig`, `agent.ResolveRole`, and `shellquote.Single` all retain call sites). The remaining `skill-prompt` mentions in `docs/memory/runtime/` are planned hydrate-stage rewrites, not discovered redundancy.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `TestSkillPromptLaunchers` moves to a new `cmd/fab/launcher_prompt_test.go` rather than into `batch_test.go` or `operator_test.go` | It spans all three launchers; a dedicated file avoids implying ownership by one of them | S:70 R:95 A:90 D:80 |
| 2 | Certain | Key-count assertions bump from 11/12 to 12/13 rather than being dropped | They are the struct-parity guard; the new key is intentional | S:85 R:95 A:95 D:90 |
| 3 | Certain | Memory edits are left to hydrate; apply sweeps skills, specs and help bundle only | Pipeline contract — hydrate owns `docs/memory/` | S:90 R:95 A:95 D:95 |

3 assumptions (3 certain, 0 confident, 0 tentative).
