# Proposal — the model provider interface

> **Change**: `cachicamas-ai-model-provider` · **Milestone**: AI-20 · **Nodes**: AI-20.1 … AI-20.5
> **Phase**: proposal · **Date**: 2026-08-01 · **Project**: cachicamas (witsaba)
> **Wave**: 2 "Stream" — the wave's **join point**; apply lands last in the wave
> **Depends on**: AI-03, AI-10, AI-12, AI-14, AI-19 · **Blocks**: AI-21 and everything after it
> **Input**: `exploration.md` (Engram `sdd/cachicamas-ai-model-provider/explore`)
> **Lineage**: AI-20 under doc 0002. The retired pre-2026-07-30 AI-16 cycle that shared this
> change name is superseded; its artifacts live under Engram `retired/old-plan/*`.

---

## Intent

Layer 1 has a complete vocabulary and no outward face. Nothing above it can be written —
not the fake, not the conformance suite, not an adapter, not the agent loop — until one call
exists that takes a normalized request and hands back a normalized event stream.

Everything AI-20 declares already exists. Its work is not deciding; it is **spelling four
already-shipped decisions into one Go declaration and pinning them mechanically**: AI-02.1
fixed the carrier, AI-03.1 fixed optional-capability discovery, AI-10/AI-12 fixed the request,
AI-04/AI-19 fixed how failures are reported. Left unspelled, each downstream milestone
re-derives them, differently.

## Scope

### In scope — three locks

1. **The interface shape.** One streaming method: context and normalized request in, the AI-02
   carrier plus an `error` out. **No vendor type and no wire type on the boundary** — pinned by
   an AST guard in `src/agenttest/` that resolves `../ai` from its own source file, plus an
   external-package stub proving a non-`ai` package can implement it.
2. **The pre-stream / mid-stream split.** The boundary is **handover**, not the first event
   (AI-02.1 § 7). Pre-stream: the failure is returned, no carrier and no goroutine ever exist,
   with validation ordered before the already-cancelled-context case. Mid-stream: the failure
   is the single terminal event, one sender, one closing site, every send waiting on
   cancellation, and the one sanctioned loss path stated so a consumer that treats a missing
   terminal after its own cancellation as corruption is the party in error.
3. **Optional capability is discovered, not required.** AI-03.1 § 9's mechanism, restated and
   **not reopened**: one separate contract per capability, asserted on the provider value; the
   core interface never widens. In v1 only `CAP-O-02` token counting is askable and gets a
   contract. `CAP-O-01` and `CAP-O-03` are receive-side — observed, never discovered. Absence
   is a clean absence: not an error, not a zero, and never a Layer 1 fallback estimate.

Plus the interface's documentation carrying AI-02.1 § 9's eight ownership statements.

### Out of scope

- Any concrete adapter and any vendor SDK (wave 4); retry policy (AI-35); buffer sizing (AI-34).
- The scripted fake (AI-21) and the conformance suite (AI-23) — both blocked *by* this change,
  so AI-20.3 builds its own single-purpose test producer rather than borrowing one.
- Event kinds and payloads (AI-14); the failure taxonomy's categories (AI-19). AI-20 names the
  carrier and the delivery paths, never what rides on them.
- Reopening AI-02.1 or AI-03.1. A change there is an amendment to those specs, in the PR that
  needs it — never a local judgement here.

## Capabilities

### New Capabilities

- `ai-model-provider`: the provider interface — its signature, its pre-stream and mid-stream
  contracts, the boundary guard, and optional-capability discovery.

### Modified Capabilities

- None. `ai-stream-lifecycle`, `ai-minimum-capabilities` and `ai-contract-vocabulary` are
  **read and cited by identifier, never modified**.

## Approach

`exploration.md` finds no architectural freedom left and recommends the single-streaming-method
shape with Go-idiomatic optional interfaces; the alternative — capability query methods on the
core interface — is rejected by inheritance, since it is the exact widening AI-20.5 item 3's
pin exists to catch. This proposal adopts that recommendation.

Four items are **implementation detail, deferred to `sdd-design`**, none of which reopens an
upstream decision: an externally-visible "was this ever constructed" accessor on `Request` so
AI-20.2's contract is testable from `agenttest` (precedent: `MessageID.IsZero`,
`Segment.IsZero`); AI-20.3's own minimal test producer; the guard's target-file resolution via
`runtime.Caller`; and the guard's bite-proof procedure, which follows AI-00.3's existing
manual-mutation-and-revert precedent rather than inventing a mechanism.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/provider.go` | New | The interface, the eight-statement contract in GoDoc, the optional token-counting contract |
| `backend/agent/src/ai/provider_test.go` | New | AI-20.2 / AI-20.3 with a single-purpose in-file producer |
| `backend/agent/src/ai/request.go` | Modified | Possibly one accessor for the zero/never-constructed question |
| `backend/agent/src/agenttest/provider_test.go` | New | AI-20.1 item 2 stub, AI-20.5 discovery and clean absence |
| `backend/agent/src/agenttest/provider_signature_guard_test.go` | New | AI-20.4 AST guard + its bite proof |
| `backend/agent/src/ai/doc.go` | Modified | Package contract paragraph naming the new boundary |

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| AI-14 and AI-19 have not landed; the carrier's element type and the terminal error's shape are charter contracts, not code | Certain | Apply is ordered last in the wave by design. Spec and design cite AI-14/AI-19 by charter item and re-verify at apply; guessing a type name is a defect, not a shortcut |
| Guard fragility — `runtime.Caller` path resolution breaks silently if `src/ai` or `src/agenttest` moves | Low | Already recorded in `agenttest/doc.go` and ADR 0005 Guard C. The bite proof is what makes the guard's failure loud |
| Scope drift into a fake, an adapter, or a widened interface | Medium | Any new vendor import, capability query method, or reusable double is a signal the scope moved — stop and reopen, do not absorb |
| Review budget over 400 lines | Certain | Wave budget accepted up front as `exception-ok` (5000 lines); leaf boundary is the commit boundary |

## Rollback Plan

Every change is additive: new files, one possible accessor, one doc paragraph. No existing
signature moves, so no shipped caller compiles differently. Reverting this milestone's commits
leaves Layer 1 exactly as Wave 1 left it. `git revert` per leaf commit is a clean boundary, and
because nothing downstream exists yet, the blast radius of a full revert is zero.

## Dependencies

- **AI-14** (event envelope) and **AI-19** (failure taxonomy) merged — hard, blocking.
- AI-03, AI-10, AI-12 merged. Layer 1 stays stdlib-only; `go.mod` at zero requires until AI-24.
- Strict TDD; `make test` from `backend/agent/` (`go test -race -v ./...`).

## Success Criteria

- [ ] One streaming method takes a context and a normalized request and returns the AI-02
      carrier plus an error, with no vendor or wire type anywhere in the signature.
- [ ] A package that is not `ai` implements the interface, compiles, and is exercised.
- [ ] An invalid request fails with a typed error before any carrier or goroutine exists; an
      already-cancelled context reports its category, after validation.
- [ ] The producer closes the stream exactly once on completion, terminal error and
      cancellation; cancellation closes within bounded time under `-race`, with no send
      after close, and the saturated-buffer path closes bare exactly as sanctioned.
- [ ] A consumer discovers token counting on a provider that advertises it and observes a
      clean absence on one that does not; a provider implementing only the required surface is
      fully conformant.
- [ ] The AST guard passes, and both bite mutations — a vendor type, a changed carrier — fail
      it; recorded and dropped.
- [ ] `make test` green and `make lint` clean in `backend/agent/`; both AI-00 import guards pass.
