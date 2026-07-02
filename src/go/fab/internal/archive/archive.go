package archive

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/atomicfile"
	"github.com/sahil87/fab-kit/src/go/fab/internal/backlog"
	"github.com/sahil87/fab-kit/src/go/fab/internal/change"
	"github.com/sahil87/fab-kit/src/go/fab/internal/dispatch"
	"github.com/sahil87/fab-kit/src/go/fab/internal/intake"
	"github.com/sahil87/fab-kit/src/go/fab/internal/lines"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
)

// ErrAlreadyArchived is returned (wrapped with the destination path) when the
// archive destination already exists. Callers can detect it via errors.Is to
// treat re-archiving as an idempotent soft skip.
var ErrAlreadyArchived = errors.New("change already archived")

// ArchiveResult holds the YAML output for archive operations.
type ArchiveResult struct {
	Action  string
	Name    string
	Move    string
	Index   string
	Pointer string
	Backlog string
}

// RestoreResult holds the YAML output for restore operations.
type RestoreResult struct {
	Action  string
	Name    string
	Move    string
	Index   string
	Pointer string
}

// parseDateBucket extracts yyyy and mm from a YYMMDD-prefixed folder name.
func parseDateBucket(name string) (string, string, error) {
	if len(name) < 6 {
		return "", "", fmt.Errorf("invalid folder name '%s': expected YYMMDD prefix", name)
	}
	for _, c := range name[:6] {
		if c < '0' || c > '9' {
			return "", "", fmt.Errorf("invalid folder name '%s': expected YYMMDD prefix", name)
		}
	}
	yy := name[0:2]
	mm := name[2:4]
	return "20" + yy, mm, nil
}

// Archive moves a change to the archive directory. When description is empty,
// it is derived mechanically from the change's intake title (with a humanized
// slug fallback). Archive stays pure — it performs only move/index/pointer and
// has no backlog dependency; ArchiveWithBacklog orchestrates the backlog mark.
func Archive(fabRoot, changeArg, description string) (*ArchiveResult, error) {
	if changeArg == "" {
		return nil, fmt.Errorf("<change> argument is required for archive")
	}

	folder, err := resolve.ToFolder(fabRoot, changeArg)
	if err != nil {
		// Only a genuine not-found (the name matches no active change) may mean
		// "already archived" — that is the idempotent re-archive soft skip
		// (exit 0 at the cmd layer). An ambiguous name is a real user error and
		// MUST surface as-is, not be silently soft-skipped by guessing against
		// the archive (jznd (d)). Ambiguous or absent archive matches fall
		// through to the original error.
		if errors.Is(err, resolve.ErrNotFound) {
			if _, archivedDir, archErr := resolveArchive(fabRoot, changeArg); archErr == nil {
				return nil, fmt.Errorf("%w: %s", ErrAlreadyArchived, archivedDir)
			}
		}
		return nil, err
	}

	// Derive the index description from the intake title before the folder is
	// moved out of fab/changes/ (intake.md is still in the source folder here).
	if description == "" {
		description = intake.DescriptionFor(fabRoot, folder)
	}

	changesDir := filepath.Join(fabRoot, "changes")
	archiveDir := filepath.Join(changesDir, "archive")
	changeDir := filepath.Join(changesDir, folder)

	// Capture whether the archived change is the active one BEFORE the folder
	// moves: resolve validates the pointer target (mz4q F08), so a post-rename
	// resolution would see a dangling pointer and never report the archived
	// change as active — the pointer would silently stay behind. Read the link
	// directly; exact folder match only.
	repoRoot := filepath.Dir(fabRoot)
	pointerWasActive := false
	if target, err := os.Readlink(filepath.Join(repoRoot, ".fab-status.yaml")); err == nil {
		pointerWasActive = resolve.ExtractFolderFromSymlink(target) == folder
	}

	// 1. Move to archive/yyyy/mm/
	bucketYear, bucketMonth, err := parseDateBucket(folder)
	if err != nil {
		return nil, err
	}
	destDir := filepath.Join(archiveDir, bucketYear, bucketMonth)
	os.MkdirAll(destDir, 0755)
	destPath := filepath.Join(destDir, folder)
	if _, err := os.Stat(destPath); err == nil {
		return nil, fmt.Errorf("%w: %s", ErrAlreadyArchived, destPath)
	}
	if err := os.Rename(changeDir, destPath); err != nil {
		return nil, fmt.Errorf("move to archive: %w", err)
	}

	// 1b. Delete the change's .fab-dispatch/{id}/ state dir — dispatch artifacts
	// are transient comms, not history, so they are removed on archive and NOT
	// recreated on restore (one of the two deterministic cleanup paths; the
	// other is `fab dispatch clean`, no automatic GC). Best-effort: an absent
	// dir is a no-op, and a removal error must not undo the completed move.
	if id := resolve.ExtractID(folder); id != "" {
		_ = os.RemoveAll(dispatch.DirFor(repoRoot, id))
	}

	// 2. Update index, then backfill unindexed entries from the just-written
	// content (no post-write re-read). A failed index write must not report
	// "updated" — the YAML result carries the honest status and the error
	// propagates alongside the (non-nil) result, since the move has already
	// happened.
	indexFile := filepath.Join(archiveDir, "index.md")
	indexStatus, indexContent, indexErr := updateIndex(indexFile, folder, description)
	if indexErr == nil {
		indexErr = backfillIndex(archiveDir, indexFile, indexContent)
	}
	if indexErr != nil {
		indexStatus = "failed"
	}

	// 3. Clear pointer if active (captured pre-rename above)
	pointerStatus := "skipped"
	if pointerWasActive {
		change.SwitchNone(fabRoot)
		pointerStatus = "cleared"
	}

	result := &ArchiveResult{
		Action:  "archive",
		Name:    folder,
		Move:    "moved",
		Index:   indexStatus,
		Pointer: pointerStatus,
	}
	if indexErr != nil {
		return result, fmt.Errorf("change moved to archive but index update failed: %w", indexErr)
	}
	return result, nil
}

// ArchiveWithBacklog runs Archive, then marks the originating backlog item
// done. The 4-char change ID is the backlog ID when the change came from
// backlog, so the mark is a deterministic exact-ID match. Archive's error
// (including ErrAlreadyArchived) propagates unchanged; the backlog status is
// recorded on result.Backlog. A missing backlog file is a silent no-op
// (MarkDone returns "not_found", nil), but a genuine read/write failure
// (permissions, disk full) is propagated so callers don't report a misleading
// success. The archive move has already happened by then, so result is
// returned alongside the error. A partial Archive result (move succeeded,
// index update failed) still gets its backlog mark — the move is
// irreversible and a re-run soft-skips, so skipping the mark would strand
// the item — with both errors joined. Those mark-failure and partial-archive
// cases are the only ones where a non-nil result is returned alongside an
// error; when Archive itself errors (including ErrAlreadyArchived), the
// result is nil.
//
// On ErrAlreadyArchived the backlog mark is still attempted (best-effort) so a
// re-run recovers a previously-failed mark — MarkDone is idempotent and returns
// "already" when the item was marked before. ErrAlreadyArchived propagates
// unchanged with a nil result, keeping the callers' soft-skip exit semantics
// untouched.
func ArchiveWithBacklog(fabRoot, changeArg, description string) (*ArchiveResult, error) {
	result, archiveErr := Archive(fabRoot, changeArg, description)
	if result == nil {
		if errors.Is(archiveErr, ErrAlreadyArchived) {
			// Re-archive soft skip: still attempt the mark so a re-run recovers
			// a previously-failed one. The folder is re-derived from whichever
			// location the change lives in now — fab/changes/ (destination-
			// exists case) or the archive (the usual soft-skip path, where
			// resolve.ToFolder fails because the source folder is gone). The
			// mark outcome is deliberately not propagated: the soft-skip
			// contract is the caller-visible behavior.
			folder, rerr := resolve.ToFolder(fabRoot, changeArg)
			if rerr != nil {
				folder, _, rerr = resolveArchive(fabRoot, changeArg)
			}
			if rerr == nil {
				_, _ = backlog.MarkDone(backlog.Path(fabRoot), resolve.ExtractID(folder))
			}
		}
		return nil, archiveErr
	}
	id := resolve.ExtractID(result.Name)
	status, markErr := backlog.MarkDone(backlog.Path(fabRoot), id)
	result.Backlog = status
	if markErr != nil {
		markErr = fmt.Errorf("archive succeeded but marking backlog item %q done failed: %w", id, markErr)
	}
	return result, errors.Join(archiveErr, markErr)
}

// Restore moves a change from the archive back to active.
func Restore(fabRoot, changeArg string, doSwitch bool) (*RestoreResult, error) {
	if changeArg == "" {
		return nil, fmt.Errorf("<change> argument is required for restore")
	}

	folder, resolvedDir, err := resolveArchive(fabRoot, changeArg)
	if err != nil {
		return nil, err
	}

	changesDir := filepath.Join(fabRoot, "changes")
	archiveDir := filepath.Join(changesDir, "archive")

	// 1. Move from archive (resolveArchive returns full dir path)
	moveStatus := "restored"
	destPath := filepath.Join(changesDir, folder)
	if _, err := os.Stat(destPath); err == nil {
		moveStatus = "already_in_changes"
	} else {
		if err := os.Rename(resolvedDir, destPath); err != nil {
			return nil, fmt.Errorf("restore: %w", err)
		}
	}

	// 2. Remove from index. A failed read/rewrite is surfaced as
	// `index: failed` with the error returned alongside the (non-nil)
	// result — the move has already happened, mirroring ArchiveWithBacklog's
	// partial-success contract.
	indexFile := filepath.Join(archiveDir, "index.md")
	indexStatus, indexErr := removeFromIndex(indexFile, folder)

	// 3. Optionally switch. A failed activation is surfaced as
	// `pointer: failed` — rendering it as "skipped" would report the
	// requested --switch as not requested.
	pointerStatus := "skipped"
	if doSwitch {
		_, err := change.Switch(fabRoot, folder)
		if err == nil {
			pointerStatus = "switched"
		} else {
			pointerStatus = "failed"
		}
	}

	result := &RestoreResult{
		Action:  "restore",
		Name:    folder,
		Move:    moveStatus,
		Index:   indexStatus,
		Pointer: pointerStatus,
	}
	if indexErr != nil {
		return result, fmt.Errorf("change restored but index update failed: %w", indexErr)
	}
	return result, nil
}

// IsArchived reports whether changeArg unambiguously matches an archived
// change (flat or nested entry). Used by batch archive to route
// genuinely-archived names to the soft-skip path instead of the
// unresolvable-name warning.
func IsArchived(fabRoot, changeArg string) bool {
	_, _, err := resolveArchive(fabRoot, changeArg)
	return err == nil
}

// List returns archived change folder names from both flat and nested entries.
func List(fabRoot string) ([]string, error) {
	archiveDir := filepath.Join(fabRoot, "changes", "archive")
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		return nil, nil
	}

	topLevel, err := os.ReadDir(archiveDir)
	if err != nil {
		return nil, err
	}

	var results []string

	// Flat entries: archive/{name}/ (skip year directories)
	for _, e := range topLevel {
		if e.IsDir() && !isYearDir(e.Name()) {
			results = append(results, e.Name())
		}
	}

	// Nested entries: archive/yyyy/mm/{name}/
	for _, yearEntry := range topLevel {
		if !yearEntry.IsDir() || !isYearDir(yearEntry.Name()) {
			continue
		}
		yearDir := filepath.Join(archiveDir, yearEntry.Name())
		monthEntries, _ := os.ReadDir(yearDir)
		for _, monthEntry := range monthEntries {
			if !monthEntry.IsDir() {
				continue
			}
			monthDir := filepath.Join(yearDir, monthEntry.Name())
			changeEntries, _ := os.ReadDir(monthDir)
			for _, ce := range changeEntries {
				if ce.IsDir() {
					results = append(results, ce.Name())
				}
			}
		}
	}

	return results, nil
}

// FormatArchiveYAML formats an ArchiveResult.
func FormatArchiveYAML(r *ArchiveResult) string {
	return fmt.Sprintf("action: %s\nname: %s\nmove: %s\nindex: %s\npointer: %s\nbacklog: %s",
		r.Action, r.Name, r.Move, r.Index, r.Pointer, r.Backlog)
}

// FormatRestoreYAML formats a RestoreResult.
func FormatRestoreYAML(r *RestoreResult) string {
	return fmt.Sprintf("action: %s\nname: %s\nmove: %s\nindex: %s\npointer: %s",
		r.Action, r.Name, r.Move, r.Index, r.Pointer)
}

// resolveArchive returns (folderName, fullDirPath, error).
// Scans both flat entries (archive/{name}/) and nested entries (archive/yyyy/mm/{name}/).
func resolveArchive(fabRoot, override string) (string, string, error) {
	if override == "" {
		return "", "", fmt.Errorf("<change> argument is required for restore")
	}

	archiveDir := filepath.Join(fabRoot, "changes", "archive")
	if _, err := os.Stat(archiveDir); os.IsNotExist(err) {
		return "", "", fmt.Errorf("No archive folder found.")
	}

	type entry struct {
		name string
		dir  string
	}
	var entries []entry

	// Flat entries: archive/{name}/ (skip 4-digit year directories)
	topLevel, _ := os.ReadDir(archiveDir)
	for _, e := range topLevel {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		if isYearDir(name) {
			continue
		}
		entries = append(entries, entry{name, filepath.Join(archiveDir, name)})
	}

	// Nested entries: archive/yyyy/mm/{name}/
	for _, yearEntry := range topLevel {
		if !yearEntry.IsDir() || !isYearDir(yearEntry.Name()) {
			continue
		}
		yearDir := filepath.Join(archiveDir, yearEntry.Name())
		monthEntries, _ := os.ReadDir(yearDir)
		for _, monthEntry := range monthEntries {
			if !monthEntry.IsDir() {
				continue
			}
			monthDir := filepath.Join(yearDir, monthEntry.Name())
			changeEntries, _ := os.ReadDir(monthDir)
			for _, ce := range changeEntries {
				if ce.IsDir() {
					entries = append(entries, entry{ce.Name(), filepath.Join(monthDir, ce.Name())})
				}
			}
		}
	}

	if len(entries) == 0 {
		return "", "", fmt.Errorf("No archived changes found.")
	}

	overrideLower := strings.ToLower(override)

	// Exact match
	for _, e := range entries {
		if strings.ToLower(e.name) == overrideLower {
			return e.name, e.dir, nil
		}
	}

	// Substring match
	var partials []entry
	for _, e := range entries {
		if strings.Contains(strings.ToLower(e.name), overrideLower) {
			partials = append(partials, e)
		}
	}

	if len(partials) == 1 {
		return partials[0].name, partials[0].dir, nil
	}
	if len(partials) > 1 {
		names := make([]string, len(partials))
		for i, p := range partials {
			names[i] = p.name
		}
		return "", "", fmt.Errorf("Multiple archives match \"%s\": %s.", override, strings.Join(names, ", "))
	}

	return "", "", fmt.Errorf("No archive matches \"%s\".", override)
}

// isYearDir returns true if the name is a 4-digit year directory.
func isYearDir(name string) bool {
	if len(name) != 4 {
		return false
	}
	for _, c := range name {
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

// updateIndex inserts the new entry below the index header and atomically
// rewrites the file. It returns the index status ("updated"/"created"), the
// rewritten content (consumed by backfillIndex, which must not re-read the
// file), and any read/write error — never an unconditional success status.
func updateIndex(indexFile, folder, description string) (string, string, error) {
	indexStatus := "updated"
	data, err := os.ReadFile(indexFile)
	if err != nil {
		if !os.IsNotExist(err) {
			return "", "", fmt.Errorf("read archive index: %w", err)
		}
		indexStatus = "created"
		data = []byte("# Archive Index\n\n")
	}

	// Normalize description
	description = strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, description)
	description = strings.TrimSpace(description)

	newEntry := fmt.Sprintf("- **%s** — %s", folder, description)

	indexLines := lines.Split(string(data))

	var result []string
	if len(indexLines) >= 2 {
		result = append(result, indexLines[0], indexLines[1])
	} else {
		result = append(result, indexLines[0], "")
	}
	result = append(result, newEntry)
	if len(indexLines) > 2 {
		result = append(result, indexLines[2:]...)
	}

	content := strings.Join(result, "\n")
	if err := atomicfile.WriteFile(indexFile, []byte(content), 0o644); err != nil {
		return "", "", fmt.Errorf("write archive index: %w", err)
	}
	return indexStatus, content, nil
}

// backfillIndex appends entries for archived folders missing from the index.
// indexContent is the content updateIndex just wrote — passed in so the
// function never re-reads the file it derives from. When entries are
// missing, the whole file is rewritten atomically.
func backfillIndex(archiveDir, indexFile, indexContent string) error {
	var missing []string
	backfillEntry := func(name string) {
		marker := fmt.Sprintf("**%s**", name)
		if !strings.Contains(indexContent, marker) {
			missing = append(missing, fmt.Sprintf("- **%s** — (no description — pre-index archive)", name))
		}
	}

	topLevel, _ := os.ReadDir(archiveDir)

	// Flat entries (pre-migration)
	for _, e := range topLevel {
		if e.IsDir() && !isYearDir(e.Name()) {
			backfillEntry(e.Name())
		}
	}

	// Nested entries: archive/yyyy/mm/{name}/
	for _, yearEntry := range topLevel {
		if !yearEntry.IsDir() || !isYearDir(yearEntry.Name()) {
			continue
		}
		yearDir := filepath.Join(archiveDir, yearEntry.Name())
		monthEntries, _ := os.ReadDir(yearDir)
		for _, monthEntry := range monthEntries {
			if !monthEntry.IsDir() {
				continue
			}
			monthDir := filepath.Join(yearDir, monthEntry.Name())
			changeEntries, _ := os.ReadDir(monthDir)
			for _, ce := range changeEntries {
				if ce.IsDir() {
					backfillEntry(ce.Name())
				}
			}
		}
	}

	if len(missing) == 0 {
		return nil
	}

	content := indexContent
	if !strings.HasSuffix(content, "\n") {
		content += "\n"
	}
	content += strings.Join(missing, "\n") + "\n"
	if err := atomicfile.WriteFile(indexFile, []byte(content), 0o644); err != nil {
		return fmt.Errorf("backfill archive index: %w", err)
	}
	return nil
}

// removeFromIndex deletes the index entry for folder and atomically
// rewrites the file. The rewrite is always derived from the complete file —
// this is the one index function that writes back what it read, so a
// partial read would silently delete every entry after the abort point.
// Read/write failures return "failed" with the error; a missing file or
// absent entry stays the benign ("not_found", nil).
func removeFromIndex(indexFile, folder string) (string, error) {
	indexLines, err := lines.ReadFileLines(indexFile)
	if err != nil {
		if os.IsNotExist(err) {
			return "not_found", nil
		}
		return "failed", fmt.Errorf("read archive index: %w", err)
	}

	marker := fmt.Sprintf("**%s**", folder)

	var found bool
	var kept []string
	for _, line := range indexLines {
		if strings.Contains(line, marker) {
			found = true
			continue
		}
		kept = append(kept, line)
	}

	if !found {
		return "not_found", nil
	}

	if err := atomicfile.WriteFile(indexFile, []byte(strings.Join(kept, "\n")), 0o644); err != nil {
		return "failed", fmt.Errorf("write archive index: %w", err)
	}
	return "removed", nil
}
