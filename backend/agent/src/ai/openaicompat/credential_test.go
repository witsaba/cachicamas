package openaicompat

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// TestCredential_NeverRendersRawToken covers S-APC-053 (R-APC-014): the
// credential's default, string, verbose and Go-syntax formatting forms, and
// its serialized form, never contain the raw token.
//
// This is a type-shape opacity assertion only, not a wire-level secrecy
// claim (S-APC-054, S-APC-055 discharge that distinction as a recorded
// review confirmation, tracked separately in this milestone's evidence
// log): the bound on what a wire error body may carry is AI-32.5's, and the
// exhaustive sentinel sweep is AI-36.1's.
func TestCredential_NeverRendersRawToken(t *testing.T) {
	t.Parallel()

	const token = "sk-super-secret-token-value"
	cred := NewCredential(token)

	renderings := map[string]string{
		"%v":  fmt.Sprintf("%v", cred),
		"%s":  fmt.Sprintf("%s", cred),
		"%+v": fmt.Sprintf("%+v", cred),
		"%#v": fmt.Sprintf("%#v", cred),
	}

	jsonBytes, err := json.Marshal(cred)
	if err != nil {
		t.Fatalf("json.Marshal(cred) error = %v, want nil", err)
	}
	renderings["json.Marshal"] = string(jsonBytes)

	for verb, rendering := range renderings {
		if strings.Contains(rendering, token) {
			t.Errorf("rendering via %s leaked the raw token: %q (S-APC-053)", verb, rendering)
		}
	}
}
