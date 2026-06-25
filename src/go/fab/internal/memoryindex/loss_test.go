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
		"| File | Description |\n" +
		"|------|-------------|\n" +
		"| [login](login.md) | old desc |\n"
	rendered := "" +
		"| File | Description |\n" +
		"|------|-------------|\n" +
		"| [login](login.md) | improved desc |\n"
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
		"| File | Description |\n" +
		"|------|-------------|\n" +
		"| [login](login.md) | Curated login flow |\n"
	// Regen: same row, but description gone to "—" (frontmatter missing).
	rendered := "" +
		"| File | Description |\n" +
		"|------|-------------|\n" +
		"| [login](login.md) | — |\n"
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
	// The login row's description is already "—" → nothing curated to lose, even
	// though the file drifts (a sibling row's description improved).
	existing := "" +
		"| File | Description |\n" +
		"|------|-------------|\n" +
		"| [login](login.md) | — |\n" +
		"| [signup](signup.md) | old desc |\n"
	rendered := "" +
		"| File | Description |\n" +
		"|------|-------------|\n" +
		"| [login](login.md) | — |\n" +
		"| [signup](signup.md) | improved desc |\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/index.md", Existing: existing, Rendered: rendered, LinkBase: "auth"},
	}, setExists(map[string]bool{"auth/login.md": true, "auth/signup.md": true}))

	if report.Tier != TierBenignDrift {
		t.Errorf("login already — (sibling improvement drift only) → TierBenignDrift, got %d", report.Tier)
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
	benignExisting := "| [x](x.md) | old |\n"
	benignRendered := "| [x](x.md) | new |\n"
	lossExisting := "| [y](y.md) | Curated |\n"
	lossRendered := "| [y](y.md) | — |\n"
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
		{Path: "b/index.md", Existing: "| [y](y.md) | Curated |\n", Rendered: "| [y](y.md) | — |\n", LinkBase: "b"},
	}, setExists(map[string]bool{"b/y.md": true}))
	if loss.Tier == TierDestructiveLoss && len(loss.Losses) == 0 {
		t.Error("tier 2 must carry at least one loss")
	}

	benign := Classify([]CheckTarget{
		{Path: "a/index.md", Existing: "| [x](x.md) | old |\n", Rendered: "| [x](x.md) | new |\n", LinkBase: "a"},
	}, setExists(map[string]bool{"a/x.md": true}))
	if benign.Tier == TierBenignDrift && (!benign.Drift || len(benign.Losses) != 0) {
		t.Errorf("tier 1 must have Drift=true and zero losses, got Drift=%v losses=%v", benign.Drift, benign.Losses)
	}
}

// --- log.md targets: always benign drift, never tier 2 (FKF, OQ4) ---------

func TestClassify_LogDrift_IsBenign(t *testing.T) {
	// A drifted log.md (stale C-lite projection) is benign drift (tier 1) — its
	// list entries are not index table rows, and no new tier-2 category exists.
	existing := "# Log — Auth\n<!-- gen -->\n\n## 2026-01-01\n- [login](/auth/login.md) — old summary (abcd)\n"
	rendered := "# Log — Auth\n<!-- gen -->\n\n## 2026-06-15\n- [login](/auth/login.md) — new summary (abcd)\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/log.md", Existing: existing, Rendered: rendered, LinkBase: "auth", IsLog: true},
	}, setExists(map[string]bool{"auth/login.md": true}))

	if report.Tier != TierBenignDrift {
		t.Errorf("drifted log.md → TierBenignDrift, got %d (losses %v)", report.Tier, report.Losses)
	}
	if len(report.Losses) != 0 {
		t.Errorf("log.md drift must produce no destructive losses, got %v", report.Losses)
	}
}

func TestClassify_LogHandText_NotTier2(t *testing.T) {
	// A hand-curated log the C-lite projection can't reproduce is STILL benign —
	// the log surface introduces no tier-2 category (intake assumption #9 / OQ4).
	// (Crafted so that, were the index detectors to run, the bundle-relative /-
	// link could not even false-positive; the IsLog guard is what guarantees it.)
	existing := "# Log — Auth\n\n## 2026-01-01\n- Curated prose entry a human wrote (xyz1)\n"
	rendered := "# Log — Auth\n<!-- gen -->\n\n## 2026-06-15\n- [login](/auth/login.md) — generated (abcd)\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/log.md", Existing: existing, Rendered: rendered, LinkBase: "auth", IsLog: true},
	}, setExists(map[string]bool{"auth/login.md": true}))

	if report.Tier == TierDestructiveLoss {
		t.Errorf("log.md drift must never be tier 2, got losses %v", report.Losses)
	}
	if report.Tier != TierBenignDrift {
		t.Errorf("hand-edited log.md → TierBenignDrift, got %d", report.Tier)
	}
}

func TestClassify_LogClean_NoDrift(t *testing.T) {
	// A born-FKF log.md (byte-identical to its regenerated form) is tier 0.
	content := "# Log — Auth\n<!-- gen -->\n\n## 2026-06-15\n- [login](/auth/login.md) — s (abcd)\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/log.md", Existing: content, Rendered: content, IsLog: true},
	}, setExists(map[string]bool{}))
	if report.Tier != TierClean || report.Drift {
		t.Errorf("byte-identical log.md → TierClean, got tier %d drift %v", report.Tier, report.Drift)
	}
}

// TestClassify_SeedMergeLogDrift_IsBenign pins R5 (oovf): a log.md whose drift is
// driven by a seed-merge — the rendered content gains preserved pre-FKF seed
// entries the existing on-disk file lacks — is BENIGN drift (tier 1), never tier 2.
// The merged seed must NOT be reported as destructive loss; the IsLog guard keeps
// the index-row detectors from ever running on the log's list entries.
func TestClassify_SeedMergeLogDrift_IsBenign(t *testing.T) {
	// On-disk: only the git-projected entry. Rendered: the seed entry (an older,
	// authored date carrying pre-FKF history) merged beneath it.
	existing := "# Log — Auth\n<!-- gen -->\n\n## 2026-06-15\n- **Update** [login](/auth/login.md) — recent (abcd)\n"
	rendered := "# Log — Auth\n<!-- gen -->\n\n## 2026-06-15\n- **Update** [login](/auth/login.md) — recent (abcd)\n" +
		"\n## 2026-02-09\n- **Creation** [login](/auth/login.md) — initial pre-FKF creation (h3v7)\n"
	report := Classify([]CheckTarget{
		{Path: "docs/memory/auth/log.md", Existing: existing, Rendered: rendered, LinkBase: "auth", IsLog: true},
	}, setExists(map[string]bool{"auth/login.md": true}))

	if report.Tier != TierBenignDrift {
		t.Errorf("seed-merge log.md drift → TierBenignDrift, got %d (losses %v)", report.Tier, report.Losses)
	}
	if len(report.Losses) != 0 {
		t.Errorf("a preserved seed must never be reported as destructive loss, got %v", report.Losses)
	}
}

// --- Parser unit tests -----------------------------------------------------

func TestParseIndexRows_GeneratedShapes(t *testing.T) {
	rootRow := parseIndexRows("| [auth](auth/index.md) | Auth domain |\n")
	if len(rootRow) != 1 || rootRow[0].Target != "auth/index.md" || rootRow[0].Description != "Auth domain" {
		t.Errorf("root row parse mismatch: %+v", rootRow)
	}
	domRow := parseIndexRows("| [login](login.md) | Login flow |\n")
	if len(domRow) != 1 || domRow[0].Text != "login" || domRow[0].Target != "login.md" || domRow[0].Description != "Login flow" {
		t.Errorf("domain row parse mismatch: %+v", domRow)
	}
	// A legacy 3-column domain row (a not-yet-migrated committed index) still
	// parses to the same link + description — the extra trailing cell is ignored.
	legacyRow := parseIndexRows("| [login](login.md) | Login flow | 2026-06-01 |\n")
	if len(legacyRow) != 1 || legacyRow[0].Target != "login.md" || legacyRow[0].Description != "Login flow" {
		t.Errorf("legacy 3-column row parse mismatch: %+v", legacyRow)
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
