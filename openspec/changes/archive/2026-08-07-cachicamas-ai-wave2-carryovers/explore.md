# Exploration: cachicamas-ai-wave2-carryovers (AI-41)

> **Milestone:** AI-41 — Discharge the Wave-2 carryovers (Wave 5 — Harden)
> **Doc 0002 charter:** `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 2233–2257
> **Worktree:** `cachicamas-worktrees/ai-41-wave2-carryovers` · **Branch:** `feat/ai-41-wave2-carryovers` (base `origin/main` @ `f2e460d`)
> **Engram:** `sdd/cachicamas-ai-wave2-carryovers/explore` (obs #2662)
> **Date:** 2026-08-07

---

## AI-41.1 — CheckEmit rule 4 failure path

**Location correction.** `ai.CheckEmit` lives at `backend/agent/src/ai/event.go:339-368`, in package `ai` — **not** in the `agenttest` testkit. `agenttest` has zero references to `CheckEmit`. It is governed by AI-14's spec `openspec/specs/ai-event-envelope/spec.md`, which the testkit spec cites only as a dependency.

**The four rules and their coverage:**

| # | Rule | Code | Test coverage |
|---|------|------|---------------|
| 1 | kind registered | `event.go:341-346` | `event_test.go:73-88` — `TestCheckEmit_PayloadlessEvent_…` |
| 2 | sequence != 0 | `event.go:347-352` | `sequence_test.go:171-199` — `TestCheckEmit_UnstampedSentinel_…` |
| 3 | block-scoped kind has block index >= 1 | `event.go:353-363` | `event_test.go:187-210` — `TestCheckEmit_BlockScopedEventWithZeroBlockIndex_…` |
| 4 | `e.payload.validate(...)` | `event.go:364-366` | **UNTESTED — this is the carryover** |

**Why the gap exists.** Documented verbatim at `openspec/specs/ai-event-envelope/spec.md:322-324` ("Carried forward", item W1): the only payload AI-14 can construct is `export_test.go`'s `WitnessPayload`, whose `validate()` (`export_test.go:63-68`) "returns nil unconditionally… a rule is added once a test needs one to fail." Every production payload (AI-15…AI-19) validates eagerly in its constructor, so an `Event` carrying a payload that fails its own `validate()` is unconstructible through the public API. Rule 4 is reachable only via the test-only witness.

**Resolution selected.** The spec names two options — give the witness payload a controllable failure mode, or record rule 4 as deliberately unreachable defence. AI-41's test-list wording ("proven **directly**") selects the first.

**Recommended shape.** Add a `rejectWith *Violation` field to `WitnessPayload` (default `nil` = passes, preserving every existing test) plus a test-only constructor, e.g. `NewFailingWitnessEvent(block BlockIndex, reject *Violation) Event`, in `export_test.go`. The new test builds a `Stamper`-stamped, correctly block-indexed witness event carrying a non-nil `rejectWith`, asserts `CheckEmit` returns that exact violation, and asserts rules 1–3 do **not** fire — proving rule 4 specifically rather than an earlier rule short-circuiting.

**Spec surface.** `openspec/specs/ai-event-envelope/spec.md` needs a new requirement at the next free id `R-AEE-021` (R-AEE-001…020 exist) with scenarios `S-AEE-071` / `S-AEE-072` (S-AEE-001…070 exist).

---

## AI-41.2 — Redacting formatting method on the provider-failure payload

**Confirmed state.** `*Failure` (`backend/agent/src/ai/provider_failure.go`, struct at 320-330, pointer-receiver methods throughout) carries `Error()`, `Unwrap()`, `Is()`, `Category()`, `RawLabel()`, `Retryable()`, `RetryAfter()`, `StatusClass()`, `RequestID()`, `PartialOutput()`, `Delivery()`, `validate()` — but **no `String()` and no `GoString()`**.

`Error()` (337-354) is already redacted: it excludes the cause, raw label, status class and request ID. Proven by the planted-secret test at `provider_failure_test.go:1172-1240`.

**The leak.** Because `Failure` implements `error`, `fmt` dispatch already routes `%v` / `%s` / `%q` safely to `Error()`. `GoStringer` is consulted **exclusively** for `%#v`. So `%#v` is the one remaining verb, and it falls back to reflection over every unexported field — including `cause error`, which may wrap raw provider-body or credential-adjacent text.

**Sibling pattern.** 21 call sites in package `ai` follow the identical shape — `String()` plus `func (T) GoString() string { return T.String() }`:
`completion.go:106` · `event.go:319` · `content_part.go:303` · `text_events.go:111/213/281` · `tool_call_event.go:169/251/336` · `reasoning_event.go:146/264/375` · `request.go:781` · `tool_result.go:171` · `response_start.go:116` · `reasoning_content.go:271` · `tool_call.go:190` · `system_instruction.go:199/211` · `request_extension.go:51`.

**Redaction precedents in-tree.** `openaicompat/credential.go:31-43` (`Credential.String`/`GoString` both return `"<redacted>"`) and `openaicompat/openrouter/wrapper.go:90-101` (`redactedProvider.String`/`GoString` both return a fixed label — added explicitly to stop `%#v` reflecting into an embedded `*openaicompat.Client`'s unexported credential field).

**Why `Violation` needs none.** `validation.go:253-261` — its `rule error` is always a fixed sentinel and `at Path` holds positions only. No raw cause is ever stored, so it is structurally safe even under reflection. That asymmetry is precisely why AI-41.2 targets `Failure`.

**Recommended minimal fix.** `func (f *Failure) GoString() string { return f.Error() }` — delegates to the already-redacted renderer rather than reflecting, matching the sibling *pattern* without duplicating a second renderer whose redaction guarantees would have to be kept in sync.

**Open design decision (for `sdd-design`).** The charter says "a redacting alternate-verb formatting method" (singular), which supports the minimal `GoString`-only reading; but "matching the pattern its sibling payloads already carry" can be read as requiring the literal `String()` + `GoString()` pair. Resolve explicitly, do not assume.

**Spec surface.** `openspec/specs/ai-provider-errors/spec.md` (AI-19; R-AIP-001…015, S-AIP-001…055, NFR-AIP-A…E all exist) needs a new `R-AIP-016` with `S-AIP-056` / `S-AIP-057`, appended after R-AIP-015 and before the NFR section.

---

## Test strategy and conventions

- **Canonical runner:** `backend/agent/Makefile:90-92` — `make test` = `go test -race -v ./...` run from `backend/agent/`.
- **File naming:** the `a_i-NN_*_test.go` milestone-numbered convention (AI-33/34/35) is confined to `backend/agent/src/ai/openaicompat/` and its `openrouter` subpackage. The top-level `ai` package — where both AI-41 leaves live — uses **topical** file names (`event_test.go`, `provider_failure_test.go`, `sequence_test.go`, `export_test.go`). Extend those existing files; do not introduce `a_i-41_*_test.go`.
- **Adversarial redaction pattern:** already in-tree at `content_part_test.go:414-418` — loop over `%v` / `%s` / `%+v` / `%#v`, assert a planted `"CANARY"` string never appears. Reuse that exact shape for AI-41.2 alongside the existing planted-secret test.

## Spec-amendment surface (archive time)

| File | Change |
|------|--------|
| `openspec/specs/ai-event-envelope/spec.md` | New `R-AEE-021` + `S-AEE-071`/`072`; append-only note on the "Carried forward" section (322-324) recording W1 discharged |
| `openspec/specs/ai-provider-errors/spec.md` | New `R-AIP-016` + `S-AIP-056`/`057` |
| `openspec/specs/ai-stream-testkit/spec.md:39` | Already names AI-41 as owner; the charter's Acceptance line requires this to read as **closed**, so append-only amend to record both items discharged |

## Size forecast

| Kind | Estimate |
|------|----------|
| Production | ~6–8 lines (`provider_failure.go` `GoString` + doc comment) |
| Test support | ~15–20 lines (`export_test.go` `WitnessPayload` extension + constructor) |
| Tests | ~80–100 lines (rule-4 test ~30–45; `%#v` redaction test ~35–50) |
| **Total** | **~100–130 lines** — well under the 1000-line budget |

## Risks

1. Adding `GoString` to `*Failure` changes **only** `%#v` output. Grep confirms zero `%#v` / `GoString` hits in `provider_failure_test.go` today, so no regression is expected — re-verify with `make test` after the change.
2. `backend/agent/src/ai/import_boundary_test.go` guards import prefixes only, not method surfaces. Unaffected by either leaf.
3. `export_test.go`'s "MUST NOT run in parallel" discipline applies to tests holding a `RegisterTestKind` registration. The recommended per-instance `rejectWith` field introduces no shared mutable state, so the new rule-4 test can likely keep `t.Parallel()` if it uses the already-registered `KindTestWitness`. Confirm at design time.
4. AI-41.2's method shape (`GoString`-only vs. `String`+`GoString` pair) is genuinely ambiguous from the charter alone — `sdd-design` must resolve it explicitly.

## Ready for proposal

Yes. Both leaves are additive and low-risk; the two open decisions above are design-phase decisions, not blockers.
