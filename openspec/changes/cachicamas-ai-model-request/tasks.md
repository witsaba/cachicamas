# Tasks — the normalized request core

> **Change**: `cachicamas-ai-model-request`
> **Milestone**: AI-10 · **Nodes**: AI-10.1 … AI-10.6, all `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-model-request/spec.md`, `design.md`
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-1`
> **Depends on**: AI-00 … AI-09 merged
> **Blocks**: AI-11, AI-12, AI-20, AI-26
> **Evidence gate**: recorded green `make test` and clean `make lint` in `backend/agent/` (`go test -race -v ./...`)

---

## Entry point for the resuming agent

This half of the milestone implements **AI-10.1 and AI-10.2 only**. **AI-10.3, AI-10.4, AI-10.5 and AI-10.6 are not implemented.**

The resuming agent starts at **§ Phase AI-10.3**, below. Every box in phases AI-10.3 … AI-10.6 is unchecked, and every decision those phases need is already made in `design.md` §§ 5–12 and marked `[provisional]` there. A `[provisional]` decision may be changed only by recording the reason in `design.md` **before** writing the test; absent a recorded reason, implement what is written.

The first thing AI-10.3 does is append `ErrMisplaced` to `validation.go` **and both of its registry mirrors in the same commit**. `validation_registry_internal_test.go` fails and says so if only one mirror moves.

## Node types and what they close on

| Node | Type | Closes on |
| --- | --- | --- |
| AI-10.1 … AI-10.6 | `[leaf]` | Every test-list item taken red → green → refactored, **in order**, with both outputs recorded here |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so `design.md` § 13 applies: land the narrowest thing that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

Discovered cases are **appended** to the owning leaf's test list, never substituted for a planned one and never pruned to fit the budget.

### Evidence provenance

Every transcript in this file is **observed** — pasted from a command run while implementing the item it sits under. Where a transcript is trimmed for length that is said at the point of the trim. Nothing here is reconstructed from a commit after the fact, and nothing is written before the command that produced it was run.

---

## Review Workload Forecast

Recorded before the work, with actuals filled in as each phase closed.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `spec.md`, `design.md`, `tasks.md` (on top of the landed `explore.md`, `proposal.md`) | ~900 prose | _filled at close_ | Low | 35 min |
| AI-10.1 walking skeleton | `request.go`, `request_test.go` | ~200 Go prod, ~400 Go test | _filled at close_ | **High** — every later request milestone and every adapter inherits the shape | 45 min |
| AI-10.2 segmented system instruction | `system_instruction.go`, `system_instruction_test.go`, `request.go`, `request_test.go` | ~120 Go prod, ~300 Go test | _filled at close_ | Medium — the shape AI-11.1 attaches markers to | 30 min |
| AI-10.3 … AI-10.6 | not implemented in this half | ~350 Go prod, ~900 Go test | — | Medium-High | ~90 min |

### Budget reassessment — split trigger 4 fired before the first test was written

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when a milestone's projected diff, tests included, pushes it past the review budget. **It does, by a wide margin, and `explore.md` § 7 recorded it before a line of Go was written.** The reassessment:

- **Split trigger 1 does not fire on the milestone as a whole**, but the milestone *is* six leaves rather than one behavior, and the leaves are individually shippable. So the mitigation is not to cut the test list but to **make the leaf boundary the commit boundary**, which is where doc 0002 puts the PR-chain boundary anyway. The chain exists in history and needs no rework to be reviewed in slices.
- **The milestone is worked as two chained halves at the .2/.3 line.** `proposal.md` forecast the split at .3/.4; it was moved one leaf earlier during execution because AI-10.3 alone carries three cross-region dispositions plus an appended rule class with two registry mirrors, and two prior agents on this milestone died on transcript size. The boundary moved to protect the evidence, not to cut scope: AI-10.3's plan is unchanged and complete in `design.md` §§ 5–7.
- **Production is small; documentation and tests are where the weight is.** The exported surface the rest of the layer inherits is one request type, one system-instruction type, one segment type and five option constructors. What is large is the contract documentation — where the reasoning is put in front of the person who adds the sixth generation option — and the tests.
- **Nothing is cut to fit.** Cases discovered during implementation are appended, never pruned.

---

## Phase AI-10.1 — walking skeleton: a minimal valid request `[leaf]`

**Deliverable:** `backend/agent/src/ai/request.go`, `backend/agent/src/ai/request_test.go`.
**Spec:** `R-AMR-001` … `R-AMR-004`, `R-AMR-017`. **Design:** §§ 2, 4, 8.

- [ ] **Item 1** — WHEN a request is constructed with a model identity and one user text message THEN it validates, and both read back exactly **from an external package**.
- [ ] **Item 2** — WHEN the model identity is empty, or there are no messages, THEN validation fails with AI-04 sentinels.
- [ ] **Item 3** — Generation options carry through construction and read back — the neutral vocabulary decided in AI-01, no more.

_Appended items are added below as they are discovered._

---

## Phase AI-10.2 — segmented system instruction `[leaf]`

**Deliverable:** `backend/agent/src/ai/system_instruction.go`, `system_instruction_test.go`; one option constructor and one accessor added to `request.go`.
**Spec:** `R-AMR-005`, `R-AMR-006`, `R-AMR-007`. **Design:** § 3, § 12.1.

- [ ] **Item 1** — WHEN a request carries a system instruction as **ordered segments** THEN segment order and content round-trip exactly.
- [ ] **Item 2** — The single-segment convenience path produces a request **indistinguishable** from one built segment-by-segment with one segment.
- [ ] **Item 3** — An absent system instruction is legal and **distinguishable from one empty segment**.
- [ ] **Item 4** — Segment construction rules fail through AI-04 sentinels (empty segment, whitespace-only segment).

---

## Phase AI-10.3 — messages, tools and tool choice on the request `[leaf]` — **NOT IMPLEMENTED**

**Deliverable:** `request.go` extended; `validation.go` plus **both** registry mirrors; `request_test.go` extended.
**Spec:** `R-AMR-008` … `R-AMR-012`. **Design:** §§ 5, 6, 7, and § 4 rows 6–9.

- [ ] **Item 0** *(prerequisite, do this first)* — Append `ErrMisplaced` to `validation.go`'s class set **and** to `ruleClasses`, **and** update `validation_registry_internal_test.go` and `validation_test.go` in the **same commit**. The internal guard fails and says so if only one mirror moves. `design.md` § 5.3 carries the doc comment to write.
- [ ] **Item 1** — Message order and intra-message content order are preserved exactly through construction and readback.
- [ ] **Item 2** — The tool set and tool choice attach to the request, and AI-08.3's cross-validation runs at the request boundary too. **Call `ToolChoice.ValidateAgainst(ToolSet)`; reimplement none of its three rules.**
- [ ] **Item 3** — Role-versus-content-kind rules are enforced from `design.md` § 5.1's table, all twelve cells, in both directions, reporting `ErrMisplaced`.
- [ ] **Item 4** — An orphan tool result fails with `ErrUnresolvedReference`; an orphan tool **call** succeeds; a duplicate call identity fails with `ErrDuplicate` at the second occurrence; a result appearing before its call **succeeds**, pinning `design.md` § 6.3's deliberate non-decision.

---

## Phase AI-10.4 — validation happens once, before I/O `[leaf]` — **NOT IMPLEMENTED**

**Spec:** `R-AMR-013`, `R-AMR-014`. **Design:** § 9.

- [ ] **Item 1** — The first failure in the documented order is reported, identically across runs.
- [ ] **Item 2** — Validation is total over the regions.
- [ ] **Item 3** — The request path's dependency closure contains no network and no filesystem package, asserted mechanically in the AI-00.3 guard style. Extend `import_boundary_test.go` rather than adding a parallel guard.

---

## Phase AI-10.5 — whole-request round trip `[leaf]` — **NOT IMPLEMENTED**

**Deliverable:** a new test file under `backend/agent/src/agenttest/`.
**Spec:** `R-AMR-015`. **Design:** § 10.

- [ ] **Item 1** — An external-package test walks a request holding every part variant plus segments, tools and options, and reconstructs an **equal** request from what it read.
- [ ] **Item 2** *(pin)* — The walk's kind handling is exhaustive over `PartKinds()`: a kind added without a readable accessor fails this pin.

---

## Phase AI-10.6 — immutability `[leaf]` — **NOT IMPLEMENTED**

**Spec:** `R-AMR-016`. **Design:** § 11.

- [ ] **Item 1** — Mutating anything a reader returned leaves the request observably unchanged.
- [ ] **Item 2** — Mutating the values passed to the constructor leaves the request observably unchanged.
- [ ] **Item 3** — Two requests built from identical inputs compare equal by `design.md` § 11.2's documented equality, and neither is affected by operations on the other. Decide there whether to export `Equal`; the recorded default is yes.

---

## Evidence

_Filled at the close of each implemented phase, from observed command output._

### Guards

- `go.mod` must carry **zero requires**.
- Both AI-00 import guards must pass.
- `validation.go` is **unchanged by this half**; `ErrMisplaced` is AI-10.3's first task.

---

## Doc 0002 amendments

_None so far._ The revert-and-record clause is triggered only when a leaf's first red cannot be driven green in small steps. If it fires, the amendment lands in this change as `> **Amended 2026-07-31** …` with the superseded text struck through, and is reported prominently.

## Register amendments

**None made.** One gap is reported in `proposal.md` § "Vocabulary check": `V-REQ-26` names the generation-option concept and its admission test but enumerates no members. `design.md` § 8 carries the table an amendment would cite. `openspec/specs/ai-contract-vocabulary/spec.md` is **not edited by this change**.
