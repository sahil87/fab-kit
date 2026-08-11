# Intake: Make `fab status refresh` change argument optional + patch the dispatch-epilogue canon

**Change**: 260811-izj3-refresh-optional-change-arg
**Created**: 2026-08-11

## Origin

Promptless dispatch via `/fab-proceed` (create-new prefix step, `{questioning-mode} = promptless-defer`), from a synthesized description of the live conversation:

> Every dispatched pipeline worker is instructed (via `_preamble.md` § Dispatch-Prompt Obligations, obligation 3) to end with a terminal `fab status refresh` epilogue. But the CLI requires the change argument — `statusRefreshCmd` in `src/go/fab/cmd/fab/status.go` declares `Use: "refresh <change>"` with `Args: cobra.ExactArgs(1)` — so the bare form exits 2. The canonical prose that dispatch prompts are composed from writes the **bare form** in ~5 places in `src/kit/skills/_preamble.md` and 3 places in `docs/specs/harness-adapters.md`. Workers copy the bare form, hit exit 2, and burn a self-recovery turn — observed repeatedly across sessions (most recently 2026-08-11), and recorded as a recurring lesson in early August without the canon ever being patched. Ship both fixes in one change: (1) patch the canon to spell the change token; (2) make the CLI argument optional with active-change fallback (root-cause).

Both decisions were made in the originating conversation; no interactive questions were asked (promptless-defer). All grounding facts below were re-verified against the working tree at intake time.

## Why

1. **The pain point.** The dispatch-prompt contract (obligation 3, binding all three adapters — native Agent-tool, headless CLI, interactive pane) requires every worker to end with a terminal `fab status refresh`. The canonical prose spells the **bare** command, but the CLI hard-requires the change argument (`cobra.ExactArgs(1)` → exit 2 on the bare form). Every worker that copies the canon verbatim hits exit 2 and burns a self-recovery turn figuring out it must append the change ID. This is pure waste, multiplied across every dispatched stage of every pipelined change.

2. **The consequence of not fixing.** The failure recurs on every dispatch (observed repeatedly, most recently 2026-08-11) and was recorded as a recurring lesson in early August — yet the canon was never patched, so the lesson keeps re-triggering. Left alone, every future worker pays the same tax, and non-Claude workers on the CLI adapters (codex/agy/kimi) may recover less gracefully than Claude does.

3. **Why this approach.** Two-sided fix, both cheap:
   - *Patch the canon* (the immediate cause): spell the change token at every site that states the literal command, so freshly composed dispatch prompts carry a working epilogue.
   - *Make the CLI forgiving* (the root cause): the bare form is a perfectly reasonable reading of "run the refresh epilogue" — accept it via active-change fallback. Risk is tiny: refresh is idempotent and pull-based (it recomputes a change's fields from that change's own artifacts — no cross-contamination possible), and the self-heal seams (`fab status advance`/`finish`, `fab preflight`) recompute anyway, so a mis-resolved or missed refresh is caught at the next transition.
   - The fallback infrastructure **already exists**: `resolve.ToFolder(fabRoot, "")` (reached via `resolve.ToAbsStatus`) resolves the active change from the `.fab-status.yaml` symlink when the override is empty — the exact resolution `fab preflight` uses when called bare. No new resolution code is needed.

## What Changes

### 1. CLI: `fab status refresh [<change>]` with active-change fallback

`statusRefreshCmd` at `src/go/fab/cmd/fab/status.go:405-424` (current: `Use: "refresh <change>"`, `Args: cobra.ExactArgs(1)`, `withStatusLock(args[0], ...)`):

```go
// after
Use:   "refresh [<change>]",
Args:  cobra.MaximumNArgs(1),
RunE: func(cmd *cobra.Command, args []string) error {
    return withStatusLock(optArg(args, 0), func(st *sf.StatusFile, statusPath, fabRoot string) error {
        ...
```

- `withStatusLock("")` → `resolve.ToAbsStatus(fabRoot, "")` → `resolve.ToFolder(fabRoot, "")` → `resolveFromCurrent` (the `.fab-status.yaml` symlink path) — identical to `fab preflight`'s bare resolution. The `optArg` helper already exists in `status.go` (used by `statusResetCmd` and others).
- **No active change + no argument** → `resolveFromCurrent`'s existing error propagates (non-zero exit, stderr message) — consistent with every other bare-invocable command. No new error text.
- `internal/refresh.Refresh` is untouched — this change is argument-parsing only.
- Update the cobra `Short`/help text only if the current wording states the argument as required.

### 2. Canon patch: spell the change token at every epilogue site

Spell the epilogue command **with the change token** — `fab status refresh <change>` (workers substitute the 4-char change ID they were dispatched with) — matching the form `docs/specs/hooks.md` already uses. Sites verified at intake time:

**`src/kit/skills/_preamble.md`** (5 sites):
| Line | Context |
|------|---------|
| ~331 | § Worker Continuation — "end with the terminal `fab status refresh`" |
| ~349 | § Pane-arm continuation step 1 — "obligations 1 and 3 (result file, terminal `fab status refresh`)" |
| ~419 | Pane steering bullet — "still owes `{stage}-result.yaml` and terminal `fab status refresh`" |
| ~483 | § Dispatch-Prompt Obligations item 3 — "End with a terminal `fab status refresh` epilogue" |
| ~487 | Block-contract carve-out — "REQUIRING the terminal `fab status refresh`" |

**`docs/specs/harness-adapters.md`** (3 mirror sites — this is the top-level owner spec for the dispatch-prompt obligations, NOT a skill mirror, so it IS in scope under constitution v1.5.0):
| Line | Context |
|------|---------|
| ~95 | Continuation obligations summary |
| ~361 | § Dispatch-prompt obligations item 3 |
| ~522 | Steering-is-contract-neutral bullet |

### 3. Sweep the whole class up front

Grep repo-wide (`src/kit/skills/`, `docs/specs/`, `docs/memory/`) for bare `fab status refresh` mentions and fix every one **in epilogue/worker-contract contexts** — i.e., any site that states the literal command a worker or prompt-composer would copy. Additional candidates found at intake time (verify each during apply):

- `docs/specs/glossary.md:32` — "Dispatch adapter" entry states the obligations triple incl. "terminal `fab status refresh`"
- `docs/memory/_shared/context-loading.md` (~118, ~133) — dispatch-prompt obligations prose
- `docs/memory/pipeline/hooks-may-enhance-never-own.md` (~15) — "the `fab status refresh` epilogue is the protocol-owned step"
- `docs/memory/pipeline/execution-skills.md` (~470) — continuation-epilogue design decision
- `docs/memory/runtime/dispatch.md` (~599) — steering-contract-neutral requirement

**Sweep rule** (owner-or-pointer aligned): a site that spells a *runnable command line* (or the command a worker copies into its epilogue) gets the token; a pure noun-phrase *name* for the mechanism ("the refresh epilogue", "terminal-refresh obligation") with no runnable command stays as-is — it is a pointer, not a restatement.

**Explicitly out of scope / no edit:**
- `docs/specs/hooks.md` — already spells `fab status refresh <change>` correctly; leave as-is. Its signature mention stays valid (the with-arg form remains supported).
- Downstream pointer sites that reference § Dispatch-Prompt Obligations without the literal command (`src/kit/skills/_pipeline.md` ~94, `src/kit/skills/fab-continue.md` ~72 — "terminal refresh" only) — no edit per the owner-or-pointer rule.
- `docs/specs/skills/SPEC-_preamble.md` — structural quick-reference mirror; this change is prose-only (no flow/tool-usage/sub-agent change), so the constitution v1.5.0 narrowed mirror rule does NOT trigger. No mirror updates for any skill.
- Descriptive mechanism prose about refresh's *self-heal* role (e.g., `src/kit/skills/_intake.md` ~88, `_generation.md` ~139) — these describe refresh as a mechanism, not the worker epilogue command; no edit.

### 4. `_cli-fab.md` signature row + tests (constitution-mandated)

- **`src/kit/skills/_cli-fab.md:124`** — the `fab status` table's `refresh` row currently shows `refresh <change>`; update to `refresh [<change>]` and note the omitted-arg active-change fallback (same resolution as bare `fab preflight`).
- **Tests**: extend `src/go/fab/cmd/fab/refresh_selfheal_test.go` (or a sibling `cmd/fab` test file, whichever fits existing structure): (a) bare `fab status refresh` with an active `.fab-status.yaml` symlink refreshes that change; (b) bare invocation with no active change exits non-zero with the resolution error; (c) the existing with-arg behavior is unchanged (likely already covered — verify, don't duplicate).
- Also sweep memory files that state the CLI signature: `docs/memory/pipeline/hooks-may-enhance-never-own.md` (~21, requirement prose says "by `fab status refresh <change>`") and `docs/memory/distribution/kit-architecture.md` (~290, "pull-based via `fab status refresh <change>`") — both remain *true* after the change (the with-arg form still works) but should note the now-optional argument where they document the command surface.

### 5. Toolkit Standards check (pre-ship gate, not a code change)

This change touches the CLI surface, so per the constitution's Toolkit Standards article: run `shll standards`, identify entries governing the CLI surface / help output, and check the new `refresh [<change>]` signature against them before shipping. If `shll` is unavailable, consult sahil87/shll `docs/site/standards/`.

## Affected Memory

- `pipeline/hooks-may-enhance-never-own`: (modify) owns the refresh-epilogue principle and the `fab status refresh <change>` requirement prose — document the optional argument + active-change fallback and the epilogue spelling
- `runtime/dispatch`: (modify) worker-contract requirements state the terminal refresh epilogue — update the spelled command form
- `pipeline/execution-skills`: (modify) continuation-epilogue design decision states the terminal refresh — update the spelled command form
- `_shared/context-loading`: (modify) dispatch-prompt obligations prose states the terminal refresh — update the spelled command form
- `distribution/kit-architecture`: (modify) CLI command-surface list states `fab status refresh <change>` — note the optional argument

## Impact

- **Go**: `src/go/fab/cmd/fab/status.go` (`statusRefreshCmd` only — ~4 lines); tests in `src/go/fab/cmd/fab/` (`refresh_selfheal_test.go` or sibling). `internal/refresh` and `internal/resolve` untouched. Test scope per code-quality.md: `go test ./cmd/fab/...` from `src/go/fab` first (add `./internal/refresh/...` only if touched); widen only if cross-cutting.
- **Kit skills (canonical source only — never `.claude/skills/`)**: `src/kit/skills/_preamble.md` (5 sites), `src/kit/skills/_cli-fab.md` (1 row). Prose-only skill edits → no SPEC-mirror updates (constitution v1.5.0).
- **Specs**: `docs/specs/harness-adapters.md` (3 sites), `docs/specs/glossary.md` (1 candidate site).
- **Memory**: the five files listed under Affected Memory (stale-literal sweep + command-surface note).
- **Behavior**: purely additive CLI change — every existing invocation (`fab status refresh <change>`) behaves identically; only the previously-erroring bare form gains behavior. No migration needed (no user-data restructuring). Dispatch prompts composed from the patched canon carry a working epilogue either way.

## Open Questions

- None. (Promptless-defer mode: no questions were asked; no decision scored Unresolved — the originating conversation pre-resolved the design and the codebase answers the rest.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Ship both fixes (canon patch + optional CLI arg) in this one change | Discussed — explicitly decided in the originating conversation | S:95 R:80 A:90 D:95 |
| 2 | Certain | No-arg fallback = existing `resolve.ToFolder(fabRoot, "")` symlink path via `optArg(args, 0)` + `cobra.MaximumNArgs(1)`; `internal/refresh` untouched | Verified in codebase — `withStatusLock("")` already resolves the active change exactly as bare `fab preflight` does | S:85 R:85 A:95 D:90 |
| 3 | Confident | Bare invocation with no active change errors non-zero via `resolveFromCurrent`'s existing message (no new error text, no silent no-op) | Not explicitly discussed; single obvious default — consistent with every other bare-invocable command | S:55 R:80 A:85 D:75 |
| 4 | Confident | Epilogue canon spelled `fab status refresh <change>` (worker substitutes its 4-char change ID), matching hooks.md's existing correct form | Discussed ("e.g. `fab status refresh <change>` / the 4-char change ID"); exact placeholder wording is the agent's call | S:65 R:85 A:80 D:75 |
| 5 | Confident | Sweep rule: runnable-command sites get the token; pure noun-phrase mechanism names without a runnable command stay (owner-or-pointer) | Reconciles the "fix every one" instruction with code-quality.md's owner-or-pointer anti-pattern; judged per-site during apply | S:65 R:85 A:80 D:70 |
| 6 | Certain | No `docs/specs/skills/SPEC-*.md` mirror updates — skill edits are prose-only | Constitution v1.5.0 narrowed mirror rule answers deterministically (no flow/tool-usage/sub-agent change) | S:80 R:90 A:95 D:85 |
| 7 | Confident | Memory files carrying the stale literal are updated in this change (apply sweep + hydrate), per the Affected Memory list | code-quality.md § Sibling & Mirror Sweeps: grep the old claim repo-wide, update the whole class up front | S:60 R:80 A:80 D:75 |
| 8 | Certain | `src/kit/skills/_cli-fab.md` refresh row updated + `cmd/fab` test updates included | Constitution Additional Constraints mandate both for any CLI signature change | S:90 R:85 A:100 D:95 |
| 9 | Confident | Tests extend `refresh_selfheal_test.go` (or sibling `cmd/fab` file); scope `go test ./cmd/fab/...` first | Existing test file already covers cmd-level refresh; code-quality.md test strategy says scope to affected packages first | S:60 R:85 A:80 D:75 |
| 10 | Certain | Pre-ship `shll standards` check for CLI-surface-governing entries | Constitution Toolkit Standards article mandates the check before changing the CLI surface | S:85 R:90 A:95 D:90 |
| 11 | Certain | `docs/specs/hooks.md` left as-is (already correct; with-arg form remains supported) | Discussed — explicit decision in the originating conversation | S:90 R:90 A:90 D:90 |

11 assumptions (6 certain, 5 confident, 0 tentative, 0 unresolved).
