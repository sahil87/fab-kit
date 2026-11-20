# Intake: Reserved `tool-default` model value

**Change**: 261118-djnz-tool-default-model-token
**Created**: 2026-11-18

## Origin

> Execute step F2 (Reserved use-the-tool's-default model value) of the plan at
> /home/sahil/code/sahil87/fab-kit/fab/plans/sahil/26-11-16-model-pins.md. Read the whole plan first
> (Decisions and Rules bind you), then implement only the F2 Brief.

One-shot, operator-spawned. The plan (`fab/plans/sahil/26-11-16-model-pins.md`, also present in this
worktree) is the authority; its **Decisions** table and **Rules for the operator** bind this change.

F2 carries a **user gate** ("Ask the user for the token before starting"). Asked at intake; the user
chose **`tool-default`** over the plan's proposed `cli-default` and over `@default`. Options rejected and
why (from the question): `default` collides with Claude Code's own `default` model setting; `auto`
collides with custom Claude models such as `auto[1m]`; `@default` needs YAML quoting. The user picked
`tool-default` because it matches the UI wording the plan already uses ("tool default (…)").

The F2 Brief, verbatim (with the chosen token substituted for the proposal):

> - Resolution treats the token as empty for `{model}` substitution (the token is dropped and the
>   tool's default applies) and for `model_alias` (empty → omit the Agent tool's `model` param, so
>   the native rung inherits the session model).
> - `fab config set` accepts it where it refuses `""`. It is a real value, so it wins over lower tiers.
> - The token is exempt from F3's descend rule: it resolves to an empty model before the rung check.
> - `fab agent -o yaml` reports `model: ""` with `source.model` naming the key that held the token.
> - `fab agent models` (D1) shows it as the first row per tool, labelled with the discovered default
>   (`tool default (gpt-6.1-sol)`).
> - Document it in `fab config explain`, `stage-models.md`, `config.md`, memory and `_cli-fab*.md`.

Plan rules that apply: rule 4 orders fab-kit steps F1 → D1 → F3 → **F2** → N1 and says D1/F3/F2 use
**today's `provider` spelling** (N1 renames to `cli` later). This branch already carries F1, D1 and F3
(cherry-picked in `b6b2e0d1`): the claude fills are `opus`/`sonnet`, `fab agent models` exists
(`src/go/fab/cmd/fab/agent_models.go`, `src/go/fab/internal/agentmodels/`), and F3's native-rung
descent is live (`agent.HasNativeModel`, `dispatch.NativeAvailabilityFor`, `noDispatchCapabilityError`).
F3 left explicit hand-off notes for this step (present-tense "a future … token resolves to empty
before this check") in `src/go/fab/internal/agent/agent.go` (`HasNativeModel` doc comment),
`docs/specs/stage-models.md:643` and `docs/memory/runtime/providers-and-profiles.md:336`.

## Why

1. **The problem.** Plan problem #5: "There is no way to say 'use the tool's own default' for a tool
   that only takes concrete slugs (codex, agy). An empty value is refused by `fab config set`." Today
   the model precedence is `flag > agent.profiles.<role>.model > providers.<p>.profiles.<role>.model >
   providers.<p>.profiles.default.model > empty`, and an empty leaf falls through to the next tier. So
   for codex — whose built-in fills are concrete (`gpt-6-astra`, `gpt-5.6-sol`, `gpt-5.6-luna`) — a
   user can never reach the "empty" rung: `fab config unset agent.profiles.review.model` lands on the
   kit fill `gpt-5.6-sol`, and `fab config set … ""` is refused (exit 1, "refusing to set … to an empty
   value … run fab config unset … instead"). Codex's own default is already `gpt-6.1-sol` (from
   `~/.codex/config.toml`), so the only way to follow it is to pin `gpt-6.1-sol` — which then goes stale
   on the next codex release, the exact failure the plan exists to remove.
2. **The consequence of not fixing it.** Users of concrete-slug tools stay on the kit's release cadence
   or on hand-written pins; hexokit H3 ("offer the tool default in the UI") has nothing to commit.
3. **Why this approach.** A reserved, non-empty token is a *real value* at whatever tier it is written,
   so it beats the lower tiers (unlike `""`, which falls through by design), and the resolver collapses
   it to empty *after* precedence, reusing the existing "empty drops the `{model}` token" composition
   (`spawn.WithProfile`) and the existing native inherit path (empty model → no Agent-tool `model`
   param). No new placeholder grammar, no change to the empty-skip cascade, no per-tool logic.

## What Changes

Use today's `provider` spelling everywhere (plan rule 4). The token string is exactly `tool-default`
(lowercase, hyphen, exact case-sensitive match, no trimming).

### 1. The token constant and one collapse helper (`src/go/fab/internal/agent/agent.go`)

- Add an exported constant, e.g. `const ToolDefaultModel = "tool-default"`, with a doc comment stating
  its meaning (use the tool's own default model; resolves to an empty model after precedence).
- Add one small helper that maps the token to `""` and returns every other string verbatim (e.g.
  `func EffectiveModel(model string) string`). No other validation — provider neutrality and
  document-don't-validate stand.
- Update `HasNativeModel`'s doc comment from the F3 hand-off wording ("A future 'use the tool's default'
  token resolves to empty before this predicate runs") to present truth naming `ToolDefaultModel`.

### 2. Resolution collapses the token after precedence

In `ResolveRoleWithSource` (`internal/agent/agent.go`), after the full model precedence has run
(including the `--model` flag override), apply the helper to `p.Model`. **`source.Model` is NOT cleared**
— it keeps naming the rung that held the token (`agent.profiles.review`, `providers.codex.profiles.review`,
`providers.codex.profiles.default`, or `flag`). This is the one place every resolution path flows
through: `fab agent`, `fab resolve-agent`, `fab dispatch start` (`agent.Resolve`), `fab pane open`
(`ResolveRoleWith`), operator/batch launches (`spawn`, `ResolveRole`), setupcheck probes, and
configref's `liveRoleProfiles`.

`composeAgentResolution` (`cmd/fab/resolution.go`) has a second path, `request.ProviderOnly`, that
builds the profile straight from the overrides without calling `ResolveRoleWithSource`
(`fab agent --provider X --model tool-default …`). Apply the same helper there so `--model tool-default`
behaves identically on both addressing forms.

Because the collapse happens before `HasNativeModel` / `dispatch.NativeAvailabilityFor` see the model,
the token is automatically **exempt from F3's descend rule** — no change to the rung selector itself.

Precedence consequence (it is a real value, so it wins over lower tiers):

```yaml
# ~/.fab-kit/config.yaml
agent:
  profiles:
    review: { provider: codex, model: tool-default, effort: high }
```

```
$ fab agent review -o yaml
selector: review
kind: role
role: review
provider: codex
model: ""
effort: high
command: codex --dangerously-bypass-approvals-and-sandbox -c model_reasoning_effort=high   # no -m pair
model_alias: ""
...
source:
    provider: agent.profiles.review
    model: agent.profiles.review          # the key that held the token
    effort: agent.profiles.review
dispatch:
    rung: headless
    command: codex exec … (no -m pair)
```

(The exact codex command shape comes from the built-in template; what matters is that the `{model}`
token and its flag drop out exactly as for an empty model today.) Without the token the same config
minus the model line resolves `model: gpt-5.6-sol` from `providers.codex.profiles.review`.

Claude on the native rung: `agent.profiles.doing.model: tool-default` with `dispatch.mode: native`
resolves `model: ""`, `model_alias: ""`, **no `dispatch:` block** (native kept, not descended), and
`source.model: agent.profiles.doing` — the orchestrator omits the Agent tool's `model` param and the
sub-agent inherits the session model. On a CLI rung (`headless`/`pane`) the `--model {model}` pair drops
and the `claude` CLI uses its own default.

`fab resolve-agent <stage|role>` prints an empty `model=` line for the token (with or without
`--alias`), and its `dispatch=` line carries the command with the model pair dropped.

Effort is out of scope: `tool-default` is a **model-only** token; an `effort: tool-default` passes
through verbatim like any other effort string (no special handling).

### 3. `fab config set` accepts the token; the empty-value refusal points at it

`fab config set` already accepts any non-empty scalar on model keys (`validateSetValue` in
`cmd/fab/config_validate.go` has no model case), so
`fab config set --system agent.profiles.review.model tool-default` writes the leaf today. Changes:

- Add tests proving it for `agent.profiles.<role>.model` and `providers.<p>.profiles.<role>.model`, and
  that the written value wins over the provider fill on resolution.
- Extend the empty-value refusal in `configSetCmd` (`cmd/fab/config.go`, the
  `refusing to set %q to an empty value` error) so that, **for a key ending in `.model`**, it also says
  how to request the tool's own default, e.g. `… — run \`fab config unset <key>\` to remove the override
  instead, or \`fab config set <key> tool-default\` to use the tool's own default model`. Non-model keys
  keep today's message. Update the command's `Long` help to mention the token alongside the
  empty-refusal sentence.

### 4. F3's no-capability remedy mentions the token

`noDispatchCapabilityError` (`cmd/fab/resolve_agent.go`) currently suggests "set the model to an alias
(opus/sonnet/haiku/fable) or leave it empty to inherit" when native was skipped for an alias-less model.
Add the token as a remedy in that branch (e.g. "… or set it to `tool-default`"). Update the test that
asserts the message.

### 5. `fab agent models` prepends a tool-default row per tool (`internal/agentmodels/models.go`)

In `discoverWith`, after merge and the per-row normalization, prepend one row so it is the **first row
for every tool** (built-ins, `models_command` tools, and tools with no adapter such as agy):

```json
{"cli":"codex","id":"tool-default","label":"tool default (gpt-6.1-sol)","efforts":["low","medium","high","xhigh","max"],"default":false,"source":"reserved"}
```

- `id`: the token (`agent.ToolDefaultModel`).
- `label`: `tool default (<id of the row with default: true>)` when discovery reported a default;
  plain `tool default` when no default is known (discovery failed, agy, kimi without `default_model`).
- `efforts`: the default model row's `efforts` when there is one, else the tool's known set (claude's
  `claudeEfforts`), else `[]`.
- `default: false` — the tool's actual default model row keeps `default: true`; the token row is not a
  model.
- `source`: a new constant `SourceReserved = "reserved"`, distinct from `tool`/`alias`/`fill`, so
  consumers (hexokit H3) can identify the row without string-matching the id.
- Never deduped against or merged with other rows (a tool reporting a literal `tool-default` id is
  ignored for this purpose — the token is reserved).
- The table output (`writeModelTable`) prints it like any row (source column `reserved`).

Fail-open is unchanged: a discovery error still yields fills plus a warning, and the token row is still
first (with the plain `tool default` label).

### 6. Documentation

Present-truth updates, owner-or-pointer respected (deployed skills restate; fab-kit docs point/carry):

- `fab config explain` prose (`src/go/fab/internal/configref/configref.go`): the `agent.profiles`
  field Description and the `providers` field Description/segment prose (the precedence paragraph near
  "empty drops the placeholder's token, so the CLI's own default applies") gain one sentence: the
  reserved model value `tool-default` is a real value that wins over lower tiers and resolves to an
  empty model, so the tool's own default applies (and native inherits the session model). If the
  rendered managed-fence/advert text changes, follow the established precedent for advert digests in
  `configupgrade` (the previous digest joins the legacy list, as F1 did) so `fab config upgrade`
  re-renders the fence — check whether the changed text is inside a digested segment before doing so.
- `src/kit/skills/_cli-fab.md`: `fab agent` / `fab resolve-agent` resolution (the token collapses to
  `model: ""`, `source.model` keeps the key), `fab agent models` (the first `reserved` row, its label
  and efforts rules), and `fab config set` (accepts the token; the `.model` refusal hint). Grep
  `_cli-fab-pane.md` for model/inherit wording that needs the same note.
- `src/kit/skills/_preamble.md` § Per-Stage Model Resolution: only if its "Empty value" wording needs
  to say the token also yields an empty model — keep it a one-clause addition or leave it if the
  resolver output already makes it implicit (resolved YAML shows `model: ""`).
- `docs/specs/stage-models.md`: replace the "a future 'use the tool's default' token" wording at line
  ~643 with the present rule; add the token to the precedence/default-profiles prose.
- `docs/specs/config.md`: the empty-value refusal and the token as the way to request a tool's default.
- `docs/specs/glossary.md`: add a "tool-default" entry (sibling sweep of aggregate specs).
- Memory (hydrate): `docs/memory/runtime/providers-and-profiles.md` (resolution, native rule at ~336,
  `fab agent models` rows), `docs/memory/_shared/configuration.md` (config set behavior).

No migration: no existing user data is restructured.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) the `tool-default` reserved model value — collapse after
  precedence, `source.model` keeps the holding key, native-rung exemption from F3's descent, and the
  `reserved` first row in `fab agent models`
- `_shared/configuration`: (modify) `fab config set` accepts `tool-default` on model keys and the
  `.model` empty-refusal hint

## Impact

- **Go**: `src/go/fab/internal/agent/agent.go` (+ `agent_test.go`), `src/go/fab/cmd/fab/resolution.go`,
  `src/go/fab/cmd/fab/resolve_agent.go` (+ `resolve_agent_test.go`), `src/go/fab/cmd/fab/config.go`
  (+ `config_test.go`), `src/go/fab/internal/agentmodels/models.go` (+ `models_test.go`,
  `cmd/fab/agent_models_test.go`), `src/go/fab/internal/configref/configref.go` (+ tests/golden
  fixtures if the prose is snapshot-tested; possibly `internal/configupgrade` legacy digest list).
- **Kit**: `src/kit/skills/_cli-fab.md` (constitution: CLI changes MUST update the owning CLI partial).
- **Specs**: `docs/specs/stage-models.md`, `docs/specs/config.md`, `docs/specs/glossary.md`.
- **Downstream**: hexokit H3 consumes the `reserved` row (`id`, `label`) from `fab agent models --json`;
  N1 later renames `provider` → `cli` around this code but keeps the token unchanged.
- No new dependencies, no migration, no change to `dispatch.SelectMode`.

## Open Questions

- None blocking. The token spelling (the plan's open decision for F2) was resolved at intake by the user.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | The token is exactly `tool-default` (case-sensitive, exact match) | Asked — user chose `tool-default` over `cli-default` and `@default` at the F2 user gate | S:95 R:70 A:95 D:95 |
| 2 | Certain | Scope is the F2 Brief only; N1 rename, H3 UI work and the D1 stale-pin warning are excluded; use today's `provider` spelling | User instruction ("implement only the F2 Brief") and plan rule 4 | S:95 R:90 A:95 D:95 |
| 3 | Certain | The token collapses to `""` after the full model precedence (incl. `--model`), so it wins over lower tiers and is exempt from F3's descent | F2 Brief bullets 1–3; F3 hand-off notes in agent.go / stage-models.md / providers-and-profiles.md | S:90 R:80 A:90 D:90 |
| 4 | Certain | `source.model` keeps the rung that held the token (`agent.profiles.<role>`, `providers.<p>.profiles.<role>`, `…default`, or `flag`) | F2 Brief bullet 4 verbatim | S:95 R:85 A:90 D:95 |
| 5 | Certain | Collapse in `ResolveRoleWithSource` plus the `ProviderOnly` branch of `composeAgentResolution`, via one shared helper | Every resolution caller (agent, resolve-agent, dispatch start, pane open, spawn, setupcheck, configref) flows through ResolveRoleWithSource; ProviderOnly bypasses it | S:70 R:85 A:85 D:75 |
| 6 | Confident | The token is model-only; `effort: tool-default` is not special | Brief speaks only of `{model}` and `model_alias`; efforts already accept empty via unset and have no concrete-slug problem | S:65 R:85 A:75 D:75 |
| 7 | Confident | `fab config set` needs no validation change (already accepts it); add tests and a `.model`-key hint to the empty-value refusal | `validateSetValue` has no model case; the hint turns the refusal into a pointer at the new opt-out | S:70 R:90 A:80 D:70 |
| 8 | Confident | `fab agent models` token row: `source: "reserved"`, `default: false`, efforts from the default row else the tool's set else `[]`, label `tool default (<default id>)` or plain `tool default` | Brief fixes position and label; source/efforts/default flag unspecified — chosen so H3 can identify the row by source and offer sensible efforts | S:50 R:70 A:60 D:50 |
| 9 | Confident | F3's no-capability error adds the token as a remedy | Natural sibling of the "leave it empty to inherit" remedy; low-risk wording change | S:60 R:90 A:80 D:75 |
| 10 | Confident | Docs: configref explain prose, `_cli-fab.md`, stage-models.md, config.md, glossary.md, memory at hydrate; bump the advert legacy digest only if digested fence text changes | Brief's documentation bullet plus code-quality sibling-sweep rule; F1 digest precedent | S:75 R:85 A:75 D:70 |
| 11 | Certain | No migration | No user data is restructured; the token is purely additive | S:85 R:90 A:95 D:95 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved).
