# _preamble

## Contents

- Summary
- Subsection Inventory
- Flow

## Summary

Shared context preamble loaded by every Fab skill. Defines path conventions, context loading layers (always-load — descriptive, with a skill-file-wins override and a derived, never-enumerated exception set; change context; memory lookup; source code), the **Skill Helper Declaration** frontmatter convention (including stage-conditional in-body loading), inlined **Naming Conventions**, inlined **Run-Kit (rk) Reference**, the **Common fab Commands** headline table, the next-steps convention (with a skill-file-declared ending opt-out) with state table, a pointer to the skill invocation protocol (defined in `fab-clarify.md` since 260611-zc9m), subagent dispatch pattern with standard subagent context and **Per-Stage Model Resolution** (260613-l3ja — `fab resolve-agent <stage>` before each pipeline-stage dispatch; resolved model+effort passed to the Agent dispatch with empty ⇒ omit/inherit; review is unexceptional — one review sub-agent resolved once, like every other stage (260704-pag2, was "resolves once for both reviewers + merge"); per-stage selection applies on every post-intake stage — every post-intake stage now dispatches a sub-agent (including plain `/fab-continue` as a one-stage sequencer), so `fab resolve-agent` applies uniformly across apply/review/hydrate, with the residual advisory case narrowed to a stage skill genuinely run with no dispatch at all (260613-fgxx); **the two halves dispatch through two seams (260613-m3d4)** — model via the Agent tool `model` param (a hard enum of short aliases `opus`/`sonnet`/`haiku`/`fable`, resolved with `fab resolve-agent <stage> --alias` so the alias is emitted directly — the deterministic Agent-tool adapter that replaced the earlier prompt-side id→alias hand-mapping; 260613-yky7) and effort via an explicit imperative instruction in the subagent prompt (the Agent tool has no effort param; omitted when empty), plus a **compliance-visibility** expectation that each site surface the resolved `model=/effort=` so a skipped/mis-resolved tier is visible rather than silent; resolution itself stays provider-neutral; the lone residual is a first-class per-sub-agent effort param on the Agent tool — a harness ask outside fab's control; the resolve-agent output is a byte-stable `model=` line plus optional `effort=`/`provider=` lines and an **optional `dispatch=` line** emitted when the resolved tier's provider carries a `dispatch_command` (the CLI-dispatch opt-in — absent ⇒ native Agent-tool dispatch, no fallback to a session command for a *headless* dispatch; 260702-24ec, renamed from the per-tier `spawn_command`/`spawn=` in 260702-tykw) **or, per the `dispatch.watchable` opt-in (260806-mnri), when that field is `true` and the orchestrator sits inside tmux, for a `session_command`-only provider** — tmux presence deciding pane vs native, with the line then carrying the substituted `session_command`), and — new in **260702-aetz (3d)** — the canonical **§ CLI-Adapter Dispatch** + **§ Dispatch-Prompt Obligations** subsections that WIRE `dispatch=` into the seam: dispatch sites now **branch on `dispatch=` presence** at the single `fab resolve-agent <stage> --alias` call (absent ⇒ native two-seam dispatch, byte-preserving; present ⇒ the CLI adapter `fab dispatch` — start-on-stdin → `sleep 30` poll → the five-state machine `running`/`done`/`failed`/`failed (no-result)`/`orphaned`, with the model/effort riding the `dispatch=` command so the Agent-tool seams do not apply, and no cleanup after `done`), each site surfaces `dispatch=` alongside `model=/effort=/provider=` for compliance visibility, and EVERY adapter's prompt carries the dispatch-prompt obligations (produce `{stage}-result.yaml` — a real file at `.fab-dispatch/{id}/{stage}-result.yaml` for both `fab dispatch` modes / native structural return, with the load-bearing `status` vs `verdict` split; standard subagent context files; a terminal `fab status refresh` epilogue) plus the refined **block-contract carve-out** (prohibit `fab status` *transition* commands, REQUIRE the terminal `fab status refresh`; the orchestrator still owns all transitions) — wiring-only against the fixed contract `docs/specs/harness-adapters.md`; extended in **260805-j3cm** by a single **user-directed override paragraph**: when the user directs a provider/model/effort for specific stages ("run review on codex"), the site adds `--provider`/`--model`/`--effort` to its **existing single** `fab resolve-agent <stage> --alias` call — no new dispatch machinery, no persistent state (an override is per-invocation, so N stages means the same flags on N resolve calls), the two seams / branch-on-`dispatch=` / compliance-visibility rules all unchanged, with the load-bearing caveat that an invocation-time override binds the **native Agent-tool arm ONLY** — `fab dispatch start` accepts no override flags and re-resolves the stage from config itself, so a `dispatch=` line that appears only because of a `--provider` swap is **not actionable** — and the two remedies are NOT interchangeable: dispatching natively with the overridden model/effort is executable only for a **within-claude** `--model`/`--effort` override (the Agent tool's `model` param is a Claude-alias enum, so a non-Claude model has no native seam), which leaves the **config/tier override** that `dispatch start`'s own re-resolution sees as the SOLE executable path for a cross-provider `--provider` override; the resolved `dispatch=` is still re-read after an override, to notice the mismatch rather than act on it — and the swap-back asymmetry that `--provider claude` from a non-claude tier resolves an **empty** `model=` (claude's fill lives on the built-in *tiers*, not on the provider, so the swap has no `providers.claude.model` rung to refill from) — read by the seams as inherit-the-session-model, so pair it with an explicit `--model`; both caveats stay inside the SAME single paragraph (the "exactly one paragraph" constraint holds); extended in **260805-zxe0** by the **interactive-pane mode** — an *option inside* the `dispatch=`-present arm (`fab dispatch start --pane`), never a third branch and never resolver-visible: since **260805-l9ng** the mode **auto-resolves** rather than being opt-in per invocation (pane inside tmux — `$TMUX` set — headless outside), so dispatch sites pass **no** mode flag by default and add `--pane` only to force a window from a dispatcher outside tmux or `--headless` only to force an unattended run inside tmux (mutually exclusive; the pane path needs **both** a reachable tmux server **and** a `session_command` on the resolved provider, so a forced `--pane` hard-errors without either, while an auto-selected pane **soft-falls-back to headless** with a per-shape stderr notice — so neither a stale `$TMUX` nor a `dispatch_command`-only provider breaks an unattended dispatch — and the `dispatched …` line names the selection source when auto fired), and it reaches a **subset of three states** (`running`/`done`/`orphaned` — the two exit-code-derived states are unreachable without an exit-code channel, the state strings staying byte-identical), has **no `logs` analogue** (`fab pane capture [-L <server>] <pane>` instead of `fab dispatch logs` — the socket is included whenever the dispatch used `--server`, and the `fab dispatch logs` report prints the exact command since `status --json` carries `pane` but not `server`), requires a reachable tmux server **only when the pane was selected explicitly** (`--pane`/`--server`) — where the requirement is a hard error — while an auto-selected pane treats it as a soft prerequisite and falls back to headless, opens a `fab-{id}-{stage}` window carrying **no operator `»`/`›` marker**, and is **contract-neutral under steering** (a steered worker still owes its result file + terminal refresh and still runs no transition command); the obligations gain the note that **delivery mechanism varies** (dispatched prompt / stdin / prompt file + one-line pointer) while prompt *content* is composed identically; extended in **260806-mnri** by the **§ Recovery policy** — bounded, orchestrator-owned recovery composed OVER the existing five states (none added, renamed, or re-tabled; the result-file contract and prompt obligations untouched): **restart is tier 2's only recovery verb**, never a nudge (stage state lives in artifacts, so `fab dispatch restart` from the persisted prompt is cheap and deterministic, and it re-derives its mode from the *current* environment, so a pane dispatch orphaned by a dead tmux server relaunches headless); the budget is **exactly one restart per stage dispatch held in ORCHESTRATOR CONTEXT** with **no on-disk counter or history** (last-attempt-only preserved — worst case after a context loss is one extra restart); `orphaned` spends it **automatically** then resumes polling, `failed` gets **no automatic** restart (a deterministic failure would loop) though the orchestrator MAY read the log tail and spend the **same** budget on a clearly-transient signature, and `failed (no-result)` **always escalates and never restarts** (a contract violation needs eyes); **peek-on-suspicion** takes a READ-ONLY peek every **10th result-less poll** (`fab dispatch logs --tail 40` headless / `fab pane capture [-L <server>] <pane>` pane) and classifies three ways — (a) progressing ⇒ keep polling, (b) parked/dead-ended ⇒ kill + restart within the same budget, (c) awaiting genuine human input ⇒ notify **without killing** — because a wedged worker and a busy one both read `running`; escalation is per-mode evidence + a **gated `rk notify`** (`command -v rk`, fail-silent) + the stage's existing failure path, adding no state and no transition; the pipeline's verb set is exactly **peek / kill / restart / notify / stop** with **NO send-keys from the pipeline, ever** (nudging and answering stay operator and human territory — a pipeline nudge channel would fork the cross-adapter contract, since a native worker has no such channel); and the **pane subset carries the same policy** while `failed`/`failed (no-result)` stay *unreachable* there rather than newly handled, a pointer to the SRAD autonomy framework (extracted to `_srad.md` in 260611-zc9m), and slimmed confidence scoring (gate threshold + invocation; schema/formula/template moved to `_cli-fab.md` § fab score).

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
│           moving a stage to CLI dispatch needs a config/tier
│           override, not a flag. The two remedies are NOT
│           interchangeable: dispatching natively with the
│           overridden model/effort works only for a
│           WITHIN-CLAUDE --model/--effort override (the Agent
│           tool's model param is a Claude-alias enum), so a
│           cross-provider --provider override's SOLE
│           executable path is the config/tier override.
│  ├─ CLI-Adapter Dispatch (260702-aetz / 3d — canonical)
│  │  Branch on dispatch= at the resolve-agent call:
│  │   absent  ⇒ native Agent-tool dispatch (two seams above)
│  │   present ⇒ fab dispatch (start-on-stdin → sleep 30 poll →
│  │             five states running/done/failed/
│  │             failed (no-result)/orphaned; profile rides the
│  │             dispatch= command so Agent-tool seams don't apply;
│  │             NO fallback to a session command for a HEADLESS
│  │             dispatch; no cleanup after done). Sites reference
│  │             this, don't restate the machine.
│  │  dispatch.watchable (260806-mnri) adds a SECOND reason the
│  │    line can be present — the branch is UNCHANGED. With the
│  │    field true (bool, default false, scope both ⇒ settable
│  │    once machine-wide) AND $TMUX set, a session_command-only
│  │    provider (the built-in claude) also emits dispatch=,
│  │    carrying the substituted session_command. TMUX PRESENCE
│  │    DECIDES pane vs native: outside tmux the line is omitted
│  │    and the stage stays NATIVE, never headless CLI (headless
│  │    stays gated on a real dispatch_command). A provider's own
│  │    dispatch_command WINS; watchable only ADDS eligibility for
│  │    providers with none — replacing the "uncomment claude's
│  │    dispatch_command" footgun, which also flipped every
│  │    out-of-tmux dispatch to headless CLI. NO wiring change:
│  │    sites branch on presence and never execute the value, and
│  │    fab dispatch start re-resolves internally (inside tmux its
│  │    auto ladder picks pane and composes the same
│  │    session_command). Edge, documented not solved: tmux dying
│  │    between resolve and start ⇒ start soft-falls-back to
│  │    headless and errors on the missing dispatch_command.
│  │  Pane mode = an OPTION INSIDE the dispatch=-present arm,
│  │    never a third branch (260805-zxe0); the mode AUTO-RESOLVES
│  │    (260805-l9ng): pane inside tmux ($TMUX set), headless
│  │    outside, so pass NO mode flag by default — --pane forces a
│  │    window from a dispatcher outside tmux, --headless forces an
│  │    unattended run inside tmux (mutually exclusive: usage
│  │    error). The pane path needs BOTH a reachable tmux server
│  │    and a session_command: forced --pane hard-errors without
│  │    either, while an AUTO-selected pane soft-falls-back to
│  │    headless with a per-shape stderr notice (a stale $TMUX must
│  │    not break an unattended dispatch; a dispatch_command-only
│  │    provider must dispatch the same in and out of tmux). The
│  │    dispatched … line names the selection source (auto: tmux /
│  │    no tmux / tmux unreachable / no session_command) when auto
│  │    fired — surface it like the rest of the profile.
│  │    Three reachable states (running/done/orphaned —
│  │    failed/failed (no-result) have no exit-code channel);
│  │    no `logs` analogue (fab pane capture [-L <server>] <pane>
│  │    — include the socket whenever --server was used; the
│  │    fab dispatch logs report prints the exact command, since
│  │    status --json carries pane but not server); steering is
│  │    contract-neutral; TWO-TIER tmux hierarchy (260806-mnri):
│  │    operator→agents = windows (unchanged), agent→workers =
│  │    PANES splitting the agent's own window, stacked in a
│  │    right column (-v off the last fab- sibling, else -h off
│  │    $TMUX_PANE); new window fab-{id}-{stage} is the FALLBACK
│  │    ($TMUX_PANE unset, or --server naming another socket);
│  │    identity fab-{id}-{stage} rides the pane TITLE (split) or
│  │    the window NAME (fallback), no »/› marker in either.
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
│  │    orphaned ⇒ spend it automatically, then resume polling;
│  │    failed ⇒ NO automatic restart (a deterministic failure would
│  │      loop), but the orchestrator MAY read the log tail and
│  │      spend the SAME budget on a clearly-transient signature;
│  │    failed (no-result) ⇒ ALWAYS escalate, never restart
│  │      (a contract violation needs eyes);
│  │    peek on suspicion ⇒ every 10th result-less poll take a
│  │      READ-ONLY peek (fab dispatch logs --tail 40 headless /
│  │      fab pane capture pane) and classify three ways:
│  │      (a) progressing ⇒ keep polling; (b) parked/dead-ended ⇒
│  │      kill + restart in the same budget; (c) awaiting genuine
│  │      human input ⇒ notify, NEVER kill, NEVER type;
│  │    escalation = surface per-mode evidence + gated rk notify
│  │      (command -v rk, fail-silent) + stop per the stage's
│  │      existing failure path (no new state, no new transition);
│  │    the pipeline's verb set is exactly peek/kill/restart/
│  │      notify/stop — NO send-keys from the pipeline, ever
│  │      (nudging/answering stays operator + human territory; a
│  │      pipeline nudge channel would fork the cross-adapter
│  │      contract, since a native worker has none);
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
│     standard subagent context files;
│     terminal `fab status refresh` epilogue;
│     delivery mechanism varies (prompt / stdin / prompt file +
│       pointer) but prompt CONTENT is composed identically;
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
