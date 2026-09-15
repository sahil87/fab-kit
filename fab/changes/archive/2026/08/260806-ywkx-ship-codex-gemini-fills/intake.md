# Intake: Ship Built-in Codex/Gemini Per-Role Fills

**Change**: 260806-ywkx-ship-codex-gemini-fills
**Created**: 2026-08-06

## Origin

Change 3 of a three-change series (2j2i → j9nh → ywkx) designed in a `/fab-discuss` session on 2026-08-06 with fab-kit's owner. Depends on j9nh (the `providers.<name>.profiles` per-role fill shape). The user's directive, verbatim:

> Now for the codex config, I want fab-kit to be able to use sensible defaults if the user asks to use codex.

This change is deliberately separate from j9nh because it **reverses a recorded design decision** and the reversal rationale should live in its own artifact: change `ho9y` established "no new built-in providers in Go"; `260805-j3cm` reversed it narrowly for grammar strings only, explicitly keeping model IDs out ("non-claude model IDs rot at CLI cadence, so they belong in config, not in a release"). This change reverses it fully — fab-kit ships model/effort fills for codex and gemini.

## Why

After j9nh, the flagship UX is `agent.workers: codex` (or `gemini`) — one line to move all pipeline workers to another provider. Without shipped fills, that line resolves an **empty model for every role**: the placeholder token drops and the provider CLI's own configured default applies, identically for doing/review/hydrate/fast. That "works" but silently defeats the entire point of role differentiation — pro-grade models for apply/review, a flash-grade model for ship — and makes the sensible-defaults promise hollow exactly where a user first exercises it.

Why the rot argument no longer wins:

1. **Release cadence**: fab-kit ships every few days (v2.16.x); the managed fence regenerates on every `fab config upgrade`, so users see refreshed suggestions at kit cadence, not CLI cadence.
2. **Loud, cheap failure**: a stale model ID fails immediately at the provider CLI with an obvious one-line config override as the fix (`providers.codex.profiles.default.model`).
3. **The alternative fails silently**: empty-model resolution degrades role differentiation with no error at all — worse than a loud stale ID.
4. **Data, not code**: after 2j2i the fills are rows in `defaults.yaml` — a bump is a data diff, reviewable at a glance.

## What Changes

### 1. Fill rows in `defaults.yaml` (the only functional change)

Add `profiles:` maps to the codex and gemini provider entries. Proposed values — **the exact model IDs and effort levels MUST be verified against current codex/gemini CLI documentation at apply time**; fab passes values through verbatim and validates nothing:

```yaml
providers:
  codex:
    session_command: '…(unchanged)…'
    dispatch_command: '…(unchanged)…'
    profiles:
      default: { model: gpt-5.3-codex, effort: high }
      doing:   { effort: xhigh }          # model inherits from this provider's default role
      review:  { effort: xhigh }
      fast:    { model: gpt-5.3-codex-mini, effort: low }
  gemini:
    session_command: '…(unchanged)…'
    dispatch_command: '…(unchanged)…'
    profiles:
      default: { model: gemini-2.5-pro }   # no effort — gemini CLI has no reasoning-effort flag
      fast:    { model: gemini-2.5-flash }
```

Roles absent from a provider's map fall through to that provider's `default` role per j9nh's precedence — codex `hydrate`/`operator` resolve `gpt-5.3-codex`/`high`; gemini non-fast roles resolve `gemini-2.5-pro`.

### 2. Refresh-each-release policy (documented, not automated)

- `docs/specs/stage-models.md`: a short section stating the policy — built-in non-claude fills are refreshed at kit-release cadence, are unvalidated pass-through values, and are overridden by one config line; note the ho9y → j3cm → this-change decision lineage.
- Fence comments (`fab config reference` rendering): the codex/gemini blocks show the shipped fills with a one-line "override to pin a newer model" note. Per j9nh's advertise demotion these render in `fab config reference` output, not in every project's fence — no fence size regression.

### 3. Tests

- Extend the embedded-defaults validation test (from 2j2i): codex/gemini profile rows parse, roles are valid role names, gemini rows carry no effort.
- Extend the pinned-profile drift guard: the shipped fills join the "deliberate change" pin so a bump is always an explicit test-acknowledged edit.
- Resolution tests: `agent.workers: codex` end-to-end — each stage resolves the expected model/effort through the j9nh precedence chain.

## Affected Memory

- `runtime/providers-and-tiers`: (modify) built-ins are no longer grammar-only; document the fills + refresh policy (file may be `runtime/providers-and-profiles` after j9nh's hydrate)
- `_shared/configuration`: (modify) providers section — shipped fills, override pattern

## Impact

- `src/go/fab/internal/agent/defaults.yaml` + tests (validation, pin, resolution)
- `docs/specs/stage-models.md`, `docs/specs/config.md` (registry note if fills surface there), memory files above
- No CLI change, no schema change, no migration (additive data)
- Smallest change of the series; the substantive content is the recorded decision reversal

## Open Questions

*(none — design settled in discussion; model-ID verification is an apply-time task, tracked as a Tentative assumption)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship fills as embedded `defaults.yaml` data (not config-seeded into user projects) | Series design — seeding live values into projects would pin rot in place; binary data refreshes on upgrade | S:85 R:80 A:90 D:85 |
| 2 | Confident | Reversing ho9y/j3cm's no-shipped-fills stance is correct | Discussed explicitly with rationale (cadence, loud failure, silent alternative); user's directive requires it | S:80 R:70 A:80 D:75 |
| 3 | Confident | Exact model IDs (gpt-5.3-codex, gpt-5.3-codex-mini, gemini-2.5-pro, gemini-2.5-flash) and effort levels | Placeholders from discussion (easily corrected — one data row); MUST be verified against current codex/gemini CLI docs at apply time — fab validates nothing | S:50 R:80 A:45 D:50 |
| 4 | Confident | Sparse per-role maps with default-role fallthrough (not all six roles enumerated per provider) | j9nh's precedence makes fallthrough well-defined; enumerating all roles would just duplicate the default row | S:70 R:80 A:85 D:80 |
| 5 | Confident | Policy is documentation-only (no staleness automation, no CI check against provider APIs) | Keeping fab validation-free is an established design stance ("compatibility is the harness's concern") | S:65 R:80 A:85 D:80 |

5 assumptions (1 certain, 4 confident, 0 tentative, 0 unresolved).
