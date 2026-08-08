---
name: fab-dedupe
description: "Sweep a scoped area for duplicated utilities, cluster them by behavioral shape, and draft one intake per accepted cluster group. Read-only until you approve."
helpers: [_generation, _srad, _intake]
---

# /fab-dedupe [scope]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Pre-flight
- Arguments
- Context Loading
- Command Logging
- Detector Configuration
- Behavior
- Memory Home
- Key Properties

## Purpose

`/fab-dedupe` finds code that should have been a shared utility and wasn't. It sweeps a scoped area for duplicate and near-duplicate functions, groups them into **clusters**, decomposes each cluster into a shared core plus opt-in variation layers, proposes a canonical home, and — for the clusters you accept — drafts a change intake per cluster group.

The skill **does not refactor**. It produces intakes; `/fab-fff` (or `/fab-continue`) does the work with review in the loop. This is deliberate: collapsing two 70%-overlapping helpers into one is design work with real semantic risk, and the pipeline already has a gated stage for that.

**Why an intake and not a bespoke report**: the intake's `## Assumptions` SRAD table is the natural per-cluster risk record ("these 12 helpers are identical" = Certain; "these two differ in error handling" = Tentative), and `fab score`'s intake gate already refuses to auto-run a change full of Tentatives. A custom artifact would reinvent that machinery.

**Language-agnostic by construction.** Detectors are configured commands, probed at run time and skipped silently when absent. A repo with no detectors installed still works — the agent sweeps unaided, just with a smaller seed.

---

## Pre-flight

1. Verify `fab/project/config.yaml` and `fab/project/constitution.md` exist
2. **If either missing, STOP**: `fab/ is not initialized. Run /fab-setup first to bootstrap the project.`

Do **not** run `fab preflight` — this skill operates without an active change and must not resolve or disturb one.

---

## Arguments

- **`[scope]`** *(optional)* — natural-language description of what to sweep: `"test setup helpers in src/go"`, `"src/components"`, `"the API client layer"`.

Resolve the scope to a concrete path set:

- A path or glob that exists → use it directly
- Natural language → map to paths using `source_paths` / `test_paths` from `config.yaml`, then confirm the resolved set with the user before sweeping
- No argument → default to `source_paths`, and **warn** that a full-repo sweep on a large codebase produces a long cluster list; suggest narrowing

Scope is a filter on *where clusters may be found*, not on where their members may live: a cluster seeded inside the scope MAY include members outside it, and those members MUST be reported (see Step 3). Consolidating half a cluster is worse than not consolidating it.

---

## Context Loading

Load the **always-load layer** per `_preamble.md` §1.

Additionally read `docs/memory/_shared/utilities.md` if it exists (see § Memory Home) — it records what has already been consolidated, so the sweep does not re-propose finished work.

Do **not** load change artifacts. There is no active change at this point.

---

## Command Logging

After context loading:

```bash
fab log command "fab-dedupe"
```

---

## Detector Configuration

Detectors **seed** the sweep; they do not perform it. Every shipping duplicate-detection tool finds Type-1/2 clones (literal or renamed); the clusters worth consolidating are usually Type-3/4 — same behavior, different names, different bodies — which is the agent's job.

Read `consolidate.detectors` from `fab/project/config.yaml`. Absent → use the default list below.

```yaml
consolidate:
  detectors:
    - jscpd --reporters json --output {out} {paths}
```

Placeholders substituted before execution:

| Placeholder | Value |
|---|---|
| `{paths}` | resolved scope paths, each **shell-quoted**, joined by single spaces |
| `{out}` | a scratch directory the skill creates for this run, **shell-quoted** |

**Quote both before substituting.** Substitute shell-quoted values — single-quote each path (`'…'`, with an embedded `'` written `'\''`) — so a path containing a space, `$`, `;`, or any other metacharacter reaches the detector as one intact argument instead of splitting the command or executing part of it. The values come from scope resolution and project config, not a trusted allowlist, so quoting is required, not defensive.

**Probe before running, fail silent.** For each configured detector, check the binary with `command -v <bin> >/dev/null 2>&1` before invoking it. Missing → skip it silently and continue. This mirrors the `rk` discipline in `_preamble.md` § Run-Kit Reference: never error, never warn, on an absent optional tool. Report which detectors ran in the sweep report (§ Step 4) so a skipped one is visible as information, not as failure.

**A detector that runs but exits non-zero is not a STOP.** Duplication tools routinely exit non-zero to signal "threshold exceeded" — that is a finding, not an error. Parse whatever output exists, note the exit code in the report, and continue. This is an explicit per-skill exception to `_preamble.md` § Common fab Commands' failure rule, which governs `fab` commands, not third-party tools.

### Suggested per-language detectors

Not defaults — add to project config where the repo warrants it. Verified 2026-07-28.

| Language | Detector | Clone type | Notes |
|---|---|---|---|
| any | `jscpd` | Type-1/2 | 223 formats, single binary (v5 is a Rust rewrite — no Node runtime), JSON + SARIF + an `ai` reporter for LLM pipelines. The universal default. |
| TypeScript | `similarity-ts` | **Type-3** | Tree-edit-distance (TSED). The strongest available Type-3 tool. **No JSON output** — parse `path:line` text. |
| Go | `golangci-lint run --enable-only dupl --out-format json` | Type-1/2 | JSON via golangci-lint; bare `dupl` has none. |
| Rust | `cargo-dupes --format json` | Type-2/3 | AST normalization + Dice coefficient. |
| Python | `pylint --disable=all --enable=R0801` | Type-1/2 | Ruff has no duplication rule and has declined the feature. |
| Ruby | `flay` | Type-2/3 | AST-based. |
| any | `qlty smells --json` | Type-2 | 40+ languages, AST-node-kind hashing. Alternative to jscpd if already installed. |

Two things worth knowing before you trust detector output: **no shipping CLI does true Type-4** (semantically equivalent, different implementation) — claims to the contrary resolve to unmaintained research artifacts; and on Go the best available is Type-1/2, so the agent carries proportionally more of the clustering load there than on TypeScript.

---

## Behavior

### Step 1: Resolve Scope

Resolve `[scope]` to concrete paths per § Arguments. Echo the resolved set and the file count before proceeding — a scope that silently resolved to the wrong subtree wastes the whole sweep.

### Step 2: Run Detectors

For each configured detector: probe, substitute placeholders, run, collect output. Record for each — ran / skipped-missing / ran-with-exit-code-N.

Detector output is a **seed set**: pairs or groups of files and line ranges that share literal or token-level similarity. Treat it as a hint about where to look, never as the cluster list.

### Step 3: Cluster by Behavioral Shape

This is the skill's core, and the part no tool does.

Read the functions in scope — seeded by detector output but **not limited to it** — and group them into clusters where members solve the same problem, regardless of name, signature, or implementation. A cluster of `setupScoreFixture` / `setupChangeFixture` / `setupArchiveFixture` that share no token runs is exactly the target case, and no detector will hand it to you.

For each cluster record:

- **Members** — `file:line` per member, with the current name
- **Shared behavior** — one line: the problem all members solve
- **Divergences** — where members genuinely differ; these are the semantics a unified helper must preserve or deliberately drop. This field carries the risk.
- **Proposed canonical home** — an existing shared location if one fits, otherwise a new one, following the repo's own layout conventions (read neighboring directories; do not impose a convention from another project)
- **Shared core + variation layers** — the layered decomposition, below. **Not** a single flat signature.
- **Call-site count** — how many places change
- **Members outside scope** — flagged explicitly (see § Arguments)

#### Layered decomposition — do not record one flat signature

Real clusters are **layered**: a base behavior every member needs, plus opt-in layers that only some members need. Record it that way.

1. **Base** — the behavior every member of the cluster needs, with the full member list.
2. **Layers** — each opt-in behavior on top of the base, named, with exactly which members require it.
3. **Unified API** — the shape that expresses base + layers: functional options, an options struct, a base helper plus thin wrappers, or an explicit *"these two should stay separate"* when no single API is honest.

Worked example (the `src/go` sweep that motivated this rule):

| Layer | Required by |
|---|---|
| tempdir + `fab/changes` | all members |
| + `fab/project/config.yaml` | change, score |
| + a change dir with `.status.yaml` | archive, score, resolve |
| + `.fab-status.yaml` symlink | archive |
| + kit override with templates | change |

→ `newFabRoot(t, opts...)` with functional options — **not** one flat `newFixtureRepo(t, ...)` signature covering every member's needs.

**Why this matters, not just how**: forcing one signature per cluster produces a bad API — every member's specifics get folded into one parameter list, and members that need only the base look artificially divergent. The layered form also feeds Step 4's ranking (below) and gives the drafted intake a per-member migration story instead of a single lossy signature.

**Do not cluster on name similarity alone.** Two functions named `parseConfig` in different packages may be unrelated; two named `mustTempRepo` and `newFixtureDir` may be identical. Read the bodies.

**A cluster of one is not a cluster.** Drop it.

### Step 4: Rank and Present

Rank clusters by consolidation value: high call-site count and low divergence first, low count and high divergence last. A 12-member cluster whose members mostly need only the base layer is a trivial win; a 3-member cluster with subtle behavioral differences may not be worth doing at all.

**A member that needs only the base layer counts as LOW divergence** — even when its body looks different from the others'. Textual difference is not behavioral divergence; what matters is how many layers a member pulls in. Rank on the layer profile, not on how similar the bodies read.

Present the sweep report — read-only, nothing has been written:

```
Consolidation sweep — src/go (test helpers)
Detectors: jscpd (skipped — not installed)
Scanned: 104 files, 312 functions

  1. Fab-root test scaffolding — 12 members, 12 call sites, low divergence
     internal/{resolve,change,archive,score,...}_test.go
     → newFabRoot(t, opts...) in internal/fabtest
     Base: tempdir + fab/changes (all 12)
     Layers: +project config (2) · +change dir (3) · +active symlink (1) · +kit override (1)

  2. YAML frontmatter parsing — 3 members, 8 call sites, medium divergence
     ...

  2 clusters. Which should become changes? (all / 1,3 / none)
```

Then **ask** which clusters to act on **as a plain conversational question** — the reply grammar is `all`, a comma-separated list of numbers (`1,3`), or `none`. Do **not** use a structured multi-select picker: a conversational reply matches how every other fab skill interacts, sets no new precedent, and handles an arbitrary number of clusters (a structured picker caps at four options).

Accepting nothing is a valid, complete outcome — the report itself is useful.

### Step 5: Draft Intakes

For each accepted cluster group, read `.claude/skills/_intake/SKILL.md` and execute the **Create-Intake Procedure** (Steps 0–9) with `{questioning-mode} = interactive`.

Group by refactor coherence, not by count: clusters sharing a canonical home belong in one change; unrelated clusters get separate changes. **Prefer one intake per cluster.** Bundling makes apply all-or-nothing and forces review to issue one verdict over several independent refactors.

Bind the procedure's inputs as follows.

**Step 0 (Parse Input)** — the description is natural language: `consolidate {cluster shared-behavior} into {canonical home}`. No backlog or Linear ID applies, so no collision check runs and each invocation creates a fresh change. Re-running a sweep therefore creates *new* changes rather than resuming — see § Key Properties.

**Step 2 (Gap Analysis)** — check `docs/memory/_shared/utilities.md` and open changes for this cluster. Already consolidated, or a change in flight for it → report and skip that cluster; do not create a duplicate.

**Step 5 (Generate `intake.md`)** — the cluster data populates the template:

| Intake section | Content |
|---|---|
| `## Origin` | The scope argument, the detectors that ran, and their exit codes. Falsifiable evidence: a reader can re-run the sweep. |
| `## Why` | The duplication itself — member count, call-site count, and the cost of leaving it (every new site copies a member again). |
| `## What Changes` | The canonical home, the **layered API** (base + layers, as recorded in Step 3), and per-member migration notes — which layers each member needs. Concrete: the template says do not summarize or abstract. |
| `## Impact` | Every call site, and every member outside the swept scope. |
| `## Affected Memory` | `_shared/utilities` — `(new)` on first consolidation, `(modify)` after. |
| `## Open Questions` | Divergences the sweep could not resolve. |
| `## Assumptions` | One SRAD row per divergence, plus one for the canonical-home choice. See below. |

**SRAD grading (Step 8)** — grade honestly; the intake gate depends on it:

- Identical members, obvious home → **Certain**
- Minor divergence with a clear unified API → **Confident**
- Divergence where consolidating might drop a behavior → **Tentative**
- Cannot tell whether members are genuinely the same → **Unresolved**, and ask (the SRAD Critical Rule applies — this skill is interactive)

A cluster that grades mostly Tentative is telling you it should not be auto-consolidated. Let it fail the gate rather than talking the score up.

**STOP after the procedure's Step 9** (intake at `ready`). Do not activate the
change or create a branch; those are `/fab-new`-only steps.

### Output

Per drafted intake, report name and confidence, then apply `_preamble.md`
§ Activation Preamble for each drafted name at intake state:

```
Drafted 2 changes:
  260728-a1b2-consolidate-test-fixtures    Confidence: 4.2 / 5.0 (6 decisions)
  260728-c3d4-consolidate-frontmatter      Confidence: 3.1 / 5.0 (4 decisions)

Next: {per § Activation Preamble, once per drafted name}
```

When no clusters are accepted, end with the report and no `Next:` line — nothing was created.

### Error Handling

| Condition | Behavior |
|---|---|
| `config.yaml` / `constitution.md` missing | STOP — pre-flight message |
| Scope resolves to no files | STOP — report the resolved set; suggest a correction |
| All detectors missing | Continue — sweep unaided, note it in the report |
| A detector exits non-zero | Continue — parse available output, note the exit code |
| A detector's output is unparseable | Continue — treat as no seed from that detector, note it |
| No clusters found | Report "no consolidation candidates in {scope}" and stop — a legitimate outcome |
| User accepts no clusters | Report and stop; nothing written |
| `fab change new` fails for one cluster | Report it, continue with the remaining accepted clusters, list failures at the end |

The Create-Intake Procedure's own error conditions apply from Step 5 onward. No activation or git rows — those steps never run.

---

## Memory Home

Consolidated utilities are recorded in **`docs/memory/_shared/utilities.md`**. The path is fixed — `_shared` is defined as "cross-cutting concerns spanning all domains," which is what a utility index is, and there is no config override for it.

Do **not** create a top-level `utilities` domain for this. Every other domain is organized by subject matter; this one would be organized by code role and cut across all of them. If it outgrows a single file, `/docs-reorg-memory` will say so and it can be promoted then.

The skill does not write this file. It is listed in the intake's `## Affected Memory`, and **hydrate writes it** on the normal pipeline path — no special memory-writing path, no drift between what shipped and what is recorded.

The file SHOULD carry an honest coverage header, because a utilities index that silently under-covers is worse than none — every consumer trusts it equally whether it lists 10% or 90%:

```markdown
Coverage: swept `src/go` (test helpers) 2026-07-28 · not yet swept: `src/kit`, `scripts`
```

---

## Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | No |
| Runs preflight? | No |
| Read-only? | Until Step 5 — the sweep and report write nothing; intakes are created only for accepted clusters |
| Idempotent? | Partially — the sweep is idempotent; re-running after accepting clusters creates *new* changes (natural-language input has no dedup). Step 2's gap analysis is the guard: it skips clusters already consolidated or already in flight |
| Advances stage? | Yes — each drafted intake to `ready` |
| Modifies `.fab-status.yaml`? | No — changes are not activated |
| Modifies git state? | No |
| Refactors code? | **No** — that is the drafted change's job |
