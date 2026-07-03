# Intake: Templated Claude session_command

**Change**: 260703-gvxd-templated-claude-session-command
**Created**: 2026-07-03

## Origin

Created by the `_intake` Create-Intake Procedure in `promptless-defer` mode, dispatched by `/fab-proceed`. The design was synthesized from a `/fab-discuss` conversation on 2026-07-03; all decisions below carry that conversation's rationale verbatim.

> Make the claude provider's `session_command` use explicit `{model}`/`{effort}` template placeholders instead of relying on `spawn.WithProfile`'s implicit append mode — in both the scaffold and the built-in Go constant, keeping resolved output byte-identical.

## Why

1. **Explicitness over magic.** Today the live claude `session_command` in the scaffold is a plain command; the tier profile (`--model`/`--effort`) is appended invisibly by `spawn.WithProfile`'s append mode, documented only in a comment. A user reading the scaffold cannot see where the tier profile lands. With explicit `{model}`/`{effort}` placeholders, the substitution point is visible in the config itself.
2. **Internal consistency.** Claude's own commented `dispatch_command` one line below is already templated (`claude -p --dangerously-skip-permissions --model {model} --effort {effort}`), as are the codex/gemini starter blocks. The live claude `session_command` is the **only** non-templated command in the file.
3. **If we don't fix it**: the inconsistency persists in every new project scaffolded by `fab init`, and `fab config reference` keeps teaching the append-mode-magic form as the canonical example while every other command in the same block demonstrates templates.
4. **Why this approach**: template mode already exists and is proven (shipped by `260702-6tmi-spawn-command-placeholders`, PR #456). Placing the placeholders at the END of the command makes template substitution produce **byte-identical** resolved commands to today's append mode — zero behavior change, verified empirically (see Impact).
5. **Empty-value semantics are equivalent**: template mode drops the flag token + placeholder token when a value is empty, matching append mode's omit rule (see `WithProfile` docs in `src/go/fab/internal/spawn/spawn.go`).

## What Changes

### 1. Scaffold line (`src/kit/scaffold/fab/project/config.yaml:66`)

Change:

```yaml
session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")"'
```

to:

```yaml
session_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
```

Placeholders go at the END deliberately: template substitution then produces byte-identical resolved commands to today's append mode. Keep the single-quoted YAML string style.

Also update the surrounding comment block (lines 40–63): the mechanism sentence "`{model}`/`{effort}` placeholders are substituted from the resolved tier profile; a plain Claude command has --model/--effort appended instead" **stays true** (append mode remains for plain commands), but any wording that presents the live claude line as the plain/append-mode example must be updated — after this change the claude line is a template like every other command in the block. Watch the "claude is the built-in default (session_command shown live)" sentence: "live" (uncommented) remains correct, but re-read the paragraph for implications that claude demonstrates append mode.

### 2. Built-in Go constant (`src/go/fab/internal/agent/agent.go:49`)

`agent.DefaultSessionCommand` gets the same templated form:

```go
const DefaultSessionCommand = `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}`
```

Update its doc comment (lines 45–48) as needed. Reason this is included (a scaffold-only change was rejected): `fab config reference` renders its providers block from this constant — `src/go/fab/internal/configref/configref.go:92` sets `SessionCommand: agent.DefaultSessionCommand`, rendered into the template at `session_command: '{{ .SessionCommand }}'` (~line 183). A scaffold-only change would make `fab init` output contradict `fab config reference` output — **live drift, not just doc drift**.

`spawn.DefaultSpawnCommand` (`src/go/fab/internal/spawn/spawn.go:13`) aliases this constant and is the fallback when a provider has no `session_command`; template mode resolves it identically, so the fallback stays correct with no code change in spawn.go beyond doc-comment accuracy checks.

Check `configref.go` surrounding prose/comments (~lines 150–190) describing the claude block for wording that assumes the plain form.

### 3. Explicit non-changes

- **`spawn.WithProfile` append mode stays untouched.** It is load-bearing for existing user configs: the 2.12.1→2.13.0 migration moved old `agent.spawn_command` values verbatim into `providers.claude.session_command`, and some of those carry user-pinned `--model`/`--effort` flags that rely on append-last/last-wins.
- **No migration ships with this change.** Existing plain-form configs keep working via append mode; this only changes what new projects get from `fab init` and what `fab config reference` shows. Shipped migrations (including the recent #468 backfill, `260702-fyn5`, which writes the plain form) stay frozen — migrations are frozen historical artifacts; the form drift is cosmetic since both forms resolve identically.
- **This repo's own `fab/project/config.yaml:58`** (plain form, unquoted style) stays untouched — it is an existing user config covered by append mode, consistent with the no-migration decision.

### 4. Sweep class (documentation_accuracy / cross_references — this repo's #1 rework cause; in-scope up front)

Repo-wide grep for the literal `claude --dangerously-skip-permissions -n` (already performed at intake; re-run at apply) — classify every occurrence as **update** (describes the scaffold/default going forward: templated form, or reworded mechanism prose) vs **keep** (historical/migration text, append-mode compat documentation):

**Go source + tests (update):**

- `src/go/fab/internal/agent/agent.go:49` — the constant + doc comment (§2 above).
- Tests asserting the raw constant string or resolved commands byte-exactly. Grep found the literal in: `internal/dispatch/dispatch_test.go`, `internal/config/config_test.go`, `internal/spawn/spawn_test.go`, `cmd/fab/batch_switch_test.go`, `cmd/fab/batch_new_test.go`, `cmd/fab/pane_process_test.go`, `cmd/fab/resolve_agent_test.go`, `cmd/fab/config_test.go`, `cmd/fab/agent_test.go`. Tests asserting the **raw constant** need the new templated form; tests asserting **resolved output for default tiers** should pass unchanged (byte-identical resolution is the acceptance check — a resolved-output test that breaks indicates a real regression, not a fixture to bend; Constitution VII).

**Kit skills (update, canonical sources under `src/kit/` only — never `.claude/skills/`):**

- `src/kit/skills/_cli-fab.md` — passages presenting claude as the append-mode case: §fab config reference (~line 317, "`session_command` live" / "codex/gemini remain template text only"), §fab operator (~line 828 — "for a plain Claude `session_command` (no placeholder) it **appends**…" mechanism prose stays true, but the built-in default now takes the substitution path, so composition descriptions of a fully-defaulted launch change), §fab agent (~line 868), §fab batch (~lines 882–883).
- `src/kit/skills/fab-operator.md` — lines ~452 and ~702 quote the plain default command verbatim.
- `src/kit/skills/_preamble.md` — the `WithProfile` grammar-forgiving passage (~line 325): mechanism description stays true; update the example framing ("appends `--model <full-id> --effort <level>` as before" — the built-in default is no longer the plain/append example).

**SPEC mirrors (update — treat the whole mirror class as in-scope per code-quality.md § Sibling & Mirror Sweeps, not just literal-carrying files):**

- `docs/specs/skills/SPEC-fab-operator.md` — lines 36 and 164 quote the plain default verbatim.
- `docs/specs/skills/SPEC-_cli-fab.md` — line 39 mechanism summary ("substituted/appended"); sweep whole file.
- `docs/specs/skills/SPEC-_preamble.md` — no literal hit found at intake, but in the mirror class if `_preamble.md` is touched.

**Other specs (update):**

- `docs/specs/stage-models.md:149` — quotes the plain default in the providers table; also §Skill wiring append-mode prose (~lines 319–320).
- `docs/specs/architecture.md:233` — quotes the plain default.

**Memory files (update — see Affected Memory):**

- `docs/memory/runtime/providers-and-tiers.md:27`, `docs/memory/_shared/configuration.md:59,65`, `docs/memory/runtime/operator.md:291`, `docs/memory/distribution/kit-architecture.md:122,327`. (`docs/memory/runtime/dispatch.md` grepped clean for the literal at intake — verify at apply, likely no edit.)

**Keep verbatim (historical/compat):**

- `src/kit/migrations/2.12.1-to-2.13.0.md:88` — frozen migration text (writes what 2.13.0 shipped).
- Completed change artifacts under `fab/changes/*/` (`260702-tykw`, `260702-ho9y` intake/plan files) — historical records.
- `fab/project/config.yaml:58` — this repo's live config (§3 above).

## Affected Memory

- `runtime/providers-and-tiers`: (modify) config excerpt at line 27 quotes the plain claude `session_command` — update to the templated form
- `_shared/configuration`: (modify) lines 59 and 65 quote the plain default as the built-in `session_command` value — update value + re-check surrounding append-mode prose
- `runtime/operator`: (modify) line 291 quotes the `spawn.DefaultSpawnCommand` fallback value — the constant changes, so the quoted fallback changes
- `distribution/kit-architecture`: (modify) lines 122 and 327 quote the built-in claude fallback constant — update both

## Impact

- **Code**: `src/go/fab/internal/agent/agent.go` (1 constant + comment); comment-accuracy checks in `src/go/fab/internal/configref/configref.go` and `src/go/fab/internal/spawn/spawn.go`. No logic changes.
- **Kit content**: `src/kit/scaffold/fab/project/config.yaml` (line + comment block), `src/kit/skills/{_cli-fab,fab-operator,_preamble}.md`.
- **Docs**: 3 SPEC mirrors, 2 specs, 4 memory files (lists above).
- **Tests**: update raw-constant assertions across the 9 test files listed in What Changes §4; affected Go packages: `internal/agent`, `internal/spawn`, `internal/configref`, `internal/config`, `internal/dispatch`, `cmd/fab`.
- **Acceptance criteria** (byte-identical resolution, captured pre-change on the default config):
  - `fab agent default --print` → `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5 --effort xhigh`
  - `fab agent operator --print` → `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-sonnet-5 --effort medium`
  - Both must be byte-identical after the change; `go test` green on the affected packages.
- **Constraints** (constitution): CLI/Go changes MUST update `src/kit/skills/_cli-fab.md` + ship tests; skill-file changes MUST update their `docs/specs/skills/SPEC-*.md` mirrors; edit canonical sources under `src/kit/` only, never `.claude/skills/`.
- **No user-data restructuring** → no migration required (and none ships, by design — see What Changes §3).

## Open Questions

- None — every decision point was resolved in the `/fab-discuss` conversation or graded Certain/Confident below (no Unresolved rows to defer).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scaffold `session_command` gains ` --model {model} --effort {effort}` at the END (single-quoted YAML style kept), so template substitution yields byte-identical resolved output to today's append mode | Discussed — verified empirically: `fab agent default --print` currently yields `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5 --effort xhigh` | S:90 R:85 A:95 D:95 |
| 2 | Certain | `agent.DefaultSessionCommand` (agent.go:49) gets the same templated form — scaffold-only rejected because `fab config reference` renders from the constant (configref.go:92), which would create live drift vs `fab init` | Discussed — live-drift rationale; `spawn.DefaultSpawnCommand` aliases the constant and template mode resolves it identically, so the no-session_command fallback stays correct | S:95 R:80 A:90 D:90 |
| 3 | Certain | `spawn.WithProfile` append mode stays untouched | Discussed — load-bearing for existing user configs: the 2.12.1→2.13.0 migration moved `agent.spawn_command` values verbatim, some carrying user-pinned `--model`/`--effort` flags relying on append-last/last-wins | S:90 R:70 A:90 D:90 |
| 4 | Certain | No migration ships; shipped migrations (incl. the #468 `260702-fyn5` backfill writing the plain form) stay frozen | Discussed — migrations are frozen historical artifacts; existing plain-form configs keep working via append mode; form drift is cosmetic since both forms resolve identically | S:90 R:75 A:85 D:85 |
| 5 | Confident | This repo's own `fab/project/config.yaml:58` (plain form) stays untouched | Follows the no-migration decision — it is an existing user config covered by append mode; not explicitly discussed, trivially reversible if the user prefers dogfooding the new form | S:60 R:90 A:75 D:70 |
| 6 | Confident | Grep-sweep classification rule: live docs/specs/skills/memory occurrences → templated form or reworded mechanism prose; historical artifacts (shipped migrations, completed change folders, this repo's config) → keep verbatim | Discussed as a rule ("classify each as update vs keep"); per-occurrence judgment applied and recorded at apply | S:75 R:85 A:80 D:70 |
| 7 | Certain | `docs/memory/distribution/kit-architecture.md` (lines 122, 327) joins the sweep class — it quotes the fallback constant but was absent from the discussed file list | Mechanical application of the discussed repo-wide grep instruction; found at intake verification | S:80 R:90 A:90 D:85 |
| 8 | Confident | change_type = `refactor` (overriding inferred `feat`) — form restructuring with byte-identical resolved behavior, no new capability | change-types.md: "Code restructuring without behavior change"; the capability (template mode) shipped in 260702-6tmi | S:65 R:90 A:80 D:70 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
