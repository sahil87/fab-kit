# Per-Stage Model Selection via Agent Roles

> **Status:** Design intent (pre-implementation, now implemented — 260613-l3ja, reshaped in
> 260806-j9nh). This spec captures the design for letting a project run different pipeline stages
> on different models. The canonical data is `src/go/fab/defaults.yaml` (the embedded
> depth knobs and CLI table, including claude's per-role fills) plus the `stageRoles` and
> `roleDepth` maps in `src/go/fab/internal/agent/agent.go`; the two tables in this doc are verified
> mirrors of them (drift-guarded — see § Drift guard).

Fab runs a six-stage pipeline (`intake → apply → review → hydrate → ship → review-pr`). Without this
feature every stage runs on whatever model the session was launched with — the orchestrator's
foreground model, or the model a dispatched sub-agent inherits. This feature lets a **project**
assign a CLI and model to each stage, so judgment-heavy stages (review) run at a high-end model
+ effort while the mechanical stage (ship) runs cheaper.

The control surface is deliberately small, and it is **two knobs**: pick a CLI for the agents
you talk to, and a CLI for the agents the pipeline dispatches. Everything beneath that — which
stages cluster into which role, which model each role gets on each CLI — is fab-owned default
or a sparse escape hatch.

---

## Vocabulary

Four words, each with exactly one meaning:

| Term | Meaning |
|------|---------|
| **role** | One of six fixed slot names — `default`, `operator`, `doing`, `review`, `hydrate`, `fast`. A role is *what an agent is for*. |
| **profile** | A concrete `{cli, model, effort}` value. What a role resolves *to*. |
| **CLI** | A named set of independent launch capabilities (`interactive_command`, `headless_command`, and `native`) plus its per-role fills. Capability presence says how a rung runs; `dispatch.mode` chooses the preference ceiling and descends `pane → native → headless`. |
| **Tier 1 / Tier 2** | Agent **depth**: Tier 1 is the agents a user talks to (a `fab agent` session, the operator, a `fab batch` worker); Tier 2 is the agents pipeline stages dispatch to. |

"Tier" means depth and *only* depth. A watchable pane worker is still Tier 2 — the defining property
is "owes a result artifact and owns no transitions", not "never spoken to". (Ship/review-pr workers
excepted on the transition half — they self-manage their own stage's transitions; see
[`harness-adapters.md`](harness-adapters.md) § Dispatch-prompt obligations.)

---

## Why this is possible now

The pipeline already dispatches most post-intake stages as **sub-agents** (see `_preamble.md`
§ Subagent Dispatch). The move to sub-agents was driven by context isolation — a six-stage pipeline
whose automatic run spans five stages (review-pr is entered manually) cannot fit in one context
window, so each stage runs in a fresh context and returns a
structured result (the exceptions — a *continued* native apply worker across rework cycles, and the
`/fab-ff`/`/fab-fff` **light lane**, which runs non-review stages inline in the orchestrator's context
with no dispatch at all (`_pipeline.md` § Light Lane) — change nothing here: the continued worker's
profile is still injected at the stage's first dispatch, and an inline stage never leaves the session
model). That same dispatch seam is the natural injection point
for a per-stage model: the orchestrator sets the sub-agent's model **at dispatch time**.

This makes per-stage model selection fundamentally a property of **dispatched sub-agent runs**. Since
260613-fgxx collapsed the post-intake dual execution mode, **every** post-intake stage dispatches a
sub-agent — including plain `/fab-continue`, which is now a one-stage sequencer that resolves the role
and dispatches its stage's block (`/fab-ff`, `/fab-fff`, `/fab-proceed` orchestrate the same way).
The one carve-out is the light lane above: review stays dispatched in both lanes, so per-stage
selection applies uniformly to apply/review/hydrate regardless of which command drove them, wherever
a dispatch exists.
See § Foreground limitation for the narrow case (a stage skill run with no dispatch at all) it cannot
cover.

---

## The two knobs — `agent.session` and `agent.workers`

The advertised config surface is exactly two scalars, each naming a CLI:

```yaml
agent:
  session: claude    # Tier 1 — what you talk to: fab agent, rk operator, fab batch
  workers: codex     # Tier 2 — what pipeline stages dispatch to
```

Both default to `claude`, so a fresh install writes nothing. The **role → depth partition** is fixed
and fab-owned:

| Depth | Knob | Roles |
|-------|------|-------|
| Tier 1 (session) | `agent.session` | `default`, `operator` |
| Tier 2 (workers) | `agent.workers` | `doing`, `review`, `hydrate`, `fast` |

**The split is mechanically real, not cosmetic.** `agent.session` applies at **launch time** — fab
cannot switch a running session's CLI — while `agent.workers` applies at **every stage dispatch**
(`fab agent <stage> -o yaml`). `intake` rides `default` and is therefore a session role for exactly that
reason: it runs foreground in the user's own session.

Both keys are scope `both`, so "claude for what I talk to, codex for the workers" is settable once
per machine in `~/.fab-kit/config.yaml`.

---

## Roles are the referent taxonomy; profiles are what they resolve to

A profile is a **named triple of `{cli, model, effort}`** — not a bare model. The invocation
**command** does NOT live on the role: it lives on the **CLI** (see [§ CLIs](#CLIs)),
so a role is pure budget/referent policy. Effort is a first-class spend dial, and what a user means by
a role is the CLI, the model, *and* how hard it thinks.

Six **roles** form the vocabulary — concrete referents ("the operator", "the reviewer"), not
cognitive modes. A role is **stage-named only where it maps 1:1 to a single referent** (`review`,
`hydrate`); `default`, `doing`, and `fast` keep role names because each is **multi-referent** (`fast`
governs the ship stage *and* the `/fab-proceed` prefix-step dispatches — see § Skill wiring):

| Role | Referent |
|------|----------|
| `default` | Spawned worker sessions (`fab batch`), `fab agent` with no role, intake (advisory only — foreground), and the `/fab-proceed` create-intake dispatch. *(`fab agent --cli <name>` deliberately resolves NO role — it is a sibling addressing mode that bypasses this taxonomy entirely; see § CLIs.)* |
| `operator` | The operator coordinator session (`rk operator`). |
| `doing` | **Execution that must not err** — apply writes the diff; review-pr fixes already-articulated feedback. |
| `review` | **The critic** — review reads a diff and discovers what's wrong. Its own role (not folded into `doing`) so the critic's model and effort can be dialed independently of the author's; the separation that matters is the fresh context and adversarial framing, not a different model family. |
| `hydrate` | **Memory writing** — hydrate merges the change into `docs/memory/` as current truth. Its own role so it can run on a different model/effort than apply. |
| `fast` | **Speed on near-mechanical work** — ship's commit/push/PR mechanics plus a faithful PR-description summary, and the `/fab-proceed` prefix steps (`/fab-switch`, `/git-branch`). Multi-referent, so it keeps its role name. |

There is no `thinking` role: with `review` its own role, `thinking`'s only remaining stage would be
intake, which never dispatches (it is pre-boundary, foreground). Intake rides `default`, honestly — it
runs wherever the interactive session runs.

### Default role profiles

fab-kit ships a default `{cli, model, effort}` per role. The CLI comes from the built-in
depth knobs (both `claude`); the model and effort come from **`clis.claude.profiles.<role>`**.
Both live in **`src/go/fab/defaults.yaml`** — a data file **embedded into the binary**
(`go:embed`, never read from the kit cache at runtime) and shaped as a config-file fragment, so it is
versioned with the kit. The Claude models are the family **aliases** `opus` and `sonnet`, not
versioned IDs: `claude --model` and the Agent tool's `model` parameter both take them and resolve the
newest model of the family, so the defaults track the installed Claude Code and a new Claude model
needs no kit release. A project that wants a specific model pins it with a full ID
(`clis.claude.profiles.<role>.model: claude-opus-5-5`); one that wants the installed tool's own
default without pinning an ID uses the reserved model value `tool-default`.

| Role | CLI | Model | Effort |
|------|----------|-------|--------|
| `default` | `claude` | `opus` | `high` |
| `operator` | `claude` | `sonnet` | `medium` |
| `doing` | `claude` | `sonnet` | `high` |
| `review` | `claude` | `opus` | `high` |
| `hydrate` | `claude` | `sonnet` | `high` |
| `fast` | `claude` | `sonnet` | `medium` |

This is the verified mirror of what `defaults.yaml` composes. A drift-guard test fails if the two
disagree (see § Drift guard).

**Why these defaults.** `doing` runs Sonnet at `high` — apply is the highest-output-volume stage, and
the kit default accepts the apply↔review coupling risk (see § apply↔review coupling) in exchange for
Sonnet-rate apply; a project that wants an Opus author sets `agent.profiles.doing`. `review` runs
Opus/`high` — code review is a named Opus strength, and review is now the only Opus stage downstream of intake:
the critic carries the judgment a cheaper author may miss (author/critic separation is enforced by the
*fresh context and adversarial framing*, not by model choice). `hydrate` runs Sonnet/`high` — it writes
memory from a finished, reviewed diff, which is summarization well within Sonnet's range. Note
that on the native dispatch arm the effort half is advisory anyway (§ Effort asymmetry), so the model
choice is what actually carries. `default` runs
Opus/`high` — the role covers interactive sessions and advisory intake scoring, where the user is
talking to the model directly and the intake's judgment calls gate the whole pipeline, and
`high` is the recommended effort sweet spot (`xhigh` buys marginal gains at disproportionate
latency/cost). `operator` runs
Sonnet/`medium` (highest-volume coordinator, pattern-matching work, escalation discipline makes the
cheaper model safe). `fast` sits at the mechanical floor on Sonnet/`medium` — effort at `medium` (not
`low`) buys margin for faithful PR-description comprehension. The explicit effort is
load-bearing in both directions, not decorative, because fab passes `--effort` on every CLI arm and never
relies on a model's own default, and an alias can move to a model whose default differs. Today Opus 5.5
(what `opus` resolves to) defaults to `medium`, so the explicit `high` is what keeps the Opus roles at
`high`; Sonnet 5.5 defaults to `high`, so the explicit `medium` is what keeps `operator` and `fast` at
`medium`. `doing`/`hydrate`'s `high` equals Sonnet 5.5's default today and stays explicit anyway — the
next Sonnet may default differently. Cost-conscious projects opt
any role down themselves (see § Config schema).

---

## The fixed stage → role mapping (fab-owned, NOT overridable)

fab owns which stage belongs to which role. The mapping is **fixed and non-overridable** — it is fab's
considered judgment from a dimensional analysis (judgment density, cost-of-error, output volume,
determinism). Users override what a role *costs* (budget), never which stages belong to it (taxonomy).

| Stage | Role |
|-------|------|
| `intake` | `default` |
| `review` | `review` |
| `apply` | `doing` |
| `review-pr` | `doing` |
| `hydrate` | `hydrate` |
| `ship` | `fast` |

This is the verified mirror of the `stageRoles` map in `src/go/fab/internal/agent/agent.go`
(drift-guarded). The mapping is exhaustive — every one of the six pipeline stages belongs to exactly
one role. `intake → default` is **advisory only**: intake runs foreground in the user's own session,
which fab cannot re-model (see § Foreground limitation). Where a stage and a role share a name
(`review`, `hydrate`), the stage maps to that same-named role — see § Resolution's fixed-point rule.
(`ship` is a stage but not a role — it maps to the multi-referent `fast` role.)

**Critical distinction — `review` vs `review-pr`.** They share the word "review" but not the role.
`review` is **the critic** (reads a diff and discovers what's wrong from nothing → its own `review`
role); `review-pr` is **responsive** (triages and fixes feedback someone else already generated →
`doing`). They are deliberately in **different roles** — do not group them.

There is **no `stage_roles` config** (stage→role reassignment is not a user knob), **no per-stage
escape hatch** (a stage cannot be pinned individually outside its role), and **no user-overridable
role→depth partition**. Disagreement with the taxonomy is an upstream fab-kit issue, not a project
knob.

---

## CLIs

The launch **capability grammar** lives in a top-level `clis:` table, not on the roles. Each
CLI is an opaque, user-chosen name mapping to three independent dispatch capabilities plus a per-role fill map:

- **`interactive_command`** — opens an interactive agent **session** (`rk operator` / `fab batch` /
  `fab agent`, and the pane-dispatch adapter's worker). It is **pure launch grammar**: fab appends
  nothing to it, and a pane worker's stage prompt is delivered afterwards by `fab dispatch deliver`
  (see [`harness-adapters.md`](harness-adapters.md) § 3), so a CLI's eligibility for pane dispatch
  does not depend on whether its CLI accepts a positional initial prompt. It is reachable **two ways**: through a role (the role names a CLI, and its
  `{model, effort}` are substituted) or **directly**, via
  `fab agent --cli <name> [--model <id>] [--effort <level>]`, which bypasses role resolution
  entirely — a CLI-addressed spawn for the "give me a codex session right here" case, where no
  role need name the CLI first. The launcher's positional also accepts a **stage** name (mapped
  through the fixed `stageRoles` table to its role, exactly as `fab agent <stage> -o yaml` resolves), and combines
  with `--cli` to re-resolve that role's fills from the named CLI (`-t` prints the raw template,
  `--headless` picks `headless_command`, and `-o yaml` carries the full structured resolution, including
  raw template, fill mode, per-field provenance, model alias, and labelled non-native dispatch — all
  print-mode sinks). The direct form is a **lookup**, not a new validation surface: an
  unknown name errors listing the available CLIs, while resolved command strings still pass
  through verbatim. See `_cli-fab.md` § fab agent.
- **`headless_command`** — runs ONE headless **stage task** via `fab dispatch`.
- **`native`** — boolean capability for the in-harness Agent-tool adapter. CLI names are opaque,
  so fab never infers native support from a name or model.
- **`profiles.<role>`** — the CLI's **per-role fill**: `{model, effort}` for "when this CLI
  plays this role". Keyed by role name; the `default` entry doubles as the CLI's **cross-role
  fallback**. Scope `both`, so a machine-wide fill is settable once in `~/.fab-kit/config.yaml`, where
  it outranks the project file.

The capabilities are deliberately independent. Session and headless dispatch are different
invocations of the same binary (claude interactive `-n` vs headless `-p`; codex TUI vs `codex exec`),
and native is not a command at all. Presence describes **how** a rung runs; it never chooses policy.
`dispatch.mode` owns preference, and no command field substitutes for another.

**Per-role fills, not one fill per CLI.** A single `{model, effort}` per CLI would resolve
the *same* model for every role, so swapping `agent.workers` to another CLI would flatten the
whole role taxonomy. Keyed by role, the swap keeps the differentiation: `apply` gets that CLI's
strongest model, `ship` its cheapest.

### Built-in CLIs

fab-kit ships **four built-in CLIs** — `claude` (the default), `codex`, `agy` and `kimi` — in the
`clis:` block of the module-root embedded `defaults.yaml`.

The sample below is **shape-only**: it shows the command grammars and the per-CLI fill SHAPE
(which roles each map carries, and whether its rows are model+effort or model-only), with the model
IDs elided. This spec states shipped model IDs in exactly one place — the four-column table
in § Default role profiles, which covers `claude` and is drift-guarded. For the live values of every
CLI, read `src/go/fab/defaults.yaml` (canonical), or render them:

- `fab config explain CLIs --json` — projects every CLI's fill map, resolved.
- `fab agent <stage|role> -o yaml` — the structured resolution a stage (or a role named directly) actually gets.

(`fab config explain CLIs` prints every CLI's fill map in its human-readable form too,
claude's included.)

```yaml
clis:
  claude:
    native: true
    interactive_command: 'claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
    headless_command: 'claude -p --permission-mode bypassPermissions --model {model} --effort {effort}'
    profiles:                                 # dense — all six roles carry model + effort
      default:  { model: ..., effort: ... }
      operator: { model: ..., effort: ... }
      doing:    { model: ..., effort: ... }
      review:   { model: ..., effort: ... }
      hydrate:  { model: ..., effort: ... }
      fast:     { model: ..., effort: ... }
  codex:
    interactive_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
    headless_command: 'codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
    profiles:                                 # dense — every role pins its own model (no inheritance from `default`)
      default:  { model: ..., effort: ... }
      doing:    { model: ..., effort: ... }
      review:   { model: ..., effort: ... }   # a DIFFERENT model from doing — the critic never shares the author's blind spots
      hydrate:  { model: ..., effort: ... }
      operator: { model: ..., effort: ... }
      fast:     { model: ..., effort: ... }
  agy:
    interactive_command: 'agy --dangerously-skip-permissions --model {model}'
    headless_command: 'sh -c ''agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'''
    profiles:                                 # model-only: the reasoning level rides the ID suffix
      default: { model: ... }
      fast:    { model: ... }
  kimi:
    interactive_command: 'kimi --auto -m {model}'
    headless_command: 'sh -c ''kimi -m {model} -p "$(cat)"'''
    # NO profiles: kimi's -m takes a user-config model alias, not a catalog ID.
```

**Three of the four carry fills.** claude's and codex's maps are DENSE — every role carries its own
model and effort, so no shipped row leans on the cross-role fallback. agy's map is SPARSE: a role
absent from it resolves agy's `default` entry, so agy's non-`fast` roles land on agy's `default`
model without a row of their own. The per-FIELD merge rule still exists (a user override carrying
`effort` only takes its model from the CLI's `default`), but no shipped codex row relies on it.

**Why these codex fills** (role-relative; `fab config explain CLIs` prints the live IDs). `default`
and `doing` take the catalog's top model at `high` — `default` is the judgment-heavy session role, `doing`
the execution that must not err. `review` deliberately runs a **different** model from `doing`, at `xhigh`
(the highest effort fab ships on any codex role): a critic on the author's own model shares the author's blind spots, and
the 2026-08-10 four-CLI comparison (`docs/findings/`) recorded reviewer strictness tracking the
worker model on every arm. `hydrate` takes that second model at `high` — sweep-heavy memory prose where
it performed well. `operator` and `fast` take the cheap model. The map is dense so a `default` bump
never silently repoints another role, which is exactly what the earlier effort-only `doing`/`review`
rows did. Two effort rules: codex's `ultra` level ("maximum reasoning with automatic task delegation")
is **never shipped** — a worker spawning its own sub-agents breaks the single-worker dispatch contract
and the `{stage}-result.yaml` protocol — and `max` is an explicit-override level only. Known
limitation: per-model codex pricing was not available when the table was set, so the tier ladder is
**inferred** from codex's own model descriptions and priority order, not measured.

**All four are pane-capable.** Each built-in ships an `interactive_command`; agy's exact grammar is
`agy --dangerously-skip-permissions --model {model}`. A fresh agy workspace can park at an interactive
trust prompt even under the bypass flag. That wall is an ordinary readiness-gate judgment round, and
the answer is remembered for the exact workspace path. An operator may pre-seed that exact path in
`trustedWorkspaces` in `~/.gemini/antigravity-cli/settings.json`; fab does not write the CLI's
trust store. Backlog `[agik]` retains the live open → ready → deliver verification after quota reset,
while the command grammar and readiness-gate contract are fixture-pinned.

**kimi's equivalent probe closed** (2026-08-10, kimi 0.34.0), which is why it ships one. Its two
unknowns were answered rather than designed around: the first-run `Trust this folder?` wall is an
ordinary readiness-gate **judgment round** (`parked` → one Enter → ready, remembered per folder, so it
amortizes across a checkout), and its **input echo** verifies now that delivery's echo check ignores
box-drawing runes as well as whitespace — kimi draws vertical side rules down its input box, so a
wrapped pointer arrives in the capture with `││` interleaved between the halves.

**Full-auto posture, per FORM.** Claude uses `--permission-mode bypassPermissions` on both its forms,
codex `--dangerously-bypass-approvals-and-sandbox` on both its forms, and agy
`--dangerously-skip-permissions` on both forms. kimi carries `--auto` on its INTERACTIVE
form only: `kimi -p` is already non-interactive and auto-approves tool calls, and *errors* when
combined with `--yolo`/`--auto` (`Cannot combine --prompt with --yolo`). Pipeline workers are
unattended and have no channel for answering approval prompts. Users who require approval-gated
workers override the corresponding `clis.<name>.interactive_command` or `.headless_command`.

Three per-CLI shapes are load-bearing:

- **agy's fills carry no `effort`** — its model IDs *embed* the reasoning level as an ID suffix
  (`gemini-3.1-pro-high`, `gemini-3.6-flash-low`), which is the same reason its command grammars carry
  no `{effort}` placeholder: a separate `--effort` flag would fight the suffix. An agy-resolved role
  emits no `effort=` line.
- **kimi ships NO fills, deliberately.** Its `-m` takes a **user-config model alias** (managed installs
  expose `kimi-code`/`k3`; custom-CLI installs differ), not a vendor catalog ID — so any value
  fab-kit pinned would be wrong for some installs. The empty `{model}` triggers the token-drop
  (§ Fill precedence), removing `-m` and its placeholder as a pair, and kimi runs on the user's own
  configured `default_model`. A user who wants role differentiation pins
  `clis.kimi.profiles.<role>.model` themselves. This is the one built-in for which an EMPTY
  resolution is correct rather than a gap.
- **agy's and kimi's `headless_command`s nest a shell.** Both CLIs take the prompt as the ARGUMENT to
  `-p` and ignore stdin, while `fab dispatch` delivers the prompt on stdin. POSIX expands `$(cat)`
  *before* the redirect applies, so only the nested form — `sh -c '<cli> … -p "$(cat)"'` — puts the
  redirected prompt on the inner shell's stdin. The placeholder is always INTERIOR to that quoted
  string (the `-p "$(cat)"` tail follows it), which is what keeps the token-drop from taking the
  closing quote with it.

Consequences:

- **Naming any built-in resolves with zero `clis:` config** — `agent.workers: codex`, an
  `agent.profiles.<role>.cli`, or `fab agent --cli codex` / the deprecated `fab resolve-agent <stage>
  --cli codex` all work on a fresh project. A `clis:` block is for *overriding* a grammar or
  a fill, not for registering these CLIs.
- **Role differentiation survives a CLI swap.** `agent.workers: codex` runs apply on codex's top
  catalog model, review on a *different* model at `xhigh`, and ship on
  the cheaper `fast` model at `low` — which is the whole point of keying fills by role. Shipping no
  fills resolved an *empty* model identically for all four, silently flattening the taxonomy.
- **A built-in CLI is inert until named.** Adding the rows changes no default behavior (both
  depth knobs ship `claude`), which is why presence=intent — the rule that keeps behavior-changing
  config commented — does not force the table out of Go.
- **Claude carries all three capabilities; codex, agy, and kimi carry both command fields and are
  non-native.** Under the default `dispatch.mode: native`, claude
  resolves native while the non-claude built-ins descend to headless. Adding a command never
  changes policy by itself.
- The reference renders all four blocks **live and uniformly** — in a fence or
  `init --system` scaffold every CLI line carries exactly one leading `#` prefix (no
  doubled `# #` marker; an inline `# ...` note stays content), and
  presence=intent is untouched (a built-in stays inert until named). Their command strings render as YAML
  single-quoted scalars with interior quotes doubled, so uncommenting a nested-shell
  `headless_command` yields valid YAML. Hoisting a whole block PINS its fills
  against kit-release refreshes — the segment prose warns to override a single
  field (`clis.<name>.profiles.<role>.model`) instead.

### Refreshing the non-claude fills

Built-in non-claude fills are **refreshed at kit-release cadence**, by editing `defaults.yaml`. The
claude fills need no refresh: they are the family aliases `opus`/`sonnet`, which the installed
`claude` resolves to the newest model of the family. codex and agy have no such alias — their model
flags take a concrete catalog slug — so their fills are concrete IDs. There
is no staleness automation: fab runs no CI check against CLI APIs and no model-catalog fetch —
resolution passes every string through **verbatim** (the no-validation contract; compatibility is the
harness's concern).

That is safe because of the failure shape on each side:

- **A stale ID fails loudly and cheaply.** The CLI CLI rejects it immediately, and the fix is one
  config line: `clis.<name>.profiles.<role>.model`. Because `clis` is scope `both`, that
  line is settable once machine-wide in `~/.fab-kit/config.yaml`, where it outranks the project file.
- **Shipping nothing failed silently.** An empty model resolved the CLI's own default identically for
  every role, with no error at all — worse than a loud stale ID, because it defeats role
  differentiation exactly where a user first exercises the knob.
- **A bump is a data diff.** The fills are rows in an embedded YAML file, reviewable at a glance, and
  pinned by `TestNonClaudeCLIFillsArePinned` so the edit is always test-acknowledged.

Verify a fill against the **installed** binary rather than from memory — `_cli-agents.md` § discovery
recipes records what to run per CLI. The fills are never seeded into a user's `config.yaml`: they live
in the binary, so an upgrade refreshes them and no project pins rot in place.

> **Decision lineage — `260731-ho9y` → `260805-j3cm` → `260806-ywkx` → `260908-wcib`.** ho9y shipped the non-claude CLIs as
> *uncomment-to-opt-in template text* and recorded "no new built-in CLIs are added in Go". j3cm
> reversed that narrowly, for **grammar strings only**, explicitly keeping model IDs out ("non-claude
> model IDs rot at CLI cadence, so they belong in config, not in a release"). ywkx completes the
> reversal: fills ship too. What survives from ho9y is its actual rule — presence implies intent, so
> anything whose presence changes behavior ships commented — and it is *preserved*, because a built-in
> CLI is inert until a knob, role override, or flag names it. What is retired is the rot argument,
> on the four grounds above plus release cadence: fab-kit ships every few days, so users see refreshed
> suggestions at kit cadence rather than CLI cadence. wcib is the first catalog refresh under the ywkx
> policy: the codex fills followed the installed CLI's catalog (verified from its cached model list) and
> the map went dense so a `default` bump can never again repoint another role silently.

**CLI names are opaque — fab NEVER infers a CLI from a model string** (`claude-*` → claude
would need a CLI registry, which the no-validation/CLI-neutrality contract refuses).

### Model discovery — `fab agent models`

`fab agent models [cli] [--json] [--repo <path>]` asks each tool what it offers and always merges
in the resolved CLI's profile fills. Rows carry `{cli, id, label, efforts, default, source}`;
`source` distinguishes discovered `tool` rows, built-in `alias` rows, `fill` rows, and the `reserved`
row. Every CLI starts with `{id: "tool-default", source: "reserved", default: false}`: its label
is `tool default (<default-id>)` when a default is known and plain `tool default` otherwise, and its
efforts come from the default row when non-empty, else Claude's known set for Claude, else `[]`. The
real default keeps `default: true`; a tool-reported row with the reserved ID is dropped. Subsequent
tool rows retain tool order, fills are sorted and deduplicated by ID with precedence
`tool > alias > fill`, and an unlisted tool default is appended.

Codex discovery runs the bounded debug command `codex debug models` and reads `$CODEX_HOME/config.toml`;
Claude reads its settings and local override under `$CLAUDE_CONFIG_DIR`; Kimi reads model tables from
`~/.kimi-code/config.toml`; agy returns fills only. Optional `clis.<name>.models_command` replaces
the built-in adapter with a command that prints the same row array. Discovery is read-only, uncached,
and fails open to the reserved row plus fills with stderr warnings rather than blocking a caller.

### Fill precedence

```
cli:       invocation --cli
                >  agent.profiles.<role>.cli
                >  the role's depth knob (agent.session | agent.workers)
                >  the built-in claude

model / effort  invocation flag
(per field):    >  agent.profiles.<role>.<field>
                >  clis.<p>.profiles.<role>.<field>
                >  clis.<p>.profiles.default.<field>
                >  empty
```

**The preset rung.** An active agent preset (see [§ Agent presets](#agent-presets--agentpreset-and-agent_presets))
adds fields to the `agent.*` tier — it is not a rung of its own. The preset's fields enter **at the config
layer that chose it, directly below that layer's own explicit `agent.*` fields**, so in effect:

```
agent.* tier, per field:   explicit field in a layer ABOVE the choosing layer
                            >  explicit field in the choosing layer
                            >  the preset's field  (enters here)
                            >  explicit field in a layer BELOW the choosing layer
```

Invocation flags stay on top of everything. A preset's `profiles.<role>.cli` lands in
`agent.profiles.<role>.cli` and its `workers`/`session` land in the depth knobs, so the chain above is
unchanged — and a CLI a preset names **re-derives model/effort from that CLI's own fills** exactly as an
`agent.profiles.<role>.cli` override does (a preset that sets only `cli` never inherits the old CLI's model).

`<p>` is the **resolved** CLI, so a CLI swap re-derives model and effort from the new
CLI's own fills. `empty` keeps its existing meaning: an empty YAML `model` value is the "inherit the
session model" signal, and on a command the placeholder's token (plus a preceding `-`-flag) is dropped
by `spawn.WithProfile`, so the CLI's own default applies. The reserved model value `tool-default` is an
ordinary non-empty value through this chain, so it wins over lower tiers; after the winning rung (an
invocation flag included) is known, it collapses to empty. YAML therefore reports `model: ""` and
`model_alias: ""` while `source.model` retains the key that held the token. Native inherits the session
model, and CLI composition drops the model flag pair exactly as for any other empty resolved model.

**There is exactly ONE cross-role fallback chain, and it lives on the CLI side.** `agent.profiles`
is sparse: an unset field is simply not an override, and `agent.profiles.default` is the `default`
**role's** own override — never a fallback source for the other five. Cross-role fallback is
`clis.<p>.profiles.default`.

**Why one chain.** Two competing chains are what made a cross-CLI *cutoff rule* necessary in the
first place: with agent-side inheritance, `{cli: codex}` on a role would inherit a **claude**
model through the same per-field merge, and fab had to detect and cut that. With model/effort always
sourced from the resolved CLI's own fills, **no value can be foreign to the CLI that will run
it**, so the cutoff rule, its per-field ownership tracking, and its cross-scope limitation are all
gone.

The one value that *does* survive a CLI swap is an explicit `agent.profiles.<role>.model` — a pin
the user wrote by hand. That is not inheritance; it is the user's own escape hatch, and removing it on
a swap would make the field unusable.

> **Retired with the cutoff: the cross-scope ownership limitation.** The cutoff computed per-field
> ownership over the *merged* config, so it could not fire across the system↔project scope boundary —
> a documented limitation pinned by `TestResolveCrossScopeCascadeLimitation`. Both the rule and the
> pin are deleted: with no ownership computation there is no scope-boundary gap to document.

## Config schema — the two knobs, `agent.profiles`, and `clis:`

All three are optional in `fab/project/config.yaml`. The Go `Config` struct widens freely — yaml
unmarshalling ignores unknown keys, so existing configs are unaffected (the same property that made
`stage_hooks` free to add).

```yaml
agent:
  session: claude          # Tier 1 — fab agent / rk operator / fab batch   (roles: default, operator)
  workers: claude          # Tier 2 — the pipeline's stage workers            (roles: doing, review, hydrate, fast)

  # The sparse per-role escape hatch. Every field optional; a set field beats
  # the knob (cli) or the cli's own fill (model/effort). NO cross-role
  # inheritance — `default` here is the default ROLE, not a fallback.
  profiles:
    review: { cli: codex }                       # e.g. run just the critic elsewhere
    fast:   { model: claude-haiku-4-5, effort: low }  # e.g. run ship cheaper

clis:
  claude:
    native: true
    interactive_command: 'claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
    headless_command: 'claude -p --permission-mode bypassPermissions --model {model} --effort {effort}'
  # codex, agy and kimi are BUILT-IN clis carrying grammar (and, for the
  # first two, fills) — `fab config explain` shows their blocks LIVE like
  # claude's, and a copy hoisted into your config PINS the shown values against
  # kit-release refreshes. Uncomment only to OVERRIDE a grammar or pin a newer
  # model; prefer a single-field override (clis.<name>.profiles.<role>.model).
  # (Shape only below; § Built-in clis above carries the live values,
  # and `fab config explain` prints what your binary actually ships.)
  # codex:
  #   interactive_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
  #   headless_command: 'codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
  #   profiles:
  #     default: { model: <codex-model-id>, effort: high }   # e.g. — pin a newer model here
  # agy:
  #   interactive_command: 'agy --dangerously-skip-permissions --model {model}'  # no {effort}: reasoning rides the model suffix
  #   headless_command: 'sh -c ''agy … --model {model} -p "$(cat)"'''   # no {effort} flag; nested shell so $(cat) reads the piped prompt
  #   profiles:
  #     default: { model: <agy-model-id> }  # e.g. — no effort: the reasoning level rides the ID suffix
  # kimi:
  #   interactive_command: 'kimi --auto -m {model}'               # --auto: the full-auto flag its headless form rejects
  #   headless_command: 'sh -c ''kimi -m {model} -p "$(cat)"'''   # no --yolo: kimi -p rejects it and already auto-approves
  #   profiles:
  #     default: { model: <your-kimi-alias> }  # e.g. — fab ships no kimi fill at all

dispatch:
  mode: native              # preference ceiling: pane → native → headless
```

- `agent.preset` (scalar, scope `both`) selects one named preset from the top-level `agent_presets:` table;
  see § Agent presets. Both are optional and unset by default.
- Keys under `agent.profiles:` are the six role names: `default`, `operator`, `doing`, `review`,
  `hydrate`, `fast`. Keys under `clis.<name>.profiles:` are the same six.
- Each `agent.profiles` value is a `{cli, model, effort}` object (the command lives on the
  CLI). Any field MAY be set; an unset field is not an override and resolution continues down the
  fill precedence.
- A role omitted entirely (or an absent `profiles:` block) uses the knob's CLI and that
  CLI's own fills.
- An **empty model** signals "inherit the session/orchestrator model" once resolution bottoms out.
- The `{model}`/`{effort}` placeholders in a CLI command are substituted at resolve time via the
  same `internal/spawn` template machinery. *This spec covers the config schema and the optional YAML
  `dispatch:` mapping; the dispatch that RUNS a `headless_command` (`fab dispatch`) and the skill
  dispatch-seam wiring share the cross-adapter contract fixed by
  [`harness-adapters.md`](harness-adapters.md).*

### Agent presets — `agent.preset` and `agent_presets`

A **preset** is a NAME for a whole role assignment: a sparse bundle of fields fab already has —
`session`, `workers`, and `profiles.<role>.{cli,model,effort}` — chosen by the scalar `agent.preset`
(env `FAB_AGENT_PRESET`, which comes from the mechanical `FAB_` + uppercased-dotted-key env layer, no
special case). It adds no role vocabulary and does not touch the stage→role mapping; it only supplies
`agent.*` fields at one extra place in the cascade.

```yaml
agent:
  preset: codex-review          # scope both: project file, system file, or FAB_AGENT_PRESET

agent_presets:                  # definitions; scope both; merge per key: built-in < project < system < env
  codex-review:
    description: Only the review role on Codex     # optional, shown by `fab agent presets`
    profiles:
      review: { cli: codex }
```

**Built-in presets** (the `agent_presets:` block of `defaults.yaml`; inert until selected, so a fresh
install resolves byte-identically):

| Preset | Sets |
|--------|------|
| `claude` | `session: claude`, `workers: claude` |
| `codex-workers` | `workers: codex` |
| `codex-review` | `profiles.review.cli: codex` (review's model/effort re-derive from `clis.codex.profiles.review`) |

**Selection and splice.** Exactly one preset is active: the value of `agent.preset` at the highest layer
(env > system > project) that sets a non-empty one — a lower layer's preset is shadowed entirely, nothing
from it is merged. Its fields are spliced in at that layer, directly below that layer's own explicit
`agent.*` fields (§ Fill precedence). The splice lives once in the config loader (the only place layers are
distinct), so `fab agent`, `fab dispatch start` (pane, native and headless), `fab batch`, the operator
launcher and `fab config show` all see the same tree. A preset chosen by `fab agent --preset <name>` is
equivalent to choosing it at the env layer for that one resolution; it is resolution-only and is not exported
into a launched session or into `fab dispatch start` (flags never reach it — use config or
`FAB_AGENT_PRESET`).

**Reserved `none`.** `agent.preset: none` (or `FAB_AGENT_PRESET=none`) wins the selection like any name
but activates nothing, so a tab or machine opts out of a lower layer's preset. A definition named `none` is
ignored with a warning and never listed. An empty `FAB_AGENT_PRESET=` is unset.

**Unknown name.** At resolve time: `fab: warning: unknown agent preset "<name>" (agent.preset from
<layer>); ignoring it` on stderr, resolved as if unset — a spawn never fails for a stale name.
`fab config set [--system] agent.preset <name>` is the one place it is an error (it rejects a name that is
neither defined nor `none`, naming the defined set). `agent_presets.<name>.<field>` dotted leaves are
settable/unsettable like `clis.<name>.<field>`.

**Visibility.** `fab agent presets [--json]` lists the defined set (`--json` is a cross-repo contract:
`{name, description, source, session?, workers?, profiles?}`, sparse, `source` = highest contributing layer
`built-in|system|project|env`, no `active` flag). `fab agent -o yaml` gains a top-level `preset:` key when
a preset is active and labels each preset-supplied `source.*` field `agent_presets.<name>`; `fab config show
--origin` attributes each preset-supplied `agent.*` leaf to the choosing layer with detail `preset <name>`.

### Advertised surface vs. full schema

The **managed fence** in every project's `config.yaml` scaffolds only the advertised fields, and on the
agent side that is exactly the two knobs (plus `dispatch.mode` / `dispatch.column_width` /
`dispatch.min_cols` / `dispatch.min_rows` / `dispatch.reap_done`).
`agent.profiles` and the whole `clis:` table are `advertise: false`: still documented in
`fab config explain` (YAML + `--json`) and in [`config.md`](config.md), but no longer ~90 commented
lines in every repo. Users pick a CLI per depth; the machinery is there when they need it.

### Migrating from `provider` (the 2.31.0 CLI rename)

The entries of the `clis:` table were called *providers* before 2.31.0. The tools themselves use
"provider" for the **model vendor** (kimi's `[providers.<vendor>]`, Claude Code routing to a model
gateway), so fab's word collided with theirs in every config file and conversation — and once
`fab agent models` started reading the tools' own configs, in fab's own code too. The word is now
**CLI** (label "CLI", prose "CLI tool"; see [glossary](glossary.md) § CLI tool). The rename is pure:
resolution results, precedence, dispatch rungs and every behaviour are identical, and a model vendor
named in a tool's own config keeps its name.

| Old | New |
|-----|-----|
| config `providers:` | `clis:` |
| `agent.profiles.<role>.provider` | `agent.profiles.<role>.cli` |
| `FAB_PROVIDERS` | `FAB_CLIS` |
| `--provider <name>` (`fab agent`, `fab resolve-agent`, `fab pane open`) | `--cli <name>` |
| `fab agent -o yaml` `provider:` / `source.provider:`; `fab pane open --json` `"provider"` | `cli:` / `source.cli:` / `"cli"` |
| `-o yaml` `kind: provider` | `kind: cli` (a scalar value cannot be dual-emitted) |

**Deprecation window: one minor release** after the release that ships the rename; the old spellings
are removed no earlier than the minor after that, in a separate change. During the window:

- **Reads are silent and per-field.** One normalizer in `internal/config` folds `providers` into `clis`
  and `.provider` into `.cli` on every raw layer map (project file, system file, environment overlay)
  before the cascade merge, the new spelling winning per leaf and an empty new leaf falling back to
  the old one; an explicit `native: false` still disables a built-in `true`. Because it runs per
  layer, a half-migrated machine (system file on `providers:`, project on `clis:`) resolves exactly as
  two same-spelling layers would, and every consumer — resolver, `show`, `explain`, setup checks,
  `fab agent models` — sees only the new shape. `FAB_PROVIDERS` is read as an alias of `FAB_CLIS`
  (`FAB_CLIS` wins per field; `EnvOrigins` names the variable actually used).
- **Writes accept the old key.** `fab config set` maps `providers.…` → `clis.…` and `….provider` →
  `….cli` *before* the value gate, writes the new key (removing a same-path old twin — for the role leaf also a live legacy `agent.tiers.<role>.provider` — so a file
  holds one spelling), and warns on stderr `<old> is deprecated, wrote <new>`; `fab config unset`
  with either spelling removes **both** at that path, the legacy tiers twin included (leaving the old one live would silently keep the
  override in effect). One key-mapping helper (`config.CanonicalKey`) serves set, unset, `explain` and keyed `show`.
  `explain providers` answers with the `clis` explanation plus `providers is the deprecated spelling of
  clis`, and keyed `show <old key>` (also `--origin`/`--json`) prints the canonical key's stdout with a
  stderr-only `fab: note:` naming the new spelling. Released HexoKit still writes the old keys, which is why `set`/`unset` accept them (2.19.0's
  `session_command` rename did not need to).
- **`fab agent --provider`** stays registered but hidden: it behaves as `--cli`, prints
  `--provider is deprecated, use --cli` on stderr, and giving both with different values is a usage
  error. `-o yaml` carries `cli:` in the slot `provider:` occupied followed by a duplicate `provider:`
  (same for `source`). `fab resolve-agent`'s frozen `provider=<name>` line is unchanged.
- **Diagnostics warn.** `fab setup check` reports each live old spelling in either config file (and
  `FAB_PROVIDERS`) as a warning naming the new key; there is no per-read warning because skills parse
  `fab agent` output. `fab config upgrade` carries a live `providers:` key to `clis:`; the
  `2.30.1-to-2.31.0` migration rewrites both config files (including the nested `.provider` leaf).

Old spellings stay in historical text (earlier migrations, memory logs, archived changes, plans).

### Migrating from `agent.tiers`

`agent.tiers` was renamed `agent.profiles`, and the flat `clis.<name>.model`/`.effort` fill folded
into `clis.<name>.profiles.default`. Both old spellings are **still read** — per role for
`agent.tiers`, and as an alias for `profiles.default` for the flat fill — so a config keeps resolving
before it is rewritten. The `2.16.19-to-2.17.0` migration performs the on-disk rewrite in both the
project config and `~/.fab-kit/config.yaml`.

One semantic that the rename does **not** preserve, and the migration cannot: the old `agent.tiers`
re-based every unset field from its `default` tier. A project that relied on `agent.tiers.default.model`
to set every tier's model must now either set `agent.session`/`agent.workers` (if the intent was a
CLI choice) or write the model on `clis.<name>.profiles.default` (if the intent was a model
choice).

One pre-migration wrinkle: the legacy alias resolves **after** the scope cascade, so while a config is
half-migrated the *spelling* can outrank the scope — a project-scope `agent.profiles.<role>` beats a
system-scope `agent.tiers.<role>` for that role, inverting the usual `system > project` precedence
(pinned by `TestResolveCrossScopeLegacyAliasPrecedence`). The migration sweeps both scopes, so the
window closes as soon as it runs; the retired-cutoff note above is about the *ownership* computation,
not this alias.

---

## Structured resolution — `fab agent <stage|role> -o yaml`

Resolution lives in **Go**, not in the prompt — the cascade is volatile logic that would drift across
skill files if reasoned about in markdown. The YAML print sink returns the concrete resolution for a
stage (or role); skills consume its keys and reason about nothing.

```
fab agent <stage|role> -o yaml [--cli <name>] [--model <id>] [--effort <level>] [--preset <name>]
```

The deprecated `fab resolve-agent <stage|role>` command remains a working, byte-stable line projection
of the same shared engine. Its `--alias` flag and ordered-line contract are preserved for compatibility;
pipeline skills consume the YAML surface above.

1. Take a **stage** name (`intake`/`apply`/`review`/`hydrate`/`ship`/`review-pr`) or a **role**
   name (`default`/`operator`/`doing`/`review`/`hydrate`/`fast`) — the positional argument accepts
   either. A stage maps through the fixed stage→role mapping; a role resolves directly (the path
   `fab agent` and the operator launcher use). **Role names are checked first.** The two name sets
   overlap only at **fixed points**: a name shared by a stage and a role (`review`, `hydrate`) is one
   where the stage maps to that same-named role (`stageRoles[name] == name`), so the role-first check
   resolves such a name to the same profile either interpretation would — the order is immaterial for
   results. (A drift-guard test in `internal/agent` asserts every collision is a fixed point.) `ship`
   is a stage but NOT a role — it maps to the `fast` role — so `fab agent ship -o yaml` resolves the stage
   while `fab agent fast -o yaml` resolves the role, both to the same profile.
2. Resolve the role → `{cli, model, effort}` through the fill precedence above: the CLI from
   an `agent.profiles.<role>.cli` override, else the role's depth knob, else the built-in claude;
   then model and effort per field from an `agent.profiles.<role>` field, else that CLI's
   `profiles.<role>` fill, else its `profiles.default` fill, else empty.
2a. **Apply invocation-time overrides** (`--cli`/`--model`/`--effort`) — the top rung, riding the
   same single resolution call. `--cli` swaps the CLI and **re-derives the optional `dispatch:`
   mapping from `dispatch.mode` plus the NAMED CLI's capabilities** — so the selected rung and key presence can differ from the
   stage's unoverridden one, but that is a **query result, not an adapter move**: `fab dispatch`
   re-resolves from config and accepts no overrides, so only a config override actually relocates a
   stage between the native and CLI adapters (§ Skill wiring → User-directed overrides). A swap
   re-derives an unoverridden model/effort from the new CLI's own per-role fills (an explicit
   `agent.profiles.<role>` pin still wins). `--model`/`--effort` are valid **without** `--cli` (a
   within-role override) — `fab agent` accepts the same bare overrides as a final post-refill layer on
   every addressing form, so the two commands agree on override legality. All three key on whether the
   flag was *supplied* (cobra's `Flag.Changed`), not on value emptiness — so an explicitly-empty
   `--cli=` resolves an empty CLI (a **lookup** failure: non-zero exit naming the resolvable
   set — built-in table ∪ the project's `clis:` keys, sorted — mirroring `fab agent`'s error)
   rather than falling back to the depth knob.
3. **Emit without validation** (see § No validation). fab does not check the CLI, model, or
   effort against any CLI's accepted set; it echoes the resolved strings as-is except for the
   exact `tool-default` collapse defined in § Fill precedence. Provenance is not cleared with it:
   `source.model` still names the rung that supplied the token.
4. Output: structured YAML carrying `cli`, the resolved `model`, `model_alias`, `effort`, composed
   `command`, `template`, `fill_mode`, per-field `source`, an optional `preset:` key (the active
   preset's name — omitted when none is active; `source.cli`/`.model`/`.effort` then read
   `agent_presets.<name>` for each field the preset itself supplied, while a model/effort re-derived
   from the preset-chosen CLI's fills keeps its `clis.<p>.profiles.<role>` source),
   an optional `dispatch:` mapping, and a
   trailing `skill_prefix` (`$` for `codex`, `/` otherwise — the receiving harness's explicit
   skill-invocation prefix).
   `model_alias` carries the native Agent-tool alias for recognized Claude IDs and is empty for
   non-Claude IDs. `dispatch:` is derived by the `dispatch.mode` descent ladder: native omits the key;
   pane/headless include a labelled `rung` and their composed `command`, plus `reason` with the
   semicolon-joined descent causes when any rung was skipped. Key presence remains the only
   skill-side branch, and skills never execute `dispatch.command`. The command's
   `{model}`/`{effort}` placeholders are substituted via
   `internal/spawn`'s template resolution (reused, not reimplemented), using the role's own resolved
   model/effort — and the dispatch command's `{model}` is **always the resolved model verbatim**,
   never translated to an alias (see § Harness-adapter boundary).
5. **Deterministic** for the same config. Non-zero exit only on a
   real error: an unreadable/malformed config, an unknown stage/role name, a supplied `--cli`
   that resolves to no CLI, or a configured CLI with no reachable dispatch capability.
   A stage that resolves to a default is success, not an error.

### Dispatch preference — `dispatch.mode`

`dispatch.mode` is a `pane | native | headless` preference ceiling (**default `native`**, **scope
`both`** — settable once machine-wide). Resolution starts at that rung and descends only through
`pane → native → headless`, selecting the first possible adapter:

- **Pane** requires tmux (`$TMUX` at the resolver seam; a real probe at `fab dispatch open`) and `interactive_command`.
- **Native** requires `native: true` and a resolved model that is empty (session-model inherit) or maps to an Agent-tool alias. A non-empty alias-less model descends with `native unavailable: model has no native alias`; the reserved `tool-default` value resolves to empty before this check, so it keeps native and inherits the session model.
- **Headless** requires `headless_command`.
- Missing prerequisites skip rungs; no selection ever ascends. Thus `pane` is the watchable preference,
  `native` is quiet/in-context, and `headless` is detached-only.
- Capability presence says how, never whether. Each mode composes only its own field, and adding
  claude's headless command does not move it off native under the default preference.
- **Skill branch:** `dispatch:` absent iff native, present for pane/headless; branch on key presence and
  never execute the value. `fab dispatch start` re-resolves internally from current config/environment.
- **Aliasing is arm-specific:** YAML `model_alias` supplies the native seam; `dispatch.command` always
  embeds the resolved model verbatim. `dispatch.reason` carries the semicolon-joined skip list whenever
  selection descended, including `native unavailable: model has no native alias`.
- If start-time re-resolution lands on native (for example tmux died after pane resolution),
  `fab dispatch` errors before writing state and tells the caller to re-run `fab agent <stage> -o yaml`.

---

## No validation — verbatim pass-through (CLI-neutral)

The shared resolution engine does **NOT** validate the model or effort against any CLI's accepted set. It
maps stage→role→`{model, effort}` and **echoes both strings verbatim**, whatever they are, with the sole
reserved-value exception: exact `tool-default` collapses to an empty model after precedence. Examples of
verbatim values include `xhigh` for an Opus model, `high` for Sonnet, `reasoning_effort`-style values for
a non-Claude model a project might configure, or an empty effort. fab has no CLI-specific knowledge
in the resolution path.

**Rationale (Constitution Principle I — CLI neutrality):** validating against Claude's effort
enum (`low/medium/high/xhigh/max`, Opus-only `xhigh`, Haiku-rejects-all) would hard-code Claude into
the resolver and bolt the door on other agents. Keeping it open — verbatim pass-through — is what lets
a project switch the underlying agent by pointing a knob at another CLI and giving it per-role
fills in that CLI's model IDs and effort vocabulary, with nothing in fab rejecting it. The
**safety net moves from fab to the runtime/harness**: a misconfigured pair (e.g. Claude
`{model: claude-sonnet-4-6, effort: xhigh}`, which Sonnet rejects with a 400) is *not* corrected by fab
— it surfaces as a dispatch-time error. This is the accepted tradeoff for portability. fab does **not**
"degrade gracefully", drop an incompatible effort, or warn on one — earlier design iterations proposed
that; it was removed.

**For reference only** (NOT enforced by fab) — Claude's effort validity, which is why the fab-kit
*defaults* are chosen to be valid:

| Model family | Accepts effort? | Valid values |
|--------------|-----------------|--------------|
| Opus / Fable | Yes | `low`, `medium`, `high`, `xhigh`, `max` |
| Sonnet | Yes | `low`, `medium`, `high`, `max` (no `xhigh` — Opus-family only) |
| Haiku | **No** | — (effort param returns HTTP 400) |

This table explains *why fab-kit's shipped defaults are what they are* — but it is documentation of
fab's default choices, not a rule the resolver enforces on user overrides.

### Haiku excluded from the defaults (not forbidden)

Haiku is **absent from the default fills**, for two reasons: it has no effort parameter (passing
effort 400s), and the one stage that might want a fast/cheap model (the `ship` stage, governed by the
`fast` role) needs faithful PR-description comprehension that Haiku does unreliably — so the `fast`
default is Sonnet/`medium`. This is **exclusion from the defaults, not a prohibition**: a user MAY
still point a role at Haiku (pass-through doesn't forbid it); fab just doesn't ship it as a default.

---

## Skill wiring — orchestrator/dispatch consume `fab agent -o yaml`

This section is the design record; the deployed wiring rules live in `_preamble.md` § Per-Stage Model Resolution and `_cli-fab.md` § fab agent (the deployed owners).

The orchestrators (`/fab-ff`, `/fab-fff`, `/fab-proceed`, `/fab-adopt`) and `/fab-continue`'s sub-agent dispatch call
`fab agent <stage> -o yaml` immediately before dispatching each stage's sub-agent, **surface** the
resolved YAML — at minimum `cli`, `model`, `model_alias`, `effort`, and `dispatch:` presence — so a skipped or mis-resolved role or CLI
dispatch is visible in output rather than silent, the available stand-in for an enforcement guard
since dispatch is harness-internal), and apply the resolved **model AND effort** through their two
seams:

- **Model → the Agent tool's `model` param.** The Agent `model` param is a hard enum of short aliases (`opus`/`sonnet`/`haiku`/`fable`) that rejects full IDs, so the native arm reads the YAML `model_alias` key. Empty alias for an empty resolved model means omit the parameter and inherit. A non-empty model with no alias never reaches this arm: resolution descends to a supported CLI rung and records why in `dispatch.reason` (see § Harness-adapter boundary).
- **Effort → an explicit instruction in the subagent prompt.** The Agent tool has no `effort` param, so the YAML `effort` value is injected as an imperative line in the dispatched prompt (e.g., ``Operate at `high` reasoning effort for this task.``) and the sub-agent self-selects. Empty effort → omit the instruction. See § Effort asymmetry for how far that actually carries.

### Effort asymmetry — the two arms are not equally reliable

**Model differentiation works on both dispatch arms. Effort differentiation does not.**

- On the **native Agent-tool arm**, effort rides an *instruction in the prompt* — there is no `effort`
  parameter to set. That instruction is **not reliably honored**: the session-level reasoning effort
  dominates, so a sub-agent dispatched from a `high`-effort session generally keeps running at the
  session's effort no matter what its prompt asks for. This is a known Claude Code limitation (GitHub
  issues #64033 and #39220), not a fab bug, and fab has no other seam to reach.
- On the **CLI arms** — `fab dispatch` headless and pane, and the operator launcher — effort rides
  `--effort` (or the CLI's own equivalent) inside a *composed command line*, which the harness
  reads as configuration rather than as a request. Effort differentiation is trustworthy there.

Practical consequence: treat a per-role `effort` as **advisory on the native arm and binding on the CLI
arms**. A project that genuinely needs a cheaper-thinking stage should reach for a cheaper *model* (or
run that stage through `fab dispatch`), not a lower effort on a natively-dispatched stage. The clean
fix — a first-class per-sub-agent `effort` parameter on the Agent tool — is a harness ask outside fab's
control (§ Foreground limitation's scope note).

### User-directed overrides

When the user directs a CLI/model for specific stages ("run review on codex"), the dispatch site
adds the override flags to its **existing single** `fab agent <stage> -o yaml` call
(§ Resolution step 2a). Nothing else about the seam changes — one resolve call per stage, the same two
seams, the same branch on `dispatch:` key presence, the same compliance-visibility obligation. There is
**no new dispatch machinery and no persistent state**: an override is per-invocation, so "use codex for
the next N stages" means the same flags on those N resolve calls. The load-bearing caveat: **an
invocation-time override binds the native Agent-tool arm only.** `fab dispatch start` takes no override
flags — it re-resolves the stage from config itself (`agent.Resolve`) — so an overridden profile never
reaches either `fab dispatch` mode, and a `dispatch:` mapping that appears *only* because of a
`--cli` swap is **not actionable**. The two remedies are **not interchangeable**. Dispatching the
stage natively with the overridden model/effort is executable only for a **within-claude**
`--model`/`--effort` override: the native adapter's model seam is the Agent tool's `model` param, a hard
**Claude-alias enum** (`opus`/`sonnet`/`haiku`/`fable` — § Harness-adapter boundary), so a non-Claude
model has no native seam to ride. For a **cross-CLI `--cli` override** the **config override**
(`agent.workers`, or `agent.profiles.<role>.cli`) that `dispatch start`'s own re-resolution will
see is therefore the **sole executable path** — the invocation flag can only report the mismatch. Sites
still re-read the resolved `dispatch:` key after an override rather than assuming the stage's unoverridden
adapter — the branch rule is unchanged — but they read it to *notice* the mismatch, not to act on it.
See [`harness-adapters.md`](harness-adapters.md) § Relationship to `stage-models.md`.

The **`review` stage resolves once** (on its own `review` role) and applies the resolved
`{cli, model, effort}` to its **single** review sub-agent — exactly like every other stage
(260704-pag2 collapsed review to one sub-agent; there is no longer a second reviewer or a merge to
resolve, so review carries no special resolution rule).

### Caller-invariant resolution — every dispatched seam resolves a role

**A stage (or dispatched step) resolves the same role regardless of which caller drives it** —
`/fab-continue`, `/fab-ff`, `/fab-fff`, or `/fab-proceed`. Two seams that formerly dispatched at the
inherited session model now resolve a role like every other:

- **`/fab-continue`'s ship and review-pr rows.** These delegate to `/git-pr` and `/git-pr-review`, and
  resolve `fab agent ship -o yaml` / `fab agent review-pr -o yaml` before
  dispatching that worker — surfacing the required YAML keys, then **branching on `dispatch:` key
  presence like every other dispatch site**: absent ⇒ the native arm applies the two seams (empty ⇒
  omit); present ⇒ the CLI adapter per `_preamble.md` § CLI-Adapter Dispatch — **mirroring
  `/fab-fff`'s full-lane ship step exactly** (`/fab-fff` terminates at ship — review-pr is dispatched
  only via `/fab-continue` or run manually with `/git-pr-review`; in the light lane those steps run
  inline with no YAML resolution). This closes the caller asymmetry where `/fab-fff` resolved a role
  for ship/review-pr but plain `/fab-continue` did not. `/git-pr` and `/git-pr-review` self-manage their own
  `fab status` transitions on every arm — their dispatch prompts carry the result-file and
  terminal-refresh obligations but not the block-contract transition prohibition (the carve-out is
  owned by [`harness-adapters.md`](harness-adapters.md) § Dispatch-prompt obligations).
- **`/fab-proceed`'s prefix steps.** The prefix-step dispatches were previously exempt ("no
  stage resolution — they dispatch at the inherited model). They now resolve a **role by name**
  (the resolver accepts a role name positionally, the same path `fab agent <role>` uses — no Go change):
  `/fab-switch` and `/git-branch` resolve `fab agent fast -o yaml`; the `_intake` create-intake
  dispatch resolves `fab agent default -o yaml`. (Intake itself remains advisory-only on the
  foreground `/fab-new` path, which no resolution can govern.) Both surface the required YAML keys and dispatch
  through the two seams (empty ⇒ omit). This is why `fast` is multi-referent — it governs the ship stage
  *and* these prefix-step dispatches.

`_cli-fab.md` documents the `fab agent` YAML schema; `_cli-fab.md` documents the deprecated `fab resolve-agent` compatibility
contract (Constitution constraint: CLI changes MUST update the CLI reference partial that owns the command's family — `_cli-fab.md` (core, including `fab agent`) and `_cli-fab-pane.md` (`fab pane`, `fab dispatch`)). `architecture.md` documents the `agent:` + `clis:` config blocks
alongside the existing `stage_hooks` example.

### Harness-adapter boundary (the only Claude-Code-specific layer)

Per-stage selection is **CLI-neutral by construction**, not Claude-locked:

- *Portable layers (no CLI knowledge):* the `agent:` + `clis:` config schema, and the
  entire shared resolution path projected by `fab agent <stage|role> -o yaml` (stage→role→`{cli, model, effort}`). The engine
  does no validation and echoes strings verbatim, so a project can switch agents by pointing a depth
  knob at another CLI and giving it per-role fills in that CLI's model IDs and effort
  vocabulary (`gpt-5 / reasoning_effort:high`, `<vendor-model> / <its-knob>`) and nothing in fab rejects it.
- *Harness-specific layer (the adapter):* injecting the resolved model+effort into the actual
  sub-agent dispatch is harness behavior, and the two halves use **two different seams** in Claude
  Code. **The model rides the Agent tool's `model` parameter** — a hard enum that takes a short alias
  (`opus`/`sonnet`/`haiku`/`fable`), not a full versioned id — so the model
  half reads **`model_alias` from `fab agent <stage> -o yaml`**, the deterministic Agent-tool adapter:
  the structured surface maps a resolved versioned Claude ID to its short alias (prefix-matched,
  so dated variants like `claude-haiku-4-5-20251001` resolve to `haiku`), carries a model that already
  is an alias — the built-in Claude fills `opus`/`sonnet` — through as itself, and keeps an empty
  Claude profile as the empty inherit signal; for a non-Claude ID, `model` retains the ID while
  `model_alias` is empty, and resolution descends instead of selecting the native Agent-tool rung).
  This replaces the earlier prompt-side hand-mapping
  instruction (where the orchestrator was told to translate the id by hand on every dispatch — brittle
  and easy to fumble) with a Go-side translation that cannot be skipped. **The effort rides an
  instruction in the subagent prompt** (the Agent tool exposes no effort parameter) — with the
  reliability caveat in § Effort asymmetry. The skill wiring
  names both explicitly as the Claude-Code adapter, not as universal truth. This coupling is **not
  introduced by this feature** — fab's entire existing subagent-dispatch design (`_preamble.md` §
  Subagent Dispatch) is already Claude-Code-shaped. Per-stage selection is exactly as portable as fab's
  existing dispatch: no more, no less. *(The operator launcher path is the deliberate exception — it
  does not consume `fab agent -o yaml` or `model_alias`: it resolves the **operator**-role profile
  in-process, and `spawn.WithProfile` composes the command with the resolved model verbatim. A `claude`
  CLI invocation accepts full IDs and aliases alike, so no alias mapping is involved. `WithProfile` is grammar-forgiving: it
  **substitutes** the resolved values into a `{model}`/`{effort}` **template** `interactive_command` —
  including the built-in claude default, which is templated, and a codex command — all-or-nothing (any
  placeholder disables the append entirely); an empty value drops the placeholder's token and a
  preceding `-`-flag — and,
  for a command carrying **no placeholder** (a plain-form config carried forward from before the
  templated default), it instead **appends** `--model <model> --effort <level>` at the END (last-wins;
  each flag omitted independently when its value is empty, per the `empty ⇒ omit` convention). Placing
  the default's placeholders last makes substitution byte-identical to the former append — so a
  non-Claude worker CLI is configurable without the launcher emitting Claude-only flags; 260702-6tmi,
  templated default 260703-gvxd.)*
- *Cross-harness stage dispatch (the optional `dispatch:` mapping):* the resolved `dispatch.mode` + capability
  ladder is the seam for handing one stage to a native Agent-tool, pane, or headless CLI adapter.
  Pane/headless resolution emits `dispatch.rung` plus `dispatch.command` with `{model}`/`{effort}` substituted via
  `internal/spawn`, and `dispatch.reason` when selection descended; native resolution omits the mapping. This adapter is
  the **inverse aliasing rule** from the Agent-tool `model` param: `dispatch.command` **ALWAYS
  embeds the resolved model verbatim and never translates a versioned ID to an alias**, because an
  external CLI's model flag may take only concrete IDs — CLI dispatch never aliases. (For the built-in
  Claude fills the resolved value already is the alias, which `claude --model` accepts.) Thus
  `model_alias` carries the native half while `dispatch.command` carries the resolved model for the
  CLI half. The field is **independent of** a CLI's
  `interactive_command` (which opens whole sessions); each ladder rung requires its own capability and
  command fields never substitute for one another. *`fab agent -o yaml` emits the mapping; the
  dispatch that RUNS the selected adapter (`fab dispatch`) and the skill dispatch-seam wiring that consumes it both
  shipped.* **The
  native Agent-tool adapter described in this section is one of *three* dispatch adapters catalogued
  in [`harness-adapters.md`](harness-adapters.md)** — the two `fab dispatch` modes are the others:
  **headless CLI** (3c) and **interactive pane** (`fab dispatch open` → `ready` → `deliver`,
  260805-zxe0, a worker in a tmux pane the user can watch and steer — split into the dispatching agent's
  own window, or a new window when there is no pane to split). That spec fixes the cross-adapter dispatch protocol
  (dispatch-prompt obligations, the five-state machine plus each adapter's reachable subset,
  hooks-enhance-never-own) all three share; the skill
  dispatch-seam wiring against it lives in `_preamble.md` § CLI-Adapter Dispatch + § Dispatch-Prompt
  Obligations (3d).
  **Mode resolution is shared**: `fab agent -o yaml` and the `fab dispatch` launch verbs consume the same
  pure selector. Explicit flags precede automatic `dispatch.mode` descent; pane composes the resolved
  CLI's `interactive_command`, native uses the Agent-tool capability, and headless composes
  `headless_command`. A missing capability skips its automatic rung, never substitutes a field, and
  never ascends. The pane path performs a real tmux probe; under `restart`'s automatic selection a
  failed probe re-descends on `tmux unreachable`, while `open` — where pane is explicit — hard-errors.
  Landing on native yields re-resolution guidance before state writes, and `start` hands a pane landing
  to `open` the same way.
- *Claude-flavored data (overridable):* the `claude` CLI's shipped fills use Claude model
  IDs/effort — the fills that apply while both depth knobs sit on their `claude` default; `codex` and
  `agy` ship their own model IDs (§ Built-in CLIs). Every CLI's fills are
  fab-kit's defaults for that CLI, fully replaceable via the depth knobs plus
  `clis.<name>.profiles`.
- *v1 scope is architecture-neutral + documented — NOT shipped/tested against a non-Claude harness.* No
  CLI-detection, no non-Claude integration test. The acceptance proof is
  "a non-Claude project can point a knob elsewhere and nothing in fab rejects it",
  not "we ran it on a non-Claude harness." Shipped+tested multi-CLI support is explicitly out of
  scope.

---

## apply↔review coupling

The apply stage produces the diff the review stage critiques, so the two are **economically coupled**:
if `apply` runs on a cheaper model than `review`, a sharper reviewer bounces the cheaper executor's
work more often, driving **more rework cycles** (capped at 3 per `code-review.md`). Three expensive
review rounds can cost more than running `apply` on the capable model once. "Cheaper apply = cheaper
pipeline" is therefore *not* strictly true.

Apply also has the highest output volume, which argues for the cheaper model. The kit default takes
that side: `doing` runs on Sonnet while `review` stays on Opus, **accepting** the coupling risk. Whether
a Sonnet author under an Opus critic actually raises must-fix findings and rework cycles is
**unmeasured** — the mechanism is real, its size is not known. A project that sees elevated rework sets
`agent.profiles.doing` back to an Opus model.

`doing` and `review` are **separate roles that resolve to different models** (see § Default role
profiles for the shipped values) — the role separation is what makes disagreeing with the default a
one-row config change rather than a taxonomy change. The author/critic separation that actually does
the work is the **fresh context and adversarial framing**, not model diversity; the different models
are a cost choice, not a review-quality mechanism.

---

## Model upgrade path

The shipped assignment is the table in § Default role profiles, which is the drift-guarded mirror of
`defaults.yaml` and the only place this spec states model IDs. (Deliberate: prose that restates the
IDs rots silently between bumps, which is exactly what happened to this section before.) The durable
shape is the rationale in that section: the frontier model for code review and direct interactive
work; Sonnet for the high-volume author and memory writer (`doing`, `hydrate`), at the mechanical
floor, and for the highest-volume coordinator.

fab bumps the values in **one place** (the `clis.claude.profiles` block of `defaults.yaml`) each
release, and every non-overriding project upgrades for free. The per-role fill table is fab's curated
judgment per release, not a fixed effort-per-role-rank rule — a new top model does not mechanically
promote every role, and two roles resolving to the same profile in a given release is a legitimate
outcome, not a bug. A project that pins a role opts **out** of fab's upgrade curve for that role
(correct behavior — naming it here).

### Upgrade note — the hydrate split (no migration)

Before 260719-g55d, `hydrate` mapped to `doing`, so a project carrying a `doing` override governed its
hydrate stage through that override. After the split, `hydrate` is its own role: a project with a
`doing` override but **no** `hydrate` override resolves the `hydrate` kit default
(§ Default role profiles) for the hydrate stage, not its `doing` value. **No config key changed
meaning or went inert** — the `doing` override still governs apply and review-pr exactly as before, and
`hydrate` is a newly-recognized key that was simply ignored before. Because nothing was restructured,
that shipped as an **upgrade note, not a migration**: a project that wants hydrate to keep tracking its
old `doing` value adds an `agent.profiles.hydrate:` override with that value.

### Upgrade note — the dense codex map (no migration)

Before 260908-wcib, codex's shipped `doing`/`review` rows carried effort only and `hydrate` had no row,
so the deprecated flat `clis.codex.model` (an alias for `profiles.default` — § Fill precedence)
reached those three roles through the per-field merge. After it, every codex row pins a model, so a
flat `model` reaches only the `default` role. A project that relied on the flat spelling to steer
`doing`/`review`/`hydrate` should write the modern per-role spelling
`clis.codex.profiles.<role>.model`. **No config key changed and nothing on disk restructures**, so
this is an upgrade note, not a migration — the same precedent as the hydrate split above.

---

## Foreground limitation (advisory only)

A sub-agent's model is set at dispatch time by the orchestrator. Per-stage model selection is honored
on dispatched sub-agent runs.

**Post-intake stages no longer have a foreground path (260613-fgxx).** The post-intake dual execution
mode was collapsed: apply/review/hydrate always dispatch a sub-agent, and plain `/fab-continue` is a
one-stage sequencer that resolves `fab agent <stage> -o yaml` and dispatches the stage's block just
like an orchestrator. So structured stage resolution applies uniformly across those stages regardless of
caller — this closes **Gap 1a** of the per-stage-model finding (foreground stages can't resolve a
role). Intake is pre-boundary: it runs in the main session and resolves no role.

The residual advisory-only case is narrow: a stage skill genuinely run with **no dispatch at all**.
There fab cannot switch the session model mid-run, so the configured profile is **advisory only** — the
skill MAY note "this stage is configured for X; you're on Y" but MUST NOT attempt to switch models.

> **Scope note**: this section reconciles the foreground limitation with the single post-intake
> execution mode (260613-fgxx, Change A). The **effort half** of per-stage selection — injected into the
> subagent prompt as an explicit instruction (since the Claude Code Agent tool has no effort parameter)
> — and the **compliance-visibility** behavior (surfacing the resolved YAML keys at each
> dispatch site) are written in by 260613-m3d4 (Change C); see § Skill wiring above, and § Effort
> asymmetry for how far the prompt seam actually carries. The **lone
> residual** is a first-class per-sub-agent `effort` parameter on the Agent tool (the
> per-stage-model finding's Gap 2 clean fix) — a harness ask outside fab's control, deliberately not built here.

---

## Drift guard

The two tables above (§ Default role profiles and § The fixed stage → role mapping) are verified
mirrors of `src/go/fab/defaults.yaml` (§ Default role profiles, via the knobs and
`clis.claude.profiles` it composes) and the `stageRoles` map in
`src/go/fab/internal/agent/agent.go` (§ The fixed stage → role mapping). The code side is canonical. A
test in that package (`TestDocTablesMatchAgentMaps`) parses both tables from this doc and fails if
either disagrees with the code — same pattern as `TestDocTablesMatchScoringMaps` for
`docs/specs/change-types.md`. The inline-YAML `clis.<name>.profiles` samples in § Built-in
CLIs are **shape-only** (IDs elided) and deliberately unguarded — they show which roles each map
carries, not what the rows resolve to; the rendered `fab config explain CLIs` output derives its
fill lines from `ResolveCLI` and is guarded by `TestConfigReferenceDocumentsCLIFill` in
`cmd/fab`. The non-claude fill values themselves are pinned by `TestNonClaudeCLIFillsArePinned`.

The built-in preset table in § Agent presets mirrors the `agent_presets:` block of `defaults.yaml`;
`TestDefaultsFileBuiltinPresetsArePinned` pins the three shapes and `TestDefaultsFileBuiltinPresetsAreWellFormed`
validates every built-in preset (known roles, built-in CLI names, only the allowed fields, `none` never defined).

The embedded data file has its own guard: a YAML typo is no longer a compile error, so
`TestDefaultsFileIsWellFormed` (and its siblings in `defaults_test.go`) parse `defaults.yaml`
independently and assert both knobs are set, every role's claude fill is present and populated, each
FILLED built-in carries per-role fills whose `default` entry names a model (with none using the
deprecated flat spelling, and agy's carrying no effort), kimi's fill map is asserted EMPTY, and the
package's tables and exported command values are wired to the file's keys.

---

## Out of scope (deferred)

- **User (`~/.fab-kit`) config layer** — was dropped here; subsequently shipped as the system rung
  (`260708-lpb5`) of the current four-tier cascade (environment > system
  `~/.fab-kit/config.yaml` > project > defaults). `agent` and
  `clis` are both scope `both`, so the depth knobs, a per-role CLI fill, and a role override
  are all settable once per machine.
- **Non-claude default fills** — *no longer deferred*: shipped by `260806-ywkx`. See § Built-in
  CLIs and § Refreshing the non-claude fills.
- **Role-granular reviewer keys** — obsolete: review is now a single sub-agent (260704-pag2), so there are no per-role reviewer/merge profiles to key on; the stage/role is the unit.
- **Per-invocation `--model-<stage>` flags** on the orchestrators — still deferred as an
  *orchestrator* surface. The equivalent capability exists one level down, on the resolution
  surface itself: `fab agent <stage> -o yaml [--cli] [--model] [--effort]` (`260805-j3cm`), which
  every dispatch site already calls exactly once per stage — so a per-stage override needs no new
  orchestrator flag surface and no new dispatch machinery.
- **Cost/latency telemetry** per role — out of scope; this is selection only.
- **Shipped/tested multi-CLI support** — out of scope; v1 proves architecture-neutrality only.
