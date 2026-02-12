# User Flow Diagrams

> Visual maps of the Fab workflow — how commands connect and what each flow looks like in practice.

---

## 1. How Development Works Today

The stages every developer already follows — define what to build, design it, break it down, code it, review it, close it. Fab doesn't invent new stages; it gives each one a name and a place.

```mermaid
flowchart TD
    B[brief] -->|"define requirements"| S[spec]
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

Each transition is now a `/fab-*` command. Shortcuts (`/fab-ff`, `/fab-fff`) let you skip ahead when the change is straightforward.

```mermaid
flowchart TD
    B[brief] -->|"/fab-continue"| S[spec]
    S -->|"/fab-continue"| T[tasks]
    T -->|"/fab-apply"| A[apply]
    A -->|"/fab-review"| R[review]
    R -->|"/fab-archive"| AR[archive]

    %% Shortcuts
    B -->|"/fab-ff"| T
    B -->|"/fab-fff"| AR

    %% Rework from review
    R -.->|"fix code"| A
    R -.->|"revise"| REWORK["spec / tasks"]

    %% Styles
    style B fill:#e8f4f8,stroke:#2196F3
    style S fill:#e8f4f8,stroke:#2196F3
    style T fill:#e8f4f8,stroke:#2196F3
    style A fill:#fff3e0,stroke:#FF9800
    style R fill:#fff3e0,stroke:#FF9800
    style AR fill:#e8f5e9,stroke:#4CAF50
```

---

## 3. Full Command Map

All `/fab-*` commands and how they tie together. Solid arrows are the primary flow; dashed arrows are lateral/utility actions.

```mermaid
flowchart TD
    subgraph setup ["Setup (once per project)"]
        INIT["/fab-init"]
        HYDRATE["/fab-hydrate"]
    end

    subgraph planning ["Planning"]
        NEW["/fab-new &lt;desc&gt;"]

        subgraph continue ["fab-continue (one stage at a time)"]
            direction TB
            CONT_S["/fab-continue → spec (from brief)"]
            CONT_T["/fab-continue → tasks (from spec)"]
            CONT_S --> CONT_T
        end

        FF["/fab-ff"]
        FFF["/fab-fff"]
        CLARIFY["/fab-clarify
        (refine any planning artifact)"]
    end

    subgraph execution ["Execution"]
        APPLY["/fab-apply"]
        REVIEW["/fab-review"]
    end

    subgraph completion ["Completion"]
        ARCHIVE["/fab-archive"]
    end

    subgraph utility ["Utility (anytime)"]
        STATUS["/fab-status"]
        SWITCH["/fab-switch"]
        HELP["/fab-help"]
        BACKFILL["/fab-backfill
        (docs → design gap detection)"]
    end

    %% Setup
    INIT --> HYDRATE
    INIT -->|"or skip hydrate"| NEW
    HYDRATE --> NEW

    %% Brief fans out to three planning paths
    NEW --> CONT_S
    NEW --> FF
    NEW --> FFF

    %% Clarify connects to the continue block
    CLARIFY -.->|"refine, then resume"| continue

    %% Into execution
    CONT_T --> APPLY
    FF --> APPLY
    FFF --> APPLY

    %% Execution
    APPLY --> REVIEW
    FFF -.->|"auto"| REVIEW

    %% Review outcomes
    REVIEW -->|"pass"| ARCHIVE
    FFF -.->|"auto"| ARCHIVE
    REVIEW -.->|"fix code"| APPLY
    REVIEW -.->|"revise"| continue

    %% Styles
    style setup fill:#f0f0f0,stroke:#999
    style planning fill:#e8f4f8,stroke:#2196F3
    style continue fill:#d6eaf8,stroke:#2196F3
    style execution fill:#fff3e0,stroke:#FF9800
    style completion fill:#e8f5e9,stroke:#4CAF50
    style utility fill:#fce4ec,stroke:#e91e63
    style BACKFILL fill:#fff,stroke:#999,stroke-dasharray: 5 5
    style CLARIFY fill:#fff,stroke:#999,stroke-dasharray: 5 5
```

---

## 4. Change State Diagram (ROUGH - NOT FINAL)

The complete state machine showing how a change progresses through all stages. Each stage can be in one of four states: `pending`, `active`, `done`, or `failed` (review only). The diagram shows normal forward flow, shortcuts, rework paths, and the commands that cause each transition.

```mermaid
stateDiagram-v2
    [*] --> brief: /fab-new <description>

    brief --> spec: /fab-continue
    brief --> tasks: /fab-ff (skip spec)
    brief --> archive: /fab-fff (full pipeline)

    spec --> spec: /fab-clarify (refine)
    spec --> tasks: /fab-continue

    tasks --> tasks: /fab-clarify (refine)
    tasks --> apply: /fab-apply

    apply --> review: /fab-review

    review --> archive: pass (all checks ✓)
    review --> apply: fail → fix code
    review --> tasks: fail → revise tasks
    review --> spec: fail → revise spec

    archive --> [*]: /fab-archive (hydrate & complete)

    note right of brief
        First pipeline stage
        Contains: requirements,
        goals, constraints
        Created by /fab-new
        State: active → done
    end note

    note right of spec
        Stage states:
        • pending (not started)
        • active (in progress)
        • done (complete)

        Commands:
        • /fab-continue (advance)
        • /fab-clarify (refine)
    end note

    note right of tasks
        Stage states:
        • pending
        • active
        • done

        Checklist auto-generated
        when tasks complete
    end note

    note right of apply
        Execution phase
        • Tasks run in order
        • Tests after each task
        • Resumable (markdown ✓)
    end note

    note right of review
        Stage states:
        • pending
        • active
        • done
        • failed (triggers rework)

        Validates: tasks ✓,
        checklist ✓, tests ✓,
        spec match ✓
    end note

    note right of archive
        Completion phase
        • Hydrates to fab/docs/
        • Moves to archive/
        • Clears fab/current
    end note

    %% Styles
    classDef planning fill:#e8f4f8,stroke:#2196F3,stroke-width:2px
    classDef execution fill:#fff3e0,stroke:#FF9800,stroke-width:2px
    classDef completion fill:#e8f5e9,stroke:#4CAF50,stroke-width:2px
    classDef input fill:#f3e5f5,stroke:#9C27B0,stroke-width:2px

    class brief input
    class spec,tasks planning
    class apply,review execution
    class archive completion
```
