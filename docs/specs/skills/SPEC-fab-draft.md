# fab-draft

## Summary

Creates a new change intake without activating the change. Since 260611-szxd (f031) the skill file is a **thin delta over `/fab-new`**: its body instructs the agent to read `.claude/skills/fab-new/SKILL.md` and execute its Pre-flight, Arguments, and Steps 0–9 exactly as written there (self-name mentions read as `/fab-draft`), with four deltas — there is no duplicated copy of the shared steps. Used to queue changes for later without switching the active context. After creation, run `/fab-switch {name}` to activate.

**Re-run contract** (Constitution III): inherited from fab-new Steps 0–9 — a backlog/Linear-ID re-run detects the existing non-archived change and routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of erroring; a natural-language re-run intentionally creates a new change each run. Declared in the skill's Key Properties section (kept locally — it IS the delta).

**Helpers**: Declares `helpers: [_generation, _srad]` in frontmatter per `docs/specs/skills.md § Skill Helpers` (the executed fab-new steps need both).

## Delta over /fab-new

| # | Delta |
|---|-------|
| 1 | **Step 9 tail**: after `fab status advance {name} intake`, the change is NOT activated — the user must run `/fab-switch {name}` (replaces fab-new's Step 9 closing sentence about Step 10) |
| 2 | **Skip Steps 10–11 entirely** — no `fab change switch`, no `git` command. Stated explicitly and prominently in the skill body (the known failure mode of the delta form is an agent running activation by momentum; the body instructs a re-check before any `fab change switch`/`git` invocation) |
| 3 | **Output**: fab-new's Output block minus the `Activated:` and `Branch:` lines; `Next:` per the Activation Preamble convention (`_preamble.md` § Activation Preamble — names `/fab-draft`) |
| 4 | **Error Handling**: fab-new's table minus the activation/git rows |

## Flow

```
User invokes /fab-draft <description>
│
├─ Read: _preamble.md (always-load layer: 7 project files)
├─ Read: .claude/skills/fab-new/SKILL.md   ◄── the delta indirection
│
├─ Execute fab-new Pre-flight, Arguments, Steps 0–9
│  (parse input → slug → gap analysis → create change [collision check]
│   → conversation mining → intake.md write ◄── HOOK → verify change type
│   → confidence score → SRAD questions → advance intake to ready)
│  — see SPEC-fab-new.md for the per-step tool detail
│
└─ STOP after Step 9 (deltas 1–2: no activation, no git branch;
   .fab-status.yaml symlink is NOT created)
```

### Tools used

Same as `/fab-new` Steps 0–9 (see `SPEC-fab-new.md`): Read, Write (`intake.md`), Bash (`fab change new`, `fab status set-change-type` override-only, `fab score`, `fab status advance`, `fab status add-issue`), MCP (Linear, optional). No `fab change switch`, no git commands.

### Sub-agents

None.

### Bookkeeping commands (hook candidates)

| Step | Command | Trigger |
|------|---------|---------|
| 6 | `fab status set-change-type` | Only if the hook-inferred type is wrong (the intake-write hook owns `change_type`) |
| 7 | `fab score --stage intake` | After intake.md write |
| 9 | `fab status advance` | After all intake work complete |

### Difference from /fab-new

`/fab-draft` omits Steps 10 and 11 from `/fab-new`:
- **No Step 10** — change is not activated (`.fab-status.yaml` symlink is not created)
- **No Step 11** — git branch is not created

The output `Next:` line uses the activation preamble: `/fab-switch {name} to make it active, then /fab-continue, /fab-fff, /fab-ff, or /fab-clarify`.
