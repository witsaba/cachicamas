// AI-26.4 — tool declarations translate byte-faithfully and in
// deterministic declaration order (R-ART-010, R-ART-011).
//
// The registered case below is this node's byte-exact proof (S-ART-035,
// S-ART-036) and, run as an independent `go test` process and repeated
// across independent invocations exactly like
// TestExpectationCases_MatchByteExact already is for R-ART-003
// (translation_test.go, S-ART-009), is also the fresh-process tool-order
// determinism proof this node requires (R-ART-011, S-ART-038): a
// same-process double-translate would not be sufficient evidence, for the
// same reason translation_test.go's own pin is not — Go re-randomizes
// map-range start per range-statement execution, so an in-process check
// can pass by chance even when a map leaks into the translation path. The
// failure mode a leak here produces is silent: a valid body, no error and
// no test failure — only a broken vendor cache prefix and an input bill
// roughly ten times larger than it should be (S-ART-041).
package openaicompat_test

import (
	"errors"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// searchFlightsSchema is a tool's argument schema carrying a distinctive
// key order (top-level: type, properties, required, additionalProperties
// — not alphabetical; nested "properties": destination, cabin_class — not
// alphabetical either) and distinctive whitespace (indentation, an
// irregular run of spaces before one nested object, internal newlines).
// Any decode/re-encode round trip would compact this whitespace away and
// alphabetically re-sort every one of these keys (encoding/json.Marshal
// sorts a map's keys) — see tool.go's own citation of why that never
// happens (S-ART-035, S-ART-037).
const searchFlightsSchema = `{
  "type": "object",
  "properties": {
    "destination":   { "type": "string" },
    "cabin_class": {"type":"string","enum":["economy","business"]}
  },
  "required": ["destination"],
  "additionalProperties":    false
}`

const cancelBookingSchema = `{"type":"object","properties":{"booking_id":{"type":"string"}},"required":["booking_id"]}`

const getWeatherSchema = `{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`

func init() {
	registerExpectations(
		expectationCase{
			name: "tools carry byte-faithful schema, name, description and declaration order",
			build: func() ai.Request {
				searchFlights := mustTool(ai.NewTool(
					"search_flights",
					"Search available flights between two cities.",
					[]byte(searchFlightsSchema),
				))
				cancelBooking := mustTool(ai.NewTool(
					"cancel_booking",
					"Cancel an existing booking by its identifier.",
					[]byte(cancelBookingSchema),
				))
				getWeather := mustTool(ai.NewTool(
					"get_weather",
					"Get the current weather for a location.",
					[]byte(getWeatherSchema),
				))
				// Declaration order is deliberately not alphabetical
				// (alphabetical would be cancel_booking, get_weather,
				// search_flights): S-ART-038/S-ART-039 require the
				// caller's own order to be what appears on the wire, not
				// a sorted one.
				tools := mustToolSet(ai.NewToolSet(searchFlights, cancelBooking, getWeather))
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Plan a trip to Paris.")))),
				}, ai.WithTools(tools)))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Plan a trip to Paris."}],"tools":[` +
				`{"type":"function","function":{"name":"search_flights","description":"Search available flights between two cities.","parameters":` + searchFlightsSchema + `}},` +
				`{"type":"function","function":{"name":"cancel_booking","description":"Cancel an existing booking by its identifier.","parameters":` + cancelBookingSchema + `}},` +
				`{"type":"function","function":{"name":"get_weather","description":"Get the current weather for a location.","parameters":` + getWeatherSchema + `}}` +
				`],"stream":true,"stream_options":{"include_usage":true}}`,
		},
	)
}

// mustTool panics if err != nil. See mustRequest in translation_test.go.
func mustTool(tool ai.Tool, err error) ai.Tool {
	if err != nil {
		panic(err)
	}
	return tool
}

// mustToolSet panics if err != nil. See mustRequest in translation_test.go.
func mustToolSet(tools ai.ToolSet, err error) ai.ToolSet {
	if err != nil {
		panic(err)
	}
	return tools
}

// TestTool_ReversedOrderTwin_TranslatesDifferently proves S-ART-039: two
// requests declaring the same tools but in reversed order translate to
// different bodies — declaration order is genuinely carried, not
// incidentally equal — mirroring
// TestSystemSegment_ReversedOrderTwin_TranslatesDifferently
// (system_segment_test.go, AI-26.2).
func TestTool_ReversedOrderTwin_TranslatesDifferently(t *testing.T) {
	build := func(tools ...ai.Tool) ai.Request {
		toolSet, err := ai.NewToolSet(tools...)
		if err != nil {
			t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
		}
		request, err := ai.NewRequest("gpt-4o", []ai.Message{
			mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
		}, ai.WithTools(toolSet))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		return request
	}

	alpha, err := ai.NewTool("alpha_tool", "The first tool.", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}
	beta, err := ai.NewTool("beta_tool", "The second tool.", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}

	forward := build(alpha, beta)
	reversed := build(beta, alpha)

	forwardBytes, err := openaicompat.Translate(forward)
	if err != nil {
		t.Fatalf("Translate(forward): unexpected error: %v", err)
	}
	reversedBytes, err := openaicompat.Translate(reversed)
	if err != nil {
		t.Fatalf("Translate(reversed): unexpected error: %v", err)
	}

	if string(forwardBytes) == string(reversedBytes) {
		t.Fatalf("Translate(forward) == Translate(reversed) = %s, want different bytes — declaration order must be carried, not incidentally equal", forwardBytes)
	}
}

// TestToolSet_DuplicateOrInvalidNamesNeverReachTranslate is a regression
// pin, not a red-required scenario (tasks.md's fourth evidence class:
// green on first write because a different, already-shipped package's
// code already structurally satisfies the property — disclosed rather
// than fabricated as a red). ai.NewToolSet (AI-08.2, tool_set.go) already
// rejects a duplicate tool name with ai.ErrDuplicate, and a tool that
// never passed ai.NewTool (reported by its empty name) with ai.ErrEmpty,
// both before a ToolSet value can exist at all. ai.WithTools takes an
// already-constructed ToolSet, and ToolSet's own tools field is
// unexported with no mutator, so a Request handed to
// openaicompat.Translate can never carry a duplicate or invalid-shaped
// tool declaration. tool.go therefore contains no duplicate-detection or
// name-validation logic of its own — it does not need any — and this test
// exercises ai.NewToolSet's own upstream behaviour to pin that assumption
// down, not anything this package implements.
func TestToolSet_DuplicateOrInvalidNamesNeverReachTranslate(t *testing.T) {
	valid, err := ai.NewTool("get_weather", "Get the weather.", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}
	duplicate, err := ai.NewTool("get_weather", "A second declaration of the same name.", []byte(`{"type":"object"}`))
	if err != nil {
		t.Fatalf("ai.NewTool returned %v, want no failure", err)
	}

	if _, err := ai.NewToolSet(valid, duplicate); !errors.Is(err, ai.ErrDuplicate) {
		t.Fatalf("ai.NewToolSet(valid, duplicate): err = %v, want ai.ErrDuplicate", err)
	}
	if _, err := ai.NewToolSet(valid, ai.Tool{}); !errors.Is(err, ai.ErrEmpty) {
		t.Fatalf("ai.NewToolSet(valid, ai.Tool{}): err = %v, want ai.ErrEmpty", err)
	}
}
