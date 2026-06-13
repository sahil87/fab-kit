package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/sahil87/fab-kit/src/go/fab/internal/spawn"
	"github.com/spf13/cobra"
)

func operatorCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "operator",
		Short: "Launch operator in a dedicated tmux tab (singleton)",
		RunE:  runOperator,
	}
	cmd.AddCommand(operatorTickStartCmd(), operatorTimeCmd())
	return cmd
}

func runOperator(cmd *cobra.Command, args []string) error {
	w := cmd.OutOrStdout()

	// Must be inside tmux
	if os.Getenv("TMUX") == "" {
		return fmt.Errorf("not inside a tmux session")
	}

	tabName := "operator"

	// Singleton: exact, server-wide window-name match. The previous
	// `select-window -t operator` guard was wrong on two axes: tmux target
	// resolution falls back to name-prefix and glob matching (any
	// `operator-*` window satisfied it), and it was session-scoped (an
	// operator in another session on the same server was missed, breaking
	// the per-SERVER singleton). Enumerating `list-windows -a` and comparing
	// names exactly fixes both, and distinguishes "window absent" from
	// "tmux error" instead of conflating them.
	windows, stderr, err := pane.RunCmd("tmux", "list-windows", "-a", "-F", "#{window_id}\t#{window_name}")
	if err != nil {
		return pane.StderrError(fmt.Errorf("tmux list-windows: %w", err), stderr)
	}
	if windowID, found := findWindowExact(windows, tabName); found {
		// Window IDs (@N) are server-global and exempt from target-grammar
		// prefix/glob resolution, so selection is exact. select-window makes
		// it the current window of its session; the best-effort switch-client
		// moves the user's client there when the match is in another session
		// (failure ignored — the singleton invariant is already preserved).
		if _, stderr, err := pane.RunCmd("tmux", "select-window", "-t", windowID); err != nil {
			return pane.StderrError(fmt.Errorf("tmux select-window: %w", err), stderr)
		}
		_, _, _ = pane.RunCmd("tmux", "switch-client", "-t", windowID)
		fmt.Fprintf(w, "Switched to existing %s tab.\n", tabName)
		return nil
	}

	// Resolve the new window's working directory. The git repo root is
	// incidental, not essential: it is used ONLY as the tmux window's -c <dir>.
	// The operator is a per-tmux-server singleton for cross-repo coordination, so
	// its natural launch point is a neutral parent directory with no git. Try the
	// repo root (preserves "start in repo root" inside a repo); on failure fall
	// back to os.Getwd() ("start where I am"). Error only when both fail — a
	// failing os.Getwd() means a genuinely broken environment.
	windowDir, err := gitRepoRoot()
	if err != nil {
		windowDir, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("cannot determine working directory: %w", err)
		}
	}

	// Read the spawn command from the project config when a fab/ project is
	// resolvable. The operator is a per-tmux-server cross-repo coordinator and may
	// be launched from a neutral directory with no fab/ project (e.g. ~/code), so a
	// missing project is NOT an error — fall back to the default spawn command.
	// (The doing-tier resolution below likewise degrades to its built-in default
	// when no project is resolvable, so a no-fab/ launch is fully defaulted.)
	spawnCmd := spawn.DefaultSpawnCommand
	if fabRoot, err := resolve.FabRoot(); err == nil {
		spawnCmd = spawn.Command(filepath.Join(fabRoot, "project", "config.yaml"))
	}

	// Resolve the doing-tier {model, effort} so the operator launches its
	// coordinating agent on a deliberately-chosen model. We probe `apply` (NOT a
	// tier name) because `fab resolve-agent` takes a STAGE, and `apply` is the
	// canonical member of the fab-owned, FIXED stage→tier mapping's `doing` tier.
	// This couples to that internal mapping on purpose: if `apply` is ever
	// remapped to a different tier, this dependency must surface here. On ANY
	// failure — the command erroring (an installed `fab` predating resolve-agent,
	// or no resolvable fab project) or empty/unparseable output — resolveDoingProfile
	// falls back to the in-process built-in doing default (pass "" on error).
	out, _, resolveErr := pane.RunCmd("fab", "resolve-agent", "apply")
	if resolveErr != nil {
		out = ""
	}
	profile := resolveDoingProfile(out)
	spawnCmd = spawn.WithProfile(spawnCmd, profile.Model, profile.Effort)

	// Create new tab running the operator skill
	shellCmd := fmt.Sprintf("%s '/fab-operator'", spawnCmd)
	if _, stderr, err := pane.RunCmd("tmux", "new-window", "-c", windowDir, "-n", tabName, shellCmd); err != nil {
		return pane.StderrError(fmt.Errorf("tmux new-window failed: %w", err), stderr)
	}

	fmt.Fprintf(w, "Launched %s.\n", tabName)
	return nil
}

// findWindowExact scans `tmux list-windows -a` output (format
// `#{window_id}\t#{window_name}`) for a window whose name equals name
// exactly, returning its server-global window ID. The format deliberately
// carries NO leading #{session_name} field: an unused leading field would
// let a tab inside a session name shift the columns and silently miss the
// match. The name is the LAST field and is split with a bounded SplitN, so
// window names containing tabs survive. Pure function, extracted for unit
// tests.
func findWindowExact(out, name string) (windowID string, found bool) {
	for _, line := range strings.Split(strings.TrimRight(out, "\n"), "\n") {
		parts := strings.SplitN(line, "\t", 2)
		if len(parts) != 2 {
			continue
		}
		if parts[1] == name {
			return parts[0], true
		}
	}
	return "", false
}

// resolveDoingProfile parses `fab resolve-agent apply` stdout — the byte-stable
// lines `model=<id>` and optional `effort=<level>` — into an agent.Profile,
// falling back to fab-kit's built-in doing default on empty or unparseable
// output. It is a PURE function: the caller does the live shell-out and passes
// the captured stdout (or "" when the command itself errored — e.g. an installed
// `fab` that predates `resolve-agent`), so the parse+fallback is unit-testable
// without exec. Empty/garbage stdout ⇒ the doing default, so the caller can pass
// "" on any command error.
func resolveDoingProfile(stdout string) agent.Profile {
	// defaultTiers always carries TierDoing (guarded by the agent drift-guard
	// test), so the ok return is unconditionally true here.
	def, _ := agent.DefaultTier(agent.TierDoing)

	var p agent.Profile
	for _, line := range strings.Split(stdout, "\n") {
		switch {
		case strings.HasPrefix(line, "model="):
			p.Model = strings.TrimPrefix(line, "model=")
		case strings.HasPrefix(line, "effort="):
			p.Effort = strings.TrimPrefix(line, "effort=")
		}
	}

	// A valid resolve-agent run always emits a model= line, so a parsed model is
	// the signal that the output was real; anything else (empty stdout, garbage,
	// command error routed in as "") falls back to the built-in doing default.
	if p.Model == "" {
		return def
	}
	return p
}

// gitRepoRoot returns the git repo root for the current directory. On
// failure the error carries git's stderr detail (e.g. "fatal: not a git
// repository ...") rather than a bare exit status.
func gitRepoRoot() (string, error) {
	out, stderr, err := pane.RunCmd("git", "rev-parse", "--show-toplevel")
	if err != nil {
		return "", pane.StderrError(err, stderr)
	}
	return strings.TrimSpace(out), nil
}

// stateDir returns the XDG state base dir, spec-compliant and uniform on Linux
// and macOS. It honors XDG_STATE_HOME only when set AND absolute; otherwise it
// falls back to $HOME/.local/state. Deliberately NOT ~/Library/... on macOS —
// terminal users expect ~/.local/state, and the Go stdlib has no UserStateDir().
func stateDir() (string, error) {
	if s := os.Getenv("XDG_STATE_HOME"); s != "" && filepath.IsAbs(s) {
		return s, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".local", "state"), nil
}

// slugify converts a tmux socket path into a filesystem-safe, deterministic,
// collision-free slug. The rule: escape literal "-" by doubling it ("-" → "--")
// FIRST, then strip the leading separator and replace remaining path separators
// with a single "-". Escaping before substitution makes the mapping injective —
// a lone "-" in the output is always a former separator, a "--" is always a
// literal "-" from the source — so distinct socket paths produce distinct slugs
// (e.g. "/tmp/tmux-1000/default" → "tmp-tmux--1000-default" and
// "/tmp/tmux/1000/default" → "tmp-tmux-1000-default" no longer collide).
func slugify(s string) string {
	s = strings.ReplaceAll(s, "-", "--")
	s = strings.TrimPrefix(s, string(os.PathSeparator))
	s = strings.TrimPrefix(s, "/")
	s = strings.ReplaceAll(s, string(os.PathSeparator), "-")
	s = strings.ReplaceAll(s, "/", "-")
	if s == "" {
		return "default"
	}
	return s
}

// serverSlug derives a filesystem-safe slug from the tmux socket path. It falls
// back to "default" when tmux cannot be queried (the operator must still
// function if the #{socket_path} query fails).
func serverSlug(server string) string {
	out, err := exec.Command("tmux", pane.WithServer(server, "display-message", "-p", "#{socket_path}")...).Output()
	if err != nil {
		return "default"
	}
	return slugify(strings.TrimSpace(string(out)))
}

// StatePath returns the server-keyed operator state file path:
// <stateDir>/fab/operator/<server-slug>.yaml. The parent directory is created
// with MkdirAll (0o755). The path is keyed by the tmux socket (via serverSlug)
// so cross-repo coordination state has a stable, server-scoped home rather than
// living at a repo-rooted .fab-operator.yaml.
func StatePath(server string) (string, error) {
	base, err := stateDir()
	if err != nil {
		return "", err
	}
	dir := filepath.Join(base, "fab", "operator")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return filepath.Join(dir, serverSlug(server)+".yaml"), nil
}
