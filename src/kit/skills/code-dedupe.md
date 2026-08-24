---
name: code-dedupe
description: "Sweep a scoped area for duplicated utilities, cluster them by behavioral shape, and present a ranked consolidation report. Fully read-only: suggestions only, applies nothing, drafts nothing; structure/placement review is /code-reorg's."
helpers: [_srad]
---

# /code-dedupe [scope]

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

`/code-dedupe` finds code that should have been a shared utility and wasn't. It sweeps a scoped area for duplicate and near-duplicate functions, groups them into **clusters**, decomposes each cluster into a shared core plus opt-in variation layers, proposes a canonical home, and presents a ranked, evidence-backed consolidation report.

The report is the skill's **terminal output and entire effect**. The skill is fully read-only: it modifies no files, refactors nothing, creates no changes, and creates no git state. It does **not** draft intakes or decide how a cluster gets consolidated (micro change vs `/fab-new` vs ignore) — that routing is the user's per-cluster choice. Each cluster MAY carry an informational suggested-next-action line (e.g. a ready-to-paste `/fab-new consolidate {shared behavior} into {canonical home}`), which the skill never executes.

Hard boundary: **content duplication only**. Where files live and what they are called — placement, naming, folder shape — is `/code-reorg`'s scope; structural smells encountered here are reported as pointers to `/code-reorg`, never analyzed.

"Do nothing" is a first-class outcome: a clean scope yields a plain `no consolidation candidates in {scope}` close. That is a success, not a failure.

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
fab log command "code-dedupe"
```

---

## Detector Configuration

Detectors only seed Type-1/2 candidates; the agent finds behaviorally equivalent Type-3/4 clusters.

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

**Already-done guard.** Check each cluster against `docs/memory/_shared/utilities.md` (when present) and in-flight changes (`fab/changes/`): a cluster already consolidated, or with a change in flight for it, is dropped from the proposals and noted in the report — re-proposing finished or in-progress work is noise.

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

Layering avoids a lossy flat API, drives divergence ranking, and preserves each member's migration story.

**Do not cluster on name similarity alone.** Two functions named `parseConfig` in different packages may be unrelated; two named `mustTempRepo` and `newFixtureDir` may be identical. Read the bodies.

**A cluster of one is not a cluster.** Drop it.

### Step 4: Rank, Grade, and Present

Rank clusters by consolidation value: high call-site count and low divergence first, low count and high divergence last. A 12-member cluster whose members mostly need only the base layer is a trivial win; a 3-member cluster with subtle behavioral differences may not be worth doing at all.

**Base-only members count as LOW divergence**; rank layer profiles, not textual similarity.

Grade each cluster's consolidation confidence per `_srad.md` (record the per-dimension scores alongside the grade):

- Identical members, obvious home → **Certain**
- Minor divergence with a clear unified API → **Confident**
- Divergence where consolidating might drop a behavior → **Tentative**
- Cannot tell whether members are genuinely the same → **Unresolved**

A cluster that grades mostly Tentative is telling you it should not be consolidated wholesale — say so in the report rather than talking it up.

Present the sweep report and **stop**:

```
/code-dedupe — src/go (test helpers)
Detectors: jscpd (skipped — not installed)
Scanned: 104 files, 312 functions

  1. Fab-root test scaffolding — 12 members, 12 call sites, low divergence    Confidence: Confident (S:.. R:.. A:.. D:..)
     internal/{resolve,change,archive,score,...}_test.go
     → newFabRoot(t, opts...) in internal/fabtest
     Base: tempdir + fab/changes (all 12)
     Layers: +project config (2) · +change dir (3) · +active symlink (1) · +kit override (1)
     Suggested next action: /fab-new consolidate fab-root test scaffolding into internal/fabtest

  2. YAML frontmatter parsing — 3 members, 8 call sites, medium divergence    Confidence: Tentative (S:.. R:.. A:.. D:..)
     ...

## For /code-reorg (structure, not duplication)

  - internal/util/ reads as a junk drawer — run /code-reorg internal

No files were modified. Suggested next actions are informational — acting on any cluster is your call.
```

Rules:

- Structural/placement smells appear **only** in the separate `For /code-reorg` section — never as clusters.
- Suggested-next-action lines are **informational only**; the skill never executes them. A suggested `/fab-new` line SHOULD name `_shared/utilities` for the change's Affected Memory (see § Memory Home).
- Clusters dropped by the already-done guard are noted, not silently omitted.
- When no clusters are found, close with `no consolidation candidates in {scope}` — a success.
- **No `Next:` pipeline line.** The report ends the skill — a documented opt-out per `_preamble.md` § Next Steps Convention (the skill file wins, like `/fab-discuss`'s ready signal and `/code-reorg`'s report).

### Error Handling

| Condition | Behavior |
|---|---|
| `config.yaml` / `constitution.md` missing | STOP — pre-flight message |
| Scope resolves to no files | STOP — report the resolved set; suggest a correction |
| All detectors missing | Continue — sweep unaided, note it in the report |
| A detector exits non-zero | Continue — parse available output, note the exit code |
| A detector's output is unparseable | Continue — treat as no seed from that detector, note it |
| No clusters found | Report `no consolidation candidates in {scope}` — a success, not a failure |

---

## Memory Home

Consolidated utilities are recorded in **`docs/memory/_shared/utilities.md`**. The path is fixed — `_shared` is defined as "cross-cutting concerns spanning all domains," which is what a utility index is, and there is no config override for it.

Do **not** create a top-level `utilities` domain for this. Every other domain is organized by subject matter; this one would be organized by code role and cut across all of them. If it outgrows a single file, `/docs-reorg-memory` will say so and it can be promoted then.

The skill only **reads** this file (Step 3's already-done guard). It is written by hydrate when a consolidation change ships on the normal pipeline path — which is why a suggested `/fab-new` line names `_shared/utilities` for the change's Affected Memory — no special memory-writing path, no drift between what shipped and what is recorded.

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
| Read-only? | Yes — modifies no files, creates no changes, advances no stage, creates no git state |
| Idempotent? | Yes — same scope + same code ⇒ same clusters |
| Advances stage? | No |
| Modifies `.fab-status.yaml`? | No |
| Modifies git state? | No |
| Refactors code or drafts intakes? | **No** — the report is the terminal output; routing each cluster is the user's choice |
| Judges placement/naming/structure? | **No** — content duplication only; structure is pointed at `/code-reorg` |
| Outputs `Next:` line? | No — ends with the report (opt-out per `_preamble.md` § Next Steps Convention) |
