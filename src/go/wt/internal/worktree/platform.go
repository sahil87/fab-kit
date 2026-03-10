package worktree

import (
	"os"
	"runtime"
)

// DetectOS returns the operating system as "macos", "linux", or "unknown".
func DetectOS() string {
	switch runtime.GOOS {
	case "darwin":
		return "macos"
	case "linux":
		return "linux"
	default:
		return "unknown"
	}
}

// IsByobuSession returns true if running inside a byobu session.
func IsByobuSession() bool {
	if os.Getenv("BYOBU_TTY") != "" || os.Getenv("BYOBU_BACKEND") != "" || os.Getenv("BYOBU_SESSION") != "" {
		return true
	}
	if os.Getenv("TMUX") != "" && os.Getenv("BYOBU_CONFIG_DIR") != "" {
		return true
	}
	return false
}

// IsTmuxSession returns true if running inside a plain tmux session (not byobu).
func IsTmuxSession() bool {
	return os.Getenv("TMUX") != "" && !IsByobuSession()
}
