// AI-26.1 — the wire skeleton's expectation harness, registries and
// determinism pin.
//
// Expectation cases are inline literals (NFR-ART-D): this adapter uses no
// golden-file tree and no -update flag. What proves cross-run determinism
// is that the "want" bytes below are checked into source and compared in a
// fresh `go test` process (S-ART-009) — never where they live.
package openaicompat_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// expectationCase is one translation this harness proves byte-exact.
//
// want is a raw-string inline literal, never a golden file (NFR-ART-D):
// the case and its expected bytes appear together in one review hunk.
type expectationCase struct {
	name  string
	build func() ai.Request
	want  string
}

// expectationCases is the registry every node's table self-registers
// cases into via its own init(), so a later slice's test file adds cases
// with no edit to this one (design.md "Expectation harness", following
// AI-27's sweepTranscripts precedent).
var expectationCases []expectationCase

// registerExpectations appends cases to the shared registry from each
// node's own init(). It is called at most a handful of times, from a
// fixed set of init() functions across this package's test files — never
// ranged as a map — so registration order is append order.
func registerExpectations(cases ...expectationCase) {
	expectationCases = append(expectationCases, cases...)
}

func init() {
	registerExpectations(
		expectationCase{
			name: "one user text message",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}))
			},
			want: `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true}}`,
		},
		expectationCase{
			name: "differs only in model identity",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o-mini", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
				}))
			},
			want: `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true}}`,
		},
	)
}

// TestExpectationCases_MatchByteExact is the suite's primary proof
// (R-ART-002, S-ART-006, S-ART-007): every registered case's translation
// equals its checked-in expectation, byte for byte. Run as an independent
// `go test` process and repeated across independent invocations, this is
// also the cross-run determinism proof (R-ART-003, S-ART-009) — never the
// same-process double-translate below, which is only a pin.
func TestExpectationCases_MatchByteExact(t *testing.T) {
	for _, tc := range expectationCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := openaicompat.Translate(tc.build())
			if err != nil {
				t.Fatalf("Translate: unexpected error: %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("Translate() =\n%s\nwant\n%s", got, tc.want)
			}
		})
	}
}

// TestTranslate_TwoModelsDifferOnlyInModelField proves S-ART-007 directly,
// beyond the byte-exact registry walk above: two requests differing only
// in model identity translate to bodies differing in exactly that field.
func TestTranslate_TwoModelsDifferOnlyInModelField(t *testing.T) {
	message := mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!"))))

	first, err := openaicompat.Translate(mustRequest(ai.NewRequest("gpt-4o", []ai.Message{message})))
	if err != nil {
		t.Fatalf("Translate (first): %v", err)
	}
	second, err := openaicompat.Translate(mustRequest(ai.NewRequest("gpt-4o-mini", []ai.Message{message})))
	if err != nil {
		t.Fatalf("Translate (second): %v", err)
	}

	wantFirst := `{"model":"gpt-4o","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true}}`
	wantSecond := `{"model":"gpt-4o-mini","messages":[{"role":"user","content":"Hello, world!"}],"stream":true,"stream_options":{"include_usage":true}}`
	if string(first) != wantFirst {
		t.Fatalf("Translate(gpt-4o) = %s, want %s", first, wantFirst)
	}
	if string(second) != wantSecond {
		t.Fatalf("Translate(gpt-4o-mini) = %s, want %s", second, wantSecond)
	}
}

// TestTranslate_SameProcessDoubleTranslate_IsAPin is a cheap same-process
// sanity check — explicitly NOT the determinism proof (R-ART-003,
// S-ART-010, S-ART-012). Go re-randomizes map-range start per
// range-statement execution, so a same-process check can pass by chance
// even when a map leaks into the translation path. The proof is
// TestExpectationCases_MatchByteExact, run as an independent `go test`
// process and repeated across independent invocations (S-ART-009).
func TestTranslate_SameProcessDoubleTranslate_IsAPin(t *testing.T) {
	req := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleUser, mustPart(ai.NewText("Hello, world!")))),
	}))

	first, err := openaicompat.Translate(req)
	if err != nil {
		t.Fatalf("Translate (first call): %v", err)
	}
	second, err := openaicompat.Translate(req)
	if err != nil {
		t.Fatalf("Translate (second call): %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("same-process double-translate differed:\nfirst=%s\nsecond=%s", first, second)
	}
}

// mustRequest panics if err != nil. A test-only helper: expectationCase's
// build func is total (func() ai.Request, no error), so a construction
// failure surfaces as a panic inside the t.Run closure that invokes it,
// which go test reports as a failed subtest.
func mustRequest(req ai.Request, err error) ai.Request {
	if err != nil {
		panic(err)
	}
	return req
}

// mustMessage panics if err != nil. See mustRequest.
func mustMessage(msg ai.Message, err error) ai.Message {
	if err != nil {
		panic(err)
	}
	return msg
}

// mustPart panics if err != nil. See mustRequest.
func mustPart(part ai.Part, err error) ai.Part {
	if err != nil {
		panic(err)
	}
	return part
}
