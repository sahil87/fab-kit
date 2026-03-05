# Intake: Gemini CLI Agent-Aware Sync

**Change**: 260303-l6nk-gemini-cli-agent-aware-sync
**Created**: 2026-03-03
**Status**: Draft

## Origin

> Add Gemini CLI support to fab-kit and implement agent-aware folder creation during sync. Two features: (1) Add Gemini CLI as a new agent target in `fab/.kit/sync/2-sync-workspace.sh` — Gemini uses `.gemini/skills/<name>/SKILL.md` format (directory-based, copies, same as Claude Code/Codex). (2) Make agent folder creation conditional on whether the agent CLI is available in PATH — detect which agents are installed (claude, opencode, codex, gemini) and only create/sync dot folders for agents that are actually present, rather than always creating all of them. This keeps workspaces clean.

Conversational mode. Discussed during a `/fab-discuss` session where we explored the sync architecture, agent deployment formats, and cross-agent skill visibility. Key decisions were made about directory formats and detection strategy before this intake was created.

## Why

1. **Gemini CLI is now a top-5 coding agent** with a generous free tier and growing adoption. Fab-kit currently supports Claude Code, OpenCode, and Codex but has no Gemini target — users on Gemini CLI get no skill deployment.

2. **Workspace pollution**: The current sync script unconditionally creates `.claude/`, `.opencode/`, `.agents/` directories and syncs skills to all three, even when only one agent is actually installed. This clutters the workspace and creates noise in `.gitignore` and version control for agents the developer doesn't use.

3. **If we don't fix it**: Gemini CLI users can't use fab-kit skills without manual setup. Developers using a single agent still get three dot-folders they don't need.

## What Changes

### 1. Gemini CLI agent target

Add a new `sync_agent_skills` call for Gemini CLI in `fab/.kit/sync/2-sync-workspace.sh`:

- **Dot folder**: `.gemini/skills/`
- **Format**: `directory` (same as Claude Code and Codex)
- **Mode**: `copy` (Gemini CLI reads skill files, not symlinks)
- **Layout**: `.gemini/skills/<name>/SKILL.md`

Also add a corresponding `clean_stale_skills` call.

### 2. Agent-aware conditional sync

Replace the unconditional agent sync block with detection logic:

```bash
# Detect which agents are available in PATH
# For each known agent, check command availability and only sync if present
```

Agent detection mapping:

| Agent | CLI command | Dot folder |
|-------|-----------|------------|
| Claude Code | `claude` | `.claude/skills/` |
| OpenCode | `opencode` | `.opencode/commands/` |
| Codex CLI | `codex` | `.agents/skills/` |
| Gemini CLI | `gemini` | `.gemini/skills/` |

When an agent is not detected:
- Skip `sync_agent_skills` for that agent
- Skip `clean_stale_skills` for that agent
- Do **not** delete existing dot folders (the agent may be temporarily unavailable)
- Print a skip message: `Skipping {agent}: not found in PATH`

When no agents are detected, warn but don't fail — the user may be setting up skills before installing an agent.

### 3. Scaffold updates

If Gemini CLI has scaffold requirements (e.g., permissions file like `.claude/settings.local.json`), add them to `fab/.kit/scaffold/.gemini/`. Research Gemini CLI's equivalent during implementation.

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Document Gemini CLI as a supported agent target and the agent detection mechanism
- `fab-workflow/distribution`: (modify) Update the agent deployment matrix with Gemini CLI entry and conditional sync behavior

## Impact

- `fab/.kit/sync/2-sync-workspace.sh` — primary implementation file
- `fab/.kit/scaffold/` — potential new scaffold entries for Gemini
- `.gemini/skills/` — new directory created during sync
- Existing agent sync behavior changes from unconditional to conditional
- No breaking changes — agents already installed continue to work identically

## Open Questions

- Does Gemini CLI require any scaffold files (equivalent to `.claude/settings.local.json`)? Needs research during implementation.
- Should GitHub Copilot (`.github/`) be added as a fifth agent target in this change, or deferred?

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Gemini uses directory-based SKILL.md format | Discussed — confirmed via Gemini CLI docs during session | S:95 R:85 A:95 D:95 |
| 2 | Certain | Agent detection uses `command -v` | Discussed — standard POSIX mechanism, already used in sync script for `jq` | S:90 R:90 A:95 D:95 |
| 3 | Certain | Gemini dot folder is `.gemini/skills/` | Discussed — confirmed via Gemini CLI docs | S:95 R:85 A:95 D:95 |
| 4 | Confident | Copy mode (not symlink) for Gemini | Gemini CLI is directory-based like Claude/Codex; symlink support unconfirmed | S:75 R:85 A:70 D:80 |
| 5 | Confident | Skip-don't-delete for missing agents | Discussed — avoids destroying user's existing dot folders if agent temporarily absent | S:80 R:70 A:80 D:85 |
| 6 | Tentative | GitHub Copilot deferred to separate change | Not discussed — scope decision; Copilot uses `.github/` which is a shared namespace | S:50 R:80 A:60 D:55 |
<!-- assumed: Copilot deferred — .github/ is a shared namespace with non-skill content, warrants separate analysis -->

6 assumptions (3 certain, 2 confident, 1 tentative).
