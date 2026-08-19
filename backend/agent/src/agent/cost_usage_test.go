// AG-16 — S-CST-007, the ai.Usage → payload conversion table test
// (R-CST-003).
//
// This is the ONE test in this change that is package-internal rather
// than package agent_test. NFR-CST-001 states the default ("every
// behavioral test MUST live in package agent_test") and its own
// carve-out in the same sentence: "R-CST-003's conversion MAY
// additionally be exercised through an internal table test, provided
// every behavioral claim above is also observable externally."
// costFromUsage is deliberately package-private (design DD1: no
// Layer 3 exists yet, external tests read events through the
// accessors, and a smaller surface keeps the rollback clean), so a
// directly-callable test of the conversion itself cannot be written
// from agent_test at all — Go's export rule leaves no other route.
// Every behavioral claim this conversion feeds is independently and
// externally proven by cost_turn_emission_test.go and
// cost_session_test.go (S-CST-001..006, 008..013), satisfying
// NFR-CST-001's proviso.
//
// Both substrate filters carry this file's exact suffix
// (cost_usage_test.go), landed in the same commit that introduces it,
// per R-LSK-004's naming discipline.
package agent

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// TestCostFromUsage_TableDriven — S-CST-007. The conversion is total
// (defined for every ai.Usage value, the zero value included), pure,
// and presence-preserving in both directions: a reported value maps
// to a reported figure of the same magnitude, and an unreported value
// maps to an absent figure. ai.Tokens(0) MUST map to a reported
// nought and MUST NOT be conflated with absence.
func TestCostFromUsage_TableDriven(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		usage       ai.Usage
		wantFigures CostFigures
		wantPres    costPresence
	}{
		{
			name:        "empty ai.Usage — every figure absent",
			usage:       ai.Usage{},
			wantFigures: CostFigures{},
			wantPres:    costPresence{},
		},
		{
			name: "fully reported usage — every figure present at its magnitude",
			usage: ai.Usage{
				Input:      ai.Tokens(1200),
				Output:     ai.Tokens(340),
				CacheRead:  ai.Tokens(80),
				CacheWrite: ai.Tokens(20),
				Reasoning:  ai.Tokens(96),
			},
			wantFigures: CostFigures{
				InputTokens:      1200,
				OutputTokens:     340,
				CacheReadTokens:  80,
				CacheWriteTokens: 20,
				ReasoningTokens:  96,
			},
			wantPres: costPresence{input: true, output: true, cacheRead: true, cacheWrite: true, reasoning: true},
		},
		{
			name: "usage reporting only zeros — every figure present at zero, never absent",
			usage: ai.Usage{
				Input:      ai.Tokens(0),
				Output:     ai.Tokens(0),
				CacheRead:  ai.Tokens(0),
				CacheWrite: ai.Tokens(0),
				Reasoning:  ai.Tokens(0),
			},
			wantFigures: CostFigures{},
			wantPres:    costPresence{input: true, output: true, cacheRead: true, cacheWrite: true, reasoning: true},
		},
		{
			name:        "mixed: input alone reported",
			usage:       ai.Usage{Input: ai.Tokens(100)},
			wantFigures: CostFigures{InputTokens: 100},
			wantPres:    costPresence{input: true},
		},
		{
			name:        "mixed: output alone reported",
			usage:       ai.Usage{Output: ai.Tokens(55)},
			wantFigures: CostFigures{OutputTokens: 55},
			wantPres:    costPresence{output: true},
		},
		{
			name:        "mixed: cache-read alone reported",
			usage:       ai.Usage{CacheRead: ai.Tokens(30)},
			wantFigures: CostFigures{CacheReadTokens: 30},
			wantPres:    costPresence{cacheRead: true},
		},
		{
			name:        "mixed: cache-write alone reported",
			usage:       ai.Usage{CacheWrite: ai.Tokens(15)},
			wantFigures: CostFigures{CacheWriteTokens: 15},
			wantPres:    costPresence{cacheWrite: true},
		},
		{
			name:        "mixed: reasoning alone reported",
			usage:       ai.Usage{Reasoning: ai.Tokens(12)},
			wantFigures: CostFigures{ReasoningTokens: 12},
			wantPres:    costPresence{reasoning: true},
		},
		{
			name: "pairwise-distinct usage — every figure present, no two magnitudes equal",
			usage: ai.Usage{
				Input:      ai.Tokens(11),
				Output:     ai.Tokens(22),
				CacheRead:  ai.Tokens(33),
				CacheWrite: ai.Tokens(44),
				Reasoning:  ai.Tokens(55),
			},
			wantFigures: CostFigures{
				InputTokens:      11,
				OutputTokens:     22,
				CacheReadTokens:  33,
				CacheWriteTokens: 44,
				ReasoningTokens:  55,
			},
			wantPres: costPresence{input: true, output: true, cacheRead: true, cacheWrite: true, reasoning: true},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			gotFigures, gotPres := costFromUsage(tt.usage)

			if gotFigures != tt.wantFigures {
				t.Errorf("costFromUsage(%+v) figures = %+v, want %+v", tt.usage, gotFigures, tt.wantFigures)
			}
			if gotPres != tt.wantPres {
				t.Errorf("costFromUsage(%+v) presence = %+v, want %+v", tt.usage, gotPres, tt.wantPres)
			}

			// Cross-check every figure's magnitude and discriminator
			// against the corresponding ai.TokenCount's own reading
			// directly — the conversion must not silently disagree
			// with Layer 1's own idiom.
			if n, ok := tt.usage.Input.Count(); uint64(n) != gotFigures.InputTokens || ok != gotPres.input {
				t.Errorf("input: costFromUsage = (%d, %v), ai.TokenCount.Count() = (%d, %v)", gotFigures.InputTokens, gotPres.input, n, ok)
			}
			if n, ok := tt.usage.Output.Count(); uint64(n) != gotFigures.OutputTokens || ok != gotPres.output {
				t.Errorf("output: costFromUsage = (%d, %v), ai.TokenCount.Count() = (%d, %v)", gotFigures.OutputTokens, gotPres.output, n, ok)
			}
			if n, ok := tt.usage.CacheRead.Count(); uint64(n) != gotFigures.CacheReadTokens || ok != gotPres.cacheRead {
				t.Errorf("cache_read: costFromUsage = (%d, %v), ai.TokenCount.Count() = (%d, %v)", gotFigures.CacheReadTokens, gotPres.cacheRead, n, ok)
			}
			if n, ok := tt.usage.CacheWrite.Count(); uint64(n) != gotFigures.CacheWriteTokens || ok != gotPres.cacheWrite {
				t.Errorf("cache_write: costFromUsage = (%d, %v), ai.TokenCount.Count() = (%d, %v)", gotFigures.CacheWriteTokens, gotPres.cacheWrite, n, ok)
			}
			if n, ok := tt.usage.Reasoning.Count(); uint64(n) != gotFigures.ReasoningTokens || ok != gotPres.reasoning {
				t.Errorf("reasoning: costFromUsage = (%d, %v), ai.TokenCount.Count() = (%d, %v)", gotFigures.ReasoningTokens, gotPres.reasoning, n, ok)
			}
		})
	}
}

// TestCostFromUsage_ZeroValuePayloadReportsEveryFigureAbsent —
// S-CST-007's zero-value-payload clause: a payload not built from a
// reported ai.Usage must report every figure absent, which is the
// coherent reading rather than a special case. costFromUsage(ai.Usage{})
// already proves this in the table above; this test names it as its
// own case so the claim is not merely incidental to the empty-usage
// row.
func TestCostFromUsage_ZeroValuePayloadReportsEveryFigureAbsent(t *testing.T) {
	t.Parallel()

	figures, pres := costFromUsage(ai.Usage{})

	if figures != (CostFigures{}) {
		t.Errorf("costFromUsage(ai.Usage{}) figures = %+v, want the zero CostFigures", figures)
	}
	if pres != (costPresence{}) {
		t.Errorf("costFromUsage(ai.Usage{}) presence = %+v, want every figure absent", pres)
	}
	if _, ok := figuresAndPresenceAllAbsent(figures, pres); !ok {
		t.Errorf("costFromUsage(ai.Usage{}) does not report every figure absent")
	}
}

// figuresAndPresenceAllAbsent reports whether every one of the five
// figures is absent — a small local helper so the assertion above
// reads as "all five", not as five repeated field checks.
func figuresAndPresenceAllAbsent(_ CostFigures, p costPresence) (costPresence, bool) {
	allAbsent := !p.input && !p.output && !p.cacheRead && !p.cacheWrite && !p.reasoning
	return p, allAbsent
}
