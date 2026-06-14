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

**Prior art — PR #413 (`m3d4`, merged) already documents this mismatch, but as a manual
instruction.** #413 ("Apply per-stage model tier uniformly") rewrote the dispatch-wiring prose
across `_preamble.md`, `fab-ff/fff/continue.md`, and `stage-models.md` and added an explicit note:
the Agent `model` param takes a short alias, *"so the orchestrator maps the resolved id to the
alias at the dispatch seam."* That is the **prompt-side hand-mapping** approach — it tells the
dispatching agent to translate the id by hand on every dispatch. **This change replaces that
brittle instruction with a deterministic Go-side translation**: a `--alias` flag on
`fab resolve-agent` that emits the alias directly, so no agent ever hand-maps. (The live failure
*was* an agent fumbling exactly that hand-map — encoding it in Go removes the failure mode.)

Interaction mode: conversational. Decisions taken in discussion:
- **Fix via a `--alias` flag** (encode the mapping deterministically in Go), explicitly rejecting
  the prompt-only hand-mapping option (which is what #413 currently ships).
- **Do NOT switch tier defaults to aliases.** Analyzed and rejected — it would break
  provider-neutrality, weaken the Fable version-pin discipline, and force a coordinated edit across
  the Go map + two drift-guarded spec tables + config comments + migration, pushing a harness quirk
  into the provider-neutral core. See Design Decisions / Assumptions.

## Why

**Problem.** The orchestrated pipeline (`/fab-ff`, `/fab-fff`, `/fab-continue`) dispatches every
post-intake stage as a sub-agent, and per #413 each dispatch is *instructed to hand-map* the
resolved full model ID to the Agent-tool alias. Hand-mapping at the prompt layer is exactly the
step that failed in the live run — it is brittle and easy for a dispatching agent to skip or get
wrong.

**Consequence if unfixed.** Every orchestrated dispatch carries an avoidable failure mode: the
agent must remember to translate `claude-opus-4-8` → `opus` (and the dated/family variants)
correctly, every time, by following prose. A single miss reproduces the original
"Invalid tool parameters" failure.

**Why this approach.** The mismatch is genuinely **specific to the Claude-Code Agent-tool
surface**, which `stage-models.md` already names as the **harness-adapter boundary**. Moving the
translation from a per-dispatch prose instruction into a deterministic resolver flag (`--alias`)
puts it at exactly that boundary and makes it impossible to fumble. The provider-neutral default
(full ID) is untouched for the CLI/operator path; `--alias` is an opt-in Claude-Code adapter.

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
  provider's model still gets its string through unchanged.

The flag wiring formats output via the existing `formatAgentProfile` path — `resolveAgentCmd`
applies `ModelAlias` to `profile.Model` before formatting when `--alias` is set (alternatively,
pass a flag into a small format variant; plan decides the cleanest seam). The `model=` /
`effort=` line contract and byte-stability are otherwise unchanged.

### 3. Skill wiring — REPOINT the existing (post-#413) adapter prose at `--alias`

> **Note**: #413 already wrote the id→alias adapter prose into these files as a *hand-mapping*
> instruction ("the orchestrator maps the resolved id to the alias at the dispatch seam"). This
> change does NOT add new adapter docs — it **edits the existing prose** to say "resolve with
> `fab resolve-agent <stage> --alias` (emits the alias directly)" instead of "map the id by hand".
> Each site below currently contains a hand-map phrasing that must be repointed.

- **`src/kit/skills/_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution** — the
  "Harness-adapter boundary (Claude Code)" paragraph currently says "the orchestrator maps the
  resolved id to the alias at the dispatch seam." Repoint to: resolve the model half with
  `fab resolve-agent <stage> --alias`, which emits an Agent-tool-valid alias on the `model=` line;
  pass it straight into the Agent `model` param (empty ⇒ omit/inherit). (Canonical instruction.)
- **`src/kit/skills/fab-ff.md`** (per-stage-model note, ~line 37) — currently "model via the Agent
  tool's `model` param". Repoint the resolve call to `fab resolve-agent <stage> --alias` for the
  model half.
- **`src/kit/skills/fab-fff.md`** (per-stage-model note ~line 37; ship-resolve ~line 47; review-pr
  resolve ~line 57) — same repointing at each `fab resolve-agent ...` call used for an Agent-tool
  dispatch.
- **`src/kit/skills/fab-continue.md`** (lines ~19, ~52, ~161) — the one-stage-sequencer note, the
  sub-agent dispatch contract, and the nested-reviewers note each resolve a stage for Agent-tool
  dispatch; repoint each to `--alias`.

**Effort half is unchanged.** #413 routes effort via a subagent-prompt instruction (the Agent tool
has no effort param). `--alias` touches only the `model=` line; the effort-prompt seam is left
exactly as #413 shipped it.

**Operator launcher path is NOT changed.** `fab operator` / the `_cli-fab.md` operator path
appends `--model <full-id>` to a `claude` **CLI** invocation, which accepts full IDs. It keeps
resolving **without** `--alias`. The CLI and Agent-tool paths deliberately diverge.

### 4. CLI reference — `_cli-fab.md`

Constitution constraint (CLI changes MUST update `_cli-fab.md`): update the `fab resolve-agent`
entry (§ around line 217) to document the `--alias` flag — what it emits (short alias on the
`model=` line), that it's a Claude-Code Agent-tool adapter, that default behavior is unchanged
(full ID), that the `effort=` line is unaffected, and that empty/non-Claude models pass through
verbatim.

### 5. Spec — `docs/specs/stage-models.md`

Update the adapter prose (§ Skill wiring and § Harness-adapter boundary) where #413 wrote "the
orchestrator maps the resolved id to the alias at the dispatch seam" — change it to describe the
`--alias` flag as the deterministic mechanism for the Agent-tool model half. The two
**drift-guarded tables** (default tier profiles, stage→tier mapping) are **NOT touched** — full
IDs stay canonical, so `TestDocTablesMatchAgentMaps` is unaffected.

### 6. Go test coverage

Add tests (test-alongside, `**/*_test.go`):
- `ModelAlias` unit tests: each of opus/sonnet/haiku/fable full IDs → alias; dated haiku variant →
  `haiku`; empty → empty; unmapped (`gpt-5`) → verbatim.
- `resolve-agent --alias` command-level test: `apply --alias` emits `model=opus` + `effort=high`;
  without `--alias` still emits `model=claude-opus-4-8` (regression guard for the unchanged
  default); a tier with empty model under `--alias` still emits an empty `model=` line.

## Affected Memory

- `pipeline/stage-models`: (modify) — the resolve-agent `--alias` flag and the two-surface
  (CLI vs Agent-tool) adapter split; supersedes the #413 hand-mapping instruction with a
  deterministic resolver flag. *(Exact memory file path confirmed at hydrate — the pipeline
  domain owns stage-models / dispatch wiring.)*

## Impact

- `src/go/fab/internal/agent/agent.go` — new `ModelAlias` function (+ its test file).
- `src/go/fab/cmd/fab/resolve_agent.go` — new `--alias` bool flag; apply mapping pre-format (+ test).
- `src/kit/skills/_preamble.md` — Harness-adapter boundary paragraph (repoint hand-map → `--alias`).
- `src/kit/skills/fab-ff.md`, `fab-fff.md`, `fab-continue.md` — per-stage-model dispatch notes
  (repoint each Agent-tool-dispatch resolve call to `--alias`).
- `src/kit/skills/_cli-fab.md` — resolve-agent signature gains `--alias`.
- `docs/specs/stage-models.md` — adapter prose (repoint hand-map → `--alias`); no drift-guarded
  table touched.
- **No change**: `defaultTiers` / `stageTiers` maps, the operator launcher path, `agent.tiers`
  config schema, the `model=`/`effort=` default output contract, the effort-prompt seam from #413.

## Open Questions

- (Resolved as a Confident assumption.) Unmapped / non-Claude model ID under `--alias`: pass
  through verbatim, not error — `--alias` is a best-effort Claude-Code adapter, not a validator.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix via a Go-side `--alias` flag on `fab resolve-agent`, replacing #413's prompt-side hand-mapping instruction | User chose this explicitly; the live failure was an agent fumbling the hand-map, and #413 currently ships exactly that hand-map as prose — determinism in Go removes the failure mode | S:95 R:80 A:92 D:95 |
| 2 | Certain | Do NOT switch tier defaults to aliases; full IDs stay canonical in `defaultTiers` + drift-guarded spec tables | User raised this alternative; rejected — breaks provider-neutrality, weakens the Fable version-pin, forces a coordinated multi-file edit pushing a harness quirk into the provider-neutral core | S:90 R:75 A:90 D:90 |
| 3 | Certain | Default behavior (no `--alias`) byte-identical to today (full model ID); CLI/operator path and the #413 effort-prompt seam unchanged | The `claude` CLI `--model` flag accepts full IDs; only the Agent-tool enum rejects them. `--alias` touches only the `model=` line | S:95 R:85 A:95 D:95 |
| 4 | Certain | This change EDITS the post-#413 adapter prose (repoints hand-map → `--alias`); it does not add new adapter documentation | #413 already wrote the id→alias prose into _preamble/ff/fff/continue/stage-models as a hand-map instruction; verified post-rebase. Editing-not-adding is the accurate scope | S:90 R:85 A:90 D:90 |
| 5 | Confident | Mapping is prefix-based (`claude-haiku-` → `haiku`) so dated variants (`claude-haiku-4-5-20251001`) resolve | The Agent enum is family-level; full IDs carry version/date suffixes. Prefix match is the robust mapping | S:75 R:80 A:85 D:80 |
| 6 | Confident | Unmapped / non-Claude model under `--alias` passes through verbatim (not an error) | Preserves provider-neutrality — `--alias` is a Claude-Code adapter, not a validator; a non-Claude override still flows. Low-risk and reversible | S:65 R:80 A:75 D:70 |
| 7 | Confident | `ModelAlias` lives in `internal/agent` (alongside the tier tables + `Resolve`) | That package already owns the model vocabulary and is the drift-guard's subject; cohesive home | S:75 R:85 A:90 D:80 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
