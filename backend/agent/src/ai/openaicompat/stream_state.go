// AI-28.1.2 — the mapper state machine: response identity (absorbed from
// slice 1's own producerState), text-block minting (R-ATS-008), and the
// finish-reason gate that feeds the sentinel-deferred completion (R-ATS-010,
// design.md D4/D5/D9).
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

package openaicompat

import (
	"fmt"

	"github.com/cachicamas/backend/agent/src/ai"
)

// textBlockIndex is R-ATS-008 rule 1's constant: at most one text block is
// minted per stream, always at this index.
const textBlockIndex ai.BlockIndex = 1

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
// with %w for the same reason errMalformedIdentity does; unexported for
// the same reason (R-ATS-025).
var errMissingRequiredField = fmt.Errorf("openaicompat: recognized chunk was missing a C1 required top-level field: %w", ai.ErrMalformedResponse)

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
	if s.terminalSeen {
		if _, present := contentText(choice.Delta.Content); present {
			return nil, errDeltaAfterClose
		}
		if choice.FinishReason != nil {
			return nil, errDuplicateClose
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

	if text, present := contentText(choice.Delta.Content); present {
		// R-ATS-020 row 4 (S-ATS-077): a choice item carrying content with
		// a negative or absent index is a violation, checked before this
		// content contributes any block-minting or delta event.
		if !choice.hasValidIndex() {
			return nil, errDeltaWithNoOpenBlock
		}
		if !s.blockOpen && !s.blockClosed {
			start, err := ai.NewTextBlockStart(textBlockIndex)
			if err != nil {
				return nil, err
			}
			events = append(events, start)
			s.blockOpen = true
		}
		delta, err := ai.NewTextDelta(textBlockIndex, text)
		if err != nil {
			return nil, err
		}
		events = append(events, delta)
	}

	if s.terminalSeen && s.blockOpen {
		end, err := ai.NewTextBlockEnd(textBlockIndex)
		if err != nil {
			return nil, err
		}
		events = append(events, end)
		s.blockOpen = false
		s.blockClosed = true
	}

	return events, nil
}

// buildCompletion constructs the sentinel-triggered completion from
// whatever finish reason this mapper's chunks captured, and s.usage —
// wholly absent (ai.Usage{}'s zero value) unless a usage chunk populated
// it along the way (R-ATS-015…016, AI-28.3). An error here means no
// terminal chunk was ever observed before the sentinel arrived (design.md
// D9, spec-silent) — the caller (stream.go's run) folds it into
// errIncompleteStream, unchanged from slice 1's own handling.
func (s *mapperState) buildCompletion() (ai.Event, error) {
	return ai.NewCompletion(s.finishReason, s.usage)
}
