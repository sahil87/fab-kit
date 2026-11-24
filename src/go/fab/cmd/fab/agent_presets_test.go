package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configscope"
)

// presetTestRepo is agentTestRepo plus an isolated HOME and cleared FAB_* registry
// variables (so a developer's own preset env/system config cannot reach the test).
func presetTestRepo(t *testing.T, projectYAML string) (repo, home string) {
	t.Helper()
	home = t.TempDir()
	t.Setenv("HOME", home)
	for _, key := range configscope.DottedKeys() {
		t.Setenv("FAB_"+strings.ToUpper(strings.ReplaceAll(key, ".", "_")), "")
	}
	t.Setenv("FAB_PROVIDERS", "")
	repo = agentTestRepo(t, projectYAML)
	return repo, home
}

func runAgentPresets(t *testing.T, args ...string) (stdout, stderr string, err error) {
	t.Helper()
	cmd := agentCmd()
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(append([]string{"presets"}, args...))
	err = cmd.Execute()
	return out.String(), errb.String(), err
}

func runAgentArgs(t *testing.T, args ...string) (stdout string, err error) {
	t.Helper()
	cmd := agentCmd()
	var out, errb bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&errb)
	cmd.SetArgs(args)
	err = cmd.Execute()
	return out.String(), err
}

func TestAgentPresetsJSONContract(t *testing.T) {
	_, home := presetTestRepo(t, "agent_presets:\n  mine:\n    description: mine\n    workers: agy\n    profiles:\n      review: { model: m1, effort: high }\n")
	writeSystemConfig(t, home, "agent_presets:\n  codex-review:\n    description: tweaked\n")

	stdout, stderr, err := runAgentPresets(t, "--json")
	if err != nil || stderr != "" {
		t.Fatalf("err=%v stderr=%q", err, stderr)
	}
	var rows []map[string]any
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatalf("not a JSON array: %v\n%s", err, stdout)
	}
	var names []string
	byName := map[string]map[string]any{}
	for _, r := range rows {
		n := r["name"].(string)
		names = append(names, n)
		byName[n] = r
	}
	if got, want := strings.Join(names, ","), "claude,codex-review,codex-workers,mine"; got != want {
		t.Fatalf("names = %s, want %s (sorted, no none)", got, want)
	}

	// Sparse: codex-workers carries only name/description/source/workers.
	cw := byName["codex-workers"]
	if len(cw) != 4 || cw["workers"] != "codex" || cw["source"] != "built-in" {
		t.Errorf("codex-workers = %v", cw)
	}
	// A system tweak of a built-in reports system and keeps the built-in fields.
	cr := byName["codex-review"]
	if cr["source"] != "system" || cr["description"] != "tweaked" {
		t.Errorf("codex-review = %v", cr)
	}
	profiles := cr["profiles"].(map[string]any)["review"].(map[string]any)
	if len(profiles) != 1 || profiles["cli"] != "codex" {
		t.Errorf("codex-review.profiles.review = %v", profiles)
	}
	// Project definition: model/effort without cli — cli omitted, never empty.
	mine := byName["mine"]
	if mine["source"] != "project" {
		t.Errorf("mine source = %v", mine["source"])
	}
	role := mine["profiles"].(map[string]any)["review"].(map[string]any)
	if _, has := role["cli"]; has || role["model"] != "m1" || role["effort"] != "high" {
		t.Errorf("mine review = %v", role)
	}
	if _, has := mine["session"]; has {
		t.Errorf("unset session was emitted: %v", mine)
	}
	for _, r := range rows {
		if _, has := r["active"]; has {
			t.Errorf("row carries an active flag: %v", r)
		}
	}
}

func TestAgentPresetsEnvSourceAndTable(t *testing.T) {
	presetTestRepo(t, "{}\n")
	t.Setenv("FAB_AGENT_PRESETS", "{envmade: {description: from env, session: codex}}")
	stdout, _, err := runAgentPresets(t, "--json")
	if err != nil {
		t.Fatal(err)
	}
	var rows []presetJSON
	if err := json.Unmarshal([]byte(stdout), &rows); err != nil {
		t.Fatal(err)
	}
	found := false
	for _, r := range rows {
		if r.Name == "envmade" {
			found = true
			if r.Source != "env" || r.Session != "codex" {
				t.Errorf("envmade = %+v", r)
			}
		}
	}
	if !found {
		t.Fatalf("env-defined preset missing: %s", stdout)
	}

	table, _, err := runAgentPresets(t)
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(table), "\n")
	if got := strings.Join(strings.Fields(lines[0]), " "); got != "NAME SOURCE DESCRIPTION SETS" {
		t.Errorf("header = %q", got)
	}
	if !strings.Contains(table, "codex-review") || !strings.Contains(table, "review.cli=codex") {
		t.Errorf("table = %s", table)
	}
	if strings.Contains(table, "none") {
		t.Errorf("table lists none: %s", table)
	}
}

func TestAgentPresetsRepoFlag(t *testing.T) {
	presetTestRepo(t, "{}\n")
	target := t.TempDir()
	dir := filepath.Join(target, "fab", "project")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("agent_presets:\n  elsewhere:\n    workers: kimi\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	stdout, _, err := runAgentPresets(t, "--json", "--repo", target)
	if err != nil || !strings.Contains(stdout, `"name":"elsewhere"`) {
		t.Fatalf("err=%v stdout=%s", err, stdout)
	}
}

func TestAgentPresetFlagEquivalentToEnv(t *testing.T) {
	presetTestRepo(t, "agent:\n  workers: claude\n")
	viaFlag, err := runAgentArgs(t, "review", "--preset", "codex-workers", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("FAB_AGENT_PRESET", "codex-workers")
	viaEnv, err := runAgentArgs(t, "review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if viaFlag != viaEnv {
		t.Errorf("--preset differs from FAB_AGENT_PRESET:\nflag:\n%s\nenv:\n%s", viaFlag, viaEnv)
	}
	if !strings.Contains(viaFlag, "cli: codex\n") || !strings.Contains(viaFlag, "preset: codex-workers\n") {
		t.Errorf("yaml = %s", viaFlag)
	}
}

func TestAgentPresetYAMLSourcesAndOmission(t *testing.T) {
	presetTestRepo(t, "{}\n")

	// No preset: no `preset:` key (omitted like dispatch:).
	plain, err := runAgentArgs(t, "review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(plain, "preset:") {
		t.Errorf("preset key present with no preset active:\n%s", plain)
	}

	out, err := runAgentArgs(t, "review", "--preset", "codex-review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Preset string `yaml:"preset"`
		CLI    string `yaml:"cli"`
		Source struct {
			CLI    string `yaml:"cli"`
			Model  string `yaml:"model"`
			Effort string `yaml:"effort"`
		} `yaml:"source"`
	}
	if err := yaml.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatal(err)
	}
	if doc.Preset != "codex-review" || doc.CLI != "codex" {
		t.Errorf("doc = %+v", doc)
	}
	if doc.Source.CLI != "agent_presets.codex-review" || doc.Source.Model != "clis.codex.profiles.review" || doc.Source.Effort != "clis.codex.profiles.review" {
		t.Errorf("source = %+v", doc.Source)
	}

	// A role the preset does not touch resolves as before, with the preset still reported.
	other, err := runAgentArgs(t, "doing", "--preset", "codex-review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(other, "cli: claude\n") || strings.Contains(other, "agent_presets.") {
		t.Errorf("doing under codex-review:\n%s", other)
	}
}

func TestAgentPresetFlagGuards(t *testing.T) {
	presetTestRepo(t, "{}\n")
	if _, err := runAgentArgs(t, "--cli", "codex", "--preset", "codex-review", "--print"); err == nil ||
		!strings.Contains(err.Error(), "--preset") {
		t.Fatalf("bare --cli + --preset must be a usage error, got %v", err)
	}
	// With a selector the pair is legal.
	if _, err := runAgentArgs(t, "review", "--cli", "codex", "--preset", "codex-workers", "--print"); err != nil {
		t.Fatalf("selector + --cli + --preset: %v", err)
	}
	// --preset is resolution-only: it never exports FAB_AGENT_PRESET into the exec'd session.
	var gotEnv []string
	orig := execAgent
	execAgent = func(_ string, _ []string, env []string) error { gotEnv = env; return nil }
	t.Cleanup(func() { execAgent = orig })
	if _, err := runAgentArgs(t, "review", "--preset", "codex-workers"); err != nil {
		t.Fatal(err)
	}
	for _, e := range gotEnv {
		if strings.HasPrefix(e, "FAB_AGENT_PRESET=") && e != "FAB_AGENT_PRESET=" {
			t.Errorf("--preset exported %q into the launched session", e)
		}
	}
	// resolve-agent keeps its byte-stable surface: no --preset flag.
	if resolveAgentCmd().Flags().Lookup("preset") != nil {
		t.Error("fab resolve-agent must not gain --preset")
	}
	// Help states the resolution-only caveat.
	help := agentCmd().Flags().Lookup("preset").Usage
	for _, want := range []string{"FAB_AGENT_PRESET", "fab dispatch start", "agent.preset"} {
		if !strings.Contains(help, want) {
			t.Errorf("--preset help missing %q: %s", want, help)
		}
	}
}

func TestAgentPresetUnknownWarnsAndMatchesUnset(t *testing.T) {
	presetTestRepo(t, "{}\n")
	want, err := runAgentArgs(t, "review", "-o", "yaml")
	if err != nil {
		t.Fatal(err)
	}
	var got string
	stderr := captureStderrOutput(t, func() {
		got, err = runAgentArgs(t, "review", "--preset", "nope", "-o", "yaml")
	})
	if err != nil {
		t.Fatalf("unknown preset must never fail a resolution: %v", err)
	}
	if got != want {
		t.Errorf("unknown preset changed resolution:\n%s\nvs\n%s", got, want)
	}
	if !strings.Contains(stderr, `fab: warning: unknown agent preset "nope"`) {
		t.Errorf("stderr = %q", stderr)
	}
}

func captureStderrOutput(t *testing.T, fn func()) string {
	t.Helper()
	old := os.Stderr
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stderr = w
	defer func() { os.Stderr = old }()
	fn()
	w.Close()
	var buf bytes.Buffer
	_, _ = buf.ReadFrom(r)
	return buf.String()
}

func TestConfigSetAgentPresetValidation(t *testing.T) {
	setupConfigRepo(t, "project:\n  name: test\n")
	_, _, err := runConfigSplit(t, "set", "agent.preset", "bogus")
	if err == nil {
		t.Fatal("undefined preset name must be refused")
	}
	for _, w := range []string{"agent.preset", "bogus", "claude, codex-review, codex-workers", "none"} {
		if !strings.Contains(err.Error(), w) {
			t.Errorf("error must mention %q: %v", w, err)
		}
	}
	if out, _, err := runConfigSplit(t, "set", "agent.preset", "codex-review"); err != nil || !strings.Contains(out, "Set agent.preset") {
		t.Fatalf("defined name: out=%q err=%v", out, err)
	}
	if _, _, err := runConfigSplit(t, "set", "agent.preset", "none"); err != nil {
		t.Fatalf("none: %v", err)
	}
	// A preset defined by a dotted set is then selectable.
	if _, _, err := runConfigSplit(t, "set", "agent_presets.mine.workers", "agy"); err != nil {
		t.Fatalf("dotted agent_presets leaf: %v", err)
	}
	if _, _, err := runConfigSplit(t, "set", "agent.preset", "mine"); err != nil {
		t.Fatalf("project-defined name: %v", err)
	}
	if _, _, err := runConfigSplit(t, "set", "agent_presets.mine.profiles.review.cli", "codex"); err != nil {
		t.Fatalf("deep dotted leaf: %v", err)
	}
	if _, _, err := runConfigSplit(t, "set", "agent_presets.mine.bogus", "x"); err == nil {
		t.Fatal("unknown preset field accepted")
	}
	if _, _, err := runConfigSplit(t, "unset", "agent_presets.mine.profiles.review.cli"); err != nil {
		t.Fatalf("unset dotted leaf: %v", err)
	}
	if _, _, err := runConfigSplit(t, "unset", "agent.preset"); err != nil {
		t.Fatalf("unset agent.preset: %v", err)
	}
}

func TestConfigShowOriginPresetDetail(t *testing.T) {
	repo, _ := setupConfigRepo(t, "agent:\n  preset: codex-workers\n  session: agy\n")
	_ = repo
	out, _, err := runConfigSplit(t, "show", "--origin")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out, "agent.workers = codex") {
		t.Fatalf("show lacks the spliced workers leaf:\n%s", out)
	}
	for _, line := range strings.Split(out, "\n") {
		switch {
		case strings.HasPrefix(line, "agent.workers = codex"):
			if !strings.Contains(line, "(preset codex-workers)") || !strings.Contains(line, "config.yaml") {
				t.Errorf("preset leaf origin = %q", line)
			}
		case strings.HasPrefix(line, "agent.session = agy"):
			if strings.Contains(line, "preset") {
				t.Errorf("explicit leaf labelled as preset: %q", line)
			}
		case strings.HasPrefix(line, "agent.preset = "):
			if strings.Contains(line, "(preset") {
				t.Errorf("selector labelled as preset-supplied: %q", line)
			}
		}
	}

	jsonOut, _, err := runConfigSplit(t, "show", "--origin", "--json")
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range jsonRows(t, jsonOut) {
		if rowString(t, row, "key") != "agent.workers" {
			continue
		}
		if rowString(t, row, "origin") != "project" || rowString(t, row, "origin_detail") != "preset codex-workers" {
			t.Errorf("agent.workers row = %s", row)
		}
	}

	// Bare show prints the spliced tree — what consumers resolve.
	bare, _, err := runConfigSplit(t, "show", "agent.workers")
	if err != nil || strings.TrimSpace(bare) != "codex" {
		t.Errorf("bare show agent.workers = %q err=%v", bare, err)
	}
}

func TestConfigShowOriginPresetFromEnvAndShadow(t *testing.T) {
	_, home := setupConfigRepo(t, "agent:\n  workers: agy\n")
	writeSystemConfig(t, home, "agent:\n  workers: claude\n")
	t.Setenv("FAB_AGENT_PRESET", "codex-workers")
	out, _, err := runConfigSplit(t, "show", "agent.workers", "--origin")
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if !strings.Contains(lines[0], "codex") || !strings.Contains(lines[0], "$FAB_AGENT_PRESET (preset codex-workers)") || !strings.Contains(lines[0], "(effective)") {
		t.Errorf("effective line = %q\n%s", lines[0], out)
	}
	if len(lines) != 4 || !strings.Contains(lines[1], "claude") || !strings.Contains(lines[1], "(shadowed)") {
		t.Errorf("stack = %s", out)
	}
}

func TestResolveAgentSeesPresetFromConfig(t *testing.T) {
	presetTestRepo(t, "agent:\n  preset: codex-review\n")
	got := agent.DefaultCLIName
	_ = got
	out, err := runAgentArgs(t, "review", "--print")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(out, "codex") {
		t.Errorf("config-chosen preset not applied to the session command: %q", out)
	}
}

// TestConfigRejectsReservedNonePresetKey: `none` can never be a definition, so a
// set or unset of any agent_presets.none[.…] key fails before writing.
func TestConfigRejectsReservedNonePresetKey(t *testing.T) {
	repo, _ := setupConfigRepo(t, "project:\n  name: test\n")
	path := filepath.Join(repo, "fab", "project", "config.yaml")
	before, _ := os.ReadFile(path)
	for _, args := range [][]string{
		{"set", "agent_presets.none.workers", "codex"},
		{"set", "agent_presets.none.profiles.review.cli", "codex"},
		{"set", "agent_presets.none.workers", "codex", "--system"},
		{"unset", "agent_presets.none.workers"},
		{"unset", "agent_presets.none"},
	} {
		_, _, err := runConfigSplit(t, args...)
		if err == nil || !strings.Contains(err.Error(), "reserved") {
			t.Errorf("%v: want a reserved-name error, got %v", args, err)
		}
	}
	if after, _ := os.ReadFile(path); string(after) != string(before) {
		t.Errorf("a refused key modified the project config:\n%s", after)
	}
	// Names merely containing "none" stay legal.
	if _, _, err := runConfigSplit(t, "set", "agent_presets.nonesuch.workers", "agy"); err != nil {
		t.Errorf("agent_presets.nonesuch must be legal: %v", err)
	}
}
