package openaicompat

// Credential is an opaque bearer-token value. See client_test.go and
// credential_test.go for why its raw token is never observed by reading a
// field, and R-APC-014 for why its opacity is a type-shape property, not a
// wire-level secrecy guarantee.
type Credential struct {
	token string
}

// NewCredential builds a Credential from a raw bearer-token value.
func NewCredential(token string) Credential {
	return Credential{token: token}
}

// isEmpty reports whether the credential carries no token.
func (c Credential) isEmpty() bool {
	return c.token == ""
}

// bearer renders the credential's HTTP Authorization header value. This is
// the one place the raw token becomes visible, at the request-construction
// boundary this milestone attaches it at (R-APC-001, R-APC-013).
func (c Credential) bearer() string {
	return "Bearer " + c.token
}

// String never renders the raw token (R-APC-014, S-APC-053): it covers the
// %v, %s and %+v formatting verbs. This is a type-shape opacity property,
// not a wire-level secrecy guarantee — see doc.go and R-APC-014.
func (c Credential) String() string {
	return "<redacted>"
}

// GoString never renders the raw token either (R-APC-014, S-APC-053): it
// covers %#v, which String alone does not — without it, Go's default
// Go-syntax representation of a struct prints every field, including
// unexported ones. There is deliberately no MarshalJSON or MarshalText:
// encoding/json already omits an unexported field with no exported
// counterpart, so json.Marshal is safe by construction.
func (c Credential) GoString() string {
	return "<redacted>"
}
