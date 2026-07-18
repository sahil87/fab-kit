# Intake: Conform Repo to Standardized Toolkit Name — shll toolkit

**Change**: 260718-udwv-shll-toolkit-rename
**Created**: 2026-07-18

## Origin

One-shot `/fab-new` invocation with a fully-specified task brief (verbatim below). The brief pre-decides the blockquote text (byte-exact), the sweep scope, the constitution edit, and the do-not-touch identifier list — intake questioning was zero-question (all decisions Certain/Confident).

> Task: Conform this repo to the toolkit's standardized name — "shll toolkit".
>
> The toolkit formerly named "sahil87 toolkit" is now the **shll toolkit** (sahil87/shll#56). The readme-extraction standard's canonical README blockquote changed accordingly. This repo's constitution already binds it to revised standards without amendment — this task is the conformance work.
>
> Precondition: `shll standards readme-extraction` runs on this machine and shows the new blockquote (below). If not, run `shll update`; if it still shows the old line, stop and report — do not proceed from memory.
>
> 1. **README blockquote** — replace the toolkit blockquote with this exact line, byte-identical, keeping the mandated head order (H1 -> blockquote -> badges): `> Part of the [shll toolkit](https://shll.ai) — see all projects there.`
> 2. **Prose sweep** — replace remaining `sahil87 toolkit` -> `shll toolkit` and `sahil87 tool(s)` -> `shll tool(s)` wherever they appear as prose: README, `docs/site/**` (including the skill bundle `docs/site/skill.md` if present), CLI help text and user-visible strings (update their test goldens), and `fab/project/` files. If this repo embeds docs in the binary (skill bundle or similar), re-run its sync step so drift-guard tests pass.
> 3. **Constitution (cosmetic, same PR)** — in the Toolkit Standards article, change "part of the sahil87 toolkit" to "part of the shll toolkit" and bump `Last Amended` per the file's governance line. Nothing else in the article changes.
> 4. **Do NOT touch identifiers**: `sahil87/tap` formula names, `github.com/sahil87/…` and `raw.githubusercontent.com/sahil87/…` URLs, the `sahil87/shll` canonical-source reference in the constitution article, and any GitHub-owner constants in code. Historical artifacts (`fab/changes/` archives) stay untouched.
>
> Ship per this repo's normal flow (one fab change -> PR). Tests green; if help text changed, the help-dump JSON shape is unchanged (text-only edits — no `schema_version` bump).

**Precondition — VERIFIED at intake time (2026-07-18)**: `shll standards readme-extraction` on this machine renders the revised standard; its canonical blockquote section reads exactly:

```markdown
> Part of the [shll toolkit](https://shll.ai) — see all projects there.
```

No `shll update` was needed. Do not re-derive this from memory downstream — it is confirmed against the live standard.

## Why

1. **Problem**: The toolkit this repo belongs to was renamed from "sahil87 toolkit" to "shll toolkit" (sahil87/shll#56), and the readme-extraction standard's canonical README blockquote changed with it. This repo's README still carries the old blockquote (`> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.`) plus scattered old-name prose, so it is now non-conformant with a standard its constitution binds it to.
2. **Consequence if unfixed**: The constitution's `### Toolkit Standards` article (v1.4.0) makes revised standards binding *without further amendment* — every future pipeline run that touches README/docs/site must check against the standards, and the repo currently fails that check. shll.ai pulls the README slice daily; the stale blockquote and old name render on the public site.
3. **Approach**: A single mechanical conformance change (exact blockquote splice + scoped prose sweep + cosmetic constitution wording), shipped as one fab change → one PR. No alternatives were considered because the standard prescribes the byte-exact target text.

## What Changes

The full occurrence map below was produced by repo-wide grep at intake time (excluding `fab/changes/` archives and gitignored `.claude/skills/` deployed copies). It is exhaustive — CLI help text and user-visible Go strings contain **zero** occurrences (verified: the only `sahil87` hits in `src/go/` are identifiers — `sahil87/tap/fab-kit` formula strings in `sync.go:149`/`update.go:26`, the `githubRepo = "sahil87/fab-kit"` constant in `download.go:21`, and code comments referencing `sahil87/shll.ai` — all untouched). Therefore: **no test goldens change, no help-dump changes, no `schema_version` bump** (the brief's conditional simply does not fire).

### 1. README blockquote (`README.md:3`)

Replace:

```markdown
> Part of [@sahil87's open source toolkit](https://shll.ai) — see all projects there.
```

with this exact line, byte-identical:

```markdown
> Part of the [shll toolkit](https://shll.ai) — see all projects there.
```

Head order is already H1 (`# Fab Kit`, line 1) → blockquote (line 3) → badges (line 5) and MUST remain so — this is a single-line content replacement, no reordering.

### 2. README prose (`README.md:21`)

`Installs fab-kit (plus the shll meta-CLI) via Homebrew, handling tap trust automatically. To install the entire sahil87 toolkit instead:` — replace `sahil87 toolkit` → `shll toolkit`. Only this one prose occurrence exists in the README beyond the blockquote.

### 3. Skill bundle (`docs/site/skill.md:53`) + embedded copy re-sync

Line 53 currently reads:

```markdown
fab is one member of the [@sahil87 toolkit](https://shll.ai) and composes with its siblings
```

Replace the link text `[@sahil87 toolkit]` → `[shll toolkit]` (the `@` sigil is part of the old name styling and drops with it; URL unchanged):

```markdown
fab is one member of the [shll toolkit](https://shll.ai) and composes with its siblings
```

Then re-run the embed sync so the drift guard passes: `bash scripts/sync-skill.sh` refreshes the committed copy `src/go/fab/cmd/fab/skill.md` from the canonical `docs/site/skill.md`; `TestSkillEmbedMatchesCanonical` (in `src/go/fab/cmd/fab/skill_test.go`) pins them byte-identical. The edit does not change the line count, so the ≤150-line bundle-budget test is unaffected.

### 4. Kit skill prose (`src/kit/skills/_cli-fab.md:583`) + SPEC mirror (`docs/specs/skills/SPEC-_cli-fab.md:33`)

Both files carry the phrase `the sahil87 toolkit-wide \`skill\` standard` — replace `sahil87 toolkit-wide` → `shll toolkit-wide` in each. The SPEC mirror update is constitution-mandated ("Changes to skill files MUST update the corresponding `docs/specs/skills/SPEC-*.md`"). The surrounding `shll docs/site/standards/skill.md` references are repo/path identifiers and stay verbatim. (Deployed copies under `.claude/skills/` are gitignored `fab sync` output from the installed kit version — never edited; the fix reaches them via a future release.)

### 5. Constitution — Toolkit Standards article (`fab/project/constitution.md:38`)

In the article body, replace `This tool is part of the sahil87 toolkit` → `This tool is part of the shll toolkit`. Nothing else in the article changes — in particular the `sahil87/shll repository's docs/site/standards/ tree` canonical-source reference stays verbatim (identifier).

Governance line handling (`Last Amended` bump per the brief): the line currently reads `**Version**: 1.4.0 | **Ratified**: 2026-02-06 | **Last Amended**: 2026-07-18` and today is 2026-07-18, so the bumped value is byte-identical — the date stays `2026-07-18`. Version stays `1.4.0`: the file's own amendment convention (see the 260601-j6cs and 260611-zc9m comments) bumps the version only when a normative rule changes; this is cosmetic wording. Per that same convention, append a dated HTML amendment comment after the existing ones, briefly noting the cosmetic rename (toolkit renamed sahil87 → shll per sahil87/shll#56; no normative change; no version bump). The existing historical amendment comments (including 260717-y8it's "sahil87 toolkit" wording at lines 56–61) are records of past amendments and stay untouched.

### 6. Explicitly out of scope / untouched (verified occurrence-by-occurrence)

- All URLs: `github.com/sahil87/…`, `raw.githubusercontent.com/sahil87/…`, `img.shields.io/…/sahil87/…` badge URLs
- `sahil87/tap` formula names (Go strings in `src/go/fab-kit/internal/sync.go`, `update.go`; docs)
- `githubRepo = "sahil87/fab-kit"` (`src/go/fab-kit/internal/download.go:21`) and `REPO="sahil87/fab-kit"` (`scripts/install.sh:13`)
- `sahil87/shll` canonical-source references (constitution article, `scripts/sync-skill.sh` comment, `helpdump.go` comment)
- `fab/changes/` archives (historical artifacts) — several carry the old blockquote/name verbatim; stay as-is
- `docs/memory/distribution/log.md` + `log.seed.md` entries about the wvrdz→sahil87 GitHub-org migration (org identifiers, not the toolkit name)
- `docs/specs/findings/binary-review-2026-06-12.md:1047` (`repo sahil87/fab-kit` — identifier)
- CLI help text / cobra strings / help-dump goldens — zero occurrences exist, nothing to change

## Affected Memory

- `distribution/distribution`: (modify) Two present-truth mentions of the old name — the frontmatter/overview description (line 11: "fab-kit's conformance to the sahil87 toolkit's published standards") and the Toolkit Standards section (line 399: "fab-kit is part of the **sahil87 toolkit**") — update to "shll toolkit" at hydrate; note the rename source (sahil87/shll#56) where the section explains the standards binding.

## Impact

- **Files edited (apply)**: `README.md` (2 lines), `docs/site/skill.md` (1 line), `src/go/fab/cmd/fab/skill.md` (regenerated via `scripts/sync-skill.sh`), `src/kit/skills/_cli-fab.md` (1 line), `docs/specs/skills/SPEC-_cli-fab.md` (1 line), `fab/project/constitution.md` (1 line + appended amendment comment)
- **Files edited (hydrate)**: `docs/memory/distribution/distribution.md` (2 passages) + regenerated memory indexes (`fab memory-index`)
- **Tests**: `go test ./...` under `src/go/` must stay green — the relevant guard is `TestSkillEmbedMatchesCanonical` (byte-equality of embedded bundle vs. canonical) and the bundle line-budget test; no golden files change
- **No behavior change**: zero Go logic touched; the only `src/go/` diff is the embedded markdown asset
- **Public site**: shll.ai's next README-slice pull picks up the new blockquote/name automatically
- **Ship**: one PR to `main` per normal flow; branch `260718-udwv-shll-toolkit-rename` (current worktree branch `std3` is a placeholder — see branch handling at activation)

## Open Questions

*None — the brief pre-resolved all decision points, and the precondition was verified live.*

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | README blockquote replaced with the brief's exact line, byte-identical, head order preserved | Text given verbatim in the brief AND verified against the live `shll standards readme-extraction` output at intake time | S:95 R:90 A:100 D:100 |
| 2 | Certain | CLI help text, goldens, and help-dump are untouched (no `schema_version` bump question arises) | Verified by grep: zero `sahil87 toolkit`/`sahil87 tool` occurrences in user-visible Go strings — only identifier constants, which the brief excludes | S:85 R:95 A:95 D:90 |
| 3 | Confident | `[@sahil87 toolkit]` in `docs/site/skill.md:53` becomes `[shll toolkit]` — the `@` sigil drops with the old name | Brief's mapping is `sahil87 toolkit` → `shll toolkit`; a literal splice would yield `[@shll toolkit]`, which matches neither the new name nor the standard's own phrasing "the [shll toolkit](https://shll.ai)" | S:60 R:90 A:80 D:75 |
| 4 | Confident | Prose sweep extends to `src/kit/skills/_cli-fab.md:583` and its constitution-mandated mirror `docs/specs/skills/SPEC-_cli-fab.md:33` | Brief's principle is "wherever they appear as prose"; the enumerated list is illustrative scope, and the constitution requires SPEC mirrors to track skill-file edits | S:55 R:85 A:80 D:70 |
| 5 | Confident | Constitution: `Last Amended` stays `2026-07-18` (bump target equals current value — amended today), version stays 1.4.0, dated amendment comment appended | File's own convention: 260601/260611 comments show non-normative amendments bump no version but always leave a dated comment; brief says "nothing else in the article changes" — the comment block sits outside the article | S:60 R:90 A:85 D:70 |
| 6 | Confident | Historical records keep old wording: `fab/changes/` archives AND the constitution's existing 260717-y8it amendment comment (lines 56–61) | Brief explicitly freezes archives; amendment comments are the same class — dated records of what was true when written | S:55 R:90 A:80 D:70 |
| 7 | Certain | `docs/memory/distribution/distribution.md` old-name mentions are updated at hydrate (Affected Memory), not in the apply sweep | Standard pipeline ownership: memory is hydrate's artifact; both passages are present-truth claims now stale | S:70 R:95 A:90 D:85 |

7 assumptions (3 certain, 4 confident, 0 tentative, 0 unresolved).
