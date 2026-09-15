# Intake: Operator Spawn Target Session — Delegate to `rk tab new`'s Role-Aware Default

**Change**: 260913-1nxw-operator-spawn-session-rk-default
**Created**: 2026-09-13

## Origin

Promptless dispatch (`/fab-proceed` create-new path, `{questioning-mode} = promptless-defer`) from a synthesized description handed down by the team lead on 2026-09-13. This is the **planned follow-through** of change `260913-iyb4-operator-spawn-session-selector` (merged as PR sahil87/fab-kit#675, released in fab-kit v2.26.4). The sequencing was a user decision in the iyb4 discussion: **skill fix first, then this** — once run-kit's `rk tab new` resolves a default session on its own, fab drops its interim selector behind a capable-HexoKit gate.

> Operator spawn target session — delegate to `rk tab new`'s role-aware default; drop the fab-side selector. run-kit backlog `[xga4]` → run-kit PR #966 "rk tab new — Role-Aware Default Session Resolution" is released in run-kit **v3.19.59** (verified installed: `rk --version` = `run-kit version v3.19.59`). run-kit backlog done and released → do the fab side.

Key decisions carried across the boundary (all user-directed; numbered as in the source description and mirrored by § What Changes A–I):

1. §6 step 2 stops choosing — keep the step's slot and number, replace its body with a short delegation.
2. Step 7's raw form makes `--session` optional; the `--json` report now also carries `session`/`session_rung`.
3. Step 8 takes `<session>` from the `--json` report's `session` field only.
4. §2 rk Gate gains the capability half for this feature — probe `session_rung` in `rk tab new --help`, not a version string.
5. §8 Settings row → "none — rk tab new's role-aware default (§6 step 2)".
6. `_cli-agents.md` § Spawn Composition raw form: `--session` optional; bullet rewritten to "rk resolves the landing session".
7. **Net shorter again** — `wc -w src/kit/skills/fab-operator.md` MUST be lower after apply than at origin/main; step 2 MUST shrink to ≤ 60 words.
8. Memory (hydrate): item 1 → delegation text; item 4 → optional `--session`; the "Spawn Target Session Is Selected Deterministically…" Design Decision rewritten **in place** to present truth; two sibling DDs adjusted; `agent-primitives.md` § Spawn composition repointed.
9. Sibling sweep before finishing apply (grep list in § What Changes I).

**What rk now does** (verbatim from `rk tab new --help` on v3.19.59 — quoted, not re-derived):

> Default session resolution runs in rungs: outside tmux the target server's current session; inside tmux the caller's own session — unless the caller sits in a run-kit infrastructure session (`_rk-operator` and friends), which never gains a spawned window beside itself. The pick is then deterministic over the server's user sessions: the sole user session, else the one rooted at `--cwd`'s main worktree (linked worktrees resolve through git's common dir, so `<repo>.worktrees/<name>` matches a session started at `<repo>`), else the most-attached one (ties break to the `rk mux sessions` row order). With no user session at all the command fails "nowhere to spawn" and creates nothing — pass `--session =S` to name one. The deciding rung is reported as the `--json` "session_rung" key and, on the human path, as a stderr note.
>
> `--json` prints `{"session", "session_rung", "window_id", "pane_id"}` inside the standard `{"ok","result"}` envelope instead of the bare @N — `session` is tmux's own report of where the window landed, `session_rung` why it was chosen (`explicit|caller|server|sole-user|cwd-root|most-attached`). `--ready` (requires `--json` and a `--` command) also waits for the new pane's boot readiness … and adds the verdict as the JSON "ready" key.

**Gap Analysis (this intake)**: every skill line the description cites was verified against the worktree at origin/main `3d7bd152` (v2.26.4): `fab-operator.md:77` (gate probe), `:80` (either-half explanation), `:83` (STOP line), `:479` (step 2, **121 words**), `:500`/`:503` (step 7 raw form + parenthetical), `:510–512` (step 8), `:800` (§8 row); `_cli-agents.md:87` (raw form) and `:91` (the pin bullet). No active change or `fab/backlog.md` item covers this follow-through (the iyb4 intake recorded it under "§ G. Follow-up recorded, out of scope"; `fab/backlog.md` carries no matching entry). `docs/specs/operator.md` has no session prose (its only related token is a v10 history row naming `rk cron list --json`, human-curated, untouched). **One skill site the description's known-sites list missed**: `fab-operator.md:820` — the §9 Key Properties "Requires HexoKit?" row restates the gate probe (`command -v rk` + `rk cron list --json`) and must be extended with the new half. **Memory sites the description missed**: `docs/memory/runtime/operator.md:35` (the startup rk-gate paragraph, which spells the probe verbatim) and the Design Decision "rk Is a Hard Dependency of the Operator Skill" (`operator.md:553–557`, whose Decision names both probe halves and whose Rejected row says "the gate floor is the capability the clock needs") — both need the new half at hydrate. Baseline measured for decision 7: `wc -w src/kit/skills/fab-operator.md` = **14594** at origin/main. Precedent for a `--help`-discriminant capability probe already exists in the kit: `_cli-fab-operator.md` § fab operator probes `rk operator --help` for `--workers` ("the binary is probed, never the version string").

## Why

**The problem.** After iyb4 the operator skill (`src/kit/skills/fab-operator.md`) carries a five-rung target-session selector in §6 "Spawning an Agent" step 2, plus its zero-candidate clause, announce rule, and restatements in §8, step 7, and memory. That selector was explicitly an **interim**: the iyb4 Design Decision's Rejected row records "a run-kit default session first … kept as a follow-up: `rk tab new` resolving a default session when the caller sits in an `_rk-*` session, after which fab can drop step 2 and the flag behind the capable-HexoKit gate". run-kit v3.19.59 has now shipped exactly that: `rk tab new` refuses to land a window beside an infrastructure caller, resolves a deterministic default over user sessions (sole → cwd-root → most-attached → row order), fails loudly with "nowhere to spawn" when there is none, and reports both the landing `session` and the deciding `session_rung` in its `--json` result.

**Two copies of one policy.** The fab-side rungs (b) sole candidate, (c) `path` equals the target repo root, and (e) highest `attached` then row order are the same policy rk now applies as `sole-user`, `cwd-root`, and `most-attached`. Keeping both is the drift mechanism the project's owner-or-pointer rule exists to kill (`fab/project/code-quality.md` § Anti-Patterns "Stating an owned rule AND pointing at its owner"): rk owns session roles (`rk mux sessions` derives them from its reserved constants) and the tab primitive, so rk is the right home for "where does a spawned window land". The operator skill should stop carrying session policy at all — this is the whole point of the z597 → cx52 → 4a8m → iyb4 thread, and it is what makes the skill net shorter again.

**Why a capability probe is mandatory.** A pre-3.19.59 rk given `rk tab new` with `--session` omitted from an `_rk-operator` caller lands the window in the operator's own session — the original z597 bug — **silently**. Dropping `--session` without gating on the capability would reintroduce the misplacement for every user whose run-kit lags fab-kit. The §2 rk Gate already exists for exactly this class (it probes `rk cron list --json` for an rk predating `rk cron`), and fab's rk convention is capability probes, not version parsing; `rk tab new --help` naming `session_rung` is a cheap, side-effect-free discriminant for the feature.

**If we do nothing.** The skill keeps 121 words of selector plus its satellites for a decision rk already makes; rk's `session_rung` report goes unread; and the two rung lists drift the first time either side changes (e.g. rk adds a rung, or someone "fixes" fab's list again as 4a8m did).

**Why this approach over the alternatives.** Delegation + capability gate keeps user-visible behaviour identical (the operator still never asks; the landing session is the one iyb4's rungs b/c/e would have picked), deletes prose instead of adding it, and puts the policy where the substrate is. Alternatives rejected are listed at the end of § What Changes.

## What Changes

Canonical sources only: `src/kit/skills/fab-operator.md` and `src/kit/skills/_cli-agents.md` at apply; `docs/memory/runtime/operator.md` and `docs/memory/runtime/agent-primitives.md` at hydrate. Never `.agents/skills/` or `.claude/skills/` (deployed copies). Deployed content cites no fab-kit-only paths (Constitution V; `TestKitContentCitesNoRepoLocalPaths` in `src/go/fab-kit/cmd/fab/kit_portability_test.go` must stay green: `cd src/go/fab-kit && go test ./cmd/fab/`).

### A. §6 "Spawning an Agent" step 2 — stop choosing, delegate (`fab-operator.md:479`)

**Keep the slot and the number** — steps 3–8 and every "step 7" / "step 8" / "steps 1–3" / "steps 4–8" / "eight steps" cross-reference in the skill (`:588`, `:603`, `:678`, `:679`, `:682`) and in memory (`operator.md:107` "8 steps (target repo, target session, …)", `:406`, `:442`) stay valid with no renumbering. Step 2 is still "target session"; it is now delegated.

**Current body (121 words)**:

> 2. **Establish target session** — the tmux session for the new window, pinned at step 7. Candidates: `rk mux sessions --json` rows (the `result` array in run-kit's envelope form) with `role: "user"`, minus the operator's own session. First rung naming exactly one candidate wins: (a) the §8 "Spawn target session" setting, if user-set; (b) the sole candidate; (c) `path` equals the target repo root; (d) most panes whose `repo` (`fab pane map --all-sessions --json`) is the target repo; (e) highest `attached`, then first row. Zero candidates → `nowhere to spawn` error (attended: output; unattended: §5), no spawn. Announce one line (spawn output, never §5): session + rung. Never trust a persisted `scope.session` alone; the ambient session is never an implicit target.

**Replacement (shape; apply owns wording; target ≤ 60 words)**:

> 2. **Target session** — resolved by `rk tab new` at step 7: omit `--session` and rk applies its role-aware default, reporting `session` + `session_rung`. Pass `--session =<name>` only when the user set the §8 "Spawn target session" override. Never trust a persisted `scope.session` alone; the ambient session is never an implicit target.

Rules for the replacement:

- The z597 invariant sentence "Never trust a persisted `scope.session` alone; the ambient session is never an implicit target" is kept verbatim — still true: the operator's own session is never a target, and rk now enforces it (an `_rk-*` caller "never gains a spawned window beside itself").
- **Delete** the a–e rung list, the candidate-set definition (`rk mux sessions --json` rows with `role: "user"`), the `fab pane map --all-sessions --json` pane-count rung, the zero-candidate clause, and the fab-side announce rule.
- The zero-user-session case needs no clause here: rk's own `nowhere to spawn` failure (non-zero exit, nothing created) is surfaced at step 7 exactly like any other `rk tab new` error — "surface it, never silently retry against the ambient session" — with no special text in step 2.
- The announce becomes part of step 7: one line echoing rk's `session` and `session_rung` from the `--json` result (e.g. `→ session fab_kit (cwd-root)`). Apply owns the exact rendering; it stays in the spawn output, never §5.
- The run-kit envelope parenthetical ("the `result` array in run-kit's envelope form") that iyb4 kept in step 2 may move to step 7's report sentence (which already names the envelope) or be dropped — step 7 already says "the `result` object of run-kit's `{"ok":true,"result":{…}}` envelope".

### B. Step 7 — raw form and parenthetical (`fab-operator.md:500–503`)

Raw form becomes:

```sh
rk tab new [--session =<name>] --cwd <worktree> --name <wt> --ready --json -- <spawn-argv…> ["<prompt>"]
```

The parenthetical currently reads: "(`=<session>` pins the target session from step 2 — there is no ambient-session fallback, and an unknown session errors loudly at spawn: surface it, never silently retry against the ambient session. `<wt>` is the worktree name from step 3.)" Rewrite it to say:

- `--session =<name>` is passed **only** for the §8 override; otherwise rk resolves the landing session (step 2).
- An unknown explicit session still errors loudly at spawn; so does rk's "nowhere to spawn" — surface either, never retry against the ambient session.
- The `--json` report (the `result` object of run-kit's envelope) now carries `session` and `session_rung` alongside `window_id`, `pane_id`, and `ready` — step 8 consumes `session`; the one-line announce echoes `session` + `session_rung`.
- `<wt>` is the worktree name from step 3 (unchanged).

The `ready:` handling sentence (`parked`/`narrow` judgment rounds; `gone` one bounded retry then escalate) and the Window marks paragraph are unchanged.

### C. Step 8 — `<session>` comes from the report (`fab-operator.md:510–512`)

Both command lines keep their shape:

```sh
fab operator track add <id> --kind pane --pane <pane-id> --session <session> --repo <repo> [--change <change-id> --branch <branch>] [--stage <stage>] [--stop-stage <stage>] [--spawned-by <item-id>] [--depends-on <id,…>]
fab operator track update <id> --scope '{"pane":"<pane-id>","session":"<session>"}'
```

Add one clause stating that `<session>` is the `--json` report's `session` field — tmux's own report of where the window landed — and no other source (not the §8 setting, not a persisted `scope.session`, not the operator's ambient session). No `fab` command signature changes.

### D. §2 rk Gate — capability half for default-session resolution (`fab-operator.md:72–83`)

**Current**:

```bash
command -v rk >/dev/null 2>&1 && rk cron list --json >/dev/null 2>&1
```

> If either half fails (rk absent, or an installed rk predating `rk cron` — the capability probe), STOP:
> `Error: the operator requires HexoKit — brew install sahil87/tap/run-kit`

**Replacement (recommended probe; apply may pick an equivalent that is cheap and side-effect-free)**:

```bash
command -v rk >/dev/null 2>&1 && rk cron list --json >/dev/null 2>&1 && rk tab new --help 2>&1 | grep -q session_rung
```

> If any part fails (rk absent, or an installed rk predating `rk cron` or `rk tab new`'s default-session resolution — the capability probes), STOP:
> `Error: the operator requires HexoKit — brew install sahil87/tap/run-kit`

- The STOP message text stays; a `brew upgrade` hint MAY be appended if it fits on the same line (e.g. `… — brew install (or upgrade) sahil87/tap/run-kit`). Apply owns it.
- Rationale to carry into the prose (one clause is enough): without this half, a pre-3.19.59 rk would silently reintroduce the z597 misplacement once `--session` is omitted.
- Probe style follows the kit's existing convention — capability, never version: `rk cron list --json` (this gate), `rk operator --help` containing `--workers` (`_cli-fab-operator.md` § fab operator). `session_rung` is the discriminant because it appears only in help text that ships with the feature.
- **Sibling site**: the §9 Key Properties row `| Requires HexoKit? | Yes — hard stop without it (§2 rk Gate: \`command -v rk\` + \`rk cron list --json\`) |` (`fab-operator.md:820`) is extended to name the third half (e.g. `+ \`rk tab new --help\` ∋ session_rung`).

### E. §8 Settings row (`fab-operator.md:800`)

**Current**: `| Spawn target session | none — §6 step 2's selector decides; set only by the user (rung a) | "spawn into session {name}" |`

**Replacement (shape; apply owns wording)**: `| Spawn target session | none — rk tab new's role-aware default (§6 step 2) | "spawn into session {name}" → passes --session =<name> |`

The setting remains a pure user override, session-scoped, reset on compaction (the §8 footnote is unchanged).

### F. `_cli-agents.md` § Spawn Composition — raw form + pin bullet (`_cli-agents.md:87`, `:91`)

This helper deploys to every caller, not only the operator, so it stays **mechanics-only** (owner-or-pointer; the full `rk tab new` contract stays tool-owned via `rk skill`).

Code block becomes:

```sh
rk tab new [--session =<session>] --cwd <dir> --name <name> --ready --json -- <composed-argv…> ["<initial-prompt>"]
```

The bullet currently reading "**`--session`, `--cwd`, and `--name` pin where the tab lands** — there is no ambient-session guesswork. Which session is the right target is the caller's policy, not this file's." is rewritten to (shape): "**rk resolves the landing session** (role-aware default — an `_rk-*` caller never gets the window beside itself; `--session =S` overrides) and reports it as `session`/`session_rung` in the `--json` result; `--cwd`/`--name` still pin directory and name." No policy prose about *which* session is right — that is rk's.

The `--ready` bullet's list of report keys (`window_id`/`pane_id`) may mention `session`/`session_rung` in passing; nothing else in the section changes.

### G. Net shorter — measured (decision 7)

| Measure | Baseline (origin/main `3d7bd152`) | Requirement after apply |
|---------|-----------------------------------|-------------------------|
| `wc -w src/kit/skills/fab-operator.md` | 14594 | strictly lower |
| §6 step 2 word count (`sed -n 479p … \| wc -w`) | 121 (iyb4) | ≤ 60 |

Apply records both post-apply numbers in its result; review checks them.

### H. Memory (hydrate) — present truth, in place

`docs/memory/runtime/operator.md`:

- § Repo- and session-targeted spawning **item 1** (`:94`) → the delegation text: `rk tab new` resolves the landing session (role-aware default: the `_rk-*` caller is never a target; sole user session → session rooted at `--cwd`'s main worktree → most-attached → row order; "nowhere to spawn" fails loudly); `--session =<name>` only for the §8 override; the operator echoes rk's `session` + `session_rung` in one line; the ambient session is never an implicit target.
- **Item 4** (`:97`) → raw form with `[--session =<name>]`; the `--json` report carries `session`, `session_rung`, `window_id`, `pane_id`, `ready`; `track add --session` consumes `session`.
- The 5xnx paragraph (`:107`) "8 steps (target repo, target session, …)" stays valid unchanged.
- § Settings row (`:257`) → as decision E.
- Startup rk-gate paragraph (`:35`) → the probe gains the `rk tab new --help` ∋ `session_rung` half and its "predating `rk tab new`'s default-session resolution" explanation.
- Design Decision **"Spawn Target Session Is Selected Deterministically Per Spawn, Never Asked, Never Persisted-and-Trusted"** (`:447–451`) — rewritten **in place** (same heading or a present-truth retitle, e.g. "Spawn Target Session Is Resolved by `rk tab new`, Never Asked, Never Persisted-and-Trusted"): **Decision**: delegated to `rk tab new`'s role-aware default; `--session` only for the user override; rk's rung echoed; the operator carries no selector. **Why**: condense the z597/cx52/iyb4 rationale — the ambient default is wrong from `_rk-operator`; asking has negative value (fires exactly when the answer is obvious, recurs per compaction); a fab-side selector was the interim; the substrate is the right home because rk owns session roles and the tab primitive. **Rejected**: keep the earlier rejections condensed (persisted `session` field; config-only default session; ambient-as-default; ask-first / ask-as-tie-break; pane-count majority alone; more evidence tiers) and **add** "keeping the fab-side a–e selector now that rk resolves it (two copies of one policy drift)". Trail: extend `*Updated by*` with `260913-1nxw-operator-spawn-session-rk-default (selector removed; delegated to rk tab new's role-aware default behind the capability gate)`.
- Design Decision **"Spawn-Target Candidacy Delegates to `rk mux sessions`"** (`:453–457`) → retitle/rewrite so candidacy **and selection** live in rk: `rk tab new`'s role-aware default picks the session; `rk mux sessions`' roles are the same derivation (reserved constants, `reserved` catch-all). Drop "the rows' `path` and `attached` facts decide the selector's rungs (c) and (e)".
- Design Decision **"The Spawn Command Reports Its Own Landing (`rk tab new --json`)"** (`:459–463`) → add `session` (consumed by `track add --session`) and `session_rung` (echoed in the announce) to what is consumed; the Why gains "an omitted `--session` is resolved by rk and the report says where and why".
- Design Decision **"rk Is a Hard Dependency of the Operator Skill"** (`:553–557`) → the Decision's probe list gains the third half; the Rejected row's "the gate floor is the capability the clock needs" becomes "the capabilities the clock and the spawn need".

`docs/memory/runtime/agent-primitives.md` § Spawn composition (`:33`): raw form `[--session =<session>]`; replace "`--session`/`--cwd`/`--name` pin where the tab lands (no ambient-session guesswork — which session is right is the caller's policy; the operator's session selector lives in `fab-operator.md` §6)" with "rk resolves the landing session (role-aware default; `--session =S` overrides; reported as `session`/`session_rung`); `--cwd`/`--name` pin directory and name".

`docs/memory/log.md` is never hand-edited (hydrate appends). `docs/memory/runtime/index.md` is generated from frontmatter; no description change is needed (operator.md's description already says "repo/session-targeted spawning … run-kit is a hard dependency (the §2 rk Gate)").

### I. Sibling sweep before finishing apply (decision 9)

```sh
grep -rnE -- '--session =<session>|rung a|rung \(a\)|sole candidate|first row|nowhere to spawn|selector|session_rung' src/kit/ docs/memory/ | grep -v '/log.md' | grep -v 'fab/changes/archive/'
```

Update every **skill** hit; list every **memory** hit for hydrate. Known hits at intake (all verified):

| File | Line | Site | Action |
|------|------|------|--------|
| `src/kit/skills/fab-operator.md` | 77 / 80 / 83 | §2 rk Gate probe, explanation, STOP | D |
| `src/kit/skills/fab-operator.md` | 479 | §6 step 2 | A |
| `src/kit/skills/fab-operator.md` | 500–503 | step 7 raw form + parenthetical | B |
| `src/kit/skills/fab-operator.md` | 510–512 | step 8 | C |
| `src/kit/skills/fab-operator.md` | 800 | §8 row | E |
| `src/kit/skills/fab-operator.md` | 820 | §9 Key Properties "Requires HexoKit?" | D (missed by the source list) |
| `src/kit/skills/_cli-agents.md` | 87 / 91 | raw form / pin bullet | F |
| `docs/memory/runtime/operator.md` | 35, 94, 97, 257, 447–463, 553–557 | see H | hydrate |
| `docs/memory/runtime/agent-primitives.md` | 33 | see H | hydrate |

Pointers that **stay**: `fab-operator.md:588` ("target-repo + target-session → worktree → …"), `:603` ("The same eight steps run"), `:678` ("establish the change's target repo and target session"), `:679` ("steps 4–8"); memory `operator.md:107` ("8 steps (target repo, target session, …)"). The word "selector" also appears in the kit for the **`fab agent` addressing form** (`_cli-agents.md:70`, `:74`, `:172`) and for the **dispatch mode selector** (`dispatch.md`, `providers-and-profiles.md`, `_shared/configuration.md`) — unrelated; the sweep must leave those alone.

### Non-goals / constraints

- No Go changes, no migration, no run-kit changes (the rk side is shipped in v3.19.59), no CLI reference changes (no `fab` command signature moves; `_cli-fab-operator.md` is untouched).
- No renumbering of §6 steps.
- No edit to `docs/specs/operator.md` (human-curated; its v10 row is history).
- Deployed skills cite no fab-kit-only paths (Constitution V); `cd src/go/fab-kit && go test ./cmd/fab/` stays green.
- Change type: **refactor** — policy relocation with unchanged user-visible behaviour (the operator still never asks; the landing session is the one iyb4's rungs b/c/e would pick). The keyword heuristics land on `feat` (no `fix`/`refactor`/`docs`/`test`/`ci`/`chore` keyword in the description) — override to `refactor` via `fab status set-change-type`.

### Alternatives rejected

- **Renumber §6 to seven steps** — touches every "step N" cross-reference in the skill and memory for no behavioural gain; keep the slot.
- **Keep the fab-side selector alongside rk's** — two copies of one policy, the drift mechanism the owner-or-pointer rule exists to kill.
- **No capability probe** — a pre-3.19.59 rk would silently reintroduce the z597 misplacement.
- **A fab-side minimum-version check parsing `rk --version`** — fab's rk convention is capability probes (`rk cron list --json`, `rk operator --help` ∋ `--workers`, `rk skill`), never version strings.

## Affected Memory

- `runtime/operator`: (modify) § Repo- and session-targeted spawning items 1 and 4 → delegation + optional `--session` + `session`/`session_rung`; startup rk-gate paragraph → third probe half; § Settings row; Design Decisions "Spawn Target Session Is Selected Deterministically…" (rewritten in place, `*Updated by*` extended), "Spawn-Target Candidacy Delegates to `rk mux sessions`" (candidacy and selection in rk), "The Spawn Command Reports Its Own Landing" (`session`/`session_rung` consumed), "rk Is a Hard Dependency of the Operator Skill" (probe list + Rejected wording).
- `runtime/agent-primitives`: (modify) § Spawn composition raw form `[--session =<session>]` and the "operator's session selector lives in `fab-operator.md` §6" clause → "rk resolves the landing session".

## Impact

- **Files at apply**: `src/kit/skills/fab-operator.md` (§2 rk Gate incl. the §9 Key Properties restatement, §6 step 2/7/8, §8 row), `src/kit/skills/_cli-agents.md` (§ Spawn Composition raw form + one bullet). Two files, ~8 sites.
- **Files at hydrate**: `docs/memory/runtime/operator.md`, `docs/memory/runtime/agent-primitives.md` (+ `log.md` via hydrate's own append).
- **Runtime dependency**: run-kit ≥ v3.19.59 for the operator skill — enforced by the extended §2 gate, not by a version string. Users on an older rk see the existing STOP line at operator startup (with an optional upgrade hint) instead of a silently misplaced window.
- **Behavioural equivalence**: rk's `sole-user` / `cwd-root` / `most-attached` rungs correspond to iyb4's (b)/(c)/(e); iyb4's rung (d) — most panes in the target repo — has no rk analogue and is dropped (rk's `cwd-root` joins through git's common dir, so a worktree under `<repo>.worktrees/` still matches a session started at `<repo>`, covering the case rung (d) was for). rk's `caller` and `server` rungs never apply to the operator (an `_rk-operator` caller is always inside tmux and never its own target).
- **Edge case — "nowhere to spawn" after the worktree exists**: the zero-user-session failure now surfaces at step 7 (after `wt create` at step 3, pointer activation at step 4, and dependency cherry-picks at step 5) rather than at step 2. A worktree may therefore be left behind. This matches every other step-7 failure today (an unknown explicit `--session`, a `gone` verdict): the operator surfaces the error, and a later spawn of the same item respawns with `--reuse`. No new cleanup step is added.
- **Tests**: no Go code changes; `cd src/go/fab-kit && go test ./cmd/fab/` (the deployed-content portability guard) must stay green after the skill edits.
- **No** config keys, migrations, `fab` command signatures, or `.status.yaml` fields change.

## Open Questions

- None blocking. Wording-level choices (exact probe spelling, whether the STOP line gains a `brew upgrade` hint, the announce line's exact rendering, whether the envelope parenthetical moves to step 7) are apply's — see the Assumptions table.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill-prose-only change in canonical `src/kit/skills/fab-operator.md` + `_cli-agents.md`; no Go, no migration, no run-kit, no CLI reference, no deployed-copy edits | Discussed — user constraint; Constitution V and code-quality.md anti-patterns pin the canonical path; the rk side shipped in v3.19.59 (verified `rk --version`) | S:95 R:90 A:100 D:95 |
| 2 | Certain | §6 step 2 keeps its slot and number; body replaced with a delegation of ≤ 60 words; a–e rungs, candidate set, zero-candidate clause, and fab-side announce deleted; the z597 invariant sentence kept verbatim | Discussed — user decision 1 and the renumbering rejection; every "step N" cross-reference verified (`:588`, `:603`, `:678–679`, memory `:107`) | S:95 R:85 A:95 D:95 |
| 3 | Certain | Step 7 raw form becomes `rk tab new [--session =<name>] …`; `--session` only for the §8 override; unknown explicit session and rk's "nowhere to spawn" both error loudly, never a retry against the ambient session | Discussed — user decision 2; behaviour quoted from `rk tab new --help` v3.19.59 | S:95 R:90 A:95 D:95 |
| 4 | Certain | Step 8's `<session>` for `track add --session` / `track update --scope` is the `--json` report's `session` field and no other source | Discussed — user decision 3; the field is tmux's own report of the landing | S:95 R:90 A:95 D:95 |
| 5 | Certain | §2 rk Gate gains a capability half for default-session resolution; the STOP message text is unchanged (an upgrade hint MAY be appended on the same line) | Discussed — user decision 4; a pre-3.19.59 rk would silently reintroduce z597 | S:95 R:85 A:95 D:95 |
| 6 | Confident | The probe is `rk tab new --help 2>&1` piped through `grep -q session_rung`, `&&`-chained onto the existing one-liner; apply may substitute an equivalent cheap, side-effect-free probe | Recommended in the description with an explicit apply latitude (2–3 equivalent spellings, clear front-runner); precedent `rk operator --help` ∋ `--workers` in `_cli-fab-operator.md`; `session_rung` verified present in v3.19.59 help text | S:75 R:90 A:75 D:65 |
| 7 | Certain | §8 Settings row → "none — rk tab new's role-aware default (§6 step 2)" / override passes `--session =<name>`; the setting stays a pure, session-scoped user override | Discussed — user decision 5 (shape given; apply owns wording) | S:90 R:95 A:95 D:90 |
| 8 | Certain | `_cli-agents.md` § Spawn Composition: `--session` optional in the code block; the pin bullet becomes "rk resolves the landing session … `--cwd`/`--name` still pin directory and name"; mechanics-only, full contract stays tool-owned (`rk skill`) | Discussed — user decision 6; owner-or-pointer anti-pattern in code-quality.md | S:90 R:95 A:95 D:90 |
| 9 | Certain | `wc -w src/kit/skills/fab-operator.md` must be < 14594 after apply and step 2 ≤ 60 words; baseline measured at origin/main `3d7bd152` | Discussed — user decision 7; both numbers measured at intake | S:95 R:100 A:100 D:95 |
| 10 | Certain | The §9 Key Properties "Requires HexoKit?" row (`fab-operator.md:820`) is extended with the new probe half | Gap Analysis found this restatement of the gate probe; code-quality.md § Sibling Sweeps makes it must-fix | S:85 R:95 A:100 D:95 |
| 11 | Certain | Hydrate updates the memory's startup rk-gate paragraph (`operator.md:35`) and the DD "rk Is a Hard Dependency of the Operator Skill" (`:553–557`) with the third probe half, in addition to the sites the description lists | Gap Analysis; both spell the probe verbatim; sibling-sweep class "the memory file documenting a skill's behavior" | S:85 R:95 A:100 D:95 |
| 12 | Certain | Memory DDs rewritten in place per § What Changes H: the selector DD → delegation (trail extended), the candidacy DD → candidacy + selection in rk, the landing-report DD → `session`/`session_rung` consumed; `agent-primitives.md:33` repointed; `log.md` never hand-edited | Discussed — user decision 8, with the exact Decision/Why/Rejected content given | S:95 R:90 A:95 D:95 |
| 13 | Certain | Change type is `refactor` (policy relocation, user-visible behaviour unchanged), overriding the heuristic `feat` default via `fab status set-change-type` | Discussed — user directed `refactor`; the description carries no type keyword so refresh infers `feat` | S:95 R:100 A:100 D:95 |
| 14 | Confident | The "nowhere to spawn" failure is left to step 7 (a worktree from step 3 may remain; a later spawn respawns with `--reuse`); no pre-check of `rk mux sessions` and no cleanup step is added in step 2 | Discussed — user chose "surfaced as any other `rk tab new` error"; consistent with today's unknown-session / `gone` failures at step 7; cheap to revisit via `/fab-clarify` if the leftover worktree proves annoying — a pre-check vs. delegate-and-surface is a UX preference, not derivable from the code | S:70 R:85 A:70 D:65 |
| 15 | Confident | The one-line announce echoes rk's `session` + `session_rung` from the `--json` result in the spawn output, never via §5; exact rendering is apply's | Discussed — user gave the content ("echo rk's session and session_rung in one line"), not the format | S:75 R:90 A:80 D:65 |
| 16 | Confident | iyb4's rung (d) (pane-count in target repo) has no rk analogue and is dropped without replacement; rk's `cwd-root` (git common-dir join) covers the worktree case it served | Verified from `rk tab new --help`; user framed the landing as "the same one iyb4's rungs b/c/e would pick"; whether a multi-session, zero-pane case ever picked differently under (d) is unverified | S:70 R:85 A:75 D:70 |
| 17 | Confident | The run-kit envelope parenthetical iyb4 kept in step 2 moves to step 7's report sentence or is dropped (step 7 already names the envelope) | Word-budget consequence of decision 7; no behaviour attached; move vs. drop is apply's call | S:60 R:95 A:80 D:70 |
| 18 | Confident | `docs/specs/operator.md` is not edited (human-curated; its only related token is a v10 history row) and `runtime/index.md` needs no description change (generated from frontmatter; the description already covers session-targeted spawning and the rk Gate) | Verified by grep at intake; Constitution VI | S:85 R:100 A:95 D:90 |

18 assumptions (13 certain, 5 confident, 0 tentative, 0 unresolved).
