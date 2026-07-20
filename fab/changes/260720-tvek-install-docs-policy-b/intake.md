# Intake: Install Docs Policy B Conformance

**Change**: 260720-tvek-install-docs-policy-b
**Created**: 2026-07-20

## Origin

One-shot `/fab-new` invocation with a fully-specified task:

> Conform this repo's install documentation to the shll toolkit's install-composition standard, Policy B. Read the authoritative standard first: /home/sahil/code/sahil87/shll/docs/site/standards/install-composition.md (rendered on https://shll.ai). Policy B: per-tool READMEs and doc pages must not carry per-formula "brew install sahil87/tap/\<tool\>" install instructions; installation points to https://shll.ai (curl bootstrap: `curl -fsSL https://shll.ai/install | sh`; subset installs remain supported via `shll install <tool>`). Task: audit README.md and docs/site/ for per-formula install instructions and replace them with the shll.ai pointer. IMPORTANT distinction: replace install *instructions* (sections telling the user how to install), but KEEP incidental mentions such as actionable error-hint examples in standards/conformance text (Policy A mandates those hints) and historical/changelog references. Mechanical docs-only change; keep all usage and feature content intact.

The authoritative standard was read at intake time (`sahil87/shll` repo, `docs/site/standards/install-composition.md`). Key normative lines:

- **Policy B**: "Per-tool READMEs and the tap README MUST NOT carry per-formula `brew install` instructions. They link to https://shll.ai for install steps — the curl bootstrap or `shll install`."
- **Supported-vs-unsupported line**: individual formula installs remain *supported* (`brew install sahil87/tap/<tool>` works); what is unsupported is *documenting* them per-repo.
- **Policy A context** (already shipped for this repo as PR #511): missing-sibling paths must emit an actionable install hint of the verbatim form `<tool> is not installed. Install it: brew install sahil87/tap/<tool>` — so hint *examples* quoted in docs are mandated content, not violations.

The full audit of README.md and docs/site/ was performed at intake time; the complete hit list with replace/keep verdicts is in **What Changes** below.

## Why

1. **Pain point**: fab-kit's docs still carry per-formula install instructions in two files — `docs/site/install.md` tells users to `brew tap sahil87/tap && brew install fab-kit` and `brew install sahil87/tap/wt sahil87/tap/idea`, and README.md repeats the wt/idea per-formula line twice. The constitution's Toolkit Standards article (v1.4.0) binds this repo to the shll published standards, and `install-composition` Policy B forbids exactly this: seven copies of the install dance drift, and every change to the install story (tap-trust requirement, bootstrap change) has to be chased across every repo plus the tap.
2. **Consequence of not fixing**: fab-kit remains non-conformant with a binding standard; the next bootstrap or tap change silently strands these pages with stale instructions (the exact drift failure mode the standard names).
3. **Approach**: mechanical docs-only replacement, mirroring README.md's `## Install` section which is *already* conformant (curl bootstrap, added when the shll pointer landed). No binary changes — the Go hint strings (`batch_new.go`, `batch_switch.go`, doctor, update) are Policy A's binary half and are mandated, not violations. The companion PR #511 already shipped Policy A's formula/degradation half; this change completes the repo's install-composition conformance with the Policy B docs half.

## What Changes

Full audit result. Grep basis: `brew install|brew tap|shll.ai/install` over README.md and docs/site/ (docs/site/skill.md and docs/site/workflows.md have zero hits — no changes there).

### docs/site/install.md — "Install the CLI" section (lines 12–17): REPLACE

Current:

```bash
brew tap sahil87/tap
brew install fab-kit
```

Replace the Homebrew framing + code block with the shll.ai curl bootstrap, matching README's conformant Install section:

```sh
curl -fsSL https://shll.ai/install | sh -s -- fab-kit
```

with prose noting: installs fab-kit (plus the shll meta-CLI) via Homebrew with tap trust handled automatically; to install the entire shll toolkit instead: `curl -fsSL https://shll.ai/install | sh`; link https://shll.ai as the canonical install reference. **Keep** the two-CLI role table (`fab` router / `fab-kit` lifecycle) — that is feature content, not install instruction.

### docs/site/install.md — companions block (lines 26–42): REPLACE the instruction, KEEP the hint example

- Line 29 code block `brew install sahil87/tap/wt sahil87/tap/idea` → replace with the shll subset-install form: `shll install wt idea` (with a https://shll.ai pointer). Adjust the immediately-preceding sentence ("install from their own formulas") to say they install via `shll install` / shll.ai; they remain independent projects with their own release cadences.
- **KEEP line 39 verbatim**: the parenthetical error-hint example ``(`wt is required for 'fab batch new' — install it via: brew install sahil87/tap/wt`)`` quotes the actual binary hint (verified: matches `src/go/fab/cmd/fab/batch_new.go:66` exactly). Policy A mandates actionable hints; quoting the real hint in degradation prose is an incidental mention, not an install instruction.
- **Keep** the wt/idea role table and the graceful-degradation prose — usage/feature content.

### docs/site/install.md — third-party utilities (lines 50, 88): KEEP

`brew install yq jq gh direnv` and `brew install go just` are **not** toolkit formulas — Policy B governs per-formula `sahil87/tap/<tool>` instructions for the seven toolkit tools only. Third-party prerequisite instructions stay.

### README.md — `## Install` (lines ~16–26): NO CHANGE (already conformant)

Already carries `curl -fsSL https://shll.ai/install | sh -s -- fab-kit` + the full-toolkit variant. This is the model the install.md replacement mirrors.

### README.md — Prerequisites table row (line ~111): REPLACE the parenthetical

Current cell: ``Recommended companions (`brew install sahil87/tap/wt sahil87/tap/idea`) — worktree isolation and the idea backlog; see [Companion tools](#companion-tools)``

Replace the parenthetical with the shll form, e.g.: ``Recommended companions (`shll install wt idea` — see [shll.ai](https://shll.ai)) — worktree isolation and the idea backlog; see [Companion tools](#companion-tools)``

### README.md — `## Companion tools` (line ~643–649): REPLACE

Current: "installed from their own formulas" + code block `brew install sahil87/tap/wt sahil87/tap/idea`.

Replace the code block with `shll install wt idea` and adjust the lead sentence to point install steps at https://shll.ai, keeping the independent-release-cadence fact and all graceful-degradation prose and the role table below it.

### README.md — third-party utilities (lines ~95, ~122): KEEP

Same third-party carve-out as install.md.

### Out of scope (explicit)

- **Go source hint strings** (`batch_new.go:66`, `batch_switch.go:74`, `update.go:25`, `sync.go:150`, `doctor.go:107`, `prereqs.go`): binary-half Policy A territory — hints are mandated. No code changes; this is a docs-only change.
- **The tap README** (`sahil87/homebrew-tap`): different repo.
- **`docs/specs/` and `docs/memory/`**: not published to the site (readme-extraction standard: only README.md and docs/site/** are pulled); mentions there are internal/historical documentation of the formula reality, which remains true. Exception: the hydrate-stage memory update below.

### Conformance cross-check (readme-extraction standard)

The edit must not break the README-slice pull: replacement links to shll.ai are absolute `https://` (external links absolute-by-author — rule 5), no structural changes to the H1/blockquote/badges head or footer headings, no new relative links. Verified at intake: the replacement content is conformant.

## Affected Memory

- `distribution/distribution.md`: (modify) Record Policy B conformance under § shll.ai Public Docs Site / Toolkit Standards Conformance — install documentation (README + docs/site) is centralized on shll.ai per the install-composition standard; per-formula `brew install` lines were removed from README.md and docs/site/install.md (Policy A's formula/hint half shipped separately as PR #511). The file's factual claims that wt/idea *are installable* via their own formulas stay — the mechanism remains supported; only documenting it per-repo is out.

## Impact

- **Files**: `README.md` (2 edits), `docs/site/install.md` (2 sections), `docs/memory/distribution/distribution.md` (hydrate). No Go, no tests, no skills, no SPEC mirrors (no skill files touched).
- **change_type**: docs. No migration (no user-data restructuring).
- **Public surface**: both files are shll.ai pull surfaces (README slice + docs/site pages), so the fix lands on the public site on the next pull — that is the point.
- **Risk**: minimal; purely textual. The one accuracy-sensitive spot is keeping the quoted wt hint in install.md aligned with the binary (verified matching today).

## Open Questions

None — the task specifies the policy, the replacement pointer, and the keep/replace distinction; the audit resolved all locations.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Scope is README.md + docs/site/ only; Go hint strings and the tap README untouched | Task says "docs-only"; hints are Policy A-mandated binary half; tap README is another repo | S:95 R:90 A:95 D:95 |
| 2 | Certain | Keep install.md:39 error-hint example verbatim | Task explicitly instructs KEEP; verified it matches `batch_new.go:66` exactly | S:95 R:90 A:95 D:95 |
| 3 | Certain | Third-party utility instructions (`yq jq gh direnv`, `go just`) stay | Policy B's MUST NOT covers per-formula toolkit (`sahil87/tap/<tool>`) instructions; third-party prerequisites are outside it | S:85 R:90 A:90 D:90 |
| 4 | Confident | install.md "Install the CLI" replacement = curl bootstrap pair (subset `sh -s -- fab-kit` + full toolkit), keeping the two-CLI role table | Mirrors README's already-conformant Install section — the in-repo precedent; task names both bootstrap forms | S:80 R:85 A:85 D:80 |
| 5 | Confident | Companion install pointers become `shll install wt idea` + absolute https://shll.ai link | Task: "subset installs remain supported via shll install \<tool\>"; standard names `shll install` as the composition point; absolute link per readme-extraction rule 5 | S:75 R:85 A:80 D:75 |
| 6 | Confident | Hydrate target is `distribution/distribution.md` § Toolkit Standards Conformance; formula-reality claims in that file stay | The section already records per-standard conformance (audited at shll v0.0.23); mechanism-still-supported line comes from the standard itself | S:65 R:85 A:80 D:75 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
