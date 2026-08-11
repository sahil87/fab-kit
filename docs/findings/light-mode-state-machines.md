# fab State Machines — Today vs. the Light Lane

> **Status: adopted** (2026-08-11 discussion, validated on loom's archive; implemented by
> 260811-3ol6-light-lane-inline-small-changes — the "after" views are the shipped behavior).
> Companion
> interactive page: [light-mode-state-machines.html](light-mode-state-machines.html) (same
> diagrams with zoom/pan; serve locally to view). Sources of truth for the "today" views:
> `src/go/fab/internal/status/status.go` (transitions, `AllowedStates`),
> `docs/memory/pipeline/change-lifecycle.md`, `src/kit/skills/_pipeline.md`.

**The adopted design changes neither state machine.** The stage graph (Machine 1) and the per-stage
state vocabulary/transitions (Machine 2) are byte-identical before and after. What changes is the
**execution locus** — who runs each stage's behavior — routed by a single **plan-size check** that
rides an existing seam: `plan.md` is co-generated inline at apply entry in *both* lanes, and the
resulting task count decides the lane. One fork, decided once, no mid-run mode machinery (no
promotion valve). Validated against loom's archive: **21% of 972 completed changes** (195/927
measurable; 137 with PRs) would have run light end-to-end; only ~7% of light runs would see any
rework, handled inline under the same cycle budget.

---

## Machine 1 — the stage pipeline (today)

Every post-intake stage runs in a dispatched worker (native subagent / tmux pane / headless CLI).

```mermaid
flowchart TD
  NEW(["/fab-new"]):::external
  INTAKE["intake — sole confidence gate · MAIN SESSION"]:::session
  APPLY["apply — plan.md co-gen + tasks · WORKER"]:::dispatched
  REVIEW["review — fresh critic · WORKER"]:::dispatched
  HYDRATE["hydrate — memory writer · WORKER"]:::dispatched
  SHIP["ship — /git-pr · SUBAGENT"]:::dispatched
  RPR["review-pr — /git-pr-review · SUBAGENT"]:::dispatched
  PARKED["review: failed — parked at exhaustion"]:::parked
  WAIT["stays active — Copilot timeout, re-run /git-pr-review"]:::external
  ARCH(["/fab-archive — not a stage"]):::external
  ADOPT(["/fab-adopt — off-pipeline entry"]):::external

  NEW --> INTAKE
  INTAKE -->|"finish"| APPLY
  APPLY -->|"finish"| REVIEW
  REVIEW -->|"pass · finish"| HYDRATE
  REVIEW -->|"fail + reset apply, ≤ max_cycles (3)"| APPLY
  REVIEW -.->|"exhaustion · fail only"| PARKED
  HYDRATE -->|"finish"| SHIP
  SHIP -->|"finish"| RPR
  RPR -->|"finish"| ARCH
  RPR -.-> WAIT
  ADOPT -.->|"apply: skipped"| REVIEW

  classDef session fill:#e6ebf2,stroke:#64748b,color:#16283c
  classDef dispatched fill:#dcebfa,stroke:#0f4c81,color:#16283c
  classDef parked fill:#fae1e1,stroke:#b91c1c,color:#16283c
  classDef external fill:#f2f4f7,stroke:#9aa8b8,stroke-dasharray:5 4,color:#5b6b7e
```

## Machine 1 — after: combined light + full

Two fully parallel lanes, zero crossings. Review appears in each lane but is the **same**
fresh-worker dispatch in both — drawn twice for legibility, never inline. (Adopt entry and the
Copilot-timeout edge are unchanged — omitted.)

```mermaid
flowchart TD
  NEW(["/fab-new"]):::external
  INTAKE["intake — sole confidence gate ≥ 3.0 · MAIN SESSION"]:::session
  PLANGEN["plan.md co-generation — INLINE, always · apply entry, both lanes"]:::inline
  SIZE{"≤ 5 tasks? (--light / --full override)"}:::decision

  subgraph LANE_L["LIGHT lane — inline (23% of loom history)"]
    direction TB
    LEXEC["task execution — INLINE"]:::inline
    LREV["review — fresh critic WORKER (same dispatch as full)"]:::dispatched
    LHYD["hydrate — INLINE"]:::inline
    LSHIP["ship — INLINE"]:::inline
    LRPR["review-pr — INLINE · poll has no yield seam"]:::inline
  end

  subgraph LANE_F["FULL lane — delegated (today's bracket)"]
    direction TB
    FAPPLY["apply worker — plan exists, co-gen skipped"]:::dispatched
    FREV["review — fresh critic WORKER"]:::dispatched
    FHYD["hydrate — WORKER"]:::dispatched
    FSHIP["ship — /git-pr SUBAGENT"]:::dispatched
    FRPR["review-pr — /git-pr-review SUBAGENT"]:::dispatched
  end

  PARKED["review: failed — parked at exhaustion"]:::parked
  ARCH(["/fab-archive"]):::external

  NEW --> INTAKE
  INTAKE -->|"finish intake"| PLANGEN
  PLANGEN --> SIZE
  SIZE -->|"yes · LIGHT"| LEXEC
  SIZE -->|"no · FULL"| FAPPLY

  LEXEC -->|"finish apply"| LREV
  LREV -->|"pass"| LHYD
  LREV -->|"fail · rework INLINE ≤ max_cycles (3)"| LEXEC
  LREV -.->|"exhaustion"| PARKED
  LHYD -->|"finish"| LSHIP
  LSHIP -->|"finish"| LRPR
  LRPR -->|"finish"| ARCH

  FAPPLY -->|"finish apply"| FREV
  FREV -->|"pass"| FHYD
  FREV -->|"fail · ≤ max_cycles (3)"| FAPPLY
  FREV -.->|"exhaustion"| PARKED
  FHYD -->|"finish"| FSHIP
  FSHIP -->|"finish"| FRPR
  FRPR -->|"finish"| ARCH

  classDef session fill:#e6ebf2,stroke:#64748b,color:#16283c
  classDef dispatched fill:#dcebfa,stroke:#0f4c81,color:#16283c
  classDef inline fill:#fdf1d7,stroke:#b45309,color:#16283c
  classDef parked fill:#fae1e1,stroke:#b91c1c,color:#16283c
  classDef external fill:#f2f4f7,stroke:#9aa8b8,stroke-dasharray:5 4,color:#5b6b7e
  classDef decision fill:#f2f4f7,stroke:#b45309,stroke-width:2px,color:#16283c
```

### Execution locus per stage

| Stage / step | Today (all changes) | After — light lane | After — full lane |
|---|---|---|---|
| `intake` | main session | main session | main session |
| plan co-generation (apply entry) | inside the apply worker | **inline** (the lane decision reads its output) | **inline** — new for full too; the worker receives the finished plan via the plan-exists seam |
| task execution (apply) | worker (native/pane/headless) | **inline** | worker, co-gen skipped |
| `review` | fresh worker | fresh worker — **kept** (reviewer independence survives at every size) | fresh worker |
| `hydrate` | worker | **inline** | worker |
| `ship` | /git-pr subagent | **inline** | /git-pr subagent |
| `review-pr` | /git-pr-review subagent | **inline** (kills the yield-seam hazard the sync-poll directive fights) | /git-pr-review subagent |

## Machine 2 — states within a stage (UNCHANGED — zero new states or transitions)

From `internal/status/status.go` (`defaultTransitions` + review/review-pr overrides). ® = review /
review-pr only.

```mermaid
flowchart TD
  P["pending"]:::plainS
  A["active"]:::activeS
  R["ready"]:::readyS
  D["done"]:::doneS
  F["failed ®"]:::failedS
  S["skipped"]:::doneS

  P -->|"start (pre hook may block)"| A
  F -->|"start ®"| A
  A -->|"advance"| R
  A -->|"finish (+ auto-activate next pending)"| D
  R -->|"finish"| D
  R -->|"reset (cascade ↓ pending)"| A
  D -->|"reset (cascade ↓ pending)"| A
  S -->|"reset"| A
  A -->|"fail ®"| F
  P -->|"skip (cascade ↓ pending→skipped)"| S
  A -->|"skip"| S

  classDef plainS fill:#f2f4f7,stroke:#64748b,color:#16283c
  classDef activeS fill:#dcebfa,stroke:#0f4c81,color:#16283c
  classDef readyS fill:#f2f4f7,stroke:#b45309,color:#16283c
  classDef doneS fill:#d9efe9,stroke:#0f766e,color:#16283c
  classDef failedS fill:#fae1e1,stroke:#b91c1c,color:#16283c
```

Per-stage nuances from `AllowedStates`: `intake` holds only `active/ready/done`; `ship` and
`review-pr` can never hold `ready`; only `review`/`review-pr` can hold `failed`. `refresh` is not
a transition (pull-based recompute). `reset` runs from `{done, ready, skipped}`; `skip` from
`{pending, active}`.

## The lane decision — a one-time fork, not a machine

```mermaid
flowchart LR
  PG["plan.md co-generated — inline · apply entry · both lanes"]:::session
  Q{"task count ≤ 5?"}:::decision
  L["LIGHT — execute + hydrate + ship + review-pr inline; review stays a fresh worker"]:::inline
  FU["FULL — today's bracket; worker receives the pre-generated plan"]:::dispatched

  PG --> Q
  Q -->|"yes (23% of loom history)"| L
  Q -->|"no, or --full"| FU

  classDef session fill:#e6ebf2,stroke:#64748b,color:#16283c
  classDef dispatched fill:#dcebfa,stroke:#0f4c81,color:#16283c
  classDef inline fill:#fdf1d7,stroke:#b45309,color:#16283c
  classDef decision fill:#f2f4f7,stroke:#b45309,stroke-width:2px,color:#16283c
```

No promotion valve: light rework stays inline under the same `max_cycles` budget; exhaustion parks
to the manual menu identically in both lanes; a parked light run re-enters however the user
chooses, including `--full`. Safe because every diff — light or full — passes the same independent
review: a misclassified small change wastes bounded inline rework, never ships unreviewed.
A rejected earlier draft inferred the lane from change type at intake; loom's data killed it (68%
of type-eligible changes outgrow the threshold; 154 small feat/fix changes escape it).

## Loom evidence (972 archived changes, 927 with recoverable task counts)

Rework risk is a clean gradient over plan size — task count is the signal, change type is noise:

| Plan size | Share of changes | Rework rate |
|---|---|---|
| 1–3 tasks | 9% | 8% |
| 4–5 tasks | 13% | 7% |
| 6–8 tasks | 21% | 11% |
| 9–12 tasks | 22% | 16% |
| 13+ tasks | 34% | 29% |

The ≤5 cut: 210 light-eligible at plan time; **195 (21% of all) finish with zero rework**, 137
with recorded PRs. Analysis script: session scratchpad `loom_light_analysis.py` over
`fab/changes/archive/**/.status.yaml` + `.history.jsonl` review-failed events.

## Adjacent optimizations (independently adoptable)

1. **Full mode gets faster too** — inline co-gen means the dispatched apply worker starts with the
   plan in hand; its cold start does task execution only.
2. **Light rework = worker continuation for free** — the orchestrator *is* the author; the whole
   § Worker Continuation apparatus becomes a full-lane-only concern.
3. **Guard the inline co-gen worst case** — an obviously-large intake scope skips inline co-gen
   and dispatches apply-with-co-gen exactly as today.
4. **Threshold as a project knob** (e.g. `light_max_tasks`, default 5) — loom's gradient is
   smooth; repos with different change anatomy can cut bolder.
5. **Stamp the lane into history** — a `weight` event in `.history.jsonl` makes realized savings
   measurable and doubles as the resume-consistency record.
6. **Attack the last wall-clock: the Copilot poll** — requesting the Copilot review at *ship*
   (PR creation) overlaps the wait with the pipeline tail; benefits every run, independent of
   light mode.
