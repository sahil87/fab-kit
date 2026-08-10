---
type: memory
description: "The providers & agent-roles model — `agent.session`/`agent.workers` depth knobs, per-session `FAB_AGENT_SESSION`/`FAB_AGENT_WORKERS` overrides and `--workers` launch sugar; the `providers:` table as pure capability grammar (`interactive_command`, `native`, `headless_command`) with per-role fills; six fixed roles, fill precedence, stage→role mapping, `fab resolve-agent`, `fab agent`, the `dispatch.mode` preference descent, and consumers."
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

Tier 1/Tier 2 name depth and nothing else. A pane worker is still Tier 2: the defining property is "owes a result artifact and owns no transitions" ([dispatch.md](/runtime/dispatch.md)), not "never spoken to".

This file is the model — the depth knobs, the four built-in providers, the six roles and the fixed partition, the fill precedence, the stage→role mapping, the `fab resolve-agent`/`fab agent` surfaces, and who consumes the resolution. The **config-schema authority** is [_shared/configuration.md](/_shared/configuration.md) § `providers` and § `agent`; the **dispatch-seam wiring** is [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution; the pre-implementation design intent is `docs/specs/stage-models.md` (drift-guarded against the Go maps).

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
| **Tier 2** (workers) | `doing`, `review`, `hydrate`, `fast` | `agent.workers` | **every stage dispatch** (`fab resolve-agent`) |

The split is mechanically real rather than cosmetic, which is what makes depth the right knob axis. `intake` rides `default` and is therefore a session role for exactly that reason — it runs foreground in the user's own session.

A knob supplies only the **provider** rung. Model and effort always come from the resolved provider's own per-role fills, so naming a provider is a complete configuration.

#### Scenario: one knob re-points every dispatched stage

- **GIVEN** `agent: { workers: agy }` and no other agent keys
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `provider=agy` (apply → `doing`, a workers role)
- **AND** `fab resolve-agent operator` still emits `provider=claude` (a session role)

#### Scenario: no `agent:` block at all

- **GIVEN** a config with no `agent:` block
- **WHEN** any role or stage is resolved
- **THEN** the provider is `claude` for every role, on fab-kit's shipped per-role models

### Requirement: Per-session provider selection — environment and launch flags

`FAB_AGENT_SESSION` and `FAB_AGENT_WORKERS` SHALL provide process-tree-local overrides for the two depth knobs. The current process and its descendants resolve these variables above project and system config, so two shell sessions in separate worktrees can select different worker providers without changing committed configuration. The variables are instances of the generic registry-derived environment mechanism documented in [_shared/configuration.md](/_shared/configuration.md) § Override Cascade & Scope Enforcement; both are honored because their registry rows have `scope: both`.

- **`FAB_AGENT_SESSION=<provider>`** selects the Tier-1 provider for commands launched by that process tree.
- **`FAB_AGENT_WORKERS=<provider>`** selects the Tier-2 provider used by `fab resolve-agent` and by `fab dispatch start` when it re-resolves a stage.
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

- **`interactive_command`** — **pure launch grammar**: how to open an interactive agent session, and nothing more. It makes the provider pane-capable when tmux is available. Prompt delivery is fab's, not the command's — no code path appends a prompt argument to it ([dispatch.md](/runtime/dispatch.md) § Prompt delivery), so pane capability does not depend on whether the CLI accepts a positional initial prompt.
- **`native`** — declares native Agent-tool capability. Provider names are opaque, so fab never infers it.
- **`headless_command`** — runs one headless stage task via `fab dispatch`; the prompt is supplied on stdin.

Each provider MAY additionally carry **`profiles`** — a map keyed by role name, each value `{model, effort}`: *"when this provider plays this role, use this model/effort."* It supplies the command's `{model}`/`{effort}` placeholders (§ Fill precedence). A provider's **`profiles.default` doubles as its cross-role fallback** — a role absent from the map resolves the `default` entry, then empty. The map merges per **role** and then per **field** over the built-in table, exactly as the command fields do, and `providers` is `scope: both`, so a fill is settable once per machine.

Capability fields are never merged or substituted for one another. Their presence says **how** a provider can run, never **which** mode is selected. `dispatch.mode` supplies that preference and the shared selector descends pane → native → headless, starting at the configured rung and never ascending.

**fab-kit ships FOUR built-in providers — `claude`, `codex`, `agy`, and `kimi`** (`defaultProviders` in `internal/agent`, parsed from the embedded `defaults.yaml` — § The built-in defaults are an embedded `defaults.yaml`; each command string is additionally exposed as a package var, `DefaultInteractiveCommand` / `DefaultCodex*` / `DefaultAgyHeadlessCommand` / `DefaultKimi*`, that `internal/configref` interpolates, so no literal is duplicated):

| Built-in | `interactive_command` | `native` | `headless_command` | `profiles` |
|----------|-------------------|----------|--------------------|------------|
| `claude` | templated default | `true` | `claude -p --dangerously-skip-permissions --model {model} --effort {effort}` | all six roles |
| `codex` | `codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` | absent | `codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}` | sparse — `default`, `doing`, `review`, `fast` |
| `agy` | **none** → dispatch-only (§ The dispatch-only built-in) | absent | `sh -c 'agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'` | sparse — `default`, `fast` (model only) |
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

The non-claude built-ins run **full-auto**, because a stage worker is unattended and has no approval-answering channel: codex carries `--dangerously-bypass-approvals-and-sandbox` on both forms, agy carries `--dangerously-skip-permissions`, and kimi carries `--auto` on its **interactive** command alone — its dispatch form takes **no approval flag at all**, because `kimi -p` is already non-interactive, auto-approves tool calls, and *rejects* `--yolo`/`--auto`. kimi is the one built-in whose full-auto posture is asymmetric across its two commands, and the asymmetry is the CLI's, not a choice.

**Prompt delivery differs per CLI, and the grammar absorbs the difference.** `fab dispatch` always pipes the stage prompt to the command's **stdin**. codex reads stdin directly via its `exec` subcommand, as does claude's headless command through `-p`. agy and kimi do not — their `-p` takes the prompt as an **argument** and ignores stdin — so both dispatch commands nest a shell: `sh -c '<cli> … -p "$(cat)"'`. The nesting is load-bearing, not stylistic (§ Design Decisions → "Nested-Shell `$(cat)` Idiom"). agy's command additionally raises `--print-timeout` to `120m`, since that CLI's 5-minute default would kill a long stage worker.

**The dispatch-only built-in.** `agy` alone carries **no `interactive_command`**, the mirror image of claude carrying no `headless_command`. What an `interactive_command` confers is eligibility for **pane-mode dispatch**, where fab types a one-line pointer to the stage prompt file into the pane and verifies that it landed — so what decides the field is not the CLI's argument grammar but its **first-run behavior and input echo**. agy fails on the first: it gates a fresh workspace behind an interactive trust prompt even under `--dangerously-skip-permissions`, and worktree-per-change makes every dispatch a fresh workspace, so a pane worker parks before it can be delivered to. Backlog `[agik]` owns that probe and agy's roster flip. With no `interactive_command`, automatic selection skips the pane rung and descends to headless (`descended: pane unavailable: no interactive_command` — [dispatch.md](/runtime/dispatch.md)) and `fab dispatch open` hard-errors with the `providers.agy.interactive_command` hint. A user who wants an interactive agy session may add one in their own config.

**kimi is pane-capable**, shipping `kimi --auto -m {model}` alongside its headless command, because both halves of that same question were answered against kimi 0.34.0 (2026-08-10):

- **First run**: kimi gates a fresh folder behind a `Trust this folder?` wall, which is an ordinary readiness-gate **judgment round** — the pane reads `parked`, one Enter clears it, and the answer is remembered per folder, so it amortizes across every later pane worker in that checkout ([dispatch.md](/runtime/dispatch.md) § `fab dispatch ready`). It needs no code.
- **Input echo**: kimi draws vertical rules down both sides of its input box, so a hard-wrapped pointer arrives with `││` interleaved between the halves. `deliver`'s echo verification counts occurrences with box-drawing runes dropped alongside whitespace, so the pointer verifies ([dispatch.md](/runtime/dispatch.md) § `fab dispatch deliver`).

kimi has no interactive-initial-prompt flag — its `-p` is the non-interactive form, and a bare positional parses as a subcommand — which costs it nothing: delivery is post-open in every case, so pane capability never depended on that flag.

`fab config explain` presents all four as **live, uniformly-rendered built-in defaults** — every provider line carries exactly one leading `#` prefix in a fence or `init --system` scaffold (no doubled `# #` marker; an inline `# ...` note on a command line stays content), and the segment prose warns that hoisting a whole block PINS its fills against kit-release refreshes; see [configuration.md](/_shared/configuration.md) § `providers` for the rendered presentation and its parse-side guarantees, and § The machinery is documented, not scaffolded for why the managed fence omits the block.

**Provider names are opaque — fab NEVER infers a provider from a model string.** Two entries fronting the same vendor under different names are different providers, each with its own fills.

#### Scenario: the preference selects among independent capabilities

- **GIVEN** built-in claude and `dispatch.mode: pane`
- **WHEN** tmux and its `interactive_command` are available
- **THEN** `fab resolve-agent` emits `dispatch=` with the interactive command
- **WHEN** tmux is unavailable
- **THEN** the selector descends to claude's `native: true` and omits `dispatch=`
- **AND** `dispatch.mode: headless` starts at headless and emits claude's substituted `headless_command`, even though native capability also exists

#### Scenario: `kimi` resolves an empty model and drops the flag pair

- **GIVEN** `agent.workers: kimi` and no `providers.kimi` block
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `provider=kimi`, the `model=` line is empty, and `dispatch=sh -c 'kimi -p "$(cat)"'` — the `-m {model}` pair dropped as a unit, the quoted `"$(cat)"` segment left syntactically intact

#### Scenario: `kimi` is eligible for a pane worker and a session

- **GIVEN** `agent.workers: kimi`, `dispatch.mode: pane`, a reachable tmux server, and no `providers.kimi` block
- **WHEN** `fab dispatch open` runs for a stage
- **THEN** the pane opens on `kimi --auto` — the empty `{model}` dropping the `-m` pair — and the readiness gate answers the first-run trust wall before `deliver` types the pointer
- **AND** `fab agent --provider kimi --print` prints that same composed command instead of erroring on a missing `interactive_command`

#### Scenario: the dispatch-only built-in inside tmux descends to headless

- **GIVEN** `agent.workers: agy` and a reachable tmux server
- **WHEN** `fab dispatch start` runs in auto mode
- **THEN** automatic selection skips the pane rung and lands on headless, reporting `mode: headless (descended: pane unavailable: no interactive_command)` — no pane worker is spawned and nothing orphans
- **AND** `fab dispatch open` instead exits non-zero naming `providers.agy.interactive_command`, persisting no dispatch record

#### Scenario: naming a built-in provider with no `providers:` block

- **GIVEN** a config with `agent: { profiles: { review: { provider: codex, model: <codex-model-id>, effort: high } } }` and no `providers:` block at all
- **WHEN** `fab resolve-agent review` runs
- **THEN** it emits `provider=codex` plus the command selected by the current `dispatch.mode`; with the default `native` preference, codex descends to its headless capability
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
- **WHEN** `fab resolve-agent apply` (role `doing`) runs
- **THEN** `effort=high`, and the model comes from the resolved provider's `doing` fill — **not** from `agent.profiles.default.model`

### Requirement: The built-in defaults are an embedded `defaults.yaml`

The built-in tables — the depth knobs, the four-provider table with every provider's per-role fills, and the `dispatch:` block carrying the three built-in dispatch defaults (`mode`/`column_width`/`reap_done`) — are **data**, and live in `src/go/fab/internal/agent/defaults.yaml`, shaped exactly as a **user config-file fragment**: the same `agent:` / `providers:` / `dispatch:` keys a project writes to override them. The file is the **single value source for every built-in default** — no Go source carries an independent copy; the `config.DefaultDispatch*` symbols are package-level vars the package's own `init()` assigns from the parsed `dispatch:` block (the import cycle forbids `internal/config` reading them back — see [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Dispatch Defaults Are Init-Injected from `defaults.yaml`"). The file is compiled into the binary via `//go:embed` and unmarshalled **once at package initialization** into `config.Config` — the same struct `config.LoadPath` fills from a user's `config.yaml` — so the built-in shape and the config schema cannot diverge. A malformed file **panics at init** (compiled-in bytes make a parse failure a defective build artifact, not a runtime condition), and nothing is read from the kit cache at runtime, so resolution cannot break on a missing or corrupt cache. `defaults_test.go` is the file's safety net against a YAML typo: it asserts the parse, exhaustive role/provider coverage with non-empty fields, the per-provider command-field presence/absence (including that agy carries **no** `interactive_command` while kimi's is pinned **by value** — `kimi --auto -m {model}`, the probed invocation, so an edit to `--yolo` or to a pinned-model form cannot reach a pane worker unnoticed), that every **fill-carrying** provider has a non-empty `profiles.default.model` while kimi's empty fill map is asserted as deliberate rather than skipped, that agy's fills set no `effort`, that no built-in uses the deprecated flat fill spelling, that the `dispatch:` block pins the three dispatch defaults and their injection wiring, and that the file defines no keys outside the `agent:`/`providers:`/`dispatch:` surface.

**The YAML/Go split is the overridable/fixed boundary.** What lives in `defaults.yaml` is user-overridable by writing the same key in `fab/project/config.yaml` (or `~/.fab-kit/config.yaml`); what stays in Go is fab-owned policy — `stageRoles` (the fixed stage→role mapping) and `roleDepth` (the role→depth partition). The config-fragment shape is also what makes `defaults.yaml` the **physical source of the cascade's defaults tier** (see [_shared/configuration.md](/_shared/configuration.md) § Override Cascade): consumers still reach those defaults at the existing point-of-use seams rather than through `LoadPath`'s merge, while the read-model surfaces merge them as a real tier 0 through the registry projection `configref.DefaultsMap` — so folding the file into the loader's own merge would be a merge-order change, not a parser change.

**A model bump edits `defaults.yaml`** and runs the tests. The file carries the bump procedure inline and names the drift guards that will name *themselves* if a mirror falls behind — `TestDefaultsFileIsWellFormed` (the file itself), `TestDefaultRoleProfilesArePinned` (the deliberate-change pin for claude) and `TestNonClaudeProviderFillsArePinned` (the same pin for codex/agy, plus kimi's deliberate empty map), `TestDocTablesMatchAgentMaps` and `TestMirrorDocsMatchDefaultProfiles` (`docs/specs/stage-models.md`'s table and its inline-YAML sample, which is checked per **enclosing provider** so all four providers' blocks are guarded and agy's effort-less rows still match), `TestConfigReferenceDocumentsProviderFill` (the codex/agy fill lines in the rendered `fab config explain`), `TestCLIFabReferenceListsDefaultRoles` (`_cli-fab.md`'s enumeration). Every other consumer derives from `DefaultProfile()` or `ResolveProvider`, so a bump touches nothing else.

#### Scenario: the defaults travel inside the binary

- **GIVEN** a `fab` binary run from a directory with no fab-kit cache present
- **WHEN** `fab resolve-agent apply` runs
- **THEN** it resolves the built-in profile from the embedded bytes — no filesystem read of the kit cache occurs, and no on-disk state can make resolution fail

### Requirement: Fill precedence — one chain, provider-anchored

Resolution SHALL follow one precedence chain, implemented **once** in `internal/agent` (`ResolveRoleWith`, which `cmd/fab` never re-implements):

**Provider**: invocation `--provider` → `agent.profiles.<role>.provider` → the role's depth knob (`agent.session` or `agent.workers`) → built-in `claude`.

**Model / effort, per field**: invocation flag → `agent.profiles.<role>.<field>` → `providers.<p>.profiles.<role>.<field>` → `providers.<p>.profiles.default.<field>` → empty.

`<p>` is the **resolved** provider (post-override), which is what makes a provider swap safe: model and effort are re-derived from the new provider's own fills rather than carried from the old one, so **nothing is ever inherited across providers in the first place**. An explicit `agent.profiles.<role>.model` still survives a swap — a pin the user wrote is the user's own escape hatch, not inheritance.

There is exactly **one** cross-role fallback and it lives on the **provider** side (`providers.<p>.profiles.default`). The agent side has none.

An empty value keeps its established meaning: `spawn.WithProfile`'s token-drop (so the CLI's own default applies) and, on the `model=` line, "inherit the session/orchestrator model". The depth knob is applied only when `--provider` was **not** supplied, so an explicitly-empty `--provider=` resolves an empty provider — a lookup failure for the caller to report — rather than falling through to the knob.

#### Scenario: a provider swap refills from that provider

- **GIVEN** `agent.workers: codex` and `providers.codex.profiles: { review: { model: <codex-model-id>, effort: high } }`
- **WHEN** `fab resolve-agent review` runs
- **THEN** `provider=codex`, `model=<codex-model-id>`, `effort=high`
- **AND** the other workers roles keep codex's **shipped** fills (the override merges per role, then per field)
- **AND** with no `providers:` block at all, every workers role still resolves codex's own shipped fill — never claude's values

#### Scenario: a provider carrying no fills resolves empty

- **GIVEN** `agent.workers: mine` and a project-defined `providers.mine` carrying commands but no `profiles` — the same shape the built-in `kimi` ships deliberately
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `model=` is empty and no `effort=` line is emitted, with both placeholder tokens dropped from the composed command so the CLI's own default applies
- **AND** an agy-resolved role emits no `effort=` line either — its fills are model-only, and the grammar carries no `{effort}` for one to land in

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

### Requirement: `fab resolve-agent <stage|role> [--alias] [--provider <name>] [--model <id>] [--effort <level>]` resolution surface

`fab resolve-agent` SHALL accept a **stage** name OR a **role** name positionally — a stage maps through the fixed mapping, a role resolves directly, and **role names are checked first** (`agent.RoleForName`). The two name sets overlap only at **fixed points**: a name shared by a stage and a role (`review`, `hydrate`) is one where the stage maps to that same-named role (`stageRoles[name] == name`), so the role-first check resolves such a name to the identical profile either interpretation would (`ship` is a stage but NOT a role — it maps to `fast` — so `resolve-agent ship` and `resolve-agent fast` both resolve to the `fast` profile, one via the stage mapping, the other directly). It resolves the role → `{provider, model, effort}` per § Fill precedence and emits, **verbatim, with NO validation**:

- `model=<id>` (always; empty = the inherit signal),
- `effort=<level>` (omitted when empty),
- `provider=<name>` (omitted when empty),
- `dispatch=<command>` — absent exactly when the shared mode selector lands on native; present with the substituted `interactive_command` for pane or `headless_command` for headless. Selection starts at `dispatch.mode`, descends pane → native → headless without ascending, and fails only when no reachable capability exists. The command always embeds the full model ID even under `--alias`.

`--alias` maps the `model=` line to the Claude-Code Agent-tool short alias (`opus`/`sonnet`/`haiku`/`fable`) — the Agent tool's `model` param is a hard enum that rejects full IDs; the `dispatch=` line is unaffected (full ID).

**Invocation-time overrides** are the fill precedence's top rung, applied by re-running the chain with the flags on top (`agent.ResolveRoleWith`) rather than patching a resolved profile:

- **`--provider <name>`** swaps the provider, refills model/effort from that provider, and re-runs the same `dispatch.mode` ladder against its capabilities. The emitted `dispatch=` presence may therefore differ from the unoverridden profile.
- **`--model <id>` / `--effort <level>`** override the corresponding field and are valid **without** `--provider` — a within-role override of the profile this pure query would otherwise print. This is a deliberate, documented **asymmetry with `fab agent`**, where `--model`/`--effort` remain usage errors without `--provider` (a session launcher with two mutually exclusive addressing modes, where a bare `--model` would invent an undocumented override surface).
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

### Requirement: `dispatch.mode` — preference-bounded capability descent

`dispatch.mode` is `pane`, `native`, or `headless`, defaults to `native`, and has scope `both`. The project layer overrides the machine-wide system preference. It is a **ceiling**: selection starts at that rung and moves only downward through pane → native → headless.

- Pane is possible when tmux is available and the provider has `interactive_command`.
- Native is possible when the provider declares `native: true`.
- Headless is possible when the provider has `headless_command`.
- The first possible rung wins. Selection fails only when no rung at or below the preference is possible.
- Invalid values warn and fall back to `native`. The retired boolean has no read-time alias.
- `fab resolve-agent` and `fab dispatch start|restart` use the same pure selector; start/restart supply the real tmux reachability result before writing any state.

#### Scenario: native preference descends for a non-native provider

- **GIVEN** `dispatch.mode: native` and a role resolving to built-in codex
- **WHEN** `fab resolve-agent <stage>` runs
- **THEN** native is skipped and `dispatch=` carries codex's headless command

#### Scenario: pane preference never ascends

- **GIVEN** `dispatch.mode: pane`, no reachable tmux, and built-in claude
- **WHEN** the role is resolved
- **THEN** the selector descends to native and omits `dispatch=`; it does not jump directly to headless while native is possible

#### Scenario: headless preference ignores higher capabilities

- **GIVEN** `dispatch.mode: headless` and built-in claude, which also supports pane and native
- **WHEN** the role is resolved
- **THEN** `dispatch=` carries claude's headless command because the selector never ascends

### Requirement: `fab agent [role] [--provider <name> [--model <id>] [--effort <level>]] [--workers <provider>] [--print] [--repo <path>]` — session launcher

`fab agent` SHALL compose an interactive session command in one of **two mutually exclusive addressing modes** and **exec it in the current shell**:

- **Role-addressed** (the `[role]` positional) — resolve a role profile through the full chain (`default` when the role is omitted; any of the six role names accepted) and compose `providers.<profile.provider>.interactive_command` with the role's `{model}`/`{effort}`. Since `default` and `operator` are session roles, this is the path `agent.session` governs: `fab agent` starts the default-role agent right here, `fab agent operator` starts the coordinator profile.
- **Provider-addressed** (`--provider <name>`) — **bypass role resolution entirely**: look up `providers.<name>` directly via `agent.ResolveProvider` (project config per-field-merged over the built-in table, exactly as the role path's provider lookup does) and compose its `interactive_command` with the `--model`/`--effort` values. This is the "give me a codex session right here" form — no role need name the provider first.

Both modes compose through the same `spawn.WithProfile` (template substitution or Claude-style flag append — see [configuration.md](/_shared/configuration.md) § `providers`) and share `--print`/`--repo`:

- `--print` prints the fully-resolved command instead of executing — the output is **profile-resolved** (model/effort substituted), so callers that spawn from the printed command get the profile.
- `--repo <path>` reads the target repo's config (the operator's fetch-another-repo's-command use case). Composes with either mode.
- `--workers <provider>` sets `FAB_AGENT_WORKERS=<provider>` in the environment passed to the exec seam without changing either addressing mode's command resolution. An entry inherited from the parent environment is removed rather than shadowed, so the override is authoritative regardless of how a consumer resolves duplicate entries. It is accepted with `--print`, but printed output remains the command alone because no child process is executed.
- `fab agent` exec does NOT TTY-guard — exec-and-let-the-CLI-fail is acceptable (the underlying agent CLI already handles no-TTY), matching the document-don't-validate contract.

Provider-mode rules:

- **Omitted `--model`/`--effort`** leave the value empty and follow `WithProfile`'s empty-value rule (template mode drops the placeholder's token plus a preceding `-`-flag; append mode omits the flag), so a profile-free provider invocation results and the installed CLI's own default model applies — the way to spawn a provider whose current model IDs the caller does not know. Provider-level flags without placeholders remain in that command. **This path bypasses the provider's per-role fills too**: `providers.<name>.profiles` is deliberately NOT consulted by `fab agent --provider` (the bypass is total — it skips role resolution *and* the fill), so an empty flag means empty, not the configured fill. The fills apply on the resolution surface (`fab resolve-agent`, role resolution), not on this launcher.
- **`--model`/`--effort` require `--provider`** — supplying either alone is a usage error (non-zero exit).
- **`--provider` and the `[role]` positional are mutually exclusive** — supplying both is a usage error naming the exclusion (a hand-written `RunE` check, since cobra's `MarkFlagsMutuallyExclusive` relates only flags and the role is a positional).
- Both guards, and the mode selection itself, key on cobra's `Flag.Changed` — whether the flag was **supplied** — not on its value being non-empty, so `fab agent doing --provider=` and `fab agent --model= --print` still error rather than falling through to the role path.
- **An unknown provider name** is a non-zero-exit **lookup** failure listing the available names (`agent.ProviderNames`: fab-kit's built-in table ∪ the project's `providers:` keys via `config.ProviderNames`, sorted). Listing resolvable *names* is not validation of a command's *content* — resolved strings still pass through verbatim.
- A provider that resolves but carries no `interactive_command` yields the `configure providers.<name>.interactive_command` hint error on either path.

The procedural knowledge for *using* a composed command — opening it in a tmux window, delivering a prompt, peeking, awaiting — plus the per-provider invocation grammar and model-discovery recipes live in the `_cli-agents` helper: see [agent-primitives.md](/runtime/agent-primitives.md).

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
- **WHEN** `fab resolve-agent apply` runs
- **THEN** `effort=medium` — the legacy key still resolves
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
**Decision**: See the authoritative record in [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Providers Extracted; Roles; `review_tools` → `code-review.md`". In brief: a top-level `providers:` table carries independent session/native/headless capabilities plus fills; roles resolve `{provider, model, effort}`; dispatch preference stays separate; review policy moved to `code-review.md`; `fab agent` owns session launch; and `resolve-agent` exposes `dispatch=` plus `provider=`.
**Why**: Conflating provider mechanics with role/budget policy actively confused (two fields both named `spawn_command`; the `thinking` name hid its referent, which was review). Extraction plus role naming attack the confusion at its source; commands living on the provider make `{provider, model, effort}` composition safe (no cross-semantics command inheritance).
**Rejected**: Merging the two command fields; folding a command in as a `default`-role field (implies the rejected cross-fallback); keeping `thinking`; provider inference from model strings; a `fab spawn-command` deprecation alias.
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### Positional Stage-or-Role; `provider=` Line; No TTY Guard
**Decision**: `fab resolve-agent` accepts a stage OR role name positionally (role names checked first; shared names are fixed points), carries a `provider=` line, aliases only `model=`, and leaves `dispatch=` commands on the full model ID. `fab agent` exec does not TTY-guard.
**Why**: Reuse the existing positional surface rather than add flag surface for no disambiguation benefit; surface the provider rather than re-derive it downstream; keep the no-validation/document-don't-guard contract for TTY.
**Rejected**: A `--role` flag (surface for no benefit); inferring provider downstream (re-does resolution); a TTY guard (the agent CLI already handles no-TTY).
*Introduced by*: 260702-tykw-agent-providers-role-tiers

### `hydrate` Is Its Own Role; `fast` Keeps Its Role Name (Multi-Referent)
**Decision**: `hydrate` is its own role rather than part of `doing`, giving six roles (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`). A role is stage-named only where it maps 1:1 to a single referent (`review`, `hydrate`); `default`, `doing`, and `fast` keep role names because each is multi-referent — `fast` in particular governs both the ship stage and the `/fab-proceed` prefix-step dispatches.
**Why**: Memory writing (hydrate) is knowledge work with a different cognitive profile than apply's diff work, so it deserves its own model/effort — grouped under `doing` it could never run cheaper or on a different model than apply. A stage name (`ship`) would misname `fast` once it also governs the prefix steps.
**Rejected**: Naming the role `ship` (misnames a multi-referent role and would force an unnecessary carry-forward migration); six stage-named roles / dissolving roles entirely (`default`/`doing`/`fast` are genuinely multi-referent and the role names carry the why); a user-overridable stage→role mapping or per-stage escape hatch (taxonomy stays fab-owned).
*Introduced by*: 260719-g55d-stage-model-tier-defaults-v2

### `--provider` Is a Sibling Addressing Mode, Not a Role Override
**Decision**: `fab agent --provider <name>` bypasses role resolution entirely — a direct `providers.<name>` lookup with `--model`/`--effort` as the profile — rather than synthesizing an ad-hoc role or overriding a named role's fields. `--model`/`--effort` are usage errors without `--provider`, and `--provider` is mutually exclusive with the `[role]` positional. All three guards key on cobra's `Flag.Changed` (was the flag supplied?) rather than value emptiness.
**Why**: A role is a fab-owned slot resolved through the fill precedence; provider-addressed spawning answers a different, mechanical question ("give me a codex session"), and a resolved role already names a provider, so mixing them has no coherent semantics. Bypass leaves the resolution path untouched, so no existing path changes behavior. On the role path the profile IS the role's, so a bare `--model` would either invent an undocumented override surface or be silently ignored — an explicit usage error is the only honest option and is trivially relaxable later. Emptiness-based guards would let `--provider=` and `--model=` fall silently through to the role path, so supplied-ness is the correct test.
**Rejected**: A role-provider override flag (mutates role/budget policy to express a mechanics question); auto-creating a synthetic role (invents state the config never declared); letting `--model` override a resolved role's model on this launcher (a second, undocumented override surface); silently ignoring the flags (surprise-inducing CLI behavior); cobra's `MarkFlagsMutuallyExclusive` for the role pairing (it relates flags only — the role is a positional).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

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
**Decision**: Both codex command forms carry `--dangerously-bypass-approvals-and-sandbox` and agy's headless command carries `--dangerously-skip-permissions`; kimi carries `--auto` on its interactive command and no approval flag on its headless one, because `kimi -p` is already non-interactive, auto-approves tool calls, and *rejects* `--yolo`/`--auto`. Project and system provider-command overrides remain the approval-gated escape hatch.
**Why**: Headless and pane stage workers are unattended and have no approval-answering channel; ship and review-pr also require network and repository operations. Explicit bypass grammar gives the non-claude built-ins the same autonomous execution policy as claude's shipped `--dangerously-skip-permissions` command — expressed in each CLI's own vocabulary, including "no flag" where the CLI's headless mode already implies it.
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

### agy Is Dispatch-Only Until Its Interactive First Run Is Probed
**Decision**: The agy built-in carries a `headless_command` and deliberately no `interactive_command`, the mirror image of claude carrying no `headless_command`. What gates its roster flip is a **probe of first-run behavior and input echo**, owned by backlog `[agik]`.
**Why**: An `interactive_command` is what makes a provider **pane-eligible**, and a pane worker is delivered to by typing — so what matters is whether the CLI can be *reached at the keyboard* on a fresh workspace, not whether it accepts a positional prompt. agy fails that: it gates a fresh workspace behind an interactive trust prompt even under `--dangerously-skip-permissions`, and worktree-per-change makes every dispatch a fresh workspace, so a pane worker parks before it can be delivered to. Withholding the field makes automatic resolution skip the pane rung and descend to headless (`descended: pane unavailable: no interactive_command`) and makes `fab dispatch open` fail loudly with the config-key hint, which is the honest state until the probe says otherwise.
**Rejected**: Shipping the interactive grammar ahead of the probe and documenting the hazard (a parked worker is a per-stage stall, exactly what the descent exists to prevent); adding a per-provider "pane-ineligible" config flag (the absent `interactive_command` already encodes it, and the no-cross-fallback rule already reads it correctly); a provider-specific trust-store pre-seed in Go (undocumented-format provider machinery inside a provider-neutral binary — see [dispatch.md](/runtime/dispatch.md) § First-run walls).
**Known consequence**: `fab agent --provider agy` and `fab operator` on agy need an `interactive_command` the user adds themselves — which also re-enables pane eligibility for it ahead of the probe.
*Introduced by*: 260808-rpsr-remove-gemini-add-agy-kimi; *Updated by*: 260809-3oz7-pane-readiness-gate-sendkeys-delivery, 260810-ki9v-kimi-pane-enablement

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

### Overrides Land on `resolve-agent`, Not New Dispatch Machinery — and Bind the Native Arm Only
**Decision**: Invocation-time provider/model/effort overrides are flags on `fab resolve-agent`, the single resolution call every dispatch site already makes, and the skill wiring is one passthrough paragraph. On that surface `--model`/`--effort` are valid without `--provider` (a within-role override), while `fab agent` keeps its requires-`--provider` rule. Because `fab dispatch start` carries no override surface — it re-resolves the stage from config — the overrides bind the **native Agent-tool arm** only, and every doc restating the surface carries that scope.
**Why**: Every dispatch site already makes exactly one `resolve-agent --alias` call and already branches on `dispatch=`, so overriding at that call reuses the whole seam with zero new machinery; a separate override channel would need its own precedence rules and its own compliance-visibility contract. `resolve-agent` is a pure query whose whole output is a profile, so overriding one field of it is unambiguous and useful; `fab agent` is a session launcher with two mutually exclusive addressing modes. Scoping to the native arm keeps the docs describing what the code does instead of a workflow that errors on the headless arm and silently runs the wrong provider on the pane arm.
**Rejected**: Plumbing the flags through `fab dispatch start` (a second resolution surface with its own precedence and visibility contract); per-stage override config keys (persistent state for a per-run intent); relaxing `fab agent` to match (re-litigates a deliberate shipped decision); forbidding bare `--model` on `resolve-agent` for symmetry (a usage error for an unambiguous query).
*Introduced by*: 260805-j3cm-builtin-provider-templates-and-fill

### The Built-in Tables Are an Embedded Data File, Parsed Through the Config Schema
**Decision**: The provider grammars, claude's per-role fills, the depth knobs, and the `dispatch.*` built-in defaults live in `src/go/fab/internal/agent/defaults.yaml`, embedded with `//go:embed` and unmarshalled at package init into `config.Config` — the struct the user-config loader fills — with a malformed file panicking. `stageRoles` and `roleDepth` stay Go map literals, making the YAML/Go split the overridable/fixed boundary.
**Why**: The data is deeply nested (per-role fills per provider), and as Go literals it is verbose to review and unreadable to a non-Go reader doing a model bump. A data file also removes the built-ins↔`fab config explain` drift class **by construction** rather than by drift-guard test: the reference renders from the same bytes the resolver parses, so there is no Go-map→YAML translation left to drift. Parsing through the config schema is the only reading that keeps the file's config-fragment shape from silently diverging from the schema it imitates, and it makes the eventual layer-0 unification a merge-order change rather than a parser change. Embedding rather than reading `$(fab kit-path)` at runtime keeps the "resolution cannot fail" property: kit and binary release atomically, so an on-disk read buys nothing and adds a binary↔kit skew failure mode. Panicking is the honest response to compiled-in bytes that do not parse — a defective build artifact, not a state a released binary can reach.
**Rejected**: Reading the file from the kit cache at runtime (binary↔kit version skew on a path that cannot fail today); a bespoke defaults struct in `internal/agent` (a second schema definition to keep in sync) or hand-walking a `map[string]any` (loses the schema's yaml tags and zero-value semantics); returning an error from the accessors (widens the API of a behavior-neutral change for an unreachable state) or falling back to hardcoded Go values (restores the duplicate literals); `sync.Once`-on-first-use (defers a build defect to an arbitrary call site — there is no I/O to make lazy); moving `stageRoles`/`roleDepth` into the file too (taxonomy is fab-owned policy, and the split is what signals which half a user may override); placing the file under `src/kit/` (that tree deploys to the kit cache and would imply runtime reading).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0; *Updated by*: 260809-wll4-config-source-consolidation

### The Built-in Command Identifiers Are Package Vars Sourced from the Embedded File
**Decision**: `DefaultInteractiveCommand` and the non-claude `DefaultCodex*`/`DefaultAgyHeadlessCommand`/`DefaultKimiHeadlessCommand` identifiers are package `var`s reading the parsed provider entries, not `const` literals; `spawn.DefaultSpawnCommand`, their only compile-time-constant consumer, is a `var` re-export of the same value. The identifiers are kept rather than replaced by `ResolveProvider(nil, name)` calls. The `config.DefaultDispatch*` symbols follow the same pattern — package-level vars carrying no literals, assigned from the parsed `dispatch:` block by agent's `init()` (the import cycle makes push-from-agent the only allowed direction — see [_shared/configuration.md](/_shared/configuration.md) § Design Decisions → "Dispatch Defaults Are Init-Injected from `defaults.yaml`").
**Why**: `defaults.yaml` owns these strings — keeping them as Go literals would put every command text in two places and reintroduce by hand exactly the drift the data file removes by construction. Keeping the *names* leaves `internal/configref`, `internal/configupgrade`, and every test call site untouched.
**Rejected**: Keeping the constants canonical and having `defaults.yaml` restate them (drift-by-test, the thing being removed); a test asserting const↔YAML equality (the same drift, one indirection later); deleting the identifiers and rewriting every consumer (inflates a behavior-neutral diff).
*Introduced by*: 260806-2j2i-embed-agent-defaults-layer0; *Updated by*: 260809-wll4-config-source-consolidation

### `DefaultProfile` Is Resolution Against a Nil Config
**Decision**: `agent.DefaultProfile(role)` is defined as `ResolveRole(nil, role)` rather than a lookup in a separate built-in role→profile table.
**Why**: There is no single built-in role→profile map to read any more — the values live under `providers.claude.profiles`, reached through the same chain everything else uses — and nil-config resolution is exactly "the built-in answer". It also keeps `fab config explain` sourcing its per-role defaults from the resolver rather than from a second table that could drift.
**Rejected**: A parallel built-in profile map (a second table to keep in sync with the resolver, which is the drift class the embedded data file exists to remove).
**Consequence — the registry row is the STATIC default; the read model composes a live one**: `configref.Fields()`'s `agent.profiles` default is built from `DefaultProfile`, which is knob-blind by construction. So `fab config show --origin` recomposes that one row through `configref.DefaultsMapFor`, resolving each role against the **live** config (`agent.ResolveRole`) with the user's own `agent.profiles`/`agent.tiers` entries stripped first — the defaults tier reports the built-in a user override *shadows*, never an echo of that override. The drill-down and `fab resolve-agent` therefore agree on which provider a role would dispatch to, both reading the knob. See [_shared/configuration.md](/_shared/configuration.md) § Six-Verb Surface.
*Introduced by*: 260806-j9nh-agent-profiles-session-workers

### Unknown Provider Is a Lookup Failure That Names the Resolvable Set
**Decision**: An unknown `--provider` name exits non-zero listing the available provider names — the sorted union of fab-kit's built-in table and the project's `providers:` keys, exposed as `agent.ProviderNames(cfg)` over the nil-safe `config.ProviderNames()` accessor. The message is produced by a single `unknownProviderError(cfg, name)` helper in `cmd/fab`, shared verbatim by both flag-accepting commands (`fab agent`'s provider-addressed mode and `fab resolve-agent --provider`).
**Why**: The union is exactly the set `ResolveProvider` will accept, so it is the only set whose listing is actionable; sorting makes the message stable for tests and for readers. Naming resolvable *names* is not validation of command *content* — resolved command strings still pass through verbatim, preserving document-don't-validate (fab never infers a provider from a model string). One helper keeps the two commands' errors byte-identical, so a caller learns one phrasing.
**Rejected**: Listing only the project's configured providers (omits the built-in `claude`, which resolves fine); a bare "unknown provider" error (leaves the caller to guess the config surface); validating the resolved command string (breaks the document-don't-validate contract); a per-command copy of the formatter (two phrasings of one contract to drift).
*Introduced by*: 260805-nvad-cli-agents-helper-provider-spawn

## Consumers

The provider/role resolution feeds three runtime consumers:

- **The dispatch seam** (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`, and `/fab-continue`'s one-stage sequencer) calls `fab resolve-agent <stage> --alias` before each post-intake stage's sub-agent and **branches on the resolved `dispatch=` line**: absent ⇒ native Agent-tool dispatch (model via the Agent `model` param, effort via a prompt instruction); present ⇒ the CLI adapter `fab dispatch` (the profile rides the `dispatch=` command). Every stage it resolves is a Tier-2 role, so `agent.workers` is the knob it consults. See [_shared/context-loading.md](/_shared/context-loading.md) § Per-Stage Model Resolution and [pipeline/execution-skills.md](/pipeline/execution-skills.md) § Status-transition ownership.
- **The operator launcher** (`fab operator`) resolves the **operator** role in-process and composes its session command from that role's provider `interactive_command` + profile. See [operator.md](/runtime/operator.md).
- **Batch worker spawns** (`fab batch new`/`switch` and the operator's repo-targeted worker spawns) compose from the **default**-role provider `interactive_command` + profile — so workers spawn WITH a profile. See [operator.md](/runtime/operator.md) and [distribution/kit-architecture.md](/distribution/kit-architecture.md).

The latter two are Tier-1 roles, so `agent.session` is what governs them — and it binds at **launch**: a running session keeps the provider it started on.

[`fab dispatch`](/runtime/dispatch.md) runs the command selected by the current ladder; this file and `fab resolve-agent` only resolve and emit it. `fab dispatch start` re-resolves the ladder so pane reachability is current at launch time.
