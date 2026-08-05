// AI-32.2 / AI-32.3 — the stream-failure path (R-AEM-010…014, S-AEM-040…055).
//
// failureFromErrorFrame builds R-AEM-010's terminal failure from the raw
// bytes of an in-band error frame: category Unknown, the vendor label
// (type falling back to code) carried as RawLabel, and ErrInBandErrorFrame
// in the cause chain so errors.Is distinguishes it from any transport
// failure (R-AEM-010, R-AEM-011). outputPreceded is the caller's own
// tracked fact — the same provenance stream.go's run tracks for every
// other failure path (design.md D7).
//
// # In-band vs. transport: identity, not text
//
// Distinguishing the two is an errors.Is(err, ErrInBandErrorFrame)
// question — the spec's stable identity match, never a substring scan of
// message text (R-AEM-010, R-AEM-011). ErrInBandErrorFrame is declared in
// errors.go and is the *only* thing this file attaches to a frame-derived
// failure's cause chain.

package openaicompat

import (
	"bytes"
	"encoding/json"

	"github.com/cachicamas/backend/agent/src/ai"
)

// inBandErrorFrameText is inBandErrorFrame's fixed diagnostic text —
// never reflects the payload, never reflects outputPreceded. The
// in-band failure's RawLabel travels on the *ai.Failure; this cause
// carries only the identity R-AEM-011's errors.Is relies on.
const inBandErrorFrameText = "openaicompat: in-band error frame"

// inBandErrorFrame is the unexported, structural cause this file attaches
// to a frame-derived failure so the chain carries ErrInBandErrorFrame
// without reproducing any payload bytes through Error() — the same
// redacted-text discipline capture.go's own capturedBody already keeps
// (R-AEM-016).
type inBandErrorFrame struct {
	payload []byte
	label   string
}

// Error returns the fixed diagnostic text for an in-band error frame cause
// (R-AEM-010). The payload is deliberately not reproduced: R-AEM-016's
// structural redaction posture applies here too — R-AEM-010's identity
// is ErrInBandErrorFrame under errors.Is, not a substring of Error() text.
func (e *inBandErrorFrame) Error() string { return inBandErrorFrameText }

// Unwrap returns ErrInBandErrorFrame so the wrapped cause chain reaches
// the stable identity under errors.Is (R-AEM-010, R-AEM-011,
// S-AEM-040/043).
func (e *inBandErrorFrame) Unwrap() error { return ErrInBandErrorFrame }

// failureFromErrorFrame is the package-level construction site the
// stream-integration test (and any future producer-integration code in
// stream.go's run loop, post-2a) calls. It returns R-AEM-010's terminal
// mid-stream failure: category Unknown, vendor label as RawLabel, and
// ErrInBandErrorFrame in the cause chain. outputPreceded is supplied by
// the caller, exactly as stream.go's run tracks it for every other
// failure site (design D7): true iff at least one normalized output
// event has already been emitted on this stream.
//
// A nil payload, a payload parse that yields no label, and a payload of
// any other shape all produce a usable failure — R-AEM-006's tolerant
// vendor-body parsing is reused for the label extraction so a malformed
// frame still surfaces a typed terminal failure rather than crashing the
// decoder or leaving the carrier open.
func failureFromErrorFrame(payload []byte, outputPreceded bool) *ai.Failure {
	label, _ := parseVendorErrorBody(payload)

	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnknown,
		RawLabel: label,
		Cause: &inBandErrorFrame{
			payload: payload,
			label:   label,
		},
	}, outputPreceded)
	if err != nil {
		// category is always a valid vocabulary member, so construction
		// cannot fail for inputs this function always supplies itself —
		// the same posture mapErrorResponse and the producer's own
		// emitFailure already take for their own MidStreamFailure calls.
		return nil
	}
	return failure
}

// isInBandErrorFrame reports whether data is the JSON shape of a vendor
// error body — {"error": {…}} — dispatching a frame mid-stream that is
// not a chunk at all (R-AEM-010). The check is a tiny preamble over the
// raw JSON: an object starting with `{` whose only key (or whose first
// key) is "error", matched without unwrapping the value. A frame whose
// JSON parses to anything else — a chunk, an unknown-type frame
// (R-ATS-017), or invalid JSON — returns false.
//
// The intentional narrowness: this is dialect-conventional behavior
// (R-AEM-010, cited negative E4) and the wire shape is so stable that a
// one-byte check before decodeChunk is the right cheap filter. A frame
// that happens to start with `{"error":…` and a top-level "choices"
// array would still report false here — the only key is "error".
func isInBandErrorFrame(data []byte) bool {
	if len(data) == 0 {
		return false
	}
	trim := bytes.TrimSpace(data)
	if len(trim) == 0 || trim[0] != '{' {
		return false
	}

	// Probe for an "error" top-level key by parsing into a struct that
	// declares a single field name "error". A frame whose top-level JSON
	// is wrapped ({"error":{…}}) carries that key; a bare ({…}) does
	// too if it actually has the vendor fields — both pass through
	// failureFromErrorFrame's tolerant parseVendorErrorBody.
	var probe struct {
		Error json.RawMessage `json:"error"`
	}
	if err := json.Unmarshal(trim, &probe); err != nil {
		return false
	}
	return len(probe.Error) > 0
}
