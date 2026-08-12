// Tests for AG-04.4 — the every-kind-constructible guard (R-AEV-009) and
// the scope fence (R-AEV-010). Mirrors
// `backend/agent/src/ai/event_registry_test.go`'s witness-table shape:
// two legs per kind — a no-argument constructor closure and a
// payload-accessor closure — cross-checked bidirectionally against
// [agent.EventKinds]. Unlike AI-14, no export_test.go witness bridge is
// needed: AG-04 registers real production kinds from birth, so this guard
// lives wholly in package agent_test over the public surface.
//
// This guard closes on bite proof (S-AEV-082, S-AEV-083), not on green: a
// guard that has only ever been green is unproven.
//
// # Recorded scope (S-AEV-084, R-AEV-009)
//
// This guard proves construction-time exhaustiveness of the kind registry
// — nothing wider. It MUST NOT be read as proof of envelope invariant 3
// (observer asynchrony), which AG-04 does not touch at all, nor of the
// full loop-level typed-error path, which is AG-11's.
package agent_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/ai"
)

// eventKindWitness is what a registered event kind must supply to prove
// its legs.
type eventKindWitness struct {
	// registeredName is this kind's registered name, stated here rather
	// than read from the registry so the registry is checked rather than
	// trusted.
	registeredName string

	// construct is leg 1: a constructor reachable from another package,
	// adapted to a shared no-argument signature. Each kind's own
	// constructor takes its own parameters; the closure supplies fixed
	// valid arguments so every witness fits the same field.
	construct func() (agent.Event, error)

	// read is leg 2: the exported accessor for this kind's payload,
	// adapted to one signature. `any` stands in for whichever concrete
	// payload type the kind's own accessor returns.
	read func(agent.Event) (any, bool)
}

// eventKindWitnesses is the guard's table, keyed by the registered
// constant — one entry per kind AG-04 (4) + AG-05 (11) register, in
// the same commit as the kind's own addition. AG-05's kinds follow
// AG-04.4's extensibility experiment pattern (R-AEV-012); extending
// this map is the only place a new kind needs to be added for the
// every-kind-constructible guard to admit it.
var eventKindWitnesses = map[agent.EventKind]eventKindWitness{
	agent.EventKindRunStart: {
		registeredName: "run_start",
		construct:      func() (agent.Event, error) { return agent.NewRunStart("run_registry_witness") },
		read:           func(e agent.Event) (any, bool) { return e.RunStart() },
	},
	agent.EventKindRunEnd: {
		registeredName: "run_end",
		construct: func() (agent.Event, error) {
			return agent.NewRunEnd("run_registry_witness", agent.RunOutcomeCompleted, nil)
		},
		read: func(e agent.Event) (any, bool) { return e.RunEnd() },
	},
	agent.EventKindTurnStart: {
		registeredName: "turn_start",
		construct: func() (agent.Event, error) {
			return agent.NewTurnStart("run_registry_witness", "turn_registry_witness")
		},
		read: func(e agent.Event) (any, bool) { return e.TurnStart() },
	},
	agent.EventKindTurnEnd: {
		registeredName: "turn_end",
		construct: func() (agent.Event, error) {
			return agent.NewTurnEnd("run_registry_witness", "turn_registry_witness", agent.TurnOutcomeFinished, nil)
		},
		read: func(e agent.Event) (any, bool) { return e.TurnEnd() },
	},
	// AG-05.1 (message family) — 6 kinds, all PlacementTurn, all
	// BracketRoleNone, all CardinalityAny, all Terminal:false
	// (R-AEV-012, R-AMT-001, R-AMT-002). See message_text.go and
	// message_reasoning.go.
	agent.EventKindMessageStartText: {
		registeredName: "message_start_text",
		construct: func() (agent.Event, error) {
			return agent.NewMessageStartText("run_registry_witness", "turn_registry_witness", witnessMessageID)
		},
		read: func(e agent.Event) (any, bool) { return e.MessageStartText() },
	},
	agent.EventKindMessageDeltaText: {
		registeredName: "message_delta_text",
		construct: func() (agent.Event, error) {
			return agent.NewMessageDeltaText("run_registry_witness", "turn_registry_witness", witnessMessageID, 0, "x")
		},
		read: func(e agent.Event) (any, bool) { return e.MessageDeltaText() },
	},
	agent.EventKindMessageEndText: {
		registeredName: "message_end_text",
		construct: func() (agent.Event, error) {
			return agent.NewMessageEndText("run_registry_witness", "turn_registry_witness", witnessMessageID)
		},
		read: func(e agent.Event) (any, bool) { return e.MessageEndText() },
	},
	agent.EventKindMessageStartReasoning: {
		registeredName: "message_start_reasoning",
		construct: func() (agent.Event, error) {
			return agent.NewMessageStartReasoning("run_registry_witness", "turn_registry_witness", witnessMessageID)
		},
		read: func(e agent.Event) (any, bool) { return e.MessageStartReasoning() },
	},
	agent.EventKindMessageDeltaReasoning: {
		registeredName: "message_delta_reasoning",
		construct: func() (agent.Event, error) {
			return agent.NewMessageDeltaReasoning("run_registry_witness", "turn_registry_witness", witnessMessageID, 0, "x")
		},
		read: func(e agent.Event) (any, bool) { return e.MessageDeltaReasoning() },
	},
	agent.EventKindMessageEndReasoning: {
		registeredName: "message_end_reasoning",
		construct: func() (agent.Event, error) {
			return agent.NewMessageEndReasoning("run_registry_witness", "turn_registry_witness", witnessMessageID)
		},
		read: func(e agent.Event) (any, bool) { return e.MessageEndReasoning() },
	},
	// AG-05.2 (tool execution family) — 5 kinds, all PlacementTurn,
	// all BracketRoleNone, all CardinalityAny, all Terminal:false.
	// See tool_event.go.
	agent.EventKindToolStart: {
		registeredName: "tool_start",
		construct: func() (agent.Event, error) {
			return agent.NewToolStart("run_registry_witness", "turn_registry_witness", "call-witness", 0, "echo", []byte("{}"))
		},
		read: func(e agent.Event) (any, bool) { return e.ToolStart() },
	},
	agent.EventKindToolProgress: {
		registeredName: "tool_progress",
		construct: func() (agent.Event, error) {
			return agent.NewToolProgress("run_registry_witness", "turn_registry_witness", "call-witness", 0, 0, []byte("{}"))
		},
		read: func(e agent.Event) (any, bool) { return e.ToolProgress() },
	},
	agent.EventKindToolEndSuccess: {
		registeredName: "tool_end_success",
		construct: func() (agent.Event, error) {
			return agent.NewToolEndSuccess("run_registry_witness", "turn_registry_witness", "call-witness", 0, []byte("ok"))
		},
		read: func(e agent.Event) (any, bool) { return e.ToolEndSuccess() },
	},
	agent.EventKindToolEndResultFailure: {
		registeredName: "tool_end_result_failure",
		construct: func() (agent.Event, error) {
			return agent.NewToolEndResultFailure("run_registry_witness", "turn_registry_witness", "call-witness", 0, []byte("nope"))
		},
		read: func(e agent.Event) (any, bool) { return e.ToolEndResultFailure() },
	},
	agent.EventKindToolEndExecutionFailure: {
		registeredName: "tool_end_execution_failure",
		construct: func() (agent.Event, error) {
			return agent.NewToolEndExecutionFailure("run_registry_witness", "turn_registry_witness", "call-witness", 0, witnessToolFailure)
		},
		read: func(e agent.Event) (any, bool) { return e.ToolEndExecutionFailure() },
	},
}

// witnessMessageID and witnessToolFailure are shared fixtures used by
// the AG-05 witness table rows so each constructor closure is a one-
// liner. They live in event_registry_test.go's test-only scope; they
// are not exported and are not part of the package's public surface.
var witnessMessageID ai.MessageID
var witnessToolFailure *agent.Failure

func init() {
	// Mint a non-zero MessageID at process init by constructing one
	// through the public Layer 1 surface (NewMessage) and reading its
	// minted identity. The message itself is discarded; we keep only
	// the identity. This is the documented way to obtain a non-zero
	// MessageID from outside the ai package.
	part, err := ai.NewText("witness-fixture")
	if err != nil {
		panic("event_registry_test.go: ai.NewText failed: " + err.Error())
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		panic("event_registry_test.go: ai.NewMessage failed: " + err.Error())
	}
	witnessMessageID = msg.ID()

	inner, err := ai.MidStreamFailure(ai.FailureReport{Category: ai.FailureCategoryUnavailable}, false)
	if err != nil {
		panic("event_registry_test.go: ai.MidStreamFailure failed: " + err.Error())
	}
	f, err := agent.NewFailure(inner)
	if err != nil {
		panic("event_registry_test.go: agent.NewFailure failed: " + err.Error())
	}
	witnessToolFailure = f
}

// TestEventKindRegistration_EveryRegisteredKind_HasConstructorAndAccessor
// is the exhaustiveness guard (R-AEV-009). It fails when a kind
// agent.EventKinds() carries has no witness, when a witness kind is not
// carried by agent.EventKinds(), or when a witness leg does not do what
// it promises — the bidirectional cross-check that closes only on a
// recorded bite (S-AEV-082, S-AEV-083), not on green alone.
func TestEventKindRegistration_EveryRegisteredKind_HasConstructorAndAccessor(t *testing.T) {
	t.Parallel()

	kinds := agent.EventKinds()
	if len(kinds) == 0 {
		t.Fatal("agent.EventKinds() is empty; the exhaustiveness assertion would pass vacuously")
	}

	t.Run("every registered kind has a witness, and every witness a kind", func(t *testing.T) {
		t.Parallel()

		for _, k := range kinds {
			if _, ok := eventKindWitnesses[k]; !ok {
				t.Errorf("%v is carried by agent.EventKinds() but has no entry in eventKindWitnesses", k)
			}
		}
		for k := range eventKindWitnesses {
			if !containsEventKind(kinds, k) {
				t.Errorf("eventKindWitnesses carries a witness for %v, which agent.EventKinds() does not enumerate", k)
			}
		}
	})

	// S-AEV-080, S-AEV-081: at least one instance constructed per kind,
	// each through the public surface, each passed through CheckEmit
	// (R-AEV-001's validation gate) once stamped.
	var constructedCount int
	for k, witness := range eventKindWitnesses {
		t.Run(witness.registeredName, func(t *testing.T) {
			t.Parallel()

			t.Run("leg 1 — a constructor", func(t *testing.T) {
				t.Parallel()

				event, err := witness.construct()
				if err != nil {
					t.Fatalf("the witness for %v returned %v, want no failure", witness.registeredName, err)
				}
				if event.Kind() != k {
					t.Fatalf("the witness for %v built an event of kind %v, want %v", witness.registeredName, event.Kind(), k)
				}

				var lane agent.LaneStamper
				stamped := lane.Stamp(event)
				if err := agent.CheckEmit(stamped); err != nil {
					t.Errorf("agent.CheckEmit(stamped %v) = %v, want nil — every witness-constructed instance must pass the R-AEV-001 validation gate", witness.registeredName, err)
				}
			})

			t.Run("leg 2 — a payload accessor", func(t *testing.T) {
				t.Parallel()

				event, err := witness.construct()
				if err != nil {
					t.Fatalf("the witness for %v returned %v, want no failure", witness.registeredName, err)
				}
				if _, ok := witness.read(event); !ok {
					t.Errorf("the accessor for %v reported no payload on an event of its own kind", witness.registeredName)
				}

				// The accessor must also answer no for the zero Event: the
				// second result distinguishes "not this kind" from "this
				// kind, empty".
				if _, ok := witness.read(agent.Event{}); ok {
					t.Errorf("the accessor for %v reported a payload on the zero Event", witness.registeredName)
				}
			})
		})
		constructedCount++
	}
	if constructedCount == 0 {
		t.Fatal("zero witnesses were exercised; S-AEV-080's positive report would be vacuous")
	}
	t.Logf("every-kind-constructible guard constructed at least one instance of %d registered kind(s)", constructedCount)
}

func containsEventKind(haystack []agent.EventKind, want agent.EventKind) bool {
	for _, k := range haystack {
		if k == want {
			return true
		}
	}
	return false
}

// S-AEV-090 — the registered kind set is enumerated as exactly the
// run/turn/message/tool bracket kinds AG-04 + AG-05 register, and
// contains no permission, cost, delegation or compaction kind under
// any name (R-AEV-010's scope fence; retightened from "exactly 4" to
// "exactly 15" by AG-05 across this PR's first two commits; the
// final retightening to "exactly 15" lands in AG-05.3).
func TestEventKinds_ScopeFence_ExactlyRunAndTurnFamilies(t *testing.T) {
	t.Parallel()

	got := agent.EventKinds()
	want := []agent.EventKind{
		agent.EventKindRunStart, agent.EventKindRunEnd,
		agent.EventKindTurnStart, agent.EventKindTurnEnd,
		// AG-05.1 (message family) — 6 message kinds (R-AMT-001, R-AMT-002)
		agent.EventKindMessageStartText, agent.EventKindMessageDeltaText, agent.EventKindMessageEndText,
		agent.EventKindMessageStartReasoning, agent.EventKindMessageDeltaReasoning, agent.EventKindMessageEndReasoning,
		// AG-05.2 (tool execution family) — 5 tool kinds (R-AMT-005, R-AMT-006, R-AMT-007)
		agent.EventKindToolStart, agent.EventKindToolProgress,
		agent.EventKindToolEndSuccess, agent.EventKindToolEndResultFailure, agent.EventKindToolEndExecutionFailure,
	}
	if len(got) != len(want) {
		t.Fatalf("agent.EventKinds() = %v (%d kinds), want %v (%d kinds) — AG-05 retightens the scope fence from \"exactly 4\" to \"exactly 15\" (4 AG-04 + 6 AG-05.1 + 5 AG-05.2)",
			got, len(got), want, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("agent.EventKinds()[%d] = %v, want %v", i, got[i], want[i])
		}
	}

	// AG-05 retightens this list: AG-04's "no message/tool" half is
	// now legitimate (AG-05's own register); the AG-06 "no permission/
	// cost/delegation/compaction" half is unchanged. The forbidden
	// names check now only catches AG-06's families.
	forbiddenNames := []string{"permission", "cost", "delegation", "compaction"}
	for _, k := range got {
		name := k.String()
		for _, forbidden := range forbiddenNames {
			if containsFold(name, forbidden) {
				t.Errorf("registered kind %q names a %s family; R-AEV-010 forbids AG-04 and AG-05 from registering AG-06's families under any name", name, forbidden)
			}
		}
	}
}

// S-AEV-091 — the "adding a kind" procedure comment states the ordered
// steps a later milestone follows, and states that following them
// requires no edit to the validator's rule engine.
func TestEventDescriptorDoc_StatesTheSixStepProcedure(t *testing.T) {
	t.Parallel()

	doc := packageFileDoc(t, "event_descriptor.go")
	wantPhrases := []string{
		"declare the constant",
		"add an unexported payload type",
		"add the exported constructor",
		"add the exported accessor",
		"add the registry row",
		"add the name to",
		"no edit to the validator's rule engine",
	}
	for _, phrase := range wantPhrases {
		if !containsFold(doc, phrase) {
			t.Errorf("event_descriptor.go's documentation does not state step %q of the six-step adding-a-kind procedure", phrase)
		}
	}
}

// S-AEV-084 — the guard's recorded scope note states that it proves
// construction-time exhaustiveness only, and closes neither envelope
// invariant 3 nor the loop-level typed-error path.
func TestEventRegistryDoc_StatesTheGuardsRecordedScope(t *testing.T) {
	t.Parallel()

	doc := packageFileDoc(t, "event_registry_test.go")
	if !containsFold(doc, "construction-time exhaustiveness") {
		t.Errorf("event_registry_test.go's documentation does not state the guard proves construction-time exhaustiveness only:\n%s", doc)
	}
	if !containsFold(doc, "invariant 3") {
		t.Errorf("event_registry_test.go's documentation does not state the guard closes no part of envelope invariant 3:\n%s", doc)
	}
	if !containsFold(doc, "loop-level typed-error path") {
		t.Errorf("event_registry_test.go's documentation does not state the guard closes no part of the loop-level typed-error path:\n%s", doc)
	}
}

