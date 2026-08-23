# Intake: Add /fab-dedupe skill

**Change**: 260728-4v91-add-fab-dedupe-skill
**Created**: 2026-07-28

## Origin

> Add /fab-dedupe skill — sweeps a scoped area for duplicate utilities, clusters them by
> behavioral shape, drafts one intake per accepted cluster. Includes new consolidate config
> keys in the Go config registry.

Produced by a `/fab-discuss` session on 2026-07-28 that began as a review of
`fab/plans/sahil/26-07-22-reuse-awareness-codemap.md` (the `[ruaw]` / `[cmap]` backlog pair) and
converged on a narrower, shippable skill. The session's decision trail:

1. **Review of the existing plan** surfaced that Part 2 (`fab codemap`, a Go subsystem with
   WASM grammar plugins) is justified by a duplication rate nobody has measured, and that its
   `annotate` step depends on `dispatch_command`, which is absent by default.
2. **A correction from the user**: fab-kit is deployed into many repos (TypeScript, Go,
   Python, Rust, HTML). An earlier argument of the agent's — that Part 2 was
   "dogfooding-hostile" because *this* repo's duplication is markdown mirrors — was withdrawn
   as generalizing from N=1. The design requirement that survived is **genericity**:
   TypeScript and Go are the priority targets, but the skill must work on any repo.
3. **Tool research** (verified 2026-07-28, sources checked for maintenance status) established
   the detector landscape and, decisively, that **no shipping CLI performs true Type-4
   (semantically-equivalent) clone detection**.
4. **A dry run against `src/go`** validated the core clustering step and corrected the design
   (see § Why, "What the dry run changed").
5. **Naming** was decided by the user from four candidates (`fab-dedupe`, `fab-reuse`,
   `fab-consolidate`, `fab-extract`) → **`fab-dedupe`**: the term of art, short in a
   frequently-typed command, and the "implies deletion" objection is answered by the skill
   being read-only.

Draft files exist at `src/kit/skills/fab-dedupe.md` and
`docs/specs/skills/SPEC-fab-dedupe.md`, written during the session before the pipeline was
started. They are **working material, not the deliverable** — apply should treat them as a
starting draft to be corrected (notably the Step 3 flat-signature bug in § Why) rather than
as finished work to be preserved.

## Why

**The problem.** `fab/project/code-quality.md` names "duplicating existing utilities instead
of reusing them" as an anti-pattern, but nothing gives an agent an inventory to check
against. Duplication accumulates silently, and the existing `[ruaw]`/`[cmap]` backlog items
propose expensive machinery (a curated inventory that degrades silently; a Go subsystem with
a plugin marketplace) to prevent *future* duplication — while doing nothing about what has
already accumulated.

**The consequence of not fixing it.** Every new call site copies an existing helper again.
In this repo alone the measured state is: **~20 distinct `setupXRepo`-shaped test helpers,
~17 test files independently scaffolding a fake fab project tree, across 32 test packages and
30.5k lines of test code, with no `testutil` package**. That number only grows, and each
addition makes the eventual consolidation more expensive.

**Why this approach over the alternatives.**

- *Over `[cmap]` (`fab codemap`)*: the skill is polyglot on day one via a single universal
  detector, needs no Go subsystem, no WASM runtime, no plugin registry, and no Constitution I
  amendment. `fab codemap` requires a grammar pack per language before it produces value in
  any repo. The skill also generates the duplication baseline that would justify — or refute
  — building `codemap` later.
- *Over `[ruaw]` (curated inventory)*: `[ruaw]`'s inventory has a silent-degradation failure
  (no consumer can tell whether it covers 10% or 90% of the codebase) and a cold-start
  problem (an inventory built over an unconsolidated codebase is an index of the mess). An
  inventory produced *as the output of a real consolidation pass* is honest by construction.
- *Over a skill that refactors directly*: collapsing two 70%-overlapping helpers is design
  work with real semantic risk. The pipeline already has a gated stage for that. A skill with
  its own refactor loop would bypass the SRAD gate, `code-quality.md`, and the reviewer —
  which exist for exactly this class of change.

**Why an intake is the right output artifact.** The intake template's sections map onto
cluster data without distortion (`## What Changes` ← canonical home + signature; `## Impact` ←
call sites; `## Origin` ← the scope and detector commands, which makes the evidence
falsifiable). The decisive fit is `## Assumptions`: the SRAD table is a natural per-cluster
risk record, and `fab score`'s flat 3.0 intake gate already refuses to auto-run a change whose
divergences grade Tentative. A bespoke report format would reinvent that gate.

**What the dry run changed.** A dry run on `src/go` (2026-07-28, zero detectors installed —
which usefully exercised the unaided path every fresh repo hits) read the bodies of
`setupFabRoot`, `setupChangeFixture`, `setupArchiveFixture`, and `setupScoreFixture`. Two
findings, one of which corrects the draft:

1. *Confirmed the premise.* All four share a skeleton (`t.Helper()` → `t.TempDir()` →
   `filepath.Join(dir, "fab")` → `MkdirAll(fabRoot/changes)` → return `fabRoot`), but are
   6–20 lines with different literals, different return arities, and interleaved specifics.
   The common run is ~4 lines split by divergent code — **a token-based detector such as
   jscpd would have contributed nothing on this cluster.** The agent's reading is the only
   mechanism that finds it.
2. *Corrected the design.* It is **not one cluster with one signature** — it is a shared base
   plus opt-in layers:

   | Layer | Required by |
   |---|---|
   | tempdir + `fab/changes` | all members |
   | + `fab/project/config.yaml` | change, score |
   | + a change dir with `.status.yaml` | archive, score, resolve (via its own `createChange`) |
   | + `.fab-status.yaml` symlink | archive |
   | + kit override with templates | change |

   So the unified API is `newFabRoot(t, opts...)` with functional options, not the flat
   `newFixtureRepo(t, opts...)` the draft sketched. Notably `resolve_test.go` has **already**
   factored change-creation into a separate `createChange` helper — the right decomposition,
   and the shape the shared helper should take.

   The draft skill's Step 3 records a single "proposed signature" per cluster. That is too
   flat: real clusters are layered, and forcing one signature per cluster produces a bad API.
   **Step 3 must decompose into a shared core plus variation layers**, and Step 4's ranking
   must treat "member needs only the base layer" as *low* divergence even when member bodies
   look different.

## What Changes

### 1. New skill: `src/kit/skills/fab-dedupe.md`

A user-invocable skill, `/fab-dedupe [scope]`. Structurally a **thin call-site over the
shared `_intake` Create-Intake Procedure** (Steps 0–9, `{questioning-mode} = interactive`),
stopping at intake `ready` — no activation, no git branch. It is `/fab-draft` with a
cluster-analysis front end and a fan-out tail (N intakes rather than one).

Frontmatter:

```yaml
---
name: fab-dedupe
description: "Sweep a scoped area for duplicated utilities, cluster them by behavioral shape, and draft one intake per accepted cluster group. Read-only until you approve."
helpers: [_generation, _srad, _intake]
---
```

**Pre-flight**: verify `config.yaml` + `constitution.md`; do **not** run `fab preflight` — the
skill operates without an active change and must not resolve or disturb one.

**Behavior (5 steps)**:

- **Step 1 — Resolve scope.** `[scope]` is natural language or a path/glob. Natural language
  resolves against `source_paths`/`test_paths` and is confirmed with the user before sweeping.
  A bare invocation defaults to `source_paths` with a warning about list length. Echo the
  resolved path set and file count before proceeding.

  Scope filters *where clusters may be found*, not *where their members may live*: a cluster
  seeded inside the scope MAY include members outside it, and those members MUST be reported.
  Consolidating half a cluster is worse than not consolidating it.

  **Cluster acceptance is collected as a plain conversational reply** (`all` / `1,3` / `none`)
  — not via a structured multi-select. Decided during intake (was O2): it matches how every
  other fab skill interacts, sets no new precedent, and handles arbitrary cluster counts
  (a structured picker caps at 4 options).

- **Step 2 — Run detectors.** For each configured detector: probe with
  `command -v <bin> >/dev/null 2>&1`, substitute placeholders, run, collect output. Record
  per detector: ran / skipped-missing / ran-with-exit-code-N.

- **Step 3 — Cluster by behavioral shape.** The core step, and the part no tool performs.
  Read functions in scope — *seeded by* detector output but **not limited to it** — and group
  members that solve the same problem regardless of name, signature, or implementation.

  Per cluster record: members (`file:line` + current name); shared behavior (one line);
  **divergences** (where members genuinely differ — this field carries the risk); proposed
  canonical home (following the repo's own layout conventions, read from neighboring
  directories); **shared core + variation layers** (see below); call-site count; and members
  outside the swept scope, flagged explicitly.

  **Layered decomposition (per the dry-run correction).** Do not record a single flat
  signature. Record the base behavior every member needs, then each opt-in layer with the
  members requiring it, then the unified API expressing that shape (functional options,
  options struct, base + wrappers, or an explicit "these two should stay separate").

  Two rules: **do not cluster on name similarity alone** (two `parseConfig` functions in
  different packages may be unrelated; `mustTempRepo` and `newFixtureDir` may be identical —
  read the bodies); and **a cluster of one is not a cluster** — drop it.

- **Step 4 — Rank and present.** Rank by consolidation value: high call-site count and low
  divergence first. A member needing only the base layer counts as **low** divergence even
  when its body looks different. Present the read-only sweep report, then **ask** which
  clusters to act on. Accepting nothing is a valid, complete outcome.

  ```
  Consolidation sweep — src/go (test helpers)
  Detectors: jscpd (skipped — not installed)
  Scanned: 104 files, 312 functions

    1. Fab-root test scaffolding — 12 members, 12 call sites, low divergence
       internal/{resolve,change,archive,score,...}_test.go
       → newFabRoot(t, opts...) in internal/fabtest
       Base: tempdir + fab/changes (all 12)
       Layers: +project config (2) · +change dir (3) · +active symlink (1) · +kit override (1)

    2. ...

  2 clusters. Which should become changes? (all / 1,3 / none)
  ```

- **Step 5 — Draft intakes.** Per accepted cluster group, read
  `.claude/skills/_intake/SKILL.md` and execute the Create-Intake Procedure (Steps 0–9,
  `{questioning-mode} = interactive`). Group by refactor coherence, not count — clusters
  sharing a canonical home go in one change; unrelated clusters get separate changes.
  **Prefer one intake per cluster** (bundling makes apply all-or-nothing and forces one review
  verdict over independent refactors).

  Binding into the shared procedure: **Step 0** takes natural-language input
  (`consolidate {shared behavior} into {canonical home}`) — no backlog/Linear ID, so no
  collision check runs and each invocation creates a fresh change. **Step 2 (gap analysis)**
  is the duplicate-work guard: check the utilities memory file and in-flight changes per
  cluster; skip those already handled. **Step 5** populates the template from cluster data
  (Origin ← scope + detector commands + exit codes; Why ← member/call-site counts and the
  cost of leaving it; What Changes ← canonical home, layered API, per-member migration notes;
  Impact ← every call site plus out-of-scope members; Affected Memory ← the utilities file,
  `(new)` first time then `(modify)`; Open Questions ← unresolved divergences).
  **Step 8 (SRAD)** grades honestly: identical members + obvious home → Certain; minor
  divergence with a clear unified API → Confident; divergence that might drop a behavior →
  Tentative; cannot tell whether members are the same → Unresolved, and ask. A cluster
  grading mostly Tentative is telling you it should not be auto-consolidated — let it fail the
  gate rather than talking the score up.

  **STOP after Step 9.** No activation, no git branch. Those steps live only in
  `fab-new.md`'s tail, which this skill never reads, so there is no momentum hazard.

**Output**: per drafted intake, name and confidence; then the Activation Preamble `Next:`
line per `_preamble.md` § Activation Preamble. When no clusters are accepted, end with the
report and **no** `Next:` line — nothing was created.

**Error handling**: config/constitution missing → STOP. Scope resolves to no files → STOP
with the resolved set shown. All detectors missing → continue unaided, note in report. A
detector exits non-zero → **continue** (see below). Unparseable detector output → continue,
treat as no seed. No clusters found → report and stop (a legitimate outcome). User accepts
nothing → report and stop. `fab change new` fails for one cluster → report, continue with the
rest, list failures at the end.

### 2. Detector layer — fail-silent, config-driven

Detectors **seed** the sweep; they never perform it. Read `consolidate.detectors` from project
config; absent → default to jscpd alone.

```yaml
consolidate:
  detectors:
    - jscpd --reporters json --output {out} {paths}
```

Placeholders substituted before execution: `{paths}` (space-joined resolved scope paths) and
`{out}` (a scratch directory the skill creates for the run).

**Two behavioral rules, both load-bearing:**

- **Probe then fail silent.** `command -v <bin>` before invoking; missing → skip silently and
  continue. This mirrors `_preamble.md` § Run-Kit Reference's discipline for `rk`: never
  error, never warn, on an absent optional tool. Report which detectors ran so a skip is
  visible as *information*, not as failure.
- **Non-zero detector exit is NOT a STOP.** Duplication tools conventionally exit non-zero to
  signal "threshold exceeded" — a finding, not an error. Parse available output, note the exit
  code, continue. This is an **explicit per-skill exception** to `_preamble.md` § Common fab
  Commands' failure rule, which governs `fab` commands rather than third-party tools, and the
  skill must say so in those words.

**Suggested per-language detectors** (documented in the skill as a table, not defaults —
verified 2026-07-28):

| Language | Detector | Clone type | Note |
|---|---|---|---|
| any | `jscpd` | Type-1/2 | Default. 223 formats; v5 is a Rust rewrite (no Node runtime); JSON + SARIF + an `ai` reporter for LLM pipelines |
| TypeScript | `similarity-ts` | **Type-3** | Strongest Type-3 tool available (TSED tree-edit-distance). **No JSON output** — text parsing required |
| Go | `golangci-lint run --enable-only dupl --out-format json` | Type-1/2 | JSON via golangci-lint; bare `dupl` emits none |
| Rust | `cargo-dupes --format json` | Type-2/3 | AST normalization + Dice coefficient |
| Python | `pylint --disable=all --enable=R0801` | Type-1/2 | Ruff has declined a duplication rule |
| Ruby | `flay` | Type-2/3 | AST-based |
| any | `qlty smells --json` | Type-2 | 40+ languages; alternative to jscpd |

The skill must state two facts plainly, because they set correct expectations: **no shipping
CLI does true Type-4 detection** (claims resolve to unmaintained research artifacts), and **on
Go the best available tier is Type-1/2**, so Go sweeps lean harder on the agent than
TypeScript sweeps do.

### 3. Config registry: one new key (Go)

The key is added to the registry in `src/go/fab/internal/configref/configref.go`, following
the `checklist.extra_categories` entry shape exactly (`Key` / `Default` / `Description` /
`Scope` / `Advertise` / `Segment`).

**`consolidate.detectors`** — `Default: nil`, `Scope: ScopeProject`, `Advertise: true`.
Description: "Duplicate-detection commands `/fab-dedupe` runs to seed its sweep. Each entry is
a shell command template; `{paths}` and `{out}` are substituted at run time. Missing binaries
are skipped silently."

Segment (rendered into the managed fence):

```
# consolidate.detectors — duplicate-detection commands /fab-dedupe runs to seed
# its sweep. Each entry is a shell command template; {paths} (the resolved scope)
# and {out} (a scratch dir) are substituted at run time. A detector whose binary
# is missing is skipped silently; a non-zero exit is treated as a finding, not an
# error. Detectors only SEED the sweep — the agent does the clustering.
# consolidate:
#   detectors:
#     - jscpd --reporters json --output {out} {paths}
```

**No `consolidate.memory_file` key.** The memory home is **hardcoded** to
`docs/memory/_shared/utilities.md` in the skill prose. Decided during intake (was O1): the key
would double the registry surface and the golden-fixture churn for a knob with no known
consumer. Add it later if a repo genuinely needs a different home — adding a registry key is
cheap and non-breaking; carrying an unused one is not. The skill MUST NOT reference
`consolidate.memory_file` as an override.

**Consequential detail**: `src/go/fab/internal/configupgrade/golden_test.go` holds golden
fixtures of the rendered managed fence. Adding advertised registry keys **changes that
rendered output**, so the golden fixtures must be regenerated in this change. There is no
test file in `configref/` itself; coverage for the new keys goes in the existing
`configupgrade` and `cmd/fab/config_test.go` suites. Per `code-review.md`, "a `.go` change
without accompanying test updates is a must-fix gap."

### 4. SPEC mirror: `docs/specs/skills/SPEC-fab-dedupe.md`

Constitution-required (Additional Constraints: "Changes to skill files MUST update the
corresponding `docs/specs/skills/SPEC-*.md` file"). Follows the house shape observed in
`SPEC-fab-draft.md`: Summary, design rationale, Flow diagram, Tools used, Sub-agents (none),
Bookkeeping commands, Difference-from-`/fab-draft` table, Re-run contract, Known limitations.

### 5. Aggregate spec + memory updates (the sweep class)

Per `code-quality.md` § Sibling & Mirror Sweeps — the named most-common rework cause in this
project — the whole class must be swept **up front**:

- `docs/specs/skills.md` — per-skill behavior, add `/fab-dedupe`
- `docs/specs/glossary.md` — terms introduced: *cluster*, *detector*, *canonical home*
- `docs/specs/config.md` — the two new registry keys (this spec is the per-field metadata
  table the registry renders)
- `src/kit/skills/fab-help.md` + `SPEC-fab-help.md` — the command list
- `src/kit/skills/_cli-fab.md` — **only if** a `fab` command signature changes. Expected: no
  change, since the skill adds no subcommand. Verify rather than assume.
- `docs/memory/pipeline/` — a memory file for the skill's behavior (hydrate's job, but the
  Affected Memory list below must name it so hydrate does not miss it)

### 6. Memory home: `docs/memory/_shared/utilities.md`

`_shared` is defined as "cross-cutting concerns spanning all domains," which is exactly what a
utility index is. **A top-level `utilities` domain is deliberately rejected**: every existing
domain is organized by subject matter, whereas this would be organized by *code role* and cut
across all of them. Promotion is deferred to `/docs-reorg-memory` if the single file outgrows
itself.

The skill **never writes this file**. It appears in each drafted intake's `## Affected
Memory`, and **hydrate writes it** on the normal pipeline path — so the record cannot drift
from what actually shipped.

The file carries an honest coverage header, because a silently under-covering index is worse
than no index (consumers trust it identically at 10% and 90% coverage):

```markdown
Coverage: swept `src/go` (test helpers) 2026-07-28 · not yet swept: `src/kit`, `scripts`
```

Note: this change creates the **skill**, not the file. `utilities.md` comes into existence the
first time a real `/fab-dedupe`-drafted consolidation is hydrated.

## Affected Memory

- `pipeline/fab-dedupe`: (new) The `/fab-dedupe` skill — sweep/cluster/draft behavior, the
  detector seeding model and its fail-silent contract, layered cluster decomposition, and the
  read-only-until-accept property
- `pipeline/planning-skills`: (modify) Add `/fab-dedupe` to the `_intake` consumer set — the
  fourth consumer and the first *fan-out* call site (Steps 0–9 run once per accepted cluster
  group). Touches the frontmatter `description:`, the `{questioning-mode} = interactive`
  binding, the per-consumer call-site table, the helper-declaration mechanics paragraph, and
  the `Pre-Boundary Intake-Creation Extracted to _intake` Design Decision. *(Path corrected
  during apply rework: the intake originally named the non-existent `pipeline/skills-overview`;
  the real file is `docs/memory/pipeline/planning-skills.md`.)*
- `_shared/config-conventions`: (modify) The two new `consolidate.*` registry keys — verify
  the actual config-carrying filename in `docs/memory/_shared/index.md` during apply

## Impact

**New files (2)**
- `src/kit/skills/fab-dedupe.md` — the skill (draft exists; correct the Step 3 flat-signature
  bug before shipping)
- `docs/specs/skills/SPEC-fab-dedupe.md` — the mirror (draft exists)

**Modified — Go (1 source + tests)**
- `src/go/fab/internal/configref/configref.go` — one registry entry
  (`consolidate.detectors`)
- `src/go/fab/internal/configupgrade/golden_test.go` — **golden fixtures regenerate** (the
  rendered fence changes)
- `src/go/fab/cmd/fab/config_test.go` — coverage for the new keys

**Modified — docs/specs (sweep class)**
- `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/config.md`
- `src/kit/skills/fab-help.md` + `docs/specs/skills/SPEC-fab-help.md`
- `src/kit/skills/_cli-fab.md` — verify; expected unchanged

**Backlog**
- `fab/backlog.md` — `[ruaw]` is substantially absorbed by this skill (its inventory arrives
  as consolidation output rather than as a curated artifact). Update or annotate it; do not
  silently leave it as though untouched. `[cmap]` is unaffected and stays.

**Not in scope**
- No refactor of the `setupXRepo` cluster this change discovered. That is the first *use* of
  the skill, and belongs in its own change.
- No `fab` subcommand, no new binary surface, no Constitution amendment.
- Cross-repo duplication (utilities duplicated *between* separate repos) — a materially
  harder problem, explicitly out of scope.

## Open Questions

Both questions raised at intake were resolved with the user before this artifact reached
`ready`; they are recorded as Certain assumptions 19 and 20 rather than left open.

- ~~**O1** — Ship `consolidate.memory_file` in v1?~~ **Resolved**: no. Hardcode
  `docs/memory/_shared/utilities.md`; ship only `consolidate.detectors`.
- ~~**O2** — Structured multi-select or conversational reply for cluster acceptance?~~
  **Resolved**: conversational reply.

None outstanding.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill named `fab-dedupe` | User chose it from four presented candidates (`fab-dedupe`/`fab-reuse`/`fab-consolidate`/`fab-extract`) | S:100 R:85 A:95 D:100 |
| 2 | Certain | Output artifact is an intake per accepted cluster, not a bespoke report | User confirmed ("The output of this skill - is it an intake?" → yes); the SRAD table + flat 3.0 gate already provide per-cluster risk grading a custom format would reinvent | S:95 R:75 A:95 D:90 |
| 3 | Certain | Skill does not refactor — it drafts intakes; the pipeline consolidates | Discussed and agreed; a self-contained refactor loop would bypass the SRAD gate, code-quality.md, and the reviewer | S:95 R:80 A:100 D:95 |
| 4 | Certain | Thin call-site over `_intake` Steps 0–9, stop at `ready` (the `/fab-draft` shape) | `_intake.md` documents exactly this consumer pattern; `/fab-draft` is the working precedent | S:90 R:85 A:100 D:95 |
| 5 | Certain | Memory home is `docs/memory/_shared/utilities.md`, written by hydrate | `_shared` is defined as cross-cutting concerns; hydrate-writes prevents drift from what shipped | S:85 R:80 A:95 D:90 |
| 6 | Certain | Registry entries follow the `checklist.extra_categories` shape | Read directly from `configref.go` — same optionality, `Advertise: true`, `Segment` rendering | S:90 R:85 A:100 D:95 |
| 7 | Certain | Golden fixtures in `configupgrade/golden_test.go` must regenerate | Verified: advertised registry keys render into the managed fence, which the fixtures capture | S:95 R:80 A:100 D:100 |
| 8 | Certain | Detectors seed; the agent clusters | Verified 2026-07-28 that no shipping CLI does Type-4; the dry run confirmed jscpd would find nothing on the target cluster | S:95 R:85 A:95 D:95 |
| 9 | Certain | Cluster records decompose into shared core + variation layers, not one flat signature | Dry run on four `src/go` members proved the layered shape; the flat form would produce a bad API | S:100 R:75 A:95 D:95 |
| 10 | Confident | Non-zero detector exit is a finding, not a STOP (explicit exception to the preamble failure rule) | Duplication tools conventionally exit non-zero on "threshold exceeded"; the preamble rule governs `fab` commands, not third-party tools | S:75 R:80 A:90 D:80 |
| 11 | Confident | Detector probing is fail-silent, mirroring the `rk` discipline | `_preamble.md` § Run-Kit Reference is the established in-repo precedent for optional external tools | S:80 R:85 A:90 D:85 |
| 12 | Confident | Default detector list is jscpd alone | Only tool present in every language row; single binary, no runtime (v5 is a Rust rewrite), and ships an `ai` reporter built for LLM pipelines | S:75 R:90 A:85 D:80 |
| 13 | Confident | One intake per cluster (grouped only by shared canonical home) | Bundling makes apply all-or-nothing and forces one review verdict over independent refactors | S:75 R:75 A:85 D:80 |
| 14 | Confident | Sweep-class files: skills.md, glossary.md, config.md, fab-help + mirror | `code-quality.md` § Sibling & Mirror Sweeps names this the most common rework cause and requires an up-front sweep | S:80 R:70 A:90 D:85 |
| 15 | Confident | Existing draft files are working material, to be corrected rather than preserved | Written pre-pipeline in this session; the dry run has since invalidated their Step 3 | S:85 R:85 A:85 D:75 |
| 16 | Tentative | Affected Memory filenames are conceptual — actual paths verified during apply | `docs/memory/pipeline/` and `_shared/` contents were not enumerated at intake time; the domain choice is confident, the filenames are not | S:60 R:70 A:45 D:55 |
| 17 | Tentative | `_cli-fab.md` needs no update (no `fab` command signature changes) | The skill adds no subcommand, but the constitution's CLI constraint is read strictly by reviewers; verify during apply rather than assume | S:65 R:75 A:55 D:60 |
| 18 | Tentative | `[ruaw]` backlog entry is updated/annotated rather than removed | It is substantially absorbed, but its process layer (ledger, `reuse` acceptance category) is not built here; the disposition is a judgment call | S:55 R:80 A:50 D:50 |
| 19 | Certain | Ship only `consolidate.detectors`; hardcode the memory home (no `memory_file` key) | Asked — user chose hardcoding: the key doubles registry surface and golden-fixture churn for a knob with no known consumer, and adding it later is cheap and non-breaking | S:100 R:90 A:90 D:100 |
| 20 | Certain | Step 4 collects cluster acceptance as a plain conversational reply | Asked — user chose conversational: matches every other fab skill, sets no new precedent, and handles arbitrary cluster counts (a structured picker caps at 4) | S:100 R:95 A:90 D:100 |

20 assumptions (11 certain, 6 confident, 3 tentative, 0 unresolved).
