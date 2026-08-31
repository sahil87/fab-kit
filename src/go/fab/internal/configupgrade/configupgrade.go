// Package configupgrade mechanically reconciles fab/project/config.yaml against
// the binary-owned field registry (internal/configref). It is the ONLY writing
// engine for existing config files: the
// comment-clobbering setFabVersion masher is deleted, so every config.yaml write
// flows through this comment-aware, byte-stable package.
//
// THE FIELD-CATEGORY MODEL (docs/specs/config.md § Advertise semantics). At upgrade
// time every registry field is one of:
//   - A) live (user-overridden)     → kept VERBATIM above the managed fence,
//     including the user's own comments. Presence = intent: a live field is an
//     override even when its value equals the default; NEVER auto-removed.
//   - B) not overridden              → absent (inherited from defaults).
//   - C) not overridden, advertise:true → regenerated as a fully-commented scaffold
//     INSIDE the managed fence, so the user can discover and opt in.
//
// THE MANAGED FENCE. A region delimited by byte-exact splice anchors:
//
//	# >>> fab reference (kit X.Y.Z) >>> ----…
//	…commented C-field scaffold…
//	# <<< end fab reference <<< ----…
//
// Upgrade rewrites ONLY the region between (and including) the two anchor lines;
// everything outside is the user's. A legacy project file gets a fence appended;
// a legacy system scaffold first has only recognized generated lines replaced.
// The kit-version stamp in the BEGIN line makes staleness visible
// (it feeds the Check drift probe behind `fab config upgrade --check`). Every
// scaffolded
// block is fully commented INCLUDING parent keys (a live `agent:` key over
// comment-only children is exactly what the old masher collapsed to `agent: null`).
// The fence omits fields already overridden as live keys above it.
//
// PARKED REMOVALS. A live top-level key no longer in the registry is parked in a
//
//	# removed in X.Y.Z (parked by fab config upgrade — delete when done):
//	#   key: value
//
// block BELOW the fence, its value serialized in the comment — never silently
// deleted. Parkings are user territory: appended exactly once, never regenerated
// away on a later run.
//
// RENAMES. A live top-level key matching some registry row's renamed_from is
// carried to the new key mechanically (value verbatim), replacing the per-rename
// hand-written-migration pattern. renamed_from is "" on every row today, so this
// path fires only for a seeded rename.
//
// OUTPUT DISCIPLINE. Byte-stable and idempotent — running Upgrade twice yields a
// byte-identical file (the internal/memoryindex precedent: pure renders, golden +
// idempotence tests). The write goes through internal/atomicfile.WriteFile.
package configupgrade

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/sahil87/fab-kit/src/go/fab/internal/atomicfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/configref"
)

// fenceWidth is the total column width the anchor lines are dash-padded to, so the
// BEGIN and END rules line up as a visible box (matching the worked example in
// fab/plans/sahil/26-07-08-config-upgrade.md § The fence).
const fenceWidth = 76

// beginPrefix / endPrefix are the byte-exact splice-anchor prefixes. The full
// anchor line is the prefix followed by a run of `-` padding out to fenceWidth.
// beginPrefix carries a `%s` for the kit-version stamp.
const (
	beginPrefix = "# >>> fab reference (kit %s) >>> "
	endPrefix   = "# <<< end fab reference <<< "
)

// beginLineRe matches the fence BEGIN anchor regardless of the stamped version or
// the dash-pad width, so a re-run finds the existing fence even after a version
// bump changed the stamp. endLineRe matches the END anchor likewise.
var (
	beginLineRe = regexp.MustCompile(`^# >>> fab reference \(kit [^)]*\) >>> -*\s*$`)
	endLineRe   = regexp.MustCompile(`^# <<< end fab reference <<< -*\s*$`)
)

// parkedHeaderRe matches a parked-removal block header (any version phrasing), so
// a re-run recognizes an already-parked block and never re-parks the same key. The
// version portion may contain spaces ("an earlier release"), so match lazily up to
// the fixed "(parked by fab config upgrade" marker.
var parkedHeaderRe = regexp.MustCompile(`^# removed in .+ \(parked by fab config upgrade`)

// knownGeneratedSystemParagraphDigests is the byte-exact identity catalog for
// every distinct paragraph emitted by released `fab config init --system`
// versions from v2.15.1 through v2.23.7, plus the current header. Keeping
// fingerprints rather than obsolete command/prose blocks makes the preservation
// boundary explicit without carrying hundreds of dead config lines in the
// binary. TestGeneratedSystemParagraphCatalogIncludesCurrentRenderer forces a
// registry change to add its new rendering while retaining all older identities.
//
// Exact paragraph identity is deliberately stronger than structural matching:
// changing or appending even one line misses the digest, so R10a preserves the
// whole paragraph as user content.
//
// Why not SHAPE-based accounting instead (a line is generated iff it parses as
// a commented key path belonging to the live registry, or matches a fixed
// envelope form — value-insensitive, so it never depends on remembered text)?
// Because R10a's edited-copy case IS a value-only edit: a hand-changed
// `#   mode: pane` sitting under a correct `[both]` suffix and a correct
// `# Full prose:` pointer. Shape matching is value-blind by construction, so it
// would account for every line of that copy and delete it again — the exact
// silent deletion R10a exists to forbid. Accounting must be value-sensitive,
// which means byte-exact text, which for historical renderings means remembered
// text; the digest catalog is that memory in compact form. Its maintenance cost
// is bounded by the test above: a prose edit changes the current rendering's
// digest and fails the suite until the new identity is appended, while every
// older identity stays — the catalog is append-only and self-enforcing, so a
// missed entry requires bypassing the test itself.
var knownGeneratedSystemParagraphDigests = map[string]struct{}{
	// v2.15.1
	"8ae7c26c402e9eb92671ada7ff8ab74fd1ea77e62fee7ed0ea6d4c01b0059d37": {},
	"b1bea7569fe91eae4f5314cf45d8aa4fe74083c63c860d97e6516a0ad4fb62e1": {},
	"cd942d00aeea81a10f06555896e6816f65f3419f220a886a02efb08c8794a445": {},
	// v2.16.x additions
	"635e940f4a98e41012d31ed4a23811e3911aa3dc2881874841db4e7687a981a2": {},
	"d4e1e2defbaf7ebbcde41b0dfa9279261d3f31570e37b7ab56c1492ca9a7c26b": {},
	"dd75b253c3875ebe46cf8aef9ae961c8431d294134e4e90f0de5ea4b9fe6fdcf": {},
	"b93e684257565246ed859b0919714665af841e6c92320d8ffd1f403a65e6fa99": {},
	"85fcf74c3964affc2b55c5eae5571be32c77a4cc3b42d50867117632c106e4c8": {},
	"f3a52c3a59a0a912c214a491ae3139032ce8b9cdbcbae4ab674481705e5ae730": {},
	"51ecf3813e442ff8f4878621c6459267932bd6dcf0f3c060cfae6c29677d3cec": {},
	"ef8931bec634b3809b3a5cd18853d3704a3b8ee4b0771785d507284e68d5de21": {},
	"8b8548b0527a965ccc9ccef3849c2a9c3b706611b80a7344d161137ef0b0c36c": {},
	"08c97dce5d9e3c986c1a800aa88b24c26ea8be9b8081f5660d6d128c2bd052de": {},
	"9b147878f6f2face5f386248589e29b0198201343193c0242d53314749f8ec0e": {},
	// v2.17.x additions
	"a4348bef86de9de95cb8fb8b5581b1aabde0e8284c50acccc756d0a557aa696d": {},
	"a8d09c269b33942ea2933d9448813570c89915ff533173e3c9ebf21ddc3a1188": {},
	"9357ffd1fedfc3c39cdd3cd19e9744738962714482f4f77f5e002b5dc07bb89d": {},
	"2c517fcd3c0f02c1426b82e8f953d51420e299a0bf70697dfba6679d17711544": {},
	// v2.18.x additions
	"bbe046f65eab64f0dec922ffe0ad48c801c198e6103f3c37920dcc309bbd7079": {},
	"bfdc8b5751e158913913ca067424bc99a41f186e8a6d430ce9afd4019afc6e21": {},
	"4a27edfc2c70abfe811bde2a701fb7d6e625f029c91d89a43fdf883ee981fbd2": {},
	"e10c0e4abdd79ad7aad3ff49b92b689dd7b6060a3b920bfb7e1779e086544de0": {},
	"00303674d45e1d07db5c62675465dbbaef2aeefcfc749c4de96ecb5faac587d5": {},
	"10ef9df9e2061b6064f1687045a51e86a366d8158a9988ecf03191a9a0e0956d": {},
	// v2.19.x additions
	"bcd4619025a01a93bd5ecdb408167ec4d58de680621deca2b07754846351e050": {},
	"35445ea3a6667e4c004658a5e2906bcb36c436a6143a426e5941a520e7a55ca4": {},
	"221faf9b7b4adbb39df5847938013ad64b7cab0da709a3d9be4b2ee452359380": {},
	"0be26fc1d868405e02a73d51d6cabe6be3f6b4ebbbdfedc275634f029a86f75c": {},
	"20f90b461709754742751f9eb60ddd7f23719a36927616973d73d1c5086eb835": {},
	"3b6df3c6aade43ea60f3cd88e841d1c99dbe2a112a305e87b862706540450098": {},
	"6350bdfad219c111677129796777328fd3666eebddc0fc120396ba472914601a": {},
	// v2.20.9 and v2.23.5 additions
	"4a1c4c4d2e66e8496bd740cc7295f506e8b393fa4940fea968f943a7bfd26e09": {},
	"03efe32c365db8d64cd43096eb5a3ff3159a126f8e3ff4de8b75fab26db4b763": {},
	// Current system header (the current advert digests are already above).
	"fe4bd2e1bbd7d22925c18de88e1af87d1ebc559aaf68714a580dbf5bf5e691e6": {},
}

// fenceHeaderComment returns the explanatory preamble emitted at the top of a
// target's fence body. The prose is single-sourced while the command spelling is
// target-specific (`fab config upgrade` for projects, the --system form for the
// machine-level file).
func fenceHeaderComment(command string) string {
	return `# Overridable fields you have NOT overridden, with current defaults.
# REGENERATED by ` + "`" + command + "`" + ` on every upgrade — edits inside this
# fence are overwritten. To override a field: move it ABOVE the fence and
# uncomment it.`
}

// FieldFilter decides whether a registry field belongs to a reconciliation
// target. It is part of Target so init, upgrade, and surgical mutation all use
// the same scope definition.
type FieldFilter func(configref.Field) bool

// Target describes one config file reconciled by the shared engine.
type Target struct {
	Path            string
	Header          string
	FieldFilter     FieldFilter
	FencePreamble   string
	advertisedOnly  bool
	adoptLegacyFile bool
}

// ProjectTarget returns the existing project-config behavior. All registry
// fields are known, while only advertise:true rows enter the managed fence.
func ProjectTarget(path string) Target {
	return Target{
		Path:           path,
		FieldFilter:    func(configref.Field) bool { return true },
		FencePreamble:  fenceHeaderComment("fab config upgrade"),
		advertisedOnly: true,
	}
}

// SystemTarget returns the machine-level target. Every system-visible field is
// scaffolded, including rows intentionally omitted from project fences.
func SystemTarget(path string) Target {
	return Target{
		Path:            path,
		Header:          SystemScaffoldHeader,
		FieldFilter:     SystemField,
		FencePreamble:   fenceHeaderComment("fab config upgrade --system"),
		adoptLegacyFile: true,
	}
}

// SystemField is the shared, testable scope predicate used by system init,
// upgrade, and mutation.
func SystemField(field configref.Field) bool {
	return field.Scope == configref.ScopeSystem || field.Scope == configref.ScopeBoth
}

func (t Target) includesField(field configref.Field) bool {
	return t.FieldFilter == nil || t.FieldFilter(field)
}

func (t Target) includesFenceField(field configref.Field) bool {
	return t.includesField(field) && (!t.advertisedOnly || field.Advertise)
}

func (t Target) reconciliationFields(fields []configref.Field) []configref.Field {
	if t.FieldFilter == nil {
		return fields
	}
	selected := make([]configref.Field, 0, len(fields))
	for _, field := range fields {
		if t.includesField(field) {
			selected = append(selected, field)
		}
	}
	return selected
}

// Result reports what an Upgrade run did (or what a Check run computed), for
// the command's advisory output.
type Result struct {
	// Changed is true when the on-disk file was rewritten (Upgrade) or WOULD be
	// rewritten by a real run (Check) — false = already byte-identical, a no-op.
	Changed bool
	// Report holds advisory lines for the user (B-hygiene notes, parked-key
	// notices, rename carries). Never fatal — informational only.
	Report []string
}

// Upgrade reconciles the config.yaml at path against the registry, stamping the
// fence with kitVersion. It preserves live (A) fields verbatim, regenerates the
// managed fence of commented C fields, parks unknown live keys once below the
// fence, and carries renamed_from renames. The write is atomic and the output is
// byte-stable/idempotent. A missing file is treated as empty (a fresh fence-only
// file is written).
func Upgrade(target Target, kitVersion string) (Result, error) {
	rendered, changed, report, err := computeUpgrade(target, kitVersion)
	if err != nil {
		return Result{}, err
	}
	if !changed {
		return Result{Changed: false, Report: report}, nil
	}
	if err := os.MkdirAll(filepath.Dir(target.Path), 0o755); err != nil {
		return Result{}, fmt.Errorf("configupgrade: creating parent directory for %s: %w", target.Path, err)
	}
	if err := atomicfile.WriteFile(target.Path, []byte(rendered), 0o644); err != nil {
		return Result{}, fmt.Errorf("configupgrade: writing %s: %w", target.Path, err)
	}
	return Result{Changed: true, Report: report}, nil
}

// Check computes exactly what Upgrade would do to the config.yaml at path and
// reports the verdict WITHOUT writing: Result.Changed is true when a real run
// would rewrite the file (a stale fence kit-version stamp, unparked unknown
// keys, a missing fence, or any rendered-content delta — including a missing
// file, which a real run would create), false when the file is already clean. It
// shares Upgrade's entire compute path (computeUpgrade), so the drift probe can
// never disagree with an applying run about what would change. This is the
// engine behind `fab config upgrade --check`.
func Check(target Target, kitVersion string) (Result, error) {
	_, changed, report, err := computeUpgrade(target, kitVersion)
	if err != nil {
		return Result{}, err
	}
	return Result{Changed: changed, Report: report}, nil
}

// computeUpgrade runs the whole reconciliation computation — registry load, file
// read, render, and the refuse-unparseable validation — without any write. It is
// the single compute path Upgrade and Check share (Upgrade writes the rendered
// bytes when changed, Check only reports the verdict), so the two can never
// disagree about whether a run would change the file. changed is true when the
// file is missing (a run would create it) or the rendered bytes differ from the
// original.
func computeUpgrade(target Target, kitVersion string) (rendered string, changed bool, report []string, err error) {
	fields, err := configref.Fields()
	if err != nil {
		return "", false, nil, err
	}

	original, existed, err := readFile(target.Path)
	if err != nil {
		return "", false, nil, err
	}

	rendered, report = render(original, fields, target, kitVersion)

	// Refuse to write YAML that does not parse (SF-c defense-in-depth). The
	// comment-aware splice manipulates raw lines, so a pathological input (e.g. an
	// interior column-0 comment inside a live block) could in principle produce
	// malformed YAML; a config that fails to load bricks every fab command, so we
	// validate the rendered bytes and refuse — surfacing the parse error — rather
	// than overwrite the user's file with something unparseable. The original file
	// is left untouched.
	if err := validateYAML(rendered); err != nil {
		return "", false, nil, fmt.Errorf("configupgrade: refusing to write %s — the reconciled output does not parse as YAML (%w); the file was left unchanged", target.Path, err)
	}

	return rendered, !existed || rendered != original, report, nil
}

// validateYAML reports whether s parses as a YAML document (into a generic map),
// so Upgrade can refuse to write a file that would fail to load. Comments and the
// fully-commented fence are inert to the parser; only the live keys are validated.
func validateYAML(s string) error {
	var m map[string]any
	return yaml.Unmarshal([]byte(s), &m)
}

// InitSeed carries the A-class identity values `fab config init --project` writes
// LIVE above the managed fence on a fresh repo. Any field left empty is omitted
// from the live block (and, for test_paths, left to the fence to advertise). Every
// other config field is fence territory from day one — presence=intent means an
// init-pinned knob or role profile would be an accidental override.
type InitSeed struct {
	Name        string
	Description string
	SourcePaths []string
	TestPaths   []string
}

// RenderInitProject generates a fresh fab/project/config.yaml for `fab config init
// --project`: the seeded A-class identity block (name/description/source_paths, and
// test_paths when detected) written LIVE, followed by the managed fence of
// commented C fields. It shares the exact fence renderer Upgrade uses (renderFence),
// so a freshly-generated file and an upgraded one carry a byte-identical fence.
// NO `agent:` key is ever written live — presence=intent means an init-pinned knob
// or role profile would be an accidental override, so agent config stays
// fence-only. The fence omits the seeded live keys.
func RenderInitProject(seed InitSeed, kitVersion string) (string, error) {
	fields, err := configref.Fields()
	if err != nil {
		return "", err
	}

	preamble := renderInitPreamble(seed)
	liveKeys := liveTopLevelKeys(preamble)
	fence := renderFence(ProjectTarget(""), fields, liveKeys, kitVersion)
	return assemble(preamble, fence, nil, nil), nil
}

// RenderSystemScaffold generates the fresh system config shape from the same
// target descriptor upgrade and mutation use: canonical header followed by a
// managed fence containing every system-visible field.
func RenderSystemScaffold(kitVersion string) (string, error) {
	fields, err := configref.Fields()
	if err != nil {
		return "", err
	}
	target := SystemTarget("")
	fence := renderFence(target, fields, nil, kitVersion)
	return assemble(target.Header, fence, nil, nil), nil
}

// initHeader is the top-of-file banner the generated project config carries — it
// states the presence=override contract so a reader knows why a field being live
// is meaningful.
const initHeader = `# fab/project/config.yaml — project overrides.
# Presence = override: any live field here wins over ~/.fab-kit/config.yaml and
# built-in defaults, even if its value equals the default. Generated by
# ` + "`fab config init`" + `; reconciled on every upgrade by ` + "`fab config upgrade`" + `.`

// renderInitPreamble emits the seeded identity block: the header banner then the
// live A-class fields present in the seed. Absent seed values are omitted (their
// fields stay fence-advertised where advertise:true). Output is deterministic.
func renderInitPreamble(seed InitSeed) string {
	var b strings.Builder
	b.WriteString(initHeader)

	if seed.Name != "" || seed.Description != "" {
		b.WriteString("\n\nproject:")
		if seed.Name != "" {
			fmt.Fprintf(&b, "\n    name: %s", yamlScalar(seed.Name))
		}
		if seed.Description != "" {
			fmt.Fprintf(&b, "\n    description: %s", yamlScalar(seed.Description))
		}
	}

	if len(seed.SourcePaths) > 0 {
		b.WriteString("\n\nsource_paths:")
		for _, p := range seed.SourcePaths {
			fmt.Fprintf(&b, "\n    - %s", yamlScalar(p))
		}
	}

	if len(seed.TestPaths) > 0 {
		b.WriteString("\n\ntest_paths:")
		for _, p := range seed.TestPaths {
			fmt.Fprintf(&b, "\n    - %s", yamlScalar(p))
		}
	}

	return b.String()
}

// yamlScalar renders a string as a YAML scalar, quoting it when it contains a
// character that would need quoting (a glob `*`, a leading/trailing space, a colon,
// a `#`, a backslash, a control character, or another YAML-significant indicator).
// Config identity values are simple strings; the conservative quoting keeps glob
// test-path patterns (`**/*_test.go`) valid. When quoting, a backslash and an
// embedded double-quote are BOTH escaped (a raw `\` inside a double-quoted YAML
// scalar is itself an escape lead-in — leaving it unescaped would corrupt the value
// or produce invalid YAML), and control characters are escaped so the emitted line
// is always parseable.
func yamlScalar(s string) string {
	if s == "" {
		return `""`
	}
	if strings.ContainsAny(s, "*:#{}[]&!|>%@`\"'\\") || s != strings.TrimSpace(s) || hasControlChar(s) {
		var b strings.Builder
		b.WriteByte('"')
		for _, r := range s {
			switch r {
			case '\\':
				b.WriteString(`\\`)
			case '"':
				b.WriteString(`\"`)
			case '\n':
				b.WriteString(`\n`)
			case '\t':
				b.WriteString(`\t`)
			case '\r':
				b.WriteString(`\r`)
			default:
				if r < 0x20 {
					fmt.Fprintf(&b, `\x%02x`, r)
					continue
				}
				b.WriteRune(r)
			}
		}
		b.WriteByte('"')
		return b.String()
	}
	return s
}

// hasControlChar reports whether s contains an ASCII control character (< 0x20),
// which must be escaped inside a double-quoted YAML scalar.
func hasControlChar(s string) bool {
	for _, r := range s {
		if r < 0x20 {
			return true
		}
	}
	return false
}

// readFile reads path, returning ("", false, nil) when it does not exist.
func readFile(path string) (string, bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", false, nil
		}
		return "", false, err
	}
	return string(data), true, nil
}

// render is the pure core: given the original file text, the registry fields, and
// the kit-version stamp, it returns the reconciled file text and the advisory
// report. Extracted from Upgrade (which owns file I/O) so it is directly
// unit-testable and provably deterministic.
func render(original string, fields []configref.Field, target Target, kitVersion string) (string, []string) {
	hadFence := hasManagedFence(original)
	preamble, belowFence, existingParked := splitFence(original)

	// Everything outside the fence is the user's and is NEVER silently dropped.
	// Non-parked content the user placed BELOW the fence (e.g. a live override
	// appended after the END anchor) is hoisted ABOVE the fence into the preamble,
	// where it is classified exactly like any other live key (kept verbatim if
	// known, parked if unknown, carried if renamed). This makes the layout
	// self-healing: the next run re-parses it as an ordinary live A field. See R2.1.
	if strings.TrimSpace(belowFence) != "" {
		preamble = strings.TrimRight(preamble, "\n") + "\n" + strings.TrimLeft(belowFence, "\n")
	}
	preamble = normalizeTargetPreamble(preamble, fields, target)
	out, report := reconcilePreamble(preamble, existingParked, target.reconciliationFields(fields), target, kitVersion)
	if target.adoptLegacyFile && original != "" && !hadFence {
		report = append([]string{"adopted existing unfenced system config into the managed fence (live keys and unrecognized comments preserved)"}, report...)
	}
	return out, report
}

// reconcilePreamble performs the Upgrade-only reconciliation of already-extracted
// user-owned live content while preserving parked blocks from an existing fence.
// Surgical mutation deliberately does not call this function: rename carry,
// unknown-key parking, and B-hygiene are whole-file upgrade concerns and must not
// affect an unrelated key during set/unset.
func reconcilePreamble(preamble string, existingParked []string, fields []configref.Field, target Target, kitVersion string) (string, []string) {
	var report []string

	// Live top-level keys the user has above the fence (the A set), plus their
	// verbatim source blocks for rename carry-forward.
	liveKeys := liveTopLevelKeys(preamble)

	// Carry renames: a live key matching some row's RenamedFrom is rewritten to the
	// new key in place (value verbatim). Only TOP-LEVEL→top-level renames are
	// carriable here; the one row carrying a renamed_from today (agent.profiles,
	// from agent.tiers) is a same-top-level rename and is deliberately skipped —
	// internal/config reads `agent.tiers` as a deprecated alias so it keeps
	// resolving, and the 2.16.19-to-2.17.0 migration does the on-disk rewrite.
	preamble, liveKeys, renameReport := carryRenames(preamble, liveKeys, fields)
	report = append(report, renameReport...)

	// Park unknown live keys (live top-level keys not documented by any registry
	// row and not the fence/parked markers). Removed from the live preamble,
	// appended once below the fence.
	preamble, newlyParked := extractUnknownLiveKeys(preamble, fields)

	// Build the fence: project reconciliation retains its established top-level
	// suppression, while the system target uses the mutation path's leaf-aware
	// filtering. That keeps a live agent.workers override from hiding the sibling
	// agent.session advert and, critically, makes SetSystem followed by Check clean.
	var fence string
	if target.adoptLegacyFile {
		livePaths := liveDottedPaths(preamble)
		fence = renderFenceWithSegments(target, fields, kitVersion, func(index int, field configref.Field) string {
			return mutationFenceSegment(target, fields, index, livePaths)
		})
	} else {
		fence = renderFence(target, fields, liveKeys, kitVersion)
	}

	// B-hygiene advisory (presence=intent — never a mutation): live fields whose
	// value equals the current default. Reported only, never removed.
	report = append(report, bHygieneReport(preamble, fields)...)

	out := assemble(preamble, fence, existingParked, newlyParked)
	for _, k := range sortedKeys(newlyParked) {
		report = append(report, fmt.Sprintf("parked removed field %q below the fence (delete when done)", k))
	}
	return out, report
}

// renderMutation rebuilds only the managed fence around an already-mutated live
// area. Unlike reconcilePreamble it does not carry renames, park unknown keys, or
// inspect unrelated values for upgrade advisories. Mutation fence rendering is
// leaf-aware: a shared segment contains only its advertised rows that are not live.
func renderMutation(preamble string, existingParked []string, fields []configref.Field, target Target, kitVersion string) (string, []string) {
	livePaths := liveDottedPaths(preamble)
	fence := renderFenceWithSegments(target, fields, kitVersion, func(index int, f configref.Field) string {
		return mutationFenceSegment(target, fields, index, livePaths)
	})
	return assemble(preamble, fence, existingParked, nil), nil
}

func liveDottedPaths(document string) [][]string {
	var paths [][]string
	var root map[string]any
	if err := yaml.Unmarshal([]byte(document), &root); err == nil {
		var walk func([]string, any)
		walk = func(path []string, value any) {
			paths = append(paths, append([]string{}, path...))
			if children, ok := value.(map[string]any); ok {
				for key, child := range children {
					next := append(append([]string{}, path...), key)
					walk(next, child)
				}
			}
		}
		for key, value := range root {
			walk([]string{key}, value)
		}
		return paths
	}

	// Unset is intentionally a repair path for malformed values. If a surviving
	// unrelated value still prevents YAML decoding, retain the scanner's exact
	// block-form paths as the conservative fallback.
	for _, entry := range scanLiveKeyLines(strings.Split(document, "\n")) {
		paths = append(paths, append([]string{}, entry.path...))
	}
	return paths
}

func mutationFenceSegment(target Target, fields []configref.Field, ownerIndex int, livePaths [][]string) string {
	top := topLevel(fields[ownerIndex].Key)
	var live [][]string
	advertised := 0
	for i := ownerIndex; i < len(fields) && topLevel(fields[i].Key) == top; i++ {
		if i > ownerIndex && fields[i].Segment != "" {
			break
		}
		if !target.includesFenceField(fields[i]) {
			continue
		}
		advertised++
		fieldPath := strings.Split(fields[i].Key, ".")
		if containsPath(livePaths, fieldPath) {
			live = append(live, fieldPath)
		}
	}
	if len(live) == 0 {
		return fields[ownerIndex].ShortSegment
	}
	if len(live) == advertised {
		return ""
	}
	return segmentWithoutLiveRows(fields[ownerIndex].ShortSegment, live)
}

func segmentWithoutLiveRows(segment string, livePaths [][]string) string {
	lines := strings.Split(CommentOutSegment(segment), "\n")
	virtual := append([]string{}, lines...)
	for i, line := range virtual {
		virtual[i] = strings.TrimPrefix(line, "# ")
	}

	remove := make(map[int]bool, len(livePaths))
	for _, entry := range scanLiveKeyLines(virtual) {
		for _, live := range livePaths {
			if pathEqual(entry.path, live) {
				remove[entry.line] = true
				break
			}
		}
	}
	out := make([]string, 0, len(lines)-len(remove))
	for i, line := range lines {
		if !remove[i] {
			out = append(out, line)
		}
	}
	return strings.Join(out, "\n")
}

func containsPath(paths [][]string, want []string) bool {
	for _, path := range paths {
		if pathEqual(path, want) {
			return true
		}
	}
	return false
}

// splitFence divides the original file into three parts:
//   - preamble: the content BEFORE the fence (the user's live A fields and comments).
//   - belowFence: NON-parked content found AFTER the END anchor — the user's, and
//     NEVER dropped. The caller hoists it above the regenerated fence so it is
//     classified like any other live key (R2.1). Empty in a well-formed fab-written
//     file; non-empty when the user appended an override below the fence.
//   - parked: the parked-removal comment blocks below the fence, preserved verbatim.
//
// The fence body itself is discarded (it is regenerated). When no fence exists,
// everything is preamble (parked blocks are only ever created by Upgrade, below a
// fence, so a fence-less file has neither parked blocks nor below-fence content).
func splitFence(original string) (preamble, belowFence string, parked []string) {
	if original == "" {
		return "", "", nil
	}
	lines := strings.Split(original, "\n")

	beginIdx, endIdx := -1, -1
	for i, ln := range lines {
		if beginIdx == -1 && beginLineRe.MatchString(ln) {
			beginIdx = i
			continue
		}
		if beginIdx != -1 && endLineRe.MatchString(ln) {
			endIdx = i
			break
		}
	}

	if beginIdx == -1 || endIdx == -1 {
		// No fence — the whole file is preamble (parked blocks never exist without
		// a fence, since only Upgrade writes them and it always writes a fence).
		return original, "", nil
	}

	pre := strings.Join(lines[:beginIdx], "\n")
	post := lines[endIdx+1:]
	parked, nonParked := extractParkedBlocks(post)
	return pre, strings.Join(nonParked, "\n"), parked
}

func hasManagedFence(document string) bool {
	begin, end := false, false
	for _, line := range strings.Split(document, "\n") {
		if !begin && beginLineRe.MatchString(line) {
			begin = true
			continue
		}
		if begin && endLineRe.MatchString(line) {
			end = true
			break
		}
	}
	return begin && end
}

// normalizeTargetPreamble installs a target-owned header and, for the system
// target, removes only byte-exact generated paragraphs from an unfenced legacy
// scaffold. If even one line was appended or edited, the paragraph does not
// match and survives whole above the new fence (R10a).
func normalizeTargetPreamble(preamble string, fields []configref.Field, target Target) string {
	if target.Header == "" {
		return preamble
	}

	paragraphs := splitParagraphs(preamble)
	kept := make([]string, 0, len(paragraphs))
	for _, paragraph := range paragraphs {
		lines := strings.Split(paragraph, "\n")
		hasLive := false
		for _, line := range lines {
			trimmed := strings.TrimSpace(line)
			if trimmed != "" && !strings.HasPrefix(trimmed, "#") {
				hasLive = true
				break
			}
		}
		if hasLive {
			kept = append(kept, paragraph)
			continue
		}
		if isGeneratedSystemParagraph(paragraph, fields, target) {
			continue
		}
		kept = append(kept, paragraph)
	}

	preserved := strings.Trim(strings.Join(kept, "\n\n"), "\n")
	if preserved == "" {
		return target.Header
	}
	return target.Header + "\n\n" + preserved
}

// isGeneratedSystemParagraph authorizes deletion only when the WHOLE paragraph
// is accounted for by one registry rendering. Current renderings are compared
// directly; released historical renderings use the exact digest catalog above.
// This is intentionally not a line-set or structural match: either could accept
// a user-edited hybrid whose individual markers still resemble generated prose.
func isGeneratedSystemParagraph(paragraph string, fields []configref.Field, target Target) bool {
	if paragraph == SystemScaffoldHeader || paragraph == legacySystemScaffoldHeader {
		return true
	}
	for _, field := range fields {
		if !target.includesFenceField(field) || field.ShortSegment == "" {
			continue
		}
		if paragraph == CommentOutSegment(field.ShortSegment) {
			return true
		}
	}
	digest := systemParagraphDigest(paragraph)
	_, ok := knownGeneratedSystemParagraphDigests[digest]
	return ok
}

func systemParagraphDigest(paragraph string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(paragraph)))
}

func splitParagraphs(document string) []string {
	lines := strings.Split(strings.Trim(document, "\n"), "\n")
	var paragraphs []string
	var current []string
	flush := func() {
		if len(current) == 0 {
			return
		}
		paragraphs = append(paragraphs, strings.Join(current, "\n"))
		current = nil
	}
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			flush()
			continue
		}
		current = append(current, line)
	}
	flush()
	return paragraphs
}

// extractParkedBlocks separates the post-fence lines into the parked-removal blocks
// (each `# removed in …` header plus the contiguous `#`-prefixed / blank lines up to
// the next parked header or live line) and the NON-parked remainder. A parked block
// stays below the fence verbatim; the non-parked remainder is the user's own content
// (a live override they appended below the fence) and is returned separately so the
// caller can hoist it above the fence rather than drop it (R2.1). Blank lines that
// sit between blocks are attributed to neither list (they are structural padding the
// assembler re-inserts), so a fab-written file round-trips with only its parked
// blocks preserved and no spurious blank-line churn.
func extractParkedBlocks(lines []string) (parked []string, nonParked []string) {
	for i := 0; i < len(lines); {
		if strings.TrimSpace(lines[i]) == "" {
			// Structural padding between blocks — belongs to neither list.
			i++
			continue
		}
		if parkedHeaderRe.MatchString(lines[i]) {
			start := i
			i++
			for i < len(lines) && (strings.HasPrefix(lines[i], "#") || strings.TrimSpace(lines[i]) == "") {
				// Stop at the next parked header — it starts its own block.
				if parkedHeaderRe.MatchString(lines[i]) {
					break
				}
				i++
			}
			// Trim trailing blank lines that trailed into this block's capture.
			block := strings.TrimRight(strings.Join(lines[start:i], "\n"), "\n")
			parked = append(parked, block)
			continue
		}
		// A non-parked, non-blank line: the user's own below-fence content. Preserve
		// it (and any following lines up to the next parked header) for hoisting.
		start := i
		i++
		for i < len(lines) && !parkedHeaderRe.MatchString(lines[i]) {
			i++
		}
		nonParked = append(nonParked, strings.TrimRight(strings.Join(lines[start:i], "\n"), "\n"))
	}
	return parked, nonParked
}

// topLevelKeyRe matches a live (non-commented, column-0) `key:` line, capturing the
// key. Indented lines and `#` comments are deliberately excluded — this collects
// exactly the top-level live keys (the A set / override units).
var topLevelKeyRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_]*):(\s|$)`)

// liveTopLevelKeys returns the set of live top-level keys present in the preamble.
func liveTopLevelKeys(preamble string) map[string]bool {
	keys := map[string]bool{}
	for _, ln := range strings.Split(preamble, "\n") {
		if m := topLevelKeyRe.FindStringSubmatch(ln); m != nil {
			keys[m[1]] = true
		}
	}
	return keys
}

// registryTopLevelKeys returns the set of top-level keys the registry documents
// (the dotted keys collapsed to their first segment), plus the TOP-LEVEL rename map
// used by carryRenames.
//
// Rename carry is a TOP-LEVEL-key operation: the upgrader rewrites the column-0
// `key:` token and preserves the whole value block verbatim, so it can only carry a
// rename whose old and new keys are DIFFERENT top-level keys. Two registry shapes
// are therefore deliberately excluded from the rename map:
//   - a same-top-level rename (e.g. `a.x` → `a.y`, both under top-level `a`) —
//     collapsing to `a` → `a` is a no-op that would otherwise log a spurious
//     "carried rename" line on every run;
//   - a nested rename (a renamed_from whose top-level segment differs but which is
//     itself a nested key, e.g. `a.x` → `b.y`) — carrying it as a whole-`a:`-block
//     rename to `b:` would corrupt the value, so it is left for a future
//     nested-aware carry rather than mishandled here.
//
// Both are detected by comparing the FULL dotted keys, not just their first
// segments. ONE row carries renamed_from today — `agent.profiles`, recording the
// 260806-j9nh rename from `agent.tiers` — and it is a same-top-level rename, so it
// is skipped by the first exclusion above and the map is still empty in production
// (that rename ships as a hand-written migration instead). The guard therefore
// matters for that row, for seeded-rename fixtures, and for future renames.
func registryTopLevelKeys(fields []configref.Field) (known map[string]bool, renames map[string]string) {
	known = map[string]bool{}
	renames = map[string]string{}
	for _, f := range fields {
		known[topLevel(f.Key)] = true
		if f.RenamedFrom == "" {
			continue
		}
		oldTop, newTop := topLevel(f.RenamedFrom), topLevel(f.Key)
		// Only a genuine top-level→top-level rename is carriable here: both the
		// old and new keys must be bare top-level keys (no dotted suffix) AND
		// distinct. A same-top-level or nested rename is skipped (see doc above).
		if oldTop == newTop || strings.Contains(f.RenamedFrom, ".") || strings.Contains(f.Key, ".") {
			continue
		}
		renames[oldTop] = newTop
	}
	return known, renames
}

// carryRenames rewrites a live top-level key that matches a registry row's
// renamed_from to the new key (value verbatim), updating the live-key set. Only the
// key token on the top-level line is rewritten; the value and any nested block are
// preserved byte-for-byte.
//
// It SKIPS a carry (and reports the skip) when the target key is already live: yaml
// v3 rejects a duplicate top-level key, so blindly renaming `old` → `new` when the
// user already has a live `new:` would produce a config that fails to load and
// bricks every fab command (violating fail-open). When both keys are present the
// user's explicit `new:` wins and `old` is left in place for the unknown-key parker
// to handle.
func carryRenames(preamble string, liveKeys map[string]bool, fields []configref.Field) (string, map[string]bool, []string) {
	_, renames := registryTopLevelKeys(fields)
	if len(renames) == 0 {
		return preamble, liveKeys, nil
	}
	var report []string
	lines := strings.Split(preamble, "\n")
	for i, ln := range lines {
		m := topLevelKeyRe.FindStringSubmatch(ln)
		if m == nil {
			continue
		}
		old := m[1]
		newKey, ok := renames[old]
		if !ok {
			continue
		}
		if liveKeys[newKey] {
			// Target already live — a carry would emit a duplicate top-level key
			// (yaml.v3 rejects it → LoadPath errors → every fab command bricks).
			// Leave `old` in place (the parker handles it) and note the skip.
			report = append(report, fmt.Sprintf("skipped rename %q → %q (target key already set; left %q in place)", old, newKey, old))
			continue
		}
		lines[i] = newKey + ln[len(old):]
		delete(liveKeys, old)
		liveKeys[newKey] = true
		report = append(report, fmt.Sprintf("carried rename %q → %q (value preserved)", old, newKey))
	}
	return strings.Join(lines, "\n"), liveKeys, report
}

// extractUnknownLiveKeys removes live top-level keys not documented by any registry
// row from the preamble and returns them as key→serialized-block for parking. The
// serialized block is the key's own top-level line and any following indented lines
// (its value), captured verbatim for the parked comment.
func extractUnknownLiveKeys(preamble string, fields []configref.Field) (string, map[string]string) {
	known, _ := registryTopLevelKeys(fields)
	lines := strings.Split(preamble, "\n")

	parked := map[string]string{}
	var kept []string
	for i := 0; i < len(lines); {
		m := topLevelKeyRe.FindStringSubmatch(lines[i])
		if m == nil {
			kept = append(kept, lines[i])
			i++
			continue
		}
		key := m[1]
		// Capture the block: the key line plus any following indented/blank lines
		// (and interior column-0 comments) that belong to its value, up to the next
		// top-level key or a column-0 line that starts fresh content.
		start := i
		i++
		i = advancePastBlock(lines, i)
		block := lines[start:i]
		if known[key] {
			kept = append(kept, block...)
			continue
		}
		parked[key] = strings.TrimRight(strings.Join(block, "\n"), "\n")
	}
	return strings.Join(kept, "\n"), parked
}

// advancePastBlock returns the index of the first line AFTER the value block that
// began before `start`. An indented line (space/tab lead) or a blank line is part of
// the block. A column-0 `#` comment is INTERIOR to the block — and thus part of it —
// only when indented block content resumes before the next top-level key; a trailing
// column-0 comment that is NOT followed by more indented content ends the block (it
// belongs to whatever comes next, typically the following key's own comment). This
// keeps a user's live block with an interior column-0 comment intact rather than
// splitting it and orphaning the indented lines below the comment (which would
// render unparseable YAML). See SF-c.
func advancePastBlock(lines []string, start int) int {
	i := start
	for i < len(lines) {
		ln := lines[i]
		if ln == "" || ln[0] == ' ' || ln[0] == '\t' {
			i++
			continue
		}
		if strings.HasPrefix(ln, "#") {
			// A column-0 comment continues the block only if indented content
			// resumes before the next top-level key. Look ahead.
			if commentPrecedesBlockContinuation(lines, i+1) {
				i++
				continue
			}
			return i
		}
		// A column-0 non-comment, non-blank line — a new top-level key or other
		// fresh content. The block ends here.
		return i
	}
	return i
}

// commentPrecedesBlockContinuation reports whether, scanning forward from idx over a
// run of column-0 comments and blank lines, the first non-comment/non-blank line is
// an INDENTED line (block continuation) rather than a fresh column-0 key. Used to
// decide whether a column-0 comment is interior to the current block.
func commentPrecedesBlockContinuation(lines []string, idx int) bool {
	for i := idx; i < len(lines); i++ {
		ln := lines[i]
		if ln == "" || strings.HasPrefix(ln, "#") {
			continue
		}
		return ln[0] == ' ' || ln[0] == '\t'
	}
	return false
}

// renderFence builds the managed fence: the BEGIN anchor (stamped with kitVersion),
// the fixed header comment, the fully-commented ShortSegment of each advertise:true
// field NOT already live above the fence, then the END anchor. The fence renders
// the file-bound SHORT form (260809-wll4 R6/R7: short description + scope tag +
// explain pointer), never the long explain prose. Fields are emitted in registry
// order for byte-stability.
func renderFence(target Target, fields []configref.Field, liveKeys map[string]bool, kitVersion string) string {
	return renderFenceWithSegments(target, fields, kitVersion, func(_ int, f configref.Field) string {
		if liveKeys[topLevel(f.Key)] {
			return ""
		}
		return f.ShortSegment
	})
}

func renderFenceWithSegments(target Target, fields []configref.Field, kitVersion string, segmentFor func(int, configref.Field) string) string {
	var b strings.Builder
	b.WriteString(anchorLine(fmt.Sprintf(beginPrefix, kitVersion)))
	b.WriteString("\n")
	b.WriteString(target.FencePreamble)

	for i, f := range fields {
		if !target.includesFenceField(f) || f.Segment == "" {
			continue
		}
		segment := segmentFor(i, f)
		if segment == "" {
			continue // already overridden above the fence — not re-advertised
		}
		b.WriteString("\n#\n")
		b.WriteString(CommentOutSegment(segment))
	}

	b.WriteString("\n")
	b.WriteString(anchorLine(endPrefix))
	return b.String()
}

// anchorLine pads prefix with `-` out to fenceWidth (never truncating a longer
// prefix). Exactly one trailing space in the prefix separates the label from the
// dash rule, matching the worked example.
func anchorLine(prefix string) string {
	if len(prefix) >= fenceWidth {
		return prefix
	}
	return prefix + strings.Repeat("-", fenceWidth-len(prefix))
}

// CommentOutSegment ensures every non-blank line of a rendered reference Segment is
// a comment, so the scaffolded block is fully inert (including its parent keys —
// the exact thing the old masher collapsed to `agent: null`). Blank lines stay
// blank.
//
// COLUMN-0 RULE. The skip test is on the RAW line (`#` at column 0), not on the
// trimmed line. A segment carries a TWO-LEVEL comment scheme:
//
//   - FENCE level — the segment's own prose lines, whose `#` already sits at
//     column 0. These are already comments at the fence's own level and are left
//     as-is.
//   - CONTENT level — YAML the segment deliberately ships commented INSIDE a live
//     block (the agent block's `  # profiles:` / `  #   review: { provider: codex }`
//     example lines), where the `#` is INDENTED because it belongs to the
//     YAML structure, not to the fence.
//
// A trimmed test would skip the content-level lines, leaving their `#` at column
// 2/4 while every live line got its marker at column 0 — the ragged fence users
// see in a generated config reference. Prefixing them instead puts every fence
// line's marker at column 0 and makes the reverse operation exact: stripping the
// leading `# ` from every line of a block restores the segment BYTE-EXACTLY —
// which is precisely what `configref.providersSegment`'s prose instructs the
// reader to do.
//
// It is the SINGLE comment-out helper shared by the managed fence (here) and
// `fab config init --system` (cmd/fab), which imports it — there is no second copy.
func CommentOutSegment(segment string) string {
	lines := strings.Split(segment, "\n")
	for i, ln := range lines {
		if strings.TrimSpace(ln) == "" {
			continue
		}
		if strings.HasPrefix(ln, "#") {
			continue // fence-level prose — its marker is already at column 0
		}
		lines[i] = "# " + ln
	}
	return strings.Join(lines, "\n")
}

// assemble stitches the final file: preamble, a blank separator, the fence, then the
// parked-removal blocks (existing first — preserved order — then newly parked, sorted
// for determinism). The result always ends in exactly one trailing newline.
func assemble(preamble, fence string, existingParked []string, newlyParked map[string]string) string {
	var b strings.Builder

	pre := strings.TrimRight(preamble, "\n")
	if pre != "" {
		b.WriteString(pre)
		b.WriteString("\n\n")
	}
	b.WriteString(fence)

	allParked := append([]string{}, existingParked...)
	for _, k := range sortedKeys(newlyParked) {
		allParked = append(allParked, renderParkedBlock(k, newlyParked[k]))
	}
	for _, blk := range allParked {
		b.WriteString("\n\n")
		b.WriteString(strings.TrimRight(blk, "\n"))
	}

	b.WriteString("\n")
	return b.String()
}

// renderParkedBlock formats one parked removal: the header line, then the removed
// key's serialized block indented under a `#   ` comment prefix.
func renderParkedBlock(key, block string) string {
	var b strings.Builder
	b.WriteString("# removed in " + parkedVersionPlaceholder + " (parked by fab config upgrade — delete when done):")
	for _, ln := range strings.Split(block, "\n") {
		b.WriteString("\n#   ")
		b.WriteString(ln)
	}
	return b.String()
}

// parkedVersionPlaceholder is the version stamped into a parked-removal header.
// The registry does not carry the release in which a field was removed (the row is
// simply gone), so the header uses a stable "an earlier release" phrasing rather
// than inventing a version. This keeps the block byte-stable across runs (a
// version-specific header would churn on every kit bump). The literal is a named
// constant so a future registry that DOES record removal versions can key on it.
const parkedVersionPlaceholder = "an earlier release"

// bHygieneReport is the presence=intent advisory (decision 2 / R2.1): it lists live
// top-level keys whose value equals the current built-in default — a field the user
// could remove and inherit the same value. It is ADVISORY ONLY: the upgrader never
// removes or mutates a live field, so a false positive here costs nothing. Only
// registry fields carrying a non-nil Default (today `providers`, `agent.profiles`, and the two depth knobs)
// can match; every other field has no built-in default, so a live value there is
// always a genuine override and is never flagged.
//
// The comparison parses the live preamble into a generic map and deep-compares each
// live subtree against the registry's own materialized projection
// (configref.DefaultsMap, already normalized to the generic shape). Both sides are
// therefore decoded values, which genericEqual compares tolerantly. A parse failure
// yields no findings (advisory — never fatal). Reports are stable-ordered by
// registry order.
func bHygieneReport(preamble string, fields []configref.Field) []string {
	var live map[string]any
	if err := yaml.Unmarshal([]byte(preamble), &live); err != nil || live == nil {
		return nil // can't parse (or nothing live) → no advisory (never fatal)
	}
	// The built-in side is the registry's own projection — the same materialized
	// defaults tier the read model merges (configref.DefaultsMap), so there is no
	// second nesting/merging implementation to drift. A projection failure yields no
	// advisory, matching the parse-failure path above (B-hygiene is never fatal).
	defaults, err := configref.DefaultsMap()
	if err != nil {
		return nil
	}
	seen := map[string]bool{}
	var report []string
	for _, f := range fields {
		if f.Default == nil {
			continue // no built-in default → a live value is always a real override
		}
		top := topLevel(f.Key)
		if seen[top] {
			continue // one advisory per top-level key (agent.profiles → agent)
		}
		liveVal, ok := live[top]
		if !ok {
			continue // not a live key → nothing to compare
		}
		seen[top] = true
		def := defaults[top]
		if def == nil {
			continue
		}
		if genericEqual(liveVal, def) {
			report = append(report, fmt.Sprintf("field %q equals the current default — you can remove it to inherit (kept as-is: presence=intent)", top))
		}
	}
	return report
}

// genericEqual deep-compares two decoded-YAML/JSON generic values (maps, slices,
// scalars) for structural equality, tolerating the map[string]any vs map[any]any
// shape difference yaml.v3 can produce. Scalars compare by their formatted string so
// an int from JSON and an int from YAML (or an untyped number) compare equal without
// per-numeric-type branching — config values are strings/lists/maps, so no precision
// concern applies.
func genericEqual(a, b any) bool {
	am, aok := asGenericMap(a)
	bm, bok := asGenericMap(b)
	if aok || bok {
		if !aok || !bok || len(am) != len(bm) {
			return false
		}
		for k, av := range am {
			bv, ok := bm[k]
			if !ok || !genericEqual(av, bv) {
				return false
			}
		}
		return true
	}
	as, aok := a.([]any)
	bs, bok := b.([]any)
	if aok || bok {
		if !aok || !bok || len(as) != len(bs) {
			return false
		}
		for i := range as {
			if !genericEqual(as[i], bs[i]) {
				return false
			}
		}
		return true
	}
	return fmt.Sprintf("%v", a) == fmt.Sprintf("%v", b)
}

// asGenericMap coerces a decoded YAML/JSON value to map[string]any when it is a
// mapping (handling both map[string]any and map[any]any).
func asGenericMap(v any) (map[string]any, bool) {
	switch m := v.(type) {
	case map[string]any:
		return m, true
	case map[any]any:
		out := make(map[string]any, len(m))
		for k, val := range m {
			ks, ok := k.(string)
			if !ok {
				return nil, false
			}
			out[ks] = val
		}
		return out, true
	default:
		return nil, false
	}
}

// topLevel collapses a dotted registry key to its top-level YAML key
// ("agent.profiles" → "agent", "project.name" → "project", "source_paths" →
// "source_paths").
func topLevel(key string) string {
	if i := strings.IndexByte(key, '.'); i >= 0 {
		return key[:i]
	}
	return key
}

// sortedKeys returns the map's keys in sorted order (deterministic output).
func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
