package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

func TestClassifyProcess(t *testing.T) {
	agents := agentBinaryNames(nil)
	tests := []struct {
		name     string
		comm     string
		cmdline  string
		expected string
	}{
		{"comm claude", "claude", "claude --flag", "agent"},
		{"comm path and case", "/opt/bin/Claude-Code", "claude-code --flag", "agent"},
		{"built-in codex", "codex", "codex", "agent"},
		{"built-in agy", "agy", "agy", "agent"},
		{"built-in kimi", "kimi", "kimi", "agent"},
		{"cmdline argv zero fallback", "node", "/opt/bin/claude --flag", "agent"},
		{"cmdline argv one fallback", "node", "node /opt/bin/codex --flag", "agent"},
		{"prompt text outside bounded fallback", "node", "node runner ask claude for help", "node"},
		{"node", "node", "node service.js", "node"},
		{"git", "git", "git status", "git"},
		{"gh", "GH", "gh pr view", "git"},
		{"shell", "zsh", "/bin/zsh", "other"},
		{"other", "python", "python worker.py", "other"},
		{"empty", "", "", "other"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := classifyProcess(tc.comm, tc.cmdline, agents)
			if result != tc.expected {
				t.Errorf("classifyProcess(%q, %q) = %q, want %q", tc.comm, tc.cmdline, result, tc.expected)
			}
		})
	}
}

func TestAgentBinaryNames(t *testing.T) {
	cfg := &config.Config{Providers: map[string]config.ProviderConfig{
		"custom":        {InteractiveCommand: "TOKEN='a b' /opt/agents/my-agent --model m"},
		"path-equals":   {InteractiveCommand: "/opt/agents/my=agent --model m"},
		"shell-wrapper": {InteractiveCommand: "FAB_TOKEN=x /bin/zsh -lc my-agent"},
	}}
	names := agentBinaryNames(cfg)

	for _, name := range []string{"claude", "claude-code", "codex", "agy", "kimi", "my-agent", "my=agent"} {
		if !names[name] {
			t.Errorf("agentBinaryNames omitted %q: %v", name, names)
		}
	}
	for _, shell := range []string{"sh", "bash", "zsh", "fish"} {
		if names[shell] {
			t.Errorf("agentBinaryNames included shell %q: %v", shell, names)
		}
	}
}

// TestLoadAgentBinaryNames_NoProjectAppliesSystemLayer: outside a fab project
// the loader falls back to the project-free cascade rather than built-ins only,
// so a system-layer (~/.fab-kit) provider definition still counts as liveness
// evidence.
func TestLoadAgentBinaryNames_NoProjectAppliesSystemLayer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.MkdirAll(filepath.Join(home, ".fab-kit"), 0o755); err != nil {
		t.Fatal(err)
	}
	system := "providers:\n  sysagent:\n    interactive_command: /opt/agents/sys-agent --auto\n"
	if err := os.WriteFile(filepath.Join(home, ".fab-kit", "config.yaml"), []byte(system), 0o644); err != nil {
		t.Fatal(err)
	}

	origWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origWd) })
	if _, err := resolve.FabRoot(); err == nil {
		t.Fatal("test cwd unexpectedly resolves a fab project — precondition broken")
	}

	names := loadAgentBinaryNames()
	if !names["sys-agent"] {
		t.Errorf("loadAgentBinaryNames() = %v, want the system-layer provider binary %q outside a project", names, "sys-agent")
	}
}

func TestInteractiveCommandBinary(t *testing.T) {
	tests := []struct {
		name    string
		command string
		want    string
	}{
		{"quoted assignment value", "TOKEN='a b' /opt/agents/my-agent --flag", "my-agent"},
		{"equals in executable path", "/opt/agents/my=agent --flag", "my=agent"},
		{"plain assignment prefix", "FOO=1 kimi --auto", "kimi"},
		{"multiple assignment prefixes", "FOO=1 _BAR_2='x y' codex", "codex"},
		{"quoted assignment-looking word is executable", "'FOO=1' /opt/agents/my-agent", "foo=1"},
		{"assignment only", "FOO=1 BAR='x y'", ""},
		{"empty", "", ""},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := interactiveCommandBinary(tc.command); got != tc.want {
				t.Errorf("interactiveCommandBinary(%q) = %q, want %q", tc.command, got, tc.want)
			}
		})
	}
}

// TestParsePSCmdlines verifies the `ps -axo pid=,args=` parser that backs
// the darwin single-pass cmdline join (F37): numeric-first PID, remainder is
// args (robust against spaces in command paths), malformed lines skipped.
func TestParsePSCmdlines(t *testing.T) {
	t.Run("typical ps output", func(t *testing.T) {
		out := "    1 /sbin/launchd\n" +
			"  345 /Applications/Visual Studio Code.app/Contents/MacOS/Electron --type=renderer\n" +
			" 8210 claude --dangerously-skip-permissions\n"
		got := parsePSCmdlines(out)
		want := map[int]string{
			1:    "/sbin/launchd",
			345:  "/Applications/Visual Studio Code.app/Contents/MacOS/Electron --type=renderer",
			8210: "claude --dangerously-skip-permissions",
		}
		if len(got) != len(want) {
			t.Fatalf("parsed %d entries, want %d: %v", len(got), len(want), got)
		}
		for pid, args := range want {
			if got[pid] != args {
				t.Errorf("parsePSCmdlines[%d] = %q, want %q", pid, got[pid], args)
			}
		}
	})

	t.Run("pid with no args yields empty cmdline", func(t *testing.T) {
		got := parsePSCmdlines("  42\n")
		if v, ok := got[42]; !ok || v != "" {
			t.Errorf("parsePSCmdlines[42] = (%q, %t), want (\"\", true)", v, ok)
		}
	})

	t.Run("malformed lines are skipped", func(t *testing.T) {
		got := parsePSCmdlines("PID ARGS\nnotanumber /bin/x\n  7 /bin/y\n")
		if len(got) != 1 || got[7] != "/bin/y" {
			t.Errorf("parsePSCmdlines = %v, want only {7: /bin/y}", got)
		}
	})

	t.Run("empty input yields empty map", func(t *testing.T) {
		if got := parsePSCmdlines(""); len(got) != 0 {
			t.Errorf("parsePSCmdlines(\"\") = %v, want empty", got)
		}
	})
}

func TestHasAgentInTree(t *testing.T) {
	t.Run("provider extension drives has_agent", func(t *testing.T) {
		agents := agentBinaryNames(nil)
		nodes := []ProcessNode{
			{
				PID: 100, Comm: "zsh", Classification: classifyProcess("zsh", "/bin/zsh", agents),
				Children: []ProcessNode{
					{PID: 200, Comm: "kimi", Cmdline: "kimi --auto", Classification: classifyProcess("kimi", "kimi --auto", agents)},
				},
			},
		}
		if !hasAgentInTree(nodes) {
			t.Error("expected provider-derived classification to set has_agent")
		}
	})

	t.Run("agent at root", func(t *testing.T) {
		nodes := []ProcessNode{
			{PID: 100, Comm: "claude", Classification: "agent"},
		}
		if !hasAgentInTree(nodes) {
			t.Error("expected hasAgentInTree to be true")
		}
	})

	t.Run("agent nested", func(t *testing.T) {
		nodes := []ProcessNode{
			{
				PID: 100, Comm: "zsh", Classification: "other",
				Children: []ProcessNode{
					{PID: 200, Comm: "claude", Classification: "agent"},
				},
			},
		}
		if !hasAgentInTree(nodes) {
			t.Error("expected hasAgentInTree to be true for nested agent")
		}
	})

	t.Run("no agent", func(t *testing.T) {
		nodes := []ProcessNode{
			{
				PID: 100, Comm: "zsh", Classification: "other",
				Children: []ProcessNode{
					{PID: 200, Comm: "node", Classification: "node"},
				},
			},
		}
		if hasAgentInTree(nodes) {
			t.Error("expected hasAgentInTree to be false")
		}
	})

	t.Run("empty tree", func(t *testing.T) {
		if hasAgentInTree(nil) {
			t.Error("expected hasAgentInTree to be false for empty tree")
		}
	})

	t.Run("deeply nested agent", func(t *testing.T) {
		nodes := []ProcessNode{
			{
				PID: 100, Comm: "zsh", Classification: "other",
				Children: []ProcessNode{
					{
						PID: 200, Comm: "node", Classification: "node",
						Children: []ProcessNode{
							{PID: 300, Comm: "claude", Classification: "agent"},
						},
					},
				},
			},
		}
		if !hasAgentInTree(nodes) {
			t.Error("expected hasAgentInTree to be true for deeply nested agent")
		}
	})
}

func TestProcessJSONStructure(t *testing.T) {
	t.Run("json output has correct fields", func(t *testing.T) {
		out := processJSON{
			Pane:    "%5",
			PanePID: 12345,
			Processes: []ProcessNode{
				{
					PID: 12345, PPID: 1, Comm: "zsh", Cmdline: "/bin/zsh",
					Classification: "other",
					Children: []ProcessNode{
						{
							PID: 12350, PPID: 12345, Comm: "claude",
							Cmdline:        "claude --dangerously-skip-permissions",
							Classification: "agent",
							Children:       []ProcessNode{},
						},
					},
				},
			},
			HasAgent: true,
		}

		var buf bytes.Buffer
		enc := json.NewEncoder(&buf)
		enc.SetIndent("", "  ")
		if err := enc.Encode(out); err != nil {
			t.Fatal(err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
			t.Fatal(err)
		}

		expectedFields := []string{"pane", "pane_pid", "processes", "has_agent"}
		for _, field := range expectedFields {
			if _, ok := result[field]; !ok {
				t.Errorf("JSON output missing field %q", field)
			}
		}

		if result["has_agent"] != true {
			t.Errorf("has_agent should be true, got %v", result["has_agent"])
		}
	})

	t.Run("process node has correct fields", func(t *testing.T) {
		node := ProcessNode{
			PID: 12345, PPID: 1, Comm: "zsh", Cmdline: "/bin/zsh",
			Classification: "other", Children: []ProcessNode{},
		}

		data, err := json.Marshal(node)
		if err != nil {
			t.Fatal(err)
		}

		var result map[string]interface{}
		if err := json.Unmarshal(data, &result); err != nil {
			t.Fatal(err)
		}

		expectedFields := []string{"pid", "ppid", "comm", "cmdline", "classification", "children"}
		for _, field := range expectedFields {
			if _, ok := result[field]; !ok {
				t.Errorf("process node JSON missing field %q", field)
			}
		}
	})
}

func TestPrintProcessTree(t *testing.T) {
	t.Run("human readable output", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		nodes := []ProcessNode{
			{
				PID: 12345, PPID: 1, Comm: "zsh", Classification: "other",
				Children: []ProcessNode{
					{
						PID: 12350, PPID: 12345, Comm: "claude", Classification: "agent",
						Children: []ProcessNode{},
					},
				},
			},
		}

		printProcessTree(cmd, "%5", 12345, nodes, true)

		output := buf.String()
		if !strings.Contains(output, "Pane %5") {
			t.Errorf("output missing pane ID: %q", output)
		}
		if !strings.Contains(output, "12345") {
			t.Errorf("output missing PID: %q", output)
		}
		if !strings.Contains(output, "claude") {
			t.Errorf("output missing process name: %q", output)
		}
		if !strings.Contains(output, "[agent]") {
			t.Errorf("output missing classification: %q", output)
		}
		if !strings.Contains(output, "Agent process detected") {
			t.Errorf("output missing agent detected message: %q", output)
		}
	})

	t.Run("no agent message when no agent", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		nodes := []ProcessNode{
			{PID: 100, Comm: "zsh", Classification: "other", Children: []ProcessNode{}},
		}

		printProcessTree(cmd, "%5", 100, nodes, false)

		output := buf.String()
		if strings.Contains(output, "Agent process detected") {
			t.Errorf("output should not contain agent detected message: %q", output)
		}
	})

	t.Run("indentation for nested processes", func(t *testing.T) {
		var buf bytes.Buffer
		cmd := &cobra.Command{}
		cmd.SetOut(&buf)

		nodes := []ProcessNode{
			{
				PID: 100, Comm: "zsh", Classification: "other",
				Children: []ProcessNode{
					{
						PID: 200, Comm: "node", Classification: "node",
						Children: []ProcessNode{
							{PID: 300, Comm: "git", Classification: "git", Children: []ProcessNode{}},
						},
					},
				},
			},
		}

		printProcessTree(cmd, "%5", 100, nodes, false)

		output := buf.String()
		lines := strings.Split(strings.TrimRight(output, "\n"), "\n")

		// Check that nested processes are indented
		foundNode := false
		foundGit := false
		for _, line := range lines {
			if strings.Contains(line, "200 node") {
				foundNode = true
				if !strings.HasPrefix(line, "  ") {
					t.Errorf("child process should be indented: %q", line)
				}
			}
			if strings.Contains(line, "300 git") {
				foundGit = true
				if !strings.HasPrefix(line, "    ") {
					t.Errorf("grandchild process should be double-indented: %q", line)
				}
			}
		}
		if !foundNode {
			t.Error("expected to find node process in output")
		}
		if !foundGit {
			t.Error("expected to find git process in output")
		}
	})
}

func TestPaneProcessCmd(t *testing.T) {
	t.Run("requires pane argument", func(t *testing.T) {
		cmd := paneProcessCmd()
		cmd.SetArgs([]string{})
		err := cmd.Execute()
		if err == nil {
			t.Fatal("expected error for missing argument, got nil")
		}
	})

	t.Run("json flag defaults to false", func(t *testing.T) {
		cmd := paneProcessCmd()
		jsonFlag, _ := cmd.Flags().GetBool("json")
		if jsonFlag {
			t.Error("expected json to default to false")
		}
	})
}

func TestPaneProcessServerFlag(t *testing.T) {
	t.Run("--server flag inherited from pane parent", func(t *testing.T) {
		parent := paneCmd()
		var sub *cobra.Command
		for _, c := range parent.Commands() {
			if c.Use == "process <pane>" {
				sub = c
				break
			}
		}
		if sub == nil {
			t.Fatal("paneCmd did not register a process subcommand")
		}
		flag := sub.Flags().Lookup("server")
		if flag == nil {
			flag = sub.InheritedFlags().Lookup("server")
		}
		if flag == nil {
			t.Fatal("expected --server flag to be visible on pane process subcommand")
		}
		if flag.Shorthand != "L" {
			t.Errorf("expected shorthand \"L\", got %q", flag.Shorthand)
		}
		if flag.DefValue != "" {
			t.Errorf("expected empty default, got %q", flag.DefValue)
		}
	})

	t.Run("--server flag value is parsed as string", func(t *testing.T) {
		// Build the full pane command tree so that persistent flag parsing works
		// the way cobra would at runtime.
		parent := paneCmd()
		// Parse --server runKit at the parent level; cobra should propagate it.
		parent.SetArgs([]string{"process", "%5", "--server", "runKit"})
		// We can't actually execute because runPaneProcess calls tmux; just verify
		// parsing the flag doesn't error out before RunE runs.
		// Instead, inspect the persistent flag value after manual Parse.
		if err := parent.ParseFlags([]string{"--server", "runKit"}); err != nil {
			t.Fatalf("ParseFlags(--server runKit) returned error: %v", err)
		}
		val, err := parent.PersistentFlags().GetString("server")
		if err != nil {
			t.Fatalf("GetString(\"server\") error: %v", err)
		}
		if val != "runKit" {
			t.Errorf("expected server=\"runKit\", got %q", val)
		}
	})
}
