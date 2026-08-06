package openaicompat

import (
	"net/url"
	"testing"
)

// TestJoinRequestPath covers R-APC-006's full table (S-APC-024, S-APC-025,
// S-APC-026, S-APC-027): every base shape joins to the expected request
// URL, with no dropped segment and no doubled separator. S-APC-024 and
// S-APC-025 are the trailing-slash and non-trailing-slash pair that proves
// RFC 3986 reference-merging's silent-replacement footgun does not occur
// here — both must retain the base's sub-path in full.
func TestJoinRequestPath(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		base     string
		segments []string
		want     string
	}{
		{
			name:     "trailing-slash base",
			base:     "http://example.invalid/v1/",
			segments: []string{"chat/completions"},
			want:     "http://example.invalid/v1/chat/completions",
		},
		{
			name:     "sub-path base without trailing slash (the RFC 3986 footgun)",
			base:     "http://example.invalid/proxy/openai",
			segments: []string{"chat/completions"},
			want:     "http://example.invalid/proxy/openai/chat/completions",
		},
		{
			name:     "root base without trailing slash",
			base:     "http://example.invalid",
			segments: []string{"chat/completions"},
			want:     "http://example.invalid/chat/completions",
		},
		{
			name:     "doubled interior separators",
			base:     "http://example.invalid/v1",
			segments: []string{"chat//completions"},
			want:     "http://example.invalid/v1/chat/completions",
		},
		{
			name:     "empty relative path, no segments",
			base:     "http://example.invalid/v1",
			segments: nil,
			want:     "http://example.invalid/v1",
		},
		{
			name:     "empty relative path, one empty segment",
			base:     "http://example.invalid/v1",
			segments: []string{""},
			want:     "http://example.invalid/v1",
		},
		{
			name:     "base carrying a query string",
			base:     "http://example.invalid/v1?key=abc",
			segments: []string{"chat/completions"},
			want:     "http://example.invalid/v1/chat/completions?key=abc",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			base, err := url.Parse(tc.base)
			if err != nil {
				t.Fatalf("url.Parse(%q) error = %v", tc.base, err)
			}

			got := joinRequestPath(base, tc.segments...)
			if got.String() != tc.want {
				t.Errorf("joinRequestPath(%q, %v) = %q, want %q (S-APC-024..027)", tc.base, tc.segments, got.String(), tc.want)
			}
		})
	}
}

// TestJoinRequestPath_AbsoluteURLShapedSegmentStaysALiteralPathComponent
// covers S-APC-028: a relative segment shaped like an absolute URL is
// treated as a literal path component of the base and does not redirect
// the request to another host.
func TestJoinRequestPath_AbsoluteURLShapedSegmentStaysALiteralPathComponent(t *testing.T) {
	t.Parallel()

	base, err := url.Parse("http://example.invalid/v1")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}

	got := joinRequestPath(base, "http://evil.example.invalid/x")

	if got.Host != "example.invalid" {
		t.Errorf("joined host = %q, want %q — an absolute-URL-shaped segment must not redirect the request (S-APC-028)", got.Host, "example.invalid")
	}
	if got.Scheme != "http" {
		t.Errorf("joined scheme = %q, want %q (S-APC-028)", got.Scheme, "http")
	}
}

// TestJoinRequestPath_NonDestructiveOnStoredBase covers S-APC-029: one
// stored base used to build two requests to two different relative paths
// does not accumulate either request's segments onto the other, and the
// base itself is never mutated by JoinPath.
func TestJoinRequestPath_NonDestructiveOnStoredBase(t *testing.T) {
	t.Parallel()

	base, err := url.Parse("http://example.invalid/v1")
	if err != nil {
		t.Fatalf("url.Parse() error = %v", err)
	}
	const baseBefore = "http://example.invalid/v1"

	first := joinRequestPath(base, "chat/completions")
	second := joinRequestPath(base, "embeddings")

	const wantFirst = "http://example.invalid/v1/chat/completions"
	const wantSecond = "http://example.invalid/v1/embeddings"

	if first.String() != wantFirst {
		t.Errorf("first joined path = %q, want %q (S-APC-029)", first.String(), wantFirst)
	}
	if second.String() != wantSecond {
		t.Errorf("second joined path = %q, want %q (S-APC-029)", second.String(), wantSecond)
	}
	if base.String() != baseBefore {
		t.Errorf("stored base mutated: now %q, want unchanged %q (S-APC-029)", base.String(), baseBefore)
	}
}
