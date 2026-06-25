package memoryindex

// Minimal parser for EXISTING index files — the new I/O-free half of
// `--check`'s loss classifier. `--check` today only string-compares; to detect
// tombstones (dropped rows) and flattened groupings it must read the *current*
// index's rows and headings. This parser is deliberately bounded to exactly
// what the three loss categories need (markdown table rows + ATX headings),
// not a general markdown parser (see the change's plan.md inline SRAD
// assumption on parser scope).

import (
	"strings"
)

// indexRow is one parsed markdown table row of an index file: the link text,
// its target, and the first description cell. Rows without a `[text](target)`
// first cell (separator rows, header rows, non-table lines) are skipped.
type indexRow struct {
	// Text is the link label (the `[Text]` part).
	Text string
	// Target is the link target (the `(target)` part) verbatim.
	Target string
	// Description is the second table cell, trimmed (may be "" or "—").
	Description string
}

// parseIndexRows extracts the table rows of an index file. It recognizes the
// generated shapes — root `| [name](name/index.md) | desc |` and domain
// `| [base](base.md) | desc |` — plus any hand-curated row of the same
// `| [text](target) | desc | ... |` form (extra trailing cells are ignored, so
// a legacy 3-column domain row still parses to the same link + description).
// Header rows (`| Domain |`), separator rows (`|---|`), and non-row lines are
// ignored.
func parseIndexRows(content string) []indexRow {
	var rows []indexRow
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) == 0 {
			continue
		}
		text, target, ok := parseLinkCell(cells[0])
		if !ok {
			continue // first cell is not a [text](target) link → not a data row
		}
		desc := ""
		if len(cells) >= 2 {
			desc = strings.TrimSpace(cells[1])
		}
		rows = append(rows, indexRow{Text: text, Target: target, Description: desc})
	}
	return rows
}

// rowsByTarget indexes parsed rows by link target (last write wins) for O(1)
// existing-vs-rendered correlation in the description-loss detector.
func rowsByTarget(rows []indexRow) map[string]indexRow {
	m := make(map[string]indexRow, len(rows))
	for _, r := range rows {
		m[r.Target] = r
	}
	return m
}

// splitTableRow splits a `| a | b | c |` line into trimmed cells, dropping the
// leading/trailing empty cells produced by the boundary pipes.
func splitTableRow(line string) []string {
	parts := strings.Split(line, "|")
	// Drop the empty first/last segments from the boundary pipes.
	if len(parts) > 0 && strings.TrimSpace(parts[0]) == "" {
		parts = parts[1:]
	}
	if len(parts) > 0 && strings.TrimSpace(parts[len(parts)-1]) == "" {
		parts = parts[:len(parts)-1]
	}
	for i := range parts {
		parts[i] = strings.TrimSpace(parts[i])
	}
	return parts
}

// parseLinkCell parses a `[text](target)` markdown link cell, returning the
// text and target. ok is false when the cell is not a single link (e.g. a
// plain header label or a separator dash run).
func parseLinkCell(cell string) (text, target string, ok bool) {
	cell = strings.TrimSpace(cell)
	if !strings.HasPrefix(cell, "[") {
		return "", "", false
	}
	close := strings.Index(cell, "](")
	if close < 0 {
		return "", "", false
	}
	end := strings.Index(cell[close:], ")")
	if end < 0 {
		return "", "", false
	}
	text = cell[1:close]
	target = cell[close+2 : close+end]
	return text, target, true
}

// relMemoryTarget resolves an index row's link target to a docs/memory/-
// relative path for the on-disk tombstone check, given linkBase (the index
// file's directory relative to docs/memory/: "" for the root, "<domain>" for a
// domain index). It returns ok=false for external (scheme://) or absolute
// (/...) targets — those are intentional outbound links and never tombstones.
// A generated row's target is `name/index.md` (root) or `base.md`/`sub/index.md`
// (domain); the row "exists" when its containing folder/file is on disk, so we
// resolve to the directory the target lives in (its first path segment under
// linkBase), which is exactly what Gather's os.ReadDir walk would list.
func relMemoryTarget(linkBase, target string) (string, bool) {
	target = strings.TrimSpace(target)
	if target == "" {
		return "", false
	}
	// External or absolute → never a generated row.
	if strings.Contains(target, "://") || strings.HasPrefix(target, "/") {
		return "", false
	}
	// Strip a leading ./ ; reject parent-escaping links (they point outside the
	// index's own folder — not a row the generator would produce here).
	target = strings.TrimPrefix(target, "./")
	if strings.HasPrefix(target, "../") || target == ".." {
		return "", false
	}
	// Join with linkBase to get a docs/memory/-relative path, then take the
	// row's anchoring filesystem entry: for `name/index.md` that is the `name`
	// folder; for `base.md` that is the file itself.
	full := target
	if linkBase != "" {
		full = linkBase + "/" + target
	}
	// The generator drops a row when its on-disk anchor is gone. For a folder
	// link (`.../index.md`) the anchor is the folder; for a file link the
	// anchor is the file. Use the path minus a trailing `/index.md`.
	anchor := strings.TrimSuffix(full, "/index.md")
	if anchor == full {
		// Not a folder-index link → the file itself is the anchor.
		anchor = full
	}
	return anchor, true
}

// parseStructuralHeadings returns the `## ` / `### ` ATX headings in the root
// index that are NOT part of the generated output (the generated root has no
// `##`/`###` headings; `## Sub-Domains` is a domain-index heading, excluded
// defensively). These are the custom groupings the domains-only regen drops.
func parseStructuralHeadings(content string) []string {
	var out []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			continue
		}
		heading := strings.TrimSpace(strings.TrimLeft(line, "#"))
		if heading == "Sub-Domains" {
			continue // generated domain-index heading, never a custom grouping
		}
		out = append(out, heading)
	}
	return out
}
