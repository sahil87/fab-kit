package main

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

var version = "dev"

// fabKitArgs is the static allowlist of arguments routed to fab-kit, derived
// from the shared internal.LifecycleCommands table (the single source of
// truth for the workspace command set).
var fabKitArgs = internal.LifecycleCommandSet()

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	arg := os.Args[1]

	switch {
	case arg == "--version" || arg == "-v":
		cfg, _ := internal.ResolveConfig()
		printVersion(os.Stdout, version, cfg)
	case arg == "--help" || arg == "-h" || arg == "help":
		printHelp()
	case fabKitArgs[arg]:
		execFabKit(os.Args[1:])
	default:
		execFabGo(os.Args[1:])
	}
}

// printVersion writes the system version and, when cfg is non-nil, the project-pinned version.
func printVersion(w io.Writer, sysVersion string, cfg *internal.ConfigResult) {
	fmt.Fprintf(w, "fab %s\n", sysVersion)
	if cfg != nil {
		fmt.Fprintf(w, "project: %s\n", cfg.FabVersion)
	}
}

// execFabKit dispatches to the fab-kit binary via syscall.Exec.
func execFabKit(args []string) {
	bin, err := exec.LookPath("fab-kit")
	if err != nil {
		// Fall back to fab-kit next to the current binary
		self, _ := os.Executable()
		bin = filepath.Join(filepath.Dir(self), "fab-kit")
	}
	argv := append([]string{bin}, args...)
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot exec fab-kit: %s\n", err)
		os.Exit(1)
	}
}

// execFabGo resolves the fab version (project-pinned when config.yaml is
// present, router-bundled otherwise), ensures the binary is cached, and
// replaces the current process with fab-go. Corrupted config (parse error)
// is the only path that hard-errors at the router; missing config falls
// through to the bundled version so config-free commands (completion, help,
// kit-path, pane, etc.) work outside a fab repo. Per-command guards in
// fab-go remain authoritative for "needs project state" errors.
func execFabGo(args []string) {
	cfg, err := internal.ResolveConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	fabVersion := version
	if cfg != nil {
		fabVersion = cfg.FabVersion
	}

	bin, err := internal.EnsureCached(fabVersion)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}

	argv := append([]string{bin}, args...)
	if err := syscall.Exec(bin, argv, os.Environ()); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: cannot exec fab-go: %s\n", err)
		os.Exit(1)
	}
}

// printHelp composes help output from both sub-binaries. The workspace
// section is rendered in-process from the shared LifecycleCommands table —
// deliberately NOT by exec'ing `fab-kit --help` — so it renders even when the
// fab-kit binary is absent, and its Shorts can never drift from the cobra
// registrations (enforced by test in cmd/fab-kit).
func printHelp() {
	fmt.Printf("fab %s — workspace & workflow toolkit\n\n", version)
	fmt.Println("Workspace commands:")
	nameWidth := 0
	for _, c := range internal.LifecycleCommands {
		if len(c.Name) > nameWidth {
			nameWidth = len(c.Name)
		}
	}
	for _, c := range internal.LifecycleCommands {
		fmt.Printf("  %-*s  %s\n", nameWidth, c.Name, c.Short)
	}
	fmt.Println()

	// Show workflow commands. Inside a fab repo, use the project-pinned version.
	// Outside a fab repo, fall back to the router's bundled version so all
	// workflow commands remain discoverable from scratch tabs. Errors are
	// silently swallowed — the help section is best-effort.
	cfg, _ := internal.ResolveConfig()
	var fabVersion string
	if cfg != nil {
		fabVersion = cfg.FabVersion
	} else {
		fabVersion = version
	}
	if bin, err := internal.EnsureCached(fabVersion); err == nil {
		if out, err := exec.Command(bin, "--help").Output(); err == nil {
			fmt.Println("Workflow commands (fab-go):")
			fmt.Print(string(out))
			fmt.Println()
		}
	}

	fmt.Println("Flags:")
	fmt.Println("  --version, -v   Show version")
	fmt.Println("  --help, -h      Show this help")
}
