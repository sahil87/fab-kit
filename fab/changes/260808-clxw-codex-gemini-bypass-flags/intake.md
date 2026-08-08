# Intake: Codex/Gemini Provider Bypass Flags

**Change**: 260808-clxw-codex-gemini-bypass-flags
**Created**: 2026-08-08

## Origin

Promptless dispatch (`{questioning-mode} = promptless-defer`) from a conversation-synthesized change description — no interactive questioning; any would-be-asked decision is recorded in `## Assumptions` as `Deferred — promptless dispatch`.

> Add approval/sandbox bypass flags to the built-in codex and gemini provider commands in `src/go/fab/internal/agent/defaults.yaml`.
>
> During a previous /fab-fff run with `agent.workers: codex`, the ship-stage worker hit an interactive approval wall while apply/review/hydrate workers ran fine. Those stages only read/write repo files, which codex's default sandbox permits; ship (role `fast`) needs `git push` + `gh pr create` — network access and out-of-workspace mutations — which codex's default approval/sandbox policy gates. review-pr (role `doing`) would hit the same wall on its `gh` calls.

Key decisions made in the originating conversation (encoded as assumptions below):

1. Fix in the built-in `defaults.yaml`, not per-project config — claude's `--dangerously-skip-permissions` establishes the policy that fab-dispatched workers ship full-auto.
2. Add codex's bypass flag to BOTH codex commands (`session_command` and `dispatch_command`). Likely `--dangerously-bypass-approvals-and-sandbox`; `--full-auto` was considered and rejected because its workspace-write sandbox still blocks the network access ship needs. The exact flag grammar MUST be verified against the installed codex CLI (`codex --help`) during apply — not assumed from memory.
3. Sibling fix for gemini (its `--approval-mode`/`--yolo` flag family; likewise verify exact grammar against the installed gemini CLI). Sibling-sweep discipline: fixing codex without gemini is the project's most common rework cause.
4. A user-side interim stopgap (overriding `providers.codex.dispatch_command` above the managed fence in `~/.fab-kit/config.yaml`) was noted but is NOT part of this change.

## Why

**Problem**: With `agent.workers: codex`, CLI-dispatched pipeline stages run codex under its default approval/sandbox policy. Read/write-inside-workspace stages (apply, review, hydrate) pass, but **ship** (role `fast`, runs `git push` + `gh pr create` — network access and out-of-workspace mutations) parks at an interactive approval prompt that a headless `codex exec` worker can never answer, and **review-pr** (role `doing`) hits the same wall on its `gh` calls. The dispatch reads `running` forever; the pipeline stalls until a human peeks and kills it.

**Consequence of not fixing**: `agent.workers: codex` is advertised as a one-line, fully-working configuration (`fab/project/config.yaml` fence prose, `docs/specs/stage-models.md`, `_cli-fab.md` all say naming a built-in on a depth knob "needs no other config"). That claim is false for any pipeline that reaches ship — every /fab-fff run under codex workers wedges at ship. Gemini has the identical latent defect (its `--approval-mode` default gates shell/network actions), it just hasn't been exercised yet.

**Why this approach**: The claude provider's shipped `session_command` (`claude --dangerously-skip-permissions ...`) already establishes the policy: fab-dispatched workers ship full-auto — fab's own dispatch design gives the orchestrator no keystroke channel to answer prompts (the pipeline's verb set is peek/kill/restart/notify/stop/reap; it NEVER sends keys), so an approval-gated worker is a design mismatch, not a configuration preference. Fixing the built-in table (embedded via go:embed in `internal/agent`) fixes every consumer at once; a per-project override would leave every other project broken and contradict the "one-line workers swap" promise. Per-user stopgaps remain available (any `providers.*` field is user-overridable) but are out of scope.

## What Changes

### 1. `src/go/fab/internal/agent/defaults.yaml` — codex commands gain the bypass flag

Current:

```yaml
  codex:
    session_command: 'codex -m {model} -c model_reasoning_effort={effort}'
    dispatch_command: 'codex exec -m {model} -c model_reasoning_effort={effort}'
```

Target shape (flag spelling to be verified at apply against the installed CLI — `codex --help` / `codex exec --help`):

```yaml
  codex:
    session_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
    dispatch_command: 'codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
```

- The flag goes on **both** commands (conversation decision 2). `--full-auto` is rejected: its workspace-write sandbox still blocks the network access ship needs.
- **Verify the exact flag grammar against the installed codex CLI during apply** — run `codex --help` and `codex exec --help`; do not assume the spelling from memory. `docs/memory/runtime/agent-primitives.md` already records the discovery recipe (`codex --version`, then `codex --help` / `codex exec --help`) as the project's pattern for exactly this.
- Flag placement must keep `{model}`/`{effort}` placeholders intact (spawn.WithProfile template mode substitutes every occurrence; empty value drops the placeholder token plus a preceding `-`-flag — placing the bypass flag before the placeholders keeps it clear of the token-drop rule).
- Update the adjacent YAML comments (the codex block comment explains `codex exec`/stdin; add why the bypass flag ships — parity with claude's `--dangerously-skip-permissions`, headless workers cannot answer approval prompts).

### 2. `src/go/fab/internal/agent/defaults.yaml` — gemini commands gain the sibling flag

Current:

```yaml
  gemini:
    session_command: 'gemini -m {model}'
    dispatch_command: 'gemini -m {model}'
```

Target shape (flag family `--approval-mode yolo` / `--yolo`; exact grammar to be verified at apply against the installed gemini CLI — `gemini --help`):

```yaml
  gemini:
    session_command: 'gemini --yolo -m {model}'
    dispatch_command: 'gemini --yolo -m {model}'
```

- Both gemini commands get the flag, mirroring the codex decision. <!-- assumed: gemini flag on BOTH commands — conversation named the sibling fix without enumerating commands; symmetry with codex decision 2 and claude's full-auto session command -->
- **Constraint from the pinned tests**: `TestResolveProvider_BuiltInCodexAndGemini` (agent_test.go:897) asserts `DefaultGeminiDispatchCommand` contains no `-p` substring, and `cmd/fab/config_test.go:353-356` forbids specific rendered substrings (`gemini -m {model} -c model_reasoning_effort`, `gemini -m {model} --effort`, `gemini -m {model} {effort}`, `gemini -m {model} -p`). Neither `--yolo` nor `--approval-mode yolo` trips these, but re-run them and re-check after the exact grammar is verified.
- Update the gemini block comment likewise.

### 3. Drift-guard / test sweep (same change, per constitution)

The defaults.yaml header enumerates its mirrors. Impact by mirror:

- `TestDefaultsFileIsWellFormed`, `TestDefaultsFileProviders`, `TestPackageTablesMatchDefaultsFile` (defaults_test.go) — derive from the parsed file and the exported vars (`DefaultCodexSessionCommand`, `DefaultCodexDispatchCommand`, `DefaultGeminiSessionCommand`, `DefaultGeminiDispatchCommand` — all read from the embedded file in agent.go, no literal copies), so they self-update; run them.
- `TestDefaultRoleProfilesArePinned` / `TestNonClaudeProviderFillsArePinned` (agent_test.go) — pin per-role FILLS (model/effort), not command strings; unaffected, run to confirm.
- `TestDocTablesMatchAgentMaps` + `TestMirrorDocsMatchDefaultProfiles` — machine-check role fills in `docs/specs/stage-models.md`, NOT the command strings; the command-string restatements in docs are swept **manually** (next section).
- `TestCLIFabReferenceListsDefaultRoles` — pins the role enumeration in `_cli-fab.md`, not commands; unaffected.
- `TestResolveProvider_BuiltInCodexAndGemini` (agent_test.go:865) — compares against the exported vars (self-updating) plus the gemini `-p`/`{effort}` negative guards noted above.
- Go test fixtures that restate the OLD default strings as **user-config fragments** (`internal/spawn/spawn_test.go`, `internal/config/config_test.go:213-241`, `cmd/fab/agent_test.go`, `cmd/fab/resolve_agent_test.go`, `cmd/fab/batch_*_test.go`) test parsing/substitution mechanics of arbitrary user strings, not the shipped defaults — no update required for correctness; leave them unless a test name claims to pin a default.
- Consider extending the pinned-command test surface: an assertion that both codex commands and both gemini commands carry the bypass flag (the same shape as the existing gemini no-`-p` guard) makes the policy regression-proof. Go changes ship test updates (constitution).
- Run scope: `go test ./internal/agent/... ./internal/configref/... ./internal/config/... ./cmd/fab/...` from `src/go/fab` (widen if failures point elsewhere).

### 4. `internal/configref/configref.go` — verify interpolation, update prose

`providersSegment` interpolates all four codex/gemini command strings from the canonical `agent.DefaultCodex*`/`DefaultGemini*` vars (configref.go:678-683) — **no literal copy, so the rendered config reference picks up the new strings by itself**. Two residuals:

- The per-provider notes prose (configref.go:662-672, the `#   codex —` / `#   gemini —` lines) explains each grammar; add the bypass-flag rationale there so the rendered reference explains the new flag.
- claude's commented `dispatch_command` example (configref.go:676) is a literal but is NOT part of this change (claude unchanged).

### 5. Documentation sweep (manual — not machine-guarded for command strings)

Every restatement of the codex/gemini command strings, swept in the same change:

- `docs/specs/stage-models.md` — inline-YAML defaults sample (lines ~221-239) and the config-reference excerpt (lines ~380-394).
- `docs/specs/architecture.md` — providers block sample (lines ~254-265).
- `docs/specs/config.md` — mentions the providers table; no command-string literals found (lines 85, 102, 276-277, 354 are prose/fills) — check, expect no edit.
- `src/kit/skills/_cli-fab.md` — the `dispatch=` example lines (338, 355, 363: `dispatch=codex exec -m <codex-model-id> -c model_reasoning_effort=...`) and the § providers prose (line 411) describing the codex/gemini grammars. Per constitution, the skill edit requires the `docs/specs/skills/SPEC-_cli-fab.md` mirror update in the same change; on a CLI-surface change treat the whole SPEC mirror class as the sweep class.
- Memory files: see Affected Memory below (hydrate owns them, listed for the sweep class).
- Grep discipline: sweep with `grep -rn "codex exec -m\|codex -m {model}\|gemini -m {model}"` repo-wide before finishing apply; behavior-claim sweeps must include user-facing string literals.

### Out of scope

- claude's commands (already full-auto).
- The user-side stopgap override in `~/.fab-kit/config.yaml` (conversation decision 4).
- No migration: `defaults.yaml` is binary-embedded defaults-layer data, not user data — user configs are untouched and any existing user override of `providers.codex.*`/`providers.gemini.*` continues to win over the new defaults via the per-field cascade merge.

## Affected Memory

- `runtime/providers-and-profiles`: (modify) provider table (lines 80-81) and GIVEN/THEN scenarios (lines 116, 122, 237, 301-304) restate the codex/gemini command grammars; § grammar-specifics prose (lines 99-101) explains each flag choice — add the bypass-flag rationale
- `_shared/configuration`: (modify) § providers — template-mode example (`'codex exec -m {model} -c model_reasoning_effort={effort}'`, line 165) and the gemini no-`-p` note (line 181)
- `runtime/agent-primitives`: (modify) GIVEN scenario (line 51) uses the codex `session_command` literal

## Impact

- **Code**: `src/go/fab/internal/agent/defaults.yaml` (4 command strings + comments); possible new pinned-flag assertion in `src/go/fab/internal/agent/agent_test.go` or `defaults_test.go`; prose-only touch in `src/go/fab/internal/configref/configref.go` (per-provider notes).
- **Docs**: `docs/specs/stage-models.md`, `docs/specs/architecture.md`, `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-_cli-fab.md`; `docs/specs/config.md` checked (expect no edit). Memory files per Affected Memory (hydrate).
- **Tests**: `go test ./...` scoped to `internal/agent`, `internal/configref`, `internal/config`, `cmd/fab` first.
- **Behavioral**: any fab install with `agent.workers: codex` (or gemini) starts dispatching fully-bypassed workers by default — the intended policy (claude parity). Users who want approval-gated workers override the provider command in config (the same per-field override surface that exists today).
- **Security posture**: deliberate — fab-dispatched workers are autonomous by design (the pipeline cannot answer prompts); this change makes codex/gemini match the shipped claude posture rather than introducing a new one.
- **No migration, no CLI-signature change, no schema change.**

## Open Questions

- None blocking. The two verify-at-apply items (exact codex and gemini flag grammar against the installed CLIs) are execution steps with a decided procedure, not open decisions.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Fix lands in the built-in `defaults.yaml` (embedded provider table), not per-project config | Discussed — conversation decision 1; claude's `--dangerously-skip-permissions` establishes the full-auto policy for fab-dispatched workers | S:90 R:75 A:90 D:90 |
| 2 | Certain | Codex bypass flag goes on BOTH `session_command` and `dispatch_command` | Discussed — conversation decision 2, explicit | S:90 R:85 A:85 D:85 |
| 3 | Confident | Codex flag is `--dangerously-bypass-approvals-and-sandbox` (not `--full-auto`), with exact grammar verified against the installed CLI (`codex --help`) at apply | Discussed — `--full-auto` rejected (workspace-write sandbox still blocks ship's network access); verification procedure decided in conversation, matches the agent-primitives discovery-recipe pattern | S:80 R:80 A:70 D:75 |
| 4 | Confident | Gemini sibling fix uses its `--approval-mode`/`--yolo` flag family, grammar likewise verified against the installed gemini CLI at apply | Discussed — conversation decision 3; exact spelling deferred to the same verify-at-apply procedure | S:75 R:80 A:65 D:70 |
| 5 | Confident | Gemini flag applies to both gemini commands (session + dispatch) | Inferred — conversation named the sibling fix without enumerating commands; symmetry with decision 2 and with claude's full-auto session command; trivially reversible | S:55 R:85 A:80 D:75 |
| 6 | Certain | User-side stopgap (`~/.fab-kit/config.yaml` override) is out of scope | Discussed — conversation decision 4, explicit exclusion | S:95 R:90 A:95 D:95 |
| 7 | Confident | No migration ships: defaults.yaml is binary-embedded defaults-layer data, not user data; existing user provider overrides keep winning via the cascade | Inferred — context.md § Migrations covers user-data restructuring only; the three-layer cascade (project > system > defaults) makes the new default a pure fallback | S:60 R:75 A:85 D:80 |
| 8 | Confident | Existing Go test fixtures restating the old strings as user-config fragments need no update; only default-pinning tests and the enumerated mirrors are in the sweep class | Inferred — those fixtures test parse/substitution mechanics of arbitrary user strings (verified by reading spawn_test.go, config_test.go, resolve_agent_test.go call sites); defaults derive from exported vars with no literal copies | S:55 R:80 A:80 D:70 |
| 9 | Certain | Doc/SPEC/test sweep (stage-models.md, architecture.md, _cli-fab.md + SPEC-_cli-fab.md, memory files) ships in the same change | Constitution Additional Constraints + code-quality.md § Sibling & Mirror Sweeps mandate it | S:85 R:80 A:95 D:90 |

9 assumptions (4 certain, 5 confident, 0 tentative, 0 unresolved).
