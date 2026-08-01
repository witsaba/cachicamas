// AI-14.2 — the per-stream sequence.
//
// The full contract — the stamper, the sentinel, the cross-stream rule — lands
// with AI-14.2 (R-AEE-007 … R-AEE-010). [Sequence]'s bare type is declared
// here ahead of that, because [Event] (event.go, AI-14.1) holds one: an event
// without a place to carry its sequence is not the type this milestone
// specifies. Declaring the type costs nothing structurally — it has no
// behavior yet — and keeps the field's type real from Event's first commit
// rather than standing in for it with a placeholder that would need to change
// shape later.
package ai

// Sequence is the per-stream position AI-14.2 assigns to an event (V-STR-13).
//
// It is 1-based and contiguous within one stream (R-AEE-007); the zero value
// is the documented sentinel for "never stamped" (R-AEE-010) — see
// [Stamper] for the assigning half of the contract.
type Sequence uint64
