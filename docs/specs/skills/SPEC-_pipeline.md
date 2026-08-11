# _pipeline
Shared pipeline bracket for /fab-ff and /fab-fff (/fab-adopt partially consumes the rework loop + hydrate dispatch): intake gate, context loading, resumability, Steps 1–3, bounded auto-rework, exhaustion handling. Parameterized by `{driver}`, `{terminal}`, `{confidence header}`, `{max_cycles}` (default 3).
## Flow
```
Driver (fab-ff / fab-fff) reads _pipeline.md with {driver}/{terminal} bound
├─ Pre-flight: fab preflight; gate (skip if --force) fab score --check-gate --stage intake → STOP if < 3.0
├─ Resumability: skip done stages; per-stage dispatch via fab resolve-agent <stage> --alias (dispatch= absent ⇒ native Agent / present ⇒ CLI adapter)
├─ Step 1 Apply → subagent /fab-continue Apply → fab status finish intake/apply
├─ Step 2 Review → subagent /fab-continue Review (_review.md)
│  ├─ Pass: finish review → Step 3
│  └─ Fail: auto-rework loop ≤{max_cycles} (resume apply worker when reachable, fresh review each cycle); exhaustion: fab status fail review → STOP
├─ Step 3 Hydrate → subagent /fab-continue Hydrate → fab status finish hydrate
└─ {terminal} = hydrate → complete / review-pr → driver Steps 4–5
```
### Tools used
Bash only — `fab preflight`, `fab score`, `fab resolve-agent`, `fab status *`.
### Sub-agents
| Agent | Step | Purpose |
|-------|------|---------|
| /fab-continue (Apply) | 1, rework | Plan co-gen + tasks; resumed at rework when reachable |
| /fab-continue (Review) | 2, rework | Runs `_review.md`; always fresh |
| /fab-continue (Hydrate) | 3 | Memory hydration |
