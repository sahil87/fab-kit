# Plan: Build-Time CLI Help-Dump → shll.ai

**Change**: 260602-xob7-cli-help-dump-shll-ai
**Status**: In Progress
**Intake**: `intake.md`

## Requirements

<!-- Requirements derived from intake.md. Contract is frozen; SRAD assumptions
     carried/refined in ## Assumptions. -->

### Producer: `fab help-dump` command

#### R1: Hidden CI-only subcommand on the rich `fab` CLI
The rich `fab` CLI (`src/go/fab/cmd/fab`) MUST expose a hidden, argument-less cobra subcommand `help-dump` that serializes the live command tree of the assembled root command to stdout as JSON. The command MUST be `Hidden: true` and accept `cobra.NoArgs`, and MUST be registered in `main.go`'s `root.AddCommand(...)` list.

- **GIVEN** the built `fab` binary
- **WHEN** `fab help-dump` is run with no arguments
- **THEN** it writes the contract JSON for the full command tree to stdout and exits 0
- **AND** `help-dump` does not appear in `fab --help` (it is hidden)

#### R2: Frozen top-level JSON contract
The emitted document MUST be a JSON object with keys in this exact order: `tool`, `version`, `captured_at`, `schema_version`, `root`. `tool` MUST be the literal string `"fab"`. `schema_version` MUST be the literal integer `1`. `version` MUST be the value of the binary's `main.version` variable (NOT hardcoded). `captured_at` MUST be the current time formatted as RFC3339 in UTC. `root` MUST be a `Node`.

- **GIVEN** a serialized help-dump document
- **WHEN** the JSON is parsed
- **THEN** `tool == "fab"`, `schema_version == 1`, `version` equals the injected `main.version`, and `captured_at` matches RFC3339 UTC
- **AND** the top-level key order is `tool, version, captured_at, schema_version, root`

#### R3: Node shape and recursive structure
Each `Node` MUST have keys `name`, `path`, `short`, `usage`, `text`, `commands` where: `name = cmd.Name()`, `path = cmd.CommandPath()`, `short = cmd.Short`, `usage = cmd.UseLine()`, `text = cmd.UsageString()`, and `commands` is a recursive `[]Node`. The `commands` slice MUST serialize as `[]` (never `null`) for leaf commands — i.e. the slice is always initialized non-nil.

- **GIVEN** a leaf command with no surviving children
- **WHEN** its node is serialized
- **THEN** its `commands` field is `[]`, not `null`
- **AND** a non-leaf command's `commands` contains one node per surviving child

#### R4: Tree-walk filters applied at every level
The walk MUST recurse via `cmd.Commands()` and, at EVERY level, MUST skip any child where `Name() == "completion"`, `Name() == "help"`, or `Hidden == true`. The `Hidden` filter self-excludes `help-dump` from its own output. Surviving children MUST be sorted by `Name()` for byte-stable output.

- **GIVEN** a command tree containing `completion`, `help`, and a hidden command among its children
- **WHEN** the tree is walked
- **THEN** none of `completion`, `help`, or the hidden command appear in the output `commands`
- **AND** surviving children are emitted in ascending `Name()` order at every level

#### R5: Byte-faithful JSON encoding
The encoder MUST use 2-space indentation (`SetIndent("", "  ")`) and MUST disable HTML escaping (`SetEscapeHTML(false)`) so that raw `-h` bytes (`<`, `>`, `&`) in `text`/`usage` are preserved byte-for-byte.

- **GIVEN** a command whose help text contains `<`, `>`, or `&`
- **WHEN** the document is encoded
- **THEN** those characters appear literally (not as `<` etc.) in the output
- **AND** the document is indented with 2 spaces per level

### Testing

#### R6: Producer unit test coverage
A test file `src/go/fab/cmd/fab/helpdump_test.go` (following the style of `fabhelp_test.go`) MUST verify: the walk drops `completion`/`help`/hidden children; leaves emit `[]` not `null`; `path`/`usage`/`text` are captured; top-level `tool == "fab"` and `schema_version == 1`; and `version` reflects the value passed in (not hardcoded).

- **GIVEN** a synthetic cobra tree (root + visible child + hidden child + fake `completion` and `help`)
- **WHEN** the dump function is invoked with an arbitrary version string
- **THEN** the assertions above hold and the test passes under `go test ./cmd/fab/`

### CI Delivery

#### R7: Build-time dump + fatal validation
`.github/workflows/release.yml` MUST, after the `Build all targets` step, run `./dist/bin/fab-go-linux-amd64 help-dump > help/fab-kit.json` and validate it with `jq -e '.tool=="fab" and .schema_version==1 and (.version|length>0) and (.root|type=="object")'`. The dump + validate portion MUST be fatal to the release (a malformed dump is a real bug).

- **GIVEN** the release job has built all targets
- **WHEN** the help-dump step runs
- **THEN** `help/fab-kit.json` is produced from the linux/amd64 binary and validated with `jq -e`
- **AND** if validation fails the release job fails

#### R8: Non-fatal auto-merging PR into shll.ai
The workflow MUST open an auto-merging PR into `sahil87/shll.ai` writing `help/fab-kit.json`, authenticated with the `SHLLAI_TOKEN` secret, via clone-with-token + branch + `gh pr create` + `gh pr merge --auto --squash` (mirroring the existing Update Homebrew tap pattern). This PR portion MUST be non-fatal to the release. The step MUST be idempotent: when the staged file is byte-identical to the target (`git diff --cached --quiet`), it MUST skip PR creation. Branch and PR title MUST embed `${{ steps.version.outputs.version }}`.

- **GIVEN** a successful, validated dump
- **WHEN** the delivery step runs and the content differs from shll.ai's current `help/fab-kit.json`
- **THEN** a branch `help-dump/fab-kit-<version>` is pushed and an auto-merge-squash PR is opened
- **AND** when content is byte-identical, no PR is created (idempotent)
- **AND** any failure of the PR portion does not fail the fab-kit release

#### R9: fab-kit does not retain the artifact
`help/fab-kit.json` is a transient CI artifact owned by shll.ai. The fab-kit repo tree MUST NOT commit `help/fab-kit.json`; `help/` SHOULD be added to `.gitignore`.

- **GIVEN** the help-dump step writes `help/fab-kit.json` in the workspace
- **WHEN** the repo state is inspected
- **THEN** `help/` is gitignored and no `help/fab-kit.json` is committed to fab-kit

### Documentation

#### R10: CLI reference entry for `fab help-dump`
`src/kit/skills/_cli-fab.md` MUST gain an entry documenting `fab help-dump`, noting it is a hidden, CI/build-time-only command and describing its contract output. Format MUST match the existing single-command sections (e.g. `## fab kit-path`).

- **GIVEN** the constitution rule that CLI changes update `_cli-fab.md`
- **WHEN** a reviewer reads `_cli-fab.md`
- **THEN** a `## fab help-dump` section documents the command, its hidden/CI-only nature, and its JSON contract

### Non-Goals

- The shll.ai site-side consumer (Astro loader + "Command reference" UI) — tracked in the shll.ai repo.
- Dumping the `fab-kit` shim (`src/go/fab-kit`) — only the rich `fab` CLI is dumped.
- Enabling "Allow auto-merge" / branch protection at the shll.ai repo level — a shll.ai config prerequisite, not a fab-kit code change.
- Excluding `captured_at` from the idempotency comparison — per-release timestamp churn is accepted (intake Open Question / assumption #8).

### Design Decisions

1. **Hidden cobra subcommand over a standalone binary**: reuses the exact assembled `root` object and inherits `main.version` from the existing `fab_ldflags`. — *Why*: zero risk of dumping a divergent tree; no second ldflags wiring. — *Rejected*: a separate `cmd/helpdump` binary (needs its own root construction + ldflags).
2. **Walk `cmd.Commands()` recursively, read structured cobra fields**: not regex-parsing `-h`. — *Why*: robust to help-text formatting changes; yields the structured fields the site needs. — *Rejected*: regex-scraping `-h` output.
3. **`SetEscapeHTML(false)` + 2-space indent**: matches the frozen `wt.json` reference byte style and preserves raw `-h` bytes. — *Why*: byte-stable, diff-friendly output. — *Rejected*: default encoder (escapes `<`/`>`/`&`).

## Tasks

### Phase 1: Producer

- [x] T001 Create `src/go/fab/cmd/fab/helpdump.go` with `HelpDoc`/`Node` structs (frozen field order/tags), `dumpDoc(root *cobra.Command, version string) HelpDoc`, recursive `buildNode` with completion/help/hidden filters + `Name()` sort + non-nil `commands` slice, and `helpDumpCmd()` (`Hidden: true`, `cobra.NoArgs`, `RunE` encoding with `SetIndent("", "  ")` + `SetEscapeHTML(false)`) <!-- R1 R2 R3 R4 R5 -->
- [x] T002 Register `helpDumpCmd()` in `src/go/fab/cmd/fab/main.go` `root.AddCommand(...)` list <!-- R1 -->

### Phase 2: Tests

- [x] T003 Create `src/go/fab/cmd/fab/helpdump_test.go` (mirroring `fabhelp_test.go` style): synthetic tree asserting filters drop completion/help/hidden, leaves emit `[]` not null, path/usage/text captured, `tool=="fab"`, `schema_version==1`, `version` reflects passed-in value, children sorted, and HTML not escaped <!-- R6 -->

### Phase 3: CI Delivery

- [x] T004 Edit `.github/workflows/release.yml`: add a `Help-dump → shll.ai` step after `Build all targets` running the dump + fatal `jq -e` validation against `./dist/bin/fab-go-linux-amd64` writing `help/fab-kit.json` <!-- R7 -->
- [x] T005 In the same step, add the non-fatal, idempotent auto-merging PR into `sahil87/shll.ai` (clone-with-token + branch + `gh pr create` + `gh pr merge --auto --squash`, `SHLLAI_TOKEN` via `env:`, skip on `git diff --cached --quiet`) <!-- R8 -->
- [x] T006 Add `help/` to `.gitignore` so the transient artifact is never committed to fab-kit <!-- R9 -->

### Phase 4: Documentation

- [x] T007 Add a `## fab help-dump` section to `src/kit/skills/_cli-fab.md` documenting the hidden/CI-only command and its JSON contract <!-- R10 -->

## Execution Order

- T001 blocks T002 and T003 (structs/funcs must exist first)
- T004 blocks T005 (same workflow step)
- T006, T007 independent

## Acceptance

### Functional Completeness

- [x] A-001 R1: `fab help-dump` exists as a hidden, `NoArgs` cobra subcommand registered in `main.go` and absent from `fab --help`
- [x] A-002 R2: Output JSON has `tool=="fab"`, `schema_version==1`, `version` from `main.version`, RFC3339-UTC `captured_at`, top-level key order `tool,version,captured_at,schema_version,root`
- [x] A-003 R3: Each node has `name/path/short/usage/text/commands`; leaves emit `commands: []` not `null`
- [x] A-004 R4: `completion`, `help`, and hidden commands are filtered at every level; surviving children sorted by `Name()`
- [x] A-005 R5: Encoder uses 2-space indent and `SetEscapeHTML(false)`; `<`/`>`/`&` preserved literally
- [x] A-006 R6: `helpdump_test.go` exists and passes, asserting filters/leaf-`[]`/captured fields/`tool`/`schema_version`/`version`-passthrough
- [x] A-007 R7: `release.yml` runs the dump from `dist/bin/fab-go-linux-amd64` after `Build all targets` with a fatal `jq -e` validation
- [x] A-008 R8: The shll.ai PR step is non-fatal, idempotent (skips on byte-identical), token-authed, and embeds the version in branch/title
- [x] A-009 R9: `help/` is gitignored; no `help/fab-kit.json` committed to fab-kit
- [x] A-010 R10: `_cli-fab.md` documents `fab help-dump` as hidden/CI-only with its contract

### Scenario Coverage

- [x] A-011 R3 R4: A synthetic-tree test exercises the leaf-`[]`, filter, and sort scenarios

### Code Quality

- [x] A-012 Pattern consistency: New Go code follows the package's cobra constructor idiom (`xCmd() *cobra.Command`, `cmd.OutOrStdout()`, error wrapping) and the test follows `fabhelp_test.go` style
- [x] A-013 No unnecessary duplication: Reuses cobra's own tree/field accessors and stdlib `encoding/json`; no reimplemented walkers or hand-rolled JSON

### Documentation Accuracy

- [x] A-014 R10: The `_cli-fab.md` entry accurately reflects the implemented command name, hidden/CI-only status, and contract fields

### Cross References

- [x] A-015 R7 R8: The CI step path (`dist/bin/fab-go-linux-amd64`) and the Homebrew-tap clone pattern referenced in the plan match the actual `justfile` naming and `release.yml` step they mirror

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Carry forward the frozen contract (intake #2/#3) and the hidden-subcommand placement (intake #6) verbatim; no apply-time deviation. | Intake froze these against the live `wt.json` and the verified `main.go`/`justfile` reality; implementation confirmed they hold. | S:97 R:75 A:95 D:97 |
| 2 | Certain | `version` field is wired from the package-level `var version` (same var cobra's `Version:` uses), passed into `dumpDoc(root, version)`. | `main.go` already declares `var version = "dev"` populated by `fab_ldflags`; the test passes an arbitrary value to prove non-hardcoding. | S:90 R:80 A:95 D:92 |
| 3 | Confident | Sort children by `Name()` explicitly even though cobra `.Commands()` is already alphabetical, to lock byte-stability. | Intake #12; redundant with cobra default but cheap insurance against future cobra ordering changes. | S:70 R:90 A:90 D:88 |
| 4 | Confident | Implement the entire CI delivery (dump+validate fatal, PR non-fatal+idempotent) as ONE workflow step with `continue-on-error` scoped only around the PR portion via `|| echo "::warning::..."`, keeping dump/validate fatal in the same `run:` block. | Intake #10/#11 require split fatality; a single step with an inline guard (validate exits non-zero → step fails; PR wrapped in `|| warning`) is the simplest faithful encoding and mirrors the tap step's single-step shape. Reversible CI tweak. | S:62 R:80 A:80 D:72 |
| 5 | Confident | Add `help/` to `.gitignore` (not just rely on never-committing). | Intake §3 marks it optional ("Optionally add `help/`"); adding it makes R9 enforceable and prevents accidental commits of the transient artifact. Trivially reversible. | S:65 R:90 A:85 D:80 |

5 assumptions (2 certain, 3 confident, 0 tentative).
