package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/agentmodels"
)

const modelsCommandFixture = `providers:
  mytool:
    models_command: 'printf ''[{"id":"m1","label":"Model One","efforts":["high"],"default":true,"cli":"wrong","source":"fill"}]'''
    profiles:
      default: { model: fill-model }
`

func runAgentModels(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	cmd := agentCmd()
	var stdout, stderr bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&stderr)
	cmd.SetArgs(append([]string{"models"}, args...))
	err := cmd.Execute()
	return stdout.String(), stderr.String(), err
}

func isolateModelsHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("CLAUDE_CONFIG_DIR", filepath.Join(home, "no-claude"))
	t.Setenv("CODEX_HOME", filepath.Join(home, "no-codex"))
}

func TestAgentModelsJSON(t *testing.T) {
	agentTestRepo(t, modelsCommandFixture)
	isolateModelsHome(t)
	stdout, stderr, err := runAgentModels(t, "mytool", "--json")
	if err != nil || stderr != "" {
		t.Fatalf("err=%v stderr=%q", err, stderr)
	}
	var rows []agentmodels.Row
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("stdout is not a JSON array: %v\n%s", err, stdout)
	}
	if len(rows) != 3 {
		t.Fatalf("rows = %+v", rows)
	}
	if rows[0].CLI != "mytool" || rows[0].ID != agent.ToolDefaultModel || rows[0].Label != "tool default (m1)" || rows[0].Default || rows[0].Source != agentmodels.SourceReserved {
		t.Errorf("reserved row = %+v", rows[0])
	}
	if len(rows[0].Efforts) != 1 || rows[0].Efforts[0] != "high" {
		t.Errorf("reserved row efforts = %v, want [high]", rows[0].Efforts)
	}
	if rows[1].CLI != "mytool" || rows[1].ID != "m1" || rows[1].Label != "Model One" || !rows[1].Default || rows[1].Source != agentmodels.SourceTool {
		t.Errorf("tool row = %+v", rows[1])
	}
	if rows[2].Source != agentmodels.SourceFill || rows[2].Efforts == nil {
		t.Errorf("fill row = %+v", rows[2])
	}
	var raw []map[string]json.RawMessage
	if err := json.Unmarshal([]byte(stdout), &raw); err != nil {
		t.Fatal(err)
	}
	if len(raw[0]) != 6 {
		t.Errorf("row keys = %v, want exactly six", raw[0])
	}
}

func TestAgentModelsTable(t *testing.T) {
	agentTestRepo(t, modelsCommandFixture)
	isolateModelsHome(t)
	stdout, _, err := runAgentModels(t, "mytool")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(stdout), "\n")
	if got := strings.Join(strings.Fields(lines[0]), " "); got != "CLI MODEL LABEL EFFORTS SOURCE DEFAULT" {
		t.Errorf("header = %q", got)
	}
	if !strings.Contains(lines[1], agent.ToolDefaultModel) || !strings.Contains(lines[1], "tool default (m1)") || !strings.Contains(lines[1], agentmodels.SourceReserved) {
		t.Errorf("first data row = %q, want the reserved tool-default row", lines[1])
	}
	if !strings.Contains(lines[2], "Model One") || !strings.HasSuffix(strings.TrimSpace(lines[2]), "*") {
		t.Errorf("default row = %q", lines[2])
	}
}

func TestAgentModelsUnknownProviderAndArity(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	isolateModelsHome(t)
	_, _, err := runAgentModels(t, "nope")
	if err == nil {
		t.Fatal("unknown provider succeeded")
	}
	for _, want := range []string{"nope", "agy, claude, codex, kimi"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("error = %q, want %q", err, want)
		}
	}
	if _, _, err := runAgentModels(t, "claude", "codex"); err == nil {
		t.Fatal("two providers succeeded")
	}
}

func TestAgentModelsRepoFlag(t *testing.T) {
	agentTestRepo(t, modelsCommandFixture)
	isolateModelsHome(t)
	target := t.TempDir()
	projectDir := filepath.Join(target, "fab", "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	config := strings.Replace(modelsCommandFixture, "m1", "target-model", 1)
	if err := os.WriteFile(filepath.Join(projectDir, "config.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runAgentModels(t, "mytool", "--json", "--repo", target)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(stdout, "target-model") || strings.Contains(stdout, `"id":"m1"`) {
		t.Errorf("stdout = %s", stdout)
	}
}

func TestAgentModelsNoProjectAndWarnings(t *testing.T) {
	home := noProjectDir(t)
	kimiDir := filepath.Join(home, ".kimi-code")
	if err := os.MkdirAll(kimiDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(kimiDir, "config.toml"), []byte("[models.test]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	stdout, stderr, err := runAgentModels(t, "codex", "--json")
	if err != nil {
		t.Fatalf("project-free models: %v", err)
	}
	var rows []agentmodels.Row
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil || len(rows) == 0 {
		t.Fatalf("rows=%+v err=%v", rows, err)
	}
	if rows[0].ID != agent.ToolDefaultModel || rows[0].Label != "tool default" || rows[0].Source != agentmodels.SourceReserved {
		t.Errorf("reserved row = %+v", rows[0])
	}
	for _, row := range rows[1:] {
		if row.CLI != "codex" || row.Source != agentmodels.SourceFill {
			t.Errorf("row = %+v", row)
		}
	}
	if !strings.Contains(stderr, "warning: codex:") || !strings.Contains(stderr, "executable file not found") {
		t.Errorf("stderr = %q", stderr)
	}

	stdout, _, err = runAgentModels(t, "--json")
	if err != nil {
		t.Fatal(err)
	}
	rows = nil
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatal(err)
	}
	seen := make(map[string]bool)
	for _, row := range rows {
		seen[row.CLI] = true
	}
	for _, name := range agent.ProviderNames(nil) {
		if !seen[name] {
			t.Errorf("provider %s missing from project-free rows", name)
		}
	}
}

func TestAgentModelsDoesNotChangeParentForms(t *testing.T) {
	agentTestRepo(t, "project:\n  name: test\n")
	isolateModelsHome(t)
	yamlOutput, err := runAgentYAML(t, "yaml", "review")
	if err != nil || !strings.Contains(yamlOutput, "selector: review\n") {
		t.Errorf("review YAML err=%v output=%s", err, yamlOutput)
	}
	printOutput, err := runAgentPrint(t, "apply")
	if err != nil {
		t.Fatal(err)
	}
	model, effort := roleFill(t, agent.RoleDoing)
	if want := builtinClaudeCommand(model, effort); printOutput != want {
		t.Errorf("print = %q, want %q", printOutput, want)
	}
	passthrough, err := runAgentPrint(t, "--", "--resume")
	if err != nil || !strings.HasSuffix(passthrough, " '--resume'\n") {
		t.Errorf("passthrough err=%v output=%q", err, passthrough)
	}
}
