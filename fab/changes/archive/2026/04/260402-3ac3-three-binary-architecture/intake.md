# Intake: Three-Binary Architecture

**Change**: 260402-3ac3-three-binary-architecture
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Discussion session (`/fab-discuss`): "Should fab-sync.sh be absorbed into fab binary? If yes, which binary - shim or fab-go?" Evolved into a broader architectural decision about binary separation and the future removal of `fab/.kit/` from repos. Conversational, multi-step — decisions built on each other over several rounds.

Key discussion thread:
1. Started with sync placement → concluded fab-kit (shim) not fab-go, because sync is workspace lifecycle
2. User raised version-mismatch scenario (project pins v45, not in cache) → led to future model where no `.kit/` in repo, sync pulls from cache
3. User proposed three-binary split for testability → each binary independently invocable
4. Settled on negative-match routing for `fab` → help composes from both sub-binaries

## Why

1. **Sync is a workspace lifecycle operation, not a workflow operation.** It belongs alongside `init` and `upgrade` (bootstrap/reconcile the workspace), not alongside `resolve` and `status` (run the workflow). Today's shim already blurs this with init/upgrade living in it — sync is the natural third member.

2. **The shim is untestable in isolation.** You can't run `fab init --help` without the shim potentially trying to dispatch to fab-go. Three binaries means `fab-kit -h`, `fab-go -h`, and `fab -h` each work independently.

3. **Future: `fab/.kit/` will not be in the repo.** Kit content will live only in the cache (`~/.fab-kit/versions/{ver}/`). Sync becomes "ensure correct version is cached + deploy content from cache to workspace" — a bootstrapping concern, not a workflow concern.

## What Changes

### 1. Rename shim → fab-kit

The current shim binary (`src/go/shim/`) becomes `fab-kit`. It owns workspace lifecycle:

- `fab-kit init` — initialize fab in a repo (existing)
- `fab-kit upgrade` — upgrade to a different version (existing)
- `fab-kit sync` — reconcile workspace with pinned version (new, absorbs `fab-sync.sh` + `2-sync-workspace.sh`)

Source move: `src/go/shim/` → `src/go/fab-kit/` (or rename the binary output; source path TBD).

### 2. New thin `fab` router binary

A new binary at `src/go/fab/cmd/fab/main.go` — wait, that path is already fab-go. The router needs its own location, e.g. `src/go/router/` or the existing shim path repurposed.

The router:
- Reads the first argument
- If it matches the fab-kit command set (`init`, `upgrade`, `sync`, `--version`, `-v`, `--help`, `-h`, `help`) → exec `fab-kit` with all args
- Otherwise → resolve version from `config.yaml`, ensure fab-go cached, exec `fab-go` with all args
- **Negative match**: the fab-kit command set is small and stable; everything else goes to fab-go. Same pattern as today's `nonRepoCommands` map in `dispatch.go`.

`fab help` composes help from both binaries:
- Always shows fab-kit commands (workspace group)
- When inside a repo (config.yaml exists), also queries `fab-go --help` and shows workflow commands
- Outside a repo, shows workspace commands only

### 3. Absorb `fab-sync.sh` into `fab-kit sync`

Port the sync logic from shell to Go inside the fab-kit binary:

**Currently in shell** (`fab/.kit/scripts/fab-sync.sh` + `fab/.kit/sync/2-sync-workspace.sh`, ~490 lines total):
- Directory scaffolding (create `fab/changes`, `docs/memory`, `docs/specs`)
- Scaffold tree-walk with two merge strategies (fragment-merge for JSON permissions + line dedup, copy-if-absent for regular files)
- Multi-agent skill deployment (Claude Code → copies, OpenCode → symlinks, Codex → copies, Gemini → copies)
- Stale skill cleanup and legacy migration
- Version stamp tracking (`fab/.kit-sync-version`)
- Project-level extensibility (`fab/sync/*.sh` custom scripts)

**Go implementation**:
- Reuse existing `CachedKitDir()`, `EnsureCached()` from cache.go
- Reuse `copyDir()` pattern from init.go for file deployment
- Port fragment merge logic (JSON permission merging, line dedup)
- Port multi-agent skill deployment with per-agent format strategies
- Keep project-level `fab/sync/*.sh` extensibility by exec'ing those scripts from Go
- Stale cleanup and version tracking

### 4. Update build system

**Justfile changes:**
- `build-shim` → `build-fab-kit` (or rename to build both fab + fab-kit)
- Add build recipe for the new `fab` router binary
- `build-target` recipe: build 5 binaries instead of 4 (fab, fab-kit, fab-go, wt, idea)
- `build-all`: cross-compile produces 20 binaries (5 × 4 platforms) instead of 16

**Packaging changes (`scripts/just/`):**
- `package-brew.sh`: include `fab-kit` in brew archives (currently: fab, wt, idea → becomes: fab, fab-kit, wt, idea)
- `package-kit.sh`: kit archives still contain `fab-go` + kit content (unchanged)

**CI changes (`.github/workflows/release.yml`):**
- Upload 5th binary per platform to GitHub release
- Update brew archive creation to include fab-kit
- Update Homebrew tap formula to install 4 binaries: `bin.install "fab"`, `bin.install "fab-kit"`, `bin.install "wt"`, `bin.install "idea"`
- Add formula test for `fab-kit --version`

### 5. Version dispatch moves from shim to router

Today the shim reads `config.yaml`, calls `EnsureCached()`, and execs fab-go. This logic moves to the `fab` router. The router becomes the version-aware dispatcher.

`fab-kit` does NOT need version dispatch for its own commands — `init`, `upgrade`, and `sync` work with whatever version is pinned in config.yaml (or no config at all for `init`). However, `sync` does need to call `EnsureCached()` to ensure the kit content is cached — so fab-kit needs access to the cache/download infrastructure.

Options:
- Shared Go module with cache/download code used by both `fab` and `fab-kit`
- `fab-kit` imports from a shared `internal/` package

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Binary architecture changes from 2-binary (shim + fab-go) to 3-binary (fab + fab-kit + fab-go)
- `fab-workflow/distribution`: (modify) Build produces 5 binaries, brew installs 4, kit archives unchanged

## Impact

- **Build system**: Justfile, packaging scripts, CI workflow all need updates for the new binary count
- **Homebrew tap**: Formula installs 4 binaries instead of 3
- **Shell scripts**: `fab-sync.sh` and `fab/.kit/sync/2-sync-workspace.sh` become dead code (can be removed or kept as fallback during transition)
- **Shared code**: Cache/download infrastructure needs to be importable by both `fab` (router) and `fab-kit`
- **Go module structure**: May need restructuring if shim and router share code (currently `src/go/shim/` is self-contained)
- **Existing hooks**: `fab hook sync` in fab-go is unaffected (it syncs hook registrations, not skill files)
- **Related backlog items**: [uqy8] directly addressed, [41gc] and [ub2y] partially addressed

## Open Questions

- ~~Should the transition be phased or clean cut?~~ Resolved: clean cut — remove fab-sync.sh, ship fab-kit sync directly.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Sync belongs in fab-kit, not fab-go | Discussed — user confirmed sync is workspace lifecycle, same family as init/upgrade | S:95 R:70 A:90 D:90 |
| 2 | Certain | Three binaries: fab (router), fab-kit (lifecycle), fab-go (engine) | Discussed — user proposed and confirmed for testability (each independently invocable) | S:95 R:60 A:85 D:90 |
| 3 | Certain | Negative match routing in fab | Discussed — user confirmed option 3: fab knows fab-kit commands, everything else → fab-go | S:95 R:80 A:90 D:95 |
| 4 | Certain | Each binary owns its own --help | Discussed — user confirmed; fab composes help from both sub-binaries | S:90 R:85 A:85 D:90 |
| 5 | Certain | Future: no fab/.kit/ in repo, content from cache | Discussed — user stated this as the direction; sync deploys from cache to workspace | S:90 R:50 A:80 D:85 |
| 6 | Confident | Build produces 5 binaries (fab, fab-kit, fab-go, wt, idea) | Follows from three-binary decision; user asked about build impact, answer was +1 binary | S:80 R:70 A:80 D:85 |
| 7 | Confident | Brew installs fab + fab-kit + wt + idea (not fab-go) | fab-go is version-pinned per project, served from cache — same as today | S:75 R:75 A:80 D:80 |
| 8 | Confident | Project-level fab/sync/*.sh extensibility preserved | Shell scripts exec'd from Go; maintains existing project customization hook | S:60 R:65 A:75 D:70 |
| 9 | Certain | Both binaries live in one Go module at src/go/fab-kit/ with two cmd/ entries (cmd/fab/ and cmd/fab-kit/) sharing internal/ | Discussed — user confirmed single-module, two-binary pattern; no workspaces or extra modules needed | S:95 R:80 A:90 D:95 |
| 10 | Certain | Source directory renamed src/go/shim/ → src/go/fab-kit/ | Discussed — user confirmed | S:95 R:80 A:90 D:95 |

10 assumptions (7 certain, 2 confident, 0 tentative, 0 unresolved).
