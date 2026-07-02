---
type: memory
description: "Smart context loading convention — descriptive 7-file always-load layer (skill file wins; exception set rule-derived from skill files, never enumerated in the preamble — d9rs), opt-in skill helpers (7-value allowlist incl. `_srad`/`_pipeline`/`_intake`) + stage-conditional loading, standard subagent context (orchestrators incl. `/fab-proceed`), per-stage model resolution at the dispatch seam (`fab resolve-agent <stage>`, provider-neutral, Claude-Code adapter named, review-resolves-once — l3ja; applies on every post-intake stage since the single-dispatch collapse, advisory only for a genuinely no-dispatch run — fgxx; the two halves dispatch through two seams — model via the Agent `model` param (short-alias enum opus/sonnet/haiku/fable, resolved with `fab resolve-agent <stage> --alias` so the alias is emitted directly — the deterministic adapter that superseded m3d4's prompt-side id→alias hand-map; yky7), effort via an imperative subagent-prompt instruction (no Agent effort param; omit when empty), plus a compliance-visibility expectation that each site surface the resolved `model=/effort=` — m3d4; residual = a per-sub-agent effort param on the Agent tool, a harness ask; output is two byte-stable lines plus an optional third `spawn=` line when the resolved tier carries a per-tier `spawn_command` — present=CLI dispatch / absent=native, `spawn=` embeds the FULL model ID even under `--alias`, 24ec), selective domain loading, SRAD protocol pointer, scoped Next Steps Convention, generic fab-command failure rule (unconditional non-zero exit → STOP; `fab log command` exits 0 by contract)"
---
# Context Loading

**Domain**: _shared

## Overview

The context loading convention defines how fab skills load project context before execution. It is implemented in `$(fab kit-path)/skills/_preamble.md` as a shared preamble read by all skills. The convention uses a layered approach: always-load essentials, change-specific artifacts, and selective domain memory loading.

## Requirements

### Always Load Layer (Descriptive — Skill File Wins)

The always-load layer is the **default** every skill inherits **unless the skill's own Context Loading section says otherwise** — the skill file wins (made explicit by 260611-zc9m; the contract is descriptive, not exhaustive, so self-exempting skills no longer contradict the preamble). Override is opt-in, not opt-out-by-silence: a skill with no Context Loading section still defaults to the full layer.

**The exception set is rule-derived, never enumerated in the preamble (d9rs)**: `_preamble.md` §1 no longer names the exception skills — the authoritative source for any override is the skill file itself (its `## Context Loading` section, or an explicit context note near its header, e.g. `fab-proceed.md`'s "skips preflight/context loading itself" note). The preamble keeps only illustrative examples (`/fab-setup` and `/docs-hydrate-memory` skip the layer entirely; `/fab-operator` loads a reduced 3-file set). This is the root-cause fix for the enumeration having drifted: the zc9m-era preamble list named four skips while the shipped override set was larger (`/fab-help`, `/fab-archive`, `/docs-hydrate-specs`, `/docs-reorg-memory`, `/docs-reorg-specs`, and `/fab-proceed` also declare their own context behavior). `/docs-hydrate-memory` gained an explicit `## Context Loading` section in the same change — previously its exemption existed only as a preamble list entry, with no skill-file override for the rule to key on. See § Exception Skills below for the shipped set as of d9rs.

Skills on the default path read seven files as baseline context:

1. `fab/project/config.yaml` — project configuration: identity (name/description), `source_paths`/`test_paths`, true-impact excludes, plan-acceptance extra categories, `review_tools` toggles, agent spawn command, optional `stage_hooks` (descriptor rewritten in c5tr — it formerly advertised "naming conventions, model tiers", both removed/dead surfaces)
2. `fab/project/constitution.md` — project principles and constraints (MUST/SHOULD/MUST NOT rules)
3. `fab/project/context.md` — free-form project context: tech stack, conventions, architecture *(optional — no error if missing)*
4. `fab/project/code-quality.md` — coding standards for apply/review: principles, anti-patterns, test strategy *(optional — no error if missing)*
5. `fab/project/code-review.md` — review policy: severity definitions, scope, rework budget *(optional — no error if missing)*
6. `docs/memory/index.md` — documentation landscape (which domains exist; a domain may contain sub-domains, surfaced in that domain's index via a `## Sub-Domains` table — see Selective Domain Loading)
7. `docs/specs/index.md` — specifications landscape (pre-implementation design intent, human-curated)

This gives the agent awareness of project settings, constraints, project context, coding standards, review policy, the documentation landscape, and the specifications landscape before generating any artifact.

The only universal helper beyond the 7 project files is `_preamble.md`. Additional helpers are declared per-skill via the `helpers:` frontmatter field — see **Skill Helper Declaration (Opt-In)** below. Naming conventions and run-kit (rk) recipes that were previously separate helpers are now inlined into `_preamble.md` (§ Naming Conventions, § Run-Kit (rk) Reference). Common `fab` commands are inlined into `_preamble.md` § Common fab Commands so most skills do not need `_cli-fab`.

### Skill Helper Declaration (Opt-In)

Skills declare additional helper files via the `helpers:` frontmatter list. Allowed values (seven since 260613-3xaj, which added `_intake`; six since 260611-szxd, which added `_pipeline`): `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`, `_intake`. The agent MUST read `.claude/skills/{helper}/SKILL.md` for each declared helper after reading `_preamble` and before executing the skill body.

**Stage-conditional loading** (260611-zc9m): a skill MAY instead load a helper at its point of use via an explicit in-body read instruction (e.g., "read `.claude/skills/_review/SKILL.md` before entering Review Behavior"). Frontmatter `helpers:` declares unconditional pre-body loads; in-body read instructions declare conditional ones — a helper loaded this way is intentionally absent from the frontmatter list, so the frontmatter contract stays honest. `/fab-continue` is the sole current user: `_generation` at apply entry / intake-`active` regeneration, `_review` at Review Behavior entry (see [pipeline/execution-skills.md](/pipeline/execution-skills.md)).

Current mapping (post-260613-3xaj):

| Skill(s) | `helpers:` |
|----------|------------|
| `fab-new`, `fab-draft` | `[_generation, _srad, _intake]` (3xaj added `_intake`; both keep `_generation`/`_srad` declared directly — the `_pipeline` precedent, consumers declare underlying helpers rather than inheriting transitively) |
| `fab-continue` | `[_srad]` (+ point-of-use in-body reads of `_generation`/`_review`) |
| `fab-ff`, `fab-fff` | `[_generation, _review, _srad, _pipeline]` (orchestrator-level rework edits `plan.md` sections directly, so `_generation` stays unconditional — finding f074 refuted; `_pipeline` is the shared ff/fff pipeline bracket and constitutes the wrappers' entire body, so its load is unconditional by construction — szxd) |
| `fab-clarify` | `[_srad]` |
| `fab-operator` | `[_cli-fab, _cli-external]` |
| All others (16 skills) | omitted / `[]` (load only `_preamble`) |

`_naming` and `_cli-rk` are NOT allowed values — their content is inlined into `_preamble`. `_preamble` itself is implicit and never listed. `/fab-proceed` declares **no** `helpers:` (it dispatches `_intake` as a subagent prompt — the subagent reads the helper — exactly as it dispatched `/fab-new` before 3xaj). The internal helpers `_generation`, `_review`, `_pipeline`, and `_intake` themselves carry no `helpers:` frontmatter — they reference what they need in-body and rely on the consumer (or dispatched subagent) having loaded it.

**One shared helper per pipeline phase (completed by 3xaj).** The four internal orchestration/mechanics helpers decompose the workflow symmetrically — each is a shared body parameterized by call-site-specific knobs, with call-site tails staying in the consumer files:

| Phase | Helper | Knob(s) | Consumers |
|-------|--------|---------|-----------|
| artifact mechanics | `_generation` | — | `fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff` |
| review mechanics | `_review` | — | `fab-continue`, `fab-ff`, `fab-fff` |
| **pre-intake orchestration** | **`_intake`** | `{questioning-mode}` | `fab-new`, `fab-draft`, `fab-proceed` |
| post-intake orchestration | `_pipeline` | `{driver}`, `{terminal}` | `fab-ff`, `fab-fff` |

`_intake` (3xaj) is the **pre-boundary** counterpart to the **post-boundary** `_pipeline` (szxd): intake is the single context-bearing boundary in the pipeline; everything up to and including intake creation runs in the main session context (pre-boundary: `_intake`), everything after runs as dispatched subagents over the intake artifact (post-boundary: `_pipeline`). Both extractions mirror the same shape (shared body + one-or-two knobs + call-site tails). See [pipeline/planning-skills.md](/pipeline/planning-skills.md) § The `_intake` Shared Create-Intake Procedure for the full pre-boundary decomposition.

### Preflight Script for Change Context

Skills that operate on an active change resolve the change context by running `src/kit/scripts/lib/preflight.sh [change-name]` via Bash. The script accepts an optional first positional argument as a change name override. When provided, the script resolves the change using case-insensitive substring matching against folder names in `fab/changes/` (excluding `archive/`) instead of reading `.fab-status.yaml`. The override is transient — `.fab-status.yaml` is never modified. When no argument is provided, the script falls back to reading `.fab-status.yaml` (backward compatible).

The matching supports full folder names, partial slug matches, and 4-char random IDs (e.g., `r3m7`). Exact match takes priority; single partial match resolves directly; multiple matches or no match produce a non-zero exit with a descriptive error.

The script validates project initialization, the change directory, and `.status.yaml`, then outputs structured YAML with name, stage, branch, progress, and checklist fields. On non-zero exit, the agent stops and surfaces the stderr error message. On success, the agent uses the stdout YAML instead of re-reading `.status.yaml`.

Since the preflight script validates `config.yaml` and `constitution.md` existence, skills using preflight don't need separate existence checks for these files — they only need to read them for content.

The existing 4-step inline validation sequence (check current, check directory, check .status.yaml, check config/constitution) remains documented in `_preamble.md` as reference for what the script validates internally.

### Generic fab-Command Failure Rule

`_preamble.md` § Common fab Commands "Key behaviors" carries a generic failure rule covering every fab invocation, not just preflight: **any fab command that exits non-zero → STOP and surface stderr** — resumability handles the re-run. The rule is unconditional — there is no guard-marked exemption class anymore (the former "not explicitly marked best-effort (`2>/dev/null || true`)" qualifier and its call-site guard boilerplate were retired in 260612-ye8r): `fab log command` can never trip the rule through internal failure because it owns its best-effort contract in Go — it always exits 0 given valid usage, surfacing internal failures as a stderr warning only (a cobra arg-count error still exits non-zero before RunE), so no shell guard is needed. The rule **defers to explicit per-skill handling** where a skill intentionally branches on a non-zero exit by design (e.g., `fab-proceed`'s active-change probe, `fab-discuss`'s context probe, `git-pr`'s already-shipped check, `fab-archive`'s archive-state check) — those carve-outs are unaffected. This closes the gap where only preflight (§2 step 2) had a stated non-zero-exit STOP and a mid-pipeline failure of any other fab command (e.g., `fab status finish`) had no defined handling, risking skills proceeding with silently diverged state.

### Selective Domain Loading

When operating on an active change, skills selectively load relevant memory files based on the change's scope. An Affected Memory entry is either **flat** (`{domain}/{name}`) or **sub-domained** (`{domain}/{sub-domain}/{name}` — used after an over-wide domain has been split by `docs-reorg-memory`). Loading is an **up-to-3-hop walk**:

1. Read the intake's Affected Memory section to identify relevant domains (and sub-domains)
2. **Domain index**: for each referenced domain, read `docs/memory/{domain}/index.md` — its `## Sub-Domains` table lists any sub-domains the domain contains
3. **Sub-domain index** *(only if the entry is sub-domained)*: when the referenced file lives in a sub-domain (3-part `{domain}/{sub-domain}/{name}` form), read `docs/memory/{domain}/{sub-domain}/index.md` next
4. **File**: read the specific memory file referenced — `docs/memory/{domain}/{name}.md` for a flat entry, or `docs/memory/{domain}/{sub-domain}/{name}.md` for a sub-domained entry
5. If a referenced domain, sub-domain, or file doesn't exist yet, note this and proceed without error (it will be created during hydrate)
6. Do not load unrelated domains — keeps context focused and efficient

A flat domain is just the degenerate 2-hop case (domain index → file); the sub-domain index hop is taken only when the Affected Memory entry carries the 3-part form. This matches `_preamble.md` § Memory File Lookup and `SPEC-_preamble.md` (renamed from `SPEC-preamble.md` in 260611-uliv — partial SPECs keep the leading underscore). The always-load layer is unchanged in *which* files it loads (root + domain indexes); only its description acknowledges that a domain may contain sub-domains.

This applies to all skills operating on an active change, not just spec-writing skills.

### Standard Subagent Context

When orchestrator skills (`/fab-ff`, `/fab-fff`, and the prefix-step orchestrator `/fab-proceed` — added to the preamble's § Subagent Dispatch orchestrator list in d9rs; it dispatches the `_intake` Create-Intake Procedure (3xaj — was the full `/fab-new` skill), `/fab-switch`, and `/git-branch` as prefix steps before delegating) or middle agents (`/fab-continue`) dispatch subagents via the Agent tool, the subagent prompt MUST instruct the subagent to read a standard set of project files **before** executing its task. This is defined in `_preamble.md` § Standard Subagent Context and is distinct from the Always Load layer (which is for the parent agent itself).

The standard subagent context includes:

**Required** (subagent reports error if missing):
- `fab/project/config.yaml`
- `fab/project/constitution.md`

**Optional** (skip gracefully if missing):
- `fab/project/context.md`
- `fab/project/code-quality.md`
- `fab/project/code-review.md`

This is a subset of the Always Load layer — it includes the 5 `fab/project/**` files but excludes `docs/memory/index.md` and `docs/specs/index.md` (which are navigation aids for the parent agent, not project principles needed by subagents).

**Nested dispatch**: When a subagent dispatches its own sub-subagent (e.g., review sub-agent within `/fab-continue`), the inner prompt MUST also include the standard subagent context instruction. The same 5 files are loaded at every nesting level.

**Relationship to Always Load**: The Always Load layer is what the parent agent reads. The Standard Subagent Context is what the parent agent instructs its subagents to read. The parent does not re-pass `docs/memory/index.md` or `docs/specs/index.md` to subagents — those are for the parent's own domain awareness.

### Per-Stage Model Resolution (l3ja)

Per-stage model selection is wired into the **sub-agent dispatch seam** (`_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution is the canonical contract). **Immediately before dispatching each pipeline stage's sub-agent**, the dispatching skill — the orchestrators `/fab-ff`, `/fab-fff`, `/fab-proceed`, and `/fab-continue`'s own sub-agent dispatch — runs `fab resolve-agent <stage>` and passes the resolved profile into the Agent dispatch:

- **Output** is two byte-stable stdout lines, `model=<id>` and `effort=<level>` (the `effort=` line is omitted when the resolved tier has no effort), **plus an optional third `spawn=<command>` line** (24ec) — emitted **only when the resolved tier carries a per-tier `spawn_command`** (`agent.tiers.<tier>.spawn_command`), mirroring the effort-omit rule. Its **presence signals CLI dispatch** (the tier's stages are dispatched by running the command, e.g. codex); its **absence signals native Agent-tool dispatch** (today's behavior) — there is NO fallback to `agent.spawn_command` (the two are independent surfaces; see [configuration.md](/_shared/configuration.md) § `agent` `tiers`). The `{model}`/`{effort}` placeholders are substituted at resolve time via 6tmi's `spawn.WithProfile` (reused), and the `spawn=` line **always embeds the FULL model ID even under `--alias`** — CLI dispatch never aliases (aliasing is the Agent-tool-only adaptation; an external CLI's `--model` flag takes a full ID). `_cli-fab.md` § fab resolve-agent is the CLI reference. Consuming the `spawn=` line into an actual cross-harness dispatch is 3c/3d's job — this change only *emits* it; the dispatch-seam skills that inject model/effort do not read `spawn=`.
- **Empty model** ⇒ omit the dispatch `model` param entirely (inherit the orchestrator/session model — today's behavior). **Empty effort** ⇒ omit the effort instruction (see § The two halves dispatch through two seams below).
- The resolver maps `<stage>` → its fixed fab-owned tier → a `{model, effort}` profile (the `agent.tiers` project override per-field-merged over fab-kit's default — see [configuration.md](/_shared/configuration.md) § `agent`). The stage→tier mapping is NOT user-overridable; `agent.tiers` (tier redefinition) is the sole override surface. Full design: [`docs/specs/stage-models.md`](../../specs/stage-models.md).

**The two halves dispatch through two seams (model param + prompt instruction — m3d4).** The resolved profile has a model half and an effort half, and Claude Code consumes them through *different* seams: the **model** rides the Agent tool's `model` parameter (empty ⇒ omit/inherit); the **effort** is injected as an explicit imperative line in the dispatched subagent prompt — ``Operate at `<effort>` reasoning effort for this task.`` — because the Agent tool has **no `effort` parameter**. Empty effort ⇒ omit the instruction (mirroring the empty-model omit rule). The effort-via-prompt seam is imperfect (it relies on the sub-agent honoring the instruction rather than the harness enforcing it) but is the only per-sub-agent effort seam available today; a first-class per-sub-agent `effort` param on the Agent tool would close it cleanly and is the **residual harness ask** ([`docs/specs/stage-models.md`](../../specs/stage-models.md) § Skill wiring + `docs/findings/per-stage-model-tier-application.md` § Suggested directions item 4 — out of fab's control, not built).

**Compliance visibility (m3d4).** Each dispatch site MUST **surface the resolved `model=/effort=` lines** — carry them into the dispatch prompt (the effort line is already there per the seam above; co-locate the model line) and/or echo them in the orchestrator's own step output — so a *skipped* `fab resolve-agent` call (the sub-agent silently inherits the session profile) or a mis-resolved tier is **visible rather than silent**. There is no code-level guard fab can install (dispatch is harness-internal), so visibility is the available seam; the canonical contract also notes an all-empty resolution (both `model=` and `effort=` empty) is itself worth surfacing/asserting rather than dispatching blind.

**Harness-adapter boundary (Claude Code).** The resolution (stage→tier→`{model, effort}`) is **provider-neutral**; injecting it into the actual dispatch is **harness-specific**. For Claude Code the model adapter is the **Agent tool's `model` parameter** and the effort adapter is the subagent-prompt instruction above. One concrete harness detail: the Agent tool's `model` param is a **hard enum of short aliases** (`opus`/`sonnet`/`haiku`/`fable`), **not** the full versioned id (`claude-opus-4-8`) that the plain `fab resolve-agent` emits — so for Agent-tool dispatch the model half is resolved with **`fab resolve-agent <stage> --alias`** (yky7), which emits the Agent-tool-valid short alias directly on the `model=` line (a deterministic Go-side translation: prefix-matched so dated variants resolve, empty ⇒ empty inherit-signal, a non-Claude override passes through verbatim). This **supersedes the earlier prompt-side hand-mapping** (where the orchestrator was told to map the resolved id → alias by hand at the dispatch seam — m3d4's instruction, which was brittle and easy to fumble); no agent ever hand-maps now. Named explicitly as the Claude-Code adapter, not as universal truth; the coupling is not new (fab's entire subagent-dispatch design is already Claude-Code-shaped), so per-stage selection is exactly as portable as fab's existing dispatch. *(The operator launcher path is the deliberate exception — for a non-templated Claude `spawn_command` it appends `--model <full-id>` to a `claude` CLI invocation, which accepts full IDs, so it resolves WITHOUT `--alias`. Since 6tmi `spawn.WithProfile` appends only when the `spawn_command` carries no `{model}`/`{effort}` placeholder; a **templated** `spawn_command` is instead resolved by substituting the resolved profile into the placeholders — see [configuration.md](/_shared/configuration.md) § `agent` `spawn_command` placeholders and [runtime/operator.md](/runtime/operator.md).)*

**Review resolves once.** The `review` stage spawns two reviewer sub-agents (inward + outward) plus a merge. The dispatcher resolves `fab resolve-agent review` **once** and applies the same model AND the same effort-prompt instruction to all three — the mechanical merge runs at the reviewer's tier, an accepted stage-granularity tradeoff.

**Per-stage selection applies on every post-intake stage (fgxx).** Per-stage selection is a property of dispatched sub-agent runs — and since fgxx collapsed the post-intake dual execution mode, **every post-intake stage now dispatches a sub-agent**, including plain `/fab-continue` (a one-stage sequencer that resolves `fab resolve-agent <stage>` and dispatches its stage's block). So there is no post-intake foreground execution path left to be the exception; `fab resolve-agent` applies uniformly across apply/review/hydrate regardless of caller. Intake is pre-boundary — it runs in the main session and is not tiered. The only residual advisory case is a stage skill genuinely run with **no dispatch at all**: fab cannot switch the session model mid-run, so such a skill MAY note "this stage is configured for X; you're on Y" but MUST NOT attempt to switch. (The effort half of the tier is now injected via the subagent-prompt instruction — see § The two halves dispatch through two seams above (m3d4) — so it is no longer a deferred follow-up; the lone remaining residual is a first-class per-sub-agent `effort` param on the Agent tool, a harness ask outside fab's control.)

This subsection documents *where the resolution call sits* and *how the profile is consumed* (a dispatch-seam concern, parallel to Standard Subagent Context above). The config schema (`agent.tiers`, the tiers, the fixed mapping) and the design rationale (no-validation, fixed-mapping-vs-budget) live in [configuration.md](/_shared/configuration.md).

### SRAD Protocol (via the `_srad` Helper)

The SRAD autonomy framework lives in the dedicated `_srad.md` helper (extracted from `_preamble.md` in 260611-zc9m), declared via frontmatter `helpers:` by the six planning skills — `fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`. It is **no longer part of the always-load layer**: `_preamble.md` retains only a ~3-line pointer (§ SRAD Autonomy Framework (pointer)), so non-planning skills do not pay for the framework. The framework defines:
- **SRAD scoring table** — four dimensions evaluated on a continuous 0–100 scale per decision point
- **Fuzzy-to-grade mapping** — composite score via weighted mean (w_S=0.20, w_R=0.30, w_A=0.30, w_D=0.20 — R and A up-weighted in 4yi8 from 0.25 each), mapped to **indicative-only** grades via half-open bands: composite ≥ 80 Certain, 50 ≤ c < 80 Confident, 20 ≤ c < 50 Tentative, else Unresolved (4yi8 — bands shifted from the c5tr 85/60/30 set to align with the demerit penalty-curve knees; the grade is derived from the composite and never read by the score formula)
- **No Critical Rule, no hard-fail (4yi8)** — the former "R < 25 AND A < 25 forces Unresolved" override and the "any Unresolved row → 0.0" short-circuit are removed; blocking is emergent from the demerit penalty curve (a `composite < 20` row penalizes ≥ 2.0), and reversibility is carried by R's 0.30 weight rather than a separate rule
- **Confidence grades** — Certain, Confident, Tentative, Unresolved with corresponding artifact markers
- **Worked examples** — Example 1 compressed to the one-liner style of Examples 2/3 during the extraction; semantics preserved
- **Artifact markers** — `<!-- assumed: ... -->` for Tentative, `<!-- clarified: ... -->` for resolved assumptions
- **Assumptions Summary Block** — standard format with required `Scores` column for per-dimension data; all four grades (Certain, Confident, Tentative, Unresolved) recorded

The companion confidence-scoring internals — the `.status.yaml` `confidence:` schema, the score formula (the 4yi8 demerit model: `score = clamp(5.0 − Σ penalty(composite), 0, 5)`, no coverage factor and no `expected_min` in the score path), and the status-template notes — live in `_cli-fab.md` § fab score (extended), also moved out of the preamble by 260611-zc9m. Agents never compute the score: `fab score` (Go) does, reading `intake.md` as the sole scoring source. `_preamble.md` § Confidence Scoring keeps only the **Gate Threshold** (single flat-3.0 intake gate via `fab score --check-gate --stage intake`) and **Invocation** (who scores, when — the invoker list completed in d9rs: `/fab-new` **and `/fab-draft`** persist the intake score after generation; `/fab-clarify` re-persists it in **both modes** — Suggest Step 7 and Auto Mode step 4 — not just suggest mode). The preamble's former Bulk Confirm subsection is likewise a one-sentence pointer — `fab-clarify.md` (Step 2, Suggest Mode) is the sole authority for the trigger and semantics (see [pipeline/clarify.md](/pipeline/clarify.md)).

### Next Steps Convention (State Table, Scoped MUST)

The `_preamble.md` preamble defines a **state-keyed Next Steps Convention** that skills use to derive their `Next:` output lines. The MUST is **scoped** (260611-zc9m): it applies **unless the skill's own Output or Key Properties section defines a different ending** — the skill file wins, mirroring the §1 context-loading contract. The exemption basis is a skill-file-declared ending, not a "pipeline-state skill" classification (which contradicted its own examples — `/git-pr` advances ship and `/git-pr-review` runs review-pr transitions, yet both declare their own completion output; `/fab-discuss`'s ready signal and `/fab-operator`'s status frame are the other current exemptions). The convention includes:

1. **State Table** — 10 states (none, initialized, intake, apply, review pass, review fail, hydrate, ship, review-pr pass, review-pr fail) each mapping to available commands and a default
2. **State derivation rules** — how to determine the current state from `config.yaml` existence, `.fab-status.yaml`, and `.status.yaml` progress map
3. **Lookup procedure** — determine state, look up in table, output default first
4. **Activation preamble** — when a skill creates/restores a change without activating it (`/fab-draft` always, `/fab-archive restore` without `--switch`), the `Next:` line includes a `/fab-switch {name}` instruction before state-derived commands (`/fab-new` auto-activates and does not need it)

No skill duplicates or maintains its own suggestion logic — skills on the default path derive from this single canonical table.

### Exception Skills

The exception set is **declared by the skill files themselves** (the preamble never enumerates it — d9rs). The shipped overrides as of d9rs, per each skill's own `## Context Loading` section (or header context note):

- `/fab-setup` — bootstraps structure, doesn't need project memory
- `/fab-switch` — navigation only (requires no always-load files; the former preamble claim that it "loads only config.yaml" was dropped in 260611-zc9m — the skill file wins)
- `/fab-status` — read-only status display, minimal context
- `/docs-hydrate-memory` — ingests/generates memory content, doesn't pre-load the landscape (explicit `## Context Loading` override section added in d9rs)
- `/fab-help` — uses no context at all
- `/fab-archive` — none beyond preflight (`fab change archive` reads `intake.md` and the backlog itself)
- `/docs-hydrate-specs`, `/docs-reorg-memory`, `/docs-reorg-specs` — load their own doc-tree working sets (memory/spec indexes + files); no config, constitution, or active change
- `/fab-proceed` — skips preflight/context loading itself, delegating all pipeline context loading to `/fab-fff` (header context note)

**Partial exception**: `/fab-operator` loads only `config.yaml`, `constitution.md`, and `context.md` (260611-zc9m — code-quality, code-review, and both doc indexes serve artifact generation/review, which the operator never does, and a long-lived session re-pays every loaded file after each `/clear`). See [runtime/operator.md](/runtime/operator.md).

**Special case**: `/fab-discuss` is *not* an exception — it loads the full 7-file always-load layer. However, it is the only skill whose entire purpose is to surface that layer. Other skills load the always-load layer as a preamble to generating or validating artifacts; `fab-discuss` loads it as its primary output, presenting an orientation summary for exploratory discussion sessions. It does not run preflight, does not require an active change, and does not advance any stage. Since 260611-zc9m its skill file no longer restates the 7-file list — it points at `_preamble.md` §1, keeping only its do-not-run-preflight / no-change-artifacts deltas (`fab-proceed.md`'s Standard Subagent Context list was replaced with a preamble pointer in the same change).

## Design Decisions

### `--alias` Flag — Deterministic Go-Side id→alias Translation at the Agent-Tool Seam
**Decision**: The Agent-tool model half is resolved with `fab resolve-agent <stage> --alias`, a boolean flag that emits the Claude-Code short alias (`opus`/`sonnet`/`haiku`/`fable`) directly on the `model=` line. The two Claude-Code surfaces deliberately diverge: the `claude` **CLI** `--model` flag (operator launcher / `spawn_command`) accepts full IDs **and** aliases, so it keeps resolving WITHOUT `--alias` (full ID); the **Agent tool's `model` param** is a hard JSON-schema enum (`["sonnet","opus","haiku","fable"]`) that rejects full IDs, so orchestrator/sequencer sub-agent dispatch resolves WITH `--alias`. The mapping lives in `agent.ModelAlias` (`internal/agent`, alongside the tier tables + `Resolve`): prefix-matched (`claude-opus-` → `opus`, etc.) so dated variants (`claude-haiku-4-5-20251001` → `haiku`) resolve; empty → empty (preserves the inherit signal); an unmapped/non-Claude ID (`gpt-5`) → verbatim pass-through (adapter, not validator — preserves provider-neutrality). Applied in `resolveAgentCmd`'s RunE as a one-line pre-format mutation of `profile.Model`; `formatAgentProfile` stays a pure, byte-stable formatter. Default (no flag) is byte-identical to today (full ID); `--alias` touches only the `model=` line, never `effort=`.
**Why**: This **supersedes PR #413's (m3d4) prompt-side hand-mapping** — the prose instruction "the orchestrator maps the resolved id → alias at the dispatch seam," which told every dispatching agent to translate the id by hand on each dispatch. The live failure this fixes *was* an agent fumbling exactly that hand-map (it passed `claude-opus-4-8` verbatim into the Agent tool's `model` param, hitting "Invalid tool parameters"). Encoding the map in Go at the harness-adapter boundary `stage-models.md` already names makes the failure mode impossible to reproduce. Tier defaults were deliberately NOT switched to aliases (rejected — see below), so the provider-neutral full-ID default and the drift-guarded spec tables stay untouched.
**Rejected**: (a) Keeping the #413 prompt-only hand-mapping (brittle; the failure mode that prompted this change). (b) Switching `defaultTiers` / the two drift-guarded `stage-models.md` tables to aliases (breaks provider-neutrality, weakens the Fable version-pin discipline, and forces a coordinated edit across the Go map + spec tables + config comments + migration — pushing a harness quirk into the provider-neutral core; `TestDocTablesMatchAgentMaps` stays unaffected by keeping full IDs canonical). (c) Threading a bool into `formatAgentProfile` (couples the formatter to a flag it doesn't need — the empty-model branch already does the right thing because `ModelAlias("")` → `""`). (d) Making `--alias` a Claude-only validator that errors on non-Claude IDs (would break the provider-neutral pass-through).
*Introduced by*: 260613-yky7-resolve-agent-alias-flag

### Preamble Context Diet — Consumer-Specific Content Moves to Opt-In Homes
**Decision**: Content in the always-loaded `_preamble.md` that serves only a subset of skills (or no live skill) is relocated to opt-in homes, with short pointers left behind: the SRAD framework → new `_srad.md` helper (declared by the 6 planning skills); confidence-scoring schema/formula/template → `_cli-fab.md` § fab score (extended) (preamble keeps Gate Threshold + Invocation); Bulk Confirm → one-sentence pointer (`fab-clarify.md` Step 2 is sole authority); the dormant `[AUTO-MODE]` Skill Invocation Protocol → `fab-clarify.md` (its sole referencer; Auto Mode retained — user decision: move, not delete); Operator Spawning Rules → `_cli-external.md` wt section (one repo-targeting rule, duplicate dropped). The §1 always-load contract and the Next:-line MUST become **descriptive with a skill-file-wins override**, and the helper model gains **stage-conditional in-body loading** (used by `/fab-continue` for `_generation`/`_review`). Preamble: 32,790 → 22,313 B (−32.0%); every non-planning skill saves the full ~10.5KB per invocation; relocated content is paid only by its consumers.
**Why**: The preamble was 2–26x the body of the skill being run and roughly a third of it served a small subset of skills. Duplicated copies (bulk-confirm trigger, spawning rules, restated context lists in fab-proceed/fab-discuss) had already drifted once. The existing `helpers:` mechanism plus fab-kit's `listSkills` auto-deploy (`internal/skills.go`; lived in `sync.go` until the 260612-tb6f split) meant the reduction needed zero Go changes and zero semantic loss — content moves, it doesn't disappear.
**Rejected**: Deleting the dormant `[AUTO-MODE]`/Auto-Mode pair (user chose move-over-delete — preserves behavior). Prose compression alone (saves far less, leaves the wrong-audience placement problem). An explicit exempt-skill list for the Next:-line MUST (goes stale with every new skill; a skill-file-declared-ending basis is self-maintaining — and the "pipeline-state skill" basis contradicted its own examples, since `/git-pr`/`/git-pr-review` do advance pipeline state).
*Introduced by*: 260611-zc9m-preamble-context-diet

### External Sub-Domain Addressing (Up-to-3-Hop Selective Load)
**Decision**: When an over-wide domain is split into sub-domains, the sub-domain file is addressed **externally** — the Affected Memory contract, the always-load layer, and selective loading all gain a `{domain}/{sub-domain}/{file}` form. Selective domain loading becomes an up-to-3-hop walk: domain index → (only if the entry is sub-domained) sub-domain index → file. A flat domain stays the degenerate 2-hop case (no sub-domain index hop, byte-identical to pre-change behavior).
**Why**: External addressing makes sub-domains first-class and navigable — there is no "find-the-file-anywhere-under-the-domain" resolver ambiguity (the failure mode of the Internal/duplicate-truth-file alternative). loom's historical External-style index churn was the *hand-edited-index* problem, which `tciy` already eliminated (sub-domain indexes are generated by `fab memory-index` too), so External's only historical downside is moot post-`tciy`; its upside (explicit, navigable addressing) stands.
**Rejected**: Internal addressing (sub-domain files resolved by search under the domain) — re-introduces resolver ambiguity and a duplicate-truth-file failure mode. A flat-only model (never sub-dividing) — leaves over-wide domains (e.g. `fab-workflow` at 20 files > ~12) with no structural escape valve.
*Introduced by*: 260607-sx7a-reorg-memory-shape-rebalance

### Smart Loading for All Skills on Active Changes
**Decision**: Expanded "Memory Lookup" from spec-writing-only to all skills operating on an active change.
**Why**: Agents need domain awareness for planning, implementation, and review — not just spec writing.
**Rejected**: Per-skill opt-in — too much maintenance overhead and easy to miss new skills.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Always Load docs/specs/index.md
**Decision**: Added `docs/specs/index.md` to the "Always Load" layer as a 4th baseline file.
**Why**: Gives every skill awareness of the specifications landscape (pre-implementation design intent) alongside the documentation landscape. The index is lightweight and human-curated, so context cost is minimal.
**Rejected**: Loading design index only when relevant — same inconsistency risk as with memory/index.md.
*Introduced by*: 260207-bb1q-add-specs-index

### Always Load docs/memory/index.md
**Decision**: Added `docs/memory/index.md` to the "Always Load" layer alongside config.yaml and constitution.md.
**Why**: Gives every skill baseline awareness of the documentation landscape. The index is lightweight (a table of domains), so the context cost is minimal.
**Rejected**: Loading only when needed — would require each skill to independently decide, leading to inconsistency.
*Introduced by*: 260207-q7m3-separate-hydrate-smart-context

### Always-Load `_cli-rk` Skill for rk Capabilities *(Superseded)*
**Decision**: Added `_cli-rk.md` as an optional always-load skill in `_preamble.md`, separate from `_cli-external.md`.
**Why**: rk iframe+proxy capabilities benefit every fab session (visual display of diagrams, plans, slide decks), not just operator sessions. Centralizing the visual display recipe in `_cli-rk.md` (rather than baking it into visual-explainer) gives any skill the superpower via separation of concerns.
**Rejected**: Adding rk to `_cli-external.md` and promoting to always-load — would bloat every session with operator-specific content (wt, idea, /loop). Also rejected decentralized approach (iframe logic in visual-explainer only) — forces other skills to duplicate logic or use visual-explainer as a middleman.
*Introduced by*: 260416-mgsm-add-cli-rk-skill
*Superseded by*: 260418-or0o-flatten-skill-helpers — `_cli-rk.md` content inlined into `_preamble.md` § Run-Kit (rk) Reference; the separate helper file is deleted. The silent-fail-when-rk-missing design is preserved verbatim in the inlined subsection.

### Flatten Helper Include Tree
**Decision**: Collapse the helper always-load set from `{_preamble, _cli-fab, _naming, _cli-rk}` to `{_preamble}` only. Inline `_naming` and `_cli-rk` into `_preamble`. Add a new per-skill `helpers:` frontmatter field listing the additional helpers each skill needs (`_generation`, `_review`, `_cli-fab`, `_cli-external`). Inline the 6 most-used `fab` commands into `_preamble` § Common fab Commands. Compress `_cli-fab` from 773 lines to ≤300.
**Why**: Two root causes. (1) Universal "also read" fanout from `_preamble` shipped ~1324 lines of helper content that 15 of 24 skills didn't use. (2) Agents silently skipped 2nd-layer "also read" directives — pointer-based loading was non-deterministic. Replacing the fanout with explicit, frontmatter-declared helpers is auditable, grep-able, and reliable (agents read frontmatter before body). Inlining the smallest helpers and the commonest commands eliminates the 2nd layer for most skills entirely.
**Rejected**: (a) Splitting `_preamble` further — deepens the tree, worsens skip-rate. (b) Relying on prompt caching — doesn't fix correctness when pointers are silently skipped. (c) Full inline of `_cli-fab` — adds ~500 lines to universal load. (d) Renaming `_`-prefix to visible names (backlog `[84bh]`) — addresses visibility but not fanout; structural fix supersedes it.
*Introduced by*: 260418-or0o-flatten-skill-helpers

### Standard Subagent Context as Centralized Template
**Decision**: Added a Standard Subagent Context subsection to `_preamble.md` § Subagent Dispatch, listing the 5 `fab/project/**` files that every subagent must read. Skills reference this template instead of maintaining ad-hoc file lists.
**Why**: Each skill that dispatched subagents maintained its own context list, creating silent quality gaps (forgotten files) and drift risk (new files not propagated). Centralizing in `_preamble.md` ensures all subagents — at any nesting depth — inherit project principles automatically.
**Rejected**: Including `docs/memory/index.md` and `docs/specs/index.md` in subagent context — these are navigation aids for the parent agent, not project principles needed by subagents.
*Introduced by*: 260318-dzze-standard-subagent-context
