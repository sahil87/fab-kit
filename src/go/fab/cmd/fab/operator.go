package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
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

	// Resolve the operator-tier {provider, model, effort} so the operator launches
	// its coordinating agent on a deliberately-chosen profile. This names the seam
	// properly — the operator has its OWN tier (previously it borrowed the doing
	// tier). The operator is a per-tmux-server cross-repo coordinator and may be
	// launched from a neutral directory with no fab/ project (e.g. ~/code), so a
	// missing project is NOT an error: config.Load returns an empty config, and
	// agent.ResolveTier/ResolveProvider then degrade to fab-kit's built-in
	// operator tier + built-in claude provider — a no-fab/ launch is fully
	// defaulted.
	spawnCmd := operatorSpawnCommand()

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

// operatorSpawnCommand resolves the operator tier's session command in-process:
// the operator tier → its provider → that provider's session_command, with
// {model}/{effort} substituted via internal/spawn. A missing/unreadable fab
// project degrades to fab-kit's built-in operator tier + built-in claude provider
// (config.Load returns an empty config; agent.ResolveTier/ResolveProvider both
// fall back to the built-ins), so a neutral-directory launch is fully defaulted.
// A provider without a session_command falls back to spawn.DefaultSpawnCommand
// (still profile-substituted) rather than erroring — the operator must always
// launch.
func operatorSpawnCommand() string {
	var cfg *config.Config
	if fabRoot, err := resolve.FabRoot(); err == nil {
		cfg, _ = config.Load(fabRoot) // nil on error → nil-safe accessors below
	}

	profile := operatorProfile(cfg)

	sessionCmd := spawn.DefaultSpawnCommand
	if prov, ok := agent.ResolveProvider(cfg, profile.Provider); ok && prov.SessionCommand != "" {
		sessionCmd = prov.SessionCommand
	}
	return spawn.WithProfile(sessionCmd, profile.Model, profile.Effort)
}

// operatorProfile resolves the operator-tier profile from cfg, degrading to the
// built-in operator default when cfg is nil or the tier cannot be resolved. Pure
// (no exec / no filesystem), so the fallback is unit-testable.
func operatorProfile(cfg *config.Config) agent.Profile {
	if p, err := agent.ResolveTier(cfg, agent.TierOperator); err == nil {
		return p
	}
	// defaultTiers always carries TierOperator (guarded by the drift-guard test).
	def, _ := agent.DefaultTier(agent.TierOperator)
	return def
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
