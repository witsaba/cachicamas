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
