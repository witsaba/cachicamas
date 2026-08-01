# Proposal — response lifecycle events

> **Change**: `cachicamas-ai-response-events` · **Milestone**: AI-15 (Wave 2 "Stream")
> **Nodes**: AI-15.1 `[leaf]` · AI-15.2 `[leaf]` · **Depends on**: AI-13 (merged), AI-14 (concurrent) · **Blocks**: AI-16 … AI-19
> **Predecessor**: `exploration.md` (this folder) · **Register terms owned**: `V-STR-19`, `V-STR-20`

## Intent

A stream currently has no way to say "the provider started answering" or "the provider finished normally". Without the first, a consumer cannot attribute a stream to a provider response id or learn which model actually served it; without the second, a normal end is indistinguishable from a failed one, which is the shape defect `C4` produced. AI-15 lands both events so that an **empty but successful** response — start, then completion, no content — is legal, ordered, and distinguishable from every failure.

## Scope

### In scope

- Two event payloads in `backend/agent/src/ai`, registered into AI-14's kind registry: **response-start** (provider response identity + model actually used) and **completion** (`FinishReason` + `Usage`, marked terminal).
- Behavioral spec + design + TDD tasks proving AI-15.1's two and AI-15.2's three test-list items.
- Register amendment appending the response-side nouns AI-15 needs (see Modified Capabilities).

### Out of scope

| Excluded | Owner |
| --- | --- |
| The envelope, sequence, kind registry, invariant checker itself | AI-14 |
| Text / reasoning / tool-call block events | AI-16 … AI-18 |
| Terminal **error** event and failure taxonomy | AI-19 |
| Any new finish-reason value or usage field | AI-13 — consumed unchanged |
| Vendor mapping of response ids / served model | AI-31 |

## Capabilities

### New Capabilities

- `ai-response-events`: the response-start and completion events, their fields, their validation, and their relationship to the stream's ordering invariants.

### Modified Capabilities

- `ai-contract-vocabulary`: `V-STR-19` names the response-start event but defines neither *response identity* nor *model actually used*; `V-REQ-21` (model identity) is request-side and does not cover the served model. Register § 9 rule 2 requires appending the missing nouns in the PR that needs them. The spec phase confirms the exact appends.

## Approach

1. **Two payload types, not one discriminated type** — kind stays derived from payload (`V-STR-11`), so AI-14.1's "every registered kind has a constructible payload" table stays honest.
2. **Reuse, do not redefine** — `FinishReason` and `Usage` are embedded as-is; absent token counts stay absent across the event boundary.
3. **Provider identity is a plain, byte-exact `string`** — the `ToolCall.ID()` / `Request.Model()` precedent, not the minted `MessageID` pattern. Validation is non-empty only (`ErrEmpty` via `FirstFailure`).
4. **Spec behavior, not Go signatures**, until AI-14 is code-complete.

## Dependency note — AI-14 primitive required

AI-15 needs a **generic terminal-event / single-instance-per-stream primitive from AI-14**: AI-15.1 test 2 requires "at most one event of kind X per stream" and AI-15.2 test 2 requires a generic "terminal" property. If AI-14 ships only block start/delta/end triple invariants, **AI-15's design phase must extend them** — by amending AI-14's checker, never by special-casing AI-15's two kinds inside it, which would break registry-driven extensibility.

## Affected areas

| Area | Impact |
| --- | --- |
| `backend/agent/src/ai/` (2 payload files + tests) | New |
| AI-14's kind registry and invariant checker | Modified (registration; possibly generalized) |
| `openspec/specs/ai-contract-vocabulary/spec.md` | Amended (append-only) |
| `go.mod`, existing Wave 0/1 files | None |

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| AI-14's actual signatures differ from its charter → rework | High | Spec behavior only; `sdd-apply` gated behind AI-14's merge |
| AI-14 lacks the generic terminal/single-instance hook | Medium | Dependency note above; raise with AI-14 before its spec freezes |
| "Model actually used" conflated with the requested model | Medium | Distinct register noun + a test where served ≠ requested |
| Empty response mistaken for a failure | Low | AI-15.2 item 3 asserts it against every failure shape available pre-AI-19 |

## Rollback plan

Purely additive: two new Go files, one new change folder, one append-only register amendment. `git revert` of the commit range restores the package to AI-14's surface. No existing declaration changes, so nothing else recompiles differently. If AI-14's checker is generalized here, that edit reverts with the same range.

## Dependencies

- **AI-14** merged in this worktree before `sdd-apply` starts (hard gate).
- **AI-13** `finish_reason.go` / `usage.go` — consumed unchanged.
- **AI-04** validation taxonomy; **AI-02** stream terms. No new Go dependency; module keeps zero requires.

## Success criteria

- [ ] Every AI-15.1 and AI-15.2 test-list item taken red → green → refactored, in order, evidence recorded.
- [ ] `make test` (`go test -race -v ./...`) green in `backend/agent/`, lint clean.
- [ ] Response-start and completion are readable from an external package without a type switch over unexported types.
- [ ] A second response-start, and any event after completion, are both rejected by AI-14's invariant checker.
- [ ] Absent usage fields remain absent after crossing the event boundary.
- [ ] A start-then-completion stream validates as legal and is distinguishable from a failure.
- [ ] Register amendment landed in this PR, or explicitly recorded as "no noun missing".
