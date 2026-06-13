# Intake: resolve-agent --alias flag (Claude-Code model alias adapter)

**Change**: 260613-yky7-resolve-agent-alias-flag
**Created**: 2026-06-13

## Origin

Surfaced during a `/fab-discuss` session investigating a live pipeline failure. A previous
agent, dispatching the **apply** stage sub-agent, resolved `fab resolve-agent apply` →
`model=claude-opus-4-8` and passed that full model ID verbatim into the **Agent tool's `model`
parameter**. The dispatch failed with **"Invalid tool parameters."** The agent recovered by
hand-mapping `claude-opus-4-8` → `opus`.

The user initially hypothesized the `agent.tiers` config structure was wrong. Investigation
disproved that: the config, the `defaultTiers` Go map, and `fab resolve-agent`'s verbatim output
are all **correct per `docs/specs/stage-models.md`** (provider-neutral, full-ID, drift-guarded).
The real fault is a **vocabulary mismatch between two distinct Claude-Code surfaces**:

| Surface | Accepts full ID `claude-opus-4-8`? | Accepts alias `opus`? |
|---|---|---|
| `claude` **CLI** `--model` flag (operator launcher, `spawn_command`) | Yes | Yes |
| **Agent tool** `model` param (orchestrator sub-agent dispatch) | **No — hard enum** | Yes |

The Agent tool's `model` parameter is a hard JSON-schema enum: `["sonnet","opus","haiku","fable"]`.
It rejects full IDs. The CLI `--model` flag accepts both (`claude --help`: "Provide an alias ...
or a model's full name").

Interaction mode: conversational. The user chose the fix shape explicitly:
- **Fix via a `--alias` flag** (encode the mapping deterministically in Go), rejecting the
  prompt-only hand-mapping option — *because the live failure was precisely an agent fumbling the
  hand-map*.
- The user also raised "put aliases in the tiers instead of full IDs"; this was analyzed and
  **rejected** (it would break provider-neutrality, weaken the Fable version-pin discipline, and
  require a coordinated edit across the Go map + two drift-guarded spec tables + config comments +
  migration — pushing a harness quirk into the provider-neutral core). See Design Decisions.

## Why

**Problem.** The orchestrated pipeline (`/fab-ff`, `/fab-fff`, `/fab-continue`) is *broken at the
Agent-tool dispatch seam* whenever a stage resolves to a full model ID — which is every stage
under the shipped defaults. Per-stage model selection (feature #406/#407) cannot actually dispatch
a sub-agent with its resolved model without a manual workaround.

**Consequence if unfixed.** Every orchestrated run either fails at first sub-agent dispatch or
silently depends on the agent improvising a full-ID→alias mapping by hand — exactly the
error-prone step that failed in the live run. Per-stage model selection is effectively unusable in
its primary (orchestrated) mode.

**Why this approach.** The mismatch is genuinely **specific to the Claude-Code Agent-tool
surface**, not to fab's resolution logic. `stage-models.md` already names "injecting the resolved
model into the Agent dispatch" as the **harness-adapter boundary** — the one Claude-Code-specific
layer. The fix belongs *exactly there*: a `--alias` flag that emits the Claude-Code alias on
demand, leaving the provider-neutral default (full ID) untouched for the CLI/operator path.
Encoding the mapping in Go (vs. prompt-side) makes it deterministic — the resolver, not a
per-dispatch agent guess, owns the translation.

## What Changes

### 1. New `--alias` flag on `fab resolve-agent <stage>`

A boolean `--alias` flag on the cobra command in `src/go/fab/cmd/fab/resolve_agent.go`. Default
(absent) = today's behavior exactly (emit full model ID). When set, the `model=` line emits the
Claude-Code short alias instead. The `effort=` line is **unaffected** by `--alias`.

```
$ fab resolve-agent apply
model=claude-opus-4-8
effort=high

$ fab resolve-agent apply --alias
model=opus
effort=high
```

### 2. Alias mapping in `internal/agent`

A new exported function in `src/go/fab/internal/agent/agent.go` (the natural home — alongside the
tier tables and `Resolve`). Signature shape (final name decided in plan):

```go
// ModelAlias maps a full Claude model ID to its Claude-Code short alias
// (the Agent tool's `model` enum: opus/sonnet/haiku/fable). Returns the input
// VERBATIM when no mapping applies (empty string, or an unrecognized/non-Claude
// ID) — preserving provider-neutrality: --alias is a Claude-Code adapter, not a
// validator. Prefix-matched so claude-haiku-4-5-20251001 → haiku.
func ModelAlias(model string) string
```

Mapping (prefix-based to absorb dated variants like `claude-haiku-4-5-20251001`):

| Full ID prefix | Alias |
|---|---|
| `claude-opus-` | `opus` |
| `claude-sonnet-` | `sonnet` |
| `claude-haiku-` | `haiku` |
| `claude-fable-` | `fable` |

- **Empty model** → empty (no-op): preserves the "inherit the session model" signal. The `model=`
  line stays empty under `--alias`.
- **Unmapped / non-Claude ID** (e.g. `gpt-5`) → **returned verbatim** (pass-through). This keeps
  `--alias` from becoming a Claude-only validator: a project that overrode a tier to another
  provider's model still gets its string through unchanged. (Decision point — see Open Questions
  / Assumptions; leaning verbatim pass-through.)

The flag wiring formats output via the existing `formatAgentProfile` path — `resolveAgentCmd`
applies `ModelAlias` to `profile.Model` before formatting when `--alias` is set (alternatively,
pass a flag into a small format variant; plan decides the cleanest seam). The `model=` /
`effort=` line contract and byte-stability are otherwise unchanged.

### 3. Skill wiring — orchestrator/dispatch use `--alias` for the Agent-tool path

The canonical instruction lives in **`src/kit/skills/_preamble.md` § Subagent Dispatch →
Per-Stage Model Resolution**, specifically the **"Harness-adapter boundary (Claude Code)"**
paragraph (~line 346). Today it says the resolved model goes into "the Agent tool's `model`
parameter" but does not account for the enum mismatch. Update it to instruct: **for the
Claude-Code Agent-tool dispatch, resolve with `fab resolve-agent <stage> --alias`** so the emitted
`model=` is already an Agent-tool-valid alias; pass it straight into the Agent tool `model` param
(empty ⇒ omit/inherit, unchanged).

The three consuming skills echo the preamble and must be updated to pass `--alias` (or to defer to
the preamble's updated instruction — keep the existing defer-to-preamble pattern, just ensure the
`--alias` requirement is unambiguous):
- `src/kit/skills/fab-ff.md` (line ~37 note)
- `src/kit/skills/fab-fff.md` (lines ~37, ~47, ~57 notes)
- `src/kit/skills/fab-continue.md` (line ~154 review-dispatch note)

**Operator launcher path is NOT changed.** `fab operator` / the `_cli-fab.md:554` path appends
`--model <full-id>` to a `claude` **CLI** invocation, which accepts full IDs. It must keep
resolving **without** `--alias`. The CLI and Agent-tool paths deliberately diverge.

### 4. CLI reference — `_cli-fab.md`

Constitution constraint (CLI changes MUST update `_cli-fab.md`): update the `fab resolve-agent`
entry (§ around line 227) to document the `--alias` flag — what it emits, that it's a Claude-Code
Agent-tool adapter, that default behavior is unchanged (full ID), and that empty/non-Claude models
pass through verbatim.

### 5. Spec — `docs/specs/stage-models.md`

Update **§ Harness-adapter boundary (the only Claude-Code-specific layer)** to document the
two-surface vocabulary split and the `--alias` flag as the adapter mechanism for the Agent-tool
path. This is a spec-of-design update (the spec already frames this boundary as the Claude-Code
adapter — we're making the alias mechanism concrete). The two **drift-guarded tables** (default
tier profiles, stage→tier mapping) are **NOT touched** — full IDs stay canonical, so
`TestDocTablesMatchAgentMaps` is unaffected.

### 6. Go test coverage

Add tests (test-alongside, `**/*_test.go`):
- `ModelAlias` unit tests: each of opus/sonnet/haiku/fable full IDs → alias; dated haiku variant →
  `haiku`; empty → empty; unmapped (`gpt-5`) → verbatim.
- `resolve-agent --alias` command-level test: `apply --alias` emits `model=opus` + `effort=high`;
  without `--alias` still emits `model=claude-opus-4-8` (regression guard for the unchanged
  default); a tier with empty model under `--alias` still emits an empty `model=` line.

## Affected Memory

- `pipeline/stage-models`: (modify) — the resolve-agent `--alias` flag and the two-surface
  (CLI vs Agent-tool) adapter split. *(Exact memory file path confirmed at hydrate — the pipeline
  domain owns stage-models / dispatch wiring; this may be the stage-models memory file or the
  dispatch-wiring file. No new spec-level behavior beyond the documented adapter mechanism.)*

## Impact

- `src/go/fab/internal/agent/agent.go` — new `ModelAlias` function (+ its test file).
- `src/go/fab/cmd/fab/resolve_agent.go` — new `--alias` bool flag; apply mapping pre-format (+ test).
- `src/kit/skills/_preamble.md` — Harness-adapter boundary paragraph (canonical instruction).
- `src/kit/skills/fab-ff.md`, `fab-fff.md`, `fab-continue.md` — per-stage-model dispatch notes.
- `src/kit/skills/_cli-fab.md` — resolve-agent signature gains `--alias`.
- `docs/specs/stage-models.md` — § Harness-adapter boundary (no drift-guarded table touched).
- **No change**: `defaultTiers` / `stageTiers` maps, the operator launcher path, `agent.tiers`
  config schema, the `model=`/`effort=` default output contract.

## Open Questions

- Unmapped / non-Claude model ID under `--alias`: pass through verbatim, or error? (Leaning
  verbatim pass-through to preserve provider-neutrality — `--alias` is a best-effort Claude-Code
  adapter, not a validator. Recorded as a Confident assumption below.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix via a Go-side `--alias` flag on `fab resolve-agent`, not prompt-side hand-mapping | User chose this explicitly in discussion; the live failure was an agent fumbling the hand-map, so determinism in Go is the durable fix | S:95 R:80 A:90 D:95 |
| 2 | Certain | Do NOT switch tier defaults to aliases; full IDs stay canonical in `defaultTiers` + drift-guarded spec tables | User raised this alternative; analyzed and rejected — breaks provider-neutrality, weakens the Fable version-pin, and forces a coordinated multi-file edit pushing a harness quirk into the provider-neutral core | S:90 R:75 A:90 D:90 |
| 3 | Certain | Default behavior (no `--alias`) is byte-identical to today (full model ID); CLI/operator path unchanged | The `claude` CLI `--model` flag accepts full IDs (`claude --help` confirms); only the Agent-tool enum rejects them. Two surfaces must diverge | S:95 R:85 A:95 D:95 |
| 4 | Confident | Mapping is prefix-based (`claude-haiku-` → `haiku`) so dated variants (`claude-haiku-4-5-20251001`) resolve | The Agent enum is family-level (opus/sonnet/haiku/fable); full IDs carry version/date suffixes. Prefix match is the robust mapping | S:75 R:80 A:85 D:80 |
| 5 | Confident | Unmapped / non-Claude model under `--alias` passes through verbatim (not an error) | Preserves provider-neutrality — `--alias` is a Claude-Code adapter, not a validator; a non-Claude override still flows. Open question, but the leaning is clear and low-risk (reversible) | S:65 R:80 A:75 D:70 |
| 6 | Confident | `ModelAlias` lives in `internal/agent` (alongside the tier tables + `Resolve`) | That package already owns the model vocabulary and is the drift-guard's subject; cohesive home | S:75 R:85 A:90 D:80 |
| 7 | Confident | Skills keep the defer-to-preamble pattern; canonical `--alias` instruction lives in `_preamble.md` Harness-adapter boundary, echoed by ff/fff/continue | Matches the existing single-source convention (the three skills already defer to the preamble for per-stage model resolution) | S:80 R:85 A:90 D:85 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
