// AI-28.1.2 — the mapper state machine: response identity (absorbed from
// slice 1's own producerState), text-block minting (R-ATS-008), and the
// finish-reason gate that feeds the sentinel-deferred completion (R-ATS-010,
// design.md D4/D5/D9).
//
// # Block boundaries are minted here, not read from the wire (C3, verify-report W4)
//
// C3 (negative, load-bearing): the Chat Completions stream schema defines
// no block, content-part, start, or stop framing events of any kind — text
// arrives solely as flat delta.content string fragments (C7), with
// boundaries signaled only implicitly (role on the first delta, an empty
// delta with a non-null finish_reason on the last chunk). Every
// TextBlockStart/TextBlockEnd this file mints below (textBlockIndex,
// applyChunk) is therefore this adapter's OWN boundary, never a
// vendor-reported one — R-ATS-008's own contract, restated here at its
// minting source rather than only at bridge_test.go's own citations
// (which prove the wire carries no block bytes to RENDER, not why this
// file mints them on decode).
//
// # One state machine, not two (a slice-1→slice-2 structural consequence)
//
// Slice 1 split frame-to-event mapping across two things: stream.go's own
// producerState (identity, finish-reason capture) and stream.go's own
// applyFrame. Design's own "Requirement homes" table assigns R-ATS-007…010
// to "chunk.go + stream_state.go", not stream.go, so this slice centralizes
// every frame-to-event decision — including R-ATS-004's identity
// establishment, which slice 1 owned — into this one mapperState. stream.go
// now only recognises the terminal sentinel and drives the read loop; it
// decodes nothing and maps nothing itself.
//
// # Completion stays deferred to the sentinel (D4, D9, unchanged from slice 1)
//
// mapperState never constructs an ai.Completion. It accumulates the finish
// reason and the block's open/closed state across chunks; buildCompletion
// is called only once, at the terminal sentinel (stream.go's run), because
// C4's usage chunk — arriving after choice 0's terminal chunk — is what a
// complete completion record eventually needs (a later slice's own
// R-ATS-015/016; this slice always reports absent usage).
//
// # The terminal→sentinel window (R-ATS-008, C-1 corrective; AI-28.5 rows 1/2)
//
// Once a chunk sets terminalSeen, applyChunk closes an open block on that
// same chunk. A LATER chunk in that window is either delta-less or the
// usage chunk (an empty choices array, C4) — both absorbed as zero events
// by construction: hasChoice is false for the usage chunk, and a
// delta-less choice item contributes nothing to either the finish-reason
// gate or the content check — or it is one of AI-28.5's own two violations
// this same window can carry: a choice-0 content STRING (R-ATS-020 row 1,
// "delta after close", S-ATS-074) or a second non-null finish_reason
// (row 2, "duplicate close", S-ATS-075). Both are checked explicitly, once
// terminalSeen is already true from a PRIOR chunk — never on the chunk
// that sets terminalSeen for the first time.
//
// # AI-30 tool-call accumulation (R-ATL-001…006)
//
// mapperState grows a per-call map keyed by wire index, plus a shared
// nextBlockIndex allocator starting at 1 (D1). Each element of
// choice 0's tool_calls array is processed in array order: missing
// index → errToolElementMissingIndex (S-ATL-011); identity merge (unset→set,
// identical-repeat no-op S-ATL-015, differing non-empty → errToolCallIdentityMismatch
// S-ATL-016); once id+name are set, ToolCallStart is emitted and any
// pending fragments flush as one ToolCallDelta each; non-empty fragments
// append to per-call buffers bounded by toolCallAccumulationCap.
//
// Assembly happens exactly once at close, through ai.NewToolCall
// (R-ATC-007), on the terminal chunk — never per-fragment, never
// per-chunk. The ordinal is the position in toolOpenOrder (R-ATL-005,
// R-ATC-012: this package ships no accumulator).

package openaicompat

import (
	"errors"
	"fmt"

	"github.com/cachicamas/backend/agent/src/ai"
)

// textBlockIndex is R-ATS-008 rule 1's constant: at most one text block is
// minted per stream, always at this index — an adapter-minted boundary
// (C3, this file's own header comment), never one the wire itself carries.
const textBlockIndex ai.BlockIndex = 1

// toolCallAccumulationCap bounds the per-call argument-byte buffer
// (R-ATL-006, D5): 4× DefaultMaxFrameBytes (8 MiB × 4 = 32 MiB) so any
// single frame this adapter can decode (≤ 8 MiB) cannot on its own trip
// the per-call cap, but a runaway accumulation across many frames does.
// Recorded as an unexported package constant in captureLimit's mold.
// The relation is strict (S-ATL-029): accumulation reaching exactly the
// cap still completes; exceeding it fails.
//
// The tenth failure category's second consumer (D5/S-ATL-032):
// errors.go records that AI-30.1 item 4 now exists; the per-call cap is
// the second consumer that closes the loop on the resource-exhaustion
// concept errors.go named but did not absorb. Widening ai.FailureCategory
// is a separate change.
const toolCallAccumulationCap = 4 * DefaultMaxFrameBytes

// errMalformedIdentity is this package's own unexported cause for a first
// chunk whose id or model surfaced, through ai.NewResponseStart's own
// constructor error, as present but empty (design.md D7, N-7b): both
// fields are required non-empty by that constructor. Wraps
// ai.ErrMalformedResponse with %w so errors.Is(err, ai.ErrMalformedResponse)
// holds; unexported so S-ART-054's allowlist stays untouched — no new
// exported sentinel (R-ATS-025).
var errMalformedIdentity = fmt.Errorf("openaicompat: first chunk's response identity was malformed: %w", ai.ErrMalformedResponse)

// errUnrecognizedFinishReason is this package's own unexported cause for a
// terminal chunk whose finish_reason is not byte-exactly one of C2's five
// enum members (design.md D5/N-6, R-ATS-010, S-ATS-039). Wraps
// ai.ErrMalformedResponse with %w for the same reason errMalformedIdentity
// does; unexported for the same reason (R-ATS-025).
var errUnrecognizedFinishReason = fmt.Errorf("openaicompat: terminal chunk's finish_reason was outside the recognised enum: %w", ai.ErrMalformedResponse)

// errMissingRequiredField is this package's own unexported cause for a
// chunk recognized by isChunk() (object present and correct) that is
// nonetheless missing a C1 required top-level field this milestone
// re-validates on every chunk, not merely the first (R-ATS-021, S-ATS-081;
// chunk.go's wireChunk.hasRequiredFields). Wraps ai.ErrMalformedResponse
// with %w for the same reason errMalformedIdentity does; unexported for the
// same reason (R-ATS-025).
var errMissingRequiredField = fmt.Errorf("openaicompat: recognized chunk was missing a C1 required top-level field: %w", ai.ErrMalformedResponse)

// errToolElementMissingIndex is this package's own unexported cause for a
// tool_calls element with no `index` key (R-ATL-002, S-ATL-011, C9.1's
// required list). Wraps ai.ErrMalformedResponse for the same reason
// errMalformedIdentity does; unexported for the same reason (R-ATS-025).
var errToolElementMissingIndex = fmt.Errorf("openaicompat: a tool_calls element arrived without an index: %w", ai.ErrMalformedResponse)

// errToolCallIdentityMismatch is this package's own unexported cause for a
// tool call whose later element carries a DIFFERENT non-empty id or name
// from the one already established (R-ATL-003, S-ATL-016). Wraps
// ai.ErrMalformedResponse for the same reason; unexported for the same
// reason (R-ATS-025). Identical repeats (S-ATL-015) are tolerated, not
// failed.
var errToolCallIdentityMismatch = fmt.Errorf("openaicompat: a tool call's id or name differed from the value already established: %w", ai.ErrMalformedResponse)

// errToolCallMissingIdentity is this package's own unexported cause for a
// tool call that closed without id or name ever supplied (R-ATL-003,
// S-ATL-017/053). Wraps ai.ErrMalformedResponse for the same reason;
// unexported for the same reason (R-ATS-025).
var errToolCallMissingIdentity = fmt.Errorf("openaicompat: a tool call closed without id or name ever supplied: %w", ai.ErrMalformedResponse)

// errToolCallOverCap is this package's own unexported cause for an
// accumulation whose per-call byte total exceeded toolCallAccumulationCap
// (R-ATL-006, D5, S-ATL-028). Wraps ai.ErrMalformedResponse for the same
// reason errMalformedIdentity does — a payload that exceeds the cap may
// be perfectly well-formed JSON, but is reported under the malformed
// category (ErrFrameTooLarge's existing compromise form, restated in this
// comment so the cause and its rationale travel together).
var errToolCallOverCap = fmt.Errorf("openaicompat: a tool call's argument accumulation exceeded the documented per-call cap: %w", ai.ErrMalformedResponse)

// errToolDeltaAfterCloseToolCalls is this package's own unexported cause
// for a tool_calls element arriving after choice 0's terminal chunk
// (D4, R-ATS-020 row 1's tool-call analog, S-ATL-074's mirror).
// Wraps ai.ErrMalformedResponse for the same reason; unexported for the
// same reason (R-ATS-025).
var errToolDeltaAfterCloseToolCalls = fmt.Errorf("openaicompat: a tool_calls element arrived after choice 0's terminal chunk: %w", ai.ErrMalformedResponse)

// errTextAfterToolBlock is this package's own unexported cause for an
// unrepresentable shape: content text arriving AFTER a tool block has
// already consumed block index 1 (D1's edge case, R-ATL-010 posture).
// Wraps ai.ErrMalformedResponse for the same reason; unexported for the
// same reason (R-ATS-025).
var errTextAfterToolBlock = fmt.Errorf("openaicompat: content text arrived after a tool block consumed block index 1: %w", ai.ErrMalformedResponse)

// R-ATS-020's five-row structural violation table (AI-28.5, doc 0002's own
// row names) each get their own unexported cause, all wrapping
// ai.ErrMalformedResponse for the same reason errMalformedIdentity does,
// all unexported for the same reason (R-ATS-025, no new sentinel). Row 5
// ("close without open") needs none of its own: a terminal chunk with no
// usable response identity is the same shape errMalformedIdentity already
// covers (S-ATS-078).
var (
	// errDeltaAfterClose is row 1 (S-ATS-074): a choice-0 content string
	// arriving after that choice's terminal chunk and before the terminal
	// sentinel — R-ATS-008's own "what may legally follow the terminal
	// chunk" paragraph names this the violation reading of that window, in
	// contrast to a delta-less or usage chunk there (S-ATS-032).
	errDeltaAfterClose = fmt.Errorf("openaicompat: a content delta arrived after choice 0's terminal chunk: %w", ai.ErrMalformedResponse)

	// errDuplicateClose is row 2 (S-ATS-075): a second chunk carrying a
	// non-null finish_reason for choice 0.
	errDuplicateClose = fmt.Errorf("openaicompat: a second chunk carried a non-null finish_reason for choice 0: %w", ai.ErrMalformedResponse)

	// errSecondResponseStart is row 3 (S-ATS-076): a chunk whose top-level
	// id differs from the response identity already established.
	errSecondResponseStart = fmt.Errorf("openaicompat: a later chunk's id differed from the established response identity: %w", ai.ErrMalformedResponse)

	// errDeltaWithNoOpenBlock is row 4 (S-ATS-077), doc 0002's own
	// structural name: a choice item whose index is negative or absent
	// while carrying a content string.
	errDeltaWithNoOpenBlock = fmt.Errorf("openaicompat: a content delta's choice item carried a negative or absent index: %w", ai.ErrMalformedResponse)
)

// toolCallState is the per-call accumulation record (R-ATL-002/003/004,
// D3). One entry per call, keyed by wire index. id/name are the
// "currently established" identity values ("" = not yet supplied; S-ATL-014).
// args holds the accumulated fragment bytes byte-exact (S-ATL-022).
// pending holds fragments received before identity completed — flushed
// as one ToolCallDelta each at start (R-ATL-003, R-ATL-004).
type toolCallState struct {
	block   ai.BlockIndex
	id      string
	name    string
	args    []byte
	pending [][]byte
	started bool
}

// mapperState is AI-28.1.2's frame-to-events mapper. Its zero value is
// ready to use: no chunk has been read yet, no block minted, no terminal
// seen — the same "ready without a constructor" posture ai.Stamper's own
// zero value carries.
type mapperState struct {
	// identityEstablished is true once the first chunk's id/model produced
	// a ResponseStart (R-ATS-004).
	identityEstablished bool

	// responseID is the id byte-exactly established by the first chunk
	// (R-ATS-004), remembered so a LATER chunk carrying a different id can
	// be recognized as R-ATS-020 row 3's own violation (S-ATS-076,
	// "second response start") rather than silently ignored.
	responseID string

	// blockOpen is true between a minted TextBlockStart and its matching
	// TextBlockEnd (R-ATS-008 rules 2/3).
	blockOpen bool

	// blockClosed is true once the one block this stream may ever mint has
	// closed — distinct from blockOpen==false before any block ever opened
	// (rule 4: a content-free stream mints no block, open or closed).
	blockClosed bool

	// finishReason is choice 0's captured, gate-accepted finish reason
	// (R-ATS-010), read only by buildCompletion at the sentinel.
	finishReason ai.FinishReason

	// terminalSeen is true once choice 0's terminal chunk (a non-null,
	// gate-accepted finish_reason) has been observed.
	terminalSeen bool

	// usage is the last populated usage chunk's mapping (AI-28.3, D10): the
	// zero ai.Usage{} — every field absent — until a chunk carrying a
	// non-null usage object arrives, at which point this is OVERWRITTEN
	// wholesale (never merged) with usageFromWire's result. A later
	// usage:null chunk (C4) never touches this field.
	usage ai.Usage

	// toolCalls is the per-call accumulation map keyed by wire index
	// (R-ATL-002, D3). Each entry is the call's own buffered state —
	// per-call buffers, never shared (R-ATL-004, S-ATL-019/020).
	toolCalls map[int]*toolCallState

	// toolOpenOrder records the wire indexes in first-seen order — the
	// call's ordinal is its 1-based position in this slice (R-ATL-005,
	// S-ATL-024, R-ATC-012). This package ships no ordinal field on the
	// call state (S-ATL-058): the slice IS the derivation.
	toolOpenOrder []int

	// nextBlockIndex is the shared next-free block allocator starting at 1
	// (R-ATL-002, D1). textBlockIndex (1) is consumed by the first text
	// block; every new tool call takes the counter value and increments.
	// Never equal to the wire index (R-ATL-002). ≥ 1, stream-wide unique,
	// stable per call.
	nextBlockIndex ai.BlockIndex

	// textBlockConsumed is true once the text block has been opened —
	// D1's edge case detector: content text arriving AFTER a tool block
	// has already consumed index 1 yields errTextAfterToolBlock rather
	// than a silent re-index (R-ATL-010 posture).
	textBlockConsumed bool
}

// applyChunk maps one decoded, non-sentinel chunk to zero or more
// normalized events, in emission order (R-ATS-004, R-ATS-007, R-ATS-008,
// R-ATS-010):
//
//  1. at most one ResponseStart, only on the first chunk ever applied;
//  2. at most one TextBlockStart, immediately before this chunk's own
//     delta, only when this is the content stream's first chunk;
//  3. at most one TextDelta, when this chunk's choice-0 content is a JSON
//     string (R-ATS-007's trichotomy, chunk.go's contentText);
//  4. at most one TextBlockEnd, when this chunk carries choice 0's
//     terminal finish_reason and a block was open.
//
// Usage mapping (R-ATS-015/016, AI-28.3) is not part of the numbered
// sequence above: it never contributes an event of its own — usage
// surfaces only inside the eventual Completion, at the sentinel
// (buildCompletion) — so it is applied unconditionally, ahead of the
// choice-0 checks below, since the dedicated usage chunk (C4) carries an
// EMPTY choices array and would otherwise never reach any per-choice logic
// at all.
//
// A malformed identity or an out-of-enum finish_reason returns a nil event
// slice and this package's own unexported cause — the caller (stream.go's
// run) never inspects a partial events slice on error.
func (s *mapperState) applyChunk(chunk wireChunk) ([]ai.Event, error) {
	var events []ai.Event

	// R-ATS-021/S-ATS-081: a recognized chunk (the caller already confirmed
	// isChunk()) missing a C1 required field is a broken KNOWN shape, never
	// a skip — checked first, and uniformly on every chunk, not merely the
	// first (chunk.go's wireChunk.hasRequiredFields doc comment explains the
	// field scope).
	if !chunk.hasRequiredFields() {
		return nil, errMissingRequiredField
	}

	// R-ATS-020 row 3 (S-ATS-076): a chunk whose id differs from the
	// already-established response identity is a second response start,
	// checked before ever touching identity establishment itself — which
	// only ever runs once, on the chunk that has not yet established it.
	if s.identityEstablished && chunk.ID != s.responseID {
		return nil, errSecondResponseStart
	}

	if !s.identityEstablished {
		started, err := ai.NewResponseStart(chunk.ID, chunk.Model)
		if err != nil {
			return nil, errMalformedIdentity
		}
		s.identityEstablished = true
		s.responseID = chunk.ID
		events = append(events, started)
	}

	if chunk.Usage != nil {
		// C4: last populated usage chunk wins, no cumulative merge (D10) —
		// this OVERWRITES s.usage wholesale rather than folding fields in.
		s.usage = usageFromWire(*chunk.Usage)
	}

	choice, hasChoice := chunk.choice0()
	if !hasChoice {
		// The usage chunk (C4) and any other choice-less frame contribute
		// nothing beyond identity and usage, both already handled above.
		return events, nil
	}

	// R-ATS-020 rows 1/2 (S-ATS-074/075): once choice 0's terminal chunk
	// has already closed things, on a PRIOR chunk, the only shapes the wire
	// may legally carry in that window are delta-less chunks and the usage
	// chunk (R-ATS-008's own "what may legally follow the terminal chunk"
	// paragraph) — both already absorbed as zero events below, since this
	// chunk's own choice item carries neither a content string nor a
	// non-null finish_reason. A content string here is row 1 (delta after
	// close); a second non-null finish_reason here is row 2 (duplicate
	// close). Checked before this chunk's own finish_reason/content are
	// otherwise processed, so the terminal chunk itself — which SETS
	// terminalSeen for the first time, on this same call — never trips
	// either row.
	//
	// Row 2 carries one documented widening beyond the strict OpenAI
	// streaming spec: OpenRouter (and the Google / Anthropic-with-usage
	// providers it fronts) emits a usage-enrichment chunk AFTER the
	// terminal chunk on several routes (notably google/gemini-3.7-flash
	// and anthropic/claude-* with usage blocks). That enrichment chunk
	// re-asserts the same finish_reason ("stop") alongside its usage
	// payload. The strict-spec reading of R-ATS-020 row 2 is to fail the
	// stream on this duplicate, which here would mean firing
	// errDuplicateClose, aborting the stream, and discarding the
	// already-streamed text — the user-visible symptom is a frontend
	// empty assistant bubble with a trailing `event: error` frame.
	//
	// The safe disposition is to accept the duplicate AS A NO-OP when it
	// carries no NEW information beyond the duplicate finish_reason:
	// empty (or absent / null) delta.content AND no delta.tool_calls.
	// The carrier has already seen terminalSeen=true and the completion
	// event has either been emitted on the terminal chunk itself or is
	// about to be minted by buildCompletion at the sentinel; passing the
	// enrichment chunk through silently lets usage arrive via the same
	// chunk (usage is mapped earlier in this method, ahead of the
	// terminal-window check) without losing the text already streamed.
	// Repeated delta.role on the duplicate chunk is benign — wireDelta
	// has no Role field, so encoding/json drops it — and is also
	// implicitly accepted by this widening.
	//
	// Chunks that duplicate finish_reason WHILE carrying new content (a
	// non-empty content string, or any tool_calls element) still fail
	// with errDuplicateClose — only the "duplicate with no new info"
	// shape is widened. errDeltaAfterClose below is likewise UNCHANGED:
	// a non-null content key (even an empty "") AFTER terminal still
	// fires row 1; only the duplicate-finish_reason check is widened.
	if s.terminalSeen {
		if choice.FinishReason != nil {
			// Row 2, narrowed: only the "duplicate with new info" shape
			// fails. A no-new-info duplicate (OpenRouter usage enrichment)
			// is absorbed — see comment block above.
			if len(choice.Delta.ToolCalls) > 0 {
				return nil, errDuplicateClose
			}
			if text, present := contentText(choice.Delta.Content); present && text != "" {
				return nil, errDuplicateClose
			}
			return events, nil
		}
		if _, present := contentText(choice.Delta.Content); present {
			return nil, errDeltaAfterClose
		}
		// Tool-call elements in the terminal window are an analogous
		// violation (D4). The terminal chunk itself may legally carry
		// tool_calls (its own applyChunk processes them before close) —
		// this check fires only on a LATER chunk after terminalSeen.
		if len(choice.Delta.ToolCalls) > 0 {
			return nil, errToolDeltaAfterCloseToolCalls
		}
		return events, nil
	}

	if choice.FinishReason != nil {
		reason, ok := rawStrictFinishReason(*choice.FinishReason)
		if !ok {
			return nil, errUnrecognizedFinishReason
		}
		s.finishReason = reason
		s.terminalSeen = true
	}

	// Process this chunk's tool_calls elements. Before processing the first
	// tool element, if the text block is open and the chunk carries no
	// content text of its own, close the text block first — the conformance
	// case mixedTextAndToolCallCase requires TextBlockEnd to come BEFORE
	// any tool-call event (text precedes tool semantically). D4's terminal
	// close path remains: terminalSeen-set chunks still close text blocks
	// before emitting tool-block ends (S-ATL-024).
	if len(choice.Delta.ToolCalls) > 0 && s.blockOpen {
		_, textPresent := contentText(choice.Delta.Content)
		if !textPresent {
			end, err := ai.NewTextBlockEnd(textBlockIndex)
			if err != nil {
				return nil, err
			}
			events = append(events, end)
			s.blockOpen = false
			s.blockClosed = true
		}
	}
	for _, elem := range choice.Delta.ToolCalls {
		more, err := s.applyToolElement(elem)
		if err != nil {
			return nil, err
		}
		events = append(events, more...)
	}

	if text, present := contentText(choice.Delta.Content); present {
		// R-ATS-020 row 4 (S-ATS-077): a choice item carrying content with
		// a negative or absent index is a violation, checked before this
		// content contributes any block-minting or delta event.
		if !choice.hasValidIndex() {
			return nil, errDeltaWithNoOpenBlock
		}
		// D1 edge case: content text arriving AFTER a tool block has
		// already consumed block index 1 is unrepresentable — typed
		// failure, never a silent re-index (R-ATL-010 posture).
		if s.toolCalls != nil && !s.textBlockConsumed {
			// Has any tool call been opened? toolCalls being non-nil
			// with at least one entry means yes.
			sawTool := false
			for _, st := range s.toolCalls {
				if st.started {
					sawTool = true
					break
				}
			}
			if sawTool {
				return nil, errTextAfterToolBlock
			}
		}
		if !s.blockOpen && !s.blockClosed {
			start, err := ai.NewTextBlockStart(textBlockIndex)
			if err != nil {
				return nil, err
			}
			events = append(events, start)
			s.blockOpen = true
			s.textBlockConsumed = true
			// Mark nextBlockIndex so the first tool call skips 1.
			if s.nextBlockIndex < 2 {
				s.nextBlockIndex = 2
			}
		}
		delta, err := ai.NewTextDelta(textBlockIndex, text)
		if err != nil {
			return nil, err
		}
		events = append(events, delta)
	}

	if s.terminalSeen {
		// Close text block first (if open).
		if s.blockOpen {
			end, err := ai.NewTextBlockEnd(textBlockIndex)
			if err != nil {
				return nil, err
			}
			events = append(events, end)
			s.blockOpen = false
			s.blockClosed = true
		}
		// Then assemble and close every open tool call, in toolOpenOrder
		// (R-ATL-005 / S-ATL-024, ascending ordinal).
		for _, wireIdx := range s.toolOpenOrder {
			st := s.toolCalls[wireIdx]
			if st == nil {
				continue
			}
			if st.id == "" || st.name == "" {
				return nil, errToolCallMissingIdentity
			}
			part, err := ai.NewToolCall(st.id, st.name, st.args)
			if err != nil {
				// R-ATL-009: malformed assembly carries the raw partial
				// fragment bytes through a typed cause. The bytes reach
				// stream.go's failure path via this return; the
				// cause-typed return (this is error, not an event) is what
				// carries them.
				return nil, &malformedToolCallAssembly{data: st.args, wireIdx: wireIdx}
			}
			tc, _ := part.ToolCall()
			ended, err := ai.NewToolCallEnd(st.block, tc.Arguments())
			if err != nil {
				return nil, err
			}
			events = append(events, ended)
		}
	}

	return events, nil
}

// applyToolElement processes one wireToolCallElement per D3's sequence.
// On error returns nil and the package's unexported cause; on success
// returns the additional events (ToolCallStart and any ToolCallDelta)
// this element contributed.
func (s *mapperState) applyToolElement(elem wireToolCallElement) ([]ai.Event, error) {
	if elem.Index == nil {
		return nil, errToolElementMissingIndex
	}
	wireIdx := *elem.Index

	if s.toolCalls == nil {
		s.toolCalls = make(map[int]*toolCallState)
	}
	st, exists := s.toolCalls[wireIdx]
	if !exists {
		// Mint a new block from the shared allocator (D1):
		// - if no text block has been minted yet, the first tool call
		//   takes block 1 (textBlockIndex). conformance case 1
		//   (toolCallInterleavedCase) uses block 1 for the first tool.
		// - if a text block has already been minted (textBlockIndex = 1),
		//   the first tool call takes block 2.
		if s.nextBlockIndex < 1 {
			if s.textBlockConsumed {
				s.nextBlockIndex = 2
			} else {
				s.nextBlockIndex = 1
			}
		}
		st = &toolCallState{block: s.nextBlockIndex}
		s.nextBlockIndex++
		s.toolCalls[wireIdx] = st
		s.toolOpenOrder = append(s.toolOpenOrder, wireIdx)
	}

	// Identity merge (S-ATL-013/014/015/016/017).
	if elem.Function != nil {
		if len(elem.Function.Name) > 0 {
			nameStr := string(unquoteJSONString(elem.Function.Name))
			if st.name == "" {
				st.name = nameStr
			} else if nameStr != st.name {
				return nil, errToolCallIdentityMismatch
			}
		}
	}
	if len(elem.ID) > 0 {
		idStr := string(unquoteJSONString(elem.ID))
		if st.id == "" {
			st.id = idStr
		} else if idStr != st.id {
			return nil, errToolCallIdentityMismatch
		}
	}

	// Fragment handling.
	var fragment []byte
	if elem.Function != nil && len(elem.Function.Arguments) > 0 {
		fragment = unquoteJSONString(elem.Function.Arguments)
	}

	// If identity not yet complete, queue the fragment (R-ATL-003:
	// start precedes any delta). Otherwise emit start + flush + delta.
	var events []ai.Event
	if !st.started {
		// Need both id and name before we can emit start.
		if st.id != "" && st.name != "" {
			start, err := ai.NewToolCallStart(st.block, st.id, st.name)
			if err != nil {
				return nil, err
			}
			events = append(events, start)
			st.started = true
			// Flush any queued fragments as one ToolCallDelta each,
			// boundaries preserved (S-ATL-036's count behavior).
			for _, pending := range st.pending {
				if len(pending) == 0 {
					continue
				}
				d, err := ai.NewToolCallDelta(st.block, pending)
				if err != nil {
					return nil, err
				}
				events = append(events, d)
				if err := s.appendArgs(st, pending); err != nil {
					return nil, err
				}
			}
			st.pending = nil
		} else {
			// Queue until identity completes; if the chunk had a fragment,
			// queue it too. Empty fragments are no-ops (S-ATL-033).
			if len(fragment) > 0 {
				st.pending = append(st.pending, fragment)
			}
			return events, nil
		}
	}

	// Identity complete (started). Apply this fragment if non-empty.
	if len(fragment) == 0 {
		// S-ATL-033/035: empty or absent argument is a no-op.
		return events, nil
	}

	// Cap check BEFORE appending (D5: "len(args)+len(frag) > cap fails").
	if len(st.args)+len(fragment) > int(toolCallAccumulationCap) {
		return nil, errToolCallOverCap
	}
	if err := s.appendArgs(st, fragment); err != nil {
		return nil, err
	}
	d, err := ai.NewToolCallDelta(st.block, fragment)
	if err != nil {
		return nil, err
	}
	events = append(events, d)
	return events, nil
}

// appendArgs appends fragment to st.args while enforcing the per-call cap
// (D5). The cap check lives here rather than at the call site so every
// append path (initial flush of pending, post-start deltas, and any
// future re-entry) goes through one bounded function. Returns
// errToolCallOverCap on overflow.
func (s *mapperState) appendArgs(st *toolCallState, fragment []byte) error {
	if len(st.args)+len(fragment) > int(toolCallAccumulationCap) {
		return errToolCallOverCap
	}
	st.args = append(st.args, fragment...)
	return nil
}

// malformedToolCallAssembly is R-ATL-009's typed cause for a call whose
// accumulated bytes are not well-formed JSON at close. The bytes are
// reachable only through the unexported bytes() accessor — Error() is a
// fixed string built from no captured byte (R-AEM-016 / capturedBody's
// posture). Unwrap chains to ai.ErrMalformedResponse so errors.Is holds.
//
// The bytes are bounded by captureLimit (8 KiB, capture.go's own) —
// reused, never a second, independent bound (R-AEM-015, design D6).
type malformedToolCallAssembly struct {
	data    []byte
	wireIdx int
}

// Error returns a fixed diagnostic string (R-AEM-016): never built from
// the captured bytes, so no captured content can reach any rendered
// failure text.
func (m *malformedToolCallAssembly) Error() string {
	return "openaicompat: a tool call's assembled arguments were not well-formed JSON"
}

// Unwrap returns the wrapped ai.ErrMalformedResponse so errors.Is(err,
// ai.ErrMalformedResponse) holds.
func (m *malformedToolCallAssembly) Unwrap() error {
	return ai.ErrMalformedResponse
}

// bytes returns the raw accumulated bytes, bounded by captureLimit on
// read. Unexported and never rendered by any Error()/%v/%+v path.
func (m *malformedToolCallAssembly) bytes() []byte {
	if len(m.data) > captureLimit {
		out := make([]byte, captureLimit)
		copy(out, m.data[:captureLimit])
		return out
	}
	return m.data
}

// malformedAssemblyBytes is the exported-side helper (R-ATL-009,
// S-ATL-045/047): errors.As the typed cause and reads its bytes. This is
// the test-side path the conformance uses; it cannot be invoked by
// shipped production code because bytes() is unexported and the type is
// likewise unexported. Test files in this package see it directly.
func malformedAssemblyBytes(err error) ([]byte, bool) {
	var m *malformedToolCallAssembly
	if errors.As(err, &m) {
		return m.bytes(), true
	}
	return nil, false
}

// buildCompletion constructs the sentinel-triggered completion from
// whatever finish reason this mapper's chunks captured, and s.usage —
// wholly absent (ai.Usage{}'s zero value) unless a usage chunk populated
// it along the way (R-ATS-015…016, AI-28.3). An error here means no
// terminal chunk was ever observed before the sentinel arrived (design.md
// D9, spec-silent) — the caller (stream.go's run) folds it into
// errIncompleteStream, unchanged from slice 1's own handling.
//
// # AI-31.1 — Matched stop-sequence recorded as deliberate absence (R-ACP-004, S-ACP-009)
//
// The completion carries no field naming which stop sequence fired,
// because the wire carries no such field anywhere — U4 NEGATIVE:
// stop_sequence → 0 hits, stop_reason → 0 hits; the chunk, choice and
// delta schemas declare no stop-related field beyond finish_reason
// itself; StopConfiguration states only that the returned text will
// not contain the stop sequence. This adapter therefore CANNOT derive,
// reconstruct or infer a matched-sequence value, and S-ACP-010 pins the
// behaviour by proving that an extra unknown stop_sequence key on the
// terminal chunk drains to the same completion (nothing is read, not
// nothing was offered). The matched-sequence disposition is recorded
// here as a deliberate and cited absence, never as a silent omission —
// see spec R-ACP-004 in
// openspec/changes/cachicamas-ai-provider-completion/specs/ai-provider-completion/spec.md.
func (s *mapperState) buildCompletion() (ai.Event, error) {
	return ai.NewCompletion(s.finishReason, s.usage)
}

// truncateOpenCalls closes every still-open tool call at stream end with
// its accumulated bytes — raw, never canonicalized (R-ATL-009, D7, Q2
// coordinator ruling). Called only from stream.go's failure paths after
// D8's text-block close, never from the clean close path (where
// applyChunk's terminal-chunk branch already handled them).
func (s *mapperState) truncateOpenCalls() ([]ai.Event, error) {
	var events []ai.Event
	for _, wireIdx := range s.toolOpenOrder {
		st := s.toolCalls[wireIdx]
		if st == nil {
			continue
		}
		if st.id == "" || st.name == "" {
			return nil, errToolCallMissingIdentity
		}
		// Carry the raw bytes; NewToolCallEnd does not validate them.
		ended, err := ai.NewToolCallEnd(st.block, st.args)
		if err != nil {
			return nil, err
		}
		events = append(events, ended)
	}
	return events, nil
}
