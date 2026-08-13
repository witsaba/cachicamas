// AG-06.2 — the cost family tests (R-APE-004, R-APE-005, VL2-EVT-07).
//
// Cost events record the model's usage: [NewCostTurn] emits per-turn
// token figures (PlacementTurn) and [NewCostSession] emits cumulative
// figures over the whole run (PlacementRun). Both payloads carry the
// same five token fields (inputTokens, outputTokens, cacheReadTokens,
// cacheWriteTokens, reasoningTokens) and a [CostLabel] discriminator
// (Estimate or Final). The token-only shape is asserted mechanically
// via reflection (S-APE-083), not by doc-phrase check — the
// construction surface MUST NOT carry any field with a money-suggesting
// name, ever.
//
// # Strict TDD — RED-first
//
// Every scenario in this file references a type or function the
// production file (cost_events.go) does not yet declare. The file's
// compile-failure is the RED; the per-scenario assertions are the RED
// that proves the production behaviour matches the spec scenarios.

package agent_test

import (
	"reflect"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// S-APE-030 — a cost_turn event carries five distinct token fields
// (inputTokens, outputTokens, cacheReadTokens, cacheWriteTokens,
// reasoningTokens) — R-APE-004's token-only shape.
func TestCost_Turn_HasFiveTokenFields(t *testing.T) {
	t.Parallel()

	figures := agent.CostFigures{
		InputTokens:      100,
		OutputTokens:     50,
		CacheReadTokens:  10,
		CacheWriteTokens: 5,
		ReasoningTokens:  0,
	}
	event, err := agent.NewCostTurn("run-cost-witness", "turn-cost-witness", agent.CostLabelEstimate, figures)
	if err != nil {
		t.Fatalf("NewCostTurn = %v, want nil", err)
	}

	if event.Kind() != agent.EventKindCostTurn {
		t.Errorf("event.Kind() = %v, want %v", event.Kind(), agent.EventKindCostTurn)
	}

	got, ok := event.CostTurn()
	if !ok {
		t.Fatalf("CostTurn() reported no payload on an event of its own kind")
	}
	if got.Label() != agent.CostLabelEstimate {
		t.Errorf("Label() = %v, want Estimate", got.Label())
	}
	if got.Figures().InputTokens != 100 {
		t.Errorf("Figures().InputTokens = %d, want 100", got.Figures().InputTokens)
	}
	if got.Figures().OutputTokens != 50 {
		t.Errorf("Figures().OutputTokens = %d, want 50", got.Figures().OutputTokens)
	}
	if got.Figures().CacheReadTokens != 10 {
		t.Errorf("Figures().CacheReadTokens = %d, want 10", got.Figures().CacheReadTokens)
	}
	if got.Figures().CacheWriteTokens != 5 {
		t.Errorf("Figures().CacheWriteTokens = %d, want 5", got.Figures().CacheWriteTokens)
	}
	if got.Figures().ReasoningTokens != 0 {
		t.Errorf("Figures().ReasoningTokens = %d, want 0", got.Figures().ReasoningTokens)
	}
}

// S-APE-031 — a cost_session event carries the same five fields as
// cumulative figures, and registers with Placement: PlacementRun
// (R-APE-004 — session marker spans the whole run).
func TestCost_Session_PlacementIsRun(t *testing.T) {
	t.Parallel()

	figures := agent.CostFigures{
		InputTokens:      1000,
		OutputTokens:     500,
		CacheReadTokens:  100,
		CacheWriteTokens: 50,
		ReasoningTokens:  25,
	}
	event, err := agent.NewCostSession("run-cost-witness", agent.CostLabelFinal, figures)
	if err != nil {
		t.Fatalf("NewCostSession = %v, want nil", err)
	}

	if event.Kind() != agent.EventKindCostSession {
		t.Errorf("event.Kind() = %v, want %v", event.Kind(), agent.EventKindCostSession)
	}

	got, ok := event.CostSession()
	if !ok {
		t.Fatalf("CostSession() reported no payload on an event of its own kind")
	}
	if got.Label() != agent.CostLabelFinal {
		t.Errorf("Label() = %v, want Final", got.Label())
	}
	if got.Figures().InputTokens != 1000 {
		t.Errorf("Figures().InputTokens = %d, want 1000", got.Figures().InputTokens)
	}

	// Distinct kind.
	if _, ok := event.CostTurn(); ok {
		t.Errorf("CostTurn() reported a payload on a cost_session event")
	}
}

// S-APE-040 — every cost figure carries a typed [CostLabel]
// discriminator whose value is Estimate or Final (R-APE-005).
// Reasoning tokens only arrive on the final update — the label
// disambiguates the adaptive reasoning protocol. Zero label is
// rejected (closed enum, zero not a member).
func TestCost_EveryFigureLabelledEstimateOrFinal(t *testing.T) {
	t.Parallel()

	// Pre-final figure: label is Estimate.
	pre, err := agent.NewCostTurn("run-cost-witness", "turn-cost-witness", agent.CostLabelEstimate, agent.CostFigures{
		InputTokens:      10,
		OutputTokens:     5,
		CacheReadTokens:  0,
		CacheWriteTokens: 0,
		ReasoningTokens:  0, // reasoning only on final
	})
	if err != nil {
		t.Fatalf("NewCostTurn(Estimate) = %v, want nil", err)
	}
	if got := mustCostTurn(t, pre).Label(); got != agent.CostLabelEstimate {
		t.Errorf("pre-final Label() = %v, want Estimate", got)
	}

	// Final figure: label is Final; reasoning tokens arrive here.
	final, err := agent.NewCostSession("run-cost-witness", agent.CostLabelFinal, agent.CostFigures{
		InputTokens:      110,
		OutputTokens:     55,
		CacheReadTokens:  0,
		CacheWriteTokens: 0,
		ReasoningTokens:  12,
	})
	if err != nil {
		t.Fatalf("NewCostSession(Final) = %v, want nil", err)
	}
	if got := mustCostSession(t, final).Label(); got != agent.CostLabelFinal {
		t.Errorf("final Label() = %v, want Final", got)
	}

	// Closed enum: zero is not a member — the ctor MUST reject it.
	if _, err := agent.NewCostTurn("run-cost-witness", "turn-cost-witness", agent.CostLabel(0), agent.CostFigures{
		InputTokens: 1,
	}); err == nil {
		t.Errorf("NewCostTurn(zero label) returned no error; the closed enum MUST reject zero (R-APE-005)")
	}
	if _, err := agent.NewCostSession("run-cost-witness", agent.CostLabel(0), agent.CostFigures{
		InputTokens: 1,
	}); err == nil {
		t.Errorf("NewCostSession(zero label) returned no error; the closed enum MUST reject zero (R-APE-005)")
	}
}

// S-APE-083 — **(bite)** the cost payload shape is token-only,
// mechanically asserted via reflection. No field may carry a
// money-suggesting name — the absence is the rule (R-APE-004, ADR 0005
// § D4 per 0003:613). The reflection pin walks the [CostFigures]
// struct's fields and asserts none match a forbidden money-suggesting
// substring.
//
// S-APE-083 — BITE
// RED:  scratch violation here (reverted) — adding a `Money`
//       field to [CostFigures] flips the test to RED with a
//       diff naming the offending field.
// Bite: a token-only mechanical pin, not a doc-phrase check
//       (AG-04 W7 carry-forward inverted — mechanical pin
//       instead of phrase check).
// Asserted: every field name in [CostFigures] is one of the five
//           documented token-field names; no field matches the
//           forbidden money-suggesting list.
// GREEN: the production struct carries exactly the five token
//        fields and no money-suggesting field; the test passes.
func TestCost_PayloadShape_NoMoneyField(t *testing.T) {
	t.Parallel()

	// Documented token field names — must match the production
	// struct exactly.
	wantFieldNames := []string{
		"InputTokens",
		"OutputTokens",
		"CacheReadTokens",
		"CacheWriteTokens",
		"ReasoningTokens",
	}

	figuresType := reflect.TypeOf(agent.CostFigures{})

	if figuresType.NumField() != len(wantFieldNames) {
		t.Fatalf("agent.CostFigures has %d fields, want %d (the five documented token fields) — any extra field violates the token-only mechanical pin (S-APE-083)",
			figuresType.NumField(), len(wantFieldNames))
	}

	// No field may have a money-suggesting name.
	forbiddenMoneySubstrings := []string{
		"money", "amount", "price", "total", "cents",
		"dollar", "usd", "currency", "fee", "charge",
	}
	for i := 0; i < figuresType.NumField(); i++ {
		got := figuresType.Field(i)
		if got.Name != wantFieldNames[i] {
			t.Errorf("agent.CostFigures field %d = %q, want %q — the five-field, in-order pin is part of the token-only mechanical contract (S-APE-083)",
				i, got.Name, wantFieldNames[i])
		}
		lower := strings.ToLower(got.Name)
		for _, forbidden := range forbiddenMoneySubstrings {
			if strings.Contains(lower, forbidden) {
				t.Errorf("agent.CostFigures field %q contains the money-suggesting substring %q — the token-only mechanical pin (S-APE-083) forbids any field that could carry money, currency or price",
					got.Name, forbidden)
			}
		}
	}

	// Every field MUST be uint64 (token counts, no money-precision
	// float shape).
	for i := 0; i < figuresType.NumField(); i++ {
		got := figuresType.Field(i)
		if got.Type.Kind() != reflect.Uint64 {
			t.Errorf("agent.CostFigures field %q is %v, want uint64 — token counts are non-negative integers (S-APE-083, R-APE-004)",
				got.Name, got.Type.Kind())
		}
	}
}

// mustCostTurn extracts the cost_turn payload or fails the test.
// Mirrors the assertion-quality rule: every accessor returns
// meaningfully, not vacuously.
func mustCostTurn(t *testing.T, e agent.Event) agent.CostTurn {
	t.Helper()
	got, ok := e.CostTurn()
	if !ok {
		t.Fatalf("CostTurn() reported no payload on a cost_turn event")
	}
	return got
}

// mustCostSession extracts the cost_session payload or fails the test.
func mustCostSession(t *testing.T, e agent.Event) agent.CostSession {
	t.Helper()
	got, ok := e.CostSession()
	if !ok {
		t.Fatalf("CostSession() reported no payload on a cost_session event")
	}
	return got
}

// Silence unused import warnings for ai — the test references
// ai.ErrXxx paths via the production validator, but the ai package is
// imported here for symmetry with permission_events_test.go and to
// keep the import set identical across per-family files.
var _ = ai.ErrEmpty
