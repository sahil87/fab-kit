# Intake: Add the reserved `fab skill topics` verb

**Change**: 260903-psgm-fab-skill-topics-verb
**Created**: 2026-09-03

## Origin

Promptless dispatch (`/fab-proceed` create-new path) from a live conversation in which the shll toolkit's `skill` standard was re-read live (`shll standards skill`; canonical source `sahil87/shll` `docs/site/standards/`, rendered on shll.ai) and found to have been extended with a machine-readable topic-enumeration mandate. The constitution's `### Toolkit Standards` article binds fab-kit to these standards without further amendment. Verification against the live standard text and the installed fab 2.23.11 (which matches the project pin) was performed on 2026-09-02 during that conversation.

> Add the reserved `fab skill topics` verb mandated by the shll toolkit skill standard. `<tool> skill topics` prints the tool's content-topic names, one per line, raw to stdout — stderr empty, exit 0. A tool shipping zero topic pages prints empty stdout (zero bytes) and exits 0. fab ships no topic pages, so the required behavior is: empty stdout, empty stderr, exit 0. Today `fab skill topics` fails: exit 2, `ERROR: unknown command "topics" for "fab skill"` on stderr.

Key decisions carried from the conversation (see `## Assumptions` for grades):

- Minimal conformance change only — add the `topics` positional to `fab skill` (Go binary), printing nothing and exiting 0.
- The change MUST update `src/kit/skills/_cli-fab.md` and MUST include Go test updates (constitution CLI constraint).
- The topic-page-only mandates (`Topics:` help line, unknown-topic fail-fast naming valid topics, core-bundle topic index) are NOT added now, but must not be precluded — the standard names fab-kit as an expected future topic-pages candidate.
- One design point was deliberately left open in the conversation: the error shape for `fab skill <unknown-topic>` (recorded as a deferred row below).

## Why

1. **Problem**: The shll `skill` standard now reserves the topic name `topics` in every adopting tool's topic namespace and mandates that `<tool> skill topics` be a scriptable, always-available answer to "what topics do you have?" — binding **every** adopting tool, topic pages or not. fab adopted the `skill` standard (fskl, audited PASS at shll v0.0.23) and is therefore bound; today `fab skill topics` is a cobra usage error (exit 2, `ERROR: unknown command "topics" for "fab skill"` on stderr — `skill.go` uses `cobra.NoArgs`), which violates the new mandate.
2. **Consequence if unfixed**: fab is non-conformant with a published toolkit standard the constitution binds it to. Any composer or script that probes topic enumeration across the toolkit (e.g., the shll composer `shll skill <tool> <topic>` machinery, or a cross-tool topic index builder) gets a hard failure from fab instead of the standard's "zero topics" signal (empty stdout, exit 0).
3. **Why this approach**: The standard fixes the surface exactly — a positional `topics` verb (a `--list` flag was explicitly rejected upstream because the shll composer forwards positional args verbatim), raw stdout, exit 0, empty output for a zero-topic tool. fab ships zero topic pages (`docs/site/skill/` does not exist), so the minimal conforming implementation prints nothing and exits 0. Anything more (help-line enumeration, topic index, unknown-topic fail-fast naming valid topics) belongs to the topic-page mandates that do not bind a zero-topic tool today and would be speculative.

## What Changes

### Go binary: `fab skill` accepts the reserved `topics` positional

`src/go/fab/cmd/fab/skill.go` — `skillCmd()` currently uses `Args: cobra.NoArgs`, making every argued invocation a usage error (exit 2 via the binary-wide `run()` classification). New behavior:

| Invocation | Behavior |
|------------|----------|
| `fab skill` | Unchanged — prints the embedded bundle byte-identical to `docs/site/skill.md`, stderr empty, exit 0 |
| `fab skill topics` | **New** — prints empty stdout (zero bytes), stderr empty, exit 0 (fab ships zero topic pages) |
| `fab skill <anything-else>` | Remains a usage error → exit 2 (cobra's default unknown-argument error; see deferred assumption #7 on the error shape) |

Implementation shape (Confident assumption, apply decides details): replace `cobra.NoArgs` with a custom `Args` validator (or equivalent in-command branch) that accepts zero args or exactly the single arg `topics` — **not** a registered cobra child command. A child command would surface `topics` in `fab skill --help` and add a node to the frozen `fab help-dump` JSON tree (shll.ai command-reference pull surface); the plain positional keeps both surfaces byte-stable and matches the standard's "machine affordance, not a topic page" framing. When the arg is `topics`, `RunE` writes nothing and returns nil. Since output is zero bytes by construction, keep the seam pattern (`runSkill`-style) only as far as it stays natural — no framing, no trailing newline.

The router is unaffected: `skill ∉ LifecycleCommands`, always-routed; adding a positional changes no routing.

### Tests: `src/go/fab/cmd/fab/skill_test.go`

Per the constitution's CLI constraint, Go test updates ship in the same change:

- `fab skill topics` through the assembled cobra command → empty stdout (zero bytes), empty stderr, exit success (nil error).
- Bare `fab skill` contract unchanged (existing `TestSkill_StdoutByteIdentical` / `TestSkill_EmptyStderrThroughCobra` keep passing).
- `TestSkill_RejectsArgs` updated: an unknown positional (e.g., `unexpected`) is still a usage error, while `topics` is not — the test comment currently says "the standard says `fab skill` takes no args/flags" and must be reworded to the new contract. Whatever unknown-arg error-shape decision is made is pinned by this test (deferred assumption #7).

### CLI reference: `src/kit/skills/_cli-fab.md` § fab skill

Update the signature block and prose (currently: "Takes no args/flags (`cobra.NoArgs`); an argued invocation is a usage error → exit 2"). New content: the reserved `topics` positional — prints content-topic names one per line raw to stdout, stderr empty, exit 0; fab currently ships zero topic pages so output is empty (zero bytes); `topics` is reserved by the shll `skill` standard in every tool's topic namespace and is a machine affordance, not a topic page (never listed as a content topic). Other positionals remain usage errors → exit 2.

### Docs/memory sweep (stale-claim class)

Both distribution-domain memory files restate the "no args / `cobra.NoArgs` / argued invocation is a usage error" claim and must be updated at hydrate:

- `docs/memory/distribution/distribution.md` § Toolkit Standards Conformance — the `skill` standard PASS entry ("`cobra.NoArgs` makes an argued invocation a usage error (exit 2 ...)").
- `docs/memory/distribution/kit-architecture.md` § fab-go Subcommands — the `fab skill` entry (same claim, plus "visible zero-arg toolkit-standard bundle command").

Per code-quality.md § Sibling Sweeps, grep repo-wide for the claim class before finishing apply (known occurrences beyond the two memory files: `skill.go` doc comments, `skill_test.go` comments, `_cli-fab.md` § fab skill; check `docs/specs/architecture.md`'s config-free roster mention of `fab skill` for arg-shape claims).

### Explicit non-changes

- `docs/site/skill.md` is untouched — it is 124/150 lines, mentions no topic surface today, and the reserved verb MUST NOT be listed as a content topic.
- No `docs/site/skill/` directory, no canonical `topics.md` page, no `Topics:` help line, no core-bundle topic index — those mandates bind only when topic pages ship. The implementation must not preclude them (the custom Args validator is trivially extended to a topic-name set later).
- No new exit codes, no router change, no config surface, no migration (no user data restructured).

## Affected Memory

- `distribution/distribution.md`: (modify) Update the `skill` standard conformance entry in § Toolkit Standards Conformance — the reserved `topics` verb, the new no-longer-`cobra.NoArgs` arg contract, and the zero-topic empty-output behavior.
- `distribution/kit-architecture.md`: (modify) Update the `fab skill` entry in § fab-go Subcommands — same claim class ("visible zero-arg", "`cobra.NoArgs` makes an argued invocation a usage error").

## Impact

- **Code**: `src/go/fab/cmd/fab/skill.go` (Args validation + `topics` branch), `src/go/fab/cmd/fab/skill_test.go` (new + reworded tests). Scope the test run to the `cmd/fab` package first (`go test ./cmd/fab/...` from `src/go/fab/`).
- **Kit docs**: `src/kit/skills/_cli-fab.md` § fab skill (constitution-mandated).
- **Memory**: the two distribution-domain files above (hydrate).
- **Specs**: `docs/specs/architecture.md` mentions `fab skill` in the config-free roster — sweep for arg-shape claims; no structural spec change expected.
- **Unaffected**: `docs/site/skill.md` (byte-frozen for this change — the embed drift guard `TestSkillEmbedMatchesCanonical` keeps passing untouched), `fab help-dump` output (no new cobra command node), router `LifecycleCommands`, exit-code conventions.
- **Release sizing**: additive CLI surface → next release MINOR by convention.

## Open Questions

- Should `fab skill <unknown-topic>` keep cobra's default unknown-argument usage error (exit 2), or adopt the standard's unknown-topic fail-fast shape now ("non-zero exit, error on stderr naming the valid topics" — which today would name none)? The standard's unknown-topic rule textually applies only to tools shipping topic pages; the conversation deliberately left this open and required only that the decision made be recorded and test-pinned.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | `fab skill topics` prints empty stdout (zero bytes), empty stderr, exit 0 | Mandated verbatim by the live shll `skill` standard for a zero-topic-page tool; verified against the standard text 2026-09-02; constitution's Toolkit Standards article binds without amendment | S:95 R:80 A:95 D:95 |
| 2 | Certain | Positional `topics` verb, not a flag | The standard reserves the positional form and explicitly rejected `--list` (the shll composer forwards positional args verbatim) | S:95 R:70 A:95 D:95 |
| 3 | Certain | `topics` is a machine affordance, never a content topic — no `docs/site/skill/topics.md`, no `Topics:` help line, no topic-index listing; `docs/site/skill.md` stays untouched | Standard states this explicitly; skill.md is at 124/150 lines and mentions no topic surface today | S:90 R:85 A:90 D:90 |
| 4 | Certain | Constitution CLI constraint applies: update `src/kit/skills/_cli-fab.md` § fab skill and ship Go test updates in the same change | Constitution Additional Constraints + code-review.md project rule state this deterministically | S:90 R:80 A:100 D:95 |
| 5 | Certain | Memory sweep covers `distribution/distribution.md` (skill-standard conformance entry) and `distribution/kit-architecture.md` (fab skill entry) — both restate the stale "`cobra.NoArgs` / argued invocation is a usage error" claim | Verified by grep in this session; code-quality.md § Sibling Sweeps names the memory-file class | S:80 R:85 A:90 D:85 |
| 6 | Confident | Topic-page-only mandates (`Topics:` help line, unknown-topic fail-fast naming valid topics, core-bundle topic index) are NOT implemented now, but the implementation must not preclude them | Conversation decision; the standard applies those rules only to tools shipping topic pages and names fab-kit as a future topic-pages candidate | S:85 R:75 A:80 D:70 |
| 7 | Unresolved | Error shape for `fab skill <unknown-topic>`: front-runner is keeping cobra's default unknown-argument usage error (exit 2), since the standard's fail-fast rule textually binds only topic-shipping tools and would today name zero valid topics | Deferred — promptless dispatch; the conversation explicitly left this open and required only that the decision be recorded and test-pinned. Trivially reversible either way | S:40 R:80 A:35 D:35 |
| 8 | Confident | Implement via a custom `Args` validator on the existing `skillCmd` (accept zero args or exactly `topics`), not a registered cobra child command | A child command would surface `topics` in `fab skill --help` and add a node to the frozen `fab help-dump` JSON (shll.ai pull surface); the plain positional keeps both byte-stable and matches the "machine affordance" framing | S:70 R:80 A:75 D:65 |

8 assumptions (5 certain, 2 confident, 0 tentative, 1 unresolved).
