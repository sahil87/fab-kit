# Intake: Native rung descends when the model has no Agent-tool alias

**Change**: 261117-3ms2-native-rung-descends-without-alias
**Created**: 2026-11-17

## Origin

> Execute step F3 (Native rung descends when the model has no Agent-tool alias) of the plan at
> fab/plans/sahil/26-11-16-model-pins.md. Read the whole plan first (Decisions and Rules bind you),
> then implement only the F3 Brief.

One-shot, operator-spawned. The plan (`fab/plans/sahil/26-11-16-model-pins.md`) is the authority; its
**Decisions** table and **Rules for the operator** bind this change. The F3 Brief, verbatim:

> - First verify what happens today when the resolved Claude model has no Agent-tool alias
>   (`glm-latest[1m]`) and the native rung is chosen: does the orchestrator run the sub-agent on the
>   session model, or descend? Record the answer in the change's intake.
> - Then: a non-empty model with an empty `model_alias` makes the native rung unavailable for that
>   role, so resolution descends to the next rung the tool supports (headless for claude), and
>   `fab agent -o yaml` says why (e.g. `dispatch.reason: model has no native alias`). An empty model
>   keeps today's inherit behaviour, and so will F2's token once it lands (F2 treats it as empty
>   before this rule runs).
> - Non-goals: aliasing custom models.

Plan rules that apply here: rule 4 orders fab-kit steps F1 → D1 → **F3** → F2 → N1, and says D1/F3/F2
use **today's `provider` spelling** (N1 renames later). This branch already carries D1 (cherry-picked
as `a0fba715`), so F1's alias fills (`opus`/`sonnet`) and `agent.IsModelAlias` are in place.

## Why

### What happens today (verified 2026-11-17 — the Brief's first bullet)

Built from this worktree (`go build ./cmd/fab`), run in a scratch `HOME` with no project, `$TMUX`
unset, and this system config:

```yaml
dispatch:
  mode: native
agent:
  profiles:
    doing:
      model: glm-latest[1m]
```

`fab agent apply -o yaml` printed:

```yaml
selector: apply
kind: stage
role: doing
provider: claude
model: glm-latest[1m]
effort: high
command: claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model glm-latest[1m] --effort high
model_alias: ""
template: claude --permission-mode bypassPermissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}
fill_mode: template
source:
    provider: built-in
    model: agent.profiles.doing
    effort: providers.claude.profiles.doing
skill_prefix: /
```

No `dispatch:` key → the native rung. `fab agent review -o yaml` (unpinned) resolved `model: opus`,
`model_alias: opus`, native — correct.

**Answer: the orchestrator runs the sub-agent on the session model, silently. It does not descend.**
The chain that produces it:

1. `composeAgentResolution` (`src/go/fab/cmd/fab/resolution.go`) sets `ModelAlias` only when
   `agent.IsModelAlias(agent.ModelAlias(model))` — `glm-latest[1m]` gets `""`.
2. `dispatchResolutionFor` calls `dispatch.SelectMode(..., prov.Native, ...)`. Claude has
   `native: true`, so under `dispatch.mode: native` (the shipped default) — or `pane` with no tmux —
   native resolves and `Dispatch` is nil. The model plays no part in rung selection.
3. Every dispatch site branches **only** on `dispatch:` presence (`_preamble.md` § CLI-Adapter
   Dispatch; `_pipeline.md` step 1; `fab-continue.md`). Absent → native Agent-tool dispatch, using
   the `model_alias` seam. The seam table says the Agent tool's `model` parameter "accepts the Claude
   aliases `opus`/`sonnet`/`haiku`/`fable`" and an empty value means "omit the parameter … inherit the
   orchestrator/session model". There is no rule that says "non-empty model + empty alias → stop".
   So the Agent call carries no `model`, and the worker runs on whatever the session runs.
4. Nor can the operator escape onto the CLI adapter: `fab dispatch start` re-resolves the stage from
   config, lands on the same native rung, and refuses with `nativeDispatchError` ("stage … resolves
   to native mode … dispatch natively when the `dispatch:` key is absent").

The user asked for `glm-latest[1m]` and got Opus (or whatever the session is on), with only a
`model_alias: ""` in the surfaced YAML as a hint. That is the plan's original complaint in another
shape: a model string that silently does not take effect.

### Why descend, rather than error or warn

- `claude -p --model glm-latest[1m]` (claude's `headless_command`) **does** honour the custom model —
  Claude Code takes any configured model string; only the Agent tool's `model` enum is limited to the
  four aliases. So a rung that can carry the model exists; picking it is strictly better than running
  the wrong model.
- `dispatch.mode` is already a **ceiling with descent** (pane → native → headless; never ascends). A
  rung whose prerequisite is missing is skipped today (`pane unavailable: no tmux`, `native
  unavailable`). "The native seam cannot carry this model" is one more missing prerequisite of the
  native rung; reusing the ladder needs no new policy surface, and every dispatch site already handles
  `dispatch:`-present.
- An error would block the stage for a config the user wrote deliberately; a warning leaves the silent
  substitution in place.

## What Changes

### 1. The rule (Go: rung selection)

The native rung is **available for a resolution** iff:

```
provider declares native: true
AND (model == ""  OR  model has a native alias)
```

where "has a native alias" is exactly today's `model_alias` computation:
`agent.IsModelAlias(agent.ModelAlias(model))` — true for `opus`/`sonnet`/`haiku`/`fable` and for
versioned Claude IDs (`claude-opus-5-5`, `claude-haiku-4-5-20251001`), false for anything else
(`glm-latest[1m]`, `kimi-latest[1m]`, `gpt-5.6-sol`, `auto[1m]`).

- **Empty model** → native stays available; today's inherit behaviour (omit the Agent tool's `model`,
  run on the session model) is unchanged.
- **Alias or versioned Claude ID** → unchanged (native, `model_alias` set).
- **Non-empty model with no alias** → native is skipped; the ladder continues to the next rung the
  provider supports. For built-in claude that is headless (`claude -p … --model glm-latest[1m] …`).
  If the provider has no lower rung, the ladder is empty and the existing no-capability error fires —
  see § 4.
- The rule is **provider-neutral**: it keys on the model string and the native capability, never on
  the provider name. A user-defined provider that declares `native: true` gets the same treatment.
- Ordering: the rule only matters where native would otherwise be picked. Under `dispatch.mode: pane`
  with tmux available, pane resolves first and the rule never runs. Under `pane` without tmux, the
  ladder skips pane, then skips native for the alias reason, then lands on headless. Under
  `headless`, nothing changes.
- **F2 hand-off (not implemented here):** F2's "use the tool's default" token resolves to an empty
  model *before* this rule runs, so it keeps native-with-inherit. This change needs no code for that;
  the predicate takes the already-resolved model string. Note it in the code comment and docs so F2
  slots in without revisiting the rule.

One predicate, one owner. Put it in `internal/agent` next to `ModelAlias`/`IsModelAlias` (e.g.
`agent.HasNativeModel(model string) bool` — "empty or aliasable"), and have `composeAgentResolution`'s
`ModelAlias` assignment and every rung-selection site share it rather than re-deriving the alias
check.

### 2. `dispatch.SelectMode` learns *why* native is unavailable

Today `SelectMode` takes `native bool` and records the fixed skip text `"native unavailable"`. To let
the reason name the cause, replace the bool with a caller-supplied availability value, mirroring the
existing `TmuxAvailability` pattern in `internal/dispatch/pane_mode.go` (the caller computes it; the
pure function only reports it):

```go
// NativeAvailability is the caller-observed native-rung prerequisite.
type NativeAvailability string

const (
	NativeAvailable    NativeAvailability = "available"
	NativeNoCapability NativeAvailability = "no native capability"   // provider lacks native: true
	NativeNoAlias      NativeAvailability = "model has no native alias"
)
```

Skip texts appended by the native branch:

| Availability | Skip text |
|---|---|
| `NativeNoCapability` | `native unavailable` — **unchanged**, byte-for-byte (tests, `fab doctor` probe docs and `_cli-fab-pane.md` quote it) |
| `NativeNoAlias` | `native unavailable: model has no native alias` |

So `dispatch start`'s existing stderr line becomes, for the F3 case:
`dispatch selection: mode: headless (descended: native unavailable: model has no native alias)`.

(Exact type/constant names are the plan's call; the two skip strings above are the contract.)

### 3. `fab agent -o yaml` says why: `dispatch.reason`

`agent.DispatchResolution` gains a `reason` field, emitted after `command`, `omitempty`:

```go
type DispatchResolution struct {
	Rung    string `yaml:"rung"`
	Command string `yaml:"command"`
	Reason  string `yaml:"reason,omitempty"`
}
```

It carries the ladder's **skip list** — the descent causes joined with `"; "` (the same strings
`SelectMode` already builds) — and is omitted when the selected rung is the configured preference
(nothing was skipped). For the F3 case:

```yaml
dispatch:
    rung: headless
    command: claude -p --permission-mode bypassPermissions --model glm-latest[1m] --effort high
    reason: 'native unavailable: model has no native alias'
```

General by design, not F3-specific: a codex role under the default `native` preference will now also
show `reason: native unavailable`, and `pane` without tmux shows
`pane unavailable: no tmux; native unavailable: …`. This is additive YAML (a new key inside the
existing `dispatch:` mapping); dispatch sites still branch only on `dispatch:` presence and `rung`.
`SelectMode` already returns an `AutoReason` like `mode: headless (descended: …)`; the plan may either
expose the skipped list directly or strip that wrapper — the YAML value is the bare skip list shown
above.

`fab resolve-agent`'s frozen line protocol (`Resolution.Lines`) does **not** gain a line: it already
emits `dispatch=<command>` from the same `Dispatch` pointer, so the descent shows up there for free,
and its byte-stable contract forbids new lines.

### 4. Every rung-selection site applies the same rule

`SelectMode` is called in four places; all must compute native availability with the shared predicate
so the query and the launcher agree (otherwise `fab agent -o yaml` says headless and
`fab dispatch start` still refuses with `nativeDispatchError`):

| Site | Model it checks |
|---|---|
| `cmd/fab/resolution.go` `dispatchResolutionFor` (behind `fab agent -o yaml` and `fab resolve-agent`) | the resolved `profile.Model` (after overrides) |
| `cmd/fab/dispatch_start.go` `resolveMode` (start/restart/open) | `agent.Resolve(cfg, stage)`'s `profile.Model` — thread the profile (or its model) into `resolveMode` |
| `cmd/fab/dispatch_start.go` pane-validation re-descent (~line 369) | same profile |
| `internal/setupcheck/probes.go` `ProbeDispatchMode` (`fab doctor`) | the `doing` role profile it already resolves — so a pinned alias-less doing model reports the Warn "would descend at dispatch time: … native unavailable: model has no native alias" |

**No-rung error.** When the alias rule empties the ladder (provider has `native: true` but no
`headless_command`, preference `native`), `noDispatchCapabilityError` today suggests
"configure providers.X.native for native" — wrong, native is already set. The error must name the
real cause and remedies instead, e.g.:

```
provider "p" has no dispatch capability at or below dispatch.mode "native": model "glm-latest[1m]" has no native Agent-tool alias; configure providers.p.headless_command for headless, or set the model to an alias (opus/sonnet/haiku/fable) or leave it empty to inherit: no dispatch mode is available
```

(Wording is the plan's call; the requirement is: names the model, says it has no native alias, and
does not tell the user to configure `native`.)

### 5. Invocation overrides

`fab agent <stage> --model <alias-less> -o yaml` now resolves a `dispatch:` mapping (the override
feeds `profile.Model`). That is consistent with the existing rule that a `dispatch:` caused only by an
override is a query result, not actionable (`fab dispatch start` takes no override flags) — no new
handling; the docs that state that rule just gain this case as an example.

### 6. Tests (constitution: Go CLI changes MUST include tests)

- `internal/agent`: predicate table — `""` → true; `opus`/`sonnet`/`haiku`/`fable` → true;
  `claude-opus-5-5`, `claude-haiku-4-5-20251001` → true; `glm-latest[1m]`, `gpt-5.6-sol`, `auto[1m]`
  → false.
- `internal/dispatch` `SelectMode` table: `native` preference + `NativeNoAlias` + headless → headless
  with reason `native unavailable: model has no native alias`; `NativeNoAlias` + no headless →
  `ErrNoMode`; `pane` preference + tmux absent + `NativeNoAlias` → headless with both skip texts;
  `pane` + tmux available → pane (rule not reached); existing `native unavailable` cases unchanged.
- `cmd/fab` resolution / `fab agent -o yaml` tests: `agent.profiles.doing.model: glm-latest[1m]` under
  `dispatch.mode: native` → `dispatch: {rung: headless, command: claude -p … --model glm-latest[1m] …,
  reason: …}`, `model_alias: ""`; empty model and `opus` → no `dispatch:` (regression guard); a codex
  role shows `reason: native unavailable`.
- `fab dispatch start` test: same config → launches headless (no `nativeDispatchError`), stderr
  carries the descent line.
- No-rung error test for a native-only user provider with an alias-less model.
- `setupcheck` probe test for the Warn finding.
- Update existing expectations that assert exact `dispatch:` YAML (the new `reason` key appears
  wherever a descent happened).

### 7. Docs (deployed kit content restates rules; no fab-kit-only citations — Constitution V)

- `src/kit/skills/_preamble.md` § Per-Stage Model Resolution: the model-seam row's "An empty alias for
  a non-Claude model has no native seam" becomes the rule — a non-empty model with no alias never
  resolves native; the resolver descends and `dispatch.reason` says why. Native-branch dispatch with
  an empty `model_alias` therefore only ever means "empty model → inherit".
- `src/kit/skills/_cli-fab.md` § fab agent: YAML schema row for `dispatch` gains `reason`; the ladder
  description gains the alias prerequisite; the `model_alias` row notes the consequence.
- `src/kit/skills/_cli-fab-pane.md`: where it quotes descent strings, add the new one.
- `docs/specs/stage-models.md` and `docs/specs/harness-adapters.md`: ladder prerequisite + `reason`.
- `fab config explain dispatch.mode` prose in `internal/configref` if it enumerates native's
  prerequisite (check; update if it does).
- Memory per Affected Memory.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) dispatch-mode descent — native's prerequisite now includes "model empty or aliasable"; `fab agent -o yaml` schema gains `dispatch.reason`; new scenario for the alias-less model; Design Decision recording "descend, not error/warn" and the verified pre-change behaviour
- `runtime/dispatch`: (modify) `fab dispatch start`/`restart`/`open` apply the same native-availability rule and its descent text
- `_shared/context-loading`: (modify) Per-Stage Model Resolution — the native model seam never sees a non-empty alias-less model; empty `model_alias` on the native branch means inherit only
- `distribution/setup`: (modify) the `fab doctor` dispatch-mode probe's descent reasons include the new string

## Impact

- Go: `src/go/fab/internal/agent/agent.go` (predicate), `internal/agent/resolution.go`
  (`DispatchResolution.Reason`), `internal/dispatch/pane_mode.go` (`SelectMode` native input + skip
  text), `cmd/fab/resolution.go`, `cmd/fab/dispatch_start.go`, `cmd/fab/resolve_agent.go`
  (`noDispatchCapabilityError`), `internal/setupcheck/probes.go`, plus their tests.
- Kit skills: `_preamble.md`, `_cli-fab.md`, `_cli-fab-pane.md` (src/kit — deployed copies are
  regenerated by `fab sync`, never edited).
- Specs: `docs/specs/stage-models.md`, `docs/specs/harness-adapters.md`.
- External consumers: hexokit's `riff.ResolveAgent` reads only `command` and `skill_prefix` from
  `fab agent -o yaml` (plan § Current state) — the new nested `dispatch.reason` key does not affect it.
  Skills branch on `dispatch:` presence and `rung` — unaffected.
- Behaviour change visible to users: a role pinned to an alias-less model under the default
  `dispatch.mode: native` now runs headless (`claude -p`) instead of silently on the session model.
- Spelling: today's `provider` everywhere (plan rule 4 — N1 renames).

## Open Questions

- None blocking. The plan fixes the rule and the surface; naming of the Go type/constants and the exact
  no-rung error wording are left to the plan.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Today's behaviour: native rung resolves (no `dispatch:`), `model_alias: ""`, the Agent call omits `model`, and the worker silently runs on the session model; `fab dispatch start` refuses with `nativeDispatchError` | Verified by running the worktree binary with the plan's exact config, plus reading the dispatch-site rules in `_preamble.md`/`_pipeline.md` | S:95 R:90 A:95 D:95 |
| 2 | Certain | Rule: non-empty model with no native alias makes the native rung unavailable; resolution descends (headless for claude); empty model keeps inherit | Stated verbatim in the F3 Brief | S:95 R:80 A:95 D:95 |
| 3 | Certain | Aliasing custom models is out of scope; F2's token is not implemented here | F3 Brief non-goal; plan step order F3 → F2 | S:95 R:90 A:95 D:95 |
| 4 | Certain | Use today's `provider` spelling in code, YAML and docs | Plan rule 4: D1/F3/F2 use today's spelling, N1 renames | S:95 R:90 A:95 D:95 |
| 5 | Confident | "Has a native alias" = `agent.IsModelAlias(agent.ModelAlias(model))`, the same check that fills `model_alias`, shared via one predicate in `internal/agent` | Brief keys the rule on empty `model_alias`; one owner prevents query/launcher drift | S:85 R:80 A:85 D:80 |
| 6 | Confident | The rule is applied at all four `SelectMode` sites (agent YAML, dispatch start, its pane re-descent, `fab doctor` probe) | Otherwise `fab agent -o yaml` says headless while `fab dispatch start` still refuses native — the CLI adapter would be unreachable | S:80 R:75 A:85 D:85 |
| 7 | Confident | `SelectMode`'s `native bool` becomes a caller-computed availability value (TmuxAvailability pattern); `native unavailable` stays byte-identical for the no-capability case; the new skip text is `native unavailable: model has no native alias` | Mirrors existing package idiom; preserves strings quoted by tests/docs; matches the Brief's example wording | S:75 R:80 A:80 D:70 |
| 8 | Confident | `dispatch.reason` is a general, `omitempty` key carrying the ladder's skip list (present whenever a descent happened, so codex roles also show `native unavailable`), not an F3-only field | More useful and uniform than a one-cause field; additive inside `dispatch:`, which consumers don't parse beyond `rung`/`command` | S:70 R:75 A:75 D:65 |
| 9 | Confident | When the alias rule empties the ladder, the no-capability error names the model/alias cause and does not suggest configuring `native` | Today's remedy list would point at an already-set key | S:75 R:85 A:85 D:75 |
| 10 | Confident | `fab resolve-agent`'s line protocol gets no new line; its existing `dispatch=` line reflects the descent automatically | Byte-stable deprecated surface; same `Dispatch` pointer feeds it | S:80 R:85 A:90 D:85 |

10 assumptions (4 certain, 6 confident, 0 tentative, 0 unresolved).
