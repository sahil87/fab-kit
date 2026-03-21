# Intake: Pin GitHub Actions to commit SHAs for supply chain safety

**Change**: 260321-nq8j-pin-github-actions-sha
**Created**: 2026-03-21
**Status**: Draft

## Origin

> INFRA-345: Supply chain hardening — pin all external GitHub Action references in fab-kit to commit SHAs.

Part of org-wide initiative prompted by two major supply chain attacks:
1. **Shai-Hulud 2.0** (Nov 2025) — npm worm compromised PostHog + 754 packages via `preinstall` hooks, exfiltrated cloud creds and tokens.
2. **Trivy Tag Poisoning** (Mar 2026) — TeamPCP force-pushed 75/76 version tags in `aquasecurity/trivy-action`, dumping runner process memory.

Parent intake context at `/tmp/fab-intake-fab-kit.md`. One-shot intake — all decisions are clear from parent context.

## Why

Our org-wide assessment found 0/358 GitHub Action references were SHA-pinned — all used mutable version tags. If an attacker force-pushes a tag (as happened with `trivy-action`), every CI run using that tag executes attacker-controlled code with full access to repository secrets and `GITHUB_TOKEN`.

SHA-pinning makes action references immutable: the resolved code is locked to a specific commit regardless of what happens to the tag. This is a safe, non-functional change — actions resolve to the same code they were using before.

## What Changes

### SHA-pin 3 external GitHub Actions in `.github/workflows/release.yml`

Replace mutable version-tag references with full 40-character commit SHA pins. The original tag is preserved as a trailing `# tag` comment for readability and upgrade tracking:

```yaml
# Before
- uses: actions/checkout@v4
- uses: actions/setup-go@v5
- uses: extractions/setup-just@v2

# After
- uses: actions/checkout@34e114876b0b11c390a56381ad16ebd13914f8d5 # v4
- uses: actions/setup-go@40f1582b2485089dde7abd97c1529aa768e1baff # v5
- uses: extractions/setup-just@dd310ad5a97d8e7b41793f8ef055398d51ad4de6 # v2
```

Internal actions (`wvrdz/github-actions@stable`) are not pinned — they are org-controlled and not subject to external supply chain risk.

## Affected Memory

- None — CI hardening change with no spec-level behavior impact on the fab-kit workflow.

## Impact

- **File**: `.github/workflows/release.yml` — only file affected (only workflow in repo)
- **Behavior**: Unchanged — SHAs resolve to the same commits the tags pointed to at pin time
- **Maintenance**: Future action upgrades require updating the SHA + comment tag rather than bumping the version tag alone

## Open Questions

- None — scope, approach, and specific SHAs are all determined by the parent context.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Only `release.yml` needs pinning | Only workflow file in this repo | S:95 R:95 A:95 D:95 |
| 2 | Certain | Internal actions (`wvrdz/*`) excluded | Org-controlled, per parent context directive | S:95 R:90 A:95 D:95 |
| 3 | Certain | Tag preserved as trailing `# tag` comment | Org convention defined in parent context | S:90 R:95 A:90 D:90 |
| 4 | Confident | SHAs resolve to current tag heads at pin time | Verified during implementation; safe non-functional change | S:80 R:90 A:70 D:85 |

4 assumptions (3 certain, 1 confident, 0 tentative, 0 unresolved).
