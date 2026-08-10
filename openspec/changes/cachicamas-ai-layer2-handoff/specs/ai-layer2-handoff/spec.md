# Spec — the Layer 2 readiness contract

> **Introduced by**: `cachicamas-ai-layer2-handoff` (AI-40, Wave 6 — Hand off; **last** milestone, Layer 1's exit)
> **Status**: delta — promoted to `openspec/specs/ai-layer2-handoff/spec.md` at archive
> **Capability**: `ai-layer2-handoff` (**new** — full spec, no MODIFIED/REMOVED/RENAMED)
> **Requirement IDs**: `R-L2H-0NN` · **Scenario IDs**: `S-L2H-0NN` (both prefixes verified free across `openspec/specs/` and `openspec/changes/`)
> **Format**: RFC 2119 + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Composes (cited, never modified here)**: `ai-model-provider` · `ai-fake-provider` (AI-21) · `ai-stream-testkit` (AI-22.3 `ai.CheckStream`) · `ai-provider-conformance-suite` (`R-CNF-017`, `R-CNF-018`; deltaed in this change) · `ai-openrouter-first-provider` (`R-OR-05`) · AI-00.3 import-boundary guard
> **Decision forks binding this spec**: proposal D1–D7 (*recommended, pending maintainer ratification*)
> **Charter**: doc 0002 lines 2369–2410; inherited publication duty at line 2402; item-6 language at line 2446

---

## Purpose

Layer 1 is code-complete but not handed off. A Layer 2 author cannot today answer what may be relied
on, what the adapter supports, or how to write a Layer 2 test: no frozen-surface declaration exists,
the nine-entry capability record is published nowhere a consumer reads, and `backend/agent` ships zero
runnable examples. Two publication duties delegated to this node are unpublished, and doc 0002's
completion checklist contradicts its own status line.

This capability states what MUST be true for Layer 1 to declare itself handed off: a third-party
consumer compiles and runs against the v1 surface with zero vendor imports; four examples are compiled
**and run** by the ordinary test command; the capability matrix is published with its generator and its
reopen trigger and cannot rot silently; both inherited publication duties are stated; the v1 surface is
enumerated and frozen; every completion-checklist item is walked to its closing node; and the
never-cancelled abandoned-consumer case is documented as contract because it cannot be tested to
termination.

## Definitions

- **The consumer proof** — the external-package test that stands in for the future Layer 2 in miniature.
- **The fake** — AI-21's scripted provider (`agenttest.NewProvider`, `Script`/`Step`/`Emit`/`Hold`, `ErrScriptsExhausted`).
- **The matrix** — the nine-row capability record over `CAP-R-01…05` and `CAP-O-01…04`, published in package documentation.
- **The committed expectation** — the unexported, already-committed record fixture that the conformance run asserts against, and the sole source of the matrix's rows.
- **The drift guard** — the read-only test that compares the published rows to the committed expectation.
- **The boundary guard** — AI-00.3's existing module-wide import-boundary test; reused unmodified.
- **The compatibility statement** — the durable `decision.md` artifact carrying the frozen-surface enumeration, the eighteen-item walk, and the never-cancelled posture.

---

## Requirements

### R-L2H-001 — A third-party consumer compiles and runs against the v1 surface with zero vendor imports

An external-package test living outside both the neutral contract package and the test-support package
MUST construct a request, obtain a provider from the fake, drain the resulting event stream to its
terminal event, exercise a **scripted error** path, and exercise a **cancellation** path. Its import
closure MUST contain only the standard library, the module's own neutral contract package, and the
module's test-support package — **no vendor SDK, no HTTP client library, no adapter package**. The
zero-vendor-import property MUST be established by the existing module-wide boundary-guard mechanism
rather than by human inspection, and that guard MUST NOT be modified, extended, or given a new
allowlist entry to accommodate this package.

#### Scenarios

- **S-L2H-001** — Given the consumer proof's source, when its declared package clause is read, then it
  is an external test package that is neither the neutral contract package nor the test-support package.
- **S-L2H-002** — Given the consumer proof, when it runs under the ordinary test command with `-race`,
  then it constructs a request, drains the stream to exactly one terminal event, and passes.
- **S-L2H-003** — Given a script whose step yields a failure, when the consumer proof drains it, then
  the drain ends on the typed failure and the test asserts the failure is inspectable through the
  neutral error surface without reading any vendor type.
- **S-L2H-004** — Given a script that holds mid-stream, when the consumer proof cancels the request
  context, then the stream closes within a bounded deadline and the test attributes closure to
  cancellation rather than to completion.
- **S-L2H-005** — Given the merged change, when the boundary guard's dependency walk runs over the whole
  module, then the new package is included in its sweep and reports zero vendor imports.
- **S-L2H-006** — Given the merged diff, when a reviewer looks for edits to the boundary-guard test or
  its allowlist, then none exists.

### R-L2H-002 — Four runnable examples are compiled and run by the ordinary test command

Package documentation MUST ship exactly four runnable examples covering **request construction**,
**streaming**, **tool-call reconstruction**, and **error inspection**. Each MUST be a runnable example —
one that is executed and whose output is verified by the test run — not a compile-only fragment and not
a fenced code block in a comment. All four MUST be compiled **and run** by `cd backend/agent && make test`
under `-race`, so a drift between an example and the surface it demonstrates fails the build rather than
rotting silently. The examples MUST attach to the documentation surface a Layer 2 author reads, and MUST
NOT require a credential, a network call, or wall-clock sleeping.

#### Scenarios

- **S-L2H-007** — Given the merged change, when the test run executes, then four runnable examples are
  discovered and run, and each verifies its own output.
- **S-L2H-008** — Given the four examples, when their subjects are enumerated, then they are exactly
  request construction, streaming, tool-call reconstruction, and error inspection — one each.
- **S-L2H-009** — Given an example whose expected output is altered by one character, when the test run
  executes, then that example fails; given the surface it demonstrates changes incompatibly, it fails to
  compile.
- **S-L2H-010** — Given the example file, when its import closure is read, then it requires no
  credential, performs no network I/O, and no example orders events with a sleep.
- **S-L2H-011** — Given the examples that consume the test-support package from an external test package,
  when the package is built, then no import cycle exists and compilation succeeds.

### R-L2H-003 — The nine-row capability matrix is published entry-for-entry, with its generator and its reopen trigger

Package documentation MUST publish the capability matrix as **nine rows** — one per capability in
`CAP-R-01…05` and `CAP-O-01…04` — each row carrying the same capability, standing, and outcome as the
committed expectation, entry-for-entry and in the same order. The published matrix MUST be **transcribed
from** that expectation and MUST NOT be regenerated, re-decided, re-ordered, or amended by this change.
The publication MUST name the conformance test that generates the record, so a reader can re-derive the
rows. The `CAP-O-01` reasoning row MUST be published as a **struck verdict with a stated reopen trigger**,
naming the trigger identifiers and the ADR requirement inline; it MUST NOT be phrased as a permanent
property of the adapter.

#### Scenarios

- **S-L2H-012** — Given the published matrix, when its rows are compared to the committed expectation,
  then there are exactly nine rows and every row matches in capability, standing, outcome, and order.
- **S-L2H-013** — Given the published matrix, when a reader looks for its provenance, then it names the
  conformance test that generates the record.
- **S-L2H-014** — Given the `CAP-O-01` row, when its text is read, then it names the reopen trigger
  identifiers and states that reopening requires an ADR.
- **S-L2H-015** — Given the `CAP-O-01` row, when it is searched for language asserting reasoning is
  permanently or inherently unsupported, then none is present.
- **S-L2H-016** — Given the merged diff, when a reviewer looks for edits to the conformance suite's
  behavior, to any adapter, or to the committed expectation, then none exists.

### R-L2H-004 — The published matrix is protected by a read-only drift guard

A test MUST parse the published rows out of the documentation file and compare them entry-by-entry to
the committed expectation, failing when any row differs, is missing, is added, or is reordered. The
guard MUST be **read-only**: it MUST NOT alter conformance-suite behavior, adapter behavior, or the
committed expectation, and MUST NOT require any of them to become exported. It MUST resolve the
documentation file by relative path using the established in-repository mechanism, and MUST run under
the ordinary test command.

#### Scenarios

- **S-L2H-017** — Given the merged change, when the drift guard runs under the ordinary test command,
  then it passes.
- **S-L2H-018** — Given a single published row altered in capability, standing, outcome, or position,
  when the drift guard runs, then it fails and its message names the differing row.
- **S-L2H-019** — Given a row deleted from the published matrix, and separately a tenth row added, when
  the drift guard runs, then it fails in both directions and names the count mismatch.

### R-L2H-005 — Item 6's wire clause is published as not exercisable in v1, without reopening AI-17's stream half

Package documentation MUST publish completion-checklist item 6's **wire** clause as **not exercisable in
v1**, naming its cause: `AI-26.6` landed as a refusal and `AI-29.2` is struck by AI-29, so there is no
reasoning on this wire to round-trip and no v1 node can close the wire half. The **stream** half —
closed by **AI-17** (`R-ARE-009` / `R-ARE-010`) and recorded on the **G12(b)** spine row — MUST be named
in the same statement as **unaffected and not reopened**. The distinction language MUST be lifted from
doc 0002 line 2446 and the AI-40.2 amendment at line 2402, not paraphrased. The publication MUST NOT
tick item 6's checkbox and MUST NOT strike, weaken, or restate AI-17's closure as open.

#### Scenarios

- **S-L2H-020** — Given the published duty, when it is read, then it states the wire clause is not
  exercisable in v1 and names `AI-26.6`'s refusal and `AI-29.2`'s striking as the cause.
- **S-L2H-021** — Given the published duty, when it is read, then the same statement names AI-17,
  `R-ARE-009` and `R-ARE-010` and asserts the stream half is unaffected and not reopened.
- **S-L2H-022** — Given the published duty and doc 0002 lines 2402 and 2446, when the distinction
  language is compared, then it is lifted from those lines rather than paraphrased.
- **S-L2H-023** — Given the merged change, when doc 0002's completion checklist is read, then item 6 is
  still `[ ]`, and no artifact in this change ticks it or claims it closed.

### R-L2H-006 — Layer 2's obligation to strip vendor reasoning on the wire is published beside duty 1

Package documentation MUST publish, **beside and not in place of** R-L2H-005's clause, that Layer 2 MUST
strip the OpenRouter `reasoning_details` field on the wire. The publication MUST attribute the obligation
to AI-29's recorded absence — a deliberate decision, not an oversight — and MUST cite AI-29's
`decision.md` by identifier so a reader can reach the original record.

#### Scenarios

- **S-L2H-024** — Given the published documentation, when both inherited duties are located, then both
  are present, adjacent, and neither replaces the other.
- **S-L2H-025** — Given the strip-reasoning duty, when its justification is read, then it attributes the
  obligation to AI-29's recorded absence and cites AI-29's `decision.md` by identifier.

### R-L2H-007 — The v1 surface is enumerated, declared frozen, and its experimental parts are marked

The compatibility statement MUST enumerate the v1 surface **by capability/behavior**, declare it frozen
as of this milestone, and state exactly what Layer 2 may rely on. Anything not frozen MUST be marked
**experimental** in the same enumeration, so a consumer can tell the two apart without inference; if no
part is experimental, the statement MUST say so explicitly rather than omitting the category. Package
documentation MUST carry a short pointer paragraph naming that a freeze happened and giving the path to
the statement, plus identifier-level detail where identifiers are native to that surface; it MUST NOT
duplicate the enumeration.

#### Scenarios

- **S-L2H-026** — Given the compatibility statement, when its surface enumeration is read, then each
  entry is stated as a capability or behavior and is marked either frozen or experimental.
- **S-L2H-027** — Given the compatibility statement, when a reader asks what Layer 2 may rely on, then
  the answer is stated directly and does not require reading source.
- **S-L2H-028** — Given package documentation, when a consumer runs the documentation tool over the
  neutral contract package, then a paragraph states that the v1 surface is frozen and gives the
  statement's path.
- **S-L2H-029** — Given package documentation and the compatibility statement, when both are compared,
  then the documentation carries a summary and a path, not a second copy of the enumeration.

### R-L2H-008 — All eighteen completion-checklist items are walked, each citing its closing node

The compatibility statement MUST walk all **eighteen** completion-checklist items **in order**, each row
citing the node or nodes that closed it, using the traceability spine as the template — including item
6's struck `AI-29.2` and item 18's `AI-40.1` / `AI-40.3`. The walk MUST **report** each item's status as
of the milestone that closed it and MUST NOT re-decide or re-verify the underlying evidence. In doc 0002,
items **11, 12, 14, 15, 16, 17** MUST each be ticked **individually with its own closing evidence cited**,
never as a blanket sweep; item **18** ticks at this change's close; item **6** MUST remain `[ ]` by
design. The status line MUST read 42 of 42, and a dated AI-40 close amendment blockquote MUST follow the
established pattern.

#### Scenarios

- **S-L2H-030** — Given the compatibility statement, when its walk is read, then eighteen rows appear in
  checklist order and every row cites at least one closing node.
- **S-L2H-031** — Given the walk, when item 6's row is read, then it cites the struck `AI-29.2` and
  states the wire half is not exercisable in v1; when item 18's row is read, then it cites `AI-40.1` and
  `AI-40.3`.
- **S-L2H-032** — Given doc 0002 after this change, when items 11, 12, 14, 15, 16 and 17 are read, then
  each is `[x]` and each has its own one-line evidence citation naming the closing milestone's amendment.
- **S-L2H-033** — Given doc 0002 after this change, when item 6 is read, then it is `[ ]`; when item 18
  is read, then it is `[x]`; when the status line is read, then it reads 42 of 42.
- **S-L2H-034** — Given the walk, when a row is inspected for re-verification of an already-closed
  item's evidence, then none is performed — the row reports status and cites the closing node.

### R-L2H-009 — The never-cancelled abandoned-consumer posture is documented as contract

The compatibility statement MUST restate, as documented contract, that the **caller owns the context**,
the **producer blocks until the context ends**, and **abandoning a stream without cancelling is a
contract violation by the consumer**. It MUST name the abandoned-**then-cancelled** coverage
(`AI-23.5`, `AI-33.3`) as the tested neighbour, and MUST mark the never-cancelled case explicitly as
**untestable to termination** — stating why documentation is the enforcement mechanism rather than a
test. It MUST NOT claim the never-cancelled case is covered by a test.

#### Scenarios

- **S-L2H-035** — Given the compatibility statement, when the abandoned-consumer section is read, then
  all three clauses appear: caller owns the context, producer blocks until the context ends, abandoning
  without cancelling violates the contract.
- **S-L2H-036** — Given that section, when its evidence is read, then `AI-23.5` and `AI-33.3` are named
  as the abandoned-then-cancelled coverage.
- **S-L2H-037** — Given that section, when the never-cancelled case is read, then it is marked untestable
  to termination with a stated reason, and no claim of test coverage for it appears.

### NFR-L2H-A — Additive only: no relocation, no behavior change, no new dependency

This change MUST be additive. It MUST NOT relocate `src/agenttest` or the neutral contract package, and
MUST NOT introduce any file move — the sibling relationship on which the existing signature guard's
relative-path resolution depends MUST hold unchanged. It MUST NOT modify conformance-suite behavior,
adapter behavior, the committed expectation, or the AI-00.3 boundary guard. `backend/agent/go.mod` and
`go.sum` MUST be byte-identical to base.

#### Scenarios

- **S-L2H-038** — Given the merged diff, when it is inspected for renames or moves, then none exists and
  the signature guard resolves its target and passes unmodified.
- **S-L2H-039** — Given the merged diff, when `go.mod` and `go.sum` are compared to base, then both are
  byte-identical.
- **S-L2H-040** — Given the change reverted in isolation, when the module's tests run, then they pass
  exactly as before and nothing on the base branch depended on anything this change created.

### NFR-L2H-B — Gates and evidence

Every test-list item of AI-40.1 … AI-40.3 MUST be taken red → green → refactored **in order** under
strict TDD, with both outputs recorded in `tasks.md`. The milestone MUST close on recorded green
`cd backend/agent && make test` (`go test -race -v ./...`), `make lint` reporting 0 issues, and
`make build` exiting 0.

#### Scenarios

- **S-L2H-041** — Given `tasks.md`, when a reviewer walks the milestone's test-list items, then each
  carries recorded red output, recorded green output, and a refactor note.
- **S-L2H-042** — Given the merged change, when the test, lint and build commands run from
  `backend/agent`, then the test run is green under `-race`, lint reports 0 issues, and build exits 0.

---

## Acceptance criteria

1. An external-package test constructs a request, invokes the fake, drains events, handles a scripted
   error **and** a cancellation, and its zero-vendor-import property is proven by the existing boundary
   guard rather than by inspection (`R-L2H-001`).
2. Four runnable examples cover request construction, streaming, tool-call reconstruction and error
   inspection, and are compiled **and run** by `make test` (`R-L2H-002`).
3. The nine-row matrix is published entry-for-entry identical to the committed expectation, citing its
   generating test and naming the `CAP-O-01` reopen trigger (`R-L2H-003`).
4. The published matrix cannot rot silently: a differing, missing, added or reordered row fails the
   drift guard (`R-L2H-004`).
5. Item 6's wire clause is published as not exercisable in v1, with AI-17's stream-half closure
   explicitly unaffected and not reopened (`R-L2H-005`).
6. Layer 2's obligation to strip `reasoning_details` is published beside duty 1, attributed to AI-29
   (`R-L2H-006`).
7. The v1 surface is enumerated and frozen, experimental parts marked, with a documentation pointer
   (`R-L2H-007`).
8. All eighteen checklist items are walked with closing-node citations; doc 0002 items 11/12/14/15/16/17
   and 18 are ticked per-item, item 6 stays `[ ]`, and the status line reads 42 of 42 (`R-L2H-008`).
9. The never-cancelled abandoned-consumer posture is documented as contract, with the tested neighbour
   named and the case marked untestable to termination (`R-L2H-009`).
10. No move, no behavior change, no dependency change; test/lint/build gates green (`NFR-L2H-A`,
    `NFR-L2H-B`).
11. `ai-provider-conformance-suite` acceptance item 10 reads **nine** entries (see the sibling delta in
    this change folder).

## Traceability to the charter

| Charter node / duty | Requirement |
|---|---|
| AI-40.1 — consumer proof | `R-L2H-001` |
| AI-40.2 (1) — runnable examples | `R-L2H-002` |
| AI-40.2 (2) — capability matrix published | `R-L2H-003`, `R-L2H-004` |
| AI-40.2 duty 1 — item 6 wire clause (doc 0002 line 2402) | `R-L2H-005` |
| AI-40.2 duty 2 — Layer 2 strips reasoning | `R-L2H-006` |
| AI-40.3 (1) — frozen v1 surface | `R-L2H-007` |
| AI-40.3 (2) — eighteen-item checklist walk | `R-L2H-008` |
| AI-40.3 (3) — never-cancelled abandoned consumer | `R-L2H-009` |
| Proposal reconciliation + gates | `NFR-L2H-A`, `NFR-L2H-B`, delta on `ai-provider-conformance-suite` |

## Left to design

This spec deliberately does not decide: the file and package names carrying each obligation; the parse
strategy the drift guard uses to read the published rows; whether the streaming and tool-call examples
stay in one file or split across two packages (proposal D3's documented fallback); or the exact wording
lifted for R-L2H-005. `sdd-design` owns all four, and owes a compile-only proof that the external test
package's import of the test-support package creates no cycle (`S-L2H-011`) before any example body is
written.
