# Intake: Distill No-Arg Survey Mode

**Change**: 260718-ukpf-distill-noarg-survey
**Created**: 2026-07-18

## Origin

Promptless dispatch from a `/fab-discuss` DX-evaluation session the user approved for implementation. The synthesized description (verbatim decisions below) was handed to a create-intake sub-operation with `{questioning-mode} = promptless-defer` — no questions asked; agent-decided details are graded rows in `## Assumptions`.

> Make `/docs-distill-memory`'s `<domain>` argument optional by adding a no-arg survey mode, and make the `Next:` line dynamic. This is a change to the skill markdown only — no Go/CLI changes (the survey is performed inline by the agent via frontmatter reads + grep; no new `fab` subcommand).

## Why

1. **Inconsistent no-arg behavior.** `/docs-distill-memory` with no argument currently aborts with `"Name a domain to distill, e.g. /docs-distill-memory pipeline. Run one domain per run."` (Error Handling, `src/kit/skills/docs-distill-memory.md`) — and that abort doesn't even list available domains, though the unknown-domain error does (`"Available: {list domain folders}."`). Sibling skills do better: `/fab-switch` lists available changes on no-arg; `/docs-reorg-memory` runs tree-wide with no argument.
2. **No way to ask "what's left to distill?"** Distillation is a one-time corpus sweep (the forward-looking FKF writers — hydrate, `/docs-hydrate-memory`, `/docs-reorg-memory` — already emit present truth going forward), so the natural workflow is "run it until nothing's left" — yet users end up tracking remaining domains by hand (the actual user was doing exactly this in session memory: "remaining domains: distribution, runtime, _shared, memory-docs").
3. **The one-domain-per-run rule is enforced at the wrong seam.** Its rationale (per-domain approval granularity + rewrite-quality/context budget) is a property of the analysis+apply unit, not the invocation. Forcing the user to name the domain buys no safety — the Step 3 approval gate (apply all / cherry-pick / skip) already protects everything.

If unfixed: the skill stays needlessly harder to invoke than its siblings, and the "run until done" loop keeps requiring hand-tracked state outside the tool.

**Rejected alternatives** (from the discussion):

- **Multi-domain invocation expanding to sequential per-domain runs with per-domain approval gates** — deferred/rejected for now: the auto-pick already gives a one-keystroke loop, and a single session chewing through all domains erodes rewrite quality as context fills. The multiple-domains abort stays.
- **Persistent distilled-state marker/tracking file** — unnecessary: distillation is a one-time remediation sweep, and extra state violates the docs-are-source-of-truth ethos (Constitution II); survey-scan each time instead.

## What Changes

Skill markdown only — no Go binary changes, so no `_cli-fab.md` update and no Go test obligations. The survey is performed inline by the agent (frontmatter reads + grep); there is no new `fab` subcommand.

### 1. `<domain>` becomes optional — no-arg survey mode

The `## Arguments` entry changes from `<domain>` *(required)* to optional; the header `# /docs-distill-memory <domain>` becomes `# /docs-distill-memory [<domain>]`.

**No-arg invocation runs a survey mode** — a cheap heuristic scan over all domains:

- **Mechanical `description:` frontmatter defects**: value over the 500-char cap; change-ids present (a `— xu0k`-style suffix or a `(d9rs)`-style citation) — the same §3.2 defect classes distillation already fixes.
- **Narration markers in bodies**: a grep for common transition-narration patterns (e.g. `renamed`, `supersed`, `was \``) — seeded from the skill's own Step 1 narration-pattern list.

The survey then:

1. **Reports per-domain status** (e.g. which domains have flagged files and how many).
2. **Auto-selects the first domain with candidates**, announces the pick, and proceeds into the existing one-domain flow — full read of the selected domain, per-file report, Step 3 approval gate — all unchanged.
3. If the survey finds nothing anywhere, reports the terminal "all domains distilled (survey heuristic)" case with the caveat (change area 3).

**An explicit `<domain>` argument remains as the override** and forces a full read of that domain regardless of survey heuristics.

**Survey exclusions match distillation's**: skip `index.md`, `log.md` (generated), `log.seed.md` (curated seed input), `_shared/removed-domains.md` (tombstone exemption); recurse into sub-domains like Step 1 does.

**Pre-flight / Error Handling changes meaning**: the "No `<domain>` argument" abort row is **replaced** by survey mode. The ambiguous-domain abort, unknown-domain abort, and multiple-domains abort all **stay** unchanged.

### 2. Dynamic `Next:` line

After a domain completes, the skill emits the surveyed remaining candidate domains, e.g.:

```
Next: /docs-distill-memory distribution (3 files flagged), /docs-distill-memory runtime (1 file flagged), …
```

or reports "all domains distilled" when none remain. This **replaces** the static placeholder line `Next: /docs-distill-memory {another-domain}, /docs-reorg-memory, or /fab-new` (last line of the skill). Per `_preamble.md` § Next Steps Convention, a skill's own Output/Key Properties ending wins over the State Table — this skill already ends with its own line, so making it dynamic is convention-clean.

On an explicit-`<domain>` invocation (no upfront survey ran), the completion step runs the survey to populate the `Next:` line; on a no-arg invocation the initial survey results may be reused minus the completed domain (a domain the user skipped or only partially cherry-picked stays listed while it still carries flagged files).

### 3. Survey caveat stated in output

The survey is heuristic — narration-marker greps catch common patterns, but a domain could pass the cheap scan while still carrying superseded-state prose. That is fine for ranking/picking (the full read still happens once a domain is selected); the only silent-skip risk is the "survey says all clean" terminal case, so the output MUST state the caveat, e.g.:

```
Survey is heuristic; run /docs-distill-memory <domain> to force a full read of a specific domain.
```

### 4. Sweep class (constitution + code-quality.md § Sibling & Mirror Sweeps)

All edited in this change — never the `.claude/skills/` deployed copies:

- **`src/kit/skills/docs-distill-memory.md`** — canonical source: header, frontmatter `description:` (currently "One domain per run; read-only until you approve" — one-domain-per-apply-run stays true; add the optional-domain/survey capability), `## Arguments`, `## Pre-flight`, `## Behavior` (survey step), `## Output` (survey report + caveat + dynamic `Next:`), `## Error Handling` (drop the no-arg abort row), `## Key Properties` (scope row), the final `Next:` line.
- **`docs/specs/skills/SPEC-docs-distill-memory.md`** — mirror: Summary, Flow diagram (survey branch), Tools table if affected.
- **`docs/specs/skills.md`** — aggregate: the `## /docs-distill-memory <domain>` heading and its Purpose/Behavior/Key-properties restatement (lines ~760–774) restate the required-argument and one-domain behavior — update to `[<domain>]` + survey mode.
- **`docs/memory/memory-docs/distill.md`** — the memory file documenting this skill (its `description:` frontmatter and "One Domain Per Run, Propose-Then-Apply" requirement restate the required-domain invocation) — in the sweep class even though hydrate owns the final write; see Affected Memory.

## Affected Memory

- `memory-docs/distill.md`: (modify) — invocation semantics change: `<domain>` optional, no-arg survey mode (heuristics, exclusions, auto-pick, caveat), dynamic `Next:` line; the One-Domain-Per-Run requirement is rescoped to the analysis+apply unit (unchanged per run) rather than the invocation. Add a Design Decisions entry for the rejected alternatives (multi-domain sequential runs; persistent distilled-state marker).

(`docs/memory/memory-docs/index.md` / `log.md` regenerate via `fab memory-index` — never hand-edited.)

## Impact

- **Files**: 4 markdown files (skill source + SPEC mirror + skills.md aggregate + memory file at hydrate). No Go/CLI changes, no `_cli-fab.md`, no Go tests, no migrations (no user-data restructuring — Constitution/context.md § Migrations not triggered).
- **Behavior**: purely additive for existing invocations — explicit-`<domain>` runs are unchanged except for the dynamic `Next:` line; all existing aborts except the no-arg row stay. Idempotency (Constitution III) is preserved: the survey is read-only and re-runnable; a fully-distilled tree surveys clean every time.
- **Guardrails unchanged**: one domain per apply run; read-only-until-approval posture; Step 3 approval gate (apply all / cherry-pick / skip); multiple-domains abort.

## Open Questions

None — the design discussion resolved all decision points; agent-decided details are graded below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-arg invocation runs survey mode (heuristic scan → per-domain report → auto-pick first candidate → announce → existing one-domain flow) instead of aborting | Discussed — user approved this exact design for implementation | S:95 R:85 A:95 D:95 |
| 2 | Certain | Survey is inline agent work (frontmatter reads + grep); no new `fab` subcommand, no Go changes | Discussed — explicitly scoped to skill markdown only | S:95 R:85 A:95 D:95 |
| 3 | Certain | Explicit `<domain>` stays as the override and forces a full read regardless of survey heuristics | Discussed — stated verbatim in the approved decisions | S:95 R:90 A:95 D:95 |
| 4 | Certain | Dynamic `Next:` line lists surveyed remaining candidates (or "all domains distilled"), replacing the static `{another-domain}` placeholder | Discussed — decision 2 of the approved design | S:95 R:90 A:95 D:90 |
| 5 | Certain | Survey caveat is stated in output; the terminal "survey says all clean" case must carry it | Discussed — decision 3 of the approved design | S:90 R:90 A:95 D:90 |
| 6 | Certain | Guardrails unchanged: one domain per apply run, Step 3 approval gate, read-only-until-approval, multiple-domains/ambiguous/unknown aborts stay | Discussed — listed as constraints; only the no-arg abort row is replaced | S:95 R:85 A:95 D:95 |
| 7 | Certain | Survey exclusion set = distillation's (`index.md`, `log.md`, `log.seed.md`, `_shared/removed-domains.md`), recursing sub-domains | Discussed constraint; mirrors the skill's existing Step 1 skip list | S:90 R:90 A:95 D:95 |
| 8 | Certain | Sweep class = skill source + SPEC mirror + `docs/specs/skills.md` aggregate (verified: its `## /docs-distill-memory <domain>` heading + behavior restatement) + `memory-docs/distill.md`; never `.claude/skills/` copies | Constitution + code-quality.md § Sibling & Mirror Sweeps; aggregate coverage grep-verified | S:90 R:85 A:95 D:95 |
| 9 | Confident | Survey scan order and auto-pick order = the domain order of `docs/memory/index.md`'s domain table (deterministic; matches the user-facing landscape); `Next:` candidates listed in the same order | Not discussed — obvious default; presentational and trivially reversible skill prose | S:55 R:90 A:75 D:65 |
| 10 | Confident | Narration-marker grep list is seeded from the skill's own Step 1 narration patterns (`renamed`, `supersed`, `was \``, `superseding the historical`, `inverts`); the "e.g." phrasing keeps it extensible at apply | Description gives examples with "e.g."; the skill body already enumerates the canonical patterns | S:60 R:90 A:85 D:70 |
| 11 | Confident | On explicit-`<domain>` invocations the survey runs at completion to populate the dynamic `Next:` line; no-arg invocations may reuse the initial survey minus the completed domain | "Emits the surveyed remaining candidate domains" implies a survey result at completion; cheap and read-only either way | S:50 R:90 A:80 D:70 |
| 12 | Confident | A skipped or partially cherry-picked domain stays in the `Next:` candidate list while it still carries flagged files | The line reports surveyed truth; hiding a still-flagged domain would contradict the survey's purpose | S:45 R:90 A:80 D:70 |
| 13 | Confident | Survey heuristic set is exactly the three discussed classes (over-cap `description:`, change-ids in `description:`, narration markers in bodies); a missing `type: memory` is NOT a survey signal (the full read catches it once a domain is selected) | The approved design enumerates a closed defect list; silently expanding scope would deviate from the discussion | S:60 R:90 A:70 D:70 |
| 14 | Confident | Skill frontmatter `description:` is updated to advertise the optional domain/survey mode while keeping the still-true "one domain per run" (per apply) phrasing | Sweep-class constraint names the frontmatter; wording is presentational and reversible | S:55 R:90 A:85 D:75 |

14 assumptions (8 certain, 6 confident, 0 tentative, 0 unresolved).
