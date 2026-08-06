// AI-26.8.2 — the exhaustive policy walk: every feature this milestone's
// inventory (feature_inventory_test.go) discovers resolves to exactly one
// disposition — translate, drop or refuse — and each non-translate
// disposition is proven true by an actual Translate() call, not merely
// declared (S-ART-074 … S-ART-079, S-ART-083).
//
// # Witnesses, not a re-implementation
//
// refuseWitnesses and dropWitnesses are this file's own half of
// feature_inventory_test.go's "per-feature witness table" (S-ART-069): one
// request-building closure per non-translate policy row. A
// translate-disposition feature is deliberately NOT witnessed again here —
// see feature_inventory_test.go's
// TestFeatureInventory_NonTranslateFeaturesHaveExactlyOneWitness for why
// that would be pure duplication of Phases 1-7's own expectationCases.
//
// The four refuse witnesses (one per PartKind:/ReasoningState: row this
// package's policy marks refuse) all build a reasoning-bearing request —
// deliberately, not by oversight: every one of them exercises the same
// single mechanism (policy.go's refuseReasoning), because "reasoning" is
// the only refuse-disposition feature this milestone's own mechanical
// inventory currently produces — verified here by construction (every
// dispositionRefuse row this file's own witness table covers is
// reasoning-shaped), not assumed, per Phase 7's own hand-off question.
// reasoning_refusal_test.go's own 9 registered refusalCases remain the
// authoritative position/state-independence proof (S-ART-051, S-ART-052);
// this file's own job is narrower and different: proving every policy row
// marked refuse is backed by a real refusal, not re-proving position
// independence a second time.
package openaicompat

import (
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
)

// refuseWitness is one non-translate policy row this file proves genuinely
// refuses, naming the disposition-marked feature (8.8, S-ART-075).
type refuseWitness struct {
	feature string
	build   func(t *testing.T) ai.Request
}

// dropWitness is one non-translate policy row this file proves genuinely
// drops: a marked request and its otherwise-identical, unmarked twin
// translate byte-identically (8.9, S-ART-076).
type dropWitness struct {
	feature string
	build   func(t *testing.T) (plain, marked ai.Request)
}

// mustReasoningWitnessRequest builds the smallest request carrying part as
// its one assistant message's one content part — the shape every
// refuseWitnesses entry below shares.
func mustReasoningWitnessRequest(t *testing.T, part ai.Part) ai.Request {
	t.Helper()
	message, err := ai.NewMessage(ai.RoleAssistant, part)
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	request, err := ai.NewRequest("gpt-4o", []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest: %v", err)
	}
	return request
}

// mustPolicyText builds a text part or fails the test — a small local
// helper so the witness/twin builders in this file stay one expression
// deep.
func mustPolicyText(t *testing.T, text string) ai.Part {
	t.Helper()
	part, err := ai.NewText(text)
	if err != nil {
		t.Fatalf("ai.NewText: %v", err)
	}
	return part
}

// refuseWitnesses is the witness table for every featurePolicy row whose
// disposition is dispositionRefuse (policy.go).
var refuseWitnesses = []refuseWitness{
	{
		feature: "PartKind: reasoning",
		build: func(t *testing.T) ai.Request {
			part, err := ai.NewReasoning("a PartKind-level reasoning witness", nil)
			if err != nil {
				t.Fatalf("ai.NewReasoning: %v", err)
			}
			return mustReasoningWitnessRequest(t, part)
		},
	},
	{
		feature: "ReasoningState: text",
		build: func(t *testing.T) ai.Request {
			part, err := ai.NewReasoning("a text-state reasoning witness", nil)
			if err != nil {
				t.Fatalf("ai.NewReasoning: %v", err)
			}
			return mustReasoningWitnessRequest(t, part)
		},
	},
	{
		feature: "ReasoningState: redacted",
		build: func(t *testing.T) ai.Request {
			part, err := ai.NewRedactedReasoning([]byte("opaque-redacted-policy-witness"))
			if err != nil {
				t.Fatalf("ai.NewRedactedReasoning: %v", err)
			}
			return mustReasoningWitnessRequest(t, part)
		},
	},
	{
		feature: "ReasoningState: token-only",
		build: func(t *testing.T) ai.Request {
			part, err := ai.NewReasoning("", []byte("opaque-token-only-policy-witness"))
			if err != nil {
				t.Fatalf("ai.NewReasoning: %v", err)
			}
			return mustReasoningWitnessRequest(t, part)
		},
	},
}

// buildMessagesRegionPolicyTwin mirrors cache_marker_test.go's own
// buildMessagesRegionTwin idiom, re-derived locally: this file is package
// openaicompat (internal, so it can read resolveFeaturePolicy/
// featurePolicy directly), while cache_marker_test.go is package
// openaicompat_test — two different compiled packages sharing one
// directory, so nothing unexported crosses between them.
func buildMessagesRegionPolicyTwin(t *testing.T) (plain, marked ai.Request) {
	t.Helper()
	build := func(mark bool) ai.Request {
		message, err := ai.NewMessage(ai.RoleUser, mustPolicyText(t, "plan a trip"))
		if err != nil {
			t.Fatalf("ai.NewMessage: %v", err)
		}
		if mark {
			message = message.MarkCacheBoundary()
		}
		request, err := ai.NewRequest("gpt-4o", []ai.Message{message})
		if err != nil {
			t.Fatalf("ai.NewRequest: %v", err)
		}
		return request
	}
	return build(false), build(true)
}

// buildSystemRegionPolicyTwin mirrors cache_marker_test.go's own
// buildSystemRegionTwin idiom — see buildMessagesRegionPolicyTwin's own
// comment for why it is re-derived here rather than imported.
func buildSystemRegionPolicyTwin(t *testing.T) (plain, marked ai.Request) {
	t.Helper()
	build := func(mark bool) ai.Request {
		segment, err := ai.NewSegment("be terse")
		if err != nil {
			t.Fatalf("ai.NewSegment: %v", err)
		}
		if mark {
			segment = segment.MarkCacheBoundary()
		}
		system, err := ai.NewSystemInstruction(segment)
		if err != nil {
			t.Fatalf("ai.NewSystemInstruction: %v", err)
		}
		message, err := ai.NewMessage(ai.RoleUser, mustPolicyText(t, "plan a trip"))
		if err != nil {
			t.Fatalf("ai.NewMessage: %v", err)
		}
		request, err := ai.NewRequest("gpt-4o", []ai.Message{message}, ai.WithSystemInstruction(system))
		if err != nil {
			t.Fatalf("ai.NewRequest: %v", err)
		}
		return request
	}
	return build(false), build(true)
}

// buildToolsRegionPolicyTwin mirrors cache_marker_test.go's own
// buildToolsRegionTwin idiom — see buildMessagesRegionPolicyTwin's own
// comment for why it is re-derived here rather than imported.
func buildToolsRegionPolicyTwin(t *testing.T) (plain, marked ai.Request) {
	t.Helper()
	build := func(mark bool) ai.Request {
		tool, err := ai.NewTool("get_weather", "Get the current weather for a location.",
			[]byte(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`))
		if err != nil {
			t.Fatalf("ai.NewTool: %v", err)
		}
		if mark {
			tool = tool.MarkCacheBoundary()
		}
		tools, err := ai.NewToolSet(tool)
		if err != nil {
			t.Fatalf("ai.NewToolSet: %v", err)
		}
		message, err := ai.NewMessage(ai.RoleUser, mustPolicyText(t, "plan a trip"))
		if err != nil {
			t.Fatalf("ai.NewMessage: %v", err)
		}
		request, err := ai.NewRequest("gpt-4o", []ai.Message{message}, ai.WithTools(tools))
		if err != nil {
			t.Fatalf("ai.NewRequest: %v", err)
		}
		return request
	}
	return build(false), build(true)
}

// buildForeignExtensionPolicyTwin mirrors extension_test.go's own
// TestExtension_ForeignNamespaceIgnoredWhole idiom — see
// buildMessagesRegionPolicyTwin's own comment for why it is re-derived here
// rather than imported.
func buildForeignExtensionPolicyTwin(t *testing.T) (plain, marked ai.Request) {
	t.Helper()
	message, err := ai.NewMessage(ai.RoleUser, mustPolicyText(t, "plan a trip"))
	if err != nil {
		t.Fatalf("ai.NewMessage: %v", err)
	}
	withoutForeign, err := ai.NewRequest("gpt-4o", []ai.Message{message})
	if err != nil {
		t.Fatalf("ai.NewRequest (without foreign namespace): %v", err)
	}
	withForeign, err := ai.NewRequest("gpt-4o", []ai.Message{message},
		ai.WithProviderExtension("anthropic", []byte(`"anthropic_only_field":"should never appear"`)))
	if err != nil {
		t.Fatalf("ai.NewRequest (with foreign namespace): %v", err)
	}
	return withoutForeign, withForeign
}

// dropWitnesses is the witness table for every featurePolicy row whose
// disposition is dispositionDrop (policy.go).
var dropWitnesses = []dropWitness{
	{feature: "CacheRegion: tools", build: buildToolsRegionPolicyTwin},
	{feature: "CacheRegion: system", build: buildSystemRegionPolicyTwin},
	{feature: "CacheRegion: messages", build: buildMessagesRegionPolicyTwin},
	{feature: "WithProviderExtension: foreign namespace", build: buildForeignExtensionPolicyTwin},
}

// unresolvedPolicyFeatures reports which of inventory has no featurePolicy
// entry at all — resolveFeaturePolicy's own "ok == false" case, the walk's
// exact failure mode (S-ART-074's "none unresolved"). Both
// TestPolicyWalk_EveryInventoriedFeatureResolvesToExactlyOneDisposition
// (the real walk, over the real inventory) and
// TestPolicyWalk_UnaccountedFeatureFailsTheWalk (8.11's own durable bite
// proof, over a synthetic inventory) call this identical function, so the
// walk proven to bite is the walk actually run.
func unresolvedPolicyFeatures(inventory []string) []string {
	var unresolved []string
	for _, feature := range inventory {
		if _, ok := resolveFeaturePolicy(feature); !ok {
			unresolved = append(unresolved, feature)
		}
	}
	return unresolved
}

// TestPolicyWalk_EveryInventoriedFeatureResolvesToExactlyOneDisposition is
// 8.7's own exhaustive walk (S-ART-074): every feature fullInventory
// discovers resolves to exactly one of dispositionTranslate/Drop/Refuse,
// carrying a non-empty recorded reason — never unresolved.
func TestPolicyWalk_EveryInventoriedFeatureResolvesToExactlyOneDisposition(t *testing.T) {
	inventory := fullInventory(t)
	if len(inventory) == 0 {
		t.Fatal("the discovered inventory is empty; this walk would pass vacuously")
	}

	if unresolved := unresolvedPolicyFeatures(inventory); len(unresolved) > 0 {
		t.Errorf("features with no policy entry at all: %v", unresolved)
	}

	for _, feature := range inventory {
		entry, ok := resolveFeaturePolicy(feature)
		if !ok {
			continue // already reported above.
		}
		switch entry.disposition {
		case dispositionTranslate, dispositionDrop, dispositionRefuse:
			// exactly one recognised disposition — the property this loop checks.
		default:
			t.Errorf("feature %q resolves to an unrecognised disposition value %d", feature, entry.disposition)
		}
		if strings.TrimSpace(entry.reason) == "" {
			t.Errorf("feature %q carries an empty recorded reason (S-ART-076 requires one)", feature)
		}
	}
}

// TestPolicyWalk_UnaccountedFeatureFailsTheWalk is S-ART-078's own durable
// bite proof: an inventory member with no policy entry fails the WALK
// itself (unresolvedPolicyFeatures, the function
// TestPolicyWalk_EveryInventoriedFeatureResolvesToExactlyOneDisposition
// runs), not only feature_inventory_test.go's lower-level
// unaccountedFeatures comparison (S-ART-070/071's own bite proofs) —
// reusing that same synthetic-extra-feature technique as the concrete
// staged case this task names, rather than a third re-implementation.
func TestPolicyWalk_UnaccountedFeatureFailsTheWalk(t *testing.T) {
	const scratchFeature = "scratch: a feature the walk has never accounted for"

	inventory := append(fullInventory(t), scratchFeature)
	unresolved := unresolvedPolicyFeatures(inventory)

	want := []string{scratchFeature}
	if !slices.Equal(unresolved, want) {
		t.Fatalf("unresolvedPolicyFeatures = %v, want %v", unresolved, want)
	}
}

// TestPolicyWalk_EveryRefuseFeatureFailsNamingIt is 8.8's own RED
// (S-ART-075): translating refuseWitnesses' request fails with an AI-19
// refusal identifiable via errors.Is(err, ai.ErrUnsupportedCapability),
// with no wire body produced, for every policy row marked refuse.
func TestPolicyWalk_EveryRefuseFeatureFailsNamingIt(t *testing.T) {
	for _, witness := range refuseWitnesses {
		t.Run(witness.feature, func(t *testing.T) {
			entry, ok := resolveFeaturePolicy(witness.feature)
			if !ok {
				t.Fatalf("refuseWitnesses names %q, which featurePolicy does not carry", witness.feature)
			}
			if entry.disposition != dispositionRefuse {
				t.Fatalf("refuseWitnesses names %q, whose policy disposition is not refuse", witness.feature)
			}

			got, err := Translate(witness.build(t))
			if err == nil {
				t.Fatalf("Translate: no error, want a refusal for feature %q", witness.feature)
			}
			if got != nil {
				t.Fatalf("Translate: got %d byte(s) of wire body alongside a refusal, want none", len(got))
			}
			if !errors.Is(err, ai.ErrUnsupportedCapability) {
				t.Fatalf("Translate error = %v, want errors.Is(err, ai.ErrUnsupportedCapability)", err)
			}
		})
	}
}

// TestPolicyWalk_EveryDropFeatureTranslatesByteIdenticalToItsTwin is 8.9's
// own RED (S-ART-076): a request marking dropWitnesses' feature translates
// byte-identically to its otherwise-identical, unmarked twin, and the
// policy row backing it carries a non-empty recorded reason.
func TestPolicyWalk_EveryDropFeatureTranslatesByteIdenticalToItsTwin(t *testing.T) {
	for _, witness := range dropWitnesses {
		t.Run(witness.feature, func(t *testing.T) {
			entry, ok := resolveFeaturePolicy(witness.feature)
			if !ok {
				t.Fatalf("dropWitnesses names %q, which featurePolicy does not carry", witness.feature)
			}
			if entry.disposition != dispositionDrop {
				t.Fatalf("dropWitnesses names %q, whose policy disposition is not drop", witness.feature)
			}
			if strings.TrimSpace(entry.reason) == "" {
				t.Fatalf("policy feature %q carries no recorded reason", witness.feature)
			}

			plain, marked := witness.build(t)
			plainBytes, err := Translate(plain)
			if err != nil {
				t.Fatalf("Translate(plain): unexpected error: %v", err)
			}
			markedBytes, err := Translate(marked)
			if err != nil {
				t.Fatalf("Translate(marked): unexpected error: %v", err)
			}
			if string(plainBytes) != string(markedBytes) {
				t.Fatalf("Translate(marked) =\n%s\nwant byte-identical to Translate(plain) =\n%s", markedBytes, plainBytes)
			}
		})
	}
}

// TestPolicyWalk_EveryRefusalUsesTheSameCategoryUniformly is 8.12's own
// confirmation (S-ART-079): every refuse-disposition feature's failure is
// reachable through the identical ai.ErrUnsupportedCapability sentinel.
// reasoning_refusal_test.go's own TestPolicy_NoNewSentinelsExported (re-run
// as part of this slice's own suite, not duplicated here) is what keeps "no
// second, differently-categorised refusal sentinel exists" a mechanical,
// standing guarantee across this package's own non-test sources; this test
// is that guard's positive half, over every refuse witness this
// milestone's inventory currently produces.
func TestPolicyWalk_EveryRefusalUsesTheSameCategoryUniformly(t *testing.T) {
	for _, witness := range refuseWitnesses {
		t.Run(witness.feature, func(t *testing.T) {
			_, err := Translate(witness.build(t))
			if err == nil {
				t.Fatalf("Translate: no error, want a refusal")
			}
			if !errors.Is(err, ai.ErrUnsupportedCapability) {
				t.Fatalf("Translate error = %v, want errors.Is(err, ai.ErrUnsupportedCapability) — every refuse-disposition feature must use the identical category", err)
			}
		})
	}
}

// TestTranslate_LeavesTheRequestEqualToAnIndependentlyBuiltCopy is 8.13's
// purity pin (S-ART-083, NFR-ART-C): translating a request exercising a
// system instruction, messages of every role, a tool call and its result,
// a declared tool, a tool choice, every generation option and a provider
// extension leaves it equal — ai.Request.Equal, the exported comparison,
// not an unexported field walk — to an independently built copy, both
// before and after Translate runs.
func TestTranslate_LeavesTheRequestEqualToAnIndependentlyBuiltCopy(t *testing.T) {
	build := func(t *testing.T) ai.Request {
		t.Helper()

		segment, err := ai.NewSegment("be terse")
		if err != nil {
			t.Fatalf("ai.NewSegment: %v", err)
		}
		system, err := ai.NewSystemInstruction(segment.MarkCacheBoundary())
		if err != nil {
			t.Fatalf("ai.NewSystemInstruction: %v", err)
		}

		tool, err := ai.NewTool("get_weather", "Get the current weather.",
			[]byte(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`))
		if err != nil {
			t.Fatalf("ai.NewTool: %v", err)
		}
		tools, err := ai.NewToolSet(tool.MarkCacheBoundary())
		if err != nil {
			t.Fatalf("ai.NewToolSet: %v", err)
		}

		choice, err := ai.NewToolChoice(ai.ToolChoiceAuto)
		if err != nil {
			t.Fatalf("ai.NewToolChoice: %v", err)
		}

		userMessage, err := ai.NewMessage(ai.RoleUser, mustPolicyText(t, "what's the weather in Tokyo?"))
		if err != nil {
			t.Fatalf("ai.NewMessage (user): %v", err)
		}

		callPart, err := ai.NewToolCall("call_1", "get_weather", []byte(`{"location":"Tokyo"}`))
		if err != nil {
			t.Fatalf("ai.NewToolCall: %v", err)
		}
		assistantMessage, err := ai.NewMessage(ai.RoleAssistant, callPart)
		if err != nil {
			t.Fatalf("ai.NewMessage (assistant): %v", err)
		}

		resultPart, err := ai.NewToolResult("call_1", "18C, light rain")
		if err != nil {
			t.Fatalf("ai.NewToolResult: %v", err)
		}
		toolMessage, err := ai.NewMessage(ai.RoleTool, resultPart)
		if err != nil {
			t.Fatalf("ai.NewMessage (tool): %v", err)
		}

		request, err := ai.NewRequest("gpt-4o",
			[]ai.Message{userMessage, assistantMessage, toolMessage},
			ai.WithSystemInstruction(system),
			ai.WithTools(tools),
			ai.WithToolChoice(choice),
			ai.WithMaxOutputTokens(256),
			ai.WithTemperature(0.4),
			ai.WithTopP(0.9),
			ai.WithStopSequences("STOP"),
			ai.WithProviderExtension(Namespace, []byte(`"top_k":40`)),
		)
		if err != nil {
			t.Fatalf("ai.NewRequest: %v", err)
		}
		return request
	}

	request := build(t)
	independentCopy := build(t)
	if !request.Equal(independentCopy) {
		t.Fatalf("the two independently built requests were not equal before Translate ran at all — this pin's own fixture is broken")
	}

	if _, err := Translate(request); err != nil {
		t.Fatalf("Translate: unexpected error: %v", err)
	}

	if !request.Equal(independentCopy) {
		t.Fatalf("Translate mutated request: it no longer equals an independently built copy (NFR-ART-C)")
	}
}
