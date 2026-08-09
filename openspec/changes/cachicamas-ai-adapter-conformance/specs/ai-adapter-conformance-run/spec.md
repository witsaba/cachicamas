# Spec — the deterministic adapter conformance run

> **Change**: `cachicamas-ai-adapter-conformance` · **Milestone**: AI-38 (doc 0002) · **Wave**: 6 — Hand off
> **Capability**: `ai-adapter-conformance-run` (**new** — full spec, no MODIFIED/REMOVED/RENAMED)
> **Requirement IDs**: `R-ACR-0NN` · **Scenario IDs**: `S-ACR-0NN` (prefix verified free across `openspec/specs/` and `openspec/changes/`)
> **Format**: RFC 2119 + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Composes (cited, not modified here)**: `ai-provider-conformance-suite` (`R-CNF-001`…`R-CNF-019`, `R-CNF-027`) · `ai-openrouter-first-provider` (`R-OR-05`, `R-OR-06`) · `ai-provider-error-mapping` (`R-AEM-014`) · `ai-minimum-capabilities` (`CAP-R-01…05`, `CAP-O-01…04`)
> **Maintainer decisions binding this spec**: Engram `#2763` (`sdd/cachicamas-ai-adapter-conformance/decisions`)

---

## Purpose

The AI-23 suite has never been run unscoped against a real adapter; its only unscoped subject is
AI-21's in-process fake. This capability states what MUST be true once the shipped OpenRouter
adapter is judged by that suite: which run counts as evidence, where the transcripts come from,
what the emitted capability record must equal, and that the verdict survives adversarial stream
fragmentation. It judges no vendor and adds no network dependency to the default test run.

## Definitions

- **The run** — one execution of the unscoped suite entry point against an adapter-backed factory.
- **The subject** — the shipped OpenRouter bridge over the real `openaicompat` client.
- **A transcript** — the recorded wire bytes a conformance case replays through the subject.
- **The recording helper** — the tool that serialises a live stream into the transcript format.
- **The expectation** — the committed nine-entry capability record the run is asserted against.

---

## Requirements

### R-ACR-001 — Only an unscoped run is acceptance evidence, and no case may be skipped

The unscoped suite entry point run against the adapter factory MUST be the sole acceptance gate for
this milestone. Every required capability (`CAP-R-01`…`CAP-R-05`) MUST pass in that run. The
conformance drivers MUST carry zero skip directives at close, and the run MUST NOT be narrowed by a
waiver, an allow-list, or an environment condition. A scoped, per-capability run MAY survive as a
debugging affordance but MUST NOT be cited, in code, comment, report, or spec, as evidence of full
conformance.

#### Scenarios

- **S-ACR-001** — Given the shipped adapter factory, when the unscoped suite runs, then every
  required capability case executes and passes, and the run reports a pass verdict.
- **S-ACR-002** — Given the merged conformance drivers, when a reviewer enumerates skip directives
  and waivers in them, then none exists.
- **S-ACR-003** — Given a scoped per-capability run, when its verdict is read, then it is reported
  as non-evidential and no acceptance artifact cites it as full conformance.

### R-ACR-002 — Every mismatch is resolved as a defect on a named side, with a recorded rationale

Each case that fails the first unscoped run MUST be resolved either as a defect in the suite case or
as a defect in the adapter. The resolution MUST record which side moved and why. A case MUST NOT be
closed by weakening its assertion to whatever the adapter already does without that rationale, and
MUST NOT be closed by deletion.

#### Scenarios

- **S-ACR-004** — Given the recorded first-run failure set, when the change closes, then each entry
  carries a resolution naming the side that moved and the reason.
- **S-ACR-005** — Given the merged suite and adapter, when the case count of the first run is
  compared with the case count at close, then no case was removed to obtain a pass.

### R-ACR-003 — Every transcript is regenerable from a captured stream, and hand edits fail

Every conformance transcript MUST be producible by the recording helper from real streamed wire
bytes, in the exact byte format the replay harness consumes. Committed transcripts MUST be
byte-identical to the helper's output for the same captured bytes, enforced by a drift guard so a
hand edit fails the suite. The helper's capture mode MUST be proven against a **local
openaicompat-speaking endpoint** and MUST NOT require, or spend against, a vendor network call; the
default test run MUST NOT depend on any credential or network access.

#### Scenarios

- **S-ACR-006** — Given the recording helper pointed at a local openaicompat-speaking endpoint, when
  it captures a stream, then it emits a transcript in the format the replay harness consumes and the
  suite replays it unchanged.
- **S-ACR-007** — Given a committed transcript and the helper's output for the same captured bytes,
  when the drift guard compares them, then they are byte-identical.
- **S-ACR-008** — Given a committed transcript edited by hand, when the drift guard runs, then it
  fails naming the drifted transcript.
- **S-ACR-009** — Given the default test run with no credential in the environment, when it
  executes, then it passes and makes no outbound network request.

### R-ACR-004 — The capability record is generated by the run and asserted over all nine entries

The record asserted MUST be the record a real unscoped run emits, compared entry by entry against
the committed nine-entry expectation. Asserting the factory's declared capability pointers MUST NOT
substitute for that comparison. Every one of the nine entries MUST be asserted; none may be left
unchecked. `CAP-O-04` (retry) MUST be recorded `satisfied` for this subject. `CAP-O-01` (reasoning)
MUST remain `absent` under the default non-reasoning model; a generated `satisfied` for `CAP-O-01`
MUST block the change and escalate as an AI-29 reopen trigger, and MUST NOT be absorbed by updating
the expectation.

#### Scenarios

- **S-ACR-010** — Given a completed unscoped run, when its emitted record is compared with the
  committed expectation, then all nine entries match and the comparison is entry-by-entry.
- **S-ACR-011** — Given the assertion code, when a reviewer looks for a declared-pointer check
  standing in for the generated record, then none exists and no entry is left unasserted.
- **S-ACR-012** — Given the run's emitted record, when `CAP-O-04` is read, then its outcome is
  `satisfied` and both adapter factories declared retry identically.
- **S-ACR-013** — Given a run that emits `CAP-O-01` as `satisfied`, when the assertion executes,
  then it fails naming the reopen trigger, and the expectation is not amended to accommodate it.

### R-ACR-005 — The verdict survives adversarial stream fragmentation

Every conformance transcript MUST be replayed through the adapter split at adversarial byte offsets,
and the suite MUST produce an **identical capability record** at every split point, not merely
identical decoded frames. A canonical unsplit run MUST be pinned to an expected record so a
uniformly broken harness cannot pass vacuously. Transcript size MUST be bounded first; only if the
full offset cross-product then exceeds the runtime budget MAY offsets be sampled, and both the bound
and any sampling MUST be recorded in the suite.

#### Scenarios

- **S-ACR-014** — Given each conformance transcript split at every admitted byte offset, when each
  split replay runs, then the emitted capability record is identical to the canonical unsplit run's.
- **S-ACR-015** — Given the canonical unsplit run, when its record is compared with the pinned
  expected record, then they match, so a uniformly broken harness fails rather than passing
  vacuously.
- **S-ACR-016** — Given a transcript exceeding the recorded size bound, when the bound check runs,
  then it fails naming the transcript; and given offset sampling in effect, when the suite is read,
  then the sampling rule and the bound are both stated.

### R-ACR-006 — Retry declaration parity across adapter factories

Every factory that constructs the same adapter for conformance purposes MUST declare the retry
capability identically. A disagreement between two such factories MUST fail mechanically rather than
being discovered by review.

#### Scenarios

- **S-ACR-017** — Given the adapter's conformance factories, when their retry declarations are read,
  then all are equal and equal to the value the committed expectation records.
- **S-ACR-018** — Given one factory's retry declaration mutated to disagree, when the suite runs,
  then it fails naming both factories.

---

## Non-functional requirements

### NFR-ACR-A — Determinism, dependency purity, and no in-place spec edits

The whole run MUST be deterministic under the race detector across repeated executions and MUST NOT
depend on wall-clock timing for correctness; bounded deadlines proving a call does not hang are
permitted. This change MUST add no NEW module dependency: `backend/agent/go.mod` and `go.sum` MUST
stay byte-identical to the base commit's copies, whatever requires that base commit already carries.
(This capability's own requirements — R-ACR-001…R-ACR-006 above — are proven entirely by test code
and delta-spec text; none of them needs a new dependency to satisfy, so byte-identity to base is the
correct, checkable form of this obligation regardless of what the base commit's own `go.mod` already
declares.) Canonical behaviour MUST change only through this change's delta specs; no file under
`openspec/specs/` is edited in place before archive.

- **S-ACR-019** — Given the milestone's test set run repeatedly under the race detector, when the
  results are compared, then they are identical and no test performs vendor network I/O.
- **S-ACR-020** — Given the merged change, when `go.mod`/`go.sum` are diffed against the base commit
  and the import guards are read, then the diff is empty and both guards pass.
- **S-ACR-021** — Given the merged diff, when a reviewer looks for an edit under `openspec/specs/`,
  then none exists.

---

## Acceptance criteria

1. The unscoped run passes every required capability, with zero skips and zero waivers
   (`R-ACR-001`).
2. Every first-run failure carries a named side and a recorded rationale (`R-ACR-002`).
3. Every transcript is helper-generated, byte-identical under the drift guard, captured with zero
   vendor network spend (`R-ACR-003`).
4. The generated nine-entry record equals the committed expectation, `CAP-O-04` is `satisfied`, and
   a generated `CAP-O-01 = satisfied` blocks (`R-ACR-004`).
5. Every transcript yields an identical record at every split offset, anchored by a pinned canonical
   run (`R-ACR-005`).
6. Retry is declared identically by every adapter conformance factory (`R-ACR-006`).
7. The suite is deterministic under `-race`, adds no NEW dependency (`go.mod`/`go.sum` byte-identical
   to base), and edits no promoted spec in place (`NFR-ACR-A`).
