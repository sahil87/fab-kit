# Fab Kit

A structured development workflow for AI agents. You describe a change, AI plans it, implements it, reviews it, and saves what it learned — so future changes start with better knowledge. Plain markdown prompts, no SDK, no vendor lock-in. Works with Claude Code, Codex, Cursor, and Windsurf.

AI agents write code fast. The bottleneck is now your clarity: did you define the problem well enough? Fab Kit sits at that bottleneck — it forces structured thinking before implementation, grounds every session in your project's actual context, and gets cheaper to run as agents improve.

> **[Try it now](#quick-start)** | **[Understand the concepts](#why-fab-kit)** | **[Glossary](docs/specs/glossary.md)** (new to Fab terminology?)

**Contents:** [The 6 Stages](#the-6-stages) · [Prerequisites](#prerequisites) · [Quick Start](#quick-start) · [Why Fab Kit](#why-fab-kit) · [The 5 Cs](#the-5-cs-of-quality) · [Commands](#command-quick-reference) · [Stage Coverage](#stage-coverage-by-command) · [Learn More](#learn-more)

## The 6 Stages

Every change (a self-contained feature or fix with its own folder) moves through six stages:

```mermaid
flowchart TD
    subgraph planning ["Planning"]
        direction LR
        B["1 INTAKE"] --> S["2 SPEC"] --> T["3 TASKS"]
    end
    subgraph execution ["Execution"]
        direction LR
        A["4 APPLY"] --> V["5 REVIEW"]
    end
    subgraph completion ["Completion"]
        direction LR
        AR["6 HYDRATE"]
    end

    T --> A
    V --> AR

    style planning fill:#64b5f6,stroke:#1565C0,color:#1a1a1a
    style execution fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style completion fill:#81c784,stroke:#2E7D32,color:#1a1a1a
```

| # | Stage | Purpose | Artifact |
|---|-------|---------|----------|
| 1 | **Intake** | Capture intent, scope, approach | `intake.md` |
| 2 | **Spec** | Define requirements | `spec.md` |
| 3 | **Tasks** | Break into implementation checklist | `tasks.md` + `checklist.md` |
| 4 | **Apply** | Execute the tasks | Code changes |
| 5 | **Review** | Sub-agent validates against spec and [constitution](#code-quality-as-a-guardrail) (your project's architectural rules) | Prioritized findings report |
| 6 | **Hydrate** | Save learnings into project memory (`docs/memory/`) | Memory updates |

Each stage produces a persistent artifact. Interrupt anything - `/fab-continue` picks up from the last checkpoint.

Review is performed by a **sub-agent** running in a separate context - a fresh perspective that validates against both your spec and [project constitution](#code-quality-as-a-guardrail). Findings are prioritized (must-fix, should-fix, nice-to-have) and the agent triages them, looping back for automatic rework on the issues that matter most.

A change folder looks like this:

```
fab/changes/260101-abcd-add-spinner/
├── intake.md        # What you want and why
├── spec.md          # Requirements (generated)
├── tasks.md         # Implementation plan (generated)
├── checklist.md     # Progress tracking
└── .status.yaml     # Pipeline state (+ .fab-status.yaml at repo root tracks active change)
```

## Quick Start

### Prerequisites

#### Using Fab Kit

Install with [Homebrew](https://brew.sh/) (macOS and Linux):

```bash
brew tap sahil87/tap
brew install fab-kit

# Other utilities fab depends on
brew install yq jq gh direnv
```

This installs the `fab` CLI (router), `fab-kit` (workspace lifecycle), and standalone tools `wt` (worktree manager) and `idea` (backlog manager).

* After installing `gh`, authenticate with `gh auth login`.
* After installing `direnv`, add the hook [to your shell](https://direnv.net/docs/hook.html).

| Tool | Purpose |
|------|---------|
| [fab-kit](https://github.com/sahil87/fab-kit) | The `fab` CLI router, workspace lifecycle (`init`/`upgrade-repo`/`sync`), `wt`, and `idea` |
| [yq](https://github.com/mikefarah/yq) | YAML processing for status files and schemas |
| [jq](https://jqlang.github.io/jq/) | JSON processing for settings merge during sync |
| [gh](https://cli.github.com/) | GitHub CLI - used for releases and PR workflows |
| [direnv](https://direnv.net/) | Auto-loads `.envrc` to set workspace environment variables |

#### Developing Fab Kit

In addition to the above:

```bash
brew install go just
```

| Tool | Purpose |
|------|---------|
| [Go](https://go.dev/) | Required for building binaries from source (`src/go/`) |
| [just](https://just.systems/) | Task runner for build, test, and release recipes |

### 1. Install

#### New project

```bash
fab init
```

This downloads the latest release to the system cache, sets `fab_version` in `fab/project/config.yaml`, and runs `fab sync` to deploy skills — all in one step. No curl scripts or manual downloads.

**Then in your AI agent:**

```
/fab-setup    # Claude Code
$fab-setup    # Codex
```

This generates `fab/project/constitution.md` and other project configuration files. Run `fab doctor` to verify your setup.

#### Updating from a previous version

```bash
fab upgrade-repo              # upgrades to latest version
fab upgrade-repo 0.44.0       # upgrades to a specific version
```

If the upgrade reports a version mismatch, run `/fab-setup migrations` in your AI agent to apply migrations. Safe to re-run.

To re-deploy skills and scaffold structure without downloading a new release (useful when developing fab-kit itself):

```bash
fab sync
```

### 2. Your first change

Fab Kit skills are slash commands you type into an AI agent's chat, not the terminal. Open a session in your project directory:

- **Claude Code:** `claude` in terminal
- **Codex:** `codex` in terminal
- **Cursor / Windsurf:** open the project, use the chat panel

Then type the commands below in the agent's prompt:

```bash
# In your AI agent:

# Creation - creates change folder, writes intake.md, activates the change
/fab-new Add a loading spinner to the submit button

# Planning - generates spec.md (structured requirements)
/fab-continue
# Planning - generates tasks.md (implementation checklist)
/fab-continue
# Execution - implements the code, checking off tasks as it goes
/fab-continue
# Execution - reviews implementation against spec + constitution
/fab-continue
# Completion - saves learnings into docs/memory/
/fab-continue

# Completion - archives the change folder
/fab-archive
```

At any point, run `/fab-status` to see where you are.

For small changes, `/fab-ff` (fast-forward) skips intermediate planning stages - gated by a [confidence score](#structured-autonomy-not-guesswork) that ensures ambiguity is low enough for safe execution. Both `/fab-ff` and `/fab-fff` (full fast-forward) auto-loop between apply and sub-agent review, fixing issues automatically before escalating to you.

### 3. Going parallel

While AI works on one change, start another in a separate [git worktree](https://git-scm.com/docs/git-worktree) (an isolated copy of your repo):

```
/fab-new Add error toast for failed submissions
```

Each change is a self-contained folder - multiple AI sessions run in parallel without conflicts. `/fab-new` auto-activates, so you can start working immediately. Use `/fab-draft` to queue a change without switching to it. [How the assembly line works →](docs/specs/assembly-line.md)

### Troubleshooting

Run `fab doctor` to check all prerequisites (git, yq, direnv hook, etc.) and diagnose common setup issues.

- `direnv allow` doesn't work - reload your shell or run `eval "$(direnv export zsh)"`
- `/fab-setup` not recognized - re-run `fab sync` to deploy skills
- **After cloning a repo that uses Fab Kit** - run `fab sync` once. Agent skills and hooks live in `.claude/` which is gitignored by default, so each developer needs to deploy them locally.
- **A stage fails mid-way** - run `/fab-continue` to resume from the last checkpoint. All stage artifacts are persisted, so no progress is lost.
- **AI produces bad code** - the review sub-agent catches it. `/fab-ff` and `/fab-fff` auto-loop between apply and review (up to 3 cycles) before escalating to you.
- **Abandon a change** - delete the change folder, or run `/fab-archive` to move it to the archive.

## Why Fab Kit

AI coding tools give you speed but leave you to manage quality and knowledge yourself. Fab Kit gives you all four:

| [**Speed**](#parallel-by-default) | [**Knowledge**](#shared-memory-that-grows-with-your-project) | [**Quality**](#code-quality-as-a-guardrail) | [**Autonomy**](#structured-autonomy-not-guesswork) |
|:---:|:---:|:---:|:---:|
| Parallel changes - never idle | Compounds with every change | Constitution + self-correcting review | Confidence-scored - assumes or asks based on context |

### Parallel by Default

<!-- Diagram: Traditional one-at-a-time workflow vs assembly line. In the traditional approach, you and AI alternate between working and idle. In the assembly line, you create batches of changes while AI executes previous batches - both stay busy. -->
```
  ██ = working    ░░ = idle

              One at a time
              ─────────────

  You    ██░░░░░░░░██░░░░░░░░██░░░░░░░░██░░░░░░░░
  AI     ░░████████░░████████░░████████░░████████

  Create, wait, review. Create, wait, review.
  More waiting than working.


              Assembly line
              ─────────────

  You    ██████░░█████████░██░█████████░██░░░░░░░
  AI     ░░░░░░██████████░████████████░░████████░

  Create a batch, hand off, create the next batch.
  Both always working.
```

Without Fab, you describe a task, wait while AI works, review, repeat. With Fab, you batch structured changes - each in its own folder and worktree - and create the next batch while AI executes the current one.

Three properties make this work:

- **Self-contained change folders** - Each change has its own spec, tasks, and status. No shared state - parallel changes don't interfere during development.
- **Git worktree isolation** - Each change runs in its own [worktree](https://git-scm.com/docs/git-worktree). Parallel AI sessions can't step on each other.
- **Resumable pipeline** - Every stage produces a persistent artifact. Interrupt anything, resume later.

### Shared Memory That Grows With Your Project

Most AI tools give each session a private memory that disappears when the session ends. Fab saves learnings from every completed change into `docs/memory/` - a domain-organized knowledge base committed to git and shared with the entire team.

```
  ┌──────────┐    hydrate     ┌──────────────┐
  │ spec.md  │ ─────────────▶ │ docs/memory/ │
  └──────────┘                └──────┬───────┘
       ▲                             │
       │       context for next      │
       └──────── change ─────────────┘
```

This creates a self-reinforcing cycle:

- **Every change makes the next one better** - Design decisions from `spec.md` merge into memory. Future changes load those files as context, so AI starts with real knowledge of your system instead of guessing.
- **Team knowledge, not personal notes** - Memory lives in git. Every developer and every AI session reads the same source of truth. Onboarding means cloning the repo.
- **Bootstrap from existing docs** - `/docs-hydrate-memory` ingests documentation from Notion, Linear, or local files. The pipeline keeps it current from there.
- **Structured, not append-only** - Memory is organized by domain (`auth/`, `payments/`, `users/`). `/docs-reorg-memory` restructures as it grows. `/docs-hydrate-specs` updates spec files with relevant details from memory.

### Code Quality as a Guardrail

AI writes code fast. Without structure, it also skips requirements, ignores architectural conventions, and ships the first thing that works. Fab enforces quality through structure, a constitution, and self-correcting review.

```
        ┌───────────────────────────────┐
        │  fab/project/constitution.md  │
        │    MUST · SHOULD · MUST NOT   │
        └───────────────┬───────────────┘
                        │
  intake → spec → tasks → apply ⇄ review → hydrate
             ↑       ↑       ↑    ↗
             └───────┴───────┴────┘
                sub-agent review
                with prioritized
                findings
```

- **Stages that can't be skipped** - The pipeline requires intake, spec, and tasks before any code is written. The AI can't jump straight to implementation. Before code is written, the [SRAD framework](#structured-autonomy-not-guesswork) ensures planning decisions are grounded in context - not silently guessed.
- **Project constitution** - `fab/project/constitution.md` defines your architectural rules using MUST/SHOULD/MUST NOT. Every spec, task breakdown, and review checks against it - not just the change's requirements.
- **Review that fixes, not just flags** - A **sub-agent** reviews in a fresh context, returning prioritized findings. The applying agent triages by severity and loops back to the right stage:

| Review finds | Priority | Loops back to | What happens |
|-------------|----------|---------------|--------------|
| Spec mismatch, failing tests | Must-fix | → apply | Unchecks failed tasks, re-runs them |
| Missing/wrong tasks | Must-fix | → tasks | Revises tasks, re-applies |
| Requirements were wrong | Must-fix | → spec | Updates spec, regenerates tasks |
| Code quality issue | Should-fix | → apply | Addressed when clear and low-effort |
| Style suggestion | Nice-to-have | - | May be skipped |

`/fab-fff` and `/fab-ff` auto-loop between apply and review (up to 3 cycles) - each re-review uses a fresh sub-agent. `/fab-ff` falls back to interactive rework after exhausting auto-retries. A typical `/fab-fff` run uses 2-4 agent turns per stage; the sub-agent review spawns a separate context.

#### The 5 Cs of Quality

Five configuration files shape how AI works in your project. Each answers a different question:

| C | File | Question |
|---|------|----------|
| **Constitution** | `fab/project/constitution.md` | What are our non-negotiable principles? |
| **Context** | `fab/project/context.md` | What are we working with? |
| **Code Quality** | `fab/project/code-quality.md` | How should code look when we write it? |
| **Code Review** | `fab/project/code-review.md` | What should we look for when we validate? |
| **Config** | `fab/project/config.yaml` | What are the project's factual settings? |

Notice the author-vs-critic split: `code-quality.md` guides the **writing** agent during apply - coding standards, anti-patterns, test strategy. `code-review.md` guides the **reviewing** sub-agent during review - severity definitions, scope boundaries, rework budget. Different cognitive modes, different concerns, different files.

All five are optional except `constitution.md` and `config.yaml`. They live in `fab/project/`. Run `/fab-setup` to generate them from scaffolds with sensible defaults.

### Structured Autonomy, Not Guesswork

AI tools either ask too many questions or silently assume. Fab uses **SRAD** - a 4-dimension framework - to decide which to do for each decision point during planning.

**S**ignal strength · **R**eversibility · **A**gent competence · **D**isambiguation type

Each dimension scores how safe it is to assume. The scores aggregate into a confidence grade:

| Grade | What happens |
|-------|-------------|
| **Certain** | Proceeds silently - deterministic from config/codebase |
| **Confident** | Proceeds, noted in assumptions summary |
| **Tentative** | Proceeds with marker - resolvable via `/fab-clarify` |
| **Unresolved** | Blocks and asks - too ambiguous to guess |

Grades aggregate into a **confidence score** that gates `/fab-ff`. If ambiguity is too high, the pipeline refuses to run and tells you what to clarify - no silent guesswork, no unnecessary interruption. [How SRAD works →](docs/specs/srad.md)

## Command Quick Reference

> **Prefix:** Use `/fab-*` in Claude Code, `$fab-*` in Codex.

### Pipeline

| Command | Purpose |
|---------|---------|
| `/fab-new <description>` | Start a new change — creates the intake and activates it |
| `/fab-draft <description>` | Create a change intake without activating it (queue for later) |
| `/fab-continue` | Advance to the next stage (or reset to a specific stage) |
| `/fab-ff` | Fast-forward through hydrate — confidence-gated, auto-rework loop |
| `/fab-fff` | Fast-forward further through ship + PR review — same gates as ff |
| `/fab-clarify` | Refine the current artifact — resolve gaps without advancing |
| `/fab-archive` | Archive a completed change (or restore an archived one) |
| `/fab-proceed` | Context-aware orchestrator — detects state, runs setup steps, then delegates to `/fab-fff` |

### Setup & Status

| Command | Purpose |
|---------|---------|
| `/fab-setup` | Bootstrap fab/ structure, manage config/constitution, apply migrations |
| `/fab-status` | Show current change state — name, branch, stage, checklist, next command |
| `/fab-switch` | Switch active change (or list available changes) |
| `/fab-help` | Show workflow overview and command summary |
| `/fab-discuss` | Load project context for an exploratory discussion session |

### Git

| Command | Purpose |
|---------|---------|
| `/git-branch` | Create or switch to the git branch matching the active change |
| `/git-pr` | Commit, push, and create a GitHub PR |
| `/git-pr-review` | Process PR review comments — triage and fix feedback |

### Documentation

| Command | Purpose |
|---------|---------|
| `/docs-hydrate-memory [sources...]` | Ingest external docs or generate memory from codebase analysis |
| `/docs-hydrate-specs` | Detect gaps between memory and specs, propose additions |
| `/docs-reorg-memory` | Analyze memory files for themes, suggest reorganization |
| `/docs-reorg-specs` | Analyze spec files for themes, suggest reorganization |

### Multi-Agent Coordination

The operator (`/fab-operator`) is a long-running coordination layer that sits in its own tmux pane, observing and directing agents across other panes. It is optional and useful when running multiple agent sessions simultaneously.

| Command | Purpose |
|---------|---------|
| `/fab-operator` | Multi-agent coordination — monitoring, auto-answering, autopilot queues, dependency-aware spawning |

[Operator version history →](docs/specs/operator.md)

### CLI Subcommands

| Command | Purpose |
|---------|---------|
| `fab sync` | Repair symlinks, scaffold structure, deploy skills |
| `fab doctor` | Diagnose common setup issues |
| `fab fab-help` | Print workflow overview to terminal |
| `fab operator` | Launch operator in a dedicated tmux tab |
| `fab batch new` | Create worktree tabs from backlog items |
| `fab batch switch` | Open tmux tabs in worktrees for one or more changes |
| `fab batch archive` | Archive multiple completed changes in one session |

## Stage Coverage by Command

Which pipeline stages each command covers. Taller bars = more automation. Read left-to-right from most manual to most automated. Arrows show the typical manual path from idea to PR.

```mermaid
block-beta
    columns 12

    hdr_label["wt create →"]:1 header0["/fab-discuss"] header1["/fab-draft"] header2["/fab-switch"] headerN["/fab-new"] header3["/git-branch"] header4["/fab-continue"] header5["/fab-ff"] header6["/git-pr \n /git-pr-review"] header8["/fab-fff"] header9["/fab-proceed"] space:1

    space:12

    s01["context"]:1 d_ctx["project context"]:1 space:10
    s11["intake"]:1 space:1 fd_in["intake"]:1 space:1 fnew_in["intake"]:1 space:5 p_in["intake"]:1 space:1
    s02["change active"]:1 space:2 sw_act["change active"]:1 fnew_act["change active"]:1 space:1 space:4 p_sw["change active"]:1 space:1
    s03["branch name"]:1 space:4 gb_br["branch name"]:1 space:4 p_br["branch name"]:1 space:1
    s04["spec"]:1 space:5 c_stg["one stage ▾"]:1 ff_sp["spec"]:1 space:1 fff_sp["spec"]:1 p_sp["spec"]:1 space:1
    s05["tasks"]:1 space:5 c_stg2["one stage ▾"]:1 ff_ta["tasks"]:1 space:1 fff_ta["tasks"]:1 p_ta["tasks"]:1 space:1
    s06["apply"]:1 space:5 c_stg3["one stage ▾"]:1 ff_ap["apply"]:1 space:1 fff_ap["apply"]:1 p_ap["apply"]:1 space:1
    s07["review"]:1 space:5 c_stg4["one stage ▾"]:1 ff_rv["review"]:1 space:1 fff_rv["review"]:1 p_rv["review"]:1 space:1
    s08["hydrate"]:1 space:5 c_stg5["one stage"]:1 ff_hy["hydrate"]:1 space:1 fff_hy["hydrate"]:1 p_hy["hydrate"]:1 space:1
    s09["ship"]:1 space:5 space:1 space:1 gp_sh["PR raised"]:1 fff_pr["PR raised"]:1 p_pr["PR raised"]:1 space:1
    s10["review-pr"]:1 space:5 space:1 space:1 gp_rp["PR reviewed"]:1 fff_rp["PR reviewed"]:1 p_rp["PR reviewed"]:1 space:1

    %% Arrows — multiple paths from top-left to bottom-right
    d_ctx --> fd_in
    d_ctx --> fnew_in
    d_ctx --> p_in
    fd_in --> sw_act
    sw_act --> gb_br
    fnew_in --> fnew_act
    fnew_act --> gb_br
    gb_br --> ff_sp
    gb_br --> fff_sp
    ff_hy --> gp_sh

    %% Header styles
    style hdr_label fill:none,stroke:none,color:#999
    style header0 fill:#4dd0e1,stroke:#00838f,color:#1a1a1a
    style header1 fill:#ce93d8,stroke:#7B1FA2,color:#1a1a1a
    style header2 fill:#ce93d8,stroke:#7B1FA2,color:#1a1a1a
    style headerN fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style header3 fill:#b0bec5,stroke:#546e7a,color:#1a1a1a,stroke-dasharray: 5 5
    style header4 fill:#64b5f6,stroke:#1565C0,color:#1a1a1a
    style header5 fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style header6 fill:#b0bec5,stroke:#546e7a,color:#1a1a1a,stroke-dasharray: 5 5
    style header8 fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style header9 fill:#ffb74d,stroke:#E65100,color:#1a1a1a

    %% Row labels
    style s01 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s02 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s11 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s03 fill:#bdbdbd,stroke:#757575,color:#1a1a1a,stroke-dasharray: 5 5
    style s04 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s05 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s06 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s07 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s08 fill:#bdbdbd,stroke:#757575,color:#1a1a1a
    style s09 fill:#bdbdbd,stroke:#757575,color:#1a1a1a,stroke-dasharray: 5 5
    style s10 fill:#bdbdbd,stroke:#757575,color:#1a1a1a,stroke-dasharray: 5 5

    %% fab-discuss (Explore — teal)
    style d_ctx fill:#4dd0e1,stroke:#00838f,color:#1a1a1a

    %% fab-draft (Change lifecycle — purple)
    style fd_in fill:#ce93d8,stroke:#7B1FA2,color:#1a1a1a

    %% fab-switch (Change lifecycle — purple)
    style sw_act fill:#ce93d8,stroke:#7B1FA2,color:#1a1a1a

    %% fab-new (Automation — green, creates intake + activates)
    style fnew_in fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style fnew_act fill:#81c784,stroke:#2E7D32,color:#1a1a1a

    %% git-branch (Git utilities — blue-grey)
    style gb_br fill:#b0bec5,stroke:#546e7a,color:#1a1a1a,stroke-dasharray: 5 5

    %% fab-continue (Stage advance — blue)
    style c_stg fill:#64b5f6,stroke:#1565C0,color:#1a1a1a,stroke-dasharray: 5 5
    style c_stg2 fill:#64b5f6,stroke:#1565C0,color:#1a1a1a,stroke-dasharray: 5 5
    style c_stg3 fill:#64b5f6,stroke:#1565C0,color:#1a1a1a,stroke-dasharray: 5 5
    style c_stg4 fill:#64b5f6,stroke:#1565C0,color:#1a1a1a,stroke-dasharray: 5 5
    style c_stg5 fill:#64b5f6,stroke:#1565C0,color:#1a1a1a,stroke-dasharray: 5 5

    %% fab-ff (Automation — green)
    style ff_sp fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style ff_ta fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style ff_ap fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style ff_rv fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style ff_hy fill:#81c784,stroke:#2E7D32,color:#1a1a1a

    %% git-pr (Git utilities — blue-grey)
    style gp_sh fill:#b0bec5,stroke:#546e7a,color:#1a1a1a,stroke-dasharray: 5 5

    %% git-pr-review (Git utilities — blue-grey)
    style gp_rp fill:#b0bec5,stroke:#546e7a,color:#1a1a1a,stroke-dasharray: 5 5

    %% fab-fff (Automation — green)
    style fff_sp fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style fff_ta fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style fff_ap fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style fff_rv fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style fff_hy fill:#81c784,stroke:#2E7D32,color:#1a1a1a
    style fff_pr fill:#81c784,stroke:#2E7D32,color:#1a1a1a,stroke-dasharray: 5 5
    style fff_rp fill:#81c784,stroke:#2E7D32,color:#1a1a1a,stroke-dasharray: 5 5

    %% fab-proceed (Orchestrator — amber)
    style p_sw fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style p_in fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style p_br fill:#ffb74d,stroke:#E65100,color:#1a1a1a,stroke-dasharray: 5 5
    style p_sp fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style p_ta fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style p_ap fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style p_rv fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style p_hy fill:#ffb74d,stroke:#E65100,color:#1a1a1a
    style p_pr fill:#ffb74d,stroke:#E65100,color:#1a1a1a,stroke-dasharray: 5 5
    style p_rp fill:#ffb74d,stroke:#E65100,color:#1a1a1a,stroke-dasharray: 5 5
```

## Packages

Fab Kit includes standalone CLI tools — general-purpose developer workflow utilities independent of the fab pipeline. They're installed system-wide via `brew install fab-kit`. See [packages.md](docs/specs/packages.md) for details.

| Package | Purpose |
|---------|---------|
| **idea** | Per-repo idea backlog in `fab/backlog.md` - add, list, edit, complete, remove |
| **wt** | Git worktree management - create, open, list, delete worktrees with random naming |

### Development

After cloning, run Go tests:

```bash
just test               # run all tests
just build-fab-kit      # build fab + fab-kit to dist/bin/
```

## Learn More

- **[The Assembly Line](docs/specs/assembly-line.md)** - batch scripts, Gantt charts, and the full numbers behind parallel development
- **[Design & Workflow Details](docs/specs/overview.md)** - principles, detailed stage descriptions, example workflows
- **[User Flow Diagrams](docs/specs/user-flow.md)** - visual maps of the full pipeline, shortcuts, rework paths, and state machine
- **[Full Command Reference](docs/specs/skills.md)** - detailed behavior for every `/fab-*` skill
- **[SRAD Autonomy Framework](docs/specs/srad.md)** - how the pipeline handles ambiguity, confidence scoring, and autonomous execution gates
- **[Glossary](docs/specs/glossary.md)** - all Fab terminology defined
- **[Contributing](CONTRIBUTING.md)** - developing, extending, and releasing Fab Kit
