package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
)

var version = "dev"

// fabKitArgs is the static allowlist of arguments routed to fab-kit.
var fabKitArgs = map[string]bool{
	"init":    true,
	"upgrade": true,
	"sync":    true,
	"update":  true,
	"doctor":  true,
}

func main() {
	if len(os.Args) < 2 {
		printHelp()
		return
	}

	arg := os.Args[1]

	switch {
	case arg == "--version" || arg == "-v":
		fmt.Printf("fab %s\n", version)
	case arg == "--help" || arg == "-h" || arg == "help":
		printHelp()
	case fabKitArgs[arg]:
		execFabKit(os.Args[1:])
	default:
		execFabGo(os.Args[1:])
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

// execFabGo resolves the fab version from config.yaml, ensures the binary is
// cached, and replaces the current process with fab-go.
func execFabGo(args []string) {
	cfg, err := internal.ResolveConfig()
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
	if cfg == nil {
		fmt.Fprintln(os.Stderr, "Not in a fab-managed repo. Run 'fab init' to set one up.")
		os.Exit(1)
	}

	bin, err := internal.EnsureCached(cfg.FabVersion)
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

// printHelp composes help output from both sub-binaries.
func printHelp() {
	fmt.Printf("fab %s — workspace & workflow toolkit\n\n", version)
	fmt.Println("Workspace commands:")
	fmt.Println("  init       Initialize fab in the current repo")
	fmt.Println("  upgrade    Upgrade to a specific or latest version")
	fmt.Println("  sync       Sync workspace (skills, directories, scaffold)")
	fmt.Println("  update     Update fab-kit itself via Homebrew")
	fmt.Println()

	// Show workflow commands only inside a fab-managed repo
	cfg, _ := internal.ResolveConfig()
	if cfg != nil {
		bin, err := internal.EnsureCached(cfg.FabVersion)
		if err == nil {
			out, err := exec.Command(bin, "--help").Output()
			if err == nil {
				fmt.Println("Workflow commands (fab-go):")
				fmt.Print(string(out))
				fmt.Println()
			}
		}
	}

	fmt.Println("Flags:")
	fmt.Println("  --version, -v   Show version")
	fmt.Println("  --help, -h      Show this help")
}
