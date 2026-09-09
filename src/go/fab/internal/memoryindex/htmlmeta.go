package memoryindex

// HTML metadata extraction for topic rows: the <title> is a LABEL source only
// (never a description — adoption replaces a manual row's description whenever
// the generated one is non-"—", and a title-as-description would clobber
// hand-written manual rows on every regeneration), and the description comes
// solely from <meta name="description">. Standard library only: two tags do
// not justify a tokenizer dependency (go.mod stays at cobra + yaml).

import (
	"html"
	"os"
	"strings"
)

// htmlHeadMeta scans an HTML document's head region — bounded by </head> or
// the first <body — for the first <title>…</title> and the first <meta …> tag
// carrying name="description" (either attribute order, single or double
// quotes). The whole head is scanned rather than a fixed byte prefix, so a
// self-contained page with a large inline <style> before <title> is still
// handled. Values are entity-unescaped and whitespace-collapsed. An unreadable
// file yields two empty strings (graceful degradation, like readH1).
func htmlHeadMeta(path string) (title, description string) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", ""
	}
	head := string(data)
	lower := strings.ToLower(head)
	end := len(head)
	for _, marker := range []string{"</head>", "<body"} {
		if i := strings.Index(lower, marker); i >= 0 && i < end {
			end = i
		}
	}
	head, lower = head[:end], lower[:end]

	if i := tagStart(lower, "<title"); i >= 0 {
		open := strings.IndexByte(lower[i:], '>')
		if open >= 0 {
			rest := i + open + 1
			if j := strings.Index(lower[rest:], "</title>"); j >= 0 {
				title = collapseHTMLWhitespace(html.UnescapeString(head[rest : rest+j]))
			}
		}
	}

	for search := 0; ; {
		i := tagStart(lower[search:], "<meta")
		if i < 0 {
			break
		}
		i += search
		tagEnd := scanTagEnd(lower, i)
		if tagEnd < 0 {
			break
		}
		tag := head[i : tagEnd+1]
		if name, ok := htmlAttr(tag, "name"); ok && strings.EqualFold(name, "description") {
			if content, ok := htmlAttr(tag, "content"); ok {
				description = collapseHTMLWhitespace(html.UnescapeString(content))
			}
			break
		}
		search = tagEnd + 1
	}
	return title, description
}

// tagStart finds the next occurrence of an opening tag named by prefix (e.g.
// "<title") whose following byte ends the name — '>', whitespace, or '/' — so
// "<titlex" never matches "<title". Returns -1 when absent.
func tagStart(lower, prefix string) int {
	for search := 0; ; {
		i := strings.Index(lower[search:], prefix)
		if i < 0 {
			return -1
		}
		i += search
		j := i + len(prefix)
		if j >= len(lower) || lower[j] == '>' || lower[j] == '/' || lower[j] == ' ' || lower[j] == '\t' || lower[j] == '\n' || lower[j] == '\r' {
			return i
		}
		search = j
	}
}

// scanTagEnd returns the index of the '>' closing the tag that starts at i
// ("<…"), skipping over quoted attribute values so a '>' inside quotes does
// not end the tag early. Returns -1 when the tag never closes.
func scanTagEnd(lower string, i int) int {
	var quote byte
	for j := i; j < len(lower); j++ {
		c := lower[j]
		if quote != 0 {
			if c == quote {
				quote = 0
			}
			continue
		}
		if c == '"' || c == '\'' {
			quote = c
		} else if c == '>' {
			return j
		}
	}
	return -1
}

// htmlAttr extracts an attribute value from a tag, case-insensitively on the
// attribute name, accepting single- or double-quoted values. The name must be
// a standalone attribute (preceded by whitespace) so "data-name" never matches
// "name". The returned value is still entity-escaped.
func htmlAttr(tag, attr string) (string, bool) {
	lower := strings.ToLower(tag)
	for search := 0; ; {
		i := strings.Index(lower[search:], attr)
		if i < 0 {
			return "", false
		}
		i += search
		if i == 0 || lower[i-1] != ' ' && lower[i-1] != '\t' && lower[i-1] != '\n' && lower[i-1] != '\r' && lower[i-1] != '/' {
			search = i + len(attr)
			continue
		}
		j := i + len(attr)
		if j >= len(tag) || tag[j] != '=' {
			search = j
			continue
		}
		j++ // past '='
		if j >= len(tag) || tag[j] != '"' && tag[j] != '\'' {
			search = j
			continue
		}
		quote := tag[j]
		k := strings.IndexByte(tag[j+1:], quote)
		if k < 0 {
			return "", false
		}
		return tag[j+1 : j+1+k], true
	}
}

// collapseHTMLWhitespace folds every whitespace run to a single space.
func collapseHTMLWhitespace(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
