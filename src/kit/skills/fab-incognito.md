---
name: fab-incognito
description: "Prime the agent with fab process knowledge for a redesign discussion — loads the kit's process helper partials and a skill catalog, deliberately skips docs/memory and docs/specs, and holds a standing rule not to open them for the rest of the session. Use when rethinking how the fab process should work rather than how the current system does work. Read-blind, not trace-free: the invocation is still logged to the active change's history when one exists."
helpers: [_pipeline, _intake, _srad, _generation, _review]
---

# /fab-incognito

> Read the `_preamble` skill first (deployed to `.agents/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Arguments
- Context Loading
- Standing Session Rule
- Command Logging
- Behavior
- Orientation Summary
- Error Handling
- Key Properties

---

## Purpose

Prime the agent for a **redesign** discussion. Where `/fab-discuss` loads the project's documented landscape (the memory and specs indexes), this skill loads the kit's **operative behavior** — the process helper partials that define how a fab task is performed, plus a catalog of every deployed skill — and deliberately leaves the project's `docs/memory/` and `docs/specs/` closed. Present-truth memory anchors an agent to defend what exists; incognito anchors the conversation on how the process works so proposals can question it.

The name is the browser metaphor with one inversion: browser incognito means "don't *record* history"; this skill means "don't *read* history". It is **read-blind, not trace-free** — the invocation is logged to the active change's history when one exists (`fab log command` writes nothing when no change is active), and any skill invoked later in the session may still write artifacts. No artifact generation, no stage advancement — purely read-only.

---

## Arguments

None.

---

## Context Loading

This section **overrides** the always-load layer, per `_preamble.md` §1 ("the skill file wins"). The skill loads a reduced 5-file `fab/project/` set and skips both doc indexes.

1. **Project layer** — read `fab/project/config.yaml` and `fab/project/constitution.md` (required — error if missing) and `fab/project/context.md`, `fab/project/code-quality.md`, `fab/project/code-review.md` (optional — skip gracefully). Do **NOT** read `docs/memory/index.md` or `docs/specs/index.md`. The constitution's MUST rules are the guardrails a redesign still has to respect; the indexes are the narrative it should not be anchored on.
2. **Process helpers** — the frontmatter `helpers:` list loads `_pipeline`, `_intake`, `_srad`, `_generation`, `_review` in full (plus `_preamble`, loaded universally): six partials, roughly 120 KB together. These are the fab process knowledge and the primary payload of this skill.
3. **Skill catalog** — for every deployed `.agents/skills/*/SKILL.md`, read only the frontmatter `name` and `description` (never the body). Include the helper partials (`_*`) and the `internal-*` skills; mark entries with `user-invocable: false` as helpers. Any mechanism that reads frontmatter only is fine, e.g.:

   ```bash
   for f in .agents/skills/*/SKILL.md; do
     awk '/^---$/{c++; next} c==1 && /^(name|description|user-invocable):/' "$f"
   done
   ```

   If `.agents/skills/` is missing or empty, STOP with: `Deployed skills not found — run fab sync.`
4. **Kit version** — `cat "$(fab kit-path)/VERSION"`; fall back to `unknown` when the file is missing (the same source and fallback `fab fab-help` uses). Deployed skills are the *released* kit, which can lag the binary and the repository; the version line is how the user notices when that matters.
5. **Lazy loads — on demand, never up front** — full skill bodies (`.agents/skills/<name>/SKILL.md`) and the five CLI reference partials `_cli-fab`, `_cli-fab-pane`, `_cli-fab-operator`, `_cli-agents`, `_cli-external` (~274 KB together). Open one only when the discussion turns to that skill or command family. Size rationale: every deployed skill together is ~834 KB against the ~36 KB always-load layer; incognito is a *different anchor*, not less context, so it must not consume the window at start.
6. **Active change** — run `fab resolve --folder --or-none`. `(none)` ⇒ "No active change" (an expected state, not an error; a non-zero exit is a real error — surface it per `_preamble.md`'s failure rule). If it prints a folder name, read `fab/changes/{name}/.status.yaml` and derive the current stage from its `progress` map — the stage holding `active` or `ready` (or `failed` for a parked review/review-pr); all stages `done`/`skipped` means the change is complete. Do **not** load change artifacts (intake, plan). Do **not** run preflight.

---

## Standing Session Rule

This is a behavior obligation for the rest of the conversation, not a loading note:

> For the remainder of this discussion, do **NOT** open any file under `docs/memory/` or `docs/specs/` — including `docs/memory/index.md` and `docs/specs/index.md` — and do **NOT** follow `_preamble.md` § Memory File Lookup. The single exception: the user names a specific file; open exactly that file, without walking its domain index. Reason about the process from the loaded helpers and the skill catalog; when a claim about current behavior is needed, cite the skill that implements it, not the memory that describes it.

The rule binds the *discussion*. A skill the user invokes later (`/fab-new`, `/fab-proceed`, …) follows its own Context Loading section — the skill file wins — and is not constrained by this rule.

---

## Command Logging

After context loading, log the command invocation:

```bash
fab log command "fab-incognito"
```

Logging is deliberate: incognito governs what is *read*, not what is *recorded* (read-blind, not trace-free). The entry lands in the active change's `.history.jsonl`; with no active change `fab log command` exits 0 and records nothing — there is no project-level telemetry target.

---

## Behavior

1. Load the `fab/project/` layer (skip optional files gracefully)
2. Load the six process helpers via `helpers:`
3. Build the skill catalog from deployed frontmatter
4. Read the kit version
5. Resolve the active change via `fab resolve --folder --or-none`
6. Log the command
7. Output the **Orientation Summary** (see format below)
8. Hold the **Standing Session Rule** for the rest of the conversation

---

## Orientation Summary

```
Project: {name} — {description}
Kit: {version}   (deployed skills read from .agents/skills/)

Process helpers loaded:
  _preamble, _pipeline, _intake, _srad, _generation, _review

Skill catalog: {N} skills ({U} user-invocable, {H} helpers)

Project files: {loaded list} / not found: {missing optional list}

Memory and specs: NOT loaded (incognito — will not be opened unless you name a file)

Active change: {name} (stage: {stage})  — or "No active change"

Ready to discuss the system, incognito. What would you like to rethink?
```

The summary:
- Takes the catalog counts from the frontmatter scan and `Kit:` from `$(fab kit-path)/VERSION`
- Carries the `Memory and specs: NOT loaded …` line verbatim — it is the skill's defining property and the user's cue that the Standing Session Rule is in force
- Shows the active change name and stage if one exists (light touch — no deep loading)
- Ends with the ready signal, not a `Next:` pipeline command — a documented opt-out per `_preamble.md` § Next Steps Convention (the skill file wins, like `/fab-discuss`'s ready signal)

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `fab/project/config.yaml` or `fab/project/constitution.md` missing | Stop: `Project not initialized — run /fab-setup first.` |
| `.agents/skills/` missing or empty | Stop: `Deployed skills not found — run fab sync.` |
| `$(fab kit-path)/VERSION` missing | Not an error — print `Kit: unknown` |
| `fab resolve --folder --or-none` exits non-zero | Surface stderr per `_preamble.md`'s failure rule (`(none)` is the expected no-change result, not an error) |

---

## Key Properties

| Property | Value |
|----------|-------|
| Arguments | None |
| Requires active change? | No |
| Runs preflight? | No |
| Read-only? | Yes — modifies no files |
| Idempotent? | Yes |
| Advances stage? | No |
| Outputs `Next:` line? | No — ends with the incognito ready signal |
| Loads `docs/memory/*` / `docs/specs/*`? | **No** — and holds the Standing Session Rule afterwards |
