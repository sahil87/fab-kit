package agentmodels

import (
	"reflect"
	"testing"
)

func TestTOMLScanner(t *testing.T) {
	doc, err := parseTOML([]byte(`default_model = "top" # comment
other = 42
[models."quoted"]
display_name = "Quoted # name"
support_efforts = [ "low", "high", ]
[models.bare]
display_name = 'Bare'
[[hooks]]
command = "ignored"
`))
	if err != nil {
		t.Fatal(err)
	}
	if doc.top["default_model"] != "top" {
		t.Errorf("top = %v", doc.top)
	}
	if len(doc.tables) != 3 {
		t.Fatalf("tables = %+v", doc.tables)
	}
	if got := doc.tables[0].strings["display_name"]; got != "Quoted # name" {
		t.Errorf("display_name = %q", got)
	}
	if got := doc.tables[0].arrays["support_efforts"]; !reflect.DeepEqual(got, []string{"low", "high"}) {
		t.Errorf("efforts = %v", got)
	}
	if name, ok := kimiModelName(doc.tables[0].header); !ok || name != "quoted" {
		t.Errorf("quoted header = %q, %v", name, ok)
	}
	if name, ok := kimiModelName(doc.tables[1].header); !ok || name != "bare" {
		t.Errorf("bare header = %q, %v", name, ok)
	}
	if _, ok := kimiModelName(doc.tables[2].header); ok {
		t.Error("hook table recognized as a model")
	}
}

func TestTOMLScannerSkipsUnsupportedMultilineValues(t *testing.T) {
	doc, err := parseTOML([]byte(`default_model = "k3"
[providers.example]
ports = [1, 2]
"a=b" = 1
args = [
  "--flag",
  "[not-a-header]",
]
description = """
multi-line value
[also-not-a-header]
"""
display_name = 42
[models.real]
display_name = "Real"
support_efforts = ["low"]
`))
	if err != nil {
		t.Fatal(err)
	}
	if doc.top["default_model"] != "k3" {
		t.Errorf("top-level default = %q", doc.top["default_model"])
	}
	if len(doc.tables) != 2 || doc.tables[1].header != "models.real" {
		t.Fatalf("tables = %+v", doc.tables)
	}
	if got := doc.tables[1].strings["display_name"]; got != "Real" {
		t.Errorf("model display_name = %q", got)
	}
}

func TestTOMLScannerMalformedInput(t *testing.T) {
	for _, input := range []string{
		`[models."broken"`,
		"[models.broken]\ndisplay_name = \"unterminated",
		"[models.broken]\nsupport_efforts = [ \"low\" \"high\" ]",
		"default_model = unquoted",
		"[models.broken]\nsupport_efforts = high",
		"[providers.x]\nargs = [\n  \"never closed\"\n[models.lost]\ndisplay_name = \"Lost\"",
	} {
		if _, err := parseTOML([]byte(input)); err == nil {
			t.Errorf("parseTOML(%q) succeeded, want error", input)
		}
	}
}
