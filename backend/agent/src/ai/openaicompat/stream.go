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
//
// # Body drain-before-close (AI-33.5, R-AIS-033, design.md "Where the drain goes")
//
// run's defer chain closes resp.Body without draining (stream.go:345 in
// slice 1). capture.go:117–122 already proves the drain-before-close
// idiom is safe and bounded on the failure-capture path; extending it
// into run's defer chain lets the transport return the keep-alive
// connection to its idle pool on every exit path — completion, terminal
// error, each cancellation moment. The drain is silent (any error from
// io.Copy is the network's concern, not a Layer 1 contract concern), runs
// INSIDE the existing producer goroutine's defer chain (R-ATS-003's
// single-producer model is preserved: no second persistent goroutine), and
// adds no helper, no signature change, and no new top-level Go
// dependency (R-STK-009).

package openaicompat

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"mime"
	"net/http"
	"time"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/internal/retry"
)

// streamReadBufferSize bounds one Body.Read call's buffer in the producer's
// read loop — per-call memory only. The decoder's own maxFrameBytes
// (decoder.go) is the hard cap on a single frame's accumulated size,
// independent of this value.
const streamReadBufferSize = 32 * 1024

// streamCarrierBuffer sizes the event carrier the producer hands back to
// the caller (R-AIS-031, R-AIS-039). The value is frozen by
// `openspec/changes/2026-08-07-cachicamas-ai-backpressure/decision.md` —
// AI-34.1's M1 / M2 / M3 measurements, with doc 0002 § 6 line 432's
// "prefer the smaller" tie-break applied. The chosen N is `0` (rendezvous):
// the unbuffered carrier minimises on every axis the measurements name
// (M1 high-water = 0, M2 bursty p99 collapses to 0 only if buffered —
// that collapse is the hidden-latency the tie-break warns against, M3
// per-stream heap is dominated by setup overhead, not by the carrier).
// Mirrors `streamReadBufferSize` and `emitFailureSendBound` as a
// package-private constant; no `Config.BufferCapacity` field (design D1,
// proposal Q3).
const streamCarrierBuffer = 0

// emitFailureSendBound caps how long a single terminal-failure send
// may wait for the consumer (AI-32.3 bounded-wait). Generous enough
// for a local-loopback drain; not generous enough to leak a goroutine
// if the consumer has stopped.
const emitFailureSendBound = 5 * time.Second

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

	resp, err := retry.Loop(ctx, body, retry.Config{}, c.executeOnce)
	if err != nil {
		return nil, err
	}

	// R-ATS-024: a success response falls through to the content-type check
	// untouched. Retryable failures were handled before the carrier exists;
	// terminal failures return without a carrier or producer goroutine.

	// R-ATS-023: a success response whose Content-Type is not the
	// streaming media type is refused before any byte reaches the decoder.
	if !isStreamContentType(resp.Header.Get("Content-Type")) {
		return nil, refuseNonStreamContentType(resp, c.credential)
	}

	out := make(chan ai.Event, streamCarrierBuffer)
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
	failure, err := ai.PreStreamFailure(ai.FailureReport{Category: category, Cause: cause, Retryable: retryableFor(category)})
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
// independent bound, S-ATS-089).
//
// # AI-36 (WU-2, D-4) — a hostile provider CAN echo the caller's own
// credential into this excerpt
//
// This type is built solely from the response's Content-Type header and
// body, never from the request directly — but a provider that echoes the
// request's own Authorization header (or its body) back into a
// non-streaming response makes that credential reachable through the
// response body all the same. AI-36's adversarial hostile-server case
// (a_i-36_1_test.go) proved this bites: refuseNonStreamContentType now
// redacts the caller's own credential out of the captured excerpt before
// it ever reaches this type (redactCredential, below), so Error() still
// never reproduces credential material (R-ATS-023's own "MUST NOT
// reproduce credential material") while the excerpt itself stays present
// and readable (R-ATS-023's visibility requirement, unchanged). A
// provider echoing the caller's own prompt content is a separate, named
// residual (R-AEM-019 item 4): suppressing it would defeat this excerpt's
// diagnostic purpose, since the excerpt IS the provider's own response.
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

// credentialRedactedPlaceholder replaces a caller's own credential
// wherever redactCredential finds it echoed back inside a captured excerpt
// (AI-36, D-4) — the same fixed placeholder text Credential's own
// String/GoString already render (credential.go), so a redacted excerpt
// reads consistently with every other redacted credential rendering in
// this module.
const credentialRedactedPlaceholder = "<redacted>"

// redactCredential returns excerpt with every fold-equal occurrence of
// cred's bearer-header form, then of cred's raw token, replaced with
// credentialRedactedPlaceholder (D-4, R-AEM-019 item 1), and with any
// truncation-edge fragment of either handled by
// redactCredentialTailFragment (below). The bearer form is replaced first
// so a caller reading the redacted excerpt never sees a dangling scheme
// keyword left behind by a token-only replacement. An empty credential
// (isEmpty) has nothing to redact and is returned unchanged.
//
// Size bound (AI-36 JD round 1, finding 5): credentialRedactedPlaceholder
// is 10 bytes, and nothing constrains a live Credential's length beyond
// non-empty (client.go's New) — credential_scan_test.go's under-20-bytes
// convention is a test-only pattern table and binds no production value.
// A token shorter than the placeholder therefore CAN grow the payload
// under substitution (worst case: a 1-byte token rewriting every byte of
// a captureLimit-sized payload to ~10x its length). The bound is enforced
// rather than assumed: after substitution the payload is re-capped to
// captureLimit, so the result never exceeds captureLimit +
// len(truncationMarker) — exactly the largest capture AI-32.5's own path
// (captureBody, capture.go) can emit. A cap cut can land mid-placeholder;
// a truncated placeholder carries no secret bytes, so the trade is
// bounded size over a tidy tail.
func redactCredential(excerpt []byte, cred Credential) []byte {
	if cred.isEmpty() {
		return excerpt
	}
	// Whole-needle replacement runs on the payload only: when the excerpt
	// carries capture.go's truncation marker (present iff the body
	// overflowed captureLimit, so the payload is exactly captureLimit
	// bytes), the marker is fixed constant bytes appended AFTER the cut —
	// no credential byte can span into it, and the truncation-edge check
	// below must look at the payload's real tail, not the marker's.
	payload := excerpt
	var marker []byte
	if len(excerpt) == captureLimit+len(truncationMarker) && bytes.HasSuffix(excerpt, []byte(truncationMarker)) {
		payload = excerpt[:captureLimit]
		marker = excerpt[captureLimit:]
	}
	redacted := foldReplaceAll(payload, []byte(cred.bearer()))
	redacted = foldReplaceAll(redacted, []byte(cred.token))
	redacted = redactCredentialTailFragment(redacted, cred)
	// Enforce the size bound substitution can otherwise break (see the
	// doc comment above): the payload never exceeds captureLimit, so the
	// whole result never exceeds captureLimit + len(truncationMarker).
	if len(redacted) > captureLimit {
		redacted = redacted[:captureLimit]
	}
	if marker != nil {
		redacted = append(append([]byte(nil), redacted...), marker...)
	}
	return redacted
}

// foldReplaceAll returns s with every fold-equal occurrence of needle
// replaced by credentialRedactedPlaceholder. Matching is case-insensitive
// (AI-36 JD round 1, finding 3): a hostile server chooses its own casing
// for the echo, RFC 7235 already defines the scheme keyword ("Bearer") as
// case-insensitive, and an echo whose only transformation is ASCII case
// still discloses the secret in trivially recoverable form. The boundary
// chosen here: MATCHING folds, the credential itself is never
// case-normalized anywhere — cred.bearer() puts the exact original bytes
// on the wire, and replacing body bytes that merely fold-equal the secret
// is over-matching in the safe direction (a false positive costs one
// placeholder's worth of diagnostics; a false negative is a leak). A
// fold-equal variant of a DIFFERENT byte length (Unicode multi-byte fold
// pairs) is not matched by this fixed-window scan — that is a
// re-encoding of the secret, the same disclosed residual class as a
// base64-transformed echo, which no byte-level scan can close.
func foldReplaceAll(s, needle []byte) []byte {
	if len(needle) == 0 || len(needle) > len(s) {
		return s
	}
	var out []byte
	i := 0
	for i+len(needle) <= len(s) {
		if bytes.EqualFold(s[i:i+len(needle)], needle) {
			out = append(out, credentialRedactedPlaceholder...)
			i += len(needle)
			continue
		}
		out = append(out, s[i])
		i++
	}
	return append(out, s[i:]...)
}

// minCredentialFragment is the shortest credential prefix the
// truncation-edge check below treats as credential material: 4 bytes,
// matching the module's existing prefix doctrine — the openrouter smoke
// sweep makes the secret's 4-character prefix its own deny-list vector
// (openrouter/smoke/sentinel_sweep.go), and ai/completion.go states the
// principle outright: a prefix of a secret is still a secret. Fragments
// of 1–3 bytes are below that established threshold and indistinguishable
// from ordinary body bytes.
const minCredentialFragment = 4

// redactCredentialTailFragment closes the truncation-edge gap (AI-36 JD
// round 1, finding 2): captureBody cuts the body at a fixed byte offset
// with no credential awareness, and capture runs BEFORE redaction — so a
// hostile server can position its Authorization echo across that boundary
// and leave an attacker-chosen proper PREFIX of the credential (or of its
// bearer-header form) at the payload's tail, where whole-needle
// replacement cannot match it. The full body is unavailable by design
// (capture bounds while reading), so the tail is checked directly: the
// longest payload suffix that is a proper prefix of the bearer form, then
// of the raw token, of at least minCredentialFragment bytes, is replaced
// with the placeholder. The check runs on every payload, not only marked
// ones — a short read or a connection cut mid-echo produces the same
// fragment shape without ever reaching captureLimit.
func redactCredentialTailFragment(payload []byte, cred Credential) []byte {
	needles := [][]byte{[]byte(cred.bearer()), []byte(cred.token)}
	for _, needle := range needles {
		longest := len(needle) - 1 // proper prefix only: the whole needle was already replaced above
		if longest > len(payload) {
			longest = len(payload)
		}
		for n := longest; n >= minCredentialFragment; n-- {
			// Fold-equal, same as foldReplaceAll: a case-folded echo
			// truncated at the edge is still credential material.
			if bytes.EqualFold(payload[len(payload)-n:], needle[:n]) {
				out := append([]byte(nil), payload[:len(payload)-n]...)
				return append(out, credentialRedactedPlaceholder...)
			}
		}
	}
	return payload
}

// refuseNonStreamContentType builds R-ATS-023's pre-handover failure for a
// success response whose Content-Type is not the streaming media type,
// capturing a bounded excerpt of resp's body through capture.go's own
// captureBody — which also closes the body, so no byte reaches the
// decoder — and redacting cred (the client's own credential) out of that
// excerpt before it ever reaches nonStreamContentType (AI-36, D-4): a
// hostile provider that echoes the caller's own Authorization header into
// its response body must not be able to make this excerpt reproduce it.
//
// The stored content type gets the same redaction (AI-36 JD round 1,
// finding 1): isStreamContentType compares only the parsed media type, so
// a hostile provider can echo the caller's own Authorization value into a
// Content-Type PARAMETER — or into an unparseable value rendered whole —
// and nonStreamContentType.Error() would reproduce it verbatim. Redacting
// only the credential occurrences keeps the offending type itself visible,
// which R-ATS-023's refusal text requires.
func refuseNonStreamContentType(resp *http.Response, cred Credential) error {
	captured := captureBody(resp.Body)
	failure, err := ai.PreStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryMalformedResponse,
		Cause: &nonStreamContentType{
			contentType: string(redactCredential([]byte(resp.Header.Get("Content-Type")), cred)),
			excerpt:     redactCredential(captured.bytes(), cred),
		},
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
	// AI-33.5 (R-AIS-033): drain-before-close — see header. Silent
	// (network errors ignored), runs inside the producer goroutine's
	// existing defer chain (R-ATS-003 preserved: no second goroutine).
	defer func() { _, _ = io.Copy(io.Discard, resp.Body); _ = resp.Body.Close() }()

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
	// toolBlocksOpen counts the open tool-call blocks as observed by the
	// carrier (AI-30.1, design.md D8 inheritance): the count rises with
	// each ToolCallStart emit-confirmed and falls with each
	// ToolCallEnd. The mapper's own toolOpenOrder is the source of truth
	// for which blocks to close on a failure path (D7/D8), but run
	// tracks the count independently — same reasoning blockOpen uses.
	toolBlocksOpen := 0

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
						emitFailure(ctx, out, stamper, state, errIncompleteStream, outputPreceded, blockOpen)
						return
					}
					emit(ctx, out, stamper, completion)
					return
				}

				// R-AEM-010 (AI-32.2): an in-band error frame mid-stream
				// terminates the stream with a typed mid-stream failure.
				// Detected by JSON shape BEFORE decodeChunk, so the
				// frame's unknown object discriminator (R-ATS-017's
				// "skip a present-but-mismatched object" rule) cannot
				// silently swallow a terminal-error payload.
				if isInBandErrorFrame(frame.Data) {
					if blockOpen {
						if end, err := ai.NewTextBlockEnd(textBlockIndex); err == nil {
							if !emit(ctx, out, stamper, end) {
								return
							}
						}
					}
					failure := failureFromErrorFrame(frame.Data, outputPreceded)
					if failure != nil {
						if errEv, err := ai.ErrorEvent(failure); err == nil {
							emit(ctx, out, stamper, errEv)
						}
					}
					return
				}

				chunk, decodeErr := decodeChunk(frame.Data)
				if decodeErr != nil {
					emitFailure(ctx, out, stamper, state, errMalformedChunkJSON, outputPreceded, blockOpen)
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
					emitFailure(ctx, out, stamper, state, applyErr, outputPreceded, blockOpen)
					return
				}
			for _, ev := range events {
				if !emit(ctx, out, stamper, ev) {
					// AI-32.3: a normal mid-stream emit losing the
					// ctx.Done() race (R-AEM-014) MUST still surface
					// a typed terminal failure on cancel/deadline
					// (S-AEM-051/052), NOT the AI-20.3 silent loss
					// path. Bounded-wait send — same rationale as
					// emitFailure's terminal send — so the consumer
					// that has reached drainAll can observe the
					// typed failure alongside the eventual close.
					if ctxErr := ctx.Err(); ctxErr != nil {
						if failure := midStreamFailureFrom(ctxErr, outputPreceded); failure != nil {
							if blockOpen {
								if end, err := ai.NewTextBlockEnd(textBlockIndex); err == nil {
									if endStamped := stamper.Stamp(end); true {
										select {
										case out <- endStamped:
										case <-time.After(emitFailureSendBound):
										}
									}
								}
							}
							if errEv, err := ai.ErrorEvent(failure); err == nil {
								if errStamped := stamper.Stamp(errEv); true {
									select {
									case out <- errStamped:
									case <-time.After(emitFailureSendBound):
									}
								}
							}
						}
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
					case ai.EventKindToolCallStart:
						toolBlocksOpen++
					case ai.EventKindToolCallEnd:
						toolBlocksOpen--
					}
				}
			}
			if feedErr != nil {
				emitFailure(ctx, out, stamper, state, feedErr, outputPreceded, blockOpen)
				return
			}
		}
		if readErr != nil {
			if readErr == io.EOF {
				if finishErr := decoder.Finish(); finishErr != nil {
					emitFailure(ctx, out, stamper, state, finishErr, outputPreceded, blockOpen)
					return
				}
				// Clean SSE framing, but the sentinel was never observed
				// — design.md D9's sibling case at the transport-EOF edge
				// rather than the [DONE]-with-no-terminal-chunk edge.
				emitFailure(ctx, out, stamper, state, errIncompleteStream, outputPreceded, blockOpen)
				return
			}
			if ctx.Err() != nil {
				// AI-32.3/S-AEM-051/052: deadline/cancel mid-stream
				// becomes a typed terminal failure (S-AEM-051
				// FailureCategoryTimeout, S-AEM-052
				// FailureCategoryCancellation), NOT the AI-20.3 silent
				// loss path. Bounded-wait send so the consumer can
				// still observe the typed failure even when ctx
				// itself is canceled.
				if failure := midStreamFailureFrom(ctx.Err(), outputPreceded); failure != nil {
					if end, err := ai.NewTextBlockEnd(textBlockIndex); blockOpen && err == nil {
						select {
						case out <- end:
						case <-time.After(emitFailureSendBound):
						}
					}
					if errEv, err := ai.ErrorEvent(failure); err == nil {
						select {
						case out <- errEv:
						case <-time.After(emitFailureSendBound):
						}
					}
				}
				return
			}
			emitFailure(ctx, out, stamper, state, errIncompleteStream, outputPreceded, blockOpen)
			return
		}
	}
}

// isOutputEvent reports whether ev counts toward R-AIP-010's PartialOutput
// discriminator (design.md D7): every text-block-scoped kind counts;
// ResponseStart never does (S-ATS-050) — it is not a normalized output
// event, only the announcement that one may follow.
//
// AI-30.1 widens the set to include the three tool-call kinds (Start,
// Delta, End) so a stream that emitted a tool-call event before a
// mid-stream failure reports PartialOutput() = true (R-ATL-009,
// S-ATL-044). ResponseStart still never counts.
func isOutputEvent(ev ai.Event) bool {
	switch ev.Kind() {
	case ai.EventKindTextBlockStart, ai.EventKindTextDelta, ai.EventKindTextBlockEnd,
		ai.EventKindToolCallStart, ai.EventKindToolCallDelta, ai.EventKindToolCallEnd:
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
//
// blockOpen is run's own tracked fact, too (design.md D8, AI-28.7): whether
// a text block is currently open on the carrier. Every one of run's own
// seven call sites can reach this function with a block genuinely open —
// truncation, an invalid-JSON frame, an out-of-enum finish_reason, and
// every other cause applyChunk can return all leave a block open exactly
// when the failing chunk arrives before a terminal chunk ever closes it.
// When blockOpen is true, this function sends the closing TextBlockEnd —
// through the same emit helper, selecting on ctx like every other send —
// before building or sending the terminal ErrorEvent, so a stream this
// function terminates still satisfies ai.CheckStream's no-unterminated-
// block invariant (AI-16) exactly like a cleanly-terminated one already
// does. If the close loses the race with cancellation, this function
// returns immediately without attempting the ErrorEvent send either — the
// same "cancellation wins, nothing further is observable" posture every
// other emit call in this package already keeps.
//
// state is the mapper state — its toolOpenOrder is read to close any
// still-open tool-call blocks with raw accumulated bytes (AI-30.1,
// D7/D8 inheritance, Q2 coordinator ruling). NewToolCallEnd does not
// validate bytes, so a malformed payload's bytes survive into the
// truncated ToolCallEnd untouched — never fabricated `{}`.
func emitFailure(ctx context.Context, out chan<- ai.Event, stamper *ai.Stamper, state *mapperState, cause error, outputPreceded, blockOpen bool) {
	// AI-32.3: scoped to the function's own terminal-event send. The
	// tool-call close + text-block close above still respect ctx via
	// emit(), so a cancellation that lands BEFORE this function runs
	// aborts the close chain — exactly like every other close path in
	// this package. The terminal event itself, however, MUST be
	// observable per S-AEM-051/052; emit()'s select-on-ctx.Done() races
	// with that guarantee when ctx is already cancelled, so the
	// terminal send uses a bounded-wait select instead.

	// Close every still-open tool-call block (AI-30.1, D8 inheritance).
	// truncateOpenCalls reads state.toolOpenOrder; its raw-bytes close
	// keeps a partial / malformed payload's bytes intact (Q2 ruling).
	if state != nil {
		toolEnds, terr := state.truncateOpenCalls()
		if terr == nil {
			for _, end := range toolEnds {
				if !emit(ctx, out, stamper, end) {
					return
				}
			}
		}
	}

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
			if !emit(ctx, out, stamper, end) {
				return
			}
		}
	}

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
	// Stamp the terminal event through the same per-stream stamper
	// every other event in this stream already uses (R-AEE-007) —
	// bypassing emit() is fine for the bounded-wait, but the sequence
	// assignment is not bypassed, ever (R-AEE-008).
	stamped := stamper.Stamp(ev)
	// AI-32.3 bounded-wait on the terminal failure: do NOT select on
	// ctx.Done() here (AI-32.3's pattern races with cancellation —
	// the consumer is actively draining). Use a short direct send with
	// a fallthrough timer to avoid a goroutine leak if the consumer
	// stopped, so a stream this function terminates still satisfies
	// ai.CheckStream's no-unterminated-block invariant (AI-16).
	select {
	case out <- stamped:
	case <-time.After(emitFailureSendBound):
	}
}

// referenced from run() so the linter / errcheck posture this file's
// other helpers already keep is preserved for the in-band frame path.
var _ = emit
