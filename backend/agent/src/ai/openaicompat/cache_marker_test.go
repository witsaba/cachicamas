// AI-26.2/AI-26.4 — cache-boundary markers are dropped whole (R-ART-006).
//
// The proof is a twin comparison, mirroring cache_boundary_test.go's
// markedEquivalentRequest / buildRequestMarkingOneCarrier idiom at the
// neutral level: build a request and an otherwise-identical twin marking
// one cache region's carrier, then compare. That file compares
// ai.Request.Equal; this one compares openaicompat.Translate's wire bytes,
// which is what R-ART-006 actually requires.
//
// This exercises, and does not modify, AI-11.3's advisory contract
// (openspec/specs/ai-cache-breakpoints/spec.md, R-ACB-008): "markers are
// advisory: a translator may ignore every one of them." R-ACB-008's own
// text conditions that on the translator reading a region's full
// substantive content and simply never consulting the marker on it —
// S-ACB-035 requires the region to be complete, not merely present, for
// the twin comparison to be non-vacuous. See doc.go's "Cache-boundary
// markers vanish whole" section: as of tool.go (AI-26.4), every region
// ai.CacheRegions() enumerates (messages, system, tools) has a renderer
// that emits a carrier's full substantive content onto the wire, so this
// file's walk now runs every sub-test to completion, non-vacuously — none
// is recorded SKIP any longer.
package openaicompat_test

import (
	"testing"

	"github.com/cachicamas/backend/agent/src/ai"
	"github.com/cachicamas/backend/agent/src/ai/openaicompat"
)

// buildMessagesRegionTwin builds two requests identical but for a
// cache-boundary marker on the one message each carries — mirroring
// cache_boundary_test.go's markedEquivalentRequest idiom (a plain and a
// marked twin built from one shape), but returned for byte comparison of
// Translate's output rather than ai.Request.Equal.
func buildMessagesRegionTwin(t *testing.T) (plain, marked ai.Request) {
	t.Helper()

	build := func(mark bool) ai.Request {
		part, err := ai.NewText("plan a trip")
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}
		msg, err := ai.NewMessage(ai.RoleUser, part)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}
		if mark {
			msg = msg.MarkCacheBoundary()
		}
		request, err := ai.NewRequest("gpt-4o", []ai.Message{msg})
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		return request
	}

	return build(false), build(true)
}

// buildSystemRegionTwin builds two requests identical but for a
// cache-boundary marker on the one system-instruction segment each
// carries — mirroring buildMessagesRegionTwin above. Now that system.go
// (AI-26.2) renders a segment's full substantive content — its text, in
// this adapter's system role — onto the wire, this twin comparison is
// non-vacuous under AI-11.3's own S-ACB-035.
func buildSystemRegionTwin(t *testing.T) (plain, marked ai.Request) {
	t.Helper()

	build := func(mark bool) ai.Request {
		segment, err := ai.NewSegment("be terse")
		if err != nil {
			t.Fatalf("ai.NewSegment returned %v, want no failure", err)
		}
		if mark {
			segment = segment.MarkCacheBoundary()
		}
		system, err := ai.NewSystemInstruction(segment)
		if err != nil {
			t.Fatalf("ai.NewSystemInstruction returned %v, want no failure", err)
		}

		part, err := ai.NewText("plan a trip")
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}
		msg, err := ai.NewMessage(ai.RoleUser, part)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		request, err := ai.NewRequest("gpt-4o", []ai.Message{msg}, ai.WithSystemInstruction(system))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		return request
	}

	return build(false), build(true)
}

// buildToolsRegionTwin builds two requests identical but for a
// cache-boundary marker on the one tool declaration each carries —
// mirroring buildMessagesRegionTwin/buildSystemRegionTwin above. Now that
// tool.go (AI-26.4) renders a tool's full substantive content — its name,
// description and schema bytes — onto the wire, this twin comparison is
// non-vacuous under AI-11.3's own S-ACB-035.
func buildToolsRegionTwin(t *testing.T) (plain, marked ai.Request) {
	t.Helper()

	build := func(mark bool) ai.Request {
		tool, err := ai.NewTool("get_weather", "Get the current weather for a location.",
			[]byte(`{"type":"object","properties":{"location":{"type":"string"}},"required":["location"]}`))
		if err != nil {
			t.Fatalf("ai.NewTool returned %v, want no failure", err)
		}
		if mark {
			tool = tool.MarkCacheBoundary()
		}
		tools, err := ai.NewToolSet(tool)
		if err != nil {
			t.Fatalf("ai.NewToolSet returned %v, want no failure", err)
		}

		part, err := ai.NewText("plan a trip")
		if err != nil {
			t.Fatalf("ai.NewText returned %v, want no failure", err)
		}
		msg, err := ai.NewMessage(ai.RoleUser, part)
		if err != nil {
			t.Fatalf("ai.NewMessage returned %v, want no failure", err)
		}

		request, err := ai.NewRequest("gpt-4o", []ai.Message{msg}, ai.WithTools(tools))
		if err != nil {
			t.Fatalf("ai.NewRequest returned %v, want no failure", err)
		}
		return request
	}

	return build(false), build(true)
}

// TestCacheMarker_MarkedTwinAcrossEveryRegion walks every cache region
// ai.CacheRegions() enumerates (R-ART-006, S-ART-022) — mechanically, not
// hand-listed, so a later slice's own region-rendering work is picked up
// here with no edit to this loop or this walk's structure.
//
// The messages region (appendMessageObject, body.go, AI-26.1), the system
// region (appendSystemMessageObject, system.go, AI-26.2) and the tools
// region (appendToolObject, tool.go, AI-26.4) all now have a renderer
// that emits a carrier's full substantive content — role/text, segment
// text in the system role, and a tool's name/description/schema bytes,
// respectively — onto the wire, so every sub-test runs to completion
// (S-ART-021), non-vacuously under AI-11.3's own S-ACB-035. None is
// recorded SKIP: this walk's own loop/switch structure needed no edit to
// pick up tool.go's landing — only the tools case's body needed its
// placeholder t.Skip replaced with the same twin-comparison shape the
// other two regions already used, exactly as this file's own prior
// comment anticipated.
func TestCacheMarker_MarkedTwinAcrossEveryRegion(t *testing.T) {
	for _, region := range ai.CacheRegions() {
		t.Run(region.String(), func(t *testing.T) {
			switch region {
			case ai.CacheRegionMessages:
				plain, marked := buildMessagesRegionTwin(t)

				plainBytes, err := openaicompat.Translate(plain)
				if err != nil {
					t.Fatalf("Translate(plain): unexpected error: %v", err)
				}
				markedBytes, err := openaicompat.Translate(marked)
				if err != nil {
					t.Fatalf("Translate(marked): unexpected error: %v", err)
				}
				if string(plainBytes) != string(markedBytes) {
					t.Fatalf("Translate(marked) =\n%s\nwant byte-identical to Translate(plain) =\n%s", markedBytes, plainBytes)
				}

			case ai.CacheRegionSystem:
				plain, marked := buildSystemRegionTwin(t)

				plainBytes, err := openaicompat.Translate(plain)
				if err != nil {
					t.Fatalf("Translate(plain): unexpected error: %v", err)
				}
				markedBytes, err := openaicompat.Translate(marked)
				if err != nil {
					t.Fatalf("Translate(marked): unexpected error: %v", err)
				}
				if string(plainBytes) != string(markedBytes) {
					t.Fatalf("Translate(marked) =\n%s\nwant byte-identical to Translate(plain) =\n%s", markedBytes, plainBytes)
				}

			case ai.CacheRegionTools:
				plain, marked := buildToolsRegionTwin(t)

				plainBytes, err := openaicompat.Translate(plain)
				if err != nil {
					t.Fatalf("Translate(plain): unexpected error: %v", err)
				}
				markedBytes, err := openaicompat.Translate(marked)
				if err != nil {
					t.Fatalf("Translate(marked): unexpected error: %v", err)
				}
				if string(plainBytes) != string(markedBytes) {
					t.Fatalf("Translate(marked) =\n%s\nwant byte-identical to Translate(plain) =\n%s", markedBytes, plainBytes)
				}

			default:
				t.Fatalf("unhandled cache region %v — ai.CacheRegions() grew and this walk was not updated", region)
			}
		})
	}
}
