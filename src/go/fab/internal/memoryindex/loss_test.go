package memoryindex

import (
	"testing"
)

// alwaysExists / neverExists are the two trivial memExists predicates the
// classifier tests use; targeted tests supply their own set-membership func.
func setExists(present map[string]bool) func(string) bool {
	return func(rel string) bool { return present[rel] }
}

// --- Tier 0: clean ---------------------------------------------------------

func TestClassify_Clean_NoDrift(t *testing.T) {
	content := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Auth domain |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/index.md", Existing: content, Rendered: content, IsRoot: true},
	}, setExists(map[string]bool{"auth": true}))

	if report.Tier != TierClean {
		t.Errorf("identical existing/rendered → TierClean, got %d", report.Tier)
	}
	if report.Drift {
		t.Error("no byte difference → Drift must be false")
	}
	if len(report.Losses) != 0 {
		t.Errorf("clean tree → no losses, got %v", report.Losses)
	}
}

// --- Tier 1: benign drift (improved description / refreshed date) ----------

func TestClassify_BenignDrift_ImprovedDescription(t *testing.T) {
	existing := "" +
		"| File | Description | Last Updated |\n" +
		"|------|-------------|-------------|\n" +
		"| [login](login.md) | old desc | 2026-01-01 |\n"
	rendered := "" +
		"| File | Description | Last Updated |\n" +
		"|------|-------------|-------------|\n" +
		"| [login](login.md) | improved desc | 2026-06-15 |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/index.md", Existing: existing, Rendered: rendered, LinkBase: "auth"},
	}, setExists(map[string]bool{"auth/login.md": true}))

	if report.Tier != TierBenignDrift {
		t.Errorf("improved description (still present) → TierBenignDrift, got %d", report.Tier)
	}
	if !report.Drift {
		t.Error("content differs → Drift must be true")
	}
	if len(report.Losses) != 0 {
		t.Errorf("benign drift → no destructive losses, got %v", report.Losses)
	}
}

// --- Tier 2 category 1: curated description → "—" --------------------------

func TestClassify_DescriptionLoss(t *testing.T) {
	existing := "" +
		"| File | Description | Last Updated |\n" +
		"|------|-------------|-------------|\n" +
		"| [login](login.md) | Curated login flow | 2026-01-01 |\n"
	// Regen: same row, but description gone to "—" (frontmatter missing).
	rendered := "" +
		"| File | Description | Last Updated |\n" +
		"|------|-------------|-------------|\n" +
		"| [login](login.md) | — | 2026-01-01 |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/index.md", Existing: existing, Rendered: rendered, LinkBase: "auth"},
	}, setExists(map[string]bool{"auth/login.md": true}))

	if report.Tier != TierDestructiveLoss {
		t.Fatalf("description→— → TierDestructiveLoss, got %d (losses %v)", report.Tier, report.Losses)
	}
	if len(report.Losses) != 1 || report.Losses[0].Category != LossDescription {
		t.Fatalf("want one description loss, got %v", report.Losses)
	}
	if report.Losses[0].Detail != "Curated login flow" {
		t.Errorf("loss Detail should be the lost curated text, got %q", report.Losses[0].Detail)
	}
}

func TestClassify_DescriptionAlreadyMissing_NotALoss(t *testing.T) {
	// Existing already "—" → nothing curated to lose even though the row drifts.
	existing := "" +
		"| File | Description | Last Updated |\n" +
		"|------|-------------|-------------|\n" +
		"| [login](login.md) | — | 2026-01-01 |\n"
	rendered := "" +
		"| File | Description | Last Updated |\n" +
		"|------|-------------|-------------|\n" +
		"| [login](login.md) | — | 2026-06-15 |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/index.md", Existing: existing, Rendered: rendered, LinkBase: "auth"},
	}, setExists(map[string]bool{"auth/login.md": true}))

	if report.Tier != TierBenignDrift {
		t.Errorf("existing already — (date drift only) → TierBenignDrift, got %d", report.Tier)
	}
}

// --- Tier 2 category 2: tombstone row dropped ------------------------------

func TestClassify_TombstoneLoss_RootFolderGone(t *testing.T) {
	existing := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Auth |\n" +
		"| [lib-bdash](lib-bdash/index.md) | Removed in #123 |\n"
	rendered := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Auth |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/index.md", Existing: existing, Rendered: rendered, IsRoot: true},
	}, setExists(map[string]bool{"auth": true})) // lib-bdash absent

	if report.Tier != TierDestructiveLoss {
		t.Fatalf("dropped on-disk-absent row → TierDestructiveLoss, got %d (losses %v)", report.Tier, report.Losses)
	}
	var got *Loss
	for i := range report.Losses {
		if report.Losses[i].Category == LossTombstone {
			got = &report.Losses[i]
		}
	}
	if got == nil {
		t.Fatalf("want a tombstone loss, got %v", report.Losses)
	}
	if got.Detail != "lib-bdash/index.md" {
		t.Errorf("tombstone Detail should be the dropped link target, got %q", got.Detail)
	}
}

func TestClassify_Tombstone_StruckThrough(t *testing.T) {
	// Strikethrough is a corroborating hint, not required — disk-resolution is
	// the signal. A struck row whose folder is gone is still a tombstone.
	existing := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [~~old~~](old/index.md) | gone | \n" +
		"| [auth](auth/index.md) | Auth |\n"
	rendered := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Auth |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/index.md", Existing: existing, Rendered: rendered, IsRoot: true},
	}, setExists(map[string]bool{"auth": true}))

	if report.Tier != TierDestructiveLoss {
		t.Errorf("struck tombstone with absent folder → TierDestructiveLoss, got %d", report.Tier)
	}
}

func TestClassify_ExternalLinks_NeverTombstone(t *testing.T) {
	// An external/absolute link target must never false-positive as a tombstone,
	// even when the index drifts for another (benign) reason.
	existing := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [docs](https://example.com/docs) | external |\n" +
		"| [abs](/etc/passwd) | absolute |\n" +
		"| [auth](auth/index.md) | old |\n"
	rendered := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [docs](https://example.com/docs) | external |\n" +
		"| [abs](/etc/passwd) | absolute |\n" +
		"| [auth](auth/index.md) | new |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/index.md", Existing: existing, Rendered: rendered, IsRoot: true},
	}, setExists(map[string]bool{"auth": true})) // example.com / etc not on disk

	for _, l := range report.Losses {
		if l.Category == LossTombstone {
			t.Errorf("external/absolute link reported as tombstone: %v", l)
		}
	}
	if report.Tier != TierBenignDrift {
		t.Errorf("only-benign drift among external links → TierBenignDrift, got %d", report.Tier)
	}
}

// --- Tier 2 category 3: custom grouping flattened --------------------------

func TestClassify_GroupingLoss(t *testing.T) {
	existing := "" +
		"# Memory Index\n\n" +
		"### Apps\n\n" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Auth |\n\n" +
		"### Packages\n\n"
	rendered := "" +
		"# Memory Index\n\n" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Auth |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/index.md", Existing: existing, Rendered: rendered, IsRoot: true},
	}, setExists(map[string]bool{"auth": true}))

	if report.Tier != TierDestructiveLoss {
		t.Fatalf("custom root headings → TierDestructiveLoss, got %d (losses %v)", report.Tier, report.Losses)
	}
	var headings []string
	for _, l := range report.Losses {
		if l.Category == LossGrouping {
			headings = append(headings, l.Detail)
		}
	}
	if len(headings) != 2 {
		t.Fatalf("want 2 grouping losses (Apps, Packages), got %v", headings)
	}
}

func TestClassify_SubDomainsHeading_NotGrouping(t *testing.T) {
	// `## Sub-Domains` is a generated domain-index heading — it must never count
	// as a custom grouping. (And grouping only runs on the root anyway.)
	got := parseStructuralHeadings("# X\n\n## Sub-Domains\n\n| a | b |\n")
	if len(got) != 0 {
		t.Errorf("Sub-Domains must be excluded from structural headings, got %v", got)
	}
}

// --- Highest-tier-wins + invariants ---------------------------------------

func TestClassify_HighestTierWins(t *testing.T) {
	// One file benign-drifts, another has a destructive loss → tier 2 overall.
	benignExisting := "| [x](x.md) | old | d |\n"
	benignRendered := "| [x](x.md) | new | d |\n"
	lossExisting := "| [y](y.md) | Curated | d |\n"
	lossRendered := "| [y](y.md) | — | d |\n"
	report := Classify([]CheckTarget{
		{Path: "a/index.md", Existing: benignExisting, Rendered: benignRendered, LinkBase: "a"},
		{Path: "b/index.md", Existing: lossExisting, Rendered: lossRendered, LinkBase: "b"},
	}, setExists(map[string]bool{"a/x.md": true, "b/y.md": true}))

	if report.Tier != TierDestructiveLoss {
		t.Errorf("any destructive loss → tier 2 overall, got %d", report.Tier)
	}
}

func TestClassify_Invariants_Tier2HasLoss_Tier1HasDrift(t *testing.T) {
	// Tier 2 ⇒ at least one loss; tier 1 ⇒ drift true, zero losses.
	loss := Classify([]CheckTarget{
		{Path: "b/index.md", Existing: "| [y](y.md) | Curated | d |\n", Rendered: "| [y](y.md) | — | d |\n", LinkBase: "b"},
	}, setExists(map[string]bool{"b/y.md": true}))
	if loss.Tier == TierDestructiveLoss && len(loss.Losses) == 0 {
		t.Error("tier 2 must carry at least one loss")
	}

	benign := Classify([]CheckTarget{
		{Path: "a/index.md", Existing: "| [x](x.md) | old | d |\n", Rendered: "| [x](x.md) | new | d |\n", LinkBase: "a"},
	}, setExists(map[string]bool{"a/x.md": true}))
	if benign.Tier == TierBenignDrift && (!benign.Drift || len(benign.Losses) != 0) {
		t.Errorf("tier 1 must have Drift=true and zero losses, got Drift=%v losses=%v", benign.Drift, benign.Losses)
	}
}

// --- Parser unit tests -----------------------------------------------------

func TestParseIndexRows_GeneratedShapes(t *testing.T) {
	rootRow := parseIndexRows("| [auth](auth/index.md) | Auth domain |\n")
	if len(rootRow) != 1 || rootRow[0].Target != "auth/index.md" || rootRow[0].Description != "Auth domain" {
		t.Errorf("root row parse mismatch: %+v", rootRow)
	}
	domRow := parseIndexRows("| [login](login.md) | Login flow | 2026-06-01 |\n")
	if len(domRow) != 1 || domRow[0].Text != "login" || domRow[0].Target != "login.md" || domRow[0].Description != "Login flow" {
		t.Errorf("domain row parse mismatch: %+v", domRow)
	}
}

func TestParseIndexRows_SkipsHeaderAndSeparator(t *testing.T) {
	content := "" +
		"| Domain | Description |\n" +
		"|--------|-------------|\n" +
		"| [auth](auth/index.md) | Auth |\n"
	rows := parseIndexRows(content)
	if len(rows) != 1 {
		t.Errorf("header + separator must be skipped, got %d rows: %+v", len(rows), rows)
	}
}

func TestRelMemoryTarget(t *testing.T) {
	cases := []struct {
		base, target, want string
		ok                 bool
	}{
		{"", "auth/index.md", "auth", true},
		{"", "lib-bdash/index.md", "lib-bdash", true},
		{"auth", "login.md", "auth/login.md", true},
		{"auth", "sub/index.md", "auth/sub", true},
		{"", "https://example.com", "", false},
		{"", "/abs/path", "", false},
		{"auth", "../other.md", "", false},
		{"", "", "", false},
	}
	for _, c := range cases {
		got, ok := relMemoryTarget(c.base, c.target)
		if ok != c.ok || got != c.want {
			t.Errorf("relMemoryTarget(%q,%q) = (%q,%v), want (%q,%v)", c.base, c.target, got, ok, c.want, c.ok)
		}
	}
}
