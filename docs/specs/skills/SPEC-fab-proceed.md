# fab-proceed

## Summary

Context-aware orchestrator — detects pipeline state via a 5-step detection pipeline, runs prefix steps (fab-new, fab-switch, git-branch) as subagents, then delegates to `/fab-fff` via the Skill tool. No arguments, no flags — infers everything from context. Idempotent — re-running detects completed steps and skips them. Reads `_preamble.md` (per skill convention) but skips running preflight and defers project-context loading to `/fab-fff`.

Conversation context is the interpretive lens for any unactivated intakes: an unactivated intake is only resumed when it is clearly relevant to the current conversation or there is no competing conversation signal. An unrelated draft never hijacks the pipeline when the current conversation is about a different topic.

## Flow

```
User invokes /fab-proceed
│
├─ Step 1: Active Change Check
│  └─ Bash: fab resolve --folder 2>/dev/null
│     ├─ exits 0 → active change found, go to Step 2
│     └─ exits non-zero → run Steps 3 and 4 (order-independent)
│
├─ Step 2: Branch Check (only if active change found)
│  └─ Bash: git branch --show-current
│     ├─ matches change name → dispatch /fab-fff only
│     └─ does not match → dispatch /git-branch → /fab-fff
│
├─ Step 3: Conversation Classification (only if no active change)
│  └─ Classify conversation as substantive or empty/thin
│     Substantive = contains at least one of:
│       technical requirements, design decisions,
│       specific values, problem statements.
│     Anything else (greeting-only, chatty, empty) = empty/thin.
│     Single classifier — no "thin but non-empty" tier.
│
├─ Step 4: Unactivated Intake Scan (only if no active change)
│  └─ ls -d fab/changes/*/intake.md | grep -v archive/ | sort -t- -k1,1r
│     ├─ ≥1 candidate → retain full list for relevance check
│     └─ none
│
├─ Step 5: Dispatch Decision — combines Steps 1-4
│  └─ Apply the dispatch table (see below).
│     When substantive + ≥1 intake, run Relevance Assessment
│     across ALL candidates:
│       • Read title + Origin + Why + What Changes per candidate
│       • Clearly relevant = shared topic + overlapping terminology
│         + consistent scope (partial/vague overlap does NOT qualify)
│       • Ambiguous → not clearly relevant (asymmetric-bias rule)
│       • Date-recency tiebreak ONLY among equally-relevant candidates
│
├─ Prefix Dispatch (subagents)
│  ├─ ┌──────────────────────────────────────────┐
│  │  │ SUB-AGENT: /fab-new (if dispatched)      │
│  │  │  Read: .claude/skills/fab-new/SKILL.md   │
│  │  │  Input: synthesized description          │
│  │  │    (from conversation ONLY —             │
│  │  │     never from bypassed drafts)          │
│  │  │  Returns: created change folder name     │
│  │  └──────────────────────────────────────────┘
│  ├─ ┌──────────────────────────────────────────┐
│  │  │ SUB-AGENT: /fab-switch (if dispatched)   │
│  │  │  Read: .claude/skills/fab-switch/SKILL.md│
│  │  │  Bash: fab change switch "<change-name>" │
│  │  │  Returns: switch confirmation            │
│  │  └──────────────────────────────────────────┘
│  └─ ┌──────────────────────────────────────────┐
│     │ SUB-AGENT: /git-branch (if dispatched)   │
│     │  Read: .claude/skills/git-branch/SKILL.md│
│     │  Returns: branch action result           │
│     └──────────────────────────────────────────┘
│
└─ Terminal Delegation (Skill tool, NOT subagent)
   └─ Skill: /fab-fff
      └─ Runs in main context with full user visibility
```

### Dispatch Table

| Active change? | Branch matches? | Conversation | Unactivated intake? | Relevant? | Prefix steps | Terminal |
|----------------|-----------------|--------------|---------------------|-----------|--------------|----------|
| Yes | Yes | — | — | — | (none) | /fab-fff |
| Yes | No | — | — | — | /git-branch | /fab-fff |
| No | — | Substantive | None | — | /fab-new → /git-branch | /fab-fff |
| No | — | Substantive | ≥1 | Clearly relevant | /fab-switch → /git-branch | /fab-fff |
| No | — | Substantive | ≥1 | Not clearly relevant | /fab-new → /git-branch (emit bypass notes) | /fab-fff |
| No | — | Empty/thin | ≥1 | — | /fab-switch → /git-branch (pick by date-recency) | /fab-fff |
| No | — | Empty/thin | None | — | (error — stop) | — |

### Asymmetric-Bias Rule

When relevance is genuinely ambiguous, the candidate MUST be classified as *not clearly relevant*. Failure modes are asymmetric:

- **False positive** (activate unrelated draft): corrupts the draft, conflates features in pipeline output, recovery requires manual rollback.
- **False negative** (create new when draft was relevant): draft remains intact; user sees the bypass note and can run `/fab-switch {name}` to recover.

Biasing toward the recoverable failure is the design intent.

### Sub-agents

| Agent | When | Purpose |
|-------|------|---------|
| /fab-new | Substantive + no intake, OR substantive + ≥1 intake but none clearly relevant | Create change from synthesized description (conversation only — never bypassed drafts) |
| /fab-switch | Substantive + clearly relevant intake, OR empty/thin + ≥1 intake | Activate the selected change |
| /git-branch | Any non-error row except "active + matching branch" | Create or checkout the matching branch |

### Output Format

When bypassed drafts exist, emit one Note line per draft BEFORE any step reports:

```
Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.
```

Multiple Notes appear in date-descending order. No Notes on the activation path or on the empty/thin + activate branch.

### Key differences from /fab-fff and /fab-ff

- Does NOT load `_preamble.md` or run preflight — delegates that to `/fab-fff`
- Does NOT accept arguments or flags — infers everything from state detection
- Prefix steps are subagents; terminal `/fab-fff` is via Skill tool (not subagent)
- Zero-prompt posture — relevance ambiguity is resolved by the asymmetric-bias rule, never by asking the user
