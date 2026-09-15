# Intake: Retire the Codex→Claude Review Cascade

**Change**: 260808-rvcs-retire-codex-claude-review-cascade
**Created**: 2026-08-09

## Origin

One-shot invocation: `/fab-new rvcs` — backlog entry `[rvcs]` (2026-08-09) from `fab/backlog.md`:

> Retire the Codex→Claude cascade in _review.md — outdated now that agent.profiles.review.provider (+ FAB_AGENT_PROFILES per-session, C1) gives reviewer choice a first-class home. The cascade predates the provider system: hardcoded 'command -v codex' shell-out inside the single review agent, its own parallel toggle surface (code-review.md § Review Tools codex/claude entries), bypasses fab resolve-agent/providers-table grammar/model+effort fills/dispatch modes, and composes badly (opus-at-high-effort worker babysits a ~10-min codex subprocess poll; review provider=codex would spawn codex-inside-codex). Evidence 2026-08-09 (fp02 review): Codex arm only CONFIRMED the Claude worker's top findings, discovered nothing new, +~15 min wall-clock. Cross-vendor eyes remain post-ship via Copilot at review-pr. SCOPE (skill-prose only, zero Go): (1) delete § Codex→Claude Cascade from src/kit/skills/_review.md + reword the 'reviewer diversity comes from the cascade' claims (diff-only/fab-adopt semantics unchanged: worker is the holistic reviewer, zero findings passes); (2) SPEC-_review.md mirror + whole mirror-class grep for cascade/Codex (skills.md, fab-adopt/fab-continue/_pipeline restatements); (3) code-review.md § Review Tools: codex/claude entries dead — copilot entry STAYS (git-pr-review reads it); sweep kit template/docs of the section; (4) hydrate: pipeline/execution-skills.md cascade paragraphs + superseding design-decision entry ('single review agent stays (pag2); diversity delegated to agent.profiles.review + post-ship Copilot'), _shared/configuration.md § review_tools pointer. OPTIONAL follow-up if second opinion is missed later: provider-resolved second-opinion knob run headless via the providers table — do NOT build it in this change.

The backlog entry is prescriptive: it names the mechanism to delete, the surviving toggle, the sweep class, the hydrate targets, and one explicit non-goal. Gap analysis (this session) confirmed the file inventory against the current tree — see What Changes for exact locations.

## Why

1. **The pain point**: `_review.md` § Codex→Claude Cascade is a pre-provider-system relic. It hardcodes a `command -v codex` shell-out inside the single review agent, carries its own parallel toggle surface (`code-review.md` § Review Tools `codex`/`claude` entries), and bypasses everything the provider system now owns: `fab resolve-agent`, the `providers:` table grammar, per-role model/effort fills, and the dispatch-mode ladder. Reviewer choice now has a first-class home — `agent.profiles.review.provider` (plus `FAB_AGENT_PROFILES` per-session from C1, PR #553) — so the cascade is a second, inferior mechanism for the same decision.

2. **The consequence of not fixing**: the cascade composes badly with the provider system it predates. An opus-at-high-effort review worker babysits a ~10-minute codex subprocess poll (wasted wall-clock at the most expensive tier); configuring `review: { provider: codex }` would spawn codex-inside-codex. Empirical evidence (2026-08-09, fp02 review): the Codex arm only CONFIRMED the Claude worker's top findings, discovered nothing new, and added ~15 minutes of wall-clock. The cascade costs real time and yields no marginal findings.

3. **Why deletion over alternatives**: cross-vendor eyes are preserved post-ship via Copilot at review-pr (`/git-pr-review`), and per-stage provider choice is preserved via `agent.profiles.review`. A provider-resolved "second opinion" knob (run headless via the providers table) was considered and explicitly deferred — do NOT build it in this change; it is an optional follow-up only if a second opinion is missed later.

## What Changes

Skill-prose only. Zero Go changes, zero Go test changes (verified: the only Go-side mention, a comment at `src/go/fab/cmd/fab/config_test.go:723` — "review_tools (retired to code-review.md § Review Tools)" — stays accurate because the section survives with the `copilot` entry).

### 1. `src/kit/skills/_review.md` — delete the cascade, reword diversity claims

- **Delete `### Codex→Claude Cascade`** (currently lines 110–121): the whole section — the cascade description, the Review Tools gating prose ("absent section/entry = enabled", `- codex: false` example, the copilot-is-git-pr-review-only note), and the 5-step check-config/attempt-Codex/check-config/attempt-Claude/graceful-no-op procedure.
- **Frontmatter `description:`** (line 3): drop "(Codex→Claude cascade with full repo access)" — the holistic-diff focus areas and full repo access remain, performed by the single review worker itself.
- **Findings & Verdict** (line 139): the zero-findings-passes example currently reads "e.g. all reviewer tools disabled via `code-review.md` § Review Tools, or unavailable". Reword so the rule survives without referencing the retired toggle — zero findings still passes best-effort in `diff-only` mode (adoption must not hard-block).
- **Semantics explicitly unchanged**: the single review worker (pag2) IS the holistic reviewer — it already judges the diff on its own merits with full repo access (§ Holistic-Diff Focus Areas stays verbatim); `mode: full | diff-only` gating is untouched; the pass/fail rule ("any must-fix → fail", zero findings passes) is untouched; `/fab-adopt`'s diff-only path is untouched.

### 2. SPEC mirror + whole mirror-class sweep for cascade/Codex claims

Per code-quality.md § Sibling & Mirror Sweeps, grep the class repo-wide for `cascade`/`Codex` (excluding the config four-layer cascade and the dispatch descent ladder, which are unrelated uses of the word). Known sites from gap analysis:

- `docs/specs/skills/SPEC-_review.md` — the constitution-required mirror: header description (lines 3/10), the mode paragraph (line 12: "the Codex→Claude cascade … identical for both modes"), the single-dispatch paragraph (line 14: "reviewer diversity is preserved by the Codex→Claude external-tool cascade, kept as a step inside the single agent" → reword: diversity delegated to `agent.profiles.review` + post-ship Copilot), the flow diagram (line 69: `└─ Codex→Claude Cascade [both modes]`), the Bash tool row (line 119: "the Codex→Claude cascade tools"), and the sub-agents note (line 123).
- `src/kit/skills/fab-continue.md:176` — drop "Codex→Claude cascade" from the merged-procedure parenthetical ("plan-conformance steps + holistic-diff focus areas + Codex→Claude cascade").
- `docs/specs/skills/SPEC-fab-continue.md` — flow diagram line 117 (`Codex→Claude cascade (graceful no-op)`) and the sub-agent table line 201 ("holistic full-repo diff review via Codex→Claude cascade").
- `docs/specs/skills.md:928` — the git-pr-review rollup's contrast sentence "There is no Codex/Claude cascade — Copilot is the only automated reviewer" reads as a contrast against a mechanism that will no longer exist anywhere; simplify (the Copilot-toggle clause stays). Also sweep skills.md's review-stage rollup for any cascade restatement.
- `src/kit/skills/_pipeline.md` and `src/kit/skills/fab-adopt.md` — gap analysis found no cascade restatements (their "cascade" hits are the status-transition cascade, unrelated); the sweep re-verifies during apply.

### 3. `code-review.md` § Review Tools — codex/claude entries retired, copilot stays

- `src/kit/scaffold/fab/project/code-review.md` § Review Tools (lines ~55–74): remove the `- codex / claude — the review-stage Codex → Claude cascade …` explanation bullet and the `- codex: false` / `- claude: false` lines from the commented example block. The section itself STAYS with the `copilot` entry (read by `/git-pr-review` Phase 2; `--tool copilot` force-overrides). Update the section's intro comment if it references the retired entries.
- This repo's own `fab/project/code-review.md` carries no § Review Tools section — nothing to edit there.
- No migration ships: after the deletion nothing reads `codex`/`claude` bullets, so stale entries in existing user projects' `code-review.md` are inert prose in an optional, user-owned file (see Assumptions #6).

### 4. Hydrate — memory updated to present truth

- `docs/memory/pipeline/execution-skills.md`: remove/reword the cascade paragraphs — the Review Mode paragraph's "cascade disabled via § Review Tools" zero-findings example (line ~148), the single-agent-dispatch paragraph's "Reviewer diversity comes from the Codex→Claude external-tool cascade" (line ~150), the Holistic-diff focus-areas bullet's full cascade description (line ~155), the git-pr-review Phase 2 paragraph's "the Codex→Claude cascade belongs to the single review agent in `_review.md`" pointer (line ~57), and the two Design Decisions entries whose Why/Rejected text leans on the cascade (lines ~417–418 Copilot-only Phase 2, ~429 Single Review Agent). Add a superseding design-decision entry: **single review agent stays (pag2); reviewer diversity delegated to `agent.profiles.review` + post-ship Copilot at review-pr**.
- `docs/memory/_shared/configuration.md` § `code-review.md` (line ~282): the § Review Tools bullet currently documents two tool sets (`codex`/`claude` for the cascade; `copilot` for git-pr-review). Rewrite to copilot-only.
- Historical records stay verbatim: `log.md`/`log.seed.md` prior entries, `src/kit/migrations/2.12.1-to-2.13.0.md`, `docs/specs/findings/skills-review-2026-06-11.md` (see Assumptions #5). Hydrate appends new log entries per the normal procedure.

### Non-Goals

- **No second-opinion knob.** The provider-resolved second-opinion follow-up (headless via the providers table) is explicitly out of scope — backlog directive: "do NOT build it in this change."
- **No Go changes**, no `_cli-fab.md` changes (no CLI surface touched), no migration file.
- **No change to review semantics**: single worker, mode gating, preconditions, findings tiers, pass/fail rule, and `{stage}-result.yaml` schema all byte-unchanged in meaning.

## Affected Memory

- `pipeline/execution-skills.md`: (modify) remove the Codex→Claude cascade paragraphs from the review-stage sections; supersede the diversity claims in the Single Review Agent and Copilot-only Phase 2 design decisions; add the superseding design-decision entry (diversity via `agent.profiles.review` + post-ship Copilot)
- `_shared/configuration.md`: (modify) § `code-review.md` Review Tools description — copilot-only (codex/claude entries retired)

## Impact

- **Kit skill prose**: `src/kit/skills/_review.md` (section deletion + description + zero-findings example), `src/kit/skills/fab-continue.md` (one parenthetical)
- **Scaffold**: `src/kit/scaffold/fab/project/code-review.md` (§ Review Tools comment block)
- **Specs (mirror class)**: `docs/specs/skills/SPEC-_review.md`, `docs/specs/skills/SPEC-fab-continue.md`, `docs/specs/skills.md`
- **Memory (hydrate)**: `docs/memory/pipeline/execution-skills.md`, `docs/memory/_shared/configuration.md`
- **Behavior delta**: review no longer shells out to `codex`/`claude` CLIs; ~15 min wall-clock saved per review when codex is installed. Reviewer diversity is provided by `agent.profiles.review.provider` (pre-ship) and Copilot at review-pr (post-ship). No Go, no tests, no CLI surface.

## Open Questions

*(none — the backlog entry is prescriptive on scope, the surviving toggle, the sweep class, and the non-goal)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is skill-prose only — zero Go/test changes; the one Go-side mention (`config_test.go:723` comment) stays accurate since § Review Tools survives with `copilot` | Backlog states "skill-prose only, zero Go"; gap analysis verified the Go hits are the untouched provider system | S:95 R:90 A:95 D:95 |
| 2 | Certain | The `copilot` entry in `code-review.md` § Review Tools stays; only `codex`/`claude` entries are removed from scaffold + docs | Backlog explicit: "copilot entry STAYS (git-pr-review reads it)" | S:95 R:90 A:95 D:90 |
| 3 | Certain | No provider-resolved second-opinion knob is built | Backlog explicit: "do NOT build it in this change" | S:100 R:95 A:100 D:100 |
| 4 | Certain | Review contract unchanged: single worker (pag2) is the holistic reviewer; zero findings still passes; `diff-only`/`fab-adopt` unaffected | Backlog states "diff-only/fab-adopt semantics unchanged: worker is the holistic reviewer, zero findings passes"; only the external-tool step is deleted | S:90 R:80 A:90 D:85 |
| 5 | Confident | Historical artifacts stay verbatim: migration `2.12.1-to-2.13.0.md`, `docs/specs/findings/skills-review-2026-06-11.md`, prior `log.md`/`log.seed.md` entries | FKF present-truth applies to living docs; migrations, dated findings, and logs are frozen records of past states — established repo convention | S:70 R:80 A:85 D:80 |
| 6 | Confident | No migration to strip stale `- codex: false`/`- claude: false` bullets from existing user projects' `code-review.md` | After deletion nothing reads them — inert prose in an optional user-owned file; the migrations rule targets structured data the kit reads/restructures (backlog names no migration) | S:55 R:75 A:65 D:55 |
| 7 | Confident | `change_type` overridden `feat` → `refactor` | Prose-only retirement of an internal mechanism with the review contract preserved; not `feat` (no new capability), not `docs` (behavior-bearing skill prose: codex is no longer invoked) | S:60 R:85 A:70 D:60 |
| 8 | Certain | Sweep mechanism is a repo-wide grep for `cascade`/`Codex` over the mirror class before finishing apply, excluding unrelated uses (config four-layer cascade, dispatch descent ladder, status-transition cascade) | code-quality.md § Sibling & Mirror Sweeps — must-fix review category in this repo; gap analysis already enumerated the sites | S:90 R:85 A:95 D:90 |

8 assumptions (5 certain, 3 confident, 0 tentative, 0 unresolved).
