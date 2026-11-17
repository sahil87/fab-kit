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

type tomlParser struct {
	doc          tomlDocument
	current      int
	continuation *tomlContinuation
}

func parseTOML(data []byte) (tomlDocument, error) {
	parser := tomlParser{
		doc:     tomlDocument{top: make(map[string]string)},
		current: -1,
	}
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		if err := parser.consumeLine(scanner.Text(), lineNumber); err != nil {
			return parser.doc, err
		}
	}
	if err := scanner.Err(); err != nil {
		return parser.doc, fmt.Errorf("read TOML: %w", err)
	}
	// A skipped value still open at EOF has swallowed every later line, so the
	// document is incomplete — surface it rather than return a silently short read.
	if parser.continuation != nil {
		return parser.doc, fmt.Errorf("unterminated value at end of file")
	}
	return parser.doc, nil
}

func (p *tomlParser) consumeLine(raw string, lineNumber int) error {
	if p.continuation != nil {
		if p.continuation.consume(raw) {
			p.continuation = nil
		}
		return nil
	}
	line := strings.TrimSpace(stripTOMLComment(raw))
	if line == "" {
		return nil
	}
	if strings.HasPrefix(line, "[") {
		return p.startTable(line, lineNumber)
	}
	key, value, ok := cutTOMLKeyValue(line)
	if !ok {
		return nil
	}
	key, value = strings.TrimSpace(key), strings.TrimSpace(value)
	if p.current < 0 {
		return p.consumeTopLevel(key, value, lineNumber)
	}
	return p.consumeTableValue(key, value, lineNumber)
}

// cutTOMLKeyValue splits a `key = value` line at the first '=' outside a quoted
// key, so a quoted key containing '=' (`"a=b" = 1`) is not cut inside the quotes.
func cutTOMLKeyValue(line string) (key, value string, ok bool) {
	var quote byte
	for index := 0; index < len(line); index++ {
		switch {
		case quote == '"' && line[index] == '\\':
			index++
		case quote != 0:
			if line[index] == quote {
				quote = 0
			}
		case line[index] == '"' || line[index] == '\'':
			quote = line[index]
		case line[index] == '=':
			return line[:index], line[index+1:], true
		}
	}
	return "", "", false
}

func (p *tomlParser) startTable(line string, lineNumber int) error {
	header, array, err := parseTOMLHeader(line)
	if err != nil {
		return fmt.Errorf("line %d: malformed table header", lineNumber)
	}
	p.doc.tables = append(p.doc.tables, tomlTable{
		header:  header,
		array:   array,
		strings: make(map[string]string),
		arrays:  make(map[string][]string),
	})
	p.current = len(p.doc.tables) - 1
	return nil
}

func (p *tomlParser) consumeTopLevel(key, value string, lineNumber int) error {
	if key != "model" && key != "default_model" {
		p.skipUnsupported(value)
		return nil
	}
	parsed, ok, err := parseTOMLString(value)
	if err != nil || !ok {
		return fmt.Errorf("line %d: malformed string for %s", lineNumber, key)
	}
	p.doc.top[key] = parsed
	return nil
}

func (p *tomlParser) consumeTableValue(key, value string, lineNumber int) error {
	table := &p.doc.tables[p.current]
	isModel := !table.array && strings.HasPrefix(table.header, "models.")
	if !isModel || (key != "display_name" && key != "support_efforts") {
		p.skipUnsupported(value)
		return nil
	}
	if key == "display_name" {
		parsed, ok, err := parseTOMLString(value)
		if err != nil || !ok {
			return fmt.Errorf("line %d: malformed string for %s", lineNumber, key)
		}
		table.strings[key] = parsed
		return nil
	}
	parsed, err := parseTOMLStringArray(value)
	if err != nil {
		return fmt.Errorf("line %d: malformed array for %s", lineNumber, key)
	}
	table.arrays[key] = parsed
	return nil
}

// skipUnsupported records enough lexical state to ignore arrays and quoted
// strings spanning lines. The adapters deliberately support only the small TOML
// subset they read; every other value is skipped without interpreting its content.
func (p *tomlParser) skipUnsupported(value string) {
	var continuation tomlContinuation
	if !continuation.consume(value) {
		p.continuation = &continuation
	}
}

type tomlContinuation struct {
	brackets int
	quote    byte
	triple   string
}

func (c *tomlContinuation) consume(line string) bool {
	for index := 0; index < len(line); index++ {
		if c.triple != "" {
			if strings.HasPrefix(line[index:], c.triple) {
				c.triple = ""
				index += 2
			}
			continue
		}
		if c.quote != 0 {
			if c.quote == '"' && line[index] == '\\' {
				index++
				continue
			}
			if line[index] == c.quote {
				c.quote = 0
			}
			continue
		}
		if line[index] == '#' {
			return c.brackets == 0
		}
		if strings.HasPrefix(line[index:], `"""`) || strings.HasPrefix(line[index:], "'''") {
			c.triple = line[index : index+3]
			index += 2
			continue
		}
		if line[index] == '"' || line[index] == '\'' {
			c.quote = line[index]
			continue
		}
		switch line[index] {
		case '[':
			c.brackets++
		case ']':
			c.brackets--
			if c.brackets <= 0 {
				return true
			}
		}
	}
	return c.brackets == 0 && c.quote == 0 && c.triple == ""
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
