# Spec: Remove router config gate + add `shell-init` wrapper

**Change**: 260511-c432-fix-completion-outside-repo
**Created**: 2026-05-12
**Affected memory**: `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Dynamic completion handlers (change IDs, stage names, change types via Cobra `ValidArgsFunction`) — explicitly deferred to a separate change.
- Removing the `internal.ResolveConfig` distinction between "missing config" and "corrupted config" — corrupted-config still hard-errors at the router. Only the missing-config path becomes a soft fall-through.
- Auditing or modifying any individual fab-go subcommand's per-command guards — those are already correct (audit confirmed).

## Router: Config Gate Removal

### Requirement: Always-Route Policy

The router (`src/go/fab-kit/cmd/fab/main.go`) SHALL route every non-fab-kit command to `fab-go` regardless of whether `fab/project/config.yaml` is present. The router SHALL NOT short-circuit dispatch based on config presence.

When `fab/project/config.yaml` is present, the router SHALL exec the `fab-go` binary cached at `~/.fab-kit/versions/{cfg.FabVersion}/fab-go`.

When `fab/project/config.yaml` is absent, the router SHALL exec the `fab-go` binary cached at `~/.fab-kit/versions/{routerVersion}/fab-go`, where `routerVersion` is the router's build-time `version` constant.

When `fab/project/config.yaml` is present but corrupted (parse error), the router SHALL exit non-zero with the parse error from `internal.ResolveConfig`. This path is unchanged.

#### Scenario: Inside a fab repo
- **GIVEN** the current working directory is inside a fab-managed repo with a valid `config.yaml`
- **WHEN** the user runs `fab <any-non-fab-kit-command>`
- **THEN** the router execs `~/.fab-kit/versions/{cfg.FabVersion}/fab-go` with the user's arguments
- **AND** the project-pinned version is used

#### Scenario: Outside a fab repo
- **GIVEN** the current working directory is NOT inside a fab-managed repo
- **WHEN** the user runs `fab <any-non-fab-kit-command>`
- **THEN** the router execs `~/.fab-kit/versions/{routerVersion}/fab-go` with the user's arguments
- **AND** the router-bundled version is used
- **AND** no "Not in a fab-managed repo" error is emitted by the router

#### Scenario: Corrupted config (dispatch path)
- **GIVEN** `fab/project/config.yaml` exists but cannot be parsed
- **WHEN** the user runs a `fab` command that dispatches through `execFabGo` (i.e., any non-fab-kit command other than the router-inline `--help`/`-h`/`help`/`--version`/`-v` paths)
- **THEN** the router exits non-zero with the parse error from `internal.ResolveConfig`

#### Scenario: Corrupted config (inline help/version paths)
- **GIVEN** `fab/project/config.yaml` exists but cannot be parsed
- **WHEN** the user runs `fab --help`, `fab -h`, `fab help`, `fab --version`, or `fab -v`
- **THEN** the command exits 0 — these router-inline paths use `cfg, _ := internal.ResolveConfig()` and silently ignore parse errors (help and version are best-effort and must remain available even with a broken config)

### Requirement: Removal of Router-Side Config Gate

The router SHALL NOT contain a `fabGoNoConfigArgs` allowlist. The router SHALL NOT contain a `resolveFabVersion` helper that returns a `shouldExit` signal. The router SHALL NOT emit the message `Not in a fab-managed repo. Run 'fab init' to set one up.` from any code path.

Version selection (project-pinned vs. router-bundled) SHALL be inline in `execFabGo` and consistent with the existing pattern in `printHelp` (i.e., `if cfg != nil { v = cfg.FabVersion } else { v = routerVersion }`).

#### Scenario: Map is gone
- **GIVEN** the post-change source of `src/go/fab-kit/cmd/fab/main.go`
- **WHEN** the file is grep'd for `fabGoNoConfigArgs`
- **THEN** zero matches are found

#### Scenario: Helper is gone
- **GIVEN** the post-change source of `src/go/fab-kit/cmd/fab/main.go`
- **WHEN** the file is grep'd for `resolveFabVersion`
- **THEN** zero matches are found

#### Scenario: Error message is gone
- **GIVEN** the post-change source of `src/go/fab-kit/cmd/fab/main.go`
- **WHEN** the file is grep'd for `Not in a fab-managed repo`
- **THEN** zero matches are found

### Requirement: fab-go Self-Guards Are Authoritative

Subcommands that require project state SHALL continue to fail-closed outside a fab repo via their existing per-command guards (typically a call to `resolve.FabRoot()`). The router SHALL NOT duplicate or pre-empt these guards.

#### Scenario: Workflow command outside a repo
- **GIVEN** the current working directory is NOT inside a fab-managed repo
- **WHEN** the user runs `fab preflight` (or `score`, `resolve`, `status`, `change`, `log`, `batch`, `fab-help`)
- **THEN** fab-go exits non-zero with `ERROR: fab/ directory not found`
- **AND** the error originates from fab-go's per-command guard, not the router

#### Scenario: Config-independent command outside a repo
- **GIVEN** the current working directory is NOT inside a fab-managed repo
- **WHEN** the user runs `fab kit-path`
- **THEN** fab-go exits 0 and prints the absolute path to the system kit cache directory

#### Scenario: Help command outside a repo
- **GIVEN** the current working directory is NOT inside a fab-managed repo
- **WHEN** the user runs `fab --help`, `fab -h`, `fab help`, `fab help <subcommand>`, `fab completion zsh`, or `fab <subcommand> --help`
- **THEN** the command exits 0 with the expected help/completion output
- **AND** no "Not in a fab-managed repo" error is emitted

## fab-go: `shell-init` Command

### Requirement: `fab shell-init <shell>` Command

A new top-level `fab-go` subcommand named `shell-init` SHALL exist. It SHALL accept exactly one positional argument: the target shell name. Valid values are `bash`, `zsh`, and `fish`. The command SHALL emit, on standard output, a shell-completion script identical to the output of `fab completion <shell>` for the same shell.

The command SHALL be registered in `src/go/fab/cmd/fab/main.go` via `root.AddCommand(shellInitCmd())`, alongside the existing command factories.

The command source SHALL live at `src/go/fab/cmd/fab/shellinit.go` with a single exported factory function `shellInitCmd() *cobra.Command`, matching the convention of sibling files (`kitpath.go`, `operator.go`, `fabhelp.go`).

The command's `Short` description SHALL be `Emit shell completion script for sourcing (alias for 'completion <shell>')` or equivalent wording.

#### Scenario: Bash output
- **GIVEN** any working directory
- **WHEN** the user runs `fab shell-init bash`
- **THEN** the command exits 0
- **AND** the stdout output is byte-identical to `fab completion bash`

#### Scenario: Zsh output
- **GIVEN** any working directory
- **WHEN** the user runs `fab shell-init zsh`
- **THEN** the command exits 0
- **AND** the stdout output starts with `#compdef fab`
- **AND** the output is byte-identical to `fab completion zsh`

#### Scenario: Fish output
- **GIVEN** any working directory
- **WHEN** the user runs `fab shell-init fish`
- **THEN** the command exits 0
- **AND** the output is byte-identical to `fab completion fish`

#### Scenario: Invalid shell name
- **GIVEN** any working directory
- **WHEN** the user runs `fab shell-init powershell` (or any other non-supported value)
- **THEN** the command exits non-zero with an error message listing the supported shells (`bash`, `zsh`, `fish`)

#### Scenario: Missing argument
- **GIVEN** any working directory
- **WHEN** the user runs `fab shell-init` with no argument
- **THEN** the command exits non-zero with a usage error

#### Scenario: Too many arguments
- **GIVEN** any working directory
- **WHEN** the user runs `fab shell-init zsh extra`
- **THEN** the command exits non-zero with a usage error

#### Scenario: Eval-able install one-liner
- **GIVEN** a shell session
- **WHEN** the user runs `eval "$(fab shell-init zsh)"`
- **THEN** tab-completion for `fab` is activated in the current shell

### Requirement: Implementation Delegation

The `shell-init` command SHALL delegate the script generation to Cobra's built-in completion APIs on the root command (`GenBashCompletionV2`, `GenZshCompletion`, `GenFishCompletion`) rather than re-implementing the completion script. The bash path SHALL use `GenBashCompletionV2(out, true)` to match the implementation of Cobra's built-in `completion bash` subcommand (which uses V2 with descriptions enabled); using V1 (`GenBashCompletion`) would produce different output and violate the byte-identical contract below. This keeps `shell-init` semantically equivalent to `completion <shell>`.

#### Scenario: Implementation parity
- **GIVEN** the implementation of `shell-init`
- **WHEN** the source file `src/go/fab/cmd/fab/shellinit.go` is inspected
- **THEN** it invokes one of `cmd.Root().GenBashCompletionV2(out, true)`, `cmd.Root().GenZshCompletion(...)`, or `cmd.Root().GenFishCompletion(out, true)` based on the argument

## Tests

### Requirement: Router Test Updates

The test file `src/go/fab-kit/cmd/fab/main_test.go` SHALL be updated as follows:

- `TestFabGoNoConfigArgs` SHALL be removed entirely (the symbol it tests no longer exists).
- `TestResolveFabVersion` SHALL be removed entirely (the function it tests no longer exists).
- `TestFabKitArgs` SHALL remain unchanged.
- `TestVersion` and `TestPrintVersion` SHALL remain unchanged.
- A new test, `TestExecFabGoVersionSelection` (or equivalent), SHALL cover the inline version-selection logic: it MAY use a small extracted helper if the inline body is non-trivial; otherwise the version-selection logic MAY be considered covered by integration smoke tests.

The router SHALL continue to pass `go test ./src/go/fab-kit/cmd/fab/...`.

#### Scenario: Deleted tests
- **GIVEN** the post-change source of `src/go/fab-kit/cmd/fab/main_test.go`
- **WHEN** the file is grep'd for `TestFabGoNoConfigArgs` or `TestResolveFabVersion`
- **THEN** zero matches are found

#### Scenario: Surviving tests pass
- **GIVEN** the post-change codebase
- **WHEN** `go test ./src/go/fab-kit/cmd/fab/...` is run
- **THEN** all tests pass

### Requirement: `shell-init` Tests

A test file SHALL exist at `src/go/fab/cmd/fab/shellinit_test.go`. It SHALL verify:

- `shell-init bash` produces non-empty output.
- `shell-init zsh` produces output beginning with `#compdef fab`.
- `shell-init fish` produces non-empty output.
- For each supported shell, `shell-init <shell>` output is byte-identical to the same root command's built-in completion generator (`GenBashCompletionV2(out, true)`, `GenZshCompletion(out)`, `GenFishCompletion(out, true)`). This guards against the implementation drifting away from a pure delegation to Cobra's `completion <shell>`.
- `shell-init powershell` returns a non-nil error.
- `shell-init` (no args) returns a non-nil error.
- `shell-init zsh extra` returns a non-nil error.

#### Scenario: Test coverage
- **GIVEN** the post-change codebase
- **WHEN** `go test ./src/go/fab/cmd/fab/...` is run
- **THEN** all tests including `shellinit_test.go` pass

## Documentation

### Requirement: README Install Update

`README.md` SHALL include, in its installation or quickstart section, a documented one-liner for activating shell completion:

```sh
eval "$(fab shell-init zsh)"   # or bash / fish
```

Or equivalent guidance. The line `fab completion <shell>` MAY also be mentioned for users who prefer to save the script to a file.

#### Scenario: README contains shell-init
- **GIVEN** the post-change `README.md`
- **WHEN** the file is grep'd for `shell-init`
- **THEN** at least one match exists in an install/setup context

### Requirement: Architecture Spec Update

`docs/specs/architecture.md` SHALL describe the router's always-route policy (and the version-selection rule: project-pinned when config exists, bundled otherwise). It SHALL NOT describe the removed `fabGoNoConfigArgs` allowlist as the mechanism. If the prior text references that allowlist, the reference SHALL be removed or rewritten.

#### Scenario: No stale allowlist reference
- **GIVEN** the post-change `docs/specs/architecture.md`
- **WHEN** the file is grep'd for `fabGoNoConfigArgs`
- **THEN** zero matches are found

#### Scenario: New router policy is documented
- **GIVEN** the post-change `docs/specs/architecture.md`
- **WHEN** the file is read
- **THEN** it describes that the router always routes to fab-go and that version selection is `project-pinned if config present, else router-bundled`

## Design Decisions

1. **Drop the router's config gate entirely** (chosen approach: route everything; let fab-go self-guard).
   - *Why*: Audit (run outside a fab repo against the cached `fab-go` binary) confirmed that every workflow command already produces a clean `ERROR: fab/ directory not found` (exit 1) via its own guard. The router's gate is redundant, less accurate (says "Run `fab init`" even for commands like `kit-path` that need no init), and blocks legitimate use cases (sourcing `fab completion zsh` from a `~/.zshrc` that loads outside fab projects).
   - *Rejected*: Extending `fabGoNoConfigArgs` by 3-4 entries — patches the symptom while leaving the design wrong. Every new config-free command in the future would need a router-side allowlist update.

2. **`shell-init` is a `tu`-style alias for `completion`**, not an instructional emitter that prints "add this to your rc file".
   - *Why*: One-step install (`eval "$(fab shell-init zsh)"`) matches `tu shell-init` semantics that motivated this work. An instructional emitter requires a second command invocation, which is more confusing than helpful.
   - *Rejected*: Instructional emitter, or omitting `shell-init` entirely and pointing users at `fab completion` — both fail the discoverability test (new users don't know about Cobra's `completion` convention; "shell-init" is the verb they already think in).

3. **`shell-init` lives in fab-go, not the router**.
   - *Why*: The router has no Cobra command tree (only manual arg-routing in `main.go`). fab-go is where Cobra auto-registers `completion` and exposes `GenZshCompletion` etc. There is no alternative.
   - *Rejected*: Router-side implementation — would require either re-hosting Cobra in the router or exec'ing fab-go, both of which defeat the purpose.

4. **Delete `fabGoNoConfigArgs` and `resolveFabVersion`, don't deprecate them**.
   - *Why*: They have no remaining purpose. Keeping them as no-ops invites confusion. Tests delete cleanly (`TestFabGoNoConfigArgs` and `TestResolveFabVersion` go with them).
   - *Rejected*: Keep as no-ops for one release — premature optimization for a non-existent backward-compatibility concern (these are package-private Go symbols).

5. **Version skew (outside-repo commands run on the router-bundled fab-go) is acceptable**.
   - *Why*: In practice, only pure-doc commands and config-free commands (`completion`, `help`, `kit-path`, `pane`, `operator`'s switch path, hooks) reach the bundled-version path. Commands that need project state are rejected by fab-go's own guards before reaching any version-sensitive logic. Completion scripts are sourced once into the shell rc and don't churn with each fab release.
   - *Rejected*: Pinning the bundled version to a stable release tag, or shipping a separate `fab-go-stub` for pure-doc commands — both add complexity for a non-problem.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Drop the router's config gate entirely instead of extending `fabGoNoConfigArgs` | Confirmed from intake #1; user explicitly broadened scope. Audit confirmed it's safe (no crashes, all per-command guards are correct) | S:95 R:80 A:95 D:95 |
| 2 | Certain | fab-go's per-command guards are the authoritative answer to "does this need config?" | Confirmed from intake #2; audit verified every workflow command produces clean `ERROR: fab/ directory not found` (exit 1) outside a repo | S:100 R:90 A:100 D:100 |
| 3 | Certain | `kit-path`, `pane`, `operator`, `hook session-start` are genuinely config-free outside a repo | Confirmed from intake #3; verified directly against `~/.fab-kit/versions/1.9.4/fab-go` in `/tmp` | S:100 R:95 A:100 D:100 |
| 4 | Certain | Add `fab shell-init <shell>` as a `tu`-style alias for `completion` | Confirmed from intake #4; user explicitly requested this | S:95 R:85 A:95 D:95 |
| 5 | Certain | Delete `fabGoNoConfigArgs` and `resolveFabVersion` entirely; not as no-ops | Upgraded from intake Confident — no remaining purpose, tests delete cleanly, package-private symbols carry no backward-compatibility concern | S:90 R:80 A:90 D:90 |
| 6 | Certain | `shell-init` lives in fab-go at `src/go/fab/cmd/fab/shellinit.go`, factory `shellInitCmd()`, registered via `root.AddCommand` in `main.go` | Confirmed from intake #7 and #12; main.go has 13 `AddCommand` entries, convention is one-command-one-file with `xCmd()` factory matching `kitpath.go`/`operator.go`/`fabhelp.go` | S:95 R:85 A:95 D:95 |
| 7 | Certain | `shell-init` delegates to `cmd.Root().GenBashCompletionV2(out, true) / GenZshCompletion / GenFishCompletion(out, true)` | Cobra exposes these methods on the root command; Cobra's auto-generated `completion bash` subcommand uses `GenBashCompletionV2(out, true)` internally (so we mirror that to keep byte-identical parity), while `completion zsh|fish` use the same single generator each | S:95 R:80 A:95 D:95 |
| 8 | Certain | Inline version selection (`if cfg != nil { v = cfg.FabVersion } else { v = routerVersion }`) in `execFabGo` | Pattern already used in `printHelp` at main.go:142-149 — copy verbatim. No helper needed | S:90 R:85 A:90 D:90 |
| 9 | Certain | Version skew (router-bundled fab-go outside a repo) is acceptable | Confirmed from intake #6; user explicitly confirmed during discussion. In practice only pure-doc commands reach the bundled path | S:90 R:75 A:90 D:85 |
| 10 | Certain | Memory updates land in `kit-architecture.md` (modify) — primarily lines 255-266 (Router section) | Confirmed by reading the file: lines 255-266 contain the canonical router/`fabGoNoConfigArgs` documentation. Section is the natural target for a rewrite | S:95 R:80 A:95 D:90 |
| 11 | Certain | `operator` outside-repo behavior stays as-is | Confirmed from intake #10 via code reading: switch path (`tmux select-window`) is safe anywhere; launch path self-guards via `resolve.FabRoot()` at operator.go:50. Not accidental | S:85 R:80 A:90 D:85 |
| 12 | Certain | `hook session-start` silent no-op is correct; no debug message | Confirmed from intake #11 via code reading at hook.go:113-119 — explicit "swallow" comments. Same pattern in `hookStopCmd`, `hookUserPromptCmd`. Deliberate design | S:95 R:90 A:95 D:90 |
| 13 | Certain | `shell-init` rejects unknown shells via Cobra `Args` validation listing `bash`, `zsh`, `fish` | Standard Cobra pattern; matches `tu shell-init` argument validation | S:90 R:90 A:90 D:90 |
| 14 | Certain | Corrupted-config path (parse error) continues to hard-error at router | Preserved from current behavior in `execFabGo` — only the `cfg == nil && err == nil` case (missing config) becomes a soft fall-through | S:95 R:90 A:95 D:95 |
| 15 | Confident | `TestExecFabGoVersionSelection` is optional — integration smoke tests suffice if inline body stays trivial | Inline body is two lines; explicit unit test adds little over end-to-end coverage. Spec allows either approach | S:75 R:80 A:80 D:75 |
| 16 | Confident | README install snippet uses `eval "$(fab shell-init zsh)"` (zsh primary) but mentions bash/fish | Matches the user's shell (zsh per system env); README will likely reach a mixed audience, so multi-shell mention is right | S:75 R:90 A:80 D:75 |

16 assumptions (14 certain, 2 confident, 0 tentative, 0 unresolved).
