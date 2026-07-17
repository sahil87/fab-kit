# Toolkit Standards Conformance Report — fab-kit

**Change**: `260717-ptwh-toolkit-standards-conformance`
**Audited against**: **shll v0.0.23** (`shll version`'s shll row)
**Binary audited**: freshly built worktree `src/go/fab/cmd/fab` (NOT the installed `fab` — version-skew trap). Envelope confirmed on the built artifact.
**Standards enumerated at apply entry** (`shll standards`, authoritative over the intake snapshot — identical set):

| Standard | Scope | Governs |
|----------|-------|---------|
| `principles` | foundation | The ten toolkit CLI principles |
| `help-dump` | binary | Machine-readable help contract |
| `readme-extraction` | repo | README + `docs/site/` structure |
| `skill` | binary+repo | Agent skill bundle (`<tool> skill`) |

---

## `help-dump` — 1 gap (1 fixed, 0 deferred)

Executed the standard's "Verifying conformance" checklist verbatim against the built binary.

| Check | Result |
|-------|--------|
| Exits 0, valid JSON to stdout only, stderr empty | PASS |
| Envelope is `{tool, version, schema_version, root}` — **no `captured_at`** | **GAP → FIXED** (was emitting `captured_at`) |
| `completion`, `help`, and all hidden commands absent from the tree | PASS (0 occurrences each; `help-dump` self-excludes) |
| `version` reflects the built binary (from `main.version` ldflags, not a literal) | PASS |
| Minimal test pins exit 0 + valid JSON + expected `tool`/`schema_version` | PASS (extended) |
| help-dump is Cobra-`Hidden`; tree walked via `rootCmd.Commands()` (never `-h` parsing); every node carries `text` + `short`/`usage`/`path`; `schema_version` integer `1` | PASS (78 nodes, 0 missing keys, 0 empty `text`) |

**Gap — `captured_at` emitted (forbidden).** The standard: *"Do not emit `captured_at`. The capture timestamp is owned by shll.ai — a tool cannot know its own capture time. The puller stamps it after capture."* `helpdump.go:17` declared `CapturedAt string \`json:"captured_at"\`` and `dumpDoc` populated it with `time.Now().UTC()`; `helpdump_test.go` pinned the violation (presence + RFC3339 validity + key-order including `captured_at`).

**Fixed here** (files):
- `src/go/fab/cmd/fab/helpdump.go` — removed the `CapturedAt` field from `HelpDoc` and its population in `dumpDoc`; dropped the now-unused `time` import; added an envelope-contract comment. `schema_version` stays `1` (removing a consumer-owned field is not a breaking schema change).
- `src/go/fab/cmd/fab/helpdump_test.go` — replaced the `captured_at`-presence/RFC3339 assertions with a `captured_at`-absence assertion on the encoded bytes (`TestDumpDoc_NoCapturedAt`); fixed the key-order assertion to `tool, version, schema_version, root`; added `TestHelpDumpCmd_MinimalConformance`, an end-to-end run of the real command asserting exit 0 + valid JSON to stdout + empty stderr + expected `tool`/`schema_version` + no `captured_at`.
- `src/kit/skills/_cli-fab.md` — updated the documented envelope (removed `captured_at`), added the no-`captured_at` ownership note, and corrected stale prose (the release-workflow push step was torn down in `260603-mtf9`; shll.ai *pulls* the reference). The SPEC mirror `docs/specs/skills/SPEC-_cli-fab.md` carries only a one-line inventory row (no envelope detail) — no content change required.

**Post-fix verification**: rebuilt binary → `fab help-dump` emits `{tool, version, schema_version, root}`, `has captured_at: false`, exit 0, stderr empty, valid JSON. **help-dump now PASSES.**

---

## `readme-extraction` — 8 gaps (8 fixed, 0 deferred)

Executed the standard's "Verifying conformance" checklist verbatim against `README.md` + `docs/site/{install,workflows}.md`.

| Check | Result |
|-------|--------|
| README top is `#` H1 → toolkit blockquote → contiguous badges → tagline as first prose | PASS |
| Relative targets (`](./`, `](../`, `](docs/`): each points into `docs/site/` (from README), stays inside `docs/site/`, or is absolute; no relative images anywhere | **GAP → FIXED** (8 slice-region relative links leaving the published set) |
| No `#gh-*-mode-only` fragments; site-bound diagrams are committed rendered images referenced absolutely | PASS (each diagram is an absolute `raw.githubusercontent.com/.../docs/img/*.svg` image, immediately followed by its `<details>` mermaid source) |
| No `docs/site/` page named `overview`/`readme`/`commands` | PASS (only `install`, `workflows`) |
| README cross-links its `docs/site/` pages + the absolute command-reference URL | PASS (`docs/site/install.md`/`workflows.md` links present; `https://shll.ai/tools/fab-kit/commands/`) |
| `docs/site/**` closed-set rules (closure, external absolute, all images absolute, README→docs/site natural) | PASS (inter-page links use `./install.md`/`./workflows.md`; all other links absolute GitHub-blob; no images) |

**Gap — 8 relative links leaving the published set, in the README slice region.** The site slice runs from the head to the first denylisted heading (`## Development`, line 660). Within it, 8 links pointed to targets outside the published set (README slice + `docs/site/**`) using relative paths, which 404 on shll.ai (`ReadmeSlice.astro` does no relative-base rewrite for non-`docs/site/` targets):
- `docs/specs/companions.md` (Companion tools)
- `docs/specs/{assembly-line,overview,user-flow,skills,srad,glossary}.md` (Learn More — 6 links)
- `CONTRIBUTING.md` (Learn More "Contributing" bullet, line 658)

**Fixed here**: `README.md` — all 8 rewritten to absolute `https://github.com/sahil87/fab-kit/blob/main/<path>` URLs. The below-boundary `CONTRIBUTING.md` link (line 670, GitHub-only, not in the slice) and the sanctioned auto-rewritten `docs/site/*.md` links (lines 11/88/228) were correctly left untouched.

**Command-reference tool-slug check** (intake-flagged as `fab` vs `fab-kit` ambiguity): resolved by verifying shll.ai's actual routing — pages render under `tools/fab-kit/` (confirmed in `sites/astro-starlight-terminal1/{src/content/docs,dist}/tools/fab-kit/commands`). The README's `https://shll.ai/tools/fab-kit/commands/` is **correct** — the standard's `/<tool>/commands/` is a template with `<tool>` = `tools/fab-kit` for this repo. Not a violation.

**readme-extraction now PASSES** (all mechanical-contract violations fixed).

---

## `principles` — 4 gaps (0 fixed here, 4 deferred)

Each of the ten principles was assessed against `fab`'s actual behavior (prompt/TTY sites, stdout/stderr routing, `--json`/`--dry-run`/`--yes` coverage, exit paths, error wording, idempotency, output volume) across `cmd/fab/*.go` + `internal/`.

| # | Principle | Verdict | Disposition |
|---|-----------|---------|-------------|
| 1 | Non-interactive by default | **PASS** | The sole prompt (`batch_archive.go:126`) is fully guarded: `--yes`/`-y`, non-TTY refusal naming the flag (`isStdinTTY` seam), mutual-exclusion. No other prompt/hang site (dispatch/pane/operator reads are pipe-fed data, not confirmations); `batch_switch` forces git probes non-interactive (`GIT_TERMINAL_PROMPT=0`, `BatchMode=yes`). |
| 2 | stdout=data / stderr=diagnostics | **GAP (routing PASS)** | Stream routing is clean (no diagnostics on stdout, no data on stderr). Gap: the `fab status` query subcommands emit hand-parsed `key:value` lines with no `--json`. Whole-surface addition → **deferred `[jx4w]`**. (YAML emitters `preflight`/`impact`/`score` already offer a stable machine-parseable schema — not flagged.) |
| 3 | Help is a published contract | **GAP** | help-dump conformant (above); layered `-h` has `Short` + occasional `Long` but **no command uses cobra `Example:`** — help lacks example invocations. Spans ~7 user-facing commands → **deferred `[b91h]`**. |
| 4 | Fail fast with actionable errors | **GAP** | Error *wording* PASS (what-failed/why/what-next throughout). Exit codes: `main.go:52-55` maps **every** error to `os.Exit(1)` (no `SetFlagErrorFunc`) — verified: usage errors (`fab score` no-arg, unknown flag) and operational errors both exit 1, so the toolkit's `2` = usage-error convention is unmet. Restructuring (must reconcile with existing domain-specific exit-2: `memory_index` destructive-loss=2, pane 2/3) → **deferred `[swon]`**. |
| 5 | Visible mutation boundaries | **PASS** | Read-vs-write clear from verb naming + help; the one destructive bulk write (`batch archive`) has `--dry-run` sharing the real code path + explicit consent; `config init` refuses to overwrite. |
| 6 | Stateless / retry-safe | **PASS** | Commands re-derive state each run (status re-loads `.status.yaml`; preflight self-heals); idempotent mutators (`add-issue`/`add-pr`, archive counts already-archived as skipped); aligns with Constitution III. |
| 7 | Compose, don't reinvent | **PASS (caveat)** | Shells out to `wt`/`gh`/`git`/`tmux` rather than reimplementing; forces subprocesses non-interactive. Caveat: `batch_switch` hardcodes `wt`'s flag contract rather than probing `wt --help`, but degrades safely (offline probe → positional → wt re-checks and errors visibly). Not a violation given the fallback — no fix, no defer. |
| 8 | Graceful degradation | **PASS** | Missing optional deps = typed error / skip, not crash (no-tmux typed errors; `log command` best-effort exit 0; per-change archive warn-and-continue; missing config → built-in default). No ANSI color emitted, so TTY-color-gating is vacuously satisfied. |
| 9 | Bounded, high-signal output | **GAP** | Unbounded log output is capped (`dispatch logs --tail N`). Gap: no command offers `--quiet`; `batch archive`/`batch switch` print per-change progress with no suppress. Additive on two commands (new flag + mirror + tests each) → **deferred `[o5f9]`**. |
| 10 | Agent-discoverable documentation | **PASS (SHOULD)** | Served by readme-extraction (now conformant) + skill (below); the hidden `help-dump` provides the machine tree; the constitution mandates `_cli-fab.md` upkeep on CLI changes. |

**Fixed here**: none (no gap met the "small and additive — a missing flag on one command / one misrouted stream / one unhelpful error" bar; routing, error wording, TTY-guarding, `--dry-run`/`--yes` were already conformant).

**Deferred** (each a `fab/backlog.md` entry, fresh 4-char ID, referencing this change and shll v0.0.23):
- `[swon]` — P4: route cobra usage errors to exit 2, reconciled with existing domain-specific exit-2 uses.
- `[jx4w]` — P2: `--json` for the `fab status` query subcommands (stable object schema).
- `[b91h]` — P3: `Example:` blocks on multi-flag user-facing commands.
- `[o5f9]` — P9: `--quiet` on `batch archive` + `batch switch`.

---

## `skill` — deferred, not yet adopted

`fab skill` does not exist (no subcommand; no `docs/site/skill.md`). Per the standard's own Adoption section — *"Phased, per-repo… No tool ships `skill` today… A tool without a `skill` subcommand is not yet in violation"* (principle №10 is a SHOULD). **Not implemented in this change** by design. The consumer-side pairing is already tracked at backlog `[clix]`.

---

## Summary

| Standard | Result |
|----------|--------|
| `help-dump` | PASS after fix — 1 gap fixed (`captured_at` removed: `helpdump.go` + `helpdump_test.go` + `_cli-fab.md`) |
| `readme-extraction` | PASS after fix — 8 slice-region relative links made absolute (`README.md`) |
| `principles` | 6 PASS (1,5,6,7,8,10) + 4 deferred (P2→`[jx4w]`, P3→`[b91h]`, P4→`[swon]`, P9→`[o5f9]`); routing/wording/TTY/consent all conformant |
| `skill` | deferred, not yet adopted (standard's Adoption section; consumer half at `[clix]`) |

All mechanical-contract violations (help-dump + readme-extraction) are **fixed here**. All principle gaps are additive-multi-command or restructuring-sized and are **deferred** with backlog references. Tests green.
