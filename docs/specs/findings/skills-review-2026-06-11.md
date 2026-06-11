# fab-kit Skills Review — 2026-06-11

> Multi-agent review of `src/kit/skills/` (30 files, ~5,974 lines): 16 independent reviewers (11 cluster deep-reads + 5 cross-cutting sweeps over duplication, convention conformance, context budget, staleness, DX), findings consolidated by a triage pass, every high/medium finding adversarially verified by a dedicated refuter agent, then a completeness critic spawned 4 follow-up sweeps (idempotency vs Constitution III, fab-CLI failure handling, skill/template coupling, frontmatter description accuracy). ~194 agents, ~8.5M tokens.

**Totals**: 296 raw findings → 201 consolidated → **134 confirmed** (5 gap-phase duplicates merged), 16 refuted by verifiers (Appendix A), 75 low-severity findings reported unverified (Appendix B).

IDs (`f###` main run, `g#-#` follow-up sweeps) are stable for cross-referencing. Line numbers refer to the state of the tree at commit ae79e04c.

**Severity**: 7 high · 76 medium · 51 low (verifier-adjusted)


## 1. Correctness & contradictions (49)

### `f018` [HIGH] Watch dedup hole: moving item IDs from `known` to `completed` re-enables spawning

**Files**: fab-operator.md

Tick step 2 deduplicates only against `known`. When a watch-spawned agent reaches stop_stage, the item ID is moved out of `known` into `completed`. A Linear issue that still matches the query (e.g., stop_stage=intake leaves the issue open) is re-detected the next tick and respawned in a loop. This is a behavior bug, not phrasing.

**Recommendation**: In §7 Tick Behavior step 2, deduplicate against `known` plus `completed`; or change 'move the item ID from known to completed' to 'add to completed (retain in known until pruned)'.

**Evidence**: fab-operator.md:587 'Deduplicate — skip items in `known` list'; :596 'move the item ID from `known` to `completed`'.

**Verifier (high confidence)**: Confirmed: fab-operator.md:587 dedups only against `known`; :596 moves IDs to `completed`; YAML example (152-154) shows DEV-985 absent from `known`. No mitigation: monitored entry is removed at stop_stage (:188), so concurrency counts can't block respawn, and intake doesn't change Linear status, so the query re-matches — respawn loop is real. Fix is local: no Go code or other skill parses watch fields; SPEC mirror has no dedup detail. Also fixes contradiction with `known` = "already-handled item IDs". High severity stands.


### `f024` [HIGH] Bootstrap step 1a never configures a fab-init-created config.yaml

**Files**: fab-setup.md, init.go, sync.go

Canonical flow is `fab init` then `/fab-setup` (README:130-138). init.go writes config.yaml containing only `fab_version` BEFORE sync's copy-if-absent runs, so the scaffold template with `{PROJECT_NAME}` is never copied. Step 1a's trigger ('missing or raw template') is false, so bootstrap reports 'config.yaml already exists — skipping' and project.name/description/source_paths are never collected.

**Recommendation**: Change step 1a's trigger (fab-setup.md:75) to: missing, raw template, OR missing required fields `project.name`/`project.description`. This is an intentional behavior change. Also ensure Config Create Mode preserves an existing `fab_version` key (the scaffold template lacks it; overwriting from template would break the router's version resolution).

**Evidence**: fab-setup.md:75-76 ('If exists and not a raw template: report "config.yaml already exists — skipping"'); init.go:34-39 (setFabVersion writes fab_version-only YAML before Sync); sync.go:313-324 (scaffold copy-if-absent skips existing config.yaml); README.md:135-138 ('Then in your AI agent: /fab-setup').

**Verifier (high confidence)**: Confirmed end-to-end: init.go writes fab_version-only config before sync's copy-if-absent (sync.go:315), so step 1a (fab-setup.md:75-76) skips and project/source_paths/stage_directives defaults are silently lost on every canonical new-project install. Doctor checks tools only; no later phase catches it. Strongest counter: user can manually run /fab-setup config, and nothing hard-crashes — but bootstrap falsely reports success. fab_version-preservation caveat is real (config.go:69 errors without it). Also update pre-flight line 181.


### `f005` [MEDIUM] fab-continue dispatch table prescribes invalid state transitions (start vs reset)

**Files**: fab-continue.md, status.go · merged from 4 reports

The Step 1 table instructs 'finish intake → start apply', but finish auto-activates apply to active and `fab status start` only accepts pending — the call errors, and _preamble explicitly says never call start after finish. The review-fail row instructs 'start <change> apply' while apply is done (start invalid); the Verdict section correctly uses reset. Internal contradiction plus guaranteed CLI errors on the main path.

**Recommendation**: In fab-continue.md Step 1 (lines 43 and 49), change to 'finish intake (auto-activates apply) → execute apply'. In the review row (line 52), change 'start <change> apply fab-continue' to 'reset <change> apply fab-continue' to match the Verdict section.

**Evidence**: fab-continue.md:43 'Finish intake, start apply, then execute apply'; :49 'finish intake → start apply → execute apply'; :52 'Fail: run `fail <change> review` then `start <change> apply fab-continue`' vs :150 Verdict uses `fab status reset <change> apply`. _preamble.md:256 'never call start after finish'. status.go:38-41: start From:[pending]; Finish auto-activates next stage to active (preflight verified, status.go:160-168).

**Verifier (high confidence)**: All quotes verified: lines 43/49/52 prescribe `start` where status.go (start From:[pending], Finish auto-activates next stage) guarantees CLI errors; line 150 and _preamble.md:256 contradict them. fab-ff.md:52,70 and fab-fff.md:52,70 already use the recommended phrasing. Counter-evidence for severity: state-machine guards prevent corruption — errors are loud and recoverable, and correct commands sit nearby (Step 4, Verdict) — so medium, not high. SPEC mirrors (skills.md:298, SPEC-fab-continue.md:32) need the same edit (normal procedure).


### `f006` [MEDIUM] Status-transition ownership contradicts between orchestrators and dispatched sub-behaviors — double finish/fail errors

**Files**: fab-continue.md, fab-ff.md, fab-fff.md

ff/fff dispatch fab-continue's Apply/Review Behavior — whose own text runs finish/fail/reset with driver fab-continue — then the orchestrator runs the same transitions again with its driver. The CLI rejects finish from done (status.go:40, From:[active,ready]), so the second call errors. Hydrate inverts ownership (subagent runs finish, orchestrator's driver). A subagent following Review Behavior may also hit the interactive rework menu.

**Recommendation**: Pick one owner. Either: (a) ff/fff subagent prompts explicitly say 'do NOT run fab status commands; return results only' and orchestrator owns all transitions; or (b) subagent owns them and ff/fff drop their 'On success: run fab status finish...' lines. Add a 'when invoked as subagent, skip §Verdict / step-7 finish' rule to fab-continue's behavior sections.

**Evidence**: fab-ff.md:60 'On success: run fab status finish <change> apply fab-ff' vs fab-continue.md:134 (Apply step 7 runs finish apply fab-continue); fab-ff.md:68/70 vs fab-continue.md:148/150 (Verdict runs finish/fail+reset itself); fab-ff.md:101 subagent 'runs fab status finish <change> hydrate fab-ff' vs fab-continue.md:175 (driver fab-continue); _review.md:16-18 says verdict transitions 'remain in each orchestrator's own file'. Same pattern: fab-continue.md:54 re-finishes ship that git-pr.md:236 already finished.

**Verifier (high confidence)**: All cited lines verified; status.go:40 rejects finish-from-done, and the preamble dispatch rule ("follow the behavior section") gives subagents the finish steps, so double transitions are textually mandated. Hydrate driver inversion (fab-ff.md:101 vs fab-continue.md:175) and ship re-finish (fab-continue.md:54 vs git-pr.md guarded finish) confirmed. Downgraded to medium: errors can't corrupt state, ff/fff's "returns findings/status" wording plus the "manual rework — /fab-continue only" marker partially scope the subagent. Fix is cheap; only SPEC mirrors need updating.


### `f012` [MEDIUM] Review pass criterion is hedged: 'may pass' is the pipeline's only pass rule

**Files**: _review.md, fab-continue.md

Findings Merge step 4: 'any must-fix → fails' is deterministic, but the converse is 'review may pass'. No file defines when it does NOT pass — fab-continue's Verdict and fab-ff/fff consume the pass/fail result without criteria. One agent passes on should-fix-only findings; another exercises 'may' discretion and fails, burning bounded auto-rework cycles. The zero-findings case is also not literally covered.

**Recommendation**: This is a behavior decision: make step 4 deterministic — e.g., 'no must-fix findings (including zero findings) → review passes' — or explicitly define the discretion conditions. State the chosen rule once in _review.md Findings Merge; orchestrators already defer here.

**Evidence**: _review.md:154-155: 'If **any must-fix** finding exists ... → review **fails**' / 'If only should-fix and/or nice-to-have findings remain ... → review **may pass**'. fab-continue.md:146-150 (Verdict: 'Pass: Run fab status finish...' with no criterion); fab-ff.md:66 ('returns structured findings ... with pass/fail status').

**Verifier (high confidence)**: Verified: _review.md:154-155 hedges with "may pass"; fab-continue Verdict and fab-ff/fff consume pass/fail with no criterion; mirror at docs/specs/skills.md:504. SPEC-_review.md:62 already states the deterministic form, and the constitution mandates unattended post-intake operation — both favor the fix. No code/cross-ref breakage; "may" has no documented rationale (wording from #113). Severity inflated: must-fix→fail is deterministic, so worst case is wasted bounded rework cycles, not bad ships.


### `f013` [MEDIUM] git-* allowed-tools allowlists exclude fab/yq/edit tools their bodies require

**Files**: git-branch.md, git-pr-review.md, git-pr.md · merged from 3 reports

git-pr declares `allowed-tools: Bash(git:*), Bash(gh:*)` yet runs `fab status start/finish`, `fab change resolve`, `fab pr-meta`. git-pr-review additionally runs `yq -i` and must Read/Edit source files to apply fixes. git-branch runs `fab change resolve` with only `Bash(git:*)`. Unlisted commands trigger permission prompts (or blocks), breaking the "fully autonomous — no prompts" contract, especially when dispatched as subagents by fab-fff Steps 4–5.

**Recommendation**: Add `Bash(fab:*)` to all three; add `Bash(yq:*)`, Read, and Edit to git-pr-review's allowed-tools — or drop the allowed-tools field entirely to match every other skill in the kit.

**Evidence**: git-pr.md:4 (frontmatter) vs :22 (`fab status start`), :178 (`fab pr-meta`), :226 (`fab status add-pr`); git-pr-review.md:4 vs :219-221 (`yq -i`), :125-129 (read/edit files for fixes); git-branch.md:4 vs :44-50 (`fab change resolve`)

**Verifier (high confidence)**: Mismatch confirmed at all cited lines; only these 3 skills carry allowed-tools. But severity is inflated: fab-fff dispatches via general-purpose Agent tool ("full tool access", _preamble.md), where frontmatter isn't enforced; operator spawn uses --dangerously-skip-permissions (spawn.go:10); scaffold fragment-settings.local.json already allowlists Bash(fab:*)/git/gh. Residual gap: yq and Edit in direct interactive use. Fix is safe — SPECs, Go code, and constitution never reference allowed-tools.


### `f015` [MEDIUM] git-pr-review: 'STOP' in Steps 1-4 contradicts Step 6's status routing

**Files**: fab-fff.md, git-pr-review.md

Terminal paths in Steps 1-4 say 'STOP', but Step 6 routes those same outcomes to fab status finish/fail ('no PR found' -> fail; 'no actionable comments' -> finish; 'no reviews' -> finish). A literal STOP skips Step 6 entirely, leaving review-pr active; fab-fff explicitly expects 'no reviews -> stage done'. Two agents diverge on stage state.

**Recommendation**: Replace each terminal 'STOP' in Steps 1, 2, and 4 with 'go to Step 6 with outcome {success|failure|no-reviews}', and state in Step 6 that it is the single exit point for all terminal paths after Step 0 resolution.

**Evidence**: git-pr-review.md:33-34, :66, :97-98, :134 ('print `No actionable comments.` and STOP') vs :186-188 (Step 6 outcome table); fab-fff.md:113 ('If no reviews found: the stage completes as `done`')

**Verifier (high confidence)**: Confirmed: Steps 1/2/4 say bare "STOP" while Step 6 (:186-188) routes those exact outcomes to finish/fail; fab-fff.md:113 expects no-reviews→done. SPEC mirror :77/:115 gates Step 6.5 on the "no-reviews path", proving intent that Step 6 runs. git-pr.md:125 shows the convention ("record... Then STOP") missing here. No cross-refs break. Downgraded to medium: Step 6's table is in-context so divergence is probabilistic, and status writes are best-effort/recoverable.


### `f016` [MEDIUM] git-pr-review: Copilot-timeout path can mark review-pr done while telling the user to re-run

**Files**: git-pr-review.md

The 10-minute timeout prints 'Re-run /git-pr-review to process when ready' (line 95), but Step 6 has no class for it — if treated as 'no reviews found' it calls finish, marking review-pr done with the review still pending (State Table then says /fab-archive). And start only handles pending/failed->active (line 25), so re-run can't reactivate a done stage.

**Recommendation**: Add an explicit fourth outcome to Step 6: 'timeout — review requested but pending: leave the stage active, no finish, no fail'. Keep the re-run message.

**Evidence**: git-pr-review.md:95 ('Copilot review requested but not yet available. Re-run /git-pr-review... STOP (clean finish — no error, no fail event)') vs :188 ('On no reviews... Call fab status finish'); :25 ('start command handles both pending and failed -> active')

**Verifier (high confidence)**: All citations verified (lines 95, 186-188, 25; status.go:48 confirms start won't reactivate done). Counter "STOP skips Step 6" fails: Step 6 enumerates Step 1's no-PR STOP, and 6.5 gates on the no-reviews finish path — timeout plausibly maps to finish→done, then fab-fff:48/operator:188 treat pipeline complete with review pending. Fix is safe; SPEC mirror update is normal. Downgraded: nondeterministic (needs >10min Copilot delay + agent picking class 3) and recoverable via manual re-run.


### `f017` [MEDIUM] Operator must read/write server-keyed state file but is told never to compute its path; no CLI exposes it

**Files**: fab-operator.md, operator_tick_start.go

The skill requires the operator to read state at init/tick and persist after every change, yet says the binary derives the path and 'the operator does not compute it'. No fab command prints the path (StatePath is internal; tick-start outputs only tick/now and writes only tick_count/last_tick_at). Init also claims the file 'is created' with monitored/autopilot/branch_map — no actor does that.

**Recommendation**: Add a `fab operator state-path` subcommand (or make `tick-start` emit `state: <path>`), document it in _cli-fab.md, and rewrite §2 Init step 1 and §4 tick steps 1/6 to name that mechanism explicitly. Also have init create the file with the full schema (including `watches: {}`).

**Evidence**: fab-operator.md:65 'the binary derives the path via `fab operator tick-start` — the operator does not compute it'; :121 'the operator never needs to compute it'; :168 'After writing the monitored entry to the server-keyed state file'; :277 'Persist — write updated state'. operator_tick_start.go:34 uses internal StatePath; stdout is only 'tick: N / now: HH:MM'.

**Verifier (high confidence)**: Quotes verified: fab-operator.md:65/121 forbid computing the path; only tick-start/time subcommands exist; tick-start prints only tick/now (operator_tick_start.go:77). But impact is overstated: _cli-fab.md:426 (frontmatter helper, loaded at startup) documents the full derivation incl. slugify example, so the operator can locate the file despite the prose. Contradiction real, behavior not broken. Fix is additive — operator_test.go uses strings.Contains, so a state: line breaks nothing; constitution mandates the doc updates anyway. High → medium.


### `f019` [MEDIUM] Rework-exhaustion path is ambiguous: guidance promises a menu /fab-continue cannot show, and the failed-review state has no dispatch row

**Files**: fab-continue.md, fab-ff.md, fab-fff.md

After 3 failed cycles, fab-ff/fab-fff say 'Run /fab-continue for manual rework options', but fab-continue's rework menu only appears after its own review fails — at that point it re-runs apply/review autonomously instead. Status events for cycles 2-3 are unspecified (fail+reset only stated for the first failure), so the final stage state is undefined, and fab-continue's dispatch table has no row for review=failed (Step 1 only handles `pending`).

**Recommendation**: In fab-ff Step 2 / fab-fff Step 2, specify the status events fired on each cycle's re-review failure and the exact terminal state at exhaustion; add a review-failed row to fab-continue's dispatch table (Step 1) that presents the rework menu directly; reword the stop message to describe what /fab-continue will actually do.

**Evidence**: fab-ff.md:85-95 stop message; fab-ff.md:70 fail+reset specified once; fab-fff.md:77-85 same; fab-continue.md:40 'If progress is `pending`, run fab status start' (failed unhandled); fab-continue.md:52 review row covers only `active`/`ready`; menu defined only at fab-continue.md:150-158.

**Verifier (high confidence)**: All cited claims verified. Stronger than stated: _preamble.md:278 and templates.md:60 treat review=failed as a resting state with a rework-menu next-action, yet fab-continue dispatch (line 40/52) lacks a failed row, and Go CurrentStage skips failed (derives hydrate, dead-ends on "Review has not passed"). Counter-evidence: conformant fail+reset leaves apply=active, so users reach the menu after one re-review — not hard-blocked. Hence medium, not high. Fix aligns skills with existing specs; nothing breaks.


### `f020` [MEDIUM] fab-new/fab-draft persist confidence before SRAD questions, never re-score or regrade after answers

**Files**: fab-draft.md, fab-new.md

Step 7 persists/displays the score, then Step 8 asks questions (Step 9 confirms order: 'generation, type inference, confidence, questions'). Step 8 never says to update intake.md, regrade answered Unresolved rows, or re-run fab score. Since unresolved>0 forces score 0.0, the persisted/displayed confidence stays 0.0 even after the user answers everything.

**Recommendation**: Swap Steps 7 and 8 (questions before scoring), and add to Step 8: write each answer into intake.md, upgrade the row's grade (e.g., to Certain with 'Asked — {outcome}'), then score once in the final step. Mirror in both files.

**Evidence**: fab-new.md:92-106 (Step 7 before Step 8), fab-new.md:112 '(generation, type inference, confidence, questions)'; identical at fab-draft.md:94-112; no re-score instruction exists in either Step 8

**Verifier (high confidence)**: Verified: Step 7 scores before Step 8 questions (fab-new.md:92-110, fab-draft.md:94-112); no regrade/re-score instruction anywhere for intake-time answers; score.go:324 hard-zeros unresolved>0. Caveat finding missed: PostToolUse artifact-write hook (hook.go:255-268) auto-rescores on intake.md edits — but _preamble.md:478 keeps answered rows graded Unresolved, so 0.0 persists and the ff gate blocks until /fab-clarify regrades. Recovery path exists and only SPEC mirrors need updating, so medium not high.


### `f022` [MEDIUM] fab-clarify never specifies where tentative questions are actually asked

**Files**: _preamble.md, fab-clarify.md

Step 1.5 only scans and queues ('Present tentative assumption questions first'). Step 2's note assumes tentative resolution already happened ('this flow runs on the already-updated artifact'), and Step 3 handles only 'remaining non-tentative' questions. The ask-and-resolve mechanics for tentative questions exist in no step — one agent asks them in 1.5, another never asks them.

**Recommendation**: Add an explicit 'Step 1.7: Resolve Tentative Questions' (or extend Step 1.5) defining how tentative questions are presented, answered, and written back (grade upgrade, Scores, marker replacement) before the Step 2 count. Update _preamble.md:542-544, which already claims 'tentative resolution in Step 1.5', to match.

**Evidence**: fab-clarify.md:54 'Present tentative assumption questions ... first' vs :58 'already-updated artifact' vs :137 'For each remaining non-tentative question'; _preamble.md:542 'after tentative resolution in Step 1.5'

**Verifier (high confidence)**: Quotes verified (fab-clarify.md:54,58,137; _preamble.md:542-544). Git history confirms: refactor #334 moved tentatives before bulk confirm; its spec said ask them "one at a time per existing behavior" but that mechanics link was dropped, and Step 3 now excludes tentatives while Step 4 is tied to Step 3. Counter-evidence capping severity: Step 1.5 specifies recommendation framing and Step 4 mechanics are strongly implied, so agents likely improvise correctly — risk is inconsistent grade/Scores write-back, not broken flow. SPEC mirror update is normal procedure; no Go/constitution conflicts.


### `f023` [MEDIUM] Contradiction: Critical Rule scope — line 403 includes /fab-continue, line 520 says intake-time skills only

**Files**: _preamble.md, fab-continue.md

Line 403: Unresolved low-R/low-A decisions 'MUST always be asked — even in /fab-new and /fab-continue'. Line 520: 'The SRAD Critical Rule ... applies at intake-time skills only (/fab-new, /fab-clarify)' — omitting /fab-continue. An agent at fab-continue's intake stage could legitimately ask or silently assume depending on which line it follows.

**Recommendation**: Reconcile: fab-continue.md:64 establishes the intent (questions at intake stage only; apply decides-and-records). Amend line 520 to '(/fab-new, /fab-clarify, /fab-continue at intake)' or amend line 403 to drop /fab-continue — pick one; this is a behavior decision, not just wording.

**Evidence**: _preamble.md:403 'even in `/fab-new` and `/fab-continue`' vs _preamble.md:520 'applies at intake-time skills only (`/fab-new`, `/fab-clarify`)'; fab-continue.md:64 '**Intake only**: Apply SRAD ... Budget: 1-2 unresolved questions'

**Verifier (high confidence)**: Quotes verified; contradiction is real. But not a "behavior decision": fab-continue.md:64 already mandates SRAD with 1-2 unresolved questions at intake, so fix is amending _preamble.md:520 (plus the same phrasing mirrored in docs/memory/pipeline/clarify.md:130 and planning-skills.md:126, and the SPEC mirror). Severity inflated: divergence window is only fab-continue's backward-compat intake-generation path, and the skill's own explicit budget overrides the gate-explainer prose, making silent assumption unlikely.


### `f047` [MEDIUM] fab-status mandates ANSI yellow the render path strips

**Files**: fab-operator.md, fab-status.md · merged from 2 reports

fab-status requires the Impact line be 'highlighted in yellow (terminal `\e[33m...\e[0m`)' when thresholds are exceeded. fab-operator §4 states — 'empirically verified' — that ANSI SGR escapes do not survive the assistant-message→markdown render path and that emoji is the only color channel. fab-status output goes through the same path, so the MUST is unsatisfiable and may emit literal escape garbage.

**Recommendation**: Replace the ANSI requirement in fab-status.md:53 with the operator's surviving channels: prefix the Impact line with a warning emoji (e.g., 🟡/⚠️) and/or bold when over threshold, mirroring fab-operator's health-emoji convention.

**Evidence**: fab-status.md:53 "The line MUST be highlighted in yellow (terminal `\e[33m...\e[0m` or equivalent)" vs fab-operator.md:200 "ANSI SGR escapes (`\e[…m`) do NOT survive this path — they are stripped whether emitted as literal `\e[` text or as real ESC bytes (empirically verified)."

**Verifier (high confidence)**: Both quotes verified verbatim (fab-status.md:53, fab-operator.md:200). SPEC-fab-status.md:30 confirms status is agent-formatted assistant-message output — the same render path operator §4 proved strips ANSI (yellow mandate: 2026-05-08; stripping discovery: 2026-06-10, never reconciled). Only cross-reference is SPEC-fab-status.md:23 "Yellow-highlighted" — normal mirror update. Weakness: "or equivalent" makes the MUST technically satisfiable via emoji, but the only named mechanism is dead, silently losing the over-threshold bloat signal. Medium stands.


### `f052` [MEDIUM] Common Error Messages table: 3 of 4 rows don't match actual CLI error strings

**Files**: _cli-fab.md, resolve.go

`Status file not found: {path}` exists nowhere in the Go source. `Cannot resolve change '{arg}'` is actually `No change matches "{arg}".` (resolve.go:122). `No active changes found` fires when an override is given and fab/changes/ is empty (resolve.go:95); the absent-symlink path emits `No active change.` instead. Agents matching documented text for error recovery will never match.

**Recommendation**: Regenerate the table (_cli-fab.md:463-470) from the actual strings in internal/resolve/resolve.go (No change matches, Multiple changes match, No active change, No active changes found), or delete it and rely on stderr passthrough per _preamble preflight step 2.

**Evidence**: _cli-fab.md:467-470 vs resolve.go:95 'No active changes found.', :119 'Multiple changes match "%s": %s.', :122 'No change matches "%s".', :139/:159 'No active change.'; grep -rn 'Status file not found' src/go → no hits.

**Verifier (high confidence)**: Confirmed: 'Status file not found' and 'Cannot resolve change' exist nowhere in src/go (grep clean); actual strings at resolve.go:95,119,122,139,159 match the finding verbatim. Minor nuance: row 470's string does exist — its cause column is wrong, not the string. No SPEC mirror for _cli-fab exists; no other file references the table; fab-switch/fab-discuss corroborate the real strings. No constitution conflict. Medium severity appropriate.


### `f057` [MEDIUM] Codex→Claude cascade has no invocation contract for either CLI

**Files**: _review.md

The cascade says 'run Codex as the reviewer' / 'attempt Claude' but gives no command signature, no non-interactive flags, no way to pass the diff/focus-areas prompt, no output-parsing guidance, and no definition of 'fails' (non-zero exit? timeout? empty output?). Two sub-agents will invoke these tools differently. Compare git-pr-review.md, which spells out exact gh commands.

**Recommendation**: Add exact non-interactive invocation forms to the Cascade section of _review.md (command, how the diff+prompt is supplied, what constitutes failure, how output maps to the three tiers) — mirroring the precision of git-pr-review.md:87-92.

**Evidence**: _review.md:113 'if found and enabled, run Codex as the reviewer'; :115 'attempt Claude: `command -v claude` — if found and enabled, run Claude as the reviewer'; :116 'unavailable or fail' undefined. grep shows no codex invocation documented in any other skill or helper.

**Verifier (high confidence)**: Confirmed: _review.md:112-116 has only `command -v` checks; no invocation form, prompt mechanism, failure definition, or output mapping anywhere in src/ or docs/. Decisive: pre-3a85fa45 git-pr-review.md specified `codex review --base`, `claude -p` with enriched-prompt format and failure/posting semantics — lost when relocated to _review.md. Only counter-argument (LLM sub-agent can improvise) fails: bare codex/claude launch interactive UIs that hang Bash. Fix touches only _review.md + SPEC mirror; no parser/cross-ref breakage. Medium severity stands.


### `f067` [MEDIUM] Gap Analysis step is under-specified — no corpus, no method, no 'covered' definition

**Files**: fab-draft.md, fab-new.md

Step 2 is two sentences: 'Check for existing mechanisms or scope concerns covering the idea.' It names no sources (memory index? specs index? existing changes? backlog?), no depth, and no criterion for 'covered'. Two agents will check entirely different things, and one may skip it. The SRAD table relies on this step ('gap analysis before folder creation').

**Recommendation**: Specify the corpus and criterion in Step 2: scan docs/memory/index.md and docs/specs/index.md (already always-loaded) plus fab change list for an existing change/mechanism that already implements or conflicts with the described behavior; 'covered' = an existing artifact already provides the requested outcome.

**Evidence**: fab-new.md:42-44 / fab-draft.md:44-46 (entire step is two sentences); _preamble.md:409 'gap analysis before folder creation' presumes a defined procedure

**Verifier (high confidence)**: Confirmed: fab-new.md:42-44 / fab-draft.md:44-46 give one sentence, no corpus/criterion; nothing elsewhere defines it (preamble:409, glossary:117 only restate). Recommendation is additive — no cross-refs break; constitution Principle II supports it; indexes are always-loaded (preamble:43-44). Strongest counter: SPEC mirrors hint "Read/Grep: existing skills, specs, memory", but deployed skill lacks it and no 'covered' criterion exists anywhere. Caveat: keep the existing "scope concerns" clause. Medium holds — missed gaps feed unattended fab-fff.


### `f081` [MEDIUM] Direct-file mode `migrations [file]` is under-specified

**Files**: fab-setup.md

One sentence defines it (line 323). Unspecified: whether `[file]` is a path or a filename under `$(fab kit-path)/migrations/`; whether Step 1's 'current >= target: stop' applies; whether Applying-a-Migration step 5 still writes `TO` to `.kit-migration-version` (could move the version backwards or skip ahead); which output format applies. Two agents diverge on version stamping.

**Recommendation**: Add a short 'Direct File Mode' subsection under Migrations Behavior specifying: filename resolved under `$(fab kit-path)/migrations/`, version comparison and discovery skipped, whether the version stamp is written (state the intended behavior explicitly), and the single-migration output format.

**Evidence**: fab-setup.md:323 ('When `[file]` is provided, read and apply that specific migration file directly, bypassing version range discovery.') — the only sentence governing this mode; fab-setup.md:379 (step 5 'Update version: write `TO`') leaves stamping ambiguous for ad-hoc files.

**Verifier (high confidence)**: Confirmed: fab-setup.md:322 is the sole sentence for direct-file mode; step 5 (line 379) stamps TO unconditionally, leaving ad-hoc stamping ambiguous. Git history (f0a439b9) shows the mode was born one-sentence; legacy fab-update had no [file] arg, so no lost detail exists. No Go code or cross-reference breaks; constitution's idempotency principle strengthens the finding (backwards version stamp corrupts migration state). Medium severity fair given rarity vs. state-corruption consequence.


### `f082` [MEDIUM] Config menu items 5-7 edit markdown files but the Edit Section Flow is YAML-only

**Files**: fab-setup.md

Menu options 5-7 target context.md/code-quality.md/code-review.md, but the Edit Section Flow mandates string replacement to preserve YAML comments and validates 'YAML parseable, required fields present' — inapplicable to markdown. Config Output reports '{N} sections updated in fab/project/config.yaml' even for .md edits, and the Context loading note (line 172) says only config.yaml is loaded.

**Recommendation**: Add a branch to Edit Section Flow (lines 220-228): for sections 5-7, read and edit the target .md file directly, skip YAML validation, and report the actual file path in output. Update line 172 to 'loads the file being edited'.

**Evidence**: fab-setup.md:206-208 (menu items '5. context.md', '6. code-quality.md', '7. code-review.md') vs fab-setup.md:223-225 ('NOT full YAML rewrite (preserves comments)... Validate — YAML parseable, required fields present') and fab-setup.md:232 ('{N} sections updated in fab/project/config.yaml').

**Verifier (high confidence)**: All quotes verified. context.md/code-quality.md/code-review.md are standalone files (scaffold confirms; config.yaml has no such keys), yet Edit Section Flow mandates YAML string-replacement/validation and reports config.yaml updates. Strongest refutation — agent would self-correct — fails because instructions are contradictory, not merely incomplete. Recommendation is additive; no cross-references break (only SPEC-fab-setup.md mirror, normal procedure). Severity medium stands.


### `f085` [MEDIUM] fab-archive's documented 'already archived' soft-skip is unreachable through its own flow

**Files**: fab-archive.md

Archive mode runs `fab preflight` first, and change resolution excludes archive/. After a successful archive, re-running /fab-archive fails preflight ('No change matches' or missing symlink) and STOPs per preamble §2 step 2 — it never reaches `fab change archive`'s exit-0 'already archived' path (which only fires when source AND destination folders both exist). The 'Idempotent? Yes — re-archive is a soft skip' claim rests on this dead path; actual re-runs surface a raw error.

**Recommendation**: Add an archive-mode error-handling row (the mode currently has no error table, unlike restore): on preflight no-match, run `fab change archive-list`, and if the name matches an archived entry report 'already archived' as a soft skip instead of the raw error.

**Evidence**: fab-archive.md:36 (preflight first), :65 ("If the command prints `already archived: ...` and exits 0"), :108 ("re-archive is a soft skip"); _preamble.md:60 (resolution "excluding `archive/`"); src/go/fab/internal/archive/archive.go:87-89 (ErrAlreadyArchived only when destPath already exists while source still resolves)

**Verifier (high confidence)**: All cited lines verified. Preflight (fab-archive.md:36) uses resolve.ToFolder which excludes archive/ (resolve.go:145,187), so post-archive re-runs fail with 'No change matches' (resolve.go:122) and STOP per _preamble.md:55 — never reaching the exit-0 'already archived' path (archive.go:65,88-89; cmd/fab/archive.go:29-31), which needs source+dest both present. Recommendation is additive, mirrors restore's error table, archive-list already in SPEC allowed tools; supports Constitution III. Only counterpoint: path reachable in duplicate-copy edge case, which the finding concedes.


### `f088` [MEDIUM] fab-switch no-argument flow specifies two different listing mechanisms

**Files**: fab-switch.md

Behavior says 'Scan fab/changes/ (exclude archive/)' for the no-arg list, while the Output section says the skill 'reads fab change list output (format name:display_stage:display_state:score)'. One agent runs ls and shows bare folder names; another runs the CLI and shows stage+confidence. Divergent user-visible behavior.

**Recommendation**: Replace No Argument Flow steps 1-3 with: run `fab change list`; empty output → no-changes message; otherwise number the rows (parsing the 4-field format) and wait for selection. Delete the 'Scan fab/changes/' instruction.

**Evidence**: fab-switch.md:30: "Scan `fab/changes/` (exclude `archive/`)" vs :97: "the skill reads `fab change list` output (format `name:display_stage:display_state:score` ...) and displays confidence info alongside stage info"

**Verifier (high confidence)**: Both cited lines verified verbatim (fab-switch.md:31 vs :97). SPEC-fab-switch.md already shows the no-arg flow as `fab change list`, so the Behavior section's "Scan fab/changes/" is the outlier and the fix aligns skill with its own SPEC — no SPEC churn needed. Go ListWithOptions confirms archive exclusion, 4-field format, "fab/changes/ not found." error, and empty output for no changes, so no behavior is lost. No cross-references break. Medium severity stands.


### `f100` [MEDIUM] git-branch rename path can hijack another change's unpushed branch

**Files**: git-branch.md

On any non-main branch without upstream, the skill renames the current branch (lines 119-126). If the user sits on a different change's local-only branch (e.g., after /fab-switch), running /git-branch silently renames that change's branch away — destroying the branch-name/change association the kit relies on. Only main/master are guarded. The behavior itself is the hazard.

**Recommendation**: Add a guard to Step 4: rename only when the current branch name does not match any other change folder under fab/changes/ (e.g., via `fab change resolve <current-branch>` failing); otherwise create a new branch. Explicit behavior change.

**Evidence**: git-branch.md:122-126 ('No upstream (local-only branch) — rename the current branch: git branch -m "{branch_name}"') — no check that the current branch belongs to another change

**Verifier (high confidence)**: Confirmed at git-branch.md:116-126; rename guarded only by upstream check. Design intent (change 260226-3g6f) targeted disposable wt-create branches and assumed users wouldn't invoke on a branch they want kept — but fab-proceed auto-chains /fab-switch → /git-branch, so no user judgment applies. Mitigants: git branch -m loses no commits (recoverable); worktree-per-change flow avoids it. Guard preserves wt rename (random names don't resolve); only SPEC/memory mirrors need updating. Caveat: recommended checkout -b fallback still inherits old change's HEAD.


### `f105` [MEDIUM] reorg-memory: move-section excluded from Link Impact requirement

**Files**: docs-reorg-memory.md

Line 102 requires Link Impact only for 'split-domain / merge-domain / flatten / move' rows. But a move-section migration relocating a ##/### block across folders re-bases the block's relative links and breaks anchors pointing at the moved heading. The Step 5.5 dangling-link guard catches this only post-apply — the user approves without seeing this blast radius.

**Recommendation**: Extend the Link Impact rule at line 102 to cover move-section rows whose source and target files live in different folders (links inside the moved block, plus links/anchors targeting the moved heading).

**Evidence**: docs-reorg-memory.md:102 'For any `split-domain` / `merge-domain` / `flatten` / `move` row (any move-bearing migration)...' — move-section (defined :100 as 'relocate a `##`/`###` block between files') is omitted; guard at :127 only fires after apply

**Verifier (high confidence)**: Verified verbatim: :102 excludes move-section while claiming "any move-bearing migration"; SPEC mirror :19 mirrors the gap. Compounding: apply step :124 rewrites only Link-Impact-listed links, and stale .md#anchors (which exist, distribution/setup.md:62-63) can evade the :127 guard since the file target still resolves. Git history (#379) shows no deliberate exclusion; fix is additive, no Go/skill/constitution cross-refs break. Strongest counter — hard-block guard recovers path breaks — doesn't cover anchors or pre-approval transparency.


### `f115` [MEDIUM] 'One operator per server' enforcement claim is overstated — the singleton window check is session-scoped

**Files**: fab-operator.md, operator.go

§8 claims the server-wide singleton is 'already enforced by the operator window'. The launcher runs `tmux select-window -t operator`, which resolves window names within the current session only — a second session on the same server can launch a second operator. Both would write the same server-keyed state file (tick races, double auto-answers).

**Recommendation**: Fix the launcher to search all sessions (`tmux list-windows -a -F ...`) before creating — a Go behavior change, stated explicitly — or soften §8 to say enforcement is per-session and the server-singleton is a user convention.

**Evidence**: fab-operator.md:616 'This matches the server-wide singleton already enforced by the `operator` window'; operator.go:39 `exec.Command("tmux", "select-window", "-t", tabName)` — bare window-name targets resolve in the current session.

**Verifier (high confidence)**: Empirically confirmed (tmux 3.6a): select-window -t operator from session s2 fails while s1 holds the window, so a second operator spawns and shares the socket-keyed state file (operator.go StatePath). Skill explicitly supports multi-session servers, so the scenario is in-scope. SPEC-fab-operator.md:146 repeats the false premise — needs the same edit (normal mirror procedure). Only caveat: a proper Go fix needs switch-client for cross-session jump. Severity medium stands.


### `f119` [MEDIUM] Autopilot confidence gate 'flag and wait' has no exit path and duplicates /fab-fff's own gate

**Files**: fab-operator.md

Autopilot step 3 says 'check confidence score. If below threshold, flag and wait' — it never says how (run `fab score --check-gate --stage intake <id>` in the target repo?), what ends the wait (only /fab-clarify changes the score), or whether the queue skips/halts. The Failures list omits this case. Meanwhile /fab-fff already gates internally, so the queue can stall silently.

**Recommendation**: Specify the command and CWD, and define the outcome: e.g., 'gate via `fab score --check-gate --stage intake <id>`; on failure, skip the change, report it (like review-exhausted), and continue the queue' — and add the case to the §6 Failures list.

**Evidence**: fab-operator.md:509 '3. **Gate** — check confidence score. If below threshold, flag and wait'; :553 Failures list covers review-exhausted/rebase/cherry-pick/pane-death/timeouts but not gate failure.

**Verifier (high confidence)**: Quotes verified verbatim (fab-operator.md:509, :553). SPEC-fab-operator.md:110 claims the failure matrix covers "confidence below gate" but the skill omits it — real skill/spec drift; the fix restores conformance. Strongest counters: _cli-fab.md:85-86 documents the gate command, stage-timeout (>30m) flags eventually, and "skip <change>" interrupts exist — but none define the wait's exit, and stacked chaining blocks dependents. No cross-references break; recommendation matches existing skip semantics. Medium severity stands.


### `g1-2` [MEDIUM] fab-new and fab-draft violate Constitution Principle III on re-run, with no declared contract or recovery path

**Files**: change.go, constitution.md, fab-draft.md, fab-new.md

Constitution III: 'Running a skill twice with the same inputs SHALL produce the same result.' Re-running /fab-new with natural language creates a second change folder (fresh random ID, change.go:51); re-running with a backlog ID hard-errors 'Change ID already in use' (change.go:45-47). Neither skill documents re-run behavior or how to resume an interrupted creation (folder made, intake not generated). Behavior itself is the problem here, not just wording.

**Recommendation**: In fab-new.md/fab-draft.md Step 3, detect an existing non-archived change for the detected backlog/Linear ID and route to resume (/fab-switch then /fab-continue, whose intake-active row regenerates a missing intake, fab-continue.md:50) instead of erroring. Map the `fab change new` collision failure row (fab-new.md:221, fab-draft.md:151) to that recovery guidance, and state the NL re-run semantics (new change each run) explicitly.

**Evidence**: constitution.md:12 'All skills MUST be safe to re-run without side effects'; fab-new.md:48-53 unconditional `fab change new`; change.go:47 'return "", fmt.Errorf("Change ID '%s' already in use (%s)", ...)'; fab-new.md:221 '`fab change new` failure | Surface stderr output to user and stop' — no resume pointer.

**Verifier (high confidence)**: All citations verify verbatim (constitution.md:12, change.go:45-51, fab-new.md:48-53/221, fab-draft.md:151, fab-continue.md:50). Strongest counters: Principle III's no-corruption clause is honored (collision error is the safeguard and names the existing folder), and fab-proceed already routes existing intakes to /fab-switch. But every sibling skill declares "Idempotent? Yes"; fab-new/fab-draft uniquely don't, and the fix is additive, breaking no cross-references. Severity inflated: deterministic, data-safe, recoverable — medium, not high.


### `g1-3` [MEDIUM] Hydrate re-run can duplicate memory Changelog/Design Decision entries; idempotency claim omits hydrate

**Files**: docs-hydrate-memory.md, fab-continue.md

Hydrate Behavior (fab-continue.md:174) instructs updating Requirements/Design Decisions/Changelog with no merge-without-duplication rule. If hydrate is interrupted after memory writes but before `fab status finish` (line 175), re-run (directly or via fab-ff/fab-fff Step 3) repeats the writes — duplicate Changelog rows/entries. Principle III forbids data corruption on repeat. Notably the skill's own idempotency claim (line 209) covers only 'planning regenerates, apply resumes, review re-validates' — hydrate is unaddressed.

**Recommendation**: Add to fab-continue.md Hydrate step 4: before appending, check target memory files for an existing entry referencing this change (by change name) and update in place — the same 'replaced in place (not duplicated)' contract _review.md:75 uses for ## Deletion Candidates. Extend the line-209 Key Properties claim to cover hydrate. Contrast: docs-hydrate-memory.md:19/176 already mandates merge-without-duplication.

**Evidence**: fab-continue.md:174 'update existing (Requirements, Design Decisions, Changelog, ...)' with no dedup instruction; fab-continue.md:209 'Idempotent? | Yes — planning regenerates, apply resumes, review re-validates' (hydrate absent); docs-hydrate-memory.md:19 'content is merged without duplication'.

**Verifier (high confidence)**: All cited lines verified verbatim. Re-run path confirmed: fab-ff:99/fab-fff:91 skip hydrate only when done, so an interrupt between step 4 writes and step 5 finish re-executes writes. Constitution III MUST forbids data corruption on re-run; dedup contract already exists in docs-hydrate-memory.md:19/176 and _review.md:75, so fix is consistent and purely additive — no cross-references break. Mitigation (step 2 loads target memory files) is probabilistic only; medium severity stands.


### `g2-4` [MEDIUM] fab-operator's per-tick fab commands have no failure behavior, unlike sibling operations in the same file

**Files**: fab-operator.md

Tick step 1 runs `fab operator tick-start` and `fab pane map --all-sessions --json` every 3 minutes for the operator's lifetime with no defined non-zero-exit behavior — yet the same skill defines exit-2/exit-3 handling for `fab pane window-name` (lines 174, 186) and a 3-strikes disable for watch query failures (line 586). The operator is long-lived; mid-session fab breakage (upgrade, PATH change after /clear) is plausible.

**Recommendation**: Add a tick-failure rule to §4 Tick Behavior mirroring the watch pattern: on `fab operator tick-start` or `fab pane map` non-zero exit, log one italic line and skip the tick; after 3 consecutive failures stop the loop and alert the user. Cover `fab operator time` (line 296) under the same rule.

**Evidence**: fab-operator.md:198 'Snapshot — run `fab operator tick-start` ... Then run `fab pane map --all-sessions --json`' (no failure branch); contrast line 174 'A non-zero exit — pane vanished ... (exit 2) or any other tmux error (exit 3 ...) — causes the operator to log one line and continue' and line 586 'On failure: set `last_error`, skip this watch for this tick. After 3 consecutive failures: disable the watch, alert user.'

**Verifier (high confidence)**: All cited lines verified verbatim (198 no failure branch; 174/186 exit-2/3 handling; 586 3-strikes; 296 uncovered). operator_tick_start.go confirms non-zero exits are real (state read/parse/write errors). Line 94 explicitly excludes ticks from fallback handling, proving the gap. Recommendation is purely additive; only the SPEC mirror needs the standard update. Strongest counter ("LLM reacts to errors anyway") fails: the file specifies failure behavior for cosmetic operations but not the core snapshot. Medium severity stands — skill markdown is the implementation per constitution.


### `g3-2` [MEDIUM] Plan template contradicts itself and the skills on which SRAD grades plan.md ## Assumptions may contain

**Files**: _generation.md, _preamble.md, plan.md

plan.md:214 says 'All four SRAD grades may appear', yet its own placeholder row (:219) and summary line (:221) allow only Certain/Confident/Tentative, matching the doctrine in plan.md:38-39, _generation.md:74-76, fab-continue.md:64. Meanwhile _preamble.md:477 mandates 'Include all four grades' for every planning artifact. An apply agent gets contradictory instructions for genuinely unresolvable points.

**Recommendation**: Fix plan.md:214 to 'Three grades only (Certain/Confident/Tentative) — Unresolved is intake-only', and scope _preamble.md:477's all-four-grades rule (Assumptions Summary Block) to intake artifacts, stating plan.md ## Assumptions excludes Unresolved.

**Evidence**: plan.md:214 'All four SRAD grades may appear' vs plan.md:219 '{Certain|Confident|Tentative}' and :221 summary without {U}; _generation.md:75 'graded SRAD assumption (Certain/Confident/Tentative)'; _preamble.md:477 'Include all four grades (Certain, Confident, Tentative, Unresolved)'.

**Verifier (high confidence)**: All cited lines verified verbatim; contradiction born in one commit (62b2608a). Doctrine weight (plan.md:38/219/221, _generation.md:75, fab-continue.md:64, _preamble.md:520 "Critical Rule applies at intake-time skills only") confirms three-grades-at-apply is intended; plan.md:214 is the outlier. No breakage: Go scorer reads intake.md only; templates.md mirror omits plan ## Assumptions. Caveat: _preamble:477 fix must keep four grades for fab-ff cumulative summaries covering intake artifacts (:479).


### `f028` [LOW] Frontmatter freeze contradicts reference-instead-of-reexplain under helper-declaration semantics

**Files**: _preamble.md, internal-skill-optimize.md

Rule 4 says replace re-explanations with a reference to _generation.md; rule 2 freezes frontmatter. Per _preamble, a skill only loads helpers declared in frontmatter helpers:. Replacing inline content with a _generation reference without adding helpers: [_generation] means that content is never loaded at runtime — silently removing functionality, violating rule 1.

**Recommendation**: Amend rule 2 to require updating the helpers: list whenever a helper reference is introduced; restrict rule 4's bare 'see X' references to always-loaded _preamble content. Also clarify whether rule 2 freezes all frontmatter or only name/description.

**Evidence**: internal-skill-optimize.md:46 ('Preserve frontmatter exactly'), :48 ('replace ... with "Apply the SRAD framework (see _preamble.md)"') vs _preamble.md:109 'Skills that declare no helpers: list (or an empty list) load only _preamble.'

**Verifier (high confidence)**: Quotes verified, but rule 4's example cites _preamble (always loaded); the _generation pathway is only the Analysis table (line 33), whose example concepts all live in _preamble. All five skills referencing _generation already declare helpers:[_generation]; constraint line 86 limits referencing to _preamble.md; rule 1 plus the human-approval gate guard against silent loss. Hazard is latent with no current instance. Fix is a cheap one-file clarification; no SPEC mirror exists for internal-* skills. High severity is inflated.


### `f045` [LOW] State Table missing ship/review-pr derivations and mid-stage states

**Files**: _preamble.md, fab-continue.md · merged from 2 reports

State derivation (lines 284-290) covers (none) through hydrate but not ship or review-pr (pass/fail), which appear as table rows. Semantics also flip mid-table: intake/apply mean stage-active, review/hydrate mean stage-done — so a skill ending while review or hydrate is active matches no row. 'review (fail) → (rework menu)' is only defined in fab-continue.md:150, unreachable from other skills.

**Recommendation**: Add derivation rows for ship (`progress.ship == done`) and review-pr pass/fail; state explicitly that active mid-stages (review/hydrate/ship active) map to their own stage's continue command; replace '(rework menu)' with 'see /fab-continue Verdict-Fail' or list the three rework commands inline.

**Evidence**: _preamble.md:284-290 derives only (none)/initialized/intake/apply/review/hydrate; table rows :280-282 include ship and review-pr; :278 '*(rework menu)*' with default '—'

**Verifier (high confidence)**: Facts confirmed at _preamble.md:269-290: ship/review-pr rows lack derivations; active-vs-done semantics do flip (review-active after fab-continue.md:124 matches no row). But "unreachable" is overstated: fab-ff.md:92 and fab-fff.md:84 explicitly point to /fab-continue, and fab-status/fab-switch use Go routing (change.go:427-440) covering all stages — no skill misroutes today. Fix is additive, breaks nothing (SPEC-preamble mirrors table only at summary level), aligns with constitution's six-stage pipeline. Worth doing as cheap completeness fix; severity low, not medium.


### `f058` [LOW] Parsimony 100-line threshold has no defined operational effect

**Files**: _review.md

Step 7 declares a 'Threshold for stricter scrutiny: 100 net added lines (advisory, hard-coded)' then says below threshold 'the pass still runs and MAY emit findings'. Nothing states what changes above the threshold — 'stricter scrutiny' is undefined, so the threshold is a no-op an agent cannot act on deterministically.

**Recommendation**: Behavior decision: either define the above-threshold behavior (e.g., mandatory sweep of all four categories vs. opportunistic below) in step 7, or delete the threshold sentence entirely. Note the scaffold code-review.md:49 comment also advertises this threshold, so update both together.

**Evidence**: _review.md:53: 'Threshold for stricter scrutiny: **100 net added lines** (advisory, hard-coded — not project-configurable). Below threshold the pass still runs and MAY emit findings.' No clause defines above-threshold behavior.

**Verifier (high confidence)**: Claim accurate for _review.md:53, but above-threshold behavior IS defined elsewhere: ogf2 spec ("above the threshold, the agent SHOULD scrutinize more aggressively... advisory — NOT a gate") and SPEC-_review.md:89 already state it. _review.md just dropped that clause. Fix = restore the one clause (define branch only); deleting the sentence would violate the ogf2 requirement and break scaffold code-review.md:49 + SPEC mirror. Deliberately non-deterministic by design, so severity is low, not medium.


### `f065` [LOW] Bulk-confirm response parsing has no rule for unnumbered natural replies

**Files**: fab-clarify.md

The format table recognizes only numbered items, ranges, and the exact 'all ✓/ok/yes'. A natural reply like 'looks good' or 'all good except 3' matches nothing; combined with 'Items not mentioned remain Confident', a literal agent silently confirms zero items while the user believes they confirmed everything. A liberal agent confirms all. Divergent outcomes.

**Recommendation**: Add one fallback row to the Response Parsing table: any unnumbered global affirmative → treat as 'all ✓' after echoing the interpretation; anything else unparseable → re-prompt once with the expected formats.

**Evidence**: fab-clarify.md:84-95 format table (only '{#}. ...', '{start}-{end}. ...', 'all ✓|ok|yes'); :95 'Items not mentioned remain Confident (unchanged)'

**Verifier (high confidence)**: Table at fab-clarify.md:84-95 verified: no fallback for unnumbered replies. But "silent" is overstated — Step 6 summary (Resolved: 0) and Step 7 mandatory re-score expose a no-op; failure mode is fail-safe (grades unchanged, skill idempotent) and line 74 already instructs the response format. No cross-references break (_preamble.md:538-544 doesn't enumerate formats); only the routine SPEC mirror update is needed. One-row fix is cheap and additive. Severity low, not medium.


### `f069` [LOW] "finish intake first if still active" misses the common `ready` state

**Files**: fab-ff.md, fab-fff.md

/fab-new leaves intake at `ready` (its Step advances intake to ready), so the normal fab-new → fab-ff/fff path has intake `ready`, not `active`. A literal agent matching the formal state name skips the finish, leaving apply `pending`; the subsequent finish-apply then errors. `finish` accepts both active and ready, so only the conditional wording is wrong.

**Recommendation**: In fab-ff.md:52 and fab-fff.md:52, change 'finish intake first if still active' to 'if progress.intake is not done, finish intake' — matching the formal state vocabulary used everywhere else in these files.

**Evidence**: fab-ff.md:52 / fab-fff.md:52 'finish intake first if still active: `fab status finish <change> intake fab-ff`'; fab-new.md:110-113 'advance intake to `ready`: fab status advance {name} intake'; status.go:40 finish From:[active, ready].

**Verifier (high confidence)**: All evidence verified: fab-ff/fff.md:52 say "if still active"; fab-new.md Step 9 leaves intake `ready`; status.go:40 finish From [active,ready] with auto-activate apply. Literal reading skips finish, apply stays pending, later finish-apply errors. Fix is safe: phrase unique to these files; SPEC-fab-ff.md:22 lists the command unconditionally, so fix aligns skill with mirror. Severity inflated: finish succeeds from ready, command shown inline, colloquial reading works, failure self-recoverable — low, not medium.


### `f086` [LOW] fab-archive hard-codes Next state 'initialized' even when the archived change wasn't active

**Files**: fab-archive.md

The archive Output template ends 'Next: {per state table — initialized}', but the same template allows 'Pointer: — skipped, not active' (archiving a non-active change by name). In that case another change is still active, so per _preamble state derivation the state is NOT initialized and the suggested commands are wrong.

**Recommendation**: Make the Next line conditional in the Output section: pointer 'cleared' → initialized state; pointer 'skipped' → derive from the still-active change's stage (or instruct: 'run the State Table lookup against the current active change').

**Evidence**: fab-archive.md:94 ("Pointer:  ✓ .fab-status.yaml removed             (or: — skipped, not active)") and :98 ("Next: {per state table — initialized}"); _preamble.md:286: initialized requires "no active change (`.fab-status.yaml` symlink is absent)"

**Verifier (high confidence)**: Confirmed: fab-archive.md:94/:98 as quoted; archive.go:103-108 emits pointer:skipped whenever the target isn't active, including when another change IS active — then state isn't initialized (_preamble.md:286) and /fab-new is wrong. Fix matches house style (restore's conditional Next at :185); no spec/Go cross-refs break. Downgrade to low: _preamble.md:265 already mandates deriving the state reached, skipped-with-no-active-change still yields initialized correctly, and it's only a footer suggestion.


### `f089` [LOW] fab-switch promises a corruption warning nothing produces

**Files**: fab-switch.md

Error Handling says: missing .status.yaml → 'Switch anyway, warn: Warning: .status.yaml not found — change may be corrupted.' But Behavior fully delegates ('displays the command's stdout directly'), and the Go Switch silently defaults to stage 'unknown' with no warning. Context Loading's 'Loads matched change's .status.yaml' is the only hook, contradicting the single-Bash-call delegation. Ambiguous who detects and warns.

**Recommendation**: Either delete the Error Handling row and the .status.yaml claim at line 23 (accept the binary's 'Stage: unknown' display), or specify concretely: 'if stdout shows `Stage: unknown`, append the corruption warning' — no extra file read needed.

**Evidence**: fab-switch.md:108 (warning text) vs :37 ("Delegate to `fab change switch` via a single Bash call") and :60 ("displays the command's stdout directly"); src/go/fab/internal/change/change.go:170-206 — `Switch` guards `sf.Load` with `err == nil` and emits no warning on failure

**Verifier (high confidence)**: Confirmed: fab-switch.md:108 promises a warning no actor produces — Go Switch (change.go:169-206) silently defaults to "unknown" via `err == nil` guard, and lines 37/60 mandate displaying stdout directly. resolveOverride matches folder names only, so the edge case is reachable. No cross-references break (warning text unique; SPEC mirror lacks it). Prefer recommendation option 2 (key warning off "Stage: unknown" in stdout). Severity inflated: rare edge case, cosmetic warning only — low.


### `f093` [LOW] git-pr-review: no defined branch for zero fetched comments (common Copilot 'no issues' review)

**Files**: git-pr-review.md

Copilot frequently submits a review with zero inline comments, which passes the Phase 2 poll (review count > 0). Step 3 then fetches zero non-reply comments, and Step 4's only exit ('If all comments are informational') is vacuous on an empty set — one agent prints '0 comments triaged' and proceeds, another STOPs. The stage outcome differs.

**Recommendation**: Add to Step 3: 'If zero non-reply comments are fetched, print `No actionable comments.` and go to Step 6 success path.'

**Evidence**: git-pr-review.md:105-108 (fetch, skip replies), :118 ('For each fetched comment'), :134 ('If all comments are informational → ... STOP') — no empty-set rule

**Verifier (high confidence)**: Gap confirmed: Phase 2 poll (line 94) checks review count only, unlike Phase 1's inline-comment guard (lines 61-66); no empty-set rule in Steps 3-4; line 134 is vacuous on zero comments. But "stage outcome differs" is wrong: both interpretations converge at Step 6, which explicitly lists "no actionable comments" as success→finish, and Step 5 says "do NOT stop here". Divergence is messages/phase metrics only. One-line fix, no cross-refs broken; SPEC mirror update is routine. Severity: low.


### `f097` [LOW] git-pr type inference uses raw substring matching — 'prefix'/'fixture' trigger type fix

**Files**: git-pr.md

Step 0b.3 matches case-insensitive substrings over the whole intake content: 'fix' matches prefix/suffix/fixture, 'rename' matches inside other words, and most intakes mention 'bug' or 'fix' incidentally — biasing inference heavily toward `fix`. The behavior itself is the problem, not the wording; this fallback only fires when change_type is null, limiting blast radius.

**Recommendation**: Specify whole-word matching restricted to the intake title and `## Why` section (e.g., 'match as standalone words, not substrings'), keeping the same keyword lists and precedence.

**Evidence**: git-pr.md:39-41 ('read the intake content and pattern-match (case-insensitive)... Contains any of: "fix", "bug", "broken", "regression" → type = fix')

**Verifier (high confidence)**: Quote accurate (git-pr.md:39-42). Severity inflated: fallback fires only when change_type is null — fab-new/fab-draft persist it at creation, so 0b.3 is near-dead code; result isn't persisted, affects only PR title prefix; LLM executor blunts mechanical 'prefix'→fix matches, though incidental-'fix' bias is real. Recommendation is scoped too narrowly: the identical heuristic in fab-new.md:77-85 / fab-draft.md:81-86 (primary, persisted inference) and docs/specs/change-types.md:72-84 should change together. No constitution conflict.


### `f101` [LOW] hydrate-memory ingest Step 1 directory branch contradicts mode routing

**Files**: docs-hydrate-memory.md

The classification table routes any existing directory to Generate mode (line 43), yet Ingest Step 1 says 'If directory, recursively read all .md files' (line 57) — unreachable per the table. An agent given a folder of markdown notes could ingest it (Step 1) or gap-scan it (table); the two interpretations produce entirely different output.

**Recommendation**: Delete 'If directory, recursively read all `.md` files.' from Step 1 (classification table stays authoritative). If folders of notes SHOULD be ingestable, that is a behavior change — add an explicit classification rule instead (e.g., directory containing only .md files → Ingest).

**Evidence**: docs-hydrate-memory.md:43 '| Folder | Resolves to existing directory | **Generate** |' vs :57 '**Local path**: Read directly. If directory, recursively read all `.md` files.'

**Verifier (high confidence)**: Confirmed: lines 43 vs 57 contradict. Git history shows the sentence is leftover from the pre-generate-mode skill (afb705cf supported directory ingest; d2f0b895 added the table routing folders to Generate but kept the stale sentence). Memory docs (hydrate.md:29, hydrate-generate.md:123) and SPEC mirror document folders→Generate, so deletion loses no intended behavior; no cross-references. Severity downgraded: the routing table gates mode before Step 1, so real misrouting risk is low — dead-text cleanup.


### `f102` [LOW] hydrate-specs: Step 3 excludes implementation details but Step 4 ranks them

**Files**: docs-hydrate-specs.md

Step 3 says 'Exclude topics that are purely implementation detail' (line 51); Step 4's impact scale defines 'Low: implementation details' (line 55). If they are excluded as gaps, the Low tier is unreachable. One agent filters them out entirely, another surfaces them as Low-ranked gaps that can consume a top-3 slot.

**Recommendation**: Make Step 4's Low tier mean 'supporting/peripheral concepts' (or similar) and keep Step 3's exclusion; or drop the Step 3 exclusion and let Low-ranked gaps fall below the top-3 cap naturally. State which behavior is intended.

**Evidence**: docs-hydrate-specs.md:51 'Exclude topics that are purely implementation detail.' vs :55 'Rank by impact (High: core behavioral rules... Low: implementation details).'

**Verifier (high confidence)**: Quotes at lines 51/55 verified exact. Counterpoint: "purely" makes Low technically reachable for mixed topics, so "unreachable" is overstated — but the ambiguity is real, and docs/specs/skills.md:686 keeps the ranking while dropping the exclusion, proving drift. Fix is a one-line wording change; only mirror updates needed (normal procedure); no code parses the tiers. Severity inflated: interactive per-gap confirmation caps harm at one wasted top-3 slot.


### `f103` [LOW] reorg-memory claims reserved domains are depth-exempt; fab memory-index only exempts width

**Files**: docs-reorg-memory.md, memoryindex.go

docs-reorg-memory:29 exempts _shared/_unsorted from 'the width/depth bounds' and forbids flattening them. The CLI exempts reserved domains from width warnings only — depthWarnings runs unconditionally per domain (memoryindex.go ~line 243/260). So fab memory-index can warn 'exceeds depth — consider flattening' on _shared while the skill refuses to act; user gets contradictory advice. hydrate-memory:79 correctly says width-only.

**Recommendation**: Align docs-reorg-memory:29 to 'exempt from the width bound' (matching hydrate-memory:79 and the CLI), and decide whether 'never flatten' should survive a CLI depth warning — if reserved domains really are fully exempt, the Go depth check needs the reserved guard instead.

**Evidence**: docs-reorg-memory.md:29 'exempt from the width/depth bounds. Never propose splitting, merging, or flattening them' vs memoryindex.go:47 'reservedDomains are exempt from the width warning' and unconditional `depthWarnings(memRoot, domainDir)` append

**Verifier (high confidence)**: Confirmed: docs-reorg-memory.md:29 claims width/depth exemption; memoryindex.go guards only width (line 244), depthWarnings appended unconditionally (line 260). Every other source (hydrate-memory:79, fab-continue:174, _cli-fab:338, templates.md spec, cmd memory_index.go:28, kit-architecture memory) says width-only — and index recursion is one level, so depth>3 files in _shared are unindexed, vindicating the unconditional Go check. Fix the skill (also lines 56/84) plus SPEC mirror. Severity low: rare trigger, advisory-only warnings.


### `f110` [LOW] skill-optimize writes skill files without the constitution-mandated SPEC update

**Files**: constitution.md, internal-skill-optimize.md

Execution ends at 'write the optimized file(s)'. The constitution requires every change to src/kit/skills/*.md to update the corresponding docs/specs/skills/SPEC-*.md. Since this skill loads neither the constitution nor the preamble, the agent cannot know the rule — every approved run mechanically violates a MUST.

**Recommendation**: Add a post-write step to both execution modes: 'update docs/specs/skills/SPEC-{skill}.md to reflect structural changes' — or explicitly document an exemption for content-neutral condensation in the constitution.

**Evidence**: internal-skill-optimize.md:64 ('On approval, write the optimized file') and :78 vs constitution.md:32 'Changes to skill files (src/kit/skills/*.md) MUST update the corresponding docs/specs/skills/SPEC-*.md file'.

**Verifier (high confidence)**: Cited lines verified (skill 63-64/78, constitution:32); execution lacks a SPEC step. But the premise is partly false: Pre-flight step 1 loads _preamble.md, whose Always Load section mandates reading constitution.md — the rule is discoverable. Condensation is behavior-neutral and SPECs mirror behavior, so drift risk is low; internal-* skills already lack SPEC mirrors (de facto carve-out). A one-line post-write step breaks no cross-references. Real but overstated.


### `f111` [LOW] skill-optimize compares working-tree sources against possibly stale deployed helpers

**Files**: internal-skill-optimize.md

Pre-flight reads _preamble/_generation from .claude/skills/ — deployed from the installed kit-cache version — while optimization targets are working-tree src/kit/skills/*.md. When the working tree is ahead of the released kit, 'already defined in _preamble' judgments use the wrong baseline, cutting or keeping the wrong lines.

**Recommendation**: Point Pre-flight steps 1-2 at src/kit/skills/_preamble.md and src/kit/skills/_generation.md — the same tree being edited.

**Evidence**: internal-skill-optimize.md:21-22 ('Read the `_preamble` skill (deployed to `.claude/skills/`)') vs :14 ('Resolves to `src/kit/skills/{skill-name}.md`').

**Verifier (high confidence)**: Confirmed: lines 21-22 read _preamble/_generation from .claude/skills/ while lines 14-15 target src/kit/skills/. sync.go deploySkills uses CachedKitDir(fabVersion) — deployed copies track the pinned cache, not the working tree. Fix aligns with constitution ("src/kit/ is canonical"); skill is fab-kit-repo-only, no SPEC mirror or cross-refs break. Downgrade to low: copies currently identical, `just install` refreshes cache, writes are user-gated, and stale baseline usually keeps (not cuts) lines.


### `f118` [LOW] No effectiveness check or clear retry bound on auto-answers — ineffective keystrokes loop forever

**Files**: fab-operator.md

Rule 3 sends `y` to Claude Code permission prompts (which are numbered menus — rule 4 would send `1`; ordering makes `y` win). If the keystroke doesn't take, the unchanged prompt re-matches every tick and is re-answered indefinitely. §3's Bounded Retries covers 'Stuck agent nudge | 1' but never says whether auto-answers count as nudges.

**Recommendation**: Add to §5 Sending Auto-Answers: after sending, re-capture next tick; if the same prompt is still displayed after N (e.g., 2) identical auto-answers, escalate. Clarify in §3's table that this bound covers auto-answers, and resolve the rule 3 vs rule 4 overlap (permission prompts are numbered menus).

**Evidence**: fab-operator.md:325-327 rules evaluated in order, '3. Claude Code permission prompt → `y`' before '4. Numbered menu'; :338 abort only 'if output changed since detection' — unchanged output re-sends; :105 'Stuck agent nudge | 1' with no mention of repeated auto-answers.

**Verifier (high confidence)**: Facts verified: rule ordering (:323-328), abort only on changed output (:338), no auto-answer bound (:105); SPEC:56 excludes input-waiting agents from stuck detection, so loop is per-spec possible. But SPEC-fab-operator.md:97 documents "No cooldown or retry limit — PR review is the safety net" as a deliberate decision, and the loop is visible (per-tick logs, 15m red/stuck marker). Fix is cheap and matches the watch 3-strike precedent, but must consciously amend that decision; drop the rule-3/4 portion (intentional precedence, unverifiable premise). Severity low.


### `f121` [LOW] fab-continue contradicts itself on intake-stage work and leaves regenerated intakes unscored

**Files**: fab-continue.md

fab-continue.md:72 states '/fab-continue operates only at apply and later, where there is no scoring', yet the dispatch table (lines 49-50) has intake rows ('generate intake if missing') and the Reset Flow (line 185) regenerates intake.md. Since the skill forbids scoring, a regenerated intake leaves stale confidence in .status.yaml, contradicting fab-new Step 7 / fab-clarify Step 7 which persist authoritative intake scores.

**Recommendation**: Fix line 72 to acknowledge the intake-generation paths, and explicitly state whether intake regeneration must run `fab score --stage intake` afterwards (it should, to keep the persisted score authoritative for /fab-status, /fab-switch, and `fab change list`). This is a behavior decision — flag, do not silently rewrite.

**Evidence**: fab-continue.md:72 'No scoring at any stage /fab-continue runs ... operates only at apply and later' vs fab-continue.md:50 '`intake` | `active` | generate intake if missing' and :185 'Intake reset regenerates the intake artifact'

**Verifier (high confidence)**: Contradiction confirmed verbatim: fab-continue.md:72 vs :50, :64 ("Intake only: Apply SRAD before generating"), :185; SPEC mirror even says fab-continue "handles intake". But severity is inflated: CheckGate (score.go:100-109) recomputes from intake.md, so the ff/fff gate is unaffected — staleness is display-only, and the line-50 path leaves 0.0/"not yet scored", not a wrong score. Only Reset Flow truly stales. Fix breaks no cross-refs; scoring after regeneration keeps intake the sole source (sync _preamble.md:527-530 + SPEC mirror).


### `f123` [LOW] fab-continue's ship/review-pr rows re-fire status events the delegated git skills already perform

**Files**: fab-continue.md, git-pr-review.md, git-pr.md

fab-continue's dispatch table instructs 'Execute /git-pr behavior → on completion run finish <change> ship fab-continue', but git-pr Step 4b already finishes ship (best-effort). Same for review-pr vs git-pr-review Step 6. The second finish hits an already-done stage and exits non-zero with no `|| true` in fab-continue, producing a spurious error and a misattributed driver.

**Recommendation**: In fab-continue.md Step 1 dispatch table, change the ship and review-pr rows to 'Execute /git-pr (/git-pr-review) behavior — it handles the stage start/finish/fail events itself; do not re-run finish/fail.'

**Evidence**: fab-continue.md:54-55 'on completion `finish <change> ship fab-continue`' / 'pass: `finish <change> review-pr fab-continue`'; git-pr.md:231-239 Step 4b runs `fab status finish <change> ship git-pr`; git-pr-review.md:183-190 Step 6 runs finish/fail for review-pr.

**Verifier (high confidence)**: Confirmed: status.go:40 only allows finish from active/ready; second finish exits non-zero ("current state is 'done'"). fab-fff.md:99/109 and SPEC-fab-continue.md:106 already use the recommended delegate-owns-events pattern. But severity inflated: failed finish writes nothing, so driver stays git-pr (no misattribution), no state corruption, workflow doesn't break. Caveat: with fab-continue's change-name override, git-pr resolves only the active change, so fab-continue's finish is the sole finisher there — fix should keep a conditional fallback.


### `g1-4` [LOW] Rework-cycle budget in fab-ff/fab-fff is conversation-local; re-entry resets the 3-cycle cap silently

**Files**: _cli-fab.md, fab-ff.md, fab-fff.md

Both skills cap autonomous rework at 3 cycles (fab-ff.md:72-95, fab-fff.md:77-87) but never persist the count: `fab status fail <change> review` is invoked without the `[rework]` argument the CLI accepts (_cli-fab.md:61), and 'Resumable — re-running picks up from the first incomplete stage' (fab-ff.md:15, fab-fff.md:15) says nothing about the counter. A crash, /clear, or plain re-run grants a fresh 3-cycle budget — the bound is unenforceable across invocations. Whether re-run-resets-budget is intended is undefined; a behavior decision is needed.

**Recommendation**: Decide and document re-entry semantics in both Resumability sections: either 'a re-run grants a fresh 3-cycle budget' (explicit), or derive prior cycles on resume by counting review `failed` events in .history.jsonl and pass the rework arg to `fab status fail` so the data exists.

**Evidence**: fab-ff.md:70 'Run `fab status fail <change> review`' (no rework arg) vs _cli-fab.md:61 '`fail <change> <stage> [driver] [rework]`'; status.go:272 `log.Review(fabRoot, statusFile.Name, "failed", rework)` — the persistence hook exists but is never fed.

**Verifier (high confidence)**: Core claim confirmed: neither Resumability section consults prior cycles, so re-runs get a fresh budget. But evidence mechanism is wrong: fail review auto-logs a failed event to .history.jsonl even without the rework arg (log.go:76-86), and stage_metrics.apply.iterations persists — data already exists, only reading it is missing. fab-ff.md:15 frames re-run as post-intervention; fab-operator.md:553 skips on review-exhausted, never re-dispatches. Worth one documentation sentence per skill (plus SPEC mirrors); drop the rework-arg suggestion. Severity: low.


### `g1-5` [LOW] Non-atomic fail→reset sequence: interruption between the two commands strands review in `failed` with no documented recovery

**Files**: fab-continue.md, fab-ff.md, fab-fff.md

All three orchestrators prescribe `fab status fail <change> review` then `reset <change> apply` as two separate commands (fab-continue.md:150, fab-ff.md:70, fab-fff.md:70). Interrupted in between: review=failed, apply=done. On re-entry, apply is skipped (done) and review re-executes, but both verdict transitions reject `failed` (finish/fail require active/ready, status.go:48-52) — a dead end. fab-continue.md:40 only `start`s a stage when `pending`, missing the failed→active transition that exists exactly for this.

**Recommendation**: Add to the resume guards (fab-continue Step 1; fab-ff/fab-fff Resumability): 'if `progress.review` is `failed`, run `fab status start <change> review` first' (the review-specific failed→active transition, status.go:48). This is Principle III's stated purpose — recovery from interruptions.

**Evidence**: status.go:48 review override '"start": {From: []string{"pending", "failed"}, To: "active"}' is never invoked by any skill for the review stage; fab-continue.md:40 'If progress is `pending`, run `fab status start ...`' — `failed` not handled.

**Verifier (high confidence)**: All cited lines verified verbatim; no skill invokes start for review, and git-pr-review.md:22 already uses the exact failed→active pattern for review-pr — strong precedent, zero breakage. Downgrade to low: window is two consecutive bash calls; re-entry's fail branch self-heals (reset apply works from done, cascade clears failed); /fab-continue apply Reset Flow is a documented escape; _cli-fab.md:57 documents start pending/failed→active. Only the pass-verdict re-entry truly dead-ends.


### `g2-2` [LOW] No generic rule for fab command failure; fab status mutations between pipeline stages have undefined failure behavior

**Files**: _preamble.md, fab-continue.md, fab-ff.md, fab-fff.md, fab-new.md

_preamble.md defines exit-code handling only for `fab preflight` (line 55) and best-effort `fab log` (57, 258). Every `fab status finish/fail/reset/advance` and `fab score` site in the orchestrators is unguarded; error tables (fab-ff.md:130-136, fab-fff.md:153-161, fab-new.md:216-227) never cover the CLI failing between stages, so an agent may proceed with .status.yaml diverged from actual progress.

**Recommendation**: Add one sentence to _preamble.md § Common fab Commands 'Key behaviors' (line 254): any fab command not explicitly marked best-effort (`2>/dev/null || true`) that exits non-zero → STOP and surface stderr; resumability handles re-runs. One generic rule covers fab-ff.md:52/60/68/70, fab-fff.md:52/60/68/70, fab-continue.md:76-82/124/148, fab-new.md:89/96/113 without per-skill rows.

**Evidence**: fab-ff.md:60 'On success: run `fab status finish <change> apply fab-ff`' — no failure branch; fab-ff.md error table (130-136) lists only preflight/intake/gate/task/review conditions. fab-new.md:113 `fab status advance {name} intake` unguarded; its table (216-227) covers only `fab change new`/`fab change switch` failures. Contrast git-pr-review.md:186 explicit `2>/dev/null || true`.

**Verifier (high confidence)**: All cited lines verified accurate; no generic rule exists in _preamble.md or _cli-fab.md. But the recommendation's wording is overbroad: fab-proceed.md:38, fab-discuss.md:40, git-pr.md:182, fab-archive.md:155 intentionally branch on non-zero fab exits without `|| true` — the rule must defer to explicit per-skill handling. Severity inflated: statusman failures are rare, cascade loudly (failed finish blocks next stage), and resumability self-heals; downgrade to low.



## 2. Context budget (8)

### `f003` [HIGH] SRAD framework taxes every skill but serves only 6 planning skills

**Files**: _generation.md, _preamble.md, fab-clarify.md · merged from 2 reports

SRAD Autonomy Framework + Worked Examples + Artifact Markers + Assumptions Summary (lines 371-481) is consumed only by fab-new, fab-draft, fab-continue, fab-ff, fab-fff, fab-clarify. The other ~16 skills (fab-status, fab-archive, git-pr, docs-*, etc.) load it on every invocation for nothing.

**Recommendation**: Extract lines 371-481 into a new `_srad` helper; add `_srad` to the Allowed values (line 103); declare it in the six planning skills' `helpers:` (fab-clarify currently declares none). Leave a 3-line pointer in _preamble. Worked Examples 1-3 (lines 415-437) can additionally be compressed to the Example-2/3 one-liner style.

**Evidence**: _preamble.md:371-481; `grep -ln SRAD *.md` hits only fab-new, fab-clarify, fab-ff, fab-draft, fab-fff, fab-continue (+ helpers); fab-clarify.md has no `helpers:` frontmatter

**Verifier (high confidence)**: Verified: lines 371-481 are SRAD (7.7KB, 24% of preamble); consumers are exactly the 6 cited skills (+_generation/_cli-fab); fab-clarify lacks helpers:. No Go coupling — sync.go listSkills auto-deploys any new .md; helpers allowlist is prose-only. Minor corrections: non-consumers ~14 not ~16 (4 skills skip _preamble entirely); internal-skill-optimize.md:33,48 pointers to "SRAD in _preamble.md" need updating. SPEC mirror update is normal procedure. Could not refute.


### `f004` [HIGH] Context-budget sweep: per-invocation load ranges 36.7KB–131.5KB; _preamble (32.3KB) dominates every skill

**Files**: _cli-fab.md, _preamble.md, fab-operator.md

Measured per-invocation context (skill body + _preamble 32,260B + declared helpers + 7 always-load files 13,122B). fab-operator tops at 134.6KB; pipeline orchestrators 70–77KB; even tiny skills like fab-discuss (3KB body) pay 48.4KB. Five quantified reductions identified (see separate findings): preamble SRAD split (~11.8KB×15 skills), _cli-fab split (14.1KB×operator), dead preamble sections (~4.9KB×17), stage-time helper loading (8.7–19.2KB×orchestrators), operator compression (~5KB).

**Recommendation**: Apply the five reduction findings below in order of leverage: (1) move SRAD+Confidence out of _preamble, (2) split _cli-fab, (3) delete/move dead preamble sections, (4) defer _generation/_review loading to stage entry, (5) compress fab-operator.

**Evidence**: wc -c totals (bytes; P=preamble 32260, AL=always-load 13122, G=8685, R=10530, CF=31647, CE=6502): fab-operator 134,628 (51097+P+CF+CE+AL); fab-continue 79,201 (14604+P+G+R+AL); fab-fff 73,877; fab-ff 72,378; fab-new 63,588 (9521+P+G+AL); fab-draft 61,191; git-pr-review 59,728*; git-pr 57,988*; fab-proceed 57,952; docs-reorg-memory 55,668*; fab-clarify 54,137; fab-setup 53,477 (AL-exempt); fab-archive 52,870; git-branch 49,254*; docs-hydrate-specs 48,734; docs-reorg-specs 48,454*; fab-discuss 48,400; internal-retrospect 47,307*; fab-help 46,603 nominal/1,221 actual*; internal-skill-optimize 45,529; docs-hydrate-memory 39,919 (AL-exempt); fab-switch 37,666 (config-only); fab-status 37,563 (AL-exempt). *=skill body never instructs (or disclaims) part of this load — see universal-load finding.

**Verifier (high confidence)**: Reproduced every figure: helper/byte counts exact (P=32260, CF=31647, CE=6502, G=8685, R=10530), AL layer = exactly 13122B across the 7 preamble-mandated files, all 23 row sums verified, exemptions match preamble line 34, fab-help "no context" confirmed. Nits only: internal-skill-optimize could be 49,966 if AL applies; internal-consistency-check omitted; bytes not tokens, AL repo-specific. Constitution line 31 hard-codes _cli-fab.md name (amendment needed if split) but nothing breaks. High severity stands: preamble is 2-26x skill body on every invocation.


### `f042` [MEDIUM] Confidence Scoring formula/schema internals belong in _cli-fab, not _preamble

**Files**: _cli-fab.md, _preamble.md, fab-clarify.md · merged from 2 reports

Agents never compute the score — `fab score` does. Yet the always-loaded preamble carries the formula (503-512), .status.yaml schema (490-499), template internals (534-536), and Bulk Confirm (538-544) whose 'Step 1.5'/'Step 2 in Suggest Mode' references are only meaningful inside fab-clarify.md, which already defines the flow at its line 56.

**Recommendation**: Keep only Gate Threshold + Invocation (who scores, when, 3.0 flat gate). Move Formula/Schema/Template into `_cli-fab` § fab score (extended). Delete the Bulk Confirm subsection — fab-clarify.md:56-133 is already the authoritative definition; the preamble copy duplicates its trigger condition verbatim.

**Evidence**: _preamble.md:538-544 references 'Step 1.5' and 'Step 2 in Suggest Mode' (fab-clarify-internal step numbers); fab-clarify.md:56 '### Step 2: Bulk Confirm (Confident Assumptions)'

**Verifier (high confidence)**: All citations verified: _preamble.md:490-544 carries schema/formula/template/Bulk Confirm; :542/:544 reference fab-clarify-internal steps; fab-clarify.md:56-133 is the authoritative duplicate. Score is computed in Go (score.go:333-341), formula also in docs/specs/srad.md. No cross-refs break. Caveat: _cli-fab is helper-loaded only by fab-operator, so moved content is invisible to scoring skills — acceptable since none behaviorally need it; kept Gate Threshold+Invocation covers orchestrators. Severity medium stands (~61 of 544 always-loaded lines).


### `f090` [MEDIUM] fab-status: unclear whether preamble §2 steps 4-5 (artifact load, logging) apply

**Files**: fab-status.md

fab-status exempts itself only from Always Load, then says 'Run the preflight script' — which is preamble §2, whose step 5 loads intake.md and plan.md and step 4 logs the command. A literal agent reads intake+plan (significant waste for a glance command); another skips both and never logs telemetry. Sibling skills (fab-discuss, fab-switch) spell out both decisions; fab-status is silent.

**Recommendation**: Add to fab-status's Context Loading section: 'Do not load change artifacts (intake.md, plan.md)' (mirroring fab-discuss.md:36) and an explicit Command Logging line `fab log command "fab-status" "<id>" 2>/dev/null || true`.

**Evidence**: fab-status.md:28 (exempts Always Load only), :34-38 (runs preflight, nothing about §2 steps 4-5); _preamble.md:57-58 (step 4 logging, step 5 "Load all completed artifacts in the change folder (`intake.md`, `plan.md`)")

**Verifier (high confidence)**: Verified: fab-status.md:28 exempts only Always Load; :34 invokes preflight (=_preamble §2 step 1), and §2 steps 4-5 (log, load intake/plan) are unaddressed. Siblings fab-discuss:36-51, fab-switch:67, fab-help:21 spell out both. SPEC-fab-status flow shows no artifact load AND no logging, so the logging line adds behavior — must reconcile SPEC mirror and the skill's "no side effects" Key Property, but every other user-invocable skill logs, and log_test.go already uses "fab-status".


### `f117` [MEDIUM] Operator loads code-quality, code-review, and both doc indexes it never uses — against its own 'Context discipline'

**Files**: _preamble.md, fab-operator.md

§2 loads the full always-load layer (7 files) into a long-lived session whose §1 principle reserves the context window for coordination state and forbids reading change artifacts. The operator never generates code, reviews, or memory — code-quality.md, code-review.md, memory index, and specs index are dead weight re-paid after every /clear. This is a deliberate behavior change proposal.

**Recommendation**: Trim §2 Context Loading to config.yaml, constitution.md, context.md; add fab-operator to _preamble §1's exception list (alongside /fab-setup, /fab-status) so the contract stays consistent.

**Evidence**: fab-operator.md:43 'Load the always-load layer... (config, constitution, context, code-quality, code-review, memory index, specs index)' vs :29 'Context discipline. The operator never reads change artifacts... Its context window is reserved for coordination state'.

**Verifier (high confidence)**: Verified: fab-operator.md:43 loads all 7 files; code-quality/code-review/doc indexes are used nowhere in 646 lines. Strongest counter: §1 discipline literally forbids only change artifacts, so no letter-level contradiction — spirit-level only. But multi-repo design strengthens it (§6: "Do NOT use the operator's own config.yaml"). No Go/test couplings; only _preamble:34, skills.md:59, SPEC-fab-operator.md:22 need normal mirror edits. /fab-switch precedent exists for partial loads. Medium severity defensible for the long-lived, /clear-cycling operator.


### `f122` [MEDIUM] Orchestrators load _generation+_review (19.2KB) at invocation even when the current stage needs neither

**Files**: _preamble.md, fab-continue.md

fab-continue (the highest-frequency skill) declares helpers [_generation, _review], loading 19,215B up front. But _generation matters only at apply entry when plan.md is absent, and _review only at the review stage. A hydrate/ship/review-pr invocation, or an apply-resume with plan.md present, pays for both unused. Same for fab-ff/fab-fff (whose sub-stages run in subagents anyway).

**Recommendation**: Change _preamble § Skill Helper Declaration semantics to allow stage-conditional loading (e.g., 'read .claude/skills/_review/SKILL.md when entering review behavior'), and move fab-continue's helpers into per-stage read instructions in Apply/Review Behavior sections. Saves 8.7–19.2KB on most fab-continue invocations (77.3KB → 58–69KB).

**Evidence**: fab-continue.md:4 `helpers: [_generation, _review]`; _preamble.md:109 mandates reading all declared helpers "After reading `_preamble` and before executing the skill body". fab-continue.md:100 shows plan generation is skipped when plan.md exists; review/_review content unused at hydrate/ship stages (fab-continue.md:162-177).

**Verifier (high confidence)**: Verified: fab-continue.md:4 declares both helpers; _preamble.md:109 mandates unconditional pre-body reads; sizes are exactly 8685+10530=19215B; hydrate/ship/review-pr use neither; fab-ff/fff subagents re-read helpers themselves (fab-ff.md:54,66). No Go/test coupling — skills.md:52 says helper validation is convention-only; only SPEC mirrors need updating. Minor gap: _generation also serves the rare intake-active regeneration path (fab-continue.md:50), so the conditional rule must cover it. Tradeoff: conditional loading is agent-compliance-dependent, but in-body helper references at lines 101/144 already backstop it.


### `f041` [LOW] Dormant [AUTO-MODE] Skill Invocation Protocol occupies always-loaded _preamble

**Files**: _preamble.md, fab-clarify.md · merged from 3 reports

_preamble.md:310-334 defines the [AUTO-MODE] protocol, then admits at line 325 'No skill currently invokes another with the [AUTO-MODE] prefix' (auto-clarify removed in 1.10.0). Its only remaining reference is fab-clarify's Auto Mode, itself 'retained for future use' (fab-clarify.md:181). ~25 lines of dead protocol taxed on every skill invocation.

**Recommendation**: Move the protocol definition into fab-clarify.md (its sole referencer) and leave a 2-line pointer in _preamble, or delete both dormant sections until an orchestrator actually uses the prefix. Deleting fab-clarify's Auto Mode is a behavior decision — surface it to the kit owner rather than cutting silently.

**Evidence**: _preamble.md:325 'No skill currently invokes another with the `[AUTO-MODE]` prefix'; grep shows AUTO-MODE appears only in _preamble.md and fab-clarify.md

**Verifier (high confidence)**: Verified: _preamble.md:325 and fab-clarify.md:181 quotes exact; protocol is dormant; preamble is read by 24/29 skills. Finding undercounts references: _preamble.md:348 (live Subagent Dispatch section) also cites [AUTO-MODE] and needs a one-line fix, plus glossary.md:113 and two SPEC mirrors (normal procedure). No Go/code coupling; constitution unaffected. Severity inflated: ~25 of 544 preamble lines (~4.5%), so low, not medium.


### `f043` [LOW] Bulk-confirm trigger/semantics duplicated between _preamble and fab-clarify

**Files**: _preamble.md, fab-clarify.md · merged from 2 reports

_preamble.md:538-545 ('Bulk Confirm (Confident Assumptions)') restates fab-clarify's trigger (confident >= 3, confident > tentative + unresolved), upgrade semantics (S → 95), and even fab-clarify's internal step numbering. fab-clarify is the only consumer, so every other skill pays ~10 lines per invocation, and the two copies can drift (step numbers already coupled).

**Recommendation**: Cut the preamble subsection to one sentence ('/fab-clarify offers a bulk-confirm flow for Confident assumptions — defined in fab-clarify.md') or delete it; keep fab-clarify.md Step 2 as the sole authority.

**Evidence**: _preamble.md:542 'triggered when confident >= 3 and confident > tentative + unresolved' duplicates fab-clarify.md:60-64; _preamble.md:544 hardcodes 'Step 2 in Suggest Mode ... (Step 1.5)'

**Verifier (high confidence)**: Confirmed: _preamble.md:538-544 duplicates fab-clarify.md:60-64/107-117 (trigger, S→95, hardcoded Step 2/Step 1.5 numbering). No other consumer; no cross-refs break (SPEC-preamble has no bulk-confirm node). Drift already manifest: memory docs clarify.md:174 and planning-skills.md:209 still call it "Step 1.5". Only nit: section is ~7 lines (538-544), not ~10. Severity downgraded to low — small share of a 544-line preamble, copies currently agree in skill sources.



## 3. Duplication, simplification & structure (19)

### `f007` [HIGH] fab-ff/fab-fff duplicate the apply/review/hydrate bracket and rework loop verbatim

**Files**: _review.md, fab-ff.md, fab-fff.md · merged from 5 reports

107 of fab-ff's 136 lines (79%) appear verbatim in fab-fff; most remaining diffs are driver-name token swaps. Only genuinely fff-specific content is Steps 4-5 (~24 lines) plus output/error rows. Drift has already begun: divergent rework-section structure, post-bail guidance, gate terminology, and Assumptions output exist in one twin but not the other.

**Recommendation**: Extract the shared bracket (Pre-flight gate, Context Loading, Behavior note, Steps 1-3 apply/review/hydrate, rework loop, bail message) into a `_pipeline.md` helper parameterized by driver name and terminal stage; add `_pipeline` to _preamble's allowed helpers list. fab-ff/fab-fff become thin wrappers declaring scope and extra steps.

**Evidence**: git diff --no-index fab-ff.md fab-fff.md: 54 insertions/29 deletions. Drift: fab-ff.md:95 post-bail /fab-clarify guidance absent from fab-fff; fab-ff.md:15 'Two gates' vs fab-fff.md:15 'single intake confidence gate'; fab-ff.md:72/83 uses '#### Auto-Rework Loop'/'#### Stop' headings vs fab-fff.md:77 inline '**Retry cap**'; fab-fff.md:15 repeats 'stops at hydrate' twice.

**Verifier (high confidence)**: All evidence verified; duplication is 88% counting driver-token swaps. Drift already harms: fab-ff:15 "Two gates" contradicts fab-fff:15/"sole confidence gate" constitution framing; post-bail /fab-clarify guidance lost in fff; _review.md:160 pointer already stale (rework loop is Step 2, not 3). Helper allowlist (_preamble.md:103) supports adding _pipeline; _review/_generation are precedent. Counter-evidence: _review.md:14-17 deliberately keeps orchestration per-file — but that scopes _review only, and its stale pointer proves the status-quo cost.


### `f031` [MEDIUM] fab-new and fab-draft are near-verbatim twins (Steps 0-9)

**Files**: _generation.md, fab-draft.md, fab-new.md, git-branch.md · merged from 6 reports

Steps 0–8 plus Pre-flight/Arguments are byte-identical (modulo three self-name mentions); only Step 9's tail, activation/branch steps, Output, and three error rows differ. Every change to intake creation must land in 2 skills plus 2 SPEC files. No drift yet, but the high-severity findings above each need fixing twice.

**Recommendation**: Make fab-draft a thin delta: 'Read .claude/skills/fab-new/SKILL.md; execute its Steps 0–9 with these deltas: Step 9 tail = change NOT activated; skip Steps 10–11; Output/Next per Activation Preamble; drop the activation/git error rows.' Do not move shared steps into _generation — fab-continue/ff/fff also load it and would pay the tax.

**Evidence**: diff fab-new.md fab-draft.md shows changes only at lines 2-3,7,29-31,60/62,69/71,116-185/118, Output tail, error rows, Next lines; fab-draft.md is 158 lines, fab-new.md 231

**Verifier (high confidence)**: Diff confirms claim byte-for-byte (231 vs 158 lines; divergence only where cited). Recommendation is grounded: "Activation Preamble" exists (_preamble.md:298-306, names /fab-draft), cross-skill SKILL.md reads are precedented (fab-proceed.md:146), and SPEC-fab-draft.md:5 already describes fab-draft as "identical to /fab-new through Step 9". No step-number cross-refs or Go coupling break. Strongest counter: a delta skill risks an agent running Steps 10-11 (activation) by momentum — real but mitigable tradeoff, not a refutation.


### `f032` [MEDIUM] fab-new Step 11 inlines git-branch's five-case branch logic

**Files**: fab-new.md, git-branch.md · merged from 3 reports

fab-new.md:130-185 reproduces git-branch.md's repo check, 5 cases, commands, and report strings (~56 lines, two sources of truth for identical branch semantics). Already drifting slightly: git-branch has explicit STOPs and frames rev-parse as a condition; fab-new Case 2 lists rev-parse and checkout as sequential commands with no conditional framing.

**Recommendation**: Compress Step 11's five cases into one condition/command/report table (~15 lines), add 'evaluate in order, first match wins' plus a keep-in-sync comment referencing git-branch.md Step 4. Inline copy is the right call for runtime tokens (git-branch.md is 171 lines), so dedupe-by-reference is not recommended.

**Evidence**: fab-new.md:148-180 vs git-branch.md:92-132 — same commands and report strings ('created, leaving {old_branch} intact', 'renamed from {old_branch}'); git-branch.md:98,108 have STOP markers absent in fab-new

**Verifier (high confidence)**: Verified: fab-new.md:130-185 duplicates git-branch.md:23-137 cases/commands/report strings; drift confirmed (STOPs at git-branch.md:98,109 absent; fab-new:153-156 unconditional rev-parse+checkout block). Inline was deliberate (archive hgv7 spec.md:129-131) but only vs subagent dispatch — recommendation keeps inline, so no conflict. No Go/test/skill cross-refs on Step 11 text; SPEC-fab-new.md:54-59 mirror update is normal procedure. architecture.md:327 ("fab-new does not handle branches") is already stale, corroborating drift. Drift is cosmetic today, so medium (not high) stands.


### `f040` [MEDIUM] Operator Spawning Rules stated in three places; move out of always-loaded _preamble

**Files**: _cli-external.md, _preamble.md, fab-operator.md · merged from 4 reports

The 'run wt create in the target repo; read spawn command via fab spawn-command --repo, never own config.yaml' rule appears in _cli-external's wt note AND its tmux note, again in fab-operator §6, and the wt create examples duplicate _preamble § Operator Spawning Rules — a section every skill pays for in always-load though only fab-operator uses it. Three copies of MUST-rules will drift.

**Recommendation**: Move _preamble's 'Operator Spawning Rules' (lines 152-173, ~22 always-load lines saved) into _cli-external's wt section; keep fab-operator §6 as the normative step-by-step procedure; reduce _cli-external to tool syntax plus ONE repo-targeting note (drop the repeat at line 107's tmux bullet).

**Evidence**: _cli-external.md:38 and _preamble.md:161 contain the identical command `wt create --non-interactive --worktree-name <name> <change-folder-name>`; _cli-external.md:41 and :107 both state the `fab spawn-command --repo <target-repo>` / 'not the operator's own config.yaml' rule; fab-operator.md:388-392 restates both.

**Verifier (high confidence)**: All cited lines verified: identical wt-create command at _preamble.md:161 and _cli-external.md:38; spawn-command rule duplicated at _cli-external.md:41 and :107; fab-operator.md:388-392 restates both. Only fab-operator declares helpers:[_cli-external]; "Operator Spawning Rules" heading referenced nowhere else in src/, docs/, or Go code. No constitution conflict; only SPEC-preamble.md:16 mirror needs updating (normal procedure). Weakest counter: copies are currently consistent and §6 stays normative, so drift is potential not actual — medium stands.


### `f049` [MEDIUM] fab-operator spawn sequence restated 4-5x

**Files**: fab-operator.md · merged from 2 reports

The canonical 6-step spawn sequence (§6, lines 383-396) is re-walked in each of the three 'Working a Change' forms (461-483), in Autopilot steps 1-2 (507-508), and in Watches step 4 (590-591) — each repeating 'establish target repo / wt create in it / fab spawn-command --repo / enroll with repo+session'. Only the initial command sent differs.

**Recommendation**: Keep the canonical sequence once; replace the three 'Working a Change' walkthroughs with a 3-row table mapping entry form → initial command (`/fab-switch <change> && /fab-proceed`, `/fab-new <escaped-text>`, `/fab-new <id>`) + 'run the §6 spawn sequence'. Autopilot step 1-2 and Watches step 4 become one-line references.

**Evidence**: fab-operator.md:459 'All three run the repo-targeted spawn sequence above — establish the target repo first, create the worktree...' then each variant (462-465, 471-474, 478-482) re-lists the same steps; :508 and :590 repeat 'read `<spawn_cmd>` via `fab spawn-command --repo ...`' again.

**Verifier (high confidence)**: Confirmed: sequence at 385-396 is restated at 459, 462-465, 471-474, 479-482, 507-508, 590-591. No external cross-refs break (zero grep hits outside fab-operator.md; SPEC mirror summarizes spawning once). Watches step 4 already uses the §6-reference pattern, refuting the "local restatement aids compliance" objection. Caveat: table must preserve shell-escaping (473), idea-lookup pre-step (478), --reuse (507), watch enrollment extras (591). Multi-repo edits touched all 5 sites — drift cost is demonstrated, severity medium stands.


### `f077` [MEDIUM] Bootstrap steps 1c-1g, 1i, 1k duplicate fab sync scaffolding, and sync runs last

**Files**: fab-setup.md, setup.md, sync.go

`fab sync` already copy-if-absent installs context.md, code-quality.md, code-review.md, both index.md files (scaffoldTreeWalk), creates fab/changes/+archive/+.gitkeep (scaffoldDirectories), and merges .gitignore entries. The skill places sync at step 1j, after hand-instructing the agent to do all of this manually (1c-1g, 1i, 1k). Memory doc confirms 'running fab sync alone creates a complete structural scaffold'.

**Recommendation**: Move step 1j (`fab sync`) to immediately after Phase 0 and delete steps 1c-1g, 1i, and 1k, keeping only 1a/1b (interactive config/constitution). Rewrite Bootstrap Output (lines 142-164) accordingly. Explicit behavior-order change; outcome identical via idempotency. Saves roughly 55 always-loaded lines.

**Evidence**: fab-setup.md:83-140 vs sync.go:204-229 (directories/.gitkeep), sync.go:266-329 (scaffold copy-if-absent), and docs/memory/distribution/setup.md delegation table ('Directories... fab-kit sync; Skeleton files... fab-kit sync; .gitignore entries... fab-kit sync').

**Verifier (high confidence)**: Confirmed: scaffold/ holds all files 1c-1g copy (scaffoldTreeWalk copy-if-absent, sync.go:266-329); scaffoldDirectories (204-229) covers 1i; fragment-.gitignore `.fab-*` glob subsumes 1k. setup.md:84-96 assigns ownership to fab-kit sync. Strongest counter: Sync() needs config.yaml fab_version (sync.go:43-45, config.go:68-71), so sync-first fails pre-`fab init` — but the skill is itself sync-deployed and that path already fails at 1j today. Add sync-failure guard; renumber 1h's "step 1j" reference.


### `f080` [MEDIUM] Migrations version reading/parsing/comparison stated three times, duplicating the binary

**Files**: fab-setup.md

Version handling is restated: Context Loading 1 (read both files), Pre-flight 1-2 (existence) and 4 (parse as integers), Step 1 (manual compare), plus the standalone Semver Comparison section — all owned by `fab migrations-status`, which the skill itself says owns discovery and which exits non-zero on missing version files. `fab migrations-status --json` is also instructed twice (lines 328 and 349).

**Recommendation**: Delete Pre-flight checks 1, 2, 4 (lines 334-337), Step 1 (lines 339-343), and the Semver Comparison section (lines 460-462). Run migrations-status once; pick the equal/ahead/no-op output by comparing the returned `local`/`engine` fields (one-line rule). Saves ~20 always-loaded lines.

**Evidence**: fab-setup.md:326-328, 334-337, 339-343, 460-462 vs fab-setup.md:347 ('Discovery is owned by the binary — do NOT scan, parse, validate, or sort') and _cli-fab.md fab migrations-status entry ('Non-zero only on a genuine error (missing fab/.kit-migration-version, missing engine VERSION...)').

**Verifier (high confidence)**: Confirmed: version read/parse/compare appears at fab-setup.md:326, 334-337, 339-343, 460-462, and migrations-status is instructed twice (328, 349). migrations_status.go:61,71 exits non-zero with the same remediation hints, and --json returns local/engine, so deleting the duplicates loses nothing. No anchor/cross-references to Semver Comparison; constitution unaffected. Minor caveats: line 347's "do NOT parse" targets the migrations dir (contradiction framing slightly stretched), and the fix should also drop Context Loading 1.


### `f094` [MEDIUM] git-pr resolves the active change up to six times while warning against re-resolution

**Files**: git-pr.md

Steps 0a, 0b, 1, 1b, 3c, and 4a each instruct a fresh `fab change resolve 2>/dev/null`, while Step 3c itself says 'do NOT re-run fab change resolve — reuse the single resolution to avoid inconsistency' (line 175). The file contradicts its own rule and repeats the resolve+intake-exists check three times.

**Recommendation**: Resolve once in a unified Step 0 ('resolve change context: {name}, {has_fab}, {has_intake}, {change_type}'), then reference those variables in Steps 0b, 1, 1b, 3c, and 4a; delete the per-step resolve commands.

**Evidence**: git-pr.md:19, :37, :70, :81, :161, :174, :225 (resolve instructions) vs :175 ('do NOT re-run fab change resolve — reuse the single resolution')

**Verifier (high confidence)**: Confirmed: resolve instructed at git-pr.md:19,37,70,81,161,172,225; resolve+intake check repeated 3x. Caveat: line 175's "do NOT re-run" is scoped to the pr-meta call, not file-wide, so "contradicts its own rule" is slightly overstated — but Step 3c resolves twice despite "attempt once" (line 171). Cross-refs (_cli-fab.md:266-292, prmeta.go) cite Step 0b/3c by name and survive since the recommendation keeps those steps. No behavior loss: resolution deterministic, pointer never changes mid-run. Medium severity fair.


### `f098` [MEDIUM] git-pr-review states the triage taxonomy three times and restates step rules in Rules section

**Files**: git-pr-review.md

The fix/defer/skip/informational taxonomy appears as classification-with-examples (line 118), disposition-intent definitions (lines 120-123), and the Disposition Reference table (lines 237-247). The Rules section (225-233) restates Step 5's git-reset behavior, Step 5.5's dedup/best-effort replies, and Step 6.5's best-effort push nearly verbatim. Roughly 25 lines of repetition per invocation.

**Recommendation**: Merge Step 4 items 1 and 3 into one classify-and-assign list (keep the examples); keep the Disposition Reference table as the single reply-format source and drop reply formats from Step 5.5 item 1; cut Rules to the two non-restated lines (fully autonomous; targeted fixes only).

**Evidence**: git-pr-review.md:118 vs :120-123 vs :237-247 (same taxonomy); :228-233 vs :148, :162, :178, :205 (rules restating steps)

**Verifier (high confidence)**: Confirmed: taxonomy at :118, :120-123, :237-247 with near-verbatim definitions; Rules :227-233 restate :9, :148, :178, :205, :141/162/199. No external refs to Rules or Step 4 item numbers; reply prefixes used by dedup (:162) stay in the kept table. Two cautions: preserve 7-char-SHA/description detail when dropping formats from 5.5, and keep the general fail-fast line (:228) — it has no other general statement. SPEC mirror update is normal procedure.


### `f107` [MEDIUM] reorg-specs drifted behind reorg-memory: no link handling, no migration Kind

**Files**: docs-reorg-memory.md, docs-reorg-specs.md

Most of the 176-vs-116-line asymmetry is justified (specs are flat, index hand-curated, no fab index command). But spec files do cross-link (docs/specs/overview.md:77,145-148; srad.md:135), and reorg-specs has no Link Impact preview, no dangling-link verify (Step 5 checks headings only), no git-mv guidance, and its Migration Map (:60-63) only models section moves while the apply summary counts '{C} files created' (:91) — file-level migrations are unrepresentable.

**Recommendation**: Port three scaled-down pieces from reorg-memory into reorg-specs: a Kind column (move-section / move-file / new-file) in the Migration Map, the Link Impact note for move-bearing rows, and the no-dangling-link verify in Step 5. Leave sub-domain/shape machinery out — that asymmetry is correct.

**Evidence**: docs-reorg-specs.md:60-63 Migration Map columns '# | Section | From | To | Rationale'; :75 'verify no headings lost' (no link check); :91 'After apply: ...{C} files created' — vs docs-reorg-memory.md:102-111 Link Impact and :127 hard-block guard

**Verifier (high confidence)**: All cited lines verified exactly. 7 docs/specs files have intra-spec relative links; reorg-specs Step 5 checks headings only, so applied reorgs can dangle links silently. Port is self-contained (reorg-memory:175 says link rewriting is skill-driven, not a fab subcommand). No external reference touches the Migration Map format; only SPEC mirror + skills.md need routine updates. Bonus: SPEC-docs-reorg-specs.md:5 claims "Same pattern as docs-reorg-memory", so drift contradicts its own mirror. Medium severity stands.


### `f116` [MEDIUM] Status-frame spec (~100 lines) is embedded inside tick step 1, with the render-path rationale stated four times

**Files**: fab-operator.md

§4's 7-step tick list is interrupted by the full frame format (lines 200-271) between steps 1 and 2. The 'markdown render path / ANSI is stripped / emoji is the color channel' constraint is repeated at lines 200, 204, 263, and 265 (including design history 'an earlier iteration colored an ANSI-wrapped glyph'). Maintainer rationale, not agent instruction.

**Recommendation**: Extract a 'Status Frame Format' subsection after Tick Behavior (step 1 ends 'emit the status frame — see Status Frame Format'). Collapse lines 200, 263, 265 into one rule: 'Emit bare markdown (no code fence, no headings, no ANSI); channels: tables, emoji, bold, italic, code spans, plain URLs.' Keep the example and the two column tables.

**Evidence**: fab-operator.md:198-271 frame spec inside step 1; :200 'ANSI SGR escapes... are stripped... (empirically verified)'; :204 fence warning repeats the same failure mode; :265 'Why emoji + table, not ANSI: an earlier iteration colored an ANSI-wrapped glyph...' restates :200.

**Verifier (high confidence)**: Confirmed: frame spec is ~74 lines (not ~100) inside tick step 1; rationale repeats at 200/204/253/263/265. Line 265's design history duplicates docs/memory/runtime/operator.md:374-385, and constitution assigns "why" to specs/memory. No breaking cross-refs: §4 references survive; "tick step 1" refs are archived artifacts only; SPEC mirror update is normal. Caveats: keep line 204's runtime no-fence rule (agent-critical, distinct) and "plain URLs"/no-headings in the collapsed rule; spec merged yesterday (PR #388), so churn is fresh.


### `g3-4` [MEDIUM] Change-type inference duplicated in fab-new/fab-draft with drift from the hook that also writes status.yaml change_type

**Files**: artifact.go, fab-draft.md, fab-new.md, hook.go

fab-new.md:77-90 and fab-draft.md:77-93 instruct manual keyword inference ('Contains any of' substrings; refactor list omits 'redesign') then set-change-type. The PostToolUse hook already infers and writes change_type on every intake.md write using word-boundary regexes including 'redesign' (artifact.go:90-111, hook.go:261-263). Divergent semantics flip types ('specify' → test by skill, feat by hook), and any later intake edit (/fab-clarify) re-fires the hook, silently overwriting the skill's value. change_type drives scoring expected_min and the parsimony skip list.

**Recommendation**: Align the contract: either replace Step 6's keyword list in both skills with 'the intake-write hook sets change_type; verify via preflight, override with fab status set-change-type only if wrong', or copy the Go list exactly (add 'redesign', state word-boundary matching) in both files.

**Evidence**: fab-new.md:80 'refactor, restructure, consolidate, split, rename' vs artifact.go:96 regexp '\b(refactor|restructure|consolidate|split|rename|redesign)\b'; hook.go:261-263 changeType := hooklib.InferChangeType(...); status.SetChangeType(...) fires on every intake.md write.

**Verifier (high confidence)**: All citations verified verbatim. Hook is wired PostToolUse Write+Edit (sync.go:24-25), so clarify edits silently overwrite skill-set values; change_type drives expected_min (_preamble.md:512, feat:7 vs test:3) and parsimony skip (_review.md:53). Drift is actually three-way: docs/specs/change-types.md:77 and git-pr.md:40-42 also omit "redesign". Recommendation keeps set-change-type as override, so no cross-refs break; only SPEC mirrors/change-types.md need normal updates. Medium severity stands.


### `f046` [LOW] Preamble context lists re-duplicated verbatim in fab-proceed and fab-discuss

**Files**: _preamble.md, fab-discuss.md, fab-proceed.md · merged from 3 reports

fab-proceed.md:130-139 restates the Standard Subagent Context 5-file required/optional list already normative in _preamble.md:352-364. fab-discuss.md:26-36 restates the 7-file always-load list from _preamble §1. Both skills instruct reading _preamble first, so the copies add no information but will silently diverge when the canonical lists change (e.g., adding an 8th always-load file).

**Recommendation**: In fab-proceed, replace lines 132-139 with 'include the standard subagent context files per _preamble.md § Standard Subagent Context' (the pattern _review.md:35 already uses). In fab-discuss, replace the file list with 'Load the always-load layer per _preamble.md §1' and keep only the do-not-run-preflight deltas.

**Evidence**: fab-proceed.md:132-139 == _preamble.md:357-363 (same 5 files, same required/optional split); fab-discuss.md:28-34 == _preamble.md:38-44

**Verifier (high confidence)**: Confirmed: fab-proceed.md:132-139 verbatim-matches _preamble.md:356-363; fab-discuss.md:28-34 matches _preamble.md:38-44. Both skills already mandate reading _preamble first, and _review.md:35,98 proves the reference-only pattern works. No Go/spec cross-refs break; constitution has no self-containment rule. Downgraded to low: both copies explicitly cite the canonical _preamble section, so divergence is not silent, and the do-not-run-preflight delta is preserved by the recommendation.


### `f055` [LOW] Extended sections for change/resolve/log restate _preamble content that is always co-loaded

**Files**: _cli-fab.md, _preamble.md

Every loader of _cli-fab also loads _preamble, yet the resolve flag table (_cli-fab.md:117-127) repeats _preamble.md:251 verbatim, the change subcommand list (:37-44) repeats :250, and the log calling convention (:101-109) repeats :249. Double-maintenance has already produced drift (preflight finding above). The headline/extended split is good in principle but fragile — it only works if both files are updated together.

**Recommendation**: In each '(extended)' section keep only genuinely additive details — e.g. the archive YAML soft-skip caveat (:46), `--indicative` no-op notes (:65-66), log exit-code semantics (:107) — and delete restated signatures/flag tables, pointing to _preamble § Common fab Commands instead.

**Evidence**: _cli-fab.md:118 'fab resolve [--id|--folder|--dir|--status|--pane] [<change>]' duplicates _preamble.md:251 exactly; _cli-fab.md:40 'switch <name> | --none' duplicates _preamble.md:250.

**Verifier (medium confidence)**: Verified: _cli-fab:118=_preamble:251 signature, :40=:250 switch; sole loader fab-operator also reads _preamble; drift confirmed (_cli-fab:92 omits preflight `id`, present in _preamble:247 and preflight.go:97). But finding overstates: log block shares only 1 of 4 subcommands with :249, and the resolve flag table is additive (per-flag outputs, --pane $TMUX) — deleting tables as recommended would lose behavior. Apply surgically (pure restatements only). One loader, few overlapping lines: severity low.


### `f075` [LOW] fab-proceed Asymmetric-Bias rationale is spec content, not instruction

**Files**: fab-proceed.md

The operative rule is one sentence (ambiguous relevance → not clearly relevant). The following ~9 lines explain why the failure modes are asymmetric — pure design rationale paid on every invocation. The constitution places design 'why' in docs/specs, not in runtime skill files.

**Recommendation**: Keep fab-proceed.md:113 (the MUST sentence) plus one line ('bias toward the recoverable failure: a bypassed draft is recoverable via /fab-switch; an activated wrong draft is not'). Move the false-positive/false-negative breakdown to docs/specs/skills/SPEC-fab-proceed.md.

**Evidence**: fab-proceed.md:111-122 — rule at :113, then 'The failure modes are asymmetric: ... False positive ... False negative ... Biasing toward the recoverable failure is the design intent.' Constitution VI: specs record the 'why'.

**Verifier (high confidence)**: Verified: rule at fab-proceed.md:113; lines 115-120 are rationale only. SPEC-fab-proceed.md:85-92 already contains the identical breakdown, so the fix is a pure delete-from-skill (no spec move needed). Constitution VI ("specs record the why") supports it. No Go code or other skill references the breakdown; memory file duplicates it independently. Recommendation's one-line summary preserves the bias signal. Severity inflated: zero behavior change, ~6 lines saved.


### `f078` [LOW] Step 1k appends a redundant .gitignore entry already covered by sync's .fab-* fragment

**Files**: fab-setup.md, fragment-.gitignore

Sync's fragment merge adds `.fab-*` to .gitignore (step 1j runs before 1k). Step 1k then checks for the literal `.fab-status.yaml`, doesn't find it, and appends it — producing a duplicate-coverage entry on every fresh bootstrap and diverging from sync's `.fab-*` convention.

**Recommendation**: Delete step 1k (lines 138-140) and the 'Updated: .gitignore' output line (line 158); .gitignore ownership belongs to `fab sync` per the distribution memory's delegation table.

**Evidence**: fab-setup.md:138-140 ('If `.fab-status.yaml` is not listed, append it') vs scaffold/fragment-.gitignore line 2 (`.fab-*`), merged by sync.go lineEnsureMerge (sync.go:434-506).

**Verifier (high confidence)**: Confirmed: sync.go:73 runs scaffoldTreeWalk; lineEnsureMerge (sync.go:434-506) merges fragment's `.fab-*` into root .gitignore during step 1j, making step 1k's literal `.fab-status.yaml` append redundant. Delegation table (docs/memory/distribution/setup.md:89) assigns .gitignore to sync. Only SPEC-fab-setup.md:42-43,77 mirror 1k (normal update); no Go/test dependence. But harm is one harmless redundant line — duplication cleanup, not a bug. Severity medium is inflated; low.


### `f087` [LOW] fab-archive restore mode is specified twice (two top-level documents in one file)

**Files**: fab-archive.md

Restore's purpose and arguments appear in the main header section (lines 14, 24-30) and again verbatim in a second `#`-level document (lines 117-128: Purpose, Arguments, plus boilerplate Pre-flight/Context Loading). Two top-level titles in one skill file; duplicated text will drift (e.g., a future flag added in one place only).

**Recommendation**: Merge: keep mode detection and both argument lists once at the top; demote line 117's section to `## Restore Mode` containing only its unique content (Behavior, Output, Error Handling, Key Properties); delete the duplicated Purpose/Arguments/Pre-flight/Context Loading at lines 119-142.

**Evidence**: fab-archive.md:26-28 vs :125-128 (same `restore <change-name> [--switch]` arguments restated); :6 `# /fab-archive [<change-name>] | restore ...` vs :117 `# /fab-archive restore <change-name> [--switch]`

**Verifier (high confidence)**: Confirmed: two real top-level docs (lines 6, 117); restore args duplicated (26-28 vs 127-128); only skill with this structure; SPEC mirror is already single-doc. No cross-refs break. Caveat: recommendation wrongly calls restore Pre-flight (132-135) duplicated boilerplate — it uniquely waives preflight/hydrate-guard, opposite of archive mode (34-37); the merge must preserve it as mode-specific. Severity lowered: ~6 duplicated lines, currently consistent, one-file blast radius.


### `f099` [LOW] git-pr-review Phase Sub-State Tracking: undefined <status_file>, misplaced section, raw yq writes

**Files**: git-pr-review.md

The phase-tracking section sits after Step 6.5 although its writes occur during Steps 2-5.5 — a linearly-executing agent has passed all five trigger points before reading it. `<status_file>` is never defined (agent must guess `fab/changes/{name}/.status.yaml`). The `yq -i` mutation also bypasses `fab status`, the otherwise-exclusive writer of `.status.yaml` per _cli-fab.

**Recommendation**: Define `<status_file>` explicitly, move the phase table before Step 2 (or add one-line 'set phase: X' markers inside Steps 2, 4, 5, 5.5), and consider a `fab status set-phase` subcommand to keep .status.yaml writes CLI-owned.

**Evidence**: git-pr-review.md:207-221 (section placement; line 219: 'yq -i ".stage_metrics..." <status_file>' — path never defined)

**Verifier (high confidence)**: Placement (207-221 after Step 6.5) and undefined `<status_file>` confirmed, though the path is inferable from line 196. The "exclusive writer" claim is false: _cli-fab shows fab score and hooks also write .status.yaml, and SPEC-git-pr-review.md:136 explicitly specifies "Direct .status.yaml writes (via yq, not fab CLI)" — deliberate design; drop the set-phase subcommand idea. Phase field has zero consumers, writes best-effort, skills load whole-file. Cheap doc fix (define path, inline markers) still worthwhile; severity low.


### `f113` [LOW] skill-optimize Analysis table and Optimization Rules restate each other; 80-line rule stated twice

**Files**: internal-skill-optimize.md

Three of seven bloat signals duplicate optimization rules (redundant re-explanation ↔ rule 4, excessive output examples ↔ rule 6, over-specified error tables ↔ rule 7), and the under-80-lines skip appears in both batch step 2 and the final Constraints bullet. The skill whose purpose is condensing carries roughly 15 redundant lines itself.

**Recommendation**: Merge the Analysis signal table and Optimization Rules into one signal-to-action table; state the 80-line threshold once (in Constraints) and reference it from batch mode.

**Evidence**: internal-skill-optimize.md:33 vs :48; :34 vs :50; :37 vs :51; :68 ('Skip files under 80 lines') vs :88 ('If a skill is already under 80 lines, report it as "Already lean — skipped"').

**Verifier (high confidence)**: All four cited overlaps verified verbatim (33/48, 34/50, 37/51, 68/88). No cross-refs break: no SPEC mirror exists, nothing references these sections. But "15 redundant lines" overstates (~5-7), and the full table-merge recommendation overreaches — rules 1,2,3,5,8 are invariants, not signals; scope should be deduping the three pairs plus the 80-line threshold. Internal-only skill, no behavioral risk: severity is low, not medium.



## 4. Staleness (20)

### `f014` [HIGH] fab-help prints retired pre-1.10.0 spec/tasks pipeline

**Files**: fab-help.md, fabhelp.go · merged from 2 reports

fab-help.md declares the Go subcommand 'the single source of truth for help content', but that source prints the retired pipeline: 'Planning stages: spec → tasks' and 'Execution stages: apply → review → hydrate'. The constitution mandates six stages with no spec stage. Every /fab-help invocation shows users a wrong workflow.

**Recommendation**: Fix fabhelp.go:103-104 to render the six-stage pipeline (intake → apply → review → hydrate → ship → review-pr) and update fabhelp_test.go per constitution constraint (CLI changes need test updates). The skill md itself needs no change.

**Evidence**: fabhelp.go:103-104: `"Planning stages: spec → tasks"` / `"Execution stages: apply → review → hydrate"`; fab-help.md:25: "The subcommand is the single source of truth for help content"; constitution.md:34: "The core pipeline is six stages ... there is no separate `spec` stage"

**Verifier (high confidence)**: Verified verbatim: fabhelp.go:103-104 prints "Planning stages: spec → tasks" / "Execution stages: apply → review → hydrate"; constitution.md:34 mandates six stages with no spec stage; fab-continue/_cli-fab error on spec/tasks. No test or code parses that text (fabhelp_test.go asserts none of it), stale string exists only in fabhelp.go plus archives, and SPEC-fab-help.md names no stages — fix is isolated and safe. Severity high stands: every /fab-help advertises stages the CLI rejects.


### `f008` [MEDIUM] _cli-fab missing fab pane window-name family the sole consumer depends on

**Files**: _cli-fab.md, fab-operator.md, pane_window_name.go · merged from 5 reports

_cli-fab.md:151 declares `fab pane <map|capture|send|process>` as the complete subcommand set, but the CLI also has `fab pane window-name ensure-prefix <pane> <char>` and `replace-prefix <pane> <from> <to>` (pane_window_name.go:19-43), which fab-operator.md:168-183 requires for enrollment renames. Violates the constitution constraint that CLI changes MUST update _cli-fab.md.

**Recommendation**: Add a `window-name` subsection to the fab pane section documenting ensure-prefix/replace-prefix signatures and idempotent-prefix semantics; fix the subcommand enumeration on line 151.

**Evidence**: _cli-fab.md:151 'fab pane <map|capture|send|process> [flags...]' vs pane_window_name.go:32 'Use: "ensure-prefix <pane> <char>"' and :43 'Use: "replace-prefix <pane> <from> <to>"'; fab-operator.md:171 'fab pane window-name ensure-prefix <pane> »'. Constitution.md:31.

**Verifier (high confidence)**: Confirmed: _cli-fab.md:151 enumerates only map|capture|send|process; zero window-name mentions file-wide. pane.go:21 wires paneWindowNameCmd; ensure-prefix/replace-prefix match cited signatures; constitution MUST-update clause confirmed. Fix is additive, breaks nothing (enumeration string unique to _cli-fab.md; no SPEC-_cli-fab mirror exists). Severity downgraded: fab-operator.md inlines full usage incl. exit codes 2/3 and idempotency, so the "sole consumer" works fine today — harm is doc drift, not operational failure.


### `f027` [MEDIUM] skill-optimize batch mode would rewrite newer helper partials (_review/_cli-fab/_cli-external)

**Files**: internal-skill-optimize.md · merged from 4 reports

Batch mode excludes only _preamble.md and _generation.md. The helper set has since grown: _review.md (161 lines), _cli-fab.md (470), _cli-external.md (129) all exceed the 80-line skip, so a batch run would condense the canonical CLI reference and review helper as if they were skills — exactly what Constraints intends to forbid.

**Recommendation**: In Arguments (line 15) and Constraints (line 87), change the exclusion from two named files to all `_*.md` partials; have Pre-flight read all partials as reference context, not targets.

**Evidence**: internal-skill-optimize.md:15 'except `_preamble.md` and `_generation.md` (shared preambles, not skills)' and :87; the file was last substantively touched at commit 97b12086, before _review/_cli-* helpers existed.

**Verifier (high confidence)**: Lines 15/87 confirmed verbatim; _review(161)/_cli-fab(470)/_cli-external(129) all exceed the 80-line skip and lack protection. Constitution line 31 makes _cli-fab.md the mandated CLI reference, so condensing it is real damage. Fix breaks nothing: no SPEC mirror exists; fabhelp.go:200 already treats `_` prefix as non-skill. Severity downgraded: batch mode requires explicit user approval before writes, the skill is rarely run, and git recovers losses. Evidence detail wrong: _cli-* predate 97b12086.


### `f029` [MEDIUM] internal-retrospect recommends nonexistent /meta:* commands

**Files**: internal-retrospect.md · merged from 2 reports

The Suggested Actions exemplars instruct emitting 'Run /meta:scriptify' and 'Run /meta:review {skill-file}'. No /meta:* skill exists anywhere in the repo (grep across src/ and docs/ finds only these two lines). Retrospectives will direct users to commands that cannot run, or different agents will substitute different guesses.

**Recommendation**: Replace the exemplars with current commands — '/internal-skill-optimize {skill}' for skill condensing; name a real mechanism for the scriptify case or drop that bullet.

**Evidence**: internal-retrospect.md:37-38: '`Run /meta:scriptify` to extract {X}' / '`Run /meta:review {skill-file}` to fix {Y}'; repo-wide grep for '/meta:' returns only these lines.

**Verifier (high confidence)**: Confirmed: internal-retrospect.md:37-38 cite /meta:scriptify and /meta:review; repo-wide grep finds no /meta:* skill (only these lines + the synced .claude/skills mirror). internal-skill-optimize exists as a valid substitute; no SPEC mirror or cross-references break. Strongest counter: lines are "e.g.:" format exemplars, not imperative steps, in an auxiliary internal skill — so impact is dead-command suggestions in retrospective output, not pipeline breakage. Downgrade high to medium.


### `f048` [MEDIUM] Internal skills and helpers missing SPEC files / undiscoverable

**Files**: SPEC-hooks.md, _cli-external.md, _cli-fab.md, _generation.md, fabhelp.go, internal-consistency-check.md, internal-retrospect.md, internal-skill-optimize.md · merged from 2 reports

docs/specs/skills/ has no SPEC for internal-consistency-check, internal-retrospect, internal-skill-optimize, _cli-fab, _cli-external, or _generation, while _review and _preamble do get SPECs (with inconsistent naming: SPEC-_review.md keeps the underscore, SPEC-preamble.md strips it). SPEC-hooks.md exists with no matching skill file.

**Recommendation**: Create the six missing SPEC files (or document in docs/specs/skills an explicit exclusion policy for internal-* and helper files); normalize the underscore convention between SPEC-_review.md and SPEC-preamble.md; note SPEC-hooks.md as covering `fab hook` (non-skill) or relocate it out of skills/.

**Evidence**: ls docs/specs/skills/ shows 24 SPEC files covering all 21 user skills plus _review/preamble/hooks; no SPEC-internal-*.md, SPEC-_cli-*.md, or SPEC-_generation.md exist. Constitution.md:32 "Changes to skill files (`src/kit/skills/*.md`) MUST update the corresponding `docs/specs/skills/SPEC-*.md` file".

**Verifier (high confidence)**: Verified: 6 skill files lack SPECs; SPEC-_review vs SPEC-preamble naming inconsistent; constitution.md:32 quote exact; no exclusion policy documented anywhere. Gap is accidental, not policy: SPEC-_cli-rk.md existed for a helper until 260418-or0o; 260604-rj31 edited _cli-fab.md with no SPEC to update. SPEC-hooks.md is itself stale (cites deleted src/kit/sync/5-sync-hooks.sh, src/kit/hooks/). Counters: fab help exclusion is deliberate (fabhelp.go:200); Constitution VI disfavors retroactive spec generation — prefer the exclusion-policy branch; SPEC-preamble rename touches context-loading.md:68,179.


### `f050` [MEDIUM] Skill-Specific Autonomy Levels table carries pre-1.10.0 residue for fab-continue

**Files**: _generation.md, _preamble.md

The fab-continue column says interruption budget '1-2 per stage' and output includes '[NEEDS CLARIFICATION] count', but post-spec-merge there is one asking stage (intake) and _generation.md:73 bans [NEEDS CLARIFICATION] markers in plan generation. Also 'ask top ~3 unresolved' conflicts with fab-continue.md:64's '1-2 unresolved questions' budget.

**Recommendation**: Update the fab-continue column (lines 409-411): posture 'SRAD at intake stage only; apply decides-and-records', budget '1-2 at intake; 0 at apply+', output drop the [NEEDS CLARIFICATION] count. Align the question count with fab-continue.md:64.

**Evidence**: _preamble.md:410 '1-2 per stage', :411 '[NEEDS CLARIFICATION] count' vs _generation.md:73-76 'No `[NEEDS CLARIFICATION]` markers. Those are an intake-only construct'

**Verifier (high confidence)**: Confirmed verbatim: _preamble.md:409-411 says '1-2 per stage', 'top ~3 unresolved', '[NEEDS CLARIFICATION] count' for fab-continue, while fab-continue.md:42/64 says intake is the only planning/asking stage with a 1-2 budget and _generation.md:73-76 bans markers at apply. Fix loses no behavior; mirrors (docs/specs/srad.md:280-286, SPEC-preamble.md) carry the same residue and updating them is normal procedure. Could not refute. Nuance: '~3' may echo the Critical Rule cap at _preamble.md:403, but still conflicts in-column.


### `f053` [MEDIUM] fab status 'Full subcommand table' omits six visible subcommands

**Files**: _cli-fab.md, status.go

The table at _cli-fab.md:52-70 is labeled 'Full subcommand table' but omits `all-stages`, `progress-map`, `display-stage`, `plan`, `confidence`, and `validate-status-file` — all visible (non-Hidden) subcommands in status.go. An agent trusting the 'Full' claim would hand-parse .status.yaml instead of using the query subcommands.

**Recommendation**: Add the six missing query subcommands as table rows (one line each), or retitle the table 'Mutating + key query subcommands' and add one sentence listing the read-only query subcommands.

**Evidence**: status.go:66 'all-stages', :80 'progress-map <change>', :133 'display-stage <change>', :150 'plan <change>', :169 'confidence <change>', :190 'validate-status-file <change>' — none Hidden (only :336 set-checklist is); _cli-fab.md:52 'Full subcommand table'.

**Verifier (high confidence)**: Confirmed: all six subcommands registered non-Hidden in status.go (lines 66/80/133/150/190; only set-checklist:336 Hidden); table at _cli-fab.md:52-70 omits them. Worse than claimed: _preamble.md:243 promises _cli-fab covers "every subcommand", and the six appear nowhere in skills/specs. No cross-references break; constitution has no conflicting rule. Only mitigation: fab preflight exposes stage/plan/confidence for the active change, slightly softening impact. Medium stands.


### `f062` [MEDIUM] Intake template comments still reference the dead spec stage

**Files**: _generation.md, intake.md

The Intake Generation Procedure (step 1) directs the agent into templates/intake.md, whose comments reference 'spec generation', 'spec generation time', and 'the spec-stage agent' — all contradicting _generation.md's own invocation note ('There is no spec stage... intake.md → plan.md → code') and constitution constraint. Agents generating intakes ingest contradictory stage vocabulary every run.

**Recommendation**: Update the three template comments to post-merge vocabulary: 'primary input for plan generation (apply entry)', 'SRAD handles prioritization at apply entry', 'the intake-stage agent and the apply-entry agent'. No behavior change — terminology only.

**Evidence**: templates/intake.md:29-30 'primary input for spec generation'; :51-52 'SRAD handles prioritization at spec generation time'; :59-60 'between the intake-stage agent and the spec-stage agent' vs _generation.md:56-57 'There is no `spec` stage and no separate `spec.md` artifact'.

**Verifier (high confidence)**: Verified verbatim: intake.md:29-30/:51-52/:58-59 reference spec generation/spec-stage agent; _generation.md:56-57 and constitution.md:34 say no spec stage exists. No Go code or test depends on the comment text; only docs/specs/templates.md:80,126 mirrors it (normal SPEC-mirror update). Fix is comment-only, constitution-aligned. Recommendation misses a fourth stale reference: intake.md:46 "Helps scope the spec." Medium severity holds — stale vocabulary enters every intake run.


### `f079` [MEDIUM] Step 1h describes an unreachable 'new project' version-stamping branch

**Files**: fab-setup.md, sync.go

Step 1h claims sync stamps the engine version when 'no fab/project/config.yaml'. But `fab sync` errors without config.yaml ('not in a fab-managed repo'), and in bootstrap config.yaml is created at 1a before sync at 1j. Engine-version stamping actually happens in `fab init` (stampMigrationVersion, explicitly before Sync). The 'New project: Created: ...({engine_version})' output line can never be produced by this skill.

**Recommendation**: Shrink step 1h (lines 110-121) to two lines: 'Handled by fab sync — existing file preserved; pre-existing projects without it get 0.1.0 (run /fab-setup migrations)'. Drop the new-project branch and its output line.

**Evidence**: fab-setup.md:114 ('New project (no fab/project/config.yaml): copies $(fab kit-path)/VERSION') vs sync.go:43-45 (errors when config nil) and init.go:41-47 ('This must happen before Sync — otherwise scaffoldDirectories... writes 0.1.0').

**Verifier (high confidence)**: Confirmed: fab-setup.md:114/119 vs sync.go:43-45 (errors when cfg nil) and init.go:41-47 (engine stamping lives in init, pre-Sync). In the skill, 1a creates config.yaml before 1j sync, so sync's new-project branch (sync.go:257-263) is unreachable; only 0.1.0/OK lines print. Best counterpoint: the branch exists in code and is reachable in exotic nested layouts — but not via this skill. No cross-refs break (only routine SPEC-fab-setup.md mirror update). Medium stands: agent could misreport and mask the migrations hint.


### `f114` [MEDIUM] Legacy `.fab-operator.yaml` name used ~9 times for a file no longer named that

**Files**: fab-operator.md

The state file is server-keyed `<server-slug>.yaml` under XDG state; §4 says old repo-rooted `.fab-operator.yaml` files are abandoned. Yet the skill keeps calling the live file `.fab-operator.yaml` (§1 x3, §4 heading and tick step 6, §5, §6, §7, §9). Tick step 6 'write updated state to `.fab-operator.yaml`' literally instructs writing a repo-rooted file.

**Recommendation**: Define 'operator state file' once in §4 with the real path, then replace every `.fab-operator.yaml` mention (lines 23, 29, 33, 117, 277, 344, 551, 565, 645) with 'operator state file'.

**Evidence**: fab-operator.md:117 heading '### `.fab-operator.yaml`' immediately followed by :121 'The state file is server-keyed, not repo-rooted'; :277 'Persist — write updated state to `.fab-operator.yaml`'; :121 'Old repo-rooted `.fab-operator.yaml` files... are not migrated'.

**Verifier (high confidence)**: Confirmed: 10 occurrences (finding missed line 638). Go StatePath() proves live file is <server-slug>.yaml; .fab-operator.yaml is the abandoned legacy name. Tick step 6 (line 277) plus tick-start's path-less stdout makes a literal repo-rooted write plausible, which init step 1 never reads — state loss on /clear. Counter-evidence: §4/§9 define the real path adjacent to the name, so it reads as shorthand; severity is borderline medium/low. No anchors or code reference the heading; only the SPEC mirror needs the normal update.


### `f126` [MEDIUM] CONTRIBUTING.md contributor reading path teaches removed architecture (stageman.sh, fab/.kit/ scripts, spec stage)

**Files**: CONTRIBUTING.md

The kit-developer entry document still documents a 'Stage Manager (stageman.sh)' bash utility at fab/.kit/scripts/lib/, with the example `get_stage_number "spec"  # Get position (2)` — the spec stage was removed in v1.10.0 and kit content now lives in the ~/.fab-kit cache per Constitution V. A new contributor following the documented path hits nonexistent files and a dead stage.

**Recommendation**: Rewrite CONTRIBUTING.md's 'Stage Manager' section to reference the Go CLI (`fab status`, `fab preflight`) and the 6-stage pipeline, and fix the reading-path links that point at fab/.kit/ paths.

**Evidence**: CONTRIBUTING.md '## Stage Manager' section: 'fab/.kit/scripts/lib/stageman.sh --help' and 'get_stage_number "spec"                 # Get position (2)'; constitution.md:18 places kit content in ~/.fab-kit/versions/<version>/kit/; constitution.md:34 'there is no separate `spec` stage'.

**Verifier (high confidence)**: Confirmed: CONTRIBUTING.md:83-112 documents stageman.sh at fab/.kit/scripts/lib/ with get_stage_number "spec"; neither path nor script exists anywhere live (only archived changes). Constitution V (~/.fab-kit cache) and the 6-stage/no-spec constraint corroborate. No skill/spec/Go code references the section, so rewriting breaks nothing; fab status/preflight are real CLI commands. Caveats: the 8 reading-path links actually resolve (dead paths are only in the code block), and "v1.10.0" is unverified (removal was change j6cs). Severity medium stands.


### `g4-2` [MEDIUM] fab-continue description uses non-canonical stage names and omits ship/review-pr

**Files**: fab-continue.md

Description says "planning, implementation, review, or hydrate" — 'planning' matches no stage in the 6-stage pipeline (pre-1.10 residue), 'implementation' isn't the stage name, and ship/review-pr are missing even though the body's dispatch table executes both (fab-continue.md:54-55). An agent selecting by description won't route ship/review-pr work to fab-continue.

**Recommendation**: Rewrite the frontmatter description to the canonical list, e.g. "Advance the active change one pipeline stage — intake, apply, review, hydrate, ship, or review-pr — or reset to a given stage." No body change needed; update the matching SPEC file per constitution.

**Evidence**: fab-continue.md:3: "planning, implementation, review, or hydrate" vs fab-continue.md:54-55 dispatch rows for `ship` ("Execute /git-pr behavior") and `review-pr` ("Execute /git-pr-review behavior")

**Verifier (high confidence)**: Confirmed: fab-continue.md:3 says "planning, implementation, review, or hydrate" while constitution.md:34 canonizes intake/apply/review/hydrate/ship/review-pr and the body (lines 22, 54-55) handles all six. The stale string exists only in the source and the fab-sync-deployed copy; no code or skill quotes it, so the fix breaks nothing. SPEC mirror update is normal procedure. Strongest counter: explicit Next:/fab-continue pointers make misrouting rare — insufficient to refute; medium severity stands.


### `f030` [LOW] Retired script names (statusman/changeman/logman) survive across ~9 skills

**Files**: fab-archive.md, fab-discuss.md, fab-draft.md, fab-fff.md, fab-new.md, fab-setup.md, fab-status.md, git-branch.md, git-pr-review.md, git-pr.md · merged from 7 reports

Eleven occurrences of pre-Go-binary component names remain: 'statusman' (fab-new:55, fab-draft:57, fab-fff:99,109, git-pr:243, git-pr-review:190), 'changeman' (git-branch:55,155, git-pr:243,252), 'logman' (fab-setup:55, fab-discuss:54). fab-archive's restore mode also calls `fab change restore` 'the script' six times. 'changeman not found' (git-pr:252) implies a checkable binary that doesn't exist — genuinely ambiguous to an agent.

**Recommendation**: Global rename: 'statusman'→'fab status', 'changeman'→'fab change resolve', 'logman'→'fab log'; in fab-archive restore sections replace 'the script' with 'the command'. Rewrite git-pr.md:243/252 to 'if `fab change resolve` succeeded and `fab status add-pr` ran'.

**Evidence**: grep hits: fab-new.md:55, fab-draft.md:57, fab-fff.md:99,109, git-pr.md:243,252, git-pr-review.md:190, git-branch.md:55,155, fab-setup.md:55, fab-discuss.md:54; fab-archive.md:135,141,161,194-198 ('the script').

**Verifier (high confidence)**: All 11 cited occurrences verified at exact lines; src/kit/scripts/ is gone (names live only in src/benchmark/ and archived changes), and SPEC skill mirrors already have 0 occurrences, so skills lag their own mirrors. Recommendation touches prose only — every executable command already uses fab status/change/log, and the git-pr 243/252 rewrite matches Step 4a exactly. Downgraded to low: nothing fails at runtime; purely clarity/staleness.


### `f033` [LOW] fab-operator autopilot retains geometric glyphs its own frame spec bans

**Files**: fab-operator.md · merged from 5 reports

Section 4 (lines 253-265) mandates emoji health markers and explicitly bans geometric glyphs ('●◌✗ render monochrome and are NOT used'). But §6 Autopilot line 519 still instructs: 'entries with ▶ that show ✓ (green) are complete, the one showing ● (green) / ◌ (yellow) is current' — residue from the pre-emoji frame design. An agent following §6 renders the banned glyphs.

**Recommendation**: Update fab-operator.md:519 to the §4 vocabulary: '… that show ✅ are complete, the one showing 🟢/🟡 is current.'

**Evidence**: fab-operator.md:519 '✓ (green) … ● (green) / ◌ (yellow)' vs fab-operator.md:253 'geometric glyphs like `●◌✗` render monochrome and are NOT used'

**Verifier (high confidence)**: Confirmed: fab-operator.md:519 uses ✓/●/◌ that line 253 explicitly bans; SPEC mirror and docs/memory/runtime/operator.md (PR #387, 2026-06-10) both already canonize emoji (✅/🟢/🟡), so the fix aligns skill with spec and breaks nothing — the stale phrasing occurs nowhere else. Severity downgraded: line 519 is a descriptive aside that itself defers to "§4" for rendering, so agents are unlikely to actually emit banned glyphs; harm is a misleading reading guide.


### `f036` [LOW] fab status fail documented as '(review only)' but review-pr fail is valid and used

**Files**: _cli-fab.md, _preamble.md, fab-continue.md, git-pr-review.md · merged from 4 reports

git-pr-review Step 6 calls `fab status fail <change> review-pr`. The Go binary supports this (status.go:252 'review/review-pr only'), but _preamble (line 252) and _cli-fab (line 61) both document fail as '(review only)' — an agent cross-checking the helper docs would conclude the call is invalid and might skip it, since failures are silenced by `|| true`.

**Recommendation**: Update '(review only)' to '(review/review-pr only)' in _preamble.md line 252 and _cli-fab.md line 61.

**Evidence**: git-pr-review.md:187 vs _preamble.md:252 ('fail <stage> (review only)'), _cli-fab.md:61; src/go/fab/internal/status/status.go:252 ('// Fail transitions a stage to failed (review/review-pr only).')

**Verifier (high confidence)**: Confirmed: status.go stageTransitions allows fail on review-pr and CLI help (cmd/fab/status.go:286) says "review/review-pr only", while _preamble.md:252, _cli-fab.md:61, fab-continue.md:81, docs/specs/user-flow.md:84, docs/memory/pipeline/change-lifecycle.md:101 all say "(review only)". Fix is trivial and breaks nothing; sweep all five spots plus SPEC mirrors. Downgraded to low: the skip scenario is speculative and worst case is bookkeeping drift (stage stays active), not behavior loss.


### `f054` [LOW] fab score table presents 'Gate' and 'Intake gate' as distinct modes — they are the identical invocation (spec-stage residue)

**Files**: _cli-fab.md, score.go

Rows 2 and 3 of the modes table (_cli-fab.md:85-86) distinguish `fab score --check-gate <change>` from `fab score --check-gate --stage intake <change>`. Since `--stage` defaults to intake and only one gate exists post-1.10.0, these are byte-identical behavior. The two-row split is residue of the retired spec gate and implies a second gate exists.

**Recommendation**: Collapse to a single Gate row, or delete the whole 'fab score (extended)' section (_cli-fab.md:78-87) — _preamble.md:248 already documents both modes accurately in one row.

**Evidence**: score.go:44 '--stage ... default "intake" (intake; spec retired in 1.10.0)'; _preamble.md:516 'There is exactly **one** confidence gate'; _cli-fab.md:85-86 lists 'Gate' and 'Intake gate' as separate modes.

**Verifier (high confidence)**: Verified: score.go ignores `stage` entirely (CheckGate/Compute always read intake.md, flat 3.0), so rows 85-86 are byte-identical invocations; commit 97b12086 confirms spec-gate residue. Constitution:34 supports collapse. Caveat: only collapse the rows — deleting the whole section loses the Normal row's unique ".status.yaml write / no indicative key" detail absent from _preamble:248. Severity inflated: both commands behave identically, harm is conceptual only and _preamble:516 (always loaded) corrects it.


### `f060` [LOW] _generation.md consumer list stale; two procedures with disjoint consumers

**Files**: _generation.md

Header says the file is 'used by both /fab-continue and /fab-ff'. Actual declarers: fab-new, fab-draft, fab-continue, fab-ff, fab-fff. Moreover the consumers are disjoint: fab-new/fab-draft invoke only the Intake procedure (loading ~70 unused plan-gen lines), while fab-continue/ff/fff invoke only the Plan procedure (intake-gen lines unused).

**Recommendation**: Fix the frontmatter description (line 3) and intro note (line 11) to list all five consumers and which procedure each uses. Optionally split into _generation-intake/_generation-plan (requires extending _preamble.md:103's allowed-helpers list) if the per-load waste matters.

**Evidence**: _generation.md:3 'used by fab-continue and fab-ff'; :11 'used by both `/fab-continue` and `/fab-ff`'. grep helpers: fab-new.md:4 and fab-draft.md:4 declare [_generation]; fab-continue.md/fab-ff.md/fab-fff.md:4 declare [_generation, _review]. fab-continue/ff/fff never reference 'Intake Generation Procedure' (grep empty).

**Verifier (high confidence)**: All claims verified: _generation.md:3/:11 name only fab-continue/fab-ff; five skills declare it (fab-new/draft use Intake procedure only, continue/ff/fff use Plan procedure only — grep confirms). docs/specs/skills.md:46-47 already lists all five, so only the header is stale; no SPEC-generation.md mirror, fix breaks nothing. Downgrade to low: consumers name the correct procedure inline, so behavior unaffected; two-line text fix. Optional split correctly flagged as needing _preamble.md:103 + spec updates.


### `f064` [LOW] fab-clarify Auto Mode is dead code and 1.10.0 intake-only rationale is restated four times

**Files**: fab-clarify.md

No skill invokes clarify with [AUTO-MODE] (stated at lines 17 and 181; _preamble.md:325-329). Auto Mode (lines 179-187) plus its <!-- blocking: --> marker (absent from the preamble's marker table) are retained 'for future use'. The intake-only/spec-removal explanation is repeated at lines 14, 27, 36, and 181 — historical residue paid on every clarify invocation.

**Recommendation**: Delete the Auto Mode section and the mode split in Purpose (preamble's Skill Invocation Protocol already preserves the [AUTO-MODE] definition); trim Arguments line 27 to 'Any positional argument is treated as a change name'; keep the intake-only rationale once, in Purpose. Drop 'Both Suggest and Auto modes recompute' from Step 7 (line 171).

**Evidence**: fab-clarify.md:14, :17, :27, :36, :171, :179-187; _preamble.md:325-329 'No skill currently invokes another with the [AUTO-MODE] prefix'; <!-- blocking: --> at :184 has no consumer and is missing from _preamble.md:440-446 marker table

**Verifier (high confidence)**: All cited lines verified; no caller of [AUTO-MODE] or consumer of blocking-marker exists (fab-ff/fff explicitly forbid in-bracket clarify; constitution frontloads judgment to intake). Counter-evidence: retention is deliberate ("for future use", _preamble:329), and line 36 repetition is load-bearing error text. Implementing also requires _preamble:444/:544, stale SPEC-fab-clarify:12, glossary:113 updates — doc-only, no behavior breaks. Dead section is labeled dormant so cannot mislead; severity is low, not medium.


### `g3-5` [LOW] Template Status header fields (Draft / In Progress) are written once and never updated or read by any skill

**Files**: intake.md, plan.md

intake.md:5 '**Status**: Draft' and plan.md:4 '**Status**: In Progress' are scaffolded into every change but no skill in src/kit/skills/ ever reads or flips them (grep finds no Draft/In Progress handling). Shipped changes permanently claim Draft/In Progress while real state lives in .status.yaml progress — misleading dead contract fields.

**Recommendation**: Delete the '**Status**:' line from both templates (state of record is .status.yaml); alternatively add explicit update instructions to fab-new/fab-continue, but deletion is cheaper and removes the drift.

**Evidence**: templates/intake.md:5 '**Status**: Draft'; templates/plan.md:4 '**Status**: In Progress'; grep -n 'Draft|In Progress' across src/kit/skills/*.md returns no skill that updates either field.

**Verifier (high confidence)**: Claim verified: templates/intake.md:5 and plan.md:4 contain the Status lines; no skill or Go code reads/updates them; archived shipped changes (e.g., archive/2026/04/260423-qszh-*/) still say Draft/In Progress. Only extra writer found: migrations/1.8.0-to-1.9.0.md:56 also scaffolds the line (frozen migration, no parser — not a break, just edit for consistency). Spec mirror docs/specs/templates.md:75,178,287 needs the normal mirror update. Severity low is correct; deletion is cheap and safe.


### `g4-8` [LOW] _generation helper description lists a stale consumer set (fab-continue and fab-ff only)

**Files**: _generation.md

Both the frontmatter description and the body preamble say _generation is "used by fab-continue and fab-ff", but five skills declare it in helpers: fab-new, fab-draft, fab-continue, fab-ff, fab-fff. Maintainers (and internal-skill-optimize, which reads it as the dedup reference) get a wrong picture of the blast radius.

**Recommendation**: Update _generation.md:3 description and the line-11 blockquote to name all five consumers, or drop the consumer list entirely ("shared logic for intake and plan generation; loaded via frontmatter helpers:").

**Evidence**: _generation.md:3,11: "used by fab-continue and fab-ff" vs `helpers: [_generation]` in fab-new.md:4, fab-draft.md:4, fab-fff.md:4

**Verifier (high confidence)**: Confirmed: _generation.md:3,11 say "used by fab-continue and fab-ff" but five skills declare AND use it (fab-new.md:73, fab-draft.md:75 follow Intake Generation Procedure directly). docs/specs/skills.md:46-47 already lists all five, so only the skill file is stale; no SPEC-generation mirror, no other reference to the phrase, no constitution conflict. Weak counterpoint: fab-fff uses it only via /fab-continue subagent — irrelevant since fab-new/fab-draft are direct consumers. Two-line fix; severity low stands.



## 5. Consistency & conventions (24)

### `f011` [HIGH] _cli-external /loop constraints contradict fab-operator's tick/loop lifecycle

**Files**: _cli-external.md, fab-operator.md · merged from 2 reports

The /loop Constraints block says the loop stops 'when the monitored set becomes empty' and that 'autopilot uses its own cadence (default 2m); replaces any existing monitoring loop'. fab-operator defines one 3m loop that runs while monitored set OR autopilot OR watches exist; no 2m autopilot loop exists anywhere. An agent following _cli-external would stop the loop while autopilot/watches are active, or spawn a phantom 2m loop.

**Recommendation**: Delete the '### Constraints' subsection from _cli-external.md (lines 124-129), keeping only the /loop invocation syntax. Loop lifecycle policy is owned by fab-operator.md §4 'The Loop' and Tick step 7 — leave it there as the single source.

**Evidence**: _cli-external.md:127-129 ('Stop: when the monitored set becomes empty...', 'autopilot uses its own cadence (default 2m)') vs fab-operator.md:115 ('runs as long as the monitored set is non-empty, an autopilot queue is active, or any watch is configured. When all three are empty, stop'), fab-operator.md:68,275,278,625 (single `/loop 3m "operator tick"`, autopilot dispatched inside the same tick).

**Verifier (high confidence)**: Confirmed verbatim. _cli-external.md:124-129 is stale text from archived change 260311-5c11 (old single-repo design); fab-operator.md:115/278 supersedes it (3m loop, three-condition lifecycle, autopilot inside tick; no 2m loop exists). Concrete harm: watch-only operator would stop its loop, killing watch polling. Only fab-operator loads _cli-external; no spec mirror of the text; nothing breaks. Caveat: deletion drops 'One loop at a time' (only implicit in §4) — keep a one-line pointer to fab-operator §4.


### `f001` [MEDIUM] _preamble universal contract (always-load exemptions) out of sync with ~10 skills

**Files**: _preamble.md, docs-hydrate-specs.md, docs-reorg-memory.md, docs-reorg-specs.md, fab-archive.md, fab-help.md, fab-switch.md, git-branch.md, git-pr-review.md, git-pr.md, skills.md · merged from 10 reports

_preamble.md:34 exempts only fab-setup/fab-status/docs-hydrate-memory (fab-switch partial) from the 7-file always-load, yet docs-reorg-*, docs-hydrate-specs, git-branch, fab-switch, fab-archive self-declare config/constitution not required, and 9 skills lack the preamble-read line entirely. _preamble.md:265 ('Every skill MUST end with Next:') is violated by git-pr, git-pr-review, git-branch, fab-discuss, fab-operator. An agent cannot determine which rule wins.

**Recommendation**: Make _preamble.md descriptive, not exhaustive: change §1 to 'unless the skill's own Context Loading section says otherwise', and scope the Next:-line MUST to pipeline-state skills (or list the exempt skills). Also reconcile fab-switch: _preamble.md:34 says it loads config.yaml; fab-switch.md:110/123 says config is not required.

**Evidence**: _preamble.md:34, :265; docs-reorg-specs.md:27 'Does NOT require .fab-status.yaml, config, or constitution'; fab-switch.md:110 'config not required'; `grep -L 'Read the \`_preamble\`' *.md` returns 9 non-helper skills; git-pr.md:259 ends 'Shipped.' with no Next: line

**Verifier (high confidence)**: Verified verbatim: _preamble.md:34/:265, docs-reorg-specs.md:27, fab-switch.md:110/:123 vs preamble's "loads only config.yaml", git-pr ends "Shipped." with zero Next: lines; fab-discuss.md:104/fab-operator.md:641 declare "Outputs Next:? No" while loading the preamble — direct contradiction. skills.md:90 vs :659 mirrors it. Nit: 8 (not 9) non-helper skills lack the preamble line. No cross-refs break; constitution unaffected. Severity inflated: failure modes are wasted context/inconsistent Next:, not corruption — medium.


### `f010` [MEDIUM] fab-operator uses raw tmux against its own fab pane helper mandate

**Files**: _cli-external.md, _cli-fab.md, fab-operator.md · merged from 3 reports

_cli-external (a declared helper, always co-loaded) says 'Use fab pane capture instead of raw tmux capture-pane' and 'Use fab pane send instead of raw tmux send-keys'. fab-operator §5 specifies raw `tmux capture-pane -t <pane> -p -S -20` and `tmux send-keys`, hand-rolling the pane-exists/idle validation that `fab pane send` performs natively. Two agents will pick different mechanisms.

**Recommendation**: Pick one canonical path: either rewrite §5 Question Detection step 1 and Sending Auto-Answers to use `fab pane capture`/`fab pane send` (collapsing §3 Pre-Send steps 1–2 into the binary, behavior change — say so), or add an explicit raw-tmux carve-out for §5 in _cli-external's Usage Notes.

**Evidence**: fab-operator.md:306 'Capture: `tmux capture-pane -t <pane> -p -S -20`'; :338 'Before `tmux send-keys`: verify pane exists and agent is still idle'; :11 'routes commands via `tmux send-keys`'. _cli-external.md:105-106 'Use `fab pane capture` instead of raw `tmux capture-pane`... Use `fab pane send` instead of raw `tmux send-keys`'.

**Verifier (high confidence)**: Confirmed line-for-line. Decisive: intake 260403-tam1-pane-commands (PR #311) explicitly deferred fab-operator migration to fab pane capture/send as a follow-up that never happened. No cross-ref breakage: raw tmux in skills is confined to fab-operator §5 (+SPEC mirror, normal update); new-window stays raw by design. `fab pane capture -l 20 --raw` reproduces `-S -20`; pane_send.go implements §3 steps 1-2. Downgraded to medium: both paths validate equivalently, so divergence is inconsistency, not wrong behavior.


### `f025` [MEDIUM] fab-setup valid config sections contradict (4 vs 7)

**Files**: fab-setup.md · merged from 3 reports

Line 18 says 'Valid sections: project, source_paths, stage_directives, checklist'. Line 176 adds `context`, `code-quality`, `code-review`, matching the 8-item menu (lines 200-211). An agent handling `/fab-setup config context` rejects it per line 18 but accepts it per line 176 — two agents act differently.

**Recommendation**: Delete the 'Valid sections:' list from the Arguments bullet (line 18) and point to Config Arguments (line 176) as the single source of truth.

**Evidence**: fab-setup.md:18 ('Valid sections: `project`, `source_paths`, `stage_directives`, `checklist`') vs fab-setup.md:176 ('Valid values: `project`, `source_paths`, `stage_directives`, `checklist`, `context`, `code-quality`, `code-review`').

**Verifier (high confidence)**: Confirmed: fab-setup.md:18 lists 4 valid sections; :176 and the 8-item menu (:200-211) list 7, matching scaffold files (context.md, code-quality.md, code-review.md exist). Line 18 is stale. No other skill/spec/src reference cites the 4-item list, so deleting it breaks nothing; only the SPEC mirror needs the normal update. Severity inflated: the executing Config Behavior section is correct, and a wrong rejection is recoverable via the menu — medium, not high.


### `f026` [MEDIUM] Header context-loading exception contradicts per-subcommand Context loading notes

**Files**: fab-setup.md

Line 10 tells config/constitution subcommands to load the full Always Load layer (7 files) on re-run ('Load them only if they already exist'). But line 172 says config 'Loads fab/project/config.yaml only... Does NOT load constitution, memory, or specs', and line 250 says constitution loads two files, no memory/specs. Agents load 1 vs 7 files depending on which rule they follow — a recurring context-cost difference.

**Recommendation**: Rewrite line 10 to cover only the bare bootstrap ('skip Always Load on first run; on re-run follow each behavior section's Context loading note') and let lines 172/250 be authoritative for subcommands.

**Evidence**: fab-setup.md:10 ('Skip the "Always Load" context layer if files don't exist... Load them only if they already exist') vs fab-setup.md:172 ('Loads `fab/project/config.yaml` only... Does NOT load constitution, memory, or specs') and fab-setup.md:250.

**Verifier (high confidence)**: Quotes verified: fab-setup.md:10 ("Load them only if they already exist") vs :172 ("Does NOT load constitution, memory, or specs") and :250 — genuine contradiction; header executes first in reading order. Fix breaks nothing: no other file references the header exception, SPEC-fab-setup.md lacks this text, and _preamble.md:34 already fully exempts /fab-setup, so narrowing line 10 improves consistency. But impact is ~5 extra small file reads, no incorrect behavior — severity high is inflated; medium.


### `f035` [MEDIUM] read-_preamble instruction missing from 9+ skills

**Files**: _preamble.md, docs-reorg-memory.md, docs-reorg-specs.md, fab-help.md, fab-proceed.md, fab-switch.md, git-branch.md, git-pr-review.md, git-pr.md, internal-consistency-check.md, internal-retrospect.md, internal-skill-optimize.md · merged from 5 reports

_preamble states every skill "should begin with" the canonical read instruction. 9 skills lack it entirely (fab-help, git-branch, git-pr, git-pr-review, docs-reorg-memory, docs-reorg-specs, internal-consistency-check, internal-retrospect, internal-skill-optimize); fab-switch reworded the second sentence; fab-proceed drops the blockquote. Skills that never read _preamble silently lose the State Table, naming conventions, and command tables — docs-reorg-memory even cites "_preamble § Memory File Lookup" without instructing the read.

**Recommendation**: Either add the exact canonical line to the deviating skills (at minimum docs-reorg-memory, which cross-references _preamble content), or amend _preamble.md line 11 to name the deliberately preamble-free skill classes; normalize fab-switch.md line 8 to the canonical wording.

**Evidence**: _preamble.md:11-12 (canonical wording); fab-switch.md:8 ("Only after that Read completes, proceed with any Bash calls."); fab-proceed.md:8 (plain text, not blockquote); docs-reorg-memory.md:33 references `_preamble` § Memory File Lookup with no read instruction; git-pr-review.md has no _preamble mention at all

**Verifier (high confidence)**: Confirmed: 9 skills lack the canonical line while specs claim universal loading (docs/specs/skills.md:40, SPEC-preamble.md:5); docs-reorg-memory.md:33 cites _preamble content it never loads. Two corrections: internal-skill-optimize.md:21 DOES instruct the read (procedure step 1), and fab-switch.md:8's wording is a deliberate race fix (commit c1a7c933) — do NOT normalize it. SPEC-fab-help documents "No context loading" as deliberate. Pursue the documented-exemptions branch plus the docs-reorg-memory fix; skip the fab-switch normalization.


### `f037` [MEDIUM] Question budgets contradict across _preamble and planning skills

**Files**: _preamble.md, fab-clarify.md, fab-continue.md, fab-draft.md, fab-new.md · merged from 4 reports

The SRAD Critical Rule says Unresolved questions 'count toward the skill's question budget (max ~3)' including in /fab-new, but fab-new declares 'No fixed question cap' and a conversational mode for 5+ Unresolved. The autonomy table gives fab-continue 'ask top ~3 unresolved' yet 'Interruption budget: 1-2 per stage', while fab-continue.md says 'Budget: 1-2'. Agents cannot deterministically pick a question count.

**Recommendation**: Pick one number per skill: delete '(max ~3)' from the Critical Rule (point to the autonomy table instead), and align the fab-continue row to a single value ('1-2 per stage', matching fab-continue.md:64) by removing 'ask top ~3 unresolved' from the Posture cell.

**Evidence**: _preamble.md:403 'count toward the skill's question budget (max ~3)'; _preamble.md:409-410 fab-continue 'Surface tentative, ask top ~3 unresolved' vs 'Interruption budget: 1-2 per stage'; fab-new.md:106 'No fixed question cap'; fab-continue.md:64 'Budget: 1-2 unresolved questions'.

**Verifier (high confidence)**: All cited lines confirmed verbatim. Cannot refute: no skill has a ~3 budget (fab-new: no cap, fab-continue: 1-2, ff/fff: 0), and _preamble.md:403 contradicts its own table at :410. Stronger evidence: docs/specs/srad.md:226 says budget-override ("even when exhausted"), so "(max ~3)" is spec drift inverting a MUST-ask rule. No Go/SPEC-mirror/constitution coupling breaks; srad.md:282-283 needs the same fix (normal procedure). Medium severity fair.


### `f044` [MEDIUM] fab batch switch branch_prefix contradicts _preamble 'no prefix' branch rule

**Files**: _cli-fab.md, _preamble.md, batch_switch.go · merged from 2 reports

_preamble's Git Branch convention states 'The branch name equals the change folder name directly. No prefix.' But `fab batch switch` applies a configurable `branch_prefix` ({branch_prefix}{folder_name}), implemented in batch_switch.go. A project setting branch_prefix gets branches that /git-branch (following the preamble) will never match.

**Recommendation**: Either document the branch_prefix exception in § Naming Conventions Git Branch (and teach /git-branch to honor it), or remove branch_prefix from batch switch — this is a behavior conflict in the system, not just wording; flag for an explicit decision.

**Evidence**: _preamble.md:140 'No prefix.' vs _cli-fab.md:458 'Branch naming: `{branch_prefix}{folder_name}`' and src/go/fab/cmd/fab/batch_switch.go:145 getBranchPrefix

**Verifier (high confidence)**: All citations verified. Stronger than stated: commit 930ab5e2 (slim-config, Feb 2026) deliberately removed git.branch_prefix system-wide ("no prefix"); Go migration 0f196ea1 reintroduced it only in batch_switch.go. Key is absent from scaffold config and schemas — zombie config. Behavioral risk real: /git-branch would rename the prefixed local branch. Fix = remove prefix from batch_switch.go + _cli-fab.md:458 + kit-architecture.md:140. No constitution conflict. Only mitigation: undocumented key means few projects set it, so medium (not high) is right.


### `f061` [MEDIUM] Acceptance format mandates R# on every item, contradicting baseline Code Quality items

**Files**: _generation.md, plan.md

Step 6 mandates '- [ ] A-{NNN} R#: {outcome}' naming a requirement '(REQUIRED)' for each acceptance item, yet the same step derives Code Quality items from code-quality.md and the template's own baseline items A-007/A-008 carry no R#. An agent must either invent a fake R# or violate the REQUIRED format — two agents will diverge.

**Recommendation**: In _generation.md step 6, scope the R# requirement to requirement-derived categories (Functional Completeness, Behavioral Correctness, Removal Verification, Scenario Coverage, Edge Cases) and define the explicit format for Code Quality/Security/extra-category items (e.g., `A-{NNN} {label}: {outcome}`), matching the template's A-007/A-008 shape.

**Evidence**: _generation.md:122-124: 'Each item follows the format: `- [ ] A-{NNN} R#: {declarative outcome}` — naming the requirement it accepts (REQUIRED)' vs templates/plan.md:193-194: '- [ ] A-007 Pattern consistency: ...' / '- [ ] A-008 No unnecessary duplication: ...' (no R#).

**Verifier (high confidence)**: Confirmed verbatim at _generation.md:122-123 and plan.md:193-194. Contradiction is wider than claimed: plan.md's own header (lines 32-33, 53) and the spec mirror (docs/specs/templates.md:170,196) repeat the R# mandate against the template's R#-less A-007/A-008. Code Quality items are always included, so the unsatisfiable REQUIRED format triggers every plan. Go hook parses only the checkbox prefix, no breakage; constitution silent. Minor quibble: template's Security A-009 does carry R#, so the recommendation slightly overreaches there.


### `f070` [MEDIUM] Cumulative Assumptions output: ff omits it, fff has it, _preamble attributes it to ff

**Files**: _preamble.md, fab-ff.md, fab-fff.md

fab-fff's Output template includes an '## Assumptions (cumulative)' block; fab-ff's Output template has none. _preamble says the cumulative summary rule applies to /fab-ff, and its autonomy table gives fab-ff output as 'Tasks + apply/review/hydrate output' — 'Tasks' is residue of the removed tasks stage. Three sources disagree; two agents render different ff output.

**Recommendation**: Decide whether fab-ff emits the cumulative Assumptions block. If yes, add the block to fab-ff.md's Output template; either way, fix _preamble.md:479 to name the correct skill(s) and replace 'Tasks' in _preamble.md:411's fab-ff Output cell with the actual output description.

**Evidence**: fab-fff.md:127-128 '## Assumptions (cumulative) / {table with Artifact column...}'; fab-ff.md:107-122 Output has no Assumptions section; _preamble.md:479 'For `/fab-ff`, the output summary is **cumulative** across all generated stages'; _preamble.md:411 fab-ff Output: 'Tasks + apply/review/hydrate output'.

**Verifier (high confidence)**: All cited lines verified verbatim. "Tasks" residue confirmed: commit f6ef2aa3 collapsed the tasks stage; old fab-ff generated tasks.md. Decisive: docs/specs/srad.md:284 already has the corrected cell — fab-ff Output = "Cumulative Assumptions summary + apply/review/hydrate output" — so the fix aligns _preamble/fab-ff.md with the spec. No Go code or other skill references break. Mitigation: _preamble 457/479 normatively mandate the block anyway (display drift, not data loss), so medium, not high.


### `f071` [MEDIUM] Rework-cycle choreography under-specified (status commands, re-review dispatch shape, per-cycle repetition)

**Files**: fab-ff.md, fab-fff.md

The loop defines cycle = rework action + re-review, but not the per-cycle mechanics: is 're-run apply' a fresh /fab-continue Apply-Behavior subagent? Is 'fresh sub-agent for re-review' a full Review-Behavior dispatch or direct _review.md inward+outward dispatch? Are finish-apply / fail-review / reset-apply re-run on each subsequent cycle? Two implementations leave different .status.yaml histories.

**Recommendation**: Add one explicit cycle procedure to the rework section (and to the shared helper if extracted): per cycle, run reset/finish commands X and Y, re-dispatch apply via Apply Behavior subagent, re-dispatch review via Review Behavior subagent. State that the fail+reset pair is repeated on every failed re-review.

**Evidence**: fab-ff.md:70 fail+reset stated once before the loop; fab-ff.md:77-79 each heuristic ends 're-run apply, then spawn a **fresh sub-agent** for re-review' with no status commands; fab-fff.md:77 'each cycle = one rework action + one re-review by a fresh sub-agent' — dispatch target and status writes unstated.

**Verifier (high confidence)**: Quotes verified (ff:70, ff:77-79, fff:73-75, fff:77). Go state machine (status.go:37-61) permits divergent valid sequences; each active transition increments stage_metrics.review.iterations (status.go:552), feeding PR meta via prmeta.go — histories genuinely diverge. _review.md:159 defers the loop to orchestrators (with a stale "Step 3" pointer), confirming the gap. Recommendation is additive; no cross-refs break; constitution unaffected. Medium severity stands — core autonomous loop, but pipeline outcomes converge either way.


### `f073` [MEDIUM] Hydrate has no error path for unchecked Tasks/Acceptance items

**Files**: fab-continue.md

Hydrate's precondition says all ## Tasks and ## Acceptance items MUST be [x], and Step 1 re-validates, but no action is specified on violation (the review-done precondition has 'If not: STOP'; this one doesn't), and the Error Handling table has no row. Review can pass while leaving acceptance boxes unchecked, so this path is reachable; agents will diverge between stopping, self-checking boxes, or proceeding.

**Recommendation**: In Hydrate Behavior Preconditions, attach an explicit action to the checkbox precondition: either 'If not: STOP with "{N} acceptance items unchecked — re-run review"' or a deterministic reconcile rule (verify each unchecked item against the diff, mark [x] when met). Add the matching row to the Error Handling table.

**Evidence**: fab-continue.md:166-167 'All items in `plan.md` `## Tasks` and `## Acceptance` MUST be `[x]`' (no action); :171 'Final validation: all ... are `[x]`'; Error Handling :193-201 only covers 'Review not passed for hydrate'. _review.md:42 shows the inward sub-agent may leave items `[ ]` with reason when not met.

**Verifier (high confidence)**: Verified: fab-continue.md:166 has "If not: STOP", :167 has no action; Error Handling lacks the row; _review.md:42 allows unchecked items. Path is empirically reachable — user memory documents the oy0k run (review passed, 25 boxes unchecked, hydrate improvised reconcile). No cross-reference breakage; ff/fff inherit by delegation; spec mirror update is routine. Caveat: prefer the reconcile-rule option over STOP — STOP would stall autonomous pipelines and contradict documented practice. Severity medium is correct.


### `f083` [MEDIUM] Next Steps Reference hard-codes 'initialized' state and omits a Next: line for updates

**Files**: fab-setup.md

Line 494 fixes the post-migrations state to `initialized`, but migrations can run with an active change (actual state intake/apply/etc.), contradicting _preamble's 'Look up the state reached' procedure. Line 492 ('no further action needed') gives config/constitution updates no Next: line, contradicting _preamble's 'Every skill MUST end its output with a Next: line'.

**Recommendation**: Change line 494 to 'derive from actual state (initialized if no active change; otherwise the active change's stage)'. Replace line 492 with an explicit Next: line derived from current state.

**Evidence**: fab-setup.md:492-494 ('After config/constitution update: (no further action needed...)'; 'After migrations: state = `initialized`') vs _preamble.md:265 ('Every skill MUST end its output with a `Next:` line derived from the State Table... Look up the state reached').

**Verifier (high confidence)**: Could not refute. Lines 492/494 confirmed verbatim; _preamble.md:265 mandates a state-derived Next: line and :286 defines initialized as no-active-change. fab-setup.md:482 and :11/:327 confirm migrations run with an active change allowed, so hard-coded `initialized` misdirects (/fab-new mid-change after kit upgrade). No cross-refs break: section text exists only in fab-setup.md; SPEC-fab-setup.md lacks it; no Go code parses Next:. fab-operator's documented exemption is no precedent — fab-setup claims state-table derivation at line 488.


### `f096` [MEDIUM] Same status-commit procedure has opposite push-failure semantics in git-pr vs git-pr-review

**Files**: git-pr-review.md, git-pr.md

git-pr Step 4c STOPs on a status-commit push failure even though the PR was already created and ship already finished (Step 4b runs first) — the skill reports failure after shipping succeeded. git-pr-review Step 6.5 deliberately softens the identical procedure to best-effort, citing 'terminal stage'. The rationale applies equally to git-pr's post-success bookkeeping.

**Recommendation**: Align git-pr Step 4c to git-pr-review 6.5's best-effort push (report, retain local commit, don't STOP). This is a behavior change — making it explicit: post-PR bookkeeping should never fail the shipped state.

**Evidence**: git-pr.md:247 ('If commit or push fails → report the error and STOP') vs git-pr-review.md:205 ('This softens git-pr's fail-fast push specifically for the terminal stage')

**Verifier (high confidence)**: Quotes verified (git-pr.md:247, git-pr-review.md:205); step order 3c→4b→4c confirmed. Best counter: divergence is documented as deliberate ("terminal stage") with a silent retry path (git-pr.md:125). Outweighed by fab-fff.md:101, which assumes git-pr failure leaves ship active — false after 4c (4b marked it done), so a transient push failure aborts fab-fff and skips review-pr. No constitution conflict; only normal mirror updates (SPEC-git-pr.md:67-70, git-pr Rules, git-pr-review.md:205 parenthetical).


### `f106` [MEDIUM] reorg-memory legacy hand-edit fallback contradicts its own generated-index rules

**Files**: docs-hydrate-memory.md, docs-reorg-memory.md

Error Handling row :160 says if fab memory-index is unavailable, 'fall back to hand-updating affected index.md files' — contradicting :126 'Do not hand-edit index files' and Key Properties :176 'Indexes hand-edited? No'. Per Constitution V the kit is version-pinned to the binary, so 'older binary' cannot occur. Twin asymmetry: hydrate-memory defines no memory-index failure row at all.

**Recommendation**: Delete the ':160 fab memory-index unavailable' fallback row (the scenario is impossible under version-pinned deployment, and it licenses behavior two other sections forbid). If a failure row is wanted, make it 'abort and report' and add the same row to docs-hydrate-memory's Error Handling table.

**Evidence**: docs-reorg-memory.md:160 'Warn; fall back to hand-updating affected `index.md` files (legacy path)' vs :126 'Do **not** hand-edit index files — they are generated' and :176 '| Indexes hand-edited? | No'

**Verifier (high confidence)**: Confirmed: :160 fallback row contradicts :126 and :176. SPEC mirror (SPEC-docs-reorg-memory.md) already says "never hand-edited" with no fallback row, so deleting it re-aligns skill with SPEC. Constitution V pins kit to binary via fab sync, making "older binary" impossible; memory_index.go exists. No cross-references break (sole grep hit is the cited line). Hydrate twin lacks the row, confirming asymmetry. Weak counter: a downgraded-binary-without-resync edge case, but unsupported deployment. Medium severity stands.


### `g2-3` [MEDIUM] fab memory-index unavailability handled in docs-reorg-memory but undefined at the other two call sites, which forbid the only manual fallback

**Files**: docs-hydrate-memory.md, docs-reorg-memory.md, fab-continue.md

docs-reorg-memory.md:160 defines an older-binary fallback (warn, hand-update indexes, suggest upgrade). docs-hydrate-memory.md:83/153 and fab-continue.md:174 (hydrate, which runs unattended inside ff/fff) run the same command while mandating 'never hand-edit index rows' with no failure row — contradictory instructions if the command is absent or fails, leaving the agent to improvise.

**Recommendation**: Pick one behavior and state it at all three sites: either copy docs-reorg-memory.md:160's fallback row into docs-hydrate-memory's Error Handling table (lines 180-188) and fab-continue's Hydrate Behavior step 4, or replace it everywhere with 'warn, skip index regeneration, tell user to upgrade fab' (preserving the never-hand-edit invariant).

**Evidence**: docs-reorg-memory.md:160 '| `fab memory-index` unavailable (older binary) | Warn; fall back to hand-updating affected `index.md` files (legacy path)...' vs fab-continue.md:174 'run `fab memory-index` to regenerate ... — never hand-edit index rows' with no failure condition in its Error Handling table (191-201).

**Verifier (high confidence)**: All quotes verified: reorg:160 fallback row exists; hydrate:83/153 and fab-continue:174 forbid hand-edits with no memory-index failure row (tables 180-188, 191-201). Skew is real (project-pinned fab_version, per-repo skill sync). No cross-refs break: "older binary" appears nowhere else; SPEC mirrors lack error tables; constitution silent. Prefer the recommendation's second option (warn/skip/upgrade everywhere) — the hand-edit fallback itself contradicts the single-writer invariant in docs/specs/templates.md:376,558.


### `f038` [LOW] Documented preflight field lists omit id/display_stage/display_state that skills use

**Files**: _cli-fab.md, _preamble.md, fab-continue.md, fab-status.md, preflight.go · merged from 4 reports

fab-continue Step 1 dispatches on preflight's `display_state` and fab-status renders `display_stage`/`display_state`, but _preamble §2 step 3 lists only id/name/change_dir/stage/progress/plan/confidence, and _cli-fab's preflight section also omits `id`. The binary emits all nine fields. Agents are told to parse fields the contract never declares.

**Recommendation**: Add `display_stage` and `display_state` to the field list in _preamble.md §2 step 3 (line 56) and add `id`, `display_stage`, `display_state` to _cli-fab.md's `fab preflight (extended)` section (line 92).

**Evidence**: _preamble.md:56; _cli-fab.md:92; fab-continue.md:40 ("Dispatch on preflight's derived `stage` and `display_state`"); fab-status.md:45; preflight.go FormatYAML emits `display_stage:`/`display_state:` (preflight.go:101-102)

**Verifier (high confidence)**: All cited lines verified verbatim; FormatYAML (preflight.go:97-102) emits all nine fields while _preamble.md:56 and _cli-fab.md:92 declare seven/six. Fix is additive, breaks no cross-references, and the constitution is silent on this. Severity inflated: consuming skills (fab-continue.md:40, fab-status.md:45) name the fields at point of use and agents parse the actual stdout YAML, so no behavioral failure — pure contract drift. Recommendation misses _preamble.md:247, which has the same omission.


### `f051` [LOW] Artifact Markers 'Placed by' enumeration omits fab-draft and fab-fff

**Files**: _preamble.md, fab-draft.md

Line 444 says `<!-- assumed: -->` markers are placed by 'All planning skills (fab-new, fab-continue, fab-ff)'. fab-draft and fab-fff also generate intakes/plans via _generation and produce Assumptions tables. An agent following the enumeration literally would skip markers in drafted intakes, so /fab-clarify's marker scan would miss their Tentative assumptions.

**Recommendation**: Change the 'Placed by' cell to 'all planning skills' without parenthetical enumeration, or enumerate all five (fab-new, fab-draft, fab-continue, fab-ff, fab-fff). Same fix wherever planning skills are enumerated (the Autonomy Levels table at 405-413 also omits fab-draft).

**Evidence**: _preamble.md:444 'All planning skills (fab-new, fab-continue, fab-ff)'; fab-draft.md:4 'helpers: [_generation]' and fab-draft.md:135 emits an '## Assumptions' table

**Verifier (high confidence)**: Claim confirmed: _preamble.md:444 enumerates only fab-new/fab-continue/fab-ff; fab-draft generates intakes via _generation, applies SRAD (Step 8), and routes to /fab-clarify, and fab-fff mirrors fab-ff exactly. Fix is one cell plus the docs/specs/srad.md:93 mirror; nothing else references the string. Severity downgraded: the Confidence Grades table (line 398) unconditionally maps Tentative to the marker, so agents likely place markers regardless — impact is documentation inconsistency, not probable behavior loss.


### `f108` [LOW] Family-wide closing-section divergence; hydrate-specs has no Next: guidance

**Files**: docs-hydrate-memory.md, docs-hydrate-specs.md, docs-reorg-memory.md, docs-reorg-specs.md

hydrate-memory ends with an Idempotency section and a Next: line (:174-176, :192) but no Key Properties; the other three end with Key Properties and no Next: line. _preamble:265 says 'Every skill MUST end its output with a Next: line' — hydrate-specs loads the preamble yet gives no Next: guidance, so the MUST is unmet or improvised. 'Safe to re-run' is also stated three times in hydrate-memory (:3, :19, :176).

**Recommendation**: Standardize the family: give hydrate-memory a Key Properties table (fold the Idempotency prose into its 'Idempotent?' row, deleting :174-176), and append a Next: line to docs-hydrate-specs (state-table derived, like docs-hydrate-memory:192). Reorg skills follow whatever the preamble-loading decision (separate finding) lands on.

**Evidence**: docs-hydrate-memory.md:174-176 (§Idempotency), :192 'Next: {per state table — initialized}'; docs-hydrate-specs.md:97-105 Key Properties with no Next:; _preamble.md:265 'Every skill MUST end its output with a `Next:` line'

**Verifier (high confidence)**: All cited facts verified line-for-line. No cross-references break: SPEC mirrors are flow summaries lacking both sections; Go code references names only. Counter-evidence: fab-setup keeps Idempotency AND Key Properties (coexistence precedented), and _preamble's generic Lookup Procedure (:292-296) lets agents derive Next: at runtime, so the MUST is dischargeable — a consistency gap, not a hard violation. fab-proceed/internal-skill-optimize share the omission. Fix is cheap and preserves behavior if Idempotency prose is folded, not deleted. Severity medium is inflated; low.


### `f109` [LOW] consistency-check subagents skip the Standard Subagent Context and diverge from preamble dispatch convention

**Files**: _preamble.md, internal-consistency-check.md

Agents are spawned via 'Task tool, subagent_type: Explore' with prompts carrying only source_paths — no instruction to read config/constitution, which _preamble marks MUST for every subagent prompt. Without constitution VI (specs are pre-implementation design intent), Agent 1's 'missing implementations' category invites false positives on legitimately spec-ahead designs. The lean 3-agent fan-out is good but fragile.

**Recommendation**: Add the standard 5-file context instruction to the three agent prompts; either align dispatch wording with _preamble's 'Agent tool (general-purpose)' or document why read-only Explore agents are intentionally exempt.

**Evidence**: internal-consistency-check.md:25 ('Task tool, subagent_type: `Explore`'), :29 ('missing implementations') vs _preamble.md:339 and :354 'Every subagent prompt MUST instruct the subagent to read the following project files'; constitution.md:21.

**Verifier (high confidence)**: Citations accurate, but _preamble's MUST is under "Subagent Dispatch (Orchestrator Skills)" (line 337-339), scoped to sub-skill dispatch by fab-ff/fab-fff; internal-consistency-check never loads _preamble and nothing in src/, docs/specs/, or Go references it — no formal violation, no cross-refs break. Still worth adding constitution/config reads to the three Explore prompts (cheap; cuts spec-ahead false positives per constitution VI). Do NOT switch Explore to general-purpose (loses read-only safety); take the document-exemption option. No SPEC mirror exists for internal-* skills.


### `f120` [LOW] Two canonical command lines for autopilot worktree creation: cherry-pick model vs _cli-external's `--base` example

**Files**: _cli-external.md, fab-operator.md

_cli-external's autopilot-respawn example passes `wt create ... --base <prev-change>` (branch from prior change). fab-operator never passes `--base` to wt create — dependencies are satisfied by cherry-pick (§6), and 'implicit `--base` chaining' is just depends_on terminology. Two agents could build dependent worktrees with different histories (branched-from-dep vs main+squashed-cherry-pick), producing different PR diffs.

**Recommendation**: Pick one: update _cli-external.md:39's example to drop `--base` (matching §6 Autopilot step 1 '`--reuse` for respawns'), or state in §6 that respawns recreate from the previous branch via `--base`. Also rename 'implicit `--base` chaining' to 'implicit depends_on chaining' to stop overloading the flag name.

**Evidence**: _cli-external.md:39 '**Example — autopilot respawn**: `wt create --non-interactive --reuse --worktree-name <name> <branch> --base <prev-change>`' vs fab-operator.md:507 'create worktree in it (`--reuse` for respawns)' and :388 spawn step 2 with no `--base`; :499 'Implicit `--base` chaining by default: every change after the first gets `depends_on: ...`'.

**Verifier (high confidence)**: Confirmed: _cli-external.md:35/39's wt-create --base text is a stale operator4 leftover (added 315388d9; operator7 3df2e47a switched to cherry-pick and redefined --base as a depends_on directive). Fix is cheap, breaks nothing. But severity is inflated: wt's --base only applies when creating a new branch — respawns with --reuse/existing branch ignore it — and ancestor-pruning makes even the worst case content-equivalent vs main. Real doc inconsistency, minimal behavioral harm.


### `g1-7` [LOW] Idempotency declaration convention is missing exactly on the riskiest mutators (fab-new, fab-draft, git-pr)

**Files**: fab-draft.md, fab-new.md, git-pr.md

Most skills declare a re-run contract — Key Properties 'Idempotent?' rows (fab-continue.md:209, fab-archive.md:108, git-branch.md:166, fab-operator.md:639, etc.) or Idempotency sections (fab-setup.md:466, docs-hydrate-memory.md:174). fab-new, fab-draft, and git-pr declare nothing; git-pr only implies it via 'Skip steps that are already done' (git-pr.md:268). git-pr is in fact re-run-safe (already-shipped path, idempotent add-pr per _cli-fab.md:68, `git diff --cached --quiet` guard at git-pr.md:246) — the contract just isn't stated.

**Recommendation**: Add the standard Key Properties 'Idempotent?' row to git-pr.md stating its actual contract (re-run after ship is a no-op via the lines 117-125 path), and to fab-new.md/fab-draft.md once finding 2's re-run behavior is decided.

**Evidence**: grep for 'Idempot' matches 16 skill files but none of fab-new.md, fab-draft.md, git-pr.md; git-pr.md:268 'Skip steps that are already done (no uncommitted → skip commit, PR exists → skip create)' is the only implicit statement.

**Verifier (high confidence)**: All cited lines verified; 16 files match "Idempot", the three named files don't. But "exactly the riskiest" is overstated: fab-ff/fab-fff and internal-* also lack it, and glossary.md:36 already documents git-pr's contract. The three files have no Key Properties section, so the fix adds a section, not a row. Constitution III mandates idempotency anyway. git-pr half is cheap and actionable; fab-new/fab-draft half is deferred on finding 2.


### `g3-6` [LOW] fab-clarify updates Assumptions rows but not the template-mandated summary count line

**Files**: fab-clarify.md, intake.md

The intake template mandates a trailing count line '{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved)' (intake.md:71). fab-clarify's Artifact Update (fab-clarify.md:109-117) and Step 4 (:147) reclassify table rows to Certain but never instruct refreshing that line, so it goes stale after every clarify session. Score is unaffected (score.go:228-285 counts table rows only), but the persisted artifact contradicts itself.

**Recommendation**: Add one sentence to fab-clarify.md Artifact Update (after :117) and Step 4: 'Recompute the summary count line beneath the table to match the updated grades.'

**Evidence**: intake.md:71 '{N} assumptions ({Ce} certain, {Co} confident, {T} tentative, {U} unresolved).'; fab-clarify.md:147 'Reclassify resolved entry to Certain in ## Assumptions table' — no mention of the count line anywhere in fab-clarify.md.

**Verifier (high confidence)**: Verified: intake.md:71 mandates the count line; fab-clarify.md (Artifact Update :107-117, Step 4 :147) never mentions it; score.go countGrades (:228-294) parses only |-rows, so score is unaffected as claimed. Strongest counter: nothing parses the line and _preamble.md:471 frames it as display output — but the template persists it, and the intake table is the declared state-transfer mechanism, so staleness can mislead downstream agents. One-sentence fix plus SPEC-fab-clarify.md mirror update; no cross-references break. Severity low is correct.


### `g4-5` [LOW] fab-fff mischaracterizes git-pr-review's no-reviews behavior, omitting the Copilot request and 10-minute poll

**Files**: fab-fff.md, git-pr-review.md

fab-fff Step 5 tells the orchestrator the subagent "prints a stop message and completes" when no reviews exist. The actual git-pr-review (and its own description) requests a Copilot review and polls up to 10 minutes, possibly then processing comments. The orchestrator's expectation is stale — it may misreport the stage or treat a legitimately polling subagent as hung.

**Recommendation**: Update fab-fff.md:109 to match the callee: "If no reviews exist, the subagent requests a Copilot review and polls up to 10 minutes; if none appears it completes as a no-op." Verify fab-fff.md:113's no-op claim still aligns (it does — git-pr-review.md:188).

**Evidence**: fab-fff.md:109: "If no reviews exist, prints a stop message and completes" vs git-pr-review.md:3/9: "requests an automated Copilot review and polls for up to 10 minutes"

**Verifier (high confidence)**: Verified: fab-fff.md:109 quote is exact; git-pr-review.md:70-98 requests Copilot and polls 20x30s, then may process comments. No cross-refs break; SPEC-fab-fff.md lacks the stale sentence. But fab-fff's actual outcome handling (109/111/113) already matches the callee (no-op→done per git-pr-review.md:188); only descriptive text is stale, and the "hung subagent" harm is speculative (fab-fff has no timeout). Downgrade to low. Bonus: docs/specs/skills.md:785 still describes the removed cascade.



## 6. Developer experience (14)

### `f034` [MEDIUM] Mandatory Next: line convention violated by many skills with no declared exceptions

**Files**: _preamble.md, docs-hydrate-specs.md, fab-discuss.md, fab-draft.md, fab-help.md, fab-new.md, fab-operator.md, fab-status.md, fab-switch.md, git-branch.md, git-pr-review.md, git-pr.md · merged from 5 reports

_preamble says every skill MUST end with a Next: line. git-pr ends with "Shipped." (state reached: review-pr active → Next: /git-pr-review), git-pr-review/git-branch/fab-help/docs-hydrate-specs/docs-reorg-*/internal-* output none (only fab-discuss and fab-operator document their opt-out). Separately, fab-new and fab-draft hardcode Next: lists that omit /fab-proceed, which the State Table's intake row includes.

**Recommendation**: Add state-derived Next: lines to git-pr (after "Shipped.") and git-pr-review (pass → /fab-archive, fail → /git-pr-review) at minimum, since they sit mid-pipeline; add /fab-proceed to fab-new.md:231 and fab-draft.md:158; or amend _preamble.md:265 to scope the MUST to pipeline-state skills.

**Evidence**: _preamble.md:265 ("Every skill MUST end its output with a `Next:` line"); git-pr.md:256-260; git-pr-review.md (no Next anywhere); fab-new.md:231 vs _preamble.md:275 (intake row includes /fab-proceed)

**Verifier (high confidence)**: All cited facts verified: _preamble.md:265 MUST (also global in docs/specs/skills.md:90, glossary.md:118); git-pr.md:259 ends "Shipped." with no Next despite finishing ship; git-pr-review/git-branch/fab-help/docs-hydrate-specs/docs-reorg-*/internal-* have zero Next; only fab-discuss:91/104 and fab-operator:641 opt out; fab-new.md:231 and fab-draft.md:158 omit /fab-proceed vs _preamble.md:275, and fab-new.md:209 self-contradicts line 231. No parser depends on "Shipped." or Next lines. Minor: fab-status/fab-switch in files list do emit Next.


### `f068` [MEDIUM] fab-clarify emits the Next: line before recomputing confidence, and never displays the new score

**Files**: fab-clarify.md

Step 6 (Coverage Summary) ends with the Next: line; Step 7 then runs fab score. This violates the preamble rule that every skill 'MUST end its output with a Next: line', and the recomputed score — the main payoff of clarifying — is persisted but never shown to the user (fab-new Step 7 displays it; clarify does not).

**Recommendation**: Reorder: make recompute Step 6 (and add 'display the score line from stdout, format Confidence: {score} / 5.0 ({N} decisions)'), Coverage Summary + Next: line Step 7, keeping Step 8 (no advance) last.

**Evidence**: fab-clarify.md:154-167 (summary with 'Next:' at :166) precedes :169-171 (Step 7 recompute); _preamble.md:265 'Every skill MUST end its output with a Next: line'

**Verifier (high confidence)**: Confirmed: fab-clarify.md:166 Next: line precedes Step 7 recompute (:169-171), which only persists — no display — while fab-new.md:92-100 displays the score; _preamble.md:265 mandates Next: last. Reorder breaks nothing: only SPEC-fab-clarify.md mirror plus a one-line renumber in SPEC-hooks.md:220; no Go/skill code references step numbers. Weakest point: Step 7 is a Bash call, so the Next:-rule breach is technical, but the hidden score (the gate-relevant payoff) stands. Medium is fair.


### `f091` [MEDIUM] fab-status — the most output-centric skill — has no canonical Output template

**Files**: fab-status.md

fab-switch and fab-archive each show a literal output block; fab-status describes its status block in one 9-item prose sentence ('version header, change name, branch, stage with state qualifier, next action, progress table...'). Two agents will produce different layouts, label spellings, and orderings for the kit's primary orientation display.

**Recommendation**: Add an `## Output` section with the exact rendered template (like fab-switch.md:84-93), including the progress-table shape, Impact line, and warning placement; cut the prose enumeration in the line-46 bullet down to pointers into that template.

**Evidence**: fab-status.md:46: "Renders the full status block: version header, change name, branch, stage with state qualifier (out of 6 total stages), next action, progress table with symbols ... plan counts ... confidence score, optional Impact line ... version drift warning" — no literal template anywhere in the file

**Verifier (high confidence)**: Confirmed: fab-status.md has no Output section; line 46 prose matches; fab-switch.md:82-93 has a literal template. Strengthens the case: fab-status.md:40 says the skill itself formats (fab-switch's template is Go-rendered stdout). No parsers of the output; only SPEC mirror update needed. Counter-evidence: Impact line, warning texts, confidence/stage/Next formats are already pinned exactly — variance is mainly progress-table shape, version header, ordering, so severity is low-end medium but defensible.


### `f092` [MEDIUM] git-pr-review: 10-minute poll loop mechanics under-specified

**Files**: git-pr-review.md

'Poll every 30 seconds, up to 20 attempts' gives no execution strategy: a single Bash loop (20x30s = 600s) exceeds the default 120s Bash tool timeout and would be killed mid-wait, likely misread as failure. Also undefined: whether the first poll is immediate, and what a transiently failing gh pr view poll does (abort? count as attempt?).

**Recommendation**: In Step 2 Phase 2, specify: run each attempt as a separate Bash call `sleep 30 && gh pr view ...` (or one loop with an explicit >=620000ms timeout), and 'a failed poll counts as an attempt; do not abort'.

**Evidence**: git-pr-review.md:90-95 ('Poll every 30 seconds, up to 20 attempts' — no sleep mechanism, no per-poll error path)

**Verifier (high confidence)**: Verified: git-pr-review.md:90 says only "Poll every 30 seconds, up to 20 attempts" — no sleep mechanism or per-poll error path. Real impact: the Rules' "fail fast" (line 228) means a transient poll failure can abort and mark the stage failed; a single-loop impl dies at the 120s Bash default. Additive fix; only the SPEC mirror needs updating (normal). Caveats: rec's ">=620000ms" exceeds the 600000ms Bash max, and allowed-tools lacks sleep — fix should add Bash(sleep:*).


### `f112` [MEDIUM] Repo-only internal skills ship to every user project with no environment guard

**Files**: internal-retrospect.md, internal-skill-optimize.md, kit-architecture.md

Sync deploys all skills/*.md, so user projects receive internal-skill-optimize, yet it hardcodes src/kit/skills/ paths that exist only in the fab-kit repo — single mode STOPs with a misleading 'Skill not found', and batch mode has no empty-set path. internal-retrospect likewise suggests knowledge belongs in _preamble.md, which downstream users cannot edit.

**Recommendation**: Add a Pre-flight guard ('STOP: this skill runs only in the fab-kit repo' when src/kit/skills/ is absent) or exclude internal-* from sync deployment; reword internal-retrospect line 19's destination list for user-project context.

**Evidence**: kit-architecture.md:166 'All `*.md` files in `$(fab kit-path)/skills/` are deployed'; internal-skill-optimize.md:14,23; internal-retrospect.md:19.

**Verifier (high confidence)**: All cited lines verified; sync.go listSkills has no internal-* filter, and constitution line 18 confirms src/kit/ is absent downstream, so the skill always misfires there. No cross-refs break; fabhelp.go already hides internal-* (precedent). Recommendation aligns with constitution. Caveat: prefer the Pre-flight guard over sync exclusion — exclusion would also drop internal-retrospect/internal-consistency-check, which are usable downstream. internal-retrospect:19 claim slightly overstated (diagnostic list, not destination), but lines 37-38 cite nonexistent /meta: skills, reinforcing the finding.


### `f124` [MEDIUM] Bare 'No active change.' error breaks _preamble's promise that preflight stderr contains a suggested fix

**Files**: _preamble.md, resolve.go

Skills are told to surface preflight stderr verbatim because 'it contains the specific error and suggested fix'. For the most common new-user failure (no active change, zero candidates) the message is just 'No active change.' — no /fab-new or /fab-switch pointer. Only the multiple-candidates variant includes guidance. Constitution rule: CLI changes must also update _cli-fab.md.

**Recommendation**: Append guidance to the zero-candidate and symlink-missing errors in resolve.go (e.g., 'No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.'), mirroring fab-switch.md:32's wording.

**Evidence**: resolve.go:139 and resolve.go:159-160 return bare 'No active change.'; resolve.go:161 shows the multi-change variant does include '— use /fab-switch'; _preamble.md:55 'surface the stderr message to the user (it contains the specific error and suggested fix)'.

**Verifier (high confidence)**: Verified: resolve.go:139/:159 return bare "No active change."; :161 includes /fab-switch hint; _preamble.md:55 promises a suggested fix; preflight.go:46-49 passes resolve errors through unwrapped while its own errors say "Run /fab-setup." No test asserts the exact string (change_test.go:287 is the unrelated SwitchNone path, Contains-based). Fix requires _cli-fab.md error-table + test + SPEC mirror updates — normal procedure. Minor nit: fab-switch.md:32 suggests /fab-draft, not /fab-switch, as second option.


### `f125` [MEDIUM] No skill-authoring checklist for kit developers; conventions scattered across five places and already drifting

**Files**: fabhelp.go, skills, skills.md

Adding a skill correctly requires: preamble-read line + helpers frontmatter (_preamble), Next: convention, a SPEC-*.md (constitution rule), the helper-mapping table in skills.md, and fabhelp.go's skillToGroupMap. No checklist or template exists, and drift proves it: 6 skills have no SPEC file (3 internal-*, _generation, _cli-fab, _cli-external) and 4 skills are unmapped in fab-help.

**Recommendation**: Add a 'New Skill Checklist' section to docs/specs/skills.md (or a SPEC-template.md in docs/specs/skills/) enumerating: frontmatter fields, preamble line, helpers declaration, Next: line, Error Handling + Key Properties tables, SPEC file, skills.md mapping row, help grouping. Create the 6 missing SPEC files or amend the constitution rule to scope which skills require SPECs.

**Evidence**: Constitution.md:32 'Changes to skill files ... MUST update the corresponding docs/specs/skills/SPEC-*.md'; ls docs/specs/skills shows no SPEC for internal-consistency-check, internal-retrospect, internal-skill-optimize, _generation, _cli-fab, _cli-external; fabhelp.go:23-41 map omits fab-proceed/fab-operator/git-branch/git-pr-review.

**Verifier (high confidence)**: All claims verified: constitution.md:32 quote exact; 6 SPECs missing (SPEC-_review.md proves partials aren't exempt); fabhelp.go:23-41 omits the 4 named skills; no checklist exists in docs/, CONTRIBUTING.md, or _preamble.md. Bonus drift: skills.md:49 says "19 skills", actual 18. Only counter-evidence: fab-help's "Other" fallback (fabhelp.go:136-149) keeps unmapped skills visible, so half the drift is cosmetic. Recommendation is additive, breaks nothing. Medium stands.


### `g4-6` [MEDIUM] fab-proceed description omits its zero-argument posture; arguments are silently ignored

**Files**: fab-proceed.md

Every other pipeline skill accepts `<change-name>`, so an agent choosing from descriptions may invoke `/fab-proceed <change>` to run a named change. The body silently ignores all arguments (fab-proceed.md:24) and instead resolves the active change or conversation context — potentially running a different change than the user named, with no warning.

**Recommendation**: Add the constraint to the frontmatter description, e.g. "...then delegates to fab-fff. Takes no arguments — infers everything from conversation; use /fab-fff <change> to target a named change." Optionally have the body emit a one-line notice when arguments are dropped instead of pure silence (behavior change — flagging explicitly).

**Evidence**: fab-proceed.md:24: "None. `/fab-proceed` does not accept arguments or flags. Any arguments passed are silently ignored." — absent from the description at line 3

**Verifier (high confidence)**: Quote verified verbatim at fab-proceed.md:24; description (line 3) omits it. Premise holds: fab-fff/ff/continue/switch all take <change-name>. Strongest pro-evidence: SPEC-fab-proceed.md:5 summary already says "No arguments, no flags" — description just needs alignment. No cross-references break; constitution silent on descriptions. Mitigation: body corrects agents post-invocation, but worst case routes autonomous fff (impl→ship→PR) at the wrong change, so medium stands. Optional notice part is a flagged behavior change.


### `g1-8` [LOW] Reset-to-apply with all tasks checked is a silent pass-through, not a re-run

**Files**: fab-continue.md

Reset Flow step 4 (fab-continue.md:185) says execution-stage resets 're-run' but task checkboxes are NOT reset. With all tasks `[x]`, Task Execution step 4 (fab-continue.md:124) immediately re-finishes apply — `/fab-continue apply` silently advances to review having redone nothing. The plan.md-deletion escape hatch is documented for plan regeneration, but nothing tells the user to uncheck tasks to actually re-execute work.

**Recommendation**: Append one sentence to fab-continue.md:185: 'to re-execute completed tasks, uncheck them in plan.md `## Tasks` first (with a `<!-- rework: reason -->` note), otherwise the reset re-finishes apply immediately.'

**Evidence**: fab-continue.md:185 'Execution stages (apply onward) re-run (task checkboxes NOT reset; `plan.md` is also preserved...)'; fab-continue.md:124 'If all checked: run `fab status finish <change> apply fab-continue` (auto-activates review). Stop.'

**Verifier (high confidence)**: Mechanics confirmed verbatim (fab-continue.md:185, :124): reset-to-apply with all tasks [x] re-finishes apply immediately. Slightly overstated — the uncheck/<!-- rework: --> mechanism IS documented at fab-continue.md:154 and skills.md:293 says "re-running unchecked tasks" — but not in the Reset Flow section, and "re-run" at :185 misleads. Recommendation is one additive sentence, reuses existing convention (fab-ff.md:77, fab-fff.md:73), breaks no cross-refs, constitution-safe (behavior is intentional idempotency; only wording fixed). Severity low is correct.


### `g2-5` [LOW] First-touch skills give no usable path when the fab binary itself is not installed

**Files**: fab-help.md, fab-setup.md

The two skills a brand-new user hits first assume fab exists: fab-setup's pre-flight STOP prescribes 'Run fab sync' — unrunnable when the failure is command-not-found — and its Phase 0 gate is `fab doctor`, a fab subcommand that cannot diagnose fab's own absence. fab-help.md:22 runs `fab fab-help` with no failure handling at all.

**Recommendation**: In fab-setup.md Pre-flight Check (lines 29-34), branch the STOP message: command-not-found → 'fab CLI not installed — brew install fab-kit' (per constitution Principle V); other non-zero → current message. Add one line to fab-help.md Behavior: if `fab fab-help` fails, print the same install hint instead of the raw shell error.

**Evidence**: fab-setup.md:34 'If either check fails, STOP immediately. Output: `Kit not found. Run 'fab sync' or 'fab upgrade-repo' to populate the cache.`' — fab sync requires the missing binary; fab-help.md:21-22 'fab log command "fab-help" 2>/dev/null || true / fab fab-help' with no error row anywhere in the file.

**Verifier (medium confidence)**: Quotes verified verbatim (fab-setup.md:31-34, fab-help.md:21-22; no error handling in fab-help). But "first-touch" framing is overstated: skills reach .claude/skills/ only via fab sync (sync.go:519) and scaffold fragment-.gitignore ignores /.claude (repo tracks 0 .claude files), so no-binary users normally lack the skills entirely; scenario needs uninstall/PATH break or non-default committed .claude. Fix is two lines, breaks nothing ("Kit not found" string exists only in fab-setup.md), constitution cites brew install fab-kit. Severity low is correct.


### `g4-10` [LOW] git-branch description omits the standalone-branch fallback for unmatched arguments

**Files**: git-branch.md

Description says the branch matches "the active (or specified) change", but when an explicit argument matches no change, the body creates a branch with the literal argument as its name (git-branch.md:56-62). A typo'd change name therefore creates a stray branch; conversely, the skill's usefulness for arbitrary branch creation is hidden.

**Recommendation**: Append to the frontmatter description: "Unmatched explicit names fall back to a standalone branch with that literal name." (The body already prints a notice; this only surfaces it at selection time.)

**Evidence**: git-branch.md:3 vs git-branch.md:56-58: "enter **standalone fallback** — use the raw argument as a literal branch name"

**Verifier (high confidence)**: Verified: git-branch.md:3 omits fallback; lines 56-62 confirm standalone fallback. Strongest counter-evidence: body line 11 already documents it and runtime prints a notice, so impact is selection-time only. Still worthDoing: SPEC-git-branch.md:5 already includes the fallback in its one-liner, so the fix aligns description with the spec; no cross-references break; no constitution conflict. Severity low stands. Optional: skills.md:596 purpose line also omits it.


### `g4-4` [LOW] docs-reorg-memory description hides its rebalancer role — indistinguishable from docs-reorg-specs

**Files**: docs-reorg-memory.md, docs-reorg-specs.md

The two reorg descriptions are word-for-word parallel, but docs-reorg-memory's body additionally defines the memory rebalancer: shape diagnosis, split/merge/flatten of domains, link rewriting, fab memory-index regeneration (docs-reorg-memory.md:14, 174). Requests like "split this over-wide memory domain" or "rebalance memory" have no description-level match, risking non-invocation of the only skill that does this.

**Recommendation**: Extend docs-reorg-memory's frontmatter description, e.g. append "Also the memory rebalancer — diagnoses folder shape and splits/merges/flattens domains, rewriting links, on approval." Keep docs-reorg-specs unchanged so the pair stays distinguishable.

**Evidence**: docs-reorg-memory.md:3 description is structurally identical to docs-reorg-specs.md:3; docs-reorg-memory.md:14: "This is also the **memory rebalancer**"; :174: "supersedes any separate `/fab-rebalance-memory`"

**Verifier (high confidence)**: Quotes verified: both line-3 descriptions are word-for-word parallel; lines 14/174 confirm the rebalancer role. Recommendation is additive frontmatter-only — no Go or skill cross-refs embed the description (fabhelp.go uses name only), and SPEC-docs-reorg-memory.md:5 already advertises the rebalancer, so mirror churn is near-zero. Counter-evidence: skills ARE distinguishable (memory vs spec), and "suggest reorganization" plausibly matches "rebalance" with no competing skill — non-invocation risk modest. Worth doing; severity low.


### `g4-7` [LOW] fab-switch description omits --none deactivation; no skill description covers 'deactivate the active change'

**Files**: fab-switch.md

fab-switch supports `--none` to remove the `.fab-status.yaml` symlink (fab-switch.md:15, 49-51), but the description only mentions switching and listing. A "deactivate the current change" request matches nothing on the description surface, so the capability is undiscoverable at invocation time.

**Recommendation**: Append to the frontmatter description: "Pass --none to deactivate the current change."

**Evidence**: fab-switch.md:3 vs fab-switch.md:15: "`--none` — deactivate the current change by removing the `.fab-status.yaml` symlink"

**Verifier (high confidence)**: Confirmed: fab-switch.md:3 description omits --none while line 15 defines it; no other skill description mentions deactivation (fab-archive's "clear pointer" is archive-scoped). Recommendation is purely additive — the description string is not referenced elsewhere, and SPEC-fab-switch.md:5 already says "Supports deactivation via --none", so the mirror is already aligned. No constitution conflict. Severity low is correct; explicit invocations work, only NL routing discoverability suffers.


### `g4-9` [LOW] git-pr description doesn't disclose that PRs are created as drafts

**Files**: git-pr.md

The description promises "create a GitHub PR", but the body always creates a draft (`gh pr create --draft`, both primary and fallback paths). A user or orchestrator invoking it for a ready-for-review PR gets a draft with no signal at selection time.

**Recommendation**: Change the frontmatter description to "...create a draft GitHub PR — no prompts, no questions." (One word; no behavior change.)

**Evidence**: git-pr.md:3 vs git-pr.md:212: "`gh pr create --draft --title ...`" and :214 fallback "`gh pr create --draft --fill`"

**Verifier (high confidence)**: Confirmed: git-pr.md:3 description omits "draft" while both creation paths (:212, :214) use `gh pr create --draft`; no skill ever runs `gh pr ready`, so draft is terminal. glossary.md:36 and skills.md:755 already say "draft", so the fix improves consistency. Nothing parses the description string; only SPEC-git-pr.md:5 and skills.md:735 mirrors need the same one-word update (normal procedure). One-word change, no behavior loss, no constitution conflict. Severity low stands.



## Appendix A — Refuted findings (16)

Reviewer claims that did not survive adversarial verification — kept here because the refutation reasons are themselves informative about intentional design.

- **[f002] Zero-consumer Run-Kit reference + Operator Spawning Rules bloat always-loaded _preamble** — Facts verified (lines 152-173/176-237; grep matches only _preamble). But zero-static-consumers is by design: memory _shared/context-loading.md:156-161 documents rk as ambient capability for EVERY fab session (emergent runtime use, invisible to grep), reaffirmed in #341 (inlined, not deleted) and ce6c3310 (slimmed but kept). Moving rk to operator-only _cli-external loses that behavior and reverses a documented decision. Salvageable residue: Operator Spawning Rules (~22 lines) move — valid but small; fab-operator §5 mostly supersedes it.
- **[f009] 470-line _cli-fab has one consumer using almost none of it — split** — Facts verified (470 lines, sole consumer fab-operator, ~6 command families used). But docs/specs is not packaged by scripts/release.sh — moving sections there strands _preamble.md:243's deployed "full reference" pointer and kit-architecture.md:12's canonical-reference contract. "Maintainer-only" sections document commands deployed skills invoke (memory-index, pr-meta, doctor, migrations-status). Constitution:31 mandates one doc target; splitting doubles drift risk. Tax is one-time ~4K tokens per long-lived singleton operator session. Right fix: re-compress to ≤300 lines per 260418-or0o, not relocate.
- **[f021] Change-type inference via substring keywords over full intake content over-triggers deterministically** — Quote accurate, but "deterministic" is wrong: change_type is actually set by the artifact-write hook (hook.go:261 -> artifact.go:95-101) using word-boundary regexes — "prefix"/"specific" cannot match — and it re-fires on every intake.md edit, overwriting Step 6. Gate is flat 3.0; type only shifts expected_min/cover and PR cosmetics. The skill-only, title-scoped fix diverges from the Go owner and misses git-pr.md:39-42. Real residual: skill/spec wording drifted from Go (\b, "redesign").
- **[f039] Stale 1.10.0 changelog annotations persist across 7-8 files** — Facts confirmed (12 "1.10.0" hits at cited lines, VERSION 2.1.5, 0.47.0 pin). But the recommendation loses behavior: migrations/1.9.7-to-1.10.0.md state-1 deliberately defers spec.md folding to _generation.md's legacy block (migration-time folding rejected to avoid resumability deadlock); deleting it orphans mid-spec repos. "(1.10.0)" is house style mirrored ~17x in docs/specs and Go comments; 0.47.0 sits in a deliberately-wrong example. Only "one-release" label is stale — a one-word fix.
- **[f056] pr-meta and memory-index contracts are fully duplicated in consumer skills that never load this file** — Citations verified; consumers truly never load _cli-fab (only fab-operator declares it). But "fully duplicated" overstates: consumers inline ~3 condensed sentences; the output contracts exist only in _cli-fab. Slimming breaks docs/memory/pipeline/execution-skills.md:48, which designates _cli-fab as home of pr-meta's "full signature, output contract, and exit codes" (also templates.md:115). Duplication is the deliberate or0o helpers-opt-in design; constitution line 31 forces _cli-fab sync on CLI changes, so drift risk is small.
- **[f059] Expired 'one-release back-compat' legacy spec.md ingestion still present** — Quotes accurate, but removal breaks live behavior: migration 1.9.7-to-1.10.0.md State 1 explicitly leaves spec.md-without-plan.md changes "for on-apply ingestion" by this exact block (a plan.md stub would deadlock the resumability guard). All 23 migrations back to 0.2.0 are retained, so upgrades from <=1.9.7 stay supported indefinitely; ingestion fires at first apply post-upgrade. docs/specs/skills.md:460 mirrors the fold; CHK-* still live in glossary/templates. Merge is 10 days old; no v1.10.0 tag (shipped as v2.0.0). Correct fix: reword "one-release", not delete.
- **[f063] Structured Output section duplicated for inward/outward, with a severity-mapping tension** — Facts confirmed: _review.md:85 and :136 share the verbatim closing sentence; :82 vs :57-60 mapping exists. But only one sentence duplicates — per-tier bullets differ per agent (agent-specific examples, not redundancy). The downgrade risk is speculative: :53 mandates parsimony findings carry "the mapped severity" at classification, pre-merge. No heading-level cross-references break (fab-continue:144, fab-ff/fff:66 cite the file generically). Merging reduces per-dispatch self-containment for marginal dedup; benefit below churn (skill + SPEC mirror + skills.md:499).
- **[f066] 4-char natural-language inputs are captured by the backlog branch and abort instead of falling back** — Facts confirmed at fab-new.md:35,34,226-227 and fab-draft.md:37,153-154. But the fix loses fail-safe behavior: fab-operator.md:477-484 spawns unattended agents with '/fab-new <backlog-id>' from autopilot queues — NL fallback on a typo'd/stale ID would silently build a garbage change from a 4-char string. Abort-on-local-miss vs fallback-on-remote-Linear-failure is a defensible design, and 4-char NL descriptions are rare and useless as intake input. Severity is low, not medium.
- **[f072] ff/fff never invoke set-acceptance despite listing it; acceptance metrics go stale on autonomous runs** — Staleness claim is false: the PostToolUse hook (hook.go:270-298; SPEC-hooks.md:44-50; templates.md:61) auto-updates acceptance_completed on every plan.md write, and the inward reviewer marks [x] acceptance items in plan.md (_review.md step 2). Adding set-acceptance to ff/fff contradicts SPEC-hooks.md:216-221, which removed those calls deliberately. The line-42 note is a mutation-policy list, not a usage inventory — it also lists start/advance, which ff/fff never invoke.
- **[f074] ff/fff orchestrators load helpers and deep context their subagents re-load anyway** — Facts verified, but the fix's premise is false. ff/fff's rework loop has the ORCHESTRATOR itself editing plan.md ## Requirements/## Tasks/## Acceptance (fab-ff.md:77-79), which needs _generation.md's format contract (T{NNN}, A-{NNN} R#, REQUIRED <!-- R# --> traceability, _generation.md:87-91) — so _generation is not apply-subagent-only. _generation.md:11/_review.md:11 explicitly name ff/fff as consumers; _review.md:159 places the rework loop in these orchestrators. Section 2 already loads plan.md, so the proposed slimming is incoherent. Redundancy is happy-path-only context cost.
- **[f076] fab-switch/git-branch prefix dispatch mixes skill-following with a literal CLI command and is heavyweight** — Quotes accurate, but the divergence claim fails: fab-switch.md:37-41 says the skill's argument flow IS "Delegate to `fab change switch` via a single Bash call" — the CLI line composes with reading the skill and pins the non-interactive entry point. Pattern is spec'd (SPEC-fab-proceed.md:58-59); 5-file context is mandated by _preamble.md:352-357 MUST. Dropping the CLI line risks the interactive no-arg flow; orchestrator Bash loses `fab log command` telemetry and violates the dispatch convention. Only a phrasing inconsistency remains.
- **[f084] Pre-flight failure message gives wrong guidance on a fresh repo** — Premise false: fab kit-path never reads config.yaml. Router falls back to bundled version when config is missing (fab-kit/cmd/fab/main.go:70-76) and EnsureCached auto-downloads; kitpath.go resolves kit/ beside the cached fab-go binary. So a never-init repo doesn't hit this failure. Also /fab-setup only exists after fab sync deploys it (.claude is gitignored), so pre-init invocation is unreachable. fab-setup.md:34 deliberately mirrors kitpath.go:18, where sync/upgrade-repo is correct; fab init would wrongly re-pin fab_version.
- **[f095] git-pr 'nothing to do' path: 'silently, no errors' contradicts Step 4c's fail-fast and print** — Citations verified (git-pr.md:125, :247, :250), so textual tension exists. But the path ends with "Then STOP" either way — no abort divergence, identical git commands; only a cosmetic print differs. "(silently, no errors)" is an explicit call-site override matching established style (4a/4b already carve out silent best-effort exceptions to Fail fast). Recommendation option 1 restates existing semantics; option 2 loses intended quiet behavior. No cross-refs: phrase exists only in git-pr.md; SPEC mirror omits this path.
- **[f104] Shape bounds defined in three places (two skills + Go code)** — Duplication confirmed at all three cited spots, but it's ~8-way (also fab-continue.md:174, _cli-fab.md:334, docs/specs/templates.md:459, memory-docs/templates.md:124, configuration.md:125). The rec's 2-line summary keeps all four numbers in hydrate-memory, so drift surface is unchanged; the Go comment already delegates to the skills; and configuration.md:125 records this multi-place layout as a deliberate tciy decision. Churn exceeds benefit.
- **[g2-1] $(fab kit-path) template reads have no exit guard; failure is misdiagnosed as kit corruption** — Cited text confirmed (no guard at _generation.md:29,59; generic error rows). But the failure scenario is precluded: fab-new/fab-draft run `fab change new` (with stderr-surfacing error row) before the template read; fab-continue/ff/fff dispatch on `fab preflight` (non-zero on error). Also LLM agents see fab's stderr from the failed substitution, so misdiagnosis is unlikely; when kit-path succeeds but template is missing, "kit may be corrupted" IS correct. Guard step + 3 table splits + SPEC mirrors exceed benefit.
- **[g4-3] fab-ff description implies it runs intake and diverges from fab-fff's phrasing, weakening the ff/fff disambiguation** — Quotes verified, but no real contradiction: _preamble.md:516 scopes to confidence gates; fab-ff's gate (2) is a rework stop, and SPEC-fab-ff.md:5 deliberately uses the same "Two gates" frame (both from commit 62b2608a). "From intake through hydrate" is canonical vocabulary (glossary.md:49/115, user-flow.md:38, 3 memory files), and fab-ff does finish the intake stage (fab-ff.md:52). The recommended rewrite would diverge from 6+ docs. Only a wording nit remains.


## Appendix B — Unverified low-severity findings (75)

Consolidated low-severity findings that were not adversarially verified (cap of 90 verifications applied to high/medium first). Treat as plausible leads, not confirmed facts.

- **[f127] fab-new/fab-draft hardcoded Next: lines drift from the state table** (fab-new.md, fab-draft.md) — Delete the hardcoded trailing Next: lines (fab-new.md:231, fab-draft.md:158) and keep only the per-state-table directive already present in each Output block.
- **[f128] Removed spec artifact still listed in fab-discuss/fab-operator** (fab-discuss.md, fab-operator.md) — Drop 'spec'/'specs' from the artifact enumerations at fab-discuss.md:42 and fab-operator.md:29, leaving '(intake, plan)'.
- **[f129] Stale cross-references (_review points to wrong orchestrator step)** (_review.md, fab-ff.md, fab-fff.md, fab-continue.md, fab-archive.md) — Fix _review.md:160 to 'Step 2'; extend fab-continue.md:3 description to '...review, hydrate, ship, or PR review'; change fab-archive.md:98 to derive Next: from the remaining active change's state (initialized only when the pointer was cleared).
- **[f130] Small staleness nits (Section 2 direction, preflight id, yq writes)** (_preamble.md, _cli-fab.md, git-pr-review.md) — (1) Change to 'Section 2 below'. (2) Add `id` to _cli-fab.md:92. (3) Either add a `fab status set-phase` subcommand (and update constitution-mandated _cli-fab.md) or annotate git-pr-review's yq writes as a documented exception to the event-command convention.
- **[f131] Intake gate + --force bypass stated three times within _preamble** (_preamble.md) — Delete the Invocation paragraph at line 532 (keep the bullet list above it); let Gate Threshold (516-520) be the single statement. Trim the line-248 row to '--check-gate returns non-zero below the intake gate (see § Confidence Scoring)'.
- **[f132] Stale residue: '_naming/_cli-rk inlined' note and scattered 1.10.0 changelog asides** (_preamble.md) — Cut line 105 or reduce to 'helpers not in the Allowed list are invalid'. Time-bound the 1.10.0 transition notes (one-release back-compat per _generation's precedent) and remove them in the next minor; the rationale already lives in the constitution amendment comment.
- **[f133] Standard Subagent Context partially duplicates the always-load layer and silently drops the two indexes** (_preamble.md) — State the relationship explicitly in § Standard Subagent Context: either 'subagents executing a skill file inherit §1; this list is the floor for skill-less prompts (indexes intentionally excluded because the dispatch prompt scopes memory)' — or align both lists.
- **[f134] Context Loading §4 embeds fab-continue-specific steps in the universal layer** (_preamble.md, fab-continue.md) — Keep §4 items 1-2 (scope-to-touched-files rule, genuinely universal); move items 4-5 into fab-continue.md/_review.md where Pattern Extraction and review re-reading are already specified — fab-continue.md:106-118 duplicates the same four pattern categories.
- **[f135] File's own command enumeration and router list have drifted from its sections and the Go allowlist** (_cli-fab.md, main.go) — Add migrations-status and memory-index to the line-27 enumeration and migrations-status to the line-17 router list — or drop both inline lists and let section headings be the index.
- **[f136] Undocumented visible commands/flags: fab shell-init and fab change list --show-stats** (_cli-fab.md, shellinit.go, change.go) — Add `--show-stats` to the `list` row (_cli-fab.md:41); add a one-line `fab shell-init` entry (or include it in the maintainer-reference split if it is human-setup-only).
- **[f137] Operator state-path slug algorithm is over-specified and triple-documented; consumer is told not to compute it** (_cli-fab.md, fab-operator.md) — Cut _cli-fab.md:426 to two sentences (server-keyed file under `$XDG_STATE_HOME/fab/operator/<server-slug>.yaml`, derived by the binary, no migration of old `.fab-operator.yaml`); keep the operator skill's :65/:121 as the single behavioral statement, trimming one of the two repeats there.
- **[f138] fab hook section documents runtime internals no agent acts on** (_cli-fab.md) — Reduce to a 2-3 line summary (hooks exist, all exit 0, `fab hook sync` registers them) or move the full detail to the maintainer reference proposed in the structural split.
- **[f139] Traceability requirement stated three times within Plan Generation** (_generation.md) — Keep the normative statement in step 4 (lines 87-91); delete the step 5 bullet (line 104) and the '(REQUIRED)' restatement in step 6 (lines 122-124) — the format strings carry the contract. Leave the template comment as the in-artifact reminder.
- **[f140] Intake 'Generation rule' blockquote duplicates step 3's concreteness instructions** (_generation.md) — Collapse: keep the one-sentence state-transfer rationale in the blockquote, move all concreteness/verbatim instructions into step 3 only (deleting 'If a design decision was discussed with specific values — include them verbatim. Do not summarize or abstract.' from the blockquote).
- **[f141] PostToolUse hook field list drifted between _generation.md and fab-continue.md** (_generation.md, fab-continue.md) — Keep the authoritative field list in _generation.md step 7; reduce fab-continue.md:101 to 'The PostToolUse hook updates the plan: block automatically (see Plan Generation Procedure step 7)'.
- **[f142] Review preconditions define a STOP message only for missing Acceptance** (_review.md) — Extend _review.md Preconditions with the two missing error paths: 'plan.md not found — run /fab-continue (apply) first.' and 'plan.md missing Tasks section.' — paralleling the existing Acceptance message.
- **[f143] Default-branch resolution for the outward diff is unspecified** (_review.md) — Specify the resolution command in the Context bullet, e.g., `git symbolic-ref refs/remotes/origin/HEAD --short` with origin/main as last resort — one line, removes the improvisation.
- **[f144] Parsimony step misuses 'Findings Merge step' for intra-agent output assembly** (_review.md) — Reword _review.md:62 to 'Findings are included in the inward sub-agent's structured output (and thus flow through the Findings Merge with everything else)' — clarification only, no behavior change.
- **[f145] Inward validation step 1 can never fail given precondition 2** (_review.md) — Either delete step 1, or keep it explicitly as re-verification with a defined outcome: 'if any task is unchecked, return a Must-fix finding' — pick one so the sub-agent's behavior on failure is deterministic.
- **[f146] Good but fragile: '## Deletion Candidates' parser contract and skip-list maintained in three files** (_review.md, plan.md, fab-continue.md) — Declare _review.md step 8 the single normative source: have templates/plan.md:59-68 and fab-continue.md:173 reference it ('per _review.md Deletion Candidates contract') instead of restating the skip list and placement rules; within _review.md, state the skip list once and have step 8 say 'skipped under the same conditions as Step 7' (it already does — keep that form).
- **[f147] Confidence output format contradicts itself within fab-new and fab-draft (cover field)** (fab-new.md, fab-draft.md) — Pick one format (the Output template's, with cover) and delete the 'Output format:' line from Step 7, leaving 'Display per the Output section'. Fix in both files.
- **[f148] fab-clarify repeats its two STOP messages with divergent wording** (fab-clarify.md) — Keep each message once in the Error Handling table; have the Pre-flight section and Step 1 reference the table ('STOP per Error Handling') instead of restating text. Move the missing-intake check out of the post-intake bullet into its own pre-flight line.
- **[f149] Coverage Summary categories (Clear, Deferred, Outstanding) are undefined** (fab-clarify.md) — Add a one-line definition per row in Step 6 (e.g., Resolved = answered this session; Clear = scanned, no gap; Deferred = user skipped/stopped early; Outstanding = queued but unasked beyond the cap).
- **[f150] Individual-question reclassification omits the Scores/Rationale updates that bulk confirm mandates** (fab-clarify.md) — In Step 4 item 2, reference the bulk-confirm Artifact Update table: apply the same Rationale strings and S → 95 update for individually answered questions.
- **[f151] Good but fragile: Conversation Context Mining depends on fab-proceed's description synthesis under subagent dispatch** (fab-new.md, fab-proceed.md) — Add one sentence to Step 4: 'When invoked by a dispatcher (subagent context with no conversation), all prior-discussion content must arrive via the <description> — see /fab-proceed Description Synthesis.' Keeps the contract explicit for future callers.
- **[f152] fab-proceed restates step-gating conditions four times** (fab-proceed.md) — Keep the per-step parenthetical annotations (Steps 3/4) and the dispatch table; delete the second sentence of fab-proceed.md:30 and the sentence at :52 ('When Step 1 found an active change, Steps 3 and 4 SHALL NOT run...').
- **[f153] Terminal Skill-tool delegation of /fab-fff is good but fragile — undocumented exception to _preamble's 'never the Skill tool'** (fab-proceed.md, _preamble.md) — Add one line to _preamble.md § Subagent Dispatch: 'Exception: a terminal delegation (the orchestrator's final step, e.g., /fab-proceed → /fab-fff) MAY use the Skill tool since no pipeline context follows it.' Keep fab-proceed's rationale sentence as-is.
- **[f154] fab-proceed's intake scan cannot distinguish drafts from completed or abandoned changes** (fab-proceed.md) — Filter candidates in Step 4 to changes whose pipeline is incomplete (e.g., skip those with `review-pr: done` via the resolve/status query), or add one sentence documenting that completed-but-unarchived changes are eligible and the handoff is a no-op.
- **[f155] Error/stop outputs lack the mandatory Next: line** (fab-proceed.md, fab-continue.md) — Append Next: lines to the specified stop outputs: fab-proceed.md:186 → 'Next: /fab-new (or /fab-draft)'; fab-continue.md:56 all-done block → 'Next: /fab-archive' (per the review-pr (pass) state row).
- **[f156] fab-proceed skips command telemetry entirely** (fab-proceed.md) — After dispatch resolves a change (post fab-new/fab-switch, before the /fab-fff handoff), add: `fab log command "fab-proceed" "<id>" 2>/dev/null || true` — consistent with _preamble §2 step 4's best-effort pattern.
- **[f157] Arguments section and Argument Classification table state the same routing twice** (fab-setup.md) — Keep the Argument Classification table as the single router; fold the validate redirect text and unknown-subcommand message into table cells and reduce the Arguments section to the one-line subcommand list.
- **[f158] Symlink residue: deployment is copy-mode, not symlinks** (fab-setup.md) — Replace 'symlink' with 'skill copies' at lines 61 and 468 (Idempotency section).
- **[f159] Step 1j deployment pattern '.claude/skills/fab-{name}/SKILL.md' is wrong for non-fab skills** (fab-setup.md) — Change line 133 to `.claude/skills/{skill}/SKILL.md`.
- **[f160] upgrade-repo no-op stamping note stated twice in Migrations section** (fab-setup.md) — Keep the note once at Step 2.3 (line 355); delete the parenthetical block at line 440.
- **[f161] Applying a Migration reconstructs the filename instead of using the binary's `file` field** (fab-setup.md) — Change line 375 to 'Read `$(fab kit-path)/migrations/{file}` (the `file` field from the applicable entry)'.
- **[f162] Mid-chain failure recovery silently depends on migration files being idempotent (good but fragile)** (fab-setup.md, 1.9.7-to-1.10.0.md) — Add one sentence to Applying a Migration (after line 380): 'Migration files MUST be internally idempotent — re-running after a partial application is the recovery path.' (Also belongs in docs/memory/distribution/migrations.md Migration File Format.)
- **[f163] 'No Migrations Apply' output leaves a stale local version with no user guidance** (fab-setup.md) — Append one line to the No Migrations Apply template (line 437): 'Run fab upgrade-repo to stamp fab/.kit-migration-version to {target}.' This is an explicit output-behavior addition, not a silent rewrite.
- **[f164] Keep the three concerns in one skill, but the whole 494 lines load per subcommand** (fab-setup.md, setup.md) — Do not split. Apply the cuts from the other findings (steps 1c-1k, migrations version triplication, Semver section, duplicate routing/output blocks) — together roughly 100 lines, ~20% of the per-invocation cost — and keep subcommand sections self-contained so agents can skip to the routed section.
- **[f165] Gap-skip output template duplicates the multi-step template** (fab-setup.md) — Delete the 'Migration with Gap Skip' template; add a one-line note under the multi-step template: 'Any gap_skips lines print before the first [i/N] entry.'
- **[f166] SPEC-fab-setup.md has drifted from the skill (stale paths, symlinks, fab-sync.sh)** (SPEC-fab-setup.md, fab-setup.md) — When applying the skill fixes, update SPEC-fab-setup.md in the same change: kit-path-based pre-flight, `fab sync` (not fab-sync.sh), copies (not symlinks), and the revised bootstrap step list.
- **[f167] Good but fragile: skill control flow keyed to exact CLI stderr strings with no documented contract** (fab-switch.md, fab-archive.md, _cli-fab.md) — Document the matched stderr substrings as a stability contract in _cli-fab.md (constitution already requires _cli-fab updates for CLI changes), or add Go tests asserting the exact prefixes.
- **[f168] fab-switch is the only skill with a variant preamble-read line** (fab-switch.md) — Either revert fab-switch.md:8 to the standard sentence, or move the 'no Bash before the preamble Read completes' rule into the canonical blurb in _preamble.md so all skills inherit it.
- **[f169] fab-switch defines two different exact messages for the same no-changes condition** (fab-switch.md) — Keep the fuller line-32 message as canonical and make the Error Handling row reference it ('see No Argument Flow step 2') instead of restating a shorter variant.
- **[f170] fab-archive restore mode never logs command telemetry** (fab-archive.md) — Add to restore Step 1: `fab log command "fab-archive" "<restored-change>" 2>/dev/null || true` after a successful `fab change restore` (mirroring fab-switch.md:64-70).
- **[f171] fab-archive hydrate guard blocks changes whose hydrate stage was skipped** (fab-archive.md) — Change fab-archive.md:37 to 'If progress.hydrate is neither done nor skipped, STOP' — note this is a deliberate behavior change and should be confirmed against the skip semantics.
- **[f172] fab-status instructs redundant raw-file reads alongside preflight** (fab-status.md) — Drop `.fab-status.yaml` from line 42; scope the `.status.yaml` read to 'for `true_impact` and `change_type` only (not emitted by preflight)'; remove the `fab status` mention at line 40. Longer term, extend preflight YAML with those two fields and drop the read entirely.
- **[f173] fab-archive Key Properties self-contradicts on .status.yaml modification** (fab-archive.md) — Change fab-archive.md:109 to 'Only `last_updated` (via the command); no stage/progress fields' or verify the Go command's actual behavior and state it plainly.
- **[f174] git-pr Step 0a/4b conditions require an unspecified .status.yaml read and are redundant** (git-pr.md) — Drop both progress.ship conditions; keep the unconditional best-effort `fab status start/finish ... || true` calls (identical behavior, fewer instructions).
- **[f175] git-pr: resolution 'source' is stored 'for use in Step 3c' but never used** (git-pr.md) — Delete the 'and the resolution source... for use in Step 3c' clause from line 54.
- **[f176] git-pr enumerates the seven valid types three times** (git-pr.md) — Keep the line-33 list as canonical; in resolution steps 1-2 say 'a valid type' without re-enumerating; keep the bottom table only for the per-type descriptions and Conventional Commits note (or fold descriptions into line 33 and delete the table).
- **[f177] git-branch: Output and Error Handling sections duplicate the Behavior steps verbatim** (git-branch.md) — Delete the ## Output section (Step 5 is the single source); keep the Error Handling table but remove the corresponding inline prose from Steps 1-2, or vice versa — one statement per rule.
- **[f178] git-branch branch-existence check matches any ref, not just local branches** (git-branch.md) — Change the check to `git rev-parse --verify "refs/heads/{branch_name}"` (or `git show-ref --verify --quiet "refs/heads/{branch_name}"`).
- **[f179] git-pr starts the ship stage before the main/master branch guard** (git-pr.md) — Move the branch guard (Step 2) ahead of Step 0a, or condition Step 0a on not being on main/master.
- **[f180] Copilot reviewer matching depends on gh's GraphQL bot-login form (good but fragile)** (git-pr-review.md) — Use a prefix match in the poll jq (`select(.author.login | startswith("copilot-pull-request-reviewer"))`) and add a one-line note that REST and GraphQL render the login differently.
- **[f181] git-pr-review --tool machinery is ~12 lines for a single valid value** (git-pr-review.md) — Delete Step 1.5; fold into Phase 2's configuration paragraph: 'If invoked with `--tool copilot`, skip this config check (any other --tool value: print `Invalid tool: {name}. Valid values: copilot.` and stop).'
- **[f182] hydrate-memory hard-codes the 'initialized' Next: state** (docs-hydrate-memory.md, _preamble.md) — Change docs-hydrate-memory:192 to 'Next: {per state table — state reached}' (matching the preamble's Lookup Procedure), or explicitly state the skill always reports the initialized row regardless of active change.
- **[f183] hydrate-memory pre-flight error message conflates two failure conditions** (docs-hydrate-memory.md) — Split into two messages, or reword the shared one to 'docs/memory/ or docs/memory/index.md not found. Run /fab-setup first.' Keep the 'Do NOT create these' rule as-is.
- **[f184] Error Handling tables restate pre-flight and argument rules verbatim** (docs-hydrate-memory.md, docs-hydrate-specs.md, docs-reorg-memory.md, docs-reorg-specs.md) — Keep Error Handling rows only for runtime failures not stated elsewhere (unreachable URL, write failure, dangling link); drop the rows duplicating pre-flight aborts — or invert: keep the table and reduce Pre-flight to 'see Error Handling'. Apply the same choice across all four files.
- **[f185] Stale reference to never-existing /fab-rebalance-memory** (docs-reorg-memory.md) — Trim docs-reorg-memory:174 to 'Yes — shape diagnosis + split/merge/flatten + the file-moving apply path live here', dropping the supersedes clause.
- **[f186] hydrate-specs mischaracterizes the forward hydrate direction** (docs-hydrate-specs.md) — Reword line 17 to: 'Hydrate normally flows into memory (change artifacts, external sources); hydrate-specs flows the other way: memory → specs.'
- **[f187] Good but fragile: generated-index model depends on description: frontmatter nobody repairs** (docs-hydrate-memory.md, docs-reorg-memory.md) — Behavior addition (flagging explicitly, not a silent rewrite): extend docs-reorg-memory Step 5.3 to also add description: frontmatter to any moved file lacking it, and have the Shape Report (Step 3) count description-less files so the gap is at least visible.
- **[f188] Approval tooling specified in single mode but not batch mode** (internal-skill-optimize.md) — Specify AskUserQuestion with explicit options (Apply all / Select specific / Cancel) in batch step 5, mirroring single-skill mode.
- **[f189] consistency-check Classification rubric defined after use and never given to the auditing agents** (internal-consistency-check.md) — Move the Classification definitions into the Execution section and state that they are included verbatim in each agent prompt alongside the resolved source_paths.
- **[f190] internal-retrospect has no empty/short-session path** (internal-retrospect.md) — Add one line after the intro: if the conversation contains no prior task work, output 'Nothing to retrospect — run at the end of a working session' and stop.
- **[f191] False claim: 3-minute tick cadence described as 'sub-minute resolution'** (fab-operator.md) — Replace with 'Tick cadence (default 3m) is sufficient resolution for the 30m threshold — no new polling infrastructure is required.'
- **[f192] Frame example shows a `gmail-deploys` watch but the schema allows only linear|slack sources** (fab-operator.md) — Rename the example watch to a slack- or linear-plausible name (e.g., `slack-deploys`), or extend §7 with a gmail source if that's intended.
- **[f193] Idle-message timing is ambiguous ('between ticks' is not an executable moment)** (fab-operator.md) — Reword §4 Idle Message: 'End each tick's output (after the action footnote) with the idle line, using `fab operator time --interval {interval}` for now/next values.'
- **[f194] Branch Fallback and branch-alignment checks silently depend on branch == change-folder-name (fragile vs branch_prefix)** (fab-operator.md, _cli-fab.md) — Either reconcile `fab batch switch`'s branch_prefix with _preamble's 'No prefix' branch convention, or make §3 step 4 compare against the resolved branch from `branch_map`/monitored entry rather than raw folder-name equality.
- **[f195] Watch '3 consecutive failures → disable' counter has no persisted field** (fab-operator.md) — Add a `consecutive_failures` field to the §7 watch schema (reset to 0 on success), or change the rule to a stateless one (e.g., disable when `last_error` persists across 3 ticks by timestamp comparison).
- **[f196] Change-type keyword inference table copied three times** (fab-new.md, fab-draft.md, git-pr.md) — When fab-new/fab-draft Steps 0-9 are factored into _generation.md (see twin finding), the table lands there once. In git-pr, annotate the subset with 'derived from the inference table in _generation.md — keep keyword lists in sync' so the intentional truncation does not drift silently.
- **[f197] docs-reorg twins: divergence is justified — parallel maintenance sustainable, no merge warranted** (docs-reorg-memory.md, docs-reorg-specs.md, docs-hydrate-memory.md, docs-hydrate-specs.md) — No shared-core refactor. Optionally extract only the identical Theme-table format (Step 2) and confirm-options line ('apply all / cherry-pick / skip') if a docs-skills helper ever materializes; today the duplication is below the threshold where a helper pays for itself.
- **[f198] Command telemetry coverage is inconsistent — git-*, docs-*, and internal-* skills never log `fab log command`** (git-pr.md, git-pr-review.md, git-branch.md, docs-hydrate-memory.md, docs-hydrate-specs.md, docs-reorg-memory.md, docs-reorg-specs.md) — Add the standard best-effort line (`fab log command "<skill>" 2>/dev/null || true`) after PR/branch resolution in git-pr Step 1, git-pr-review Step 1, git-branch Step 2, and after pre-flight in the four docs-* skills — or document the exclusion in _preamble §2 step 4.
- **[f199] git-pr accepts an undocumented positional {type} argument; git-pr-review documents --tool outside any Arguments section** (git-pr.md, git-pr-review.md) — Change git-pr's H1 to `# /git-pr [type]` and add a two-line Arguments section pointing at Step 0b; convert git-pr-review's `--tool` paragraph (line 11) into the standard `## Arguments` section.
- **[f200] Key Properties table present in 14 skills but absent from 10 others including the core orchestrators** (fab-new.md, fab-draft.md, fab-ff.md, fab-fff.md, git-pr.md, git-pr-review.md, docs-hydrate-memory.md) — Add the standard Key Properties table to fab-new, fab-draft, fab-ff, fab-fff, git-pr, git-pr-review, and docs-hydrate-memory (internal-* skills are arguably exempt as kit-developer tools — if so, say nowhere or note it once).
- **[f201] fab-status Impact line hard-codes the label 'excluding fab/docs' though excludes are config-driven** (fab-status.md) — Change the template at fab-status.md:50 to derive the label from config, e.g. `excluding {true_impact_exclude joined} +{excl_net} ...`, matching _cli-fab.md:284's pr-meta behavior.