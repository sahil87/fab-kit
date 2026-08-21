# Merge topologies

`fab operator autopilot` runs a queue of changes through the fab pipeline and opens a PR for each. How those PRs relate to `main` — and to each other — is controlled by `--mode`. There are three:

| Mode | Shorthand | PRs merged | Same-repo chain shape |
|------|-----------|------------|------------------------|
| `cherry-pick-ladder` (default) | `▂▄▆` | held until "merge all" | each PR based on `main`, carrying cherry-picked copies of earlier PRs |
| `merge-auto` | `░▒▓█` | merged as each completes | none — every change rebases onto the latest `main` independently |
| `stacked-prs` | `▄▀` | held until "merge all" | each PR based on the previous PR's branch (true stack, no cherry-pick commit) |

## `cherry-pick-ladder`

```
                    ┌───┐
            ┌───┐   │ C │
    ┌───┐   │ B │   ├╌╌╌┤
    │ A │   ├╌╌╌┤   │ b'│
    │   │   │ a'│   │ a'│
────┴───┴───┴───┴───┴───┴──▶ main
     PR1     PR2     PR3
```

Every PR stands on `main`; each successive diff is taller because it carries cherry-picked copies of its predecessors (`a'`, `b'`) below the dotted line. All PRs are held open together and merged base-first once you ask for "merge all".

## `merge-auto`

```
    ┌───┐          ┌───┐          ┌───┐
    │ A │          │ B │          │ C │
────┴─▼─┴●─────────┴─▼─┴●─────────┴─▼─┴●──▶ main
       merged         merged         merged
```

Nothing coexists and nothing is held: the operator merges each PR into `main` the moment it lands (▼ into ●), `main` advances, and the next change starts from the advanced line. No batch review, no re-stacking.

## `stacked-prs`

```
                    ┌───┐
            ┌───┐   │ C │  PR3 · base: B
    ┌───┐   │ B │   └───┘
    │ A │   └───┘  PR2 · base: A
────┴───┴──────────────────▶ main
    PR1 · base: main
```

Uniform height: every diff shows only its own delta, because each dependent branch is created off its dependency's *branch* rather than off `main` (no cherry-pick commit). The diagonal is load-bearing — merging a bottom box means re-seating the ones above it, so a "merge all" retargets and rebases each PR up the chain after the one below it lands.

## Choosing a mode

- **`cherry-pick-ladder`** — the safe default. Every PR is independently reviewable against `main`, at the cost of restated diffs further up the chain.
- **`merge-auto`** — for a queue of genuinely independent changes you trust to land one at a time without a batch review step.
- **`stacked-prs`** — for a queue with real same-repo dependencies where you want each PR's diff to show only its own change, at the cost of the merge-all choreography (base retarget + rebase per PR) when the stack comes down.

All three share the same underlying pipeline (`intake → apply → review → hydrate → ship → review-pr`) and queue-ordering rules — mode only changes *when* PRs merge and *how* same-repo dependents are branched. See the `fab-operator` skill's Coordination Patterns section for the full autopilot mechanics (dependency resolution, ordered merge, CI-failure handling).
