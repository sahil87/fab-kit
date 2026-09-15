# Intake: Tighten Slug Validation Regex

**Change**: 260811-nztj-tighten-slug-validation-regex
**Created**: 2026-08-11

## Origin

One-shot autonomous run from backlog bug [nztj], dispatched with instructions to verify the bug against the current codebase and, if still valid, drive the full pipeline (`/fab-new` → `/fab-fff`) without user interaction. Ambiguous calls are decided by agent judgment and recorded in `## Assumptions`.

> - [ ] [nztj] 2026-08-06: (BUG) slugRegex (change.go:21) accepts uppercase and unbounded words, contradicting documented lowercase/2-6-word slug rules — tighten the regex (case-insensitive-filesystem rationale in architecture.md:58)

**Verification performed before creation** (bug confirmed still valid):

- `src/go/fab/internal/change/change.go:21` — the pattern `^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$` accepts uppercase letters, any number of words, and consecutive hyphens (`a--b`).
- `docs/specs/architecture.md:58` — "All components are **lowercase only** — avoids collisions on case-insensitive filesystems (macOS default, Windows)."
- `docs/specs/architecture.md:54` and `docs/specs/naming.md:18` — slug documented as "2-6 word kebab-case description".
- `docs/memory/pipeline/change-lifecycle.md:25` — "All components MUST be lowercase."
- `src/kit/skills/_preamble.md` § Naming Conventions — "slug — 2-6 word kebab-case description".

So `fab change new --slug MyChange` or a 12-word slug passes validation today, producing folder/branch names that contradict every documented rule and (for uppercase) risk collisions on case-insensitive filesystems.

## Why

1. **The pain point**: `slugRegex` is the only enforcement point for slug format (used by both `change.New` at change.go:30 and `change.Rename` at change.go:113), and it silently accepts input the documentation forbids. An uppercase slug creates a folder and git branch whose name can collide with a differently-cased sibling on case-insensitive filesystems (macOS default, Windows) — the exact rationale documented at architecture.md:58. Unbounded word count lets degenerate slugs (whole sentences kebab-cased) become permanent folder/branch identities.

2. **Consequence of not fixing**: docs, memory, and validation disagree. Agents follow the documented rule (all 427 existing changes — 48 active + 379 archived — are lowercase, 2+ words), but a human or misbehaving agent calling `fab change new`/`fab change rename` directly can create a non-conforming, hard-to-rename identity (the `{YYMMDD}-{XXXX}` prefix is immutable; the folder name propagates into the git branch, log.md attribution, and PR metadata). The validation exists precisely to catch this and currently doesn't.

3. **Why this approach**: tighten the regex to match the documented contract — the docs are consistent across specs, memory, and skills, and the historical corpus already conforms. The alternative (loosening the docs to match the regex) would abandon the case-insensitive-filesystem rationale, which is a real technical constraint, not a style preference.

## What Changes

### `src/go/fab/internal/change/change.go` — tighten `slugRegex`

Replace the regex at line 21:

```go
// before
var slugRegex = regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`)

// after
var slugRegex = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+){1,5}$`)
```

The new pattern enforces, in one expression:
- **lowercase only** (`[a-z0-9]` — digits remain allowed, matching existing slugs like `embed-agent-defaults-layer0`)
- **2-6 words**: one mandatory head word plus 1-5 `-word` groups
- **no leading/trailing hyphen** (structurally impossible — each hyphen must be followed by a word)
- **no consecutive hyphens** (`a--b` no longer matches; kebab-case implies single separators)

### Error messages at both call sites

Update the two identical error strings (change.go:31 and change.go:114) to state the actual rule:

```go
return "", fmt.Errorf("Invalid slug format '%s' (expected 2-6 lowercase kebab-case words, e.g. 'add-oauth-support')", slug)
```

Both `New` and `Rename` keep sharing the single `slugRegex` — no divergence between create and rename validation.

### `src/go/fab/internal/change/change_test.go` — test coverage

Existing invalid-slug tests (`"my feature!"`, `"-starts-with-hyphen"`, empty) still pass. Add cases:
- uppercase rejected: `"MyFeature"`, `"my-Feature"`
- single word rejected: `"cleanup"`
- 7 words rejected: `"a-b-c-d-e-f-g"`
- consecutive hyphens rejected: `"my--feature"`
- boundary accepted: 2 words (`"my-feature"` — already covered) and 6 words (`"one-two-three-four-five-six"`)
- digits accepted: `"fix-v2-parser"`
- `Rename` rejects an invalid new slug (e.g. uppercase) — same regex, but guards the second call site

### `src/kit/skills/_cli-fab.md` — document the enforced format

Constitution: "Changes to the `fab` CLI (Go binary) MUST ... update `src/kit/skills/_cli-fab.md` with any new or changed command signatures." Signatures are unchanged, but the `new`/`rename` rows (lines 99-100) should note the enforced slug format so agents composing `--slug` values see the constraint at the call site, e.g. append to the `new` row: "`<slug>` must be 2-6 lowercase kebab-case words".

### Explicitly out of scope

- `idRegex` (change.go:22) — already strict, untouched.
- Existing folder names — validation runs only at `fab change new` / `fab change rename`; the 6 archived changes with 7-word slugs (of 379 archived) predate enforcement and remain valid on disk, unaffected.
- Spec files (`architecture.md`, `naming.md`) — already state the rule; specs are human-curated (Constitution VI) and need no edit.

## Affected Memory

- `pipeline/change-lifecycle.md`: (modify) the "All components MUST be lowercase" statement (line 25) gains enforcement — note that `fab change new`/`rename` now validate lowercase + 2-6 words and reject non-conforming slugs with a descriptive error.

## Impact

- **Code**: `src/go/fab/internal/change/change.go` (1 regex + 2 error strings), `src/go/fab/internal/change/change_test.go` (new cases). No other Go code references `slugRegex` or constructs slugs (verified: callers are `cmd/fab/change.go` only; all test fixtures across the repo use 2-word lowercase slugs).
- **Kit**: `src/kit/skills/_cli-fab.md` slug-format note (constitution-mandated CLI-doc sync).
- **Behavioral**: `fab change new`/`rename` now reject uppercase, 1-word, 7+-word, and `--`-containing slugs with a clear error. Agents already generate conforming slugs (`_intake.md` Step 1 instructs "2-6 word slug, lowercase"); the rare over-long generated slug now fails fast with a self-explanatory message instead of minting a non-conforming identity.
- **Tests**: `go test ./src/go/fab/internal/change/` plus the cmd exec tests.

## Open Questions

*None — the input names the exact file, line, documented rule, and rationale; all ambiguity was resolvable from docs and the historical corpus (recorded below).*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Enforce lowercase-only in the regex | The documented rationale (case-insensitive filesystem collisions, architecture.md:58) is technical, not stylistic; memory (change-lifecycle.md:25) says MUST; all 427 historical slugs conform | S:85 R:90 A:95 D:90 |
| 2 | Confident | Enforce the full 2-6 word bound (min AND max), not lowercase-only | Backlog names "unbounded words" as part of the bug; docs consistently say "2-6 words"; all 427 historical slugs have ≥2 words; 6 archived 7-word slugs predate enforcement and are unaffected (validation only at create/rename); easy to loosen later if it proves too strict | S:75 R:90 A:75 D:55 |
| 3 | Certain | Consecutive hyphens become invalid as a structural byproduct of the word-group regex | Kebab-case semantics; no historical slug contains `--`; the old regex's acceptance of `a--b` was accidental | S:70 R:90 A:90 D:85 |
| 4 | Certain | Update both error strings to state the enforced rule with an example | Error text must match validation or the rejection is unactionable; both call sites share the regex so both messages change identically | S:80 R:95 A:95 D:90 |
| 5 | Confident | Add a slug-format note to `_cli-fab.md` `new`/`rename` rows rather than treating this as a no-doc-change | Constitution mandates `_cli-fab.md` sync for CLI changes; the signature is unchanged but the accepted-value contract tightened, which is what agents composing `--slug` need to know | S:65 R:90 A:80 D:70 |
| 6 | Confident | Memory update scoped to one line in `pipeline/change-lifecycle.md` (enforcement note); no spec edits | Specs already state the rule (Constitution VI: human-curated, describe intent — intent is unchanged); memory records what actually shipped, and what shipped is new enforcement | S:60 R:85 A:80 D:70 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
