# Intake: Kimi Pane Enablement — Box-Drawing-Tolerant Squeeze + Ship interactive_command

**Change**: 260810-ki9v-kimi-pane-enablement
**Created**: 2026-08-10

## Origin

Conversational (`/fab-discuss` session 2026-08-09/10). The user asked:

> For #2, I want you to test your assumptions about kimi — whether you are able to use it interactively.

A live probe was run (kimi 0.34.0, raw tmux pane in the solar-numbat worktree, `fab pane send --force` + manual capture/Enter choreography, no dispatch record) and **root-caused the known deliver-verification failure** first recorded in the wll4 probe (2026-08-09). The user then approved drafting this change, sequenced **first** of three (before `260810-ug8b-config-reference-legibility` in parallel and `260810-1lah-provider-generic-pane-verbs` which is gated on this change merging).

The change ID `ki9v` is deliberately pinned: the agik change (PR #564, unmerged as of drafting) recorded "kimi flip = [ki9v]" as the designated backlog item; reusing the ID preserves archive-time backlog linkage once that branch merges.

## Why

1. **Pain point**: kimi is a built-in provider but dispatch-only. Interactive/pane use *works* — verified live end-to-end (trust wall Enter-cleared, prompt typed, correct `KIMI-PANE-OK` response) — but `fab dispatch deliver`'s echo-verification structurally cannot pass against kimi's TUI, so kimi ships no `interactive_command` and pane dispatch is closed to it.
2. **Root cause (now precise)**: the post-3oz7 verifier `countWrapped`/`squeeze` (`src/go/fab/internal/dispatch/gate.go:337`) normalizes by dropping **whitespace only**. Kimi's input box draws **vertical side borders** (`│`); a line that wraps inside the box interleaves `││` between the wrapped halves:

   ```
   │ > Reply with exactly KIMI-PANE-OK and nothing else. Do not use      │
   │   any tools.                                                        │
   ```

   so the squeezed capture contains `…Donotuse││anytools…` and the squeezed needle never matches. This is the exact failure mode `countWrapped`'s own comment predicted for an unprobed boxed TUI ("A TUI that boxed its input line with vertical rules would interleave frame runes and read 0; kimi is unprobed"). Kimi is now probed; the prediction is confirmed. claude and agy draw borderless input boxes (rules only, no side borders) and pass today's whitespace-only squeeze.
3. **Consequence of not fixing**: kimi pane workers stay unreachable; the pane adapter's provider roster stays claude-only in practice (agy's flip is PR #564's territory); the failure mode remains a loud gate escalation on anyone who opts kimi in by hand (as the user's system config already does).
4. **Why this approach**: extending `squeeze` is a one-function, verifier-local fix at the root cause. Alternatives rejected: (a) verify submission instead of echo — bigger contract change, loses the pre-Enter safety check; (b) manual per-provider delivery choreography (the wll4 workaround: probe + BSpace + Enter) — provider-specific, unautomatable; (c) leave kimi headless-only — forfeits pane capability that demonstrably works.

## What Changes

### 1. Box-drawing-tolerant `squeeze` (`src/go/fab/internal/dispatch/gate.go`)

Extend `squeeze` to drop **box-drawing runes (U+2500–U+257F)** in addition to whitespace:

```go
func squeeze(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsSpace(r) || (r >= 0x2500 && r <= 0x257F) {
			return -1
		}
		return r
	}, s)
}
```

This makes `countWrapped` tolerate side-bordered input boxes for both of its call sites (the `ready` sentinel echo at `gate.go:185` and the deliver pointer echo at `gate.go:334`). Safety: the needle (a `ReadySentinel` or a prompt-file pointer line) never legitimately contains box-drawing runes, and false-positive risk from dropping them in the haystack is negligible — the failure mode of a wrong answer remains a loud double failure into the gate's escalation, never a false success (existing design property, preserved).

Update `countWrapped`'s comment block: kimi is no longer "unprobed" — record the probe result (side borders, `││` interleave, 2026-08-10, kimi 0.34.0) and that the box-rune drop is what admits boxed TUIs. Add a test case with a captured kimi-style boxed wrap (needle split across two `│`-bordered lines) alongside the existing claude/agy-shaped cases.

### 2. Ship kimi's `interactive_command` built-in default (`src/go/fab/internal/agent/defaults.yaml`)

```yaml
kimi:
  interactive_command: kimi --auto -m {model}
```

- Mirrors the user's proven system-config opt-in (`fab config set --system providers.kimi.interactive_command 'kimi --auto -m {model}'`, in production since the wll4 pipeline ran kimi pane workers with manual recovery). Kimi ships no fills, so `{model}` resolves empty and the token-drop rule removes `-m` — kimi falls back to the user's own `default_model` (existing, deliberate behavior).
- **No `{prompt}` placeholder**: kimi has no interactive-initial-prompt flag (bare positional parses as a subcommand — rpsr probe). Kimi's pane arm is pure post-open delivery: `open` → readiness gate (the first-run "Trust this folder?" wall is `parked`, one Enter clears it, remembered per folder — a standard judgment-round answer, no code needed) → `deliver` with the now-tolerant verify.
- Flip `defaults_test.go`'s roster assertions: the test currently asserts kimi ships **no** `interactive_command` (`defaults_test.go:147–163`, citing backlog [agik]) precisely so an unprobed command couldn't select pane mode and park stages. The probe gap is closed; the assertion inverts for kimi (present, exact value) while remaining absent-asserting for agy *if PR #564 has not merged first* (see Impact).

### 3. Prose/roster sweep — "dispatch-only" phrasings

Kimi's capability change invalidates roster claims across the sweep class (the rpsr lesson: roster-count phrasings escape keyword greps — sweep deliberately):

- `configref.go` `providersSegment` prose: "agy and kimi ship NO interactive_command — they are DISPATCH-ONLY built-ins…" and the kimi per-provider note ("No interactive_command either — its interactive first-run and input echo have not been probed against the pane-delivery choreography") — both now false for kimi; rewrite to reflect the shipped command and the probe result. The kimi block in the rendered reference gains its `interactive_command` line.
- `src/kit/skills/_cli-agents.md` grammar dictionary (kimi row: interactive form + first-run wall note).
- `docs/specs/stage-models.md` / `docs/specs/harness-adapters.md` if they carry the dispatch-only roster claim.
- Rendered-fence/byte-stable fixtures in `internal/configupgrade` and `cmd/fab` tests (the reference now renders one more kimi line).

## Affected Memory

- `runtime/dispatch.md`: (modify) deliver/ready verification — record the box-drawing-tolerant squeeze and the boxed-TUI class it admits.
- `runtime/providers-and-profiles.md`: (modify) kimi capability roster — interactive_command shipped, pane-capable.
- `runtime/agent-primitives.md`: (modify) grammar dictionary — kimi interactive form and first-run trust wall.
- `_shared/configuration.md`: (modify) provider capability grammar examples if they enumerate per-provider capabilities.

## Impact

- `src/go/fab/internal/dispatch/gate.go` + `gate_test.go` — the squeeze fix and new wrap cases.
- `src/go/fab/internal/agent/defaults.yaml` + `defaults_test.go` — kimi capability flip.
- `src/go/fab/internal/configref/configref.go` + fence fixtures — reference prose and kimi block rendering.
- `src/kit/skills/_cli-agents.md` + SPEC mirrors per the sibling-sweep class; specs carrying roster claims.
- **Coordination, PR #564 (agik — agy interactive_command + `{prompt}` placeholder + `spawn.DeliverPrompt`)**: not merged in this tree. Both changes edit `defaults.yaml`, `defaults_test.go`, and the same configref roster prose. Whichever lands second rebases; if #564 merges first, this change's sweep must preserve agy's then-current capability lines and the `{prompt}` machinery (kimi's command uses no `{prompt}` either way).
- **Coordination, `260810-ug8b-config-reference-legibility`**: runs in parallel; both regenerate the rendered-reference fixtures — mechanical rebase for whichever lands second.
- **Sequencing**: `260810-1lah-provider-generic-pane-verbs` is gated on this change merging (it relocates the gate code this change edits).

## Open Questions

- (none)

## Assumptions

| # | Grade | Decision | Rationale | Scores |
|---|-------|----------|-----------|--------|
| 1 | Certain | Root cause is the whitespace-only squeeze vs kimi's `│` side borders | Verified live 2026-08-10 (kimi 0.34.0): wrap interleaves `││`; matches gate.go's own predicted failure mode | S:95 R:85 A:95 D:95 |
| 2 | Certain | Fix = extend squeeze to drop U+2500–U+257F box-drawing runes | Discussed and accepted; verifier-local, needle never contains box runes, loud-failure property preserved | S:90 R:85 A:90 D:90 |
| 3 | Certain | Ship `kimi --auto -m {model}` as the built-in interactive_command | Exact value proven in the user's system config through a real pipeline (wll4); empty-model token-drop is existing behavior | S:90 R:80 A:90 D:90 |
| 4 | Certain | Trust wall needs no code — readiness gate's judgment rounds already handle it | Probed: `parked` → one Enter → ready, remembered per folder; exactly the gate's designed wall class | S:80 R:75 A:85 D:85 |
| 5 | Confident | Range-based rune drop (U+2500–U+257F), not a broader "all non-alphanumerics" normalization | Narrowest change that admits boxed TUIs; broader drops raise false-positive surface without a probed need | S:65 R:80 A:80 D:70 |
| 6 | Confident | Rebase posture vs PR #564 rather than folding agy work in | agik is its own shipped pipeline awaiting merge; this change touches the kimi half only | S:70 R:75 A:80 D:75 |
| 7 | Confident | `defaults_test.go` keeps asserting agy's interactive_command ABSENT (pre-#564 baseline in this tree) | True in this tree today; flips to present if #564 merges first — apply re-checks at branch time | S:55 R:85 A:70 D:60 |

7 assumptions (4 certain, 3 confident, 0 tentative, 0 unresolved).
