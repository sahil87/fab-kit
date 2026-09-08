package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/shellquote"
	"github.com/spf13/cobra"
)

func TestSkillPromptCLI(t *testing.T) {
	chdirTestEnv(t, t.TempDir(), map[string]string{"FAB_AGENT_SESSION": "codex"})
	for _, tc := range []struct {
		args []string
		want string
	}{
		{[]string{"--provider", "codex", "fab-fff", "ab12"}, "$fab-fff ab12\n"},
		{[]string{"--provider", "custom", "fab-new", "  $HOME\n /fab-ff\n"}, "/fab-new   $HOME\n /fab-ff\n\n"},
		{[]string{"fab-fff"}, "/fab-fff\n"}, // Never inherit the sender's config/env.
		{[]string{"--provider", "codex", "fab-ff", "--", "--light ab12"}, "$fab-ff --light ab12\n"},
	} {
		var out, errOut bytes.Buffer
		if code := run(append([]string{"skill-prompt"}, tc.args...), &out, &errOut); code != 0 || out.String() != tc.want || errOut.Len() != 0 {
			t.Errorf("%v: exit %d, stdout %q, stderr %q; want %q", tc.args, code, out.String(), errOut.String(), tc.want)
		}
	}
	var out, errOut bytes.Buffer
	if code := run([]string{"skill-prompt", "--json", "--provider", "codex", "fab-fff", "ab12"}, &out, &errOut); code != 0 {
		t.Fatalf("json: exit %d: %s", code, &errOut)
	}
	var got map[string]string
	if err := json.Unmarshal(out.Bytes(), &got); err != nil || got["provider"] != "codex" || got["skill"] != "fab-fff" || got["prompt"] != "$fab-fff ab12" {
		t.Fatalf("json = %s, parse error %v", &out, err)
	}
}

func TestSkillPromptUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{}, {"a", "b", "c"}, {"/fab-fff"}, {"$fab-fff"}, {""}, {"a\nb"}, {"a b"},
		{"../skill"}, {"fab-fff", "--json", "--shell-quote"},
		{"fab-fff", "--provider", "codex", "--repo", "/tmp"}, {"fab-fff", "--repo="},
	} {
		var out, errOut bytes.Buffer
		if code := run(append([]string{"skill-prompt"}, args...), &out, &errOut); code != 2 || out.Len() != 0 || errOut.Len() == 0 {
			t.Errorf("%v: exit %d, stdout %q, stderr %q", args, code, out.String(), errOut.String())
		}
	}
}

func TestSkillPromptRepoProvider(t *testing.T) {
	for _, tc := range []struct{ name, config, want string }{
		{"codex", "agent:\n  session: codex\n", "$fab-fff ab12\n"},
		{"role override", "agent:\n  session: claude\n  profiles:\n    default:\n      provider: codex\n", "$fab-fff ab12\n"},
		{"interactive only", "agent:\n  session: custom\nproviders:\n  custom:\n    interactive_command: custom\n", "/fab-fff ab12\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root, _ := batchSwitchFixture(t, "claude")
			if err := os.WriteFile(filepath.Join(root, "fab", "project", "config.yaml"), []byte(tc.config), 0o644); err != nil {
				t.Fatal(err)
			}
			// Prove this reads the TARGET rather than the caller's repo, and
			// works without any reachable pane/native/headless dispatch rung.
			chdirTestEnv(t, t.TempDir(), map[string]string{"TMUX": ""})
			var out, errOut bytes.Buffer
			if code := run([]string{"skill-prompt", "--repo", root, "fab-fff", "ab12"}, &out, &errOut); code != 0 || out.String() != tc.want {
				t.Fatalf("exit %d, stdout %q, stderr %q; want %q", code, out.String(), errOut.String(), tc.want)
			}
		})
	}
}

func TestSkillPromptShellRoundTrip(t *testing.T) {
	arguments := "  $HOME $(printf EXPANDED) `printf EXPANDED` 'single' \"double\"; /fab-ff\nnext\n"
	var out, errOut bytes.Buffer
	if code := run([]string{"skill-prompt", "--provider", "codex", "--shell-quote", "fab-new", arguments}, &out, &errOut); code != 0 {
		t.Fatalf("exit %d: %s", code, &errOut)
	}
	// Match the operator recipe: expand the quoted token from a variable into
	// the outer shell's command string, then let the window shell parse it once.
	cmd := exec.Command("/bin/sh", "-c", `sh -c "printf '%s' $FAB_TEST_PROMPT_TOKEN"`)
	cmd.Env = append(os.Environ(), "FAB_TEST_PROMPT_TOKEN="+strings.TrimSuffix(out.String(), "\n"))
	got, err := cmd.Output()
	if err != nil || string(got) != "$fab-new "+arguments {
		t.Fatalf("shell round trip = %q, error %v", got, err)
	}
	// Existing-pane routing decodes ONLY the renderer's quoted token before
	// passing a literal argument to the sender; raw command substitution would
	// discard the argument's trailing newlines.
	cmd = exec.Command("/bin/sh", "-c", `eval "fab_skill_prompt=$FAB_TEST_PROMPT_TOKEN"; printf '%s' "$fab_skill_prompt"`)
	cmd.Env = append(os.Environ(), "FAB_TEST_PROMPT_TOKEN="+strings.TrimSuffix(out.String(), "\n"))
	got, err = cmd.Output()
	if err != nil || string(got) != "$fab-new "+arguments {
		t.Fatalf("existing-pane round trip = %q, error %v", got, err)
	}
}

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
