# Intake: Operator Question-Sweep Command (`fab pane questions`)

**Change**: 260823-dckc-pane-questions-sweep
**Created**: 2026-08-23

## Origin

<!-- How was this change initiated? -->

> Change 4 / Phase B2 of `fab/plans/sahil/26-08-23-operator-offload-plan.md` ("Operator
> offload plan — narrative state + load reduction"), invoked via `/fab-new` as the final
> item of the plan's ordered same-repo queue. The plan was written from a `/fab-discuss`
> brainstorm, then revised after a four-agent review (Go feasibility, doctrine coherence,
> failure red-team, scope). All three predecessors are **merged to main**: Phase A
> (`operator note`, PR #615), Phase C2 (auto-merge choreography, PR #616), and Phase B1
> (`fab operator tick-start --diff` + `candidates:`/`fleet:` blocks, PR #617, main commit
> 88e4d808) — so unlike an earlier drafting of this change, B1's `candidates:` block
> exists and is already the operator tick's sweep population (`fab-operator.md` §4 tick
> step 2 references it today).
>
> This intake covers **only** the "B2 · `fab pane questions [--all-sessions] [--panes
> <id>...] [--json]`" section of the plan: a new `fab pane questions` command that sweeps
> candidate panes for pending questions/prompts, moving the mechanical parts of the
> operator's §5 auto-nudge detection (capture, guards, indicator-pattern scanning) from
> the operator's LLM-driven prose into the `fab` binary, plus the accompanying skill and
> memory rewrites that wire the operator's §5 detection onto the new command.
>
> **Prior-attempt salvage**: a first run of this change
> (`260823-ekh9-pane-questions-sweep`) died mid-apply in a since-deleted worktree; its
> work survives as stash commit `1749af376ee48709fcc0678e32f56ec93ad5796e` in the shared
> stash stack ("wt-delete: saved from worktree 'opal-geyser'"). It contains a complete
> `src/go/fab/cmd/fab/pane_questions.go` (306 lines — command, capture seam, pure
> guard/indicator scanner) and the one-line `pane.go` registration, both verified
> compatible with current main (the code calls B1's `collectPaneRows`, and
> `pane.Capture`/`AgentStateWaiting`/`AgentStateIdle`/`paneRow` all exist unchanged). It
> contains **no tests and no skill/memory edits**. Apply SHOULD seed the Go
> implementation from this stash (`git show 1749af37:src/go/fab/cmd/fab/pane_questions.go`)
> rather than rewriting it, then review it against the requirements, and write the
> tests + skill/memory edits fresh.

## Why

1. **The problem**: every operator tick that has a `waiting` or `idle` monitored agent
   today does a manual capture-and-scan per pane — the operator (running as an LLM) reads
   `fab-operator.md` §5's guard/pattern list from memory and applies it freehand for each
   pane. This is mechanical work (regex matching over captured text) being redone in
   natural-language reasoning every tick, which is exactly what the project's Full
   Mediation doctrine says should live in the binary instead ("agents never compute or
   hand-write what the binary can own" — already applied to `fab score`, `fab operator
   note`, and B1's tick-start `--diff` snapshot/diff).
2. **Consequence of not fixing it**: the operator's per-tick context cost stays high (N
   per-pane capture round-trips reasoned over in prose — B1 removed the map/diff cost but
   the per-candidate capture-and-scan half of the tick remains manual), and the
   guard/pattern logic has no test coverage of its own — it can silently drift from the
   documented list as the skill file is edited, with no regression signal.
3. **Why this approach**: mirror the pattern already used for the pane family (`fab pane
   map`, `fab pane capture`) — the binary does the deterministic sweep (discovery, guard
   checks, pattern matching) and returns a small structured result (matches + skip
   reasons); the operator (LLM) keeps only the genuinely judgment-requiring steps: the §5
   Answer Model (Routine vs. Strategic classification, answer selection, escalation) and
   the two delivery guards (§3 pre-send gate, re-capture-before-send). The plan
   explicitly calls out that indicator class 4 ("Claude Code permission/tool approval
   prompts") is *not* mechanical — it's prose, not a regex — but is covered by the other
   concrete-pattern classes (2/3/5/7 overlap it almost entirely); novel prompt shapes
   stay operator judgment via manual capture. That split is preserved here: the binary
   implements only the concrete mechanical patterns. This completes the plan's B-phase
   result: an idle-fleet tick = two commands (`tick-start --diff`, `pane questions`),
   near-zero LLM context.

## What Changes

### 1. New command: `fab pane questions`

Added to the `fab pane` command family (`src/go/fab/cmd/fab/pane_questions.go`, new file,
registered in `pane.go`'s `AddCommand` list alongside `map`/`capture`/etc.) — seeded from
the stash commit's finished implementation (see Origin):

```
fab pane questions [--all-sessions] [--panes <id>...] [--json] [--server <name>]
```

- **`--panes <id>...`** — explicit pane IDs to sweep (repeatable / comma-separated via
  `StringSlice`). The primary tick-time source is B1's `candidates:` block from
  `fab operator tick-start --diff` (pane IDs, waiting-first then idle) — but the command
  does not require or assume that source (on-demand sweeps and non-operator callers pass
  any IDs).
- **`--all-sessions`** — discover candidate panes across every tmux session on the target
  server, reusing the SAME discovery machinery `fab pane map --all-sessions` uses
  (`collectPaneRows(sessionAll, ...)` — no new discovery code). Mutually exclusive with
  `--panes` (`cmd.MarkFlagsMutuallyExclusive`, the same pattern `pane map` uses for
  `--session`/`--all-sessions`).
- **No flags** — default to the current tmux session's panes (same `sessionDefault` mode
  and `$TMUX` guard `fab pane map` uses without `--session`/`--all-sessions`).
- **`--json`** — machine-readable output (schema below); default is a short
  human-readable listing.
- **`--server <name>`** — the existing persistent `pane` flag (`-L`), already wired at
  the `paneCmd()` level; consumed by the new subcommand, no new flag. (The plan's
  accepted server-scoping asymmetry: `questions` inherits `--server`; `tick-start` keys
  off the current socket.)

**Population** (discovery modes): when the command performs its own discovery
(`--all-sessions` or the no-flag default), the candidate set is filtered to panes whose
resolved `agent_state` is `waiting` or `idle` — mirroring §5's sweep population. Panes
with unknown state (`—` / empty, no `@rk_agent_state`) are excluded from discovery,
matching B1's `candidates:` exclusion-by-construction. An explicitly-listed `--panes` ID
that turns out not to be waiting/idle is skipped with `state_changed` (below).

### 2. Per-pane sweep procedure

For every candidate pane (in this exact order — a race/short-circuit contract, not just a
list of checks):

1. **Resolve current `agent_state`** via one up-front `collectPaneRows(mode, "", server)`
   call (the B1-extracted helper in `panemap.go` — same `main` package, called directly;
   no `panemap.go` changes needed). Build a `map[paneID]paneRow` from the result.
   - Pane ID not found in the map (dead/gone since it was listed as a candidate) → skip
     with reason **`capture_failed`** (a dead pane cannot be captured either, so this
     folds into the same skip class — plan Tests: "dead pane ⇒ `capture_failed` skip").
   - Pane found but `agent_state` not `waiting`/`idle` (`active` or unknown) → skip with
     reason **`state_changed`** ("a pane no longer waiting/idle at capture time is
     skipped, closing half the candidates-to-capture race" — plan B2 text).
2. **Capture** the pane's last 20 lines via a **new injectable capture seam** — a
   package-level var in `pane_questions.go`, `paneCaptureFn func(server, paneID string,
   lines int) (string, error)`, defaulting to `pane.Capture` (which execs tmux directly
   with no test seam today — the `rkPanesRunner`/`operatorStatePathOverride` precedent).
   Lines fixed at **20** per the plan (not a flag). Capture error (e.g. pane died between
   state-resolve and capture) → skip with reason **`capture_failed`**.
3. **Blank capture guard**: empty or all-whitespace capture → skip with reason
   **`blank_capture`** ("cannot determine").
4. **Claude turn-boundary guard**: either of the last 2 lines matches `^\s*>\s*$` → skip
   with reason **`turn_boundary`** (normal human-turn boundary, not a question).
5. **Indicator scan** (pure function, no I/O — see below): scan for the mechanical
   indicator classes, bottom-most match wins.
   - No match → skip with reason **`no_indicator`**.
   - Match → include in `matches:` with the pane ID, its `agent_state` (as read in step
     1 — "as read at capture time" per the plan), the matched indicator class name, and
     a snippet (the matched line).

### 3. Indicator classes implemented (pure function, table-driven)

Exactly the skill's §5 Question Detection list **minus class 4** (Claude Code
permission/tool prompts — explicitly non-mechanical per the plan, covered by the overlap
with classes 2/3/5/7):

1. Lines ending with `?` — **only the actual last non-empty line** of the capture, under
   120 chars, not starting with `#`, `//`, `*`, `>`, or a leading timestamp.
2. `[Y/n]`, `[y/N]`, `(y/n)`, `(yes/no)` (case-insensitive).
3. `Allow?`, `Approve?`, `Confirm?`, `Proceed?`.
4. `Do you want to...`, `Should I...`, `Would you like...` (case-insensitive).
5. Lines ending with `:` (CLI input prompts).
6. Enumerated options: `[1-9]\)` (e.g. `1)`, `2)`).
7. `Press.*key`, `press.*enter`, `hit.*enter` (case-insensitive).

**Scan algorithm**: split into non-blank lines; walk bottom-most upward; for the actual
last line, test class 1 first, then classes 2–7 in listed order; for every other line,
test only classes 2–7 (class 1 is scoped to "last non-empty line only" per the skill
text). The first (bottom-most) line where any class matches wins, reporting that class's
name as the `indicator`.

Indicator class names in output: `question_mark`, `yes_no`, `action_word`,
`imperative_question`, `colon_prompt`, `enumerated_options`, `press_key`.
<!-- assumed: exact indicator-name strings for JSON output — the plan does not name them; carried over from the stashed implementation as stable, greppable identifiers. -->

### 4. Output schema

```jsonc
// --json
{
  "matches": [
    {
      "pane": "%3",
      "agent_state": "waiting",
      "indicator": "imperative_question",
      "snippet": "Do you want to proceed with this action?"
    }
  ],
  "skipped": [
    { "pane": "%5", "reason": "no_indicator" },
    { "pane": "%7", "reason": "state_changed" }
  ]
}
```

Human (non-JSON) output: one line per match (`pane [agent_state] indicator: snippet`),
one line per skip, then a count line (`N matched, M skipped`). Empty candidate set →
`No candidate panes.` and exit 0, mirroring `fab pane map`'s "No tmux panes found."
convention.

Exit codes: read-only best-effort sweep — the pane-family per-pane scheme (2 = missing,
3 = tmux failure) does NOT apply, because a missing/dead candidate is a normal skip
outcome (`capture_failed`), not a command failure. Exit 0 on a clean run regardless of
match/skip counts; non-zero only on a usage error (e.g. `$TMUX` unset with no
`--all-sessions`/`--panes`) or a hard discovery failure (`collectPaneRows` error — tmux
not running), the same shape `fab pane map` has.

### 5. Skill rewrites

**`src/kit/skills/fab-operator.md` §5 Question Detection** — replace steps 1–4 (manual
`rk mux capture`/raw tmux capture + the operator applying the two guards and the
indicator list from memory) with a single `fab pane questions --panes <ids>` sweep over
the per-tick population (the tick's `candidates:` block from `tick-start --diff` —
population policy unchanged, just re-expressed as the command's input). Steps 5–6 (no
match → stuck detection; match → Answer Model) unchanged. The section gains:

- one line stating the class-4 split: Claude Code permission/tool prompts are **not**
  mechanized — covered in practice by the yes_no/action_word/imperative/enumerated
  classes; novel prompt shapes remain operator judgment via an on-demand `--panes` sweep
  or manual capture.
- the plan-mandated **version-skew fallback line in §4** (one line): if `tick-start
  --diff` or `fab pane questions` errors as unknown flag/command (new skill against an
  old installed binary — bottle lag is a recorded recurring lesson), fall back to the
  flagless tick (full pane map + per-pane manual capture) for the session and report the
  mismatch once. (B1 shipped this line for `--diff`; extend it to name `pane questions`
  if its wording doesn't already cover both.)

The §3 Pre-Send Validation gate and § Sending Auto-Answers' re-capture-before-send guard
are **explicitly retained unchanged** — `questions` output is detection input only,
never a license to send blind (the batch capture is older than today's just-in-time
capture, so the guard matters more, not less). Add one cross-referencing sentence to
§ Sending Auto-Answers stating that `questions` output does not replace them.

**`src/kit/skills/_cli-fab.md` § fab pane** — new `### questions` entry
(constitution-mandated), same format as the existing `map`/`capture` entries: signature,
flags table, JSON field table, skip-reason enum, exit-code notes. Documented as a
first-class skill-facing verb (NOT dispatch-internal, unlike `capture`/`process`/`kill`)
— the deliberate carve-out the plan calls out.

**`src/kit/skills/_cli-agents.md` § Peek** — add a short paragraph: `fab pane questions`
is a **policy-bearing sweep** (fab-operator's own indicator patterns, not raw peek), and
is therefore the named exception to "skill-facing peek rides run-kit's substrate twins" —
it lives in `fab pane`, not `rk mux`, on purpose.

### 6. Memory rewrites (hydrate)

**`docs/memory/runtime/pane-commands.md`** — re-scope the Part 7 fence sentence (its
Design Decision currently ends "...never the fab pane verbs") to raw peek specifically,
naming `questions` as the exception (a policy-bearing sweep, not a peek primitive).

**`docs/memory/runtime/operator.md`** — the B2 half of the plan's hydrate row (B1's
hydrate already landed its half): rewrite the "Hardcoded patterns" Design Constraints
bullet (patterns now live in the binary, tested), update the question-detection policy
passage (the "operator's own detection policy" framing + the steps list) to reflect
detection-via-`fab pane questions` with the Answer Model and delivery guards retained
LLM-side, and extend the Full Mediation decision's scope note (first detection-policy
offload).

## Affected Memory

- `runtime/pane-commands`: (modify) re-scope the Part 7 "never the fab pane verbs" fence
  to raw peek, naming `fab pane questions` as the exception.
- `runtime/operator`: (modify) B2 half of the plan's hydrate row — "Hardcoded patterns"
  constraint bullet, detection-policy passage rewired to the binary sweep, Full
  Mediation scope extension. (An earlier drafting deferred this file because B1 was
  unmerged; B1 is now on main, so the deferral reason is void.)

## Impact

- **New file**: `src/go/fab/cmd/fab/pane_questions.go` (seeded from stash `1749af37`;
  command + capture seam + pure guard/indicator scanner).
- **New file**: `src/go/fab/cmd/fab/pane_questions_test.go` (table-driven tests over the
  pure scanner + guards; capture-seam-stubbed command tests; NOT in the stash — written
  fresh).
- **Modified**: `src/go/fab/cmd/fab/pane.go` (register the subcommand; the stash carries
  this as a 3-line diff).
- **No changes** to `panemap.go` (calls B1's existing `collectPaneRows`),
  `operator_tick_start.go`, `operator_note.go`, or `internal/pane` (the capture seam
  lives in the new file).
- **Skill docs**: `src/kit/skills/fab-operator.md` (§5 rewrite + §4 skew-fallback line),
  `src/kit/skills/_cli-fab.md` (new questions entry), `src/kit/skills/_cli-agents.md`
  (§ Peek carve-out paragraph).
- **Memory**: `docs/memory/runtime/pane-commands.md`, `docs/memory/runtime/operator.md`
  (per Affected Memory).
- No config schema changes, no new `fab/project/config.yaml` fields, no operator state
  file schema changes.

## Open Questions

(none — the plan section is fully specified, all predecessors are merged, and the one
intentional implementation choice — exact JSON indicator-name strings — is carried from
the stashed implementation as a Confident assumption.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab pane questions` lives under the existing `fab pane` group, new file `pane_questions.go`, registered in `pane.go` | Plan places it there explicitly ("it lives in fab, not rk"); matches every pane-subcommand's file/registration pattern | S:90 R:90 A:95 D:95 |
| 2 | Certain | Capture fixed at 20 lines, not a flag | Plan states "capture last 20 lines" as fixed behavior, matching §5's existing `-l 20` capture | S:95 R:95 A:95 D:95 |
| 3 | Certain | Skip reasons exactly `turn_boundary`, `blank_capture`, `no_indicator`, `capture_failed`, `state_changed` | Verbatim from the plan's B2 and Tests sections | S:100 R:90 A:100 D:100 |
| 4 | Certain | Indicator class 4 (Claude Code permission/tool prompts) excluded from the mechanical scanner | Plan states it is "not mechanical" and covered by classes 2/3/5/7; three reviewers independently | S:95 R:85 A:95 D:90 |
| 5 | Certain | Apply seeds the Go implementation from stash commit `1749af37` rather than rewriting | User instruction in the /fab-new invocation ("review it for reusable code"); verified compatible with current main (`collectPaneRows`, `pane.Capture`, `AgentState*`, `paneRow` all exist unchanged) | S:90 R:85 A:90 D:90 |
| 6 | Confident | Population: `--panes` (explicit, tick-time fed by B1's `candidates:`), `--all-sessions` (discovery filtered to waiting/idle), current-session default (same filter) | Plan gives the flag signature; B1 is merged so `candidates:` is the live tick source; the waiting/idle filter mirrors §5's population policy and B1's exclusion-by-construction | S:75 R:75 A:85 D:80 |
| 7 | Confident | Discovery reuses `collectPaneRows` directly (same `main` package, no exported API, no panemap.go changes) | B1 extracted exactly this helper for reuse ("panemap.go's row collection is extracted into a collectPaneRows-style helper"); the stashed code already does this | S:70 R:80 A:90 D:80 |
| 8 | Confident | Capture seam is a package-level var `paneCaptureFn` in `pane_questions.go` wrapping `pane.Capture` | Plan cites the `rkPanesRunner`/`operatorStatePathOverride` precedent by name; keeps the seam local rather than modifying `internal/pane` | S:70 R:85 A:85 D:75 |
| 9 | Confident | Exit codes: 0 on any clean sweep (matches/skips are data); non-zero only on usage error or hard discovery failure — `pane map`'s shape, not the pane-family 2/3 scheme | `questions` sweeps multiple panes like `map` (no single target to be "missing"); a dead candidate is an expected skip (`capture_failed`), not a failure | S:60 R:80 A:80 D:75 |
| 10 | Confident | JSON indicator-name strings `question_mark`/`yes_no`/`action_word`/`imperative_question`/`colon_prompt`/`enumerated_options`/`press_key` | Plan doesn't name them; carried from the stashed implementation — stable, greppable, consistent with the skill's class descriptions | S:50 R:90 A:70 D:70 |
| 11 | Confident | Memory scope is BOTH `runtime/pane-commands` (fence re-scope) AND `runtime/operator` (B2 half: hardcoded-patterns bullet, detection-policy passage, Full Mediation scope) | Plan's hydrate rows name both; the earlier drafting's deferral of operator.md existed only because B1 was unmerged — B1 is now on main and its hydrate landed its own half, leaving the B2-specific sentences (verified present at operator.md:175-182, :275) | S:75 R:70 A:80 D:75 |
| 12 | Confident | §4 version-skew fallback: extend/verify the skew line to cover `fab pane questions` alongside `tick-start --diff` | Plan's "one line in §4" names both commands; B1 shipped the line for `--diff`, so this change verifies wording and extends only if needed | S:65 R:85 A:80 D:80 |

12 assumptions (5 certain, 7 confident, 0 tentative, 0 unresolved).
