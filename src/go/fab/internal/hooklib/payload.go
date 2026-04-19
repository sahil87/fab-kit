package hooklib

import (
	"encoding/json"
	"io"
)

// SessionPayload represents the hook stdin JSON payload for session-scoped
// events (Stop, SessionStart, UserPromptSubmit). Only the fields the hook
// handlers need are extracted — other payload keys are ignored.
type SessionPayload struct {
	SessionID      string `json:"session_id"`
	TranscriptPath string `json:"transcript_path"`
}

// ParseSessionPayload reads a session-scoped hook payload from r and returns
// the decoded struct. An empty input is returned as a zero-valued struct with
// no error so callers can treat "empty stdin" identically to "missing
// session_id". A present but malformed JSON body returns an error so
// callers can distinguish this from the absent case and swallow it per the
// hooks' swallow-on-error discipline.
func ParseSessionPayload(r io.Reader) (SessionPayload, error) {
	data, err := io.ReadAll(r)
	if err != nil {
		return SessionPayload{}, err
	}
	if len(data) == 0 {
		return SessionPayload{}, nil
	}

	var p SessionPayload
	if err := json.Unmarshal(data, &p); err != nil {
		return SessionPayload{}, err
	}
	return p, nil
}
