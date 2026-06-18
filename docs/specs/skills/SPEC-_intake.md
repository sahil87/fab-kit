# _intake

## Summary

Shared pre-boundary **Create-Intake Procedure** (fab-new Steps 0–9) used by three skills: `/fab-new`, `/fab-draft`, and `/fab-proceed`'s create-new dispatch (added in 260613-3xaj — extract-intake-helper). It completes the helper symmetry: the pre-boundary skill family (intake creation, which runs in the main session context because it needs the live conversation) now has a single shared body, mirroring the post-boundary `_pipeline.md`. Single authoritative source for Steps 0 (parse input) · 1 (generate slug) · 2 (gap analysis) · 3 (create change, incl. backlog/Linear collision pre-checks and `fab change new` flags) · 4 (conversation context mining — the load-bearing context-flush at the boundary) · 5 (generate `intake.md`, delegating to `_generation.md` § Intake Generation Procedure) · 6 (verify hook-owned `change_type`) · 7 (confidence, `fab score --stage intake`) · 8 (SRAD-based question selection — *the parameterized step*) · 9 (advance intake to `ready`).

**Parameter** (bound by each consumer's own file):

| Parameter | `/fab-new` | `/fab-draft` | `/fab-proceed` dispatch |
|-----------|-----------|--------------|-------------------------|
| `{questioning-mode}` — how Step 8 resolves ambiguity | `interactive` | `interactive` | `promptless-defer` |

- **`interactive`** — Step 8 asks the user via SRAD (no fixed cap; conversational mode when 5+ Unresolved). The existing `/fab-new`/`/fab-draft` behavior.
- **`promptless-defer`** — Step 8 records each would-be-asked Unresolved decision as an Unresolved row with Rationale `Deferred — promptless dispatch` instead of asking, per the `_srad.md` § Critical Rule promptless-dispatch carve-out (quoted verbatim in the helper). The intake gate is the structural backstop: a deferred decision blocks only when its composite is below 20 (emergent from the demerit curve, no special gate) — so a genuine unknown must be scored with honestly-low dimensions to land it there.

This is the **only** behavioral fork in intake creation, and it is legitimately invocation-level (who resolves ambiguity: human-now vs. defer-and-surface) — exactly parallel to the post-boundary autonomy fork.

**Extraction boundary** (do NOT over-extract — the procedure is purely "given I've decided to create an intake, do it, Steps 0–9"):
- **Activate (Step 10) + branch (Step 11)** stay as a tail in `fab-new.md` — a different responsibility (make the change active + checked out vs. queue it), not a questioning-mode parameter.
- **`/fab-proceed`'s state detection + relevance assessment** stay in `fab-proceed.md` — they decide *whether* to call `_intake`, not how to create one.

**Self-name genericization** (260613-3xaj): the lifted Step 4 refers to "the invoking skill" / "this invocation" rather than "this `/fab-new` invocation", structurally retiring `fab-draft`'s former "read self-name mentions as `/fab-draft`" prose instruction. No `{self-name}` parameter — the text is invocation-agnostic, not invocation-named.

**Helpers**: carries NO `helpers:` frontmatter. It references `_generation` (Step 5) and `_srad` (Step 8) in-body and relies on the consumer having loaded them — the consumer-declared model, matching `_pipeline`/`_review`/`_generation` (none of which carry `helpers:`). `/fab-new` and `/fab-draft` declare `helpers: [_generation, _srad, _intake]`; `/fab-proceed` declares none and dispatches `_intake` to a subagent that loads them.

This is an internal partial (`user-invocable: false`, `disable-model-invocation: true`, `metadata: internal: true`) — never invoked directly. Canonical source is the flat `src/kit/skills/_intake.md`; `fab sync` deploys it to `.claude/skills/_intake/SKILL.md`.

## Flow

```
Consumer (fab-new / fab-draft / fab-proceed dispatch) reads _intake.md with {questioning-mode} bound
│
├─ Step 0: Parse Input
│  ├─ Linear ID? ──► MCP: mcp__claude_ai_Linear__get_issue
│  ├─ Backlog ID? ──► Read: fab/backlog.md (optional [ISSUE_ID] bracket)
│  └─ Natural language ──► use as-is
│
├─ Step 1: Generate Slug (2-6 word kebab; SHALL NOT include Linear issue ID)
│
├─ Step 2: Gap Analysis (existing mechanisms / scope concerns)
│
├─ Step 3: Create Change
│  ├─ [backlog ID] collision pre-check: fab resolve --id {id} → EQUALITY with {id};
│  │  on match, fab resolve --folder {id} names the existing change → route to resume
│  ├─ [Linear ID] collision pre-check: grep -lw "{ISSUE_ID}" fab/changes/*/.status.yaml
│  ├─ [existing non-archived change] → route to resume (/fab-switch + /fab-continue), STOP
│  │  (NL re-run intentionally creates a new change each run)
│  ├─ Bash: fab change new --slug <slug> --log-args <desc> [--change-id <4char> if backlog]
│  └─ [if Linear] Bash: fab status add-issue <change> <id>
│
├─ Step 4: Conversation Context Mining (context-flush at the boundary)
│  └─ Extract decisions / rejected alternatives / constraints / specific values
│     → encode as Certain/Confident assumptions in the intake table
│     (genericized: "the invoking skill", not "this /fab-new invocation")
│
├─ Step 5: Generate intake.md
│  └─ Delegate to _generation.md § Intake Generation Procedure   ◄── HOOK CANDIDATE (intake write)
│
├─ Step 6: Verify Change Type (hook-owned — the intake-write hook set it in Step 5)
│  ├─ Bash: grep '^change_type:' fab/changes/{name}/.status.yaml
│  └─ [only if wrong] Bash: fab status set-change-type <change> <type>
│
├─ Step 7: Confidence (authoritative — intake is the sole scoring source)
│  └─ Bash: fab score --stage intake <change>             ◄── bookkeeping
│
├─ Step 8: SRAD-Based Question Selection  *(THE PARAMETERIZED STEP)*
│  ├─ {questioning-mode} = interactive → ask via SRAD (no cap; conversational at 5+ Unresolved)
│  └─ {questioning-mode} = promptless-defer → record each Unresolved as
│     "Deferred — promptless dispatch" row; return them for the dispatcher to surface
│
└─ Step 9: Advance Intake to Ready
   └─ Bash: fab status advance <change> intake   (then control returns to the call site)
```

### Tools used

| Tool | Purpose |
|------|---------|
| Read | `_generation.md` (Step 5), `_srad.md` (Step 8 — both questioning modes), templates, backlog, project files |
| Write | `intake.md` (via the Intake Generation Procedure) |
| Bash | `fab change new`, `fab resolve --id`/`--folder` (collision pre-check), `fab status set-change-type` (override only), `fab score`, `fab status advance`, `fab status add-issue` |
| MCP (Linear) | Fetch issue details (optional path) |

### Sub-agents

None — the procedure runs inside the consuming skill's (or dispatched subagent's) context. Under `/fab-proceed` the *procedure itself* is what is dispatched as a subagent (per `fab-proceed.md` § Create-Intake Dispatch); it spawns no further sub-agents.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| 6 | `fab status set-change-type` | Only if the hook-inferred type is wrong (the intake-write hook owns `change_type`) |
| 7 | `fab score --stage intake` | After intake.md write |
| 9 | `fab status advance` | After all intake work complete |
