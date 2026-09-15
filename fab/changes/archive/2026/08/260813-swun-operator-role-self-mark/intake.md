# Intake: Operator Role Self-Mark

**Change**: 260813-swun-operator-role-self-mark
**Created**: 2026-08-13

## Origin

Backlog item `[swun]` (fab/backlog.md, 2026-08-13), one-shot invocation via `/fab-new swun`:

> fab-operator skill self-marks its tmux window as operator at startup — fail-silent 'command -v rk && rk role operator' step in the skill preamble. Companion to run-kit @rk_role window option: operator row renders pinned directly under the sidebar SESSIONS header (outside its session group), server-scoped radio (one operator per tmux server); rk owns the option contract + radio semantics, fab is just the producer

## Why

1. **The pain point**: The operator is the server-wide coordination singleton, but in run-kit's web dashboard its window is indistinguishable from every ordinary agent window — a user viewing sessions remotely must hunt through session groups to find the pane that answers questions and routes work. run-kit is adding an `@rk_role` tmux window option so a window marked `operator` renders pinned directly under the sidebar SESSIONS header (outside its session group), with server-scoped radio semantics (one operator per tmux server — matching fab's own one-operator-per-server model). For that rendering to happen, something must *produce* the mark, and only the operator itself knows when it starts and which window it owns.

2. **If we don't**: the dashboard's operator pinning ships on run-kit's side with no producer — the feature is inert for fab operators, and the primary consumer of the operator (a user coordinating remotely through the dashboard) keeps hunting.

3. **Why this approach**: a fail-silent one-line startup step in the skill is the minimal producer. The division of ownership is explicit in the backlog: **rk owns the option contract, the pinned rendering, and the radio semantics; fab is just the producer** — so fab ships no Go code, no state, no conflict handling, just the `rk role operator` call at the moment the operator identity is established. This mirrors the existing fab-owned `rk notify` usage pattern (gated, fail-silent, one line).

## What Changes

### 1. New startup step in `src/kit/skills/fab-operator.md` §2 Startup

Add a role self-mark step, placed immediately after the **Tmux Gate** (the mark is a tmux window option — meaningless without tmux, and the gate has just guaranteed `$TMUX` is set) and before the wt Gate:

```markdown
### Role Mark

Mark this tmux window as the operator for run-kit's dashboard (`@rk_role` window
option — rk owns the option contract, pinned rendering, and the one-operator-per-server
radio semantics; fab is only the producer):

​```bash
command -v rk >/dev/null 2>&1 && rk role operator >/dev/null 2>&1 || true
​```

Fail-silent by contract (`_preamble.md` § Run-Kit (rk) Reference): rk absent, or an
installed rk predating the `role` subcommand, MUST NOT surface an error or block startup.
```

Exact wording may be tightened during apply; the load-bearing parts are:

- **The command**: `command -v rk >/dev/null 2>&1 && rk role operator >/dev/null 2>&1 || true` — the `command -v` gate per the `_preamble.md` rk rule, PLUS full output/exit suppression on the `rk role` call itself. The suppression is load-bearing, not cosmetic: probed 2026-08-13, the installed run-kit does **not yet ship** a `role` subcommand (`Error: unknown command "role"`), so without suppression every operator startup on a current rk prints an error. Version skew must degrade to a no-op.
- **Placement**: after Tmux Gate, before wt Gate.
- **Idempotent** (Constitution III): re-running re-sets the same option; a restarted operator in the same window is a no-op.
- **No unmark step**: the operator has no clean exit hook, and radio/staleness semantics are rk-owned — a later operator marking itself on the same server is rk's radio to resolve.

### 2. Pointer in `src/kit/skills/_cli-external.md` § rk (run-kit)

That section is the declared home of fab-owned rk usage and currently enumerates only the escalation `rk notify` send. Per the owner-or-pointer rule (code-quality.md), add a one-line **pointer** — not a restatement of the command — noting the second fab-owned usage: the operator's startup role self-mark, owned by `fab-operator.md` §2 Startup. Sweep the enumerations that list fab-owned rk content:

- `_cli-external.md` frontmatter `description:` ("…the escalation rk-notify usage…")
- `_cli-external.md` line ~29 in-body summary of what the file carries
- `_cli-external.md` § rk (run-kit) intro

### 3. Sibling sweep at apply

- `docs/specs/skills.md` § `/fab-operator` — aggregate spec restating per-skill facts (a known sibling-sweep class); update its startup skeleton only if it enumerates the startup gates/steps.
- `docs/memory/runtime/operator.md` — the operator memory file documents Startup in detail (Context Loading, wt Gate); hydrate adds the role-mark step there. Its `description:` frontmatter mentions "the operator-role launcher" — distinct concept (the `fab operator` launcher command); do not conflate.

**Non-goals**: no change to the `fab operator` Go launcher, no new `fab` subcommand, no tests (markdown-only), no migration (no user data restructured), nothing on the rk side (companion feature ships in run-kit separately).

## Affected Memory

- `runtime/operator`: (modify) add the startup role self-mark step (fail-silent `rk role operator` after the Tmux Gate) to the Startup documentation; note the rk-owns-contract / fab-is-producer split

## Impact

- `src/kit/skills/fab-operator.md` — §2 Startup gains the Role Mark step (primary edit)
- `src/kit/skills/_cli-external.md` — § rk pointer + description-enumeration sweep
- `docs/specs/skills.md` — sweep-check only (§ `/fab-operator` startup skeleton)
- `docs/memory/runtime/operator.md` — hydrate
- No Go code, no tests, no migrations. Deployed `.claude/skills/` copies regenerate via `fab sync` — never edited directly.

## Open Questions

- None.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is skill-markdown only — no Go/`fab operator` launcher change | Backlog states "step in the skill preamble"; rk owns option contract + radio semantics, fab is just the producer | S:90 R:85 A:90 D:90 |
| 2 | Certain | Fully suppress the `rk role` call's output and exit status (stderr/stdout redirect plus a true fallback) beyond the `command -v` gate | Probed 2026-08-13: installed rk predates the `role` subcommand and errors on it; the `_preamble.md` fail-silent rule extends to version skew | S:75 R:95 A:85 D:80 |
| 3 | Confident | Placement: new step in §2 Startup, after the Tmux Gate, before the wt Gate | Window option requires tmux, which the gate just guaranteed; one obvious spot, trivially movable | S:70 R:95 A:75 D:70 |
| 4 | Confident | Invocation is `rk role operator` verbatim per the backlog | rk has not shipped the subcommand yet, so the final surface could differ; fail-silent suppression covers any skew until it lands | S:80 R:90 A:60 D:70 |
| 5 | Confident | No unmark/cleanup step and no radio-conflict handling in fab | Backlog assigns radio semantics to rk; operator has no clean exit hook; a stale mark is rk's staleness concern | S:65 R:90 A:75 D:65 |
| 6 | Confident | `_cli-external.md` § rk gets a pointer (not a command restatement) to the new fab-owned usage | Owner-or-pointer rule in code-quality.md; keeps the section's fab-owned-usage enumeration accurate without a drift-prone copy | S:55 R:90 A:80 D:60 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).
