# Plan: HexoKit Banner Sweep (plan row C7, fab-kit)

**Change**: 260911-y50t-hexokit-banner-sweep
**Intake**: `intake.md`

> Read `intake.md` § What Changes → § Classification first. It is the load-bearing rule for
> every task below: only bucket **A** (present-tense product noun) flips to HexoKit; buckets
> **B** (`rk`-prefixed substrate), **C** (repo / formula / PR identifiers), **D** (past-tense
> narrative), and **E** (the migration file) stay byte-identical. This is a classify-then-edit
> pass, not a search-and-replace. Re-derive the occurrence list with a fresh
> `grep -rn -i 'run-kit' src/kit` before editing — the intake's line numbers are as of `c3c6a5d8`.

## Requirements

### Branding: README banner

#### R1: README blockquote carries the revised toolkit banner
`README.md` line 3 MUST be exactly `> Part of [HexoKit](https://hexokit.com) — see all projects there.` (em-dash U+2014, link text "HexoKit" only). No other README line changes; the head order `#` H1 → blockquote → contiguous badge run → prose MUST be preserved (readme-extraction rule 1).

- **GIVEN** `README.md` at `c3c6a5d8` with line 3 `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`
- **WHEN** apply completes
- **THEN** `sed -n 3p README.md` prints exactly `> Part of [HexoKit](https://hexokit.com) — see all projects there.`
- **AND** `git diff --stat -- README.md` shows one line changed, one line removed
- **AND** line 21 still reads "Installs the entire shll toolkit via Homebrew …" (X4-tier phrase, untouched)

### Branding: Skill prose product mentions

#### R2: Every present-tense product mention in kit skills reads HexoKit
Every bucket-A occurrence of the product noun `run-kit` / `run-kit's` in `src/kit/skills/*.md` — 31 prose mentions across `fab-operator.md` (10), `_cli-fab-pane.md` (12), `_cli-external.md` (3), `_cli-fab-operator.md` (5), `_cli-agents.md` (1), plus the `_cli-external.md` frontmatter description — MUST read `HexoKit` / `HexoKit's`. After apply, `grep -rn -i 'run-kit' src/kit` MUST list exactly **11** survivors, all in buckets C/D/E:

| File | Survivors | Bucket |
|------|-----------|--------|
| `src/kit/skills/fab-operator.md` | `brew install sahil87/tap/run-kit` (§2 gate line); "run-kit #913" (§ narrative example text); "github-pr · run-kit" ×2 (fleet-table rows); "Tell me when run-kit #913 merges" (§ track-add example) | C ×5 |
| `src/kit/skills/_cli-fab-pane.md` | "the one carrying run-kit PR #755" | C |
| `src/kit/skills/_cli-fab-pane.md` | "run-kit once joined pane state by `session:window_index`" | D |
| `src/kit/skills/_cli-external.md` | "The Homebrew formula and primary binary are `run-kit` (`sahil87/tap/run-kit`)" | C ×2 |
| `src/kit/migrations/2.13.6-to-2.14.0.md` | lines 22 and 124 | E ×2 |

- **GIVEN** the 53 `run-kit` occurrences in `src/kit` at `c3c6a5d8`
- **WHEN** apply completes
- **THEN** `grep -rn -i 'run-kit' src/kit | wc -l` prints `11` and every survivor matches a row in the table above
- **AND** the operator §2 gate line reads exactly ``Error: the operator requires HexoKit — brew install sahil87/tap/run-kit``
- **AND** no bucket-C/D/E text changed (`git diff` shows no hunk touching those fragments)

#### R3: Substrate identifiers are byte-identical
No `rk`-prefixed identifier changes: `rk <verb>` invocations, `@rk_*` tmux options, `rk-*` names, `RK_*` env vars, `rk skill`, `rk notify`. The per-pattern counts over `src/kit` MUST equal the pre-apply baseline.

- **GIVEN** a pre-apply baseline `grep -rhoE '@rk_[a-z_]+|\brk (mux|pane|cron|notify|skill|operator|agent|context|role)\b|\brk-[a-z]+' src/kit | sort | uniq -c` saved to the scratchpad
- **WHEN** the same command runs after apply
- **THEN** the two outputs are identical (`diff` exits 0)

#### R4: The two product-named section headings rename and no pointer goes stale
`_preamble.md` `## Run-Kit (rk) Reference` MUST become `## HexoKit (rk) Reference` (heading + TOC entry). `_cli-external.md` `## rk (run-kit)` MUST become `## rk (HexoKit)` (heading + TOC entry + frontmatter description). Every `§` pointer to either heading in `src/kit/skills/` and `docs/specs/skills.md` MUST name the new heading exactly: `fab-operator.md` (lines 86, 96 → `§ HexoKit (rk) Reference`; line 419 → `§ rk (HexoKit)`), `_cli-agents.md:154`, `code-dedupe.md:103` (currently the imprecise `§ Run-Kit Reference` — normalise to `§ HexoKit (rk) Reference`), `_preamble.md:187` (→ `§ rk (HexoKit)`), `docs/specs/skills.md:40`. Memory pointer sites are hydrate's (intake § Affected Memory) — apply does not edit `docs/memory/`.

- **GIVEN** the headings and pointer sites above
- **WHEN** apply completes
- **THEN** `grep -rn 'Run-Kit\|rk (run-kit)' src/kit docs/specs/skills.md` prints nothing
- **AND** `grep -c 'HexoKit (rk) Reference' src/kit/skills/_preamble.md` prints `2` (TOC + heading) and `grep -c 'rk (HexoKit)' src/kit/skills/_cli-external.md` prints `3` (description + TOC + heading)
- **AND** `docs/specs/skills.md:40` reads "(§ Naming Conventions and § HexoKit (rk) Reference respectively)"

#### R5: The `fab skill` bundle flips and its Go mirror stays identical
`docs/site/skill.md:81` MUST read `- **\`rk\` (HexoKit)** — fab is a pure *consumer* of HexoKit's \`@rk_pane_agent_state\` tmux` (two tokens flip; line 71's "shll toolkit" phrase is untouched). `scripts/sync-skill.sh` MUST be run so `src/go/fab/cmd/fab/skill.md` is byte-identical, and the bundle tests MUST pass.

- **GIVEN** `docs/site/skill.md` and its embedded mirror identical at `c3c6a5d8`
- **WHEN** the line is edited and `scripts/sync-skill.sh` runs
- **THEN** `diff -q docs/site/skill.md src/go/fab/cmd/fab/skill.md` reports no difference
- **AND** `go test ./src/go/fab/cmd/fab/ -run 'TestSkillBundle'` passes (line budget, static-only, drift guard)

#### R6: Only canonical sources change
The diff MUST touch no path under `.agents/` or `.claude/` (deployed copies — Constitution V), no `src/go/**/*.go` file, no `docs/memory/` file (hydrate-owned), and no `src/kit/migrations/` file. The Constitution V portability guard MUST stay green.

- **GIVEN** the apply diff
- **WHEN** `git diff --name-only` runs
- **THEN** every path is one of: `README.md`, `src/kit/skills/{fab-operator,_cli-fab-pane,_cli-external,_cli-fab-operator,_cli-agents,_preamble,code-dedupe}.md`, `docs/site/skill.md`, `src/go/fab/cmd/fab/skill.md`, `docs/specs/skills.md`, or `fab/changes/260911-y50t-hexokit-banner-sweep/*`
- **AND** `go test ./src/go/fab-kit/cmd/fab/` passes

### Non-Goals

- Go source comments and `--help` strings (43 `run-kit` mentions in `src/go/**`) — no command surface changes; a Go-side prose pass is a separate follow-up candidate.
- `docs/memory/` narrative beyond the anchor and verbatim-quote sites in intake § Affected Memory — hydrate-owned, and deliberately narrow (plan D11; run-kit's X3 row owns the eventual memory sweep).
- `docs/specs/` beyond `skills.md:40`'s anchor — `harness-adapters.md`, `hooks.md`, the SRAD rationale's data tables keep "run-kit".
- Every "shll toolkit" / shll.ai phrase (README:21, `docs/site/skill.md:71`, `docs/site/install.md`) — Phase 2 / X4.
- Repo links, formula names, PR numbers (`sahil87/run-kit`, `sahil87/tap/run-kit`, "#913", "PR #755") — R1/R2 (deferred Phase 3).
- Any `rk`/`@rk_`/`rk-`/`RK_` identifier — plan D2, permanent.
- `src/kit/migrations/2.13.6-to-2.14.0.md` — a frozen per-version artifact.
- Updating the plan doc's C7 row in the run-kit repo — done by hand after ship.

### Design Decisions

#### Five-bucket classification governs which `run-kit` tokens flip
**Decision**: Every `run-kit` occurrence in kit content is classified before editing into A (present-tense product noun → HexoKit), B (`rk`-prefixed substrate → untouched), C (repo / formula / PR identifier → untouched until the deferred renames), D (past-tense narrative → untouched), or E (frozen migration → untouched). Section headings that name the product are bucket A and rename with a same-change pointer sweep.
**Why**: The rebrand renames the product, not the substrate (plan D2) and not history (D11); GitHub, Homebrew, and PR identifiers are external objects that keep their names until R1/R2. A search-and-replace would corrupt all three classes, and leaving the headings would keep "Run-Kit" in the always-loaded preamble after the product stopped being called that.
**Rejected**: (a) flip every `run-kit` token — breaks formula install lines and repo-scoped PR references, and rewrites history; (b) flip nothing but the README — leaves deployed skills naming a product that no longer exists under that name, propagating into customer repos via `fab sync`; (c) rename headings without sweeping pointers — leaves dead `§` anchors, a review must-fix.
*Introduced by*: 260911-y50t-hexokit-banner-sweep

#### Formula and binary literals stay `run-kit` inside product-named prose
**Decision**: Where a sentence mixes the product noun with a still-current identifier — the §2 gate line ``requires HexoKit — brew install sahil87/tap/run-kit``, `_cli-external.md`'s "The Homebrew formula and primary binary are `run-kit` (`sahil87/tap/run-kit`)" — the noun flips and the identifier stays, even though the result names two things on one line.
**Why**: Both identifiers are true today (formula renames at R1, long binary at C3) and are typed by users; changing them ahead of the rename would break the install line. The plan accepts this "two names on one machine" gap as cosmetic.
**Rejected**: Rewording the identifier sentence away — it carries the one fact an agent needs to install the substrate.
*Introduced by*: 260911-y50t-hexokit-banner-sweep

## Tasks

### Phase 1: Setup

- [x] T001 Snapshot the substrate baseline: run `grep -rhoE '@rk_[a-z_]+|\brk (mux|pane|cron|notify|skill|operator|agent|context|role)\b|\brk-[a-z]+' src/kit | sort | uniq -c > /tmp/claude-1001/-home-sahil-code-sahil87-fab-kit-worktrees-ivory-grove/1c8a908a-41ce-4f5a-a9aa-a38097063ad3/scratchpad/rk-baseline.txt` (create the directory if needed), and re-derive the full occurrence list with `grep -rn -i 'run-kit' src/kit > .../run-kit-before.txt`; confirm 53 lines and reconcile against intake § Classification before editing anything <!-- R3 -->

### Phase 2: Core Implementation

- [x] T002 [P] Edit `README.md` line 3 to exactly `> Part of [HexoKit](https://hexokit.com) — see all projects there.` (em-dash); touch nothing else in the file <!-- R1 -->
- [x] T003 [P] Edit `src/kit/skills/fab-operator.md`: flip the 10 bucket-A product mentions (intake § 2 table — §1 cadence line, §2 "capable run-kit is on PATH", §2 "notifications are run-kit", §2 gate line noun only, §2 role-mark "run-kit's dashboard", §4 clock paragraph ×2, §5 exclusion rule "run-kit derives the roles", §7 merge-all "no run-kit follow-up", §9 "Requires run-kit?"); update the three pointers (two `§ Run-Kit (rk) Reference` → `§ HexoKit (rk) Reference`, one `§ rk (run-kit)` → `§ rk (HexoKit)`); leave the 5 bucket-C survivors (`sahil87/tap/run-kit`, "run-kit #913" ×2, "github-pr · run-kit" ×2) byte-identical <!-- R2, R4 -->
- [x] T004 [P] Edit `src/kit/skills/_cli-fab-pane.md`: flip the 12 bucket-A mentions (§ dispatch-internal verbs "run-kit's substrate twins"; § agent state ×6 — "written by run-kit's", "follows run-kit's `@rk_<scope>_<name>` scheme", "run-kit dual-writes", "written by run-kit from the first release after v3.18.7", "after run-kit removes its own legacy reads", "needs no run-kit software installed"; § enumeration "delegated to run-kit"; § identity-key contract ×2; `fab pane ready` and `fab dispatch ready` "sentinel-capable run-kit" ×2); leave "the one carrying run-kit PR #755" (C) and "run-kit once joined pane state" (D) byte-identical <!-- R2 -->
- [x] T005 [P] Edit `src/kit/skills/_cli-external.md`: frontmatter description "rk (run-kit)" → "rk (HexoKit)"; TOC entry and `## rk (run-kit)` heading → `rk (HexoKit)`; flip "run-kit is the tmux session manager …" → "HexoKit is …", "relying on run-kit's fail-silent-by-contract guarantee", and "pins the operator window in run-kit's dashboard"; leave the sentence "The Homebrew formula and primary binary are `run-kit` (`sahil87/tap/run-kit`); `rk` is the invocation used throughout fab skills." verbatim <!-- R2, R4 -->
- [x] T006 [P] Edit `src/kit/skills/_cli-fab-operator.md` (§ fab operator "capable run-kit is on PATH"; § tick-start state-path paragraph ×4 — "run-kit mirrors the exact slug rule", "pinned in run-kit's operator-cron spec", "coordinated run-kit change" ×2) and `src/kit/skills/_cli-agents.md` (state-writer caveat "the writer is run-kit's `rk agent setup`" → HexoKit's; pointer `§ Run-Kit (rk) Reference` → `§ HexoKit (rk) Reference`) <!-- R2, R4 -->
- [x] T007 [P] Edit `src/kit/skills/_preamble.md` (TOC entry and `## Run-Kit (rk) Reference` heading → `HexoKit (rk) Reference`; the in-section pointer "`_cli-external.md` § rk (run-kit)" → `§ rk (HexoKit)`) and `src/kit/skills/code-dedupe.md` (pointer "`_preamble.md` § Run-Kit Reference" → "`_preamble.md` § HexoKit (rk) Reference") <!-- R4 -->
- [x] T008 [P] Edit `docs/specs/skills.md` line 40: "(§ Naming Conventions and § Run-Kit (rk) Reference respectively)" → "(§ Naming Conventions and § HexoKit (rk) Reference respectively)" <!-- R4 -->
- [x] T009 Edit `docs/site/skill.md` line 81 (`**\`rk\` (run-kit)**` → `**\`rk\` (HexoKit)**`; "consumer of run-kit's" → "consumer of HexoKit's"), then run `scripts/sync-skill.sh` and `go test ./src/go/fab/cmd/fab/ -run 'TestSkillBundle' -count=1` <!-- R5 -->

### Phase 3: Integration & Edge Cases

- [x] T010 Verify: (a) `grep -rn -i 'run-kit' src/kit` prints exactly 11 lines matching R2's survivor table; (b) `grep -rn 'Run-Kit\|rk (run-kit)' src/kit docs/specs/skills.md` prints nothing; (c) re-run T001's substrate grep and `diff` against the baseline (must be identical); (d) `diff -q docs/site/skill.md src/go/fab/cmd/fab/skill.md`; (e) `git diff --name-only` contains no `.agents/`, `.claude/`, `docs/memory/`, `src/kit/migrations/`, or `*.go` path; (f) `go test ./src/go/fab-kit/cmd/fab/ -count=1` (Constitution V guard) passes; (g) `sed -n 1,5p README.md` shows H1 → new blockquote → blank → badges <!-- R1, R2, R3, R4, R5, R6 -->

## Execution Order

- T001 blocks everything (baseline must exist before edits)
- T002–T008 are independent and may run in parallel
- T009 is independent of T002–T008 but runs `go test`, so run it after the kit edits to keep one test pass
- T010 runs last

## Acceptance

### Functional Completeness

- [x] A-001 R1: `README.md` line 3 is exactly `> Part of [HexoKit](https://hexokit.com) — see all projects there.` and the README diff is a single-line change
- [x] A-002 R2: `grep -rn -i 'run-kit' src/kit` lists exactly 11 survivors, each matching a row of R2's survivor table (5 in `fab-operator.md`, 2 in `_cli-fab-pane.md`, 2 in `_cli-external.md`, 2 in the 2.13.6→2.14.0 migration)
- [x] A-003 R2: the operator §2 gate line reads ``Error: the operator requires HexoKit — brew install sahil87/tap/run-kit``
- [x] A-004 R3: the substrate-identifier count output is identical before and after apply
- [x] A-005 R4: `_preamble.md` heading and TOC read `HexoKit (rk) Reference`; `_cli-external.md` description, TOC, and heading read `rk (HexoKit)`
- [x] A-006 R4: `grep -rn 'Run-Kit\|rk (run-kit)' src/kit docs/specs/skills.md` is empty; every former pointer names the new heading exactly (incl. `code-dedupe.md`'s normalised `§ HexoKit (rk) Reference`)
- [x] A-007 R5: `docs/site/skill.md:81` reads `- **\`rk\` (HexoKit)** — fab is a pure *consumer* of HexoKit's \`@rk_pane_agent_state\` tmux` and the Go mirror is byte-identical
- [x] A-008 R6: the diff touches only the paths R6 enumerates

### Behavioral Correctness

- [x] A-009 R2: bucket-C/D/E fragments are byte-identical — "run-kit #913", "github-pr · run-kit", "run-kit PR #755", "run-kit once joined pane state", `sahil87/tap/run-kit`, the `_cli-external.md` formula/binary sentence, and both migration lines show no diff hunk
- [x] A-010 R1: README line 21 ("Installs the entire shll toolkit via Homebrew …"), `docs/site/skill.md:71`, and `docs/site/install.md` "shll toolkit" phrases are unchanged
- [x] A-011 R2: no "HexoKit" replaced a token inside a backticked `rk …` command, an `@rk_*` option, a URL, or a formula path (spot-check every `HexoKit` in the diff is prose)

### Scenario Coverage

- [x] A-012 R5: `go test ./src/go/fab/cmd/fab/ -run 'TestSkillBundle' -count=1` passes after the sync
- [x] A-013 R6: `go test ./src/go/fab-kit/cmd/fab/ -count=1` passes (no fab-kit-only path introduced into deployed content)
- [x] A-014 R1: README head order is `#` H1 → blockquote → blank line → contiguous badge run → prose (readme-extraction rule 1)

### Edge Cases & Error Handling

- [x] A-015 R2: the `_cli-fab-pane.md` § agent state sentence "written by HexoKit from the first release after v3.18.7 (the one carrying run-kit PR #755, its dual-read change)" mixes product and PR identifier exactly as designed — not over-flipped, not under-flipped
- [x] A-016 R4: `docs/memory/` is untouched by apply (its anchor sites are hydrate's — intake § Affected Memory lists them)

### Code Quality

- [x] A-017 Canonical source only: no edit under `.agents/skills/` or `.claude/skills/` (code-quality anti-pattern "Editing deployed skills directly")
- [x] A-018 Owner-or-pointer preserved: heading renames update pointers, never restate the pointed-at rule (code-quality "Stating an owned rule AND pointing at its owner")
- [x] A-019 Sibling sweep complete: every in-kit and spec pointer to the two renamed headings updated in this change (code-quality § Sibling Sweeps)
- [x] A-020 No fab-kit-only path cited from deployed content (code-quality "Citing fab-kit-only paths from deployed content"); the Go guard in A-013 is the mechanical check
- [x] A-021 Pattern consistency: edited sentences keep their surrounding style (possessive `HexoKit's` where `run-kit's` was; capitalisation `HexoKit` everywhere, never `Hexokit`/`hexokit` in prose)
- [x] A-022 No unnecessary duplication: no new restatement of the classification rule is added to a skill — it lives in this plan's Design Decisions and (via hydrate) in memory

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`
- Hydrate scope is intake § Affected Memory: four memory files, anchor text + the two verbatim gate-line quotes in `runtime/operator.md` + one Design Decisions entry (lift the first DD above). Do not sweep the other ~155 `run-kit` mentions in `docs/memory/`.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `_cli-external.md:180–181` (formula/binary sentence) stays verbatim; only line 179's lead noun flips | Resolves intake assumption 14: the sentence is a pure identifier statement, true today; the minimal edit is the safest | S:70 R:95 A:90 D:85 |
| 2 | Certain | `code-dedupe.md`'s imprecise `§ Run-Kit Reference` is normalised to the exact new heading `§ HexoKit (rk) Reference` rather than to a parallel `§ HexoKit Reference` | A pointer should name the heading exactly; fixing the drift while touching the line costs nothing | S:75 R:95 A:95 D:90 |
| 3 | Certain | Apply does not edit `docs/memory/`; the four anchor/quote files are hydrate's | Pipeline ownership: hydrate owns memory; intake § Affected Memory already enumerates the sites | S:80 R:90 A:95 D:90 |
| 4 | Confident | The substrate baseline regex (`@rk_*`, `rk <verb>`, `rk-*`) is a sufficient byte-identity proxy for R3 | The regex covers every substrate class the plan's D2 names that appears in kit prose; a wider `grep -c 'rk'` would be noisy with prose words | S:65 R:90 A:80 D:70 |
| 5 | Certain | "written by HexoKit from the first release after v3.18.7" keeps the version number unqualified | The version is the product's; no rename applies to version strings | S:70 R:95 A:90 D:90 |

5 assumptions (4 certain, 1 confident, 0 tentative).
