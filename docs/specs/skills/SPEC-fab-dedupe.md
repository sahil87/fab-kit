# fab-dedupe
Sweeps a scoped area for duplicated functions, clusters them by behavioral shape (shared core + opt-in variation layers), and drafts one change intake per accepted cluster group. Read-only through the sweep and report; writes only on accepted clusters.
## Flow
```
User invokes /fab-dedupe [scope]
├─ Pre-flight: config.yaml + constitution.md exist (no fab preflight); Read: _preamble.md
├─ Bash: fab log command "fab-dedupe"
├─ Resolve scope → paths; probe + run configured detectors (fail-silent)
├─ Cluster by behavioral shape; rank → report → ASK (all / 1,3 / none)
└─ Per accepted group → Create-Intake Procedure Steps 0–9 (see SPEC-_intake.md); STOP after Step 9 (no activation)
```
### Tools used
Read (memory, source), Bash (`fab log command`, detector probes), Write (`intake.md` via the shared procedure). No `fab preflight`, no `fab change switch`, no git.
### Sub-agents
None.
