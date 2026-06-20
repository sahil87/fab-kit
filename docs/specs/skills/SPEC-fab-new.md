# fab-new

## Summary

Creates a new change from a natural language description, Linear ticket, or backlog ID. Since 260613-3xaj (extract-intake-helper) the skill is a **thin call-site** over the shared `_intake` Create-Intake Procedure: its body reads `.claude/skills/_intake/SKILL.md` and executes the **Create-Intake Procedure** (Steps 0–9) with `{questioning-mode} = interactive`, then runs its own **Steps 10–11 tail** (activate + git branch). The procedure generates the change folder, writes `intake.md`, verifies the hook-inferred change type (the PostToolUse intake-write hook owns `change_type`; the skill overrides via `set-change-type` only if wrong), computes the authoritative intake confidence (no `indicative` flag — 1.10.0), and advances intake to `ready`; `/fab-new`'s tail then activates the change and creates the matching git branch.

**Extraction boundary** (260613-3xaj): Steps 0–9 are NOT inlined here — they live in `_intake.md` (see `SPEC-_intake.md`). Only the activate (Step 10) + branch (Step 11) tail, Output block, and activation/git error rows stay in `fab-new.md` — a different responsibility (make the change active + checked out) that is NOT a questioning-mode parameter.

**Re-run contract** (idempotency — a fab-kit design principle): a backlog/Linear-ID re-run detects the existing non-archived change and routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of erroring; a natural-language re-run intentionally creates a new change each run. Implemented in the shared procedure's Step 3; declared in the skill's Key Properties section.

**Output ordering** (260612-c5tr): the Output template ends with the Assumptions summary as the final content block immediately before the `Next:` line, per `_srad.md` § Assumptions Summary Block (order: intake → Confidence → Activated → Branch → Assumptions → `Next:`); the block is omitted from output only when 0 assumptions were made.

**Helpers**: Declares `helpers: [_generation, _srad, _intake]` in frontmatter per `docs/specs/skills.md § Skill Helpers` (`_intake` added in 260613-3xaj; `_generation`/`_srad` kept declared directly, mirroring the `_pipeline` precedent where consumers declare underlying helpers alongside the orchestration helper).

**Prose optimization** (260620-skop): a `## Contents` TOC added to the skill file (>100 lines); no content trimmed (the skill is already thin post-`_intake` extraction) and no behavioral change (Flow / Tools / Sub-agents unchanged).

## Flow

```
User invokes /fab-new <description>
│
├─ Read: _preamble.md (always-load layer: 7 project files)
├─ Read: .claude/skills/_intake/SKILL.md   (helpers: declaration — also _generation, _srad)
│
├─ Steps 0–9: Create-Intake Procedure (_intake.md, {questioning-mode} = interactive)
│  │  (parse input → slug → gap analysis → create change [collision check]
│  │   → conversation mining → intake.md write ◄── HOOK → verify change type
│  │   → confidence score → SRAD questions [interactive] → advance to ready)
│  └─ see SPEC-_intake.md for the per-step tool detail
│
├─ Step 10: Activate Change
│  └─ Bash: fab change switch "{name}"
│
└─ Step 11: Create Git Branch (single first-match-wins table —
   │         260611-szxd f032; kept in sync with git-branch.md Step 4
   │         via an in-file comment; same cases, commands, and
   │         report strings)
   ├─ Bash: git rev-parse --is-inside-work-tree   (repo check — skip if fails)
   ├─ Context reads: git branch --show-current ·
   │  git status --porcelain | grep -v "fab/changes/{name}/" | wc -l
   │  ({dirty_count} — fab-new-only divergence: excludes the change's own
   │   just-created artifacts, which always exist uncommitted by Step 11;
   │   git-branch counts the full porcelain output) ·
   │  git rev-parse --verify "{name}" ·
   │  git rev-parse --verify "origin/{name}" ·
   │  git config branch.{current}.remote ·
   │  fab change resolve "$(git branch --show-current)"
   ├─ Evaluate the 6-row table in order, first match wins (260612-g8st):
   │  already-on-target (no-op) / target-exists-locally (checkout) /
   │  target-on-remote-only (checkout --track origin/{name}) /
   │  on-main (checkout -b) / local-only + rename guard passes —
   │  resolves to no change OR to this SAME change (branch -m) /
   │  different-change's local-only branch or pushed branch
   │  (checkout -b, leaving {old_branch} intact)
   └─ Dirty-tree note (260612-g8st): {dirty_count} > 0 on a
      checkout -b / branch -m row appends a non-blocking
      " — note: {N} uncommitted change(s) carried over from
      {old_branch}" to the report line (warn, never stash-prompt)
```

### Tools used

Steps 0–9 tool usage now lives in the shared procedure — see `SPEC-_intake.md` § Tools used (Read templates/backlog/project files, Write `intake.md`, Bash `fab change new`/`fab resolve`/`fab status set-change-type`/`fab score`/`fab status advance`/`fab status add-issue`, MCP Linear). `fab-new`'s own tail (Steps 10–11) uses:

| Tool | Purpose |
|------|---------|
| Read | `.claude/skills/_intake/SKILL.md` and the `helpers:` files (`_generation`, `_srad`); always-load layer |
| Bash | `fab change switch` (Step 10) |
| Bash (git) | `git rev-parse --is-inside-work-tree`, `git branch --show-current`, `git status --porcelain` (dirty count, excluding `fab/changes/{name}/`), `git rev-parse --verify` (local + `origin/{name}`), `git config branch.{current}.remote`, `git checkout -b`, `git checkout`, `git checkout --track`, `git branch -m` (Step 11) |

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

Steps 6/7/9 bookkeeping (`fab status set-change-type` override-only, `fab score --stage intake`, `fab status advance`) now belong to the shared procedure — see `SPEC-_intake.md`. `fab-new`'s tail:

| Step | Command | Trigger |
|------|---------|---------|
| 10 | `fab change switch` | After the Create-Intake Procedure advanced intake to ready |
| 11 | `git checkout -b` / `git checkout` / `git checkout --track` / `git branch -m` | After change activated |
