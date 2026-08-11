# Intake: `fab pane send --answer` Mode

**Change**: 260811-answ-pane-send-answer-mode
**Created**: 2026-08-12

## Origin

One-shot `/fab-new answ` from backlog entry `[answ]` (2026-08-11):

> `fab pane send --answer` mode — the idle-gate refuses `waiting`, but `waiting` is the operator auto-answer's PRIMARY target (fab-operator.md § Auto-Nudge names it the first-class trigger), so the operator routes around the binary with raw `tmux send-keys` (fab-operator.md:23/375) while `_cli-agents.md:95` says "prefer `fab pane send` … let the binary hold the gate" — the gate refuses its main legitimate use case and never actually gates. Add `--answer`: permits `waiting`, still refuses `active`, keeps pane-exists validation; `--force` stays the skip-everything override. Rewire the operator's send paths + `_cli-agents.md` § Pre-Send Validation onto the gated binary. OPEN DESIGN QUESTION (settle at intake): unknown-state handling under `--answer` (plain send warns-and-proceeds on unknown — same posture, or stricter?). Surfaces: `pane_send.go` (idleGate) + tests, `_cli-fab.md` § fab pane send, fab-operator/_cli-agents skill sources (SPEC-mirror tree retired in constitution 1.6.0 — no mirror work), runtime/pane-commands.md + operator.md at hydrate. Companion of the additive pane/dispatch surface-completion change (kill/await/--json/exit-codes) — kept separate because this one changes a refusal contract.

The flagged open design question was asked at intake (SRAD, interactive): the user chose **same posture — warn and proceed on unknown state under `--answer`**, consistent with plain send.

## Why

1. **The pain point**: `fab pane send`'s three-state gate (`idleGate`, `src/go/fab/cmd/fab/pane_send.go:88-96`) refuses both `active` and `waiting`. But `waiting` — an agent blocked on a permission prompt / menu / elicitation — is the *primary target* of the operator's auto-answer flow (`fab-operator.md` § Auto-Nudge names `waiting` the first-class, event-driven trigger). The only bypass is `--force`, which also skips the `active` check — exactly the state a send *should* never hit unattended. So the operator's send paths use raw `tmux send-keys` (`fab-operator.md:23`, `:375`) while `_cli-agents.md:95` simultaneously instructs "prefer `fab pane send` … let the binary hold the gate". The gate refuses its main legitimate use case and, because callers route around it, never actually gates anything.

2. **If we don't fix it**: the contradiction between the two skill files persists; every operator answer-send bypasses pane-existence validation, the exit-code family scheme, and the literal-text (`-l`) safety that `fab pane send` provides; and the `active`-state protection is effectively decorative on the operator path.

3. **Why this approach**: a mode flag (`--answer`) is the minimal change that splits the gate's two refusal cases — "never interrupt a working agent" (`active`, still refused) vs "this send IS the answer the blocked agent is waiting for" (`waiting`, now permitted). Alternatives rejected: making `--force` softer would remove the only skip-everything escape hatch; changing the default gate to permit `waiting` would let ordinary command-routing sends cut across a pending human answer. This change deliberately alters a refusal contract, which is why it was kept separate from the additive pane/dispatch surface-completion companion change (yxyi, PR #589).

## What Changes

### 1. `--answer` flag on `fab pane send` (Go binary)

`src/go/fab/cmd/fab/pane_send.go`: add `--answer` bool flag. Gate matrix after the change:

| Agent state | plain send | `--answer` | `--force` |
|-------------|-----------|------------|-----------|
| `idle` | send | send | send |
| `waiting` | refuse (exit 1) | **send** | send |
| `active` | refuse (exit 1) | **refuse (exit 1)** | send |
| unknown (`—`) | warn + send | **warn + send** (same posture — settled at intake) | send |

- Pane-existence validation (step 1, exit 2/3 family scheme) is unchanged and applies in every mode — `--answer` does not touch it, `--force` continues to skip only the state check.
- `--force` stays the skip-everything override; when both flags are given, `--force` semantics win (the state check is skipped entirely).
- The refusal under `--answer` keeps the existing error shape and exit 1, naming the state (e.g. `agent in pane %5 is not idle (state: active)` — exact wording may be adjusted so the `--answer` refusal reads correctly, but the exit-code contract is fixed).
- Implementation: extend `idleGate` (or a sibling pure function) with the mode so the matrix stays unit-testable without cobra/tmux plumbing; extend `pane_send_test.go` accordingly (Go changes ship tests — code-review must-fix rule).

### 2. `_cli-fab.md` § fab pane send

Document the new flag and the gate matrix above (CLI ⇒ docs constraint). The section's validation-pipeline prose (`_cli-fab.md:551-553`) gains the `--answer` branch: `waiting` → send under `--answer`; `active` → refuse in both non-force modes; unknown → warn-and-send in both.

### 3. Operator send paths (`src/kit/skills/fab-operator.md`)

Rewire onto the gated binary:

- **Auto-answer path** (§ Auto-Nudge / answer delivery, `fab-operator.md:375`): replace raw `tmux send-keys` with `fab pane send --answer <pane> <text>` for text answers. The surrounding choreography (re-capture before send, abort if output changed, delivery probe on no-resume) is unchanged.
- **Routed-command path** (§3 pre-send policy, `fab-operator.md:103-106`): policy unchanged — the operator still asks the user before sending to a non-`idle` agent; after explicit confirmation targeting a `waiting` agent, the send uses `--answer` instead of `--force`/raw keys, so pane-existence validation and the `active` refusal still hold at send time.
- The § header prose (`fab-operator.md:23` "routes commands via `tmux send-keys`") is updated to name the gated binary.
- **Carve-out that stays raw**: answers that are key names rather than literal text (bare Enter, arrow keys, `C-c`) cannot ride `fab pane send` (it sends literal text via `send-keys -l` precisely to avoid key-name interpretation). Those remain raw `tmux send-keys`, called out explicitly as the one remaining raw path for delivered workers.

### 4. `_cli-agents.md` § Pre-Send Validation

Update the section (around `_cli-agents.md:95`) so its "prefer `fab pane send` … let the binary hold the gate" instruction is actually satisfiable for answer-sends: plain send for command routing to idle agents, `--answer` for answering a detected prompt on a `waiting` agent, `--force` as the deliberate skip-everything override. Same key-name carve-out note as above.

### 5. Explicitly out of scope

- No SPEC-mirror work — the `docs/specs/skills/` mirror tree was retired in constitution 1.6.0.
- No migration — no user data (`config.yaml`, `.status.yaml`, archive layout) is restructured.
- No change to `fab dispatch deliver`'s choreography — it builds on `fab pane deliver`'s own readiness gate, not on `fab pane send`'s idle gate.

## Affected Memory

- `runtime/pane-commands.md`: (modify) `fab pane send` gate contract gains the `--answer` mode and the four-row state matrix
- `runtime/operator.md`: (modify) operator send paths documented as riding the gated binary (`--answer` for auto-answer), raw `tmux send-keys` reduced to the key-name carve-out

## Impact

- `src/go/fab/cmd/fab/pane_send.go` + `src/go/fab/cmd/fab/pane_send_test.go` — flag, gate matrix, tests
- `src/kit/skills/_cli-fab.md` § fab pane send — flag surface + validation pipeline prose
- `src/kit/skills/fab-operator.md` — lines ~23, ~103-106, ~375 region (send paths, pre-send policy wiring)
- `src/kit/skills/_cli-agents.md` § Pre-Send Validation — gate description and preference rule
- Behavior-claim sweep: grep for the old contract phrases (e.g. "not idle", "refuses `waiting`", "prefer `fab pane send`") across `src/kit/skills/` and `docs/` — the twin/aggregate sweep classes apply (code-quality.md § Sibling Sweeps); include test comments and contrastive phrases per the recurring-lessons sweep taxonomy
- No new binaries, no config schema change, no migration

## Open Questions

*(none — the one flagged question, unknown-state handling under `--answer`, was asked and resolved at intake: warn-and-proceed)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `--answer` permits `waiting` (and `idle`), still refuses `active`; pane-existence validation unchanged | Backlog entry states this verbatim — it is the change's definition | S:95 R:90 A:95 D:95 |
| 2 | Certain | Unknown state under `--answer` warns-and-proceeds (same posture as plain send) | Asked — user chose same posture; keeps uninstrumented-pane auto-answer on the gated binary instead of forcing `--force` | S:90 R:85 A:95 D:90 |
| 3 | Confident | When `--answer` and `--force` are both given, `--force` wins (state check skipped entirely) | Backlog fixes `--force` as the skip-everything override; superset semantics is the one non-surprising combination | S:65 R:90 A:85 D:70 |
| 4 | Confident | `--answer` refusal of `active` keeps exit 1 + state-naming error shape; exact message wording may adjust | Existing family scheme (2 = pane missing, 3 = other tmux failure, 1 = gate refusal) is established contract; only prose is flexible | S:70 R:85 A:80 D:70 |
| 5 | Confident | Operator routed-command confirm policy unchanged — `--answer` replaces the *mechanism* after confirmation, not the ask-first policy | Backlog says "rewire the send paths", not "change the policy"; §3 item 2's confirm-before-send is operator judgment, orthogonal to the gate | S:75 R:80 A:80 D:75 |
| 6 | Confident | Key-name answers (bare Enter, arrows, `C-c`) remain raw `tmux send-keys` — `fab pane send` deliberately sends literal text only | `send-keys -l` design comment in pane_send.go:59-61; expressing key names would require new flag surface out of this change's scope | S:55 R:85 A:60 D:55 |

6 assumptions (2 certain, 4 confident, 0 tentative, 0 unresolved).
