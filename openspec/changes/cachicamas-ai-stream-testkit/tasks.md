# Tasks — the stream test kit

> **Change**: `cachicamas-ai-stream-testkit`
> **Milestone**: AI-22 · **Nodes**: AI-22.1, AI-22.2, AI-22.3, AI-22.5 `[leaf]`; AI-22.4 `[decision]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-02
> **Worktree**: `cachicamas-worktrees/ai-wave-3` · **Branch**: `feat/2026-08-02-cachicamas-ai-layer1-wave-3`
> **Inputs**: `proposal.md`, `specs/ai-stream-testkit/spec.md` (13 requirements, 46 scenarios), `design.md` (D1–D5)
> **Depends on**: AI-21 (shipped on this branch) · **Blocks**: AI-23, AI-33, doc 0003's hardening wave
> **Evidence gate**: recorded green `make test` (`go test -race -v ./...`) **and** clean `make lint`, both from `backend/agent/` (`NFR-STK-G` / `S-STK-046`)

---

## Entry point for the resuming agent

Nothing in this milestone is implemented; every box below is unchecked. Order follows design's own D1 → D5 sequence: AI-22.2/.3/.5 each consume a `Recording` or raw channel that AI-22.1 defines, so AI-22.1 goes first. AI-22.4 (leak helper) has no hard code dependency on the others but stays in D-order. **One writer, one branch, sequential — do not run concurrent apply agents** (prior wave's prefix-collision lesson). Commit per leaf; a per-leaf commit is the rollback boundary.

Threat matrix: **N/A** per `design.md` — no routing, shell, subprocess, VCS/PR automation, or process-integration boundary in this package.

## Node types and what they close on

| Node | Type | Closes on |
| --- | --- | --- |
| AI-22.1, .2, .3, .5 | `[leaf]` | Every test-list item taken red → green → refactored, in order, with both outputs recorded here |
| AI-22.4 | `[decision]` | Same red→green→refactor discipline, plus the recorded rejection of a third-party leak detector (`R-STK-009`) — the rejection record *is* the deliverable, not a separate ADR |

Strict TDD is on. A RED that passes immediately because earlier work already covers it is legitimate and must be recorded honestly, never forced.

---

## Review Workload Forecast

| Field | Value |
| --- | --- |
| Estimated changed lines | Production ~300–450; tests ~650–950; `doc.go` ~10–15; openspec (existing proposal/spec/design ~480 + this `tasks.md` ~300–450) ~780–930 → **total ~1 700–2 400** |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR — the wave PR (AI-21 + AI-22 + AI-23), leaf commits as the reviewable boundary |
| Delivery strategy | single-pr |
| Chain strategy | size-exception |

```text
Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

### Running-total flag for the orchestrator — read before committing to AI-23

AI-21 verified actual: **2 241** code + **569** openspec = **2 810** of the wave's shared 5 000-line ceiling. This forecast's own estimate for AI-22 is **~1 700–2 400**, already well above the proposal's original ~700–1 100 guess — the same test/doc density overrun AI-11 recorded on this branch. **Running total AI-21 actual + AI-22 forecast = ~4 510–5 210.** The upper end of that range **meets or exceeds the 5 000 ceiling before AI-23 (9 leaves, per the proposal's own risk table) is even planned.** Recommend the orchestrator re-check the actual AI-22 diff against this forecast immediately after apply, and decide AI-23's PR boundary (same wave PR vs. its own size-exception PR) **before** running `sdd-tasks`/`sdd-apply` for AI-23 — do not assume the proposal's "ships with AI-21 and AI-23 in one PR" framing still holds.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
| --- | --- | --- | --- | --- | --- |
| 1 — AI-22.1 | A stream drains to a reusable, order-preserving `Recording`, deadline-safe | wave PR, commit 1 | `go test -race -run 'TestDrainAndRecord\|TestRecording' ./src/agenttest/` | N/A — stdlib-only helper, no runnable process; full-package harness is unit 5 (`make test`) | `git revert` commit 1: removes `stream_kit_record.go` + its test |
| 2 — AI-22.2 | A diff between two recordings names the first divergence, bounded and exhaustive | wave PR, commit 2 | `go test -race -run 'TestRequireSameEvents\|TestSummary' ./src/agenttest/` | N/A — same reason | `git revert` commit 2: removes `stream_kit_diff.go` + its test |
| 3 — AI-22.3 | Ordering delegates to `ai.CheckStream`; contiguity is new and precise | wave PR, commit 3 | `go test -race -run 'TestRequireValidStream\|TestCheckContiguity' ./src/agenttest/` | N/A — same reason | `git revert` commit 3: removes `stream_kit_ordering.go` + its test |
| 4 — AI-22.4 | A leaking scenario is caught, opt-in, serial-only, third-party detector recorded as rejected | wave PR, commit 4 | `go test -race -run 'TestRequireNoGoroutineLeak' ./src/agenttest/` | N/A — in-process `runtime.NumGoroutine()` accounting, no external process; serial-only by contract | `git revert` commit 4: removes `stream_kit_leak.go` + its test |
| 5 — AI-22.5 | A carrier view iterates without owning/closing the stream; `doc.go` names the kit | wave PR, commit 5 | `go test -race -run 'TestIter\|TestSignatureGuard' ./src/agenttest/` | `make test` in `backend/agent/` — the whole-module run proves AI-20.4's guard and both AI-00 import guards still hold | `git revert` commit 5: removes `stream_kit_iter.go`, its test, and the `doc.go` edit |

---

## Phase AI-22.1 — timeout-safe drain and record `[leaf]`

**Deliverable:** `stream_kit_record.go`, `stream_kit_record_test.go`.
**Spec:** `R-STK-001`, `R-STK-002`. **Design:** D1.
**Depends on:** AI-21's fake provider (producer under test).

- [x] **Item 1** (`R-STK-001`) — Draining is bounded by a deadline and never hangs.
  - [x] 1.1 RED: test a producer that emits two events and never closes; drain with a short deadline; confirm the specific expected failure (`tb.Fatalf` naming the deadline and the two events received). `S-STK-001`.
  - [x] 1.2 RED: test a fully scripted stream that closes normally with a generous deadline; confirm expected failure is `undefined: DrainAndRecord` (function does not exist yet). `S-STK-002`.
  - [x] 1.3 RED: test a stream that closes bare after cancellation with events undelivered; confirm same expected failure as 1.2. `S-STK-003`.
    - RED output (1.1–1.3, written and run together in `stream_kit_record_test.go`): `go test -race -run 'TestDrainAndRecord' -v ./src/agenttest/...` → `src/agenttest/stream_kit_record_test.go:74:19: undefined: agenttest.DrainAndRecord` (+ two more `undefined: agenttest.DrainAndRecord`/`undefined: agenttest.DefaultDrainTimeout` sites) → `FAIL github.com/cachicamas/backend/agent/src/agenttest [build failed]`. Honestly noted: 1.1's own RED is this same shared compile failure, not yet a runtime `tb.Fatalf` — the runtime deadline-naming assertion 1.1 describes is what its test *checks* once GREEN, exercised for the first time by 1.4 below, matching 1.2/1.3's own stated pattern.
  - [x] 1.4 GREEN: implement `DrainAndRecord(tb testing.TB, ch <-chan ai.Event, timeout time.Duration) Recording` and `DefaultDrainTimeout` per design D1 (select loop, `tb.Fatalf` on deadline, never hang). Confirm 1.1–1.3 pass. GREEN output: `go test -race -run 'TestDrainAndRecord' -v ./src/agenttest/...` → `--- PASS: TestDrainAndRecord_ClosesBareWithEventsUndelivered_ReportsNormalClosureNotDeadlineFailure`, `--- PASS: TestDrainAndRecord_FullyScriptedStream_ReturnsCompleteRecordingWithoutReachingDeadline`, `--- PASS: TestDrainAndRecord_NeverCloses_FailsNamingDeadlineAndEventsReceived`, `PASS`, `ok github.com/cachicamas/backend/agent/src/agenttest 1.350s`. Implemented `Recording`/`Events()`/`Len()` together with `DrainAndRecord` in this same GREEN step (one D1 decision, one file; `agenttest_test` is deliberately external, so item 1's own tests structurally need `rec.Len()` to verify anything beyond "did it fail" — see item 2's honest pass-immediately note below). Deliberately did NOT add nil-channel/zero-timeout guards yet — held for item 3's own genuine RED/GREEN.
  - [x] 1.5 REFACTOR: note whether the select loop matches `requireClosedWithin`/`drainFake`'s shape closely enough to keep as one code path, or record why it diverges. Note: identical shape (select on receive-with-ok vs. `time.After` deadline, `tb.Fatalf` naming the deadline and count on expiry) — the only divergence is returning a `Recording` value instead of a bare `[]ai.Event`, which is R-STK-002's own reason to exist. No further extraction needed; `go vet ./...` clean.

- [x] **Item 2** (`R-STK-002`) — One recording backs many assertions without re-draining.
  - [x] 2.1 RED: test one recording of a four-event stream consumed by three different kit assertions in turn; confirm the specific expected failure. `S-STK-004`.
  - [x] 2.2 RED: test reading a recording twice, mutating the first read's slice, confirm the second read is unaffected; confirm expected failure. `S-STK-005`.
  - [x] 2.3 RED: test a recording of a scripted stream compared against the script for exact order, no add/reorder/coalesce/omit; confirm expected failure. `S-STK-006`.
    - RED output (2.1–2.3): `go test -race -run 'TestRecording_' -v ./src/agenttest/...` → all three **PASS immediately**: `--- PASS: TestRecording_Events_MutatingFirstReadDoesNotAffectSecondRead`, `--- PASS: TestRecording_MatchesScriptedOrderExactly_NoAddReorderCoalesceOmit`, `--- PASS: TestRecording_OneRecordingBacksThreeAssertionsInTurnWithoutReDraining` (3 subtests). Honestly recorded, not forced: `Events()`'s fresh-copy semantics were already implemented as part of item 1's GREEN (1.4), since `agenttest_test`'s external-package posture made `Recording`/`Events()`/`Len()` structurally inseparable from `DrainAndRecord`'s own testability — no gap found.
  - [x] 2.4 GREEN: implement `Recording` and `(Recording).Events() []ai.Event` returning a fresh copy per design D1. Confirm 2.1–2.3 pass. GREEN output: no extension needed — same PASS output as the RED run above (2.1's/2.2's/2.3's own assertions, run again as confirmation, unchanged).
  - [x] 2.5 REFACTOR: confirm `Events()` is the single copy point — no other exported method exposes the backing slice. Confirmed: `Recording`'s only exported members are `Events()` (returns `slices.Clone(r.events)`) and `Len()` (returns `int`, no slice exposure); the unexported `events` field is unreachable from outside the package.

- [x] **Item 3** *(appended, `NFR-STK-E`)* — Extreme inputs never panic.
  - [x] 3.1 RED: table test `DrainAndRecord` against a nil channel and a zero timeout; confirm the specific expected failure (attributable `tb.Fatalf`, not a panic). `S-STK-044` (partial). RED output (nil channel, zero timeout, negative timeout, table-driven): `go test -race -run 'TestDrainAndRecord_ExtremeInputs' -v ./src/agenttest/...` → genuine failures, no panic: `failure message "agenttest: DrainAndRecord did not close within -1s (0 event(s) received so far) — want a bounded close (R-STK-001)" does not specifically name a non-positive timeout`; same shape for `zero_timeout` (`"...within 0s..."`) and `nil_channel` (`"...within 30ms..."` — no mention of "nil"). `--- FAIL: TestDrainAndRecord_ExtremeInputs_FailAttributablyNeverPanic` (3 subtests failed on message-content assertions only — the select loop's emergent safety already meant neither input panicked or hung, exactly NFR-STK-E's non-negotiable half; the RED is specifically about attributability, per 3.1's own wording).
  - [x] 3.2 GREEN: guard both cases explicitly in `DrainAndRecord`. Confirm 3.1 passes. GREEN output: added explicit `if ch == nil` / `if timeout <= 0` guards, each with a message naming the specific extreme input, ahead of the select loop → `go test -race -run 'TestDrainAndRecord|TestRecording' -v ./src/agenttest/...` → all subtests `--- PASS`, including `TestDrainAndRecord_ExtremeInputs_FailAttributablyNeverPanic` (nil_channel, zero_timeout, negative_timeout), `PASS`, `ok github.com/cachicamas/backend/agent/src/agenttest 1.354s`.

- [x] **AI-22.1 close:** record green `make test` and clean `make lint`; confirm `DrainAndRecord`'s and `Recording`'s doc comments each cite `R-STK-001`/`R-STK-002` (`NFR-STK-D`); commit `feat(agenttest): timeout-safe drain and reusable recording (AI-22.1)`.
  - `make test`: `ok github.com/cachicamas/backend/agent/src/agenttest 1.461s`, `ok github.com/cachicamas/backend/agent/src/ai 3.282s`.
  - `make lint`: first run caught `revive: package-comments` on `stream_kit_record.go` (milestone-header comment attached directly to `package agenttest` with no blank line — the exact AI-21 9.5 gotcha, recorded there too) → fixed by inserting the blank line → `0 issues.`
  - Doc comments: `DrainAndRecord`'s cites `(R-STK-001, R-STK-002)`; `Recording`'s cites `(R-STK-002)`; `Events()`'s cites `(R-STK-002)`. Confirmed.
  - Commit: `feat(agenttest): timeout-safe drain and reusable recording (AI-22.1)`.

---

## Phase AI-22.2 — readable event diffs `[leaf]`

**Deliverable:** `stream_kit_diff.go`, `stream_kit_diff_test.go`.
**Spec:** `R-STK-003`, `R-STK-004`. **Design:** D2.
**Depends on:** AI-22.1 (`Recording`/`[]ai.Event`).

- [x] **Item 1** (`R-STK-003`) — A difference is reported as the first divergence, located precisely.
  - [x] 1.1 RED: two four-event recordings differing at index 2; confirm the failure names index 2 and both sides' kind/sequence, not index 3. `S-STK-007`.
  - [x] 1.2 RED: two element-wise-equal recordings; confirm no failure is produced. `S-STK-008`.
  - [x] 1.3 RED: a three-event and a five-event recording sharing their first three; confirm the report names index 3 and the two extra events' kinds. `S-STK-009`.
    - RED output (1.1–1.3): `go test -race -run 'TestRequireSameEvents' -v ./src/agenttest/...` → `src/agenttest/stream_kit_diff_test.go:112:12: undefined: agenttest.RequireSameEvents` (× 3 call sites) → `FAIL [build failed]`.
  - [x] 1.4 GREEN: implement `RequireSameEvents(tb testing.TB, got, want []ai.Event)` per design D2 (`reflect.DeepEqual` per index, shorter-length reporting). Confirm 1.1–1.3 pass. GREEN output: implemented using `ev.String()` (kind+seq, deliberately payload-free — R-STK-004's summary table is item 2's own, separable addition since R-STK-003 only requires naming index/kind/sequence, not payload) → `go test -race -run 'TestRequireSameEvents' -v ./src/agenttest/...` → `--- PASS: TestRequireSameEvents_DifferAtIndex2_NamesIndexAndBothSidesKindAndSequence`, `--- PASS: TestRequireSameEvents_ElementWiseEqual_NoFailure`, `--- PASS: TestRequireSameEvents_LengthMismatch_NamesShorterEndAndExtraKinds`, `PASS`.
  - [x] 1.5 REFACTOR: confirm the length-mismatch path and the value-mismatch path share one reporting function, not two near-duplicates. Extracted `reportDivergence(tb, index, got, want string)` as the one `tb.Fatalf` call site; both paths now format their two description strings and funnel through it. Re-ran → still `PASS`.

- [x] **Item 2** (`R-STK-004`) — Payload rendering is bounded; no kind renders as nothing.
  - [x] 2.1 RED: an event whose payload greatly exceeds the cap; confirm the rendering is at most `summaryRuneCap = 32` runes with an elision marker. `S-STK-010`.
  - [x] 2.2 RED: one event of every registered kind; confirm each produces a non-empty, kind-naming rendering. `S-STK-011`.
  - [x] 2.3 RED: a scratch kind added to the registry without a summary entry; confirm the exhaustiveness test fails and names the unsummarised kind. `S-STK-012`.
    - RED output (2.1/2.2, automated — genuine failures against item 1's payload-free `.String()` rendering): `go test -race -run 'TestRequireSameEvents_PayloadFarExceedsCap|TestRequireSameEvents_EveryRegisteredKind|TestRequireSameEvents_UnregisteredKind' -v ./src/agenttest/...` → cap test: `does not carry the elision marker...`, `...does not carry the payload's first 32 runes...`, `...does not report the payload's true length (200)`; all 12 per-kind subtests: `renders <kind> using the generic event(kind seq=N) fallback..., want a kind-specific, payload-aware summary` → `--- FAIL` (13 failures total). `TestRequireSameEvents_UnregisteredKind_RendersDistinctlyNotBlank` **passed immediately** (honest: `ai.Event.String()`'s existing "unset" naming for the zero kind already satisfies non-blank distinctness — no gap).
  - [x] 2.4 GREEN: implement the `map[ai.EventKind]func(ai.Event) string` summary table (12 accessors) and the exhaustiveness test asserting table keys ≡ `ai.EventKinds()`, per design D2. Confirm 2.1–2.3 pass; revert the scratch kind after confirming 2.3 bites. GREEN output: implemented the 12-entry table + `boundedFragment`/`summarize` → all 2.1/2.2 tests `PASS`, but this **regressed item 1's own S-STK-007 test** (`DifferAtIndex2...`: message stopped naming `seq=3` — the new per-kind renderers don't include sequence, only `RequireSameEvents`'s per-index loop did via the OLD `.String()` call). Fixed by having the per-index divergence report append `(seq=%d)` explicitly alongside `summarize(...)`, restoring R-STK-003's sequence-naming while keeping R-STK-004's bounded summary — re-ran the full `TestRequireSameEvents` group → all 6 tests `PASS`. **Exhaustiveness proof (S-STK-012, task-instructed manual verification)**: temporarily deleted the `tool_call_end` table entry; discovered the FIRST exhaustiveness-check design (substring `"event(%s seq=%d)"`) no longer detected the gap, because the sequence-duplication fix above had already changed the fallback's shape — a second, genuine bug caught by re-running. Strengthened the test's detection to `"<kind> (seq="` (bare kind name immediately followed by the appended sequence, vs. a real summary's `<kind>(<payload fields>)`) → re-ran with the entry still removed → **`--- FAIL`**, naming `tool_call_end` exactly: `renders tool_call_end as the bare, payload-free fallback "tool_call_end (seq=", want a kind-specific, payload-aware summary (exhaustiveness, S-STK-012)`. Restored the entry → re-ran full `TestRequireSameEvents` group → all `PASS`.
  - [x] 2.5 REFACTOR: confirm structural fields render whole and free-form bytes render as `len=N head="…"` — no code path bypasses the cap. Confirmed by reading `summaryTable`: `responsestart`/`toolcallstart`'s ids and name, `text_block_start`/`text_block_end`'s block index, `reasoningblockstart`'s redacted flag, `completion`'s finish reason and usage all render whole via `%v`/`%q`/`%d`; every free-form byte field (`reasoningdelta`'s fragment, `reasoningblockend`'s token, `text_delta`'s delta, `tool_call_delta`'s fragment, `tool_call_end`'s arguments, `error`'s raw label) routes through the one `boundedFragment` helper — no second cap implementation exists.

- [x] **Item 3** *(appended, `NFR-STK-E`)* — Extreme inputs never panic.
  - [x] 3.1 RED: `RequireSameEvents` against two empty recordings and against one-element recordings; confirm no panic, attributable failure only where they actually differ. `S-STK-044` (partial). RED/GREEN output: `go test -race -run 'TestRequireSameEvents_ExtremeInputs' -v ./src/agenttest/...` → **all 4 subtests PASS immediately** (both-empty/no-failure, empty-vs-one-element/fails-naming-index-0, one-element-equal/no-failure, one-element-differing/fails-naming-index-0) — honestly recorded, no forced failure: the `min(len(got), len(want))`-bounded loop and the length-mismatch branch already handle 0/1-length slices safely by construction (Go's `for range 0` is a no-op, slicing `events[0:]` on an empty/one-element slice never panics).
  - [x] 3.2 GREEN: confirm the length-mismatch path already handles zero/one length without a nil-slice panic; add a guard only if 3.1 shows one is needed. Confirmed: no guard needed, matches 3.1's finding.

- [x] **AI-22.2 close:** record green `make test` and clean `make lint`; confirm both exported functions' doc comments cite `R-STK-003`/`R-STK-004`; commit `feat(agenttest): readable, bounded event diffs (AI-22.2)`.
  - `make test`: `ok github.com/cachicamas/backend/agent/src/agenttest 1.445s`, `ok github.com/cachicamas/backend/agent/src/ai (cached)`.
  - `make lint`: `0 issues.` (clean on first run this time — the package-comment blank-line convention from AI-22.1's fix was already followed).
  - Doc comments: `RequireSameEvents`'s cites `(R-STK-003)` and `(R-STK-004)` both, in its own doc comment. Confirmed.
  - Commit: `feat(agenttest): readable, bounded event diffs (AI-22.2)`.

---

## Phase AI-22.3 — ordering and gap assertions `[leaf]`

**Deliverable:** `stream_kit_ordering.go`, `stream_kit_ordering_test.go`.
**Spec:** `R-STK-005`, `R-STK-006`. **Design:** D3 (honors design D10 from AI-21).
**Depends on:** AI-22.1 (`Recording`); `ai.CheckStream` (AI-14.4, read-only).

- [x] **Item 1** (`R-STK-005`) — Ordering is delegated to `ai.CheckStream`, never reimplemented.
  - [x] 1.1 RED: a recording carrying a terminal event followed by a text delta; confirm the failure carries `ai.CheckStream`'s own verdict, unmodified. `S-STK-013`.
  - [x] 1.2 RED: a well-formed recording; confirm the assertion passes and reports no violation. `S-STK-014`.
    - RED output (1.1–1.2): `go test -race -run 'TestRequireValidStream' -v ./src/agenttest/...` → `src/agenttest/stream_kit_ordering_test.go:58:12: undefined: agenttest.RequireValidStream` (× 2 call sites) → `FAIL [build failed]`.
  - [x] 1.3 GREEN: implement `RequireValidStream(tb testing.TB, rec Recording)` calling `ai.CheckStream` first, per design D3. Confirm 1.1–1.2 pass. GREEN output: implemented calling only `ai.CheckStream` (not yet `CheckContiguity`, which doesn't exist until item 2 — genuinely separable since 1.2's fixture is also contiguous) → `go test -race -run 'TestRequireValidStream' -v ./src/agenttest/...` → `--- PASS: TestRequireValidStream_TerminalFollowedByDelta_CarriesCheckStreamsOwnVerdictUnmodified`, `--- PASS: TestRequireValidStream_WellFormedRecording_Passes`, `PASS`. S-STK-013's own assertion computes the expected message by calling `ai.CheckStream` directly in the test and checking the Fatalf output contains that exact `.Violation().Error()` string — not a hardcoded golden — so it stays correct across a future descriptor change, per design's own stated reason for delegating.
  - [x] 1.4 Inspection *(not an automated RED/GREEN, `S-STK-015`)*: confirm at leaf close that no kind/block/terminal ordering rule is reimplemented in this file — grep for any local re-derivation of `ai.CheckStream`'s logic; record the confirmation here. Confirmed: `grep -n "BlockRole\|Terminal\|Cardinality\|checkBlockOrdering" src/agenttest/stream_kit_ordering.go` → no matches. `RequireValidStream` only calls `ai.CheckStream`; it never inspects `BlockRole`/`Cardinality`/terminal flags itself.

- [x] **Item 2** (`R-STK-006`) — Sequence contiguity is asserted; a gap is named precisely.
  - [x] 2.1 RED: a recording sequenced 1, 2, 3, 4; confirm the specific expected failure (`undefined: CheckContiguity`). `S-STK-016`.
  - [x] 2.2 RED: a recording sequenced 1, 2, 4; confirm the failure names missing sequence 3 and the two neighbouring events (indices + sequence values 2 and 4). `S-STK-017`.
  - [x] 2.3 RED: a recording whose first event carries sequence 2; confirm the failure names the start-at-1 violation, not a mid-stream gap. `S-STK-018`.
  - [x] 2.4 RED: a recording sequenced 1, 2, 2; confirm the failure names the repeated sequence and its index. `S-STK-019`.
    - RED output (2.1–2.4, written and run together): `go test -race -run 'TestCheckContiguity' -v ./src/agenttest/...` → `src/agenttest/stream_kit_ordering_test.go:123:22: undefined: agenttest.CheckContiguity` (× 4 call sites) → `FAIL [build failed]`. Fixture note: since `CheckContiguity` reads only `.Sequence()`, all four tests reuse one shared `stampSequences(t, seqs []int)` helper — a fresh, independent `ai.Stamper` per requested value (called N times, keeping only the last result) — since a single shared `Stamper` can only ever count upward and cannot produce the deliberately-invalid gap/repeat/decrease sequences these scenarios need.
  - [x] 2.5 GREEN: implement `CheckContiguity(events []ai.Event) error` per design D3 (walk encounter order, assert seq 1 then `prev+1`). Confirm 2.1–2.4 pass; confirm `RequireValidStream` runs `CheckContiguity` after `ai.CheckStream`. GREEN output: `go test -race -run 'TestCheckContiguity|TestRequireValidStream' -v ./src/agenttest/...` → all 6 tests `--- PASS` (no regression on item 1's `WellFormedRecording` test this time — its fixture's sequences 1,2,3 are contiguous by design, planned ahead after AI-22.2's regression lesson), `PASS`, `ok github.com/cachicamas/backend/agent/src/agenttest 1.335s`. `RequireValidStream` now calls `ai.CheckStream` then `CheckContiguity(events)`, each producing its own `tb.Fatalf` citing its own requirement (`R-STK-005`/`R-STK-006`).
  - [x] 2.6 REFACTOR: confirm the gap/start/repeat failure messages share one formatting helper. Extracted `contiguityErrorf(format string, args ...any) error` (prefix + `(R-STK-006)` suffix) up front, alongside the GREEN implementation — all three violation branches (start-at-1, repeat/decrease, gap) call it; no near-duplicate formatting exists. Re-ran → still `PASS`.

- [x] **Item 3** *(appended, `NFR-STK-E`)* — Extreme inputs never panic.
  - [x] 3.1 RED: `CheckContiguity` and `RequireValidStream` against an empty recording and a one-element recording; confirm no panic and an attributable result (empty = valid; single element must start at 1). `S-STK-044` (partial). RED/GREEN output: `go test -race -run 'TestCheckContiguityAndRequireValidStream_ExtremeInputs' -v ./src/agenttest/...` → **all 4 subtests PASS immediately** (`CheckContiguity` empty/nil valid, single-element seq=1 valid, single-element seq=5 fails attributably naming `event[0]`, `RequireValidStream` empty recording valid) — honestly recorded: the range loop over `events` is a no-op for length 0, and the `i==0` branch never accesses `events[i-1]`, so both were already panic-safe by construction.
  - [x] 3.2 GREEN: guard the empty/len-1 cases explicitly. Confirm 3.1 passes. Confirmed: no guard needed, matches 3.1's finding.

- [x] **AI-22.3 close:** record green `make test` and clean `make lint`; confirm both exported functions cite `R-STK-005`/`R-STK-006`; commit `feat(agenttest): delegate ordering, assert new contiguity (AI-22.3)`.
  - `make test`: `ok github.com/cachicamas/backend/agent/src/agenttest 1.427s`, `ok github.com/cachicamas/backend/agent/src/ai (cached)`.
  - `make lint`: `0 issues.`
  - Doc comments: `RequireValidStream` cites both `(R-STK-005)`/`(R-STK-006)`; `CheckContiguity` cites `(R-STK-006)`. Confirmed.
  - Commit: `feat(agenttest): delegate ordering, assert new contiguity (AI-22.3)`.

---

## Phase AI-22.4 — leak-detection mechanism `[decision]`

**Deliverable:** `stream_kit_leak.go` (header records the rejection + ADR trigger), `stream_kit_leak_test.go`.
**Spec:** `R-STK-007` … `R-STK-010`. **Design:** D4.
**Depends on:** none of AI-22.1–.3 at the code level; ordered here per D-sequence.

- [x] **Item 1** (`R-STK-007`) — Opt-in, amplitude-based, never implicit.
  - [x] 1.1 RED: a scenario that leaks one goroutine per call, wrapped at `leakRepeats = 50`; confirm the failure names observed growth against the repeat count. `S-STK-020`.
  - [x] 1.2 RED: a scenario that leaks nothing, wrapped at the same repeat count; confirm it passes within tolerance. `S-STK-021`.
    - RED output (1.1–1.2, written together with items 4 and 5's tests since all target the same not-yet-existing function): `go test -race -run 'TestRequireNoGoroutineLeak' -v ./src/agenttest/...` → `src/agenttest/stream_kit_leak_test.go:30:12: undefined: agenttest.RequireNoGoroutineLeak` (× 5 call sites across all of this file's tests) → `FAIL [build failed]`. Fixture note: the leaking scenario spawns a goroutine blocked on `<-done`, where `done` is closed via `t.Cleanup` — deterministic (no timing dependency) and self-releasing, so the deliberately-leaked goroutines never pollute a later test's own count.
  - [x] 1.3 GREEN: implement `RequireNoGoroutineLeak(tb testing.TB, scenario func())` per design D4 (50 repeats, 50ms settle, `after <= before + leakRepeats/2`). Confirm 1.1–1.2 pass. GREEN output: `go test -race -run 'TestRequireNoGoroutineLeak' -v ./src/agenttest/...` → all 5 tests `--- PASS`, `PASS`, `ok github.com/cachicamas/backend/agent/src/agenttest 1.821s`. Stress-checked for flakiness (goroutine counting is inherently timing-sensitive): 5× repeated `go test -race -count=1 ./src/agenttest/...` full-package runs → all 5 `ok`, no flakes.
  - [x] 1.4 Inspection *(`S-STK-022`)*: confirm an existing stream test that does not call the helper is unchanged in behaviour, runtime and outcome — regression run only, no new test. Confirmed via this leaf's own `go test -race ./...` full-package run (below): every pre-existing test in `src/agenttest` and `src/ai` still passes, none reference `RequireNoGoroutineLeak`.

- [x] **Item 2** (`R-STK-008`, non-negotiable) — Serial-only, and says so on its own surface.
  - [x] 2.1 GREEN: implement the `tb.Setenv` sentinel pin (mechanical serial-only enforcement) per design D4; write the GoDoc stating the `t.Parallel()` incompatibility and the process-wide-count reason. `S-STK-023`. Done: `RequireNoGoroutineLeak` calls `tb.Setenv(leakSerialOnlySentinel, "1")` before anything else; its doc comment's "Serial-only, non-negotiably (R-STK-008)" section states both the incompatibility and the process-wide-count reason in full sentences.
  - [x] 2.2 Inspection (`S-STK-024`): confirm every test this milestone adds that uses the helper does not call `t.Parallel()`. Confirmed: `grep -n "t.Parallel()" src/agenttest/stream_kit_leak_test.go` → zero matches (the two hits are inside comments, not code) — every one of this file's 5 tests, including subtests/closures, is non-parallel.
  - [x] 2.3 Confirm via 1.1's math (`S-STK-025`): `leakRepeats/2 = 25` tolerance still fails a scenario leaking 1/iteration at 50 repeats — record the check, no new test needed. Confirmed empirically by 1.3's own GREEN run: the leaking scenario (1 goroutine/call × 50 calls ⇒ ~50 growth) correctly exceeded the 25-tolerance band and failed (`TestRequireNoGoroutineLeak_LeakingScenario_FailsNamingObservedGrowth` — PASS means the helper correctly failed against it).

- [x] **Item 3** (`R-STK-009`, decision deliverable) — The mechanism decision is recorded, including the rejected alternative.
  - [x] 3.1 Write the file-header decision record in `stream_kit_leak.go`: `go.uber.org/goleak` rejected for this change (needs its own ADR per `AGENTS.md` rule 5; breaks `doc.go`'s dependency-free pin), rejection scoped and reversible. `S-STK-027`. Done — cites `openspec/AGENTS.md` rule 5 verbatim ("New top-level dependency ⇒ ADR first. Document rationale, alternatives, rollback.") and states the rejection is scoped/reversible.
  - [x] 3.2 Inspection at close (`S-STK-026`): confirm `go.mod` is unchanged and declares no new require. Confirmed: `git diff --stat -- backend/agent/go.mod` → empty; `go.mod` still reads `module .../agent` / `go 1.26.3`, no `require` block.

- [x] **Item 4** (`R-STK-010`) — Leak assertions cover cancellation and abandoned-then-cancelled only.
  - [x] 4.1 RED: a stream cancelled mid-consumption, repeated under the helper; confirm no growth beyond tolerance. `S-STK-028`.
  - [x] 4.2 RED: a consumer that stops reading, then the caller cancels, repeated under the helper; confirm no growth beyond tolerance. `S-STK-029`. (RED output shared with 1.1/1.2 above — same compile failure, same file.)
  - [x] 4.3 GREEN: confirm 4.1–4.2 pass against AI-21's fake provider scripted for both paths; no new production code beyond Item 1's helper. Confirmed: both `TestRequireNoGoroutineLeak_CancelledMidConsumption_NoGrowthBeyondTolerance` and `TestRequireNoGoroutineLeak_AbandonedThenCancelled_NoGrowthBeyondTolerance` `--- PASS`, sharing one `abandonedCancelScenario(t, drainAfterCancel bool)` helper against `agenttest.NewProvider` — no code added to `stream_kit_leak.go` beyond item 1's GREEN.
  - [x] 4.4 Doc-check (`S-STK-030`): confirm the abandoned-never-cancelled path is stated out of scope in `stream_kit_leak.go`'s header, citing `ai-stream-lifecycle` § 5's untestability reason. Added a "Coverage is narrowed to two paths, deliberately (R-STK-010)" header section, quoting § 5 directly ("no test proves a goroutine never exits...", "the abandoned-then-cancelled path is testable and is where the leak assertions go; the abandoned-never-cancelled path is the documented violation and gets no test pretending otherwise") — read from `openspec/specs/ai-stream-lifecycle/spec.md` § 5 directly, not paraphrased from memory.

- [x] **Item 5** *(appended, `NFR-STK-E`)* — Extreme inputs never panic.
  - [x] 5.1 RED: `RequireNoGoroutineLeak` given a scenario that is a true no-op closure (zero work); confirm it passes without panic. `S-STK-044` (partial). (RED output shared with 1.1/1.2/4.1/4.2 above.)
  - [x] 5.2 GREEN: confirm no special-casing is needed; record the result. Confirmed: `TestRequireNoGoroutineLeak_NoOpScenario_PassesWithoutPanic` `--- PASS` with zero special-casing — `for range leakRepeats { scenario() }` against an empty closure is simply 50 no-op calls.

- [x] **AI-22.4 close:** record green `make test` and clean `make lint`; confirm the helper's doc comment cites `R-STK-007`; commit `feat(agenttest): opt-in serial-only leak detection, third-party detector rejected (AI-22.4)`.
  - `make test`: `ok github.com/cachicamas/backend/agent/src/agenttest 1.926s`, `ok github.com/cachicamas/backend/agent/src/ai (cached)`.
  - `make lint`: `0 issues.`
  - Doc comment: `RequireNoGoroutineLeak`'s first paragraph cites `(R-STK-007)`. Confirmed.
  - Commit: `feat(agenttest): opt-in serial-only leak detection, third-party detector rejected (AI-22.4)`.

---

## Phase AI-22.5 — carrier view `[leaf]`

**Deliverable:** `stream_kit_iter.go`, `stream_kit_iter_test.go`; modify `doc.go`.
**Spec:** `R-STK-011` … `R-STK-013`. **Design:** D5.
**Depends on:** none of AI-22.1–.4 at the code level; ordered here per D-sequence.

- [x] **Item 1** (`R-STK-011`) — The view never owns and never closes the stream.
  - [x] 1.1 RED: a stream and a view over it; iterate to completion; confirm the view performed no close — only the producer's own close occurred. `S-STK-031`.
  - [x] 1.2 RED: a view abandoned before the stream ends, then the caller cancels; confirm the stream terminates exactly as without a view interposed. `S-STK-032`.
    - RED output (1.1–1.2): `go test -race -run 'TestIter_' -v ./src/agenttest/...` → `src/agenttest/stream_kit_iter_test.go:29:20: undefined: agenttest.NewIter` (× 2 call sites) → `FAIL [build failed]`.
  - [x] 1.3 GREEN: implement `NewIter(ch <-chan ai.Event) *Iter` per design D5. Confirm 1.1–1.2 pass. GREEN output: implemented `NewIter` + a minimal `Events(ctx)` (plain `for ev := range it.ch` — no `ctx`-select, no `Err()` yet; genuinely separable since S-STK-031/032's own scenarios cancel the STREAM's context directly, never the view's `ctx` parameter) → `go test -race -run 'TestIter_' -v ./src/agenttest/...` → `--- PASS: TestIter_IterateToCompletion_ProducersOwnCloseIsWhatEndsIt`, `--- PASS: TestIter_AbandonedThenCancelled_StreamTerminatesExactlyAsWithoutAView`, `PASS`. S-STK-031's own proof is partly structural: `Iter.ch` is typed `<-chan ai.Event` (receive-only), so `close(it.ch)` inside the view would not even compile — the test only needs to confirm the CHANNEL correctly reports closed after the view's loop ends.

- [x] **Item 2** (`R-STK-012`) — Terminal error surfaced after the loop; cancellation respected during it.
  - [x] 2.1 RED: a stream ending in a terminal error; iterate to the end, then inspect `Err()`; confirm the terminal failure is reported with its category intact. `S-STK-033`.
  - [x] 2.2 RED: a stream that completes normally; confirm `Err()` reports none after the loop. `S-STK-034`.
  - [x] 2.3 RED: a mid-iteration cancellation; confirm iteration ends before a bounded wait deadline rather than blocking. `S-STK-035`.
    - RED output (2.1–2.3): `go test -race -run 'TestIter_' -v ./src/agenttest/...` → `src/agenttest/stream_kit_iter_test.go:115:18: view.Err undefined (type *agenttest.Iter has no field or method Err, but does have unexported field err)` (× 3 call sites) → `FAIL [build failed]`. Genuinely distinct from item 1's RED: item 1's tests still compiled and passed at this point (their own dependency, `NewIter`, already existed).
  - [x] 2.4 GREEN: implement `(*Iter).Events(ctx context.Context) iter.Seq[ai.Event]` and `(*Iter).Err() error` per design D5 (range-over-func, `bufio.Scanner`-style post-loop error). Confirm 2.1–2.3 pass. GREEN output: added the `ctx`-`select` branch and terminal-error tracking to `Events`, plus `Err()` → `go test -race -run 'TestIter_' -v ./src/agenttest/...` → all 5 `TestIter_*` tests `--- PASS` (items 1 and 2 together, no regression), `PASS`, `ok github.com/cachicamas/backend/agent/src/agenttest 1.579s`. Stress-checked for flakiness (channel-close-vs-`ctx.Done()` race at end of stream): 5× repeated `go test -race -count=1 -run 'TestIter_' ./src/agenttest/...` → all 5 `ok`, no flakes. Design note: the `case <-ctx.Done(): if it.err == nil { it.err = ctx.Err() }` guard is load-bearing — without the `it.err == nil` check, a terminal-error event immediately followed by the channel's own close could race against an already-cancelled `ctx` in `select`, letting `ctx.Err()` overwrite the already-recorded terminal failure non-deterministically; the guard makes the "terminal failure wins" precedence sticky rather than last-write-wins.
  - [x] 2.5 REFACTOR: confirm `Err()`'s precedence (terminal failure, else `ctx.Err()`, else nil) is exercised by all three tests above, not assumed. Confirmed: S-STK-033 exercises "terminal failure" (asserts a non-nil `*ai.Failure`); S-STK-034 exercises "else nil" (normal completion, no cancellation); S-STK-035 exercises "else `ctx.Err()`" (cancellation, no terminal event). All three branches of the precedence are covered by a real, distinct scenario.

- [x] **Item 3** (`R-STK-013`) — Lives outside the provider package; signature guard passes unmodified.
  - [x] 3.1 Diff-check (`S-STK-036`): confirm no edit exists to `src/ai/provider.go` or to AI-20.4's signature-guard test file. Confirmed: `git diff --stat -- backend/agent/src/ai/provider.go backend/agent/src/agenttest/provider_signature_guard_test.go` → empty.
  - [x] 3.2 Regression run (`S-STK-037`): run the existing signature guard; confirm it still passes, observing the single decided method with its decided carrier result. Confirmed: `go test -race -run 'TestModelProviderInterface_SignatureGuard' -v ./src/agenttest/...` → `--- PASS: TestModelProviderInterface_SignatureGuard`, `PASS`.

- [x] **Item 4** *(appended, `NFR-STK-E`)* — Extreme inputs never panic.
  - [x] 4.1 RED: `NewIter` against a nil channel, and `Err()` called before any iteration; confirm no panic and an attributable result. `S-STK-044` (partial).
  - [x] 4.2 GREEN: guard both cases explicitly. Confirm 4.1 passes. RED/GREEN output: `go test -race -run 'TestIter_ExtremeInputs' -v ./src/agenttest/...` → **both subtests PASS immediately** (nil channel: `Events` yields 0 events via the `it.ch == nil` guard already added in item 1's GREEN; `Err()` before any iteration: `it.err`'s zero value is `nil`, Go's own default) — honestly recorded, no forced failure; both guards were already in place from earlier items.

- [x] **Item 5** *(`NFR-STK-F`)* — Package documentation names the kit.
  - [x] 5.1 Modify `doc.go`: name the stream test kit alongside AI-21's fake, retain AI-21's existing framing, state the dependency-free pin `R-STK-009` depends on. `S-STK-045`. Done: extended the existing "library role" numbered item to name all five exported kit entry points (`DrainAndRecord`/`Recording`, `RequireSameEvents`, `RequireValidStream`/`CheckContiguity`, `RequireNoGoroutineLeak`, `Iter`); added a new "Dependency-free (R-STK-009)" section citing the AI-22.4 rejection decision and `go.mod`'s zero requires; AI-21's proof-role/library-role numbered framing and the "Contract-faithful" / "sibling of src/ai" sections are otherwise untouched. Verified rendering with `go doc ./src/agenttest`.

- [x] **AI-22.5 close:** record green `make test` and clean `make lint`; confirm `Iter`'s exported methods cite `R-STK-011`/`R-STK-012`; commit `feat(agenttest): carrier view over a stream, package doc updated (AI-22.5)`.
  - `make test`: `go test -race -count=1 ./...` (forced fresh run) → `ok github.com/cachicamas/backend/agent/src/agenttest 1.886s`, `ok github.com/cachicamas/backend/agent/src/ai 3.533s`.
  - `make lint`: first run caught `revive: empty-block` in `stream_kit_iter_test.go` (a deliberately-empty `for range ... {}` drain loop) → fixed by counting yielded events and asserting the count (strengthens the test, not just satisfies the linter) → re-run: `0 issues.`
  - Doc-citation gap found and fixed during this close's own NFR-STK-D sweep (see milestone close checklist below): `Recording.Len()`'s doc comment cited no requirement identifier; `NewIter`'s cited only `NFR-STK-E`, not `R-STK-011`. Both fixed; re-ran `make lint` + `go test -race -count=1 ./...` → still clean/green.
  - Doc comments: `Iter`'s doc cites `(R-STK-011)`; `NewIter` cites `(R-STK-011)`; `Events` cites `(R-STK-011, R-STK-012)`; `Err` cites `(R-STK-012)`. Confirmed.
  - Commit: `feat(agenttest): carrier view over a stream, package doc updated (AI-22.5)`.

---

## Milestone close

- [ ] `make test` green in `backend/agent/` (`go test -race -v ./...`) — paste the transcript.
- [ ] `make lint` clean in `backend/agent/` — paste the transcript. Run before every commit, not only at the end.
- [ ] `go.mod` still zero requires; both AI-00 import guards pass (`NFR-STK-A`, `S-STK-038`).
- [ ] `src/ai/` diff is empty (`NFR-STK-B`, `S-STK-039`); `src/ai` and AI-21's tests pass identically with the change reverted in isolation (`S-STK-040`).
- [ ] `ai/provider_test.go` and AI-21's `fake_*_test.go` files diff empty; both local helpers still compile and run unchanged (`NFR-STK-C`, `S-STK-041`).
- [ ] Every exported helper's doc comment names a Layer 1 requirement identifier (`NFR-STK-D`, `S-STK-042`) — confirm across all five files in one pass.
- [ ] Full suite run twice under `-race`; results identical (`S-STK-043`). Extreme-input items (3/3/3/5/4 across the five leaves) all green (`S-STK-044`).
- [ ] `doc.go` names both the fake and the kit, retains the dependency-free pin (`S-STK-045`).
- [ ] Every test-list item above carries recorded RED output, recorded GREEN output and a refactor note (`NFR-STK-G`, `S-STK-046`) — confirm no item was skipped.
- [ ] Record actual vs. forecast changed-line count in the Review Workload Forecast table above; update the running-total flag against AI-21's 2 810 and the wave's 5 000 ceiling.
- [ ] Never push, never merge, never open a PR, never `git stash`.

## Key Learnings

1. AI-22's spec and design were read in full and found mutually consistent, requiring no reconciliation pass unlike AI-21.
2. Design's D1→D5 file order matches the code dependency order because AI-22.2, .3 and .5 each consume AI-22.1's `Recording` or raw channel.
3. Three spec scenarios per leaf are inspection-only pins (code-diff or doc-content checks) rather than automated RED/GREEN Go tests, and were recorded as such rather than forced into fake tests.
4. NFR-STK-E's extreme-input requirement spans every exported entry point, so it was distributed as one appended test-list item per leaf instead of one late cross-cutting file.
5. The combined AI-21 actual plus this milestone's own forecast approaches or may exceed the wave's 5 000-line ceiling before AI-23 is planned, which is flagged explicitly for the orchestrator.
