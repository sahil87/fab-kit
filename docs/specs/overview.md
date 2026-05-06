# Fab Workflow Specification

> **Fab** (fabricate) - A Specification-Driven Development workflow

## Design Principles

### 1. Pure Prompt Play
No system installation required. All workflow logic lives in `src/kit/` as markdown templates and skill definitions that any AI agent can execute.

### 2. Memory Is the Source of Truth
Code serves documentation, not the other way around. The memory files (`docs/memory/`) are the source of truth for what the system does and why it works the way it does.

### 3. Change Folder First
All work happens in change folders. Each change captures its requirements (`spec.md`), which get hydrated into memory files on completion.

### 4. Stage Visibility
Always know where you are. Each change folder has a `.status.yaml` manifest that tracks current stage and progress. The `.fab-status.yaml` symlink at repo root points to the active change's `.status.yaml`, providing instant access to whichever change is in flight — no scanning or guessing required. Run `/fab-status` for a quick check.

### 5. Skill-Based Interface
Use skills (not rigid commands) for better agent interoperability. Skills are more naturally invocable by AI agents.

### 6. Git-Optional
Fab tracks changes in directories, not branches. A change folder is the unit of identity — the same change can be worked on across multiple branches, worktrees, or even repos. When git is available, `/fab-switch` offers to create or adopt a branch, but no branch information is stored in `.status.yaml`. Commits, pushes, and PRs remain your responsibility — Fab never couples its state to git state.

---

## Getting Started

For installation and setup, see the [Quick Start in the README](../../README.md#quick-start).

**Prerequisites**: An existing project directory (git repo recommended but not required) and an AI agent that supports skill definitions (e.g., Claude Code, Cursor, Windsurf).

After bootstrapping, use `/docs-hydrate-memory` to ingest existing documentation (Notion URLs, Linear URLs, local files) into `docs/memory/`. See [Skills Reference](skills.md#docs-hydrate-memory-sources) for details.

---

## The 5 Core Stages (7 with Ship + Review-PR)

Changes progress through 5 core stages plus 2 PR-side stages (`ship`, `review-pr`):

```mermaid
flowchart TD
    subgraph planning ["Planning"]
        direction LR
        B["1 INTAKE"] --> S["2 SPEC"]
    end
    subgraph execution ["Execution"]
        direction LR
        A["3 APPLY"] --> V["4 REVIEW"]
    end
    subgraph completion ["Completion"]
        direction LR
        AR["5 HYDRATE"] --> H[/"Hydrate into memory"/]
    end

    S --> A
    V --> AR

    style planning fill:#e8f4f8,stroke:#2196F3
    style execution fill:#fff3e0,stroke:#FF9800
    style completion fill:#e8f5e9,stroke:#4CAF50
```

### Stage Details

| # | Stage | Purpose | Artifact | Includes |
|---|-------|---------|----------|----------|
| 1 | **Intake** | Intent, scope, approach | `intake.md` | Created by `/fab-new` (auto-activates) or `/fab-draft` (no activation) with adaptive SRAD-driven questioning |
| 2 | **Spec** | What's changing | `spec.md` | Clarification of ambiguities, [NEEDS CLARIFICATION] markers |
| 3 | **Apply** | Generate plan + execute | `plan.md` + code changes | Entry sub-step: write `plan.md` with `## Tasks` and `## Acceptance` sections from `spec.md`. Main sub-step: execute the unchecked tasks under `## Tasks`, run tests, mark `[x]` |
| 4 | **Review** | Validate via sub-agent | validation report | Sub-agent review with prioritized findings (must-fix / should-fix / nice-to-have); inward sub-agent inspects items under `plan.md` `## Acceptance` |
| 5 | **Hydrate** | Complete & hydrate | memory updates | Hydrate spec into memory files |

The pipeline continues with `ship` (`/git-pr` creates the PR) and `review-pr` (`/git-pr-review` triages PR feedback) for a total of 7 stages end to end.

### User Flow

For detailed visual maps of how commands connect — including shortcuts, rework paths, and the full state machine — see **[User Flow Diagrams](user-flow.md)**.

---

## Quick Reference

| Skill | Purpose | Creates |
|-------|---------|---------|
| `/fab-setup` | Bootstrap fab/ structure | `config.yaml`, `constitution.md`, `memory/`, skill symlinks (idempotent) |
| `/docs-hydrate-memory [sources...]` | Ingest external sources into docs/memory/ | Updated `docs/memory/` with indexes |
| `/fab-new` | Start change (creates intake + activates) | `intake.md`, `.status.yaml` |
| `/fab-draft` | Create change intake without activating | `intake.md`, `.status.yaml` |
| `/fab-continue [<stage>]` | Next artifact (or reset to stage) | Next stage artifact |
| `/fab-ff` | Fast-forward through hydrate (confidence-gated) | spec + apply (plan + execute) + sub-agent review + hydrate |
| `/fab-fff` | Fast-forward-further through review-pr (confidence-gated) | All artifacts through hydrate + ship + review-pr |
| `/fab-clarify` | Deepen current artifact | Refined artifact (in place) |
| `/fab-continue` → apply | Implement | Code changes |
| `/fab-continue` → review | Validate (sub-agent) | Prioritized findings report |
| `/fab-continue` → hydrate | Complete & hydrate | Updated memory |
| `/fab-archive` | Archive completed change | Folder moved to archive/ |
| `/fab-switch` | Change active change | Updated pointer file |
| `/fab-status` | Check progress | Status display |
| `/fab-discuss` | Prime agent with project context for discussion | — (read-only) |
| `fab batch new` | Create changes from backlog items | Worktree + tmux tab per item |
| `fab batch switch` | Switch to existing changes | Worktree + tmux tab per change |
| `fab batch archive` | Archive completed changes | Worktree + tmux tab per change |

---

## Example Workflow

### Standard Flow
```bash
# 1. Start new change
/fab-new Add dark mode support with system preference detection

# 2. Intake generated with clarifying questions
# (answer questions, refine if needed)

# 3. Continue to spec
/fab-continue
# → Creates spec.md with requirements for this change
# → Asks clarifying questions about ambiguities

# 4. Implement (apply auto-generates plan.md, then runs tasks)
/fab-continue
# → Writes plan.md with ## Tasks and ## Acceptance sections
# → Executes unchecked tasks, marks each [x]

# 5. Review
/fab-continue
# → Validates implementation, checks plan.md ## Acceptance items

# 6. Hydrate
/fab-continue
# → Saves learnings into docs/memory/

# 7. Archive
/fab-archive
# → Moves change folder to archive/
```

### Fast Track (small changes)
```bash
/fab-new Add loading spinner to submit button
/fab-fff
# → Fast-forwards through spec, tasks, apply, review, hydrate, ship, and PR review
```

---

## Further Reading

- [User Flow Diagrams](user-flow.md) — visual maps of the full pipeline, shortcuts, rework paths, and state machine
- [Architecture](architecture.md) — directory structure, config, conventions
- [Skills Reference](skills.md) — detailed behavior for each `/fab-*` skill
- [Templates](templates.md) — artifact formats and acceptance generation

