# Tasks — finish reasons and usage

> **Change**: `cachicamas-ai-completion-metadata`
> **Milestone**: AI-13 · **Nodes**: AI-13.1, AI-13.2, AI-13.3, AI-13.4 — all `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-completion-metadata/spec.md`, `design.md`
> **Branch**: `feat/ai-13-completion-metadata`
> **Depends on**: AI-00 … AI-03 merged; AI-04 merged on the Wave 1 branch
> **Blocks**: AI-15.2, AI-31
> **Evidence gate**: recorded green `make test` in `backend/agent/` (`go test -race -v ./...`)

---

## Node types and what they mean for this list

All four nodes are `[leaf]`: each closes only when every test-list item is taken red → green → refactored, **in order**. Two items are `*(pin)*` — AI-13.1 item 5 and AI-13.4 item 2 — which doc 0002 exempts from red-first and does **not** exempt from being mechanical. Each must be recorded biting against a deliberate scratch violation.

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so every red step follows `design.md` § 8: land the narrowest stub that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.

All output blocks below are verbatim from `go test -race`, pasted after the run. Nothing in this file is written before the run it records.

---

## Order of the two chains, and why

doc 0002's AI-13 graph has two independent chains: `AI-13.1 → AI-13.2` (finish reasons) and `AI-13.3 → AI-13.4` (usage). AI-13.3 is explicitly marked *parallel with* AI-13.1, so either order is legal. **The finish-reason chain runs first.**

The defence, so it is a choice and not an accident:

1. **It is the milestone's headline.** AI-13 sits where it does because it absorbs **G12(c)** — doc 0001 § 3.2's row and § 7's `G12` entry both name refusal and pause as the reason the milestone cannot wait. Landing the charter's own reason first means the riskiest downstream commitment (the vocabulary AI-31.1 must map every vendor onto, and AI-40 freezes) is settled while the review budget is untouched.
2. **It is the walking skeleton.** doc 0002 § "Ordering inside a milestone" asks for the thinnest end-to-end path first. A provider string entering `NormalizeFinishReason` and leaving as a validated vocabulary value is that path; the usage record is a data structure with no path through it.
3. **The usage chain carries the decision that most needs a fresh reviewer.** `design.md` § 5.3's inclusive-versus-exclusive call is the one a wrong answer makes expensive and silent. Putting it second puts it in its own commit, against a diff a reviewer reaches with the vocabulary already understood.

No dependency runs the other way, so this ordering costs nothing if it is ever disputed.

---

## Review Workload Forecast

Forecast recorded **before** the first test. The actual column is filled after the last one, because the difference is the point.

| Slice | Files | Forecast | **Actual** | Risk | Reviewer time |
| --- | --- | --- | --- | --- | --- |
| SDD planning artifacts | `explore.md`, `proposal.md`, `specs/ai-completion-metadata/spec.md`, `design.md`, `tasks.md` | ~600 prose | _pending_ | Low | 25 min |
| AI-13.1 + AI-13.2 | `src/ai/finish_reason.go`, `src/ai/finish_reason_test.go` | ~380 Go | _pending_ | **Medium** — AI-31.1 maps every vendor onto it; AI-40 freezes it | 30 min |
| AI-13.3 + AI-13.4 | `src/ai/usage.go`, `src/ai/usage_test.go` | ~370 Go | _pending_ | **Medium** — the inclusive/exclusive decision | 30 min |
| **Total** | 4 new Go files, 5 new markdown files | ~750 Go | _pending_ | **Medium** | **~85 min** |

### Budget reassessment — trigger 4 fires, and this is the record of it

doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and split trigger 4 fires when the projected diff pushes the milestone past the review budget. **The forecast already exceeds it**, so this is the reassessment made in advance rather than a silent overrun discovered afterwards. It is revisited against the actual figures in the verification pass.

- **The milestone is genuinely two contracts, and doc 0002 says so.** Its charter names two deliverables ("a closed finish-reason vocabulary … a usage record …") and its graph has two chains that share no symbol. Split trigger 5 — "two agents could work it concurrently without conflict" — is satisfied *between the chains* and by nothing else.
- **The split is cut into history rather than deferred.** One commit per leaf-chain boundary, so if a reviewer wants two PRs the cut is `finish_reason*` against `usage*` and no rework is needed.
- **The argument against actually splitting.** `CAP-R-03` makes the two values one capability: "a stream that finishes normally ends with a completion event carrying a finish reason **and** a usage record". AI-15.2 consumes both in one event. Landing half of `CAP-R-03` is the partial-contract state defect **C4** came from. The recommendation is one PR reviewed in two passes, in commit order.
- **No other split trigger fires.** Each leaf's test list is 2 to 5 items against a limit of 7.

---

## Phase AI-13.1 — The finish-reason vocabulary `[leaf]`

Five test-list items, in doc 0002's order.

### T-ACM-1 — Item 1: the vocabulary is closed and each value is constructible

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-001` (`S-ACM-001`, `S-ACM-002`), `R-ACM-002` (`S-ACM-003`, `S-ACM-004`).

---

### T-ACM-2 — Item 2: refusal and content filter are distinct, and the line is documented

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-003` (`S-ACM-005`, `S-ACM-006`).

---

### T-ACM-3 — Item 3: provider strings normalize after trimming and lowering

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-004` (`S-ACM-007`, `S-ACM-008`), `R-ACM-005` (`S-ACM-011`).

---

### T-ACM-4 — Item 4: an unrecognized string maps to unknown without error

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-004` (`S-ACM-009`, `S-ACM-010`), `R-ACM-005` (`S-ACM-012`), `NFR-ACM-B`.

---

### T-ACM-5 — Item 5 *(pin)*: exhaustiveness

- [ ] **PIN LANDED GREEN**
- [ ] **SHOWN TO BITE** against a scratch value, then scratch removed
- [ ] **REFACTOR**

**Proves:** `R-ACM-006` (`S-ACM-013`, `S-ACM-014`).

---

## Phase AI-13.2 — Three-way distinguishability `[leaf]`

Two test-list items.

### T-ACM-6 — Item 1: three states, and collapsing them is compile-visible

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-007` (`S-ACM-015`, `S-ACM-016`, `S-ACM-017`).

---

### T-ACM-7 — Item 2: the obligation attached to each value

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-008` (`S-ACM-018`, `S-ACM-019`).

---

## Phase AI-13.3 — Usage: absent is not zero `[leaf]`

Three test-list items.

### T-ACM-8 — Item 1: absent is distinguishable from zero on every count

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-009` (`S-ACM-020`, `S-ACM-021`, `S-ACM-022`).

---

### T-ACM-9 — Item 2: constructible with any subset present

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-010` (`S-ACM-023`, `S-ACM-024`, `S-ACM-025`).

---

### T-ACM-10 — Item 3: readable from an external package, field by field

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-009`, `R-ACM-010`.

---

## Phase AI-13.4 — Cost-formula clarity `[leaf]`

Two test-list items.

### T-ACM-11 — Item 1: the inclusive-or-exclusive semantics, pinned against a cache-hit record

- [ ] **RED**
- [ ] **GREEN**
- [ ] **REFACTOR**

**Proves:** `R-ACM-011` (`S-ACM-026`, `S-ACM-027`, `S-ACM-028`).

---

### T-ACM-12 — Item 2 *(pin)*: the formula's term list is the field set

- [ ] **PIN LANDED GREEN**
- [ ] **SHOWN TO BITE** against a scratch field, then scratch removed
- [ ] **REFACTOR**

**Proves:** `R-ACM-012` (`S-ACM-029`, `S-ACM-030`, `S-ACM-031`).

---

## Verification pass

- [ ] `make test` in `backend/agent/` — green, `-race`, every package. Tail recorded.
- [ ] `make lint` — `go vet` plus `golangci-lint` at the pinned `v2.9.0`, clean.
- [ ] `go.mod` still declares **zero requires**.
- [ ] Both AI-00 import guards pass.
- [ ] `validation.go` and `validation_test.go` untouched — this change is a consumer of AI-04, not an editor of it.
- [ ] No file under `openspec/specs/` touched: this milestone needs no register amendment.
- [ ] Revert-and-record clause: whether it fired, and any doc 0002 amendment it required.

---

## What a later milestone inherits

| Node | Inherits |
| --- | --- |
| **AI-15.2** | Both values, ready to be carried by a completion event, and AI-13.3's absent-versus-zero property to preserve across the event boundary — which that node's test list already restates as its own item |
| **AI-23.8** | The conformance case for absent-versus-zero in usage, which `openspec/specs/ai-minimum-capabilities/spec.md` § 11 marks **required** despite its node |
| **AI-31.1** | The vocabulary every vendor stop value maps onto, and the recorded pause-turn obligation — resume, replaying received content verbatim |
| **AI-31.3** | The inclusive/exclusive semantics to verify against a real cache-hit transcript |
| **Layer 2 / Layer 3** | Honest token counts, with `V-OUT-07` cost events and `V-OUT-08` prices left entirely to them |
| **AI-05's reviewer** | Two closed vocabularies built without sight of each other. `design.md` § 3.2 records that a reconciliation pass is expected and that nothing here anticipates it |
