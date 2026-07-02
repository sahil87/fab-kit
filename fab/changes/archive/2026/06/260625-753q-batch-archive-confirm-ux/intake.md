# Intake: Redesign `fab batch archive` confirmation & preview UX

**Change**: 260625-753q-batch-archive-confirm-ux
**Created**: 2026-06-25

## Origin

Initiated from a live design conversation about the ergonomics of `fab batch archive`. The
discussion settled on a complete redesign of the command's flag surface and execution model.
This change was dispatched promptlessly (no interactive questioning) via `/fab-proceed`; every
agreed behavior below is taken verbatim from that conversation, which is the sole source.

> Redesign the `fab batch archive` confirmation & preview UX — replace the opt-in `--all`
> execution flag with a safe default-prompt model (bare invocation lists then prompts
> `Archive these N? [y/N]` with default No), add a `--yes`/`-y` non-interactive escape hatch,
> rename `--list` to `--dry-run`, add a non-TTY guard that refuses (rather than hangs) without
> `--yes`, keep explicit-args archiving prompt-free, error on `--dry-run --yes`, and preserve
> the existing empty-set `No archivable changes found.` exit-0 behavior (finding F49).

**Mode**: one-shot promptless dispatch. All decisions were agreed in the conversation; nothing
was left to the implementing agent's discretion at the design level.

## Why

**Problem.** Today `fab batch archive` requires an explicit `--all` flag to actually archive —
a bare invocation defaults to `--list` (see `src/go/fab/cmd/fab/batch_archive.go:53-55`). That
polarity was set deliberately in change `260612-ye8r` for two reasons: family consistency with
`batch new` / `batch switch` (all three list-by-default), and safety (archive moves are
effectively irreversible *within* the archive loop — once `archive.ArchiveWithBacklog` moves a
folder, the loop does not roll it back). The friction: the routine end-of-change case ("archive
everything that's ready") needs the `--all` flag *every single time*.

**Consequence if unfixed.** The most common archive action stays a two-token command
(`fab batch archive --all`) for what is conceptually "archive the done changes." The flag is
pure ceremony for the common path while still leaving a footgun shape (a destructive-ish bulk op
gated only by a flag the user types reflexively).

**Why this approach.** We want zero-flag ergonomics for the common case WITHOUT making a
destructive-ish bulk op fire silently on a bare command, and WITHOUT losing a non-interactive
path for scripts and agents. A default-No confirmation prompt on the bare command gives the
ergonomics (no flag to remember) while keeping a human in the loop for the irreversible bulk
move; `--yes`/`-y` preserves the automation path. This is the well-understood "list-then-confirm,
with a `--yes` escape hatch" pattern (apt, npm, gh all use it).

**Rationale to preserve in code/docs.** `batch new` and `batch switch` keep `--all` and stay
list-by-default; `batch archive` deliberately *diverges* because it is the one
irreversible-within-loop bulk mutation in the family, so it earns the interactive confirm prompt.
The earlier-considered alternative — invert the default to execute-by-default with a `--try` /
preview flag — was **rejected** in favor of this prompt-based safety model.

## What Changes

This is a CLI command-signature change. Per the constitution's Additional Constraints, it MUST
update `src/kit/skills/_cli-fab.md` and ship corresponding test updates; because it touches the
spec/memory docs describing the command, the SPEC/memory mirror sweep applies.

### 1. Remove the `--all` flag entirely

Delete the `--all` boolean flag and its `allFlag` plumbing
(`batch_archive.go:17,29,34,63-73`). It is fully replaced by the new bare-prompt path plus
`--yes`. No deprecation alias is kept — the flag is removed outright.

### 2. Bare `fab batch archive` → list, then confirm (default No)

A bare invocation (no args, no flags) now:

1. Computes the archivable set (changes with `progress.hydrate` == `done` or `skipped`, via the
   existing `allArchivableNames` / `isArchivable`).
2. Lists the N archivable changes (the existing list output is reused).
3. Prompts `Archive these N? [y/N]` with **default No** — a bare Enter, or any non-`y`/`yes`
   answer, aborts with no action (exit 0, nothing archived).
4. On `y` / `yes`, archives all of them via the existing `archiveLoop`.

Default-No means the safe answer is the one a reflexive Enter selects.

### 3. Add `--yes` / `-y` → skip the prompt and archive all archivable changes

This is the non-interactive path that replaces `--all`. `fab batch archive --yes` (or `-y`)
computes the archivable set and archives all of it with no prompt — identical resolved behavior
to the old `--all`, just renamed to the universal "assume yes / non-interactive" convention.

> The name `--confirm` was explicitly **rejected**: it reads backwards (it sounds like "require
> confirmation," the opposite of its function). `--yes`/`-y` is the universal convention for
> "assume yes / run non-interactively."

### 4. Rename `--list` → `--dry-run`

`fab batch archive --dry-run` lists what *would* be archived; no prompt, no action — exactly the
behavior `--list` has today. Ship a **single** preview flag: do NOT keep both `--list` and
`--dry-run`. `--list` is removed; `--dry-run` is its replacement name.

### 5. Non-TTY guard (load-bearing)

If stdin is **not a TTY** and `--yes` was not passed, the command MUST NOT reach the prompt
(prompting against a non-TTY stdin would hang on EOF, or read an empty line and silently abort —
both are wrong for an unattended runtime). Instead it refuses with guidance and exits non-zero,
e.g.:

```
ERROR: refusing to prompt for confirmation on a non-interactive stdin.
Re-run with --yes to archive non-interactively.
```

`--yes` is the automation escape hatch. This matters because the batch family runs under the
tmux/operator runtime, where stdin is frequently not a TTY. (Detection is via the standard
`isatty(stdin)` check — see Open Questions for the exact helper.)

### 6. Explicit args → archive named changes WITHOUT prompting

`fab batch archive foo bar` archives the named changes with no prompt — naming them IS the opt-in
(the user has already been explicit about *what* to archive). The confirmation prompt applies
**only** to the bare / archive-all path, not to explicit-args invocations. This preserves the
existing explicit-args behavior (resolution, per-change archivability check, warn-and-skip on
unresolvable/not-ready names, `No valid changes to archive.` exit-1 when nothing resolves).

### 7. `--dry-run --yes` is contradictory → error

Passing both `--dry-run` and `--yes` is a usage contradiction ("preview only" vs. "assume yes and
do it"). The command MUST error out (non-zero exit) with a clear message rather than silently
picking one, e.g. `ERROR: --dry-run and --yes are mutually exclusive.`

### 8. Preserve F49 behavior (empty set)

When nothing is archivable, print `No archivable changes found.` and exit 0 — do NOT prompt over
an empty set. This is the behavior documented as finding F49
(`docs/specs/findings/binary-review-2026-06-12.md`) and currently lives at
`batch_archive.go:64-72`. The empty-set check happens *before* any prompt or non-TTY guard so the
benign no-op path is unchanged. (The footer line `Archived 0, skipped 0, failed 0.` from today's
`--all` empty path: confirm whether the bare/`--yes` empty path still prints it — see Open
Questions.)

### Resulting flag matrix

| Invocation | Behavior |
|------------|----------|
| `fab batch archive` (TTY) | List N, prompt `Archive these N? [y/N]` (default No); `y` archives all |
| `fab batch archive` (non-TTY) | Refuse + guidance, exit non-zero (unless `--yes`) |
| `fab batch archive --yes` / `-y` | Archive all archivable, no prompt (replaces `--all`) |
| `fab batch archive --dry-run` | List what would be archived, no prompt, no action (replaces `--list`) |
| `fab batch archive foo bar` | Archive named changes, no prompt |
| `fab batch archive --dry-run --yes` | Error (mutually exclusive), non-zero |
| (nothing archivable) | `No archivable changes found.`, exit 0, no prompt |

### Affected source & doc areas

- **`src/go/fab/cmd/fab/batch_archive.go`** — command implementation: flag definitions
  (remove `--all`/`--list`, add `--yes`/`-y` and `--dry-run`), the prompt, non-TTY detection,
  the `--dry-run --yes` guard, and arg handling. Rewrite the stale `260612-ye8r` rationale
  comment at lines 48-55 to explain the new prompt/`--yes` model.
- **`src/go/fab/cmd/fab/batch_archive_test.go`** — tests (Go changes ship tests; test-alongside).
  Must cover: bare+TTY prompt yes; bare+TTY prompt no/Enter (aborts, exit 0, nothing archived);
  `--yes` archives all without prompt; `--dry-run` lists only; non-TTY-without-`--yes` refusal +
  non-zero exit; explicit-args archive without prompt; `--dry-run --yes` error; empty-set exit 0.
- **`src/kit/skills/_cli-fab.md`** — SPEC mirror of the CLI command reference (constitution-required
  to stay in sync). Update the `fab batch archive` flag documentation at lines ~783 (the family
  signature `[--list] [--all]`) and ~787 (the archive subcommand behavior paragraph).
- **`docs/specs/overview.md`** (line ~106), **`docs/specs/architecture.md`** (line ~439),
  **`docs/specs/assembly-line.md`** (line 128 contains the literal `fab batch archive --all`,
  which MUST be updated) — command tables / examples.
- **`docs/memory/pipeline/change-lifecycle.md`** (line ~156 — the `--list`/`--all` semantics
  paragraph) — update to the new prompt/`--yes`/`--dry-run` model.
- **`docs/memory/runtime/pane-commands.md`** — checked: no current `batch archive` flag
  reference (only `pane map --all-sessions`), so likely no edit needed; re-verify during apply.
- **`docs/specs/findings/binary-review-2026-06-12.md`** finding F49 — the empty-set exit-0
  behavior is **preserved**, so this likely needs only a note (if anything), not a behavior
  change. Update its context only if needed.

## Affected Memory

- `pipeline/change-lifecycle.md`: (modify) Update the `fab batch archive` semantics paragraph —
  replace the `--all` / `--list` description with the bare-prompt (default No) + `--yes`/`-y` +
  `--dry-run` model, the non-TTY guard, the explicit-args-no-prompt rule, the `--dry-run --yes`
  mutual-exclusion error, and confirm the preserved empty-set exit-0 (F49) behavior.

## Impact

- **CLI command signature** — `fab batch archive` flag surface changes (removes `--all` and
  `--list`; adds `--yes`/`-y` and `--dry-run`). Any script or agent currently invoking
  `fab batch archive --all` MUST switch to `--yes`; any `--list` caller MUST switch to
  `--dry-run`. The operator/tmux runtime path runs non-interactively, so those call sites
  (if any) must pass `--yes`.
- **`src/go/fab/cmd/fab/`** — `batch_archive.go` + `batch_archive_test.go` (the only Go files
  touched). Reuses existing helpers (`allArchivableNames`, `isArchivable`, `archiveLoop`,
  `listArchivable`, `resolve.ToFolder`) — the change is concentrated in the flag layer and the
  new prompt/guard logic.
- **Docs/specs/memory** — the SPEC-mirror + doc-table sweep class listed above
  (`_cli-fab.md`, `overview.md`, `architecture.md`, `assembly-line.md`, `change-lifecycle.md`).
- **No migration needed** — this is a behavior/flag change, not a user-data (config /
  `.status.yaml` / archive-layout) restructuring, so no `src/kit/migrations/` file is required.
- **No skill (`src/kit/skills/*.md`) prose change beyond `_cli-fab.md`** is anticipated, but the
  apply agent MUST grep the skill tree for any `batch archive --all` / `--list` literal and sweep
  the whole mirror class (per code-quality.md § Sibling & Mirror Sweeps).

## Open Questions

- Which TTY-detection helper does the codebase already use? (e.g. an existing `isatty`/
  `term.IsTerminal(int(os.Stdin.Fd()))` call elsewhere in `src/go/fab/`.) Reuse the existing
  pattern rather than introducing a new dependency.
- How should the prompt and non-TTY detection be made testable? Cobra's `cmd.InOrStdin()` is the
  natural seam for reading the answer; the TTY check reads `os.Stdin` directly and may need a
  small injection seam (or a test that drives the non-TTY path via a non-terminal stdin) so the
  prompt/guard branches are unit-testable like `archiveLoop` already is.
- Does the bare/`--yes` empty-set path still print the `Archived 0, skipped 0, failed 0.` footer
  (as today's `--all` empty path does at `batch_archive.go:70`), or only `No archivable changes
  found.`? Pick one and keep the test assertion consistent.
- Should `-y` be the only short alias, and should `--dry-run` get a short alias too? (Assumed:
  `-y` for `--yes`, no short alias for `--dry-run` — see Assumptions.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Remove `--all` entirely (no deprecation alias) | Explicitly agreed in conversation; config/conversation deterministically answer it; trivially reversible if wrong | S:95 R:80 A:95 D:95 |
| 2 | Certain | Bare invocation (TTY) lists then prompts `Archive these N? [y/N]` with default No; `y` archives all | Exact wording and default-No polarity agreed verbatim in conversation | S:95 R:75 A:90 D:95 |
| 3 | Certain | Add `--yes`/`-y` as the non-interactive escape hatch (replaces `--all`); reject `--confirm` | Agreed verbatim, including the explicit rejection of `--confirm` with rationale | S:95 R:80 A:95 D:95 |
| 4 | Certain | Rename `--list` → `--dry-run`; ship a single preview flag, do NOT keep both | Agreed verbatim ("Ship a SINGLE preview flag — do NOT keep both") | S:95 R:80 A:90 D:95 |
| 5 | Certain | Non-TTY without `--yes` → refuse with guidance + non-zero exit (do not prompt) | Agreed verbatim and flagged load-bearing; the operator/tmux runtime is non-TTY | S:90 R:65 A:85 D:90 |
| 6 | Certain | Explicit args archive named changes WITHOUT prompting | Agreed verbatim — naming them is the opt-in; prompt applies only to bare/archive-all | S:95 R:80 A:90 D:95 |
| 7 | Certain | `--dry-run --yes` is contradictory → error out (non-zero) | Agreed verbatim | S:90 R:85 A:90 D:90 |
| 8 | Certain | Preserve F49: empty set prints `No archivable changes found.` and exits 0, no prompt | Agreed verbatim; matches current behavior at batch_archive.go:64-72; preserved | S:95 R:85 A:95 D:95 |
| 9 | Certain | CLI-signature change ⇒ update `_cli-fab.md` + ship tests; sweep SPEC/doc mirror class | Constitution Additional Constraints + code-quality.md § Sibling & Mirror Sweeps mandate it | S:90 R:70 A:100 D:90 |
| 10 | Confident | Affected memory limited to `pipeline/change-lifecycle.md` (modify); `runtime/pane-commands.md` likely needs no edit | Grep shows the `--all`/`--list` semantics live in change-lifecycle.md:156; pane-commands.md has no batch-archive flag reference | S:75 R:70 A:80 D:75 |
| 11 | Confident | No migration required — flag/behavior change, not user-data restructuring | No config/`.status.yaml`/archive-layout schema change; context.md § Migrations criteria not met | S:80 R:75 A:90 D:85 |
| 12 | Tentative | `-y` is the sole short alias; `--dry-run` gets no short alias | `-y` is universal; conversation specified `--yes`/`-y` for one and bare `--dry-run` for the other, implying no `--dry-run` short form — but not stated explicitly <!-- assumed: -y only short alias, --dry-run has none — inferred from conversation flag list, not stated --> | S:55 R:75 A:60 D:55 |
| 13 | Tentative | Empty-set path keeps the `Archived 0, skipped 0, failed 0.` footer alongside `No archivable changes found.` | Mirrors today's `--all` empty path (batch_archive.go:70); harmless either way, low blast radius <!-- assumed: keep the zero-count footer on the empty path to match current --all output --> | S:50 R:80 A:65 D:60 |
| 14 | Tentative | Reuse an existing TTY-detection pattern from `src/go/fab/` (e.g. `term.IsTerminal`) rather than adding a new dependency | Codebase-consistency principle (code-quality.md); the exact existing helper is unconfirmed until apply greps for it <!-- assumed: reuse existing isatty/term.IsTerminal pattern; confirm during apply --> | S:55 R:70 A:65 D:60 |

14 assumptions (9 certain, 2 confident, 3 tentative, 0 unresolved).
