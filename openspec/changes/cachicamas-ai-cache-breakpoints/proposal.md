# Proposal — cache-boundary markers

> **Change**: `cachicamas-ai-cache-breakpoints` · **Milestone**: AI-11 · **Nodes**: AI-11.1 … AI-11.3
> **Phase**: proposal · **Date**: 2026-08-01 · **Project**: cachicamas (witsaba)
> **Branch**: `feat/ai-11-cache-breakpoints` · **Base**: `07d2027`
> **Depends on**: AI-10 · **Blocks**: AI-24, AI-26.2 · **Closes**: Layer 1 half of **G4**
> **Input**: `explore.md`

---

## Intent

The request cannot express a prompt-cache boundary. Caching is opt-in per breakpoint on at least one target provider, and cached reads cost roughly a tenth of fresh input, so a contract with nowhere to put a marker can never obtain that discount — and the omission is invisible until the bill arrives (doc 0001 § 3.2).

AI-10 landed the half that had to come first: the system instruction is ordered, individually markable segments from birth. This milestone lands the markers themselves, on all three carriers `V-REQ-23` names, plus the cap and the cascade an adapter reads them in.

## Scope

### In scope

- A cache-boundary marker on `Segment`, `Tool` and `Message`, set by a copy-returning method and read back from another package.
- `MaxCacheBoundaries` — a documented, exported cap, enforced at request validation with `ErrOutOfRange`.
- `Request.CacheBoundaries()` — the request's markers in **tools → system → messages** order, whatever order they were set in.
- The advisory contract, proven: a translator that ignores every marker still translates the request unchanged.

### Out of scope

- Measuring cache hit rates (`V-REQ-25` — Layer 1 never does).
- Usage accounting — AI-13.3 already carries `CacheRead` and `CacheWrite`; this milestone adds nothing there, and AI-11.3 item 2 pins it.
- Rendering markers onto a provider's wire shape (AI-24, AI-26.2); prefix stability across turns (Layer 2); deciding *where* breakpoints belong (Layer 2's pre-request hook).
- A marker on a tool choice, a content part or a generation option. `V-REQ-23` names three carriers; a fourth is additive later.

## Capabilities

### New Capabilities

- `ai-cache-breakpoints`: cache-boundary markers on the normalized request — placement, cap, cascade order, and the advisory contract.

### Modified Capabilities

- None. `openspec/specs/ai-contract-vocabulary/spec.md` is **read and cited, never modified**: `V-REQ-23`, `V-REQ-24` and `V-REQ-25` are already written and owned by AI-11.

## Approach

`explore.md` § 4 compares three placement shapes and two read shapes. Chosen: **a copy-returning `MarkCacheBoundary()` with an `IsCacheBoundary()` reader on all three types** — the only shape that moves no existing signature and keeps the mark *on* the thing, as `V-REQ-23` requires — plus **one enumerator** whose walk is shared by the adapter's read path and the cap rule's count, so a listing and a count cannot drift.

No new sentinel: `ErrOutOfRange`'s GoDoc already cites `V-REQ-24` by name.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/cache_boundary.go` | New | `CacheRegion`, `CacheBoundary`, `MaxCacheBoundaries`, the enumerator, the cap rule |
| `backend/agent/src/ai/{system_instruction,tool,message}.go` | Modified | one field, two methods each |
| `backend/agent/src/ai/request.go` | Modified | one rule row; `String()` names the boundary count |
| `backend/agent/src/ai/cache_boundary_test.go` | New | AI-11.1, AI-11.2 (`ai_test`) |
| `backend/agent/src/agenttest/cache_boundary_test.go` | New | AI-11.3 — marker-blind translator plus its control |

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| AI-10.3 … AI-10.6 land a different tool accessor, rule order or `Equal` decision | Medium | `explore.md` § 8 is the re-verification register: eight rows, each with file and symbol. Run it before the first test. |
| AI-10.6's equality ignores markers, so a round trip drops them silently | Low | `R-ACB-004` requires markers in equality; the test fails loudly if not. Escalate rather than patch around. |
| Review budget over 400 lines | Certain | Forecast recorded in `tasks.md`; wave budget accepted as `exception-ok`. Leaf boundary is the commit boundary. |

## Rollback Plan

Every change is additive: one new file, one new field on each of three types, one rule row. Reverting the milestone's commits removes the marker surface and leaves AI-10's request exactly as it was — no caller of the landed surface compiles differently, because no existing signature moves. `git revert` per leaf commit is a clean boundary.

## Dependencies

- AI-10 merged (leaves .1 … .6). This branch is based on `07d2027`; the rest is landing concurrently in `../ai-wave-1`.
- stdlib only. `go.mod` stays at zero requires until AI-24.

## Success Criteria

- [ ] A system segment, a tool declaration and a message each carry a marker that round-trips through construction and readback, from an external package.
- [ ] A marked request and its unmarked twin validate identically and differ in no region but marker state.
- [ ] A request over `MaxCacheBoundaries` fails before any I/O with `ErrOutOfRange` naming the excess.
- [ ] `Request.CacheBoundaries()` yields tools → system → messages regardless of set order.
- [ ] A translator that never reads a marker produces byte-identical output for a marked request and its unmarked twin, and a marker-aware control proves the assertion is not vacuous.
- [ ] The usage record is untouched.
- [ ] `make test` green and `make lint` clean in `backend/agent/`; `go.mod` at zero requires; both AI-00 import guards pass.
