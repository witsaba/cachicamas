// AI-10.4 item 3's consequence — a JSON syntax check with no transitive I/O.
//
// tool_call.go's argument-bytes rule (V-REQ-17) needs "is this syntactically
// well-formed JSON", and the standard library's answer is
// encoding/json.Valid. That package transitively imports "fmt", which imports
// "os" — confirmed with `go list -deps encoding/json`, which reports "os" and
// "io/fs" in the closure. AI-10.4's no-I/O guard
// (import_boundary_test.go, TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage)
// forbids exactly that for the package request.go lives in, and tool_call.go
// is that package.
//
// This file is the fix: a hand-rolled RFC 8259 syntax scanner, checking
// nothing encoding/json.Valid would not also check, and checking it against
// the same cases tool_call_test.go already carries plus the additional ones
// this file's own internal test adds. It is syntax only, never schema — the
// same boundary ErrMalformed's own GoDoc states — and it decodes nothing: no
// value is ever produced, only a yes/no answer to "is this one complete JSON
// value, optionally padded with whitespace, and nothing else."

package ai

// isWellFormedJSON reports whether data is exactly one syntactically valid
// JSON value (RFC 8259), optionally surrounded by whitespace, with no other
// bytes before or after it.
//
// It matches encoding/json.Valid's contract for this package's purpose:
// object, array, string, number, true, false and null are all accepted at the
// top level — tool_call_test.go pins a bare JSON string as a whole value on
// purpose, because the rule is syntax, not shape.
func isWellFormedJSON(data []byte) bool {
	i := skipJSONSpace(data, 0)
	i, ok := scanJSONValue(data, i)
	if !ok {
		return false
	}
	i = skipJSONSpace(data, i)
	return i == len(data)
}

// jsonSpace is exactly RFC 8259 § 2's four whitespace bytes. Nothing else —
// not a form feed, not a vertical tab — is JSON whitespace.
func isJSONSpace(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

func skipJSONSpace(data []byte, i int) int {
	for i < len(data) && isJSONSpace(data[i]) {
		i++
	}
	return i
}

// scanJSONValue scans one JSON value starting at i and returns the index
// immediately after it, or (i, false) when no value starts there.
func scanJSONValue(data []byte, i int) (int, bool) {
	if i >= len(data) {
		return i, false
	}
	switch c := data[i]; {
	case c == '{':
		return scanJSONObject(data, i)
	case c == '[':
		return scanJSONArray(data, i)
	case c == '"':
		return scanJSONString(data, i)
	case c == '-' || (c >= '0' && c <= '9'):
		return scanJSONNumber(data, i)
	case c == 't':
		return scanJSONLiteral(data, i, "true")
	case c == 'f':
		return scanJSONLiteral(data, i, "false")
	case c == 'n':
		return scanJSONLiteral(data, i, "null")
	default:
		return i, false
	}
}

// scanJSONLiteral matches one of the three fixed keywords the grammar allows.
func scanJSONLiteral(data []byte, i int, literal string) (int, bool) {
	end := i + len(literal)
	if end > len(data) || string(data[i:end]) != literal {
		return i, false
	}
	return end, true
}

// scanJSONObject scans "{" member ("," member)* "}" | "{" "}", assuming
// data[i] == '{'.
func scanJSONObject(data []byte, i int) (int, bool) {
	i++ // consume '{'
	i = skipJSONSpace(data, i)
	if i < len(data) && data[i] == '}' {
		return i + 1, true
	}
	for {
		var ok bool
		if i >= len(data) || data[i] != '"' {
			return i, false
		}
		if i, ok = scanJSONString(data, i); !ok {
			return i, false
		}
		i = skipJSONSpace(data, i)
		if i >= len(data) || data[i] != ':' {
			return i, false
		}
		i = skipJSONSpace(data, i+1)
		if i, ok = scanJSONValue(data, i); !ok {
			return i, false
		}
		i = skipJSONSpace(data, i)
		if i >= len(data) {
			return i, false
		}
		switch data[i] {
		case ',':
			i = skipJSONSpace(data, i+1)
		case '}':
			return i + 1, true
		default:
			return i, false
		}
	}
}

// scanJSONArray scans "[" value ("," value)* "]" | "[" "]", assuming
// data[i] == '['.
func scanJSONArray(data []byte, i int) (int, bool) {
	i++ // consume '['
	i = skipJSONSpace(data, i)
	if i < len(data) && data[i] == ']' {
		return i + 1, true
	}
	for {
		var ok bool
		if i, ok = scanJSONValue(data, i); !ok {
			return i, false
		}
		i = skipJSONSpace(data, i)
		if i >= len(data) {
			return i, false
		}
		switch data[i] {
		case ',':
			i = skipJSONSpace(data, i+1)
		case ']':
			return i + 1, true
		default:
			return i, false
		}
	}
}

// scanJSONString scans one quoted string, assuming data[i] == '"'.
//
// Unescaped control bytes (below 0x20) are rejected, per RFC 8259 § 7. Bytes
// 0x80 and above — UTF-8 continuation and lead bytes of a multi-byte rune —
// fall through to the default case and are consumed one at a time; this is a
// syntax check, not a UTF-8 validator, and encoding/json.Valid does not
// reject malformed UTF-8 inside a string either.
func scanJSONString(data []byte, i int) (int, bool) {
	i++ // consume opening '"'
	for i < len(data) {
		switch c := data[i]; {
		case c == '"':
			return i + 1, true
		case c == '\\':
			next, ok := scanJSONEscape(data, i+1)
			if !ok {
				return i, false
			}
			i = next
		case c < 0x20:
			return i, false
		default:
			i++
		}
	}
	return i, false // unterminated string
}

// scanJSONEscape scans the character after a backslash, assuming i is the
// index immediately following it.
func scanJSONEscape(data []byte, i int) (int, bool) {
	if i >= len(data) {
		return i, false
	}
	switch data[i] {
	case '"', '\\', '/', 'b', 'f', 'n', 'r', 't':
		return i + 1, true
	case 'u':
		if i+4 >= len(data) {
			return i, false
		}
		for j := 1; j <= 4; j++ {
			if !isHexDigit(data[i+j]) {
				return i, false
			}
		}
		return i + 5, true
	default:
		return i, false
	}
}

func isHexDigit(c byte) bool {
	return (c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')
}

// scanJSONNumber scans one number: "-"? int frac? exp?, per RFC 8259 § 6.
func scanJSONNumber(data []byte, i int) (int, bool) {
	start := i
	if i < len(data) && data[i] == '-' {
		i++
	}
	if i >= len(data) || !isDigit(data[i]) {
		return start, false
	}
	if data[i] == '0' {
		i++
	} else {
		i = scanJSONDigits(data, i)
	}
	if i < len(data) && data[i] == '.' {
		i++
		if i >= len(data) || !isDigit(data[i]) {
			return start, false
		}
		i = scanJSONDigits(data, i)
	}
	if i < len(data) && (data[i] == 'e' || data[i] == 'E') {
		i++
		if i < len(data) && (data[i] == '+' || data[i] == '-') {
			i++
		}
		if i >= len(data) || !isDigit(data[i]) {
			return start, false
		}
		i = scanJSONDigits(data, i)
	}
	return i, true
}

func isDigit(c byte) bool { return c >= '0' && c <= '9' }

func scanJSONDigits(data []byte, i int) int {
	for i < len(data) && isDigit(data[i]) {
		i++
	}
	return i
}
