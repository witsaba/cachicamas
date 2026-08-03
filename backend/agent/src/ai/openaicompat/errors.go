package openaicompat

import (
	"errors"
	"fmt"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ErrFrameTooLarge is returned by Feed when a single in-progress frame's
// accumulated size grows strictly greater than the decoder's configured
// hard cap (R-ASD-019). The relation is deliberately strict: a frame whose
// accumulation reaches exactly the cap, and no further, still decodes
// (S-ASD-063) — doc 0002 says a frame "exceeding" the cap aborts, and
// exactly-at-cap is not exceeding.
//
// # A recorded compromise, not a clean fit
//
// ErrFrameTooLarge names ai's malformed-response category (see Category,
// below), which is a deliberate, reasoned compromise: an over-cap frame
// may be perfectly well-formed and merely too large. AI-19's nine-member
// failure-category vocabulary has no resource-exhaustion member, and
// widening a shipped, promoted Layer 1 contract from inside this
// framing-only milestone is out of charter (R-ASD-019, R-ASD-021).
//
// The vocabulary is documented closed and append-only (R-AIP-005), so
// appending a tenth, resource-exhaustion category is the sanctioned path
// if this compromise is ever revisited — but only once a second consumer
// of the same resource-exhaustion concept exists. Doc 0002 records that
// second consumer as AI-30.1 item 4, the per-call accumulation cap (this
// node bounds a single frame; that one bounds the sum across a call).
// Widening the taxonomy belongs to the taxonomy owner as its own change,
// never to this framing-only milestone (S-ASD-068, R-ASD-021).
//
// # Distinct from ErrTruncated under errors.Is
//
// ErrFrameTooLarge and ErrTruncated (below) both name the same AI-19
// category, yet MUST stay distinguishable from each other (R-ASD-020):
// collapsing cap-exceeded and truncation into one identity would strand a
// caller with no way to tell "too large" from "cut short" short of
// inspecting message text. errors.Is(ErrFrameTooLarge, ErrTruncated) is
// always false, and vice versa.
//
// ErrFrameTooLarge wraps ai.ErrMalformedResponse with %w, so
// errors.Is(err, ai.ErrMalformedResponse) holds — the repo-native AI-19
// matching idiom — while ErrFrameTooLarge itself remains the identity a
// caller matches for the cap specifically.
//
// AI-27 never constructs an ai.Failure, ai.PreStreamFailure or
// ai.MidStreamFailure value from this sentinel (S-ASD-064): a pure byte
// decoder has no outputPreceded fact and no delivery carrier to supply
// either constructor. AI-28 and AI-32 own turning a named category into a
// constructed provider failure.
var ErrFrameTooLarge = fmt.Errorf("openaicompat: frame exceeded the configured hard cap: %w", ai.ErrMalformedResponse)

// ErrTruncated is returned by Finish (AI-27.6) when the end-of-input
// signal arrives with a partial frame still pending: a non-empty data or
// event-type accumulation, or a partial line never terminated. It is
// declared here, beside ErrFrameTooLarge, so both structural decode
// failures' identities live in one place, ahead of AI-27.6's own
// implementation of the Finish-time detection that returns it.
//
// Like ErrFrameTooLarge, ErrTruncated wraps ai.ErrMalformedResponse via
// %w and stays distinct from ErrFrameTooLarge under errors.Is (R-ASD-020).
var ErrTruncated = fmt.Errorf("openaicompat: stream ended with a partial frame pending: %w", ai.ErrMalformedResponse)

// Category reports the AI-19 failure category a decoder error names, and
// whether err is one of this package's own two sentinels at all. Both
// ErrFrameTooLarge and ErrTruncated name ai.FailureCategoryMalformedResponse
// (R-ASD-019, R-ASD-020) — the two stay distinct from each other while
// reporting the identical category, which is the proof that the
// distinction lives at the error-identity level, not the category level
// (S-ASD-067).
//
// Category never constructs an ai.Failure value of any kind: it only
// names a category from AI-19.2's vocabulary. AI-28 and AI-32 own
// construction.
func Category(err error) (ai.FailureCategory, bool) {
	if errors.Is(err, ErrFrameTooLarge) || errors.Is(err, ErrTruncated) {
		return ai.FailureCategoryMalformedResponse, true
	}
	return 0, false
}
