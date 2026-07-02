package spawn

import (
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
)

// DefaultSpawnCommand is the fallback when config.yaml has no agent.spawn_command.
const DefaultSpawnCommand = "claude --dangerously-skip-permissions"

// Command reads agent.spawn_command from the given config.yaml path via the
// shared internal/config loader (the single config.yaml parser). Returns the
// configured command, or DefaultSpawnCommand if the key is missing, empty, or
// the file cannot be read/parsed. The path-based signature is kept because
// `fab spawn-command --repo <path>` builds the path from an arbitrary repo
// root.
func Command(configPath string) string {
	cfg, err := config.LoadPath(configPath)
	if err != nil {
		return DefaultSpawnCommand
	}

	if cmd := cfg.GetSpawnCommand(); cmd != "" {
		return cmd
	}
	return DefaultSpawnCommand
}

// Placeholder tokens recognized in a templated spawn_command. Their presence
// (either one) switches WithProfile from append mode to template mode.
const (
	modelPlaceholder  = "{model}"
	effortPlaceholder = "{effort}"
)

// WithProfile injects the resolved model/effort into spawnCmd. It operates in
// one of two modes, selected by whether spawnCmd contains a placeholder:
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
//     tier choice overrides whatever the spawn_command defaulted to.
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

// StripPlaceholders resolves a templated cmd with an EMPTY profile, degrading a
// template to a clean invocation (placeholders and their flag tokens stripped
// per the empty-value rule in resolveTemplate); a non-templated command is
// returned verbatim. It is a thin named wrapper over WithProfile(cmd, "", "")
// that states the intent at raw-consumer sites — code paths that interpolate
// the configured agent.spawn_command into a shell command WITHOUT a resolved
// profile (fab spawn-command, fab batch new, fab batch switch). Those sites
// must not leak literal {model}/{effort} braces into the shell.
func StripPlaceholders(cmd string) string {
	return WithProfile(cmd, "", "")
}

// isTemplate reports whether spawnCmd contains at least one placeholder, which
// switches WithProfile (and fab spawn-command) into template mode.
func isTemplate(spawnCmd string) bool {
	return strings.Contains(spawnCmd, modelPlaceholder) ||
		strings.Contains(spawnCmd, effortPlaceholder)
}

// resolveTemplate substitutes {model}/{effort} in a templated spawnCmd.
//
// The two paths are structurally distinct:
//
//   - When BOTH substituted values are non-empty, substitution is a plain
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
func resolveTemplate(spawnCmd, model, effort string) string {
	// All-non-empty path: preserve the raw string's whitespace exactly.
	if model != "" && effort != "" {
		out := strings.ReplaceAll(spawnCmd, modelPlaceholder, model)
		return strings.ReplaceAll(out, effortPlaceholder, effort)
	}

	// Empty-value path: tokenize so a dangling flag can be dropped.
	tokens := strings.Fields(spawnCmd)
	out := make([]string, 0, len(tokens))

	for _, tok := range tokens {
		// A token carries at most one flavor of placeholder in the documented
		// grammar; a token with no placeholder is kept verbatim.
		switch {
		case strings.Contains(tok, modelPlaceholder) && model == "",
			strings.Contains(tok, effortPlaceholder) && effort == "":
			// Empty-value drop: drop this token, plus a preceding `-`-flag token.
			if n := len(out); n > 0 && strings.HasPrefix(out[n-1], "-") {
				out = out[:n-1]
			}
		default:
			// Substitute every occurrence of each placeholder (non-empty values;
			// an empty value here means the token has no placeholder of that kind).
			tok = strings.ReplaceAll(tok, modelPlaceholder, model)
			tok = strings.ReplaceAll(tok, effortPlaceholder, effort)
			out = append(out, tok)
		}
	}
	return strings.Join(out, " ")
}
