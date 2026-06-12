---
name: fab-proceed
description: "Context-aware orchestrator — detects state, runs prefix steps (fab-new, fab-switch, git-branch as needed), then delegates to fab-fff. Takes no arguments — infers everything from conversation; use /fab-fff <change> to target a named change."
---

# /fab-proceed

Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

> `/fab-proceed` follows `_preamble.md` conventions but skips preflight/context loading itself — it delegates all pipeline context loading to `/fab-fff`.

---

## Purpose

Detect the current pipeline state and automatically run whatever prefix steps are needed (fab-new, fab-switch, git-branch) before handing off to `/fab-fff` for the full pipeline. Zero-argument, zero-flag — the skill infers everything from context. Idempotent — re-running detects completed steps and skips them.

Conversation context is the interpretive lens for any unactivated intakes that exist: an unactivated intake is only resumed when it is either clearly relevant to the current conversation or there is no competing conversation signal. An unrelated draft NEVER hijacks the pipeline when the current conversation is about a different topic.

---

## Arguments

None. `/fab-proceed` does not accept arguments or flags. Any arguments passed are silently ignored.

---

## State Detection

Detect the current state by executing the following checks. The skill MUST NOT prompt the user for input at any detection step — it either resolves automatically or errors. Steps 3 and 4 produce independent signals and MAY execute in either order; both feed into Step 5.

### Step 1: Active Change Check

```bash
fab resolve --folder 2>/dev/null
```

If exits 0, an active change exists. Capture the folder name and go to Step 2. If exits non-zero, skip Step 2 and proceed to Steps 3 and 4.

### Step 2: Branch Check

*(Only runs when Step 1 found an active change.)*

Compare the current git branch with the resolved change folder name:

```bash
git branch --show-current
```

If the current branch matches the change folder name, the branch is already set up.

When Step 1 found an active change, Steps 3 and 4 SHALL NOT run — dispatch uses only the active-change rows of the dispatch table.

### Step 3: Conversation Classification

*(Only runs when no active change was found in Step 1.)*

Classify the prior conversation as **substantive** or **empty/thin**. Substantive means the conversation contains at least one of:

- Technical requirements
- Design decisions
- Specific values (config structures, API shapes, exact behaviors)
- Problem statements with enough detail to generate an intake

Anything else — greeting-only, chatty, literally empty — is **empty/thin**. This is the single classifier for `/fab-proceed`: there is no separate "thin but non-empty" tier, no word-count threshold, no domain-terminology match.

### Step 4: Unactivated Intake Scan

*(Only runs when no active change was found in Step 1.)*

Enumerate candidate intakes:

```bash
ls -d fab/changes/*/intake.md 2>/dev/null | grep -v archive/ | sed 's|fab/changes/||;s|/intake.md||' | sort -t- -k1,1r
```

The pipeline lists change folders with intakes, excludes archived changes, extracts folder names, and sorts by `YYMMDD` date prefix in descending order. Retain the full list — the date-descending sort is used only for tiebreaks in Step 5, not to pre-pick a single candidate.

### Step 5: Dispatch Decision

Combine the signals from Steps 1–4 per the dispatch table below.

#### Dispatch Table

| Active change? | Branch matches? | Conversation | Unactivated intake? | Relevant? | Steps to run |
|----------------|-----------------|--------------|---------------------|-----------|--------------|
| Yes | Yes | — | — | — | `/fab-fff` only |
| Yes | No | — | — | — | `/git-branch` → `/fab-fff` |
| No | — | Substantive | None | — | `/fab-new` → `/git-branch` → `/fab-fff` |
| No | — | Substantive | ≥1 | Clearly relevant | `/fab-switch` → `/git-branch` → `/fab-fff` |
| No | — | Substantive | ≥1 | Not clearly relevant | `/fab-new` → `/git-branch` → `/fab-fff` (emit bypass notes) |
| No | — | Empty/thin | ≥1 | — | `/fab-switch` → `/git-branch` → `/fab-fff` (pick by date-recency) |
| No | — | Empty/thin | None | — | Error — stop |

The `Relevant?` column is evaluated only when Conversation is Substantive AND Unactivated intake is ≥1. In the Empty/thin + ≥1 intake row, no relevance check runs — pick the most-recent by `YYMMDD` prefix. This preserves the "resume yesterday's draft" flow, and is safe because an empty/thin conversation carries no competing signal that could conflict with the intake's content.

---

## Relevance Assessment

*(Applies only when Step 3 classified the conversation as Substantive AND Step 4 found ≥1 unactivated intake.)*

For each candidate intake, score its topical relevance to the current conversation:

1. Read the candidate's `intake.md`: title heading, `## Origin`, `## Why`, and `## What Changes` sections (at minimum). Do not rely on the folder slug alone — slugs are terse and routinely misrepresent content.
2. Judge topical overlap between each intake and the conversation. "Clearly relevant" requires shared topic, overlapping terminology, and consistent scope. Partial, vague, or tangential overlap MUST NOT qualify.
3. Classify each candidate as **clearly relevant** or **not clearly relevant**.
4. If ≥1 candidate is clearly relevant: select the best match. If multiple candidates are equally clearly relevant, use the date-descending prefix tiebreak (`sort -t- -k1,1r | head -1`).
5. If no candidate is clearly relevant: fall through to `/fab-new`, and surface every scanned draft as a bypass note (see Output Format).

### Asymmetric-Bias Rule

When a candidate's relevance is genuinely ambiguous (neither clearly relevant nor clearly unrelated), it MUST be classified as **not clearly relevant**. This biases toward creating a new intake.

The failure modes are asymmetric:

- **False positive** (activate unrelated draft): corrupts the draft's content, wastes its original intent, conflates two unrelated features in pipeline output. Recovery requires manual rollback.
- **False negative** (create new when draft was relevant): leaves the draft intact and recoverable. The user sees the bypass note, and can run `/fab-switch <name>` to recover.

Biasing toward the recoverable failure is the design intent.

Relevance judgment is performed by the invoking agent inline — no external classifier, embedding index, or `fab` subcommand is added.

---

## Dispatch Behavior

### Subagent Dispatch (Prefix Steps)

Each prefix step (`/fab-new`, `/fab-switch`, `/git-branch`) SHALL be dispatched as a subagent using the Agent tool (`subagent_type: "general-purpose"`) per `_preamble.md` § Subagent Dispatch. Each subagent prompt MUST instruct the subagent to read the standard subagent context files per `_preamble.md` § Standard Subagent Context.

#### fab-new Dispatch

Runs when the dispatch table selects `/fab-new`: either substantive conversation + no intake, or substantive conversation + ≥1 intake but none clearly relevant.

1. Synthesize a description from the conversation (see Conversation Context Synthesis below). The synthesis MUST NOT pull from bypassed drafts — only the live conversation is the source.
2. Dispatch subagent: read `.claude/skills/fab-new/SKILL.md`, invoke `/fab-new` with the synthesized description
3. Capture the created change folder name from the subagent result

#### fab-switch Dispatch

Runs when the dispatch table selects `/fab-switch` (substantive + clearly relevant, or empty/thin + ≥1 intake).

1. Dispatch subagent: read `.claude/skills/fab-switch/SKILL.md`, invoke `fab change switch "<change-name>"`
2. Capture the switch confirmation from the subagent result

#### git-branch Dispatch

Runs when the dispatch table selects `/git-branch` (all non-error branches except the "active change + matching branch" one).

1. Dispatch subagent: read `.claude/skills/git-branch/SKILL.md`, follow its behavior for the active change
2. Capture the branch creation/checkout result from the subagent result

### Conversation Context Synthesis

When `/fab-proceed` dispatches `/fab-new`, it SHALL synthesize a description from the conversation by extracting:

- **Decisions made** — specific choices with rationale
- **Alternatives rejected** — options considered and why they were ruled out
- **Constraints identified** — boundaries or requirements surfaced
- **Specific values agreed upon** — config structures, API shapes, exact behaviors

The synthesized description MUST be substantive enough for `/fab-new` to generate a complete intake without prompting. Do not fabricate details — capture what was said. Do not mix in content from bypassed drafts; if a bypassed draft contains overlapping details, ignore them during synthesis — the bypassed draft is left untouched for the user to reconcile later.

### fab-fff Terminal Delegation

The final `/fab-fff` invocation is NOT dispatched as a subagent — it is invoked via the Skill tool in the current context. This ensures `/fab-fff` runs in the main context with full user visibility of its output, confidence gates, and pipeline progress.

The skill SHALL NOT pass `--force` or any other flags to `/fab-fff`. If `/fab-fff` fails a confidence gate, it stops normally and the user intervenes.

---

## Error Handling

| Condition | Action |
|-----------|--------|
| Empty/thin conversation and no intake | Output: `Nothing to proceed with — start a discussion or run /fab-new (or /fab-draft) first.` Stop. |
| fab-new subagent fails | Surface the error from fab-new and stop. Do not proceed to further steps. |
| fab-switch subagent fails | Surface the error from fab-switch and stop. |
| git-branch subagent fails | Surface the error from git-branch and stop. |
| fab-fff gate failure | `/fab-fff` stops normally with its own gate failure message. `/fab-proceed` does not retry or bypass the gate. |

Errors from any sub-skill propagate to the user and halt execution. The skill does not retry failed steps. When a decision is genuinely irresolvable (e.g., malformed intake files that cannot be parsed for relevance), the skill SHALL error with a descriptive message rather than prompt — the zero-prompt posture applies to error paths as well.

---

## Output

```
/fab-proceed — detecting state...

{Bypass notes, if any — one line per bypassed draft, emitted BEFORE any step reports}

{Step reports, one per line — only for steps actually executed}

Handing off to /fab-fff...
{fab-fff takes over and produces its own output}
```

### Bypass Notes

Emitted only when the dispatch table selected `/fab-new` despite ≥1 unactivated intake being present (the "Substantive + ≥1 intake + Not clearly relevant" row). For each scanned unactivated intake, emit one line using this exact wording:

```
Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.
```

When multiple drafts are bypassed, Note lines SHALL be emitted in date-descending order (matching the scan order). Bypass notes appear BEFORE any step reports so the reader sees context before action.

When the skill activates an existing intake (the "clearly relevant" row or the "empty/thin + ≥1 intake" row), or when no unactivated intakes were scanned, NO bypass notes are emitted.

### Step Report Format

Only for steps actually executed:

- `Created intake: {change-name}` (when `/fab-new` ran)
- `Activated: {change-name}` (when `/fab-switch` ran)
- `Branch: {branch-name} ({action})` (when `/git-branch` ran; action = created / checked out / already active)

When only `/fab-fff` is needed (active change + matching branch), output shows only the detecting-state line and the handoff line before `/fab-fff` output.

---

## Key Properties

| Property | Value |
|----------|-------|
| Arguments | None |
| Flags | None |
| Requires active change? | No — can create one from conversation context or activate a relevant draft |
| Runs preflight? | No — delegates to `/fab-fff` |
| Read-only? | No — may create change, switch pointer, create branch |
| Idempotent? | Yes — re-running detects completed steps and skips them |
| Advances stage? | No directly — `/fab-fff` handles stage advancement |
| Outputs Next line? | Inherits from `/fab-fff` |
