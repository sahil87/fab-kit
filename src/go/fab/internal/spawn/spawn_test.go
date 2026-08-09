package spawn

import (
	"os"
	"path/filepath"
	"testing"
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
		// touch a preceding token (exercises the `n > 0` guard in resolveTemplate).
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
