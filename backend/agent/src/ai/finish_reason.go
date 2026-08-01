// AI-13.1 and AI-13.2 — the finish-reason vocabulary.
//
// This file implements V-MET-01 … V-MET-08: the closed vocabulary stating why
// generation stopped. The register requires it to be "closed and complete from
// birth, because collapsing distinct stop conditions into a fallback is a
// loop-termination defect above Layer 1, not a cosmetic gap". Refusal and
// pause-turn are therefore here in the first version of the file rather than
// added to a frozen enum later — doc 0001's G12(c), absorbed.
//
// # What this file decides, and what it refuses to decide
//
// It decides which stop conditions are distinguishable and how a provider's own
// string becomes one of them. It does not decide what a consumer should do
// about any of them: V-OUT-10 reserves loop termination for Layer 2, and "the
// finish reason is the input to that decision, not the decision". So each value
// documents the obligation it places on a consumer, and no method answers it.
//
// # The zero value names nothing
//
// The constant block starts at a blank identifier, so FinishReason(0) is not a
// member of the vocabulary and Validate rejects it. This costs one value and
// buys the distinction V-MET-11 draws for token counts, applied here: "nobody
// set this" and "the provider said something I do not recognise" stay two
// facts. Were unknown the zero value, a completion built without a finish
// reason would report a deliberate, recorded observation nobody made.

package ai

import "strings"

// FinishReason is the closed vocabulary value stating why generation stopped
// (V-MET-01).
//
// Its zero value is not a member of the vocabulary; see the file documentation.
// Values are compared for equality and switched on; a consumer's switch should
// be exhaustive over the seven values below, because adding an eighth is a
// contract change this package's own tests are built to make visible.
type FinishReason uint8

// The vocabulary. Seven values, complete from birth (V-MET-02 … V-MET-08).
const (
	_ FinishReason = iota // the zero value names no finish reason

	// FinishReasonStop is V-MET-02: generation ended because the model
	// finished. The consumer's obligation is nothing — the turn is complete
	// and the response is whole.
	FinishReasonStop

	// FinishReasonLength is V-MET-03: generation ended because an output limit
	// was reached. The consumer's obligation is to treat the response as
	// truncated: it is not a complete answer, and continuing it is a different
	// request rather than a retry.
	FinishReasonLength

	// FinishReasonToolCalls is V-MET-04: generation ended because the model
	// requested one or more tool calls. The consumer's obligation is to
	// schedule them and continue the turn with their results.
	FinishReasonToolCalls

	// FinishReasonContentFilter is V-MET-05: generation ended because a
	// provider-side content filter intervened.
	//
	// This is not FinishReasonRefusal and the line between them is worth
	// stating where it will be read. A content filter is a decision taken
	// *beside* the model, before or after it, by the provider. The model may
	// have been willing; a consumer's correct response may therefore be a
	// different model, a different provider, or surfacing a policy message —
	// none of which is the right response to a model that declined.
	FinishReasonContentFilter

	// FinishReasonRefusal is V-MET-06: generation ended because the model
	// **declined**. Distinct from unknown and from pause: a decline is a final
	// answer.
	//
	// The decision is the model's own, and it is complete: the response is an
	// answer of the form "no". The consumer's obligation is to treat it as a
	// finished turn. Retrying the same request is not a recovery, and routing
	// it to a fallback provider as if it were a failure is how a refusal
	// becomes an infinite loop.
	FinishReasonRefusal

	// FinishReasonPauseTurn is V-MET-07: generation **paused** and can be
	// resumed. Distinct from a decline and from unknown: the correct response
	// is to resume, not to stop or to guess.
	//
	// The consumer's obligation, recorded here for AI-31.1 and Layer 2 to
	// honor: resume the turn, replaying the content already received
	// **verbatim**. A paused turn is one turn split across two provider calls,
	// so content that is re-sent altered — re-serialised, re-ordered, or
	// stripped of an opaque round-trip token — is a different turn.
	FinishReasonPauseTurn

	// FinishReasonUnknown is V-MET-08: the recorded outcome for a provider stop
	// condition this vocabulary does not recognise. A recorded state — "I do
	// not recognise this string" — never a silent substitute for refusal or
	// pause.
	//
	// The consumer's obligation is to treat the turn as ended and to make the
	// unrecognised condition visible, because the value means the vocabulary
	// is behind the provider rather than that the provider said nothing.
	FinishReasonUnknown

	// finishReasonLimit is the first value outside the vocabulary. It must stay
	// last.
	//
	// It lives inside the constant block rather than being written as a literal
	// so that appending a value moves the bound with it, with no second edit to
	// remember. That is half of the exhaustiveness pin: a value appended above
	// this line starts out valid, has no string form and does not round-trip, so
	// it fails three assertions rather than passing unnoticed. A value appended
	// *below* this line is outside the vocabulary and fails validation wherever
	// it is used, which is loud in a different way.
	finishReasonLimit
)

// finishReasonNames is the stable string form of each value.
//
// It is an array sized by the bound, not a map: the size follows the vocabulary
// automatically, so an appended constant gets an empty entry and renders as the
// placeholder instead of silently inheriting a neighbour's name.
var finishReasonNames = [finishReasonLimit]string{
	FinishReasonStop:          "stop",
	FinishReasonLength:        "length",
	FinishReasonToolCalls:     "tool_calls",
	FinishReasonContentFilter: "content_filter",
	FinishReasonRefusal:       "refusal",
	FinishReasonPauseTurn:     "pause_turn",
	FinishReasonUnknown:       "unknown",
}

// unnamedFinishReason renders a value that is not in the vocabulary.
//
// It is deliberately not a vocabulary word and deliberately not "unknown":
// "unknown" is a recorded outcome about a provider, and this is a statement
// about a value that names nothing.
const unnamedFinishReason = "invalid"

// String returns the stable string form of the finish reason.
//
// The form is this package's own spelling, not any provider's, and it round-
// trips: NormalizeFinishReason(r.String()) == r for every member of the
// vocabulary. A value outside the vocabulary — including the zero value —
// renders as a fixed placeholder and never panics.
func (r FinishReason) String() string {
	if r == 0 || r >= finishReasonLimit || finishReasonNames[r] == "" {
		return unnamedFinishReason
	}
	return finishReasonNames[r]
}

// Validate reports whether the finish reason is a member of the closed
// vocabulary, positioned at the given path.
//
// A value outside the vocabulary is a caller-contract failure of class
// [ErrNotInVocabulary] — V-REQ-01's rule, applied to this vocabulary: a value
// outside it is a caller's bug, not an unknown passed through. [FinishReasonUnknown]
// is inside the vocabulary and therefore valid; it is what "passed through"
// looks like when the provider, rather than the caller, said something
// unrecognised.
func (r FinishReason) Validate(at ...Step) error {
	return FirstFailure(func() *Violation {
		if r == 0 || r >= finishReasonLimit {
			return Invalid(ErrNotInVocabulary, at...)
		}
		return nil
	})
}

// finishReasonBySpelling maps the provider stop values seen in the wild onto
// the vocabulary.
//
// The map is only ever looked up, never iterated: AI-04's rule that no
// unordered iteration may decide anything in this package holds, because a
// lookup is deterministic and an iteration over this table decides nothing.
//
// Per-vendor mapping is AI-31.1's, not this table's. What lives here are the
// spellings common enough that every adapter would otherwise special-case them
// identically.
// Every key is already trimmed and lower-cased, because that is the form
// NormalizeFinishReason looks up.
var finishReasonBySpelling = map[string]FinishReason{
	"stop":               FinishReasonStop,
	"end_turn":           FinishReasonStop,
	"stop_sequence":      FinishReasonStop,
	"complete":           FinishReasonStop,
	"length":             FinishReasonLength,
	"max_tokens":         FinishReasonLength,
	"max_output_tokens":  FinishReasonLength,
	"tool_calls":         FinishReasonToolCalls,
	"tool_use":           FinishReasonToolCalls,
	"function_call":      FinishReasonToolCalls,
	"content_filter":     FinishReasonContentFilter,
	"safety":             FinishReasonContentFilter,
	"recitation":         FinishReasonContentFilter,
	"prohibited_content": FinishReasonContentFilter,
	"refusal":            FinishReasonRefusal,
	"refused":            FinishReasonRefusal,
	"pause_turn":         FinishReasonPauseTurn,
	"pause":              FinishReasonPauseTurn,
	"unknown":            FinishReasonUnknown,
	"other":              FinishReasonUnknown,
}

// NormalizeFinishReason maps a provider's own stop value into the vocabulary.
//
// The value is trimmed of surrounding whitespace and lower-cased before it is
// matched, because vendors disagree about casing and a table that matched only
// one convention would send every stop condition of the other vendor to
// unknown.
//
// It is **total**. Every string yields a finish reason, including the empty one,
// and an unrecognised value yields [FinishReasonUnknown]. There is no error
// return, because there is no failure: V-MET-08 makes an unrecognised provider
// stop condition a *recorded outcome*, not a fault. Providers add stop values
// without notice, and a normalizer that panicked or errored on one would turn a
// vendor release note nobody read into a Layer 1 crash.
//
// It is deliberately no cleverer than this. No hyphen folding, no prefix match,
// no nearest-neighbour: every additional inference is a rule some vendor's next
// spelling will satisfy by accident.
func NormalizeFinishReason(providerStopValue string) FinishReason {
	spelling := strings.ToLower(strings.TrimSpace(providerStopValue))
	if reason, ok := finishReasonBySpelling[spelling]; ok {
		return reason
	}
	return FinishReasonUnknown
}
