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

import (
	"context"

	"github.com/cachicamas/backend/agent/src/ai"
)

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

// resolveTokenAccounting resolves the coming logical turn's measured
// size (R-CTX-006, R-CTX-007). Package-private — no Layer 3 exists yet
// (0003:110); external tests read ContextPrompt.Accounting through a
// recording strategy (the AG-16 cost_usage surface argument).
//
// The three states, in the order this resolver decides them:
//  1. buildLoopRequest fails -> Unavailable. The turn's own build
//     (loop.go:304) stays the single abort authority; duplicating
//     that decision would create a second failure site for one
//     condition.
//  2. provider does not satisfy ai.TokenCounter (a clean absence,
//     R-AMP-018) -> Estimated, via estimateTokens.
//  3. provider satisfies ai.TokenCounter:
//     - CountTokens errors, or answers with a nil error and an
//     absent count -> Unavailable, NEVER Estimated. An advertised
//     counter that declines is non-conformant, not absent
//     (R-AMP-019) — falling back to the estimate would launder it
//     into a working one.
//     - CountTokens answers with a present count -> Reported.
func resolveTokenAccounting(ctx context.Context, provider ai.ModelProvider, opts TurnOptions, system string, transcript []ai.Message) TokenAccounting {
	req, err := buildLoopRequest(opts, system, transcript) // loop.go:713-726 — the SAME builder the turn uses
	if err != nil {
		return TokenAccounting{}
	}

	counter, ok := provider.(ai.TokenCounter) // the ONLY discovery mechanism (ai/provider.go:112-113)
	if !ok {
		return TokenAccounting{tokens: estimateTokens(req), source: TokenSourceEstimated}
	}

	tc, cerr := counter.CountTokens(ctx, req)
	n, present := tc.Count()
	if cerr != nil || !present {
		return TokenAccounting{}
	}
	return TokenAccounting{tokens: n, source: TokenSourceReported}
}

// estimateBytesPerToken (D) = 4: byte-level BPE averages ~4 UTF-8
// bytes per token on mixed prose; bytes, not runes, because a 3-byte
// CJK ideograph is roughly one token (runes under-count it ~4x, bytes
// under 2x). Errs toward over-counting dense ASCII — the safe
// direction for a budget check (compaction asked earlier, never later).
const estimateBytesPerToken = 4

// estimateTokensPerMessage (M) = 4: chat encodings add role and
// delimiter tokens per message that no byte count sees (~3-4 per
// message); 4 keeps the same conservative direction. Dominant on
// short-message transcripts, where byte counting alone is worst.
const estimateTokensPerMessage = 4

// estimateTokens = ceil(B/D) + M*K over req (R-CTX-009). PURE: no
// clock, no env, no randomness, no I/O — TestEstimateTokens_
// NoClockNoIO_SourceScan asserts this directly, because the
// ambient-authority guard forbids only os, os/exec, syscall,
// io/ioutil (ambient_authority_test.go:73-94) — NOT time.
//
// Approximate BY CONSTRUCTION, unbounded in both directions: it
// over-counts dense ASCII, under-counts non-Latin scripts, base64-like
// content, and unusual tool schemas. Its result is only ever labelled
// TokenSourceEstimated — R-CTX-008 makes it mechanically unreadable
// without that label, so no path can treat it as exact.
//
// B is the exact accessor walk (R-CTX-009), pinned so a later Layer 1
// field cannot be missed silently:
//   - the system instruction's segment text;
//   - every message's content parts: text, tool-call name+arguments,
//     tool-result content, and reasoning text;
//   - every declared tool's name, description and schema.
func estimateTokens(req ai.Request) int64 {
	var b int64

	if system, ok := req.SystemInstruction(); ok {
		for _, seg := range system.Segments() {
			b += int64(len(seg.Text()))
		}
	}

	messages := req.Messages()
	for _, msg := range messages {
		for _, part := range msg.Content() {
			if text, ok := part.Text(); ok {
				b += int64(len(text))
			}
			if call, ok := part.ToolCall(); ok {
				b += int64(len(call.Name()))
				b += int64(len(call.Arguments()))
			}
			if result, ok := part.ToolResult(); ok {
				b += int64(len(result.Content()))
			}
			if reasoning, ok := part.Reasoning(); ok {
				b += int64(len(reasoning.Text()))
			}
		}
	}

	if tools, ok := req.Tools(); ok {
		for _, tool := range tools.Tools() {
			b += int64(len(tool.Name()))
			b += int64(len(tool.Description()))
			b += int64(len(tool.Schema()))
		}
	}

	k := int64(len(messages))
	return (b+estimateBytesPerToken-1)/estimateBytesPerToken + estimateTokensPerMessage*k
}
