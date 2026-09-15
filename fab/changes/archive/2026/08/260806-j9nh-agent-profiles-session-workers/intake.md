# Intake: Agent Profiles Reshape + Session/Workers Knobs

**Change**: 260806-j9nh-agent-profiles-session-workers
**Created**: 2026-08-06

## Origin

Change 2 of a three-change series (2j2i → j9nh → ywkx) designed in a `/fab-discuss` session on 2026-08-06 with fab-kit's owner. Depends on 2j2i (embedded `defaults.yaml` substrate). The design was reached iteratively; the pivotal user directives, verbatim:

> Should we maintain a tierxmodel mapping for all providers? Then the user can just nudge the main agent to use say.. "gemini" subagents. Then fab would be able to map which gemini agent to use for what tier.

> Also I want some clarity and nomenclature on the "tier" or the "level" of the agent we are talking about. Tier 1: Agents that user talks to. This could be the operator. Or another agent started with "rk riff" or "fab agent". […] Then there's Tier 2. These are agents the user generally doesn't talk to. Eg: the apply or the review agent within a fab workflow.

> The amount of configuration being provided - the user is barely going to change it. What we need to provide are sensible defaults. Where's the setting that allows me to say "claude for tier1, gemini for tier2"?

> does this change also reduce the size of the config.yaml presented to the user?

Key decisions settled in the discussion: rename the config concept "tiers" → **profiles**, freeing "tier" for agent **depth** (Tier 1 / Tier 2); per-role fill maps live on **providers**; the advertised config surface is exactly **two knobs** (`agent.session`, `agent.workers` — the user confirmed these names over `tier1`/`tier2`); the provider-name cutoff rule and the flat provider fill are removed; the fence's agent section shrinks to ~a dozen lines. Related prior art: change `260719-g55d` created the current six-tier taxonomy (this change evolves its *config surface*, not its stage→role mapping); backlog `[xz4f]` records the standing complaint that "the current breakup of tiers isn't working".

## Why

Four problems, one reshape:

1. **"Tier" is overloaded.** It means both the named `{provider, model, effort}` config profiles (`agent.tiers`, stage-models.md "role tiers") and agent depth (`_preamble.md` recovery policy: "Tier 2 (orchestrator → stage worker)"). The user thinks in depth terms; the config speaks role terms.
2. **The config surface is mis-pitched.** Users will not hand-tune six role profiles; they want "claude for what I talk to, gemini for the workers." Today that intent has no expression shorter than six tier overrides.
3. **The flat provider fill defeats tiering on provider swap.** A provider carries one `model`/`effort`, so `provider: codex` on any tier resolves the *same* model for every role — which is why the awkward provider-name cutoff rule exists at all.
4. **The fence is bloated.** ~90 of ~145 comment lines in every project's `config.yaml` explain provider/tier machinery almost nobody overrides.

The depth split is also mechanically real, which is why it's the right knob axis: `session` applies at **launch time** (fab cannot switch a running session's provider), while `workers` applies at **every stage dispatch** (`fab resolve-agent`).

## What Changes

### 1. Two advertised knobs: `agent.session` / `agent.workers`

```yaml
agent:
  session: claude    # Tier 1 — what the user talks to: fab agent, fab operator, fab batch
  workers: gemini    # Tier 2 — what pipeline stages dispatch to: apply, review, hydrate, ship, review-pr
```

Both default to `claude`; a fresh install writes nothing. Role partition (fixed, fab-owned): `default`, `operator` → session; `doing`, `review`, `hydrate`, `fast` → workers.

### 2. `agent.tiers` → `agent.profiles` (rename + semantics change)

- Registry `renamed_from` carry-forward, so old configs keep resolving.
- New semantics: a **sparse per-role override** table — `agent.profiles.<role>.{provider, model, effort}`, each field optional, beating the knob/provider-fill rungs below.
- **Removed**: the agent-side `default`-role inheritance. `agent.profiles.default` is just the `default` role's override, NOT a fallback source for other roles. (Today `agent.tiers.default.model: X` re-bases every tier that doesn't set its own model.) Cross-role fallback now lives on the provider side only — one fallback chain, not two competing ones.

### 3. Per-role provider fills: `providers.<name>.profiles`

- New nested map: `providers.<name>.profiles.<role>.{model, effort}` — "when this provider plays this role, use this model/effort."
- The claude built-in role models **move off the agent side onto the provider** in `defaults.yaml` (see §6).
- **Removed**: the flat `providers.<name>.model`/`.effort` fill — folded into `providers.<name>.profiles.default` (migration, §7).
- **Removed**: the provider-name cutoff rule. Swapping a role's provider naturally re-fills from that provider's own profile map; no special inheritance-loss rule remains to document.

### 4. Resolution precedence (the new chain, implemented in `fab resolve-agent`)

- **Provider**: invocation `--provider` flag → `agent.profiles.<role>.provider` → depth knob (`agent.session` or `agent.workers` by the role's partition) → built-in `claude`.
- **Model / effort (per field)**: invocation flag → `agent.profiles.<role>.<field>` → `providers.<prov>.profiles.<role>.<field>` → `providers.<prov>.profiles.default.<field>` → empty (placeholder token drops; the CLI's own default applies).
- CLI surface unchanged: `fab resolve-agent <stage|role> [--alias] [--provider|--model|--effort]`, same `model=`/`effort=`/`provider=`/`dispatch=` output lines, same `--alias` semantics, same `dispatch=` emission rules (`dispatch_command` presence; `dispatch.watchable` opt-in).

### 5. Launch-time wiring

`fab agent`, `fab operator`, and `fab batch` (the Tier-1 spawners) consult `agent.session` when composing spawn commands (`spawn.WithProfile` path). Stage dispatch sites are untouched — they already call `fab resolve-agent <stage>`, which now consults `agent.workers` internally.

### 6. `defaults.yaml` reshaped to the new schema

```yaml
agent:
  session: claude
  workers: claude
providers:
  claude:
    session_command: '…(unchanged)…'
    profiles:
      default:  { model: claude-fable-5,  effort: high }
      operator: { model: claude-sonnet-5, effort: medium }
      doing:    { model: claude-opus-5,   effort: xhigh }
      review:   { model: claude-opus-5,   effort: xhigh }
      hydrate:  { model: claude-opus-5,   effort: high }
      fast:     { model: claude-sonnet-5, effort: medium }
  codex:  { …commands unchanged, no profiles yet — ywkx ships those… }
  gemini: { …commands unchanged, no profiles yet… }
```

Resolved defaults are byte-identical to today's for every stage (the six claude profiles carry the same values; they've only moved). Until ywkx lands, `workers: codex|gemini` resolves empty models (the provider CLI's own default applies) — accepted intermediate state.

### 7. Migration + registry

- `src/kit/migrations/` file (constitution: user-data restructuring ships as a migration): rewrites user configs — `agent.tiers:` → `agent.profiles:`; `providers.<p>.model/effort` → `providers.<p>.profiles.default.{model,effort}`. Also sweeps `~/.fab-kit/config.yaml` (system scope) per the established worktree/system-sweep lesson.
- Config registry: `renamed_from` on `agent.profiles`; new fields (`agent.session`, `agent.workers`, `providers.*.profiles`) registered with scope `both`.

### 8. Fence slimming (the user's explicit size requirement)

Advertise category A: `agent.session`, `agent.workers` (+ `dispatch.watchable` stays as-is). Demote the machinery (`agent.profiles`, `providers.*` including `profiles`) to a non-advertised category — visible in `fab config reference --json` and documented in `docs/specs/config.md`, not rendered per-project. Target: the fence's provider/agent commentary drops from ~90 lines to ~a dozen (the two knobs, one example, one pointer).

### 9. Nomenclature sweep

- Vocabulary: **role** = the six fixed slot names; **profile** = a `{provider, model, effort}` value; **provider** = grammar + per-role fills; **Tier 1 / Tier 2** = agent depth (session agents vs. dispatched stage workers). A watchable pane worker is still Tier 2 — the defining property is "owes a result artifact and owns no transitions", not "never spoken to".
- Go identifiers follow (`TierDefault` → role constants, `defaultTiers` → parsed profiles, `TierForStage` → `RoleForStage`, etc.) — internal, no compat surface; the six role *strings* are unchanged, so `fab resolve-agent review` keeps working.
- Doc sweep (the full SPEC-mirror class, swept up front per code-quality.md § Sibling & Mirror Sweeps): `docs/specs/stage-models.md` (rewrite), `config.md`, `glossary.md`, `architecture.md`, `_preamble.md` § Per-Stage Model Resolution + § Subagent Dispatch, `_cli-fab.md` § resolve-agent + § config, `_cli-agents.md` (tier-addressed spawn prose), affected `docs/specs/skills/SPEC-*.md` mirrors, and the memory files listed below.

### 10. Effort-asymmetry note (rides along)

Add to `stage-models.md`: effort injected via subagent-prompt instruction is **not reliably honored** on the native Agent-tool arm (session-level effort dominates — Claude Code known limitation, GitHub issues #64033/#39220); effort differentiation is only trustworthy where `--effort` rides a composed CLI command (`fab dispatch` headless/pane, operator launcher). Model differentiation works on both arms.

## Affected Memory

- `runtime/providers-and-tiers`: (modify) full rewrite to the roles/profiles/knobs model — likely renamed to `runtime/providers-and-profiles` during hydrate
- `_shared/configuration`: (modify) schema section — knobs, profiles, removed cutoff/flat-fill, fence slimming
- `_shared/context-loading`: (modify) per-stage model resolution prose (precedence chain, nomenclature)
- `runtime/agent-primitives`: (modify) tier-addressed → role-addressed spawn composition; provider-form semantics
- `runtime/operator`: (modify) operator launcher consults `agent.session`
- `distribution/kit-architecture`: (modify) stale `fab resolve-agent <stage|tier>` / `fab agent [tier]` signatures and "provider/tier resolution" prose (:298, :305, :311) → the role/profile vocabulary and the two depth knobs
- `runtime/index` + `_shared/index`: (regenerate) the row descriptions for the files above are regenerated by `fab memory-index` at hydrate, not hand-edited

## Impact

- Go: `internal/agent` (resolution rewrite, renames), `internal/config` (registry: renames + new fields), `internal/configupgrade` (fence rendering + advertise demotion), `cmd/fab` (resolve-agent, agent, batch, operator launcher), `internal/spawn` consumers; full test updates alongside (constitution: CLI change ⇒ `_cli-fab.md` + tests)
- Kit: `src/kit/skills/` (`_preamble`, `_cli-fab`, `_cli-agents`, any skill restating tier prose), new `src/kit/migrations/` file
- Specs: stage-models (rewrite), config, glossary, architecture, skills/SPEC-* mirrors
- Memory: the files above
- Largest change of the series; review risk concentrates in the doc sweep (SPEC-mirror class)

## Open Questions

*(none — design settled in discussion)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Knob names `session`/`workers` (not `tier1`/`tier2`) | User: "session/workers definitely better" | S:95 R:85 A:95 D:95 |
| 2 | Certain | Rename `agent.tiers` → `agent.profiles`; "tier" freed for depth (Tier 1/Tier 2) | User requested the nomenclature clarity; rename approach discussed and accepted | S:90 R:70 A:90 D:85 |
| 3 | Certain | Advertised surface = the two knobs only; fence agent section shrinks to ~a dozen lines | User: "the user is barely going to change it… what we need are sensible defaults"; size reduction explicitly confirmed as a goal | S:90 R:80 A:90 D:90 |
| 4 | Confident | Remove agent-side `default`-role inheritance (single fallback chain, provider side only) | Flagged to the user as the one behavior change to sanity-check; no objection; it is what makes the cutoff-rule removal coherent | S:70 R:60 A:80 D:70 |
| 5 | Confident | Dual back-compat: registry `renamed_from` (read-time) + migration file (rewrite) | Both mechanisms exist and are the established pattern (config.md, constitution migration rule) | S:70 R:75 A:85 D:80 |
| 6 | Certain | Role partition default/operator → session; doing/review/hydrate/fast → workers | Derives from launch-time vs dispatch-time mechanics discussed; intake runs foreground (advisory) under default, which is a session-side role | S:75 R:80 A:85 D:80 |
| 7 | Confident | `fab resolve-agent` CLI surface unchanged (names, flags, output lines) | Dispatch-seam skills parse these lines; keeping them stable avoids touching every dispatch site | S:70 R:65 A:85 D:80 |
| 8 | Confident | Exact advertise category for demoted fields (B vs C) and exact fence wording | Registry categories exist but the right B/C split is an apply-time judgment against config.md's category definitions | S:55 R:85 A:70 D:55 |
| 9 | Certain | Go identifier renames follow the new vocabulary (Role*/profile terms) | Internal-only surface; six role strings unchanged so no CLI/positional compat break | S:65 R:80 A:90 D:80 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
