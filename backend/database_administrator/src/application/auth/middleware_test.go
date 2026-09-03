// Package auth — middleware_test.go locks the X-Internal-Secret
// verifier contract per spec R-BE-001 / R-MIDDLEWARE-1 / S-BE-010 /
// S-BE-011.
//
// The verifier is a pure function: it takes the header contents and
// the expected secret and returns nil (allowed) or one of three
// sentinels (missing / wrong / allowed). The Echo wrapping lives in
// interfaces/http/auth_handler.go (PR-2 commit that registers the
// routes); the test here exercises the underlying logic so a future
// change to the wire contract (e.g. switch from X-Internal-Secret to
// Bearer JWT) only requires changing the wrapper, not the gate.
package auth

import (
	"errors"
	"strings"
	"testing"
)

const (
	testSecretShort  = "K6y9Qk2vR4mP8sN3"
	testSecretWrong  = "different-secret-here"
	testSecretLonger = "K6y9Qk2vR4mP8sN3X-trailing-bytes"
)

// TestInternalSecret_Valid covers the happy path: a matching
// non-empty header value MUST return nil so the protected handler
// runs.
func TestInternalSecret_Valid(t *testing.T) {
	if err := CheckInternalSecret(testSecretShort, testSecretShort); err != nil {
		t.Fatalf("CheckInternalSecret(%q, %q): unexpected error %v",
			testSecretShort, testSecretShort, err)
	}
}

// TestInternalSecret_Missing covers S-BE-010: an empty header MUST
// return ErrMissingInternalSecret so the handler maps it to 401.
func TestInternalSecret_Missing(t *testing.T) {
	err := CheckInternalSecret("", testSecretShort)
	if err == nil {
		t.Fatal("CheckInternalSecret(\"\", secret): expected ErrMissingInternalSecret, got nil")
	}
	if !errors.Is(err, ErrMissingInternalSecret) {
		t.Errorf("CheckInternalSecret(\"\", secret): error = %v, want errors.Is(ErrMissingInternalSecret)", err)
	}
}

// TestInternalSecret_Wrong covers S-BE-011: a non-empty but wrong
// header MUST return ErrWrongInternalSecret so the handler maps it
// to 401. The error must NOT distinguish "wrong value" from "right
// value at a different time" so an attacker cannot probe.
//
// Note: the empty-header case is covered by TestInternalSecret_Missing
// and is intentionally excluded here (the verifier distinguishes
// missing from wrong for operational logging).
func TestInternalSecret_Wrong(t *testing.T) {
	cases := []string{
		testSecretWrong,
		testSecretLonger,                // different length
		strings.ToUpper(testSecretShort), // same bytes, different case
		"K6y9Qk2vR4mP8sN",               // shorter prefix
		"K6y9Qk2vR4mP8sN3+",              // same length, one byte off
	}
	for _, got := range cases {
		t.Run("got="+got, func(t *testing.T) {
			err := CheckInternalSecret(got, testSecretShort)
			if err == nil {
				t.Fatalf("CheckInternalSecret(%q, secret): expected error, got nil", got)
			}
			if !errors.Is(err, ErrWrongInternalSecret) {
				t.Errorf("CheckInternalSecret(%q, secret): error = %v, want errors.Is(ErrWrongInternalSecret)", got, err)
			}
		})
	}
}

// TestInternalSecret_EmptyExpected covers the misconfiguration
// gate: an empty expected secret MUST return ErrEmptyExpected so the
// handler refuses to start (defense-in-depth: a missing
// AUTH_INTERNAL_SECRET env should be a 500, not a bypass).
func TestInternalSecret_EmptyExpected(t *testing.T) {
	err := CheckInternalSecret(testSecretShort, "")
	if err == nil {
		t.Fatal("CheckInternalSecret(header, \"\"): expected ErrEmptyExpected, got nil")
	}
	if !errors.Is(err, ErrEmptyExpected) {
		t.Errorf("CheckInternalSecret(header, \"\"): error = %v, want errors.Is(ErrEmptyExpected)", err)
	}
}

// TestInternalSecret_DoesNotLeakValue covers a wire-hygiene
// invariant: the error message MUST NOT include the supplied
// expected secret, so a misconfigured log line does not exfiltrate
// the secret. The error message may include the (header) input as
// long as it is bounded (we only check the expected side here).
func TestInternalSecret_DoesNotLeakValue(t *testing.T) {
	err := CheckInternalSecret("anything", testSecretShort)
	if err == nil {
		t.Fatal("CheckInternalSecret: expected error for mismatched value")
	}
	if strings.Contains(err.Error(), testSecretShort) {
		t.Errorf("error message leaks expected secret: %q", err.Error())
	}
}