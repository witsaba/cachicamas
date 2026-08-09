# Delta — `ai-provider-conformance-suite`

> **Change**: `cachicamas-ai-adapter-conformance` (AI-38, Wave 6) · **Target**: [`openspec/specs/ai-provider-conformance-suite/spec.md`](../../../../specs/ai-provider-conformance-suite/spec.md)
> **Identifier allocation**: next free requirement is `R-CNF-028` (`R-CNF-020`…`R-CNF-026` are consumed by two archived changes and `R-CNF-027` by AI-36 — see that file's identifier note). Next free scenario is `S-CNF-082` (`S-CNF-081` is the highest in use).
> **Maintainer decisions binding this delta**: Engram `#2763` — (1) cancellation resolves **suite-side**; the shipped adapter behaviour under `ai-provider-error-mapping` `R-AEM-014` / `S-AEM-051…055` stands. (4) finish reasons a dialect cannot express are recorded absences, with **no** `R-ACP-002` reopen.
> **Format**: RFC 2119 + Given/When/Then per `openspec/config.yaml`.

---

## MODIFIED Requirements

### R-CNF-010 — All nine failure categories are iterated exhaustively against the shipped enumerator, with dialect collapse recorded rather than skipped

The suite MUST iterate **every** member of the failure-category vocabulary by consuming AI-19's
exported enumerator, not a hand-written list, and MUST assert each category is expressible and
classifiable on both delivery paths. A new category added upstream MUST cause the suite to fail
until a case covers it; the exhaustiveness MUST be enforced mechanically, never by reviewer
vigilance.

A category a subject's wire dialect **collapses** on the mid-stream path — because the dialect
carries no discriminator that could preserve it — MUST still produce exactly one typed terminal, and
that terminal's category MUST equal the subject's declared collapse value. The suite MUST record
this as a **dialect-aware collapse** naming both the category and the dialect; it MUST NOT read as a
skip and MUST NOT read as a pass of the original category. The enumerator-driven exhaustive
iteration and the mechanical drift failure MUST remain intact for the vocabulary as a whole; only
the per-subject mid-stream classification claim is narrowed. A category the dialect **can** express
mid-stream MUST still be classified as itself.
(Previously: every category was required to arrive mid-stream classified as itself on every subject,
which no dialect lacking a mid-stream error discriminator can satisfy.)

#### Scenarios

- **S-CNF-024** *(modified, AI-38)* — Given the nine enumerated failure categories, when the suite
  runs against a subject, then each is exercised and classified through the one vocabulary on the
  pre-stream path; and on the mid-stream path each category the subject's dialect can express is
  classified as itself, and each category the dialect collapses arrives as exactly one typed
  terminal carrying the declared collapse value, recorded as a dialect-aware collapse naming the
  category and the dialect.
- **S-CNF-025** — Given a hypothetical tenth category added to the enumerator without a case, when
  the suite's exhaustiveness check runs, then it fails naming the uncovered category.
- **S-CNF-087** *(added, AI-38)* — Given a category a subject's dialect can express mid-stream that
  arrives carrying any other category, when the suite runs, then the case fails naming the expected
  and observed categories — the dialect-aware collapse is not available as a general escape.

### R-CNF-011 — Cancellation closes within bounded time, leaks no goroutine, and admits at most one typed cancellation terminal

The suite MUST assert that cancelling the caller-owned signal ends the stream within a bounded
deadline (`CAP-R-04`) and that the stream is closed exactly once by its producer. The suite MUST
accept **either** a bare close **or** exactly one terminal event whose failure payload is of the
cancellation category (`ai-provider-error-mapping` `R-AEM-014`) as conformant cancellation shapes,
and MUST reject anything else: more than one terminal, a terminal of any other category, or any
event after the terminal. Leak freedom MUST be asserted through AI-22's opt-in, amplitude-based
`R-STK-007` helper; the suite MUST NOT install an implicit global leak check and MUST NOT call
`t.Parallel()` in any test that uses it.
(Previously: cancellation admitted only a bare close, which conflicted with the promoted
`R-AEM-014` typed-terminal obligation; the suite over-specified one of two legal shapes.)

#### Scenarios

- **S-CNF-026** — Given a stream cancelled mid-consumption, when the suite waits with a bounded
  deadline, then the stream closes before the deadline and is closed exactly once.
- **S-CNF-027** — Given the cancellation scenario repeated under the leak helper, when it completes,
  then no goroutine growth beyond the helper's stated tolerance is observed.
- **S-CNF-028** — Given every suite test using the leak helper, when a reviewer enumerates them,
  then none calls `t.Parallel()`.
- **S-CNF-082** *(added, AI-38)* — Given a cancelled stream from a subject that emits one terminal
  carrying a cancellation-category failure, when the suite asserts cancellation, then the case
  passes; and given a subject that emits two terminals, or one terminal of any non-cancellation
  category, or any event after the terminal, then the case fails naming the observed shape.

### R-CNF-012 — The abandoned-then-cancelled saturated path drops cleanly, inventing no terminal beyond the admitted cancellation terminal

The suite MUST assert AI-20.3's saturated-drop physics: a consumer that stops reading until the
buffer saturates, whose caller then cancels, MUST see the stream close with **no undelivered event
forced through, no already-delivered event discarded, and no leak**. The closure MUST be either bare
or carry exactly one cancellation-category terminal; no other terminal MUST be invented and no
second terminal MUST appear. The suite MUST NOT assert the **abandoned-never-cancelled** path; that
narrowing is inherited from `ai-stream-lifecycle` § 5's untestability ruling and is recorded here
rather than left silently absent.
(Previously: the saturated path required a strictly bare close, forbidding the cancellation-category
terminal `R-AEM-014` requires of the shipped adapter.)

#### Scenarios

- **S-CNF-029** — Given a consumer that stops reading until the buffer saturates and a caller that
  then cancels, when the stream terminates, then it closes with no undelivered event forced through
  and the events already delivered intact.
- **S-CNF-030** — Given that scenario repeated under the leak helper, when it completes, then no
  goroutine growth beyond tolerance is observed.
- **S-CNF-031** — Given the landed suite, when a reader looks for an abandoned-never-cancelled case,
  then it is stated out of scope with the § 5 reason cited.
- **S-CNF-083** *(added, AI-38)* — Given the saturated-then-cancelled path on a subject that closes
  bare and on a subject that emits one cancellation-category terminal, when the suite runs, then
  both pass; and given a subject that invents a terminal of another category on that path, then the
  case fails naming the invented category.

### R-CNF-016 — `CAP-O-03` cache-boundary honoring, and the required cases living in this node

The suite MUST cover `CAP-O-03` through the declared capability expectation of `R-CNF-002`: honoring
MUST be asserted as consumer-visible behaviour when offered, and recorded `absent` when the factory
declares it is not. This node MUST additionally carry two cases that are **required** because they
exercise `CAP-R-03`: (a) a normally-finished stream ends carrying a finish reason from the closed
vocabulary, iterated **exhaustively over all seven values**; and (b) usage honours absent-versus-zero
per field, so an absent count is distinguishable from a reported zero. Because the finish-reason
vocabulary exports no enumerator, the seven values MUST be hand-listed in the suite **behind a drift
guard** that fails when the upstream vocabulary gains or loses a member; the suite MUST NOT modify
`src/ai` to obtain an enumerator.

A finish-reason value the subject's wire dialect cannot express MUST be recorded as a **dialect-aware
absence** — unsatisfiable and unviolated — naming the value and the dialect, and MUST NOT read as a
pass and MUST NOT be silently skipped. The exhaustive iteration and the drift guard MUST remain
intact for the vocabulary as a whole; only the per-subject reachability claim is narrowed. A value
the dialect **can** express but the subject does not produce MUST still fail.
(Previously: `S-CNF-043` claimed every one of the seven finish reasons is reachable on every
subject's normally-finished stream, which no strict single-dialect adapter can satisfy.)

#### Scenarios

- **S-CNF-042** — Given a subject declaring cache-boundary honoring offered, when the suite
  exercises it, then honoring is observed as consumer-visible behaviour; given one declaring it not
  offered, the entry is `absent` with a reported skip.
- **S-CNF-043** *(modified, AI-38)* — Given all seven finish reasons and a subject, when the suite
  iterates them, then each value its dialect can express is reachable on a normally-finished stream
  and is a closed-vocabulary value, and each value its dialect cannot express is recorded as a
  dialect-aware absence naming the value and the dialect.
- **S-CNF-044** — Given a hypothetical eighth finish reason added upstream without a suite case, when
  the drift guard runs, then it fails naming the uncovered value; given a value removed, it likewise
  fails.
- **S-CNF-045** — Given a usage record with one count absent and another reported as zero, when the
  suite reads them, then the two are distinguishable and neither is coerced into the other.
- **S-CNF-046** — Given the standing of cases (a) and (b), when it is computed, then both are
  **required** despite living in the optional-capability node.
- **S-CNF-084** *(added, AI-38)* — Given a subject whose dialect can express a finish reason but
  which never produces it, when the suite runs, then the case fails naming the value — the
  dialect-aware absence is not available as a general escape.

## ADDED Requirements

### R-CNF-028 — A scoped run is never presentable as evidence of full conformance

The suite's scoped, per-capability entry point MUST remain available as a debugging affordance, and
its verdict MUST be reported as non-evidential. A scoped verdict MUST NOT be recorded as a
conformance pass, MUST NOT emit a record that a consumer could mistake for a total record, and MUST
NOT be cited as acceptance evidence in code, comment, report, or spec. Only the unscoped entry
point's run MUST count as full-conformance evidence. This obligation is a requirement of the suite,
not a documentation comment.

#### Scenarios

- **S-CNF-085** — Given a scoped per-capability run, when its verdict and any emitted record are
  read, then both are marked non-evidential and the record is not total over the nine entries.
- **S-CNF-086** — Given an acceptance artifact that cites a scoped run as full-conformance evidence,
  when the suite's evidence check runs, then it fails naming the citation.

---

## REMOVED Requirements

None.

## RENAMED Requirements

None.

## Traceability

| Delta | Basis |
|---|---|
| `R-CNF-010` modified | Locked decision (4)'s mechanism generalized, Engram `#2763`; design D5 `DialectConstraints` seam; `stream_failure.go` hardcodes `FailureCategoryUnknown` for in-band frames, collapsing every mid-stream category on the openaicompat dialect |
| `R-CNF-011` / `R-CNF-012` modified | Decision (1), Engram `#2763`; conflict with promoted `R-AEM-014` / `S-AEM-051…055` (AI-32.3) |
| `R-CNF-016` modified | Decision (4), Engram `#2763`; AI-29/AI-31 unsatisfiable-and-unviolated precedent; no `R-ACP-002` reopen |
| `R-CNF-028` added | Proposal T1; `ai-adapter-conformance-run` `R-ACR-001` consumes it |
