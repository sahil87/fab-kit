package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"text/tabwriter"

	"github.com/sahil87/fab-kit/src/go/fab/internal/config"
	"github.com/sahil87/fab-kit/src/go/fab/internal/resolve"
	"github.com/spf13/cobra"
)

// presetRoleJSON / presetJSON are the wire shape of `fab agent presets --json` — a
// CROSS-REPO CONTRACT read by HexoKit (documented in _cli-fab.md). Every field but
// name/description/source is omitempty: a preset is sparse, and a field it does not
// set is omitted, never emitted empty. There is deliberately no "active" flag.
type presetRoleJSON struct {
	CLI    string `json:"cli,omitempty"`
	Model  string `json:"model,omitempty"`
	Effort string `json:"effort,omitempty"`
}

type presetJSON struct {
	Name        string                    `json:"name"`
	Description string                    `json:"description"`
	Source      string                    `json:"source"`
	Session     string                    `json:"session,omitempty"`
	Workers     string                    `json:"workers,omitempty"`
	Profiles    map[string]presetRoleJSON `json:"profiles,omitempty"`
}

// agentPresetsCmd implements `fab agent presets [--repo <path>] [--json]` — list the
// defined agent presets (built-in ∪ system ∪ project ∪ env definitions, merged per
// key), sorted by name. The reserved `none` is never listed. Template:
// agentModelsCmd.
func agentPresetsCmd() *cobra.Command {
	var jsonOutput bool
	var repo string
	cmd := &cobra.Command{
		Use:   "presets",
		Short: "List the defined agent presets (agent_presets) and where each comes from",
		Long: "Lists every defined agent preset — built-in, system, project and environment " +
			"definitions merged per key — sorted by name; the reserved name `none` is never listed. " +
			"Select one with `agent.preset` (FAB_AGENT_PRESET in the environment) or " +
			"`fab agent --preset <name>`. With --json prints an array of " +
			"{name, description, source, session?, workers?, profiles?}, where source is the highest " +
			"layer that contributed to the definition (built-in, system, project or env) and fields a " +
			"preset does not set are omitted.",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			layers, err := loadPresetLayers(repo)
			if err != nil {
				return err
			}
			defs, sources := config.AgentPresetDefinitions(layers)
			names := make([]string, 0, len(defs))
			for name := range defs {
				names = append(names, name)
			}
			sort.Strings(names)

			if jsonOutput {
				rows := make([]presetJSON, 0, len(names))
				for _, name := range names {
					rows = append(rows, presetRow(name, defs[name], sources[name]))
				}
				return json.NewEncoder(cmd.OutOrStdout()).Encode(rows)
			}
			return writePresetTable(cmd, names, defs, sources)
		},
	}
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "print a JSON array of preset definitions (a contract read by HexoKit)")
	cmd.Flags().StringVar(&repo, "repo", "", "repo root to read the config from (default: current repo)")
	return cmd
}

// loadPresetLayers loads the raw cascade layers: the --repo project, the current
// project, or the project-free layers outside any fab/ project.
func loadPresetLayers(repo string) (*config.Layers, error) {
	if repo != "" {
		return config.LoadLayers(filepath.Join(repo, "fab", "project", "config.yaml"))
	}
	fabRoot, err := resolve.FabRoot()
	if errors.Is(err, resolve.ErrNoFabRoot) {
		return config.LoadLayersNoProject(), nil
	}
	if err != nil {
		return nil, err
	}
	return config.LoadLayers(filepath.Join(fabRoot, "project", "config.yaml"))
}

// presetRow projects one definition onto the JSON contract, omitting unset fields.
func presetRow(name string, p config.AgentPreset, source config.PresetSource) presetJSON {
	row := presetJSON{Name: name, Description: p.Description, Source: string(source), Session: p.Session, Workers: p.Workers}
	for role, rp := range p.Profiles {
		if rp.CLI == "" && rp.Model == "" && rp.Effort == "" {
			continue
		}
		if row.Profiles == nil {
			row.Profiles = make(map[string]presetRoleJSON)
		}
		row.Profiles[role] = presetRoleJSON{CLI: rp.CLI, Model: rp.Model, Effort: rp.Effort}
	}
	return row
}

func writePresetTable(cmd *cobra.Command, names []string, defs map[string]config.AgentPreset, sources map[string]config.PresetSource) error {
	writer := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, "NAME\tSOURCE\tDESCRIPTION\tSETS")
	for _, name := range names {
		fmt.Fprintf(writer, "%s\t%s\t%s\t%s\n", name, sources[name], defs[name].Description, presetSummary(defs[name]))
	}
	return writer.Flush()
}

// presetSummary is the compact "what it sets" column: session/workers, then each
// role's fields in role-name order.
func presetSummary(p config.AgentPreset) string {
	var parts []string
	if p.Session != "" {
		parts = append(parts, "session="+p.Session)
	}
	if p.Workers != "" {
		parts = append(parts, "workers="+p.Workers)
	}
	roles := make([]string, 0, len(p.Profiles))
	for role := range p.Profiles {
		roles = append(roles, role)
	}
	sort.Strings(roles)
	for _, role := range roles {
		rp := p.Profiles[role]
		if rp.CLI != "" {
			parts = append(parts, role+".cli="+rp.CLI)
		}
		if rp.Model != "" {
			parts = append(parts, role+".model="+rp.Model)
		}
		if rp.Effort != "" {
			parts = append(parts, role+".effort="+rp.Effort)
		}
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}
