package memoryindex

import (
	"fmt"
	"path"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

// matchGlob adds recursive ** segments to path.Match's shell-glob syntax.
// Literal brackets use escaping (\[archived\]) or bracket classes ([[]archived]).
func matchGlob(pattern, name string) bool {
	p, n := strings.Split(strings.Trim(pattern, "/"), "/"), strings.Split(strings.Trim(name, "/"), "/")
	var match func([]string, []string) bool
	match = func(p, n []string) bool {
		if len(p) == 0 {
			return len(n) == 0
		}
		if p[0] == "**" {
			return match(p[1:], n) || len(n) > 0 && match(p, n[1:])
		}
		if len(n) == 0 {
			return false
		}
		ok, _ := path.Match(p[0], n[0])
		return ok && match(p[1:], n[1:])
	}
	return match(p, n)
}

// matchAny reports whether name matches any of the root-relative slash glob
// patterns — the single matcher shared by superseded and exclude.
func matchAny(patterns []string, name string) bool {
	for _, p := range patterns {
		if matchGlob(p, name) {
			return true
		}
	}
	return false
}

func validateGlobs(field string, patterns []string) error {
	for _, p := range patterns {
		if p == "" || strings.HasPrefix(p, "/") {
			return fmt.Errorf("docs_index.roots.%s: invalid relative glob %q", field, p)
		}
		for _, segment := range strings.Split(p, "/") {
			if segment == ".." {
				return fmt.Errorf("docs_index.roots.%s: parent traversal in %q", field, p)
			}
			if _, err := path.Match(segment, ""); err != nil {
				return fmt.Errorf("docs_index.roots.%s: %q: %w", field, p, err)
			}
		}
	}
	return nil
}

var versionFolder = regexp.MustCompile(`^v([0-9]+)$`)

func supersededSummary(children []*docFolder, count int) string {
	versions := []int{}
	for _, c := range children {
		m := versionFolder.FindStringSubmatch(c.data.Name)
		if m == nil {
			return fmt.Sprintf("Superseded — %d files", count)
		}
		v, err := strconv.Atoi(m[1])
		if err != nil {
			return fmt.Sprintf("Superseded — %d files", count)
		}
		versions = append(versions, v)
	}
	if len(versions) == 0 {
		return fmt.Sprintf("Superseded — %d files", count)
	}
	sort.Ints(versions)
	return fmt.Sprintf("%d superseded versions (v%d–v%d), %d files", len(versions), versions[0], versions[len(versions)-1], count)
}
