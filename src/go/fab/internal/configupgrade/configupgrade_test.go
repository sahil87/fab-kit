package configupgrade

// Behavioral tests over the SHIPPED registry (configref.Fields). These assert the
// engine's contract properties against the real field set — complementing the
// full-document goldens in golden_test.go, which pin exact bytes over a small
// synthetic set. Split out so a real-registry prose edit churns neither the goldens
// nor these (which assert behavior, not bytes).

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/agent"
	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

// fieldsForTest returns the shipped registry, failing the test on a construction
// error (the registry lint / role-profile invariant).
func fieldsForTest(t *testing.T) []configref.Field {
	t.Helper()
	f, err := configref.Fields()
	if err != nil {
		t.Fatalf("configref.Fields: %v", err)
	}
	return f
}

// TestRender_AppendsFenceToLegacyFile: a legacy config.yaml with no fence gets a
// managed fence appended at the bottom, with byte-exact anchors, while the user's
// live keys and their own comments above are preserved verbatim.
func TestRender_AppendsFenceToLegacyFile(t *testing.T) {
	fields := fieldsForTest(t)
	legacy := `project:
    name: fab-kit
    description: FAB Kit

# pin review to fable — sonnet missed doc-claim regressions
agent:
    tiers:
        review:
            model: claude-fable-5
            effort: xhigh
`
	out, _ := render(legacy, fields, ProjectTarget(""), "2.15.0")

	// The user's preamble (live keys + their comment) is preserved verbatim as a prefix.
	if !strings.HasPrefix(out, legacy) {
		t.Errorf("legacy preamble must be preserved verbatim as a prefix.\n--- got ---\n%s", out)
	}
	// Byte-exact BEGIN/END anchors (dash-padded to fenceWidth).
	wantBegin := "# >>> fab reference (kit 2.15.0) >>> " + strings.Repeat("-", fenceWidth-len("# >>> fab reference (kit 2.15.0) >>> "))
	wantEnd := "# <<< end fab reference <<< " + strings.Repeat("-", fenceWidth-len("# <<< end fab reference <<< "))
	if !strings.Contains(out, wantBegin) {
		t.Errorf("missing byte-exact BEGIN anchor %q", wantBegin)
	}
	if !strings.Contains(out, wantEnd) {
		t.Errorf("missing byte-exact END anchor %q", wantEnd)
	}
	// Ends in exactly one trailing newline.
	if !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
		t.Error("output must end in exactly one trailing newline")
	}
	// The rendered document must parse as YAML.
	if err := validateYAML(out); err != nil {
		t.Errorf("rendered legacy output does not parse: %v", err)
	}
}

// TestRender_FenceOmitsOverriddenFields: a field the user has overridden live above
// the fence is NOT re-advertised inside the fence (it shows what you could override
// but haven't). agent.profiles is live here, and it shares the `agent:` top-level key
// with the two advertised depth knobs, so the fence must not scaffold `agent:` at all.
func TestRender_FenceOmitsOverriddenFields(t *testing.T) {
	fields := fieldsForTest(t)
	src := `agent:
    profiles:
        review:
            model: claude-fable-5
`
	out, _ := render(src, fields, ProjectTarget(""), "2.15.0")

	_, fenceBody, _ := sliceFence(t, out)
	if strings.Contains(fenceBody, "agent:") {
		t.Errorf("fence must omit the already-overridden agent field.\n--- fence ---\n%s", fenceBody)
	}
	// But an un-overridden advertise field IS scaffolded (fully commented).
	if !strings.Contains(fenceBody, "# dispatch:") {
		t.Errorf("fence must advertise the un-overridden dispatch field.\n--- fence ---\n%s", fenceBody)
	}
}

// TestRender_FenceDemotesAgentMachinery pins the 260806-j9nh fence-slimming
// requirement: the advertised agent surface is exactly the two depth knobs, so an
// un-overridden project's fence carries `agent: session/workers` and NO `providers:`
// or `agent.profiles` scaffold. Those rows keep their registry entries and their
// `fab config explain` segments — they are demoted from the per-project fence
// only.
func TestRender_FenceDemotesAgentMachinery(t *testing.T) {
	fields := fieldsForTest(t)
	out, _ := render("", fields, ProjectTarget(""), "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)

	for _, want := range []string{"agent.session", "agent.workers"} {
		if !strings.Contains(fenceBody, want) {
			t.Errorf("fence must advertise %q.\n--- fence ---\n%s", want, fenceBody)
		}
	}
	// The machinery keys must not be SCAFFOLDED. Their segments open with a
	// `# providers:` / `#   profiles:` block line, which is what a scaffold looks
	// like; the knob segment's prose may still NAME them in a pointer line.
	for _, absent := range []string{"# providers:", "#   profiles:"} {
		if strings.Contains(fenceBody, absent) {
			t.Errorf("fence must not scaffold %q (demoted to advertise:false).\n--- fence ---\n%s", absent, fenceBody)
		}
	}
}

// TestRender_FenceFullyComments the C-field scaffold, INCLUDING parent keys — a
// live `agent:` over comment-only children is the `agent: null` masher bug the
// fence design exists to prevent. Every non-blank line inside the fence body is a
// comment.
func TestRender_FenceFullyComments(t *testing.T) {
	fields := fieldsForTest(t)
	out, _ := render("", fields, ProjectTarget(""), "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)

	for _, ln := range strings.Split(fenceBody, "\n") {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(ln), "#") {
			t.Errorf("fence body must be fully commented; found live line %q", ln)
		}
	}
	// A parent key must appear only in commented form (never a live `providers:`).
	if strings.Contains(fenceBody, "\nproviders:") {
		t.Error("fence must not carry a LIVE providers: parent key (must be commented)")
	}
}

// TestCommentOutSegment_MarkersAtColumnZero: EVERY non-blank line of a commented
// segment carries its comment marker at column 0 — the alignment defect. A
// deliberately-commented CONTENT line (an indented `#`, e.g. the agent block's
// `  # profiles:` example lines) must gain the fence-level
// `# ` prefix like any live line; only a line whose `#` is ALREADY at column 0 is
// fence-level prose and is left as-is.
func TestCommentOutSegment_MarkersAtColumnZero(t *testing.T) {
	segment := "# prose at column 0\n" +
		"#\n" +
		"live_key:\n" +
		"  child: 1\n" +
		"    # deliberately-commented content at column 4\n" +
		"  # commented block at column 2\n" +
		"  #   nested: value"
	want := "# prose at column 0\n" +
		"#\n" +
		"# live_key:\n" +
		"#   child: 1\n" +
		"#     # deliberately-commented content at column 4\n" +
		"#   # commented block at column 2\n" +
		"#   #   nested: value"
	if got := CommentOutSegment(segment); got != want {
		t.Errorf("comment markers must all land at column 0.\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

// TestCommentOutSegment_ShippedRegistryAlignment: over the SHIPPED registry, every
// non-blank line of every commented segment starts with `#` at column 0 — the
// property the fence's visual alignment rests on. The guard that a new registry row
// carrying deliberately-commented content (the agent block's commented
// agent.profiles example lines) cannot reintroduce a ragged fence.
// Both renderings are walked: the long Segment (explain) and the file-bound
// ShortSegment (fence/scaffold/materializer, 260809-wll4 R6/R7).
func TestCommentOutSegment_ShippedRegistryAlignment(t *testing.T) {
	for _, f := range fieldsForTest(t) {
		for form, seg := range map[string]string{"Segment": f.Segment, "ShortSegment": f.ShortSegment} {
			if seg == "" {
				continue
			}
			for i, ln := range strings.Split(CommentOutSegment(seg), "\n") {
				if strings.TrimSpace(ln) == "" {
					continue
				}
				if !strings.HasPrefix(ln, "#") {
					t.Errorf("field %q %s line %d is not commented at column 0: %q", f.Key, form, i, ln)
				}
			}
		}
	}
}

// TestCommentOutSegment_BlockStripRestoresSegment: the reverse operation
// `configref.providersSegment`'s prose promises — "strip the leading '# ' from every
// line of a block" — restores the segment's YAML block BYTE-EXACTLY, so a user who
// uncomments a whole block gets valid YAML (the providers block restores to all
// four built-in provider blocks LIVE at their original indent). Verified over the
// shipped registry; the fence-level prose lines (already column-0) are unchanged by
// the commenting and so are not part of the strip. Both renderings are walked:
// the long Segment (explain) and the file-bound ShortSegment (260809-wll4 R6/R7).
func TestCommentOutSegment_BlockStripRestoresSegment(t *testing.T) {
	for _, f := range fieldsForTest(t) {
		for form, seg := range map[string]string{"Segment": f.Segment, "ShortSegment": f.ShortSegment} {
			if seg == "" {
				continue
			}
			orig := strings.Split(seg, "\n")
			got := strings.Split(CommentOutSegment(seg), "\n")
			for i, want := range orig {
				if strings.TrimSpace(want) == "" || strings.HasPrefix(want, "#") {
					if got[i] != want {
						t.Errorf("field %q %s line %d: a blank/prose line must pass through unchanged: %q → %q", f.Key, form, i, want, got[i])
					}
					continue
				}
				if !strings.HasPrefix(got[i], "# ") {
					t.Errorf("field %q %s line %d: a block line must gain the `# ` prefix: %q", f.Key, form, i, got[i])
					continue
				}
				if restored := strings.TrimPrefix(got[i], "# "); restored != want {
					t.Errorf("field %q %s line %d: stripping `# ` must restore the segment byte-exactly: %q, want %q", f.Key, form, i, restored, want)
				}
			}
		}
	}
}

// TestRender_ParksUnknownLiveKey: a live top-level key absent from the registry is
// removed from the live YAML and parked in a comment block below the fence, its
// value serialized. The live key must be gone from the active config.
func TestRender_ParksUnknownLiveKey(t *testing.T) {
	fields := fieldsForTest(t)
	src := `project:
    name: t

legacy_mode: true
`
	out, report := render(src, fields, ProjectTarget(""), "2.15.0")

	preamble, _, postfence := sliceFence(t, out)
	if strings.Contains(preamble, "legacy_mode:") {
		t.Errorf("unknown live key must be removed from the active config.\n--- preamble ---\n%s", preamble)
	}
	if !strings.Contains(postfence, "# removed in") || !strings.Contains(postfence, "#   legacy_mode: true") {
		t.Errorf("unknown key must be parked below the fence with its value.\n--- postfence ---\n%s", postfence)
	}
	// The user's real field is preserved.
	if !strings.Contains(preamble, "name: t") {
		t.Errorf("the user's live project field must be preserved.\n--- preamble ---\n%s", preamble)
	}
	if len(report) == 0 || !strings.Contains(strings.Join(report, "\n"), "legacy_mode") {
		t.Errorf("the report must note the parked key, got %v", report)
	}
}

// TestRender_PreservesUserCommentOnLiveField: a user's own comment on a live A
// field is preserved byte-for-byte (outside the fence is the user's).
func TestRender_PreservesUserCommentOnLiveField(t *testing.T) {
	fields := fieldsForTest(t)
	comment := "# pin review to fable — sonnet missed doc-claim regressions"
	src := comment + `
agent:
    tiers:
        review:
            model: claude-fable-5
`
	out, _ := render(src, fields, ProjectTarget(""), "2.15.0")
	if !strings.Contains(out, comment) {
		t.Errorf("user comment on a live field must be preserved verbatim.\n--- got ---\n%s", out)
	}
}

// TestRender_EmptyFileWritesFenceOnly: a fresh (empty) file becomes a fence-only
// document — no preamble, valid anchors, ends in one newline.
func TestRender_EmptyFileWritesFenceOnly(t *testing.T) {
	fields := fieldsForTest(t)
	out, _ := render("", fields, ProjectTarget(""), "2.15.0")
	if !strings.HasPrefix(out, "# >>> fab reference (kit 2.15.0) >>> ") {
		t.Errorf("empty file should start with the BEGIN anchor, got:\n%s", out)
	}
	if !strings.HasSuffix(out, "\n") || strings.HasSuffix(out, "\n\n") {
		t.Error("output must end in exactly one trailing newline")
	}
}

// TestRender_BelowFenceLiveOverrideHoisted (R2.1 regression, the review's must-fix
// #1): a live override the user appends BELOW the fence is HOISTED above it and
// preserved — never silently dropped. This runs against the REAL registry, so the
// subject must be a currently-registered advertised field (`true_impact_exclude`);
// an unregistered key would be parked rather than hoisted and would test nothing.
func TestRender_BelowFenceLiveOverrideHoisted(t *testing.T) {
	fields := fieldsForTest(t)
	first, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.15.0")
	withBelow := first + "\ntrue_impact_exclude:\n    - docs/\n"

	out, _ := render(withBelow, fields, ProjectTarget(""), "2.15.0")
	if !strings.Contains(out, "true_impact_exclude:\n    - docs/") {
		t.Fatalf("a live override appended below the fence must NOT be dropped.\n--- got ---\n%s", out)
	}
	preamble, fenceBody, _ := sliceFence(t, out)
	if !strings.Contains(preamble, "true_impact_exclude:\n    - docs/") {
		t.Errorf("below-fence override must be hoisted ABOVE the fence (live A field).\n--- preamble ---\n%s", preamble)
	}
	if strings.Contains(fenceBody, "true_impact_exclude") {
		t.Errorf("the now-live true_impact_exclude must be omitted from the regenerated fence.\n--- fence ---\n%s", fenceBody)
	}
	// Idempotent: a third run over the hoisted document is byte-identical.
	third, _ := render(out, fields, ProjectTarget(""), "2.15.0")
	if third != out {
		t.Errorf("hoisted below-fence content must be idempotent.\n--- out ---\n%s\n--- third ---\n%s", out, third)
	}
}

// TestRender_BelowFenceUnknownKeyParked: an UNKNOWN key the user appends below the
// fence is hoisted, then parked (not left dangling below the fence, and not dropped).
func TestRender_BelowFenceUnknownKeyParked(t *testing.T) {
	fields := fieldsForTest(t)
	first, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.15.0")
	withBelow := first + "\nmy_custom_key: 99\n"

	out, report := render(withBelow, fields, ProjectTarget(""), "2.15.0")
	_, _, postfence := sliceFence(t, out)
	if !strings.Contains(postfence, "#   my_custom_key: 99") {
		t.Errorf("an unknown key appended below the fence must be parked, not dropped.\n--- postfence ---\n%s", postfence)
	}
	if len(report) == 0 || !strings.Contains(strings.Join(report, "\n"), "my_custom_key") {
		t.Errorf("report must note the parked below-fence key, got %v", report)
	}
}

// TestRender_InteriorColumn0CommentInLiveBlock (SF-c): a live block with an interior
// column-0 comment between its indented lines is captured as ONE block — the
// trailing indented lines are not orphaned. The rendered output parses as YAML.
func TestRender_InteriorColumn0CommentInLiveBlock(t *testing.T) {
	fields := fieldsForTest(t)
	// An unknown live block whose value carries an interior column-0 comment.
	src := "custom_block:\n    a: 1\n# a comment the user wrote inside the block\n    b: 2\n"
	out, _ := render(src, fields, ProjectTarget(""), "2.15.0")
	_, _, postfence := sliceFence(t, out)
	// Both indented children must be parked together (b must not orphan above the fence).
	for _, want := range []string{"#   custom_block:", "#       a: 1", "#       b: 2"} {
		if !strings.Contains(postfence, want) {
			t.Errorf("interior-comment block must be parked whole; missing %q.\n--- postfence ---\n%s", want, postfence)
		}
	}
	if err := validateYAML(out); err != nil {
		t.Errorf("rendered output must parse as YAML: %v\n%s", err, out)
	}
}

// TestRender_BHygieneFlagsEqualsDefault (SF-d): a live field whose value equals the
// built-in default is flagged in the advisory report — but never removed
// (presence=intent). Uses providers, whose default is fab-kit's FOUR built-in
// providers (claude's session command; the non-claude command pairs) — so the live
// fixture must restate all four to be an equals-default case.
func TestRender_BHygieneFlagsEqualsDefault(t *testing.T) {
	fields := fieldsForTest(t)
	// Most built-ins ship per-role fills (260806-ywkx), so an equals-default
	// fixture must restate every one. They are DERIVED from agent.ResolveProvider
	// rather than typed out, so a model bump does not turn this into a false
	// negative that silently stops exercising the equals-default path — and a fill
	// added to any provider is picked up here automatically.
	src := "providers:\n"
	for _, name := range agent.ProviderNames(nil) {
		prov, _ := agent.ResolveProvider(nil, name)
		src += "    " + name + ":\n"
		// Single-quoted with interior quotes DOUBLED: the nested-shell dispatch
		// commands (`sh -c '… -p "$(cat)"'`) contain single quotes, and naive
		// wrapping would close the scalar early and make this fixture unparseable.
		// The rule has one owner — configref, the renderer this fixture mirrors.
		if prov.InteractiveCommand != "" {
			src += "        interactive_command: " + configref.YAMLSingleQuoted(prov.InteractiveCommand) + "\n"
		}
		if prov.HeadlessCommand != "" {
			src += "        headless_command: " + configref.YAMLSingleQuoted(prov.HeadlessCommand) + "\n"
		}
		if prov.Native {
			src += "        native: true\n"
		}
		if len(prov.Profiles) == 0 {
			continue
		}
		src += "        profiles:\n"
		for _, role := range agent.RoleNames() {
			fill, ok := prov.Profiles[role]
			if !ok {
				continue
			}
			// Render only the fields the fill carries — a sparse effort-only entry
			// or a model-only one must round-trip as itself, not gain an empty key.
			var set []string
			if fill.Model != "" {
				set = append(set, "model: "+fill.Model)
			}
			if fill.Effort != "" {
				set = append(set, "effort: "+fill.Effort)
			}
			src += "            " + role + ": { " + strings.Join(set, ", ") + " }\n"
		}
	}
	out, report := render(src, fields, ProjectTarget(""), "2.15.0")

	if !strings.Contains(out, "providers:") {
		t.Error("the live providers field must be kept (presence=intent — never auto-removed)")
	}
	joined := strings.Join(report, "\n")
	if !strings.Contains(joined, "providers") || !strings.Contains(joined, "equals the current default") {
		t.Errorf("B-hygiene must flag providers==default, got %v", report)
	}
}

// TestRender_BHygieneSilentOnRealOverride: a genuinely different live value is NOT
// flagged (no false positive).
func TestRender_BHygieneSilentOnRealOverride(t *testing.T) {
	fields := fieldsForTest(t)
	src := "providers:\n    claude:\n        interactive_command: 'my-custom-agent --flag'\n"
	_, report := render(src, fields, ProjectTarget(""), "2.15.0")
	if strings.Contains(strings.Join(report, "\n"), "equals the current default") {
		t.Errorf("a real override must not be flagged as equals-default, got %v", report)
	}
}

// TestRender_FenceAdvertsCarryScopeAnnotations (260809-wll4 R6/R7): every field
// advert rendered into the managed fence carries its [project|system|both] scope
// tag sourced from the registry row, a `fab config explain <key>` pointer, and —
// for the preference-class (scope both) adverts — the `fab config set --system`
// machine-wide pointer instead of the old invite-to-uncomment essay phrasing.
// Runs over the SHIPPED registry.
func TestRender_FenceAdvertsCarryScopeAnnotations(t *testing.T) {
	fields := fieldsForTest(t)
	out, _ := render("", fields, ProjectTarget(""), "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)

	advertised := 0
	for _, f := range fields {
		if !f.Advertise || f.ShortSegment == "" {
			continue
		}
		advertised++
		if tag := "[" + string(f.Scope) + "]"; !strings.Contains(fenceBody, tag) {
			t.Errorf("fence must carry the scope tag %q for advertised field %q.\n--- fence ---\n%s", tag, f.Key, fenceBody)
		}
	}
	if advertised == 0 {
		t.Fatal("no advertised fields — the fence would render empty")
	}
	if !strings.Contains(fenceBody, "# Full prose: fab config explain ") {
		t.Errorf("fence adverts must point at `fab config explain <key>` for the full prose.\n--- fence ---\n%s", fenceBody)
	}
	if !strings.Contains(fenceBody, "# Settable machine-wide: fab config set --system ") {
		t.Errorf("scope-both fence adverts must point at `fab config set --system <key> <value>`.\n--- fence ---\n%s", fenceBody)
	}
	// The diet: the fence no longer carries the long-form machine-wide phrasing
	// that invited an uncomment-in-repo — the explain pointer replaces it.
	if strings.Contains(fenceBody, "outranks the project file") {
		t.Errorf("fence must not carry the long-form machine-wide essay phrasing (the diet moved it to `fab config explain`).\n--- fence ---\n%s", fenceBody)
	}
}

// TestRender_FenceAdvertisesConsolidateDetectors: the `consolidate.detectors` key
// added for /code-dedupe (260728-4v91, as /fab-dedupe) is an advertised C field, so an un-overridden
// project gets it scaffolded — fully commented — into the managed fence, and a
// project that HAS overridden it keeps its live block with no fence duplicate
// (presence=intent). Runs over the SHIPPED registry: this is the guard that the new
// row actually reaches every user's config.yaml on the next `fab config upgrade`.
func TestRender_FenceAdvertisesConsolidateDetectors(t *testing.T) {
	fields := fieldsForTest(t)

	// Un-overridden: scaffolded into the fence, commented.
	out, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)
	if !strings.Contains(fenceBody, "# consolidate.detectors") {
		t.Errorf("fence must advertise the un-overridden consolidate.detectors field.\n--- fence ---\n%s", fenceBody)
	}
	if strings.Contains(fenceBody, "\nconsolidate:") {
		t.Error("fence must not carry a LIVE consolidate: parent key (must be commented)")
	}

	// Overridden: the live block survives verbatim and is NOT re-advertised.
	src := "consolidate:\n    detectors:\n        - jscpd {paths}\n"
	out2, _ := render(src, fields, ProjectTarget(""), "2.15.0")
	if !strings.Contains(out2, "- jscpd {paths}") {
		t.Errorf("a live consolidate override must be preserved verbatim.\n--- got ---\n%s", out2)
	}
	_, fenceBody2, _ := sliceFence(t, out2)
	if strings.Contains(fenceBody2, "consolidate") {
		t.Errorf("fence must omit the already-overridden consolidate field.\n--- fence ---\n%s", fenceBody2)
	}
}

// TestUpgrade_RefusesUnparseableOutput (SF-c): Upgrade validates the reconciled
// bytes and REFUSES to overwrite the file when the result would not parse as YAML,
// leaving the original file byte-untouched. Exercised via a live block whose value
// is broken YAML (an unclosed flow mapping) — it survives the line-splice but fails
// the parser, so a config that would brick every fab command is never written.
func TestUpgrade_RefusesUnparseableOutput(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	original := "project: {name: t\n" // unclosed flow map → the assembled doc won't parse
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Upgrade(ProjectTarget(path), "2.15.0")
	if err == nil {
		t.Fatal("Upgrade must REFUSE to write output that does not parse as YAML")
	}
	if !strings.Contains(err.Error(), "does not parse") {
		t.Errorf("refusal error should explain the parse failure, got: %v", err)
	}
	after, _ := os.ReadFile(path)
	if string(after) != original {
		t.Errorf("a refused upgrade must leave the original file untouched.\n--- original ---\n%s\n--- after ---\n%s", original, string(after))
	}
}

// TestCheck_StaleStampReportsDriftWithoutWrite: a file whose fence was stamped by
// an older kit version is DRIFT — Check reports Changed=true (a real run would
// restamp the fence) and leaves the file byte-identical on disk.
func TestCheck_StaleStampReportsDriftWithoutWrite(t *testing.T) {
	fields := fieldsForTest(t)
	stale, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.14.0")
	path := writeMutationFixture(t, stale)

	res, err := Check(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !res.Changed {
		t.Error("a stale fence kit-version stamp is drift — Check must report Changed")
	}
	if got := readMutationFixture(t, path); got != stale {
		t.Errorf("Check must never write the file.\n--- before ---\n%s\n--- after ---\n%s", stale, got)
	}
}

// TestCheck_UnparkedUnknownKeyReportsDriftWithoutWrite: a live key absent from
// the registry is drift (a real run would park it below the fence); the report
// names the key that would be parked and the file is left untouched.
func TestCheck_UnparkedUnknownKeyReportsDriftWithoutWrite(t *testing.T) {
	const src = "project:\n    name: t\n\nlegacy_mode: true\n"
	path := writeMutationFixture(t, src)

	res, err := Check(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !res.Changed {
		t.Error("an unparked unknown key is drift — Check must report Changed")
	}
	if !strings.Contains(strings.Join(res.Report, "\n"), "legacy_mode") {
		t.Errorf("the report must name the key a run would park, got %v", res.Report)
	}
	if got := readMutationFixture(t, path); got != src {
		t.Errorf("Check must never write the file.\n--- before ---\n%s\n--- after ---\n%s", src, got)
	}
}

// TestCheck_MissingFenceReportsDriftWithoutWrite: a legacy file with no managed
// fence is drift (a real run would append one); the file is left untouched.
func TestCheck_MissingFenceReportsDriftWithoutWrite(t *testing.T) {
	const legacy = "project:\n    name: t\n    description: d\n"
	path := writeMutationFixture(t, legacy)

	res, err := Check(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !res.Changed {
		t.Error("a file with no managed fence is drift — Check must report Changed")
	}
	if got := readMutationFixture(t, path); got != legacy {
		t.Errorf("Check must never write the file.\n--- before ---\n%s\n--- after ---\n%s", legacy, got)
	}
}

// TestCheck_MissingFileReportsDriftAndCreatesNothing: a missing config.yaml is
// drift (a real run would create a fresh fence-only file); Check creates nothing.
func TestCheck_MissingFileReportsDriftAndCreatesNothing(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")

	res, err := Check(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if !res.Changed {
		t.Error("a missing config.yaml is drift — a real run would create it, so Check must report Changed")
	}
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Errorf("Check must not create the file, stat: %v", statErr)
	}
}

// TestCheck_CleanFileReportsNoDrift: a file already reconciled at this kit
// version is clean — Changed=false and the file stays byte-identical.
func TestCheck_CleanFileReportsNoDrift(t *testing.T) {
	fields := fieldsForTest(t)
	clean, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.15.0")
	path := writeMutationFixture(t, clean)

	res, err := Check(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	if res.Changed {
		t.Errorf("an already-reconciled file is clean — Check must not report drift, report: %v", res.Report)
	}
	if got := readMutationFixture(t, path); got != clean {
		t.Errorf("Check must never write the file.\n--- before ---\n%s\n--- after ---\n%s", clean, got)
	}
}

// TestCheck_AgreesWithUpgrade is the compute-without-write contract: the drift
// probe shares Upgrade's compute path, so it can never disagree with a real run
// — on a drifted fixture Check and Upgrade report the same verdict AND the same
// report lines, and once Upgrade has applied, Check is clean.
func TestCheck_AgreesWithUpgrade(t *testing.T) {
	const src = "project:\n    name: t\n\nlegacy_mode: true\n"
	path := writeMutationFixture(t, src)

	checkRes, err := Check(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	upRes, err := Upgrade(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	if checkRes.Changed != upRes.Changed {
		t.Errorf("Check and Upgrade disagree on drift: check=%v upgrade=%v", checkRes.Changed, upRes.Changed)
	}
	if got, want := strings.Join(checkRes.Report, "\n"), strings.Join(upRes.Report, "\n"); got != want {
		t.Errorf("Check and Upgrade disagree on the report.\n--- check ---\n%s\n--- upgrade ---\n%s", got, want)
	}

	after, err := Check(ProjectTarget(path), "2.15.0")
	if err != nil {
		t.Fatalf("Check after Upgrade: %v", err)
	}
	if after.Changed {
		t.Errorf("Check must be clean right after Upgrade applied, report: %v", after.Report)
	}
}

// TestRender_FenceAdvertisesDispatchMode: the `dispatch.mode` preference is an
// advertised C field, so an un-overridden project gets
// it scaffolded — fully commented — into the managed fence, and a project that HAS
// set it keeps its live block with no fence duplicate (presence=intent). Runs over
// the SHIPPED registry: this is the guard that the new row actually reaches every
// user's config.yaml on the next `fab config upgrade`.
func TestRender_FenceAdvertisesDispatchMode(t *testing.T) {
	fields := fieldsForTest(t)

	// Un-overridden: scaffolded into the fence, commented.
	out, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)
	if !strings.Contains(fenceBody, "# dispatch.mode") {
		t.Errorf("fence must advertise the un-overridden dispatch.mode field.\n--- fence ---\n%s", fenceBody)
	}
	if strings.Contains(fenceBody, "\ndispatch:") {
		t.Error("fence must not carry a LIVE dispatch: parent key (must be commented)")
	}

	// Overridden: the live block survives verbatim and is NOT re-advertised.
	out2, _ := render("dispatch:\n    mode: pane\n", fields, ProjectTarget(""), "2.15.0")
	if !strings.Contains(out2, "mode: pane") {
		t.Errorf("a live dispatch override must be preserved verbatim.\n--- got ---\n%s", out2)
	}
	_, fenceBody2, _ := sliceFence(t, out2)
	if strings.Contains(fenceBody2, "dispatch.mode") {
		t.Errorf("fence must omit the already-overridden dispatch field.\n--- fence ---\n%s", fenceBody2)
	}
}

// TestRender_FenceAdvertisesDispatchColumnWidth: `dispatch.column_width` reaches the
// fence through the SHARED `dispatch:` segment rather than a segment of its own —
// `dispatch` is one YAML block, so two separately-uncommentable `# dispatch:`
// parents would collide into a duplicate key. The guard therefore checks the two
// properties that shape depends on: the width's scaffold line is present (an
// empty-Segment row is otherwise skipped outright by renderFence), and the fence
// carries exactly ONE commented dispatch parent. Runs over the SHIPPED registry,
// so it is the guard that the new key reaches every user's config.yaml on the next
// `fab config upgrade`.
func TestRender_FenceAdvertisesDispatchColumnWidth(t *testing.T) {
	fields := fieldsForTest(t)

	out, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)
	if !strings.Contains(fenceBody, "# dispatch.mode / dispatch.column_width / dispatch.reap_done") {
		t.Errorf("fence must advertise the un-overridden dispatch.column_width field (named in the shared dispatch header).\n--- fence ---\n%s", fenceBody)
	}
	wantScaffold := "#   column_width: " + strconv.Itoa(config.DefaultDispatchColumnWidth)
	if !strings.Contains(fenceBody, wantScaffold) {
		t.Errorf("fence must scaffold %q.\n--- fence ---\n%s", wantScaffold, fenceBody)
	}
	if n := strings.Count(fenceBody, "# dispatch:"); n != 1 {
		t.Errorf("fence carries %d `# dispatch:` parents, want exactly 1 (both dispatch keys share one block)", n)
	}

	// Overridden: the whole block is live above the fence, so neither dispatch key
	// is re-advertised (override detection is top-level-key scoped).
	out2, _ := render("dispatch:\n    column_width: 20\n", fields, ProjectTarget(""), "2.15.0")
	if !strings.Contains(out2, "column_width: 20") {
		t.Errorf("a live dispatch override must be preserved verbatim.\n--- got ---\n%s", out2)
	}
	_, fenceBody2, _ := sliceFence(t, out2)
	if strings.Contains(fenceBody2, "dispatch.column_width") {
		t.Errorf("fence must omit the already-overridden dispatch field.\n--- fence ---\n%s", fenceBody2)
	}
}

// TestRender_FenceAdvertisesDispatchReapDone: `dispatch.reap_done` is the third key
// under the shared `dispatch:` segment, so it reaches the fence the same way
// `column_width` does — no Segment of its own, one commented parent for all three.
// The guard checks the properties that shape depends on (the scaffold line is
// present, and the parent count is STILL exactly one now that a third key shares
// it), plus the override suppression. Runs over the SHIPPED registry, so it is the
// guard that the new key reaches every user's config.yaml on the next
// `fab config upgrade`.
func TestRender_FenceAdvertisesDispatchReapDone(t *testing.T) {
	fields := fieldsForTest(t)

	out, _ := render("project:\n    name: t\n", fields, ProjectTarget(""), "2.15.0")
	_, fenceBody, _ := sliceFence(t, out)
	if !strings.Contains(fenceBody, "# dispatch.mode / dispatch.column_width / dispatch.reap_done") {
		t.Errorf("fence must advertise the un-overridden dispatch.reap_done field (named in the shared dispatch header).\n--- fence ---\n%s", fenceBody)
	}
	wantScaffold := "#   reap_done: " + strconv.FormatBool(config.DefaultDispatchReapDone)
	if !strings.Contains(fenceBody, wantScaffold) {
		t.Errorf("fence must scaffold %q.\n--- fence ---\n%s", wantScaffold, fenceBody)
	}
	if n := strings.Count(fenceBody, "# dispatch:"); n != 1 {
		t.Errorf("fence carries %d `# dispatch:` parents, want exactly 1 (all three dispatch keys share one block)", n)
	}

	// Overridden: the whole block is live above the fence, so no dispatch key is
	// re-advertised (override detection is top-level-key scoped).
	out2, _ := render("dispatch:\n    reap_done: false\n", fields, ProjectTarget(""), "2.15.0")
	if !strings.Contains(out2, "reap_done: false") {
		t.Errorf("a live dispatch override must be preserved verbatim.\n--- got ---\n%s", out2)
	}
	_, fenceBody2, _ := sliceFence(t, out2)
	if strings.Contains(fenceBody2, "dispatch.reap_done") {
		t.Errorf("fence must omit the already-overridden dispatch field.\n--- fence ---\n%s", fenceBody2)
	}
}

// The mutation regressions below are deliberately written before the Set/Unset
// implementation. They preserve the six silent-success failures discovered on
// the discarded k0v3 branch as day-one contracts for the fresh writer.

func TestMutationRegression_BlockFormLeafKeepsSiblings(t *testing.T) {
	path := writeMutationFixture(t, `agent:
    profiles:
        review:
            provider: claude
            model: old-model
            effort: xhigh
`)

	if _, err := Set(path, "agent.profiles.review.model", "new-model", "test"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := readMutationFixture(t, path)
	for _, want := range []string{
		"provider: claude",
		"model: new-model",
		"effort: xhigh",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("setting one block-form leaf dropped sibling/content %q\n--- got ---\n%s", want, got)
		}
	}
	if strings.Contains(got, "model: old-model") {
		t.Errorf("old leaf value survived the replacement\n--- got ---\n%s", got)
	}
}

func TestMutationRegression_MaterializationUsesOneRegistryRenderer(t *testing.T) {
	path := writeMutationFixture(t, "")
	match, ok, err := configref.ResolveKey("dispatch.column_width")
	if err != nil || !ok {
		t.Fatalf("ResolveKey: ok=%v err=%v", ok, err)
	}

	if _, err := Set(path, "dispatch.column_width", "42", "test"); err != nil {
		t.Fatalf("Set: %v", err)
	}
	got := readMutationFixture(t, path)
	live := strings.TrimRight(strings.SplitN(got, "# >>> fab reference", 2)[0], "\n")
	// Derive the expected live bytes from the fence renderer's own output at test
	// time. If dispatchSegment changes its indentation or structure, a second
	// literal materializer in Set will drift and this assertion will fail.
	want := materializationFromRenderedFence(t, match.Owner, "dispatch.column_width", "42")
	if live != want {
		t.Fatalf("set materialization did not derive from the fence's registry renderer\n--- got ---\n%s\n--- want ---\n%s", live, want)
	}
}

func materializationFromRenderedFence(t *testing.T, owner configref.Field, key, value string) string {
	t.Helper()
	fence := renderFence(ProjectTarget(""), []configref.Field{owner}, map[string]bool{}, "test")
	_, fenceBody, _ := sliceFence(t, fence)
	virtual := strings.Split(fenceBody, "\n")
	for i, line := range virtual {
		virtual[i] = strings.TrimPrefix(line, "# ")
	}

	parts := strings.Split(key, ".")
	var pathLines []string
	for _, entry := range scanLiveKeyLines(virtual) {
		if pathHasPrefix(parts, entry.path) {
			line := virtual[entry.line][:entry.colon+1]
			if pathEqual(entry.path, parts) {
				line += " " + value
			}
			pathLines = append(pathLines, line)
		}
	}
	if len(pathLines) != len(parts) {
		t.Fatalf("rendered fence did not contain materialization path %q\n--- fence ---\n%s", key, fence)
	}
	return strings.Join(pathLines, "\n")
}

func TestMutationRegression_MaterializationKeepsEveryAncestor(t *testing.T) {
	original, _ := render("", fieldsForTest(t), ProjectTarget(""), "test")
	path := writeMutationFixture(t, original)
	if _, err := Set(path, "stage_hooks.apply.pre", "./scripts/check.sh", "test"); err != nil {
		t.Fatalf("Set stage hook: %v", err)
	}

	got := readMutationFixture(t, path)
	live := strings.TrimRight(strings.SplitN(got, "# >>> fab reference", 2)[0], "\n")
	const want = "stage_hooks:\n  apply:\n    pre: ./scripts/check.sh"
	if live != want {
		t.Fatalf("set did not materialize the full registry ancestry\n--- got ---\n%s\n--- want ---\n%s", live, want)
	}
	cfg, err := config.LoadPath(path)
	if err != nil {
		t.Fatalf("LoadPath after Set: %v", err)
	}
	if hook := cfg.GetStageHook("apply"); hook.Pre != "./scripts/check.sh" {
		t.Fatalf("effective apply hook = %+v, want pre command", hook)
	}

	if _, err := Unset(path, "stage_hooks.apply.pre", "test"); err != nil {
		t.Fatalf("Unset stage hook: %v", err)
	}
	if got := readMutationFixture(t, path); got != original {
		t.Fatalf("set/unset did not restore the original bytes\n--- got ---\n%s\n--- want ---\n%s", got, original)
	}
}

func TestMutationRegression_KeyAxisTyposRefuseWithoutWrite(t *testing.T) {
	for _, key := range []string{
		"agent.profile.review.model",
		"dispatch.colum_width",
	} {
		t.Run(key, func(t *testing.T) {
			const original = "agent:\n    workers: claude\n"
			path := writeMutationFixture(t, original)
			if _, err := Set(path, key, "codex", "test"); err == nil {
				t.Fatalf("Set(%q) succeeded, want unknown-key error", key)
			}
			if got := readMutationFixture(t, path); got != original {
				t.Fatalf("rejected key changed the file\n--- got ---\n%s\n--- want ---\n%s", got, original)
			}
		})
	}
}

func TestMutationRegression_ScalarOnlySetRefusesWithoutWrite(t *testing.T) {
	const original = "agent:\n    workers: claude\n"
	for _, tc := range []struct {
		name, key, value, want string
	}{
		{"structural map key", "agent", "{workers: {bad: kind}}", "scalar leaf"},
		{"structural dispatch key", "dispatch", "{colum_width: 42}", "scalar leaf"},
		{"nested map key", "agent.profiles.review", "{model: pinned}", "scalar leaf"},
		{"sequence leaf", "source_paths", "[src/]", "scalar leaf"},
		{"mapping value", "agent.workers", "{a: b}", "single-line YAML scalar"},
		{"sequence value", "agent.workers", "[codex]", "single-line YAML scalar"},
		{"null value", "agent.workers", "null", "single-line YAML scalar"},
		{"multiline value", "agent.workers", "[codex,\nnext]", "single-line YAML scalar"},
		// configvalue.Parse accepts block collections (environment overrides need
		// them); set refuses them on its own — one-line and multi-line alike.
		{"block mapping value", "agent.workers", "a: b", "single-line YAML scalar"},
		{"multiline block mapping value", "agent.workers", "custom:\n  interactive_command: tool", "single-line YAML scalar"},
		{"block sequence value", "agent.workers", "- codex\n- next", "single-line YAML scalar"},
		{"block scalar indicator", "agent.workers", "|", "single-line YAML scalar"},
		{"escaped multiline scalar", "agent.workers", `"codex\nnext"`, "single-line YAML scalar"},
		{"comment-bearing value", "agent.workers", "codex # note", "must not contain a YAML comment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := writeMutationFixture(t, original)
			_, err := Set(path, tc.key, tc.value, "test")
			if err == nil || !strings.Contains(err.Error(), tc.want) || !strings.Contains(err.Error(), "edit") {
				t.Fatalf("Set(%q, %q) error = %v, want %q plus manual-edit guidance", tc.key, tc.value, err, tc.want)
			}
			if got := readMutationFixture(t, path); got != original {
				t.Fatalf("rejected set changed the file\n--- got ---\n%s\n--- want ---\n%s", got, original)
			}
		})
	}
}

func TestMutationRegression_QuoteHolesRefuseWithoutWrite(t *testing.T) {
	t.Run("quoted key", func(t *testing.T) {
		const original = "agent:\n    workers: claude\n"
		path := writeMutationFixture(t, original)
		if _, err := Set(path, `"agent".workers`, "codex", "test"); err == nil {
			t.Fatal("Set accepted a quoted key")
		}
		if got := readMutationFixture(t, path); got != original {
			t.Fatalf("rejected quoted key changed the file\n--- got ---\n%s\n--- want ---\n%s", got, original)
		}
	})

	t.Run("unterminated quoted live value is repairable", func(t *testing.T) {
		const original = `# keep the file note
agent:
    workers: "claude # value data, not a comment
    session: codex # keep the sibling note
`
		path := writeMutationFixture(t, original)
		if _, err := Unset(path, "agent.workers", "test"); err != nil {
			t.Fatalf("Unset malformed quoted value: %v", err)
		}

		got := readMutationFixture(t, path)
		live := strings.TrimRight(strings.SplitN(got, "# >>> fab reference", 2)[0], "\n")
		const wantLive = `# keep the file note
agent:
    session: codex # keep the sibling note`
		if live != wantLive {
			t.Fatalf("unset did not remove the unterminated quoted value byte-exactly\n--- got ---\n%s\n--- want ---\n%s", live, wantLive)
		}
		if strings.Contains(got, "value data, not a comment") {
			t.Fatalf("leading-quote scanner promoted malformed value bytes to a comment\n--- got ---\n%s", got)
		}
	})
}

func TestMutationRegression_CommentsSurviveSetUnsetAndRepeatIsFrozen(t *testing.T) {
	const original = `# user: keep the file-level note
agent:
    profiles:
        review:
            # user: keep the interior note
            model: old-model # user: keep the inline note
            effort: xhigh # user: keep the sibling note
`
	path := writeMutationFixture(t, original)

	if _, err := Set(path, "agent.profiles.review.model", "new-model", "test"); err != nil {
		t.Fatalf("first Set: %v", err)
	}
	first := readMutationFixture(t, path)
	for _, comment := range []string{
		"# user: keep the file-level note",
		"# user: keep the interior note",
		"# user: keep the inline note",
		"# user: keep the sibling note",
	} {
		if !strings.Contains(first, comment) {
			t.Errorf("Set lost comment %q\n--- got ---\n%s", comment, first)
		}
	}

	if _, err := Set(path, "agent.profiles.review.model", "new-model", "test"); err != nil {
		t.Fatalf("second Set: %v", err)
	}
	if second := readMutationFixture(t, path); second != first {
		t.Fatalf("setting the same value twice is not byte-identical\n--- first ---\n%s\n--- second ---\n%s", first, second)
	}

	if _, err := Unset(path, "agent.profiles.review.model", "test"); err != nil {
		t.Fatalf("Unset: %v", err)
	}
	unset := readMutationFixture(t, path)
	for _, comment := range []string{
		"# user: keep the file-level note",
		"# user: keep the interior note",
		"# user: keep the inline note",
		"# user: keep the sibling note",
	} {
		if !strings.Contains(unset, comment) {
			t.Errorf("Unset lost comment %q\n--- got ---\n%s", comment, unset)
		}
	}
	if !strings.Contains(unset, "effort: xhigh") {
		t.Fatalf("Unset dropped a sibling leaf\n--- got ---\n%s", unset)
	}
}

// TestSystemMutation_ScopeScaffoldAndManagedFence: `set --system` on a missing
// file creates a scaffold whose system header lives INSIDE the fence (the
// above-fence region starts with the live override), with the managed fence
// and the system-upgrade command spelling intact.
func TestSystemMutation_ScopeScaffoldAndManagedFence(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".fab-kit", "config.yaml")
	if _, err := SetSystem(path, "providers", "{custom: {interactive_command: tool}}", "test"); err == nil || !strings.Contains(err.Error(), "scalar leaf") {
		t.Fatalf("SetSystem collection key error = %v, want scalar-leaf refusal", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("rejected collection mutation created a file: %v", err)
	}
	if _, err := SetSystem(path, "source_paths", "[src/]", "test"); err == nil {
		t.Fatal("SetSystem accepted a project-scoped key")
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("rejected system mutation created a file: %v", err)
	}

	if _, err := SetSystem(path, "agent.workers", "codex", "test"); err != nil {
		t.Fatalf("SetSystem: %v", err)
	}
	got := readMutationFixture(t, path)
	if !strings.HasPrefix(got, "agent:") {
		t.Fatalf("system mutation must open with the live override (header is fence-owned)\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, "\n  workers: codex") {
		t.Fatalf("missing system override\n--- got ---\n%s", got)
	}
	inside := strings.SplitN(got, "# <<< end fab reference", 2)[0]
	if !strings.Contains(inside, SystemScaffoldHeader) {
		t.Fatalf("system scaffold header missing inside the fence\n--- got ---\n%s", got)
	}
	if strings.Count(got, ">>> fab reference") != 1 || strings.Count(got, "<<< end fab reference") != 1 {
		t.Fatalf("system mutation must retain exactly one managed fence\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, "REGENERATED by `fab config upgrade --system`") {
		t.Fatalf("system fence preamble does not name the system upgrade command\n--- got ---\n%s", got)
	}
	if strings.Contains(got, "stage_hooks") {
		t.Fatalf("system fence contains a project-scoped field\n--- got ---\n%s", got)
	}
	check, err := Check(SystemTarget(path), "test")
	if err != nil {
		t.Fatalf("Check after SetSystem: %v", err)
	}
	if check.Changed {
		t.Fatalf("SetSystem left the managed system file drifted\n--- got ---\n%s", got)
	}
}

func TestSystemMutation_UnsetKnownAbsentIsByteStable(t *testing.T) {
	const original = "# owned system comments\nagent:\n    workers: codex # keep me\n"
	path := writeMutationFixture(t, original)
	if _, err := UnsetSystem(path, "agent.session", "test"); err != nil {
		t.Fatalf("UnsetSystem absent: %v", err)
	}
	if got := readMutationFixture(t, path); got != original {
		t.Fatalf("absent unset changed bytes\n--- got ---\n%s", got)
	}
	if _, err := UnsetSystem(path, "agent.workers", "test"); err != nil {
		t.Fatalf("UnsetSystem present: %v", err)
	}
	got := readMutationFixture(t, path)
	if !strings.Contains(got, "# keep me") || strings.Contains(strings.SplitN(got, "# >>> fab reference", 2)[0], "workers:") {
		t.Fatalf("system unset failed comment-preserving removal\n--- got ---\n%s", got)
	}
	if strings.Count(got, ">>> fab reference") != 1 || strings.Count(got, "<<< end fab reference") != 1 {
		t.Fatalf("system unset must leave exactly one managed fence\n--- got ---\n%s", got)
	}
	check, err := Check(SystemTarget(path), "test")
	if err != nil || check.Changed {
		t.Fatalf("system unset left the target drifted: %+v, %v", check, err)
	}
}

func legacySystemScaffoldForTest(t *testing.T) string {
	t.Helper()
	var segments []string
	for _, field := range fieldsForTest(t) {
		if SystemField(field) && field.ShortSegment != "" {
			segments = append(segments, CommentOutSegment(field.ShortSegment))
		}
	}
	return legacySystemScaffoldHeader + "\n\n" + strings.Join(segments, "\n\n") + "\n"
}

// legacyV2198ToV2220ProvidersAdvert is byte-for-byte output from the released
// v2.19.8, v2.20.7, v2.20.8, and v2.22.0 registries. Keep it independent of
// configref.Fields: the regression is specifically that a current-registry
// fixture changes in lockstep with the recognizer and cannot exercise history.
const legacyV2198ToV2220ProvidersAdvert = `# providers — named agent capability grammars (interactive/headless/native
# commands) plus per-role {model, effort} fills. fab-kit ships claude (the
# default), codex, agy and kimi; naming a built-in on a depth knob needs no
# entry here. [both]
# Settable machine-wide: fab config set --system providers.<name>.<field> <value>
# Full prose: fab config explain providers
# providers:
#   claude:
#     native: true
#     interactive_command: 'claude --dangerously-skip-permissions -n "$(basename "$(pwd)")" --model {model} --effort {effort}'
#     headless_command: 'claude -p --dangerously-skip-permissions --model {model} --effort {effort}'
#   codex:
#     interactive_command: 'codex --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
#     headless_command: 'codex exec --dangerously-bypass-approvals-and-sandbox -m {model} -c model_reasoning_effort={effort}'
#     profiles:
#       default: { model: gpt-5.6-sol, effort: high }
#       doing: { effort: xhigh }
#       fast: { model: gpt-5.6-luna, effort: low }
#       operator: { model: gpt-5.6-luna, effort: medium }
#       review: { effort: xhigh }
#   agy:
#     interactive_command: 'agy --dangerously-skip-permissions --model {model}'   # trust wall handled by the readiness gate; no {effort} flag
#     headless_command: 'sh -c ''agy --dangerously-skip-permissions --print-timeout 120m --model {model} -p "$(cat)"'''   # no {effort} flag; nested shell so $(cat) reads the piped prompt
#     profiles:
#       default: { model: gemini-3.1-pro-high }
#       fast: { model: gemini-3.6-flash-low }
#   kimi:
#     interactive_command: 'kimi --auto -m {model}'   # --auto is the full-auto flag its headless form rejects
#     headless_command: 'sh -c ''kimi -m {model} -p "$(cat)"'''   # no fills shipped, so the empty {model} drops -m`

const legacyV2198ToV2220SystemScaffold = legacySystemScaffoldHeader + "\n\n" + legacyV2198ToV2220ProvidersAdvert + "\n"

var legacyScaffoldVersions = []string{"2.19.8", "2.20.7", "2.20.8", "2.22.0"}

func TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer(t *testing.T) {
	paragraphs := []string{SystemScaffoldHeader}
	for _, field := range fieldsForTest(t) {
		if SystemField(field) && field.ShortSegment != "" {
			paragraphs = append(paragraphs, CommentOutSegment(field.ShortSegment))
		}
	}
	for _, paragraph := range paragraphs {
		digest := systemParagraphDigest(paragraph)
		if _, ok := knownGeneratedSystemParagraphDigests[digest]; !ok {
			firstLine := strings.SplitN(paragraph, "\n", 2)[0]
			t.Errorf("current generated paragraph %q is missing digest %s from the historical catalog", firstLine, digest)
		}
	}
}

func TestSystemUpgrade_AdoptsReleasedHistoricalScaffoldsAfterRegistryDrift(t *testing.T) {
	for _, version := range legacyScaffoldVersions {
		t.Run(version, func(t *testing.T) {
			path := writeMutationFixture(t, legacyV2198ToV2220SystemScaffold)

			if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
				t.Fatalf("Upgrade historical system target: %v", err)
			}
			got := readMutationFixture(t, path)

			if strings.Count(got, ">>> fab reference") != 1 || strings.Count(got, "<<< end fab reference") != 1 {
				t.Fatalf("historical adoption must emit exactly one fence\n--- got ---\n%s", got)
			}
			for _, field := range fieldsForTest(t) {
				if !SystemField(field) || field.ShortSegment == "" {
					continue
				}
				marker := "# Full prose: fab config explain " + field.Key
				if count := strings.Count(got, marker); count != 1 {
					t.Errorf("historical adoption emitted %d adverts for %q, want 1\n--- got ---\n%s", count, field.Key, got)
				}
			}
			if strings.Contains(got, "claude --dangerously-skip-permissions") {
				t.Fatalf("historical claude command survived above the regenerated fence\n--- got ---\n%s", got)
			}
			if count := strings.Count(got, "#     interactive_command:"); count != 4 {
				t.Fatalf("historical provider commands were duplicated: got %d interactive commands, want 4\n--- got ---\n%s", count, got)
			}
		})
	}
}

func TestSystemUpgrade_LineCompleteAccountingPreservesMixedAndEditedParagraphs(t *testing.T) {
	t.Run("note appended without blank separator", func(t *testing.T) {
		const note = "# user: keep this provider note"
		original := legacySystemScaffoldHeader + "\n\n" + legacyV2198ToV2220ProvidersAdvert + "\n" + note + "\n"
		path := writeMutationFixture(t, original)

		if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
			t.Fatalf("Upgrade mixed historical paragraph: %v", err)
		}
		got := readMutationFixture(t, path)
		preserved := legacyV2198ToV2220ProvidersAdvert + "\n" + note
		if !strings.Contains(got, preserved) {
			t.Fatalf("mixed paragraph was not preserved whole\n--- got ---\n%s", got)
		}
		if count := strings.Count(got, "# Full prose: fab config explain providers"); count != 2 {
			t.Fatalf("mixed paragraph should remain beside the regenerated advert: got %d copies, want 2\n--- got ---\n%s", count, got)
		}
	})

	t.Run("user-edited generated copy", func(t *testing.T) {
		const oldValue = "#       default: { model: gpt-5.6-sol, effort: high }"
		const editedValue = "#       default: { model: my-local-model, effort: high }"
		edited := strings.Replace(legacyV2198ToV2220ProvidersAdvert, oldValue, editedValue, 1)
		path := writeMutationFixture(t, legacySystemScaffoldHeader+"\n\n"+edited+"\n")

		if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
			t.Fatalf("Upgrade edited historical paragraph: %v", err)
		}
		got := readMutationFixture(t, path)
		if !strings.Contains(got, edited) || !strings.Contains(got, editedValue) {
			t.Fatalf("edited advert was not preserved as user content\n--- got ---\n%s", got)
		}
		if count := strings.Count(got, "# Full prose: fab config explain providers"); count != 2 {
			t.Fatalf("edited paragraph should remain beside the regenerated advert: got %d copies, want 2\n--- got ---\n%s", count, got)
		}
	})

	t.Run("note appended to historical header", func(t *testing.T) {
		const note = "# user: keep this machine note"
		path := writeMutationFixture(t, legacySystemScaffoldHeader+"\n"+note+"\n")

		if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
			t.Fatalf("Upgrade mixed historical header: %v", err)
		}
		got := readMutationFixture(t, path)
		if !strings.Contains(got, legacySystemScaffoldHeader+"\n"+note) {
			t.Fatalf("mixed header paragraph was not preserved whole\n--- got ---\n%s", got)
		}
	})
}

func TestSystemUpgrade_LineCompleteAccountingAppliesAfterAdoption(t *testing.T) {
	path := writeMutationFixture(t, "")
	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
		t.Fatalf("initial Upgrade: %v", err)
	}
	current := readMutationFixture(t, path)
	const editedValue = "#       default: { model: later-user-edit, effort: high }"
	edited := strings.Replace(legacyV2198ToV2220ProvidersAdvert,
		"#       default: { model: gpt-5.6-sol, effort: high }", editedValue, 1)
	// The header is fence-owned, so the file opens with the fence anchor; attach
	// the edited paragraph to the anchor line (no blank separator keeps it one
	// paragraph, mirroring the original above-the-header form).
	withEditedCopy := strings.Replace(current, "# >>> fab reference", edited+"\n# >>> fab reference", 1)
	if err := os.WriteFile(path, []byte(withEditedCopy), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
		t.Fatalf("later Upgrade: %v", err)
	}
	got := readMutationFixture(t, path)
	if !strings.Contains(got, edited) || !strings.Contains(got, editedValue) {
		t.Fatalf("later upgrade discarded an edited advert above the fence\n--- got ---\n%s", got)
	}
}

func TestSystemMutation_LineCompleteAccountingPreservesWholeParagraph(t *testing.T) {
	const note = "# user: mutation must keep this attached note"
	mixed := legacySystemScaffoldHeader + "\n\n" + legacyV2198ToV2220ProvidersAdvert + "\n" + note
	tests := []struct {
		name     string
		original string
		mutate   func(string) error
	}{
		{
			name:     "set system",
			original: mixed + "\n",
			mutate: func(path string) error {
				_, err := SetSystem(path, "agent.workers", "codex", "2.30.0")
				return err
			},
		},
		{
			name:     "unset system",
			original: mixed + "\n\nagent:\n  workers: codex\n",
			mutate: func(path string) error {
				_, err := UnsetSystem(path, "agent.workers", "2.30.0")
				return err
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			path := writeMutationFixture(t, tc.original)
			if err := tc.mutate(path); err != nil {
				t.Fatalf("mutation: %v", err)
			}
			got := readMutationFixture(t, path)
			if !strings.Contains(got, legacyV2198ToV2220ProvidersAdvert+"\n"+note) {
				t.Fatalf("mutation did not preserve the mixed paragraph whole\n--- got ---\n%s", got)
			}
			if count := strings.Count(got, "# Full prose: fab config explain providers"); count != 2 {
				t.Fatalf("mixed paragraph should remain beside the regenerated advert: got %d copies, want 2\n--- got ---\n%s", count, got)
			}
		})
	}
}

func TestSystemUpgrade_AdoptsLegacyScaffoldPreservingLiveConfigAndComments(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".fab-kit", "config.yaml")
	legacy := legacySystemScaffoldForTest(t) + `
# work laptop only
agent:
  workers: codex # pinned for reviews
`
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(legacy), 0o644); err != nil {
		t.Fatal(err)
	}
	var before map[string]any
	if err := yaml.Unmarshal([]byte(legacy), &before); err != nil {
		t.Fatalf("legacy fixture does not parse: %v", err)
	}

	result, err := Upgrade(SystemTarget(path), "2.30.0")
	if err != nil {
		t.Fatalf("Upgrade system target: %v", err)
	}
	if !result.Changed || !strings.Contains(strings.Join(result.Report, "\n"), "adopted existing unfenced system config") {
		t.Fatalf("legacy adoption result = %+v", result)
	}
	got := readMutationFixture(t, path)
	if strings.Count(got, ">>> fab reference") != 1 || strings.Count(got, "<<< end fab reference") != 1 {
		t.Fatalf("adoption must emit exactly one fence\n--- got ---\n%s", got)
	}
	for _, advert := range []string{"# dispatch:", "# providers:", "# autopilot:"} {
		if count := strings.Count(got, advert); count != 1 {
			t.Errorf("adoption emitted %d copies of %q, want 1\n--- got ---\n%s", count, advert, got)
		}
	}
	if !strings.Contains(got, "# work laptop only\nagent:\n  workers: codex # pinned for reviews") {
		t.Fatalf("adoption did not preserve the live block and attached comment verbatim\n--- got ---\n%s", got)
	}
	if strings.Index(got, "agent:\n  workers: codex") > strings.Index(got, "# >>> fab reference") {
		t.Fatalf("live system override was not hoisted above the fence\n--- got ---\n%s", got)
	}
	if strings.Contains(got, "stage_hooks") {
		t.Fatalf("system scaffold advertised a project-scoped field\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, "REGENERATED by `fab config upgrade --system`") {
		t.Fatalf("system fence preamble names the wrong command\n--- got ---\n%s", got)
	}
	var after map[string]any
	if err := yaml.Unmarshal([]byte(got), &after); err != nil {
		t.Fatalf("adopted file does not parse: %v", err)
	}
	if !reflect.DeepEqual(after, before) {
		t.Fatalf("adoption changed effective live YAML: before=%#v after=%#v", before, after)
	}

	second, err := Upgrade(SystemTarget(path), "2.30.0")
	if err != nil {
		t.Fatalf("second Upgrade: %v", err)
	}
	if second.Changed {
		t.Fatalf("second system upgrade must be a no-op: %+v", second)
	}
	if got2 := readMutationFixture(t, path); got2 != got {
		t.Fatal("second system upgrade was not byte-identical")
	}
}

func TestSystemUpgrade_CheckAndParkingAreByteStable(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".fab-kit", "config.yaml")
	const original = "dispatch:\n  mode: pane\n\nretired_preference:\n  nested: kept\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	check, err := Check(SystemTarget(path), "2.30.0")
	if err != nil {
		t.Fatalf("Check drifted system file: %v", err)
	}
	if !check.Changed {
		t.Fatal("unfenced system config must report drift")
	}
	if got := readMutationFixture(t, path); got != original {
		t.Fatal("system Check wrote the drifted file")
	}

	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
		t.Fatalf("Upgrade system file: %v", err)
	}
	first := readMutationFixture(t, path)
	if !strings.Contains(first, "#   retired_preference:") || !strings.Contains(first, "#     nested: kept") {
		t.Fatalf("removed system key was not parked with its value\n--- got ---\n%s", first)
	}
	if strings.Contains(strings.SplitN(first, "# >>> fab reference", 2)[0], "retired_preference:") {
		t.Fatalf("removed system key remained live above the fence\n--- got ---\n%s", first)
	}
	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
		t.Fatalf("second Upgrade: %v", err)
	}
	if second := readMutationFixture(t, path); second != first {
		t.Fatal("parked system block changed on the second upgrade")
	}
	clean, err := Check(SystemTarget(path), "2.30.0")
	if err != nil || clean.Changed {
		t.Fatalf("Check after Upgrade = %+v, %v; want clean", clean, err)
	}
}

// TestSystemUpgrade_ParksProjectScopedKeyUnderOutOfScopeHeader: a registry-known
// but scope:project key live in the machine-level file is NOT a removal — it must
// park under the distinct out-of-scope header (never "removed in …"), while a
// truly unknown key still parks under the removal header. The broadened
// parkedHeaderRe recognizes both header forms, so a second run neither re-parks
// nor churns the blocks.
func TestSystemUpgrade_ParksProjectScopedKeyUnderOutOfScopeHeader(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".fab-kit", "config.yaml")
	const original = "dispatch:\n  mode: pane\n\nstage_hooks:\n  apply:\n    post: make test\n\nretired_preference: 1\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Upgrade(SystemTarget(path), "2.30.0")
	if err != nil {
		t.Fatalf("Upgrade system file: %v", err)
	}
	got := readMutationFixture(t, path)
	if !strings.Contains(got, "# out of scope for this config layer — registry-known, unused at this layer (parked by fab config upgrade — delete when done):\n#   stage_hooks:\n#     apply:\n#       post: make test") {
		t.Errorf("project-scoped key must park under the out-of-scope header with its value\n--- got ---\n%s", got)
	}
	if strings.Count(got, "# removed in ") != 1 ||
		!strings.Contains(got, "# removed in an earlier release (parked by fab config upgrade — delete when done):\n#   retired_preference: 1") {
		t.Errorf("truly unknown key must park under the removal header\n--- got ---\n%s", got)
	}
	if strings.Contains(strings.SplitN(got, "# >>> fab reference", 2)[0], "stage_hooks:") {
		t.Errorf("out-of-scope key remained live above the fence\n--- got ---\n%s", got)
	}
	if report := strings.Join(result.Report, "\n"); !strings.Contains(report, `parked out-of-scope field "stage_hooks"`) {
		t.Errorf("report must note the out-of-scope parking, got %v", result.Report)
	}

	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
		t.Fatalf("second Upgrade: %v", err)
	}
	if second := readMutationFixture(t, path); second != got {
		t.Fatal("second upgrade was not byte-identical (parked out-of-scope block not recognized)")
	}
	clean, err := Check(SystemTarget(path), "2.30.0")
	if err != nil || clean.Changed {
		t.Fatalf("Check after Upgrade = %+v, %v; want clean", clean, err)
	}
}

func TestSystemUpgrade_RefusesMalformedLiveConfigWithoutWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), ".fab-kit", "config.yaml")
	const malformed = "agent: [\n"
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(malformed), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err == nil {
		t.Fatal("system Upgrade must reject malformed live YAML")
	}
	if got := readMutationFixture(t, path); got != malformed {
		t.Fatal("rejected system Upgrade changed the original bytes")
	}
}

func TestMutation_FlowUnsetPrunesEmptyAncestors(t *testing.T) {
	path := writeMutationFixture(t, "agent: {profiles: {review: {model: pinned}}}\n")
	if _, err := Unset(path, "agent.profiles.review.model", "test"); err != nil {
		t.Fatalf("Unset flow leaf: %v", err)
	}
	got := readMutationFixture(t, path)
	live := strings.SplitN(got, "# >>> fab reference", 2)[0]
	if strings.Contains(live, "review") || strings.Contains(live, "profiles") {
		t.Fatalf("flow unset left empty override ancestors\n--- got ---\n%s", got)
	}
}

func TestMutation_FlowMappingSplicesPreserveRawFormatting(t *testing.T) {
	const original = "agent: { profiles: { review: { model: old-model, effort: xhigh } }, session: claude } # keep flow style\n"
	path := writeMutationFixture(t, original)

	if _, err := Set(path, "agent.profiles.review.model", "new-model", "test"); err != nil {
		t.Fatalf("Set flow leaf: %v", err)
	}
	got := readMutationFixture(t, path)
	wantLive := "agent: { profiles: { review: { model: new-model, effort: xhigh } }, session: claude } # keep flow style"
	if live := strings.TrimRight(strings.SplitN(got, "# >>> fab reference", 2)[0], "\n"); live != wantLive {
		t.Fatalf("flow set normalized bytes outside the target value\n--- got ---\n%s\n--- want ---\n%s", live, wantLive)
	}

	if _, err := Unset(path, "agent.profiles.review.model", "test"); err != nil {
		t.Fatalf("Unset flow leaf: %v", err)
	}
	got = readMutationFixture(t, path)
	wantLive = "agent: { profiles: { review: { effort: xhigh } }, session: claude } # keep flow style"
	if live := strings.TrimRight(strings.SplitN(got, "# >>> fab reference", 2)[0], "\n"); live != wantLive {
		t.Fatalf("flow unset normalized bytes outside the removed pair\n--- got ---\n%s\n--- want ---\n%s", live, wantLive)
	}
}

func TestMutation_FlowMappingInsertionIsRaw(t *testing.T) {
	path := writeMutationFixture(t, "agent: { workers: claude }\n")
	if _, err := Set(path, "agent.session", "codex", "test"); err != nil {
		t.Fatalf("Set missing flow leaf: %v", err)
	}
	got := readMutationFixture(t, path)
	wantLive := "agent: { workers: claude, session: codex }"
	if live := strings.TrimRight(strings.SplitN(got, "# >>> fab reference", 2)[0], "\n"); live != wantLive {
		t.Fatalf("flow insertion did not preserve the existing raw mapping\n--- got ---\n%s\n--- want ---\n%s", live, wantLive)
	}
}

func TestMutation_BlockScalarBodiesAreReplacedAndRemoved(t *testing.T) {
	for _, indicator := range []string{"|", ">-"} {
		t.Run(indicator, func(t *testing.T) {
			original := "agent:\n" +
				"    workers: " + indicator + " # keep scalar note\n" +
				"        claude\n" +
				"        model: this is scalar content, not a key\n" +
				"    session: claude\n"

			setPath := writeMutationFixture(t, original)
			if _, err := Set(setPath, "agent.workers", "codex", "test"); err != nil {
				t.Fatalf("Set block scalar: %v", err)
			}
			setLive := strings.SplitN(readMutationFixture(t, setPath), "# >>> fab reference", 2)[0]
			if !strings.Contains(setLive, "workers: codex # keep scalar note") ||
				!strings.Contains(setLive, "session: claude") ||
				strings.Contains(setLive, "this is scalar content") || strings.Contains(setLive, "        claude") {
				t.Fatalf("set did not replace the complete block-scalar body\n%s", setLive)
			}

			unsetPath := writeMutationFixture(t, original)
			if _, err := Unset(unsetPath, "agent.workers", "test"); err != nil {
				t.Fatalf("Unset block scalar: %v", err)
			}
			unsetLive := strings.SplitN(readMutationFixture(t, unsetPath), "# >>> fab reference", 2)[0]
			if !strings.Contains(unsetLive, "# keep scalar note") ||
				!strings.Contains(unsetLive, "session: claude") ||
				strings.Contains(unsetLive, "workers:") ||
				strings.Contains(unsetLive, "this is scalar content") || strings.Contains(unsetLive, "        claude") {
				t.Fatalf("unset orphaned or retained block-scalar body bytes\n%s", unsetLive)
			}
		})
	}
}

func TestMutation_UnsetRepairsWrongKindLiveValue(t *testing.T) {
	path := writeMutationFixture(t, "dispatch:\n    column_width: not-an-int # preserve repair note\n")
	if _, err := Unset(path, "dispatch.column_width", "test"); err != nil {
		t.Fatalf("Unset wrong-kind live value: %v", err)
	}
	got := readMutationFixture(t, path)
	live := strings.SplitN(got, "# >>> fab reference", 2)[0]
	if strings.Contains(live, "column_width:") || !strings.Contains(live, "# preserve repair note") {
		t.Fatalf("unset did not repair the wrong-kind override while preserving its note\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, "#   column_width: "+strconv.Itoa(config.DefaultDispatchColumnWidth)) {
		t.Fatalf("unset did not re-advertise the inherited field in the fence\n--- got ---\n%s", got)
	}
}

func TestMutation_UnsetReadvertisesMissingSiblingField(t *testing.T) {
	path := writeMutationFixture(t, "agent:\n    session: codex\n    workers: claude\n")
	if _, err := Unset(path, "agent.workers", "test"); err != nil {
		t.Fatalf("Unset agent.workers: %v", err)
	}

	got := readMutationFixture(t, path)
	live := strings.SplitN(got, "# >>> fab reference", 2)[0]
	if !strings.Contains(live, "session: codex") || strings.Contains(live, "workers:") {
		t.Fatalf("unset did not keep the live sibling while removing workers\n--- got ---\n%s", got)
	}
	if !strings.Contains(got, "#   workers: claude") {
		t.Fatalf("unset did not re-advertise the inherited sibling field\n--- got ---\n%s", got)
	}
	fence := strings.SplitN(got, "# >>> fab reference", 2)[1]
	if strings.Contains(fence, "#   session:") {
		t.Fatalf("unset re-advertised the still-live sibling field\n--- got ---\n%s", got)
	}
}

func TestMutation_SetAdvertisesOnlyMissingSiblingField(t *testing.T) {
	path := writeMutationFixture(t, "")
	if _, err := Set(path, "agent.workers", "codex", "test"); err != nil {
		t.Fatalf("Set agent.workers: %v", err)
	}

	got := readMutationFixture(t, path)
	live := strings.SplitN(got, "# >>> fab reference", 2)[0]
	if !strings.Contains(live, "workers: codex") || strings.Contains(live, "session:") {
		t.Fatalf("set did not materialize only workers\n--- got ---\n%s", got)
	}
	fence := strings.SplitN(got, "# >>> fab reference", 2)[1]
	if !strings.Contains(fence, "#   session: claude") {
		t.Fatalf("set did not keep advertising the inherited sibling field\n--- got ---\n%s", got)
	}
	if strings.Contains(fence, "#   workers:") {
		t.Fatalf("set re-advertised the live sibling field\n--- got ---\n%s", got)
	}
}

func TestMutation_OpaqueProviderNamesSetUnset(t *testing.T) {
	for _, name := range []string{"123", "true", "on", "-local", "测试"} {
		t.Run(name, func(t *testing.T) {
			original, _ := render("", fieldsForTest(t), ProjectTarget(""), "test")
			path := writeMutationFixture(t, original)
			key := "providers." + name + ".interactive_command"
			if _, err := Set(path, key, "tool", "test"); err != nil {
				t.Fatalf("Set(%q): %v", key, err)
			}
			got := readMutationFixture(t, path)
			serializedName := name
			if name == "123" || name == "true" || name == "on" {
				serializedName = strconv.Quote(name)
			}
			if !strings.Contains(got, serializedName+":") || !strings.Contains(got, "interactive_command: tool") {
				t.Fatalf("Set(%q) did not materialize the provider\n--- got ---\n%s", key, got)
			}
			cfg, err := config.LoadPath(path)
			if err != nil {
				t.Fatalf("LoadPath after Set(%q): %v", key, err)
			}
			provider, ok := cfg.GetProvider(name)
			if !ok || provider.InteractiveCommand != "tool" {
				t.Fatalf("Set(%q) was not effective after load: provider=%+v ok=%v", key, provider, ok)
			}

			if _, err := Unset(path, key, "test"); err != nil {
				t.Fatalf("Unset(%q): %v", key, err)
			}
			if got := readMutationFixture(t, path); got != original {
				t.Fatalf("Set/Unset(%q) did not restore the original bytes\n--- got ---\n%s\n--- want ---\n%s", key, got, original)
			}
		})
	}
}

func TestMutation_UnencodableProviderNamesRefuseWithoutWrite(t *testing.T) {
	for _, key := range []string{
		"providers.#local.interactive_command",
		"providers.local:dev.interactive_command",
		"providers. local.interactive_command",
		"providers.local .interactive_command",
		"providers.local\nname.interactive_command",
	} {
		t.Run(key, func(t *testing.T) {
			const original = "agent:\n    workers: claude\n"
			path := writeMutationFixture(t, original)
			if _, err := Set(path, key, "tool", "test"); err == nil || !strings.Contains(err.Error(), "dotted config-key grammar") {
				t.Fatalf("Set(%q) error = %v, want dotted-key-grammar refusal", key, err)
			}
			if got := readMutationFixture(t, path); got != original {
				t.Fatalf("rejected provider name changed the file\n--- got ---\n%s\n--- want ---\n%s", got, original)
			}
		})
	}
}

func TestMutation_UnrelatedUnknownKeyStaysLiveAcrossSetUnset(t *testing.T) {
	const original = `legacy_mode: true # unrelated user-owned key
agent:
    workers: claude
`
	path := writeMutationFixture(t, original)

	setResult, err := Set(path, "agent.workers", "codex", "test")
	if err != nil {
		t.Fatalf("Set: %v", err)
	}
	assertUnknownMutationKeyLive(t, path, "codex")
	for _, line := range setResult.Report {
		if strings.Contains(line, "parked") {
			t.Fatalf("Set reported unrelated reconciliation %q", line)
		}
	}

	unsetResult, err := Unset(path, "agent.workers", "test")
	if err != nil {
		t.Fatalf("Unset: %v", err)
	}
	assertUnknownMutationKeyLive(t, path, "")
	for _, line := range unsetResult.Report {
		if strings.Contains(line, "parked") {
			t.Fatalf("Unset reported unrelated reconciliation %q", line)
		}
	}
}

func assertUnknownMutationKeyLive(t *testing.T, path, workers string) {
	t.Helper()
	got := readMutationFixture(t, path)
	live := strings.SplitN(got, "# >>> fab reference", 2)[0]
	if !strings.Contains(live, "legacy_mode: true # unrelated user-owned key") {
		t.Fatalf("mutation removed the unrelated unknown live key\n--- got ---\n%s", got)
	}
	if strings.Contains(got, "#   legacy_mode: true") || strings.Contains(got, `parked removed field "legacy_mode"`) {
		t.Fatalf("mutation parked the unrelated unknown live key\n--- got ---\n%s", got)
	}
	if workers != "" && !strings.Contains(live, "workers: "+workers) {
		t.Fatalf("mutation did not set requested key to %q\n--- got ---\n%s", workers, got)
	}
	if workers == "" && strings.Contains(live, "workers:") {
		t.Fatalf("mutation did not unset requested key\n--- got ---\n%s", got)
	}
}

func writeMutationFixture(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func readMutationFixture(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	return string(data)
}

// ============================================================
// System header inside the fence (260831-n2eo) — T005–T008.
// ============================================================

// T005 shape: the regenerated system config opens with a live key (or the
// fence anchor when nothing is overridden), and the header appears exactly
// once, INSIDE the fence anchors.
func TestSystemShape_OpensWithLiveKeyAndHeaderInsideFence(t *testing.T) {
	t.Run("live overrides present opens with a live key", func(t *testing.T) {
		path := writeMutationFixture(t, "dispatch:\n  mode: pane\n\nagent:\n  session: claude\n  workers: kimi\n")
		if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
			t.Fatalf("Upgrade: %v", err)
		}
		got := readMutationFixture(t, path)
		if !strings.HasPrefix(got, "dispatch:") {
			t.Fatalf("line 1 is not a live key\n--- got ---\n%s", got)
		}
		if !strings.Contains(got, "agent:\n  session: claude\n  workers: kimi") {
			t.Fatalf("live overrides not preserved verbatim\n--- got ---\n%s", got)
		}
		assertHeaderOnceInsideFence(t, got)
	})

	t.Run("no overrides opens with the fence BEGIN anchor", func(t *testing.T) {
		path := writeMutationFixture(t, "")
		if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
			t.Fatalf("Upgrade: %v", err)
		}
		got := readMutationFixture(t, path)
		if !strings.HasPrefix(got, "# >>> fab reference") {
			t.Fatalf("line 1 is not the fence BEGIN anchor\n--- got ---\n%s", got)
		}
		assertHeaderOnceInsideFence(t, got)
	})
}

// T006 ownership invariant: an EDITED header copy above the fence (R10a
// preserves it) is never duplicated by a freshly installed pristine copy — the
// header prose appears exactly once, and a user comment above the fence
// survives.
func TestSystemOwnership_EditedAboveFenceHeaderNeverDuplicated(t *testing.T) {
	t.Run("edited v2.23.8 header yields exactly one copy", func(t *testing.T) {
		edited := strings.Replace(SystemScaffoldHeader,
			"all repos on this machine", "all my repos (edited)", 1)
		path := writeMutationFixture(t, edited+"\n\nagent:\n  workers: codex\n")
		if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
			t.Fatalf("Upgrade: %v", err)
		}
		got := readMutationFixture(t, path)
		if !strings.Contains(got, edited) {
			t.Fatalf("edited header paragraph was not preserved as user content\n--- got ---\n%s", got)
		}
		above := strings.SplitN(got, "# >>> fab reference", 2)[0]
		if strings.Contains(above, SystemScaffoldHeader) {
			t.Fatalf("a pristine header copy was installed above the edited one (two-copy outcome)\n--- got ---\n%s", got)
		}
	})

	t.Run("user comment above the fence survives", func(t *testing.T) {
		path := writeMutationFixture(t, "# work laptop only\nagent:\n  workers: codex # pinned\n")
		if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
			t.Fatalf("Upgrade: %v", err)
		}
		got := readMutationFixture(t, path)
		if !strings.HasPrefix(got, "# work laptop only\nagent:") {
			t.Fatalf("user comment above the fence did not survive at the top\n--- got ---\n%s", got)
		}
	})
}

// testV2238HeaderAboveFence is the verbatim v2.23.8 (tag) SystemScaffoldHeader
// paragraph — a HARD CODED copy of the shipped text, not the current constant —
// so the relocation test exercises the historical shape real users carry, not
// a fixture that drifts whenever the registry renderer changes. Kept
// independent of configref.Fields for the same reason as
// legacyV2198ToV2220ProvidersAdvert.
const testV2238HeaderAboveFence = `# ~/.fab-kit/config.yaml — system-level fab config (all repos on this machine).
# Resolves ABOVE any project's fab/project/config.yaml (and below environment
# overrides), so a preference set here beats a repo's committed suggestion. Only
# preference-class fields (scope: system/both) are honored here; a project-scoped
# field placed in this file is ignored with a warning.
#
# This scaffold is generated by ` + "`fab config init --system`" + ` and refreshed by
# ` + "`fab config upgrade --system`" + ` from the same per-field metadata table as
# ` + "`fab config explain`" + `, so it cannot drift from the schema. Every block in
# the managed fence below is COMMENTED — move a field above the fence and uncomment
# it to set a system-level override.`

// T008 relocation: a v2.23.8-shaped file (pristine header ABOVE the fence)
// relocates — the header is recognized as generated and discarded, leaving
// exactly one copy inside the fence, with live keys byte-identical and
// `config show`-relevant YAML unchanged.
func TestSystemRelocation_DiscardsPristineAboveFenceHeader(t *testing.T) {
	path := writeMutationFixture(t,
		testV2238HeaderAboveFence+"\n\ndispatch:\n  mode: pane\n\nagent:\n  session: claude\n  workers: kimi\n")
	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
		t.Fatalf("Upgrade: %v", err)
	}
	got := readMutationFixture(t, path)
	assertHeaderOnceInsideFence(t, got)
	if strings.Contains(got, testV2238HeaderAboveFence) {
		t.Fatalf("the v2.23.8 above-fence header survived (two-copy outcome)\n--- got ---\n%s", got)
	}
	if !strings.HasPrefix(got, "dispatch:\n  mode: pane\n\nagent:\n  session: claude\n  workers: kimi") {
		t.Fatalf("live keys were not byte-identical after relocation\n--- got ---\n%s", got)
	}

	// Idempotent: a second run is byte-identical.
	if _, err := Upgrade(SystemTarget(path), "2.30.0"); err != nil {
		t.Fatalf("second Upgrade: %v", err)
	}
	if second := readMutationFixture(t, path); second != got {
		t.Fatal("relocation was not idempotent")
	}

	// set/unset on a post-change file keep the fence intact and --check clean.
	if _, err := SetSystem(path, "agent.workers", "claude", "2.30.0"); err != nil {
		t.Fatalf("SetSystem: %v", err)
	}
	if _, err := UnsetSystem(path, "agent.workers", "2.30.0"); err != nil {
		t.Fatalf("UnsetSystem: %v", err)
	}
	clean, err := Check(SystemTarget(path), "2.30.0")
	if err != nil || clean.Changed {
		t.Fatalf("set/unset on a post-change file left the target drifted: %+v, %v", clean, err)
	}
}

// assertHeaderOnceInsideFence asserts the canonical SystemScaffoldHeader
// appears exactly once in the file, positioned strictly between the BEGIN and
// END anchors, and that no fab-generated header text sits above the fence.
func assertHeaderOnceInsideFence(t *testing.T, got string) {
	t.Helper()
	if count := strings.Count(got, SystemScaffoldHeader); count != 1 {
		t.Fatalf("header appears %d times, want exactly 1\n--- got ---\n%s", count, got)
	}
	headerIdx := strings.Index(got, SystemScaffoldHeader)
	beginIdx := strings.Index(got, "# >>> fab reference")
	endIdx := strings.Index(got, "# <<< end fab reference")
	if beginIdx == -1 || endIdx == -1 || headerIdx < beginIdx || headerIdx > endIdx {
		t.Fatalf("header is not strictly inside the fence anchors\n--- got ---\n%s", got)
	}
}
