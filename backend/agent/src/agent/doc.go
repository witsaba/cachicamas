// Package agent is Layer 2 of the cachicamas agent stack: the agent loop.
//
// AG-03 (Layer 2, Wave 1) brought this package into existence and gave it
// two mechanically enforced boundaries — an import guard and a no-ambient-
// authority guard — before it carried any behavior. AG-04 lands the first
// behavior: the event envelope every agent event travels in, its per-lane
// ordering, the run and turn lifecycle families, and the stream-contract
// validator that judges any hand-built sequence against them (ADR 0005 §
// D1, § D2; agent-event-envelope).
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
package agent
