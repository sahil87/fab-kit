# User Flow Diagrams

> Visual maps of the Fab workflow — how commands connect and what each flow looks like in practice.

---

## 1. How Development Works Today

The stages every developer already follows — define what to build, design it, break it down, code it, review it, close it. Fab doesn't invent new stages; it gives each one a name and a place.

```mermaid
flowchart TD
    B[intake] -->|"define requirements"| S[spec]
    S -->|"break down work"| T[tasks]
    T -->|"write code"| A[apply]
    A -->|"validate"| R[review]
    R -->|"document & close"| AR[archive]

    %% Rework
    R -.->|"fix issues"| A
    R -.->|"rethink approach"| REWORK["spec / tasks"]

    %% Styles
    style B fill:#e8f4f8,stroke:#2196F3
    style S fill:#e8f4f8,stroke:#2196F3
    style T fill:#e8f4f8,stroke:#2196F3
    style A fill:#fff3e0,stroke:#FF9800
    style R fill:#fff3e0,stroke:#FF9800
    style AR fill:#e8f5e9,stroke:#4CAF50
```

---

## 2. The Same Flow, With Fab

Each transition is now a `/fab-*` command. Shortcuts (`/fab-ff`, `/fab-fff`) run the full pipeline in one invocation. `/fab-archive` is a separate housekeeping step after the pipeline completes.

```mermaid
flowchart TD
    B[intake] -->|"/fab-continue"| S[spec]
    S -->|"/fab-continue"| T[tasks]
    T -->|"/fab-continue"| A[apply]
    A -->|"/fab-continue"| R[review]
    R -->|"/fab-continue"| H[hydrate]

    %% Post-pipeline housekeeping
    H -->|"/fab-archive"| AR[archive]

    %% Shortcuts (full pipeline, from spec onward)
    S -->|"/fab-ff
    (confidence-gated, auto-rework loop)"| H
    B -->|"/fab-fff
    (full pipeline, autonomous rework)"| H

    %% Apply-review loop (sub-agent review with auto-rework)
    R -.->|"sub-agent review
    auto-rework (fab-ff, fab-fff)"| A

    %% Rework (reset to any earlier stage)
    H -.->|"Revise anytime using
    /fab-continue &lt;stage&gt;"| REWORK["spec / tasks / apply / review"]

    %% Styles
    style B fill:#e8f4f8,stroke:#2196F3
    style S fill:#e8f4f8,stroke:#2196F3
    style T fill:#e8f4f8,stroke:#2196F3
    style A fill:#fff3e0,stroke:#FF9800
    style R fill:#fff3e0,stroke:#FF9800
    style H fill:#e8f5e9,stroke:#4CAF50
    style AR fill:#f0f0f0,stroke:#999
```

---

## 3A. Setup & Utilities

Commands that live outside the change pipeline — run once per project or anytime.

```mermaid
flowchart LR
    subgraph setup ["Setup (once per project)"]
        INIT["/fab-init"]
        HYDRATE["/docs-hydrate-memory"]
    end

    subgraph utility ["Utility (anytime)"]
        direction TB
        STATUS["/fab-status"] ~~~ HELP["/fab-help"] ~~~ UPDATE["/fab-update"]
        BACKFILL["/docs-hydrate-specs"] ~~~ REORG_MEM["/docs-reorg-memory"] ~~~ REORG_SPEC["/docs-reorg-specs"]
        STATUS ~~~ BACKFILL
    end

    subgraph shell ["Shell Utilities"]
        direction TB
        UPGRADE["fab-upgrade.sh"] ~~~ BATCH_NEW["batch-fab-new-backlog.sh"]
        BATCH_SWITCH["batch-fab-switch-change.sh"] ~~~ BATCH_ARCHIVE["batch-fab-archive-change.sh"]
        UPGRADE ~~~ BATCH_SWITCH
    end

    setup ~~~ utility ~~~ shell

    INIT --> HYDRATE

    %% Styles
    style setup fill:#f0f0f0,stroke:#999
    style utility fill:#fce4ec,stroke:#e91e63
    style shell fill:#f0f0f0,stroke:#999
```

## 3B. Change Flow

The pipeline for a single change: creation, execution (with shortcuts), and completion. Solid arrows are the primary flow; dashed arrows are lateral/utility actions.

```mermaid
flowchart TD
    subgraph creation ["Creation (once per change)"]
        NEW["/fab-new &lt;desc&gt;"]
    end

    subgraph change_exec ["Change Execution"]
        SWITCH["/fab-switch &lt;change-id&gt;"]

        CONT_S["/fab-continue → spec"]

        subgraph execution ["Execution"]
            direction TB
            CONT_T["/fab-continue → tasks"]
            APPLY["/fab-continue → apply"]
            REVIEW["/fab-continue → review"]
            HYD["/fab-continue → hydrate"]
            CONT_T --> APPLY
        end

        FF["/fab-ff
        (confidence-gated, auto-rework loop)"]
        FFF["/fab-fff
        (full pipeline, autonomous rework)"]

        FF ~~~ FFF

        CLARIFY["/fab-clarify
        (refine any planning artifact)"]

        RESET["To revise any stage:
        /fab-continue &lt;stage&gt;"]
    end

    subgraph completion ["Completion"]
        FAB_ARCHIVE["/fab-archive
        (housekeeping)"]
    end

    %% Creation → Activation → Spec
    NEW --> SWITCH
    SWITCH --> CONT_S

    %% Shortcut alternatives
    CONT_S -.-> FF
    SWITCH -.-> FFF
    FF --> HYD
    FFF --> HYD

    %% Clarify connects to the execution block
    CLARIFY -.->|"refine, then resume"| execution

    %% Spec into execution
    CONT_S --> CONT_T

    %% Execution flow
    APPLY --> REVIEW

    %% Review outcomes (sub-agent review with prioritized findings)
    REVIEW -->|"pass"| HYD
    REVIEW -.->|"fail → auto-rework
    (sub-agent, fab-ff/fab-fff)"| APPLY
    HYD -->|"move to archive"| FAB_ARCHIVE

    %% Styles
    style creation fill:#f3e5f5,stroke:#9C27B0
    style change_exec fill:#fff3e0,stroke:#FF9800
    style execution fill:#d6eaf8,stroke:#2196F3
    style completion fill:#e8f5e9,stroke:#4CAF50
    style FAB_ARCHIVE fill:#f0f0f0,stroke:#999
    style CLARIFY fill:#fff,stroke:#999,stroke-dasharray: 5 5
    style RESET fill:#fff,stroke:#999,stroke-dasharray: 5 5
```

---

## 4. Change State Diagram

The complete state machine showing how a change progresses through all stages. Each stage can be in one of four states: `pending`, `active`, `done`, or `failed` (review only). The diagram shows normal forward flow, shortcuts, rework paths, and the commands that cause each transition.

```mermaid
stateDiagram-v2
    direction TB

    [*] --> intake: /fab-new

    intake --> spec: /fab-continue

    spec --> tasks: /fab-continue
    spec --> hydrate: /fab-ff (confidence-gated, auto-rework loop)
    intake --> hydrate: /fab-fff (full pipeline, autonomous rework)

    tasks --> apply: /fab-continue

    apply --> review: /fab-continue

    review --> hydrate: pass (all checks ✓)
    review --> apply: auto-rework (sub-agent, fab-ff/fab-fff)
    review --> earlier_stage: /fab-continue ‹stage› (manual)

    state "spec / tasks / apply" as earlier_stage

    hydrate --> [*]: /fab-archive

    note right of intake
        Created by /fab-new
        Contains: requirements,
        goals, constraints
    end note

    note right of spec
        Confidence score calculated,
        /fab-clarify to improve
    end note

    note right of apply
        Tasks run in order
        Tests after each task
        Resumable (markdown ✓)
    end note

    note right of review
        Sub-agent review:
        prioritized findings
        (must-fix / should-fix /
        nice-to-have).
        Auto-rework loop in
        fab-ff and fab-fff.
    end note

    %% Styles
    classDef planning fill:#e8f4f8,stroke:#2196F3,stroke-width:2px
    classDef execution fill:#fff3e0,stroke:#FF9800,stroke-width:2px
    classDef completion fill:#e8f5e9,stroke:#4CAF50,stroke-width:2px
    classDef input fill:#f3e5f5,stroke:#9C27B0,stroke-width:2px

    class intake input
    class spec,tasks planning
    class apply,review execution
    class hydrate completion
```
