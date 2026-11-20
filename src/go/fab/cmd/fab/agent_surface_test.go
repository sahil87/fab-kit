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

// --- Selector + --cli re-resolve fills (R2) ---

func TestAgentSelectorCLIRefillsFromCLI(t *testing.T) {
	agentTestRepo(t, `clis:
  kimi:
    interactive_command: "kimi --auto -m {model}"
`)
	out, err := runAgentPrint(t, "apply", "--cli", "kimi")
	if err != nil {
		t.Fatalf("agent apply --cli kimi --print: %v", err)
	}
	if strings.TrimSpace(out) != "kimi --auto" {
		t.Errorf("selector + --cli must re-resolve fills from the named CLI, got %q", out)
	}
}

func TestAgentSelectorCLIRefillKeepsPinningAgentProfile(t *testing.T) {
	agentTestRepo(t, `clis:
  kimi:
    interactive_command: "kimi --auto -m {model}"
agent:
  profiles:
    doing: { model: claude-haiku-4-5 }
`)
	out, err := runAgentPrint(t, "doing", "--cli", "kimi")
	if err != nil {
		t.Fatalf("agent doing --cli kimi --print: %v", err)
	}
	if !strings.Contains(out, "claude-haiku-4-5") {
		t.Errorf("an explicit agent.profiles.<role>.model pin still wins over the CLI swap, got %q", out)
	}
}

// --- Bare-cli (no selector) fill-bypass preserved (R2) ---

func TestAgentBareCLIStillBypasses(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "--cli", "kimi")
	if err != nil {
		t.Fatalf("agent --cli kimi --print: %v", err)
	}
	if strings.TrimSpace(out) != "kimi --auto" {
		t.Errorf("bare --cli must still supply bypass fills (kimi's empty model drops -m), got %q", out)
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

func TestAgentTemplateCLIPicksThatCLI(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	kimiProv, _ := agent.ResolveCLI(nil, "kimi")
	want := strings.TrimSpace(kimiProv.InteractiveCommand)
	out, err := runAgentExec(t, "apply", "--cli", "kimi", "-t")
	if err != nil {
		t.Fatalf("agent apply --cli kimi -t: %v", err)
	}
	if strings.TrimSpace(out) != want {
		t.Errorf("-t with a CLI must pick that CLI's template, got %q want %q", out, want)
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
	agentTestRepo(t, `clis:
  myagent:
    interactive_command: "myagent"
`)
	_, err := runAgentPrint(t, "--headless", "--cli", "myagent")
	if err == nil {
		t.Fatal("expected a missing-capability error for --headless on a CLI without headless_command")
	}
	if !strings.Contains(err.Error(), "clis.myagent.headless_command") {
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
		"cli: claude",
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
	if len(keys) != 13 {
		t.Errorf("-o yaml native resolution must emit the thirteen non-dispatch keys (twelve plus the deprecated provider duplicate), got %d: %v", len(keys), keys)
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
		"cli: claude",
		"model: " + model,
		"effort: " + effort,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("-o yaml role document missing %q, got:\n%s", want, out)
		}
	}
	keys := yamlKeys(t, out)
	if len(keys) != 13 {
		t.Errorf("-o yaml native role resolution must emit the thirteen non-dispatch keys (twelve plus the deprecated provider duplicate), got %d: %v", len(keys), keys)
	}
}

// TestAgentOutputYAMLBuiltinClaudeFillIsAlias: the built-in claude fills are the
// family aliases, so the review role resolves `opus` on every surface — the model,
// the native Agent-tool alias, and the composed claude command.
func TestAgentOutputYAMLBuiltinClaudeFillIsAlias(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentYAML(t, "yaml", "review")
	if err != nil {
		t.Fatalf("agent review -o yaml: %v", err)
	}
	var doc agent.Resolution
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("parse -o yaml: %v\n%s", err, out)
	}
	if doc.Model != "opus" || doc.ModelAlias != "opus" {
		t.Errorf("model/model_alias = %q/%q, want opus/opus:\n%s", doc.Model, doc.ModelAlias, out)
	}
	if !strings.Contains(doc.Command, "--model opus") {
		t.Errorf("command should carry --model opus, got %q", doc.Command)
	}
}

// TestAgentOutputYAMLNonClaudeModelHasNoAlias: a non-Claude model keeps an empty
// model_alias even though bare aliases now resolve to themselves.
func TestAgentOutputYAMLNonClaudeModelHasNoAlias(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentYAML(t, "yaml", "review", "--model", "gpt-5")
	if err != nil {
		t.Fatalf("agent review --model gpt-5 -o yaml: %v", err)
	}
	if !strings.Contains(out, "model: gpt-5\n") || !strings.Contains(out, `model_alias: ""`) {
		t.Errorf("non-Claude model must resolve an empty model_alias, got:\n%s", out)
	}
}

func TestAgentOutputYAMLToolDefaultModel(t *testing.T) {
	t.Run("role pin wins over codex fill", func(t *testing.T) {
		agentTestRepo(t, `agent:
  profiles:
    review: { cli: codex, model: tool-default }
`)
		out, err := runAgentYAML(t, "yaml", "review")
		if err != nil {
			t.Fatalf("agent review -o yaml: %v", err)
		}
		var doc agent.Resolution
		if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("parse -o yaml: %v\n%s", err, out)
		}
		if doc.Model != "" || doc.Source.Model != "agent.profiles.review" {
			t.Errorf("model/source.model = %q/%q, want empty/agent.profiles.review", doc.Model, doc.Source.Model)
		}
		for _, command := range []string{doc.Command, doc.Dispatch.Command} {
			if strings.Contains(command, " -m ") || strings.Contains(command, agent.ToolDefaultModel) || strings.Contains(command, "{model}") {
				t.Errorf("command must drop the model flag pair, got %q", command)
			}
		}
	})

	t.Run("CLI-only flag collapses identically", func(t *testing.T) {
		agentTestRepo(t, "project:\n  name: test\n")
		out, err := runAgentYAML(t, "yaml", "--cli", "codex", "--model", agent.ToolDefaultModel)
		if err != nil {
			t.Fatalf("agent --cli codex --model tool-default -o yaml: %v", err)
		}
		var doc agent.Resolution
		if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
			t.Fatalf("parse -o yaml: %v\n%s", err, out)
		}
		if doc.Model != "" || doc.Source.Model != "flag" {
			t.Errorf("model/source.model = %q/%q, want empty/flag", doc.Model, doc.Source.Model)
		}
		for _, command := range []string{doc.Command, doc.Dispatch.Command} {
			if strings.Contains(command, " -m ") || strings.Contains(command, agent.ToolDefaultModel) || strings.Contains(command, "{model}") {
				t.Errorf("command must drop the model flag pair, got %q", command)
			}
		}
	})
}

func TestAgentOutputYAMLToolDefaultNativeAndHeadless(t *testing.T) {
	for _, mode := range []string{"native", "headless"} {
		t.Run(mode, func(t *testing.T) {
			agentTestRepo(t, `dispatch:
  mode: `+mode+`
agent:
  profiles:
    doing: { model: tool-default }
`)
			out, err := runAgentYAML(t, "yaml", "apply")
			if err != nil {
				t.Fatalf("agent apply -o yaml: %v", err)
			}
			var doc agent.Resolution
			if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
				t.Fatalf("parse -o yaml: %v\n%s", err, out)
			}
			if doc.Model != "" || doc.ModelAlias != "" || doc.Source.Model != "agent.profiles.doing" {
				t.Errorf("model/alias/source = %q/%q/%q, want empty/empty/agent.profiles.doing", doc.Model, doc.ModelAlias, doc.Source.Model)
			}
			if strings.Contains(doc.Command, "--model") || strings.Contains(doc.Command, agent.ToolDefaultModel) {
				t.Errorf("interactive command must drop --model, got %q", doc.Command)
			}
			if mode == "native" {
				if doc.Dispatch != nil {
					t.Errorf("native resolution must omit dispatch, got %+v", doc.Dispatch)
				}
			} else {
				if doc.Dispatch == nil {
					t.Fatal("headless resolution must carry dispatch")
				}
				if strings.Contains(doc.Dispatch.Command, "--model") || strings.Contains(doc.Dispatch.Command, agent.ToolDefaultModel) {
					t.Errorf("dispatch command must drop --model, got %q", doc.Dispatch.Command)
				}
			}
		})
	}
}

func TestAgentOutputYAMLBareCLIForm(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentYAML(t, "yaml", "--cli", "kimi")
	if err != nil {
		t.Fatalf("agent --cli kimi -o yaml: %v", err)
	}
	for _, want := range []string{"kind: cli", "cli: kimi", `role: ""`, `selector: ""`} {
		if !strings.Contains(out, want) {
			t.Errorf("-o yaml bare-cli document missing %q, got:\n%s", want, out)
		}
	}
	keys := yamlKeys(t, out)
	if len(keys) != 14 {
		t.Errorf("-o yaml non-native CLI resolution must emit all fourteen keys (thirteen plus the deprecated provider duplicate), got %d: %v", len(keys), keys)
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
clis:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    native: true
    profiles:
      doing: { model: claude-opus-5, effort: high }
agent:
  profiles:
    doing: { cli: oracle }
`,
			args: []string{"yaml", "apply"},
			want: `selector: apply
kind: stage
role: doing
cli: oracle
provider: oracle
model: claude-opus-5
effort: high
command: oracle tui -m claude-opus-5 -e high
model_alias: opus
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    cli: agent.profiles.doing
    provider: agent.profiles.doing
    model: clis.oracle.profiles.doing
    effort: clis.oracle.profiles.doing
skill_prefix: /
`,
		},
		{
			name: "headless labels rung and keeps non-Claude alias empty",
			config: `dispatch:
  mode: native
clis:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    headless_command: "oracle exec -m {model} -e {effort}"
    profiles:
      doing: { model: gpt-5, effort: xhigh }
agent:
  profiles:
    doing: { cli: oracle }
`,
			args: []string{"yaml", "apply"},
			want: `selector: apply
kind: stage
role: doing
cli: oracle
provider: oracle
model: gpt-5
effort: xhigh
command: oracle tui -m gpt-5 -e xhigh
model_alias: ""
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    cli: agent.profiles.doing
    provider: agent.profiles.doing
    model: clis.oracle.profiles.doing
    effort: clis.oracle.profiles.doing
dispatch:
    rung: headless
    command: oracle exec -m gpt-5 -e xhigh
    reason: native unavailable
skill_prefix: /
`,
		},
		{
			name: "pane labels rung and aliases dated Claude ID",
			config: `dispatch:
  mode: pane
clis:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    headless_command: "oracle exec -m {model} -e {effort}"
    profiles:
      doing: { model: claude-haiku-4-5-20251001, effort: medium }
agent:
  profiles:
    doing: { cli: oracle }
`,
			tmux: "/tmp/tmux-1000/default,1,0",
			args: []string{"yaml", "apply"},
			want: `selector: apply
kind: stage
role: doing
cli: oracle
provider: oracle
model: claude-haiku-4-5-20251001
effort: medium
command: oracle tui -m claude-haiku-4-5-20251001 -e medium
model_alias: haiku
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    cli: agent.profiles.doing
    provider: agent.profiles.doing
    model: clis.oracle.profiles.doing
    effort: clis.oracle.profiles.doing
dispatch:
    rung: pane
    command: oracle tui -m claude-haiku-4-5-20251001 -e medium
skill_prefix: /
`,
		},
		{
			name: "bare CLI preserves empty inherit fields",
			config: `dispatch:
  mode: headless
clis:
  oracle:
    interactive_command: "oracle tui -m {model} -e {effort}"
    headless_command: "oracle exec -m {model} -e {effort}"
`,
			args: []string{"yaml", "--cli", "oracle"},
			want: `selector: ""
kind: cli
role: ""
cli: oracle
provider: oracle
model: ""
effort: ""
command: oracle tui
model_alias: ""
template: oracle tui -m {model} -e {effort}
fill_mode: template
source:
    cli: flag
    provider: flag
    model: ""
    effort: ""
dispatch:
    rung: headless
    command: oracle exec
skill_prefix: /
`,
		},
		{
			name: "codex CLI carries the dollar skill prefix",
			config: `dispatch:
  mode: native
clis:
  codex:
    interactive_command: "codex tui -m {model} -e {effort}"
    native: true
`,
			args: []string{"yaml", "--cli", "codex", "--model", "gpt-x", "--effort", "high"},
			want: `selector: ""
kind: cli
role: ""
cli: codex
provider: codex
model: gpt-x
effort: high
command: codex tui -m gpt-x -e high
model_alias: ""
template: codex tui -m {model} -e {effort}
fill_mode: template
source:
    cli: flag
    provider: flag
    model: flag
    effort: flag
dispatch:
    rung: headless
    command: codex exec --dangerously-bypass-approvals-and-sandbox -m gpt-x -c model_reasoning_effort=high
    reason: 'native unavailable: model has no native alias'
skill_prefix: $
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

func TestAgentOutputYAMLDispatchReasonAndNativeModelRule(t *testing.T) {
	const aliaslessConfig = `dispatch:
  mode: native
agent:
  profiles:
    doing: { model: "glm-latest[1m]" }
`

	t.Run("alias-less claude model descends with the model and reason", func(t *testing.T) {
		agentTestRepo(t, aliaslessConfig)
		got, err := runAgentYAML(t, "yaml", "apply")
		if err != nil {
			t.Fatalf("agent -o yaml: %v", err)
		}
		for _, want := range []string{
			"model: glm-latest[1m]\n",
			"model_alias: \"\"\n",
			"    rung: headless\n",
			"    command: claude -p --permission-mode bypassPermissions --model glm-latest[1m] --effort high\n",
			"    reason: 'native unavailable: model has no native alias'\n",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("stdout = %q, want it to contain %q", got, want)
			}
		}
	})

	for _, tc := range []struct {
		name  string
		model string
		alias string
	}{
		{"empty model inherits", "", ""},
		{"bare alias stays native", "opus", "opus"},
		{"versioned claude model stays native", "claude-opus-5-5", "opus"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			agentTestRepo(t, aliaslessConfig)
			got, err := runAgentYAML(t, "yaml", "apply", "--model="+tc.model)
			if err != nil {
				t.Fatalf("agent -o yaml --model=%s: %v", tc.model, err)
			}
			if strings.Contains(got, "dispatch:\n") {
				t.Errorf("stdout = %q, native resolution must omit dispatch", got)
			}
			aliasYAML := tc.alias
			if aliasYAML == "" {
				aliasYAML = `""`
			}
			if !strings.Contains(got, "model_alias: "+aliasYAML+"\n") {
				t.Errorf("stdout = %q, want model_alias %q", got, tc.alias)
			}
		})
	}

	t.Run("non-native CLI reports the capability reason", func(t *testing.T) {
		agentTestRepo(t, "dispatch:\n  mode: native\nagent:\n  workers: codex\n")
		got, err := runAgentYAML(t, "yaml", "apply")
		if err != nil {
			t.Fatalf("agent -o yaml: %v", err)
		}
		if !strings.Contains(got, "    reason: native unavailable\n") || strings.Contains(got, "model has no native alias") {
			t.Errorf("stdout = %q, want only the native-capability reason", got)
		}
	})

	t.Run("headless preference has no descent reason", func(t *testing.T) {
		agentTestRepo(t, `dispatch:
  mode: headless
agent:
  profiles:
    doing: { model: "glm-latest[1m]" }
`)
		got, err := runAgentYAML(t, "yaml", "apply")
		if err != nil {
			t.Fatalf("agent -o yaml: %v", err)
		}
		if !strings.Contains(got, "dispatch:\n") || strings.Contains(got, "    reason:") {
			t.Errorf("stdout = %q, preferred headless must carry dispatch without reason", got)
		}
	})
}

func TestAgentOutputYAMLNoCapabilityMatchesResolveAgent(t *testing.T) {
	agentTestRepo(t, `dispatch:
  mode: native
clis:
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

// --- fills-less CLI + --headless token-drop (A-014) ---

// TestAgentHeadlessFillslessCLIDropsTokens: --headless with a CLI that
// ships no fills drops the placeholder token per spawn.WithProfile's
// empty-value rule (kimi's headless grammar carries {model} but no fill) — no
// panic, no validation, verbatim pass-through holds.
func TestAgentHeadlessFillslessCLIDropsTokens(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	out, err := runAgentPrint(t, "--cli", "kimi", "--headless")
	if err != nil {
		t.Fatalf("agent --cli kimi --headless --print: %v", err)
	}
	if strings.Contains(out, "{model}") || strings.Contains(out, "-m ") {
		t.Errorf("fills-less --headless must drop the {model} pair per the token-drop rule, got %q", out)
	}
	kimi, _ := agent.ResolveCLI(nil, "kimi")
	want := spawn.WithProfile(kimi.HeadlessCommand, "", "") + "\n"
	if out != want {
		t.Errorf("got %q, want the empty-fill substitution of kimi's headless grammar %q", out, want)
	}
}

// --- -t with an unknown --cli (A-015) ---

// TestAgentTemplateUnknownCLIErrors: -t still resolves the CLI, so an
// unknown one is the shared lookup failure (unknownCLIError's phrasing),
// not a template-specific message.
func TestAgentTemplateUnknownCLIErrors(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	_, err := runAgentExec(t, "apply", "--cli", "bogus", "-t")
	assertFlagUsageError(t, "", err, "unknown CLI", "bogus", "configure it under clis.")
}
