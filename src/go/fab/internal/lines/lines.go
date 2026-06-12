// Package lines provides whole-file line reading for the small markdown/YAML
// files fab parses. It replaces the unchecked bufio.Scanner idiom: os.ReadFile
// is all-or-nothing, so a partial line list is impossible — the caller either
// gets every line or an error, and bufio's 64KB MaxScanTokenSize line limit
// does not apply.
package lines

import (
	"os"
	"strings"
)

// ReadFileLines reads path fully and returns its lines. Each line is
// TrimSuffix'd of "\r" to preserve bufio.ScanLines' CRLF behavior. Read
// failures (including a missing file) are returned as errors — never as a
// silent empty slice. Note that unlike a scanner, a file ending in "\n"
// yields a trailing empty line (strings.Split semantics).
func ReadFileLines(path string) ([]string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return Split(string(data)), nil
}

// Split returns the lines of in-memory content with the same CRLF semantics
// as ReadFileLines: split on "\n", trailing "\r" trimmed per line.
func Split(content string) []string {
	split := strings.Split(content, "\n")
	for i, line := range split {
		split[i] = strings.TrimSuffix(line, "\r")
	}
	return split
}
