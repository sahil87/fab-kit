//go:build windows

package dispatch

import "fmt"

// errPOSIXOnly is the v1 Windows-unsupported error. `fab dispatch` relies on a
// POSIX shell (setsid/timeout) for its supervisor-free detached launch and on
// POSIX process-group signalling for kill; rather than half-work, it errors
// clearly on Windows. This is a compile-time reality — the launch/signal
// syscalls live only in dispatch_posix.go — not a runtime GOOS string check.
var errPOSIXOnly = fmt.Errorf("fab dispatch requires a POSIX shell (setsid/timeout); Windows is not supported in v1")

// WrapperArgv is unused on Windows (Launch errors before it is needed) but is
// defined so the platform-independent core still references a single signature.
func WrapperArgv(cmd, promptPath, logPath, exitPath string, timeoutSecs int) []string {
	return nil
}

// Launch is unsupported on Windows — surfaces the POSIX-only error.
func Launch(argv []string, cwd string) (pid, pgid int, err error) {
	return 0, 0, errPOSIXOnly
}

// Alive conservatively reports false on Windows — the POSIX kill(pid,0) probe
// does not apply, and dispatch never launches here so no live pid ever exists.
func Alive(pid int) bool { return false }

// KillGroup is unsupported on Windows — surfaces the POSIX-only error.
func KillGroup(pgid int) error { return errPOSIXOnly }
