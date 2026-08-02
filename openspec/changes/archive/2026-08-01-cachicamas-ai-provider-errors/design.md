# Design — the provider error taxonomy and the terminal error event

> **Change**: `cachicamas-ai-provider-errors` · **Milestone**: AI-19 (Wave 2 keystone)
> **Predecessors**: [`proposal.md`](proposal.md) · [`spec.md`](specs/ai-provider-errors/spec.md) · Engram `sdd/cachicamas-ai-provider-errors/{proposal,spec,explore}`
> **Status**: designed · **Apply gate**: AI-14 + AI-15 must land in this worktree first (NFR-AIP-E)

## Technical Approach

One new production file, `backend/agent/src/ai/provider_failure.go`, delivers the whole surface: a closed `FailureCategory` vocabulary (AI-04/AI-13 enum idiom), one concrete `*Failure` type carried by both delivery paths (`R-AIP-013`), per-category sentinels reached via an `Is` method, presence-typed retry-after (usage.go idiom), the two perpendicular axes of `R-AIP-010`, and the terminal error event integration appended to AI-14's registry. No new dependency; `errors` and `time` are standard library.

## Resolutions of the three flagged open items

1. **AI-19.6 split trigger — spec reading confirmed.** The trigger is *metadata-driven* growth ("category-specific metadata … grows the list past seven"), not base member count. This design adds **no category-specific field**: retry-after and status class are uniform across all nine categories. AI-19.6 is not appended; `sdd-tasks` re-checks against the landed count.
2. **Raw-label bound — drop-whole, not truncate, not construction failure.** See Decision D6.
3. **AI-14 sealed-payload — confirmed at design level; recipe pinned.** AI-14's `design.md` has now landed: `eventPayload interface { kind() EventKind; validate(at Path) *Violation }` is sealed (unexported methods, package `ai` only), matching spec `R-AEE-003`'s unsatisfiable-outside-`package ai` requirement (`S-AEE-007/008`). `R-AIP-001`'s "C4 is not expressible" claim stands unmodified. The Go spellings section below pins the exact `EventKindError`/`ErrorEvent`/`ErrorPayload` integration against AI-14's real `eventRegistry`/`eventRegistration`/`EventDescriptor` API and its `event_descriptor.go` 6-step kind-adding recipe — no longer provisional. `S-AIP-055` still re-verifies against landed code at the apply gate as a final check, not because the API is unknown.

## The Go spellings (Interfaces / Contracts)

```go
// The closed vocabulary — V-FAIL-06. finish_reason.go idiom exactly.
type FailureCategory uint8

const (
    _ FailureCategory = iota // zero value names no category
    FailureCategoryAuthentication
    FailureCategoryAuthorization
    FailureCategoryRateLimit
    FailureCategoryUnavailable            // unavailable/overloaded
    FailureCategoryTimeout
    FailureCategoryCancellation           // required, not optional (AI-02.1)
    FailureCategoryMalformedResponse
    FailureCategoryUnsupportedCapability
    FailureCategoryUnknown
    failureCategoryLimit                  // stays last; the append-only bound
)

func (c FailureCategory) String() string            // [failureCategoryLimit]string array, "invalid" placeholder
func (c FailureCategory) Validate(at ...Step) error // ErrNotInVocabulary via FirstFailure/Invalid
func FailureCategories() []FailureCategory          // fresh slice, 1..limit-1, stable order (S-AIP-015/016)

// Per-category errors.Is sentinels — array-pinned, exhaustiveness-tested.
var (
    ErrAuthentication, ErrAuthorization, ErrRateLimited,
    ErrUnavailable, ErrTimeout, ErrCancelled,
    ErrMalformedResponse, ErrUnsupportedCapability, ErrUnknownFailure error
)

// The delivery axis — V-FAIL-11/12 only; carries no output fact (R-AIP-012).
type DeliveryPath uint8
const (
    _ DeliveryPath = iota
    DeliveryPreStream
    DeliveryMidStream
    deliveryPathLimit
)

// Presence-typed retry-after input — mirrors usage.go's TokenCount/Tokens.
type RetryDelay struct{ /* d time.Duration; present bool */ }
func Delay(d time.Duration) RetryDelay

// Constructor input. Exported fields; zero value = "nothing reported".
type FailureReport struct {
    Category    FailureCategory
    Retryable   bool
    RetryAfter  RetryDelay
    RawLabel    string // bounded per D6; empty = none
    StatusClass int    // 1..5 (hundreds class); 0 = absent
    RequestID   string // bounded per D6; empty = none
    Cause       error
}

// The two constructors — the fourth cell (pre-stream × output) is unconstructible
// because PreStreamFailure takes no output flag (S-AIP-033).
func PreStreamFailure(r FailureReport) (*Failure, error)
func MidStreamFailure(r FailureReport, outputPreceded bool) (*Failure, error)

// The one concrete type — V-FAIL-05, both delivery paths.
type Failure struct{ /* all fields unexported */ }
func (f *Failure) Error() string                          // category text only; never the cause's (D5)
func (f *Failure) Unwrap() error                          // the cause — chain not severed (S-AIP-047)
func (f *Failure) Is(target error) bool                   // matches own category sentinel (D4)
func (f *Failure) Category() FailureCategory
func (f *Failure) Retryable() bool
func (f *Failure) RetryAfter() (time.Duration, bool)      // presence separate from value
func (f *Failure) PartialOutput() bool                    // V-FAIL-09; no delivery info in name
func (f *Failure) Delivery() DeliveryPath                 // the separate axis
func (f *Failure) RawLabel() string
func (f *Failure) StatusClass() (int, bool)
func (f *Failure) RequestID() string

// Terminal event integration — appended to AI-14's landed registry via its
// event_descriptor.go 6-step kind-adding recipe (finalized against the
// landed design; no longer provisional):
//   1. const EventKindError EventKind = eventKindEnd; move eventKindEnd = EventKindError + 1
//   2. *Failure implements eventPayload{ kind() EventKind; validate(at Path) *Violation } directly (D7)
//   3. constructor: func ErrorEvent(f *Failure) (Event, error) — rejects nil with ErrEmpty at At("payload")
//   4. accessor:    func (e Event) ErrorPayload() (*Failure, bool) — R-AEE-005-shaped typed accessor
//   5. registry line: eventRegistry = append(eventRegistry, eventRegistration{
//        name: "error",
//        descriptor: EventDescriptor{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: true},
//      })
//   6. doc.go kind list: add "error — terminal, at-most-one, no block role"
func ErrorEvent(f *Failure) (Event, error)          // rejects nil with ErrEmpty at At("payload")
func (e Event) ErrorPayload() (*Failure, bool)      // R-AEE-005-shaped typed accessor
```

Construction validates via `FirstFailure`: category through `Validate(At("category"))`; `StatusClass` outside 0..5 rejects with `ErrOutOfRange` at `At("statusClass")` (NFR-AIP-C — AI-04 sentinels only). Nil `*Failure` and the zero `Failure` never panic (NFR-AIP-B): `Error()` renders `"no provider failure"` / the invalid placeholder.

## Architecture Decisions

| # | Decision | Alternatives rejected | Rationale |
|---|---|---|---|
| D1 | One file `provider_failure.go`, external tests in `provider_failure_test.go` (`ai_test`), internal pins in `provider_failure_internal_test.go` | `errors.go` (too generic beside `validation.go`) | Screaming, matches `finish_reason.go`/`content_part.go` naming |
| D2 | `FailureCategory uint8`, `iota` from `_`, trailing bound, `[limit]string` names, `Validate` → `ErrNotInVocabulary`, slice-returning `FailureCategories()` | Open strings (violates V-FAIL-06); map-backed names | Verbatim shipped idiom; loop over `1..limit-1` makes an appended member appear with zero call-site edits (S-AIP-016) |
| D3 | Constructor input is one exported `FailureReport` struct + two constructors | Functional options (no house precedent); long positional lists | Zero value = "nothing reported", matching `Usage`; keeps both constructors' signatures stable as safe metadata grows |
| D4 | Category matching via `Is(target) bool` method; `Unwrap() error` returns only the cause; **no umbrella sentinel** | `Unwrap() []error{sentinel, cause}` (breaks `errors.Unwrap` callers, muddies "the cause"); umbrella `ErrProviderFailure` (becomes the lazy alternative R-AIP-014 forbids) | `errors.Is` consults `Is` at each hop then follows `Unwrap`, so sentinel match and cause chain both survive one wrap (S-AIP-045/047); "any provider failure?" is `errors.As(&f)` |
| D5 | `Error()` = fixed prefix + category's registered name (+ nothing else); never cause text, never the raw label | Rendering label/status inline | `Violation.Error()` precedent (`R-AIE-006`); machine fields stay accessors (S-AIP-031); redaction is a type property (S-AIP-028) |
| D6 | **Raw label & request ID bound: 64 bytes; over-long or control-character-bearing input is dropped whole (accessor reports empty); construction succeeds** | Truncate (a prefix of a secret is still a secret — `structuralName`'s recorded rule); reject construction (turns diagnostics into failures during failure handling, at the worst moment) | Real vendor codes (`context_length_exceeded`, `req_…` ids) fit in 64; anything longer is a body/credential in the wrong slot. Satisfies `S-AIP-019`'s "rejected" arm; label stays diagnostic, not mandatory (S-AIP-021) |
| D7 | `*Failure` itself implements AI-14's sealed payload interface; the terminal event's payload **is** the failure | Wrapper `errorPayload` struct | Makes `R-AIP-013`/`S-AIP-041` ("same concrete type on both paths") literal — nothing to keep in sync; legal because both compile in `package ai`, which is the C4-by-construction argument |
| D8 | Delivery = 2-member `DeliveryPath` enum, zero-value invalid; partial output = bare bool | 3-member shape enum (is G8 verbatim, prohibited by R-AIP-012); `MidStream() bool` (zero value would forge a legal pre-stream reading) | Axes stay perpendicular in the type; zero `Failure` reports an invalid delivery, consistent with "the zero value names nothing" |

## Data Flow

    adapter (other pkg)                          consumer
      │ PreStreamFailure(report) ──► *Failure ──► returned error ──► errors.Is / errors.As
      │ MidStreamFailure(report, outputPreceded)
      │        └──► *Failure ──► ErrorEvent(f) ──► Event{terminal} ──► e.ErrorPayload() ──► same *Failure

Both arrows end at the same concrete type — `R-AIP-015`'s identical accessor sets hold structurally.

## File Changes

| File | Action | Description |
|---|---|---|
| `backend/agent/src/ai/provider_failure.go` | Create | Category vocabulary, sentinels, `Failure`, `RetryDelay`, `FailureReport`, both constructors, label bounding, `ErrorEvent`/`ErrorPayload` |
| `backend/agent/src/ai/event.go` | Modify (append-only) | Step 1+5 of AI-14's `event_descriptor.go` 6-step recipe: `EventKindError` constant (moves `eventKindEnd`), `eventRegistry` append line with descriptor `{Role: BlockRoleNone, Cardinality: CardinalityAtMostOne, Terminal: true}` |
| `backend/agent/src/ai/doc.go` | Modify (append-only) | Step 6 of the recipe: add "error" to the kind-list paragraph |
| `backend/agent/src/ai/provider_failure_test.go` | Create | External `ai_test` package — S-AIP-001…050 incl. the C4 external-construction proof (no `agenttest` extension needed: `ai_test` is already "another package") |
| `backend/agent/src/ai/provider_failure_internal_test.go` | Create | Exhaustiveness pins: every category has a name, a sentinel, and enumeration coverage |

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Unit (external) | All 55 scenarios; C4 proof; wrap reach-through; planted-sentinel redaction; three shapes; both-paths parity | Table-driven in `ai_test`, strict TDD red→green per `tasks.md`, `make test` (`go test -race -v ./...`) |
| Unit (internal) | Name/sentinel array exhaustiveness, bound constants | Internal test file, mirrors finish-reason pins |
| Integration / E2E | None at Layer 1 | Recorded-stream checks reuse AI-14's checker once landed |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration required. Additive file; rollback is `git revert` before AI-20 lands (proposal's rollback plan).

## Open Questions

- [x] ~~Final spellings of `EventKindError`, `ErrorEvent`, `ErrorPayload` follow AI-14's landed kind-addition recipe~~ — **resolved**: AI-14's `design.md` landed (D6/D3, `event_descriptor.go`'s 6-step recipe). Names pinned above against the real `eventRegistry`/`eventRegistration`/`EventDescriptor` API; no longer provisional. Still re-verified at the apply gate per `S-AIP-055` in case landed code diverges from this reconciliation.
- [ ] AI-14's `StreamReport.Terminated()` (its D3) is the presence-required promotion AI-19's recorded-stream tests must assert — never by editing `CheckStream`. AI-19 consumes this shape, does not define it.

> **Deviation note**: exceeds the 800-word design budget; house precedent (AI-14/AI-16 wave-2 artifacts) carries decision tables and pinned contracts at this density.

---

> **Archive-time note, 2026-08-01 — one recorded file-placement deviation and one carried-forward gap.**
>
> 1. **Step 6 of the recipe landed in `event.go`, not `doc.go`.** The File Changes table above assigns the kind-list entry to `doc.go`; the landed recipe put it in `event.go`'s `EventKind` GoDoc instead. Benign — apply followed the landed-code precedent, which `S-AIP-055` explicitly authorises. Recorded as **D14** in the Wave 2 verify report.
> 2. **`*Failure` is the only Wave 2 payload without a `GoString()`.** D5's `Error()` rule holds and was reproduced clean for `%v`, `%s` and `%+v`, and the wrapped cause's text does **not** leak. But `%#v` falls back to reflection and reproduces two provider-supplied fields — the sanitized, bounded `rawLabel` and `requestID`. `R-AIP-009`'s literal claim survives; the wave's own four-verb rendering convention does not. Recorded as **W2** in the verify report and owned by Wave 3; the fix is one `GoString()` method plus one four-verb canary test.
