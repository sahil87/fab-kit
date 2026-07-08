package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupInitRepo creates a temp repo rooted at a fab/ dir and chdirs into it, so
// resolve.FabRoot() resolves to <tmp>/fab. Returns the fab root path. HOME is
// isolated by the package TestMain.
func setupInitRepo(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	fabRoot := filepath.Join(repo, "fab")
	if err := os.MkdirAll(filepath.Join(fabRoot, "project"), 0o755); err != nil {
		t.Fatal(err)
	}
	chdirTestEnv(t, repo, map[string]string{})
	return fabRoot
}

// TestConfigUpgradeCommand drives `fab config upgrade` end to end: it appends a
// managed fence to a legacy config.yaml, preserves the live keys, exits 0, and a
// second run is a byte-identical no-op reporting "already up to date".
func TestConfigUpgradeCommand(t *testing.T) {
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")
	legacy := "project:\n    name: t\n    description: d\n\nlegacy_mode: true\n"
	if err := os.WriteFile(cfgPath, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}

	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"upgrade"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("`config upgrade` returned an error: %v", err)
	}

	written, _ := os.ReadFile(cfgPath)
	got := string(written)
	if !strings.Contains(got, "# >>> fab reference") {
		t.Errorf("upgrade must append a managed fence, got:\n%s", got)
	}
	if !strings.Contains(got, "name: t") {
		t.Error("upgrade must preserve the live project field")
	}
	if !strings.Contains(got, "#   legacy_mode: true") {
		t.Error("upgrade must park the unknown live key below the fence")
	}
	if !strings.Contains(out.String(), "Upgraded") {
		t.Errorf("first run should report Upgraded, got: %q", out.String())
	}

	// Second run: no-op, byte-identical.
	cmd2 := configCmd()
	var out2 strings.Builder
	cmd2.SetOut(&out2)
	cmd2.SetErr(&out2)
	cmd2.SetArgs([]string{"upgrade"})
	if err := cmd2.Execute(); err != nil {
		t.Fatalf("`config upgrade` (2nd) returned an error: %v", err)
	}
	written2, _ := os.ReadFile(cfgPath)
	if string(written2) != got {
		t.Error("second `config upgrade` churned the file (must be idempotent)")
	}
	if !strings.Contains(out2.String(), "already up to date") {
		t.Errorf("second run should report already up to date, got: %q", out2.String())
	}

	// Extra positional arg rejected (cobra.NoArgs).
	cmd3 := configCmd()
	var errBuf strings.Builder
	cmd3.SetOut(&errBuf)
	cmd3.SetErr(&errBuf)
	cmd3.SetArgs([]string{"upgrade", "extra"})
	if err := cmd3.Execute(); err == nil {
		t.Error("`config upgrade extra` should be rejected (cobra.NoArgs)")
	}
}

// TestConfigInitProjectCommand drives `fab config init --project`: it generates a
// config.yaml with the seeded identity fields live and the managed fence, does NOT
// pin agent.tiers, and refuses to overwrite an existing file.
func TestConfigInitProjectCommand(t *testing.T) {
	fabRoot := setupInitRepo(t)
	cfgPath := filepath.Join(fabRoot, "project", "config.yaml")

	cmd := configCmd()
	var out strings.Builder
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs([]string{"init", "--project",
		"--name", "MyProj", "--description", "a demo",
		"--source-path", "src/", "--test-path", "**/*_test.go"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("`config init --project` returned an error: %v", err)
	}

	data, err := os.ReadFile(cfgPath)
	if err != nil {
		t.Fatalf("expected a generated config.yaml: %v", err)
	}
	got := string(data)
	for _, want := range []string{"name: MyProj", "description: a demo", "source_paths:", "- src/", "test_paths:", "# >>> fab reference"} {
		if !strings.Contains(got, want) {
			t.Errorf("generated config missing %q:\n%s", want, got)
		}
	}
	// agent.tiers must NOT be pinned live (presence=intent) — the only `agent:`
	// mention allowed is the commented fence scaffold.
	for _, ln := range strings.Split(got, "\n") {
		if strings.HasPrefix(ln, "agent:") {
			t.Error("init --project must not pin agent.tiers live")
		}
	}

	// Refuses to overwrite.
	cmd2 := configCmd()
	var out2 strings.Builder
	cmd2.SetOut(&out2)
	cmd2.SetErr(&out2)
	cmd2.SetArgs([]string{"init", "--project", "--name", "Other"})
	if err := cmd2.Execute(); err == nil {
		t.Error("init --project must refuse to overwrite an existing config.yaml")
	}
}

// TestConfigInitBareAndBothFlagsRejected: bare `fab config init` is a usage error,
// and passing both --system and --project errors.
func TestConfigInitBareAndBothFlagsRejected(t *testing.T) {
	setupInitRepo(t)

	bare := configCmd()
	var b1 strings.Builder
	bare.SetOut(&b1)
	bare.SetErr(&b1)
	bare.SetArgs([]string{"init"})
	if err := bare.Execute(); err == nil {
		t.Error("bare `config init` should be a usage error")
	}

	both := configCmd()
	var b2 strings.Builder
	both.SetOut(&b2)
	both.SetErr(&b2)
	both.SetArgs([]string{"init", "--system", "--project"})
	if err := both.Execute(); err == nil {
		t.Error("`config init --system --project` should error (mutually exclusive)")
	}
}
