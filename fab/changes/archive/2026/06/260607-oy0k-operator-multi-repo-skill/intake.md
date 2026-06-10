# Intake: Operator Multi-Repo Skill + Specs

**Change**: 260607-oy0k-operator-multi-repo-skill
**Created**: 2026-06-07
**Status**: Draft

## Origin

> feat: operator multi-repo skill + specs — Update the fab-operator skill (`src/kit/skills/fab-operator.md`) and its specs for multi-repo/multi-session coordination on one tmux server.

This is **change 2 of a 2-change split** that makes `/fab-operator` coordinate fab agents across multiple repos and multiple tmux sessions on a single tmux server. The split was decided in a `/fab-discuss` session (2026-06-07):

- **Change 1** (`feat: operator multi-repo Go primitives`) — the binary-side mechanism: server-keyed XDG state file, per-repo `mainRoot` in `fab pane map`, and a `fab spawn-command --repo` helper. Created separately via `/fab-new`.
- **Change 2** (this change) — the skill-side policy: re-frame `fab-operator.md` and its specs around the multi-repo model that change 1's primitives enable.

**This change depends on change 1.** It consumes primitives (server-keyed state path, `repo` field in `fab pane map --json`, `fab spawn-command --repo`) that change 1 introduces. Since both changes live in the same repo (fab-kit), the operator's autopilot would cherry-pick change 1 into change 2's worktree before spawning (same-repo dependency semantics).

**Design decisions** (locked in the discussion, recorded in memory `operator-multi-repo-design.md`):

1. **Isolation unit = tmux server.** One operator per tmux server, matching the existing `tmux select-window -t operator` server-wide singleton. No `--name` dimension. A second operator means a second tmux server (`tmux -L <label>`). "Multiple sessions, same server" share one operator and one state file.
2. **State file keyed by tmux socket path** (not by repo, not by a fixed global path). A fixed path would force a machine-wide singleton, which was explicitly rejected. Location: `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`. (Change 1 implements the path derivation; this change documents it in the skill.)
3. **No migration** of old repo-rooted `.fab-operator.yaml` files — they are abandoned in place. Fresh start on the new path, acceptable because operators don't survive a binary upgrade anyway (the monitored set is re-derivable from live `»`-prefixed panes).
4. **Cross-repo dependencies = ordering-only.** Same-repo `depends_on` cherry-picks as today; cross-repo `depends_on` is a pure ordering barrier (wait for the dependency to reach its `stop_stage`, never merge code).

## Why

**Problem.** Today `/fab-operator` is implicitly single-repo, single-session. The skill's prose addresses panes as if they all live in one repo and one session: the tick runs `fab pane map` without `--all-sessions` (so it only sees the operator's own session), spawning reads the operator's own `config.yaml` rather than the target repo's, and `depends_on` cherry-picks assume one branch namespace. A user who works across several repos in several tmux sessions on one server cannot have a single operator coordinate all of them.

**Consequence of not fixing.** The user must run a separate operator per repo (one tmux server each, or no operator at all for some repos), losing the central value of the operator: a single pane of glass that monitors, auto-answers, and drives agents wherever they run. Cross-repo autopilot queues and cross-repo watches are impossible.

**Why this approach.** The Go primitives are already ~80% pane-centric — each pane resolves via its own CWD → its own git worktree → its own `fab/`. The observation layer barely needs to change (change 1 handles the small gaps). The bulk of the multi-repo work is **re-framing the skill's mental model** from "one repo, one session" to a `(session, repo, pane)` addressing tuple. Splitting Go (change 1) from skill+specs (this change) separates *mechanism* from *policy*: change 1 ships and is independently testable; this change layers the coordination behavior on top of the primitives it provides.

The Constitution requires that skill-file changes update the corresponding `docs/specs/skills/SPEC-*.md`, so this change is inherently skill + spec, not skill alone.

## What Changes

All changes are in `src/kit/skills/fab-operator.md` plus its spec and the external-CLI helper. **No Go code** changes here (that is change 1). The skill consumes change 1's primitives.

### B1 — Re-frame addressing as `(session, repo, pane)`

- **§1 Principles**: add a "Multi-repo aware" principle. The operator spans repos and sessions on one tmux server; pane ID remains the primary key (server-global, stable), with repo and session as dimensions layered on top. Every monitored entry, `branch_map` entry, and watch is repo-qualified.
- **§4 `.fab-operator.yaml` schema** — extend with repo/session dimensions:

```yaml
monitored:
  r3m7:
    pane: "%3"
    repo: /home/user/code/foo        # NEW — absolute main-worktree root
    session: work                     # NEW — tmux session name
    stage: apply
    agent: active
    stop_stage: null
    spawned_by: null
    depends_on: []
    branch: 260324-r3m7-add-retry-logic
    enrolled_at: "..."
    last_transition: "..."
branch_map:                           # CHANGED — value is now {branch, repo}
  ab12: { branch: 260324-ab12-fix-auth, repo: /home/user/code/foo }
  cd34: { branch: 260324-cd34-add-oauth, repo: /home/user/code/bar }
watches:
  linear-bugs:
    enabled: true
    source: linear
    query: { project: "DEV", status: [Backlog, Todo], assignee: "@me" }
    target_repo: /home/user/code/foo  # NEW — which repo spawned changes land in
    stop_stage: intake
    known: [...]
    completed: [...]
    ...
```

### B2 — Tick uses `--all-sessions` (§4 Tick Behavior)

- Step 1 snapshot: replace `fab pane map` with `fab pane map --all-sessions --json`. Group rows by `repo` (using change 1's new `repo` JSON field), then by `session`.
- Status frame: repo-section the previously-flat list. Each repo gets a header (or a `repo`/`session` column added to every row). The health glyphs, autopilot `▶`, and `⚠` markers are unchanged — only the grouping changes.

Example reshaped frame:

```
── Operator ── 17:32 ── tick #47 ── 7 tracked ──

  ~/code/foo (session: work)
    [change]  r3m7   ▶ ● apply → review
    [change]  ab12     ✓ hydrate
  ~/code/bar (session: side)
    [change]  k8ds   ▶ ◌ review · idle 8m
  [watch]   linear-bugs  → ~/code/foo   ● 2 known · 1 completed · 3m ago
```

### B3 — Repo-targeted spawning (§6 Spawning an Agent / Working a Change)

Every spawn flow must first establish **which repo** the work targets, then:

1. Run `wt create --non-interactive` **in that repo's directory** (so the worktree lands under `$(dirname target-repo)/<repo>.worktrees/`, not the operator's repo).
2. Read **that repo's** `agent.spawn_command` via the new `fab spawn-command --repo <target-repo>` helper (change 1), rather than the operator's own `config.yaml`.
3. Open the agent tab and enroll with `repo` and `session` recorded.

Window markers (`»` / `›`) are unchanged — they key on server-global pane IDs.

### B4 — Two-tier dependencies (§6 Dependency Resolution)

Split `depends_on` resolution by repo:

```
for each dep in depends_on:
    if dep.repo == change.repo:
        # same-repo: cherry-pick as today
        git cherry-pick --no-commit origin/main..<dep-branch> && git commit ...
    else:
        # cross-repo: ORDERING-ONLY barrier
        wait until dep reaches its stop_stage, then spawn. No code merge.
```

**Documented caveat (REQUIRED):** an ordering-only cross-repo dependency gives the dependent agent **no code** from its dependency — it is a pure sequencing constraint. This is correct only for *logical* dependencies ("don't start the frontend change until the API change merges"), never for code-level dependencies. The skill must state this explicitly so users do not expect cross-repo `depends_on` to make the dependency's code available.

The ancestor-pruning logic (`git merge-base --is-ancestor`) applies only within the same-repo subset of the dependency set.

### B5 — Repo-scoped autopilot (§6 Autopilot, Ordered Merge)

- A queue **may now span repos**, with mixed dependency semantics: implicit `--base` chaining cherry-picks within a repo and degrades to ordering-only across repo boundaries.
- Ordered merge (`gh pr merge` + CI wait) tracks **per-repo PR sequences** — the merge order respects each repo's own dependency chain. The queue-completion summary annotates each PR with its repo.
- CI-failure halt behavior is **halt-dependents-only**: a CI failure in one repo halts that repo's merge sub-sequence AND any other repo whose queue items carry a cross-repo `depends_on` pointing into the failed chain; truly independent repos' sub-sequences continue merging. The queue-completion summary reports which sub-sequences halted vs. completed, and escalates the failure to the user. <!-- clarified 2026-06-08: user chose halt-dependents-only over the conservative halt-all default, to maximize independent-repo throughput while still respecting cross-repo ordering barriers -->
  - **Implementation note for apply**: "dependent" is determined transitively over the cross-repo `depends_on` graph — a repo halts if any of its queued items depends (directly or via another halted item) on a PR in the failed chain. Repos with no such edge continue.

### B6 — Watches gain `target_repo` (§7)

- Add `target_repo` to the watch schema (above). A Linear/Slack watch that spawns agents must declare which repo the spawned change lands in.
- Conversational management gains repo targeting: e.g., *"watch Linear project DEV, spawn into ~/code/foo, stop at intake"* sets `target_repo`. *"What are you watching?"* lists each watch's `target_repo`.

### B7 — Configuration & Key Properties (§8, §9)

- **§8 Configuration**: document the one-operator-per-server model explicitly.
- **§9 Key Properties**: update the `.fab-operator.yaml` row to note the server-keyed XDG location (`$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`) instead of repo-rooted.

### Spec & helper updates (Constitution-mandated)

- **`docs/specs/skills/SPEC-fab-operator.md`** — update the per-skill spec for the multi-repo model (flow diagrams, schema, addressing tuple). Required by the Constitution: skill-file changes MUST update the corresponding SPEC file.
- **`src/kit/skills/_cli-external.md`** — note `fab pane map --all-sessions` usage in the tick, and the repo-targeted `wt create` flow.

## Affected Memory

- `runtime/operator`: (modify) `/fab-operator` behavior — multi-repo/multi-session coordination, `(session, repo, pane)` addressing, repo-qualified monitored set / branch_map / watches, two-tier dependency resolution. <!-- updated 2026-06-08: was fab-workflow/execution-skills; the fab-workflow pseudo-domain was restructured into pipeline/memory-docs/distribution/runtime in PR #381, and the operator's behavioral contract now lives in docs/memory/runtime/operator.md -->

(Implementation-only details of change 1's Go primitives are hydrated by change 1, not here. This change's memory impact is the operator's *behavioral* contract.)

## Impact

- **Skill**: `src/kit/skills/fab-operator.md` (primary — §1, §4, §6, §7, §8, §9), `src/kit/skills/_cli-external.md` (tick + spawn flow notes).
- **Specs**: `docs/specs/skills/SPEC-fab-operator.md`.
- **Consumes from change 1**: server-keyed state path, `repo` field in `fab pane map --json`, `fab spawn-command --repo` helper. These must exist before this change's behavior is correct — hence the dependency.
- **No Go code, no templates, no migrations** in this change.
- **Deployment**: skill edits to `src/kit/skills/` take effect after `fab sync` (the `.claude/skills/` copies are gitignored and regenerated). No user-data restructuring.

## Open Questions

_All previously-open questions resolved during /fab-clarify (2026-06-08):_

- **B5 CI-failure scope across repos** — RESOLVED: **halt-dependents-only**. A CI failure halts the failing repo's sub-sequence plus any repo with a cross-repo `depends_on` into the failed chain; truly independent repos continue. (Superseded the earlier "halt all" lean.)
- **B2 frame layout** — RESOLVED: **repo-section headers** (per-repo header line with indented entries), chosen for scannability over per-row repo/session columns.

## Clarifications

### Session 2026-06-08

| # | Q | A |
|---|---|---|
| 9 (B2) | Status frame layout: repo-section headers vs. per-row repo/session columns? | Repo-section headers (per-repo header line with indented entries, for scannability) |
| 10 (B5) | Cross-repo CI-failure during ordered merge: halt all, halt only failing repo, or halt dependents only? | Halt-dependents-only — halt the failing repo + any repo with a cross-repo `depends_on` into the failed chain; independent repos continue |

### Session 2026-06-08 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 6 | Confirmed | — |
| 7 | Confirmed | — |
| 8 | Confirmed | — |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Isolation unit = tmux server; one operator per server, no `--name` dimension | Discussed — user explicitly chose one-operator-per-server; matches existing `tmux select-window -t operator` singleton | S:98 R:70 A:90 D:95 |
| 2 | Certain | State file keyed by tmux socket path under `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`; skill documents this | Discussed — user confirmed; rejected fixed global path (would force machine-wide singleton). Path derivation is change 1's job | S:95 R:65 A:88 D:92 |
| 3 | Certain | No migration of old repo-rooted `.fab-operator.yaml`; abandon in place | Discussed — user explicitly said no migration needed | S:98 R:80 A:90 D:98 |
| 4 | Certain | Cross-repo `depends_on` = ordering-only (no code merge); same-repo = cherry-pick as today | Discussed — user selected "Allow, no cherry-pick" over forbid/full-cross-repo | S:95 R:55 A:85 D:90 |
| 5 | Certain | Ship as skill+specs change separate from Go primitives; this change depends on change 1 | Discussed — user selected "Split Go / skill" over one-change / stack-of-4 | S:98 R:75 A:92 D:95 |
| 6 | Certain | `branch_map` value becomes `{branch, repo}` (was bare branch string) | Clarified — user confirmed | S:95 R:50 A:80 D:75 |
| 7 | Certain | Pane ID stays primary key; `repo`/`session` are added dimensions, not replacements | Clarified — user confirmed | S:95 R:60 A:85 D:80 |
| 8 | Certain | Cross-repo dependent agent gets NO code from its dep — documented as logical-only caveat | Clarified — user confirmed | S:95 R:65 A:82 D:78 |
| 9 | Certain | Status frame uses repo-section headers (vs. per-row repo/session columns) | Clarified — user confirmed repo-section headers for scannability | S:95 R:80 A:55 D:50 |
| 10 | Certain | Cross-repo CI-failure during ordered merge halts the failing repo + any repo whose queue items have a cross-repo `depends_on` into the failed chain; truly independent repos continue | Clarified — user changed from "halt all" to "halt dependents only" (most precise; independent repos keep merging) | S:95 R:60 A:55 D:48 |

10 assumptions (10 certain, 0 confident, 0 tentative, 0 unresolved).
