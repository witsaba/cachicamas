// AI-26.6 — the reasoning refusal, and this package's one shared refusal
// door (R-ART-015). AI-26.8 grows this file with the total feature policy
// (R-ART-020, R-ART-021): featurePolicy, disposition and
// resolveFeaturePolicy, below — refuse and refuseReasoning are unchanged by
// that growth, reused exactly as Phase 7 left them (see this file's own
// growth section, below, for the verified hand-off).
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

// AI-26.8.2 — the total feature policy (R-ART-021).
//
// Verified before writing this table, not assumed from Phase 7's own
// hand-off (tasks.md's Phase 7 closure, items 1-2): refuse's bare
// feature-string signature needs no change — nothing below constructs a
// second refusal shape, and every refuse-disposition row's own witness
// (policy_walk_test.go's refuseWitnesses) still resolves through
// refuseReasoning, the one existing call site. "Reasoning" — the PartKind
// and its three ReasoningState members — is confirmed the ONLY
// refuse-disposition feature this milestone's own mechanical inventory
// produces (feature_inventory_test.go's discoverVocabularyFeatures against
// this table, cross-checked by
// TestFeatureInventory_MatchesProductionPolicyExactly): nothing in AI-12's
// ten With* constructors and nothing in the other four vocabularies needs a
// second refusal path. refuseReasoning's own pre-appendBody, whole-request
// walk (translation.go's Translate) therefore needed no change either: this
// slice adds no second request-level check for Translate to compose with
// it.

// disposition is how this adapter treats one inventoried feature: render it
// onto the wire, drop it silently for a recorded reason, or refuse the
// whole request naming it. Every inventoried feature (26.8.1) resolves to
// exactly one — R-ART-021's own "total" requirement — never zero (silently
// unaccounted, R-ART-020's forbidden outcome) and never more than one
// (S-ART-074, feature_inventory_test.go's
// TestFeatureInventory_NoDuplicateFeatureNamesInPolicy).
type disposition uint8

const (
	// dispositionTranslate renders the feature onto the wire, unconditionally
	// faithful to what the caller supplied. Every case where this package
	// already renders a caller's own value verbatim.
	dispositionTranslate disposition = iota + 1

	// dispositionDrop renders no wire artefact for the feature at all, for a
	// recorded reason (S-ART-076) — this adapter's own already-established
	// "vanish whole" contract (cache-boundary markers; a foreign-namespace
	// provider extension; see doc.go's own sections for each), never a
	// partial or best-effort rendering.
	dispositionDrop

	// dispositionRefuse fails the whole request through refuse (above),
	// naming the feature, before appendBody renders anything (R-ART-021,
	// reusing R-ART-015's one door — never a second one).
	dispositionRefuse
)

// policyEntry is one row of featurePolicy: one inventoried feature, its
// disposition, and why (design.md decision 6).
type policyEntry struct {
	feature     string
	disposition disposition
	reason      string
}

// featurePolicy is the total policy table (R-ART-021): every feature
// feature_inventory_test.go's mechanical inventory discovers has exactly
// one row here — 4 content-part kinds, 3 reasoning states, 4 tool-choice
// modes, 3 cache regions and 3 roles (the five runtime-enumerated
// vocabularies, 17 rows total), plus 9 unsplit With* constructors and
// WithProviderExtension's own two-way split (design.md decision 6), 11
// rows total — 28 rows overall, verified against the live ai package by
// TestFeatureInventory_MatchesProductionPolicyExactly rather than assumed
// from this count.
//
// A slice, never a map — this package's own "nothing may let an unordered
// iteration decide anything" discipline (design.md "Map discipline"),
// restated here for an audit table rather than the wire path: a row's own
// position carries no meaning Translate depends on, but the convention
// stays uniform rather than becoming "the wire path only", a special case
// a later reader would have to remember. A slice is also what lets
// TestFeatureInventory_NoDuplicateFeatureNamesInPolicy treat a repeated
// feature name as a mechanical failure rather than a silent map overwrite.
//
// Three (not one) rows record dispositionRefuse — "PartKind: reasoning"
// plus its own three "ReasoningState: *" rows — and three (not one) record
// the cache-region "vanishes whole" reason under dispositionDrop. This is
// deliberate redundancy, not oversight: every runtime-enumerated
// vocabulary is walked, and cross-checked against this table, at the SAME
// member-level granularity (feature_inventory_test.go's
// discoverVocabularyFeatures), so a member-level row exists uniformly for
// all five vocabularies rather than five different, individually
// special-cased shapes. See doc.go's "The policy is total" section for the
// full rationale.
var featurePolicy = []policyEntry{
	// AI-10 base surface — content-part kinds (ai.PartKinds()).
	{feature: "PartKind: text", disposition: dispositionTranslate,
		reason: "model-visible text renders as this vendor's own string/content-part-object shape (R-ART-007)"},
	{feature: "PartKind: reasoning", disposition: dispositionRefuse,
		reason: "this vendor's wire schema carries no field for reasoning content (R-ART-015)"},
	{feature: "PartKind: tool_call", disposition: dispositionTranslate,
		reason: "renders as an element of the assistant message's own tool_calls array (R-ART-012)"},
	{feature: "PartKind: tool_result", disposition: dispositionTranslate,
		reason: "renders as its own distinct role:\"tool\" wire message (R-ART-012)"},

	// AI-10 base surface — reasoning states (ai.ReasoningStates()). Every
	// state is swept up by the "PartKind: reasoning" row above via the same
	// single mechanism (refuseReasoning matches on PartKind, not on
	// ReasoningState); these three rows exist so the mechanical,
	// member-level inventory walk has one entry per member to cross-check,
	// not so a caller could pick a different disposition per state.
	// reasoning_refusal_test.go's own 9 registered refusalCases already
	// prove all three refuse identically (S-ART-051).
	{feature: "ReasoningState: text", disposition: dispositionRefuse,
		reason: "a PartKind: reasoning part in this state refuses identically to every other state (R-ART-015)"},
	{feature: "ReasoningState: redacted", disposition: dispositionRefuse,
		reason: "a PartKind: reasoning part in this state refuses identically to every other state (R-ART-015)"},
	{feature: "ReasoningState: token-only", disposition: dispositionRefuse,
		reason: "a PartKind: reasoning part in this state refuses identically to every other state (R-ART-015)"},

	// AI-10 base surface — tool-choice modes (ai.ToolChoiceModes()).
	{feature: "ToolChoiceMode: auto", disposition: dispositionTranslate,
		reason: "renders as the bare wire string \"auto\" (R-ART-016)"},
	{feature: "ToolChoiceMode: none", disposition: dispositionTranslate,
		reason: "renders as the bare wire string \"none\" (R-ART-016)"},
	{feature: "ToolChoiceMode: required", disposition: dispositionTranslate,
		reason: "renders as the bare wire string \"required\" (R-ART-016)"},
	{feature: "ToolChoiceMode: specific", disposition: dispositionTranslate,
		reason: "renders as {\"type\":\"function\",\"function\":{\"name\":...}} (R-ART-016)"},

	// AI-11 cache markers — cache regions (ai.CacheRegions()). Every region
	// is dropped whole for the identical one reason (AI-11.3's advisory
	// contract, doc.go's "Cache-boundary markers vanish whole"); three rows
	// for the same member-level-uniformity reason the reasoning states above
	// have three, not because any region is treated differently from
	// another — cache_marker_test.go's own walk already proves all three
	// drop identically (S-ART-022).
	{feature: "CacheRegion: tools", disposition: dispositionDrop,
		reason: "this vendor caches automatically and exposes no client-supplied cache-boundary annotation (AI-11.3, R-ART-006)"},
	{feature: "CacheRegion: system", disposition: dispositionDrop,
		reason: "this vendor caches automatically and exposes no client-supplied cache-boundary annotation (AI-11.3, R-ART-006)"},
	{feature: "CacheRegion: messages", disposition: dispositionDrop,
		reason: "this vendor caches automatically and exposes no client-supplied cache-boundary annotation (AI-11.3, R-ART-006)"},

	// AI-10 base surface — roles (ai.Roles()).
	{feature: "Role: user", disposition: dispositionTranslate,
		reason: "renders as an ordinary {\"role\":\"user\",...} wire message"},
	{feature: "Role: assistant", disposition: dispositionTranslate,
		reason: "renders as an ordinary {\"role\":\"assistant\",...} wire message, its own tool_calls array included when present"},
	{feature: "Role: tool", disposition: dispositionTranslate,
		reason: "renders as one or more distinct role:\"tool\" wire messages, never a content-part element on another message (R-ART-012)"},

	// AI-12 options and pass-through — the ten AST-scanned With* option
	// constructors (request.go, request_extension.go, system_instruction.go
	// — package ai).
	{feature: "WithModel", disposition: dispositionTranslate,
		reason: "the model identity renders verbatim as the wire \"model\" field (R-ART-002)"},
	{feature: "WithMessages", disposition: dispositionTranslate,
		reason: "each message renders per its own role and content (R-ART-007..R-ART-014)"},
	{feature: "WithMaxOutputTokens", disposition: dispositionTranslate,
		reason: "renders as the wire \"max_tokens\" field when set, explicitly absent when not (R-ART-018)"},
	{feature: "WithTemperature", disposition: dispositionTranslate,
		reason: "renders as the wire \"temperature\" field when set (R-ART-016)"},
	{feature: "WithTopP", disposition: dispositionTranslate,
		reason: "renders as the wire \"top_p\" field when set (R-ART-016)"},
	{feature: "WithStopSequences", disposition: dispositionTranslate,
		reason: "renders as the wire \"stop\" array field when set (R-ART-016)"},
	{feature: "WithTools", disposition: dispositionTranslate,
		reason: "renders as the wire \"tools\" array, byte-faithful schema splice (R-ART-010, R-ART-011)"},
	{feature: "WithToolChoice", disposition: dispositionTranslate,
		reason: "renders as the wire \"tool_choice\" field (R-ART-016)"},
	{feature: "WithSystemInstruction", disposition: dispositionTranslate,
		reason: "each segment renders as its own role-\"system\" wire message, in order (R-ART-005)"},
	{feature: "WithProviderExtension: own namespace", disposition: dispositionTranslate,
		reason: "this adapter's own reserved namespace merges its bytes into the wire body, raw (R-ART-019)"},
	{feature: "WithProviderExtension: foreign namespace", disposition: dispositionDrop,
		reason: "a namespace other than this adapter's own reserved Namespace is never read or rendered (R-ART-019)"},
}

// resolveFeaturePolicy returns feature's own row in featurePolicy, and
// whether one exists — S-ART-074's "none unresolved" is a caller checking
// this second result, not a caller that gets some zero-value disposition
// back by accident.
//
// A linear scan over the slice, not a map lookup: featurePolicy is a slice
// for the same reason every other registry in this package and its
// neighbours is (design.md "Map discipline") — restated here for an audit
// table that carries no wire-path consequence of its own, so the
// convention stays uniform across every registry this package declares
// rather than being "the wire path only", a special case a later reader
// would have to remember. 28 rows makes the scan's cost immaterial.
func resolveFeaturePolicy(feature string) (policyEntry, bool) {
	for _, entry := range featurePolicy {
		if entry.feature == feature {
			return entry, true
		}
	}
	return policyEntry{}, false
}
