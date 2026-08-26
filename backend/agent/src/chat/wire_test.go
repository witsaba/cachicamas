// CH-02.1 — pin the camelCase wire contract that the chat layer1
// frontend (frontend/src/lib/chat-types.ts) depends on. The wire types
// in chat/wire.go MUST serialise with camelCase keys — the Go default
// would emit PascalCase (MessageID, Index, Delta, FinishReason, ...) and
// break every read in chat-api.ts:437-505 and parseTranscript
// (chat-types.ts:296-420). This file locks that contract so a future
// refactor that drops the json tags fails RED at compile-or-test time.

package chat_test

import (
	"encoding/json"
	"testing"

	"github.com/cachicamas/backend/agent/src/chat"
)

// TestWireEventsSerializeToCamelCaseJSON — every exported field on every
// WireEvent variant serialises with the camelCase key the frontend
// frozen contract expects. Without explicit json tags Go's encoding/json
// defaults to the field name, which is PascalCase for exported Go
// fields — exactly the bug this test guards against.
func TestWireEventsSerializeToCamelCaseJSON(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		ev   chat.WireEvent
		want string
	}{
		{
			name: "MessageStart",
			ev:   chat.MessageStart{MessageID: "msg-7", Index: 0},
			want: `{"messageId":"msg-7","index":0}`,
		},
		{
			name: "MessageDelta",
			ev:   chat.MessageDelta{Index: 0, Delta: "hi"},
			want: `{"index":0,"delta":"hi"}`,
		},
		{
			name: "MessageEnd",
			ev:   chat.MessageEnd{Index: 0, FinishReason: chat.FinishReasonStop},
			want: `{"index":0,"finishReason":"stop"}`,
		},
		{
			name: "TurnEnd_nil_FinishReason",
			ev:   chat.TurnEnd{FinishReason: nil},
			want: `{}`,
		},
		{
			name: "TurnEnd_with_FinishReason",
			ev:   chat.TurnEnd{FinishReason: ptr("stop")},
			want: `{"finishReason":"stop"}`,
		},
		{
			name: "Error",
			ev:   chat.Error{Kind: "server", Message: "x"},
			want: `{"kind":"server","message":"x"}`,
		},
		{
			name: "ToolCallStart",
			ev: chat.ToolCallStart{
				WireCallID: "wc-1",
				Tool:       "search",
				Arguments:  "{}",
			},
			want: `{"wireCallId":"wc-1","tool":"search","arguments":"{}"}`,
		},
		{
			name: "PermissionDecisionRequired",
			ev: chat.PermissionDecisionRequired{
				WireCallID: "wc-2",
				Tool:       "write_file",
				Arguments:  `{"path":"x"}`,
			},
			want: `{"wireCallId":"wc-2","tool":"write_file","arguments":"{\"path\":\"x\"}"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := json.Marshal(tc.ev)
			if err != nil {
				t.Fatalf("json.Marshal(%T) returned %v, want nil", tc.ev, err)
			}
			if string(got) != tc.want {
				t.Errorf("json.Marshal(%T)\n got %s\nwant %s", tc.ev, got, tc.want)
			}
		})
	}
}

// ptr is a tiny helper to take the address of a string literal — Go
// can't address a literal directly.
func ptr(s string) *string { return &s }