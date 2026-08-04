// AI-28.1.1 — the producer shell: Stream's entry point, the pre-stream
// contract, the read/decode loop, response identity, and close discipline.
//
// This file lands R-ATS-001…006: the adapter's one streaming entry point,
// satisfying ai.ModelProvider at run time (R-ATS-001); the pre-stream
// contract — validate, then Translate, then consult ctx, with nothing
// observable before validation passes (R-ATS-002); a minimal transcript
// draining end to end, sequenced from 1, closed exactly once (R-ATS-003);
// the vendor's response identity and served model landing in the start
// event, byte-exact (R-ATS-004); and AI-20.3's cancellation discipline —
// every send selects, one closing site, no leak (R-ATS-005).
//
// # Producer-lifecycle order: validate → Translate → ctx (design.md D1, N-7a)
//
// req.IsZero() is checked first, then Translate(req) — which reuses AI-26's
// own validation/refusal and performs no I/O — then ctx.Err(). This order
// means a request Translate refuses (a reasoning-refusal, R-ART-015)
// reports the refusal, not a concurrent cancellation, when both hold: the
// spec pins only validation-before-cancellation (S-ATS-007), and is silent
// on where Translate's own refusal sits relative to ctx. Translate is read
// here as part of "validate once, before I/O, before consulting the
// context" (R-ATS-002) — its refusal is a property of the request, so it
// outranks a property of the call's context.
//
// # Text mapping now lives in stream_state.go (AI-28.1.2)
//
// R-ATS-007…010 — choice-0 content deltas, block minting, byte-exact
// reconstruction and the finish-reason gate — are implemented by
// stream_state.go's mapperState, not this file. This file's own job is
// unchanged from slice 1: drive the read/decode loop, recognise the
// terminal sentinel, and translate whatever the mapper reports into
// selected sends and the one closing site. errMalformedIdentity (a
// present-but-empty first-chunk id or model, design.md D7/N-7b) and the
// finish-reason gate's own malformed cause now live beside mapperState in
// stream_state.go, for the same reason: both are decisions the mapper
// makes while reading a chunk, not this file's read loop.
//
// # outputPreceded is now tracked, not hardcoded (design.md D7, R-AIP-010)
//
// Slice 1's own version of this file hardcoded emitFailure's outputPreceded
// argument to false, because no text event could exist yet to precede a
// failure with — its own comment named R-ATS-007 landing in AI-28.1.2 as
// the trigger to revisit this. That milestone is this one: run now tracks
// whether any text-block-scoped event (isOutputEvent, below) was emitted
// before a given failure, and passes that fact through, unconditionally
// (D7: "outputPreceded = any text event already emitted; ResponseStart
// does not count", S-ATS-050).
//
// # A sentinel with no terminal chunk ever observed (design.md D9, spec-silent)
//
// ai.NewCompletion rejects a zero-value ai.FinishReason. Since this file
// only ever calls it at the terminal sentinel with whatever finish reason
// a terminal chunk supplied — or the zero value, if none ever arrived —
// that same rejection is what turns "the sentinel arrived early" into the
// malformed-response terminal design.md's D9 calls for, with no separate
// branch needed to detect it.

package openaicompat

import (
	"context"
	"fmt"
	"io"
	"net/http"

	"github.com/cachicamas/backend/agent/src/ai"
)

// streamReadBufferSize bounds one Body.Read call's buffer in the producer's
// read loop — per-call memory only. The decoder's own maxFrameBytes
// (decoder.go) is the hard cap on a single frame's accumulated size,
// independent of this value.
const streamReadBufferSize = 32 * 1024

// doneSentinel is the terminal sentinel's data payload (C5): the six-byte
// literal a frame carries to signal clean termination, recognised here
// before any attempt to decode the frame as a chunk (R-ATS-012, a later
// slice's own requirement, honored already since slice 1).
const doneSentinel = "[DONE]"

// errIncompleteStream is this package's own unexported cause for a stream
// that ended — by transport EOF, by a non-cancellation read failure, or by
// the terminal sentinel arriving with no terminal chunk ever observed
// (design.md D9, spec-silent) — without ever reaching this dialect's own
// well-formed completion. Distinct from the decoder's own ErrTruncated
// (errors.go): that sentinel names an incomplete SSE frame specifically;
// this one names an incomplete dialect-level protocol regardless of
// whether SSE framing itself was clean. Wraps ai.ErrMalformedResponse with
// %w for the same reason errMalformedIdentity does.
var errIncompleteStream = fmt.Errorf("openaicompat: stream ended before reaching a well-formed completion: %w", ai.ErrMalformedResponse)

// Stream implements ai.ModelProvider (R-ATS-001): it sends req to the
// OpenAI-compatible endpoint and returns the normalized events the
// response answers with.
//
// The pre-stream contract (R-ATS-002): req is validated exactly once,
// before any I/O, in the order req.IsZero() → Translate(req) → ctx.Err()
// — see this file's own doc comment for why. Nothing is observable before
// that order clears: no network connection attempt, no channel
// allocation, no goroutine.
func (c *Client) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) {
	if req.IsZero() {
		return nil, ai.Invalid(ai.ErrEmpty, ai.At("request"))
	}

	body, err := Translate(req)
	if err != nil {
		return nil, err
	}

	if ctxErr := ctx.Err(); ctxErr != nil {
		return nil, preStreamCancellation(ctxErr)
	}

	httpReq, err := c.newRequest(ctx, body, "chat", "completions")
	if err != nil {
		return nil, preStreamTransportFailure(ctx, err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, preStreamTransportFailure(ctx, err)
	}

	out := make(chan ai.Event)
	go run(ctx, resp, out)
	return out, nil
}

// preStreamCancellation builds the pre-handover cancellation failure
// (R-ATS-002, S-ATS-006): reported only once req has already validated,
// under ai.FailureCategoryCancellation, ai.DeliveryPreStream.
func preStreamCancellation(cause error) error {
	failure, err := ai.PreStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryCancellation,
		Cause:    cause,
	})
	if err != nil {
		// FailureCategoryCancellation is always a member of the
		// vocabulary, so this can never actually fail — handled rather
		// than ignored because errcheck requires it (agenttest's fake
		// provider precedent, fake_provider.go).
		return err
	}
	return failure
}

// preStreamTransportFailure builds a pre-handover failure for a request
// that could not be built or sent at all — a request-construction or
// transport-level failure, neither of which this milestone authors a
// dedicated taxonomy for (that is AI-32.1's charter, R-ATS-024). Its
// category is ai.FailureCategoryCancellation when ctx is already done at
// the point of failure, else ai.FailureCategoryUnavailable as the closest
// existing member — a provisional mapping this package discloses here
// rather than leaving unclassified, pending AI-32.1's fuller status-based
// taxonomy.
func preStreamTransportFailure(ctx context.Context, cause error) error {
	category := ai.FailureCategoryUnavailable
	if ctx.Err() != nil {
		category = ai.FailureCategoryCancellation
	}
	failure, err := ai.PreStreamFailure(ai.FailureReport{Category: category, Cause: cause})
	if err != nil {
		return err
	}
	return failure
}

// run is the stream's one producer goroutine (R-ATS-003, R-ATS-005): reads
// the response body, drives Decoder.Feed, maps every decoded chunk through
// stream_state.go's mapperState (AI-28.1.2: response identity, text-block
// minting, the finish-reason gate), and closes out exactly once — the one
// deferred close in this package (S-ATS-019).
//
// The terminal sentinel (C5) is recognised here, before any attempt to
// decode a frame as a chunk — [DONE] is never handed to decodeChunk.
// Completion is built separately, only at the sentinel (design.md D4/D9):
// the mapper accumulates identity and finish-reason state across chunks,
// but constructs no completion event of its own.
func run(ctx context.Context, resp *http.Response, out chan<- ai.Event) {
	defer close(out)
	defer func() { _ = resp.Body.Close() }()

	stamper := &ai.Stamper{}
	decoder := NewDecoder(0)
	state := &mapperState{}
	outputPreceded := false

	buf := make([]byte, streamReadBufferSize)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			frames, feedErr := decoder.Feed(buf[:n])
			for _, frame := range frames {
				if frame.Event != defaultEventType {
					// R-ATS-017: a frame whose SSE event type is not the
					// default type is skipped unconditionally — never
					// decoded, never applied, never treated as the
					// sentinel even if its data happens to match (C5
					// states no event-type requirement, but R-ATS-017's
					// own skip rule is unconditional and this is the
					// simpler, single-branch reading of it).
					continue
				}

				if string(frame.Data) == doneSentinel {
					completion, compErr := state.buildCompletion()
					if compErr != nil {
						emitFailure(ctx, out, stamper, errIncompleteStream, outputPreceded)
						return
					}
					emit(ctx, out, stamper, completion)
					return
				}

				chunk, decodeErr := decodeChunk(frame.Data)
				if decodeErr != nil {
					emitFailure(ctx, out, stamper, errIncompleteStream, outputPreceded)
					return
				}
				if !chunk.isChunk() {
					// R-ATS-017/D3: a present-but-mismatched object
					// discriminator means this frame's JSON is not
					// recognized as a chunk at all — skipped, not applied.
					continue
				}

				events, applyErr := state.applyChunk(chunk)
				if applyErr != nil {
					emitFailure(ctx, out, stamper, applyErr, outputPreceded)
					return
				}
				for _, ev := range events {
					if !emit(ctx, out, stamper, ev) {
						return
					}
					if isOutputEvent(ev) {
						outputPreceded = true
					}
				}
			}
			if feedErr != nil {
				emitFailure(ctx, out, stamper, feedErr, outputPreceded)
				return
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				if finishErr := decoder.Finish(); finishErr != nil {
					emitFailure(ctx, out, stamper, finishErr, outputPreceded)
					return
				}
				// Clean SSE framing, but the sentinel was never observed
				// — design.md D9's sibling case at the transport-EOF edge
				// rather than the [DONE]-with-no-terminal-chunk edge.
				emitFailure(ctx, out, stamper, errIncompleteStream, outputPreceded)
				return
			}
			if ctx.Err() != nil {
				// Cancellation aborted Body.Read (AI-20.3): the sanctioned
				// loss path — close with no terminal event, through the
				// one deferred close above, never a second closing site
				// (R-ATS-005).
				return
			}
			emitFailure(ctx, out, stamper, errIncompleteStream, outputPreceded)
			return
		}
	}
}

// isOutputEvent reports whether ev counts toward R-AIP-010's PartialOutput
// discriminator (design.md D7): every text-block-scoped kind counts;
// ResponseStart never does (S-ATS-050) — it is not a normalized output
// event, only the announcement that one may follow.
func isOutputEvent(ev ai.Event) bool {
	switch ev.Kind() {
	case ai.EventKindTextBlockStart, ai.EventKindTextDelta, ai.EventKindTextBlockEnd:
		return true
	default:
		return false
	}
}

// emit stamps ev with this stream's Stamper and sends it, selecting on
// ctx.Done() alongside the send (R-ATS-005, AI-20.3) — the one emit
// helper every event this file produces goes through (design.md D6). It
// reports whether the send completed; false means cancellation won the
// race, and the caller must return without proceeding further.
func emit(ctx context.Context, out chan<- ai.Event, stamper *ai.Stamper, ev ai.Event) bool {
	stamped := stamper.Stamp(ev)
	select {
	case out <- stamped:
		return true
	case <-ctx.Done():
		return false
	}
}

// emitFailure builds and sends the D7 mid-stream terminal (design.md D7,
// R-ATS-025): category via Category(cause) when cause is one of the
// decoder's own two sentinels, else ai.FailureCategoryMalformedResponse —
// every cause this file or stream_state.go constructs on its own
// (errMalformedIdentity, errUnrecognizedFinishReason, errIncompleteStream)
// already names that category. outputPreceded is the caller's own tracked
// fact (run, above) — whether any text-block-scoped event was already
// emitted on this stream — never derived here.
func emitFailure(ctx context.Context, out chan<- ai.Event, stamper *ai.Stamper, cause error, outputPreceded bool) {
	category, ok := Category(cause)
	if !ok {
		category = ai.FailureCategoryMalformedResponse
	}
	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: category, Cause: cause}, outputPreceded)
	if err != nil {
		// category is always a valid member here, so this can never
		// actually fail — handled rather than ignored (errcheck).
		return
	}
	ev, err := ai.ErrorEvent(failure)
	if err != nil {
		return
	}
	emit(ctx, out, stamper, ev)
}
