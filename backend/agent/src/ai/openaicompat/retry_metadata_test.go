// AI-32.4 — the rate-limit telemetry carrier's RED suite (R-AEM-009,
// S-AEM-036…039). Internal package: RateLimitTelemetry is exported, but
// this file also drives it through the unexported mapResponse so the
// carrier is proven wired into the real cause chain, not merely
// constructible in isolation (design.md D2).
package openaicompat

import (
	"errors"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// TestRetryMetadata_AllThreeHeadersIndividuallyAddressable proves
// S-AEM-036: errors.As retrieves a carrier reporting all three values
// individually, not as one concatenated string.
func TestRetryMetadata_AllThreeHeadersIndividuallyAddressable(t *testing.T) {
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"Content-Type: application/json\n" +
		"X-Ratelimit-Limit-Requests: 3000\n" +
		"X-Ratelimit-Remaining-Requests: 2999\n" +
		"X-Ratelimit-Reset-Requests: 20ms\n" +
		"\n" +
		`{"error":{"type":"rate_limit_error","message":"slow down","param":null,"code":null}}`
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatalf("errors.As(failure, &telemetry) = false, want a *RateLimitTelemetry reachable through the cause chain")
	}
	if telemetry.LimitRequests != "3000" {
		t.Errorf("LimitRequests = %q, want %q", telemetry.LimitRequests, "3000")
	}
	if telemetry.RemainingRequests != "2999" {
		t.Errorf("RemainingRequests = %q, want %q", telemetry.RemainingRequests, "2999")
	}
	if telemetry.ResetRequests != "20ms" {
		t.Errorf("ResetRequests = %q, want %q", telemetry.ResetRequests, "20ms")
	}
}

// TestRetryMetadata_PartialSubsetTolerated proves S-AEM-037: only one
// header present still yields a retrievable carrier, the other two fields
// empty (absent, not an error).
func TestRetryMetadata_PartialSubsetTolerated(t *testing.T) {
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"X-Ratelimit-Remaining-Requests: 0\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatalf("errors.As(failure, &telemetry) = false, want a carrier even with only one header present")
	}
	if telemetry.RemainingRequests != "0" {
		t.Errorf("RemainingRequests = %q, want %q", telemetry.RemainingRequests, "0")
	}
	if telemetry.LimitRequests != "" {
		t.Errorf("LimitRequests = %q, want empty (absent)", telemetry.LimitRequests)
	}
	if telemetry.ResetRequests != "" {
		t.Errorf("ResetRequests = %q, want empty (absent)", telemetry.ResetRequests)
	}
}

// TestRetryMetadata_CredentialHeaderNeverCaptured proves S-AEM-038: a
// credential-bearing header alongside the telemetry headers never reaches
// any rendering of the carrier or the failure — the allowlist, not a
// blocklist, is what keeps it out.
//
// The planted value is deliberately short enough (17 bytes after "sk-")
// that it does NOT match credential_scan_test.go's own
// sk-[A-Za-z0-9_-]{20,} pattern — this file is already internal package
// openaicompat, outside that guard's external-package scope, but the
// literal is kept sub-threshold anyway so a future refactor that widens
// the guard's scope cannot make this file collateral damage.
func TestRetryMetadata_CredentialHeaderNeverCaptured(t *testing.T) {
	const planted = "sk-planted-AEM038"
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"Authorization: Bearer " + planted + "\n" +
		"X-Ratelimit-Remaining-Requests: 5\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	if strings.Contains(failure.Error(), planted) {
		t.Fatalf("Failure.Error() = %q leaks the planted credential", failure.Error())
	}

	// The fixture carries a telemetry header, so the carrier MUST be
	// reachable — a bare `if errors.As` here passed vacuously when a
	// verify-phase mutation probe disabled telemetry capture entirely
	// (verify finding W2). The premise is asserted, then the leak checks.
	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatal("errors.As found no *RateLimitTelemetry in the chain, want one — the fixture carries X-Ratelimit-Remaining-Requests")
	}
	if strings.Contains(telemetry.Error(), planted) {
		t.Errorf("RateLimitTelemetry.Error() = %q leaks the planted credential", telemetry.Error())
	}
	if telemetry.LimitRequests == planted || telemetry.RemainingRequests == planted || telemetry.ResetRequests == planted {
		t.Errorf("planted credential captured into an allowlisted field: %+v", telemetry)
	}
}

// headerCaptureLeak reports whether rendered reproduces headerName or
// headerValue, and which one — the pure check both
// TestCaptureRateLimitTelemetry_NeverReproducesHeaderNameOrValue (AI-36,
// WU-3, S-APC-079) and TestHeaderDiagnostics_NoneCaptureWholeHeaderSet
// (S-APC-080) drive, so the positive-control sub-test can prove the check
// itself is falsifiable without touching a real *testing.T.
func headerCaptureLeak(rendered, headerName, headerValue string) (leaked bool, what string) {
	if strings.Contains(rendered, headerName) {
		return true, "header name"
	}
	if strings.Contains(rendered, headerValue) {
		return true, "header value"
	}
	return false, ""
}

// TestCaptureRateLimitTelemetry_NeverReproducesHeaderNameOrValue covers
// S-APC-079: a response carrying a credential-bearing header (Authorization)
// alongside a sentinel value never reproduces either the header's NAME or
// its VALUE through any rendering of the header-capturing diagnostic — a
// stronger, name-inclusive widening of TestRetryMetadata_CredentialHeaderNeverCaptured
// (S-AEM-038) above, which only ever asserted value-absence. The inline
// positive control proves headerCaptureLeak itself is falsifiable.
func TestCaptureRateLimitTelemetry_NeverReproducesHeaderNameOrValue(t *testing.T) {
	const headerName = "Authorization"
	// Runtime-assembled (never a contiguous "Bearer sk-..." literal in
	// this source file): credential_scan_test.go's widened recursive
	// sweep (AI-36 WU-5) now covers every _test.go file in this tree.
	sentinelValue := "Bearer " + "s" + "k" + "-ai36-header-capture-sentinel-9f2c7a"

	raw := "HTTP/1.1 429 Too Many Requests\n" +
		headerName + ": " + sentinelValue + "\n" +
		"X-Ratelimit-Remaining-Requests: 7\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}

	var telemetry *RateLimitTelemetry
	if !errors.As(failure, &telemetry) {
		t.Fatal("errors.As found no *RateLimitTelemetry in the chain, want one — the fixture carries X-Ratelimit-Remaining-Requests")
	}

	renderings := map[string]string{
		"failure.Error()":                 failure.Error(),
		"telemetry.Error()":               telemetry.Error(),
		"fmt.Sprintf(\"%+v\", telemetry)": fmt.Sprintf("%+v", telemetry),
	}
	for label, rendered := range renderings {
		if leaked, what := headerCaptureLeak(rendered, headerName, sentinelValue); leaked {
			t.Errorf("%s reproduces the %s (want neither name nor value): %q", label, what, rendered)
		}
	}

	if telemetry.LimitRequests == sentinelValue || telemetry.RemainingRequests == sentinelValue || telemetry.ResetRequests == sentinelValue {
		t.Errorf("the credential-bearing header's value was captured into an allowlisted field: %+v", telemetry)
	}

	t.Run("inline positive control: a deliberately-leaking rendering fails the same check", func(t *testing.T) {
		leakingName := fmt.Sprintf("captured header %s present", headerName)
		if leaked, what := headerCaptureLeak(leakingName, headerName, sentinelValue); !leaked || what != "header name" {
			t.Fatalf("headerCaptureLeak(%q) = (%v, %q), want (true, \"header name\") — the check itself must be falsifiable (S-APC-079)", leakingName, leaked, what)
		}
		leakingValue := fmt.Sprintf("captured value=%s", sentinelValue)
		if leaked, what := headerCaptureLeak(leakingValue, headerName, sentinelValue); !leaked || what != "header value" {
			t.Fatalf("headerCaptureLeak(%q) = (%v, %q), want (true, \"header value\") — the check itself must be falsifiable (S-APC-079)", leakingValue, leaked, what)
		}
	})
}

// TestHeaderDiagnostics_NoneCaptureWholeHeaderSet covers S-APC-080: every
// header-capturing diagnostic in the module reads only through an
// explicit admission list, none captures the whole header set, and a
// header newly present on a response stays absent from every diagnostic
// until it is explicitly admitted (design.md AD-5/D-6).
//
// The first sub-test pins the capture surface against the real source:
// it parses this package's production files and requires the set of
// header names they read to equal productionHeaderReadPin exactly, with
// zero whole-header-set iteration. captureRateLimitTelemetry's three
// names are only part of that surface — failure_map.go also reads
// Retry-After and X-Request-Id, and stream.go reads Content-Type.
func TestHeaderDiagnostics_NoneCaptureWholeHeaderSet(t *testing.T) {
	t.Parallel()

	t.Run("production reads exactly the reviewed header names, and never the whole header set", func(t *testing.T) {
		t.Parallel()

		// "." is this package's own directory: go test runs every test
		// binary with its package directory as the working directory.
		sites := scanProductionHeaderReads(t, ".")
		if len(sites) == 0 {
			t.Fatal("scanProductionHeaderReads found no header read at all — the scan itself is broken, since captureRateLimitTelemetry alone reads three (S-APC-080)")
		}

		read := make(map[string]bool, len(productionHeaderReadPin))
		for _, site := range sites {
			switch site.kind {
			case "range":
				t.Errorf("%s:%d iterates the whole header set (range over %s), want reads only through the explicit admission list (S-APC-080): whole-set capture admits every header a provider chooses to send, including a credential-bearing one nobody reviewed", site.file, site.line, site.name)
				continue
			case "Values":
				t.Errorf("%s:%d calls Header.Values(%s), want Header.Get through the explicit admission list (S-APC-080): Values captures every repetition of a header, which is not what this package's reviewed capture surface covers", site.file, site.line, site.name)
				continue
			case "index":
				t.Errorf("%s:%d indexes the header map directly (reading %q), want Header.Get through the explicit admission list (S-APC-080): direct indexing bypasses canonical-key handling and is not the reviewed read shape (JD round 1, finding 4)", site.file, site.line, site.name)
				continue
			case "clone", "copy":
				t.Errorf("%s:%d captures the whole header set (%s of %s), want reads only through the explicit admission list (S-APC-080): a whole-set copy admits every header a provider chooses to send, including a credential-bearing one nobody reviewed (JD round 1, finding 4)", site.file, site.line, site.kind, site.name)
				continue
			}
			if _, admitted := productionHeaderReadPin[site.name]; !admitted {
				t.Errorf("%s:%d reads header %q, which is not in productionHeaderReadPin (S-APC-080). A newly captured header must first be reviewed for credential-bearing content — a credential can ride any header a provider chooses — and only then added to this pin deliberately, together with the assertions proving its value never reaches a rendering", site.file, site.line, site.name)
				continue
			}
			read[site.name] = true
		}

		for name, why := range productionHeaderReadPin {
			if !read[name] {
				t.Errorf("productionHeaderReadPin admits header %q (%s) but no production file in this package reads it any more — drop the entry so the pin keeps naming the real capture surface (S-APC-080)", name, why)
			}
		}
	})

	t.Run("a header newly present on a response, not on the allowlist, is absent from the carrier until admitted", func(t *testing.T) {
		t.Parallel()

		const newHeaderName = "X-Provider-Experimental-Debug-Token"
		const newHeaderValue = "unadmitted-header-sentinel-4d8e1f"

		raw := "HTTP/1.1 429 Too Many Requests\n" +
			newHeaderName + ": " + newHeaderValue + "\n" +
			"X-Ratelimit-Remaining-Requests: 12\n" +
			"\n" +
			"{}"
		failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
		if failure == nil {
			t.Fatal("mapResponse = nil")
		}
		var telemetry *RateLimitTelemetry
		if !errors.As(failure, &telemetry) {
			t.Fatal("errors.As found no *RateLimitTelemetry, want one — the fixture carries X-Ratelimit-Remaining-Requests")
		}
		if telemetry.LimitRequests == newHeaderValue || telemetry.RemainingRequests == newHeaderValue || telemetry.ResetRequests == newHeaderValue {
			t.Errorf("the unadmitted header's value was captured into an allowlisted field: %+v", telemetry)
		}
		if leaked, what := headerCaptureLeak(fmt.Sprintf("%+v", telemetry), newHeaderName, newHeaderValue); leaked {
			t.Errorf("%%+v rendering reproduces the unadmitted header's %s, want it absent until explicitly admitted (S-APC-080)", what)
		}
	})
}

// productionHeaderReadPin is the reviewed, exhaustive set of response
// header names this package's own production files read, each mapped to
// the reason it was admitted. It is the pin
// TestHeaderDiagnostics_NoneCaptureWholeHeaderSet enforces against the
// real source (S-APC-080): every entry was reviewed for credential-bearing
// content before admission, and a read of any other name fails the test.
//
// Scope is this directory's non-test files only — sub-packages such as
// openrouter carry their own capture surface and their own guard.
var productionHeaderReadPin = map[string]string{
	headerRateLimitLimitRequests:     "retry_metadata.go — rate-limit telemetry (R-AEM-009)",
	headerRateLimitRemainingRequests: "retry_metadata.go — rate-limit telemetry (R-AEM-009)",
	headerRateLimitResetRequests:     "retry_metadata.go — rate-limit telemetry (R-AEM-009)",
	"Retry-After":                    "failure_map.go — retry scheduling (R-AEM-001)",
	"X-Request-Id":                   "failure_map.go — request-id capture (R-AEM-007)",
	"Content-Type":                   "stream.go — non-streaming content-type refusal (R-ATS-023)",
}

// headerReadSite is one production site that reads from an HTTP header.
// name carries the resolved header name for a Get/Values call, or the
// receiver's source text for a whole-header-set range.
type headerReadSite struct {
	file string
	line int
	kind string // "Get", "Values", "index", "clone", "copy" or "range"
	name string
}

// scanProductionHeaderReads parses every non-test .go file directly in dir
// and returns each site that READS an HTTP header: a Get or Values call on
// a header-like operand, an index expression into one, a Clone of one, a
// Copy/Clone call taking one as an argument, or a range over one. Writes
// (Set/Add/Del) are deliberately out of scope — this pin covers what comes
// off the wire and can reach a diagnostic, not what this package puts on
// the wire.
//
// Deliberate simplifications keep the guard small and fail-closed:
//
//   - Header names are resolved textually, not by type checking: a string
//     literal argument is unquoted and an identifier is looked up in the
//     package's own string const declarations (that is how the
//     headerRateLimit* names resolve). Anything else is returned verbatim
//     inside "<unresolved: …>", so it fails the pin instead of passing.
//   - Operands are matched syntactically, not through go/types. An
//     expression is header-like when any of these holds: its source text
//     contains "header" case-insensitively (resp.Header, a parameter named
//     header); it is an identifier the file DECLARES with a type whose
//     source text contains "header" (func telemetryFrom(h http.Header) —
//     the name alone says nothing); it is a SELECTOR whose field name was
//     so declared (type diag struct{ h http.Header }, read as d.h.Get);
//     or it is a name ASSIGNED from a non-call expression that is itself
//     header-like, propagated to a fixpoint so multi-hop aliases (h :=
//     resp.Header; g := h) are covered.
//
// What this cannot catch, stated rather than claimed away (AI-36 JD
// round 1 finding 4 and round 2 finding C — earlier versions of this
// comment claimed the matcher over-matches WITHIN the syntactic domain,
// and both rounds falsified that claim with purely syntactic escapes).
// The honest statement is narrower: the matcher over-matches on source
// text (any expression mentioning "header" counts, whatever its type),
// but it is NOT complete even syntactically, and no bounded set of
// syntactic rules would make it so. Known-open shapes:
//
//   - A header set typed through a name that does not spell "header":
//     type hdr = http.Header, or any named type or interface whose
//     declaration this scan does not follow.
//   - A call RESULT aliased to a bland name (h := f(resp), where neither
//     f's name nor its arguments mention a header) — call results are
//     deliberately excluded from alias tracking, because most of them are
//     derived values rather than the set.
//   - A set reached through a struct field, map entry, slice element or
//     channel whose declaring type is in another file's or another
//     package's source, which this per-file scan never reads.
//   - Anything reached via reflection or an interface dispatch.
//
// Closing those requires type-checking the package with go/types,
// deliberately not added here — the guard is a cheap tripwire over the
// shapes a reviewer plausibly writes, not a soundness proof.
// productionHeaderReadPin turns every read the scan DOES see into a
// reviewed admission or a loud failure, and
// TestScanProductionHeaderReads_SeesEvasionShapes pins the covered
// shapes so a regression in coverage fails rather than passes silently.
func scanProductionHeaderReads(t *testing.T, dir string) []headerReadSite {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("os.ReadDir(%q) error = %v, want nil", dir, err)
	}

	type parsedFile struct {
		name string
		src  []byte
		ast  *ast.File
	}
	fset := token.NewFileSet()
	var files []parsedFile
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		src, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			t.Fatalf("os.ReadFile(%q) error = %v, want nil", name, readErr)
		}
		parsed, parseErr := parser.ParseFile(fset, name, src, 0)
		if parseErr != nil {
			t.Fatalf("parser.ParseFile(%q) error = %v, want nil", name, parseErr)
		}
		files = append(files, parsedFile{name: name, src: src, ast: parsed})
	}
	if len(files) == 0 {
		t.Fatalf("no production .go file found in %q — the scan would pass vacuously", dir)
	}

	consts := map[string]string{}
	for _, file := range files {
		for _, decl := range file.ast.Decls {
			gen, ok := decl.(*ast.GenDecl)
			if !ok || gen.Tok != token.CONST {
				continue
			}
			for _, spec := range gen.Specs {
				value, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for i, ident := range value.Names {
					if i >= len(value.Values) {
						continue
					}
					if text, ok := stringLiteral(value.Values[i]); ok {
						consts[ident.Name] = text
					}
				}
			}
		}
	}

	var sites []headerReadSite
	for _, file := range files {
		source := func(node ast.Node) string {
			return string(file.src[fset.Position(node.Pos()).Offset:fset.Position(node.End()).Offset])
		}

		// Pass 1 — collect the file's header-like names (JD round 1,
		// finding 4; widened by JD round 2, finding C): names DECLARED with
		// a type whose source text contains "header" (func params, results,
		// receivers, struct fields, var specs), and names ASSIGNED from a
		// non-call expression that is itself header-like. A call result is
		// deliberately excluded from alias tracking: derived values
		// (header.Get returns a string) are not the set, and a whole-set
		// copy through a call (Clone, maps.Copy/maps.Clone) is recorded at
		// the call itself below.
		//
		// The pass runs to a FIXPOINT (JD round 2, finding C): marking is
		// driven by headerLike rather than by raw source text, so an alias
		// of an alias (h := resp.Header; g := h) is marked on the sweep
		// after the one that marked its source. A single sweep saw only
		// the first hop.
		headerTyped := map[string]bool{}
		containsHeader := func(node ast.Node) bool {
			return strings.Contains(strings.ToLower(source(node)), "header")
		}
		markIdent := func(expr ast.Expr) {
			if ident, ok := expr.(*ast.Ident); ok && ident.Name != "_" {
				headerTyped[ident.Name] = true
			}
		}
		// headerLike reports whether expr denotes the header set: its own
		// source text mentions "header", it is an identifier already
		// marked, or it is a SELECTOR whose field name is already marked —
		// the struct-field-qualified read d.h.Get(…), where d.h is a
		// SelectorExpr and neither its source text nor its shape is an
		// identifier (JD round 2, finding C).
		headerLike := func(expr ast.Expr) bool {
			if containsHeader(expr) {
				return true
			}
			switch typed := expr.(type) {
			case *ast.Ident:
				return headerTyped[typed.Name]
			case *ast.SelectorExpr:
				return headerTyped[typed.Sel.Name]
			}
			return false
		}
		for marked := -1; marked != len(headerTyped); {
			marked = len(headerTyped)
			ast.Inspect(file.ast, func(node ast.Node) bool {
				switch typed := node.(type) {
				case *ast.Field:
					if typed.Type != nil && containsHeader(typed.Type) {
						for _, name := range typed.Names {
							markIdent(name)
						}
					}
				case *ast.ValueSpec:
					if typed.Type != nil && containsHeader(typed.Type) {
						for _, name := range typed.Names {
							markIdent(name)
						}
					}
					for i, value := range typed.Values {
						if _, isCall := value.(*ast.CallExpr); !isCall && headerLike(value) && i < len(typed.Names) {
							markIdent(typed.Names[i])
						}
					}
				case *ast.AssignStmt:
					for i, lhs := range typed.Lhs {
						rhs := typed.Rhs[0]
						if len(typed.Rhs) == len(typed.Lhs) {
							rhs = typed.Rhs[i]
						}
						if _, isCall := rhs.(*ast.CallExpr); !isCall && headerLike(rhs) {
							markIdent(lhs)
						}
					}
				}
				return true
			})
		}

		// Pass 2 — record every read/capture site against those operands.
		ast.Inspect(file.ast, func(node ast.Node) bool {
			switch typed := node.(type) {
			case *ast.CallExpr:
				selector, ok := typed.Fun.(*ast.SelectorExpr)
				if !ok {
					return true
				}
				switch selector.Sel.Name {
				case "Get", "Values":
					if !headerLike(selector.X) {
						return true
					}
					name := "<unresolved: no single argument>"
					if len(typed.Args) == 1 {
						name = resolveHeaderName(typed.Args[0], consts, source)
					}
					sites = append(sites, headerReadSite{
						file: file.name,
						line: fset.Position(typed.Pos()).Line,
						kind: selector.Sel.Name,
						name: name,
					})
				case "Clone":
					// Whole-set capture through the method form
					// (resp.Header.Clone()).
					if headerLike(selector.X) {
						sites = append(sites, headerReadSite{
							file: file.name,
							line: fset.Position(typed.Pos()).Line,
							kind: "clone",
							name: source(selector.X),
						})
					}
				}
				// Whole-set capture through a copying function taking the
				// set as an argument (maps.Copy(dst, resp.Header),
				// maps.Clone(resp.Header)). The callee name is matched, not
				// the maps package specifically — any Copy/Clone taking a
				// header-like argument is a capture worth surfacing.
				if selector.Sel.Name == "Copy" || selector.Sel.Name == "Clone" {
					for _, arg := range typed.Args {
						if headerLike(arg) {
							sites = append(sites, headerReadSite{
								file: file.name,
								line: fset.Position(typed.Pos()).Line,
								kind: "copy",
								name: source(arg),
							})
							break
						}
					}
				}
			case *ast.IndexExpr:
				// Direct map indexing (resp.Header["Authorization"]) reads
				// a header value with no Get call at all.
				if headerLike(typed.X) {
					sites = append(sites, headerReadSite{
						file: file.name,
						line: fset.Position(typed.Pos()).Line,
						kind: "index",
						name: resolveHeaderName(typed.Index, consts, source),
					})
				}
			case *ast.RangeStmt:
				if !headerLike(typed.X) {
					return true
				}
				sites = append(sites, headerReadSite{
					file: file.name,
					line: fset.Position(typed.Pos()).Line,
					kind: "range",
					name: source(typed.X),
				})
			}
			return true
		})
	}
	return sites
}

// resolveHeaderName resolves a Get/Values argument to the header name it
// reads: a string literal directly, an identifier through the package's
// own string consts. Anything else comes back wrapped in "<unresolved: …>"
// so the pin rejects it rather than silently admitting it.
func resolveHeaderName(arg ast.Expr, consts map[string]string, source func(ast.Node) string) string {
	if text, ok := stringLiteral(arg); ok {
		return text
	}
	if ident, ok := arg.(*ast.Ident); ok {
		if text, known := consts[ident.Name]; known {
			return text
		}
	}
	return "<unresolved: " + source(arg) + ">"
}

// stringLiteral reports the unquoted value of expr when it is an
// interpreted string literal.
func stringLiteral(expr ast.Expr) (string, bool) {
	lit, ok := expr.(*ast.BasicLit)
	if !ok || lit.Kind != token.STRING {
		return "", false
	}
	text, err := strconv.Unquote(lit.Value)
	if err != nil {
		return "", false
	}
	return text, true
}

// TestScanProductionHeaderReads_SeesEvasionShapes is the judgment-day
// round-1 finding-4 pin: the pre-fix scanner under-matched — a Get through
// a receiver whose source text lacks "header" (func telemetryFrom(h
// http.Header)), a direct index read (resp.Header["Authorization"]), a
// whole-set Clone, and a whole-set maps.Copy were all invisible, while the
// doc comment claimed the guard over-matches. Round 2's finding C added
// two more purely syntactic escapes that survived that fix: a read
// qualified through a struct FIELD declared http.Header (d.h.Get, whose
// operand is a SelectorExpr, not an identifier) and a TWO-HOP alias (h :=
// resp.Header; g := h, where only the first hop was marked). Each shape
// is planted in a synthetic production file inside a temp dir and must
// come back as a recorded site; a shape the scanner cannot see is a
// silent hole in S-APC-080's "never the whole header set" assertion.
func TestScanProductionHeaderReads_SeesEvasionShapes(t *testing.T) {
	t.Parallel()

	const synthetic = `package synthetic

import (
	"maps"
	"net/http"
)

// telemetryFrom reads through a receiver whose source text lacks "header":
// only its DECLARED TYPE names http.Header.
func telemetryFrom(h http.Header) string {
	return h.Get("Authorization")
}

// indexRead reaches the header value without calling Get at all.
func indexRead(resp *http.Response) []string {
	return resp.Header["Authorization"]
}

// wholeSetClone captures the entire header set through the Clone method.
func wholeSetClone(resp *http.Response) http.Header {
	return resp.Header.Clone()
}

// wholeSetCopy captures the entire header set through maps.Copy.
func wholeSetCopy(resp *http.Response, dst map[string][]string) {
	maps.Copy(dst, resp.Header)
}

// aliasedRange iterates the whole set through a bland alias.
func aliasedRange(resp *http.Response) int {
	hs := resp.Header
	n := 0
	for range hs {
		n++
	}
	return n
}

// diag holds the set in a struct FIELD, so the read below qualifies
// through a selector (d.h) rather than a bare identifier.
type diag struct{ h http.Header }

// fieldQualifiedRead reads through that field.
func (d diag) fieldQualifiedRead() string {
	return d.h.Get("X-Request-Id")
}

// twoHopAliasRange iterates the whole set through a SECOND-generation
// alias: g is assigned from h, whose own source text says nothing about a
// header.
func twoHopAliasRange(resp *http.Response) int {
	h := resp.Header
	g := h
	n := 0
	for range g {
		n++
	}
	return n
}
`
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "synthetic.go"), []byte(synthetic), 0o644); err != nil {
		t.Fatalf("os.WriteFile(synthetic.go) error = %v, want nil", err)
	}

	sites := scanProductionHeaderReads(t, dir)

	type wantSite struct {
		kind string
		name string // "" means any
		why  string
	}
	wants := []wantSite{
		{kind: "Get", name: "Authorization", why: "a Get through a receiver named h, declared http.Header — receiver-name matching alone cannot see it"},
		{kind: "index", name: "Authorization", why: `resp.Header["Authorization"] reads a header value with no Get call at all`},
		{kind: "clone", why: "resp.Header.Clone() captures the whole header set"},
		{kind: "copy", why: "maps.Copy(dst, resp.Header) captures the whole header set"},
		{kind: "range", name: "hs", why: "a range over an alias assigned from resp.Header iterates the whole set"},
		{kind: "Get", name: "X-Request-Id", why: "a Get qualified through a struct field declared http.Header — the operand is the selector d.h, not a bare identifier (JD round 2, finding C)"},
		{kind: "range", name: "g", why: "a range over a two-hop alias (h := resp.Header; g := h) iterates the whole set; alias marking must propagate transitively (JD round 2, finding C)"},
	}
	for _, want := range wants {
		found := false
		for _, site := range sites {
			if site.kind == want.kind && (want.name == "" || site.name == want.name) {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("scanProductionHeaderReads missed the %q shape (name %q): %s — an invisible shape is a silent hole in S-APC-080's guard (JD round 1, finding 4); got sites: %+v", want.kind, want.name, want.why, sites)
		}
	}
}

// TestRetryMetadata_FailureErrorTextIsFixedAndClean proves S-AEM-039:
// with full telemetry attached, Failure.Error() equals exactly
// "provider failure: rate_limit" — no header name or value included. Exact
// equality is the strongest form of "contains no header name or value":
// any leaked byte would already break it.
func TestRetryMetadata_FailureErrorTextIsFixedAndClean(t *testing.T) {
	raw := "HTTP/1.1 429 Too Many Requests\n" +
		"X-Ratelimit-Limit-Requests: 3000\n" +
		"X-Ratelimit-Remaining-Requests: 0\n" +
		"X-Ratelimit-Reset-Requests: 20s\n" +
		"\n" +
		"{}"
	failure := mapResponse(mustParseResponse(t, raw), observedAtFixed)
	if failure == nil {
		t.Fatal("mapResponse = nil")
	}
	const want = "provider failure: rate_limit"
	if got := failure.Error(); got != want {
		t.Fatalf("Failure.Error() = %q, want exactly %q", got, want)
	}
}
