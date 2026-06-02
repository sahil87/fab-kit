# Intake: Decouple wt and idea from fab-kit

**Change**: 260506-4rtx-decouple-wt-idea
**Created**: 2026-05-06
**Status**: Draft

## Origin

This change emerged from a `/fab-discuss` session reviewing the current state of the `wt` and `idea` packages, which had been extracted into standalone repositories (`../wt`, `../idea`) but were still vendored as `src/go/wt/` and `src/go/idea/` inside fab-kit. The user's question:

> "We have made wt and idea separate repos in their own right. (Check ../idea and ../wt repos). Can we now remove these from fab-kit repo? Are they tightly coupled?"

Investigation findings:

1. **Source: fully decoupled.** Both are independent Go modules with no imports from `src/go/fab/` or `src/go/fab-kit/`. The two external repos already have ports merged (`wt` PR #1 `Port wt CLI from fab-kit into standalone repo`, `idea` PR #1 `Port idea CLI from fab-kit into standalone repo`).
2. **Runtime: loose coupling via `exec.Command`.** Two fab-Go call-sites shell out to the `wt` binary on PATH (`src/go/fab/cmd/fab/batch_new.go:116`, `src/go/fab/cmd/fab/batch_switch.go:96`). These continue to work as long as `wt` is on PATH.
3. **Distribution: tightly coupled.** All four binaries (`fab`, `fab-kit`, `wt`, `idea`) ship in a single `brew-{os}-{arch}.tar.gz` from this repo's release pipeline; `.github/formula-template.rb` installs all four; `fab-kit.rb` in `sahil87/homebrew-tap` mirrors that bundling.
4. **External release pipelines: already wired.** Both external repos have full release workflows that build per-platform tarballs, create GitHub Releases, and push updated formulas to `sahil87/homebrew-tap`. The user confirmed both have published releases — `Formula/wt.rb` and `Formula/idea.rb` are now at `v0.0.1` with real shas.

User decisions during discussion:

- Each repo keeps its own formula in `sahil87/homebrew-tap` (already true).
- fab-kit's formula declares them as `depends_on` so `brew install fab-kit` continues to install all three transitively. No user-facing regression.
- `.claude/skills/_cli-external.md` stays as-is — it documents how skills *use* the binaries, which is unchanged.
- Stale `wvrdz/tap` references in docs are leftovers from the `260401-ixzv-org-migrate-mit-license` migration that should have been swept; fix them in the same pass while editing distribution docs.
- Historical artifacts (archived change folders, the `0.43.1-to-0.44.0.md` migration doc) are immutable — leave alone.

## Why

**Problem**: Two CLIs that were extracted into their own repos with their own release pipelines still have vendored copies in fab-kit. Every fab-kit release rebuilds and re-publishes `wt` and `idea` binaries from this repo's `src/go/{wt,idea}/`, even though the canonical source now lives elsewhere. Result: two release pipelines emitting two divergent versions of the same binary, with fab-kit's copy effectively a stale fork.

**Consequence if unfixed**:

- Source drift between fab-kit's `src/go/{wt,idea}/` and the external repos. Bugfixes shipped to `../wt` won't reach fab-kit users until someone manually re-ports.
- Confusing version semantics: `wt --version` reports the fab-kit version (e.g., `1.6.2`), not the wt repo's version (`0.0.1`). Users can't tell which release they're on.
- CI cycle time and maintenance burden: every fab-kit release runs `go test` and cross-compiles two CLIs that aren't fab-kit's responsibility anymore.
- Doc/skill confusion about where the canonical source lives.

**Why this approach over alternatives**:

| Alternative | Verdict |
|---|---|
| Keep vendored, accept drift | Rejected — defeats the purpose of the extraction; doubles maintenance. |
| Vendor via Go module dependency on `github.com/sahil87/wt` | Rejected — Go module pinning still ties fab-kit's release cadence to wt's, and produces yet another wt binary built from fab-kit's CI. |
| Bundle wt/idea binaries in fab-kit's brew tarball but build them from the external repos at CI time | Rejected — coupling moves from source to CI; fab-kit releases are blocked when external repo CI is broken. |
| **Chosen**: `depends_on "sahil87/tap/wt"` + `depends_on "sahil87/tap/idea"` | Homebrew handles transitive install; each binary is versioned and released independently; fab-kit's CI shrinks. |

## What Changes

### 1. Homebrew formula template (`.github/formula-template.rb`)

Drop installation and `--version` testing of `wt`/`idea`; add Homebrew dependency declarations. After the change:

```ruby
class FabKit < Formula
  desc "Specification-driven development toolkit — router and workspace lifecycle manager"
  homepage "https://github.com/sahil87/fab-kit"
  version "VERSION_PLACEHOLDER"
  license "MIT"

  depends_on "sahil87/tap/wt"
  depends_on "sahil87/tap/idea"

  on_macos do
    on_arm do
      url "https://github.com/sahil87/fab-kit/releases/download/v#{version}/brew-darwin-arm64.tar.gz"
      sha256 "SHA_DARWIN_ARM64"
    end
    # ... (other platforms unchanged)
  end

  def install
    bin.install "fab"
    bin.install "fab-kit"
  end

  test do
    assert_match version.to_s, shell_output("#{bin}/fab --version")
    assert_match version.to_s, shell_output("#{bin}/fab-kit --version")
  end
end
```

The `desc` field is also tightened — current text "router, workspace manager, worktree manager, and backlog tool" no longer applies.

### 2. Brew packaging script (`scripts/just/package-brew.sh`)

Drop `wt` and `idea` from staging, copy, chmod, and `tar` arguments. Updated relevant lines:

```bash
# Header
# Package brew archives into dist/ (per-platform: fab, fab-kit)

# Body — remove wt/idea variables
fab="dist/bin/fab-${os}-${arch}"
fab_kit="dist/bin/fab-kit-${os}-${arch}"

for bin in "$fab" "$fab_kit"; do
  if [ ! -f "$bin" ]; then
    echo "ERROR: Missing $bin — run 'just build-all' first."
    exit 1
  fi
done

# Staging — remove wt/idea cp + chmod
cp "$fab" "$staging/fab"
cp "$fab_kit" "$staging/fab-kit"
chmod +x "$staging/fab" "$staging/fab-kit"

COPYFILE_DISABLE=1 tar czf "$archive" -C "$staging" fab fab-kit
```

### 3. justfile

Three recipes simplify:

```just
# test — drop the two cd lines for src/go/{idea,wt}
test:
    cd src/go/fab && go test ./... -count=1
    cd src/go/fab-kit && go test ./... -count=1

# test-v — same edits as test
test-v:
    cd src/go/fab && go test ./... -v -count=1
    cd src/go/fab-kit && go test ./... -v -count=1

# build — drop idea wt from rename loop
for bin in fab-go fab-kit fab; do

# build-target — drop the two _build-binary calls; update comment to "3 binaries"
build-target os arch:
    just _build-binary src/go/fab ./cmd/fab fab-go {{os}} {{arch}} '{{fab_ldflags}}'
    just _build-binary src/go/fab-kit ./cmd/fab-kit fab-kit {{os}} {{arch}} '{{fab_kit_ldflags}}'
    just _build-binary src/go/fab-kit ./cmd/fab fab {{os}} {{arch}} '{{fab_kit_ldflags}}'

# build-all — comment update only ("3 binaries x 4 platforms = 12")
```

### 4. Source removal

Delete the two vendored Go modules entirely:

- `rm -rf src/go/idea/`
- `rm -rf src/go/wt/`

No fab/fab-kit Go code imports either module — verified via grep across `src/go/fab/` and `src/go/fab-kit/`.

### 5. Documentation updates

#### `docs/specs/packages.md`

Reframe wt and idea as external dependencies. Replace `**Binary**: src/go/wt/` and `**Binary**: src/go/idea/` lines with links to the standalone repos. Drop sentences that describe them as "compiled from `src/go/`". Keep functional documentation (subcommand reference, shell-setup recipe, query matching semantics) — that material describes the binaries' behavior, which doesn't change. Add a note at the top of the page that wt and idea are now standalone packages with their own release cadence; fab-kit declares them as Homebrew dependencies.

#### `docs/memory/fab-workflow/distribution.md`

This is the biggest doc edit:

- "Three-binary architecture" / "four binaries" / "five binaries" claims update to reflect the new shape: fab-kit's brew tarball ships 2 binaries (`fab`, `fab-kit`); the user-visible install set is still 4 (`fab`, `fab-kit`, `wt`, `idea`) but `wt`/`idea` come from sibling formulas.
- Binary table (around line 238) — split into "fab-kit-owned binaries" vs "external dependencies".
- Archive contents description (line 228) — fab-kit's `package-brew` produces 2-binary tarballs.
- Release pipeline description (line 271) — "20 binaries (fab router, fab-kit, fab-go, idea, wt × 4 platforms)" becomes "12 binaries (fab router, fab-kit, fab-go × 4 platforms)".
- Fix 5 stale `wvrdz/tap` references to `sahil87/tap`. Specific occurrences: lines 15, 18, 30, 66, 360 (and verify any others surface during the edit).

#### `docs/memory/fab-workflow/migrations.md`

Fix the stale `brew tap wvrdz/tap && brew install fab-kit` instruction at line 79 → `brew tap sahil87/tap && brew install fab-kit`.

#### `docs/memory/fab-workflow/kit-architecture.md`

Fix the stale `brew tap wvrdz/tap` reference at line 188 → `brew tap sahil87/tap`. Line 618 lives inside the historical changes table — leave that row alone (immutable history).

#### `docs/specs/glossary.md`

Light-touch update if any glossary entry describes wt/idea as fab-kit-internal binaries. Adjust to "external standalone CLIs, available via the same tap".

#### `docs/specs/user-flow.md`

Light-touch — the diagram references wt commands, which still work. Verify wording around "installed by fab-kit" if present.

#### Items deliberately NOT touched (historical artifacts)

- `src/kit/migrations/0.43.1-to-0.44.0.md` — historical migration doc, immutable.
- `fab/changes/260325-lhhk-brew-install-system-shim/intake.md` — historical change artifact.
- `fab/changes/archive/**` — archived change folders.
- "260401-ixzv-org-migrate-mit-license" rows in changes tables — historical.

### 6. CI workflow (`.github/workflows/release.yml`)

No structural changes needed — the workflow drives `just` recipes which now produce 2-binary tarballs automatically. Worth a once-over after edits to confirm step labels still read accurately.

### 7. Upgrade path for existing users

Existing users on `fab-kit 1.6.2` have `bin/wt` and `bin/idea` symlinks owned by the `fab-kit` formula (cellar copies under `Cellar/fab-kit/1.6.2/bin/`). When the new fab-kit formula's `depends_on` triggers installation of the standalone `wt` and `idea` formulas, those formulas need to take ownership of the same symlinks — without `link_overwrite`, Homebrew refuses with "Refusing to link: wt" / "Refusing to link: idea".

**Mitigation already shipped (across three repos)**:
- `sahil87/homebrew-tap` commit `8f9ade1` — `Formula/wt.rb` and `Formula/idea.rb` declare `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"`.
- `sahil87/wt` commit `c264680` — `.github/formula-template.rb` mirrors the `link_overwrite` so the next tagged release of `wt` regenerates the formula with it intact.
- `sahil87/idea` commit `6745505` — same mirror in `idea`'s template.

This lets the standalone formulas replace the fab-kit-owned symlinks silently during `brew upgrade fab-kit`. Pre-existing users get a frictionless upgrade.

**Release version**: Land this change as fab-kit `1.7.0` (minor bump). User-visible install behavior is preserved (`brew install sahil87/tap/fab-kit` still installs all four CLIs transitively); the formula contract change is internal.

**Release notes content**: Include a "What changed" entry explaining the split, and a troubleshooting fallback line for the rare case `link_overwrite` doesn't resolve cleanly:

```
brew unlink wt idea && brew upgrade fab-kit
```

This is belt-and-suspenders — `link_overwrite` should make it unnecessary, but documenting the escape hatch costs nothing.

## Affected Memory

- `fab-workflow/distribution.md`: (modify) Reflect the new packaging boundary — fab-kit's brew tarball ships 2 binaries, wt/idea come via Homebrew dependencies. Also fix stale wvrdz/tap → sahil87/tap references that survived the org migration.
- `fab-workflow/migrations.md`: (modify) One stale `wvrdz/tap` instruction.
- `fab-workflow/kit-architecture.md`: (modify) One stale `wvrdz/tap` instruction.

## Impact

**Build & release**:
- `justfile` recipes (`test`, `test-v`, `build`, `build-target`, `build-all`) — net simpler; build matrix shrinks from 5 binaries × 4 platforms to 3 × 4.
- `scripts/just/package-brew.sh` — drops two binaries from staging.
- `.github/formula-template.rb` — gains two `depends_on` lines, drops two installs/asserts.
- `.github/workflows/release.yml` — no structural changes; downstream effects from the recipes flow through.

**Source**:
- Delete `src/go/idea/` (cmd, internal, go.mod, go.sum).
- Delete `src/go/wt/` (cmd, internal, go.mod, go.sum).

**No-touch zones**:
- `src/go/fab/cmd/fab/batch_new.go` and `batch_switch.go` continue to invoke `wt` via `exec.Command` — works because `wt` is still on PATH (now via the `wt` formula).
- `src/kit/skills/_cli-external.md`, `src/kit/skills/fab-operator.md`, `src/kit/skills/_cli-fab.md`, `src/kit/skills/_preamble.md`, `src/kit/skills/fab-new.md`, `src/kit/skills/fab-draft.md` — unchanged. They reference the binaries and how skills use them, which is invariant under this change.

**Downstream consumers**:
- Homebrew tap (`sahil87/homebrew-tap`) — the `Formula/fab-kit.rb` already in the tap will be replaced by the next CI release, picking up the new `depends_on` lines from `formula-template.rb`. No manual edit needed in the tap repo.
- End users on macOS/Linux who installed via `brew install sahil87/tap/fab-kit` — `brew upgrade fab-kit` should resolve `depends_on` cleanly. Users who installed `wt`/`idea` standalone before this change get a no-op on those formulas.

**Out of scope**:
- Repository archival of `wt`/`idea` from fab-kit's git history — `git log` will still show the historical commits, which is correct.
- Backporting bug fixes that diverged between fab-kit's vendored copy and the external repos. The external repos are now canonical; any divergence is theirs to reconcile (and external `wt v0.0.1` was a direct port, so divergence should be minimal).
- Changes to wt or idea behavior. This change does not modify either CLI.

## Open Questions

- Are there any non-archived consumers (CI scripts, tooling outside this repo) that hardcode `dist/bin/wt` or `dist/bin/idea` paths from a fab-kit checkout? If so, they break with this change. Best-effort search of consumer projects pre-merge would mitigate.
- Is the `release.yml` workflow's `Update Homebrew tap` step expected to fully overwrite `Formula/fab-kit.rb` (idempotent), or could a leftover `wt`/`idea` install line linger if anything goes sideways? (Reading the workflow: `cp dist/fab-kit.rb /tmp/tap/Formula/fab-kit.rb` is a full overwrite — confirmed idempotent.)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | wt and idea ship as separate Homebrew formulas in `sahil87/tap` | Discussed — both formulas already exist at `v0.0.1` with real shas in `../homebrew-tap/Formula/{wt,idea}.rb`; published releases confirmed | S:95 R:90 A:90 D:95 |
| 2 | Certain | fab-kit's formula uses `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"` (not bundled binaries, not Go module dep, not CI-time external builds) | Discussed — user explicitly chose `depends_on` over alternatives | S:95 R:80 A:90 D:95 |
| 3 | Certain | `_cli-external.md` and other skills that invoke wt/idea binaries are unchanged | Discussed — user confirmed; binaries remain on PATH via Homebrew | S:95 R:95 A:95 D:95 |
| 4 | Certain | `src/go/idea/` and `src/go/wt/` are deleted entirely | Discussed — external repos are canonical; no Go imports cross the boundary, verified via grep across `src/go/fab/` and `src/go/fab-kit/` | S:90 R:60 A:95 D:90 |
| 5 | Certain | Stale `wvrdz/tap` references in active docs are fixed to `sahil87/tap` in the same pass | Discussed — leftovers from `260401-ixzv-org-migrate-mit-license`; user explicitly asked to sweep them while editing distribution docs | S:90 R:90 A:95 D:90 |
| 6 | Certain | Historical artifacts are immutable: `src/kit/migrations/0.43.1-to-0.44.0.md`, archived change folders, intake/spec/tasks files inside `fab/changes/archive/**` | Discussed — user explicitly excluded these | S:95 R:95 A:95 D:95 |
| 7 | Certain | fab-kit's brew tarball shrinks from 4 binaries to 2 (`fab`, `fab-kit`); 12-binary build matrix replaces the current 20-binary matrix | Mechanical consequence of removing wt/idea — derivable directly from the justfile and package-brew.sh edits; no judgment call | S:90 R:85 A:95 D:95 |
| 8 | Certain | The `release.yml` workflow needs no structural change — it invokes `just` recipes which now produce 2-binary tarballs automatically | Verified by reading `release.yml` — every step delegates to `just`; behavior is fully determined | S:90 R:85 A:95 D:90 |
| 9 | Confident | Existing users on `fab-kit 1.6.2` upgrading via `brew upgrade fab-kit` will resolve `depends_on` cleanly; rare conflict resolved via `brew unlink wt idea && brew upgrade fab-kit` | Standard Homebrew `depends_on` migration pattern; minor edge case worth documenting | S:70 R:60 A:75 D:75 |
| 10 | Certain | `docs/specs/packages.md` keeps the functional reference (subcommands, shell-setup recipe, query semantics) and reframes the "Binary" lines and bundling claims | Functional content describes user-facing CLI behavior which is invariant under this change; only sourcing/distribution claims need rewording | S:90 R:85 A:95 D:90 |
| 11 | Certain | `docs/specs/glossary.md` and `docs/specs/user-flow.md` are light-touch — only check that any "installed by fab-kit" / "fab-kit-internal" framing is updated | Both files reviewed during gap analysis; references are operational ("wt create ..."), not architectural | S:90 R:90 A:90 D:90 |
| 12 | Certain | Land this change as fab-kit `1.7.0` (minor bump) | Clarified — user confirmed; user-visible install behavior preserved via `depends_on` transitivity, packaging refactor is internal | S:95 R:80 A:90 D:95 |
| 13 | Certain | Release notes include the `brew unlink wt idea && brew upgrade fab-kit` troubleshooting fallback (belt-and-suspenders alongside `link_overwrite`) | Clarified — user confirmed inclusion as a troubleshooting note | S:90 R:85 A:90 D:90 |
| 14 | Certain | Conflict prevention via `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"` in the standalone formulas; templates updated in `../wt` and `../idea` so the next tagged release preserves the line | Already shipped: `sahil87/homebrew-tap@8f9ade1`, `sahil87/wt@c264680`, `sahil87/idea@6745505` | S:95 R:90 A:95 D:95 |

14 assumptions (13 certain, 1 confident, 0 tentative, 0 unresolved). Run /fab-clarify to review.
