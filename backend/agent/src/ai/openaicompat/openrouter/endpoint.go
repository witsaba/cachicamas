package openrouter

// openrouterBaseURL is the only endpoint this wrapper targets (D1, D3):
// the OpenRouter HTTP gateway for the OpenAI-compatible Chat Completions
// dialect. A caller needing a different endpoint uses the shipped
// openaicompat package directly — there is no Endpoint field on Config,
// deliberately, so a wrapper built with this package always speaks to
// OpenRouter and only OpenRouter.
const openrouterBaseURL = "https://openrouter.ai/api/v1"

// openrouterDefaultModel is the model identifier this wrapper substitutes
// when Config.Model is the zero value (R-OR-03, R-OR-05). It is a paid,
// non-reasoning model whose capabilities preserve AI-29's struck verdict
// (CAP-O-01 = absent): the conformance bridge declares reasoning
// explicitly false, the capability record records CAP-O-01 as absent, and
// TestOpenRouterAdapter_DefaultModelChangeFailsUntilSpecAmendment pins
// this constant value as a build-time gate against a silent default
// swap (R-OR-05 sub-scenario 2). Switching the default to a reasoning-
// capable model requires an explicit ADR AND a spec amendment that
// reopens AI-29's struck verdict under trigger #1 — never a silent
// edit of this constant.
const openrouterDefaultModel = "openai/gpt-4o"

// effectiveModel returns the model identifier NewProvider wires into the
// wrapper's request-intercepting transport. When cfg.Model is non-empty
// the caller's deliberate value wins; when zero, openrouterDefaultModel
// substitutes. The decision is made once at construction so the wire
// never sees a zero-value model on its outbound request body.
func effectiveModel(cfg Config) string {
	if cfg.Model != "" {
		return cfg.Model
	}
	return openrouterDefaultModel
}