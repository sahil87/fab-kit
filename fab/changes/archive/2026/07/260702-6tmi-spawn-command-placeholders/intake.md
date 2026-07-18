# Intake: Provider-forgiving spawn_command via `{model}`/`{effort}` placeholders

**Change**: 260702-6tmi-spawn-command-placeholders
**Created**: 2026-07-02

## Origin

Drafted via `/fab-draft` out of a `/fab-discuss` session on making fab-kit usable with multiple agent CLIs; will be handed off to another agent for execution.

> I also want to discuss how we can make the spawn_command more forgiving to a model where fab-kit can be used with multiple agents. Eg: codex spawning a claude session for the "apply" step.

This is **change 2 of a three-change series** scoped in that discussion:

- **Change 1** (`260702-6nke-config-reference-command`): `fab config reference` — generated reference config.yaml. Independent; see coordination note in § What Changes.
- **Change 2 (this)**: make `spawn.WithProfile` grammar-forgiving via placeholder substitution — the smallest fix that lets a non-Claude CLI be configured as `agent.spawn_command` without the operator launcher producing broken flags.
- **Change 3** (not yet drafted, spec-first): per-tier `spawn_command` + a CLI dispatch adapter for cross-harness *stage* dispatch ("codex orchestrator runs apply on claude"). Change 3 builds on this change's placeholder rules; it is explicitly out of scope here.

User approved the placeholder approach explicitly: "if `spawn_command` contains `{model}`/`{effort}`, substitute; otherwise append Claude-style (back-compat). Provider grammar moves into user config, consistent with the resolver's verbatim/no-validation philosophy."

## Why

**Problem.** `spawn.WithProfile` (`src/go/fab/internal/spawn/spawn.go`) blindly appends ` --model <id> --effort <level>` to the configured `agent.spawn_command`. That is *Claude CLI grammar*. Its one production consumer is the `fab operator` launcher (`src/go/fab/cmd/fab/operator.go:103`), which resolves the doing-tier profile and injects it before opening the operator tab. Configure `spawn_command: codex ...` and the launcher emits flags codex doesn't understand (codex wants `-m <model>`; it has no `--effort` — reasoning depth is `-c model_reasoning_effort=<level>`).

**Why it matters.** fab's entire resolution path (`fab resolve-agent`, `agent.tiers`) is deliberately provider-neutral — no validation, verbatim pass-through (Constitution Principle I; `docs/specs/stage-models.md` § No validation). The append in `WithProfile` is the one place fab *composes a command line*, and it hard-codes Claude grammar there, contradicting the philosophy. It also blocks the multi-agent direction: change 3's per-tier spawn commands reuse these placeholder rules.

**Consequence of not fixing.** Any project that points `agent.spawn_command` at a non-Claude CLI gets a broken `fab operator` launch — silently composed, failing only at tmux-window spawn time. Multi-agent use of fab stays Claude-only at the worker boundary for no essential reason.

## What Changes

### 1. Template mode in `spawn.WithProfile` (`src/go/fab/internal/spawn/spawn.go`)

- If `spawnCmd` contains `{model}` or `{effort}` (literal braces), it is a **template**:
  - Substitute every occurrence of each placeholder with the resolved value.
  - **Template mode is all-or-nothing**: the presence of *any* placeholder disables appending entirely. A half without a placeholder is simply not injected — the author's deliberate choice (prevents e.g. `--effort high` being appended to a codex command that only templated `{model}`).
- If **no placeholder** is present: today's append behavior, byte-for-byte (back-compat — existing Claude configs and the `DefaultSpawnCommand` fallback are untouched).
- **Empty-value rule (template mode)**: an empty model/effort is the "inherit/omit" signal (`_preamble.md` § Per-Stage Model Resolution). On substitution of an empty value: drop the whitespace-delimited token containing the placeholder, and also drop the immediately preceding token when it begins with `-`. This cleanly handles all common shapes:
  - `-m {model}` → both tokens dropped
  - `--model {model}` → both tokens dropped
  - `--model={model}` → single token dropped
  - `-c model_reasoning_effort={effort}` → `model_reasoning_effort=` token and preceding `-c` dropped

Example configs after this change:

```yaml
agent:
  # Claude (no placeholders — append fallback, unchanged behavior):
  spawn_command: 'claude --dangerously-skip-permissions --effort xhigh -n "$(basename "$(pwd)")"'

  # Codex (template mode — provider grammar lives in the config):
  spawn_command: 'codex -m {model} -c model_reasoning_effort={effort}'
```

### 2. `fab spawn-command` resolves placeholders (leak prevention)

Edge discovered during drafting: the `/fab-operator` *skill* spawns workers using raw `fab spawn-command --repo <target>` output with **no** profile injection (`src/kit/skills/fab-operator.md` § spawn sequence). A templated `spawn_command` would leak literal `{model}`/`{effort}` braces into worker spawn commands.

Fix: `fab spawn-command` (`src/go/fab/cmd/fab/spawn_command.go`) applies the template resolution with an **empty profile** before printing — with the empty-value rule above, a templated command degrades to a clean invocation (placeholders and their flag tokens stripped), and a non-templated command prints verbatim as today. This is a CLI output-behavior change → constitution requires updating `src/kit/skills/_cli-fab.md` and tests.

### 3. Tests (`spawn_test.go`, `spawn_command_test.go`)

Table-driven cases: no-placeholder append (existing cases stay green); both placeholders substituted; single placeholder (other half NOT appended); empty model / empty effort / both empty under each token shape above; multiple occurrences of one placeholder; placeholder embedded mid-word; `fab spawn-command` stripping on a templated config.

### 4. Prose sweep (mirror class — enumerate up front per code-quality.md § Sibling & Mirror Sweeps)

The append behavior is described in several places; all must be updated in the same change:

- `src/kit/skills/_preamble.md` (§ Per-Stage Model Resolution, operator-launcher exception note, ~line 325) + its SPEC mirror
- `src/kit/skills/fab-operator.md` (Key Properties row "appends `--model`/`--effort`", ~line 703) + its SPEC mirror
- `src/kit/skills/_cli-fab.md` (`fab spawn-command` entry) + SPEC mirror if present
- `docs/specs/stage-models.md` (§ Harness-adapter boundary operator-launcher note, ~line 247; § Skill wiring)
- `src/kit/scaffold/fab/project/config.yaml` (`spawn_command` comment gains one line noting the optional placeholders)
- Sweep by grep for "appends `--model`" / "WithProfile" repo-wide at apply — do not rely on this list alone.

**Coordination with change 1 (6nke)**: `fab config reference` documents `agent.spawn_command`; whichever change lands second must ensure the reference text mentions placeholder semantics.

### Non-Goals

- **Per-tier `spawn_command`** (`agent.tiers.<tier>.spawn_command`) — change 3.
- **CLI dispatch adapter / headless stage dispatch** (running pipeline stages as subprocesses on another harness) — change 3, spec-first.
- **No provider validation** — fab still never checks model/effort values; placeholders only relocate *where* the strings land.
- **No new CLI commands or flags.**

## Affected Memory

- `_shared/configuration`: (modify) `agent.spawn_command` schema gains placeholder semantics (template mode, append fallback, empty-value token-drop rule)
- `_shared/context-loading`: (modify) the per-stage-model-resolution paragraph's operator-launcher exception note (appends → substitutes-when-templated)
- `runtime/operator`: (modify) launcher composition behavior and the worker-spawn leak-prevention in `fab spawn-command`
- `distribution/kit-architecture`: (modify) `fab spawn-command` output contract note

## Impact

- `src/go/fab/internal/spawn/spawn.go` + `spawn_test.go` — template mode, empty-value rule
- `src/go/fab/cmd/fab/spawn_command.go` + `spawn_command_test.go` — empty-profile resolution before print
- `src/go/fab/cmd/fab/operator.go` — no call-shape change expected (consumer of `WithProfile`)
- `src/kit/skills/_preamble.md`, `fab-operator.md`, `_cli-fab.md` + SPEC mirrors; `docs/specs/stage-models.md`; scaffold comment
- No config schema change (placeholders are just string content); no migration needed; no `.status.yaml` change

## Open Questions

*(none — decision points graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Placeholder syntax is literal `{model}` / `{effort}`; every occurrence substituted | User named the syntax explicitly in discussion | S:90 R:85 A:90 D:85 |
| 2 | Certain | No placeholders ⇒ append fallback, byte-identical to today | User approved back-compat explicitly; existing configs must not change behavior | S:90 R:90 A:95 D:90 |
| 3 | Confident | Any-placeholder ⇒ full template mode (the non-templated half is NOT appended) | Prevents cross-grammar contamination; alternative (per-half independence) would append Claude flags to non-Claude commands | S:40 R:90 A:80 D:55 |
| 4 | Confident | Empty-value rule: drop the placeholder's token + preceding `-`-prefixed token | Simplest deterministic rule covering all four common flag shapes; alternatives (raw empty substitution, full templating language) rejected as broken/heavy | S:45 R:90 A:75 D:45 |
| 5 | Confident | `fab spawn-command` resolves templates with an empty profile before printing | Emerged from drafting analysis (not user-discussed): prevents literal-brace leak into the operator skill's worker spawns; alternative (print verbatim + document hazard) has a clear loser profile | S:35 R:85 A:80 D:60 |
| 6 | Certain | Implementation confined to `internal/spawn` + `spawn_command.go`; `operator.go` call shape unchanged | Single production consumer confirmed by sweep; the seam already exists | S:75 R:90 A:90 D:85 |
| 7 | Confident | Prose sweep class as enumerated in § What Changes item 4 | Grep-verified today (preamble:325, fab-operator:703, stage-models:247); apply must re-sweep — per-file lists systematically under-cover | S:60 R:85 A:85 D:75 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
