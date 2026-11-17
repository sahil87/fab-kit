# Intake: fab agent models — ask each tool for its models

**Change**: 261117-rhpz-agent-models-discovery
**Created**: 2026-11-17

## Origin

> Execute step D1 (fab agent models: ask each tool for its models) of the plan at /home/sahil/code/sahil87/fab-kit/fab/plans/sahil/26-11-16-model-pins.md. Read the whole plan first (Decisions and Rules bind you), then implement only the D1 Brief.

One-shot, operator-spawned `/fab-new`. The plan (also checked in at `fab/plans/sahil/26-11-16-model-pins.md`) is the design authority; its **Decisions** table and **Rules for the operator** bind this change. D1's dependency F1 (Claude fills → `opus`/`sonnet` aliases) is already on this branch (commit `9b0a1b6e`, "operator: cherry-pick mp-f1 dependency"): `src/go/fab/defaults.yaml` now ships `providers.claude.profiles.*.model` as `opus`/`sonnet`.

The D1 Brief, verbatim (the binding spec for this change):

> New verb `fab agent models [<cli>] [--json]`, listing what each tool offers:
>
> - Output rows: `{cli, id, label, efforts, default, source}` where `source` is `tool` (discovered), `alias` (built-in alias) or `fill` (kit fill). Kit fills are always included, so the list is never smaller than today's. `efforts` is per model when the tool reports it, else the tool's known set, else empty. `default: true` marks the tool's own default model.
> - Built-in adapters, run with a short timeout, each failing open (fills plus a warning line):
>   - **codex:** `codex debug models`; keep `visibility == "list"`; `efforts` from `supported_reasoning_levels[].effort`; default from `~/.codex/config.toml` `model`.
>   - **claude:** the four aliases, plus `modelPicker.options[]` from `~/.claude/settings.json` (`label` from the option; drop the aliases when `replaceBuiltInOptions: true`); default from top-level `model`. Efforts `low…max`.
>   - **kimi:** `[models."<alias>"]` keys of `~/.kimi-code/config.toml`; default from `default_model`.
>   - **agy:** fills only until someone probes what agy exposes.
> - Config locations honour the tools' own overrides: `CLAUDE_CONFIG_DIR` (default `~/.claude`; merge `settings.local.json` over `settings.json`), `CODEX_HOME` (default `~/.codex`), and kimi's `~/.kimi-code`.
> - User-defined tools: optional `providers.<name>.models_command` that prints the same JSON rows (N1 renames the key with the rest).
> - Read-only; no cache in v1 (callers fetch on demand). Note the codex surface is a debug command: pin its parse to the fields above and test against a recorded fixture.
> - Tests: each adapter against fixtures, including malformed input and timeout.
> - Document in `_cli-fab-operator.md` (the `fab agent` family), `stage-models.md`, memory.

Binding plan Decisions that shape this change:

- **Ask the tool for its models** — fab owns discovery, one adapter per built-in tool plus an optional `models_command` for user-defined tools; hexokit only ever calls fab (H4 consumes `fab agent models --json`), never reads tool configs itself.
- **Discovery fails open** — any discovery error or timeout returns the kit's fills plus a warning; it must never block resolution or the settings page.
- **Rule 4** — D1 uses today's `provider` spelling for config keys and flags (`providers.<name>.models_command`); N1 renames everything later. The new output *row* key is `cli` because the Brief names it so (see Assumption 4).
- **Out of scope here**: F3 (native-rung descent), F2 (the tool-default token), N1 (the rename), and the plan's open D1 follow-up (warning on stale pins from `fab agent <stage> -o yaml`).

## Why

1. **The pain.** Every model release breaks someone's fab agent config, and one root cause (plan Problem #3) is that the only model list anyone has is the *kit's guess*: hexokit's Settings → Agents suggestions are built from `providers.<p>.profiles.*.model` fills (`fabsysconfig.go` `modelSuggestions`). Those lag the tools (codex's own default on the user's machine is already `gpt-6.1-sol`; fab's review fill is `gpt-5.6-sol`) and never show models the user added — custom Claude Code `modelPicker` entries (`glm-latest[1m]`, `kimi-latest[1m]`, `auto[1m]`, …) or kimi's 367 `[models."<alias>"]` tables.
2. **The consequence of not fixing it.** Users can only pick what the kit last shipped, so they pin stale strings by hand; a stale pin is only detected when a stage dispatches. H4 (hexokit suggestions + per-model efforts) and F2 (tool-default row) both depend on a discovery verb that does not exist yet.
3. **Why this approach.** The plan's Decision puts discovery in fab, behind one verb, so one implementation serves the hexokit UI, the operator, and later `fab config explain` — the same "fab owns the file, hexokit wraps the CLI" split the Agents tab already uses. Rejected: hexokit reading `~/.codex`, `~/.claude`, `~/.kimi-code` directly (re-implementation caused the agy inherit-label bug, H2); caching (v1 says no — codex's call measured ~0.16 s on the user's machine).

## What Changes

### 1. New subcommand `fab agent models [<provider>] [--json]`

Registered as a cobra **child of `agentCmd()`** (`src/go/fab/cmd/fab/agent.go:89`), implemented in a new `src/go/fab/cmd/fab/agent_models.go`. `agentCmd` already has a custom `Args` validator and `RunE`; cobra resolves `fab agent models …` to the child before the parent's positional `[role|stage]` handling, and no role or stage is named `models`, so existing forms (`fab agent review -o yaml`, `fab agent -- --resume`, `fab agent --provider codex --print`) are unaffected — a regression test pins this.

- **Positional `[<provider>]`** (optional, at most one): limits output to that provider. Unknown name → non-zero exit naming the available providers, mirroring `fab agent --provider`'s unknown-provider lookup error. (The Brief writes `<cli>`; per Rule 4 the help text uses today's word, e.g. `fab agent models [provider] [--json]`, and N1 renames it.)
- **No positional**: every provider from `agent.ProviderNames(cfg)` (built-ins `claude`, `codex`, `agy`, `kimi` plus any user-defined `providers:` entry), in that function's order.
- **`--json`**: stdout is a single JSON **array** of rows (no envelope). Without `--json`: a human-readable, column-aligned table.
- **`--repo <path>`**: same semantics as `fab agent --repo` — reuses `loadRepoConfig(repo)` (`agent.go:400`), so the verb works **outside a fab project** via `config.LoadNoProject()` (env > system > built-ins). hexokit's settings page runs it without a project.
- **Warnings** (discovery failures): one line per failure on **stderr**, in both modes, shaped `warning: <provider>: <reason>` (e.g. `warning: codex: codex debug models: exec: "codex": executable file not found in $PATH — showing kit fills only`). H4 surfaces these lines in its `Warnings`.
- **Exit codes**: discovery failures never change the exit code (0). Non-zero only for usage errors, an unknown provider name, or a config-load error.
- **Read-only**: writes nothing; no cache.

Example (`--json`, abbreviated):

```json
[
  {"cli": "claude", "id": "opus", "label": "opus", "efforts": ["low","medium","high","xhigh","max"], "default": false, "source": "alias"},
  {"cli": "claude", "id": "glm-latest[1m]", "label": "GLM 5.3 (Latest)", "efforts": ["low","medium","high","xhigh","max"], "default": false, "source": "tool"},
  {"cli": "claude", "id": "claude-fable-5-1[1m]", "label": "claude-fable-5-1[1m]", "efforts": ["low","medium","high","xhigh","max"], "default": true, "source": "tool"},
  {"cli": "codex", "id": "gpt-6.1-sol", "label": "GPT-6.1 Sol", "efforts": ["low","medium","high","xhigh","max"], "default": true, "source": "tool"},
  {"cli": "codex", "id": "gpt-6-astra", "label": "GPT-6 Astra", "efforts": ["low","medium","high","xhigh","max","ultra"], "default": false, "source": "tool"},
  {"cli": "agy", "id": "gemini-3.1-pro-high", "label": "gemini-3.1-pro-high", "efforts": [], "default": false, "source": "fill"}
]
```

Human table (illustrative):

```
CLI     MODEL                 LABEL              EFFORTS                  SOURCE  DEFAULT
claude  opus                  opus               low,medium,high,xhigh,max alias
codex   gpt-6.1-sol           GPT-6.1 Sol        low,medium,high,xhigh,max tool    *
```

### 2. Row schema and merge rules

```go
type ModelRow struct {
    CLI     string   `json:"cli"`     // provider name (Brief's key; N1 renames the config side)
    ID      string   `json:"id"`      // the string fab would substitute into {model}
    Label   string   `json:"label"`   // tool-supplied display name; falls back to ID
    Efforts []string `json:"efforts"` // always an array in JSON (never null)
    Default bool     `json:"default"` // the TOOL's own default model
    Source  string   `json:"source"`  // "tool" | "alias" | "fill"
}
```

Per provider, rows are assembled in this order and **de-duplicated by `id`** (first occurrence keeps its position; the higher-precedence source wins: `tool` > `alias` > `fill`; `default` is OR-ed across duplicates; a non-empty label/efforts from the winner is kept, else taken from the duplicate):

1. Rows from the adapter (or `models_command`), in the tool's own order.
2. **Kit fills**: every distinct non-empty `model` across the *resolved* provider's `profiles.*` (i.e. `agent.ResolveProvider(cfg, name)` — built-in fills merged with any user `providers.<p>.profiles` overrides), sorted for determinism, `source: fill`. Fills are **always** included, so the list is never smaller than today's suggestion list. (Pins in `agent.profiles.<role>.model` are NOT included — they are not something the tool offers.)
3. **Tool default not otherwise listed**: if the tool reports a default whose id is in no row, append a row for it (`source: tool`, `default: true`) so a default is always representable.

`efforts` resolution per row: the model's own list when the tool reports one; else the tool's known set (claude's fixed set; codex's union is NOT invented — codex fills that discovery did not return get `[]`); else `[]`.

### 3. Built-in adapters (new package `src/go/fab/internal/agentmodels/`)

One file per adapter plus a shared runner; adapters are keyed by the **built-in provider name** (`claude`, `codex`, `kimi`, `agy`). Each adapter returns `([]ModelRow, defaultID string, err error)`; any error → the provider still gets its fills, plus one warning line. Process launches use `exec.CommandContext` with a named timeout constant (proposal: `discoveryTimeout = 3 * time.Second`); a timeout is reported as a warning (`timed out after 3s`). File-reading adapters need no timeout.

**codex** — run `codex debug models` (binary resolved from `$PATH`), parse stdout JSON. Parse is **pinned** to exactly these fields (it is a debug command, not a contract) and tested against a recorded fixture:

```json
{"models":[{"slug":"gpt-6.1-sol","display_name":"GPT-6.1 Sol","visibility":"list",
  "default_reasoning_level":"medium",
  "supported_reasoning_levels":[{"effort":"low"},{"effort":"medium"},{"effort":"high"},{"effort":"xhigh"},{"effort":"max"}]}]}
```

- keep only `visibility == "list"` (drops e.g. `gpt-reserve`, `codex-auto-review`, which are `hide`);
- `id` = `slug`, `label` = `display_name`, `efforts` = `supported_reasoning_levels[].effort` in order (some models add `ultra`);
- default = top-level `model` of `$CODEX_HOME/config.toml` (`CODEX_HOME` defaults to `~/.codex`); on the user's machine line 1 is `model = "gpt-6.1-sol"`. A missing config file is not an error (just no default); an unparseable one is a warning.
- Extra/unknown JSON fields are ignored; a missing `models` key or non-JSON stdout → warning + fills.

**claude** — no process launch; read `$CLAUDE_CONFIG_DIR/settings.json` (default dir `~/.claude`) and merge `settings.local.json` from the same dir over it (top-level keys `model` and `modelPicker` from the local file replace the base file's when present). Rows:

- the four built-in aliases `opus`, `sonnet`, `haiku`, `fable` (`source: alias`, label = id) — **omitted** when `modelPicker.replaceBuiltInOptions` is `true`;
- then each `modelPicker.options[]` entry: `id` = `.model`, `label` = `.label` (fallback id), `source: tool` (the `description` field is not carried);
- default = top-level `model` (e.g. `claude-fable-5-1[1m]`);
- `efforts` for every claude row = `["low","medium","high","xhigh","max"]` (Brief: "Efforts low…max").
- Missing settings files → just the aliases (+ fills), no warning; malformed JSON → warning + aliases + fills (aliases are built-in knowledge, not discovered, so they survive a broken file unless `replaceBuiltInOptions` was readable and true).

**kimi** — read `~/.kimi-code/config.toml` (no env override exists; the Brief names this path). Rows from `[models."<alias>"]` table headers (also accept bare `[models.<alias>]`): `id` = the alias key (e.g. `openrouter/nex-agi/nex-n2.6`), `label` = the table's `display_name` when present, `efforts` = the table's `support_efforts` array when present, `source: tool`. Default = top-level `default_model`. The user's file has 367 model tables; other tables (`[providers.openrouter]`, `[thinking]`, `[[hooks]]`) are ignored. The file contains API credentials in `[providers.*]` — the parser MUST NOT echo any non-model content into rows or warnings.

**agy** — no adapter: fills only, **no** warning (it is not a failure; the Brief says "fills only until someone probes").

**TOML reading** (codex `model`, kimi): a minimal line scanner in `internal/agentmodels` — tracks the current `[table]` header, reads top-level `key = "string"` before the first header, `[models."…"]` headers, and inside a model table `display_name = "…"` and `support_efforts = [ "…", … ]` (single-line arrays). No new Go module dependency (`src/go/fab/go.mod` has only cobra/pflag/yaml.v3). See Assumption 7.

### 4. User-defined tools — `providers.<name>.models_command`

- New optional field on `config.ProviderConfig` (`src/go/fab/internal/config/config.go:116`): `ModelsCommand string \`yaml:"models_command"\``. Merged per field in `agent.ResolveProvider` like the command fields (user value overrides; no built-in ships one).
- When set (for **any** provider, built-in included), it **replaces** that provider's built-in adapter. Run via `sh -c` with the same timeout. Stdout must be a JSON array of rows `{id, label?, efforts?, default?}`; `cli` and `source` in the command's output are ignored and set by fab (`cli` = provider name, `source: tool`). Rows with an empty `id` are dropped. Non-zero exit, timeout, or malformed JSON → warning + fills.
- Registered in `src/go/fab/internal/configref/configref.go`: the structured `providerDefault` projection gains `models_command` (omitempty; no built-in carries it), and the `providers` row's description / `fab config explain providers` prose documents it. It is NOT added to the advertised managed-fence template (no built-in default to advertise). No migration: it is a new optional key, no user data is restructured.

### 5. Documentation (sibling sweep)

- `src/kit/skills/_cli-fab.md` § `fab agent` — add the `fab agent models` signature, row schema, ordering/dedupe, fail-open + stderr warnings, `--json`, `--repo`, `models_command`. (The Brief says `_cli-fab-operator.md`; that file was removed with the operator surface in 261117-egu4 and `fab agent` is now documented in `_cli-fab.md` — Constitution Additional Constraints names `_cli-fab.md` as the owner. Deployed content must restate rules, never cite `src/go`/`docs/` paths.)
- `docs/specs/stage-models.md` — a short section on model discovery (`fab agent models`), the source taxonomy, fail-open, and `models_command`. Check `src/go/fab/internal/agent/stagemodels_doc_test.go` (a drift guard over this spec) still passes.
- `docs/specs/config.md` — mention `providers.<name>.models_command` if the spec enumerates provider fields.
- Memory (hydrate): see Affected Memory.

### 6. Tests (test-alongside, Constitution VII)

- `internal/agentmodels`: per adapter against fixtures under `testdata/` — codex (recorded `debug models` JSON incl. a `hide` row and an `ultra` effort; malformed JSON; missing `models`; timeout via a stub binary that sleeps on a temp `$PATH`; binary absent), claude (aliases only; options + label; `replaceBuiltInOptions: true`; `settings.local.json` override; malformed JSON; `CLAUDE_CONFIG_DIR` honoured), codex default via `CODEX_HOME`, kimi (quoted + bare headers, `display_name`/`support_efforts`, `default_model`, non-model tables ignored, malformed line), `models_command` (valid rows, `cli`/`source` overwritten, non-zero exit, malformed, timeout).
- Merge: dedupe precedence `tool > alias > fill`, default OR-ing, fills always present, default-not-listed row appended, `efforts` serialised as `[]` not `null`.
- `cmd/fab`: `fab agent models --json` / table / positional filter / unknown provider error / `--repo` / works with no fab project; regression that `fab agent review -o yaml` and `fab agent -- --resume` still route to the parent. Tests isolate `HOME`, `CLAUDE_CONFIG_DIR`, `CODEX_HOME`, `PATH` per case.
- `configref`/`config`: `models_command` loads and merges; `fab config show --json` / explain tests updated if they snapshot provider fields.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) new requirement for `fab agent models` — discovery verb, row schema/source taxonomy, per-tool adapters and config locations, fail-open with stderr warnings, `providers.<name>.models_command`; Design Decision entry for "fab owns discovery; fails open; no cache in v1".
- `_shared/configuration`: (modify) `providers.<name>.models_command` as an optional provider field.

## Impact

- **Go**: new `src/go/fab/internal/agentmodels/` package (+ `testdata/`); new `src/go/fab/cmd/fab/agent_models.go` (+ tests); `agent.go` registers the child command; `internal/config/config.go` (`ModelsCommand`); `internal/agent/agent.go` `ResolveProvider` merge; `internal/configref/configref.go` (projection + explain prose). No new module dependency.
- **External processes**: `codex debug models` and any `models_command` are launched (bounded by timeout) only when `fab agent models` runs — never during resolution, dispatch, or `fab config`.
- **Kit content**: `src/kit/skills/_cli-fab.md`.
- **Docs**: `docs/specs/stage-models.md`, possibly `docs/specs/config.md`, memory files above.
- **Downstream**: hexokit H4 consumes `fab agent models --json` (array of rows, warnings on stderr); F2 will later prepend a tool-default row per tool; N1 renames `models_command`'s parent key and the positional's help word.

## Open Questions

- None blocking. The plan's two D1 open decisions are settled for v1: no cache; the stale-pin warning in `fab agent <stage> -o yaml` is a follow-up, out of scope.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is the D1 Brief only; F3/F2/N1 and the stale-pin warning follow-up are excluded | User instruction ("implement only the D1 Brief") and plan Rules | S:95 R:85 A:95 D:95 |
| 2 | Certain | Discovery fails open: errors/timeouts yield fills + a warning, exit 0; never invoked from resolution/dispatch | Binding plan Decision | S:95 R:80 A:90 D:90 |
| 3 | Certain | Config key spelled `providers.<name>.models_command`; positional help uses "provider" | Plan Rule 4 (D1 uses today's spelling; N1 renames) and the Brief's own "(N1 renames the key with the rest)" | S:90 R:80 A:90 D:85 |
| 4 | Confident | Output row key is `cli` (not `provider`) despite Rule 4 | The Brief names the row fields `{cli, id, label, efforts, default, source}` explicitly; a brand-new output has no compat burden and H4 then never needs a rename | S:75 R:65 A:70 D:60 |
| 5 | Confident | `--json` emits a bare JSON array; warnings go to stderr as `warning: <provider>: <reason>` lines in both modes | Brief says `models_command` "prints the same JSON rows" (array) and "a warning line"; H4 says "surface fab's warning lines" | S:70 R:70 A:70 D:65 |
| 6 | Certain | Docs land in `_cli-fab.md` § fab agent instead of the Brief's `_cli-fab-operator.md` | That file no longer exists (removed in 261117-egu4); constitution now names `_cli-fab.md` as the `fab agent` owner | S:80 R:85 A:90 D:85 |
| 7 | Confident | Read TOML (codex `model`, kimi model tables) with a minimal in-package line scanner rather than adding a TOML module dependency | Only ~4 keys are needed and go.mod is deliberately small; a real TOML lib would be more robust to exotic syntax (multi-line arrays, inline tables). Easy to swap later | S:45 R:70 A:55 D:45 |
| 8 | Confident | Implemented as a cobra child of `agentCmd` in new `cmd/fab/agent_models.go` + new `internal/agentmodels` package | Brief names the verb `fab agent models`; no role/stage collides with `models`; package split keeps agent.go's resolver untouched | S:75 R:75 A:75 D:70 |
| 9 | Confident | Fills = distinct models across the *resolved* provider's `profiles.*` (user fill overrides included); `agent.profiles` pins excluded | "Kit fills are always included, so the list is never smaller than today's" — today's hexokit list is exactly the fill scan | S:70 R:80 A:70 D:65 |
| 10 | Confident | Dedupe by id with source precedence tool > alias > fill, default OR-ed; a reported default not in any row is appended as a `tool` row | Brief requires one row per model and `default: true` on the tool's default; claude's default (`claude-fable-5-1[1m]`) is often not an alias/option | S:60 R:80 A:70 D:60 |
| 11 | Confident | `models_command` replaces the built-in adapter when set on any provider; fab overwrites `cli`/`source` in its rows | User config should be able to override a built-in's discovery; source must stay truthful | S:60 R:80 A:70 D:65 |
| 12 | Confident | codex adapter launches `codex` from `$PATH` (not the first word of the configured `interactive_command`); timeout a named constant of ~3s | Brief says `codex debug models`; measured ~0.16s locally; "short timeout" | S:65 R:85 A:65 D:60 |
| 13 | Confident | kimi rows carry `display_name` as label and `support_efforts` as efforts when present | Brief: efforts "per model when the tool reports it"; the real file carries both keys per model table | S:65 R:85 A:75 D:70 |
| 14 | Confident | claude efforts fixed to `low, medium, high, xhigh, max`; missing settings files are not a warning | Brief: "Efforts low…max"; an absent optional file is not a discovery failure | S:70 R:85 A:70 D:70 |
| 15 | Confident | No migration and no managed-fence advert for `models_command` | New optional key, no user data restructured; no built-in default to advertise | S:70 R:80 A:80 D:75 |

15 assumptions (4 certain, 11 confident, 0 tentative, 0 unresolved).
