// AI-21 — the scripted fake provider: an exported, importable
// ai.ModelProvider that reproduces AI-20's pre-stream and mid-stream
// physics exactly — contract-faithful, not convenient — driven by a
// per-call Script a test supplies instead of a vendor.
//
// Its mid-stream physics are ai/provider_test.go's scriptProvider,
// reproduced rather than approximated (R-AFP-004, AI-21's non-negotiable
// requirement): one producer goroutine per stream, one closing site that
// runs on every exit path after the last send attempt, and every send —
// the terminal one included — selecting on cancellation as well as on the
// stream. On top of that shape this milestone adds a script queue
// (AI-21.8), per-stream sequencing through a fresh ai.Stamper (AI-21.1) and
// request capture (AI-21.6).

package agenttest

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"sync"

	"github.com/cachicamas/backend/agent/src/ai"
)

// ErrScriptsExhausted is the sentinel a Stream call against an exhausted
// script queue wraps (R-AFP-020). It is a fixture defect — the caller asked
// for more calls than the fake was scripted with — reported as a typed
// error rather than a captured testing.TB.Fatal: Layer 2's agent loop calls
// Stream from its own goroutines, where TB.Fatal is unsafe.
//
// # Exhaustion behavior, stated explicitly (NFR-AFP-D)
//
// A Stream call made once every scripted Script has been consumed fails
// loudly and immediately, synchronously within the call — it never blocks,
// never hangs, and starts no producer goroutine. It returns a nil channel
// (no usable carrier) and a non-nil error satisfying
// errors.Is(err, ErrScriptsExhausted), wrapped with fmt.Errorf naming the
// failed call's 1-based ordinal and the total number of scripts queued
// ("Stream call %d of %d scripted"). It never replays the last consumed
// script, and it never returns a channel that simply closes as though the
// call had succeeded with no events — either outcome would teach a
// consumer the wrong physics silently. This is the one documented
// exception to this package's totality guarantee: every other exported
// entry point, given any input, is panic-free.
var ErrScriptsExhausted = errors.New("agenttest: script queue exhausted")

// Provider is the scripted fake: an ai.ModelProvider consuming one Script
// per Stream call, in the order NewProvider was given them (AI-21.8).
//
// Every field is guarded by mu: a Provider holds mutable state — the queue
// index and the captured request history — and concurrent Stream calls
// against the same value must stay -race-clean (R-AFP-003).
type Provider struct {
	mu       sync.Mutex
	scripts  []Script
	next     int
	requests []ai.Request
}

// NewProvider constructs a fake queued with scripts, one consumed per
// Stream call, in the order given.
func NewProvider(scripts ...Script) *Provider {
	return &Provider{scripts: slices.Clone(scripts)}
}

// Stream implements ai.ModelProvider (R-AFP-001). It runs AI-20's pre-stream
// contract in its documented order — validation, then nil-context
// normalization, then cancellation — before this milestone's own addition,
// exhaustion, checked last so validation-before-cancellation fidelity
// (R-AFP-013) is preserved. See [ErrScriptsExhausted] for exhaustion's own
// documented behavior in full.
func (p *Provider) Stream(ctx context.Context, req ai.Request) (<-chan ai.Event, error) {
	if req.IsZero() {
		return nil, ai.Invalid(ai.ErrEmpty, ai.At("request"))
	}

	// NFR-AMP-D, restated for the fake: a nil context must not panic. It is
	// treated as an uncancelled background context.
	if ctx == nil {
		ctx = context.Background()
	}

	if err := ctx.Err(); err != nil {
		failure, ferr := ai.PreStreamFailure(ai.FailureReport{
			Category: ai.FailureCategoryCancellation,
			Cause:    err,
		})
		if ferr != nil {
			// FailureCategoryCancellation is always a member of the
			// vocabulary, so this can never actually fail — handled
			// rather than ignored because errcheck requires it.
			return nil, ferr
		}
		return nil, failure
	}

	script, callNum, total, ok := p.consume(req)
	if !ok {
		return nil, fmt.Errorf("agenttest: Stream call %d of %d scripted: %w", callNum, total, ErrScriptsExhausted)
	}

	steps := stampSteps(script.Steps)
	if report := ai.CheckStream(stepEvents(steps)); report.Violation() != nil {
		// A script that misplaces an event — most commonly, one that
		// follows a terminal error (R-AFP-009) — is a fixture-authoring
		// mistake, decidable the moment the fake is driven, before any
		// goroutine starts: this is "the fake", the same posture Emit
		// takes for an unconstructed event (S-AFP-006, S-AFP-021).
		panic(fmt.Sprintf("agenttest: scripted stream violates ordering: %v", report.Violation()))
	}

	out := make(chan ai.Event, script.Buffer)
	go produce(ctx, out, steps)
	return out, nil
}

// stepEvents extracts the ordered events a script's Emit steps carry,
// skipping Hold steps (which carry none), for ai.CheckStream to validate
// before any goroutine starts.
func stepEvents(steps []Step) []ai.Event {
	events := make([]ai.Event, 0, len(steps))
	for _, step := range steps {
		if step.isHold {
			continue
		}
		events = append(events, step.event)
	}
	return events
}

// consume pops the next unconsumed script under mu, capturing req exactly
// when a script is popped (R-AFP-014): capture never happens for a call
// rejected earlier in Stream's pre-stream checks. It reports the script,
// this call's 1-based ordinal, the total scripted, and whether a script was
// available at all.
func (p *Provider) consume(req ai.Request) (script Script, callNum, total int, ok bool) {
	p.mu.Lock()
	defer p.mu.Unlock()

	total = len(p.scripts)
	if p.next >= total {
		return Script{}, p.next + 1, total, false
	}
	script = p.scripts[p.next]
	p.next++
	p.requests = append(p.requests, req)
	return script, p.next, total, true
}

// Requests returns every request captured by a call that consumed a
// script, in call order: Requests()[i] corresponds to script i
// (R-AFP-014). The result is a fresh slice on every call; each ai.Request
// it holds is already immutable through its own accessors, so only the
// slice that holds them needs cloning (R-AFP-015).
func (p *Provider) Requests() []ai.Request {
	p.mu.Lock()
	defer p.mu.Unlock()
	return slices.Clone(p.requests)
}

// ScriptsRemaining returns the number of scripted Scripts on this
// Provider that have not yet been consumed by a Stream call
// (R-CCP-009 / S-CCP-082). Tests that need to distinguish "no
// over-invocation" from "over-invocation masked by an exhausted-queue
// early return" use this to prove a spare script was available and
// left unconsumed; the shipped agenttest.Requests() cannot tell those
// two cases apart because R-AFP-020's ErrScriptsExhausted path never
// appends to p.requests.
func (p *Provider) ScriptsRemaining() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return len(p.scripts) - p.next
}

// stampSteps assigns a fresh ai.Stamper's sequence to every Emit step's
// event — one Stamper per Stream call, so sequencing is per-stream, never
// per-provider, and two streams from the same fake (or two different
// fakes) never share a counter (R-AFP-003). A Hold step carries no event
// and is passed through unstamped: only emitted events consume a sequence
// number.
func stampSteps(steps []Step) []Step {
	var stamper ai.Stamper
	out := make([]Step, len(steps))
	for i, step := range steps {
		if step.isHold {
			out[i] = step
			continue
		}
		out[i] = Step{event: stamper.Stamp(step.event)}
	}
	return out
}

// produce is the fake's one producer goroutine, run once per Stream call
// (R-AMP-009): it runs every step in order, then returns straight into the
// deferred close — the one closing site, reached on every exit path, after
// the last send attempt. Every send selects on ctx.Done() (R-AMP-010):
// with no receiver and a full buffer, this is also the sanctioned loss
// path — cancellation wins, the event (and everything scripted after it)
// is dropped, and the function returns into the deferred close, bare, with
// no terminal event forced through. A Hold step signals Reached, then
// blocks the identical way: selecting on the Gate's release as well as on
// ctx.Done(), so a never-released Gate still honors bounded cancellation.
func produce(ctx context.Context, out chan<- ai.Event, steps []Step) {
	defer close(out)

	for _, step := range steps {
		if step.isHold {
			step.gate.markReached()
			select {
			case <-step.gate.released:
			case <-ctx.Done():
				return
			}
			continue
		}
		select {
		case out <- step.event:
		case <-ctx.Done():
			return
		}
	}
}
