# Design — the stream test kit

> **Change**: `cachicamas-ai-stream-testkit` · **Milestone**: AI-22 · **Phase**: design · **Date**: 2026-08-02

## Technical Approach

Five new files under `backend/agent/src/agenttest/` (`stream_kit_*.go`, one concern per file, milestone-ID headers like AI-21's `fake_*` files — the distinct prefix avoids colliding with the fake's namespace). All helpers take `<-chan ai.Event` or `[]ai.Event`, never `*agenttest.Provider`, so they work against any `ai.ModelProvider`. `src/ai` is read-only; `go.mod` unchanged; stdlib only.

## Architecture Decisions

### D1 — 22.1 drain API: channel in, `Recording` out, fail-fast on deadline

**Choice**: `DrainAndRecord(tb testing.TB, ch <-chan ai.Event, timeout time.Duration) Recording` — the two precedents' select loop (`requireClosedWithin`, `drainFake`), promoted verbatim: `tb.Fatalf` on deadline, never hang. `Recording.Events()` returns a fresh copy so one drain backs many assertions. Exported `DefaultDrainTimeout = 2 * time.Second` restates both precedents' constant.
**Rejected**: coupling to `*agenttest.Provider` (excludes future real providers); returning `error` (every caller rewrites the fatal); `*testing.T` (TB also serves benchmarks/fuzz, and `Helper`/`Fatalf`/`Setenv` all live on TB).

### D2 — 22.2 diff: exact detection, bounded rendering, registry-driven summaries

**Choice**: `RequireSameEvents(tb, got, want []ai.Event)` finds the first divergence with `reflect.DeepEqual` per index (length mismatch reports at the shorter length), then reports `index, got-summary, want-summary`. Summaries come from a `map[ai.EventKind]func(ai.Event) string` table using the 12 exported accessors (`TextDelta()`, `ToolCallEnd()`, …): structural fields rendered whole (block, finish reason, tool name/id, ids), free-form bytes (deltas, args, tokens) as `len=N head="…"` capped at `summaryRuneCap = 32` runes with an elision marker — never dumped. A test asserts table keys ≡ `ai.EventKinds()`, so a new kind fails loudly instead of going unsummarized.
**Rejected**: comparing by summary string (truncation collisions fake equality — detection must be exact, rendering bounded); reusing `Event.String()` (deliberately payload-free; the kit's bounded rendering is a test-only diagnostic surface, which is why it lives in `agenttest`, not `ai`).

### D3 — 22.3 ordering: wrap `CheckStream`, new contiguity pass (honors D10)

**Choice**: `CheckContiguity(events []ai.Event) error` walks encounter order asserting seq 1, then `prev+1`; a violation names the missing sequence and both neighbours: `missing seq 4 between event[2] (text_delta seq=3) and event[3] (text_block_end seq=5)`. `RequireValidStream(tb, rec Recording)` runs `ai.CheckStream` first (AI-14.4's invariants, verdict shape untouched), then `CheckContiguity`.
**Rejected**: reimplementing AI-14.4's rules (per AI-21 design D10 and the proposal — descriptor changes would need double maintenance).

### D4 — 22.4 leak helper: opt-in, serial-only, mechanically enforced via `t.Setenv`

**Choice**: `RequireNoGoroutineLeak(tb testing.TB, scenario func())` — opt-in (never automatic). It runs `scenario` `leakRepeats = 50` times between `runtime.NumGoroutine()` snapshots, settles 50ms, asserts `after <= before + leakRepeats/2` (the `fake_cancellation_test.go` S-AFP-031 amplification: real leaks grow ~linearly with repeats, sibling jitter does not). Serial-only contract: process-wide counts cannot attribute goroutines under `t.Parallel()`, and Go offers no API to query a caller's parallelism — so the helper calls `tb.Setenv` on a sentinel variable, which the testing package itself panics on in a parallel test. The doc contract becomes a mechanical pin, this codebase's house style.
**Rejected**: `go.uber.org/goleak` — needs its own ADR (`AGENTS.md` rule 5) and breaks `doc.go`'s dependency-free pin until AI-24; this rejection record is 22.4's `[decision]` deliverable. Also rejected: widened tolerance to "support" parallel use (unbounded band = no assertion); automatic instrumentation on every drain (false positives in every existing parallel test).

### D5 — 22.5 iterator: `iter.Seq` view with post-loop `Err()`

**Choice**: Go 1.26.3 ⇒ range-over-func. `NewIter(ch <-chan ai.Event) *Iter`; `(*Iter).Events(ctx context.Context) iter.Seq[ai.Event]` yields every event (terminal error event included) until channel close, `ctx.Done()`, or the caller breaks; `(*Iter).Err() error` after the loop returns the terminal error event's `*ai.Failure` (via `ErrorPayload()`), else `ctx.Err()` if cancellation ended the loop, else nil — range-over-func cannot return a value, so the error surfaces `bufio.Scanner`-style. The view never owns, drains, or closes the stream (V-STR-22); it lives entirely in `agenttest`, so AI-20.4's guard (which parses only `src/ai/provider.go`) passes unmodified.
**Rejected**: `Next() (Event, bool)` pull iterator (non-idiomatic at this Go version; `iter.Pull` exists for callers who need it); swallowing the error event from the yield (the view must stay a faithful view).

## Data Flow

    any ModelProvider.Stream ──→ <-chan ai.Event
            │                          │
      DrainAndRecord (22.1)      NewIter(ch).Events(ctx) (22.5, view only)
            │
        Recording ──→ RequireSameEvents (22.2)
            └───────→ RequireValidStream (22.3) = ai.CheckStream + CheckContiguity
    RequireNoGoroutineLeak(tb, scenario) (22.4) — wraps whole scenarios, serial-only

## File Changes

| File (`backend/agent/src/agenttest/`) | Action | Description |
|---|---|---|
| `stream_kit_record.go` | Create | 22.1 `DrainAndRecord`, `Recording`, `DefaultDrainTimeout` |
| `stream_kit_diff.go` | Create | 22.2 `RequireSameEvents`, summary table |
| `stream_kit_ordering.go` | Create | 22.3 `CheckContiguity`, `RequireValidStream` |
| `stream_kit_leak.go` | Create | 22.4 helper; header records the goleak rejection + ADR trigger |
| `stream_kit_iter.go` | Create | 22.5 `Iter` |
| `stream_kit_{record,diff,ordering,leak,iter}_test.go` | Create | One proof file per leaf (`agenttest_test`) |
| `doc.go` | Modify | Name the test kit alongside the fake |

## Interfaces / Contracts

```go
func DrainAndRecord(tb testing.TB, ch <-chan ai.Event, timeout time.Duration) Recording
func (r Recording) Events() []ai.Event // fresh copy; Len() int
func RequireSameEvents(tb testing.TB, got, want []ai.Event)
func CheckContiguity(events []ai.Event) error
func RequireValidStream(tb testing.TB, rec Recording)
func RequireNoGoroutineLeak(tb testing.TB, scenario func()) // serial-only (t.Setenv pin)
func NewIter(ch <-chan ai.Event) *Iter
func (it *Iter) Events(ctx context.Context) iter.Seq[ai.Event]
func (it *Iter) Err() error
```

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (per leaf) | Deadline fatal (never-closing producer), replay without re-drain, first-divergence report bounded, seeded gap names N + neighbours, leak caught on a deliberately-leaking scenario, iterator error/cancel | RED-first per strict TDD, from `agenttest_test`, against AI-21's fake |
| Exhaustiveness | Summary table covers `ai.EventKinds()` | Registry-comparison test, `event_registry_test.go` style |
| Guards | AI-20.4 signature guard, AI-00 import guards | Must pass unmodified; `make test`/`make lint` from `backend/agent/` |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Test-only stdlib helpers.

## Migration / Rollout

No migration. Purely additive; `requireClosedWithin`/`drainFake` stay untouched (D-3 of the proposal round). Rollback = revert leaf commits.

## Open Questions

None blocking.
