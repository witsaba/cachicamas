```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:78b48b13d8735f9a2df79ef5c7d3c9dc4ebfde99ee31bae00439b7d84d7b7ce0
verdict: pass
blockers: 0
critical_findings: 0
requirements: 2/2
scenarios: 4/4
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:b088f1c2274ed140d54a6f8afd445693dfe40a50f99117f204631d160975faa1
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

# Verify Report — `cachicamas-ai-wave2-carryovers` (AI-41)

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 — Discharge the Wave-2 carryovers (doc 0002:2233–2257) · **Wave 5 — Harden**
> **Phase**: verify (closes the change's apply → verify → archive cycle)
> **Project**: cachicamas (witsaba)
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-41-wave2-carryovers`
> **Branch**: `feat/ai-41-wave2-carryovers` · **HEAD**: `b82cfdf` · **Base**: `origin/main@f2e460d`
> **Method**: every claim in `apply-progress.md` was re-executed, not accepted. Both RED states were independently reproduced with a non-mutating `go test -overlay`, so the worktree was never modified by this phase.

> **Archive note (2026-08-07)**: this report is the snapshot at verify time, HEAD `b82cfdf`. `sdd-archive` applied the recommended fix for **W-1** in commit `b271637` (narrowing the `S-AIP-008` exemption from `{Error, String, GoString}` to `{GoString}`), then re-ran `make test`, `make lint` and `make build`, all green. W-1 is therefore RESOLVED, not outstanding, as of archive close — see `archive-report.md` for the final-state record. **W-2** remains recorded as a process gap for future milestones, not a defect in this change.

---

## § 1. Identity

| Field | Value |
| --- | --- |
| **Change name** | `cachicamas-ai-wave2-carryovers` |
| **Milestone** | AI-41 (nodes AI-41.1, AI-41.2) |
| **Branch / HEAD** | `feat/ai-41-wave2-carryovers` / `b82cfdf` (2 commits ahead of `origin/main@f2e460d`) |
| **Module** | `backend/agent` — layered (ADR 0005 § D1) |
| **Spec scope** | `specs/ai-event-envelope/spec.md` (`R-AEE-021`; `S-AEE-071`, `S-AEE-072`) · `specs/ai-provider-errors/spec.md` (`R-AIP-016`; `S-AIP-056`, `S-AIP-057`) · `specs/ai-stream-testkit/spec.md` (requirement-free; archive-step wording only) |
| **Persistence mode** | hybrid (Engram + OpenSpec files) |
| **Delivery strategy** | `exception-ok` · review budget 1000 lines · actual 220 (+208/−12) at verify time, later 226 (+214/−12) after `b271637` |
| **Test runner** | `cd backend/agent && make test` (= `go test -race -v ./...`) |
| **Lint** | `cd backend/agent && make lint` (`go vet` + golangci-lint v2.9.0) |
| **Strict TDD** | ON — verified, not merely reported (§ 4) |

### Gates state (re-executed at verify time)

| Gate | Status | Evidence |
| --- | --- | --- |
| `cd backend/agent && make test` | ✅ PASS (exit 0) | 2375 PASS / 0 FAIL / 13 SKIP across 7 packages (`agenttest`, `ai`, `ai/internal/retry`, `ai/openaicompat` 164.9s, `openrouter`, `.../conformance`, `.../smoke`) |
| `cd backend/agent && make lint` | ✅ 0 issues (exit 0) | `go vet ./...` clean; `golangci-lint run --config=.golangci.yml ./...` → `0 issues.` |
| `cd backend/agent && make build` | ✅ PASS (exit 0) | `go build -trimpath ./...`, no output |
| `git diff origin/main -- backend/agent/go.mod backend/agent/go.sum` | ✅ EMPTY | byte-identical; zero new imports on either leaf |
| `git diff --stat origin/main` | ✅ 4 files, +208/−12 = **220 lines** | 780 lines of headroom under the 1000-line budget; no exception needed |
| `git log --oneline origin/main..HEAD` | ✅ 2 commits | `ca2ede7` (AI-41.1), `b82cfdf` (AI-41.2); conventional subjects; no AI-attribution trailers |
| Import boundary guard | ✅ PASS | `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` and `TestLayer1Package_IsImportableFromAnExternalPackage_Compiles` both `--- PASS` |
| File-naming convention | ✅ CONFIRMED | zero `a_i-41_*_test.go` files anywhere; all four changed files are topical (`event_test.go`, `export_test.go`, `provider_failure.go`, `provider_failure_test.go`) |
| Git worktree state | ✅ clean | only untracked `openspec/changes/cachicamas-ai-wave2-carryovers/` — the same expected state AI-35's verify recorded at this point; `sdd-archive` commits it under `openspec/changes/archive/` |

The 13 SKIPs are pre-existing (env-gated OpenRouter smoke + conformance bridge legs) and are untouched by this change — the diff contains no `_test.go` file outside `src/ai/`.

---

## § 2. Acceptance verdict (top-line)

**PASS WITH WARNINGS** — 2 of 2 requirements verified, 4 of 4 scenarios verified COMPLIANT at runtime, **0 CRITICAL**, **2 WARNING**, **3 SUGGESTION**.

Both leaves are genuinely proven, not merely named. The strongest evidence is that this phase reproduced both RED states independently: with the production change removed by overlay, `TestFailure_GoString_*` fails on all three planted canaries plus the nil case, and `TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4` fails with the exact message `apply-progress.md` cited. Neither test is vacuous.

The single deviation — apply narrowing a pre-existing out-of-scope guard (`S-AIP-008`) — is justified by that scenario's own spec text and does **not** break it, but the exemption is measurably broader than necessary. It is a WARNING, not a blocker. (Resolved at archive — see the archive note above.)

---

## § 3. Per-requirement verification

### R-AEE-021 — The emission boundary surfaces the offered payload's own rejection, proven directly

| Scenario | Status | Test | Evidence |
| --- | --- | --- | --- |
| **S-AEE-071** — planted rejection surfaced, matched by identity, sentinel and position intact | ✅ COMPLIANT | `TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4` at `backend/agent/src/ai/event_test.go:229–267` | `--- PASS` in `make test`. Four positive assertions, all present and all load-bearing (see below). |
| **S-AEE-072** — earlier rules satisfied by construction; only rule 4 can be responsible; default-state payload still accepted | ✅ COMPLIANT | same test, `event_test.go:256–266` | `--- PASS`. Two negative sentinel assertions + the zero-regression acceptance leg. |

**S-AEE-071 assertion-strength audit (scrutinised as instructed).** The spec demands identity, sentinel **and** position — not a non-nil check. All three are present at `event_test.go`:

| Line | Assertion | Proves |
| --- | --- | --- |
| `:239–241` | `err == nil → t.Fatal` | the boundary rejects at all |
| `:244–246` | `errors.As(err, &violation)` → `t.Fatalf` | a `*ai.Violation` is reachable |
| `:247–249` | `if violation != planted` — **pointer** comparison | **failure-value identity**: the exact planted value, "not merely an equal one" (the test's own words) |
| `:250–252` | `errors.Is(err, sentinelRule)` where `sentinelRule = ai.ErrMalformed` | **sentinel rule** survives unchanged through `Unwrap` |
| `:253–255` | `reflect.DeepEqual(violation.Path(), planted.Path())` | **position** — the planted `ai.At("witness_rule")`, unchanged |

A non-nil check alone would **not** have satisfied this; the identity assertion is what carries the requirement. Confirmed against the production path: `CheckEmit`'s rule 4 is `return e.payload.validate(Path{At("event")})` (`event.go:364–366`) and `WitnessPayload.validate` is `return w.rejectWith` (`export_test.go:79`), which **ignores** the `at` parameter — so the surfaced violation is the planted pointer with the payload's own position, exactly as `R-AEE-021` requires ("carrying its own position rather than one derived from where validate was called").

**Negative assertions confirmed present** (`event_test.go:256–261`): `errors.Is(err, ai.ErrNotInVocabulary)` must be false and `errors.Is(err, ai.ErrOutOfRange)` must be false. Both are real discriminators, because `CheckEmit`'s rules use disjoint sentinels: rule 1 → `ErrNotInVocabulary` (`event.go:343`), rule 2 → `ErrOutOfRange` at `event.sequence` (`event.go:349`), rule 3 → `ErrOutOfRange` at `event.block` (`event.go:360`), rule 4 → whatever the payload planted (here `ErrMalformed`, `validation.go:91`). The planted sentinel is provably distinct from every earlier rule's.

**S-AEE-072 attributability — the structural argument holds, verified in source.** The claim "only rule 4 can fire" was checked against the code rather than accepted:

1. **Rule 1 satisfied**: `KindTestWitness` is registered at package-test-load time by `export_test.go:84–92`'s `init()`, so `eventRegistryEntry` returns `ok == true`.
2. **Rule 2 satisfied**: the event is stamped via `var s ai.Stamper; s.Stamp(...)` (`event_test.go:235–236`), yielding a non-zero, legal `Sequence` per `R-AEE-010`.
3. **Rule 3 exits early — confirmed**: the witness kind's descriptor is registered as `EventDescriptor{Role: BlockRoleNone, Cardinality: CardinalityAny, Terminal: false}` (`export_test.go:88–91`), and `CheckEmit`'s rule 3 opens with `if entry.descriptor.Role == BlockRoleNone { return nil }` (`event.go:353–357`). The block-scoped rule therefore returns **without a verdict**, exactly as `S-AEE-072` asserts. This is not a paraphrase of the design; it is the two source facts read at HEAD.
4. **Zero-regression half** (`event_test.go:263–266`): the same event built through `ai.NewWitnessEvent(1)` (whose `rejectWith` is nil by default) is asserted to be accepted — proving the additive field left every pre-existing construction path reporting success, as `R-AEE-021` clause (b) requires.

**RED independently reproduced.** Overlaying `export_test.go` to restore `validate` to `return nil`:

```text
--- FAIL: TestCheckEmit_PayloadReportsOwnViolation_SurfacedAsRule4 (0.00s)
    event_test.go:240: ai.CheckEmit(rejecting) = nil, want the planted violation surfaced — validate still returns nil unconditionally
```

Byte-identical to the message `apply-progress.md` reported. The one-line GREEN (`return w.rejectWith`) is the sole cause of the pass.

**Status: PASS** — 2/2 scenarios COMPLIANT.

---

### R-AIP-016 — Redaction is a property of the failure payload, not of the caller's formatting verb

| Scenario | Status | Test | Evidence |
| --- | --- | --- | --- |
| **S-AIP-056** — planted sentinel absent from all four verbs; no other cause fragment in the Go-syntax output | ✅ COMPLIANT | `TestFailure_GoString_RedactsLikeError/no verb reproduces a planted canary (S-AIP-056)` at `provider_failure_test.go:1294–1305` | `--- PASS` |
| **S-AIP-057** — Go-syntax rendering byte-for-byte identical to the redacted textual rendering; absent payload total under all four verbs | ✅ COMPLIANT | `.../%#v is byte-for-byte identical to Error() (S-AIP-057)` at `:1307–1313` + `TestFailure_GoString_NilReceiver_TotalByDelegation` at `:1322–1341` | `--- PASS` ×5 (parent + `%v`, `%s`, `%+v`, `%#v`) |

**S-AIP-056 non-vacuity — independently confirmed by reproducing the RED, not by trusting the report.** The committed test plants all three canaries in the fields where today's reflective `%#v` rendering actually reproduces them (`provider_failure_test.go:1287–1292`): `RawLabel: "raw-label-CANARY-77f3"`, `RequestID: "request-id-CANARY-88e1"`, and `Cause: canaryCause("cause-CANARY-99a2")` — where `canaryCause` is a test-local **value-kind** defined string type (`:1267`), not an `errors.New` pointer. Overlaying `provider_failure.go` to delete `GoString` produced:

```text
--- FAIL: TestFailure_GoString_RedactsLikeError/no_verb_reproduces_a_planted_canary_(S-AIP-056)
    provider_failure_test.go:1301: fmt.Sprintf("%#v", f) = "&ai.Failure{category:0x9, retryable:false,
      retryAfter:ai.RetryDelay{delay:0, present:false}, rawLabel:\"raw-label-CANARY-77f3\", statusClass:0,
      requestID:\"request-id-CANARY-88e1\", cause:\"cause-CANARY-99a2\", delivery:0x1, partialOutput:false}",
      contains the planted canary "raw-label-CANARY-77f3", want it excluded
    (…same for "request-id-CANARY-88e1" and "cause-CANARY-99a2")
```

All **three** canaries leak. This matches the orchestrator's independent RED byte-for-byte and empirically vindicates design decision **D-4**: had the cause been a pointer-shaped `errors.New`, `fmt`'s reflective walk would have printed a bare hex address and the canary planted there would have passed before `GoString` existed — a vacuous RED. The value-kind choice is what makes the third canary real, and the RED output proves the trap was live.

**S-AIP-057 byte-for-byte agreement — confirmed.** `provider_failure_test.go:1310–1312`: `want := f.Error()` then `if got := fmt.Sprintf("%#v", f); got != want`. Exact string equality against the type's own textual rendering, computed at runtime — never a hard-coded literal. The same overlay RED shows it failing with `want "provider failure: unknown"`. This assertion is also what discharges `S-AIP-056`'s second clause ("no other fragment of the wrapped cause appears in the Go-syntax output"): byte equality with a fixed-prefix-plus-category string is strictly stronger than any fragment-absence enumeration.

**Nil-receiver totality — confirmed asserting against the method, not a literal.** `provider_failure_test.go:1325`: `want := (*ai.Failure)(nil).Error()`. The assertion is computed once from the production method and compared across all four verbs, each sub-test wrapped in `defer recover()` to prove no panic (`:1331–1335`). The overlay RED confirms it is a real check: without `GoString`, `%#v` renders `"(*ai.Failure)(nil)"` against `want "no provider failure"`. Implementation matches **D-5** exactly — `func (f *Failure) GoString() string { return f.Error() }` (`provider_failure.go:371`) with no second nil check; totality is inherited from `Error`'s single nil branch (`:349–354`), so `NFR-AIP-B` holds with one nil check in one place.

**Status: PASS** — 2/2 scenarios COMPLIANT.

---

### `ai-stream-testkit` delta (requirement-free)

Correctly carries no `R-STK-*` or `S-STK-*` identifier and consumes none. It fixes, in advance, the exact append-only target wording for the carryover ledger at `openspec/specs/ai-stream-testkit/spec.md:39`. Verified per `sdd-spec`'s own note (risk iv): checked by archive-step acceptance criteria, not by requirement count. The canonical line 39 is confirmed still in its pre-amendment "Assigned 2026-08-03 (AI-24)" state at HEAD — correct, because the amendment is an archive-phase task (A.4), not an apply-phase one.

---

## § 4. Numbering and traceability

| Check | Result | Evidence |
| --- | --- | --- |
| `R-AEE-021` follows the canonical maximum | ✅ | canonical max is `R-AEE-020`; `S-AEE-071`/`S-AEE-072` follow `S-AEE-070` |
| `R-AIP-016` follows the canonical maximum | ✅ | canonical max is `R-AIP-015`; `S-AIP-056`/`S-AIP-057` follow `S-AIP-055` |
| No identifier collision | ✅ | spec-phase risk 7.4 stays closed at HEAD |
| No Go identifier leaked into a delta spec | ✅ | all three delta files are behavior-only |

---

## § 5. Design coherence

| Decision | Followed? | Evidence |
| --- | --- | --- |
| **D-1** — `GoString()` only, no `String()` (a `String` on an `error` type is unreachable through `fmt`) | ✅ Yes | `provider_failure.go:371` adds `GoString` and nothing else; `go doc ./src/ai Failure` shows no `String` method |
| **D-2** — additive `rejectWith *Violation`; `validate` returns it; `NewRejectingWitnessEvent` constructor | ✅ Yes | `export_test.go:54`, `:79`, `:115–117`; nil default keeps the struct comparable and every existing path green |
| **D-3** — the rule-4 test uses `t.Parallel()` | ✅ Yes | `event_test.go:230`; the test registers no kind, so `export_test.go:29–30`'s serial discipline does not bind it — verified against that file comment |
| **D-4** — canaries in `RawLabel` + `RequestID` + a **value-kind** cause | ✅ Yes | `provider_failure_test.go:1267`, `:1287–1292`; empirically proven necessary by the reproduced RED |
| **D-5** — nil receiver total by delegation, no second nil check | ✅ Yes | `provider_failure.go:371` is a pure delegation; `Error`'s nil branch at `:349–354` is the only one |
| **Blast radius** — "zero `%#v`/`GoString` occurrences in `provider_failure_test.go`" | ⚠️ Incomplete | The design's grep-based check missed the reflection-based `S-AIP-008` guard. See **W-1**. |

---

## § 6. The `S-AIP-008` deviation — independent judgement

`apply` modified a **pre-existing, out-of-scope** guard: the `S-AIP-008` sub-test inside `TestErrorEvent_TerminalExclusivity_NeverBothCompletionAndError` (`provider_failure_test.go:189–217`). It previously asserted **zero** shared exported method names between `ai.Completion` and `*ai.Failure`; it now skips three names via `diagnosticRenderers := map[string]bool{"Error": true, "String": true, "GoString": true}` (`:192`).

### 6.1 Is the narrowing justified by `S-AIP-008`'s own spec text?

**Yes.** `openspec/specs/ai-provider-errors/spec.md:76` reads: "no accessor name or return type is shared **in a way that lets a consumer read a category off a completion or a finish reason off a failure**." That trailing clause is a **purposive qualifier**, not decoration. The old test implemented an unconditional reading that was strictly stronger than the scenario it claimed to prove — a test over-fitted to its spec.

Checked against behaviour rather than naming: `Completion.GoString()` delegates to `Completion.String()`, which returns the fixed literal `"completion"` (`completion.go:99`, `:106`) — it carries no category and no failure data. `(*Failure).GoString()` delegates to `Error()`, which returns `"provider failure: " + f.category.String()` (`provider_failure.go:353`) — it carries no finish reason and no usage. Neither direction of the prohibited read is opened. The narrowing is therefore **faithful to the spec**, and the apply phase's root-cause reasoning is correct.

### 6.2 Is the three-name exemption broader than necessary?

**Yes — measurably, and I proved it rather than asserting it.** Authoritative method sets at HEAD (`go doc`):

- `ai.Completion`: `FinishReason`, `GoString`, `String`, `Usage`
- `*ai.Failure`: `Category`, `Delivery`, `Error`, `GoString`, `Is`, `PartialOutput`, `RawLabel`, `RequestID`, `RetryAfter`, `Retryable`, `StatusClass`, `Unwrap`

**The intersection is exactly `{GoString}`.** Consequently:

- **`"Error"` is dead today.** The loop iterates `completionMethods`, and `Completion` has no `Error` method — so that map entry can never be consulted. It is unreachable code in a guard test.
- **`"String"` is an unnecessary pre-emptive weakening.** `Completion` has `String`; `*Failure` does not. Exempting it buys nothing now and disables a future signal.

Empirical proof of the detection loss, via non-mutating overlay. I planted `func (f *Failure) String() string { return "finish reason: stop" }` — a method that literally reports completion-shaped data under an exempted name — and ran the guard both ways:

| Exemption map | Result |
| --- | --- |
| Current `{Error, String, GoString}` | `ok github.com/cachicamas/backend/agent/src/ai` — **the collision passes silently** |
| Narrowed `{GoString}` | `FAIL … both ai.Completion and ai.Failure export a non-diagnostic accessor named "String"` |

A one-name exemption is therefore **strictly better**: it is sufficient today (verified — the full sub-test passes unchanged with `{"GoString": true}` against unmodified production code) and strictly more sensitive tomorrow.

Against that: "the three diagnostic renderers of Go's formatting protocol" (`error`, `fmt.Stringer`, `fmt.GoStringer`) is a genuinely principled, closed, well-known category rather than an ad-hoc list, and the test's comment documents the reasoning honestly. That is a defensible design; it is simply not the tightest one, and the tightest one costs one line.

### 6.3 Are the guard's real teeth intact?

**Yes — verified by execution, not inspection alone.** The four data-accessor assertions at `:205–216` are byte-for-byte unchanged and completely unexempted:

```
!completionMethods["FinishReason"] → t.Fatal   (fixture guard)
 failureMethods["FinishReason"]    → t.Error   (the load-bearing prohibition)
!failureMethods["Category"]        → t.Fatal   (fixture guard)
 completionMethods["Category"]     → t.Error   (the load-bearing prohibition)
```

Probe: overlaying `func (f *Failure) FinishReason() FinishReason` and `func (f *Failure) Usage() Usage` onto production code makes the guard fire three times —

```text
provider_failure_test.go:202: both ai.Completion and ai.Failure export a non-diagnostic accessor named "FinishReason"
provider_failure_test.go:202: both ai.Completion and ai.Failure export a non-diagnostic accessor named "Usage"
provider_failure_test.go:209: ai.Failure exports FinishReason, want a finish reason readable only off a completion
```

The exhaustive loop still catches arbitrary new names, and the two named prohibitions still fire independently.

### 6.4 Can a future data accessor collide undetected?

**Only under one of the three exempted names.** A future `Category()` , `Usage()`, `Retryable()` or any other name is caught (proven in 6.3). The residual hole is exactly: a data-bearing method named `Error`, `String`, or `GoString` added to whichever type currently lacks it. That is narrow and contrary to Go convention — but it is a real hole, and 6.2's `String()` probe demonstrates it concretely. Narrowing to `{GoString}` shrinks the hole from three names to one, and that one (`GoString`) is the collision the spec text explicitly tolerates.

### 6.5 Verdict on the deviation

**ACCEPTABLE WITH A NARROWER EXEMPTION.** Not a blocker: the spec text authorises the narrowing, the guard's load-bearing assertions are provably intact, the change is isolated to a single test body, and it is independently revertible via `git revert b82cfdf`.

**Recommendation, stated plainly**: replace `map[string]bool{"Error": true, "String": true, "GoString": true}` with `map[string]bool{"GoString": true}` and reword the accompanying comment to name the one real collision. It is a one-line change, it keeps the full suite green (verified), it deletes a provably-unreachable map entry, and it restores a detection capability this report empirically showed was lost. If the reviewer prefers the "three formatting-protocol renderers" framing, the change is acceptable as-is — but the narrower form is the better of the two, and I recommend it.

**Archive-time resolution**: this recommendation was applied verbatim in commit `b271637`; see `archive-report.md` for the final proof.

---

## § 7. Task completeness

| Metric | Value |
| --- | --- |
| Apply-phase tasks (Phases 1–3, WU-1…WU-5) | 19 |
| Apply-phase tasks complete `[x]` | **19 / 19** |
| Archive-phase tasks (Phase 4, A.1–A.5) | 5, all `[ ]` at verify time — **correctly deferred** |

**Phase 4 was correctly NOT executed by apply.** Independently confirmed at HEAD, not read from the report:

| Archive task | Target | State at HEAD (verify time) |
| --- | --- | --- |
| A.1 promote `R-AEE-021` | `openspec/specs/ai-event-envelope/spec.md` | ❌ not present — correct |
| A.2 discharge blockquote | same file, "Carried forward" `:322–324` | ❌ not appended — correct (section still reads as the open `W1` record) |
| A.3 promote `R-AIP-016` | `openspec/specs/ai-provider-errors/spec.md` | ❌ not present — correct |
| A.4 amend carryover ledger | `openspec/specs/ai-stream-testkit/spec.md:39` | ❌ still "Assigned 2026-08-03 (AI-24)" — correct |
| A.5 amend doc 0002 | `docs/architecture/milestones/0002-…md` | ❌ not amended — correct |

No canonical spec and no architecture doc is touched by either commit. `git diff --name-only origin/main` returns exactly the four Go files.

---

## § 8. Strict TDD compliance

| Check | Result | Details |
| --- | --- | --- |
| TDD evidence reported | ✅ | `apply-progress.md` documents RED → GREEN → REFACTOR for both leaves, with verbatim command output |
| All work units have tests | ✅ | 2/2 behavioural leaves; WU-5 is the gate run, not a test author |
| **RED confirmed — reproduced, not trusted** | ✅ | Both REDs re-created here with `go test -overlay` (§ 3). Leaf 1's failure message is byte-identical to the report's; leaf 2's shows all three canaries leaking, matching the report and the orchestrator's independent run. |
| GREEN confirmed (tests pass now) | ✅ | 3 new top-level tests + 7 sub-tests all `--- PASS` in the full `-race` run |
| Triangulation adequate | ✅ | `R-AEE-021`: 6 distinct assertions across 2 scenarios in one test (identity / sentinel / position / 2 negatives / zero-regression). `R-AIP-016`: 4 verbs × 3 canaries + 1 byte-equality + 4 nil verbs. Each spec scenario has ≥ 1 dedicated named assertion. |
| Safety net for modified files | ✅ | All four files are modifications, not new. The safety net is the pre-existing suite — and it worked: it is precisely what caught the `S-AIP-008` collision that the design's grep missed. |
| REFACTOR | ➖ | Not objectively verifiable; both doc-comment refreshes landed in the same edit as their GREEN, as planned |

**TDD compliance**: 6/7 verifiable checks passed; the discipline was genuinely followed, not merely claimed. The decisive evidence is that removing either production change re-reds the corresponding test with the exact message apply recorded.

### Test layer distribution (informational)

| Layer | Tests | Files | Notes |
| --- | --- | --- | --- |
| Unit (in-process, no I/O) | 3 top-level + 7 sub-tests | `event_test.go`, `provider_failure_test.go` | Pure `ai` package; no HTTP, no filesystem, no golden files |
| Integration | 0 | — | Correctly none: both requirements are about in-process rendering and validation |
| E2E | 0 | — | N/A |

### Assertion quality audit (Step 5f)

Every new and modified assertion was read. No tautology, no orphan-empty check, no type-only assertion standing alone, no ghost loop, no smoke test, no mock-heavy file. Both verb loops iterate a **literal, non-empty** slice (`[]string{"%v", "%s", "%+v", "%#v"}`), so they cannot silently execute zero iterations.

| File | Line | Assertion | Note | Severity |
| --- | --- | --- | --- | --- |
| `event_test.go` | 247 | `violation != planted` | Pointer identity — the strongest available form | ✅ OK |
| `event_test.go` | 253 | `reflect.DeepEqual(violation.Path(), planted.Path())` | Logically implied by line 247 on the success path; still independent on failure (`t.Errorf`, not `Fatalf`) and mandated by the spec's "same position" clause | ✅ OK (see S-1) |
| `event_test.go` | 256, 259 | `errors.Is(err, ErrNotInVocabulary/ErrOutOfRange)` must be false | Real discriminators — disjoint from rule 4's sentinel | ✅ OK |
| `provider_failure_test.go` | 1300 | `strings.Contains(rendered, canary)` over 4 verbs × 3 canaries | Non-vacuous, proven by RED reproduction | ✅ OK |
| `provider_failure_test.go` | 1310 | `got != want` where `want = f.Error()` | Runtime-derived expectation, never a literal | ✅ OK |
| `provider_failure_test.go` | 1325 | `want := (*ai.Failure)(nil).Error()` | Runtime-derived; cannot drift from `noProviderFailure` | ✅ OK |

**Assertion quality**: ✅ 0 CRITICAL, 0 WARNING — all assertions verify real behaviour.

### Quality metrics

| Check | Result |
| --- | --- |
| Linter | ✅ 0 issues (`make lint` → `0 issues.`) |
| Vet | ✅ clean (`go vet ./...`) |
| Coverage | ➖ Not available — no coverage target in the Makefile; per `strict-tdd-verify.md` Step 5d this is not a failure |

---

## § 9. Findings

### CRITICAL — none

### WARNING

- **W-1 — The `S-AIP-008` exemption is broader than the collision that forced it.** Only `GoString` collides between `ai.Completion` and `*ai.Failure` at HEAD. `"Error"` is unreachable by the loop (Completion has no `Error` method) and `"String"` pre-emptively disables a live signal. Empirically demonstrated in § 6.2: a `*Failure.String()` returning `"finish reason: stop"` passes the current guard and fails the narrowed one. **Recommended fix**: `map[string]bool{"GoString": true}`, one line, suite stays green. Does not block archive. **RESOLVED at archive in commit `b271637`.**
- **W-2 — The design's blast-radius method was too weak, and the gap is uncorrected.** `design.md` cleared this change by grepping `provider_failure_test.go` for the strings `%#v` and `GoString` (count 0) and concluded there was no impact. That method cannot see a *reflection-based* guard, which is exactly what `S-AIP-008` is. Apply recovered well, but a future change adding an exported method to a type covered by a method-set guard will hit the same blind spot. **Recommended follow-up**: when a change adds an exported method, grep for `reflect.TypeOf(<Type>` / `NumMethod` in addition to literal usage strings. Process finding, not a code defect. **Recorded as a process gap for future milestones at archive — not corrected in this change, correctly out of scope.**

### SUGGESTION

- **S-1 — Redundant-on-success position assertion.** `event_test.go:253`'s `reflect.DeepEqual(violation.Path(), planted.Path())` is implied by the pointer-identity assertion seven lines above it whenever the test passes. It is harmless, spec-mandated, and still meaningful on the failure path (both use `t.Errorf`). No change needed; noted so a future reader does not mistake it for independent coverage.
- **S-2 — `S-AIP-056`'s second clause is proven transitively.** "No other fragment of the wrapped cause appears in the Go-syntax output" is discharged by `S-AIP-057`'s byte-for-byte equality, not by a direct fragment enumeration. Byte equality is strictly stronger, so coverage is real — but the two scenarios are now coupled. If `S-AIP-057` is ever relaxed, `S-AIP-056`'s second clause loses its proof.
- **S-3 — `apply-progress.md` miscounts its own tasks.** It reports "18/18 apply-phase tasks"; the actual count in `tasks.md` is 19 (WU-1 ×3, WU-2 ×4, WU-3 ×4, WU-4 ×4, WU-5 ×4). All 19 are `[x]` and all were verified, so this is a cosmetic self-report error with no effect on completeness.

---

## § 10. Acceptance check

- [x] **`R-AEE-021`** holds — `S-AEE-071` and `S-AEE-072` COMPLIANT; identity, sentinel and position all asserted; attributability structurally verified in source (`BlockRoleNone` → rule 3 early-exits at `event.go:355–356`).
- [x] **`R-AIP-016`** holds — `S-AIP-056` and `S-AIP-057` COMPLIANT; three canaries proven non-vacuous by reproducing the RED; `%#v` byte-identical to `Error()`; nil receiver total by delegation.
- [x] **`ai-stream-testkit` delta** — requirement-free by design; archive-step wording fixed in advance; canonical `:39` correctly still un-amended at HEAD.
- [x] **`make test`** — exit 0, 2375 PASS / 0 FAIL / 13 pre-existing SKIP.
- [x] **`make lint`** — exit 0, `0 issues.`
- [x] **`make build`** — exit 0.
- [x] **`go.mod` / `go.sum`** — byte-identical to `origin/main`; zero new imports.
- [x] **Import boundary guard** — both Layer-1 import tests PASS.
- [x] **No `a_i-41_*_test.go`** — confirmed; that convention stays confined to `openaicompat/`.
- [x] **Changed lines** — 220 (+208/−12) across 4 files at verify time, against a 1000-line budget.
- [x] **Phase 4 (A.1–A.5) not executed** — confirmed against all five canonical targets at HEAD, at verify time.
- [x] **Strict TDD** — genuinely followed; both REDs independently reproduced.
- [x] **`S-AIP-008` exemption narrowed to `{GoString}`** — RESOLVED at archive in commit `b271637` (recommended, not required, at verify time — W-1).

---

## § 11. Handover to `sdd-archive`

The archive phase must still:

1. **A.1** — promote `R-AEE-021` (+ `S-AEE-071`, `S-AEE-072`) into `openspec/specs/ai-event-envelope/spec.md` after `R-AEE-020`.
2. **A.2** — append the dated discharge blockquote to that spec's "Carried forward" section (`:322–324`), append-only; the existing paragraph is load-bearing history and must survive byte-for-byte.
3. **A.3** — promote `R-AIP-016` (+ `S-AIP-056`, `S-AIP-057`) into `openspec/specs/ai-provider-errors/spec.md` after `R-AIP-015`, before the non-functional-requirements section.
4. **A.4** — amend the carryover ledger at `openspec/specs/ai-stream-testkit/spec.md:39`, append-only, using the exact target text pinned in this change's `ai-stream-testkit` delta. Touch no `R-STK-*` requirement.
5. **A.5** — amend `docs/architecture/milestones/0002-…md`: append the AI-41 close blockquote after the AI-35 one, bump the shipped counter `36 of 42` → `37 of 42`, drop AI-41 from "Remaining in Wave 5", update the top status line. **Caution**: AI-41 ships *before* AI-36/AI-37 — do not claim a contiguous landed range.
6. **Commit the change folder** under `openspec/changes/archive/2026-08-07-cachicamas-ai-wave2-carryovers/` (currently untracked, matching the AI-35 pattern at this stage).
7. **Carry W-1 into the PR description** so the reviewer sees the `S-AIP-008` narrowing and this report's recommendation explicitly, rather than discovering it in the diff.

**Recommended next phase**: `sdd-archive`.
