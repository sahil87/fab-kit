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
    R -->|"document learnings"| H[hydrate]
    H -->|"commit & push"| SH[ship]
    SH -->|"process feedback"| RP[review-pr]
    RP -->|"close"| AR[archive]

    %% Rework
    R -.->|"fix issues"| A
    R -.->|"rethink approach"| REWORK["spec / tasks"]

    %% Styles
    style B fill:#e8f4f8,stroke:#2196F3
    style S fill:#e8f4f8,stroke:#2196F3
    style T fill:#e8f4f8,stroke:#2196F3
    style A fill:#fff3e0,stroke:#FF9800
    style R fill:#fff3e0,stroke:#FF9800
    style H fill:#fff3e0,stroke:#FF9800
    style SH fill:#e8f5e9,stroke:#4CAF50
    style RP fill:#e8f5e9,stroke:#4CAF50
    style AR fill:#f0f0f0,stroke:#999
```

---

## 2. The Same Flow, With Fab

Each transition is now a `/fab-*` command. `/fab-ff` fast-forwards from intake through hydrate; `/fab-fff` fast-forwards further through ship and PR review. `/fab-archive` is a separate housekeeping step after the pipeline completes.

```mermaid
flowchart TD
    B[intake] -->|"/fab-continue"| S[spec]
    S -->|"/fab-continue"| T[tasks]
    T -->|"/fab-continue"| A[apply]
    A -->|"/fab-continue"| R[review]
    R -->|"/fab-continue"| H[hydrate]
    H -->|"/git-pr"| SH[ship]
    SH -->|"/git-pr-review"| RP[review-pr]

    %% Post-pipeline housekeeping
    RP -->|"/fab-archive"| AR[archive]

    %% Shortcuts
    B -->|"/fab-ff
    (fast-forward, confidence-gated)"| H
    B -->|"/fab-fff
    (fast-forward-further, confidence-gated)"| RP

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
    style H fill:#fff3e0,stroke:#FF9800
    style SH fill:#e8f5e9,stroke:#4CAF50
    style RP fill:#e8f5e9,stroke:#4CAF50
    style AR fill:#f0f0f0,stroke:#999
```

---

## 3. Change Flow

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

        subgraph shipping ["Shipping"]
            direction TB
            SHIP["/git-pr → ship"]
            RP["/git-pr-review → review-pr"]
            SHIP --> RP
        end

        FF["/fab-ff
        (fast-forward through hydrate)"]
        FFF["/fab-fff
        (fast-forward further through review-pr)"]

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

    %% Shortcut alternatives (both start from intake)
    SWITCH -.-> FF
    SWITCH -.-> FFF
    FF --> HYD
    FFF --> RP

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
    HYD --> SHIP
    RP -->|"move to archive"| FAB_ARCHIVE

    %% Styles
    style creation fill:#f3e5f5,stroke:#9C27B0
    style change_exec fill:#fff3e0,stroke:#FF9800
    style execution fill:#d6eaf8,stroke:#2196F3
    style shipping fill:#d5f5e3,stroke:#4CAF50
    style completion fill:#e8f5e9,stroke:#4CAF50
    style FAB_ARCHIVE fill:#f0f0f0,stroke:#999
    style CLARIFY fill:#fff,stroke:#999,stroke-dasharray: 5 5
    style RESET fill:#fff,stroke:#999,stroke-dasharray: 5 5
```

---

## 4. Change State Diagram

The complete state machine showing how a change progresses through all stages. Each stage can be in one of five states: `pending`, `active`, `ready`, `done`, or `failed` (review only). The diagram shows normal forward flow, shortcuts, rework paths, and the commands that cause each transition.

```mermaid
stateDiagram-v2
    direction TB

    [*] --> intake: /fab-new

    intake --> spec: /fab-continue

    spec --> tasks: /fab-continue
    intake --> hydrate: /fab-ff (fast-forward, confidence-gated)
    intake --> review_pr: /fab-fff (fast-forward-further, confidence-gated)

    tasks --> apply: /fab-continue

    apply --> review: /fab-continue

    review --> hydrate: pass (all checks ✓)
    review --> apply: auto-rework (sub-agent, fab-ff/fab-fff)
    review --> earlier_stage: /fab-continue ‹stage› (manual)

    state "spec / tasks / apply" as earlier_stage

    hydrate --> ship: /git-pr
    ship --> review_pr: /git-pr-review
    review_pr --> [*]: /fab-archive

    state "review-pr" as review_pr

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

    note right of ship
        Commit, push, create PR
    end note

    %% Styles
    classDef planning fill:#e8f4f8,stroke:#2196F3,stroke-width:2px
    classDef execution fill:#fff3e0,stroke:#FF9800,stroke-width:2px
    classDef shipping fill:#e8f5e9,stroke:#4CAF50,stroke-width:2px
    classDef input fill:#f3e5f5,stroke:#9C27B0,stroke-width:2px

    class intake input
    class spec,tasks planning
    class apply,review,hydrate execution
    class ship,review_pr shipping
```

---

## 5. Per-Stage State Machine

Section 4 shows which *stage* a change is at. This section shows how each individual stage transitions between *states*. Every stage tracks its own progress as one of: `pending`, `active`, `ready`, `done` (and `failed` for review). The events that drive transitions are issued by `statusman.sh`.

```mermaid
stateDiagram-v2
    direction LR

    [*] --> pending
    pending --> active: start
    pending --> skipped: skip ²

    active --> ready: advance
    active --> done: finish
    active --> skipped: skip ²
    active --> failed: fail ¹

    failed --> active: start ¹

    ready --> done: finish
    ready --> active: reset

    done --> active: reset
    done --> [*]

    skipped --> active: reset
    skipped --> [*]

    note right of failed
        ¹ Review stage only
    end note

    note right of skipped
        ² Cascades downstream.
           Not available for intake.
    end note

```

### Side-effects

| Event | Side-effect |
|-------|-------------|
| **finish** | If the next stage in the pipeline is `pending`, it is automatically set to `active` |
| **reset** | All downstream stages are cascaded to `pending` |
| **skip** | All downstream `pending` stages are cascaded to `skipped` |

Source of truth: [`fab/.kit/schemas/workflow.yaml`](../../fab/.kit/schemas/workflow.yaml)

---

## 6. Command Composition

How atomic commands compose into shortcuts. Each layer adds automation over the one below it.

```
fab-proceed  = git-branch + fab-fff  (on active change)
fab-proceed  = fab-new + fab-switch + git-branch + fab-fff  (after fab-discuss)
fab-fff      = fab-ff + ship + review-pr
fab-ff       = spec → tasks → apply → review → hydrate
fab-continue = advance one stage
```

```mermaid
sequenceDiagram
    actor Dev as Developer
    participant W as Workspace
    participant P as Pipeline

    Note over W: worktree · change folder · branch
    Note over P: spec · tasks · apply · review · hydrate · ship · review-pr

    rect rgb(224, 224, 224)
    Note over Dev,P: /fab-discuss — explore without a change
    Dev->>W: loads project context (config, constitution, memory, specs)
    Note right of W: read-only · no active change required
    end

    rect rgb(243, 229, 245)
    Note over Dev,P: Manual — one command per stage
    Dev->>W: /fab-new — create change + intake
    Dev->>W: /fab-switch — activate change
    Dev->>W: /git-branch — create branch
    Dev->>P: /fab-continue — spec
    Dev->>P: /fab-continue — tasks
    Dev->>P: /fab-continue — apply
    Dev->>P: /fab-continue — review
    Dev->>P: /fab-continue — hydrate
    Dev->>P: /git-pr — ship
    Dev->>P: /git-pr-review — review-pr
    Dev->>P: /fab-archive — archive
    end

    rect rgb(232, 244, 248)
    Note over Dev,P: /fab-ff — fast-forward to hydrate
    Dev->>P: spec → tasks → apply → review → hydrate
    Note right of P: confidence-gated
    end

    rect rgb(232, 245, 233)
    Note over Dev,P: /fab-fff — full pipeline through review-pr
    Dev->>P: spec → tasks → apply → review → hydrate → ship → review-pr
    Note right of P: confidence-gated
    end

    rect rgb(255, 243, 224)
    Note over Dev,P: /fab-proceed — on active change
    Dev->>W: git-branch
    Dev->>P: fab-fff (full pipeline)
    end

    rect rgb(255, 220, 180)
    Note over Dev,P: /fab-discuss + /fab-proceed — new change from scratch
    Dev->>W: /fab-discuss — load project context
    Dev->>W: fab-new + fab-switch + git-branch
    Dev->>P: fab-fff (full pipeline)
    end
```
