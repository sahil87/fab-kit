// Package shellquote provides shell-token quoting shared by command composers.
package shellquote

import "strings"

// Single wraps value in POSIX shell single quotes and escapes embedded single
// quotes with the standard close/escape/reopen sequence.
func Single(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}
