# Intake: Providers Config Template — Three Providers Pre-Filled

**Change**: 260702-ho9y-providers-config-template
**Created**: 2026-07-03

## Origin

> By default, fill both session_command and dispatch_command for 3 providers: claude, codex and gemini. So the user has a template to work on.

Follow-up to `260702-tykw-agent-providers-role-tiers` (agent config v3 — providers + role tiers), which the user reports as already implemented. tykw introduced the `providers:` section with per-provider `session_command`/`dispatch_command`; its config reference showed only claude live and codex commented. This change upgrades the reference/scaffold presentation: **all three providers (claude, codex, gemini), each with both command fields written out**, so a user configuring a non-claude provider copies and adapts rather than composing command grammar from scratch. Requested in the same `/fab-discuss` session that designed tykw (conversational, one-shot instruction).

## Why

1. **Problem**: after tykw, a user who wants a second provider must author the provider block from nothing — they need to know each CLI's session-vs-headless subcommand split (`codex` vs `codex exec`), its model/effort flag grammar, and how `{model}`/`{effort}` placeholders work, before anything runs. The config reference showing only claude (and a codex stub) leaves the hardest part — correct command strings — as an exercise.
2. **Consequence if unfixed**: the cross-harness dispatch capability (the point of the 3a–3d series and tykw) stays practically claude-only for most users; misauthored command templates fail at dispatch time (fab validates nothing, by contract), which reads as "multi-provider doesn't work."
3. **Approach**: ship the knowledge as *template text* in the generated config reference (`fab config reference` / `configref.go`) and the new-project scaffold — complete, realistic command pairs for claude, codex, and gemini, commented-out where uncommenting changes behavior. This follows the reference's existing convention ("opt-in override blocks appear commented-out with defaults shown, so uncommenting is opting in") rather than shipping new built-in provider defaults in Go — templates are user-editable guidance; built-ins would embed third-party CLI opinions into the binary and go stale silently.

## What Changes

### 1. `configref.go` — the `providers:` block becomes a three-provider template

The generated reference's providers section is rewritten to (exact content, subject to the Tentative gemini verifications below):

```yaml
# providers — agent CLI invocation grammar, referenced by tiers via `provider:`.
# {model}/{effort} placeholders are substituted from the tier profile at resolve
# time (a claude session_command without placeholders gets --model/--effort
# appended). session_command opens an interactive session; dispatch_command runs
# one headless stage task via `fab dispatch` — ABSENT dispatch_command = native
# Agent-tool dispatch (the default, claude-family models only).
providers:
  claude:
    session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"'
    # dispatch_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'
    #   uncomment to run claude stages as headless CLI processes instead of
    #   native Agent-tool sub-agents (the default while absent)
  # codex:
  #   session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
  #   dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
  # gemini:
  #   session_command: 'gemini -m {model}'
  #   dispatch_command: 'gemini -m {model} -p'
```

Presentation rules:

- **All six command strings are present as text** (the user's ask: both fields, three providers) — but **anything whose uncommenting changes default behavior ships commented**: claude's `dispatch_command` (present → claude flips from native to CLI dispatch) and the entire codex/gemini blocks (opt-in providers). Claude's `session_command` appears live, mirroring the built-in default (harmless restatement, consistent with how baseline keys show live example values).
- **No new built-in providers in Go**: `claude` remains the only built-in provider; codex/gemini exist purely as commented template text until a user uncomments them into their project config.
- Gemini's commands carry no `{effort}` placeholder — the gemini CLI has no reasoning-effort flag; the empty-effort token-drop rule is not relied on, the placeholder is simply omitted.

### 2. New-project scaffold gets the same template

`src/kit/scaffold/fab/project/config.yaml` gains the identical commented `providers:` template block (claude live line included or fully commented — planner's call consistent with how the scaffold treats other baseline keys), so a fresh `fab init` project sees the three-provider template without running `fab config reference`.

### 3. Verification obligations at apply (recorded, not assumed silently)

- **Gemini CLI grammar** must be verified against current gemini CLI docs at apply time: model flag (`-m/--model`), headless invocation (`-p/--prompt` semantics), and session invocation. The strings above are the best current understanding, not verified fact. <!-- assumed: gemini CLI flags (-m model, -p headless) — to be verified against live gemini CLI docs during apply -->
- **Prompt-delivery conformance**: `fab dispatch` delivers the stage prompt per the harness-adapters contract; verify each template's `dispatch_command` actually receives the prompt that way (codex exec: stdin/arg; gemini `-p`: may require the prompt as the flag's argument, which could need a `{prompt}`-style seam or documented caveat). If a provider's grammar cannot conform, the template line ships with an explicit comment saying what to adapt.
- **Codex model example**: any model IDs shown in comments use current real IDs (e.g. `gpt-5-codex`) — cosmetic, verify at apply.

### 4. Docs and tests

- `_cli-fab.md` § `fab config reference` schema-coverage line updated (three-provider template).
- `configref` tests / golden output updated; scaffold conformance test if one exists.
- SPEC mirrors for any touched skill files; `architecture.md` config example extended only if it restates the providers block.

## Affected Memory

- `_shared/configuration.md`: (modify) config reference now ships a three-provider template (claude/codex/gemini, both commands each; commented opt-in presentation)
- `runtime/providers-and-tiers.md`: (modify) note the template block as the on-ramp for adding a provider (file created by tykw's hydrate; if it does not exist yet, hydrate creates it with this content folded in)

## Impact

- `src/go/fab/internal/configref/configref.go` + tests (primary surface)
- `src/kit/scaffold/fab/project/config.yaml`
- `src/kit/skills/_cli-fab.md` (constitution: CLI-adjacent doc)
- Spec mirrors as touched (`docs/specs/architecture.md` example if applicable)
- **Depends on**: `260702-tykw-agent-providers-role-tiers` merged (the `providers:` schema this templates). Small, single change — no series expected.
- **Out of scope**: new built-in provider defaults in Go; any change to resolution/dispatch behavior; validating provider commands (no-validation contract unchanged).

## Open Questions

- Should the scaffold show the claude `session_command` line live (restating the built-in) or fully commented? (Reference shows it live; scaffold minimalism may argue commented.)
- If the gemini CLI's headless mode cannot receive the dispatch prompt the way the harness-adapters contract delivers it, does the template ship with a caveat comment or is gemini dropped to session_command-only in the template?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Template text in reference + scaffold, NOT new built-in providers in Go | "So the user has a template to work on" reads as editable template; matches the reference's commented-opt-in convention; built-ins would embed volatile third-party CLI opinions in the binary. Easily revisited | S:70 R:85 A:80 D:70 |
| 2 | Certain | Claude's `dispatch_command` ships commented (native dispatch stays the default); codex/gemini blocks ship fully commented | Uncommenting must be the opt-in act — a live claude dispatch_command would silently flip claude stages from native to CLI dispatch | S:65 R:80 A:90 D:85 |
| 3 | Certain | Codex command grammar: `codex -m {model} -c model_reasoning_effort={effort}` (session) / `codex exec …` (dispatch) | Established in the tykw design discussion and existing configref example | S:80 R:85 A:85 D:85 |
| 4 | Confident | Gemini command grammar: `gemini -m {model}` (session) / `gemini -m {model} -p` (dispatch), no `{effort}` placeholder | Best current understanding of the gemini CLI; trivially reversible template text, but flags and headless prompt delivery MUST be verified against live docs at apply (§ 3) | S:45 R:80 A:40 D:45 |
| 5 | Confident | Both reference (`configref.go`) and new-project scaffold carry the template | User said "by default… template to work on" — the scaffold is the default a new project sees; reference is the canonical full-options doc | S:60 R:85 A:75 D:70 |
| 6 | Certain | Depends on tykw's providers schema being merged first | Templates a schema that must exist; user confirmed tykw implemented | S:90 R:90 A:95 D:95 |
| 7 | Confident | No validation of template commands (misconfig still fails at dispatch time) | Standing no-validation/provider-neutrality contract, unchanged by this change | S:75 R:90 A:95 D:90 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
