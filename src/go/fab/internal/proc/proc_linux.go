//go:build linux

package proc

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// ClaudePID returns the grandparent PID of the calling process by reading
// /proc/$PPID/status and parsing the "PPid:" line. Given Claude's
// `claude → sh -c → hook` invocation pattern, the caller's parent is the
// shell and the grandparent is Claude itself.
//
// Returns a wrapped error when the PPID file cannot be read or the PPid
// field cannot be parsed.
func ClaudePID() (int, error) {
	ppid := os.Getppid()
	path := fmt.Sprintf("/proc/%d/status", ppid)

	f, err := os.Open(path)
	if err != nil {
		return 0, fmt.Errorf("reading %s: %w", path, err)
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "PPid:") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			return 0, fmt.Errorf("parsing %s: malformed PPid line %q", path, line)
		}
		gp, err := strconv.Atoi(fields[1])
		if err != nil {
			return 0, fmt.Errorf("parsing %s: invalid PPid %q: %w", path, fields[1], err)
		}
		return gp, nil
	}
	if err := scanner.Err(); err != nil {
		return 0, fmt.Errorf("reading %s: %w", path, err)
	}
	return 0, fmt.Errorf("parsing %s: PPid line not found", path)
}
