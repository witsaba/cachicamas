// Internal tests for json_syntax.go — AI-10.4 item 3's consequence.
//
// Internal by necessity rather than preference: isWellFormedJSON is
// unexported, and tool_call_test.go already exercises it indirectly through
// ai.NewToolCall for the cases that contract cares about. This file tests the
// scanner itself, directly, against the full RFC 8259 grammar it implements —
// broader coverage than any one caller needs, because a syntax checker with a
// silent gap in one production (numbers, escapes, nesting) is a correctness
// bug waiting for the byte sequence that finds it.

package ai

import "testing"

// TestIsWellFormedJSON_RFC8259Grammar_MatchesEncodingJSONValid pins
// isWellFormedJSON's behavior against every production of RFC 8259: object,
// array, string (plain, escaped, unicode-escaped), number (integer,
// fractional, exponent, negative), and the three literals — plus the
// malformed counterpart of each, and the two structural cases tool_call.go's
// own contract depends on: leading/trailing whitespace is padding, and
// trailing non-whitespace garbage after a complete value is rejected.
func TestIsWellFormedJSON_RFC8259Grammar_MatchesEncodingJSONValid(t *testing.T) {
	t.Parallel()

	cases := []struct {
		name string
		data string
		want bool
	}{
		// Objects.
		{"empty object", `{}`, true},
		{"object with one member", `{"a":1}`, true},
		{"object with several members", `{"a":1,"b":"two","c":[3,4]}`, true},
		{"object with internal whitespace", "{ \"a\" : 1 , \"b\" : 2 }", true},
		{"nested objects", `{"a":{"b":{"c":1}}}`, true},
		{"object missing a closing brace", `{"a":1`, false},
		{"object with a trailing comma", `{"a":1,}`, false},
		{"object with an unquoted key", `{a:1}`, false},
		{"object missing a colon", `{"a" 1}`, false},
		{"object with two values back to back", `{"a":1}{"b":2}`, false},

		// Arrays.
		{"empty array", `[]`, true},
		{"array of numbers", `[1,2,3]`, true},
		{"nested arrays", `[[1,2],[3,4]]`, true},
		{"array with mixed kinds", `[1,"two",true,null,{"k":"v"}]`, true},
		{"array missing a closing bracket", `[1,2`, false},
		{"array with a trailing comma", `[1,2,]`, false},
		{"array with two commas", `[1,,2]`, false},

		// Strings.
		{"empty string", `""`, true},
		{"plain string", `"hello"`, true},
		{"string with a simple escape", `"line\nbreak"`, true},
		{"string with an escaped quote", `"a \"quoted\" word"`, true},
		{"string with an escaped backslash", `"back\\slash"`, true},
		{"string with a unicode escape", `"café"`, true},
		{"string with raw non-ASCII bytes", "\"café\"", true},
		{"unterminated string", `"unterminated`, false},
		{"string with an unrecognised escape", `"bad\qescape"`, false},
		{"string with a short unicode escape", `"\u12"`, false},
		{"string with a raw control byte", "\"a\tb\"", false},

		// Numbers.
		{"zero", `0`, true},
		{"a positive integer", `42`, true},
		{"a negative integer", `-42`, true},
		{"a fraction", `3.14`, true},
		{"a negative fraction", `-0.5`, true},
		{"an exponent", `1e10`, true},
		{"a negative exponent", `1E-10`, true},
		{"a fraction with an exponent", `6.02e23`, true},
		{"a number with a leading zero", `01`, false},
		{"a bare minus sign", `-`, false},
		{"a bare decimal point", `1.`, false},
		{"a fraction with no digits", `1.e5`, false},
		{"an exponent with no digits", `1e`, false},

		// Literals.
		{"true", `true`, true},
		{"false", `false`, true},
		{"null", `null`, true},
		{"a misspelled literal", `tru`, false},
		{"an unrecognised bareword", `undefined`, false},

		// The two structural rules the request path cares about most.
		{"whitespace-padded value", "  \t\n {\"a\":1} \r\n ", true},
		{"whitespace only, no value", "  \t\n ", false},
		{"empty input", ``, false},
		{"a bare string is a whole JSON value", `"a bare string is a whole JSON value"`, true},
		{"trailing garbage after a complete value", `{"a":1} garbage`, false},
		{"leading garbage before a value", `garbage {"a":1}`, false},

		// tool_call_test.go's own fixtures, pinned here too so a regression in
		// this file is caught at the unit that owns it, not only at the
		// contract two layers up.
		{"tool_call_test.go: a truncated object", `{"path":`, false},
		{"tool_call_test.go: not JSON at all", `path=/etc/hosts`, false},
		{"tool_call_test.go: two values back to back", `{"a":1}{"b":2}`, false},
		{"tool_call_test.go: a well-formed call's arguments", `{"path":"/etc/hosts"}`, true},
		{"tool_call_test.go: a schema-mismatched but syntactically valid object", `{"wildly":"wrong","for":["every","tool"]}`, true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			if got := isWellFormedJSON([]byte(tc.data)); got != tc.want {
				t.Errorf("isWellFormedJSON(%q) = %t, want %t", tc.data, got, tc.want)
			}
		})
	}
}
