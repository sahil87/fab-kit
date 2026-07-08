# Plan: Preserve config.yaml Comments on fab_version Stamping

**Change**: 260708-yogn-preserve-config-comments-fab-version
**Intake**: `intake.md`

## Requirements

### Config Stamping: Byte-Preserving `fab_version` Update

#### R1: `setFabVersion` MUST preserve all untouched bytes of an existing config.yaml
`setFabVersion` SHALL update only the line it owns (`fab_version`) when the config file already exists, preserving every comment (header, inline, comment-only block), key order, indentation, and blank line byte-for-byte. It MUST NOT unmarshal-then-remarshal the whole file (the current behavior, which strips comments, alphabetizes keys, normalizes indentation, and collapses comment-only mapping keys to `null`).

- **GIVEN** a config.yaml with header comments, a commented-out `agent:` template block, non-alphabetical keys, and 2-space nested indentation
- **WHEN** `setFabVersion(path, "2.14.0")` is called
- **THEN** every line except the `fab_version:` line is byte-identical to the input
- **AND** the commented `agent:` block is NOT collapsed to `agent: null`
- **AND** key order and indentation are unchanged

#### R2: `setFabVersion` MUST replace the value of a top-level `fab_version:` line in place
When the file contains a top-level `fab_version:` line — a line starting at column 0 (index 0) whose first non-key token is the key `fab_version:`; indented or `#`-commented occurrences do NOT count — the function SHALL replace that line's value with the new version, preserving everything else.

- **GIVEN** a config.yaml containing the top-level line `fab_version: 2.13.1`
- **WHEN** `setFabVersion(path, "2.14.0")` is called
- **THEN** that line becomes `fab_version: 2.14.0` and no other line changes
- **AND** an indented `    fab_version:` or commented `# fab_version:` occurrence is never matched as the top-level line

#### R3: `setFabVersion` MUST preserve a trailing same-line comment on the `fab_version:` line
When the top-level `fab_version:` line carries a trailing `#` comment, the replacement SHALL preserve that comment by replacing only the value token. Whole-line replacement is a pre-authorized fallback if value-token replacement proves fiddly; the choice made MUST be recorded in `## Assumptions`.

- **GIVEN** the line `fab_version: 1.2.3  # pinned`
- **WHEN** `setFabVersion(path, "2.14.0")` is called
- **THEN** the line becomes `fab_version: 2.14.0  # pinned` (comment preserved)

#### R4: `setFabVersion` MAY write a previously-quoted value unquoted
When the existing value is quoted (`fab_version: "0.42.0"`), the replacement MAY write the new value unquoted. `readFabVersion` (`config.go:88`) unmarshals into a string field and parses both forms.

- **GIVEN** the line `fab_version: "0.42.0"`
- **WHEN** `setFabVersion(path, "0.43.0")` is called
- **THEN** `readFabVersion(path)` returns `0.43.0` (quoting of the written value is unconstrained)

#### R5: `setFabVersion` MUST append `fab_version:` when absent
When the file exists but has no top-level `fab_version:` line, the function SHALL append `fab_version: <version>` as a new final line, ensuring the file ends with exactly one trailing newline (no doubled or missing newline regardless of whether the input ended with a newline).

- **GIVEN** a config.yaml with `project:` content but no `fab_version:` line
- **WHEN** `setFabVersion(path, "2.14.0")` is called
- **THEN** the file gains `fab_version: 2.14.0` as its last line, followed by exactly one `\n`, with all prior content preserved

#### R6: `setFabVersion` MUST create the file (and parents) when missing
When the file does not exist, the function SHALL create parent directories (`os.MkdirAll`, as today) and write a file containing just `fab_version: <version>`. This keeps `TestSetFabVersion_NewFile` green.

- **GIVEN** a path whose parent directories do not exist
- **WHEN** `setFabVersion(path, "0.43.0")` is called
- **THEN** the parent dirs are created and the file contains `fab_version: 0.43.0` such that `readFabVersion` returns `0.43.0`

#### R7: Output MUST remain valid YAML and readable via `readFabVersion` in every case
In all cases (create, replace, append), the result SHALL parse as YAML and `readFabVersion(path)` SHALL return the new version. Read errors other than not-exist MUST keep failing loudly (do not silently swallow a genuine read failure); the splice no longer requires a parse to write, but the postcondition (valid YAML + correct readback) is enforced by tests. Any deliberate deviation from the prior `cannot parse existing config.yaml` error contract MUST be recorded in `## Assumptions`.

- **GIVEN** any of the above scenarios
- **WHEN** `setFabVersion` returns without error
- **THEN** `readFabVersion(path)` returns the new version and the file is valid YAML
- **AND** a genuine read error (not os.IsNotExist) surfaces as a returned error rather than being ignored

### Design Decisions

1. **Targeted line splice, not a `yaml.Node` round-trip** (Intake Assumption 1): iterate the file's lines, find the first top-level `fab_version:` line, replace its value in place; else append. — *Why*: byte-preserving for everything the function does not own, trivially testable, minimal. The function is slated for deletion by the separate `fab config upgrade` effort, so generality is not worth it. — *Rejected*: `yaml.Node` round-trip (statusfile.go pattern) — still re-normalizes indentation/style and is overkill for one guaranteed-top-level scalar.
2. **LF-delimited line handling** (Intake Assumption 10): split on `\n`; untouched lines keep their original bytes by construction. — *Why*: repo/toolchain is LF-only.
3. **Remove the now-unused `gopkg.in/yaml.v3` import from init.go if the splice no longer references it** — the import stays used elsewhere in the package (config.go), so package build is unaffected.

### Non-Goals

- The general comment-preserving config writer — that is the separate `fab config upgrade` effort which will delete `setFabVersion` entirely.
- Any migration file — this change restructures no user data (it stops the binary from destroying it); no `fab/` layout or `.status.yaml` shape change.
- Any CLI signature change (⇒ no `_cli-fab.md` update) or skill change (⇒ no `SPEC-*.md` mirror).

## Tasks

### Phase 2: Core Implementation

- [x] T001 Rewrite `setFabVersion` in `src/go/fab-kit/internal/init.go` as a targeted line splice: MkdirAll parents; if file missing, write `fab_version: <version>\n`; else read content, split into lines, find the first line matching a top-level `fab_version:` key (starts at column 0, not indented, not a `#` comment), replace its value in place preserving any trailing same-line comment; if no such line, append `fab_version: <version>` as the final line with exactly one trailing newline. Preserve a genuine (non-not-exist) read error as a returned error. <!-- R1 R2 R3 R4 R5 R6 R7 -->
- [x] T002 Remove the `gopkg.in/yaml.v3` import from `src/go/fab-kit/internal/init.go` if the rewritten `setFabVersion` no longer references it (verify no other symbol in init.go uses it first). <!-- R1 -->

### Phase 3: Tests

- [x] T003 In `src/go/fab-kit/internal/init_test.go`, keep `TestSetFabVersion_NewFile` and `TestSetFabVersion_ExistingFile` green and add preservation tests: (a) header + inline + comment-only-block comments survive byte-for-byte; (b) a fully-commented `agent:` template block is NOT collapsed to `agent: null`; (c) non-alphabetical key order is preserved; (d) 2-space nested indentation is untouched; (e) replace-if-present updates the top-level value in place; (f) append-if-missing adds `fab_version:` as the last line with exactly one trailing newline; (g) quoted-value replacement (`fab_version: "0.42.0"` → new) with readback; (h) trailing same-line-comment preservation; (i) `readFabVersion` returns the new version in every case. <!-- R1 R2 R3 R4 R5 R6 R7 -->

### Phase 4: Validation

- [x] T004 Run `go test ./internal/...` from `src/go/fab-kit/` (scope-first); confirm the new preservation tests pass and `TestUpgrade_*` in `upgrade_test.go` (which exercises the seam via `readFabVersion` readbacks) stays green. <!-- R1 R2 R3 R4 R5 R6 R7 -->

## Execution Order

- T001 blocks T002 (import removal depends on the rewrite), T003, and T004.
- T004 runs last (validation gate).

## Acceptance

### Functional Completeness

- [x] A-001 R1: An existing config.yaml with comments, non-alphabetical keys, and custom indentation is byte-identical after `setFabVersion` except the `fab_version:` line; the whole-file unmarshal/remarshal pipeline is gone.
- [x] A-002 R2: A top-level `fab_version:` line's value is replaced in place; indented/commented `fab_version:` occurrences are never matched as the top-level line.
- [x] A-003 R3: A trailing same-line comment on the `fab_version:` line is preserved (or whole-line-replacement fallback taken and noted in Assumptions).
- [x] A-004 R4: A previously-quoted value is replaced and `readFabVersion` returns the new version.
- [x] A-005 R5: A file lacking `fab_version:` gains it as the final line with exactly one trailing newline, all prior content preserved.
- [x] A-006 R6: A missing file (with missing parents) is created containing just `fab_version: <version>`; `TestSetFabVersion_NewFile` passes.
- [x] A-007 R7: Output is valid YAML and `readFabVersion` returns the new version in every case; a genuine (non-not-exist) read error still surfaces.

### Behavioral Correctness

- [x] A-008 R1: The recurring failure case (a commented `agent:`/providers template block) survives — the block is NOT collapsed to `agent: null` — verified by a dedicated test.

### Scenario Coverage

- [x] A-009 R1: `go test ./internal/...` from `src/go/fab-kit/` passes, including new preservation tests and the existing `TestUpgrade_*` readback assertions.

### Edge Cases & Error Handling

- [x] A-010 R5: Append case handles both an input ending with a newline and one not ending with a newline, producing exactly one trailing newline.
- [x] A-011 R2: Only the FIRST top-level `fab_version:` line is targeted; an indented or commented occurrence is left untouched.

### Code Quality

- [x] A-012 Pattern consistency: The rewritten `setFabVersion` follows surrounding init.go conventions (error wrapping with `fmt.Errorf`, `os` file ops, function focus < ~50 lines).
- [x] A-013 No unnecessary duplication: Existing helpers/imports reused; no new dependency added; unused `yaml` import removed from init.go if applicable.
- [x] A-014 Go changes ship tests: The `.go` change carries corresponding test updates in the same change (Constitution VII, test-alongside).
- [x] A-015 No migration / no CLI-signature / no SPEC-mirror obligations triggered: scope is confirmed Go-only in `src/go/fab-kit/internal/` (documentation_accuracy / cross_references extra categories).

## Notes

- Check items as you review: `- [x]`
- All acceptance items must pass before `/fab-continue` (hydrate)
- If an item is not applicable, mark checked and prefix with **N/A**: `- [x] A-NNN **N/A**: {reason}`

## Deletion Candidates

- None — the code this change made redundant (the `map[string]interface{}` unmarshal→`yaml.Marshal` pipeline and init.go's `gopkg.in/yaml.v3` import) was deleted within the change itself; no other symbol, branch, or config became unused (`readFabVersion`, both call sites `init.go:41` / `upgrade.go:120`, and `yaml.v3` elsewhere in the package remain live). `setFabVersion` itself is a recorded future deletion by the separate `fab config upgrade` effort, not made redundant by this change.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Targeted line splice (find first top-level `fab_version:` line, replace value in place; else append), not a `yaml.Node` round-trip | Intake Assumption 1 — user chose splice; byte-preserving, minimal, function is slated for deletion | S:95 R:70 A:90 D:90 |
| 2 | Confident | Trailing same-line comment preserved via value-token replacement (kept the comment; no whole-line fallback needed) | Intake Assumption 3 pre-authorized the fallback but value-token replacement was clean to implement | S:85 R:85 A:80 D:75 |
| 3 | Certain | Top-level `fab_version:` = line at column 0 whose first token is `fab_version:`; indented/commented occurrences excluded | Intake Assumption 5 — stated verbatim; matches YAML top-level semantics | S:90 R:85 A:90 D:85 |
| 4 | Certain | Quoted existing values rewritten unquoted | Intake Assumption 4 — `readFabVersion` parses both forms | S:90 R:90 A:85 D:80 |
| 5 | Confident | Genuine (non-not-exist) read errors are returned; the prior `cannot parse existing config.yaml` message is dropped because the splice no longer parses to write (postcondition tests enforce valid-YAML output) | Intake R7 authorized a sensible error-contract change with a note; parse-to-write is gone so a parse-error path no longer exists | S:75 R:80 A:80 D:75 |
| 6 | Certain | LF-delimited line handling; untouched lines keep original bytes by construction | Intake Assumption 10 — repo/toolchain is LF-only | S:80 R:90 A:85 D:80 |

6 assumptions (4 certain, 2 confident, 0 tentative).
