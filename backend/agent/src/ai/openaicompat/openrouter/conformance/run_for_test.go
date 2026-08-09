// AI-38 — the unscoped acceptance driver for the OpenRouter conformance
// bridge, plus the two surviving per-capability drivers kept as debugging
// affordances (R-ACR-001, R-CNF-028, design D1, tasks.md Phase 0 WU1).
//
// # The acceptance gate
//
// TestOpenRouterAdapter_FullConformance is the sole acceptance evidence for
// this milestone (R-ACR-001): agenttest.RunConformance runs the suite's
// whole registered case table, unscoped, against the OpenRouter conformance
// bridge factory. Every required capability MUST pass in that run; the
// driver carries no skip, no waiver, and no environment condition. It does
// NOT call t.Parallel() — the suite's cancellation cases use
// RequireNoGoroutineLeak, whose tb.Setenv call panics under a parallel
// ancestor (design D1).
//
// # Debugging affordances only (R-CNF-028)
//
// TestOpenRouterAdapter_StreamingText and TestOpenRouterAdapter_ToolCalls
// below call the suite's scoped agenttest.RunConformanceFor entry point.
// Their verdict is never cited as conformance evidence — R-CNF-028 requires
// a scoped run to be reported non-evidential — and TestOpenRouterAdapter_
// FullConformance above is what Phase 8's acceptance check reads. The three
// formerly-t.Skip'd drivers (CompletionMetadata, Cancellation, Terminal)
// are removed: their capabilities are exercised, unscoped, by the full
// conformance run instead, and each defect that run surfaces is resolved
// on a named side in Phases 2-6 (R-ACR-002) rather than skipped.

//nolint:revive // underscore in package name per task plan § PR #2 2.1: "package openrouter_conformance"
package openrouter_conformance

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
)

// TestOpenRouterAdapter_StreamingText covers R-CNF-005 +
// R-CNF-006: the bridge's real *openaicompat.Client speaks real
// HTTP to a local httptest.Server for the streaming-text cases
// (text/order_contiguity_byte_exact_reconstruction,
// text/empty_completion_is_legal). Both cases PASS.
func TestOpenRouterAdapter_StreamingText(t *testing.T) {
	t.Parallel()

	agenttest.RunConformanceFor(t, conformanceBridgeFactory(), agenttest.CapStreamingText)
}

// TestOpenRouterAdapter_ToolCalls covers R-CNF-007 + R-CNF-008:
// the bridge's real *openaicompat.Client speaks real HTTP for
// the four tool-call cases (fragmented_interleaved,
// zero_delta_whole_call, ordinal, mixed_text_and_tool_ends_on
// _tool_call_finish_reason). All four cases PASS via the
// bridge's bridgeWriteToolStartChunk / bridgeWriteToolDeltaChunk
// wire-shape renderers (R-ATL-001 / C9.1, R-ATC-010, R-ATC-012).
func TestOpenRouterAdapter_ToolCalls(t *testing.T) {
	t.Parallel()

	agenttest.RunConformanceFor(t, conformanceBridgeFactory(), agenttest.CapToolCalls)
}

// TestOpenRouterAdapter_FullConformance is the unscoped acceptance gate
// (R-ACR-001, R-CNF-028, design D1): agenttest.RunConformance runs the
// suite's whole registered case table — every required capability,
// including CapCompletionMetadata, CapCancellation and CapTerminal, which
// the now-removed per-capability drivers used to skip — against the real
// *openaicompat.Client speaking real HTTP to a local httptest.Server.
//
// This is Phase 0's RED baseline (tasks.md WU1): the failing case names
// this run reports are the binding evidence Phases 2-6 resolve against,
// each on a named side, with a recorded rationale (R-ACR-002). No
// t.Parallel(): see file header.
func TestOpenRouterAdapter_FullConformance(t *testing.T) {
	agenttest.RunConformance(t, conformanceBridgeFactory())
}