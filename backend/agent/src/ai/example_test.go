// AI-40.2 — four runnable examples covering request construction,
// streaming, tool-call reconstruction and error inspection (R-L2H-002,
// design AD-2). Each carries a deterministic // Output: block, so
// `make test` compiles AND runs every one — a drift between an example
// and the surface it demonstrates fails the build rather than rotting
// silently (S-L2H-007, S-L2H-009).
//
// package ai_test is an external test package; importing src/agenttest
// for the streaming and tool-call examples creates no cycle (Go permits
// an external test package to import a package that itself imports the
// package under test) — proved by this file compiling under `go vet`
// (design AD-2's compile-first step). No example reads a credential,
// performs network I/O, or orders events with a sleep (S-L2H-010).
package ai_test

import (
	"context"
	"fmt"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// ExampleNewRequest demonstrates request construction: one user text
// part, wrapped in a message, wrapped in a request.
func ExampleNewRequest() {
	part, err := ai.NewText("What is the capital of France?")
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	req, err := ai.NewRequest("example-model", []ai.Message{message})
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	fmt.Println("model:", req.Model())
	fmt.Println("messages:", len(req.Messages()))
	// Output:
	// model: example-model
	// messages: 1
}

// ExampleModelProvider_streaming demonstrates driving an ai.ModelProvider
// (here, the AI-21 scripted fake) and draining its event stream in
// order.
func ExampleModelProvider_streaming() {
	part, err := ai.NewText("Say hi")
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	req, err := ai.NewRequest("example-model", []ai.Message{message})
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	start, err := ai.NewTextBlockStart(1)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	delta, err := ai.NewTextDelta(1, "hi")
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	end, err := ai.NewTextBlockEnd(1)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	completion, err := ai.NewCompletion(ai.FinishReasonStop, ai.Usage{})
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	provider := agenttest.NewProvider(agenttest.Script{
		Steps: []agenttest.Step{
			agenttest.Emit(start),
			agenttest.Emit(delta),
			agenttest.Emit(end),
			agenttest.Emit(completion),
		},
	})

	ch, err := provider.Stream(context.Background(), req)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	for event := range ch {
		switch event.Kind() {
		case ai.EventKindTextBlockStart:
			fmt.Println("text_block_start")
		case ai.EventKindTextDelta:
			fmt.Println("text_delta")
		case ai.EventKindTextBlockEnd:
			fmt.Println("text_block_end")
		case ai.EventKindCompletion:
			fmt.Println("completion")
		}
	}
	// Output:
	// text_block_start
	// text_delta
	// text_block_end
	// completion
}

// ExampleModelProvider_toolCallReconstruction demonstrates reconstructing
// a streamed tool call: the name arrives on the start event, and the
// end event's arguments are authoritative — byte-for-byte equal to the
// ordered concatenation of the call's delta fragments.
func ExampleModelProvider_toolCallReconstruction() {
	part, err := ai.NewText("What is the weather in Paris?")
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	req, err := ai.NewRequest("example-model", []ai.Message{message})
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	start, err := ai.NewToolCallStart(1, "call-1", "get_weather")
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	delta1, err := ai.NewToolCallDelta(1, []byte(`{"city":`))
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	delta2, err := ai.NewToolCallDelta(1, []byte(`"Paris"}`))
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	end, err := ai.NewToolCallEnd(1, []byte(`{"city":"Paris"}`))
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	provider := agenttest.NewProvider(agenttest.Script{
		Steps: []agenttest.Step{
			agenttest.Emit(start),
			agenttest.Emit(delta1),
			agenttest.Emit(delta2),
			agenttest.Emit(end),
		},
	})

	ch, err := provider.Stream(context.Background(), req)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	var name string
	var arguments []byte
	for event := range ch {
		switch event.Kind() {
		case ai.EventKindToolCallStart:
			toolStart, _ := event.ToolCallStart()
			name = toolStart.Name()
		case ai.EventKindToolCallEnd:
			toolEnd, _ := event.ToolCallEnd()
			arguments = toolEnd.Arguments()
		}
	}

	fmt.Println("name:", name)
	fmt.Println("arguments:", string(arguments))
	// Output:
	// name: get_weather
	// arguments: {"city":"Paris"}
}

// ExampleFailure_inspection demonstrates inspecting a scripted terminal
// error through the neutral failure surface only: Event.ErrorPayload,
// then Failure.Category and Failure.Retryable — never a vendor type.
func ExampleFailure_inspection() {
	part, err := ai.NewText("Ping")
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	message, err := ai.NewMessage(ai.RoleUser, part)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	req, err := ai.NewRequest("example-model", []ai.Message{message})
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	failure, err := ai.MidStreamFailure(ai.FailureReport{
		Category:  ai.FailureCategoryUnavailable,
		Retryable: true,
	}, false)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}
	errEvent, err := ai.ErrorEvent(failure)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	provider := agenttest.NewProvider(agenttest.Script{
		Steps: []agenttest.Step{
			agenttest.Emit(errEvent),
		},
	})

	ch, err := provider.Stream(context.Background(), req)
	if err != nil {
		fmt.Println("unexpected error:", err)
		return
	}

	for event := range ch {
		payload, ok := event.ErrorPayload()
		if !ok {
			continue
		}
		fmt.Println("category:", payload.Category())
		fmt.Println("retryable:", payload.Retryable())
	}
	// Output:
	// category: unavailable
	// retryable: true
}
