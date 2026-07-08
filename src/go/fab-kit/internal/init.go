package internal

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Init handles `fab init` — scaffolds a new fab project or updates an existing one.
// systemVersion is the embedded version of the fab-kit binary, threaded into
// Sync so the version guard compares against the real binary version.
func Init(systemVersion string) error {
	// 0. Precondition: must be inside a git repository — checked BEFORE any
	// download or config write, so a failed init leaves no stale artifacts
	// behind (sync would fail on this anyway, but only after downloading the
	// release and writing config.yaml). The resolved root anchors all init
	// writes, so `fab init` from a subdirectory targets the repo root —
	// matching Sync, which resolves the same root internally.
	repoRoot, err := gitRepoRoot()
	if err != nil {
		return fmt.Errorf("fab init requires a git repository — run 'git init' first (%w)", err)
	}

	// 1. Resolve latest version
	fmt.Println("Resolving latest fab-kit version...")
	latest, err := LatestVersion()
	if err != nil {
		return fmt.Errorf("cannot resolve latest version: %w", err)
	}
	fmt.Printf("Latest version: %s\n", latest)

	// 2. Ensure cached — the returned path is the pinned fab-go binary used to
	// generate config.yaml from the registry (single-writer discipline).
	fabGoBin, err := EnsureCached(latest)
	if err != nil {
		return err
	}

	// 3. Stamp fab/.fab-version (the plain-text sibling that replaced the
	// config.yaml fab_version: key — 260708-j0qm). config.yaml is no longer
	// version-stamped; fab config upgrade is its only writer going forward.
	if err := stampFabVersion(repoRoot, latest); err != nil {
		return err
	}
	fmt.Printf("Set fab version %s in fab/.fab-version\n", latest)

	// 4. Generate config.yaml from the registry via the pinned fab-go
	// (`fab config init --project`) — the scaffold config.yaml was retired. On a
	// fab-go that predates the subcommand, fall back to a minimal embedded stub so
	// a fresh repo never fails preflight for lack of a config.yaml (fail-open).
	configPath := filepath.Join(repoRoot, "fab", "project", "config.yaml")
	if err := generateProjectConfig(fabGoBin, repoRoot, configPath); err != nil {
		return err
	}

	// 5. Stamp .kit-migration-version to the engine version. This must happen
	// before Sync — otherwise scaffoldDirectories sees the just-written
	// config.yaml and classifies the project as "existing", writing 0.1.0
	// and triggering a spurious migration prompt on every fresh init.
	if err := stampMigrationVersion(repoRoot, latest); err != nil {
		return err
	}

	// 6. Run sync — the kit version is passed explicitly; systemVersion (the
	// embedded binary version) feeds the version guard (F22).
	fmt.Println("Setting up project...")
	if err := runSync(systemVersion, latest, false, false); err != nil {
		return fmt.Errorf("sync failed: %w", err)
	}

	fmt.Printf("\nfab initialized (v%s). Run /fab-setup in your AI agent to configure.\n", latest)
	return nil
}

// stampMigrationVersion writes fab/.kit-migration-version to the given version,
// creating fab/ if needed. Used by Init to mark a freshly-created project as
// already at the engine version, so scaffoldDirectories doesn't classify it as
// a legacy project and write 0.1.0.
func stampMigrationVersion(repoRoot, version string) error {
	path := filepath.Join(repoRoot, "fab", ".kit-migration-version")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cannot create fab/ directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(version+"\n"), 0644); err != nil {
		return fmt.Errorf("cannot write .kit-migration-version: %w", err)
	}
	return nil
}

// stampFabVersion writes fab/.fab-version to the given version, creating fab/ if
// needed. This is the sibling of stampMigrationVersion (same plain-text,
// one-line-plus-newline shape) that replaced the old config.yaml fab_version:
// stamp (260708-j0qm): deployed-kit version vs migration baseline are kept
// distinct, and config.yaml is no longer written by init/upgrade — fab config
// upgrade is its only writer going forward.
func stampFabVersion(repoRoot, version string) error {
	path := filepath.Join(repoRoot, dotFabVersionRelPath)
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("cannot create fab/ directory: %w", err)
	}
	if err := os.WriteFile(path, []byte(version+"\n"), 0644); err != nil {
		return fmt.Errorf("cannot write fab/.fab-version: %w", err)
	}
	return nil
}

// generateProjectConfig writes the initial fab/project/config.yaml by shelling out
// to the pinned fab-go binary's `fab config init --project` (which generates it
// from the registry — the scaffold config.yaml was retired). fab-kit performs the
// mechanical, non-interactive detection of the A-class identity seed (project name
// from the repo folder, source_paths from common on-disk directories, test_paths
// from ecosystem marker files) and passes it as `--name`/`--source-path`/
// `--test-path` flags (R5.3), so the generated config carries LIVE identity fields
// rather than an empty header+fence. /fab-setup's Config Create Mode later refines
// this interactively (it asks the user and can override any detected value).
//
// FAIL-OPEN: if the pinned fab-go predates `fab config init --project` (non-zero
// exit / unknown command), fall back to a minimal EMBEDDED STUB config.yaml — a
// tiny bounded copy of the A-class identity fields, carrying the same detected seed
// — rather than a printed instruction, so a fresh repo never fails preflight for
// lack of a config.yaml (user-confirmed decision). A pre-existing config.yaml is
// never overwritten (`fab config init --project` refuses; the stub path checks too).
func generateProjectConfig(fabGoBin, repoRoot, configPath string) error {
	if _, err := os.Stat(configPath); err == nil {
		// Already present (e.g. re-run over an existing repo) — leave it untouched.
		return nil
	}

	seed := detectProjectSeed(repoRoot)

	args := []string{"config", "init", "--project"}
	if seed.name != "" {
		args = append(args, "--name", seed.name)
	}
	for _, p := range seed.sourcePaths {
		args = append(args, "--source-path", p)
	}
	for _, p := range seed.testPaths {
		args = append(args, "--test-path", p)
	}

	cmd := exec.Command(fabGoBin, args...)
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	// The shell-out is "successful generation" only when it exits 0 AND actually
	// wrote the file — a fab-go that predates the subcommand may exit 0 for an
	// unknown flag on some cobra versions, or exit non-zero. Either way, if no
	// config.yaml landed, fall open to the embedded stub so init never bricks.
	if err == nil {
		if _, statErr := os.Stat(configPath); statErr == nil {
			fmt.Println("Generated fab/project/config.yaml from the config registry")
			return nil
		}
	}
	fmt.Printf("Note: installed fab-go could not generate config.yaml (%s); writing a minimal stub. Run `fab config upgrade` after upgrading to refresh it.\n", strings.TrimSpace(string(out)))
	return writeStubConfig(configPath, seed)
}

// projectSeed is the mechanically-detected A-class identity seed fab-kit passes to
// `fab config init --project`. Every field may be empty (a field with no confident
// detection is left to the fence to advertise, and refined later by /fab-setup).
type projectSeed struct {
	name        string
	sourcePaths []string
	testPaths   []string
}

// detectProjectSeed derives the identity seed non-interactively from the repo on
// disk. It is deliberately conservative — it only emits a value it can infer
// mechanically, leaving anything ambiguous empty for /fab-setup to fill:
//   - name: the repo folder basename (the same signal the worktree/branch naming
//     uses); an empty/"/" basename yields "".
//   - source_paths: common implementation directories that actually exist (src/).
//   - test_paths: the ecosystem test-glob for detected marker files, mirroring the
//     /fab-setup Config Create Mode marker table. Multi-marker repos union their
//     pattern sets (deduped, stable order).
func detectProjectSeed(repoRoot string) projectSeed {
	var seed projectSeed

	if base := filepath.Base(repoRoot); base != "" && base != "." && base != string(filepath.Separator) {
		seed.name = base
	}

	for _, dir := range []string{"src"} {
		if fi, err := os.Stat(filepath.Join(repoRoot, dir)); err == nil && fi.IsDir() {
			seed.sourcePaths = append(seed.sourcePaths, dir+"/")
		}
	}

	seed.testPaths = detectTestPaths(repoRoot)
	return seed
}

// testMarker pairs an on-disk marker file with the anchored test_paths patterns it
// implies. The anchoring (suffix/prefix/infix/source-root) is what makes the
// test/impl classification reliable — a bare substring like `**/*test*` miscounts
// production code (attestation.go, latest.go). Mirrors the /fab-setup Config Create
// Mode marker table so the Go detection and the skill agree.
var testMarkers = []struct {
	markers  []string
	patterns []string
}{
	{markers: []string{"go.mod"}, patterns: []string{"**/*_test.go"}},
	{markers: []string{"pytest.ini", "pyproject.toml", "setup.cfg"}, patterns: []string{"**/test_*.py", "**/*_test.py"}},
	{markers: []string{"pom.xml", "build.gradle"}, patterns: []string{"**/src/test/**"}},
	// Rust (Cargo.toml) uses inline #[cfg(test)] tests — not glob-addressable, so no
	// pattern is emitted (left to the fence, matching the skill's "leave empty" row).
}

// detectTestPaths reads the repo's root marker files and returns the union of the
// anchored test-glob pattern sets they imply (deduped, first-seen order). Empty when
// no recognized marker is present (the impact breakdown then collapses to a single
// total — today's behavior). JS/TS detection is intentionally omitted here: it
// requires parsing package.json deps or globbing for *.spec/*.test files, which is
// /fab-setup's interactive job — the Go layer stays to unambiguous single-file
// markers.
func detectTestPaths(repoRoot string) []string {
	var out []string
	seen := map[string]bool{}
	for _, tm := range testMarkers {
		matched := false
		for _, m := range tm.markers {
			if _, err := os.Stat(filepath.Join(repoRoot, m)); err == nil {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		for _, p := range tm.patterns {
			if !seen[p] {
				seen[p] = true
				out = append(out, p)
			}
		}
	}
	return out
}

// stubConfigHeader is the fixed banner of the minimal embedded fallback config.yaml,
// written only when the pinned fab-go predates `fab config init --project`. The stub
// is deliberately spare (no managed fence): its sole job is to exist so preflight
// passes; the next `fab upgrade-repo` runs `fab config upgrade` and materializes the
// full fence. /fab-setup refines the identity fields.
const stubConfigHeader = `# fab/project/config.yaml — minimal stub written by ` + "`fab init`" + ` because the
# installed fab-go predates registry-based generation. Run ` + "`fab config upgrade`" + `
# (or ` + "`fab upgrade-repo`" + `) after upgrading to materialize the full reference fence.`

// renderStubConfig builds the embedded stub from the detected seed, so the stub
// (like the registry-generated file) carries the detected identity fields live
// rather than a hardcoded placeholder. Missing seed values fall back to the standard
// placeholders (name/description) or are omitted (source_paths/test_paths) so the
// document always parses and always carries the required project.name/description.
func renderStubConfig(seed projectSeed) string {
	name := seed.name
	if name == "" {
		name = "My Project"
	}
	var b strings.Builder
	b.WriteString(stubConfigHeader)
	b.WriteString("\nproject:\n")
	fmt.Fprintf(&b, "  name: %q\n", name)
	b.WriteString("  description: \"One-line project description\"\n")

	src := seed.sourcePaths
	if len(src) == 0 {
		src = []string{"src/"}
	}
	b.WriteString("\nsource_paths:\n")
	for _, p := range src {
		fmt.Fprintf(&b, "  - %s\n", p)
	}

	if len(seed.testPaths) > 0 {
		b.WriteString("\ntest_paths:\n")
		for _, p := range seed.testPaths {
			fmt.Fprintf(&b, "  - %q\n", p)
		}
	}
	return b.String()
}

// writeStubConfig writes the embedded stub (carrying the detected seed), creating
// fab/project/ as needed. It refuses to overwrite an existing config.yaml (defensive
// — the caller already checked, but the stub path must never clobber user data).
func writeStubConfig(configPath string, seed projectSeed) error {
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0755); err != nil {
		return fmt.Errorf("cannot create fab/project/ directory: %w", err)
	}
	if err := os.WriteFile(configPath, []byte(renderStubConfig(seed)), 0644); err != nil {
		return fmt.Errorf("cannot write stub config.yaml: %w", err)
	}
	return nil
}

// copyDir copies src directory to dst, creating dst if needed.
// Existing files in dst are overwritten; existing files not in src are left alone.
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		destPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, 0755)
		}

		// Skip .gitkeep in bin/ — we want bin/ to stay clean
		if strings.HasSuffix(relPath, "bin/.gitkeep") {
			// Still create the directory
			return os.MkdirAll(filepath.Dir(destPath), 0755)
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destPath, data, info.Mode())
	})
}
