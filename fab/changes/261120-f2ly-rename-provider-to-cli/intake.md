# Intake: Rename provider → cli

**Change**: 261120-f2ly-rename-provider-to-cli
**Created**: 2026-11-20

## Origin

> Execute step N1 (Rename provider → cli across config, flags, output, skills and docs, with a
> deprecation window) of the plan at fab/plans/sahil/26-11-16-model-pins.md. Read the whole plan
> first (Decisions and Rules bind you), then implement only the N1 Brief.

One-shot `/fab-new`, driven by the plan doc `fab/plans/sahil/26-11-16-model-pins.md` (in this repo).
N1 is the last fab-kit step of that plan (order F1 → D1 → F3 → F2 → N1); F1, D1, F3 and F2 have
already merged (`git log`: #705 Claude aliases, #706 `fab agent models`, #707 native-rung descent,
#708 `tool-default` model value). The plan's **Decisions** table and **Rules** bind this change. The
operative N1 Brief, verbatim:

> The word is decided: `cli` (user, 2026-11-16), label "CLI", prose "CLI tool".
>
> - Config: `providers:` → `clis:`; `agent.profiles.<role>.provider` → `.cli`. Register both in
>   `configref` with `renamed_from`; the resolver reads either spelling per field, new wins.
>   `fab config set`/`unset` accept the old key during the window, writing the new key and warning
>   `<old> is deprecated, wrote <new>`. Add a `src/kit/migrations/` entry that rewrites user files.
> - CLI: `--provider` → `--cli` on `fab agent`; keep `--provider` as a hidden deprecated alias.
> - Output: `fab agent -o yaml` emits `cli:` and, for the window, also `provider:` with the same
>   value; `source.provider` → `source.cli` likewise. Everything inside fab and the kit's skills
>   that parses this output moves to `cli:` in the same change.
> - Rename Go identifiers, skill prose (`_cli-fab*.md`, `_preamble.md`, `fab-setup.md`, the
>   operator skills), specs and memory. Where "provider" means a model vendor, leave it.
> - `docs/specs/glossary.md` gains "CLI tool" and records the rename.
> - Non-goals: any behaviour change. A pure rename plus compatibility. rk's own "provider" naming
>   for agent CLIs (`cmd/rk/agent_registry.go`, `riff`) is rk's business and out of scope; hexokit
>   relabels only the fab-facing Agents tab (H5).
> - Do not release until H5 is released (rule 5). State the deprecation window in the release notes.

Binding decision row (plan § Decisions): *Config `providers:` → `clis:`, `agent.profiles.<role>.provider`
→ `.cli`, flag `--provider` → `--cli`, `fab agent -o yaml` key `provider:` → `cli:`. UI label "CLI".
Old spellings stay readable and settable for at least one minor release, with a deprecation warning.
Precedents: `session_command` → `interactive_command` and `agent.tiers` → `agent.profiles` used
per-field aliases plus a migration. Here `set`/`unset` must also accept the old key during the window
(unlike 2.19.0), because released hexokit writes it.*

Context the plan records that shapes the work: released hexokit (pre-H5) derives its tool list from
`providers.` rows of `fab config show --json` and writes `agent.profiles.<role>.provider` via
`fab config set`. rk's spawn path reads only `command` and `skill_prefix` from `fab agent -o yaml`,
never `provider:`. Plan rule 5: **N1 must not be released before hexokit H5 is released** — this
change may merge (cherry-pick-ladder merge mode) but its release is gated.

## Why

1. **The word is wrong.** fab's `providers:` entries are CLI coding tools (claude, codex, agy, kimi),
   not model providers. The tools themselves use "provider" for the model vendor — kimi's
   `[providers.openrouter]`, Claude Code routing to Fireworks — so fab's term collides with the
   tools' own vocabulary in every config file and conversation where both appear. Now that D1
   (`fab agent models`) reads the tools' own configs, the collision is in fab's own code too.
2. **If we don't fix it**, every new surface (D1's rows already say `cli`, hexokit's Agents tab, the
   F2 token `tool-default`) compounds the mismatch, and the rename only grows: today it is ~2,600
   Go occurrences, ~480 in `src/kit`, ~1,400 in docs.
3. **Why this approach** (rename + compatibility window, not a hard break): released hexokit writes
   the old keys and reads the old rows, and users' config files carry `providers:` blocks and
   `agent.profiles.<role>.provider` pins. The established precedents (2.19.0 `session_command` →
   `interactive_command`, 2.17.0 `agent.tiers` → `agent.profiles`) are read-time per-field aliases
   plus a hand-written migration. N1 follows them and additionally keeps `set`/`unset` accepting the
   old key (2.19.0 did not), because a released hexokit issues those writes.

## What Changes

This is a **pure rename plus compatibility** — resolution results, precedence, dispatch rungs, and
every behaviour stay identical. The new canonical vocabulary:

| Old | New |
|-----|-----|
| config `providers:` (top-level table) | `clis:` |
| `providers.<name>.{interactive_command,headless_command,native,models_command,profiles}` | `clis.<name>.{…same leaves…}` |
| `agent.profiles.<role>.provider` | `agent.profiles.<role>.cli` |
| env tier `FAB_PROVIDERS` | `FAB_CLIS` |
| `fab agent --provider <name>` | `fab agent --cli <name>` |
| `fab agent -o yaml` `provider:` / `source.provider:` | `cli:` / `source.cli:` |
| UI/prose "provider" (meaning a fab table entry) | "CLI" (label) / "CLI tool" (prose) |

### 1. Config schema and resolver (`internal/config`, `internal/agent`, `defaults.yaml`)

- `src/go/fab/defaults.yaml`: the built-in table moves from `providers:` to `clis:` (claude, codex,
  agy, kimi entries unchanged in content).
- `internal/config`: `Config` gains `CLIs map[string]CLIConfig \`yaml:"clis"\`` as the canonical
  field; the old `Providers` map stays as a **deprecated read-time alias** (`yaml:"providers"`),
  following the `SessionCommand`/`Tiers` comment precedent ("Deprecated: …, read-time alias only").
  `ProviderConfig` → `CLIConfig`. `RoleProfile` gains `CLI string \`yaml:"cli"\`` with
  `Provider string \`yaml:"provider"\`` kept as the deprecated alias.
- **Per-field merge, new wins.** Within one config layer, when the same `<name>` exists under both
  `clis:` and `providers:`, each leaf resolves from `clis.<name>.<leaf>` when non-empty and falls back
  to `providers.<name>.<leaf>` (and per role/field inside `profiles`). The same rule per role for
  `agent.profiles.<role>.cli` over `.provider` (and the already-deprecated `agent.tiers.<role>` entries
  read both `cli` and `provider` the same way). Across the cascade tiers (env > system > project >
  built-in) nothing changes: the normalized table merges exactly as `providers:` does today, so a
  half-migrated machine (system file on `providers:`, project on `clis:`) resolves identically to
  today. The normalization is done once at load/merge time so every consumer (`ResolveRole*`,
  `ProviderNames`→`CLINames`, dispatch, `fab agent models`, setup checks) sees only the new shape.
- `native` presence (`NativeSet`) must survive the alias merge — an explicit `providers.x.native:
  false` still disables a built-in `true`.
- **Env tier:** the derived env var for the top-level `clis` key is `FAB_CLIS`; `FAB_PROVIDERS` stays
  read as the deprecated alias (new wins when both are set). `EnvOrigins` reports whichever was used.

### 2. Registry (`internal/configref`) and `fab config upgrade` fence

- The registry row keyed `providers` becomes `clis` with `renamed_from: providers`, and the
  `agent.profiles` row's leaf vocabulary (`isProfileKey`-style matching at `configref.go:1319`:
  `contains([]string{"provider","model","effort"}, parts[3])`) accepts `cli` as canonical and
  `provider` as the deprecated spelling. The row today carries the informational
  `renamed_from: "providers.<name>.session_command, providers.<name>.dispatch_command"` — the
  apply agent decides how to keep that history (e.g. the field text, or a comment) given the row
  can carry one `RenamedFrom` string; the top-level `providers` → `clis` carry must be the
  machine-readable one, because `registryTopLevelKeys` will then carry a live top-level `providers:`
  key to `clis:` on `fab config upgrade` (value block verbatim; it already skips when `clis:` is
  also live).
- `roleProfileDefault` JSON (`configref.go:244`) `provider` → `cli`; `providerDefault`/
  `providerProfileDefault` → `cliDefault`/`cliProfileDefault`. `fab config explain --json` rows and
  `fab config show --json` therefore carry `clis.` keys for built-ins (hexokit H5 reads both).
- The advertised fence text moves to the new words: `# agent.session / agent.workers — CLI tools for
  the two agent DEPTHS …`, `fab config set --system agent.session <cli>`, and the examples
  `#   operator: { cli: codex }` / `#   review: { cli: codex }`. Because the fence content changes, the
  current providers/agent advert digests are added to `configupgrade.go`'s legacy-digest list so
  `fab config upgrade` re-renders the fence (the mechanism every prior fill bump used, e.g. F1).
- `fab config explain providers` / `explain clis`: the new key explains; the old key resolves to the
  same explanation with a one-line note that it is the deprecated spelling (keyed lookups of the
  old spelling must not fail during the window).

### 3. `fab config set` / `unset` accept the old key

- `fab config set agent.profiles.review.provider codex` → writes `agent.profiles.review.cli: codex`
  and warns on stderr: `agent.profiles.review.provider is deprecated, wrote agent.profiles.review.cli`.
  Same for any `providers.<name>.<leaf>` → `clis.<name>.<leaf>` set. The value gate
  (`validateSetValue` — known-CLI check on `agent.session`/`agent.workers`/`agent.profiles.<role>.cli`)
  runs on the mapped key. Exit 0 as for a normal set.
- When the target file still carries the old spelling at that same path, the write removes it so the
  file holds a single spelling (otherwise the new key wins anyway, but the stale old key would linger
  and confuse a later `unset`).
- `fab config unset agent.profiles.review.provider` (or `.cli`) removes **both** spellings at that
  path in the target file, warning `<old> is deprecated, …` when the old key was named. An unset of
  only the new key while an old-spelling key stays live would silently leave the override in effect.
- The `--system` path and the shadow/live-tier notices work on the mapped key unchanged.

### 4. `fab agent` flag and output

- `--cli <name>` replaces `--provider <name>` with identical semantics (with a selector: re-resolve
  from that CLI's fills; alone: address `clis.<name>` directly, bypassing role resolution).
  `--provider` stays registered as a **hidden** flag (`MarkHidden`) that behaves as `--cli` and
  prints `--provider is deprecated, use --cli` on stderr. Supplying both with different values is a
  usage error; the same value is accepted. Cobra's `Flag.Changed` semantics for the explicit-empty
  `--cli=` form are preserved (`ProviderSet` → `CLISet`).
- `--workers <cli>` help text and the `fab agent models [<cli>]` arg/Short text move to "CLI".
- `fab agent -o yaml` (`internal/agent/resolution.go`): `Resolution` gains `CLI string
  \`yaml:"cli"\`` in the slot `provider:` occupies today, immediately followed by a deprecated
  `Provider string \`yaml:"provider"\`` carrying the same value for the window. `Source` likewise
  gains `cli:` with `provider:` duplicated. Example:

  ```yaml
  selector: review
  kind: stage
  role: review
  cli: codex
  provider: codex        # deprecated duplicate of cli, removed after the window
  model: gpt-5.6-sol
  effort: xhigh
  command: …
  model_alias: ""
  template: …
  fill_mode: template
  source:
      cli: agent.profiles.review
      provider: agent.profiles.review
      model: clis.codex.profiles.review
      effort: clis.codex.profiles.review
  dispatch: …
  skill_prefix: …
  ```

  Source *values* that name config keys switch to the new spelling (`clis.codex.profiles.review`,
  `agent.profiles.review.cli`-style paths as the resolver reports them today, with `providers.` →
  `clis.`). The bare-flag form's `kind:` value changes `provider` → `cli` (a value cannot be
  dual-emitted).
- `fab resolve-agent` (deprecated line protocol, `Resolution.Lines()`): it emits a literal
  `provider=<name>` line today. The protocol is documented **byte-stable** and the command itself is
  deprecated, so its output is left unchanged (`provider=` stays, no `cli=` line added); only the Go
  field it reads is renamed. `_cli-fab.md` notes that the line keeps the old word.
- Every in-repo consumer that parses this output moves to `cli:` in the same change: the kit skills
  (`_preamble.md` § Per-Stage Model Resolution — "surface at minimum `provider`, `model`, …" →
  `cli`; the override table's "Cross-provider `--provider`" row → "Cross-CLI `--cli`" and
  `agent.profiles.<role>.provider` → `.cli`; `_pipeline.md`, `_cli-fab.md`, `_cli-fab-pane.md`,
  `_cli-agents.md`, `fab-setup.md`, `_srad.md` where they quote it), and Go callers
  (`dispatch_start.go`, `pane_open.go`, `batch*.go`, `resolve_agent.go`, `setupcheck`).

### 5. Go identifier rename

Mechanical rename wherever the identifier denotes a fab `clis:` entry, e.g.: `ProviderConfig` →
`CLIConfig`, `ResolveProvider` → `ResolveCLI`, `ProviderNames` → `CLINames`,
`unknownProviderError` → `unknownCLIError`, `providerFills` → `cliFills`, `Overrides.Provider`/
`ProviderSet` → `CLI`/`CLISet`, `ProviderOnly` → `CLIOnly`, local `provider` vars → `cli`, plus the
matching comments and error strings (`provider %q has no %s; configure providers.%s.%s` →
`CLI %q has no %s; configure clis.%s.%s`; the unknown-name lookup error lists "available CLIs").
Tests (`*_test.go`) are updated alongside, plus **new** tests for every compatibility path:
old-spelling config resolves identically; mixed spellings merge per field with new winning; env
`FAB_PROVIDERS` alias; `set`/`unset` with the old key (written key, warning text, both-spelling
removal); hidden `--provider` (works, warns, conflict error); `-o yaml` dual keys.

**Leave alone** where "provider" means a model vendor or something outside fab's table: e.g. kimi's
`[providers.<vendor>]` tables read by `agentmodels` (D1's kimi adapter), Claude Code's model
provider routing, rk's own `agent_registry.go`/`riff` naming (out of repo anyway), and the
already-deprecated `session_command`/`dispatch_command` and flat `model`/`effort` aliases keep
working (they now live under `clis.<name>` and `providers.<name>` alike).

### 6. Migration

New `src/kit/migrations/2.30.1-to-2.31.0.md` (minor bump: config-schema rename; the file carries the
standard *"The exact version range is a placeholder — the slot this change ships in is
release-owned and set at release time"* note, per `2.29.0-to-2.30.0.md`). Modeled on
`2.18.1-to-2.19.0.md`:

- Sweeps **both** `fab/project/config.yaml` and `~/.fab-kit/config.yaml`.
- Rewrites the live top-level `providers:` key to `clis:` (value block verbatim) — or, when both are
  live in one file, merges the old block's entries into `clis:` (new wins per leaf) and removes
  `providers:`; and rewrites every live `agent.profiles.<role>.provider:` to `cli:` (also under the
  legacy `agent.tiers.<role>` if present).
- Sentinel/idempotency: skip when neither file carries a live old key. Commented fence lines are
  never touched (`fab config upgrade` regenerates the fence).
- Notes that `FAB_PROVIDERS` in a user's shell environment should be renamed to `FAB_CLIS`
  (reported, not edited), and that the binary keeps reading old spellings for the window so nothing
  breaks before the migration runs.

`src/kit/VERSION` is **not** bumped (release-owned).

### 7. Skills, specs, memory, docs prose

- Kit skills (`src/kit/skills/`): `_cli-fab.md` (the `fab agent` family: `--cli`, hidden
  `--provider`, the `-o yaml` schema with `cli:` + deprecated `provider:`, `fab agent models
  [<cli>]`, `fab config set/unset` old-key behaviour), `_cli-fab-pane.md`, `_cli-agents.md`,
  `_preamble.md` (always-load bullet: "the `clis:` table (independent per-CLI
  `interactive_command`/`headless_command`/`native` …)", § Per-Stage Model Resolution),
  `_pipeline.md`, `_srad.md`, `fab-setup.md`. The "operator skills" the brief names were retired in
  261117-egu4 (fab operator moved to HexoKit) — there is no operator skill left to edit.
- Help text and `cmd/fab/skill.md`.
- Specs (`docs/specs/`): `stage-models.md`, `config.md`, `harness-adapters.md`, `architecture.md`,
  `templates.md`, `glossary.md`. `glossary.md` **adds a "CLI tool" entry** (an agent CLI fab can
  launch — claude, codex, agy, kimi, or a user-defined one — configured under `clis:`) and records
  the rename (the `providers` entry becomes `clis` with "`providers` is the pre-2.31 spelling, still
  read"), and rewords Role/Profile/Depth knob entries (`{cli, model, effort}`).
- Memory (`docs/memory/`): `runtime/providers-and-profiles.md` is renamed
  `runtime/clis-and-profiles.md` with links/indexes updated; `_shared/configuration.md`,
  `runtime/dispatch.md`, `runtime/agent-primitives.md`, `runtime/pane-commands.md`,
  `_shared/context-loading.md`, `distribution/kit-architecture.md`, `distribution/setup.md`,
  `distribution/migrations.md` (new migration entry), `pipeline/execution-skills.md`. Memory edits
  land at hydrate; apply touches only code, kit skills, specs and docs/site where needed.
- `docs/site/` (published docs), `README.md` if they mention `providers:`/`--provider`.
- **Historical text is not rewritten:** existing migrations (`2.12.1-to-2.13.0.md`,
  `2.16.19-to-2.17.0.md`, `2.18.1-to-2.19.0.md`, …), memory `log.md` entries, archived changes,
  plans, and changelogs describe the past and keep "provider".

### 8. Release gating (no code)

This change must not be released until hexokit H5 is released (plan rule 5). The PR body states this
and the deprecation window: old spellings (`providers:`, `.provider`, `FAB_PROVIDERS`, `--provider`,
the duplicate `provider:` output keys) are accepted for **one minor release** after the one that
ships N1 and may be removed no earlier than the following minor (user-confirmed). Removal is a separate future
change.

## Affected Memory

- `runtime/providers-and-profiles.md`: (remove) renamed to `runtime/clis-and-profiles.md`
- `runtime/clis-and-profiles.md`: (new) the renamed file — `clis:` table, `agent.profiles.<role>.cli`, deprecated-spelling aliases and the window
- `_shared/configuration.md`: (modify) `clis:` key, `FAB_CLIS`, `set`/`unset` old-key mapping, registry `renamed_from`
- `runtime/dispatch.md`: (modify) CLI vocabulary for dispatch rungs/capabilities
- `runtime/agent-primitives.md`: (modify) `fab agent --cli`, hidden `--provider`, `-o yaml` `cli:` + deprecated `provider:` keys, `kind: cli`
- `runtime/pane-commands.md`: (modify) vocabulary
- `_shared/context-loading.md`: (modify) always-load bullet wording for the `clis:` table
- `distribution/migrations.md`: (modify) the new 2.30.1-to-2.31.0 migration
- `distribution/setup.md`: (modify) setup wizard/check wording
- `distribution/kit-architecture.md`: (modify) vocabulary
- `pipeline/execution-skills.md`: (modify) vocabulary where it quotes resolution output

## Impact

- **Go** (`src/go/fab`): `internal/config`, `internal/configref` (+`defaultsmap.go`),
  `internal/configupgrade` (legacy digests, set/unset key mapping), `internal/configscope`,
  `internal/agent` (`agent.go`, `resolution.go`, `skill_prompt.go`), `internal/agentmodels`,
  `internal/dispatch`, `internal/pane`, `internal/spawn`, `internal/setupcheck`, `internal/shellparse`,
  `cmd/fab/{agent,agent_models,resolve_agent,resolution,dispatch_start,dispatch_open,pane_open,pane_process,pane_ready,batch*,config,config_validate,config_show_json,setup,setup_wizard}.go`,
  `defaults.yaml`, `defaults.go`, `cmd/fab/skill.md`, and their tests. ~2,600 textual occurrences
  of "provider" (most mechanical).
- **Kit**: `src/kit/skills/*` (~480 occurrences across `_cli-fab.md`, `_cli-agents.md`,
  `_cli-fab-pane.md`, `_preamble.md`, `_pipeline.md`, `_srad.md`, `fab-setup.md`), one new migration.
- **Docs**: `docs/specs/*` (~260), `docs/memory/*` (hydrate), `docs/site/*`.
- **External consumers**: hexokit Agents tab (handled by H5, release-gated), any user script reading
  `provider:` from `fab agent -o yaml` (dual-emit keeps it working through the window), user config
  files (read-time alias + migration + `fab config upgrade` top-level carry).
- **No behaviour change**: the same configs resolve to the same `{cli, model, effort, command,
  dispatch}`; an existing test asserting a resolved value must keep passing with only key/identifier
  spelling updated.

## Open Questions

- None. The plan's N1 open decision (deprecation-window length) was asked and resolved: one minor
  release — see Assumptions #4.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New spellings are `clis:`, `agent.profiles.<role>.cli`, `--cli`, output `cli:`/`source.cli`; label "CLI", prose "CLI tool" | Plan § Decisions + N1 Brief, user-decided 2026-11-16 | S:95 R:60 A:95 D:95 |
| 2 | Certain | Pure rename + compatibility: no resolution/precedence/dispatch behaviour changes | N1 Brief non-goals | S:95 R:70 A:90 D:95 |
| 3 | Certain | Resolver reads both spellings per field, new wins; old map/fields kept as deprecated read-time aliases normalized at load | N1 Brief; matches the 2.19 `session_command` and 2.17 `agent.tiers` precedents in `internal/config` | S:90 R:70 A:90 D:85 |
| 4 | Certain | Deprecation window: one minor release — old spellings accepted through the minor after the one that ships N1, removed no earlier than the next; stated in PR body/release notes; removal is a later change | Asked — user chose one minor release (plan's proposal) | S:95 R:85 A:90 D:90 |
| 5 | Confident | `fab config set` with an old key writes the new key, warns `<old> is deprecated, wrote <new>` on stderr, and removes the old key at that path in the same file | Brief specifies write+warning text; removing the stale twin avoids a lingering override | S:75 R:80 A:70 D:65 |
| 6 | Confident | `fab config unset` (old or new key) removes both spellings at that path in the target file | Otherwise unsetting `.cli` while `.provider` is live silently keeps the override; Brief requires unset to accept the old key | S:55 R:80 A:70 D:60 |
| 7 | Certain | Read-time use of old spellings is silent on ordinary reads (`fab agent`, dispatch); the deprecation warning fires on `set`/`unset`, the hidden `--provider` flag, `fab config validate`/setup checks, and the migration | Asked — user chose write/validate seams over every-config-load (skills parse `fab agent` output; 2.19 alias was silent at read time) | S:95 R:80 A:90 D:90 |
| 8 | Confident | `--provider` stays as a hidden flag behaving as `--cli`, warning on stderr; both given with different values is a usage error | Brief: "hidden deprecated alias"; conflict handling is the conventional choice | S:80 R:85 A:75 D:70 |
| 9 | Confident | `-o yaml` emits `cli:` in `provider:`'s current slot followed by a duplicate `provider:`; same for `source`; `source` values use the `clis.` spelling | Brief: emit both for the window; key ordering keeps the schema's documented order stable | S:80 R:75 A:75 D:70 |
| 10 | Confident | Bare-flag form's `kind:` value changes `provider` → `cli` with no duplicate | A scalar value cannot be dual-emitted; no Go code branches on it — the only in-repo mention is `_cli-fab.md`'s schema table, updated in the same change | S:40 R:75 A:65 D:55 |
| 19 | Confident | Deprecated `fab resolve-agent` keeps its byte-stable `provider=` line unchanged | The line protocol is documented byte-stable and the command is already deprecated; changing it would break the one property it exists for | S:45 R:80 A:70 D:60 |
| 11 | Confident | Env tier alias: `FAB_CLIS` canonical, `FAB_PROVIDERS` still read (new wins) | Env var name derives from the top-level key; keeping the old one is the same compatibility rule | S:50 R:80 A:75 D:70 |
| 12 | Confident | Registry: `clis` row carries `renamed_from: providers` so `fab config upgrade` carries the top-level key; the migration additionally handles nested `agent.profiles.<role>.provider` and the both-live merge | `registryTopLevelKeys` carries only top-level renames; Brief requires a migration that rewrites user files | S:75 R:75 A:80 D:70 |
| 13 | Confident | Migration file named `2.30.1-to-2.31.0.md` with the "placeholder range, release-owned" note; `src/kit/VERSION` not bumped | Current VERSION 2.30.1; precedent `2.29.0-to-2.30.0.md`; versions bumped by release commits | S:70 R:90 A:85 D:75 |
| 14 | Confident | Fence advert text changes, so current advert digests join `configupgrade.go`'s legacy list | Same mechanism F1 and every prior fill/advert change used | S:70 R:85 A:80 D:80 |
| 15 | Certain | Memory file `runtime/providers-and-profiles.md` renamed to `runtime/clis-and-profiles.md` (links updated at hydrate) | Asked — user chose to rename the file | S:95 R:80 A:90 D:90 |
| 16 | Certain | Historical text (old migrations, memory logs, archived changes, plans, changelogs) keeps "provider" | They record past behaviour; rewriting history would falsify it | S:70 R:90 A:90 D:85 |
| 17 | Certain | "Provider" meaning a model vendor (kimi `[providers.<vendor>]`, Claude Code routing) and rk's own naming stay untouched; operator skills no longer exist (retired 261117-egu4) | N1 Brief "where provider means a model vendor, leave it" + out-of-scope rk; `src/kit/skills/` has no operator skill | S:90 R:85 A:90 D:85 |
| 18 | Certain | This change merges but is not released before hexokit H5 is released | Plan rule 5 | S:95 R:60 A:95 D:95 |

19 assumptions (10 certain, 9 confident, 0 tentative, 0 unresolved).
