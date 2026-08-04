// AI-23.2 extension — R-CNF-024: a Script's ordered steps are
// introspectable from OUTSIDE package agenttest, read-only. External
// (package agenttest_test), matching provider_test.go's own precedent for
// proving a surface's readability from another package.

package agenttest_test

import (
	"reflect"
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustIntrospectableScript builds a small emit-only script this file's
// S-CLA-033 test both introspects and drains, so the two can be compared
// event for event.
func mustIntrospectableScript(t *testing.T) agenttest.Script {
	t.Helper()

	responseStart, err := ai.NewResponseStart("resp-introspect", "model-introspect")
	if err != nil {
		t.Fatalf("ai.NewResponseStart returned %v, want no failure", err)
	}
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	delta, err := ai.NewTextDelta(1, "introspected")
	if err != nil {
		t.Fatalf("ai.NewTextDelta returned %v, want no failure", err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		t.Fatalf("ai.NewCompletion returned %v, want no failure", err)
	}

	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(responseStart),
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// TestStepIntrospection_EmitOnlyScript_MatchesWhatTheFakeProviderDrains
// proves S-CLA-033: a script authored in package agenttest and read from
// this separate external test package reconstructs, event for event, the
// same ordered list of ai.Event values a fake provider built from that same
// script emits when drained — kind-for-kind and payload-for-payload via
// typed accessors. Sequence is deliberately excluded from the comparison:
// introspected steps are unstamped (seq 0), and stamping is the producer's
// job, never the script's.
func TestStepIntrospection_EmitOnlyScript_MatchesWhatTheFakeProviderDrains(t *testing.T) {
	script := mustIntrospectableScript(t)

	var introspected []ai.Event
	for i, step := range script.Steps {
		if step.IsHold() {
			t.Fatalf("step %d of an emit-only script reports IsHold() = true, want every step to emit", i)
		}
		ev, ok := step.Event()
		if !ok {
			t.Fatalf("step %d of an emit-only script's Event() reports ok = false, want the emitted event", i)
		}
		introspected = append(introspected, ev)
	}

	provider := agenttest.NewProvider(script)
	ch, err := provider.Stream(t.Context(), mustSimpleRequest(t))
	if err != nil {
		t.Fatalf("Stream returned %v, want no failure", err)
	}
	drained := agenttest.DrainAndRecord(t, ch, agenttest.DefaultDrainTimeout).Events()

	if len(introspected) != len(drained) {
		t.Fatalf("introspected %d event(s), drained %d, want equal lengths (S-CLA-033)", len(introspected), len(drained))
	}
	for i := range introspected {
		if introspected[i].Kind() != drained[i].Kind() {
			t.Errorf("event %d: introspected kind = %v, drained kind = %v, want equal", i, introspected[i].Kind(), drained[i].Kind())
		}
	}

	// Payload-for-payload via typed accessors, for the two payload-bearing
	// kinds this script carries.
	introStart, ok := introspected[0].ResponseStart()
	if !ok {
		t.Fatal("introspected event 0 carries no ResponseStart payload")
	}
	drainedStart, ok := drained[0].ResponseStart()
	if !ok {
		t.Fatal("drained event 0 carries no ResponseStart payload")
	}
	if introStart.ResponseID() != drainedStart.ResponseID() || introStart.ServedModel() != drainedStart.ServedModel() {
		t.Errorf("introspected ResponseStart = (%q, %q), drained = (%q, %q), want equal", introStart.ResponseID(), introStart.ServedModel(), drainedStart.ResponseID(), drainedStart.ServedModel())
	}

	introDelta, ok := introspected[2].TextDelta()
	if !ok {
		t.Fatal("introspected event 2 carries no TextDelta payload")
	}
	drainedDelta, ok := drained[2].TextDelta()
	if !ok {
		t.Fatal("drained event 2 carries no TextDelta payload")
	}
	if introDelta.Delta() != drainedDelta.Delta() {
		t.Errorf("introspected delta = %q, drained delta = %q, want equal", introDelta.Delta(), drainedDelta.Delta())
	}

	// Sequence is deliberately excluded from event-for-event equality:
	// introspected steps are unstamped, stamping is the producer's job.
	if introspected[0].Sequence() != 0 {
		t.Errorf("introspected event 0 Sequence() = %d, want 0 (unstamped)", introspected[0].Sequence())
	}
	if drained[0].Sequence() == 0 {
		t.Error("drained event 0 Sequence() = 0, want a real stamped sequence (the producer stamps on drain)")
	}
}

// TestStepIntrospection_MixedEmitAndHoldScript_ReportsWhichIsWhichWithoutDraining
// proves S-CLA-034: a script mixing emit steps with hold/gate steps
// reports, per step, which of the two it is; the emit steps' events are
// recoverable while the hold/gate steps are identified as carrying no
// event. Introspection only, never draining — a Hold step would block the
// producer forever with nothing left in this test to release it.
func TestStepIntrospection_MixedEmitAndHoldScript_ReportsWhichIsWhichWithoutDraining(t *testing.T) {
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockStart returned %v, want no failure", err)
	}
	gate := agenttest.NewGate()
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		t.Fatalf("ai.NewTextBlockEnd returned %v, want no failure", err)
	}

	script := agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Hold(gate),
		agenttest.Emit(end),
	}}

	wantHold := []bool{false, true, false}
	for i, step := range script.Steps {
		if got := step.IsHold(); got != wantHold[i] {
			t.Errorf("step %d IsHold() = %v, want %v (S-CLA-034)", i, got, wantHold[i])
		}
	}

	ev, ok := script.Steps[0].Event()
	if !ok || ev.Kind() != ai.EventKindTextBlockStart {
		t.Errorf("step 0 Event() = (%v, ok=%v), want (a text block start event, true)", ev, ok)
	}
	if _, ok := script.Steps[1].Event(); ok {
		t.Error("the Hold step's Event() reports ok = true, want false — a hold/gate step carries no event")
	}
	ev, ok = script.Steps[2].Event()
	if !ok || ev.Kind() != ai.EventKindTextBlockEnd {
		t.Errorf("step 2 Event() = (%v, ok=%v), want (a text block end event, true)", ev, ok)
	}
}

// TestStepIntrospection_ExportedMethodSet_ExposesOnlyEventAndIsHold proves
// S-CLA-035's structural half: agenttest.Step's exported method set is
// exactly {Event, IsHold} — no setter, no Gate() accessor (which would
// expose a mutation door), no rewriting constructor reachable through the
// value a caller was handed.
func TestStepIntrospection_ExportedMethodSet_ExposesOnlyEventAndIsHold(t *testing.T) {
	typ := reflect.TypeOf(agenttest.Step{})
	want := map[string]bool{"Event": true, "IsHold": true}
	if got := typ.NumMethod(); got != len(want) {
		names := make([]string, got)
		for i := range got {
			names[i] = typ.Method(i).Name
		}
		t.Fatalf("agenttest.Step exports %d method(s) %v, want exactly %d: %v (S-CLA-035)", got, names, len(want), want)
	}
	for i := range typ.NumMethod() {
		name := typ.Method(i).Name
		if !want[name] {
			t.Errorf("agenttest.Step exports unexpected method %q, want only Event/IsHold (S-CLA-035, no mutation door)", name)
		}
	}
}
