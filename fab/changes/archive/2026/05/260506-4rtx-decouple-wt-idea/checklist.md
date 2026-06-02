# Quality Checklist: Decouple wt and idea from fab-kit

**Change**: 260506-4rtx-decouple-wt-idea
**Generated**: 2026-05-06
**Spec**: `spec.md`

## Functional Completeness

- [ ] CHK-001 fab-kit Formula Declares wt and idea as Dependencies: `.github/formula-template.rb` contains `depends_on "sahil87/tap/wt"` and `depends_on "sahil87/tap/idea"`; `def install` block contains exactly `bin.install "fab"` and `bin.install "fab-kit"`; `test do` block contains exactly two `assert_match` calls.
- [ ] CHK-002 wt and idea Standalone Formulas Use link_overwrite: `link_overwrite "bin/wt"` and `link_overwrite "bin/idea"` are present in the deployed `sahil87/homebrew-tap` formulas (commit `8f9ade1`) and in the templates of `sahil87/wt` (`c264680`) and `sahil87/idea` (`6745505`). Pre-shipped — verify via `cd ../homebrew-tap && grep link_overwrite Formula/{wt,idea}.rb`.
- [ ] CHK-003 justfile Build Targets Three Binaries Per Platform: `justfile` `test`/`test-v` recipes only `cd` into `src/go/fab/` and `src/go/fab-kit/`; `build` rename loop covers `fab-go fab-kit fab` only; `build-target` invokes `_build-binary` exactly three times; `build-all` comment reflects 12 binaries.
- [ ] CHK-004 package-brew Stages Two Binaries: `scripts/just/package-brew.sh` stages exactly `fab` and `fab-kit`; the existence-check loop, `cp`, `chmod`, and `tar` arg list all reflect the two-binary set.
- [ ] CHK-005 src/go/wt and src/go/idea are Removed: directories no longer exist; `ls src/go/` shows only `fab/` and `fab-kit/`.
- [ ] CHK-006 Runtime exec.Command Sites Continue to Work: `src/go/fab/cmd/fab/batch_new.go:116` and `batch_switch.go:96` continue to invoke `exec.Command("wt", ...)` unchanged.
- [ ] CHK-007 docs/memory/fab-workflow/distribution.md Reflects New Topology: 4-binary/5-binary/20-binary references replaced; binary table split into fab-kit-owned vs external dependencies; new Design Decision entry referencing 260506-4rtx.
- [ ] CHK-008 docs/memory/fab-workflow/migrations.md Sweeps Stale Tap: live prose contains `sahil87/tap`, no live `wvrdz/tap` references.
- [ ] CHK-009 docs/memory/fab-workflow/kit-architecture.md Sweeps Stale Tap: live prose contains `sahil87/tap`; only the historical Changelog row at line 618 retains `wvrdz`.
- [ ] CHK-010 docs/specs/packages.md Reframes wt and idea: `**Binary**: src/go/...` lines replaced with references to `github.com/sahil87/{wt,idea}` and standalone formulas; functional reference content (subcommand tables, flags, recipes) retained.
- [ ] CHK-011 Light-Touch Sweeps in Glossary and User-Flow: `docs/specs/glossary.md` and `docs/specs/user-flow.md` reviewed for "fab-kit-internal binary" framing or `wvrdz` strings; corrected if found, no changes if absent.
- [ ] CHK-012 release.yml Requires No Structural Changes: `.github/workflows/release.yml` is unmodified; the `Update Homebrew tap` step continues to fully overwrite `Formula/fab-kit.rb` only.
- [ ] CHK-013 First Release Carrying This Change is fab-kit 1.7.0: `src/kit/VERSION` is `1.7.0`.
- [ ] CHK-014 Release Notes Include Upgrade Troubleshooting Note: a release-notes draft documents the wt/idea decoupling in 1–3 sentences and the `brew unlink wt idea && brew upgrade fab-kit` fallback (framed as fallback, not required pre-step).

## Behavioral Correctness

- [ ] CHK-015 Brew tarball contents: `tar tzf dist/brew-darwin-arm64.tar.gz` lists exactly `fab` and `fab-kit` post-`just package-brew`; the same applies to `darwin-amd64`, `linux-arm64`, `linux-amd64`.
- [ ] CHK-016 Build matrix shrinks 5→3, 20→12: `just build-all` produces exactly 12 binaries in `dist/bin/` (`{fab,fab-kit,fab-go} × {darwin,linux} × {arm64,amd64}`); no `wt-*` or `idea-*` binaries.
- [ ] CHK-017 Test invocations limited to fab and fab-kit: `just test` does not attempt to enter `src/go/wt/` or `src/go/idea/` (no missing-directory errors).
- [ ] CHK-018 Source removal is total: `grep -r "src/go/idea\|src/go/wt\|fab-kit/src/go/idea\|fab-kit/src/go/wt" src/go/` produces zero matches.

## Removal Verification

- [ ] CHK-019 `src/go/wt/` directory absent (no `cmd/`, `internal/`, `go.mod`, `go.sum`).
- [ ] CHK-020 `src/go/idea/` directory absent (no `cmd/`, `internal/`, `go.mod`, `go.sum`).
- [ ] CHK-021 No `bin.install "wt"` or `bin.install "idea"` lines remain in `.github/formula-template.rb`.
- [ ] CHK-022 No `wt --version` or `idea --version` assertions remain in `.github/formula-template.rb` `test do` block.
- [ ] CHK-023 No `wt=` / `idea=` variable assignments or `cp`/`chmod` lines for wt/idea remain in `scripts/just/package-brew.sh`.
- [ ] CHK-024 No `_build-binary src/go/wt` or `_build-binary src/go/idea` invocations remain in `justfile`.

## Scenario Coverage

- [ ] CHK-025 "Fresh install resolves all four binaries" scenario validated: a clean machine running `brew install sahil87/tap/fab-kit` from the new formula gets `fab`, `fab-kit`, `wt`, `idea` all on PATH (best-effort verification — full validation requires post-release manual check; document the test plan).
- [ ] CHK-026 "Upgrade from fab-kit 1.6.2 with bundled wt/idea" scenario test plan documented: explicit upgrade path described in release notes (run `brew upgrade fab-kit` and verify `wt --version` reports the standalone formula's version, not `1.6.2`).
- [ ] CHK-027 "just build-all produces 12 binaries" scenario verified locally: post-build, `dist/bin/` contains exactly 12 files matching the pattern (T013).
- [ ] CHK-028 "Brew tarball contents" scenario verified locally: `tar tzf dist/brew-darwin-arm64.tar.gz` lists exactly `fab` and `fab-kit` (T014).
- [ ] CHK-029 "Source tree no longer contains the modules" scenario verified: `ls src/go/` shows only `fab/` and `fab-kit/`.

## Edge Cases & Error Handling

- [ ] CHK-030 Idempotent overwrite of `Formula/fab-kit.rb`: confirm `.github/workflows/release.yml`'s `cp dist/fab-kit.rb /tmp/tap/Formula/fab-kit.rb` is a full overwrite — no stale `bin.install "wt"` line could linger after release.
- [ ] CHK-031 Historical content preserved: changelog rows in `distribution.md` (line 360 "260401-46hw", line 361 "260401-ixzv"), `kit-architecture.md` (line 618), and any other historical changelog tables remain untouched.
- [ ] CHK-032 Operational examples preserved: `wt create`, `idea add`, command-map references in `glossary.md` and `user-flow.md` are unchanged in syntax.

## Code Quality

- [ ] CHK-033 Pattern consistency: documentation edits follow the existing tone, structure, and section conventions of each affected memory/spec file.
- [ ] CHK-034 No unnecessary duplication: the `link_overwrite` documentation in `distribution.md` references the implementation in the tap formulas rather than reproducing the formula source.
- [ ] CHK-035 Readability over cleverness (project principle): documentation rewrites use direct, minimal prose — no redundant qualifiers, no defensive hedging where assertions are warranted.
- [ ] CHK-036 No god-function refactoring (project anti-pattern): all edits are scoped to declared targets; no opportunistic restructuring of unrelated sections.
- [ ] CHK-037 No magic strings (project anti-pattern): tap names (`sahil87/tap`), formula names (`fab-kit`, `wt`, `idea`), and version numbers (`1.7.0`) appear consistently across all edited files.

## Documentation Accuracy (project-extra)

- [ ] CHK-038 Cross-reference integrity: any inter-file references in edited memory/spec docs (links to other domains, change folders) remain valid post-edit.
- [ ] CHK-039 Affected memory header in `spec.md` accurately lists `distribution.md`, `migrations.md`, `kit-architecture.md` as files that hydrate will modify.
- [ ] CHK-040 Versioning claims consistent: every place fab-kit's binary count is mentioned in memory/specs reflects the new 2-direct + 2-via-`depends_on` topology.

## Cross References (project-extra)

- [ ] CHK-041 Standalone repos linked: `docs/specs/packages.md` and `docs/memory/fab-workflow/distribution.md` link or reference `github.com/sahil87/wt` and `github.com/sahil87/idea` consistently.
- [ ] CHK-042 Tap commit references in change artifacts: where the spec/intake mentions tap commit `8f9ade1`, wt commit `c264680`, idea commit `6745505`, the references are accurate.
- [ ] CHK-043 Constitution Principle VI honored: `docs/specs/` files (packages.md, glossary.md, user-flow.md) are edited by the human-curated path (manual edit), not auto-generated by tooling.

## Notes

- Check items as you review: `- [x]`
- All items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] CHK-NNN **N/A**: {reason}`
