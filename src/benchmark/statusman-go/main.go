package main

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

var stages = []string{"intake", "spec", "tasks", "apply", "review", "hydrate", "ship", "review-pr"}

// StatusFile represents the .status.yaml structure using ordered yaml.Node
// to preserve key order on round-trip.

func readStatusNode(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("cannot read %s: %w", path, err)
	}
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("cannot parse %s: %w", path, err)
	}
	// doc is a Document node; the mapping is its first content child
	if doc.Kind != yaml.DocumentNode || len(doc.Content) == 0 {
		return nil, fmt.Errorf("unexpected YAML structure in %s", path)
	}
	return doc.Content[0], nil
}

func writeStatusAtomic(path string, root *yaml.Node) error {
	doc := &yaml.Node{Kind: yaml.DocumentNode, Content: []*yaml.Node{root}}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("cannot serialize YAML: %w", err)
	}
	tmp := path + fmt.Sprintf(".%d.%d", os.Getpid(), time.Now().UnixMilli())
	if err := os.WriteFile(tmp, out, 0644); err != nil {
		return fmt.Errorf("cannot write temp file: %w", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return fmt.Errorf("cannot rename temp file: %w", err)
	}
	return nil
}

// nodeGet returns the value node for a given key in a mapping node.
func nodeGet(m *yaml.Node, key string) *yaml.Node {
	for i := 0; i < len(m.Content)-1; i += 2 {
		if m.Content[i].Value == key {
			return m.Content[i+1]
		}
	}
	return nil
}

// nodeSet sets a scalar value for a key in a mapping node.
func nodeSet(m *yaml.Node, key, value string) {
	for i := 0; i < len(m.Content)-1; i += 2 {
		if m.Content[i].Value == key {
			m.Content[i+1].Value = value
			return
		}
	}
	// Key not found — append
	m.Content = append(m.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

func nowISO() string {
	return time.Now().Format(time.RFC3339)
}

// ─── progress-map ─────────────────────────────────────────────────────────────

func progressMap(statusFile string) error {
	root, err := readStatusNode(statusFile)
	if err != nil {
		return err
	}
	progress := nodeGet(root, "progress")
	for _, stage := range stages {
		state := "pending"
		if progress != nil {
			if v := nodeGet(progress, stage); v != nil {
				state = v.Value
			}
		}
		fmt.Printf("%s:%s\n", stage, state)
	}
	return nil
}

// ─── set-change-type ──────────────────────────────────────────────────────────

func setChangeType(statusFile, changeType string) error {
	valid := map[string]bool{"feat": true, "fix": true, "refactor": true, "docs": true, "test": true, "ci": true, "chore": true}
	if !valid[changeType] {
		return fmt.Errorf("invalid change type '%s'", changeType)
	}
	root, err := readStatusNode(statusFile)
	if err != nil {
		return err
	}
	nodeSet(root, "change_type", changeType)
	nodeSet(root, "last_updated", nowISO())
	return writeStatusAtomic(statusFile, root)
}

// ─── finish ───────────────────────────────────────────────────────────────────

func finish(statusFile, stage string) error {
	root, err := readStatusNode(statusFile)
	if err != nil {
		return err
	}
	ts := nowISO()

	progress := nodeGet(root, "progress")
	if progress == nil {
		return fmt.Errorf("no progress block in status file")
	}

	currentState := "pending"
	if v := nodeGet(progress, stage); v != nil {
		currentState = v.Value
	}
	if currentState != "active" && currentState != "ready" {
		return fmt.Errorf("cannot finish stage '%s' — current state is '%s'", stage, currentState)
	}

	// Set stage to done
	nodeSet(progress, stage, "done")

	// Find next stage
	var nextStage string
	for i, s := range stages {
		if s == stage && i < len(stages)-1 {
			nextStage = stages[i+1]
			break
		}
	}

	// Auto-activate next pending stage
	activateNext := false
	if nextStage != "" {
		nextState := "pending"
		if v := nodeGet(progress, nextStage); v != nil {
			nextState = v.Value
		}
		if nextState == "pending" {
			nodeSet(progress, nextStage, "active")
			activateNext = true
		}
	}

	// Stage metrics — completed_at for current stage
	metrics := nodeGet(root, "stage_metrics")
	if metrics != nil {
		if stageMetrics := nodeGet(metrics, stage); stageMetrics != nil {
			nodeSet(stageMetrics, "completed_at", ts)
			// Set flow style for compact output
			stageMetrics.Style = yaml.FlowStyle
		}

		// Metrics for next stage activation
		if activateNext {
			nextMetrics := &yaml.Node{
				Kind:  yaml.MappingNode,
				Style: yaml.FlowStyle,
				Content: []*yaml.Node{
					{Kind: yaml.ScalarNode, Value: "started_at"},
					{Kind: yaml.ScalarNode, Value: ts},
					{Kind: yaml.ScalarNode, Value: "driver"},
					{Kind: yaml.ScalarNode, Value: "benchmark"},
					{Kind: yaml.ScalarNode, Value: "iterations"},
					{Kind: yaml.ScalarNode, Value: "1", Tag: "!!int"},
				},
			}
			// Replace or append
			found := false
			for i := 0; i < len(metrics.Content)-1; i += 2 {
				if metrics.Content[i].Value == nextStage {
					metrics.Content[i+1] = nextMetrics
					found = true
					break
				}
			}
			if !found {
				metrics.Content = append(metrics.Content,
					&yaml.Node{Kind: yaml.ScalarNode, Value: nextStage},
					nextMetrics,
				)
			}
		}
	}

	nodeSet(root, "last_updated", ts)
	return writeStatusAtomic(statusFile, root)
}

// ─── CLI ──────────────────────────────────────────────────────────────────────

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: statusman {progress-map|set-change-type|finish} <status_file> [args...]")
		os.Exit(1)
	}

	var err error
	switch os.Args[1] {
	case "--help", "-h":
		fmt.Println("statusman-go: Go benchmark contender")
		fmt.Println("Usage: statusman {progress-map|set-change-type|finish} <status_file> [args...]")
	case "progress-map":
		if len(os.Args) < 3 {
			fmt.Fprintln(os.Stderr, "Usage: statusman progress-map <status_file>")
			os.Exit(1)
		}
		err = progressMap(os.Args[2])
	case "set-change-type":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: statusman set-change-type <status_file> <type>")
			os.Exit(1)
		}
		err = setChangeType(os.Args[2], os.Args[3])
	case "finish":
		if len(os.Args) < 4 {
			fmt.Fprintln(os.Stderr, "Usage: statusman finish <status_file> <stage>")
			os.Exit(1)
		}
		err = finish(os.Args[2], os.Args[3])
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", os.Args[1])
		os.Exit(1)
	}

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %s\n", err)
		os.Exit(1)
	}
}
