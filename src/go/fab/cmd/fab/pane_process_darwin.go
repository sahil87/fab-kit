//go:build darwin

package main

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// psEntry holds a parsed ps output entry.
type psEntry struct {
	pid  int
	ppid int
	comm string
}

// discoverProcessTree discovers the process tree for a given PID on macOS
// by using ps to enumerate processes and filtering by PPID traversal.
func discoverProcessTree(pid int) ([]ProcessNode, error) {
	out, err := exec.Command("ps", "-o", "pid,ppid,comm", "-ax").Output()
	if err != nil {
		return nil, fmt.Errorf("ps: %w", err)
	}

	var entries []psEntry
	lines := strings.Split(strings.TrimSpace(string(out)), "\n")
	for _, line := range lines[1:] { // skip header
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}
		p, err := strconv.Atoi(fields[0])
		if err != nil {
			continue
		}
		pp, err := strconv.Atoi(fields[1])
		if err != nil {
			continue
		}
		// comm may have path -- take the last component
		comm := fields[2]
		if idx := strings.LastIndex(comm, "/"); idx >= 0 {
			comm = comm[idx+1:]
		}
		entries = append(entries, psEntry{pid: p, ppid: pp, comm: comm})
	}

	// Build children map
	childrenMap := make(map[int][]psEntry)
	entryMap := make(map[int]psEntry)
	for _, e := range entries {
		childrenMap[e.ppid] = append(childrenMap[e.ppid], e)
		entryMap[e.pid] = e
	}

	rootEntry, ok := entryMap[pid]
	if !ok {
		return nil, fmt.Errorf("PID %d not found in ps output", pid)
	}

	node := buildNodeFromPS(rootEntry, childrenMap)
	return []ProcessNode{node}, nil
}

// buildNodeFromPS recursively builds a ProcessNode from ps data.
func buildNodeFromPS(entry psEntry, childrenMap map[int][]psEntry) ProcessNode {
	cmdline := getPSCmdline(entry.pid)

	node := ProcessNode{
		PID:            entry.pid,
		PPID:           entry.ppid,
		Comm:           entry.comm,
		Cmdline:        cmdline,
		Classification: ClassifyProcess(entry.comm),
		Children:       []ProcessNode{},
	}

	for _, child := range childrenMap[entry.pid] {
		node.Children = append(node.Children, buildNodeFromPS(child, childrenMap))
	}

	return node
}

// getPSCmdline gets the full command line for a PID via ps.
func getPSCmdline(pid int) string {
	out, err := exec.Command("ps", "-o", "args=", "-p", strconv.Itoa(pid)).Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}
