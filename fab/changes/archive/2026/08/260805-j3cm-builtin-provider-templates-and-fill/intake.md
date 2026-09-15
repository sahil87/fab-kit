# Intake: Built-in Provider Templates + Per-Provider Fill + Invocation Overrides

**Change**: 260805-j3cm-builtin-provider-templates-and-fill
**Created**: 2026-08-05

## Origin

> Promote codex/gemini invocation templates to live Go built-in providers (grammar only), add per-provider default model/effort fill config, and invocation-time provider/model overrides on the resolution surface

Conversational origin (`/fab-discuss` session, 2026-08-05) — Phase 3 of the agent-orchestration series (Phase 1 = nvad `_cli-agents` helper + `fab agent --provider`, merged as PR #523; Phase 2 = zxe0 interactive pane dispatch, in flight). The user articulated the target model in three layers: (1) config.yaml knows the string templates to invoke gemini/codex/claude, (2) an extra set of config fills those templates, (3) so a session asked to "work with codex" has a good base to start from. Mapping onto the existing architecture: layer 1 = the `providers:` table with codex/gemini promoted from inert commented template text to live built-ins; layer 2 = `agent.tiers` (exists) **plus a new per-provider default-fill slot**; layer 3 = invocation-time override flags on the resolution surface.

## Why

1. **The pain point**: fab can only resolve the `claude` provider with zero config. The codex/gemini invocation strings are already inside the binary (as `configref` template text rendered into every config fence) but are inert until a user hand-uncomments them into `providers:` — so `agent.tiers.doing: {provider: codex, ...}` fails with "unknown provider" on a fresh project, `fab agent --provider codex` (shipped in nvad) errors, and "use codex for this stage" requires config surgery first. Additionally, a tier that sets `provider: codex` but omits `model` inherits a **claude** model through default-tier per-field inheritance — the documented cross-provider footgun fab chose to document rather than fix, because there was previously no correct value to fill with.

2. **The consequence of not fixing it**: cross-provider work stays config-gated at exactly the moment it should be frictionless (mid-session "work with codex"); the footgun stays armed; and every project (or machine) re-derives the same three template strings fab already ships as text.

3. **Why this approach**: promote the *grammar* (invocation templates) to Go built-ins while keeping *volatile fill values* (model IDs) in user config — grammar changes rarely (binary release cadence is fine); model IDs rot in weeks and must never be baked into the binary for non-claude providers. The per-provider fill slot gives "work with codex" a user-pinned base and provides the correct value that kills the cross-provider inheritance footgun. The config cascade (project > system `~/.fab-kit/config.yaml` > defaults, already shipped) means fill values are naturally settable once per machine.

## What Changes

### 1. Live built-in providers: codex and gemini (grammar only)

`defaultProviders` in `internal/agent` gains `codex` and `gemini` entries carrying exactly the shipped starter-template command strings:

```go
// grammar only — NO model/effort fill values for non-claude built-ins
"codex":  {SessionCommand: "codex -m {model} -c model_reasoning_effort={effort}",
           DispatchCommand: "codex exec -m {model} -c model_reasoning_effort={effort}"},
"gemini": {SessionCommand: "gemini -m {model}",
           DispatchCommand: "gemini -m {model}"}, // no {effort}: gemini CLI has no reasoning-effort flag; no -p: fab dispatch pipes prompt to stdin
```

- `agent.ResolveProvider` behavior is unchanged (project `providers:` per-field-merges over built-ins); the built-in table just has three rows instead of one.
- **Deliberate consequence**: `codex`/`gemini` become resolvable by *naming* them (a tier override or `--provider codex`) with zero `providers:` block. Note the claude built-in carries no `dispatch_command` (native dispatch); codex/gemini built-ins DO carry one — naming them in a tier flips those stages to CLI dispatch, which is exactly what selecting a non-claude provider means.
- This **explicitly reverses** the ho9y decision "no new built-in providers are added in Go — codex/gemini are template text only." The reversal is user-directed and narrow: grammar strings only. The `fab config reference` fence text and the scaffold's starter template are regenerated to present codex/gemini as built-in defaults (commented reference-style like every other default) rather than as uncomment-to-opt-in blocks; the per-provider notes (gemini no-`{effort}`/no-`-p`, codex stdin) move with them.

### 2. Per-provider default fill: `providers.<name>.model` / `providers.<name>.effort`

Two new optional per-provider config fields — the "extra set of config that fills the templates":

```yaml
# ~/.fab-kit/config.yaml (system scope — set once per machine) or project config.yaml
providers:
  codex:
    model: gpt-5.3-codex     # default fill when no tier/flag supplies one
    effort: high
```

- Go built-ins carry **no** fill values for codex/gemini (rot); claude's built-in fill continues to live where it does today — on the built-in tiers.
- Registered in the config field registry (`[]Field`) with scope `both`, advertised alongside `providers` command fields; rendered by `fab config reference`; no migration (purely additive fields).
- **Fill precedence** (final, applied at resolution time): **invocation flag > explicit tier field > provider default fill > empty** (empty → the existing `WithProfile` token-drop → the CLI's own default).
- **Cross-provider inheritance fix**: when a tier *explicitly sets* `provider:` but leaves `model`/`effort` unset, the unset fields SHALL fill from the named provider's default fill (then empty) — **not** from default-tier/built-in-tier per-field inheritance, which would supply another provider's model. A tier that does not set `provider:` inherits exactly as today (no behavior change for the all-claude default world, where every built-in tier carries an explicit model anyway). The footgun note in docs shrinks to: "override a tier cross-provider and the model fills from that provider's default; pin `model:` on the tier to be explicit."

### 3. Invocation-time overrides on the resolution surface

`fab resolve-agent <stage|tier>` gains `--provider <name> [--model <id>] [--effort <level>]`:

- `--provider` swaps the resolved profile's provider (and re-derives `dispatch=` from the named provider's `dispatch_command`); `--model`/`--effort` override the corresponding fields; unset override fields follow the fill precedence above (tier explicit field is NOT retained for a swapped provider's model — a cross-provider swap fills from the new provider's default fill, then empty; same rule as §2).
- `--alias` interplay: unchanged semantics — best-effort claude-prefix mapping; a non-claude overridden model passes through verbatim on `model=`, and `dispatch=` always embeds the full ID.
- `--model`/`--effort` without `--provider` remain valid here (they override within the resolved tier's provider) — unlike `fab agent`, where the requires-`--provider` rule stands (session launcher vs pure query; document the asymmetry).
- **Skill wiring (minimal)**: `_preamble.md` § Per-Stage Model Resolution gains one paragraph: when the user directs a provider/model for specific stages ("run review on codex"), the dispatch site passes the override flags on its existing single `resolve-agent` call — the seam, branch-on-`dispatch=`, and compliance-visibility rules are unchanged. No new dispatch machinery.

### 4. Sweep class (declared up front)

- Go: `internal/agent` (defaultProviders, resolution precedence), `internal/config` (+ registry fields), `internal/configref` (fence text: codex/gemini blocks move from opt-in template to built-in-default presentation; per-provider fill fields documented), `cmd/fab/resolve_agent*.go` (override flags) — all with tests (precedence table cases, cross-provider fill, override flags, fence goldens).
- `src/kit/skills/_cli-fab.md` — § fab resolve-agent (new flags, precedence), § providers/agent notes.
- `src/kit/skills/_preamble.md` — § Per-Stage Model Resolution override paragraph.
- `src/kit/skills/_cli-agents.md` — dictionary cross-reference: "codex/gemini are built-in providers; fill via providers.<name>.model" replaces any uncomment-first phrasing.
- `docs/specs/stage-models.md` (precedence + built-ins), `docs/specs/config.md` (registry rows, scope), SPEC mirrors (`SPEC-_preamble.md`, `SPEC-_cli-fab.md`, `SPEC-_cli-agents.md`), aggregate specs (`skills.md`, `glossary.md`, `architecture.md`) where they restate the one-built-in-provider or uncomment-to-opt-in model.
- `fab/project/config.yaml` fence in this repo regenerates via `fab config upgrade` at release time (not hand-edited here beyond what tests require).

### Non-goals

- No baked model IDs in Go for codex/gemini (built-ins are grammar-only).
- No semantic provider branching in Go ("if codex then…") — provider names stay opaque; runtime differences stay absorbed by template strings + `WithProfile`.
- No named tier-profile sets (`agent.profiles.*` switching whole tier maps per run) — deferred until per-stage overrides prove insufficient.
- No validation that a named provider's binary exists (document-don't-validate stands; errors surface at spawn/dispatch).
- No change to the fixed stage→tier mapping, the dispatch five-state machine, or zxe0's pane adapter (orthogonal).

## Affected Memory

- `runtime/providers-and-tiers.md`: (modify) three built-in providers, per-provider fill fields, fill precedence, cross-provider inheritance fix, resolve-agent override flags
- `_shared/configuration.md`: (modify) `providers` schema gains model/effort fill fields + registry/scope rows; ho9y reversal recorded as a Design Decision
- `_shared/context-loading.md`: (modify) Per-Stage Model Resolution override paragraph mirror
- `distribution/kit-architecture.md`: (modify) only if it restates the one-built-in-provider claim (grep-verify)
- `runtime/agent-primitives.md`: (modify) dictionary phrasing: built-in providers replace uncomment-first instructions

## Impact

- `src/go/fab/`: `internal/agent`, `internal/config`, `internal/configref`, `cmd/fab/` (resolve-agent flags) + tests
- `src/kit/skills/`: `_cli-fab.md`, `_preamble.md`, `_cli-agents.md`
- `docs/specs/`: `stage-models.md`, `config.md`, skills SPEC mirrors, aggregates as swept
- No migration (additive config fields; no user-data restructuring)
- Orthogonal to zxe0 (pane dispatch consumes resolved profiles unchanged); builds on nvad's shipped `--provider` seam

## Open Questions

*(none — the three-layer direction, grammar-only built-ins, and the fill/override split were stated by the user in the originating discussion; residual details are graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Promote codex/gemini to live Go built-in providers, grammar strings only | User-stated direction ("teach config.yaml what string templates to use to invoke gemini, codex, claude") | S:90 R:75 A:90 D:90 |
| 2 | Certain | No model IDs in Go built-ins for codex/gemini; fill values live in user config (per-provider fields + tiers + flags) | User-stated layer 2 ("extra set of config to fill these templates"); model IDs rot at CLI cadence, grammar at binary cadence | S:85 R:80 A:90 D:85 |
| 3 | Confident | Fill precedence: invocation flag > explicit tier field > provider default fill > empty (token-drop → CLI default) | Follows flag>config>default convention; each layer is the more specific intent | S:70 R:75 A:85 D:80 |
| 4 | Confident | Cross-provider inheritance fix: a tier explicitly setting `provider:` fills unset model/effort from that provider's defaults, never from another provider's tier inheritance | Kills the documented footgun with the value the user's layer-2 config now supplies; all-claude default world byte-unchanged (built-in tiers all pin models) | S:65 R:70 A:85 D:80 |
| 5 | Confident | Overrides land on `fab resolve-agent` (the resolution surface skills already call), not as new dispatch machinery; `_preamble` gains one passthrough paragraph | Layer 3 ("flexibility during stage invocation time") with minimal seam change; dispatch sites already make exactly one resolve call | S:70 R:75 A:85 D:75 |
| 6 | Confident | ho9y reversal is recorded explicitly (Design Decision in plan + memory), and the config fence/scaffold re-present codex/gemini as built-in defaults rather than opt-in blocks | Presence=intent is preserved for *behavior*: a built-in provider is inert until a tier or flag names it; the reversal is documented, not silent | S:65 R:70 A:80 D:75 |
| 7 | Confident | `fab agent`'s `--model/--effort` require-`--provider` rule stands (asymmetry with resolve-agent documented) | nvad shipped that rule deliberately (usage-error over silent tier override); resolve-agent is a pure query where within-tier override is meaningful | S:60 R:80 A:80 D:75 |
| 8 | Tentative | Named tier-profile sets (`agent.profiles.*`, whole-map per-run switching) stay out of scope <!-- assumed: per-stage override flags suffice for "use codex for the next N stages"; profiles deferred until real friction --> | The orchestrator can pass the same override to N dispatches; a profiles layer is additive later if switching whole sets becomes frequent | S:55 R:85 A:70 D:60 |

8 assumptions (2 certain, 5 confident, 1 tentative, 0 unresolved).
