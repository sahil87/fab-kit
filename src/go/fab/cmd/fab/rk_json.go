package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
)

// unwrapRkJSON normalizes an `rk … --json` document to its bare payload,
// accepting BOTH of run-kit's output shapes: the bare document older
// releases print, and the standard envelope newer releases wrap every
// `--json` verb in — `{"ok":true,"result":<doc>}` on success and
// `{"ok":false,"error":{"code":…,"message":…}}` on failure. The presence of
// the `ok` key (not its value) is the discriminator, so a bare object that
// happens to lack `ok` passes through unchanged like a bare array does. The
// shape sniff IS the compatibility story: no rk version gate, no probe.
//
// Errors: an `ok:false` envelope (the error text carries rk's code and
// message), an `ok:true` envelope with `result` absent or null, malformed
// JSON, and empty or whitespace-only input. Callers keep their existing
// degrade branch — an unwrap error takes exactly the path an unmarshal
// error took before the envelope existed.
func unwrapRkJSON(data []byte) ([]byte, error) {
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 {
		return nil, errors.New("rk json: empty output")
	}
	if trimmed[0] == '[' {
		if !json.Valid(trimmed) {
			return nil, errors.New("rk json: malformed bare array")
		}
		return trimmed, nil
	}
	var probe struct {
		OK     *bool           `json:"ok"`
		Result json.RawMessage `json:"result"`
		Error  *struct {
			Code    string `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}
	if bytes.Equal(trimmed, []byte("null")) {
		// A bare null would unmarshal into an empty slice downstream and read
		// as "zero rows" — enough to trigger a seed or suppress a fallback.
		// Only an established document is a payload.
		return nil, errors.New("rk json: null document")
	}
	if err := json.Unmarshal(trimmed, &probe); err != nil {
		return nil, fmt.Errorf("rk json: %w", err)
	}
	if probe.OK == nil {
		return trimmed, nil // a bare object, not an envelope
	}
	if !*probe.OK {
		if probe.Error != nil {
			return nil, fmt.Errorf("rk json: ok=false (%s: %s)", probe.Error.Code, probe.Error.Message)
		}
		return nil, errors.New("rk json: ok=false")
	}
	if len(probe.Result) == 0 || bytes.Equal(probe.Result, []byte("null")) {
		return nil, errors.New("rk json: ok=true envelope without a result")
	}
	return probe.Result, nil
}
