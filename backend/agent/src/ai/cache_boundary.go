// AI-11.2 — the cache-boundary cap and the invalidation-cascade enumerator.
//
// V-REQ-24 (breakpoint cap) and V-REQ-25 (invalidation cascade) share one
// walk: cacheBoundaries below is the single function both the cap rule and
// the exported enumerator call, so a count and a listing cannot disagree
// (design.md § 2 — "the cap and the cascade share one walk").

package ai

import "strconv"

// MaxCacheBoundaries is the documented ceiling on the number of
// cache-boundary markers (V-REQ-23) one request may carry, summed across
// every region.
//
// It is exported for [MaxToolNameLen]'s reason: a caller that generates
// breakpoints needs the ceiling before it constructs rather than after it
// fails. The value is the smallest hard cap any target provider enforces —
// the intersection, not the union, [MaxToolNameLen]'s own argument — because
// raising a cap later is additive and lowering one is breaking.
const MaxCacheBoundaries = 4

// CacheRegion is the closed, ordered vocabulary of the three regions the
// invalidation cascade (V-REQ-25) runs through. Its numeric order is the
// cascade order: tools invalidate first, then the system instruction, then
// messages.
type CacheRegion uint8

// The cache-region vocabulary. Append-only, following role.go's pattern:
// constants from iota + 1, so a zero value is rejected exactly like a wild
// one, and an enumerator that walks the constant space rather than the name
// table (see [CacheRegions]).
const (
	// CacheRegionTools is the tool-declaration region.
	CacheRegionTools CacheRegion = iota + 1

	// CacheRegionSystem is the system-instruction region.
	CacheRegionSystem

	// CacheRegionMessages is the message region.
	CacheRegionMessages

	// cacheRegionFirst and cacheRegionEnd bound the declared constant space.
	// They are not members and are never exported.
	cacheRegionFirst = CacheRegionTools
	cacheRegionEnd   = CacheRegionMessages + 1
)

// cacheRegionNames is the registration table, and the single source of a
// region's rendering and its validity.
//
// A slice indexed by the constant itself rather than a map, for AI-04's
// reason: nothing in this package may let an unordered iteration decide
// anything.
var cacheRegionNames = []string{
	CacheRegionTools:    "tools",
	CacheRegionSystem:   "system",
	CacheRegionMessages: "messages",
}

// CacheRegions returns the cache-region vocabulary in cascade order.
//
// It enumerates the declared constants rather than the table — role.go's
// reason applies verbatim: a member declared without a table entry would be
// invisible to an enumeration derived from the table, and the exhaustiveness
// pin would be decorative.
func CacheRegions() []CacheRegion {
	out := make([]CacheRegion, 0, int(cacheRegionEnd-cacheRegionFirst))
	for r := cacheRegionFirst; r < cacheRegionEnd; r++ {
		out = append(out, r)
	}
	return out
}

// String renders the region.
//
// A member renders as its stable, lowercase, registered name; anything else
// renders as "cacheregion(N)", a shape no parser accepts.
func (r CacheRegion) String() string {
	if name := cacheRegionName(r); name != "" {
		return name
	}
	return "cacheregion(" + strconv.FormatUint(uint64(r), 10) + ")"
}

// cacheRegionName returns a region's registered name, or "" when the region
// has no table entry — which is exactly what makes it not a member.
func cacheRegionName(r CacheRegion) string {
	if int(r) >= len(cacheRegionNames) {
		return ""
	}
	return cacheRegionNames[r]
}

// CacheBoundary names one cache-boundary marker: the region it was placed in
// and its ordinal within that region.
//
// Comparable, so a test may assert a whole sequence with slices.Equal.
type CacheBoundary struct {
	region CacheRegion
	index  int
}

// Region returns the boundary's region.
func (b CacheBoundary) Region() CacheRegion { return b.region }

// Index returns the boundary's ordinal within its region.
func (b CacheBoundary) Index() int { return b.index }

// String renders the boundary as "region[index]", for example "tools[1]".
func (b CacheBoundary) String() string {
	return b.region.String() + "[" + strconv.Itoa(b.index) + "]"
}

// CacheBoundaries returns the request's cache-boundary markers in
// tools → system → messages order (V-REQ-25), ascending by ordinal within
// each region. A fresh slice on every call.
func (r Request) CacheBoundaries() []CacheBoundary {
	return r.options.cacheBoundaries(r.messages)
}

// cacheBoundaries is the one walk: [Request.CacheBoundaries] and
// [requestDraft.cacheBoundaryCapRule] both call it, so a reported count and
// an enforced count cannot disagree (S-ACB-032).
//
// It reads the tool set and the system instruction from the draft — the
// request's own options — and the messages from the parameter, because the
// message sequence is not part of requestDraft: it is NewRequest's own
// parameter, validated separately (design.md § 3.2).
func (d requestDraft) cacheBoundaries(messages []Message) []CacheBoundary {
	var boundaries []CacheBoundary

	if d.hasTools {
		for i, tool := range d.tools.Tools() {
			if tool.IsCacheBoundary() {
				boundaries = append(boundaries, CacheBoundary{region: CacheRegionTools, index: i})
			}
		}
	}
	if d.hasSystem {
		for i, seg := range d.system.Segments() {
			if seg.IsCacheBoundary() {
				boundaries = append(boundaries, CacheBoundary{region: CacheRegionSystem, index: i})
			}
		}
	}
	for i, message := range messages {
		if message.IsCacheBoundary() {
			boundaries = append(boundaries, CacheBoundary{region: CacheRegionMessages, index: i})
		}
	}

	return boundaries
}

// cacheBoundaryCapRule is rule 10 of NewRequest's documented order
// (design.md § 3.3, `explore.md` § 8 row 3 corrected): the total marker
// count across every region must not exceed [MaxCacheBoundaries].
//
// It runs after every rule that admits a region it counts — in particular
// after rule 9, which can still reject the tool set — so a count is never
// taken over a region a later rule then rejects (design.md § 2).
func (d requestDraft) cacheBoundaryCapRule(messages []Message) Rule {
	return func() *Violation {
		if len(d.cacheBoundaries(messages)) > MaxCacheBoundaries {
			return Invalid(ErrOutOfRange, At("cacheBoundaries"))
		}
		return nil
	}
}
