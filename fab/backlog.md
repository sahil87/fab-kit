## Open

- [ ] [ngaw] 2026-02-23: Quality gate - how to decide which PR has had deep thought vs just surface level?
- [ ] [v34t] 2026-02-23: A timeline or user journey mermaid diagram showing which commands are typed in the main repo vs the worktree
- [ ] [lc3m] 2026-03-01: Switch fab/.kit/LICENSE from PolyForm Internal Use to MIT. Reasoning: fab-kit is a process (prompts, markdown, shell scripts) — not protectable IP in the copyright sense. The US Copyright Office ruled prompts convey unprotectable ideas. PolyForm Internal Use blocks community contributions and ecosystem growth while providing only illusory control. MIT gives attribution (copyright notice preserved), maximum adoption, zero friction, and lets the community keep the tool evolving — the real moat is speed and opinionation, not licensing restrictions.

## Go Migration — Phase 1 (fab binary)

Dependency graph determines ordering. Each step can begin once its dependencies are done.

```
Step 1: gm01 (scaffold)
           │
     ┌─────┼─────────┐
     ▼     ▼         ▼
Step 2: gm02    gm03    gm04      ← parallel: status, resolve, log (no inter-dependencies)
     │     │         │
     └─────┼─────────┘
           ▼
Step 3: gm05                      ← preflight: calls status + resolve + log in-process
           │
     ┌─────┴─────┐
     ▼           ▼
Step 4: gm06    gm07              ← parallel: change + score (both depend on preflight/status)
     │           │
     └─────┬─────┘
           ▼
Step 5: gm08                      ← archive: depends on change + status
           ▼
Step 6: gm09                      ← parity tests: run same inputs on bash + Go, diff outputs
           │
     ┌─────┴─────┐
     ▼           ▼
Step 7: gm10    gm11              ← parallel: local cross-compilation in release + switchover (update skill callers)
           │
           ▼
Step 8: gm12                      ← remove old shell scripts (deferred — only after confidence period)
```

- [x] [gm01] 2026-03-05: Scaffold Go module at src/fab-go/ with cobra CLI skeleton, shared internal/yaml package for .status.yaml struct, and subcommand stubs
- [x] [gm02] 2026-03-05: Port statusman.sh → `fab status` subcommand (43 yq calls, biggest performance win)
- [x] [gm03] 2026-03-05: Port resolve.sh → `fab resolve` subcommand (dependency of preflight and others)
- [x] [gm04] 2026-03-05: Port logman.sh → `fab log` subcommand (JSON append-only logging)
- [x] [gm05] 2026-03-05: Port preflight.sh → `fab preflight` subcommand (in-process calls to status + resolve + log)
- [x] [gm06] 2026-03-05: Port changeman.sh → `fab change` subcommand (change lifecycle)
- [x] [gm07] 2026-03-05: Port calc-score.sh → `fab score` subcommand (confidence scoring)
- [x] [gm08] 2026-03-05: Port archiveman.sh → `fab archive` subcommand (archive/restore)
- [x] [gm09] 2026-03-05: Parity test harness — run identical inputs through bash scripts and Go binary, diff stdout/stderr/exit codes/file mutations
- [x] [gm10] 2026-03-05: Local cross-compilation in fab-release.sh — build fab binary for darwin-arm64, darwin-amd64, linux-arm64, linux-amd64 via `GOOS/GOARCH go build`; produce per-platform kit-{os}-{arch}.tar.gz plus generic kit.tar.gz (no binary, fallback); update fab-upgrade.sh to detect platform and download correct archive; update bootstrap one-liner with platform detection
- [x] [gm11] 2026-03-05: Switchover — update all skill scripts and bash callers to invoke `fab` binary instead of shell scripts
- [x] [gm12] 2026-03-05: Remove shim layer from old shell scripts — make `fab` binary the sole code path (after confidence period)
- [x] [gm13] 2026-03-05: Remove old shell scripts from lib/ (deferred — only after confidence period in production)

## Go Migration — Phase 2 (wt binary)

Depends on Phase 1 completion. wt01 (shared library) must come first; then individual commands are parallel.

```
Step 1: wt01 (shared lib)
           │
     ┌─────┼──────┬──────┬──────┐
     ▼     ▼      ▼      ▼      ▼
Step 2: wt02   wt03   wt04   wt07   ← parallel: create, list, open, init+status (leaf commands)
     │     │      │      │
     └─────┼──────┘      │
           ▼             │
Step 3: wt05   wt06      │        ← parallel: delete (needs list), pr (needs create)
     │     │             │
     └─────┼─────────────┘
           ▼
Step 4: wt08                      ← parity tests: bash vs Go for each command
           │
     ┌─────┴─────┐
     ▼           ▼
Step 5: wt09    wt10              ← parallel: cross-compile + switchover
           ▼
Step 6: wt11                      ← remove old shell scripts (deferred)
```

- [ ] [wt01] 2026-03-05: Port wt-common.sh shared library → internal/worktree/ Go package (git worktree operations, name generation, stash/rollback)
- [ ] [wt02] 2026-03-05: Port wt-create → `wt create` subcommand
- [ ] [wt03] 2026-03-05: Port wt-list → `wt list` subcommand
- [ ] [wt04] 2026-03-05: Port wt-open → `wt open` subcommand (editor/terminal/file-manager launch)
- [ ] [wt05] 2026-03-05: Port wt-delete → `wt delete` subcommand (interactive rollback, branch cleanup)
- [ ] [wt06] 2026-03-05: Port wt-pr → `wt pr` subcommand (gh integration)
- [ ] [wt07] 2026-03-05: Port wt-init and wt-status → `wt init` and `wt status` subcommands
- [ ] [wt08] 2026-03-05: Parity tests — run identical inputs through bash wt-* scripts and Go wt binary, diff outputs
- [ ] [wt09] 2026-03-05: Cross-compile wt binary, integrate into fab-release.sh alongside fab binary
- [ ] [wt10] 2026-03-05: Switchover — update env-packages.sh and PATH to use Go wt binary
- [ ] [wt11] 2026-03-05: Remove old wt shell scripts from packages/wt/ (deferred — only after confidence period)
