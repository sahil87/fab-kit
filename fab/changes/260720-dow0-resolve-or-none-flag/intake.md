# Intake: fab resolve --or-none — absence as a first-class query result

**Change**: 260720-dow0-resolve-or-none-flag
**Created**: 2026-07-20

## Origin

Promptless dispatch (Create-Intake Procedure, `{questioning-mode} = promptless-defer`). The change description was synthesized verbatim from a design conversation with the user (fab-kit's owner); all decisions below were settled in that conversation and are treated as decided, not proposed. Raw input:

> **Title/scope**: `fab resolve` — treat "no active change" as a first-class query result instead of an error, via an opt-in `--or-none` flag, and migrate the five resolve-as-probe call sites to it.
>
> **Problem (root cause)**: Fab's state machine treats "no active change" as a first-class state (`initialized` in `_preamble.md`'s State Table), but `fab resolve` — documented as a pure query — can only fail on it: `src/go/fab/internal/resolve/resolve.go` returns a not-found error ("No active change. Run /fab-new…", resolve.go:192) and every error exits 1. Callers probing for the state must therefore read the error channel as a data channel, and Claude Code renders any non-zero Bash exit as an error (red) — so an expected, valid state renders as an alarm on every `/fab-discuss` run, `/fab-proceed` state detection, `_intake` backlog-ID probe, `fab-new` rename guard, and `fab-adopt` collision guard (five sanctioned resolve-as-probe sites). This is alarm fatigue by design and erodes the one thing the error channel is for.
>
> The classification machinery already exists internally: `internal/resolve/resolve.go:17-20` defines typed sentinels `ErrNotFound` / `ErrAmbiguous` (via `classifiedError` with `Unwrap`), distinct from infrastructure errors (fab/ dir not found, I/O failures). The CLI surface (`src/go/fab/cmd/fab/resolve.go`) flattens all of them to exit 1. The fix is exposing the existing classification at the CLI surface.
>
> **Decisions (settled)**: 1. opt-in flag (working name `--or-none`, final name agent-decidable) mapping state-sentinel failures to `(none)` + exit 0. 2. Sentinel mapping: `ErrNotFound` → `(none)` (bare and override); `ErrAmbiguous` → `(none)` only bare; ambiguous-with-override and infrastructure errors stay non-zero. 3. Token exactly `(none)`, not `none`, not empty. 4. Default (flagless) behavior unchanged — absence-as-data strictly opt-in. 5. Migrate the five probe call sites; keep `git-branch.md` strict. 6. `fab preflight` untouched — preflight = validation gate, resolve = pure query that can answer "none" when asked.
>
> **Sub-decision left open (agent-decidable)**: fab-adopt/fab-new use the `fab change resolve` spelling (flag-free by documented invariant). Either (a) add `--or-none` there too, or (b) migrate those sites to top-level `fab resolve --folder --or-none` — (b) preserves the documented invariant.
>
> **Rejected alternatives**: skill-side `|| echo "(none)"`; distinct exit code (e.g. 2) for not-found; changing bare default to exit 0 + `(none)`.
>
> **Constraints**: CLI change ⇒ `_cli-fab.md` + `_preamble.md` Common-Commands row + Go tests (test-alongside). Touched skills ⇒ SPEC mirrors, whole class swept up front. Canonical sources under `src/kit/skills/` only. Version-lock: binary + kit ship together — flag and skill migration land atomically.

## Why

1. **The pain point**: `fab resolve` is documented as a pure query ("Pure query — converts change reference to canonical output. No side effects." — `_preamble.md` § Common fab Commands), and fab's own state machine names "no active change" a first-class state (`initialized` in `_preamble.md`'s State Table, derived as "config.yaml exists AND no active change"). Yet the CLI can only *fail* on that state: `internal/resolve/resolve.go` returns `notFoundf("No active change. Run /fab-new <description> to start one, or /fab-switch to activate an existing one.")` (resolve.go:192, and again at the zero-candidates fallback), and `main.go` maps every error to a non-zero exit. Five sanctioned skill sites probe for exactly this state (`/fab-discuss` context loading, `/fab-proceed` state detection, `_intake` backlog-ID probe, `fab-new` rename guard, `fab-adopt` collision guard) and must read the error channel as a data channel. Claude Code renders any non-zero Bash exit as an error (red), so an expected, valid state renders as an alarm on every one of these runs.

2. **The consequence if unfixed**: alarm fatigue by design. Every `/fab-discuss` and `/fab-proceed` invocation in a fresh repo state opens with a red error frame for a non-error. Agents and users habituate to red output, eroding the one thing the error channel is for — signaling actual failures (fab/ dir missing, I/O errors, genuine misuse).

3. **Why this approach**: the classification machinery already exists internally — `internal/resolve/resolve.go:17-20` defines typed sentinels `ErrNotFound` / `ErrAmbiguous` via `classifiedError` (with `Unwrap`, matched by `errors.Is`), deliberately distinct from infrastructure errors. Only the CLI surface (`src/go/fab/cmd/fab/resolve.go`) flattens them all to exit 1. Exposing the existing classification at the CLI surface via an opt-in flag fixes the root cause (the state model's "none" state has no success-channel representation) without changing any existing caller's contract.

### Rejected alternatives (from the design conversation)

- **Skill-side `|| echo "(none)"` incantation** — no CLI change needed, but repeats at every call site and swallows infrastructure errors too ("fab/ directory not found" would masquerade as "no active change") — treats the symptom, not the model.
- **Distinct exit code for not-found (e.g. exit 2)** — doesn't help: the harness paints any non-zero red; there is no "expected non-zero" annotation in Claude Code. (Also collides with the toolkit convention where exit 2 = usage error — `docs/memory/distribution/kit-architecture.md` (swon).)
- **Changing bare `fab resolve` default to exit 0 + `(none)` on absence** — silently breaks `$(…)` consumers and removes the hard-stop behavior legitimate callers (e.g. `git-branch.md`) rely on.

## What Changes

### 1. CLI: `--or-none` flag on `fab resolve` (Go)

In `src/go/fab/cmd/fab/resolve.go`:

- Register a new boolean flag `--or-none` (working name; final name agent-decidable, `--or-none` is the settled default choice). It is **NOT** part of the five-flag mutually-exclusive output-mode group — it composes with all of `--id`/`--folder`/`--dir`/`--status`/`--pane`.
- In the resolve path (around `runResolve`'s `resolve.ToFolder(fabRoot, changeArg)` call), when the flag is set and resolution fails, map **state-sentinel** failures to a successful "none" result — print exactly the token `(none)` to stdout, exit 0:
  - `errors.Is(err, resolve.ErrNotFound)` → `(none)`, exit 0 — for **both** bare resolution and an explicit `<change>` override argument.
  - `errors.Is(err, resolve.ErrAmbiguous)` **and** no override argument was given (`changeArg == ""`) → `(none)`, exit 0 — "multiple changes exist, none active" IS the no-active-change state.
  - `ErrAmbiguous` **with** an explicit override → stays a non-zero error (the caller named something and it matched several — a real user error).
  - Infrastructure errors (`resolve.FabRoot()` failure, I/O) → always non-zero, flag or no flag.
- No `internal/resolve` change expected — the sentinels (`ErrNotFound`/`ErrAmbiguous` via `classifiedError` with `Unwrap`, `internal/resolve/resolve.go:17-44`) already exist; this change only exposes them at the CLI surface.
- **Output token is exactly `(none)`** — NOT `none` (a legal 4-char change ID; collision risk with `--id` mode output) and NOT empty output (illegible to the primary consumer — an agent reading a transcript — and hazardous in command substitution: `cd $(fab resolve --dir …)` with empty output cds to `$HOME`). `(none)` replaces the mode-specific output for every output mode.
- `--pane` composition: the sentinel mapping applies to the **change-resolution** step only. A pane-lookup failure after successful resolution (`no tmux pane found for change …`, `not inside a tmux session`) is not a state sentinel and stays non-zero.
- **Default behavior unchanged**: without the flag, absence-as-error remains (output modes are consumed in command substitution and some callers want the hard stop). Absence-as-data is strictly opt-in. Exit-code convention otherwise untouched (0 success / 1 operational / 2 usage).
- Sub-decision resolved as **(b)**: `fab change resolve` stays flag-free (preserving the documented `_preamble.md` invariant "the query flags live on top-level `fab resolve` only"); the two call sites currently using that spelling migrate to top-level `fab resolve --folder --or-none` instead (see § 3). `runResolve` stays the single shared implementation.

### 2. Go tests (test-alongside)

Extend `src/go/fab/cmd/fab/resolve_test.go` with the new flag paths:

- bare not-found + `--or-none` → stdout `(none)`, exit 0
- explicit-override not-found + `--or-none` → stdout `(none)`, exit 0
- bare ambiguous (multiple changes, no active pointer) + `--or-none` → stdout `(none)`, exit 0
- explicit-override ambiguous + `--or-none` → non-zero error (unchanged message)
- infrastructure error (no fab/ root) + `--or-none` → non-zero
- flag absent → all existing behavior byte-identical (regression coverage)
- output-mode composition (at least `--folder --or-none` and `--id --or-none` on the none path both emit `(none)`)

### 3. Skill call-site migration (five probe sites; canonical sources under `src/kit/skills/` only)

| Site | Today | After |
|------|-------|-------|
| `src/kit/skills/fab-discuss.md` Context Loading step 1 (~line 32) | `fab resolve --folder 2>/dev/null` — non-zero ⇒ "No active change" | `fab resolve --folder --or-none` — stdout `(none)` ⇒ "No active change" |
| `src/kit/skills/fab-proceed.md` State Detection Step 1 (~line 46) | `fab resolve --folder 2>/dev/null` — exit-code branch | `fab resolve --folder --or-none` — branch on `(none)` vs folder name |
| `src/kit/skills/_intake.md` backlog-ID probe (~line 67) | `fab resolve --id {id} 2>/dev/null` — failure ⇒ no existing change | `fab resolve --id {id} --or-none` — `(none)` ⇒ no existing change; the exact-ID **equality** compare is preserved unchanged (`(none)` can never equal a 4-char ID, so the compare naturally rejects it) |
| `src/kit/skills/fab-new.md` rename guard (~lines 79, 94–95) | `fab change resolve "$(git branch --show-current)" 2>/dev/null` — "fails" vs "succeeds with another change" | `fab resolve --folder "$(git branch --show-current)" --or-none` — `(none)` vs folder output (sub-decision (b)) |
| `src/kit/skills/fab-adopt.md` collision guard (~line 62) | `fab change resolve "$(git branch --show-current)" 2>/dev/null` succeeds ⇒ STOP | `fab resolve --folder "$(git branch --show-current)" --or-none` outputs a folder (≠ `(none)`) ⇒ STOP |

- With the flag, the `2>/dev/null` suppression at these sites can likely drop too (stderr guidance then only appears on real errors) — **agent-decidable per site** at apply. Note: the success-path stderr note `(resolved from single active change)` still prints on single-change fallback resolution; weigh that per site before dropping suppression.
- **Keep the plain strict form where absence is a genuine STOP**: `src/kit/skills/git-branch.md` bare resolution (its no-argument path displays resolve's stderr and STOPs — that is a hard stop by design). `git-branch.md` is not migrated.
- `fab preflight` untouched — it remains the strict validation gate. The conceptual split is worth documenting where the two commands are described: **preflight = validation gate (non-zero by design); resolve = pure query that can answer "none" when asked** (decision 6).

### 4. Reference docs (CLI change ⇒ doc obligations)

- `src/kit/skills/_cli-fab.md` § fab resolve (extended): document the flag, the sentinel mapping (not-found both forms / ambiguous bare-only / infrastructure never), the exact `(none)` token and its rationale, composition with the five output modes and `--server`, and that `fab change resolve` remains flag-free per the preserved invariant.
- `src/kit/skills/_preamble.md` § Common fab Commands: the `fab resolve` row's canonical form (today `fab resolve --folder 2>/dev/null`) updates to the probe form `fab resolve --folder --or-none`; the row's purpose text gains the none-result semantics. The `fab change` row's note ("the query flags live on top-level `fab resolve` only") stays true under (b) — verify wording still reads correctly.
- `src/kit/skills/_intake.md` prose around the probe (planning-skills mirror text) updated in the same pass.

### 5. SPEC-mirror + aggregate sweep (whole class up front, per code-quality.md § Sibling & Mirror Sweeps)

Every touched `src/kit/skills/*.md` gets its `docs/specs/skills/SPEC-*.md` mirror updated in the same change: `SPEC-fab-discuss.md`, `SPEC-fab-proceed.md`, `SPEC-_intake.md`, `SPEC-fab-new.md`, `SPEC-fab-adopt.md`, `SPEC-_cli-fab.md`, `SPEC-_preamble.md`. Per the code-quality note, on a CLI/command-signature change treat **all** of a touched skill's SPEC mirrors as the sweep class, not just files carrying the literal changed phrase.

Aggregate specs restating resolve semantics: `docs/specs/skills.md` line ~419 restates `/fab-proceed`'s state-detection step 1 (`fab resolve --folder`) — update it. Verified: `docs/specs/architecture.md`, `glossary.md`, `harness-adapters.md` mention only `fab resolve-agent` today (no plain-resolve restatement), but re-grep `fab resolve` (excluding `resolve-agent`) repo-wide at apply before finishing, per the sweep discipline.

### 6. Version-lock / no migration

The `fab` binary and the kit ship together per release, so the flag and the skill-side migration land atomically — no cross-version skew window. No user data (config, `.status.yaml`, archive layout) is restructured — **no `src/kit/migrations/` file is needed**.

## Affected Memory

- `pipeline/change-lifecycle`: (modify) documents the `fab resolve` resolution pattern and pure-query/no-side-effects contract — gains the opt-in none-result semantics (`--or-none`, `(none)` token, sentinel mapping) and the preflight-vs-resolve conceptual split
- `distribution/kit-architecture`: (modify) documents the `fab resolve` flag surface (mutually-exclusive output modes, `fab change resolve` thin wrapper) and the exit-code convention — gains the `--or-none` flag and its exit-0 none path
- `pipeline/planning-skills`: (modify) documents the `_intake` backlog-ID probe form (`fab resolve --id {id}` + equality compare) — probe form changes to `--or-none`
- `pipeline/execution-skills`: (modify) documents `/fab-proceed` state detection step 1 (`fab resolve --folder`) — probe form changes to `--or-none`

## Impact

- **Go**: `src/go/fab/cmd/fab/resolve.go` (flag registration + sentinel mapping in/around `runResolve`, ~20–30 lines) and `src/go/fab/cmd/fab/resolve_test.go` (new flag-path cases). `internal/resolve` untouched. Scope test runs to the `cmd/fab` package first (`code-quality.md` § Test Strategy).
- **Kit skills** (canonical sources): `fab-discuss.md`, `fab-proceed.md`, `_intake.md`, `fab-new.md`, `fab-adopt.md`, `_cli-fab.md`, `_preamble.md` under `src/kit/skills/` — never `.claude/skills/` (gitignored deployed copies).
- **Specs**: 7 SPEC mirrors + `docs/specs/skills.md` (aggregate) — see § 5.
- **Memory**: 4 files at hydrate — see Affected Memory.
- **Behavioral risk**: zero for existing callers — the flag is strictly opt-in and the flagless path is regression-covered. The migrated skill sites change their branch condition from exit-code to stdout-token; each site's surrounding logic (equality compares, STOP conditions) is preserved as specified in § 3.
- **Untouched by design**: `fab preflight` (strict validation gate), `git-branch.md` (absence is a genuine STOP), `fab change resolve` surface (stays flag-free per (b)).

## Open Questions

None — all decisions were settled in the design conversation; the explicitly agent-decidable items (final flag name, sub-decision (a)/(b), per-site `2>/dev/null` drop) are recorded as Confident assumptions (#5, #6, #7) with their settled defaults.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Add opt-in `--or-none` flag to `fab resolve` mapping state-sentinel failures to `(none)` + exit 0; flagless default behavior byte-identical | Settled in design conversation (decisions 1, 4) — absence-as-data strictly opt-in because output modes are consumed in `$(…)` and some callers want the hard stop | S:95 R:75 A:90 D:95 |
| 2 | Certain | Output token is exactly `(none)` — not `none`, not empty output | Settled (decision 3) — `none` is a legal 4-char change ID (collision with `--id` output); empty output is illegible in transcripts and hazardous in command substitution (`cd $(…)` → `$HOME`) | S:95 R:80 A:90 D:95 |
| 3 | Certain | Sentinel mapping: `ErrNotFound` → `(none)` (bare + override); `ErrAmbiguous` → `(none)` bare-only (override-ambiguous stays a real error); infrastructure errors always non-zero | Settled (decision 2) — "multiple changes exist, none active" IS the no-active-change state; a named-but-ambiguous reference is a genuine user error | S:90 R:70 A:90 D:90 |
| 4 | Certain | Migrate exactly the five sanctioned probe sites; `git-branch.md` keeps the strict form; `fab preflight` untouched as the validation gate | Settled (decisions 5, 6) — preflight = gate (non-zero by design), resolve = query that can answer "none" when asked | S:90 R:80 A:90 D:90 |
| 5 | Confident | Final flag name stays `--or-none` | Delegated with a stated default ("final name agent-decidable; `--or-none` is the default choice"); no competing flag-naming convention found in the repo | S:80 R:90 A:75 D:70 |
| 6 | Confident | Sub-decision resolved as (b): migrate the fab-new/fab-adopt sites to top-level `fab resolve --folder --or-none`; `fab change resolve` stays flag-free | Delegated with a stated lean — (b) preserves the documented `_preamble.md` invariant "the query flags live on top-level `fab resolve` only" and avoids widening the wrapper's surface | S:85 R:75 A:85 D:70 |
| 7 | Confident | Drop `2>/dev/null` at migrated sites where stderr then carries only real-error guidance — final call per site at apply | Delegated per site ("agent-decidable per site"); caveat noted: the success-path stderr note `(resolved from single active change)` still prints on single-change fallback | S:75 R:95 A:80 D:60 |
| 8 | Confident | `--or-none` composes with all five output modes (outside the mutually-exclusive group); `(none)` replaces the mode-specific output; post-resolution `--pane` lookup failures stay non-zero | Derived from decision 2's sentinel-only principle (pane-lookup failures are not state sentinels) and decision 3's cross-mode token analysis (`--id`/`--dir` cases argued explicitly) | S:70 R:80 A:85 D:70 |
| 9 | Certain | Sweep obligations: Go tests alongside; `_cli-fab.md` + `_preamble.md` Common-Commands row; all 7 touched-skill SPEC mirrors + `docs/specs/skills.md` swept up front; no migration file (no user-data restructuring) | Constitution Additional Constraints + code-quality.md § Sibling & Mirror Sweeps + context.md § Migrations give a deterministic answer | S:90 R:85 A:95 D:95 |

9 assumptions (5 certain, 4 confident, 0 tentative, 0 unresolved).
