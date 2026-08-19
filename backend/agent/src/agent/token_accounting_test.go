// AG-17.2 — strict-TDD tests for token accounting's pure, package-
// internal surfaces (R-CTX-006..010, S-CTX-012, S-CTX-016..018, bite
// S-CTX-013).
//
// This file is package-internal (NFR-CTX-001's own carve-out, the
// AG-16 cost_usage_test.go precedent, archive design.md:176): the
// estimate's own table and the resolver's three-state table are pure
// functions with no external surface. Every claim about what a
// STRATEGY observes is asserted externally, in context_strategy_test.go,
// through a recording strategy.
package agent

import "testing"

// TestTokenSource_StringRendersDistinctly — part of S-CTX-014 (task
// 1.8). The three provenance states render to three distinct,
// non-empty strings (ai/usage.go:72-77's log-line argument).
func TestTokenSource_StringRendersDistinctly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source TokenSource
		want   string
	}{
		{"zero value renders unavailable", TokenSourceUnavailable, "unavailable"},
		{"reported", TokenSourceReported, "reported"},
		{"estimated", TokenSourceEstimated, "estimated"},
	}

	seen := map[string]bool{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.source.String()
			if got != tt.want {
				t.Errorf("TokenSource(%d).String() = %q, want %q", tt.source, got, tt.want)
			}
			seen[got] = true
		})
	}
	if len(seen) != 3 {
		t.Errorf("the three TokenSource values rendered %d distinct string(s), want 3: %v", len(seen), seen)
	}
}
