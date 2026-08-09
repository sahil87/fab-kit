package spawn

import (
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/shellquote"
)

// DefaultSpawnCommand is the fallback interactive command Command returns when
// the default role's provider resolves no interactive_command (or config.yaml
// cannot be read/parsed) — the fallback keys on the RESOLVED provider, which the
// agent.session knob selects, not on the claude entry specifically. Its value is
// the built-in claude provider's interactive command, re-exported from
// internal/agent (the provider table's owner) so raw-consumer sites keep a
// single spelling. Like the
// underlying value it is a {model}/{effort} TEMPLATE — callers resolve it
// through WithProfile (template mode), which yields the same byte-identical
// command the former plain form produced via append mode.
//
// A var, not a const, because the string it re-exports is now parsed from
// internal/agent's embedded defaults.yaml rather than written as a Go literal.
var DefaultSpawnCommand = agent.DefaultInteractiveCommand

// Command reads the default provider's interactive command from the given
// config.yaml path via the shared internal/config loader (the single config.yaml
// parser).
// Returns providers.<default-role.provider>.interactive_command resolved over
// fab-kit's built-in provider table, or DefaultSpawnCommand if it resolves empty
// or the file cannot be read/parsed. The path-based signature is kept because
// `fab agent --repo <path>` builds the path from an arbitrary repo root.
func Command(configPath string) string {
	cfg, err := config.LoadPath(configPath)
	if err != nil {
		return DefaultSpawnCommand
	}

	// The interactive command lives on the default role's provider (which the
	// agent.session knob selects). Resolve the role to find which provider, then
	// that provider's interactive command.
	profile, err := agent.ResolveRole(cfg, agent.RoleDefault)
	if err != nil {
		return DefaultSpawnCommand
	}
	if prov, ok := agent.ResolveProvider(cfg, profile.Provider); ok {
		if prov.InteractiveCommand != "" {
			return prov.InteractiveCommand
		}
	}
	return DefaultSpawnCommand
}

// Placeholder tokens recognized in a templated spawn_command. The first two are
// the PROFILE placeholders: their presence (either one) switches WithProfile from
// append mode to template mode.
//
// promptPlaceholder is deliberately NOT one of them. It marks where a pane
// worker's pointer prompt is delivered, which is resolved by WithPrompt in a
// separate pass — see that function for why the mode decision must not key on it.
const (
	modelPlaceholder  = "{model}"
	effortPlaceholder = "{effort}"
	promptPlaceholder = "{prompt}"
)

// WithProfile injects the resolved model/effort into spawnCmd. It operates in
// one of two modes, selected by whether spawnCmd contains a PROFILE placeholder
// ({model}/{effort}; {prompt} deliberately does not participate — see WithPrompt):
//
//   - Template mode (spawnCmd contains "{model}" or "{effort}"): substitute
//     every occurrence of each placeholder with the resolved value. Template
//     mode is all-or-nothing — the presence of ANY placeholder disables the
//     append below entirely, so a value whose placeholder is absent from the
//     template is simply not injected (this prevents e.g. a Claude --effort
//     flag being appended to a codex command that only templated {model}).
//     Provider grammar therefore lives in the user's config, consistent with
//     the resolver's verbatim/no-validation philosophy.
//   - Append mode (no placeholder): today's behavior, byte-for-byte. Append
//     --model/--effort to the END of spawnCmd (last-wins), omitting each flag
//     when its value is empty; model before effort. Appending last is
//     deliberate: the configured spawn_command may already pin a --model/
//     --effort, and a trailing occurrence wins on the claude CLI (duplicate
//     --effort is accepted without a parse error), so the caller's deliberate
//     role choice overrides whatever the spawn_command defaulted to.
//
// An empty value mirrors the documented `empty ⇒ omit` convention (_preamble.md
// § Per-Stage Model Resolution): in append mode it omits the flag entirely; in
// template mode it triggers the empty-value token-drop rule (see resolveTemplate).
func WithProfile(spawnCmd, model, effort string) string {
	if isTemplate(spawnCmd) {
		return resolveTemplate(spawnCmd, model, effort)
	}

	var b strings.Builder
	b.WriteString(spawnCmd)
	if model != "" {
		b.WriteString(" --model ")
		b.WriteString(model)
	}
	if effort != "" {
		b.WriteString(" --effort ")
		b.WriteString(effort)
	}
	return b.String()
}

// isTemplate reports whether spawnCmd contains at least one PROFILE placeholder,
// which switches WithProfile into template mode. {prompt} is deliberately not
// consulted: a command carrying only `-i {prompt}` must still take append mode so
// its --model/--effort are appended (see WithPrompt).
func isTemplate(spawnCmd string) bool {
	return strings.Contains(spawnCmd, modelPlaceholder) ||
		strings.Contains(spawnCmd, effortPlaceholder)
}

// HasPrompt reports whether spawnCmd carries the {prompt} placeholder — the
// signal DeliverPrompt branches on to choose between substituting the prompt at
// the placeholder and appending it as a positional argument.
func HasPrompt(spawnCmd string) bool {
	return strings.Contains(spawnCmd, promptPlaceholder)
}

// WithPrompt resolves the {prompt} placeholder — the delivery point for an
// agent's initial prompt — under the same substitute-or-drop contract
// {model}/{effort} have (see substitute). It is a NO-OP on a command carrying no
// {prompt}, whatever the value. The value is substituted VERBATIM: a caller
// delivering user-derived text quotes it first (DeliverPrompt is that caller).
//
// Two kinds of caller reach it: DeliverPrompt, for the seams that HAVE initial
// content, and the one promptless session seam (bare `fab agent`), which passes
// "" to drop the placeholder and its flag.
//
// It is a SEPARATE pass rather than a third WithProfile parameter, and callers
// run it AFTER WithProfile, for three reasons:
//
//   - The lifetimes differ. Model and effort are known at profile resolution;
//     a pane worker's pointer only exists once `fab dispatch start` has persisted
//     the stage prompt file, and the promptless seam never has one at all.
//   - The MODE decision must not key on {prompt}. isTemplate covers the profile
//     placeholders only, so a command carrying just `-i {prompt}` still takes
//     WithProfile's append mode and receives its --model/--effort — folding
//     {prompt} into isTemplate would silently suppress that append.
//   - Running last means a pointer whose text happens to contain `{model}` is
//     never re-substituted.
//
// One grammar therefore serves both kinds of launch: agy's built-in
// `agy … --model {model} -i {prompt}` becomes a plain session at the promptless
// launch (the `-i {prompt}` pair token-drops) and carries the shell-quoted prompt
// at every prompt-carrying one. agy's `-i`/`--prompt-interactive` is a
// VALUE-TAKING flag, so a bare trailing `-i` would hard-error the promptless
// launch — the placeholder is what lets one command field serve both.
func WithPrompt(spawnCmd, prompt string) string {
	return substitute(spawnCmd, placeholderSub{promptPlaceholder, prompt})
}

// DeliverPrompt composes spawnCmd with an initial prompt, in whichever of the
// two delivery shapes spawnCmd declares. It is the SINGLE implementation of the
// delivery grammar: every seam that launches an agent WITH initial content —
// pane dispatch (internal/dispatch.WindowCommand), the operator launcher, and
// both `fab batch` worker spawns — goes through it, so the grammar cannot drift
// between them.
//
//   - PLACEHOLDER (HasPrompt): the shell-quoted prompt is substituted AT the
//     {prompt} placeholder. This is what a value-taking initial-prompt flag
//     requires — the agy built-in's `-i {prompt}` (`--prompt-interactive` takes a
//     value, so a bare positional never reaches it and is silently discarded).
//   - POSITIONAL (no placeholder): the shell-quoted prompt is APPENDED as the
//     command's final argument. This is the original and still-default shape,
//     kept byte-for-byte for every provider that declares no placeholder
//     (claude, codex, and any user command written before placeholders existed).
//
// The prompt is shell-quoted on BOTH shapes: it carries user-derived text (a
// backlog item's content, a change name, a repo-derived pointer path), and an
// embedded single quote would otherwise terminate the argument early and let the
// remainder be interpreted by the spawning shell.
//
// The empty-prompt case belongs to WithPrompt, not here: a seam with NO initial
// content (bare `fab agent`) must DROP the placeholder pair rather than deliver
// an empty argument, so it calls WithPrompt(cmd, "") directly.
func DeliverPrompt(spawnCmd, prompt string) string {
	quoted := shellquote.Single(prompt)
	if HasPrompt(spawnCmd) {
		return WithPrompt(spawnCmd, quoted)
	}
	return spawnCmd + " " + quoted
}

// resolveTemplate substitutes {model}/{effort} in a templated spawnCmd.
func resolveTemplate(spawnCmd, model, effort string) string {
	return substitute(spawnCmd,
		placeholderSub{modelPlaceholder, model},
		placeholderSub{effortPlaceholder, effort})
}

// placeholderSub pairs a placeholder token with the value substituted for it.
type placeholderSub struct {
	token string
	value string
}

// substitute resolves one or more placeholders in a command string. It is the
// single implementation of the substitute-or-drop grammar every placeholder
// shares — resolveTemplate passes {model}/{effort}, WithPrompt passes {prompt}.
//
// The two paths are structurally distinct:
//
//   - When EVERY substituted value is non-empty, substitution is a plain
//     strings.ReplaceAll over the RAW command string — the author's whitespace
//     (multi-space runs, tabs) is preserved exactly, because non-empty
//     substitution needs no token surgery.
//
//   - When at least one substituted value is EMPTY (the "inherit/omit" signal),
//     the command is tokenized on whitespace so a dangling flag can be dropped:
//     rather than leave e.g. `-m` or `model_reasoning_effort=`, we drop the
//     whitespace-delimited token containing the empty placeholder AND the
//     immediately preceding token when it begins with `-`. Surviving tokens are
//     rejoined with a single space (so whitespace-run preservation applies only
//     to the all-non-empty path above). This cleanly handles the common flag
//     shapes:
//
//     -m {model}                         → both tokens dropped
//     --model {model}                    → both tokens dropped
//     --model={model}                    → single token dropped (no preceding -flag)
//     -c model_reasoning_effort={effort} → the `...={effort}` token and `-c` dropped
//
// Grammar limits: the token-drop rule is quote-blind and covers only the four
// flag shapes above. A placeholder inside quotes (e.g. `"{model}"`), or one
// preceded by a valueless flag that begins with `-` (e.g. `--verbose {model}`
// with an empty model, or the argument separator `-- {model}`), is OUTSIDE the
// supported grammar — the empty-value drop would remove the wrong preceding
// token. Templated spawn_commands are expected to use plain value-carrying
// flags (`-m`, `--model`, `--model=`, `-c key=`).
//
// Quote-blindness has one consequence worth naming, because two built-ins now
// nest a shell (`sh -c 'kimi -m {model} -p "$(cat)"'`): a droppable placeholder
// must not be the LAST token of such a command, or the drop takes the closing
// quote with it. Both nested-shell built-ins put `-p "$(cat)"` after the
// placeholder, so the drop is always INTERIOR and the quoting survives — that
// ordering is load-bearing, not incidental. agy's interactive command ends on the
// droppable `{prompt}`, but it nests no shell, so the hazard does not reach it.
func substitute(spawnCmd string, subs ...placeholderSub) string {
	// Whitespace-preserving fast path: taken when no placeholder that ACTUALLY
	// APPEARS in spawnCmd would substitute an empty value. Gating on the present
	// placeholders (not merely "every value is non-empty") means a command
	// carrying only {model} with a non-empty model still preserves its raw
	// whitespace even when effort is empty — the absent {effort} needs no
	// token-drop, so tokenizing (which collapses whitespace runs) is unwarranted.
	// It is also what makes WithPrompt a true no-op on the {prompt}-free
	// built-ins, whose nested-shell quoting must survive untokenized.
	needsDrop := false
	for _, s := range subs {
		if s.value == "" && strings.Contains(spawnCmd, s.token) {
			needsDrop = true
			break
		}
	}
	if !needsDrop {
		out := spawnCmd
		for _, s := range subs {
			out = strings.ReplaceAll(out, s.token, s.value)
		}
		return out
	}

	// Empty-value path: tokenize so a dangling flag can be dropped.
	tokens := strings.Fields(spawnCmd)
	out := make([]string, 0, len(tokens))

	for _, tok := range tokens {
		// A token may carry several placeholder flavors (e.g.
		// `--profile={model}-{effort}`), and the substitution branch resolves
		// every occurrence of each; a token with no placeholder is kept verbatim.
		// Token-drop fires only when a placeholder present in the token has an
		// empty substitution value.
		drop := false
		for _, s := range subs {
			if s.value == "" && strings.Contains(tok, s.token) {
				drop = true
				break
			}
		}
		if drop {
			// Empty-value drop: drop this token, plus a preceding `-`-flag token.
			if n := len(out); n > 0 && strings.HasPrefix(out[n-1], "-") {
				out = out[:n-1]
			}
			continue
		}
		// Substitute every occurrence of each placeholder (non-empty values;
		// an empty value here means the token has no placeholder of that kind).
		for _, s := range subs {
			tok = strings.ReplaceAll(tok, s.token, s.value)
		}
		out = append(out, tok)
	}
	return strings.Join(out, " ")
}
