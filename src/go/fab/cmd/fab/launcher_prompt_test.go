package main

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/shellquote"
	"github.com/spf13/cobra"
)

// Exercise each actual launcher with fake external commands. The captured
// window command is then parsed by a shell with fake agent functions, proving
// its final argv retains the prompt's dollar sign and metacharacters.
func TestSkillPromptLaunchers(t *testing.T) {
	for _, provider := range []string{"codex", "claude", "agy", "kimi", "custom", "missing"} {
		for _, launcher := range []string{"new", "switch", "operator"} {
			t.Run(provider+"/"+launcher, func(t *testing.T) {
				stubNoRK(t)
				root, change := batchSwitchFixture(t, "claude")
				content := "Fix $HOME and $(printf EXPANDED) `printf EXPANDED` 'quotes'"
				if err := os.WriteFile(filepath.Join(root, "fab", "backlog.md"), []byte("- [ ] [ab12] 2026-09-08: "+content+"\n"), 0o644); err != nil {
					t.Fatal(err)
				}
				cfg := fmt.Sprintf("agent:\n  session: %s\nproviders:\n  custom:\n    interactive_command: custom\n", provider)
				if err := os.WriteFile(filepath.Join(root, "fab", "project", "config.yaml"), []byte(cfg), 0o644); err != nil {
					t.Fatal(err)
				}
				capture := filepath.Join(t.TempDir(), "window-command")
				bin := t.TempDir()
				for name, body := range map[string]string{
					"wt":   "printf '%s\\n' /fake/worktree",
					"git":  `case "$1" in rev-parse) printf '%s\n' "$PWD";; show-ref) exit 1;; esac`,
					"tmux": `if [ "$1" = new-window ]; then for arg do last=$arg; done; printf '%s' "$last" > ` + shellquote.Single(capture) + `; fi`,
				} {
					if err := os.WriteFile(filepath.Join(bin, name), []byte("#!/bin/sh\n"+body+"\n"), 0o755); err != nil {
						t.Fatal(err)
					}
				}
				t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
				var cmd *cobra.Command
				skill, arguments := "fab-operator", ""
				switch launcher {
				case "new":
					cmd, skill, arguments = batchNewCmd(), "fab-new", content
					cmd.SetArgs([]string{"ab12"})
				case "switch":
					cmd, skill, arguments = batchSwitchCmd(), "fab-switch", change
					cmd.SetArgs([]string{change})
				default:
					cmd = operatorCmd()
					cmd.SetArgs([]string{})
				}
				cmd.SetOut(&bytes.Buffer{})
				cmd.SetErr(&bytes.Buffer{})
				if err := cmd.Execute(); err != nil {
					t.Fatal(err)
				}
				windowCommand, err := os.ReadFile(capture)
				if err != nil {
					t.Fatal(err)
				}
				const suffix = `; exec "$SHELL"`
				if !strings.HasSuffix(string(windowCommand), suffix) {
					t.Fatalf("missing shell fallback: %s", windowCommand)
				}
				prefix := "/"
				if provider == "codex" {
					prefix = "$"
				}
				want := prefix + skill
				if arguments != "" {
					want += " " + arguments
				}
				prelude := ""
				for _, name := range []string{"claude", "codex", "agy", "kimi", "custom"} {
					prelude += name + `() { for arg do last=$arg; done; printf '%s' "$last"; }; `
				}
				got, err := exec.Command("/bin/sh", "-c", prelude+strings.TrimSuffix(string(windowCommand), suffix)).Output()
				if err != nil || string(got) != want {
					t.Fatalf("launcher prompt = %q, error %v; want %q; command %s", got, err, want, windowCommand)
				}
			})
		}
	}
}
