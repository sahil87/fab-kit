//go:build !windows

package dispatch

import (
	"fmt"
	"os/exec"
	"syscall"
)

// Launch starts the wrapper detached in a new session/process group and returns
// the child pid and pgid. cwd is the repository root. SysProcAttr.Setsid makes
// the child (the `sh` in WrapperArgv) a session AND process-group leader (pgid
// == pid), detaching it from the orchestrator's process group so the dispatch
// survives the orchestrator dying — this is the SOLE detach mechanism (see
// WrapperArgv on why the `setsid` binary is deliberately not also prefixed). The
// recorded pid is therefore the live worker shell, which liveness/kill track.
// The child is not waited on — the shell records $? into {stage}.exit itself.
func Launch(argv []string, cwd string) (pid, pgid int, err error) {
	if len(argv) == 0 {
		return 0, 0, fmt.Errorf("empty launch argv")
	}
	c := exec.Command(argv[0], argv[1:]...)
	c.Dir = cwd
	c.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := c.Start(); err != nil {
		return 0, 0, fmt.Errorf("launch dispatch: %w", err)
	}
	pid = c.Process.Pid
	// With Setsid the child is a session/group leader, so its pgid equals its
	// pid. Confirm via the syscall so a future launch-attr change can't silently
	// desync the recorded pgid.
	gpid, gerr := syscall.Getpgid(pid)
	if gerr != nil {
		gpid = pid
	}
	// Release our handle — we never wait on the detached child.
	_ = c.Process.Release()
	return pid, gpid, nil
}

// Alive reports whether pid is a live process. It is the POSIX-standard
// kill(pid, 0) liveness probe: nil means alive, EPERM means
// alive-but-unsignalable, anything else (ESRCH, etc.) is dead.
func Alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	err := syscall.Kill(pid, 0)
	if err == nil {
		return true
	}
	if err == syscall.EPERM {
		return true
	}
	return false
}

// KillGroup sends SIGTERM to the whole process group (negative pgid) so the
// detached command and all its children die together. It is idempotent: ESRCH
// ("no such process" — the group is already gone) is a benign no-op, not an
// error. pgid <= 0 is rejected to avoid signalling pid 0 / the caller's own
// group via a negative argument.
func KillGroup(pgid int) error {
	if pgid <= 0 {
		return fmt.Errorf("invalid pgid %d", pgid)
	}
	err := syscall.Kill(-pgid, syscall.SIGTERM)
	if err == nil || err == syscall.ESRCH {
		return nil
	}
	return fmt.Errorf("kill process group %d: %w", pgid, err)
}
