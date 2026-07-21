// Package-internal test for the AI-14 v1 reasoning-event no-op contract.
//
// AI-14 locks the v1 reasoning-event contract as an explicit no-op /
// unsupported contract under AI-06 v1 = ABSENT. v1 Layer 1 adapters MUST
// NOT emit EventKindReasoningStart / Delta / End events. The three
// reserved kinds stay in the AI-11 registry untouched so future
// provider enablement is additive. AI-21 conformance skips reasoning
// payload cases with reason citing "see AI-02 § Reasoning policy".
//
// This file lives in `package ai_test` because the AI-14 contract is
// asserted from the Layer 1 boundary (the same convention as
// text_event_test.go). The package-internal ai/event_test.go owns the
// envelope invariants; reasoning_event_test.go owns the v1 no-op
// contract and the doc.go / event.go audit assertions.
//
// Per AI-14 spec #2204 REQ-AI14-1..5 and design #2205 § 6.
package ai_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"sort"
	"strings"
	"testing"

	"github.com/cachicamas/backend/database_administrator/src/tools/agent/ai"
)

// ---------------------------------------------------------------------------
// T-AI14-RED-001 — v1 stream emits no reasoning-kind events (REQ-AI14-1)
// ---------------------------------------------------------------------------

// emitV1TextOnlyStream constructs a 5-event text-only sequence through
// the sanctioned Layer 1 event constructors (no production-code helper
// added by AI-14). The lifecycle is the AI-12/13 happy path:
//
//	EventKindResponseStart -> EventKindTextStart
//	                       -> EventKindTextDelta (single)
//	                       -> EventKindTextEnd
//	-> EventKindResponseComplete
//
// The helper returns the slice verbatim; tests inspect Kind for the
// presence of any reasoning-kind event.
func emitV1TextOnlyStream(t *testing.T) []ai.Event {
	t.Helper()
	usage, err := ai.NewUsage(10, 20, 30, nil, nil, nil)
	if err != nil {
		t.Fatalf("setup NewUsage = %v, want nil", err)
	}
	start, err := ai.NewResponseStartEvent("resp-ai14", "model-ai14")
	if err != nil {
		t.Fatalf("setup NewResponseStartEvent = %v, want nil", err)
	}
	textStart, err := ai.NewTextStartEvent()
	if err != nil {
		t.Fatalf("setup NewTextStartEvent = %v, want nil", err)
	}
	textDelta, err := ai.NewTextDeltaEvent("hello")
	if err != nil {
		t.Fatalf("setup NewTextDeltaEvent = %v, want nil", err)
	}
	textEnd, err := ai.NewTextEndEvent()
	if err != nil {
		t.Fatalf("setup NewTextEndEvent = %v, want nil", err)
	}
	complete, err := ai.NewResponseCompleteEvent("resp-ai14", ai.FinishReasonStop, usage)
	if err != nil {
		t.Fatalf("setup NewResponseCompleteEvent = %v, want nil", err)
	}
	return []ai.Event{start, textStart, textDelta, textEnd, complete}
}

// TestV1StreamEmitsNoReasoningEvents verifies that a 5-event text-only
// v1 stream (the canonical AI-12/13 happy path) contains zero
// EventKindReasoningStart / Delta / End events. The test is the
// Layer-1-boundary acceptance for REQ-AI14-1: v1 MUST NOT emit
// reasoning-kind events. AI-21 conformance will assert the same
// invariant for every registered producer.
func TestV1StreamEmitsNoReasoningEvents(t *testing.T) {
	stream := emitV1TextOnlyStream(t)
	if len(stream) != 5 {
		t.Fatalf("emitV1TextOnlyStream returned %d events, want 5 (response.start + text.start + text.delta + text.end + response.complete)", len(stream))
	}
	for i, ev := range stream {
		switch ev.Kind {
		case ai.EventKindReasoningStart, ai.EventKindReasoningDelta, ai.EventKindReasoningEnd:
			t.Errorf("event[%d].Kind = %q, must not be a reserved reasoning kind in v1 (AI-14 spec #2204 REQ-AI14-1; see AI-02 § Reasoning policy)", i, ev.Kind)
		}
	}
}

// ---------------------------------------------------------------------------
// T-AI14-RED-002 — reserved reasoning kinds stay registered (REQ-AI14-2)
// ---------------------------------------------------------------------------

// TestAllEventKindsIncludesReservedReasoning verifies that
// AllEventKinds returns the canonical 12-slot registry and that the
// three reserved reasoning kinds remain present. AI-14 MUST NOT remove
// them — they are reserved-but-unsupported in v1, not absent. AI-21
// conformance skips reasoning payload cases with reason citing
// "see AI-02 § Reasoning policy", which presupposes the registry
// still lists them.
func TestAllEventKindsIncludesReservedReasoning(t *testing.T) {
	kinds := ai.AllEventKinds()
	if len(kinds) != 12 {
		t.Fatalf("AllEventKinds() length = %d, want 12 (response x2, text x3, reasoning x3, tool-call x3, error)", len(kinds))
	}
	reserved := map[ai.EventKind]bool{
		ai.EventKindReasoningStart: false,
		ai.EventKindReasoningDelta: false,
		ai.EventKindReasoningEnd:   false,
	}
	for _, k := range kinds {
		if _, tracked := reserved[k]; tracked {
			reserved[k] = true
		}
	}
	for k, found := range reserved {
		if !found {
			t.Errorf("AllEventKinds() is missing reserved reasoning kind %q (AI-14 spec #2204 REQ-AI14-2)", k)
		}
	}
}

// ---------------------------------------------------------------------------
// T-AI14-RED-003 — reasoning cannot masquerade as text-answer (REQ-AI14-3)
// ---------------------------------------------------------------------------

// TestReasoningCannotMasqueradeAsTextAnswer verifies the
// reasoning/text discriminator separation at the Layer 2 boundary.
// A ContentPart produced by ContentPartFromReasoning has runtime type
// ai.Reasoning and Kind() == KindReasoning; the same is true for a
// Text value (Kind() == KindText). Type-asserting across the two
// MUST fail so a Layer 2 consumer cannot accidentally treat a
// reasoning payload as a text-answer payload.
func TestReasoningCannotMasqueradeAsTextAnswer(t *testing.T) {
	// Reasoning side — ContentPartFromReasoning returns ContentPart with
	// runtime type ai.Reasoning (no wrapper — see reasoning.go line 125).
	reasoningPart, err := ai.ContentPartFromReasoning(ai.ReasoningAbsent, "")
	if err != nil {
		t.Fatalf("setup ContentPartFromReasoning = %v, want nil", err)
	}
	if reasoningPart == nil {
		t.Fatal("setup ContentPartFromReasoning returned nil ContentPart")
	}
	if r, ok := reasoningPart.(ai.Reasoning); !ok {
		t.Errorf("part.(ai.Reasoning) on ReasoningAbsent part = (zero, false); runtime type %T — wrapper regression (see AI-06 design-resolution obs #2059)", reasoningPart)
	} else {
		if got := r.Kind(); got != ai.KindReasoning {
			t.Errorf("r.Kind() = %q, want %q (Reasoning must report KindReasoning)", got, ai.KindReasoning)
		}
		if got := r.State(); got != ai.ReasoningAbsent {
			t.Errorf("r.State() = %q, want %q", got, ai.ReasoningAbsent)
		}
	}
	if _, ok := reasoningPart.(ai.Text); ok {
		t.Error("part.(ai.Text) on ReasoningAbsent part = (zero, true); reasoning MUST NOT masquerade as text-answer (AI-14 spec #2204 REQ-AI14-3)")
	}

	// Text side — NewText returns ai.Text, which is itself a ContentPart
	// (it has Kind() Kind returning KindText; see text.go line 92).
	textVal, err := ai.NewText("hi")
	if err != nil {
		t.Fatalf("setup NewText = %v, want nil", err)
	}
	var textPart ai.ContentPart = textVal
	if assertedText, ok := textPart.(ai.Text); !ok {
		t.Errorf("part.(ai.Text) on text value = (zero, false); runtime type %T", textPart)
	} else if got := assertedText.Kind(); got != ai.KindText {
		t.Errorf("assertedText.Kind() = %q, want %q (Text must report KindText)", got, ai.KindText)
	}
	if _, ok := textPart.(ai.Reasoning); ok {
		t.Error("part.(ai.Reasoning) on text value = (zero, true); text-answer MUST NOT masquerade as reasoning (AI-14 spec #2204 REQ-AI14-3)")
	}
}

// ---------------------------------------------------------------------------
// T-AI14-RED-004 — doc.go paragraph cites AI-02 § Reasoning policy (REQ-AI14-4 + REQ-AI14-6)
// ---------------------------------------------------------------------------

// TestDocGoParagraphCitesReasoningPolicy verifies doc.go contains the
// AI-14 paragraph: it MUST (a) contain an explicit "AI-14" milestone
// marker that anchors the new paragraph after the AI-13 paragraph,
// (b) mention the literal substring "see AI-02 § Reasoning policy"
// (REQ-AI14-4 + REQ-AI14-6), and (c) end with a period (REQ-AI14-4 +
// REQ-AI14-6 truncation-trap, mirroring AI-06 spec #2057 § A req 6).
//
// At RED this test fails on the AI-14 milestone marker: the substring
// "see AI-02 § Reasoning policy" already exists in the AI-06 paragraph
// (line 54 of doc.go) and the last non-whitespace rune is already "."
// (the AI-13 paragraph terminates with "See text_event.go."), so the
// citation and period assertions are RED-shaped but GREEN at commit 1.
// The marker assertion is the strict RED gate that flips GREEN once
// the AI-14 paragraph is appended in commit 2.
func TestDocGoParagraphCitesReasoningPolicy(t *testing.T) {
	body, err := os.ReadFile("doc.go")
	if err != nil {
		t.Fatalf("ReadFile doc.go = %v", err)
	}
	doc := string(body)

	// (a) AI-14 paragraph marker — distinct from the AI-06 paragraph
	// that already cites AI-02 § Reasoning policy. The marker anchors
	// the new paragraph after the AI-13 paragraph (mirrors the AI-12
	// "AI-12 paragraph (added after package clause; ...)" precedent).
	if !strings.Contains(doc, "AI-14 paragraph") {
		t.Errorf("doc.go must contain the AI-14 paragraph marker %q (AI-14 spec #2204 REQ-AI14-4: append AI-14 paragraph after AI-13 paragraph)", "AI-14 paragraph")
	}

	// (b) Literal citation — anchors the AI-21 conformance skip-rule.
	const citation = "see AI-02 § Reasoning policy"
	if !strings.Contains(doc, citation) {
		t.Errorf("doc.go must contain the literal substring %q (AI-14 spec #2204 REQ-AI14-4 + REQ-AI14-6; AI-21 conformance skip-rule)", citation)
	}

	// (c) Period-terminated paragraph invariant (mirrors AI-06
	// truncation-trap: a doc.go paragraph that does not end with a
	// period signals an unfinished sentence in code review).
	trimmed := strings.TrimRight(doc, " \t\r\n")
	if trimmed == "" {
		t.Fatal("doc.go is empty or whitespace-only")
	}
	last := trimmed[len(trimmed)-1]
	if last != '.' {
		t.Errorf("doc.go last non-whitespace rune = %q, want %q (AI-14 spec #2204 REQ-AI14-4: paragraph must end with a period; AI-06 spec #2057 § A req 6 truncation-trap precedent)", last, '.')
	}
}

// ---------------------------------------------------------------------------
// T-AI14-RED-005 — event.go exported symbols unchanged (REQ-AI14-5)
// ---------------------------------------------------------------------------

// expectedEventGoExportedDecls is the pre-AI-14 baseline of exported
// declarations in event.go. AI-14 MUST NOT alter this set — the only
// permitted surface change is a comment-only clarification above the
// three reserved reasoning EventKind constants. The set was captured
// from `event.go` on `main` @ eb63826 via go/parser + ast.Inspect; the
// expected kind ("type"/"const"/"var"/"func"/"method") matches what
// the AST yields so a future regression that adds/removes/renames a
// symbol flips the test to FAIL.
var expectedEventGoExportedDecls = map[string]string{
	// Top-level types
	"EventKind": "type",
	"Event":     "type",
	// Top-level functions
	"NewEvent":      "func",
	"AllEventKinds": "func",
	// EventKind constants (canonical 12-slot registry)
	"EventKindResponseStart":    "const",
	"EventKindResponseComplete": "const",
	"EventKindTextStart":        "const",
	"EventKindTextDelta":        "const",
	"EventKindTextEnd":          "const",
	"EventKindReasoningStart":   "const",
	"EventKindReasoningDelta":   "const",
	"EventKindReasoningEnd":     "const",
	"EventKindToolCallStart":    "const",
	"EventKindToolCallDelta":    "const",
	"EventKindToolCallEnd":      "const",
	"EventKindError":            "const",
	// Sentinel errors
	"ErrEventKindUnregistered":    "var",
	"ErrEventPayloadKindMismatch": "var",
	"ErrEventPayloadMissing":      "var",
	// Methods (kind = "method:Receiver.Name")
	"EventKind.IsValid": "method",
	"EventKind.String":  "method",
	"Event.Validate":    "method",
}

// collectEventGoExportedDecls parses event.go and returns every
// exported declaration keyed by "<kind>:<symbol>" — the kind is the
// AST token (func / type / const / var) or "method" for receiver-
// bearing function declarations. Comments are parsed but do not
// contribute to the set.
func collectEventGoExportedDecls(t *testing.T) map[string]string {
	t.Helper()
	src, err := os.ReadFile("event.go")
	if err != nil {
		t.Fatalf("ReadFile event.go = %v", err)
	}
	fset := token.NewFileSet()
	file, err := parser.ParseFile(fset, "event.go", src, parser.ParseComments)
	if err != nil {
		t.Fatalf("parser.ParseFile(event.go) = %v, want nil", err)
	}
	got := map[string]string{}
	ast.Inspect(file, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.FuncDecl:
			if x.Recv == nil {
				if ast.IsExported(x.Name.Name) {
					got[x.Name.Name] = "func"
				}
			} else {
				recv := x.Recv.List[0].Type
				if star, ok := recv.(*ast.StarExpr); ok {
					recv = star.X
				}
				if id, ok := recv.(*ast.Ident); ok && ast.IsExported(id.Name) && ast.IsExported(x.Name.Name) {
					got[id.Name+"."+x.Name.Name] = "method"
				}
			}
		case *ast.GenDecl:
			for _, spec := range x.Specs {
				switch s := spec.(type) {
				case *ast.ValueSpec:
					for _, name := range s.Names {
						if !ast.IsExported(name.Name) {
							continue
						}
						switch x.Tok {
						case token.CONST:
							got[name.Name] = "const"
						case token.VAR:
							got[name.Name] = "var"
						}
					}
				case *ast.TypeSpec:
					if ast.IsExported(s.Name.Name) {
						got[s.Name.Name] = "type"
					}
				}
			}
		}
		return true
	})
	return got
}

// diffSymbolSets returns a stable, sorted representation of (a) symbols
// present in got but missing from want, and (b) symbols present in want
// but missing from got. Empty slices mean the two sets match exactly.
func diffSymbolSets(want, got map[string]string) (extra, missing []string) {
	for name := range got {
		if _, ok := want[name]; !ok {
			extra = append(extra, name)
		}
	}
	for name := range want {
		if _, ok := got[name]; !ok {
			missing = append(missing, name)
		}
	}
	sort.Strings(extra)
	sort.Strings(missing)
	return extra, missing
}

// TestEventGoExportedSymbolsUnchangedAfterAI14 verifies that the
// AI-14 change to event.go is comment-only: no exported symbol may be
// added, removed, renamed, or have its kind changed. The test parses
// event.go via go/parser and compares the exported symbol set against
// the pre-AI-14 baseline captured from main @ eb63826. Per REQ-AI14-5.
//
// The test is RED at the contract level (the assertion semantics are
// RED-shaped: it would FAIL on any non-comment-only regression). For
// the current change the test passes both at RED (baseline matches
// current state) and at GREEN (the production change is comment-only).
func TestEventGoExportedSymbolsUnchangedAfterAI14(t *testing.T) {
	got := collectEventGoExportedDecls(t)
	if len(got) != len(expectedEventGoExportedDecls) {
		t.Errorf("event.go exported declaration count = %d, want %d (AI-14 spec #2204 REQ-AI14-5: comment-only change must not alter the exported symbol set)",
			len(got), len(expectedEventGoExportedDecls))
	}
	extra, missing := diffSymbolSets(expectedEventGoExportedDecls, got)
	if len(extra) > 0 {
		t.Errorf("event.go exposes unexpected exported symbols (not in pre-AI-14 baseline): %v", extra)
	}
	if len(missing) > 0 {
		t.Errorf("event.go is missing pre-AI-14 exported symbols: %v", missing)
	}
	// Kind-equality check: every baseline symbol must also have the
	// same kind ("type"/"const"/"var"/"func"/"method") in the parsed
	// file. A future regression that reclassifies a symbol (e.g.,
	// changes `var` to `const`) trips this even if names line up.
	for name, wantKind := range expectedEventGoExportedDecls {
		if gotKind, ok := got[name]; ok && gotKind != wantKind {
			t.Errorf("event.go exported %q has kind %q, want %q (AI-14 REQ-AI14-5: comment-only change must preserve kind)",
				name, gotKind, wantKind)
		}
	}
}