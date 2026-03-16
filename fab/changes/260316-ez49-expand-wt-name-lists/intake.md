# Intake: Expand wt name lists and fix wt list output

**Change**: 260316-ez49-expand-wt-name-lists
**Created**: 2026-03-16
**Status**: Draft

## Origin

> Need to increase the universe of names used by wt create
> Also: wt list output dashes (---) aren't aligned. We can also remove the dashes if needed.

Two-part request. No prior conversation.

## Why

The `wt create` command generates random worktree directory names using an adjective-noun pattern (e.g., `swift-fox`). The current word lists contain 48 adjectives and 48 nouns, yielding 2,304 unique combinations. While this works for light usage, the collision space is relatively small — especially for users who create and destroy worktrees frequently, or teams with multiple developers on the same project. Expanding the word lists increases the namespace and reduces the likelihood of name exhaustion or the need for retries in `GenerateUniqueName`.

If left unchanged, users with heavy worktree usage will increasingly hit retry loops or may eventually exhaust the namespace (particularly if old worktree directories linger on disk).

Separately, `wt list` output has a cosmetic bug: the separator dashes under column headers are sized to the header text length (e.g., 4 dashes for "Name") instead of the dynamic column width. This causes visual misalignment when worktree names or branches are longer than the header labels. The user is open to removing the dash separator row entirely as an alternative fix.

## What Changes

### Expand adjective and noun lists in `names.go`

**File**: `src/go/wt/internal/worktree/names.go`

Grow both word lists from 48 entries to ~120 each, roughly tripling the namespace from ~2,304 to ~14,400 combinations.

**Adjective list expansion** — add words that are:
- Short (1–3 syllables, ideally ≤7 characters) for readable directory names
- Positive or neutral connotation (no negative/grim words)
- Distinct from existing entries (no near-synonyms)
- Consistent with the existing style: nature, space, quality, and temperament themes

**Noun list expansion** — add animals that are:
- Short names (ideally ≤8 characters) for readable directory names
- Real animals (no mythical creatures)
- Distinct from existing entries (different animal families where possible)
- Grouped thematically (existing groups: raptors, mustelids, big cats, marine mammals, reptiles; new groups could include: primates, insects, fish, canids, ungulates, etc.)

### Update comments

Update the `~50` count comments at the top of each list to reflect the actual new count.

### Fix `wt list` separator alignment

**File**: `src/go/wt/cmd/list.go`

The separator row (lines 211-216) uses `len(headers[N])` to determine dash count, but the header and data rows use dynamically computed `colWidths[N]`. This causes misalignment:

```
  Name           Branch      Status  Path
  ----           ------      ------  ----     ← "Name" dashes = 4, but column is 14 wide
  very-long-name main                /path/...
```

**Fix**: Either (a) use `colWidths[N]` for the separator row to match the column widths, or (b) remove the separator row entirely — the user is fine with either approach. Removing the dashes is simpler and arguably cleaner output.

### No logic changes to name generation

The `GenerateRandomName()` and `GenerateUniqueName()` functions remain unchanged — they already work with any list size via `len()`.

## Affected Memory

- `fab-workflow/distribution`: (modify) If the packages spec mentions word list sizes, update the count

## Impact

- **`src/go/wt/internal/worktree/names.go`** — word list expansion
- **`src/go/wt/cmd/list.go`** — separator alignment fix
- **`wt` binary** — needs rebuild after edit (standard `go build`)
- **Existing worktrees** — no impact; names are assigned at creation time only
- **Tests** — if any tests assert on list length or list output formatting, update them

## Open Questions

None — the scope is clear and self-contained.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Expand to ~120 entries per list | User asked to "increase the universe"; tripling is a meaningful increase without making the file unwieldy | S:70 R:95 A:80 D:85 |
| 2 | Certain | Keep adjective-noun pattern | User said "names used by wt create" — no indication to change the naming scheme itself | S:75 R:90 A:90 D:95 |
| 3 | Certain | No logic changes to GenerateRandomName/GenerateUniqueName | Functions already handle variable-length lists; only the data needs to change | S:80 R:95 A:95 D:95 |
| 4 | Confident | Short, positive words only | Existing list follows this pattern; readable directory names require brevity | S:60 R:90 A:85 D:75 |
| 5 | Confident | Real animals only for nouns | Existing list is all real animals; consistency is the obvious default | S:60 R:90 A:85 D:80 |
| 6 | Certain | Single file for name lists | `names.go` is the sole location for these lists | S:80 R:95 A:95 D:95 |
| 7 | Confident | Remove separator row rather than fix alignment | User said "we can also remove the dashes if needed" — simpler fix, cleaner output | S:70 R:90 A:70 D:65 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
