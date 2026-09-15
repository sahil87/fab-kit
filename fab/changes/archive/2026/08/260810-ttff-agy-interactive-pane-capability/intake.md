# Intake: agy-interactive-pane-capability

**Change**: 260810-ttff-agy-interactive-pane-capability
**Created**: 2026-08-10

## Origin

Conversational — raised during a `/fab-discuss` session on provider coupling and capability framing:

> By default all agents today definitely have an interactive mode. The fab-kit memory specifying that agy doesn't have an interactive mode — is outright incorrect. What an agent might or might not have is the headless mode. Also — because pane dispatch mode works using the "interactive_command", it implies that the dispatch mode "pane" should work with most agents.

Discussion confirmed the framing error and its staleness: the memory prose conflates "the CLI has an interactive mode" (universally true — that is what a TUI is) with "fab ships an `interactive_command` template for it" (a fab decision). The stated blocker for withholding agy's template — its first-run trust prompt parks a pane worker before delivery — predates the 3oz7 readiness gate, whose judgment rounds were built to clear exactly such walls (kimi's identical `Trust this folder?` wall is cleared this way today, one Enter, remembered per folder). `defaults.yaml` still says "Backlog [agik] owns that probe" although agik shipped (PR #564), and the rpsr probe additionally found agy's trust store is seedable (`~/.gemini/antigravity-cli/settings.json`, exact-match).

## Why

1. **Pain point**: agy is the sole built-in provider whose pipeline stages cannot ride pane workers — automatic resolution descends to headless (`pane unavailable: no interactive_command`) and `fab dispatch open` hard-errors — for a reason that no longer holds. Separately, the memory/spec prose teaches the wrong mental model ("agy has no interactive mode"), which misleads future provider work.
2. **Consequence of not fixing**: pane-mode parity stays artificially broken for one provider; the incorrect capability framing propagates into future provider onboarding decisions (the next provider gets judged by the wrong axis).
3. **Why this approach**: ship the template as built-in data (one `defaults.yaml` row — the same shape kimi's flip took in ki9v) and correct the framing at its sources, rather than leaving it as per-user config. The readiness gate is generic; no agy-specific code is needed.

## What Changes

### 1. Ship `providers.agy.interactive_command` in `defaults.yaml`

Add to the agy block in `src/go/fab/internal/agent/defaults.yaml`:

```yaml
interactive_command: 'agy --dangerously-skip-permissions --model {model}'
```

- Full-auto posture flag matches its own headless form (`--dangerously-skip-permissions`), the convention all built-ins follow.
- No `{effort}` placeholder — agy's model IDs embed the reasoning level as an ID suffix (existing rule, unchanged).
- No initial-prompt grammar needed: since 1lah/3oz7, the stage prompt is delivered *after* launch by `fab dispatch deliver` (pointer typed via tmux + echo-verified), so pane capability rides launch grammar alone.
<!-- assumed: exact interactive_command value — mirrors agy's shipped headless posture; the plan may adjust flags after checking `agy --help` against the installed binary -->
- Rewrite the surrounding comments: delete the "agy deliberately carries NO interactive_command" block and the stale "[agik] owns that probe" references; record the first-run trust wall as an ordinary readiness-gate judgment round (kimi precedent), noting the trust store is additionally user-seedable.

### 2. Flip the test pins

- `defaults_test.go`: replace the "agy carries **no** `interactive_command`" assertion with a pin **by value** of the shipped command (the same treatment kimi's `kimi --auto -m {model}` gets), so an edit cannot reach pane workers unnoticed.
- `agent.go`: add the `DefaultAgyInteractiveCommand` sibling if the per-name export pattern is kept for consistency with codex/kimi (or fold into whatever generic accessor exists by then).

### 3. Sweep the "dispatch-only" framing (the bulk of the change)

The claim "agy is dispatch-only / carries no interactive_command" is load-bearing prose in many places. Sweep class (grep `dispatch-only`, `no interactive_command`, `agik`):

- `src/go/fab/internal/agent/defaults.yaml` comments (part of item 1)
- `src/go/fab/internal/configref/configref.go` — rendered reference prose ("codex and kimi carry pane + …; agy is dispatch-only", the agy block comment, capability enumeration lines)
- `docs/specs/stage-models.md` — "**One of the four is dispatch-only**" section rewritten: all four built-ins are pane-capable; the per-provider open question is first-run behavior, now answered for agy
- `docs/memory/runtime/providers-and-profiles.md` — § "The dispatch-only built-in" replaced (see item 4)
- `docs/memory/runtime/dispatch.md` — the "no-interactive_command shape is a **shipped** configuration" example loses its only instance; rephrase as a user-config possibility, not a shipped one
- `docs/memory/runtime/agent-primitives.md` — agy entry's pane caveat replaced with the shipped grammar
- `src/kit/skills/_cli-fab.md` — any dispatch-only/agy-capability restatements
- Fixture files under test data that pin rendered reference output (tests will name them)

### 4. Reframe the capability model in memory

Write the corrected framing where the wrong one lived (`providers-and-profiles.md` primarily): **every agent CLI has an interactive mode — pane capability is the default expectation for any provider, because it rides launch grammar plus the generic readiness gate; headless grammar is the capability that varies per CLI and needs probing.** An absent `interactive_command` in a user's `providers:` block remains a valid configuration the descent ladder handles; it is just no longer a shipped state.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) § The dispatch-only built-in replaced with the pane-by-default framing; agy entry gains its shipped interactive_command + trust-wall note
- `runtime/dispatch`: (modify) descent-ladder example no longer cites agy as the shipped no-interactive_command instance
- `runtime/agent-primitives`: (modify) agy grammar entry updated — interactive form recorded, pane caveat dropped

## Impact

- `src/go/fab/internal/agent/defaults.yaml`, `defaults_test.go`, `agent.go` (per-name export)
- `src/go/fab/internal/configref/configref.go` + any pinned rendered-reference fixtures
- `docs/specs/stage-models.md`
- `src/kit/skills/_cli-fab.md` (if it restates the roster's capabilities)
- Three runtime memory files (above)
- No migration: built-in defaults are embedded data; user configs are untouched (presence=intent means a user's own `providers.agy.*` overrides still win per-field)

**Verification constraint**: agy quota is exhausted until ~2026-08-16, so a live pane-worker run cannot be part of this change's verification. Tests (pin-by-value, gate fixtures) and the kimi-precedent gate mechanics carry verification; a live probe is a recorded follow-up.

## Open Questions

- None — decisions were settled in the originating discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship agy's `interactive_command` as built-in data in `defaults.yaml`, not user-config guidance | User-stated in discussion; identical shape to kimi's ki9v flip | S:90 R:85 A:90 D:90 |
| 2 | Certain | First-run trust wall is handled by the existing readiness-gate judgment rounds — no agy-specific code | Generic mechanism; kimi's identical wall is cleared this way in production today | S:85 R:80 A:90 D:85 |
| 3 | Confident | Command value `agy --dangerously-skip-permissions --model {model}` (no effort/prompt placeholders) | Mirrors agy's own shipped headless posture; effort rides the ID suffix; delivery is post-launch since 1lah. Plan verifies flags against the installed CLI | S:70 R:85 A:75 D:70 |
| 4 | Confident | No trust-store seeding shipped — the gate suffices; seeding stays a user-side optimization noted in memory | rpsr probe found seeding possible, but gating is the generic path and needs no per-provider file format knowledge | S:65 R:90 A:80 D:75 |
| 5 | Confident | Live agy pane verification deferred past quota reset (~2026-08-16); change verifies by tests + fixtures | Quota exhaustion is external; pin-by-value test covers the shipped grammar; follow-up recorded | S:75 R:80 A:70 D:70 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).
