// Package agent is Layer 2 of the cachicamas agent stack: the agent loop.
//
// AG-03 (Layer 2, Wave 1) brings this package into existence and gives it
// two mechanically enforced boundaries — an import guard and a no-ambient-
// authority guard — before it carries any behavior. It exists and declares
// nothing yet: no type, no constant, no function, no variable beyond this
// comment. The event loop itself is out of scope at AG-03 and arrives from
// AG-04 onward (ADR 0005 § D1, § D2).
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
package agent
