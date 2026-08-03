// AI-26.6 — the reasoning refusal, and this package's one shared refusal
// door (R-ART-015).
//
// A neutral request carrying a reasoning content part is neutrally valid:
// ai.NewRequest and ai.NewMessage both accept it (AI-10 and AI-12's own
// contract). What fails here is this vendor's own expressiveness, not the
// caller's request — which is why refuse builds an AI-19
// [ai.PreStreamFailure] carrying [ai.FailureCategoryUnsupportedCapability],
// never an AI-04 [ai.Violation]. See doc.go's "Refusal taxonomy: AI-25 vs
// AI-26" section (NFR-ART-E) for the full comparison and its AI-03 §10.4
// citation — this file implements that section's own worked example
// verbatim, not a rediscovery of it.
//
// refuse is the one place this package ever constructs that failure
// shape (design.md decision 4). AI-26.8.2's exhaustive-walk refusals (a
// later slice, for every other refuse-disposition feature its own policy
// table names) call this same function, so a later reader finds exactly
// one construction site for "this package refuses X", never one per
// site — and S-ART-054's "zero new sentinels" claim is a direct
// consequence: every refusal this package will ever produce is reachable
// only through the same, already-exported ai.ErrUnsupportedCapability
// (provider_failure.go, AI-19); this file declares no sentinel of its
// own (reasoning_refusal_test.go's TestPolicy_NoNewSentinelsExported
// keeps that claim mechanical and live for every future change here).

package openaicompat

import (
	"errors"

	"github.com/cachicamas/backend/agent/src/ai"
)

// refuse constructs the refusal failure naming feature as the unsupported
// capability (S-ART-053): an AI-19 [ai.PreStreamFailure] whose category
// is [ai.FailureCategoryUnsupportedCapability] and whose Cause names
// feature in prose. It is reachable two ways, deliberately: the typed
// check every caller should reach for first,
// errors.Is(err, ai.ErrUnsupportedCapability) — routed through
// [ai.Failure.Is], never a string compare — to learn THAT a capability
// was refused; and errors.Unwrap(err).Error(), which returns feature
// verbatim, to learn WHICH one, without reading this package's source
// ([ai.Failure.Error] deliberately never reproduces its own wrapped
// Cause's text — provider_failure.go's own documented redaction
// posture — so this is the only path to the prose).
//
// [ai.PreStreamFailure]'s own error return is unreachable from here:
// [ai.FailureCategoryUnsupportedCapability] is always a member of the
// closed vocabulary [ai.FailureCategory.Validate] checks, so construction
// cannot fail for a category this function always supplies. Reaching
// that branch anyway would be this package's own defect, not a caller's,
// so it panics rather than silently downgrading to a less specific error
// a caller could not act on.
func refuse(feature string) error {
	failure, err := ai.PreStreamFailure(ai.FailureReport{
		Category: ai.FailureCategoryUnsupportedCapability,
		Cause:    errors.New(feature + " is not supported by this adapter"),
	})
	if err != nil {
		panic("openaicompat: ai.PreStreamFailure construction failed unexpectedly: " + err.Error())
	}
	return failure
}

// refuseReasoning reports refuse's own error for req's first reasoning
// content part — walking every message in req.Messages(), and within
// each, every part of message.Content(), both in order — or nil when req
// carries none at all (R-ART-015).
//
// Translate (translation.go) calls this before appendBody runs, at all:
// R-ART-015 requires translation to fail with NO wire body produced, in
// every reasoning state and at every position (S-ART-051, S-ART-052) —
// including a reasoning part that sits after one or more otherwise fully
// renderable messages, or after other parts within its own message — so
// the whole request has to be checked to completion before a single byte
// of the body is appended, never discovered mid-render after some of the
// body already exists. Scanning every message and every one of its parts,
// rather than stopping at the first message or the first part, is what
// makes the outcome identical regardless of which message or which
// position within a message carries the part
// (reasoning_refusal_test.go's own position-varying registered cases
// prove this directly, not by inspection).
//
// Only an ai.RoleAssistant message can carry a PartKindReasoning part at
// all (request.go's own rolePermittedKinds table, enforced by
// ai.NewRequest before Translate is ever reached), so this function's own
// walk needs no role check of its own to stay correct: it simply never
// finds one in a message of another role.
//
// Because this runs before appendBody, appendContentPartObject's own
// default-case panic (message.go) is no longer reachable for
// PartKindReasoning specifically, at any position, in any message — see
// that function's own doc comment, updated alongside this file landing.
func refuseReasoning(req ai.Request) error {
	for _, message := range req.Messages() {
		for _, part := range message.Content() {
			if part.Kind() == ai.PartKindReasoning {
				return refuse("reasoning content")
			}
		}
	}
	return nil
}
