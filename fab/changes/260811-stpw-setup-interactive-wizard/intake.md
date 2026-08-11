# Intake: fab setup Interactive Wizard

**Change**: 260811-stpw-setup-interactive-wizard
**Created**: 2026-08-12

## Origin

One-shot invocation: `/fab-new stpw`, resolving backlog entry `[stpw]` (2026-08-11):

> `fab setup` interactive wizard — C2 of the setup series (C1 = the `fab setup check` probe/doctor, pgbq, in flight 2026-08-11; bare `fab setup` prints "Yet to be implemented" until this C2 wizard takes the seat). Interview layer over C1's probe: scope banner (system tier by default, `--project` to retarget), 4-question default path — agent.session, agent.workers (options filtered to DETECTED providers, annotated), dispatch.mode (filtered by $TMUX viability, ladder semantics stated) — plus an opt-in advanced section (agent.profiles.operator/review, dispatch.column_width, dispatch.reap_done; skip advanced questions whose current value equals the built-in default and was never overridden). Invariants: capability is detected never asked; each question defaults to the current effective value + origin (all-Enter run = zero writes, idempotent); diff-before-write summary; writes ONLY via existing `fab config set` semantics (surgical, presence=intent, never whole-file); question footers point at `fab config explain <key>` (owner-or-pointer). Decisions settled in 2026-08-11 discussion: name stays `fab setup` with the /fab-setup skill delegating its config-interview portion (flow change ⇒ SPEC mirror update per constitution 1.5.0); bare stdin prompts, no TUI dependency (Constitution I); non-interactive parity via `--defaults`/flags. Deferred to C3: pane warm-up/trust seeding, per-provider fill editing, section entry points (`fab setup agents|dispatch`). Prereq verified FIXED: `show --origin` renders system-tier provider rows on 2.19.4 (#572).

The entry records decisions settled in a 2026-08-11 discussion; those are encoded below as Certain/Confident assumptions. Verified at intake time: C1 has **merged** (PR #588, `feat: fab setup check environment doctor`) — `src/go/fab/cmd/fab/setup.go` ships `fab setup check` with all probing in the reusable `internal/setupcheck` package, and bare `fab setup` prints the placeholder this change replaces.

## Why

1. **The pain point**: configuring fab's agent/dispatch preferences today requires knowing the key names (`agent.session`, `agent.workers`, `dispatch.mode`, …), their valid values, which tier to write them to, and running `fab config set --system <key> <value>` once per key. C1's doctor *detects* the environment (provider roster, capabilities, tmux, viable dispatch modes) but is read-only by hard invariant — it can diagnose "your configured worker provider's binary is missing" but cannot help fix it. The bare `fab setup` seat has been a "Yet to be implemented" placeholder since C1 shipped, explicitly reserved for this wizard.
2. **If we don't build it**: onboarding a new machine or switching providers stays a manual, documentation-driven multi-command exercise; the placeholder seat ships indefinitely; C1's structured `Report` (built precisely so "the future wizard consumes the same `Report` to filter its interview options without shelling out" — `docs/memory/distribution/setup.md`) has no consumer.
3. **Why an interview layer over C1's probe** (vs. alternatives): capability is *detected, never asked* — the wizard only asks preference questions (which provider, which mode) with options pre-filtered to what the probe found, so it cannot configure something the machine can't run. Writes go through the existing surgical `fab config set` semantics rather than any whole-file generation, so the wizard adds zero new config-write machinery and inherits presence=intent, fence handling, and tier targeting for free.

## What Changes

### 1. Bare `fab setup` becomes the wizard (`src/go/fab/cmd/fab/setup.go`)

Replace the placeholder `RunE` with the interactive wizard. Flow:

1. **Probe**: run `setupcheck.Run(setupCheckInput())` — the same call `fab setup check` makes — and consume the structured `Report` (provider roster with `Interactive`/`Headless`/`Native` capabilities and binary presence, tmux signal, config findings). No new probing, no shell-outs.
2. **Scope banner**: state the write target up front. Default target is the **system tier** (`~/.fab-kit/config.yaml`, i.e. `fab config set --system` semantics); a `--project` flag retargets to `fab/project/config.yaml`. Example banner: `Configuring the system tier (~/.fab-kit/config.yaml) — machine-wide preferences. Use --project to target this repo instead.`
3. **Default path — 4 questions** (see §2).
4. **Opt-in advanced section** (see §3).
5. **Diff-before-write summary** then writes (see §4).

`fab setup check` is untouched. The existing subcommand wiring (`cmd.AddCommand(setupCheckCmd())`) and the doctor's no-writes invariant stay as-is; the wizard is a sibling behavior on the bare command, not a change to `check`.

### 2. The 4-question default path

Every question shares one contract: it shows the **current effective value and its origin** as the default (Enter = keep, contributing to the all-Enter zero-write run), and its footer points at `fab config explain <key>` for the full prose (owner-or-pointer — the wizard never restates the reference documentation).

- **Q1 `agent.session`** — options are the provider names from the `Report`, **filtered to detected providers** (executables found on PATH per `ProviderProbe.Found()`), each **annotated** with its capabilities (e.g. `claude (interactive, headless, native)`).
- **Q2 `agent.workers`** — same option list and annotation as Q1.
- **Q3 `dispatch.mode`** — options filtered by viability: `pane` is offered only when the tmux signal says the pane rung is reachable (`$TMUX` present per the probe); `native`/`headless` viability follows the roster's capability columns. The question text **states the ladder semantics**: the value is a preference *ceiling* over `pane → native → headless` — resolution descends from it and never ascends, so a too-high setting degrades softly rather than erroring.
- **Q4 — advanced-section opt-in**: `Configure advanced options (agent.profiles.operator/review, dispatch.column_width, dispatch.reap_done)? [y/N]`.

Prompts are **bare stdin reads** (`bufio`-style line reads against the command's `InOrStdin()`), no TUI dependency, per Constitution I and the settled 2026-08-11 decision.

### 3. Opt-in advanced section

Keys: `agent.profiles.operator` (provider), `agent.profiles.review` (provider), `dispatch.column_width`, `dispatch.reap_done`.

**Skip rule** (per the backlog entry, read literally): skip any advanced question whose current value equals the built-in default *and* was never overridden at any tier — the section surfaces existing customizations for review rather than walking every knob. When every advanced question skips (the fresh-machine case), print a one-line note naming the skipped keys with the `fab config explain <key>` pointer instead of showing an empty section, so opting in never yields silence.

### 4. Writes — diff-before-write, surgical set, idempotent no-op

- After the interview, print a **diff summary** of every key whose answer differs from the current effective value (`<key>: <old> → <new> (target: system|project)`), then confirm before writing.
- **Zero changed answers = zero writes**: an all-Enter run prints "nothing to change" and exits 0 without touching any file (Constitution III — idempotent, byte-identical re-runs).
- Writes go **only through the existing `fab config set` code path** (the same surgical, scalar, leaf-level, fence-aware write `fab config set [--system] <key> <value>` performs — presence=intent, never whole-file regeneration). Reuse the internal set function directly rather than shelling out to a child `fab` process.

### 5. Non-interactive parity — `--defaults`

- `fab setup --defaults` runs the full flow non-interactively, accepting every question's default (the current effective value) — which by the zero-write invariant makes it a no-op write-wise; it still prints the banner, resolved answers, and the "nothing to change" summary. Composable with `--project`.
- When stdin is not a TTY and `--defaults` was not passed, fail with a usage hint (`stdin is not a TTY — use --defaults for non-interactive runs`) rather than hanging on a read or silently defaulting.
- Richer per-key answer flags and section entry points (`fab setup agents|dispatch`) are **deferred to C3** per the backlog entry — `--defaults` and `--project` are the only new flags in this change.

### 6. Skill and reference-doc updates

- **`src/kit/skills/fab-setup.md`**: the `/fab-setup` skill delegates its config-interview portion for the wizard-covered preference keys to `fab setup` (settled 2026-08-11: the name stays `fab setup`). The skill's existing identity-field create-mode (`fab config init --project` shell-out for name/description/source_paths/test_paths) is untouched — the wizard covers agent/dispatch preference keys, a disjoint set.
- **`src/kit/skills/_cli-fab.md`**: document the new bare-`fab setup` behavior and the `--defaults`/`--project` flags (constitution: CLI changes MUST update `_cli-fab.md`).
- **`docs/specs/skills.md` § fab-setup**: the backlog's settled "SPEC mirror update per constitution 1.5.0" is **OBE** — constitution 1.6.0 (260811-rehi) deleted the `docs/specs/skills/SPEC-*.md` mirror tree outright. The successor obligation is keeping the fab-setup section of `docs/specs/skills.md` consistent with the new delegation flow.

### 7. Tests

Extend `src/go/fab/cmd/fab/setup_test.go`: wizard flow driven through injected stdin (`cmd.SetIn`) — all-Enter zero-write run, answer-change produces the diff and the surgical write, `--defaults` non-interactive run, non-TTY-without-`--defaults` error, provider-option filtering from a fixture `Report`, dispatch.mode filtering without `$TMUX`, advanced skip rule (all-skipped note + overridden-key surfaced). Constitution: CLI change ships with test updates.

### Out of scope (deferred to C3 per the backlog entry)

- Pane warm-up / trust seeding
- Per-provider fill editing (`providers.<name>.profiles.*`)
- Section entry points (`fab setup agents|dispatch`)

## Affected Memory

- `distribution/setup`: (modify) — replace the "bare `fab setup` prints a placeholder" claims with the wizard's behavior (flow, scope targeting, invariants, `--defaults`, coexistence with `check`); update the /fab-setup delegation table for the config-interview handoff
- `distribution/kit-architecture`: (modify) — internal package map / setup-command coverage if the wizard lands as a new internal package or materially extends the setup surface description

## Impact

- **Go**: `src/go/fab/cmd/fab/setup.go` (wizard replaces placeholder; new `--defaults`/`--project` flags), possibly a new sibling internal package for the interview loop; `src/go/fab/cmd/fab/setup_test.go`. Read-only consumers: `internal/setupcheck` (Report), `internal/config`/`internal/configref` (effective values, origins, built-in defaults, set path).
- **Kit skills**: `src/kit/skills/fab-setup.md` (delegation), `src/kit/skills/_cli-fab.md` (command reference).
- **Specs/docs**: `docs/specs/skills.md` § fab-setup; hydrate updates the memory files above.
- **No changes** to `fab setup check`, `fab config` verbs, the registry, or any migration.

## Open Questions

*(none — all decision points graded below; the backlog entry plus C1's shipped code answered everything else)*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | All detection comes from `setupcheck.Run`'s `Report`; the wizard adds no probing and never asks capability questions | Backlog invariant ("capability is detected never asked"); C1's memory states the package was built for exactly this consumption | S:90 R:80 A:95 D:90 |
| 2 | Certain | Write target defaults to the system tier with `--project` to retarget; writes go only through the existing surgical `fab config set` semantics | Settled in the backlog entry verbatim; all four keys are `[both]`-scoped so system-tier default is valid | S:95 R:70 A:90 D:90 |
| 3 | Certain | Bare stdin line prompts, no TUI dependency; idempotent all-Enter zero-write run; diff-before-write summary; footers point at `fab config explain <key>` | All settled 2026-08-11 decisions recorded in the backlog entry; align with Constitutions I and III | S:95 R:75 A:90 D:90 |
| 4 | Certain | The "SPEC mirror update per constitution 1.5.0" settled decision is OBE — constitution 1.6.0 (260811-rehi) deleted the mirror tree; update `docs/specs/skills.md` § fab-setup instead, plus `_cli-fab.md` per the standing CLI constraint | Constitution history is explicit; the successor obligation is documented in the 1.6.0 amendment note | S:85 R:90 A:95 D:90 |
| 5 | Confident | The "4-question default path" = `agent.session`, `agent.workers`, `dispatch.mode`, plus the advanced-section opt-in prompt as the fourth question; the diff-before-write confirm follows regardless | The entry names only three keys for a "4-question" path; the opt-in prompt is the natural fourth and the flow's invariants are unaffected by the count | S:55 R:85 A:60 D:55 |
| 6 | Confident | Advanced skip rule read literally — skip keys at built-in default and never overridden (the section reviews existing customizations); when all skip, print a note naming the keys with the `explain` pointer | The entry's wording is explicit; the fresh-machine all-skip consequence is handled by the fallback note; first-time advanced setup remains available via `fab config set` (and C3's section entry points) | S:65 R:75 A:40 D:45 |
| 7 | Confident | Non-TTY stdin without `--defaults` errors with a usage hint rather than hanging or auto-defaulting | Not discussed in the entry, but easily reversible and the predictable-failure option; auto-defaulting would make CI runs silently interactive-shaped | S:30 R:85 A:60 D:50 |
| 8 | Confident | `--defaults` accepts every default non-interactively (zero-write no-op by the idempotence invariant), composable with `--project`; no per-key answer flags in this change | "Non-interactive parity via `--defaults`/flags" names the flag; richer flag surfaces sit with C3's deferred section entry points | S:60 R:80 A:65 D:60 |
| 9 | Confident | Q1/Q2 option lists drop undetected providers outright (filter, not annotate-as-missing); Q3 drops `pane` when tmux is absent, and states ladder semantics in the question text | The entry says "filtered to DETECTED providers" and "filtered by $TMUX viability" — filtering is removal; annotation is specified separately for capabilities | S:70 R:80 A:70 D:60 |
| 10 | Confident | The interview loop lands beside C1's code (extension of `setup.go` or a new sibling internal package consuming `Report`); exact placement is a plan-time decision | Follows C1's cmd-owns-wiring / internal-owns-logic split; either placement is cheap to revisit | S:55 R:90 A:75 D:70 |
| 11 | Confident | Writes reuse the internal config-set function in-process rather than exec'ing a child `fab config set` — identical semantics, no subprocess; the exact function seam is a plan-time detail | "Writes ONLY via existing `fab config set` semantics" pins the semantics, not the invocation mechanism; in-process is the codebase norm and trivially swappable | S:60 R:85 A:70 D:65 |

11 assumptions (4 certain, 7 confident, 0 tentative, 0 unresolved).
