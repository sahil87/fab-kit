# Intake: Smart Copilot Review Detection

**Change**: 260303-n30u-smart-copilot-review-detection
**Created**: 2026-03-03
**Status**: Draft

## Origin

> User asked: "In the `git-pr` command, is there a way to confirm whether a Copilot review has been even requested or not?" — followed by an empirical testing session against the GitHub API to validate the approach before designing the change.

One-shot interaction mode, preceded by a `/fab-discuss` exploration session. Key decisions were validated empirically via live API calls against PRs #191 and #192 before this intake was created.

## Why

`git-pr` Step 6 currently polls `GET /pulls/{n}/reviews` for up to 6 minutes (30s intervals, 12 attempts) hoping a Copilot review appears. On repos without Copilot review enabled, this wastes 6 minutes every time. On repos with Copilot, there's no explicit request — it relies on GitHub's auto-trigger, which may not fire for all PR types.

If we don't fix it: every `/git-pr` invocation on a repo without Copilot burns 6 minutes of polling for nothing. Even on repos with Copilot, the flow is fragile — it can't distinguish "review is coming" from "review will never come."

The approach: use the GitHub API to explicitly request a Copilot review, and treat any failure as "not available" — skipping polling entirely. This is simpler and more robust than trying to detect Copilot enablement through repo settings.

## What Changes

### Replace blind polling with a 3-phase detect/request/poll flow

The current Step 6 in `git-pr.md` and Step 2 in `git-pr-fix.md` are replaced with a smarter approach:

**Phase 1 — Check if already reviewed:**
```bash
gh api repos/{owner}/{repo}/pulls/{number}/reviews \
  --jq '.[] | select(.user.login == "copilot-pull-request-reviewer[bot]") | .id'
```
If a review ID is returned → skip directly to triage (Step 3 of `git-pr-fix`). This handles re-runs and PRs where the review completed before we checked.

**Phase 2 — Request review:**
```bash
gh api repos/{owner}/{repo}/pulls/{number}/requested_reviewers \
  -X POST -f 'reviewers[]=copilot-pull-request-reviewer[bot]'
```
- **Any non-2xx response** (422, 403, 404, network error, etc.) → print `Copilot review not available — skipping.` and exit. No polling.
- **Success (2xx)** → proceed to Phase 3.

**Phase 3 — Poll for completion:**
Same as today: check `GET /pulls/{n}/reviews` for `copilot-pull-request-reviewer[bot]` every 30 seconds, max 12 attempts. Timeout → print timeout message and exit.

### API login name discrepancy (empirically confirmed)

The Copilot bot uses different login names in different API contexts:
- In `GET /requested_reviewers` response: `"login": "Copilot"` (type: Bot)
- In `GET /reviews` response (once submitted): `"login": "copilot-pull-request-reviewer[bot]"`
- In `POST /requested_reviewers` request body: `reviewers[]=copilot-pull-request-reviewer[bot]`

This was confirmed empirically on PR #192 (wvrdz/fab-kit). The skill files should document this discrepancy so future maintainers understand why different login strings appear in different API calls.

### Affected skills and modes

| Skill | Mode | Current behavior | New behavior |
|-------|------|-----------------|--------------|
| `git-pr` Step 6 | Wait mode (inline) | Blind poll 12x30s | Phase 1 → 2 → 3 |
| `git-pr-fix` Step 2 | Wait mode (from git-pr) | Blind poll 12x30s | Phase 1 → 2 → 3 |
| `git-pr-fix` Step 2 | Standalone mode | Single check, bail if absent | Phase 1 → 2, single-check Phase 3 (no polling loop) |

Standalone mode change: currently does a single `GET /reviews` check and bails. New behavior adds the `POST /requested_reviewers` attempt first — if it fails, bail immediately (same as before, but now with a clear "not available" message rather than "not found"). If it succeeds but no review exists yet, do a single check and bail (the standalone caller can re-run later).
<!-- assumed: standalone mode should also attempt the POST request — consistent behavior across modes, and the POST is idempotent -->

## Affected Memory

- `fab-workflow/execution-skills`: (modify) Update git-pr and git-pr-fix copilot review detection logic

## Impact

- `fab/.kit/skills/git-pr.md` — Step 6 rewrite
- `fab/.kit/skills/git-pr-fix.md` — Step 2 rewrite
- `.claude/skills/git-pr/SKILL.md` — installed copy sync
- `.claude/skills/git-pr-fix/SKILL.md` — installed copy sync

No script changes. No template changes. No config changes.

## Open Questions

- None — the approach was validated empirically before intake creation.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Use POST /requested_reviewers to detect Copilot availability | Discussed — empirically validated on PR #192, user chose this approach | S:95 R:90 A:95 D:95 |
| 2 | Certain | Treat any non-2xx from POST as "not available" | Discussed — user explicitly requested "handle all kinds of errors in the same flow" | S:95 R:90 A:90 D:95 |
| 3 | Certain | Keep same polling parameters (30s, 12 attempts) for Phase 3 | Discussed — no change to polling behavior itself, only to the gating logic | S:90 R:95 A:90 D:95 |
| 4 | Confident | Standalone git-pr-fix should also attempt the POST request | Consistent behavior across modes; POST is idempotent; adds "not available" clarity to standalone mode | S:70 R:85 A:80 D:75 |
| 5 | Certain | Document the login name discrepancy in skill files | Discussed — empirically confirmed: "Copilot" vs "copilot-pull-request-reviewer[bot]" across endpoints | S:95 R:95 A:95 D:95 |

5 assumptions (4 certain, 1 confident, 0 tentative, 0 unresolved).
