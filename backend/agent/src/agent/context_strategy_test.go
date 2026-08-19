// AG-17.1 — strict-TDD tests for the context strategy seam
// (R-CTX-001..005, S-CTX-001..010, bites S-CTX-003/008).
//
// Every scenario lives in package agent_test (NFR-CTX-001): the
// external-test posture proves every behavioral claim from outside the
// package, with no reach into unexported surface.
package agent_test

import (
	"context"
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
)

// TestContextVerdict_HasNoFields — S-CTX-005, charter AG-17.1 scenario
// 2, the type half. The shipped verdict type reports zero fields when
// inspected reflectively, and carries no exported method and no
// constructor function that could produce a value distinguishable
// from its zero value: the only value NoOpContextStrategy.Resolve can
// ever return IS the zero value, because no other value exists to
// return (R-CTX-003).
func TestContextVerdict_HasNoFields(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(agent.ContextVerdict{})
	if typ.NumField() != 0 {
		t.Fatalf("agent.ContextVerdict has %d field(s), want 0 (R-CTX-003: no field can request compaction)", typ.NumField())
	}
	if typ.NumMethod() != 0 {
		t.Errorf("agent.ContextVerdict (value type) declares %d exported method(s), want 0 -- a method could still manufacture a distinguishable value", typ.NumMethod())
	}
	ptrTyp := reflect.TypeOf(&agent.ContextVerdict{})
	if ptrTyp.NumMethod() != 0 {
		t.Errorf("*agent.ContextVerdict declares %d exported method(s), want 0", ptrTyp.NumMethod())
	}

	var zero agent.ContextVerdict
	verdict := agent.NoOpContextStrategy{}.Resolve(context.Background(), agent.ContextPrompt{})
	if verdict != zero {
		t.Errorf("NoOpContextStrategy.Resolve() = %#v, want the zero ContextVerdict", verdict)
	}
}

// TestContextBudget_AbsentVsStatedZero — S-CTX-009. The zero
// ContextBudget reads as absence through the two-result accessor;
// ContextBudgetOf(0) reads a limit of zero together with presence;
// the two values are NOT equal -- absence and a stated nought are
// distinguishable, exactly as ai.Tokens(0) is distinguishable from
// ai.TokenCount{} (ai/usage.go:34-47).
func TestContextBudget_AbsentVsStatedZero(t *testing.T) {
	t.Parallel()

	var zero agent.ContextBudget
	if limit, present := zero.Limit(); present || limit != 0 {
		t.Errorf("zero ContextBudget.Limit() = (%d, %v), want (0, false) -- the zero value is absence, not a budget of zero tokens", limit, present)
	}

	statedZero := agent.ContextBudgetOf(0)
	limit, present := statedZero.Limit()
	if !present || limit != 0 {
		t.Errorf("ContextBudgetOf(0).Limit() = (%d, %v), want (0, true) -- a stated zero is present", limit, present)
	}

	if zero == statedZero {
		t.Error("the zero ContextBudget equals ContextBudgetOf(0); absence and a stated nought MUST be distinguishable values")
	}
}

// TestContextBudget_NegativeIsAbsentTotal — S-CTX-010, first half.
// ContextBudgetOf is total: a negative input yields the absent zero
// value, with no error and no panic. Triangulated across several
// negative inputs, including the int64 minimum, so a Fake-It
// implementation that special-cases -1 cannot pass.
func TestContextBudget_NegativeIsAbsentTotal(t *testing.T) {
	t.Parallel()

	var zero agent.ContextBudget

	for _, n := range []int64{-1, -2, -1000, -9223372036854775808} {
		got := agent.ContextBudgetOf(n)
		if limit, present := got.Limit(); present || limit != 0 {
			t.Errorf("ContextBudgetOf(%d).Limit() = (%d, %v), want (0, false) -- a negative limit is not a budget", n, limit, present)
		}
		if got != zero {
			t.Errorf("ContextBudgetOf(%d) = %#v, want the zero ContextBudget (%#v)", n, got, zero)
		}
	}
}

// TestTokenAccounting_UnreadableWithoutSource — S-CTX-014, the
// distinguishability half. The accounting type exposes no exported
// field and exactly ONE exported method -- Tokens() (int64,
// TokenSource) -- so the figure is physically unreadable without its
// provenance in the same call. The NumOut()==2 assertion is the
// load-bearing check bite S-CTX-015 defeats: a bare single-result
// Tokens() int64 declared BESIDE this one would push NumMethod to 2,
// and removing the two-result accessor entirely is a compile failure
// at every call site below -- both are valid RED evidence for the
// bite.
func TestTokenAccounting_UnreadableWithoutSource(t *testing.T) {
	t.Parallel()

	typ := reflect.TypeOf(agent.TokenAccounting{})
	for i := range typ.NumField() {
		if f := typ.Field(i); f.IsExported() {
			t.Fatalf("agent.TokenAccounting exports field %q, want zero exported fields (R-CTX-008)", f.Name)
		}
	}
	if typ.NumMethod() != 1 {
		t.Fatalf("agent.TokenAccounting declares %d exported method(s), want exactly 1 (Tokens() (int64, TokenSource)) -- a second accessor would let a consumer bypass the provenance", typ.NumMethod())
	}
	method := typ.Method(0)
	if method.Name != "Tokens" {
		t.Fatalf("agent.TokenAccounting's one method is named %q, want %q", method.Name, "Tokens")
	}
	if got := method.Type.NumOut(); got != 2 {
		t.Fatalf("Tokens() declares %d result(s), want exactly 2 (figure, source) -- a single-result accessor would let a consumer read the figure without meeting its provenance", got)
	}

	var zero agent.TokenAccounting
	tokens, source := zero.Tokens()
	if tokens != 0 || source != agent.TokenSourceUnavailable {
		t.Errorf("zero TokenAccounting.Tokens() = (%d, %v), want (0, TokenSourceUnavailable) -- the zero value reads as no figure, never as 0 tokens", tokens, source)
	}

	rendered := map[string]bool{}
	for _, s := range []agent.TokenSource{agent.TokenSourceUnavailable, agent.TokenSourceReported, agent.TokenSourceEstimated} {
		rendered[s.String()] = true
	}
	if len(rendered) != 3 {
		t.Errorf("the three TokenSource values render to %d distinct string(s), want 3", len(rendered))
	}
}
