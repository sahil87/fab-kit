# fab-dedupe

## Summary

Sweeps a scoped area of the codebase for duplicated and near-duplicated functions, clusters them by **behavioral shape**, decomposes each cluster into a **shared core plus opt-in variation layers**, proposes a canonical home, and drafts one change intake per accepted cluster group. Read-only through the sweep and report; writes only when the user accepts clusters.

The skill is a **thin call-site** over the shared `_intake` Create-Intake Procedure (Steps 0–9, `{questioning-mode} = interactive`), stopping at intake `ready` — no activation, no git branch. Structurally it is `/fab-draft` with a cluster-analysis front end and a fan-out tail (N intakes rather than one).

**It does not refactor.** The consolidation itself runs through the normal pipeline (`/fab-fff` or `/fab-continue`) with review in the loop. This is the central design decision: merging two 70%-overlapping helpers is design work with semantic risk, and bypassing intake/review would discard the SRAD gate, `code-quality.md`, and the reviewer — all of which exist for exactly this class of change.

## Why an intake, not a bespoke artifact

The intake template's sections map onto cluster data without distortion (`## What Changes` ← canonical home + layered API; `## Impact` ← call sites; `## Origin` ← scope + detector commands as falsifiable evidence). The decisive fit is `## Assumptions`: the SRAD table is a per-cluster risk record, and `fab score`'s flat 3.0 intake gate already refuses to auto-run a change whose divergences grade Tentative. A custom report format would reinvent that gate.

## Clusters are layered, not flat

The central correction from the `src/go` dry run (2026-07-28). A cluster is **not** one behavior with one signature — it is a base every member needs plus opt-in layers only some members need:

| Layer | Required by |
|---|---|
| tempdir + `fab/changes` | all members |
| + `fab/project/config.yaml` | change, score |
| + a change dir with `.status.yaml` | archive, score, resolve |
| + `.fab-status.yaml` symlink | archive |
| + kit override with templates | change |

The honest unified API is therefore `newFabRoot(t, opts...)` with functional options — not a flat signature folding every member's needs into one parameter list. Notably `resolve_test.go` had **already** factored change-creation into a separate `createChange` helper: the right decomposition, arrived at independently.

Two consequences bind the skill:

1. **Step 3 records base + named layers + the unified API expressing them** (functional options, options struct, base + wrappers, or an explicit "keep these separate"), never a single "proposed signature" field.
2. **Step 4 ranks on the layer profile, not on textual similarity** — a member that needs only the base layer counts as **LOW** divergence even when its body reads differently. Ranking on how similar bodies look would systematically mis-rank the clusters most worth consolidating.

## Detectors seed; the agent clusters

Detector tools find **Type-1/2** clones (literal / renamed). The clusters worth consolidating are typically **Type-3/4** — same behavior, different names, different bodies. Verified 2026-07-28: **no shipping CLI performs true Type-4 detection**; every such claim resolves to an unmaintained research artifact or a Type-3 engine with LLM explanation attached. The agent's clustering is therefore the load-bearing step, and detector output is explicitly a hint about where to look.

Concretely: the four `setup*Fixture` helpers above share a common run of roughly four lines, split by divergent code — a token-based detector reports nothing on them, yet they are the target case.

**Configuration** is `consolidate.detectors` in project config: a list of command templates with `{paths}` / `{out}` placeholders, defaulting to jscpd alone. Each is probed with `command -v` and **skipped silently when absent**, mirroring `_preamble.md` § Run-Kit Reference's fail-silent discipline. A repo with zero detectors installed still works.

**Non-zero detector exit is not a STOP** — duplication tools conventionally exit non-zero to signal "threshold exceeded," which is a finding. This is an explicit per-skill exception to `_preamble.md`'s failure rule, which governs `fab` commands rather than third-party tools.

### Language coverage (verified 2026-07-28)

| Language | Detector | Clone type | Constraint |
|---|---|---|---|
| any | jscpd | Type-1/2 | Default. 223 formats; v5 is a Rust rewrite (no Node runtime); JSON + SARIF + an `ai` reporter for LLM pipelines |
| TypeScript | similarity-ts | **Type-3** | Strongest available Type-3 tool (TSED tree-edit-distance); **no JSON output** — text parsing required |
| Go | golangci-lint `dupl` | Type-1/2 | JSON via golangci-lint; bare `dupl` emits none |
| Rust | cargo-dupes | Type-2/3 | `--format json` |
| Python | pylint R0801 | Type-1/2 | Ruff has declined a duplication rule |
| Ruby | flay | Type-2/3 | AST-based |
| any | qlty | Type-2 | 40+ languages; alternative to jscpd |

TypeScript and Go are the priority targets. TS has the better story — `similarity-ts` is production-grade Type-3. On Go the best available is Type-1/2, so the agent carries proportionally more of the clustering load, which is worth knowing when judging output quality per language.

## Scope semantics

`[scope]` filters *where clusters may be found*, not *where their members may live*. A cluster seeded inside the scope may include members outside it, and those members MUST be reported — consolidating half a cluster is worse than not consolidating it.

Natural-language scopes resolve against `source_paths` / `test_paths` and are confirmed with the user before sweeping. A bare invocation defaults to `source_paths` with a warning about list length on large repos.

## Cluster acceptance is conversational

Step 4 asks which clusters to act on as a **plain conversational question** with the reply grammar `all` / `1,3` / `none` — not a structured multi-select. Decided at intake: it matches how every other fab skill interacts, sets no new precedent, and handles an arbitrary cluster count (a structured picker caps at four options). Accepting nothing is a valid, complete outcome.

## Memory home

Consolidated utilities are recorded in `docs/memory/_shared/utilities.md`. The path is **hardcoded in the skill prose — there is no config override.** `_shared` is the correct domain: it is defined as cross-cutting concerns spanning all domains, which is what a utility index is. A `consolidate.memory_file` key was considered and deliberately not shipped — it would double the registry surface for a knob with no known consumer, and adding a registry key later is cheap and non-breaking.

A top-level `utilities` domain is **deliberately rejected**: every existing domain is organized by subject matter, whereas this would be organized by code role and would cut across all of them. Promotion is deferred to `/docs-reorg-memory` if the single file outgrows itself.

The skill never writes this file. It appears in each intake's `## Affected Memory`, and **hydrate writes it** on the normal pipeline path — so the record cannot drift from what actually shipped. The file carries an honest coverage header (swept areas + date, unswept areas named), because a silently under-covering index is worse than no index: consumers trust it identically at 10% and 90% coverage.

## Flow

```
User invokes /fab-dedupe [scope]
│
├─ Pre-flight: config.yaml + constitution.md exist (NOT fab preflight — no active change)
├─ Read: _preamble.md (always-load layer: 7 project files)
├─ Read: docs/memory/_shared/utilities.md (if present — avoids re-proposing finished work)
├─ Bash: fab log command "fab-dedupe"        (no change ID — none active)
│
├─ Step 1: Resolve scope → concrete paths; echo resolved set + file count
├─ Step 2: Probe + run configured detectors (fail-silent on missing; non-zero exit tolerated)
├─ Step 3: Cluster by behavioral shape (seeded by detectors, NOT limited to them)
│          per cluster: members, shared behavior, divergences, canonical home,
│          SHARED CORE + VARIATION LAYERS + unified API, call-site count,
│          out-of-scope members
├─ Step 4: Rank (call-sites ↑ / layer-divergence ↓; base-only member = LOW) →
│          present report → ASK conversationally (all / 1,3 / none)
│
└─ Step 5: per accepted cluster group →
   Read .claude/skills/_intake/SKILL.md
   Create-Intake Procedure Steps 0–9, {questioning-mode} = interactive
   (NL input → fresh change each run; Step 2 gap analysis skips already-consolidated
    or in-flight clusters; cluster data populates the template — What Changes carries
    the layered API and per-member migration notes; one SRAD row per divergence
    + one for the canonical-home choice)
   │
   └─ STOP after Step 9 (no activation, no git branch;
      .fab-status.yaml symlink is NOT created)
```

### Tools used

Read (memory, source files), Bash (`fab log command`, `command -v` probes, detector commands, then the Create-Intake Procedure's own `fab change new` / `fab status set-change-type` / `fab score` / `fab status advance`), Write (`intake.md` via the shared procedure).

Cluster acceptance at Step 4 is a conversational reply, not an AskUserQuestion call. SRAD questions at Step 8 belong to the shared procedure.

No `fab preflight`. No `fab change switch`. No git commands.

### Sub-agents

None. The sweep runs in the invoking session so the user can steer scope and cluster acceptance interactively.

### Bookkeeping commands (hook candidates)

`fab log command "fab-dedupe"` at entry (no change ID). All other bookkeeping belongs to the shared procedure — see `SPEC-_intake.md` (Step 6 `fab status set-change-type` override-only, Step 7 `fab score --stage intake`, Step 9 `fab status advance`), executed once per drafted intake.

## Configuration surface

One registry key, `consolidate.detectors` (`src/go/fab/internal/configref/configref.go`): `Default: nil`, `Scope: project`, `Advertise: true`, rendered into the `fab config upgrade` managed fence and `fab config reference`. Its top-level key `consolidate` is registered in `internal/configscope`'s single-source scope taxonomy, which the registry lint cross-checks. See [config](../config.md).

## Difference from /fab-draft

Both stop at intake `ready` with no activation and no branch. The differences:

| | `/fab-draft` | `/fab-dedupe` |
|---|---|---|
| Input | a description | a scope |
| Front end | none | detector sweep + behavioral clustering + ranked report |
| Intakes produced | exactly 1 | 0..N (0 is a valid outcome) |
| Read-only phase | none | everything before Step 5 |
| Content source | user's description + conversation | cluster analysis (members, layers, divergences, call sites) |

## Re-run contract

The sweep is idempotent — same scope, same code, same clusters. Drafting is not: cluster descriptions are natural-language input, which per the shared procedure's Step 3 creates a fresh change on every run (no dedup). The guard against duplicate work is **Step 2's gap analysis**, which checks `docs/memory/_shared/utilities.md` and in-flight changes per cluster and skips those already handled.

## Known limitations

- **Type-4 is out of reach for detectors** — the agent is the only Type-3/4 mechanism, so recall depends on how much code it can read. Large scopes degrade recall; narrow scopes are the mitigation, and the reason scoping is a first-class argument.
- **Go's detector tier is Type-1/2 only**, so Go sweeps lean harder on the agent than TypeScript sweeps.
- **`similarity-ts` has no machine-readable output** — text parsing is required, and its format is not contractually stable.
- **Intra-repo only.** Duplication across separate repos is out of scope and is a materially harder problem.
