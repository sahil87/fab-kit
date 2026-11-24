---
type: memory
description: "CLI mechanics and role policy — session/workers depth knobs, environment and launch overrides, four pane-capable built-ins with independent interactive/native/headless grammar and per-role fills, six roles, precedence, stage mapping, `fab agent` YAML resolution, agent presets, the reserved `tool-default` model value, `fab agent models` discovery, deprecated `fab resolve-agent` compatibility, the deprecated pre-2.31 `provider` spellings, dispatch-mode descent, and consumers."
---
# CLIs & Agent Profiles

**Domain**: runtime

## Overview

Agent config splits **CLI mechanics** (how to invoke an agent) from **role policy** (which model + effort a given job runs at). CLIs live in a top-level `clis:` table carrying command grammar plus a per-role fill map; the six **roles** are fab-owned slot names, and a **profile** is the concrete `{cli, model, effort}` a role resolves to. The advertised user surface is two knobs — `agent.session` and `agent.workers` — which pick a CLI by agent **depth**.

**Vocabulary** (used consistently across the config, the Go packages, and the docs):

| Term | Meaning |
|------|---------|
| **role** | one of six fixed slot names — `default`, `operator`, `doing`, `review`, `hydrate`, `fast` |
| **profile** | a concrete `{cli, model, effort}` value a role resolves to |
| **CLI** | independent pane (`interactive_command`), native (`native`), and headless (`headless_command`) capabilities plus per-role fills |
| **Tier 1 / Tier 2** | agent **depth** — the agents a user talks to vs. the agents pipeline stages dispatch to |

Tier 1/Tier 2 name depth and nothing else. A pane worker is still Tier 2: the defining property is "owes a result artifact and owns no transitions" ([dispatch.md](/runtime/dispatch.md)) — ship/review-pr workers are the single carve-out, self-managing their own stage's transitions — not "never spoken to".

This file is the model — the depth knobs, the four built-in CLIs, the six roles and the fixed partition, the fill precedence, the stage→role mapping, the structured `fab agent -o yaml` surface, the `fab agent models` discovery verb, the deprecated `fab resolve-agent` compatibility projection, and who consumes the resolution. The **config-schema authority** is [_shared/configuration.md](/_shared/configuration.md) § `clis` and § `agent`; the **dispatch-seam wiring** is [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution; the pre-implementation design intent is `docs/specs/stage-models.md` (drift-guarded against the Go maps).

## Requirements

### Requirement: Two depth knobs — `agent.session` and `agent.workers`

`fab/project/config.yaml` SHALL support two scalar keys naming a CLI, and they SHALL be the whole advertised agent surface:

```yaml
agent:
  session: claude    # Tier 1 — what you talk to: fab agent, fab batch, rk operator
  workers: agy       # Tier 2 — what pipeline stages dispatch to
```

Both default to `claude`, so a fresh install writes nothing and resolves fab-kit's shipped profiles. Both are `scope: both`, so the choice is settable once machine-wide in `~/.fab-kit/config.yaml`.

The **role→depth partition is fixed and fab-owned** (`roleDepth` in `internal/agent`, read externally through the exported `agent.IsSessionRole`):

| Depth | Roles | CLI from | Applies at |
|-------|-------|---------------|------------|
| **Tier 1** (session) | `default`, `operator` | `agent.session` | **launch** time — fab cannot switch a running session's CLI |
| **Tier 2** (workers) | `doing`, `review`, `hydrate`, `fast` | `agent.workers` | **every stage dispatch** (`fab agent <stage> -o yaml`) |

The split is mechanically real rather than cosmetic, which is what makes depth the right knob axis. `intake` rides `default` and is therefore a session role for exactly that reason — it runs foreground in the user's own session.

A knob supplies only the **CLI** rung. Model and effort always come from the resolved CLI's own per-role fills, so naming a CLI is a complete configuration.

#### Scenario: one knob re-points every dispatched stage

- **GIVEN** `agent: { workers: agy }` and no other agent keys
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `cli: agy` (apply → `doing`, a workers role)
- **AND** `fab agent operator -o yaml` still emits `cli: claude` (a session role)

#### Scenario: no `agent:` block at all

- **GIVEN** a config with no `agent:` block
- **WHEN** any role or stage is resolved
- **THEN** the CLI is `claude` for every role, on fab-kit's shipped per-role models

### Requirement: Agent presets — a named role assignment chosen by `agent.preset`

A **preset** is a NAME for a **sparse** bundle of fields fab already has — `session`, `workers`, `profiles.<role>.{cli,model,effort}`, plus a display-only `description` — defined in the top-level `agent_presets` table and chosen by the scalar `agent.preset` (scope `both`; env `FAB_AGENT_PRESET` via the registry-driven env layer, no special-case env code). A preset adds fields to the `agent.*` tier and nothing else: no new role vocabulary, no change to the stage→role map, and the resolver and every consumer keep their existing semantics. Because selection is config (the env layer included), it reaches `fab dispatch start` in all three dispatch modes, which an invocation flag cannot.

```yaml
agent:
  preset: codex-workers      # one name instead of two knobs plus per-role overrides
agent_presets:
  mine:
    description: Codex workers, claude critic
    workers: codex
    profiles:
      review: { cli: claude }
```

- **Built-ins.** The embedded `defaults.yaml` ships exactly `claude` (`session: claude`, `workers: claude`), `codex-workers` (`workers: codex`) and `codex-review` (`profiles.review.cli: codex`), each with a `description`. They are inert until selected, so with nothing set resolution is identical to the knobs-only result. `internal/agent`'s `init()` pushes the parsed table into `config.BuiltinAgentPresets`; the definitions merge per key, built-in < project < system < env. `codex-review` sets only the review CLI, so its model/effort re-derive from codex's own fills like any `agent.profiles.<role>.cli` override.
- **One active preset, spliced at the choosing layer.** The active preset is the `agent.preset` value at the highest layer (env > system > project) with a non-empty value — exactly one, never merged with a lower layer's choice. Its fields enter **at that layer, directly below that layer's own explicit `agent.*` fields**: they beat every lower layer's explicit fields, lose to explicit fields at the same or a higher layer, and invocation flags stay on top. The splice is `applyPreset` in `internal/config`, called by both the typed loader and `layersFrom`, so `fab config show` prints the tree consumers resolve from.
- **Reserved `none`.** `none` wins the selection like any name but activates nothing ("no preset here, even if a lower layer chose one"); it is never a definition (`config set` rejects `agent_presets.none[.…]`; a hand-written one is warned about and ignored). An empty value is unset.
- **Unknown name.** At resolve time an unknown name emits `fab: warning: unknown agent preset "<name>" (agent.preset from <layer>); ignoring it` and resolves as if no preset were chosen (no fall-through to a lower layer's preset); a spawn never fails for a stale name. `fab config set [--system] agent.preset <name>` is the one place it is an error: a name that is neither defined (built-in ∪ system ∪ project ∪ env) nor `none` is a usage error naming the sorted defined set.
- **Provenance.** `Config` carries the active preset's name, choosing layer and the set of `agent.*` leaves it actually supplied (a leaf is supplied only when no layer at or above the choosing layer sets it). `ResolveRoleWithSource` labels those fields `agent_presets.<name>` in `source.*`; fields re-derived from a preset-chosen CLI keep their `clis.<p>.profiles.<role>` label. `fab agent -o yaml` emits a top-level `preset: <name>` key only when a preset is active. `fab config show --origin` attributes each preset-supplied leaf to the choosing layer with detail `preset <name>`.
- **`fab agent --preset <name>`** is equivalent to choosing the preset at the env layer for one resolution: it loses to explicit env `FAB_AGENT_*` fields, flags stay on top, `none` disables a configured preset, an unknown name warns. It is resolution-only — it does not export `FAB_AGENT_PRESET` into an exec'd session and does not reach `fab dispatch start` (set `agent.preset` or export the variable for that). Combined with a bare `--cli` (no selector) it is a usage error, since that form bypasses role resolution. It is not on `fab resolve-agent`.
- **`fab agent presets [--repo <path>] [--json]`** lists every defined preset sorted by name (never `none`). Human output is a table of name, source, description and set fields. `--json` is an array of `{name, description, source, [session], [workers], [profiles]}` — sparse (unset fields omitted), with **no `active` flag**; `source` ∈ `built-in`|`system`|`project`|`env` is the highest layer contributing to the definition. This array is a cross-repo contract consumed by HexoKit.

#### Scenario: an explicit field at the choosing layer beats the preset

- **GIVEN** project `agent.preset: codex-workers` and project `agent.workers: agy`
- **WHEN** a workers role resolves
- **THEN** the CLI is `agy` — the explicit field at the choosing layer wins over the spliced preset

#### Scenario: a preset beats lower layers' explicit fields

- **GIVEN** env `FAB_AGENT_PRESET=codex-workers` and system `agent.workers: claude`
- **WHEN** a workers role resolves
- **THEN** the CLI is `codex` (`source.cli: agent_presets.codex-workers`), with model/effort from codex's own fills

#### Scenario: the highest layer's choice shadows the rest

- **GIVEN** project `agent.preset: codex-review` and system `agent.preset: codex-workers`
- **WHEN** roles resolve
- **THEN** only `codex-workers` applies; nothing from `codex-review` is merged
- **AND** `FAB_AGENT_PRESET=none` in the environment cancels both

#### Scenario: a stale name never fails a spawn

- **GIVEN** `agent.preset: gone` in a config file and no definition named `gone`
- **WHEN** `fab agent apply -o yaml` or `fab dispatch start` resolves
- **THEN** a `fab: warning: unknown agent preset "gone"` line is written to stderr and resolution equals the no-preset result

### Requirement: Per-session CLI selection — environment and launch flags

`FAB_AGENT_SESSION` and `FAB_AGENT_WORKERS` SHALL provide process-tree-local overrides for the two depth knobs. The current process and its descendants resolve these variables above project and system config, so two shell sessions in separate worktrees can select different worker CLIs without changing committed configuration. The variables are instances of the generic registry-derived environment mechanism documented in [_shared/configuration.md](/_shared/configuration.md) § Override Cascade & Scope Enforcement; both are honored because their registry rows have `scope: both`.

- **`FAB_AGENT_SESSION=<cli>`** selects the Tier-1 CLI for commands launched by that process tree.
- **`FAB_AGENT_WORKERS=<cli>`** selects the Tier-2 CLI projected by `fab agent <stage> -o yaml` and used by `fab dispatch start` when it re-resolves a stage.
- The values are YAML-parsed config overrides, not CLI-validation surfaces; CLI names remain opaque and an unknown name fails at the existing resolution lookup.
- An empty variable behaves as unset — whether blank or spelled as an empty YAML value (`null`, `""`), per the cascade's one emptiness rule. The override is never persisted in `config.yaml`, a change artifact, or runtime state.

The session-spawning commands SHALL expose `--workers <cli>` as sugar for exporting `FAB_AGENT_WORKERS` into the spawned process tree:

- `fab agent --workers <cli>` appends the exact assignment to the exec environment in both role-addressed and CLI-addressed modes. It does not alter Tier-1 session-command resolution, and `--print` continues to print only that resolved command.
- `fab batch new --workers <cli>` and `fab batch switch --workers <cli>` prefix a shell-quoted assignment (`FAB_AGENT_WORKERS='<cli>'`) to the agent command inside the interactive-shell wrap of a newly launched tmux window. Embedded quotes and shell metacharacters remain data. `rk operator --workers <v>` carries the value on argv and rk validates it (letters/digits/`_`/`-`).
- The flag value passes through without validation. There is no `--session` flag; callers use `FAB_AGENT_SESSION` directly.

These config variables are unrelated to the fab-kit binary's `FAB_AGENTS` skills-list override. The user-facing CLI-selection variables are `FAB_AGENT_SESSION` and `FAB_AGENT_WORKERS`; the environment mechanism itself applies generically to every `both`/`system` registry row. (2d1w)

#### Scenario: two sessions select different worker CLIs

- **GIVEN** two process trees rooted in separate worktree shells
- **WHEN** one launches with `FAB_AGENT_WORKERS=kimi3` and the other with `FAB_AGENT_WORKERS=codex`
- **THEN** every Tier-2 stage in each tree resolves its own selected CLI
- **AND** neither worktree's `fab/project/config.yaml` changes

#### Scenario: `--workers` is launch-only sugar

- **GIVEN** `fab agent --workers codex` or a tmux launcher with the same flag
- **WHEN** the new session starts
- **THEN** its descendants inherit `FAB_AGENT_WORKERS=codex`
- **AND** the launching command's Tier-1 CLI/profile resolution and persisted state are unchanged

### Requirement: CLIs table — pure capabilities plus per-role fills

`fab/project/config.yaml` SHALL support a top-level `clis:` map keyed by **opaque, user-chosen CLI names**. Each CLI MAY independently carry:

- **`interactive_command`** — **pure launch grammar**: how to open an interactive agent session, and nothing more. It makes the CLI pane-capable when tmux is available. Prompt delivery is fab's, not the command's — no code path appends a prompt argument to it ([dispatch.md](/runtime/dispatch.md) § Prompt delivery), so pane capability does not depend on whether the CLI accepts a positional initial prompt. **Exec contract**: the command MUST exec its binary, so the binary — not a wrapper shell — owns the pane's foreground. The readiness gate's takeover precondition requires exactly that (a shell foreground reports `booting` and is never typed into — [dispatch.md](/runtime/dispatch.md) § `fab dispatch ready`), so a wrapper that keeps a shell in the foreground is unsupported by design and fails **observably**: every probe reports `booting` with the shell prompt in the snippet until the wiring's consecutive-booting allowance is spent and the run escalates — never silently.
- **`native`** — declares native Agent-tool capability. CLI names are opaque, so fab never infers it.
- **`headless_command`** — runs one headless stage task via `fab dispatch`; the prompt is supplied on stdin.
- **`models_command`** — optional shell command that prints the models the tool offers as a JSON array of rows; it replaces the CLI's built-in discovery adapter for `fab agent models` and plays no part in resolution or dispatch (§ `fab agent models`). No built-in ships one.

**Launch environment.** fab runs both commands under the user's interactive shell — `${SHELL:-/bin/sh} -i -c '<command>'` — at its dispatch, `fab pane open`, and batch launch seams (`fab agent` execs in the caller's own shell and is not wrapped), so env exported only by interactive-shell startup (CLI auth tokens, PATH additions in `~/.zshrc`) is present for the CLI. The wrap is a launch mechanism, not CLI grammar: it is not part of either command string, and the dispatch record's `spawn_cmd` and `fab agent -o yaml`'s `command` stay unwrapped. Hand-wrapping a command in `$SHELL -i -c` in a repo config is unnecessary (a pre-wrapped command still works — it nests, sourcing the rc twice). The launch assumes POSIX shells; a non-POSIX tmux default-shell (fish, nushell) fails the launch. Full contract: [agent-primitives.md](/runtime/agent-primitives.md) § Fab launches CLI commands under the user's interactive shell.

Each CLI MAY additionally carry **`profiles`** — a map keyed by role name, each value `{model, effort}`: *"when this CLI plays this role, use this model/effort."* It supplies the command's `{model}`/`{effort}` placeholders (§ Fill precedence). A CLI's **`profiles.default` doubles as its cross-role fallback** — a role absent from the map resolves the `default` entry, then empty. The map merges per **role** and then per **field** over the built-in table, exactly as the command fields do, and `clis` is `scope: both`, so a fill is settable once per machine.

The capability fields (`interactive_command`, `native`, `headless_command`) are never merged or substituted for one another. Their presence says **how** a CLI can run, never **which** mode is selected. `dispatch.mode` supplies that preference and the shared selector descends pane → native → headless, starting at the configured rung and never ascending.

Every supported agent CLI has an interactive mode, so built-in CLIs SHALL ship `interactive_command` launch grammar and rely on the generic readiness gate for boot and first-run walls. Headless operation varies by CLI and its `headless_command` grammar MUST be probed independently. A user-defined CLI MAY omit `interactive_command`; automatic selection skips the pane rung, and an explicit pane launch reports the existing configuration-key hint.

**fab-kit ships FOUR built-in CLIs — `claude`, `codex`, `agy`, and `kimi`** (`defaultCLIs` in `internal/agent`, parsed from the embedded `defaults.yaml` — § The built-in defaults are an embedded `defaults.yaml`; each command string is additionally exposed as a package var, `DefaultInteractiveCommand` / `DefaultCodex*` / `DefaultAgy*` / `DefaultKimi*`, that `internal/configref` interpolates, so no literal is duplicated):

| Built-in | `interactive_command` | `native` | `headless_command` | `profiles` |
|----------|-------------------|----------|--------------------|------------|
| `claude` | templated default | `true` | `claude -p --permission-mode bypassPermissions --model {model} --effort {effort}` | all six roles |
| `codex` | `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` | absent | `codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` | all six roles — dense; `doing` and `review` on different models |
| `agy` | `agy --dangerously-skip-permissions --model {model}` | absent | `sh -c 'agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'` | sparse — `default`, `fast` (model only) |
| `kimi` | `kimi --auto -m {model}` | absent | `sh -c 'kimi -m {model} -p "$(cat)"'` | **none** — deliberate |

**claude, codex and agy ship per-role fills**, so naming one of them on a depth knob is a complete configuration: `agent.workers: codex` resolves a model suited to each role, and **role differentiation survives the CLI swap** (apply runs on codex's top catalog model, review on a different model at `xhigh`, ship on the cheaper `fast` model at `low`). claude's and codex's maps are **dense** — every role pins its own model and effort, so no shipped row leans on the cross-role fallback. agy's map is **sparse** — a role absent from it resolves agy's `default` entry. The merge is per **field**, so a user override carrying `effort` only still takes its model from the CLI's `default`; no shipped codex row relies on that.

Two per-CLI fill shapes are load-bearing:

- **agy's fills are model-only** — that CLI's model IDs *embed* the reasoning level as a suffix (`gemini-3.1-pro-high`), so the effort rides the ID and the grammar carries no `{effort}` placeholder for one to land in. The IDs are concrete catalog entries from `agy models`, so they are rows to bump when that catalog moves.
- **kimi ships NO fills at all**, and empty is the correct built-in there rather than a gap: its `-m` takes a **user-config model alias** (managed installs expose `kimi-code`/`k3`; custom CLIs differ), so any pinned value would break non-managed installs. The empty `{model}` drops the `-m` pair through `spawn.WithProfile`'s token-drop and kimi falls back to the user's own `default_model`. A user wanting role differentiation pins `clis.kimi.profiles.<role>.model` themselves.

claude's fills are the family **aliases** `opus` (`default`, `review`) and `sonnet` (`operator`, `doing`, `hydrate`, `fast`): `claude --model` and the Agent tool's `model` parameter both take them and resolve the newest model of the family, so they track the installed Claude Code and need no refresh. Non-claude fills (codex, agy) are **concrete IDs** — their model flags take a catalog slug and have no "latest" alias — **refreshed at kit-release cadence** by editing `defaults.yaml`. Every fill is **unvalidated pass-through** values like every other resolved string, and a stale one fails loudly at the CLI CLI and is corrected by **one config line** — `clis.<name>.profiles.<role>.model`, settable once per machine because `clis` is `scope: both`. There is no staleness automation: no CI check against provider APIs, and resolution never fetches a model catalog — `fab agent models` lists what a tool offers on demand but feeds nothing back into the fills (§ Design Decisions → "Built-in CLIs Ship Grammar AND Per-Role Fills"; `docs/specs/stage-models.md` § Refreshing the non-claude fills carries the policy and its decision lineage). **Current values are not restated here** — run `fab config explain` (renders live from the embedded defaults) or read the drift-guarded sample in [stage-models.md](../../specs/stage-models.md) § Built-in clis.

A project's `clis:` block per-field-merges over the built-ins via `agent.ResolveCLI(name)`, including the profiles map.

**Naming a built-in is sufficient — no `clis:` block required.** A depth knob, an `agent.profiles` entry, or an invocation flag that names `codex`/`agy`/`kimi` resolves with zero config, and the configured dispatch preference then applies to that CLI's capabilities — none of the three declares `native`, so their stages descend to headless. Adding or overriding a command never changes mode policy by itself. A built-in CLI is otherwise **inert**: adding a row changes no default behavior, which is why these rows live in Go while behavior-changing config still ships commented (see § Design Decisions → "Built-in CLI Grammar in Go, Fill Values in Config").

```yaml
clis:
  codex:
    profiles:                 # override one shipped fill; the rest stay built-in
      review: { model: <codex-model-id>, effort: xhigh }
    # commands inherit the built-in grammar; override either to change it
```

The two command fields are deliberately unmerged because session and dispatch are **different invocations of the same binary** (claude interactive `-n <name>` vs headless `-p`; codex TUI vs `codex exec`) — no single template expresses both. `{model}`/`{effort}` placeholders in either command are substituted at resolve time via `spawn.WithProfile` (reused, not reimplemented — see [configuration.md](/_shared/configuration.md) § `clis` for the template/append modes and the empty-value token-drop rule).

The non-claude built-ins run **full-auto**, because a stage worker is unattended and has no approval-answering channel: codex carries `--dangerously-bypass-approvals-and-sandbox` on both forms, agy carries `--dangerously-skip-permissions` on both forms, and kimi carries `--auto` on its **interactive** command alone — its dispatch form takes **no approval flag at all**, because `kimi -p` is already non-interactive, auto-approves tool calls, and *rejects* `--yolo`/`--auto`. kimi is the one built-in whose full-auto posture is asymmetric across its two commands, and the asymmetry is the CLI's, not a choice.

**Prompt delivery differs per CLI, and the grammar absorbs the difference.** `fab dispatch` always pipes the stage prompt to the command's **stdin**. codex reads stdin directly via its `exec` subcommand, as does claude's headless command through `-p`. agy and kimi do not — their `-p` takes the prompt as an **argument** and ignores stdin — so both dispatch commands nest a shell: `sh -c '<cli> … -p "$(cat)"'`. The nesting is load-bearing, not stylistic (§ Design Decisions → "Nested-Shell `$(cat)` Idiom"). agy's command additionally raises `--print-timeout` to `120m`, since that CLI's 5-minute default would kill a long stage worker.

**agy is pane-capable.** Its built-in `interactive_command` is `agy --dangerously-skip-permissions --model {model}`, so sessions and pane dispatch work without a CLI override. A fresh workspace can park at agy's interactive trust prompt even under the bypass flag; the readiness gate classifies that as an ordinary judgment round, after which delivery proceeds. The answer is remembered for the exact workspace path. An operator may pre-seed that exact path under `trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json`; fab neither guesses paths nor writes the CLI's trust store.

**kimi is pane-capable**, shipping `kimi --auto -m {model}` alongside its headless command, because both halves of that same question were answered against kimi 0.34.0 (2026-08-10):

- **First run**: kimi gates a fresh folder behind a `Trust this folder?` wall, which is an ordinary readiness-gate **judgment round** — the pane reads `parked`, one Enter clears it, and the answer is remembered per folder, so it amortizes across every later pane worker in that checkout ([dispatch.md](/runtime/dispatch.md) § `fab dispatch ready`). It needs no code.
- **Input echo**: kimi draws vertical rules down both sides of its input box, so a hard-wrapped pointer arrives with `││` interleaved between the halves. `deliver`'s echo verification counts occurrences with box-drawing runes dropped alongside whitespace, so the pointer verifies ([dispatch.md](/runtime/dispatch.md) § `fab dispatch deliver`).

kimi has no interactive-initial-prompt flag — its `-p` is the non-interactive form, and a bare positional parses as a subcommand — which costs it nothing: delivery is post-open in every case, so pane capability never depended on that flag.

`fab config explain` presents all four as **live, uniformly-rendered built-in defaults** — every CLI line carries exactly one leading `#` prefix in a fence or `init --system` scaffold (no doubled `# #` marker; an inline `# ...` note on a command line stays content), and the segment prose warns that hoisting a whole block PINS its fills against kit-release refreshes; see [configuration.md](/_shared/configuration.md) § `clis` for the rendered presentation and its parse-side guarantees, and § The machinery is documented, not scaffolded for why the managed fence omits the block.

**CLI names are opaque — fab NEVER infers a CLI from a model string.** Two entries fronting the same vendor under different names are different CLIs, each with its own fills.

#### Scenario: the preference selects among independent capabilities

- **GIVEN** built-in claude and `dispatch.mode: pane`
- **WHEN** tmux and its `interactive_command` are available
- **THEN** `fab agent <stage> -o yaml` includes `dispatch: { rung: pane, command: … }`
- **WHEN** tmux is unavailable
- **THEN** the selector descends to claude's `native: true` and omits the `dispatch:` key
- **AND** `dispatch.mode: headless` starts at headless and emits claude's substituted `headless_command`, even though native capability also exists

#### Scenario: `kimi` resolves an empty model and drops the flag pair

- **GIVEN** `agent.workers: kimi` and no `clis.kimi` block
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `cli: kimi`, `model: ""`, and `dispatch.command: sh -c 'kimi -p "$(cat)"'` — the `-m {model}` pair dropped as a unit, the quoted `"$(cat)"` segment left syntactically intact

#### Scenario: `kimi` is eligible for a pane worker and a session

- **GIVEN** `agent.workers: kimi`, `dispatch.mode: pane`, a reachable tmux server, and no `clis.kimi` block
- **WHEN** `fab dispatch open` runs for a stage
- **THEN** the pane opens on `kimi --auto` — the empty `{model}` dropping the `-m` pair — and the readiness gate answers the first-run trust wall before `deliver` types the pointer
- **AND** `fab agent --cli kimi --print` prints that same composed command instead of erroring on a missing `interactive_command`

#### Scenario: the agy built-in is eligible for a pane worker and a session

- **GIVEN** `agent.workers: agy` and a reachable tmux server
- **WHEN** `fab dispatch open` runs for a stage under pane preference
- **THEN** it opens the exact composed `agy --dangerously-skip-permissions --model <resolved-model>` command
- **AND** the readiness gate reports any first-run trust wall for an ordinary judgment round before `deliver` types the pointer
- **AND** `fab agent --cli agy --print` composes the model-free `agy --dangerously-skip-permissions` command instead of reporting a missing capability

#### Scenario: naming a built-in CLI with no `clis:` block

- **GIVEN** a config with `agent: { profiles: { review: { cli: codex, model: <codex-model-id>, effort: high } } }` and no `clis:` block at all
- **WHEN** `fab agent review -o yaml` runs
- **THEN** it emits `cli: codex` plus the `dispatch:` mapping selected by the current `dispatch.mode`; with the default `native` preference, codex descends to its headless capability
- **AND** the same default preference resolves built-in claude natively because claude declares `native: true`

### Requirement: Six roles with fixed referents

The role names are the six fixed slots — `default`, `operator`, `doing`, `review`, `hydrate`, `fast`. A role is **stage-named only where it maps 1:1 to a single referent** (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is **multi-referent** — `fast` governs the ship stage AND the `/fab-proceed` prefix-step dispatches (`/fab-switch`, `/git-branch`), and `default` governs intake, `fab batch`/`fab agent`, and the `/fab-proceed` create-intake dispatch. There is no `thinking` role: with `review` its own role, `thinking`'s only remaining stage would be intake, which never dispatches.

| Role | Referents | Depth |
|------|-----------|-------|
| `default` | intake (advisory, foreground); `fab batch` worker sessions; `fab agent` with no role; the `/fab-proceed` create-intake dispatch | Tier 1 |
| `operator` | the operator coordinator session (launched by `rk operator`, which resolves the role via `fab agent operator -o yaml`) | Tier 1 |
| `doing` | `apply`, `review-pr` — execution that must not err | Tier 2 |
| `review` | `review` — the critic (its own role so its model/effort dial independently of the author's) | Tier 2 |
| `hydrate` | `hydrate` — memory writing (its own role so it runs on a different model/effort than apply) | Tier 2 |
| `fast` | `ship` — near-mechanical work — plus the `/fab-proceed` prefix steps | Tier 2 |

**Current profiles**: run `fab config explain` (renders live from the embedded defaults) or see the drift-guarded table in [stage-models.md](../../specs/stage-models.md) § Default role profiles. Not restated here — this table owns the *roles*, which change far less often than the models.

`agent.profiles` is the **sparse per-role escape hatch** beneath the knobs: `agent.profiles.<role>.{cli, model, effort}`, every field optional, each set field beating the knob (CLI) or the CLI's fill (model/effort). It carries **no cross-role inheritance** — `agent.profiles.default` is the `default` role's own override, not a fallback source for the other five. A Claude model takes one of three forms: a family **alias** (`opus`/`sonnet`/`haiku`/`fable` — valid on both dispatch seams and what the built-in fills use, tracking the newest model of the family), a **versioned ID** (`claude-sonnet-5-5` — pins a specific model; `model_alias` maps it to its family), or a bare family ID (`claude-sonnet`), which is invalid at the API and fails both seams.

#### Scenario: a per-role override dials one role only

- **GIVEN** `agent.profiles: { default: { model: X }, doing: { effort: high } }`
- **WHEN** `fab agent apply -o yaml` (role `doing`) runs
- **THEN** `effort: high`, and the model comes from the resolved CLI's `doing` fill — **not** from `agent.profiles.default.model`

### Requirement: The built-in defaults are an embedded `defaults.yaml`

The built-in tables — the depth knobs, the four-CLI table with every CLI's per-role fills, the `dispatch:` block carrying the five built-in dispatch defaults (`mode`/`column_width`/`min_cols`/`min_rows`/`reap_done`) — are **data**, and live in `src/go/fab/defaults.yaml`, shaped exactly as a **user config-file fragment**: the same `agent:` / `clis:` / `dispatch:` keys a project writes to override them. The file is the **single value source for every built-in default** — no Go source carries an independent copy; the `config.DefaultDispatch*` symbols are package-level vars the package's own `init()` assigns from the parsed `dispatch:` block (the import cycle forbids `internal/config` reading them back — see [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Dispatch Defaults Are Init-Injected from `defaults.yaml`"). The file is compiled into the binary via `//go:embed` and unmarshalled **once at package initialization** into `config.Config` — the same struct `config.LoadPath` fills from a user's `config.yaml` — so the built-in shape and the config schema cannot diverge. A malformed file **panics at init** (compiled-in bytes make a parse failure a defective build artifact, not a runtime condition), and nothing is read from the kit cache at runtime, so resolution cannot break on a missing or corrupt cache. `defaults_test.go` is the file's safety net against a YAML typo: it asserts the parse, exhaustive role/CLI coverage with non-empty fields, and both command fields for every built-in, and that codex's fill map is dense (every role names a model and effort, none `ultra`). The agy and kimi interactive values are pinned **by value** — `agy --dangerously-skip-permissions --model {model}` and `kimi --auto -m {model}` — so a plausible edit cannot reach pane workers unnoticed. It also asserts that every **fill-carrying** CLI has a non-empty `profiles.default.model` while kimi's empty fill map is deliberate rather than skipped, that agy's fills set no `effort`, that no built-in uses the deprecated flat fill spelling, that the `dispatch:` block pins the five dispatch defaults and their injection wiring, and that the file defines no keys outside the `agent:`/`clis:`/`dispatch:` surface.

**The YAML/Go split is the overridable/fixed boundary.** What lives in `defaults.yaml` is user-overridable by writing the same key in `fab/project/config.yaml` (or `~/.fab-kit/config.yaml`); what stays in Go is fab-owned policy — `stageRoles` (the fixed stage→role mapping) and `roleDepth` (the role→depth partition). The config-fragment shape is also what makes `defaults.yaml` the **physical source of the cascade's defaults tier** (see [_shared/configuration.md](/_shared/configuration.md) § Override Cascade): consumers still reach those defaults at the existing point-of-use seams rather than through `LoadPath`'s merge, while the read-model surfaces merge them as a real tier 0 through the registry projection `configref.DefaultsMap` — so folding the file into the loader's own merge would be a merge-order change, not a parser change.

**A model bump edits `defaults.yaml`** and runs the tests. The file carries the bump procedure inline and names the drift guards that will name *themselves* if a mirror falls behind — `TestDefaultsFileIsWellFormed` (the file itself), `TestDefaultRoleProfilesArePinned` (the deliberate-change pin for claude) and `TestNonClaudeCLIFillsArePinned` (the same pin for codex/agy, plus kimi's deliberate empty map), `TestDefaultsFileCLIs` (codex's map is dense — every role names a model and no row ships `ultra`; agy's stays model-only; kimi's stays empty), `TestDocTablesMatchAgentMaps` (`docs/specs/stage-models.md`'s four-column claude table — the one doc mirror that spells a shipped ID), and `TestConfigReferenceDocumentsCLIFill` (the fill lines in the rendered `fab config explain`, derived from `ResolveCLI`). The spec's inline `clis:` sample is shape-only (IDs elided) and deliberately unguarded. Every other consumer derives from `DefaultProfile()` or `ResolveCLI`, so a bump touches nothing else.

#### Scenario: the defaults travel inside the binary

- **GIVEN** a `fab` binary run from a directory with no fab-kit cache present
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** it resolves the built-in profile from the embedded bytes — no filesystem read of the kit cache occurs, and no on-disk state can make resolution fail

### Requirement: Fill precedence — one chain, CLI-anchored

Resolution SHALL follow one precedence chain, implemented **once** in `internal/agent` (`ResolveRoleWith`, which `cmd/fab` never re-implements):

**CLI**: invocation `--cli` → `agent.profiles.<role>.cli` → the role's depth knob (`agent.session` or `agent.workers`) → built-in `claude`.

**Model / effort, per field**: invocation flag → `agent.profiles.<role>.<field>` → `clis.<p>.profiles.<role>.<field>` → `clis.<p>.profiles.default.<field>` → empty.

`<p>` is the **resolved** CLI (post-override), which is what makes a CLI swap safe: model and effort are re-derived from the new CLI's own fills rather than carried from the old one, so **nothing is ever inherited across CLIs in the first place**. An explicit `agent.profiles.<role>.model` still survives a swap — a pin the user wrote is the user's own escape hatch, not inheritance.

There is exactly **one** cross-role fallback and it lives on the **CLI** side (`clis.<p>.profiles.default`). The agent side has none.

An empty value keeps its established meaning: `spawn.WithProfile`'s token-drop (so the CLI's own default applies) and, in the YAML `model` key, "inherit the session/orchestrator model".

**Reserved model value `tool-default`.** The exact, case-sensitive string `tool-default` (`agent.ToolDefaultModel`) requests the tool's own default model — the way to follow a tool whose fills are concrete slugs (codex, agy) without pinning an ID that goes stale. Through the chain above it is an ordinary non-empty value, so it **wins over lower tiers** at whichever tier holds it (an empty leaf, by contrast, falls through). After the full model precedence has run — the `--model` invocation override included — `agent.EffectiveModel` collapses it to an empty model, once, in `ResolveRoleWithSource`; the CLI-only `--cli X --model tool-default` form applies the same helper, so both addressing forms behave identically. Everything downstream sees an ordinary empty model: YAML `model` and `model_alias` are `""`, `{model}` and its flag pair drop from CLI commands, and native inherits the session model. `source.model` is **not** cleared — it keeps naming the rung that held the token (`agent.profiles.<role>`, `clis.<p>.profiles.<role>`, `clis.<p>.profiles.default`, or `flag`). The token is model-only: an `effort` of `tool-default` passes through verbatim like any other effort string. It is a documented value, not a validated one — resolution stays CLI-neutral. The depth knob is applied only when `--cli` was **not** supplied, so an explicitly-empty `--cli=` resolves an empty CLI — a lookup failure for the caller to report — rather than falling through to the knob.

#### Scenario: `tool-default` beats the CLI fill and collapses to an empty model

- **GIVEN** `agent.profiles.review: { cli: codex, model: tool-default, effort: high }`
- **WHEN** `fab agent review -o yaml` runs
- **THEN** `model` and `model_alias` are `""`, `source.model` is `agent.profiles.review`, and the composed codex command carries no model flag pair
- **AND** without the `model` line the same config resolves codex's `clis.codex.profiles.review` fill

#### Scenario: a CLI swap refills from that CLI

- **GIVEN** `agent.workers: codex` and `clis.codex.profiles: { review: { model: <codex-model-id>, effort: high } }`
- **WHEN** `fab agent review -o yaml` runs
- **THEN** `cli: codex`, `model: <codex-model-id>`, `effort: high`
- **AND** the other workers roles keep codex's **shipped** fills (the override merges per role, then per field)
- **AND** with no `clis:` block at all, every workers role still resolves codex's own shipped fill — never claude's values

#### Scenario: a CLI carrying no fills resolves empty

- **GIVEN** `agent.workers: mine` and a project-defined `clis.mine` carrying commands but no `profiles` — the same shape the built-in `kimi` ships deliberately
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `model` and `effort` are empty, with both placeholder tokens dropped from the composed command so the CLI's own default applies
- **AND** an agy-resolved role also has an empty `effort` value — its fills are model-only, and the grammar carries no `{effort}` for one to land in

### Requirement: Fixed, non-overridable stage → role mapping

The stage→role mapping is **fab-owned and NOT user-overridable** (`stageRoles` in `internal/agent`; no `stage_roles` config, no per-stage escape hatch):

| Stage | Role |
|-------|------|
| `intake` | `default` (advisory only — foreground) |
| `apply` | `doing` |
| `review` | `review` |
| `hydrate` | `hydrate` |
| `ship` | `fast` |
| `review-pr` | `doing` |

`review` and `review-pr` are deliberately in **different** roles despite the shared word: `review` is the critic (discovers what's wrong from a diff); `review-pr` is responsive (fixes already-articulated feedback). `hydrate` is its own role — memory writing runs on a different model/effort than apply's diff work. The config overrides *what a role means* (budget), never *which stages belong to it* (taxonomy).

### Requirement: deprecated `fab resolve-agent <stage|role> [--alias] [--cli <name>] [--model <id>] [--effort <level>]` compatibility surface

`fab resolve-agent` is deprecated for skill dispatch; skills consume `fab agent <stage|role> -o yaml`. The compatibility command SHALL continue to accept a **stage** name OR a **role** name positionally — a stage maps through the fixed mapping, a role resolves directly, and **role names are checked first** (`agent.RoleForName`). The two name sets overlap only at **fixed points**: a name shared by a stage and a role (`review`, `hydrate`) is one where the stage maps to that same-named role (`stageRoles[name] == name`), so the role-first check resolves such a name to the identical profile either interpretation would (`ship` is a stage but NOT a role — it maps to `fast` — so `resolve-agent ship` and `resolve-agent fast` both resolve to the `fast` profile, one via the stage mapping, the other directly). It resolves the role → `{cli, model, effort}` per § Fill precedence and emits, **verbatim, with NO validation**:

- `model=<id>` (always; empty = the inherit signal),
- `effort=<level>` (omitted when empty),
- `provider=<name>` (omitted when empty) — the resolved CLI name, under the **pre-2.31 word**: the protocol is frozen byte-stable and the command deprecated, so no `cli=` line is added and the value is what `fab agent -o yaml` calls `cli`,
- `dispatch=<command>` — absent exactly when the shared mode selector lands on native; present with the substituted `interactive_command` for pane or `headless_command` for headless. Selection starts at `dispatch.mode`, descends pane → native → headless without ascending, and fails only when no reachable capability exists. A non-empty model with no native Agent-tool alias also lands off native (see § `dispatch.mode`). The command always embeds the resolved model verbatim, even under `--alias`.

`--alias` maps the `model=` line to the Claude-Code Agent-tool short alias (`opus`/`sonnet`/`haiku`/`fable`) — the Agent tool's `model` param is a hard enum that rejects full IDs; a model that already is an alias passes through as itself, and the `dispatch=` line is unaffected (resolved model verbatim).

**Invocation-time overrides** are the fill precedence's top rung, applied by re-running the chain with the flags on top (`agent.ResolveRoleWith`) rather than patching a resolved profile:

- **`--cli <name>`** swaps the CLI, refills model/effort from that CLI, and re-runs the same `dispatch.mode` ladder against its capabilities. The emitted `dispatch=` presence may therefore differ from the unoverridden profile. The pre-2.31 spelling `--provider <name>` stays as a hidden deprecated alias (§ Deprecated spellings stay readable).
- **`--model <id>` / `--effort <level>`** override the corresponding field and are valid **without** `--cli` — a within-role override of the profile this pure query would otherwise print. `fab agent` accepts the same bare overrides (a final post-refill layer on every addressing form), so the two commands agree on override legality.
- All three key on cobra's `Flag.Changed` — whether the flag was **supplied** — so `--model=` explicitly *clears* the role's model (emitting the inherit signal) rather than being silently ignored, and `--cli=` is a lookup failure rather than a fall-through to the knob.
- An **unknown `--cli` name** is a non-zero-exit **lookup** failure listing the resolvable names, byte-identical to `fab agent`'s because both call the shared `unknownCLIError(cfg, name)` helper in `cmd/fab`. Overrides themselves are applied with **no validation** (CLI neutrality).

**Overrides bind the native Agent-tool arm only.** The emitted `dispatch=` line is a **query result**, not an adapter move: `fab dispatch start` takes no override flags and re-resolves the stage from config itself, so an overridden profile cannot ride the CLI adapter. Relocating a stage between native Agent-tool dispatch and CLI dispatch takes a **config override** (a depth knob, or `agent.profiles.<role>.cli`), never an invocation flag. See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution for the dispatch-site wiring and the two-remedies rule.

#### Scenario: `--alias` aliases `model=` while `dispatch=` keeps the resolved ID

- **GIVEN** a role pinned to a versioned Claude ID and resolving to a CLI with a `headless_command`
- **WHEN** `fab resolve-agent <stage> --alias` runs
- **THEN** `model=` carries the short alias while `dispatch=` embeds the versioned ID verbatim
- **AND** under `--cli codex --model <codex-model-id> --alias` the non-Claude model passes through **verbatim** on `model=` (no prefix matched) while `dispatch=` embeds the same ID

#### Scenario: overriding a stage's CLI on the resolve call

- **GIVEN** a default config (apply → `doing` → claude)
- **WHEN** `fab resolve-agent apply --cli codex` runs
- **THEN** `provider=codex` with codex's own `doing` fill — its shipped `doing` model and effort — and `dispatch=codex exec --dangerously-bypass-approvals-and-sandbox …` with both placeholders substituted
- **WHEN** `fab resolve-agent apply --effort high` runs (no `--cli`)
- **THEN** `provider=claude` with the role's own model and `effort=high` — a within-role override, not a usage error
- **WHEN** `fab resolve-agent apply --cli bogus` runs
- **THEN** it exits non-zero naming `bogus`, lists `agy, claude, codex, kimi`, and prints no profile

### Requirement: One resolution object, one composer, two projections

`internal/agent.Resolution` is the complete resolution result shared by `fab resolve-agent` and `fab agent`. It carries `selector`, `kind`, `role`, `cli` (plus the deprecated duplicate `provider` with the same value, and `source.provider` alongside `source.cli`, filled for the deprecation window), the resolved `model` (verbatim after the `tool-default` collapse), `model_alias`, `effort`, the raw `template`, `fill_mode` (`template` or `append`), the composed `command`, per-field `source` provenance, and an optional `dispatch` value containing a labelled `rung`, the composed `command`, and (after a descent) a `reason`. A nil dispatch denotes the native rung.

`cmd/fab/resolution.go` owns the single profile-to-resolution composer. Both commands normalize their addressing and overrides into that composer, then render projections of the returned value: `Resolution.Lines(alias)` produces `fab resolve-agent`'s frozen ordered line protocol, while `Resolution.YAML()` produces `fab agent -o yaml`'s full schema. The composer uses `agent.ResolveRoleWithSource` for the profile and provenance, `spawn.IsTemplate` plus `spawn.WithProfile` for fill mode and command composition, and `dispatch.SelectMode` for the optional dispatch projection. `ResolveRoleWith` delegates to `ResolveRoleWithSource`, so fill precedence and its trace share one implementation; `spawn.IsTemplate` exports the discriminator already used by `spawn.WithProfile`.

Only the YAML sink on `fab agent` requests dispatch derivation. `--print`, exec, and `-t` consume the same resolution object without selecting a dispatch rung, while `-o yaml` and `fab resolve-agent` share the actionable no-capability error posture.

#### Scenario: line and YAML projections agree

- **GIVEN** the same selector, config, overrides, and tmux availability
- **WHEN** `fab resolve-agent` renders its line projection and `fab agent -o yaml` renders its structured projection
- **THEN** their CLI, model, effort, and non-native dispatch command come from the same `agent.Resolution`
- **AND** the line projection preserves its byte-stable `model=` / optional `effort=` / optional `provider=` / optional `dispatch=` contract

### Requirement: `dispatch.mode` — preference-bounded capability descent

`dispatch.mode` is `pane`, `native`, or `headless`, defaults to `native`, and has scope `both`. The project layer overrides the machine-wide system preference. It is a **ceiling**: selection starts at that rung and moves only downward through pane → native → headless.

- Pane is possible when tmux is available and the CLI has `interactive_command`.
- Native is possible when the CLI declares `native: true` **and** the resolved model is empty or has a native Agent-tool alias (`agent.HasNativeModel`: empty, `opus`/`sonnet`/`haiku`/`fable`, or a versioned Claude ID that maps to one). The predicate keys only on the model string, so it is CLI-neutral. An empty model keeps native (the Agent call omits `model` and inherits the session model); the reserved `tool-default` value resolves to empty before the predicate runs, so it is exempt from the descent below and keeps native.
- Headless is possible when the CLI has `headless_command`.
- The first possible rung wins. Selection fails only when no rung at or below the preference is possible.
- Invalid values warn and fall back to `native`. The retired boolean has no read-time alias.
- `fab agent <stage> -o yaml`, `fab dispatch start|restart|open`, and the `fab doctor` dispatch-mode probe compute native availability with the same predicate and use the same pure selector (`dispatch.NativeAvailability`: available / no native capability / model has no native alias); start/restart supply the real tmux reachability result before writing any state.
- Descent causes are strings: `pane unavailable: no tmux`, `pane unavailable: tmux unreachable`, `pane unavailable: no interactive_command`, `native unavailable` (CLI lacks `native: true`), and `native unavailable: model has no native alias`.
- When the alias rule leaves the ladder empty (the CLI declares `native: true`, has no `headless_command`, and the model has no alias), the error names the model and its missing native alias, offers `clis.<p>.headless_command`, an alias, an empty model, or `tool-default`, does not suggest configuring `native`, and still wraps `dispatch.ErrNoMode`.

#### Scenario: native preference descends for a non-native CLI

- **GIVEN** `dispatch.mode: native` and a role resolving to built-in codex
- **WHEN** `fab agent <stage> -o yaml` runs
- **THEN** native is skipped and `dispatch.command` carries codex's headless command
- **AND** `dispatch.reason` is `native unavailable`

#### Scenario: a model with no native alias descends to headless

- **GIVEN** `dispatch.mode: native`, built-in claude, and `agent.profiles.doing.model: glm-latest[1m]`
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `model_alias` is empty and `dispatch` carries `rung: headless`, claude's headless command with `--model glm-latest[1m]`, and `reason: native unavailable: model has no native alias`
- **AND** `fab dispatch start <change> apply` launches headless and prints `dispatch selection: mode: headless (descended: native unavailable: model has no native alias)` instead of refusing with native-dispatch guidance

#### Scenario: an empty or aliased model stays native

- **GIVEN** `dispatch.mode: native`, built-in claude, and a doing model that is empty, `tool-default`, `opus`, or `claude-opus-5-5` (including an empty `--model=` invocation override)
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** no `dispatch:` key is emitted

#### Scenario: pane preference never ascends

- **GIVEN** `dispatch.mode: pane`, no reachable tmux, and built-in claude
- **WHEN** the role is resolved
- **THEN** the selector descends to native and omits `dispatch:`; it does not jump directly to headless while native is possible

#### Scenario: headless preference ignores higher capabilities

- **GIVEN** `dispatch.mode: headless` and built-in claude, which also supports pane and native
- **WHEN** the role is resolved
- **THEN** `dispatch.command` carries claude's headless command because the selector never ascends

### Requirement: `fab agent [role|stage] [--cli <name>] [--model <id>] [--effort <level>] [--headless] [-t|--template] [-o yaml] [--workers <cli>] [--preset <name>] [-p|--print] [--repo <path>] [-- <agent-args>...]` — session launcher

`fab agent` SHALL compose an interactive session command through an **addressing form** and **exec it in the current shell** (or print it — one print-family sink per invocation):

- **Selector-addressed** (the `[role|stage]` positional) — resolve a role profile through the full chain (`default` when the positional is omitted; the six role names and the six stage names — `intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr` — are accepted; a stage name maps through the fixed `stageRoles` table to its role via `agent.RoleForName`, role set first, and the `review`/`hydrate` collisions are fixed points so either name resolves identically) and compose `clis.<profile.cli>.interactive_command` with the role's `{model}`/`{effort}`. Since `default` and `operator` are session roles, this is the path `agent.session` governs: `fab agent` starts the default-role agent right here, `fab agent operator` starts the coordinator profile.
- **Selector + `--cli`** — re-resolves the selector's role from the named CLI's own fills (`agent.ResolveRoleWith` with the CLI pinned): the CLI pin refills from `clis.<p>.profiles.<role>`, then that CLI's `.default`, then empty, while an explicit `agent.profiles.<role>.model` pin still wins. So `fab agent apply --cli kimi --print` prints kimi's own command shape with kimi's fills, not claude's model patched in.
- **CLI-addressed** (`--cli <name>`, no selector) — **bypass role resolution entirely**: look up `clis.<name>` directly via `agent.ResolveCLI` (project config per-field-merged over the built-in table, exactly as the selector path's CLI lookup does) and compose its `interactive_command` with the `--model`/`--effort` values. This is the "give me a codex session right here" form — no role need name the CLI first.

All forms compose through the same `spawn.WithProfile` (template substitution or Claude-style flag append — see [configuration.md](/_shared/configuration.md) § `clis`); `--model`/`--effort` are **general final overrides**, valid on every addressing form (bare, selector, selector+`--cli`, CLI-alone), applied after role resolution/CLI refill (verbatim, except that exact `tool-default` collapses to an empty model with `source.model: flag`) and keyed on the flag being **supplied** (cobra's `Flag.Changed`) — the YAML and deprecated line projections share these override semantics. The print-family sinks share `--repo`:

- `-p, --print` prints the fully-resolved command instead of executing — the output is **profile-resolved** (model/effort substituted), so callers that spawn from the printed command get the profile. `-p` is a pure shorthand, byte-identical to `--print`.
- `-t, --template` prints the selected CLI's command template **unsubstituted** — a tap before the fill step, `{model}`/`{effort}` placeholders intact. It implies print, combines with any selector, `--cli`, and `--headless` (they pick which template), and rejects `--model`/`--effort` with a usage error (they feed the substitution step that `-t` skips).
- `-o, --output yaml` emits the full resolution as structured YAML. `selector`, `kind`, `role`, `cli`, the deprecated duplicate `provider` (the same value as `cli`, emitted in the slot right after it), `model`, `effort`, and `command` appear first in that order, followed by `model_alias`, raw `template`, `fill_mode`, per-field `source`, an optional `preset` (the active preset's name — present only when a preset is active and the form is role-resolved, so absent for the bare `--cli` form), optional labelled `dispatch`, and a trailing always-present `skill_prefix`. `model_alias` is the Agent-tool short alias for Claude IDs and empty otherwise (a non-empty model with an empty `model_alias` never resolves native, so an empty `model_alias` on the native branch always means an empty model); `source` names the actual CLI/model/effort precedence rung — a preset-supplied field is labelled `agent_presets.<name>`, and a field re-derived from a preset-chosen CLI keeps its `clis.<p>.profiles.<role>` label — `cli` followed by a deprecated duplicate `provider` with the same value, then `model` and `effort`; rung names that denote config keys use the `clis.` / `.cli` spellings (empty means inherit; when `model` collapsed from `tool-default`, `source.model` still names the rung that held the token); `dispatch` is absent exactly for native and otherwise contains `rung: pane|headless`, its substituted `command`, and an `omitempty` `reason` after `command` — the descent causes joined with `; ` (the skip list, without the `mode: … (descended: …)` wrapper), absent when the selected rung is the preferred one; `skill_prefix` is the receiving harness's explicit skill-invocation prefix (`$` for `codex`, `/` for every other CLI — `agent.SkillPrefix` owns the rule, see [agent-primitives.md](/runtime/agent-primitives.md) § Skill Prompt Rendering) and is the last key whether or not `dispatch` is present; the deprecated `fab resolve-agent` line protocol never carries it. YAML alone derives dispatch. It implies print, accepts exactly `yaml` (anything else is a usage error), and is mutually exclusive with `--print` and with `-t` — one output sink per invocation.
- `--provider <name>` is the **hidden deprecated spelling of `--cli`** (`MarkHidden`, absent from `--help`): it behaves exactly as `--cli` and prints `--provider is deprecated, use --cli` on stderr; supplying both with different values is a usage error, equal values are accepted, and the explicit-empty `Flag.Changed` semantics carry over. One shared helper (`cmd/fab/cli_flag.go`) registers the pair on `fab agent`, `fab resolve-agent`, and `fab pane open`.
- `--headless` resolves the CLI's `headless_command` instead of `interactive_command`. It is valid only in the print-family modes (`--print`, `-t`, `-o yaml`) — exec of a headless command is a usage error. A CLI with no `headless_command` hard-errors naming the config key (`configure clis.<name>.headless_command`).
- `--repo <path>` reads the target repo's config (the operator's fetch-another-repo's-command use case). Composes with any addressing form.
- `--workers <cli>` sets `FAB_AGENT_WORKERS=<cli>` in the environment passed to the exec seam without changing any addressing form's command resolution. An entry inherited from the parent environment is removed rather than shadowed, so the override is authoritative regardless of how a consumer resolves duplicate entries. It is accepted with `--print`, but printed output remains the command alone because no child process is executed.
- `--preset <name>` chooses an agent preset for this one resolution, as if `FAB_AGENT_PRESET` were set — resolution-only (nothing is exported into the exec'd session) and rejected with a bare `--cli`; see § Agent presets. `fab agent presets [--json]` lists the defined presets.
- `fab agent` exec does NOT TTY-guard — exec-and-let-the-CLI-fail is acceptable (the underlying agent CLI already handles no-TTY), matching the document-don't-validate contract.

CLI-form rules (the **bare** `--cli <name>` form — selector forms take the fill rungs):

- **Omitted `--model`/`--effort`** leave the value empty and follow `WithProfile`'s empty-value rule (template mode drops the placeholder's token plus a preceding `-`-flag; append mode omits the flag), so a profile-free CLI invocation results and the installed CLI's own default model applies — the way to spawn a CLI whose current model IDs the caller does not know. CLI-level flags without placeholders remain in that command. **This path bypasses the CLI's per-role fills** (the bypass is total — it skips role resolution *and* the fill), so an empty flag means empty, not the configured fill. The fills apply on selector-addressed role resolution (`fab agent <stage|role> -o yaml` or selector+`--cli`), not on the bare launcher form.
- Mode selection and the guards key on cobra's `Flag.Changed` — whether the flag was **supplied** — not on its value being non-empty, so an explicitly-empty `--cli=` resolves through the lookup and an explicitly-empty `--model=` clears the field on the selector paths rather than being ignored.
- **An unknown CLI name** is a non-zero-exit **lookup** failure listing the available names (`agent.CLINames`: fab-kit's built-in table ∪ the project's `clis:` keys via `config.CLINames`, sorted). Listing resolvable *names* is not validation of a command's *content* — resolved strings still pass through verbatim.
- A CLI that resolves but carries no valid command slot yields a config-key hint error — `configure clis.<name>.interactive_command` by default, or `configure clis.<name>.headless_command` when `--headless` is set — on either addressing path.

Skill-bearing session launchers retain the receiving CLI alongside the composed command. Batch new/switch use the shared [skill prompt renderer](/runtime/agent-primitives.md); a Claude executable fallback also sets the rendering CLI to Claude. Skill prose takes the prefix from `fab agent … -o yaml`'s `skill_prefix` for fresh spawns and from live process evidence for existing panes.

The procedural knowledge for *using* a composed command — opening it in a tmux window, delivering a prompt, peeking, awaiting — plus the per-CLI invocation grammar and model-discovery recipes (each led by `fab agent models <cli>`) live in the `_cli-agents` helper: see [agent-primitives.md](/runtime/agent-primitives.md).

#### Scenario: a stage selector and selector+`--cli` re-resolve

- **GIVEN** a default config (apply → `doing` → claude; the built-in kimi ships empty fills)
- **WHEN** `fab agent apply --print` runs
- **THEN** stdout is the `doing` role's composed command, byte-identical to `fab agent doing --print`
- **AND** `fab agent apply --cli kimi --print` prints kimi's own command shape (`kimi --auto`) with kimi's fills — the empty `{model}` drops the `-m` pair — rather than claude's model patched into kimi's template
- **AND** `fab agent bogus --print` exits non-zero, naming the valid role and stage names

#### Scenario: CLI-addressed spawn with no model supplied

- **GIVEN** `clis.codex.interactive_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'`
- **WHEN** `fab agent --cli codex --print` runs
- **THEN** stdout is `codex --dangerously-bypass-approvals-and-sandbox` — both placeholder tokens and their preceding flags are dropped while the CLI-level bypass flag remains — so the CLI's own default model applies
- **AND** `fab agent --cli codex --model <codex-model-id> --effort high --print` prints `codex --dangerously-bypass-approvals-and-sandbox -m <codex-model-id> -c model_reasoning_effort=high`

#### Scenario: unknown CLI name

- **GIVEN** a config whose `clis:` block defines only `claude`
- **WHEN** `fab agent --cli bogus --print` runs
- **THEN** it exits non-zero, naming `bogus` and listing the available CLI names, and prints no command

### Requirement: `fab agent models [cli] [--json] [--repo <path>]` — model discovery

`fab agent models` SHALL list the models each agent CLI tool offers, as a read-only query: it writes nothing, caches nothing, and is never invoked from resolution, dispatch, or `fab config` — the external processes it launches (`codex debug models`, a `models_command`) run only when this verb does. It is a cobra child of `fab agent`; `models` is neither a role nor a stage name, so every other `fab agent` form (selector, `--cli`, `-o yaml`, `--print`, `--` passthrough) routes to the parent unchanged.

- **Scope.** No positional covers every name in `agent.CLINames(cfg)` (built-ins plus user-defined `clis:` entries), in that sorted order; one positional limits output to that CLI; more than one is a usage error. An unknown name is a non-zero lookup failure through the shared `unknownCLIError(cfg, name)` helper, listing the resolvable names. `--repo` reuses `loadRepoConfig`, so the verb also works outside a fab project (environment > system > built-ins).
- **Output.** `--json` prints one JSON array of rows to stdout — no envelope, `efforts` always an array. Without it, stdout is a column-aligned table with header `CLI MODEL LABEL EFFORTS SOURCE DEFAULT` (efforts comma-joined, `*` marking the tool default); the table is a human view, not a machine contract.
- **Fail-open.** A discovery failure never changes the exit code (0) and never empties the list: the CLI keeps its reserved row and kit fills and one line goes to **stderr**, `warning: <cli>: <reason>`, in both output modes. Non-zero exits are limited to usage errors, an unknown CLI, and a config-load error.

**Row schema** — `{cli, id, label, efforts, default, source}`:

| Field | Meaning |
|-------|---------|
| `cli` | the CLI name |
| `id` | the string fab would substitute into `{model}` |
| `label` | the tool's display name; falls back to `id` |
| `efforts` | the model's own list when the tool reports one, else the tool's known set, else `[]` |
| `default` | `true` only for the tool's own default model |
| `source` | `tool` (reported by the tool or its `models_command`), `alias` (a built-in Claude family alias), `fill` (a kit fill), or `reserved` (the `tool-default` row) |

**Reserved row.** Every CLI's first row — built-ins, `models_command` tools, and tools with no adapter such as agy — is `{id: "tool-default", source: "reserved", default: false}`, so a settings UI can identify it by `source` without string-matching the id. Its `label` is `tool default (<id of the default row>)` when discovery identified a default and plain `tool default` otherwise (discovery failed, agy, kimi without `default_model`); its `efforts` are the default row's when non-empty, else claude's known set for claude, else `[]`. The tool's actual default row keeps `default: true`; the reserved row is not a model. It is never deduplicated against or merged with other rows: a tool or `models_command` row whose id is literally `tool-default` is dropped, since the token is reserved. It is present on the fail-open path too.

**Merge.** After the reserved row, per CLI, rows are assembled from (1) the adapter's or `models_command`'s rows in the tool's own order, (2) the kit fills — every distinct non-empty `model` across `agent.ResolveCLI(cfg, name).Profiles`, so user fill overrides count and `agent.profiles` pins do not, sorted, `source: fill` — and (3) the tool-reported default, appended as `{source: tool, default: true}` when no row carries its id. Rows are de-duplicated by `id`: the first occurrence keeps its position, the higher source wins (`tool` > `alias` > `fill`; `reserved` never participates), `default` is OR-ed across duplicates, and an empty label or efforts on the winner is back-filled from the duplicate. Fills are always present, so the list is never smaller than the fill set.

**Per-tool adapters** (package `internal/agentmodels`; each returns rows, a default id, and an error that becomes the warning). Process launches run under a named 3-second timeout (`timed out after 3s`); file-reading adapters need none.

| CLI | Source | Rows | Default |
|----------|--------|------|---------|
| `codex` | `codex debug models` from `$PATH` | `models[]` with `visibility == "list"`: `slug` → `id`, `display_name` → `label`, `supported_reasoning_levels[].effort` → `efforts`; every other field ignored | top-level `model` of `$CODEX_HOME/config.toml` (`~/.codex` by default) |
| `claude` | `settings.json` under `$CLAUDE_CONFIG_DIR` (`~/.claude` by default), with `settings.local.json` from the same directory replacing its top-level `model` and `modelPicker` | the aliases `opus`/`sonnet`/`haiku`/`fable` (`alias`) unless `modelPicker.replaceBuiltInOptions` is `true`, then each `modelPicker.options[]` (`model` → `id`, `label`) as `tool`; every row carries efforts `low, medium, high, xhigh, max` | top-level `model` |
| `kimi` | `~/.kimi-code/config.toml` (no environment override) | each `[models."<alias>"]` (or bare `[models.<alias>]`) table: alias → `id`, `display_name` → `label`, `support_efforts` → `efforts` | top-level `default_model` |
| `agy` | none | fills only | — |

The codex surface is a debug command rather than a contract, so the parse is pinned to exactly the fields above and tested against a recorded fixture. Missing config files are not failures (no warning); a binary that is absent, exits non-zero, times out, or prints malformed or `models`-less JSON yields a warning plus fills; a malformed claude settings file warns but still yields the aliases. kimi's config carries credentials under `[providers.*]`, so nothing outside the model tables is ever copied into a row or a warning. agy and a user-defined CLI without `models_command` yield fills only, with no warning — absence of an adapter is not a failure.

**`models_command`.** When `clis.<name>.models_command` is set — on a built-in or a user-defined CLI — it replaces that CLI's adapter. It runs through `sh -c` under the same timeout and prints a JSON array of `{id, label?, efforts?, default?}`; fab sets `cli` to the CLI name and `source` to `tool` regardless of what the command prints, drops rows with an empty `id`, and treats a non-zero exit, a timeout, or malformed JSON as a warning plus fills. The key is optional, merges per field in `ResolveCLI` like the command fields, is settable through `fab config set`, is documented by `fab config explain clis`, and is not advertised in the managed fence (no built-in default to show); adding it restructures no user data, so no migration ships.

TOML is read by a small in-package line scanner: top-level `key = "string"` pairs, `[table]` / `[a."b"]` / `[[array]]` header tracking, and within a table `key = "string"` and single-line string arrays. A `key = value` line splits at the first `=` outside quotes, a value outside the consumed keys that the scanner cannot read (a multi-line array, an integer array) is skipped rather than failing the file, and a skipped value still open at end of file is a parse error — a warning plus fills, never a silently short read.

#### Scenario: no fab project, codex missing from `$PATH`

- **GIVEN** a directory with no `fab/` anywhere up the tree and no `codex` binary on `$PATH`
- **WHEN** `fab agent models codex --json` runs
- **THEN** stdout is a JSON array holding the reserved `tool-default` row (label `tool default`) followed by codex's kit fills, each `source: "fill"`
- **AND** stderr carries one `warning: codex: …` line and the exit code is 0

#### Scenario: claude rows merge aliases, fills, and the configured default

- **GIVEN** claude settings with no `modelPicker` and top-level `model: claude-fable-5-1[1m]`
- **WHEN** `fab agent models claude --json` runs
- **THEN** the rows are `tool-default` (`reserved`, label `tool default (claude-fable-5-1[1m])`), `opus`, `sonnet`, `haiku`, `fable` (`alias`, the `opus`/`sonnet` fills merged into them) followed by `claude-fable-5-1[1m]` (`tool`, `default: true`)

#### Scenario: `models_command` replaces the adapter and fab owns `cli` and `source`

- **GIVEN** `clis: { mytool: { interactive_command: mytool, models_command: "printf '[{\"id\":\"m1\",\"cli\":\"x\",\"source\":\"fill\"}]'" } }`
- **WHEN** `fab agent models mytool --json` runs
- **THEN** the rows are the reserved `tool-default` row followed by `{"cli":"mytool","id":"m1","label":"m1","efforts":[],"default":false,"source":"tool"}`

#### Scenario: agy yields fills only

- **WHEN** `fab agent models agy --json` runs
- **THEN** stdout holds the reserved `tool-default` row (label `tool default`, `efforts: []`) followed by agy's fills as `source: "fill"`, and stderr is empty

### Requirement: The machinery is documented, not scaffolded

The registry (`internal/configref`) advertises `agent.session` and `agent.workers` (`advertise: true`) and marks `agent.profiles` and `clis` `advertise: false`. So the `fab config upgrade` **managed fence** in each project scaffolds a ~20-line `agent:` block — the two knobs, the role→depth partition compacted to one line per depth, and a commented `profiles:` example — and no `clis:` block at all, while `fab config explain` (and `fab config explain --json`, and `fab config init --system`) still document both in full. The rendered partition lines derive from the exported `agent.IsSessionRole`, so the fence never re-encodes which roles sit at which depth. Because `agent` is one YAML block, the whole block is rendered by a **single** segment (owned by the `agent.session` row) — two segments emitting a live `agent:` parent would collide into a duplicate key. See [configuration.md](/_shared/configuration.md) § Schema Discovery.

### Requirement: Deprecated spellings stay readable

Deprecated spellings SHALL keep resolving so a config that has not yet run the corresponding migration is never silently ignored:

- **The pre-2.31 "provider" vocabulary** — the agent CLIs fab launches were called *providers* until 2.31.0, which collided with the tools' own use of the word for the model vendor. The old spellings stay accepted for **one minor release** after the one that ships the rename, and are removed no earlier than the following minor, in a separate change:

  | Old spelling | Canonical | During the window |
  |--------------|-----------|-------------------|
  | top-level `providers:` table | `clis:` | read as `clis:`, per entry and per leaf (recursing into `profiles.<role>`); the new spelling wins per field, an empty new leaf falls through to the old one, and an explicit `native: false` under either spelling still disables a built-in `true` |
  | `agent.profiles.<role>.provider` (and legacy `agent.tiers.<role>.provider`) | `agent.profiles.<role>.cli` | read as `.cli` under the same per-field rule |
  | env `FAB_PROVIDERS` | `FAB_CLIS` | read as a second env source for the `clis` key (`EnvOrigins["clis"]` names the variable actually used); `FAB_CLIS` wins; a malformed alias warns naming `$FAB_PROVIDERS` and is skipped |
  | `--provider <name>` on `fab agent`, `fab resolve-agent`, `fab pane open` | `--cli <name>` | hidden alias, stderr warning, conflicting values are a usage error |
  | `provider:` / `source.provider:` in `fab agent -o yaml`; `"provider"` in `fab pane open --json` | `cli:` / `source.cli:` / `"cli"` | emitted as a duplicate next to the new key with the same value; the bare-flag form's `kind:` value is `cli` (a scalar cannot be dual-emitted) |

  `fab resolve-agent`'s `provider=<name>` output line is **not** part of the window — it keeps the old word permanently because the line protocol is frozen byte-stable.

  Compatibility lives at exactly two seams in `internal/config` (`deprecated.go`), so no other package carries dual-spelling logic. `NormalizeDeprecated` folds the old spellings into the new ones on every **raw layer map** — project file, system file, environment overlay — before the cascade merge, so the resolver, `show`, `--origin`, `explain`, setup checks, and `fab agent models` only ever see the new shape and the typed `Config` carries no deprecated field. `config.CanonicalKey` (with `TwinKeys` and `DeprecatedUses`) maps a dotted key for the write-side and diagnostic surfaces. Because normalization precedes the merge, a half-migrated machine (system file on `providers:`, project on `clis:`) cascades exactly as same-spelling layers do — unlike the older twins below, the spelling never decides over the scope.

  Reads stay **silent**: agent resolution, dispatch, and plain or unkeyed `fab config show` print no deprecation warning, because skills parse `fab agent` output and per-call stderr noise would risk breaking captures. The old keys are answered or warned about only at these seams:

  - `fab config set` given an old key maps it through `CanonicalKey`, runs the value gate on the **mapped** key, writes the new key, removes a same-path old twin so the file holds one spelling (a set of the new key removes a live twin too — for `agent.profiles.<role>.cli` that includes `agent.tiers.<role>.provider`), reports the new key, prints `<old> is deprecated, wrote <new>` on stderr, and exits 0; `--system` follows the same path.
  - `fab config unset` given either spelling removes **both** spellings at that path (an unset of only the new key would otherwise leave the old override in effect), warning when the old key was named.
  - Keyed `fab config show <old key>` (also with `--origin`/`--json`) prints exactly what the canonical key prints, plus a stderr-only `fab: note: <old> is the deprecated spelling of <new>`; `fab config explain providers` answers with the `clis` explanation and a deprecation note (a leading comment line for YAML, stderr for `--json`).
  - `fab setup check` (`setupcheck.ProbeDeprecatedSpellings`) warns — never fails — on each live old spelling in either config file (`<old> is deprecated; rename to <new> in <file> (fab-setup migrations)`) and on a set `FAB_PROVIDERS`. There is no separate `fab config validate` command.
  - `fab config upgrade` carries a live top-level `providers:` key to `clis:` verbatim (the registry's `clis` row carries `renamed_from: providers`; the older `session_command`/`dispatch_command` history lives in the row's prose, since a row holds one machine-readable carry). The pre-rename fence digests are catalogued so an existing fence re-renders without a user-edit notice. The `2.30.1-to-2.31.0` migration rewrites both config files, including the nested `.provider` leaf and a both-live `providers:`/`clis:` merge — see [distribution/migrations.md](/distribution/migrations.md).

- **`agent.tiers.<role>`** — read per role as the fallback when `agent.profiles` has no entry for that role (`config.GetAgentProfile`), so a half-migrated config resolves every role. The registry row for `agent.profiles` carries `renamed_from: agent.tiers`; the `2.16.19-to-2.17.0` migration rewrites the disk.
- **`clis.<name>.model` / `.effort`** — the flat per-CLI fill, a true **alias** for `profiles.default`: `ResolveCLI` folds an override's flat fields into **that override's own** `profiles.default`, per field, before merging fab-kit's built-in table (`withFlatFillAlias`). So a user's flat pin outranks the shipped `profiles.default` it means to replace, while the user's own `profiles.default` still wins over their flat spelling — and a built-in **role** fill still outranks the folded value, exactly as it would outrank a hand-written `profiles.default`. Because the fold happens per field, a flat `model` reaches only the roles whose shipped row sets no model of its own — on codex, whose map is dense, that is the `default` role alone; on agy, every non-`fast` role.
- **`clis.<name>.session_command` / `.dispatch_command`** — the pre-2.19 spellings of the renamed command fields, read as **per-field fallbacks**: a non-empty `interactive_command`/`headless_command` wins, independently per field (`ResolveCLI`), so a half-migrated config resolves both fields. The alias is silent (no deprecation warning): the struct keeps both yaml tags, so decode retains whichever spelling a layer carries, and `ResolveCLI` applies the per-field preference after the cascade merges — so every cascade layer, env `FAB_CLIS` values included, resolves either spelling. The `2.18.1-to-2.19.0` migration rewrites both scopes on disk, and the `fab config set/unset/explain/show` dotted-key matcher accepts the new command spellings only — this alias is a read affordance, not a write surface (unlike the 2.31 vocabulary rename above, whose old keys `set`/`unset` map).

The read-time aliases are what make the rename safe on their own: `configupgrade`'s `renamed_from` carry is a **top-level-key** operation and deliberately skips a rename *inside* the `agent:` block, so without the aliases `fab config upgrade` (auto-run by `fab upgrade-repo`) would leave a live `agent:` block that had silently stopped being read. The migration `2.16.19-to-2.17.0` performs the on-disk rewrite in **both scopes** (project `fab/project/config.yaml` and system `~/.fab-kit/config.yaml`) and warns on the two shapes it cannot mechanically preserve: an `agent.tiers.default` carrying model/effort (whose cross-role re-basing has no successor) and a flat CLI fill, whose **reach** differs across the 2.17.0 boundary in opposite directions per CLI — a gain of one role on claude (exhaustive six-role map, where a flat fill is inert pre-2.17.0 and fills the `default` role after), a loss on any sparse-map non-claude CLI (those maps shipped no fills pre-2.17.0, so a flat fill that reached every role stops wherever fab-kit ships a role fill). The rewrite itself is resolution-neutral where both scopes are swept together; the one shape where it is not is the mixed-layer inversion below, which the migration names in its verification steps. See [distribution/migrations.md](/distribution/migrations.md).

**Cross-scope precedence inverts during the pre-migration window.** The alias resolves *after* the scope cascade: `LoadPath` merges the system and project layers per key first, leaving `profiles` and `tiers` as two separate maps, and `GetAgentProfile` then prefers `profiles` wherever it carries the role. So for a role written in the **new** spelling in one scope and the **legacy** spelling in the other, the spelling decides rather than the scope — a project-layer `agent.profiles.<role>` beats a system-layer `agent.tiers.<role>`, inverting the documented system > project precedence. It bites only a hand-half-migrated pair of scopes, and running the migration (which sweeps both files) restores normal precedence. Pinned by `TestResolveCrossScopeLegacyAliasPrecedence` in `cmd/fab` — the layer that can compose both scopes — and stated at `GetAgentProfile`'s doc comment.

**The CLI-side flat fill has the same twin.** `withFlatFillAlias` also runs *after* the scope cascade, so a project-layer `clis.<name>.profiles.default` beats a system-layer flat `clis.<name>.model` per field — the spelling decides rather than the scope, the same inversion one layer over. Retiring both twins needs a layer-aware fold (aliasing before `config.MergeLayers`, or handing the resolver per-layer inputs); until then the shape is documented in the `2.16.19-to-2.17.0` migration's verification steps as the mixed-layer exception.

**The renamed command fields inherit the same shape.** `MergeLayers` merges the raw YAML trees per key and the command-field fallback runs after, so during the migration window a system-scope `interactive_command` outranks a project-scope `session_command` for the same CLI — the spelling decides rather than the scope, consistent with new-spelling-wins-per-field. Deliberate: sweeping both scopes with the `2.18.1-to-2.19.0` migration restores normal precedence.

#### Scenario: a legacy `agent.tiers:` block still resolves

- **GIVEN** a config carrying only `agent: { tiers: { doing: { effort: medium } } }`
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `effort: medium` — the legacy key still resolves
- **AND** a config carrying **both** `profiles.doing` and `tiers.doing` resolves from `profiles.doing`

## Design Decisions

### The Depth Knob Is the CLI Rung, Not a Whole-Profile Switch
**Decision**: `agent.session`/`agent.workers` supply only the **CLI** rung of the chain; model and effort always come from the resolved CLI's own per-role fills. The role→depth partition behind them is fixed and fab-owned (`default`/`operator` → session; `doing`/`review`/`hydrate`/`fast` → workers), exposed to renderers through `agent.IsSessionRole` so no consumer re-encodes it.
**Why**: It is what makes "claude for tier 1, codex for tier 2" a two-word config while still letting each CLI keep sensible per-role models. A knob carrying a whole profile would name one model for every role it governs, flattening the role taxonomy and duplicating the per-role fills the CLI table already owns. Depth is the right axis because it is mechanically real: a session role's CLI is fixed when the session launches, a workers role's is re-resolved at every dispatch.
**Rejected**: Making the knob a `{cli, model, effort}` object (one model for every role it governs — the role differentiation the per-role fills exist to provide, lost in the advertised surface); keying the partition off stage names (the partition is about *depth*, and `default`/`operator` are not stages); a user-overridable partition (fab-owned taxonomy, like the stage→role map).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

### A Preset Is a Rung Spliced at the Choosing Layer
**Decision**: A preset is a sparse bundle of existing `agent.*` fields selected by one scalar (`agent.preset`), and exactly one is active — the highest layer's choice. Its fields are spliced in once, in the config loader, at the layer that chose it, directly below that layer's own explicit `agent.*` fields; provenance rides on `Config` so `internal/agent` can label `source.*`.
**Why**: Layer identity (env/system/project) exists only in the loader — `internal/agent` receives an already-merged `*Config` — so the positional rule cannot be expressed downstream, and one implementation behind both the typed and the provenance load keeps `fab config show` and every consumer on the same tree. Selecting by config (env layer included) reaches `fab dispatch start` in all three dispatch modes, which a flag cannot; one scalar gives HexoKit a single per-tab value (`FAB_AGENT_PRESET`) without parsing or rewriting config.
**Rejected**: Splicing in `internal/agent` (layers are gone); merging presets from several layers (an unpredictable union — one name wins whole); exporting `FAB_AGENT_PRESET` from `fab agent --preset` (it would leak into the launched session and imply it reaches dispatch); an `active` flag in `fab agent presets --json` (activeness is per-resolution context, not a property of a definition); storing a preset on a change.
*Introduced by*: 261124-e9nz-agent-presets

### One Fallback Chain, on the CLI Side Only
**Decision**: Cross-role fallback exists exactly once, as `clis.<p>.profiles.default`. The agent side has none — `agent.profiles` is sparse and `agent.profiles.default` is the `default` role's own override, never a fallback source for another role. Model/effort are always sourced from the **resolved** CLI's fills, so a CLI swap re-derives them.
**Why**: A single CLI-anchored chain makes cross-CLI leakage structurally impossible — no value can be foreign to the CLI that will run it — so no cutoff rule, per-field ownership tracking, or net-configured-CLI gate is needed to police it, and there is no cross-scope ownership limitation to document. Two competing chains are what force such a rule into existence.
**Rejected**: Agent-side `default`-role inheritance plus a cross-CLI field cutoff (preserves the footgun and its documentation debt, including a cascade-blind ownership computation); erroring when a role resolves an empty model (an empty model is the established inherit-the-session-model / CLI-default signal); validating CLI/model compatibility (breaks CLI neutrality).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

### Read-Time Aliases Back a Rename, Not Just the Migration
**Decision**: Every renamed CLI/agent key stays readable as a deprecated alias in the resolver, positioned below its modern counterpart in the chain, while the migration performs the on-disk rewrite in both scopes. Three families carry it: `agent.tiers` (per role), the flat `clis.<p>.model`/`.effort` (per field, folded into `profiles.default`), and `clis.<p>.session_command`/`.dispatch_command` (per field, against `interactive_command`/`headless_command`). All three are silent — no deprecation warning — and all three are **read** affordances: the `fab config` write surface accepts the modern spelling only. The `providers` → `clis` vocabulary rename is the one deliberate exception, with its own window and write-side mapping (the two Design Decisions below).
**Why**: The registry's `renamed_from` carry-forward is a **top-level-key** operation in `configupgrade`, so it deliberately skips a rename sitting *inside* the `agent:` block or inside a `clis.<name>:` block — `renamed_from` alone is metadata, not a working carry. Without a read-time alias, `fab config upgrade` (auto-run by `fab upgrade-repo`) would leave a live block that silently stopped being read until the user ran `/fab-setup migrations`. Per-field/per-role preference rather than whole-block preference is what makes a **half-migrated** config resolve completely instead of resolving half of it and dropping the rest.
**Rejected**: Teaching `configupgrade` nested rename carry (a splice-engine change far beyond the scope, and the migration already does it correctly); relying on the migration alone (a silent behavior-change window for every user between upgrade and migration run); whole-block preference (one modern key would shadow every sibling still in the old spelling).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers; 260809-n1he-rename-provider-command-fields

### Command Fields Name Interaction Mode, Not Depth
**Decision**: See the authoritative record in [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "CLI Command Fields Name Their Interaction Mode, Not the Agent's Depth". In brief: `interactive_command`/`headless_command` split by how the agent is run, not by the depth of the agent running it — a Tier-2 pane worker runs the interactive command — and the `fab config` dotted-key write surface accepts those spellings exclusively.
**Why**: Depth-flavored names advertise an invariant the pane rung breaks, and `dispatch_command` collides with both the `fab dispatch` verbs and the `dispatch.*` block.
**Rejected**: Depth-aligned names; a hard break with no alias; old spellings on the write surface.
*Introduced by*: 260809-n1he-rename-provider-command-fields

### CLIs Extracted; Roles; `fab agent` Retires `fab spawn-command`
**Decision**: See the authoritative record in [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "CLIs Extracted; Roles; `review_tools` → `code-review.md`". In brief: a top-level `clis:` table carries independent session/native/headless capabilities plus fills; roles resolve `{cli, model, effort}`; dispatch preference stays separate; review policy moved to `code-review.md`; `fab agent` owns session launch and structured resolution; the deprecated `resolve-agent` projection exposes `dispatch=` plus `provider=`.
**Why**: Conflating CLI mechanics with role/budget policy actively confused (two fields both named `spawn_command`; the `thinking` name hid its referent, which was review). Extraction plus role naming attack the confusion at its source; commands living on the CLI make `{cli, model, effort}` composition safe (no cross-semantics command inheritance).
**Rejected**: Merging the two command fields; folding a command in as a `default`-role field (implies the rejected cross-fallback); keeping `thinking`; CLI inference from model strings; a `fab spawn-command` deprecation alias.
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### Deprecated Spellings Normalize at the Raw-Layer Seam
**Decision**: One `internal/config` function (`NormalizeDeprecated`) folds `providers:` and `.provider` into `clis:` and `.cli` on every raw layer map — project, system, env — before `MergeLayers`, the new spelling winning per leaf, and one key-mapping helper (`CanonicalKey`/`TwinKeys`/`DeprecatedUses`) serves the write-side and diagnostic surfaces. The typed `Config` carries no deprecated field and `RoleProfile` no `Provider`.
**Why**: Config is merged as generic maps and decoded once, so normalizing the maps means the resolver, `show --origin`, `explain`, setup checks, and `fab agent models` need no dual-spelling logic. It also keeps the cascade spelling-neutral: a system file on `providers:` and a project file on `clis:` merge exactly as same-spelling layers do.
**Rejected**: A deprecated `Providers` struct field merged in each getter (the `SessionCommand` style) — it leaks into every consumer and into `show`, which renders raw maps, and reintroduces the spelling-over-scope inversion of the older twins.
*Introduced by*: 261120-f2ly-rename-provider-to-cli

### Deprecation Warnings Fire Only at Write, Diagnostic, and Flag Seams
**Decision**: Ordinary reads of an old spelling are silent. The warning fires on `fab config set`/`unset` with an old key, the hidden `--provider` flag, keyed `show`/`explain` of an old key (a note), `fab setup check`, and the migration. `set`/`unset` additionally map and clean the old key rather than refuse it, and `unset` removes both spellings at the path.
**Why**: Skills parse `fab agent` output, and released HexoKit reads `providers.` rows and writes `agent.profiles.<role>.provider` through `fab config set`, so a hard break or a per-load warning would break captures and a released consumer; mapping the write keeps the file on one spelling, and unsetting only the new key while an old twin stayed live would silently leave the override in effect. The window is one minor release, then removal in a separate change.
**Rejected**: Warning on every config load (stderr noise in captured output); refusing old keys on `set` as the 2.19 `session_command` rename did (a released writer would fail); leaving a same-path old twin after a `set` or `unset` (a stale key that later shadows or confuses an `unset`).
*Introduced by*: 261120-f2ly-rename-provider-to-cli

### `resolve-agent` Keeps the Old Word in Its Frozen Line
**Decision**: `fab resolve-agent`'s `provider=<name>` line is unchanged by the rename — no `cli=` line is added — while the Go field it reads is renamed.
**Why**: The line protocol is documented byte-stable and the command is already deprecated, so changing it would break the one property it exists for. `fab agent -o yaml`, which carries the `cli:` key (and the deprecated `provider:` duplicate), is the surface skills consume.
**Rejected**: Dual-emitting a `cli=` line (breaks byte-stability for every line-protocol caller).
*Introduced by*: 261120-f2ly-rename-provider-to-cli

### Positional Stage-or-Role; `provider=` Line; No TTY Guard
**Decision**: The deprecated `fab resolve-agent` compatibility surface accepts a stage OR role name positionally (role names checked first; shared names are fixed points), carries a `provider=` line, aliases only `model=`, and leaves `dispatch=` commands on the resolved model verbatim. `fab agent` exec does not TTY-guard.
**Why**: Reuse the existing positional surface rather than add flag surface for no disambiguation benefit; surface the CLI rather than re-derive it downstream; keep the no-validation/document-don't-guard contract for TTY.
**Rejected**: A `--role` flag (surface for no benefit); inferring CLI downstream (re-does resolution); a TTY guard (the agent CLI already handles no-TTY).
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### Resolution Composition Lives Beside Both CLI Consumers
**Decision**: `internal/agent` owns `Resolution`, its line/YAML projections, and the traced role resolver; `cmd/fab/resolution.go` owns the single composer used by both `fab resolve-agent` and `fab agent`.
**Why**: `internal/spawn` imports `internal/agent`, so composing through `spawn.WithProfile` inside `internal/agent` would create an import cycle. Both CLI consumers already share package `main`, which provides one composition site without changing the package graph.
**Rejected**: Moving `WithProfile` into `internal/agent`; creating an `internal/resolution` package for a composer with only two package-main consumers.
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

### Provenance Is Captured During Fill Resolution
**Decision**: `ResolveRoleWithSource` returns the resolved profile with the CLI/model/effort source rungs, and `ResolveRoleWith` delegates to it while discarding only the trace.
**Why**: Capturing the source when each precedence rung is consulted preserves a single fill-precedence implementation and distinguishes equal values supplied by different rungs.
**Rejected**: Inferring provenance after resolution by comparing the final value with every candidate rung.
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

### Structured YAML Shares the Resolver's Dispatch Failure Semantics
**Decision**: `fab agent -o yaml` derives dispatch for role, stage, and CLI addressing and returns the same actionable no-capability error as `fab resolve-agent`; every other `fab agent` sink skips dispatch derivation.
**Why**: Structured consumers receive the same dispatch truth as the frozen line surface, including the distinction between native absence and an unreachable ladder.
**Rejected**: Omitting `dispatch` when selection fails; deriving dispatch only for stage selectors; adding dispatch-derived errors to `--print`, exec, or `-t`.
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

### `hydrate` Is Its Own Role; `fast` Keeps Its Role Name (Multi-Referent)
**Decision**: `hydrate` is its own role rather than part of `doing`, giving six roles (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`). A role is stage-named only where it maps 1:1 to a single referent (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is multi-referent — `fast` in particular governs both the ship stage and the `/fab-proceed` prefix-step dispatches.
**Why**: Memory writing (hydrate) is knowledge work with a different cognitive profile than apply's diff work, so it deserves its own model/effort — grouped under `doing` it could never run cheaper or on a different model than apply. A stage name (`ship`) would misname `fast` once it also governs the prefix steps.
**Rejected**: Naming the role `ship` (misnames a multi-referent role and would force an unnecessary carry-forward migration); six stage-named roles / dissolving roles entirely (`default`/`doing`/`fast` are genuinely multi-referent and the role names carry the why); a user-overridable stage→role mapping or per-stage escape hatch (taxonomy stays fab-owned).
*Introduced by*: 260719-g55d-stage-model-tier-defaults-v2

### Bare `--cli` Is a Total Bypass; Selector+`--cli` Re-Resolves
**Decision**: `fab agent --cli <name>` with **no selector** bypasses role resolution entirely — a direct `clis.<name>` lookup with `--model`/`--effort` as the whole profile — rather than synthesizing an ad-hoc role. A **selector combined with `--cli`** re-resolves the selector's role from the named CLI's own fills (`agent.ResolveRoleWith` with the CLI pinned, the CLI rung of the same precedence chain). `--model`/`--effort` are general overrides, valid on every addressing form and applied verbatim after resolution/refill. All guards key on cobra's `Flag.Changed` (was the flag supplied?) rather than value emptiness.
**Why**: The bare-CLI form answers a different, mechanical question ("give me a codex session") from role resolution, and the bypass leaves the resolution path untouched. The pin form occupies exactly the precedence chain's existing CLI rung, so combining a selector with `--cli` is unambiguous — an explicitly-empty `--cli=` or `--model=` falling silently through would lose that signal, making supplied-ness the correct test.
**Rejected**: A role-addressed CLI pin treated as the total bypass (the selector form refills through the chain rather than bypassing it); auto-creating a synthetic role (invents state the config never declared); silently ignoring `--model`/`--effort` on any form (surprise-inducing CLI behavior).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn; *Updated by*: 260901-77vz-fab-agent-surface-extension

### One Output Sink per Invocation on `fab agent`
**Decision**: `-o yaml` is mutually exclusive with `--print` and `-t` (usage errors); `-t` with `--print` is tolerated (`-t` implies print).
**Why**: Each print-family flag selects a different sink (command line / template / structured doc); silently preferring one over another would make output shape depend on flag order.
**Rejected**: Precedence ordering (surprising, undocumentable cheaply); allowing `-o yaml -t` to select two output sinks (the full YAML document already carries the raw template as data).
*Introduced by*: 260901-77vz-fab-agent-surface-extension

### Selector Detection Reuses `agent.RoleForName` — No New Hoist
**Decision**: Selector-kind detection in `cmd/fab/agent.go` reuses the exported `agent.RoleForName` (+ `agent.IsRoleName` for `kind` reporting); no new `internal/agent` symbol was hoisted.
**Why**: The role-first/stage-fallback resolution already exists with the fixed-point guarantee tested (`review`/`hydrate` colliding names map to the identically-named role), so a new helper would duplicate the probe.
**Rejected**: A new `SelectorKind` helper in `internal/agent` — premature.
*Introduced by*: 260901-77vz-fab-agent-surface-extension

### Launch Flags Export One Shared Variable
**Decision**: All four `--workers` surfaces export `FAB_AGENT_WORKERS`; `fab agent` uses its exec environment and the tmux launchers use a shell-quoted assignment prefix.
**Why**: The spawned process tree reaches the same `LoadPath` seam used by both resolver and dispatch re-resolution, with no second selection mechanism.
**Rejected**: New resolution flags on dispatch, tmux-version-specific `new-window -e`, CLI validation, or persisted config edits.
*Introduced by*: 260808-2d1w-env-override-layer-launch-flags

### Built-in CLIs Ship Grammar AND Per-Role Fills, Refreshed at Kit-Release Cadence
**Decision**: fab-kit's built-in CLI table carries both halves — invocation grammar *and* per-role fills — inside the binary, as rows in the embedded `defaults.yaml`, for every CLI tool whose `-m` takes a portable model identifier (claude, codex, agy). claude's fills are family aliases that track the installed tool (§ "Claude Fills Are Family Aliases"); the non-claude fills are concrete IDs and carry no staleness automation: no CI check against provider APIs, no model-catalog fetch. They are refreshed at **kit-release cadence** by editing the file, pass through **unvalidated** like every other resolved string, and are corrected by **one config line** (`clis.<name>.profiles.<role>.model`, settable once per machine because `clis` is `scope: both`). They are never seeded into a user's `config.yaml`, so an upgrade refreshes them and no project pins rot in place.
**Why**: Shipping no fills was the *silent* failure: `agent.workers: codex` resolved an empty model identically for all four workers roles, so the CLI CLI's own default ran everywhere and the role taxonomy flattened exactly where a user first exercises the knob. A stale ID is the loud, cheap failure by comparison — the CLI rejects it immediately and the fix is one line. Two more facts close the gap the rot argument was defending: a bump is a reviewable data diff in an embedded YAML file, pinned by a test so it is always deliberate; and fab-kit releases every few days, so users see refreshed suggestions at kit cadence rather than CLI cadence. presence=intent is untouched — a built-in CLI is inert until a knob, role override, or flag names it, so adding fills changes no default behavior while both depth knobs ship `claude`.
**Rejected**: Keeping fills out of the release (preserves the silent role-flattening for the flagship one-line UX); a CI staleness check or catalog fetch against provider APIs (fab is validation-free by constitution, and a network dependency in resolution is worse than a stale string); seeding the fills into each project's `config.yaml` (pins in every repo, rotting independently of the binary); inferring a CLI from a model string (breaks CLI neutrality).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill; *Updated by*: 260806-ywkx-ship-codex-gemini-fills, 260808-rpsr-remove-gemini-add-agy-kimi, 261117-vyrt-claude-fills-opus-sonnet-aliases

### Non-Claude Built-in Commands Run Full-Auto
**Decision**: Both codex command forms carry `--dangerously-bypass-approvals-and-sandbox` and both agy command forms carry `--dangerously-skip-permissions`; kimi carries `--auto` on its interactive command and no approval flag on its headless one, because `kimi -p` is already non-interactive, auto-approves tool calls, and *rejects* `--yolo`/`--auto`. Project and system CLI-command overrides remain the approval-gated escape hatch.
**Why**: Headless and pane stage workers are unattended and have no approval-answering channel; ship and review-pr also require network and repository operations. Explicit bypass grammar gives the non-claude built-ins the same autonomous execution policy as claude's shipped `--permission-mode bypassPermissions` command — expressed in each CLI's own vocabulary, including "no flag" where the CLI's headless mode already implies it.
**Rejected**: Codex `--full-auto` (retains a workspace-write sandbox that blocks required network operations); bypassing only `headless_command` (leaves pane workers gated); adding an approval flag to kimi's dispatch form for symmetry (the CLI errors on it); approval-gated built-ins or per-project fixes (break the complete one-knob CLI swap).
*Introduced by*: 260808-clxw-codex-gemini-bypass-flags; *Updated by*: 260808-rpsr-remove-gemini-add-agy-kimi

### Codex's Fills Are Catalog Slugs; agy's Embed Effort in the ID; kimi Ships None
**Decision**: `clis.codex.profiles` ships concrete model slugs for all six roles. `clis.agy.profiles` ships two model-only rows (`default`, `fast`) and no `effort` anywhere. `clis.kimi` ships no `profiles` map at all.
**Why**: Each CLI's own model-addressing vocabulary decides the shape. The codex CLI exposes no alias mechanism (`-m` takes a slug), so concrete IDs are the only option; the shipped ones are read from the installed CLI's own cached model catalog (`~/.codex/models_cache.json`: slug, priority, description, supported reasoning levels) rather than from documentation, and the supported reasoning levels are checked there too. agy's IDs *embed* the reasoning level as a suffix (`gemini-3.1-pro-high`), so effort has no separate field to occupy and a `--effort` flag would fight the suffix. kimi's `-m` takes a **user-config alias** rather than a catalog ID, so there is no value that is correct for every install — the empty model drops the flag and the user's own `default_model` applies. For agy, sparseness buys role differentiation cheaply: rows omitted from its map inherit `default` through the per-field merge.
**Rejected**: Pinning kimi's managed-install alias `k3` (breaks custom-CLI installs) or giving it a lone `default` fill (same problem, one row smaller); a separate `--effort` flag on agy alongside its suffixed IDs; codex slugs quoted from docs or memory rather than the installed catalog (placeholders carried in from an intake were absent from it entirely, and the next tier down carried deprecation notices).
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills; *Updated by*: 260808-rpsr-remove-gemini-add-agy-kimi, 260908-wcib-refresh-codex-builtin-fills

### The Codex Fill Map Is Dense by Policy; Author and Critic Run Different Models
**Decision**: `clis.codex.profiles` ships all six roles with both `model` and `effort` set. `default`/`doing` take the catalog's top model at `high`; `review` takes a different model at `xhigh`; `hydrate` the same second model at `high`; `operator`/`fast` the cheap model. `TestDefaultsFileCLIs` rejects any codex row without a model and any row shipping `ultra`; `max` is an explicit-override level only.
**Why**: Effort-only rows made a `default` bump silently repoint `doing`/`review` through the per-field merge, and a reader could not see what a role resolved to without knowing that rule. Putting the critic on a different model than the author keeps the reviewer from sharing the author's blind spots (the 2026-08-10 four-CLI comparison in `docs/findings/` recorded reviewer strictness tracking the worker model on every arm). Codex's `ultra` level means automatic task delegation, and a worker spawning its own sub-agents breaks the single-worker dispatch contract and the `{stage}-result.yaml` protocol. Per-model codex pricing was not available when the table was set; the tier ladder is inferred from codex's own model descriptions and priority order, not measured. One consequence rides the density: the deprecated flat `clis.codex.model` alias reaches only the `default` role (§ Deprecated spellings) — an upgrade note in `docs/specs/stage-models.md`, not a migration, since nothing on disk restructures.
**Rejected**: Keeping effort-only rows (the inheritance is the defect); the same model for author and critic (shared blind spots); `gpt-5.6-terra` as a default (no evidence in this repo; A/B candidate only); a migration for the flat-alias reach change (no config key changed).
*Introduced by*: 260908-wcib-refresh-codex-builtin-fills

### The Nested-Shell `"$(cat)"` Idiom Delivers stdin to an Argument-Taking CLI
**Decision**: agy's and kimi's `headless_command`s wrap the CLI in `sh -c '… -p "$(cat)"'` rather than invoking it directly.
**Why**: `fab dispatch` delivers the stage prompt on **stdin**, but both CLIs take the prompt as an *argument* to `-p` and never read stdin. POSIX expands `$(cat)` **before** the `< file` redirect applies, so the un-nested form reads the *outer* stdin and the worker starts with an empty prompt; nesting a shell makes the inner `sh`'s stdin the redirected prompt file. Verified end-to-end in dispatch shape (`cmd < prompt > log 2>&1`) against both CLIs. Absorbing the difference in the CLI's grammar keeps the dispatch machinery CLI-neutral — no per-CLI prompt-delivery branch in Go.
**Rejected**: Passing `-p` with no argument (both CLIs error: `flag needs an argument: -p`); teaching `fab dispatch` a per-CLI prompt-delivery mode (CLI mechanics belong in the CLI's command string).
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi

### Pane Capability Follows Launch Grammar Plus the Generic Gate
**Decision**: Every built-in CLI ships pane launch grammar, and first-run walls are handled through the CLI-neutral readiness gate. agy carries the exact interactive grammar `agy --dangerously-skip-permissions --model {model}`; its exact workspace path may be user-seeded under `trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json`, while fab does not write that CLI-owned store.
**Why**: Interactive operation is common to agent CLIs, while headless prompt grammar is CLI-specific. The readiness gate already separates launch from input and classifies boot and first-run walls without CLI branches.
**Rejected**: Withholding `interactive_command` because a CLI has a first-run wall; encoding a separate pane-eligibility bit; CLI-specific trust-wall handling or fab-owned trust-store writes.
*Introduced by*: 260810-ttff-agy-interactive-pane-capability

### Exec contract for CLIs
**Decision**: A CLI's `interactive_command` must exec its binary, so the binary — not a wrapper shell — owns the pane's foreground; all four built-ins already do. The contract is stated once in the CLI-authoring guidance (`_cli-agents.md`); a wrapper CLI fails observably (`booting` with the shell prompt in the snippet, then escalation), so no escape hatch is needed and no new field is added.
**Why**: The readiness gate's takeover precondition keys on the pane's foreground command, so takeover is the one shape requirement a pane CLI's command must satisfy; with the gate enforcing it visibly, the contract needs no mechanism beyond documentation.
**Rejected**: Time-bounding the precondition (reopens the cooked-tty false-ready window after N seconds); comparing against `spawn_cmd` (brittle across `bash -c` argv shapes).
*Introduced by*: 260829-57mp-pane-readiness-agent-takeover

### kimi Ships the Probed `kimi --auto -m {model}` Verbatim
**Decision**: kimi's built-in `interactive_command` is exactly `kimi --auto -m {model}` — the invocation probed live against the CLI (2026-08-10, kimi 0.34.0) — and the value is pinned by a test rather than merely asserted present.
**Why**: A pane-eligibility field is only as good as the invocation it carries, and this one is proven end-to-end — it is the string a user-level `clis.kimi.interactive_command` override ran real pane workers with. `--auto` is where kimi's full-auto posture has to live, since the headless `-p` form rejects it; the `-m {model}` pair costs nothing because kimi ships no fills, so the empty model drops the pair and the CLI's own `default_model` applies. Pinning the value keeps a plausible-looking edit from shipping an unprobed invocation to every pane worker.
**Rejected**: `kimi --yolo -m {model}`, the form the older documentation illustrated (unproven against the delivery choreography, where `--auto` is what the probe actually used); presence-only assertion (any edit would pass); leaving the command to per-user config (the capability is a property of the CLI, not of one machine).
*Introduced by*: 260810-ki9v-kimi-pane-enablement

### A Built-in Roster Change Ships No Migration
**Decision**: Adding or retiring a built-in CLI is a data edit to `defaults.yaml` plus its drift-guarded mirrors — no migration file, no compatibility alias, no special-case code path for a name the table does not define.
**Why**: The `clis:` table is user-extensible and CLI names are opaque pass-through strings, so a config naming a CLI fab-kit does not define already has a defined behavior: the ordinary unknown-CLI lookup error, which names the resolvable set. That is a loud, one-line-fixable failure, and a user who wants a retired built-in back writes its block into their own config. A migration would have to guess a replacement CLI for a role the user chose deliberately.
**Rejected**: A rewrite migration mapping a retired name onto a surviving CLI (guesses intent, and silently changes which model runs a user's stages); a deprecation alias keeping the old name resolvable (carries dead grammar at release cadence — the cost the retirement removes).
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi

### The Flat CLI Fill Is a Real Alias for `profiles.default`
**Decision**: `ResolveCLI` folds an override's deprecated flat `model`/`effort` into **that override's own** `profiles.default`, per field, before merging fab-kit's built-in table — rather than reading the flat value as a rung *below* `profiles.default` during fill resolution. The user's own `profiles.default` wins where it sets a field; a built-in **role** fill still outranks the folded value.
**Why**: The flat spelling is *documented* as an alias for `profiles.default`, but was implemented as a lower-precedence rung. The two were indistinguishable while no non-claude built-in carried a `profiles.default`; shipping one makes the difference load-bearing, and the rung form would silently shadow a pre-migration user's own pinned model with fab-kit's shipped one — a regression introduced *by* shipping the fills, for exactly the un-migrated configs the alias exists to serve. Fixing the fold rather than the symptom also collapses `cliFill` to a two-rung role→default read.
**Rejected**: Keeping the rung and accepting the shadowing (a silent regression for the configs the alias serves); special-casing "built-in vs user" inside `cliFill` (it receives an already-merged `CLIConfig` and cannot tell them apart).
**Known consequence**: the fold runs *after* the scope cascade, so a system-layer `profiles.default` beats a project-layer flat fill per field — the CLI-side twin of the `agent.profiles`/`agent.tiers` cross-scope inversion, pre-existing and byte-identical under the former rung form. Documented in the `2.16.19-to-2.17.0` migration; retiring both twins needs a layer-aware fold.
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills

### Invocation Overrides Ride Structured Resolution and Bind the Native Arm Only
**Decision**: Invocation-time CLI/model/effort overrides are flags on `fab agent <stage> -o yaml`, the single resolution call every dispatch site makes. Because `fab dispatch start` carries no override surface — it re-resolves the stage from config — the overrides bind the **native Agent-tool arm** only, and every doc restating the surface carries that scope.
**Why**: Every dispatch site makes exactly one structured resolution call and branches on `dispatch:` key presence, so overriding at that call reuses the whole seam with zero new machinery; a separate override channel would need its own precedence rules and its own compliance-visibility contract. The YAML output is a pure projection of the profile, so overriding one field is unambiguous and useful. Scoping to the native arm keeps the docs describing what the code does instead of a workflow that errors on the headless arm and silently runs the wrong CLI on the pane arm.
**Rejected**: Plumbing the flags through `fab dispatch start` (a second resolution surface with its own precedence and visibility contract); per-stage override config keys (persistent state for a per-run intent); forbidding bare `--model` on the resolution surface for symmetry (a usage error for an unambiguous query).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

### The Built-in Tables Are an Embedded Data File, Parsed Through the Config Schema
**Decision**: The CLI grammars, claude's per-role fills, the depth knobs, and the `dispatch.*` built-in defaults live in `src/go/fab/defaults.yaml`, embedded with `//go:embed` and unmarshalled at package init into `config.Config` — the struct the user-config loader fills — with a malformed file panicking. `stageRoles` and `roleDepth` stay Go map literals, making the YAML/Go split the overridable/fixed boundary.
**Why**: The data is deeply nested (per-role fills per CLI), and as Go literals it is verbose to review and unreadable to a non-Go reader doing a model bump. A data file also removes the built-ins↔`fab config explain` drift class **by construction** rather than by drift-guard test: the reference renders from the same bytes the resolver parses, so there is no Go-map→YAML translation left to drift. Parsing through the config schema is the only reading that keeps the file's config-fragment shape from silently diverging from the schema it imitates, and it makes the eventual layer-0 unification a merge-order change rather than a parser change. Embedding rather than reading `$(fab kit-path)` at runtime keeps the "resolution cannot fail" property: kit and binary release atomically, so an on-disk read buys nothing and adds a binary↔kit skew failure mode. Panicking is the honest response to compiled-in bytes that do not parse — a defective build artifact, not a state a released binary can reach.
**Rejected**: Reading the file from the kit cache at runtime (binary↔kit version skew on a path that cannot fail today); a bespoke defaults struct in `internal/agent` (a second schema definition to keep in sync) or hand-walking a `map[string]any` (loses the schema's yaml tags and zero-value semantics); returning an error from the accessors (widens the API of a behavior-neutral change for an unreachable state) or falling back to hardcoded Go values (restores the duplicate literals); `sync.Once`-on-first-use (defers a build defect to an arbitrary call site — there is no I/O to make lazy); moving `stageRoles`/`roleDepth` into the file too (taxonomy is fab-owned policy, and the split is what signals which half a user may override); placing the file under `src/kit/` (that tree deploys to the kit cache and would imply runtime reading).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0; *Updated by*: 260809-wll4-config-source-consolidation

### `defaults.yaml` Lives at the Module Root; a One-File Root Package Owns the Embed
**Decision**: `defaults.yaml` sits at the Go module root (`src/go/fab/defaults.yaml`), where a one-file root package (`src/go/fab/defaults.go`, `package fab`) owns the `//go:embed` and exports the raw bytes as `DefaultsYAML []byte`. `internal/agent` keeps all parsing, validation, and init-injection, consuming the bytes via `fabroot.DefaultsYAML`.
**Why**: The file is the most-referred-to data file in the module — the one humans open to check or bump a model — and the module root is the highest placement `go:embed` permits (the directive cannot reach above the embedding package's directory). Splitting embed ownership from parsing keeps the embed-over-kit-cache single-source design and every drift guard untouched: the move changes no value, symbol, or resolution rule.
**Rejected**: Repo-root or `src/` placement (a compile error under `go:embed`'s package-directory rule, and a copy-at-build step would reintroduce the two-copies drift the embedded single source exists to kill); leaving the file beside its consumer with only doc pointers (does not fix the discoverability complaint, and the `internal/` path wrongly signals "implementation detail" for user-facing overridable data).
*Introduced by*: 260823-a3mu-hoist-defaults-module-root

### The Built-in Command Identifiers Are Package Vars Sourced from the Embedded File
**Decision**: `DefaultInteractiveCommand` and the non-claude `DefaultCodex*`/`DefaultAgy*`/`DefaultKimi*` identifiers are package `var`s reading the parsed CLI entries, not `const` literals; `spawn.DefaultSpawnCommand`, their only compile-time-constant consumer, is a `var` re-export of the same value. The identifiers are kept rather than replaced by `ResolveCLI(nil, name)` calls. The `config.DefaultDispatch*` symbols follow the same pattern — package-level vars carrying no literals, assigned from the parsed `dispatch:` block by agent's `init()` (the import cycle makes push-from-agent the only allowed direction — see [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Dispatch Defaults Are Init-Injected from `defaults.yaml`").
**Why**: `defaults.yaml` owns these strings — keeping them as Go literals would put every command text in two places and reintroduce by hand exactly the drift the data file removes by construction. Keeping the *names* leaves `internal/configref`, `internal/configupgrade`, and every test call site untouched.
**Rejected**: Keeping the constants canonical and having `defaults.yaml` restate them (drift-by-test, the thing being removed); a test asserting const↔YAML equality (the same drift, one indirection later); deleting the identifiers and rewriting every consumer (inflates a behavior-neutral diff).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0; *Updated by*: 260809-wll4-config-source-consolidation

### `DefaultProfile` Is Resolution Against a Nil Config
**Decision**: `agent.DefaultProfile(role)` is defined as `ResolveRole(nil, role)` rather than a lookup in a separate built-in role→profile table.
**Why**: There is no single built-in role→profile map to read any more — the values live under `clis.claude.profiles`, reached through the same chain everything else uses — and nil-config resolution is exactly "the built-in answer". It also keeps `fab config explain` sourcing its per-role defaults from the resolver rather than from a second table that could drift.
**Rejected**: A parallel built-in profile map (a second table to keep in sync with the resolver, which is the drift class the embedded data file exists to remove).
**Consequence — the registry row is the STATIC default; the read model composes a live one**: `configref.Fields()`'s `agent.profiles` default is built from `DefaultProfile`, which is knob-blind by construction. So `fab config show --origin` recomposes that one row through `configref.DefaultsMapFor`, resolving each role against the **live** config (`agent.ResolveRole`) twice, per leaf: the `cli` leaf against `knobsOnly` (the user's `agent.profiles`/`agent.tiers` entries stripped — the defaults tier reports the built-in a user override *shadows*, never an echo of that override), the `model`/`effort` leaves against `fillsOnly` (each per-role `cli` override kept, per-role model/effort overrides stripped — the built-in fill is a function of the CLI the role actually dispatches to, and an overridden CLI's own fills appear in no higher tier, so nothing echoes). The drill-down and `fab agent <stage|role> -o yaml` therefore agree both on which CLI a role would dispatch to and on the model/effort fills it would carry. The same two projections feed `show --origin --json`'s optional `derived_from`, which names the knob or fill leaf (via `agent.ResolveRoleWithSource`) that determined a default-tier row's value. When a fill holds the reserved `tool-default` value, the default-tier row shows `tool-default` (the value the holding key actually carries) rather than the collapsed empty model. See [_shared/configuration.md](/_shared/configuration.md) § Six-Verb Surface.
*Introduced by*: 260806-j9nh-agent-profiles-session-workers; *Updated by*: 260812-05wy-config-show-role-provider-fills

### Unknown CLI Is a Lookup Failure That Names the Resolvable Set
**Decision**: An unknown `--cli` name exits non-zero listing the available CLI names — the sorted union of fab-kit's built-in table and the project's `clis:` keys, exposed as `agent.CLINames(cfg)` over the nil-safe `config.CLINames()` accessor. The message is produced by a single `unknownCLIError(cfg, name)` helper in `cmd/fab`, shared verbatim by both flag-accepting commands (`fab agent`'s CLI-addressed mode and `fab resolve-agent --cli`).
**Why**: The union is exactly the set `ResolveCLI` will accept, so it is the only set whose listing is actionable; sorting makes the message stable for tests and for readers. Naming resolvable *names* is not validation of command *content* — resolved command strings still pass through verbatim, preserving document-don't-validate (fab never infers a CLI from a model string). One helper keeps the two commands' errors byte-identical, so a caller learns one phrasing.
**Rejected**: Listing only the project's configured CLIs (omits the built-in `claude`, which resolves fine); a bare "unknown CLI" error (leaves the caller to guess the config surface); validating the resolved command string (breaks the document-don't-validate contract); a per-command copy of the formatter (two phrasings of one contract to drift).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

### User-Facing Go Strings Name the Skill-Consumed Resolution Surface
**Decision**: User-facing Go strings that direct stage or role resolution name `fab agent <stage|role> -o yaml`; the native-dispatch error also states the `dispatch:`-key absence discriminator, and tests pin both the error guidance and the generated config-reference text.
**Why**: Executable guidance and generated reference prose are part of the same behavioral claim as the skills. Keeping them on the structured surface prevents the binary from contradicting the workflow it ships.
**Rejected**: Splitting user-facing resolution guidance between `fab agent` and the deprecated compatibility command, or treating executable strings as outside the documentation claim.
*Introduced by*: 260901-u6es-fab-agent-yaml-skill-migration

### The Kit Default Is the Opus Family, Never Fable
**Decision**: The built-in Claude fills put the Opus-tier roles (`default`, `review`) on the `opus` alias — the top coding tier at its best price/capability point — and the `doing`/`hydrate`/`operator`/`fast` roles on `sonnet`. Fable is never a kit default; a user who wants it dials a role through the config cascade (a system-tier `clis.claude.profiles.default.model` override is the established shape).
**Why**: The kit default is what every un-overridden project pays for on every dispatch, so it should sit at the best price/capability point of the tier rather than at the tier's ceiling. Fable is a separate family, so the `opus` alias never resolves to it.
**Rejected**: `claude-fable-5-1` / `fable` as the `default` fill (about 2.5x the price with additional safety gating — a per-user choice the cascade already supports, not a kit-wide one); raising `review` to `xhigh` (a separate judgment about the critic's cost/benefit, not part of a model choice).
*Introduced by*: 260923-vvdv-bump-claude-defaults-opus-5-5; *Updated by*: 261117-vyrt-claude-fills-opus-sonnet-aliases

### The Claude Fills Track One Model Generation Across All Six Roles
**Decision**: All six built-in Claude role fills sit on the same model generation — `default` and `review` on `opus`, `doing`, `hydrate`, `operator`, and `fast` on `sonnet`. Because the fills are family aliases, the installed Claude Code moves the whole table to a new generation at once; the family split between the two tiers is held fixed.
**Why**: The fills are the bottom tier of the cascade, so they are what an un-overridden project runs on every dispatch. Straddling two generations leaves capability on the table in whichever roles lag — `operator` is the highest-volume role in the kit and `fast` carries ship's PR-description comprehension — and a split table invites reading the lag as a deliberate tier choice when it is only an unfinished bump. Sonnet 5.5 is the same list price as Sonnet 5, so currency in the volume roles costs nothing.
**Rejected**: Re-tiering a role across families while bumping the generation (a cost/quality judgment, deliberately kept out of a model-ID bump so the two decisions stay separately reviewable — the `doing`/`hydrate` re-tier shipped as its own decision); changing the efforts alongside the models (`medium` is what the Sonnet roles' agentic/multi-step workload wants, independent of generation).
*Introduced by*: 260929-9dwn-bump-claude-defaults-sonnet-5-5; *Updated by*: 261117-vyrt-claude-fills-opus-sonnet-aliases

### Explicit `effort` on Every Claude Fill Is Load-Bearing, in Both Directions
**Decision**: Every row in the built-in Claude fills carries an explicit `effort`, and the fill block's comment in `src/go/fab/defaults.yaml` records why: fab never relies on a model's built-in effort default. Both claude command templates substitute `--effort {effort}`, so the fill value is the only thing that holds each role at its intended level — all the more because the alias fills can move to a model whose default differs.
**Why**: The two model families default in opposite directions from what the role rationale in `stage-models.md` § Default role profiles argues for. Opus 5.5 (what `opus` resolves to today) defaults to `medium`, one level below the `high` the Opus roles want, so dropping `effort: high` would silently step every un-overridden Opus dispatch down. Sonnet 5.5 defaults to `high`, one level above the `medium` the Sonnet roles want, so dropping `effort: medium` would silently step `operator` and `fast` up. `doing`/`hydrate`'s `effort: high` equals Sonnet 5.5's default and stays explicit by the same rule — the next Sonnet generation may default differently. Treating the effort column as redundant with the model's own default breaks in both directions.
**Rejected**: Leaving the effort column implicit on rows where it matched the model's default (the coupling to a model-side default is exactly the drift this pin prevents, and the direction of that drift changes with each model generation); documenting the constraint only in the spec (the value's own file is where the next editor looks first — the spec sentence and the YAML comment carry it together, with the YAML beside the value).
*Introduced by*: 260923-vvdv-bump-claude-defaults-opus-5-5

### `doing` and `hydrate` Run Sonnet; `review` Is the Downstream Opus Critic
**Decision**: The built-in Claude `doing` and `hydrate` fills are `sonnet`/`high`; `default` (intake, interactive sessions) and `review` stay `opus`/`high`, so review is the only Opus stage downstream of intake. The kit default accepts the apply↔review coupling risk: a Sonnet author under an Opus critic may raise must-fix findings and rework cycles (capped by the project's rework budget). The size of that effect is unmeasured; a project that sees elevated rework sets `agent.profiles.doing` back to an Opus model.
**Why**: `doing` (apply) and `hydrate` are the pipeline's highest-output-volume stages and dominate per-change spend. Hydrate writes memory from a finished, reviewed diff — summarization well within Sonnet's range. The owner chose a kit-wide default so every un-overridden project gets the cheaper split; the role separation keeps disagreement a one-row override rather than a taxonomy change.
**Rejected**: A machine-local `--system` override instead of a kit default (the owner wanted it for every project); re-tiering `hydrate` only and keeping an Opus author (argued on the coupling — a strong author minimizes rework a sharp critic would trigger — and overruled by the owner as an unmeasured cost); dropping the explicit `effort: high` now that it equals Sonnet 5.5's default (fab never relies on a model's own default).
*Introduced by*: 261030-lc7k-retier-doing-hydrate-sonnet

### Claude Fills Are Family Aliases
**Decision**: The six built-in Claude fills are the family aliases `opus` and `sonnet`, not versioned IDs. Codex's and agy's fills stay concrete IDs. Resolution sets `model_alias` both when a versioned Claude ID maps to its family and when the model already is one of the four Agent-tool aliases (`agent.IsModelAlias`, derived from the same `modelAliasPrefixes` table as `agent.ModelAlias`, which keeps passing an alias through verbatim). Composed CLI commands carry the resolved model verbatim, so a built-in Claude dispatch runs `claude --model opus`. A versioned ID remains the way to pin a specific model.
**Why**: Versioned fills forced a kit release, a fence re-render, and a doc sweep for every Claude model launch — the CLIs advert was re-rolled four times for Claude fill changes alone — and left un-upgraded kits on older models the installed tool already superseded. `claude --model` and the Agent tool's `model` parameter both take the aliases and resolve the newest model of the family, so the defaults track the tool. Without the `IsModelAlias` check, a bare alias would resolve an empty `model_alias` and the native rung would silently inherit the session model.
**Rejected**: Bumping versioned IDs per release (the churn this removes); aliases for codex/agy (their model flags take a concrete catalog slug and have no "latest" alias); changing `ModelAlias` to return `""` for an unmapped input (it is also the `--alias` line adapter, which must pass non-Claude models through); validating model strings (fab is verbatim pass-through).
*Introduced by*: 261117-vyrt-claude-fills-opus-sonnet-aliases

### A Model Without a Native Alias Descends Instead of Erroring or Warning
**Decision**: A non-empty model with no Agent-tool alias removes the native rung; the ladder descends to the next supported rung (headless for claude). An empty model keeps native and inherits the session model, and the reserved `tool-default` value — collapsed to empty before the predicate runs — keeps native too. The rule is one predicate (`agent.HasNativeModel`) shared by every rung-selection site.
**Why**: The Agent tool's `model` parameter accepts only `opus`/`sonnet`/`haiku`/`fable`. Before this rule a role pinned to `glm-latest[1m]` resolved native with an empty `model_alias`, the Agent call omitted `model`, and the worker silently ran on the session model — while `fab dispatch start` refused the same stage with native-dispatch guidance, leaving no path to the pinned model. `claude -p --model <custom>` honours any configured model string, and descent reuses the ceiling ladder every dispatch site already handles (`dispatch:` present ⇒ CLI adapter).
**Rejected**: Erroring (blocks a stage for a config the user wrote deliberately); warning (leaves the silent model substitution in place); aliasing custom models (no faithful mapping).
*Introduced by*: 261117-3ms2-native-rung-descends-without-alias

### Native Availability Is a Reason-Carrying Value and `dispatch.reason` Is General
**Decision**: `dispatch.SelectMode` takes a caller-computed `NativeAvailability` (the `TmuxAvailability` idiom) instead of a bool; the no-capability skip text stays `native unavailable` and the alias skip text is `native unavailable: model has no native alias`. `dispatch.reason` exposes the skip list for any descent (a codex role under the default `native` preference also shows `native unavailable`), and `fab resolve-agent`'s frozen line protocol gains no line.
**Why**: Keeps the selector pure while letting the reason name the real cause; one uniform field explains every descent and matches `dispatch start`'s stderr line.
**Rejected**: A second bool parameter (no place for the reason text); an alias-only reason field (ad hoc); a new `resolve-agent` line (breaks the byte-stable contract).
*Introduced by*: 261117-3ms2-native-rung-descends-without-alias

### `tool-default` Is a Reserved Real Value That Collapses After Precedence
**Decision**: The reserved model value `tool-default` (`agent.ToolDefaultModel`) is an ordinary non-empty string through the whole model precedence chain and is collapsed to an empty model once, afterwards, by `agent.EffectiveModel` in `ResolveRoleWithSource` (and in the CLI-only branch of `composeAgentResolution`). `source.model` keeps the key that held the token. `fab agent models` lists a `source: reserved` `tool-default` row first for every tool.
**Why**: A concrete-slug tool (codex, agy) could never reach the empty rung — `unset` lands on the kit fill and `set ""` is refused — so following the tool's own default meant pinning a slug that goes stale on the next release. Because the token is a real value it beats lower tiers, which an empty leaf deliberately does not; collapsing after precedence reuses the existing empty-model token-drop and native inherit path with no new placeholder grammar, and it makes the token exempt from the native-rung descent without touching the rung selector. `tool-default` was chosen over `default` (collides with Claude Code's own `default` model setting), `auto` (collides with custom models such as `auto[1m]`), and `@default` (needs YAML quoting); it matches the "tool default (…)" UI wording.
**Rejected**: Treating empty as a real value (inverts the empty-skip cascade for every tier); a per-tool "default" sentinel (CLI-specific logic in a CLI-neutral resolver); collapsing inside `spawn.WithProfile` (native and `model_alias` would still see the token); clearing `source.model` on collapse (loses which key holds the opt-out); a model-and-effort token (efforts already accept empty and have no concrete-slug problem).
*Introduced by*: 261118-djnz-tool-default-model-token

### fab Owns Model Discovery and It Fails Open
**Decision**: One verb, `fab agent models`, asks each installed tool for its models through per-tool adapters (or a user's `models_command`); every failure degrades to the kit fills plus a stderr warning with exit 0, and there is no cache. Callers — hexokit's settings page, the operator, later `fab config explain` — consume this verb and never read tool configs themselves.
**Why**: The kit's fills lag the tools and never show models the user added (custom `modelPicker` entries, kimi's model tables), so a list built from fills alone sends users to hand-pin stale strings. One implementation behind one verb serves every caller. Discovery reads unstable surfaces (`codex debug models` is a debug command), so it must never block a caller, and a settings page must render even when a tool is missing. A `codex debug models` call measured ~0.16 s, so a cache buys nothing in v1.
**Rejected**: hexokit reading `~/.codex`, `~/.claude`, `~/.kimi-code` itself (re-implementation drift — the agy inherit-label bug); caching in v1; failing the command when discovery fails (a hard dependency on third-party tool health); folding discovery into resolution (validation, which fab's verbatim pass-through contract refuses).
*Introduced by*: 261117-rhpz-agent-models-discovery

### Minimal TOML Line Scanner Instead of a TOML Dependency
**Decision**: Codex's `model` and kimi's model tables are read by a small in-package scanner that tracks `[table]` headers and reads single-line string and string-array values.
**Why**: Only four keys are needed, and `go.mod` stays at cobra/pflag/yaml.v3. Unreadable values outside the consumed keys are skipped, so an exotic line elsewhere in a large user file does not discard the whole read.
**Rejected**: A TOML module — more robust to exotic syntax, but a new dependency for four keys; it can be swapped in later behind the same function.
*Introduced by*: 261117-rhpz-agent-models-discovery

## Consumers

The CLI/role resolution feeds three runtime consumers:

- **The dispatch seam** (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`, and `/fab-continue`'s one-stage sequencer) calls `fab agent <stage> -o yaml` before each post-intake stage's sub-agent and **branches on `dispatch:` key presence**: absent ⇒ native Agent-tool dispatch (model via `model_alias` on the Agent `model` param, effort via the YAML `effort` value in a prompt instruction); present ⇒ the CLI adapter `fab dispatch` (the profile rides `dispatch.command`, which skills never execute themselves). The native seam therefore never sees a non-empty alias-less model. Every stage it resolves is a Tier-2 role, so `agent.workers` is the knob it consults. The ship (`fast`) and review-pr (`doing`) sites — `/fab-fff` Step 4 and `/fab-continue`'s ship/review-pr rows — branch on the same rule, so a resolved pane/headless mapping is executed there rather than silently degrading to the inherited session model; their delegated `/git-pr` / `/git-pr-review` workers self-manage their own stage's transitions on every arm. See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership.
- **The operator launch** is HexoKit's (`rk operator`): rk resolves the **operator** role through `fab agent operator -o yaml` / `--print` and composes its session command from that role's CLI `interactive_command` + profile. fab has no operator launcher of its own (see [operator.md](/runtime/operator.md)).
- **Batch worker spawns** (`fab batch new`/`switch`) compose from the **default**-role CLI `interactive_command` + profile — so workers spawn WITH a profile. As interactive spawns, they carry the same wrap and inside-the-wrap shell fallback, so the tab survives the agent's exit as a shell in the same cwd (see [agent-primitives.md](/runtime/agent-primitives.md) § Spawn composition). See [distribution/kit-architecture.md](/distribution/kit-architecture.md).

The latter two are Tier-1 roles, so `agent.session` is what governs them — and it binds at **launch**: a running session keeps the CLI it started on.

[`fab dispatch`](/runtime/dispatch.md) runs the command selected by the current ladder; `fab agent -o yaml` resolves and emits the structured selection. `fab dispatch start` re-resolves the ladder so pane reachability is current at launch time.
