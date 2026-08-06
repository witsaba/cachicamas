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
	"context"
	"encoding/json"
	"errors"
	"net"

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

// categorizeStreamError maps a non-frame, transport-side failure err to
// its AI-19 failure category (design.md D6, R-AEM-014). It is the
// single derivation point stream-side errors reach the AI-19 vocabulary
// through — shared with mapErrorResponse's retryableFor derivation, so
// stage 1's stage-1 status mapping and stage 2's transport mapping
// report the same category for the same cause shape.
//
// Order (design D6, recorded here verbatim so a reviewer can match
// against the spec's S-AEM-051…055 verbatim): context.Canceled first,
// then context.DeadlineExceeded, then net.Error.Timeout(), then the
// decoder's own Category() sentinels, then any other error →
// Unavailable. The "canceled checked first" note in design D6 is
// load-bearing: *url.Error wraps context.Canceled with Timeout() set
// inconsistently across transports, so a net.Error.Timeout() check on
// the un-rewrapped error would classify cancellation as Timeout on
// some transports. Sentinel checks are transport-independent.
func categorizeStreamError(err error) (ai.FailureCategory, bool) {
	if err == nil {
		return 0, false
	}

	// Context-cancellation and context-deadline are sentinel checks
	// (R-AEM-014). errors.Is traverses the wrap chain.
	if errors.Is(err, context.Canceled) {
		return ai.FailureCategoryCancellation, true
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return ai.FailureCategoryTimeout, true
	}

	// A non-context net-level timeout (a transport read deadline, a
	// syscall EAGAIN window) — checked ONLY after the sentinel checks,
	// because *url.Error wrapping context.DeadlineExceeded would
	// otherwise match here too with the same shape.
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return ai.FailureCategoryTimeout, true
	}

	// Decoder-sentinel-derived failures: the package-local
	// ErrFrameTooLarge / ErrTruncated (errors.go's Category) AND the
	// unexported wrappers (errMalformedIdentity, errIncompleteStream,
	// errMalformedChunkJSON, the AI-28.5 row sentinels) all wrap
	// ai.ErrMalformedResponse via %w. errors.Is matches each of those —
	// package-local or unexported — uniformly, so the categorization
	// stays total over every cause this producer's own run loop ever
	// constructs. (R-AEM-010's "MalformedResponse" shape.)
	if errors.Is(err, ai.ErrMalformedResponse) {
		return ai.FailureCategoryMalformedResponse, true
	}

	// Default: a transport-side error we don't recognise by shape —
	// connection reset, unexpected EOF, a broken pipe. retryableFor
	// already maps Unavailable → true (R-AEM-003, design D6).
	return ai.FailureCategoryUnavailable, true
}

// midStreamFailureFrom builds a mid-stream failure from a transport-side
// error err (R-AEM-012…014), with the category derived through
// categorizeStreamError and the cause reachable through Unwrap. The
// returned failure's Retryable() reflects the same shared derivation
// (failure_map.go's retryableFor) the status path already uses —
// Retryable is a classification derived from category alone, never set
// per-call (design D6).
//
// outputPreceded is supplied by the caller, exactly as
// failureFromErrorFrame reads it.
func midStreamFailureFrom(err error, outputPreceded bool) *ai.Failure {
	category, _ := categorizeStreamError(err)
	failure, constructErr := ai.MidStreamFailure(ai.FailureReport{
		Category:  category,
		Retryable: retryableFor(category),
		Cause:     err,
	}, outputPreceded)
	if constructErr != nil {
		// category is always a valid vocabulary member by the
		// categorization itself, so construction cannot fail for inputs
		// this function always supplies — the same posture every other
		// ai.MidStreamFailure call in this package already takes.
		return nil
	}
	return failure
}
