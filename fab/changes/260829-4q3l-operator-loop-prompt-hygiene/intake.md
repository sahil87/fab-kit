# Intake: Operator Loop-Prompt Hygiene

**Change**: 260829-4q3l-operator-loop-prompt-hygiene
**Created**: 2026-08-29

## Origin

> Operator loop-prompt hygiene — lean tick prompt as a hard rule, post-compaction reload procedure, /loop twin fix.
>
> A live operator session was observed re-firing the literal `/fab-operator` slash command on every heartbeat tick (via the /loop skill's dynamic ScheduleWakeup mode). Slash commands macro-expand their full source text into the turn every firing; `src/kit/skills/fab-operator.md` is ~86KB (~21k tokens), so at the 3m cadence this burned ~400k tokens/hour of context on skill text alone and would exhaust the context window in roughly 10 ticks. The skill canon already says the loop prompt is `/loop "operator tick"` (§4, §9), but it is stated once, in prose, buried in a large file — the session drifted from it and then "rediscovered" it. The fix is prose-only, in kit skills.

**Interaction mode**: promptless dispatch (`/fab-draft`-style create-intake via the shared `_intake` procedure with `{questioning-mode} = promptless-defer`). No questions were asked; every decision the intake author would have asked is recorded as a deferred Unresolved row in `## Assumptions`. The description was synthesized from a prior conversation in which the user made these decisions:

- Fix is **skill prose only**, in `src/kit/skills/` canonical sources — never `.claude/skills/`.
- **Out of scope** (user decisions): (a) slimming the `_cli-fab.md` load; (b) shrinking the per-tick status frame / a `--quiet` tick output — a separate follow-up FULL change to run after this one; (c) a Go-side operator daemon heartbeat — logged as backlog idea `[2ne8]`.
- Light lane expected (few tasks). Follow the owner-or-pointer rule (`fab/project/code-quality.md`): state each rule once in its owner and point elsewhere.

## Why

1. **The pain point.** The operator is a long-lived session whose heartbeat is a `/loop`. The loop prompt is the single highest-leverage string in the whole skill: whatever it says is re-executed every 3m (or 90s) for hours. Today `fab-operator.md` states the correct prompt — `/loop "operator tick"` — exactly once in running prose at §4 line ~145 and again as a fragment in the §4 "One-loop invariant" bullet and the §9 `Uses /loop?` row. Nothing says what the prompt MUST NOT be. A session that composes its own loop line (rather than copying one) can plausibly reach for `/fab-operator` — "the skill that knows what a tick is" — and the harness then macro-expands ~21k tokens of skill source into every tick. Observed cost: ~400k tokens/hour of pure skill text; context exhausted in ~10 ticks.

2. **The consequence of not fixing it.** Every operator run is one composed loop line away from silently burning its entire context on re-reading its own instructions. The failure is quiet (the ticks still "work") until the window fills, at which point the operator loses conversational continuity mid-coordination. The durable state file (§4) protects monitored/autopilot/branch_map/notes, but everything the user said in-session is lost, and the loop keeps firing against a compaction summary with no tick procedure in context.

3. **The adjacent defect the same session exposed.** The §1 "Self-manage context" principle tells the agent to `/clear` "near capacity". `/clear` is a **user-only** harness command — the agent cannot invoke it. What actually happens is harness **auto-compaction**: the skill body drops out of context, the loop keeps firing `operator tick`, and the agent has no procedure for "I was told to tick but I no longer hold the tick procedure". The principle therefore describes a mechanism that cannot run and omits the one that does.

4. **The stale twin.** `_cli-external.md` § /loop Constraints restates operator cadence facts that contradict `fab-operator.md`: "Stop: when the monitored set becomes empty" (owner says: while monitored set OR autopilot queue OR watches OR an in-progress merge sequence remain) and "Autopilot override: autopilot uses its own cadence (default 2m); replaces any existing monitoring loop" (owner says: one loop at a time, 3m/90s adaptive). `fab-operator.md` §4 in turn *points back* at `_cli-external.md` for the "default `2m`" autopilot cadence — a circular ownership with no real owner. This exact contradiction was already flagged in `docs/specs/findings/skills-review-2026-06-11.md` (line ~1278) and never fixed. The owner-or-pointer anti-pattern in `code-quality.md` names this drift mechanism precisely.

5. **Why this approach.** Prose-only, in the canonical skill sources, because the defect *is* prose: the rule exists but is neither prominent nor negatively bounded (no MUST NOT), and the context-recovery principle names the wrong mechanism. A literal copyable block + an explicit MUST NOT + a real reload procedure is the smallest change that removes the failure mode. Alternatives considered and rejected: shrinking the tick's status frame (real, but orthogonal — deferred to a follow-up FULL change by user decision); a Go-side heartbeat daemon (removes the LLM from the tick entirely — much larger, backlog `[2ne8]`); slimming `_cli-fab.md` (user said no).

## What Changes

All edits land in `src/kit/skills/` canonical sources (Constitution V; `code-quality.md` anti-pattern "Editing `.claude/skills/` directly"). No Go changes are expected (see Assumption 1). Run `fab sync` after editing so the deployed copies match, but never edit them.

### 1. Lean tick prompt as a hard rule — `fab-operator.md` §4 "The Loop" / "Adaptive cadence", mirrored as a pointer in §9

**§4 opening paragraph (line ~145)** currently reads: *"The loop is the operator's heartbeat — a `/loop "operator tick"` that runs as long as …"*. Immediately after the "Adaptive cadence" bullets, add a new subsection **`### Loop Prompt`** (the **owner** of this rule) containing:

1. A literal block of the exact invocations the agent copies — never composes:

   ```
   /loop 3m "operator tick"      # normal cadence
   /loop 90s "operator tick"     # tightened cadence (§4 Adaptive cadence, §8)
   ```

2. An explicit prohibition, in RFC-2119 form, with its one-line reason:

   > The loop prompt **MUST be the bare text `operator tick`**. It **MUST NOT** be `/fab-operator` or any other slash command. Reason: a slash command macro-expands its full source into the turn on **every** firing — `fab-operator.md` alone is ~21k tokens, so a `/fab-operator` loop prompt re-pays the whole skill each tick (~400k tokens/hour at `3m`) and exhausts the context window in roughly ten ticks. The tick procedure (§4 Tick Behavior) is already in context; the prompt only needs to *name* it.

3. The **dynamic (self-paced) mode** clause: `/loop` also has a mode with no fixed interval where the model self-paces by passing a wakeup prompt back each tick (the `ScheduleWakeup` mechanism). In that mode the wakeup prompt passed back **MUST likewise be the bare `operator tick` text**, never a slash command — the same rule, applied to the string the agent hands back rather than the string it typed. (Whether the operator is permitted to use dynamic mode at all is Assumption 9 — deferred; the intake only requires that *if* used, the prompt rule holds.)

4. A one-line pointer forward: "Recovery when this procedure is no longer in context: § Post-Compaction Reload."

**Existing sentences to reconcile** (state-once): the §4 opening sentence keeps `/loop "operator tick"` as the *name* of the heartbeat but should read *"a `/loop` whose prompt is the bare `operator tick` (§ Loop Prompt)"*; the "One-loop invariant" bullet's `e.g. restart /loop 90s "operator tick"` stays (it is the same literal). The **"Autopilot composition" bullet** (line ~152, *"autopilot's own cadence (default `2m`, `_cli-external.md`) governs the loop"*) and **Tick Behavior step 7's** parenthetical *"(autopilot uses §6's cadence)"* (line ~278) both reference a cadence that no owner defines once `_cli-external.md` stops stating it — see Assumption 8 (deferred) for the two resolutions; whichever is chosen, both sentences are rewritten so the loop has exactly one cadence owner (§4 Adaptive cadence).

**§9 Key Properties `Uses /loop?` row** (line ~817): append a pointer, not a restatement — *"… loop prompt is the bare `operator tick` — never a slash command (§4 Loop Prompt)"*.

**§2 Init step 5** (line ~97) currently: *"Output: `Operator ready.` (+ `Loop active ({interval}).` if loop started)"*. Change so the ready line carries the exact loop line the agent just used, so the agent copies rather than composes:

```
Operator ready.
Operator ready. Loop active (3m) — /loop 3m "operator tick"
```

(second form when the loop started; `{interval}` is the active cadence, `3m` or `90s`). This is a **skill-prose** output line: the Go launcher (`src/go/fab/cmd/fab/operator.go`) prints no `Operator ready` / `Loop active` text — verified by grep — so no Go change is involved (Assumption 1). Step 4 (*"start the single loop per §4 Adaptive cadence"*) gains *"using the literal from § Loop Prompt"*.

### 2. Replace the "Self-manage context" principle with a real procedure — `fab-operator.md` §1 row + new §4 subsection

**§1 Principles row** (line 40) currently:

> | Self-manage context | Near capacity, `/clear`, reload context and the state file, then resume; monitored/autopilot state survives (§4). |

Replace with:

> | Survive compaction | The agent cannot `/clear` itself. When a tick fires and §4 Tick Behavior is no longer in context (harness auto-compaction, or a session resumed from a summary), run `/fab-operator` **once** to reload, re-run §2 Init, then resume lean `operator tick` firings; monitored/autopilot/branch_map/notes survive in the server-keyed state file (§4 Post-Compaction Reload). |

**New §4 subsection `### Post-Compaction Reload`** (the **owner** of the procedure), placed after `### Loop Prompt`:

- **Trigger** — a tick (`operator tick`) arrives and the §4 Tick Behavior procedure is not in context. Concretely: the agent cannot see the numbered Snapshot → Auto-nudge → Watches → Autopilot → Removals → Observed-field updates → Loop lifecycle list. Typical causes: harness auto-compaction of a long session; a fresh session resumed from a conversation summary; a user `/clear`.
- **Procedure** — (1) run `/fab-operator` exactly **once** — this reloads the skill body and its helpers (`_cli-agents`, `_cli-fab`, `_cli-external`) and re-runs §2 Startup including Init (state file re-read via `fab operator state`, `fab pane map --all-sessions`, loop re-establishment per § Loop Prompt); (2) treat the tick that triggered the reload as consumed — Init's fresh `fab operator tick-start --diff` on the next tick will re-emit any level-triggered deltas, so nothing is lost; (3) continue with bare `operator tick` firings. **Never** put `/fab-operator` into the loop prompt as the way to "stay reloaded" — that is the §1 failure mode this procedure exists to replace.
- **Durable-state fact** (kept from the old row) — monitored set, autopilot queue, `branch_map`, watches, and notes all live in the server-keyed operator state file and survive compaction, `/clear`, crash, and restart; only session-scoped settings (§8) and in-conversation context are lost.
- **`/clear` is a user action.** It remains valid — a user may `/clear` a bloated operator — and it lands on this same procedure (the next tick, or the user's next message, finds no procedure in context). The agent-side mechanism is *compaction → one-shot `/fab-operator` reload*; the skill never instructs the agent to `/clear`.

**Sweep of every other `/clear` mention in `fab-operator.md`** so wording is consistent (`/clear` may stay where it describes a user action or a durability property; it must not appear as something the agent does):

| Line ~ | Current | Rewrite direction |
|--------|---------|-------------------|
| 49 (§2 Context Loading) | "a long-lived session re-pays any loaded file after every `/clear`" | "… after every reload (compaction, `/clear`, or restart — § Post-Compaction Reload)" |
| 94 (§2 Init step 2) | "(supports `/clear` recovery)" | "(this is what makes § Post-Compaction Reload lossless)" |
| 221 (§4 Monitored Set) | "`/clear`-restored entries" | "reload-restored entries" |
| 718 (§6 Auto-Merge rule 4) | "survives `/clear`, crash, and abandonment" | "survives compaction, `/clear`, crash, and abandonment" |
| 798 (§8 Settings) | "reset on `/clear` or session restart" | "reset on compaction, `/clear`, or session restart" |

### 3. Fix the stale twin — `_cli-external.md` § /loop Constraints

Current block (lines ~238–243):

```
### Constraints

- **One loop at a time** — there SHALL be at most one active `/loop` in a session
- **Start**: when the first change is enrolled in monitoring and no loop is running
- **Stop**: when the monitored set becomes empty, or on explicit user command
- **Autopilot override**: autopilot uses its own cadence (default 2m); replaces any existing monitoring loop
```

Rewrite so `_cli-external.md` owns only the tool-level fact and points at the operator for everything policy-shaped (owner-or-pointer rule; `_cli-external.md` § Reference Model already says it carries only fab-owned content and delegates the rest):

```
### Constraints

- **One loop at a time** — there SHALL be at most one active `/loop` in a session; changing the interval means re-establishing *the* loop, never adding a second.
- **Operator policy lives in `fab-operator.md` §4** — start/stop conditions, the `3m`/`90s` adaptive cadence, the autopilot composition, and the mandatory bare-text loop prompt (`operator tick`, never a slash command — § Loop Prompt). This file does not restate them.
```

Also add a **Usage** note that `/loop` has a self-paced (no-interval) mode and that the prompt rule in `fab-operator.md` § Loop Prompt applies to the wakeup prompt in that mode too — one sentence, pointer only. Remove "Start", "Stop", and "Autopilot override" bullets entirely (they are the contradicting restatements). The `docs/specs/findings/skills-review-2026-06-11.md` finding at line ~1278 is a historical report and is **not** edited (Assumption 6).

### 4. Sibling sweep (per `code-quality.md` § Sibling Sweeps)

Grep repo-wide (excluding `.claude/`, `fab/changes/archive/`, historical `docs/specs/findings/`, and `log*.md`) for `operator tick`, `/loop`, `Self-manage context`, `default 2m`, and `/clear` in operator context, and update every occurrence in the class:

- `docs/memory/runtime/operator.md` — line ~129 "`/loop` lifecycle" paragraph (add the loop-prompt rule + the compaction-reload procedure as present truth; drop any "default 2m" autopilot cadence claim per Assumption 8); line ~31 context-loading paragraph if it says "re-pays after `/clear`"; § Design Decisions gets an entry in the four-field shape for the loop-prompt rule + reload procedure (hydrate decides final placement).
- `docs/memory/_shared/context-loading.md` — line ~200 operator partial-exception paragraph (mentions `/clear`).
- `docs/specs/skills.md` § `/fab-operator` — line ~1102 "continuity across `/clear` comes from the server-keyed operator state file" → "continuity across compaction or `/clear` …"; the Flow block line ~1111 "runs a continuous /loop cycle" may add "(prompt: bare `operator tick`)".
- `docs/specs/glossary.md` — currently defines no operator loop/tick term; **no entry is added** (Assumption 7).
- `docs/specs/operator.md` — the v4 history-table row "`/loop`-driven monitoring" is a version-history fact and is not edited.

## Affected Memory

- `runtime/operator`: (modify) `/loop` lifecycle paragraph gains the mandatory bare-text loop prompt (`operator tick`, never a slash command; applies to the dynamic-mode wakeup prompt too) and the Init ready-line literal; the "Self-manage context" principle is replaced by the compaction → one-shot `/fab-operator` reload procedure; the autopilot "default 2m" cadence claim is resolved per Assumption 8; a Design Decisions entry records why (slash-command macro-expansion cost; agent cannot `/clear`).
- `_shared/context-loading`: (modify) the operator partial-exception paragraph's `/clear` wording aligned to "reload (compaction / `/clear` / restart)".
- `distribution/kit-architecture`: (modify — only if needed) the `_cli-external.md` one-line description mentions "/loop"; update only if the Constraints rewrite changes what the file is said to carry (it still carries a /loop note, so likely no edit).

## Impact

- **Files**: `src/kit/skills/fab-operator.md` (§1 row, §2 Context Loading + Init steps 4–5, §4 new `### Loop Prompt` + `### Post-Compaction Reload`, Adaptive-cadence bullets, Tick Behavior step 7, §4 Monitored Set, §6 rule 4, §8 Settings, §9 row), `src/kit/skills/_cli-external.md` (§ /loop Usage + Constraints), `docs/memory/runtime/operator.md`, `docs/memory/_shared/context-loading.md`, `docs/specs/skills.md`.
- **Code**: none. No `src/go/` change; no tests. `fab sync` regenerates deployed copies.
- **Behavioral contract**: the operator's loop prompt becomes a MUST/MUST NOT rule; the context-recovery mechanism changes from an impossible agent-side `/clear` to compaction → one-shot `/fab-operator` reload; `_cli-external.md` stops owning any operator cadence/lifecycle facts.
- **Lane**: light (prose edits in ~5 files, one sweep).
- **Explicitly out of scope**: `_cli-fab.md` load slimming; status-frame shrink / `--quiet` tick (follow-up FULL change); Go operator heartbeat daemon (backlog `[2ne8]`); any change to `/loop` itself (harness-owned).

## Open Questions

None — the three deferred decisions (Assumptions 8–10) were resolved before apply: autopilot rides the single `3m`/`90s` loop; either `/loop` mode is permitted with the bare-text prompt rule binding both; the Init ready line always echoes the loop literal.
