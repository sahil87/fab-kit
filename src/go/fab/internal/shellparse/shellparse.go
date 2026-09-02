// Package shellparse provides the small shell-token parsing subset needed to
// identify leading executables in fab's provider command templates.
package shellparse

import "strings"

type word struct {
	text string
	raw  string
}

// Words splits a command template into words, honoring single- and
// double-quoted spans (no escape processing — the provider grammars ship
// none). Quote characters are stripped. This is deliberately not a full shell
// lexer: it exists to find a command's leading executable, and nested
// `sh -c '<inner>'` forms keep their inner command as one quoted word.
func Words(s string) []string {
	parsed := parseWords(s)
	words := make([]string, 0, len(parsed))
	for _, word := range parsed {
		words = append(words, word.text)
	}
	return words
}

func parseWords(s string) []word {
	var words []word
	var cur strings.Builder
	var raw strings.Builder
	var quote byte // 0 = outside quotes
	flush := func() {
		if cur.Len() > 0 {
			words = append(words, word{text: cur.String(), raw: raw.String()})
			cur.Reset()
		}
		raw.Reset()
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		switch {
		case quote != 0:
			raw.WriteByte(c)
			if c == quote {
				quote = 0
			} else {
				cur.WriteByte(c)
			}
		case c == '\'' || c == '"':
			raw.WriteByte(c)
			quote = c
		case c == ' ' || c == '\t':
			flush()
		default:
			raw.WriteByte(c)
			cur.WriteByte(c)
		}
	}
	flush()
	return words
}

// LeadingCommand returns the first word that is not a POSIX NAME=value
// assignment prefix. Assignment names use the portable shell identifier
// grammar: [A-Za-z_][A-Za-z0-9_]*. An equals sign elsewhere in a word is data,
// so executable paths such as /opt/agent=canary are returned unchanged.
func LeadingCommand(command string) string {
	for _, word := range parseWords(command) {
		if !isAssignmentPrefix(word.raw) {
			return word.text
		}
	}
	return ""
}

func isAssignmentPrefix(raw string) bool {
	equals := strings.IndexByte(raw, '=')
	if equals <= 0 || !isNameStart(raw[0]) {
		return false
	}
	for i := 1; i < equals; i++ {
		if !isNameChar(raw[i]) {
			return false
		}
	}
	return true
}

func isNameStart(c byte) bool {
	return c == '_' || c >= 'A' && c <= 'Z' || c >= 'a' && c <= 'z'
}

func isNameChar(c byte) bool {
	return isNameStart(c) || c >= '0' && c <= '9'
}
