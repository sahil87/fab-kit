# _preamble

## Contents

- Summary
- Subsection Inventory
- Flow

## Summary

Shared context preamble loaded by every Fab skill. Defines path conventions, context loading layers (always-load — descriptive, with a skill-file-wins override and a derived, never-enumerated exception set; change context; memory lookup; source code), the **Skill Helper Declaration** frontmatter convention (including stage-conditional in-body loading), inlined **Naming Conventions**, inlined **Run-Kit (rk) Reference**, the **Common fab Commands** headline table, the next-steps convention (with a skill-file-declared ending opt-out) with state table, a pointer to the skill invocation protocol (defined in `fab-clarify.md` since 260611-zc9m), subagent dispatch pattern with standard subagent context and **Per-Stage Model Resolution** (260613-l3ja — `fab resolve-agent <stage>` before each pipeline-stage dispatch; resolved model+effort passed to the Agent dispatch with empty ⇒ omit/inherit; review resolves once for both reviewers + merge; per-stage selection applies on every post-intake stage — every post-intake stage now dispatches a sub-agent (including plain `/fab-continue` as a one-stage sequencer), so `fab resolve-agent` applies uniformly across apply/review/hydrate, with the residual advisory case narrowed to a stage skill genuinely run with no dispatch at all (260613-fgxx); **the two halves dispatch through two seams (260613-m3d4)** — model via the Agent tool `model` param (a hard enum of short aliases `opus`/`sonnet`/`haiku`/`fable`, resolved with `fab resolve-agent <stage> --alias` so the alias is emitted directly — the deterministic Agent-tool adapter that replaced the earlier prompt-side id→alias hand-mapping; 260613-yky7) and effort via an explicit imperative instruction in the subagent prompt (the Agent tool has no effort param; omitted when empty), plus a **compliance-visibility** expectation that each site surface the resolved `model=/effort=` so a skipped/mis-resolved tier is visible rather than silent; resolution itself stays provider-neutral; the lone residual is a first-class per-sub-agent effort param on the Agent tool — a harness ask outside fab's control; the resolve-agent output is two byte-stable lines (`model=`/`effort=`) plus an **optional third `spawn=` line** emitted only when the resolved tier carries a `spawn_command` (the per-tier CLI-dispatch opt-in — absent ⇒ native Agent-tool dispatch, NO fallback to `agent.spawn_command`; 260702-24ec)), and — new in **260702-aetz (3d)** — the canonical **§ CLI-Adapter Dispatch** + **§ Dispatch-Prompt Obligations** subsections that WIRE `spawn=` into the seam: dispatch sites now **branch on `spawn=` presence** at the single `fab resolve-agent <stage> --alias` call (absent ⇒ native two-seam dispatch, byte-preserving; present ⇒ the CLI adapter `fab dispatch` — start-on-stdin → `sleep 30` poll → the five-state machine `running`/`done`/`failed`/`failed (no-result)`/`orphaned`, with the model/effort riding the `spawn=` command so the Agent-tool seams do not apply, and no cleanup after `done`), each site surfaces `spawn=` alongside `model=/effort=` for compliance visibility, and BOTH adapters' prompts carry the dispatch-prompt obligations (produce `{stage}-result.yaml` — CLI file at `.fab-dispatch/{id}/{stage}-result.yaml` / native structural return, with the load-bearing `status` vs `verdict` split; standard subagent context files; a terminal `fab status refresh` epilogue) plus the refined **block-contract carve-out** (prohibit `fab status` *transition* commands, REQUIRE the terminal `fab status refresh`; the orchestrator still owns all transitions) — wiring-only against the fixed contract `docs/specs/harness-adapters.md`, a pointer to the SRAD autonomy framework (extracted to `_srad.md` in 260611-zc9m), and slimmed confidence scoring (gate threshold + invocation; schema/formula/template moved to `_cli-fab.md` § fab score).

This is an internal partial (`user-invocable: false`) — it is never invoked directly. Skills reference it via the opening instruction: "Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding."

**Prose optimization** (260620-skop): a `## Contents` TOC added to `_preamble.md` (structural check, file >100 lines); no prose trimmed and no behavioral change (Flow / Subsection Inventory unchanged). This SPEC also received a `## Contents` block under the same structural check.

## Subsection Inventory

Post-260418-or0o, `_preamble.md` contains four additional subsections inlined from previously-separate helpers or lifted out of `_cli-fab.md`. Each is a canonical source within `_preamble`:

| Subsection | Purpose | Canonical source |
|------------|---------|------------------|
| `## Skill Helper Declaration` | Documents the per-skill `helpers:` frontmatter field, its 7 allowed values (`_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`, `_intake` — `_intake` added in 260613-3xaj for the pre-boundary Create-Intake Procedure consumed by `fab-new`/`fab-draft`), semantics (read each helper after `_preamble`, before body), stage-conditional in-body loading (point-of-use reads — used by `fab-continue` for `_generation`/`_review`), and default (empty → load only `_preamble`). Explicitly states that `_naming` and `_cli-rk` are inlined (not allowed as values) and that `_preamble` is implicit. | `_preamble.md` itself |
| `## Naming Conventions` | Change folder pattern (`{YYMMDD}-{XXXX}-{slug}`), git branch naming (matches folder name), worktree directory naming (`{adjective}-{noun}`). The operator spawning rules moved to `_cli-external.md`'s wt section (260611-zc9m). | `_preamble.md` (inlined from the deleted `_naming.md`) |
| `## Run-Kit (rk) Reference` | The universal silent-fail **detection rule** (`command -v rk`, skip silently when absent) plus a **pointer**: the full `rk` command body — `rk context` (server-URL discovery, iframe windows, proxy URL pattern, 4-step Visual Display Recipe) and `rk notify` — was relocated to `_cli-external.md` § rk (run-kit), so only operator skills (which load `_cli-external`) pay for the reference detail; every skill still carries the inline detection/fail-silent rule. | `_preamble.md` (detection rule; full body now in `_cli-external.md` § rk) |
| `## Common fab Commands` | Headline table of 6 most-used fab command families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`) with purpose and canonical invocation form. Cross-references `_cli-fab` for exhaustive flag documentation. Its "Key behaviors" list includes the generic failure rule: any fab command that exits non-zero → STOP and surface stderr (deferring to explicit per-skill handling where a skill intentionally branches on a non-zero exit; `fab log command` can never trip the rule through internal failure — given valid usage it always exits 0, surfacing internal failures as a stderr warning only (cobra arg-count errors exit non-zero before RunE), so the former `2>/dev/null \|\| true` guard boilerplate is retired as of 260612-ye8r). The `fab change` row's canonical form is `fab resolve --folder` — the query flags exist only on top-level `fab resolve`; `fab change resolve` takes a bare `[<override>]` (the former `fab change resolve --folder` canonical form was an invalid command, fixed in 260612-k4ge). | `_preamble.md` |

## Flow

```
Skill reads _preamble.md
│
├─ Path Convention
│  (all paths relative to repo root)
│
├─ Context Loading
│  ├─ Layer 1: Always Load (descriptive — the skill's own
│  │  Context Loading section wins; the exception set is
│  │  derived from each skill file, never enumerated —
│  │  e.g. fab-setup and docs-hydrate-memory skip the layer,
│  │  fab-operator loads a reduced 3-file set)
│  │  Read: config.yaml, constitution.md,
│  │        context.md*, code-quality.md*,
│  │        code-review.md*, memory/index.md,
│  │        specs/index.md
│  │  (no other helper — additional helpers
│  │   declared per-skill via frontmatter)
│  │
│  ├─ Layer 2: Change Context
│  │  Bash: fab preflight [change-name]
│  │  Bash: fab log command "<skill>" "<id>"
│  │  Read: change artifacts (intake, plan)
│  │
│  ├─ Layer 3: Memory File Lookup (up to 3-hop walk)
│  │  Read: intake affected memory refs ({domain}/{file} or {domain}/{sub-domain}/{file})
│  │  Read: docs/memory/{domain}/index.md
│  │  Read: docs/memory/{domain}/{sub-domain}/index.md   (only if the ref names a sub-domain)
│  │  Read: docs/memory/{domain}/[{sub-domain}/]{file}.md
│  │
│  └─ Layer 4: Source Code Loading
│     Read: source files from task/requirements refs
│     Read: neighboring files (pattern context)
│
├─ Skill Helper Declaration
│  (defines the `helpers:` frontmatter field —
│   allowed: _generation, _review, _cli-fab,
│            _cli-external, _srad, _pipeline, _intake;
│   plus stage-conditional in-body loading)
│
├─ Naming Conventions (inlined from _naming)
│  (change folder / git branch / worktree patterns —
│   operator spawning rules live in _cli-external.md)
│
├─ Run-Kit (rk) Reference
│  (detection / fail-silent rule + pointer;
│   full body — context, iframe, proxy, server
│   URL, visual recipe, notify — in _cli-external § rk)
│
├─ Common fab Commands
│  (headline table for 6 most-used families:
│   preflight, score, log command, change,
│   resolve, status — see _cli-fab for rest)
│
├─ Next Steps Convention
│  (state table lookup → "Next:" line — skills whose
│   Output/Key Properties declare a different ending
│   are exempt; the skill file wins)
│  (adoption note 260630-t54n: /fab-adopt needs no new
│   row — it enters with apply skipped + review active and
│   drives review→hydrate→ship→review-pr, states the table
│   already covers; a skipped stage is passed over by the
│   lookup exactly like a done stage)
│
├─ Skill Invocation Protocol (pointer)
│  (protocol defined in fab-clarify.md)
│
├─ Subagent Dispatch
│  ├─ Dispatch pattern (6 items)
│  ├─ Standard Subagent Context
│  │  Read: config.yaml, constitution.md,
│  │        context.md*, code-quality.md*,
│  │        code-review.md*
│  │  (applied at every nesting level)
│  └─ Per-Stage Model Resolution (260613-l3ja, m3d4)
│     Bash: fab resolve-agent <stage> before each
│           pipeline-stage sub-agent dispatch; SURFACE the
│           resolved model=/effort= (visibility — a skip is
│           then detectable; 260613-m3d4), then dispatch via
│           TWO SEAMS: model → Agent tool `model` param
│           (empty ⇒ omit/inherit; param is a short-alias enum
│           opus/sonnet/haiku/fable — resolve with
│           `fab resolve-agent <stage> --alias`, yky7)
│           and effort → imperative instruction in the subagent
│           prompt (no Agent effort param; empty ⇒ omit; m3d4).
│           Resolution itself is provider-neutral;
│           review resolves once for both reviewers + merge;
│           per-stage selection applies on every post-intake
│           stage (each now dispatches a sub-agent, incl. plain
│           /fab-continue as a one-stage sequencer) — advisory
│           only for a genuinely no-dispatch run (260613-fgxx).
│           Residual: a per-sub-agent effort param on the Agent
│           tool (harness ask, not built).
│  ├─ CLI-Adapter Dispatch (260702-aetz / 3d — canonical)
│  │  Branch on spawn= at the resolve-agent call:
│  │   absent  ⇒ native Agent-tool dispatch (two seams above)
│  │   present ⇒ fab dispatch (start-on-stdin → sleep 30 poll →
│  │             five states running/done/failed/
│  │             failed(no-result)/orphaned; profile rides the
│  │             spawn= command so Agent-tool seams don't apply;
│  │             NO fallback to agent.spawn_command; no cleanup
│  │             after done). Sites reference this, don't restate
│  │             the machine.
│  └─ Dispatch-Prompt Obligations (BOTH adapters — 260702-aetz)
│     produce {stage}-result.yaml (CLI file
│       .fab-dispatch/{id}/{stage}-result.yaml / native
│       structural return; status vs verdict split);
│     standard subagent context files;
│     terminal `fab status refresh` epilogue;
│     block-contract carve-out (no fab status TRANSITION
│       commands; REQUIRE terminal fab status refresh —
│       orchestrator still owns transitions)
│
├─ SRAD Autonomy Framework (pointer)
│  (framework extracted to _srad.md — loaded via
│   helpers: by the six planning skills)
│
└─ Confidence Scoring (gate threshold + invocation only;
   schema/formula/template in _cli-fab.md § fab score)
   Bash: fab score <change>

* = optional, skip if missing
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | all context layer files |
| Bash | `fab preflight`, `fab log command`, `fab score` |

### Sub-agents

None — `_preamble.md` is a convention document consumed by skills, not an executor. Subagent dispatch patterns are defined here but executed by the consuming skill.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| Change context | `fab log command "<skill>" "<id>"` | After preflight parse |
| Confidence scoring | `fab score --stage intake <change>` | After intake generation / clarify (intake is the sole scoring source; no scoring at apply or later) |
