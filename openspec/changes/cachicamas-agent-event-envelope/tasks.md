# Tasks: `cachicamas-agent-event-envelope` (AG-04)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1400–2200 (500–750 production across `event.go`, `event_descriptor.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `failure.go`, `stream_check.go`, `doc.go`; 900–1500 test across 4 new `_test.go` files + `doc_contract_guard_test.go` amendment) |
| 400-line budget risk | High (against default) / Medium-High (against this run's 1000-line budget) |
| Chained PRs recommended | No — four nodes are causally ordered (AG-04.2/.3 both depend on AG-04.1's envelope; AG-04.4 depends on both AG-04.2 and AG-04.3) and `doc.go`/`doc_contract_guard_test.go` must land in the same commit (AD-3) — no safe chain boundary |
| Suggested split | Single PR, `size:exception`, four internal commits (one per node) |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

Rationale: explore.md § 7 forecasts 1400–2200 lines, 1.4–2.2x AG-03's already-over-ceiling 1005 lines, because AG-04 is the first milestone shipping actual production Go (envelope, kind registry, per-lane ordering, run/turn payloads, typed-failure wrap, a two-level scope-engine validator) against 11 charter scenarios (51 spec scenario ids) each needing its own RED. The user pre-authorized `size:exception` against a 1000-line budget for exactly this shape of change (session preflight, proposal settled input 6); splitting is not proposed because AG-04.2/.3 both depend on AG-04.1 and AG-04.4 depends on both, and `doc.go`'s `L2C-04` row must land with its guard-table entry in one commit (AD-3) — any chain boundary would either duplicate the envelope across PRs or separate a guarded row from its check.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 (whole change, 4 commits) | Envelope + ordering + run/turn lifecycle + invariant pins + guard, all behind the stream-contract validator | PR 1 (only) | `cd backend/agent && go test -race -v ./src/agent/... ./src/ai/...` | `cd backend/agent && make test && make lint` | Delete `backend/agent/src/agent/*.go` production+test files added by this change (keep `doc.go` at its post-AG-03 3-row state, `import_boundary_test.go`, `ambient_authority_test.go` untouched); revert `doc.go`/`doc_contract_guard_test.go` to AG-03's merged state; per proposal Rollback plan levels 1–3 |

## Phase 1: AG-04.1 — Envelope and validation (leaf)

- [x] 1.1 Create `sequence.go`: `Sequence` (uint64, 1-based, zero = never stamped), `LaneStamper` (`struct{ last Sequence }`, zero-value ready, `Stamp(Event) Event`); doc comment states the one-writer-per-lane precondition, citing AG-01 § 5 (AD-5). (R-AEV-002)
- [x] 1.2 Create `event_descriptor.go`: `EventDescriptor{Bracket, Placement, Cardinality, Terminal}`, `BracketRole`, `Placement`, `Cardinality`; document Layer 2's six-step "adding a kind" procedure in the file comment (AD-1; supports R-AEV-010's S-AEV-091, closed later in Phase 4).
- [x] 1.3 Create `event.go`: `EventKind` (4 constants: RunStart, RunEnd, TurnStart, TurnEnd), `RunID`, `TurnID`, `Event` (unexported payload/seq/run/turn/parent), `Kind()` derivation (nil payload → 0, never a member), `Sequence()`, `Run()`, `Turn() (TurnID, bool)`, `Parent() (RunID, bool)`, `CheckEmit`, registry table pairing kind → descriptor, `EventKinds()`. (R-AEV-001, R-AEV-003) **Deviation** (recorded in apply-progress.md): also created `run_events.go`'s `RunStart` payload + `NewRunStart`/`NewDelegatedRunStart` in this phase, ahead of design's AG-04.2 node attribution — AG-04.1's own scenarios (kind derivation, parent-before-delegation) need at least one constructible event kind, mirroring the same reconciliation tasks.md already documents for stream_check.go.
- [x] 1.4 Create `stream_check.go` with the minimal `CheckStream`/`StreamReport` contiguity core only (Data Flow rule 1: `seq != seq_expect` violation) — the two-level scope engine (rules 2-6) lands in Phase 2 (AD-1 divergence 1; design Testing Strategy AG-04.1 row explicitly needs "minimal `CheckStream` contiguity core" here for S-AEV-010/011).
- [x] 1.5 Add ordering-invariants prose to `doc.go`, below the L2C table, NOT as a guarded row (AD-3): states ordering is per-consumer-lane, independent, contiguous, 1-based, and that Layer 2's counter is not `ai.Sequence`. (R-AEV-011 AG-04.1 part, S-AEV-100)
- [x] 1.6 **RED** — S-AEV-001/002/003: in `envelope_test.go` (new, `package agent_test`), write failing tests for kind-derives-from-payload, nil-payload validation failure (names missing payload), mismatched-payload validation failure (names offered kind + payload).
- [x] 1.7 **RED** — S-AEV-004/005: failing tests for external-package identity reads (run/turn/parent, no build tag, no internal import) and for the closed kind vocabulary containing no `ai.EventKind` alias/re-export.
- [x] 1.8 **RED** — S-AEV-010/011: failing tests stamping two independent hand-built streams via separate `LaneStamper`s from concurrently running goroutines, checked under `-race` and by `CheckStream`; each must be contiguous and 1-based, unaffected by the other.
- [x] 1.9 **RED** — S-AEV-012/013: failing tests for a stream starting at a non-1 value and a stream with a gap/repeat; confirm rejection names the rule (012) and the offending position (013).
- [x] 1.10 **RED** — S-AEV-020/021/022: failing tests for a delegated run-start carrying its parent identity (readable externally), a top-level run-start reporting "no parent" as a distinguishable state, and the exported surface declaring no delegation/subagent mechanism.
- [x] 1.11 **GREEN**: implement 1.1–1.5 until 1.6–1.10 pass; confirm `cd backend/agent && go test ./src/agent/...` green.
- [x] 1.12 Record each RED's failing output (1.6–1.10) before its GREEN, in the change's evidence log.

## Phase 2: AG-04.2 — Run and turn lifecycle + validator scope engine (leaf, depends on Phase 1)

- [x] 2.1 Create `run_events.go`: run-start/run-end unexported payloads, `RunOutcome` (Completed/Interrupted/Failed, zero not a member), `NewRunStart(run RunID)`, `NewDelegatedRunStart(run, parent RunID)`, `NewRunEnd(run RunID, outcome RunOutcome, f *Failure)` (`f` required iff Failed, forbidden otherwise), accessors. (R-AEV-004) **Deviation** (recorded in apply-progress.md): `failure.go` (design AD-2, tasks.md 3.1) was also created in this phase, ahead of its AG-04.3 node attribution — RunEnd/TurnEnd's own signatures need the `*Failure` type to compile. Its dedicated scenarios (S-AEV-070..073) still land at Phase 3 as planned.
- [x] 2.2 Create `turn_events.go`: turn-start/turn-end payloads, `TurnOutcome` (Finished/Aborted), `NewTurnStart(run, turn)`, `NewTurnEnd(run, turn, outcome, f *Failure)`, accessors. (R-AEV-005)
- [x] 2.3 Extend `stream_check.go` with the two-level scope engine (Data Flow rules 2–6: nothing-follows-terminal, bracket transitions for opens-run/closes-run/opens-turn/closes-turn, misplaced-outside-open-bracket, unused Placement/Cardinality seams for AG-05/AG-06); `CheckStream(events []Event) StreamReport` production-exported, no test-only build tag; `(StreamReport).Violation() error` (first violation in stream order). (R-AEV-006, AD-1)
- [x] 2.4 **RED** — S-AEV-030/031/032/033/034: failing tests in `stream_check_test.go` (new) for accepted single-bracket sequence, duplicate run-start (names duplicate), missing run-end (names unclosed bracket), event-after-run-end (names the offending event), sequence not opening with run-start (names the event that preceded the bracket).
- [x] 2.5 **RED** — S-AEV-035: failing test constructing run-end with each of Completed/Interrupted/Failed, confirming typed distinguishability and that no route constructs run-end without an outcome.
- [x] 2.6 **RED** — S-AEV-040/041/042/043: failing tests for accepted non-overlapping turn pairs inside the run bracket, overlapping turn-start (names overlap), turn bracket escaping the run bracket (names escaped bracket), unclosed turn before run-end (names unclosed turn).
- [x] 2.7 **RED** — S-AEV-044: failing test for typed turn-outcome distinguishability (Finished vs Aborted) and no-route-without-outcome.
- [x] 2.8 **RED** — S-AEV-050/051: failing tests proving `CheckStream` compiles and runs from an external package other than `agent`/`agent_test`, no build tag, no test-only import; and that its declaration is exported, non-`_test.go`, and takes a finite ordered slice, not a channel.
- [x] 2.9 **RED** — S-AEV-052/053/054: failing tests for single-rule-violation reporting naming only that rule/position, a fully valid run + two non-overlapping turn brackets accepted with an empty violation set, and a rule-coverage enumeration confirming every expressible rule of R-AEV-001–R-AEV-005 has a corresponding check and none exceeds this spec.
- [x] 2.10 **GREEN**: implement 2.1–2.3 until 2.4–2.9 pass; confirm `go test -race ./src/agent/...` green.
- [x] 2.11 Record each RED's failing output (2.4–2.9) before its GREEN.

## Phase 3: AG-04.3 — Invariant pins + doc-contract row (leaf, depends on Phase 1; may run in parallel with Phase 2)

- [x] 3.1 Create `failure.go`: exported `Failure` wrapping unexported `*ai.Failure`; `NewFailure(f *ai.Failure) (*Failure, error)` (nil rejected); `Category() ai.FailureCategory`, `Delivery() ai.DeliveryPath`, `Retryable() bool`, `Unwrap() error` (returns the wrapped `*ai.Failure`, sentinel chain intact). (R-AEV-008, AD-2)
- [x] 3.2 Add the `L2C-04` row to `doc.go` and its byte-identical entry to `expectedLayer2ContractRows` in `doc_contract_guard_test.go`, in the **same commit** (AD-3; the guard compares row count first, then entry-for-entry in order — split them and `make test` breaks mid-stack, `doc_contract_guard_test.go:105-123`). (R-AEV-011 membership-criterion part, S-AEV-101)
- [x] 3.3 Write the `agent-package-scaffold` spec delta at `openspec/changes/cachicamas-agent-event-envelope/specs/agent-package-scaffold/spec.md`: amend `R-AGP-002`'s expectation-table description from three rows to four, following how AG-03 amended `agent-module-scaffold` in place (Modified capability, AD-3, proposal § Modified capabilities).
- [x] 3.4 **RED** — S-AEV-060/061: failing tests in `invariant_pin_test.go` (new) enumerating every exported route from a delta kind to a payload (none exists at AG-04 — the pin is a structural absence: assert the registry contains exactly the 4 bracket kinds and no accumulated-message constructor is exported), and asserting `doc.go` states the delta rule and that AG-05 inherits the absent route. (R-AEV-007)
- [x] 3.5 **RED** — S-AEV-070/071: failing tests for category+cause reachable as typed values through `Failure`, and pre-stream vs mid-stream `Failure` instances distinguishable via `Delivery()` without parsing text.
- [x] 3.6 **RED** — S-AEV-072/073: failing test mapping every Layer 2 failure category to a declared `ai.FailureCategory` member (all 9, declared in source not inferred); audit assertion across the whole test suite that no test asserts on failure-message-string content as meaning.
- [x] 3.7 **RED** — S-AEV-102 (bite): add a pin test asserting `doc.go`'s raw bytes still contain both the 1.5 ordering-invariants sentence and the 3.2 membership-criterion sentence; scratch-remove/contradict each in turn, confirm the test FAILS naming the missing/divergent statement, record output, then revert. **See Risks** — AD-3 calls the ordering-invariants statement "pinned behaviorally by the validator's rejection scenarios," not textually; this task implements a literal textual pin so S-AEV-102's wording is met without inventing a new guarded-row mechanism AD-3 explicitly excludes for prose.
- [x] 3.8 **GREEN**: implement 3.1–3.2 until 3.4–3.7 pass; confirm `go test ./src/agent/...` green.

## Phase 4: AG-04.4 — Every-kind-constructible guard + scope fence (guard, depends on Phase 2 and Phase 3)

- [ ] 4.1 Create `event_registry_test.go` (new, `package agent_test`): `map[agent.EventKind]eventKindWitness`, two legs per kind (no-arg constructor closure `func() (agent.Event, error)`; payload-accessor closure `func(agent.Event) (any, bool)`), cross-checked bidirectionally against `agent.EventKinds()`.
- [ ] 4.2 **GREEN baseline**: confirm the guard passes over the four registered kinds, reports having constructed at least one instance per kind, each instance passed through the R-AEV-001 validation gate. (S-AEV-080, S-AEV-081)
- [ ] 4.3 **RED** — S-AEV-082 (bite, closing evidence for R-AEV-009): plant a scratch `EventKind` constant + registry row with no constructible payload; run `go test ./src/agent/...`; confirm the guard FAILS naming the offending kind; record the failing output; delete the scratch addition; confirm it is absent from the merged diff.
- [ ] 4.4 **RED** — S-AEV-083: plant a witness-table entry naming a kind the registry does not contain; confirm the guard FAILS naming the unknown entry (proves the cross-check is bidirectional, not containment-only); record, then revert.
- [ ] 4.5 **RED/GREEN** — S-AEV-090: failing-then-passing test enumerating the registered kind set, asserting it is exactly run-start/run-end/turn-start/turn-end and contains no message/tool/permission/cost/delegation/compaction kind under any name. (R-AEV-010 scope fence)
- [ ] 4.6 Confirm S-AEV-091: a test or documentation-reading assertion confirms the 1.2 six-step procedure comment states the ordered steps and that following them requires no edit to the validator's rule engine.
- [ ] 4.7 **RED/GREEN** — S-AEV-092 (extensibility experiment, closing evidence for R-AEV-010): in a scratch addition, add one new kind following the documented six-step procedure exactly; confirm the guard and validator both compile and run without editing their own rule-engine logic; record the result; then revert the scratch kind before merge.
- [ ] 4.8 **GREEN**: confirm `go test -race ./src/agent/...` green with zero scratch artifacts present.

## Phase 5: Regression, evidence close-out, milestone doc update

- [ ] 5.1 Confirm `import_boundary_test.go` and `ambient_authority_test.go` pass with zero changes to their own logic. (NFR-AEV-003)
- [ ] 5.2 Confirm `backend/agent/go.mod`, `go.sum`, `Makefile`, `.golangci.yml` are byte-unchanged (`git diff --stat` shows no entry).
- [ ] 5.3 Run `cd backend/agent && make test` (`go test -race -v ./...`) and `make lint`; confirm both green/clean, recorded pre- and post-change.
- [ ] 5.4 Update the status header of `docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md` (currently "4 of 24 milestones shipped", AG-03 named latest) to "5 of 24", naming AG-04 (event envelope + ordering invariants) as newly landed, following the same pattern as AG-03's own status-header update (`dbd7a33e`).
- [ ] 5.5 Assemble every recorded RED (S-AEV-001…102, tasks 1.6–1.10, 2.4–2.9, 3.4–3.7, 4.3–4.7) into the PR evidence section; write the PR description stating why the change exceeds the default 400-line budget, citing the pre-authorized `size:exception` against the 1000-line budget. (NFR-AEV-004)
- [ ] 5.6 Confirm `git status` clean and no scratch violation file remains in the merged diff.

## Phase 6: Spec promotion (at archive)

- [ ] 6.1 Merge the `agent-package-scaffold` delta (3.3) into `openspec/specs/agent-package-scaffold/spec.md` in place, with an "Amended \<date\> (AG-04)" header note, per that spec's promotion discipline.
- [ ] 6.2 Promote `openspec/changes/cachicamas-agent-event-envelope/specs/agent-event-envelope/spec.md` verbatim as the new `openspec/specs/agent-event-envelope/spec.md` (new capability, no prior spec to merge against).

## Risks

- **`stream_check.go` spans two commits** (Phase 1 creates the contiguity core, Phase 2 adds the scope engine) — design's File Changes table attributes the whole file to node AG-04.2, but its own Testing Strategy table requires the contiguity core to exist in AG-04.1 for S-AEV-010/011. Not a contradiction once both tables are read together, but flagged so the split is not mistaken for scope creep.
- **S-AEV-102 vs AD-3's "pinned behaviorally" language**: the spec's bite scenario literally requires a scratch edit that "removes or contradicts" the ordering-invariants *statement* to fail a named test, while AD-3 states ordering invariants are pinned only behaviorally (via the validator's rejection scenarios), not by a guarded doc row. Task 3.7 resolves this with a literal textual-presence pin test rather than reopening AD-3's guarded-row decision. Recorded, not silently picked.
- **Review budget**: High risk against the default 400-line budget is accepted via pre-authorized `size:exception`; no chained-PR decision is needed before apply.
- **Ownership boundary (R-AEV-010)**: the guard in Phase 4 MUST iterate exactly AG-04's four kinds — a fifth entry for any AG-05/AG-06 family fails the scope fence by design; reviewer checklist item.
