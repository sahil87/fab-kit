# Intake: Remove wt/idea Homebrew Dependencies with Graceful Degradation

**Change**: 260720-nnda-remove-wt-idea-brew-deps
**Created**: 2026-07-20

## Origin

One-shot `/fab-new` invocation with a fully pre-made four-part design (all key decisions arrived resolved in the prompt):

> Remove the wt and idea Homebrew dependencies from fab-kit and add graceful degradation for their absence. Four changes: (1) delete both depends_on lines (sahil87/tap/wt, sahil87/tap/idea) from .github/formula-template.rb. (2) In src/kit/skills/_cli-external.md, update the "Absent-binary discipline" section (~lines 121-145) and the wt/idea section headers (~lines 151, 195) that say "assumed present — bare": move wt and idea out of the assumed-present class and gate their use-time delegations (wt skill, idea help-dump, lines ~143-144) with command -v exactly like the rk/hop lines directly above them. (3) Add an exec.LookPath("wt") guard at the top of the run loop in src/go/fab/cmd/fab/batch_new.go (~line 118) and src/go/fab/cmd/fab/batch_switch.go (~line 122), returning a clear error like: wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt (one upfront error instead of N cryptic per-item exec failures; follow the existing LookPath pattern in src/go/fab-kit/internal/prereqs.go). (4) In src/kit/skills/fab-operator.md, add one preflight command -v wt probe near the top (wt create is the first action for any request; if absent, stop with the install hint) instead of gating each call site (lines ~35, 110, 436), and gate the idea show pre-step (~line 541) with a command -v graceful skip. Edit only the source skills under src/kit/skills/ — the .claude/skills mirrors are generated. Rationale: toolkit-wide decision to remove inter-tool brew dependencies; the absent-binary policy doc premise (wt/idea guaranteed by brew depends_on) no longer holds.

All file/line claims were verified against the working tree at intake time (line references below are current-tree accurate; the prompt's approximations are noted where they differ).

## Why

1. **The pain point**: The shll toolkit has made a toolkit-wide decision to remove inter-tool Homebrew dependencies. fab-kit's formula currently forces `wt` and `idea` installs via `depends_on "sahil87/tap/wt"` / `depends_on "sahil87/tap/idea"` (`.github/formula-template.rb:7-8`). Once those lines are deleted, the entire premise of the current absent-binary policy — "`wt`/`idea` are guaranteed present because brew installs them transitively" — no longer holds, yet skills invoke them **bare** and `fab batch new`/`switch` exec `wt` blindly per item.
2. **The consequence of not doing it (or doing only half)**: Removing `depends_on` without graceful degradation leaves fresh installs (which no longer get wt/idea) hitting failures in the worst possible shapes: `fab batch new --all` produces N cryptic per-item `wt create: exec: "wt": executable file not found` failures instead of one clear upfront error; the operator's first action for any request (`wt create`) fails mid-choreography; `_cli-external.md`'s "assumed present — bare" doctrine instructs agents to surface raw `command not found` errors. Conversely, not removing the deps leaves fab-kit non-conformant with the toolkit decision.
3. **Why this approach**: Degrade at the *entry points*, not at every call site. One `exec.LookPath("wt")` guard at the top of each batch run loop (following the existing `prereqs.go` LookPath pattern) gives one actionable error instead of N. One preflight `command -v wt` probe at the top of the operator skill (wt create is the first action for any request) beats gating its ~3 call sites individually. The skill-doc delegations (`wt skill`, `idea help-dump`) get the same `command -v` gate the `rk`/`hop` lines directly above them already use — a proven, in-file pattern.

## What Changes

### 1. Formula template: delete both `depends_on` lines

`.github/formula-template.rb` lines 7-8, delete exactly:

```ruby
  depends_on "sahil87/tap/wt"
  depends_on "sahil87/tap/idea"
```

Nothing else in the formula changes (binaries installed, test block, URLs are untouched). This is the template the release workflow stamps (`VERSION_PLACEHOLDER`/`SHA_*`); no tap-repo edit happens in this change — the next release publishes the dependency-free formula. Note: brew does not uninstall existing deps on upgrade, so already-installed users keep wt/idea; the degradation paths below matter primarily for fresh installs.

### 2. `src/kit/skills/_cli-external.md`: reclassify wt/idea out of "assumed-present"

Current state (verified):
- **§ The `skill` delegation** (lines ~44-58): prose says "**bare** for `wt`/`idea` (assumed-present)" and the code block at lines 54-55 shows `wt skill` / `idea skill` bare.
- **§ Absent-binary discipline (two install classes)** (lines ~118-145): defines the two-class model — "**Assumed-present — `wt`, `idea`.** These are Homebrew `depends_on` of `fab-kit` … Invoke them **bare**" vs. "**Genuinely-optional — `rk`, `hop`**" — and its closing code block (lines 141-144) shows the gated rk/hop lines directly above bare `wt skill` / `idea help-dump` lines.
- **§ wt header note** (line ~159): "read them at use-time via `wt skill` (usage) / `wt help-dump` (flags), **assumed present — bare**, per § Reference Model"; line ~153 says "Installed system-wide via `brew install fab-kit`."
- **§ idea section** (line ~197): same "assumed present — bare" phrasing; intro line ~195 says "Installed system-wide via `brew install fab-kit`".

New state:
- The two-class model collapses to **one class**: all four owned binaries (`wt`, `idea`, `rk`, `hop`) are separate sibling formulas that may legitimately be absent; every use-time delegation (`skill` and `help-dump`) is `command -v`-gated and fails silently. The section retitles accordingly (e.g., "Absent-binary discipline" without "(two install classes)").
- The use-time delegation lines for wt/idea become exactly the rk/hop shape:

```sh
command -v wt   >/dev/null 2>&1 && wt skill        # gated, fail silently
command -v idea >/dev/null 2>&1 && idea help-dump  # gated, fail silently
```

- The "Homebrew `depends_on` of `fab-kit`" / "Installed system-wide via `brew install fab-kit`" claims are replaced with standalone-formula install pointers: `brew install sahil87/tap/wt` and `brew install sahil87/tap/idea`.
- The distinction that survives: `wt` is still *functionally required* for worktree-based flows (operator spawning, `fab batch new`/`switch`) — those entry points stop with an install hint (changes 3 and 4) rather than silently skipping. The doc should state this so "fail silently" (delegations) vs. "stop with hint" (functional entry points) is not read as a contradiction.
- The § Reference Model summary rows and the `help-dump` delegation note ("scopes to the same four owned binaries") keep their content but drop the class asymmetry.

### 3. Go: upfront `exec.LookPath("wt")` guards in `fab batch new` / `fab batch switch`

- `src/go/fab/cmd/fab/batch_new.go`: the per-item loop starting ~line 96 execs `wt create` per backlog ID (~line 118). Insert the guard after the tmux check / before the loop (one upfront error, not N per-item failures):

```go
if _, err := exec.LookPath("wt"); err != nil {
    return fmt.Errorf("wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt")
}
```

  `batch_new.go` does not currently import `os/exec` — add it (`batch_switch.go` already imports it for `exec.Command`).
- `src/go/fab/cmd/fab/batch_switch.go`: same guard before its per-change loop (~line 93; its `wt create` exec is at ~line 122), with message text `wt is required for 'fab batch switch' — install it via: brew install sahil87/tap/wt`.
- Pattern precedent: `src/go/fab-kit/internal/prereqs.go` (`exec.LookPath` + "Install with: brew install …" error shape).
- **Tests** (required by review policy "Go changes ship tests"): add cases to `batch_new_test.go` / `batch_switch_test.go` asserting the upfront error when `wt` is absent from PATH (e.g., `t.Setenv("PATH", …)` with a dir that lacks wt) and that the guard sits before any per-item work.
- **`src/kit/skills/_cli-fab.md` § fab batch** (~line 1054): note the new upfront wt requirement/error for `new`/`switch` (behavior addition to documented commands; no signature/flag changes).

### 4. `src/kit/skills/fab-operator.md`: one preflight probe + gated idea pre-step

- **Preflight probe (one, near the top)** — not per call site. `wt create` is the operator's first action for any new request (§ Spawn-in-worktree, line ~35; other call sites at lines ~110 and ~436 stay unmodified). Add a single preflight check near the top of the skill (with the startup/session-setup steps):

```sh
command -v wt >/dev/null 2>&1 || # STOP: wt is required for operator spawning — install it via: brew install sahil87/tap/wt
```

  If absent, the operator stops with the install hint rather than failing mid-spawn-sequence.
- **idea pre-step gating** (line ~541, the Working-a-Change "Backlog ID or Linear issue" row): the `idea show <id>` lookup becomes `command -v idea`-gated with a graceful skip — when `idea` is absent, skip the lookup and proceed to spawn `/fab-new <id>` unchanged (`/fab-new` resolves the backlog ID itself from `fab/backlog.md`, so nothing is functionally lost).

### 5. Mirror & sibling sweep (same change, per code-quality § Sibling & Mirror Sweeps)

Files carrying the now-false "depends_on / assumed-present / four binaries via brew install fab-kit" claims, verified by repo-wide grep:

- `docs/specs/skills/SPEC-_cli-external.md` — lines 5 and 17 restate the two-install-class model ("bare for wt/idea, command -v-gated fail-silent for rk/hop") → update to the one-class model.
- `docs/specs/skills/SPEC-fab-operator.md` — spawn-sequence prose (§6 summary, line ~40) → mention the new preflight probe and gated idea pre-step.
- `docs/specs/companions.md:3` — "fab-kit's Homebrew formula declares them as dependencies (`depends_on …`), so `brew install sahil87/tap/fab-kit` lands all four binaries … in a single step" → rewrite to standalone installs + graceful degradation summary.
- `docs/specs/architecture.md:429` — "The formula also declares `depends_on` for the standalone `wt` … and `idea` … CLIs" → update.
- `docs/site/install.md` (~lines 22-30) — the four-binary table + "declared as Homebrew dependencies of fab-kit" claim → wt/idea become separately-installed companions (`brew install sahil87/tap/wt`, `brew install sahil87/tap/idea`). Constitution § Toolkit Standards: docs/site/ and CLI error-surface changes MUST be checked against `shll standards` before ship.

Memory files (updated at hydrate, listed in Affected Memory): `distribution/distribution` (the formula `depends_on` SHALL-claim and fresh-install four-binaries behavior), `distribution/kit-architecture` (batch-command behavior), `runtime/operator` (preflight probe).

## Affected Memory

- `distribution/distribution`: (modify) Formula section — drop the `depends_on "sahil87/tap/wt"` / `"sahil87/tap/idea"` SHALL-claims and the "fresh install lands four binaries" behavior; record the standalone-install model and the graceful-degradation surfaces.
- `distribution/kit-architecture`: (modify) § Batch Commands — record the upfront `exec.LookPath("wt")` guard + error contract on `fab batch new`/`switch`.
- `runtime/operator`: (modify) Record the single preflight `command -v wt` probe (stop-with-hint) and the `command -v idea`-gated graceful-skip on the backlog pre-step.

## Impact

- **Go**: `src/go/fab/cmd/fab/batch_new.go` (+ `os/exec` import), `batch_switch.go`, `batch_new_test.go`, `batch_switch_test.go`. No command signatures or flags change; one new early-error behavior on each command.
- **Kit skills (canonical sources only)**: `src/kit/skills/_cli-external.md`, `src/kit/skills/fab-operator.md`, `src/kit/skills/_cli-fab.md` (§ fab batch note). `.claude/skills/` mirrors are generated by `fab sync` — never edited.
- **Release surface**: `.github/formula-template.rb` — next release publishes a dependency-free formula. User-visible install-set change (fresh installs no longer get wt/idea) → release-notes-worthy; existing installs unaffected (brew keeps installed deps).
- **Docs**: `docs/specs/skills/SPEC-_cli-external.md`, `docs/specs/skills/SPEC-fab-operator.md`, `docs/specs/companions.md`, `docs/specs/architecture.md`, `docs/site/install.md`.
- **Standards**: `shll standards` conformance check required before ship (docs/site/ + CLI error text touched).
- **Not in scope**: no migration (no user-data restructuring); `fab doctor` / `prereqs.go` gain no wt/idea checks; operator call sites at lines ~110/~436 are not individually gated (the single preflight probe covers them); the tap repo itself is untouched.

## Open Questions

None — the invocation resolved all four change areas with specific files, mechanisms, and error text; residual choices are graded below.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Edit only canonical sources under `src/kit/skills/`; never touch `.claude/skills/` deployed copies | Explicit in the request and Constitution V / code-quality anti-pattern | S:100 R:90 A:100 D:100 |
| 2 | Certain | Formula change is the template only (`.github/formula-template.rb`); the tap repo is untouched — the release workflow publishes the dependency-free formula | Template is the canonical source the workflow stamps; explicit line references given | S:85 R:90 A:90 D:90 |
| 3 | Certain | Guard placement: single `exec.LookPath("wt")` check after the tmux check / before each per-item loop (not inside it); `batch_new.go` adds the `os/exec` import; error strings exactly as specified per command name | Request specifies "one upfront error instead of N cryptic per-item exec failures" + prereqs.go pattern; verified batch_switch.go already imports os/exec | S:90 R:85 A:90 D:85 |
| 4 | Certain | Go guards ship tests in `batch_new_test.go`/`batch_switch_test.go` (PATH-manipulation cases) and `_cli-fab.md` § fab batch gains the wt-requirement note | code-review.md: "Go changes ship tests" is must-fix; constitution: CLI behavior changes update `_cli-fab.md` | S:70 R:85 A:90 D:85 |
| 5 | Certain | Scope excludes `fab doctor`/`prereqs.go` wt-idea checks, per-call-site operator gating (lines ~110/~436), and any migration | Request enumerates exactly four changes and says "one preflight probe … instead of gating each call site"; no user-data restructuring occurs | S:85 R:90 A:80 D:75 |
| 6 | Confident | `_cli-external.md` collapses the two-class model into ONE gated class (all four owned binaries `command -v`-gated fail-silent for `skill`/`help-dump` delegations), while noting wt's functional entry points (batch, operator) stop-with-hint instead | Request says gate wt/idea "exactly like the rk/hop lines directly above them"; keeping a vestigial two-class table with identical behavior would be noise, but the exact section restructure is mine | S:80 R:75 A:70 D:65 |
| 7 | Confident | idea-absent graceful skip = skip the `idea show <id>` lookup and spawn `/fab-new <id>` unchanged — `/fab-new` resolves backlog IDs from `fab/backlog.md` itself, so no functionality is lost | "Graceful skip" specified; /fab-new's Step 0 backlog resolution makes the lookup advisory | S:75 R:80 A:80 D:70 |
| 8 | Confident | Mirror sweep class (SPEC-_cli-external, SPEC-fab-operator, companions.md, architecture.md:429, docs/site/install.md) is updated in this same change; memory files at hydrate | Not in the request, but code-quality § Sibling & Mirror Sweeps + review rules make SPEC-mirror sync must-fix; class enumerated by repo-wide grep for depends_on/assumed-present claims | S:65 R:75 A:90 D:80 |
| 9 | Confident | `shll standards` conformance check runs before ship (docs/site/install.md + CLI error text are governed surfaces) | Constitution § Toolkit Standards MUST-rule; which specific standards apply is read at apply time from the live enumeration | S:60 R:85 A:85 D:80 |
| 10 | Confident | Install pointers in docs/error text use the standalone formulas (`brew install sahil87/tap/wt`, `brew install sahil87/tap/idea`); wt/idea remain recommended companions, not dropped from docs | Request's error text names `brew install sahil87/tap/wt`; distribution memory confirms standalone formulas exist in sahil87/tap since 1.7.0 | S:70 R:80 A:85 D:80 |

10 assumptions (5 certain, 5 confident, 0 tentative, 0 unresolved).
