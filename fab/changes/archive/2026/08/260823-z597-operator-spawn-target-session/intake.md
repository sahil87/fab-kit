# Intake: Operator Spawn Target-Session Derivation

**Change**: 260823-z597-operator-spawn-target-session
**Created**: 2026-08-24

## Origin

Synthesized from a `/fab-discuss` session and dispatched promptless via `/fab-proceed` (create-intake, `{questioning-mode} = promptless-defer`). The user observed a real failure and accepted a five-part design in discussion; this intake captures that design faithfully.

> Title direction: operator spawn target-session derivation — pass `-t` on every operator `tmux new-window`.
>
> Problem (real failure observed): `fab-operator.md` §6 Spawning an Agent step 6 opens worker tabs via `tmux new-window -n "»<wt>" -c <worktree-path> "<spawn_cmd> '<command>'"` with no `-t <session>` target, and `_cli-agents.md` § Spawn Composition's raw form likewise never mentions session targeting. `tmux new-window` without `-t` resolves against the ambient session — the session of the pane running the command. The operator now runs in its own dedicated tmux session, distinct from the work session(s) holding agent windows, so spawned worker windows silently land in the operator's session where the user never sees them (findable only via `fab pane map --all-sessions`), and enrollment records the wrong session context. This affects all three spawn paths, which all route through §6 step 6: worker tabs, watch spawns (§7 step 4), and autopilot spawns. The gap is structural: §6 has an explicit "Establish target repo" step with a "never rely on the operator's CWD" rule, but no parallel "establish target session" step exists anywhere, despite §1 declaring `(session, repo, pane)` addressing and §8 declaring the operator spans multiple sessions on one server.

## Why

1. **The pain point**: The operator runs as a singleton window in its own dedicated tmux session, while the agent windows it coordinates live in one or more separate work sessions on the same server. Every operator spawn path (§6 worker tabs, §7 step 4 watch spawns, autopilot spawns — all routing through §6 step 6) issues `tmux new-window` with **no `-t` target**, so tmux resolves the window against the *ambient* session — the operator's own. Spawned worker windows silently land in the operator's session where the user never sees them (they surface only via `fab pane map --all-sessions`), and §6 step 7's enrollment (`fab operator enroll … --session <name>`) records the wrong session context.

2. **The consequence if unfixed**: Every spawn from a dedicated-session operator misplaces its worker window — a silent, systematic failure of the operator's core function. The user loses visibility into spawned agents; the status frame's session grouping and the `(session, repo, pane)` addressing (§1) carry wrong data at enrollment time.

3. **Why this approach**: The gap is structural, not a one-line typo. §6 already models the exact analogous problem for repos — step 1 "Establish target repo" with the "never rely on the operator's CWD" rule (currently stated at step 2, Create worktree) — but no parallel "establish target session" step exists anywhere, despite §1 declaring `(session, repo, pane)` addressing and §8 declaring the operator spans multiple sessions on one server. The fix mirrors the existing pattern: an explicit derivation step plus a MUST-pass-`-t` rule, with mechanical hardening (`-P -F` print-back) that converts silent misplacement into a checked output. The `-P -F` precedent already exists in fab's own Go spawn path (`src/go/fab/internal/pane/create.go` uses `-P -F '#{pane_id}'` on every `new-window`/`split-window`).

## What Changes

Markdown/skill-only. Canonical sources under `src/kit/skills/` (never `.claude/skills/` — gitignored deployed copies). No Go changes.

### 1. `fab-operator.md` §6 — new spawn-sequence step "Establish target session"

Insert a new step parallel to step 1 "Establish target repo" (natural placement: new step 2, renumbering current steps 2–7 → 3–8; downstream references to §6 step numbers must be swept — e.g. §6 step 5's repo-targeting rule is cited by `docs/memory/runtime/operator.md` and `_cli-external.md`). The step derives the target session via a fallback ladder:

a. **Live derivation (primary)**: the session holding existing monitored agents for the target repo, re-verified from the current tick snapshot / `fab pane map --all-sessions` (per §1 Re-derive-state; the persisted `session` field on monitored entries is context-not-identity — a display/context dimension, never a join key — and must not be trusted stale; the snapshot rows carry per-pane `repo` and are keyed on pane IDs).
b. **Else**: the session holding any monitored agent (the common single-work-session case).
c. **Else**: a §8 natural-language session-scoped setting ("spawn into session {name}") if the user set one.
d. **Cold start (no signal at all)**: ask the user once and keep the answer as the §8 setting for the session.

The normative rule the step writes down: the operator MUST pass `-t "<session>:"` on every `new-window`; **the ambient session is never an implicit target** — the exact mirror of the existing "never rely on the operator's CWD" rule.

### 2. `fab-operator.md` §6 step 6 (currently) — mechanical hardening of the spawn command

The open-agent-tab command becomes:

```sh
tmux new-window -t "<session>:" -P -F '#{session_name} #{pane_id}' -n "»<wt>" -c <worktree-path> "<spawn_cmd> '<command>'"
```

and the printed `#{session_name}` / `#{pane_id}` values feed straight into the enrollment step's `fab operator enroll --pane <pane-id> --session <name>` (both flags are required by the enroll contract in `_cli-fab.md` § fab operator enroll). This closes a latent documentation gap — the skill never said how enrollment learns the pane/session post-spawn — and verifies where the window actually landed, converting silent misplacement into a checked output. Precedent: fab's own Go spawn path already uses `-P -F '#{pane_id}'` (`src/go/fab/internal/pane/create.go`).

### 3. `_cli-agents.md` § Spawn Composition — mechanical caveat on the raw form

Add a short caveat to the raw form (`tmux new-window -n "<name>" -c "<dir>" "<composed-cmd> '<initial-prompt>'"`): `new-window` without `-t` targets the **ambient** session; a caller whose session differs from where the window should land must pass `-t '<session>:'`. Mechanics only — no operator policy (the fallback ladder stays in `fab-operator.md` per that file's agent-primitives-vs-orchestration scope boundary and the owner-or-pointer rule in `fab/project/code-quality.md`).

### 4. `fab-operator.md` §8 Settings — new setting row

Add a row for the spawn-session setting to the §8 Settings table (session-scoped, natural-language overridable, like the existing rows — e.g. Setting: "Spawn target session", Default: derived per §6's ladder, Override: "spawn into session {name}"). Like the other rows it resets on `/clear` or session restart and is not an operator-state-file field.

### 5. Sibling sweep (per `fab/project/code-quality.md` § Sibling Sweeps)

- `_cli-external.md` § tmux — the `new-window` command row (line ~178: `tmux new-window -n <name> -c <dir> "<cmd>"`) and the `new-window` usage-note bullet (line ~185, which points at Spawn Composition and the operator's window-marker policy): keep them pointing; update the command form/pointer so it does not contradict the `-t` rule.
- `docs/memory/runtime/operator.md` and `docs/memory/runtime/agent-primitives.md` — the runtime-domain memory files documenting the operator skill's spawn sequence and § Spawn Composition's raw form (`agent-primitives.md` line ~33 reproduces the raw `tmux new-window` shape verbatim). Memory updates land at hydrate.
- `docs/specs/skills.md` — **checked during intake: it does NOT restate the spawn command** (no `new-window` occurrence; its `/fab-operator` section describes spawning at behavior level only). No edit expected there; the sweep re-verifies.
- **Deliberate non-target**: `docs/specs/harness-adapters.md` (~line 231) and `docs/memory/runtime/dispatch.md` document the *dispatch adapter's* `new-window` fallback shape — a different mechanism where the ambient/`$TMUX_PANE`-based resolution is correct by design (the dispatching agent already sits in the work session). The sweep must not "fix" those.

### Constraints

- Markdown/skill-only change; canonical sources are `src/kit/skills/*.md` — never edit `.claude/skills/`.
- Owner-or-pointer rule: the fallback-ladder policy is owned by `fab-operator.md`; `_cli-agents.md` carries only the mechanical `-t` fact; `_cli-external.md` keeps pointing.
- No Go changes in this change.

### Out of scope (agreed backlog-idea follow-up, not this change)

Go-side siblings sharing the ambient-session assumption: `fab pane open`'s new-window shape (`src/go/fab/internal/pane/create.go`), `fab batch new`/`fab batch switch` (`src/go/fab/cmd/fab/batch_new.go:141`, `batch_switch.go:141`). The `fab operator` launcher itself (`operator.go:106`) is deliberately ambient and fine.

## Affected Memory

- `runtime/operator.md`: (modify) the operator skill's spawn-sequence documentation gains the "Establish target session" step, the `-t`/`-P -F` hardened spawn form, the enroll feed, and the §8 setting row
- `runtime/agent-primitives.md`: (modify) § Spawn Composition documentation gains the raw form's ambient-session `-t` caveat (mechanics only)

## Impact

- **Files (apply)**: `src/kit/skills/fab-operator.md` (§6 new step + step renumbering + hardened step-6 command + §8 Settings row), `src/kit/skills/_cli-agents.md` (§ Spawn Composition caveat), `src/kit/skills/_cli-external.md` (§ tmux new-window row + usage-note bullet).
- **Files (hydrate)**: `docs/memory/runtime/operator.md`, `docs/memory/runtime/agent-primitives.md` (+ regenerated memory indexes).
- **Behavior contract**: operator spawn behavior changes (all three spawn paths gain explicit session targeting); no CLI/Go surface changes, so no `_cli-fab.md` command-signature updates and no test updates are triggered by the CLI⇒docs+tests rule.
- **Scale**: small — three skill files, two memory files, no code.

## Open Questions

- None. (Promptless dispatch: no decisions scored below the Unresolved threshold — see Assumptions.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | New §6 "Establish target session" step with the 4-rung fallback ladder (live derivation for the target repo → any monitored agent's session → §8 setting → cold-start ask-once), plus the MUST-pass `-t "<session>:"` / never-ambient rule | Discussed — user accepted this recommendation verbatim in `/fab-discuss` | S:95 R:85 A:90 D:90 |
| 2 | Certain | §6 spawn command hardened to `tmux new-window -t "<session>:" -P -F '#{session_name} #{pane_id}' …`, printed values fed into `fab operator enroll --pane/--session` | Discussed — user accepted; verified against the enroll contract (`--pane`/`--session` required) and the Go `-P -F` precedent in `internal/pane/create.go` | S:90 R:85 A:95 D:90 |
| 3 | Certain | `_cli-agents.md` § Spawn Composition gets a mechanics-only `-t` caveat on the raw form; the ladder/policy stays in `fab-operator.md` | Discussed — user accepted; matches the file's stated scope boundary and code-quality.md's owner-or-pointer rule | S:90 R:90 A:95 D:90 |
| 4 | Certain | §8 Settings gains a session-scoped, NL-overridable spawn-session row (resets on `/clear`, not an operator-state-file field) | Discussed — user accepted; shape matches the existing four rows | S:90 R:90 A:90 D:85 |
| 5 | Certain | Sweep class = `_cli-external.md` § tmux new-window row + bullet (~178/185), `docs/memory/runtime/operator.md` + `agent-primitives.md`; `docs/specs/skills.md` verified clean at intake (no spawn-command restatement); dispatch-adapter docs (`harness-adapters.md`, `runtime/dispatch.md`) are deliberate non-targets | Discussed sweep class, verified by repo-wide `new-window` grep during intake | S:85 R:90 A:90 D:85 |
| 6 | Certain | Markdown/skill-only; edit canonical `src/kit/skills/*.md` only; Go siblings (`fab pane open`, `fab batch new/switch`) deferred to a backlog-idea follow-up; `operator.go:106` launcher stays ambient by design | Explicitly agreed constraints and out-of-scope list from the discussion | S:95 R:90 A:95 D:95 |
| 7 | Confident | Placement/renumbering: insert as new step 2 (directly after step 1 "Establish target repo"), renumber current 2–7 → 3–8, and sweep downstream §6 step-number citations (e.g. "step 5" repo-targeting references in `runtime/operator.md`, `_cli-external.md`) | Not explicitly discussed; "parallel to Establish target repo" implies adjacency, and both consumers of the session value (open-tab, enroll) come later; renumbering fallout is mechanical and reversible | S:60 R:90 A:80 D:65 |
| 8 | Confident | Ladder rung (b) applies when exactly one session holds monitored agents; with multiple candidate sessions and none matching the target repo, fall through to rungs (c)/(d) rather than guessing | Discussion framed rung (b) as "the common single-work-session case"; ambiguity across multiple sessions has no safe default, and falling through is the conservative reading | S:55 R:80 A:70 D:55 |
| 9 | Confident | The `-P -F` print-back is a verification feed, not a compare-and-escalate branch: with `-t "<session>:"` tmux errors if the session is absent (surfaced per the operator's normal error handling), and the printed `#{session_name}` is recorded via enroll as the landed-session confirmation | Discussion said "verifies where the window actually landed, converting silent misplacement into a checked output"; tmux semantics make a landed-elsewhere-silently case unreachable once `-t` is passed | S:55 R:85 A:75 D:60 |
| 10 | Confident | Cold-start rung (d) "ask the user once" applies as written on user-initiated spawns; an unattended tick spawn (watch/autopilot) with no signal at any rung escalates per the operator's existing escalation path instead of silently picking the ambient session | The never-ambient rule is the design's one absolute; the operator already has a notification/escalation channel for exactly this shape (§5) | S:50 R:80 A:70 D:55 |

10 assumptions (6 certain, 4 confident, 0 tentative, 0 unresolved).
