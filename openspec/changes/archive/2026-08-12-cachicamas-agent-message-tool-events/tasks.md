# Tasks: `cachicamas-agent-message-tool-events` (AG-05)

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines | 1500–2200 (backend/agent/ + openspec planning) |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR with 3 internal commits (AG-05.1, AG-05.2, AG-05.3) |
| Delivery strategy | `single-pr` |
| Chain strategy | `size-exception` |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Rationale: `size:exception` pre-authorized this session against 1000-line budget; AG-05.3 reconstruction references both families, so chained PRs are NOT safe. The AG-04 verification report (W1/W2/W3/W4) is the implicit constraint — every behavioral task lands in an external test package (`agent_test`) and the helper is bite-tested RED before the property is GREEN.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 (AG-05.1) | Message family — 6 kinds registered, constructible, kind-segregation invariant | Same PR, commit 1 | `cd backend/agent && make test` (`-run AMT-001\|AMT-002\|AMT-003\|AMT-004`) | N/A — no live producer yet | Revert commit 1: registry rows for 6 message kinds + R-AMT-001..004 implementation removed; R-AMT-005..009 unchanged |
| 2 (AG-05.2) | Tool execution family — 5 kinds registered, three-state typed outcomes, call ordinal | Same PR, commit 2 | `cd backend/agent && make test` (`-run AMT-005\|AMT-006\|AMT-007`) | N/A — no live producer yet | Revert commit 2: registry rows for 5 tool kinds + R-AMT-005..007 implementation removed; R-AMT-008 (reconstruction) unchanged |
| 3 (AG-05.3) | Reconstruction property + scope-fence retightening | Same PR, commit 3 | `cd backend/agent && make test` (`-run AMT-008\|AMT-009`) + `S-AMT-081` 16th-scratch bite | N/A — test-only | Revert commit 3: `reconstruction_test.go` deleted; scope-fence reverts to "exactly 4" |

## Scenario count commitment (W9 lesson from AG-04)

`7 charter Gherkin scenarios in 3 leaves → 15 of 12–18 spec scenarios after per-rule expansion + bite + reconstruction sub-scenarios` (plus 4 bites: S-AMT-021, S-AMT-071, S-AMT-072, S-AMT-081). This count MUST be restated identically in `apply-progress.md`. AG-04's W9 finding was a scenario-count error that propagated three artifacts deep before verify caught it.

## Phase 1: AG-05.1 — Message family (6 kinds, R-AMT-001..004)

- [x] 1.1 RED: extend `event_registry_test.go` witness-table to assert exactly 15 kinds (4 AG-04 + 11 AG-05); run, expect FAIL
- [x] 1.2 GREEN: register `message_start_text`, `message_delta_text`, `message_end_text` in `message_text.go` (payload + ctors + 3 `EventDescriptor` rows + kind constants, co-located); witness passes at 10
- [x] 1.3 RED: write `message_text_test.go` S-AMT-001 (kind segregation) + S-AMT-002 (no reasoning field in text payload); run, expect FAIL
- [x] 1.4 GREEN: implement text bracket constructors satisfying S-AMT-001 + S-AMT-002
- [x] 1.5 RED: write S-AMT-010 (reasoning bracket lifecycle independent of text); run, expect FAIL
- [x] 1.6 GREEN: register `message_start_reasoning`, `message_delta_reasoning`, `message_end_reasoning` in `message_reasoning.go`; witness at 13; S-AMT-010 passes
- [x] 1.7 RED: write S-AMT-020 (deltas carry index + fragment only, no snapshot); run, expect FAIL
- [x] 1.8 RED — bite: write S-AMT-021 in `invariant_pin_test.go` (no delta → accumulated-payload route); run, expect FAIL (proves the pin)
- [x] 1.9 GREEN: implement delta constructor surface; S-AMT-020 + S-AMT-021 pass (satisfies R-AMT-003)
- [x] 1.10 RED: write S-AMT-030 (whole vs fragmented message reconstruct equally); run, expect FAIL
- [x] 1.11 GREEN: implement `reconstructMessage` helper (test-only); S-AMT-030 passes (satisfies R-AMT-004)
- [x] 1.12 `make test`, `golangci-lint cache clean`, `make lint` green for AG-05.1

## Phase 2: AG-05.2 — Tool execution family (5 kinds, R-AMT-005..007)

- [x] 2.1 RED: write `tool_event_test.go` S-AMT-040 (ToolStart carries identity+name+args; ToolProgress indexed); run, expect FAIL
- [x] 2.2 GREEN: register `tool_start`, `tool_progress` in `tool_event.go`; S-AMT-040 passes
- [x] 2.3 RED: write S-AMT-050 (three typed outcomes distinct by kind — no convention inference); run, expect FAIL
- [x] 2.4 RED: write S-AMT-051 (typed-failure wrap required: `tool_end_execution_failure` without `Failure` is REJECTED); run, expect FAIL
- [x] 2.5 GREEN: register `tool_end_success`, `tool_end_result_failure`, `tool_end_execution_failure`; reuse `agent.Failure` from R-AEV-008; S-AMT-050 + S-AMT-051 pass (satisfies R-AMT-006)
- [x] 2.6 RED: write S-AMT-060 (call ordinal correlates to call order regardless of completion — payload field, NOT envelope); run, expect FAIL
- [x] 2.7 GREEN: add `ordinal uint32` to `ToolStart`/`ToolProgress`/`ToolEnd*` payloads; S-AMT-060 passes (satisfies R-AMT-007; AD-3; R-13 → doc 0002 AI-30)
- [x] 2.8 `make test`, `golangci-lint cache clean`, `make lint` green for AG-05.2

## Phase 3: AG-05.3 — Reconstruction property + scope-fence retightening (R-AMT-008, R-AMT-009)

- [x] 3.1 RED — bite: write S-AMT-071 (drop-a-delta: reconstruction fails); run, expect FAIL
- [x] 3.2 RED — bite: write S-AMT-072 (double-a-delta: reconstruction fails); run, expect FAIL
- [x] 3.3 GREEN: implement `reconstructToolOutcome` helper in `reconstruction_test.go` (matches AD-4: helpers INSIDE test file); bites fail as expected
- [x] 3.4 RED: write S-AMT-070 (interleaved 2-message + 2-tool reconstruction — independent and complete); run, expect FAIL
- [x] 3.5 GREEN: refactor helper to satisfy S-AMT-070; S-AMT-070 GREEN; bites still bite RED; S-AMT-071 + S-AMT-072 remain expected-fail (satisfies R-AMT-008)
- [x] 3.6 RED — bite: plant a 16th scratch kind following the six-step procedure (in a scratch test file); write S-AMT-081 (16th kind fails by count); run, expect FAIL
- [x] 3.7 GREEN: update `S-AEV-090` scope-fence in `event_registry_test.go` from "exactly 4" to "exactly 15" (same commit as new kinds); S-AMT-081 bites RED; revert the scratch file (NOT in merged diff)
- [x] 3.8 update `doc_contract_guard_test.go` so the doc-matrix guard iterates all 15 kinds (auto-extends; no logic change)
- [x] 3.9 update `event_descriptor.go` six-step procedure doc note with `Terminal: false` explicit (W3 latent-trap guard; rule engine unchanged)
- [x] 3.10 extend `event.go` `EventKind` const block + `eventRegistry` table (the 11 new rows)
- [x] 3.11 update `doc.go` to carry `L2C-05` prose alongside `L2C-04`
- [x] 3.12 update `agent_test_helpers_test.go` to reuse `requireViolationPosition`; add AG-05 helper signatures
- [x] 3.13 `make test`, `golangci-lint cache clean`, `make lint` green for AG-05.3

## Phase 4: Final gates + cleanup

- [x] 4.1 `golangci-lint cache clean && make lint` (full pass; cite `6c821c0a` precedent)
- [x] 4.2 `make build` clean
- [x] 4.3 `make vuln-check` clean (NOT in `make all`; runs explicitly)
- [x] 4.4 AG-03 boundary guards pass with zero changes: `import_boundary_test.go`, `ambient_authority_test.go`
- [x] 4.5 `go.mod` / `go.sum` byte-unchanged from main
- [x] 4.6 `stream_check.go` / `failure.go` / `sequence.go` byte-unchanged (the AG-05 bet)
- [x] 4.7 Single PR opened with `size:exception` documented in PR body (citing braejan's standing instruction in AG-04 session_summary)
- [x] 4.8 Worktree `cachicamas-worktrees/agent-layer2-wave1-ag05` left in place for verify phase (per orchestrator's instruction); branch `feat/agent-layer2-wave1-ag05` retained locally + remotely
- [x] 4.9 Apply-progress saved to Engram with scenario count `7 charter → 15 spec + 4 bites` restated identically

## Risks

- W3 latent trap (W3 from AG-04): any new kind declaring `Terminal: true` with `BracketRoleNone` would slip past Phase 1 + Phase 2. Mitigated by 3.9 (six-step procedure restated).
- Vacuous helper (proposal risk 2): `reconstructMessage`/`reconstructToolOutcome` could pass trivially. Mitigated by bites 3.1 + 3.2 RED-first.
- Scenario count drift (W9): 7 charter → 15 spec + 4 bites must be restated identically in `apply-progress.md`.
- Lint cache phantom finding (AG-04 gotcha): `golangci-lint cache clean` before every lint gate (cite `6c821c0a`).
- `make vuln-check` not in `make all` (Engram `obs #2944`): run explicitly at 4.3.
