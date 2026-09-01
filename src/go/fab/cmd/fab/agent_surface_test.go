package main

import (
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"gopkg.in/yaml.v3"
)

// assertFlagUsageError checks that a usage error (cobra's flag-handling
// failure) names the wanted phrase.
func assertFlagUsageError(t *testing.T, gotOut string, err error, want ...string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected a usage error, got %q output", gotOut)
	}
	for _, phrase := range want {
		if !strings.Contains(err.Error(), phrase) {
			t.Errorf("usage error %q should contain %q", err.Error(), phrase)
		}
	}
}

// runAgentExec captures agentCmd's full invocation (including `-t`/`-o`) with
// its own error surface. T008's exec-mode use is a usage error, so it never
// reaches the exec path.
func runAgentExec(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := agentCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

// runAgentYAML is runAgentExec's counterpart for -o — executes `-o yaml` (or
// another format) with its own usage-error surface asserted by the caller.
func runAgentYAML(t *testing.T, args ...string) (string, error) {
	t.Helper()
	return runAgentExec(t, append([]string{"-o"}, args...)...)
}

// --- Stage selectors (R1) ---

func TestAgentStageSelectorPrintsRoleCommand(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "apply")
	if err != nil {
		t.Fatalf("agent apply --print: %v", err)
	}
	model, effort := roleFill(t, agent.RoleDoing)
	want := builtinClaudeCommand(model, effort)
	if out != want {
		t.Errorf("stage selector must resolve its mapped role's command, got %q want %q", out, want)
	}
}

func TestAgentStageSelectorUnknownErrors(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentPrint(t, "bogus")
	assertFlagUsageError(t, "", err, "unknown selector")
	for _, want := range []string{"apply", "default", "doing", "fast", "hydrate", "intake", "operator", "review", "review-pr", "ship"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("unknown-selector error should name valid names incl. %q, got: %v", want, err)
		}
	}
}

// --- Selector + --provider re-resolve fills (R2) ---

func TestAgentSelectorProviderRefillsFromProvider(t *testing.T) {
	agentTestRepo(t, `providers:
  kimi:
    interactive_command: "kimi --auto -m {model}"
`)
	out, err := runAgentPrint(t, "apply", "--provider", "kimi")
	if err != nil {
		t.Fatalf("agent apply --provider kimi --print: %v", err)
	}
	if strings.TrimSpace(out) != "kimi --auto" {
		t.Errorf("selector + --provider must re-resolve fills from the named provider, got %q", out)
	}
}

func TestAgentSelectorProviderRefillKeepsPinningAgentProfile(t *testing.T) {
	agentTestRepo(t, `providers:
  kimi:
    interactive_command: "kimi --auto -m {model}"
agent:
  profiles:
    doing: { model: claude-haiku-4-5 }
`)
	out, err := runAgentPrint(t, "doing", "--provider", "kimi")
	if err != nil {
		t.Fatalf("agent doing --provider kimi --print: %v", err)
	}
	if !strings.Contains(out, "claude-haiku-4-5") {
		t.Errorf("an explicit agent.profiles.<role>.model pin still wins over the provider swap, got %q", out)
	}
}

// --- Bare-provider (no selector) fill-bypass preserved (R2) ---

func TestAgentBareProviderStillBypasses(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "--provider", "kimi")
	if err != nil {
		t.Fatalf("agent --provider kimi --print: %v", err)
	}
	if strings.TrimSpace(out) != "kimi --auto" {
		t.Errorf("bare --provider must still supply bypass fills (kimi's empty model drops -m), got %q", out)
	}
}

// --- --model/--effort post-refill overrides (R3) ---

func TestAgentRoleOverridesViaFlags(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "review", "--model", "claude-sonnet-5", "--effort", "max")
	if err != nil {
		t.Fatalf("agent review --model --effort --print: %v", err)
	}
	for _, want := range []string{"claude-sonnet-5", "max"} {
		if !strings.Contains(out, want) {
			t.Errorf("role + --model/--effort must apply the final verbatim overrides, got %q missing %q", out, want)
		}
	}
}

func TestAgentStageOverridesViaFlags(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "apply", "--model", "claude-sonnet-5")
	if err != nil {
		t.Fatalf("agent apply --model --print: %v", err)
	}
	if !strings.Contains(out, "claude-sonnet-5") {
		t.Errorf("stage + --model must apply the final verbatim override, got %q", out)
	}
}

func TestAgentDefaultRoleOverridesViaFlags(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "--model", "claude-haiku-4-5-20251001")
	if err != nil {
		t.Fatalf("agent --model --print: %v", err)
	}
	if !strings.Contains(out, "claude-haiku-4-5-20251001") {
		t.Errorf("bare --model on the default role must override the fill, got %q", out)
	}
}

func TestAgentExplicitEmptyModelClearsField(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "--model=")
	if err != nil {
		t.Fatalf("agent --model= --print: %v", err)
	}
	if strings.Contains(out, "--model") {
		t.Errorf("explicitly-empty --model= must clear the fill (token-drop), got %q", out)
	}
}

// --- -t/--template (R4) ---

func TestAgentTemplateStagePrintsRaw(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentExec(t, "apply", "-t")
	if err != nil {
		t.Fatalf("agent apply -t: %v", err)
	}
	want := agent.DefaultInteractiveCommand
	if strings.TrimSpace(out) != want {
		t.Errorf("-t must print the un-substituted template, got %q want %q", out, want)
	}
}

func TestAgentTemplateProviderPicksThatProvider(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	kimiProv, _ := agent.ResolveProvider(nil, "kimi")
	want := strings.TrimSpace(kimiProv.InteractiveCommand)
	out, err := runAgentExec(t, "apply", "--provider", "kimi", "-t")
	if err != nil {
		t.Fatalf("agent apply --provider kimi -t: %v", err)
	}
	if strings.TrimSpace(out) != want {
		t.Errorf("-t with a provider must pick that provider's template, got %q want %q", out, want)
	}
}

func TestAgentTemplateRejectsModelAndEffort(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	for _, flag := range []string{"--model", "--effort"} {
		_, err := runAgentExec(t, "apply", flag, "k3", "-t")
		assertFlagUsageError(t, "", err, "have no effect with --template")
	}
}

// --- --headless (R5) ---

func TestAgentHeadlessResolvesHeadlessCommand(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "doing", "--headless")
	if err != nil {
		t.Fatalf("agent doing --headless --print: %v", err)
	}
	model, effort := roleFill(t, agent.RoleDoing)
	want := strings.NewReplacer("{model}", model, "{effort}", effort).Replace(agent.DefaultHeadlessCommand)
	if strings.TrimSpace(out) != want {
		t.Errorf("--headless must resolve the headless_command slot, got %q want %q", out, want)
	}
}

func TestAgentHeadlessExecIsUsageError(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentExec(t, "doing", "--headless")
	assertFlagUsageError(t, "", err, "--headless is valid only in the print-family modes")
}

func TestAgentHeadlessMissingCapabilityHardErrors(t *testing.T) {
	agentTestRepo(t, `providers:
  myagent:
    interactive_command: "myagent"
`)
	_, err := runAgentPrint(t, "--headless", "--provider", "myagent")
	if err == nil {
		t.Fatal("expected a missing-capability error for --headless on a provider without headless_command")
	}
	if !strings.Contains(err.Error(), "providers.myagent.headless_command") {
		t.Errorf("--headless must hard-error naming the config key, got: %v", err)
	}
}

// --- -o yaml (R6) ---

// yamlKeys unmarshals a `-o yaml` document and returns its top-level key set.
func yamlKeys(t *testing.T, doc string) map[string]any {
	t.Helper()
	var m map[string]any
	if err := yaml.Unmarshal([]byte(doc), &m); err != nil {
		t.Fatalf("-o yaml output did not parse: %v\ndoc:\n%s", err, doc)
	}
	return m
}

func TestAgentOutputYAMLStageSelector(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentYAML(t, "yaml", "apply")
	if err != nil {
		t.Fatalf("agent apply -o yaml: %v", err)
	}
	model, effort := roleFill(t, agent.RoleDoing)
	for _, want := range []string{
		"selector: apply",
		"kind: stage",
		"role: doing",
		"provider: claude",
		"model: " + model,
		"effort: " + effort,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("-o yaml stage document missing %q, got:\n%s", want, out)
		}
	}
	if !strings.Contains(out, "command:") {
		t.Errorf("-o yaml document should carry a command key, got:\n%s", out)
	}
	keys := yamlKeys(t, out)
	if len(keys) != 11 {
		t.Errorf("-o yaml native resolution must emit the eleven non-dispatch keys, got %d: %v", len(keys), keys)
	}
}

func TestAgentOutputYAMLRoleSelector(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentYAML(t, "yaml", "doing")
	if err != nil {
		t.Fatalf("agent doing -o yaml: %v", err)
	}
	model, effort := roleFill(t, agent.RoleDoing)
	for _, want := range []string{
		"selector: doing",
		"kind: role",
		"role: doing",
		"provider: claude",
		"model: " + model,
		"effort: " + effort,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("-o yaml role document missing %q, got:\n%s", want, out)
		}
	}
	keys := yamlKeys(t, out)
	if len(keys) != 11 {
		t.Errorf("-o yaml native role resolution must emit the eleven non-dispatch keys, got %d: %v", len(keys), keys)
	}
}

func TestAgentOutputYAMLBareProviderForm(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentYAML(t, "yaml", "--provider", "kimi")
	if err != nil {
		t.Fatalf("agent --provider kimi -o yaml: %v", err)
	}
	for _, want := range []string{"kind: provider", "provider: kimi", `role: ""`, `selector: ""`} {
		if !strings.Contains(out, want) {
			t.Errorf("-o yaml bare-provider document missing %q, got:\n%s", want, out)
		}
	}
	keys := yamlKeys(t, out)
	if len(keys) != 12 {
		t.Errorf("-o yaml non-native provider resolution must emit all twelve keys, got %d: %v", len(keys), keys)
	}
}

func TestAgentOutputYAMLFullSchemaGoldens(t *testing.T) {
	tests := []struct {
		name   string
		config string
		tmux   string
		args   []string
		want   string
	}{
		{
			name: "native omits dispatch",
			config: `dispatch:
  mode: native
providers:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    native: true
    profiles:
      doing: { model: claude-opus-5, effort: high }
agent:
  profiles:
    doing: { provider: oracle }
`,
			args: []string{"yaml", "apply"},
			want: `selector: apply
kind: stage
role: doing
provider: oracle
model: claude-opus-5
effort: high
command: oracle tui -m claude-opus-5 -e high
model_alias: opus
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    provider: agent.profiles.doing
    model: providers.oracle.profiles.doing
    effort: providers.oracle.profiles.doing
`,
		},
		{
			name: "headless labels rung and keeps non-Claude alias empty",
			config: `dispatch:
  mode: native
providers:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    headless_command: "oracle exec -m {model} -e {effort}"
    profiles:
      doing: { model: gpt-5, effort: xhigh }
agent:
  profiles:
    doing: { provider: oracle }
`,
			args: []string{"yaml", "apply"},
			want: `selector: apply
kind: stage
role: doing
provider: oracle
model: gpt-5
effort: xhigh
command: oracle tui -m gpt-5 -e xhigh
model_alias: ""
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    provider: agent.profiles.doing
    model: providers.oracle.profiles.doing
    effort: providers.oracle.profiles.doing
dispatch:
    rung: headless
    command: oracle exec -m gpt-5 -e xhigh
`,
		},
		{
			name: "pane labels rung and aliases dated Claude ID",
			config: `dispatch:
  mode: pane
providers:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    headless_command: "oracle exec -m {model} -e {effort}"
    profiles:
      doing: { model: claude-haiku-4-5-20251001, effort: medium }
agent:
  profiles:
    doing: { provider: oracle }
`,
			tmux: "/tmp/tmux-1000/default,1,0",
			args: []string{"yaml", "apply"},
			want: `selector: apply
kind: stage
role: doing
provider: oracle
model: claude-haiku-4-5-20251001
effort: medium
command: oracle tui -m claude-haiku-4-5-20251001 -e medium
model_alias: haiku
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    provider: agent.profiles.doing
    model: providers.oracle.profiles.doing
    effort: providers.oracle.profiles.doing
dispatch:
    rung: pane
    command: oracle tui -m claude-haiku-4-5-20251001 -e medium
`,
		},
		{
			name: "bare provider preserves empty inherit fields",
			config: `dispatch:
  mode: headless
providers:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    headless_command: "oracle exec -m {model} -e {effort}"
`,
			args: []string{"yaml", "--provider", "oracle"},
			want: `selector: ""
kind: provider
role: ""
provider: oracle
model: ""
effort: ""
command: oracle tui
model_alias: ""
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    provider: flag
    model: ""
    effort: ""
dispatch:
    rung: headless
    command: oracle exec
`,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			agentTestRepo(t, tc.config)
			t.Setenv("TMUX", tc.tmux)
			got, err := runAgentYAML(t, tc.args...)
			if err != nil {
				t.Fatalf("agent -o %v: %v", tc.args, err)
			}
			if got != tc.want {
				t.Errorf("stdout = %q, want exact full-schema bytes %q", got, tc.want)
			}
		})
	}
}

func TestAgentOutputYAMLNoCapabilityMatchesResolveAgent(t *testing.T) {
	agentTestRepo(t, `dispatch:
  mode: native
providers:
  void:
    interactive_command: void
agent:
  workers: void
`)

	_, resolveErr := runResolveAgentCmd(t, "apply")
	if resolveErr == nil {
		t.Fatal("resolve-agent must fail when no rung at or below native is reachable")
	}
	_, agentErr := runAgentYAML(t, "yaml", "apply")
	if agentErr == nil {
		t.Fatal("agent -o yaml must fail when no rung at or below native is reachable")
	}
	if agentErr.Error() != resolveErr.Error() {
		t.Errorf("agent error = %q, resolve-agent error = %q", agentErr, resolveErr)
	}
}

func TestAgentOutputNonYAMLError(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentYAML(t, "json", "apply")
	assertFlagUsageError(t, "", err, "--output accepts exactly <yaml>")
}

func TestAgentOutputMutuallyExclusiveWithPrint(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentExec(t, "apply", "-o", "yaml", "--print")
	assertFlagUsageError(t, "", err, "mutually exclusive")
}

func TestAgentOutputMutuallyExclusiveWithTemplate(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentExec(t, "apply", "-o", "yaml", "-t")
	assertFlagUsageError(t, "", err, "mutually exclusive")
}

// --- -p shorthand (R7) ---

func TestAgentPrintShortFlagEqualsLong(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	short, errShort := runAgentPrint(t, "-p")
	long, errLong := runAgentPrint(t, "--print")
	if errShort != nil || errLong != nil {
		t.Fatalf("-p/--print must both work: short %v, long %v", errShort, errLong)
	}
	if short != long {
		t.Errorf("-p must be byte-identical to --print, got short %q want long %q", short, long)
	}
}

// --- exec seam with a stage selector (R1) ---

// TestAgentStageSelectorExecsRoleCommand: a stage selector on the EXEC path
// resolves the mapped role's command and hands it to `sh -c` — the same
// composition --print shows, through the syscall seam (execAgent is stubbed).
func TestAgentStageSelectorExecsRoleCommand(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")

	originalExec := execAgent
	t.Cleanup(func() { execAgent = originalExec })
	var gotArgv []string
	execAgent = func(path string, argv, env []string) error {
		gotArgv = append([]string(nil), argv...)
		return nil
	}

	cmd := agentCmd()
	cmd.SetArgs([]string{"apply"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agent apply (exec): %v", err)
	}
	model, effort := roleFill(t, agent.RoleDoing)
	want := builtinClaudeCommand(model, effort)
	if len(gotArgv) != 3 || gotArgv[2] != strings.TrimSuffix(want, "\n") {
		t.Errorf("stage selector must exec the mapped role's command via sh -c, got argv %#v want %q", gotArgv, want)
	}
}

// --- fills-less provider + --headless token-drop (A-014) ---

// TestAgentHeadlessFillslessProviderDropsTokens: --headless with a provider that
// ships no fills drops the placeholder token per spawn.WithProfile's
// empty-value rule (kimi's headless grammar carries {model} but no fill) — no
// panic, no validation, verbatim pass-through holds.
func TestAgentHeadlessFillslessProviderDropsTokens(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "--provider", "kimi", "--headless")
	if err != nil {
		t.Fatalf("agent --provider kimi --headless --print: %v", err)
	}
	if strings.Contains(out, "{model}") || strings.Contains(out, "-m ") {
		t.Errorf("fills-less --headless must drop the {model} pair per the token-drop rule, got %q", out)
	}
	kimi, _ := agent.ResolveProvider(nil, "kimi")
	want := spawn.WithProfile(kimi.HeadlessCommand, "", "") + "\n"
	if out != want {
		t.Errorf("got %q, want the empty-fill substitution of kimi's headless grammar %q", out, want)
	}
}

// --- -t with an unknown --provider (A-015) ---

// TestAgentTemplateUnknownProviderErrors: -t still resolves the provider, so an
// unknown one is the shared lookup failure (unknownProviderError's phrasing),
// not a template-specific message.
func TestAgentTemplateUnknownProviderErrors(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentExec(t, "apply", "--provider", "bogus", "-t")
	assertFlagUsageError(t, "", err, "unknown provider", "bogus", "configure it under providers.")
}
