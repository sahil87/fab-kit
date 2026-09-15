# Intake: Embed Agent Defaults as defaults.yaml (Layer 0)

**Change**: 260806-2j2i-embed-agent-defaults-layer0
**Created**: 2026-08-06

## Origin

Change 1 of a three-change series (2j2i → j9nh → ywkx) designed in a `/fab-discuss` session on 2026-08-06 with fab-kit's owner. The series overhauls the agent-config surface: (1) this change relocates the built-in defaults from Go literals to an embedded YAML data file (behavior-neutral substrate), (2) j9nh reshapes the schema (`agent.tiers` → `agent.profiles`, per-role provider fills, `agent.session`/`agent.workers` depth knobs), (3) ywkx ships built-in codex/gemini model fills.

> Where would fab-kit manage its own internal default of the tier → model mapping per provider? Would this shift to some yaml file in the repo?

Key decisions from the discussion (all settled with the user):

- Move the built-in defaults data to a YAML file in the repo, **embedded in the binary via `go:embed`** — explicitly NOT read from the kit cache at runtime.
- The file is shaped exactly like a user config-file fragment, so it can later serve as layer 0 of the existing config cascade (project > system > defaults).
- The file lives next to the embedding package (`src/go/fab/internal/agent/`), NOT under `src/kit/` (that tree deploys to the kit cache and would imply runtime reading).
- This change is deliberately **behavior-neutral**: today's schema, today's values, byte-stable resolution. The schema reshape is j9nh's job.

## Why

Today fab-kit's built-in agent defaults live as Go map literals in `src/go/fab/internal/agent/agent.go`: `defaultTiers` (the six tier→profile rows, "the ONE place bumped when a new top model lands") and `defaultProviders` (the three grammar-only provider rows). Three problems, all of which the series aggravates:

1. **The next change (j9nh) makes the data deeply nested.** Per-role fill maps per provider (`providers.<name>.profiles.<role>.{model,effort}`) are data-shaped; as Go literals they get verbose and hard to review.
2. **The fence renderer is a format translation.** `fab config reference` re-renders Go maps into commented YAML, and a fleet of drift-guard tests (`TestDefaultTierProfilesArePinned`, `TestDocTablesMatchAgentMaps`, `TestMirrorDocsMatchDefaultTiers`, `TestCLIFabReferenceListsDefaultTiers`) exists largely to police restatements of values that originate in Go. With the source of truth already in YAML, built-ins↔fence drift becomes impossible by construction rather than guarded by test.
3. **A model bump requires reading Go.** As a data file, a bump is a few-character diff reviewable without Go knowledge.

If we don't do this first, j9nh's diff mixes a schema redesign with a data-representation move, inflating its review surface. Isolating the behavior-neutral plumbing here keeps j9nh's diff pure signal.

Why embed rather than read from `$(fab kit-path)` at runtime: kit and binary release atomically, so an on-disk YAML updates at exactly the same cadence as a Go constant — runtime reading gains nothing and adds a binary↔kit version-skew failure mode (`fab resolve-agent` cannot break from a missing/corrupt cache today, and must stay that way).

## What Changes

### 1. New embedded data file: `src/go/fab/internal/agent/defaults.yaml`

Shaped as a config-file fragment in **today's schema**, carrying today's exact values (verbatim from `agent.go`):

```yaml
# fab-kit's built-in agent defaults — the defaults layer under the config
# cascade (project > system > this file). Embedded via go:embed; a model
# bump edits this file and cuts a release.
providers:
  claude:
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
  codex:
    session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
    dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
  gemini:
    session_command: 'gemini -m {model}'
    dispatch_command: 'gemini -m {model}'
agent:
  tiers:
    default:  { provider: claude, model: claude-fable-5,  effort: high }
    operator: { provider: claude, model: claude-sonnet-5, effort: medium }
    doing:    { provider: claude, model: claude-opus-5,   effort: xhigh }
    review:   { provider: claude, model: claude-opus-5,   effort: xhigh }
    hydrate:  { provider: claude, model: claude-opus-5,   effort: high }
    fast:     { provider: claude, model: claude-sonnet-5, effort: medium }
```

### 2. Data-source swap in `internal/agent`

- `defaultTiers` and `defaultProviders` Go literals are replaced by structures parsed once (package init or `sync.Once`) from the embedded file.
- The public API is unchanged: `DefaultTier()`, `TierForStage()`, `IsTierName()`, `TierNames()`, and every consumer (resolve-agent, operator launcher, `fab agent`, configupgrade fence renderer) behave byte-identically.
- `stageTiers` (the fixed stage→tier map) **stays in Go** — it is fab-owned policy with tested invariants (the review/hydrate fixed-point property), not tunable data. The boundary is deliberate: everything in defaults.yaml is user-overridable by writing the same keys in config; everything remaining in Go is not.
- This change swaps the data source only; it does NOT rewrite the config-merge/resolution code paths (that is j9nh's precedence rewrite). The file being shaped as a config fragment is what makes that later unification possible.

### 3. Validation test

A YAML typo is no longer a compile error, so add a test that parses the embedded file and asserts: all six tiers present, every profile field non-empty where today's values are non-empty, all three providers present with their command fields, and the parse itself errors cleanly. The existing pinned-profile and doc-mirror drift guards keep passing unchanged (they read through `DefaultTier()`).

### 4. Doc touch-ups (small)

- `docs/specs/stage-models.md`: the "where the defaults live / how to bump a model" prose points at `defaults.yaml` instead of the Go map.
- The `agent.go` comment block ("TO BUMP A MODEL: edit the line here…") moves to/points at the YAML file.
- No CLI signature changes → no `_cli-fab.md` command changes (verify: only prose that names `agent.go` as the bump site, if any).

## Affected Memory

- `runtime/providers-and-tiers`: (modify) the "kit defaults" provenance — defaults now live in embedded `defaults.yaml`, not Go literals; bump procedure updated
- `_shared/configuration`: (modify) note the defaults layer's physical source under the cascade (minor)

## Impact

- `src/go/fab/internal/agent/agent.go` (literals removed, parse-on-init added), new `src/go/fab/internal/agent/defaults.yaml`, new validation test
- Existing tests in `internal/agent`, `internal/configupgrade`, `cmd/fab` (resolve-agent/agent/batch tests) must pass **unchanged** — they are the behavior-neutrality proof
- `docs/specs/stage-models.md` (bump-site prose), memory files above
- No CLI surface change, no config schema change, no migration

## Open Questions

*(none — design settled in discussion)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Embed via `go:embed`; never read from kit cache at runtime | Discussed explicitly — user agreed; avoids binary↔kit skew failure mode | S:90 R:80 A:95 D:90 |
| 2 | Certain | File keeps today's schema and values; reshape deferred to j9nh | Series design settled in discussion — this change is behavior-neutral by contract | S:90 R:85 A:95 D:90 |
| 3 | Certain | File location `src/go/fab/internal/agent/defaults.yaml` (next to the embedding package, not `src/kit/`) | Discussed — `src/kit/` deploys to the kit cache and would imply runtime reading; no objection raised | S:75 R:85 A:85 D:80 |
| 4 | Confident | Data-source swap only; config-merge unification deferred to j9nh's precedence rewrite | Minimal-diff refactor discipline; the byte-stable test contract is easiest to hold this way | S:65 R:75 A:85 D:75 |
| 5 | Certain | `stageTiers` stays in Go (policy, not data) | Discussed — the YAML/Go boundary doubles as the overridable/fixed signal; user agreed | S:80 R:80 A:90 D:85 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
