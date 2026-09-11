# Intake: Operator loading and dependency — rk-mandatory skill, `_cli-fab.md` split by command family, live clock cadence

**Change**: 260910-si4k-operator-rk-mandatory-helper-split
**Created**: 2026-09-11

## Origin

Created by `/fab-proceed`'s `_intake` prefix step (promptless-defer dispatch — no questions asked; would-be questions are recorded as deferred Unresolved rows in `## Assumptions`). The source of truth is the live `/fab-discuss` conversation that produced the plan doc `fab/plans/sahil/26-09-11-operator-generic-tracking.md` (uncommitted in the worktree) and its study `docs/wiki/operator-tick-anatomy.html`. This change implements **that plan's "Change A" only**. "Change B" (the generic `tracked` item model) is explicitly out of scope and is named here only as a Non-Goal.

> Operator loading and dependency: rk-mandatory skill, `_cli-fab.md` split by command family, live clock cadence in the ready line and compact frame. Change A only — no behaviour change, docs/skill only, no Go. (1) run-kit becomes a hard dependency of the operator *skill*: one startup gate in §2 replaces every scattered `command -v rk` check; delete the raw `tmux send-keys` fallbacks, the `/loop` degraded clock and the `Clock: none` ready-line form, the rk-absent notify ladder and the `Notify channel` setting, the version-skew fallbacks, and the rk-absent pane-map/delivery-probe arms; keep `rk notify`'s fail-silent semantics and the Go binary's own fallbacks. (2) Sweep `_cli-agents.md`'s 12 rk-absent arms; delete `_cli-external.md` § /loop if the operator was its only consumer. (3) Split `_cli-fab.md` by command family so the operator loads its slice — owner-or-pointer, each section exactly one home; re-declare every skill's `helpers:`; update `_preamble.md` § Skill Helper Declaration and the Constitution's "MUST update `src/kit/skills/_cli-fab.md`" wording (no version bump; dated HTML comment). (4) Live cadence: §2 Init step 5 renders ONE template from `rk cron list --json` fields; the §4 compact frame line gains the schedule after the tracked count. (5) Docs sweep of the sibling class.

**Decisions the user took in the conversation** (recorded as Certain, not reopened): probes for external items will live in fab, never as an rk wake source (Change B territory — no rk-side alternatives are to be proposed); the operator's git choreography (§6 dependency resolution, merge modes, ordered merge, auto-merge) stays inline in `fab-operator.md` and Change A must not move or trim it; rk is mandatory for the operator skill.

**Alternatives the user rejected**: making `helpers:` section-addressable inside one big `_cli-fab.md` (the helper model is file-based; a split keeps owner-or-pointer clean); extracting the choreography to an on-demand helper (main job of the operator — it must survive a compaction reload with a merge sequence armed).

## Why

**The operator re-pays 3,425 lines on every reload.** `/fab-operator` (`src/kit/skills/fab-operator.md`, 903 lines) declares `helpers: [_cli-agents, _cli-fab, _cli-external]`, so one load is 903 + `_cli-fab.md` 1,475 + `_preamble.md` 545 + `_cli-agents.md` 255 + `_cli-external.md` 247. The operator uses roughly 400 of `_cli-fab.md`'s 1,475 lines — the `## fab operator`, `## fab pane`, `## fab agent` and (via `_cli-agents`/`_preamble` pointers) `## fab dispatch` sections; the other ≈900 lines (change/status/score/config/docs-index/pr-meta/…) serve planning and pipeline skills the operator never runs. Unlike a one-shot skill, the operator is long-lived and reloads on every compaction, `/clear`, and restart (§4 Post-Compaction Reload — the tick that finds no procedure in context runs `/fab-operator` once), so the cost recurs for the life of the session.

**The skill documents a mode nobody runs.** 27 passages in `fab-operator.md` (and 12 in `_cli-agents.md`) describe what to do when rk is absent, when an installed rk predates a verb, or when a raw `tmux send-keys` must stand in for `rk mux send`. But the operator's clock (the `operator tick` rk cron entry), role mark (`rk role operator`), send gate (`rk mux send` plain/`--answer`/`--force`/`--key`), spawn readiness (`rk mux await --ready`), session candidacy (`rk mux sessions --json`), and escalation (`rk notify`) are all run-kit already. Without rk there is no clock (the `/loop` fallback is Claude-only and was retired as the primary cadence in 260909-6t88), so "operator without rk" is not a degraded mode — it is a non-functional one that the prose keeps alive at reload cost and at drift risk (every fallback arm is a second copy of a rule that changes when the rk-side contract changes).

**Cadence is rendered from memory.** §2 Init step 5 offers four literal ready-line variants with a hard-coded `(backoff 60s–30m, wakes on agent-state-change)` — already stale against the live entry (`rk cron list --json` reports `backoff 1m→30m`, `deliver: immediate`), and the compact frame shows no cadence at all. The plan's Change B will let the binary tune the entry's schedule per tracked set, so the frame needs to read cadence from the entry, not from a literal; Change A lands that read now because it depends on nothing unreleased.

**If we do nothing**: every operator reload keeps paying ≈2,300 lines of reference the operator does not use, the 39 fallback passages keep drifting against the rk contract they mirror, and the ready line keeps stating a cadence that is not the entry's. Change B would then land on top of a 903-line file still carrying all of it.

**Why this approach**: the helper model is already file-based and per-skill (`helpers:` frontmatter, `_preamble.md` § Skill Helper Declaration), so splitting `_cli-fab.md` by command family is the mechanism the kit already has — no section-addressing, no loader change, no Go. Making rk mandatory for the operator *skill* (not for the binary) matches how `wt` is already treated in §2 wt Gate: one STOP with an install hint, no per-call-site gating.

## What Changes

Everything below is markdown under `src/kit/skills/`, `fab/project/`, and `docs/`. **No `.go` file changes** (so the constitution's CLI⇒docs+tests rule is not triggered). `.agents/skills/` and `.claude/skills/` are deployed copies — never edited; `fab sync` regenerates them and enumerates every `*.md` in the skills dir, so the new helper files deploy with no binary change (verified in `src/go/fab-kit/internal/skills.go` `listSkills`; `fab help` excludes `_*` partials; `lifecycle_collision_test.go` parses only `_cli-fab.md`'s router line in § Calling Convention, which stays in the core file).

### 1. run-kit is a hard dependency of the operator skill (`fab-operator.md`)

**One startup gate** in §2 Startup, placed as its own subsection immediately after `### Tmux Gate` and before `### Role Mark` (the role mark and everything after it assume rk):

```markdown
### rk Gate

The operator's clock, role mark, send gate, spawn readiness, and notifications are run-kit. Probe once here — no later call site is individually gated:

```bash
command -v rk >/dev/null 2>&1 && rk cron list --json >/dev/null 2>&1
```

If either half fails (rk absent, or an installed rk predating `rk cron` — the capability probe), STOP:

```
Error: the operator requires run-kit — brew install sahil87/tap/run-kit
```

This is the operator's deliberate exception to `_preamble.md` § Run-Kit (rk) Reference's fail-silent rule: that rule protects skills for which rk is an optional enhancement; for the operator rk is the substrate, so absence is a startup error, not a degradation.
```

The gate is idempotent (a re-run after a `/fab-operator` reload re-probes harmlessly). `_preamble.md`'s rk section itself is **not** edited (plan Non-Goal: `_preamble.md` changes are limited to the helper-declaration wording).

**Delete from `fab-operator.md`** (today's line numbers, for the apply agent's orientation — verify by grep, not by number):

| Site | Today | Change |
|------|-------|--------|
| §0 intro (l.24) | "routes commands and answers via `rk mux send` when rk is installed (`command -v rk`-gated — …), degrading to raw `tmux send-keys` behind its own §3 state gate when rk is absent — never an error" | Drop the gate qualifier and the whole "degrading to raw …" clause; the sentence names `rk mux send` (plain / `--answer` / `--key`) as *the* send path |
| §0 intro (l.26) | "the built-in launcher below is the rk-absent fallback … Its exact degraded behavior … is documented in `_cli-fab.md` § fab operator and is the canonical §9 Key Properties rows below" | Keep the fact that bare `fab operator` delegates to `rk operator` and that the binary has a built-in launcher (that is a **binary** fallback, out of scope — see Non-Goals); shorten to one sentence pointing at the CLI reference; drop "degraded behavior" prose from the skill body |
| §2 Role Mark (l.77–80) | `command -v rk >/dev/null 2>&1 && rk role operator >/dev/null 2>&1 \|\| true` + "extended to version skew: an absent rk, or an installed rk predating the `role` subcommand, degrades to a silent no-op" | `rk role operator >/dev/null 2>&1 \|\| true` — keep the `\|\| true` (a role-mark failure is never startup-blocking; staleness and radio conflicts are rk's to resolve — that sentence stays); delete the `command -v` prefix and the version-skew sentence |
| §2 Init step 4 (l.97) | "Verify the clock — fail-silent, gated on `command -v rk`: … Absent rk, a failing `rk cron list --json` (a pre-cron rk), or a missing entry is a degraded state, never an error. No loop is started — the entry (or the §4 Degraded Fallback) is the whole clock story" | The gate already proved `rk cron list --json` works; step 4 reads it for the entry and its `muted`/`muted_until`/`schedule_summary`/`deliver` fields (§ 4 below). A **missing** `operator tick` entry after the gate passed STOPs: `Error: no operator-tick cron entry on this server — run rk operator to seed it` (see Assumptions — the `Clock: none` form is deleted per the user; the operator cannot tick without the entry, and `rk operator` seeds it idempotently). Delete the fallback clauses |
| §2 Init step 5 (l.99–107) | Four literal ready-line variants incl. `Clock: none — … /loop 3m "operator tick"` | Replaced by the single template in § 4 below |
| §3 Pre-Send Validation item 2 (l.128) | "with rk installed (`command -v rk`), a `waiting` target rides `rk mux send --answer`, an `active` target requires `rk mux send --force` …; with rk absent, the confirmed send is raw `tmux send-keys` behind the operator's own state gate just performed" | Keep the `--answer`/`--force` mapping; delete "with rk installed (`command -v rk`)" and the "with rk absent …" clause |
| §4 Degraded Fallback (l.195–197) | Whole subsection (`/loop 3m "operator tick"`) | Delete; remove its Contents entry and the two cross-references in §4 Tick Payload ("and the fallback loop's prompt (§ Degraded Fallback below)") and §4 Post-Compaction Reload step 3 ("(or the fallback loop prompt)") |
| §4 Tick Behavior step 1 (l.325, tail) | "**Version-skew fallback** (two rungs, softest first): if `--quiet` errors … drop `--quiet` …; if `tick-start --diff` or `fab pane questions` errors as an unknown flag/command … fall back to the flagless tick …" | Delete the whole paragraph (the skill and the binary ship together; a skew is a `fab upgrade-repo` problem, not a per-tick branch) |
| §5 Notification Send (l.476–483) | "**When `rk` is absent** … configurable via the §8 `Notify channel` setting:" + the ntfy.sh / Discord webhook / `PushNotification` / Slack MCP bullets + "**All notify sends fail silently** (the fallback path matches `rk notify`'s contract …)" | Delete the rk-absent paragraph and the four bullets. **Keep** one sentence: "`rk notify` fails silently by contract — a notification that cannot be delivered MUST NOT crash or stall the operator; it logs one line and keeps ticking" (that is rk's contract, not a fallback) |
| §5 Sending Auto-Answers (l.487–489) | "when rk is installed (`command -v rk`-gated) …. When rk is absent, the answer is raw `tmux send-keys` (keys and literal text alike) behind the same gate — never an error." / "(`rk`-gated with raw-tmux fallback)" / "On the rk-absent raw path, apply the delivery probe … instead of re-sending blind." | Delete the gate qualifiers and the three rk-absent clauses; the delivery-verification sentence for the rk path stays |
| §6 Spawning step 2 (l.533) | "**Candidate source, rk-first**: behind the standard fail-silent gate plus a capability probe (`command -v rk >/dev/null 2>&1 && rk mux sessions --json` — …) … **Fallback (rk absent or verb-less)**: exclude `_rk-*`-prefixed sessions by name …" | Candidate source is `rk mux sessions --json` `role: "user"` rows, unconditionally; delete the gate/probe framing and the name-prefix fallback sentence (the exclusion *rule* — never the operator's own session, never infrastructure sessions — stays; only the rk-absent mechanism goes) |
| §8 One Operator Per Server (l.869) | "(or `tmux -L <label> send-keys …` on the rk-absent raw path)" | Delete the parenthetical |
| §8 Settings (l.877) | `Notify channel` row | Delete the row; the table keeps Stuck threshold and Spawn target session |
| §9 Key Properties (l.897) | "the rows below describe the rk-absent fallback launcher" | Reword: the launcher rows describe the binary's built-in launcher (owned by the CLI reference); the skill body no longer describes a degraded mode |
| §9 Key Properties Cadence row (l.901) | "Claude-only `/loop` fallback when the entry cannot exist (§4 Degraded Fallback);" | Delete the clause; add "live cadence rendered from `rk cron list --json` (§2 Init step 5, §4 Status Frame Format)" |
| §9 Key Properties | — | Add a row `Requires run-kit? \| Yes — hard stop without it (§2 rk Gate: `command -v rk` + `rk cron list --json`)` beside the existing `Requires tmux?` row |
| §2 Context Loading (l.52) | "`_cli-fab` (fab command reference), and `_cli-external` (wt, idea, tmux, /loop reference)" | Re-list the declared helpers per § 3 below; drop "/loop" |

**Keep untouched**: §6 Dependency Resolution, Dependency Declaration, Working a Change, Autopilot (the whole choreography, l.573–807 today) — user-decided; `rk notify` fail-silent semantics; every reference to the Go binary's own fallbacks (built-in launcher in `fab operator`, `fab pane map`'s tmux enumeration path, dispatch-internal `fab pane capture/kill/process`) — those are binary behaviour other consumers rely on and are described in the CLI reference, not deleted.

**Rule for the sweep**: delete a passage when it exists only to describe rk absence or an rk older than the gate's floor (`rk cron`), or a raw-tmux substitute for an rk verb the operator uses. Keep raw-tmux mechanics that serve a non-absence purpose (the pre-delivery judgment rounds of `_preamble.md` § The pane readiness gate type into a not-yet-delivered pane with raw `tmux send-keys` by design — that is not an rk-absent path).

### 2. Sweep `_cli-agents.md` (12 arms) and `_cli-external.md`

`_cli-agents.md` is declared only by `fab-operator` (verified: the only `helpers:` list naming it), so with rk mandatory for its sole consumer its rk-absent arms go — under the same rule as § 1:

| Section | Today | Change |
|---------|-------|--------|
| § Scope Boundary (l.39) | "— each with a raw-`tmux` fallback when rk is absent. fab's own `fab pane capture`/`kill`/`process` are dispatch-internal — kept for the rk-less pane arm …" | Delete the "each with a raw-tmux fallback" clause. **Keep** the dispatch-internal sentence (binary-side; other consumers) |
| § Pre-Send Validation step 2 (l.115) | "When rk is installed (`command -v rk`), `rk mux send` enforces this gate …" and "**When rk is absent (never an error):** hold the gate yourself — … deliver with raw `tmux send-keys` … followed by the § Delivery Probe manual recipe." | Delete the gate qualifier and the whole rk-absent sentence; `rk mux send` *is* the gate |
| § Pre-Send Validation step 3 (l.117) | "Before a raw `tmux send-keys` (the rk-absent fallback here, the pre-delivery judgment rounds, any manual driving) …" | Keep the step (the judgment rounds and manual driving are live consumers); delete "the rk-absent fallback here," |
| § Delivery Probe (l.125) | "`rk mux send` (`command -v rk`-gated) mechanizes the same probe …; the manual recipe below is the fallback for non-fab/raw-tmux driving — every send when rk is absent included — …" | Delete the gate qualifier and "— every send when rk is absent included —"; the manual recipe stays as the explanation and the judgment-round procedure |
| § Peek Output (l.140) | "when rk is present (`command -v rk`-gated) … When rk is absent, fail open to raw `tmux capture-pane -p -t <pane>` (pipe through `tail -N` …) — never an error." | Delete both clauses |
| § Peek Process tree (l.142) | "(rk-gated) … failing open to raw `tmux kill-pane` when rk is absent" | Delete both clauses |
| § Await (l.152–154) | "**`rk mux await` (when rk is present)** — `command -v rk`-gated" / "**Poll (the rk-less fallback)**" / "`rk notify` — `command -v rk`-gated and fail-silent" | Drop the gate qualifiers; rename item 2 to "**Poll** (when no `await` signal fits — e.g. a completion signal that is a screen pattern)"; keep `rk notify`'s fail-silent note without the gate |
| Frontmatter `description` | mentions raw-tmux fallbacks | Re-word to match |

Also re-point its 11 `_cli-fab.md § fab …` pointers per § 3.

`_cli-external.md`: **delete `## /loop`** (l.229–247) and its `- /loop (fallback reference)` Contents entry and the "and /loop (fallback-scoped)" / "/loop-fallback notes" phrases in the frontmatter description. **Consumer check done at intake** (repo-wide grep for `/loop` over `src/kit`, `docs/memory`, `docs/specs`, `fab/project`): the only consumers of the section were `fab-operator.md` §4 Degraded Fallback, §2 Init step 5 and §9 Cadence — all deleted in § 1. `_cli-agents.md` § Skill Prompts item 3 (l.108) names `/loop` as an example of a native TUI control that is *not* prefix-transformed — that is a mention, not a consumer; leave it. Also in `_cli-external.md`: drop `command -v rk`-gate qualifiers in § rk (run-kit) (Operator escalation send and the pointer subsections) that exist only to describe absence — the file is operator-only — but keep § Reference Model's `<tool> skill` capability-probe/shll.ai bundle-page delegation (that is generic tool-version handling for four binaries, not an rk-absent arm). Re-point its 4 `_cli-fab.md § fab …` pointers per § 3.

### 3. Split `_cli-fab.md` by command family

**Cut** (today's `## ` boundaries in `src/kit/skills/_cli-fab.md`; counts are lines between headings):

| Section | Today | Lines | New home |
|---------|-------|------:|----------|
| `## fab pane` | l.526–634 | 109 | **`_cli-fab-pane.md`** (new) |
| `## fab dispatch` | l.635–858 | 224 | **`_cli-fab-pane.md`** (new) |
| `## fab operator` | l.1187–1374 | 188 | **`_cli-fab-operator.md`** (new) |
| `## fab agent` | l.1375–1432 | 58 | **`_cli-fab-operator.md`** (new) — *placement deferred; see Assumptions #14 and Open Questions* |
| everything else — Calling Convention, change, status, score, preflight, log, resolve, resolve-agent, config, doctor, migrations-status, kit-path, setup, shell-init, skill, impact, pr-meta, docs-index, fab-help, help-dump, batch, Common Error Messages | — | ≈896 | **`_cli-fab.md`** (stays) |

Moved text is moved **verbatim** (owner-or-pointer: each section has exactly one home; nothing is duplicated, nothing is paraphrased in transit). Each new file gets the partial frontmatter shape used by `_cli-fab.md` today:

```yaml
---
name: _cli-fab-pane
description: "Fab CLI reference — the `fab pane` and `fab dispatch` command families (pane primitives: map/capture/process/window-name/open/ready/deliver/kill/questions; the two-mode stage-dispatch manager). Split out of _cli-fab so operator and dispatch consumers load only this slice."
user-invocable: false
disable-model-invocation: true
metadata:
  internal: true
---
# Fab CLI Reference — pane & dispatch

> Loaded via a skill's `helpers: [_cli-fab-pane]` frontmatter. Core commands (change/status/score/preflight/log/resolve/config/…) are in `_cli-fab`; `fab operator`/`fab agent` in `_cli-fab-operator`.

## Contents
- fab pane
- fab dispatch
```

(`_cli-fab-operator.md` mirrors it with `fab operator` / `fab agent`.) `_cli-fab.md`'s header blurb and `## Contents` shrink to the sections it keeps and gain one pointer line naming the two family files. Every file keeps a TOC (`internal-skill-optimize`'s structural rule for partials over 100 lines).

**Re-point every pointer** into a moved section. Intake-time grep (`_cli-fab(\.md)?` followed by `§ fab (dispatch|pane|operator|agent)`) found these sites — the apply agent re-runs the grep and re-targets each to the section's new file:

| File | Sites | Target family |
|------|------:|---------------|
| `src/kit/skills/fab-operator.md` | 15 | operator (state/tick-start/note/autopilot/watch/launcher), pane (`§ fab pane · questions`, identity-key contract) |
| `src/kit/skills/_cli-agents.md` | 11 | agent (×5), pane (×4 incl. `§ fab pane → § agent state`, `§ fab pane ready`), operator (`§ fab operator tick-start`), dispatch (×1) |
| `src/kit/skills/_preamble.md` | 6 (+ `§ reap`, `§ fab dispatch ready`) | dispatch (×4), agent (×2) — `_preamble.md § Per-Stage Model Resolution` and `§ CLI-Adapter Dispatch` |
| `src/kit/skills/_cli-external.md` | 4 | operator (×2), pane (×2) |
| `src/kit/skills/fab-archive.md` | 1 | dispatch |
| `fab/project/constitution.md` (Principle V example "`_cli-fab.md` § fab dispatch") | 1 | dispatch — becomes "`_cli-fab-pane.md` § fab dispatch" |
| `docs/specs/hooks.md` ×3, `docs/specs/stage-models.md` ×2, `docs/memory/runtime/pane-commands.md` ×2, `docs/memory/distribution/kit-architecture.md` ×1 | 8 | pane / agent — docs sweep |

**Intra-file references that become cross-file** after the cut (re-point in the moved/kept text): `## fab resolve-agent` (stays core) → `§ fab agent` ×4 (l.271, 283, 319, 350) and `§ fab dispatch` ×1 (l.349); `## fab pane` (moves) → `§ fab operator tick-start` (l.542); `## fab dispatch` (moves) → `§ fab agent` (l.665). References between `## fab pane` and `## fab dispatch` (l.596, 600, 639) stay intra-file because both land in `_cli-fab-pane.md`. Pointers to sections that stay in core (`§ fab score`, `§ fab status`, `§ fab change`, …) are untouched.

**Re-declare `helpers:`** for what each skill actually uses (today's lists in parentheses):

| Skill | New `helpers:` | Why |
|-------|----------------|-----|
| `fab-operator` | `[_cli-fab-operator, _cli-fab-pane, _cli-agents, _cli-external]` (was `[_cli-agents, _cli-fab, _cli-external]`) | Uses `fab operator`, `fab pane`, `fab agent --print --repo` (spawn), and `fab dispatch` by `_cli-agents`/`_preamble` pointer; needs none of the core family — its only core commands (`fab log command`, `fab resolve` in §3 Branch Fallback) are covered by `_preamble.md` § Common fab Commands. CLI-reference load 1,475 → ≈580 lines |
| `fab-fff`, `fab-ff`, `fab-continue`, `fab-adopt` | *deferred — Assumptions #16*: user's stated intent is to add `_cli-fab-pane` (they dispatch); today they declare no `_cli-fab` at all and reach `§ fab dispatch` through `_preamble.md` § CLI-Adapter Dispatch pointers | Default if unresolved at apply: add `_cli-fab-pane` as the user wrote |
| Planning skills (`fab-new`, `fab-draft`, `fab-clarify`, `code-reorg`, `code-dedupe`) and every other skill | unchanged | None declares `_cli-fab` today; `_preamble.md` § Common fab Commands covers their surface |
| `_preamble.md` example block (`helpers: [_generation, _review, _srad, _pipeline]`) | unchanged | Still a valid example |

**`_preamble.md` § Skill Helper Declaration**: "**Allowed values**: `_generation`, `_review`, `_cli-fab`, `_cli-fab-operator`, `_cli-fab-pane`, `_cli-external`, `_cli-agents`, `_srad`, `_pipeline`, `_intake`." (8 → 10). Its `_cli-fab` mentions elsewhere (§ Common fab Commands "See `_cli-fab` for the full reference", § Confidence Scoring "`_cli-fab.md` § fab score") stay — those sections remain in core.

**Constitution** (`fab/project/constitution.md`, Additional Constraints): "Changes to the `fab` CLI (Go binary) MUST include corresponding test updates and MUST update the CLI reference partial that owns the command's family — `src/kit/skills/_cli-fab.md` (core), `_cli-fab-pane.md` (`fab pane`, `fab dispatch`) or `_cli-fab-operator.md` (`fab operator`, `fab agent`) — with any new or changed command signatures". Principle V's example "(e.g. `_cli-fab.md` § fab dispatch)" → "(e.g. `_cli-fab-pane.md` § fab dispatch)". **Wording amendment only — no new or changed MUST rule**, so **no version bump** (stays 1.8.0; the udwv / jjg0 / t513 / yd9s precedent); `Last Amended` → this change's date; append a dated HTML comment `<!-- 2026-09-11 (260910-si4k): … -->` explaining the family-file rename. **Sibling restatements** of the same rule get the same wording: `fab/project/code-quality.md` § Anti-Patterns (project-specific) "Changing a CLI command without updating `_cli-fab.md` + tests" and `fab/project/code-review.md` § Project-Specific Review Rules "CLI ⇒ docs + tests".

### 4. Live cadence in the ready line and the compact frame

**Verified facts (run-kit v3.19.46, `rk cron list --json`, 2026-09-11)** — the skill copies fields, never composes them:

```json
{
  "id": "hk6c",
  "name": "operator tick",
  "schedule": { "kind": "backoff", "min": "1m0s", "max": "30m0s" },
  "schedule_summary": "backoff 1m→30m",
  "wake_on": { "event": "agent-state-change", "scope": "server", "debounce": "1m0s" },
  "target": "role:operator",
  "deliver": "immediate",
  "muted": false
}
```

`schedule` is an **object**; the display string is **`schedule_summary`** (the conversation's description said "`schedule` (string)" — corrected here from the live output). `muted` is the effective state (indefinite mute or live lease); `muted_until` (unix seconds) is present only while a lease is live. The operator-tick entry is the row whose `target` is `role:operator` (the same resolution `fab operator`'s clock verbs use).

**§2 Init step 5 — one template**, replacing the four literals:

```
Operator ready. Clock: rk cron "operator tick" · {schedule_summary} · {deliver}[ · muted[ until HH:MM]]
```

Rendering rules: `{schedule_summary}` and `{deliver}` are the JSON values verbatim; append ` · muted` when `muted` is true; append ` until HH:MM` (local time) when `muted_until` is present. Renders today as `Operator ready. Clock: rk cron "operator tick" · backoff 1m→30m · immediate`, or `… · immediate · muted until 14:30`. (Whether to spell the field name — `· deliver immediate` — is deferred: Assumptions #15.) A missing entry STOPs per § 1 (no `Clock: none` form).

**§4 Status Frame Format — compact frame** gains the schedule after the tracked count:

```
🛰️ **Operator** · {HH:MM} · tick #{N} · **{tracked} tracked** · {schedule_summary} · no change[ · {W} waiting]
```

e.g. `🛰️ **Operator** · 22:06 · tick #5122 · **6 tracked** · backoff 1m→30m · no change`. Update the "Compact frame" row of the Element/Format table and the fenced documentation example accordingly. The **full-frame header line is unchanged** (the user asked for the compact line only). **Source per tick**: `rk cron list --json` is re-read once per tick — §4 Tick Behavior step 7 ("Clock lifecycle — none to manage") becomes "Clock — read `rk cron list --json` for the operator-tick entry; render its `schedule_summary` on the frame (cadence is never carried from a previous tick or composed from memory); no lifecycle to manage — the tracked-set verbs mute/unmute, rk evaluates the schedule". This honours §1 "Re-derive state" and keeps the frame live after a user `rk cron edit`.

### 5. Docs sweep (the sibling class)

| File | What |
|------|------|
| `docs/memory/runtime/operator.md` | Remove the rk-absent / raw-tmux / `/loop` / `Notify channel` / version-skew prose in the present-truth sections (Context Loading helper list l.31, pre-send l.51, spawn candidacy l.94, clock l.130, Notification Send l.197, re-capture l.208, Settings l.260, launcher l.275–279); record the new §2 rk Gate, the single ready-line template and the compact-frame cadence; add a Design Decision "rk is a hard dependency of the operator skill" that **supersedes** "Claude-Only /loop Fallback Instead of Hard-Requiring rk" (l.587) and the role-mark version-skew decision (l.512) — superseded, not silently deleted; update the frontmatter `description` ("and the Claude-only /loop fallback") |
| `docs/memory/runtime/agent-primitives.md` | Requirement sections restating `_cli-agents.md`'s rk-absent arms (Peek "capture is the universal fallback" l.79–83 — keep the *uninstrumented-pane* caveat, drop the rk-absent clause; Await l.95; pre-send l.34) |
| `docs/memory/_shared/context-loading.md` | "Allowed values (eight)" → ten; the `helpers:` mapping table row for `fab-operator`; any restatement of the `_cli-fab` shape |
| `docs/memory/distribution/kit-architecture.md` | Skills tree (l.25–27), the helper-loading table (l.344–351), the mapping list (l.360), "enumerates the eight allowed values" (l.363), and the `_cli-fab.md § fab agent` pointer (l.317) |
| `docs/memory/runtime/pane-commands.md` | Two `_cli-fab.md § fab pane …` pointers → `_cli-fab-pane.md` |
| `docs/specs/operator.md` | Version History: add `v10 — rk-mandatory skill (one startup gate, 27 fallback passages deleted), CLI reference split by command family (`_cli-fab-operator`/`_cli-fab-pane`), live cadence from `rk cron list --json` in the ready line and compact frame`; update "The current operator (v9) evolved through nine iterations" |
| `docs/specs/skills.md` | § Skill Helpers "Allowed values (8)" → 10 and the mapping table; the `/fab-operator` section's Purpose/Context/Flow/Tools lines that restate `command -v rk`-gating, raw `tmux send-keys`, and the Claude-only `/loop` fallback (l.1099, 1101, 1115, 1117, 1121); § Adding a skill checklist item 3's helper list |
| `docs/specs/hooks.md`, `docs/specs/stage-models.md` | Pointer re-targets only |
| `docs/memory/runtime/index.md`, `docs/memory/index.md` | Regenerated by `fab docs-index` (descriptions carry "/loop fallback" today) |

Log/seed files (`log.md`, `log.seed.md`) are generated — not hand-edited.

### 6. Verification before apply finishes

1. Repo-wide grep (excluding `fab/changes/`, `docs/wiki/`, `fab/plans/`, `*/log.md`, `*/log.seed.md`, `docs/specs/findings/`) for `rk is absent|rk-absent|when rk is absent|raw tmux|send-keys|/loop|Notify channel|version-skew|ntfy|Discord webhook|PushNotification` — every remaining hit must be either a binary-side fallback description (pane map enumeration, built-in launcher, dispatch-internal verbs), the judgment-round raw-send carve-out, or a historical Design Decision.
2. Repo-wide grep for `_cli-fab` — every hit either names the core file for a section that stayed, or names the family file for a moved section.
3. `wc -l src/kit/skills/_cli-fab*.md` — the three files sum to today's 1,475 (± the added frontmatter/Contents lines); no section appears in two files (`grep -c '^## fab pane' …` = 1 across the set).
4. `go test ./src/go/fab-kit/cmd/fab/ -run 'TestKitContentCitesNoRepoLocalPaths|TestKitPortabilityMatcher'` and `go test ./src/go/fab/cmd/fab/ -run TestLifecycleCollision` — the new deployed files cite no fab-kit-only path and the router-line anchor still parses. (Running existing tests only; no test is added or changed — no Go changed.)
5. `fab sync` deploys `_cli-fab-operator` and `_cli-fab-pane` to `.agents/skills/` (and `.claude/skills/`) with no warnings.

### Non-Goals (Change B — later, `fab/plans/sahil/26-09-11-operator-generic-tracking.md` § Change B)

Any state-file or tick-document change; the generic `tracked` item model and `fab operator track` verbs; the one-table frame with Kind/Checked/Next columns; `rk cron edit`-driven derived schedules; `rk tab new`/`rk tab mark` adoption; `has_agent`; `rk mux capture --classify`; any Go change; deleting the binary's built-in launcher or `fab pane map`'s tmux fallback; any rk-side probe/wake mechanism (decided against); moving or trimming the §6 choreography; editing `_preamble.md` beyond § Skill Helper Declaration and the pointer re-targets.

## Affected Memory

- `runtime/operator`: (modify) rk Gate, single ready-line template, compact-frame cadence, deleted fallback prose, superseding Design Decision for the `/loop`-fallback and role-mark version-skew decisions
- `runtime/agent-primitives`: (modify) rk-absent arms removed from the pre-send / peek / await requirement sections
- `runtime/pane-commands`: (modify) pointer re-targets to `_cli-fab-pane`
- `_shared/context-loading`: (modify) helper allowed-values list (10), `fab-operator` mapping row, the family-split shape of the CLI reference
- `distribution/kit-architecture`: (modify) skills tree, helper-loading table, mapping list, allowed-values count, `§ fab agent` pointer

## Impact

- **Files**: `src/kit/skills/fab-operator.md` (≈900 → ≈780 lines), `_cli-agents.md` (255 → ≈180), `_cli-external.md` (247 → ≈200), `_cli-fab.md` (1,475 → ≈896), new `_cli-fab-operator.md` (≈250) and `_cli-fab-pane.md` (≈335), `_preamble.md` (pointer re-targets + allowed values), `fab-archive.md` (1 pointer), skills whose `helpers:` change; `fab/project/constitution.md`, `code-quality.md`, `code-review.md`; `docs/memory/runtime/{operator,agent-primitives,pane-commands}.md`, `docs/memory/_shared/context-loading.md`, `docs/memory/distribution/kit-architecture.md`, `docs/specs/{operator,skills,hooks,stage-models}.md`, regenerated indexes.
- **Operator reload cost**: 3,425 → ≈2,290 lines (skill ≈780 + CLI reference ≈585 + `_preamble` 545 + `_cli-agents` ≈180 + `_cli-external` ≈200).
- **Behaviour**: none for a user with run-kit installed (every deleted arm was the not-taken branch). An operator started without run-kit, or with an rk predating `rk cron`, now STOPs at startup with an install hint instead of running clock-less.
- **No Go, no tests changed**; the existing portability guard and router-line test are run as verification.
- **Deploy**: `fab sync` picks up the two new partials automatically; `fab help` ignores `_*` files. Validation of `helpers:` values is convention-only (`fab sync` does not reject unknown names), so the allowed-values lists in `_preamble.md` / `skills.md` / `context-loading.md` / `kit-architecture.md` are the only guard — hence the sweep.
- **Risk**: the sibling sweep (this project's most common rework cause) — the intake's grep list in § 6 is the acceptance check. Constitution V: the new deployed files must not cite `docs/specs/*`, `docs/memory/*`, `docs/site/*`, `src/go/*` — moved text is verbatim from an already-guarded file, so this holds unless new prose is added.

## Open Questions

- **`fab agent` placement** (deferred — Assumptions #14). The proposal bundles `## fab agent` (58 lines) with `## fab operator`. Evidence at intake: its consumers are the pipeline dispatch seam (`_preamble.md` § Per-Stage Model Resolution / § CLI-Adapter Dispatch, `_pipeline.md`, `fab-continue`, `fab-fff`, `fab-proceed`), `_cli-agents.md` § Spawn Composition (×5), and `_cli-fab.md` § resolve-agent (×4, stays core); the operator body calls `fab agent` twice (`--print --repo` for spawns). Options: (a) with operator as proposed — two files; (b) with pane/dispatch — the `dispatch:` seam consumer; (c) its own `_cli-fab-agent.md` — three files, loadable by both sides for 58 lines. Recommendation: (c) — it is the one section with consumers on both sides of the operator/pipeline divide. Default if unresolved at apply: (a), as the user wrote.
- **Pipeline skills' `helpers:`** (deferred — Assumptions #16). Add `_cli-fab-pane` unconditionally to `fab-fff`/`fab-ff`/`fab-continue`/`fab-adopt` (+≈335 lines per run; the shipped `dispatch.mode: native` default never enters the CLI-adapter branch), load it stage-conditionally at the `dispatch:`-present branch (the `/fab-continue` `_generation`/`_review` precedent), or keep today's pointer-only access? Recommendation: stage-conditional in-body read. Default if unresolved: unconditional, as the user wrote.
- **Ready-line `deliver` spelling** (deferred — Assumptions #15): `· backoff 1m→30m · immediate` (bare value, as the template) vs `· deliver immediate`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | rk is a hard dependency of the operator skill: one §2 gate — `command -v rk` AND `rk cron list --json` succeed — else STOP with `Error: the operator requires run-kit — brew install sahil87/tap/run-kit` | User-decided in the discussion; the exact probe and error text are the user's | S:95 R:70 A:90 D:95 |
| 2 | Certain | Probes for external items live in fab; no rk-side wake/probe alternative is proposed anywhere in this change | User-decided (Change B territory, decided against rk-side wake) | S:95 R:80 A:85 D:95 |
| 3 | Certain | §6 git choreography (dependency resolution, merge modes, ordered merge, auto-merge) stays inline in `fab-operator.md`, untouched by A | User-decided; extraction rejected — must survive a compaction reload with a merge armed | S:95 R:85 A:90 D:95 |
| 4 | Certain | The Go binary's own fallbacks stay (built-in launcher, `fab pane map` tmux enumeration, dispatch-internal `fab pane capture/kill/process`); no `.go` edits | User-stated non-goal; other consumers rely on them; keeps the CLI⇒tests rule untriggered | S:90 R:85 A:90 D:90 |
| 5 | Certain | Delete `_cli-external.md` § /loop (and its Contents/description mentions) | Intake grep: the only consumers were `fab-operator.md`'s degraded clock, ready line and Cadence row — all deleted here; `_cli-agents.md:108` is an example mention, not a consumer | S:85 R:95 A:95 D:90 |
| 6 | Certain | Ready line and compact frame render `schedule_summary` (string), `deliver`, `muted`, `muted_until` from `rk cron list --json`; the entry is the row with `target == "role:operator"` | Verified on run-kit v3.19.46: `schedule` is an object, `schedule_summary` is the display string — corrects the description's "`schedule` (string)" | S:80 R:90 A:90 D:85 |
| 7 | Certain | Change type `refactor`; constitution wording amendment only — no version bump (stays 1.8.0), `Last Amended` updated, dated HTML comment appended | udwv / jjg0 / t513 / yd9s precedent: no MUST rule added or changed | S:90 R:90 A:95 D:90 |
| 8 | Certain | New helper files need no Go change to deploy | `fab sync` enumerates every `*.md` in the skills dir (`listSkills`); `fab help` skips `_*`; `lifecycle_collision_test` parses only the router line, which stays in core `_cli-fab.md` | S:85 R:90 A:95 D:95 |
| 9 | Certain | `fab/project/code-quality.md` and `code-review.md` restatements of the CLI⇒`_cli-fab.md` rule are swept with the Constitution wording | Sibling-sweep rule; both files restate the constraint verbatim today | S:70 R:95 A:85 D:80 |
| 10 | Certain | `docs/memory/runtime/operator.md` DD "Claude-Only /loop Fallback Instead of Hard-Requiring rk" (and the role-mark version-skew DD) are superseded by a new DD, not deleted | FKF present-truth style keeps rationale history in Design Decisions | S:70 R:90 A:80 D:80 |
| 11 | Confident | A missing operator-tick entry after the gate passes STOPs with `Error: no operator-tick cron entry on this server — run rk operator to seed it`; the ready line has exactly one template (no `Clock: none` variant) | User asked to delete the `Clock: none` form; the operator cannot tick without the entry and `rk operator` seeds it idempotently — an actionable stop beats a clock-less session | S:60 R:90 A:65 D:55 |
| 12 | Confident | Cadence is re-read each tick (`rk cron list --json` once per tick, Tick step 7) and rendered on the compact line; the full-frame header is unchanged | User specified the compact line only; §1 "Re-derive state" + "never composed from memory" favour a per-tick read over carrying the Init value; one cheap local command | S:55 R:90 A:60 D:50 |
| 13 | Confident | Version-skew arms below the gate floor are deleted with the rk-absent arms (Role Mark "predating `role`", §6 step 2 "verb-less" name-prefix fallback, tick step 1 two-rung fallback); the generic `<tool> skill` shll.ai bundle-page fallback in `_preamble.md`/`_cli-external.md` § Reference Model stays | The gate's `rk cron list --json` probe defines the supported floor; the bundle-page pointer is tool-delegation for four binaries, not an rk-absent arm | S:70 R:80 A:70 D:65 |
| 14 | Unresolved | `## fab agent` (58 lines) goes with `## fab operator` in `_cli-fab-operator.md` (two new files) vs its own `_cli-fab-agent.md` (three) vs with pane/dispatch. Default if unresolved at apply: two files as proposed; recommendation: three (`fab agent` has consumers on both the operator and pipeline sides) | Deferred — promptless dispatch | S:45 R:65 A:35 D:30 |
| 15 | Unresolved | Ready-line spelling: `· backoff 1m→30m · immediate` (bare `{deliver}` value, the template as written) vs `· deliver immediate`. Default: bare value | Deferred — promptless dispatch | S:55 R:95 A:60 D:50 |
| 16 | Unresolved | `fab-fff`/`fab-ff`/`fab-continue`/`fab-adopt` `helpers:`: add `_cli-fab-pane` unconditionally (user's stated intent; +≈335 lines per run) vs stage-conditional in-body read at the `dispatch:`-present branch vs pointer-only (today). Default if unresolved: unconditional as the user wrote; recommendation: stage-conditional | Deferred — promptless dispatch | S:60 R:85 A:55 D:40 |
| 17 | Confident | `_cli-agents.md` raw-tmux mechanics that serve the pre-delivery judgment rounds (pane-mode clear guard, manual probe-and-retype recipe) stay; only their rk-absent framing is deleted | `_preamble.md` § The pane readiness gate types into a not-yet-delivered pane by design — a live consumer, not an absence path | S:70 R:85 A:80 D:70 |
| 18 | Confident | The operator's exception to `_preamble.md` § Run-Kit fail-silent rule is stated in `fab-operator.md` §2 rk Gate (pointing at the rule); `_preamble.md`'s rk section is not edited | Plan non-goal limits `_preamble.md` edits to the helper-declaration wording; owner-or-pointer: the preamble owns the universal rule, the skill declares its exception | S:65 R:85 A:75 D:65 |

18 assumptions (10 certain, 5 confident, 0 tentative, 3 unresolved).
