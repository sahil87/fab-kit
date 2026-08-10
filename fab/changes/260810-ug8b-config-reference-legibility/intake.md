# Intake: Config Reference Legibility — Flatten Provider Blocks + Reorder Dispatch Above Agent

**Change**: 260810-ug8b-config-reference-legibility
**Created**: 2026-08-10

## Origin

Conversational (`/fab-discuss` session 2026-08-09/10 reviewing `fab config init --system --print` output). The user's raw asks, refined through discussion:

> 1) What do the double `# #` hashes in front of all providers other than claude mean? … the double hashes don't mean anything — just remove them.
> 3) Move the dispatch section above the agent section.

The double-hash semantics were explained (the inner `#` keeps non-claude blocks commented after a whole-block hoist, preventing kit-refreshed fills from being pinned as live overrides — `configref.go` presentation decision from 260805-j3cm). Two options were presented: keep the distinction with a legend line, or flatten to single-hash with the pinning warning moved into prominent prose. **The user chose flatten.** The dispatch-above-agent reorder was accepted without objection as a trivial registry row reorder.

## Why

1. **Pain point**: the rendered provider reference in `fab config init --system --print` (and the managed fence) shows non-claude provider blocks with a doubled comment marker (`# # codex:`). The semantics — "this line stays commented after you strip one `# ` layer to hoist the block" — are real but illegible: the project's own author could not decode them and read them as noise. A presentation whose meaning is invisible to its primary reader has failed regardless of the correctness of its intent.
2. **Consequence of not fixing**: every reader of the reference block trips over the same question; the protective intent (avoiding pinned stale fills) protects nothing if nobody understands it, while the visual noise actively degrades trust in the rest of the reference.
3. **Why flatten over the legend alternative**: a legend line preserves the two-tier presentation but keeps the visual irregularity; the user weighed both and chose uniform single-hash presentation, accepting the tradeoff that a whole-block hoist now yields four live provider overrides. The mitigation is to carry the "these fills refresh at kit-release cadence — a hoisted copy pins them; override per-field instead" warning prominently in the per-provider prose rather than encoding it in comment depth.

Secondary reorder: `dispatch:` (the mode/policy block) currently renders *below* `agent:` and `providers:`. Policy-before-capability reads better: `dispatch.mode` is the top-level decision (pane → native → headless), the agent knobs and providers table are what it consumes.

## What Changes

### 1. Flatten the non-claude provider blocks (`providersSegment`, `src/go/fab/internal/configref/configref.go`)

Current rendering (inside `providersYAML` / the block built around `configref.go:1108–1132`): the `codex:`/`agy:`/`kimi:` blocks and their `profiles:` fill lines are emitted **pre-commented** (`  # codex:` …), so `configupgrade.CommentOutSegment` produces the doubled `# #` form in any rendered fence, and stripping one `# ` layer from the whole block leaves them commented while claude comes out live.

New rendering: emit all four provider blocks **uniformly live** in the segment's canonical form (same indentation family as claude's block today). `profilesLines` loses its `# `-prefixing for non-claude fills (the function currently hardcodes `  #   profiles:` / `  #     {role}: …` — those become plain lines). After this change:

- In a rendered fence / `init --system --print`, every provider line carries exactly **one** `#`.
- Stripping `# ` from the whole block yields a live `providers:` mapping containing all four built-ins — valid YAML (the `YAMLSingleQuoted` guarantee is unaffected and still load-bearing for the agy/kimi nested-shell commands).
- The prose above the blocks gains a **prominent warning** (short, near the blocks, not buried mid-essay), approximately: `# NOTE: uncommenting a provider block makes its fills a live override that PINS these values — kit releases refresh the built-in fills, but a pinned copy shadows every refresh. Prefer overriding a single field (providers.<name>.profiles.<role>.model).` Exact wording at apply's discretion; the two required elements are *pins-on-hoist* and *prefer per-field override*.
- Prose lines that describe the old presentation must be swept: `"Every non-claude block below is commented because it merely restates a built-in default; claude's three capabilities are shown live as the baseline example."` and the function-comment paragraph at `configref.go:707–720` ("Presentation (260805-j3cm)…", "This segment carries the registry's only DELIBERATELY-COMMENTED content lines…"), plus the "strip the leading '# ' from a whole block" instruction at ~`configref.go:791–792` and the `YAMLSingleQuoted` doc comment's reference to it (the mechanism survives; the sentence describing per-provider comment layers changes).

### 2. Reorder `dispatch.*` registry rows above the `agent.*` rows (`configref.go` field registry)

Move the three rows `dispatch.mode` / `dispatch.column_width` / `dispatch.reap_done` (currently at ~`configref.go:602–650`) to immediately **before** `agent.session` (~`configref.go:542`). Resulting rendered order in the fence and in `init --system`: `…consolidate → dispatch → agent → providers → stage_hooks…`.

- Registry order is presentation order for the fence, `init --system`, and `config show` walks; no key semantics change.
- The `providers` row stays adjacent to (after) the `agent` rows — the discussion asked only for dispatch-above-agent.
- Any prose cross-references of the form "agent.session / agent.workers, **above**" inside `providersSegment`/`dispatchSegment` must be re-checked for directional words ("above"/"below") and corrected for the new order.

### 3. Test and fixture updates

Byte-stable rendering assertions break by design: `cmd/fab` config tests and `internal/configupgrade` fence-fixture tests that assert the rendered reference text (including any that call `YAMLSingleQuoted` to compose expected provider lines) are regenerated to the new single-hash, reordered form. No migration ships — the managed fence self-heals on the next `fab config upgrade`, and no key is renamed or restructured (presence=intent is untouched; a flattened reference block still registers no override until a user hoists it).

## Affected Memory

- `_shared/configuration.md`: (modify) documents the fence/reference presentation and the provider capability grammar — update the described rendering (single-hash provider blocks, pins-on-hoist warning, dispatch-before-agent order).

## Impact

- `src/go/fab/internal/configref/configref.go` — `providersSegment`, `providersYAML`, `profilesLines`, `providersShortSegment` (short form must match the flattened presentation), registry row order, directional prose.
- `src/go/fab/internal/configupgrade/` tests (fence fixtures), `src/go/fab/cmd/fab/` config command tests.
- `docs/specs/config.md` — describes the fence/reference; check for presentation-order or comment-depth claims and update.
- Rendered output surfaces: managed fence in every project `config.yaml` (self-heals on upgrade), `fab config init --system`, `fab config explain`.
- Go change ⇒ ships test updates (constitution); no skill files change unless a skill restates the presentation (grep `# #` and "commented reference" across `src/kit/`).
- **Collision note**: change `260810-ki9v-kimi-pane-enablement` adds kimi's `interactive_command` to the built-in defaults, which flows into this same rendered reference — whichever lands second takes a mechanical fixture rebase.

## Open Questions

- (none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Flatten non-claude provider blocks to single-hash (uniform live rendering in the segment) | Discussed — user chose flatten over the keep-with-legend alternative explicitly | S:95 R:70 A:90 D:95 |
| 2 | Certain | Carry a prominent pins-on-hoist warning in the provider prose | Discussed — stated condition of the flatten option, user accepted by choosing it | S:90 R:90 A:90 D:90 |
| 3 | Certain | Move dispatch.* registry rows above agent.* rows | Discussed — user directive, no objection raised in analysis | S:95 R:85 A:95 D:95 |
| 4 | Confident | No migration; fence self-heals via `fab config upgrade` | No key rename/restructure — matches the migrations rule (restructuring only); presentation-only regeneration | S:70 R:80 A:85 D:80 |
| 5 | Confident | `providers` row stays after the `agent` rows (only dispatch moves) | User asked only for dispatch-above-agent; minimal-move default | S:60 R:90 A:75 D:70 |
| 6 | Confident | Exact warning wording and placement (immediately above the provider blocks) | Two required elements fixed by discussion; phrasing left to apply | S:55 R:90 A:70 D:60 |

6 assumptions (3 certain, 3 confident, 0 tentative, 0 unresolved).
