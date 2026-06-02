# Tasks: Decouple wt and idea from fab-kit

**Change**: 260506-4rtx-decouple-wt-idea
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Formula and Build Edits

- [x] T001 [P] Update `.github/formula-template.rb`: drop `bin.install "wt"` and `bin.install "idea"` from `def install`; drop the two `--version` asserts from `test do`; add `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"` (place after `license`, before `on_macos do`); tighten `desc` to "Specification-driven development toolkit — fab router and fab-kit workspace lifecycle manager".

- [x] T002 [P] Update `scripts/just/package-brew.sh`: change header comment to `# Package brew archives into dist/ (per-platform: fab, fab-kit)`; remove the `wt=` and `idea=` variable assignments; reduce the existence-check loop to `for bin in "$fab" "$fab_kit"`; remove `cp "$wt"` and `cp "$idea"` lines; reduce `chmod +x` to `"$staging/fab" "$staging/fab-kit"`; reduce final `tar` arg list to `fab fab-kit`.

- [x] T003 [P] Update `justfile` `test` recipe: remove the `cd src/go/idea && go test ./... -count=1` and `cd src/go/wt && go test ./... -count=1` lines (lines 11–12).

- [x] T004 [P] Update `justfile` `test-v` recipe: remove the `cd src/go/idea && go test ./... -v -count=1` and `cd src/go/wt && go test ./... -v -count=1` lines (lines 18–19).

- [x] T005 [P] Update `justfile` `build` recipe rename loop: change `for bin in fab-go idea wt fab-kit fab; do` to `for bin in fab-go fab-kit fab; do` (line 31).

- [x] T006 [P] Update `justfile` `build-target` recipe: remove the two `_build-binary src/go/idea ./cmd idea ...` and `_build-binary src/go/wt ./cmd wt ...` calls (lines 70–71); update the comment from "5 binaries" to "3 binaries" (line 67).

- [x] T007 [P] Update `justfile` `build-all` recipe comment: change "5 binaries x 4 platforms = 20" to "3 binaries x 4 platforms = 12" (line 75).

## Phase 2: Source Removal

- [x] T008 Remove `src/go/idea/` directory entirely (`rm -rf src/go/idea`). Verify with `ls src/go/` showing only `fab/` and `fab-kit/`.

- [x] T009 Remove `src/go/wt/` directory entirely (`rm -rf src/go/wt`). Verify with `ls src/go/` showing only `fab/` and `fab-kit/`.

- [x] T010 Verify no fab/fab-kit Go code imports the removed modules: run `grep -rn "src/go/idea\|src/go/wt\|fab-kit/src/go/idea\|fab-kit/src/go/wt" src/go/fab/ src/go/fab-kit/` — must produce zero matches.

## Phase 3: Build Verification

- [x] T011 Run `just test` and confirm only `src/go/fab/` and `src/go/fab-kit/` test suites execute (passes or pre-existing failures only — no missing-package errors from removed `cd src/go/{wt,idea}` lines).

- [x] T012 Run `just build` and confirm `dist/bin/fab`, `dist/bin/fab-kit`, `dist/bin/fab-go` exist; `dist/bin/wt` and `dist/bin/idea` do NOT exist.

- [x] T013 Run `just build-target darwin arm64` and confirm exactly 3 binaries are produced for that platform (`fab-darwin-arm64`, `fab-kit-darwin-arm64`, `fab-go-darwin-arm64`).

- [x] T014 Run `just package-brew` (after `just build-all`) and confirm `tar tzf dist/brew-darwin-arm64.tar.gz` lists exactly two entries: `fab` and `fab-kit`.

## Phase 4: Documentation Updates

- [x] T015 [P] Rewrite `docs/memory/fab-workflow/distribution.md`:
  - Replace `wvrdz/homebrew-tap` with `sahil87/homebrew-tap` and `wvrdz/tap` with `sahil87/tap` in active prose (lines 15, 18, 30, 66, and any others surfacing during the edit). Do NOT modify the historical Changelog row at line 360 ("260401-46hw-brew-install-system-shim") or line 361 ("260401-ixzv-org-migrate-mit-license").
  - Update Section "Homebrew Formula" to describe the formula installing 2 binaries (`fab`, `fab-kit`) and declaring `depends_on` for wt and idea; the user-visible install set is still 4 binaries via dependency resolution.
  - Update "Three-binary architecture" / "four binaries" framing throughout — fab-kit's brew tarball ships 2 binaries.
  - Update the "Build Recipes (justfile)" section: `build` compiles 3 binaries; `build-target` produces 3 per platform; `build-all` produces 12 total (3 × 4).
  - Update the "Five Go binaries" table (currently lines 234–243): split into fab-kit-owned (`fab`, `fab-kit`, `fab-go`) and external dependencies (`wt` → `github.com/sahil87/wt`, `idea` → `github.com/sahil87/idea`); annotate that wt/idea are installed via Homebrew dependency resolution from `sahil87/tap`.
  - Update "package-brew" entry: `(fab, fab-kit)` instead of `(fab, fab-kit, wt, idea)`.
  - Update CI workflow description: 12 binaries cross-compiled (not 20).
  - Update "Release Archive Contents" prose mentioning Homebrew-distributed binaries.
  - Add a Design Decision entry: "Decouple wt and idea via depends_on (260506-4rtx)" — chosen rationale: external repos canonical, fab-kit CI shrinks; rejected: vendor-via-Go-module, CI-time external builds, accept-drift.
  - Add a requirement under "Homebrew Distribution" describing `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"` in the standalone formulas as the upgrade-conflict mitigation.

- [x] T016 [P] Update `docs/memory/fab-workflow/migrations.md` line 79: change `brew tap wvrdz/tap && brew install fab-kit` to `brew tap sahil87/tap && brew install fab-kit`. Run `grep -n "wvrdz" docs/memory/fab-workflow/migrations.md` after the edit and verify zero matches.

- [x] T017 [P] Update `docs/memory/fab-workflow/kit-architecture.md` line 188: change the live `brew tap wvrdz/tap` reference to `brew tap sahil87/tap`. Do NOT modify the Changelog row at line 618 ("260401-ixzv-org-migrate-mit-license") — that is historical. Run `grep -n "wvrdz" docs/memory/fab-workflow/kit-architecture.md` and confirm only the historical Changelog row matches.

- [x] T018 [P] Update `docs/memory/fab-workflow/index.md` line 14 (the `distribution` row description): refresh the parenthetical from "(4 binaries)" / "5 binaries, 20 cross-compiled" to reflect the new shape — fab-kit installs 2 binaries directly (depends_on for 2 more), 3 binaries built, 12 cross-compiled.

- [x] T019 [P] Rewrite `docs/specs/packages.md`:
  - Replace the lead sentence "Both binaries are compiled from `src/go/` and installed system-wide via Homebrew" with a sentence stating wt and idea are now standalone packages in their own repositories, declared as Homebrew dependencies of fab-kit.
  - Replace `**Binary**: src/go/wt/ (Go binary, included in per-platform release archives)` with a reference to `github.com/sahil87/wt` and the standalone formula `sahil87/tap/wt`.
  - Replace `**Binary**: src/go/idea/ (Go binary, included in per-platform release archives; installed as `idea` via Homebrew)` with a reference to `github.com/sahil87/idea` and the standalone formula `sahil87/tap/idea`.
  - Retain all functional reference content: `wt` subcommand table, `wt create` flags, `wt shell-setup` recipe, `idea` subcommand table, query semantics, worktree behavior. Do NOT modify these.

- [x] T020 [P] Light-touch sweep of `docs/specs/glossary.md`: search for live prose describing wt/idea as fab-kit-internal binaries or any `wvrdz` references in active prose; update to reflect Homebrew-dependency framing. Operational examples (`wt create`, `idea add`) are unchanged. Historical changelog rows (if any) are unchanged.

- [x] T021 [P] Light-touch sweep of `docs/specs/user-flow.md`: same approach as T020.

## Phase 5: Release Prep

- [x] T022 Update `src/kit/VERSION`: bump from `1.6.2` to `1.7.0`. (This is the version that will land this change; the actual `git tag v1.7.0` is performed by `release.sh` post-merge, not in this change.)

- [x] T023 Draft a "What's New" / upgrade-notes addition for the next release notes (this content will be picked up by `just release-notes` from the relevant commits, but the content itself is captured here as a planning artifact in case manual release notes editing is needed): explain the wt/idea decoupling in 1–3 sentences; mention `link_overwrite` handles symlink ownership transfer transparently; document the troubleshooting fallback `brew unlink wt idea && brew upgrade fab-kit` for the rare case it fails. Save the draft as a comment block in the change folder (`fab/changes/260506-4rtx-decouple-wt-idea/release-notes-draft.md`) so it survives PR review without polluting the actual `dist/release-notes.md` (which is regenerated by CI).

---

## Execution Order

- **Phase 1** (T001–T007) is fully parallelizable — all tasks edit different files or different sections of `justfile`. Note: T003–T007 all touch `justfile`; treat them as a single edit group running sequentially within an interactive `Edit` session.
- **Phase 2** (T008–T010) depends on Phase 1 completion (so the build/justfile already has the references removed before the source disappears, avoiding intermediate broken states). T008 and T009 can run in parallel; T010 runs after both.
- **Phase 3** (T011–T014) depends on Phase 2 completion. T011 (test) and T012 (build) can run in parallel. T013 and T014 are post-build verifications and are sequential.
- **Phase 4** (T015–T021) is fully parallelizable across files but depends on Phase 1 (formula template content informs distribution.md descriptions). Each task touches a different file.
- **Phase 5** (T022–T023) depends on Phase 4 to keep the changeset coherent before bumping VERSION.

### justfile edit grouping

T003, T004, T005, T006, T007 all modify `justfile`. The agent should treat these as one focused edit pass through the file rather than five separate `Edit` invocations.
