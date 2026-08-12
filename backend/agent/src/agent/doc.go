// Package agent is Layer 2 of the cachicamas agent stack: the agent loop.
//
// AG-03 (Layer 2, Wave 1) brought this package into existence and gave it
// two mechanically enforced boundaries — an import guard and a no-ambient-
// authority guard — before it carried any behavior. AG-04 lands the first
// behavior: the event envelope every agent event travels in, its per-lane
// ordering, the run and turn lifecycle families, and the stream-contract
// validator that judges any hand-built sequence against them (ADR 0005 §
// D1, § D2; agent-event-envelope). AG-05 extends the registry with the
// message lifecycle (VL2-EVT-04) and tool execution (VL2-EVT-05) families
// — 11 kinds — and adds the AG-05.3 reconstruction property, biting
// twice on a vacuous helper before going GREEN (agent-message-tool-events).
//
// The layer contract below is machine-checked, not remembered: each row is
// parsed back out of this file's own bytes and diffed against a committed
// expectation table in doc_contract_guard_test.go (S-AGP-012..015). A later
// milestone that appends a guarded paragraph here MUST add its row and its
// table entry in the same pull request, or the guard fails.
//
//	L2C-01	Imports: the Go standard library, github.com/cachicamas/backend/agent/src/ai and its measured transitive closure — nothing else, deny-by-default (ADR 0005 § D1 row 2; import_boundary_test.go).
//	L2C-02	No I/O of its own: no environment read, no filesystem access, no network call, no process spawn (ADR 0005 § D1 row 2; ambient_authority_test.go and import_boundary_test.go).
//	L2C-03	The event stream is the only upward contract: Layer 2 reports exclusively through emitted events; callers observe the stream, they never reach into the loop (AG-00 vocabulary).
//	L2C-04	Stream membership: a fact belongs on the event stream or it does not exist upward — if it is not on the stream, no frontend can render it and no log can reconstruct it. This is the criterion every later event family is judged by (doc 0003 AG-04 acceptance; agent-event-envelope).
//	L2C-05	Message and tool families are reconstructable: a session log carrying a message-text bracket, a reasoning bracket, or a tool-call bracket reconstructs each message and each tool outcome independently and completely, so a frontend can render "what the model said" and "what the tools did" without losing interleaving order (doc 0003 AG-05 acceptance; agent-message-tool-events). Reconstruction helpers are test-only code; the property is the test.
//
// # Ordering invariants (R-AEV-011)
//
// Every event's ordering is stamped per consumer lane: independent,
// contiguous, and 1-based from the first event of that lane. Layer 2's
// ordering counter is its own, agent-level counter — it is not
// ai.Sequence, reused, aliased or extended (S-AGV-021).
//
// This paragraph is AG-04's own documentary obligation; it is not a claim
// that AG-04 closes any envelope invariant of v2 § 4.3 by itself. AG-04
// co-closes invariant 1 with AG-05.1, invariant 2 with AG-19.1, and
// invariant 4 with AG-11.2; it is absent from invariant 3 entirely, which
// AG-01.1 and AG-20.2 close without it (0003:2203, agent-event-envelope's
// Explicit non-requirements table).
//
// # The delta rule (R-AEV-007)
//
// A delta carries an index and the new fragment only, never an
// accumulated snapshot of what came before. AG-04 registers no delta
// kind, so this rule is a pin on the construction surface, not an
// assertion about a message-delta kind: the absence of any route from a
// delta kind to an accumulated-message payload is the mechanism AG-05
// inherits when it registers its own delta kinds. AG-05.1 closes
// envelope invariant 1 jointly with AG-04.3 (0003:2203); the
// co-closure is asserted mechanically by
// TestConstructionSurface_NoRouteFromDeltaKindToAccumulatedPayload
// (invariant_pin_test.go, S-AEV-060 + S-AMT-021).
package agent
