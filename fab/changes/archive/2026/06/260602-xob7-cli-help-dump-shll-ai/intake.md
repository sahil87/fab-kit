# Intake: Build-Time CLI Help-Dump → shll.ai

**Change**: 260602-xob7-cli-help-dump-shll-ai
**Created**: 2026-06-03
**Status**: Draft

## Origin

This change was initiated from backlog item `[xob7]` via `/fab-new`, one-shot (no prior
`/fab-discuss` conversation). The raw input below is the frozen brief; it carries an
explicit, pre-decided contract, producer approach, version source, and push mechanism.

> Add a build-time 'help-dump' step that emits fab's CLI help tree as `help/fab-kit.json` and
> PRs it into `sahil87/shll.ai` (the shll.ai landing site renders it as an expandable 'Command
> reference' on the fab-kit tool page). **CONTRACT (frozen — copy the reference sample at
> sahil87/shll.ai path `help/wt.json`)**: JSON shape is
> `{tool, version, captured_at (ISO-8601 UTC), schema_version: 1, root: Node}` where
> `Node = {name, path (full invocation e.g. 'fab new'), short (one-line desc), usage, text (the
> RAW -h output byte-for-byte, newlines preserved), commands: Node[] (recursive; empty array =
> leaf)}`. NOTE the file is named `help/fab-kit.json` (the repo/site slug) but the user-facing
> binary is 'fab' — set the top-level `tool` field to 'fab'. The fab-kit repo ships TWO binaries
> from nested modules (`src/go/fab/cmd/fab` built with `-ldflags '-X main.version=...'` from
> `src/kit/VERSION`, and `src/go/fab-kit/cmd/fab-kit`) — dump the user-facing 'fab' CLI
> (`src/go/fab`, module dir `src/go/fab`) which has the rich subcommand tree; do NOT dump the
> fab-kit shim unless decided otherwise. **PRODUCER (Cobra/Go)**: walk the cobra command tree
> programmatically (`rootCmd.Commands()` recursively), NOT regex-parsing -h text; per node
> capture `cmd.Name` / `cmd.CommandPath()` / `cmd.Short` / `cmd.UseLine()` and `cmd.UsageString()`
> (or `Long+UsageString`) as 'text'. FILTER OUT cobra's auto-generated 'completion' and 'help'
> subcommands and any `cmd.Hidden==true`. **VERSION**: read from the built binary (`main.version`
> via the existing `src/kit/VERSION` ldflags) — do NOT hardcode. **PUSH**: in CI after build, run
> the dump, write `help/fab-kit.json`, validate it parses, then open a PR into `sahil87/shll.ai`
> using the existing repo secret `SHLLAI_TOKEN` (contents + pull-request write) with auto-merge
> enabled (PR, not direct push to main, to avoid the multi-repo push race). This is fab-kit's
> slice of a 7-tool rollout; the shll.ai site-side consumer (Astro loader + reference UI) is
> tracked separately in the shll.ai repo.

**Verification performed during intake** (grounds the SRAD grades below):

- Fetched the live reference at `sahil87/shll.ai:help/wt.json` and confirmed the exact shape:
  `tool="wt"`, `version="1.4.2"`, `captured_at="2026-06-02T00:00:00Z"`, `schema_version=1`,
  and `root` Node with `name`/`path`/`short`/`usage`/`text`/`commands[]`. In the sample,
  `usage` = cobra's one-line `UseLine` form (e.g. `wt create [branch] [flags]`), `text` = the
  full `UsageString` (the raw `-h` body with `\n`-escaped newlines), and `completion`/`help` are
  absent — confirming the filter requirement.
- Confirmed the rich CLI root lives at `src/go/fab/cmd/fab/main.go` (`Use: "fab"`,
  `var version = "dev"`, 14 subcommands incl. `change`, `status`, `score`, `operator`, `batch`).
- Confirmed the build path: `justfile` `build-target` compiles `src/go/fab ./cmd/fab` into
  `fab-go-{os}-{arch}` with `fab_ldflags = -X main.version=$(cat src/kit/VERSION)`. This is the
  ONLY binary carrying the rich tree + a real `main.version`. (The shim also produces a `fab`
  binary from `src/go/fab-kit/cmd/fab`, but that is the shim's thin `fab`, not the rich tree.)
- Confirmed `release.yml` is the CI build pipeline (tag-push / `workflow_dispatch`), driving
  `just dist-kit → build-all → package-kit → package-brew` then token-gated upload steps.
- Confirmed `SHLLAI_TOKEN`, `shll.ai`, and any help-dump step are **not yet present** anywhere in
  the fab-kit repo — this is net-new infrastructure.
- Confirmed `sahil87/wt`'s own `release.yml` does **not** yet contain a help-dump → shll.ai step,
  so the `help/wt.json` sample was produced out-of-band. **There is no existing producer/CI
  pattern in any sibling repo to copy** — fab-kit is the first concrete implementation of the
  7-tool rollout's producer half.

## Why

1. **Problem it solves.** The shll.ai landing site wants to render a live, accurate "Command
   reference" for each of 7 tools, as an expandable tree on the tool's page. A reference that is
   hand-maintained drifts the moment a flag or subcommand changes. fab's CLI surface is large
   (14 top-level commands, many with subcommands and flags) and changes frequently, so a manual
   reference would be wrong within a release or two.

2. **Consequence of not doing it.** The site either ships no reference for fab-kit (a hole in the
   7-tool grid) or a stale, hand-authored one that misleads users about flags/commands that no
   longer exist or have changed. Either undermines the site's value proposition of being the
   authoritative command reference.

3. **Why this approach.** Generating the tree *programmatically from the live cobra command
   objects* at release time guarantees the reference is byte-accurate to the exact binary being
   shipped — version included (read from `main.version`, not hardcoded). Walking
   `rootCmd.Commands()` (rather than regex-parsing `-h` text) is robust to help-text formatting
   changes and yields structured `name`/`path`/`short`/`usage` fields the site needs for its
   tree UI, while still capturing the raw `-h` body verbatim in `text` for fidelity. Producing it
   in CI and delivering via an **auto-merging PR** (not a direct push) is the rollout-wide
   convention chosen to avoid a write race when 7 tools all push into the single shll.ai repo
   around the same release window.

## What Changes

### 1. New help-dump producer (Go, in the rich `fab` CLI)

Add a producer that walks the live cobra command tree of the `src/go/fab` root command and emits
the frozen JSON. **Chosen location: a hidden cobra subcommand `fab help-dump` on the rich CLI**
(see Assumptions #6 for the rationale and the rejected standalone-binary alternative).

Rationale for the hidden-subcommand placement, concretely:

- It has direct, in-process access to the *same* `root` command object assembled in
  `src/go/fab/cmd/fab/main.go` — no risk of dumping a different tree than the one shipped.
- It inherits `var version = "dev"` / `main.version` for free — the existing
  `fab_ldflags = -X main.version=$(cat src/kit/VERSION)` already injects the real version when
  the `fab-go` binary is built by `just build-target`. No second ldflags wiring needed.
- Marking the command `Hidden: true` keeps it out of its own dumped tree (the producer filters
  `cmd.Hidden==true`) and out of `fab --help`, so it is a build/CI-only affordance.

New file (illustrative): `src/go/fab/cmd/fab/helpdump.go`

```go
// helpDumpCmd is a hidden, CI-only command that serializes the live cobra
// command tree of the rich `fab` CLI to the frozen shll.ai contract JSON on stdout.
func helpDumpCmd() *cobra.Command {
    return &cobra.Command{
        Use:    "help-dump",
        Short:  "Emit the CLI help tree as shll.ai contract JSON (CI/build-time)",
        Hidden: true,
        Args:   cobra.NoArgs,
        RunE: func(cmd *cobra.Command, args []string) error {
            doc := dumpDoc(cmd.Root(), version) // cmd.Root() == the assembled `fab` root
            enc := json.NewEncoder(cmd.OutOrStdout())
            enc.SetIndent("", "  ")
            enc.SetEscapeHTML(false) // preserve raw -h bytes; do not escape <, >, &
            return enc.Encode(doc)
        },
    }
}
```

Wired into `main.go`'s `root.AddCommand(...)` list alongside the existing 14 commands.

#### Contract structs (mirror the frozen shape exactly)

```go
type HelpDoc struct {
    Tool          string `json:"tool"`            // literal "fab" (NOT "fab-kit")
    Version       string `json:"version"`         // from main.version (ldflags)
    CapturedAt    string `json:"captured_at"`     // RFC3339 UTC, e.g. 2026-06-03T12:34:56Z
    SchemaVersion int    `json:"schema_version"`  // literal 1
    Root          Node   `json:"root"`
}

type Node struct {
    Name     string `json:"name"`     // cmd.Name()
    Path     string `json:"path"`     // cmd.CommandPath() — e.g. "fab change new"
    Short    string `json:"short"`    // cmd.Short
    Usage    string `json:"usage"`    // cmd.UseLine()
    Text     string `json:"text"`     // cmd.UsageString() — raw -h body, byte-for-byte
    Commands []Node `json:"commands"` // recursive; [] for a leaf (never null)
}
```

#### Tree walk + filter rules (programmatic, NOT regex)

- Start at `cmd.Root()` (the `fab` root). Recurse via `cmd.Commands()`.
- **Filter out** any child where `cmd.Name() == "completion"`, `cmd.Name() == "help"`, or
  `cmd.Hidden == true` (this also drops `help-dump` itself from the output tree). Apply the
  filter at every level.
- Per node capture: `Name=cmd.Name()`, `Path=cmd.CommandPath()`, `Short=cmd.Short`,
  `Usage=cmd.UseLine()`, `Text=cmd.UsageString()`. `commands` is always a non-nil slice
  (initialize to `[]Node{}` so leaves serialize as `[]`, matching the reference, not `null`).
- Sort children deterministically by `Name` so successive dumps are byte-stable (avoids noisy
  no-op PRs). The reference `wt.json` lists children in `wt --help` order; cobra's
  `.Commands()` is already alphabetical by default, so an explicit sort both matches and locks it.
- `captured_at`: emitted as RFC3339 in UTC. In CI this is the build wall-clock; see Assumptions
  #8 for the stability caveat (changes every run → PR diff on every release even when the tree is
  identical).

#### `tool` vs filename

The emitted `tool` field is the literal string `"fab"` (the user-facing binary). The output
*file* is `help/fab-kit.json` (the repo/site slug). These intentionally differ — the producer
hardcodes `tool: "fab"`; the filename is decided by the CI write step, not the producer.

### 2. Unit test for the producer (Go)

`src/go/fab/cmd/fab/helpdump_test.go`:

- Build a small synthetic cobra tree (root + 1 subcommand + 1 hidden + a fake `completion`/`help`)
  and assert the walker drops `completion`/`help`/hidden, preserves `path`/`usage`/`text`, and
  emits `[]` (not `null`) for leaves.
- Assert top-level `tool == "fab"`, `schema_version == 1`, and that `version` reflects the passed
  value (not hardcoded).
- Optionally golden-test the encoder settings (`SetEscapeHTML(false)`, 2-space indent) so the byte
  output stays diff-stable against the reference style.

Follows the existing `fabhelp_test.go` style in the same package.

### 3. CI step: build → dump → validate → PR into shll.ai

Extend `.github/workflows/release.yml` (the existing build pipeline) with a new step **after the
binaries are built** (`just build-all` produces `dist/bin/fab-go-linux-amd64`). The step:

1. **Run the dump** using the freshly built linux/amd64 `fab-go` binary so `version` is the real
   ldflags-injected value:
   ```bash
   ./dist/bin/fab-go-linux-amd64 help-dump > help/fab-kit.json
   ```
   (Path per `justfile` `_build-binary` naming: `dist/bin/{name}-{os}-{arch}` → `fab-go-linux-amd64`.)
2. **Validate it parses** as JSON and has the required top-level keys:
   ```bash
   jq -e '.tool=="fab" and .schema_version==1 and (.version|length>0) and (.root|type=="object")' help/fab-kit.json
   ```
3. **Open an auto-merging PR into `sahil87/shll.ai`** writing `help/fab-kit.json`, authenticated
   with the existing repo secret `SHLLAI_TOKEN` (contents + pull-request write). Clone the target
   repo with the token, write the file on a fresh branch, push, `gh pr create`, then
   `gh pr merge --auto --squash`. **PR, not direct push** — to avoid the multi-repo push race when
   the other 6 tools push concurrently. Illustrative:
   ```bash
   branch="help-dump/fab-kit-${{ steps.version.outputs.version }}"
   git clone "https://x-access-token:${SHLLAI_TOKEN}@github.com/sahil87/shll.ai.git" /tmp/shllai
   mkdir -p /tmp/shllai/help
   cp help/fab-kit.json /tmp/shllai/help/fab-kit.json
   cd /tmp/shllai
   git config user.name  "github-actions[bot]"
   git config user.email "github-actions[bot]@users.noreply.github.com"
   git checkout -b "$branch"
   git add help/fab-kit.json
   git diff --cached --quiet && { echo "no change; skipping PR"; exit 0; }   # idempotent: skip when identical
   git commit -m "fab-kit: refresh command reference (v${{ steps.version.outputs.version }})"
   git push -u origin "$branch"
   GH_TOKEN="$SHLLAI_TOKEN" gh pr create --repo sahil87/shll.ai \
     --base main --head "$branch" \
     --title "fab-kit: command reference v${{ steps.version.outputs.version }}" \
     --body "Automated CLI help-dump from fab-kit release v${{ steps.version.outputs.version }}."
   GH_TOKEN="$SHLLAI_TOKEN" gh pr merge --repo sahil87/shll.ai --auto --squash "$branch"
   ```
   The producer runs in fab-kit's repo working dir (writing `help/fab-kit.json` there transiently
   for validation), then the file is copied into the cloned shll.ai checkout for the PR — fab-kit
   itself does not retain `help/` (it is a shll.ai artifact). See Assumptions #11.

4. **Placement & gating.** Step lives in the `release` job after `Build all targets`. It must NOT
   fail the release if the shll.ai PR step errors (the site reference is a downstream nicety; a
   broken token or a transient shll.ai outage should not block fab-kit's own release/tap update).
   Wrap the PR portion so its failure is non-fatal (e.g. `continue-on-error: true` on the PR step,
   or `|| echo "::warning::shll.ai PR failed"`), while the *dump + validate* portion IS fatal
   (a malformed dump is a real fab-kit bug). See Assumptions #10.

### 4. Out of scope (explicit)

- The shll.ai **site-side consumer** (Astro loader + the expandable "Command reference" UI) —
  tracked separately in the shll.ai repo. This change only produces and delivers the JSON.
- Dumping the **fab-kit shim** (`src/go/fab-kit`) — explicitly excluded; only the rich `fab` CLI.
- Enabling auto-merge **at the shll.ai repo settings level** (branch protection / "Allow
  auto-merge") — that is a shll.ai-repo configuration prerequisite, not a fab-kit code change.
  Noted as a risk in Impact.

## Affected Memory

- `fab-workflow/distribution`: (modify) The release/build pipeline gains a new CI artifact
  (`help/fab-kit.json`) and a cross-repo delivery step into shll.ai. The distribution memory file
  documents how the release pipeline assembles and ships artifacts; the new help-dump producer +
  PR step is a distribution-surface addition worth recording.
- `fab-workflow/kit-architecture`: (modify) A new hidden CI-only subcommand (`fab help-dump`) is
  added to the rich `fab` CLI's command set. If kit-architecture enumerates the binary's command
  surface or the two-binary split, note the addition (hidden, build-time only).

(If on hydrate these files turn out not to cover the release pipeline at this granularity, a small
new memory file under `fab-workflow/` documenting the help-dump → shll.ai delivery may be created
instead.)

## Impact

- **Code (fab-kit):**
  - `src/go/fab/cmd/fab/helpdump.go` (new) — producer command + structs + tree walk.
  - `src/go/fab/cmd/fab/helpdump_test.go` (new) — producer unit test.
  - `src/go/fab/cmd/fab/main.go` (modify) — register `helpDumpCmd()` in `root.AddCommand(...)`.
  - `.github/workflows/release.yml` (modify) — new build-time dump + validate + shll.ai PR step.
- **Dependencies:** `encoding/json` (stdlib) for the producer; `jq` and `gh` in CI (both available
  on `ubuntu-latest`). No new Go module dependencies.
- **Secrets / external systems:**
  - `SHLLAI_TOKEN` repo secret — must exist in fab-kit's repo settings with **contents:write +
    pull-requests:write** scope on `sahil87/shll.ai`. The brief states it already exists; this
    intake does not create it. If absent at release time, the PR step warns and the release
    otherwise succeeds (per non-fatal gating).
  - `sahil87/shll.ai` — target repo must have **"Allow auto-merge"** enabled and a merge path
    (e.g. required checks pass or none required) for `gh pr merge --auto` to actually land the PR.
    Outside fab-kit's control; a risk if not configured.
- **Cross-repo race:** The PR-with-auto-merge approach is specifically chosen over a direct push to
  shll.ai `main` to avoid non-fast-forward rejections when the 7 tools deliver concurrently.
- **Release behavior:** The dump runs only on the release path (tag push / `workflow_dispatch`),
  so day-to-day CI is unaffected. A successful release now also opens (and auto-merges) a shll.ai
  PR.
- **Edge case — empty/no-op diff:** If the command tree and version are unchanged from the last
  delivered dump (only `captured_at` differs), the PR would otherwise be a pure-timestamp churn.
  The CI step skips PR creation when the file content is byte-identical (`git diff --cached
  --quiet`); whether to also exclude `captured_at` from that comparison is the open question below.

## Open Questions

- **`captured_at` vs. idempotent diffs.** `captured_at` changes on every release run, so a naive
  byte-comparison will never be "identical" and every release opens a shll.ai PR even when the
  command tree is unchanged. Is per-release PR churn acceptable (simpler, always-fresh timestamp),
  or should the idempotency check ignore `captured_at` (compare only `tool`/`version`/`root`) and
  skip the PR when only the timestamp moved? Defaulting to per-release PR (Confident #8) since
  releases are infrequent and a fresh `captured_at` per release is informative; revisit via
  `/fab-clarify` if PR noise becomes a concern.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Dump the rich `fab` CLI from `src/go/fab` (root in `cmd/fab/main.go`), not the `fab-kit` shim. | Explicitly dictated in the brief and confirmed: only `src/go/fab` has the 14-command rich tree; the shim has 5 lifecycle commands. | S:98 R:80 A:95 D:95 |
| 2 | Certain | Emit the frozen JSON contract verbatim: top-level `{tool,version,captured_at,schema_version:1,root}`; `Node{name,path,short,usage,text,commands[]}`. | Contract is declared frozen and was byte-verified against the live `sahil87/shll.ai:help/wt.json` sample during intake. | S:98 R:70 A:95 D:98 |
| 3 | Certain | `tool` field = literal `"fab"`; output filename = `help/fab-kit.json`. They intentionally differ. | Explicitly spelled out in the brief ("set the top-level tool field to 'fab'", file named for the repo/site slug). | S:99 R:90 A:95 D:99 |
| 4 | Certain | Build the tree by walking `rootCmd.Commands()` recursively and reading `cmd.Name/CommandPath/Short/UseLine/UsageString`; do NOT regex-parse `-h`. | Dictated by the brief; the reference sample's `usage`=UseLine and `text`=UsageString mapping was confirmed against `wt.json`. | S:97 R:75 A:92 D:95 |
| 5 | Certain | Filter out `completion`, `help`, and any `cmd.Hidden==true` at every tree level. | Dictated by the brief; confirmed `completion`/`help` are absent from the reference `wt.json`. The hidden filter also self-excludes `help-dump`. | S:97 R:80 A:95 D:95 |
| 6 | Confident | Implement the producer as a **hidden cobra subcommand `fab help-dump`** on the rich CLI, rather than a standalone `cmd/helpdump` binary. | Not specified in the brief, but the hidden-subcommand approach has one obvious advantage: it reuses the exact assembled `root` object AND inherits `main.version` from the existing `fab_ldflags` for free. A standalone binary would need its own root-construction + a second ldflags wiring. Easily reversed (delete the cmd, add a binary). Marked `<!-- assumed -->` in What Changes §1. | S:55 R:65 A:80 D:78 |
| 7 | Certain | Read `version` from the built binary's `main.version`, populated by the existing `fab_ldflags = -X main.version=$(cat src/kit/VERSION)` when `just build-target` builds `fab-go`. Run the dump using `dist/bin/fab-go-linux-amd64` in CI. | Dictated ("read from the built binary, do NOT hardcode") AND deterministically answered by the codebase: only the `fab-go` target carries both the rich tree and this ldflags, so it is the single correct binary to invoke — no disambiguation remains once verified. | S:88 R:75 A:92 D:90 |
| 8 | Confident | Emit a fresh `captured_at` (UTC RFC3339, CI wall-clock) every release and open a PR per release; idempotency check skips only on byte-identical content. | The timestamp format IS signaled (ISO-8601 UTC); only the per-release-PR-churn policy is open, and "always-fresh timestamp" is the clear front-runner (releases are infrequent, a fresh stamp per release is informative). The residual open part is captured as the Open Question, not as an Unresolved. Reversible (CI-step tweak). Marked `<!-- assumed -->`. | S:62 R:70 A:62 D:72 |
| 9 | Confident | Deliver via clone-token + branch + `gh pr create` + `gh pr merge --auto --squash` into `sahil87/shll.ai`, using `SHLLAI_TOKEN`. | Dictated (PR with auto-merge, not direct push, via `SHLLAI_TOKEN`). The clone+gh pattern mirrors the existing Homebrew-tap delivery in `release.yml`; `gh`/`jq` are present on `ubuntu-latest`. | S:80 R:65 A:85 D:80 |
| 10 | Confident | The shll.ai PR step is **non-fatal** to the fab-kit release; the dump+validate step IS fatal. | Not specified, but a downstream site-reference delivery failure (bad token, shll.ai outage) should not block fab-kit's own release/tap. A malformed dump is a genuine fab-kit bug and must fail. One obvious default given the existing release-job structure. Marked `<!-- assumed -->`. | S:50 R:75 A:80 D:75 |
| 11 | Confident | Place the new step in `release.yml`'s `release` job after `Build all targets`; write `help/fab-kit.json` transiently for validation, copy into the cloned shll.ai checkout — fab-kit does not retain `help/`. | Brief says "in CI after build". `release.yml` is the only build pipeline; the binaries exist only after `build-all`. `help/` belongs to the shll.ai artifact, not fab-kit's tree. Reversible. | S:78 R:80 A:85 D:80 |
| 12 | Certain | Sort children by `Name` for byte-stable dumps. | Mechanically determined, not a tradeoff: cobra's `.Commands()` is already alphabetical and the reference `wt.json` is alphabetical, so an explicit sort both matches the reference and locks determinism. Trivially reversible, codebase fully answers it, one obvious default. | S:70 R:90 A:90 D:88 |
| 13 | Confident | No new memory file at intake time; tentatively update `fab-workflow/distribution` and `fab-workflow/kit-architecture` at hydrate. | Memory updates are a hydrate-stage concern; the affected domains are identifiable now but exact file granularity is decided at hydrate against the then-current memory tree. | S:60 R:90 A:80 D:75 |

13 assumptions (7 certain, 6 confident, 0 tentative, 0 unresolved).
