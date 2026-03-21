# Tasks: Pin GitHub Actions to commit SHAs for supply chain safety

**Change**: 260321-nq8j-pin-github-actions-sha
**Spec**: `spec.md`
**Intake**: `intake.md`

## Phase 1: Core Implementation

- [x] T001 [P] Pin `actions/checkout@v4` to SHA `34e114876b0b11c390a56381ad16ebd13914f8d5` in `.github/workflows/release.yml`
- [x] T002 [P] Pin `actions/setup-go@v5` to SHA `40f1582b2485089dde7abd97c1529aa768e1baff` in `.github/workflows/release.yml`
- [x] T003 [P] Pin `extractions/setup-just@v2` to SHA `dd310ad5a97d8e7b41793f8ef055398d51ad4de6` in `.github/workflows/release.yml`

---

## Execution Order

- T001, T002, T003 are independent [P] tasks applied in a single commit.
