# Intake: Operator Spawn Forms as Two Orthogonal Axes — Task-less Bare-Agent Spawn

**Change**: 260912-6n6o-operator-spawn-orthogonal-axes
**Created**: 2026-09-12

## Origin

Conversational — diagnosed in a user session, then dispatched promptless (`{questioning-mode} = promptless-defer`; no questions asked, would-be questions land in `## Assumptions`).

> **User, in the operator** (launched in a non-fab repo, `~/code/wvrdz/loom`): "Start a agent in a new worktree"
>
> **Operator**: refused — "I need a task description to spawn against — an empty worktree with no instructions has nothing to work on" — then attempted a tool call that failed with "Invalid tool parameters", then re-asked for a task in plain text.
>
> **User's position**: the operator is being made more generic; it always knows how to start an agent via `fab agent`; a task is not a required part of starting an agent; refusing is wrong.
>
> **Decision** (user approved the axis recast): "Orthogonality - yes - the correct direction. go ahead"

Root cause, diagnosed and agreed in the conversation: `src/kit/skills/fab-operator.md` defines spawning entirely in terms of *work*.

- §1 "Coordinate, don't execute" opens "Every task, bug report, or idea the user hands the operator is a work request and goes to an agent" — a bare spawn request matches no rule, so the model treated the task as a missing required input.
- §6 Working a Change enumerates exactly four forms, and every form carries a prompt: form 1 `/fab-fff <change>`, form 2 `/fab-new <raw>`, form 3 `/fab-new <id>`, form 4 raw task text plus the closing instruction `Commit on this branch, push, and open a draft PR against the default branch.` No form has an empty prompt.
- Form 4's completion is a chained `github-pr` item — with no task there is no PR, so the task appears structurally required.

The primitives already support the bare case, so nothing is built in Go:

- The `pane` kind is "**any agent session in a tmux pane**"; a fab change "is never required to track" (§6 The pane Kind and Spawn Rules).
- `rk tab new … -- <argv…>` accepts no trailing prompt token — the `["<initial-prompt>"]` in `_cli-agents.md` § Spawn Composition's raw form is optional. `--ready` needs only a `--` command (the session command), not a prompt (verified: `rk tab new --help`).
- `wt create --non-interactive` with no branch argument creates an exploratory worktree with a random name, or the value of `-n/--name` (verified: `wt create --help`).
- `fab agent default -o yaml --repo <repo>` works outside a fab project via the project-free config cascade (`_cli-fab-operator.md` § fab agent).

## Why

**Problem.** The operator's job is to hand agents to the user and keep them coordinated; it refused the most basic version of that job — start an agent, no instructions — because the skill's routing rules only recognise inputs shaped like work. The user then had to argue with the coordinator instead of getting a pane. The failure is a *classification* gap, not a mechanism gap: every step of the spawn sequence works with an empty prompt today.

**Consequence of not fixing.** Every "start an agent" / "open me a worktree with an agent in it" request keeps bouncing, in every repo, fab or not. Users route around the operator (running `wt create` + `fab agent` by hand), which contradicts the delegate-everything direction decided 2026-09-11 (kp3d: the operator reads plans and delegates; the single never-execute constraint). The failed tool call in the transcript also shows the model improvising when the rules leave it nowhere to go — the same dead-end pattern pudv fixed for non-fab repos, now one axis further along.

**Why this approach.** The four forms are the cartesian product of two independent yes/no questions with one cell missing, so the honest fix is to name the axes and fill the cell. That *removes* the enumeration (complexity down) instead of adding a fifth case to it. Alternatives rejected in the conversation:

1. **Bolt a "form 5 / bare-spawn clause" onto §1 and form 4** — fixes the gap but keeps the enumeration and adds a case (complexity up). Rejected in favour of the axis recast.
2. **Reword only the "asks only which repo" sentence** — the model already ignored it because it never classified the input as a work request; a stronger sentence in the same frame will not hold.
3. **Push bare spawns out of the operator** (tell the user to run `wt create` + `fab agent`) — contradicts the delegate-everything direction (kp3d).

## What Changes

All edits are skill prose in `src/kit/skills/` plus the memory mirror. **No Go, no CLI signature, no migration, no constitution amendment** (no MUST rule is added or changed). Net prose SHOULD NOT grow: the two-axis rule and one bare-agent paragraph replace the four-form list, and every sweep site that says "form 4" / "forms 1–3" becomes a cell reference.

### 1. `src/kit/skills/fab-operator.md` §6 Working a Change — the two-axis recast (owner)

Replace the intro blockquote, the "Every form runs …" sentence, the numbered forms 1–4, and the "On completion …" sentence with the following structure (wording may be tightened; content is normative):

> Every spawn runs §6's target-repo + target-session → worktree → guarded activation → dependencies → target-repo session command → tab → track sequence. What rides the tab is decided by **two independent axes**, probed once at spawn:
>
> - **Axis A — pipeline present?** `test -d <target-repo>/fab` against the main-worktree root from step 1.
> - **Axis B — task given?** The user handed the operator work — raw text, a backlog ID, a Linear issue, an existing change — or did not.
>
> | | task given | no task |
> |---|---|---|
> | **`fab/` present** | **Pipeline.** Existing change → skill `fab-fff` with argument `<change>` (item `scope.repo` or `branch_map`; the transient override targets the pipeline and step 4 activates the pointer). Raw text → skill `fab-new` with the raw description as its argument. Backlog ID / Linear issue → resolve first (optional `idea` lookup per `_cli-external.md` § Delegation and binary gate), then skill `fab-new` with argument `<id>`. Render per `_cli-agents.md` § Skill Prompts; the existence guard skips activation until `/fab-new` Step 10. Completion: the built-in predicate — PR ready, optionally archive. | **Bare agent** (below) |
> | **no `fab/`** | **Plain agent.** Prompt = the raw task text followed by the closing instruction `Commit on this branch, push, and open a draft PR against the default branch.` — not a skill invocation: no `skill_prefix`, nothing to render. Step 3 is `wt create --non-interactive --name <name>` with no branch argument. Tracked change-less at step 8: `fab operator track add <wt> --kind pane --pane <pane-id> --session <session> --repo <repo> --branch <wt>` (id = the worktree name; `--branch <wt>` lands `{branch, repo}` in `branch_map[<wt>]` and survives the item's removal); no stages, so no `--stop-stage`. **Completion is a chained `github-pr` item**: once the PR URL is known (the pane shows it, or the user relays it), `fab operator track add <wt>-pr --kind github-pr --scope '{"repo":"<repo>","pr":<n>}' --check-every 2m` then `fab operator track rm <wt>`; the user may `track rm` the pane item at any time. | **Bare agent** (below) |
>
> **Bare agent — no task, either repo type.** The same eight steps: step 3 `wt create --non-interactive` with no branch argument (wt picks a random name, or `--name <name>` when the user gave one — that name is the worktree, the branch, the tab name, and the item id); step 4's existence guard trivially skips (no change); step 5 dependencies as usual; step 6 as usual (`skill_prefix` unused); **step 7 passes no prompt token** — the argv after `--` ends at the session command, no skill prefix, no closing PR instruction; step 8 `fab operator track add <wt> --kind pane --pane <pane-id> --session <session> --repo <repo> --branch <wt>`, no `--stop-stage`, **no chained `github-pr` item**. The user drives that pane — including typing `/fab-new` in it; a change that later appears is an ordinary `changed` delta. **Completion is pane death, agent exit, or the user's `track rm`**: the operator reports and acks per §4 Tick Behavior step 4, and never respawns (respawn is queue-driven only, § Queues). **Pipeline-first still holds in the fab-project cell**: the operator sends nothing, so nothing enters the repo outside `/fab-new` — "MUST enter through `/fab-new`" binds *work the operator sends*, not the act of opening an agent.
>
> **Never `fab init` the repo** — the rule and its rationale are owned by §6 The pane Kind and Spawn Rules → Pipeline-first.

The table + bare-agent paragraph above is the **only** place the cells are defined. Every other site points at it by cell name (e.g. "§6 Working a Change → non-fab + task cell", "→ bare agent"), never by form number (owner-or-pointer, `fab/project/code-quality.md`).

### 2. `src/kit/skills/fab-operator.md` §1 Principles — "Coordinate, don't execute" (the ask rule)

Current opening: "Every task, bug report, or idea the user hands the operator **is a work request** and goes to an agent: the fab pipeline when the target repo has a `fab/` project (§6 Working a Change forms 1–3 — …), a plain agent in a fresh worktree otherwise (form 4); …"

Target (normative content):

> Every task, bug report, or idea the user hands the operator **is a work request** and goes to an agent; a request to start an agent **with no task is a bare spawn** and also goes to an agent — the operator **never refuses to start an agent**. Routing is §6 Working a Change's two axes (does the target repo have `fab/`; was a task given). A report naming a live tracked item is a send to that item's agent. …

Current closing: "The only thing the operator asks about a work request is **which repo** (plus the target-session tie-break in §6 Spawning an Agent step 2) — never whether to spawn; …"

Target:

> The only thing the operator asks about a work request **or a spawn** is **which repo** (plus the target-session tie-break in §6 Spawning an Agent step 2) — never whether to spawn, and **never for a task: a spawn request with no task is a bare spawn, not a question**; …

The rest of the row (executing prohibition, maintenance allowlist, confirmations still apply) is unchanged.

### 3. `src/kit/skills/fab-operator.md` — sweep of every "form" reference to a cell reference

| Site | Today | Target |
|---|---|---|
| §2 Context Loading (~line 56) | "… `/fab-new` → `/fab-fff` in a fab project … or a plain-agent spawn in a repo with no `fab/` (§6 Working a Change form 4)" | "… or a plain agent in a repo with no `fab/`, or a bare agent when no task was given (§6 Working a Change)" |
| §2 wt Gate (~line 100) | "§6 step 3 — every form, including the non-fab plain-agent form 4 of §6 Working a Change" | "§6 step 3 — every cell of §6 Working a Change, bare spawns included" |
| §6 The pane Kind — Pipeline-first bullet (~467) | "its work goes to a plain agent in a fresh worktree (§6 Working a Change form 4)" | "its work goes to a plain agent in a fresh worktree (§6 Working a Change → non-fab + task cell)". **Add**: "A spawn with no task is exempt for a different reason: nothing is sent, so nothing bypasses `/fab-new` (→ bare agent)." |
| §6 The pane Kind — Spawn in a worktree bullet (~468) | "— and every form-4 plain-agent spawn —" | "— and every plain-agent or bare-agent spawn —" |
| §6 The pane Kind — `--stop-stage` paragraph (~470) | "A form-4 plain-agent spawn has no stages and takes no `--stop-stage`; its completion is the chained `github-pr` item (§6 Working a Change form 4)." | "Plain-agent and bare-agent spawns have no stages and take no `--stop-stage`; the plain agent completes through its chained `github-pr` item, the bare agent through pane death / agent exit / `track rm` (§6 Working a Change)." |
| §6 Spawning an Agent step 4 Guard (~494) | "the non-fab form 4 has no change and skips the switch entirely" | "the non-fab and bare cells have no change and skip the switch entirely" |
| §6 Spawning an Agent step 6 (~496) | "`skill_prefix` goes unused because form 4 sends a raw prompt" | "`skill_prefix` goes unused in the non-fab and bare cells (raw prompt, or none)" |
| §6 Spawning an Agent step 7 (~497) | "compose the prompt (forms 1–3: …; form 4: the raw task text plus the PR instruction, no skill prefix — § Working a Change)" | "compose the prompt per § Working a Change's cell (pipeline: the selected skill and arguments per `_cli-agents.md` § Skill Prompts; plain agent: the raw task text plus the PR instruction, no skill prefix; **bare agent: no prompt token at all** — the argv ends at the session command)". The `rk tab new` line's trailing `"<prompt>"` becomes `["<prompt>"]` (optional), matching `_cli-agents.md`. |
| §6 Spawning an Agent step 7 window marks (~505) | note `"<id> · <stage>"` | unchanged for pipeline spawns; for a change-less item the note is `"<id>"` (no stage segment) |
| §6 Spawning an Agent step 8 (~506) | "a form-4 spawn never acquires a change — its id stays the worktree name and its completion is the chained `github-pr` item" | "a plain-agent spawn never acquires a change — its id stays the worktree name and its completion is the chained `github-pr` item; a bare spawn is tracked the same way and completes on pane death / agent exit / `track rm`" |
| §7 Linear and Slack Items step 4 (~761) | "(a `scope.repo` with no `fab/` takes §6 Working a Change form 4 instead — raw prompt, no `--stop-stage`, still `--spawned-by <id>`)" | "(a `scope.repo` with no `fab/` takes the non-fab + task cell — raw prompt, no `--stop-stage`, still `--spawned-by <id>`)". Linear/Slack-driven spawns always carry a task (the issue) and are never bare. |
| §7 Conversational Map — c7-sahil87 row (~773) | "spawn per §6 Working a Change form 4 (plain agent …)" | "spawn per §6 Working a Change → non-fab + task cell (plain agent …)" — rest of the row unchanged |
| §7 Conversational Map — **new row** | — | `"Start an agent in a new worktree" · "open me an agent in ~/code/loom"` → bare spawn per §6 Working a Change (no prompt token), then `fab operator track add <wt> --kind pane --pane %N --repo /home/x/code/loom --session work --branch <wt>`; completes on pane death / agent exit / `track rm`, no chained item |

Verification grep after apply, repo-wide over `src/kit/skills/` and `docs/memory/`: `form 4`, `form-4`, `forms 1–3`, `forms 1-3`, `Working a Change form`, `all four work paths` — zero hits outside `fab/changes/` and `docs/memory/**/log.md`.

### 4. `src/kit/skills/_cli-external.md` § wt → Operator Spawning Rules

- Line ~132: "A repo with **no** `fab/` has no change branch: the operator's plain-agent form runs `wt create --non-interactive --worktree-name <name>` with no branch argument (`fab-operator.md` §6 Working a Change form 4)." → "A spawn with no change branch — a repo with no `fab/`, or a bare spawn with no task — runs `wt create --non-interactive [--name <name>]` with no branch argument (`fab-operator.md` §6 Working a Change)."
- **Flag spelling**: `wt create --help` shows the worktree-name flag as `-n, --name`; there is no `--worktree-name` flag. The two code-block lines in this section (`--checkout` route and new-branch route) and the sentence above are respelled `--name`. This is a one-token correction confined to the section this change already edits; the sibling spelling in `fab-operator.md` (form 4 text, replaced wholesale by §1 above) and `docs/memory/runtime/operator.md` (the form-4 paragraph, rewritten by hydrate) are covered by the sweep.

### 5. `docs/memory/runtime/operator.md` — present-truth mirror (hydrate)

- Principles → "Coordinate, don't execute" paragraph (line ~21): mirror §1's bare-spawn sentence and the extended ask rule.
- Multi-Repo Coordination → the "All four work paths (…)" paragraph (line ~99) and the "**Non-fab repo → plain agent (form 4).**" paragraph (line ~101): replace with the two-axis statement — the table's cells as prose — plus the bare-agent paragraph; `--worktree-name` → `--name`.
- Design Decisions:
  - **Pipeline-First Routing Principle (operator7)** (line ~349): "(Working a Change form 4)" → cell reference; append `*Updated by*: 260912-6n6o-operator-spawn-orthogonal-axes (a task-less spawn sends nothing and so bypasses nothing)`.
  - **Non-Fab Repos Get a Plain Agent …** (line ~656): append `*Updated by*: 260912-6n6o-… (the form became the non-fab + task cell of the two-axis rule)`.
  - **New DD** — *Spawn Routing Is Two Orthogonal Axes, Not an Enumerated Form List*: **Decision** — the operator routes every spawn by (repo has `fab/`) × (task given); the no-task column is one cell, the bare agent, identical for both repo types; the operator never refuses to start an agent and never asks for a task. **Why** — the four forms were the product of two yes/no questions with one cell missing; the model read the missing cell as "task is a required input" and refused. Naming the axes fills the cell and deletes the enumeration. **Rejected** — a fifth form (keeps the enumeration, adds a case); rewording the ask sentence alone (the input was never classified as a work request, so the sentence never fired); pushing bare spawns out of the operator (contradicts delegate-everything, kp3d). *Introduced by*: 260912-6n6o-operator-spawn-orthogonal-axes.

### Not in scope

- Any Go change; any `fab operator track` verb or `rk tab new` change; auto-detecting a "bare" request in the binary.
- Changing respawn semantics (stays queue-driven), the built-in pane completion predicate, or the `github-pr` chaining for plain agents.
- `fab init` behaviour (rule unchanged, still owned by Pipeline-first).
- `docs/specs/operator.md`: it does not restate the forms (only the v9 spawn-in-worktree row), so no edit is required; a v13 version-table row is a human curation call (Constitution VI), not part of this change.
- Constitution: no amendment (no normative rule added or changed).

## Affected Memory

- `runtime/operator`: (modify) Principles "Coordinate, don't execute" paragraph (bare spawn + never-asks-for-a-task); Multi-Repo Coordination spawn-path paragraphs ("All four work paths …" and "Non-fab repo → plain agent (form 4)") rewritten as the two-axis rule + bare-agent paragraph, `--worktree-name` → `--name`; Design Decisions — new entry *Spawn Routing Is Two Orthogonal Axes, Not an Enumerated Form List*, plus `*Updated by*` lines on *Pipeline-First Routing Principle (operator7)* and *Non-Fab Repos Get a Plain Agent, Never the Operator's Hands, Never an Unprompted fab init*.

No other memory file documents the spawn forms (repo-wide grep for `form 4` / `forms 1–3` / `plain agent` hits only `runtime/operator.md` in `docs/memory/`). `docs/memory/runtime/index.md` is regenerated by `fab docs-index` if the file's `description:` changes; no manual edit.

## Impact

- **Files**: `src/kit/skills/fab-operator.md` (§1, §2 ×2, §6 The pane Kind ×3, §6 Spawning an Agent steps 4/6/7/8, §6 Working a Change wholesale, §7 ×3), `src/kit/skills/_cli-external.md` (§ wt Operator Spawning Rules), `docs/memory/runtime/operator.md`. Three files; no Go; no tests to add.
- **Deployed content**: both skill files deploy via `fab sync` into `.agents/skills/` / `.claude/skills/`; edits go to `src/kit/skills/` only (Constitution V, code-quality anti-pattern). Nothing new may cite a fab-kit-only path — the existing Go portability guard in `src/go/fab-kit/cmd/fab/` covers it; run `go test ./src/go/fab-kit/cmd/fab/ -run Portab` (or the package) after apply.
- **Behaviour change** for operators: a task-less "start an agent" request now produces a worktree + agent tab + tracked pane item instead of a refusal, in fab and non-fab repos alike. No change to any tasked spawn.
- **Lane**: light (few tasks, prose only). Sibling-sweep class per `fab/project/code-quality.md`: the memory file documenting the skill (`runtime/operator.md`) and the `_cli-external.md` pointer sentence — both listed above.

## Open Questions

- None blocking. Whether `docs/specs/operator.md`'s version table gains a v13 row is a human curation decision outside this change (Constitution VI).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Recast §6 Working a Change as two orthogonal axes — (target repo has `fab/`) × (task given) — replacing the enumerated forms 1–4 | Discussed — user approved "Orthogonality - yes - the correct direction. go ahead"; alternatives 1–3 rejected in conversation | S:95 R:70 A:90 D:95 |
| 2 | Certain | The fab + task cell keeps forms 1–3's behaviour verbatim and the non-fab + task cell keeps form 4's behaviour verbatim (prompt, `--branch <wt>`, chained `github-pr` completion) | Discussed — "today's forms 1–3, unchanged behavior" / "today's form 4, unchanged behavior" | S:95 R:85 A:95 D:95 |
| 3 | Certain | Bare agent = same eight-step spawn sequence; step 7 passes no prompt token, no skill prefix, no closing PR instruction; step 8 tracks a change-less `pane` item, id = worktree name, `--branch <wt>`, no `--stop-stage`, no chained `github-pr` item | Discussed — specified cell by cell; primitives verified (`rk tab new --ready` needs only a `--` command; `pane` kind tracks change-less panes) | S:95 R:80 A:90 D:90 |
| 4 | Certain | Bare-pane completion is pane death / agent exit / user `track rm`; the operator reports and acks per §4 Tick Behavior step 4 and never respawns (respawn stays queue-driven) | Discussed — stated in the decision; matches the existing `agent_exited` / `pane_death` ack path | S:90 R:80 A:90 D:90 |
| 5 | Certain | §1 ask rule extends to spawns: the operator asks only which repo (+ session tie-break); a missing task is never a question — it is a bare spawn; the operator never refuses to start an agent | Discussed — the user's core complaint; the sentence exists today for work requests and is widened | S:95 R:85 A:90 D:95 |
| 6 | Certain | Skill prose + memory only — no Go, no CLI signature, no migration, no constitution amendment | Discussed — "Nothing needs to be built in Go"; no MUST rule changes, so no constitution bump | S:90 R:90 A:95 D:95 |
| 7 | Certain | Net prose does not grow: the cell table + one bare-agent paragraph replace the four-form list; every sweep site becomes a cell reference, never a form number (owner-or-pointer) | Discussed — explicit constraint; `code-quality.md` owner-or-pointer rule | S:80 R:85 A:80 D:80 |
| 8 | Certain | Pipeline-first paragraph states explicitly that a bare spawn in a fab project sends nothing, so nothing bypasses `/fab-new` | Discussed — "State this explicitly so the model does not read MUST-enter-through-/fab-new as forbidding a bare spawn" | S:90 R:85 A:90 D:90 |
| 9 | Certain | `docs/specs/operator.md` is not edited — it does not restate the forms; a v13 version row is a human curation call | Verified by grep (only the v9 spawn-in-worktree row mentions spawning); Constitution VI | S:85 R:95 A:90 D:90 |
| 10 | Certain | Linear/Slack-driven and queue-driven spawns always carry a task or change and are never bare; their §7 / § Queues references become cell references only | Both paths spawn from an issue or a queued change by construction | S:70 R:90 A:90 D:90 |
| 11 | Confident | Respell the worktree-name flag `--name` in every sentence and the two `_cli-external.md` § Operator Spawning Rules code-block lines this change touches (`wt create --help` shows `-n, --name`; no `--worktree-name` flag exists) | Verified against the installed binary; a one-token drift fix confined to text already being rewritten; not raised in the conversation | S:50 R:90 A:95 D:70 |
| 12 | Confident | §7 Conversational Map gains a bare-spawn row ("Start an agent in a new worktree") mapping to the bare cell + `track add … --branch <wt>`; the existing c7-sahil87 row is reworded to the cell name | The map is the model's routing table and the failure was a routing miss; the sibling form-4 row already exists there | S:50 R:90 A:85 D:80 |
| 13 | Confident | Window marks for a change-less item: `rk tab mark @<window_id> auto` as usual, note `"<id>"` with no stage segment; cleared at removal as usual | Existing note contract is `"<id> · <stage>"`; a change-less pane has no stage to show; not discussed | S:45 R:90 A:80 D:75 |
| 14 | Certain | Memory: one new DD (*Spawn Routing Is Two Orthogonal Axes …*) plus `*Updated by*` lines on the Pipeline-First and Non-Fab-Plain-Agent DDs; no DD is deleted | FKF present-truth style — superseded reasoning is recorded as an update, not erased; hydrate owns the exact wording | S:70 R:90 A:85 D:85 |

14 assumptions (11 certain, 3 confident, 0 tentative, 0 unresolved).
