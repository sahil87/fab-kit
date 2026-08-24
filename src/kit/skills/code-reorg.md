---
name: code-reorg
description: "Review source-tree structure — folder shape, file placement, naming, consolidation — and present an evidence-backed findings report. Fully read-only: suggestions only, applies nothing, drafts nothing; docs/memory and docs/specs are excluded (pointer to docs-reorg-*)."
helpers: [_srad]
---

# /code-reorg [<path>]

> Read the `_preamble` skill first (deployed to `.claude/skills/` via `fab sync`). Then follow its instructions before proceeding.

---

## Contents

- Purpose
- Pre-flight
- Arguments
- Context Loading
- Behavior
- Error Handling
- Key Properties

---

## Purpose

`/code-reorg` treats folder structure as a **prediction interface**: a structure is good when a reader can predict where a thing lives from its name and what a file contains from its path. The skill sweeps a scoped source tree for concrete **prediction failures** — junk drawers, singleton folders, sibling naming inconsistency, misplaced files evidenced by co-change or import direction — and presents a ranked report of evidence-backed proposals (moves, renames, folder merges/splits), each with rationale, blast radius, and an SRAD-graded confidence. It never re-taxonomizes for taste.

The report is the skill's **terminal output and entire effect**. The skill is fully read-only: it modifies no files, creates no changes, runs no `fab status` transition, and creates no git state. It does **not** apply moves/renames, draft intakes, or decide how a finding gets fixed (micro change vs `/fab-new` vs ignore) — that routing is the user's per-finding choice. Each proposal MAY carry an informational suggested-next-action line (e.g. `micro: rename directly and commit` or a ready-to-paste `/fab-new <description>`), which the skill never executes.

Hard boundary: **structure only** — where files live and what they are called, with no behavioral change. Content and functionality judgment is out of scope; content-duplication smells are not clustered here but reported as pointers to `/code-dedupe`.

"Do nothing" is a first-class outcome: a clean tree yields a plain `no proposals — structure predicts well` close. That is a success, not a failure.

---

## Pre-flight

1. Verify `fab/project/config.yaml` and `fab/project/constitution.md` exist
2. **If either missing, STOP**: `fab/ is not initialized. Run /fab-setup first to bootstrap the project.`

Do **not** run `fab preflight` — this skill operates without an active change and must not resolve or disturb one.

---

## Arguments

- **`[<path>]`** *(optional)* — a directory inside the repo to sweep.

Resolution:

- **Argument given** — resolve it relative to the repo root. It must exist and lie inside the repo, else STOP per § Error Handling.
- **No argument** — default to all `source_paths` from `fab/project/config.yaml`, swept as **one combined scope** per run. If `source_paths` is missing or empty, STOP naming the config key.

**Docs-tree carve-out (path-based, not content-based).** A resolved path inside `docs/memory/` or `docs/specs/` is **refused** with a pointer to `/docs-reorg-memory` / `/docs-reorg-specs` — those trees have FKF-aware sibling skills. A scope that *contains* either tree (e.g. `.` or `docs/`) is accepted, but those two subtrees are **pruned** from the swept file set — record the pruning in the Step 1 scope echo. These two trees are the only exclusions: everything else inside the scoped path is in scope regardless of file type. Markdown inside a source tree is just source, judged by the same placement/naming frames.

---

## Context Loading

Load the **always-load layer** per `_preamble.md` §1. Treat `fab/project/context.md`'s documented repo structure (when present) as the project's own stated convention frame — frame (a) in Step 3.

Do **not** load change artifacts. There is no active change at this point.

---

## Behavior

### Step 1: Resolve and Echo the Scope

Resolve the scope per § Arguments, applying the docs-tree carve-out (refusal, or pruning for a containing scope). **Echo the resolved scope and its file count before analysis** — a silently mis-resolved scope wastes the whole run — noting any pruned subtrees:

```
Scope: src/ scripts/ (combined) — 214 files
Scope: docs/ — 87 files (pruned: docs/memory/, docs/specs/)
```

### Step 2: Gather Signals

All signals are mechanical and read-only. Gather:

- **Tree shape** — depth, fan-out, singleton folders (one file per folder), oversized flat dirs, junk drawers (`utils/`, `helpers/`, `misc/` whose contents share no theme). Junk-drawer and oversized-dir criteria are judgment guidance, not numeric thresholds — name the evidence (e.g. "6 of 8 files share no import or topic relation").
- **Sibling naming** — casing, plural/singular, and stutter inconsistencies among siblings (e.g. `pane/pane_send.go` in a `pane/` package).
- **Static import-direction** — a file whose imports and importers concentrate in a different folder than its own is placement evidence. This is the deterministic complement to co-change, which only proxies coupling behaviorally. Grep the language's import/include statements; follow the repo's own import idiom.
- **Co-change coupling** — from `git log`, with the mandatory noise controls below.

**Reference density is NOT a detection signal.** What links/imports into a file (the blast-radius input) is computed lazily in Step 4, for clustered proposals only.

#### Co-change noise controls (mandatory)

Raw co-change is systematically poisoned on sweep-heavy repos, so every control applies:

| Control | Default | Why |
|---------|---------|-----|
| History window | last **12 months** | Old coupling from a since-fixed layout is stale evidence |
| Bulk-commit exclusion | commits touching **> 20 files** | Repo-wide sweeps manufacture spurious pairwise coupling |
| Rename following | `-M` | Renames must not split a file's history |
| Mandated-coupling carve-out | derived at run time from `fab/project/constitution.md` and `fab/project/code-quality.md` | Pairs whose co-change is *required* by project rules (e.g. a CLI-source ⇒ CLI-reference-doc rule) are expected coupling, not misplacement evidence — exclude them before interpreting any pair |
| Cross-layer whitelist | tests, docs, specs | These legitimately co-change with the source they cover; never propose colocating across layers against ecosystem convention |

Worked command (the pairwise aggregation is the one signal too fiddly to improvise per run):

```sh
# Pairwise co-change counts over the last 12 months, rename-following,
# commits touching >20 files excluded. Substitute the resolved scope paths.
git log --since="12 months ago" -M --format='commit %H' --name-only -- <paths> \
  | awk '/^commit /{flush(); next} NF{f[n++]=$0} END{flush()}
         function flush(){if (n>0 && n<=20) for (i=0;i<n;i++) for (j=i+1;j<n;j++) print f[i] FS f[j]; n=0}' \
  | sort | uniq -c | sort -rn
```

Interpret the result as a **fraction**, not a raw count: a pair is coupling evidence when they co-change in a large share of the commits that touch either file. What fraction suffices is judgment guidance — the mandated-coupling carve-out and cross-layer whitelist are applied *before* interpreting any pair. Filter the pair list to the resolved scope before evaluating.

**Shallow or absent git history** (shallow clone, no `.git`, too few commits): skip the co-change signal and record an explicit note in the report. Tree-shape, naming, and import-direction signals still run.

### Step 3: Evaluate Against Frames

Evaluate each candidate finding against the frames **in priority order**:

1. **(a) Project's stated conventions** — `fab/project/context.md`'s documented repo structure.
2. **(b) Ecosystem convention** — e.g. Go project layout, the language's packaging norms.
3. **(c) Internal sibling consistency** — what the neighboring files/folders already do.

Every finding MUST name the Step-2 signal or named frame violation it cites.

**Taste guard (load-bearing).** A finding with no cited signal and no named frame violation is **dropped** — "I would have organized it differently" does not qualify.

**Frame-(c) tightening.** A finding citing *only* sibling consistency additionally requires a **quantified majority** (e.g. "7 of 9 siblings use pattern X"). Sibling consistency is otherwise the loophole that re-admits taste.

### Step 4: Cluster into Proposals

Group related findings into coherent units — one proposal = "rename these 4 files for sibling consistency", not four proposals. Each proposal carries:

- **Prediction failure** — the concrete reader-confusion it fixes, with its cited signal/frame.
- **Move/rename list** — the exact `from → to` operations.
- **Blast radius** — references that break (imports, doc cross-links; compute reference density now, for these proposals only) **plus in-flight exposure**: open branches and active fab changes (`fab/changes/`) touching the affected files.
- **Confidence** — SRAD-graded per `_srad.md` (grade the proposal's safety and the strength of its evidence; record the per-dimension scores alongside the grade).

**Package-scoped elevation.** Folder merges/renames in package-scoped languages (e.g. Go package directories, where import paths change and identifier collisions become possible) carry a **mandatory elevated blast grade** — such moves are never content-neutral.

### Step 5: Present the Ranked Report

Rank proposals by **highest confidence × lowest blast radius** first. Present and **stop**:

```
/code-reorg — src/ scripts/ (214 files)

## Proposals (ranked)

  1. Rename 4 pane_* files for sibling consistency     Confidence: Confident (S:.. R:.. A:.. D:..)
     Failure: reader cannot predict `pane_send.go` holds the send helper — 7 of 9 siblings drop the stutter prefix (frame c, quantified)
     Moves: pane/pane_send.go → pane/send.go (+3 more)
     Blast radius: 2 in-package references; no doc links; no open branches or active changes touch these files
     Suggested next action: micro: rename directly and commit

  2. ...

## For /code-dedupe (content duplication, not structure)

  - internal/{a,b}/retry.go look semantically duplicated — run /code-dedupe internal

No files were modified. Suggested next actions are informational — acting on any proposal is your call.
```

Rules:

- Content-duplication smells appear **only** in the separate `For /code-dedupe` section — never as proposals.
- Suggested-next-action lines are **informational only**; the skill never executes them.
- A co-change-skipped run states the skip note in the report (Step 2).
- When no finding survives the taste guard, close with `no proposals — structure predicts well` — a success.
- **No `Next:` pipeline line.** The report ends the skill — a documented opt-out per `_preamble.md` § Next Steps Convention (the skill file wins, like `/fab-discuss`'s ready signal).

---

## Error Handling

| Condition | Action |
|-----------|--------|
| `config.yaml` / `constitution.md` missing | STOP — pre-flight message |
| Path argument does not exist or lies outside the repo | STOP — error showing the resolved attempt |
| `source_paths` missing/empty and no argument given | STOP — error naming the config key (`source_paths` in `fab/project/config.yaml`) |
| Path inside `docs/memory/` or `docs/specs/` | Refuse with the `/docs-reorg-memory` / `/docs-reorg-specs` pointer; perform no analysis |
| Shallow clone or insufficient git history | Skip the co-change signal with an explicit note in the report; other signals still run |
| No findings survive the taste guard | Report `no proposals — structure predicts well` — a success, not a failure |

---

## Key Properties

| Property | Value |
|----------|-------|
| Requires active change? | No |
| Runs preflight? | No |
| Read-only? | Yes — modifies no files, creates no changes, advances no stage, creates no git state |
| Idempotent? | Yes — same tree + same history ⇒ same findings |
| Advances stage? | No |
| Modifies `.fab-status.yaml`? | No |
| Modifies git state? | No |
| Applies moves/renames or drafts intakes? | **No** — the report is the terminal output; routing each fix is the user's choice |
| Judges code content/functionality? | **No** — structure only; content duplication is pointed at `/code-dedupe` |
| Sweeps `docs/memory/` or `docs/specs/`? | **No** — refused with a pointer to `/docs-reorg-memory` / `/docs-reorg-specs` |
| Outputs `Next:` line? | No — ends with the report (opt-out per `_preamble.md` § Next Steps Convention) |
