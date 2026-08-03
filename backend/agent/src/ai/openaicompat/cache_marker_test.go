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
// ai.CacheRegions() runs one region to completion today and records the
// other two SKIP, with a named reason, rather than a fabricated PASS over
// content Translate does not render at all yet.
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

// TestCacheMarker_MarkedTwinAcrossEveryRegion walks every cache region
// ai.CacheRegions() enumerates (R-ART-006, S-ART-022) — mechanically, not
// hand-listed, so a later slice's own region-rendering work is picked up
// here with no edit to this loop or this walk's structure.
//
// Today only the messages region has a renderer (appendMessageObject,
// body.go, AI-26.1) that already emits a carrier's full substantive
// content — role and text — onto the wire, so it is the only sub-test run
// to completion (S-ART-021). The other two are recorded SKIP with the
// exact blocking reason: a marker-drop proof over content that is not
// rendered at all yet would be vacuous under AI-11.3's own S-ACB-035, not
// a proof that the marker specifically is what is dropped. Both gaps are
// structural and self-closing — see doc.go.
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
				t.Skip("blocked: system.go does not exist yet — which wire role a rendered system segment " +
					"uses (system vs developer) is an open decision this slice did not pick silently (see " +
					"doc.go's Wire-shape provenance section, claim 1); a marker-drop proof over unrendered " +
					"system-instruction content would be vacuous under AI-11.3's own S-ACB-035")

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
