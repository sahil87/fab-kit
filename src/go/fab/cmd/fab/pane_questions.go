package main

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
	"github.com/spf13/cobra"
)

func paneQuestionsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "questions",
		Short: "Sweep candidate panes for pending questions/prompts",
		Long: "Sweep candidate panes for pending questions/prompts: capture the last 20 lines\n" +
			"of each, apply the mechanical guards (Claude turn-boundary, blank capture), then\n" +
			"scan for the mechanical indicator patterns fab-operator's §5 auto-nudge policy\n" +
			"defines. This is the policy-bearing sweep the operator dispatches per tick —\n" +
			"detection input only, never a license to send blind (the delivery guards stay\n" +
			"the operator's own).",
		Args: cobra.NoArgs,
		RunE: runPaneQuestions,
	}
	cmd.Flags().Bool("all-sessions", false, "Sweep candidate panes across all tmux sessions")
	cmd.Flags().StringSlice("panes", nil, "Explicit pane IDs to sweep (repeatable or comma-separated)")
	cmd.Flags().Bool("json", false, "Output as JSON")
	cmd.MarkFlagsMutuallyExclusive("all-sessions", "panes")
	return cmd
}

// questionsCaptureLines is the fixed capture window for the sweep (not a flag —
// the plan pins this at 20 lines, matching the operator's existing §5 capture).
const questionsCaptureLines = 20

// paneCaptureFn is the injectable capture seam: pane.Capture execs tmux
// directly with no test seam today, so this package-level var lets tests
// stub content without a live pane — the rkPanesRunner/currentSessionName/
// operatorStatePathOverride precedent (pane_map.go, operator_tick_start.go).
var paneCaptureFn = pane.Capture

// questionsRowsFn is the injectable row-snapshot seam for the up-front
// collectPaneRows call — tests seed pane rows without a live tmux server
// (the tickSnapshotRows/rkPanesRunner precedent).
var questionsRowsFn = collectPaneRows

// questionMatch is one swept pane whose captured content matched a
// mechanical indicator class.
type questionMatch struct {
	Pane       string `json:"pane"`
	AgentState string `json:"agent_state"`
	Indicator  string `json:"indicator"`
	Snippet    string `json:"snippet"`
}

// questionSkip is one candidate pane the sweep did not turn into a match,
// with the mechanical reason it was skipped.
type questionSkip struct {
	Pane   string `json:"pane"`
	Reason string `json:"reason"`
}

// questionsResult is the full sweep result: --json marshals this directly.
type questionsResult struct {
	Matches []questionMatch `json:"matches"`
	Skipped []questionSkip  `json:"skipped"`
}

func runPaneQuestions(cmd *cobra.Command, args []string) error {
	allSessions, _ := cmd.Flags().GetBool("all-sessions")
	panesFlag, _ := cmd.Flags().GetStringSlice("panes")
	jsonFlag, _ := cmd.Flags().GetBool("json")
	server, _ := cmd.Flags().GetString("server")

	mode := sessionDefault
	if allSessions || len(panesFlag) > 0 {
		// --all-sessions discovers server-wide; --panes resolves explicit IDs
		// server-wide too — pane IDs are server-global, the tick's candidates:
		// block spans sessions, and this keeps --panes free of any $TMUX /
		// current-session dependency (R1).
		mode = sessionAll
	}

	// $TMUX guard only when the candidate set comes from discovery against the
	// current session — an explicit --panes list or --all-sessions needs no
	// tmux client context, mirroring `fab pane map`'s same-shaped guard.
	if !allSessions && len(panesFlag) == 0 && os.Getenv("TMUX") == "" {
		return fmt.Errorf("not inside a tmux session")
	}

	rows, err := questionsRowsFn(mode, "", server)
	if err != nil {
		return err
	}
	byPane := make(map[string]paneRow, len(rows))
	for _, r := range rows {
		byPane[r.pane] = r
	}

	candidates := panesFlag
	if len(panesFlag) == 0 {
		candidates = nil
		for _, r := range rows {
			if r.agentState == pane.AgentStateWaiting || r.agentState == pane.AgentStateIdle {
				candidates = append(candidates, r.pane)
			}
		}
	}

	result := questionsResult{Matches: []questionMatch{}, Skipped: []questionSkip{}}
	for _, id := range candidates {
		row, present := byPane[id]
		if !present {
			result.Skipped = append(result.Skipped, questionSkip{Pane: id, Reason: "capture_failed"})
			continue
		}
		if row.agentState != pane.AgentStateWaiting && row.agentState != pane.AgentStateIdle {
			result.Skipped = append(result.Skipped, questionSkip{Pane: id, Reason: "state_changed"})
			continue
		}

		content, err := paneCaptureFn(server, id, questionsCaptureLines)
		if err != nil {
			result.Skipped = append(result.Skipped, questionSkip{Pane: id, Reason: "capture_failed"})
			continue
		}
		if isBlankCapture(content) {
			result.Skipped = append(result.Skipped, questionSkip{Pane: id, Reason: "blank_capture"})
			continue
		}
		if hasTurnBoundary(content) {
			result.Skipped = append(result.Skipped, questionSkip{Pane: id, Reason: "turn_boundary"})
			continue
		}
		matched, indicator, snippet := scanIndicators(content)
		if !matched {
			result.Skipped = append(result.Skipped, questionSkip{Pane: id, Reason: "no_indicator"})
			continue
		}
		result.Matches = append(result.Matches, questionMatch{
			Pane:       id,
			AgentState: row.agentState,
			Indicator:  indicator,
			Snippet:    snippet,
		})
	}

	if jsonFlag {
		enc := json.NewEncoder(cmd.OutOrStdout())
		enc.SetIndent("", "  ")
		return enc.Encode(result)
	}

	printQuestionsHuman(cmd, result)
	return nil
}

// printQuestionsHuman renders the sweep result as plain lines: one per match,
// one per skip, then a count summary. An empty candidate set (no matches, no
// skips) prints a single line, mirroring `fab pane map`'s "No tmux panes
// found." convention.
func printQuestionsHuman(cmd *cobra.Command, result questionsResult) {
	w := cmd.OutOrStdout()
	if len(result.Matches) == 0 && len(result.Skipped) == 0 {
		fmt.Fprintln(w, "No candidate panes.")
		return
	}
	for _, m := range result.Matches {
		fmt.Fprintf(w, "%s [%s] %s: %s\n", m.Pane, m.AgentState, m.Indicator, m.Snippet)
	}
	for _, s := range result.Skipped {
		fmt.Fprintf(w, "%s: %s\n", s.Pane, s.Reason)
	}
	fmt.Fprintf(w, "%d matched, %d skipped\n", len(result.Matches), len(result.Skipped))
}

// isBlankCapture reports whether captured content is empty or all-whitespace
// ("cannot determine" per the blank-capture guard).
func isBlankCapture(content string) bool {
	return strings.TrimSpace(content) == ""
}

// turnBoundaryPattern matches the Claude Code turn-boundary prompt line — a
// bare `>` (optionally padded with whitespace) on an otherwise empty line.
var turnBoundaryPattern = regexp.MustCompile(`^\s*>\s*$`)

// hasTurnBoundary reports whether either of the last 2 lines of content is a
// bare turn-boundary prompt (normal human-turn boundary, not a question).
func hasTurnBoundary(content string) bool {
	lines := nonEmptyTrailingSplit(content)
	n := len(lines)
	if n == 0 {
		return false
	}
	start := n - 2
	if start < 0 {
		start = 0
	}
	for _, l := range lines[start:] {
		if turnBoundaryPattern.MatchString(l) {
			return true
		}
	}
	return false
}

// nonEmptyTrailingSplit splits content on "\n", dropping a single trailing
// empty element left by a final newline (pane.Capture/TailLines already
// strips trailing blank screen-padding, so this only removes the artifact
// of the terminating "\n", not real blank content).
func nonEmptyTrailingSplit(content string) []string {
	lines := strings.Split(content, "\n")
	if n := len(lines); n > 0 && lines[n-1] == "" {
		lines = lines[:n-1]
	}
	return lines
}

// Indicator class names — stable, greppable identifiers for JSON output and
// tests, one per mechanical class in fab-operator.md §5 Question Detection
// MINUS class 4 (Claude Code permission/tool prompts — not mechanical; see
// scanIndicators).
const (
	indicatorQuestionMark       = "question_mark"
	indicatorYesNo              = "yes_no"
	indicatorActionWord         = "action_word"
	indicatorImperativeQuestion = "imperative_question"
	indicatorColonPrompt        = "colon_prompt"
	indicatorEnumeratedOptions  = "enumerated_options"
	indicatorPressKey           = "press_key"
)

var (
	timestampPrefixPattern    = regexp.MustCompile(`^\s*[\[(]?\d{1,4}[-/:]\d{1,2}`)
	yesNoPattern              = regexp.MustCompile(`(?i)(\[y/n\]|\(y/n\)|\(yes/no\))`)
	actionWordPattern         = regexp.MustCompile(`\b(Allow|Approve|Confirm|Proceed)\?`)
	imperativeQuestionPattern = regexp.MustCompile(`(?i)(Do you want to|Should I|Would you like)`)
	enumeratedOptionsPattern  = regexp.MustCompile(`[1-9]\)`)
	pressKeyPattern           = regexp.MustCompile(`(?i)(Press.*key|press.*enter|hit.*enter)`)
)

// isQuestionMarkLine implements indicator class 1: the actual last non-empty
// line only, under 120 chars, ending in "?", and not starting with a
// comment/quote marker or a leading timestamp.
func isQuestionMarkLine(line string) bool {
	trimmed := strings.TrimRight(line, " \t")
	if !strings.HasSuffix(trimmed, "?") {
		return false
	}
	if len(trimmed) >= 120 {
		return false
	}
	leading := strings.TrimLeft(line, " \t")
	for _, prefix := range []string{"#", "//", "*", ">"} {
		if strings.HasPrefix(leading, prefix) {
			return false
		}
	}
	if timestampPrefixPattern.MatchString(line) {
		return false
	}
	return true
}

// matchOtherClasses tests indicator classes 2-7 against a single line, in
// listed order, returning the first class that matches.
func matchOtherClasses(line string) (indicator string, ok bool) {
	switch {
	case yesNoPattern.MatchString(line):
		return indicatorYesNo, true
	case actionWordPattern.MatchString(line):
		return indicatorActionWord, true
	case imperativeQuestionPattern.MatchString(line):
		return indicatorImperativeQuestion, true
	case strings.HasSuffix(strings.TrimRight(line, " \t"), ":"):
		return indicatorColonPrompt, true
	case enumeratedOptionsPattern.MatchString(line):
		return indicatorEnumeratedOptions, true
	case pressKeyPattern.MatchString(line):
		return indicatorPressKey, true
	}
	return "", false
}

// scanIndicators walks the captured content's non-empty lines bottom-most
// first, testing indicator class 1 (question-mark ending) only against the
// actual last line — per its documented "last non-empty line only" scope —
// and classes 2-7 against every line. The first (bottom-most) line where any
// class matches wins; ties within one line resolve by the listed class order
// (1 before 2-7 on the last line; 2-7 in the order above on every other
// line). No match anywhere returns ok=false.
func scanIndicators(content string) (ok bool, indicator string, snippet string) {
	var nonEmpty []string
	for _, l := range nonEmptyTrailingSplit(content) {
		if strings.TrimSpace(l) != "" {
			nonEmpty = append(nonEmpty, l)
		}
	}
	if len(nonEmpty) == 0 {
		return false, "", ""
	}
	lastIdx := len(nonEmpty) - 1
	for i := lastIdx; i >= 0; i-- {
		line := nonEmpty[i]
		if i == lastIdx && isQuestionMarkLine(line) {
			return true, indicatorQuestionMark, line
		}
		if ind, matched := matchOtherClasses(line); matched {
			return true, ind, line
		}
	}
	return false, "", ""
}
