// AG-17.2 — token accounting: type-level provenance and the byte-based
// estimate (R-CTX-006..011).
//
// Lives here, not in cost_usage.go (design.md DD4): that file's
// package comment scopes it to AG-16 end-to-end, and both milestones'
// single-revert rollback plans depend on whole-file deletion — sharing
// a file would entangle them. Composition, not duplication: the
// resolver reads the reported figure through the same public
// TokenCount.Count() idiom cost_usage.go names as its own route, and
// touches no cost_* file.

package agent

// TokenSource is the provenance of a token figure (R-CTX-008).
// The zero value is Unavailable, so a zero TokenAccounting{} reads as
// "no figure", never as "0 tokens".
type TokenSource int

const (
	// TokenSourceUnavailable is the zero value: no figure exists.
	// Reached when accounting cannot resolve a request to measure
	// (a buildLoopRequest failure), or when an ADVERTISED counter
	// declines to answer — an error, or a nil error with an absent
	// count (R-AMP-019: advertising binds; this is non-conformance,
	// never a clean absence, and is NEVER estimated).
	TokenSourceUnavailable TokenSource = iota

	// TokenSourceReported means the provider's ai.TokenCounter
	// reported this figure FOR THE PRE-HOOK REQUEST (R-CTX-011) —
	// never a claim about the bytes actually sent, because
	// applyPreRequestHook (loop.go:326) derives a new request AFTER
	// this seam's own request build (loop.go:304), downstream of the
	// turn boundary where accounting runs.
	TokenSourceReported

	// TokenSourceEstimated means Layer 2's documented heuristic
	// (estimateTokens) produced this figure. Reached ONLY from a
	// FAILED type assertion on ai.TokenCounter — a clean absence of
	// the capability (R-AMP-018) — and from no other path.
	TokenSourceEstimated
)

// String renders the three states distinctly (ai/usage.go:72-77's
// log-line argument).
func (s TokenSource) String() string {
	switch s {
	case TokenSourceReported:
		return "reported"
	case TokenSourceEstimated:
		return "estimated"
	default:
		return "unavailable"
	}
}

// TokenAccounting is a token figure that cannot be read without its
// provenance (R-CTX-008). Unexported fields; the only accessor is
// two-result.
type TokenAccounting struct {
	tokens int64
	source TokenSource
}

// Tokens returns the figure and its provenance. A consumer physically
// cannot obtain the number without also obtaining the source — the
// mechanical enforcement of "an estimate never masquerades as exact".
func (a TokenAccounting) Tokens() (int64, TokenSource) { return a.tokens, a.source }
