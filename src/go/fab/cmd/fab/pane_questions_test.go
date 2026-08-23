package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sahil87/fab-kit/src/go/fab/internal/pane"
)

// --- pane questions test scaffolding -----------------------------------------

// qRow builds a seeded snapshot row for the questionsRowsFn stub.
func qRow(paneID, agentState string) paneRow {
	return paneRow{pane: paneID, agentState: agentState, session: "s", repo: "/r"}
}

// stubQuestionRows replaces the questionsRowsFn seam, returning the seeded
// rows and recording the sessionMode the sweep asked for.
func stubQuestionRows(t *testing.T, rows []paneRow) *sessionMode {
	t.Helper()
	gotMode := sessionDefault
	orig := questionsRowsFn
	questionsRowsFn = func(mode sessionMode, sessionName, server string) ([]paneRow, error) {
		gotMode = mode
		return rows, nil
	}
	t.Cleanup(func() { questionsRowsFn = orig })
	return &gotMode
}

// stubQuestionCapture replaces the paneCaptureFn seam with a per-pane content
// map; a pane absent from the map fails the capture.
func stubQuestionCapture(t *testing.T, contents map[string]string) {
	t.Helper()
	orig := paneCaptureFn
	paneCaptureFn = func(server, paneID string, lines int) (string, error) {
		if lines != questionsCaptureLines {
			t.Errorf("capture lines = %d, want %d", lines, questionsCaptureLines)
		}
		if c, ok := contents[paneID]; ok {
			return c, nil
		}
		return "", errors.New("pane gone")
	}
	t.Cleanup(func() { paneCaptureFn = orig })
}

// runQuestions executes `pane questions` with the given args and returns
// stdout and the execution error.
func runQuestions(t *testing.T, args ...string) (string, error) {
	t.Helper()
	cmd := paneQuestionsCmd()
	var stdout bytes.Buffer
	cmd.SetOut(&stdout)
	cmd.SetErr(&bytes.Buffer{})
	cmd.SilenceErrors = true
	cmd.SilenceUsage = true
	cmd.SetArgs(args)
	err := cmd.Execute()
	return stdout.String(), err
}

// --- pure scanner tests ------------------------------------------------------

func TestScanIndicators(t *testing.T) {
	cases := []struct {
		name      string
		content   string
		wantMatch bool
		wantInd   string
		wantSnip  string
	}{
		// Class 1: question mark — last non-empty line only.
		{"question mark last line", "working...\nContinue?", true, indicatorQuestionMark, "Continue?"},
		{"question mark earlier line only", "Everything ok?\nplain output", false, "", ""},
		{"question mark too long", strings.Repeat("x", 119) + "?", false, "", ""},
		{"question mark hash comment", "# is this done?", false, "", ""},
		{"question mark slash comment", "// is this done?", false, "", ""},
		{"question mark star bullet", "* is this done?", false, "", ""},
		{"question mark quote", "> is this done?", false, "", ""},
		{"question mark timestamped", "12:34 is this done?", false, "", ""},
		// Classes 2-7.
		{"yes no bracket", "Overwrite file [y/N]", true, indicatorYesNo, "Overwrite file [y/N]"},
		{"yes no parens", "continue (YES/NO)", true, indicatorYesNo, "continue (YES/NO)"},
		{"action word above plain last line", "Allow?\nprocessing done", true, indicatorActionWord, "Allow?"},
		{"action word case sensitive", "allow?\nprocessing done", false, "", ""},
		{"imperative question", "Do you want to proceed", true, indicatorImperativeQuestion, "Do you want to proceed"},
		{"imperative question case-insensitive", "should I continue", true, indicatorImperativeQuestion, "should I continue"},
		{"colon prompt", "Enter your name:", true, indicatorColonPrompt, "Enter your name:"},
		{"enumerated options", "1) yes  2) no", true, indicatorEnumeratedOptions, "1) yes  2) no"},
		{"press key", "Press any key to continue", true, indicatorPressKey, "Press any key to continue"},
		{"hit enter", "hit enter to accept", true, indicatorPressKey, "hit enter to accept"},
		// Bottom-most wins (R4 scenario): earlier Allow?, last line enumerated.
		{"bottom-most wins", "Allow?\n1) yes  2) no", true, indicatorEnumeratedOptions, "1) yes  2) no"},
		// Last line tests class 1 before classes 2-7.
		{"last line question mark beats yes_no", "Continue [Y/n]?", true, indicatorQuestionMark, "Continue [Y/n]?"},
		// Blank lines are skipped; trailing newline is tolerated.
		{"blank lines ignored", "first\n\nContinue?\n\n", true, indicatorQuestionMark, "Continue?"},
		{"no indicator", "building...\ndone", false, "", ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ok, ind, snip := scanIndicators(tc.content)
			if ok != tc.wantMatch {
				t.Fatalf("match = %v, want %v (indicator %q)", ok, tc.wantMatch, ind)
			}
			if !ok {
				return
			}
			if ind != tc.wantInd {
				t.Errorf("indicator = %q, want %q", ind, tc.wantInd)
			}
			if snip != tc.wantSnip {
				t.Errorf("snippet = %q, want %q", snip, tc.wantSnip)
			}
		})
	}
}

func TestHasTurnBoundary(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    bool
	}{
		{"last line bare prompt", "output\n>\n", true},
		{"second-to-last bare prompt padded", "output\n > \ntrailing", true},
		{"prompt above last two lines", ">\nline1\nline2\nline3", false},
		{"no prompt", "line1\nline2", false},
		{"prompt with text is not a boundary", "output\n> echo hi", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := hasTurnBoundary(tc.content); got != tc.want {
				t.Errorf("hasTurnBoundary(%q) = %v, want %v", tc.content, got, tc.want)
			}
		})
	}
}

func TestIsBlankCapture(t *testing.T) {
	if !isBlankCapture("") {
		t.Error("empty string should be blank")
	}
	if !isBlankCapture("  \n \t\n  ") {
		t.Error("all-whitespace should be blank")
	}
	if isBlankCapture("x") {
		t.Error("content should not be blank")
	}
}

// --- command-level tests (stubbed seams, no live tmux) ------------------------

func TestPaneQuestionsSweepSkipsAndMatches(t *testing.T) {
	stubQuestionRows(t, []paneRow{
		qRow("%1", pane.AgentStateWaiting),
		qRow("%2", pane.AgentStateIdle),
		qRow("%3", pane.AgentStateActive),
		qRow("%4", pane.AgentStateWaiting),
		qRow("%5", pane.AgentStateWaiting),
		qRow("%6", pane.AgentStateWaiting),
	})
	stubQuestionCapture(t, map[string]string{
		"%1": "working\nDo you want to proceed",
		"%2": "just output\nnothing to ask",
		"%4": "   \n\t\n",
		"%5": "output\n>\n",
		// %6 is absent from the map: capture error.
	})

	out, err := runQuestions(t, "--panes", "%1,%2,%3,%4,%5,%6,%99")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}

	wantLines := []string{
		"%1 [waiting] imperative_question: Do you want to proceed",
		"%2: no_indicator",
		"%3: state_changed",
		"%4: blank_capture",
		"%5: turn_boundary",
		"%6: capture_failed",
		"%99: capture_failed", // dead pane: absent from the snapshot
		"1 matched, 6 skipped",
	}
	for _, want := range wantLines {
		if !strings.Contains(out, want) {
			t.Errorf("stdout missing %q\n---\n%s", want, out)
		}
	}
}

func TestPaneQuestionsPanesModeSweepsServerWide(t *testing.T) {
	// --panes resolves explicit IDs against the whole server and needs no
	// $TMUX (R1): the up-front snapshot must be sessionAll.
	mode := stubQuestionRows(t, []paneRow{qRow("%3", pane.AgentStateWaiting)})
	stubQuestionCapture(t, map[string]string{"%3": "Enter your name:"})
	t.Setenv("TMUX", "")

	out, err := runQuestions(t, "--panes", "%3")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if *mode != sessionAll {
		t.Errorf("--panes snapshot mode = %v, want sessionAll", *mode)
	}
	if !strings.Contains(out, "%3 [waiting] colon_prompt: Enter your name:") {
		t.Errorf("stdout missing match line\n---\n%s", out)
	}
}

func TestPaneQuestionsAllSessionsFiltersPopulation(t *testing.T) {
	// Discovery sweeps only waiting/idle panes; active and unknown are
	// excluded by construction (R2).
	stubQuestionRows(t, []paneRow{
		qRow("%1", pane.AgentStateWaiting),
		qRow("%2", pane.AgentStateIdle),
		qRow("%3", pane.AgentStateActive),
		qRow("%4", ""),
	})
	captured := map[string]bool{}
	orig := paneCaptureFn
	paneCaptureFn = func(server, paneID string, lines int) (string, error) {
		captured[paneID] = true
		return "plain output", nil
	}
	t.Cleanup(func() { paneCaptureFn = orig })

	out, err := runQuestions(t, "--all-sessions")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if !captured["%1"] || !captured["%2"] {
		t.Errorf("waiting/idle panes not swept: %v", captured)
	}
	if captured["%3"] || captured["%4"] {
		t.Errorf("active/unknown panes swept: %v", captured)
	}
	if !strings.Contains(out, "0 matched, 2 skipped") {
		t.Errorf("stdout missing count line\n---\n%s", out)
	}
}

func TestPaneQuestionsDefaultModeUsesCurrentSession(t *testing.T) {
	mode := stubQuestionRows(t, []paneRow{qRow("%1", pane.AgentStateWaiting)})
	stubQuestionCapture(t, map[string]string{"%1": "plain output"})
	t.Setenv("TMUX", "/tmp/tmux-1000/default,1,0")

	if _, err := runQuestions(t); err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if *mode != sessionDefault {
		t.Errorf("no-flag snapshot mode = %v, want sessionDefault", *mode)
	}
}

func TestPaneQuestionsTmuxGuard(t *testing.T) {
	t.Setenv("TMUX", "")
	_, err := runQuestions(t)
	if err == nil || !strings.Contains(err.Error(), "not inside a tmux session") {
		t.Fatalf("no-flag outside tmux: err = %v, want tmux guard error", err)
	}
}

func TestPaneQuestionsMutualExclusivity(t *testing.T) {
	t.Setenv("TMUX", "")
	if _, err := runQuestions(t, "--all-sessions", "--panes", "%3"); err == nil {
		t.Fatal("--all-sessions + --panes should be rejected")
	}
}

func TestPaneQuestionsEmptyCandidates(t *testing.T) {
	stubQuestionRows(t, []paneRow{qRow("%1", pane.AgentStateActive)})
	stubQuestionCapture(t, nil)

	out, err := runQuestions(t, "--all-sessions")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if strings.TrimSpace(out) != "No candidate panes." {
		t.Errorf("stdout = %q, want %q", strings.TrimSpace(out), "No candidate panes.")
	}
}

func TestPaneQuestionsJSONSchema(t *testing.T) {
	stubQuestionRows(t, []paneRow{
		qRow("%1", pane.AgentStateWaiting),
		qRow("%2", pane.AgentStateWaiting),
	})
	stubQuestionCapture(t, map[string]string{
		"%1": "working\nDo you want to proceed",
		"%2": "plain output",
	})

	out, err := runQuestions(t, "--json", "--panes", "%1,%2")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	var doc struct {
		Matches []struct {
			Pane       string `json:"pane"`
			AgentState string `json:"agent_state"`
			Indicator  string `json:"indicator"`
			Snippet    string `json:"snippet"`
		} `json:"matches"`
		Skipped []struct {
			Pane   string `json:"pane"`
			Reason string `json:"reason"`
		} `json:"skipped"`
	}
	if err := json.Unmarshal([]byte(out), &doc); err != nil {
		t.Fatalf("parse json: %v\n---\n%s", err, out)
	}
	if len(doc.Matches) != 1 || doc.Matches[0].Pane != "%1" ||
		doc.Matches[0].AgentState != "waiting" ||
		doc.Matches[0].Indicator != indicatorImperativeQuestion ||
		doc.Matches[0].Snippet != "Do you want to proceed" {
		t.Errorf("matches = %+v", doc.Matches)
	}
	if len(doc.Skipped) != 1 || doc.Skipped[0].Pane != "%2" || doc.Skipped[0].Reason != "no_indicator" {
		t.Errorf("skipped = %+v", doc.Skipped)
	}
}

func TestPaneQuestionsJSONEmptyArrays(t *testing.T) {
	// Empty sweeps encode [] (never null) for both arrays (R5).
	stubQuestionRows(t, []paneRow{qRow("%1", pane.AgentStateActive)})
	stubQuestionCapture(t, nil)

	out, err := runQuestions(t, "--json", "--all-sessions")
	if err != nil {
		t.Fatalf("sweep: %v", err)
	}
	if !strings.Contains(out, `"matches": []`) || !strings.Contains(out, `"skipped": []`) {
		t.Errorf("empty sweep must encode empty arrays\n---\n%s", out)
	}
}
