# Intake: Binary Truth-Telling, Error-Surfacing & Inference Correctness

**Change**: 260615-jznd-binary-truth-telling
**Created**: 2026-06-15

## Origin

Backlog item `[jznd]` (2026-06-15), **GROUP A of 2**. Distilled from `fab-recurring-lessons` memory — fixes 1, 2, 3, 6 (the Go-binary subset; the skill-prose subset is GROUP B `[qg64]`, sequenced to merge after A). All six code anchors were re-verified against the current `src/go/fab` tree on 2026-06-15 during intake (see § Impact for the verification table). This is a one-shot creation from a fully-specified backlog entry, not a conversational exploration.

> [jznd] GROUP A of 2: Binary truth-telling, error-surfacing & inference correctness (fab Go). GOAL: the fab binary computes state honestly, surfaces write errors, and infers change_type correctly. ACTIONS: (a) change_type inference — fix the `\bfix\b` over-match inside hyphenated compounds AND add an infer-once / respect-explicit-set guard; (b) acceptance truth — derive acceptance_completed from plan.md `## Acceptance` checkboxes on read instead of trusting the hook-maintained counter; (c) F21-residue — surface swallowed `os.WriteFile` errors in `lineEnsureMerge` and fix the lying `scaffoldDirectories` comment; (d) resolve sentinels — add `ErrNotFound`/`ErrAmbiguous` so archive soft-skip stops conflating not-found with ambiguous; (e) prmeta clampNonNeg — the clamp hides real negative impl net on test-heavy PRs; report honestly or annotate (check binary-review Refuted section first). CONSTRAINTS: Go change MUST add test updates + update `src/kit/skills/_cli-fab.md`. SEQUENCING: merge A BEFORE B. FALLBACK: 5 distinct fixes — if intake scores low or review balloons, peel the mirror-lint into its own change and ship it LAST. SRAD: likely below 3.0 — /fab-clarify before /fab-fff.

## Why

The fab binary is the single source of computed truth for the pipeline: it infers `change_type`, tracks acceptance progress, scaffolds project trees, resolves change references, and reports PR impact. Five distinct defects make that truth either wrong or silently lost. These were not hypothesized — each was hit in a real shipped batch effort and recorded in `fab-recurring-lessons` memory:

1. **`change_type` inferred wrong, then re-clobbered (a, 2/a)** — Two compounding sub-bugs. First, the `fix` keyword regex `(?i)\b(fix|bug|broken|regression)\b` treats the hyphen in `must-fix`/`hot-fix` as a word boundary, so an intake describing a *feature* that merely mentions "must-fix" is misclassified `fix`. Second, even after a human corrects it via `fab status set-change-type`, the PostToolUse intake-write hook re-infers on the *next* intake edit and `ApplyChangeType` overwrites the correction unconditionally — there is no infer-once or respect-explicit-set guard. This bit twice in one session (changes `5ewp`, `glwc`: feat→fix), each time silently. **Consequence if unfixed:** every intake refinement after a manual type correction silently reverts it; downstream gate thresholds and changelog grouping key off a lie.

2. **Acceptance progress drifts from reality (b)** — `acceptance_completed` is maintained as a counter in `.status.yaml`, recomputed only when the PostToolUse hook fires on a `plan.md` write. Any path that mutates state *without* firing that hook — `sed` edits (which bypass hooks entirely), direct `.status.yaml` edits, or a `plan.md` checkbox toggled by a tool the hook doesn't observe — leaves the counter stale. The readers (`preflight`, `prmeta` PR-meta) then trust a number that no longer matches the checkboxes on disk. This is the `review-acceptance-checkbox-gap` and the `sed-bypasses-hooks` class, together. **Consequence if unfixed:** PR descriptions and preflight report acceptance progress that contradicts `plan.md`.

3. **Write failures vanish (c)** — `lineEnsureMerge` in `scaffold.go` calls `os.WriteFile` twice and discards both errors, while the *sibling* `scaffoldDirectories` comment claims "Write failures are propagated." A failed scaffold write (disk full, permissions, read-only mount) is silently swallowed and the comment actively lies about it. **Consequence if unfixed:** a half-scaffolded project looks successful; the misleading comment defeats the next reader's audit.

4. **Resolve errors are untyped (d)** — `internal/resolve` returns only `fmt.Errorf` strings; there is no `ErrNotFound` / `ErrAmbiguous`. The archive soft-skip (`archive.go`, `batch_archive.go`) wants to treat "this change is already archived" (idempotent, exit 0) differently from "this name is ambiguous" (real error, exit 1) — but with only string errors it disambiguates by *re-resolving against the archive*, which itself can't tell not-found from ambiguous. The precedent exists: `internal/archive` already defines `ErrAlreadyArchived`. **Consequence if unfixed:** an ambiguous change name during batch-archive is silently soft-skipped as if already archived, masking a real user error.

5. **PR impact hides negative net (e)** — `prmeta.clampNonNeg` floors implementation net-lines at 0. On a test-heavy PR where tests added/deleted exceed the total, the real (negative) implementation net is clamped to `+0`, hiding that the change is net-deletion in production code. The backlog explicitly flags this as *decide-first*: "report honestly or annotate; check binary-review Refuted section before changing" — i.e., the clamp may have been a deliberate defense against a worse failure mode. **Consequence if unfixed (or wrongly fixed):** either PR-meta keeps lying about net impact, or we remove a clamp that was load-bearing and reintroduce the bug it guarded against.

**Why one change, not five:** all five are small, share the "binary tells the truth" theme, touch disjoint packages (hooklib/status, scaffold, resolve, prmeta), and were grouped deliberately in the backlog. The FALLBACK (below) covers the escape hatch if review balloons.

## What Changes

All changes are in the Go binary under `src/go/fab/` (resolve/status/prmeta/hooklib) and `src/go/fab-kit/` (scaffold). Per constitution: every Go change MUST add/update tests and update `src/kit/skills/_cli-fab.md` for any changed command signature or observable behavior.

### (a) + (2/a) — `change_type` inference: tighten regex + infer-once guard

Two sub-fixes, both in the change_type path.

**Sub-fix 1 — regex over-match.** `src/go/fab/internal/hooklib/artifact.go:96`:

```go
{"fix", regexp.MustCompile(`(?i)\b(fix|bug|broken|regression)\b`)},
```

Go's RE2 treats `-` as a non-word char, so `\bfix\b` matches `fix` inside `must-fix`, `hot-fix`, `bug-fix`. Tighten so a hyphen-adjacent occurrence does NOT match. Candidate approaches (RE2 has **no lookbehind/lookahead**, so the classic `(?<![\w-])` is unavailable):
- Match an explicit non-hyphen boundary, e.g. `(^|[^\w-])(fix|bug|broken|regression)([^\w-]|$)` and use submatch groups; or
- Keep `\b` but add a post-match guard that rejects a match whose adjacent char is `-`.

The exact mechanism is a design decision (see Assumptions). Whatever is chosen MUST still match standalone `fix`, `bug-free`-style intent correctly (note: "bug" in "bug-fix" SHOULD arguably still classify `fix` — the intent is to stop *feature* intakes that merely mention "must-fix" in passing from being misclassified; the precise tokens to exclude is itself a clarify point).

**Sub-fix 2 — infer-once / respect-explicit-set guard.** Today `cmd/fab/hook.go:283` (`artifactBookkeeping`) calls `hooklib.InferChangeType` on every intake write and `internal/status/status.go:324` (`ApplyChangeType`) overwrites unconditionally. There is no schema field recording that a human set the type explicitly.

**Resolved design** (clarified at intake): add a new enum field `change_type_source: inferred|explicit` to `.status.yaml` (`internal/statusfile/statusfile.go` read/write/serialize). Semantics:
- `fab status set-change-type` **always** sets `change_type_source: explicit` (alongside the type value).
- The PostToolUse hook (`artifactBookkeeping`) applies inference — and overwrites `change_type` — **only when `change_type_source` is absent or `inferred`**. When it is `explicit`, the hook skips both `InferChangeType` and `ApplyChangeType` for the type (acceptance counting etc. still run).
- Default for a fresh change / absent field is `inferred` (back-compat: pre-existing changes with no field behave exactly as today — re-inference allowed).
<!-- clarified: change_type_source enum (inferred|explicit); set-change-type always marks explicit; hook re-infers only when source != explicit -->

The enum (vs. a bool) was chosen for expressiveness — leaves room for a future `linear` / imported source without a schema migration.

### (b) — Acceptance truth: derive from checkboxes on read

**Current state (verified):** the hook (`cmd/fab/hook.go:322-332`) ALREADY recomputes `acceptance_completed` from `plan.md` `## Acceptance` checkboxes (`CountCompletedSectionItemsBounded`) on every `plan.md` write. The bug is not "the counter is never computed" — it's that the **readers trust the persisted counter** rather than the checkboxes, so any hook-bypassing mutation (sed, direct edit) makes the readers lie.

**Fix:** the read sites derive `acceptance_completed` from `plan.md` `## Acceptance` checkboxes at read time, treating the `.status.yaml` counter as (at most) a cache, not the source of truth. Read sites that consume `Plan.AcceptanceCompleted`:
- `src/go/fab/internal/preflight/preflight.go:111`
- `src/go/fab/internal/prmeta/prmeta.go:321, :329`
- `src/go/fab/cmd/fab/status.go:189` (`acceptance_completed:%d` output)

**Resolved design** (clarified at intake): **keep the write-time counter as a cache, derive on read.** Introduce a shared helper in `internal/status` — `func LiveAcceptance(changeDir string) (done, total int)` (exact name TBD at apply) — that reads `{changeDir}/plan.md` `## Acceptance` and counts checkboxes via the existing `CountSectionItemsBounded` / `CountCompletedSectionItemsBounded`. All read sites prefer the live count over the persisted counter; the `.status.yaml` counter stays as a fast cache (still written by the hook). This survives sed/direct edits because truth is recomputed at read time. Read sites need the change dir to locate `plan.md` — they currently hold the status file, so the helper takes `changeDir` (each caller derives it from the status-file path / preflight-resolved `change_dir`).
<!-- clarified: counter-as-cache + read-time derivation via internal/status.LiveAcceptance(changeDir); readers prefer live count -->
NOTE: `fab score` reads `intake.md` only (per memory `score-binary-source-version-skew`) and does NOT consume acceptance_completed, so score is out of scope for this fix.

### (c) — F21-residue: surface swallowed WriteFile errors + fix lying comment

`src/go/fab-kit/internal/scaffold.go`, `lineEnsureMerge` (~L261-334) swallows two `os.WriteFile` errors:

```go
if len(resolved) > 0 {
    os.WriteFile(dest, resolved, 0644)   // L273 — error discarded
}
...
os.WriteFile(dest, []byte(entry+"\n"), 0644)  // L294 — error discarded
```

Propagate both errors (return them up the `scaffoldTreeWalk` call chain). Fix the `scaffoldDirectories` doc comment (~L12-14) that falsely claims "Write failures are propagated" — make the comment true for the function it documents, or relocate/correct it so it doesn't misdescribe `lineEnsureMerge`. `skills.go` is already clean (per backlog) — no change there.

### (d) — Resolve typed errors

`src/go/fab/internal/resolve/resolve.go` returns only `fmt.Errorf` strings (e.g. `:77` "No active changes found.", `:101` "Multiple changes match", `:104` "No change matches"). Add typed sentinels — `ErrNotFound` and `ErrAmbiguous` — and wrap the existing messages with `%w` so callers can `errors.Is`. Follow the existing precedent: `internal/archive/archive.go:21` already declares `var ErrAlreadyArchived = errors.New(...)`.

Update the soft-skip callers to branch on the typed error instead of re-resolving against the archive to guess:
- `src/go/fab/internal/.../archive.go:66-75` (soft-skip block)
- `batch_archive.go:80-91` (soft-skip block)

So "not found in changes/" (→ check archive, idempotent soft-skip) is distinguished from "ambiguous" (→ real error, surfaced).

### (e) — prmeta clampNonNeg: report honestly OR annotate (DECIDE FIRST)

`src/go/fab/internal/prmeta/prmeta.go:255-260` defines `clampNonNeg`, used at `:228-234` to floor impl `Added/Deleted/Net` at 0 after subtracting tests from totals. On test-heavy PRs this hides a genuinely negative impl net.

**Resolved design** (clarified at intake): **annotate when clamped.** Keep the non-negative clamp on the displayed `Net` (preserving any downstream consumer that assumes non-negative), but when clamping actually occurs, surface the true value in the output — e.g. `Net: 0 (clamped from −42)`. This stops PR-meta from silently hiding net-deletion PRs without removing a possibly load-bearing guard. The apply agent MUST still read the binary-review **Refuted** section before implementing: if it adjudicated the clamp as intentionally lossy, the annotation is the minimal honest change; if it found the clamp safe to remove, escalate via `/fab-clarify` rather than silently switching to "report signed".
<!-- clarified: keep clamp, annotate output when clamping occurs (Net: 0 (clamped from −N)); apply still reads Refuted section first -->

## Affected Memory

- `pipeline/_shared/fab-recurring-lessons` (or current location): (modify) — once shipped, fixes 1/2/3/6 should be struck from the recurring-lessons carry-forward (they will no longer recur). Exact path depends on the memory-domain layout at hydrate time.

<!-- assumed: this is primarily an implementation change to the Go binary; the only spec-level behavior shift is (a/2) change_type explicit-set semantics and (b) acceptance read-time derivation, which may warrant a memory note under the relevant binary/runtime domain. Hydrate will determine exact files. -->

## Impact

**Code areas** (all verified against current tree 2026-06-15):

| Fix | File(s) | Anchor (verified) |
|-----|---------|-------------------|
| (a) regex | `src/go/fab/internal/hooklib/artifact.go` | `:96` pattern; `InferChangeType` |
| (2/a) guard | `cmd/fab/hook.go:283` (`artifactBookkeeping`), `internal/status/status.go:313-326` (`ApplyChangeType`), `internal/statusfile/statusfile.go` (new field) | no `change_type_explicit` field exists today |
| (b) acceptance | readers: `internal/preflight/preflight.go:111`, `internal/prmeta/prmeta.go:321,329`, `cmd/fab/status.go:189`; checkbox parse: `internal/hooklib/artifact.go:155` | hook already counts on write (`hook.go:322-332`) |
| (c) scaffold | `src/go/fab-kit/internal/scaffold.go` `lineEnsureMerge:273,294`; comment `:12-14` | errors swallowed; comment lies |
| (d) resolve | `internal/resolve/resolve.go` (add sentinels), `archive.go:66-75`, `batch_archive.go:80-91` | precedent: `archive.go:21 ErrAlreadyArchived` |
| (e) prmeta | `internal/prmeta/prmeta.go:255-260` (def), `:228-234` (use) | clamp floors net at 0 |

**APIs / signatures:** `fab status set-change-type` gains explicit-set semantics (behavior change → `_cli-fab.md` update). New `.status.yaml` field (schema doc update). Possible new exported helper in `internal/status` for acceptance read-time derivation. `resolve` package gains exported `ErrNotFound`/`ErrAmbiguous`.

**Tests:** each fix needs test coverage — regex non-match for `must-fix`; guard preserves explicit type across a re-infer; acceptance reader reflects a sed-edited checkbox; scaffold WriteFile error propagation; `errors.Is` against resolve sentinels in archive soft-skip; prmeta net under test-heavy diff (per chosen (e) option).

**Constraints / docs:** constitution requires test updates + `src/kit/skills/_cli-fab.md` update for the Go change. `src/kit/` is canonical (never edit `.claude/skills/` directly).

**Sequencing:** merge **before** GROUP B `[qg64]` — if A introduces a SPEC/mirror-lint gate it would otherwise enforce on B mid-flight. Disjoint surfaces (Go here, skill prose in B) so they develop in parallel.

**FALLBACK:** if intake scores low or review balloons, peel the mirror-lint (recurring-lessons #1, the Group B-lint) into its own change and ship it LAST. (Note: the mirror-lint is primarily a GROUP B concern; A's fallback is to scope down to the highest-value subset — likely (a)+(2/a) and (c)+(d) — and defer (b)/(e) if they balloon.)

## Open Questions

Resolved at intake (see § What Changes for the chosen designs):
- ~~**(2/a) guard mechanism**~~ → **enum `change_type_source: inferred|explicit`**; `set-change-type` always marks `explicit`; hook re-infers only when source != explicit. Default/absent = `inferred` (back-compat).
- ~~**(b) derivation shape**~~ → **counter-as-cache + read-time derivation** via a shared `internal/status.LiveAcceptance(changeDir)` helper; readers prefer the live count.
- ~~**(e) clampNonNeg**~~ → **annotate when clamped** (`Net: 0 (clamped from −N)`); apply still reads the binary-review Refuted section first.

- ~~**(a) token scope**~~ → **keep `bug-fix`/`hot-fix`/`bug-free` mapping to `fix`** (they describe fix work); exclude only a passing "must-fix"/"must fix" in an otherwise-feature intake. The exact lexical rule (and its test cases) is decided-and-recorded at apply — the inference still re-runs on every intake write where `change_type_source` is `inferred`/absent (it is NOT a one-time first-write classification).

## Clarifications

### Session 2026-06-15 (bulk confirm)

| # | Action | Detail |
|---|--------|--------|
| 1 | Confirmed | — |
| 2 | Confirmed | — |
| 3 | Confirmed | — |
| 4 | Confirmed | — |
| 5 | Confirmed | — |
| 6 | Confirmed | — |
| 7 | Confirmed | — |

### Session 2026-06-15

| # | Question | Answer |
|---|----------|--------|
| 8 | (a) regex token scope — which hyphenated forms still classify `fix`? | Recommended: keep `bug-fix`/`hot-fix`/`bug-free` → `fix`; exclude only passing "must-fix"/"must fix" in a feature intake. Clarified that inference re-runs on every inferring write (not a one-time first write); exact lexical rule + tests decided at apply. |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Confident | Group all 5 fixes into one change rather than splitting | Clarified — user confirmed | S:95 R:70 A:80 D:75 |
| 2 | Confident | (a) Tighten the `fix` regex via explicit non-hyphen boundary or post-match guard (RE2 has no lookbehind) | Clarified — user confirmed | S:95 R:75 A:80 D:65 |
| 3 | Certain | (c) Propagate both swallowed `os.WriteFile` errors and correct the lying comment | Clarified — user confirmed | S:95 R:75 A:90 D:90 |
| 4 | Confident | (d) Add `ErrNotFound`/`ErrAmbiguous` typed errors; branch soft-skip on `errors.Is` | Clarified — user confirmed | S:95 R:70 A:85 D:75 |
| 5 | Confident | (b) Counter-as-cache + read-time derivation via shared `internal/status.LiveAcceptance(changeDir)`; readers prefer live count | Clarified — user confirmed | S:95 R:60 A:75 D:80 |
| 6 | Confident | (2/a) New enum field `change_type_source: inferred\|explicit`; `set-change-type` always marks explicit; hook re-infers only when source != explicit; default `inferred` | Clarified — user confirmed | S:95 R:55 A:75 D:85 |
| 7 | Confident | (e) Keep clamp, annotate output when clamping occurs (`Net: 0 (clamped from −N)`); apply reads Refuted section first | Clarified — user confirmed | S:95 R:55 A:70 D:80 |
| 8 | Confident | (a) `bug-fix`/`hot-fix`/`bug-free` still map to `fix`; only passing "must-fix"/"must fix" in a feature intake excluded | Clarified — user confirmed (recommended). Exact lexical rule decided + tested at apply | S:95 R:75 A:70 D:55 |

8 assumptions (1 certain, 7 confident, 0 tentative, 0 unresolved).
