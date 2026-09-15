# Intake: docs-index — Fix Escape Runaway in Folder-Index Descriptions, and the Hardcoded Glossary Path

**Change**: 260909-esqt-docsindex-escape-roundtrip
**Created**: 2026-09-09

## Origin

Both bugs were hit in the **loom** repo (`/Users/sahil/code/wvrdz/loom`, branch `astral-mandrill`, PR #3244) while adopting `docs/specs` as a second `docs_index.roots` entry and backfilling `description:` frontmatter across 236 files. Discovered on **fab 2.24.0**; both re-confirmed present at fab-kit `main` @ `c3e9e74a` (v2.24.2) by reading the source cited below.

Filed as a single change because both live in `src/go/fab/internal/memoryindex/memoryindex.go`, both are "the generator writes a string it cannot read back correctly", and both are cheap. Split into two changes if the fixes want separate review.

---

# Bug 1 — Escape runaway: folder-index `description:` never converges

## Why

`fab docs-index` promises byte-stable, idempotent output. For any folder index whose `description:` contains an escaped double quote, it is neither: **the backslashes double on every run, without bound**, and the file is rewritten every time. In loom this made 6 files churn on every invocation, and it survives `--check` (reported as tier-1 benign drift, never a loss), so CI would show perpetual, unexplained diffs.

The comment directly above the offending line asserts the opposite — *"keeping the whole pipeline idempotent"* — so this is a violated invariant, not a missing feature.

## The defect

The round-trip is **asymmetric**: the write side escapes, the read side does not unescape.

- **Write** — `src/go/fab/internal/memoryindex/memoryindex.go:308`:
  ```go
  fmt.Fprintf(&b, "---\ndescription: %q\n---\n", d.Description)
  ```
  `%q` is Go-quoting: it wraps in `"` **and** backslash-escapes any `"` inside the value.

- **Read** — `src/go/fab/internal/frontmatter/frontmatter.go`, `stripQuotes()`:
  ```go
  func stripQuotes(s string) string {
      if len(s) >= 2 {
          if (s[0] == '"' && s[len(s)-1] == '"') || (s[0] == '\'' && s[len(s)-1] == '\'') {
              return s[1 : len(s)-1]
          }
      }
      return s
  }
  ```
  It removes the outer quote pair only. It does **not** unescape the interior.

So each cycle: read strips the outer quotes and hands back a value whose interior `\"` is still literally backslash-quote; `%q` then escapes that backslash *and* the quote, producing `\\\"`. Next run: `\\\\\"`. The escape prefix doubles every run.

This only bites **folder index files** (`index.md` / the `also_accept` landing), because only those round-trip their own `description:` through a file the generator rewrites (see the `DomainData.Description` doc comment at `memoryindex.go:173-177`). A **topic file's** `description:` is read but never rewritten, so it is stable — that asymmetry is why the bug looks intermittent.

## Reproduction (verified, minimal)

```sh
mkdir -p repro/docs/memory/demo/sub && cd repro
git init -q . && fab init
printf -- '---\nfkf_version: "0.1"\n---\n# Memory Index\n'                 > docs/memory/index.md
printf -- '---\ndescription: "Demo domain"\n---\n# Demo\n'                  > docs/memory/demo/index.md
printf -- '# Topic\nbody\n'                                                > docs/memory/demo/topic.md
printf -- '---\ndescription: "Sub with an escaped \\"quoted\\" phrase"\n---\n# Sub\n' > docs/memory/demo/sub/index.md
printf -- '# Note\nbody\n'                                                 > docs/memory/demo/sub/note.md

for i in 1 2 3 4; do fab docs-index >/dev/null 2>&1; sed -n '2p' docs/memory/demo/sub/index.md | head -c 90; echo; done
```

**Observed** — backslash run before `quoted` doubles each run: **4 → 8 → 16 → 32** (measured; the description line grew to 190 chars from ~60). `fab docs-index` reports `Updated:` for that file on every invocation, forever.

**Expected** — run 2 onwards report "already up to date" and the file is byte-identical.

Real-world instances in loom (all under `docs/specs/`, all folder indexes, each reached 60+ backslashes before I rewrote them by hand):
`architecture-v2/index.md`, `architecture-v2/3-pipeline/index.md`, `architecture-v2/2.5-controller/examples/index.md`, `architecture-v2/2.5-controller/examples/demo4-registry/index.md`, `architecture-v2/4-right-panel/index.md`, `architecture-v2/4-right-panel/instance-props/index.md`. The triggering content was ordinary and reasonable prose — e.g. a description mentioning `<Button variant="ghost"/>`, and another quoting the phrases `"a caller wants to change the canvas"` / `"the change is committed and re-rendered"`.

## Fix direction (author's judgement — not prescriptive)

Make read and write inverse operations. Options, roughly in order of preference:

1. **Emit YAML, parse YAML.** Write with a real YAML marshaller (or a single-quoted YAML scalar, doubling any interior `'`) and unescape on read to match. Most correct; also fixes adjacent hazards (a description containing a newline, a leading `%`, a trailing `:`).
2. **Keep `%q` on write, add `strconv.Unquote` on read.** Small and surgical: in `stripQuotes` (or its caller), when the value is double-quoted, try `strconv.Unquote` and fall back to the current byte-slice behavior on error. Must confirm no other caller depends on the current no-unescape semantics.
3. **Normalize before writing.** Reject or transform interior double quotes at write time (e.g. to typographic quotes). Cheapest, but lossy and surprising.

Whatever is chosen, the property to encode is: **`read(write(x)) == x` for all `x`**, including values containing `"`, `'`, `\`, and non-ASCII.

## Secondary: `--check` should catch non-convergence

`--check` classified this as tier-1 benign drift. A file that differs from its own regeneration *for the same inputs* is a convergence failure, not drift — arguably its own warning kind (or at minimum, mention in the docs that repeated `Updated:` on an unchanged tree indicates this). Worth a decision even if the answer is "leave it".

## Acceptance

- The reproduction above prints an identical line on runs 2, 3 and 4, and `fab docs-index` reports "already up to date" from run 2 on.
- A round-trip unit test over an adversarial corpus: `he said "hi"`, `it's`, `back\slash`, `<Button variant="ghost"/>`, `emoji → ✅`, a 500-char value, and a value with both quote kinds.
- Golden tests updated (`golden_test.go` pins the affected output).
- Idempotency test: generate twice into a fixture tree, assert byte-equality — this class of bug should have been caught by exactly such a test.

---

# Bug 2 — `docs/memory/index.md` hardcodes `../specs/glossary.md`

## Why

`memoryindex.go:283` writes a fixed navigation line into every generated root memory index:

```go
b.WriteString("> **New here?** Start with the [README](../../README.md) for setup and a walkthrough. For terminology, see the [Glossary](../specs/glossary.md).\n\n")
```

Both paths are assumptions about the *consuming repo's* layout, baked into the binary. In loom the glossary was moved to `docs/glossary.md` (it spans specs and memory, so it belongs to neither), which makes the generated link **permanently broken**: editing `docs/memory/index.md` works until the next `fab docs-index docs/memory`, which silently restores the dead link. There is no config key, no seed file, and no curated-block escape — the only workaround is to not run the generator, which is not a workaround.

Confirmed hardcoded in the shipped binary: `strings /Users/sahil/.fab-kit/versions/2.24.0/fab-go | grep 'specs/glossary'` matches.

This also contradicts the direction `docs-index` itself took in 2.24.0: roots became configurable, but this line still assumes `docs/memory` + `docs/specs` siblings with a glossary at a fixed path.

**Aggravating factor:** the repo's own `just check-links` did not catch the resulting broken link, because generated files sit outside its default corpus. A GitHub Copilot review on the PR is what surfaced it. Any repo relying on a link checker that skips generated output will ship this break silently.

## Fix direction (author's judgement)

Anything that stops the binary asserting the consumer's layout. In rough order of preference:

1. **Make it configurable per root** — e.g. `docs_index.roots[].nav_note` (free-text markdown, omitted when unset), or a `glossary:` path key. Fits the 2.24.0 configurable-roots direction.
2. **Put it inside the curated block**, so the first generation seeds it and thereafter the repo owns it — consistent with how `also_accept` landings already preserve prose outside the generated block.
3. **Drop the line.** It is navigation chrome; the repo's own index prose can carry it. Simplest, and arguably right — the generator's job is index tables, not editorial links.

Whichever is chosen, existing repos that *do* have `docs/specs/glossary.md` should not regress — a migration note (or defaulting to today's string when the key is unset) covers that.

## Acceptance

- A repo with its glossary at `docs/glossary.md` can generate a root memory index whose links all resolve, and that survives regeneration.
- A repo with `docs/specs/glossary.md` sees no change (or a documented one-line migration).
- No consumer-layout path strings remain hardcoded in the generated output — grep the package for `../specs/` and `../../README.md`.

---

## Affected Memory

- `runtime/docs-index` (or wherever the docs-index generator's behavior is recorded): (modify) record the round-trip contract (`read(write(x)) == x`) as a Design Decision, and the resolution of the hardcoded nav line. Confirm the actual target file at hydrate — this intake was authored from another repo and has not read fab-kit's memory tree.

## Impact

- **Files (bug 1)**: `src/go/fab/internal/memoryindex/memoryindex.go` (~line 308), `src/go/fab/internal/frontmatter/frontmatter.go` (`stripQuotes` and/or callers), plus tests.
- **Files (bug 2)**: `src/go/fab/internal/memoryindex/memoryindex.go` (~line 283), possibly the `docs_index.roots` config schema + `fab config explain`.
- **Behavior contract**: bug 1's fix changes bytes in already-corrupted index files — a one-time cleanup diff in consuming repos. Existing runaway values (`\\\\\\\"…`) will not self-heal, since the stored value genuinely contains those backslashes; consider whether the fix should collapse a run of backslashes before a quote on read, or whether repos fix their own (loom fixed 4 by hand). **This is the main open decision.**
- **Blast radius**: every repo using `fab docs-index`. Bug 1 only manifests with a quote in a folder-index description; bug 2 manifests in any repo whose glossary is not at `docs/specs/glossary.md`.

## Open Questions

1. **Migration for already-runaway values** — self-heal on read (collapse `\\+"` → `"`), or leave to consuming repos? Self-healing is friendlier but mutates content the generator does not own.
2. **Which fix shape for bug 1** — full YAML marshalling (correct, larger) vs. `strconv.Unquote` on read (surgical, keeps `%q`)?
3. **Which fix shape for bug 2** — config key, curated-block seed, or delete the line?
4. **Should `--check` gain a convergence class** for "regeneration is not a fixed point", distinct from benign drift?
5. **Is `docs/specs/` the only other hardcoded consumer path?** A grep for layout assumptions across the package is worth doing while in here.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Bug 1's root cause is the `%q` write (`memoryindex.go:308`) paired with a non-unescaping `stripQuotes` read | Both sites read at fab-kit `main` @ `c3e9e74a`; doubling reproduced 4→8→16→32 in a clean fixture repo | S:95 R:95 A:95 D:90 |
| 2 | Certain | Only folder indexes are affected; topic-file descriptions are stable | Topic descriptions are read but never rewritten; verified in the repro — the topic file held steady while the sub-folder index doubled | S:95 R:90 A:90 D:85 |
| 3 | Certain | Bug 2's line is hardcoded with no config or seed escape | Source line 283 read directly; `strings` on the shipped binary matches; the hand-edit-then-regen cycle was observed reverting in loom | S:95 R:95 A:95 D:90 |
| 4 | Confident | Both bugs belong in one change | Same file, same class, both small; splitting is trivial if review prefers it | S:80 R:70 A:60 D:40 |
| 5 | Tentative | An idempotency test (generate twice, assert byte-equality) does not currently exist for this path | Inferred from the bug surviving release, not from reading the test suite — **verify before claiming it in the plan** | S:40 R:60 A:50 D:35 |
| 6 | Tentative | Affected memory is a docs-index/runtime file | fab-kit's memory tree was not read while authoring this | S:35 R:50 A:60 D:30 |

6 assumptions (3 certain, 1 confident, 2 tentative, 0 unresolved).

## Notes for the implementing agent

- The reproduction is the fastest way in — build it first, watch the doubling, then read the two source sites.
- Bug 1's fix is easy to under-test. The regression that matters is **not** "does a quote survive one round trip" but "does the tree converge after N runs" — assert byte-equality across two consecutive generations over a fixture containing adversarial descriptions.
- Do not fix bug 1 by forbidding quotes in descriptions. Descriptions are prose about code; `<Button variant="ghost"/>` is exactly the kind of specific, useful routing signal the format is supposed to carry.
- Cross-repo verification is available: loom's PR #3244 has 236 backfilled descriptions across a 7-deep tree with `superseded` globs and `also_accept: [README.md]`. Pointing a fixed build at that tree and confirming a no-op second run is a strong end-to-end check.
