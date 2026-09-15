# Intake: config-show-compose-defaults

**Change**: 260810-rvza-config-show-compose-defaults
**Created**: 2026-08-10

## Origin

Conversational — raised during a `/fab-discuss` session on the config surface:

> Currently — it's easy for me to check the full config using `fab config show --origin`. But it feels off. `--origin` is to track the source. Not to see the final composed config. Right now, if I do `fab config show` — the default values I don't specify in system or project config get missed out. Do you have suggestions to improve this?

Discussion located the inconsistency precisely and agreed the fix: bare `fab config show` deliberately skips the built-in-defaults tier (`cmd/fab/config.go` — "Bare `show` prints the file+environment merge and never consults the defaults tier"), while its own `Long` help text claims it "resolves the effective config across the cascade … > built-in defaults — and prints it", and both the **keyed** form (`fab config show agent.workers`) and `--origin` already compose defaults. The user accepted the recommendation: make bare `show` print the fully composed config and keep `--origin` purely as the provenance annotator.

## Why

1. **Pain point**: there is no way to see the true effective config without `--origin`, which conflates two questions — "what will fab do?" (composition) and "where did each value come from?" (provenance). Users reach for a provenance flag just to see values.
2. **Consequence of not fixing**: bare `show` silently under-reports — a user reading it concludes fields like `dispatch.mode` or the provider fills are unset/unknown, when they resolve to concrete built-ins at point of use. The command also contradicts its own help text.
3. **Why this approach**: the composed view is what "show the effective config" means; the sparse "only what I set" view is the special case. The keyed form already behaves this way, so this aligns bare `show` with the rest of its own command rather than adding a new verb or flag.

## What Changes

### 1. Bare `fab config show` composes the built-in defaults tier

In `cmd/fab/config.go`, bare `show` (no key, no `--origin`) merges the defaults projection beneath env/system/project — the same `readModelDefaults` → `configref.DefaultsMap`/`DefaultsMapFor` projection the keyed and `--origin` paths already use — and prints the fully composed config as YAML. The derived `agent.profiles` rows keep composing against the live depth knobs (existing `DefaultsMapFor` behavior, reused not rebuilt).

### 2. `--origin` is unchanged

Provenance annotation (per-field origin, keyed full-stack view) stays exactly as it is; it stops being the only route to the composed values.

### 3. No sparse-view flag (initially)

No `--set-only`/`--sparse` companion flag ships in this change. The sparse view remains reachable by reading the files themselves or filtering `--origin` output for non-`default` tiers.
<!-- assumed: sparse-view flag omitted — YAGNI until someone asks; adding it later is additive and cheap -->

### 4. Help text, docs, and tests

- `Long`/`Example` text in `configShowCmd` rewritten — the "built-in defaults are NOT materialized here" carve-out is deleted; the command now does what its opening sentence says.
- `src/kit/skills/_cli-fab.md` § fab config updated (CLI behavior change ⇒ docs per constitution).
- `docs/specs/config.md` — the show-verb description updated where it records the bare-show/defaults-tier split.
- `cmd/fab/config_test.go` (and `config_show_init_test.go` if it pins bare-show output): existing bare-show cases updated to expect composed output; a case asserting a pure-default field (e.g. `dispatch.mode: native` on a bare project) appears in bare `show`.

## Affected Memory

- `_shared/configuration`: (modify) the config read-model / six-verb description of `show` — bare show now materializes the defaults tier; `--origin` described as provenance-only

## Impact

- `src/go/fab/cmd/fab/config.go` + `config_test.go` (+ any fixture pinning bare-show output)
- `src/kit/skills/_cli-fab.md`, `docs/specs/config.md`, `docs/memory/_shared/configuration.md`
- Pure query — no file writes, no migration. Output-consuming scripts that parsed bare `show` see additional keys (additive; YAML shape unchanged per key).

## Open Questions

- None — the direction was settled in the originating discussion.

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Bare `fab config show` prints the fully composed config, defaults tier included | User accepted this recommendation explicitly in discussion | S:90 R:85 A:90 D:85 |
| 2 | Certain | `--origin` semantics untouched — provenance annotator only | User's framing: "--origin is to track the source"; no behavior change requested there | S:90 R:90 A:90 D:90 |
| 3 | Confident | No sparse "only what I set" flag ships now | YAGNI — additive to introduce later; files + `--origin` filtering cover the need meanwhile | S:60 R:90 A:65 D:50 |
| 4 | Confident | Reuse the existing `DefaultsMapFor` projection for the composed tier (knob-composed derived rows included) | The keyed and `--origin` paths already use it; a second projection would drift | S:70 R:80 A:85 D:80 |
| 5 | Confident | Composed output is additive for consumers (new keys appear; existing keys keep shape) — no compatibility carve-out needed | Bare show is a human-facing pure query; `--json`-style stable surfaces are not touched | S:65 R:75 A:75 D:70 |

5 assumptions (2 certain, 3 confident, 0 tentative, 0 unresolved).
