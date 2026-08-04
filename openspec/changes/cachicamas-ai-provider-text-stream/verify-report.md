```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:b20bbdc025d9d63ce726ce3ffdc71581f99ecad61764e6835d5d17d45a4d0046
verdict: fail
blockers: 1
critical_findings: 1
requirements: 28/28
scenarios: 104/107
test_command: go test -race -count=1 ./...
test_exit_code: 0
test_output_hash: sha256:ec7ec37640c9bd2cfb4a2198f83bb2bb2d06face239530eb9ce4a92e93c7b626
build_command: go build ./...
build_exit_code: 0
build_output_hash: sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855
```

## Verification Report

**Change**: `cachicamas-ai-provider-text-stream` (AI-28 — all 7 slices + milestone close)
**Version**: spec rev 3 · design rev 2 + slice-6 correctives · citations C1…C8
**Mode**: Strict TDD
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4`
**Branch / tip**: `feat/ai-28-7-pre-decode-checks` @ `6fbade1`
**Module root**: `backend/agent` (repo root is a `go.work` workspace; `./...` from the repo root does not resolve)

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 30 |
| Tasks complete | 30 |
| Tasks incomplete | 0 |

Counted mechanically: `grep -cE '^- \[[ x]\] ' tasks.md` = 30, `grep -cE '^- \[x\] '` = 30, zero `- [ ]` task lines. (`design.md:133`'s `- [ ] None blocking` is an Open-Questions placeholder, not a task.)

### Independent artifact re-count

Every number below was re-derived from `specs/ai-provider-text-stream/spec.md`, not trusted from the spec header or apply-progress.

| Metric | Counted | Spec self-report | Match |
|---|---|---|---|
| Requirements `### R-ATS-0NN` | 28 (28 unique) | 28 | ✅ |
| Scenario bullets | 107 (107 unique, no gap in 001–107) | 107 | ✅ |
| `[test]` | 89 | 89 | ✅ |
| `[inspection]` | 18 | 18 | ✅ |
| Inspection ID set | 019,022,023,042,043,047,070,085,089,097,098,099,100,101,102,103,104,107 | identical | ✅ |

### Build & Tests Execution

**Build**: ✅ Passed — `go build ./...` exit 0, empty output.

**Tests**: ✅ 231 top-level passed / 0 failed / 0 skipped (422 including subtests).

```text
$ go test -race -count=1 ./...        # run A and run B, both exit 0
ok  github.com/cachicamas/backend/agent/src/agenttest        2.252s
ok  github.com/cachicamas/backend/agent/src/ai              3.688s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat 4.136s
```

`go test -race -count=1 -v ./src/ai/openaicompat/...` → `--- PASS: Test` = 231, `--- FAIL: Test` = 0, `--- SKIP:` = 0.
Test count reconciles with apply-progress: 230 at slice-7 close + 1 (`TestStream_FiveNullFinishChunksNoTerminal_EmitsNoCompletion`, commit `6fbade1`) = 231.

**Targeted per-slice runs**, each guarded against the vacuous-filter trap (a `-run` regex matching zero tests reports "ok" and proves nothing):

| Slice | `-run` regex | Top-level matched | Result |
|---|---|---|---|
| 1 | `TestStream_\|TestClient_HasNoStreamingEntryPoint\|TestClient_DoesNotSatisfyModelProviderAtRuntime` | 18 | PASS |
| 2 | `TestChunk\|TestContentText\|TestRawStrictFinishReason\|TestMapperState\|TestConformanceBridge` | 14 | PASS |
| 3 | `TestTerminal\|TestTruncation` | 8 | PASS |
| 4 | `TestUsage` | 7 | PASS |
| 5 | `TestTolerance\|TestKeepAlive` | 10 | PASS |
| 6 | `TestProtocolViolation\|TestCharter_` | 11 | PASS |
| 7 | `TestPreDecode` | 7 | PASS |
| **negative control** | `TestThisDoesNotExistAnywhere` | **0** | **correctly flagged VACUOUS-FILTER** |

The negative control proves the >0-match detector itself works; every real group matched >0.

**Coverage**: ➖ Not run — no coverage threshold configured in `Makefile`/`.golangci.yml`; informational only per Strict TDD rules.

### Quality Metrics

**Linter**: ✅ `make lint` (`go vet ./...` + `bin/golangci-lint run --config=.golangci.yml ./...`) → `0 issues.`
**Formatting**: ✅ `gofmt -l ./src/ai/openaicompat/` → empty.
**Dependencies**: ✅ `grep -c '^require' go.mod` = 0; `git diff --stat <merge-base(main)>..HEAD -- go.mod` empty (byte-identical); `go.mod` contains only `module` + `go` directives.

Pre-existing, out-of-scope: `gofmt -l ./src/` reports `src/ai/completion_test.go`. Verified out of scope two ways — it is absent from the list of files touched by AI-28's own 24 commits, and `git show main:backend/agent/src/ai/completion_test.go` also fails `gofmt`. Not counted against this change.

### Spec Compliance Matrix — `[test]` scenarios

**89/89 `[test]` scenarios map to a named test function that passed at runtime.** Method: extracted every literal `S-ATS-NNN` token from all `*_test.go` files, attributed each to its owning top-level `func Test…` (doc-comment blocks attributed forward to the function they document), then intersected against the verbose pass roster.

- Scenario IDs cited in test sources: 98/107. The 9 uncited are **exactly** the 9 inspection-only IDs (023, 043, 085, 089, 099, 100, 101, 103, 104) — so no `[test]` scenario lacks a citation.
- Distinct ATS-citing test functions: **75**; all 75 present in the pass roster.
- `S-ATS-041` is cited in shorthand (`S-ATS-040/041`, `bridge_test.go:188`) plus explicitly at lines 101 and 165; owner `TestConformanceBridge_StreamingText` — a regex artifact in the first pass, not a gap.
- **S-ATS-038 confirmed closed.** `TestStream_FiveNullFinishChunksNoTerminal_EmitsNoCompletion` (`stream_text_test.go:234`) drains a real five-chunk null-finish transcript and asserts all 5 deltas present, zero `Completion` events anywhere, and an `ErrorPayload` terminal. Non-vacuous: the positive delta count blocks a premature terminal from hiding the defect. The apply-phase gap is genuinely resolved; `[test]` coverage is 89/89.

Per-requirement `[test]` totals (all COMPLIANT): R-ATS-001 3, 002 4, 003 4, 004 4, 005 3, 006 2, 007 4, 008 5, 009 3, 010 4, 011 2, 012 3, 013 4, 014 3, 015 4, 016 4, 017 5, 018 2, 019 3, 020 6, 021 3, 022 2, 023 4, 024 3, 025 3, 026 0, 027 0, 028 2 = **89**.

### `[inspection]` Scenarios — all 18 performed in this session

| ID | Result | Evidence |
|---|---|---|
| S-ATS-019 | ✅ | Exactly one `close(` on the carrier in production: `stream.go:298 defer close(out)` inside `run`. The 2 other `close(` in `stream_test.go` are on a test-local `release` channel. Exactly one carrier send — `stream.go:410 case out <- stamped` — inside `select` with `case <-ctx.Done()`; all emission funnels through `emit`. |
| S-ATS-022 | ✅ | Both guards present in `provider_boundary_test.go` under their original names, assertions inverted, doc comments cite AI-28 + R-ATS-001/R-ATS-006 + S-ATS-020/021. Zero `t.Skip`, zero commented-out tests. |
| S-ATS-023 | ✅ | `doc.go:5` now reads "Streaming behaviour arrives at AI-28 (see stream.go)"; `client.go:60-62` likewise corrected. Zero remaining "arrives at AI-26" streaming prose. AI-28 declares zero exported package-level vars in `stream.go`/`chunk.go`/`stream_state.go`; `TestPolicy_NoNewSentinelsExported` (live `go/ast` scan) PASSES, so no allowlist entry is owed. |
| S-ATS-042 | ✅ | Isolated AI-28's own 24 non-merge commits by excluding all three merges' second parents (`792f464` AI-32, `1d50666` amendment, `eeb6661` AI-27). **Zero** touch any `src/agenttest/` path. Full touched-file list is confined to `src/ai/openaicompat/` + this change's openspec docs. |
| S-ATS-043 | ✅ | `agenttest.RunConformanceFor` (`conformance_scoped.go:33`, R-CNF-023) and `Step.IsHold`/`Step.Event`/`Script.Steps` (`script_introspect.go:22,26`, `fake_script.go:47`, R-CNF-024) both landed. Bridge cites both by requirement ID, declares no substitute, never invokes whole-registry `RunConformance`. Both text cases pass unmodified. |
| S-ATS-047 | ⚠️ PARTIAL | Fixture half ✅ — 53 committed occurrences of the exact bytes `data: [DONE]` across 9 test files. Citation half ⚠️ — see **W2**. |
| S-ATS-070 | ✅ | Every `len()` in `chunk.go`/`stream_state.go`/`stream.go` is a decode guard or loop bound. No minimum delta count, minimum block count, or non-empty-text requirement on any path. `len(c.Choices)==0` yields `hasChoice=false` (absorb), never a failure. |
| S-ATS-085 | ✅ | Only map read is `finishReasonEnum[raw]` — a `map[string]bool` whose zero value is the correct "not a member" answer; no dereference, cannot panic. Every slice index guarded: `raw[0]`/`raw[len-1]` by `len(raw) < 2` short-circuit; `body[i]` by loop bound and by `i+1 >= len(body)`; `body[i+1:i+5]` by `i+4 < len(body)`. Every pointer deref nil-guarded (`c.Index != nil && *c.Index >= 0`, `chunk.Usage != nil`, `choice.FinishReason != nil`). |
| S-ATS-089 | ⚠️ PARTIAL | Named-constant half ✅ — `captureLimit` (`capture.go:18`, `8 << 10`), reused, no second bound invented. Credential-scan half ⚠️ — see **W3**. |
| S-ATS-097 | ✅ | Runtime probe: `ai.FailureCategories()` returns exactly **9** members (authentication, authorization, rate_limit, unavailable, timeout, cancellation, malformed_response, unsupported_capability, unknown). AI-28's commits touch zero files under `src/ai/` outside `openaicompat/`, so the vocabulary is unwidened by construction. |
| S-ATS-098 | ✅ | 0 requires; `git diff --stat` vs merge-base empty. |
| S-ATS-099 | ✅ | `stream.go` imports context/fmt/io/mime/net/http/time + `src/ai`; `chunk.go` bytes/encoding-json/unicode-utf8 + `src/ai`; `stream_state.go` fmt + `src/ai`. Programmatic sweep of **all** production files: zero third-party imports. |
| S-ATS-100 | ✅ | Zero `EventKindReasoning`/`EventKindToolCall`/`NewReasoning`/`NewToolCall` in production. `tool_calls`/`function_call` appear only at `chunk.go:327,329` as `finishReasonEnum` map keys (C2's enum). `wireDelta` declares only `Content`, so tool-call/function-call/role/refusal delta fields are structurally dropped by decode. |
| S-ATS-101 | ✅ | `stream_state.go:202 s.usage = usageFromWire(*chunk.Usage)` — plain overwrite, never a merge, with the "last populated wins, no cumulative merge (D10)" comment. `wireUsage` decodes only `prompt_tokens`/`completion_tokens` as `*int64`; `total_tokens` and both detail objects deliberately undecoded. |
| S-ATS-102 | ✅ | **Dangling-citation sweep, this wave's named failure mode: 0 hits.** Every `C<n>` reference across all AI-28 production + test comment lines resolves within C1…C8; distinct numbers referenced = {1,2,3,4,5,6,7,8}; zero out-of-range. Counts: C1 11, C2 10, C3 3, C4 14, C5 5, C6 1, C7 5, C8 3. All eight claims exist as `## C<n> —` headers in `citations.md`. Sweep repeated over **all** production files (incl. AI-32's, which cite a different `E<n>` series): still zero out-of-range. |
| S-ATS-103 | ⚠️ PARTIAL | Range check clean, but two wire-descriptive sites lack the required citation/label — see **W2** and **W4**. This closes the confidence caveat apply-progress disclosed: the exhaustive audit finds real gaps the mechanical range check could not. |
| S-ATS-104 | ✅ | `doc.go:80-81` wraps the hash as `d4fb706e6e05d4cc9f1b33ca5` + `9b6e4f3e8edd439`; concatenation is byte-identical (40 chars) to `citations.md:6`'s pin `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439`. One pin, not two. |
| S-ATS-107 | ✅ | Zero symbols, branches or comments in `stream.go`/`chunk.go`/`stream_state.go` claim to map a Chat Completions in-stream error payload. The single `ai.ErrorEvent` use (`stream.go:436`, inside `emitFailure`) constructs the adapter's own terminal from an adapter-detected condition, never from a wire error frame. |

**Inspection summary**: 15/18 fully discharged, 3 PARTIAL (047, 089, 103), 0 failed.

### Wire-Shape Audit — code comments vs `citations.md` claim text

| Claim | Code site | Claim text | Verdict |
|---|---|---|---|
| C2 finish enum | `chunk.go:324-330` `finishReasonEnum` | "{stop, length, tool_calls, content_filter, function_call}, nullable: true" | ✅ exact five, byte-exact match, no trim/case-fold |
| C4 usage chunk | `chunk.go:49,117`, `stream_state.go:200` | "additional chunk before `data: [DONE]`… `choices` always an empty array… all other chunks carry `usage` null… not guaranteed if interrupted" | ✅ consistent (see S1 note on quoting) |
| C5 sentinel posture | `stream.go:92,292,316` | "documented in prose…, **not as a schema constant** — the only schema-level `"[DONE]"` belongs to the Assistants `DoneEvent`" | ⚠️ cites C5, omits the posture — **W2** |
| C7 delta fields | `chunk.go:15-20,220` | "exactly five optional properties, no `required` list; `content` and `refusal` additionally nullable" | ✅ trichotomy (absent/null/string) matches |
| C1 required set | `chunk.go:60`, `stream_state.go:70` | "required: choices, created, id, model, object" | ⚠️ only `model` enforced — **W1** |
| C3 negative | `bridge_test.go:18,21,125` only | "no block/content-part start or stop signal exists; the adapter mints boundaries" | ⚠️ absent from the minting source — **W4** |
| C8 usage | `chunk.go:126-137` | "required prompt_tokens/completion_tokens/total_tokens + two optional detail objects" | ✅ names `total_tokens` as required and discloses non-decoding with R-ATS-026 rationale |

### Event-Contract Conformance (AI-14.4 `ai.CheckStream`)

The AI-23.2 text conformance cases pass against real transport through the R-CNF-023 scoped entry point:

```text
--- PASS: TestConformanceBridge_StreamingText
    --- PASS: TestConformanceBridge_StreamingText/text/order_contiguity_byte_exact_reconstruction
    --- PASS: TestConformanceBridge_StreamingText/text/empty_completion_is_legal
```

8 `ai.CheckStream` call sites across 7 AI-28 test files. Spot-checked shapes — minimal drain (S-ATS-009), no-content-anywhere (S-ATS-068), empty-`choices` usage chunk (S-ATS-061), duplicate sentinel (S-ATS-054), no-block-minted (S-ATS-031): **all admitted, violation nil.**

**However — the checker does NOT admit every stream the producer emits.** Every shipped `CheckStream` call site covers a *clean* or *post-terminal* shape. No shipped test runs `CheckStream` over a **pre-terminal failure** stream, and those streams are rejected. See **CRITICAL C1**.

### Mutation Spot-Probes — 4 staged, each reverted clean

Each mutation had to make a **named existing test** fail. Confirmed `git status --porcelain` clean after every revert.

| # | Mutation | Named test that failed | Message |
|---|---|---|---|
| 1 | Raw-string finish gate accepts `"STOP"` (breaks D5/N-6) | `TestRawStrictFinishReason_OutsideEnum_RejectedEvenWhenNormalizeWouldAccept` | `chunk_test.go:200: rawStrictFinishReason("STOP") ok = true, want false — raw-string-strict, no case fold (D5, N-6)` |
| 2 | `isChunk()` absent-`object` → process (restores superseded slice-5 deviation) | `TestProtocolViolation_AbsentObjectDiscriminator_SkippedBetweenTwoContentChunks` | `violation_test.go:78: deltas = ["alpha" "INTRUDER" "omega"], want ["alpha" "omega"]` |
| 3 | Pre-decode content-type gate disabled | `TestPreDecode_NonStreamContentType_HTMLErrorPage_RefusedWithBoundedExcerpt`, `…_MissingContentTypeHeader_…`, `…_NonStreamContentTypeHugeBody_…` (3 tests) | `predecode_test.go:91: Stream() returned a non-nil channel for a non-stream content type, want nil` |
| 4 | Absent `prompt_tokens` → reported `Tokens(0)` (breaks absent-vs-zero) | `TestUsage_OmittedVsExplicitZero_UsageRecordsNotEqual` | `usage_test.go:133: usage records are equal (…), want NOT equal — an omitted field must be distinguishable from an explicit 0 (S-ATS-057)` |

The two negative-control scenarios the spec mandates (S-ATS-057, S-ATS-060) are exactly what caught mutation 4 — the non-vacuity discipline paid off.

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|---|---|---|
| R-ATS-001…006 | ✅ Implemented | One `Stream` entry point; guards flipped in place; pre-stream ordering `IsZero`→`Translate`→`ctx.Err` per D1/N-7a. |
| R-ATS-007 | ✅ Implemented | `contentText` trichotomy; byte-preserving unquoter. |
| R-ATS-008 | ⚠️ Partial | Rules 1–4 correct and tested. The preamble's "satisfying AI-16's contract" is **not** met on pre-terminal failure paths — **C1**. |
| R-ATS-009…010 | ✅ Implemented | Raw-byte split-rune preservation; raw-string-strict enum gate. |
| R-ATS-011 | ✅ Implemented | Capability-scoped bridge, test-only, both cases green. |
| R-ATS-012…014 | ✅ Implemented | Sentinel recognised pre-decode; truncation terminal; post-sentinel frames ignored. |
| R-ATS-015…016 | ✅ Implemented | Pointer-based absent-vs-zero; presence positively asserted with negative controls. |
| R-ATS-017…019 | ✅ Implemented | Event-type and discriminator skips; unknown delta fields dropped by decode; keep-alives inert. |
| R-ATS-020 | ✅ Implemented | All five rows produce typed malformed terminals; `terminalSeen`-gated window split. |
| R-ATS-021 | ⚠️ Partial | Invalid JSON ✅; C1 required-field set enforced for `model` only — **W1**. |
| R-ATS-022 | ✅ Implemented | `recover()` probe green; partial output preserved ahead of every terminal. |
| R-ATS-023…024 | ✅ Implemented | `mime.ParseMediaType` match then `mapResponse`, both before channel creation; `PreStreamFailure` direct return. |
| R-ATS-025 | ✅ Implemented | Nine unexported causes wrapping `ai.ErrMalformedResponse`; zero new exported sentinels; `src/ai` untouched. |
| R-ATS-026 | ✅ Implemented | Zero requires; stdlib + in-repo only; no reasoning/tool-call/usage-merge paths. |
| R-ATS-027 | ⚠️ Partial | Zero dangling citations, but two wire-shape sites lack a claim/label — **W2**, **W4**. |
| R-ATS-028 | ✅ Implemented | No bespoke in-band error path. |

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| D1 producer lifecycle | ✅ Yes | Ordering and step-6 pre-decode placement match exactly. |
| D2 byte-preserving unquoter | ✅ Yes | Hand-rolled; `RawMessage`; proven by mutation (slice 2's `json.Marshal` swap reproduced U+FFFD). |
| D3 skip-vs-fail | ✅ Yes | Three-way object rule as ruled in slice 6. |
| D4 block minting / completion deferral | ✅ Yes | Completion deferred to sentinel; block end precedes completion. |
| D5 / N-6 raw-string-strict gate | ✅ Yes | Mutation-proven (probe 1). |
| D6 sequencing | ✅ Yes | One `Stamper`; contiguous 1…N. |
| D7 failure construction | ✅ Yes | Unexported causes; `%w`; `errors.Is` holds. |
| **D8 close open block before terminal ErrorEvent** | ❌ **No** | **Not implemented. See C1.** |
| D9 `[DONE]` with no terminal chunk | ✅ Yes | Malformed terminal. |
| D10 usage presence mapping | ✅ Yes | Overwrite, no merge. |
| D11 conformance bridge | ✅ Yes | `RunConformanceFor`, `Script.Steps`/`Step.Event`/`Step.IsHold`, hold-step fail-fast, all three optional caps declared non-nil false. |
| D12 guard flips | ✅ Yes | Names retained, polarity inverted, no S-ART-054 edit. |

### Coordinator Rulings Audit — all four landed and honored

| Ruling | Artifact | Code | Verdict |
|---|---|---|---|
| A — S-ATS-046 spec rev 3 (well-formed sentinel frame at exact EOF) | Blockquote at `spec.md:250`; corrected scenario at `:252` | `TestTerminal_DoneWithNoPaddingBeyondMandatoryBlankLine_SameOutcome` uses `"data: [DONE]\n\n"` — terminating blank line then EOF | ✅ artifact and test agree |
| B — object three-way rule + fixture normalization | `design.md:30` corrective; Engram #2497 | `isChunk()` = `Object == chunkObjectDiscriminator` (absent→skip, mismatch→skip); `hasRequiredFields` → malformed. 146 discriminator occurrences across 11 test files; **zero** chunk fixtures still missing it | ✅ (C1-completeness caveat W1) |
| C — S-ATS-032 / S-ATS-074 window split | `spec.md:188` and `:358` explicitly contrast; row-1 window stated at `:348` | `stream_state.go` `s.terminalSeen` gate: content→`errDeltaAfterClose`, second finish→`errDuplicateClose`, else absorbed toward clean close. Empirically confirmed both outcomes | ✅ two shapes share no fixture and no outcome |
| D — S-ATS-038 closure | `tasks.md:132` RESOLVED note | `TestStream_FiveNullFinishChunksNoTerminal_EmitsNoCompletion` present, non-vacuous, passing | ✅ `[test]` coverage 89/89 |

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ✅ | Per-slice RED/GREEN evidence in `tasks.md` + TDD Cycle Evidence table in apply-progress |
| All tasks have tests | ✅ | 30/30; every `[test]` scenario reaches a named test |
| RED confirmed (artifacts exist) | ✅ | See audit below |
| GREEN confirmed (tests pass) | ✅ | 231/231 at runtime, `-race`, twice |
| Triangulation adequate | ✅ | e.g. S-ATS-012–015 triangulated 3 ways; S-ATS-093 an 11-row table; S-ATS-033/035 split-rune 2-way and 3-way |
| Safety net for modified files | ✅ | Each slice re-ran the prior baseline; counts 137→148→168→176→183→193→204→(223 post-merge)→230→231 |

**RED-evidence audit — 8 claims sampled across 6 slices, verified against the tree:**

| Claim | Verified |
|---|---|
| 1.1 RED messages | ✅ verbatim at `provider_boundary_test.go:22` and `:42` |
| 2.1 symbols `decodeChunk`/`contentText`/`rawStrictFinishReason` | ✅ `chunk.go:198,227,339` |
| 2.1 hard-coded split-rune literal (anchor, not derived) | ✅ `chunk_test.go:117 const want = "caf\xc3\xa9 shop, abierto"` |
| 4.1 two subtests in the negative-control test | ✅ exactly 2 `t.Run(` |
| 5.1 adversarial `INTRUDER` fixtures | ✅ 3 in `tolerance_test.go`, 3 in `violation_test.go` |
| 6.2 four unexported row causes | ✅ all four present |
| 7.1 typed cause + `captureLimit` reuse | ✅ `stream.go:255,271` |
| 8.2 S-ATS-038 closure test | ✅ `stream_text_test.go:234` |

**Claimed-vs-actual top-level test counts**: 9/10 files MATCH. One mismatch — see **S1**.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Integration (real loopback HTTP: `httptest.Server` → real `*Client` → `Stream`) | ~57 driver sites | 10 | stdlib `net/http/httptest` |
| Unit (pure decode/mapper seams: `contentText`, `rawStrictFinishReason`, `applyChunk`) | 15 | `chunk_test.go`, `stream_state_test.go` | stdlib `testing` |
| Conformance (AI-23.2 via R-CNF-023) | 1 (2 subcases) | `bridge_test.go` | `agenttest` |
| **Total AI-28 top-level** | **~75 ATS-citing** | **13** | |

No E2E layer — correct for an in-process adapter; no browser/subprocess boundary exists.

### Assertion Quality

**338 assertions** across the 13 AI-28 test files. Scan results:

- Tautologies (`if true`, `expect(true)`): **0**
- `t.Skip` / commented-out tests: **0**
- Mocks/stubs: **0** — real `httptest.Server` + real `*Client` throughout (mock:assertion ratio 0)
- Orphan empty-collection assertions without a non-empty companion: **0**
- Ghost loops (assertions over a possibly-empty collection): **11 mechanically flagged, all cleared on inspection.** Each is guarded by a positive discriminator that fails on an empty drain — `violation_test.go`'s shared runner `t.Fatal`s on `len(events)==0` and on a missing `ErrorPayload` before any row-specific closure runs; `terminal_test.go:246` asserts `terminalCount != 1`; `usage_test.go:256` asserts `response-start count … want exactly 1`; `stream_test.go:456`'s negative scan follows indexed positive assertions on `events[0]`.

**Assertion quality**: ✅ All assertions verify real behavior — 0 CRITICAL, 0 WARNING.

### Vacuous-Pass Sweep (Engram #2471's nine shapes)

| Shape | Hits |
|---|---|
| 1. Cannot distinguish implemented from not | 0 in shipped tests — **but see C1**: the *suite* cannot distinguish a contract-conformant failure stream from an unterminated-block one |
| 2. Implementation compared against itself | 0 — split-rune tests anchor on hard-coded literals |
| 3. Corruption magnitude unobservable | 0 |
| 4. No meaningful content after the interesting position | 0 — S-ATS-033/035 place 15 bytes / trailing text after the split; S-ATS-090/092 use fully completable transcripts |
| 5. Cannot distinguish correct from misrouted-but-unobserved | 0 in shipped tests; `INTRUDER` fixtures are the explicit defense |
| 6. Stale un-trimmed state coincidence | 0 |
| 7. Sort-coincidence | 0 — no order-sensitive fixture with pre-sorted values |
| 8. Helper builds same fixture regardless of case | 0 — `violation_test.go` rows carry distinct transcripts |
| 9. Mutation that only disables a check | 0 — all 4 of my probes produced scenario-specific failures naming the target |

**Confirmed vacuous tests: 0.** Shape 1/5 applies at suite level to the failure-path CheckStream omission (**S3**), which is the structural reason C1 survived seven slices.

### Issues Found

**CRITICAL**

**C1 — design.md D8 is not implemented: the producer emits unterminated text blocks on every pre-terminal failure path, and AI-14.4's own `ai.CheckStream` rejects those streams.**

`design.md` D8 states: *"Emit `TextBlockEnd` before the terminal `ErrorEvent` when a block is open… chosen so a recorded failure stream still satisfies `ai.CheckStream`'s no-unterminated-block invariant (AI-16)."* The design-validation gate independently confirmed it as **mandatory** (Engram #2478: *"CheckStream sweeps unterminated blocks AFTER the loop → close block before terminal ErrorEvent is mandatory (D8 correct)"*).

The only `TextBlockEnd` mint is `stream_state.go:265-271`, gated on `s.terminalSeen && s.blockOpen` — i.e. it fires only when a **terminal chunk** arrives. No failure path closes an open block. `emitFailure` (`stream.go:424-438`) builds and emits the `ErrorEvent` with no block close.

Verified empirically against the real producer (real `httptest.Server` → real `*Client` → `Stream`, using the shipped `sseServer`/`mustClient`/`validRequest`/`drainAll` helpers), separating the two windows the design distinguishes:

```text
PRE-terminal window (design records D8 as LOAD-BEARING here)
  A truncation mid-block (S-ATS-048's own fixture shape)
      kinds = [responsestart text_block_start text_delta text_delta error]
      -> VIOLATION event[2].block[1]: value is not well-formed for its documented encoding
         errors.Is(v, ai.ErrMalformed) == true  -> UNTERMINATED BLOCK
  B invalid JSON mid-block
      kinds = [responsestart text_block_start text_delta error]              -> same VIOLATION
  C out-of-enum finish_reason after content
      kinds = [responsestart text_block_start text_delta error]              -> same VIOLATION

POST-terminal window (design records D8 as a NO-OP here — confirmed)
  D delta-after-close (R-ATS-020 row 1)
      kinds = [responsestart text_block_start text_delta text_block_end error] -> CLEAN
  E clean completion control
      kinds = [responsestart text_block_start text_delta text_block_end completion] -> CLEAN
```

The probe's premise was checked against the coordinator's caution and holds: shapes A–C are **pre-terminal** (no terminal chunk ever arrives, so nothing closes the block), which is precisely the window where the design records D8 as load-bearing. The design's no-op note is confirmed by shape D — and it is exactly what makes the gap invisible to the post-terminal tests.

The violation is unambiguous. `ErrMalformed` (`src/ai/validation.go:91` = "value is not well-formed for its documented encoding") is emitted by `CheckStream` at **exactly one** site: the post-loop unterminated-block sweep (`stream_check.go:110-115`), positioned `AtIndex("event", state.startSeq), AtIndex("block", idx)`. So `event[2].block[1]` means *block 1, opened at sequence 2, never ended* — not "event index 2".

**Blast radius** — every failure path that can leave a block open, which includes the headline behaviors of three slices: R-ATS-013 truncation (AI-28.2's primary requirement, S-ATS-048/049), R-ATS-021 malformed-payload terminal (S-ATS-080/081), and R-ATS-010's out-of-enum finish reason (S-ATS-039). It also means `agenttest.RequireValidStream` — which delegates to `ai.CheckStream` and `t.Fatalf`s on violation (`stream_kit_ordering.go:25-27`) — would fail any future conformance case that records one of these streams.

**Scoping, stated honestly**: no numbered scenario fails, and all 89 `[test]` scenarios pass. R-ATS-008 rule 3 is literally silent here (it fires on "terminal chunk arrives **or** the stream terminates cleanly", neither of which happens). What is breached is R-ATS-008's **preamble** — block boundaries are *"adapter-defined behavior satisfying AI-16's contract"* — and an unterminated block does not satisfy that contract, as the repository's own AI-14.4 checker proves mechanically. Combined with an explicit, validator-confirmed design decision left unimplemented, I classify this CRITICAL rather than a design-deviation WARNING.

**Fix direction** (not applied — verification does not fix): close an open block in the failure path before emitting the terminal `ErrorEvent`, and add `ai.CheckStream` to the failure-path tests so the invariant is guarded (see S3).

**WARNING**

**W1 — R-ATS-021's "C1 required top-level field set" is enforced for `model` only.** `chunk.go`'s `hasRequiredFields()` is `return c.Model != ""`. C1 requires `choices`, `created`, `id`, `model`, `object`. `wireChunk` declares no `Created` field at all, so `created` is never decoded and a chunk omitting it is accepted rather than yielding the malformed terminal R-ATS-021 mandates. The code discloses its reasoning (`chunk.go:105-120`): `id` is covered indirectly (`ai.NewResponseStart` on the first chunk; row-3 identity comparison later — verified: `chunk.ID == ""` ≠ an established non-empty id, so it does trip `errSecondResponseStart`), `object` by `isChunk()`, `choices` deliberately unprobed because `[]` is C4's legitimate usage-chunk shape. `created` has no such coverage. S-ATS-081 passes because it tests exactly the `model` case — the requirement is broader than its scenario, and was implemented to the scenario.

**W2 — S-ATS-047 / S-ATS-103: the sentinel-recognising source cites C5 without its required dialect-conventional posture.** S-ATS-047 requires the source to cite *"C5 **together with its prose-documented, dialect-conventional posture**"*; R-ATS-027 names this one of only two dialect-conventional labels the spec carries. `stream.go:92,292,316` cite C5 but never state that the sentinel is prose-documented rather than a schema constant. Grep for `prose-documented`, `not a schema constant`, or `dialect-conventional` across AI-28's production files returns **zero** hits for the sentinel (the 4 `dialect-conventional` labels present belong to AI-26/AI-32 files citing a different `E`-series). The fixture-pin half is fully discharged (53 occurrences).

**W3 — S-ATS-089: the credential-scan guard does not cover the excerpt path.** `scanCredentialSurface` (`credential_scan_test.go:60,66`) scans only files matching `^package openaicompat_test$`. `predecode_test.go` is `package openaicompat` (internal) — and in fact **every** AI-28-authored test file is internal, so none is scanned. The scenario's second clause is therefore not literally satisfied. The substituted argument is sound and the residual risk is genuinely low — `refuseNonStreamContentType` reads only `resp.Header.Get("Content-Type")` and `resp.Body`, never `c.credential`, so it cannot reproduce a credential by construction — and apply disclosed it honestly. Recorded as a documented deviation, not a discharge.

**W4 — S-ATS-103: the block-minting source carries no C3 citation.** C3 is the load-bearing negative claim that the wire has no block framing, making every boundary adapter-minted. It is cited **only** in `bridge_test.go` (3 sites); `stream_state.go`'s minting logic, `textBlockIndex`, and the `blockOpen`/`blockClosed` state comments describe framing behavior while citing only R-ATS-008. R-ATS-008's actual prohibition is honored — no comment claims a boundary was vendor-reported — but S-ATS-103's citation requirement is not met at the site that most needs it.

**SUGGESTION**

**S1 — `tasks.md` task 6.1 overcounts.** It records *"created `violation_test.go` (11 top-level tests)"*; the file has **7**. Task 6.4's own cross-claim (`violation_test.go` + `charter_test.go` = 7 + 4 = 11) is correct, so 6.1 misattributes the slice total to one file. No coverage impact — every slice-6 scenario still maps to a passing test — but it is the "an evidence log can lie" class, found by re-counting rather than reading.

**S2 — flipped guard names are now semantically inverted.** `TestClient_HasNoStreamingEntryPoint` asserts the method **does** exist; `TestClient_DoesNotSatisfyModelProviderAtRuntime` asserts it **does** satisfy. This follows design D12 ("keep their names' successors with inverted polarity") and both doc comments explain it, so it is a deliberate, disclosed choice — but the names now actively mislead a reader grepping the suite.

**S3 — no failure-path test runs `ai.CheckStream`.** All 8 shipped `CheckStream` call sites cover clean or post-terminal shapes. This is the structural reason C1 survived seven slices and a milestone-close audit: the assertions on failure streams check `Category()`, `Delivery()`, `PartialOutput()` and delta preservation, but never the event-contract invariant. Adding `CheckStream` to `truncation_test.go` and the violation-table runner would have caught D8's absence immediately, and would guard it after the fix.

**S4 — concurrent writers in the shared worktree during verification.** `git status --porcelain` was clean at session start; during this run three untracked directories appeared — `openspec/changes/cachicamas-ai-provider-{completion,reasoning-stream,tool-stream}/`, each containing a `proposal.md` for a *different* change (AI-29/AI-30 and a completion change), written by a concurrent process. Verified harmless to this verification: zero `backend/` files touched and `HEAD` unchanged at `6fbade1` throughout. Flagged because a shared worktree during a verification window is a contamination risk for future phases.

### Verdict

**FAIL** — the change is otherwise in excellent shape (30/30 tasks, 28/28 requirements homed, 89/89 `[test]` scenarios passing under `-race` twice, 18/18 inspections performed, `make lint` 0 issues, zero dependencies, zero dangling citations, 4/4 coordinator rulings honored, 0 vacuous tests), but design decision D8 is unimplemented and the shipped producer emits streams that AI-14.4's own `ai.CheckStream` rejects on every pre-terminal failure path — a contract breach on the headline behavior of three slices. Not archive-ready until C1 is fixed and guarded by a failure-path `CheckStream` assertion.
