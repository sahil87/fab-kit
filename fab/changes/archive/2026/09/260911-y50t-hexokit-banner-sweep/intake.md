# Intake: HexoKit Banner Sweep (plan row C7, fab-kit)

**Change**: 260911-y50t-hexokit-banner-sweep
**Created**: 2026-09-11

## Origin

One-shot `/fab-new` invocation, driven by the cross-repo HexoKit rebrand plan (run-kit repo, `fab/plans/sahil/26-09-10-hexokit-rebrand.md`), row **C7** (`hexokit-banner-sweep`) applied to the **fab-kit** repo. C7 is one of six companion-repo sweeps (fab-kit, wt, idea, tu, hop, sahil87 profile) gated on **C1** — the shll repo's `hexokit-banner-and-policy` change (`ttoa`, PR [shll#98](https://github.com/sahil87/shll/pull/98), open draft, pipeline through review-pr done 2026-09-11, Copilot findings addressed). The gate condition as stated by the user (PR up, review-pr done) holds; the PR is not yet merged, so the *installed* `shll standards readme-extraction` still prints the old blockquote — the revised text below is taken verbatim from PR #98's diff.

> Per fab/plans/sahil/26-09-10-hexokit-rebrand.md row C7 (hexokit-banner-sweep), applied to the fab-kit repo, gated on C1 (shll change ttoa, PR shll#98 up, review-pr done): apply the readme-extraction standard's revised mandated blockquote to this repo's README (-> "Part of [HexoKit](https://hexokit.com) -- see all projects there"); flip PRESENT-TENSE "run-kit" PRODUCT mentions in fab-kit's skill prose (e.g. _cli-agents.md, fab-operator.md -- roughly 30 occurrences per the plan's own Risk note) to HexoKit. Substrate verbs and identifiers (`rk mux ...`, `rk pane ...`, `@rk_*`, any literal `rk`-prefixed command or option) are UNTOUCHED -- only the product-name prose changes. Repo links (github.com/sahil87/run-kit) stay as-is until the deferred R2 rename -- do not touch them. Read the plan doc's Decision log (D1-D14, esp. D2 substrate-stays and the fab-kit-prose-drift Risk) and row C7 in full first; run `shll standards` and read `readme-extraction` per the plan's pickup protocol before drafting the intake. Be careful and precise -- this is a careful present-tense-only pass, not a search-and-replace.

**Pickup protocol followed**: the plan doc was read in full (Decision log D1–D14, Naming tiers, row C7, the "fab-kit skill prose drift" Risk, Pickup protocol). `shll standards` run; `readme-extraction` read in full. Plan decisions D1–D4, D13, D14 are Confirmed (Certain, not re-opened); D11 (historical text is not renamed) is Proposed and is applied here as the plan's stated policy. Every `run-kit` occurrence in `src/kit/`, `README.md`, `docs/site/`, `src/go/`, `docs/specs/`, and `docs/memory/` was enumerated and classified before writing this intake (see § What Changes → § Classification).

## Why

**The problem.** The product fab-kit's operator and pane machinery sits on is being renamed from run-kit to **HexoKit** (plan Approach B: product rename, `rk` substrate kept — D1/D2). hexokit.com is live. fab-kit's skill prose names the *product* "run-kit" in present-tense sentences **31 times** across six skill files — "takes its cadence from run-kit's operator-tick cron entry", "when a capable run-kit is on PATH", "the operator requires run-kit" — and its README carries the toolkit banner the `readme-extraction` standard mandates as the first line under the H1, which C1 revises to name HexoKit. The plan's own Risk register calls this out: *"fab-kit's `_cli-agents.md`/`fab-operator.md` mention 'run-kit' as the product ~30×; substrate verbs (`rk mux …`) are correct and stay. C7 needs a careful present-tense-only pass."*

**What happens if we don't.** Once C3 (run-kit's own brand surfaces) lands and the site cuts over (X1/X2), every deployed fab skill keeps telling agents and readers about a product called "run-kit" that no longer exists under that name — deployed skills ship into customer repos via `fab sync`, so the stale name propagates. The README banner would be the one toolkit repo still pointing at the shll.ai brand after C1 — reintroducing the two-brand split the rebrand corrects, for a saving of one line. The plan sequences X1 (site cutover prep) on "C3, C7 merged", so this change is on the critical path to the announce.

**Why this shape.** A literal search-and-replace is wrong three ways: (1) the `rk` substrate — every `rk <verb>`, `@rk_*` option, `rk-*` name — is *correct and permanent* (D2); (2) repo and formula identifiers (`sahil87/run-kit`, `sahil87/tap/run-kit`, "run-kit #913", "run-kit PR #755") name GitHub/Homebrew objects that keep their names until the deferred Phase 3 renames (R1/R2); (3) past-tense narrative ("run-kit once joined pane state by …") and versioned migration files are historical text (D11). So the pass is **classify-then-edit**: every occurrence is placed in one of five buckets (§ Classification) and only the present-tense product bucket flips. Two section headings *are* product names (`## Run-Kit (rk) Reference`, `## rk (run-kit)`) and are pointer targets from other skills, one spec, and four memory files — renaming them requires the whole pointer class swept in the same change (code-quality.md § Sibling Sweeps), which is why the scope reaches into `docs/specs/skills.md` and four `docs/memory/` files *for anchor and verbatim-quote sites only*.

## What Changes

### 1. README banner (rule 1 of `readme-extraction`)

`README.md` line 3 — the canonical toolkit blockquote — changes to the exact line C1 mandates (byte-identical to PR shll#98's `docs/site/standards/readme-extraction.md` diff, em-dash included):

```markdown
# before
> Part of the [shll toolkit](https://shll.ai) — see all projects there.

# after
> Part of [HexoKit](https://hexokit.com) — see all projects there.
```

Nothing else in the README changes. Line 21 ("Installs the entire shll toolkit via Homebrew …") is an X4-tier "shll toolkit" phrase the plan defers to Phase 2 (D14 second pass; C1 explicitly left "every other shll.ai / 'shll toolkit' mention for X4") — untouched. The README head order (`#` H1 → blockquote → badge run → prose) is already conformant and is preserved. No Go test or CI step pins the blockquote text (verified: `grep 'Part of the\|shll toolkit' src/go .github scripts` → only the `skill.md` mirror's line 71, which is not the banner).

### 2. Skill prose: present-tense product mentions → HexoKit

Canonical sources under `src/kit/skills/` only (Constitution V; never the deployed `.agents/skills/` or `.claude/skills/` copies). The word "HexoKit" replaces the product noun "run-kit" (and its possessive "run-kit's" → "HexoKit's") in the sentences listed. Line numbers are as of `c3c6a5d8` (v2.25.1).

**`fab-operator.md`** — 10 product flips + 3 anchor updates:

| Line | Current fragment | Action |
|------|------------------|--------|
| 27 | "takes its cadence from run-kit's operator-tick cron entry" | → HexoKit's |
| 29 | "When a capable run-kit is on PATH" | → HexoKit |
| 74 | "…spawn readiness, and notifications are run-kit." | → HexoKit |
| 83 | ``Error: the operator requires run-kit — brew install sahil87/tap/run-kit`` | → `Error: the operator requires HexoKit — brew install sahil87/tap/run-kit` (product noun flips; the **formula literal `sahil87/tap/run-kit` stays** — R1) |
| 90 | "for run-kit's dashboard" | → HexoKit's |
| 171 | "it is run-kit substrate the skill documents"; "run-kit's operator-cron spec is the entry's design authority" | → HexoKit (×2) |
| 480 | "run-kit derives the roles from its own reserved constants" | → HexoKit |
| 751 | "no gap and no run-kit follow-up" | → HexoKit |
| 817 | "Requires run-kit? \| Yes" | → HexoKit |
| 86, 96 | pointers "`_preamble.md` § Run-Kit (rk) Reference" | → § HexoKit (rk) Reference (see § 3) |
| 419 | pointer "`_cli-external.md` § rk (run-kit)" | → § rk (HexoKit) (see § 3) |

**`_cli-fab-pane.md`** — 12 product flips:

| Line | Fragments | Action |
|------|-----------|--------|
| 24 | "rides run-kit's substrate twins `rk mux capture`…" | → HexoKit's (the `rk mux …` verbs stay) |
| 30 | "written by run-kit's `rk agent setup`"; "follows run-kit's `@rk_<scope>_<name>` scheme"; "run-kit dual-writes both"; "written by run-kit from the first release after v3.18.7"; "after run-kit removes its own legacy reads"; "needs no run-kit software installed" | → HexoKit (×6). **"the one carrying run-kit PR #755" stays** (repo/PR identifier) |
| 36 | "Enumeration is delegated to run-kit when present" | → HexoKit |
| 57 | "inherits run-kit's contract"; "(a run-kit companion item)" | → HexoKit (×2) |
| 94 | "when a sentinel-capable run-kit is on PATH" | → HexoKit |
| 222 | "when a sentinel-capable run-kit is on PATH" | → HexoKit |

**`_cli-external.md`** — description, TOC, heading, and 3 prose flips:

| Line | Current | Action |
|------|---------|--------|
| 3 | frontmatter description "…and rk (run-kit)." | → "rk (HexoKit)" |
| 20 | TOC entry "- rk (run-kit)" | → "- rk (HexoKit)" |
| 177 | `## rk (run-kit)` | → `## rk (HexoKit)` (heading rename — § 3) |
| 179 | "run-kit is the tmux session manager with a web UI that may host the operator's session." | → "HexoKit is the tmux session manager …" |
| 180–181 | "The Homebrew formula and primary binary are `run-kit` (`sahil87/tap/run-kit`); `rk` is the invocation used throughout fab skills." | **Stays as written** — both backticked tokens are literal formula/binary identifiers that are still `run-kit` today (formula → R1; long binary → C3, not yet landed). Apply MAY reword the lead-in ("Its Homebrew formula and long binary are …") for flow but MUST NOT change the identifiers |
| 191 | "relying on run-kit's fail-silent-by-contract guarantee" | → HexoKit's |
| 201 | "pins the operator window in run-kit's dashboard" | → HexoKit's |

**`_cli-fab-operator.md`** — 5 product flips: line 28 "when a **capable** run-kit is on PATH" → HexoKit; line 152 "run-kit mirrors the exact slug rule … pinned in run-kit's operator-cron spec … requires a coordinated run-kit change … needs no coordinated run-kit change" → HexoKit (×4).

**`_cli-agents.md`** — 1 product flip + 1 anchor: line 144 "the writer is run-kit's `rk agent setup` global agent-harness hooks" → HexoKit's (`rk agent setup` stays); line 154 pointer "§ Run-Kit (rk) Reference" → § HexoKit (rk) Reference.

**`_preamble.md`** — heading rename + TOC + 1 pointer (§ 3): line 20 TOC "Run-Kit (rk) Reference", line 172 `## Run-Kit (rk) Reference`, line 187 pointer "`_cli-external.md` § rk (run-kit)".

**`code-dedupe.md`** — 1 anchor: line 103 "`_preamble.md` § Run-Kit Reference" (already an imprecise spelling of the heading) → "`_preamble.md` § HexoKit (rk) Reference" (exact new heading).

### 3. Heading renames and the pointer sweep

Two section headings carry the product name and are cited by pointer from other files. Both rename; every pointer site updates in the same change so no `§` reference goes stale:

| Heading | File | New heading | Pointer sites to update |
|---------|------|-------------|-------------------------|
| `## Run-Kit (rk) Reference` | `_preamble.md:172` (+ TOC :20) | `## HexoKit (rk) Reference` | `fab-operator.md:86,96`; `_cli-agents.md:154`; `code-dedupe.md:103`; `docs/specs/skills.md:40`; `docs/memory/_shared/context-loading.md:33,285`; `docs/memory/distribution/kit-architecture.md:345,357`; `docs/memory/pipeline/code-dedupe.md:35`; `docs/memory/runtime/operator.md:35,188,548` |
| `## rk (run-kit)` | `_cli-external.md:177` (+ TOC :20) | `## rk (HexoKit)` | `_preamble.md:187`; `fab-operator.md:419`; `docs/memory/runtime/operator.md:188` |

The memory sites above are edited **only at the anchor text** (and at the verbatim skill quote — see § 5); `log.md`/`log.seed.md` entries and `docs/specs/findings/*` that mention the old heading are historical records and stay. Verification: after apply, `grep -rn 'Run-Kit (rk) Reference\|Run-Kit Reference\|rk (run-kit)' src/kit docs/specs/skills.md docs/memory --include='*.md' | grep -v '/log\.\|/findings/'` returns nothing.

### 4. `docs/site/skill.md` (the `fab skill` bundle) and its Go mirror

`docs/site/skill.md:81` is skill prose served to agents by `fab skill` and a live `docs/site` surface (D11 names docs/site a live surface):

```markdown
# before
- **`rk` (run-kit)** — fab is a pure *consumer* of run-kit's `@rk_pane_agent_state` tmux

# after
- **`rk` (HexoKit)** — fab is a pure *consumer* of HexoKit's `@rk_pane_agent_state` tmux
```

Then run `scripts/sync-skill.sh` so `src/go/fab/cmd/fab/skill.md` (the embedded byte-copy; currently identical) follows, and run the bundle tests (`go test ./src/go/fab/cmd/fab/ -run 'TestSkillBundle'`) — the drift guard, ≤150-line budget, and static-only check must stay green. Line 71 of the same file ("fab is one member of the [shll toolkit](https://shll.ai)") is X4-tier — untouched.

### 5. Memory touch (hydrate scope — deliberately narrow)

`docs/memory/` names run-kit 162 times, overwhelmingly as present-truth narrative about the substrate contract. This change does **not** sweep that narrative (the user scope is skill prose + README; the plan's D11 keeps memory narrative as-is and its X3 row is the eventual memory sweep). Hydrate touches memory at exactly two kinds of site:

1. **Renamed-heading anchors** (§ 3 table) — 4 files.
2. **Verbatim quotes of skill text this change edits** — `docs/memory/runtime/operator.md:35` and `:548` quote the §2 gate line ``Error: the operator requires run-kit — brew install sahil87/tap/run-kit``; they follow the skill to `requires HexoKit`.

Plus the ordinary hydrate log entries. A design-decision entry recording the classification rule (§ Classification below) goes in `runtime/operator.md` § Design Decisions so the next reader knows why the file mixes "HexoKit" (product) with "run-kit" (repo/formula/history).

### Classification — the five buckets (the load-bearing rule for apply)

Every `run-kit` occurrence in `src/kit/`, `README.md`, and `docs/site/` was placed in exactly one bucket. Apply edits bucket **A** only and MUST leave B–E byte-identical:

| Bucket | Rule | Occurrences (kit + site) | Examples |
|--------|------|--------------------------|----------|
| **A. Present-tense product noun** | "run-kit" (or "run-kit's") used as the *product* in a present-tense statement of current behavior — including the two section headings, their TOC entries, the `_cli-external.md` frontmatter description, and every in-kit `§` pointer to those headings | **42** in `src/kit` = 31 prose flips + 2 headings + 2 TOC entries + 1 description + 6 pointer sites; plus `docs/site/skill.md:81` (×2) | "run-kit's operator-tick cron entry", "a capable run-kit is on PATH", "requires run-kit" |
| **B. Substrate identifier** | anything `rk`-prefixed: `rk <verb>`, `@rk_*`, `rk-*`, `RK_*`, `rk skill`, `rk notify` | all — hundreds; not counted by the `run-kit` grep | `rk mux send`, `@rk_pane_agent_state`, `rk agent setup` |
| **C. Repo / formula / PR identifier** | `sahil87/run-kit`, `github.com/sahil87/run-kit`, `sahil87/tap/run-kit`, the literal formula/binary names in `_cli-external.md:180–181`, "run-kit #913", "github-pr · run-kit", "run-kit PR #755" | **8** (`fab-operator.md:83` formula, `:274`, `:356`, `:357`, `:776`; `_cli-fab-pane.md:30` PR #755; `_cli-external.md:180`, `:181`) | `brew install sahil87/tap/run-kit`; "Tell me when run-kit #913 merges" (a PR in the `sahil87/run-kit` repo) |
| **D. Past-tense / historical narrative** | a sentence describing what the product *did* ("once joined …") | **1** (`_cli-fab-pane.md:48`) | "The motivating bug: run-kit once joined pane state by `session:window_index`" |
| **E. Versioned artifact** | migration files are frozen per-version instructions (D11) | **2** (`src/kit/migrations/2.13.6-to-2.14.0.md:22,124`) | "run-kit's `rk agent-setup` global agent-harness hooks" |

**Reconciliation to the grep** (`grep -rn -i 'run-kit' src/kit` at `c3c6a5d8` → 53 occurrences, 0 of them `github.com/sahil87/run-kit` links): A 42 + C 8 + D 1 + E 2 = 53. Per file — `fab-operator.md` 18 (13 A: 10 prose + 3 pointers; 5 C), `_cli-fab-pane.md` 14 (12 A; 1 C; 1 D), `_cli-external.md` 8 (6 A: 3 prose + heading + TOC + description; 2 C), `_cli-fab-operator.md` 5 (5 A), `_preamble.md` 3 (3 A: heading + TOC + pointer), `_cli-agents.md` 2 (2 A: 1 prose + 1 pointer), `code-dedupe.md` 1 (1 A pointer), `migrations/2.13.6-to-2.14.0.md` 2 (2 E). The 31 prose flips are the "~30" the plan's Risk note predicted. Apply MUST re-derive this table from a fresh grep before editing rather than trusting these line numbers, and MUST end with exactly 11 `run-kit` survivors in `src/kit` (8 C + 1 D + 2 E).

### Non-Goals (explicit — apply stays out of these)

- **Go source** (`src/go/**`, 43 occurrences: doc comments and `--help` strings such as "a sentinel-capable run-kit is on PATH" in `pane_ready.go`/`dispatch_ready.go`, and `operator_track_test.go`'s `sahil87/run-kit` fixture). No command surface changes here, so the CLI ⇒ docs constraint is not triggered; flipping help strings is a candidate follow-up backlog item, not part of C7.
- **`docs/memory/` narrative** beyond § 5's anchor and verbatim-quote sites (162 → ~6 edits). **`docs/specs/`** beyond `skills.md:40`'s anchor (20 occurrences, incl. `harness-adapters.md`, `hooks.md`, `srad-scoring-rationale-v1-to-v2.md`'s data tables).
- **"shll toolkit" / shll.ai phrases** anywhere (README line 21, `docs/site/skill.md:71`, `docs/site/install.md:12,19`) — X4 in Phase 2.
- **Repo links, formula names, GitHub rename** — R1/R2 (deferred Phase 3).
- **Any `rk`/`@rk_`/`rk-`/`RK_` identifier** — D2, permanent.
- **The migration file** — E.
- **Constitution / config** — no MUST rule, no `fab/project/` field changes; `fab/project/config.yaml`'s `name: fab-kit` is untouched.
- **Updating the plan doc's C7 row** — the plan lives in the run-kit repo; the pickup protocol's step 4 (fill in folder/PR, Status) is done there by hand after ship, outside this fab-kit change.

## Affected Memory

- `runtime/operator`: (modify) anchor text `§ Run-Kit (rk) Reference` → `§ HexoKit (rk) Reference` at lines 35, 188, 548 and `§ rk (run-kit)` → `§ rk (HexoKit)` at 188; the verbatim §2 gate line at 35 and 548 follows the skill to `requires HexoKit — brew install sahil87/tap/run-kit`; one Design Decisions entry recording the five-bucket classification rule (product → HexoKit; substrate/repo/formula/history → untouched). No other run-kit narrative in the file changes.
- `_shared/context-loading`: (modify) anchor text at lines 33 and 285 (`§ Run-Kit (rk) Reference` → `§ HexoKit (rk) Reference`); the lower-case "run-kit (rk) recipes" phrase at line 33 is a present-tense product mention in a sentence being edited anyway → "HexoKit (rk) recipes".
- `distribution/kit-architecture`: (modify) anchor text at lines 345 ("inlined Run-Kit (rk) Reference") and 357 (`## Run-Kit (rk) Reference`).
- `pipeline/code-dedupe`: (modify) anchor text at line 35 (`§ Run-Kit Reference` → `§ HexoKit (rk) Reference`).

## Impact

**Files edited** (canonical sources only):

| Area | Files | Edit shape |
|------|-------|------------|
| README | `README.md` | 1 line (the blockquote) |
| Kit skills | `src/kit/skills/fab-operator.md`, `_cli-fab-pane.md`, `_cli-external.md`, `_cli-fab-operator.md`, `_cli-agents.md`, `_preamble.md`, `code-dedupe.md` | 31 product-noun flips, 2 heading renames, 2 TOC entries, 1 frontmatter description, 6 in-kit pointer updates |
| Skill bundle | `docs/site/skill.md` → `scripts/sync-skill.sh` → `src/go/fab/cmd/fab/skill.md` | 1 line (×2 tokens), mirrored |
| Specs | `docs/specs/skills.md` | 1 anchor |
| Memory (hydrate) | 4 files per § Affected Memory | anchors + 2 verbatim quotes + 1 DD entry |

**No behavior change.** No `fab` command signature, flag, exit code, or skill flow changes; no template, migration, or config change; no version bump needed beyond the normal release cadence. `fab sync` will redeploy the edited skills into `.agents/skills/`/`.claude/skills/` on the next sync.

**Tests to run**: `go test ./src/go/fab/cmd/fab/ -run 'TestSkillBundle'` (bundle drift guard / line budget / static-only after the mirror sync) and the Constitution V portability guard in the same package (`go test ./src/go/fab-kit/cmd/fab/` — no new paths are cited, so it must stay green). Then the mechanical verifications: (a) `grep -rn -i 'run-kit' src/kit README.md docs/site` shows only buckets C/D/E — expected survivors: `fab-operator.md` ×5 (83 formula, 274, 356, 357, 776), `_cli-fab-pane.md` ×2 (30 PR #755, 48), `_cli-external.md` ×2 (180, 181), `migrations/2.13.6-to-2.14.0.md` ×2; (b) `grep -rn 'Run-Kit\|rk (run-kit)' src/kit docs/specs/skills.md docs/memory | grep -v '/log\.\|/findings/'` is empty; (c) `grep -c '@rk_\|rk mux\|rk pane\|rk cron\|rk notify\|rk skill\|rk operator\|rk agent' -r src/kit` is unchanged from before apply (substrate byte-identical); (d) `diff -q docs/site/skill.md src/go/fab/cmd/fab/skill.md` reports identical; (e) `git diff --stat` touches no `.agents/`/`.claude/` path.

**Readme-extraction conformance check** (standard § Verifying conformance): head order `#` H1 → blockquote → badges unchanged; no relative image or link introduced; the blockquote is the one line the standard's revised rule 1 mandates.

**Downstream**: X1 (`hexokit-site-cutover-prep`) depends on C3 and C7 merged; the plan doc's C7 row (run-kit repo) needs its PR link and Status filled in by hand after ship.

## Open Questions

- None blocking. The exact rewording of `_cli-external.md:179–181`'s lead-in sentence (keeping the `run-kit` formula/binary literals) is left to apply within the constraint stated in § 2.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Product name is **HexoKit**; `rk` binary and every `RK_*`/`@rk_*`/`rk-*` identifier stay (plan D1/D2, Confirmed) | Plan pickup protocol: D1–D4, D13, D14 are Certain and not re-opened; user restated D2 in the prompt | S:95 R:90 A:100 D:100 |
| 2 | Certain | README blockquote becomes exactly `> Part of [HexoKit](https://hexokit.com) — see all projects there.` (em-dash, link on "HexoKit" only) | Taken byte-for-byte from PR shll#98's `readme-extraction.md` diff (D14 first pass, Confirmed); user's `--` is an ASCII rendering of the standard's em-dash | S:95 R:95 A:100 D:95 |
| 3 | Certain | Edit canonical `src/kit/skills/*.md` only, never `.agents/skills/` or `.claude/skills/` | Constitution V + code-quality anti-pattern | S:90 R:95 A:100 D:100 |
| 4 | Certain | Repo/formula/PR identifiers stay: `sahil87/run-kit`, `sahil87/tap/run-kit`, "run-kit #913", "github-pr · run-kit", "run-kit PR #755", the literal binary name in `_cli-external.md:180` | User: "Repo links stay as-is until R2"; plan R1/R2 defer formula and repo renames; a PR number is a repo-scoped identifier | S:90 R:90 A:95 D:90 |
| 5 | Confident | Rename the two product-named section headings (`## Run-Kit (rk) Reference` → `## HexoKit (rk) Reference`; `## rk (run-kit)` → `## rk (HexoKit)`) and sweep every pointer site in kit skills, `docs/specs/skills.md`, and the four memory files in this same change | A heading is a present-tense product mention; leaving it keeps "Run-Kit" in the always-loaded preamble. Pointer sweep is mandatory once renamed (code-quality § Sibling Sweeps; review treats a stale `§` as must-fix). Reversible with one grep | S:70 R:65 A:90 D:75 |
| 6 | Certain | `docs/site/skill.md:81` (the `fab skill` bundle) is in scope; mirror via `scripts/sync-skill.sh`; run the bundle tests | It is skill prose served to agents and a live docs/site surface (D11); its Go twin is a byte-copy with a drift-guard test | S:65 R:90 A:90 D:80 |
| 7 | Certain | The operator gate string becomes `Error: the operator requires HexoKit — brew install sahil87/tap/run-kit` — product noun flips, formula literal stays | The string is skill-owned (no Go twin — verified by grep); mixing product name and current formula is exactly the plan's accepted "two names on one machine" gap until R1 | S:75 R:90 A:90 D:80 |
| 8 | Certain | Memory is touched only at renamed-heading anchors and verbatim quotes of edited skill text (4 files, ~7 sites); the other ~155 `run-kit` memory mentions stay | User scope is skill prose + README; plan D11 keeps memory narrative and schedules the eventual sweep (X3); anchors/quotes must follow or they are dead pointers / stale quotes | S:70 R:85 A:85 D:75 |
| 9 | Certain | Past-tense narrative stays: `_cli-fab-pane.md:48` "run-kit once joined pane state …" | User: "present-tense only"; the sentence narrates a historical bug in the product's past (D11) | S:80 R:95 A:85 D:75 |
| 10 | Certain | `src/kit/migrations/2.13.6-to-2.14.0.md` (2 mentions) stays | Migrations are frozen per-version instruction files — versioned historical artifacts under D11; editing a shipped migration's prose has no reader | S:70 R:95 A:90 D:80 |
| 11 | Certain | Go source comments and `--help` strings (43 mentions) are out of scope; noted as a follow-up candidate | User scope is skill prose; no command surface changes so the CLI ⇒ docs rule is not triggered; the help-text/reference-doc mismatch ("run-kit" vs "HexoKit" in the sentinel-capable sentence) is cosmetic until a Go-side pass | S:70 R:90 A:85 D:70 |
| 12 | Certain | "shll toolkit"/shll.ai phrases outside the blockquote (README:21, `docs/site/skill.md:71`, `docs/site/install.md`) stay | Plan D14 second pass (X4) and C1's own scope statement leave these for Phase 2 | S:80 R:95 A:90 D:85 |
| 13 | Certain | Change type `docs` | Prose-only edits to skills, README, one spec line, and memory; no behavior or command change | S:75 R:95 A:90 D:80 |
| 14 | Confident | `_cli-external.md:179–181` — flip the lead noun to HexoKit, keep the sentence about formula and binary literally `run-kit`, and let apply pick the connective wording | Two valid shapes (leave the identifier sentence verbatim vs. reword the lead-in "Its Homebrew formula and long binary are …"); either is correct today; trivially reversible | S:55 R:90 A:75 D:45 |
| 15 | Certain | The gate C1 is treated as satisfied (PR shll#98 up, review-pr done) even though unmerged; the revised blockquote is sourced from the PR diff, not the installed standard | User stated the gate condition in the prompt; the plan's order is C1 → (C3 ∥ C7) with C1 "PR open (draft)" as its recorded status | S:85 R:80 A:80 D:80 |

15 assumptions (13 certain, 2 confident, 0 tentative, 0 unresolved).
