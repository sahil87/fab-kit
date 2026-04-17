package main

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestClassifyProcess(t *testing.T) {
	tests := []struct {
		comm     string
		expected string
	}{
		{"claude", "agent"},
		{"Claude", "agent"},
		{"CLAUDE", "agent"},
		{"claude-code", "agent"},
		{"Claude-Code", "agent"},
		{"node", "node"},
		{"Node", "node"},
		{"git", "git"},
		{"gh", "git"},
		{"GH", "git"},
		{"zsh", "other"},
		{"bash", "other"},
		{"python", "other"},
		{"", "other"},
	}

	for _, tc := range tests {
		t.Run(tc.comm, func(t *testing.T) {
			result := ClassifyProcess(tc.comm)
			if result != tc.expected {
				t.Errorf("ClassifyProcess(%q) = %q, want %q", tc.comm, result, tc.expected)
			}
		})
	}
}

func TestHasAgentInTree(t *testing.T) {
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
