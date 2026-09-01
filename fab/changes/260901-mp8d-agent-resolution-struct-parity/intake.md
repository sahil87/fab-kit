# Intake: Shared agent.Resolution Struct + Parity Tests

**Change**: 260901-mp8d-agent-resolution-struct-parity
**Created**: 2026-09-01

## Origin

> [mp8d] 2026-09-01: fab agent unification Change 2/4: shared agent.Resolution struct + parity/golden tests per fab/plans/sahil/26-09-01-fab-agent-unification.md § Change 2 -- ZERO change to fab resolve-agent's surface or output. FULL lane. depends on Change 1.

Backlog-ID invocation via `/fab-fff mp8d` (one-shot; the orchestrator created this change because no change existed for the backlog ID). Design sources, in precedence order:

1. **The plan doc** `fab/plans/sahil/26-09-01-fab-agent-unification.md` § Change 2 (written 2026-09-01 from a /fab-discuss design session). §§ Changes 3–4 (backlog `u6es`/`0i4x`) are strictly-ordered successors and OUT of scope here.
2. **Change 1's shipped intake** (`fab/changes/260901-77vz-fab-agent-surface-extension/intake.md` § 6) — pins this change as the `-o yaml` **schema authority**: Change 1 shipped the minimal key set (`selector`, `kind`, `role`, `provider`, `model`, `effort`, `command`) and named the additive extensions this change owns (`model_alias`, `template`, `fill_mode`, `source`, `dispatch`), plus the `dispatch:` absence ⇔ native rule.

**Dependency satisfied**: Change 1 (77vz) merged to main 2026-09-01 13:52 UTC as PR #635 (`30a45836`); this branch starts from that commit. Code anchors below re-verified 2026-09-01 on this branch (`fair-butte`, main-equivalent at `30a45836`).

Lane directive: **FULL lane** (backlog entry says so explicitly; touches both commands' composition paths + tests + docs).

## Why

1. **The pain point**: resolution results are composed at **two independent sites** today. `cmd/fab/resolve_agent.go` composes profile + dispatch line and renders the frozen ordered `model=`/`effort=`/`provider=`/`dispatch=` lines (`formatAgentProfile` :238, `dispatchLineFor` :177); `cmd/fab/agent.go` composes profile + command and renders the minimal `agentResolutionYAML` doc (:221–229, :327). Both call the same `internal/agent` engine, but the *result assembly* (which fields, alias handling, dispatch derivation, command substitution) is duplicated CLI-side and can drift.
2. **The consequence if unfixed**: Change 3 migrates every skill dispatch site from `fab resolve-agent <stage> --alias` to `fab agent <stage> -o yaml`, treating the YAML as the contract that replaces the frozen lines. Without parity **by construction**, the migration rests on two hand-maintained renderers agreeing forever — the exact drift class (`fab/project/code-quality.md` § owner-or-pointer, but in Go) this repo's process exists to kill. A silent divergence after migration would mis-dispatch pipeline stages.
3. **Why this approach**: extract **one resolution-result struct** (`agent.Resolution`) and make both commands' outputs *projections* of it — resolve-agent's lines and `fab agent -o yaml`'s serialization. Parity then holds by construction and is *proven* by a parity test matrix plus golden tests pinning resolve-agent's exact bytes across the refactor. `fab resolve-agent`'s flags, arguments, and output are **never mutated** (plan non-goal #1; deprecation is Change 3's docs-only concern; deletion is post-window, out of this plan).

## What Changes

Go work in `internal/agent` (struct + projections), `cmd/fab/resolve_agent.go` and `cmd/fab/agent.go` (become renderers), plus tests. Docs are minimal: `_cli-fab.md` § fab agent gains the full `-o yaml` schema; § fab resolve-agent gains one line noting the shared engine (no contract change).

### 1. The `agent.Resolution` struct (internal/agent)

One struct holding the complete resolution result. Fields per the plan (working names — Go-idiomatic casing is the implementer's; YAML tags shown where serialized):

| Field | YAML key | Content |
|-------|----------|---------|
| Selector | `selector` | the addressing input (empty for bare `--provider`) |
| Kind | `kind` | `role` \| `stage` \| `provider` (Change 1's classification, extended by the bare-provider kind `fab agent` already reports) |
| Role | `role` | resolved role (empty for bare-provider kind) |
| Provider | `provider` | resolved provider name |
| Model | `model` | **full** model ID |
| ModelAlias | `model_alias` | Agent-tool short alias via `agent.ModelAlias` (:411); **empty for non-Claude IDs**; always emitted for Claude IDs — makes a separate `--alias` flag unnecessary on the YAML surface |
| Effort | `effort` | resolved effort |
| Template | `template` | the provider's raw command template (unsubstituted, placeholders intact — the `-t` tap's content) |
| FillMode | `fill_mode` | `template` \| `append` — which `spawn.WithProfile` mode the command composed under (`internal/spawn/spawn.go` `isTemplate` :114 is the discriminator) |
| Command | `command` | the substituted command line |
| Dispatch | `dispatch` | `nil`, or `{rung: pane\|headless, command: <substituted>}` |
| Source | `source` | provenance — which config tier / fill rung supplied each value (see assumption 3) |

Composition of this struct becomes **the only place** resolution results are assembled — the current per-command assembly in `resolve_agent.go` (profile → dispatch line → formatted lines) and `agent.go` (profile → slot → composed → YAML doc) collapses into projections. **No second composition site survives** (acceptance criterion).

Purity note: `$TMUX` stays read at the cobra layer and passed in (the `internal/dispatch.SelectMode` precedent, `resolve_agent.go:149`); the struct composer takes environment as parameters.

### 2. `fab agent -o yaml` schema lands in full (additive over Change 1)

Change 1's seven keys are **frozen as-is** (order and content); this change adds `model_alias`, `template`, `fill_mode`, `source`, and `dispatch` additively. Key semantics (plan § Change 2 item 2, verbatim intent):

- **`dispatch:` key absent ⇔ native rung** — preserves today's branch-on-presence rule (`dispatch=` line absent ⇔ native) for the consumers Change 3 migrates. Derivation reuses the same `dispatch.SelectMode` ladder resolve-agent uses (`dispatchLineFor` hoists/shares — no reimplementation).
- **`dispatch.rung` is labelled** (`pane` | `headless`) — the capability resolve-agent's unlabelled `dispatch=` line cannot carry; Change 4 cashes this in by collapsing `_preamble.md`'s "attempt `start` first" discovery choreography. Nothing in *this* change touches skill choreography.
- **`model_alias` always emitted for Claude IDs** (empty value for non-Claude), so the YAML surface needs no `--alias` flag.

Example (target shape, doing role under a pane-capable config inside tmux with `dispatch.mode: pane`):

```
$ fab agent apply -o yaml
selector: apply
kind: stage
role: doing
provider: claude
model: claude-opus-5
model_alias: opus
effort: high
template: claude --model {model} --effort {effort}
fill_mode: template
command: claude --model claude-opus-5 --effort high
dispatch:
    rung: pane
    command: claude --model claude-opus-5 --effort high
source:
    provider: agent.workers
    model: providers.claude.profiles.doing
    effort: providers.claude.profiles.doing
```

(Exact `source` vocabulary and YAML field ordering are apply-time decisions — see Assumptions; the `dispatch:`-absence and labelled-rung semantics are fixed.)

### 3. `fab resolve-agent` untouched — parity by projection

Zero change to `resolve_agent.go`'s flags, args, or emitted bytes. Internally it re-renders from the shared struct: the ordered `model=`/`effort=`/`provider=`/`dispatch=` lines become a **line-projection** of `agent.Resolution` (`--alias` maps the `model=` line via `model_alias`; `dispatch=` always embeds the full ID — existing contract, now guaranteed by projecting from the one struct).

### 4. Parity test matrix + golden tests

- **Golden tests first** (before the refactor): pin `fab resolve-agent`'s exact output bytes for the matrix below, so the refactor is provably byte-neutral.
- **Parity matrix**: assert resolve-agent's lines ≡ the line-projection of the struct across: templated vs plain (append-mode) commands; empty fills (the inherit signal — empty `model=` line); `--alias` on/off; non-Claude ID pass-through (`gpt-5` verbatim); dispatch present (pane rung, headless rung) and absent (native); dated Claude variants (`claude-haiku-4-5-20251001` → `haiku`).
- `fab agent` existing sinks stay byte-identical: `--print`, exec, `-t`, and Change 1's seven `-o yaml` keys (golden coverage exists in `cmd/fab/agent_test.go` from 77vz — extend, don't weaken).

### Constraints (must-not-break)

- `fab resolve-agent` output **byte-identical before/after** for the full matrix (the acceptance bar).
- `fab agent --print` stays byte-identical for every invocation legal today (operator spawn seam, `_cli-agents.md:46-58`; cross-repo `_cli-external.md` usage).
- No change to the operator launcher's resolution path (`operator.go` `WithProfile` direct), `fab pane open` / `fab batch` / `fab dispatch start` composition, or `spawn.WithProfile` semantics (template/append modes, empty-value token-drop).
- Bare `--provider` fill-bypass semantics stay load-bearing and untouched.
- No provider grammar validation anywhere (verbatim pass-through), including the new keys.

### Documentation (same change)

- `src/kit/skills/_cli-fab.md` § fab agent: document the full `-o yaml` schema (all twelve keys, `dispatch:`-absence rule, labelled rung, `model_alias` semantics). § fab resolve-agent: **one line** noting both commands render from the shared resolution engine — no contract change, no deprecation banner (that's Change 3).
- `docs/specs/stage-models.md`: surgical mention that the structured YAML surface now carries the full resolution (human-curated spec — minimal edit).
- **Sibling sweep obligation**: grep `-o yaml`, `agentResolutionYAML`, "minimal projection", "schema extends additively later", and "Change 2" phrase classes repo-wide — Change 1 left forward-pointing prose (e.g. `agent.go:60-62` comment "minimal in this change; the schema extends additively later", `_cli-fab.md`'s minimal-key list) that this change makes stale and must update in place (`fab/project/code-quality.md` § Sibling Sweeps).

## Affected Memory

- `runtime/providers-and-profiles`: (modify) documents `fab resolve-agent` and `fab agent` surfaces — gains the shared-Resolution-struct fact, the full `-o yaml` schema, and the labelled-rung capability
- `runtime/dispatch`: (modify) only if its resolve-agent/dispatch-line prose restates the composition mechanics this change relocates — verify at hydrate; no behavior it documents changes

## Impact

- **Go**: `internal/agent/agent.go` (+`agent_test.go`) — the `Resolution` struct, its composer, provenance tracking, and projections; `cmd/fab/resolve_agent.go` + `cmd/fab/agent.go` become thin renderers (their tests extended: golden bytes + parity matrix); possibly a small seam export in `internal/spawn` for fill-mode detection (see assumption 4). Existing consumers (`roleSessionCommand` :435, operator launcher, dispatch start) untouched.
- **Skills/docs**: `_cli-fab.md` (constitution constraint: any fab CLI change MUST update it + ship tests — both in scope), `docs/specs/stage-models.md`, sweep-discovered occurrences.
- **Memory**: the files listed in Affected Memory.
- **Downstream**: Change 3 (`u6es`) migrates skill prose onto the `-o yaml` schema this change fixes; Change 4 (`0i4x`) consumes the labelled rung. Nothing here may touch skill dispatch choreography or resolve-agent's surface.

## Open Questions

*(none — the plan doc § Change 2 plus Change 1's shipped intake resolve every surface-level decision; remaining choices are implementation-detail grade and recorded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope = plan § Change 2 exactly: shared `agent.Resolution` struct, both commands as projections, parity matrix + golden tests, ZERO resolve-agent surface/output change | Backlog entry + plan doc + Change 1 intake all pin this; non-goals explicit | S:95 R:90 A:95 D:95 |
| 2 | Certain | `-o yaml` extends additively with `model_alias`/`template`/`fill_mode`/`source`/`dispatch`; `dispatch:` absent ⇔ native; `rung:` labelled (`pane` or `headless`); `model_alias` always emitted for Claude IDs (empty for non-Claude) | Pinned verbatim by plan § Change 2 item 2 and 77vz intake § 6 | S:90 R:85 A:95 D:90 |
| 3 | Confident | `source` provenance = a per-field map (at minimum `provider`/`model`/`effort`) naming the supplying config tier or fill rung (e.g. `agent.profiles.<role>`, `agent.workers`, `providers.<p>.profiles.<role>`, `providers.<p>.profiles.default`, `flag`, `built-in`); exact key vocabulary is the implementer's, recorded in `_cli-fab.md`'s schema table | Plan names the field but not its schema ("which config tier / fill rung supplied each value"); additive key, easily extended later | S:55 R:75 A:75 D:60 |
| 4 | Confident | Fill-mode detection is exposed from `internal/spawn` (export the `isTemplate` discriminator or equivalent one-line seam) rather than duplicating placeholder-detection logic in `internal/agent` | code-quality anti-pattern: duplicating existing utilities; `spawn.go:114` owns the semantics | S:60 R:85 A:85 D:75 |
| 5 | Confident | `fab agent -o yaml` derives `dispatch:` via the same `dispatch.SelectMode` path resolve-agent uses (`dispatchLineFor` hoisted/shared, `$TMUX` still read at the cobra layer and passed in) — one derivation site, pure composer | Plan's "no second composition site survives" acceptance + the SelectMode purity precedent at `resolve_agent.go:149` | S:65 R:80 A:85 D:75 |
| 6 | Confident | Change 1's seven `-o yaml` keys keep their exact names, values, and relative order; new keys append without reordering, so any early consumer of the minimal doc keeps parsing | 77vz intake promises "extends additively"; YAML consumers key-address, but order stability is the conservative read of "additive" | S:70 R:80 A:80 D:75 |
| 7 | Confident | Bare-provider kind (`fab agent --provider X -o yaml`) also emits `dispatch:`, derived from that provider's capabilities — consistent with resolve-agent's `--provider` re-derivation. (`-t` and `-o` are already mutually exclusive per Change 1's guards, so no template-sink interaction arises; the `template:` key carries the raw slot on every kind.) | Plan is silent on this corner; consistency with resolve-agent's re-derivation is the one obvious default, and tests make it cheap to revise at review | S:40 R:70 A:60 D:45 |

7 assumptions (2 certain, 5 confident, 0 tentative, 0 unresolved).
