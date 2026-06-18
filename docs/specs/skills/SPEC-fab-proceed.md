# fab-proceed

## Summary

Context-aware orchestrator — detects pipeline state via a 5-step detection pipeline, runs prefix steps (create-intake via `_intake`, fab-switch, git-branch) as subagents, then delegates to `/fab-fff` via the Skill tool. No arguments, no flags — infers everything from context. Idempotent — re-running detects completed steps and skips them. Reads `_preamble.md` (per skill convention) but skips running preflight and defers project-context loading to `/fab-fff`. **Per-stage model** (260613-l3ja): the prefix steps are NOT pipeline stages and take no `fab resolve-agent` resolution (they dispatch at the inherited model); per-stage model selection belongs to the delegated `/fab-fff`, which resolves each of its own stages per `_preamble.md` § Subagent Dispatch → Per-Stage Model Resolution.

**Create-intake dispatch via `_intake` (260613-3xaj)**: the create-new path no longer dispatches the full `/fab-new` skill. It dispatches the shared `_intake` Create-Intake Procedure (read `.claude/skills/_intake/SKILL.md`) with `{questioning-mode} = promptless-defer` — `promptless-defer` IS the defer-and-surface contract (Unresolved decisions → `Deferred — promptless dispatch` rows, surfaced before `/fab-fff`; the intake gate is the structural backstop). `/fab-proceed`'s state-detection + relevance-assessment logic — *whether* to create an intake vs. activate an existing draft — STAYS in `fab-proceed.md` (it decides whether to call `_intake`, not how to create one). Because `_intake` stops at intake `ready` and does NOT activate or branch (those are `/fab-new`'s call-site tail, omitted here), the create-new dispatch-table rows now chain `_intake` → `/fab-switch` → `/git-branch` to reach the same end state (active change + matching branch) the prior full-`/fab-new` dispatch produced inline — a parity-preserving consequence of the extraction.

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
│  └─ ls -d fab/changes/*/intake.md 2>/dev/null | grep -v archive/
│     | sed 's|fab/changes/||;s|/intake.md||' | sort -r
│     (full-folder-name descending — YYMMDD dominates, XXXX-slug
│      tail makes same-day order deterministic)
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
│       • Date-descending full-folder-name tiebreak (sort -r | head -1,
│         deterministic even same-day) ONLY among equally-relevant candidates
│
├─ Prefix Dispatch (subagents)
│  ├─ ┌──────────────────────────────────────────┐
│  │  │ SUB-AGENT: _intake (if create-new path)  │
│  │  │  Read: .claude/skills/_intake/SKILL.md   │
│  │  │  Procedure: Create-Intake Steps 0–9      │
│  │  │   {questioning-mode} = promptless-defer  │
│  │  │  Input: synthesized description          │
│  │  │    (from conversation ONLY —             │
│  │  │     never from bypassed drafts)          │
│  │  │  promptless-defer = ask NO questions;    │
│  │  │   Unresolved → intake row                │
│  │  │   "Deferred — promptless dispatch"       │
│  │  │  Stops at intake ready (no activate/      │
│  │  │   branch — chained as /fab-switch +      │
│  │  │   /git-branch below)                     │
│  │  │  Returns: created change folder name     │
│  │  │   + deferred Unresolved decisions        │
│  │  │   (surfaced before /fab-fff)             │
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
| No | — | Substantive | None | — | _intake → /fab-switch → /git-branch | /fab-fff |
| No | — | Substantive | ≥1 | Clearly relevant | /fab-switch → /git-branch | /fab-fff |
| No | — | Substantive | ≥1 | Not clearly relevant | _intake → /fab-switch → /git-branch (emit bypass notes) | /fab-fff |
| No | — | Empty/thin | ≥1 | — | /fab-switch → /git-branch (pick by date-recency) | /fab-fff |
| No | — | Empty/thin | None | — | (error — stop) | — |

**Create-new chaining (260613-3xaj)**: the create-new rows chain `_intake` → `/fab-switch` → `/git-branch`. Before 3xaj they dispatched the full `/fab-new` skill, whose Steps 10–11 activated + branched inline (so 260612-w7dp dropped the redundant trailing `/git-branch`). Now `_intake` stops at intake `ready` without activating or branching (the EXTRACTION BOUNDARY keeps activate/branch as `fab-new.md`'s tail), so `/fab-proceed` runs the dedicated `/fab-switch` (activate) + `/git-branch` prefix steps — which it already has — to reach the same end state. This makes the create-new rows symmetric with the relevant-intake rows.

### Asymmetric-Bias Rule

When relevance is genuinely ambiguous, the candidate MUST be classified as *not clearly relevant*. Failure modes are asymmetric:

- **False positive** (activate unrelated draft): corrupts the draft, conflates features in pipeline output, recovery requires manual rollback.
- **False negative** (create new when draft was relevant): draft remains intact; user sees the bypass note and can run `/fab-switch {name}` to recover.

Biasing toward the recoverable failure is the design intent.

### Sub-agents

| Agent | When | Purpose |
|-------|------|---------|
| _intake (Create-Intake Procedure) | Substantive + no intake, OR substantive + ≥1 intake but none clearly relevant | Create change from synthesized description (conversation only — never bypassed drafts) via the shared procedure (read `.claude/skills/_intake/SKILL.md`) with `{questioning-mode} = promptless-defer` (260613-3xaj — replaces the prior full-`/fab-new` dispatch). `promptless-defer` IS the **defer-and-surface contract** (260612-w7dp): asks no questions; would-be-asked Unresolved decisions land in the intake's `## Assumptions` as `Deferred — promptless dispatch` rows, are returned in the result, and `/fab-proceed` surfaces them before delegating to `/fab-fff` (whose intake gate is the structural backstop — a deferred decision blocks the gate only when its composite is below 20; blocking is emergent from the demerit curve, not a special gate, so a genuine unknown scored with honestly-low dimensions fails the gate). This is the `_srad.md` § Critical Rule promptless-dispatch carve-out — defer-and-surface satisfies the MUST-ask when no user is reachable. The procedure stops at intake `ready` (no activate/branch), so the create-new rows chain `/fab-switch` + `/git-branch` after it |
| /fab-switch | Substantive + clearly relevant intake, OR empty/thin + ≥1 intake, OR after `_intake` on the create-new rows (activate the just-created change) | Activate the selected change |
| /git-branch | Branch-mismatch row (active change, branch doesn't match), the /fab-switch-prefixed relevant-intake rows, AND the `_intake`-prefixed create-new rows (since `_intake` stops at `ready` without branching — 260613-3xaj; before 3xaj the create-new path used full `/fab-new` whose Step 11 branched inline, so 260612-w7dp had dropped the trailing /git-branch there) | Create or checkout the matching branch |

### Bypass Notes

When bypassed drafts exist, emit one Note line per draft BEFORE any step reports:

```
Note: unactivated draft {name} exists — not relevant to current conversation, left untouched.
```

Multiple Notes appear in date-descending order. No Notes on the activation path or on the empty/thin + activate branch.

### Key differences from /fab-fff and /fab-ff

- Reads `_preamble.md` (per skill convention) but skips preflight/context loading itself — pipeline context loading is delegated to `/fab-fff`
- Does NOT accept arguments or flags — infers everything from state detection
- Prefix steps are subagents; terminal `/fab-fff` is via Skill tool (not subagent)
- Zero-prompt posture — relevance ambiguity is resolved by the asymmetric-bias rule, never by asking the user
