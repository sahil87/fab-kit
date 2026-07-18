package memoryindex

// Destructive-loss classification for `fab memory-index --check`.
//
// The existing --check branch already computes the rendered-vs-existing drift
// per index file (a string compare). What it cannot do is *classify* that
// drift: distinguish a benign improvement (a better description) from a
// destructive loss (a curated description wiped to "—", a tombstone row
// silently dropped, a custom grouping flattened). This file adds that
// classifier as pure functions — the mechanical form of the three prose
// signals 5ewp's /docs-reorg-memory detected by eye.
//
// All functions here are pure except the on-disk tombstone existence check,
// which takes the directory contents already gathered by the caller (so the
// classifier itself performs no I/O — Classify receives a `pathExists`
// predicate the cmd supplies). This mirrors the RenderRoot/Gather split:
// rendering/classification is pure and unit-testable; the cmd owns the I/O.

import (
	"strings"
)

// LossCategory names one of the three destructive-loss kinds.
type LossCategory string

const (
	// LossDescription: an existing index row renders a non-empty description
	// but the regenerated row would render "—" (the file lacks `description:`
	// frontmatter) — curated text wiped on regen.
	LossDescription LossCategory = "description"
	// LossTombstone: an existing index row whose docs/memory/-relative link
	// target is absent on disk — the generator (which lists only on-disk
	// folders/files) silently drops the row on regen.
	LossTombstone LossCategory = "tombstone"
	// LossGrouping: a structural heading in the existing root index beyond the
	// generated domains-only table — flattened by the domains-only regen.
	LossGrouping LossCategory = "grouping"
)

// Loss is one destructive-loss finding.
type Loss struct {
	// Category is the loss kind.
	Category LossCategory `json:"category"`
	// Path is the repo-relative index file the loss is about.
	Path string `json:"path"`
	// Detail is the lost content: the description text, the dropped row's link
	// target, or the flattened heading.
	Detail string `json:"detail"`
}

// Tier is the severity exit-code tier: 0 clean, 1 benign drift, 2 destructive
// loss. It maps directly onto the process exit code.
type Tier int

const (
	// TierClean — no regeneration needed; every index is byte-identical.
	TierClean Tier = 0
	// TierBenignDrift — regen would change something but destroys nothing.
	TierBenignDrift Tier = 1
	// TierDestructiveLoss — regen would wipe curated/historical content.
	TierDestructiveLoss Tier = 2
)

// MalformedFinding is one malformed-frontmatter finding surfaced by `--check`.
// It is a SOURCE-file corruption (a mangled `description:` frontmatter block),
// orthogonal to the index-drift Tier scheme: it blocks `--check` (floors the
// exit at 1) independent of drift, but it is NOT a destructive-loss (tier-2)
// category and does NOT extend the `losses[]` category enum — so the
// hydrate/reorg refuse-before-regen guards (keyed on tier == 2) never fire on
// mere source corruption, and `/docs-reorg-memory`'s `losses` consumer is
// unaffected (intake #9 / plan Design Decision 1).
type MalformedFinding struct {
	// Kind is the malformed Warning kind (KindMalformedFence /
	// KindMalformedDescription).
	Kind string `json:"kind"`
	// Path is the repo-relative memory file the corruption is in.
	Path string `json:"path"`
	// Detail is the offending frontmatter value (for the description kind), "".
	Detail string `json:"detail,omitempty"`
}

// WarningFinding is one ADVISORY warning surfaced by `--check --json` (the
// machine surface [dsrx]'s survey/reorg extensions consume instead of parsing
// stderr). It carries the advisory kinds only — narration density, file size,
// _unsorted staging, and broken links; the BLOCKING kinds ride the Malformed
// array instead. Additive to the `tier`/`drift`/`losses`/`malformed` contract —
// existing consumers ignore it, and it never affects the exit code.
type WarningFinding struct {
	// Kind is the advisory Warning kind (KindNarrationDensity / KindFileSize /
	// KindUnsorted / KindBrokenLink).
	Kind string `json:"kind"`
	// Path is the repo-relative folder/file the finding is about.
	Path string `json:"path"`
	// Count is the finding's numeric measure (marker count / line count /
	// staged-file count); 0 when the kind carries no count (broken-link).
	Count int `json:"count"`
	// Bytes is the observed byte size for KindFileSize (so a byte-bound-only trip
	// — over 15KB but under the line cap — is explicable in JSON rather than
	// showing an under-cap line Count with no reason for the finding); 0 for
	// every other kind. Omitted from JSON when 0.
	Bytes int `json:"bytes,omitempty"`
	// Detail is the broken-link target for KindBrokenLink; "" otherwise.
	Detail string `json:"detail,omitempty"`
}

// LossReport is the full classification of a --check run. It is the value
// emitted by `--check --json` and the source of the exit code.
type LossReport struct {
	// Tier is the highest INDEX-DRIFT severity found (0/1/2). Unchanged by the
	// blocking-frontmatter class — corruption/escalations are in Malformed below.
	Tier Tier `json:"tier"`
	// Drift is true when any index file differs from its regenerated form
	// (true for tier 1 and tier 2; tier 2 is a strict subset of drift).
	Drift bool `json:"drift"`
	// Losses enumerates every destructive-loss finding (empty unless tier 2).
	Losses []Loss `json:"losses"`
	// Malformed enumerates the BLOCKING findings (malformed frontmatter +
	// the two description escalations — registry-gated change-id, gross
	// over-cap). Additive to the `tier`/`drift`/`losses` contract — existing
	// consumers ignore it. Non-empty ⇒ `--check` blocks (exit ≥ 1) regardless
	// of Tier. The JSON key stays `malformed` for consumer compatibility even
	// though the internal blocking set is now broader than pure corruption.
	Malformed []MalformedFinding `json:"malformed"`
	// Warnings enumerates the ADVISORY findings (density / size / _unsorted /
	// broken links). Additive; never affects the exit code. Empty-never-null.
	Warnings []WarningFinding `json:"warnings"`
}

// CheckTarget is one index file's comparison inputs: its repo-relative path,
// the existing on-disk content, and the freshly-rendered content. The cmd
// builds these from the same `targets` slice the --check branch already walks.
type CheckTarget struct {
	// Path is the repo-relative index file path (for loss reporting).
	Path string
	// Existing is the current on-disk content ("" if the file is absent).
	Existing string
	// Rendered is the content `fab memory-index` would write.
	Rendered string
	// IsRoot marks the root docs/memory/index.md (grouping detection only runs
	// there — domain/sub-domain indexes have no custom-grouping category).
	IsRoot bool
	// IsLog marks a generated log.md target (FKF §6). A log.md is a C-lite git
	// projection, not a row-table index, so its drift is always BENIGN (tier 1):
	// the description/tombstone/grouping detectors are index-row-shaped and would
	// be meaningless on log list entries. No new tier-2 category is introduced
	// for log.md / FKF frontmatter (intake OQ4 / assumption #9) — a born-FKF tree
	// is provably never tier 2, so drift on a generated log.md is a stale-regen
	// signal, never destructive loss.
	IsLog bool
	// LinkBase is the index file's directory relative to docs/memory/ (""
	// for the root, "<domain>" for a domain index, "<domain>/<sub>" for a
	// sub-domain index). A row link target is resolved against it to a
	// docs/memory/-relative path for the on-disk tombstone check.
	LinkBase string
}

// Classify compares each target's existing vs. rendered content and returns
// the severity report. memExists(relPath) reports whether a docs/memory/-
// relative path (folder or file) exists on disk — supplied by the cmd so this
// function stays pure. The highest tier across all targets wins.
func Classify(targets []CheckTarget, memExists func(relPath string) bool) LossReport {
	// Losses / Malformed / Warnings are initialized non-nil so the --json shape
	// is always `"losses": []` / `"malformed": []` / `"warnings": []` (not
	// `null`), matching the contract. Malformed and Warnings are populated by the
	// cmd (from the gathered warnings) after Classify — they are source/advisory
	// classes, not index-drift findings.
	report := LossReport{Tier: TierClean, Losses: []Loss{}, Malformed: []MalformedFinding{}, Warnings: []WarningFinding{}}

	for _, t := range targets {
		if t.Existing == t.Rendered {
			continue // byte-identical — no drift for this file
		}
		report.Drift = true

		// A log.md is a generated C-lite projection, not a row-table index — its
		// drift is always benign (tier 1). Skip the index-row destructive-loss
		// detectors for it (intake OQ4 / assumption #9: no new tier-2 category).
		if t.IsLog {
			continue
		}

		// Destructive-loss detectors run only when this file actually drifts.
		report.Losses = append(report.Losses, descriptionLosses(t)...)
		report.Losses = append(report.Losses, tombstoneLosses(t, memExists)...)
		if t.IsRoot {
			report.Losses = append(report.Losses, groupingLosses(t)...)
		}
	}

	switch {
	case len(report.Losses) > 0:
		report.Tier = TierDestructiveLoss
	case report.Drift:
		report.Tier = TierBenignDrift
	default:
		report.Tier = TierClean
	}
	return report
}

// descriptionLosses reports every existing row whose description cell is a
// real (non-empty, non-"—") curated value but whose regenerated counterpart
// for the same link target renders "—" — the curated text is wiped on regen.
func descriptionLosses(t CheckTarget) []Loss {
	rendered := rowsByTarget(parseIndexRows(t.Rendered))
	var out []Loss
	for _, ex := range parseIndexRows(t.Existing) {
		exDesc := strings.TrimSpace(ex.Description)
		if exDesc == "" || exDesc == missingCell {
			continue // nothing curated to lose
		}
		rRow, ok := rendered[ex.Target]
		if !ok {
			continue // row absent in regen → a tombstone (handled separately)
		}
		if strings.TrimSpace(rRow.Description) == missingCell {
			out = append(out, Loss{Category: LossDescription, Path: t.Path, Detail: exDesc})
		}
	}
	return out
}

// tombstoneLosses reports every existing row whose docs/memory/-relative link
// target is absent on disk. External (http(s)://) and absolute (/...) targets
// never count — they are intentional outbound links, not generated rows.
func tombstoneLosses(t CheckTarget, memExists func(relPath string) bool) []Loss {
	var out []Loss
	for _, ex := range parseIndexRows(t.Existing) {
		rel, ok := relMemoryTarget(t.LinkBase, ex.Target)
		if !ok {
			continue // external / absolute / unparseable → never a tombstone
		}
		if !memExists(rel) {
			out = append(out, Loss{Category: LossTombstone, Path: t.Path, Detail: ex.Target})
		}
	}
	return out
}

// groupingLosses reports custom structural headings in the existing root index
// that the generated domains-only output omits — they flatten on regen.
func groupingLosses(t CheckTarget) []Loss {
	var out []Loss
	for _, h := range parseStructuralHeadings(t.Existing) {
		out = append(out, Loss{Category: LossGrouping, Path: t.Path, Detail: h})
	}
	return out
}
