package main

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

func doctorCmd() *cobra.Command {
	var porcelain bool

	cmd := &cobra.Command{
		Use:   "doctor",
		Short: "Validate fab-kit prerequisites",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			failures := runDoctorChecks(cmd, porcelain)
			os.Exit(failures)
			return nil
		},
	}

	cmd.Flags().BoolVar(&porcelain, "porcelain", false, "Only print errors (no passes, hints, or summary)")
	return cmd
}

// checkResult holds the outcome of a single prerequisite check.
type checkResult struct {
	passed  bool
	name    string
	version string
	message string // failure message
	hints   []string
}

// runDoctorChecks executes all 7 prerequisite checks and returns the failure count.
func runDoctorChecks(cmd *cobra.Command, porcelain bool) int {
	w := cmd.OutOrStdout()

	if !porcelain {
		fmt.Fprintln(w, "fab-doctor: checking prerequisites...")
	}

	checks := []checkResult{
		checkGit(),
		checkFab(),
		checkBash(),
		checkYq(),
		checkJq(),
		checkGh(),
		checkDirenv(),
	}

	failures := 0
	for _, c := range checks {
		if c.passed {
			if !porcelain {
				fmt.Fprintf(w, "  \u2713 %s\n", c.version)
			}
		} else {
			failures++
			if porcelain {
				fmt.Fprintln(w, c.message)
			} else {
				fmt.Fprintf(w, "  \u2717 %s\n", c.message)
				for _, h := range c.hints {
					fmt.Fprintf(w, "    %s\n", h)
				}
			}
		}
	}

	if !porcelain {
		total := len(checks)
		passed := total - failures
		fmt.Fprintln(w)
		if failures == 0 {
			fmt.Fprintf(w, "%d/%d checks passed.\n", passed, total)
		} else if failures == 1 {
			fmt.Fprintf(w, "%d/%d checks passed. %d issue found.\n", passed, total, failures)
		} else {
			fmt.Fprintf(w, "%d/%d checks passed. %d issues found.\n", passed, total, failures)
		}
	}

	return failures
}

func checkGit() checkResult {
	path, err := exec.LookPath("git")
	if err != nil || path == "" {
		return checkResult{name: "git", message: "git \u2014 not found", hints: []string{"Install: brew install git"}}
	}
	ver := cmdOutput("git", "--version")
	ver = strings.TrimPrefix(ver, "git version ")
	return checkResult{passed: true, name: "git", version: "git " + ver}
}

func checkFab() checkResult {
	path, err := exec.LookPath("fab")
	if err != nil || path == "" {
		return checkResult{name: "fab", message: "fab \u2014 not found", hints: []string{"Install: brew install fab-kit"}}
	}
	ver := cmdOutput("fab", "--version")
	if ver == "" {
		ver = "unknown"
	}
	return checkResult{passed: true, name: "fab", version: "fab " + ver}
}

func checkBash() checkResult {
	path, err := exec.LookPath("bash")
	if err != nil || path == "" {
		return checkResult{name: "bash", message: "bash \u2014 not found", hints: []string{"Install: brew install bash"}}
	}
	out := cmdOutput("bash", "--version")
	ver := parseBashVersion(out)
	return checkResult{passed: true, name: "bash", version: "bash " + ver}
}

func checkYq() checkResult {
	path, err := exec.LookPath("yq")
	if err != nil || path == "" {
		return checkResult{name: "yq", message: "yq \u2014 not found", hints: []string{"Install: brew install yq"}}
	}
	raw := cmdOutput("yq", "--version")
	ver := parseYqVersion(raw)
	if ver == "" {
		return checkResult{
			name:    "yq",
			message: fmt.Sprintf("yq \u2014 could not parse version from: %s", raw),
			hints:   []string{"Expected yq v4+ (Mike Farah). Install: brew install yq"},
		}
	}
	major := majorVersion(ver)
	if major < 4 {
		return checkResult{
			name:    "yq",
			message: fmt.Sprintf("yq %s \u2014 version 4+ required (you have the Python version)", ver),
			hints:   []string{"Install the Go version: brew install yq"},
		}
	}
	return checkResult{passed: true, name: "yq", version: "yq " + ver}
}

func checkJq() checkResult {
	path, err := exec.LookPath("jq")
	if err != nil || path == "" {
		return checkResult{name: "jq", message: "jq \u2014 not found", hints: []string{"Install: brew install jq"}}
	}
	ver := cmdOutput("jq", "--version")
	ver = strings.TrimPrefix(ver, "jq-")
	return checkResult{passed: true, name: "jq", version: "jq " + ver}
}

func checkGh() checkResult {
	path, err := exec.LookPath("gh")
	if err != nil || path == "" {
		return checkResult{name: "gh", message: "gh \u2014 not found", hints: []string{"Install: brew install gh"}}
	}
	out := cmdOutput("gh", "--version")
	ver := parseGhVersion(out)
	return checkResult{passed: true, name: "gh", version: "gh " + ver}
}

func checkDirenv() checkResult {
	path, err := exec.LookPath("direnv")
	if err != nil || path == "" {
		return checkResult{name: "direnv", message: "direnv \u2014 not found", hints: []string{"Install: brew install direnv"}}
	}
	ver := cmdOutput("direnv", "version")

	shellName := detectShell()

	switch shellName {
	case "zsh":
		if checkDirenvHookZsh() {
			return checkResult{passed: true, name: "direnv", version: fmt.Sprintf("direnv %s (%s hook active)", ver, shellName)}
		}
		return checkResult{
			name:    "direnv",
			message: fmt.Sprintf("direnv shell hook not detected for %s", shellName),
			hints: []string{
				fmt.Sprintf("Add the following to your ~/.%src (or equivalent):", shellName),
				fmt.Sprintf("  eval \"$(direnv hook %s)\"", shellName),
			},
		}
	case "bash":
		if checkDirenvHookBash() {
			return checkResult{passed: true, name: "direnv", version: fmt.Sprintf("direnv %s (%s hook active)", ver, shellName)}
		}
		return checkResult{
			name:    "direnv",
			message: fmt.Sprintf("direnv shell hook not detected for %s", shellName),
			hints: []string{
				fmt.Sprintf("Add the following to your ~/.%src (or equivalent):", shellName),
				fmt.Sprintf("  eval \"$(direnv hook %s)\"", shellName),
			},
		}
	default:
		return checkResult{passed: true, name: "direnv", version: fmt.Sprintf("direnv %s (hook check skipped \u2014 %s not supported)", ver, shellName)}
	}
}

// cmdOutput runs a command and returns trimmed stdout, or empty on error.
func cmdOutput(name string, args ...string) string {
	out, err := exec.Command(name, args...).Output()
	if err != nil {
		return ""
	}
	// Return just the first line, trimmed
	s := strings.TrimSpace(string(out))
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return s[:idx]
	}
	return s
}

// parseBashVersion extracts the version number from bash --version output.
func parseBashVersion(s string) string {
	re := regexp.MustCompile(`version\s+([^\s(]+)`)
	m := re.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	return "unknown"
}

// parseYqVersion extracts a version number from yq --version output.
// Handles both "yq (https://...) version v4.x.y" and "yq version 4.x.y".
func parseYqVersion(s string) string {
	re := regexp.MustCompile(`(?:version\s+v?)(\d+\.\d+\.\d+)`)
	m := re.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	return ""
}

// parseGhVersion extracts version from "gh version X.Y.Z ..." output.
func parseGhVersion(s string) string {
	re := regexp.MustCompile(`version\s+([^\s]+)`)
	m := re.FindStringSubmatch(s)
	if len(m) >= 2 {
		return m[1]
	}
	return "unknown"
}

// majorVersion parses the major version from a "X.Y.Z" string.
func majorVersion(ver string) int {
	parts := strings.SplitN(ver, ".", 2)
	n, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0
	}
	return n
}

// detectShell returns the basename of $SHELL.
func detectShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return "bash"
	}
	parts := strings.Split(shell, "/")
	return parts[len(parts)-1]
}

// checkDirenvHookZsh checks if the direnv hook is active in zsh.
func checkDirenvHookZsh() bool {
	cmd := exec.Command("zsh", "-i", "-c", "typeset -f _direnv_hook")
	err := cmd.Run()
	return err == nil
}

// checkDirenvHookBash checks if the direnv hook is active in bash.
func checkDirenvHookBash() bool {
	cmd := exec.Command("bash", "-i", "-c", `[[ "${PROMPT_COMMAND:-}" == *direnv* ]]`)
	err := cmd.Run()
	return err == nil
}
