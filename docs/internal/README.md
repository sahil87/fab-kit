# docs/internal/

Maintainer- and design-facing notes that **must never reach the public docs site** (shll.ai/fab-kit).

This directory exists for the **audience axis** described in shll.ai's README-extraction
contract (§9): documentation is organized by *who it is for*, not by "wanted vs. unwanted".

| Audience | Lives in | Reaches the site? |
|----------|----------|-------------------|
| Users (tool consumers) | `README.md` slice, and the future `docs/site/` | Yes — the README slice today; `docs/site/` once §9 ships |
| GitHub-native readers | README tail (below `## Development`), `CONTRIBUTING.md` | No — fenced off by the tail boundary |
| **Maintainers** | **`docs/internal/`** | **Never** |

## Distinct from `docs/specs/`

`docs/internal/` is **not** the same as `docs/specs/`. Per Constitution VI, `docs/specs/`
holds **pre-implementation design intent** — human-curated specs that record the planned "why"
of a feature and are consulted by fab skills when generating change artifacts. `docs/internal/`
is for free-form maintainer/design notes that have no role in the fab pipeline and no place on
the public site (scratch design rationale, internal decisions, working notes).

If a note is pre-implementation design intent, it belongs in `docs/specs/`. If it is durable
"what shipped" knowledge, it belongs in `docs/memory/`. `docs/internal/` is the catch-all for
everything else that is maintainer-only and should stay off the site.
