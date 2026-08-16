# Spec — Agent permission protocol (`agent-permission-protocol`)

**Change**: `cachicamas-agent-permission-protocol` · **AG-10** (Layer 2 Wave 2, 10/24; doc 0003 lines 1005–1111) · **Closes**: G1 protocol half (R-10); v2 § 6 seam 2 · **Depends on**: AG-06.1, AG-09 · **Format**: RFC 2119 + Given/When/Then · **IDs**: `R-APP-NNN` / `S-APP-NNN`.

## Purpose

Wire the ask–suspend–resume seam `doc 0001 § 4.1` requires: per-call decision gate, parked-set suspension that holds one call while siblings continue, four typed outcomes (`AllowOnce | AllowAlways | Deny | ModifyInput`) on the stream, and cancel discipline resolving parked calls into typed aborts. Layer 2 owns the protocol; Layer 3 owns the answer (`doc 0001 § 5.1`).

## Requirements

| ID | Statement | Scenarios |
|---|---|---|
| **R-APP-001** Per-call gate | System MUST ask an injected `PermissionPolicy` before every scheduled call. Sync verdicts proceed; `Defer` parks. | S-APP-001 sync `AllowOnce` A → no event, executes. S-APP-002 `Defer` A + `AllowOnce` B → A parks, B completes. |
| **R-APP-002** Emit before park | On `Defer`, MUST emit `permission_decision_required` THEN park on per-call `chan` keyed by `callID`. Emission reaches `sink` before the parked wait blocks (R-AGE-006 narrowing); parked `Result` slot pre-populated with `callID` (R-TLS-009), re-populated on wake. | S-APP-003 one deferred call → `permission_decision_required` on `sink` BEFORE the parked goroutine blocks. |
| **R-APP-003** Typed rejection | Wake to `callID` not in parked set OR already decided MUST be rejected as a typed protocol error. MUST NOT silently drop. | S-APP-004 wake to unknown `callID-X` → typed rejection, no parked call touched. |
| **R-APP-004** AllowOnce | On `AllowOnce`, SHALL execute and emit `permission_decision_made{outcome=AllowOnce}`. | S-APP-005 sync `AllowOnce` → call executes AND `decision_made{outcome=AllowOnce}` on stream. |
| **R-APP-005** Deny typed | On `Deny`, SHALL skip execution and populate ordinal `Result` slot with `Result{Outcome: ExecutionFailure, Failure: <typed denial>}` so the model sees the denial. | S-APP-006 `Deny` (sync OR wake) → ordinal slot carries `ExecutionFailure` with typed `*Failure` AND denial visible to model. |
| **R-APP-006** ModifyInput transparent | On `ModifyInput`, SHALL execute with modified arguments; `permission_decision_made` MUST carry them and `ToolStart.Arguments()` MUST byte-equal them. | S-APP-007 `ModifyInput` with `modifiedArguments={"cmd":"ls"}` → `tool_start.Arguments() == decision_made.ModifiedArguments()` byte-for-byte AND call executes with modified args. |
| **R-APP-007** AllowAlways + Remember gate | On `AllowAlways`, SHALL execute and MUST invoke `Policy.Remember(ctx, toolName, AllowAlways)`; `true` emits `permission_resolution_remembered`, `false` suppresses (preserves `CardinalityAtMostOne`). | S-APP-008 `Remember=true` → execute + one emission; `Remember=false` → execute + no emission. |
| **R-APP-008** Sibling isolation | Parked call MUST NOT block sibling calls or downstream emission channel. | S-APP-009 A parked, B read-class `AllowOnce` → B's `Result` in ordinal slot regardless of A. |
| **R-APP-009** Cancel mid-park | On run `context.Context` cancel while calls parked, MUST walk parked set, populate each as `Result{Outcome: ExecutionFailure, Failure: <typed abort>}`, close channels, return; rejoin fully populated; no goroutine waits forever. | S-APP-010 two parked, context cancelled → both ordinal slots carry `ExecutionFailure` typed abort AND `Schedule` returns AND no leak. |
| **R-APP-010** Remembered suppresses asks | When `Policy.Remember` returned `true` for a `toolName`, identical subsequent calls in the same run MUST NOT be asked; stream shows initial `permission_resolution_remembered` then unasked executions. | S-APP-011 `Remember=true` for `toolName="read_file"` after first call → second identical call NOT consulted AND no `permission_decision_required`. |
| **R-APP-011** Layer 2 owns protocol | `PermissionPolicy` port MUST be Layer 2's contract; Layer 3 supplies implementations; Layer 2 MUST NOT define rule sets or mode flags. | S-APP-012 interface in `backend/agent/src/agent/permission_protocol.go` → Layer 3 impl consumed without naming the type. |
| **R-APP-012** Substrate preservation (NFR-TLS-003 7th carry) | MUST NOT edit `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `event.go`, `go.mod`, `go.sum`, `Makefile`, `.golangci.yml`, `import_boundary_test.go`. `TestTurn_SubstrateUntouched` widens to exclude `permission_protocol.go` and `permission_protocol_test.go`. | S-APP-013 substrate filter baseline → AG-10 lands excluding the two new files AND nine listed files remain byte-unchanged. |

## NFRs (one block)

External verifiability via `make test` in `package agent_test`; determinism + `-race` cleanliness; substrate byte-unchanged (R-APP-012); parked-set invariant bite (R-TLS-008 carry); cancellation bite under `-race`; `CardinalityAtMostOne` bite (R-APE-003 / S-APE-082 carry). **Non-reqs**: policy content / mode flag (doc 0004 CO-03); cross-session `Remember` persistence (CO-16.1); subagent tool scope (AG-19.3); AG-14's full cancellation tree; new `PermissionOutcome` member or new event kind (AG-06.1 stays byte-clean). **Depends/Blocks**: AG-06.1/AG-09/AG-01 → AG-13/AG-19. **Verify**: `make test` + `make lint` + `make build` + `make vuln-check` green; substrate-untouched via `AG09_BASE_REF` fallback; bites RED-recorded; AG-09 invariants preserved (25 kinds, `S-AEV-090` scope-fence). **Accept**: every `S-APP-001..013` evidenced; 4 Gherkin leaves covered.

## Traceability

R-APP-001/002/003 → AG-10.1; R-APP-004/005/006/007 → AG-10.2 (007 also AG-10.4); R-APP-008/009 → AG-10.3; R-APP-010 → AG-10.4; R-APP-011 boundary; R-APP-012 substrate.
