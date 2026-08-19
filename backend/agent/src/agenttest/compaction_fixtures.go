// AG-18 — shared compaction fixtures (design AD-9, tasks.md Phase 7).
//
// This package MUST NOT import github.com/cachicamas/backend/agent/src/agent:
// agent's own test binary already links package agent_test (which
// imports agenttest) together with package agent itself, so agenttest
// importing agent back would be a build-time import cycle in the test
// build specifically (a normal, non-test `go build` would not catch
// this — `go vet`/`go test` do). That is why the marks-bearing fixture
// design AD-9 calls for — a history driven through a real
// agent.Harness.Run so its turns are genuinely marked — is NOT a
// function in this file: agent.History's marked-close door is
// package-private (R-CMP-012, S-CMP-035), reachable only from inside
// package agent, so the ONLY caller able to obtain one is code that
// already imports agent directly — package agent_test, which this
// capability's own compaction_call_test.go (markedHarnessForCompaction)
// and compaction_surgery_test.go (appendAndMark, package-internal)
// already provide. What this file DOES provide is the fixture shape
// that needs no agent import at all: a mis-aligned-cut transcript,
// pure ai.Message values a caller can seed a history or a script from
// directly.

package agenttest

import "github.com/cachicamas/backend/agent/src/ai"

// MisalignedCutFixture is the mis-aligned-cut transcript builder's own
// result: an ordered transcript containing exactly one tool call at
// CallIndex and its matching result at ResultIndex (ResultIndex >
// CallIndex), plus the naive cut a caller should use to exercise
// exactly this straddle.
type MisalignedCutFixture struct {
	// Messages is the ordered transcript: a leading plain message, the
	// tool call, then its result.
	Messages []ai.Message
	// CallIndex is the tool call's own 0-based index in Messages.
	CallIndex int
	// ResultIndex is the tool result's own 0-based index in Messages
	// (always CallIndex+1 in this fixture's own shape).
	ResultIndex int
	// NaiveCut is a boundary strictly between CallIndex and
	// ResultIndex in the cut-as-prefix-length coordinate system (i.e.
	// CallIndex < NaiveCut <= ResultIndex): a compaction request naming
	// this Cut splits the pair unless it is properly resolved.
	NaiveCut int
}

// NewMisalignedCutFixture builds a scripted call/result pair straddling
// a naive cut (R-CMP-004's own scenario shape, S-CMP-010): a leading
// plain assistant message at index 0, a tool call at index 1, and its
// matching result at index 2, with NaiveCut = 2 (a prefix ending
// strictly between the call and its result). callID names the call;
// leadingText is the leading message's own content.
//
// Panics on a Layer 1 construction failure — an impossible outcome for
// the fixed, well-formed inputs this builder itself supplies, mirroring
// this package's own Emit precedent for a fixture-authoring defect that
// is decidable the moment the fixture is built, never at the point a
// caller drives it.
func NewMisalignedCutFixture(leadingText, callID string) MisalignedCutFixture {
	leadingPart, err := ai.NewText(leadingText)
	if err != nil {
		panic("agenttest: NewMisalignedCutFixture: " + err.Error())
	}
	leading, err := ai.NewMessage(ai.RoleAssistant, leadingPart)
	if err != nil {
		panic("agenttest: NewMisalignedCutFixture: " + err.Error())
	}

	callPart, err := ai.NewToolCall(callID, "a-tool", nil)
	if err != nil {
		panic("agenttest: NewMisalignedCutFixture: " + err.Error())
	}
	call, err := ai.NewMessage(ai.RoleAssistant, callPart)
	if err != nil {
		panic("agenttest: NewMisalignedCutFixture: " + err.Error())
	}

	resultPart, err := ai.NewToolResult(callID, "ok")
	if err != nil {
		panic("agenttest: NewMisalignedCutFixture: " + err.Error())
	}
	result, err := ai.NewMessage(ai.RoleTool, resultPart)
	if err != nil {
		panic("agenttest: NewMisalignedCutFixture: " + err.Error())
	}

	return MisalignedCutFixture{
		Messages:    []ai.Message{leading, call, result},
		CallIndex:   1,
		ResultIndex: 2,
		NaiveCut:    2,
	}
}
