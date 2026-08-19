// AG-17.2 — strict-TDD tests for token accounting's pure, package-
// internal surfaces (R-CTX-006..010, S-CTX-012, S-CTX-016..018, bite
// S-CTX-013).
//
// This file is package-internal (NFR-CTX-001's own carve-out, the
// AG-16 cost_usage_test.go precedent, archive design.md:176): the
// estimate's own table and the resolver's three-state table are pure
// functions with no external surface. Every claim about what a
// STRATEGY observes is asserted externally, in context_strategy_test.go,
// through a recording strategy.
package agent

import (
	"context"
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"reflect"
	"strings"
	"testing"
	"unicode/utf8"

	"github.com/cachicamas/backend/agent/src/agenttest"
	"github.com/cachicamas/backend/agent/src/ai"
)

// mustEstimateText builds one constructed text message for this file's
// estimate tests.
func mustEstimateText(t *testing.T, role ai.Role, text string) ai.Message {
	t.Helper()
	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText(%q) returned %v, want no failure", text, err)
	}
	msg, err := ai.NewMessage(role, part)
	if err != nil {
		t.Fatalf("ai.NewMessage returned %v, want no failure", err)
	}
	return msg
}

// mustEstimateRequest builds a constructed request for this file's
// estimate tests.
func mustEstimateRequest(t *testing.T, messages []ai.Message, opts ...ai.RequestOption) ai.Request {
	t.Helper()
	req, err := ai.NewRequest("estimate-test-model", messages, opts...)
	if err != nil {
		t.Fatalf("ai.NewRequest returned %v, want no failure", err)
	}
	return req
}

// TestTokenSource_StringRendersDistinctly — part of S-CTX-014 (task
// 1.8). The three provenance states render to three distinct,
// non-empty strings (ai/usage.go:72-77's log-line argument).
func TestTokenSource_StringRendersDistinctly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		source TokenSource
		want   string
	}{
		{"zero value renders unavailable", TokenSourceUnavailable, "unavailable"},
		{"reported", TokenSourceReported, "reported"},
		{"estimated", TokenSourceEstimated, "estimated"},
	}

	seen := map[string]bool{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.source.String()
			if got != tt.want {
				t.Errorf("TokenSource(%d).String() = %q, want %q", tt.source, got, tt.want)
			}
			seen[got] = true
		})
	}
	if len(seen) != 3 {
		t.Errorf("the three TokenSource values rendered %d distinct string(s), want 3: %v", len(seen), seen)
	}
}

// --- Phase 2 -- the estimate, in isolation (R-CTX-009, R-CTX-010) --------

// wantEstimate independently computes the formula
// ceil(B/D) + M*K over a caller-supplied byte total and message count,
// using the SAME exported constants estimateTokens documents, so a
// constant change stays self-consistent without this test silently
// drifting from it.
func wantEstimate(bytesTotal, messageCount int64) int64 {
	return (bytesTotal+estimateBytesPerToken-1)/estimateBytesPerToken + estimateTokensPerMessage*messageCount
}

// TestEstimateTokens_TableDriven — S-CTX-016, the computation half.
// Each row's figure equals the formula applied to that row's OWN byte
// total and message count, computed independently of estimateTokens
// itself (DD5's exact accessor walk: system instruction segments,
// message content -- text/tool-call/tool-result/reasoning -- and tool
// schemas).
func TestEstimateTokens_TableDriven(t *testing.T) {
	t.Parallel()

	t.Run("pure ASCII", func(t *testing.T) {
		t.Parallel()
		text := "hello there, this is a plain ascii message for the estimate table."
		req := mustEstimateRequest(t, []ai.Message{mustEstimateText(t, ai.RoleUser, text)})

		want := wantEstimate(int64(len(text)), 1)
		if got := estimateTokens(req); got != want {
			t.Errorf("estimateTokens(ascii) = %d, want %d", got, want)
		}
	})

	t.Run("CJK", func(t *testing.T) {
		t.Parallel()
		text := "你好世界，这是一段中文文本，用来测试字节计数而不是符文计数。"
		req := mustEstimateRequest(t, []ai.Message{mustEstimateText(t, ai.RoleUser, text)})

		byteTotal := int64(len(text))
		want := wantEstimate(byteTotal, 1)
		got := estimateTokens(req)
		if got != want {
			t.Errorf("estimateTokens(cjk) = %d, want %d", got, want)
		}

		// The CJK row's figure MUST exceed a rune-count equivalent
		// (S-CTX-016): counting runes under-counts a CJK ideograph
		// (~3 bytes, ~1 token) by roughly fourfold.
		runeCount := int64(utf8.RuneCountInString(text))
		runeBasedEstimate := wantEstimate(runeCount, 1)
		if got <= runeBasedEstimate {
			t.Errorf("byte-based estimate = %d, want it to EXCEED the rune-count equivalent = %d (bytes, not runes, is the whole point of the unit choice)", got, runeBasedEstimate)
		}
	})

	t.Run("empty transcript (the minimal valid single message)", func(t *testing.T) {
		t.Parallel()
		// ai.NewRequest requires at least one message (rule 2), so a
		// literally empty message list cannot be constructed at all --
		// this row is therefore the floor: the smallest valid
		// transcript, no system instruction, no tools.
		text := "hi"
		req := mustEstimateRequest(t, []ai.Message{mustEstimateText(t, ai.RoleUser, text)})

		want := wantEstimate(int64(len(text)), 1)
		if got := estimateTokens(req); got != want {
			t.Errorf("estimateTokens(empty transcript floor) = %d, want %d", got, want)
		}
	})

	t.Run("tool-schema-bearing", func(t *testing.T) {
		t.Parallel()
		text := "what is the weather in this city?"
		toolOne, err := ai.NewTool("get_weather", "Reports current weather for a city.", []byte(`{"type":"object","properties":{"city":{"type":"string"}}}`))
		if err != nil {
			t.Fatalf("ai.NewTool returned %v, want no failure", err)
		}
		toolTwo, err := ai.NewTool("get_forecast", "Reports a multi-day forecast.", []byte(`{"type":"object","properties":{"days":{"type":"integer"}}}`))
		if err != nil {
			t.Fatalf("ai.NewTool returned %v, want no failure", err)
		}
		toolSet, err := ai.NewToolSet(toolOne, toolTwo)
		if err != nil {
			t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
		}

		req := mustEstimateRequest(t, []ai.Message{mustEstimateText(t, ai.RoleUser, text)}, ai.WithTools(toolSet))

		byteTotal := int64(len(text))
		for _, tool := range toolSet.Tools() {
			byteTotal += int64(len(tool.Name()) + len(tool.Description()) + len(tool.Schema()))
		}
		want := wantEstimate(byteTotal, 1)
		if got := estimateTokens(req); got != want {
			t.Errorf("estimateTokens(tool-schema-bearing) = %d, want %d -- tool schemas MUST be counted (name+description+schema)", got, want)
		}
	})

	t.Run("system-instruction-bearing", func(t *testing.T) {
		t.Parallel()
		text := "hello"
		systemText := "You are a careful, terse assistant for the estimate table test."
		system, err := ai.NewSystemText(systemText)
		if err != nil {
			t.Fatalf("ai.NewSystemText returned %v, want no failure", err)
		}

		req := mustEstimateRequest(t, []ai.Message{mustEstimateText(t, ai.RoleUser, text)}, ai.WithSystemInstruction(system))

		byteTotal := int64(len(text) + len(systemText))
		want := wantEstimate(byteTotal, 1)
		if got := estimateTokens(req); got != want {
			t.Errorf("estimateTokens(system-instruction-bearing) = %d, want %d", got, want)
		}
	})

	t.Run("mixed content parts -- text, tool-call, tool-result, reasoning", func(t *testing.T) {
		t.Parallel()
		text := "let me check the weather for you."
		callID, toolName, args := "call-est-1", "get_weather", []byte(`{"city":"Bogota"}`)
		resultContent := "22C, partly cloudy"
		reasoningText := "the user asked about weather, so I should call get_weather."

		textPart, err := ai.NewText(text)
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}
		toolCallPart, err := ai.NewToolCall(callID, toolName, args)
		if err != nil {
			t.Fatalf("ai.NewToolCall returned %v, want no failure", err)
		}
		assistantMsg, err := ai.NewMessage(ai.RoleAssistant, textPart, toolCallPart)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		toolResultPart, err := ai.NewToolResult(callID, resultContent)
		if err != nil {
			t.Fatalf("ai.NewToolResult returned %v, want no failure", err)
		}
		toolMsg, err := ai.NewMessage(ai.RoleTool, toolResultPart)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		reasoningPart, err := ai.NewReasoning(reasoningText, nil)
		if err != nil {
			t.Fatalf("ai.NewReasoning returned %v, want no failure", err)
		}
		reasoningMsg, err := ai.NewMessage(ai.RoleAssistant, reasoningPart)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		messages := []ai.Message{assistantMsg, toolMsg, reasoningMsg}
		req := mustEstimateRequest(t, messages)

		byteTotal := int64(len(text) + len(toolName) + len(args) + len(resultContent) + len(reasoningText))
		want := wantEstimate(byteTotal, int64(len(messages)))
		if got := estimateTokens(req); got != want {
			t.Errorf("estimateTokens(mixed content) = %d, want %d -- every part kind's accessor walk must be exercised", got, want)
		}
	})
}

// TestEstimateTokens_Determinism — S-CTX-018. Computed twice over the
// same request value, and once more over an independently
// re-constructed EQUAL request, all three figures are identical.
func TestEstimateTokens_Determinism(t *testing.T) {
	t.Parallel()

	buildReq := func(t *testing.T) ai.Request {
		t.Helper()
		return mustEstimateRequest(t, []ai.Message{mustEstimateText(t, ai.RoleUser, "determinism check, some prose here.")})
	}

	req := buildReq(t)
	first := estimateTokens(req)
	second := estimateTokens(req)
	if first != second {
		t.Fatalf("estimateTokens computed twice over the identical request value returned %d then %d, want identical (R-CTX-010)", first, second)
	}

	reconstructed := buildReq(t)
	third := estimateTokens(reconstructed)
	if third != first {
		t.Errorf("estimateTokens over an independently re-constructed equal request = %d, want %d (identical to the first)", third, first)
	}
}

// TestEstimateTokens_NoClockNoIO_SourceScan — R-CTX-010's own
// correction: ambient_authority_test.go's forbidden set is os,
// os/exec, syscall, io/ioutil -- it does NOT forbid "time". Purity is
// therefore asserted directly by reading this file's own import list,
// never assumed from that guard.
func TestEstimateTokens_NoClockNoIO_SourceScan(t *testing.T) {
	t.Parallel()

	data, err := os.ReadFile("token_accounting.go")
	if err != nil {
		t.Fatalf("os.ReadFile(%q) returned %v, want no error", "token_accounting.go", err)
	}
	content := string(data)

	for _, forbidden := range []string{`"time"`, `"math/rand"`, `"crypto/rand"`, `"net"`, `"os"`, `"os/exec"`} {
		if strings.Contains(content, forbidden) {
			t.Errorf("token_accounting.go imports %s -- the estimate MUST be a pure function: no clock, no environment, no randomness, no I/O", forbidden)
		}
	}
}

// TestEstimateTokens_NeverConvertedToTokenCount — S-CTX-017, the
// mechanical half. An estimate is a Layer 2 approximation; an
// ai.TokenCount's presence bit means "Layer 1 REPORTED it". Nothing
// may launder the former into the latter.
//
// Two independent locks, because either alone is defeatable:
//
//  1. estimateTokens' own result type is a bare int64 -- never
//     ai.TokenCount -- so the conversion cannot happen implicitly at
//     the source.
//  2. NO non-test file in this package calls ai.Tokens, the ONLY
//     exported constructor of an ai.TokenCount (ai/usage.go:53-55;
//     the struct's fields are unexported, so a composite literal is
//     impossible from outside package ai). Layer 2 therefore cannot
//     mint a TokenCount at all, from an estimate or from anything
//     else.
//
// Lock 2 is an AST walk over every production file in this directory,
// not a grep: it ignores comments (cost_usage.go names ai.Tokens(0)
// in prose) and it covers files that do not exist yet, which is the
// gap a byte-unchanged diff check over a fixed file list cannot close.
func TestEstimateTokens_NeverConvertedToTokenCount(t *testing.T) {
	t.Parallel()

	// Lock 1 -- the estimate's own result type.
	got := reflect.TypeOf(estimateTokens).Out(0)
	if want := reflect.TypeOf(int64(0)); got != want {
		t.Errorf("estimateTokens returns %v, want %v -- an estimate MUST NOT be typed as a counted quantity", got, want)
	}
	if got == reflect.TypeOf(ai.TokenCount{}) {
		t.Errorf("estimateTokens returns ai.TokenCount -- an estimate is not a reported count (R-CTX-008)")
	}

	// Lock 2 -- no production file mints an ai.TokenCount.
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(%q) returned %v, want nil", ".", err)
	}

	fset := token.NewFileSet()
	scanned := 0
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, perr := parser.ParseFile(fset, name, nil, 0)
		if perr != nil {
			t.Fatalf("parser.ParseFile(%q) returned %v, want nil", name, perr)
		}
		scanned++

		ast.Inspect(file, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok {
				return true
			}
			sel, ok := call.Fun.(*ast.SelectorExpr)
			if !ok || sel.Sel == nil || sel.Sel.Name != "Tokens" {
				return true
			}
			pkg, ok := sel.X.(*ast.Ident)
			if !ok || pkg.Name != "ai" {
				return true
			}
			t.Errorf("%s:%d calls ai.Tokens -- Layer 2 MUST NOT construct an ai.TokenCount; the presence bit means Layer 1 reported the figure, and an estimate never did (S-CTX-017)", name, fset.Position(call.Pos()).Line)
			return true
		})
	}

	if scanned == 0 {
		t.Fatal("scanned 0 production files -- the guard asserted nothing")
	}
}

// --- Phase 3 -- the accounting resolver (R-CTX-006, R-CTX-007) -----------

// TestResolveTokenAccounting_ThreeStates — S-CTX-012, the three-state
// table. Four fixtures: (a) a counting fixture reporting a present
// figure, (b) the non-counting fake, (c) an ADVERTISED counter
// returning a non-nil error, (d) an ADVERTISED counter returning a
// NIL error with an ABSENT count. (a) yields Reported with that
// figure; (b) yields Estimated with the estimate's own figure; (c)
// and (d) BOTH yield Unavailable with no figure -- specifically NOT
// Estimated -- and their value equals the zero TokenAccounting.
func TestResolveTokenAccounting_ThreeStates(t *testing.T) {
	t.Parallel()

	opts := TurnOptions{}
	system := "system prompt for the resolver's three-state table"
	transcript := []ai.Message{mustEstimateText(t, ai.RoleUser, "resolve my accounting please")}

	t.Run("(a) reporting counter -> Reported with its own figure", func(t *testing.T) {
		t.Parallel()
		provider := agenttest.NewCountingProvider(ai.Tokens(555))

		got := resolveTokenAccounting(context.Background(), provider, opts, system, transcript)
		tokens, source := got.Tokens()
		if source != TokenSourceReported || tokens != 555 {
			t.Errorf("resolveTokenAccounting(reporting counter) = (%d, %v), want (555, TokenSourceReported)", tokens, source)
		}
	})

	t.Run("(b) non-counting fake -> Estimated with the estimate's figure", func(t *testing.T) {
		t.Parallel()
		provider := agenttest.NewProvider()

		req, err := buildLoopRequest(opts, system, transcript)
		if err != nil {
			t.Fatalf("buildLoopRequest returned %v, want no failure", err)
		}
		want := estimateTokens(req)

		got := resolveTokenAccounting(context.Background(), provider, opts, system, transcript)
		tokens, source := got.Tokens()
		if source != TokenSourceEstimated || tokens != want {
			t.Errorf("resolveTokenAccounting(non-counting) = (%d, %v), want (%d, TokenSourceEstimated)", tokens, source, want)
		}
	})

	t.Run("(c) advertised counter errors -> Unavailable, NEVER Estimated", func(t *testing.T) {
		t.Parallel()
		provider := agenttest.NewFailingCountingProvider(errors.New("advertised counter declines"))

		got := resolveTokenAccounting(context.Background(), provider, opts, system, transcript)
		var zero TokenAccounting
		tokens, source := got.Tokens()
		if source != TokenSourceUnavailable || tokens != 0 {
			t.Errorf("resolveTokenAccounting(erroring advertised counter) = (%d, %v), want (0, TokenSourceUnavailable)", tokens, source)
		}
		if got != zero {
			t.Errorf("resolveTokenAccounting(erroring advertised counter) = %#v, want the zero TokenAccounting (R-AMP-019: non-conformant, not absent, NEVER estimated)", got)
		}
	})

	t.Run("(d) advertised counter answers nil-error absent-count -> Unavailable, NEVER Estimated", func(t *testing.T) {
		t.Parallel()
		provider := agenttest.NewCountingProvider(ai.TokenCount{})

		got := resolveTokenAccounting(context.Background(), provider, opts, system, transcript)
		var zero TokenAccounting
		tokens, source := got.Tokens()
		if source != TokenSourceUnavailable || tokens != 0 {
			t.Errorf("resolveTokenAccounting(nil-error absent-count) = (%d, %v), want (0, TokenSourceUnavailable)", tokens, source)
		}
		if got != zero {
			t.Errorf("resolveTokenAccounting(nil-error absent-count) = %#v, want the zero TokenAccounting -- \"answered no figure\" is not a report", got)
		}
	})
}

// TestResolveTokenAccounting_BuildFailureYieldsUnavailable — R-CTX-006:
// a buildLoopRequest failure at the accounting point yields Unavailable
// rather than aborting; the turn's own build (loop.go:304) stays the
// single abort authority. An empty transcript makes buildLoopRequest's
// underlying ai.NewRequest fail its "at least one message" rule.
func TestResolveTokenAccounting_BuildFailureYieldsUnavailable(t *testing.T) {
	t.Parallel()

	provider := agenttest.NewCountingProvider(ai.Tokens(999))
	got := resolveTokenAccounting(context.Background(), provider, TurnOptions{}, "system prompt", nil)

	var zero TokenAccounting
	if got != zero {
		t.Errorf("resolveTokenAccounting(empty transcript, build failure) = %#v, want the zero TokenAccounting (Unavailable)", got)
	}
	if got := len(provider.CountRequests()); got != 0 {
		t.Errorf("CountRequests() = %d, want 0 -- a build failure must never reach the counter", got)
	}
}
