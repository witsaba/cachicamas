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
// # The terminal→sentinel window (R-ATS-008, C-1 corrective)
//
// Once a chunk sets terminalSeen, applyChunk closes an open block on that
// same chunk. Every later chunk this slice's own scenarios exercise in that
// window is delta-less or the usage chunk (an empty choices array, C4) —
// both are absorbed as zero events by construction: hasChoice is false for
// the usage chunk, and a delta-less choice item contributes nothing to
// either the finish-reason gate or the content check. A choice-0 content
// STRING arriving in that window is R-ATS-020 row 1's own violation
// (S-ATS-074) — the 5-row table living in AI-28.5, a later slice. No
// scenario this slice implements exercises that shape; this mapper does
// not yet special-case it, and this is a disclosed, deliberate deferral,
// not an oversight — see the slice-2 apply-progress record.

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

// mapperState is AI-28.1.2's frame-to-events mapper. Its zero value is
// ready to use: no chunk has been read yet, no block minted, no terminal
// seen — the same "ready without a constructor" posture ai.Stamper's own
// zero value carries.
type mapperState struct {
	// identityEstablished is true once the first chunk's id/model produced
	// a ResponseStart (R-ATS-004).
	identityEstablished bool

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

	if !s.identityEstablished {
		started, err := ai.NewResponseStart(chunk.ID, chunk.Model)
		if err != nil {
			return nil, errMalformedIdentity
		}
		s.identityEstablished = true
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

	if choice.FinishReason != nil {
		reason, ok := rawStrictFinishReason(*choice.FinishReason)
		if !ok {
			return nil, errUnrecognizedFinishReason
		}
		s.finishReason = reason
		s.terminalSeen = true
	}

	if text, present := contentText(choice.Delta.Content); present {
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
