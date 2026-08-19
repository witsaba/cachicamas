// AG-18 — fixture physics for compaction_fixtures.go (tasks.md Phase
// 7.2): the mis-aligned builder's pair genuinely straddles the stated
// cut. (The marks-bearing fixture's own physics are proved in package
// agent_test, where it actually lives — markedHarnessForCompaction in
// compaction_call_test.go and appendAndMark in
// compaction_surgery_test.go — since building one requires importing
// package agent directly, which this package must not do; see
// compaction_fixtures.go's own package doc comment.)
package agenttest_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/agenttest"
)

// TestNewMisalignedCutFixture_PairGenuinelyStraddlesTheCut confirms the
// fixture's own physics: NaiveCut falls strictly between CallIndex and
// ResultIndex, and both indices name a real tool-call/tool-result pair
// on the same callID within Messages.
func TestNewMisalignedCutFixture_PairGenuinelyStraddlesTheCut(t *testing.T) {
	t.Parallel()

	fx := agenttest.NewMisalignedCutFixture("leading reply", "call-fixture-001")

	if fx.CallIndex >= fx.NaiveCut || fx.NaiveCut > fx.ResultIndex {
		t.Fatalf("NaiveCut=%d does not fall strictly between CallIndex=%d and ResultIndex=%d", fx.NaiveCut, fx.CallIndex, fx.ResultIndex)
	}
	if fx.ResultIndex >= len(fx.Messages) || fx.CallIndex >= len(fx.Messages) {
		t.Fatalf("indices out of range: CallIndex=%d ResultIndex=%d, len(Messages)=%d", fx.CallIndex, fx.ResultIndex, len(fx.Messages))
	}

	callPart, ok := fx.Messages[fx.CallIndex].Content()[0].ToolCall()
	if !ok {
		t.Fatalf("Messages[%d] does not carry a tool call", fx.CallIndex)
	}
	resultPart, ok := fx.Messages[fx.ResultIndex].Content()[0].ToolResult()
	if !ok {
		t.Fatalf("Messages[%d] does not carry a tool result", fx.ResultIndex)
	}
	if callPart.ID() != resultPart.CallID() {
		t.Errorf("call ID %q does not match result's CallID %q -- the pair does not actually match", callPart.ID(), resultPart.CallID())
	}
}
