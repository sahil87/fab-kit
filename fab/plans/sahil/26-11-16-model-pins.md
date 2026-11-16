# Agent models: stop config breaking on every model release

> Plan doc, written 2026-11-16. Spans fab-kit (F*, N1, D1) and hexokit (H*). Written to be executed
> by the operator; each step is one fab change started from its **Brief** below.

**Problem.** Every new model release breaks someone's fab agent config. Nothing in fab validates
model strings, so a stale one only fails when a stage dispatches. Six things cause it:

1. **Pins are the default interaction.** HexoKit's Settings → Agents role table writes
   `agent.profiles.<role>.model` the moment a model is picked. That key outranks the kit's fills,
   so the pin never moves when the kit ships a newer model.
2. **The kit's Claude fills are concrete IDs** (`claude-opus-5-5`, `claude-sonnet-5-5`). Even an
   unpinned config waits for a kit release; the providers advert has been re-rolled four times for
   Claude fill changes alone (`configupgrade.go` legacy digests, 260908-wcib → 261030-lc7k).
3. **The model list is the kit's guess, not the tool's truth.** The UI's suggestions are the kit's
   fills, so they lag the tools (codex's own default is already `gpt-6.1-sol`; fab's fill is
   `gpt-5.6-sol`) and never show models the user added (custom Claude Code models, kimi aliases).
4. **A tool change leaves the other tool's model behind.** Switching a role from codex to claude
   while `model: gpt-5.6-sol` is pinned gives `claude --model gpt-5.6-sol`. The UI only shows a ⚠.
5. **There is no way to say "use the tool's own default"** for a tool that only takes concrete
   slugs (codex, agy). An empty value is refused by `fab config set`.
6. **"Provider" is the wrong word.** fab's `providers:` entries are CLI coding tools (claude, codex,
   agy, kimi), not model providers. The tools themselves use "provider" for the model vendor
   (kimi's `[providers.openrouter]`, Claude Code routing to Fireworks), so the word collides.

Plus a UI bug the user hit: the model dropdown lists every model on first pick, but only one when
you come back to edit.

---

## Decisions (binding for every step)

| Decision | Choice | Why |
|---|---|---|
| Inherit is the default | A role row with a tool and no model is the intended shape (`review: { cli: codex, effort: high }`). The UI's first model option is "inherit", and picking it **unsets** the key | The kit's per-role fills split models by role (astra for default/doing, sol for review/hydrate, luna for operator/fast). Inheriting them moves with kit releases |
| Claude fills use aliases | `opus` / `sonnet` instead of full IDs | `claude --model` accepts aliases and resolves the newest model; the Agent tool's `model` enum takes exactly these; `agent.ModelAlias()` (`src/go/fab/internal/agent/agent.go:412`) passes them through unchanged |
| Codex/agy fills stay concrete | Unchanged | Their `-m` takes a concrete slug; no "latest" alias exists. F2 adds the opt-out |
| **Ask the tool for its models** | fab owns discovery (`fab agent models`, D1), with one adapter per built-in tool and an optional `models_command` for user-defined tools. hexokit only calls fab, never reads tool configs itself | Same rule as today's Agents tab (fab owns the file, hexokit wraps the CLI). One implementation serves the UI, the operator and `fab config explain` |
| Discovery fails open | Any discovery error or timeout returns the kit's fills plus a warning | Discovery reads unstable surfaces (`codex debug models` is a debug command); it must never block resolution or the settings page |
| Pins remain possible | Picking a concrete model or typing a custom one still writes a pin | Some users want a fixed model. The UI flags a pin the tool no longer offers |
| Tool change resets model and effort | Same POST as the tool write: `{cli: X, model: null, effort: null}`, keeping a value only if X offers it | Removes the cross-tool stale pin |
| **Rename "provider" → "cli"** | Config `providers:` → `clis:`, `agent.profiles.<role>.provider` → `.cli`, flag `--provider` → `--cli`, `fab agent -o yaml` key `provider:` → `cli:`. UI label "CLI". Old spellings stay readable and settable for at least one minor release, with a deprecation warning | Precedents: `session_command` → `interactive_command` and `agent.tiers` → `agent.profiles` used per-field aliases plus a migration. Here `set`/`unset` must also accept the old key during the window (unlike 2.19.0), because released hexokit writes it |
| hexokit reads, never re-derives, where it can | Prefer fab's own rows and verbs over re-implementing precedence or discovery | Re-implementation caused the agy inherit-label bug (H2) |

---

## Current state (verified 2026-11-16)

- **fab-kit** at v2.28.18. Claude fills: `src/go/fab/defaults.yaml:133-138`. Codex fills: default/doing
  `gpt-6-astra`, review/hydrate `gpt-5.6-sol`, operator/fast `gpt-5.6-luna`. agy is sparse
  (`default` + `fast` only). kimi has no fills.
- **Resolution** precedence per field: invocation flag > `agent.profiles.<role>` > `providers.<p>.profiles.<role>` >
  `providers.<p>.profiles.default` > empty (empty drops the `{model}` token so the tool's default applies).
  Verified with `fab agent review -o yaml` in a scratch HOME: `{ provider: codex }` alone resolves
  `model: gpt-5.6-sol`, `effort: xhigh` from the kit fill.
- **Empty refused:** `fab config set --system agent.profiles.review.model ""` exits 1 with
  `refusing to set … to an empty value … run fab config unset … instead`.
- **What each tool can report about its models** (probed on this machine):

  | Tool | Source | What it gives |
  |---|---|---|
  | codex | `codex debug models` (JSON) | `models[]` with `slug`, `display_name`, `visibility` (`list`/`hide`), `default_reasoning_level`, `supported_reasoning_levels[].effort` (per model: `low…max`, some add `ultra`). Lists `gpt-6.1-sol`, `gpt-6.1-terra`, `gpt-6-sol`, `gpt-6-luna`, … A `debug` subcommand, so not a stable contract. `~/.codex/config.toml` `model` gives the tool's default |
  | claude | no listing command | Built-in aliases (`opus`, `sonnet`, `haiku`, `fable`) plus `modelPicker.options[].{model,label,description}` in `~/.claude/settings.json` (this machine: 10 custom models such as `glm-latest[1m]`, `kimi-latest[1m]`, `auto[1m]`; `replaceBuiltInOptions: false` keeps the aliases). Top-level `model` is the tool's default |
  | kimi | `~/.kimi-code/config.toml` | `[models."<alias>"]` tables (367 on this machine) and `default_model` |
  | agy | not installed here | Unknown; fills only until probed |

- **Native rung with a custom Claude model:** `fab agent apply -o yaml` with `dispatch.mode: native`
  and `agent.profiles.doing.model: glm-latest[1m]` resolves `provider: claude`, `model_alias: ""`
  and **no `dispatch:` block** (the native rung). The Agent tool's `model` only accepts the four
  aliases, so the custom model cannot be honoured on that rung. Not verified: whether the
  orchestrating skill then inherits the session model silently, or descends. F3 settles it.
- **hexokit Agents tab:** frontend `app/frontend/src/components/settings-agents-panel.tsx`, backend
  `app/backend/internal/fabsysconfig/fabsysconfig.go`, API `app/backend/api/fab_config.go`, tests
  `settings-agents-panel.test.tsx`, `tests/e2e/settings-agents.spec.ts`, stub `tests/e2e/_fab-config-stub.ts`.
  - **The dropdown bug:** the model cell is `<input list=…>` with a `<datalist>`
    (`settings-agents-panel.tsx:439-490`). The input is pre-filled with the saved override, and
    browsers filter a datalist by the input's current text, so on edit only the saved model is
    listed. The backend's `modelSuggestions` is correct.
  - **Suggestions come from fills only:** `modelSuggestions` (`fabsysconfig.go:407`) collects
    `providers.<p>.profiles.*.model` values, so the list is at most the kit's fills.
  - **The inherit label bug:** `buildRole` (`fabsysconfig.go:437-454`) computes `Inherited` from
    `providers.<p>.profiles.<role>.<leaf>` only and skips `profiles.default`. With an override
    set, agy's `doing` row reads "↳ provider default", but fab resolves `gemini-3.1-pro-high`.
  - **Effort for tools without `{effort}`:** the effort select always offers `low…max`, including
    for agy and kimi, whose commands carry no `{effort}`.
  - **Rename exposure:** `providerNames` derives the tool list from `providers.` rows of
    `fab config show --json`, and writes go to `agent.profiles.<role>.provider`. A fab release
    that renames both without aliases empties the tool list and fails every tool write. rk's spawn
    path is not exposed: `riff.ResolveAgent` reads only `command` and `skill_prefix` from
    `fab agent -o yaml`, never `provider:`.
- **Rename size:** "provider" appears ~1,000 times in fab-kit Go (non-test), ~490 in `src/kit`,
  ~1,400 in docs; ~130 in hexokit's Agents tab code.
- **User's machine config** (`~/.fab-kit/config.yaml`): `agent.profiles.review: { provider: codex, model: gpt-5.6-sol, effort: high }`.
  The model line is a pin equal to today's fill. Removing it is step U1.

---

## Rules for the operator executing this plan

1. Each step except U1 is one fab change, started with `/fab-new` in the step's repo, using the
   step's **Brief** as the description. Spawn with this plan's absolute path so the agent can read
   it. In the fab-kit checkout it is `fab/plans/sahil/26-11-16-model-pins.md`; hexokit steps get the
   absolute path of this file in the fab-kit checkout, since the plan does not exist in hexokit.
2. Steps marked **user gate** stop and ask. Everything else proceeds when its dependencies are met.
3. Merge mode: `cherry-pick-ladder` (PRs created, merged after the user's review). A cross-repo
   dependency means "merged **and released**", not just merged.
4. Same-repo steps that edit the same files run in order, not in parallel: fab-kit
   F1 → D1 → F3 → F2 → N1; hexokit H1 → H2 → H5 → H4 → H3. The rename comes **last** so the
   steps that fix the user's problem never wait on it; D1/F3/F2 use today's `provider` spelling
   and N1 renames them with everything else.
5. **N1 must not be released before H5 is released.** H5 makes hexokit read both spellings, so the
   Agents tab keeps working on either side of the rename.
6. U1 edits the user's machine config. The operator never does it; it reminds the user once.

---

## Steps and dependencies

| ID | Repo | Step | Depends on | Size | Gate |
|---|---|---|---|---|---|
| **U1** | — | User removes the review model pin | — | XS | **user action** |
| **F1** | fab-kit | Claude fills → `opus` / `sonnet` aliases | — | S | — |
| **H1** | hexokit | Role model cell: inherit-first list that opens unfiltered, custom entry, stale-pin flag, tool change resets model/effort | — | M | — |
| **H2** | hexokit | Backend: inherit label honours `profiles.default`; effort disabled for tools without `{effort}` | H1 | S | — |
| **H5** | hexokit | Read and write both `providers`/`provider` and `clis`/`cli` spellings; relabel "Provider" → "CLI" | H2 | S | — |
| **D1** | fab-kit | `fab agent models`: ask each tool for its models | F1 | M | — |
| **F3** | fab-kit | Native rung descends when the model has no Agent-tool alias | D1 | S | — |
| **F2** | fab-kit | Reserved "use the tool's default" model value | F3 | S | **user gate** (pick the token) |
| **N1** | fab-kit | Rename provider → cli across config, flags, output, skills and docs, with a deprecation window | F2, H5 **released** before N1 releases | L | — |
| **H4** | hexokit | Suggestions and per-model efforts from `fab agent models` | D1 released, H5 | M | — |
| **H3** | hexokit | Offer the F2 value in the model list | F2 released, H4 | S | — |

F1 and H1 start at once. H1 fixes the bug the user hit and needs nothing from fab. N1 is last on
purpose: it is the largest step, it waits on a hexokit release, and nothing the user feels depends on it.

---

## Briefs

### U1 — Remove the pin (user)

Run `fab config unset --system agent.profiles.review.model`. Then `fab agent review -o yaml`
should show `model: gpt-5.6-sol` with `source.model: providers.codex.profiles.review`, and
`effort: high` still from `agent.profiles.review`.

### F1 — Claude fills use aliases (fab-kit)

- `src/go/fab/defaults.yaml`: the six `providers.claude.profiles.*` fills become `opus` (default,
  review) and `sonnet` (doing, hydrate, operator, fast). Efforts unchanged.
- `configupgrade.go`: add the previous providers-advert digest to the legacy list (the
  261030-lc7k one becomes legacy) so `fab config upgrade` re-renders the fence, as every prior
  fill bump did.
- Keep `agent.ModelAlias()` as is; add a test that `opus`/`sonnet`/`haiku`/`fable` pass through
  unchanged and `fab agent review -o yaml` gives `model: opus`, `model_alias: opus`, with the
  rendered command carrying `--model opus`.
- Update `agent_test.go` expectations, `docs/specs/stage-models.md` (default role profiles),
  `docs/memory/runtime/providers-and-profiles.md`, `docs/memory/_shared/configuration.md`, and
  `src/kit/skills/_cli-fab.md` wherever they quote the Claude IDs. The `fab config explain providers`
  prose that says fills are "refreshed at KIT-RELEASE cadence" should say Claude's are aliases that
  track the tool, and only codex's and agy's are concrete IDs.
- Non-goals: changing codex/agy/kimi fills; validating model strings.

### H1 — Role model cell (hexokit)

In `settings-agents-panel.tsx`:

- Replace `RoleModelInput`'s `<input list>` + `<datalist>` with a list control that **always opens
  unfiltered**, whatever the saved value. Options in order:
  1. `↳ inherit (<inherited>)`, which commits `null` (unset);
  2. every suggestion for the row's effective tool;
  3. the current override when it is not a suggestion, labelled `<value> (not offered)`;
  4. `Custom…`, which swaps the cell for a free-text input (same `useTextSettingDraft` commit
     path, Escape returns to the list).

  Use a plain `<select>` while the list is short. Build it so H4 can switch to a searchable list
  (kimi reports hundreds of aliases) without changing the option order or commit behaviour.
- **Stale pin:** replace the bare ⚠ (`mismatch`) with an inline note: "pinned to `<value>`, which
  `<tool>` doesn't offer — inherit `<inherited>` instead?" plus a one-click action that commits
  `null`. Show it only when the suggestion list is non-empty and lacks the override. For claude
  the wording is "not in the list — inherit `<inherited>` to track the latest", because Claude Code
  accepts any full model ID: a pinned `claude-opus-5-5` is stale, not invalid.
- **Tool change:** `RoleRow`'s tool `onChange` sends one patch `{provider: v, model: null, effort: null}`,
  leaving out the model key when the current model override is offered by the new tool and the
  effort key when the effort override is in `doc.effortSuggestions`. Picking "inherit" for the
  tool follows the same rule against the depth knob's tool.
- Tests: unit cases for each option kind, the tool-change patch shape (kept vs cleared), and the
  stale-pin action; an e2e case in `settings-agents.spec.ts` that sets a model, reloads the tab,
  opens the list and sees every suggestion (the regression for the bug).
- Non-goals: backend changes (H2); discovery (H4); the tool-default option (H3).

### H2 — Backend inherit label and effort capability (hexokit)

In `fabsysconfig.go`:

- `buildRole`: when `providers.<p>.profiles.<role>.<leaf>` is missing, fall back to
  `providers.<p>.profiles.default.<leaf>` when computing `Inherited` (and `Effective` when no row
  exists). Test with agy `doing` (expect `gemini-3.1-pro-high`) and kimi (expect empty → "tool default").
- Add `effortCapable map[string]bool` to `Document`: true when the tool's `interactive_command`
  or `headless_command` contains `{effort}`. The frontend disables the effort select (title "this
  tool takes no effort setting") when false; the reset link still clears a stale effort override.
- Non-goals: per-model effort lists (H4).

### H5 — Survive the rename (hexokit)

- `fabsysconfig.go`: derive tool names from both `providers.<name>.*` and `clis.<name>.*` rows;
  read a role's tool from `agent.profiles.<role>.cli`, falling back to `.provider`. Write the
  spelling the installed fab accepts: probe once per `Read` (presence of any `clis.` row, or
  `fab config explain --json` listing the key) and keep the choice on the `Client`.
- Accept both spellings in `isProfileKey` / the patch keys; the frontend sends a neutral `tool` leaf
  and the backend maps it.
- Relabel the column and chips "CLI", and the knobs "Session CLI" / "Worker CLI".
- Tests: the stub serves each spelling; reads and writes work against both.
- Must be **released** before N1 is released (rule 5).

### N1 — Rename provider → cli (fab-kit)

The word is decided: `cli` (user, 2026-11-16), label "CLI", prose "CLI tool".

- Config: `providers:` → `clis:`; `agent.profiles.<role>.provider` → `.cli`. Register both in
  `configref` with `renamed_from`; the resolver reads either spelling per field, new wins.
  `fab config set`/`unset` accept the old key during the window, writing the new key and warning
  `<old> is deprecated, wrote <new>`. Add a `src/kit/migrations/` entry that rewrites user files.
- CLI: `--provider` → `--cli` on `fab agent`; keep `--provider` as a hidden deprecated alias.
- Output: `fab agent -o yaml` emits `cli:` and, for the window, also `provider:` with the same
  value; `source.provider` → `source.cli` likewise. Everything inside fab and the kit's skills
  that parses this output moves to `cli:` in the same change.
- Rename Go identifiers, skill prose (`_cli-fab*.md`, `_preamble.md`, `fab-setup.md`, the
  operator skills), specs and memory. Where "provider" means a model vendor, leave it.
- `docs/specs/glossary.md` gains "CLI tool" and records the rename.
- Non-goals: any behaviour change. A pure rename plus compatibility. rk's own "provider" naming
  for agent CLIs (`cmd/rk/agent_registry.go`, `riff`) is rk's business and out of scope; hexokit
  relabels only the fab-facing Agents tab (H5).
- Do not release until H5 is released (rule 5). State the deprecation window in the release notes.

### D1 — Ask each tool for its models (fab-kit)

New verb `fab agent models [<cli>] [--json]`, listing what each tool offers:

- Output rows: `{cli, id, label, efforts, default, source}` where `source` is `tool` (discovered),
  `alias` (built-in alias) or `fill` (kit fill). Kit fills are always included, so the list is
  never smaller than today's. `efforts` is per model when the tool reports it, else the tool's
  known set, else empty. `default: true` marks the tool's own default model.
- Built-in adapters, run with a short timeout, each failing open (fills plus a warning line):
  - **codex:** `codex debug models`; keep `visibility == "list"`; `efforts` from
    `supported_reasoning_levels[].effort`; default from `~/.codex/config.toml` `model`.
  - **claude:** the four aliases, plus `modelPicker.options[]` from `~/.claude/settings.json`
    (`label` from the option; drop the aliases when `replaceBuiltInOptions: true`); default from
    top-level `model`. Efforts `low…max`.
  - **kimi:** `[models."<alias>"]` keys of `~/.kimi-code/config.toml`; default from `default_model`.
  - **agy:** fills only until someone probes what agy exposes.
- Config locations honour the tools' own overrides: `CLAUDE_CONFIG_DIR` (default `~/.claude`;
  merge `settings.local.json` over `settings.json`), `CODEX_HOME` (default `~/.codex`), and
  kimi's `~/.kimi-code`.
- User-defined tools: optional `providers.<name>.models_command` that prints the same JSON rows
  (N1 renames the key with the rest).
- Read-only; no cache in v1 (callers fetch on demand). Note the codex surface is a debug command:
  pin its parse to the fields above and test against a recorded fixture.
- Tests: each adapter against fixtures, including malformed input and timeout.
- Document in `_cli-fab-operator.md` (the `fab agent` family), `stage-models.md`, memory.

### F3 — Native rung with a model it can't carry (fab-kit)

- First verify what happens today when the resolved Claude model has no Agent-tool alias
  (`glm-latest[1m]`) and the native rung is chosen: does the orchestrator run the sub-agent on the
  session model, or descend? Record the answer in the change's intake.
- Then: a non-empty model with an empty `model_alias` makes the native rung unavailable for that
  role, so resolution descends to the next rung the tool supports (headless for claude), and
  `fab agent -o yaml` says why (e.g. `dispatch.reason: model has no native alias`). An empty model
  keeps today's inherit behaviour, and so will F2's token once it lands (F2 treats it as empty
  before this rule runs).
- Non-goals: aliasing custom models.

### F2 — "Use the tool's default" value (fab-kit, user gate)

Ask the user for the token before starting (proposal: `cli-default`). Then:

- Resolution treats the token as empty for `{model}` substitution (the token is dropped and the
  tool's default applies) and for `model_alias` (empty → omit the Agent tool's `model` param, so
  the native rung inherits the session model).
- `fab config set` accepts it where it refuses `""`. It is a real value, so it wins over lower tiers.
- The token is exempt from F3's descend rule: it resolves to an empty model before the rung check.
- `fab agent -o yaml` reports `model: ""` with `source.model` naming the key that held the token.
- `fab agent models` (D1) shows it as the first row per tool, labelled with the discovered default
  (`tool default (gpt-6.1-sol)`).
- Document it in `fab config explain`, `stage-models.md`, `config.md`, memory and `_cli-fab*.md`.

### H4 — Suggestions from the tools (hexokit)

- `fabsysconfig.go`: fill `ModelSuggestions` from `fab agent models --json` (one call per `Read`,
  bounded by `FabTimeout`), falling back to today's fill scan when the verb is missing (older fab).
  Carry `label`, `efforts` and `default` per model; surface fab's warning lines in `Warnings`.
- Frontend: the model list shows labels (e.g. "GLM 5.3 (Latest)") with the id as secondary text,
  marks the tool's default, and turns searchable above ~30 entries (kimi). The stale-pin note now
  means "the tool doesn't offer it", not "the kit doesn't ship it".
- Effort select: offer the selected model's `efforts` when known (codex adds `ultra` for some
  models), else the tool's set; keep H2's disabled state for tools without `{effort}`.
- Non-goals: hexokit reading any tool's config itself.

### H3 — Offer the tool default in the UI (hexokit)

After F2 is **released**: add `↳ tool default (<discovered default>)` as the second option of the
model list, committing the F2 token. Take the token and label from `fab agent models --json`.
Display a row holding the token as "tool default", not the raw string.

---

## Open decisions

| Step | Question |
|---|---|
| N1 | Length of the deprecation window for the old spellings (one minor release proposed) |
| F2 | Token spelling (`cli-default` proposed). Must not collide with a real model id |
| D1 | Whether to cache discovery (codex's call is a process launch per settings-page visit). v1 says no |
| D1 | Whether `fab agent <stage> -o yaml` should warn on stderr when the resolved model is not in the discovered list — it would catch a stale pin before dispatch (the original complaint), at the cost of a tool launch per resolution. Candidate follow-up once D1 shows how fast discovery is |
