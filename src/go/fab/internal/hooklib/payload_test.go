package hooklib

import (
	"strings"
	"testing"
)

func TestParseSessionPayload_WellFormed(t *testing.T) {
	input := `{"session_id":"uuid-1","transcript_path":"/tmp/t.jsonl","hook_event_name":"Stop"}`
	p, err := ParseSessionPayload(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SessionID != "uuid-1" {
		t.Errorf("SessionID = %q, want uuid-1", p.SessionID)
	}
	if p.TranscriptPath != "/tmp/t.jsonl" {
		t.Errorf("TranscriptPath = %q, want /tmp/t.jsonl", p.TranscriptPath)
	}
}

func TestParseSessionPayload_MissingSessionID(t *testing.T) {
	input := `{"hook_event_name":"Stop"}`
	p, err := ParseSessionPayload(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", p.SessionID)
	}
}

func TestParseSessionPayload_MalformedJSON(t *testing.T) {
	input := `{not-json`
	_, err := ParseSessionPayload(strings.NewReader(input))
	if err == nil {
		t.Error("expected error for malformed JSON")
	}
}

func TestParseSessionPayload_EmptyInput(t *testing.T) {
	p, err := ParseSessionPayload(strings.NewReader(""))
	if err != nil {
		t.Fatalf("unexpected error for empty input: %v", err)
	}
	if p.SessionID != "" {
		t.Errorf("SessionID = %q, want empty", p.SessionID)
	}
	if p.TranscriptPath != "" {
		t.Errorf("TranscriptPath = %q, want empty", p.TranscriptPath)
	}
}

func TestParseSessionPayload_IgnoresExtraFields(t *testing.T) {
	input := `{"session_id":"uuid-2","cwd":"/home/user","permission_mode":"auto","extra_array":[1,2,3]}`
	p, err := ParseSessionPayload(strings.NewReader(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.SessionID != "uuid-2" {
		t.Errorf("SessionID = %q, want uuid-2", p.SessionID)
	}
}
