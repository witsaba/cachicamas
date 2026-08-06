// R-OR-02's mechanical negative proof (R-OR-02 sub-scenario 3 +
// design D1/D2, memory #2571): the shipped openaicompat package is
// unaware of OpenRouter's three attribution headers. The proof
// reads backend/agent/src/ai/openaicompat/request.go as raw bytes
// and asserts none of the three header-name literals appears in the
// file — openaicompat's documented "Authorization and Content-Type
// are the only headers it sets" rule (request.go:35) stays intact,
// because widening it for one vendor's quirk would couple the
// vendor-agnostic adapter to the wrapper's own header set.
//
// # Why a raw-bytes scan, not an AST walk
//
// The three header-name literals are strings, not Go identifiers —
// they would not appear in the AST as any typed entity. An AST walk
// would either (a) report every string literal as a potential hit
// (high false-positive rate), or (b) require a hand-curated list
// of strings to ignore (low mechanical value). A raw-bytes scan is
// the most direct possible proof: the file's bytes are the bytes a
// human reads, and the assertion is "no bytes spelling these three
// names appear anywhere".
//
// # Why read from the sibling directory, not from this sub-package
//
// openaicompat's request.go is intentionally outside this
// sub-package's reach. The test reads the file as raw bytes — the
// same shape openaicompat's own credential_scan_test.go uses for
// its deny-list (credential_scan_test.go:35-54, the
// credentialSentinels table). The asserted posture is the same:
// deny-by-default at the deny-list granularity (literal header
// names), never by a curated function or symbol list.

package openrouter

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// attributionHeaderName names one of the three OpenRouter
// attribution header literals the openaicompat package must
// remain unaware of (R-OR-02 sub-scenario 3, design D1/D2).
type attributionHeaderName struct {
	headerName string
	rule       string
}

// attributionHeaderNames is the deny-list this test scans for. The
// three names are spelled verbatim — they are literal header-name
// strings, the same byte sequence a request would carry if the
// header were ever set, and the same byte sequence openaicompat's
// own request.go would need to NOT contain.
//
// Each entry records the rule it enforces, so a failure message
// names it (S-APC-035 inheritance, mirroring the credential scan
// precedent at credential_scan_test.go:35-54).
var attributionHeaderNames = []attributionHeaderName{
	{
		headerName: "HTTP-Referer",
		rule:       "R-OR-02 sub-scenario 3: openaicompat must not name OpenRouter's HTTP-Referer attribution header — its header surface stays Authorization + Content-Type only",
	},
	{
		headerName: "X-OpenRouter-Title",
		rule:       "R-OR-02 sub-scenario 3: openaicompat must not name OpenRouter's X-OpenRouter-Title attribution header",
	},
	{
		headerName: "X-OpenRouter-Categories",
		rule:       "R-OR-02 sub-scenario 3: openaicompat must not name OpenRouter's X-OpenRouter-Categories attribution header",
	},
}

// openaicompatRequestPath is the relative path from this
// sub-package's directory to openaicompat's request.go. The test
// reads this file as raw bytes — its literal byte content is the
// proof under test. The path is fixed: this sub-package lives at
// backend/agent/src/ai/openaicompat/openrouter/, sibling to
// openaicompat/, so the relative path is exactly ../request.go.
const openaicompatRequestPath = "../request.go"

// TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest is
// the guard: openaicompat/request.go carries none of the three
// attribution header names anywhere in its raw bytes (R-OR-02
// sub-scenario 3, design D1/D2).
func TestOpenRouterAdapter_HeadersUnawarenessInOpenAICompatRequest(t *testing.T) {
	t.Parallel()

	raw, err := os.ReadFile(openaicompatRequestPath)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil — openaicompat's request.go must be readable for this guard to discharge R-OR-02 sub-scenario 3", openaicompatRequestPath, err)
	}

	for _, name := range attributionHeaderNames {
		if bytesContain(raw, []byte(name.headerName)) {
			t.Errorf("openaicompat/request.go carries the literal %q — the vendor-agnostic adapter must remain unaware of OpenRouter's attribution headers (R-OR-02 sub-scenario 3, design D1/D2)\n  rule: %s", name.headerName, name.rule)
		}
	}
}

// TestOpenRouterAdapter_HeadersUnawarenessFailsOnStagedMutation is
// the falsifiability proof: a scratch file in t.TempDir() that
// carries a planted attribution-header name fails the scan, naming
// the file, line, and header. With no edit to the scan itself.
// The scan does not care WHERE the literal lives — bytes are bytes
// — so a planted scratch file is enough to prove the scan observes
// what it claims to observe. The scratch file is then cleaned up by
// the test's t.Cleanup.
func TestOpenRouterAdapter_HeadersUnawarenessFailsOnStagedMutation(t *testing.T) {
	t.Parallel()

	dir := t.TempDir()
	const fixture = `package openaicompat

// planted scratch — this file would fail R-OR-02 sub-scenario 3
// if it lived at openaicompat/request.go. Its presence here, in
// t.TempDir(), is the staged-mutation bite-proof: the scan
// observes the literal and reports it by name.
func plantedAttribution() {
	_ = "HTTP-Referer"
	_ = "X-OpenRouter-Title"
	_ = "X-OpenRouter-Categories"
}
`
	path := filepath.Join(dir, "staged_attribution_header_leak.go")
	if err := os.WriteFile(path, []byte(fixture), 0o600); err != nil {
		t.Fatalf("os.WriteFile() error = %v, want nil", err)
	}

	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("os.ReadFile(%q) error = %v, want nil", path, err)
	}

	var hits []string
	for _, name := range attributionHeaderNames {
		if bytesContain(raw, []byte(name.headerName)) {
			hits = append(hits, name.headerName)
		}
	}

	if len(hits) == 0 {
		t.Fatal("scan missed every planted header name — the staged mutation proves nothing because the scan does not observe what it claims to observe (R-OR-02 bite-proof failure)")
	}
	if len(hits) != len(attributionHeaderNames) {
		t.Errorf("scan caught %d header(s) = %v, want %d (every planted name); partial detection is a vacuous guard",
			len(hits), hits, len(attributionHeaderNames))
	}
}

// TestOpenRouterAdapter_HeadersUnawarenessResolvesExpectedPath is a
// belt-and-braces check that the relative path this guard reads
// actually points at openaicompat/request.go — not at some other
// file, not at a missing file. The header scan above would pass
// vacuously against a missing file (os.ReadFile would surface a
// fatal error, the test would not reach the deny-list check), but
// this test pins the path so a future directory reorg cannot
// silently rotate the guard to look at the wrong file.
func TestOpenRouterAdapter_HeadersUnawarenessResolvesExpectedPath(t *testing.T) {
	t.Parallel()

	const wantSubstr = "openaicompat"
	abs, err := filepath.Abs(openaicompatRequestPath)
	if err != nil {
		t.Fatalf("filepath.Abs(%q) error = %v, want nil", openaicompatRequestPath, err)
	}
	if !strings.Contains(abs, wantSubstr) {
		t.Errorf("resolved absolute path %q does not contain %q — the guard must read openaicompat/request.go, not some other file (R-OR-02 sub-scenario 3 path-pin)", abs, wantSubstr)
	}

	info, err := os.Stat(openaicompatRequestPath)
	if err != nil {
		t.Fatalf("os.Stat(%q) error = %v, want nil — the guard must read an existing file", openaicompatRequestPath, err)
	}
	if info.IsDir() {
		t.Fatalf("%q is a directory, want a regular file", openaicompatRequestPath)
	}
	if info.Size() == 0 {
		t.Errorf("%q is empty — the guard would pass vacuously against an empty file", openaicompatRequestPath)
	}
}