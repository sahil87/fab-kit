# Intake: Autopilot Merge-Mode Config Default

**Change**: 260821-r5t5-autopilot-merge-mode-config-default
**Created**: 2026-08-21

## Origin

Dispatched by `/fab-proceed` (promptless create-intake) from a user conversation that produced a synthesized change description. Interaction mode: conversational design discussion preceding the dispatch — the decisions below were user-agreed there.

> **Problem.** `fab operator autopilot` supports three merge modes (`cherry-pick-ladder`, `merge-auto`, `stacked-prs` — added by merged change t6rq, PR #608; documented in `docs/site/merge-topologies.md` and `src/kit/skills/fab-operator.md` §6 Autopilot). The binary hardcodes the default `cherry-pick-ladder` (`src/go/fab/cmd/fab/operator_autopilot.go`, `validAutopilotModes`), and the skill documents it as default — but nothing instructs the operator agent to use the default *silently* when the user's queue request names no mode. The skill presents mode as "selected at queue start via `--mode` or natural language", and merging is confirm-before-executing (destructive tier, §3), so in practice the LLM operator stalls mid-queue-start and asks the user "which merge topology?" — often half an hour after the user handed off work, defeating unattended operation.

## Why

1. **The pain point**: the operator agent treats merge-mode selection as an open question rather than a defaulted preference. When a user hands off a queue without naming a mode, the operator pauses at queue start to ask "which merge topology?". Because autopilot is precisely the unattended-operation feature, that question typically lands ~30 minutes after the user walked away — the queue sits idle until they return. The binary *has* a default (`validAutopilotModes[0]` = `cherry-pick-ladder`, both as the `--mode` flag default and the back-compat fill for old state files), but no skill rule tells the operator to use it silently, and no config key lets a user set a different standing preference.

2. **The consequence if unfixed**: every promptless queue handoff risks a dead stall at the first confirmation. Users who prefer a non-default mode (e.g. `stacked-prs`) must remember to name it in every single queue request — there is no way to say it once.

3. **Why this approach**: a config registry key (`autopilot.merge_mode`) matches the existing preference-knob pattern exactly — `dispatch.mode` is the direct precedent (both-scope, enum-valued, machine-wide settable, outranked by an explicit flag). A skill rule making silent resolution the default behavior — with the resolved mode stated inside the *existing* upfront queue-confirmation line — preserves the user's veto without adding a round-trip. The only remaining questions are the two enumerable misfits where proceeding silently would be wrong (see What Changes §3).

**Alternatives rejected** (user-agreed during discussion):
- **`stacked-prs` as shipped default** — rejected: operationally the most fragile mode for unattended operation (merge-all requires per-PR base retarget + rebase; dependency-branch drift exposure). `cherry-pick-ladder` keeps every PR independently mergeable against main and tolerates out-of-order merges (empirically proven when #609 merged out of order into the t6rq stack with no loss). Users preferring `stacked-prs` set it with one system-level config line — that is the purpose of the key.
- **Reusing the bare word `mode` for the key** — rejected: collides conceptually with the existing `dispatch.mode`; `autopilot.merge_mode` is unambiguous.

## What Changes

### 1. New config registry key `autopilot.merge_mode`

Add a field to the `internal/configref` registry and the `internal/config` `Config` struct:

| Property | Value |
|----------|-------|
| Key | `autopilot.merge_mode` |
| Default | `cherry-pick-ladder` — sourced from the canonical Go symbol, never a second literal (see Single-sourcing below) |
| Kind | string |
| Allowed values | exactly `validAutopilotModes`: `cherry-pick-ladder` \| `merge-auto` \| `stacked-prs` |
| Scope | `both` — settable once machine-wide via `fab config set --system autopilot.merge_mode <name>`; the system tier outranks the project file per the existing four-tier cascade (environment > system > project > built-in defaults) |
| Advertise | `true` — rendered commented in every project's managed fence, following the `dispatch.mode` precedent (preference-class both-scope knob); see Assumption 9 |
| Segment | a new commented `# autopilot:\n#   merge_mode: cherry-pick-ladder` YAML block (new top-level `autopilot:` parent — no existing row owns that block, so this row carries its own `Segment`/`ShortSegment`, unlike `dispatch.column_width` which renders inside `dispatch.mode`'s) |

**Single-sourcing**: `internal/configref` rows source defaults from canonical Go symbols (`config.DefaultDispatchMode` pattern — see the comment block at `src/go/fab/internal/config/config.go:255-286`). Today the valid-modes list `validAutopilotModes` lives in `package main` (`src/go/fab/cmd/fab/operator_autopilot.go:13`), which `internal/configref` cannot import. The canonical list/default therefore moves to (or is created in) `internal/config` — e.g. an exported valid-modes slice plus `DefaultAutopilotMergeMode` referencing its first entry — and `cmd/fab`'s `validAutopilotModes` and the configref row both reference that single source. No second literal anywhere.

**Config struct + accessor**: add an `Autopilot` section to the `Config` struct (`yaml:"autopilot"`) with a `MergeMode` field, plus a nil-safe accessor following the existing `GetDispatchMode` shape. **Deliberate divergence from `GetDispatchMode`'s fail-open-with-warning posture**: an *invalid* configured value must surface as an actionable error at `fab operator autopilot start` naming the valid set (user-agreed) — merging is destructive-tier behavior, so silently falling back to a different merge topology than the one the user configured is the wrong failure mode. (Where exactly the raw-value-vs-error split sits — accessor returns raw value and `start` validates, or accessor returns `(value, error)` — is a plan-level choice; the observable contract is: absent key → default; invalid key → `start` errors naming `cherry-pick-ladder, merge-auto, stacked-prs`.)

### 2. `fab operator autopilot start` resolves mode from config

Current behavior (`src/go/fab/cmd/fab/operator_autopilot.go`): the `--mode` flag carries default `validAutopilotModes[0]`, so an absent flag is indistinguishable from an explicit `--mode cherry-pick-ladder`, and the hardcoded literal is the only default source. `start` prints nothing on success.

New behavior:

- When `--mode` is **absent** (detected via `cmd.Flags().Changed("mode")` — the flag keeps its current default for help-text display), resolve the mode from config via the standard config load/cascade (`autopilot.merge_mode`), falling back to the built-in default when the key is unset.
- When `--mode` is **present**, the flag wins unchanged.
- **Resolution ladder** (binary and skill consistent): explicit user instruction or `--mode` flag > config (`autopilot.merge_mode`) > built-in default `cherry-pick-ladder`.
- **Print the resolved mode and its source** on success — e.g. `mode: cherry-pick-ladder (flag)` / `(config)` / `(default)`, following the style of `fab dispatch`'s `mode: <rung> (preferred)` output (see Assumption 13).
- **Validation unchanged**: the resolved mode (from any source) validates against `validAutopilotModes`; an invalid *config* value errors actionably naming the valid set, exactly like an invalid flag value does today (`unknown --mode %q (valid: ...)` — the config-sourced error message should name the config key rather than the flag).
- **`loadAutopilot`'s back-compat fill is untouched**: an `autopilot` state block lacking `mode` (pre-existing state file) continues to read as the *built-in* first-entry default — NOT config-resolved. The mode was fixed at queue start and persisted; re-resolving from config at read time could silently change a running queue's topology.

### 3. Skill rule: silent resolution + enumerated pause-and-ask misfits (`src/kit/skills/fab-operator.md` §6 Autopilot)

- **Silent default rule**: when the user's queue request names no mode, the operator resolves it via the ladder (explicit instruction > config `autopilot.merge_mode` > built-in `cherry-pick-ladder`) and proceeds WITHOUT asking. The resolved mode is stated inside the **existing** upfront queue-confirmation line (the §6 "Confirm upfront (...)" prompt, which already varies by mode) — the user can veto in the same breath, so no extra round-trip is added.
- **Pause-and-ask ONLY on an enumerable misfit** — exactly two cases:
  1. The resolved mode is `merge-auto` but the queue has same-repo `depends_on` entries — implicit chaining is disabled in that mode, so the declared dependency semantics contradict the mode.
  2. The user's own message conflicts with the resolved mode (e.g. says "merge as you go" while the resolved mode is a held mode like `cherry-pick-ladder` or `stacked-prs`).
- **Mode-question format** (when the operator DOES ask): the question must include the at-a-glance shorthand glyphs (`▂▄▆` cherry-pick-ladder · `░▒▓█` merge-auto · `▄▀` stacked-prs), the three compact box diagrams, and a one-line tradeoff per mode.
- **Clarify the existing diagram caveat**: the current §6 parenthetical ("never emit them into the status frame, which stays fence-free per §4") is scoped to the **status frame only**. A mode question is an ordinary conversational message where fenced diagrams render fine — the clarified wording must make that explicit so the operator doesn't over-apply the rule.

### 4. Documentation + sibling sweep

Constitution-mandated and known-sibling updates:

- `src/kit/skills/_cli-fab.md` § fab operator autopilot (~lines 1269-1284): `--mode` default text becomes the resolution ladder; document the printed `mode: ... (source)` line and the invalid-config-value error.
- `src/kit/skills/_cli-fab.md` § fab config explain — the "Full schema coverage" key enumeration (~line 412) gains `autopilot.merge_mode`.
- `docs/specs/config.md`: the scope-taxonomy `both` row's key list (~line 192) and the advertised-but-not-set-live list (~line 265) gain `autopilot.merge_mode`; a short schema mention alongside the dispatch-preference section.
- `docs/site/merge-topologies.md` § Choosing a mode: mention that the default is settable via `autopilot.merge_mode` (one `fab config set --system` line).
- `src/kit/skills/fab-operator.md` §6 Autopilot (per §3 above).
- Canonical skill sources only under `src/kit/skills/` — never `.claude/skills/` (gitignored deployed copies).
- The managed fence in each project's `fab/project/config.yaml` regenerates from the registry on `fab config upgrade` — no hand-edit; this repo's own fence updates when the kit version bumps.

### 5. Tests (constitution: CLI change ⇒ tests)

- `src/go/fab/cmd/fab/operator_autopilot_test.go`: flag-absent → config resolution; flag-present wins; invalid config value errors naming the valid set; printed source line; back-compat fill unchanged.
- `src/go/fab/internal/configref` tests (`configref_test.go`, `defaultsmap_test.go`): new row lints clean, default sources from the canonical symbol, advertised segment renders.
- `src/go/fab/internal/config` accessor test for the new field (absent/present/invalid).
- Any golden-output tests covering `fab config init`/`explain`/`upgrade` fence content (`config_show_init_test.go`, `config_upgrade_test.go`) updated for the new advertised block.

## Affected Memory

- `_shared/configuration.md`: (modify) new `autopilot.merge_mode` registry key — scope both, advertised, canonical-symbol default, the deliberate error-not-fail-open validation divergence
- `runtime/operator.md`: (modify) autopilot mode resolution ladder, silent-default skill rule with in-confirmation-line veto, the two enumerated pause-and-ask misfits, mode-question format (glyphs + diagrams + tradeoffs), status-frame-only scope of the no-diagrams caveat

## Impact

- **Go binary**: `src/go/fab/cmd/fab/operator_autopilot.go` (start resolution + printed source + config-value validation), `src/go/fab/internal/config/config.go` (struct field, canonical default symbol / valid-modes list, accessor), `src/go/fab/internal/configref/configref.go` (registry row + segment). Additive — no existing user data restructured, **no migration file needed** (new optional key only; the migration rule triggers on restructuring existing user data).
- **Skills**: `src/kit/skills/fab-operator.md` §6, `src/kit/skills/_cli-fab.md` (two sections).
- **Docs**: `docs/specs/config.md`, `docs/site/merge-topologies.md`.
- **Tests**: `operator_autopilot_test.go`, configref tests, config accessor tests, config init/upgrade golden tests.
- **Behavioral surface**: `fab operator autopilot start` gains stdout output (previously silent on success) — additive; no caller parses its empty output today. `--mode` semantics unchanged when passed.

## Open Questions

- None — the design discussion resolved the decision points; remaining choices (exact printed-source string, accessor error-split placement) are graded assumptions below, resolvable via `/fab-clarify` if desired.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Key named `autopilot.merge_mode` (not bare `mode`) | Discussed — user chose it explicitly to avoid overloading `dispatch.mode` | S:95 R:70 A:90 D:95 |
| 2 | Certain | Default `cherry-pick-ladder`; allowed values exactly `validAutopilotModes`; scope `both` | Discussed — user-agreed; `stacked-prs`-as-default explicitly rejected (fragile for unattended merge-all; #609 out-of-order merge proved ladder robustness) | S:95 R:75 A:95 D:95 |
| 3 | Certain | Resolution ladder: explicit instruction/`--mode` flag > config > built-in default; flag always wins | Discussed — user-agreed, binary and skill kept consistent | S:95 R:75 A:90 D:95 |
| 4 | Certain | Skill proceeds silently when no mode named; resolved mode stated inside the existing upfront queue-confirmation line | Discussed — user-agreed; veto rides the existing confirmation, no extra round-trip | S:90 R:75 A:85 D:90 |
| 5 | Certain | Pause-and-ask only on the two enumerated misfits (merge-auto vs same-repo `depends_on`; user message conflicts with resolved mode) | Discussed — user-agreed exact enumeration | S:90 R:80 A:85 D:90 |
| 6 | Certain | Mode question includes glyphs, the three box diagrams, and one-line tradeoffs; no-diagrams caveat clarified as status-frame-only | Discussed — user-agreed, including the caveat-scoping clarification | S:90 R:85 A:85 D:90 |
| 7 | Certain | Invalid config value errors actionably at `start` naming the valid set — deliberate divergence from `GetDispatchMode`'s fail-open warn | Discussed — user-agreed; merge topology is destructive-tier, silent fallback is the wrong failure mode | S:90 R:80 A:80 D:85 |
| 8 | Certain | Additive change, no migration file | context.md migration rule triggers only on restructuring existing user data; a new optional key restructures nothing | S:85 R:85 A:95 D:90 |
| 9 | Confident | `advertise: true` (commented in the managed fence) | User deferred to precedent; `dispatch.mode` is the closest analogue (both-scope enum preference knob, advertised); the demotion precedent (260806-j9nh) covered role/provider machinery blocks, not top-level preference knobs | S:55 R:90 A:65 D:55 |
| 10 | Confident | Canonical valid-modes list + default move to `internal/config`; `cmd/fab` and `configref` reference it | Single-source constraint is user-stated; placement follows the `DefaultDispatchMode` canonical-symbol pattern and the only import direction the graph allows (configref already imports config; cmd/fab imports both) | S:70 R:85 A:80 D:70 |
| 11 | Certain | Flag-absent detection via `cmd.Flags().Changed("mode")`, keeping the flag default for help text | Only mechanism cobra offers to distinguish absent from explicitly-default; zero behavior change for explicit flags | S:60 R:90 A:90 D:85 |
| 12 | Confident | `loadAutopilot`'s absent-`mode` back-compat fill stays the built-in default, never config-resolved | Mode is fixed at queue start and persisted; config-resolving at read time could silently retopologize a running queue | S:50 R:80 A:80 D:70 |
| 13 | Confident | Printed source line format `mode: <name> (<source>)` where source is one of flag, config, default | User decided the WHAT (mode + source); exact string follows `fab dispatch`'s existing `mode: <rung> (<reason>)` output style | S:65 R:95 A:75 D:60 |

13 assumptions (9 certain, 4 confident, 0 tentative, 0 unresolved).
