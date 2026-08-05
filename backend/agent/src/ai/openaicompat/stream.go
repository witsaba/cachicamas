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
//
// # Pre-decode checks now gate the carrier (AI-28.6, R-ATS-023/024)
//
// Between the response arriving and the carrier being created, Stream now
// runs two checks — design.md D1's own numbered step 6, strictly before
// step 7's channel creation — so both are pre-handover, ai.PreStreamFailure
// refusals (R-ATS-025's own handover rule), never something observed on the
// carrier: a failure status routes through mapResponse (failure_map.go,
// AI-32.1) before any byte reaches the decoder (R-ATS-024), and a success
// response whose Content-Type is not the streaming media type is refused
// with a bounded body excerpt before any byte reaches the decoder
// (R-ATS-023). Both checks consume machinery this file does not own or
// widen: mapResponse is AI-32.1's own status-to-category mapper, and the
// excerpt reuses capture.go's own captureBody/captureLimit rather than a
// second, independent bound.
//
// # An open text block closes before a mid-stream terminal failure (AI-28.7, design.md D8)
//
// design.md D8: a stream this file terminates in a mid-stream failure must
// still satisfy AI-14.4's own ai.CheckStream — in particular its
// no-unterminated-block invariant (AI-16) — exactly like a stream that
// terminates cleanly already does. Every pre-terminal failure path (a
// truncated transcript, an invalid-JSON frame, an out-of-enum
// finish_reason, and every other cause stream_state.go's own applyChunk
// can return) can leave a text block open: R-ATS-008's own minting logic
// (stream_state.go) only closes a block once a chunk both sets
// terminalSeen AND that same chunk's own applyChunk call returns
// successfully — a chunk that FAILS before reaching that point leaves the
// block exactly as open as it was the instant before. run tracks that fact
// itself, from what it has actually sent on out — never from
// stream_state.go's own internal bookkeeping, which can legitimately mark
// a block open a few instructions before the corresponding event is ever
// handed to emit — see run's own blockOpen local and emitFailure's own doc
// comment.

package openaicompat

import (
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
)

// streamReadBufferSize bounds one Body.Read call's buffer in the producer's
// read loop — per-call memory only. The decoder's own maxFrameBytes
// (decoder.go) is the hard cap on a single frame's accumulated size,
// independent of this value.
const streamReadBufferSize = 32 * 1024

// doneSentinel is the terminal sentinel's data payload — C5, cited here
// together with its posture stated exactly as C5 states it
// (verify-report W2, R-ATS-012, S-ATS-047): for Chat Completions this
// sentinel is prose-documented at the pinned commit — presupposed by the
// include_usage description — and is NOT a schema constant; the only
// `enum: - "[DONE]"` at that commit belongs to the Assistants DoneEvent.
// It is therefore a dialect-conventional recognition carrying a
// fixture-pin obligation, discharged by this package's own committed
// transcript fixtures, not a schema-derived fact. The six-byte literal a
// frame carries to signal clean termination, recognised here before any
// attempt to decode the frame as a chunk (R-ATS-012, a later slice's own
// requirement, honored already since slice 1).
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

// errMalformedChunkJSON is this package's own unexported cause for a
// dispatched frame whose data is not syntactically valid JSON for its
// declared, known type (R-ATS-021, S-ATS-080) — distinct from
// errIncompleteStream, which names the stream itself ending early, not a
// single frame's own payload being broken while the stream is still very
// much in progress. Wraps ai.ErrMalformedResponse with %w for the same
// reason errMalformedIdentity does.
//
// # Corrective: previously misnamed as errIncompleteStream (slice 6)
//
// Slices 1/2 reused errIncompleteStream for this branch — functionally
// harmless, since Category(cause) does not recognize either identity and
// emitFailure's own fallback assigns ai.FailureCategoryMalformedResponse
// regardless (S-ATS-080/082 both already passed under the old identity) —
// but semantically confusing: a reader debugging "why did errIncompleteStream
// fire here" would find no incomplete STREAM, only one malformed FRAME.
// Corrected alongside R-ATS-021's own implementation work, disclosed as a
// zero-behavior-change clarity fix.
var errMalformedChunkJSON = fmt.Errorf("openaicompat: a dispatched frame's data was not valid JSON for its declared type: %w", ai.ErrMalformedResponse)

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

	// R-ATS-024: a failure status routes to AI-32's own failure mapping
	// before any byte reaches the decoder. mapResponse returns nil for a
	// 2xx response without touching resp.Body, so a success response falls
	// through to the content-type check below untouched.
	if failure := mapResponse(resp, time.Now()); failure != nil {
		return nil, failure
	}

	// R-ATS-023: a success response whose Content-Type is not the
	// streaming media type is refused before any byte reaches the decoder.
	if !isStreamContentType(resp.Header.Get("Content-Type")) {
		return nil, refuseNonStreamContentType(resp)
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

// streamContentType is the streaming media type this dialect's own request
// declares via its Accept header (request.go's newRequest): the one
// Content-Type a success response must carry for its body to ever reach the
// decoder (R-ATS-023).
const streamContentType = "text/event-stream"

// isStreamContentType reports whether contentType names the streaming media
// type, tolerating an optional parameter (a charset, for example) and case
// on the type and subtype (R-ATS-023, S-ATS-087) — mime.ParseMediaType
// already lower-cases the returned media type and separates any parameters,
// so no hand-rolled case-folding or parameter-stripping is needed here. A
// missing or unparseable value is never treated as a match — refused, never
// decoded optimistically (S-ATS-090).
func isStreamContentType(contentType string) bool {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return false
	}
	return mediaType == streamContentType
}

// nonStreamContentType is R-ATS-023's typed cause for a success response
// whose Content-Type is not the streaming media type. Unlike capture.go's
// own capturedBody — whose Error() deliberately never reproduces its bytes
// (R-AEM-016, AI-32's own, different obligation) — this type's Error() text
// embeds the excerpt itself, because R-ATS-023 requires a caller reading
// the terminal failure's rendered text to see it; ai.Failure's own Error()
// stays the frozen AI-19 form and never reproduces a wrapped cause's text
// (R-AIP-009), so this cause is what a caller unwraps to reach it. The
// excerpt is bounded by capture.go's own captureLimit (captureBody, below —
// AI-32.1's bounded-capture machinery, reused rather than a second,
// independent bound, S-ATS-089). This type never reads the request's own
// credential — it is built solely from the response's Content-Type header
// and body — so it cannot reproduce one (R-ATS-023's "MUST NOT reproduce
// credential material").
//
// Credential-scan posture (S-ATS-089 rev 4): the credential-scan guard's
// designed scope is external-package (`package openaicompat_test`) test
// files only, and every test file in this milestone is internal-package by
// load-bearing necessity — the error-mapping milestone plants
// above-threshold credential sentinels in internal test files precisely
// because the guard cannot see them there, so widening the guard would
// break the build on those deliberate plants. A package-internal sweep
// that understands deliberate plants is possible future hardening, owned
// by a future guard change; recorded here so the gap is disclosed, never
// silent.
type nonStreamContentType struct {
	contentType string
	excerpt     []byte
}

// Error renders the refusal together with the bounded body excerpt
// (R-ATS-023, S-ATS-086, S-ATS-088).
func (e *nonStreamContentType) Error() string {
	return fmt.Sprintf("openaicompat: response content type %q is not the streaming media type %q; body excerpt: %s", e.contentType, streamContentType, e.excerpt)
}

// refuseNonStreamContentType builds R-ATS-023's pre-handover failure for a
// success response whose Content-Type is not the streaming media type,
// capturing a bounded excerpt of resp's body through capture.go's own
// captureBody — which also closes the body, so no byte reaches the decoder.
func refuseNonStreamContentType(resp *http.Response) error {
	captured := captureBody(resp.Body)
	failure, err := ai.PreStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryMalformedResponse,
		Cause:    &nonStreamContentType{contentType: resp.Header.Get("Content-Type"), excerpt: captured.bytes()},
	})
	if err != nil {
		// FailureCategoryMalformedResponse is always a member of the
		// vocabulary, so this can never actually fail — handled rather
		// than ignored (errcheck), the same posture this file's other
		// pre-stream builders already take.
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
// The terminal sentinel (C5, prose-documented and dialect-conventional —
// see doneSentinel's own doc comment for the posture stated exactly as C5
// states it) is recognised here, before any attempt to decode a frame as
// a chunk — [DONE] is never handed to decodeChunk. Completion is built
// separately, only at the sentinel (design.md D4/D9):
// the mapper accumulates identity and finish-reason state across chunks,
// but constructs no completion event of its own.
func run(ctx context.Context, resp *http.Response, out chan<- ai.Event) {
	defer close(out)
	defer func() { _ = resp.Body.Close() }()

	stamper := &ai.Stamper{}
	decoder := NewDecoder(0)
	state := &mapperState{}
	outputPreceded := false
	// blockOpen mirrors, from run's own vantage point, whether a
	// TextBlockStart has been handed to emit with no matching
	// TextBlockEnd handed to emit yet (design.md D8). It is deliberately
	// NOT read from state.blockOpen: stream_state.go's own field can be
	// true for a few instructions before the corresponding event is ever
	// appended to applyChunk's returned slice — let alone sent — so
	// run's own copy, updated only once emit actually confirms a send,
	// is the one true account of what the CARRIER has observed.
	blockOpen := false

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
					// sentinel even if its data happens to match (C5, the
					// prose-documented, dialect-conventional sentinel —
					// see doneSentinel's own doc comment — states no
					// event-type requirement, but R-ATS-017's own skip
					// rule is unconditional and this is the simpler,
					// single-branch reading of it).
					continue
				}

				if string(frame.Data) == doneSentinel {
					completion, compErr := state.buildCompletion()
					if compErr != nil {
						emitFailure(out, stamper, errIncompleteStream, outputPreceded, blockOpen)
						return
					}
					emit(ctx, out, stamper, completion)
					return
				}

				// R-AEM-010 (AI-32.2): an in-band error frame mid-stream
				// terminates the stream with a typed mid-stream failure.
				// Detected by JSON shape — a frame whose payload is the
				// wrapped {"error": {…}} vendor body — BEFORE decoding as
				// a chunk, so the frame's unknown object discriminator
				// (R-ATS-017 would skip it) cannot silently swallow a
				// terminal-error payload.
				if isInBandErrorFrame(frame.Data) {
					// An open block must close before the terminal error,
					// exactly like every other mid-stream failure path
					// (design.md D8, AI-28.7): the producer-side closure
					// guarantee is uniform, not per-cause.
					if blockOpen {
						if end, err := ai.NewTextBlockEnd(textBlockIndex); err == nil {
							if !emit(ctx, out, stamper, end) {
								return
							}
						}
					}
					failure := failureFromErrorFrame(frame.Data, outputPreceded)
					if failure != nil {
						ev, err := ai.ErrorEvent(failure)
						if err == nil {
							emit(ctx, out, stamper, ev)
						}
					}
					return
				}

				chunk, decodeErr := decodeChunk(frame.Data)
				if decodeErr != nil {
					emitFailure(out, stamper, errMalformedChunkJSON, outputPreceded, blockOpen)
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
					emitFailure(out, stamper, applyErr, outputPreceded, blockOpen)
					return
				}
				for _, ev := range events {
					if !emit(ctx, out, stamper, ev) {
						// AI-32.3: even when an in-flight emit loses the
						// ctx.Done() race for a normal mid-stream event,
						// the typed terminal failure must remain
						// observable (S-AEM-051/052). Surface ctx.Err()
						// through emitFailure — its try-send onto the
						// carrier's single-slot buffer makes the terminal
						// event land without further blocking, so a
						// consumer that has reached drainAll sees the
						// typed failure alongside the eventual channel
						// close.
						if ctxErr := ctx.Err(); ctxErr != nil {
							emitFailure(out, stamper, ctxErr, outputPreceded, blockOpen)
						}
						return
					}
					if isOutputEvent(ev) {
						outputPreceded = true
					}
					switch ev.Kind() {
					case ai.EventKindTextBlockStart:
						blockOpen = true
					case ai.EventKindTextBlockEnd:
						blockOpen = false
					}
				}
			}
			if feedErr != nil {
				emitFailure(out, stamper, feedErr, outputPreceded, blockOpen)
				return
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				if finishErr := decoder.Finish(); finishErr != nil {
					emitFailure(out, stamper, finishErr, outputPreceded, blockOpen)
					return
				}
				// Clean SSE framing, but the sentinel was never observed
				// — design.md D9's sibling case at the transport-EOF edge
				// rather than the [DONE]-with-no-terminal-chunk edge.
				emitFailure(out, stamper, errIncompleteStream, outputPreceded, blockOpen)
				return
			}
			if ctxErr := ctx.Err(); ctxErr != nil {
				// AI-32.3 / R-AEM-014: a context cancellation OR deadline
				// expiring mid-stream emits a typed terminal failure
				// (FailureCategoryCancellation or FailureCategoryTimeout
				// respectively, via categorizeStreamError →
				// midStreamFailureFrom). This is the deliberate
				// AI-32.3-era evolution from the AI-20.3 "no terminal
				// event on cancellation" discipline: S-AEM-051/052
				// require the carrier to surface the typed failure so a
				// caller can distinguish Timeout from Cancellation
				// without inspecting Error() text. AI-20.3's "no leak,
				// one closing site, every send selects" posture is
				// otherwise unchanged — emitFailure still selects on
				// ctx.Done() alongside the send, and the deferred close
				// above remains the only channel-close site.
				emitFailure(out, stamper, ctxErr, outputPreceded, blockOpen)
				return
			}
			emitFailure(out, stamper, errIncompleteStream, outputPreceded, blockOpen)
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
// R-ATS-025): category via midStreamFailureFrom — which calls
// categorizeStreamError first (decoder sentinels via Category(), then
// context / net.Error / default-Unavailable branches per design D6) —
// outputPreceded is the caller's own tracked fact (run, above) — whether
// any text-block-scoped event was already emitted on this stream —
// never derived here.
//
// Every cause this file or stream_state.go constructs on its own
// (errMalformedIdentity, errUnrecognizedFinishReason,
// errIncompleteStream) lands on the decoder-sentinel branch of
// midStreamFailureFrom (MalformedResponse), matching the prior
// behaviour; the new ctxErr / deadline / cancel causes land on their
// own typed branches (Cancellation, Timeout). The exhaustive
// categorization is why no fallback to ai.FailureCategoryMalformedResponse
// is needed anymore — categorizeStreamError returns a known category
// for every non-nil input.
//
// blockOpen is run's own tracked fact, too (design.md D8, AI-28.7):
// whether a text block is currently open on the carrier. Every one of
// run's own seven call sites can reach this function with a block
// genuinely open — truncation, an invalid-JSON frame, an out-of-enum
// finish_reason, and every other cause applyChunk can return all leave
// a block open exactly when the failing chunk arrives before a
// terminal chunk ever closes it. When blockOpen is true, this function
// sends the closing TextBlockEnd — through a bounded-wait send (see
// below) — before building or sending the terminal ErrorEvent, so a
// stream this function terminates still satisfies ai.CheckStream's
// no-unterminated-block invariant (AI-16) exactly like a
// cleanly-terminated one already does.
//
// # Terminal failure is observable even on cancellation (AI-32.3, R-AEM-014)
//
// AI-32.3 deliberately diverges from AI-20.3's "no terminal event on
// cancellation" discipline: a typed terminal failure on deadline or
// cancellation MUST be observable (S-AEM-051/052) so a caller can
// distinguish Timeout from Cancellation without inspecting Error()
// text. emit()'s select-on-ctx.Done() pattern races with that
// guarantee when ctx is already cancelled — both send and ctx.Done()
// are then "ready" and Go's select picks randomly, so the terminal
// event is sometimes dropped even when the consumer is actively
// draining. This function uses bounded-wait sends (select on send
// AND a short timeout — NOT ctx.Done()) instead: when the consumer is
// ready (the S-ATS-017 drain discipline), the send completes in
// microseconds; when the consumer has stopped draining, the timer
// fires and the event is dropped. run's deferred close still signals
// termination in that case, so no goroutine leak can occur from this
// path.
func emitFailure(out chan<- ai.Event, stamper *ai.Stamper, cause error, outputPreceded, blockOpen bool) {
	if blockOpen {
		// textBlockIndex is this package's own nonzero constant — the
		// only rule NewTextBlockEnd checks — so this can never actually
		// fail; handled rather than ignored because errcheck requires it
		// (the same posture this file's other builders already take,
		// e.g. preStreamCancellation above). Falling through on an
		// unreachable error still reports the terminal failure itself,
		// which matters more than a close this package can never
		// actually fail to construct.
		if end, err := ai.NewTextBlockEnd(textBlockIndex); err == nil {
			stamped := stamper.Stamp(end)
			select {
			case out <- stamped:
				// delivered
			case <-time.After(emitFailureSendBound):
				// consumer not draining within bound; run's deferred
				// close still signals termination.
			}
		}
	}

	failure := midStreamFailureFrom(cause, outputPreceded)
	if failure == nil {
		// categorizeStreamError returns ok=false only for nil input, and
		// every caller of this function supplies a non-nil cause; reaching
		// here would be this package's own defect — the same posture
		// every other emitFailure-adjacent call already takes.
		return
	}
	ev, err := ai.ErrorEvent(failure)
	if err != nil {
		return
	}
	stamped := stamper.Stamp(ev)
	select {
	case out <- stamped:
		// delivered — AI-32.3's typed-terminal guarantee observed
	case <-time.After(emitFailureSendBound):
		// consumer not draining within bound; run's deferred close
		// still signals termination to a consumer that has stopped.
	}
}

// emitFailureSendBound is the upper bound on how long emitFailure's
// block-close and terminal-event sends wait for the consumer to be
// ready to receive. Generous enough to absorb the consumer's typical
// drain-loop iteration latency (microseconds), short enough that a
// goroutine leak in any code path is bounded. The run() deferred
// close always fires after this function returns, regardless of
// whether the consumer received the terminal event.
const emitFailureSendBound = 50 * time.Millisecond
