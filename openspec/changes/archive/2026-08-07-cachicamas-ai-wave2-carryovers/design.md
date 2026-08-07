# Design: Discharge the Wave-2 carryovers (AI-41)

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 (doc 0002 lines 2233–2257)
> **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1, NOT hexagonal — `cd backend/agent && make test`)
> **Strategy**: single-pr · strict TDD on · stdlib-only · review budget 1000 changed lines

## 1. Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-wave2-carryovers` |
| **Inputs** | `proposal.md` (Flags 1–3 pre-resolved, design confirms with independent evidence); `explore.md` (Engram #2662) |
| **Scope (in)** | W1: controllable failure mode on the test witness payload + `CheckEmit` rule-4 failure-path test. W2: redacting `GoString()` on `*ai.Failure` + adversarial planted-canary `%#v` test |
| **Scope (out)** | `String()` on `*Failure` (dead code — D-1); `String`/`GoString` on `Violation` (structurally safe); new dependencies; `a_i-41_*_test.go` files; anything already proven |
| **Package** | `backend/agent/src/ai` (top level) for both leaves — NOT `agenttest`, NOT `openaicompat` |
| **Test runner** | `cd backend/agent && make test` (`go test -race -v ./...`); `make lint` |
| **Forward guard** | `backend/agent/src/ai/import_boundary_test.go` — unaffected (guards import prefixes, not method surfaces; zero `Failure` references) |
| **Layer 1 imports** | Unchanged. Both leaves add zero imports; `go.mod` untouched (ADR 0005 § D1) |

---

## 2. Technical Approach

Two independent, additive leaves inside package `ai`, each a strict-TDD red/green/refactor cycle extending existing topical files.

**AI-41.1** makes `CheckEmit`'s fourth rule (`e.payload.validate(Path{At("event")})`, `event.go:364–366`) provable by giving the test-only `WitnessPayload` a per-instance failure mode: a `rejectWith *Violation` field whose nil default preserves every existing construction path, returned verbatim by `validate`. The new rule-4 test stamps a witness event carrying a planted violation and asserts `CheckEmit` surfaces *that exact violation* — rules 1–3 are structurally satisfied (registered kind, stamped sequence, `BlockRoleNone` early-exits rule 3), so only rule 4 can be responsible.

**AI-41.2** closes the last unredacted fmt verb on `*ai.Failure`. `Error()` already covers `%v`/`%s`/`%q` (and `%x`/`%X`) because fmt dispatches to `error` for those verbs; only `%#v` consults `GoStringer`, and with none present it reflects over every unexported field. `func (f *Failure) GoString() string { return f.Error() }` delegates to the already-redacted, already-nil-safe renderer — the same delegate-don't-reflect pattern `Completion.GoString` (`completion.go:106`), `Part.GoString` (`content_part.go:303`), and `Event.GoString` (`event.go:319`) carry.

## 3. Data Flow

    fmt.Sprintf("%#v", f)            fmt.Sprintf("%v"/"%s"/"%q", f)
            │                                   │
      handleMethods: sharpV              handleMethods: error branch
            │                                   │  (before Stringer — print.go:664-676)
            ▼                                   ▼
      f.GoString() ────────────────────▶  f.Error() ──▶ "provider failure: <category>"
        (NEW, delegates)                  (existing, redacted, nil-safe)

    CheckEmit(e) ─▶ rule1 kind ─▶ rule2 seq ─▶ rule3 block (BlockRoleNone ⇒ nil)
                                                    │
                                                    ▼
                                  rule4: payload.validate(...) ─▶ rejectWith (planted)

---

## 4. Architecture Decisions

### D-1 — Method shape: `GoString()` only, no `String()` — **CONFIRMED, independently verified**

**Choice**: `func (f *Failure) GoString() string { return f.Error() }` in `provider_failure.go`, with a doc comment stating the posture (mirroring `completion.go:101–106`). No `String()`.

**Alternatives considered**:
- `String()` + `GoString()` pair (the literal sibling shape): rejected — `String()` would be unreachable dead code (evidence below).
- Implementing `fmt.Formatter`: rejected — heavyweight, no in-tree precedent, and `Formatter` would *override* the already-proven `Error()` dispatch for every verb, widening the surface that must be proven redacted.

**Rationale — the dispatch claim, verified against the actual fmt source** (`/usr/local/go/src/fmt/print.go:622–680`, Go toolchain at `/usr/local/go`):
1. `handleMethods` consults `Formatter` first (line 638); `*Failure` implements none.
2. Under `p.fmt.sharpV` (`%#v`, lines 646–653) **only** `GoStringer` is consulted. No `GoStringer` → reflective struct walk over `category`, `retryable`, `retryAfter`, `rawLabel`, `statusClass`, `requestID`, `cause`, `delivery`, `partialOutput`.
3. Otherwise, for verbs `v, s, x, X, q` (line 659), the type switch checks `case error:` (line 665) **before** `case Stringer:` (line 671). `*Failure` implements `error`, so `Error()` wins every dispatch a `String()` could ever serve — a `String()` on `*Failure` is unreachable through fmt.
4. The governing spec text is singular and names the method: `ai-stream-testkit/spec.md:39` records W2 verbatim as "the missing redacting `GoString()` on the failure payload".
5. Reversible asymmetry: adding `String()` later is additive; removing a shipped one is a public-surface removal.

The siblings carry two methods because they are **not** errors — they need `String()` for `%v` and `GoString()` for `%#v`. `Failure` already has half the pattern, spelled `Error()`. The *pattern* being matched is delegate-to-one-redacted-renderer, not a method count.

### D-2 — Witness failure mode: additive `rejectWith *Violation` — **CONFIRMED**

**Choice**: `WitnessPayload` (`export_test.go:45–48`) gains one field, `rejectWith *Violation`; `validate` (`:68`) becomes `return w.rejectWith`; one new test-only constructor `NewRejectingWitnessEvent(block BlockIndex, reject *Violation) Event` alongside `NewWitnessEvent`/`NewTestEvent`.

**Alternatives considered**:
- A new test-only payload type with an always-failing `validate`: rejected — more code, a second registry entry, and it duplicates `WitnessPayload`'s three other proofs (derivable kind, external readability, block-index readability) for one new one.
- Recording rule 4 as deliberately unreachable defence (`ai-event-envelope/spec.md:322–324` offers it): rejected — AI-41.1's charter wording "proven **directly**" forecloses it.

**Rationale**: Verified in source — `validate`'s own doc comment (`export_test.go:63–67`) literally anticipates this change: *"a rule is added once a test needs one to fail."* Both existing constructors build `WitnessPayload{k, block}` by composite literal, leaving `rejectWith` nil, so `validate` still returns nil for every current caller — zero-regression additive. The field is a pointer, so the struct stays comparable (`event_test.go:172` compares `payload != (ai.WitnessPayload{})`). `export_test.go` never reaches a non-test build (S-AEE-013/S-AEE-017) — no production surface widens.

### D-3 — Parallel safety: the rule-4 test uses `t.Parallel()` — **CONFIRMED**

**Choice**: The new rule-4 test calls `t.Parallel()`.

**Alternative considered**: Serial execution out of caution — rejected; the discipline's own scoping makes serial execution cargo-culting, and the in-tree precedent runs parallel for exactly this shape.

**Rationale — both claims verified in source**:
1. The non-parallel discipline is scoped to registrations. `export_test.go:29–30`: *"a test holding a registration MUST NOT use t.Parallel — a concurrent truncation from another test would corrupt it"*; `RegisterTestKind`'s doc (`:104–106`) repeats it. The rule-4 test registers nothing — it uses `KindTestWitness`, registered once at package-test-load time by `init()` (`:73–81`), and carries its failure mode per-instance on the payload value. No shared mutable state.
2. Structural isolation of rule 4. `KindTestWitness`'s descriptor is `Role: BlockRoleNone` (`:79`); `CheckEmit`'s third rule returns nil on its `Role == BlockRoleNone` early exit (`event.go:355–356`). With a registered kind (rule 1) and a `Stamper`-stamped sequence (rule 2), only rule 4 can fire.
3. Precedent on both sides: `TestCheckEmit_PayloadlessEvent_...` (`event_test.go:74`) calls `t.Parallel()`; `TestCheckEmit_BlockScopedEventWithZeroBlockIndex_...` (`event_test.go:185–187`) does not, and its comment cites the registration as the reason.

### D-4 — RED-canary placement: plant where today's reflection actually leaks

**Choice**: The `%#v` RED test plants canaries in **`RawLabel`** and **`RequestID`** (via `FailureReport`), and in a **value-kind cause** (a test-local `type canaryCause string` implementing `error`), then loops `%v`/`%s`/`%+v`/`%#v` asserting no canary appears (the `content_part_test.go:414–419` shape).

**Alternative considered**: Planting the canary only in an `errors.New(...)` cause (the existing `Error()` test's shape at `provider_failure_test.go:1176`) — **rejected, it would not be red**. Verified against `fmt`'s `printValue`: an unexported field blocks the `CanInterface` re-dispatch, and a nested *pointer* at depth > 0 falls through to `fmtPointer` — so `cause: (*errors.errorString)(0x...)` renders as a **hex address, not the canary text**, and the assertion would already pass today, violating strict TDD's fail-for-the-right-reason rule.

**Rationale**: Reflected `%#v` today reproduces the struct's `string` fields verbatim (`rawLabel`, `requestID` — a short clean canary survives `sanitizeOpaqueField`'s drop-whole bound) and reproduces a value-kind cause's contents. Those placements make RED genuinely fail on `%#v` while `%v`/`%s`/`%+v` already pass via `Error()` — which the test proves rather than assumes.

### D-5 — Nil-receiver behavior: total by delegation, no second nil check

**Choice**: `GoString` performs no nil check of its own; it delegates to `Error()`, which returns the fixed `noProviderFailure` label for a nil receiver (`provider_failure.go:349–354`).

**Rationale**: fmt invokes `GoString` on a typed-nil `*Failure` because the `GoStringer` type assertion succeeds for the pointer method set; the call reaches `Error()`'s existing `f == nil` branch and returns the fixed label — total, never panicking, NFR-AIP-B preserved with one nil check in one place. A second nil check in `GoString` would be a divergent copy of the same rule.

---

## 5. Redaction Contract (the `%#v` rendering)

The Go-syntax rendering of `*Failure` is exactly `Error()`'s text: the fixed prefix `"provider failure: "` plus the category's registered name — and for a nil receiver, the fixed `noProviderFailure` label.

| Field | May appear in `%#v`? | Why |
| --- | --- | --- |
| `category` (registered name only) | **Yes** | Package-owned closed vocabulary text; the only payload-derived content `Error()` renders |
| `cause` | **NEVER** | May wrap raw provider body or credential-adjacent text — the leak W2 exists to close; reachable via `Unwrap` only (R-AIP-014) |
| `rawLabel` | **NEVER** | Raw provider text; dedicated accessor `RawLabel()` (R-AIP-006) |
| `requestID` | **NEVER** | Provider-opaque; dedicated accessor (R-AIP-009 posture) |
| `statusClass`, `retryable`, `retryAfter`, `delivery`, `partialOutput` | **NEVER** | Dedicated accessors; `Error()`'s doc (`:338–348`) already excludes them — the new verb inherits the same posture unchanged |

This is byte-identical to the proven `Error()` posture (S-AIP-028…031, `provider_failure_test.go:1172–1240`); the design adds no second renderer whose redaction guarantees would need keeping in sync.

---

## 6. File Changes

| File | Action | Description |
| --- | --- | --- |
| `backend/agent/src/ai/export_test.go` | Modify | `rejectWith *Violation` field on `WitnessPayload`; `validate` returns it; `NewRejectingWitnessEvent(block, reject)`; refresh `validate`'s doc comment (its "once a test needs one to fail" day has arrived). ~15–20 lines |
| `backend/agent/src/ai/event_test.go` | Modify | `TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4` (name indicative), `t.Parallel()`, asserting the exact planted violation. ~30–45 lines |
| `backend/agent/src/ai/provider_failure.go` | Modify | `GoString()` delegating to `Error()`, doc comment stating the posture. ~6–8 lines |
| `backend/agent/src/ai/provider_failure_test.go` | Modify | Planted-canary verb loop (D-4 placements) + nil-receiver `%#v` totality/identity. ~35–50 lines |

No new files. No `a_i-41_*_test.go` — that convention is confined to `openaicompat/` and `openrouter`; the top-level `ai` package uses topical file names. `go.mod` untouched; neither leaf adds a single import.

## 7. Interfaces / Contracts

```go
// provider_failure.go — the whole production change of AI-41.2
func (f *Failure) GoString() string { return f.Error() }

// export_test.go — test-support only, never in a non-test build
type WitnessPayload struct {
    k          EventKind
    block      BlockIndex
    rejectWith *Violation // nil = passes (every existing path); non-nil = validate returns it
}
func (w WitnessPayload) validate(_ Path) *Violation { return w.rejectWith }
func NewRejectingWitnessEvent(block BlockIndex, reject *Violation) Event
```

The witness returns `rejectWith` verbatim, ignoring the position parameter — consistent with its current signature use, and it keeps the identity assertion (Q2: identity is the load-bearing claim) trivial: the surfaced violation IS the planted pointer.

## 8. Testing Strategy — strict TDD sequencing

| Leaf | RED | GREEN | REFACTOR |
| --- | --- | --- | --- |
| **AI-41.1** (S-AEE-071/072) | Add field + constructor to `export_test.go` with `validate` still returning nil unconditionally; write the rule-4 test: stamp `NewRejectingWitnessEvent` with a planted `ai.Invalid(sentinelRule, ai.At(...))`, assert `errors.Is(err, planted)` (pointer identity) and `errors.Is(err, sentinelRule)` (through `Violation.Unwrap`), and assert NOT `errors.Is` `ErrNotInVocabulary`/`ErrOutOfRange` (rules 1–3 did not fire — risk 7.3). Fails: `CheckEmit` returns nil | `validate` becomes `return w.rejectWith` — one line | Refresh `export_test.go:63–67` doc comment |
| **AI-41.2** (S-AIP-056/057) | Canary test per D-4: `FailureReport{Category, RawLabel: canary, RequestID: canary2, Cause: canaryCause(...)}`; loop `%v/%s/%+v/%#v` asserting all canaries absent; nil sub-test asserting `fmt.Sprintf("%#v", (*Failure)(nil))` equals `(*Failure)(nil).Error()` and never panics. Fails on `%#v` (string fields + value-kind cause reflected) and on nil identity | Add `GoString` delegating to `Error()` | Doc comment polish only |

Leaves are independent; execute AI-41.1 then AI-41.2 (W1/W2 order). Full gate after each leaf: `cd backend/agent && make test` (`-race`), `make lint`.

## 9. Blast Radius — verified

- **Zero** `%#v`/`GoString` occurrences in `provider_failure_test.go` today (grep count: 0) — no test asserts the current reflected output.
- No in-tree call site formats `*ai.Failure` with `%#v`: the only `openaicompat` `%#v` hits are `credential.go`, `credential_test.go`, `openrouter/wrapper.go`, `openrouter/credential_redaction_test.go`, `decision.md` — all about credential redaction, none about `Failure`. The leak is latent (proposal Q3), fixed as defence-in-depth for AI-36's consuming sweep.
- `import_boundary_test.go`: zero `Failure` references; guards import prefixes only. Unaffected.
- `mustErrorEvent` / `terminalFailureFromEvents` (the two `Failure`-referencing test helpers with fan-in) never format with `%#v`. Unaffected.
- `go.mod`: untouched — both leaves add zero imports of any kind.

## 10. Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Both leaves are in-package formatting/validation changes.

## 11. Migration / Rollout

No migration required. Both leaves are additive and independently revertible (proposal § 10). The one rollback rule that matters: a revert must re-amend the `ai-stream-testkit` carryover line back to *open* — leaving it reading *discharged* would reproduce AI-41's own failure mode.

## 12. Open Questions

None blocking. Q2 (assert the violation's position as well as identity) is resolved here as: identity is load-bearing and asserted; the planted violation carries its own position verbatim, so a position assertion is a free extra the spec phase may pin without design change.
