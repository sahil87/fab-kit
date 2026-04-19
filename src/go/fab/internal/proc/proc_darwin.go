//go:build darwin

package proc

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// ClaudePID returns the grandparent PID of the calling process by executing
// `ps -o ppid= -p $PPID`. Given Claude's `claude → sh -c → hook` invocation
// pattern, the caller's parent is the shell and the grandparent is Claude
// itself.
//
// Returns a wrapped error when `ps` fails or the output cannot be parsed.
func ClaudePID() (int, error) {
	ppid := os.Getppid()

	out, err := exec.Command("ps", "-o", "ppid=", "-p", strconv.Itoa(ppid)).Output()
	if err != nil {
		return 0, fmt.Errorf("ps -o ppid= -p %d: %w", ppid, err)
	}

	trimmed := strings.TrimSpace(string(out))
	if trimmed == "" {
		return 0, fmt.Errorf("ps -o ppid= -p %d: empty output", ppid)
	}

	gp, err := strconv.Atoi(trimmed)
	if err != nil {
		return 0, fmt.Errorf("parsing ps output %q: %w", trimmed, err)
	}
	return gp, nil
}
