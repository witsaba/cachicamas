// AI-26.2 — cache-boundary markers are dropped whole (R-ART-006).
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
// markers vanish whole" section for why this file's walk over
// ai.CacheRegions() runs two regions (messages, system) to completion and
// records the third (tools) SKIP, with a named reason, rather than a
// fabricated PASS over content Translate does not render at all yet.
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

// TestCacheMarker_MarkedTwinAcrossEveryRegion walks every cache region
// ai.CacheRegions() enumerates (R-ART-006, S-ART-022) — mechanically, not
// hand-listed, so a later slice's own region-rendering work is picked up
// here with no edit to this loop or this walk's structure.
//
// The messages region (appendMessageObject, body.go, AI-26.1) and the
// system region (appendSystemMessageObject, system.go, AI-26.2) both have
// a renderer that already emits a carrier's full substantive content —
// role/text, and segment text in the system role, respectively — onto
// the wire, so both sub-tests run to completion (S-ART-021). The tools
// region is recorded SKIP with the exact blocking reason: a marker-drop
// proof over declarations Translate does not render at all would be
// vacuous under AI-11.3's own S-ACB-035, not a proof that the marker
// specifically is what is dropped. That gap is structural and
// self-closing — it starts proving something real once AI-26.4's
// tool.go lands, with no edit needed to this file — see doc.go.
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
				t.Skip("deferred to AI-26.4 (slice 3): tool.go does not exist yet, so Translate does not " +
					"render tool declarations at all — a marker-drop proof here would be vacuous under " +
					"AI-11.3's own S-ACB-035; this sub-test starts proving something real once slice 3 lands, " +
					"with no edit needed to this file")

			default:
				t.Fatalf("unhandled cache region %v — ai.CacheRegions() grew and this walk was not updated", region)
			}
		})
	}
}
