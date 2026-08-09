# _preamble

## Contents

- Summary
- Subsection Inventory
- Flow

## Summary

**Source organization:** `_preamble` owns shared dispatch, logging, helper, and routing contracts; consumers retain only local deltas, while subset-owned contracts such as `/fab-clarify`'s `[AUTO-MODE]` protocol remain behind concise pointers.

**Prose packaging (260808-s2sz):** the source presents Per-Stage Model Resolution as native-seam and override tables, CLI-Adapter Dispatch as a two-branch table plus four-step procedure/state/recovery/peek tables and four pane bullets, and Confidence Scoring as a one-row gate table plus invocation list. This is a structural rewrite only; the contracts summarized below are unchanged.

Shared context preamble loaded by every Fab skill. It owns path/context/helper conventions, per-stage profile resolution (read through the four-tier config cascade: environment > system > project > built-in defaults), and the canonical cross-adapter dispatch procedure. `fab resolve-agent <stage> --alias` emits ordered `model=` plus optional `effort=`/`provider=`/`dispatch=` lines. `dispatch=` is derived from the `dispatch.mode` preference ceiling and independent provider capabilities (`interactive_command`, `native`, `headless_command`) through the descending `pane → native → headless` ladder; it is absent iff native resolves and otherwise carries the substituted pane/headless command. Dispatch sites surface the lines, branch only on `dispatch=` presence, and never execute its value. The CLI arm uses `fab dispatch`, its five-state observation/recovery contract, pane hygiene, and universal result/context/terminal-refresh prompt obligations; the native arm uses the Agent model and effort seams. Explicit pane/headless overrides remain hard, automatic selection never ascends, and native re-resolution at `fab dispatch start` stops actionably before state writes. The preamble also owns **native-arm worker continuation** (§ Worker Continuation — the `apply-{id}` name, SendMessage resume, mandatory fresh-dispatch fallback, profile fixity, and apply-only scope), the one-restart recovery budget, wait/peek/escalation policy, SRAD confidence gate, logging conventions, and next-step routing.

**Worker Continuation (260808-tv3g):** `_preamble.md` § Subagent Dispatch owns the mechanics that let an auto-rework-capable orchestrator reuse its apply worker across rework cycles on the **native arm only**; `_pipeline.md` carries the pointer plus the Step-1 naming and item-3 resume-first wiring (owner-or-pointer convention). The five mechanics: **naming** — the native apply dispatch passes `name: "apply-{id}"` (4-char change ID); **continuation** — a later cycle sends the triaged findings + chosen rework action to that name via SendMessage, and instructs the worker to RE-READ from disk every artifact the orchestrator edited at that item — always plan.md — because its in-context copy predates those edits, re-stating the block contract (results only, no `fab status` transition command, terminal `fab status refresh`) but deliberately NOT re-carrying the standard subagent context files, which the worker already holds; **fallback** — an unreachable handle (session resumed/restarted, no named-agent/SendMessage capability, send error, or worker never named because the stage went through the CLI adapter) falls back to today's fresh dispatch verbatim, including the full Dispatch-Prompt Obligations, and re-establishes the name for later cycles — continuation is an optimization, never a correctness dependency (Constitution III); **profile fixity** — a resumed worker keeps its first-dispatch model/effort and `fab resolve-agent apply --alias` is not re-run on the resume path, only on fresh (initial or fallback) dispatches; **scope guard** — apply only, inside the auto-rework loop, and review workers are never named or continued (the fresh-reviewer rule is reviewer-independence design), with hydrate and every other stage unaffected and the whole `dispatch=`-present CLI branch out of scope (headless non-resumable by decision, pane resume a separate change).

This is an internal partial (`user-invocable: false`) — it is never invoked directly. Skills reference it via the opening instruction: "Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding."

**Prose optimization** (260620-skop): a `## Contents` TOC added to `_preamble.md` (structural check, file >100 lines); no prose trimmed and no behavioral change (Flow / Subsection Inventory unchanged). This SPEC also received a `## Contents` block under the same structural check.

## Subsection Inventory

Post-260418-or0o, `_preamble.md` contains four additional subsections inlined from previously-separate helpers or lifted out of `_cli-fab.md`. Each is a canonical source within `_preamble`:

| Subsection | Purpose | Canonical source |
|------------|---------|------------------|
| `## Skill Helper Declaration` | Documents the per-skill `helpers:` frontmatter field, its 8 allowed values (`_generation`, `_review`, `_cli-fab`, `_cli-external`, `_cli-agents`, `_srad`, `_pipeline`, `_intake` — `_intake` added in 260613-3xaj for the pre-boundary Create-Intake Procedure consumed by `fab-new`/`fab-draft`, plus `fab-dedupe` as of 260728-4v91; `_cli-agents` added in 260805-nvad for the agent-CLI interaction procedures + provider dictionary extracted from `fab-operator.md`, declared by `/fab-operator`), semantics (read each helper after `_preamble`, before body), stage-conditional in-body loading (point-of-use reads — used by `fab-continue` for `_generation`/`_review`), and default (empty → load only `_preamble`). Explicitly states that `_naming` and `_cli-rk` are inlined (not allowed as values) and that `_preamble` is implicit. | `_preamble.md` itself |
| `## Naming Conventions` | Change folder pattern (`{YYMMDD}-{XXXX}-{slug}`), git branch naming (matches folder name), worktree directory naming (`{adjective}-{noun}`). The operator spawning rules moved to `_cli-external.md`'s wt section (260611-zc9m). | `_preamble.md` (inlined from the deleted `_naming.md`) |
| `## Run-Kit (rk) Reference` | The universal silent-fail **detection rule** (`command -v rk`, skip silently when absent) plus a **pointer**. As of `260718-clix` the `rk` command surface (`rk context` — server-URL discovery, iframe windows, proxy URL pattern, Visual Display Recipe — and the `rk notify` contract) is **tool-owned**, read at use-time via `rk skill`, with the same **version-skew fallback** carried inline as `_cli-external.md`'s (capability-probe `rk skill`; on failure fall back silently to the shll.ai bundle-page pointer `https://shll.ai/rk/skill`, present-but-old → the pointer, absent → the `command -v` gate skips) so the always-load layer stays self-consistent for readers who have not loaded `_cli-external.md`; the pointer forwards only to the **fab-owned** rk usage (the operator's escalation `rk notify` send) in `_cli-external.md` § rk (run-kit), which only operator skills pay for. Every skill still carries the inline detection/fail-silent rule. | `_preamble.md` (detection rule + version-skew fallback; command surface delegated to `rk skill`; fab-owned usage in `_cli-external.md` § rk) |
| `## Common fab Commands` | Headline table of 6 most-used fab command families (`preflight`, `score`, `log command`, `change`, `resolve`, `status`) with purpose and canonical invocation form. Cross-references `_cli-fab` for exhaustive flag documentation. Its "Key behaviors" list includes the generic failure rule: any fab command that exits non-zero → STOP and surface stderr (deferring to explicit per-skill handling where a skill intentionally branches on a non-zero exit; `fab log command` can never trip the rule through internal failure — given valid usage it always exits 0, surfacing internal failures as a stderr warning only (cobra arg-count errors are usage errors that exit 2 before RunE — 260717-swon), so the former `2>/dev/null \|\| true` guard boilerplate is retired as of 260612-ye8r). The `fab change` row's canonical form is `fab resolve --folder` — the query flags exist only on top-level `fab resolve`; `fab change resolve` takes a bare `[<override>]` (the former `fab change resolve --folder` canonical form was an invalid command, fixed in 260612-k4ge). The `fab resolve` row's signature carries `[--or-none]` and its canonical form is the probe form `fab resolve --folder --or-none` (260720-dow0 — absence-as-data: state-sentinel failures print exactly `(none)` + exit 0, not-found always / ambiguous only bare, real errors stay non-zero; the purpose text names `fab preflight` as the strict validation gate, resolve as the pure query that can answer "none" when asked). | `_preamble.md` |

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
│            _cli-external, _cli-agents, _srad,
│            _pipeline, _intake;
│   plus stage-conditional in-body loading)
│
├─ Naming Conventions (inlined from _naming)
│  (change folder / git branch / worktree patterns —
│   operator spawning rules live in _cli-external.md)
│
├─ Run-Kit (rk) Reference
│  (detection / fail-silent rule + version-skew
│   fallback + pointer; command surface — context,
│   iframe, proxy, server URL, visual recipe, notify
│   contract — delegated to `rk skill` (capability-probe,
│   fall back to https://shll.ai/rk/skill); fab-owned
│   escalation usage in _cli-external § rk)
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
├─ Subagent Dispatch
│  ├─ Dispatch pattern (6 items)
│  ├─ Standard Subagent Context
│  │  Read: config.yaml, constitution.md,
│  │        context.md*, code-quality.md*,
│  │        code-review.md*
│  │  (applied at every nesting level, on every DISPATCH
│  │   prompt; a continuation message to an already-running
│  │   named worker does NOT re-carry them — Worker
│  │   Continuation below)
│  └─ Per-Stage Model Resolution (260613-l3ja, m3d4)
│     Bash: fab resolve-agent <stage> --alias before each
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
│           review is unexceptional — one review sub-agent,
│           resolved once like every other stage (260704-pag2);
│           per-stage selection applies on every post-intake
│           stage (each now dispatches a sub-agent, incl. plain
│           /fab-continue as a one-stage sequencer) — advisory
│           only for a genuinely no-dispatch run (260613-fgxx).
│           Residual: a per-sub-agent effort param on the Agent
│           tool (harness ask, not built).
│           User-directed overrides (260805-j3cm): add
│           --provider/--model/--effort to the SAME single
│           resolve call; per-invocation, no new machinery.
│           NATIVE ARM ONLY — fab dispatch start takes no
│           override flags (it re-resolves from config), so an
│           override-only dispatch= line is not actionable;
│           moving a stage to CLI dispatch needs a config
│           override, not a flag. The two remedies are NOT
│           interchangeable: dispatching natively with the
│           overridden model/effort works only for a
│           WITHIN-CLAUDE --model/--effort override (the Agent
│           tool's model param is a Claude-alias enum), so a
│           cross-provider --provider override's SOLE
│           executable path is the config override.
│  ├─ Worker Continuation — NATIVE ARM ONLY (260808-tv3g)
│  │  naming: the native apply dispatch passes
│  │    name: "apply-{id}" (4-char change id) from the
│  │    auto-rework-capable orchestrators (fab-ff / fab-fff,
│  │    fab-adopt as partial consumer of that loop);
│  │  continuation: a later rework cycle SendMessages the
│  │    triaged findings + chosen rework action to that name
│  │    instead of spawning fresh; the prompt instructs the
│  │    worker to RE-READ from disk every artifact the
│  │    orchestrator edited at that item — always plan.md —
│  │    because its in-context copy predates those edits;
│  │    the prompt also RE-STATES the
│  │    block contract (results only, no fab status TRANSITION
│  │    command, terminal fab status refresh) and deliberately
│  │    does NOT re-carry the standard subagent context files
│  │    (the worker holds them — that is the point);
│  │  fallback (LOAD-BEARING): handle gone (session resumed/
│  │    restarted) / no named-agent+SendMessage capability /
│  │    send errors / never named (CLI adapter) ⇒ ordinary
│  │    fresh dispatch with the full Dispatch-Prompt
│  │    Obligations, which RE-ESTABLISHES the name. An
│  │    optimization, never a correctness dependency —
│  │    worst case is today's behavior (Constitution III);
│  │  profile fixity: a resumed worker keeps its first-dispatch
│  │    model/effort; fab resolve-agent apply --alias is NOT
│  │    re-run on the resume path (resolution + surfacing
│  │    happen only on fresh dispatches, initial or fallback);
│  │  scope guard: apply ONLY, inside the auto-rework loop.
│  │    Review workers are never named and never continued
│  │    (reviewer-independence); hydrate + all other stages
│  │    unaffected; the whole dispatch=-present CLI branch is
│  │    out of scope (headless non-resumable BY DECISION,
│  │    pane resume = a separate follow-up change)
│  │
│  ├─ CLI-Adapter Dispatch (260702-aetz / 3d — canonical)
│  │  Branch on dispatch= at the resolve-agent call:
│  │   absent  ⇒ native Agent-tool dispatch (two seams above)
│  │   present ⇒ fab dispatch (start-on-stdin → BACKGROUND blocking
│  │             fab dispatch wait --timeout 300 (foreground
│  │             blocking where the harness has no notify-on-exit
│  │             background command), 260806-mkfj →
│  │             five states running/done/failed/
│  │             failed (no-result)/orphaned; profile rides the
│  │             dispatch= command so Agent-tool seams don't apply;
│  │             NO fallback to a session command for a HEADLESS
│  │             dispatch; done ⇒ read the result THEN run
│  │             fab dispatch reap (260807-zfl7) — unconditional
│  │             and dumb, the whole guard being Go-side; no STATE
│  │             cleanup after done). Sites reference
│  │             this, don't restate the machine.
│  │  reap (260807-zfl7) is wired into the step-3 done bullet: a
│  │    pane worker never exits on completion, so without it every
│  │    finished stage keeps its slice of the carved column. The
│  │    skill makes NO mode check and NO config check — reap acts
│  │    only when the record is pane-mode AND the state is done AND
│  │    dispatch.reap_done (bool, default TRUE, scope both) is
│  │    true, and the knob is read through the four-tier cascade (environment > system > project > built-in defaults)
│  │    a skill reading fab/project/config.yaml would get wrong.
│  │    Every no-op reports its reason and exits 0 (headless = the
│  │    process already exited); only a missing record errors.
│  │    Step 4's "no cleanup after done" is NARROWED, not deleted:
│  │    reap is PANE HYGIENE and removes no files, so the record +
│  │    result survive (which is why a reaped dispatch reads done
│  │    forever) and the two state-cleanup moments are unchanged.
│  │  dispatch.mode (default native, scope both) is a preference
│  │    ceiling over pane → native → headless. Resolution descends
│  │    only, using independent provider capabilities plus tmux:
│  │    pane needs interactive_command + tmux, native needs native:true,
│  │    headless needs headless_command. Presence says HOW, never
│  │    WHETHER. dispatch= is absent iff native resolves and present
│  │    with the substituted pane/headless command otherwise. Sites
│  │    still branch only on presence and never execute the value.
│  │  Pane mode = an OPTION INSIDE the dispatch=-present arm,
│  │    never a third branch (260805-zxe0). Pass NO mode flag by
│  │    default; start/restart re-resolve config + capabilities +
│  │    current environment through the same ladder. --pane and
│  │    --headless are mutually exclusive one-shot overrides whose
│  │    missing prerequisites hard-error. Automatic missing rungs
│  │    descend with `mode: <rung> (preferred)` or
│  │    `mode: <rung> (descended: <reason>[; <reason>])`; reasons
│  │    are pane unavailable: no tmux / tmux unreachable /
│  │    no interactive_command, then native unavailable. Landing on
│  │    native tells the caller to re-run resolve-agent and writes
│  │    no state.
│  │    Three reachable states (running/done/orphaned —
│  │    failed/failed (no-result) have no exit-code channel);
│  │    no `logs` analogue (fab pane capture [-L <server>] <pane>
│  │    — include the socket whenever --server was used; the
│  │    fab dispatch logs report prints the exact command, since
│  │    status --json carries pane but not server); steering is
│  │    contract-neutral; TWO-TIER tmux hierarchy (260806-mnri):
│  │    operator→agents = windows (unchanged), agent→workers =
│  │    PANES splitting the agent's own window, stacked in a
│  │    right column — -v (unsized) off the last live sibling
│  │    worker pane, else -h -l <n>% off $TMUX_PANE, CARVING the
│  │    column at dispatch.column_width (default 35, 260807-g4a5);
│  │    sibling detection reads the DISPATCH RECORDS' pane ids, not
│  │    pane titles (a harness rewrites its worker's title within
│  │    seconds), scoped to the probed socket (pane ids are
│  │    per-server), so the COLUMN INVARIANT holds: the left/right
│  │    separator is created once and never re-touched — no
│  │    select-layout, no rearranging user panes, no fighting
│  │    manual resizes; placement stays cosmetic and degrades
│  │    warn-only, every failure named on stderr with the placement
│  │    it resolved to (failed list-panes / unreadable tree ⇒ the
│  │    same sized carve; one unreadable record ⇒ the rest still
│  │    stack; a tmux without -l <n>% ⇒ retried unsized; an ABSENT
│  │    tree is the first-dispatch case, not a warning);
│  │    new window fab-{id}-{stage} is the FALLBACK
│  │    ($TMUX_PANE unset, or --server naming another socket);
│  │    identity fab-{id}-{stage} rides the pane TITLE (split) or
│  │    the window NAME (fallback), no »/› marker in either —
│  │    titles are IDENTIFICATION only, never a placement signal.
│  │  Recovery policy (260806-mnri) — bounded, orchestrator-owned,
│  │    composed OVER the five states (none added or renamed):
│  │    restart is tier 2's ONLY recovery verb (never a nudge —
│  │      stage state lives in artifacts, so fab dispatch restart
│  │      from the persisted prompt is cheap and deterministic, and
│  │      it re-derives its mode from the CURRENT environment);
│  │    budget = exactly ONE restart per stage dispatch, held in
│  │      ORCHESTRATOR CONTEXT — no on-disk counter or history
│  │      (last-attempt-only preserved; worst case after a context
│  │      loss is one extra restart);
│  │    orphaned ⇒ spend it automatically, then re-arm wait;
│  │    failed ⇒ NO automatic restart (a deterministic failure would
│  │      loop), but the orchestrator MAY read the log tail and
│  │      spend the SAME budget on a clearly-transient signature;
│  │    failed (no-result) ⇒ ALWAYS escalate, never restart
│  │      (a contract violation needs eyes);
│  │    peek on suspicion ⇒ on every TIMEOUT-RETURN of fab dispatch
│  │      wait (a `running` state printed at the --timeout 300
│  │      bound — the same ~5-min cadence 10 polls × 30s gave)
│  │      take a READ-ONLY peek (fab dispatch logs --tail 40
│  │      headless / fab pane capture pane) and classify 3 ways:
│  │      (a) progressing ⇒ re-arm wait; (b) parked/dead-ended ⇒
│  │      kill + restart in the same budget; (c) awaiting genuine
│  │      human input ⇒ notify, NEVER kill, NEVER type;
│  │    escalation = surface per-mode evidence + gated rk notify
│  │      (command -v rk, fail-silent) + stop per the stage's
│  │      existing failure path (no new state, no new transition);
│  │    the pipeline's verb set is exactly peek/kill/restart/
│  │      notify/stop/reap — NO send-keys from the pipeline, ever
│  │      (nudging/answering stays operator + human territory; a
│  │      pipeline nudge channel would fork the cross-adapter
│  │      contract, since a native worker has none);
│  │    reap is NOT kill and is not a recovery verb: kill is
│  │      recovery (any state, ungated, spent by classification
│  │      (b)); reap is hygiene (done only, knob-gated, never
│  │      terminates a running/orphaned/failed dispatch), so
│  │      nothing in this policy changes because of it;
│  │    the pane subset carries the same policy (orphaned gets the
│  │      same one restart, and a mode re-derivation lands it
│  │      headless when tmux died); failed/failed (no-result) stay
│  │      UNREACHABLE there rather than newly handled.
│  └─ Dispatch-Prompt Obligations (ALL THREE adapters — 260702-aetz,
│     third adapter 260805-zxe0)
│     produce {stage}-result.yaml (both fab dispatch modes: a file
│       at .fab-dispatch/{id}/{stage}-result.yaml / native
│       structural return; status vs verdict split; on the pane
│       mode it is the SOLE completion signal);
│     standard subagent context files (binds every DISPATCH; a
│       continuation message to an already-running named worker
│       carries obligations 1 and 3 only — Worker Continuation);
│     terminal `fab status refresh` epilogue;
│     delivery mechanism varies (prompt / stdin / prompt file +
│       pointer) but DISPATCH prompt CONTENT is composed
│       identically;
│     block-contract carve-out (no fab status TRANSITION
│       commands; REQUIRE terminal fab status refresh —
│       orchestrator still owns transitions, incl. on the pane
│       path where a user may steer the worker)
│
├─ SRAD Autonomy Framework (pointer)
│  (framework extracted to _srad.md — loaded via
│   helpers: by the planning skills)
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
