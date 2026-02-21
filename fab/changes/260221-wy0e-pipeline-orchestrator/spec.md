# Spec: Pipeline Orchestrator

**Change**: 260221-wy0e-pipeline-orchestrator
**Created**: 2026-02-21
**Affected memory**: `docs/memory/fab-workflow/pipeline-orchestrator.md`

## Non-Goals

- Parallel dispatch of independent changes — v1 is serial; parallel execution is a documented stretch goal
- Merging to main — the human handles all merges
- Byobu pane integration — stretch goal, not part of this change
- Partial dependency (starting B when A reaches a mid-pipeline stage) — all dependencies must reach `done`
- Modifying existing skills — the orchestrator composes existing tools

## Pipeline: Manifest Format

### Requirement: YAML Manifest Schema

The pipeline manifest SHALL be a YAML file stored in `fab/pipelines/` with the following structure:

```yaml
base: <branch-name>

changes:
  - id: <change-folder-name>
    depends_on: []
    stage: <stage-value>
```

The manifest MUST contain a `base` field (string, the branch root nodes branch from) and a `changes` list. Each entry MUST have `id` (string, matching a folder under `fab/changes/`) and `depends_on` (list of change IDs, may be empty). The `stage` field is OPTIONAL — absent means "not started". The `stage` field is written exclusively by the orchestrator.

**Dependency constraint**: `depends_on` MUST contain at most one entry in v1. Multi-parent dependencies (diamond DAGs) require the human to merge parent branches before the dependent change can proceed. The orchestrator SHALL reject entries with more than one `depends_on` item with: "Multi-parent dependency not supported in v1: <id> depends on <N> changes. Merge parent branches manually."
<!-- clarified: single-dependency restriction for v1 — avoids branch topology problem with multi-parent DAGs -->

Valid `stage` values: `intake`, `spec`, `tasks`, `apply`, `review`, `hydrate`, `done`, `failed`, `invalid`. The first six mirror `.status.yaml` progress stages. `done` means hydrate complete and PR created. `failed` means the pipeline failed for this change. `invalid` means prerequisites not met.

#### Scenario: Valid manifest with dependencies

- **GIVEN** a manifest with three changes: A (no deps), B (depends on A), C (depends on A)
- **WHEN** the orchestrator reads the manifest
- **THEN** it parses a DAG with A as root and B, C as dependents
- **AND** A is the only immediately dispatchable change

#### Scenario: Manifest with missing required fields

- **GIVEN** a manifest where a change entry lacks `id`
- **WHEN** the orchestrator reads the manifest
- **THEN** it exits with an error naming the malformed entry and the missing field

#### Scenario: Circular dependency detection

- **GIVEN** a manifest where A depends on B and B depends on A
- **WHEN** the orchestrator reads the manifest
- **THEN** it exits with an error: "Circular dependency detected: A <-> B"

### Requirement: Live Editing Contract

The orchestrator SHALL re-read the manifest file from disk on every loop iteration. The human MAY add new change entries to the `changes` list while the orchestrator is running. The orchestrator MUST NOT overwrite human-added entries — it SHALL only update the `stage` field of entries it is processing.

#### Scenario: Human adds entry during execution

- **GIVEN** an orchestrator running with changes A and B in the manifest
- **WHEN** the human appends change C (depends on A) to the manifest between loop iterations
- **THEN** on the next iteration, the orchestrator discovers C and dispatches it when A reaches `done`
- **AND** the human's entry (id, depends_on) is preserved exactly as written

#### Scenario: Concurrent write safety

- **GIVEN** the orchestrator is about to update A's stage to `done`
- **WHEN** the human simultaneously adds a new entry to the manifest
- **THEN** both writes succeed because the orchestrator uses `yq` for targeted field updates (not full-file rewrites)
<!-- clarified: append-only convention confirmed — human adds entries, orchestrator updates stages, yq field-level updates -->

## Pipeline: Orchestrator Core

### Requirement: Main Dispatch Loop

`run.sh` SHALL accept a manifest path as its sole argument. It SHALL run indefinitely until killed by the user (Ctrl+C / SIGINT). The main loop:

1. Re-reads the manifest from disk
2. Validates the DAG (no cycles, all `depends_on` references exist)
3. Identifies dispatchable changes: entries where all `depends_on` IDs have `stage: done` AND the entry itself either has no `stage` field (not started) or has an intermediate `stage` value (resume after interruption)
4. If a dispatchable change exists: dispatch it (serial — one at a time), monitor until completion, write terminal stage (`done` or `failed`) back to the manifest
5. If no dispatchable change exists: sleep for an interval (default 10 seconds), then re-read the manifest — the human may have added new entries
6. Loop back to step 1

The orchestrator SHALL NOT exit when all current changes are done or blocked. It SHALL keep polling the manifest for new entries. This supports the live-contract model: the human is always ahead of the machine, adding entries at their own pace. The only exit path is user-initiated termination (Ctrl+C).

On SIGINT, the orchestrator SHALL print a summary of current state before exiting.

**Resumability**: When determining which changes to dispatch, the orchestrator SHALL classify existing `stage` values as:
- **Terminal** (`done`, `failed`, `invalid`) — skip permanently
- **Intermediate** (`intake`, `spec`, `tasks`, `apply`, `review`, `hydrate`) — re-dispatch into a fresh worktree. The previous run's worktree may still exist but is ignored; the new dispatch uses a new worktree with fresh artifacts copied from the source repo.
- **Absent** (no `stage` field) — dispatch normally

This makes the orchestrator idempotent on restart: kill and re-run is always safe. The old intermediate `stage` value is overwritten once re-dispatch begins.
<!-- clarified: intermediate stages are re-dispatched on restart for idempotent resume -->

#### Scenario: Happy path — linear chain

- **GIVEN** a manifest with A (no deps) → B (depends on A) → C (depends on B)
- **WHEN** `run.sh` is invoked
- **THEN** A is dispatched first, monitored to `done`, then B, then C
- **AND** the manifest shows all three with `stage: done`
- **AND** the orchestrator continues polling for new entries until killed

#### Scenario: Failed dependency blocks downstream

- **GIVEN** a manifest with A → B → C, where A's `fab-ff` fails
- **WHEN** A is marked `stage: failed` in the manifest
- **THEN** B and C are never dispatched (blocked by A)
- **AND** the orchestrator continues polling — the human may add independent changes or fix and retry

#### Scenario: Resume after interruption

- **GIVEN** the orchestrator was killed while A was at `stage: apply` and B has no stage
- **WHEN** the orchestrator is restarted
- **THEN** A is re-dispatched into a fresh worktree (intermediate stage triggers re-dispatch)
- **AND** after A completes, B is dispatched normally
- **AND** A's old worktree from the previous run is left in place (manual cleanup)

#### Scenario: Idle polling picks up new work

- **GIVEN** all current changes are `done` and no dispatchable changes remain
- **WHEN** the orchestrator sleeps and re-reads the manifest
- **AND** the human has added change D (no deps) to the manifest
- **THEN** D is discovered and dispatched on the next iteration

#### Scenario: Fan-out from single root

- **GIVEN** a manifest with A → B, A → C (both depend on A only)
- **WHEN** A completes
- **THEN** B is dispatched next (serial — first in list order)
- **AND** after B completes, C is dispatched

#### Scenario: Multi-parent dependency rejected

- **GIVEN** a manifest where D has `depends_on: [B, C]`
- **WHEN** the orchestrator validates the manifest
- **THEN** it rejects D with: "Multi-parent dependency not supported in v1"
- **AND** other changes (without multi-parent deps) proceed normally

### Requirement: Output

`run.sh` SHALL pass through all output from `claude -p` and shell commands to stdout. The user sees full Claude output in real time, including fab-ff's artifact generation, apply progress, and review results. Between dispatches, the orchestrator SHALL print status lines:

- On dispatch: `[pipeline] Dispatching: <change-id> (worktree: <path>)`
- On completion: `[pipeline] Completed: <change-id> — done`
- On failure: `[pipeline] Failed: <change-id> — <reason>`
- On idle: `[pipeline] Waiting for new entries... (N completed, M pending)`

The `[pipeline]` prefix distinguishes orchestrator messages from Claude's output.

#### Scenario: User watches execution

- **GIVEN** the orchestrator dispatches change A
- **WHEN** Claude generates tasks and applies code
- **THEN** the full Claude output streams to the terminal
- **AND** `[pipeline]` status lines appear before and after

### Requirement: Topological Dispatch Order

The orchestrator SHALL dispatch changes in topological order. When multiple changes are dispatchable simultaneously (serial v1), it SHALL pick the first one in manifest list order. This makes execution order deterministic and predictable.

#### Scenario: Multiple roots

- **GIVEN** a manifest with A (no deps) and B (no deps)
- **WHEN** the orchestrator starts
- **THEN** A is dispatched first (appears first in the `changes` list)
- **AND** B is dispatched after A completes

### Requirement: SIGINT Summary

On SIGINT (Ctrl+C), `run.sh` SHALL trap the signal, print a structured summary of current state, and exit cleanly:

```
Pipeline stopped: <manifest-name>
  Completed: N (list of IDs)
  Failed:    N (list of IDs)
  Blocked:   N (list of IDs)
  Skipped:   N (list of IDs with stage: invalid)
  Pending:   N (list of IDs with no stage)
```

If a change is currently being dispatched when SIGINT arrives, the orchestrator SHALL note it as "in progress" in the summary. The dispatched subprocess (Claude CLI) receives the signal independently via its own signal handling.

#### Scenario: Clean shutdown with summary

- **GIVEN** 2 of 3 changes are `done`, 1 is pending
- **WHEN** the user presses Ctrl+C
- **THEN** the summary shows: "Completed: 2 (A, B)", Pending: 1 (C)
- **AND** the process exits with code 130 (standard SIGINT convention)

## Pipeline: Change Dispatch

### Requirement: Worktree Creation

`dispatch.sh` SHALL create an isolated worktree for each change, with the worktree already on the change's own branch. The branch name is `{branch_prefix}{change-id}` (derived from `config.yaml`'s `git.branch_prefix`).

**For root nodes** (empty `depends_on`):

```bash
wt-create --non-interactive --worktree-open skip <change-branch>
```

This creates a new branch from HEAD (which should be `base`). The worktree path is captured from `wt-create`'s last line of stdout.

**For dependent nodes**:

```bash
git branch <change-branch> origin/<parent-branch>
wt-create --non-interactive --worktree-open skip <change-branch>
```

The change's branch is first created from the parent's branch (pushed by the parent's dispatch), then wt-create checks it out in a new worktree.

The parent's branch name is derived as `{branch_prefix}{parent-id}`.

#### Scenario: Root node worktree

- **GIVEN** change A (`260221-a7k2-user-model`) with `depends_on: []` and manifest `base: main`
- **WHEN** dispatch.sh creates A's worktree
- **THEN** `wt-create --non-interactive --worktree-open skip 260221-a7k2-user-model` is invoked
- **AND** the worktree is on branch `260221-a7k2-user-model` based on main

#### Scenario: Dependent node worktree

- **GIVEN** change B (`260221-b3m1-auth-endpoints`) with `depends_on: [260221-a7k2-user-model]`
- **WHEN** dispatch.sh creates B's worktree
- **THEN** `git branch 260221-b3m1-auth-endpoints origin/260221-a7k2-user-model` is run first
- **AND** `wt-create --non-interactive --worktree-open skip 260221-b3m1-auth-endpoints` is invoked
- **AND** the worktree contains A's committed code (including A's hydrated memory)

### Requirement: Worktree Lifecycle

Worktrees created by `dispatch.sh` SHALL be left in place after dispatch completes, regardless of success or failure. The user is responsible for cleanup via `wt-delete`. This is consistent with the "merging is manual" philosophy — the worktree may be needed for inspection, debugging, or manual merge conflict resolution.

The SIGINT summary SHOULD list worktree paths for any in-progress or completed changes to aid cleanup.

#### Scenario: Successful dispatch leaves worktree

- **GIVEN** change A completes successfully and is marked `done`
- **WHEN** the orchestrator moves on to the next change
- **THEN** A's worktree remains on disk at `<worktrees-dir>/<name>`
- **AND** the user can inspect it, run `wt-delete`, or merge from it

#### Scenario: Failed dispatch leaves worktree

- **GIVEN** change A fails during fab-ff
- **WHEN** dispatch.sh marks A as `failed`
- **THEN** A's worktree remains on disk for debugging
- **AND** the user can enter the worktree to investigate or retry manually

### Requirement: Artifact Provisioning

After worktree creation, `dispatch.sh` SHALL copy the change's artifacts from the source repo into the worktree if they do not already exist there:

```bash
cp -r fab/changes/<id>/ <worktree>/fab/changes/<id>/
```

This ensures the change folder (intake.md, spec.md, .status.yaml) is available in the worktree regardless of which branch it was created on.

#### Scenario: Artifacts not on parent branch

- **GIVEN** change B's artifacts exist on main but B's worktree branches from A's branch
- **WHEN** dispatch.sh checks for `<worktree>/fab/changes/B/`
- **THEN** the directory does not exist (A's branch doesn't have B's artifacts)
- **AND** dispatch.sh copies `fab/changes/B/` from the source repo into the worktree

#### Scenario: Artifacts already present

- **GIVEN** change A's artifacts were committed to main, and A's worktree branches from main
- **WHEN** dispatch.sh checks for `<worktree>/fab/changes/A/`
- **THEN** the directory exists
- **AND** dispatch.sh skips the copy

### Requirement: Prerequisite Validation

Before invoking `fab-ff`, `dispatch.sh` SHALL validate that the change meets prerequisites:

1. `fab/changes/<id>/intake.md` exists in the worktree
2. `fab/changes/<id>/spec.md` exists in the worktree
3. `fab/changes/<id>/.status.yaml` exists and `confidence.score` meets the `fab-ff` gate threshold (checked via `fab/.kit/scripts/lib/calc-score.sh --check-gate <change-dir>`)

If any prerequisite fails, `dispatch.sh` SHALL write `stage: invalid` to the manifest with a reason comment and exit without running `fab-ff`.

#### Scenario: Missing spec

- **GIVEN** change A has intake.md but no spec.md
- **WHEN** dispatch.sh validates prerequisites
- **THEN** it writes `stage: invalid` for A in the manifest
- **AND** logs: "A: prerequisite failed — spec.md not found"

#### Scenario: Confidence below gate

- **GIVEN** change A has spec.md but confidence score is 1.5 (below feature threshold of 3.0)
- **WHEN** dispatch.sh validates prerequisites
- **THEN** it writes `stage: invalid` for A in the manifest
- **AND** logs: "A: prerequisite failed — confidence 1.5 below gate 3.0"

### Requirement: Pipeline Execution via Claude CLI

`dispatch.sh` SHALL execute the fab pipeline in the worktree using the Claude CLI in print mode:

```bash
cd <worktree>
claude -p --dangerously-skip-permissions "/fab-switch <id> --no-branch-change"
claude -p --dangerously-skip-permissions "/fab-ff"
```

The `--no-branch-change` flag is used because the worktree is already on the correct branch (created by wt-create). fab-switch only needs to set the `fab/current` pointer.
<!-- clarified: --dangerously-skip-permissions confirmed — user explicitly opts in by running the orchestrator -->

Each `claude -p` invocation is a separate process — it runs the prompt, produces output, and exits. The `--dangerously-skip-permissions` flag is required because `fab-ff` performs file writes and shell commands that would otherwise prompt for permission.

If `claude -p "/fab-ff"` exits with a non-zero code, `dispatch.sh` SHALL mark the change as `failed`.

**Infrastructure failure**: If `dispatch.sh` encounters an infrastructure failure (wt-create fails, `claude` binary not found, `git push` rejected, or any non-fab-ff error), it SHALL abort the orchestrator entirely with the error message. Infrastructure failures indicate a broken environment — continuing would likely fail on subsequent changes too. The orchestrator prints the SIGINT-style summary before exiting.
<!-- clarified: infrastructure failures abort the orchestrator — environment is likely broken -->

#### Scenario: Successful pipeline run

- **GIVEN** change A meets all prerequisites
- **WHEN** dispatch.sh runs `claude -p "/fab-ff"` in A's worktree
- **THEN** fab-ff generates tasks, applies, reviews, and hydrates
- **AND** dispatch.sh detects success (exit code 0 and `hydrate: done` in .status.yaml)

#### Scenario: fab-ff fails

- **GIVEN** change A's fab-ff exits non-zero (review failed after 3 cycles)
- **WHEN** dispatch.sh checks the exit code
- **THEN** it writes `stage: failed` to the manifest for A
- **AND** logs the failure reason
- **AND** the orchestrator continues to the next dispatchable change

#### Scenario: Infrastructure failure aborts

- **GIVEN** `wt-create` fails with "no space left on device"
- **WHEN** dispatch.sh catches the error
- **THEN** the orchestrator prints the summary and exits with a non-zero code
- **AND** the manifest retains the last-written stage values for all changes

### Requirement: Post-Pipeline Shipping

After successful `fab-ff` completion (hydrate done), `dispatch.sh` SHALL delegate shipping to Claude:

```bash
claude -p --dangerously-skip-permissions "Commit all changes and create a PR targeting <target-branch>. Include a summary of what this change does based on the spec."
```

The `<target-branch>` SHALL be:
- `base` from the manifest for root nodes
- The parent change's branch for dependent nodes (stacked PRs)

Using Claude for shipping gives contextual commit messages and PR descriptions informed by the spec and tasks, rather than generic "Pipeline: change-id" messages.

#### Scenario: Root node PR targets main

- **GIVEN** change A is a root node and manifest `base: main`
- **WHEN** dispatch.sh invokes Claude for shipping
- **THEN** Claude commits, pushes, and creates a PR targeting `main`
- **AND** the commit message and PR body reflect the change's spec

#### Scenario: Dependent node PR targets parent branch

- **GIVEN** change B depends on A, where A's branch is `260221-a7k2-user-model`
- **WHEN** dispatch.sh invokes Claude for shipping
- **THEN** Claude creates a PR targeting `260221-a7k2-user-model` (stacked PR)

### Requirement: Stage Reporting to Manifest

After dispatch completes (success or failure), `dispatch.sh` SHALL read the worktree's `.status.yaml` once to confirm the terminal state, then write the result to the manifest using `yq`:

```bash
yq -i ".changes[] | select(.id == \"<id>\").stage = \"done\"" <manifest>
```

No intermediate polling occurs during dispatch. In serial v1, `claude -p` runs synchronously — the full Claude output streams to stdout in real time, giving the human live visibility. The manifest is updated only on completion.

#### Scenario: Manifest updated after completion

- **GIVEN** change A completes successfully (`hydrate: done` in .status.yaml)
- **WHEN** dispatch.sh reads the terminal state
- **THEN** it writes `stage: done` to the manifest for A

## Pipeline: Example Scaffold

### Requirement: Commented-Out Example

The kit SHALL include `fab/pipelines/example.yaml` — a fully commented-out, annotated example manifest. This file serves as documentation-as-code.

The example SHALL cover:
- The `base` field and its role for root nodes
- `depends_on` syntax (empty list for roots, list of IDs for dependents)
- How the orchestrator writes `stage` fields (and what values to expect)
- A multi-level dependency example (A → B → D, A → C, diamond DAG)
- Notes on prerequisites (intake + spec + confidence score required per change)
- The live-editing contract (human adds entries while orchestrator runs)

The file MUST be fully commented out so it does not interfere with the orchestrator.

#### Scenario: Developer creates pipeline from example

- **GIVEN** a developer reads `fab/pipelines/example.yaml`
- **WHEN** they copy and uncomment relevant sections
- **THEN** they have a valid manifest ready for the orchestrator

## Design Decisions

1. **Serial execution in v1**: Process one change at a time in topological order.
   - *Why*: Simplicity — avoids concurrent worktree/process management, race conditions on manifest writes, and interleaved output. Parallel is a documented stretch goal.
   - *Rejected*: Parallel dispatch with background processes — adds PID tracking, concurrent manifest writes, output interleaving. Valuable but not v1.

2. **`claude -p` (print mode) for CLI invocation**: Each pipeline step is a separate `claude -p` call that runs a prompt and exits.
   - *Why*: Print mode is non-interactive and exits after completion — ideal for automated pipelines. Each step is isolated (no session state bleeds between steps). Exit codes indicate success/failure.
   - *Rejected*: `claude --print` (deprecated alias), `claude -c` (continue mode — requires existing session), piping commands (fragile, no clean exit handling).

3. **Claude for shipping**: Post-pipeline shipping (commit, push, PR) uses `claude -p`, same as pipeline execution.
   - *Why*: Claude generates contextual commit messages and PR descriptions informed by the spec/tasks, rather than generic "Pipeline: change-id" messages. Consistency — the entire dispatch uses the same execution model.
   - *Rejected*: Direct shell (`git add && git commit && gh pr create`) — deterministic but produces generic, context-free commit messages and PR bodies.

4. **Artifact provisioning via copy**: dispatch.sh copies change artifacts from the source repo to the worktree if not already present.
   - *Why*: Worktrees branch from parent changes, which may not have the dependent change's artifacts committed. Copy ensures the change folder is always available regardless of git branch topology.
   - *Rejected*: Requiring all artifacts committed to all branches (impractical for multi-change pipelines). Symlinks (fragile across worktrees).

5. **Stacked PRs targeting parent branch**: Dependent changes create PRs targeting their parent's branch, not main.
   - *Why*: Preserves the dependency chain in the git history. Each PR shows only the diff introduced by that specific change. Merging follows the same topological order.
   - *Rejected*: All PRs target main (would show cumulative diffs, making review harder).

6. **Append-only manifest convention**: Concurrent access safety relies on convention (human adds entries, orchestrator updates stages on separate lines) rather than file locking.
   - *Why*: Simpler than flock. The human only appends new entries; the orchestrator only updates the `stage` field of existing entries. These operations target different parts of the file. `yq` field-level updates minimize the write footprint.
   - *Rejected*: `flock`-based locking (adds complexity, portability concerns, overkill for the access pattern).

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Serial execution first, parallel as stretch | User explicitly stated. Confirmed from intake #1 | S:85 R:90 A:80 D:75 |
| 2 | Certain | YAML manifest with live human/orchestrator contract | Discussed extensively. Confirmed from intake #2 | S:90 R:85 A:85 D:85 |
| 3 | Certain | Worktrees branched from parent change's branch | User explicitly stated. Confirmed from intake #3 | S:90 R:70 A:80 D:85 |
| 4 | Certain | Use fab-ff, not fab-fff | User explicitly stated — confidence gating required. Confirmed from intake #4 | S:95 R:85 A:90 D:90 |
| 5 | Certain | Confidence score required — spec stage is prerequisite | User explicitly stated. Confirmed from intake #5 | S:90 R:80 A:85 D:85 |
| 6 | Certain | Merging to main is manual | User explicitly stated. Confirmed from intake #6 | S:95 R:80 A:90 D:90 |
| 7 | Certain | Scripts live in fab/.kit/scripts/pipeline/ | Fits constitution's pure-prompt-play principle. Confirmed from intake #7 | S:75 R:90 A:85 D:80 |
| 8 | Certain | Exit code + terminal .status.yaml read (no monitor.sh) | Discussed — monitor.sh dropped for v1 serial. Exit code is primary, .status.yaml read confirms terminal state | S:85 R:80 A:85 D:85 |
| 9 | Certain | Manifests stored in fab/pipelines/ | Implicitly confirmed — user engaged with spec across two clarify sessions without objection | S:80 R:90 A:85 D:80 |
| 10 | Certain | Append-only convention for concurrent manifest access (no flock) | Clarified — user explicitly confirmed in clarify session | S:85 R:70 A:85 D:85 |
| 11 | Certain | Byobu integration as stretch goal | User acknowledged complexity. Confirmed from intake #11 | S:80 R:90 A:80 D:85 |
| 12 | Certain | Commented-out example.yaml as documentation-as-code | User explicitly requested. Confirmed from intake #12 | S:95 R:95 A:90 D:90 |
| 13 | Certain | `--dangerously-skip-permissions` for unattended Claude CLI execution | Clarified — user explicitly confirmed in clarify session | S:85 R:75 A:85 D:85 |
| 14 | Certain | Claude for shipping (commit/push/PR via claude -p) | Discussed — user chose Claude for contextual commit messages and PR descriptions over generic shell commands | S:85 R:85 A:85 D:85 |
| 15 | Certain | Partial dependency out of scope for v1 | Logical consequence of serial execution (#1) + single-dep restriction (#17) | S:85 R:85 A:90 D:90 |
| 16 | Certain | Artifact provisioning via copy to worktree | Mechanical necessity of confirmed branching strategy (#3) — parent branches don't have dependent change artifacts | S:85 R:80 A:90 D:90 |

| 17 | Certain | Single-dependency restriction in v1 | Clarified — user confirmed. Avoids multi-parent branch topology problem | S:85 R:85 A:90 D:90 |
| 18 | Certain | Intermediate stages re-dispatched on restart | Clarified — user chose re-dispatch for idempotent resume | S:80 R:80 A:85 D:85 |
| 19 | Certain | Infrastructure failures abort the orchestrator | Clarified — user chose abort. Broken environment means continuing is futile | S:85 R:75 A:85 D:85 |

19 assumptions (19 certain, 0 confident, 0 tentative, 0 unresolved).

## Clarifications

### Session 2026-02-21

1. **Branch creation during dispatch** — Q: How should dispatch.sh create the change's branch? A: Pass the change-id as the branch argument to wt-create. For dependent nodes, create the branch from the parent's branch first via `git branch`. fab-switch uses `--no-branch-change` since the worktree is already on the correct branch.
2. **Worktree lifecycle** — Q: What happens to worktrees after dispatch? A: Left in place. User cleans up manually via wt-delete. Consistent with "merging is manual" philosophy.
3. **Orchestrator output** — Q: What does run.sh print to stdout? A: Full Claude output passthrough, with `[pipeline]` prefixed status lines between dispatches.
4. **Permission model** — Q: How to handle Claude CLI permissions? A: `--dangerously-skip-permissions` — explicit opt-in by running the orchestrator. Upgraded from Tentative to Confident (#13).
5. **Concurrent write safety** — Q: Is append-only convention acceptable? A: Yes — human adds entries, orchestrator updates stages, yq field-level updates. Upgraded from Tentative to Confident (#10).

### Session 2026-02-21 (2)

6. **Multi-dependency parent resolution** — Q: How to handle changes with multiple dependencies? A: Restrict `depends_on` to at most one entry in v1. Multi-parent requires human to merge parent branches first. Added assumption #17.
7. **Orchestrator resumability** — Q: How to handle restart after interruption? A: Re-dispatch changes with intermediate stage values into fresh worktrees. Terminal stages (done/failed/invalid) are skipped. Added assumption #18.
8. **Infrastructure failure handling** — Q: How to handle wt-create/claude/git failures? A: Abort the orchestrator entirely — broken environment means continuing is futile. Added assumption #19.
