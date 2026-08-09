# Intake: Give agy a Built-in `interactive_command`

**Change**: 260809-agik-agy-interactive-command
**Created**: 2026-08-09

## Origin

Backlog item `[agik]` via `/fab-new agik` (one-shot, one SRAD question asked). Raw backlog entry:

> [agik] 2026-08-09: Give agy/kimi an `interactive_command` (pure provider-data change, post-C4 names). Probe each CLI live in a real tmux pane BEFORE shipping (the rpsr rule): echo-on-typed-input + Enter-submit, first-run walls. Facts from 2026-08-09 marine-fox probes (agy 1.1.11): `agy -i '<prompt>'` (--prompt-interactive) VERIFIED delivering an initial prompt into an interactive session, so `interactive_command` ending in `-i` works under today's spawn-time pointer delivery; trust store = `~/.gemini/antigravity-cli/settings.json` trustedWorkspaces (EXACT-match paths, containment does NOT hold, no CLI verb to add) — trust dialog must be answered once per worktree until the pane readiness-gate change ships; agy Starter quota exhausted, resets ~2026-08-16. kimi: bare positional = subcommand error and NO interactive-initial-prompt flag exists, so kimi's interactive_command is only usable AFTER the readiness-gate/send-keys delivery change (intake on marine-fox) — probe kimi's echo behavior for that. Sweep roster-count phrasings ("dispatch-only") in _cli-agents.md/config docs when flipping capabilities.

**Scope decision (asked at intake)**: the user chose **agy only**. kimi stays dispatch-only — it has no interactive-initial-prompt flag, so a shipped `interactive_command` would make every pane-mode dispatch fail immediately (bare positional parses as a subcommand, non-zero exit); flipping kimi is a follow-up change gated on the readiness-gate/send-keys delivery redesign (change `3oz7`, intake on marine-fox).

**P1 resolution (asked mid-pipeline, 2026-08-09)**: the first apply run probed P1 and it FAILED — `-i`/`--prompt-interactive` is a **value-taking flag** on agy 1.1.11 (bare `-i` exits 2, `flag needs an argument: -i`), so a static trailing-`-i` grammar breaks session launches while omitting `-i` breaks pane delivery. The user chose the **`{prompt}` placeholder** resolution: extend spawn composition with a `{prompt}` placeholder in `interactive_command`, substituted with the shell-quoted pointer at pane dispatch and token-dropped (flag + placeholder pair) when empty at session launch — the same substitute-or-drop contract `{model}`/`{effort}` already have. This deliberately grows scope beyond pure provider data into `internal/spawn`/`internal/dispatch` composition code. Probe evidence: `plan.md` § Probe results.

## Why

1. **Pain point**: agy is pane-capable as of the 2026-08-09 marine-fox probes (`agy -i '<prompt>'` verified delivering an initial prompt into an interactive session, agy 1.1.11), but the built-in provider table still ships agy dispatch-only. Users who want a watchable agy pane worker (`dispatch.mode: pane`) or an interactive agy session (`fab agent --provider agy`, `fab operator` with an agy role) must hand-write `providers.agy.interactive_command` in their own config — and the docs actively steer them to do so with a now-partially-stale caveat.
2. **Consequence of not fixing**: the dispatch-only rationale recorded across the docs ("agy silently discards the positional, so no grammar can deliver a pane worker's prompt") is stale — the `-i` flag solves positional delivery under today's spawn-time pointer composition. Leaving it means fab under-advertises a working capability and the docs assert a falsehood about the current CLI.
3. **Why this approach**: the C4 rename (`interactive_command`/`headless_command`, PR #560, merged) was expected to make this a pure provider-data change; the P1 probe outcome (value-taking `-i`) grew it by one deliberate mechanism — the `{prompt}` composition placeholder (§ 1a, user-chosen) — plus the documentation/test sweep. The rpsr lesson (a pane-capable provider that cannot receive its prompt orphans every tmux auto-dispatched stage) is honored by probing live before ship and by keeping kimi out of scope.

## What Changes

### 1. Built-in provider table: agy gains `interactive_command`

`src/go/fab/internal/agent/defaults.yaml` (embedded via `//go:embed`, parsed once at package init) — add to the `agy` provider row:

```yaml
providers:
  agy:
    interactive_command: 'agy --dangerously-skip-permissions --model {model} -i {prompt}'
```

Grammar rationale (each element load-bearing):

- **`--dangerously-skip-permissions`** — the full-auto posture every built-in command carries (unattended stage workers cannot answer approval prompts); matches agy's existing `headless_command`.
- **`--model {model}`** — agy consumes fab's per-role model fills; **no `{effort}` placeholder** — agy's model IDs embed the reasoning level as a suffix (`gemini-3.1-pro-high`), same as its headless grammar.
- **`-i {prompt}`** (`--prompt-interactive`) — `-i` is a **value-taking flag** (P1 probe, agy 1.1.11: bare `-i` exits 2 with `flag needs an argument`). The `{prompt}` placeholder (new — § 1a below) receives the shell-quoted pointer at pane dispatch, and at session launch the empty value token-drops `-i {prompt}` as a pair, leaving a plain `agy --dangerously-skip-permissions --model <id>` session. `agy -i '<prompt>'` delivering the initial prompt was verified live 2026-08-09.
- Companion Go constant/comment updates in `src/go/fab/internal/agent/agent.go` (the comment block at ~lines 142–215 documents agy/kimi as deliberately shipping no interactive command — rewrite to kimi-only; add a `DefaultAgyInteractiveCommand` exported symbol if the existing no-duplicate-literal convention for codex requires one).

### 1a. Spawn composition: the `{prompt}` placeholder (user-chosen P1 resolution)

One `interactive_command` serves two seams — session launches (`fab agent --provider agy`, operator spawns; **no pointer**) and pane-mode dispatch (pointer delivered at spawn). P1 proved a static grammar cannot serve both for a value-taking flag like agy's `-i`. Resolution (asked; user chose over hold-for-3oz7 / ship-without-`-i` / ship-with-broken-sessions):

- **New placeholder `{prompt}`** recognized in `interactive_command`, with the same substitute-or-drop contract `{model}`/`{effort}` already have in `internal/spawn`'s `resolveTemplate`:
  - **Pane dispatch** (`internal/dispatch`): when the resolved command contains `{prompt}`, substitute the **shell-quoted** pointer (`dispatch.PointerPrompt` output) at the placeholder instead of appending it positionally via `WindowCommand`'s `resolvedCmd + " " + shellQuote(pointer)`. Commands **without** `{prompt}` keep today's positional append byte-for-byte (claude/codex unchanged — back-compat is a hard requirement).
  - **Promptless session launch** — bare `fab agent` / `fab agent --provider <name>` is the **only** seam that composes an `interactive_command` with no initial content: `{prompt}` substitutes empty ⇒ the existing empty-value token-drop rule removes the placeholder token and its preceding `-`-flag token (`-i {prompt}` → both dropped), exactly like an empty `{model}` drops `-m {model}`.
  - **Prompt-carrying launch seams** (rework cycle 1 correction — R6 originally misclassified these as promptless): the operator launcher (`cmd/fab/operator.go` appends `'/fab-operator'`), `fab batch new` and `fab batch switch` (append `/fab-new …` / `/fab-switch …` worker prompts) all deliver an **initial prompt positionally** today. Each MUST take the same two-shape branch as pane dispatch — substitute the shell-quoted prompt at `{prompt}` when present, else positional append — via **one shared helper** so the grammar has a single implementation. Without this, a `{prompt}`-carrying provider on `agent.session` silently loses those prompts (and it's a regression: pre-change agy fell through to the claude default and the prompt WAS delivered).
- **Placement contract**: the same grammar limits as `{model}`/`{effort}` token-drop apply (plain value-carrying flags; for nested-shell commands a droppable placeholder must not be the last token — agy's `interactive_command` has no nested shell, so `-i {prompt}` trailing is fine).
- **`{prompt}` must NOT flip `WithProfile`'s template-vs-append mode for model/effort on its own** — decide and record the exact mechanism at apply (e.g., a separate `WithPrompt` pass or extending `resolveTemplate`), but a command carrying only `{prompt}` and no `{model}`/`{effort}` should still get `--model`/`--effort` appended per append mode.
- **Docs**: `docs/specs/harness-adapters.md` (pointer-delivery contract gains the placeholder path), `_cli-agents.md` § Spawn Composition, `_cli-fab.md` (fab dispatch pane composition + `fab spawn-command` if it documents template mode), providers/config docs where `interactive_command` grammar is described.
- **Tests**: `spawn_test.go` (substitution + empty-drop + mode-interaction), `dispatch` composition tests (placeholder vs positional-append paths), plus the defaults_test.go grammar pin from § 4 asserting agy's command ends in `-i {prompt}`.

### 2. Trust wall: accepted and documented, NOT seeded

agy gates a fresh workspace behind an interactive trust dialog even under `--dangerously-skip-permissions`. Trust store: `~/.gemini/antigravity-cli/settings.json` `trustedWorkspaces` array — **EXACT-match absolute paths** (containment does not hold: a trusted parent does not cover a child dir), no CLI verb to add entries.

- **In scope**: docs caveat everywhere agy's pane capability is described — a pane-dispatched agy worker in a not-yet-trusted worktree parks at the trust dialog once per worktree until answered (a human answers it; the pipeline's wait-timeout peek classifies it as "waiting on genuine human input" and escalates without killing). Note that the readiness-gate change (`3oz7`) is the designed fix.
- **Out of scope**: `trustedWorkspaces` seeding (read-modify-write of agy's settings file at dispatch time). That is delivery choreography, not provider data — it belongs to the `3oz7` readiness-gate work.
- **Probe P2 must verify**: after answering the trust dialog, the `-i` initial prompt still lands as the first user message (i.e., the trust wall defers, not eats, the prompt).

### 3. Documentation sweep: dispatch-only phrasings and roster counts

agy's dispatch-only posture is asserted across skills, specs, memory, and Go comments. Every phrasing flips to **kimi as the sole dispatch-only built-in**; roster counts ("only claude and codex ship one", "the two dispatch-only built-ins") update; kimi's own dispatch-only prose stays (still true). Known sites (sweep, don't trust this list as exhaustive — grep `dispatch-only`, `interactive_command`, and roster-count phrasings like "two dispatch"/"only claude and codex"):

- `src/kit/skills/_cli-agents.md` — line ~61 (session-form note: "only claude and codex ship one … agy and kimi are dispatch-only"), ~134 (built-ins roster line), § agy (~166–194: Interactive row, § Dispatch-only section rewrites to describe the shipped `-i` grammar + trust-wall caveat; the "add one in your own config" block becomes redundant for agy), kimi's § Dispatch-only (~209) keeps its kimi-specific rationale but drops "Like agy".
- `src/kit/skills/_cli-fab.md` — ~412 (providers block paragraph: "agy and kimi are dispatch-only — no interactive_command, so no pane capability" and the dispatch-only-posture rationale sentence), ~1175 (error-reachability note: "the two dispatch-only built-ins").
- `docs/specs/stage-models.md` — ~242–259 ("Two of the four are dispatch-only"), ~433–437 (commented provider examples).
- `docs/specs/config.md` — ~325 ("they are dispatch-only built-ins (260808-rpsr)").
- `docs/specs/architecture.md` — ~279–283 (commented provider examples: "dispatch-only: no interactive_command").
- `docs/specs/harness-adapters.md` — check for agy/dispatch-only mentions.
- Constitution constraints apply: CLI-surface changes update `_cli-fab.md`; skill-file changes update the matching `docs/specs/skills/SPEC-*.md` mirrors (check whether `_cli-agents.md`/`_cli-fab.md` carry mirrors).
- `fab config explain` rendering (`internal/configref`) — agy's commented reference block gains the `interactive_command` line automatically via interpolation from the built-in table; byte-stability/coverage guard tests update. Check `configref.go` prose for dispatch-only phrasing.

### 4. Test updates

- `src/go/fab/internal/agent/defaults_test.go` — currently asserts agy and kimi carry **no** `interactive_command`; flip to kimi-only, and assert agy's `interactive_command` is non-empty, carries `{model}`, no `{effort}`, and ends in `-i` (the trailing-flag position is load-bearing for pointer delivery).
- `src/go/fab/internal/agent/defaultprofiles_mirrors_test.go`, `agent_test.go`, `spawn_test.go`, `cmd/fab/config_test.go`, `config_show_init_test.go` — update any assertions pinning agy's capability set or the rendered reference block.
- Ride-along (rpsr should-fix leftover): `config_test.go:755` "three names known today" → four.
- Ride-along (rpsr should-fix leftover): `_cli-agents.md:176` agy Approvals row says "both forms" — with agy now genuinely carrying two forms both bearing `--dangerously-skip-permissions`, verify the row reads true; correct whichever direction is wrong.

### 5. Pre-ship probes (the rpsr rule — blocking)

Run in a real tmux pane against the installed agy (1.1.11+) BEFORE ship:

- **P1 (session seam) — RESOLVED 2026-08-09 (first apply run)**: FAILED as originally posed — bare `-i` exits 2 (`flag needs an argument: -i`, usage dump); no session opens. Resolution: the `{prompt}` placeholder (§ 1a). **Residual P1'**: verify the token-dropped session form (`agy --dangerously-skip-permissions --model gemini-3.1-pro-high` — no `-i`) opens a usable plain session.
- **P2 (pane seam) — partially confirmed**: `-i '<prompt>'` launches the TUI and parks on the once-per-worktree trust dialog. **Remaining**: verify the `-i` prompt survives answering the trust dialog (lands as first user message after trust accept). No trust state was written by the first run.
- **P3 (quota) — resolved for P1**: both first-run probes resolved before any model call, so P1's verdict is quota-independent. Apply the same standard to P1'/P2: prompt landing is observable pre-response. **If any remaining probe genuinely needs quota (resets ~2026-08-16), HOLD ship until then — do not ship unprobed.**
- Echo-on-typed-input + Enter-submit behavior: record observations for the `3oz7` send-keys work while the pane is open (free rider, not a gate for this change).

### Non-goals

- kimi's `interactive_command` (follow-up after `3oz7` ships — at ship/archive time, annotate backlog `[agik]` or file a fresh backlog item for the kimi flip so it isn't lost).
- `trustedWorkspaces` seeding / any delivery-choreography code (belongs to `3oz7`).
- Changing `dispatch.mode` defaults or the descent ladder — under the default `native`, agy still resolves headless (no `native: true`); only `mode: pane` users see new behavior.
- kimi echo-behavior probing as a gate (opportunistic recording only).

## Affected Memory

- `runtime/providers-and-profiles.md`: (modify) agy's built-in-table row gains `interactive_command`; § Dispatch-only built-ins rewrites to kimi-only; the "agy and kimi Are Dispatch-Only" Design Decision is superseded for agy (record the `-i` discovery + trust-wall caveat); the "dispatch-only built-in inside tmux descends to headless" scenario re-anchors on kimi; defaults_test.go assertion description updates.
- `runtime/dispatch.md`: (modify) any pane-capability/descent examples that use agy as the no-`interactive_command` case re-anchor on kimi.
- `_shared/configuration.md`: (modify) four-provider presentation lines ("agy and kimi have a headless command only (dispatch-only …)", the codex/agy/kimi commented-block descriptions) update for agy's new capability line in the rendered reference.

## Impact

- **Go (data + comments + tests)**: `internal/agent/defaults.yaml`, `internal/agent/agent.go` (comments/exported symbols), `defaults_test.go`, `defaultprofiles_mirrors_test.go`, `agent_test.go`, `cmd/fab/config_test.go` (incl. the :755 ride-along), `config_show_init_test.go`, possibly `internal/configref/configref.go` prose + its byte-stability tests.
- **Go (composition code — § 1a)**: `internal/spawn/spawn.go` (`{prompt}` placeholder recognition + empty-drop; mode-interaction rule) + `spawn_test.go`; `internal/dispatch/dispatch.go` (`WindowCommand`/pane composition: substitute-at-placeholder vs positional-append back-compat) + dispatch tests.
- **Skills**: `src/kit/skills/_cli-agents.md`, `src/kit/skills/_cli-fab.md` (+ their `docs/specs/skills/` mirrors if present).
- **Specs**: `docs/specs/stage-models.md`, `docs/specs/config.md`, `docs/specs/architecture.md`, `docs/specs/harness-adapters.md` (check).
- **Behavioral surface**: `fab agent --provider agy` and operator agy spawns start working without user config; `dispatch.mode: pane` + agy selects the pane rung inside tmux (previously descended to headless). No behavior change under the default `dispatch.mode: native`. No migration needed (additive built-in data).
- **Release**: lands after 2.19.0 (C4 migration coupling already forces minor); this change itself is minor-compatible additive data.

## Open Questions

- ~~P1 contingency~~ — RESOLVED: P1 fired (bare `-i` hard-errors); the user chose the `{prompt}` placeholder resolution (§ 1a). No open questions remain.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is agy only; kimi stays dispatch-only, flipped in a follow-up gated on `3oz7` | Asked — user chose "agy only" over both-with-caveat and gated-on-3oz7; honors the rpsr never-ship-a-parking-pane-capability lesson | S:95 R:70 A:95 D:90 |
| 2 | Certain | Sweep every dispatch-only/roster-count phrasing (skills, specs, memory, Go comments, configref rendering) incl. SPEC mirrors | Backlog instructs the sweep explicitly; code-quality sibling-sweep discipline; rpsr showed counts escape keyword greps | S:85 R:80 A:90 D:85 |
| 3 | Confident | Trust wall is accepted + documented; `trustedWorkspaces` seeding is out of scope (belongs to `3oz7`) | Backlog frames agik as "pure provider-data change"; seeding is delivery choreography; trust dialog is once-per-worktree and human-answerable | S:60 R:75 A:70 D:65 |
| 4 | Certain | Grammar is `agy --dangerously-skip-permissions --model {model} -i {prompt}` (no `{effort}`) | P1 fired (bare `-i` hard-errors — value-taking flag); asked — user chose the `{prompt}` placeholder resolution; `-i '<prompt>'` delivery verified live | S:90 R:75 A:90 D:90 |
| 5 | Confident | Probe-before-ship is blocking; quota-blocked probes ⇒ HOLD until ~2026-08-16, never ship unprobed | The rpsr rule is recorded project discipline; delivery probes may not need quota (prompt landing visible pre-response) — P3 verifies | S:70 R:65 A:80 D:75 |
| 6 | Confident | Include the two rpsr should-fix ride-alongs (`_cli-agents.md` agy Approvals "both forms" row; `config_test.go:755` provider count) | Recorded in rpsr's review-result as candidate ride-alongs; this change touches exactly those areas | S:55 R:85 A:75 D:70 |
| 7 | Confident | Record the kimi follow-up at ship/archive (annotate backlog `[agik]` or file a fresh item) so the deferred half isn't lost | Backlog item covers both CLIs; archiving it after an agy-only ship would silently drop kimi | S:60 R:85 A:70 D:70 |
| 8 | Certain | `{prompt}` placeholder in spawn composition: substitute shell-quoted pointer at pane dispatch, token-drop flag+placeholder pair when empty; commands without `{prompt}` keep positional append byte-for-byte | Asked — user chose this over hold-for-3oz7 / ship-without-`-i` / broken-sessions; extends the existing `{model}`/`{effort}` substitute-or-drop contract | S:90 R:70 A:85 D:90 |
| 9 | Confident | `{prompt}` alone does not flip `WithProfile` into template mode for model/effort; exact mechanism (separate pass vs `resolveTemplate` extension) decided and recorded at apply | Mode semantics for `{model}`/`{effort}` are pre-existing contract; a `{prompt}`-only command should still receive appended `--model`/`--effort` | S:55 R:80 A:75 D:60 |

9 assumptions (4 certain, 5 confident, 0 tentative, 0 unresolved).
