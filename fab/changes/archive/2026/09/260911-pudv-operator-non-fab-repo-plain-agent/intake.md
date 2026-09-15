# Intake: Operator Non-Fab-Repo Work Path — Plain Agent Form

**Change**: 260911-pudv-operator-non-fab-repo-plain-agent
**Created**: 2026-09-11

## Origin

One-shot, autonomous invocation (`/fab-new` with a complete design description agreed with the user on 2026-09-11; no interactive questions — nobody was watching, so every open point below is resolved as a graded assumption). This change is a **follow-up to change 260911-1159** (operator pane kind + pid fingerprint, PR #667) and is branched from that branch, NOT main, because it edits the §6 "The pane Kind and Spawn Rules" section that only exists on that branch. Its PR must be retargeted to `260911-1159-operator-pane-kind-pid-fingerprint` after `/git-pr` creates it (stacked-PR rule).

> TITLE: Operator non-fab-repo work path — every work request goes to an agent: the fab pipeline when the repo has `fab/`, a plain agent otherwise.
>
> ORIGIN / INCIDENT (2026-09-11): the running operator, working a batch of "C7" repo-profile edits across sahil87 repos, hit `c7-sahil87` (the GitHub profile repo — no `fab/` project) and asked the user a three-way question: (1) run `fab init` there then pipeline it, (2) "direct edit, plain PR, no fab — I make the edit directly", (3) skip it. The user answered by typing a fourth option: "Start an agent on that repo, and ask it to …". Two violations of `src/kit/skills/fab-operator.md` §1 "Coordinate, don't execute": option 2 offers the one prohibited action (the operator editing files itself), and the question itself was not allowed — §1 says the only thing the operator asks about a work request is WHICH REPO, and the repo was already known.
>
> ROOT CAUSE: the skill has exactly one sanctioned delegation path, §6 "Working a Change", and all three of its forms (existing change / raw text / backlog-or-Linear id) begin with `/fab-new`, so they are unusable in a repo with no `fab/` directory. The prohibition on executing is stated; the alternative for non-fab repos is not. Facing the dead end, the operator improvised.
>
> WHAT CHANGES (skill prose only — `src/kit/skills/fab-operator.md`; verify whether `_cli-external.md` § wt or `_cli-agents.md` § Spawn Composition need a one-line pointer; NO Go change; NO run-kit change):
> 1. §6 Working a Change gains a fourth form, **Non-fab repo → plain agent**: when `<repo>/fab/` does not exist, the work request still goes to an agent in a fresh worktree via the SAME §6 spawn sequence (target repo + target session, `wt create --non-interactive` in the target repo — wt works on any git repo — session command from `fab agent default -o yaml --repo <repo>`, which degrades to the project-free config cascade when there is no `fab/` (documented in `_cli-fab-operator.md` § fab agent "Runs OUTSIDE a fab project"), `rk tab new … -- <argv> "<prompt>"`). The prompt is the raw task text plus the closing instruction "commit on this branch, push, and open a draft PR against the default branch" — NOT a skill invocation (no `skill_prefix`, nothing to render). Track it unconditionally as a `pane` item with no change (item id = worktree name — the `pane` kind from change 260911-1159 makes a change-less pane trackable); completion = a chained `github-pr` item the operator adds once the PR URL is known (or user `track rm`); state the `--stop-stage` inapplicability.
> 2. Scope **Pipeline-first** (§6 "The pane Kind and Spawn Rules" after 1159): it binds repos WITH a `fab/` project — "new work in a fab project MUST enter through `/fab-new` …". It never meant "no fab, no work".
> 3. **`fab init` is never a default.** Bootstrapping fab into a repo is a repo-structure decision that belongs to the user; the operator MAY mention it once in its report as an option, never pick it, never ask about it as a menu.
> 4. §1 "Coordinate, don't execute" sentence: "enters through §6 Working a Change (a fresh report takes the raw-text form — a `/fab-new` spawn …)" → "goes to an agent: the fab pipeline when the target repo has a `fab/` project (§6 Working a Change forms 1–3), a plain agent in a fresh worktree otherwise (form 4). Reading code … is executing and is prohibited" — keep the rest of the row verbatim (maintenance allowlist, the which-repo-only ask).
> 5. §7 Conversational Map: add a row — "Fix the broken brew line in c7-sahil87's README" (repo has no `fab/`) → spawn per form 4, then `fab operator track add c7-sahil87-<wt> --kind pane --pane %N --repo … --session …`.
> 6. Sweep every place that says or implies "every spawn is a `/fab-new`" (§2 wt Gate "fab-change/pane spawn path", §6 The pane Kind, §6 Spawning an Agent step 7's "compose the selected skill", Working a Change intro "all three work paths below MUST go through the fab pipeline") so the non-fab form is not contradicted elsewhere.
>
> NOT IN SCOPE: any Go change; auto-detecting non-fab repos in the binary; `fab init` automation; run-kit; docs/specs. Memory at hydrate: `docs/memory/runtime/operator.md` (Working a Change forms; Design Decision "Non-fab repos get a plain agent, never the operator's hands, never an unprompted fab init").
>
> CONSTRAINTS: Constitution V (1.8.0) — deployed skill text cites no fab-kit-only doc paths; do not restructure §6 Queues/merge choreography; light lane expected (skill prose, ~6 sites, one file + maybe one pointer). change_type: docs or feat — pin whichever `/fab-new` infers, prefer `feat` (a new operator behavior).

## Why

**The pain point.** The operator skill (`src/kit/skills/fab-operator.md`) states a hard prohibition — §1 "Coordinate, don't execute": the operator never reads code to diagnose, never edits files, and asks about a work request only *which repo* — but offers exactly one sanctioned delegation path, §6 "Working a Change", whose three forms all begin with `/fab-new`. `/fab-new` requires `fab/project/config.yaml` and `fab/project/constitution.md` (its Pre-flight step aborts otherwise). So in a repo with no `fab/` directory the operator has a stated prohibition and no stated alternative. On 2026-09-11 this produced the incident in Origin: facing `c7-sahil87` (a GitHub profile repo, no `fab/`), the operator improvised a three-way menu whose option 2 ("I make the edit directly") is the one prohibited action, and whose very existence violated the which-repo-only ask rule. The user had to type the correct answer themselves — "start an agent on that repo" — which is what the skill should have said.

**The consequence of not fixing.** Every multi-repo batch that touches a non-fab repo will hit the same dead end, and an improvising operator will either (a) execute in its own pane (contaminating the orchestration pane, violating §1), (b) run `fab init` in a repo the user never asked to bootstrap (a repo-structure decision the operator does not own), or (c) skip the work silently. All three are worse than the obvious right answer, and the prohibition text alone cannot prevent them because it names no exit.

**Why this approach.** The fix is prose-only and additive: the §6 spawn sequence already works for any git repo — `wt create` operates on any git repo, `fab agent default -o yaml --repo <repo>` degrades to the project-free config cascade when there is no `fab/` (documented in `_cli-fab-operator.md` § fab agent "Runs OUTSIDE a fab project"), and `rk tab new … -- <argv> "<prompt>"` takes any prompt string. The `pane` kind from change 260911-1159 makes a change-less pane trackable (a fab change is "observed, never required"). So a fourth Working-a-Change form needs no new mechanism — it reuses every existing step and swaps the skill-rendered prompt for the raw task text plus a closing PR instruction. Alternatives rejected: (i) Go-side non-fab detection or `fab init` automation — out of scope by user decision; the operator is a skill, and the decision is a one-line existence check it can make itself; (ii) allowing the operator to edit directly in non-fab repos — contradicts §1's single never-execute constraint (user decision 2026-09-11, kp3d); (iii) making `fab init` the default path — bootstrapping fab is the user's repo-structure decision, never the operator's.

## What Changes

All edits are to **`src/kit/skills/fab-operator.md`** (canonical source; deployed copies under `.agents/skills/` and `.claude/skills/` regenerate via `fab sync` and are never edited). One optional one-line pointer may land in `_cli-external.md` § wt or `_cli-agents.md` § Spawn Composition (see § Pointer verification below). No Go change. No run-kit change. No docs/specs change.

### 1. §6 Working a Change — fourth form: Non-fab repo → plain agent

Append a **form 4** to the numbered list after form 3 ("Backlog ID or Linear issue"). Target content (wording may be tightened at apply, semantics fixed):

```markdown
4. **Non-fab repo → plain agent:** when the target repo has no `fab/` directory (probe: `test -d <target-repo>/fab`), the work request still goes to an agent in a fresh worktree via the SAME spawn sequence — target repo + target session (steps 1–2), `wt create --non-interactive` in the target repo (step 3; wt works on any git repo), step 4's existence guard trivially skips (no change can exist), step 5 dependencies as usual, the session command from `fab agent default -o yaml --repo <target-repo>` (step 6; with no `fab/` it degrades to the project-free config cascade — `_cli-fab-operator.md` § fab agent "Runs OUTSIDE a fab project"), and `rk tab new … -- <spawn-argv…> "<prompt>"` (step 7). The **prompt is NOT a skill invocation** — there is no `skill_prefix` to apply and nothing to render per `_cli-agents.md` § Skill Prompts. It is the raw task text followed by the closing instruction: `Commit on this branch, push, and open a draft PR against the default branch.` Track it unconditionally (step 8) as a `pane` item with no change: item id = the worktree name from step 3 (`fab operator track add <wt> --kind pane --pane <pane-id> --session <session> --repo <repo>`). There is no fab change, so the built-in stage-based completion never fires and `--stop-stage` does not apply — do not pass it. Completion is a **chained `github-pr` item**: once the agent reports the PR URL (its pane shows it, or the user relays it), add `fab operator track add <wt>-pr --kind github-pr --scope '{"repo":"<repo>","pr":<n>}' --check-every 2m` and remove the pane item; the user may also `track rm` the pane item at any time. **Never `fab init` the repo** — bootstrapping fab is the user's repo-structure decision (§6 The pane Kind and Spawn Rules); the operator MAY mention it once in its report as an option, never pick it, never ask about it as a menu.
```

Update the completion sentence that follows the list: "On completion (all three): PR ready, optionally archive. Both raw text and backlog paths use `/fab-new` …" → "On completion (forms 1–3): PR ready, optionally archive; form 4 completes through its chained `github-pr` item. Forms 2 and 3 use `/fab-new` to generate a proper intake with traceability …" (keep the Origin-section sentence).

### 2. §6 The pane Kind and Spawn Rules — scope Pipeline-first to fab projects

Current bullet: "**Pipeline-first** — new work MUST enter through `/fab-new`, then `/fab-fff`, `/fab-ff`, or `/fab-continue`; never send raw implementation instructions or use `/fab-continue` to skip intake. Direct actions are exactly the §1 maintenance allowlist."

Target: "**Pipeline-first** — new work **in a fab project** (the target repo has a `fab/` directory) MUST enter through `/fab-new`, then `/fab-fff`, `/fab-ff`, or `/fab-continue`; never send raw implementation instructions to a fab-project agent or use `/fab-continue` to skip intake. A repo with **no** `fab/` is not exempt from delegation — its work goes to a plain agent in a fresh worktree (§6 Working a Change form 4); it is exempt only from the pipeline, because there is no pipeline there. The operator never runs `fab init` to manufacture one — bootstrapping fab into a repo is the user's repo-structure decision, mentioned at most once as an option in a report, never picked, never offered as a menu. Direct actions are exactly the §1 maintenance allowlist."

The **Spawn in a worktree** bullet stays ("Every pipeline command, including a one-line change …") but gains the clause "— and so does every form-4 plain-agent spawn" so it visibly binds both.

### 3. `fab init` is never a default

Owned in the Pipeline-first bullet (item 2 above) with a pointer from form 4 (item 1) — owner-or-pointer, not both restated in full (code-quality anti-pattern "Stating an owned rule AND pointing at its owner"). Form 4 carries the one-line operative restatement "Never `fab init` the repo" plus the pointer; the full rationale lives once in the Pipeline-first bullet.

### 4. §1 "Coordinate, don't execute" row

Current: "Every task, bug report, or idea the user hands the operator **is a work request** and enters through §6 Working a Change (a fresh report takes the raw-text form — a `/fab-new` spawn in a fresh worktree; a report naming a live tracked item is a send to that item's agent). Reading code to reproduce or diagnose, or editing files, in the operator pane **is** executing and is prohibited. …"

Target: "Every task, bug report, or idea the user hands the operator **is a work request** and goes to an agent: the fab pipeline when the target repo has a `fab/` project (§6 Working a Change forms 1–3 — a fresh report takes the raw-text form, a `/fab-new` spawn in a fresh worktree), a plain agent in a fresh worktree otherwise (form 4); a report naming a live tracked item is a send to that item's agent. Reading code to reproduce or diagnose, or editing files, in the operator pane **is** executing and is prohibited. …" — the rest of the row (maintenance allowlist, the which-repo-only ask, the §3/§6 confirmation carve-outs) stays **verbatim**.

### 5. §7 Conversational Map — new row

Add after the "Track pane %222" row:

| Utterance | Verb |
|---|---|
| "Fix the broken brew line in c7-sahil87's README" (repo has no `fab/`) | spawn per §6 Working a Change form 4 (plain agent, raw task + PR instruction, no skill prefix), then `fab operator track add c7-sahil87-<wt> --kind pane --pane %N --repo /home/x/code/c7-sahil87 --session work`; on PR URL, chain `fab operator track add c7-sahil87-<wt>-pr --kind github-pr --scope '{"repo":"/home/x/code/c7-sahil87","pr":N}' --check-every 2m` |

(Item-id convention in the row follows the description's `c7-sahil87-<wt>` example — repo name prefix plus worktree name — which is a display convenience; form 4's rule is "item id = worktree name" and the row's prefixed id is an acceptable user-readable variant of it. See Assumptions row 6.)

### 6. Sweep — every "every spawn is a `/fab-new`" site

| Site | Current claim | Target |
|---|---|---|
| §2 wt Gate, first sentence | "`wt` gates the **agent spawn path only** (§6 step 3)" | Unchanged in substance — already spawn-path generic. Add "(every form, including the non-fab plain-agent form 4)" so the gate visibly covers form 4. |
| §2 Context Loading, line ~56 | "knowing a fresh idea needs `/fab-new` → `/fab-fff`" | Append "in a fab project, or a plain-agent spawn in a repo with no `fab/` (§6 Working a Change form 4)". |
| §6 The pane Kind — Pipeline-first bullet | see item 2 | see item 2 |
| §6 The pane Kind — `--stop-stage` paragraph | "A spawn that deliberately parks early … MUST be tracked with `--stop-stage hydrate`" | Append: "A form-4 plain-agent spawn has no stages and takes no `--stop-stage`; its completion is the chained `github-pr` item (§6 Working a Change form 4)." |
| §6 Spawning an Agent step 4 | "Raw/backlog forms wait for `/fab-new` Step 10" | Append "; the non-fab form 4 has no change and skips the switch entirely (the guard's `fab resolve` fails in a repo with no `fab/`, which is the intended fail-soft)". |
| §6 Spawning an Agent step 6 | "Read both `command` … and `skill_prefix`" | Append one sentence: "In a repo with no `fab/`, `fab agent` degrades to the project-free cascade (`_cli-fab-operator.md` § fab agent) and `skill_prefix` is unused — form 4 sends a raw prompt." |
| §6 Spawning an Agent step 7 | "compose the selected skill and arguments per `_cli-agents.md` § Skill Prompts" | "compose the prompt — for forms 1–3 the selected skill and arguments per `_cli-agents.md` § Skill Prompts; for form 4 the raw task text plus the PR instruction, no skill prefix — then open the tab …" |
| §6 Spawning an Agent step 8 | "The id is the change id when known, else the worktree name from step 3; raw-text spawns are tracked at spawn and their change appears later as a `changed` delta" | Append: "a form-4 spawn never acquires a change — its id stays the worktree name and its completion is the chained `github-pr` item." |
| §6 Working a Change intro blockquote | "all three work paths below MUST go through the fab pipeline (`/fab-new` then a pipeline command for new work; …)" | "forms 1–3 below MUST go through the fab pipeline (`/fab-new` then a pipeline command for new work; the appropriate stage for already-intaked changes); form 4 is the non-fab exception — it still spawns an agent in a worktree, but there is no pipeline to enter — never raw implementation instructions to a **fab-project** agent pane." |
| §6 Working a Change "Every form runs …" sentence | "target-repo + target-session → worktree → guarded activation → dependencies → target-repo session command → tab → track sequence" | Unchanged (form 4 runs the same sequence). |
| §6 Working a Change completion sentence | "On completion (all three) … Both raw text and backlog paths use `/fab-new`" | see item 1 |
| §7 Conversational Map | — | see item 5 |
| §6 Queues / merge choreography | — | **Do not restructure** (constraint). Only touch if a sentence literally says every spawn is `/fab-new`; grep confirms none do. |

Apply MUST grep `src/kit/skills/fab-operator.md` for `fab-new`, `Pipeline-first`, `all three`, `skill_prefix`, `compose the selected skill`, `stop-stage` before finishing and reconcile every hit against this table.

### Pointer verification (`_cli-external.md` § wt, `_cli-agents.md` § Spawn Composition)

`_cli-external.md` § wt already states "wt operates on the current working directory's repo" and the repo-targeted spawning note; it does not claim the repo must be a fab project, so **no edit is required** there. `_cli-agents.md` § Spawn Composition / § Skill Prompts describe rendering a skill prompt when one is selected; they do not say a spawn must carry a skill prompt. Apply verifies both by reading them; if either contains a sentence implying "the prompt is always a skill invocation" or "the target repo is always a fab project", add a one-line pointer to `fab-operator.md` §6 Working a Change form 4 — otherwise leave both untouched. Expected outcome: no edit (Assumptions row 4).

### Constitution V compliance

Every citation in the new prose is to another kit skill (`_cli-fab-operator.md` § fab agent, `_cli-agents.md` § Skill Prompts / § Spawn Composition, `_cli-external.md` § wt), a `fab` command, or `fab-operator.md`'s own sections — never `docs/specs/*`, `docs/memory/*`, `docs/site/*`, or `src/go/*`. The Go portability guard test in `src/go/fab-kit/cmd/fab/` fails on violations; apply runs it.

## Affected Memory

- `runtime/operator`: (modify) Coordination Principles — "Coordinate, don't execute" paragraph gains the fab-project / non-fab split; § Working a Change (or its equivalent requirement block) gains form 4 (Non-fab repo → plain agent: same spawn sequence, raw prompt + PR instruction, pane item id = worktree name, chained `github-pr` completion, no `--stop-stage`); Pipeline-first scoped to fab projects; new Design Decision **"Non-fab repos get a plain agent, never the operator's hands, never an unprompted fab init"** (Decision / Why / Rejected / Introduced by 260911-pudv) — the `## Design Decisions` section already carries "Pipeline-First Routing Principle (operator7)" and "Operator7 Direct fab-new for Raw Text Spawns", which the new DD refines rather than replaces.
- `runtime/index`: (modify) description regenerated by `fab docs-index` only if the operator.md description frontmatter changes (it should not need to — the file already describes "repo/session-targeted spawning"); no hand edit.

## Impact

- **Files**: `src/kit/skills/fab-operator.md` (primary, ~10 edit sites across §1, §2, §6, §7); optionally one line in `_cli-external.md` or `_cli-agents.md` (expected: none). At hydrate: `docs/memory/runtime/operator.md`.
- **Deployed copies**: `.agents/skills/fab-operator/SKILL.md` and `.claude/skills/fab-operator/SKILL.md` regenerate via `fab sync` — never edited directly (code-quality anti-pattern; Constitution V).
- **Behavior contract**: the operator gains a sanctioned fourth delegation path and loses the implicit permission to improvise (menu / direct edit / `fab init`) when a repo has no `fab/`. No CLI surface changes; no state-file schema changes; the `pane` kind's change-less tracking (260911-1159) is consumed, not modified.
- **Tests**: no Go change, so no Go test changes; the existing Constitution V portability guard (`src/go/fab-kit/cmd/fab/` test over `src/kit/**` citations) runs as the acceptance check for the citation rule. `gofmt` is irrelevant (no `.go` touched).
- **Dependencies**: branched from `260911-1159-operator-pane-kind-pid-fingerprint` (PR #667); the PR for this change is retargeted to that branch (`gh pr edit <pr> --base 260911-1159-operator-pane-kind-pid-fingerprint`) so its diff shows only this change. Ship must record this in the PR body.
- **Lane**: light (one file, ~10 prose sites, no code) — `/fab-fff` may fork light on plan task count.

## Open Questions

- None blocking. See Assumptions for the graded judgment calls (item-id spelling in the §7 row; chained `github-pr` id convention; whether the completion sentence rewording counts as restructuring §6 — it does not, Queues is untouched).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill prose only — `src/kit/skills/fab-operator.md` edits; no Go, no run-kit, no docs/specs | Stated verbatim in the description's WHAT CHANGES and NOT IN SCOPE | S:95 R:90 A:95 D:95 |
| 2 | Certain | Form 4 reuses the §6 spawn sequence unchanged (steps 1–8) with a raw prompt instead of a rendered skill prompt | Description specifies "the SAME §6 spawn sequence"; `wt`, `fab agent --repo` (project-free cascade) and `rk tab new` all verified to work without `fab/` per their own helper docs | S:95 R:85 A:90 D:95 |
| 3 | Certain | `change_type` = `feat` — pin explicitly with `fab status set-change-type` if refresh infers otherwise | Description: "prefer `feat` (a new operator behavior)"; the slug/description contains no `fix`/`refactor` keyword but "docs" may match the `docs` regex, so an explicit pin is the safe path | S:90 R:95 A:95 D:90 |
| 4 | Confident | `_cli-external.md` § wt and `_cli-agents.md` § Spawn Composition need **no** pointer — apply verifies by reading and adds one line only if a sentence contradicts form 4 | Both helpers were grepped: § wt says nothing about fab projects; § Spawn Composition describes rendering a skill prompt when selected, not that one is mandatory. Description asks to "verify", not to edit | S:80 R:90 A:75 D:70 |
| 5 | Confident | Chained completion item id = `<wt>-pr`, kind `github-pr`, `--check-every 2m`, scope `{repo, pr}`; added by the operator when the PR URL is visible in the pane or relayed by the user; pane item then removed | Description says "a chained `github-pr` item the operator adds once the PR URL is known (or user `track rm`)"; the id suffix and cadence follow the existing §7 `pr-913` row's shape | S:75 R:85 A:80 D:70 |
| 6 | Confident | §7 row uses the description's `c7-sahil87-<wt>` item id verbatim while form 4's rule stays "item id = worktree name"; the row's repo-prefixed id is a readable variant, and the prose says so in one parenthetical | Description gives both spellings; they are not in conflict (a user may name the item anything — `track add <slug>`), so both are recorded rather than one silently dropped | S:70 R:90 A:75 D:65 |
| 7 | Confident | `fab init` rule owned once in the §6 Pipeline-first bullet; form 4 carries a one-line operative restatement plus pointer | Code-quality owner-or-pointer anti-pattern; the description wants the rule visible at the decision point (form 4) and as principle (Pipeline-first) | S:80 R:90 A:85 D:75 |
| 8 | Confident | Sweep table in What Changes § 6 is the apply checklist; apply greps `fab-new`, `Pipeline-first`, `all three`, `skill_prefix`, `compose the selected skill`, `stop-stage` in the skill before finishing | Sibling-sweep rule in code-quality.md; description item 6 lists four named sites and says "every place" | S:85 R:85 A:85 D:85 |
| 9 | Confident | Rewording the Working-a-Change completion sentence and the Pipeline-first bullet does NOT count as "restructuring §6 Queues/merge choreography"; Queues, Dependency Resolution and merge-mode prose are untouched | Constraint names Queues/merge choreography specifically; the edited paragraphs are the Working a Change list and the pane-kind principles | S:80 R:85 A:85 D:80 |
| 10 | Confident | Memory hydrate target is `docs/memory/runtime/operator.md` only (Coordination Principles paragraph, Working-a-Change requirement, new Design Decision); `runtime/index.md` regenerates only via `fab docs-index` | Description names the file and the DD title; index is generated (do-not-hand-edit banner) | S:85 R:90 A:85 D:85 |
| 11 | Tentative | The §2 Context Loading sentence ("knowing a fresh idea needs `/fab-new` → `/fab-fff`") is in the sweep even though the description did not name it | It implies every fresh idea is a `/fab-new`; description item 6 says "every place that says or implies". Low cost, easily reverted if the reviewer deems it noise | S:55 R:90 A:60 D:55 |

11 assumptions (3 certain, 7 confident, 1 tentative, 0 unresolved).
