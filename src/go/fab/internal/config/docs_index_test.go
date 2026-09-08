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
