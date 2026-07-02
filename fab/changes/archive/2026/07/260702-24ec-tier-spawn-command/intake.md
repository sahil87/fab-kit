# Intake: Per-Tier spawn_command — Cross-Harness Stage Dispatch Opt-In

**Change**: 260702-24ec-tier-spawn-command
**Created**: 2026-07-02

## Origin

<!-- How was this change initiated? Include the user's raw input/prompt, the interaction
     mode (one-shot vs. conversational), and key decisions from the conversation. -->

> Per-tier `spawn_command` in `agent.tiers` — the cross-harness opt-in for CLI stage dispatch, with a `resolve-agent` `spawn=` output line and a commented-block migration.

This is **part 3b of a four-part series** enabling cross-harness stage dispatch — running an individual pipeline stage (e.g. `apply`) headless on a *different* CLI harness (codex, etc.) while the pipeline is orchestrated from Claude Code. The series:

- **3a** — `fab status refresh` + artifact-write hook removal (drafted separately).
- **3b (this change)** — widen the tier profile with a per-tier `spawn_command`; `fab resolve-agent` emits a third `spawn=` line when a resolved tier carries one; a config-only migration documents the new field.
- **3c** — a `fab dispatch` process-manager command family (drafted in parallel; it **consumes** this change's resolution — the `spawn=` line 3b emits is 3c's input).
- **3d** — a harness-adapters spec + the skill dispatch-seam wiring and result protocol (spec-first, later).

**Recently merged foundations this change builds on**:

- **PR #455** — `fab config reference` (the generated-from-constants commented reference `config.yaml`), whose `agent.tiers` section this change extends to document `spawn_command`.
- **PR #456** — `spawn_command` `{model}`/`{effort}` placeholder substitution in `internal/spawn.WithProfile` (change `260702-6tmi`). Its `configuration.md` § `agent` memory entry explicitly names this change as the sanctioned follow-up: *"Non-goals: per-tier `spawn_command` and a cross-harness stage-dispatch adapter (a follow-up change, spec-first)."* This change is that follow-up.

Interaction mode: dispatched as a background `/fab-draft` (promptless-defer). There is no live conversation; the decisions below were pre-mined and supplied by the dispatcher as user-approved choices (recorded as Certain/Confident assumptions with rationale "Discussed — user approved …").

## Why

<!-- Explain the motivation substantively. -->

**Problem.** Today fab has two distinct agent-spawning surfaces, and neither can run *one pipeline stage* on a *different harness*:

1. `agent.spawn_command` (the worker/session boundary) — used by `fab operator`, `fab batch`, and `fab spawn-command` to open a whole interactive agent *session*. It is project-wide, not per-stage.
2. `agent.tiers.<tier>` (`{model, effort}`) — resolved per stage by `fab resolve-agent <stage>` and injected into a **native Agent-tool** sub-agent dispatch. This selects *which model/effort* a stage runs at, but always through Claude Code's own Agent tool — it cannot hand a stage off to an external CLI.

There is no way to say "run the `apply` stage by shelling out to `codex exec …` and reading its result back," which is the cross-harness goal of the 3x series. `agent.spawn_command` is the wrong seam (it opens a session, not a stage), and `agent.tiers` currently carries no dispatch-command field.

**Consequence if unfixed.** The 3c `fab dispatch` command family (drafted in parallel) has nothing to consume — it needs a resolved, per-stage command string to run. Without this change the cross-harness series stalls at 3b; every stage stays locked to native Agent-tool dispatch.

**Why this approach.** Widening the *existing* per-stage resolution surface (`agent.tiers` + `fab resolve-agent`) is the minimal, on-model extension: the tier is already the per-stage unit, `fab resolve-agent <stage>` already resolves the tier profile, and PR #456 already built the `{model}`/`{effort}` template machinery this change reuses for the `spawn=` line. The alternative — a new top-level `stage_dispatch` map or reusing `agent.spawn_command` per-stage — was rejected: the former duplicates the stage→tier resolution fab already owns, and the latter conflates the session boundary with the stage boundary (see the critical no-cross-fallback semantics below).

## What Changes

<!-- Be specific. Use subsections per change area. Include concrete examples. -->

### 1. Widen the tier profile: `{model, effort}` → `{model, effort, spawn_command}`

Add a `SpawnCommand` field to `TierProfile` in `src/go/fab/internal/config/config.go`:

```go
// TierProfile is a named {model, effort, spawn_command} agent profile. …
type TierProfile struct {
	Model        string `yaml:"model"`
	Effort       string `yaml:"effort"`
	SpawnCommand string `yaml:"spawn_command"` // NEW — the per-tier CLI-dispatch command (opt-in)
}
```

Mirror the field on the resolution-side `agent.Profile` struct (`src/go/fab/internal/agent/agent.go`) and extend the per-field merge in `agent.Resolve`:

```go
// in Resolve, after the model/effort merge:
if override.SpawnCommand != "" {
	resolved.SpawnCommand = override.SpawnCommand
}
```

**fab-kit's built-in default tiers carry NO `spawn_command`** — `defaultTiers` in `internal/agent` stays `{model, effort}` only. The `spawn_command` field is populated **exclusively from user config** (`agent.tiers.<tier>.spawn_command`). A default tier resolves with an empty `SpawnCommand`, which means "native Agent-tool dispatch" (see semantics below). This keeps the field a pure opt-in and preserves today's behavior for every project that does not set it.

### 2. The critical semantics — no cross-fallback (the load-bearing decision)

> **Mental model (user-approved):** "`agent.spawn_command` opens agent *sessions*; `agent.tiers.<tier>.spawn_command` runs one *stage*." Hire an employee vs. outsource one task. State this prominently everywhere the field is documented.

- `agent.tiers.<tier>.spawn_command` **PRESENT** → stages in that tier are **CLI-dispatched** using that command.
- `agent.tiers.<tier>.spawn_command` **ABSENT** → **native Agent-tool dispatch** (today's behavior, unchanged).

**It does NOT fall back to `agent.spawn_command`.** This is the single most important semantic in the change. A cross-fallback would silently flip *every* project that has ever set `agent.spawn_command` (which is common — this repo sets it) into CLI dispatch for all stages, changing behavior across the board on upgrade. The two fields are **independent surfaces**:

- `agent.spawn_command` remains the **worker/session boundary only** — `fab operator`, `fab batch`, `fab spawn-command`. Untouched by this change.
- `agent.tiers.<tier>.spawn_command` is the **per-stage CLI-dispatch opt-in** — new, and read only through the tier-resolution path.

The absence of a resolved tier `spawn_command` is the signal for "native dispatch"; there is no lookup of `agent.spawn_command` from the stage-dispatch path.

### 3. `fab resolve-agent <stage>` emits a third output line: `spawn=<command>`

Extend the `fab resolve-agent` output contract with a **third** line, present **ONLY when the resolved tier carries a `spawn_command`**:

```
model=<id>
effort=<level>
spawn=<command>
```

- `spawn=` is **omitted entirely** when the resolved tier has no `spawn_command` (mirroring the existing "`effort=` omitted when empty" rule). Its absence is what tells a caller "native dispatch."
- The emitted command has its `{model}`/`{effort}` placeholders **already substituted** via PR #456's template rules. **Reuse `internal/spawn`'s template resolution** (`spawn.WithProfile(tierSpawnCommand, resolvedModel, resolvedEffort)` or the equivalent existing entrypoint) — **do NOT reimplement** the substitution/token-drop logic. The model/effort fed into the substitution are the tier's own resolved model/effort (post per-field-merge).
- **Byte-stable** for the same config, like the other two lines and all `fab resolve` queries.

**Interaction with `--alias`** (user-approved):

- With `--alias`, the `model=` line stays aliased **exactly as today** (`opus`/`sonnet`/…) — no change to that behavior.
- The `spawn=` line **always embeds the FULL model ID**, never the alias. Aliasing is a Claude-Code **Agent-tool-only** adaptation (the Agent tool's `model` param is a short-alias enum); **CLI dispatch never aliases** — an external CLI's `--model` flag takes a full ID. So even under `--alias`, the `{model}` placeholder in the `spawn=` command is substituted with the full resolved ID.

Worked example (a project that sets `agent.tiers.apply`… i.e. the `doing` tier … to a codex `spawn_command`):

```
$ fab resolve-agent apply --alias
model=opus
effort=high
spawn=codex exec -m claude-opus-4-8 -c model_reasoning_effort=high
```

Here `model=opus` is aliased (Agent-tool half), but `spawn=` embeds the full `claude-opus-4-8` (CLI half). For a project with no tier `spawn_command`, `fab resolve-agent apply --alias` is byte-identical to today (two lines, no `spawn=`).

### 4. `fab config reference` template update

Update the `agent.tiers` section of the generated reference (`internal/configref`) to document the new `spawn_command` field and its opt-in semantics: **present → CLI dispatch; absent → native Agent-tool dispatch; no fallback to `agent.spawn_command`.** Since the reference is generated from constants, ensure any per-tier default rendering still reflects that fab-kit's defaults carry no `spawn_command` (the field appears only as documented-but-commented guidance, not a shipped default value).

### 5. Migration — config-only, sentinel-guarded, idempotent (user explicitly requested)

Ship a new migration file `src/kit/migrations/2.10.1-to-2.11.0.md` (next version slot — see the Impact § note on the slot) that appends a **SHORT commented block** under existing repos' `fab/project/config.yaml` `agent:` section, documenting the tier `spawn_command` field and its opt-in semantics, ending with a pointer to `fab config reference` for the full reference.

Follow the **`2.2.0-to-2.3.0` migration precedent** (the `agent.tiers` reference block): comment-sentinel idempotency, skip when config is absent or the sentinel is already present, insert under the `agent:` block. Keep the block **brief** — the reference command is the canonical documentation, so the migration only needs to *announce* the field and point at `fab config reference`. Illustrative shape (final wording to be settled at apply):

```yaml
  # agent.tiers.<tier>.spawn_command — per-stage CLI dispatch (optional, opt-in).
  # PRESENT on a tier → that tier's stages are dispatched by running this command
  # (cross-harness, e.g. codex); ABSENT → native Agent-tool dispatch (default).
  # This is INDEPENDENT of agent.spawn_command (which opens whole agent sessions):
  # there is NO fallback from a tier to agent.spawn_command. {model}/{effort}
  # placeholders are substituted at resolve time. See: fab config reference.
```

The migration is config-only (no `.status.yaml` schema change) and — like `2.9.2-to-2.10.0` — needs **no binary capability pre-check** for the comment itself, though note the field it documents does require the widened binary (that is the version-gating point of shipping it in this slot). Bump VERSION to `2.11.0` as part of the change (consistent with the recent config-additive migrations `2.7.1-to-2.8.0` and `2.9.2-to-2.10.0`, both MINOR bumps).

### 6. Non-goals (explicit)

- **The dispatch execution itself** — `fab dispatch` (the 3c process-manager command family that *runs* the resolved `spawn=` command). This change only *emits* the command; it does not execute it.
- **The skill dispatch-seam wiring and result protocol** — 3d. No skill changes wire `spawn=` into an actual cross-harness dispatch here.
- **Any validation of the spawn command string** — provider-neutral verbatim pass-through, per Constitution Principle I and `stage-models.md` § No validation. fab does not check that the command is runnable, well-formed, or points at a real binary.

## Affected Memory

<!-- Which memory files will be created, modified, or removed. -->

- `_shared/configuration`: (modify) — extend § `agent` `tiers` to document the new `spawn_command` field, the widened `TierProfile`/`AgentConfig` shape, and the no-cross-fallback semantics; update the `## Design Decisions` section (a new "per-tier spawn_command" decision, or extend the existing `agent.tiers` decision) with the hire-an-employee-vs-outsource-a-task mental model and the rejected cross-fallback alternative. Also touch the `fab config reference` coverage note if the reflection-over-`Config` test now sees a new tagged field.
- `_shared/context-loading`: (modify) — § Per-Stage Model Resolution documents the `fab resolve-agent` output contract (currently "two byte-stable stdout lines") and the Harness-adapter boundary; update to note the optional third `spawn=` line, the present/absent CLI-vs-native semantics, and that `spawn=` embeds the full model ID even under `--alias` (CLI dispatch never aliases).
- `distribution/migrations`: (modify) — add the new `2.10.1-to-2.11.0` migration to the migration inventory prose (both `docs/memory/distribution/migrations.md` and the `distribution/index.md` description string that enumerates migrations), following the pattern of the `2.9.2-to-2.10.0` entry.

## Impact

<!-- Affected code areas, APIs, dependencies, systems. -->

**Go binary (`src/go/fab/`):**

- `internal/config/config.go` — add `TierProfile.SpawnCommand` (`yaml:"spawn_command"`). Ships parse tests (the widened struct round-trips; a config with a tier `spawn_command` loads it; `GetAgentTier` returns it).
- `internal/agent/agent.go` — add `Profile.SpawnCommand`; extend `Resolve`'s per-field merge (override wins when set; default tiers carry none). Ships merge tests (tier with `spawn_command` set → resolved profile carries it; tier without → empty; per-field merge with only `spawn_command` set keeps default model/effort).
- `cmd/fab/resolve_agent.go` + `formatAgentProfile` — emit the `spawn=` line when non-empty, with `{model}`/`{effort}` substituted via `internal/spawn` (reuse, don't reimplement). Ships `resolve_agent_test.go` cases: no-tier-spawn (two lines, unchanged), tier-spawn present (three lines), `--alias` with tier-spawn (aliased `model=`, full-ID `spawn=`), placeholder substitution correctness, empty-value token-drop behavior inherited from `spawn`.
- `internal/configref/configref.go` — update the `agent.tiers` reference template + its coverage test (`cmd/fab/config_test.go` reflection-over-`Config` will require the reference to mention the new tagged field).
- `internal/spawn` — **reused, not modified** (unless a thin helper is warranted); the substitution/token-drop logic is PR #456's and stays canonical.

**Kit content (`src/kit/`):**

- `src/kit/skills/_cli-fab.md` § `fab resolve-agent` — document the third `spawn=` line, its present-only-when-tier-has-spawn_command rule, the full-ID-under-`--alias` rule, and an updated worked example. **CLI output change ⇒ this file MUST update** (Constitution Additional Constraints).
- `src/kit/migrations/2.10.1-to-2.11.0.md` — new migration (per constitution: user-data-adjacent config documentation ships as a migration file, not an ad-hoc script).
- VERSION bump to `2.11.0`.

**Specs (`docs/specs/`) — mirror-sweep class (per code-quality.md § Sibling & Mirror Sweeps; grep `resolve-agent` + `tiers` was run at intake):**

- `docs/specs/stage-models.md` — the primary spec. Update § Config schema (`agent.tiers` gains `spawn_command`), § Resolution — `fab resolve-agent <stage>` (the output now has an optional third `spawn=` line; state the omit-when-absent rule and the no-cross-fallback semantics), and § Harness-adapter boundary (the `spawn=` line is the CLI-dispatch adapter; it never aliases; contrast with the Agent-tool `model` param). Add the per-tier `spawn_command` to the design as the sanctioned 6tmi follow-up. This is the drift-guard anchor for the `internal/agent` tables — confirm the `defaultTiers`/`stageTiers` mirror tables are unaffected (they are: defaults gain no `spawn_command`).
- `docs/specs/skills/SPEC-_cli-fab.md` — mirror of `_cli-fab.md` (constitution-required to stay in sync with the skill change).
- `docs/specs/architecture.md` — documents the `agent.tiers` config block; add the `spawn_command` field.
- On a CLI/command-signature change, treat **all** of `_cli-fab.md`'s SPEC mirrors as the sweep class (code-quality.md note). Grep at apply for any other spec restating the "two stdout lines" / "two byte-stable" phrasing (`SPEC-_preamble.md`, `SPEC-fab-*`, `_preamble.md` § Per-Stage Model Resolution, `context-loading.md`) and update every occurrence to the optional-third-line contract.

**Skill prose that restates the two-line output** (grepped: `_preamble.md`, `_pipeline.md`, `fab-continue.md`, `fab-ff.md`, `fab-fff.md`, `fab-proceed.md`, `fab-operator.md`, `fab-adopt.md` all reference `resolve-agent` output): audit each at apply. Most consume `model=`/`effort=` and will be **unaffected** (they never read `spawn=` — that is 3c/3d's job), but `_preamble.md` § Per-Stage Model Resolution and `_cli-fab.md` describe the *contract* and MUST reflect the new optional line. The dispatch-seam skills that only inject model/effort do **not** need to grow `spawn=` handling in this change (that is 3c/3d) — but any prose asserting "the output is exactly two lines" is now false and must soften to "two lines, plus an optional third `spawn=` line."

**Migration slot note.** The latest git tag is `v2.10.1`; the newest existing migration is `2.9.2-to-2.10.0.md`. The next slot is therefore `2.10.1-to-2.11.0` (from the current `2.10.1` to a new `2.11.0`). Confirm at apply that no `2.10.1-to-*` migration has landed on `main` in the interim; if the tip has moved, use the actual current version as the `from` and the next MINOR as the `to`.

## Open Questions

<!-- Clarifying questions the agent couldn't resolve from context alone. -->

- Should `internal/agent.Profile` and `internal/config.TierProfile` both grow the field independently, or should `Resolve` continue to copy field-by-field (today's pattern is field-by-field with the default-then-override merge — the plan should follow it rather than introduce struct embedding)?
- Does the `spawn=` line need any shell-escaping/quoting guarantee for downstream 3c consumption, or is verbatim (post-substitution) sufficient? (Leaning verbatim per the no-validation principle — 3c owns execution and any quoting it needs — but flagging for the apply-entry design.)

## Assumptions

<!-- STATE TRANSFER: this table is the sole continuity mechanism between intake and apply. -->

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Widen `TierProfile` (config) and `Profile` (agent) with `SpawnCommand`; extend `agent.Resolve`'s per-field merge (override wins when set). fab-kit built-in default tiers carry NO `spawn_command`; the field is user-config-only. | Discussed — user approved. Deterministic from the existing `{model, effort}` per-field-merge pattern in `agent.Resolve` (config.go:137 `GetAgentTier`, agent.go:147 merge). Config-additive; yaml.v3 ignores unknown keys so it is free for existing configs. | S:95 R:80 A:95 D:95 |
| 2 | Certain | No cross-fallback: tier `spawn_command` PRESENT → CLI dispatch; ABSENT → native Agent-tool dispatch. Never falls back to `agent.spawn_command`. The two fields are independent surfaces (session boundary vs. stage dispatch). | Discussed — user approved as the load-bearing semantic. A cross-fallback would silently flip every project with a configured `agent.spawn_command` into CLI dispatch. The mental model "spawn_command opens sessions; tier spawn_command runs one stage" is stated verbatim by the user. | S:95 R:70 A:90 D:95 |
| 3 | Certain | `fab resolve-agent <stage>` emits a third `spawn=<command>` line ONLY when the resolved tier carries a `spawn_command`; omitted otherwise (mirrors the effort-omit rule). `{model}`/`{effort}` substituted via `internal/spawn`'s existing template resolution (reuse, not reimplement). Byte-stable. | Discussed — user approved. The omit-when-empty pattern already exists (`formatAgentProfile`, resolve_agent.go:73); PR #456 (`spawn.WithProfile`/`resolveTemplate`) supplies the substitution. Constitution I (reuse over duplication) + code-quality anti-pattern (no duplicating utilities). | S:90 R:75 A:90 D:90 |
| 4 | Certain | Under `--alias`: `model=` stays aliased (Agent-tool half, unchanged); `spawn=` ALWAYS embeds the FULL model ID. CLI dispatch never aliases — aliasing is Agent-tool-only. | Discussed — user approved. Consistent with the existing `--alias` semantics (context-loading.md § Per-Stage Model Resolution: the `claude` CLI accepts full IDs, the Agent tool's enum rejects them). The `{model}` placeholder feeds the full resolved ID even when `model=` is aliased. | S:90 R:80 A:95 D:90 |
| 5 | Confident | Ship a config-only, sentinel-guarded, idempotent migration in the next version slot appending a SHORT commented block under `agent:` documenting the tier `spawn_command` field, ending with a pointer to `fab config reference`. Follow the `2.2.0-to-2.3.0` precedent. | Discussed — user explicitly requested a migration and named the `2.2.0-to-2.3.0` precedent. "Short" and "pointer to reference" are the user's stated constraints. The exact block wording is a Tentative sub-decision left to apply (marked below); the migration's existence/shape/precedent is Certain-adjacent but the wording latitude drops it to Confident. | S:85 R:80 A:85 D:75 |
| 6 | Confident | Migration slot is `2.10.1-to-2.11.0` (from current `2.10.1` to a new `2.11.0`); VERSION bumps to `2.11.0`. Config-additive changes take a MINOR bump. | Latest git tag is `v2.10.1`; newest migration is `2.9.2-to-2.10.0.md`; recent config-additive migrations (`2.7.1-to-2.8.0`, `2.9.2-to-2.10.0`) all took MINOR bumps. Confident (not Certain) because the tip could move before apply — the plan must re-confirm the current version and adjust the `from` if `main` advanced. | S:80 R:75 A:80 D:80 |
| 7 | Confident | `fab config reference` (`internal/configref`) `agent.tiers` template documents the new `spawn_command` field + the present=CLI / absent=native semantics; the coverage test (reflection over `Config`) is updated for the new tagged field. | Discussed — user approved (decision 4 of the mined set). Deterministic from the existing generated-from-constants reference design (configuration.md § Schema Discovery) and its two coverage tests (`cmd/fab/config_test.go`). Confident because the exact rendered wording is authored at apply. | S:85 R:80 A:85 D:80 |
| 8 | Confident | Full mirror-sweep at apply: `stage-models.md` (§ Config schema, § Resolution, § Harness-adapter boundary), `_cli-fab.md` + `SPEC-_cli-fab.md`, `architecture.md`, and every spec/skill restating the "two stdout lines" resolve-agent contract must reflect the optional third line. Go changes ship tests (config parse, merge, resolve-agent output incl. empty/alias cases). | code-quality.md § Sibling & Mirror Sweeps (must-fix rework cause); Constitution Additional Constraints (CLI ⇒ `_cli-fab.md` + tests; skill ⇒ SPEC mirror; Go ⇒ tests). Grep for `resolve-agent`/`tiers` run at intake to seed the class. Confident because the exact set of prose occurrences is enumerated fully at apply, not intake. | S:85 R:75 A:85 D:80 |
| 9 | Confident | Non-goals held firm: no `fab dispatch` execution (3c), no skill dispatch-seam wiring / result protocol (3d), no validation of the spawn command string (verbatim pass-through, Constitution I). | Discussed — user stated the non-goals explicitly, and 6tmi's memory note pre-reserves exactly this scope split ("per-tier spawn_command and a cross-harness stage-dispatch adapter — a follow-up change, spec-first"). | S:90 R:75 A:90 D:85 |
| 10 | Tentative | Exact wording of the migration's commented block (the illustrative YAML in § 5 is a draft, not final). | The user said "short" + "pointer to reference" and named the precedent, but did not dictate the exact comment text; multiple acceptable phrasings exist. Resolved inline at apply as a graded assumption. | S:60 R:85 A:70 D:55 |
| 11 | Tentative | Whether the `spawn=` line needs any shell-quoting/escaping guarantee for 3c, or verbatim post-substitution output is sufficient (leaning verbatim). | No signal in the mined decisions; 3c owns execution and its own quoting. Verbatim aligns with the no-validation principle, but a downstream consumer's needs are not yet specified (3c is drafted in parallel). Low-blast-radius (a formatting refinement 3c could request later). | S:45 R:75 A:55 D:50 |

11 assumptions (4 certain, 5 confident, 2 tentative).
