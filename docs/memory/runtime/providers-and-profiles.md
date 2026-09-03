---
type: memory
description: "Provider mechanics and role policy — session/workers depth knobs, environment and launch overrides, four pane-capable built-ins with independent interactive/native/headless grammar and per-role fills, six roles, precedence, stage mapping, `fab agent` YAML resolution, deprecated `fab resolve-agent` compatibility, dispatch-mode descent, and consumers."
---
# Providers & Agent Profiles

**Domain**: runtime

## Overview

Agent config splits **provider mechanics** (how to invoke an agent) from **role policy** (which model + effort a given job runs at). Providers live in a top-level `providers:` table carrying command grammar plus a per-role fill map; the six **roles** are fab-owned slot names, and a **profile** is the concrete `{provider, model, effort}` a role resolves to. The advertised user surface is two knobs — `agent.session` and `agent.workers` — which pick a provider by agent **depth**.

**Vocabulary** (used consistently across the config, the Go packages, and the docs):

| Term | Meaning |
|------|---------|
| **role** | one of six fixed slot names — `default`, `operator`, `doing`, `review`, `hydrate`, `fast` |
| **profile** | a concrete `{provider, model, effort}` value a role resolves to |
| **provider** | independent pane (`interactive_command`), native (`native`), and headless (`headless_command`) capabilities plus per-role fills |
| **Tier 1 / Tier 2** | agent **depth** — the agents a user talks to vs. the agents pipeline stages dispatch to |

Tier 1/Tier 2 name depth and nothing else. A pane worker is still Tier 2: the defining property is "owes a result artifact and owns no transitions" ([dispatch.md](/runtime/dispatch.md)) — ship/review-pr workers are the single carve-out, self-managing their own stage's transitions — not "never spoken to".

This file is the model — the depth knobs, the four built-in providers, the six roles and the fixed partition, the fill precedence, the stage→role mapping, the structured `fab agent -o yaml` surface and deprecated `fab resolve-agent` compatibility projection, and who consumes the resolution. The **config-schema authority** is [_shared/configuration.md](/_shared/configuration.md) § `providers` and § `agent`; the **dispatch-seam wiring** is [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution; the pre-implementation design intent is `docs/specs/stage-models.md` (drift-guarded against the Go maps).

## Requirements

### Requirement: Two depth knobs — `agent.session` and `agent.workers`

`fab/project/config.yaml` SHALL support two scalar keys naming a provider, and they SHALL be the whole advertised agent surface:

```yaml
agent:
  session: claude    # Tier 1 — what you talk to: fab agent, fab operator, fab batch
  workers: agy       # Tier 2 — what pipeline stages dispatch to
```

Both default to `claude`, so a fresh install writes nothing and resolves fab-kit's shipped profiles. Both are `scope: both`, so the choice is settable once machine-wide in `~/.fab-kit/config.yaml`.

The **role→depth partition is fixed and fab-owned** (`roleDepth` in `internal/agent`, read externally through the exported `agent.IsSessionRole`):

| Depth | Roles | Provider from | Applies at |
|-------|-------|---------------|------------|
| **Tier 1** (session) | `default`, `operator` | `agent.session` | **launch** time — fab cannot switch a running session's provider |
| **Tier 2** (workers) | `doing`, `review`, `hydrate`, `fast` | `agent.workers` | **every stage dispatch** (`fab agent <stage> -o yaml`) |

The split is mechanically real rather than cosmetic, which is what makes depth the right knob axis. `intake` rides `default` and is therefore a session role for exactly that reason — it runs foreground in the user's own session.

A knob supplies only the **provider** rung. Model and effort always come from the resolved provider's own per-role fills, so naming a provider is a complete configuration.

#### Scenario: one knob re-points every dispatched stage

- **GIVEN** `agent: { workers: agy }` and no other agent keys
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `provider: agy` (apply → `doing`, a workers role)
- **AND** `fab agent operator -o yaml` still emits `provider: claude` (a session role)

#### Scenario: no `agent:` block at all

- **GIVEN** a config with no `agent:` block
- **WHEN** any role or stage is resolved
- **THEN** the provider is `claude` for every role, on fab-kit's shipped per-role models

### Requirement: Per-session provider selection — environment and launch flags

`FAB_AGENT_SESSION` and `FAB_AGENT_WORKERS` SHALL provide process-tree-local overrides for the two depth knobs. The current process and its descendants resolve these variables above project and system config, so two shell sessions in separate worktrees can select different worker providers without changing committed configuration. The variables are instances of the generic registry-derived environment mechanism documented in [_shared/configuration.md](/_shared/configuration.md) § Override Cascade & Scope Enforcement; both are honored because their registry rows have `scope: both`.

- **`FAB_AGENT_SESSION=<provider>`** selects the Tier-1 provider for commands launched by that process tree.
- **`FAB_AGENT_WORKERS=<provider>`** selects the Tier-2 provider projected by `fab agent <stage> -o yaml` and used by `fab dispatch start` when it re-resolves a stage.
- The values are YAML-parsed config overrides, not provider-validation surfaces; provider names remain opaque and an unknown name fails at the existing resolution lookup.
- An empty variable behaves as unset — whether blank or spelled as an empty YAML value (`null`, `""`), per the cascade's one emptiness rule. The override is never persisted in `config.yaml`, a change artifact, or runtime state.

The session-spawning commands SHALL expose `--workers <provider>` as sugar for exporting `FAB_AGENT_WORKERS` into the spawned process tree:

- `fab agent --workers <provider>` appends the exact assignment to the exec environment in both role-addressed and provider-addressed modes. It does not alter Tier-1 session-command resolution, and `--print` continues to print only that resolved command.
- `fab batch new --workers <provider>`, `fab batch switch --workers <provider>`, and `fab operator --workers <provider>` prefix a shell-quoted assignment (`FAB_AGENT_WORKERS='<provider>'`) to a newly launched tmux shell command. Embedded quotes and shell metacharacters remain data.
- An existing operator window is selected without relaunch, regardless of `--workers`.
- The flag value passes through without validation. There is no `--session` flag; callers use `FAB_AGENT_SESSION` directly.

These config variables are unrelated to the fab-kit binary's `FAB_AGENTS` skills-list override. The user-facing provider-selection variables are `FAB_AGENT_SESSION` and `FAB_AGENT_WORKERS`; the environment mechanism itself applies generically to every `both`/`system` registry row. (2d1w)

#### Scenario: two sessions select different worker providers

- **GIVEN** two process trees rooted in separate worktree shells
- **WHEN** one launches with `FAB_AGENT_WORKERS=kimi3` and the other with `FAB_AGENT_WORKERS=codex`
- **THEN** every Tier-2 stage in each tree resolves its own selected provider
- **AND** neither worktree's `fab/project/config.yaml` changes

#### Scenario: `--workers` is launch-only sugar

- **GIVEN** `fab agent --workers codex` or a tmux launcher with the same flag
- **WHEN** the new session starts
- **THEN** its descendants inherit `FAB_AGENT_WORKERS=codex`
- **AND** the launching command's Tier-1 provider/profile resolution and persisted state are unchanged

### Requirement: Providers table — pure capabilities plus per-role fills

`fab/project/config.yaml` SHALL support a top-level `providers:` map keyed by **opaque, user-chosen provider names**. Each provider MAY independently carry:

- **`interactive_command`** — **pure launch grammar**: how to open an interactive agent session, and nothing more. It makes the provider pane-capable when tmux is available. Prompt delivery is fab's, not the command's — no code path appends a prompt argument to it ([dispatch.md](/runtime/dispatch.md) § Prompt delivery), so pane capability does not depend on whether the CLI accepts a positional initial prompt. **Exec contract**: the command MUST exec its binary, so the binary — not a wrapper shell — owns the pane's foreground. The readiness gate's takeover precondition requires exactly that (a shell foreground reports `booting` and is never typed into — [dispatch.md](/runtime/dispatch.md) § `fab dispatch ready`), so a wrapper that keeps a shell in the foreground is unsupported by design and fails **observably**: every probe reports `booting` with the shell prompt in the snippet until the wiring's consecutive-booting allowance is spent and the run escalates — never silently.
- **`native`** — declares native Agent-tool capability. Provider names are opaque, so fab never infers it.
- **`headless_command`** — runs one headless stage task via `fab dispatch`; the prompt is supplied on stdin.

Each provider MAY additionally carry **`profiles`** — a map keyed by role name, each value `{model, effort}`: *"when this provider plays this role, use this model/effort."* It supplies the command's `{model}`/`{effort}` placeholders (§ Fill precedence). A provider's **`profiles.default` doubles as its cross-role fallback** — a role absent from the map resolves the `default` entry, then empty. The map merges per **role** and then per **field** over the built-in table, exactly as the command fields do, and `providers` is `scope: both`, so a fill is settable once per machine.

Capability fields are never merged or substituted for one another. Their presence says **how** a provider can run, never **which** mode is selected. `dispatch.mode` supplies that preference and the shared selector descends pane → native → headless, starting at the configured rung and never ascending.

Every supported agent CLI has an interactive mode, so built-in providers SHALL ship `interactive_command` launch grammar and rely on the generic readiness gate for boot and first-run walls. Headless operation varies by CLI and its `headless_command` grammar MUST be probed independently. A user-defined provider MAY omit `interactive_command`; automatic selection skips the pane rung, and an explicit pane launch reports the existing configuration-key hint.

**fab-kit ships FOUR built-in providers — `claude`, `codex`, `agy`, and `kimi`** (`defaultProviders` in `internal/agent`, parsed from the embedded `defaults.yaml` — § The built-in defaults are an embedded `defaults.yaml`; each command string is additionally exposed as a package var, `DefaultInteractiveCommand` / `DefaultCodex*` / `DefaultAgy*` / `DefaultKimi*`, that `internal/configref` interpolates, so no literal is duplicated):

| Built-in | `interactive_command` | `native` | `headless_command` | `profiles` |
|----------|-------------------|----------|--------------------|------------|
| `claude` | templated default | `true` | `claude -p --permission-mode bypassPermissions --model {model} --effort {effort}` | all six roles |
| `codex` | `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` | absent | `codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` | sparse — `default`, `doing`, `review`, `fast` |
| `agy` | `agy --dangerously-skip-permissions --model {model}` | absent | `sh -c 'agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'` | sparse — `default`, `fast` (model only) |
| `kimi` | `kimi --auto -m {model}` | absent | `sh -c 'kimi -m {model} -p "$(cat)"'` | **none** — deliberate |

**claude, codex and agy ship per-role fills**, so naming one of them on a depth knob is a complete configuration: `agent.workers: codex` resolves a model suited to each role, and **role differentiation survives the provider swap** (apply/review run at codex's higher effort while ship takes its cheaper `fast` model). Their non-claude maps are **sparse** — a role absent from a map resolves that provider's `default` entry — and the merge is per **field**, so a codex row carrying `effort` only takes its model from `default`.

Two per-provider fill shapes are load-bearing:

- **agy's fills are model-only** — that CLI's model IDs *embed* the reasoning level as a suffix (`gemini-3.1-pro-high`), so the effort rides the ID and the grammar carries no `{effort}` placeholder for one to land in. The IDs are concrete catalog entries from `agy models`, so they are rows to bump when that catalog moves.
- **kimi ships NO fills at all**, and empty is the correct built-in there rather than a gap: its `-m` takes a **user-config model alias** (managed installs expose `kimi-code`/`k3`; custom providers differ), so any pinned value would break non-managed installs. The empty `{model}` drops the `-m` pair through `spawn.WithProfile`'s token-drop and kimi falls back to the user's own `default_model`. A user wanting role differentiation pins `providers.kimi.profiles.<role>.model` themselves.

Non-claude fills are **refreshed at kit-release cadence** by editing `defaults.yaml`, are **unvalidated pass-through** values like every other resolved string, and a stale one fails loudly at the provider CLI and is corrected by **one config line** — `providers.<name>.profiles.<role>.model`, settable once per machine because `providers` is `scope: both`. There is no staleness automation: no CI check against provider APIs, no model-catalog fetch (§ Design Decisions → "Built-in Providers Ship Grammar AND Per-Role Fills"; `docs/specs/stage-models.md` § Refreshing the non-claude fills carries the policy and its decision lineage). **Current values are not restated here** — run `fab config explain` (renders live from the embedded defaults) or read the drift-guarded sample in [stage-models.md](../../specs/stage-models.md) § Built-in providers.

A project's `providers:` block per-field-merges over the built-ins via `agent.ResolveProvider(name)`, including the profiles map.

**Naming a built-in is sufficient — no `providers:` block required.** A depth knob, an `agent.profiles` entry, or an invocation flag that names `codex`/`agy`/`kimi` resolves with zero config, and the configured dispatch preference then applies to that provider's capabilities — none of the three declares `native`, so their stages descend to headless. Adding or overriding a command never changes mode policy by itself. A built-in provider is otherwise **inert**: adding a row changes no default behavior, which is why these rows live in Go while behavior-changing config still ships commented (see § Design Decisions → "Built-in Provider Grammar in Go, Fill Values in Config").

```yaml
providers:
  codex:
    profiles:                 # override one shipped fill; the rest stay built-in
      review: { model: <codex-model-id>, effort: xhigh }
    # commands inherit the built-in grammar; override either to change it
```

The two command fields are deliberately unmerged because session and dispatch are **different invocations of the same binary** (claude interactive `-n <name>` vs headless `-p`; codex TUI vs `codex exec`) — no single template expresses both. `{model}`/`{effort}` placeholders in either command are substituted at resolve time via `spawn.WithProfile` (reused, not reimplemented — see [configuration.md](/_shared/configuration.md) § `providers` for the template/append modes and the empty-value token-drop rule).

The non-claude built-ins run **full-auto**, because a stage worker is unattended and has no approval-answering channel: codex carries `--dangerously-bypass-approvals-and-sandbox` on both forms, agy carries `--dangerously-skip-permissions` on both forms, and kimi carries `--auto` on its **interactive** command alone — its dispatch form takes **no approval flag at all**, because `kimi -p` is already non-interactive, auto-approves tool calls, and *rejects* `--yolo`/`--auto`. kimi is the one built-in whose full-auto posture is asymmetric across its two commands, and the asymmetry is the CLI's, not a choice.

**Prompt delivery differs per CLI, and the grammar absorbs the difference.** `fab dispatch` always pipes the stage prompt to the command's **stdin**. codex reads stdin directly via its `exec` subcommand, as does claude's headless command through `-p`. agy and kimi do not — their `-p` takes the prompt as an **argument** and ignores stdin — so both dispatch commands nest a shell: `sh -c '<cli> … -p "$(cat)"'`. The nesting is load-bearing, not stylistic (§ Design Decisions → "Nested-Shell `$(cat)` Idiom"). agy's command additionally raises `--print-timeout` to `120m`, since that CLI's 5-minute default would kill a long stage worker.

**agy is pane-capable.** Its built-in `interactive_command` is `agy --dangerously-skip-permissions --model {model}`, so sessions and pane dispatch work without a provider override. A fresh workspace can park at agy's interactive trust prompt even under the bypass flag; the readiness gate classifies that as an ordinary judgment round, after which delivery proceeds. The answer is remembered for the exact workspace path. An operator may pre-seed that exact path under `trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json`; fab neither guesses paths nor writes the provider's trust store.

**kimi is pane-capable**, shipping `kimi --auto -m {model}` alongside its headless command, because both halves of that same question were answered against kimi 0.34.0 (2026-08-10):

- **First run**: kimi gates a fresh folder behind a `Trust this folder?` wall, which is an ordinary readiness-gate **judgment round** — the pane reads `parked`, one Enter clears it, and the answer is remembered per folder, so it amortizes across every later pane worker in that checkout ([dispatch.md](/runtime/dispatch.md) § `fab dispatch ready`). It needs no code.
- **Input echo**: kimi draws vertical rules down both sides of its input box, so a hard-wrapped pointer arrives with `││` interleaved between the halves. `deliver`'s echo verification counts occurrences with box-drawing runes dropped alongside whitespace, so the pointer verifies ([dispatch.md](/runtime/dispatch.md) § `fab dispatch deliver`).

kimi has no interactive-initial-prompt flag — its `-p` is the non-interactive form, and a bare positional parses as a subcommand — which costs it nothing: delivery is post-open in every case, so pane capability never depended on that flag.

`fab config explain` presents all four as **live, uniformly-rendered built-in defaults** — every provider line carries exactly one leading `#` prefix in a fence or `init --system` scaffold (no doubled `# #` marker; an inline `# ...` note on a command line stays content), and the segment prose warns that hoisting a whole block PINS its fills against kit-release refreshes; see [configuration.md](/_shared/configuration.md) § `providers` for the rendered presentation and its parse-side guarantees, and § The machinery is documented, not scaffolded for why the managed fence omits the block.

**Provider names are opaque — fab NEVER infers a provider from a model string.** Two entries fronting the same vendor under different names are different providers, each with its own fills.

#### Scenario: the preference selects among independent capabilities

- **GIVEN** built-in claude and `dispatch.mode: pane`
- **WHEN** tmux and its `interactive_command` are available
- **THEN** `fab agent <stage> -o yaml` includes `dispatch: { rung: pane, command: … }`
- **WHEN** tmux is unavailable
- **THEN** the selector descends to claude's `native: true` and omits the `dispatch:` key
- **AND** `dispatch.mode: headless` starts at headless and emits claude's substituted `headless_command`, even though native capability also exists

#### Scenario: `kimi` resolves an empty model and drops the flag pair

- **GIVEN** `agent.workers: kimi` and no `providers.kimi` block
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `provider: kimi`, `model: ""`, and `dispatch.command: sh -c 'kimi -p "$(cat)"'` — the `-m {model}` pair dropped as a unit, the quoted `"$(cat)"` segment left syntactically intact

#### Scenario: `kimi` is eligible for a pane worker and a session

- **GIVEN** `agent.workers: kimi`, `dispatch.mode: pane`, a reachable tmux server, and no `providers.kimi` block
- **WHEN** `fab dispatch open` runs for a stage
- **THEN** the pane opens on `kimi --auto` — the empty `{model}` dropping the `-m` pair — and the readiness gate answers the first-run trust wall before `deliver` types the pointer
- **AND** `fab agent --provider kimi --print` prints that same composed command instead of erroring on a missing `interactive_command`

#### Scenario: the agy built-in is eligible for a pane worker and a session

- **GIVEN** `agent.workers: agy` and a reachable tmux server
- **WHEN** `fab dispatch open` runs for a stage under pane preference
- **THEN** it opens the exact composed `agy --dangerously-skip-permissions --model <resolved-model>` command
- **AND** the readiness gate reports any first-run trust wall for an ordinary judgment round before `deliver` types the pointer
- **AND** `fab agent --provider agy --print` composes the model-free `agy --dangerously-skip-permissions` command instead of reporting a missing capability

#### Scenario: naming a built-in provider with no `providers:` block

- **GIVEN** a config with `agent: { profiles: { review: { provider: codex, model: <codex-model-id>, effort: high } } }` and no `providers:` block at all
- **WHEN** `fab agent review -o yaml` runs
- **THEN** it emits `provider: codex` plus the `dispatch:` mapping selected by the current `dispatch.mode`; with the default `native` preference, codex descends to its headless capability
- **AND** the same default preference resolves built-in claude natively because claude declares `native: true`

### Requirement: Six roles with fixed referents

The role names are the six fixed slots — `default`, `operator`, `doing`, `review`, `hydrate`, `fast`. A role is **stage-named only where it maps 1:1 to a single referent** (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is **multi-referent** — `fast` governs the ship stage AND the `/fab-proceed` prefix-step dispatches (`/fab-switch`, `/git-branch`), and `default` governs intake, `fab batch`/`fab agent`, and the `/fab-proceed` create-intake dispatch. There is no `thinking` role: with `review` its own role, `thinking`'s only remaining stage would be intake, which never dispatches.

| Role | Referents | Depth |
|------|-----------|-------|
| `default` | intake (advisory, foreground); `fab batch` worker sessions; `fab agent` with no role; the `/fab-proceed` create-intake dispatch | Tier 1 |
| `operator` | the operator coordinator session (`fab operator`) | Tier 1 |
| `doing` | `apply`, `review-pr` — execution that must not err | Tier 2 |
| `review` | `review` — the critic (its own role so its model/effort dial independently of the author's) | Tier 2 |
| `hydrate` | `hydrate` — memory writing (its own role so it runs on a different model/effort than apply) | Tier 2 |
| `fast` | `ship` — near-mechanical work — plus the `/fab-proceed` prefix steps | Tier 2 |

**Current profiles**: run `fab config explain` (renders live from the embedded defaults) or see the drift-guarded table in [stage-models.md](../../specs/stage-models.md) § Default role profiles. Not restated here — this table owns the *roles*, which change far less often than the models.

`agent.profiles` is the **sparse per-role escape hatch** beneath the knobs: `agent.profiles.<role>.{provider, model, effort}`, every field optional, each set field beating the knob (provider) or the provider's fill (model/effort). It carries **no cross-role inheritance** — `agent.profiles.default` is the `default` role's own override, not a fallback source for the other five. Model IDs are written **versioned** (e.g. `claude-sonnet-5`); bare family IDs (`claude-sonnet`) fail both dispatch seams.

#### Scenario: a per-role override dials one role only

- **GIVEN** `agent.profiles: { default: { model: X }, doing: { effort: high } }`
- **WHEN** `fab agent apply -o yaml` (role `doing`) runs
- **THEN** `effort: high`, and the model comes from the resolved provider's `doing` fill — **not** from `agent.profiles.default.model`

### Requirement: The built-in defaults are an embedded `defaults.yaml`

The built-in tables — the depth knobs, the four-provider table with every provider's per-role fills, the `dispatch:` block carrying the three built-in dispatch defaults (`mode`/`column_width`/`reap_done`), and the `autopilot:` block carrying the built-in merge-mode default (`merge_mode`) — are **data**, and live in `src/go/fab/defaults.yaml`, shaped exactly as a **user config-file fragment**: the same `agent:` / `providers:` / `dispatch:` / `autopilot:` keys a project writes to override them. The file is the **single value source for every built-in default** — no Go source carries an independent copy; the `config.DefaultDispatch*` and `config.DefaultAutopilotMergeMode` symbols are package-level vars the package's own `init()` assigns from the parsed `dispatch:`/`autopilot:` blocks (the import cycle forbids `internal/config` reading them back — see [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Dispatch Defaults Are Init-Injected from `defaults.yaml`"). The file is compiled into the binary via `//go:embed` and unmarshalled **once at package initialization** into `config.Config` — the same struct `config.LoadPath` fills from a user's `config.yaml` — so the built-in shape and the config schema cannot diverge. A malformed file **panics at init** (compiled-in bytes make a parse failure a defective build artifact, not a runtime condition), and nothing is read from the kit cache at runtime, so resolution cannot break on a missing or corrupt cache. `defaults_test.go` is the file's safety net against a YAML typo: it asserts the parse, exhaustive role/provider coverage with non-empty fields, and both command fields for every built-in. The agy and kimi interactive values are pinned **by value** — `agy --dangerously-skip-permissions --model {model}` and `kimi --auto -m {model}` — so a plausible edit cannot reach pane workers unnoticed. It also asserts that every **fill-carrying** provider has a non-empty `profiles.default.model` while kimi's empty fill map is deliberate rather than skipped, that agy's fills set no `effort`, that no built-in uses the deprecated flat fill spelling, that the `dispatch:` block pins the three dispatch defaults and their injection wiring, and that the file defines no keys outside the `agent:`/`providers:`/`dispatch:`/`autopilot:` surface.

**The YAML/Go split is the overridable/fixed boundary.** What lives in `defaults.yaml` is user-overridable by writing the same key in `fab/project/config.yaml` (or `~/.fab-kit/config.yaml`); what stays in Go is fab-owned policy — `stageRoles` (the fixed stage→role mapping) and `roleDepth` (the role→depth partition). The config-fragment shape is also what makes `defaults.yaml` the **physical source of the cascade's defaults tier** (see [_shared/configuration.md](/_shared/configuration.md) § Override Cascade): consumers still reach those defaults at the existing point-of-use seams rather than through `LoadPath`'s merge, while the read-model surfaces merge them as a real tier 0 through the registry projection `configref.DefaultsMap` — so folding the file into the loader's own merge would be a merge-order change, not a parser change.

**A model bump edits `defaults.yaml`** and runs the tests. The file carries the bump procedure inline and names the drift guards that will name *themselves* if a mirror falls behind — `TestDefaultsFileIsWellFormed` (the file itself), `TestDefaultRoleProfilesArePinned` (the deliberate-change pin for claude) and `TestNonClaudeProviderFillsArePinned` (the same pin for codex/agy, plus kimi's deliberate empty map), `TestDocTablesMatchAgentMaps` and `TestMirrorDocsMatchDefaultProfiles` (`docs/specs/stage-models.md`'s table and its inline-YAML sample, which is checked per **enclosing provider** so all four providers' blocks are guarded and agy's effort-less rows still match), `TestConfigReferenceDocumentsProviderFill` (the codex/agy fill lines in the rendered `fab config explain`), `TestCLIFabReferenceListsDefaultRoles` (`_cli-fab.md`'s enumeration). Every other consumer derives from `DefaultProfile()` or `ResolveProvider`, so a bump touches nothing else.

#### Scenario: the defaults travel inside the binary

- **GIVEN** a `fab` binary run from a directory with no fab-kit cache present
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** it resolves the built-in profile from the embedded bytes — no filesystem read of the kit cache occurs, and no on-disk state can make resolution fail

### Requirement: Fill precedence — one chain, provider-anchored

Resolution SHALL follow one precedence chain, implemented **once** in `internal/agent` (`ResolveRoleWith`, which `cmd/fab` never re-implements):

**Provider**: invocation `--provider` → `agent.profiles.<role>.provider` → the role's depth knob (`agent.session` or `agent.workers`) → built-in `claude`.

**Model / effort, per field**: invocation flag → `agent.profiles.<role>.<field>` → `providers.<p>.profiles.<role>.<field>` → `providers.<p>.profiles.default.<field>` → empty.

`<p>` is the **resolved** provider (post-override), which is what makes a provider swap safe: model and effort are re-derived from the new provider's own fills rather than carried from the old one, so **nothing is ever inherited across providers in the first place**. An explicit `agent.profiles.<role>.model` still survives a swap — a pin the user wrote is the user's own escape hatch, not inheritance.

There is exactly **one** cross-role fallback and it lives on the **provider** side (`providers.<p>.profiles.default`). The agent side has none.

An empty value keeps its established meaning: `spawn.WithProfile`'s token-drop (so the CLI's own default applies) and, in the YAML `model` key, "inherit the session/orchestrator model". The depth knob is applied only when `--provider` was **not** supplied, so an explicitly-empty `--provider=` resolves an empty provider — a lookup failure for the caller to report — rather than falling through to the knob.

#### Scenario: a provider swap refills from that provider

- **GIVEN** `agent.workers: codex` and `providers.codex.profiles: { review: { model: <codex-model-id>, effort: high } }`
- **WHEN** `fab agent review -o yaml` runs
- **THEN** `provider: codex`, `model: <codex-model-id>`, `effort: high`
- **AND** the other workers roles keep codex's **shipped** fills (the override merges per role, then per field)
- **AND** with no `providers:` block at all, every workers role still resolves codex's own shipped fill — never claude's values

#### Scenario: a provider carrying no fills resolves empty

- **GIVEN** `agent.workers: mine` and a project-defined `providers.mine` carrying commands but no `profiles` — the same shape the built-in `kimi` ships deliberately
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

### Requirement: deprecated `fab resolve-agent <stage|role> [--alias] [--provider <name>] [--model <id>] [--effort <level>]` compatibility surface

`fab resolve-agent` is deprecated for skill dispatch; skills consume `fab agent <stage|role> -o yaml`. The compatibility command SHALL continue to accept a **stage** name OR a **role** name positionally — a stage maps through the fixed mapping, a role resolves directly, and **role names are checked first** (`agent.RoleForName`). The two name sets overlap only at **fixed points**: a name shared by a stage and a role (`review`, `hydrate`) is one where the stage maps to that same-named role (`stageRoles[name] == name`), so the role-first check resolves such a name to the identical profile either interpretation would (`ship` is a stage but NOT a role — it maps to `fast` — so `resolve-agent ship` and `resolve-agent fast` both resolve to the `fast` profile, one via the stage mapping, the other directly). It resolves the role → `{provider, model, effort}` per § Fill precedence and emits, **verbatim, with NO validation**:

- `model=<id>` (always; empty = the inherit signal),
- `effort=<level>` (omitted when empty),
- `provider=<name>` (omitted when empty),
- `dispatch=<command>` — absent exactly when the shared mode selector lands on native; present with the substituted `interactive_command` for pane or `headless_command` for headless. Selection starts at `dispatch.mode`, descends pane → native → headless without ascending, and fails only when no reachable capability exists. The command always embeds the full model ID even under `--alias`.

`--alias` maps the `model=` line to the Claude-Code Agent-tool short alias (`opus`/`sonnet`/`haiku`/`fable`) — the Agent tool's `model` param is a hard enum that rejects full IDs; the `dispatch=` line is unaffected (full ID).

**Invocation-time overrides** are the fill precedence's top rung, applied by re-running the chain with the flags on top (`agent.ResolveRoleWith`) rather than patching a resolved profile:

- **`--provider <name>`** swaps the provider, refills model/effort from that provider, and re-runs the same `dispatch.mode` ladder against its capabilities. The emitted `dispatch=` presence may therefore differ from the unoverridden profile.
- **`--model <id>` / `--effort <level>`** override the corresponding field and are valid **without** `--provider` — a within-role override of the profile this pure query would otherwise print. `fab agent` accepts the same bare overrides (a final post-refill layer on every addressing form), so the two commands agree on override legality.
- All three key on cobra's `Flag.Changed` — whether the flag was **supplied** — so `--model=` explicitly *clears* the role's model (emitting the inherit signal) rather than being silently ignored, and `--provider=` is a lookup failure rather than a fall-through to the knob.
- An **unknown `--provider` name** is a non-zero-exit **lookup** failure listing the resolvable names, byte-identical to `fab agent`'s because both call the shared `unknownProviderError(cfg, name)` helper in `cmd/fab`. Overrides themselves are applied with **no validation** (provider neutrality).

**Overrides bind the native Agent-tool arm only.** The emitted `dispatch=` line is a **query result**, not an adapter move: `fab dispatch start` takes no override flags and re-resolves the stage from config itself, so an overridden profile cannot ride the CLI adapter. Relocating a stage between native Agent-tool dispatch and CLI dispatch takes a **config override** (a depth knob, or `agent.profiles.<role>.provider`), never an invocation flag. See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution for the dispatch-site wiring and the two-remedies rule.

#### Scenario: `--alias` aliases `model=` while `dispatch=` keeps the full ID

- **GIVEN** a role resolving to a provider with a `headless_command`
- **WHEN** `fab resolve-agent <stage> --alias` runs
- **THEN** `model=` carries the short alias while `dispatch=` embeds the full model ID
- **AND** under `--provider codex --model <codex-model-id> --alias` the non-Claude model passes through **verbatim** on `model=` (no prefix matched) while `dispatch=` embeds the same full ID

#### Scenario: overriding a stage's provider on the resolve call

- **GIVEN** a default config (apply → `doing` → claude)
- **WHEN** `fab resolve-agent apply --provider codex` runs
- **THEN** `provider=codex` with codex's own `doing` fill — its shipped model (inherited from codex's `default`, since the `doing` row carries effort only) at codex's `doing` effort — and `dispatch=codex exec --dangerously-bypass-approvals-and-sandbox …` with both placeholders substituted
- **WHEN** `fab resolve-agent apply --effort high` runs (no `--provider`)
- **THEN** `provider=claude` with the role's own model and `effort=high` — a within-role override, not a usage error
- **WHEN** `fab resolve-agent apply --provider bogus` runs
- **THEN** it exits non-zero naming `bogus`, lists `agy, claude, codex, kimi`, and prints no profile

### Requirement: One resolution object, one composer, two projections

`internal/agent.Resolution` is the complete resolution result shared by `fab resolve-agent` and `fab agent`. It carries `selector`, `kind`, `role`, `provider`, the full `model`, `model_alias`, `effort`, the raw `template`, `fill_mode` (`template` or `append`), the composed `command`, per-field `source` provenance, and an optional `dispatch` value containing a labelled `rung` and composed `command`. A nil dispatch denotes the native rung.

`cmd/fab/resolution.go` owns the single profile-to-resolution composer. Both commands normalize their addressing and overrides into that composer, then render projections of the returned value: `Resolution.Lines(alias)` produces `fab resolve-agent`'s frozen ordered line protocol, while `Resolution.YAML()` produces `fab agent -o yaml`'s full schema. The composer uses `agent.ResolveRoleWithSource` for the profile and provenance, `spawn.IsTemplate` plus `spawn.WithProfile` for fill mode and command composition, and `dispatch.SelectMode` for the optional dispatch projection. `ResolveRoleWith` delegates to `ResolveRoleWithSource`, so fill precedence and its trace share one implementation; `spawn.IsTemplate` exports the discriminator already used by `spawn.WithProfile`.

Only the YAML sink on `fab agent` requests dispatch derivation. `--print`, exec, and `-t` consume the same resolution object without selecting a dispatch rung, while `-o yaml` and `fab resolve-agent` share the actionable no-capability error posture.

#### Scenario: line and YAML projections agree

- **GIVEN** the same selector, config, overrides, and tmux availability
- **WHEN** `fab resolve-agent` renders its line projection and `fab agent -o yaml` renders its structured projection
- **THEN** their provider, model, effort, and non-native dispatch command come from the same `agent.Resolution`
- **AND** the line projection preserves its byte-stable `model=` / optional `effort=` / optional `provider=` / optional `dispatch=` contract

### Requirement: `dispatch.mode` — preference-bounded capability descent

`dispatch.mode` is `pane`, `native`, or `headless`, defaults to `native`, and has scope `both`. The project layer overrides the machine-wide system preference. It is a **ceiling**: selection starts at that rung and moves only downward through pane → native → headless.

- Pane is possible when tmux is available and the provider has `interactive_command`.
- Native is possible when the provider declares `native: true`.
- Headless is possible when the provider has `headless_command`.
- The first possible rung wins. Selection fails only when no rung at or below the preference is possible.
- Invalid values warn and fall back to `native`. The retired boolean has no read-time alias.
- `fab agent <stage> -o yaml` and `fab dispatch start|restart` use the same pure selector; start/restart supply the real tmux reachability result before writing any state.

#### Scenario: native preference descends for a non-native provider

- **GIVEN** `dispatch.mode: native` and a role resolving to built-in codex
- **WHEN** `fab agent <stage> -o yaml` runs
- **THEN** native is skipped and `dispatch.command` carries codex's headless command

#### Scenario: pane preference never ascends

- **GIVEN** `dispatch.mode: pane`, no reachable tmux, and built-in claude
- **WHEN** the role is resolved
- **THEN** the selector descends to native and omits `dispatch:`; it does not jump directly to headless while native is possible

#### Scenario: headless preference ignores higher capabilities

- **GIVEN** `dispatch.mode: headless` and built-in claude, which also supports pane and native
- **WHEN** the role is resolved
- **THEN** `dispatch.command` carries claude's headless command because the selector never ascends

### Requirement: `fab agent [role|stage] [--provider <name>] [--model <id>] [--effort <level>] [--headless] [-t|--template] [-o yaml] [--workers <provider>] [-p|--print] [--repo <path>] [-- <agent-args>...]` — session launcher

`fab agent` SHALL compose an interactive session command through an **addressing form** and **exec it in the current shell** (or print it — one print-family sink per invocation):

- **Selector-addressed** (the `[role|stage]` positional) — resolve a role profile through the full chain (`default` when the positional is omitted; the six role names and the six stage names — `intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr` — are accepted; a stage name maps through the fixed `stageRoles` table to its role via `agent.RoleForName`, role set first, and the `review`/`hydrate` collisions are fixed points so either name resolves identically) and compose `providers.<profile.provider>.interactive_command` with the role's `{model}`/`{effort}`. Since `default` and `operator` are session roles, this is the path `agent.session` governs: `fab agent` starts the default-role agent right here, `fab agent operator` starts the coordinator profile.
- **Selector + `--provider`** — re-resolves the selector's role from the named provider's own fills (`agent.ResolveRoleWith` with the provider pinned): the provider pin refills from `providers.<p>.profiles.<role>`, then that provider's `.default`, then empty, while an explicit `agent.profiles.<role>.model` pin still wins. So `fab agent apply --provider kimi --print` prints kimi's own command shape with kimi's fills, not claude's model patched in.
- **Provider-addressed** (`--provider <name>`, no selector) — **bypass role resolution entirely**: look up `providers.<name>` directly via `agent.ResolveProvider` (project config per-field-merged over the built-in table, exactly as the selector path's provider lookup does) and compose its `interactive_command` with the `--model`/`--effort` values. This is the "give me a codex session right here" form — no role need name the provider first.

All forms compose through the same `spawn.WithProfile` (template substitution or Claude-style flag append — see [configuration.md](/_shared/configuration.md) § `providers`); `--model`/`--effort` are **general final overrides**, valid on every addressing form (bare, selector, selector+`--provider`, provider-alone), applied verbatim after role resolution/provider refill and keyed on the flag being **supplied** (cobra's `Flag.Changed`) — the YAML and deprecated line projections share these override semantics. The print-family sinks share `--repo`:

- `-p, --print` prints the fully-resolved command instead of executing — the output is **profile-resolved** (model/effort substituted), so callers that spawn from the printed command get the profile. `-p` is a pure shorthand, byte-identical to `--print`.
- `-t, --template` prints the selected provider's command template **unsubstituted** — a tap before the fill step, `{model}`/`{effort}` placeholders intact. It implies print, combines with any selector, `--provider`, and `--headless` (they pick which template), and rejects `--model`/`--effort` with a usage error (they feed the substitution step that `-t` skips).
- `-o, --output yaml` emits the full resolution as structured YAML. `selector`, `kind`, `role`, `provider`, `model`, `effort`, and `command` appear first in that order, followed by `model_alias`, raw `template`, `fill_mode`, per-field `source`, and optional labelled `dispatch`. `model_alias` is the Agent-tool short alias for Claude IDs and empty otherwise; `source` names the actual provider/model/effort precedence rung (empty means inherit); `dispatch` is absent exactly for native and otherwise contains `rung: pane|headless` plus its substituted `command`. YAML alone derives dispatch. It implies print, accepts exactly `yaml` (anything else is a usage error), and is mutually exclusive with `--print` and with `-t` — one output sink per invocation.
- `--headless` resolves the provider's `headless_command` instead of `interactive_command`. It is valid only in the print-family modes (`--print`, `-t`, `-o yaml`) — exec of a headless command is a usage error. A provider with no `headless_command` hard-errors naming the config key (`configure providers.<name>.headless_command`).
- `--repo <path>` reads the target repo's config (the operator's fetch-another-repo's-command use case). Composes with any addressing form.
- `--workers <provider>` sets `FAB_AGENT_WORKERS=<provider>` in the environment passed to the exec seam without changing any addressing form's command resolution. An entry inherited from the parent environment is removed rather than shadowed, so the override is authoritative regardless of how a consumer resolves duplicate entries. It is accepted with `--print`, but printed output remains the command alone because no child process is executed.
- `fab agent` exec does NOT TTY-guard — exec-and-let-the-CLI-fail is acceptable (the underlying agent CLI already handles no-TTY), matching the document-don't-validate contract.

Provider-form rules (the **bare** `--provider <name>` form — selector forms take the fill rungs):

- **Omitted `--model`/`--effort`** leave the value empty and follow `WithProfile`'s empty-value rule (template mode drops the placeholder's token plus a preceding `-`-flag; append mode omits the flag), so a profile-free provider invocation results and the installed CLI's own default model applies — the way to spawn a provider whose current model IDs the caller does not know. Provider-level flags without placeholders remain in that command. **This path bypasses the provider's per-role fills** (the bypass is total — it skips role resolution *and* the fill), so an empty flag means empty, not the configured fill. The fills apply on selector-addressed role resolution (`fab agent <stage|role> -o yaml` or selector+`--provider`), not on the bare launcher form.
- Mode selection and the guards key on cobra's `Flag.Changed` — whether the flag was **supplied** — not on its value being non-empty, so an explicitly-empty `--provider=` resolves through the lookup and an explicitly-empty `--model=` clears the field on the selector paths rather than being ignored.
- **An unknown provider name** is a non-zero-exit **lookup** failure listing the available names (`agent.ProviderNames`: fab-kit's built-in table ∪ the project's `providers:` keys via `config.ProviderNames`, sorted). Listing resolvable *names* is not validation of a command's *content* — resolved strings still pass through verbatim.
- A provider that resolves but carries no valid command slot yields a config-key hint error — `configure providers.<name>.interactive_command` by default, or `configure providers.<name>.headless_command` when `--headless` is set — on either addressing path.

The procedural knowledge for *using* a composed command — opening it in a tmux window, delivering a prompt, peeking, awaiting — plus the per-provider invocation grammar and model-discovery recipes live in the `_cli-agents` helper: see [agent-primitives.md](/runtime/agent-primitives.md).

#### Scenario: a stage selector and selector+`--provider` re-resolve

- **GIVEN** a default config (apply → `doing` → claude; the built-in kimi ships empty fills)
- **WHEN** `fab agent apply --print` runs
- **THEN** stdout is the `doing` role's composed command, byte-identical to `fab agent doing --print`
- **AND** `fab agent apply --provider kimi --print` prints kimi's own command shape (`kimi --auto`) with kimi's fills — the empty `{model}` drops the `-m` pair — rather than claude's model patched into kimi's template
- **AND** `fab agent bogus --print` exits non-zero, naming the valid role and stage names

#### Scenario: provider-addressed spawn with no model supplied

- **GIVEN** `providers.codex.interactive_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'`
- **WHEN** `fab agent --provider codex --print` runs
- **THEN** stdout is `codex --dangerously-bypass-approvals-and-sandbox` — both placeholder tokens and their preceding flags are dropped while the provider-level bypass flag remains — so the CLI's own default model applies
- **AND** `fab agent --provider codex --model <codex-model-id> --effort high --print` prints `codex --dangerously-bypass-approvals-and-sandbox -m <codex-model-id> -c model_reasoning_effort=high`

#### Scenario: unknown provider name

- **GIVEN** a config whose `providers:` block defines only `claude`
- **WHEN** `fab agent --provider bogus --print` runs
- **THEN** it exits non-zero, naming `bogus` and listing the available provider names, and prints no command

### Requirement: The machinery is documented, not scaffolded

The registry (`internal/configref`) advertises `agent.session` and `agent.workers` (`advertise: true`) and marks `agent.profiles` and `providers` `advertise: false`. So the `fab config upgrade` **managed fence** in each project scaffolds a ~20-line `agent:` block — the two knobs, the role→depth partition compacted to one line per depth, and a commented `profiles:` example — and no `providers:` block at all, while `fab config explain` (and `fab config explain --json`, and `fab config init --system`) still document both in full. The rendered partition lines derive from the exported `agent.IsSessionRole`, so the fence never re-encodes which roles sit at which depth. Because `agent` is one YAML block, the whole block is rendered by a **single** segment (owned by the `agent.session` row) — two segments emitting a live `agent:` parent would collide into a duplicate key. See [configuration.md](/_shared/configuration.md) § Schema Discovery.

### Requirement: Deprecated spellings stay readable

Deprecated spellings SHALL keep resolving so a config that has not yet run the corresponding migration is never silently ignored:

- **`agent.tiers.<role>`** — read per role as the fallback when `agent.profiles` has no entry for that role (`config.GetAgentProfile`), so a half-migrated config resolves every role. The registry row for `agent.profiles` carries `renamed_from: agent.tiers`; the `2.16.19-to-2.17.0` migration rewrites the disk.
- **`providers.<name>.model` / `.effort`** — the flat per-provider fill, a true **alias** for `profiles.default`: `ResolveProvider` folds an override's flat fields into **that override's own** `profiles.default`, per field, before merging fab-kit's built-in table (`withFlatFillAlias`). So a user's flat pin outranks the shipped `profiles.default` it means to replace, while the user's own `profiles.default` still wins over their flat spelling — and a built-in **role** fill still outranks the folded value, exactly as it would outrank a hand-written `profiles.default`. Because the fold happens per field, a flat `model` on codex reaches every role whose shipped row sets no model of its own.
- **`providers.<name>.session_command` / `.dispatch_command`** — the pre-2.19 spellings of the renamed command fields, read as **per-field fallbacks**: a non-empty `interactive_command`/`headless_command` wins, independently per field (`ResolveProvider`), so a half-migrated config resolves both fields. The alias is silent (no deprecation warning): the struct keeps both yaml tags, so decode retains whichever spelling a layer carries, and `ResolveProvider` applies the per-field preference after the cascade merges — so every cascade layer, env `FAB_PROVIDERS` values included, resolves either spelling. The `providers` registry row carries an informational `renamed_from` (the top-level mechanical carry cannot fire on nested keys); the `2.18.1-to-2.19.0` migration rewrites both scopes on disk, and the `fab config set/unset/explain/show` dotted-key matcher accepts the new spellings only — the alias is a read affordance, not a write surface.

The read-time aliases are what make the rename safe on their own: `configupgrade`'s `renamed_from` carry is a **top-level-key** operation and deliberately skips a rename *inside* the `agent:` block, so without the aliases `fab config upgrade` (auto-run by `fab upgrade-repo`) would leave a live `agent:` block that had silently stopped being read. The migration `2.16.19-to-2.17.0` performs the on-disk rewrite in **both scopes** (project `fab/project/config.yaml` and system `~/.fab-kit/config.yaml`) and warns on the two shapes it cannot mechanically preserve: an `agent.tiers.default` carrying model/effort (whose cross-role re-basing has no successor) and a flat provider fill, whose **reach** differs across the 2.17.0 boundary in opposite directions per provider — a gain of one role on claude (exhaustive six-role map, where a flat fill is inert pre-2.17.0 and fills the `default` role after), a loss on any sparse-map non-claude provider (those maps shipped no fills pre-2.17.0, so a flat fill that reached every role stops wherever fab-kit ships a role fill). The rewrite itself is resolution-neutral where both scopes are swept together; the one shape where it is not is the mixed-layer inversion below, which the migration names in its verification steps. See [distribution/migrations.md](/distribution/migrations.md).

**Cross-scope precedence inverts during the pre-migration window.** The alias resolves *after* the scope cascade: `LoadPath` merges the system and project layers per key first, leaving `profiles` and `tiers` as two separate maps, and `GetAgentProfile` then prefers `profiles` wherever it carries the role. So for a role written in the **new** spelling in one scope and the **legacy** spelling in the other, the spelling decides rather than the scope — a project-layer `agent.profiles.<role>` beats a system-layer `agent.tiers.<role>`, inverting the documented system > project precedence. It bites only a hand-half-migrated pair of scopes, and running the migration (which sweeps both files) restores normal precedence. Pinned by `TestResolveCrossScopeLegacyAliasPrecedence` in `cmd/fab` — the layer that can compose both scopes — and stated at `GetAgentProfile`'s doc comment.

**The provider-side flat fill has the same twin.** `withFlatFillAlias` also runs *after* the scope cascade, so a project-layer `providers.<name>.profiles.default` beats a system-layer flat `providers.<name>.model` per field — the spelling decides rather than the scope, the same inversion one layer over. Retiring both twins needs a layer-aware fold (aliasing before `config.MergeLayers`, or handing the resolver per-layer inputs); until then the shape is documented in the `2.16.19-to-2.17.0` migration's verification steps as the mixed-layer exception.

**The renamed command fields inherit the same shape.** `MergeLayers` merges the raw YAML trees per key and the command-field fallback runs after, so during the migration window a system-scope `interactive_command` outranks a project-scope `session_command` for the same provider — the spelling decides rather than the scope, consistent with new-spelling-wins-per-field. Deliberate: sweeping both scopes with the `2.18.1-to-2.19.0` migration restores normal precedence.

#### Scenario: a legacy `agent.tiers:` block still resolves

- **GIVEN** a config carrying only `agent: { tiers: { doing: { effort: medium } } }`
- **WHEN** `fab agent apply -o yaml` runs
- **THEN** `effort: medium` — the legacy key still resolves
- **AND** a config carrying **both** `profiles.doing` and `tiers.doing` resolves from `profiles.doing`

## Design Decisions

### The Depth Knob Is the Provider Rung, Not a Whole-Profile Switch
**Decision**: `agent.session`/`agent.workers` supply only the **provider** rung of the chain; model and effort always come from the resolved provider's own per-role fills. The role→depth partition behind them is fixed and fab-owned (`default`/`operator` → session; `doing`/`review`/`hydrate`/`fast` → workers), exposed to renderers through `agent.IsSessionRole` so no consumer re-encodes it.
**Why**: It is what makes "claude for tier 1, codex for tier 2" a two-word config while still letting each provider keep sensible per-role models. A knob carrying a whole profile would name one model for every role it governs, flattening the role taxonomy and duplicating the per-role fills the provider table already owns. Depth is the right axis because it is mechanically real: a session role's provider is fixed when the session launches, a workers role's is re-resolved at every dispatch.
**Rejected**: Making the knob a `{provider, model, effort}` object (one model for every role it governs — the role differentiation the per-role fills exist to provide, lost in the advertised surface); keying the partition off stage names (the partition is about *depth*, and `default`/`operator` are not stages); a user-overridable partition (fab-owned taxonomy, like the stage→role map).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

### One Fallback Chain, on the Provider Side Only
**Decision**: Cross-role fallback exists exactly once, as `providers.<p>.profiles.default`. The agent side has none — `agent.profiles` is sparse and `agent.profiles.default` is the `default` role's own override, never a fallback source for another role. Model/effort are always sourced from the **resolved** provider's fills, so a provider swap re-derives them.
**Why**: A single provider-anchored chain makes cross-provider leakage structurally impossible — no value can be foreign to the provider that will run it — so no cutoff rule, per-field ownership tracking, or net-configured-provider gate is needed to police it, and there is no cross-scope ownership limitation to document. Two competing chains are what force such a rule into existence.
**Rejected**: Agent-side `default`-role inheritance plus a cross-provider field cutoff (preserves the footgun and its documentation debt, including a cascade-blind ownership computation); erroring when a role resolves an empty model (an empty model is the established inherit-the-session-model / CLI-default signal); validating provider/model compatibility (breaks provider neutrality).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

### Read-Time Aliases Back a Rename, Not Just the Migration
**Decision**: Every renamed provider/agent key stays readable as a deprecated alias in the resolver, positioned below its modern counterpart in the chain, while the migration performs the on-disk rewrite in both scopes. Three families carry it: `agent.tiers` (per role), the flat `providers.<p>.model`/`.effort` (per field, folded into `profiles.default`), and `providers.<p>.session_command`/`.dispatch_command` (per field, against `interactive_command`/`headless_command`). All three are silent — no deprecation warning — and all three are **read** affordances: the `fab config` write surface accepts the modern spelling only.
**Why**: The registry's `renamed_from` carry-forward is a **top-level-key** operation in `configupgrade`, so it deliberately skips a rename sitting *inside* the `agent:` block or inside a `providers.<name>:` block — `renamed_from` alone is metadata, not a working carry. Without a read-time alias, `fab config upgrade` (auto-run by `fab upgrade-repo`) would leave a live block that silently stopped being read until the user ran `/fab-setup migrations`. Per-field/per-role preference rather than whole-block preference is what makes a **half-migrated** config resolve completely instead of resolving half of it and dropping the rest.
**Rejected**: Teaching `configupgrade` nested rename carry (a splice-engine change far beyond the scope, and the migration already does it correctly); relying on the migration alone (a silent behavior-change window for every user between upgrade and migration run); whole-block preference (one modern key would shadow every sibling still in the old spelling).
*Introduced by*: 260806-j9nh-agent-profiles-session-workers; 260809-n1he-rename-provider-command-fields

### Command Fields Name Interaction Mode, Not Depth
**Decision**: See the authoritative record in [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Provider Command Fields Name Their Interaction Mode, Not the Agent's Depth". In brief: `interactive_command`/`headless_command` split by how the agent is run, not by the depth of the agent running it — a Tier-2 pane worker runs the interactive command — and the `fab config` dotted-key write surface accepts those spellings exclusively.
**Why**: Depth-flavored names advertise an invariant the pane rung breaks, and `dispatch_command` collides with both the `fab dispatch` verbs and the `dispatch.*` block.
**Rejected**: Depth-aligned names; a hard break with no alias; old spellings on the write surface.
*Introduced by*: 260809-n1he-rename-provider-command-fields

### Providers Extracted; Roles; `fab agent` Retires `fab spawn-command`
**Decision**: See the authoritative record in [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Providers Extracted; Roles; `review_tools` → `code-review.md`". In brief: a top-level `providers:` table carries independent session/native/headless capabilities plus fills; roles resolve `{provider, model, effort}`; dispatch preference stays separate; review policy moved to `code-review.md`; `fab agent` owns session launch and structured resolution; the deprecated `resolve-agent` projection exposes `dispatch=` plus `provider=`.
**Why**: Conflating provider mechanics with role/budget policy actively confused (two fields both named `spawn_command`; the `thinking` name hid its referent, which was review). Extraction plus role naming attack the confusion at its source; commands living on the provider make `{provider, model, effort}` composition safe (no cross-semantics command inheritance).
**Rejected**: Merging the two command fields; folding a command in as a `default`-role field (implies the rejected cross-fallback); keeping `thinking`; provider inference from model strings; a `fab spawn-command` deprecation alias.
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### Positional Stage-or-Role; `provider=` Line; No TTY Guard
**Decision**: The deprecated `fab resolve-agent` compatibility surface accepts a stage OR role name positionally (role names checked first; shared names are fixed points), carries a `provider=` line, aliases only `model=`, and leaves `dispatch=` commands on the full model ID. `fab agent` exec does not TTY-guard.
**Why**: Reuse the existing positional surface rather than add flag surface for no disambiguation benefit; surface the provider rather than re-derive it downstream; keep the no-validation/document-don't-guard contract for TTY.
**Rejected**: A `--role` flag (surface for no benefit); inferring provider downstream (re-does resolution); a TTY guard (the agent CLI already handles no-TTY).
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### Resolution Composition Lives Beside Both CLI Consumers
**Decision**: `internal/agent` owns `Resolution`, its line/YAML projections, and the traced role resolver; `cmd/fab/resolution.go` owns the single composer used by both `fab resolve-agent` and `fab agent`.
**Why**: `internal/spawn` imports `internal/agent`, so composing through `spawn.WithProfile` inside `internal/agent` would create an import cycle. Both CLI consumers already share package `main`, which provides one composition site without changing the package graph.
**Rejected**: Moving `WithProfile` into `internal/agent`; creating an `internal/resolution` package for a composer with only two package-main consumers.
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

### Provenance Is Captured During Fill Resolution
**Decision**: `ResolveRoleWithSource` returns the resolved profile with the provider/model/effort source rungs, and `ResolveRoleWith` delegates to it while discarding only the trace.
**Why**: Capturing the source when each precedence rung is consulted preserves a single fill-precedence implementation and distinguishes equal values supplied by different rungs.
**Rejected**: Inferring provenance after resolution by comparing the final value with every candidate rung.
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

### Structured YAML Shares the Resolver's Dispatch Failure Semantics
**Decision**: `fab agent -o yaml` derives dispatch for role, stage, and provider addressing and returns the same actionable no-capability error as `fab resolve-agent`; every other `fab agent` sink skips dispatch derivation.
**Why**: Structured consumers receive the same dispatch truth as the frozen line surface, including the distinction between native absence and an unreachable ladder.
**Rejected**: Omitting `dispatch` when selection fails; deriving dispatch only for stage selectors; adding dispatch-derived errors to `--print`, exec, or `-t`.
*Introduced by*: 260901-mp8d-agent-resolution-struct-parity

### `hydrate` Is Its Own Role; `fast` Keeps Its Role Name (Multi-Referent)
**Decision**: `hydrate` is its own role rather than part of `doing`, giving six roles (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`). A role is stage-named only where it maps 1:1 to a single referent (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is multi-referent — `fast` in particular governs both the ship stage and the `/fab-proceed` prefix-step dispatches.
**Why**: Memory writing (hydrate) is knowledge work with a different cognitive profile than apply's diff work, so it deserves its own model/effort — grouped under `doing` it could never run cheaper or on a different model than apply. A stage name (`ship`) would misname `fast` once it also governs the prefix steps.
**Rejected**: Naming the role `ship` (misnames a multi-referent role and would force an unnecessary carry-forward migration); six stage-named roles / dissolving roles entirely (`default`/`doing`/`fast` are genuinely multi-referent and the role names carry the why); a user-overridable stage→role mapping or per-stage escape hatch (taxonomy stays fab-owned).
*Introduced by*: 260719-g55d-stage-model-tier-defaults-v2

### Bare `--provider` Is a Total Bypass; Selector+`--provider` Re-Resolves
**Decision**: `fab agent --provider <name>` with **no selector** bypasses role resolution entirely — a direct `providers.<name>` lookup with `--model`/`--effort` as the whole profile — rather than synthesizing an ad-hoc role. A **selector combined with `--provider`** re-resolves the selector's role from the named provider's own fills (`agent.ResolveRoleWith` with the provider pinned, the provider rung of the same precedence chain). `--model`/`--effort` are general overrides, valid on every addressing form and applied verbatim after resolution/refill. All guards key on cobra's `Flag.Changed` (was the flag supplied?) rather than value emptiness.
**Why**: The bare-provider form answers a different, mechanical question ("give me a codex session") from role resolution, and the bypass leaves the resolution path untouched. The pin form occupies exactly the precedence chain's existing provider rung, so combining a selector with `--provider` is unambiguous — an explicitly-empty `--provider=` or `--model=` falling silently through would lose that signal, making supplied-ness the correct test.
**Rejected**: A role-addressed provider pin treated as the total bypass (the selector form refills through the chain rather than bypassing it); auto-creating a synthetic role (invents state the config never declared); silently ignoring `--model`/`--effort` on any form (surprise-inducing CLI behavior).
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
**Rejected**: New resolution flags on dispatch, tmux-version-specific `new-window -e`, provider validation, or persisted config edits.
*Introduced by*: 260808-2d1w-env-override-layer-launch-flags

### Built-in Providers Ship Grammar AND Per-Role Fills, Refreshed at Kit-Release Cadence
**Decision**: fab-kit's built-in provider table carries both halves — invocation grammar *and* per-role fills — inside the binary, as rows in the embedded `defaults.yaml`, for every provider whose CLI takes a portable model identifier (claude, codex, agy). The non-claude fills carry no staleness automation: no CI check against provider APIs, no model-catalog fetch. They are refreshed at **kit-release cadence** by editing the file, pass through **unvalidated** like every other resolved string, and are corrected by **one config line** (`providers.<name>.profiles.<role>.model`, settable once per machine because `providers` is `scope: both`). They are never seeded into a user's `config.yaml`, so an upgrade refreshes them and no project pins rot in place.
**Why**: Shipping no fills was the *silent* failure: `agent.workers: codex` resolved an empty model identically for all four workers roles, so the provider CLI's own default ran everywhere and the role taxonomy flattened exactly where a user first exercises the knob. A stale ID is the loud, cheap failure by comparison — the CLI rejects it immediately and the fix is one line. Two more facts close the gap the rot argument was defending: a bump is a reviewable data diff in an embedded YAML file, pinned by a test so it is always deliberate; and fab-kit releases every few days, so users see refreshed suggestions at kit cadence rather than CLI cadence. presence=intent is untouched — a built-in provider is inert until a knob, role override, or flag names it, so adding fills changes no default behavior while both depth knobs ship `claude`.
**Rejected**: Keeping fills out of the release (preserves the silent role-flattening for the flagship one-line UX); a CI staleness check or catalog fetch against provider APIs (fab is validation-free by constitution, and a network dependency in resolution is worse than a stale string); seeding the fills into each project's `config.yaml` (pins in every repo, rotting independently of the binary); inferring a provider from a model string (breaks provider neutrality).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill; *Updated by*: 260806-ywkx-ship-codex-gemini-fills, 260808-rpsr-remove-gemini-add-agy-kimi

### Non-Claude Built-in Commands Run Full-Auto
**Decision**: Both codex command forms carry `--dangerously-bypass-approvals-and-sandbox` and both agy command forms carry `--dangerously-skip-permissions`; kimi carries `--auto` on its interactive command and no approval flag on its headless one, because `kimi -p` is already non-interactive, auto-approves tool calls, and *rejects* `--yolo`/`--auto`. Project and system provider-command overrides remain the approval-gated escape hatch.
**Why**: Headless and pane stage workers are unattended and have no approval-answering channel; ship and review-pr also require network and repository operations. Explicit bypass grammar gives the non-claude built-ins the same autonomous execution policy as claude's shipped `--permission-mode bypassPermissions` command — expressed in each CLI's own vocabulary, including "no flag" where the CLI's headless mode already implies it.
**Rejected**: Codex `--full-auto` (retains a workspace-write sandbox that blocks required network operations); bypassing only `headless_command` (leaves pane workers gated); adding an approval flag to kimi's dispatch form for symmetry (the CLI errors on it); approval-gated built-ins or per-project fixes (break the complete one-knob provider swap).
*Introduced by*: 260808-clxw-codex-gemini-bypass-flags; *Updated by*: 260808-rpsr-remove-gemini-add-agy-kimi

### Codex's Fills Are Catalog Slugs; agy's Embed Effort in the ID; kimi Ships None
**Decision**: `providers.codex.profiles` ships concrete model slugs, sparse and per field — a `default` model plus effort, higher effort on `doing`/`review`, and a cheaper model at low effort on `fast`. `providers.agy.profiles` ships two model-only rows (`default`, `fast`) and no `effort` anywhere. `providers.kimi` ships no `profiles` map at all.
**Why**: Each CLI's own model-addressing vocabulary decides the shape. The codex CLI exposes no alias mechanism (`-m` takes a slug), so concrete IDs are the only option; the shipped ones were read from the installed CLI's own model catalog rather than from documentation, which is the closest thing to an authoritative source, and the supported reasoning levels were checked there too. agy's IDs *embed* the reasoning level as a suffix (`gemini-3.1-pro-high`), so effort has no separate field to occupy and a `--effort` flag would fight the suffix. kimi's `-m` takes a **user-config alias** rather than a catalog ID, so there is no value that is correct for every install — the empty model drops the flag and the user's own `default_model` applies. Sparseness is what buys role differentiation cheaply: rows omitted from a map inherit `default` through the per-field merge.
**Rejected**: Pinning kimi's managed-install alias `k3` (breaks custom-provider installs) or giving it a lone `default` fill (same problem, one row smaller); a separate `--effort` flag on agy alongside its suffixed IDs; codex slugs quoted from docs or memory rather than the installed catalog (the placeholders carried in from intake were absent from it entirely, and the next tier down carried deprecation notices).
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills; *Updated by*: 260808-rpsr-remove-gemini-add-agy-kimi

### The Nested-Shell `"$(cat)"` Idiom Delivers stdin to an Argument-Taking CLI
**Decision**: agy's and kimi's `headless_command`s wrap the CLI in `sh -c '… -p "$(cat)"'` rather than invoking it directly.
**Why**: `fab dispatch` delivers the stage prompt on **stdin**, but both CLIs take the prompt as an *argument* to `-p` and never read stdin. POSIX expands `$(cat)` **before** the `< file` redirect applies, so the un-nested form reads the *outer* stdin and the worker starts with an empty prompt; nesting a shell makes the inner `sh`'s stdin the redirected prompt file. Verified end-to-end in dispatch shape (`cmd < prompt > log 2>&1`) against both CLIs. Absorbing the difference in the provider's grammar keeps the dispatch machinery provider-neutral — no per-CLI prompt-delivery branch in Go.
**Rejected**: Passing `-p` with no argument (both CLIs error: `flag needs an argument: -p`); teaching `fab dispatch` a per-provider prompt-delivery mode (provider mechanics belong in the provider's command string).
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi

### Pane Capability Follows Launch Grammar Plus the Generic Gate
**Decision**: Every built-in provider ships pane launch grammar, and first-run walls are handled through the provider-neutral readiness gate. agy carries the exact interactive grammar `agy --dangerously-skip-permissions --model {model}`; its exact workspace path may be user-seeded under `trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json`, while fab does not write that provider-owned store.
**Why**: Interactive operation is common to agent CLIs, while headless prompt grammar is provider-specific. The readiness gate already separates launch from input and classifies boot and first-run walls without provider branches.
**Rejected**: Withholding `interactive_command` because a CLI has a first-run wall; encoding a separate pane-eligibility bit; provider-specific trust-wall handling or fab-owned trust-store writes.
*Introduced by*: 260810-ttff-agy-interactive-pane-capability

### Exec contract for providers
**Decision**: A provider's `interactive_command` must exec its binary, so the binary — not a wrapper shell — owns the pane's foreground; all four built-ins already do. The contract is stated once in the provider-authoring guidance (`_cli-agents.md`); a wrapper provider fails observably (`booting` with the shell prompt in the snippet, then escalation), so no escape hatch is needed and no new field is added.
**Why**: The readiness gate's takeover precondition keys on the pane's foreground command, so takeover is the one shape requirement a pane provider's command must satisfy; with the gate enforcing it visibly, the contract needs no mechanism beyond documentation.
**Rejected**: Time-bounding the precondition (reopens the cooked-tty false-ready window after N seconds); comparing against `spawn_cmd` (brittle across `bash -c` argv shapes).
*Introduced by*: 260829-57mp-pane-readiness-agent-takeover

### kimi Ships the Probed `kimi --auto -m {model}` Verbatim
**Decision**: kimi's built-in `interactive_command` is exactly `kimi --auto -m {model}` — the invocation probed live against the CLI (2026-08-10, kimi 0.34.0) — and the value is pinned by a test rather than merely asserted present.
**Why**: A pane-eligibility field is only as good as the invocation it carries, and this one is proven end-to-end — it is the string a user-level `providers.kimi.interactive_command` override ran real pane workers with. `--auto` is where kimi's full-auto posture has to live, since the headless `-p` form rejects it; the `-m {model}` pair costs nothing because kimi ships no fills, so the empty model drops the pair and the CLI's own `default_model` applies. Pinning the value keeps a plausible-looking edit from shipping an unprobed invocation to every pane worker.
**Rejected**: `kimi --yolo -m {model}`, the form the older documentation illustrated (unproven against the delivery choreography, where `--auto` is what the probe actually used); presence-only assertion (any edit would pass); leaving the command to per-user config (the capability is a property of the CLI, not of one machine).
*Introduced by*: 260810-ki9v-kimi-pane-enablement

### A Built-in Roster Change Ships No Migration
**Decision**: Adding or retiring a built-in provider is a data edit to `defaults.yaml` plus its drift-guarded mirrors — no migration file, no compatibility alias, no special-case code path for a name the table does not define.
**Why**: The `providers:` table is user-extensible and provider names are opaque pass-through strings, so a config naming a provider fab-kit does not define already has a defined behavior: the ordinary unknown-provider lookup error, which names the resolvable set. That is a loud, one-line-fixable failure, and a user who wants a retired built-in back writes its block into their own config. A migration would have to guess a replacement provider for a role the user chose deliberately.
**Rejected**: A rewrite migration mapping a retired name onto a surviving provider (guesses intent, and silently changes which model runs a user's stages); a deprecation alias keeping the old name resolvable (carries dead grammar at release cadence — the cost the retirement removes).
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi

### The Flat Provider Fill Is a Real Alias for `profiles.default`
**Decision**: `ResolveProvider` folds an override's deprecated flat `model`/`effort` into **that override's own** `profiles.default`, per field, before merging fab-kit's built-in table — rather than reading the flat value as a rung *below* `profiles.default` during fill resolution. The user's own `profiles.default` wins where it sets a field; a built-in **role** fill still outranks the folded value.
**Why**: The flat spelling is *documented* as an alias for `profiles.default`, but was implemented as a lower-precedence rung. The two were indistinguishable while no non-claude built-in carried a `profiles.default`; shipping one makes the difference load-bearing, and the rung form would silently shadow a pre-migration user's own pinned model with fab-kit's shipped one — a regression introduced *by* shipping the fills, for exactly the un-migrated configs the alias exists to serve. Fixing the fold rather than the symptom also collapses `providerFill` to a two-rung role→default read.
**Rejected**: Keeping the rung and accepting the shadowing (a silent regression for the configs the alias serves); special-casing "built-in vs user" inside `providerFill` (it receives an already-merged `ProviderConfig` and cannot tell them apart).
**Known consequence**: the fold runs *after* the scope cascade, so a system-layer `profiles.default` beats a project-layer flat fill per field — the provider-side twin of the `agent.profiles`/`agent.tiers` cross-scope inversion, pre-existing and byte-identical under the former rung form. Documented in the `2.16.19-to-2.17.0` migration; retiring both twins needs a layer-aware fold.
*Introduced by*: 260806-ywkx-ship-codex-gemini-fills

### Invocation Overrides Ride Structured Resolution and Bind the Native Arm Only
**Decision**: Invocation-time provider/model/effort overrides are flags on `fab agent <stage> -o yaml`, the single resolution call every dispatch site makes. Because `fab dispatch start` carries no override surface — it re-resolves the stage from config — the overrides bind the **native Agent-tool arm** only, and every doc restating the surface carries that scope.
**Why**: Every dispatch site makes exactly one structured resolution call and branches on `dispatch:` key presence, so overriding at that call reuses the whole seam with zero new machinery; a separate override channel would need its own precedence rules and its own compliance-visibility contract. The YAML output is a pure projection of the profile, so overriding one field is unambiguous and useful. Scoping to the native arm keeps the docs describing what the code does instead of a workflow that errors on the headless arm and silently runs the wrong provider on the pane arm.
**Rejected**: Plumbing the flags through `fab dispatch start` (a second resolution surface with its own precedence and visibility contract); per-stage override config keys (persistent state for a per-run intent); forbidding bare `--model` on the resolution surface for symmetry (a usage error for an unambiguous query).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

### The Built-in Tables Are an Embedded Data File, Parsed Through the Config Schema
**Decision**: The provider grammars, claude's per-role fills, the depth knobs, and the `dispatch.*` built-in defaults live in `src/go/fab/defaults.yaml`, embedded with `//go:embed` and unmarshalled at package init into `config.Config` — the struct the user-config loader fills — with a malformed file panicking. `stageRoles` and `roleDepth` stay Go map literals, making the YAML/Go split the overridable/fixed boundary.
**Why**: The data is deeply nested (per-role fills per provider), and as Go literals it is verbose to review and unreadable to a non-Go reader doing a model bump. A data file also removes the built-ins↔`fab config explain` drift class **by construction** rather than by drift-guard test: the reference renders from the same bytes the resolver parses, so there is no Go-map→YAML translation left to drift. Parsing through the config schema is the only reading that keeps the file's config-fragment shape from silently diverging from the schema it imitates, and it makes the eventual layer-0 unification a merge-order change rather than a parser change. Embedding rather than reading `$(fab kit-path)` at runtime keeps the "resolution cannot fail" property: kit and binary release atomically, so an on-disk read buys nothing and adds a binary↔kit skew failure mode. Panicking is the honest response to compiled-in bytes that do not parse — a defective build artifact, not a state a released binary can reach.
**Rejected**: Reading the file from the kit cache at runtime (binary↔kit version skew on a path that cannot fail today); a bespoke defaults struct in `internal/agent` (a second schema definition to keep in sync) or hand-walking a `map[string]any` (loses the schema's yaml tags and zero-value semantics); returning an error from the accessors (widens the API of a behavior-neutral change for an unreachable state) or falling back to hardcoded Go values (restores the duplicate literals); `sync.Once`-on-first-use (defers a build defect to an arbitrary call site — there is no I/O to make lazy); moving `stageRoles`/`roleDepth` into the file too (taxonomy is fab-owned policy, and the split is what signals which half a user may override); placing the file under `src/kit/` (that tree deploys to the kit cache and would imply runtime reading).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0; *Updated by*: 260809-wll4-config-source-consolidation

### `defaults.yaml` Lives at the Module Root; a One-File Root Package Owns the Embed
**Decision**: `defaults.yaml` sits at the Go module root (`src/go/fab/defaults.yaml`), where a one-file root package (`src/go/fab/defaults.go`, `package fab`) owns the `//go:embed` and exports the raw bytes as `DefaultsYAML []byte`. `internal/agent` keeps all parsing, validation, and init-injection, consuming the bytes via `fabroot.DefaultsYAML`.
**Why**: The file is the most-referred-to data file in the module — the one humans open to check or bump a model — and the module root is the highest placement `go:embed` permits (the directive cannot reach above the embedding package's directory). Splitting embed ownership from parsing keeps the embed-over-kit-cache single-source design and every drift guard untouched: the move changes no value, symbol, or resolution rule.
**Rejected**: Repo-root or `src/` placement (a compile error under `go:embed`'s package-directory rule, and a copy-at-build step would reintroduce the two-copies drift the embedded single source exists to kill); leaving the file beside its consumer with only doc pointers (does not fix the discoverability complaint, and the `internal/` path wrongly signals "implementation detail" for user-facing overridable data).
*Introduced by*: 260823-a3mu-hoist-defaults-module-root

### The Built-in Command Identifiers Are Package Vars Sourced from the Embedded File
**Decision**: `DefaultInteractiveCommand` and the non-claude `DefaultCodex*`/`DefaultAgy*`/`DefaultKimi*` identifiers are package `var`s reading the parsed provider entries, not `const` literals; `spawn.DefaultSpawnCommand`, their only compile-time-constant consumer, is a `var` re-export of the same value. The identifiers are kept rather than replaced by `ResolveProvider(nil, name)` calls. The `config.DefaultDispatch*` symbols follow the same pattern — package-level vars carrying no literals, assigned from the parsed `dispatch:` block by agent's `init()` (the import cycle makes push-from-agent the only allowed direction — see [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Dispatch Defaults Are Init-Injected from `defaults.yaml`").
**Why**: `defaults.yaml` owns these strings — keeping them as Go literals would put every command text in two places and reintroduce by hand exactly the drift the data file removes by construction. Keeping the *names* leaves `internal/configref`, `internal/configupgrade`, and every test call site untouched.
**Rejected**: Keeping the constants canonical and having `defaults.yaml` restate them (drift-by-test, the thing being removed); a test asserting const↔YAML equality (the same drift, one indirection later); deleting the identifiers and rewriting every consumer (inflates a behavior-neutral diff).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0; *Updated by*: 260809-wll4-config-source-consolidation

### `DefaultProfile` Is Resolution Against a Nil Config
**Decision**: `agent.DefaultProfile(role)` is defined as `ResolveRole(nil, role)` rather than a lookup in a separate built-in role→profile table.
**Why**: There is no single built-in role→profile map to read any more — the values live under `providers.claude.profiles`, reached through the same chain everything else uses — and nil-config resolution is exactly "the built-in answer". It also keeps `fab config explain` sourcing its per-role defaults from the resolver rather than from a second table that could drift.
**Rejected**: A parallel built-in profile map (a second table to keep in sync with the resolver, which is the drift class the embedded data file exists to remove).
**Consequence — the registry row is the STATIC default; the read model composes a live one**: `configref.Fields()`'s `agent.profiles` default is built from `DefaultProfile`, which is knob-blind by construction. So `fab config show --origin` recomposes that one row through `configref.DefaultsMapFor`, resolving each role against the **live** config (`agent.ResolveRole`) twice, per leaf: the `provider` leaf against `knobsOnly` (the user's `agent.profiles`/`agent.tiers` entries stripped — the defaults tier reports the built-in a user override *shadows*, never an echo of that override), the `model`/`effort` leaves against `fillsOnly` (each per-role `provider` override kept, per-role model/effort overrides stripped — the built-in fill is a function of the provider the role actually dispatches to, and an overridden provider's own fills appear in no higher tier, so nothing echoes). The drill-down and `fab agent <stage|role> -o yaml` therefore agree both on which provider a role would dispatch to and on the model/effort fills it would carry. See [_shared/configuration.md](/_shared/configuration.md) § Six-Verb Surface.
*Introduced by*: 260806-j9nh-agent-profiles-session-workers; *Updated by*: 260812-05wy-config-show-role-provider-fills

### Unknown Provider Is a Lookup Failure That Names the Resolvable Set
**Decision**: An unknown `--provider` name exits non-zero listing the available provider names — the sorted union of fab-kit's built-in table and the project's `providers:` keys, exposed as `agent.ProviderNames(cfg)` over the nil-safe `config.ProviderNames()` accessor. The message is produced by a single `unknownProviderError(cfg, name)` helper in `cmd/fab`, shared verbatim by both flag-accepting commands (`fab agent`'s provider-addressed mode and `fab resolve-agent --provider`).
**Why**: The union is exactly the set `ResolveProvider` will accept, so it is the only set whose listing is actionable; sorting makes the message stable for tests and for readers. Naming resolvable *names* is not validation of command *content* — resolved command strings still pass through verbatim, preserving document-don't-validate (fab never infers a provider from a model string). One helper keeps the two commands' errors byte-identical, so a caller learns one phrasing.
**Rejected**: Listing only the project's configured providers (omits the built-in `claude`, which resolves fine); a bare "unknown provider" error (leaves the caller to guess the config surface); validating the resolved command string (breaks the document-don't-validate contract); a per-command copy of the formatter (two phrasings of one contract to drift).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

### User-Facing Go Strings Name the Skill-Consumed Resolution Surface
**Decision**: User-facing Go strings that direct stage or role resolution name `fab agent <stage|role> -o yaml`; the native-dispatch error also states the `dispatch:`-key absence discriminator, and tests pin both the error guidance and the generated config-reference text.
**Why**: Executable guidance and generated reference prose are part of the same behavioral claim as the skills. Keeping them on the structured surface prevents the binary from contradicting the workflow it ships.
**Rejected**: Splitting user-facing resolution guidance between `fab agent` and the deprecated compatibility command, or treating executable strings as outside the documentation claim.
*Introduced by*: 260901-u6es-fab-agent-yaml-skill-migration

## Consumers

The provider/role resolution feeds three runtime consumers:

- **The dispatch seam** (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`, and `/fab-continue`'s one-stage sequencer) calls `fab agent <stage> -o yaml` before each post-intake stage's sub-agent and **branches on `dispatch:` key presence**: absent ⇒ native Agent-tool dispatch (model via `model_alias` on the Agent `model` param, effort via the YAML `effort` value in a prompt instruction); present ⇒ the CLI adapter `fab dispatch` (the profile rides `dispatch.command`, which skills never execute themselves). Every stage it resolves is a Tier-2 role, so `agent.workers` is the knob it consults. The ship (`fast`) and review-pr (`doing`) sites — `/fab-fff` Steps 4–5 and `/fab-continue`'s ship/review-pr rows — branch on the same rule, so a resolved pane/headless mapping is executed there rather than silently degrading to the inherited session model; their delegated `/git-pr` / `/git-pr-review` workers self-manage their own stage's transitions on every arm. See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership.
- **The operator launcher** (`fab operator`) resolves the **operator** role in-process and composes its session command from that role's provider `interactive_command` + profile. Its interactive spawn carries the shell fallback (composed command + `; exec "$SHELL"` — see [agent-primitives.md](/runtime/agent-primitives.md) § Spawn composition). See [operator.md](/runtime/operator.md).
- **Batch worker spawns** (`fab batch new`/`switch` and the operator's repo-targeted worker spawns) compose from the **default**-role provider `interactive_command` + profile — so workers spawn WITH a profile. As interactive spawns, they carry the same shell fallback, so the tab survives the agent's exit as a shell in the same cwd (see [agent-primitives.md](/runtime/agent-primitives.md) § Spawn composition). See [operator.md](/runtime/operator.md) and [distribution/kit-architecture.md](/distribution/kit-architecture.md).

The latter two are Tier-1 roles, so `agent.session` is what governs them — and it binds at **launch**: a running session keeps the provider it started on.

[`fab dispatch`](/runtime/dispatch.md) runs the command selected by the current ladder; `fab agent -o yaml` resolves and emits the structured selection. `fab dispatch start` re-resolves the ladder so pane reachability is current at launch time.
