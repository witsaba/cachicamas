```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:61d455db3fb59784385c032444af3c3bb474df1667063b19296a7dff27199a63
verdict: fail
blockers: 1
critical_findings: 1
requirements: 104/104
scenarios: 345/345
test_command: make test
test_exit_code: 0
test_output_hash: sha256:61d455db3fb59784385c032444af3c3bb474df1667063b19296a7dff27199a63
build_command: make lint
build_exit_code: 0
build_output_hash: sha256:66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a
```

## Verification Report — Layer 1 Wave 2 "Stream" (AI-14 … AI-20)

**Change**: `cachicamas-ai-layer-1` Wave 2 — seven SDD changes verified as one deliverable
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-2`
**Branch / head**: `feat/2026-08-01-cachicamas-ai-layer1-wave-2` @ `b4cbab4` (33 commits)
**Merge base**: `664a132`
**Charter**: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 851–1163
**Mode**: Strict TDD · hybrid persistence · `exception-ok` single PR, ~5000-line budget pre-accepted

**Verdict**: **FAIL — blocked on one delivery precondition.** 1 CRITICAL, 6 WARNING, 5 SUGGESTION.

Read this verdict precisely. **Every implementation dimension passes**: 223/223 tasks complete, 104
requirements landed, 345 scenarios covered, 336 tests green under `-race`, `golangci-lint` clean, both
AI-00 import guards holding, `go.mod` still at zero requires, and all seven per-milestone verdicts in § 4
are PASS. Not one line of Go needs to change.

The wave is nonetheless **blocked**, because 26 OpenSpec artifact files totalling 3 595 lines — including
all five `spec.md` files for AI-16 through AI-20 — are **untracked** and therefore absent from the pull
request and from version control entirely (**C1**). Archive cannot promote a specification that is not in
the repository, and a reviewer cannot check 12 236 lines of Go against requirements the PR does not
contain. The remedy is `git add openspec/changes/` and a commit; the verdict flips to PASS WITH WARNINGS
the moment that lands, with no re-run of the test suite required.

---

## 1. Completeness

| Metric | Value |
| --- | --- |
| Changes verified | 7 / 7 |
| Task checkboxes total | 223 |
| Task checkboxes complete | 223 |
| Task checkboxes incomplete | **0** |
| Delta-spec requirements | 104 |
| Delta-spec scenarios | 345 |
| doc 0002 Wave 2 nodes | 23 `[leaf]` + 2 `[guard]` + 0 `[decision]` = 25 |
| Nodes with a landed spec section | 25 / 25 |
| Production Go files (module total) | 33 (was 19 at Wave 1 close) |
| Test Go files (module total) | 44 (was 27 at Wave 1 close) |
| PR diff, tracked (three-dot vs `664a132`) | 47 files, 14 326 insertions, 2 deletions |
| Artifacts on disk but **untracked** | 26 files, 3 595 lines — see **C1** |
| True wave size (tracked + untracked) | 73 files, ~17 921 lines |

Per-change task state and spec size:

| Milestone | Change | Boxes | Complete | Open | Reqs | Scenarios | Prefix |
| --- | --- | --- | --- | --- | --- | --- | --- |
| AI-14 | `cachicamas-ai-event-envelope` | 55 | 55 | 0 | 20 | 70 | `R-AEE-` |
| AI-15 | `cachicamas-ai-response-events` | 25 | 25 | 0 | 11 | 41 | `R-ARP-` |
| AI-16 | `cachicamas-ai-text-events` | 37 | 37 | 0 | 11 | 32 | `R-ATE-` |
| AI-17 | `cachicamas-ai-reasoning-events` | 14 | 14 | 0 | 14 | 43 | `R-ARE-` |
| AI-18 | `cachicamas-ai-tool-call-events` | 25 | 25 | 0 | 12 | 38 | `R-ATC-` |
| AI-19 | `cachicamas-ai-provider-errors` | 38 | 38 | 0 | 15 | 55 | `R-AIP-` |
| AI-20 | `cachicamas-ai-model-provider` | 29 | 29 | 0 | 21 | 66 | `R-AMP-` |
| | **Total** | **223** | **223** | **0** | **104** | **345** | |

**Prefix collision check — clean.** Every one of the seven specs declares requirements under exactly one
prefix, and no two specs share one. The `R-ARP-` / `R-ARE-` split that the wave plan flagged (AI-15
re-prefixed off the original `R-ARE-` collision so AI-17 could keep it) held: `grep '^### R-'` yields
`AEE`, `ARP`, `ATE`, `ARE`, `ATC`, `AIP`, `AMP` — seven prefixes, seven specs, zero overlap. This is a
direct improvement on Wave 1's W3, which shipped a live `R-AMR-` collision into verification.

**Traceability.** 101 of 104 requirement IDs are cited by identifier in the landed Go source or tests
(`R-AEE` 20/20, `R-ATE` 11/11, `R-ARE` 14/14, `R-ATC` 12/12, `R-AIP` 15/15, `R-AMP` 19/21, `R-ARP`
10/11). 239 of 345 scenario IDs are cited the same way — **69 %, against Wave 1's 27 %**. Wave 1's
suggestion S1 ("adopt the `S-*` comment convention uniformly from the first leaf") was adopted by all
seven milestones. The three uncited requirements are analysed in **S5**.

---

## 2. Evidence gate — recorded execution

All commands re-run by this verification from `backend/agent/` at head `b4cbab4`. The first `make test`
returned a **cached** result (`ok ... (cached)`); it was discarded, `go clean -testcache` was run, and
the suite was executed for real. The numbers below are from the forced run.

**Tests** — `make test` (= `go test -race -v ./...`)

```text
$ go clean -testcache && make test
go test -race -v ./...
...
PASS
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.452s
...
PASS
ok  	github.com/cachicamas/backend/agent/src/ai	3.219s
exit 0
```

- **336** top-level tests PASS (Wave 1 closed at 176 — Wave 2 added 160)
- **1 271** subtest PASS lines
- **0 FAIL, 0 SKIP**
- `test_output_hash`: `sha256:61d455db3fb59784385c032444af3c3bb474df1667063b19296a7dff27199a63`

**Lint** — `make lint`

```text
$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
exit 0
```

- `build_output_hash`: `sha256:66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a`
  — byte-identical to Wave 1's, which is the correct outcome for an unchanged clean-lint transcript.

**Module purity** — `backend/agent/go.mod`

```text
module github.com/cachicamas/backend/agent

go 1.26.3
```

Zero `require` lines. Re-confirmed by inspection and by the guard below. AI-19 introduced the wave's only
new stdlib dependency, `time` (for `Failure.RetryAfter`), which is inside the allowed set.

**AI-00 boundary guards, re-run individually**

```text
$ go test -race -count=1 -v -run 'TestLayer1_ModuleHasNoDependencies_ZeroRequires|
    TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault|
    TestRequestPath_DependencyClosure' ./src/ai/
--- PASS: TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage (0.03s)
--- PASS: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.05s)
--- PASS: TestLayer1_ModuleHasNoDependencies_ZeroRequires (0.05s)
ok  	github.com/cachicamas/backend/agent/src/ai	1.562s
```

Both AI-00 guards and AI-10.4's dependency-closure guard hold at wave end.

**AI-20.4 signature guard, re-run individually** (checklist item 3, last bullet)

```text
$ go test -race -count=1 -v ./src/agenttest/ -run 'Signature|Provider'
--- PASS: TestModelProviderInterface_TokenCounter_CleanAbsenceWhenTheProviderDoesNotAdvertiseIt
--- PASS: TestModelProviderInterface_MethodSet_ExternalStubImplementsCompilesAndIsExercised
--- PASS: TestModelProviderInterface_TokenCounter_DiscoveredWhenTheProviderValueSatisfiesIt
--- PASS: TestModelProviderInterface_MethodSet_ExactlyOneStreamMethod
--- PASS: TestModelProviderInterface_SignatureGuard
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.304s
```

---

## 3. Cross-milestone consistency — the wave's real integration risk

This section is the reason a wave-level verify exists. Each item was checked against landed code, not
against any milestone's self-report.

### 3.1 Shared block-index space — **CONFIRMED, and stronger than the spec asks**

AI-16's `R-ATE-004` binds AI-17 and AI-18 to one 1-based `BlockIndex` space with `0` as the rejected
sentinel. The wave does not merely satisfy this by convention; it satisfies it **at the type level**:

- Exactly one declaration exists: `type BlockIndex uint64`, `backend/agent/src/ai/event_descriptor.go:102`.
  `git log -S"type BlockIndex uint64"` returns exactly one commit, `297f08d` — **AI-14's own foundation
  commit**, not AI-16's. AI-14 declared the shared space before any content family existed, and its
  GoDoc names the binding explicitly: *"It is 1-based; 0 is the unset value … consistent with AI-16's
  R-ATE-003, so a stream mixing AI-14's own block-scoped kinds with AI-16's text blocks reads one
  indexing convention rather than two."*
- One unexported interface, `blockPayload { blockIndex() BlockIndex }` (`event_descriptor.go:114`), is
  implemented by exactly nine payloads: `TextBlockStart/TextDelta/TextBlockEnd`,
  `ReasoningBlockStart/ReasoningDelta/ReasoningBlockEnd`,
  `ToolCallStart/ToolCallDelta/ToolCallEnd`. There is no second index type, no per-family counter and no
  family tag anywhere in the package.
- The `0` sentinel is rejected in **two** places for all three families: at construction
  (`text_events.go:55,139,228`; `tool_call_event.go:96,192,275`; `reasoning_event.go:45`) and again at
  the emission boundary (`event.go:359`, `CheckEmit` rule 3, which reads the shared interface and never
  a concrete type).

Three independent spaces are not merely absent — they are unwritable without introducing a second type
that the checker would not read.

### 3.2 `eventRegistry` / `EventKind` exhaustiveness — **CONFIRMED, 12/12 covered**

`event.go`'s `eventRegistry` carries twelve rows, one per registered kind, each pairing a name with an
`EventDescriptor`:

| # | Constant | Registered name | Role | Cardinality | Terminal | Owner |
| --- | --- | --- | --- | --- | --- | --- |
| 1 | `EventKindResponseStart` | `responsestart` | none | at-most-one | false | AI-15 |
| 2 | `EventKindCompletion` | `completion` | none | at-most-one | **true** | AI-15 |
| 3 | `EventKindReasoningBlockStart` | `reasoningblockstart` | start | any | false | AI-17 |
| 4 | `EventKindReasoningDelta` | `reasoningdelta` | delta | any | false | AI-17 |
| 5 | `EventKindReasoningBlockEnd` | `reasoningblockend` | end | any | false | AI-17 |
| 6 | `EventKindTextBlockStart` | `text_block_start` | start | any | false | AI-16 |
| 7 | `EventKindTextDelta` | `text_delta` | delta | any | false | AI-16 |
| 8 | `EventKindTextBlockEnd` | `text_block_end` | end | any | false | AI-16 |
| 9 | `EventKindToolCallStart` | `tool_call_start` | start | any | false | AI-18 |
| 10 | `EventKindToolCallDelta` | `tool_call_delta` | delta | any | false | AI-18 |
| 11 | `EventKindToolCallEnd` | `tool_call_end` | end | any | false | AI-18 |
| 12 | `EventKindError` | `error` | none | at-most-one | **true** | AI-19 |

`eventKindEnd = EventKindError + 1`, so `EventKinds()` enumerates all twelve. The guard is genuinely
bidirectional and non-vacuous:

- `eventKindWitnesses` (`event_registry_test.go:62-136`) carries a witness for all twelve, and
  `TestEventKindRegistration_TheTestKindVocabulary_HasConstructorAndAccessor` fails in **both**
  directions — a kind with no witness, and a witness with no kind.
- `productionEventKinds` (`event_registry_test.go:225-236`) is a **hand-written** list, deliberately not
  obtained from the package (the file records why: *"a list the package supplied would agree with itself
  no matter what was added to it"*), pinned by
  `TestEventKinds_ProductionVocabulary_IsExactlyTheRegisteredKinds` on both length and order.
- Three anti-vacuity `t.Fatal` guards are present (`:156`, `:280`, `:316`).
- `TestEventRegistration_EveryRegisteredKind_HasADescriptorFromStatedDomains` additionally pins every
  descriptor's `Role` and `Cardinality` to the declared constant domains.

**A stronger result than the checklist asked for.** `R-AEE-015` claims a later milestone extends the
checker *without changing the checker's source*. That is provable from git, not just from reading:

```text
$ git log --oneline main..HEAD -- src/ai/stream_check.go src/ai/event_descriptor.go src/ai/sequence.go
65d8be7 feat(ai): close out AI-14 NFRs - totality, redaction, and a clean make lint (AI-14 NFR)
4de995a feat(ai): land the AI-14.4 ordering invariants checker (AI-14.4)
ed7ffa8 feat(ai): land the AI-14.2 per-stream sequence (AI-14.2)
78eb88a feat(ai): land the AI-14.1 event envelope skeleton (AI-14.1)
297f08d feat(ai): land the AI-14 ordering-descriptor skeleton (AI-14 foundation)
```

Five milestones registered twelve kinds and **not one commit after AI-14's own NFR close-out touched the
checker, the descriptor types or the sequence**. `grep -E "EventKind[A-Z]|ResponseStart|Completion|
TextBlock|Reasoning|ToolCall|Failure" stream_check.go` returns nothing. This is the single best result in
the wave.

### 3.3 `*Failure` on both delivery paths — **CONFIRMED**

- `*Failure` satisfies AI-14's sealed `eventPayload` directly, with no wrapper type:
  `func (f *Failure) kind() EventKind { return EventKindError }` (`provider_failure.go:386`) and
  `func (f *Failure) validate(at Path) *Violation` (`:504`). Both methods are unexported, so the seal
  holds — no type outside `package ai` can be a payload.
- **Pre-stream path**: `PreStreamFailure(r FailureReport) (*Failure, error)` (`:573`) returns the value
  directly as an `error` (`Error()` at `:349`, `Unwrap()` at `:359`, `Is()` at `:375`).
- **Mid-stream path**: `MidStreamFailure(r FailureReport, outputPreceded bool) (*Failure, error)` (`:585`)
  → `ErrorEvent` (`:594`) → `Event.ErrorPayload() (*Failure, bool)` (`:614`).
- Both doors funnel through one assembler, `newFailure` (`:542`), so the two paths share one rule set
  rather than two — `R-AIP-013`'s actual claim.
- `R-AIP-012`'s perpendicularity holds: `Delivery()` (`:491`) and `PartialOutput()` (`:478`) are separate
  one-line field reads; neither derives from the other, and neither consults the other's field.
- Terminal exclusivity is exercised at runtime — `provider_failure_test.go:143` drives
  `CheckStream([]Event{Stamp(failed), Stamp(after)})` and asserts the post-terminal rejection.

### 3.4 AI-20's `ModelProvider.Stream` signature — **CONFIRMED, mechanically pinned**

`backend/agent/src/ai/provider.go:96-100`:

```go
type ModelProvider interface {
	Stream(ctx context.Context, req Request) (<-chan Event, error)
}
```

AI-10/AI-12's `Request` in, AI-14's `Event` out, receive-only channel, no vendor or wire type. The file
imports **only** `"context"`. The guard in `src/agenttest/provider_signature_guard_test.go` is a real
`go/parser` walk over the on-disk `provider.go` and asserts, by AST rather than by reflection:

- exactly one method, named `Stream` (`assertExactlyOneStreamMethod`);
- parameters are exactly `[context.Context, Request]` (`assertParams`, `:213-217`);
- results are exactly `[<-chan Event, error]`, with `isRecvChanOf` checking `ChanType.Dir == ast.RECV`
  and the element identifier `Event` (`assertResults`, `:232`);
- every import is in `modelProviderAllowedImports = {"context"}` (`:129-141`) — so a vendor type could not
  be smuggled in behind a package alias.

`R-AMP-016`'s two bite mutations are recorded verbatim in `tasks.md:112,128` (`req Request` →
`req json.RawMessage`; `<-chan Event` → `<-chan string`), both produced guard-specific RED, both reverted,
and the revert was independently confirmed (`provider.go` imports only `"context"` today, no
`encoding/json` residue).

### 3.5 Vocabulary register integrity — **CONFIRMED, exactly one amendment**

`openspec/specs/ai-contract-vocabulary/spec.md` was touched by **exactly one commit** in the whole wave:
`75559c1 docs(openspec): append V-STR-24/V-STR-25 to the contract vocabulary (AI-15.1)`.

| Check | Result |
| --- | --- |
| Amended append-only | **Yes.** `V-STR-24` provider response identity, `V-STR-25` served model, both owned by AI-15, with a full amendment blockquote in § 4 citing § 9 rule 2 for why neither was covered by `V-STR-19` or `V-REQ-21`. |
| No existing row renumbered, reworded, reordered or removed | **Confirmed by diff** — the only non-additive hunks are the two count updates below. |
| Counts updated | **Yes.** `23 → 25` stream-side, `116 → 118` total; § 10 checklist row 2's range updated `V-STR-01 … V-STR-23` → `… V-STR-25`. Arithmetic re-checked: 29 + 25 + 12 + 17 + 18 + 17 = **118**. ✅ |
| Any other milestone silently touched the register | **No.** One commit, one author milestone. |
| Every `V-*` cited by Wave 2 artifacts resolves | **Yes.** 53 distinct identifiers cited across the seven changes; **zero** unresolved against the 118 defined rows. |
| Every `V-*` cited in Go source resolves | **Yes.** 82 distinct identifiers in `backend/agent/src/`; **zero** unresolved. |

This is exactly what checklist item 5 asked for, and it is clean.

### 3.6 Reconciliation gates — **HONORED, 5 of 5 spot-checked**

Most Wave 2 designs were written before AI-14's design and code landed, so each carried an explicit
reconciliation gate. I checked five:

| Milestone | Gate | Disposition |
| --- | --- | --- |
| **AI-15** | `design.md` § "AI-14 Reconciliation Gate — RESOLVED at tasks time" | **Honored.** It corrects its own earlier text in three named places (one combined `eventRegistry` table rather than two; `CheckStream(events []Event) StreamReport` named exactly; the missing `CheckEmit` step added to the data-flow diagram). I diffed its pinned Go spellings against the landed file: `NewResponseStart(responseID, servedModel string) (Event, error)` at `response_start.go:65`, `ResponseID()` at `:95`, `ServedModel()` at `:102`, `Event.ResponseStart() (ResponseStart, bool)` at `:79` — **verbatim match, no drift**. |
| **AI-16** | `design.md:15` — `R-ATE-009`'s `[provisional]` marker | **Resolved to KEPT.** `MaxTextLen` cap ships and is enforced at `text_events.go:145`. |
| **AI-17** | `design.md:121` — `R-ATE-009` cap decision | **Confirmed kept**, and `R-ARE-013`'s own provisional sentence resolved to KEPT: `reasoning_event.go:189` rejects a non-empty fragment on a redacted block with `ErrMisplaced` at `"fragment"`. |
| **AI-19** | `design.md:15` and closing checklist `:159` | **Resolved**, both marked *"no longer provisional"* with the exact `eventRegistry`/`eventRegistration`/`EventDescriptor` API pinned. `S-AIP-055` re-verification at the apply gate was additionally executed. |
| **AI-20** | `apply-progress.md` Phase 0.1 | **Honored, and the strongest of the five** — it re-verified not against AI-14/AI-19's *design* but directly against `event.go`, `provider_failure.go`, `sequence.go` and `request.go` **source**. |

**One gate failed at design time and was caught at apply — AI-18.** Its `design.md` asserted a bare,
never-erroring `NewXxxEvent(...) Event` constructor with validation deferred entirely to `CheckEmit`,
derived from reading AI-14's design *prose* rather than AI-15's landed code. Its own Open Questions
section had flagged the assumption as unverified. At apply time the agent read `response_start.go` and
`completion.go`, found both validate eagerly and return `(Event, error)`, and re-reconciled — updating
`tasks.md` in place rather than deviating silently. This is a **correct** outcome of a gate, not a defect;
it is the only case in the wave where a gate actually bit. It does, however, leave one loose thread —
see **W1**.

**No design.md still labels anything "provisional" or "TBD" that has since landed.** Every surviving
occurrence is either a resolved-and-recorded decision or a historical quotation. Two **spec** files do
still carry live provisional markers — see **W3**.

---

## 4. Per-milestone verdict

| Milestone | Change | Charter Goal / Deliverable | Test evidence | Verdict |
| --- | --- | --- | --- | --- |
| **AI-14** | `cachicamas-ai-event-envelope` | Met, and it carries the wave. Kind derived from payload and never stored (`Event` holds only `payload` + `seq`); sealed `eventPayload`; per-stream `Stamper` with a zero-value-ready 1-based counter; descriptor-driven `CheckStream`. C3 is fixed structurally, not shrunk. | `event_test.go` (344 L), `event_registry_test.go` (403 L), `event_descriptor_test.go` (160 L), `sequence_test.go` (200 L), `sequence_guard_test.go` (693 L), `stream_check_test.go` (411 L). 20/20 requirement IDs cited. | **PASS** |
| **AI-15** | `cachicamas-ai-response-events` | Met. Two independently registered kinds, `at-most-one` + `terminal` obtained *by registration alone* with no checker generalization and no delta on `ai-event-envelope`. AI-13's `FinishReason`/`Usage` embedded by value, so `TokenCount`'s presence bit crosses the event boundary untouched (`R-ARP-008` for free). | `response_start_test.go` (377 L), `completion_test.go` (544 L). `CheckStream` driven at 8 sites. | **PASS** |
| **AI-16** | `cachicamas-ai-text-events` | Met. Three kinds, producer-stamped 1-based index, fragment-only deltas, byte-exact reconstruction including a multi-byte rune split across a delta boundary, zero-delta blocks legal, `MaxTextLen` cap kept. Ships **no** public accumulator — verified by `grep` and by `TestTextEvents_ExportedSurface_ShipsNoAccumulatorOrReconstructor`. | `text_events_test.go` (890 L), 12 top-level tests. 11/11 requirement IDs cited; 29/32 scenarios. | **PASS** |
| **AI-17** | `cachicamas-ai-reasoning-events` | Met. Structurally distinct family (not a flag on text), token whole on block-end only, absent-vs-empty token distinguishable, redaction signalled once on block-start and carried forward on the delta payload's own copied bit. `reasoning_content.go`/`_test.go` (AI-07's) verified **diff-free** — NFR-ARE-A honored. | `reasoning_event_test.go` (1 472 L — the wave's largest). 14/14 requirement IDs, 40/43 scenarios. | **PASS** — with **S2** |
| **AI-18** | `cachicamas-ai-tool-call-events` | Met. Identity and name before any argument byte; fragment-only deltas; call-end carries exact bytes, **never re-marshalled and never JSON-validated** — confirmed: `tool_call_event.go` contains no reference to `isWellFormedJSON`, `json` or `Marshal`. Call ordinal is derived from stream position and stored nowhere (`:60-64`). | `tool_call_event_test.go` (1 316 L). 12/12 requirement IDs, 35/38 scenarios. Drives `CheckStream` at `:622,640`. | **PASS** |
| **AI-19** | `cachicamas-ai-provider-errors` | Met — the wave keystone holds. Nine closed categories with `failureCategoryLimit` last; retryability and retry-after independent and presence-typed; status class bounded 1–5; one concrete `*Failure` on both paths; `errors.Is` reaching per-category sentinels via `Is()` **before** `Unwrap()`, so the wrapped cause's own chain is not shadowed. | `provider_failure_test.go` (1 294 L) + `provider_failure_internal_test.go` (153 L). 15/15 requirement IDs, 47/55 scenarios — the wave's best scenario density. | **PASS** — with **W2** |
| **AI-20** | `cachicamas-ai-model-provider` | Met — the join point closes. One method, neutral in and out, AST-pinned. Eight AI-02 §9 ownership rules restated verbatim in GoDoc. `TokenCounter` advertised only by satisfying it; `ok == false` is a clean absence with no substitute or estimate. Mid-stream contract proven with a single-purpose `scriptProvider`, `-count=15` stress-clean. | `provider_test.go` (546 L), `agenttest/provider_test.go` (193 L), `agenttest/provider_signature_guard_test.go` (312 L). 19/21 requirement IDs; 15/66 scenarios cited — the wave's weakest, see **S5**. | **PASS** |

---

## 5. doc 0002 traceability (lines 851–1163)

All **25** charter nodes are represented: 23 `[leaf]` + 2 `[guard]` (AI-14.3 no process-global sequence
state, AI-20.4 signature guard) + 0 `[decision]`. Wave 2 charters no decision node, which is why no
milestone carries a `decision.md` — correct, not a gap.

**`[guard]` node AI-14.3 — closed, and it holds at wave scope.** `sequence_guard_test.go` is a
`go/parser` + `go/types` scan of the whole `ai` package: `Named → Underlying` via `types.Unalias`, struct
fields at any depth, array elements, integer basics plus `uintptr` plus `sync/atomic` types, no depth cap,
bare type parameters failing closed, and reset detection through `types.Info.Uses`. Its allowlist is
pinned to **exactly one** entry with a mandatory non-empty rationale naming C3:

```go
if len(sequenceStateAllowlist) != 1 { … }
if entry.file != "message.go" { … }
if entry.identifier != "lastMessageID" { … }
if !strings.Contains(entry.rationale, "C3") { … }
```

Because the guard runs against the landed package rather than a fixture, its passing at head is direct
evidence that **no Wave 2 milestone introduced package-level mutable sequence state**. That is a
wave-level property, proven mechanically.

**`[guard]` node AI-20.4 — closed** with two recorded, reverted bite mutations (§ 3.4).

**AI-19.6 was correctly not appended.** The charter's split trigger (line 222) fires only if
category-specific metadata grows the list past seven items. AI-19 landed nine categories with no
category-specific metadata field, re-checked at tasks time and recorded. No node was invented.

---

## 6. Findings

### CRITICAL

**C1 — 26 OpenSpec artifact files, 3 595 lines, are untracked and absent from the deliverable.**

`git status --short` at head reports eleven untracked paths, expanding to 26 files:

| Milestone | Untracked artifacts | Lines |
| --- | --- | --- |
| AI-16 `cachicamas-ai-text-events` | **entire folder** — proposal, explore, design, `specs/ai-text-events/spec.md`, tasks, apply-progress | 703 |
| AI-17 `cachicamas-ai-reasoning-events` | **entire folder** — same six artifacts | 750 |
| AI-18 `cachicamas-ai-tool-call-events` | **entire folder** — same six artifacts | 706 |
| AI-19 `cachicamas-ai-provider-errors` | proposal, explore, design, `specs/ai-provider-errors/spec.md` | 719 |
| AI-20 `cachicamas-ai-model-provider` | proposal, explore, design, `specs/ai-model-provider/spec.md` | 717 |
| | **Total** | **3 595** |

The consequences are concrete, not procedural:

1. **The PR contains no specification for five of seven milestones.** A reviewer opening this PR sees
   12 236 lines of Go across 31 files and a spec for AI-14 and AI-15 only. `text_events.go`,
   `reasoning_event.go`, `tool_call_event.go`, `provider_failure.go` and `provider.go` — 1 743 production
   lines — arrive with no reviewable requirement.
2. **`sdd-archive` cannot run.** It promotes `openspec/changes/<name>/specs/<capability>/spec.md` into
   `openspec/specs/<capability>/spec.md`. Five of those source files are not in the repository. The
   capabilities `ai-text-events`, `ai-reasoning-events`, `ai-tool-call-events`, `ai-provider-errors` and
   `ai-model-provider` would be promoted from nothing.
3. **The loss is one command away.** A `git checkout`, `git clean -fd`, branch switch or fresh clone
   destroys all 3 595 lines, including the definitions of 73 of the wave's 104 requirements. Nothing in
   the repository would record that they ever existed.

These files are **not gitignored** — `git ls-files --others --exclude-standard` lists every one of them,
so this is unstaged work, not deliberate exclusion. The remedy is a single `git add openspec/changes/`
plus a commit. **No code change is required and no implementation defect is implied.**

Classified CRITICAL rather than WARNING because, unlike Wave 1's archive-owned findings (which were prose
corrections to artifacts that existed), this one means the artifacts are absent from version control
entirely. **Owner: whoever prepares the PR, before `sdd-archive`.**

### WARNING

**W1 — `CheckEmit`'s rule 4 was deferred by AI-14 to "AI-15+" and the wave ended without discharging it.**

`event.go:364-366` is `CheckEmit`'s fourth and last rule:

```go
func() *Violation {
    return e.payload.validate(Path{At("event")})
},
```

AI-14's `tasks.md` disclosed the gap honestly at the time: *"Rule 4 … is implemented … but has no
dedicated failure-path unit test at this milestone: the only payload type AI-14 can construct,
`WitnessPayload`, has a `validate()` hardcoded to return `nil`. … AI-15+ (whose real payloads carry
non-trivial `validate()` rules) will be the first to exercise this path."*

That never happened, and the reason is exactly the design divergence § 3.6 records. AI-15 through AI-19
all adopted **eager constructor validation returning `(Event, error)`** — the model AI-18's design gate
discovered — so an `Event` carrying a payload that fails its own `validate` is unconstructible through
the public API. Meanwhile `export_test.go:68` still reads
`func (w WitnessPayload) validate(_ Path) *Violation { return nil }`, and `NewTestEvent`
(`export_test.go:117`) always builds a `WitnessPayload`. I grepped every `CheckEmit` call site across all
test files: every one exercises rule 1 (unregistered/zero kind), rule 2 (unstamped sentinel) or rule 3
(zero block index). **Not one drives rule 4 to a violation.**

Not blocking: rule 4 is defensive depth behind a constructor that already validated, and no requirement
claims a runtime rejection through it. But the branch is untested at wave close, and the mechanism that
was supposed to close it — a later milestone with a failing payload — was designed away without anyone
noticing the deferral had lost its receiver. **Owner: Wave 3**, either by giving `WitnessPayload` a
controllable failure mode or by recording rule 4 as deliberately unreachable defence.

**W2 — `*Failure` is the only Wave 2 payload without `GoString()`, and `%#v` reproduces two
provider-supplied fields.**

Every other new payload type carries an explicit `GoString()` returning a constant — `TextBlockStart`,
`TextDelta`, `TextBlockEnd`, `ReasoningBlockStart`, `ReasoningDelta`, `ReasoningBlockEnd`,
`ToolCallStart`, `ToolCallDelta`, `ToolCallEnd`, `ResponseStart`, `Completion`, and `Event` itself.
`event.go:314-319` states why: *"Without it, `%#v` falls back to reflection and prints every field, which
would make the redaction posture a property of which verb someone reached for rather than a property of
the type."*

`*Failure` has `Error()` but no `GoString()`. Reproduced with a throwaway external test (written, run,
deleted; working tree confirmed clean afterwards):

```text
verb %v    leak=false  out=provider failure: authentication
verb %s    leak=false  out=provider failure: authentication
verb %+v   leak=false  out=provider failure: authentication
verb %#v   leak=true   out=&ai.Failure{category:0x1, retryable:false,
             retryAfter:ai.RetryDelay{delay:0, present:false},
             rawLabel:"RAWLABEL-CACHICAMA-SENTINEL", statusClass:0,
             requestID:"REQID-CACHICAMA-SENTINEL",
             cause:(*errors.errorString)(0x…), delivery:0x1, partialOutput:false}
```

Scope, stated precisely: `rawLabel` and `requestID` leak verbatim. The **wrapped cause's text does not** —
it renders as a pointer address — so a credential or response body inside the cause stays contained, and
`R-AIP-009`'s literal claim ("useful without a credential or a response body") is not violated.
`rawLabel` and `requestID` are both bounded and sanitized by `sanitizeOpaqueField` before storage, which
limits the blast radius further. The envelope is also safe: `Event.String()`/`GoString()` render only kind
and sequence, so a consumer must deliberately reach past `ErrorPayload()` and choose `%#v`.

It is nonetheless a real inconsistency with the wave's own stated posture and with Wave 1's four-verb
convention (`TestX_Formatting_..._ThroughAnyVerb`). **Owner: Wave 3.** Fix is one method plus one
four-verb canary test.

**W3 — Two specs still carry live `[provisional]` markers for rules their design resolved and their code
implemented.**

- `cachicamas-ai-text-events/specs/ai-text-events/spec.md:120` — the requirement **title** reads
  `### R-ATE-009 — A fragment is bounded by the existing text ceiling *(provisional)*`, and `:122` says
  *"This requirement is `[provisional]`: `design.md` MAY drop or replace it … if it is dropped,
  `S-ATE-022` is struck."* The spec's Open Items list repeats it at `:197`.
- `cachicamas-ai-reasoning-events/specs/ai-reasoning-events/spec.md:169` — `R-ARE-013`'s last sentence
  carries the same construction, repeated at `:234`.

Both were resolved to **KEPT** in their `design.md` (AI-16 `design.md:15`, AI-17 `design.md:12`) and both
are implemented and tested: the `MaxTextLen` cap at `text_events.go:145`, the redacted-block rejection at
`reasoning_event.go:189`. `sdd-archive` merges these deltas into `openspec/specs/`, so as written a frozen
main spec would tell a future reader that a landed, tested rule is still an open option and that a passing
scenario may yet be struck. **Owner: `sdd-archive`**, before merging — same class as Wave 1's W2.

**W4 — doc 0002 is seven milestones stale and Wave 2 touched no docs.**

`git log main..HEAD -- docs/` is empty. Line 3 still reads:

> **Status:** Wave 0 + Wave 1 complete — **14 of 41** milestones shipped. **AI-00 through AI-13 are
> landed and verified.** The `backend/agent` module exists at `backend/agent/` with 19 production files
> and 27 test files.

After Wave 2 that should read **21 of 41** (AI-00 … AI-20), with **33** production files and **44** test
files. The Layer 1 completion checklist and the traceability spine also have Wave-2-closable rows the
archive owns — notably the spine rows the Wave 1 report left open pending stream work (**G5**'s tool-call
ordinal, now Layer-1-closed by AI-18.3; **G10**/**G12(c)**, now backed by AI-15's completion event), and
doc 0001 § 9's "streams and concurrency" checklist block, which Wave 1 correctly marked five-way N/A and
which Wave 2 now makes applicable for the first time. **Owner: `sdd-archive`** — the same debt Wave 1
raised as its own W4, now compounding.

**W5 — AI-14 has no `apply-progress.md`; it is the only one of the seven without it.**

`openspec/changes/cachicamas-ai-event-envelope/` contains proposal, explore, design, spec and tasks — no
apply-progress. The *substance* is present and unusually good: `tasks.md` carries a full phase-by-phase
evidence log with verbatim RED transcripts, a lint-gap discovery with its fix, and **two** honest
deviation disclosures (the rule-4 gap that became **W1**, and `R-AEE-014`'s missing dedicated scenario,
tested directly against requirement prose instead). But the artifact the pipeline expects is absent, and
a consolidated wave-level deviations register (§ 7) has to reach into a different file for one of seven
milestones. **Owner: `sdd-archive`** — either synthesize the artifact from `tasks.md` or record the
substitution.

**W6 — AI-15's `design.md` asserts an archive that never happened.**

`cachicamas-ai-response-events/design.md` closes its reconciliation gate with: *"`sdd-apply` remains gated
behind AI-14's merge, which is satisfied (AI-14 archived, `openspec/specs/ai-event-envelope/` promoted per
doc 0002 milestone tracking)."*

`openspec/specs/ai-event-envelope/` **does not exist**; `ls openspec/specs/` shows 25 capabilities, none of
them Wave 2's. AI-14 was never archived. AI-15's own `apply-progress.md` deviation #2 states this
correctly — *"AI-14's code is merged and correct, but its OpenSpec artifact has not been
archived/promoted … flagged for a later archive pass"* — so the milestone knew, and the design.md text is
simply stale relative to its own apply record. Since designs are archived alongside specs, the false claim
would freeze. **Owner: `sdd-archive`.**

### SUGGESTION

**S1 — Registered kind names mix two naming conventions.**
`responsestart`, `completion`, `reasoningblockstart`, `reasoningdelta`, `reasoningblockend`, `error` are
unseparated; `text_block_start`, `text_delta`, `text_block_end`, `tool_call_start`, `tool_call_delta`,
`tool_call_end` are snake_case. These strings are `EventKind.String()`'s output — a public rendering
surface that will appear in logs and diagnostics above Layer 1 — and they split 6/6 by which agent landed
first. Nothing depends on the format today; unifying later means changing a rendered value consumers may
already match on. Cheapest to fix now.

**S2 — AI-17 is the only block family whose tests never drive `CheckStream`.**
AI-16 exercises the checker's block-ordering path at `text_events_test.go:672,707,872`; AI-18 at
`tool_call_event_test.go:622,640`; AI-19 at `provider_failure_test.go:143`; AI-15 at eight sites.
`reasoning_event_test.go` calls only `CheckEmit`. The descriptors are pinned by
`TestEventRegistration_EveryRegisteredKind_HasADescriptorFromStatedDomains` and the checker is
demonstrably payload-independent, so reasoning block ordering works *by construction* — but there is no
runtime evidence for it, and "by construction" is the argument that failed in Wave 1's § 5.2. One
`CheckStream` table over a reasoning start/delta/end sequence would close it.

**S3 — No test drives one realistic full stream end to end.**
Cross-family evidence is pairwise at best: `reasoning_event_test.go:707` proves a reasoning block and a
text block hold distinct indices; `text_events_test.go:253` proves a text block and a non-text block
separate by index alone. Nothing drives `responsestart → text block → reasoning block → tool-call block →
completion` (and the error-terminated variant) through `CheckStream` as one recorded stream. That
composite is the wave's actual deliverable and AI-20 is its natural owner. A single integration test in
`src/agenttest` would also give AI-20 the scenario density it currently lacks (**S5**).

**S4 — Text-event payloads have no dedicated rendering canary.**
`ToolCallEvents`, `Reasoning`, `Completion` and `ResponseStart` each carry a planted-sentinel rendering
test; `TextBlockStart`/`TextDelta`/`TextBlockEnd` do not, despite `TextDelta` carrying raw model output.
No leak is possible today — all three `String()` methods return literal constants (`text_events.go:104,
210,278`) — so this is a missing *proof*, not a defect. But the whole point of the wave's canary
convention is that the proof survives a future refactor that the constant does not.

**S5 — Three requirements have no ID citation, and one of them has no mechanical guard at all.**
`R-ARP-011` (two nouns appended to the register) and `R-AMP-016` (guard proven to bite) are both
document-level and are both verified by evidence I re-checked directly — the register diff in § 3.5 and
the reverted bite mutations in § 3.4. Neither is a gap.

`R-AMP-004` is different. It requires the interface documentation to carry AI-02 §9's eight ownership
statements plus AI-03 §9's enumerability clause. All nine are present in `provider.go:34-95` — I read them
against the source specs. But the only evidence they are present is that an agent looked. Delete rule 6
("The stream's buffer is bounded. Backpressure means waiting, never dropping") and **nothing fails**. The
wave already contains the exact tool for this: `sequence_guard_test.go` and
`TestSequenceGoFile_PackageDoc_StatesTheCrossStreamRule` both parse Go doc comments and assert on their
content. An eight-substring assertion over `provider.go`'s `ModelProvider` GoDoc would make `R-AMP-004`
mechanical for roughly fifteen lines of test.

This is also why AI-20's scenario citation rate is the wave's lowest at **15/66 (23 %)** against a wave
average of 69 % — much of `R-AMP-004`, `R-AMP-009` … `R-AMP-012` is proven by prose plus a stub provider
rather than by scenario-tagged assertions.

---

## 7. Consolidated deviations register (checklist item 8)

Every "Deviation" section across all seven milestones, collected and cross-checked as a set for the first
time. Each was individually disclosed; none was hidden.

| # | Milestone | Deviation | Class | Cross-check verdict |
| --- | --- | --- | --- | --- |
| D1 | AI-14 | `CheckEmit` rule 4 has no failure-path test; deferred to "AI-15+" | Coverage gap | **Never discharged** — escalated to **W1**. The receiving milestones designed the receiver away. |
| D2 | AI-14 | `R-AEE-014`'s "1-based, 0 rejected" has no dedicated `S-AEE` scenario; tested against requirement prose instead | Traceability | Benign. `TestCheckEmit_BlockScopedEventWithZeroBlockIndex_RejectedWithErrOutOfRange` exists and passes. |
| D3 | AI-14 | `R-AEE-009`'s cross-stream doc paragraph landed in `sequence.go`'s header, not `doc.go` as `tasks.md` said | File placement | Benign — `design.md` specified `sequence.go`; design is authoritative for placement. Test reads it via `file.Comments[0]`. |
| D4 | AI-14 | Four pre-existing `revive` `package-comments` findings fixed by adding a blank line before `package ai` in four files; two existing doc-reading tests repaired to use `file.Comments[0]` | Lint | Benign, and the fix propagated: AI-17's D12 is the same finding on a sibling file. |
| D5 | AI-15 | `event_registry_test.go` generalized (`construct` → `func() (Event, error)`, `read` → `func(Event) (any, bool)`), not listed in `design.md`'s File Changes table | Scope | Benign and **necessary** — mirrors `content_part_registry_test.go`'s established per-kind precedent. Without it no production kind could join the witness table. |
| D6 | AI-15 | `TestEventKinds_ProductionVocabulary_IsEmpty` replaced by a hand-pinned equivalent | Test change | Correct — the original assertion became false by design at the first registration. The replacement is *stronger* (order + length, hand-written list). |
| D7 | AI-15 | Engram topic-key `sdd/cachicamas-ai-response-events/apply-progress` upserted onto observation `#2173`, previously held by a pre-restart milestone of the same change name | Bookkeeping | Benign — correct `topic_key` semantics; the historical record at `#2181` is untouched. |
| D8 | AI-15 | `design.md` claims AI-14 archived and `openspec/specs/ai-event-envelope/` promoted | **Factually false** | Escalated to **W6**. Contradicted by AI-15's own apply-progress deviation #2. |
| D9 | AI-16 | None from design. One clarification: `design.md` cites `text_content_test.go` for a `MaxTextLen` idiom; no such file exists, `content_part_test.go` used instead | Doc | Benign, recorded in `tasks.md` 2.9. |
| D10 | AI-16 | `make lint` reported 1 issue at close — a `revive` finding in **AI-17's** `reasoning_event.go` — left untouched per scope instruction | Cross-milestone | **Resolved.** AI-17's D12 fixed it (`4ceb77c`). Verified: `make lint` is `0 issues` at head. The prediction was correct. |
| D11 | AI-16 / AI-17 / AI-18 | Three agents edited `event.go` and `event_registry_test.go` concurrently in one worktree; AI-16's commit `4f63977` incidentally captured all nine registry entries | Process | **Verified benign.** All twelve rows present and correct (§ 3.2); AI-18's own commit `20a483d` scoped to its two exclusive files via explicit `git add <path>`. No sibling content lost. |
| D12 | AI-17 | Fixed the pre-existing `revive` `package-comments` gap in `reasoning_event.go` (blank line), outside original scope | Lint | Benign, zero behaviour change, `make lint` 1 → 0. Closes D10. |
| D13 | AI-18 | `design.md`'s "reconciled construction model" was **wrong** — asserted a bare `NewXxxEvent(...) Event` with validation only in `CheckEmit`; AI-15's landed code validates eagerly and returns `(Event, error)` | **Design error, caught at apply** | **Correctly handled.** Re-reconciled against source, `tasks.md` updated in place. This is the gate working. Its side effect is **W1**. |
| D14 | AI-19 | `design.md` step 6 says "doc.go kind list"; landed recipe puts it in `event.go`'s `EventKind` GoDoc | File placement | Benign — followed landed-code precedent, which `S-AIP-055` explicitly authorises. |
| D15 | AI-19 | `Failure.Category()` had no owning task anywhere in `tasks.md` | Planning gap | Benign — a one-line field accessor, needed by `R-AIP-003`'s own phase. Real gap in the task graph, honestly named. |
| D16 | AI-19 | `Failure.validate` was a no-op stub through Phase 2, gaining rules in Phases 3 and 4 | Staging | Benign — Go compile order forces the callee first. Resolved by staging, not by resequencing. |
| D17 | AI-19 | `RawLabel`/`RequestID` stored unsanitized in Phase 2 before `sanitizeOpaqueField` existed | Staging | Benign — no accessor existed yet; wired in the same Phase 3 edit. |
| D18 | AI-19 | Pre-existing `gofmt` drift in `completion_test.go` (~line 524) left untouched per Safety Net rule | Hygiene | Benign — confirmed pre-existing, and `make lint` is clean at head. |
| D19 | AI-19 | **Genuine strict-TDD ordering slip**: the status-class 0..5 bound was written before its test | **TDD violation** | **Best remediation in the wave.** Test added, then *actually verified load-bearing* by deleting the bound, re-running to confirm FAIL, restoring, confirming PASS. That is stronger evidence than an ordinary first-time RED. |
| D20 | AI-19 | `Is()` (task 6.2) and per-category sentinels (task 6.4) implemented in one cycle rather than two | Batching | Benign — `Is()` has nothing to compare against until the sentinels exist. |
| D21 | AI-20 | **Two strict-TDD violations**: `scriptProvider`'s struct + pre-stream branch authored with its tests; `resolveAndParseGoFile` extracted with its tests | **TDD violation** | Confined to test-fixture/helper code, never `src/ai` production logic. Self-assessment is unusually candid: *"Rationale offered at the time … is true but does not excuse it — a stub `Stream` returning `panic(\"not implemented\")` first … was not done."* Both restated in the phase sections and in the TDD table rather than buried. |

**Set-level reading.** Twenty-one disclosures, and the cross-check changes the verdict on three of them:

- **D1 is the only one that was never discharged.** It is also the only one whose owner disappeared — D13
  removed the construction model that D1's deferral depended on, and no one connected the two. This is
  precisely the failure a wave-level verify exists to catch: two individually sound decisions that
  invalidate each other.
- **D8 is factually false** and is contradicted by its own milestone's apply record.
- **D10 correctly predicted its own resolution by a sibling milestone**, and the prediction held.

Everything else is benign, and the *pattern* is healthy: three TDD violations (D19, D21 ×2) all
volunteered rather than concealed, one of them (D19) remediated with a mutation proof stronger than the
process it violated. Nine of the twenty-one are file-placement or staging notes that record a
design/tasks/reality mismatch instead of silently following one of them.

---

## 8. Review workload and diff size (checklist item 7)

```text
$ git diff --stat main...HEAD
 47 files changed, 14326 insertions(+), 2 deletions(-)
```

| Area | Files | Lines |
| --- | --- | --- |
| `backend/` (tracked) | 31 | 12 236 |
| `openspec/` (tracked) | 16 | 2 090 (+2 −2 in the vocabulary register) |
| `openspec/` (**untracked**, C1) | 26 | 3 595 |
| **True wave total** | **73** | **~17 921** |

The 2 deletions are both in `openspec/specs/ai-contract-vocabulary/spec.md` — the two count lines AI-15.1
replaced. Nothing else in the repository was deleted or modified; the wave is otherwise purely additive.

**Against the accepted `size:exception`.** The prompt's ~5 000-line budget is exceeded by roughly 3.6×
counting the tracked diff alone, and 3.6× again if the untracked artifacts are staged. Every milestone
recorded the exception explicitly:

| Milestone | Forecast (upper) | `400-line budget risk` | Actual authored code |
| --- | --- | --- | --- |
| AI-14 | ~3 400 | High | 2 903 |
| AI-15 | ~1 300 | High | 1 143 |
| AI-16 | ~320 | **Low** | **1 171** |
| AI-17 | ~1 300 | High | 1 847 |
| AI-18 | ~1 000 | High | 1 652 |
| AI-19 | ~2 200 | High | 2 064 |
| AI-20 | ~900 | High | 1 456 |
| **Sum** | **~10 420** | | **12 236** |

All seven `tasks.md` files carry the **exact literal guard lines** `Decision needed before apply: No`,
`Chained PRs recommended: No`, `400-line budget risk: …` — 3/3 in every file. Wave 1's suggestion S4 was
adopted uniformly. Two observations:

- Five of seven forecasts were within ±30 % of actual; **AI-16 was off by 3.7×** and is the only
  milestone that declared `400-line budget risk: Low`. It forecast ~260–320 lines and landed 1 171
  (`text_events.go` 281 + `text_events_test.go` 890). Its own apply-progress claims the result was
  *"within the forecast's ~260–320 estimate for this milestone's own authored lines"*, which is not true
  as written. The forecast that most needed to be right — the only one claiming the budget was safe — was
  the one that was wrong. Not blocking (the wave-level exception covers it), but the estimate should be
  corrected before it is cited as precedent.
- Aggregate is **17 % over the sum of upper forecasts**, which for a 12 000-line wave is a genuinely good
  forecasting record.

For scale: Wave 1 shipped 33 005 insertions across 100 files under the same exception. Wave 2 is
substantially smaller.

---

## 9. Strict TDD compliance

| Check | Result | Details |
| --- | --- | --- |
| TDD evidence reported | ✅ | Six of seven carry `apply-progress.md` with RED/GREEN tables; AI-14 carries the same substance in `tasks.md` (**W5**). |
| All tasks have tests | ✅ | 223/223 boxes complete; every one of doc 0002's 25 Wave 2 nodes maps to a landed spec section and named passing tests. |
| RED confirmed | ✅ | 17 new test files, all present, compiled, executed. RED transcripts are overwhelmingly genuine compile failures (`undefined: ai.EventKindToolCallStart`, `undefined: ai.ModelProvider`, `undefined: ai.CheckStream`, `f.RawLabel undefined`, `c.Validate undefined`) — the hardest kind to fabricate. |
| GREEN confirmed | ✅ | 336/336 top-level tests pass under `-race`, re-run by this verification with a cleared test cache. |
| Transcripts observed, not reconstructed | ✅ | AI-14's Phase 5 records *"PASS on the first implementation attempt — no iteration needed for correctness, only one `gofmt -w` pass"* — an admission that lowers the apparent heroism of the record and therefore raises its credibility. AI-17 records 2 genuine FAILs before GREEN, by name. AI-18 records a self-inflicted test bug found during GREEN. |
| Triangulation adequate | ✅ | Table-driven throughout. AI-19 sweeps all 256 `FailureCategory` values against `Validate()` and cross-checks 9×9 category/sentinel pairs. AI-16 splits a multi-byte rune across a delta boundary and asserts no replacement character. AI-17 proves a token-only block and a redacted block with *identical token bytes* remain distinguishable. |
| Bite proofs for guards | ✅ | AI-20.4: two mutations (vendor stand-in, changed carrier), both recorded verbatim, both reverted, revert independently confirmed. AI-20.5: one method-set-widening mutation. AI-14.3: the guard's own scratch-package tests are permanent bites (`TestSequenceGuard_ScratchPackageLevelInteger_Fails` and four siblings). |
| Ordering violations | ⚠️ **3, all disclosed** | D19 (AI-19, status-class bound) and D21 ×2 (AI-20, both test-fixture code). None touches `src/ai` production logic except D19, which was remediated with a delete/re-run/restore mutation proof. |

**Assertion quality audit** — ✅ **no tautologies, no vacuous passes, no orphan assertions found.**

- Anti-vacuity `t.Fatal` guards are present in every exhaustiveness assertion:
  `event_registry_test.go:156,280,316`, plus the `len(sequenceStateAllowlist) != 1` pin at
  `sequence_guard_test.go:145`.
- `productionEventKinds` is deliberately hand-written rather than package-supplied, with the reason stated
  in source — the strongest form of this pattern in the codebase.
- `TestCheckStream_Source_NamesNoConcretePayloadTypeOrKindConstant` asserts a *negative structural*
  property over the checker's own source, which is what makes `R-AEE-015` a guarantee instead of a habit.
- AI-19's rendering test plants a credential/body sentinel and asserts absence, rather than asserting the
  message "looks safe".
- `TestToolCallEvents_...` at `:262` asserts that no exported method *reads like* a call-scoped counter —
  guarding the shape of a future mistake, not only the current state.

**Test layer distribution.** Unit: 328 tests in `src/ai`. Cross-package: 8 in `src/agenttest`, including
the AST signature guard and the external `stubProvider` conformance proof. E2E: none, correctly — Layer 1
has no runnable surface. Coverage tooling is still not configured in the `Makefile`; coverage analysis
skipped, which is not a failure.

---

## 10. doc 0001 § 9 review checklist, run against Wave 2

**Boundaries**

- [x] No package of `backend/agent` imports another backend module — guarded and passing.
- [x] Layer 1 imports only stdlib — zero non-stdlib dependencies, zero `require` lines. AI-19 added `time`;
      inside the allowed set and invisible to the closure guard's forbidden list.
- [ ] **N/A** — Layer 2 I/O, Layer 3 HTTP. Neither exists.
- [x] Both import guards run and pass.

**Contracts**

- [x] No vendor wire type crosses the Layer 1 method boundary — now a *mechanical* guarantee at the one
      place it matters, `ModelProvider.Stream`, via the AST guard and the `{"context"}` import allowlist.
- [x] **Every event kind is constructible** — Wave 1 marked this N/A because no event existed. It is now
      live and closed: twelve kinds, twelve witnesses, bidirectional exhaustiveness assertion.
- [x] Anything a provider must receive back byte-identical is carried opaquely — reasoning round-trip
      tokens on block-end (`R-ARE-009/010`), tool-call argument bytes never re-marshalled and never
      JSON-validated (`R-ATC-006/007`), provider response identity byte-exact (`V-STR-24`).
- [x] A new neutral field is genuinely neutral — `FailureCategoryUnknown` preserves the raw provider label
      bounded and sanitized rather than widening the vocabulary.
- [x] **Tool-call deltas remain optional** — Wave 1 marked N/A. Now closed: `R-ATC-009`, a call with zero
      deltas is legal and complete, plus `R-ATC-010`, whole and fragmented delivery indistinguishable
      after reconstruction.

**Streams and concurrency** — applicable for the first time.

- [x] Single producer, single closing site, closed exactly once — `R-AMP-009`, stated in `ModelProvider`'s
      GoDoc rules 1–8 and proven by AI-20's `scriptProvider`.
- [x] Every send selects on cancellation — `R-AMP-010`, `select{ out <- ev; <-ctx.Done() }`, `-count=15`
      stress-clean.
- [x] Bounded buffer, backpressure waits rather than drops — `R-AMP-012`, with exactly one sanctioned loss
      path and the contract naming who is in error.
- [x] Sequence state belongs to the stream, not the process — C3 fixed structurally; the package-wide
      `go/types` guard passes at wave end with a one-entry allowlist.
- [x] Run under the race detector — `make test` is `go test -race -v ./...`; 336 tests, 0 races.

**Observability and safety**

- [x] No rendering carries prompt, completion, reasoning, tool-argument or tool-result text — `Event.String`
      reads only `Kind()` and never the payload; all eleven new payload types return literal constants from
      `String()`/`GoString()`; `Failure.Error()` is a fixed prefix plus the category name. **One exception**:
      `*Failure` under `%#v` — see **W2**.
- [x] Errors are typed and inspectable — AI-04's five sentinels reused throughout with **no new sentinel
      added** (AI-14 verified `grep errors.New` returns zero across its files); AI-19 adds per-category
      sentinels reachable through `errors.Is` with `Unwrap` preserving the cause's own chain.
- [x] Mid-stream partial output is distinguishable — `R-AIP-010/011`, perpendicular to delivery path, and
      "is a naive retry safe?" answerable from the discriminator alone.
- [ ] **N/A** — process-group kill.

**Process**

- [x] The change fits the review budget, or says why it does not — it does not, and all seven say so with
      the exact literal guard lines. One forecast was materially wrong (§ 8).
- [x] Milestone identifiers appended, never renumbered — doc 0002 untouched; AI-19.6 correctly not
      appended; requirement prefixes collision-free (§ 1).
- [x] Anything deliberately left unsupported is stated — no public accumulator or reconstructor
      (`R-ATE-011`, `R-ARE-008`), no JSON validation at call-end (`R-ATC-007`, deferred to AI-30), no
      argument-length ceiling (AI-18 `design.md:19`, with the reasoning for not inventing one).

---

## 11. Verdict

**FAIL — blocked on one delivery precondition, with every implementation dimension clean.**

Wave 2 delivers all seven milestones against doc 0002 lines 851–1163 with no unmet charter obligation, no
untested requirement, no open task and no failing check. The evidence gate is clean and was re-run rather
than trusted: 336 tests pass under `-race` after a cleared test cache, `golangci-lint` reports 0 issues,
`go.mod` carries zero `require` lines, and both AI-00 import guards plus AI-10.4's dependency-closure guard
hold.

The three cross-milestone risks the wave actually carried all resolve favourably, and two resolve more
strongly than their specs demand. The block-index space is shared at the **type level** — one
`BlockIndex` declared by AI-14 before any content family existed, one unexported interface, nine
implementors, no second index type anywhere. The exhaustiveness guard covers all twelve registered kinds
bidirectionally against a deliberately hand-written list. And `R-AEE-015`'s extensibility claim is
provable from git rather than from prose: five milestones registered twelve kinds without a single commit
touching `stream_check.go`, `event_descriptor.go` or `sequence.go` after AI-14's own close-out. AI-19's
`*Failure` implements the sealed payload interface directly and is reachable on both delivery paths
through one shared assembler. AI-20's signature is pinned by a real AST walk with an import allowlist, and
its two bite mutations are recorded and confirmed reverted.

The vocabulary register was amended by exactly one commit, append-only, with correct arithmetic; all 53
`V-*` terms the artifacts cite and all 82 the source cites resolve. Requirement prefixes are
collision-free — a direct improvement on Wave 1, which shipped a live collision into verification.
Scenario traceability rose from 27 % to 69 % and requirement traceability stands at 97 %, so Wave 1's S1
was genuinely adopted, as were its S4 guard lines.

This report nonetheless returns **FAIL**, and the reason is narrow and mechanical rather than a judgement
on the engineering. A verdict is only as good as the artifact it can be checked against, and for five of
seven milestones that artifact is not in the repository. One delivery defect and one integration debt
account for the whole gap. **C1**: five of
seven milestones' specifications are untracked, which means the PR is unreviewable against its own
requirements and `sdd-archive` has nothing to promote for five capabilities — 3 595 lines that one
`git checkout` would destroy without trace. **W1**: AI-14 deferred `CheckEmit`'s rule-4 coverage to
"AI-15+", and AI-18's design gate — working correctly — replaced the construction model that deferral
depended on, so the receiver quietly ceased to exist. Neither was visible from inside a single milestone;
both are exactly what a wave-level verify is for.

The remaining warnings are one code-posture inconsistency (**W2**, `*Failure` leaking two sanitized
fields under `%#v`, reproduced) and four bookkeeping obligations the archive owns (**W3** live
`[provisional]` markers on landed rules, **W4** doc 0002 seven milestones stale, **W5** AI-14's missing
apply-progress, **W6** AI-15's design asserting an archive that never happened).

The twenty-one disclosed deviations, cross-checked as a set for the first time, hold up well: three TDD
violations all volunteered rather than concealed, one remediated with a mutation proof stronger than the
process it broke; nine file-placement or staging notes that record a mismatch rather than silently
resolving it; one prediction about a sibling milestone's lint gap that came true. Only D1 was never
discharged and only D8 is false.

**Next**: stage and commit the 26 untracked artifacts (**C1**), then re-validate this report with
`blockers: 0` — no code change, no re-run of `make test` or `make lint`, since neither depends on staging
state. After that, `sdd-archive`, carrying **W3**, **W4**, **W5** and **W6** as its own work items.
**W1** and **W2** belong to Wave 3.
