# Spec: Decouple wt and idea from fab-kit

**Change**: 260506-4rtx-decouple-wt-idea
**Created**: 2026-05-06
**Affected memory**: `docs/memory/fab-workflow/distribution.md`, `docs/memory/fab-workflow/migrations.md`, `docs/memory/fab-workflow/kit-architecture.md`

## Non-Goals

- Modifying wt or idea CLI behavior — this change does not touch either binary's runtime semantics.
- Repository archival of `wt`/`idea` source from fab-kit's git history — historical commits remain reachable via `git log`.
- Reconciling any divergence between fab-kit's vendored copy and the external repos — the external repos are now canonical; divergence (minimal at `v0.0.1`) is theirs to address.
- Backporting fixes between repos.
- Editing archived change folders or completed migration files (`src/kit/migrations/0.43.1-to-0.44.0.md`).

## Distribution: Homebrew Formula Topology

### Requirement: fab-kit Formula Declares wt and idea as Dependencies

The Homebrew formula `fab-kit` (template at `.github/formula-template.rb`, deployed to `sahil87/homebrew-tap` as `Formula/fab-kit.rb`) SHALL declare `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"`. The formula SHALL install only `fab` and `fab-kit`. The formula's `desc` field SHALL describe only fab and fab-kit (not wt or idea). The formula's `test` block SHALL assert versions for `fab` and `fab-kit` only.

#### Scenario: Fresh install resolves all four binaries
- **GIVEN** a system with no prior fab-kit, wt, or idea installation
- **WHEN** the user runs `brew install sahil87/tap/fab-kit`
- **THEN** Homebrew installs `fab-kit` and resolves `depends_on` to install `wt` and `idea` from the same tap
- **AND** all four binaries (`fab`, `fab-kit`, `wt`, `idea`) are on PATH after install
- **AND** each responds to `--version` with its own formula's version

#### Scenario: Formula desc and test exclude wt and idea
- **GIVEN** the deployed `fab-kit.rb` formula
- **WHEN** the formula is loaded by `brew`
- **THEN** the `desc` mentions only fab (router) and fab-kit (workspace lifecycle)
- **AND** the `test` block has exactly two `assert_match` calls (one for `fab --version`, one for `fab-kit --version`)
- **AND** the `def install` block contains exactly two `bin.install` calls (`"fab"`, `"fab-kit"`)

### Requirement: wt and idea Standalone Formulas Use link_overwrite

The standalone formulas `Formula/wt.rb` and `Formula/idea.rb` in `sahil87/homebrew-tap` SHALL declare `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"` respectively. Their templates in `sahil87/wt` and `sahil87/idea` (`.github/formula-template.rb`) SHALL also declare these directives so subsequent regenerations preserve them.

> Implementation status: shipped pre-spec. Tap commit `8f9ade1`, wt template commit `c264680`, idea template commit `6745505`. The spec records the requirement so future hydrate writes the contract into memory.

#### Scenario: Upgrade from fab-kit 1.6.2 with bundled wt/idea
- **GIVEN** a user with `fab-kit 1.6.2` installed (which previously bundled `wt` and `idea` symlinks)
- **WHEN** the user runs `brew upgrade fab-kit`
- **THEN** the new fab-kit formula triggers installation of standalone `wt` and `idea` via `depends_on`
- **AND** `link_overwrite` allows `wt`/`idea` formulas to take ownership of the `bin/wt` and `bin/idea` symlinks without "Refusing to link" errors
- **AND** `wt --version` and `idea --version` report the standalone formula versions, not `fab-kit 1.6.2`

#### Scenario: Standalone wt or idea release regenerates formula with link_overwrite
- **GIVEN** a tagged release of `sahil87/wt` (e.g., `v0.1.0`)
- **WHEN** the release workflow `sed`-substitutes the template into `sahil87/homebrew-tap/Formula/wt.rb`
- **THEN** the regenerated formula contains the `link_overwrite "bin/wt"` line (carried in by the template)
- **AND** the same applies to `idea` releases via `idea`'s template

## Build: Cross-Compilation Matrix

### Requirement: justfile Build Targets Three Binaries Per Platform

The `justfile` recipes `build`, `test`, `test-v`, `build-target`, and `build-all` SHALL only reference `src/go/fab/` and `src/go/fab-kit/`. References to `src/go/wt/` and `src/go/idea/` SHALL be removed. The `build-target` recipe SHALL invoke `_build-binary` exactly three times per platform (producing `fab-go`, `fab-kit`, `fab` binaries). The `build-all` recipe SHALL produce 12 binaries total (3 binaries × 4 platforms).

#### Scenario: just build produces three binaries on the host platform
- **GIVEN** a clean fab-kit checkout post-change
- **WHEN** the developer runs `just build` on macOS arm64
- **THEN** `dist/bin/fab`, `dist/bin/fab-kit`, and `dist/bin/fab-go` exist
- **AND** `dist/bin/wt` and `dist/bin/idea` do NOT exist
- **AND** the rename loop in `build` covers `fab-go fab-kit fab` only

#### Scenario: just build-all produces 12 binaries
- **GIVEN** a clean fab-kit checkout post-change
- **WHEN** the developer runs `just build-all`
- **THEN** `dist/bin/` contains exactly 12 binaries: `{fab,fab-kit,fab-go}-{darwin,linux}-{arm64,amd64}`
- **AND** no `wt-*` or `idea-*` binaries are produced

#### Scenario: just test runs only fab and fab-kit Go test suites
- **GIVEN** a clean fab-kit checkout post-change
- **WHEN** the developer runs `just test`
- **THEN** `go test ./...` runs only in `src/go/fab/` and `src/go/fab-kit/`
- **AND** no test invocations target `src/go/wt/` or `src/go/idea/`

### Requirement: package-brew Stages Two Binaries

`scripts/just/package-brew.sh` SHALL stage exactly two binaries (`fab`, `fab-kit`) per platform into the brew tarball. The header comment, the existence check loop, the `cp`/`chmod` commands, and the `tar` argument list SHALL all reflect the two-binary set. The output `dist/brew-{os}-{arch}.tar.gz` SHALL contain only `fab` and `fab-kit`.

#### Scenario: brew tarball contents
- **GIVEN** the script run after `just build-all`
- **WHEN** the script produces `dist/brew-darwin-arm64.tar.gz`
- **THEN** `tar tzf dist/brew-darwin-arm64.tar.gz` lists exactly `fab` and `fab-kit`
- **AND** `wt` and `idea` are not present in any platform tarball

## Source: Vendored Module Removal

### Requirement: src/go/wt and src/go/idea are Removed

The directories `src/go/wt/` and `src/go/idea/` SHALL be removed from the fab-kit repository in their entirety, including their `cmd/`, `internal/`, `go.mod`, and `go.sum` contents. No fab/fab-kit Go code SHALL import either module post-removal.

#### Scenario: source tree no longer contains the modules
- **GIVEN** a fab-kit checkout post-change
- **WHEN** running `ls src/go/`
- **THEN** the listing shows only `fab/` and `fab-kit/`
- **AND** running `grep -r "src/go/idea\|src/go/wt" src/go/fab/ src/go/fab-kit/` produces no results

### Requirement: Runtime exec.Command Sites Continue to Work

The fab-go runtime call-sites at `src/go/fab/cmd/fab/batch_new.go` and `src/go/fab/cmd/fab/batch_switch.go` SHALL continue to invoke `wt` via `exec.Command("wt", ...)`. The `wt` binary SHALL remain on PATH because the `wt` Homebrew formula installs it independently — guaranteed by `depends_on` in the fab-kit formula.

#### Scenario: fab batch new invokes wt
- **GIVEN** a user with `fab-kit` installed (which transitively installed `wt` via `depends_on`)
- **WHEN** the user runs a fab command path that invokes `exec.Command("wt", "create", ...)` (e.g., `fab batch new`)
- **THEN** the `wt` binary on PATH is found and executed
- **AND** the Homebrew-managed `wt` formula's binary is the one resolved

## Documentation: Sourcing and Tap References

### Requirement: docs/memory/fab-workflow/distribution.md Reflects New Topology

`docs/memory/fab-workflow/distribution.md` SHALL be rewritten to describe the new packaging topology. Changes:

- The "three-binary architecture" / "four binaries" / "five binaries" claims SHALL be updated to describe fab-kit's brew tarball as containing 2 binaries (`fab`, `fab-kit`); the user-visible install set remains 4 (`fab`, `fab-kit`, `wt`, `idea`) via Homebrew dependency resolution.
- The Binary table (currently "Five Go binaries", three columns Binary/Source/Distribution) SHALL be split or annotated so `wt` and `idea` are labeled as standalone formulas with their own source repos (`github.com/sahil87/wt`, `github.com/sahil87/idea`).
- Archive contents description SHALL describe the brew tarball as 2-binary.
- The release workflow description SHALL state 12 binaries cross-compiled (3 × 4 platforms), not 20.
- All `wvrdz/tap` and `wvrdz/homebrew-tap` strings in active prose SHALL be replaced with `sahil87/tap` and `sahil87/homebrew-tap`. The fix applies to lines 15, 18, 30, 66 and any other live references that surface during the edit. The "260401-ixzv-org-migrate-mit-license" Changelog row (around line 361) SHALL NOT be modified — it is historical.
- A Design Decision entry SHALL be added documenting the wt/idea decoupling rationale (link to this change).

#### Scenario: Live prose uses sahil87/tap
- **GIVEN** the rewritten `distribution.md`
- **WHEN** running `grep -n "wvrdz" docs/memory/fab-workflow/distribution.md`
- **THEN** the only matches are inside the historical Changelog table (rows describing past changes)
- **AND** every live prose reference to a Homebrew tap uses `sahil87/tap` or `sahil87/homebrew-tap`

#### Scenario: Binary table reflects ownership split
- **GIVEN** the rewritten `distribution.md`
- **WHEN** the reader inspects the binary table
- **THEN** the table distinguishes fab-kit-owned binaries (`fab`, `fab-kit`, `fab-go`) from external dependency binaries (`wt`, `idea`)
- **AND** wt's Source column references `github.com/sahil87/wt`
- **AND** idea's Source column references `github.com/sahil87/idea`

### Requirement: docs/memory/fab-workflow/migrations.md Sweeps Stale Tap

`docs/memory/fab-workflow/migrations.md` line 79 (the brew install instruction `brew tap wvrdz/tap && brew install fab-kit`) SHALL be updated to `brew tap sahil87/tap && brew install fab-kit`. Other `wvrdz` strings in the file (if any) within active prose SHALL be similarly fixed. Historical content SHALL NOT be modified.

#### Scenario: Live tap reference is current
- **GIVEN** the edited `migrations.md`
- **WHEN** running `grep -n "wvrdz" docs/memory/fab-workflow/migrations.md`
- **THEN** no live prose reference contains `wvrdz`

### Requirement: docs/memory/fab-workflow/kit-architecture.md Sweeps Stale Tap

`docs/memory/fab-workflow/kit-architecture.md` line 188 (the brew tap reference) SHALL be updated to `sahil87/tap`. The Changelog row at line 618 (referring to the historical org migration) SHALL NOT be modified.

#### Scenario: Live tap reference is current
- **GIVEN** the edited `kit-architecture.md`
- **WHEN** running `grep -n "wvrdz" docs/memory/fab-workflow/kit-architecture.md`
- **THEN** the only matches are inside the Changelog table

### Requirement: docs/specs/packages.md Reframes wt and idea

`docs/specs/packages.md` SHALL be reframed:

- The sentence "Both binaries are compiled from `src/go/`" SHALL be removed or rewritten to describe wt and idea as standalone packages in their own repositories.
- The `**Binary**: src/go/wt/` line SHALL be replaced with a reference to `github.com/sahil87/wt`.
- The `**Binary**: src/go/idea/` line SHALL be replaced with a reference to `github.com/sahil87/idea`.
- The functional reference (subcommand reference, `wt create` flags, `wt shell-setup` recipe, `idea` query semantics, worktree behavior) SHALL be retained — that material describes user-facing CLI behavior, which is invariant under this change.
- A note SHALL be added at the top of the page or each section that wt/idea are now standalone packages with independent release cadence; fab-kit's Homebrew formula declares them as dependencies.

#### Scenario: Source references point to standalone repos
- **GIVEN** the rewritten `packages.md`
- **WHEN** the reader inspects the wt and idea sections
- **THEN** each section identifies the binary's canonical repo (`github.com/sahil87/wt`, `github.com/sahil87/idea`)
- **AND** there are no `src/go/wt/` or `src/go/idea/` source path references in active prose

#### Scenario: Functional reference preserved
- **GIVEN** the rewritten `packages.md`
- **WHEN** the reader looks up `wt create` flags or `idea` subcommands
- **THEN** the same flag/subcommand reference content is present (only sourcing/distribution prose changed)

### Requirement: Light-Touch Sweeps in Glossary and User-Flow

`docs/specs/glossary.md` and `docs/specs/user-flow.md` SHALL be reviewed and edited only where prose describes wt/idea as fab-kit-internal binaries or includes stale `wvrdz` references in active prose. Operational examples (`wt create`, `idea add`) and command-map references SHALL NOT be modified. Historical changelog rows SHALL NOT be modified.

#### Scenario: Operational examples unchanged
- **GIVEN** the reviewed `glossary.md` and `user-flow.md`
- **WHEN** the reader inspects examples that invoke `wt` or `idea`
- **THEN** the commands themselves are unchanged
- **AND** any "installed by fab-kit" / "fab-kit-internal" framing has been corrected to "installed via Homebrew dependency"

## CI: Release Workflow

### Requirement: release.yml Requires No Structural Changes

`.github/workflows/release.yml` SHALL continue to drive release via `just` recipes (`just dist-kit`, `just build-all`, `just package-kit`, `just package-brew`, `just release-notes`, `just brew-formula`). No step SHALL be added or removed. The `Update Homebrew tap` step's `cp dist/fab-kit.rb /tmp/tap/Formula/fab-kit.rb` SHALL continue to fully overwrite the deployed formula, ensuring `link_overwrite` and other tap-side directives in `wt.rb` and `idea.rb` are not collateral-edited.

#### Scenario: Release pipeline produces the new artifact set
- **GIVEN** a developer pushes tag `v1.7.0` to `sahil87/fab-kit`
- **WHEN** the workflow runs unchanged
- **THEN** the resulting GitHub Release contains 4 kit archives + 4 brew archives (each brew archive containing only `fab` and `fab-kit`)
- **AND** the tap's `Formula/fab-kit.rb` is overwritten with the new template-rendered content (with `depends_on` lines)
- **AND** `Formula/wt.rb` and `Formula/idea.rb` in the tap are unaffected

## Versioning: Release Boundary

### Requirement: First Release Carrying This Change is fab-kit 1.7.0

The first fab-kit release that ships this change SHALL be version `1.7.0` (minor bump from `1.6.2`). The semver minor bump reflects a packaging refactor: user-visible install behavior is preserved (`brew install sahil87/tap/fab-kit` still installs all four binaries via dependency resolution), so this is not a breaking change for end users.

#### Scenario: VERSION bump
- **GIVEN** the current `src/kit/VERSION` of `1.6.2`
- **WHEN** the release author runs `just release minor`
- **THEN** `src/kit/VERSION` becomes `1.7.0`
- **AND** the release tag pushed to GitHub is `v1.7.0`

### Requirement: Release Notes Include Upgrade Troubleshooting Note

The release notes for `v1.7.0` SHALL include a section describing the wt/idea decoupling and a troubleshooting fallback for the rare case `link_overwrite` does not resolve cleanly during upgrade. The fallback line SHALL be `brew unlink wt idea && brew upgrade fab-kit`. The note SHALL be framed as a fallback (not a required pre-step), since `link_overwrite` is the primary mitigation.

#### Scenario: Release notes content
- **GIVEN** the generated `dist/release-notes.md` for `v1.7.0`
- **WHEN** the user reads the upgrade section
- **THEN** the wt/idea decoupling is explained in 1–3 sentences
- **AND** the troubleshooting fallback `brew unlink wt idea && brew upgrade fab-kit` is shown
- **AND** the note clarifies the fallback is only for cases where `link_overwrite` fails to resolve cleanly

## Design Decisions

1. **Decouple wt and idea via Homebrew `depends_on`, not Go module dependencies or CI-time external builds**:
   - *Why*: Each binary versions and releases independently from its own repo, but `brew install sahil87/tap/fab-kit` still installs all four CLIs transitively. fab-kit's CI shrinks (no longer cross-compiles wt/idea); fab-kit's release cadence decouples from theirs.
   - *Rejected*: Vendor via Go module dependency (`require github.com/sahil87/wt`) — still ties fab-kit's release to a wt version pin and produces yet another wt binary built from fab-kit's CI. Bundle binaries in fab-kit's brew tarball but build from external repos at CI time — coupling moves from source to CI; fab-kit releases are blocked when external CI is broken. Keep vendored sources, accept drift — defeats the purpose of the extraction.

2. **`link_overwrite` in standalone wt and idea formulas, not `caveats` or `pour_bottle?` in fab-kit**:
   - *Why*: `link_overwrite` is Homebrew's idiomatic mechanism for ownership transitions and runs silently. `caveats` would print instructions but still require the user to act manually. Custom `post_install` hooks fight Homebrew conventions and are fragile.
   - *Rejected*: `caveats` block in fab-kit asking users to `brew unlink wt idea` first — visible but doesn't actually solve the conflict. Custom `post_install` migration logic in fab-kit — overkill, fragile.

3. **Sweep stale `wvrdz/tap` references in the same change**:
   - *Why*: We are already editing `distribution.md`, `migrations.md`, and `kit-architecture.md` for the topology rewrite. Including the sweep avoids a separate trivial change and reduces drift between docs and reality. The "260401-ixzv-org-migrate-mit-license" change should have caught these — fixing them now closes the residual gap.
   - *Rejected*: Defer the sweep to a separate trivial change — adds bookkeeping overhead with no benefit since the same files are open anyway.

4. **Semver minor bump (1.7.0), not major (2.0.0)**:
   - *Why*: User-visible behavior of `brew install sahil87/tap/fab-kit` is preserved — all four binaries still install. The formula contract change is internal to Homebrew's dependency resolution, not user-facing.
   - *Rejected*: Major bump (2.0.0) — would signal a breaking change that does not exist. Patch bump (1.6.3) — understates the structural change in the formula.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | wt and idea ship as separate Homebrew formulas in `sahil87/tap`, currently at `v0.0.1` with real shas | Confirmed from intake #1 — both formulas exist and resolve via `brew install sahil87/tap/{wt,idea}` | S:95 R:90 A:90 D:95 |
| 2 | Certain | fab-kit's formula uses `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"` | Confirmed from intake #2 — chosen over Go module pin and CI-time builds | S:95 R:80 A:90 D:95 |
| 3 | Certain | Skills and exec.Command call-sites that invoke `wt`/`idea` are unchanged | Confirmed from intake #3 — binaries remain on PATH via `depends_on` | S:95 R:95 A:95 D:95 |
| 4 | Certain | `src/go/idea/` and `src/go/wt/` are removed entirely (cmd, internal, go.mod, go.sum) | Confirmed from intake #4 — no Go imports cross the boundary | S:90 R:60 A:95 D:90 |
| 5 | Certain | Stale `wvrdz/tap` references in active prose are fixed to `sahil87/tap` in the same pass; historical changelog rows preserved | Confirmed from intake #5; spec adds explicit "preserve historical changelog rows" guard | S:90 R:90 A:95 D:90 |
| 6 | Certain | Historical artifacts immutable (archived change folders, completed migrations, intake/spec/tasks for archived changes) | Confirmed from intake #6 | S:95 R:95 A:95 D:95 |
| 7 | Certain | Brew tarball: 4 → 2 binaries; build matrix: 20 → 12 | Confirmed from intake #7 — derivable mechanically from the justfile and package-brew.sh edits | S:90 R:85 A:95 D:95 |
| 8 | Certain | `release.yml` needs no structural change | Confirmed from intake #8; spec adds explicit guard that the `Update Homebrew tap` step continues to fully overwrite `Formula/fab-kit.rb` only (not `wt.rb` or `idea.rb`) | S:90 R:85 A:95 D:90 |
| 9 | Certain | `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"` already shipped in tap formulas (commit `8f9ade1`) and templates (commits `c264680`, `6745505`) | Upgraded from intake Confident — the prevention mechanism is in place; the spec records the requirement so memory captures it. Reversibility upgraded since reverting these is trivial. | S:95 R:90 A:95 D:95 |
| 10 | Certain | `docs/specs/packages.md` keeps functional reference, reframes Binary lines and bundling claims, adds a "now standalone" note | Confirmed from intake #10 | S:90 R:85 A:95 D:90 |
| 11 | Certain | `docs/specs/glossary.md` and `docs/specs/user-flow.md` are light-touch — only operational/architectural framing needs touchup | Confirmed from intake #11 | S:90 R:90 A:90 D:90 |
| 12 | Certain | Land this change as fab-kit `1.7.0` (minor bump) | Confirmed from intake #12 (clarified — user explicitly chose minor) | S:95 R:80 A:90 D:95 |
| 13 | Certain | Release notes include the `brew unlink wt idea && brew upgrade fab-kit` troubleshooting fallback (framed as fallback, not required pre-step) | Confirmed from intake #13 (clarified — user explicitly chose troubleshooting note) | S:90 R:85 A:90 D:90 |
| 14 | Certain | Conflict prevention via `link_overwrite` in standalone wt/idea formulas (chosen over `caveats` and custom `post_install` hooks) | Confirmed from intake #14 — already shipped across three repos | S:95 R:90 A:95 D:95 |
| 15 | Confident | Existing fab-kit 1.6.2 users running `brew upgrade fab-kit` resolve `depends_on` cleanly thanks to `link_overwrite`; the troubleshooting fallback is rarely needed | Confirmed from intake #9 — `link_overwrite` is the standard Homebrew migration pattern. Could not verify on a real macOS upgrade in spec generation. | S:75 R:70 A:80 D:80 |
| 16 | Certain | Open Question from intake — `release.yml` `Update Homebrew tap` step is idempotent | Resolved during spec generation: `cp dist/fab-kit.rb /tmp/tap/Formula/fab-kit.rb` is a full overwrite of the file regenerated from template; no leftover lines from prior versions can linger | S:90 R:85 A:90 D:85 |
| 17 | Tentative | No external (non-fab-kit) consumers hardcode `dist/bin/wt` or `dist/bin/idea` paths from a fab-kit checkout. | Open Question from intake #1; consumer projects exist (other repos that may script-build fab-kit). Best-effort search not performed in this repo. | S:50 R:60 A:55 D:65 |

17 assumptions (15 certain, 1 confident, 1 tentative, 0 unresolved).
