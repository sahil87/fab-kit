# Intake: `fab memory-index --check-loss` — Mechanical Destructive-Loss Detection

**Change**: 260615-glwc-memory-index-check-loss
**Created**: 2026-06-15

## Origin

> fab memory-index --check-loss: mechanical destructive-loss detection for pre-fab-kit memory trees, with reorg rewire and hydrate-stage guard

Direct follow-on to **5ewp** (PR #415, merged — `/docs-reorg-memory` orchestrates frontmatter backfill for pre-fab-kit trees). 5ewp shipped the *prose-based* version of pre-fab-kit compatibility detection: the reorg skill **eyeballs** the tree during its read-all-files pass to find the three ways a hand-curated tree diverges from the convention `fab memory-index` depends on (missing `description:` frontmatter, tombstone rows, custom groupings). This change makes that detection a **mechanical Go primitive** so reorg, the hydrate stage, and CI can all consult one authoritative answer instead of re-deriving it in prose.

Decisions established during the discussion that produced 5ewp and this follow-on (verified against the 2.4.0 source — this worktree is now synced to `origin/main` @ `fab_version: 2.4.0`, matching the installed binary, so the Go I read is the Go that runs):

- **The check is mechanical and the data is already in hand.** `fab memory-index` already reads every topic file's `description:` frontmatter (`memoryindex.go` `frontmatter.Field(path,"description")`) and walks the whole tree (`Gather`). The existing `--check` flag (`memory_index.go:70-83`) already compares **rendered-vs-existing** per index file and reports `out of date`. What it lacks is **classification**: `--check` treats all drift identically — it cannot distinguish a *benign* improvement (a description got better) from a *destructive* loss (a curated description becomes `—`, a tombstone row vanishes, a grouping flattens).
- **Put it in the binary, not the hydrate stage.** The natural home is `fab memory-index` itself (extend it with a loss-detection mode), because that is the exact moment of truth and every caller (reorg, hydrate skills, the hydrate stage, CI) can consult it. A per-hydrate-stage guard alone would add cost to the 99%-safe born-compatible case to catch a 1% case that does not even reach the hydrate stage (a pre-fab-kit tree's danger is its *first* regen, long before it runs the pipeline's hydrate stage).
- **Refusal, not a nudge.** Loss detection MUST exit non-zero and block (with a `→ run /docs-hydrate-memory (backfill mode)` pointer in the error), not merely warn. A warning before an irreversible destructive regen is something users click past — the same failure mode we rejected for tombstones in 5ewp ("warn-only = silent data loss").
- **Live proof-point.** During 5ewp's own review-pr stage, Copilot flagged a stale-index condition and `fab memory-index --check` mechanically detected it (exit non-zero until regenerated). That corroborates that the loss/drift check belongs in the binary — `--check` already does the "would regen change anything" half; `--check-loss` adds the "would regen *destroy* anything" half.
- **No scope creep into 5ewp.** 5ewp was explicitly scoped "no Go changes" and is merged. This is the deliberate, separate Go change that mechanizes its prose detection.

## Why

**Problem.** 5ewp's compatibility detection lives in skill *prose* — reorg's agent reads the tree and judges divergence. That works, but it is (a) **non-mechanical** (an LLM eyeballing instead of a deterministic check), (b) **single-consumer** (only reorg has it; the hydrate stage and CI do not), and (c) **drift-prone** (the prose re-states `frontmatter.Field` semantics, tombstone heuristics, and the domains-only-flatten rule that actually live in Go — two sources of truth for one fact).

**Consequence if unfixed.** The destructive-regen guard is only as reliable as an agent's prose-following, and only fires inside one skill. A user (or CI, or a future hydrate-stage caller) who runs `fab memory-index` directly on a pre-fab-kit tree gets no mechanical guard — the exact silent-loss scenario 5ewp set out to prevent, reachable by any path that doesn't go through reorg.

**Why this approach.**
1. **Single source of truth.** The loss classification (which descriptions go `—`, which tombstones drop, which groupings flatten) is computed once in Go, where the generator's own logic already lives. Reorg's prose then *calls* the primitive instead of re-deriving it — eliminating the drift and simplifying 5ewp's reorg prose in the process.
2. **Every caller benefits.** reorg (detection step), the hydrate stage (refuse-before-regen guard), and CI (`--check-loss` as a guard) all consult the same primitive.
3. **Refuse-with-pointer** makes the guard real (blocking), not advisory.

**Alternatives rejected** (from the discussion):
- *Guard in the hydrate stage only* (the user's initial framing): the danger is the *first regen* of a legacy tree, which precedes the hydrate stage; and a per-stage guard misses CI and direct-invocation paths. Put it in the command; let the stage consult it.
- *Extend `--check` to also report loss*: considered, but conflating "drift" and "destructive loss" in one flag muddies the CI contract (`--check` should stay "would anything change", which is the right pre-commit/index-staleness guard). A distinct `--check-loss` keeps each contract clean. (Open question below: distinct flag vs. a `--check` sub-classification — leaning distinct flag.)
- *Warn-only*: irreversible loss behind an advisory the user clicks past = silent data loss. Rejected (same as 5ewp's tombstone decision).

## What Changes

### 1. Go: extend `fab memory-index --check` with severity exit-code tiers

Rather than a separate `--check-loss` flag, **extend the existing `--check` flag** (`src/go/fab/cmd/fab/memory_index.go`) to classify the rendered-vs-existing drift it already computes into **benign drift** vs **destructive loss**, and encode the severity in the **exit code** (decision #11):

- **Exit 0** — indexes clean (no regen needed).
- **Exit 1** — benign drift only (regen would change something, but destroys nothing — e.g. an *improved* description, a refreshed `Last Updated`). This is the current "out of date" condition.
- **Exit 2** — destructive loss (regen would wipe curated/historical content). Writes nothing; enumerates each loss to stderr by category; ends with the pointer `→ run /docs-hydrate-memory (backfill mode) ...`.

Loss is a **strict subset of drift**, so one render pass + one comparison serves both tiers — no second flag, no `--check --check-loss` ambiguity. Callers pick their threshold: **CI / pre-commit** fails on exit ≥ 1 (any drift — the existing contract, now tiered); the **hydrate guard and reorg** fail only on exit == 2 (destructive loss). Existing `--check` consumers that treat "non-zero = out of date" keep working unchanged (any drift is still non-zero); only the *granularity* of the code is new. An optional **`--json`** form (decision #13) emits the loss report machine-readably for reorg to parse robustly (mirrors `fab migrations-status [--json]`).

The three destructive-loss categories (the mechanical form of 5ewp's three prose signals):

1. **Curated description → `—`.** For each topic-file or domain row, if the *existing* index renders a non-empty description but the *regenerated* row would render `missingCell` (`—`) because the file lacks `description:` frontmatter — that curated text is lost. (Detectable by comparing the existing index row's description cell against the rendered row's; or equivalently, a file with a non-`—` description in the current index but `frontmatter.Field == ""`.)
2. **Tombstone row dropped.** An existing index row whose `docs/memory/`-relative link target is absent on disk — the generator (which lists only on-disk folders via `Gather`'s `os.ReadDir` walk) will silently drop it on regen. Primary signal = unresolved `docs/memory/`-relative link target; strikethrough `~~...~~` is a corroborating hint, not required; external/absolute links never count (avoids false positives). Mirrors 5ewp assumption #10 exactly.
3. **Custom grouping flattened.** Structural headings/content in the existing root `index.md` beyond the generated domains-only table (`### Apps`, `### Packages`, etc.) that the domains-only `RenderRoot` output omits.

**Reuse, don't duplicate:** the rendered-vs-existing comparison machinery already exists in the `--check` branch (`memory_index.go:70-83`); the tiering adds a classifier over the same `targets` + existing-file reads. The tombstone/grouping detection reads the *existing* index files (which `--check` does not parse today — it only string-compares), so a small parser for existing index rows is the main new logic. Keep it in `internal/memoryindex` (pure functions, unit-testable like `RenderRoot`/`Gather`), surfaced via the existing cmd flag + an optional `--json` flag.

**Test integrity (Constitution VII):** new `internal/memoryindex` unit tests (loss classification: each of the three categories → exit 2, plus the benign-drift → exit 1 case and the no-change → exit 0 case) and a cmd-level exit-code test asserting the 0/1/2 tiers. Tests conform to this spec.

### 2. Rewire `docs-reorg-memory.md` detection to call the primitive

5ewp's prose-based compatibility detection (the bullets at `src/kit/skills/docs-reorg-memory.md:58-64`) is replaced by **invoking `fab memory-index --check --json`** and parsing its enumerated loss output (exit 2 = destructive loss to surface; exit 0/1 = nothing to relocate/backfill), rather than the agent re-deriving `frontmatter.Field` semantics / tombstone heuristics / flatten rules in prose. The findings report (Step 3) and the on-approval orchestration (Step 5) are unchanged in *behavior* — they now consume the primitive's structured output. This **simplifies** the reorg prose (removes the re-stated Go semantics) and removes the two-sources-of-truth drift. The older-binary fallback (`docs-reorg-memory.md:204`) extends: if the `--check` loss-tier/`--json` is unavailable (older binary — exit code is binary, no `--json`), fall back to the prose detection (keep it as the legacy path), or warn-and-upgrade.

### 3. Refuse-before-regen guard at ALL regen sites (decision #12)

Add a one-line guard at **every** site that runs `fab memory-index` to regenerate: before regenerating, consult `fab memory-index --check`; on **exit 2** (destructive loss), **refuse to regenerate** and surface the pointer to `/docs-hydrate-memory` backfill mode. The sites:

1. **`/docs-hydrate-memory`** skill's index-regen step (the primary pre-fab-kit-tree entry point).
2. **`/docs-reorg-memory`** — already covered by change-area #2 (it consumes the primitive's output directly).
3. **`/fab-continue`** pipeline **hydrate stage** behavior (defense-in-depth).

No logic is duplicated — the loss logic lives entirely in Go; each site is the same one-line exit-code check. In a born-compatible fab-kit tree the guard is **always a no-op** (exit 0/1, never 2) — every site MUST carry a brief annotation saying so, so a future reader does not mistake the pipeline-stage guard for dead code or remove it. It only ever fires on a pre-fab-kit tree.

### 4. SPEC mirrors + memory + docs

- **`docs/specs/skills/SPEC-docs-reorg-memory.md`** — reflect the rewire (detection now calls the primitive).
- **`docs/specs/skills/SPEC-docs-hydrate-memory.md`** — reflect the hydrate-skill guard.
- **`docs/specs/skills/SPEC-fab-continue.md`** — reflect the pipeline hydrate-stage guard (defense-in-depth site #3).
- **`_cli-fab.md`** (`src/kit/skills/_cli-fab.md`) — per constitution, "Changes to the `fab` CLI (Go binary) MUST update `src/kit/skills/_cli-fab.md` with any new or changed command signatures." Document the tiered `--check` exit codes (0/1/2) and the new `--json` flag on the `fab memory-index` entry. (Note: `_cli-fab`/`_cli-external` are excluded from the SPEC-mirror requirement per the uliv naming policy — they have no `SPEC-*.md` mirror — but the constitution's CLI-doc rule still applies to `_cli-fab.md` itself.)
- **`docs/memory/`** (hydrate stage of *this* change) — update `distribution/` (the `fab` command reference) and `memory-docs/` (hydrate/templates) memory for the new flag + guard; regenerate index via `fab memory-index`.

### Canonical-source note

All skill edits target `src/kit/skills/*.md` (never the gitignored `.claude/skills/` copies). Go changes in `src/go/fab/`. Run `just test` (or `go test ./...` in `src/go/fab`) — this change DOES touch Go, unlike 5ewp.

## Affected Memory

- `distribution/distribution`: (modify) `fab memory-index --check-loss` flag added to the `fab` command surface / reference
- `memory-docs/hydrate`: (modify) hydrate-stage refuse-before-regen guard consulting `--check-loss`; reorg detection now calls the primitive (cross-reference)
- `memory-docs/templates`: (modify, possibly) reorg's compatibility-detection role now mechanical (calls the primitive) — update the reorg-orchestration paragraph added by 5ewp
- `pipeline/execution-skills`: (modify, if the `/fab-continue` hydrate behavior gains the guard) record the hydrate-stage guard
- Note: the Go schema/CLI memory lives under `distribution` (the binary/command reference) and `pipeline/schemas` (helper subcommands) — confirm exact file at hydrate.

## Impact

- **Go**: `src/go/fab/cmd/fab/memory_index.go` (tiered exit codes on `--check` + new `--json` flag + RunE branch), `src/go/fab/internal/memoryindex/` (new loss-classification pure functions + existing-index-row parser), plus tests (`memoryindex_test.go`, `memory_index_test.go`, possibly a golden test for the `--json` loss-report output).
- **Skills** (canonical source): `src/kit/skills/docs-reorg-memory.md` (rewire), `src/kit/skills/docs-hydrate-memory.md` (hydrate-skill guard), `src/kit/skills/fab-continue.md` (pipeline hydrate-stage guard), `src/kit/skills/_cli-fab.md` (tiered exit codes + `--json` doc).
- **SPEC mirrors**: `SPEC-docs-reorg-memory.md`, `SPEC-docs-hydrate-memory.md`, `SPEC-fab-continue.md` (per changed skill).
- **Memory** (this change's hydrate): distribution + memory-docs (+ pipeline if hydrate behavior changes).
- **Dependency on 5ewp**: builds directly on merged 5ewp prose — must edit the shipped `docs-reorg-memory.md`/`docs-hydrate-memory.md` as they stand in main (now synced).
- **Backward compatibility**: `--check-loss` is additive; existing `--check` and bare `memory-index` behavior unchanged. The reorg older-binary fallback keeps pre-`--check-loss` binaries working (prose path).
- **Idempotency / born-compatible no-op**: on a fab-kit-native tree, `--check-loss` is always exit 0 — no behavioral change for the common case. This is a primary acceptance concern.

## Open Questions

*(Resolved via /fab-clarify 2026-06-15 — see ## Clarifications and ## Assumptions #11–#13.)*

- ~~Distinct `--check-loss` flag vs. sub-mode of `--check`?~~ **Resolved**: ONE `--check` flag, severity in exit codes (0 clean / 1 benign drift / 2 destructive loss). No separate flag (#11).
- ~~Hydrate-guard placement?~~ **Resolved**: ALL regen sites — hydrate skill + reorg + `/fab-continue` pipeline hydrate stage (defense-in-depth), each annotated as a no-op for born-compatible trees (#12).
- ~~Loss-report output format?~~ **Resolved**: human-readable stderr by default + optional `--json` (mirrors `fab migrations-status [--json]`) for reorg to parse (#13).
- **Exact existing-index-row parser scope** (not blocking): how much of the existing (possibly hand-curated, non-generated) index format must the parser tolerate to detect tombstones/groupings reliably? Bound it to what the three categories need — to finalize at apply as an inline SRAD assumption.

## Clarifications

### Session 2026-06-15 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |
| 9 | Confirmed | — |
| 10 | Confirmed | — |

### Session 2026-06-15

| # | Q | A |
|---|---|---|
| 11 | One flag or two for drift vs. destructive-loss? | ONE `--check` flag with severity exit codes (0 clean / 1 benign drift / 2 destructive loss). Loss is a strict subset of drift; one render pass; callers pick a threshold. No separate `--check-loss`. |
| 12 | Where does the refuse-before-regen guard live? | ALL regen sites (defense-in-depth): hydrate skill + reorg + `/fab-continue` pipeline hydrate stage. No logic duplication (logic is in Go); each site is a one-line exit-code check, annotated no-op for born-compatible trees. |
| 13 | Optional `--json` loss-report form? | Yes — human-readable stderr default + optional `--json` mirroring `fab migrations-status [--json]`, so reorg parses robustly. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | This is a `feat` and DOES touch Go (unlike 5ewp) | New CLI flag + classification logic in `internal/memoryindex` + tests. New capability, not a fix. | S:95 R:80 A:100 D:95 |
| 2 | Certain | Worktree is on 2.4.0 (== installed binary) so Go edits target the correct source | Synced to origin/main @ fab_version 2.4.0; resolves the [[score-binary-source-version-skew]] hazard. Verified `wTentative=1.0` in 2.4.0 source matches the binary's 1.8 score. | S:100 R:85 A:100 D:100 |
| 3 | Certain | Constitution requires `_cli-fab.md` update for the new flag + SPEC mirrors for changed skills | Constitution CLI-doc rule + skill→SPEC-mirror rule. `_cli-fab`/`_cli-external` have no SPEC mirror (uliv policy) but the CLI-doc rule still applies. | S:100 R:75 A:100 D:100 |
| 4 | Certain | All skill edits target `src/kit/skills/`, Go in `src/go/fab/`, never `.claude/skills/` | context.md:9 + constitution canonical-source rule. | S:100 R:80 A:100 D:100 |
| 5 | Certain | Loss-detection lives in the `fab memory-index` binary, not solely the hydrate stage | Clarified — user confirmed; choice is contained/reversible (relocating the check later is a small change). The data is already in hand at regen time; every caller (reorg/hydrate/CI) consults one primitive; the danger precedes the hydrate stage. | S:95 R:80 A:85 D:85 |
| 6 | Confident | `fab memory-index --check` exits non-zero + refuses on destructive loss (blocking, not advisory) | Clarified — user confirmed. A warning before irreversible loss is clicked past; same rationale as 5ewp's rejected warn-only tombstone option. | S:95 R:70 A:85 D:85 |
| 7 | Certain | Three loss categories = the mechanical form of 5ewp's three prose signals (description→—, tombstone drop, grouping flatten) | Clarified — user confirmed; verified directly against `memoryindex.go` generator logic (codebase deterministically answers what the generator drops). Category definition is contained/reversible. | S:95 R:80 A:95 D:80 |
| 8 | Confident | Reorg detection is rewired to CALL the primitive; behavior of findings-report + orchestration unchanged | Clarified — user confirmed. Single source of truth, removes prose/Go drift, simplifies reorg prose. Older-binary fallback keeps prose path. | S:95 R:65 A:80 D:80 |
| 9 | Certain | Born-compatible fab-kit trees see no behavioral change (`--check` loss-tier always exit 0/1, never 2) | Clarified — user confirmed; this is a factual property of the design, not a guess: the three loss categories require pre-fab-kit divergence, so a native tree (frontmatter present, no tombstones, domains-only index) is provably never exit 2. Idempotency (Const. III). | S:95 R:85 A:95 D:85 |
| 10 | Confident | Reuse the existing `--check` rendered-vs-existing machinery; new logic = the loss classifier + existing-index-row parser, kept pure in `internal/memoryindex` | Clarified — user confirmed. Read `memory_index.go:70-83` — the comparison exists; classification + existing-row parsing is the delta. Mirrors `RenderRoot`/`Gather` purity for testability. | S:95 R:65 A:85 D:80 |
| 11 | Confident | ONE `--check` flag with severity in exit codes — exit 0 = clean, 1 = benign drift, 2 = destructive loss (no separate `--check-loss` flag) | Clarified — user chose single-flag exit-code tiers over two flags. Least surface; loss is a strict subset of drift so one render pass serves both; callers pick a threshold (CI ≥1, hydrate/reorg ==2); well-worn 0/1/2 idiom (grep/diff). | S:95 R:55 A:60 D:90 |
| 12 | Confident | Refuse-before-regen guard at ALL regen sites: `/docs-hydrate-memory` regen, `/docs-reorg-memory` (via #8), AND `/fab-continue` pipeline hydrate stage | Clarified — user chose defense-in-depth (all sites) over hydrate-skill-only. No logic duplication (logic is in Go; each site is a one-line exit-code check). No-op on born-compatible trees — annotate so it is not mistaken for dead code. | S:95 R:60 A:60 D:90 |
| 13 | Confident | Optional `--json` loss-report form for robust reorg parsing (human-readable default) | Clarified — user confirmed. Mirrors `fab migrations-status [--json]`; makes the reorg rewire (change-area 2) cleaner to consume. | S:95 R:65 A:60 D:85 |

13 assumptions (7 certain, 6 confident, 0 tentative).
