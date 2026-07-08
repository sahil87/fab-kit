package internal

import (
	"fmt"
	"os"
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

	// 2. Ensure cached
	_, err = EnsureCached(latest)
	if err != nil {
		return err
	}

	// 3. Create/update config.yaml with fab_version, at the repo root
	configPath := filepath.Join(repoRoot, "fab", "project", "config.yaml")
	if err := setFabVersion(configPath, latest); err != nil {
		return fmt.Errorf("cannot update config.yaml: %w", err)
	}
	fmt.Printf("Set fab_version: %s in config.yaml\n", latest)

	// 4. Stamp .kit-migration-version to the engine version. This must happen
	// before Sync — otherwise scaffoldDirectories sees the just-written
	// config.yaml and classifies the project as "existing", writing 0.1.0
	// and triggering a spurious migration prompt on every fresh init.
	if err := stampMigrationVersion(repoRoot, latest); err != nil {
		return err
	}

	// 5. Run sync — the kit version is passed explicitly; systemVersion (the
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

// setFabVersion creates or updates config.yaml with the fab_version field.
//
// It owns exactly one line — the top-level `fab_version:` scalar — and touches
// nothing else. Rather than unmarshalling the whole file into a map and
// re-marshalling (which strips comments, alphabetizes keys, normalizes
// indentation, and collapses comment-only mapping keys to null), it performs a
// targeted line splice: the file is preserved byte-for-byte except the single
// line this function owns. A trailing same-line comment on that line is kept.
//
// Behavior:
//   - file missing        → create it (with parent dirs) containing just
//     `fab_version: <version>`
//   - top-level fab_version present → replace its value in place, preserving any
//     trailing comment
//   - top-level fab_version absent   → append `fab_version: <version>` as the
//     final line, with exactly one trailing newline
//
// "Top-level" means a line whose key begins at column 0 (not indented, not a
// `#` comment).
func setFabVersion(path string, version string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			// New file: write just the fab_version line.
			return os.WriteFile(path, []byte("fab_version: "+version+"\n"), 0644)
		}
		// A genuine read error (permissions, etc.) still fails loudly.
		return fmt.Errorf("cannot read existing config.yaml: %w", err)
	}

	lines := strings.Split(string(content), "\n")
	for i, line := range lines {
		if rest, ok := topLevelFabVersionValue(line); ok {
			// Replace only the value token, preserving any trailing comment.
			lines[i] = "fab_version: " + version + rest
			return os.WriteFile(path, []byte(strings.Join(lines, "\n")), 0644)
		}
	}

	// No top-level fab_version line: append one, ensuring exactly one trailing newline.
	out := string(content)
	if out != "" && !strings.HasSuffix(out, "\n") {
		out += "\n"
	}
	out += "fab_version: " + version + "\n"
	return os.WriteFile(path, []byte(out), 0644)
}

// topLevelFabVersionValue reports whether line is a top-level `fab_version:`
// entry — its key begins at column 0 (not indented, not commented). When it is,
// ok is true and rest is the portion of the line to keep after the replacement
// value: an empty string, or a preserved trailing comment (with its original
// leading whitespace), e.g. "  # pinned" for `fab_version: 1.2.3  # pinned`.
func topLevelFabVersionValue(line string) (rest string, ok bool) {
	const key = "fab_version:"
	if !strings.HasPrefix(line, key) {
		return "", false
	}
	after := line[len(key):]
	// Preserve a trailing `#` comment verbatim, including the whitespace that
	// separates it from the value.
	if idx := strings.IndexByte(after, '#'); idx >= 0 {
		ws := len(strings.TrimRight(after[:idx], " \t"))
		return after[ws:], true // whitespace run before '#' plus the comment
	}
	return "", true
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
