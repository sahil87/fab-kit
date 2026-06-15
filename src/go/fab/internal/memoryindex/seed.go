package memoryindex

// FKF log.md seed-merge (docs/specs/fkf.md §6, oovf — the cutover crux).
//
// Pre-FKF `## Changelog` rows carry rich, hand-curated history with their OWN
// authored dates and no live `.status.yaml` `summary:` to regenerate from (the
// changes are shipped/archived). DECISION b (preserve the 651 rows faithfully)
// therefore needs the generator to accept a curated SEED input and merge it into
// the generated log.md beneath the git-projected entries.
//
// The seed is a per-folder sidecar file `log.seed.md` in the FKF §6.2 entry
// format. It is an INPUT — curated, like `description:` frontmatter — never the
// generated output, so the single-writer / byte-stable discipline (FKF §5/§6) is
// preserved: `fab memory-index` stays the sole writer of log.md, the seed is just
// another gathered input it reads (never writes). This file holds the pure half
// (parseSeedLog + mergeSeedEntries); the read-from-disk wiring lives in
// memoryindex.go's GatherLogs/buildLogTarget alongside the other I/O.

import (
	"strings"
)

// seedFileName is the per-folder seed sidecar `fab memory-index` reads (never
// writes). It sits alongside the generated log.md; it is excluded from topic-file
// gathering exactly as index.md / log.md are.
const seedFileName = "log.seed.md"

// parseSeedLog parses a `log.seed.md` body into LogEntry values. The seed is in
// the FKF §6.2 rendered shape (the inverse of RenderLog), so a parse∘render round
// trip is the identity on well-formed seed entries:
//
//	## 2026-06-12
//	- **Update** [hydrate-specs](/memory-docs/hydrate-specs.md) — No-target branch added … (d9rs)
//
// The leading bold verb and the trailing `(id)` token are both optional (§6.2);
// the descriptive line is everything between the ` — ` separator and an optional
// trailing `(id)`. The seed's own `## YYYY-MM-DD` heading is preserved verbatim
// as the entry Date (the pre-FKF changelog `Date` column, independent of git).
// Lines that are not a date heading or an entry bullet are ignored (the
// generated header comment, blank lines, stray prose). Pure function.
func parseSeedLog(content string) []LogEntry {
	var entries []LogEntry
	curDate := ""
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if strings.HasPrefix(line, "## ") {
			curDate = strings.TrimSpace(line[len("## "):])
			continue
		}
		if !strings.HasPrefix(line, "- ") || curDate == "" {
			continue
		}
		if e, ok := parseSeedEntry(line, curDate); ok {
			entries = append(entries, e)
		}
	}
	return entries
}

// parseSeedEntry parses one `- {**Verb** }[base](/path.md) — summary{ (id)}`
// bullet under date into a LogEntry. Returns ok=false when the line is not a
// well-formed entry (no link cell, or no ` — ` descriptive separator).
func parseSeedEntry(line, date string) (LogEntry, bool) {
	body := strings.TrimSpace(strings.TrimPrefix(line, "- "))

	// Optional leading bold verb.
	verb := ""
	for _, v := range []string{verbCreation, verbDeprecation, verbUpdate} {
		if strings.HasPrefix(body, v+" ") {
			verb = v
			body = strings.TrimSpace(strings.TrimPrefix(body, v))
			break
		}
	}

	// Link cell: [base](/bundle/rel.md).
	if !strings.HasPrefix(body, "[") {
		return LogEntry{}, false
	}
	closeBracket := strings.Index(body, "](")
	if closeBracket < 0 {
		return LogEntry{}, false
	}
	fileBase := body[1:closeBracket]
	rest := body[closeBracket+len("]("):]
	closeParen := strings.Index(rest, ")")
	if closeParen < 0 {
		return LogEntry{}, false
	}
	bundleRel := rest[:closeParen]
	rest = rest[closeParen+1:]

	// Descriptive line: text after the " — " separator, minus an optional
	// trailing "(id)" token. RenderLog always emits exactly " — " between the
	// link and the summary, so split on the first occurrence.
	sep := strings.Index(rest, " — ")
	if sep < 0 {
		return LogEntry{}, false
	}
	desc := strings.TrimSpace(rest[sep+len(" — "):])

	id, summary := splitTrailingID(desc)
	return LogEntry{
		Date:          date,
		Verb:          verb,
		FileBase:      fileBase,
		BundleRelPath: bundleRel,
		Summary:       summary,
		ChangeID:      id,
	}, true
}

// splitTrailingID peels a trailing `(id)` token off a descriptive line, returning
// the bare id and the summary without it. The id token is recognized only when it
// is the LAST parenthesized group AND looks like a change-id token (no spaces) —
// so an in-prose "(#42)" PR ref or a "(some note)" aside is NOT mistaken for the
// id, matching how the renderer only appends a (ChangeID) when one is present.
// A missing-cell "—" summary round-trips to "".
func splitTrailingID(desc string) (id, summary string) {
	summary = strings.TrimSpace(desc)
	if strings.HasSuffix(summary, ")") {
		open := strings.LastIndex(summary, "(")
		if open >= 0 {
			tok := summary[open+1 : len(summary)-1]
			if tok != "" && !strings.ContainsAny(tok, " \t") {
				id = tok
				summary = strings.TrimSpace(summary[:open])
			}
		}
	}
	if summary == missingCell {
		summary = ""
	}
	return id, summary
}

// mergeSeedEntries unions git-projected entries with parsed seed entries for one
// folder's log.md, de-duplicating any seed entry byte-equal to a projected entry
// (same Date / FileBase / BundleRelPath / ChangeID / Summary / Verb) so a re-run
// is a no-op (Constitution III idempotency). The result is handed to RenderLog,
// whose stable date-group sort orders the output deterministically — so this
// function only needs to concatenate-then-dedupe; it relies on RenderLog for
// final ordering. Projected entries are kept ahead of seed entries in the slice
// so that, within a date, git-projected lines render before seed lines (RenderLog
// preserves input order within a date via its stable sort).
func mergeSeedEntries(projected, seed []LogEntry) []LogEntry {
	seen := make(map[LogEntry]bool, len(projected))
	out := make([]LogEntry, 0, len(projected)+len(seed))
	for _, e := range projected {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	for _, e := range seed {
		if !seen[e] {
			seen[e] = true
			out = append(out, e)
		}
	}
	return out
}
