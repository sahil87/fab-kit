package internal

import (
	"fmt"
	"os"
	"sort"
	"strings"
)

// migrationToSep is the separator between FROM and TO in a migration filename:
// {FROM}-to-{TO}.md.
const migrationToSep = "-to-"

// migrationSuffix is the required extension for migration instruction files.
const migrationSuffix = ".md"

// MigrationRange is one parsed migration file: {From}-to-{To}.md.
type MigrationRange struct {
	From string // semver, e.g. "1.9.7"
	To   string // semver, e.g. "1.10.0"
	File string // base filename, e.g. "1.9.7-to-1.10.0.md"
}

// DiscoverResult is the full outcome of a discovery pass.
type DiscoverResult struct {
	Local      string           // fab/.kit-migration-version
	Engine     string           // target/engine VERSION
	Applicable []MigrationRange // ordered list to apply, FROM ascending
	GapSkips   []string         // human-readable "no migration for X -> Y, skipping"
	Overlaps   []string         // pairs of overlapping filenames (non-empty => malformed set)
}

// parseMigrationFilename matches {FROM}-to-{TO}.md and parses both parts as
// semver. It returns false for any name that does not match the convention
// (e.g. ".gitkeep", "README.md", malformed names).
func parseMigrationFilename(name string) (MigrationRange, bool) {
	if !strings.HasSuffix(name, migrationSuffix) {
		return MigrationRange{}, false
	}
	stem := strings.TrimSuffix(name, migrationSuffix)

	idx := strings.Index(stem, migrationToSep)
	if idx < 0 {
		return MigrationRange{}, false
	}
	from := stem[:idx]
	to := stem[idx+len(migrationToSep):]
	if !isSemver(from) || !isSemver(to) {
		return MigrationRange{}, false
	}
	return MigrationRange{From: from, To: to, File: name}, true
}

// isSemver reports whether s is a bare MAJOR.MINOR.PATCH triplet of integers.
// It rejects empty parts and non-numeric segments so non-migration filenames
// (e.g. "README") are not mistaken for ranges.
func isSemver(s string) bool {
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return false
	}
	for _, p := range parts {
		if p == "" {
			return false
		}
		for _, c := range p {
			if c < '0' || c > '9' {
				return false
			}
		}
	}
	return true
}

// DiscoverMigrations scans migrationsDir for {FROM}-to-{TO}.md files, validates
// that no two ranges overlap, sorts the candidates by FROM ascending, and walks
// the discovery loop starting at local to produce the ordered applicable chain
// and any gap-skips. The engine version is recorded on the result for callers.
//
// Discovery loop:
//  1. find the first migration where FROM <= current < TO -> append, current = TO, repeat
//  2. else if a later migration exists with FROM > current -> record a gap-skip,
//     advance current to that FROM, repeat
//  3. else -> done
//
// Overlap is reported (Overlaps non-empty), not silently resolved: a malformed
// migration set must not be guessed at.
func DiscoverMigrations(migrationsDir, local, engine string) (DiscoverResult, error) {
	result := DiscoverResult{Local: local, Engine: engine}

	entries, err := os.ReadDir(migrationsDir)
	if err != nil {
		return result, fmt.Errorf("cannot read migrations dir %s: %w", migrationsDir, err)
	}

	var ranges []MigrationRange
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if r, ok := parseMigrationFilename(e.Name()); ok {
			ranges = append(ranges, r)
		}
	}

	// Sort by FROM ascending (then TO ascending) for a deterministic walk and
	// stable overlap-pair ordering.
	sort.Slice(ranges, func(i, j int) bool {
		if c := compareSemver(ranges[i].From, ranges[j].From); c != 0 {
			return c < 0
		}
		return compareSemver(ranges[i].To, ranges[j].To) < 0
	})

	// Detect overlapping ranges: A.From < B.To && B.From < A.To.
	for i := 0; i < len(ranges); i++ {
		for j := i + 1; j < len(ranges); j++ {
			a, b := ranges[i], ranges[j]
			if compareSemver(a.From, b.To) < 0 && compareSemver(b.From, a.To) < 0 {
				result.Overlaps = append(result.Overlaps,
					fmt.Sprintf("%s and %s", a.File, b.File))
			}
		}
	}

	// Walk the discovery loop.
	current := local
	for {
		// (1) first migration where FROM <= current < TO.
		if r, ok := firstApplicable(ranges, current); ok {
			result.Applicable = append(result.Applicable, r)
			current = r.To
			continue
		}
		// (2) a later migration with FROM > current.
		if r, ok := nextAhead(ranges, current); ok {
			result.GapSkips = append(result.GapSkips,
				fmt.Sprintf("No migration needed for %s -> %s, skipping.", current, r.From))
			current = r.From
			continue
		}
		// (3) done.
		break
	}

	return result, nil
}

// firstApplicable returns the first range (in sorted order) where
// FROM <= current < TO.
func firstApplicable(ranges []MigrationRange, current string) (MigrationRange, bool) {
	for _, r := range ranges {
		if compareSemver(r.From, current) <= 0 && compareSemver(current, r.To) < 0 {
			return r, true
		}
	}
	return MigrationRange{}, false
}

// nextAhead returns the earliest range (in sorted order) whose FROM is strictly
// ahead of current, used to log a gap-skip and advance.
func nextAhead(ranges []MigrationRange, current string) (MigrationRange, bool) {
	for _, r := range ranges {
		if compareSemver(r.From, current) > 0 {
			return r, true
		}
	}
	return MigrationRange{}, false
}
