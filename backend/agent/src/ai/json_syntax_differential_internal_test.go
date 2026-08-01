// Differential test for json_syntax.go — AI-10.4 item 3's consequence.
//
// encoding/json is imported here, in a test file, on purpose: this guard
// (import_boundary_test.go) excludes test imports from its scan precisely
// because a test may legitimately need something production code must not —
// see that file's own comment on why -test is omitted from this specific
// guard. Using the package this file replaces as an oracle, only in a test,
// is exactly that case.

package ai

import (
	"encoding/json"
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"
)

// TestIsWellFormedJSON_AgreesWithEncodingJSONValid_OverManyGeneratedInputs
// runs isWellFormedJSON and encoding/json.Valid — the function this package
// no longer imports, kept here only as a test oracle — against a large,
// deterministically generated corpus of JSON-shaped and near-JSON-shaped
// byte strings, and requires every verdict to agree.
//
// The RFC 8259 grammar test above pins named cases; this one is the fuzz-like
// complement, covering combinations a hand-picked table would not think to
// name — deeply nested structures, malformed fragments produced by mutating a
// valid one, and generated strings with mixed escapes.
func TestIsWellFormedJSON_AgreesWithEncodingJSONValid_OverManyGeneratedInputs(t *testing.T) {
	t.Parallel()

	const seed = 20260801 // fixed, so a failure is reproducible without -v
	rng := rand.New(rand.NewPCG(seed, seed))

	const iterations = 20000
	disagreements := 0
	for i := range iterations {
		data := []byte(generateJSONish(rng, 0))
		want := json.Valid(data)
		got := isWellFormedJSON(data)
		if got != want {
			disagreements++
			if disagreements <= 10 {
				t.Errorf("iteration %d: isWellFormedJSON(%q) = %t, json.Valid = %t", i, data, got, want)
			}
		}
	}
	if disagreements > 10 {
		t.Errorf("%d of %d generated inputs disagreed with json.Valid (first 10 shown above)", disagreements, iterations)
	}
}

// generateJSONish produces a byte string that is usually valid JSON and
// sometimes a mutation of one, recursively bounded by depth so the generator
// always terminates.
func generateJSONish(rng *rand.Rand, depth int) string {
	var b strings.Builder
	writeJSONishValue(&b, rng, depth)

	// Mutate about a third of the time, after building a structurally valid
	// value, so the corpus is not exclusively well-formed input — a checker
	// that only ever sees valid JSON cannot prove it rejects anything.
	s := b.String()
	if rng.IntN(3) == 0 {
		s = mutate(rng, s)
	}
	if rng.IntN(4) == 0 {
		s = padWithSpace(rng, s)
	}
	return s
}

func writeJSONishValue(b *strings.Builder, rng *rand.Rand, depth int) {
	choices := 7
	if depth >= 4 {
		choices = 5 // stop offering object/array once deep enough
	}
	switch rng.IntN(choices) {
	case 0:
		b.WriteString("true")
	case 1:
		b.WriteString("false")
	case 2:
		b.WriteString("null")
	case 3:
		writeJSONishNumber(b, rng)
	case 4:
		writeJSONishString(b, rng)
	case 5:
		writeJSONishArray(b, rng, depth+1)
	case 6:
		writeJSONishObject(b, rng, depth+1)
	}
}

func writeJSONishNumber(b *strings.Builder, rng *rand.Rand) {
	if rng.IntN(2) == 0 {
		b.WriteByte('-')
	}
	if rng.IntN(5) == 0 {
		b.WriteByte('0')
	} else {
		b.WriteString(strconv.Itoa(1 + rng.IntN(999)))
	}
	if rng.IntN(3) == 0 {
		b.WriteByte('.')
		b.WriteString(strconv.Itoa(rng.IntN(1000)))
	}
	if rng.IntN(3) == 0 {
		b.WriteByte([]byte{'e', 'E'}[rng.IntN(2)])
		if rng.IntN(2) == 0 {
			b.WriteByte([]byte{'+', '-'}[rng.IntN(2)])
		}
		b.WriteString(strconv.Itoa(1 + rng.IntN(10)))
	}
}

var jsonishWords = []string{"a", "café", "path", "value", "with space", `with "quote"`, `with\backslash`, "line\\nbreak", ""}

func writeJSONishString(b *strings.Builder, rng *rand.Rand) {
	b.WriteByte('"')
	b.WriteString(jsonishWords[rng.IntN(len(jsonishWords))])
	b.WriteByte('"')
}

func writeJSONishArray(b *strings.Builder, rng *rand.Rand, depth int) {
	b.WriteByte('[')
	n := rng.IntN(4)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONishValue(b, rng, depth)
	}
	b.WriteByte(']')
}

func writeJSONishObject(b *strings.Builder, rng *rand.Rand, depth int) {
	b.WriteByte('{')
	n := rng.IntN(4)
	for i := range n {
		if i > 0 {
			b.WriteByte(',')
		}
		writeJSONishString(b, rng)
		b.WriteByte(':')
		writeJSONishValue(b, rng, depth)
	}
	b.WriteByte('}')
}

// mutate applies one small, syntax-breaking edit to an otherwise valid
// fragment: dropping a byte, duplicating one, or appending garbage — the
// shapes a real caller's truncated or concatenated bytes actually take.
func mutate(rng *rand.Rand, s string) string {
	if s == "" {
		return s
	}
	switch rng.IntN(3) {
	case 0: // drop one byte
		i := rng.IntN(len(s))
		return s[:i] + s[i+1:]
	case 1: // duplicate one byte
		i := rng.IntN(len(s))
		return s[:i] + string(s[i]) + s[i:]
	default: // append trailing garbage
		return s + "}]garbage"
	}
}

func padWithSpace(rng *rand.Rand, s string) string {
	pad := []string{"", " ", "\t", "\n", "  \t\n "}
	return pad[rng.IntN(len(pad))] + s + pad[rng.IntN(len(pad))]
}
