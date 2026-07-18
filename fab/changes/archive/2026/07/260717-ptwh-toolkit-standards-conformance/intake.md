# Intake: Toolkit Standards Conformance

**Change**: 260717-ptwh-toolkit-standards-conformance
**Created**: 2026-07-18

## Origin

One-shot `/fab-new` invocation with a fully specified task brief (no prior conversation). Raw input:

> Task: Bring this repo and its tool into conformance with the sahil87 toolkit standards.
>
> Precondition: `shll standards` runs on this machine (if the subcommand is missing, run `shll update`; if it still fails, stop and report — do not proceed from memory or the website). This repo's constitution carries the Toolkit Standards article; this task is the conformance work it mandates.
>
> 1. Enumerate at runtime: run `shll standards`, then `shll standards <name>` for every listed entry. The list is authoritative — do not assume which standards exist or what they require.
> 2. Audit this repo against each standard. For mechanical contracts (machine help output, README/docs-site structure), execute the standard's own verification checklist verbatim. For the principles, assess each numbered principle against the tool's actual behavior — prompts and TTY handling, stdout/stderr separation, --json/--dry-run/--yes coverage, exit codes and error wording, idempotency, output volume.
> 3. Fix what is proportionate here: all mechanical-contract violations, and principle gaps that are small and additive (a missing flag, a misrouted stream, an unhelpful error). Larger gaps that would restructure the tool are NOT for this change — record each as a draft change or issue per this repo's convention and reference it.
> 4. Deliverable: one fab change whose PR body contains a conformance report — one section per standard with PASS or the gaps found, each gap dispositioned as fixed here (with the commit) or deferred to <ref>. Include the shll version audited against (`shll version`'s shll row), since standards are versioned with the shll release. Tests green; if the command tree changed, re-verify the machine-help contract afterward.
>
> Note on the "skill" standard specifically: if this repo has not yet implemented a `<tool> skill` subcommand, that is a known, deferred gap (per the toolkit's phased per-repo adoption — no seven-repo flag-day) — report it as "deferred, not yet adopted" rather than treating it as an in-scope fix for this change.

**Precondition verified at intake time (2026-07-18)**: `shll standards` runs and exits 0. `shll version` reports **shll v0.0.23**. Enumeration returned exactly four standards:

```
principles         foundation   The ten toolkit CLI principles every tool is built against
help-dump          binary       Machine-readable help contract every tool must emit
readme-extraction  repo         README + docs/site structure standard for toolkit repos
skill              binary+repo  Agent skill bundle standard: docs/site/skill.md served by `<tool> skill`
```

## Why

1. **The pain point**: Constitution v1.4.0 (amended 2026-07-18, change `260717-y8it`) added the `### Toolkit Standards` article: this tool "MUST conform to the toolkit's published standards" enumerated by `shll standards`. The article deliberately enumerates nothing itself — the conformance *work* it mandates has not yet been done. Drift is already confirmed: `fab help-dump` emits a `captured_at` field the help-dump standard explicitly forbids ("Do not emit `captured_at`. The capture timestamp is owned by shll.ai"), and the repo's own tests pin that wrong behavior.
2. **The consequence of not fixing**: the constitution's MUST-rule is violated from day one; shll.ai's puller consumes a non-conformant envelope; agents operating `fab` inherit whatever principle gaps exist (prompts that hang, misrouted streams, unbranchable errors); and future CLI/README changes have no conformance baseline to diff against.
3. **Why this approach**: runtime enumeration + per-standard audit + proportionate fix + per-standard report is exactly the shape the constitution article anticipates ("Before changing the CLI surface, help output, README.md, or docs/site/, the change MUST be checked against the standards governing that surface"). Auditing from the live `shll standards` output rather than memory/website keeps the audit pinned to the installed standards version (they version with the shll release).

## What Changes

### 1. Runtime enumeration (apply re-verifies)

The intake-time enumeration above is **grounding, not authority**. At apply entry, re-run `shll standards` and `shll standards <name>` for every listed entry and audit against *that* output. If the subcommand is missing, run `shll update` once; if it still fails, STOP and report — do not proceed from memory, this intake's snapshot, or the website. If the entry list differs from the four above, the runtime list wins (add/remove report sections accordingly).

### 2. Per-standard audit procedure

- **help-dump** (mechanical, binary scope): execute the standard's "Verifying conformance" checklist verbatim against a **freshly built worktree binary** (e.g. `go build ./...` under `src/go/fab`, then run the built artifact) — NOT the installed `fab` (installed is v2.15.5; worktree is ahead — known binary/source skew trap in this repo). Checklist as of v0.0.23: exits 0, valid JSON to stdout only, stderr empty; envelope is exactly `{tool, version, schema_version, root}` with **no `captured_at`**; `completion`/`help`/all hidden commands absent from the tree; `version` reflects the built binary, not a literal; a minimal test pins exit 0 + valid JSON + expected `tool`/`schema_version`. Also verify: `help-dump` is Cobra-`Hidden`, the tree is discovered by walking `rootCmd.Commands()` (never parsing `-h` text), every node carries both raw `text` and structured `short`/`usage`/`path`, and `schema_version` is integer `1`.
- **readme-extraction** (mechanical, repo scope): execute the standard's "Verifying conformance" checklist verbatim against `README.md` + `docs/site/**` (currently `install.md`, `workflows.md`). Checklist as of v0.0.23: README top is `#` H1 → canonical toolkit blockquote → contiguous badge lines → prose (first prose line = the site tagline); grep `](./`, `](../`, `](docs/` — every relative target either points into `docs/site/` from the README, stays inside `docs/site/` between tree pages, or is absolute; ALL images absolute `https://…` everywhere; no mermaid fences destined for the site; no `#gh-*-mode-only` fragments; no `docs/site/` page named `overview`/`readme`/`commands`; README cross-links its `docs/site/` pages and the absolute command-reference URL `https://shll.ai/fab/commands/` (verify the exact tool slug against how shll.ai keys this repo — `fab` vs `fab-kit` — before writing it); footer headings (`Contributing`/`Development`/`Building`/`License`/`Acknowledgements`) mark the slice end — everything site-worthy sits above the first of them. These lints are report-only on the site's side but are **mechanical-contract violations** here: fix all of them in this change.
- **principles** (foundation, 10 numbered principles): assess each against `fab`'s actual behavior — №1 non-interactive by default (every command runnable without a keyboard; confirmations flag-satisfiable via `--yes`/`-y`; non-TTY stdin ⇒ refusal naming the flag, never a hang), №2 stdout=data/stderr=diagnostics (+ `--json` on programmatically consumed commands, stable schemas), №3 help layered + `help-dump` (covered by the mechanical audit), №4 fail fast with actionable errors (what failed / why / what next; exit codes documented and meaningful — `0` success, `1` operational failure, `2` usage error), №5 visible mutation boundaries (read-vs-write clear from name+help; destructive writes support `--dry-run` sharing the real code path + explicit consent per №1), №6 stateless/retry-safe (re-derive state, idempotent commands), №7 compose don't reinvent (shell out to peer CLIs, probe capabilities via `--help`, never assume), №8 graceful degradation (missing optional dependency = skip, not error; TTY-gated color; typed "unavailable" over crash), №9 bounded high-signal output, №10 agent-discoverable documentation (SHOULD — served by the readme-extraction + skill standards). Sample the real command surface: prompt sites, stream routing, flag coverage, exit paths, error wording, re-run behavior, output volume.
- **skill** (binary + repo): `fab skill` does not exist (confirmed: `unknown command "skill" for "fab"`, and there is no `docs/site/skill.md`). Per the standard's own Adoption section ("Phased, per-repo… No tool ships `skill` today… A tool without a `skill` subcommand is not yet in violation"), report **"deferred, not yet adopted"** — do NOT implement it in this change. The consumer-side pairing is already tracked at backlog `[clix]`.

### 3. Confirmed fix: remove `captured_at` from the help-dump envelope

- `src/go/fab/cmd/fab/helpdump.go:17` declares `CapturedAt string \`json:"captured_at"\`` and populates it; the standard: "**Do not emit `captured_at`.** The capture timestamp is owned by shll.ai — a tool cannot know its own capture time. The puller stamps it after capture."
- `src/go/fab/cmd/fab/helpdump_test.go` (lines ~41–45, ~90) asserts the field's presence and RFC3339 validity — tests currently pin the violation.
- Fix: delete the field from the envelope struct and its population; flip tests to assert the envelope is exactly `{tool, version, schema_version, root}` (absence of `captured_at`); keep/extend the minimal conformance-pinning test (exit 0, valid JSON, expected `tool`/`schema_version`). Removing a field the consumer owns and stamps itself is the conformant shape, not a breaking schema change (`schema_version` stays `1`).
- Per the constitution's CLI constraint: if any command *signature* changes, update `src/kit/skills/_cli-fab.md`; output-shape-only changes still ship test updates. `help-dump` is hidden, so this does not alter the visible command tree — but since the change touches `help-dump` output, re-run the standard's verification checklist afterward regardless.

### 4. Audit-determined in-scope fixes

Fix-here bar (task-explicit): **all** mechanical-contract violations (help-dump, readme-extraction), plus principle gaps that are **small and additive** — a missing `--yes`/`--json`/`--dry-run` flag on one command, a misrouted stream, an unhelpful error message, a TTY-guard around a prompt. Constitution constraints apply to every fix: CLI signature changes update `_cli-fab.md` + tests in the same change; any `src/kit/skills/*.md` edit updates its `docs/specs/skills/SPEC-*.md` mirror (sweep the whole mirror class per `fab/project/code-quality.md` § Sibling & Mirror Sweeps); Go changes ship tests.

### 5. Deferred gaps

Principle gaps that would restructure the tool (e.g., a systemic stdout/stderr redesign, wholesale exit-code renumbering, adding `--json` across the full command surface) are NOT fixed here. Record each as a `fab/backlog.md` entry (this repo's convention for deferred work — cf. `[clix]`; use a fresh 4-char ID per entry, or a `/fab-draft` change when the work is already well-shaped) and reference it from the report section as `deferred to [id]`.

### 6. Deliverable: conformance report in the PR body

One fab change (this one). The PR body contains a conformance report with **one section per runtime-enumerated standard**, each section either `PASS` or the gaps found, each gap dispositioned as **fixed here (with the commit hash)** or **deferred to <ref>** (backlog ID / draft change / issue). The report states the shll version audited against (`shll version`'s shll row — v0.0.23 at intake; re-read at report time). The `skill` section reads "deferred, not yet adopted". Tests green (`go test ./...` scoped to touched packages first, then the affected module); if the command tree changed, re-verify the machine-help contract afterward.

## Affected Memory

- `distribution/distribution.md`: (modify) — documents the help-dump surface today; update the envelope description to the conformant shape (no `captured_at`) and record the toolkit-standards conformance posture (audited standards + version, where the report lives)
- Additional entries are audit-dependent: if README/docs-site or CLI-behavior fixes land, extend the hydrate scope to the memory files documenting those surfaces (e.g., `distribution/kit-architecture.md`) — the apply agent should treat this list as a floor, not a ceiling

## Impact

- **Go source**: `src/go/fab/cmd/fab/helpdump.go` + `helpdump_test.go` (confirmed); potentially other `src/go/fab/cmd/fab/*.go` files for small principle fixes (flags, streams, error wording). Binary scope targets the `fab` CLI (`src/go/fab`); the `fab-kit` manager/shim binaries (`src/go/fab-kit/cmd/{fab,fab-kit}`) are not separately audited unless the runtime standards text names them.
- **Repo surfaces**: `README.md`, `docs/site/install.md`, `docs/site/workflows.md` (readme-extraction fixes, if any).
- **Kit/docs**: `src/kit/skills/_cli-fab.md` + `docs/specs/skills/SPEC-*.md` mirrors (only if command signatures change); `docs/memory/distribution/*` at hydrate; `fab/backlog.md` (deferral entries).
- **Process**: PR body carries the conformance report; `shll` v0.0.23 is the audit baseline.

## Open Questions

- None blocking. The concrete fix list beyond `captured_at` is an output of the audit itself (that is the task's shape — audit-then-fix), not a pre-resolvable question; the fix/defer boundary is decided per-gap against the task's explicit bar.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Audit set = the runtime-enumerated standards (currently `principles`, `help-dump`, `readme-extraction`, `skill` @ shll v0.0.23); apply re-runs the enumeration and treats it as authoritative over this intake's snapshot | Task-explicit ("The list is authoritative — do not assume") | S:95 R:90 A:95 D:95 |
| 2 | Certain | Fix-here bar: all mechanical-contract violations + small additive principle gaps; restructuring-sized gaps deferred with references | Task-explicit (item 3) | S:90 R:80 A:90 D:90 |
| 3 | Certain | `fab skill` absence is reported "deferred, not yet adopted" — never implemented in this change | Task-explicit note + the standard's own Adoption section ("not yet in violation") | S:95 R:95 A:100 D:100 |
| 4 | Certain | Remove `captured_at` from the help-dump envelope and update the tests that pin it; `schema_version` stays 1 | Standard forbids emitting it verbatim; consumer stamps it post-capture; field confirmed at helpdump.go:17 | S:90 R:85 A:90 D:95 |
| 5 | Confident | Deferred gaps are recorded as `fab/backlog.md` entries (repo's existing convention, cf. `[clix]`); `/fab-draft` used only when a gap is already well-shaped | Task says "draft change or issue per this repo's convention" — backlog is the observed convention; the split is a judgment call | S:70 R:85 A:85 D:65 |
| 6 | Certain | Conformance is verified against a freshly built worktree binary, not the installed `fab` | Installed binary is v2.15.5 (behind worktree); repo has a documented binary/source-skew trap; trivially reversible verification choice | S:65 R:90 A:90 D:85 |
| 7 | Confident | Binary-scope audits target the `fab` CLI (`src/go/fab`); the `fab-kit`/shim binaries are out of scope unless the runtime standards text names them | Standards' examples name `fab`; shll.ai renders one tool page per toolkit entry; task says "its tool" (singular) | S:60 R:80 A:75 D:70 |
| 8 | Certain | Deliverable is one fab change whose PR body carries the per-standard report (PASS / gaps with fixed-here-commit or deferred-to-ref dispositions) + the shll version row | Task-explicit (item 4) | S:95 R:90 A:95 D:95 |
| 9 | Confident | The reported shll version is re-read at report time (`shll version`'s shll row); if shll updated mid-change, the enumeration+audit re-runs against the new version | Standards version with the shll release (task-stated); mid-change upgrade handling is inferred | S:70 R:85 A:85 D:70 |

9 assumptions (6 certain, 3 confident, 0 tentative, 0 unresolved).
