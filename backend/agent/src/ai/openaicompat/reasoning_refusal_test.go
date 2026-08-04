// AI-26.6 — the reasoning refusal: a request whose assistant message
// carries a reasoning content part fails translation with an AI-19
// PreStreamFailure carrying ai.ErrUnsupportedCapability, in every
// reasoning state and at every position, with no wire body produced
// (R-ART-015).
package openaicompat_test

import (
	"errors"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// refusalCase is one request that MUST fail translation with an AI-19
// refusal — translation_test.go's own expectationCase, mirrored for the
// refusal path (design.md "Harness"). AI-26.8's own exhaustive policy
// walk (a later slice) may register further cases here from its own test
// file with no edit to this one, the same way expectationCases already
// lets every later node add cases without touching translation_test.go.
type refusalCase struct {
	name  string
	build func() ai.Request
}

// refusalCases is the registry every refusal-producing node self-registers
// cases into via its own init() — registerExpectations' own sibling for
// the refusal path.
var refusalCases []refusalCase

// registerRefusals appends cases to the shared registry from each node's
// own init(). Called at most a handful of times from a fixed set of
// init() functions, never ranged as a map, so registration order is
// append order — registerExpectations' own contract, restated.
func registerRefusals(cases ...refusalCase) {
	refusalCases = append(refusalCases, cases...)
}

func init() {
	registerRefusals(
		// Three cases per reasoning state (S-ART-051): ai.ReasoningState's
		// own three members, one request each, each carrying nothing else
		// that could independently cause a refusal.
		refusalCase{
			name: "assistant reasoning in the text state refuses",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewReasoning("working through the steps", nil)))),
				}))
			},
		},
		refusalCase{
			name: "assistant reasoning in the redacted state refuses",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewRedactedReasoning([]byte("opaque-redacted-payload"))))),
				}))
			},
		},
		refusalCase{
			name: "assistant reasoning in the token-only state refuses",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewReasoning("", []byte("opaque-token-only-payload"))))),
				}))
			},
		},
		// Three cases varying which MESSAGE in a several-message request
		// carries the reasoning part (S-ART-052, message-level position).
		// The "first" case's request opens on an assistant message — legal
		// per role.go's own citation that this dialect does not enforce
		// user/assistant alternation (claim 4, doc.go's provenance
		// section) — chosen deliberately so this case cannot be satisfied
		// by a scan that only inspects the request's own first message
		// for an unrelated reason (it would still be the first message
		// here); the "mid" and "last" cases are what actually rule out an
		// implementation that stops scanning after message index 0.
		refusalCase{
			name: "reasoning-bearing message is the first of several",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewReasoning("first message reasoning", nil)))),
					mustMessage(ai.NewMessage(ai.RoleUser,
						mustPart(ai.NewText("Second turn, no reasoning here.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewText("Third turn, still no reasoning.")))),
				}))
			},
		},
		refusalCase{
			name: "reasoning-bearing message sits in the middle of several",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser,
						mustPart(ai.NewText("First turn, no reasoning here.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewReasoning("middle message reasoning", nil)))),
					mustMessage(ai.NewMessage(ai.RoleUser,
						mustPart(ai.NewText("Third turn, still no reasoning.")))),
				}))
			},
		},
		refusalCase{
			name: "reasoning-bearing message is the last of several",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleUser,
						mustPart(ai.NewText("First turn, no reasoning here.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewText("Second turn, still no reasoning.")))),
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewReasoning("last message reasoning", nil)))),
				}))
			},
		},
		// Three cases varying where, WITHIN one message's own content, the
		// reasoning part sits (S-ART-052, intra-message position) — the
		// specific concern Phase 5/6's own hand-off note named:
		// appendContentPartObject's array-rendering path walks a message's
		// parts one at a time, so a check that only inspected content[0]
		// of each message would miss a reasoning part later in the same
		// message's own array.
		refusalCase{
			name: "reasoning part is the first of several parts in its message",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewReasoning("leading reasoning", nil)),
						mustPart(ai.NewText("visible answer text")),
					)),
				}))
			},
		},
		refusalCase{
			name: "reasoning part sits mid-message between two other parts",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewText("before")),
						mustPart(ai.NewReasoning("middle-of-message reasoning", nil)),
						mustPart(ai.NewText("after")),
					)),
				}))
			},
		},
		refusalCase{
			name: "reasoning part is the last of several parts in its message",
			build: func() ai.Request {
				return mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
					mustMessage(ai.NewMessage(ai.RoleAssistant,
						mustPart(ai.NewText("visible answer text")),
						mustPart(ai.NewReasoning("trailing reasoning", nil)),
					)),
				}))
			},
		},
	)
}

// TestRefusalCases_FailWithUnsupportedCapability is this file's primary
// proof (S-ART-051, S-ART-052): every registered case fails translation
// with an AI-19 refusal identifiable via
// errors.Is(err, ai.ErrUnsupportedCapability) — through ai.Failure's own
// Is method, not a string match — and produces no wire body. Checking
// error identity, not merely non-nil, is deliberate: a translation that
// fails for an unrelated reason (a different capability gap, an internal
// defect) would satisfy a bare "err != nil" check just as well, which is
// exactly the vacuous-pass shape ("refused with the right typed error
// naming the feature" vs. "failed for some unrelated reason") this
// milestone's own evidence discipline exists to catch.
func TestRefusalCases_FailWithUnsupportedCapability(t *testing.T) {
	for _, tc := range refusalCases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := openaicompat.Translate(tc.build())
			if err == nil {
				t.Fatalf("Translate: no error, want a refusal")
			}
			if got != nil {
				t.Fatalf("Translate: got %d byte(s) of wire body alongside a refusal error, want none", len(got))
			}
			if !errors.Is(err, ai.ErrUnsupportedCapability) {
				t.Fatalf("Translate error = %v, want errors.Is(err, ai.ErrUnsupportedCapability)", err)
			}
		})
	}
}

// TestReasoningRefusal_NamesReasoningContentAsTheUnsupportedFeature proves
// S-ART-053: a caller can learn what capability failed without reading
// this package's source. ai.Failure.Error() deliberately never reproduces
// its own wrapped Cause's text (provider_failure.go's own documented
// redaction posture), so the feature name is reached through
// errors.Unwrap, not through the top-level error string — and that
// unwrapped cause's own text is asserted to contain "reasoning content"
// verbatim, not merely to be non-nil.
func TestReasoningRefusal_NamesReasoningContentAsTheUnsupportedFeature(t *testing.T) {
	req := mustRequest(ai.NewRequest("gpt-4o", []ai.Message{
		mustMessage(ai.NewMessage(ai.RoleAssistant,
			mustPart(ai.NewReasoning("working through the steps", nil)))),
	}))

	_, err := openaicompat.Translate(req)
	if err == nil {
		t.Fatalf("Translate: no error, want a refusal")
	}
	cause := errors.Unwrap(err)
	if cause == nil {
		t.Fatalf("errors.Unwrap(err) = nil, want a cause naming the unsupported feature")
	}
	if !strings.Contains(cause.Error(), "reasoning content") {
		t.Fatalf("errors.Unwrap(err).Error() = %q, want it to name \"reasoning content\"", cause.Error())
	}
}

// TestPolicy_NoNewSentinelsExported proves S-ART-054: AI-26 itself
// exported no error-sentinel-shaped identifier of its own. The scenario
// is change-scoped — "compared against the set before the change" — and
// every refusal openaicompat.Translate can return is reachable only
// through ai.ErrUnsupportedCapability, already exported by package ai
// (provider_failure.go, AI-19) before this milestone existed. The scan's
// want-set is therefore an enumerated allowlist, not an empty freeze:
// AI-26 contributed no entry, and the only sanctioned entries are the
// two the decoder milestone's spec requires as exported, mutually
// distinguishable identities (errors.go's ErrFrameTooLarge and
// ErrTruncated, R-ASD-019/R-ASD-020), admitted when the AI-26 and AI-27
// chains merged. doc.go's "Refusal taxonomy: AI-25 vs AI-26" section
// (NFR-ART-E) states the same scoping in prose; this test keeps it
// mechanical and live for every future change to this package —
// credential_scan_test.go's own "later nodes covered automatically"
// property (S-ART-014), restated for sentinels instead of credentials.
//
// A source scan of the package's own directory, using go/parser rather
// than a raw byte match, so a sentinel-shaped comment or string literal
// cannot false-positive it — the same go/ast-based call-site scanning
// idiom ambient_authority_test.go (AI-25.2) already establishes in this
// directory, applied here to top-level declarations instead of call
// sites. Non-test files only: an identifier exported only from a
// "_test.go" file is never visible to a real importer of this package,
// so it is not part of "the module's exported sentinel set" this
// scenario means.
func TestPolicy_NoNewSentinelsExported(t *testing.T) {
	// AI-26's own contribution to this set is zero (S-ART-054). The two
	// entries below are AI-27's, required exported and mutually
	// distinguishable by the decoder spec (R-ASD-019, R-ASD-020). A
	// future sentinel fails this scan until a spec sanctions it and this
	// list names it.
	wantSentinels := []string{
		"errors.go:ErrFrameTooLarge",
		"errors.go:ErrTruncated",
	}

	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("os.ReadDir(\".\"): %v", err)
	}

	fset := token.NewFileSet()
	var gotSentinels []string
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		file, parseErr := parser.ParseFile(fset, name, nil, 0)
		if parseErr != nil {
			t.Fatalf("parser.ParseFile(%s): %v", name, parseErr)
		}
		for _, decl := range file.Decls {
			genDecl, ok := decl.(*ast.GenDecl)
			if !ok || (genDecl.Tok != token.VAR && genDecl.Tok != token.CONST) {
				continue
			}
			for _, spec := range genDecl.Specs {
				valueSpec, ok := spec.(*ast.ValueSpec)
				if !ok {
					continue
				}
				for _, ident := range valueSpec.Names {
					if ast.IsExported(ident.Name) && strings.HasPrefix(ident.Name, "Err") {
						gotSentinels = append(gotSentinels, name+":"+ident.Name)
					}
				}
			}
		}
	}

	if !slices.Equal(gotSentinels, wantSentinels) {
		t.Fatalf("exported sentinel-shaped identifiers = %v, want %v (S-ART-054: zero new sentinels)", gotSentinels, wantSentinels)
	}
}
