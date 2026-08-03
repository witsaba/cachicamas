// AI-26.5 — tool results translate into the vendor's distinct role:"tool"
// message shape (R-ART-012); the wire tool-call identifier is always
// exactly the neutral call's identifier, never minted (R-ART-013); and
// result-to-call correlation survives translation for interleaved
// multi-call turns (R-ART-014).
package openaicompat_test

import (
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

func init() {
	registerExpectations(
		expectationCase{
			// S-ART-042: one tool result renders as exactly one wire
			// tool-role message. The tool call's own identifier
			// ("call_7f3a9c2e") is distinctive, not a coincidentally
			// simple literal like "call_1" — folding S-ART-045's
			// byte-identical-pass-through proof into this and every
			// other case below, rather than a separate, otherwise-empty
			// test (doc 0002's amended AI-26.5 charter: the
			// synthetic-minting branch has no subject for this adapter,
			// since the vendor assigns identifiers on the call's own
			// opening delta).
			name: "one tool result renders as exactly one tool-role message",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("What's the weather in Tokyo?")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_7f3a9c2e", "get_weather", nil)))),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("call_7f3a9c2e", "22C and sunny")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":"What's the weather in Tokyo?"},` +
				`{"role":"assistant","tool_calls":[{"id":"call_7f3a9c2e","type":"function","function":{"name":"get_weather","arguments":"{}"}}]},` +
				`{"role":"tool","tool_call_id":"call_7f3a9c2e","content":"22C and sunny"}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			// S-ART-043: several tool results render as one distinct
			// tool-role message per result, in the caller's own order —
			// two SEPARATE role:"tool" ai.Message values here, each
			// carrying exactly one ai.ToolResult, matching AI-24's own
			// documented parallel-function-calling flow (doc.go's claim
			// 4: the vendor's canonical worked example appends one
			// role:"tool" message per call, onto the same messages list).
			name: "several tool results render as that many distinct tool-role messages, in caller order",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Check the weather in Tokyo and Paris.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewToolCall("call_tokyo_01", "get_weather", nil)),
						mustPart(ai.NewToolCall("call_paris_02", "get_weather", nil)),
					)),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("call_tokyo_01", "22C and sunny")))),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("call_paris_02", "15C and cloudy")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":"Check the weather in Tokyo and Paris."},` +
				`{"role":"assistant","tool_calls":[` +
				`{"id":"call_tokyo_01","type":"function","function":{"name":"get_weather","arguments":"{}"}},` +
				`{"id":"call_paris_02","type":"function","function":{"name":"get_weather","arguments":"{}"}}` +
				`]},` +
				`{"role":"tool","tool_call_id":"call_tokyo_01","content":"22C and sunny"},` +
				`{"role":"tool","tool_call_id":"call_paris_02","content":"15C and cloudy"}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			// S-ART-044: a failed tool result still carries its own
			// content verbatim — not silently rendered as success. See
			// TestToolResult_FailedAndSucceededRenderIdentically below
			// for the falsifiability-focused companion proof that
			// Failed() changes nothing but identity and content.
			name: "a failed tool result carries its own content, not a success placeholder",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Book a table for 8pm.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_booking_9z", "book_table", nil)))),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolFailure("call_booking_9z", "Error: restaurant fully booked")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":"Book a table for 8pm."},` +
				`{"role":"assistant","tool_calls":[{"id":"call_booking_9z","type":"function","function":{"name":"book_table","arguments":"{}"}}]},` +
				`{"role":"tool","tool_call_id":"call_booking_9z","content":"Error: restaurant fully booked"}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			// S-ART-046: an unusual-but-valid identifier shape survives
			// unnormalized. ai.ToolCall carries no shape rule on id
			// beyond non-empty (tool_call.go's own validate), so mixed
			// case, punctuation and a leading digit are all legal — a
			// naive normalizer (case-fold, strip punctuation, trim) would
			// break this case; appendJSONString applies JSON escaping
			// only, never normalization.
			name: "an unusual but valid identifier shape survives verbatim",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("3F2A9B-Weird_ID.42", "ping", nil)))),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("3F2A9B-Weird_ID.42", "pong")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"assistant","tool_calls":[{"id":"3F2A9B-Weird_ID.42","type":"function","function":{"name":"ping","arguments":"{}"}}]},` +
				`{"role":"tool","tool_call_id":"3F2A9B-Weird_ID.42","content":"pong"}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			// S-ART-048: a turn with three tool calls and interleaved,
			// out-of-order results — each wire result names its own
			// call's identifier. Gated on claim 2's request-side
			// confirmation (1.0.2): each call carries its own distinctive
			// arguments object, so claim 2's "arguments is a wire STRING,
			// not a nested object" encoding is genuinely observable in
			// this expectation, not merely asserted.
			//
			// Declared call order is [search_flights, check_weather,
			// book_hotel]; results answer, in caller order,
			// [book_hotel, search_flights, check_weather] — a derangement
			// (no result is in the position its own call declared), which
			// defeats the natural position-based bug (the Nth-declared
			// call's identifier substituted for the Nth-encountered
			// result) at every one of the three positions. The
			// identifiers ("flt-9k2M", "wthr-3Qz", "htl-7Bv1") are also
			// not in an order sortable to the correct pairing: lexical
			// sort is [flt-9k2M, htl-7Bv1, wthr-3Qz], which does not match
			// either the call-declaration order or the result-answering
			// order, so a hypothetical sort-by-identifier substitution
			// would not coincidentally pass either (Trap 7).
			name: "interleaved multi-call results each name their own call's identifier",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Book my Paris trip.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewToolCall("flt-9k2M", "search_flights", []byte(`{"origin":"JFK","dest":"CDG"}`))),
						mustPart(ai.NewToolCall("wthr-3Qz", "check_weather", []byte(`{"city":"Paris"}`))),
						mustPart(ai.NewToolCall("htl-7Bv1", "book_hotel", []byte(`{"nights":2,"city":"Paris"}`))),
					)),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("htl-7Bv1", "Hotel booked, confirmation HB-2291")))),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("flt-9k2M", "Flight AF007 booked")))),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("wthr-3Qz", "18C, light rain")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"user","content":"Book my Paris trip."},` +
				`{"role":"assistant","tool_calls":[` +
				`{"id":"flt-9k2M","type":"function","function":{"name":"search_flights","arguments":"{\"origin\":\"JFK\",\"dest\":\"CDG\"}"}},` +
				`{"id":"wthr-3Qz","type":"function","function":{"name":"check_weather","arguments":"{\"city\":\"Paris\"}"}},` +
				`{"id":"htl-7Bv1","type":"function","function":{"name":"book_hotel","arguments":"{\"nights\":2,\"city\":\"Paris\"}"}}` +
				`]},` +
				`{"role":"tool","tool_call_id":"htl-7Bv1","content":"Hotel booked, confirmation HB-2291"},` +
				`{"role":"tool","tool_call_id":"flt-9k2M","content":"Flight AF007 booked"},` +
				`{"role":"tool","tool_call_id":"wthr-3Qz","content":"18C, light rain"}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			// S-ART-050: two calls to the SAME tool name, with different
			// identifiers, remain distinguishable — results answer in
			// reversed order relative to declaration, so a position-based
			// substitution would also be caught here, independently of
			// the interleaved three-call case above.
			name: "two calls to the same tool name remain distinguishable by identifier",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewToolCall("wthr-tokyo-1", "check_weather", []byte(`{"city":"Tokyo"}`))),
						mustPart(ai.NewToolCall("wthr-paris-2", "check_weather", []byte(`{"city":"Paris"}`))),
					)),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("wthr-paris-2", "15C and cloudy")))),
					mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("wthr-tokyo-1", "22C and sunny")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[` +
				`{"role":"assistant","tool_calls":[` +
				`{"id":"wthr-tokyo-1","type":"function","function":{"name":"check_weather","arguments":"{\"city\":\"Tokyo\"}"}},` +
				`{"id":"wthr-paris-2","type":"function","function":{"name":"check_weather","arguments":"{\"city\":\"Paris\"}"}}` +
				`]},` +
				`{"role":"tool","tool_call_id":"wthr-paris-2","content":"15C and cloudy"},` +
				`{"role":"tool","tool_call_id":"wthr-tokyo-1","content":"22C and sunny"}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
	)
}

// TestToolResult_FailedAndSucceededRenderIdentically proves S-ART-044
// beyond the registered "failed tool result" case above: it cross-checks
// that Failed() changes NOTHING about the wire shape except the call
// identity and content a caller supplied — not merely that ONE
// hand-picked failure case happens to match ONE hand-picked expectation,
// which a coincidentally-correct literal could pass without actually
// proving uniform treatment. Two requests differing ONLY in call
// identifier, content and Failed() translate to bodies differing in
// ONLY the identifier and content: substituting the succeeded case's
// identifier into the succeeded body's own bytes reproduces the failed
// body exactly, so nothing about Failed() itself is visible on the wire
// (AI-24 §6's given shape carries no field for it; tool_result.go's own
// doc comment states a failing tool's answer is content the model
// reasons about, not this package's fault taxonomy).
func TestToolResult_FailedAndSucceededRenderIdentically(t *testing.T) {
	succeeded := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_ok", "check_status", nil)))),
		mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolResult("call_ok", "service is healthy")))),
	}))
	failed := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleAssistant, mustPart(ai.NewToolCall("call_bad", "check_status", nil)))),
		mustMessage(ai.NewMessage(ai.RoleTool, mustPart(ai.NewToolFailure("call_bad", "service is healthy")))),
	}))

	succeededBytes, err := openaicompat.Translate(succeeded)
	if err != nil {
		t.Fatalf("Translate(succeeded): unexpected error: %v", err)
	}
	failedBytes, err := openaicompat.Translate(failed)
	if err != nil {
		t.Fatalf("Translate(failed): unexpected error: %v", err)
	}

	wantSucceeded := `{"model":"gpt-4o","messages":[` +
		`{"role":"assistant","tool_calls":[{"id":"call_ok","type":"function","function":{"name":"check_status","arguments":"{}"}}]},` +
		`{"role":"tool","tool_call_id":"call_ok","content":"service is healthy"}` +
		`],"stream":true,"stream_options":{"include_usage":true}}`
	wantFailed := `{"model":"gpt-4o","messages":[` +
		`{"role":"assistant","tool_calls":[{"id":"call_bad","type":"function","function":{"name":"check_status","arguments":"{}"}}]},` +
		`{"role":"tool","tool_call_id":"call_bad","content":"service is healthy"}` +
		`],"stream":true,"stream_options":{"include_usage":true}}`

	if string(succeededBytes) != wantSucceeded {
		t.Fatalf("Translate(succeeded) =\n%s\nwant\n%s", succeededBytes, wantSucceeded)
	}
	if string(failedBytes) != wantFailed {
		t.Fatalf("Translate(failed) =\n%s\nwant\n%s", failedBytes, wantFailed)
	}
	if reconstructed := strings.ReplaceAll(string(succeededBytes), "call_ok", "call_bad"); reconstructed != string(failedBytes) {
		t.Fatalf("Failed() changed more than the call identity and content:\nsucceeded (id-substituted) = %s\nfailed                     = %s", reconstructed, failedBytes)
	}
}
