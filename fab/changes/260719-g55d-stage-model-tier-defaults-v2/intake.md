# Intake: Stage-Model Tier Defaults v2

**Change**: 260719-g55d-stage-model-tier-defaults-v2
**Created**: 2026-07-20

## Origin

Promptless dispatch via `/fab-proceed` (create-intake subagent, `{questioning-mode} = promptless-defer`). The input is a change description synthesized from a design conversation with the user (fab-kit's owner); all decisions below were settled in that conversation and are captured faithfully — treat them as decided, not proposed.

> Rework fab-kit's built-in stage-model tier defaults — split `hydrate` out of `doing`, ship a new Fable-era default profile curve, and close the two untiered dispatch seams (fab-proceed prefix steps, fab-continue ship/review-pr rows).

**Corrections (2026-07-20, mid-pipeline, user-issued)**: the originally-planned `fast` → `ship` tier rename was **cancelled** (with it: the carry-forward migration and the `renamed_from` question — both void); two items were **added to scope**: per-step tier resolution for `/fab-proceed`'s prefix steps (§ 6) and tier resolution for `/fab-continue`'s ship/review-pr rows (§ 7). Everything else stands as originally written. This intake reflects the corrected design.

Related backlog item: `[xz4f]` ("The current breakup of tiers isnt working. I need access to hydrate step also separately...") overlaps this change's motivation. This change was created from the fuller synthesized design (fresh NL change, not claiming the backlog ID); xz4f's config-reset portion was already handled separately (fab-kit's own config.yaml now inherits defaults — done outside this change). The xz4f tier-granularity portion is superseded by this change.

## Why

The current 5-tier taxonomy (`default`/`operator`/`doing`/`review`/`fast`) groups apply, review-pr, and hydrate under one `doing` tier, so **hydrate cannot run cheaper (or on a different model) than apply**. The user wants per-stage granularity driven by three goals: (1) speed, (2) reducing wasted effort, (3) reducing translation layers. Two dispatch seams also run untiered today — `/fab-proceed`'s prefix-step subagents and `/fab-continue`'s ship/review-pr delegations — breaking the invariant that a stage resolves the same tier regardless of caller.

If not fixed: every non-overriding project keeps paying apply-grade cost (Opus/xhigh today) for hydrate's memory-writing work, and untiered dispatches silently inherit whatever model the session happens to run.

This change reworks the **KIT DEFAULTS** (the Go maps in `internal/agent`) plus the skill-side dispatch wiring, not project-local overrides — the spec (`docs/specs/stage-models.md`) already reserves taxonomy evolution to fab-kit ("Disagreement with the tiering is an upstream fab-kit issue, not a project knob"), and this change IS that upstream fix. fab-kit's own `fab/project/config.yaml` was already cleaned to inherit defaults (done separately, NOT part of this change; verified — it carries no live `agent:` block, only the fence).

## What Changes

### 1. New `hydrate` tier — NO renames (canonical: `src/go/fab/internal/agent/agent.go`)

- New tier `hydrate` (new constant, value `"hydrate"`).
- **No tier renames.** `fast` keeps its name (`TierFast`, value `"fast"`); a `fast` → `ship` rename was considered and **cancelled** — no `TierShip` anywhere.
- Resulting tier set (six): `default`, `operator`, `doing`, `review`, `hydrate`, `fast`.
- **Naming rationale**: a tier is stage-named only where it maps 1:1 to a **single referent** (`review`, `hydrate`). `fast` deliberately keeps its role name because it is **multi-referent** — it governs the ship stage AND the `/fab-proceed` prefix-step dispatches (`/fab-switch`, `/git-branch` — see § 6). `default` and `doing` likewise keep role names (multi-referent).
- `IsTierName`, `TierNames`, `StageNames`, `ModelAlias`, and all alias handling: mechanics untouched (they read the maps; only map contents/constants change).

### 2. Taxonomy split — stage `hydrate` moves out of tier `doing`

New FIXED stage→tier mapping (`stageTiers` map — stays fab-owned and non-overridable; NO `stage_tiers:` config and NO per-stage escape hatch is being added). **The hydrate row is the only mapping change**:

| Stage | Tier (new) | Tier (old) |
|-------|-----------|------------|
| `intake` | `default` (advisory, foreground) | `default` (unchanged) |
| `apply` | `doing` | `doing` (unchanged) |
| `review` | `review` | `review` (unchanged) |
| `hydrate` | `hydrate` | `doing` (**split out**) |
| `ship` | `fast` | `fast` (unchanged) |
| `review-pr` | `doing` | `doing` (unchanged) |

### 3. New `defaultTiers` profiles (the shipped kit defaults)

| Tier | New profile | Old profile | Rationale (from the design conversation) |
|------|-------------|-------------|------------------------------------------|
| `default` | claude / `claude-fable-5` / `high` | fable-5 / xhigh | Effort lowered — interactive sessions want the quicker working style; Anthropic guidance: high is the default sweet spot, Fable at lower efforts still exceeds prior models' xhigh |
| `doing` | claude / `claude-fable-5` / `xhigh` | opus-4-8 / xhigh | xhigh is Anthropic's stated best setting for coding/agentic work; a strong author minimizes rework cycles per the apply↔review coupling argument (stage-models.md § apply↔review coupling) |
| `review` | claude / `claude-opus-4-8` / `xhigh` | fable-5 / xhigh | Deliberate cross-model author/critic diversity — a different model family avoids the author's blind spots; code review is a named Opus 4.8 strength |
| `hydrate` (new) | claude / `claude-opus-4-8` / `high` | — (was in doing: opus/xhigh) | Knowledge work and memory writing are named Opus 4.8 strengths; high = recommended default for intelligence-sensitive-but-not-hardest work |
| `fast` | claude / `claude-sonnet-5` / `medium` | sonnet-5 / low | Effort raised from low — margin for faithful PR-description comprehension (the reason Haiku was excluded from the defaults) |
| `operator` | claude / `claude-sonnet-5` / `medium` | sonnet-5 / medium | Unchanged |

The `defaultTiers` map remains "the ONE place bumped when a new top model lands" (the Fable upgrade path); this change is that bump plus the taxonomy evolution.

**Byte-exact resolution expectations** (for `resolve_agent_test.go`): stage `ship` resolves via tier `fast` → `model=claude-sonnet-5` / `effort=medium` / `provider=claude`; `hydrate` → opus-4-8/high; `apply`/`review-pr` via `doing` → fable-5/xhigh; `review` → opus-4-8/xhigh; `intake` via `default` → fable-5/high.

### 4. Stage/tier name-collision rule (replaces the false "disjoint" claim)

The "the two name sets are disjoint" claim is **already false today**: stage `review` maps to tier `review`. It works because `resolveStageOrTier` in `src/go/fab/cmd/fab/resolve_agent.go` checks tier names FIRST (`agent.IsTierName(name)` → `ResolveTier`, else stage `Resolve`), and the collision is an identity mapping (stage review → tier review), so either interpretation resolves identically. `hydrate` adds one more identity collision. **The collision set is `{review, hydrate}` only** — `ship` never becomes a tier name.

- **Replace the "disjoint" language everywhere** with the actual rule: *a tier may share a stage's name only when that stage maps to that same-named tier* (every stage-name/tier-name collision must be a fixed point: `stageTiers[name] == name`).
- Known "disjoint" sites to sweep (grep-verified): `src/go/fab/internal/agent/agent.go` (`IsTierName` doc comment), `src/go/fab/internal/agent/agent_test.go:196` (TestIsTierName comment), `src/go/fab/cmd/fab/resolve_agent.go` (two doc comments, lines ~16 and ~93), `src/go/fab/cmd/fab/resolve_agent_test.go:67` (comment), `src/kit/skills/_cli-fab.md:270` (+ its SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md`, which also says "disjoint from stage names"), `docs/specs/stage-models.md:210`, `docs/memory/runtime/providers-and-tiers.md` (lines ~91, ~123 — memory: hydrate's job, listed in Affected Memory). Re-grep `disjoint` at apply time; ignore unrelated hits (`_pipeline.md` decision heuristics, findings archives).
- **New drift-guard test**: next to `TestDocTablesMatchAgentMaps` in the agent package (`src/go/fab/internal/agent/`), assert every stage-name/tier-name collision is a fixed point (`for each name in both sets: stageTiers[name] == name`). This guards the tier-first check order in `resolveStageOrTier` from ever silently changing a stage's resolution.
- `TestIsTierName`: `"hydrate"` joins the tier-name list; `"ship"` **stays in the not-a-tier list**.
- Behavior of `resolveStageOrTier` itself: **unchanged** (tier-first order retained; identity collisions make the order immaterial for resolution results).

### 5. Config-reference fence + registry (`src/go/fab/internal/configref/configref.go`)

The fence text is generated from the same Go constants (`agent.DefaultTier` over `agent.TierNames()`, `tierStages` map), so it updates mechanically:

- The `tierStages` reference-prose map in configref.go gains the `hydrate` entry and drops hydrate from doing's stage list. `fast`'s entry gains its prefix-step referent (see § 6 — the fence's tier-reference prose lists each tier's referents).
- The FIXED stage→tier mapping comment block and built-in default profiles block in the rendered reference both change (verify rendered output; `fab/project/config.yaml`'s fence regenerates on the next `fab config upgrade` — the advertised defaults update automatically for every project).
- **No key rename anywhere in the registry or fence** — `renamed_from` stays `""` on every row; no migration file ships with this change.

**Upgrade note (documented in stage-models.md, NOT a migration)**: a project carrying an `agent.tiers.doing` override previously governed hydrate through it; after the split its hydrate stage resolves the new `hydrate` kit default (opus-4-8/high) unless it adds a `hydrate:` override. Since no config key changes meaning or goes inert, this is an upgrade note, not user-data restructuring — no migration file.

### 6. `/fab-proceed` prefix steps get tier resolution (skill wiring only, no Go change)

`src/kit/skills/fab-proceed.md` currently exempts prefix steps from model resolution ("the prefix steps are NOT pipeline stages, so they take **no** `fab resolve-agent` resolution — they dispatch at the inherited model"). Replace that exemption with per-step tier resolution:

- The `/fab-switch` and `/git-branch` prefix-step dispatches resolve the **`fast`** tier: run `fab resolve-agent fast --alias` before dispatching each subagent.
- The `_intake` create-intake prefix-step dispatch resolves the **`default`** tier the same way (`fab resolve-agent default --alias`) — this closes the one intake path that is a dispatched subagent yet ran untiered. Intake itself remains advisory-only on the foreground `/fab-new` path, which no resolution can govern.
- Both use **tier-name** resolution — the resolver already accepts tier names positionally (the same path `fab agent <tier>` uses), so no Go change; this is skill wiring only.
- Surface the resolved `model=`/`effort=` lines per the standard compliance-visibility rule and dispatch through the two seams (model on the Agent tool's `model` param, effort as the imperative prompt instruction; empty ⇒ omit).
- Update: `src/kit/skills/fab-proceed.md` (the per-stage-model note and all three prefix-step dispatch procedures), its SPEC mirror `docs/specs/skills/SPEC-fab-proceed.md`, and every place that lists tier referents — the stage-models.md tier table, the configref fence's tier-reference prose, the glossary role-tier entry, and the Affected Memory files.

### 7. `/fab-continue`'s ship and review-pr rows get tier resolution (eliminates the caller asymmetry)

Today plain `/fab-continue` delegates ship and review-pr to `/git-pr` / `/git-pr-review` with **no** tier resolution (its Dispatch-shorthand note says no `fab resolve-agent`/dispatch-adapter branch applies to them), while `/fab-fff` Steps 4–5 DO resolve ship and review-pr before dispatching the same skills.

**The invariant after this change: a stage resolves the same tier regardless of which caller drives it** (`/fab-continue`, `/fab-ff`, `/fab-fff`, `/fab-proceed`).

- Fix: in `src/kit/skills/fab-continue.md`, the ship and review-pr rows resolve `fab resolve-agent ship --alias` / `fab resolve-agent review-pr --alias` before dispatching the `/git-pr` / `/git-pr-review` subagent, surfacing `model=`/`effort=` and applying the two seams — mirroring `/fab-fff` Steps 4–5's contract exactly.
- Everything else about those rows is unchanged: `/git-pr` and `/git-pr-review` continue to self-manage their own `fab status` transitions.
- Update the Dispatch-shorthand note accordingly, plus the SPEC mirror `docs/specs/skills/SPEC-fab-continue.md` and any memory file documenting fab-continue's dispatch behavior (hydrate's job; listed in Affected Memory).

### 8. Docs/specs/skills sweep (constitution constraints apply)

CLI change ⇒ `_cli-fab.md` + tests; skill file change ⇒ `docs/specs/skills/SPEC-*.md` mirrors; sweep the whole mirror class per code-quality.md § Sibling & Mirror Sweeps (grep old tier claims repo-wide, not just listed files):

- `docs/specs/stage-models.md` — both drift-guarded tables (§ Default tier profiles, § fixed stage→tier mapping), the five-role-tiers table (now six, `fast`'s referents now include the fab-proceed prefix steps), "Why these defaults" rationale rewritten for the new curve, the disjoint claim in § Resolution (step 1's tier-name list), § Haiku excluded (references the `fast` tier — name unchanged, effort now medium), § apply↔review coupling (currently says "doing (Opus/high)" — realign with fable/xhigh), § Fable upgrade path section updated for the new curve, § Config schema example comments, the **doing-override upgrade note** (§ 5 above).
- `src/kit/skills/_preamble.md` — § Always Load config.yaml description ("the five role tiers" → six), § Per-Stage Model Resolution tier references; its SPEC mirror `docs/specs/skills/SPEC-_preamble.md`.
- `src/kit/skills/fab-proceed.md` + `docs/specs/skills/SPEC-fab-proceed.md` — prefix-step tier resolution (§ 6).
- `src/kit/skills/fab-continue.md` + `docs/specs/skills/SPEC-fab-continue.md` — ship/review-pr tier resolution (§ 7).
- `src/kit/skills/_cli-fab.md` — § fab resolve-agent (five→six tier-name list, disjoint claim, fixed-mapping + defaults prose at lines ~270-272), § fab agent (line ~1044 five-tier list); SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md`.
- `docs/specs/config.md` (line ~198 five-tier list), `docs/specs/glossary.md` (Role tier entry, line ~74 — six tiers, `fast`'s multi-referent role), `docs/specs/architecture.md` (config example, lines ~249-261).
- Go tests updated: `agent_test.go` (`TestIsTierName` — `"hydrate"` joins the tier list, `"ship"` stays not-a-tier; new fixed-point collision test; `TestResolveTier` expectations), `stagemodels_doc_test.go` (parses the updated spec tables), `resolve_agent_test.go` (byte-exact output expectations per § 3), dispatch tests referencing tier names.
- **No migration file** (§ 5 — the rename was cancelled; nothing restructures user data).

### Explicitly out of scope (agreed follow-up, separate change)

**Sticky-apply** — reusing the named apply agent across apply↔review rework cycles within a fab-fff/ff bracket (fresh reviewer each cycle, fallback to fresh dispatch per Constitution III, native-Agent-tool-adapter-only). That is orchestration/skill wiring of a different kind from this change's dispatch-seam edits, and is deliberately NOT in this change.

### Alternatives rejected (from the design conversation)

- **`fast` → `ship` tier rename** — initially planned, then cancelled by user correction: `fast` is multi-referent after § 6 (ship stage + prefix steps), so a stage name would misname it; keeping `fast` also eliminates the carry-forward migration and `renamed_from` question entirely.
- **Six stage-named tiers / dissolving role tiers entirely** — rejected: `default`, `doing`, and `fast` are genuinely multi-referent, and role names carry the why.
- **User-overridable stage→tier mapping (`stage_tiers:` config) or per-stage `agent.stages:` escape hatch** — rejected: taxonomy stays fab-owned; this change IS the upstream taxonomy fix.
- **Grouping review-pr with hydrate in a "finishing" tier** — rejected: review-pr belongs with apply (responsive fixing by the author role); hydrate (memory writing) is the odd one out.
- **Keeping profiles as project-local overrides in fab-kit's own config** — rejected: these become the shipped kit defaults so every non-overriding project inherits the curve.

## Affected Memory

- `runtime/providers-and-tiers`: (modify) primary record — five→six role tiers, fixed stage→tier mapping table, default profiles, the "disjoint" claims (~lines 91, 123), `fast`'s new prefix-step referent, design-decision pointers
- `_shared/configuration`: (modify) `agent.tiers` registry/reference prose ("five role tiers" mentions, advertised defaults, tier-referent table), the authoritative tier design-decision record it hosts
- `_shared/context-loading`: (modify) Per-Stage Model Resolution dispatch-seam prose — tier names/counts, the fab-proceed prefix-step resolution, the fab-continue ship/review-pr resolution (caller-invariance)
- `distribution/kit-architecture`: (modify) `fab agent [tier]` "any of the five" tier-name list
- `pipeline/execution-skills`: (modify) fab-continue's ship/review-pr dispatch rows (tier resolution added) and fab-proceed's prefix-step dispatch (exemption replaced by per-step resolution)

(Hydrate sweeps the full class per code-quality.md § Sibling & Mirror Sweeps — this list is the grep-derived starting set, not a cap.)

## Impact

- **Go (fab binary)**: `src/go/fab/internal/agent/agent.go` (constants, `defaultTiers`, `stageTiers`), `src/go/fab/internal/agent/agent_test.go`, `src/go/fab/internal/agent/stagemodels_doc_test.go`, `src/go/fab/internal/configref/configref.go` (`tierStages`), `src/go/fab/cmd/fab/resolve_agent.go` (comments), `src/go/fab/cmd/fab/resolve_agent_test.go`, configref/config tests with byte-exact reference expectations, dispatch tests referencing tier names. Behavior surface: `fab resolve-agent`, `fab agent`, `fab config reference/upgrade`, operator launcher (reads operator tier — value unchanged).
- **Kit skills**: `src/kit/skills/_preamble.md`, `src/kit/skills/_cli-fab.md`, `src/kit/skills/fab-proceed.md`, `src/kit/skills/fab-continue.md` (+ SPEC mirrors under `docs/specs/skills/`).
- **Specs**: `docs/specs/stage-models.md` (drift-guarded tables + upgrade note), `docs/specs/config.md`, `docs/specs/glossary.md`, `docs/specs/architecture.md`.
- **Migrations**: none (rename cancelled; no user data restructured).
- **User-facing compatibility**: non-overriding projects inherit the new curve automatically on upgrade (fence regeneration updates advertised defaults). Projects with an `agent.tiers.doing` override: hydrate now resolves the `hydrate` kit default unless they add a `hydrate:` override (upgrade note in stage-models.md).
- Tests to run: `src/go/fab/internal/agent`, `src/go/fab/internal/configref`, `src/go/fab/cmd/fab` packages first; widen if cross-cutting.

## Open Questions

None — the carry-forward question is void (rename cancelled).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New `hydrate` tier; NO renames (`fast` keeps its name — the `ship` rename was cancelled by user correction); six-tier set `default`/`operator`/`doing`/`review`/`hydrate`/`fast` | User-issued correction — explicit and final | S:95 R:75 A:95 D:95 |
| 2 | Certain | Stage→tier split: hydrate→`hydrate` is the ONLY mapping change; ship stays on `fast`; mapping stays fab-owned and non-overridable | Discussed — upstream taxonomy evolution the spec already reserves to fab-kit | S:95 R:75 A:90 D:95 |
| 3 | Certain | New default profiles exactly as tabled (default fable/high, doing fable/xhigh, review opus/xhigh, hydrate opus/high, fast sonnet/medium, operator sonnet/medium) | Discussed — specific values agreed with per-tier rationale; one-map change, easy to re-bump | S:95 R:85 A:90 D:90 |
| 4 | Certain | Fixed-point collision rule + drift-guard test; collision set is `{review, hydrate}` only; `"ship"` stays in TestIsTierName's not-a-tier list | User correction pins the collision set; the claim is already false today (`review` collides) | S:90 R:85 A:95 D:90 |
| 5 | Certain | Sticky-apply is out of scope (separate follow-up change) | Discussed — explicitly excluded | S:95 R:90 A:95 D:95 |
| 6 | Certain | `resolveStageOrTier` tier-first check order, `IsTierName` mechanics, and alias handling stay untouched | Discussed — identity collisions make order immaterial; description says alias handling untouched | S:90 R:85 A:90 D:90 |
| 7 | Certain | No migration file and no `renamed_from` use — nothing restructures user data; the doing-override/hydrate residual ships as an upgrade note in stage-models.md | User correction: no key changes meaning or goes inert, so the migration convention does not trigger | S:90 R:80 A:90 D:90 |
| 8 | Confident | Created as a fresh NL change without claiming backlog ID `[xz4f]`; overlap recorded in Origin so the backlog item can be marked at archive time | Orchestrator supplied an NL description + slug (NL input creates a fresh change by procedure); xz4f's remaining scope is superseded, noted for traceability | S:70 R:90 A:70 D:70 |
| 9 | Certain | Affected Memory list is the grep-derived starting set (5 files); hydrate sweeps the full mirror class beyond it | code-quality.md § Sibling & Mirror Sweeps says per-file lists under-cover; list grounded by repo-wide grep | S:75 R:90 A:85 D:80 |
| 10 | Certain | `/fab-proceed` prefix steps resolve tiers per-step: `fast` for `/fab-switch` + `/git-branch`, `default` for the `_intake` create-intake dispatch; tier-name resolution, skill wiring only, no Go change | User-issued correction — explicit values per step; resolver already accepts tier names | S:95 R:85 A:90 D:95 |
| 11 | Certain | `/fab-continue`'s ship and review-pr rows resolve `ship`/`review-pr` before dispatching `/git-pr`//`git-pr-review`, mirroring fab-fff Steps 4–5; those skills keep self-managing their own status transitions; caller-invariance is the target invariant | User-issued correction — explicit contract named | S:95 R:85 A:90 D:95 |

11 assumptions (10 certain, 1 confident, 0 tentative, 0 unresolved).
