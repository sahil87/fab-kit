# Intake: Read `@rk_pane_agent_state` with legacy `@rk_agent_state` fallback

**Change**: 260828-xdmh-rk-pane-agent-state-option
**Created**: 2026-08-28

## Origin

One-shot `/fab-draft` invocation (queued, not activated). Raw input:

> fab-kit change: rename internal/pane AgentStateOption to at-rk_pane_agent_state with a fallback read of the legacy at-rk_agent_state, and add the new field to pane_map.go tmux format string with the parser preferring it (refactor). This is Change 4 of run-kit plan file /home/sahil/code/sahil87/run-kit/fab/plans/sahil/26-08-28-tmux-option-scope-naming.md -- read that files Change 4 section at that absolute path for exact scope. Note in the intake that this change must not actually be applied/executed until run-kit Change 3 (dual-read for externally-written options) has been released/merged to run-kit main, since it depends on that dual-read window existing.

This is **Change 4 of 4** in run-kit's cross-repo plan `fab/plans/sahil/26-08-28-tmux-option-scope-naming.md` (run-kit commit `67f4a553`, "tmux User-Option Scope Naming — `@rk_<scope>_<name>` + Legacy Sweep"). Changes 1–3 live in run-kit; this is the only fab-kit change. The Change 4 section of that plan was read verbatim at intake time and is reproduced under § What Changes; the plan file is the scope of record if the two ever disagree.

> ### ⛔ EXECUTION GATE — do not apply before run-kit Change 3 has shipped
>
> This change MUST NOT be applied (no `/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-proceed`) until **run-kit Change 3** ("Externally-written keys, dual-read") is **merged to run-kit `main` and released** (tagged, so `brew upgrade run-kit` delivers it). Change 3 is what makes run-kit's `rk agent setup` hooks *write* `@rk_pane_agent_state` and makes run-kit's own readers accept both names — the dual-read window. Applying this change earlier would make fab prefer a key that no hook ever writes; harmless functionally (the fallback still reads the legacy key), but the docs/skill prose would then describe a convention that does not exist yet, and the rk version floor could not be written.
>
> Verified at intake (2026-08-28): run-kit `main` is at `67f4a553` and `grep -rl rk_pane_agent_state --include='*.go'` in run-kit returns **nothing** — Change 3 has not started. Installed `rk` is `v3.18.7`.
>
> **Apply-entry precondition (first task of the plan):** confirm run-kit Change 3 is released — e.g. the run-kit tag whose `internal/tmux` code carries the `@rk_pane_agent_state` literal, and `rk --version` ≥ that tag locally. Record that version as the floor (see § rk version floor). If the check fails, STOP and report; do not proceed on the fallback alone.

## Why

**Problem.** tmux resolves `#{@opt}` by walking pane → window → session → global, so a user option's *scope* is invisible in its name, and same-name/different-scope collisions leak values across windows (the run-kit plan's trigger: a stray session-scoped `@color slate` colouring every window in the `fabKit` session). run-kit is moving every option it owns to the self-documenting scheme **`@rk_<scope>_<name>`**, scope ∈ `srv` · `ses` · `win` · `pane`. The agent-state pane option `@rk_agent_state` becomes **`@rk_pane_agent_state`**.

**fab-kit is a downstream consumer of exactly one of these keys.** fab never writes agent state (production was divested to run-kit in 260706-ioku; `docs/specs/hooks.md`, `docs/memory/runtime/runtime-agents.md`) — it *reads* `@rk_agent_state` in two code paths:

- `src/go/fab/internal/pane/pane.go:25` — `const AgentStateOption = "@rk_agent_state"`, read by `ReadAgentStateOption` via `tmux show-options -pv -t <pane> @rk_agent_state` (the `fab pane capture` path).
- `src/go/fab/cmd/fab/pane_map.go:360` — `tmuxPaneFormat` carries `#{@rk_agent_state}` as field 6 of a 7-field `list-panes -F` format; `parsePaneLines` resolves it (the `fab pane map` fallback path when `rk mux panes --json` is unavailable).

**Consequence of not changing.** Once run-kit's hooks (re-installed via `rk agent setup`) write only the new key, fab's Agent column and `fab pane capture` `agent_state` degrade to unknown (`—`) for every instrumented pane — the operator's pre-send gate (`_cli-agents.md` § Pre-Send Validation) loses its primary `waiting`/`active` signal, and `fab dispatch`'s pane-worker readiness heuristics that ride on it degrade. run-kit's plan keeps writing the legacy key during a deprecation window precisely so fab-kit can migrate without a flag day — but the follow-up ("remove legacy reads") is scheduled to land only *after* this change ships.

**Why dual-read rather than a hard switch.** Users pick up new hook text only by re-running `rk agent setup`; until they do, panes carry only the legacy key. Reading new-then-legacy keeps every existing pane working across mixed hook generations. Legacy-read removal in fab-kit is a *later* follow-up (mirrors run-kit's), not this change.

## What Changes

Run-kit plan § Change 4, verbatim scope (repo `~/code/sahil87/fab-kit`):

> - `src/go/fab/internal/pane/pane.go:25` — `AgentStateOption` → `@rk_pane_agent_state`; `ReadAgentStateOption` reads new, falls back to old (`show-options -pv`, two calls or one format with both).
> - `src/go/fab/cmd/fab/pane_map.go:360` `tmuxPaneFormat` — add `#{@rk_pane_agent_state}` as a new field; parser prefers it, falls back to the legacy field. Keep the field count change explicit in the comment block (lines 351/419).
> - Skill/doc prose mentioning `@rk_agent_state`, `@rk_role`, `@rk_url`, `@rk_type` → new names (prose only; no code path).
> - Requires an `rk` version floor note in fab-kit's docs (the version that ships run-kit Change 3).
> - Tests: parser units for new-only, old-only, both-set.

### 1. `internal/pane/pane.go` — constants + `ReadAgentStateOption`

```go
// AgentStateOption is the tmux pane user option that carries an agent's
// lifecycle state ... (existing doc comment, updated to name the new key and
// the run-kit scope-naming scheme)
const AgentStateOption = "@rk_pane_agent_state"

// LegacyAgentStateOption is the pre-scope-prefix name of AgentStateOption,
// still written by `rk agent setup` hook generations that predate run-kit's
// @rk_<scope>_<name> rename. Readers consult it only when AgentStateOption is
// unset on the pane. Removal is a follow-up once run-kit drops its own legacy
// reads.
const LegacyAgentStateOption = "@rk_agent_state"
```

`ReadAgentStateOption(paneID, server)` becomes: read `AgentStateOption` with the existing `show-options -pv -t <pane>` call; if the call errors **or** returns an empty trimmed value, repeat the identical call with `LegacyAgentStateOption`; return that result (still `""` = unknown on error). The empty-`paneID` refusal and the error→unknown mapping documented in the existing comment block stay exactly as they are. **Two `show-options -pv` calls, not one `display-message -F` format read** — see Assumptions #2 for why (pane-scope strictness; the second call only runs when the new key is unset, i.e. on legacy-hook panes and uninstrumented panes).

Semantics of the value (`<state>:<epoch>`, three states, unknown rules) are untouched — `parseAgentState`/`AgentDisplayFromOption` are not modified.

### 2. `cmd/fab/pane_map.go` — `tmuxPaneFormat` + `parsePaneLines`

Format string becomes **eight** tab-separated fields; the new key is inserted as field 6, the legacy key shifts to field 7, `#{window_id}` stays the never-empty **trailing** field (now 8):

```go
const tmuxPaneFormat = "#{pane_id}\t#{window_name}\t#{pane_current_path}\t#{session_name}\t#{window_index}\t#{@rk_pane_agent_state}\t#{@rk_agent_state}\t#{window_id}"
```

`parsePaneLines`: `SplitN(line, "\t", 8)`; graded tolerance extended by one rung:

| Field count | agent-state source | `windowID` |
|-------------|--------------------|------------|
| 8 | field 6 (`@rk_pane_agent_state`) if non-empty after `TrimSpace`, else field 7 (`@rk_agent_state`) | field 8 |
| 7 (legacy 7-field layout) | field 6 (legacy key) | field 7 |
| 6 | field 6 | `""` |
| 5 | none (unknown) | `""` |
| < 5 | line skipped | — |

The two comment blocks the plan names (the `tmuxPaneFormat` block at ~line 351 and the `parsePaneLines` block at ~line 419) MUST be rewritten to state the eight-field layout, the two possibly-empty MIDDLE fields, the prefer-new rule, and the trailing-never-empty invariant. The newline-only per-line trim (never `TrimSpace`) stays load-bearing and stays.

The **rk-delegated** path (`rk mux panes --json`, `parseRKPanes`) is unaffected — it takes rk's reconciled `agent_state` structurally and never reads either option (`pane-commands.md` § enumeration delegation).

### 3. Tests (test-alongside)

- `cmd/fab/pane_map_test.go` `TestParsePaneLines`: add cases **new-only** (field 6 set, 7 empty), **old-only** (6 empty, 7 set), **both-set with different states → new wins**, plus the existing 7-/6-/5-field legacy-line cases re-asserted against the new split width. Any fixture lines built from `tmuxPaneFormat`'s field count are updated.
- `internal/pane/pane_test.go` `TestReadAgentStateOption_Integration` (real tmux test socket): add sub-cases — only legacy key set → read; only new key set → read; both set → new value returned; neither → `""`. The writer emulation (`tmux set-option -p -t <pane> <option> <val>`) uses both constants.
- `cmd/fab/pane_capture_test.go`: sweep any literal `@rk_agent_state` fixtures/comments.
- Run `go test ./internal/pane/... ./cmd/fab/ -run 'ParsePaneLines|AgentState|PaneCapture'` first, then the two packages in full.

### 4. Prose sweep (no code path)

Grep repo-wide and update every occurrence in the sweep class (code-quality.md § Sibling Sweeps — sweep up front, including **string literals** and `*_test.go` comments per the recurring-lessons note):

- **`@rk_agent_state` → `@rk_pane_agent_state`**, with the canonical owner paragraphs gaining one sentence that the legacy `@rk_agent_state` is still read as a fallback during run-kit's deprecation window. Canonical owners: `src/kit/skills/_cli-fab.md` § fab pane → § agent state (line ~522 + JSON field table ~544); `docs/memory/runtime/runtime-agents.md`. Sweep targets (name only): `src/kit/skills/fab-operator.md` (~113/116/150/363/813), `src/kit/skills/_cli-agents.md` (~96 + 3 more), `src/go/fab/cmd/fab/skill.md:72` **and its twin** `docs/site/skill.md:72`, `docs/memory/runtime/{pane-commands,operator,agent-primitives,dispatch}.md`, `docs/memory/pipeline/{hooks-may-enhance-never-own,schemas,change-lifecycle}.md`, `docs/memory/distribution/{kit-architecture,setup}.md`, `docs/specs/{hooks,architecture,index}.md`, and the `runtime` domain description in `docs/memory/index.md` / `docs/memory/runtime/index.md` (regenerated via `fab memory-index`, description edited in the file frontmatter that feeds it).
- **`@rk_role` → `@rk_win_role`** (renamed by run-kit Change 2, which ships before Change 3): `src/kit/skills/fab-operator.md:73`, `src/kit/skills/_cli-external.md:213`, `docs/memory/runtime/operator.md:37,489`.
- **`@rk_url` / `@rk_type`**: the only fab-kit occurrences are in `docs/memory/{runtime,distribution}/log.md` — historical log rows, **left untouched**.
- **Left untouched as history**: `src/kit/migrations/2.13.6-to-2.14.0.md` (describes what that migration did at 2.14.0), all `log.md` rows, archived changes.

### 5. rk version floor note

Add to the canonical `_cli-fab.md` § agent state paragraph and `runtime-agents.md`: "`@rk_pane_agent_state` is written by run-kit ≥ **v{X}** (the release carrying run-kit Change 3 of its tmux option scope-naming plan); older `rk agent setup` hook generations write only `@rk_agent_state`, which fab still reads as a fallback." `{X}` is resolved at apply entry from the precondition check (§ Origin gate) — the run-kit tag containing the Change 3 merge. Do not guess it.

### Non-goals

- Removing the legacy `@rk_agent_state` read — a later fab-kit follow-up, sequenced after run-kit's own "remove legacy reads" follow-up.
- Any change to agent-state value semantics, the three-state set, or the unknown rules.
- Touching run-kit; touching `rk mux panes` delegation.
- A `_cli-fab.md` command-signature change (none — `fab pane map/capture` flags and JSON field names are unchanged; `agent_state` stays the JSON key).

## Affected Memory

- `runtime/runtime-agents.md`: (modify) agent-state convention section — new key name, legacy fallback, rk version floor, design decision for two-call reader
- `runtime/pane-commands.md`: (modify) `map` fallback-path format (eight fields, prefer-new) and `capture` reader description
- `runtime/operator.md`: (modify) `@rk_agent_state` → `@rk_pane_agent_state` mentions; `@rk_role` → `@rk_win_role` in the role self-mark paragraphs
- `runtime/agent-primitives.md`: (modify) pre-send gate prose naming the option
- `runtime/dispatch.md`: (modify) option-name mentions
- `runtime/index.md`: (modify) domain description names the option — regenerate via `fab memory-index`
- `pipeline/hooks-may-enhance-never-own.md`, `pipeline/schemas.md`, `pipeline/change-lifecycle.md`: (modify) option-name mentions
- `distribution/kit-architecture.md`, `distribution/setup.md`: (modify) option-name mentions

## Impact

- **Go**: `src/go/fab/internal/pane/pane.go` (2 constants, 1 function), `src/go/fab/cmd/fab/pane_map.go` (format string, parser, 2 comment blocks); tests in `internal/pane/pane_test.go`, `cmd/fab/pane_map_test.go`, `cmd/fab/pane_capture_test.go`. Behavior change is additive: every pane readable today stays readable; panes with only the new key become readable.
- **Skills**: `_cli-fab.md`, `_cli-agents.md`, `_cli-external.md`, `fab-operator.md` (prose only; no flow/tool change).
- **Docs**: memory files above; specs `hooks.md`, `architecture.md`, `index.md` (name-only edits); `src/go/fab/cmd/fab/skill.md` + `docs/site/skill.md` twins.
- **Runtime dependency**: none new — fab still reads with plain `tmux`; the version floor is documentation of when the *new* key exists, not a hard requirement.
- **Cross-repo sequencing**: blocked on run-kit Change 3 release (§ Origin gate). Release tier: patch (`refactor`).

## Open Questions

- None blocking. The rk version floor `{X}` is a lookup deferred to apply entry by design (it cannot exist until run-kit Change 3 is tagged).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Execution gate: this change is not applied until run-kit Change 3 is merged to run-kit `main` and released; the first plan task verifies this and stops on failure | User-stated in the invocation; dependency verified real at intake (no `rk_pane_agent_state` in run-kit Go at `67f4a553`) | S:95 R:90 A:95 D:95 |
| 2 | Certain | `ReadAgentStateOption` does two sequential `show-options -pv` calls (new, then legacy on error/empty) rather than one `display-message -F` with both keys | `show-options -pv` reads the pane scope strictly and its error→unknown mapping is already documented; `#{@opt}` format expansion walks pane→window→session→global — the very scope leak the run-kit plan is fixing. Extra subprocess only when the new key is unset. Plan allows either | S:80 R:90 A:80 D:70 |
| 3 | Certain | When both keys are set, the new key wins (both reader paths) | Stated in the plan (Change 3 mirrors it in run-kit: "new wins when both set") | S:95 R:90 A:95 D:95 |
| 4 | Certain | Format-string layout: new key inserted as field 6, legacy shifts to 7, `#{window_id}` stays trailing (8); parser split width 8 with one extra tolerance rung | Preserves the existing trailing-never-empty invariant and legacy-line tolerance the comment block documents; alternative (append new key before window_id as field 7) is equivalent but reads worse ("prefer field 7 over 6") | S:80 R:85 A:85 D:75 |
| 5 | Certain | Legacy key kept as an exported `LegacyAgentStateOption` constant next to `AgentStateOption` | Tests write both; code-quality forbids magic strings; exported to match the existing constant | S:70 R:95 A:85 D:80 |
| 6 | Certain | `@rk_role` prose → `@rk_win_role`; `@rk_url`/`@rk_type` occur only in `log.md` history and are left; migration `2.13.6-to-2.14.0.md` and archived changes left as history | Plan says prose-only sweep; run-kit Change 2 (which renames `@rk_role`) precedes Change 3 so it is shipped by the time this runs; log rows are dated history, not present-truth claims | S:75 R:95 A:80 D:80 |
| 7 | Confident | rk version floor value `{X}` resolved at apply entry from the run-kit tag carrying Change 3; placed in `_cli-fab.md` § agent state + `runtime-agents.md` | Unknowable at intake by construction; apply has the precondition check anyway. Owner-or-pointer: two canonical homes, sweep targets point | S:70 R:90 A:45 D:80 |
| 8 | Certain | Change type `refactor`; no `_cli-fab.md` command-signature change (flags/JSON keys unchanged) | Plan assigns `refactor`; `agent_state` JSON key and all flags are untouched | S:90 R:90 A:95 D:90 |
| 9 | Certain | Legacy-read removal in fab-kit is explicitly out of scope (later follow-up after run-kit drops its own) | Plan's follow-up section sequences removal after Change 4 ships | S:85 R:95 A:85 D:85 |

9 assumptions (8 certain, 1 confident, 0 tentative, 0 unresolved).
