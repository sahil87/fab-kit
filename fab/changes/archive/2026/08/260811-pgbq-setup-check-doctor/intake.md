# Intake: fab setup check environment doctor

**Change**: 260811-pgbq-setup-check-doctor
**Created**: 2026-08-11

## Origin

Created by `/fab-proceed` (promptless dispatch) from a synthesized description of the live 2026-08-11 design conversation.

> **Change: `fab setup --check` — a read-only environment probe/doctor for the fab CLI (Go binary).** This is C1 of a two-change series designed in discussion today; C2 (the interactive `fab setup` wizard) is already backlogged as [stpw] and is OUT of scope. C1 builds the probe layer the wizard will later reuse.

Key decisions from the conversation (mode: conversational design session, then promptless dispatch):
- Command name family is `fab setup`; C1 ships the `--check` doctor; the bare interactive wizard is C2/[stpw].
- Read-only invariant is hard: the doctor never writes config, never seeds trust stores, never launches agents.
- No TUI/prompt dependency (Constitution I spirit); C1 has no interactivity at all.
- The probe layer must be structured for reuse by the C2 wizard (internal package boundaries, not a monolithic command function).
- Four surface/depth decisions were explicitly left open and are deferred per promptless mode (see `## Open Questions` and the Unresolved rows in `## Assumptions`).

## Why

Two recurring machine-state bug classes have **no detection today**:

1. **Bottle skew** — the installed release binary and the source tree disagree while reporting the same version string. Concrete incident (2026-08-10): the 2.19.4 Homebrew bottle was built without PR #573's `defaults.yaml` change. `fab --version` said `2.19.4`, the repo at the 2.19.4 tag shipped an agy `interactive_command` default, but the binary's *embedded* defaults (go:embed of `src/go/fab/internal/agent/defaults.yaml`) said agy was headless-only. Unsetting the system-config agy override silently regressed agy panes to headless and had to be reverted. The standing rule — "verify the binary's *behavior*, not the version string" — is currently tribal knowledge living only in session memory. A version-string-only check is explicitly insufficient; the incident proves it: every version indicator agreed while behavior diverged.

2. **Unviable config** — e.g. `dispatch.mode: pane` with no tmux available, or an `agent.workers`/`agent.session` provider whose CLI binary isn't installed on PATH. Today these surface only at dispatch time, as silent ladder descents (`mode: native (descended: pane unavailable: no tmux)`) or as failures mid-pipeline — long after the misconfiguration was made, and in a context where nobody is looking for it.

If we don't build this: both classes keep recurring as multi-hour debugging incidents (the #573 skew cost a regression + revert cycle), and the C2 wizard ([stpw]) has no probe layer to build its "options filtered to DETECTED providers" interview on — detection is C1's deliverable, interviewing is C2's.

Why this approach (a read-only doctor first, wizard second): probing is pure and safe to ship alone (Constitution III — trivially idempotent since it writes nothing); the wizard depends on probe results but not vice versa; and a CI-able exit-code contract makes the doctor useful standalone (pre-flight in scripts, worktree bring-up checks) even before C2 exists.

## What Changes

### New command: `fab setup check` (fab-go binary)

A new top-level cobra command `setup` with a `check` subcommand, registered in `newRootCmd()` (`src/go/fab/cmd/fab/main.go`). The fab router (`src/go/fab-kit/cmd/fab/main.go`) routes any non-lifecycle command to fab-go by default, so no router allowlist change is needed — `setup` is not in `internal.LifecycleCommands` and must not be added there. New files follow the existing per-command pattern: `src/go/fab/cmd/fab/setup.go` + `setup_test.go`.

All output is read-only reporting. **Hard invariant: no writes** — no config mutation, no trust-store seeding, no agent/pane launches, no `.fab-*` state files, no prompts. Running it twice yields the same report (Constitution III).

Surface shape (resolved 2026-08-11 clarify): the doctor is the **`fab setup check` subcommand** (not a `--check` flag). Bare `fab setup` prints a "Yet to be implemented" message (the C2 wizard's reserved seat) and exits without running checks; exact wording is apply's call and MUST be checked against `shll standards` for the standards governing CLI surface/help output (Constitution § Toolkit Standards). <!-- clarified: subcommand form + bare-setup placeholder chosen by user; wizard C2 will occupy bare fab setup -->

### Probe layer as a reusable internal package

The probes live in a new internal package (e.g. `src/go/fab/internal/setupcheck` — final name is apply's call, following the existing `internal/` package map), NOT inline in the command function. Each probe returns structured results (name, status, detail, severity) that the command renders; the C2 wizard ([stpw]) will consume the same structs to filter its interview options ("agent.workers options filtered to DETECTED providers"). The command file owns only rendering and exit-code mapping.

### Provider roster probe

For each provider in the **effective config** — the four built-ins from the embedded `defaults.yaml` (claude, codex, kimi, agy) plus any user-defined providers merged in via the config cascade (`internal/config`, env > system > project > defaults):

- **Binary presence**: resolve the leading executable token of the provider's commands on PATH (`exec.LookPath`). Note the nested-shell forms: agy/kimi `headless_command` values are `sh -c '...'` wrappers — the probe must resolve the *provider's* executable (e.g. `agy`, `kimi`), not `sh`. The existing `internal/agent` / `internal/config` provider types (`config.ProviderConfig`, `agent.ResolveProvider`) are the data source; no new config parsing.
- **Declared capabilities**: which of `interactive_command` / `headless_command` / `native` each provider carries (capability grammar — presence says "here is how", never "select this mode").

Rendered as a table: provider | binary found (path/version where cheap) | interactive | headless | native.

### Environment facts

- `$TMUX` live or not — pane-rung viability. Reuse the existing tmux signal classification in `internal/dispatch` (`TmuxAvailable` / `TmuxAbsent` / `TmuxUnreachable`, `pane_mode.go`) rather than re-probing ad hoc.
- `gh` present, `yq` present, `rk` present (rk is optional tooling — absence is informational, per the `_preamble` fail-silent rk convention, not a failure).

### Version / skew checks

Three version sources exist and can disagree:

1. The fab binary's version (`fab --version` → router prints `fab {version}` + `project: {pin}`); the executing fab-go binary comes from `~/.fab-kit/versions/<ver>/fab-go`.
2. The kit cache version — `~/.fab-kit/versions/<ver>/kit/VERSION` (kit cache layout today: `VERSION`, `bin/`, `migrations/`, `reference/`, `scaffold/`, `skills/`, `templates/`).
3. The project pin — `fab/.fab-version` (the sole project version source since j0qm).

The doctor reports the triplet and flags mismatches. **Additionally**, a behavioral bottle-skew probe (resolved 2026-08-11 clarify): v1 ships the **override-masking check** — for each user-set `providers.<name>.*` (and other capability-bearing) override in the system/project tiers, introspect the binary's own embedded defaults (`src/go/fab/internal/agent/defaults.yaml`, embedded via `//go:embed` in `internal/agent/agent.go`) and flag any override sitting on a key with **no embedded default beneath it**: "this override is load-bearing against your installed binary — unsetting it changes behavior." This is exactly the #573 incident shape and needs no reference artifact. Explicitly rejected mechanisms: env-var overrides cannot test this (the env layer is the top cascade tier — it can only shadow lower tiers, never reveal or remove them, and empty env leaves deliberately fall through per the fp02 read model); artifact-based defaults comparison (shipping a comparable `defaults.yaml` counterpart into the kit tarball, hashing) is **deferred** — the kit cache ships no counterpart today and building one is not worth C1 scope. Network access is out of scope. <!-- clarified: v1 skew depth = version triplet + override-masking via embedded-defaults introspection; artifact comparison deferred; env-var approach analyzed and rejected -->

### Config sanity findings

Read-only analysis of the effective config against the probed environment:

- **Unviable `dispatch.mode`**: the chosen mode would descend at dispatch time, and why — reuse/echo the exact descent-reason strings the ladder already produces (`pane unavailable: no tmux`, `pane unavailable: tmux unreachable`, `pane unavailable: no interactive_command`, `native unavailable`; `internal/dispatch/pane_mode.go`).
- **Depth-knob provider missing**: `agent.session`, `agent.workers`, or any `agent.profiles.<role>.provider` names a provider whose binary is not on PATH.
- **System-tier override masking an embedded-defaults gap**: the exact #573 incident shape — a system-config `providers.<name>.*` override that supplies a capability the binary's embedded defaults lack. Report it as "this override is load-bearing against your installed binary" rather than as an error; the finding is what makes the standing memory-file GOTCHA (`agy override MUST STAY until the binary ships the default`) mechanically checkable.

### Exit-code contract

0 = healthy, non-zero = real problems found (CI-able doctor). The fab-go binary already implements the toolkit convention from backlog [swon] — `run()` in `src/go/fab/cmd/fab/main.go` maps usage errors to exit 2 and operational errors to exit 1 via the `markRunReached` seam — so usage errors need no new work. Severity mapping (resolved 2026-08-11 clarify): a **warnings-only report exits 0** — only failure-severity findings (real problems) produce exit 1. No distinct in-handler exit tier; informational findings (rk absent, load-bearing-override notes) never affect the exit code. <!-- clarified: warnings-only exits 0, user decision -->

### Documentation & tests (constitution obligations)

- `src/kit/skills/_cli-fab.md` gains the new command's signature + behavior section in the same change (Constitution Additional Constraints). The help-dump/router contract tests in `cmd/fab` (`helpdump_test.go`, router-allowlist collision test) pick up the new top-level command and must pass.
- Tests ship in the same change for the command and the probe package (code-review.md: a `.go` change without tests is a must-fix gap). Probes that shell out (LookPath, tmux) follow the existing test seams (fixture repos, PATH stubbing) used by `dispatch_*_test.go` / the fab-kit `doctor_test.go`.
- No skill flow changes: C1 touches no `src/kit/skills/*.md` flow, so no SPEC mirror updates (Constitution 1.5.0: prose-only mentions of `fab setup --check`, if any, need no mirror update).
- No migration: no user data is restructured (the command writes nothing).

### Explicit non-goals

- The interactive wizard (question flow, `fab config set` writes) — C2 / backlog [stpw].
- Trust-store seeding, pane warm-up — C3 candidates per [stpw]'s deferral list.
- Network-dependent checks (release lookup, update checks) — out of scope for the skew check.
- Provider auth/quota probing — confirmed OUT of C1 scope (resolved 2026-08-11 clarify: deferred; revisit at C2/C3 if the wizard needs it).
- Artifact-based defaults comparison for the skew check (shipping a comparable counterpart into the kit tarball) — deferred; v1 is the override-masking introspection above.

## Affected Memory

- `distribution/setup.md`: (modify) currently documents the `/fab-setup` skill bootstrap and the fab-kit seven-check `fab doctor` gate; gains the `fab setup --check` doctor — its probe set, exit contract, and relationship to the existing doctor
- `distribution/kit-architecture.md`: (modify) new fab-go top-level command in the command/exit-code/internal-package map; the embedded-defaults vs kit-cache skew surface
- `distribution/distribution.md`: (modify) version-triplet / bottle-skew detection joins the version-cache and release lifecycle contracts

## Impact

- **Go (fab module)**: new `src/go/fab/cmd/fab/setup.go` + `setup_test.go`; new internal probe package under `src/go/fab/internal/` + tests; read-only consumption of `internal/config`, `internal/agent` (provider/capability data, embedded defaults), `internal/dispatch` (tmux signal, descent reasons), `internal/kitpath` (kit cache resolution, `FAB_KIT_PATH` provenance).
- **Possibly fab-kit / release tooling**: only if the deferred skew-check mechanism decides to ship a comparable defaults artifact into the kit tarball (deferred — could also be cut to what's comparable today).
- **Kit docs**: `src/kit/skills/_cli-fab.md` new command section. Existing contract tests (`helpdump_test.go`, `examples_test.go` conventions, router collision test) updated alongside.
- **No changes** to: router allowlist (`LifecycleCommands`), skill flows/SPEC mirrors, migrations, config schema.
- **Relationship to existing `fab doctor`** (resolved 2026-08-11 clarify): **coexist with distinct jobs**. `fab doctor` (fab-kit binary, `src/go/fab-kit/cmd/fab-kit/doctor.go`: git, fab, bash, yq, jq, gh, direnv) keeps its scope untouched — "is this system good enough to use fab-kit" prerequisites — and its `/fab-setup` bootstrap gate does not change. `fab setup check` owns the fab-kit-*setup* diagnostics (config viability, providers, dispatch mode, versions/skew): what `fab doctor` deliberately does nothing about. No subsume, no cross-invocation; the gh/yq presence overlap is acceptable duplication across two binaries with different jobs. <!-- clarified: coexist, user decision — doctor = system prerequisites, setup check = setup-state diagnostics -->

## Open Questions

None — all five original open questions were resolved in the 2026-08-11 clarify session (see `## Clarifications`).

## Clarifications

### Session 2026-08-11

| # | Question | Answer |
|---|----------|--------|
| 11 | Surface shape: `--check` flag vs `check` subcommand; bare `fab setup` behavior | `fab setup check` subcommand; bare `fab setup` prints "Yet to be implemented" until C2 |
| 12 | Behavioral skew-check depth and mechanism | Version triplet + override-masking check via the binary's own embedded-defaults introspection (the #573 shape, no reference artifact). Env-var overrides analyzed and rejected (top-tier shadowing only; empty leaves fall through). Artifact-based comparison deferred |
| 13 | Provider auth/quota probes in v1? | Deferred — out of C1 scope |
| 14 | Warnings-only report exit code | 0 — only failure-severity findings exit 1; usage errors stay 2 |
| 15 | Relationship to fab-kit `fab doctor` | Coexist, distinct jobs: `fab doctor` = system prerequisites ("good enough to use fab-kit"), unchanged incl. its `/fab-setup` gate; `fab setup check` = setup-state diagnostics |

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Command family is `fab setup`; C1 ships only the `check` doctor subcommand; the interactive wizard is C2/[stpw], out of scope | Discussed — user fixed the name and the C1/C2 split; backlog [stpw] cross-references C1 as the setup-check probe/doctor, in flight 2026-08-11 | S:95 R:65 A:90 D:95 |
| 2 | Certain | Hard read-only invariant: never writes config, never seeds trust stores, never launches agents, no prompts | Discussed — stated as a hard invariant; also Constitution III (idempotent) makes this the only safe shape | S:95 R:80 A:90 D:95 |
| 3 | Certain | No TUI/prompt dependency; C1 has zero interactivity | Discussed — Constitution I spirit; C1 is non-interactive by definition | S:90 R:85 A:90 D:90 |
| 4 | Confident | Probe layer is a separate internal package returning structured results, reused by the C2 wizard; the command owns only rendering + exit mapping | Discussed — "internal package boundaries, not a monolithic command function"; exact package name/API left to apply | S:75 R:70 A:75 D:70 |
| 5 | Certain | Command lands in fab-go (`src/go/fab/cmd/fab/`), reached through the router's default (non-lifecycle) route; `LifecycleCommands` unchanged | Architecture-determined: router routes unknown commands to fab-go; probes need `internal/config`/`internal/agent`/`internal/dispatch` which live in the fab module | S:70 R:75 A:90 D:85 |
| 6 | Certain | Same-change obligations: `src/kit/skills/_cli-fab.md` update + Go tests + help-dump/router contract tests re-verified; `shll standards` checked before finalizing the CLI surface | Constitution Additional Constraints + Toolkit Standards article; code-review.md must-fix rules | S:85 R:80 A:100 D:95 |
| 7 | Confident | Probe set: provider roster (PATH presence via leading executable token of provider commands + declared interactive/headless/native capabilities), environment facts ($TMUX via internal/dispatch signals, gh, yq, rk), version triplet (binary vs kit-cache VERSION vs fab/.fab-version) + behavioral skew check, config-sanity findings (unviable dispatch.mode with descent reason, missing depth-knob provider binary, system-tier override masking an embedded-defaults gap) | Discussed in detail with concrete examples; grounded against internal/agent defaults.yaml, internal/dispatch descent reasons, and the #573 incident shape | S:80 R:65 A:70 D:70 |
| 8 | Certain | Exit-code baseline: 0 healthy / non-zero real problems; usage errors already exit 2 via fab-go's `markRunReached` seam ([swon] convention, implemented in `main.go`) | Verified in source — `run()` maps usage→2, operational→1; only the warnings-severity mapping remains open (row 12) | S:70 R:75 A:90 D:85 |
| 9 | Confident | rk absence is informational, never a failure finding | `_preamble` rk convention: all rk usage is fail-silent optional tooling | S:65 R:85 A:85 D:80 |
| 10 | Confident | No migration ships; no skill-flow/SPEC-mirror changes; any skill mention of `fab setup --check` is prose-only | Nothing restructures user data (read-only command); Constitution 1.5.0 narrows mirror triggers to flow/tool/sub-agent changes | S:65 R:80 A:75 D:70 |
| 11 | Confident | Surface shape: `fab setup check` subcommand; bare `fab setup` prints "Yet to be implemented" until C2 | Clarified — user chose the subcommand form and the bare-setup placeholder (R stays moderate: renaming a released CLI surface later is breaking) | S:95 R:40 A:90 D:90 |
| 12 | Certain | Skew-check v1 = version triplet + override-masking check via embedded-defaults introspection (override on a key with no embedded default beneath = load-bearing); artifact-based comparison deferred; env-var approach rejected (top-tier shadow-only, empty leaves fall through) | Clarified — user asked whether env overrides could test it; analysis showed no, and the introspection mechanism was chosen instead (internal mechanism, swappable later) | S:95 R:60 A:85 D:90 |
| 13 | Certain | Provider auth/quota probes deferred out of C1 | Clarified — user confirmed deferral; trivially reversible (additive later) | S:95 R:85 A:90 D:90 |
| 14 | Confident | Warnings-only report exits 0; only failure findings exit 1; usage errors 2 via existing seam; no new in-handler tier | Clarified — user chose 0 (R moderate: a documented CI exit contract is costly to flip later) | S:95 R:45 A:90 D:90 |
| 15 | Confident | Coexist: `fab doctor` = system prerequisites, unchanged incl. its `/fab-setup` gate; `fab setup check` = setup-state diagnostics; gh/yq overlap accepted | Clarified — user drew the scope boundary (posture revisitable, e.g. later subsume, hence R moderate) | S:95 R:55 A:85 D:90 |

15 assumptions (8 certain, 7 confident, 0 tentative, 0 unresolved).
