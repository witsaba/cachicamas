// Tests for AI-08.1 — a tool declaration is constructible and readable.
//
// The package under test is imported by its module path from the external test
// package ai_test, so every assertion below is written against exactly the
// surface a consumer in another package sees. V-REQ-12's declaration is a
// transport representation; nothing here executes, resolves or interprets it.
package ai_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// AI-08.1 — a declaration is constructed with a name, a description and schema
// bytes, and all three read back exactly from an external package.
//
// This is the milestone's walking skeleton: the thinnest end-to-end path
// through the public surface. Every later leaf widens it.
//
// The empty description is a case rather than an omission. Every provider a v1
// adapter could target treats a tool's description as optional, and a value
// that is merely recommended is not a caller-contract failure — R-ATD-001's
// second scenario pins that so a later reader does not mistake the absence of a
// rule for an oversight.
func TestTool_NameDescriptionAndSchema_ReadBackExactlyFromAnExternalPackage(t *testing.T) {
	t.Parallel()

	cases := []struct {
		label       string
		name        string
		description string
		schema      string
	}{
		{
			label:       "a fully populated declaration",
			name:        "get_weather",
			description: "Get the current weather for a location.",
			schema:      `{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`,
		},
		{
			label:       "an empty description is legal",
			name:        "list_files",
			description: "",
			schema:      `{"type":"object"}`,
		},
		{
			label:       "a name exactly at the documented ceiling",
			name:        "a" + strings.Repeat("b", ai.MaxToolNameLen-1),
			description: "Exactly at the ceiling.",
			schema:      `{"type":"object"}`,
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			tool, err := ai.NewTool(tc.name, tc.description, []byte(tc.schema))
			if err != nil {
				t.Fatalf("NewTool(%q, ...) returned %v, want no failure", tc.name, err)
			}
			if got := tool.Name(); got != tc.name {
				t.Errorf("tool.Name() = %q, want %q", got, tc.name)
			}
			if got := tool.Description(); got != tc.description {
				t.Errorf("tool.Description() = %q, want %q", got, tc.description)
			}
			if got := string(tool.Schema()); got != tc.schema {
				t.Errorf("tool.Schema() = %q, want %q", got, tc.schema)
			}
		})
	}
}

// AI-08.1 — schema bytes pass through unmodified: no re-marshalling, no key
// reordering, no whitespace normalization.
//
// This is the property AI-26.4 needs for a stable cache prefix, and it cannot
// be added later without breaking every fixture. V-REQ-25 makes the tool set
// the first region of a cached prefix, so a byte that changes between two
// otherwise identical requests invalidates that prefix silently — no error, no
// wrong answer, just a bill roughly ten times larger than it should be.
//
// The fixture is chosen so a normalizing implementation is *detectable*. Its
// object keys are not in alphabetical order and its whitespace is irregular, so
// a marshal-and-unmarshal round trip would rewrite it on the first pass rather
// than being idempotent. Canonical JSON would have made this test pass against
// exactly the implementation it exists to reject.
func TestTool_SchemaBytes_PassThroughByteIdentically(t *testing.T) {
	t.Parallel()

	// Non-alphabetical keys, runs of interior spaces, a tab and a newline.
	const wireSchema = `{"type":"object", "properties":{"zulu":{"type":"string"},` +
		"\n\t" + `"alpha":{"type":"number"}},   "required":["zulu"]}`

	t.Run("the bytes read back byte for byte", func(t *testing.T) {
		t.Parallel()

		tool, err := ai.NewTool("probe", "", []byte(wireSchema))
		if err != nil {
			t.Fatalf("NewTool returned %v, want no failure", err)
		}
		if got := tool.Schema(); !bytes.Equal(got, []byte(wireSchema)) {
			t.Errorf("tool.Schema() = %q, want %q — the bytes were rewritten", got, wireSchema)
		}
	})

	t.Run("the caller may mutate the slice it passed", func(t *testing.T) {
		t.Parallel()

		supplied := []byte(wireSchema)
		tool, err := ai.NewTool("probe", "", supplied)
		if err != nil {
			t.Fatalf("NewTool returned %v, want no failure", err)
		}

		supplied[0] = 'X'
		for i := range supplied {
			supplied[i] = 'Z'
		}

		if got := tool.Schema(); !bytes.Equal(got, []byte(wireSchema)) {
			t.Errorf("tool.Schema() = %q after the caller rewrote its own slice, want %q", got, wireSchema)
		}
	})

	t.Run("a consumer may mutate what a read returned", func(t *testing.T) {
		t.Parallel()

		tool, err := ai.NewTool("probe", "", []byte(wireSchema))
		if err != nil {
			t.Fatalf("NewTool returned %v, want no failure", err)
		}

		first := tool.Schema()
		for i := range first {
			first[i] = 'Z'
		}

		if got := tool.Schema(); !bytes.Equal(got, []byte(wireSchema)) {
			t.Errorf("tool.Schema() = %q after a consumer rewrote what it read, want %q", got, wireSchema)
		}

		second := tool.Schema()
		third := tool.Schema()
		if len(second) > 0 && &second[0] == &third[0] {
			t.Error("two reads share a backing array; one consumer can rewrite another's view")
		}
	})

	t.Run("bytes that no meta-schema would accept are carried unchanged", func(t *testing.T) {
		t.Parallel()

		// R-ATD-004: Layer 1 transports the bytes and never judges them.
		for _, schema := range []string{`not json at all`, `[1,2,3]`, `{`, "\x00\xff\xfe"} {
			tool, err := ai.NewTool("probe", "", []byte(schema))
			if err != nil {
				t.Errorf("NewTool with schema %q returned %v, want no failure — Layer 1 does not judge the schema", schema, err)
				continue
			}
			if got := tool.Schema(); !bytes.Equal(got, []byte(schema)) {
				t.Errorf("tool.Schema() = %q, want %q", got, schema)
			}
		}
	})

	t.Run("two schemas differing only in whitespace stay distinct", func(t *testing.T) {
		t.Parallel()

		const tight = `{"a":1,"b":2}`
		const loose = `{"a": 1,  "b": 2}`

		tightTool, err := ai.NewTool("tight", "", []byte(tight))
		if err != nil {
			t.Fatalf("NewTool(tight) returned %v, want no failure", err)
		}
		looseTool, err := ai.NewTool("loose", "", []byte(loose))
		if err != nil {
			t.Fatalf("NewTool(loose) returned %v, want no failure", err)
		}

		if bytes.Equal(tightTool.Schema(), looseTool.Schema()) {
			t.Error("two schemas differing only in whitespace read back equal; something canonicalized them")
		}
		if got := string(tightTool.Schema()); got != tight {
			t.Errorf("tight schema = %q, want %q", got, tight)
		}
		if got := string(looseTool.Schema()); got != loose {
			t.Errorf("loose schema = %q, want %q", got, loose)
		}
	})
}

// AI-08.1 — construction rules fail through AI-04's sentinels, in the
// documented order.
//
// The name rule is the intersection of what real providers accept, not the
// union: 1 to MaxToolNameLen bytes, a leading ASCII letter or underscore, and
// thereafter ASCII letters, digits, underscore or hyphen. design.md § 3.2
// carries the argument, and it is specific to names rather than to validation
// generally — a tool name is not merely sent, it comes back. The model answers
// with a tool call naming the tool, so a name an adapter had to rewrite
// outbound would need un-rewriting inbound, a bidirectional mapping AI-26 would
// own and AI-30 would have to invert.
//
// Length and shape report *different* classes because V-FAIL-17 makes a rule
// class the kind of thing a rule checks: "shorten it" and "spell it
// differently" are different facts with different fixes. The length case is a
// discovered case appended to doc 0002's AI-08.1 item 3 under the living-graph
// clause; tasks.md records the append.
func TestNewTool_BrokenConstructionRules_FailWithTheDocumentedSentinels(t *testing.T) {
	t.Parallel()

	const okSchema = `{"type":"object"}`

	cases := []struct {
		label    string
		name     string
		schema   string
		want     error
		notWant  []error
		position string
	}{
		{
			label:    "an empty name",
			name:     "",
			schema:   okSchema,
			want:     ai.ErrEmpty,
			notWant:  []error{ai.ErrOutOfRange, ai.ErrMalformed},
			position: "name",
		},
		{
			label:    "a name one byte past the ceiling",
			name:     "a" + strings.Repeat("b", ai.MaxToolNameLen),
			schema:   okSchema,
			want:     ai.ErrOutOfRange,
			notWant:  []error{ai.ErrEmpty, ai.ErrMalformed},
			position: "name",
		},
		{
			label:    "a name containing a dot",
			name:     "github.list_prs",
			schema:   okSchema,
			want:     ai.ErrMalformed,
			notWant:  []error{ai.ErrEmpty, ai.ErrOutOfRange},
			position: "name",
		},
		{
			label:    "a name containing a space",
			name:     "get weather",
			schema:   okSchema,
			want:     ai.ErrMalformed,
			notWant:  []error{ai.ErrEmpty, ai.ErrOutOfRange},
			position: "name",
		},
		{
			label:    "a name that is not ASCII",
			name:     "obtener_clima_ñ",
			schema:   okSchema,
			want:     ai.ErrMalformed,
			notWant:  []error{ai.ErrEmpty, ai.ErrOutOfRange},
			position: "name",
		},
		{
			label:    "a name beginning with a digit",
			name:     "1st_tool",
			schema:   okSchema,
			want:     ai.ErrMalformed,
			notWant:  []error{ai.ErrEmpty, ai.ErrOutOfRange},
			position: "name",
		},
		{
			label:    "a name beginning with a hyphen",
			name:     "-tool",
			schema:   okSchema,
			want:     ai.ErrMalformed,
			notWant:  []error{ai.ErrEmpty, ai.ErrOutOfRange},
			position: "name",
		},
		{
			label:    "empty schema bytes",
			name:     "get_weather",
			schema:   "",
			want:     ai.ErrEmpty,
			notWant:  []error{ai.ErrOutOfRange, ai.ErrMalformed},
			position: "schema",
		},
	}

	for _, tc := range cases {
		t.Run(tc.label, func(t *testing.T) {
			t.Parallel()

			tool, err := ai.NewTool(tc.name, "a description", []byte(tc.schema))
			if err == nil {
				t.Fatalf("NewTool(%q, ...) returned no failure, want one", tc.name)
			}
			if !errors.Is(err, tc.want) {
				t.Errorf("errors.Is(err, %v) = false, want true (err = %v)", tc.want, err)
			}
			for _, other := range tc.notWant {
				if errors.Is(err, other) {
					t.Errorf("errors.Is(err, %v) = true, want false (err = %v)", other, err)
				}
			}

			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
			}
			if got := violation.Path().String(); got != tc.position {
				t.Errorf("violation.Path() = %q, want %q", got, tc.position)
			}

			if got := tool.Name(); got != "" {
				t.Errorf("the returned declaration has name %q, want the zero Tool", got)
			}
			if got := tool.Schema(); len(got) != 0 {
				t.Errorf("the returned declaration has schema %q, want the zero Tool", got)
			}
		})
	}

	t.Run("the first failure in the documented order wins, on every run", func(t *testing.T) {
		t.Parallel()

		// Every rule violated at once: empty name, and empty schema. The
		// documented order is empty name, over-long name, malformed name,
		// empty schema — so the name's emptiness is reported, at "name".
		for range 32 {
			_, err := ai.NewTool("", "", nil)
			var violation *ai.Violation
			if !errors.As(err, &violation) {
				t.Fatalf("errors.As(err, *ai.Violation) = false, want true (err = %v)", err)
			}
			if got := violation.Path().String(); got != "name" {
				t.Fatalf("violation.Path() = %q, want %q — the documented order is not stable", got, "name")
			}
		}

		// A name that is both over-long and malformed reports the length,
		// because the length rule precedes the shape rule.
		_, err := ai.NewTool(strings.Repeat(".", ai.MaxToolNameLen+1), "", []byte(okSchema))
		if !errors.Is(err, ai.ErrOutOfRange) {
			t.Errorf("errors.Is(err, ErrOutOfRange) = false, want true (err = %v)", err)
		}
		if errors.Is(err, ai.ErrMalformed) {
			t.Errorf("errors.Is(err, ErrMalformed) = true, want false — the length rule precedes the shape rule (err = %v)", err)
		}
	})

	t.Run("no offered value reaches the failure", func(t *testing.T) {
		t.Parallel()

		const sentinelName = "CACHICAMA-SENTINEL-NAME-9f2b!"
		const sentinelSchema = "CACHICAMA-SENTINEL-SCHEMA-4d17"

		for _, err := range []error{
			mustFail(t, sentinelName, okSchema),
			mustFail(t, "", sentinelSchema),
		} {
			if strings.Contains(err.Error(), "CACHICAMA-SENTINEL") {
				t.Errorf("Error() = %q, want it to carry none of the offered values", err.Error())
			}
		}
	})
}

// mustFail constructs a declaration expected to fail and returns the failure.
func mustFail(t *testing.T, name, schema string) error {
	t.Helper()

	_, err := ai.NewTool(name, "", []byte(schema))
	if err == nil {
		t.Fatalf("NewTool(%q, ...) returned no failure, want one", name)
	}
	return err
}
