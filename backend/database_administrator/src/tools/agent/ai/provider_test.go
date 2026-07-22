package ai_test

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// packageDir returns the absolute directory of the calling test source
// file. Lets each test locate sibling files (doc.go, provider.go)
// relative to the package directory rather than the test's working
// directory — robust against `go test ./...`, `go test ./specific`,
// and IDE-driven test runs.
func packageDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller(0) failed")
	}
	return filepath.Dir(file)
}

// TestDocGoParagraph_AI16Guard (T-AI16-008) verifies spec section B.3:
// the AI-16 paragraph MUST exist in doc.go, MUST begin with the AI-16
// marker, MUST reference `provider.go`, and MUST end with the literal
// `. See provider.go.` suffix. Drift fails the test.
//
// This byte-match guard is the canonical in-package assertion for the
// AI-16 paragraph; a relaxed sister check lives in the AST guard in
// agenttest/consumer_test.go (TestModelProviderInterface_SignatureGuard).
func TestDocGoParagraph_AI16Guard(t *testing.T) {
	dir := packageDir(t)
	path := filepath.Join(dir, "doc.go")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	body := string(raw)

	const marker = "// AI-16 paragraph (added after package clause; greedy AI-07 paragraph test does not count)."
	if !strings.Contains(body, marker) {
		t.Fatalf("doc.go is missing the AI-16 paragraph marker %q. "+
			"Add the AI-16 paragraph after the AI-15 paragraph (which ends with 'See toolcall_event.go.').",
			marker)
	}

	const suffix = ". See provider.go."
	trimmed := strings.TrimRight(body, "\n\t ")
	if !strings.HasSuffix(trimmed, suffix) {
		t.Fatalf("doc.go AI-16 paragraph must end with the literal suffix %q (per spec B.3). "+
			"Body tail is %q", suffix, tail(trimmed, 80))
	}
}

// TestProviderGo_ImportsOnlyStdlib (T-AI16-009) verifies spec section C.5.d:
// the new provider.go file (the AI-16 production file) MUST import ONLY
// stdlib (`context`). Mirrors the import-boundary canary
// (TestPackageImportBoundary in import_boundary_test.go) but is scoped to
// the new file's import set: it parses provider.go directly and asserts
// every import path equals the canonical stdlib path `context`.
//
// A forbidden import (openai, anthropic, huggingface, openrouter,
// net/http, encoding/json, cachicamas/... outside tools/agent/ai) fails
// the test.
func TestProviderGo_ImportsOnlyStdlib(t *testing.T) {
	dir := packageDir(t)
	path := filepath.Join(dir, "provider.go")
	parsed, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ParseComments)
	if err != nil {
		t.Fatalf("parse %s: %v "+
			"(provider.go is the AI-16 file; missing indicates the GREEN commit has not landed)",
			path, err)
	}

	seen := map[string]bool{}
	for _, imp := range parsed.Imports {
		seen[strings.Trim(imp.Path.Value, "\"")] = true
	}

	if len(seen) == 0 {
		t.Fatalf("provider.go has no imports; it MUST import stdlib `context` (per spec C.5.d)")
	}

	if !seen["context"] {
		t.Errorf("provider.go must import stdlib `context` (per spec C.5.d); saw: %v", sortedKeys(seen))
	}
	for p := range seen {
		if p != "context" {
			t.Errorf("provider.go must import only stdlib `context` (per spec C.5.a/b/c); got forbidden import %q", p)
		}
	}
}

// tail returns the last n bytes of s, prefixed with "..." if truncated.
// Used in error messages to keep them readable when doc.go grows long.
func tail(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return "..." + s[len(s)-n:]
}

// sortedKeys returns the keys of m in stable, sorted order. Used for
// deterministic error messages.
func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	// Insertion sort (small N; deterministic without importing sort).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j-1] > out[j]; j-- {
			out[j-1], out[j] = out[j], out[j-1]
		}
	}
	return out
}
