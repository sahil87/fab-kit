package spawn

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/shellquote"
)

func TestCommand_WithInteractiveCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	// The `default` role's provider (claude by default) supplies the session command.
	os.WriteFile(configPath, []byte(`providers:
  claude:
    interactive_command: "custom-claude --model opus"
`), 0o644)

	got := Command(configPath)
	if got != "custom-claude --model opus" {
		t.Errorf("Command() = %q, want %q", got, "custom-claude --model opus")
	}
}

// TestCommand_CustomDefaultProvider: the `default` role can point at a non-claude
// provider; Command then reads THAT provider's session command.
func TestCommand_CustomDefaultProvider(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`providers:
  codex:
    interactive_command: "codex --tui"
agent:
  profiles:
    default: { provider: codex }
`), 0o644)

	got := Command(configPath)
	if got != "codex --tui" {
		t.Errorf("Command() = %q, want %q", got, "codex --tui")
	}
}

func TestCommand_EmptyInteractiveCommand(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`providers:
  claude:
    interactive_command: ""
`), 0o644)

	got := Command(configPath)
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}

func TestCommand_NoProvidersSection(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`project:
  name: "test"
`), 0o644)

	got := Command(configPath)
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}

func TestCommand_MissingFile(t *testing.T) {
	got := Command("/nonexistent/config.yaml")
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}

func TestCommand_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	os.WriteFile(configPath, []byte(`{{{invalid yaml`), 0o644)

	got := Command(configPath)
	if got != DefaultSpawnCommand {
		t.Errorf("Command() = %q, want %q", got, DefaultSpawnCommand)
	}
}

// TestWithProfile verifies the `doing`-role flag injection: both flags appended at
// the END in order model→effort (last-wins), each flag omitted entirely when its
// value is empty, and an all-empty profile leaving spawnCmd untouched.
func TestWithProfile(t *testing.T) {
	const base = "claude --dangerously-skip-permissions --effort xhigh"

	tests := []struct {
		name   string
		model  string
		effort string
		want   string
	}{
		{
			name:   "both present appended in order at end",
			model:  "claude-opus-4-8",
			effort: "high",
			want:   base + " --model claude-opus-4-8 --effort high",
		},
		{
			name:   "empty model only appends just effort",
			model:  "",
			effort: "high",
			want:   base + " --effort high",
		},
		{
			name:   "empty effort only appends just model",
			model:  "claude-opus-4-8",
			effort: "",
			want:   base + " --model claude-opus-4-8",
		},
		{
			name:   "both empty leaves spawnCmd unchanged",
			model:  "",
			effort: "",
			want:   base,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := WithProfile(base, tc.model, tc.effort)
			if got != tc.want {
				t.Errorf("WithProfile(%q, %q, %q) = %q, want %q", base, tc.model, tc.effort, got, tc.want)
			}
		})
	}
}

// TestWithProfile_Template verifies template mode: a spawnCmd containing
// {model}/{effort} is resolved by substitution (all-or-nothing — the append
// fallback is disabled), and an empty value drops the placeholder token plus a
// preceding `-`-prefixed flag token across every documented flag shape.
func TestWithProfile_Template(t *testing.T) {
	tests := []struct {
		name   string
		spawn  string
		model  string
		effort string
		want   string
	}{
		// Both placeholders substituted.
		{
			name:   "both placeholders substituted",
			spawn:  "codex -m {model} -c model_reasoning_effort={effort}",
			model:  "gpt-5",
			effort: "high",
			want:   "codex -m gpt-5 -c model_reasoning_effort=high",
		},
		// Single placeholder: the other resolved value is NOT appended.
		{
			name:   "single {model} placeholder, effort not appended",
			spawn:  "codex -m {model}",
			model:  "gpt-5",
			effort: "high",
			want:   "codex -m gpt-5",
		},
		{
			name:   "single {effort} placeholder, model not appended",
			spawn:  "codex -c model_reasoning_effort={effort}",
			model:  "gpt-5",
			effort: "high",
			want:   "codex -c model_reasoning_effort=high",
		},
		// Empty model, each token shape.
		{
			name:   "empty model drops -m and {model}",
			spawn:  "codex -m {model} -c model_reasoning_effort={effort}",
			model:  "",
			effort: "high",
			want:   "codex -c model_reasoning_effort=high",
		},
		{
			name:   "empty model drops --model and {model}",
			spawn:  "agent --model {model} --run",
			model:  "",
			effort: "",
			want:   "agent --run",
		},
		{
			name:   "empty model drops single --model={model} token, no preceding flag",
			spawn:  "agent --model={model} --run",
			model:  "",
			effort: "",
			want:   "agent --run",
		},
		// Empty effort, `-c key={effort}` shape drops the preceding -c.
		{
			name:   "empty effort drops model_reasoning_effort token and -c",
			spawn:  "codex -m {model} -c model_reasoning_effort={effort}",
			model:  "gpt-5",
			effort: "",
			want:   "codex -m gpt-5",
		},
		// Both empty.
		{
			name:   "both empty drops both flag pairs",
			spawn:  "codex -m {model} -c model_reasoning_effort={effort}",
			model:  "",
			effort: "",
			want:   "codex",
		},
		// Nested-shell dispatch grammar (260808-rpsr). agy and kimi take the
		// prompt as the ARGUMENT to -p and ignore stdin, so their built-in
		// headless_commands wrap the CLI in `sh -c '… -p "$(cat)"'`. Both the
		// substituted and the token-drop paths must leave that quoted tail intact —
		// it is what makes the piped prompt reach the worker at all.
		{
			name:   "nested-shell template substitutes without disturbing the quoted tail",
			spawn:  `sh -c 'agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'`,
			model:  "gemini-3.1-pro-high",
			effort: "high",
			want:   `sh -c 'agy --dangerously-skip-permissions --print-timeout 120m --model gemini-3.1-pro-high -p "$(cat)"'`,
		},
		{
			// kimi ships no fills, so this is its SHIPPED resolution, not an edge
			// case: the `-m {model}` pair drops as a pair and the interior quoted
			// segment survives, leaving a command kimi runs against the user's own
			// configured default_model.
			name:   "empty model drops the -m pair inside a nested shell, quoting intact",
			spawn:  `sh -c 'kimi -m {model} -p "$(cat)"'`,
			model:  "",
			effort: "",
			want:   `sh -c 'kimi -p "$(cat)"'`,
		},
		{
			// The drop must not reach back past the inner command into the outer
			// `sh -c` — eating that `-c` would destroy the invocation entirely.
			// Guarded here because `-m`'s preceding token IS the fused `'kimi`,
			// which does not begin with `-`, so the walk stops where it should.
			name:   "empty model drop does not reach the outer sh -c",
			spawn:  `sh -c 'kimi -m {model} --resume -p "$(cat)"'`,
			model:  "",
			effort: "",
			want:   `sh -c 'kimi --resume -p "$(cat)"'`,
		},
		// Multiple occurrences of one placeholder — all substituted.
		{
			name:   "multiple {model} occurrences all substituted",
			spawn:  "wrap {model} -- run --tag {model}",
			model:  "gpt-5",
			effort: "",
			want:   "wrap gpt-5 -- run --tag gpt-5",
		},
		// Placeholder embedded mid-word (no surrounding flag).
		{
			name:   "placeholder embedded mid-word substituted",
			spawn:  "agent --profile=pre-{model}-post",
			model:  "gpt-5",
			effort: "",
			want:   "agent --profile=pre-gpt-5-post",
		},
		// Empty profile on a template (the fab spawn-command leak-prevention path).
		{
			name:   "empty profile strips a fully-templated command",
			spawn:  "codex -m {model} -c model_reasoning_effort={effort}",
			model:  "",
			effort: "",
			want:   "codex",
		},
		// All-non-empty substitution preserves the raw string's whitespace runs
		// (no tokenize/rejoin — the plain-ReplaceAll path).
		{
			name:   "non-empty values preserve multi-space and tab whitespace",
			spawn:  "codex  -m  {model}\t-c model_reasoning_effort={effort}",
			model:  "gpt-5",
			effort: "high",
			want:   "codex  -m  gpt-5\t-c model_reasoning_effort=high",
		},
		// Placeholder as the FIRST token with an empty value: the drop must not
		// touch a preceding token (exercises the `n > 0` guard in substitute).
		{
			name:   "empty value on first-token placeholder, no preceding token to drop",
			spawn:  "{model} run",
			model:  "",
			effort: "",
			want:   "run",
		},
		// A single token carrying BOTH placeholders, substituted together.
		{
			name:   "single token carries both placeholders",
			spawn:  "agent --profile={model}-{effort}",
			model:  "gpt-5",
			effort: "high",
			want:   "agent --profile=gpt-5-high",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := WithProfile(tc.spawn, tc.model, tc.effort)
			if got != tc.want {
				t.Errorf("WithProfile(%q, %q, %q) = %q, want %q", tc.spawn, tc.model, tc.effort, got, tc.want)
			}
		})
	}
}

// TestHasPrompt pins the predicate DeliverPrompt branches on: only the {prompt}
// token counts, never the profile placeholders.
func TestHasPrompt(t *testing.T) {
	tests := []struct {
		name  string
		spawn string
		want  bool
	}{
		{"agy built-in shape", "agy --dangerously-skip-permissions --model {model} -i {prompt}", true},
		{"claude built-in shape (profile placeholders only)", `claude --dangerously-skip-permissions -n "x" --model {model} --effort {effort}`, false},
		{"plain command", "codex --tui", false},
		{"prompt-only command", "mycli -i {prompt}", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := HasPrompt(tc.spawn); got != tc.want {
				t.Errorf("HasPrompt(%q) = %v, want %v", tc.spawn, got, tc.want)
			}
		})
	}
}

// TestWithPrompt verifies the {prompt} pass: substitution preserves the raw
// string (no tokenize/rejoin), an EMPTY value drops the placeholder token plus a
// preceding `-`-flag under the same grammar {model}/{effort} use, and a command
// carrying no {prompt} is returned untouched whatever the value.
func TestWithPrompt(t *testing.T) {
	const agy = "agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i {prompt}"

	tests := []struct {
		name   string
		spawn  string
		prompt string
		want   string
	}{
		{
			// The shipped prompt-carrying resolution: the prompt arrives already
			// shell-quoted from the shared delivery helper, and lands as -i's value.
			name:   "agy prompt-carrying seam substitutes the quoted prompt at the placeholder",
			spawn:  agy,
			prompt: `'Read .fab-dispatch/agik/apply-prompt.md and execute it.'`,
			want:   `agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i 'Read .fab-dispatch/agik/apply-prompt.md and execute it.'`,
		},
		{
			// The shipped promptless resolution — bare `fab agent`, the one seam
			// that composes an interactive_command with no initial prompt.
			name:   "agy promptless seam drops the -i {prompt} pair",
			spawn:  agy,
			prompt: "",
			want:   "agy --dangerously-skip-permissions --model gemini-3.1-pro-high",
		},
		{
			name:   "placeholder fused into a --flag= token drops alone",
			spawn:  "mycli --initial-prompt={prompt} --run",
			prompt: "",
			want:   "mycli --run",
		},
		{
			name:   "interior placeholder drop leaves the tail intact",
			spawn:  "mycli -i {prompt} --run",
			prompt: "",
			want:   "mycli --run",
		},
		// No-placeholder commands are untouched on BOTH values — this is what keeps
		// claude/codex byte-identical through the session seams, whitespace and
		// nested-shell quoting included (the fast path never tokenizes).
		{
			name:   "no placeholder, empty value: claude built-in untouched",
			spawn:  `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5 --effort high`,
			prompt: "",
			want:   `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5 --effort high`,
		},
		{
			name:   "no placeholder, non-empty value: nested-shell command untouched",
			spawn:  `sh -c 'kimi -p "$(cat)"'`,
			prompt: "'ignored'",
			want:   `sh -c 'kimi -p "$(cat)"'`,
		},
		{
			name:   "multi-space whitespace preserved on the substitution path",
			spawn:  "mycli  -i  {prompt}",
			prompt: "'p'",
			want:   "mycli  -i  'p'",
		},
		{
			name:   "first-token placeholder with empty value has no preceding token to drop",
			spawn:  "{prompt} run",
			prompt: "",
			want:   "run",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := WithPrompt(tc.spawn, tc.prompt); got != tc.want {
				t.Errorf("WithPrompt(%q, %q) = %q, want %q", tc.spawn, tc.prompt, got, tc.want)
			}
		})
	}
}

// TestDeliverPrompt pins the shared two-shape delivery grammar every
// prompt-carrying seam runs through — pane dispatch, the operator launcher and
// both `fab batch` worker spawns. A {prompt}-carrying command receives the
// shell-quoted prompt AT the placeholder; a placeholder-free one receives it as
// an appended positional, byte-identical to the pre-placeholder form.
func TestDeliverPrompt(t *testing.T) {
	const (
		agy    = "agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i {prompt}"
		claude = `claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model claude-fable-5`
	)

	tests := []struct {
		name   string
		spawn  string
		prompt string
		want   string
	}{
		{
			name:   "placeholder shape: prompt lands as the flag's value",
			spawn:  agy,
			prompt: "/fab-operator",
			want:   "agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i '/fab-operator'",
		},
		{
			name:   "positional shape: prompt is appended, quoted",
			spawn:  claude,
			prompt: "/fab-operator",
			want:   claude + " '/fab-operator'",
		},
		{
			// Quote safety, placeholder path: a backlog item's text is user-derived
			// and may carry an apostrophe. It must not terminate the -i argument.
			name:   "placeholder shape escapes an embedded single quote",
			spawn:  agy,
			prompt: "/fab-new sahil's item",
			want:   `agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i '/fab-new sahil'\''s item'`,
		},
		{
			name:   "positional shape escapes an embedded single quote",
			spawn:  claude,
			prompt: "/fab-new sahil's item",
			want:   claude + ` '/fab-new sahil'\''s item'`,
		},
		{
			// The positional path must not tokenize: nested-shell quoting and
			// multi-space runs in the provider's own command survive verbatim.
			name:   "positional shape leaves the command verbatim",
			spawn:  `sh -c 'kimi  -p "$(cat)"'`,
			prompt: "/fab-switch abcd",
			want:   `sh -c 'kimi  -p "$(cat)"' '/fab-switch abcd'`,
		},
		{
			// A prompt whose text happens to contain a profile placeholder is
			// substituted in verbatim — the prompt pass runs last and nothing
			// re-resolves it.
			name:   "prompt text containing {model} is not re-substituted",
			spawn:  agy,
			prompt: "/fab-new document the {model} placeholder",
			want:   "agy --dangerously-skip-permissions --model gemini-3.1-pro-high -i '/fab-new document the {model} placeholder'",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := DeliverPrompt(tc.spawn, tc.prompt); got != tc.want {
				t.Errorf("DeliverPrompt(%q, %q) = %q, want %q", tc.spawn, tc.prompt, got, tc.want)
			}
		})
	}
}

// TestDeliverPromptMatchesLegacyPositionalForm is the back-compat pin: for every
// placeholder-free command, DeliverPrompt must produce exactly what the seams
// composed before the placeholder existed — `<cmd> <shell-quoted prompt>`.
func TestDeliverPromptMatchesLegacyPositionalForm(t *testing.T) {
	commands := []string{
		DefaultSpawnCommand,
		"codex --tui",
		`sh -c 'kimi -p "$(cat)"'`,
	}
	prompts := []string{"/fab-operator", "/fab-new add a thing", "/fab-switch sahil's-change"}

	for _, cmd := range commands {
		for _, prompt := range prompts {
			want := cmd + " " + shellquote.Single(prompt)
			if got := DeliverPrompt(cmd, prompt); got != want {
				t.Errorf("DeliverPrompt(%q, %q) = %q, want the byte-identical legacy form %q", cmd, prompt, got, want)
			}
		}
	}
}

// TestPromptDoesNotAffectProfileMode is the mode-interaction guard: {prompt} is
// deliberately absent from isTemplate, so a command carrying ONLY {prompt} must
// still take WithProfile's APPEND mode and receive --model/--effort. Folding
// {prompt} into the template test would silently suppress that append.
func TestPromptDoesNotAffectProfileMode(t *testing.T) {
	const promptOnly = "mycli --flag -i {prompt}"

	got := WithProfile(promptOnly, "m1", "high")
	want := "mycli --flag -i {prompt} --model m1 --effort high"
	if got != want {
		t.Errorf("WithProfile(%q, ...) = %q, want %q — {prompt} must not flip append mode", promptOnly, got, want)
	}

	// And the composed pipeline the call sites run: profile first, prompt last.
	if got := WithPrompt(got, ""); got != "mycli --flag --model m1 --effort high" {
		t.Errorf("session pipeline = %q, want %q", got, "mycli --flag --model m1 --effort high")
	}

	// A command carrying BOTH kinds still takes template mode, and the {prompt}
	// token survives WithProfile untouched for the later pass to resolve.
	const both = "agy --model {model} -i {prompt}"
	if got := WithProfile(both, "gemini-3.1-pro-high", "high"); got != "agy --model gemini-3.1-pro-high -i {prompt}" {
		t.Errorf("WithProfile(%q, ...) = %q, want the {prompt} token preserved", both, got)
	}
	// The empty-model drop path must likewise preserve it.
	if got := WithProfile(both, "", ""); got != "agy -i {prompt}" {
		t.Errorf("WithProfile(%q, empty) = %q, want %q", both, got, "agy -i {prompt}")
	}
}
