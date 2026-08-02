# Proposal — the stream test kit

> **Change**: `cachicamas-ai-stream-testkit` · **Milestone**: AI-22 · **Nodes**: AI-22.1 … AI-22.5
> **Phase**: proposal · **Date**: 2026-08-02 · **Project**: cachicamas (witsaba)
> **Wave**: 3 "Prove" — second milestone; ships in the wave's single PR with AI-21 and AI-23
> **Depends on**: AI-21 (shipped in this worktree) · **Blocks**: AI-23, AI-33, doc 0003's hardening wave
> **Input**: Engram `sdd/cachicamas-ai-stream-testkit/explore`

---

## Intent

Wave 3 has a producer (AI-21) and no concise way to assert against it. The same timeout-safe drain
loop is already written twice — `ai/provider_test.go:251` and `agenttest/fake_text_test.go:42` —
because no importable version exists; AI-23 would make it three. And nothing anywhere asserts
sequence contiguity: `ai.CheckStream` disclaims it in its own doc comment and names AI-22.3 as its
owner (design D10).

AI-22 makes `V-PRV-11` importable so no downstream test invents its own, and so a broken producer
fails on a deadline instead of hanging the suite.

## Scope

### In scope

1. **Timeout-safe drain and record (22.1)** — generalizes both ad hoc helpers; the recording is
   replayable across several assertions without re-draining.
2. **Readable event diffs (22.2)** — first divergence by index, kind and a *bounded* payload
   summary. New surface: `Event.String()` deliberately never renders payload bytes.
3. **Ordering and gap assertions (22.3)** — `ai.CheckStream` for AI-14.4's ordering, plus new
   contiguity: starts at 1, no gaps, and a gap reported precisely — which sequence is missing,
   between which two events.
4. **Leak-detection mechanism (22.4, `[decision]`)** — hand-rolled `runtime.NumGoroutine`
   accounting promoted from `fake_cancellation_test.go`'s repeated-call amplification. Must resolve
   default-vs-opt-in **and** the `t.Parallel()` interaction that precedent flags but does not solve.
5. **Iterator view (22.5)** — the `V-STR-22` carrier view AI-02.1 delegated here. Lives in
   `agenttest`; AI-20.4's signature guard passes unmodified.

### Out of scope

- **A general-purpose testing framework** (charter). Every helper asserts something a Layer 1
  contract already states.
- **A third-party leak detector (e.g. goleak)** — rejected here, not forbidden forever: it needs
  its own ADR (`AGENTS.md` rule 5) and `agenttest/doc.go` pins this package dependency-free until
  AI-24. Recording the rejection *is* 22.4's deliverable.
- **Any edit to `src/ai`** — `stream_check.go`, `sequence.go` and `provider.go` are read-only.
- The conformance suite (AI-23), which consumes these helpers.
- **Deferred but related.** (a) Leak assertions on the abandoned-**never-cancelled** path:
  `ai-stream-lifecycle` § 5 rules it untestable and scopes 22.4 to the abandoned-then-cancelled and
  cancellation paths. (b) W1/W2, the two Wave-2 carryovers AI-21 parked — still unassigned; do not
  absorb them here.

## Capabilities

### New Capabilities

- `ai-stream-testkit`: `V-PRV-11` — drain and record, event diffing, ordering and gap assertions,
  the leak-detection mechanism, and the `V-STR-22` carrier view.

### Modified Capabilities

- None. `ai-stream-lifecycle`, `ai-event-envelope`, `ai-model-provider` and the sibling
  `ai-fake-provider` (AI-21, not yet archived — read from its active change folder) are **cited by
  identifier, never modified**.

## Approach

Add the first helper files to `src/agenttest/`, one concern per file, mirroring AI-21's layout and
its milestone-ID "why before what" headers. 22.3 **composes rather than duplicates**: `ai.CheckStream`
for kind/block/terminal ordering, plus a separate walk over `Sequence()` for contiguity —
reimplementing AI-14.4's logic is rejected because any future descriptor change would then need
updating twice.

Two items are real work for `sdd-design`, not decided here: 22.4's parallel-test interaction and
tolerance band, and 22.2's bounded summary shape (per-kind extraction, registry-driven so a new
`EventKind` cannot silently go unsummarized).

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/agenttest/*.go` (`package agenttest`) | New | Drain/record, diff, assertions, leak helper, iterator view |
| `backend/agent/src/agenttest/*_test.go` | New | One proof file per AI-22 leaf |
| `backend/agent/src/agenttest/doc.go` | Modified | Name the test kit alongside the fake |
| `backend/agent/src/ai/**` | Read-only | `CheckStream`, `Stamper`, `Event` accessors; unchanged |
| `ai/provider_test.go`, `agenttest/fake_*_test.go` | Unchanged | Local helpers stay; migration is not this change |

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| 22.4's counting helper is flaky under `t.Parallel()` | **High** | Named as `sdd-design`'s job above; the decision must state serial-only, a widened band, or a documented incompatibility — not copy the precedent verbatim |
| 22.2's summary misses a new `EventKind` | Medium | Registry-driven exhaustiveness, as `content_part.go` / `event_registry_test.go` already do |
| Wave review budget exhausted before AI-23 | **High** | AI-21 verified at 2810 lines of the wave's 5000; AI-22 estimated **~700–1100** (impl ~250–350, tests ~400–600, openspec ~100–150), leaving ~1100–1500 for AI-23's 9 leaves — comparably scoped to AI-21's 2810. `sdd-tasks` must forecast this explicitly; slicing AI-23 into its own PR is the likely outcome |
| Scope creep into a testing framework | Medium | Charter's stated non-goal; every helper must cite the contract it asserts |
| Iterator view read as a second contract | Low | `V-STR-22` is a view that never owns or closes the stream; AI-20.4's guard is the mechanical pin |

## Rollback Plan

Every change is additive: new files in `src/agenttest/` plus one doc comment. No shipped signature
moves, no existing test rewritten, so `src/ai` and AI-21 compile and pass identically with or
without this change. `git revert` per leaf commit is a clean boundary. AI-23 consumes these helpers
and lands in the same PR, so reverting AI-22 alone requires reverting AI-23 too; otherwise revert
the wave branch, which returns the tree to main @ `37898c7`.

## Dependencies

- AI-21 merged into this worktree's branch. Stdlib + the module's own `ai` package only — `go.mod`
  gains nothing and **no ADR is triggered**; 22.4's decision exists precisely to keep it that way.
- Strict TDD; `make test` from `backend/agent/` (`go test -race -v ./...`), `make lint` clean.

## Success Criteria

- [ ] A producer that never closes fails the drain helper on a deadline — the run never hangs.
- [ ] One recording backs several assertions without re-draining, preserving order.
- [ ] Two differing sequences produce a failure naming the **first** divergence by index and kind,
      with payload content bounded — never dumped verbatim.
- [ ] AI-14.4's invariants and 1-based contiguity both assert over a recorded stream; a seeded gap
      names the missing sequence and its two neighbours.
- [ ] The leak mechanism is chosen with the third-party alternative recorded as rejected, its
      default-vs-opt-in stated, and its `t.Parallel()` behaviour stated — not left open.
- [ ] Abandoned-then-cancelled and cancellation paths carry leak assertions.
- [ ] An iterator-shaped loop over a stream surfaces the terminal error after the loop and respects
      cancellation, while AI-20.4's signature guard passes **unmodified**.
- [ ] `make test` green, `make lint` clean, AI-00 import guards still pass, `go.mod` unchanged.
