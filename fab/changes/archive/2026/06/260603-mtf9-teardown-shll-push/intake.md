# Intake: Tear Down Push-Side shll.ai Integration

**Change**: 260603-mtf9-teardown-shll-push
**Created**: 2026-06-03
**Status**: Draft

## Origin

Initiated from a `/fab-discuss` session followed by `/fab-fff`. The user pointed at
shll.ai's frozen contract spec
(`sahil87/shll.ai` → `docs/specs/help-dump-contract.md`, the
"Teardown directive (paste to a tool-repo agent)" section) and asked to understand the
integration update.

> There's an update in the way we integrate with shll.ai. [...] Tear down the deprecated
> push-side shll.ai integration now that shll.ai's puller is live.

**Interaction mode**: Conversational (`/fab-discuss`) → pipeline (`/fab-fff`).

**The contract change**: shll.ai has flipped its integration model from **push** to
**pull**. Previously each of 7 tool repos *pushed* their `help-dump` JSON into
`sahil87/shll.ai` via an auto-merging cross-repo PR. Now shll.ai runs its own **puller**
that invokes each tool's `help-dump` subcommand and does all capture / timestamp /
validate / commit / render itself. The tool repo's only remaining responsibility is to
*emit valid `help-dump` output*.

**Key decisions reached in discussion** (encoded as assumptions below):
- The user confirmed shll.ai's puller is **already live and proven** — the contract's
  safety precondition for teardown is met, so there is no stale-help gap.
- The user chose to remove the **entire** `Help-dump → shll.ai` CI step, including the
  fatal `help-dump` + `jq` dump/validate self-check (not just the PR transport). The
  `help-dump` command remains exercised by its Go unit tests (`helpdump_test.go`).

## Why

1. **Problem**: fab-kit's `release.yml` carries a now-obsolete push mechanism. Every
   release it dumps `help/fab-kit.json` and opens an auto-merging cross-repo PR into
   `sahil87/shll.ai` authenticated with the `SHLLAI_TOKEN` secret. With shll.ai's puller
   live, this transport is dead weight: shll.ai pulls the help tree itself by invoking
   `fab help-dump`, so the push duplicates work and maintains a cross-repo write
   credential that is no longer needed.

2. **Consequence of inaction**: A cross-repo write token (`SHLLAI_TOKEN`, contents +
   pull-requests write) stays provisioned with no purpose — unnecessary credential
   surface. The release pipeline keeps running a no-longer-needed step that opens
   redundant PRs against shll.ai (which the puller's own commits now supersede), creating
   PR churn and potential confusion about which side is authoritative.

3. **Why this approach**: The shll.ai contract explicitly directs tool-repo agents to
   tear down four components (producer CI, PR-opening logic, auto-merge wiring,
   `SHLLAI_TOKEN`) once the puller is live, while leaving the `help-dump` command
   untouched as the contract surface. We follow that directive exactly. The teardown is
   safe *now* specifically because the precondition ("puller is live and proven") is
   confirmed — the contract warns that removing producer CI before that creates a
   stale-help gap.

## What Changes

### 1. Remove the `Help-dump → shll.ai` step from `release.yml`

Delete the entire step (currently `.github/workflows/release.yml` lines ~73–137),
including its leading comment block. This removes **both** sub-parts:

- **Dump + validate** (the fatal self-check):
  ```yaml
  ./dist/bin/fab-go-linux-amd64 help-dump > help/fab-kit.json
  jq -e '.tool=="fab" and .schema_version==1 and (.version|length>0) and (.root|type=="object")' help/fab-kit.json
  ```
- **The auto-merging cross-repo PR** (the transport): the `SHLLAI_TOKEN` `env:`, the
  `git clone` of `sahil87/shll.ai`, the `help-dump/fab-kit-<version>` branch, the
  `gh pr create`, and the `gh pr merge --auto --squash`.

The step currently sits between `Build all targets` and `Package kit archives`. After
removal, `Build all targets` is immediately followed by `Package kit archives` — the
CI step list returns from 7 entries to 5.

**Maps to the contract's four teardown components:**

| Contract component | Location in fab-kit | Action |
|---|---|---|
| 1. Producer CI (walks tree → JSON) | dump+validate lines (~87–90) | Remove |
| 2. PR-opening logic | clone/branch/`gh pr create` (~92–135) | Remove |
| 3. Auto-merge wiring | `gh pr merge --auto --squash` (~136) | Remove |
| 4. `SHLLAI_TOKEN` secret | `env:` (line ~79) + repo settings | Remove from workflow; delete repo secret after confirming no other use |

### 2. Remove the dead `/help/` `.gitignore` entry

The `/help/` ignore line was added by the push change (`xob7`) because the step wrote a
transient `help/fab-kit.json`. With the step gone, nothing writes to `help/`, so the
entry is dead and SHOULD be removed for cleanliness. (Non-functional; safe either way,
but the contract's spirit is to leave no push-side residue.)

### 3. Do NOT touch the `help-dump` command — contract invariant

`src/go/fab/cmd/fab/helpdump.go` and `src/go/fab/cmd/fab/helpdump_test.go` MUST remain
untouched. The contract states explicitly: *"do NOT touch the `help-dump` command"* — the
subcommand is the contract surface shll.ai's puller invokes. Only the CI transport layer
is removed. The command stays covered by its existing Go unit tests.

### 4. Delete the `SHLLAI_TOKEN` repo secret (manual, out-of-band)

After the workflow no longer references `SHLLAI_TOKEN`, the secret SHOULD be deleted from
fab-kit's GitHub repo settings — but only after confirming no other workflow uses it. A
repo-wide grep confirms it is referenced **only** in the `Help-dump → shll.ai` step. The
actual secret deletion happens in GitHub settings (not a file edit) and will be called
out as a manual follow-up; the code change cannot perform it.

## Affected Memory

- `fab-workflow/distribution`: (modify) Remove the `Help-dump → shll.ai` release-step
  documentation (the dedicated subsection ~lines 279–285) and update the change-log table
  entry to record the teardown. The release CI step list returns to its pre-`xob7` shape.

## Impact

- **`.github/workflows/release.yml`** — delete the `Help-dump → shll.ai` step (~73–137).
  No other step depends on its outputs (`help/fab-kit.json` is consumed only by the
  removed PR sub-step).
- **`.gitignore`** — remove the `/help/` entry.
- **`docs/memory/fab-workflow/distribution.md`** — remove the push-step subsection and add
  a change-log entry (hydrate stage).
- **GitHub repo settings (manual)** — delete the `SHLLAI_TOKEN` secret. Out-of-band;
  noted as a follow-up.
- **NOT touched**: `src/go/fab/cmd/fab/helpdump.go`, `helpdump_test.go`,
  `src/kit/skills/_cli-fab.md` (the `help-dump` command and its docs survive — only the
  transport is removed).
- **External dependency**: relies on shll.ai's puller being live (confirmed). No code
  coupling to shll.ai remains after this change.

## Open Questions

None. The discussion resolved the two decision points (precondition confirmed; remove the
whole step including dump+validate). The only non-code action — deleting the
`SHLLAI_TOKEN` repo secret — is an out-of-band GitHub-settings task surfaced as a
follow-up.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | shll.ai's puller is live and proven, satisfying the teardown safety precondition. | Discussed — user explicitly confirmed "Yes, confirmed live" when asked about the precondition. Removes the stale-help-gap risk the contract warns about. | S:98 R:60 A:90 D:95 |
| 2 | Certain | Remove the **entire** `Help-dump → shll.ai` step, including the fatal dump+validate self-check — not just the PR transport. | Discussed — user explicitly chose "Remove entirely" over keeping the dump+validate as a release smoke test. `help-dump` stays covered by `helpdump_test.go`. | S:95 R:70 A:85 D:90 |
| 3 | Certain | Do NOT touch `help-dump` (`helpdump.go` / `helpdump_test.go`) — it is the contract surface the puller invokes. | Dictated by the shll.ai contract's explicit invariant: "do NOT touch the help-dump command." | S:100 R:80 A:95 D:95 |
| 4 | Confident | Remove the dead `/help/` `.gitignore` entry as part of the teardown. | Nothing writes to `help/` once the step is gone; removing the entry leaves no push-side residue. Cosmetic/non-functional, trivially reversible. | S:70 R:90 A:85 D:80 |
| 5 | Confident | `SHLLAI_TOKEN` is used only by the removed step; the secret can be deleted from repo settings as a manual follow-up. | Repo-wide grep shows `SHLLAI_TOKEN` referenced only in the `Help-dump → shll.ai` step. Secret deletion is out-of-band (GitHub settings), so noted as a follow-up, not a file edit. | S:85 R:75 A:90 D:85 |
| 6 | Confident | Update `docs/memory/fab-workflow/distribution.md` to remove the push-step subsection and add a teardown change-log entry. | Memory is the source of truth for the release pipeline; it currently documents the now-removed step. Handled at hydrate. Easily revised. | S:80 R:85 A:90 D:85 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
