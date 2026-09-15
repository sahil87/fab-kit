# Intake: Deployed Kit Content Must Not Cite fab-kit-Only Paths

**Change**: 260910-d5tk-deployed-skills-no-fabkit-paths
**Created**: 2026-09-10

## Origin

Conversational, dispatched promptless via `/fab-proceed` (`_intake` with `{questioning-mode} = promptless-defer`) from a user conversation that started with a **customer-repo bug report**. A session running fab-kit's deployed skills inside a customer project reported:

> fab-kit's generated skills reference docs/specs/change-types.md, cron.md, harness-adapters.md, etc. Those are fab-kit's own docs paths bleeding into its skill prose — a fab-kit bug, not fixable here.

The user's decision, verbatim:

> fab-kit's skills that are deployed into customer's repos, can't be referencing files within the fab-kit repo (how did this get missed — add this to the constitution.)

Key decisions reached in the conversation (all carried into `## Assumptions`):

- **Bundle, not a bare constitution edit.** The user chose one change that amends the constitution AND sweeps the offending sites AND adds a test guard, because amending alone would leave the repo violating its own new rule on the day it lands.
- **Category B convention paths stay.** `docs/memory/index.md`, `docs/specs/index.md`, `docs/memory/_shared/removed-domains.md`, `docs/memory/_shared/utilities.md` describe the *customer's* docs layout and are legitimate. Making them config-derived from `docs_index.roots` is explicitly a Non-Goal.
- **No CLI surface change.** The guard is a plain Go test, never a `fab` subcommand, so `_cli-fab.md` needs no signature update.
- **Edit only `src/kit/`** — never the deployed copies (Constitution V).

## Why

**The pain point.** `fab sync` deploys `src/kit/skills/*.md` into every customer's `.agents/skills/` and `.claude/skills/`. Thirteen sites across seven skill files cite files that exist only inside the fab-kit repository — `docs/specs/stage-models.md`, `docs/specs/harness-adapters.md`, `docs/specs/config.md`, `docs/specs/change-types.md`, `docs/specs/naming.md`, `docs/memory/memory-docs/docs-index.md`, `docs/memory/runtime/dispatch.md` — as the *authority* or *rule owner* ("`docs/specs/config.md` owns the cascade/provenance rationale", "rule owned by `docs/memory/memory-docs/docs-index.md`", "See `docs/specs/change-types.md` for the full taxonomy"). In a customer repo every one of those paths is dead. An agent following the pointer either fails the read, or — worse — reads the *customer's* `docs/specs/config.md` if one happens to exist, and takes an unrelated project's document as fab's rule.

**The consequence of not fixing it.** The bleed is growing: the oldest instance landed in 97b12086 (#308, 2026-04-02), and further instances arrived in eca310c4 (#392), 3887cfb7 (#463), and 6d5cb5ea (#658, 2026-09-09 — one day before this intake). Nothing in the repo's review, consistency check, or hydrate machinery can catch it, because all three run inside fab-kit where every path resolves. Without a written rule and a mechanical guard, each future skill edit that follows the owner-or-pointer convention will keep pointing at fab-kit-internal owners.

**Root cause — three compounding causes (why it got missed):**

1. **The owner-or-pointer rule drove it.** `fab/project/code-quality.md` says a skill may state a rule it owns OR point at the file that owns it, never both. Inside fab-kit the genuine owner of, e.g., the three-adapter contract *is* `docs/specs/harness-adapters.md`, so authors correctly pointed there. Nobody drew the boundary that a **deployed** skill's pointer may only target things that **also deploy**.
2. **No test evaluates skill prose from the customer's vantage point.** `src/go/fab-kit/internal/skills_test.go` and `sync_integration_test.go` test deployment mechanics (which files land where), not content portability. Review and `/internal-consistency-check` run in the dev repo where the paths are live.
3. **Principle V (Portability) is about structure, not prose.** It forbids assumptions about the host project's *directory structure, language, or toolchain*; it says nothing about what a deployed file may *cite*. The very change that made Portability a principle (#308) introduced the first bleed.

**Why this approach over alternatives.**

- *Bare constitution amendment only* — rejected by the user: the repo would immediately violate its own rule on 13 lines, and the review machinery still could not see the violation.
- *Fix the 13 sites only, no rule* — rejected: the four-PR history shows the pattern regenerates whenever an author follows owner-or-pointer in good faith. The rule plus the carve-out in code-quality.md fix the generator; the test makes it mechanical.
- *Make the deployed skills' pointers resolve by shipping the specs into customer repos* — rejected implicitly by Constitution V and context.md § Distribution: "The kit content tree itself is NOT copied into user projects."

## What Changes

### 1. Constitution amendment — new MUST rule under Principle V (Portability)

Append a new paragraph to `### V. Portability` in `fab/project/constitution.md`. Proposed wording (apply MAY tighten phrasing but MUST keep every clause):

> Deployed kit content (`src/kit/skills/`, `src/kit/templates/`, `src/kit/migrations/`, `src/kit/scaffold/`, `src/kit/reference/`) MUST NOT reference files that exist only in the fab-kit repository. A deployed file MAY cite another kit skill or helper (e.g. `_cli-fab.md` § fab dispatch), a `fab` command, a `fab/project/` file, a `$(fab kit-path)/…` deployed asset, or a documented convention path in the host project (e.g. `docs/memory/index.md`). It MUST NOT cite fab-kit's own `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*` as an authority or rule owner. Rule ownership that lives in fab-kit's docs is carried into the deployed file by restating the operative rule, not by pointer; the fab-kit doc may then point at the skill.

Governance block updates:

- `**Version**: 1.7.0` → `**Version**: 1.8.0` (new normative MUST rule → minor bump, per the udwv/y8it precedent recorded in the trailing comments)
- `**Last Amended**: 2026-09-08` → `**Last Amended**: 2026-09-10`
- Append a dated HTML-comment governance note in the existing style, e.g.:

```html
<!-- 2026-09-10 (260910-d5tk): Added a deployed-content citation rule to Principle V —
     kit content that `fab sync` deploys MUST NOT cite fab-kit-only paths (docs/specs/*,
     docs/memory/*, docs/site/*, src/go/*) as an authority or rule owner; owned rules are
     restated in the deployed file, and the fab-kit doc points at the skill. Motivated by a
     customer-repo report of dead docs/specs/*.md pointers in deployed skills (13 sites, 7
     files, oldest from #308 — the change that introduced Principle V). New normative
     MUST-rule added → minor version bump 1.7.0 → 1.8.0. -->
```

### 2. `code-quality.md` carve-out + `code-review.md` must-fix rule

**`fab/project/code-quality.md` § Anti-Patterns (project-specific)** — the **"Stating an owned rule AND pointing at its owner"** bullet gains the exception, and a new sibling anti-pattern is added:

- Amend the existing bullet's tail: *"… Applies to every `src/kit/skills/*.md` edit. **Exception — fab-kit-only owners**: when the owner is a file that does not deploy (fab-kit's `docs/specs/*`, `docs/memory/*`, `docs/site/*`, `src/go/*`), the deployed skill **restates the operative rule** and becomes the owner for deployed purposes; the spec/memory doc then points at the skill (or both carry it, with the skill canonical). Pointing a deployed file at a non-deployed owner is not a pointer — it is a dead link (Constitution V)."*
- New bullet: **"Citing fab-kit-only paths from deployed content."** `src/kit/**` deploys into customer repos where fab-kit's `docs/specs/`, `docs/memory/`, `docs/site/`, and `src/go/` do not exist. Cite kit skills, `fab` commands, `fab/project/` files, `$(fab kit-path)/…` assets, or the host's convention paths (`docs/memory/index.md`, `docs/specs/index.md`, `docs/memory/_shared/removed-domains.md`, `docs/memory/_shared/utilities.md`) — never fab-kit's own docs (Constitution V). Guarded by the Go test in `src/go/fab-kit/cmd/fab/`.

**`fab/project/code-quality.md` § Sibling Sweeps** — the closing sentence *"Sweeps catch a diverged copy after the fact; the owner-or-pointer convention prevents that copy from existing at all."* gets a qualifying clause: *"… — within the deployed set; across the deploy boundary the skill is the owner."*

**`fab/project/code-review.md` § Project-Specific Review Rules** — add a must-fix row:

- **Deployed content cites fab-kit-only paths** — any `src/kit/**` file that names fab-kit's `docs/specs/*`, `docs/memory/*` (outside the host convention paths), `docs/site/*`, or `src/go/*` as an authority is a must-fix; restate the rule in the skill instead (Constitution V; the Go guard fails on it).

**Sibling sweep (repo-wide) of prose that documents the owner-or-pointer convention**, so every statement of the convention carries the boundary. Known sites from `grep -rniE 'owner-or-pointer|state a rule it owns|point at the file that owns'`:

- `src/kit/skills/internal-skill-optimize.md:56` (Sibling duplication row — "a file may state a rule it owns or point at the file that owns it — never both")
- `docs/specs/skills.md:1557` (`/internal-skill-optimize` Signals — "Ownership rule: a file may state a rule it owns or point at the owner, never both")
- `docs/memory/pipeline/planning-skills.md:37`, `docs/memory/pipeline/execution-skills.md:43`, `docs/memory/memory-docs/docs-index.md:180`, `docs/memory/distribution/setup.md:24` — mention the convention in passing; update only where a sentence would now be false (most cite skill→skill or wizard→`fab config explain` pointers, which stay valid).
- `src/kit/skills/fab-new.md:41`, `src/kit/skills/fab-proceed.md:157` — skill→skill pointers, valid; no edit expected.

### 3. Sweep the Category A sites in `src/kit/skills/`

Verified with `grep -rnE 'docs/(specs|memory|site)/[A-Za-z0-9_./-]+\.md' src/kit/skills/` on this branch (main at 89e264a3). For each site: **restate the operative rule inline** where the skill needs it, or **drop the citation** where it adds nothing beyond "the design lives elsewhere". Never leave a dangling fab-kit path.

| File:line | Cites | Disposition |
|-----------|-------|-------------|
| `_preamble.md:305` | `docs/specs/stage-models.md` "owns the design" | Drop the clause — `_cli-fab.md` § fab agent already owns precedence + schema for deployed purposes |
| `_preamble.md:320` | `docs/specs/stage-models.md` § Skill wiring | Drop "See …"; the operator-launcher exception is fully stated in the sentence before it |
| `_preamble.md:356` | `docs/specs/harness-adapters.md` "owns the three-adapter contract" | Rephrase: `_preamble.md` § CLI-Adapter Dispatch is the canonical procedure; `_cli-fab.md` § fab dispatch owns runtime details |
| `_preamble.md:450` | "Per `docs/specs/harness-adapters.md` § Dispatch-prompt obligations" | Drop the "Per …" attribution — the three obligations are already stated in full here (this section becomes the deployed owner) |
| `_preamble.md:481` | `docs/memory/runtime/dispatch.md` inside an example `summary:` string | Replace with a placeholder-shaped path, e.g. `docs/memory/{domain}/{file}.md` — illustrative only |
| `_preamble.md:534` | "See `docs/specs/change-types.md` for the full taxonomy" | Drop the sentence or repoint to `_cli-fab.md` § fab score / `fab status set-change-type` (the seven type names are already enumerated in `_cli-fab.md`) |
| `_cli-fab.md:212` | "`expected_min` (in `docs/specs/change-types.md`)" | Restate: "`expected_min` is documentation-only and not part of the score path" — drop the path |
| `_cli-fab.md:285` (×2) | `docs/specs/stage-models.md` § Default role profiles; "See `docs/specs/stage-models.md`" | Drop both; the sentence already directs readers to `fab agent … -o yaml` / `fab config explain providers --json` for the live values |
| `_cli-fab.md:381` | `docs/specs/stage-models.md` § Effort asymmetry | Drop "See …"; the asymmetry is restated in full in that paragraph |
| `_cli-fab.md:415` | "See `docs/specs/config.md` for the schema" | Repoint to `fab config explain` (the generated reference IS the deployed schema) |
| `_cli-fab.md:419` | "see § fab config upgrade and `docs/specs/config.md`" | Keep the § pointer, drop the spec |
| `_cli-fab.md:442` | "`docs/specs/config.md` owns the cascade/provenance rationale and history" | Restate the four-tier cascade one-liner (environment > system `~/.fab-kit/config.yaml` > project > built-in defaults; per-leaf deep merge, empty-skip) — the operative rule an agent needs — and drop the ownership claim |
| `_cli-fab.md:637` | "Full cross-adapter contract …: `docs/specs/harness-adapters.md`" | Repoint to `_preamble.md` § CLI-Adapter Dispatch + § Dispatch-Prompt Obligations |
| `_cli-fab.md:939,941` | fab-kit's own `docs/site/skill.md`, `src/go/fab/cmd/fab/skill.md`, `scripts/sync-skill.sh` | Trim the build-provenance prose (embed source, sync script, drift-guard test name) — a customer agent needs only the contract: raw markdown, byte-stable per release, stderr empty, exit 0. Name the shll toolkit `skill` standard in prose ("the shll toolkit-wide `skill` standard, published at shll.ai") — no bare `docs/site/…` path (Assumption 3) |
| `_cli-fab.md:1445` | "the same convention `/git-branch` and `docs/specs/naming.md` carry" | Drop `docs/specs/naming.md`; `/git-branch` + `_preamble.md` § Naming Conventions already own it for deployed purposes |
| `_cli-agents.md:97` | "contract in `docs/specs/harness-adapters.md`" | Repoint to `_preamble.md` § CLI-Adapter Dispatch |
| `fab-continue.md:217`, `docs-hydrate-memory.md:37`, `docs-distill-memory.md:61,146,205`, `docs-reorg-memory.md:244` | "rule owned by `docs/memory/memory-docs/docs-index.md`" (manual block) | Restate the operative rule ONCE in the deployed set — the manual block (`<!-- fab docs-index:manual:start -->` … `:end -->`) is the one hand-managed region of a generated primary landing, preserved verbatim by `fab docs-index`, never rewritten by skills — in `_cli-fab.md` § fab docs-index (it documents the command that owns the block), and make the six skill sites point at that § instead. `docs/memory/memory-docs/docs-index.md` then records that the deployed rule text lives in `_cli-fab.md` |
| `fab-operator.md:158`, `_cli-fab.md:1267` | "run-kit's `docs/specs/cron.md`" | **External-repo reference, not a fab-kit path** — `git log --diff-filter=A -- docs/specs/cron.md` confirms the file never existed in fab-kit; both sentences attribute it to run-kit explicitly. Resolved (Assumption 3): rewrite to name run-kit's cron spec in prose — e.g. "run-kit's operator-cron spec is the entry's design authority" — with no relative path at all |
| `_cli-external.md:54` | "the tool repo's canonical `docs/site/skill.md`" (shll-toolkit convention) | Same class as the row above — rewrite to "the shll toolkit `skill` standard" in prose, no path (Assumption 3) |

<!-- clarified: 2026-09-10 — user banned ALL bare doc paths in deployed content, including attributed external-repo ones (run-kit's docs/specs/cron.md, shll's docs/site/standards/skill.md); name the repo/standard in prose or via URL instead (Assumption 3) -->
<!-- clarified: 2026-09-10 — guard exemptions are the named allowlist constant + placeholder-shape detection only; no marker, no attribution token (Assumption 4) -->

**Category B — untouched** (39 occurrences): `docs/memory/index.md`, `docs/specs/index.md`, `docs/memory/_shared/removed-domains.md`, `docs/memory/_shared/utilities.md`, and the illustrative `docs/memory/pipeline/runtime/x.md` example in `docs-distill-memory.md:118`.

**Direction-of-ownership flips in fab-kit docs.** Where a spec or memory file currently says the skills "point here" and the operative rule now lives in the skill, update the doc to point at the skill (or state both carry it, skill canonical). Known candidates: `docs/memory/memory-docs/docs-index.md` (manual-block rule), `docs/specs/harness-adapters.md` (dispatch-prompt obligations; it already says `_preamble.md` § Worker Continuation "owns the mechanics" at lines 111/395 — extend that posture to the obligations), `docs/specs/stage-models.md` § Skill wiring, `docs/specs/config.md` (cascade), `docs/specs/change-types.md` (taxonomy pointer from the preamble).

### 4. Go test guard — `src/go/fab-kit/cmd/fab/kit_portability_test.go`

A new test in the **fab-kit module** (`src/go/fab-kit/cmd/fab/`, alongside `clifab_doc_test.go`, reusing its `findRepoFile` walk-up helper). fab-kit owns `fab sync` — the deploy boundary this rule is about — so the guard lives with the deployer. No new CLI command; no `_cli-fab.md` signature change.

Behavior:

1. Walk every regular file under `src/kit/` **except** `src/kit/VERSION` and `src/kit/reference/fkf.md` (resolved via `findRepoFile(t, "src/kit")`). Scope covers `skills/`, `templates/`, `migrations/` (historical files included — Assumption 5), and `scaffold/`. `reference/fkf.md` is excluded by name because it is a drift-guarded byte-copy of the published standard (Assumption 10).
<!-- clarified: 2026-09-10 — user chose to sweep the seven historical migrations too; the guard binds to migrations uniformly (Assumption 5) -->
2. Match, per line, the regex `\bdocs/(specs|memory|site)/[A-Za-z0-9_./-]+\.md\b` and `\bsrc/go/[A-Za-z0-9_./-]+\b`.
3. **Fail** on any `docs/specs/…`, `docs/site/…`, or `src/go/…` match, and on any `docs/memory/…` match that is not in the host-convention allowlist.
4. **Allowlist** (a named constant, `hostConventionPaths`): `docs/memory/index.md`, `docs/specs/index.md`, `docs/memory/_shared/removed-domains.md`, `docs/memory/_shared/utilities.md`. Placeholder-shaped paths — any path segment containing `{…}` or a bare single-letter `x.md` leaf — are also exempt as illustrative examples.
5. **No escape hatch.** There is no `portability-ignore` marker and no attribution-token recognition (Assumption 3). External-repo references name the repo or standard in prose or via URL; any bare `docs/…` path outside the allowlist fails. The test fails loudly with `file:line — cites a repo-local doc path <p>; restate the rule, or name the external repo/standard in prose` so the fix is obvious.
6. Include a **positive fixture test** (a `t.TempDir()` tree with one violating and one allowlisted file) so the matcher itself is tested, not only the live tree — Constitution VII (tests conform to the spec, i.e. the constitution rule text).

Run `go test ./cmd/fab/... ./internal/...` in `src/go/fab-kit` before done; the guard MUST pass on the swept tree, and the fixture test proves it fails on a violating file (no `git stash` — the stash stack is shared across worktrees).

### 5. Docs (specs + memory) touched during apply, beyond the flips in § 3

- `docs/specs/skills.md` § New Skill Checklist — add item 9: *"Portability of citations — a skill cites kit skills, `fab` commands, `fab/project/` files, `$(fab kit-path)/…` assets, or host convention paths; never fab-kit's `docs/specs/*`, `docs/memory/*`, `docs/site/*`, `src/go/*` (Constitution V; guarded by the fab-kit Go test)."*
- `docs/specs/architecture.md` § Directory Structure — one sentence after the "In the fab-kit dev repo, `src/kit/` is the canonical source…" paragraph noting that `docs/specs/` and `docs/memory/` are dev-repo-only and must not be cited from `src/kit/`.

## Affected Memory

- `_shared/configuration`: (modify) § `constitution.md` Structure / § Amending Constitution — record the 1.8.0 amendment (Principle V citation rule) and the code-quality/code-review additions in the 5 Cs description
- `_shared/context-loading`: (modify) note that `_preamble.md` is the deployed owner of the dispatch-prompt obligations and CLI-adapter procedure (no longer "per `docs/specs/harness-adapters.md`"); Design Decision: *Deployed Skills Own Their Rules Across the Deploy Boundary*
- `distribution/kit-architecture`: (modify) § Portability — the citation rule and the Go guard (`src/go/fab-kit/cmd/fab/kit_portability_test.go`); § Agent Skill Deployment — content portability, not only file placement
- `distribution/distribution`: (modify) § shll.ai Public Docs Site / § Toolkit Standards Conformance — where `_cli-fab.md`'s skill-bundle provenance prose was trimmed, the memory (not the skill) carries the embed/sync/drift-guard detail
- `memory-docs/docs-index`: (modify) the manual-block rule is restated in `_cli-fab.md` § fab docs-index for deployed purposes; this file points at it (ownership direction flips)
- `runtime/operator`: (modify) § The clock — reword the two "run-kit's `docs/specs/cron.md`" citations to name run-kit's operator-cron spec in prose, matching the swept skill wording (memory files are not deployed, so this is consistency, not a guard requirement)
- `runtime/dispatch`: (modify) only if the `_preamble.md:481` example rename or the § 3 harness-adapters flip changes a claim this file makes about who owns the obligations

## Impact

- **Governance**: `fab/project/constitution.md` (Principle V + Governance block, 1.7.0 → 1.8.0), `fab/project/code-quality.md`, `fab/project/code-review.md`
- **Kit skills** (`src/kit/skills/`): `_preamble.md`, `_cli-fab.md`, `_cli-agents.md`, `_cli-external.md`, `fab-operator.md`, `fab-continue.md`, `docs-hydrate-memory.md`, `docs-distill-memory.md`, `docs-reorg-memory.md`, `internal-skill-optimize.md` — ~25 line-level edits, prose only; no flow, tool-usage, or sub-agent structure change
- **Kit migrations** (`src/kit/migrations/`): 8 files carry `docs/specs/{stage-models,fkf,naming}.md` citations (2.2.0-to-2.3.0, 2.4.2-to-2.5.0, 2.5.5-to-2.6.0, 2.6.6-to-2.7.0, 2.11.0-to-2.12.0, 2.12.1-to-2.13.0, 2.19.4-to-2.20.0; `0.2.0-to-0.3.0` and `0.7.0-to-0.8.0` cite only index.md convention paths) — scope per Assumption 5
- **Go**: one new test file in `src/go/fab-kit/cmd/fab/`; no production code; `go test ./...` in `src/go/fab-kit`
- **Specs**: `docs/specs/skills.md`, `docs/specs/architecture.md`, `docs/specs/harness-adapters.md`, `docs/specs/stage-models.md`, `docs/specs/config.md`, `docs/specs/change-types.md` (pointer-direction flips only)
- **Deployed copies**: `.agents/skills/` and `.claude/skills/` are regenerated by `fab sync` — never edited directly. CI runs `gofmt`; run it on the new test before ship.
- **Not affected**: `fab sync` behavior, any CLI signature, `docs_index.roots` handling, Category B convention paths

## Open Questions

None outstanding. Both promptless-deferred questions (external-repo doc paths; historical-migration sweep scope) were resolved by the user on 2026-09-10 — see `## Clarifications`.

## Clarifications

### Session 2026-09-10

| # | Question | Answer |
|---|----------|--------|
| 3 | Are explicitly attributed external-repo doc paths (run-kit's `docs/specs/cron.md`, shll's `docs/site/standards/skill.md`) permitted in deployed skills, and how does the guard recognize attribution? | **Ban all doc paths; name the repo instead.** Rewrite to prose or URL. Guard fails on any `docs/…` path outside the host-convention allowlist; no exemption marker. |
| 5 | Sweep the seven historical migration files citing `docs/specs/{stage-models,fkf,naming}.md`, or grandfather them? | **Sweep them too.** `fkf.md` → `$(fab kit-path)/reference/fkf.md`, other citations dropped; the guard binds to migrations uniformly. |
| 4 | *(follows from 3)* Guard exemption mechanism | Named allowlist constant + placeholder-shape detection only; no marker, no attribution token. |
| 10 | *(consequence of 3, agent-resolved)* `src/kit/reference/fkf.md` carries URL-attributed fab-kit paths but is a drift-guarded byte-copy of the published standard | Excluded from the guard by name; not rewritten. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | One bundled change: constitution amendment + code-quality/code-review carve-out + sweep of the 13 Category A sites + Go test guard | Discussed — user chose the bundle over a bare constitution edit so the repo does not violate its own rule on landing | S:95 R:70 A:95 D:95 |
| 2 | Certain | Category B host-convention paths (`docs/memory/index.md`, `docs/specs/index.md`, `_shared/removed-domains.md`, `_shared/utilities.md`, illustrative `x.md`) stay; deriving them from `docs_index.roots` is a Non-Goal | Discussed — user drew this boundary explicitly; the paths describe the customer's layout, which `_preamble.md` § Always Load already hardcodes | S:95 R:90 A:95 D:90 |
| 3 | Tentative | Deployed content MUST NOT carry any bare relative doc path to another repository, even with explicit attribution — run-kit's cron spec and the shll `skill` standard are named in prose or via URL. The four sites (`fab-operator.md:158`, `_cli-fab.md:1267`, `_cli-fab.md:941`, `_cli-external.md:54`) are rewritten; the guard fails on any `docs/…` path outside the host-convention allowlist and has NO `portability-ignore` marker or attribution-token escape hatch | Clarified — user changed to "ban all doc paths; name the repo instead": attributed paths are equally dead in a customer repo, and the customer report flagged `cron.md` specifically | S:95 R:55 A:25 D:20 |
| 4 | Tentative | Guard exemption mechanism: a named allowlist constant (`hostConventionPaths`) for the four host-convention paths + placeholder-shape detection (`{…}` segments, `x.md`) for illustrative examples; no marker, no attribution token | Clarified — user confirmed via the Assumption 3 answer ("no exemption marker needed"); an allowlist constant is the smaller surface and a marker would add prose noise to skills | S:95 R:85 A:70 D:45 |
| 5 | Tentative | The sweep retroactively edits the 7 historical migration files (2.2.0-to-2.3.0 … 2.19.4-to-2.20.0): `docs/specs/fkf.md` → `$(fab kit-path)/reference/fkf.md`, the `stage-models.md`/`naming.md` citations dropped; the guard binds to `src/kit/migrations/` uniformly, no grandfathering | Clarified — user changed to "sweep them too": migrations are applied by `/fab-setup migrations` in customer repos during upgrades, so the dead paths ship; ~8 more files in the diff | S:95 R:60 A:35 D:25 |
| 6 | Certain | Constitution bump 1.7.0 → 1.8.0, Last Amended 2026-09-10, dated HTML-comment governance note in the existing style | Discussed — user specified the bump; the trailing comments establish "new normative MUST-rule → minor bump" (y8it, udwv precedents) | S:95 R:90 A:95 D:95 |
| 7 | Confident | Constitution wording: the proposed paragraph in § 1, with `src/kit/scaffold/` and `src/kit/reference/` added to the enumerated deployed set and `$(fab kit-path)/…` assets named as an allowed citation target | Description said "refine but keep the substance"; scaffold and reference also deploy (`ls src/kit/`), and 11 skill sites already cite `$(fab kit-path)/reference/fkf.md`, which must stay legal | S:80 R:80 A:85 D:75 |
| 8 | Confident | Change type `fix` | User called it "a fab-kit bug"; the `fix` heuristic (priority 1: `bug`, `broken`) fires on this intake's text ahead of `refactor`; `fab status refresh` recomputes it — no manual override unless the inferred value disagrees | S:75 R:95 A:85 D:75 |
| 9 | Confident | Guard lives in the fab-kit module at `src/go/fab-kit/cmd/fab/kit_portability_test.go`, reusing that package's `findRepoFile` walk-up helper; plain test, not a `fab` command | Description named `clifab_doc_test.go` as the neighbor; fab-kit owns `fab sync` (the deploy boundary); both modules have a `findRepoFile` but only fab-kit's package is deploy-adjacent. Constitution's CLI constraint is untouched because no command signature changes | S:80 R:85 A:85 D:80 |
| 10 | Confident | Guard scope covers every regular file under `src/kit/` except `src/kit/VERSION` and `src/kit/reference/fkf.md` (skills, templates, migrations, scaffold) | Everything under `src/kit/` deploys (Constitution V, context.md § Distribution). `reference/fkf.md` is a byte-copy of the published standard `docs/site/fkf.md`, synced by `scripts/sync-fkf.sh` with a CI drift-guard test — it cannot be rewritten in place, and with no escape hatch (Assumption 3) the only correct treatment is a named exclusion; its fab-kit references are URL-attributed "where to edit this standard" provenance, not rule pointers | S:80 R:85 A:85 D:75 |
| 11 | Confident | Manual-block rule (six skill sites) is restated once in `_cli-fab.md` § fab docs-index and the six sites point there; `docs/memory/memory-docs/docs-index.md` flips to point at the skill | Owner-or-pointer within the deployed set: one deployed owner, five deployed pointers. `_cli-fab.md` documents the command that writes the block, so it is the natural deployed owner | S:70 R:80 A:80 D:70 |
| 12 | Confident | `_preamble.md:481`'s example `summary:` string uses a placeholder-shaped path (`docs/memory/{domain}/{file}.md`) rather than a real fab-kit memory path | Illustrative only; a placeholder shape is what the guard's example exemption recognizes and cannot mislead a customer agent | S:75 R:95 A:90 D:85 |
| 13 | Confident | `_cli-fab.md:939,941` build-provenance prose (embed source `src/go/fab/cmd/fab/skill.md`, `scripts/sync-skill.sh`, `TestSkillEmbedMatchesCanonical`) is trimmed from the skill and carried in `docs/memory/distribution/distribution.md` | A customer agent needs the `fab skill` contract, not fab's build wiring; memory is the post-implementation home for provenance | S:70 R:85 A:80 D:70 |
| 14 | Confident | Repo-wide sibling sweep of owner-or-pointer prose touches `internal-skill-optimize.md:56` and `docs/specs/skills.md:1557` (both state the bare convention); passing mentions in memory stay unless a sentence becomes false | code-quality.md § Sibling Sweeps mandates the up-front class sweep; the grep enumerated the class | S:75 R:85 A:80 D:75 |
| 15 | Confident | Direction-of-ownership flips in `docs/specs/{harness-adapters,stage-models,config,change-types}.md` are pointer-only edits (the spec keeps its design content and adds "the deployed rule text lives in `<skill>` § …") | Constitution VI: specs stay human-owned design intent; only the who-points-at-whom sentence changes | S:70 R:85 A:80 D:75 |
| 16 | Certain | Non-Goals: no `fab sync` behavior change, no CLI surface change, no `docs_index.roots`-derived convention paths, no direct edits under `.agents/skills/` or `.claude/skills/` | Discussed — user enumerated these; Constitution V + code-quality anti-pattern 1 forbid the last | S:95 R:90 A:95 D:95 |
| 17 | Confident | Hydrate targets are the seven Affected Memory files above; `runtime/dispatch` is conditional on whether a § 3 flip changes one of its claims | Derived from the domain indexes read at intake (`_shared`, `distribution`, `memory-docs`, `runtime`); the description named the same domains | S:70 R:90 A:80 D:70 |

17 assumptions. Rows 3 and 5 were deferred would-be questions (promptless dispatch), resolved by the user in the 2026-09-10 clarify session (see `## Clarifications`); grades are derived from the Scores column by `fab score`.
