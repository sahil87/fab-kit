# Spec: Pin GitHub Actions to commit SHAs for supply chain safety

**Change**: 260321-nq8j-pin-github-actions-sha
**Created**: 2026-03-21
**Affected memory**: None

## CI: GitHub Actions Supply Chain Hardening

### Requirement: External action references SHALL use commit SHA pins

Every `uses:` line in `.github/workflows/release.yml` that references an external (non-org) GitHub Action MUST specify a full 40-character commit SHA instead of a mutable version tag. The original version tag MUST be preserved as a trailing `# tag` comment for readability and upgrade tracking.

#### Scenario: Tag-pinned action is replaced with SHA
- **GIVEN** `.github/workflows/release.yml` contains `uses: actions/checkout@v4`
- **WHEN** the supply chain hardening change is applied
- **THEN** the line reads `uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4`
- **AND** the SHA is a valid 40-character hexadecimal string

#### Scenario: All 3 external actions are pinned
- **GIVEN** `release.yml` references 3 external actions: `actions/checkout@v4`, `actions/setup-go@v5`, `extractions/setup-just@v2`
- **WHEN** the change is applied
- **THEN** all 3 references use commit SHA pins with tag comments:
  - `actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4`
  - `actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5`
  - `extractions/setup-just@dd310ad5a97d8e7b41793f8ef055398d51ad4de6 # v2`
- **AND** no external action reference uses a bare version tag

#### Scenario: Internal actions are not modified
- **GIVEN** a workflow references an internal action like `wvrdz/github-actions@stable`
- **WHEN** the change is applied
- **THEN** the internal action reference remains unchanged

### Requirement: Workflow behavior SHALL be unchanged

The release workflow MUST produce identical behavior before and after SHA pinning. This is purely a reference format change — no functional, permission, or configuration differences.

#### Scenario: Release workflow succeeds after pinning
- **GIVEN** the SHA-pinned `release.yml` is committed and pushed
- **WHEN** a version tag (`v*`) is pushed to the repository
- **THEN** the release workflow triggers and completes successfully
- **AND** Go binaries are cross-compiled, kit archives are packaged, and a GitHub release is created

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only `release.yml` in scope | Only workflow file in repo — confirmed from intake #1 | S:95 R:95 A:95 D:95 |
| 2 | Certain | Internal actions excluded from pinning | Org-controlled, no supply chain risk — confirmed from intake #2 | S:95 R:90 A:95 D:95 |
| 3 | Certain | Tag comments use `# tag` format | Org convention from parent context — confirmed from intake #3 | S:90 R:95 A:90 D:90 |
| 4 | Confident | SHAs match current tag heads at pin time | Non-functional change, verified during implementation — confirmed from intake #4 | S:80 R:90 A:70 D:85 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
