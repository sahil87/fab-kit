---
type: memory
description: "`/fab-new`, `/fab-draft`, `/fab-clarify`, `/fab-ff`, `/fab-fff` — the intake-only planning stage: the shared `_generation` (Intake + unified Plan, plus Intake-from-Diff/Plan-from-Diff for `/fab-adopt`) and `_intake` (Steps 0–9, interactive | promptless-defer) helpers with thin call-sites; re-run/ID-collision resume semantics; pull-based change_type with sticky explicit set; SRAD v2 demerit confidence scoring in the `_srad` helper; fab-ff/fab-fff as thin wrappers over the `_pipeline` bracket."
---
# Planning Skills

**Domain**: pipeline

## Overview

The planning skills (`/fab-new`, `/fab-clarify`) handle the single planning stage of the 6-stage Fab pipeline: **intake**. They produce the only pre-code planning artifact (`intake.md`), which defines *what* changes and *why*. Intake is also the sole confidence gate — all human judgment is frontloaded here. Requirement capture and the implementation plan are co-generated into a single `plan.md` at apply entry — see [execution-skills.md](/pipeline/execution-skills.md) — not by a planning skill. There is no `spec` stage or `spec.md` artifact (j6cs).

`/fab-fff` and `/fab-ff` are also documented here because their planning behavior originated as planning skills. `/fab-fff` is the **full-pipeline command** (intake → review-pr, intake-gated, no frontloaded questions, autonomous rework). `/fab-ff` is the **fast-forward command** (intake → hydrate, intake-gated, autonomous rework). See sections below for details.

## Shared Generation Partial

Artifact generation is defined in a single shared partial: `$(fab kit-path)/skills/_generation.md`. Requirement generation is folded into the Plan Generation Procedure invoked by the apply skill (within `/fab-continue`) at apply entry — there is no standalone Spec Generation Procedure (j6cs).

The partial contains four procedures — the two **forward** procedures (Intake Generation, Plan Generation) and, as of t54n, two **diff-based** procedures (Intake-from-Diff, Plan-from-Diff) for `/fab-adopt`. The **Intake Generation Procedure** (consumers: `/fab-new`, `/fab-draft`, and `/fab-continue`'s intake-`active` regeneration row) and the **Plan Generation Procedure** below (consumers: `/fab-continue`, `/fab-ff`, `/fab-fff`, at apply entry). `/fab-continue` belongs to **both** forward consumer groups (d9rs):
- **Plan Generation Procedure** — template loading (`plan.md`), one walk that emits `## Requirements` (RFC-2119 statements with stable `R#` IDs + GIVEN/WHEN/THEN scenarios, generated FIRST from the intake-derived design), then paired Task + Acceptance entries per requirement with sequential `T-NNN`/`A-NNN` IDs. It reads `intake.md` as input (not a separate `spec.md`), and includes a one-release legacy `spec.md` ingestion path (fold a leftover `spec.md` into `## Requirements` if present and `plan.md` lacks them). Trace annotations are **REQUIRED** (j6cs): each `## Tasks` item carries a `<!-- R# -->` annotation; each `## Acceptance` item names its `R#`. No `[NEEDS CLARIFICATION]` markers are emitted into `plan.md` — under-specified points become graded SRAD assumptions in `## Assumptions` instead, and the walk carries an explicit numbered **`## Assumptions` emission step** (step 7, immediately before the write step — c5tr): persist every inline-graded assumption per `_srad.md` § Assumptions Summary Block — three grades only (Certain/Confident/Tentative; Unresolved is intake-only), the `Scores` column required on every row, footer `{N} assumptions ({Ce} certain, {Co} confident, {T} tentative).` The section is ALWAYS present in the artifact — `0 assumptions.` footer with no table rows when empty (`_srad`'s omit-when-zero rule is scoped to the displayed output summary only, never to artifacts, keeping `fab score` parsing uniform); the Intake Generation step 4 notes the same always-present rule. The procedure runs at apply entry, not as a separate planning stage. Counts (`task_count`, `acceptance_count`, `acceptance_completed`) flow into `.status.yaml` `plan:` block via `fab status refresh` (`internal/refresh.Refresh`), self-healed at the transition seams (`fab status advance`/`finish`, `fab preflight` — y022); skills MAY also call `fab status set-acceptance` for explicit updates (e.g., review marking acceptance items complete).
- **Diff-based procedures — Intake-from-Diff + Plan-from-Diff (t54n; sole consumer: `/fab-adopt`, see [execution-skills.md](/pipeline/execution-skills.md) § `/fab-adopt`)**. The forward procedures generate an artifact from *intent* (the description, then the intake-derived design); the diff-based procedures **reverse-engineer** the artifacts from a *fixed existing branch diff* — the artifact *describes* what shipped rather than *plans* what to build. **Intake-from-Diff**: reconstruct `intake.md` from `git diff {base}...HEAD` + the changed-file list + the PR body/title (or branch name) — Origin = `adopted from {PR url or branch}`, Why/What-Changes synthesised from the diff, **Affected Memory** inferred from which `docs/memory/` domains the changed paths map to, Impact from the changed paths; SRAD + `fab score` still apply (the reconstructed design is graded like any intake). **Plan-from-Diff**: write a deliberately **MINIMAL** `plan.md` from the *same* understanding (no re-read of the diff) — plain-language `## Requirements` (a restatement of the intake's What-Changes, grouped by change area; **the only part hydrate reads, so effort concentrates here**), a single all-`[x]` `## Tasks` stub, a single all-`[x]` `## Acceptance` stub, and a header note that apply was skipped and the plan is reverse-engineered to feed hydrate. It deliberately emits **NO** `R#`/`T-NNN`/`A-NNN` traceability IDs, GIVEN/WHEN/THEN scenarios, phases, or `[P]` markers — the apply↔review traceability loop never runs for an adopted change, so that scaffolding is dead weight. The only stable parser contract preserved is the three heading literals `## Requirements` / `## Tasks` / `## Acceptance` (confirmed against `templates/plan.md`). **Both artifacts are generated by the same main-session agent in ONE pass** (NOT via a dispatched apply) — the agent reads the diff + PR body once and writes intake then plan from that single understanding, because both merely describe one fixed existing diff and a context boundary between them would only invite drift and waste; a human-confirmation checkpoint sits between them (the late deliberation the bypass skipped, mirroring `/fab-new`'s interactive intake moment).

Command invocations are auto-logged via `preflight.sh --driver <skill-name>` — skills do not call `log-command` manually. All event commands (`start`, `advance`, `finish`, `reset`, `fail`) accept an optional `driver` parameter; skills always pass it to identify the invoking skill (e.g., `fab-continue`, `fab-ff`).

**Pull-based bookkeeping (y022)**: Bookkeeping commands (confidence scoring, change type inference, plan metadata) are supplemented by `fab status refresh`, self-healed at the transition seams (`fab status advance`/`finish`, `fab preflight`) — never a harness hook, since a hook fires only in the Claude harness and this is correctness-critical state (see [hooks-may-enhance-never-own.md](/pipeline/hooks-may-enhance-never-own.md)). Refresh is a **reliability layer** — it catches bookkeeping the agent forgets. For `plan.md`, refresh performs section-bounded parsing: counts `- [ ]` + `- [x]` items between `## Tasks` and the next `##` heading for `task_count`; same between `## Acceptance` and the next `##` heading for `acceptance_count`; counts `- [x]` in `## Acceptance` for `acceptance_completed`. Missing sections leave the corresponding fields untouched (defensive: avoid overwriting valid values with zero on a malformed in-progress write). Skills keep their existing bookkeeping instructions unchanged for agent-agnostic portability (non-Claude-Code agents rely on skill instructions only). All bookkeeping commands are idempotent, so both `fab status refresh` and the skill running the same command produces no conflict.

Each skill retains its own orchestration logic (stage guards, question handling, auto-clarify, resumability). Only the generation mechanics are shared.

## Requirements

### `/fab-new <description>`

`/fab-new` starts a new change from a natural language description. It is adaptive: clear inputs get a quick intake, vague inputs trigger conversational exploration. It creates the change folder, initializes status tracking, generates an intake (with Origin section), advances intake to `ready`, activates the change, and creates the matching git branch. Output includes `intake.md` plus activation and branch creation output.

#### Slug Generation and Change Creation

The agent generates a 2-6 word slug (lowercase, hyphen-joined, no articles/prepositions) from the description. The slug SHALL NOT include a parsed Linear issue ID — it contains only the descriptive portion (e.g., `add-oauth`); the issue ID is recorded in `.status.yaml`'s `issues` array (via `fab status add-issue`), never in the folder name. The folder name format `{YYMMDD}-{XXXX}-{slug}` is constructed by the Go CLI's `fab change new`, which handles date generation, random/provided 4-char ID, collision detection, directory creation, `created_by` detection, `.status.yaml` initialization, and status integration (`start intake fab-new`; command logging via `fab log` when `--log-args` is passed). The skill calls `fab change new --slug <slug> [--change-id <4char>] [--log-args <description>]` as a single operation and captures the folder name from stdout.

#### Adaptive Behavior (SRAD-Driven)

`/fab-new` adapts its interaction style based on the input clarity:

1. **Clear input** — SRAD scoring identifies few or no Unresolved decisions. The skill generates the intake with up to 3 targeted questions (highest blast radius), assumes all Confident/Tentative decisions, and completes quickly.
2. **Vague input** — SRAD scoring identifies many Unresolved decisions. The skill enters **conversational mode**: back-and-forth exploration with no fixed question cap, starting with the highest-impact decisions (lowest Reversibility + lowest Agent Competence). Each question builds on previous answers. The conversation ends when the confidence score reaches >= 3.0 and the user signals satisfaction, or the user terminates early.

#### Promptless Dispatch Under `/fab-proceed` (Defer-and-Surface, w7dp/3xaj)

When `/fab-proceed` runs the create-new path, it dispatches the **`_intake` Create-Intake Procedure with `{questioning-mode} = promptless-defer`** as a promptless subagent (3xaj). There is no user to ask and no `[AUTO-MODE]` prefix is sent. The dispatch prompt carries the **defer-and-surface contract**, which `promptless-defer` *encodes in the called helper*: the subagent asks NO questions — each decision SRAD would normally ask (Unresolved, including Critical-Rule hits) is instead recorded in the intake's `## Assumptions` table as an Unresolved row with Rationale `Deferred — promptless dispatch` and returned in the subagent result; `/fab-proceed` surfaces the deferred decisions as informational lines (staying zero-prompt) before delegating to `/fab-fff`. The intake gate is the **structural backstop**: a deferred decision must be scored with honestly-low dimensions so its `composite < 20`, where the demerit curve's penalty (≥ 2.0) drops the change to the 3.0 gate or below (4yi8 — blocking is emergent from the curve, not a short-circuit), so a genuine deferral fails the gate and the pipeline stops normally for the user to resolve via `/fab-clarify`. `_srad.md`'s Critical Rule carries the matching **promptless-dispatch carve-out** (cross-referencing `fab-proceed.md` § Create-Intake Dispatch): the MUST-ask is satisfied by deferring and surfacing, never by silently assuming — everywhere a user is reachable, the MUST-ask applies unchanged.

**`/git-branch` is REQUIRED on the create-new rows (3xaj).** `_intake(promptless-defer)` stops at intake `ready` and does NOT activate or branch (activate + branch are `/fab-new`'s Steps 10–11 tail, which the EXTRACTION BOUNDARY keeps at the call site and which `_intake` never runs). So the create-new rows chain **`_intake` → `/fab-switch` → `/git-branch`** precisely BECAUSE `_intake` stops short of branching — `/fab-switch` activates and `/git-branch` creates the matching branch, reaching the active-and-branched end state. The `/fab-switch`-prefixed relevant-intake rows and the branch-mismatch row keep `/git-branch` for the same reason (switching activates a change but creates no branch). See [execution-skills.md](/pipeline/execution-skills.md) for the full fab-proceed dispatch table.

#### Gap Analysis

Before committing to an intake, `/fab-new` evaluates whether the change is needed:

1. Checks for existing mechanisms in the current workflow, codebase, or memory
2. Evaluates scope — is the idea too broad (should be split) or too narrow (part of something larger)?
3. Considers alternatives — simpler approaches, extending existing skills

If an existing mechanism covers the idea, the skill presents its findings and lets the user decide whether to proceed. If no change folder is created, no `Next:` line is shown.

#### Change Initialization

The skill SHALL:
1. Generate the slug (AI task: word selection, article removal, issue ID prefixing)
2. Call `fab change new` with `--slug`, optional `--change-id` (backlog ID), and `--log-args` (description). The command handles: directory creation, `created_by` detection (`gh api user` → `git config user.name` → `"unknown"`, silent fallback), `.status.yaml` initialization from the kit template, and status integration (`start intake fab-new`; command logging via `--log-args`)
3. Generate `intake.md` from the template (including Origin section), loading `fab/project/constitution.md` and `fab/project/config.yaml` as context

The intake-creation work above — Steps 0–9 (parse input, slug, gap analysis, create change, conversation context mining, generate `intake.md`, verify change type, confidence, SRAD question selection, advance to `ready`) — lives once in the shared **`_intake` Create-Intake Procedure** (`src/kit/skills/_intake.md`; 3xaj), which `fab-new.md` invokes as a thin call-site with `{questioning-mode} = interactive`. `fab-new.md` retains only the **activate + branch tail** (Steps 10–11) below — see § The `_intake` Shared Create-Intake Procedure.

After the Create-Intake Procedure advances intake to `ready` — signaling the artifact exists and is open for `/fab-clarify` refinement — `/fab-new` auto-activates the change via `fab change switch` (Step 10) and creates the matching git branch inline (Step 11). Branch creation applies the same branch-case logic as the standalone `/git-branch` skill — Step 11 states it as a single condition/command/report table annotated "evaluate in order, first match wins" (szxd), preceded by the context commands (current branch, dirty-tree `{dirty_count}`, local target-exists check, remote `origin/{name}` check, upstream check, `fab change resolve` on the current branch) and a keep-in-sync comment referencing `git-branch.md` Step 4. The table has **six rows** (g8st): (1) already active (no-op), (2) target exists locally (checkout), (3) **target exists only on the remote** → `git checkout --track "origin/{name}"` — never recreating a divergent local with `checkout -b` (report: `checked out, tracking origin/{name}`), (4) on main/master (create), (5) local-only branch passing the rename guard (rename), (6) local-only branch belonging to a different change OR pushed branch (create, leaving the old branch intact). The **rename guard** (row 5) matches `git-branch.md` Step 4 — the two twins are kept in sync via the in-file comment, deliberately NOT delegated to `/git-branch` at runtime (inline wins on runtime token economy): rename when the current branch resolves to no change (`fab change resolve "$(git branch --show-current)"` fails — e.g., a disposable `wt create` name) **or to the SAME change being branched** (g8st — e.g., a worktree placeholder named with the change's own ID); when it matches a *different* change's branch (e.g., after `/fab-switch` away from an unpushed change), create a new branch via `git checkout -b` instead, leaving the other change's branch intact (known caveat: the new branch inherits the old change's HEAD). A **dirty-tree note** (g8st) is appended to the report line whenever `{dirty_count}` > 0 and the matched row runs `git checkout -b` or `git branch -m`: ` — note: {dirty_count} uncommitted change(s) carried over from {old_branch}` — non-blocking (warn, never stash-prompt: the step runs inside no-questions/orchestrated flows). The twins' one deliberate divergence lives outside the shared rows: fab-new derives `{dirty_count}` **excluding `fab/changes/{name}/`** — the change's own just-created artifacts (`intake.md`, `.status.yaml`, `.history.jsonl`) always exist uncommitted by Step 11 and would fire the note on every run; git-branch counts the full porcelain output. The git step is non-fatal — if not in a git repo, it warns and skips; if a git operation fails, it reports the error and the change remains activated. For create-without-activate behavior, use `/fab-draft` instead — `fab-draft.md` is also a **thin call-site over `_intake`** (3xaj): it reads `_intake.md` and executes the Create-Intake Procedure with `{questioning-mode} = interactive`, then stops at `ready` (does NOT activate, does NOT create a git branch). Its Output is fab-new's minus the `Activated:`/`Branch:` lines, ending with the Activation Preamble `Next:` line; it adds no activation/git error rows. Steps 10–11 are `fab-new.md`'s tail, which `/fab-draft` never reads — so there is no run-by-momentum hazard. The shared Steps 0–9 live in `_intake.md` (deliberately NOT in `_generation.md`, which holds only the artifact-generation mechanics `_intake` Step 5 delegates to).

#### Re-Run Semantics (Idempotency)

`/fab-new` and `/fab-draft` are partially idempotent, declared in each skill's Key Properties section with an `Idempotent?` row (`git-pr.md`'s Key Properties declares its contract: re-run after ship is a no-op via the "already shipped" path, conditioned on an **OPEN** PR — a CLOSED/MERGED PR does not short-circuit (g8st); see [execution-skills.md](/pipeline/execution-skills.md)). In Step 3 of both skills, before creating anything, an existing non-archived change for a detected ID is detected — branching by ID type:

- **Backlog ID** (4-char — embedded in the folder-name prefix): `fab resolve --id {id}`, then compare its stdout (the canonical 4-char ID of the matched change) for **equality** with `{id}` — only an exact ID match names an existing change for this ID (w7dp). Resolution is substring-based, so `{id}` occurring inside another change's *slug* also resolves — with a different canonical ID — and MUST NOT route to resume; on an exact match the folder name comes from `fab resolve --folder {id}`.
- **Linear ID** (never in folder names — slugs exclude issue IDs; the ID lives only in `.status.yaml` `issues` arrays): `grep -lw "{ISSUE_ID}" fab/changes/*/.status.yaml 2>/dev/null` — `-w` anchors on word boundaries so `DEV-123` does not match `DEV-1234` (uliv); the single-level glob naturally excludes `fab/changes/archive/`; a match's parent folder is the existing change.

On a collision the skill does NOT create a duplicate — it **routes to resume**: reports `Change {name} already exists for [{id}].` and points the user to `/fab-switch {name}` then `/fab-continue` (whose intake-`active` dispatch row regenerates a missing intake, recovering an interrupted creation). Both Error Handling tables map the `fab change new` collision failure (`Change ID already in use`) to the same recovery guidance. `change.go`'s collision error is unchanged and remains the safety net for backlog IDs only; Linear re-runs pass no `--change-id`, so the issues-array scan is their only collision guard. A **natural-language re-run intentionally creates a new change on every run** (fresh random ID) — there is no dedup for NL input.

#### Change Type (Pull-Based)

`change_type` is recomputed by `fab status refresh` (`internal/refresh.Refresh`), not owned by the skills: refresh infers and writes the type to `.status.yaml`, self-healed at the transition seams (`fab status advance`/`finish`, `fab preflight`) **whenever `change_type_source` is absent or `inferred`**, using word-boundary keyword regexes evaluated in order — `fix` → `refactor` (incl. "redesign") → `docs` → `test` → `ci` → `chore` — defaulting to `feat`. The recompute runs on demand / at the transition seams, never via a harness hook (y022). `/fab-new` and `/fab-draft` run no manual keyword inference and no unconditional `set-change-type` (uliv). Instead, Step 6 of both skills: (1) **verifies** the recomputed result by reading `change_type` from the change's `.status.yaml` (e.g., `grep '^change_type:'` — `fab preflight` does not emit this field), and (2) **overrides only if wrong** via `fab status set-change-type`.

**Explicit set is sticky (jznd).** `set-change-type` marks the type **`explicit`** by also writing `change_type_source: explicit`, and `fab status refresh` **skips both inference and the type overwrite when the source is `explicit`** (acceptance counting and other bookkeeping still run). So an override is never clobbered by a later refresh — a deliberately-corrected type survives all subsequent intake refinements and transition-seam refreshes. An absent/empty `change_type_source` (pre-jznd changes) decodes as `inferred`, so those re-infer on each refresh — only an explicit human set turns it off. (A passing `must-fix` in a feature intake does not classify as `fix`; `bug-fix`/`hot-fix`/`bug-free` and standalone `fix`/`bug`/`broken`/`regression` do — jznd.) See [schemas.md](/pipeline/schemas.md) § `.status.yaml` Change-Type Fields for the field and refresh-guard schema.

#### Confidence

After generating `intake.md` and verifying the change type, `/fab-new` persists the confidence score by calling `fab score --stage intake <change>` in normal mode (not `--check-gate`). This writes the score to `.status.yaml`, making it visible to all consumers (`/fab-switch`, `/fab-status`, `fab change list`) without recomputation. `intake.md` is the **sole, authoritative** scoring source — there is no separate spec-stage score and no `confidence.indicative` flag (`fab score` never writes it — j6cs). Output format: `Confidence: {score} / 5.0 ({N} decisions)`.

#### Output

`/fab-new` produces `intake.md` as its primary artifact. It does not generate `plan.md` or any other downstream artifacts (there is no `spec.md` artifact). The intake includes an **Origin** section recording how the change was initiated (description text, conversational vs. one-shot mode, key decisions from the conversation). After the intake, the output includes `Activated: {name}` (Step 10) and `Branch: {name} (created|created, leaving {old_branch} intact|checked out|renamed from {old_branch}|already active)` (Step 11). Step 7 (the score persist step) is titled "Confidence" (j6cs). The Output template places the **Assumptions summary as the final content block immediately before the `Next:` line** (c5tr; per `_srad.md` § Assumptions Summary Block's SHALL — order: intake → Confidence → Activated → Branch → Assumptions → `Next:`); the block is omitted from the displayed output only when 0 assumptions were made, while the artifact's `## Assumptions` section is always present regardless. The closing `Next:` line is **derived at runtime** per `_preamble.md` § Lookup Procedure (intake-state row, default first: `/fab-continue, /fab-ff, /fab-fff, /fab-proceed, or /fab-clarify`), never a hardcoded enumeration (d9rs). `/fab-draft`'s Activation Preamble `Next:` derives its post-switch command list the same way.

#### Context

Loads: config, constitution, `docs/memory/index.md` (to understand the existing memory landscape).

### The `_intake` Shared Create-Intake Procedure (3xaj)

The **pre-boundary intake-creation procedure** — `/fab-new` Steps 0–9 — lives once in the internal helper `src/kit/skills/_intake.md` (deployed to `.claude/skills/_intake/SKILL.md` by `fab sync`; 3xaj). It is the symmetric counterpart to `_pipeline.md`: where `_pipeline.md` is the shared *post*-intake orchestration bracket, `_intake.md` is the shared *pre*-intake orchestration body. Both follow the proven shape — a shared body parameterized by one knob, with call-site-specific tails staying in the call-site files.

**The one knob is `{questioning-mode}`** (`interactive` | `promptless-defer`), applied only at Step 8 (SRAD-based question selection). Every other step (0–7, 9) is mode-invariant:

- **`interactive`** — used by `/fab-new` and `/fab-draft`. Step 8 asks via SRAD (SRAD-driven question selection, no fixed cap, conversational mode when 5+ Unresolved). Byte-identical to the pre-3xaj inline fab-new Step 8.
- **`promptless-defer`** — used by `/fab-proceed`'s dispatch. Step 8 records each would-be-asked Unresolved decision as a deferred Unresolved row (Rationale `Deferred — promptless dispatch`) instead of asking, quoting the `_srad.md` § Critical Rule promptless-dispatch carve-out verbatim. Preserves `/fab-proceed`'s defer-and-surface contract exactly (see § Promptless Dispatch above).

The fork is legitimately **invocation-level** (who resolves ambiguity: human-now vs. defer-and-surface) — exactly parallel to the post-boundary autonomy fork (interactive rework menu vs. autonomous auto-rework).

**The three consumers are thin call-sites** (3xaj):

| Consumer | Call-site shape |
|----------|-----------------|
| `/fab-new` | `_intake(interactive)` + its Steps 10–11 activate/branch **tail** (stays at the call site) |
| `/fab-draft` | `_intake(interactive)`, stop at `ready` (no activate, no branch) |
| `/fab-proceed` | dispatches `_intake(promptless-defer)`; create-new dispatch rows chain `_intake` → `/fab-switch` → `/git-branch` for activate/branch parity (see § Promptless Dispatch) |

**What stays at the call site (the EXTRACTION BOUNDARY — deliberately NOT over-extracted):** (1) `/fab-new`'s activate (Step 10) + branch (Step 11) tail — a *different responsibility* (make active + checked out vs. queue it), not a questioning-mode parameter; (2) `/fab-proceed`'s state-detection + relevance-assessment, which decides *whether* to call `_intake` at all. `_intake.md` is purely "given I've decided to create an intake, do it (Steps 0–9)."

**Helper-declaration mechanics**: `_intake` is added to the `_preamble.md` § Skill Helper Declaration **Allowed-values allowlist** (now 7 values: `_generation`, `_review`, `_cli-fab`, `_cli-external`, `_srad`, `_pipeline`, `_intake` — see [_shared/context-loading.md](/_shared/context-loading.md) § Skill Helper Declaration). `fab-new.md` and `fab-draft.md` declare `helpers: [_generation, _srad, _intake]` (they keep `_generation`/`_srad` declared directly — the `_pipeline` precedent, where consumers declare the underlying helpers rather than inheriting transitively). `_intake.md` itself carries **no** `helpers:` frontmatter (matching `_pipeline`/`_review`/`_generation`), referencing `_generation` (Step 5) and `_srad` (Steps 4, 8) in-body and relying on the consumer having loaded them. `/fab-proceed` keeps **no** `helpers:` — it dispatches `_intake` as a subagent prompt (the subagent reads the helper), exactly as it dispatched `/fab-new` before. Step 5 of `_intake` delegates to `_generation.md` § Intake Generation Procedure rather than inlining artifact generation; Step 4's lifted text is genericized ("the invoking skill" rather than "this `/fab-new` invocation"), structurally retiring `fab-draft`'s former per-consumer self-name instruction.

> **Source-vs-deployed layout**: the canonical source is the **flat** `src/kit/skills/_intake.md` (every existing internal helper — `_pipeline.md`, `_generation.md`, `_srad.md`, `_review.md`, `_preamble.md` — is a flat `.md` in `src/kit/skills/`). The directory-per-skill `_intake/SKILL.md` form is the *deployed* copy `fab sync` writes under `.claude/skills/` (gitignored); skill *bodies* reference the deployed path (`.claude/skills/_intake/SKILL.md`) per the established `_preamble` convention. The canonical-source rule: never edit `.claude/skills/`.

### `/fab-continue [<change-name>] [<stage>]`

`/fab-continue` advances to the next pipeline stage — planning, implementation, review, or hydrate — and either generates the artifact or executes the stage's behavior. When called with a stage argument, it resets to that stage. When called with a change-name argument, it targets that change instead of the active one in `.fab-status.yaml` (transient — `.fab-status.yaml` is not modified). Both arguments can coexist; stage names are disambiguated first (fixed set of 6: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`), all other arguments are treated as change-name overrides. The pipeline flows intake → apply → review → hydrate → ship → review-pr.

A passed `tasks` or `spec` stage argument SHALL error immediately. `tasks` → `"tasks" stage was removed — run "fab status <event> <change> apply" instead. plan.md is now generated at apply entry.` `spec` → `"spec" stage was removed — spec.md is now generated at apply entry. Use "apply".` No alias window — the migration ensures no in-flight `.status.yaml` carries a `tasks` or `spec` key after upgrade.

#### Normal Forward Flow (no argument)

1. Read `.status.yaml` to determine current stage and state
2. **Consolidated dispatch**:
   - **intake `ready`**: Finish intake (auto-activates apply — `fab status finish` atomically marks intake `done` and activates the next pending stage; there is no separate `start apply` call, which the CLI rejects for a non-pending stage) → execute apply (apply's entry sub-step generates the unified `plan.md` with `## Requirements` + `## Tasks` + `## Acceptance`, then task execution begins)
   - **intake `active`** (backward compat for interrupted generations): Generate `intake.md`, advance to `ready` — the dispatch row instructs reading `.claude/skills/_generation/SKILL.md` first (point-of-use read; `_generation` is not in fab-continue's frontmatter `helpers:` — zc9m; see [execution-skills.md](/pipeline/execution-skills.md) § Stage-Conditional Helper Loading)
   - For execution stages (apply, review, hydrate, ship, review-pr): dispatch to the stage's behavior
3. Load relevant template + context (including `fab/project/constitution.md` for principles)
4. Generate the artifact using the shared Plan Generation Procedure from `_generation.md` (apply entry)
5. Update `.status.yaml`

There is no per-stage scoring step in `/fab-continue` — scoring happens only at intake, via `/fab-new` and `/fab-clarify` (j6cs).

#### Reset Behavior (with stage argument)

When called as `/fab-continue <stage>` (e.g., `/fab-continue apply`):
1. Target stage can be any of the 6 stages: `intake`, `apply`, `review`, `hydrate`, `ship`, `review-pr`. `tasks` and `spec` are rejected with the strict-error messages above.
2. Reset `.status.yaml` progress: set target stage to `active`; mark all stages after target as `pending`
3. For planning resets (intake), regenerate the artifact in place (update, not recreate — preserve what's still valid)
4. Downstream artifacts are invalidated where applicable: e.g., intake reset → apply pending → `plan.md` regenerated on next apply entry
5. Advance the target stage to `ready` for planning resets (not `done` — preserves `/fab-clarify` opportunity)

`fab status reset apply` modifies `.status.yaml` state only — `plan.md` persists on disk per the existing artifact-file convention. The apply skill's plan-generation sub-step is skipped on the next `/fab-continue` (idempotent on `plan.md` presence) and execution resumes from the first unchecked task. To force plan regeneration (including a fresh `## Requirements`), the user MUST delete `plan.md` before re-running `/fab-continue`.

Reset is primarily used after review identifies issues upstream — including requirement-level rework, which now edits `plan.md`'s `## Requirements` section and re-runs apply rather than resetting to a removed spec stage.

#### Context (varies by target stage)

- **Intake**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`
- **Apply** (plan generation sub-step): above + `intake.md` as the requirement-generation input

### `/fab-fff [<change-name>]` (Full Pipeline)

`/fab-fff` runs the entire Fab pipeline in a single invocation: apply (which generates the unified `plan.md` at entry — `## Requirements` + `## Tasks` + `## Acceptance` — then executes tasks) → review → hydrate → ship → review-pr. Gated on a **single intake confidence gate** (flat 3.0; j6cs — there is no spec step, spec gate, or auto-clarify invocation). Autonomously reworks on review failure with bounded retry (up to `{max_cycles}` cycles — the `Max cycles:` knob in `fab/project/code-review.md` § Rework Budget, default 3 (c5tr); escalation after 2 consecutive fix-code failures, a threshold that stays fixed). Accepts an optional change-name argument to target a specific change instead of the active one in `.fab-status.yaml`. Accepts `--force` to bypass the gate. As of szxd, `fab-fff.md` is a **thin wrapper over the shared pipeline bracket** `_pipeline.md` (declared via `helpers: [_generation, _review, _srad, _pipeline]`): the wrapper holds Purpose + Arguments + a two-row parameter table (`{driver}` = `fab-fff`, `{terminal}` = `review-pr`) plus the fff-only Steps 4–5 (ship, review-pr — incl. the timeout outcome) and driver-specific Output/error rows; the bracket holds everything the two drivers share — pre-flight (intake prerequisite + intake gate), context loading, resumability, Steps 1–3 (apply → review → hydrate), the auto-rework loop with its per-cycle choreography, the exhaustion stop, and the shared error rows with `{driver}`-parameterized messages.

#### Minimum Prerequisite

`intake.md` must exist (apply pending or later). `/fab-fff` is callable from any stage at or after intake — it picks up from the current stage and runs forward, skipping stages already `done`. The intake gate must pass unless `--force` is used. On entering apply, the condition is "if `progress.intake` is not `done`, finish intake" (`fab status finish <change> intake fab-fff`, auto-activating apply) — `/fab-new` leaves intake at `ready`, the normal hand-off state, and `finish` accepts both `active` and `ready`. If `progress.review` is `failed` (a prior exhaustion stop or an interrupted fail→reset sequence), the bracket's Resumability (`_pipeline.md`, szxd) runs `fab status start <change> review` first — the review-specific failed→active recovery transition — then resumes from the review step. This autonomous re-review recovery is the orchestrators' path only; a user-invoked `/fab-continue` on the same state presents the rework menu instead (see [execution-skills.md](/pipeline/execution-skills.md)).

#### No Auto-Clarify

There are no auto-clarify checkpoints (j6cs). The intake gate is the only "bounce" guard: a change whose intake scores below 3.0 cannot enter apply (non-`--force`), and the SRAD Critical Rule (Unresolved must be asked/bailed) applies at intake-time skills only. Inside apply, under-specified requirements are resolved inline as graded SRAD assumptions in `plan.md`'s `## Assumptions` — not via a clarify subagent. The pipeline is **resumable** — re-running `/fab-fff` skips stages already marked `done` and continues from the first incomplete stage.

#### Pipeline Flow

1. Resolve the active change (via `.fab-status.yaml` symlink); verify intake exists
2. Intake gate check (skip if `--force`)
3. Apply: plan generation sub-step writes the unified `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance` populated in one pass) → execute tasks
4. Validate implementation via review behavior — on failure, autonomously selects rework path (fix code, revise plan, revise requirements) and retries (max `{max_cycles}` cycles, default 3)
5. Hydrate into memory files
6. Ship (dispatch `/git-pr`)
7. Review-PR (dispatch `/git-pr-review`)

#### Autonomous Review Rework

On review failure, `/fab-fff` autonomously selects the rework path based on failure analysis (test failures → fix code, missing functionality → revise plan, requirement drift → revise `plan.md` `## Requirements` — there is no spec stage to reset to; j6cs). Maximum `{max_cycles}` rework cycles (the `Max cycles: {N}` line under `## Rework Budget` in `fab/project/code-review.md`, default 3 when the file/section/line is absent — c5tr). Escalation rule: after 2 consecutive "fix code" failures, the agent must escalate to "revise plan" or "revise requirements" (this threshold stays hard-coded). The per-cycle choreography is stated exactly once, in `_pipeline.md` § Auto-Rework Loop (szxd) — see [execution-skills.md](/pipeline/execution-skills.md) for the cycle sequence and the exhaustion terminal state (`review: failed`, no trailing reset). After `{max_cycles}` failed cycles, stops with a per-cycle summary pointing at `/fab-continue`, which presents the manual rework menu directly.

#### When to Use

- Want the full pipeline from intake through PR review in one command
- Clear requirements upfront, want to reach completion quickly with safety nets
- Changes needing a quality gate — the single intake gate blocks too-ambiguous changes before apply

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`.

### `/fab-ff [<change-name>]` (Fast-Forward, Gated)

`/fab-ff` runs the pipeline from intake through hydrate. Gated on a single intake confidence gate identical to `/fab-fff` (flat 3.0; j6cs). Autonomous rework on review failure (`{max_cycles}`-cycle cap, default 3; escalation after 2 consecutive fix-code). Accepts an optional change-name argument. Accepts `--force` to bypass the gate. As of szxd, `fab-ff.md` is the second **thin wrapper over `_pipeline.md`** (same `helpers:` declaration as `/fab-fff`): Purpose + Arguments + the parameter table (`{driver}` = `fab-ff`, `{terminal}` = `hydrate` — the pipeline ends after the bracket's Step 3) and a driver Output block; it adds no driver-specific error rows. Gate terminology follows the constitution's single-intake-gate framing, and the post-bail intake-deepening guidance lives in the bracket's shared stop text for both drivers — the executable, override-aware `/fab-continue <change> intake` → `/fab-clarify <change>` route (szxd/w7dp; see [execution-skills.md](/pipeline/execution-skills.md)).

#### Minimum Prerequisite

`intake.md` must exist. `/fab-ff` is callable from any stage at or after intake — it picks up from the current stage and runs forward through hydrate, skipping stages already `done`. The intake gate must pass unless `--force` is used. The same two entry guards as `/fab-fff` apply (both defined once in `_pipeline.md` since szxd): "if `progress.intake` is not `done`, finish intake" (auto-activates apply), and if `progress.review` is `failed` (prior exhaustion stop or interrupted fail→reset), run `fab status start <change> review` first before resuming from the review step.

#### Confidence Gate

`/fab-ff` has a single confidence gate identical to `/fab-fff`: the intake gate — confidence >= 3.0 (flat for all types) via `fab score --check-gate --stage intake`. The gate is skipped when `--force` is passed.

#### No Auto-Clarify

There are no auto-clarify checkpoints (j6cs); the intake gate is the only guard. Inside apply, under-specified requirements become inline graded SRAD assumptions in `plan.md`'s `## Assumptions`.

#### Pipeline Flow

1. Resolve the active change (via `.fab-status.yaml` symlink); verify intake exists
2. Intake gate check (skip if `--force`)
3. Apply: plan generation sub-step writes the unified `plan.md` (`## Requirements` + `## Tasks` + `## Acceptance`) → execute tasks
4. Validate implementation via review behavior — on failure, autonomous rework (`{max_cycles}`-cycle cap, escalation rule)
5. Hydrate into memory files

#### Autonomous Review Rework

On review failure, `/fab-ff` autonomously selects the rework path based on failure analysis (same behavior as `/fab-fff` — literally the same text since szxd: both run `_pipeline.md` § Auto-Rework Loop). Maximum `{max_cycles}` rework cycles (default 3). Escalation rule: after 2 consecutive "fix code" failures, the agent must escalate. After `{max_cycles}` failed cycles, stops with a per-cycle summary, leaving `review: failed` as the terminal state (see [execution-skills.md](/pipeline/execution-skills.md)).

#### Resumability

`/fab-ff` is resumable — re-invoking skips stages already marked `done` and continues from the first incomplete stage.

#### Confidence Recomputation

`/fab-ff` does NOT recompute the confidence score during execution. The gate check uses the persisted intake score from the last manual step (`/fab-new` or `/fab-clarify`).

#### When to Use

- Small, well-understood changes that don't need ship/review-pr
- Want to reach hydrate quickly with safety nets (intake gate + auto-rework)
- After raising confidence via `/fab-clarify` to meet the threshold

#### Context

Loads all planning context upfront: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`.

### `/fab-clarify [<change-name>]`

`/fab-clarify` deepens and refines the **intake** artifact without advancing. It is **intake-only** (j6cs): there are no `spec` or `plan` targets — under-specified requirements at apply become inline SRAD assumptions in `plan.md`, not clarify sessions. It operates in two modes depending on call context: **suggest mode** (user invocation) and **auto mode** (machine-readable result; no orchestrator currently invokes it). It is idempotent and non-advancing. Accepts an optional change-name argument to target a specific change instead of the active one in `.fab-status.yaml`. See [clarify.md](/pipeline/clarify.md) for the detailed dual-mode specification.

#### Suggest Mode (User Invocation)

When the user invokes `/fab-clarify` directly:

1. Read `.status.yaml` to determine current stage
2. Stage MUST be `intake` (`progress.intake` in `{active, ready, done}`). At apply or later, `/fab-clarify` STOPs with a pointer to `/fab-continue` for rework or editing `plan.md`'s `## Requirements` directly. A passed `spec`/`plan`/`tasks` argument is treated as a change-name (no such targets exist).
3. Load `intake.md` + relevant context
4. **Bulk confirm check** (Step 1.5): Parse the `## Assumptions` table. If `confident >= 3` AND `confident > tentative + unresolved`, display all Confident assumptions as a numbered list for conversational bulk response (confirm/change/explain). After resolution, proceed to step 5. See [clarify.md](/pipeline/clarify.md) for full details.
5. Perform a **stage-scoped taxonomy scan** for gaps, ambiguities, and `[NEEDS CLARIFICATION]` markers (categories vary by stage)
6. Present structured questions **one at a time** (max 5 per invocation), each with a recommendation and options table or suggested answer
7. **Immediately update the artifact** after each user answer (incremental, not batched)
8. User may terminate early with "done"/"good"/"no more"
9. Append audit trail under `## Clarifications > ### Session {date}` with `Q:` / `A:` entries
10. Display coverage summary (Resolved / Clear / Deferred / Outstanding)
11. Do NOT advance the stage

#### Auto Mode (Machine-Readable)

Auto mode (the `[AUTO-MODE]` prefix) operates on `intake.md` only and returns `{resolved: N, blocking: N, non_blocking: N}`. As of j6cs no orchestrator invokes it — `/fab-ff`/`/fab-fff` dropped their auto-clarify steps — but the mode is retained for future use:

1. Perform the intake taxonomy scan autonomously — no user interaction
2. Resolve gaps using available context; classify remaining gaps as blocking or non-blocking
3. Return the machine-readable result and recompute the intake score

#### Key Property

Calling `/fab-clarify` multiple times is safe — it refines the intake further each time. It never transitions to the next stage. Use `/fab-continue` when satisfied.

#### Context

- **Intake**: config, constitution, `intake.md`, target memory file(s) from `docs/memory/`

## Design Decisions

### SRAD Autonomy Framework
**Decision**: All planning skills use the SRAD framework (Signal Strength, Reversibility, Agent Competence, Disambiguation Type) to evaluate decision points and assign confidence grades (Certain, Confident, Tentative, Unresolved). All four grades are recorded in every Assumptions table with a required Scores column (`S:nn R:nn A:nn D:nn`). Unresolved rows include status context in Rationale. Each skill has a defined autonomy level and interruption budget. Dimensions are evaluated on a continuous 0–100 scale, aggregated into a composite via weighted mean — **`0.20·S + 0.30·R + 0.30·A + 0.20·D` (4yi8)** — and mapped to **indicative-only** grades via half-open bands: **composite ≥ 80 Certain, 50 ≤ c < 80 Confident, 20 ≤ c < 50 Tentative, else Unresolved (4yi8 — aligned with the demerit penalty curve's knees)**. The grade is **derived from the composite and never read by any formula** — it can never contradict its own dimensions. **There is no Critical-Rule override and no hard-fail short-circuit (4yi8)**: blocking is **emergent from the demerit curve** (a `composite < 20` row penalizes ≥ 2.0), and reversibility is carried by R's 0.30 composite weight rather than a separate rule. The autonomy table's four columns cover all six declaring skills via a covering note (c5tr): fab-draft follows the fab-new column exactly (thin delta), and fab-clarify is the escape valve itself (SRAD-prioritized questions, re-grades resolved rows, always recomputes the intake score). Worked-example arithmetic is internally consistent: Example 1 (high-ambiguity) blocks via the curve, Example 2 (low-ambiguity) scores 5.0, and Example 3 surfaces a single risky decision through R's weight — none rely on a hard-fail. The framework **lives in the dedicated `_srad.md` helper** (zc9m; `_preamble.md` keeps a ~3-line pointer), declared via frontmatter `helpers:` by exactly the six planning skills — `fab-new`, `fab-draft`, `fab-continue`, `fab-ff`, `fab-fff`, `fab-clarify`. Non-planning skills do not load it. The Critical Rule carries a **promptless-dispatch carve-out** (w7dp) cross-referencing fab-proceed's defer-and-surface contract: under a promptless dispatch the MUST-ask is satisfied by recording the decision as a `Deferred — promptless dispatch` Unresolved row and surfacing it (the intake gate backstops — a genuine unknown scored with honestly-low dimensions lands at `composite < 20` and the demerit curve's ≥ 2.0 penalty blocks the gate; 4yi8); wherever a user is reachable, the MUST-ask applies unchanged — resolving the contradictory MUSTs a subagent loading both files would otherwise receive.
**Why**: Replaces ad-hoc question selection with a principled, consistent framework. Ensures high-blast-radius decisions are always surfaced while low-value prompts are eliminated. The four-dimension scoring prevents both over-asking and silent high-risk assumptions. The R-biased weighting (0.30, alongside A at 0.30 — 4yi8) carries the Critical Rule's intent at the formula level — an irreversible decision lands in a worse composite band and is penalized harder automatically, so no separate override is needed. The zc9m extraction keeps the framework out of the always-load layer — only the skills that grade decisions pay for it.
**Rejected**: Ad-hoc question selection — inconsistent, no way to predict agent behavior. Full autonomy — too risky for Unresolved decisions with cascading consequences. Binary high/low dimension classification — lost nuance in the mid-range. Interactive relay / `[AUTO-MODE]` adoption for the promptless-dispatch carve-out — protocol surface with no requesting use case (w7dp).
*Introduced by*: 260207-09sj-autonomy-framework; *Updated by*: 260212-f9m3-enhance-srad-fuzzy (fuzzy 0–100 dimensions, weighted mean aggregation, dynamic gate thresholds); 260611-zc9m-preamble-context-diet (framework extracted to `_srad.md`, declared by the 6 planning skills; Worked Example 1 compressed to one-liner style); 260612-c5tr-scaffold-config-truth-srad-coherence (half-open grade thresholds, single `R<25 AND A<25` Critical-Rule citation, autonomy covering note for fab-draft/fab-clarify, worked-example arithmetic fixed, omit-when-zero scoped to displayed output only); 260612-w7dp-orchestrator-dispatch-review-pr-recovery (Critical-Rule promptless-dispatch carve-out cross-referencing fab-proceed's defer-and-surface contract); 260618-4yi8-srad-v2-demerit-scoring (SRAD v2 demerit confidence scoring: composite weights → `0.20/0.30/0.30/0.20`, grade bands → 80/50/20 and indicative-only, both hard-fail short-circuits removed — blocking emergent from the penalty curve)

### Intake-First with SRAD-Based Questions
**Decision**: Every change starts with an intake. The agent applies SRAD scoring to identify up to 3 Unresolved decisions with the highest blast radius and asks those. All other decisions are assumed at their assessed confidence grade and surfaced in the Assumptions summary.
**Why**: Prevents question-paralysis while catching the decisions that actually matter. SRAD scoring replaces gut-feel question selection with a repeatable evaluation method.
**Rejected**: Unlimited clarification rounds — too many back-and-forth exchanges. Fixed 3-question cap without SRAD — may ask the wrong 3 questions.
*Source*: doc/fab-spec/TEMPLATES.md, doc/fab-spec/README.md; *Updated by*: 260207-09sj-autonomy-framework

### No Frontloaded Questions in Pipeline Skills
**Decision**: Neither `/fab-ff` nor `/fab-fff` frontloads questions. Both proceed directly into apply, relying on the single intake confidence gate to block changes that are too ambiguous.
**Why**: Frontloaded questions interrupted the autonomous pipeline flow. The intake gate provides a better signal — if the intake has too many unresolved decisions, the gate blocks and the user resolves via `/fab-clarify` before retrying. This is cleaner than a mid-pipeline Q&A round.
**Rejected**: Previous design had `/fab-fff` frontloading questions — this interrupted the autonomous flow unnecessarily.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260215-237b-DEV-1027-redefine-ff-fff-scope (moved from fab-ff to fab-fff); 260314-q5p9-redesign-ff-fff-scopes (dropped frontloaded questions entirely); 260601-j6cs-merge-spec-into-apply (single intake gate after spec stage removal)

### Clarify is Non-Advancing
**Decision**: `/fab-clarify` never transitions to the next stage. It refines in place.
**Why**: Separates the concerns of "deepen the current work" from "move forward." The user explicitly chooses when to advance via `/fab-continue`.
**Rejected**: Auto-advancing after clarification — unclear when the user considers the artifact ready.
*Source*: doc/fab-spec/SKILLS.md

### Clarify Mode Selection by Call Context
**Decision**: `/fab-clarify` mode is determined by the `[AUTO-MODE]` prefix defined in the Skill Invocation Protocol — since 260611-zc9m defined in `fab-clarify.md` § Skill Invocation Protocol itself (its sole referencer; `_preamble.md` keeps a 2-line pointer). When the prefix is present (e.g., an orchestrator invoking internally), `/fab-clarify` enters auto mode. When absent (user invocation), it enters suggest mode. No `--suggest`/`--auto` flags.
**Why**: Avoids a confusing flag pair with no clear use case for user-invoked auto mode. The explicit prefix protocol makes the contract testable rather than relying on implicit call-context interpretation.
**Rejected**: Flag-based mode selection — adds complexity, no user scenario requires it. Implicit call-context detection — unreliable, not testable.
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260210-nan4-define-auto-mode-signaling (explicit `[AUTO-MODE]` protocol); 260611-zc9m-preamble-context-diet (protocol definition relocated from `_preamble.md` into `fab-clarify.md`)

### Pipeline Skills Drop Auto-Clarify (j6cs)
**Decision**: Neither `/fab-ff` nor `/fab-fff` invokes `/fab-clarify` automatically. Both auto-clarify checkpoints (the former post-spec and on-plan invocations) were removed when the spec stage was merged into apply. The single intake confidence gate is the only "bounce" guard.
**Why**: With one manual stage (intake), all human judgment is frontloaded there and gated once at flat 3.0. There is no mid-pipeline stage boundary left for an auto-clarify to sit on. Inside apply, under-specified requirements are resolved inline as graded SRAD assumptions in `plan.md`'s `## Assumptions` — the apply agent doesn't need a separate clarify subagent. The independent assumption re-grade the old spec stage performed is accepted as lost, compensated by the flat-3.0 gate (≥ every old gate), the ~1% spec-rework loom evidence, and requirement-correctness still being caught at review.
**Rejected**: Keeping an apply-entry auto-clarify (re-adds the ceremony the merge removes). A runtime "apply detects Unresolved → reset to intake" mechanism (unbuilt, redundant with the existing gate).
*Introduced by*: 260207-m3qf-clarify-dual-modes; *Updated by*: 260208-k3m7-add-fab-fff; 260215-237b-DEV-1027-redefine-ff-fff-scope; 260314-q5p9-redesign-ff-fff-scopes; 260423-qszh-merge-tasks-checklist (auto-clarify tasks → auto-clarify plan); 260601-j6cs-merge-spec-into-apply (both auto-clarify checkpoints removed)

### /fab-new as Single Adaptive Entry Point
**Decision**: Consolidate `/fab-new` and `/fab-discuss` into a single `/fab-new` that adapts via SRAD scoring.
**Why**: Three overlapping entry paths created confusion.
**Rejected**: Keeping both skills with clearer differentiation.
*Introduced by*: 260212-v5p2-simplify-stages-entry-paths

### Stage Rename: "brief" → "intake"
**Decision**: The first pipeline stage is named `intake` (not `brief`). The artifact is `intake.md`. The Intake Generation Procedure lives in `_generation.md` (which, after j6cs, contains only this and the unified Plan Generation Procedure), with an explicit generation rule emphasizing the intake is a state transfer document.
**Why**: "Brief" in English means short/concise, which triggered summarization instincts in LLMs — agents consistently produced thin briefs despite template instructions. "Intake" signals thorough initial collection in professional contexts (legal, medical, project management). The generation rule in `_generation.md` provides defense in depth alongside the name change.
**Rejected**: `handoff` — could describe any stage boundary. `charter` — too formal. Generation rule only without rename — doesn't fix the misleading name.
*Introduced by*: 260215-v4n7-DEV-1025-rename-brief-to-intake

### Shared Generation Partial
**Decision**: Artifact generation logic lives in a single shared `_generation.md` partial. After j6cs, the partial defines two procedures: the **Intake Generation Procedure** and the unified **Plan Generation Procedure** (invoked by apply at entry, emitting `## Requirements` + `## Tasks` + `## Acceptance` in one walk). The standalone Spec Generation Procedure was deleted — requirement generation folded into plan generation. Each consumer retains its own orchestration logic.
**Why**: Generation steps were nearly identical across skills, requiring every fix or behavior change to be applied in two places. Centralizing eliminates drift and makes generation behavior authoritative in one location. With qszh the Tasks + Checklist procedures collapsed into one Plan Generation Procedure; with j6cs the Spec Generation Procedure folded into it too, so a single pass over the intake-derived design emits requirements, tasks, and acceptance — the strongest possible alignment guarantee (single skill call, single context window).
**Rejected**: Keeping inline duplication — inevitable drift between the copies. Keeping a separate Spec Generation Procedure and `spec.md` — reintroduces the seam the merge removes and leaves an unread file (nothing reads `spec.md` programmatically once the gate moves to intake).
*Introduced by*: 260210-wpay-extract-shared-generation-logic; *Updated by*: 260423-qszh-merge-tasks-checklist (Tasks + Checklist procedures → unified Plan Generation Procedure); 260601-j6cs-merge-spec-into-apply (Spec Generation Procedure folded into Plan Generation; `## Requirements` co-generated at apply entry)

### Pre-Boundary Intake-Creation Extracted to `_intake` (3xaj)
**Decision**: `/fab-new` Steps 0–9 (the "create an intake" procedure) are lifted into a single internal helper `_intake.md`, parameterized by one knob `{questioning-mode}` (`interactive` | `promptless-defer`) applied only at Step 8. The three consumers (`/fab-new`, `/fab-draft`, `/fab-proceed`) become thin call-sites. The activate/branch tail (`/fab-new` Steps 10–11) and `/fab-proceed`'s state-detection stay at the call site (the EXTRACTION BOUNDARY — do NOT over-extract). This is a pure restructure: behavioral parity is the bar, zero Go code touched.
**Why**: Steps 0–9 were duplicated across the pre-boundary family via **two inconsistent reuse mechanisms both pointing at `fab-new.md`** — `/fab-draft` was a fragile prose delta (carrying a "don't activate/branch by momentum" warning precisely because the steps it must NOT run shared the body it executed), and `/fab-proceed` dispatched the full `/fab-new` skill as a promptless subagent. So `fab-new.md` was simultaneously a skill AND the de-facto shared library two others reached into. Extraction mirrors the proven `_pipeline.md` shape (shared body + one knob + call-site tails) and completes the symmetry: one shared helper per pipeline phase — `_generation` (artifact mechanics), `_review` (review mechanics), `_pipeline` (post-intake orchestration), `_intake` (pre-intake orchestration). The momentum warning **evaporates** structurally (the not-to-run steps no longer live in any body `/fab-draft` executes), and Step 4's self-name references are genericized, retiring `/fab-draft`'s per-consumer self-name instruction.
**Rejected**: Transitive inheritance of `_generation`/`_srad` through `_intake` (the `_pipeline` precedent has consumers declare underlying helpers directly — kept). A `{self-name}` parameter for Step 4 (the text only needs to be invocation-agnostic, not invocation-named — user-endorsed genericize-over-parameterize). Over-extracting activate/branch or proceed's state-detection into `_intake` (recreates the dual-mode problem on the intake side). The flat canonical-source layout `src/kit/skills/_intake.md` was chosen over `_intake/SKILL.md` (the latter is the *deployed* form `fab sync` writes; every existing canonical helper is a flat `.md`).
*Introduced by*: 260613-3xaj-extract-intake-helper

### Plan Generation Lives at Apply Entry, Not in a Separate Stage (qszh)
**Decision**: The `tasks` stage is removed from the pipeline. Plan generation (writing `plan.md` with `## Tasks` + `## Acceptance`) is an entry sub-step of the apply skill, not a stage gate. Pipeline goes from 8 stages (intake → spec → tasks → apply → review → hydrate → ship → review-pr) to 7 stages (intake → spec → apply → review → hydrate → ship → review-pr). `progress.tasks` is dropped from `.status.yaml` entirely — no rename to `progress.plan`, since with no separate stage there is no key to populate.
**Why**: The `tasks` stage was a no-decision gate. `/fab-continue` advanced spec → tasks → apply back-to-back; users never stopped at tasks. Every change paid the wall-time, token, and `.status.yaml` cost of a transition that had no decision content. Folding generation into apply makes drift between Tasks and Acceptance mechanically impossible (single skill call, single context window, single LLM, single file). The qszh collapse stopped at `tasks` and preserved the spec stage; j6cs later folded the spec stage in too (see "Spec Stage Merged into Apply" below), so apply entry now co-generates `## Requirements` alongside `## Tasks` + `## Acceptance`.
**Rejected**: (a) Keep two files, add cross-check step — adds ceremony, not less. (b) Drop checklist, review reads tasks directly — loses the imperative-vs-declarative framing review depends on. (c) Merge artifacts only, keep `tasks` stage — fixes drift but keeps the no-decision gate; xvaz workaround remains relevant. (d) Drop both spec AND tasks — loses `/fab-clarify spec`, breaks per-type spec gate, weakens review. (e) Rename `tasks` → `plan` stage — same gate, new name. (f) Rename `apply` → `execute`/`implement` — semantic gain doesn't justify migration churn across state table, `.status.yaml`, all skills, muscle memory.
*Introduced by*: 260423-qszh-merge-tasks-checklist; *Supersedes*: the proposal in `260423-xvaz-skip-tasks-simple-types` (per-type skip policy for the tasks stage) — that draft becomes obsolete by construction once qszh ships, since there is no separate stage to skip. Simple changes naturally produce a tiny `plan.md` and execute in seconds; no skip policy needed. The xvaz folder will be archived by a separate user-initiated `/fab-archive 260423-xvaz...` action.

### Spec Stage Merged into Apply; `plan.md` Carries `## Requirements` (j6cs)
**Decision**: The `spec` stage is removed from the pipeline (7 → 6 stages: intake → apply → review → hydrate → ship → review-pr). The requirement discipline that lived in a separate `spec.md` artifact (RFC-2119 + GIVEN/WHEN/THEN) is absorbed into `plan.md` as a `## Requirements` section, co-generated at apply entry alongside `## Tasks` + `## Acceptance`. The `spec.md` template is removed; `spec.md` no longer exists as an artifact. The canonical artifact set becomes `intake.md → plan.md → code`. Any `fab status` event targeting `spec` hard-errors (mirroring the `tasks` branch). SRAD confidence scoring is frontloaded to intake as the **single** confidence gate (flat 3.0 for all types); the per-type spec gate and the `confidence.indicative` flag are retired.
**Why**: Empirical loom evidence (spec median ~2 min, ~1% rework, 32% of clarify) showed the spec stage was a near-pass-through. Once the confidence gate moves to intake, nothing reads `spec.md` programmatically (verified: score/hook/artifact-matcher/git-pr all removed or repointed) — a separate file would be generated, never machine-read, dead weight. One-pass co-generation of requirements + tasks + acceptance is the strongest alignment guarantee. The intake gate IS the bounce-back valve: a failing intake never reaches `done`, so gate-checking orchestrators cannot enter apply — no new runtime "reset to intake" mechanism is needed.
**Rejected**: Keeping `spec.md` as a separate hidden apply-entry artifact (reintroduces the seam, leaves an unread file). Per-type single gate at 2.0/3.0 (would relax the entry bar for 5 of 7 types — flat 3.0 keeps every type ≥ both old gates). A runtime apply-side "detect Unresolved → reset to intake" mechanism (unbuilt, redundant with the gate). Adding an independent assumption re-grade at apply entry (re-adds ceremony; deferred unless review evidence warrants).
*Introduced by*: 260601-j6cs-merge-spec-into-apply

### Strict-Error Stance for Legacy `tasks` References (qszh)
**Decision**: All `tasks` stage references and the legacy `set-checklist` CLI command error immediately with a helpful pointer message — no alias window, no phased deprecation. `fab status start|advance|finish|reset|skip|fail <change> tasks` returns exit 1 with `"tasks" stage was removed — run ... apply instead. plan.md is now generated at apply entry.` `fab status set-checklist` returns exit 1 with `"set-checklist" is now "set-acceptance" — run fab status set-acceptance instead.` `/fab-clarify tasks` errors with a similar pointer.
**Why**: The 1.8.0-to-1.9.0 migration rewrites every in-flight `.status.yaml` so no live change carries a `tasks` key after upgrade. With no live `tasks` state to support, an alias adds maintenance burden for zero user benefit. Strict errors with pointer messages are self-documenting and steer users toward the new workflow immediately.
**Rejected**: Phased deprecation (alias for one release, error in next) — no in-flight `.status.yaml` carries `tasks` after migration, so phasing buys nothing. Silent renaming — leaves users uncertain whether a command did what they expected.
*Introduced by*: 260423-qszh-merge-tasks-checklist

### Unified Command: `/fab-continue` Absorbs Execution Stages
**Decision**: `/fab-continue` handles all 6 pipeline stages (intake → apply → review → hydrate → ship → review-pr). Apply, review, hydrate, ship, and review-pr behaviors are described as dedicated sections within `fab-continue.md`, not extracted into a shared partial. `/fab-archive` exists as a standalone housekeeping skill (not a pipeline stage) for post-hydrate cleanup. The apply behavior includes a Plan Generation sub-step at entry (writes the unified `plan.md`); see [execution-skills.md](/pipeline/execution-skills.md).
**Why**: Reduces developer command surface to 2 (`/fab-continue` + `/fab-clarify`). Execution stages are orchestration-heavy with distinct flows (plan generation + task execution, validation with rework, memory hydration) — inlining keeps each stage's behavior in one readable location.
**Rejected**: Keeping standalone `/fab-apply`, `/fab-review` — command fragmentation. Extracting to `_execution.md` partial — low reuse value since only fab-continue calls these. Splitting plan generation into a `/fab-plan` skill — adds a command surface for what is mechanically a single autonomous step at apply entry.
*Introduced by*: 260212-a4bd-unify-fab-continue; *Updated by*: 260303-he6t-extend-pipeline-through-pr (added ship + review-pr); 260423-qszh-merge-tasks-checklist (dropped tasks stage; plan generation folded into apply entry — 7 stages); 260601-j6cs-merge-spec-into-apply (dropped spec stage — 6 stages)

### `/fab-ff` and `/fab-fff` Keep Behavioral Descriptions
**Decision**: `/fab-ff` and `/fab-fff` describe execution behavior in their own orchestration context, rather than literally invoking `/fab-continue` as a sub-skill. Since szxd that description lives once in the shared `_pipeline.md` bracket (parameterized by `{driver}`/`{terminal}`) rather than duplicated inline in each file — but the principle stands: the orchestrators dispatch `/fab-continue` *behavior sections* as subagents and own all status transitions; they never invoke `/fab-continue` as a skill.
**Why**: These skills have fundamentally different orchestration: bail behavior, autonomous rework, resumability across all stages. Literal sub-skill invocation would add complexity (nested preflight checks, status conflicts) without benefit. The szxd extraction removed the remaining cost of this stance — the two files shared ~88% of their content verbatim, and every bracket edit had to land twice (with documented drift: "Two gates" vs the single intake gate).
**Rejected**: Literal `/fab-continue` invocation from fab-ff/fff — orchestration mismatch, nested state management issues. Keeping the bracket duplicated per wrapper — re-creates the drift surface.
*Introduced by*: 260212-a4bd-unify-fab-continue; *Updated by*: 260611-szxd-skills-twins-self-duplication-refactor (shared bracket extracted to `_pipeline.md`; wrappers shrank to parameter tables + driver-specific content)

### Scope Differentiation: fab-fff (Full Pipeline) vs fab-ff (Fast-Forward)
**Decision**: The difference between `/fab-ff` and `/fab-fff` is scope only. `/fab-ff` runs intake → hydrate; `/fab-fff` extends through ship → review-pr. Both have the identical single intake confidence gate, identical autonomous rework (`{max_cycles}`-cycle cap, escalation rule), and accept `--force` to bypass the gate. No frontloaded questions in either skill.
**Why**: The naming intuition: `ff` (fast-forward) = "get me to hydrate quickly." `fff` (fast-forward-further) = "go all the way through PR review." Scope is the only axis of differentiation — behavior (gate, rework) is identical. This simplifies the mental model: choose ff or fff based on how far you want to go, not based on behavioral differences.
**Rejected**: Previous design differentiated on behavior (gates, frontloaded questions, rework style) — too many axes of variation, confusing mental model.
*Introduced by*: 260215-237b-DEV-1027-redefine-ff-fff-scope; *Updated by*: 260216-knmw-DEV-1030-swap-ff-fff-review-rework (swapped review failure behavior); 260314-q5p9-redesign-ff-fff-scopes (scope-only differentiation, identical gates on both, no frontloaded questions)

### ID-Collision Re-Runs Route to Resume, Not Error (g1-2)
**Decision**: When `/fab-new` or `/fab-draft` is re-run with a backlog or Linear ID that already has an existing non-archived change, the skill detects the collision in Step 3 (backlog: `fab resolve --id {id}` compared for **equality** with `{id}` — exact-ID anchored since w7dp, so a substring hit inside another change's slug no longer false-positives into resume; Linear: `grep -lw "{ISSUE_ID}" fab/changes/*/.status.yaml` over the issues arrays — word-anchored since uliv so `DEV-123` doesn't match `DEV-1234`; Linear IDs never appear in folder names) and routes to resume (`/fab-switch {name}` + `/fab-continue`) instead of surfacing the raw `Change ID already in use` error. `change.go` keeps its collision error unchanged as the safety net (backlog IDs only). Natural-language re-runs intentionally create a new change each run.
**Why**: Constitution III (skills MUST be safe to re-run) without a Go behavior change — detection plus a recovery pointer is skill-level work, and `/fab-continue`'s intake-`active` row already regenerates a missing intake for interrupted creations. The Linear branch must scan issues arrays because folder names cannot carry the signal (slugs exclude issue IDs since 260226-jq7a).
**Rejected**: Making `fab change new` itself idempotent — hides a real collision signal other callers rely on. Detecting Linear collisions via `fab change resolve {id}` — matches folder names only, so the check could never fire.
*Introduced by*: 260611-9u91-skills-correctness-idempotency-fixes; *Updated by*: 260611-uliv-skills-staleness-sweep-frontmatter-fixes (collision grep word-anchored with `grep -lw`); 260612-w7dp-orchestrator-dispatch-review-pr-recovery (backlog pre-check exact-ID anchored — `fab resolve --id` equality compare, substring slug hits no longer route to resume)

### Change-Type Inference Is Pull-Based, Explicit Set Is Sticky (g3-4; jznd; y022)
**Decision**: `fab status refresh` (`internal/refresh.Refresh`) is the single writer for *inferred* `change_type` — it infers and writes the type whenever `change_type_source` is absent or `inferred` (word-boundary keyword regexes incl. `redesign`, first match wins, default `feat`), self-healed at the transition seams (`fab status advance`/`finish`, `fab preflight`) rather than fired by a PostToolUse hook on every `intake.md` write (the hook was removed in y022 — a hook fires only in the Claude harness, so this correctness-critical state had to become pull-based). `/fab-new` and `/fab-draft` no longer run a manual keyword-matching step or an unconditional `fab status set-change-type`; they verify the recomputed value by reading `change_type` from the change's `.status.yaml` (preflight does not emit it) and override only if wrong. **As of 260615-jznd a human override is sticky**: `fab status set-change-type` writes `change_type_source: explicit` alongside the type, and refresh skips both inference and the type overwrite when the source is `explicit` (other bookkeeping still runs). An absent/empty source decodes as `inferred` (back-compat), so re-inference on every refresh is unchanged for pre-jznd changes — only an explicit set turns it off.
**Why**: The skills' manual keyword list conflicted with the hook's regexes (the hook includes `redesign`, the skill list didn't) and the hook re-fired on every intake write, silently overwriting the skill-set value — a double-write race the old text papered over. Making the hook (and later, its pull-based y022 successor) the owner of *inference* removed the keyword-list divergence; the `change_type_source` guard (jznd) then removed the remaining overwrite race at its root, so the skill-side override is durable rather than a value the next write reverts. y022 moved the recompute off a Claude-harness-only hook so the same guarantee holds for a sed edit or a non-Claude agent. The skill-side step stays verification + recovery, the only part that needs agent judgment.
**Rejected**: Keeping the skill-side inference and documenting the overwrite race — leaves two divergent keyword lists and a known race. (The g3-4-era rejection "making the hook respect a skill-set value is a Go behavior change out of scope for a docs/skills sweep" was the correct call *for that change* — it was later shipped deliberately as the in-scope Go change jznd, with an `inferred|explicit` enum so the writer precedence is explicit rather than ambiguous.)
*Introduced by*: 260611-uliv-skills-staleness-sweep-frontmatter-fixes (g3-4 hook ownership of inference); *Updated by*: 260615-jznd-binary-truth-telling (`change_type_source: inferred|explicit` guard — explicit human set survives subsequent intake re-infer writes; `fix` regex tightened so a passing `must-fix` no longer misclassifies a feature intake); 260702-y022 (recompute moved from the PostToolUse `artifact-write` hook to the pull-based `fab status refresh`, self-healed at the transition seams — same inference/guard logic, different trigger)
*Introduced by*: 260611-uliv-skills-staleness-sweep-frontmatter-fixes

### Reset via `/fab-continue <stage>`
**Decision**: Reset to any pipeline stage by passing the stage name as an argument to `/fab-continue`. For the intake planning stage, the artifact is invalidated and regenerated. For execution stages, the stage behavior is re-run without resetting task checkboxes. `tasks` and `spec` are rejected with strict-error pointers to `apply` — both stages were removed (qszh and j6cs respectively).
**Why**: Provides a clean re-entry point after review identifies upstream issues. Reuses the existing skill rather than adding a separate `/fab-reset` command. Covers all 6 stages (intake, apply, review, hydrate, ship, review-pr). `fab status reset apply` preserves `plan.md` on disk; the apply entry sub-step skips regeneration when the file exists, so users who want a fresh plan (including a fresh `## Requirements`) must delete `plan.md` before re-running.
**Rejected**: Separate reset skill — unnecessary proliferation of skills for a rare operation. Auto-deleting `plan.md` on apply reset — violates the existing artifact-file convention (reset modifies `.status.yaml` state only; artifact files persist) and Constitution III idempotency.
*Source*: doc/fab-spec/SKILLS.md; *Updated by*: 260212-a4bd-unify-fab-continue (extended to all 6 stages); 260303-he6t-extend-pipeline-through-pr (extended to ship + review-pr — 8 stages); 260423-qszh-merge-tasks-checklist (dropped tasks stage — 7 stages; documented `reset apply` plan.md preservation); 260601-j6cs-merge-spec-into-apply (dropped spec stage — 6 stages; `spec` reset target rejected)
