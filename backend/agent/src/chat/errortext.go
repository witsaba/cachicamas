// CH-02.3 — the archetype-owned failure phrase table (R-CCP-007, R-CCP-008,
// D6). No provider-authored string ever reaches the wire: every
// ai.FailureCategory member maps to one fixed, archetype-owned English
// phrase, and a non-member value falls back to a generic one. Unwrap() and
// Error() on a failure value are never called from this file — that
// prohibition is what makes this table the ONLY source of error.Message
// text (R-CCP-008).
package chat

import "github.com/cachicamas/backend/agent/src/ai"

// phraseFor returns the fixed, archetype-owned English phrase for category
// (R-CCP-007, D6). Every ai.FailureCategory member has its own phrase; a
// value that is not a member — including the zero value — falls back to a
// generic phrase, so no category can ever reach the wire with an empty
// message.
func phraseFor(category ai.FailureCategory) string {
	switch category {
	case ai.FailureCategoryAuthentication, ai.FailureCategoryAuthorization:
		return "The assistant could not authenticate with its model provider."
	case ai.FailureCategoryRateLimit:
		return "The model provider is rate limiting requests. Try again shortly."
	case ai.FailureCategoryUnavailable:
		return "The model provider is temporarily unavailable."
	case ai.FailureCategoryTimeout:
		return "The model provider took too long to respond."
	case ai.FailureCategoryCancellation:
		return "The request was cancelled before the provider replied."
	case ai.FailureCategoryMalformedResponse:
		return "The model provider returned a response the assistant could not read."
	case ai.FailureCategoryUnsupportedCapability:
		return "The request needs a capability the model provider does not support."
	case ai.FailureCategoryUnknown:
		return "Something went wrong while talking to the model provider."
	default:
		return "Something went wrong while talking to the model provider."
	}
}
