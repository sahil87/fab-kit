package agentmodels

import (
	"bufio"
	"bytes"
	"fmt"
	"strconv"
	"strings"
)

type tomlDocument struct {
	top    map[string]string
	tables []tomlTable
}

type tomlTable struct {
	header  string
	array   bool
	strings map[string]string
	arrays  map[string][]string
}

func parseTOML(data []byte) (tomlDocument, error) {
	doc := tomlDocument{top: make(map[string]string)}
	current := -1
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := strings.TrimSpace(stripTOMLComment(scanner.Text()))
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "[") {
			header, array, err := parseTOMLHeader(line)
			if err != nil {
				return doc, fmt.Errorf("line %d: malformed table header", lineNumber)
			}
			doc.tables = append(doc.tables, tomlTable{
				header:  header,
				array:   array,
				strings: make(map[string]string),
				arrays:  make(map[string][]string),
			})
			current = len(doc.tables) - 1
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key, value = strings.TrimSpace(key), strings.TrimSpace(value)
		if current < 0 {
			parsed, isString, err := parseTOMLString(value)
			if err != nil || (!isString && (key == "model" || key == "default_model")) {
				return doc, fmt.Errorf("line %d: malformed string for %s", lineNumber, key)
			}
			if isString {
				doc.top[key] = parsed
			}
			continue
		}
		table := &doc.tables[current]
		parsed, isString, err := parseTOMLString(value)
		if err != nil || (!isString && key == "display_name") {
			return doc, fmt.Errorf("line %d: malformed string for %s", lineNumber, key)
		}
		if isString {
			table.strings[key] = parsed
			continue
		}
		if strings.HasPrefix(value, "[") {
			parsed, err := parseTOMLStringArray(value)
			if err != nil {
				return doc, fmt.Errorf("line %d: malformed array for %s", lineNumber, key)
			}
			table.arrays[key] = parsed
		} else if key == "support_efforts" {
			return doc, fmt.Errorf("line %d: malformed array for %s", lineNumber, key)
		}
	}
	if err := scanner.Err(); err != nil {
		return doc, fmt.Errorf("read TOML: %w", err)
	}
	return doc, nil
}

func parseTOMLHeader(line string) (string, bool, error) {
	if strings.HasPrefix(line, "[[") {
		if !strings.HasSuffix(line, "]]") || len(line) < 4 {
			return "", false, fmt.Errorf("malformed array header")
		}
		header := strings.TrimSpace(line[2 : len(line)-2])
		if header == "" {
			return "", false, fmt.Errorf("empty array header")
		}
		return header, true, nil
	}
	if !strings.HasSuffix(line, "]") || len(line) < 2 {
		return "", false, fmt.Errorf("malformed header")
	}
	header := strings.TrimSpace(line[1 : len(line)-1])
	if header == "" {
		return "", false, fmt.Errorf("empty header")
	}
	return header, false, nil
}

func stripTOMLComment(line string) string {
	var quote rune
	escaped := false
	for index, r := range line {
		if escaped {
			escaped = false
			continue
		}
		if quote == '"' && r == '\\' {
			escaped = true
			continue
		}
		if quote != 0 {
			if r == quote {
				quote = 0
			}
			continue
		}
		if r == '\'' || r == '"' {
			quote = r
			continue
		}
		if r == '#' {
			return line[:index]
		}
	}
	return line
}

func parseTOMLString(raw string) (string, bool, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || (raw[0] != '"' && raw[0] != '\'') {
		return "", false, nil
	}
	value, next, err := scanTOMLString(raw, 0)
	if err != nil || strings.TrimSpace(raw[next:]) != "" {
		return "", false, fmt.Errorf("malformed string")
	}
	return value, true, nil
}

func scanTOMLString(raw string, start int) (string, int, error) {
	if start >= len(raw) || (raw[start] != '"' && raw[start] != '\'') {
		return "", start, fmt.Errorf("string must be quoted")
	}
	quote := raw[start]
	escaped := false
	for index := start + 1; index < len(raw); index++ {
		if escaped {
			escaped = false
			continue
		}
		if quote == '"' && raw[index] == '\\' {
			escaped = true
			continue
		}
		if raw[index] != quote {
			continue
		}
		literal := raw[start : index+1]
		if quote == '\'' {
			return literal[1 : len(literal)-1], index + 1, nil
		}
		value, err := strconv.Unquote(literal)
		if err != nil {
			return "", index + 1, err
		}
		return value, index + 1, nil
	}
	return "", len(raw), fmt.Errorf("unterminated string")
}

func parseTOMLStringArray(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if len(raw) < 2 || raw[0] != '[' || raw[len(raw)-1] != ']' {
		return nil, fmt.Errorf("array must be single-line and bracketed")
	}
	inner := strings.TrimSpace(raw[1 : len(raw)-1])
	values := []string{}
	for index := 0; index < len(inner); {
		for index < len(inner) && (inner[index] == ' ' || inner[index] == '\t' || inner[index] == ',') {
			index++
		}
		if index == len(inner) {
			break
		}
		value, next, err := scanTOMLString(inner, index)
		if err != nil {
			return nil, err
		}
		values = append(values, value)
		index = next
		for index < len(inner) && (inner[index] == ' ' || inner[index] == '\t') {
			index++
		}
		if index < len(inner) && inner[index] != ',' {
			return nil, fmt.Errorf("expected comma")
		}
	}
	return values, nil
}

func kimiModelName(header string) (string, bool) {
	if !strings.HasPrefix(header, "models.") {
		return "", false
	}
	raw := strings.TrimSpace(strings.TrimPrefix(header, "models."))
	if raw == "" {
		return "", false
	}
	if raw[0] == '"' || raw[0] == '\'' {
		value, ok, err := parseTOMLString(raw)
		return value, ok && err == nil
	}
	return raw, true
}
