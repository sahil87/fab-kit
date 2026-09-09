package config

import (
	"gopkg.in/yaml.v3"
	"reflect"
	"testing"
)

func TestDocsIndexRoots(t *testing.T) {
	for _, tc := range []struct {
		name, yaml string
		want       []DocsIndexRoot
		bad        bool
	}{
		{name: "implicit", want: DefaultDocsIndexRoots()},
		{name: "explicit sparse", yaml: "docs_index:\n  roots:\n    - path: docs/specs\n", want: []DocsIndexRoot{{Path: "docs/specs", IndexFile: "index.md", MaxDepth: 3}}},
		{name: "empty", yaml: "docs_index: {roots: []}", want: []DocsIndexRoot{}},
		{name: "escape", yaml: "docs_index: {roots: [{path: ../outside}]}", bad: true},
		{name: "overlap", yaml: "docs_index: {roots: [{path: docs}, {path: docs/specs}]}", bad: true},
		{name: "repo root", yaml: "docs_index: {roots: [{path: .}]}", want: []DocsIndexRoot{{Path: ".", IndexFile: "index.md", MaxDepth: 3}}},
		{name: "repo root first", yaml: "docs_index: {roots: [{path: .}, {path: docs/specs}]}", bad: true},
		{name: "repo root last normalized", yaml: "docs_index: {roots: [{path: docs/specs}, {path: ./}]}", bad: true},
		{name: "landing escape", yaml: "docs_index: {roots: [{path: docs, index_file: ../index.md}]}", bad: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			var c Config
			if err := yaml.Unmarshal([]byte(tc.yaml), &c); err != nil {
				t.Fatal(err)
			}
			got, err := c.GetDocsIndexRoots()
			if tc.bad {
				if err == nil {
					t.Fatal("want invalid config")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != len(tc.want) || len(got) > 0 && !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("got %+v, want %+v", got, tc.want)
			}
		})
	}
}

// TestDocsIndexRootNavNotePresence pins nav_note's contract: absent and
// `nav_note: ""` are both the empty string (render nothing), set round-trips
// verbatim (render verbatim).
func TestDocsIndexRootNavNotePresence(t *testing.T) {
	parse := func(t *testing.T, y string) DocsIndexRoot {
		t.Helper()
		var c Config
		if err := yaml.Unmarshal([]byte(y), &c); err != nil {
			t.Fatal(err)
		}
		roots, err := c.GetDocsIndexRoots()
		if err != nil {
			t.Fatal(err)
		}
		return roots[0]
	}

	if r := parse(t, "docs_index: {roots: [{path: docs/specs}]}"); r.NavNote != "" {
		t.Errorf("absent nav_note must be empty, got %q", r.NavNote)
	}
	if r := parse(t, `docs_index: {roots: [{path: docs/specs, nav_note: ""}]}`); r.NavNote != "" {
		t.Errorf("empty nav_note must be empty, got %q", r.NavNote)
	}
	if r := parse(t, "docs_index: {roots: [{path: docs/specs, nav_note: '> See the [Guide](../guide.md)'}]}"); r.NavNote != "> See the [Guide](../guide.md)" {
		t.Errorf("set nav_note must round-trip verbatim, got %q", r.NavNote)
	}
}

// TestDocsIndexRootExcludePresence pins exclude's contract: absent is nil,
// set round-trips verbatim.
func TestDocsIndexRootExcludePresence(t *testing.T) {
	parse := func(t *testing.T, y string) DocsIndexRoot {
		t.Helper()
		var c Config
		if err := yaml.Unmarshal([]byte(y), &c); err != nil {
			t.Fatal(err)
		}
		roots, err := c.GetDocsIndexRoots()
		if err != nil {
			t.Fatal(err)
		}
		return roots[0]
	}

	if r := parse(t, "docs_index: {roots: [{path: docs/specs}]}"); r.Exclude != nil {
		t.Errorf("absent exclude must be nil, got %v", r.Exclude)
	}
	if r := parse(t, `docs_index: {roots: [{path: docs/specs, exclude: ["assets/**", "**/*.png"]}]}`); !reflect.DeepEqual(r.Exclude, []string{"assets/**", "**/*.png"}) {
		t.Errorf("set exclude must round-trip verbatim, got %v", r.Exclude)
	}
}
