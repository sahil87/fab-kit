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
    WT[new worktree] -->|"/fab-discuss"| IDEA[idea]
    IDEA -->|"/fab-new"| B[intake]
    B -->|"/fab-continue"| S[spec]
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
    IDEA -->|"/fab-proceed"| RP

    %% Apply-review loop (sub-agent review with auto-rework)
    R -.->|"sub-agent review
    auto-rework (fab-ff, fab-fff)"| A

    %% Rework (reset to any earlier stage)
    H -.->|"Revise anytime using
    /fab-continue &lt;stage&gt;"| REWORK["spec / tasks / apply / review"]

    %% Styles
    style WT fill:#f0f0f0,stroke:#999
    style IDEA fill:#f0f0f0,stroke:#999
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

## 3. Change State Diagram

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

## 4. Per-Stage State Machine

Section 3 shows which *stage* a change is at. This section shows how each individual stage transitions between *states*. Every stage tracks its own progress as one of: `pending`, `active`, `ready`, `done` (and `failed` for review). The events that drive transitions are issued by `statusman.sh`.

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

## 5. Stage Coverage by Command

See [Stage Coverage by Command](../../README.md#stage-coverage-by-command) in the README.
