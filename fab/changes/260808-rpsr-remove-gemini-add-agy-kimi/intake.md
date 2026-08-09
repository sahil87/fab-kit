# Intake: Remove Gemini, Add Antigravity (agy) and Kimi Providers

**Change**: 260808-rpsr-remove-gemini-add-agy-kimi
**Created**: 2026-08-09

## Origin

> Remove support for gemini. And add support for antigravity cli - agy. And kimi k3. Also — note that earlier we used to get an error in gemini [skill-conflict warnings between `.agents/skills/` and `.gemini/skills/`]. Ensure that this doesn't occur for antigravity. (If agy supports the standard agent skill directories, don't duplicate.)

Conversational mode: the change was shaped in a `/fab-discuss` session that included live empirical verification of both new CLIs (installed locally: agy v1.1.11, kimi-code v0.34.0). Key user decisions during discussion:

1. **No compat migration** for existing gemini users — remove support outright ("Just remove support").
2. **agy fills omit `{effort}`** — agy's model IDs embed the effort level (`gemini-3.1-pro-high`), so the separate `--effort` flag is not used ("Agreed with omitting the effort flag given the model names contain the effort").
3. **kimi ships no per-role fills** — rely on the user's configured `default_model` ("for kimi rely on the user's default_model"). The empty-placeholder-drop machinery (`spawn.WithProfile` → `resolveTemplate`, `src/go/fab/internal/spawn/spawn.go`) was code-verified to drop `-m {model}` cleanly for both `session_command` and the `dispatch=` line (`resolve_agent.go:145` uses the same `WithProfile`).

## Why

1. **Pain point**: The gemini provider is no longer used, and its skill deployment produced recurring conflict warnings — the gemini CLI reads both the generic `.agents/skills/` directory (deployed for codex) and its own `.gemini/skills/` (deployed by the Gemini sync target), so every fab-synced skill appeared twice and gemini warned per skill (`⚠ Skill conflict detected: "_intake" from .agents/skills/... is overriding the same skill from .gemini/skills/...`).
2. **Consequence of not fixing**: dead provider config ships in every kit release with fills that rot at release cadence, and users driving multi-provider setups keep hitting the duplicate-skill noise.
3. **Approach**: replace gemini with the two providers actually wanted — Antigravity (`agy`) and Kimi (`kimi`, running Kimi K3) — and fix the conflict class structurally: **deploy no per-brand skill directory for either new provider**. Both new CLIs natively read the generic workspace `.agents/skills/` directory (verified: agy discovers `<workspace>/.agents/skills/<skill>/SKILL.md`; kimi's 4-tier discovery reads both the generic `.agents/skills/` group and the brand group `.kimi/`→`.claude/`→`.codex/`, merged by priority without conflict warnings — verified live in this repo with both `.claude/skills/` and `.agents/skills/` populated). One deployment target per skill set ⇒ duplication is impossible by construction.

## What Changes

### 1. Remove the gemini built-in provider

- `src/go/fab/internal/agent/defaults.yaml` — delete the `gemini:` provider block (commands + pro/flash fills).
- `src/go/fab/internal/agent/agent.go` — remove `providerGemini`, `DefaultGeminiSessionCommand`, `DefaultGeminiDispatchCommand`, and gemini comment references.
- `src/go/fab/internal/configref/configref.go` — remove the rendered `# gemini:` block from the `fab config explain` / managed-fence text; update every "THREE built-in providers" / "claude, codex and gemini" phrase (the roster becomes **four**: claude, codex, agy, kimi).
- `src/go/fab/cmd/fab/config.go` — replace the `fab config set agent.session gemini --system` example.
- Comment sweeps: `pane.go:220` (codex/copilot/gemini), `configupgrade.go` fence comments, `config.go` internals.
- **No migration**: existing user configs referencing gemini (e.g. `agent.workers: gemini`) will fail at resolve time with the normal unknown-provider error. This is accepted and deliberate (user decision). Historical artifacts — `src/kit/migrations/*.md`, `docs/memory/*/log.md`, `log.seed.md` — keep their gemini mentions verbatim (they are records, not present-truth claims).

### 2. Add the `agy` (Antigravity) built-in provider

Provider key `agy` — matching the binary name, the same convention as `claude`/`codex`. Verified grammar (agy v1.1.11):

```yaml
agy:
  # Antigravity CLI. Model IDs embed the reasoning effort (e.g. gemini-3.1-pro-high),
  # so there is NO {effort} placeholder — a separate --effort flag exists but would
  # fight the suffix. agy -p takes the prompt as an ARGUMENT and ignores stdin, and
  # POSIX expands $(cat) BEFORE applying the < redirect, so the dispatch command
  # nests a shell: the inner sh's stdin IS the redirected prompt file.
  # --print-timeout raised from its 5m default — stage workers run far longer.
  session_command: 'agy --dangerously-skip-permissions --model {model}'
  dispatch_command: 'sh -c ''agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'''
  profiles:
    default: { model: gemini-3.1-pro-high }
    fast:    { model: gemini-3.6-flash-low }
```

Empirical verification already performed (dispatch-shaped, `cmd < prompt.md > log 2>&1`):
- `sh -c 'agy --dangerously-skip-permissions --model gemini-3.6-flash-low -p "$(cat)"' < prompt.md` → exit 0, correct response captured in the redirected log. The historically-reported non-TTY stdout bugs (antigravity-cli #76/#318) do not reproduce on v1.1.11.
- `agy -p` without an argument errors (`flag needs an argument: -p`); stdin is never read as the prompt — hence the nested-shell `"$(cat)"` idiom (verified: `sh -c 'printf "nested:%s" "$(cat)"' < in.txt` receives the file; the un-nested form receives the outer stdin).
- Model IDs from `agy models`: `gemini-3.1-pro-{high,low}`, `gemini-3.6-flash-{high,medium,low}`, `gemini-3.5-flash-*`, plus claude/gpt-oss entries. Fills stay sparse (default + fast) per the codex/gemini precedent.

### 3. Add the `kimi` built-in provider

Provider key `kimi` (kimi-code CLI, Moonshot's Kimi K3 agent). Verified grammar (v0.34.0):

```yaml
kimi:
  # kimi-code CLI (Kimi K3). Ships NO fills: -m takes a USER-CONFIG model alias
  # (managed installs: kimi-code/k3; custom providers differ), so a pinned value
  # would break non-managed setups. The empty {model} drops the -m flag entirely
  # (WithProfile empty-value token-drop) and kimi falls back to the user's
  # default_model. -p is already non-interactive and auto-approves tools; it
  # REJECTS --yolo/--auto ("Cannot combine --prompt with --yolo"), so the
  # dispatch form carries no approval flag. Same nested-shell stdin idiom as agy.
  session_command: 'kimi --yolo -m {model}'
  dispatch_command: 'sh -c ''kimi -m {model} -p "$(cat)"'''
```

Empirical verification already performed:
- `sh -c 'kimi -p "$(cat)"' < prompt.md` → exit 0, correct response.
- `kimi -p` **executes tools unattended**: a dispatch-shaped run instructed to create a file did so without approval prompts (wrote `proof.txt` containing `VERIFIED`).
- `kimi --yolo -p ...` and `kimi --auto -p ...` both error (`Cannot combine`); `--yolo` remains valid for the interactive session command.
- With no fills, resolution yields empty model → `session_command` renders `kimi --yolo`, `dispatch=` renders `sh -c 'kimi -p "$(cat)"'` (token-drop removes `-m {model}` as a pair; interior-token drops keep the quoted segment syntactically intact).

### 4. Skill sync-target changes (`src/go/fab-kit/internal/skills.go`)

- **Remove** the Gemini deploy target (`{Label: "Gemini", CLI: "gemini", BaseDir: .gemini/skills, ...}`, line 37) — this kills the skill-conflict warning class at its root.
- **Widen the `.agents/skills` gate**: the generic-directory target (currently `{Label: "Codex", CLI: "codex", BaseDir: .agents/skills}`) deploys when **any** of `codex`, `agy`, `kimi` is on PATH. Requires the `agentConfig` CLI-probe to accept multiple candidate binaries (and `FAB_AGENTS` matching any of them); label should reflect the generic nature (e.g. `Agents dir (codex/agy/kimi)`).
- **No new per-brand targets** for agy or kimi — both read `.agents/skills/` natively (see Why). Existing `.gemini/skills/` dirs in user checkouts are left behind (gitignored, harmless, no migration per user decision).
- Update `skills.go` tests accordingly (check for existing test coverage of `deploySkills` / `agentAvailable`).

### 5. Scaffold gitignore fragment

`src/kit/scaffold/fragment-.gitignore`: remove `/.gemini`; add `/.kimi` (kimi's brand project directory, which kimi sessions may create). `/.agents` is already present.

### 6. Drift-guarded tests + docs/specs/skills sweep

The defaults.yaml header enumerates the pinned tests that will name every restating doc: `TestDefaultsFileIsWellFormed`, `TestDefaultRoleProfilesArePinned`, `TestNonClaudeProviderFillsArePinned` (gemini pins → agy pins; decide representation for kimi's empty fills), `TestDocTablesMatchAgentMaps`, `TestMirrorDocsMatchDefaultProfiles`, `TestCLIFabReferenceListsDefaultRoles`. Additional test files with gemini fixtures: `defaults_test.go`, `agent_test.go`, `defaultprofiles_mirrors_test.go`, `config_test.go`, `configupgrade_test.go`, `config_show_init_test.go`, `resolve_agent_test.go`, `agent_test.go` (cmd), `config_test.go` (cmd).

Sweep classes (per code-quality.md § Sibling & Mirror Sweeps — treat the whole class as in-scope):

- **Kit skills**: `_cli-agents.md` (built-in grammar/discovery dictionary), `_cli-fab.md` (§ fab resolve-agent enumeration, provider examples), `fab-operator.md` (launcher/provider mentions).
- **SPEC mirrors**: `SPEC-_cli-agents.md`, `SPEC-_cli-fab.md`, `SPEC-_preamble.md`, `SPEC-fab-operator.md`.
- **Specs**: `stage-models.md` (default-role tables + inline-YAML sample — both drift-guarded), `config.md`, `architecture.md`, `glossary.md`, `hooks.md`, `superpowers-comparison.md` (verify each hit; some are passing mentions).
- **Memory**: see Affected Memory.
- The managed fence in `fab/project/config.yaml` regenerates via `fab config upgrade` — its text source is `configref.go` (covered above); the "fab-kit ships claude, codex and gemini" line updates there.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) provider roster claude/codex/gemini → claude/codex/agy/kimi; fill semantics (agy suffixed-effort IDs, kimi deliberate no-fills/default_model inherit); command grammar entries
- `runtime/agent-primitives`: (modify) the `_cli-agents` grammar + discovery-recipe dictionary — replace the gemini entry with agy and kimi entries ("three built-ins" count language)
- `runtime/operator`: (modify) gemini mentions in launcher/provider examples
- `runtime/pane-commands`: (modify) passing gemini mention (verify at apply; may be example text)
- `runtime/runtime-agents`: (modify) passing gemini mention
- `distribution/setup`: (modify) skill sync-target list — `.gemini/skills` removed, `.agents/skills` multi-CLI gate documented
- `distribution/kit-architecture`: (modify) deploy-target enumeration
- `distribution/migrations`: (modify) verify at apply — if the gemini mention is a present-truth claim, update; if historical example, leave
- `_shared/configuration`: (modify) provider examples in the config-cascade docs
- `pipeline/hooks-may-enhance-never-own`: (modify) passing gemini mention

(Historical files `distribution/log.md` / `log.seed.md` and `src/kit/migrations/*.md` are records — not updated.)

## Impact

- **Go source**: `src/go/fab/internal/agent/` (defaults.yaml, agent.go + tests), `src/go/fab/internal/configref/`, `src/go/fab/internal/spawn/` (no logic change expected — quoted-template drop already works; add test coverage for the nested-shell/quoted-placeholder shape), `src/go/fab/cmd/fab/` (config.go examples + tests), `src/go/fab-kit/internal/skills.go` (+ tests).
- **Kit content**: `src/kit/skills/{_cli-agents,_cli-fab,fab-operator}.md`, `src/kit/scaffold/fragment-.gitignore`.
- **Docs**: 6 spec files + 4 SPEC mirrors + 10 memory files (per sweeps above).
- **Behavior**: pure provider-roster swap — the resolver, dispatch machinery, stage→role mapping, and depth knobs are untouched. No CLI command-signature changes (so `_cli-fab.md` updates are content/enumeration only). Constitution's CLI⇒docs+tests constraint still applies to the defaults change via the pinned tests.
- **Test strategy**: scope to `internal/agent`, `internal/configref`, `internal/spawn`, `cmd/fab`, and `fab-kit/internal` packages first; the mirror/doc drift-guard tests define completion for the doc sweep.

## Open Questions

*(none — all decision points were resolved in the discussion or graded below)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove gemini outright with no compat migration; existing gemini configs fail with the normal unknown-provider error | Discussed — user explicitly decided "Just remove support" | S:95 R:70 A:95 D:95 |
| 2 | Certain | agy fills use effort-suffixed model IDs (`gemini-3.1-pro-high`) and templates carry no `{effort}` placeholder | Discussed — user agreed; avoids the suffix fighting the separate `--effort` flag | S:90 R:85 A:85 D:90 |
| 3 | Certain | kimi ships no per-role fills; empty `{model}` drops `-m` and kimi uses the user's `default_model` | Discussed — user confirmed; `WithProfile` empty-value token-drop verified in `spawn.go` for both session and `dispatch=` paths | S:90 R:90 A:90 D:85 |
| 4 | Certain | Both new dispatch_commands use the nested-shell idiom `sh -c 'CLI ... -p "$(cat)"'` | Empirically verified end-to-end in dispatch shape (`cmd < prompt > log 2>&1`) for both CLIs; POSIX expansion-before-redirection makes the un-nested form fail | S:85 R:80 A:95 D:90 |
| 5 | Certain | No per-brand skill deploy targets for agy or kimi; `.agents/skills/` covers both and deploys when any of codex/agy/kimi is on PATH | User's explicit requirement; both CLIs verified to read the generic dir natively — duplication (the gemini warning class) impossible by construction | S:95 R:85 A:90 D:90 |
| 6 | Certain | Historical artifacts (`src/kit/migrations/*.md`, memory `log.md`/`log.seed.md`) keep gemini mentions verbatim | FKF present-truth convention: records are history, not current claims | S:80 R:90 A:90 D:85 |
| 7 | Confident | agy dispatch_command raises `--print-timeout` to `120m` (default 5m0s would kill long stage workers) | Flag documented in `agy --help`; 120m chosen to exceed any realistic stage duration — value easily tuned | S:60 R:90 A:80 D:70 |
| 8 | Confident | agy fills stay sparse: `default: gemini-3.1-pro-high`, `fast: gemini-3.6-flash-low` | Matches the codex/gemini sparse-fill precedent (default is the cross-role fallback); IDs taken from live `agy models` output | S:65 R:85 A:80 D:70 |
| 9 | Certain | Session commands: `agy --dangerously-skip-permissions --model {model}`; `kimi --yolo -m {model}` (kimi's `-p` rejects `--yolo`, so the dispatch form carries no approval flag — verified `-p` auto-approves tools) | Flags verified against installed CLIs; kimi tool-execution-in-`-p` verified by live file-write test | S:70 R:85 A:85 D:80 |
| 10 | Confident | Provider key is `agy` (binary name), not `antigravity` | All existing built-in keys equal their binary names (claude/codex/gemini); key is user-facing in `agent.workers:` and FAB_AGENTS-adjacent contexts | S:45 R:75 A:55 D:50 |
| 11 | Confident | Scaffold gitignore: drop `/.gemini`, add `/.kimi` | Fragment already ignores `/.agents`; kimi sessions may create the brand dir. Low-stakes, easily amended | S:55 R:90 A:80 D:75 |
| 12 | Confident | agy dispatch omits `--disable-slash-commands` (assume prompt-embedded `/fab-*` references don't trigger print-mode slash expansion mid-prompt) | Unverified but cheaply reversible; if stage prompts misbehave, adding the flag is a one-line fix — verify during apply if feasible | S:40 R:85 A:40 D:50 |

12 assumptions (7 certain, 5 confident, 0 tentative, 0 unresolved).
