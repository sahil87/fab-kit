package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
)

// Process state constants
const (
	processStateRunning         = "running"
	processStateWaitingForInput = "waiting-for-input"
	processStateSleeping        = "sleeping"
	processStateStopped         = "stopped"
	processStateExited          = "exited"
)

func paneProcessCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "process <pane>",
		Short: "Detect OS-level state of the foreground process in a tmux pane",
		Args:  cobra.ExactArgs(1),
		RunE:  runPaneProcess,
	}
	cmd.Flags().Bool("json", false, "Output as JSON object")
	return cmd
}

// paneProcessJSON represents JSON output for pane process.
type paneProcessJSON struct {
	Pane        string  `json:"pane"`
	PID         int     `json:"pid"`
	State       string  `json:"state"`
	ProcessName string  `json:"process_name"`
	Change      *string `json:"change"`
}

func runPaneProcess(cmd *cobra.Command, args []string) error {
	paneID := args[0]
	jsonFlag, _ := cmd.Flags().GetBool("json")

	// Validate pane exists
	if err := validatePaneExists(paneID); err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "pane %s not found\n", paneID)
		os.Exit(1)
	}

	// Get pane PID (shell PID)
	shellPID, err := getPanePID(paneID)
	if err != nil {
		return fmt.Errorf("getting pane PID: %w", err)
	}

	// Find foreground process and its state
	fgPID, fgName, state := detectForegroundProcess(shellPID)

	if !jsonFlag {
		fmt.Fprintln(cmd.OutOrStdout(), state)
		return nil
	}

	// JSON mode: resolve change context
	var change *string
	paneCWD := getPaneCWD(paneID)
	if paneCWD != "" {
		wtRoot, err := gitWorktreeRoot(paneCWD)
		if err == nil {
			fabDir := filepath.Join(wtRoot, "fab")
			if _, statErr := os.Stat(fabDir); statErr == nil {
				_, folderName := readFabCurrent(wtRoot)
				if folderName != "" {
					change = &folderName
				}
			}
		}
	}

	result := paneProcessJSON{
		Pane:        paneID,
		PID:         fgPID,
		State:       state,
		ProcessName: fgName,
		Change:      change,
	}

	enc := json.NewEncoder(cmd.OutOrStdout())
	enc.SetIndent("", "  ")
	return enc.Encode(result)
}

// getPanePID returns the PID of the pane's shell process.
func getPanePID(paneID string) (int, error) {
	out, err := exec.Command("tmux", "display-message", "-t", paneID, "-p", "#{pane_pid}").Output()
	if err != nil {
		return 0, err
	}
	return strconv.Atoi(strings.TrimSpace(string(out)))
}

// detectForegroundProcess finds the foreground process in a pane's process tree
// and returns its PID, name, and state.
func detectForegroundProcess(shellPID int) (pid int, name string, state string) {
	// Walk process tree to find the foreground (leaf) process
	fgPID, fgName := findForegroundProcess(shellPID)

	if fgPID == shellPID {
		// Only the shell is running — no foreground child
		return fgPID, fgName, processStateExited
	}

	// Detect the process state
	state = detectProcessState(fgPID)
	return fgPID, fgName, state
}

// findForegroundProcess walks the process tree from shellPID to find the
// foreground (leaf) process. Returns the leaf PID and process name.
func findForegroundProcess(shellPID int) (int, string) {
	current := shellPID
	currentName := getProcessName(current)

	for {
		childPID, childName := getChildProcess(current)
		if childPID == 0 {
			return current, currentName
		}
		current = childPID
		currentName = childName
	}
}

// getChildProcess returns a single child PID and name for a given parent PID.
// Returns (0, "") if no children.
func getChildProcess(parentPID int) (int, string) {
	if runtime.GOOS == "linux" {
		return getChildProcessLinux(parentPID)
	}
	return getChildProcessDarwin(parentPID)
}

// getChildProcessLinux reads /proc to find children.
func getChildProcessLinux(parentPID int) (int, string) {
	childrenPath := fmt.Sprintf("/proc/%d/task/%d/children", parentPID, parentPID)
	data, err := os.ReadFile(childrenPath)
	if err != nil {
		return 0, ""
	}

	fields := strings.Fields(string(data))
	if len(fields) == 0 {
		return 0, ""
	}

	// Take the first child
	pid, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0, ""
	}

	return pid, getProcessName(pid)
}

// getChildProcessDarwin uses ps -axo to find children by matching ppid.
// macOS ps does not support GNU --ppid; we list all processes and filter.
func getChildProcessDarwin(parentPID int) (int, string) {
	out, err := exec.Command("ps", "-axo", "pid=,ppid=,comm=").Output()
	if err != nil {
		return 0, ""
	}

	parentStr := strconv.Itoa(parentPID)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		if fields[1] != parentStr {
			continue
		}
		pid, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		return pid, strings.Join(fields[2:], " ")
	}

	return 0, ""
}

// getProcessName returns the name of a process by PID.
func getProcessName(pid int) string {
	if runtime.GOOS == "linux" {
		return getProcessNameLinux(pid)
	}
	return getProcessNameDarwin(pid)
}

// getProcessNameLinux reads /proc/{pid}/comm.
func getProcessNameLinux(pid int) string {
	commPath := fmt.Sprintf("/proc/%d/comm", pid)
	data, err := os.ReadFile(commPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// getProcessNameDarwin uses ps to get the process name.
func getProcessNameDarwin(pid int) string {
	out, err := exec.Command("ps", "-o", "comm=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

// detectProcessState determines the process state for a given PID.
func detectProcessState(pid int) string {
	if runtime.GOOS == "linux" {
		return detectProcessStateLinux(pid)
	}
	return detectProcessStateDarwin(pid)
}

// detectProcessStateLinux uses /proc/{pid}/stat and /proc/{pid}/wchan.
func detectProcessStateLinux(pid int) string {
	statPath := fmt.Sprintf("/proc/%d/stat", pid)
	data, err := os.ReadFile(statPath)
	if err != nil {
		return processStateSleeping // graceful fallback
	}

	procState := parseProcStatState(string(data))

	switch procState {
	case "R":
		return processStateRunning
	case "T", "t":
		return processStateStopped
	case "Z", "X":
		return processStateExited
	case "S", "D":
		// Sleeping — check wchan to distinguish tty read from other sleep
		wchan := readWchan(pid)
		if isWaitingForInput(wchan) {
			return processStateWaitingForInput
		}
		return processStateSleeping
	default:
		return processStateSleeping // graceful fallback for unknown state
	}
}

// parseProcStatState extracts the state character from /proc/{pid}/stat content.
// Format: "pid (comm) S ..." — the state is the first char after the closing paren.
func parseProcStatState(stat string) string {
	// Find the closing parenthesis (comm can contain spaces and parens)
	idx := strings.LastIndex(stat, ")")
	if idx == -1 || idx+2 >= len(stat) {
		return ""
	}
	// State character is at idx+2 (space then state)
	return string(stat[idx+2])
}

// readWchan reads /proc/{pid}/wchan to determine what the process is waiting on.
func readWchan(pid int) string {
	wchanPath := fmt.Sprintf("/proc/%d/wchan", pid)
	data, err := os.ReadFile(wchanPath)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}

// isWaitingForInput returns true if the wchan value indicates the process
// is blocked waiting for terminal input. Only matches definitively tty-related
// wait channels to avoid false positives from network I/O, sleep, or poll.
func isWaitingForInput(wchan string) bool {
	// Definitively tty-related wait channels only.
	// do_select, ep_poll, poll_schedule_timeout, pipe_read, unix_stream_read_generic
	// are intentionally excluded — they match network servers, build tools, sleep, etc.
	// Ambiguous states fall back to "sleeping" per spec's graceful degradation requirement.
	waitingWchans := []string{
		"n_tty_read",
		"wait_woken",
	}

	wchanLower := strings.ToLower(wchan)
	for _, w := range waitingWchans {
		if wchanLower == w {
			return true
		}
	}

	// Heuristic: contains "tty" (catches n_tty_read variants across kernel versions)
	if strings.Contains(wchanLower, "tty") {
		return true
	}

	return false
}

// detectProcessStateDarwin uses ps and lsof for macOS.
func detectProcessStateDarwin(pid int) string {
	out, err := exec.Command("ps", "-o", "stat=", "-p", fmt.Sprintf("%d", pid)).Output()
	if err != nil {
		return processStateSleeping // graceful fallback
	}

	stat := strings.TrimSpace(string(out))
	if stat == "" {
		return processStateSleeping
	}

	firstChar := string(stat[0])

	switch firstChar {
	case "R":
		return processStateRunning
	case "T":
		return processStateStopped
	case "Z":
		return processStateExited
	case "S", "U":
		// Sleeping — check if waiting for tty input via lsof
		if isDarwinWaitingForInput(pid) {
			return processStateWaitingForInput
		}
		return processStateSleeping
	default:
		return processStateSleeping
	}
}

// isDarwinWaitingForInput uses lsof to check if the process has a tty open for reading.
func isDarwinWaitingForInput(pid int) bool {
	out, err := exec.Command("lsof", "-p", fmt.Sprintf("%d", pid), "-a", "-d", "0").Output()
	if err != nil {
		return false
	}

	output := string(out)
	// If fd 0 (stdin) is a tty device, the process is likely waiting for input
	return strings.Contains(output, "CHR") && (strings.Contains(output, "/dev/tty") || strings.Contains(output, "/dev/pts"))
}
