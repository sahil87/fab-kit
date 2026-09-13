# Intake: Operator Spawn Target-Session Selector — Deterministic, Never Asks

**Change**: 260913-iyb4-operator-spawn-session-selector
**Created**: 2026-09-13

## Origin

Promptless dispatch from a `/fab-discuss` root-cause session (2026-09-13), synthesized by the caller and handed to `/fab-proceed`'s create-new path. The interaction mode was conversational: the user reported the recurring failure, the discussion traced it through git history, and the user accepted the recommendation with one explicit sequencing decision (skill fix FIRST, run-kit default as a follow-up).

> The operator keeps stopping on the question "Which tmux session should this spawn into?" after the user hands it work — even when it has already computed a "(Recommended)" answer. Latest instance: the user said "execute the change rmzs in a new branch in origin remote"; the operator found the change on origin, then asked the session question with `state-sketch (Recommended)`. Many hours have been lost to this across many spawns. Fix the root cause — do NOT bolt more prose onto the skill.

Key decisions reached in the discussion (all user-accepted):

1. Replace §6 step 2's pane-count majority rule with an ordered, deterministic selector that cannot come up empty.
2. Delete the ask entirely — the step-2 torn clause, the §1 licence, and the §8 auto-set.
3. The skill prose must end up **net shorter**.
4. Correct the false memory rationale that justified the 4a8m collapse, and record the z597 → cx52 → 4a8m → iyb4 history.
5. Sibling-sweep every restatement.
6. Record the run-kit `rk tab new` default-session idea as an out-of-scope follow-up; do NOT edit run-kit.

Gap Analysis (this intake): every cited line was verified against the worktree at `c57f6584`; no active change or `fab/backlog.md` item covers this; `fab pane map --json` carries the per-pane `repo` field the selector needs; one extra sibling site was found that the discussion's list missed (`docs/memory/runtime/operator.md:21`); and one attribution was corrected — the §1 tie-break licence was added by **kp3d** (commit `aa9d08ee`, PR #665), not by 4a8m (whose §1 row read only "ask when ambiguous").

## Why

**The pain.** The operator (`src/kit/skills/fab-operator.md`) runs in its own `_rk-operator` session (since 260823-z597), so tmux's ambient default is the wrong session and the skill rightly forbids it. Every spawn therefore has to choose a target session. Today that choice rests on ONE evidence signal — §6 "Spawning an Agent" step 2 (`fab-operator.md:479`): "pick the candidate holding the most panes whose `cwd` is under the target repo". On the single most common spawn — start work on a repo nothing is open in yet — that count is zero in every candidate, every candidate ties, the rule declares itself torn, and the operator asks. Even with exactly one user session on the server, it asks.

**Why the tie-break does not save it.** The fallback is the §8 "Spawn target session" setting (`fab-operator.md:800`), which is session-scoped and resets on every compaction, `/clear`, or restart (§8 footnote; §4 Post-Compaction Reload). In a long-lived operator "ask once" therefore means "ask once per compaction". The memory itself records this as the reason a cold-start ask was previously rejected (`docs/memory/runtime/operator.md:450`: "recurs every session because the §8 setting is session-lifetime-scoped").

**This is a regression.** Change 260904-cx52 hit exactly this failure (the operator computed "fabKit (Recommended)" then asked anyway) and fixed it with evidence tiers — monitored agents → pane-map repo affinity → §8 setting → tier (d) "structural dominance: exactly one plausible candidate session remains → pick it", with `attached`/`windows` as supporting signals (`git show 20e287f2^:src/kit/skills/fab-operator.md`, "Establish target session"). Change 260911-4a8m (commit `20e287f2`, PR #663) then collapsed step 2 to the bare pane-count majority rule and deleted tier (d) as "speculative generality", writing into memory the rationale "the pane-count majority over `role: user` rows decides every observed case" — a claim that was false when written, because cx52's own reproduction was a zero-count case. Change 260911-kp3d then added to the §1 Coordinate-don't-execute row an explicit licence for the question ("plus the target-session tie-break in §6 Spawning an Agent step 2", `fab-operator.md:37`). A model holding a permitted question, a weak signal, and an AskUserQuestion "(Recommended)" widget converts any lean into a confirmation.

**Why the operator should decide trivially.** `rk mux sessions --json` already returns, per user-role session, `path` (the directory the session was started in — on the user's machines a user session is started in the repo it serves) and `attached` (viewer count). Neither is consulted by the current rule; cx52 explicitly demoted them to "supporting, never deciding". Verified live during the discussion and again at intake:

```json
{"ok":true,"result":[{"name":"fab_kit","role":"user","attached":0,"windows":16,
  "path":"/home/sahil/code/sahil87/fab-kit","grouped":false}]}
```

With one user session, `path` equal to the target repo, and 17 panes in that repo, the current rule still asked. A selector that reads what is already fetched never needs to.

**If we do nothing.** Every operator spawn after a compaction pays a round-trip the user has to be present for, which defeats unattended operation (queue, Linear/Slack ticks) and burns attended time; the memory's false rationale stands ready to justify a fourth collapse.

**Why this approach.** The skill fix stops the ask today, in one repo, with no release coupling. The alternatives are recorded under § Alternatives Rejected below.

## What Changes

### A. §6 "Spawning an Agent" step 2 — replace the majority rule with the ordered selector

**File**: `src/kit/skills/fab-operator.md`, the single list item at line 479 (owner site; every other site points here).

**Current** (123 words):

> 2. **Establish target session** — determine the tmux session the new agent window must land in; it is passed explicitly at step 7. Candidates are `rk mux sessions --json` rows (the `result` array in run-kit's envelope form; the same holds for every `rk … --json` read) with `role: "user"` minus the operator's own session; pick the candidate holding the most panes whose `cwd` is under the target repo (from the tick's `rk mux panes --json` snapshot, or `fab pane map --all-sessions` on demand). Tie → the §8 "Spawn target session" setting; still torn → ask once when attended, notify via §5 when unattended. Announce the choice and auto-set §8. Never trust a persisted `scope.session` alone; the ambient session is never an implicit target.

**Replacement** — the selector, evaluated in this order over the candidate set (`rk mux sessions --json` rows — the `result` array in run-kit's envelope form — with `role: "user"`, minus the operator's own session; the exclusion rule is unchanged). The first rung that decides wins; the selector cannot come up empty for a non-empty candidate set:

| Rung | Rule | Evidence source |
|------|------|-----------------|
| a | An explicit **user** override — the §8 "Spawn target session" setting when the user set it ("spawn into session X") | §8 setting |
| b | Exactly **one** candidate → it (no evidence needed; a single user session is the normal setup) | candidate count |
| c | The candidate whose `path` **equals** the target repo's main-worktree root (step 1's value) | `rk mux sessions` `path` |
| d | The candidate holding the **most panes whose `repo` is the target repo** | `fab pane map --all-sessions --json` `repo` (absolute main-worktree root) |
| e | The candidate with the highest `attached`; a residual tie falls to the **first candidate in `rk mux sessions` order** | `rk mux sessions` `attached`, row order |

**Zero candidates** (no `role: "user"` session on the server) is a loud error surfaced to the user — "nowhere to spawn" — never a question. Attended, it lands in the spawn output; unattended (queue / Linear-Slack tick), it rides the §5 Notification Send path like any other escalation. The spawn is not attempted.

**Default-and-announce stays, reduced to one line**: name the chosen session and the rung that decided it, e.g. `→ session fab_kit (rung c: path = /home/sahil/code/sahil87/fab-kit)`. The exact spelling is apply's call; the contract is *one line, session + deciding rung*.

**Deleted from step 2**: "Tie → the §8 … setting; still torn → ask once when attended, notify via §5 when unattended. Announce the choice and auto-set §8." The sentence "Never trust a persisted `scope.session` alone; the ambient session is never an implicit target" stays (it is the z597 invariant and is not about asking).

**The worktree wrinkle that makes rung (d) read `repo`, not `cwd`**: worktrees live in a SIBLING directory (`<repo>.worktrees/<name>`), not under the repo, so a `cwd`-under-repo prefix match misses every worktree pane. `fab pane map --all-sessions --json` rows carry `repo` = the absolute main-worktree root (`_cli-fab-pane.md` § map, JSON field table: "`repo` — `string|null`; absolute main-worktree root; JSON-only"), and step 1's "target repo" is already defined as that same absolute main-worktree root, so **equality** is the correct join for both rung (c) and rung (d). The old "(from the tick's `rk mux panes --json` snapshot …)" source clause goes with the old rule.

**Word budget**: the replacement MUST be ≤ 123 words for the step-2 item (a rung table counts toward it), and the whole file's word count MUST be lower after apply than before (baseline 14,604 words at `c57f6584`). Deleting the tie/ask machinery and the old source clause pays for the rungs.

### B. §1 Principles — remove the question licence

**File**: `src/kit/skills/fab-operator.md:37`, the "Coordinate, don't execute" row.

Delete the parenthetical `(plus the target-session tie-break in §6 Spawning an Agent step 2)` so the sentence reads: "The only thing the operator asks about a work request **or a spawn** is **which repo** — never whether to spawn, and **never for a task** …". Nothing else in the row changes. (Introduced by kp3d, `aa9d08ee`.)

### C. §8 Settings — the row becomes a pure user override

**File**: `src/kit/skills/fab-operator.md:800`.

**Current**: `| Spawn target session | inferred (§6 step 2 majority rule; auto-set on each announced inference) | "spawn into session {name}" |`

**Replacement** (shape): `| Spawn target session | none — §6 step 2's selector decides; set only by the user (rung a) | "spawn into session {name}" |`

The footnote "These settings are session-scoped and reset on compaction, `/clear`, or session restart" stays as is — it is now harmless, because the selector no longer depends on a remembered value for consistency across spawns. The user override is still validated by the existing rule that an unknown session errors loudly at spawn (no new validation prose).

### D. Cross-reference sites that point at step 2 (expected stable — verify, reword only if needed)

- `fab-operator.md:593` (§ Working a Change): "Every spawn runs §6's target-repo + target-session → worktree → …" — pointer, stays.
- `fab-operator.md:678` (§ Queues step 2): "run the §6 spawn sequence steps 1–3 (establish the change's target repo and target session, …)" — pointer, stays.
- `fab-operator.md:37` §1 "Multi-repo aware" row and § Spawning an Agent intro paragraph ("repo-targeted and session-targeted") — unchanged.

### E. Memory (lands at hydrate; listed here so apply's sweep and hydrate agree)

`docs/memory/runtime/operator.md`:

- **line 21** (§ Principles, "Coordinate, don't execute" restatement — the site the discussion's list missed): drop "(plus the target-session tie-break in §6 Spawning an Agent step 2)".
- **line 94** (§ Repo- and session-targeted spawning, item 1 "Derive the target session"): rewrite to state the five-rung selector, the zero-candidate error, and the one-line announce; delete the tie/ask/auto-set/"genuinely torn" prose. Keep the `rk tab new --session =<session>` pin, "the ambient session is never an implicit target", and "an unknown session errors loudly at spawn" sentences.
- **line 257** (§ Settings table): same row rewrite as C.
- **lines 447–451** (Design Decision "Spawn Target Session Is Inferred Live Per Spawn by Majority Rule, Never Persisted-and-Trusted"): rewrite **in place** (FKF present-truth — the body stays present tense; history lives in the DD trail):
  - retitle to reflect selection, e.g. "Spawn Target Session Is Selected Deterministically Per Spawn, Never Asked, Never Persisted-and-Trusted";
  - **Decision**: the ordered selector a–e, zero candidates = loud error, one-line announce, user override only via §8;
  - **Why**: keep the z597 ambient-misplacement story and the cx52 "asks exactly when the answer is obvious" finding; add that the pane-count-only rule ties at zero on the most common spawn (a repo with nothing open yet), that the §8 tie-break resets on compaction so "once" recurs, and that `rk mux sessions` already supplies `path`/`attached`;
  - **Rejected**: keep persisted `session` field, `operator.spawn_session` config key, ambient-as-default, ask-first cold start; REPLACE the final clause "evidence tiers beyond the majority rule (the pane-count majority over `role: user` rows decides every observed case; tiers were speculative generality)" with the corrected record — that claim was false when written (cx52's reproduction was a zero-count case) and the collapse regressed cx52; add "keeping the ask as a tie-break", "run-kit default first" (changes nothing while the skill pins `--session`), and "more tiers on top of the majority rule" (prose growth);
  - trail: `*Introduced by*: 260823-z597-…; evolved by 260904-cx52-…; *Updated by*: 260911-4a8m-… (collapsed to the majority rule — regression); 260913-iyb4-operator-spawn-session-selector (deterministic selector, ask deleted)`.
- The neighbouring DD "Spawn-Target Candidacy Delegates to `rk mux sessions`" stays; its "`attached`/`windows` facts feed the announcement" clause may be extended to note `path`/`attached` now decide rungs c/e.
- `docs/memory/runtime/log.md:10` is the cx52 log entry — historical, NOT edited; hydrate appends its own entry.

`docs/memory/runtime/agent-primitives.md:33`: "the operator's majority rule lives in `fab-operator.md` §6" → "the operator's session selector lives in `fab-operator.md` §6" (pointer wording only).

### F. Sibling sweep (code-quality.md § Sibling Sweeps) — before finishing apply

grep repo-wide (excluding `fab/changes/archive/` and `docs/memory/*/log.md`) for each phrase and update every hit in the skill; list memory hits for hydrate:

```
majority rule | Spawn target session | genuinely torn | tie-break | auto-set | structural dominance | cwd` is under the target repo
```

Known hits at intake: `fab-operator.md:37, :479, :800`; `docs/memory/runtime/operator.md:21, :94, :257, :447–451`; `docs/memory/runtime/agent-primitives.md:33`. `docs/specs/operator.md` has no session-selection prose (verified). `fab-new.md:49` "Tie-breaker (default-closed)" is unrelated (micro-change backstop) — leave it.

### G. Follow-up recorded, out of scope — run-kit default session for `rk tab new`

`rk tab new` should resolve a sane default session when the caller sits in an `_rk-*` session: the sole user session; else the session whose `path` is the cwd's git common dir; else the attached one. With that in place fab can later drop step 2 and the `--session =<session>` flag entirely behind the existing "capable HexoKit" gate. cx52 already named run-kit as the right long-term home and deferred it. Not built here, not filed in run-kit's backlog by this change (the user said do NOT edit run-kit); mention it in hydrate's log entry and the PR body so it is findable.

### Alternatives Rejected (from the discussion)

- **Run-kit default FIRST**: on its own it changes nothing — the skill pins `--session =<session>` on every spawn and treats an unknown session as a loud error, so the operator keeps deciding until the skill stops passing the flag; it also needs an rk release plus a fab version gate. The skill fix stops the ask today.
- **Keeping the ask as a tie-break**: the §8 setting resets on compaction so "once" recurs; the ask fires precisely when the answer is obvious (cx52's own finding).
- **A persistent config key (`operator.spawn_session`)**: rejected in z597/cx52 as a staleness trap (session names are per-server and ephemeral); still rejected.
- **More conditions/tiers on top of the majority rule**: grows prose; the user vetoed prose growth.

### Constraints

- Skill-prose-only change in the canonical `src/kit/skills/fab-operator.md` — never the deployed `.agents/skills/` or `.claude/skills/` copies (Constitution V; code-quality.md anti-pattern "Editing deployed skills directly").
- No Go changes, no migration, no run-kit changes, no CLI reference changes (no command signature moves).
- Owner-or-pointer: step 2 owns the selector; §1, §8, § Working a Change, § Queues, and memory point at it or restate only what they own (the §8 row owns the override phrase).
- Deployed content cites no fab-kit-only paths (Constitution V 1.8.0) — the skill edit references `_cli-fab-pane.md` § map and `rk` commands only.
- Net-shorter: step-2 item ≤ 123 words; `wc -w src/kit/skills/fab-operator.md` < 14,604 after apply.

## Affected Memory

- `runtime/operator.md`: (modify) § Principles line 21 licence removal; § Repo- and session-targeted spawning item 1 → five-rung selector + zero-candidate error + one-line announce; § Settings row → pure user override; Design Decision block at 447–451 rewritten in place with the corrected rationale and the z597 → cx52 → 4a8m (regression) → iyb4 trail; hydrate log entry mentions the run-kit follow-up.
- `runtime/agent-primitives.md`: (modify) § Spawn composition pointer wording "majority rule" → "session selector" (no rule content changes).

## Impact

- **Code areas**: one skill file at apply — `src/kit/skills/fab-operator.md` (§1 row, §6 step 2, §8 row; §6 cross-refs verified). Two memory files at hydrate.
- **Behavior contract**: the operator's spawn sequence gains a deterministic target-session selection and loses its only remaining spawn-time question other than "which repo". Unattended spawns (queue, Linear/Slack) can no longer stall on the session choice; the only new failure surface is the explicit zero-candidate error (no user session at all on the server).
- **Dependencies**: `rk mux sessions --json` (`role`, `path`, `attached`, row order — all present on the installed run-kit), `fab pane map --all-sessions --json` (`repo` field). No new binaries, flags, or config keys.
- **Tests**: none — prose-only; the Constitution V citation guard (`src/go/fab-kit/cmd/fab/` test) runs unchanged and must stay green (the skill edit cites no fab-kit-only paths).
- **Scale**: fix-tier; one apply task cluster (skill) + sweep; hydrate on two files.
- **Change type**: `fix` — regression of cx52's behaviour introduced by 4a8m.

## Open Questions

None blocking. Every decision the discussion left to apply or hydrate is recorded as a graded row below; no row required a user answer under promptless dispatch.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Skill-prose-only fix in canonical `src/kit/skills/fab-operator.md`; no Go, no migration, no run-kit, no deployed-copy edits | Discussed — user constraint; Constitution V and code-quality.md anti-patterns pin the canonical path | S:95 R:90 A:100 D:95 |
| 2 | Certain | Change type `fix` (regression of cx52 introduced by 4a8m) | Discussed — regression verified in git history (`20e287f2` removed cx52's tier (d)) | S:95 R:95 A:90 D:90 |
| 3 | Certain | Selector rung order a → e exactly as decided; candidate set = `rk mux sessions --json` `role: "user"` rows minus the operator's own session | Discussed — user accepted the recommendation verbatim; exclusion rule unchanged | S:95 R:80 A:85 D:90 |
| 4 | Certain | The ask is deleted entirely: step-2 torn clause, §1 licence, §8 auto-set; zero candidates = loud error, never a question | Discussed — user decision 2; the §8 reset-on-compaction fact makes any ask recur | S:95 R:80 A:85 D:90 |
| 5 | Certain | Net-shorter target is measurable: step-2 item ≤ 123 words, file total < 14,604 words after apply | Discussed — user vetoed prose growth; baselines measured at intake on `c57f6584` | S:90 R:90 A:85 D:85 |
| 6 | Certain | Rung (d) reads `fab pane map --all-sessions --json` `repo` (absolute main-worktree root) and joins by equality; the tick's `rk mux panes` `cwd` snapshot is not used for this rung | Verified: `_cli-fab-pane.md` § map documents `repo`; worktrees are siblings so `cwd` prefix-matching misses them | S:80 R:85 A:80 D:75 |
| 7 | Certain | Rung (e) picks the highest `attached`; a residual tie falls to the first row in `rk mux sessions` order | Discussed — decided in the recommendation; row order is the only remaining deterministic signal | S:90 R:85 A:80 D:85 |
| 8 | Certain | §8 "Spawn target session" row stays as the pure user override (rung a) with default "none — selector decides"; the override phrase and session-scoped footnote are unchanged | Discussed — user decision 2 keeps the row "purely as the user override" | S:85 R:90 A:80 D:80 |
| 9 | Certain | Cross-ref sites `fab-operator.md:593` and `:678` are pointers and stay; reword only if step 2's label changes | Verified at intake — both say "target session" without restating the rule | S:70 R:95 A:85 D:80 |
| 10 | Certain | Memory edits (`operator.md:21, :94, :257, :447–451`; `agent-primitives.md:33`) land at hydrate, not apply; `log.md:10` (cx52 entry) is historical and untouched | Pipeline convention — memory is hydrate's; logs are append-only history | S:90 R:90 A:90 D:90 |
| 11 | Certain | `docs/memory/runtime/operator.md:21` (§ Principles restatement) joins the sweep class although the discussion's list omitted it | code-quality.md § Sibling Sweeps — grep found it carrying the same licence parenthetical | S:80 R:95 A:95 D:95 |
| 12 | Confident | Rung (c) is exact equality between the session `path` and step 1's main-worktree root (normalized for trailing slash / symlinks); a session started in a subdirectory or a worktree falls through to (d) | Discussed — "equality against the session `path` is the right join"; a prefix rule would add conditions the user vetoed | S:70 R:85 A:70 D:55 |
| 13 | Confident | Announce = one line naming session + deciding rung, in the spawn output; exact spelling is apply's call; no §5 notification for a deterministic pick | Discussed — "one line naming the chosen session and which rung decided it"; a notification for a non-decision would re-create the interruption | S:75 R:90 A:75 D:70 |
| 14 | Confident | Zero-candidate error surfaces in the spawn output when attended and via §5 Notification Send when unattended; the spawn is not attempted | Discussed — "loud error surfaced to the user"; §5 is the operator's only unattended channel | S:70 R:85 A:75 D:70 |
| 15 | Confident | The Design Decision block is rewritten in place (retitled, Decision/Why/Rejected corrected, `*Updated by*` trail extended) rather than adding a second DD entry; the z597 → cx52 → 4a8m → iyb4 history lives in the DD trail, the body stays present-truth | FKF present-truth style; the block already carries an `*Updated by*` trail from 4a8m | S:70 R:85 A:80 D:65 |
| 16 | Confident | Rung (a)'s user override needs no new validation prose — the existing "an unknown session errors loudly at spawn" rule covers a mistyped name | Verified — rule present in step 2's memory restatement and `rk tab new --session =<session>` semantics | S:65 R:90 A:80 D:75 |
| 17 | Confident | The run-kit follow-up is recorded in this intake, hydrate's log entry, and the PR body only — not filed in run-kit's backlog by this change | Discussed — user said do NOT edit run-kit; filing it there is the user's call | S:60 R:95 A:60 D:50 |
| 18 | Confident | Step 2 keeps a shortened run-kit envelope parenthetical ("the `result` array in run-kit's envelope form") because it is the operator skill's only general `rk … --json` envelope note; the "same holds for every `rk … --json` read" tail may go if the word budget needs it | Verified — `_cli-agents.md`/`_cli-fab-operator.md` own the shape, but the skill's general aside lives only here (zx5k) | S:50 R:95 A:65 D:45 |

18 assumptions (11 certain, 7 confident, 0 tentative, 0 unresolved).
