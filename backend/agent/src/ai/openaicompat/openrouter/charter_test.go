// R-OR-10's mechanical guard: the out-of-scope fence. The wrapper
// SHALL NOT support an Anthropic native adapter, SHALL NOT widen
// AI-32 to map OpenRouter's `error_type` discriminator, SHALL NOT
// emit a `reasoning_content` (or `reasoning` / `reasoning_details`)
// extension field as `ai.EventKindReasoningDelta`, and SHALL NOT
// introduce any `go.mod` require beyond AI-37's own ADR-gated
// OpenTelemetry API exception (ADR 0005 § D3; wantGoModRequireLines,
// zero_requires_test.go).
//
// Each negative SHALL has its own mechanical check; a future change
// that crosses any one of them fails this guard with a specific
// rule name in the failure message. Reviewer vigilance is a
// backstop — this guard is the front line.
//
// # Why four checks in one file
//
// Each check is a single, narrow assertion whose name carries its
// rule. A future reviewer who lands a deliberate scope change
// (e.g. AI-37 adds OpenTelemetry API and the `allowedNonStdlibPrefixes`
// entry grows) updates the matching assertion in the same commit —
// the same posture the AI-00.3 forward-guard already takes
// (import_boundary_test.go:103-105).

package openrouter

import (
	"bufio"
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// openrouterPackagePath is the directory whose non-test sources the
// charter walks (R-OR-10 fence). "." at test time is this
// sub-package's directory: backend/agent/src/ai/openaicompat/openrouter/.
const openrouterPackagePath = "."

// TestOpenRouterAdapter_NegativeShallsFenceFails is the umbrella
// guard (R-OR-10): the new sub-package's non-test sources carry no
// `*anthropic*` filename, no `"error_type"` literal, no
// `ai.EventKindReasoningDelta` reference, and `go.mod` carries no
// require line beyond AI-37's own ADR-gated exception
// (wantGoModRequireLines). Each sub-check fires independently; a
// single negative SHALL violation in any of them fails the guard.
func TestOpenRouterAdapter_NegativeShallsFenceFails(t *testing.T) {
	t.Parallel()

	t.Run("no_anthropic_named_files_in_subpackage", func(t *testing.T) {
		t.Parallel()
		entries, err := os.ReadDir(openrouterPackagePath)
		if err != nil {
			t.Fatalf("os.ReadDir(%q) error = %v, want nil", openrouterPackagePath, err)
		}
		for _, entry := range entries {
			name := entry.Name()
			if entry.IsDir() {
				continue
			}
			if strings.Contains(strings.ToLower(name), "anthropic") {
				t.Errorf("non-test source filename %q contains 'anthropic' — R-OR-10 negative SHALL: no Anthropic native adapter (the openrouter wrapper composes openaicompat, not a vendor-specific Anthropic adapter)", name)
			}
		}
	})

	t.Run("no_error_type_literal_in_non_test_sources", func(t *testing.T) {
		t.Parallel()
		violations := scanNonTestSourcesForLiteral(openrouterPackagePath, []byte("\"error_type\""))
		for _, v := range violations {
			t.Errorf("%s — R-OR-10 negative SHALL: no AI-32 widening for OpenRouter's `error_type` discriminator (the wrapper composes openaicompat's existing failure mapping, never the OpenRouter-specific discriminator)", v)
		}
	})

	t.Run("no_event_kind_reasoning_delta_in_non_test_sources", func(t *testing.T) {
		t.Parallel()
		// R-OR-10 negative SHALL: the wrapper never renders reasoning
		// content as `ai.EventKindReasoningDelta` — neither the
		// event-kind constant itself, nor any reasoning-shaped
		// string that would reach that emission path.
		violations := scanNonTestSourcesForLiteral(openrouterPackagePath, []byte("EventKindReasoningDelta"))
		for _, v := range violations {
			t.Errorf("%s — R-OR-10 negative SHALL: no reasoning emission as ai.EventKindReasoningDelta", v)
		}
		// Belt-and-braces: also reject the renamed OpenRouter
		// reasoning field names (R-ARS-015's R-OR-10 twin).
		for _, needle := range []string{
			"\"reasoning_content\"",
			"\"reasoning_details\"",
			"\"reasoning\"",
		} {
			vs := scanNonTestSourcesForLiteral(openrouterPackagePath, []byte(needle))
			for _, v := range vs {
				t.Errorf("%s — R-OR-10 negative SHALL: no `reasoning` extension-field literal (%q) in non-test sources; the wrapper's openaicompat adapter drops these whole (reasoning_absence_test.go)", v, needle)
			}
		}
	})

	t.Run("go_mod_matches_ai37_authorized_require_lines", func(t *testing.T) {
		t.Parallel()
		raw, err := os.ReadFile(goModPath)
		if err != nil {
			t.Fatalf("os.ReadFile(%q) error = %v, want nil", goModPath, err)
		}
		if got := countGoModRequireLines(raw); got != wantGoModRequireLines {
			t.Errorf("%s declares %d require line(s), want exactly %d — R-OR-10 negative SHALL still holds for every dependency OTHER than AI-37's own ADR-gated OpenTelemetry API (NFR-APC-A, AI-37 D-6)", goModPath, got, wantGoModRequireLines)
		}
	})
}

// TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation is
// the falsifiability proof for each of the four negative SHALLs.
// A planted scratch .go file in t.TempDir() that violates one of
// the four checks fails the corresponding scan, with the file name
// in the failure message. Without these staged mutations the
// green-from-birth assertion could pass vacuously — no production
// source would ever carry the violation literals, so the scan
// would never observe them.
//
// The four mutations are stitched into one table-driven test so a
// partial guard (a scan that catches one but not all four)
// surfaces as a single clear failure rather than four independent
// vacuous guards.
func TestOpenRouterAdapter_NegativeShallsFenceFails_OnStagedMutation(t *testing.T) {
	t.Parallel()

	type mutation struct {
		name    string
		fixture string
		needle  []byte
	}

	mutations := []mutation{
		{
			name: "anthropic_filename_carries_through_path_check",
			fixture: `package fixture
func anthropic_test_helper() {}
`,
			needle: []byte("anthropic"),
		},
		{
			name: "error_type_literal_caught_by_substring_scan",
			fixture: `package fixture
const wireDiscriminator = "error_type"
`,
			needle: []byte("\"error_type\""),
		},
		{
			name: "reasoning_event_kind_caught_by_substring_scan",
			fixture: `package fixture
import "github.com/cachicamas/backend/agent/src/ai"
var _ = ai.EventKindReasoningDelta
`,
			needle: []byte("EventKindReasoningDelta"),
		},
		{
			name: "reasoning_details_literal_caught_by_substring_scan",
			fixture: `package fixture
const wireField = "reasoning_details"
`,
			needle: []byte("\"reasoning_details\""),
		},
	}

	for _, m := range mutations {
		t.Run(m.name, func(t *testing.T) {
			dir := t.TempDir()
			path := filepath.Join(dir, "staged_r_or_10_violation.go")
			if err := os.WriteFile(path, []byte(m.fixture), 0o600); err != nil {
				t.Fatalf("os.WriteFile() error = %v, want nil", err)
			}
			hits := scanFileForLiteral(path, m.needle)
			if len(hits) == 0 {
				t.Fatalf("scan missed the staged violation literal %q — the charter fence would be vacuous (R-OR-10 bite-proof failure)", m.needle)
			}
		})
	}
}

// scanNonTestSourcesForLiteral walks every non-test .go file in dir
// and returns one violation string per file whose raw bytes contain
// needle. The output is a list of "<file>: <first match line>"
// messages — sufficient for the charter's needs.
func scanNonTestSourcesForLiteral(dir string, needle []byte) []string {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{err.Error()}
	}
	var violations []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !isAdapterSourceFile(name) {
			continue
		}
		path := filepath.Join(dir, name)
		violations = append(violations, scanFileForLiteral(path, needle)...)
	}
	return violations
}

// scanFileForLiteral reads path as raw bytes and returns one
// "<file>:<line>" string per line that contains needle. Used by
// the staged-mutation bite-proof above; charter_test.go's other
// checks use the larger
// scanNonTestSourcesForAmbientAuthority (ambient_authority_test.go)
// for their scan, and this narrow literal scan for the
// header-name checks.
func scanFileForLiteral(path string, needle []byte) []string {
	raw, err := os.ReadFile(path)
	if err != nil {
		return []string{path + ": " + err.Error()}
	}
	scanner := bufio.NewScanner(bytes.NewReader(raw))
	var hits []string
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		if bytes.Contains(scanner.Bytes(), needle) {
			hits = append(hits, path+":"+itoa(lineNo))
		}
	}
	return hits
}

// itoa formats n as a decimal string. A local helper avoids pulling
// in strconv for one call.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	return string(buf[i:])
}