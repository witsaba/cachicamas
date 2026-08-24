// AG-18.3 — Phase 4: the bracket and the stream record (R-CMP-008,
// R-CMP-013), package agent_test (NFR-CMP-001).
package agent_test

import (
	"os"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/agent"
	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// TestCompaction_StreamAcceptedUnmodified — S-CMP-017, charter AG-18.3
// scenario, the placement half. A run in which a strategy-requested
// compaction completes: CheckStream accepts the recorded stream
// unmodified; the compaction bracket's own event order is turn_start,
// compaction_started, cost_turn, compaction_finished, turn_end
// (R-CMP-008); its TurnID is distinct from every model turn's TurnID
// on the same stream; and the every-kind-constructible vocabulary
// still carries exactly 25 members, AG-18 registering none.
func TestCompaction_StreamAcceptedUnmodified(t *testing.T) {
	t.Parallel()

	const toolName, callID = "read_cmp_017", "call-cmp-017"
	tool := EchoScriptedTool(toolName, agent.EffectClassRead)
	reg := agent.NewMapRegistry(map[string]agent.Tool{toolName: tool})

	ownProvider := agenttest.NewProvider(
		scriptToolCallResponse(t, callID, toolName, []byte(`{"x":1}`)),
		scriptTextResponse(t, ai.FinishReasonStop),
	)
	compactionProvider := agenttest.NewProvider(scriptTextResponse(t, ai.FinishReasonStop))
	strategy := &compactAfterNStrategy{
		skip: 1, // skip the pre-turn-1 consultation (no mark exists yet).
		request: func(prompt agent.ContextPrompt) agent.CompactionRequest {
			return agent.CompactionRequest{Provider: compactionProvider, Instruction: "summarize for S-CMP-017", Cut: len(prompt.Transcript)}
		},
	}
	h := &agent.Harness{Provider: ownProvider, System: "system prompt for S-CMP-017", Turn: agent.TurnOptions{Tools: reg}, ContextStrategy: strategy}

	sink := make(chan *agent.Event, 256)
	if _, _, err := h.Run(contextBackground(), firstMessage(t), sink); err != nil {
		t.Fatalf("Run returned %v, want nil", err)
	}
	events := drainSink(t, sink)

	if report := agent.CheckStream(events); report.Violation() != nil {
		t.Fatalf("CheckStream rejected the recorded stream: %v", report.Violation())
	}

	var compactionTurnID agent.TurnID
	for _, ev := range events {
		if ev.Kind() == agent.EventKindCompactionStarted {
			compactionTurnID, _ = ev.Turn()
		}
	}
	if compactionTurnID == "" {
		t.Fatal("no compaction_started event found on the stream")
	}

	var bracketKinds []agent.EventKind
	modelTurnIDs := map[agent.TurnID]bool{}
	for _, ev := range events {
		turn, hasTurn := ev.Turn()
		if !hasTurn {
			continue
		}
		if turn == compactionTurnID {
			bracketKinds = append(bracketKinds, ev.Kind())
		} else {
			modelTurnIDs[turn] = true
		}
	}

	wantOrder := []agent.EventKind{
		agent.EventKindTurnStart,
		agent.EventKindCompactionStarted,
		agent.EventKindCostTurn,
		agent.EventKindCompactionFinished,
		agent.EventKindTurnEnd,
	}
	if len(bracketKinds) != len(wantOrder) {
		t.Fatalf("compaction bracket kinds = %v, want %v", bracketKinds, wantOrder)
	}
	for i, want := range wantOrder {
		if bracketKinds[i] != want {
			t.Errorf("bracket[%d] = %v, want %v", i, bracketKinds[i], want)
		}
	}
	if len(modelTurnIDs) == 0 {
		t.Fatal("fixture defect: no model-turn TurnID observed on the stream")
	}
	if modelTurnIDs[compactionTurnID] {
		t.Error("the compaction bracket's own TurnID collides with a model turn's TurnID, want distinct namespaces")
	}

	if kinds := agent.EventKinds(); len(kinds) != 25 {
		t.Errorf("agent.EventKinds() returned %d kind(s), want exactly 25 (AG-06 minted the compaction family; AG-18 registers none)", len(kinds))
	}
}

// TestCompaction_SubstrateByteUnchanged — S-CMP-036, R-CMP-013,
// NFR-CMP-004, cross-referenced to S-LSK-029. Diffing the merge base
// against the working tree: every file this capability names
// byte-unchanged is byte-unchanged, the diff under
// backend/agent/src/ai/ is empty, go.mod/go.sum are byte-unchanged,
// and the every-kind-constructible guard passes at its committed
// count with AG-18 registering none.
func TestCompaction_SubstrateByteUnchanged(t *testing.T) {
	root, err := gitTopLevel(t)
	if err != nil {
		t.Fatalf("git rev-parse --show-toplevel failed: %v", err)
	}

	mainRef := os.Getenv("AG07_BASE_REF")
	if mainRef == "" {
		out, err := gitOutput(t, root, "merge-base", "HEAD", "origin/main")
		if err != nil {
			t.Fatalf("cannot determine base ref (set AG07_BASE_REF): %v", err)
		}
		mainRef = out
	}

	// AG-18's own release is exactly doc.go and doc_contract_guard_test.go
	// (AD-12); history.go, harness.go and context_strategy.go are also
	// legitimately modified by this change (S-LSK-029's own exact set).
	// Every OTHER named substrate file must stay byte-unchanged.
	untouched := []string{
		"backend/agent/src/agent/reconstruction_test.go",
		// stream_check.go, event.go and event_registry_test.go are
		// removed here — the Layer 2 audit-fix change (branch
		// fix/agent-layer2-audit-findings) legitimately edits all
		// three (per-tool-name CardinalityAtMostOne keying, R-APE-003,
		// and the corrected 25-kind registry prose), mirroring the
		// AG-18 amendment precedent on
		// TestNoRelease_SubstrateByteUnchanged.
		"backend/agent/src/agent/event_descriptor.go",
		"backend/agent/src/agent/sequence.go",
		"backend/agent/src/agent/run_events.go",
		"backend/agent/src/agent/turn_events.go",
		"backend/agent/src/agent/failure.go",
		"backend/agent/src/agent/compaction_events.go",
		"backend/agent/src/agent/cost_events.go",
		"backend/agent/src/agent/cost_events_test.go",
		"backend/agent/src/agent/cost_usage.go",
	}
	for _, path := range untouched {
		diff, err := gitDiff(t, root, mainRef, path)
		if err != nil {
			t.Fatalf("git diff %s -- %s failed: %v", mainRef, path, err)
		}
		if len(diff) != 0 {
			t.Errorf("%s is NOT byte-unchanged against %s (R-CMP-013, NFR-CMP-004):\n%s", path, mainRef, diff)
		}
	}

	aiDiff, err := gitDiff(t, root, mainRef, "backend/agent/src/ai/")
	if err != nil {
		t.Fatalf("git diff %s -- backend/agent/src/ai/ failed: %v", mainRef, err)
	}
	// CH-03 carve-out: the agent module's go.mod require bump from
	// 3 to 4 lines is reflected in TWO ai/ test files
	// (import_boundary_test.go's wantGoModRequires and
	// openrouter/zero_requires_test.go's wantGoModRequireLines)
	// per each test's own documented "update in the same commit"
	// instruction. Anything beyond the two carve-out files fails.
	ch03AiCarveout := []string{
		"backend/agent/src/ai/import_boundary_test.go",
		"backend/agent/src/ai/openaicompat/openrouter/zero_requires_test.go",
	}
	if strings.Contains(aiDiff, "diff --git") {
		chunks := strings.Split(aiDiff, "diff --git")
		var stray []string
		for _, chunk := range chunks[1:] {
			nl := strings.Index(chunk, "\n")
			header := chunk
			if nl >= 0 {
				header = chunk[:nl]
			}
			allowed := false
			for _, file := range ch03AiCarveout {
				if strings.Contains(header, file) {
					allowed = true
					break
				}
			}
			if !allowed {
				stray = append(stray, chunk)
			}
		}
		if len(stray) == 0 {
			t.Logf("CH-03 carve-out: every ai/ diff is one of the documented require-bump amendments.\n%s", aiDiff)
		} else {
			t.Errorf("backend/agent/src/ai/ was edited beyond the CH-03 wantGoModRequires / wantGoModRequireLines amendments (R-CST-007 violated):\n%s", aiDiff)
		}
	} else if len(aiDiff) != 0 {
		t.Errorf("backend/agent/src/ai/ is NOT byte-unchanged against %s -- AG-18 must never edit Layer 1:\n%s", mainRef, aiDiff)
	}

	goModSum, err := gitDiff(t, root, mainRef, "backend/agent/go.mod", "backend/agent/go.sum")
	if err != nil {
		t.Fatalf("git diff %s -- go.mod go.sum failed: %v", mainRef, err)
	}
	// CH-03 (cachicamas-chat-http-surface) carve-out: the chat
	// archetype's HTTP+SSE surface requires github.com/labstack/echo/v5
	// v5.2.1 (ADR adr/echo-v5-in-agent-module), bumping the
	// module's go.mod from 3 require lines (AI-37) to 4. The drift
	// is the recorded exception; every other go.mod / go.sum
	// modification fails this clause.
	if isCH03GoModDrift(goModSum) {
		t.Logf("CH-03 carve-out: go.mod / go.sum drift is the recorded Echo require (4th require line per wantGoModRequires); passes per D3.\n%s", goModSum)
	} else if len(goModSum) != 0 {
		t.Errorf("go.mod / go.sum drifted from %s:\n%s", mainRef, goModSum)
	}

	if len(expectedLayer2ContractRows) != 8 {
		t.Errorf("expectedLayer2ContractRows carries %d row(s), want exactly 8 (L2C-01..L2C-08) -- AG-18 amends L2C-07's text, it adds no new row", len(expectedLayer2ContractRows))
	}
	if kinds := agent.EventKinds(); len(kinds) != 25 {
		t.Errorf("agent.EventKinds() returned %d kind(s), want exactly 25", len(kinds))
	}
}
