# Tasks: Remove Rust Implementation

**Change**: 260312-96nf-remove-rust-implementation
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Delete Rust Source and Build Script

- [x] T001 [P] Delete `src/rust/` directory tree (all Cargo files, source, and tests)
- [x] T002 [P] Delete `src/scripts/just/rust-target.sh` helper script

## Phase 2: Build System and Dispatcher

- [x] T003 Remove Rust-related content from `justfile`: delete `rust_src` variable, `test-rust`, `build-rust`, `_rust-target`, `build-rust-target`, `build-rust-all` recipes, and Rust comment in `build-all`
- [x] T004 [P] Simplify `fab/.kit/bin/fab` dispatcher: remove `fab-rust` from `--version` handler, backend override block, default priority block, and error message
- [x] T005 [P] Remove `--exclude='.kit/bin/fab-rust'` from `src/scripts/just/package-kit.sh` generic archive tar command

## Phase 3: Skills and Documentation

- [x] T006 [P] Update `fab/.kit/skills/_scripts.md`: remove `fab-rust` from "Backend Priority" section and dispatcher description; update to Go-only
- [x] T007 [P] Update `docs/memory/fab-workflow/kit-architecture.md`: remove `fab-rust` from directory tree, remove/update Rust Binary section, update dispatcher description, remove Rust benchmark data, update overview to Go-only
- [x] T008 [P] Update `docs/memory/fab-workflow/distribution.md`: remove `fab-rust` from archive descriptions, remove Rust build recipes, remove Rust CI steps, remove cargo-zigbuild/Zig references, update backend override to Go-only, remove "Transition Period" section, update bootstrap and packaging descriptions
- [x] T009 [P] Update `docs/specs/packages.md`: remove `fab-rust` line from `fab/.kit/bin/` directory tree

---

## Execution Order

- T001 and T002 are independent, can run in parallel
- T003, T004, T005 can run in parallel (different files)
- T006-T009 can all run in parallel (different files, no dependencies)
- Phase 3 has no dependency on Phase 2 — all phases could technically run in parallel since all tasks touch different files
