# Design — the model provider interface

> **Change**: `cachicamas-ai-model-provider` · **Milestone**: AI-20 · **Phase**: design · **Date**: 2026-08-01
> **Predecessors**: proposal (obs #2347) · spec (obs #2353, `R-AMP-001…021`, `S-AMP-001…066`)
> **Reconciliation gate — CONFIRMED 2026-08-01 (sdd-tasks)**: re-read AI-14's landed
> `design.md` (`Event`, `EventKind`, `EventKindError` via append recipe) and AI-19's landed
> `design.md` (`*Failure`, `ErrorEvent`, `ErrorPayload`, `FailureCategoryCancellation`,
> `ErrCancelled`, `PreStreamFailure`, `MidStreamFailure`). Every symbol this design cited
> generically now has a landed exact spelling below; no divergence found. AI-20's own apply
> still MUST run after AI-15…AI-19 have landed/merged in this worktree (AI-19 depends on
> AI-14+AI-15; AI-20 depends on AI-19), so this is a name check, not an apply clearance.

## Technical Approach

One new file `provider.go` declares `ModelProvider` (one method, `Stream`) and the one askable
optional contract `TokenCounter`. `request.go` gains one accessor, `Request.IsZero`. All proof lives
in tests: `ai_test` proves pre/mid-stream contracts with a milestone-local producer; `agenttest`
proves external implementability, discovery, clean absence, and holds the AST signature guard.
Stdlib only (`go/ast`, `go/parser`, `runtime`) — `go.mod` stays at zero requires (NFR-AMP-A).

## Interfaces / Contracts

```go
// provider.go — imports ONLY "context" (guard-enforced allowlist).
type ModelProvider interface {
	// GoDoc carries, in substance: the eight AI-02.1 § 9 ownership statements,
	// AI-03.1 § 9's enumerability clause, the validation-before-cancellation
	// order (R-AMP-006/007), and the single sanctioned loss path (R-AMP-012).
	Stream(ctx context.Context, req Request) (<-chan Event, error)
}

type TokenCounter interface { // CAP-O-02, the ONLY askable optional in v1
	CountTokens(ctx context.Context, req Request) (TokenCount, error)
}

func (r Request) IsZero() bool // request.go
```

`Event` is AI-14's landed envelope (`V-STR-10`, `backend/agent/src/ai/event.go`). Mid-stream
failure rides as AI-19's terminal error event: kind `ai.EventKindError` (appended to AI-14's
registry per its append recipe), payload `*ai.Failure`, constructed by
`ai.ErrorEvent(f *ai.Failure) (ai.Event, error)` and read back via
`(ai.Event).ErrorPayload() (*ai.Failure, bool)`. Pre-stream failures return `*ai.Failure`
(constructed by `ai.PreStreamFailure(ai.FailureReport) (*ai.Failure, error)`, category
`ai.FailureCategoryCancellation`, sentinel `ai.ErrCancelled`, for the already-cancelled case) or
AI-04's `*Violation` (for the invalid-request/zero-request case) — no new sentinel (NFR-AMP-B).

## Architecture Decisions

### Decision: `Request.IsZero` reads only message emptiness

**Choice**: `return len(r.messages) == 0`. **Rationale**: `freeze()` is the sole constructor of a
non-zero `Request`, rule 2 requires ≥ 1 message, and fields are unexported, so "no messages" ⇔
"never constructed" — one fact, per `MessageID.IsZero`/`Segment.IsZero` precedent. Corollary: since
construction is the only validation door (AI-10), a provider's whole pre-stream validation
obligation (`R-AMP-006`) reduces to this check; the accessor makes it provable from `agenttest`.
**Rejected**: minted identity on `Request` (changes `Equal`, new state for an answered question);
`model=="" && len==0` (two facts for one question).

### Decision: milestone-local producer is an unexported func + struct in `provider_test.go`

**Choice**: package `ai_test`; unexported `scriptProvider{events []ai.Event, terminal error, buffer int}`
implementing `ai.ModelProvider` so tests exercise the real signature. Canonical loop: one goroutine,
`defer close(out)` (the single closing site, `R-AMP-009`), every send inside
`select { case out <- ev: case <-ctx.Done(): return }` (`R-AMP-010/011`); terminal error appended via
`ai.ErrorEvent(f *ai.Failure) (ai.Event, error)` (AI-19, landed). **Rejected**: exported helper or options struct —
that is AI-21's fake, which this change blocks (`R-AMP-013`); deliberate crudeness is the fence.

### Decision: guard = `runtime.Caller(0)` + syntactic AST match, import allowlist `{"context"}`

**Choice**: `provider_signature_guard_test.go` resolves
`filepath.Join(filepath.Dir(thisFile), "..", "ai", "provider.go")` from `runtime.Caller(0)`
(ADR 0005 Guard C sibling layout), parses with `parser.ParseFile`, then asserts syntactically (no
`go/types`): interface `ModelProvider` exists with EXACTLY one method `Stream`; params exactly
(`context.Context` selector, `Request` ident); results exactly (`ast.ChanType{Dir: RECV, Elt: Event ident}`,
`error`); file imports ⊆ `{"context"}`. Unresolvable/unparseable target → `t.Fatalf` naming the path
(`R-AMP-015`), never skip. "Exactly one method" also mechanizes half of `R-AMP-021`; the other half is
`agenttest`'s compile proof that a Stream-only stub satisfies `ModelProvider`. **Rejected**:
`go/types` loading (heavier, needs build context); textual grep (not AST, spec forbids); vendor-name
blocklist (unverifiable offline — the allowlist is stronger and total).

### Decision: bite proof = two COMPILABLE mutations, AI-00.3 record-and-revert

**Choice**: each mutation must still compile so the observed red is the guard's, not the compiler's
(a real vendor import cannot even build under zero-requires `go.mod`). Mutation 1 (vendor-type
stand-in): `req Request` → `req json.RawMessage` + `import "encoding/json"` — trips allowlist and
param assertion. Mutation 2 (changed carrier): `<-chan Event` → `<-chan string` — trips the carrier
assertion. Procedure per mutation: apply → `go test ./src/agenttest/ -run TestModelProviderInterface_SignatureGuard`
→ paste red output into `tasks.md` evidence → revert → confirm green (`R-AMP-016`, S-AMP-043…045).

### Decision: discovery by type assertion on the provider value; `TokenCounter` returns `TokenCount`

**Choice**: `counter, ok := provider.(ai.TokenCounter)`; `ok == false` IS the clean absence — no
error, no zero, no fallback (`R-AMP-018`). Result type reuses AI-13's `TokenCount`
presence-separate-from-value idiom; the `error` result carries provider failures through AI-19.
**Rejected**: aggregate `Capabilities()` query method (the exact widening `R-AMP-021` pins against);
boolean+int returns (re-derives what `TokenCount` already owns).

## Data Flow

    caller ──Stream(ctx, req)──▶ provider
       pre-stream: req.IsZero → *Violation(ErrEmpty) │ then ctx.Err →
                    ai.PreStreamFailure({Category: FailureCategoryCancellation}) → *ai.Failure (ai.ErrCancelled)
       (no carrier, no goroutine) ──error──▶ caller
       else HANDOVER (AI-02.1 § 7): ◀─chan Event── one producer goroutine
       events… ─▶ terminal (ai.ErrorEvent(*ai.Failure) EventKindError | completion) ─▶ close   [cancel+full buffer ⇒ bare close]

## File Changes

| File (`backend/agent/`) | Action | Description |
|---|---|---|
| `src/ai/provider.go` | Create | `ModelProvider`, `TokenCounter`, contract GoDoc (imports: `context` only) |
| `src/ai/request.go` | Modify | Add `Request.IsZero` |
| `src/ai/provider_test.go` | Create | AI-20.2/20.3 tests + `scriptProvider` (package `ai_test`) |
| `src/agenttest/provider_test.go` | Create | External stub, discovery, clean absence, method-set pin |
| `src/agenttest/provider_signature_guard_test.go` | Create | AST guard + bite evidence pointer |
| `src/ai/doc.go` | Modify | One boundary paragraph |

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (`ai_test`) | Pre-stream order, zero-request rejection, one close, send-vs-cancel race, bounded close, sanctioned loss | `scriptProvider` under `-race`; strict TDD per `tasks.md` item |
| External (`agenttest`) | Stub implements+compiles+exercised; discovery; clean absence; Stream-only conformance | Type assertions on stub values |
| Guard | Signature, carrier, allowlist, loud failure | AST test + two recorded bites |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or
process-integration boundary. The guard parses one in-repo source file via stdlib, no subprocess.

## Migration / Rollout

No migration. Purely additive; full revert restores Wave 1 state.

## Open Questions

- [x] AI-19's terminal-error payload and pre-stream failure type exported names — re-verified
      against AI-14's and AI-19's landed `design.md` on 2026-08-01 (sdd-tasks reconciliation
      step). Confirmed: `ai.Event`, `ai.EventKindError`, `ai.ErrorEvent`, `(ai.Event).ErrorPayload`,
      `*ai.Failure`, `ai.PreStreamFailure`, `ai.MidStreamFailure`, `ai.FailureCategoryCancellation`,
      `ai.ErrCancelled`. No name diverges from what this design assumed; still re-verify against the
      merged `event.go`/`provider_failure.go` bytes at `sdd-apply` time since AI-14/AI-19 code has
      not landed in this worktree yet — only their designs have.

> Word-budget deviation is deliberate house convention (sibling wave-2 designs carry this density).

---

> **Archive-time note, 2026-08-01 — this is the reconciliation gate that held best in the wave.**
>
> The last open question above was discharged at apply time by re-verifying not against AI-14's and AI-19's *designs* but directly against `event.go`, `provider_failure.go`, `sequence.go` and `request.go` **source**. The Wave 2 verify report § 3.6 records this as the strongest of the wave's five reconciliation gates, and it is the contrast case for AI-18's **D13**, which reconciled against design prose instead and had to be corrected mid-apply.
>
> Both bite mutations landed as designed and are recorded verbatim in `tasks.md:112,128` (`req Request` → `req json.RawMessage`; `<-chan Event` → `<-chan string`). Both produced guard-specific RED, both were reverted, and the revert was independently confirmed at wave verification: `provider.go` imports only `"context"` today, with no `encoding/json` residue.
>
> **One open suggestion at wave close: `S5`.** `R-AMP-004` requires the interface GoDoc to carry AI-02.1 § 9's eight ownership statements plus AI-03.1's enumerability clause. All nine are present in `provider.go:34-95` — but the only evidence is that a reader looked. Deleting rule 6 ("the stream's buffer is bounded; backpressure means waiting, never dropping") makes no test fail. The package already contains the exact tool to close this (`sequence_guard_test.go` and `TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule` both parse Go doc comments and assert on their content); roughly fifteen lines of test would make `R-AMP-004` mechanical.
