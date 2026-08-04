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
// # Text mapping is not this file's job yet
//
// This slice maps only what a minimal transcript needs to establish
// identity and reach a terminal event: no choice-0 content string is read
// or emitted as a text delta anywhere in this file (R-ATS-007…010 land in
// AI-28.1.2, the next slice). A transcript this file drains that happens
// to carry content deltas produces no text event for them — that gap
// closes in the next slice, not here.
//
// # A first chunk with a present-but-empty id or model (design.md D7, N-7b)
//
// ai.NewResponseStart requires both the response id and the served model
// to be non-empty. When the first chunk supplies one present but empty (or
// omits it — the same zero-value Go string either way), that constructor's
// own error is converted here into a malformed-response terminal
// (errMalformedIdentity, wrapping ai.ErrMalformedResponse) — never minted,
// defaulted or substituted to dodge the constructor (R-ATS-004, S-ATS-015).
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
	"encoding/json"
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

// errMalformedIdentity is this package's own unexported cause for a first
// chunk whose id or model surfaced, through ai.NewResponseStart's own
// constructor error, as present but empty (design.md D7, N-7b): both
// fields are required non-empty by that constructor. Wraps
// ai.ErrMalformedResponse with %w so errors.Is(err, ai.ErrMalformedResponse)
// holds; unexported so S-ART-054's allowlist stays untouched — no new
// exported sentinel (R-ATS-025).
var errMalformedIdentity = fmt.Errorf("openaicompat: first chunk's response identity was malformed: %w", ai.ErrMalformedResponse)

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

// wireChunk is the minimal, slice-1-scoped shape this file reads from a
// decoded frame: only the fields R-ATS-003…005 need to drain a minimal
// transcript — the response identity (C1) and choice 0's finish reason
// (C2). It is deliberately not chunk.go's eventual byte-preserving
// structure: no delta.content is read here at all (that lands in
// AI-28.1.2, R-ATS-007), and finish_reason is read leniently via
// ai.NormalizeFinishReason rather than gated against C2's enum (D5's
// raw-string-strict gate is R-ATS-010, also AI-28.1.2).
type wireChunk struct {
	ID      string       `json:"id"`
	Model   string       `json:"model"`
	Choices []wireChoice `json:"choices"`
}

// wireChoice is choice 0's minimal shape this file reads: only
// finish_reason, via a pointer so JSON null and an absent key both decode
// to nil, matching C2's own null-until-terminal behavior.
type wireChoice struct {
	FinishReason *string `json:"finish_reason"`
}

// producerState is the identity/terminal tracking this slice's minimal run
// loop needs — not AI-28.1.2's full mapper state machine (stream_state.go,
// a later slice, lands the block-minting, terminal-window and
// violation-table state).
type producerState struct {
	identityEstablished bool
	finishReason        ai.FinishReason
}

// run is the stream's one producer goroutine (R-ATS-003, R-ATS-005): reads
// the response body, drives Decoder.Feed, maps a minimal transcript to
// ResponseStart/Completion/ErrorEvent, and closes out exactly once — the
// one deferred close in this package (S-ATS-019).
func run(ctx context.Context, resp *http.Response, out chan<- ai.Event) {
	defer close(out)
	defer func() { _ = resp.Body.Close() }()

	stamper := &ai.Stamper{}
	decoder := NewDecoder(0)
	state := &producerState{}

	buf := make([]byte, streamReadBufferSize)
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			frames, feedErr := decoder.Feed(buf[:n])
			for _, frame := range frames {
				sentinel, ev, hasEvent, applyErr := state.applyFrame(frame)
				if applyErr != nil {
					emitFailure(ctx, out, stamper, applyErr)
					return
				}
				if hasEvent && !emit(ctx, out, stamper, ev) {
					return
				}
				if sentinel {
					completion, compErr := state.buildCompletion()
					if compErr != nil {
						emitFailure(ctx, out, stamper, errIncompleteStream)
						return
					}
					emit(ctx, out, stamper, completion)
					return
				}
			}
			if feedErr != nil {
				emitFailure(ctx, out, stamper, feedErr)
				return
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				if finishErr := decoder.Finish(); finishErr != nil {
					emitFailure(ctx, out, stamper, finishErr)
					return
				}
				// Clean SSE framing, but the sentinel was never observed
				// — design.md D9's sibling case at the transport-EOF edge
				// rather than the [DONE]-with-no-terminal-chunk edge.
				emitFailure(ctx, out, stamper, errIncompleteStream)
				return
			}
			if ctx.Err() != nil {
				// Cancellation aborted Body.Read (AI-20.3): the sanctioned
				// loss path — close with no terminal event, through the
				// one deferred close above, never a second closing site
				// (R-ATS-005).
				return
			}
			emitFailure(ctx, out, stamper, errIncompleteStream)
			return
		}
	}
}

// applyFrame maps one decoded frame under this slice's minimal shape: the
// terminal sentinel, this dialect's own identity-establishment chunk, and
// finish-reason capture. It returns at most one event — this slice emits
// only ResponseStart from a frame; Completion is built separately at the
// sentinel (design.md D1: "on the sentinel, emit Completion").
func (s *producerState) applyFrame(frame Frame) (sentinel bool, ev ai.Event, hasEvent bool, err error) {
	if string(frame.Data) == "[DONE]" {
		return true, ai.Event{}, false, nil
	}

	var chunk wireChunk
	if jsonErr := json.Unmarshal(frame.Data, &chunk); jsonErr != nil {
		return false, ai.Event{}, false, errIncompleteStream
	}

	if !s.identityEstablished {
		started, startErr := ai.NewResponseStart(chunk.ID, chunk.Model)
		if startErr != nil {
			return false, ai.Event{}, false, errMalformedIdentity
		}
		s.identityEstablished = true
		ev, hasEvent = started, true
	}

	if len(chunk.Choices) > 0 && chunk.Choices[0].FinishReason != nil {
		s.finishReason = ai.NormalizeFinishReason(*chunk.Choices[0].FinishReason)
	}

	return false, ev, hasEvent, nil
}

// buildCompletion constructs the sentinel-triggered completion from
// whatever finish reason this slice's minimal loop captured, with usage
// left wholly absent (R-ATS-015…016 land in AI-28.3, a later slice). An
// error here means no terminal chunk was ever observed before the
// sentinel arrived — design.md D9's spec-silent shape, folded into
// errIncompleteStream by run.
func (s *producerState) buildCompletion() (ai.Event, error) {
	return ai.NewCompletion(s.finishReason, ai.Usage{})
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
// every cause this file constructs on its own (errMalformedIdentity,
// errIncompleteStream) already names that category. outputPreceded is
// always false in this slice: no text event exists yet to precede a
// failure with (R-ATS-007 lands in AI-28.1.2); ResponseStart never counts
// either way (S-ATS-050).
func emitFailure(ctx context.Context, out chan<- ai.Event, stamper *ai.Stamper, cause error) {
	category, ok := Category(cause)
	if !ok {
		category = ai.FailureCategoryMalformedResponse
	}
	failure, err := ai.MidStreamFailure(ai.FailureReport{Category: category, Cause: cause}, false)
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
