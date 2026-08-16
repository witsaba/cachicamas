# Delta for `agent-permission-protocol` — AG-11 scopes the `failure.go` no-edit clause

> **Change**: `cachicamas-agent-turn-termination` · **AG-11** (Layer 2, Wave 2, milestone 11 of 24), `0003:1113-1176`
> **Modifies**: `agent-permission-protocol` (`openspec/specs/agent-permission-protocol/spec.md`, AG-10 PR #171) — `R-APP-012` only.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. The archive step REPLACES the requirement block in the main spec with the MODIFIED block below; full-block preservation is mandatory.
> **Why this delta is mandatory**: `R-APP-012` names `failure.go` in its normative "MUST NOT modify" list (`spec.md:118`). AG-11 adds `PartialOutput()` to `failure.go` (`R-ATT-006`), so the requirement must be amended rather than quietly broken. **Note the asymmetry with `R-LSK-004`**: this list names `failure.go` but does **not** name `turn_events.go`, so this delta releases `failure.go` only — `turn_events.go` is released by the `agent-loop-skeleton` delta alone.

## MODIFIED Requirements

### Requirement: Substrate preservation, 7th consecutive milestone (D8, NFR-TLS-003 carry) — `R-APP-012`

The system MUST NOT modify any of `event.go`, `event_descriptor.go`, `stream_check.go`, `sequence.go`, `import_boundary_test.go`, `backend/agent/go.mod`, `backend/agent/go.sum`, `Makefile`, or `.golangci.yml`. The substrate-untouched guards (`TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched`) SHALL widen their allowlists by exactly four exact-filename suffixes for the files AG-10 introduces — `permission_protocol.go`, `permission_protocol_test.go`, `loop_permission_e2e_test.go`, and `permission_policy_helpers_test.go` — with no wildcard, prefix, or directory-level widening.

`failure.go` was a member of this list at AG-10 and is **released for AG-11 only**, and only for the single addition `R-ATT-006` requires: a `PartialOutput() bool` accessor mirroring the nil-safe shape of `Category()` (`failure.go:44`), `Delivery()` (`:54`) and `Retryable()` (`:64`), delegating unchanged to `(*ai.Failure).PartialOutput()` (`provider_failure.go:515-520`). `NewFailure`'s nil rejection (`failure.go:33-38`) and the "AG-04 registers no separate error kind; failures ride the typed outcomes" rule (`failure.go:6-7`) MUST NOT change. Every other file above remains forbidden for AG-11, and this release does not extend to any later milestone without its own recorded delta.

AG-11 SHALL widen both guards' allowlists further by **exact filename suffixes only** — `failure.go`, `turn_events.go`, and one entry per test file AG-11 introduces — with no wildcard, no prefix match, and no directory-level widening, and the two guards' entry sets SHALL remain identical to each other. `turn_events.go` is listed here only because the guards enumerate changed files, not because this requirement ever forbade it; the normative release of `turn_events.go` lives in the `agent-loop-skeleton` delta (`R-LSK-004`).

(Previously: `failure.go` was listed as forbidden without exception, and the only widening rule described was AG-10's four newly-created files.)

#### Scenarios

- **S-APP-013** — Substrate byte-unchanged against the merge base. Given the merge base of the AG-10 branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agent/` and over `go.mod`/`go.sum`, then only allowlisted files differ, every listed substrate file is byte-unchanged, the `go.mod`/`go.sum` diff is empty, the every-kind-constructible guard still passes at 25 kinds (AG-10 adds zero), and both substrate guards pass. Verified by `TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched`, corroborated by the merge-base diff (11 of 44 `.go` files changed, all allowlisted; 33 byte-unchanged).
- **S-APP-014** — AG-11's release of `failure.go` is bounded and exact. Given the merge base of the AG-11 branch with `origin/main`, when `git diff` is taken over `backend/agent/src/agent/failure.go`, then the only change is the addition of the nil-safe `PartialOutput()` accessor — `NewFailure`, `Category`, `Delivery`, `Retryable` and `Unwrap` are byte-unchanged; and when the two guards' allowlists are compared, then they carry an identical set of exact-filename entries with no wildcard, prefix, or directory pattern; and every file still named forbidden by this requirement is byte-unchanged; and the every-kind-constructible guard still passes at 25 kinds (AG-11 adds zero). Cross-referenced to `R-ATT-006` and `R-ATT-009`.
