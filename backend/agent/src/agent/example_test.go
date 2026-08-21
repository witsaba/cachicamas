// AG-23.2 — runnable examples (R-L3H-005). Four Example* functions, one
// per subject the frozen v1 surface names: building a harness, driving
// a run, consuming events, and handling a permission suspension. Each
// carries a mandatory `// Output:` block, so `cd backend/agent && make
// test` compiles AND RUNS every one of them — a drift between an
// example and the surface it demonstrates fails the build rather than
// rotting silently.
//
// No example prints a minted identity (RunID/TurnID): those depend on
// which tests ran earlier in the same test process, so printing one
// would make an example's own output order-dependent for a reason that
// has nothing to do with the surface it documents. Every example prints
// kind names, finish reasons or permission outcomes only.
package agent_test

import (
	"context"
	"fmt"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// exampleUserMessage builds a minimal user message. Panics on
// construction failure: every input here is a well-formed literal, so a
// failure would mean the ai package's own constructor rules changed
// underneath this example, which is exactly the "fails to compile [or
// panics]" drift R-L3H-005 wants surfaced rather than hidden.
func exampleUserMessage(text string) ai.Message {
	part, err := ai.NewText(text)
	if err != nil {
		panic(err)
	}
	msg, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		panic(err)
	}
	return msg
}

// exampleTextScript scripts one provider turn returning a short
// completed text response.
func exampleTextScript(text string) agenttest.Script {
	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		panic(err)
	}
	delta, err := ai.NewTextDelta(1, text)
	if err != nil {
		panic(err)
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		panic(err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		panic(err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// exampleToolCallScript scripts one provider turn requesting a tool call.
func exampleToolCallScript(callID, toolName string, args []byte) agenttest.Script {
	start, err := ai.NewToolCallStart(1, callID, toolName)
	if err != nil {
		panic(err)
	}
	delta, err := ai.NewToolCallDelta(1, args)
	if err != nil {
		panic(err)
	}
	end, err := ai.NewToolCallEnd(1, args)
	if err != nil {
		panic(err)
	}
	completion, err := ai.NewCompletion(ai.FinishReasonToolCalls, ai.Usage{})
	if err != nil {
		panic(err)
	}
	return agenttest.Script{Steps: []agenttest.Step{
		agenttest.Emit(start),
		agenttest.Emit(delta),
		agenttest.Emit(end),
		agenttest.Emit(completion),
	}}
}

// ExampleHarness shows building a harness: injected fakes (a scripted
// provider) assembled into a Harness literal through public fields
// only, then driven once to confirm the literal is genuinely runnable.
func ExampleHarness() {
	provider := agenttest.NewProvider(exampleTextScript("hello from the example"))
	h := agent.Harness{
		Provider: provider,
		System:   "a minimal system prompt",
	}

	sink := make(chan *agent.Event, 64)
	if _, _, err := h.Run(context.Background(), exampleUserMessage("hi"), sink); err != nil {
		panic(err)
	}
	for ev := range sink {
		_ = ev // draining to close; this example's own output does not depend on any event's content.
	}

	fmt.Println("harness built with system prompt:", h.System)
	// Output:
	// harness built with system prompt: a minimal system prompt
}

// ExampleHarness_Run shows driving a run to its terminal decision and
// reading back the finish reason Run returns.
func ExampleHarness_Run() {
	provider := agenttest.NewProvider(exampleTextScript("hello from the example"))
	h := agent.Harness{
		Provider: provider,
		System:   "a minimal system prompt",
	}

	sink := make(chan *agent.Event, 64)
	_, finish, err := h.Run(context.Background(), exampleUserMessage("hi"), sink)
	for ev := range sink {
		_ = ev // draining to close; this example's own output does not depend on any event's content.
	}
	if err != nil {
		panic(err)
	}

	fmt.Println("finish reason:", finish)
	// Output:
	// finish reason: stop
}

// ExampleHarness_events shows consuming the event stream: the consumer
// switches on each event's kind (here, simply printing its name) as the
// run drives it, one event at a time.
func ExampleHarness_events() {
	provider := agenttest.NewProvider(exampleTextScript("hello from the example"))
	h := agent.Harness{
		Provider: provider,
		System:   "a minimal system prompt",
	}

	sink := make(chan *agent.Event)
	go func() {
		_, _, _ = h.Run(context.Background(), exampleUserMessage("hi"), sink)
	}()

	for ev := range sink {
		switch ev.Kind() {
		case agent.EventKindRunStart, agent.EventKindRunEnd,
			agent.EventKindTurnStart, agent.EventKindTurnEnd,
			agent.EventKindMessageStartText, agent.EventKindMessageDeltaText, agent.EventKindMessageEndText:
			fmt.Println(ev.Kind())
		}
	}
	// Output:
	// run_start
	// turn_start
	// message_start_text
	// message_delta_text
	// message_end_text
	// turn_end
	// run_end
}

// exampleDeferOncePolicy is a minimal scripted agent.PermissionPolicy,
// local to this example so a reader sees the whole mechanism in one
// place: its first Resolve call defers (parking the call); every later
// call — the re-entry after Scheduler.WakeParked releases it — allows.
type exampleDeferOncePolicy struct {
	asked int
}

func (p *exampleDeferOncePolicy) Resolve(context.Context, ai.ToolCall) agent.PermissionVerdict {
	p.asked++
	if p.asked == 1 {
		return agent.PermissionVerdict{Outcome: agent.PermissionDefer}
	}
	return agent.PermissionVerdict{Outcome: agent.PermissionOutcomeAllowOnce}
}

func (p *exampleDeferOncePolicy) Remember(context.Context, string, agent.PermissionOutcome) bool {
	return false
}

// ExampleHarness_suspension shows handling a permission suspension: the
// scripted policy defers the tool call's first decision; this consumer
// wakes the parked call the moment it observes the suspension on the
// stream, then reads the resolution back off the same stream.
func ExampleHarness_suspension() {
	tool := EchoScriptedTool("read_example_004", agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{"read_example_004": tool})
	sched := &agent.Scheduler{}
	policy := &exampleDeferOncePolicy{}

	provider := agenttest.NewProvider(
		exampleToolCallScript("call-example-004", "read_example_004", []byte(`{}`)),
		exampleTextScript("hello from the example"),
	)
	h := agent.Harness{
		Provider:  provider,
		System:    "a minimal system prompt",
		Turn:      agent.TurnOptions{Tools: reg, PermissionPolicy: policy},
		Scheduler: sched,
	}

	sink := make(chan *agent.Event)
	go func() {
		_, _, _ = h.Run(context.Background(), exampleUserMessage("hi"), sink)
	}()

	for ev := range sink {
		if req, ok := ev.PermissionDecisionRequired(); ok {
			fmt.Println("requested:", req.CallID())
			if err := sched.WakeParked(req.CallID()); err != nil {
				panic(err)
			}
			continue
		}
		if made, ok := ev.PermissionDecisionMade(); ok {
			fmt.Println("resolved:", made.Outcome())
		}
	}
	// Output:
	// requested: call-example-004
	// resolved: allow_once
}
