package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab-kit/internal"
	"github.com/spf13/cobra"
)

// migrationsStatusJSON is the machine-readable projection of DiscoverResult
// emitted by `fab migrations-status --json` and consumed by /fab-setup migrations.
type migrationsStatusJSON struct {
	Local      string              `json:"local"`
	Engine     string              `json:"engine"`
	Applicable []migrationRangeOut `json:"applicable"`
	GapSkips   []string            `json:"gap_skips"`
	Overlaps   []string            `json:"overlaps"`
}

type migrationRangeOut struct {
	From string `json:"from"`
	To   string `json:"to"`
	File string `json:"file"`
}

func migrationsStatusCmd() *cobra.Command {
	var asJSON bool
	cmd := &cobra.Command{
		Use:   "migrations-status",
		Short: "Report which migrations apply between the local and engine versions",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runMigrationsStatus(cmd, asJSON)
		},
	}
	cmd.Flags().BoolVar(&asJSON, "json", false, "Emit the discovery result as JSON")
	return cmd
}

// runMigrationsStatus resolves the local and engine versions, scans the engine
// migrations dir, runs discovery, and renders the result. It returns an error
// only on a genuine failure (missing version file, unreadable migrations dir);
// an overlap is a clean result surfaced via the overlaps field, not an error.
func runMigrationsStatus(cmd *cobra.Command, asJSON bool) error {
	cfg, err := internal.RequireManagedRepo()
	if err != nil {
		return err
	}

	// Local version: fab/.kit-migration-version.
	migVersionFile := filepath.Join(cfg.RepoRoot, "fab", ".kit-migration-version")
	localData, err := os.ReadFile(migVersionFile)
	if err != nil {
		return fmt.Errorf("cannot read fab/.kit-migration-version: %w. Run 'fab sync' to create it", err)
	}
	local := strings.TrimSpace(string(localData))

	// Engine version + migrations dir: from the cached kit for fab_version.
	// This is a read-only query — resolve the already-cached kit rather than
	// forcing a download. The kit is present whenever the repo has been synced.
	kitDir := internal.CachedKitDir(cfg.FabVersion)
	engineData, err := os.ReadFile(filepath.Join(kitDir, "VERSION"))
	if err != nil {
		return fmt.Errorf("cannot read engine VERSION at %s: %w. Run 'fab sync' to populate the kit cache", filepath.Join(kitDir, "VERSION"), err)
	}
	engine := strings.TrimSpace(string(engineData))

	migrationsDir := filepath.Join(kitDir, "migrations")
	result, err := internal.DiscoverMigrations(migrationsDir, local, engine)
	if err != nil {
		return err
	}

	out := cmd.OutOrStdout()
	if asJSON {
		payload := migrationsStatusJSON{
			Local:      result.Local,
			Engine:     result.Engine,
			Applicable: []migrationRangeOut{},
			GapSkips:   []string{},
			Overlaps:   []string{},
		}
		for _, r := range result.Applicable {
			payload.Applicable = append(payload.Applicable,
				migrationRangeOut{From: r.From, To: r.To, File: r.File})
		}
		if len(result.GapSkips) > 0 {
			payload.GapSkips = result.GapSkips
		}
		if len(result.Overlaps) > 0 {
			payload.Overlaps = result.Overlaps
		}
		enc := json.NewEncoder(out)
		enc.SetIndent("", "  ")
		return enc.Encode(payload)
	}

	fmt.Fprintf(out, "Local version:  %s\n", result.Local)
	fmt.Fprintf(out, "Engine version: %s\n", result.Engine)
	if len(result.Applicable) == 0 {
		fmt.Fprintln(out, "No migrations apply.")
	} else {
		fmt.Fprintf(out, "Migrations to apply (%d):\n", len(result.Applicable))
		for i, r := range result.Applicable {
			fmt.Fprintf(out, "  [%d/%d] %s -> %s (%s)\n", i+1, len(result.Applicable), r.From, r.To, r.File)
		}
	}
	for _, s := range result.GapSkips {
		fmt.Fprintln(out, s)
	}
	if len(result.Overlaps) > 0 {
		fmt.Fprintln(out, "Overlapping migration ranges detected:")
		for _, o := range result.Overlaps {
			fmt.Fprintf(out, "  %s\n", o)
		}
	}
	return nil
}
