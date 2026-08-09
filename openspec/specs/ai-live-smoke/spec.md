# Spec — the opt-in live smoke test

> **Introduced by**: `cachicamas-ai-live-smoke` (AI-39, Wave 6 — Hand off)
> **Status**: live
> **Capability**: `ai-live-smoke` (**new** — full spec, no MODIFIED/REMOVED/RENAMED)
> **Requirement IDs**: `R-LSM-0NN` · **Scenario IDs**: `S-LSM-0NN` (both prefixes verified free across `openspec/specs/` and `openspec/changes/`)
> **Format**: RFC 2119 + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Composes (cited, not modified here)**: `ai-openrouter-first-provider` (`R-OR-07`, `R-OR-08` — deltaed in this change) · `ai-provider-conformance-suite` (`R-CNF-027`, `S-CNF-080`) · `ai-stream-testkit` (`R-STK-005`, `R-STK-006`) · `ai-model-provider`
> **Decision forks binding this spec**: proposal D1–D7, ratified at AI-39 close (change approved, reviewed, and archived 2026-08-09)

---

## Status

This specification is **live** as of 2026-08-09 (AI-39). It was introduced by `cachicamas-ai-live-smoke` and is canonical at `/openspec/specs/ai-live-smoke/spec.md`.

---

## Purpose

A live smoke test already ships, but four of AI-39.1's five acceptance items are unmet: nothing runs
the sentinel sweep over the live test's own output, the package has no `internal/` segment so it is
importable module-wide and out of tree, the stream-shape check is one disjunction instead of three
invariants, and no setup instructions ship with it. This capability states what MUST be true of the
live smoke as a contract, independent of which vendor is under test: how it opts in, how it is
bounded, what it may and may not assert, that its own output is proven silent about secrets even
when it fails, that no application entry point can reach it, and that a human can run it without
ever writing a credential into the repository.

## Definitions

- **The gate** — the two-stage environment condition that must open before any outbound request.
- **The live run** — one credentialled execution of the smoke test against the real provider.
- **The capture sink** — the single buffer through which the live run routes every diagnostic it produces.
- **The sweep** — the module's one shared sentinel-scan implementation (`R-CNF-027`, `S-CNF-080`).
- **The positive control** — the sweep's self-test, proving on this corpus that a planted leak is detected.
- **The guard** — the module-wide reachability test over the smoke package's import path.

---

## Requirements

### R-LSM-001 — The live path opens only behind a two-stage opt-in, and the default test run never depends on a credential

The live run MUST require BOTH a credential in the process environment AND a separate explicit
run-consent variable whose value equals the exact literal `1`. The credential alone MUST NOT be
sufficient consent. When either stage is closed the test MUST skip with a reason attributing which
stage closed it, and MUST make no outbound request. The repository's default test command MUST pass,
under the race detector, with neither variable set, and MUST NOT be modified to supply either.

#### Scenarios

- **S-LSM-001** — Given neither variable set, when the default test command runs, then the smoke test
  reports a skip whose reason names the missing stage, the run exits successfully, and no outbound
  request is made.
- **S-LSM-002** — Given the credential present but the run-consent variable absent, empty, or any
  value other than the exact literal `1` (including `true`, `yes`, `0`, `" 1 "`, `"1\n"`), when the
  test runs, then it skips and no outbound request is made.
- **S-LSM-003** — Given both stages open, when the gate is evaluated, then it reports open and the
  live path is entered.

### R-LSM-002 — The live run issues exactly one `provider.Stream` invocation, bounded by a hard timeout it cannot outlive

The live run MUST issue exactly one `provider.Stream` invocation per execution; it MUST NOT itself
retry, loop, or issue a second invocation to obtain a pass. That invocation's underlying adapter
carries its own already-ratified HTTP-layer retry policy (AI-35, `retry.Loop`, default
`MaxAttempts = 3`), which this change MUST NOT modify (R-LSM-008): a single live run's one invocation
may therefore still result in up to four billed HTTP attempts on a retryable transport failure. The
request context and the drain of its stream MUST both be bounded by the same hard deadline, so a hung
or slow provider fails the test rather than hanging the suite. The deadline, the approximate per-run
cost, and the retry-driven cost multiplier MUST be stated in the package's shipped instructions.

#### Scenarios

- **S-LSM-004** — Given an open gate, when the live run executes, then exactly one `provider.Stream`
  invocation is issued and both the request context and the stream drain carry the same hard deadline.
- **S-LSM-005** — Given a provider that never produces a terminal event, when the deadline elapses,
  then the test fails on the deadline rather than blocking, and the failure is attributable to the
  timeout.
- **S-LSM-006** — Given the merged live run, when a reviewer looks in the live run's own code for a
  retry, a loop, or a second `provider.Stream` invocation, then none exists; the adapter's own
  ratified HTTP-layer retry policy is out of this scenario's scope and, unmodified by this change, may
  still issue up to four HTTP attempts underneath that one invocation.

### R-LSM-003 — Three stream-shape invariants are asserted separately, and no assertion reads model content

The live run MUST assert, as three independently failing checks: (1) a response-start event is
present; (2) at least one content event follows it; (3) exactly one terminal event exists, in
terminal position. A single disjunction over event kinds MUST NOT stand in for these three. The
exactly-one-terminal check MUST be an assertion, not a diagnostic log line. No assertion MUST depend
on, compare, or match the model's generated text, tool arguments, token counts, or any other
provider-chosen content; only event kinds, counts, order, and presence are admissible. An empty event
stream MUST fail.

#### Scenarios

- **S-LSM-007** — Given a drained live stream, when the three invariants are evaluated, then each is a
  separate check, and suppressing any one of the three leaves the other two still able to fail.
- **S-LSM-008** — Given a recorded stream with a response-start and a terminal but no content event
  between them, when the invariants run, then the content-presence check fails specifically.
- **S-LSM-009** — Given a recorded stream carrying two terminal events, when the invariants run, then
  the terminal-exclusivity check fails; and given an empty stream, when they run, then the run fails
  rather than passing vacuously.
- **S-LSM-010** — Given the merged assertions, when a reviewer looks for any comparison against model
  text, tool-argument bytes, or token counts, then none exists.

### R-LSM-004 — Every diagnostic the live run produces is swept, with a positive control, before it is published

The live run MUST route every diagnostic it produces — request metadata, drain outcome, error
renderings, and terminal diagnostics — through a single capture sink, and MUST NOT publish any of
those bytes to the test log until the sweep has run clean over that sink. The sweep MUST run on the
success path AND on every failure path, by construction rather than by a call in each branch. The
deny list MUST include at least the credential's environment-variable name, a prefix of the secret
value, and the planted prompt marker. Before a clean sweep result is trusted, the positive control
MUST pass over the same deny list; a clean result obtained without it MUST be treated as no result.
A sweep match MUST fail the test naming only the vector, MUST NOT reproduce the matched bytes, and
MUST state that output was withheld.

#### Scenarios

- **S-LSM-011** — Given a live run whose diagnostics are routed to the capture sink, when the run
  ends by any path, then the positive control has passed and the sweep has run over the sink before
  any sink byte reaches the test log.
- **S-LSM-012** — Given a live run that fails — on the gate-open path, on the request, on the drain,
  or on a stream-shape invariant — when the run ends, then the sweep has still run over the captured
  diagnostics for that path.
- **S-LSM-013** — Given a deliberately planted leak of each deny-list vector in turn, when the sweep
  runs, then the test fails for each, and the failure message names the vector, states that output
  was withheld, and contains neither the credential nor the planted prompt.
- **S-LSM-014** — Given the positive control forced to fail, when the run executes, then the run
  fails rather than reporting a clean sweep.
- **S-LSM-015** — Given a clean sweep, when the run completes, then the captured diagnostics are
  released to the test log so a maintainer loses no information.

### R-LSM-005 — Unreachability is enforced by the compiler and regression-detected by a guard that cannot pass vacuously

The smoke package's import path MUST contain an `internal` directory segment placed so that only
packages within the provider-adapter subtree may import it; that subtree MUST contain no application
entry point. A package outside that subtree, in this module or any other, MUST fail to compile when
it imports the smoke path. In addition, a guard MUST assert, deny-by-default over the module's
package closure, that no non-internal package has the smoke path in its transitive test-inclusive
dependency set. The guard MUST fail loudly when its package pattern resolves to zero packages, so it
cannot pass by observing nothing. The guard MUST state, in its own source, that this module has zero
composition roots today, that a literal walk from a composition root is therefore vacuous, and that
its value is failing the day an in-module importer appears. A skipped placeholder test MUST NOT be
used in place of this guard.

#### Scenarios

- **S-LSM-016** — Given the merged package path, when it is read, then it contains an `internal`
  segment under the provider-adapter subtree, and that subtree contains no `package main` entry point.
- **S-LSM-017** — Given a package outside that subtree that imports the smoke path, when the module is
  built, then compilation fails on import visibility.
- **S-LSM-018** — Given the guard, when it runs against the module's package closure, then no
  non-internal package has the smoke path in its transitive test-inclusive dependencies; and given a
  non-internal package that does import it, when the guard runs, then it fails naming that importer.
- **S-LSM-019** — Given the guard's package pattern narrowed to resolve to zero packages, when the
  guard runs, then it fails on the anti-vacuity check rather than passing.
- **S-LSM-020** — Given the guard's source, when it is read, then it states the zero-composition-root
  limitation of what it proves today; and when the merged test set is enumerated, then no skipped
  placeholder stands in for this proof.

### R-LSM-006 — Credential-safe setup instructions ship in the package directory

Instructions MUST ship in the smoke package's own directory. They MUST name both environment
variables, MUST instruct the reader to provide the credential through a shell export in the invoking
process only, MUST carry the exact test invocation valid for the package's merged path, and MUST
state the timeout bound and the approximate per-run cost. They MUST state explicitly that the
credential is never written to any file inside the repository. Following them MUST NOT require
creating, editing, or committing any repository file, and no example in them may show a credential
written to a file, a dotenv file, a shell history-persisted assignment, or a repository path.

#### Scenarios

- **S-LSM-021** — Given the merged package directory, when its contents are enumerated, then setup
  instructions are present in that directory.
- **S-LSM-022** — Given those instructions, when they are read, then both environment-variable names,
  a shell-export-only credential step, the exact invocation for the merged path, the timeout bound,
  and the approximate per-run cost are each present.
- **S-LSM-023** — Given those instructions, when a reader follows them end to end, then no repository
  file is created or edited, and no step writes the credential to a file.
- **S-LSM-024** — Given the merged instructions, when the stated invocation is executed against the
  merged path with the gate closed, then it resolves the package and skips, proving the path is not
  stale.

### R-LSM-007 — Relocating the package preserves the single-sweep convergence proof and every published name

Moving the smoke package MUST NOT delete a test, MUST NOT remove a published name consumers are
bound to, and MUST NOT widen the shared test-helper package's exported surface. The single-sweep
obligation (`R-CNF-027` / `S-CNF-080`) MUST remain proven after the move: the smoke side MUST assert
its own scan is equivalent to the shared sweep core, and the suite side MUST keep asserting its own
scan is equivalent to that same core, so convergence on one implementation holds from both sides. The
module MUST still contain exactly one sweep implementation.

#### Scenarios

- **S-LSM-025** — Given the merged module, when its sweep implementations are enumerated, then exactly
  one exists, and both the smoke side and the suite side are each proven equivalent to it over a
  corpus and a planted needle.
- **S-LSM-026** — Given the pre-move test set and the merged test set, when they are compared, then no
  test was deleted and every previously published name is still published.
- **S-LSM-027** — Given the shared test-helper package's exported surface before and after the change,
  when they are compared, then no name was added.

### R-LSM-008 — The live smoke adds no dependency, no entry point, and no scheduled or CI-triggered run

This change MUST add zero new `require` lines: the module's dependency set (`go.mod`/`go.sum`) MUST
stay byte-identical to `origin/main`. (The module may already carry pre-existing `require` lines from
earlier, independently-ratified changes — this requirement bounds what THIS change adds, not the
module's total declared count.) It MUST NOT create a composition root, a user-facing command, or any
CI workflow file, and MUST NOT introduce a scheduled, push-triggered, or otherwise automatic
billing-consuming run. The provider adapter under test MUST NOT be modified to make the live run pass.
Canonical behaviour MUST change only through this change's delta specs; no file under
`openspec/specs/` is edited in place before archive.

#### Scenarios

- **S-LSM-028** — Given the merged change, when `go.mod`/`go.sum` are diffed against `origin/main` and
  the import guards are read, then the diff is empty — no `require` line was added or removed — and
  both guards pass.
- **S-LSM-029** — Given the merged diff, when a reviewer looks for a new `package main` entry point, a
  CI workflow file, or any automatic trigger of the live run, then none exists.
- **S-LSM-030** — Given the merged diff, when a reviewer looks for a change to the adapter under test
  or an in-place edit under `openspec/specs/`, then none exists.

---

## Traceability to AI-39.1

| AI-39.1 item / reconciliation | Requirements | Scenarios |
|---|---|---|
| 1 — skips cleanly without credentials; default run never depends on it | `R-LSM-001` | `S-LSM-001`…`S-LSM-003` |
| 2 — one bounded request under a hard timeout | `R-LSM-002` | `S-LSM-004`…`S-LSM-006` |
| 2 — start, ≥1 content event, exactly one terminal; never model output | `R-LSM-003` | `S-LSM-007`…`S-LSM-010` |
| 3 — output never carries credential or full prompt, even on failure | `R-LSM-004` | `S-LSM-011`…`S-LSM-015` |
| 4 — unreachable from any entry point, proven mechanically | `R-LSM-005` | `S-LSM-016`…`S-LSM-020` |
| 5 — credential-safe setup instructions ship with the package | `R-LSM-006` | `S-LSM-021`…`S-LSM-024` |
| Reconciliation D2 — `S-CNF-080` convergence survives the relocation | `R-LSM-007` | `S-LSM-025`…`S-LSM-027` |
| Charter out-of-scope fence + no-CI posture (ADR 0005) | `R-LSM-008` | `S-LSM-028`…`S-LSM-030` |

## Acceptance criteria

1. Both gate stages are required, a closed gate skips attributably, and the default run is green with
   neither variable set (`R-LSM-001`).
2. Exactly one `provider.Stream` invocation per run — which the adapter's unmodified, ratified retry
   policy may expand to at most four HTTP attempts — bounded by one hard deadline shared by request
   and drain (`R-LSM-002`).
3. Three separately failing stream-shape invariants, none of which reads model content
   (`R-LSM-003`).
4. Positive control passes, then the sweep runs over the captured sink on the success path and every
   failure path, before any diagnostic is published (`R-LSM-004`).
5. The compiler refuses an out-of-subtree import, and the deny-by-default guard fails anti-vacuously
   and documents its zero-composition-root scope (`R-LSM-005`).
6. Setup instructions ship in the package directory and never require a credential file in the
   repository (`R-LSM-006`).
7. The relocation deletes no test, publishes no new helper name, and leaves exactly one sweep
   implementation proven from both sides (`R-LSM-007`).
8. Zero new `require` lines — the dependency set stays byte-identical to `origin/main` — no entry
   point, no CI workflow, no automatic billing-consuming run, no in-place canonical spec edit
   (`R-LSM-008`).
