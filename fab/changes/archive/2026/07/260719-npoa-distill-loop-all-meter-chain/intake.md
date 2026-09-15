# Intake: Distill Loop-All + Narration-Meter Fix + Reorg→Distill Chain

**Change**: 260719-npoa-distill-loop-all-meter-chain
**Created**: 2026-07-19

## Origin

Promptless dispatch from `/fab-proceed` (create-intake subagent, `{questioning-mode} = promptless-defer`). The change description was synthesized from the live conversation, in which the user made all three core decisions and explicitly rejected three alternatives:

> **Title/scope**: Simplify /docs-distill-memory usage (no-arg loops all flagged domains), fix the narration-density meter false-positive, and chain /docs-reorg-memory → /docs-distill-memory via Next: lines.
>
> 1. /docs-distill-memory no-arg default becomes a sequential all-domains loop (per-domain approval gate retained). The user's finding: nobody will re-invoke per domain; the "one domain per run" rule was only ever justified as a property of the APPROVAL UNIT, not of the invocation.
> 2. Prerequisite Go fix: the narration-density meter must stop counting allowed citations (trailing `(change-id)` citations, `*Introduced by*:` fields) — fully-distilled files never clear the flag today.
> 3. Chain reorg → distill via Next: lines (bidirectional discoverability) — `Next: /docs-distill-memory (N files flagged across M domains)` when candidates exist.
>
> Explicitly rejected by the user: an umbrella command (/docs-groom-memory) — "No need of the umbrella command"; merging the two skills — "No need to merge"; bulk approval across all domains in one prompt — collapses the per-domain human gate.

The prerequisite meter caveat is already recorded in project memory: "narration-density meter counts allowed citations, so distilled files never clear the flag — decompose stems vs id-tokens before proposing a domain."

## Why

1. **The no-arg distill flow stalls after one domain.** Today `src/kit/skills/docs-distill-memory.md` Behavior Step 0 surveys all domains via one `fab memory-index --check --json` call, auto-picks the FIRST flagged domain, runs the one-domain flow, and ends with a `Next:` line telling the user to re-invoke once per remaining domain. In practice nobody re-invokes per domain, so the corpus never converges. The "one domain per run" rule was justified as a property of the **approval unit** (a human approves per domain, seeing per-file diffs) — never of the invocation — so looping domains within one invocation loses no safety.

2. **The narration-density meter can never report success.** The `narration-density` warning (`fab memory-index --check`, FKF §3.3 distillation-debt meter, fires at ≥5 markers) counts registry-gated change-id token **occurrences** in addition to transition stems — `src/go/fab/internal/memoryindex/memoryindex.go` `topicBodyWarnings`: `markers := countNarrationStems(body) + countChangeIDOccurrences(body, reg)`. But trailing `(change-id)` citations and `*Introduced by*:` fields are exactly the provenance FKF §3.3 tells distillation to **keep** — so a fully-distilled file that retains its sanctioned citations never clears the flag, and the survey over-flags already-clean domains. Under the new loop-all default this gets worse: every run full-reads clean domains, and the terminal "all domains distilled" state is unreachable. Without the fix, part 1 is a treadmill.

3. **The reorg → distill composition order is invisible.** `/docs-distill-memory` already points at `/docs-reorg-memory` in its `Next:` line, but not vice versa. The fixed composition order — structure first (reorg), prose second (distill) — should be self-guiding with zero new command surface.

## What Changes

### 1. `/docs-distill-memory` no-arg default: sequential all-domains loop

**File**: `src/kit/skills/docs-distill-memory.md` (canonical; never `.claude/skills/`).

Current no-arg behavior (Behavior Step 0): survey once → auto-pick the FIRST flagged domain → one-domain flow (full read → per-file proposed-rewrite report → approval → apply → regen) → dynamic `Next:` line listing remaining flagged domains for the user to re-invoke per domain.

New no-arg behavior: **survey once** (unchanged single `fab memory-index --check --json` call, same four-kind aggregation, same exclusion set, same older-binary grep fallback), then **iterate EVERY flagged domain sequentially in `docs/memory/index.md` domain-table order**. Per domain, the existing one-domain flow runs unchanged as the loop body:

1. Full read of the domain's topic files (Step 1).
2. Per-file proposed-rewrite report (Step 2).
3. **Per-domain approval prompt** (Step 3: apply all / cherry-pick / skip) — deliberately retained; the user explicitly REJECTED bulk approval across all domains at once (it would collapse the human safeguard on load-bearing memory files).
4. Apply approved rewrites (Step 4) + regenerate indexes with the refuse-before-regen guard (Step 5).
5. Proceed to the next flagged domain.

Semantics:

- A **skipped** domain stays untouched and moves the loop on; it is reported in the terminal summary as skipped/remaining.
- The loop iterates the **initial** survey's flagged list (survey once — no re-survey between domains); a domain whose full read finds nothing reports "no rewrites proposed — already distilled" and the loop continues.
- Explicit `<domain>` argument is unchanged: the targeted single-domain override — forces a full read, skips the survey, no loop. The multiple-explicit-domains abort in Error Handling stays.
- An exit-2 refuse-before-regen event within one domain follows the existing per-domain handling; it does not silently swallow the remaining domains (report, then continue or stop per existing error-handling posture).
- **Terminal state of a no-arg run**: "all domains distilled" (every flagged domain processed) or a summary listing skipped/remaining domains.
- **Dynamic `Next:` line semantics change**: it now reports skipped/remaining domains (surveyed truth) rather than driving per-domain re-invocation — e.g. `Next: all domains distilled (survey heuristic) — /docs-reorg-memory or /fab-new`, or a line listing the skipped domains with flagged counts for a follow-up targeted run.

**Reframe all "one domain per run" language** — in the skill's frontmatter `description:`, Purpose, Arguments, Key Properties, Output, and Error Handling: one domain per **approval/apply unit**, iterated within a single invocation. The loop runs in the main session (the approval prompt is interactive and must reach the user; no per-domain subagent dispatch).

### 2. Prerequisite Go fix: narration-density meter stops counting allowed citations

**Files**: `src/go/fab/internal/memoryindex/memoryindex.go` (+ tests in `src/go/fab/internal/memoryindex/memoryindex_test.go`, and `src/go/fab/cmd/fab/memory_index_test.go` if the JSON-surface tests assert marker counts).

Current implementation (`topicBodyWarnings`, ~line 1260):

```go
// Narration-marker density: case-insensitive transition-stem hits + the
// registry-gated change-id token OCCURRENCES in the body (the debt meter
// counts sanctioned citations too, ...)
markers := countNarrationStems(body) + countChangeIDOccurrences(body, reg)
if markers >= NarrationMarkerWarnThreshold {
    out = append(out, Warning{Path: relPath, Kind: KindNarrationDensity, Count: markers})
}
```

with `narrationStems = []string{"no longer", "previously", "renamed", "supersed"}` and `NarrationMarkerWarnThreshold = 5`. Counting sanctioned citations was a **deliberate** prior design ("density is the signal, not violation") that this change inverts, per the recorded consequence: fully-distilled files never clear the flag.

New behavior — **decompose the marker detection**:

- **Narration STEMS count** — the existing case-insensitive stem list (`no longer` / `previously` / `renamed` / `supersed`) is unchanged (no list expansion in this change; the skill-file grep-fallback patterns like `` was ` `` / `inverts` are a separate agent-side heuristic and stay as they are).
- **Change-id TOKENS in allowed positions do NOT count**: (a) a parenthesized change-id — the trailing `(change-id)` citation form §3.3 sanctions (both full `YYMMDD-XXXX-slug` and registry-gated bare 4-char ids); (b) a change-id on an `*Introduced by*:` field line (the Design-Decisions provenance field).
- **Change-id tokens outside allowed positions still count** — a bare id woven into prose remains a narration marker (density signal for ids embedded in narration).
- Threshold (5), warning kind (`narration-density`), advisory status (never affects exit code), `Count` semantics (marker count), and the `warnings[]` JSON shape are all unchanged.

Tests ship in the same change (constitution: test-alongside; tests conform to spec): a distilled fixture carrying only trailing `(id)` citations and `*Introduced by*:` lines must NOT flag; a fixture with ≥5 stems must flag; a mixed fixture counts stems + non-allowed-position ids only.

**Prerequisite ordering**: part 2 lands before or together with part 1 inside this change (task ordering in the plan) — otherwise the loop-all default full-reads clean domains forever.

### 3. Documented-semantics sweep for the meter change

The old counting rule is restated verbatim in several doc surfaces (verified present); all must be updated in this change:

- `src/kit/skills/_cli-fab.md` § fab memory-index (~line 813): "plus registry-gated change-id token **occurrences** (sanctioned citations count too — density is the distillation-debt signal, ...)".
- `docs/specs/skills/SPEC-_cli-fab.md` (line ~36): "(transition stems + registry-gated change-id occurrences, ≥5 — the distillation-debt meter)".
- `docs/specs/fkf.md` § Present-truth debt meters (~line 273): "plus registry-gated change-id token occurrences in the body — ...; sanctioned citations count too, because density, not violation, is the signal".
- `src/kit/reference/fkf.md` (the shipped extract) — mentions the debt meters (~line 105) without the counting detail; update only if it restates the changed semantics (verify during apply).
- `src/kit/skills/docs-distill-memory.md` Step 0 note that `_shared/removed-domains.md`'s "citation-dense rows trip `narration-density`" — re-verify after the fix (tombstone rows carrying parenthesized/field-position ids may no longer trip it); the exclusion-set re-application stays regardless (harmless + still correct for the description-tier kinds).
- `docs/memory/pipeline/schemas.md` (~line 228): "sanctioned citations count, so it is advisory" — memory update at hydrate.

### 4. `/docs-reorg-memory` completion chains to `/docs-distill-memory`

**File**: `src/kit/skills/docs-reorg-memory.md` (canonical).

Reorg already runs a single `fab memory-index --check --json` call (Step 1 — one call feeds three consumers: `losses[]`, `warnings[]` `file-size`, `unsorted-nonempty`), which already carries the `narration-density` and description-tier findings. **Reuse that call's output** (no second survey call): aggregate flagged files with the same rule as distill's survey (four kinds — `description-change-id`, `description-over-cap`, `description-length`, `narration-density`; dedupe by path; sub-domain rolls up to domain; re-apply the distillation exclusion set), and at completion emit:

```
Next: /docs-distill-memory (N files flagged across M domains)
```

when N ≥ 1, listed first; the normal completion output otherwise. On the older-binary fallback path (no `warnings[]` machine surface), omit the counts (plain pointer or normal Next: line — graceful degradation alongside the existing upgrade warning). Note reorg currently ends with its own completion output (no `Next:` convention line) — the added line follows the `_preamble` § Next Steps Convention skill-file-wins carve-out.

Rationale (user decision): fixed composition order — structure first (reorg), prose second (distill) — becomes self-guiding with zero new command surface. Distill's existing `Next:` pointer at reorg is the other half of the bidirectional chain (already present).

### Explicitly rejected alternatives (user decisions — record for downstream agents)

- **Umbrella command** (e.g. `/docs-groom-memory`) orchestrating reorg+distill — rejected ("No need of the umbrella command").
- **Merging the two skills into one** — rejected ("No need to merge"): different units of change (structural migrations vs per-file prose rewrites), different approval grammars, both files already ~310 lines.
- **Bulk approval across all domains in one prompt** — rejected: collapses the per-domain human gate.

### Constraints / sweep class

- Skill edits go to canonical `src/kit/skills/*.md` — NEVER `.claude/skills/` (gitignored deployed copies).
- Constitution: every `src/kit/skills/*.md` change MUST update its `docs/specs/skills/SPEC-*.md` mirror in the same change — `SPEC-docs-distill-memory.md`, `SPEC-docs-reorg-memory.md`, `SPEC-_cli-fab.md` — and sweep the aggregate specs restating these behaviors: `docs/specs/skills.md` (distill entry lines ~760–778 restates the no-arg survey/auto-pick and the "one domain per run" property; reorg entry ~745–756), `docs/specs/glossary.md` (line ~58, "/docs-distill-memory ... One domain per run").
- Go change ships tests in the same change; `src/kit/skills/_cli-fab.md` documents the changed warning semantics (part 3).
- `docs/memory/memory-docs/distill.md`'s Design Decision *"No-arg survey with auto-pick"* (Introduced by 260718-ukpf) has a **Rejected (a)** entry — "a multi-domain invocation expanding to sequential per-domain runs with per-domain approval gates — a single session chewing through all domains erodes rewrite quality as context fills" — that this change **supersedes by user decision**; hydrate must rewrite that DD to present truth, not merely append.

## Affected Memory

- `memory-docs/distill`: (modify) no-arg semantics become the sequential all-domains loop (per-domain approval retained); "one domain per run" reframed to per-approval-unit; the 260718-ukpf DD's Rejected (a) entry (sequential per-domain runs) is superseded by user decision and must be rewritten; dynamic `Next:` semantics updated (reports skipped/remaining, no longer drives re-invocation)
- `pipeline/schemas`: (modify) `KindNarrationDensity` counting semantics — stems + change-id tokens outside allowed positions; sanctioned citations (trailing parenthesized citations, `*Introduced by*:` fields) no longer count
- Domain/root `index.md` + `log.md` regeneration via `fab memory-index` at hydrate (generated — not hand-edited)

## Impact

- **Skills (markdown)**: `src/kit/skills/docs-distill-memory.md` (major — no-arg loop, reframed language, Next: semantics), `src/kit/skills/docs-reorg-memory.md` (small — completion Next: chain), `src/kit/skills/_cli-fab.md` (narration-density row).
- **Go**: `src/go/fab/internal/memoryindex/memoryindex.go` (`topicBodyWarnings` marker computation + related comments/doc-strings at lines ~66–70, ~112–116, ~246, ~847, ~1248; `countChangeIDOccurrences` repurposed or replaced by a position-aware counter) + tests (`memoryindex_test.go`, possibly `cmd/fab/memory_index_test.go`). Scoped test run: `go test ./internal/memoryindex/... ./cmd/fab/...` from `src/go/fab`.
- **Specs**: `docs/specs/skills/SPEC-docs-distill-memory.md`, `SPEC-docs-reorg-memory.md`, `SPEC-_cli-fab.md`, `docs/specs/skills.md`, `docs/specs/glossary.md`, `docs/specs/fkf.md` (+ `src/kit/reference/fkf.md` only if it restates counting semantics).
- **Memory (hydrate)**: `docs/memory/memory-docs/distill.md`, `docs/memory/pipeline/schemas.md`.
- No CLI signature change (`fab memory-index` flags/JSON shape unchanged — only the marker-counting rule inside an existing advisory warning). No migration needed (no user-data restructuring). Behavior change is confined to warning counts and skill flow.

## Open Questions

- (none — all major decisions were made by the user in the originating conversation; remaining implementation choices are recorded as graded assumptions below)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | No-arg `/docs-distill-memory` surveys once then loops every flagged domain sequentially in `docs/memory/index.md` domain-table order, running the existing full-read → per-file report → per-domain approval → apply → regen flow per domain; explicit `<domain>` stays the targeted single-domain override (full read, no survey, no loop) | Discussed — user decided; verbatim in the change description | S:95 R:85 A:95 D:95 |
| 2 | Certain | The per-domain approval gate (apply all / cherry-pick / skip) is retained; bulk approval across all domains in one prompt is rejected | Discussed — user explicitly rejected bulk approval (collapses the human safeguard on load-bearing memory files) | S:95 R:80 A:95 D:95 |
| 3 | Certain | No umbrella command and no skill merge — the reorg→distill composition is expressed only via Next: lines | Discussed — user rejected both ("No need of the umbrella command", "No need to merge") | S:95 R:90 A:95 D:95 |
| 4 | Certain | The Go meter fix (part 2) is a prerequisite ordered before/with part 1 inside this same change, and ships with tests | Change description states the ordering; constitution mandates test-alongside | S:90 R:80 A:90 D:90 |
| 5 | Confident | Marker decomposition: existing stem list (`no longer`/`previously`/`renamed`/`supersed`) unchanged; change-id tokens in allowed positions — parenthesized `(id)` citations and `*Introduced by*:` field lines — do not count; change-id tokens outside allowed positions still count | Description names the two allowed positions; "in allowed positions do NOT count" implies positional decomposition rather than dropping id-tokens entirely; stem-list expansion is out of scope (the fix targets the false positive, not recall) | S:75 R:85 A:80 D:70 |
| 6 | Certain | Threshold (≥5), warning kind name, advisory (non-blocking) status, `Count` semantics, and the `--check --json` `warnings[]` shape are unchanged — only the marker-counting rule changes | Description scopes the fix to counting; no CLI-surface change implied anywhere | S:85 R:85 A:90 D:90 |
| 7 | Certain | The documented-semantics sweep covers `_cli-fab.md` § fab memory-index, `SPEC-_cli-fab.md`, `docs/specs/fkf.md` § debt meters, and `pipeline/schemas.md` (hydrate) — all verified to restate "sanctioned citations count"; `src/kit/reference/fkf.md` is updated only if it restates the counting detail | Grep-verified during intake: exact stale phrases located in each file | S:90 R:85 A:95 D:90 |
| 8 | Confident | Reorg's chain reuses its existing single `fab memory-index --check --json` call (no second survey), aggregates flagged files with distill's survey rule (four kinds, dedupe by path, sub-domain roll-up, exclusion set re-applied), and emits `Next: /docs-distill-memory (N files flagged across M domains)` when N ≥ 1 | Description prefers reuse ("or reuse available signal from its single call — it already consumes warnings[]"); mirroring distill's aggregation keeps the two skills' counts consistent | S:75 R:85 A:80 D:65 |
| 9 | Confident | On reorg's older-binary fallback path (no `warnings[]` machine surface), the chain line degrades gracefully — pointer without counts (or the normal Next: line), alongside the existing upgrade warning | Description covers only the machine-surface case; graceful degradation mirrors both skills' existing older-binary posture | S:60 R:90 A:75 D:60 |
| 10 | Confident | The no-arg loop runs in the main session sequentially (no per-domain subagent dispatch); the context-budget concern in the superseded DD (260718-ukpf Rejected (a)) is accepted by user decision, and hydrate rewrites that DD entry | Per-domain approval prompts are interactive and must reach the user; the description specifies a sequential loop with approval gates and mentions no dispatch | S:65 R:80 A:80 D:70 |
| 11 | Certain | The multiple-explicit-domains abort stays; skipped domains stay untouched; the loop iterates the initial survey's flagged list (no re-survey between domains); terminal output reports all-distilled or skipped/remaining, and the dynamic Next: line reports skipped/remaining instead of driving re-invocation | Description states these semantics directly ("surveys once, then iterates", "a skipped domain stays untouched", explicit argument "stays as the targeted single-domain override", terminal-state and Next:-semantics sentences) | S:85 R:85 A:90 D:85 |
| 12 | Certain | Skill edits land in canonical `src/kit/skills/` only, with the full SPEC-mirror class swept in the same change (`SPEC-docs-distill-memory.md`, `SPEC-docs-reorg-memory.md`, `SPEC-_cli-fab.md`, `docs/specs/skills.md`, `docs/specs/glossary.md`) | Constitution Additional Constraints + code-quality § Sibling & Mirror Sweeps; description restates it | S:95 R:85 A:95 D:95 |

12 assumptions (8 certain, 4 confident, 0 tentative, 0 unresolved).
