# Fab Workflow Specification

> **Fab** (fabricate) - A Specification-Driven Development workflow

## Overview

A hybrid SDD workflow that combines:
- **SpecKit's** intuitive structure, folder customization, and pure-prompt approach
- **OpenSpec's** fast-forward workflow and memory hydration

---

## Design Principles

### 1. Pure Prompt Play
No system installation required. All workflow logic lives in `fab/.kit/` as markdown templates and skill definitions that any AI agent can execute.

### 2. Memory Is the Source of Truth
Code serves documentation, not the other way around. The memory files (`fab/memory/`) are the source of truth for what the system does and why it works the way it does.

### 3. Change Folder First
All work happens in change folders. Each change captures its requirements (`spec.md`), which get hydrated into memory files on completion.

### 4. Stage Visibility
Always know where you are. Each change folder has a `.status.yaml` manifest that tracks current stage and progress. A `current` pointer file (`fab/current` contains the active change name) provides instant access to whichever change is in flight — no scanning or guessing required. Run `/fab-status` for a quick check.

### 5. Skill-Based Interface
Use skills (not rigid commands) for better agent interoperability. Skills are more naturally invocable by AI agents.

### 6. Git-Optional
Fab tracks changes in directories, not branches. A change folder is the unit of identity — the same change can be worked on across multiple branches, worktrees, or even repos. When git is available, `/fab-switch` offers to create or adopt a branch, but no branch information is stored in `.status.yaml`. Commits, pushes, and PRs remain your responsibility — Fab never couples its state to git state.

---

## Getting Started

### Prerequisites

- An existing project directory (git repo recommended but not required)
- An AI agent that supports skill definitions (e.g., Claude Code, Cursor, Windsurf)

### Bootstrap

Fab's workflow logic lives in `fab/.kit/` — but `.kit/` doesn't exist until you put it there. This is a deliberate two-phase setup:

**Phase 1: Obtain `.kit/`**

Copy the `.kit/` directory into your project:

```bash
mkdir -p fab
cp -r /path/to/fab-kit fab/.kit
```

> `.kit/` is distributed as a standalone directory. See [Distribution](architecture.md#distribution--bootstrapping) for sourcing options. When `.kit/` gets its own repository, this becomes a single clone or download.

**Phase 2: Run `/fab-init`**

```bash
/fab-init
```

This generates everything else: `config.yaml`, `constitution.md`, `memory/`, `changes/`, and skill symlinks. See [Skills Reference](skills.md#fabinit) for full behavior.

### Verify

Run `/fab-status` — it should show "No active change". You're ready. Start your first change with `/fab-new <description>`.

### Hydrating Memory from Existing Sources

After the initial bootstrap, use `/fab-hydrate` to ingest existing documentation into `fab/memory/`:

```bash
# Pull in API docs from Notion
/fab-hydrate https://notion.so/myteam/API-Spec-abc123

# Ingest local legacy docs
/fab-hydrate ./docs/legacy/

# Multiple sources at once
/fab-hydrate https://notion.so/myteam/Auth-xyz https://linear.app/myteam/project/payments-abc ./specs/
```

Supported sources: **Notion URLs**, **Linear URLs**, **local files/directories**. Each run analyzes the content, maps it to domains, and creates or merges into `fab/memory/`. See [Skills Reference](skills.md#fabhydrate-sources) for details.

---

## The 6 Stages

Changes progress through 6 stages:

```mermaid
flowchart TD
    subgraph planning ["Planning"]
        direction LR
        B["1 BRIEF"] --> S["2 SPEC"] --> T["3 TASKS"]
    end
    subgraph execution ["Execution"]
        direction LR
        A["4 APPLY"] --> V["5 REVIEW"]
    end
    subgraph completion ["Completion"]
        direction LR
        AR["6 ARCHIVE"] --> H[/"Hydrate into memory"/]
    end

    T --> A
    V --> AR

    style planning fill:#e8f4f8,stroke:#2196F3
    style execution fill:#fff3e0,stroke:#FF9800
    style completion fill:#e8f5e9,stroke:#4CAF50
```

### Stage Details

| # | Stage | Purpose | Artifact | Includes |
|---|-------|---------|----------|----------|
| 1 | **Brief** | Intent, scope, approach | `brief.md` | Created by `/fab-new` with adaptive SRAD-driven questioning |
| 2 | **Spec** | What's changing | `spec.md` | Clarification of ambiguities, [NEEDS CLARIFICATION] markers |
| 3 | **Tasks** | Implementation checklist | `tasks.md` | Auto-generated quality checklist (`checklist.md`) |
| 4 | **Apply** | Execute tasks | code changes | Run tests per task, progress tracking |
| 5 | **Review** | Validate against spec | validation report | Checklist completion, spec drift detection |
| 6 | **Archive** | Complete & hydrate | archive entry | Hydrate spec into memory files |

### User Flow

From the user's perspective, the main workflow is `/fab-new` followed by `/fab-continue` (or `/fab-ff` to fast-forward planning). `/fab-clarify` is available at any planning stage to deepen the current artifact before moving on:

```mermaid
flowchart TD
    NEW["/fab-new"]
    THINK["/fab-continue or ff"]
    APPLY["/fab-continue"]
    REVIEW["/fab-continue"]
    ARCHIVE["/fab-continue"]
    THINK ~~~ CLARIFY["/fab-clarify"]

    NEW -->|brief| THINK
    THINK -->|spec + tasks| APPLY
    APPLY -->|code changes| REVIEW
    REVIEW -->|passed| ARCHIVE
    REVIEW -->|fix code| APPLY
    REVIEW -.->|revise spec/tasks| THINK
    THINK <-.-> CLARIFY

    style NEW fill:#e8f4f8,stroke:#2196F3
    style THINK fill:#e8f4f8,stroke:#2196F3
    style APPLY fill:#fff3e0,stroke:#FF9800
    style REVIEW fill:#fff3e0,stroke:#FF9800
    style ARCHIVE fill:#e8f5e9,stroke:#4CAF50
    style CLARIFY fill:#fff,stroke:#999,stroke-dasharray: 5 5
```

---

## Quick Reference

| Skill | Purpose | Creates |
|-------|---------|---------|
| `/fab-init` | Bootstrap fab/ structure | `config.yaml`, `constitution.md`, `memory/`, skill symlinks (idempotent) |
| `/fab-hydrate [sources...]` | Ingest external sources into fab/memory/ | Updated `fab/memory/` with indexes |
| `/fab-new` | Start change (optionally with `--switch`) | `brief.md`, `.status.yaml` |
| `/fab-continue [<stage>]` | Next artifact (or reset to stage) | Next stage artifact |
| `/fab-ff` | Fast forward remaining planning | spec.md + tasks + checklist |
| `/fab-clarify` | Deepen current artifact | Refined artifact (in place) |
| `/fab-continue` → apply | Implement | Code changes |
| `/fab-continue` → review | Validate | Validation report |
| `/fab-continue` → archive | Complete & hydrate | Archive entry, updated memory |
| `/fab-switch` | Change active change | Updated pointer file |
| `/fab-status` | Check progress | Status display |

---

## Example Workflow

### Standard Flow
```bash
# 1. Start new change
/fab-new Add dark mode support with system preference detection

# 2. Brief generated with clarifying questions
# (answer questions, refine if needed)

# 3. Continue to spec
/fab-continue
# → Creates spec.md with requirements for this change
# → Asks clarifying questions about ambiguities

# 4. Continue to tasks
/fab-continue
# → Creates tasks.md with implementation checklist
# → Auto-generates checklist.md

# 5. Implement
/fab-continue
# → Executes tasks, marks completed

# 6. Review
/fab-continue
# → Validates implementation, checks checklist

# 7. Archive
/fab-continue
# → Hydrates memory/, moves to archive/
```

### Fast Track (small changes)
```bash
/fab-new Add loading spinner to submit button
/fab-ff
/fab-continue
/fab-continue
/fab-continue
```

---

## Further Reading

- [Architecture](architecture.md) — directory structure, config, conventions
- [Skills Reference](skills.md) — detailed behavior for each `/fab:*` skill
- [Templates](templates.md) — artifact formats and checklist generation

