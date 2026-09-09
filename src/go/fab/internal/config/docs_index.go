package config

import (
	"fmt"
	"path/filepath"
	"strings"
)

// DocsIndexConfig is project-owned navigation configuration.
type DocsIndexConfig struct {
	Roots []DocsIndexRoot `yaml:"roots"`
}
type DocsIndexRoot struct {
	Path       string   `yaml:"path" json:"path"`
	IndexFile  string   `yaml:"index_file" json:"index_file"`
	AlsoAccept []string `yaml:"also_accept,omitempty" json:"also_accept,omitempty"`
	Log        bool     `yaml:"log" json:"log"`
	MaxDepth   int      `yaml:"max_depth" json:"max_depth"`
	Superseded []string `yaml:"superseded,omitempty" json:"superseded,omitempty"`
	// NavNote is the root landing's navigation note (free-text markdown).
	// Presence is intent: nil renders the built-in default (the legacy "New
	// here?" line on the legacy memory root, nothing on a generic root), a
	// pointer to "" omits the line, and a pointer to non-empty renders the
	// value verbatim.
	NavNote *string `yaml:"nav_note,omitempty" json:"nav_note,omitempty"`
}

func DefaultDocsIndexRoots() []DocsIndexRoot {
	return []DocsIndexRoot{{Path: "docs/memory", IndexFile: "index.md", Log: true, MaxDepth: 3}}
}

func (c *Config) GetDocsIndexRoots() ([]DocsIndexRoot, error) {
	if c == nil || c.DocsIndex == nil {
		return DefaultDocsIndexRoots(), nil
	}
	roots := append([]DocsIndexRoot(nil), c.DocsIndex.Roots...)
	seen := map[string]bool{}
	for i := range roots {
		r := &roots[i]
		if r.Path == "" || filepath.IsAbs(r.Path) || !filepath.IsLocal(r.Path) {
			return nil, fmt.Errorf("docs_index.roots: path %q must be repo-relative", r.Path)
		}
		r.Path = filepath.ToSlash(filepath.Clean(r.Path))
		if seen[r.Path] {
			return nil, fmt.Errorf("docs_index.roots: duplicate path %q", r.Path)
		}
		seen[r.Path] = true
		if r.IndexFile == "" {
			r.IndexFile = "index.md"
		}
		if r.MaxDepth == 0 {
			r.MaxDepth = 3
		}
		if r.MaxDepth < 0 {
			return nil, fmt.Errorf("docs_index.roots: max_depth for %s must be positive", r.Path)
		}
		for _, name := range append([]string{r.IndexFile}, r.AlsoAccept...) {
			if name == "." || filepath.Base(name) != name || !strings.HasSuffix(name, ".md") || name == "log.md" || name == "log.seed.md" {
				return nil, fmt.Errorf("docs_index.roots: invalid landing filename %q", name)
			}
		}
	}
	for i, r := range roots {
		for _, s := range roots[i+1:] {
			if r.Path == "." || s.Path == "." || strings.HasPrefix(r.Path+"/", s.Path+"/") || strings.HasPrefix(s.Path+"/", r.Path+"/") {
				return nil, fmt.Errorf("docs_index.roots: overlapping paths %q and %q", r.Path, s.Path)
			}
		}
	}
	return roots, nil
}
