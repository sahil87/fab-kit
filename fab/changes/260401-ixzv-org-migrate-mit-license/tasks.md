# Tasks: Org Migration (wvrdz → sahil87) and MIT License

**Change**: 260401-ixzv-org-migrate-mit-license
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: License

- [x] T001 Create root `LICENSE` with MIT license text (copyright "2026 Sahil Ahuja"), delete `fab/.kit/LICENSE`

## Phase 2: Config and Scripts

- [x] T002 [P] Update `fab/.kit/kit.conf` line 4: `repo=wvrdz/fab-kit` → `repo=sahil87/fab-kit`
- [x] T003 [P] Update `scripts/install.sh`: change `REPO="wvrdz/fab-kit"` and comment URL from `wvrdz` to `sahil87`
- [x] T004 [P] Update `README.md`: change install curl URL from `wvrdz/fab-kit` to `sahil87/fab-kit`

## Phase 3: Go Modules

- [x] T005 [P] Update `src/go/fab/go.mod` module path and all import paths in `src/go/fab/` source files: `github.com/wvrdz/fab-kit` → `github.com/sahil87/fab-kit`
- [x] T006 [P] Update `src/go/idea/go.mod` module path and all import paths in `src/go/idea/` source files: `github.com/wvrdz/fab-kit` → `github.com/sahil87/fab-kit`
- [x] T007 [P] Update `src/go/wt/go.mod` module path and all import paths in `src/go/wt/` source files: `github.com/wvrdz/fab-kit` → `github.com/sahil87/fab-kit`

## Phase 4: Verify

- [x] T008 Run `go mod tidy` in `src/go/fab/`, `src/go/idea/`, `src/go/wt/` and verify `go build ./...` succeeds in each

---

## Execution Order

- T002, T003, T004 are independent of each other and T001
- T005, T006, T007 are independent of each other
- T008 depends on T005, T006, T007
