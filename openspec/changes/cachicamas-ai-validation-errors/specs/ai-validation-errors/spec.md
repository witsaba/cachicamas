# Spec — the validation error taxonomy

> **Change**: `cachicamas-ai-validation-errors`
> **Milestone**: AI-04 · **Nodes**: AI-04.1 `[decision]`, AI-04.2 `[leaf]`, AI-04.3 `[leaf]`
> **Phase**: spec (delta)
> **Canonical spec**: `openspec/specs/ai-validation-errors/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-07-31
> **Requirement IDs**: `R-AIE-0NN` · **Scenario IDs**: `S-AIE-0NN`
> **Binding input**: [the register](../../../../specs/ai-contract-vocabulary/spec.md) — `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13`, `V-FAIL-16`, `V-FAIL-17`, `V-REQ-22`

---

## Purpose

Wave 0's specs constrained an artifact, because a `[decision]` leaf ships no code. This one is different and the difference is stated once here so no later phase mistakes it: **AI-04 ships runtime behavior, and this spec constrains that behavior.** A scenario reads "given a caller-contract failure, when …, then …", and every one of them is verifiable by a test that runs.

Two requirements — `R-AIE-001` and `R-AIE-012` — constrain the decision artifact instead, because AI-04.1 is part of the same milestone. They are marked where they appear.

## Definitions used by this spec

Conceptual, per the register. The Go spellings are `design.md`'s.

- **A caller-contract failure** — `V-FAIL-01`. The single value a Layer 1 contract returns when a caller broke a rule.
- **A validation sentinel** — `V-FAIL-02`. The stable, matchable value naming which rule class was violated.
- **A rule class** — `V-FAIL-17`. The kind of thing a rule checks, independent of what it checks it on.
- **A position** — `V-FAIL-03` positional context. The structural location of the violation within the request.
- **A structural name** — one segment of a position, naming a field of a Layer 1 contract. Authored by this package; never caller data.
- **The rendered message** — the human-readable text a caller-contract failure produces.
- **The rule list** — the ordered sequence of checks a contract runs, per `V-FAIL-04`.
- **Caller data** — any value supplied by a caller: a model identity, a role string, a tool name, content text, argument bytes, a credential.

---

## R-AIE-001 — The decision artifact exists and answers all four items *(constrains AI-04.1)*

The change MUST produce exactly one decision artifact at `openspec/changes/cachicamas-ai-validation-errors/decision.md`. It MUST answer every item of AI-04.1's closing checklist, MUST state the rejected alternative affirmatively before rejecting it, and MUST resolve at least one borderline case beyond the one the register already works.

### Scenarios

- **S-AIE-001** — Given the change directory, when a reviewer lists its files, then exactly one `decision.md` is present AND no other artifact of the change states a decision as normative.
- **S-AIE-002** — Given the decision artifact, when a reviewer walks AI-04.1's four closing-checklist items in order, then each resolves to a section that states the decision, the rejected alternative at full strength, the ground on which it was rejected, and the conceded cost.
- **S-AIE-003** — Given the decision artifact's boundary section, when a reviewer counts resolved borderline cases, then at least two are resolved by this change itself AND each is resolved by applying the register's stated rule rather than by asserting an answer.

---

## R-AIE-002 — One caller-contract failure type

Layer 1 MUST expose exactly one concrete type for caller-contract failures. Every rule violation in every Layer 1 contract MUST report through it. There MUST NOT be a second type a consumer has to look for, and positional context MUST NOT be carried by a separate wrapper type.

### Scenarios

- **S-AIE-004** — Given any caller-contract failure produced anywhere in Layer 1, when a consumer extracts the failure by type, then exactly one extraction is required AND it succeeds.
- **S-AIE-005** — Given a caller-contract failure with no meaningful position, when a consumer extracts it, then the extraction succeeds AND the position is empty rather than the failure being of a different type.

---

## R-AIE-003 — Sentinels identify a rule class, and the set is closed and appended

A validation sentinel MUST identify a **rule class**, reusable across every Layer 1 type, and MUST NOT be specific to one type's one rule. The sentinel set MUST be closed: a violation MUST report a member of the landed set. The set MUST be extended by appending a class in the pull request that needs it, never by a milestone defining a local sentinel of its own.

A milestone that meets a violation **no landed class describes** MUST append a class rather than report the nearest-fitting one. A class reported for a violation it does not describe is a defect of the same kind as a local sentinel: it is not detectable by any test of this requirement, and it silently makes `errors.Is` answer a question the consumer did not ask.

> **Amended 2026-07-31** — the second paragraph was appended after **AI-08** met the first violation no landed class described, duplicate tool names in one set, and reported it through the nearest-fitting class (`ErrMalformed`) instead of appending. The original wording forbade a *local sentinel* and said nothing about a *stretched* one, and only the first of those two is visible in a diff. `ErrDuplicate` was appended and the call site moved; `S-AIE-034` was added below so the gap is a scenario rather than a habit. `R-AIE-003`'s original sentence is unchanged.

### Scenarios

- **S-AIE-006** — Given two different Layer 1 contracts each rejecting an empty required value, when a consumer matches both against the empty-value sentinel, then both match — "empty value" is one thing everywhere.
- **S-AIE-007** — Given the landed sentinel set, when a reviewer asks why each member exists, then each has a citable case in the register or in doc 0002, AND no member exists only in anticipation of a milestone that has not landed.
- **S-AIE-008** — Given a caller-contract failure, when a consumer matches it against every member of the sentinel set in turn, then exactly one member matches.
- **S-AIE-034** — Given a violation that no landed class describes, when the milestone meeting it reports the failure, then the reported class is one appended for that violation AND is not a landed class whose text describes something else. *(Appended 2026-07-31 by AI-08. Its worked case: a set carrying two values with the same name reports the uniqueness class and explicitly does not match the well-formedness class, because every value in it is well-formed on its own.)*

---

## R-AIE-004 — Sentinels are matchable through wrapping

A caller-contract failure MUST remain matchable against its sentinel after being wrapped at least once by an intermediate layer, so that a caller can classify a failure it received indirectly. This is `V-FAIL-02`'s own clause.

### Scenarios

- **S-AIE-009** — Given a caller-contract failure wrapped once by an intermediate error, when a consumer matches the wrapper against the failure's sentinel, then it matches.
- **S-AIE-010** — Given a caller-contract failure wrapped several times, when a consumer matches the outermost error against the failure's sentinel, then it still matches AND matching against a different sentinel of the set does not.

---

## R-AIE-005 — Positional context is extractable and unambiguous

A caller-contract failure MUST carry a position, which MAY be empty. A consumer MUST be able to extract it by type from a wrapped failure and read it **programmatically** — not only as text. The position MUST name which message, which content-part index, and which tool unambiguously.

### Scenarios

- **S-AIE-011** — Given a failure positioned at the content of a message, when a consumer extracts the position, then it yields the message field and its index, and the content field and its index, in order, each readable as a name and an integer.
- **S-AIE-012** — Given a failure positioned at one tool of the declared tool set, when a consumer extracts the position, then it identifies the tool by its index in the set — which `V-REQ-14` guarantees is ordered and deterministically iterable — AND resolves to exactly one tool.
- **S-AIE-013** — Given a failure whose position names a field with no index, when a consumer reads that segment, then the segment reports that it has no index rather than reporting a sentinel integer a caller must know to ignore.

---

## R-AIE-006 — Rendered messages are content-free by construction

A rendered message MUST NOT contain caller data. This MUST hold **by construction**, not by convention: every dynamic component of a rendered message MUST be either an integer index or a structural name that has passed a filter admitting only identifier-shaped text of bounded length. A name that fails the filter MUST be replaced whole, never truncated. The rendered message MUST NOT reproduce the text of the error the failure was constructed with; it MUST render the text of the registered rule class the failure matches.

### Scenarios

- **S-AIE-014** — Given a distinctive sentinel body — a string that could only have come from caller content — supplied as a structural name, when the failure is rendered, then the sentinel body appears nowhere in the rendered message.
- **S-AIE-015** — Given a structural name that fails the filter, when the failure is rendered, then no substring of that name appears in the message — a prefix of a secret is still a secret.
- **S-AIE-016** — Given a failure constructed from an error that wraps a registered sentinel and whose own text carries a distinctive sentinel body, when the failure is rendered, then the message carries the registered class's own text AND not the wrapper's, AND the failure still matches the wrapped sentinel.
- **S-AIE-017** — Given any rendered message, when a reviewer inspects it, then it is composed only of the registered class's text, filtered structural names, integer indices, and fixed punctuation.

---

## R-AIE-007 — Ordered first-failure

WHEN a value violates several rules at once, the reported failure MUST be the first violated rule in the rule list's order. Every rule before it MUST have been evaluated; no rule after it MAY be evaluated. Reporting MUST NOT aggregate.

### Scenarios

- **S-AIE-018** — Given a rule list in which the second, third and fifth rules are violated, when the list is evaluated, then the reported failure is the second rule's AND the third and fifth are never evaluated.
- **S-AIE-019** — Given a rule list in which no rule is violated, when the list is evaluated, then the result is a genuinely absent failure — one that a caller comparing against nothing finds absent, with no typed-but-empty value masquerading as a failure.
- **S-AIE-020** — Given an empty rule list, when it is evaluated, then the result is an absent failure and nothing panics.

---

## R-AIE-008 — Determinism, including under the race detector

The rule a value reports MUST be identical across repeated evaluations of the same value in one process, across concurrent evaluations, and under the race detector. No unordered iteration MAY decide which rule fires.

### Scenarios

- **S-AIE-021** — Given a value violating several rules, when the same rule list is evaluated many times in sequence, then every evaluation reports the same sentinel and the same position.
- **S-AIE-022** — Given the same value evaluated concurrently from several goroutines under `-race`, when the results are collected, then all are identical AND the race detector reports nothing.
- **S-AIE-023** — Given the package's own source, when a reviewer looks for what decides which rule fires, then it is the order of an ordered sequence AND no map iteration participates.

---

## R-AIE-009 — Totality: no input causes a panic

Constructing, matching, extracting from, and rendering a caller-contract failure MUST NOT panic for any input, including a nil rule, a nil failure value, a zero-value position segment, an empty position, a deeply nested position, and a maximum-size structural name.

### Scenarios

- **S-AIE-024** — Given a failure constructed with no rule, when it is rendered and matched, then nothing panics AND the message states that the rule is unnamed rather than reproducing anything.
- **S-AIE-025** — Given a nil failure value of the failure type, when it is rendered, then nothing panics.
- **S-AIE-026** — Given a position of many thousands of segments, when the failure is rendered, then nothing panics AND rendering does not recurse.
- **S-AIE-027** — Given a zero-value position segment and a structural name at and beyond the length bound, when the failure is rendered, then nothing panics.

---

## R-AIE-010 — The taxonomy is caller-contract only

This package surface MUST NOT classify, carry, or name anything that happens after a valid request leaves the process. Status codes, failure categories, retryability, partial output and terminal events are AI-19's and MUST NOT appear here.

### Scenarios

- **S-AIE-028** — Given the landed surface, when a reviewer reads every exported declaration, then none names a transport, protocol, status, retry, category or stream concept.
- **S-AIE-029** — Given the decision artifact, when a reviewer looks for the complement, then everything assigned to the other side of the boundary is named as AI-19's inheritance rather than being partially handled here.

---

## R-AIE-011 — No speculative validators

The package MUST NOT export a rule, a validator, or a sentinel bound to a Layer 1 type that AI-05 … AI-13 have not yet defined. The exercise surface MUST be limited to the failure value, the sentinel set, the position, and the ordered-rule mechanism.

### Scenarios

- **S-AIE-030** — Given the landed surface, when a reviewer reads every exported declaration, then none mentions a role, message, content part, tool, request, or option.
- **S-AIE-031** — Given the test suite, when a reviewer reads the values under test, then the multi-violation subjects are defined by the tests themselves AND no production rule exists for a type that does not.

---

## R-AIE-012 — Vocabulary discipline *(constrains the register amendment)*

Every Layer 1 noun used normatively by this change MUST resolve to a register row by identifier. A noun the register lacks MUST be appended to `openspec/specs/ai-contract-vocabulary/spec.md` in this same pull request, append-only, with a dated amendment blockquote and updated counts — never defined locally.

### Scenarios

- **S-AIE-032** — Given this change's artifacts, when a reviewer collects every Layer 1 noun used normatively, then each resolves to a register row.
- **S-AIE-033** — Given the register after this change, when a reviewer diffs it, then exactly two rows are appended with the next free `V-FAIL` ordinals, one dated blockquote is added under § 6, the counts in § 10 are updated, AND no existing row is renumbered, reworded, reordered or removed.

---

## Non-functional requirements

### NFR-AIE-A — Dependency purity

- The implementation MUST use only the Go standard library. `backend/agent/go.mod` MUST still carry zero requires.
- Both AI-00 import guards MUST still pass unchanged.

### NFR-AIE-B — Evidence

- Every test-list item of AI-04.2 and AI-04.3 MUST be taken red before green, and both outputs MUST be recorded in `tasks.md`. The one exception is AI-04.3's pin, which doc 0002 marks exempt from red-first and which MUST still be fully mechanical.
- The milestone closes only on recorded green `make test` in `backend/agent/`, which is `go test -race -v ./...`.

### NFR-AIE-C — Reviewability

- Test functions MUST follow `Test<Subject>_<Behavior>_<Expectation>` and MUST carry a banner comment citing the leaf ID.
- The Go surface MUST be readable in one sitting by someone who has read `decision.md` and nothing else.

---

## Acceptance criteria

The contract holds when:

1. `R-AIE-001` … `R-AIE-012` hold, each verified by its scenarios.
2. All four items of AI-04.1's closing checklist are answered in `decision.md`.
3. Every test-list item of AI-04.2 and AI-04.3 is closed with recorded red and green output.
4. `make test` in `backend/agent/` is green under `-race`, and the output is recorded.
5. The register carries two new rows, append-only, with correct counts.
6. **doc 0002's own acceptance criterion:** every construction and validation rule in AI-05 … AI-13 can report through this taxonomy; failures are matchable through at least one layer of wrapping; no error message carries a content body.
