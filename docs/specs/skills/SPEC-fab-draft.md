# fab-draft

## Summary

Creates a new change intake without activating the change. Since 260613-3xaj (extract-intake-helper) the skill is a **thin call-site** over the shared `_intake` Create-Intake Procedure: its body reads `.claude/skills/_intake/SKILL.md` and executes the **Create-Intake Procedure** (Steps 0–9) with `{questioning-mode} = interactive`, then **stops at intake `ready`** — no activation (no Step 10), no git branch (no Step 11). Used to queue changes for later without switching the active context. After creation, run `/fab-switch {name}` to activate.

Before 260613-3xaj, `fab-draft` was a thin *delta over `/fab-new`* (read `fab-new/SKILL.md`, execute its Steps 0–9, skip 10–11). That form carried a **momentum warning** — "running activation/branch by momentum is the known failure mode of this delta" — precisely because the steps it must NOT run (activate/branch) lived in the same `fab-new.md` body it executed. With Steps 0–9 lifted into `_intake.md`, the warning **evaporates**: `fab-draft` now reads `_intake.md` (Steps 0–9 only), and Steps 10–11 live solely in `fab-new.md`'s tail, which `fab-draft` never reads. There is no longer any body containing the not-to-run steps, so there is no momentum hazard.

**Re-run contract** (Constitution III): inherited from the shared procedure's Step 3 — a backlog/Linear-ID re-run detects the existing non-archived change and routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of erroring; a natural-language re-run intentionally creates a new change each run. Declared in the skill's Key Properties section.

**Helpers**: Declares `helpers: [_generation, _srad, _intake]` in frontmatter per `docs/specs/skills.md § Skill Helpers` (`_intake` added in 260613-3xaj; the executed Create-Intake Procedure references `_generation`/`_srad` in-body, so the consumer keeps declaring both directly — the `_pipeline` precedent).

## Difference from /fab-new

`/fab-draft` and `/fab-new` run the **same** shared Create-Intake Procedure with the **same** `{questioning-mode} = interactive`. The only difference is the tail:

| | `/fab-new` | `/fab-draft` |
|---|-----------|--------------|
| Steps 0–9 | `_intake(interactive)` | `_intake(interactive)` |
| Step 10 (activate) | Yes (`fab change switch`) | **No** — `.fab-status.yaml` symlink not created |
| Step 11 (git branch) | Yes | **No** |
| Output | with `Activated:`/`Branch:` lines | minus those lines; `Next:` per Activation Preamble |
| Error Handling | + activation/git rows | shared-procedure rows only (no activation/git — those steps never run) |

The output `Next:` line uses the activation preamble: `/fab-switch {name} to make it active, then /fab-continue, /fab-ff, /fab-fff, /fab-proceed, or /fab-clarify` — the command list after "then" is the state table's intake row, derived per `_preamble.md` § Lookup Procedure (default first, not hardcoded).

## Flow

```
User invokes /fab-draft <description>
│
├─ Read: _preamble.md (always-load layer: 7 project files)
├─ Read: .claude/skills/_intake/SKILL.md   (helpers: declaration — also _generation, _srad)
│
├─ Steps 0–9: Create-Intake Procedure (_intake.md, {questioning-mode} = interactive)
│  (parse input → slug → gap analysis → create change [collision check]
│   → conversation mining → intake.md write ◄── HOOK → verify change type
│   → confidence score → SRAD questions [interactive] → advance intake to ready)
│  — see SPEC-_intake.md for the per-step tool detail
│
└─ STOP after Step 9 (no activation, no git branch;
   .fab-status.yaml symlink is NOT created)
```

### Tools used

Same as the shared Create-Intake Procedure Steps 0–9 (see `SPEC-_intake.md`): Read, Write (`intake.md`), Bash (`fab change new`, `fab status set-change-type` override-only, `fab score`, `fab status advance`, `fab status add-issue`), MCP (Linear, optional). No `fab change switch`, no git commands.

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

All bookkeeping belongs to the shared procedure — see `SPEC-_intake.md` (Step 6 `fab status set-change-type` override-only, Step 7 `fab score --stage intake`, Step 9 `fab status advance`). `fab-draft` adds none (no Step 10/11).
