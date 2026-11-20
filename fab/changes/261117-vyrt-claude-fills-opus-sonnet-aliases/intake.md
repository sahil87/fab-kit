# Intake: Claude Fills Use opus / sonnet Aliases

**Change**: 261117-vyrt-claude-fills-opus-sonnet-aliases
**Created**: 2026-11-17

## Origin

> Execute step F1 (Claude fills → opus / sonnet aliases) of the plan at
> fab/plans/sahil/26-11-16-model-pins.md. Read the whole plan first (Decisions and Rules bind you),
> then implement only the F1 Brief.

One-shot `/fab-new` invocation. The plan (`fab/plans/sahil/26-11-16-model-pins.md`, written
2026-11-16, present in this worktree) is the binding design source. Its **Decisions** table and
**Rules for the operator** bind this change; only the **F1 Brief** is in scope. F1 has no
dependencies and is the first fab-kit step in the same-repo order F1 → D1 → F3 → F2 → N1 (plan
Rule 4) — D1, F3, F2 and N1 all build on it, so F1 uses today's `provider` spelling (N1 renames later).

The F1 Brief, verbatim:

> - `src/go/fab/defaults.yaml`: the six `providers.claude.profiles.*` fills become `opus` (default,
>   review) and `sonnet` (doing, hydrate, operator, fast). Efforts unchanged.
> - `configupgrade.go`: add the previous providers-advert digest to the legacy list (the
>   261030-lc7k one becomes legacy) so `fab config upgrade` re-renders the fence, as every prior
>   fill bump did.
> - Keep `agent.ModelAlias()` as is; add a test that `opus`/`sonnet`/`haiku`/`fable` pass through
>   unchanged and `fab agent review -o yaml` gives `model: opus`, `model_alias: opus`, with the
>   rendered command carrying `--model opus`.
> - Update `agent_test.go` expectations, `docs/specs/stage-models.md` (default role profiles),
>   `docs/memory/runtime/providers-and-profiles.md`, `docs/memory/_shared/configuration.md`, and
>   `src/kit/skills/_cli-fab.md` wherever they quote the Claude IDs. The `fab config explain providers`
>   prose that says fills are "refreshed at KIT-RELEASE cadence" should say Claude's are aliases that
>   track the tool, and only codex's and agy's are concrete IDs.
> - Non-goals: changing codex/agy/kimi fills; validating model strings.

The binding plan Decision row for this step: **"Claude fills use aliases — `opus` / `sonnet` instead
of full IDs. Why: `claude --model` accepts aliases and resolves the newest model; the Agent tool's
`model` enum takes exactly these; `agent.ModelAlias()` (`src/go/fab/internal/agent/agent.go:412`)
passes them through unchanged."** Also binding: "Codex/agy fills stay concrete — unchanged."

## Why

**Problem.** Every new Claude model release forces a fab-kit release just to move the built-in
Claude fills. The fills are concrete versioned IDs (`claude-opus-5-5`, `claude-sonnet-5-5`), so an
un-overridden project only reaches a newer model when the kit ships a new `defaults.yaml`. The
providers advert in the managed config fence has been re-rolled **four times for Claude fill
changes alone** — the legacy digest list in `src/go/fab/internal/configupgrade/configupgrade.go`
records 260908-wcib → 260923-vvdv → 260929-9dwn → 261030-lc7k. Each roll is a kit release, a fence
re-render on every user machine, and a doc/memory/spec sweep (see #684, #687, #693).

**Consequence of not fixing.** The kit's Claude fills lag the tool on every release; a user on an
older kit keeps dispatching an older model even though `claude` itself already knows a newer one.
This is cause #2 of the six in the plan's Problem statement.

**Why aliases.** `claude --model` accepts the family aliases `opus` / `sonnet` / `haiku` / `fable` and
resolves each to the newest model in that family, so the fill moves with the installed Claude Code,
not with fab-kit releases. The Agent tool's `model` parameter (the native dispatch seam) takes
exactly these four aliases, so the alias is valid on both seams with no translation. Codex and agy
stay concrete because their `-m` takes a concrete catalog slug and has no "latest" alias (plan
Decision; F2 adds an opt-out for them later).

**Rejected alternatives** (from the plan): keeping concrete IDs and bumping per release (the status
quo that causes the churn); validating model strings (explicit non-goal — fab is verbatim
pass-through).

## What Changes

### 1. `src/go/fab/defaults.yaml` — the six Claude fills

Current (lines ~120–126):

```yaml
    profiles:
      default:  { model: claude-opus-5-5, effort: high }
      operator: { model: claude-sonnet-5-5, effort: medium }
      doing:    { model: claude-sonnet-5-5, effort: high }
      review:   { model: claude-opus-5-5, effort: high }
      hydrate:  { model: claude-sonnet-5-5, effort: high }
      fast:     { model: claude-sonnet-5-5, effort: medium }
```

New:

```yaml
    profiles:
      default:  { model: opus,   effort: high }
      operator: { model: sonnet, effort: medium }
      doing:    { model: sonnet, effort: high }
      review:   { model: opus,   effort: high }
      hydrate:  { model: sonnet, effort: high }
      fast:     { model: sonnet, effort: medium }
```

Efforts are unchanged. The comment block above the claude `profiles:` currently argues the explicit
efforts from model-specific defaults ("Opus 5.5 defaults to `medium` … Sonnet 5.5 defaults to
`high` …"). Rewrite it so it no longer names a specific model generation as the fill: the fills are
family **aliases** that `claude` resolves to the newest model in the family, so the model behind a
fill moves with the installed Claude Code — which makes the explicit efforts *more* load-bearing,
not less (the next model the alias points at may default to a different effort). The Opus 5.5 /
Sonnet 5.5 defaults may stay as the illustrative current example.

The top-of-file `providers` comment (lines ~61–66, "Non-claude fills are refreshed at KIT-RELEASE
cadence …") should state that claude's fills are aliases that track the tool, and only codex's and
agy's are concrete IDs refreshed at kit-release cadence (kimi ships none). The codex/agy/kimi blocks
are NOT changed.

### 2. `src/go/fab/internal/configupgrade/configupgrade.go` — legacy advert digest

The legacy-digest set currently ends:

```go
	// Current providers advert: the claude doing/hydrate fills moved to
	// claude-sonnet-5-5 (effort high), leaving default/review on
	// claude-opus-5-5 (261030-lc7k).
	"2f55e041d31fb0a03cd59b73feeef8759f78662c1276b2d5e83f4082c8ed1ec5": {},
```

Relabel that entry as a *previous* advert (no longer "Current") and add the new current providers
advert digest (computed from the re-rendered advert after the fill change and the prose change in
§4) with a comment like `// Current providers advert: the claude fills on the opus/sonnet family
aliases (261117-vyrt).` — following the exact pattern of #693 (54226b44), which is the template for
this whole change. The new digest must be computed from the binary's actual rendering (e.g. via the
existing test helper or by printing the digest), never hand-guessed. This makes `fab config upgrade`
recognise the old fence paragraph as generated and re-render it.

### 3. `src/go/fab/cmd/fab/resolution.go` — `model_alias` for a bare alias

**Gap found during intake (not stated in the Brief, but required by its expected output).** Today:

```go
	if alias := agent.ModelAlias(profile.Model); alias != profile.Model {
		resolution.ModelAlias = alias
	}
```

`model_alias` is set only when `ModelAlias()` *changed* the string. With the fill `opus`,
`ModelAlias("opus")` returns `"opus"` (verbatim pass-through), the condition is false, and
`model_alias` comes out **empty** — so `fab agent review -o yaml` would print `model: opus`,
`model_alias: ""`. On the native rung an empty alias means the orchestrator omits the Agent tool's
`model` parameter and silently inherits the session model, which defeats the whole role split. The
Brief requires `model_alias: opus`.

Fix in the resolver, keeping `agent.ModelAlias()` unchanged (Brief: "Keep `agent.ModelAlias()` as
is"): set `model_alias` when the mapping changed the string **or** when the model already *is* one
of the four Agent-tool aliases (`opus`, `sonnet`, `haiku`, `fable`). Shape: a small exported helper
in `src/go/fab/internal/agent/agent.go` next to `modelAliasPrefixes` — e.g.
`func IsModelAlias(model string) bool` (or a `NativeModelAlias(model) string` returning the alias
or `""`) — derived from the same `modelAliasPrefixes` table's `alias` column so the four names are
not restated as magic strings. Non-Claude and unrecognised models (`gpt-5.6-sol`,
`glm-latest[1m]`) still get an empty `model_alias` (F3 later builds its descend rule on exactly
that). The legacy `fab resolve-agent --alias` line protocol already uses `ModelAlias()` directly and
emits `model=opus` for an `opus` input — unchanged.

### 4. `src/go/fab/internal/configref/configref.go` — `fab config explain providers` prose

Current (around line 787–793):

```
# claude's and codex's maps are dense (every role pins its own model); agy's is
# SPARSE: a role absent from it takes agy's `default` entry. Those fills are
# refreshed at KIT-RELEASE cadence and pass through
# unvalidated — override one with providers.<name>.profiles.<role>.model to pin a
# newer model. kimi deliberately ships NO fills …
```

Rewrite so it says: claude's fills are the family **aliases** (`opus` / `sonnet`) that `claude`
resolves to the newest model in the family, so they track the installed tool with no kit release;
only codex's and agy's are **concrete IDs**, refreshed at KIT-RELEASE cadence; all fills pass
through unvalidated and are overridable with `providers.<name>.profiles.<role>.model` (to pin a
specific model). Keep the phrases the existing test pins where they stay true
(`FOUR built-in providers`, `providers.<name>.profiles.<role>.model`, `EVERY provider's fill map is
printed below, claude's included`, `kimi deliberately ships NO fills`); keep the literal
`KIT-RELEASE cadence` attached to codex/agy. Update `src/go/fab/cmd/fab/config_test.go` (~line 810,
the phrase list and its error message "the non-claude fills ship, and are refreshed at kit
cadence") to also pin the new alias claim. The per-provider note "codex — … Its -m takes a concrete
model SLUG, so the fills below are pinned IDs" stays.

Changing this prose changes the advert's rendered bytes → this is why §2's new current digest must be
computed *after* §1 and §4 are both done.

### 5. Tests (`src/go/fab/internal/agent/agent_test.go`, `src/go/fab/cmd/fab/*_test.go`)

- `TestDefaultRoleProfilesArePinned` (agent_test.go:63): expectations become `opus` (default,
  review) and `sonnet` (operator, doing, hydrate, fast); efforts unchanged.
- `TestModelAlias` (agent_test.go:1294): add cases `"opus"→"opus"`, `"sonnet"→"sonnet"`,
  `"haiku"→"haiku"`, `"fable"→"fable"` (pass through unchanged — the Brief's required test).
- New resolver test (in `src/go/fab/cmd/fab/`, next to the existing `fab agent -o yaml` tests):
  `fab agent review -o yaml` with no config overrides yields `model: opus`, `model_alias: opus`, and
  a `command:` containing `--model opus`. Add a negative case that a non-Claude model (e.g.
  `--provider codex`) still yields an empty `model_alias`, plus a unit case for the new helper.
- `src/go/fab/internal/agent/stagemodels_doc_test.go` is the drift guard between
  `docs/specs/stage-models.md` § Default role profiles and `defaults.yaml` — it passes once the spec
  table is updated (§6).
- Sweep `src/go/fab/cmd/fab/resolve_agent_test.go`: tests derive expectations via
  `wantRoleModel`/`agent.ModelAlias(...)`, so most keep passing, but names/comments that assert
  "full ID" semantics for the default fill (e.g. `TestResolveAgentNoAliasEmitsFullID`'s comment,
  the `// apply ∈ doing → opus` comment) must be checked and corrected where they now misdescribe.
- `TestNonClaudeProviderFillsArePinned` is untouched (codex/agy fills unchanged).

Run `go test ./...` scoped to `src/go/fab/internal/agent`, `internal/configupgrade`,
`internal/configref`, `cmd/fab` first, then the full fab module.

### 6. Docs, spec, memory, skill — quoted Claude IDs and the "versioned IDs" rule

- `docs/specs/stage-models.md` § Default role profiles (lines ~129–134): the table's Model column
  becomes `opus` / `sonnet`. The "Why these defaults" paragraph's effort argument ("Opus 5.5
  defaults to `medium` … Sonnet 5.5 defaults to `high`") is reworded so it doesn't claim the fill
  IS 5.5 — the alias resolves to the current family model; keep the effort rationale. Also lines
  ~778 and ~810 (the `model_alias` / "dispatch embeds the FULL model ID" claims) — the dispatch
  command embeds the **resolved model verbatim** (now the alias for the built-in Claude fills); fix
  wording that asserts the built-in resolution yields a full versioned ID. § Refreshing the
  non-claude fills stays (it is already non-claude scoped) but may gain one sentence that claude
  needs no refresh because its fills are aliases. Spec edits are authorised by the plan's Brief.
- `docs/memory/runtime/providers-and-profiles.md`: line ~212 "Model IDs are written **versioned**
  (e.g. `claude-sonnet-5-5`); bare family IDs (`claude-sonnet`) fail both dispatch seams" becomes a
  three-form rule — the family **aliases** `opus`/`sonnet`/`haiku`/`fable` (valid on both seams;
  the built-in fills use them), full versioned IDs (`claude-sonnet-5-5`, pin a specific model), and
  bare family IDs (`claude-sonnet`) which remain invalid. Design Decisions at ~627–650 that quote
  `claude-opus-5-5`/`claude-sonnet-5-5` as the current fill get a new superseding DD ("Claude fills
  are family aliases") recording Decision/Why/Rejected/Introduced-by (261117-vyrt); the older DDs
  stay as history but their present-tense claims about the current IDs are corrected (hydrate's job,
  noted here for scope).
- `docs/memory/_shared/configuration.md` line ~242: the same "Model IDs are written versioned …"
  sentence — same three-form rewrite.
- `src/kit/skills/_cli-fab.md` lines ~323–380 (`fab resolve-agent` examples and the `--alias`
  paragraph): the examples print `model=claude-sonnet-5-5` for `apply`; they become `model=sonnet`.
  The `--alias` paragraph's claim "The default (flag absent) is the full ID (the `claude` CLI
  `--model` flag and interactive session launches consume full IDs)" and "The `dispatch=` line
  ALWAYS embeds the FULL model ID even under `--alias`" must be corrected: flag-absent emits the
  **resolved model verbatim** (an alias for the built-in Claude fills, a full ID when one is
  configured); the `dispatch=` line embeds the resolved model verbatim and is never aliased by
  `--alias`. Also sweep `_cli-fab.md` § fab agent for any `model_alias` example values. This is a
  deployed file — no fab-kit-only path citations (Constitution V).
- Sibling sweep (code-quality.md § Sibling Sweeps): grep `claude-opus-5-5`, `claude-sonnet-5-5`,
  "full ID", "versioned" across `src/kit/skills/`, `docs/specs/`, `docs/memory/`, `docs/site/`
  and fix every occurrence that describes the *current built-in fill*. Leave occurrences that are
  historical (archived change folders, DD history) or that illustrate a user pin.

### Non-goals

- Changing codex / agy / kimi fills.
- Validating model strings (fab stays verbatim pass-through).
- Changing `agent.ModelAlias()`'s behaviour.
- Anything in D1/F3/F2/N1 or the hexokit H* steps; no `provider` → `cli` rename.
- A `src/kit/migrations/` entry: no user data is restructured — fills live in the binary, and the
  fence re-render is handled by the legacy digest (§2), as in every prior fill bump. A user's own
  explicit pin (e.g. `providers.claude.profiles.review.model: claude-opus-5-5`) is left alone;
  removing pins is the user's U1 step / hexokit's H1.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) Claude fills are the `opus`/`sonnet` family aliases;
  three-form model-ID rule (alias / versioned / invalid bare family ID); `model_alias` is set for a
  model that already is an Agent-tool alias; new Design Decision superseding the versioned-ID fill DDs
- `_shared/configuration`: (modify) the "Model IDs are written versioned" sentence becomes the
  three-form rule

## Impact

- **Go**: `src/go/fab/defaults.yaml`, `src/go/fab/internal/configupgrade/configupgrade.go`,
  `src/go/fab/internal/configref/configref.go`, `src/go/fab/internal/agent/agent.go` (new helper),
  `src/go/fab/cmd/fab/resolution.go`; tests in `internal/agent/agent_test.go`,
  `cmd/fab/config_test.go`, `cmd/fab/resolve_agent_test.go` (and/or the `fab agent` yaml test file),
  plus the drift guard `internal/agent/stagemodels_doc_test.go` via the spec table.
- **Kit (deployed)**: `src/kit/skills/_cli-fab.md`.
- **Docs**: `docs/specs/stage-models.md`, `docs/memory/runtime/providers-and-profiles.md`,
  `docs/memory/_shared/configuration.md`.
- **Runtime behaviour**: every un-overridden Claude role now resolves `opus`/`sonnet`; the composed
  claude commands carry `--model opus` / `--model sonnet`; native dispatch passes the Agent tool
  `model: opus|sonnet` (unchanged values, since the old IDs aliased to the same names). Users'
  system fences re-render on `fab config upgrade`.
- **Downstream**: hexokit's suggestion list (built from fills) will show `opus`/`sonnet`; a user pin
  of `claude-opus-5-5` becomes "not in the list" — H1 handles that wording. D1/F3/F2 build on this.

## Open Questions

- None blocking. The `model_alias` resolver fix (§3) is a gap in the Brief's wording that its own
  expected output resolves; recorded as an assumption below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fills: `opus` for default/review, `sonnet` for doing/hydrate/operator/fast; efforts unchanged | Stated verbatim in the plan's F1 Brief and binding Decisions table | S:95 R:90 A:95 D:95 |
| 2 | Certain | Codex/agy/kimi fills, `agent.ModelAlias()` behaviour, and model-string validation are untouched | Brief non-goals + "Keep `agent.ModelAlias()` as is" + plan Decision "Codex/agy fills stay concrete" | S:95 R:90 A:95 D:95 |
| 3 | Confident | Fix `model_alias` in `cmd/fab/resolution.go` (set it when the model already is one of the four aliases) via a small helper beside `modelAliasPrefixes`, rather than changing `ModelAlias()` | Today's `alias != model` guard yields an empty `model_alias` for `opus`, contradicting the Brief's required `model_alias: opus`; the Brief forbids changing `ModelAlias()`, so the resolver is the only place left | S:70 R:85 A:80 D:70 |
| 4 | Confident | Relabel the 261030-lc7k digest as previous and add a newly computed current digest, computed after both the fill and prose changes | Follows the #693 (54226b44) precedent exactly; the prose change alters the advert bytes too | S:85 R:90 A:85 D:85 |
| 5 | Confident | No `src/kit/migrations/` entry; user pins left alone | Prior fill bumps (#684/#687/#693) shipped none; fills live in the binary; pin removal is U1/H1 per the plan | S:75 R:85 A:85 D:80 |
| 6 | Confident | Memory/spec/skill prose that claims built-in resolution yields a "full versioned ID" is corrected to "resolved model verbatim", and the "write versioned IDs" rule becomes a three-form rule (alias / versioned / invalid bare family) | Brief requires updating these files "wherever they quote the Claude IDs"; leaving the versioned-only rule would contradict the new fills | S:70 R:85 A:80 D:75 |
| 7 | Confident | Older memory Design Decisions are kept as history with a new superseding DD (Introduced by 261117-vyrt) | FKF present-truth style plus Design Decision history convention in providers-and-profiles.md | S:60 R:90 A:75 D:70 |

7 assumptions (2 certain, 5 confident, 0 tentative, 0 unresolved).
