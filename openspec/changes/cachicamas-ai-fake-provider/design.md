# Design — the scripted fake provider

> **Change**: `cachicamas-ai-fake-provider` · **Milestone**: AI-21 · **Phase**: design · **Date**: 2026-08-02
> **Inputs**: proposal (Engram `sdd/cachicamas-ai-fake-provider/proposal`), exploration, `ai/provider_test.go`

## Technical Approach

Add the first `package agenttest` production files: an exported `Provider` implementing `ai.ModelProvider`, whose `Stream()` reproduces `scriptProvider`'s physics verbatim in behavior — `req.IsZero()` → `ai.Invalid(ai.ErrEmpty, ai.At("request"))`, nil ctx → background, `ctx.Err()` → `PreStreamFailure(FailureCategoryCancellation)`, then one producer goroutine, one `defer close`, every send `select`ing on `ctx.Done()`, sanctioned saturated-buffer drop with a bare close. On top: a step-based script vocabulary, a per-call script queue, a `Gate` synchronization primitive, and request capture. A fresh `ai.Stamper` per `Stream()` call gives independent 1…N sequencing (AI-21.1) for free.

## Architecture Decisions

### Decision: exhaustion fails via a typed pre-stream error, not a captured `testing.TB` (AI-21.8)

**Choice**: `Stream()` on an exhausted queue returns `nil, fmt.Errorf(...: %w, ErrScriptsExhausted)` naming the call ordinal and script count.
**Alternatives considered**: capturing `testing.TB` at construction and calling `Fatal`.
**Rationale**: Layer 2's agent loop calls `Stream()` from its own goroutines, and `TB.Fatal` from a non-test goroutine is undefined behavior (`runtime.Goexit` kills the wrong goroutine) — the fake's primary consumer makes the `TB` option unsafe, not merely inelegant. A typed error is loud (propagates through the caller's existing `(nil, err)` path), never hangs (no carrier, no goroutine), never repeats (index only advances), keeps `agenttest` a pure library usable outside `go test`, and matches AI-20's own pre-stream idiom. Ordering: exhaustion is checked **after** the three contract checks, so validation-before-cancellation fidelity (AI-21.5) is preserved; exhaustion is a fixture defect, deliberately not an `*ai.Failure` — it must never look like provider physics.

### Decision: `Gate` — a two-channel synchronization point (AI-21.4)

**Choice**: exported single-use `Gate` with `Reached() <-chan struct{}` (closed by the producer on arriving at a `Hold` step) and idempotent `Release()`. The producer at a hold runs `select { case <-released: case <-ctx.Done(): return }`.
**Alternatives considered**: caller-supplied bare channels (no reached-signal — tests would need sleeps to know the stream is held); time-based delays (banned by the milestone).
**Rationale**: `Reached()` is what removes wall clocks from assertions. Held-open recipe: `[Emit(start), Hold(g), Emit(end)]`, buffer 0 — after `<-g.Reached()` the test *knows* the stream is open and empty; `Release()` then drains to close. Saturation recipe: `Script{Buffer: n, Steps: [n emits, Hold(g), Emit(late)]}` — `<-g.Reached()` proves n unread events sit in the buffer (saturated, deterministically); cancelling then exits via the hold's `ctx.Done()` branch into the bare close, dropping the late event: the AI-20.3 drop path with zero sleeps. Holding selects on `ctx.Done()` so rule 5's bounded close survives a never-released gate. Both `chan struct{}` closes are `sync.Once`-guarded; one `Gate` per `Hold` step.

### Minor decisions

| Decision | Before (scriptProvider) | After | Rationale |
| --- | --- | --- | --- |
| Uniform `Emit` vocabulary | flat `[]ai.Event` + special `terminal error` field | terminal error is just `Emit(ai.ErrorEvent(f))` as last step | one code path; `MidStreamFailure`'s `outputPreceded` already covers both AI-21.3 states |
| No `CheckEmit` in the fake | n/a | events come from `ai` constructors (already validated) + `Stamper` (rule 2) | fake reproduces provider physics, not emission policing; scripts are trusted fixtures, per scriptProvider's own stance |
| `*Provider` with mutex | stateless value | pointer type; `sync.Mutex` guards queue index + captured requests | fake now holds mutable state; concurrent `Stream` calls must be `-race`-clean |
| Capture = script consumption | no capture | `Requests()[i]` ↔ script `i`, captured only when a stream starts | 1:1 invariant is legible; `Request` immutability (`slices.Clone` accessors) makes storing the value sufficient (AI-21.6) — only the slice is cloned on read |
| Per-script `Buffer int` | per-provider | per-call capacity, exactly (`cap(ch)`) | saturation recipes need per-call control; "a capacity, not a promise" wording carries over |

## Data Flow

```
test goroutine                     Provider.Stream                producer goroutine
     │  NewProvider(s1, s2)             │                               │
     │──Stream(ctx, req)───────────────▶│ IsZero? ctx nil? ctx.Err()?   │
     │                                  │ mu: exhausted? → ErrScriptsExhausted
     │                                  │ mu: capture req, pop script   │
     │◀──(<-chan Event, nil)────────────│──go produce(steps)───────────▶│ fresh Stamper
     │                                  │                               │ defer close(out)
     │◀════ out <- Stamp(ev) ═══ [Emit] every send: select ctx.Done ════│
     │◀─g.Reached() closed────── [Hold] select released / ctx.Done ─────│
     │──g.Release()──────────────────────────────────────────────────▶ │ resume steps
     │  (or cancel ctx ─────────────▶ bare close, rest dropped)         │
```

## File Changes

| File | Action | Description |
| --- | --- | --- |
| `backend/agent/src/agenttest/fake_provider.go` | Create | `Provider`, `NewProvider`, `Stream` physics, queue, capture, `ErrScriptsExhausted` (AI-21.1/.5/.6/.8) |
| `backend/agent/src/agenttest/fake_script.go` | Create | `Script`, `Step`, `Emit`, `Hold` (AI-21.1) |
| `backend/agent/src/agenttest/fake_gate.go` | Create | `Gate` (AI-21.4) |
| `backend/agent/src/agenttest/doc.go` | Modify | reframe: proof package **and** importable library |
| `backend/agent/src/agenttest/fake_{text,tool_call,error,gate,cancellation,request_capture,reasoning,queue}_test.go` | Create | one proof file per AI-21 leaf, `package agenttest_test` |

`fake_` prefix avoids colliding with the existing AI-20 `provider_test.go`; each file opens with the milestone-ID "why before what" header per `src/ai` convention.

## Interfaces / Contracts

```go
var ErrScriptsExhausted = errors.New("agenttest: script queue exhausted")

func Emit(ev ai.Event) Step   // Step is opaque: emit | hold
func Hold(g *Gate) Step

type Script struct {
        Steps  []Step
        Buffer int // channel capacity, exactly; 0 = unbuffered
}

func NewGate() *Gate
func (g *Gate) Reached() <-chan struct{} // closed when the producer arrives at its Hold
func (g *Gate) Release()                 // idempotent; unblocks the held producer

func NewProvider(scripts ...Script) *Provider // consumes one Script per Stream call
func (p *Provider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error)
func (p *Provider) Requests() []ai.Request // cloned slice; Requests()[i] ↔ script i
```

## Testing Strategy

| Layer | What to Test | Approach |
| --- | --- | --- |
| Unit (external, `agenttest_test`) | each AI-21 leaf: text, tool-call incl. zero-delta, both terminal-error states, gate hold/saturate, pre-/mid-stream cancellation, capture immutability, reasoning incl. redacted/signature-only never in text events, queue order + exhaustion | strict TDD; `make test` from `backend/agent/` (`go test -race -v ./...`); reuse `requireClosedWithin`-style bounded drains; no sleeps in scripted-schedule assertions — `Gate.Reached()` is the mechanism |
| Race | two fakes streaming concurrently sequence independently; repeated cancel iterations | `-race` with parallel subtests, mirroring AI-20.3's 20-iteration pattern |
| Guards | AI-00 import guards + AI-20 signature guard still pass | unchanged; `agenttest` imports stdlib + `src/ai` only |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. Purely additive; rollback per proposal (revert leaf commits or the wave branch).

## Open Questions

- None. Both flagged primitives (AI-21.4 gate, AI-21.8 exhaustion) are decided above; W1/W2 stay parked per the proposal.
