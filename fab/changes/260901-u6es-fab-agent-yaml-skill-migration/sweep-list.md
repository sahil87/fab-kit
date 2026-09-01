# Sweep List: fab agent YAML Skill Migration

Captured before prose edits on 2026-09-01. Counts are matching lines from one combined
`rg` over these phrase classes: `resolve-agent`, `resolve_agent`, `--alias`, `dispatch=`,
`` `model=` line ``, and `model=/effort=`. A line matching more than one phrase class is
counted once.

## Source-built schema probe

Command (from `src/go/fab`):

```sh
go run ./cmd/fab agent apply -o yaml
```

Observed projection:

```yaml
selector: apply
kind: stage
role: doing
provider: codex
model: gpt-5.6-sol
effort: xhigh
command: codex --dangerously-bypass-approvals-and-sandbox -m gpt-5.6-sol -c model_reasoning_effort=xhigh
model_alias: ""
template: codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}
fill_mode: template
source:
    provider: agent.workers
    model: providers.codex.profiles.default
    effort: providers.codex.profiles.doing
dispatch:
    rung: pane
    command: codex --dangerously-bypass-approvals-and-sandbox -m gpt-5.6-sol -c model_reasoning_effort=xhigh
```

Classification: `dispatch:` is present, so the CLI-adapter arm applies; `model_alias` is
empty because the configured model is non-Claude. `dispatch.rung` is labelled but remains
unconsumed by choreography in this change.

## Migrate: live skill dispatch instructions

| File | Initial matching lines | Disposition |
|------|-----------------------:|-------------|
| `src/kit/skills/_preamble.md` | 18 | Migrate dispatch seam, surfacing, continuation, and CLI-adapter wording; preserve choreography |
| `src/kit/skills/_pipeline.md` | 4 | Migrate stage resolve and light-lane wording |
| `src/kit/skills/fab-continue.md` | 8 | Migrate per-stage resolver instructions |
| `src/kit/skills/fab-fff.md` | 3 | Migrate ship/review-pr resolver instructions |
| `src/kit/skills/fab-proceed.md` | 4 | Migrate role-selector resolution and key-presence wording |
| `src/kit/skills/_cli-agents.md` | 2 | Migrate operator-facing resolver references |
| `src/kit/skills/_cli-fab.md` | 34 | Mixed: migrate instructional consumers; retain and deprecate the working command's reference contract |

`src/kit/skills/fab-adopt.md` has no initial hit; it consumes the shared pipeline by pointer.

## Migrate: specifications

| File | Initial matching lines | Disposition |
|------|-----------------------:|-------------|
| `docs/specs/stage-models.md` | 49 | Mixed: migrate skill wiring; retain frozen resolve-agent contract as deprecated surface |
| `docs/specs/harness-adapters.md` | 11 | Migrate resolve step, branch discriminator, and start-probe rationale |
| `docs/specs/skills.md` | 5 | Migrate aggregate skill wiring |
| `docs/specs/glossary.md` | 4 | Migrate live resolver/dispatch definitions; retain CLI enumeration where both commands exist |
| `docs/specs/architecture.md` | 2 | Migrate aggregate wiring |
| `docs/specs/config.md` | 2 | Migrate live consumer references |
| `docs/specs/index.md` | 1 | Regenerate the stage-models summary wording manually with the spec edit |

## Migrate: present-truth memory

| File | Initial matching lines | Disposition |
|------|-----------------------:|-------------|
| `docs/memory/runtime/providers-and-profiles.md` | 55 | Mixed: migrate consumers and YAML projection; retain deprecated command contract with framing |
| `docs/memory/_shared/context-loading.md` | 18 | Migrate canonical dispatch-seam wiring and design decisions to YAML keys |
| `docs/memory/_shared/configuration.md` | 9 | Migrate live config/resolution references |
| `docs/memory/pipeline/execution-skills.md` | 8 | Migrate stage-dispatch wiring |
| `docs/memory/runtime/dispatch.md` | 5 | Migrate earlier-projection wording and re-anchor labelled-but-unconsumed rung rationale |
| `docs/memory/distribution/kit-architecture.md` | 3 | Migrate live surface description |
| `docs/memory/runtime/agent-primitives.md` | 2 | Migrate fill-consuming resolution references |
| `docs/memory/pipeline/issue-linking.md` | 2 | Migrate inline/no-resolution wording |
| `docs/memory/distribution/migrations.md` | 1 | Historical catalog wording; classify in context and keep if it describes a shipped migration |
| `docs/memory/runtime/operator.md` | 1 | Historical contrast inside a present-truth design decision; reword against current YAML surface |
| `docs/memory/runtime/index.md` | 1 | Generated index; regenerate from updated description |
| `docs/memory/_shared/index.md` | 1 | Generated index; regenerate from updated description |

## Migrate: bounded Go strings, pinned tests, and consumer comments

| File | Initial matching lines | Disposition |
|------|-----------------------:|-------------|
| `src/go/fab/cmd/fab/dispatch_start.go` | 2 | Change only the native-mode user remedy; keep implementation comment contextual |
| `src/go/fab/internal/configref/configref.go` | 1 | Change generated fence guidance |
| `src/go/fab/cmd/fab/dispatch_start_test.go` | 2 | Update pinned remedy and consumer-facing comment only |
| `src/go/fab/cmd/fab/dispatch_restart_test.go` | 2 | Update pinned remedy and consumer-facing comment only |
| `src/go/fab/cmd/fab/config_show_init_test.go` | 1 | Update fence pin if exact text is asserted |
| `src/go/fab/cmd/fab/noproject_config_test.go` | 3 | Update fence pin or skill-consumer comment only; retain config-only command coverage |
| `src/go/fab/cmd/fab/main_test.go` | 1 | Comment classification only |

## Retain: deprecated command contract and parity/golden coverage

The following are implementation/reference ownership for the still-working, frozen
`fab resolve-agent` surface. Assertions, line output, flags, and golden/parity expectations
stay byte-identical.

| File | Initial matching lines |
|------|-----------------------:|
| `src/go/fab/cmd/fab/resolve_agent.go` | 11 |
| `src/go/fab/cmd/fab/resolve_agent_test.go` | 93 |
| `src/go/fab/cmd/fab/agent_surface_test.go` | 2 |
| `src/go/fab/internal/agent/resolution.go` | 3 |
| `src/go/fab/internal/agent/resolution_test.go` | 2 |
| `src/go/fab/internal/agent/agent.go` | 4 |
| `src/go/fab/internal/agent/agent_test.go` | 1 |
| `src/go/fab/internal/dispatch/dispatch.go` | 1 |
| `src/go/fab/internal/dispatch/pane_mode.go` | 1 |
| `src/go/fab/internal/setupcheck/probes.go` | 1 |
| `src/go/fab/internal/config/config.go` | 1 |
| `src/go/fab/internal/config/config_test.go` | 1 |
| `src/go/fab/cmd/fab/agent.go` | 1 |
| `src/go/fab/cmd/fab/config.go` | 1 |
| `src/go/fab/defaults.yaml` | 2 |

## Retain: enumeration

- `src/go/fab/cmd/fab/skill.md` (2 matching lines) lists commands that still exist; keep
  `fab resolve-agent` and ensure `fab agent` remains alongside it.
- `docs/specs/glossary.md`'s CLI inventory keeps both command names while its live dispatch
  definition migrates to `fab agent -o yaml`.

## Retain: frozen history

These initial hits are dated/generated history and are never rewritten:

- `docs/memory/_shared/log.md` (7), `docs/memory/_shared/log.seed.md` (6)
- `docs/memory/pipeline/log.md` (4), `docs/memory/pipeline/log.seed.md` (3)
- `docs/memory/runtime/log.md` (1), `docs/memory/runtime/log.seed.md` (1)
- `fab/changes/archive/`, `docs/findings/`, and `fab/plans/` (excluded from the live-scope
  file list and retained wholesale)

## Final-sweep rule

At T014 every residual must fit one of: deprecated-command contract, implementation/parity
coverage, command enumeration, or frozen history. Any residual instruction telling a skill or
pipeline consumer to use `fab resolve-agent`/ordered lines is unclassified and must be fixed.

## Final sweep result

Completed after the prose, memory, and bounded Go-string edits:

- **Migrated:** live dispatch skills outside `_cli-fab.md` have zero hits for
  `resolve-agent`, `--alias`, `dispatch=`, `` `model=` line ``, or `model=/effort=`.
- **Deprecated command contract:** `_cli-fab.md`, `docs/specs/stage-models.md`, and
  `docs/memory/runtime/providers-and-profiles.md` retain the frozen signature, flags, and
  line-projection details under explicit deprecation framing.
- **Enumeration:** `docs/specs/glossary.md`, `src/go/fab/cmd/fab/skill.md`, and config-only
  command lists retain both existing command names; the resolution guidance names
  `fab agent <stage|role> -o yaml` first. `docs/site/skill.md` is the canonical source for
  the embedded `cmd/fab/skill.md` copy, so both carry the same updated enumeration.
- **Implementation/parity:** `resolve_agent.go`, its exact-output tests, shared-resolution
  internals, and engine comments retain the old token because they implement or pin the
  frozen compatibility surface. The only resolver-test edit is the T013 consumer comment;
  golden/parity expectations are unchanged.
- **History:** memory `log.md`/`log.seed.md`, archived changes, findings, and plan documents
  are untouched. `distribution/migrations.md` labels its one legacy hit as the historical
  surface of that shipped migration and names the current YAML consumer separately.
- **Generated indexes:** `fab memory-index` refreshed `_shared/index.md` and
  `runtime/index.md` from their updated topic descriptions.

No unclassified instructional residual remains, and no consumer branches on
`dispatch.rung`; the labelled rung remains deliberately unconsumed by this change.
