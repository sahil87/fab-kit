# Intake: Relocate Kit to System Cache

**Change**: 260402-gnx5-relocate-kit-to-system-cache
**Created**: 2026-04-02
**Status**: Draft

## Origin

> Discussion session via `/fab-discuss` exploring the dependency surface of `fab/.kit/` in user projects, with the goal of eliminating the in-project kit directory entirely. The conversation covered: runtime dependency audit (Go binaries, skills, hooks, templates, scripts), the new architecture (exe-sibling resolution), agent-agnostic template access (`fab kit-path`), and eliminating `kit.conf`.

Key decisions reached during discussion:
1. User projects will no longer contain a `fab/.kit/` folder
2. Kit content moves to `~/.fab-kit/versions/<version>/kit/` (already the cache location, sibling to `fab-go`)
3. Go binaries resolve kit via `os.Executable()` → sibling `kit/` directory
4. A `fab kit-path` command (or preflight YAML field) exposes the resolved path to any agent
5. Templates stay in system cache (not synced to project) — resolved via `fab kit-path` for agent-agnosticism (Codex, Cursor, etc.)
6. `kit.conf` is eliminated: `build-type` feature removed, `repo` hardcoded in Go binary
7. In the source repo, `fab/.kit/` is renamed to `src/kit/`

## Why

1. **Portability friction**: The current model copies `fab/.kit/` into every user project and every worktree. This creates sync staleness issues, duplication, and a large `.gitignore` surface.

2. **Architecture inconsistency**: `fab-go` already lives in the system cache at `~/.fab-kit/versions/<version>/fab-go`, but the content it operates on (`fab/.kit/`) is scattered across projects. The kit content and the binary that reads it should be co-located.

3. **Multi-agent support**: Skills currently hardcode `fab/.kit/skills/_preamble.md` etc. as read paths. This only works for Claude Code (which uses `.claude/skills/`). Moving to `fab kit-path` makes template/migration access agent-agnostic — any agent that can run a shell command can resolve the kit.

4. **Simpler upgrade story**: `fab upgrade` currently does an atomic swap of `fab/.kit/` in the project. With the cache model, upgrading updates `~/.fab-kit/versions/<version>/kit/` once — all projects using that version see the update immediately (after `fab sync` for skill deployment).

5. **Source repo clarity**: `fab/.kit/` is a hidden directory inside `fab/`, making it non-obvious as a source directory. `src/kit/` is explicit and sits alongside `src/go/`.

## What Changes

### 1. Source repo layout

Move `fab/.kit/` to `src/kit/` in the fab-kit development repository:

```
# Before
fab/.kit/
  skills/
  templates/
  hooks/
  migrations/
  scaffold/
  schemas/
  VERSION
  kit.conf

# After
src/kit/
  skills/
  templates/
  hooks/
  migrations/
  scaffold/
  schemas/
  VERSION
```

`kit.conf` is eliminated (see section 5 below). `src/kit/` becomes the source directory used by the build/release system to produce the distribution archive.

### 2. Kit path resolution in Go binaries

Replace all `filepath.Join(fabRoot, ".kit", ...)` patterns with exe-sibling resolution:

```go
// New shared utility (e.g., internal/kitpath/kitpath.go)
func KitDir() (string, error) {
    exe, err := os.Executable()
    if err != nil {
        return "", err
    }
    exe, err = filepath.EvalSymlinks(exe)
    if err != nil {
        return "", err
    }
    return filepath.Join(filepath.Dir(exe), "kit"), nil
}
```

This affects `fab-go` directly (it lives in `~/.fab-kit/versions/<version>/fab-go`). For `fab-kit` (system binary, not in the versions directory), kit path resolution must go through version resolution — `fab-kit` reads `fab_version` from `config.yaml`, then resolves `~/.fab-kit/versions/{version}/kit/`.

Affected Go files with current `fab/.kit/` path construction:

| File | Current reference | New resolution |
|------|------------------|----------------|
| `src/go/fab/internal/change/change.go` | `filepath.Join(fabRoot, ".kit", "templates", "status.yaml")` | `filepath.Join(kitDir, "templates", "status.yaml")` |
| `src/go/fab/internal/preflight/preflight.go` | `filepath.Join(fabRoot, ".kit", "VERSION")` | `filepath.Join(kitDir, "VERSION")` |
| `src/go/fab/internal/hooklib/sync.go` | Constructs `fab/.kit/hooks/` commands | Resolves hook path from kitDir |
| `src/go/fab/cmd/fab/fabhelp.go` | `filepath.Join(fabRoot, ".kit")` | Uses kitDir |
| `src/go/fab-kit/internal/hooksync.go` | Constructs `fab/.kit/hooks/` commands | Resolves hook path from version-resolved kitDir |
| `src/go/fab-kit/internal/init.go` | `filepath.Join(cwd, "fab", ".kit")` | Copies scaffold from version-resolved kitDir |
| `src/go/fab-kit/internal/upgrade.go` | `filepath.Join(cfg.RepoRoot, "fab", ".kit")` | Updates cache at `~/.fab-kit/versions/{version}/kit/` |
| `src/go/fab-kit/internal/download.go` | Strips `.kit/` prefix from archive entries | Update prefix handling |
| `src/go/fab-kit/internal/sync.go` | Manages `fab/.kit-migration-version` | Resolve VERSION from kitDir |

### 3. New `fab kit-path` command

Add a command that outputs the resolved kit directory path:

```bash
$ fab kit-path
/home/user/.fab-kit/versions/0.45.1/kit
```

This is the primary mechanism for agents to locate kit content. The command:
- Reads `fab_version` from `fab/project/config.yaml`
- Resolves `~/.fab-kit/versions/{version}/kit/`
- Prints the absolute path to stdout

Alternative or additionally: add a `kit_path` field to the `fab preflight` YAML output, since preflight already runs at the start of every skill. This avoids an extra command invocation.

### 4. Skill and template access

**Skills** (`_preamble.md`, `_cli-fab.md`, `_naming.md`, `_generation.md`, and all user-invocable skills) continue to be deployed to `.claude/skills/` via `fab sync`. No change to how agents discover skills.

**Templates** (`intake.md`, `spec.md`, `tasks.md`, `checklist.md`) stay in the system cache. Skills reference them via `fab kit-path`:

```markdown
# Before (in _generation.md)
1. Read the template from `fab/.kit/templates/intake.md`

# After
1. Read the template from `$(fab kit-path)/templates/intake.md`
```

**Migrations** (`*.md` files) stay in the system cache. `/fab-setup migrations` resolves them via `fab kit-path`.

This approach is agent-agnostic: any agent (Claude Code, Codex, Cursor) that can execute `fab kit-path` can resolve templates and migrations.

### 5. Eliminate kit.conf

`fab/.kit/kit.conf` currently has two fields:

```
build-type=production
repo=sahil87/fab-kit
```

- **`build-type`**: Remove the feature entirely. Remove the test-build guard from `_preamble.md` (Section "Test-Build Guard") that reads `kit.conf` and stops on `build-type=test`.
- **`repo`**: Hardcode as a Go constant in the binary:

```go
const defaultRepo = "sahil87/fab-kit"
```

Or inject via `ldflags` at build time for flexibility. The binary is always distributed from this repo, so the value is inherent to the binary.

### 6. Hook registration

Hooks are currently registered in `.claude/settings.local.json` as:

```json
"command": "bash \"$CLAUDE_PROJECT_DIR\"/fab/.kit/hooks/on-session-start.sh"
```

Two options (decide during spec):
- **Option A**: Resolve hook path from cache during `fab hook sync` and hardcode the absolute path
- **Option B**: Since hooks are thin wrappers (`bash` → `fab hook <subcommand>`), inline the `fab hook` command directly, eliminating the shell script indirection entirely:

```json
"command": "fab hook session-start"
```

Option B is simpler and eliminates the hook scripts from the kit entirely.

### 7. Build and release changes

| File | Change |
|------|--------|
| `justfile` | `cat fab/.kit/VERSION` → `cat src/kit/VERSION`; `rsync fab/.kit/` → `rsync src/kit/`; `cp -a fab/.kit/.` → `cp -a src/kit/.` |
| `scripts/release.sh` | `kit_dir="$repo_root/fab/.kit"` → `kit_dir="$repo_root/src/kit"` |
| `scripts/install.sh` | Update to install to cache, not `fab/.kit/` |
| `.gitignore` | Remove `fab/.kit/bin/*`, `!fab/.kit/bin/.gitkeep`, `fab/.kit-sync-version` entries |
| `.github/copilot-code-review.yml` | `fab/.kit/**` → `src/kit/**` |

### 8. fab sync scope reduction

`fab sync` currently copies kit content into the project. After this change:
- `fab sync` only deploys skills to the agent's native skill location (`.claude/skills/` for Claude Code)
- No kit content is copied into the project
- Sync staleness detection: already handled by #307 (compares `fab/.kit/VERSION` vs `config.yaml`'s `fab_version`) — with this change, VERSION comes from exe-sibling kit in cache

### 9. .envrc cleanup

```bash
# Remove (deprecated, no-op — scripts dir doesn't exist)
PATH_add fab/.kit/scripts

# Keep (unchanged — uses command form, not a kit path)
export WORKTREE_INIT_SCRIPT="fab sync"
```

### 10. Preflight changes

`fab preflight` staleness check was updated by #307 to compare `fab/.kit/VERSION` vs `config.yaml`'s `fab_version`. After this change:
- VERSION is read from exe-sibling kit in cache (no `fab/.kit/` in project)
- The comparison logic remains the same, just the VERSION source changes

### 11. User-facing CLI messages

Update strings in Go source:
- `"Upgrade fab/.kit/ to..."` → update to reflect cache-based upgrade
- `"Populating fab/.kit/..."` → update to reflect cache-based init
- `"Updating fab/.kit/..."` → update to reflect cache-based update

### 12. Migration for existing users

Ship a migration file (`0.46.0-to-0.47.0.md` or similar) that:
1. Verifies the system cache is populated (`~/.fab-kit/versions/<version>/kit/` exists)
2. Updates hook registrations in `.claude/settings.local.json` — replace `bash "$CLAUDE_PROJECT_DIR"/fab/.kit/hooks/on-*.sh` with inline `fab hook <subcommand>` commands
3. Removes `fab/.kit/` from the project
4. Removes `PATH_add fab/.kit/scripts` from `.envrc` if still present
5. Cleans `.gitignore` entries (remove `fab/.kit/bin/*` etc.)

## Affected Memory

- `fab-workflow/kit-architecture`: (modify) Update directory structure, kit path resolution, upgrade procedures
- `fab-workflow/distribution`: (modify) Update release packaging, upgrade flow, cache layout
- `fab-workflow/setup`: (modify) Update init flow (no longer copies kit to project)
- `fab-workflow/preflight`: (modify) Update staleness check (version from cache, not project)
- `fab-workflow/migrations`: (modify) Add migration documentation for this version
- `fab-workflow/context-loading`: (modify) Update skill/template path references
- `fab-workflow/execution-skills`: (modify) Update template access pattern (fab kit-path)
- `fab-workflow/planning-skills`: (modify) Update template access pattern
- `fab-workflow/configuration`: (modify) Remove kit.conf, update config references

## Impact

- **All three Go binaries** (fab, fab-kit, fab-go): Path resolution changes
- **All skill files** (~30 files): References to `fab/.kit/` paths change to `$(fab kit-path)/` or `.claude/skills/`
- **Build system**: justfile, release.sh, install.sh
- **CI**: copilot-code-review.yml exclusion pattern
- **User projects**: Migration removes `fab/.kit/`, updates hooks
- **Constitution**: Principle V (Portability) rewording — no longer "cp -r fab/.kit/"
- **Documentation**: ~250 files reference `fab/.kit/` (memory, specs, change archives) — bulk find-and-replace for docs, selective updates for code

## Open Questions

None — all resolved during discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Kit content co-located with fab-go at `~/.fab-kit/versions/<version>/kit/` | Discussed — user confirmed exe-sibling resolution as the pattern | S:95 R:60 A:95 D:95 |
| 2 | Certain | Source directory renamed from `fab/.kit/` to `src/kit/` in dev repo | Discussed — user specified this explicitly | S:95 R:70 A:90 D:95 |
| 3 | Certain | `kit.conf` eliminated entirely | Discussed — user confirmed: remove build-type, hardcode repo | S:95 R:80 A:90 D:95 |
| 4 | Certain | Templates accessed via `fab kit-path`, not synced to project | Discussed — user chose Option 2 (cache access) over Option 1 (sync) for agent-agnosticism | S:90 R:70 A:85 D:90 |
| 5 | Certain | User projects will no longer contain `fab/.kit/` | Discussed — user stated this explicitly as the goal | S:95 R:50 A:90 D:95 |
| 6 | Certain | `build-type` feature removed (not migrated) | Discussed — user said "can get rid of it" | S:90 R:85 A:90 D:95 |
| 7 | Certain | `repo` field hardcoded in Go binary | Discussed — user confirmed hardcoding is acceptable | S:85 R:80 A:85 D:90 |
| 8 | Certain | Shared utility `kitpath.KitDir()` using `os.Executable()` + sibling resolution | Discussed — user described the approach, confirmed during assumption review | S:90 R:80 A:85 D:85 |
| 9 | Certain | `fab-kit` resolves kit via version from config.yaml (not exe-sibling) | Discussed — user confirmed; `fab-kit` is system binary, not in versions dir | S:85 R:70 A:80 D:80 |
| 10 | Certain | Skill reference files (`_preamble`, `_cli-fab`, etc.) deployed to `.claude/skills/` via `fab sync` | Discussed — current behavior confirmed, no change needed | S:90 R:75 A:80 D:85 |
| 11 | Certain | Hook scripts replaced with inline `fab hook <subcommand>` commands | Clarified — user chose inline (eliminates hook script files from kit entirely) | S:90 R:75 A:85 D:90 |
| 12 | Certain | No sync staleness stamp file needed | Clarified — #307 already removed sync version file; staleness uses VERSION vs config fab_version | S:95 R:90 A:95 D:95 |
| 13 | Certain | `fab kit-path` as standalone command only (not preflight field) | Clarified — user confirmed standalone-only; preflight would require unnecessary coupling to fab sync | S:90 R:85 A:85 D:90 |
| 14 | Certain | Migration removes `fab/.kit/` from user projects automatically (with cache verification) | Discussed — user confirmed standard migration behavior | S:85 R:60 A:75 D:80 |

14 assumptions (14 certain, 0 confident, 0 tentative, 0 unresolved).
