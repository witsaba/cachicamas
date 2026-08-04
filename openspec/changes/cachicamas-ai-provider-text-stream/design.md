# Design — translate the response lifecycle and text

> **Change**: `cachicamas-ai-provider-text-stream` · **Milestone**: AI-28 · **Phase**: design · **Date**: 2026-08-04
> Spec is binding: `specs/ai-provider-text-stream/spec.md` **rev 2** (R-ATS-001…028). Wire facts: `citations.md` C1…C8.

> **Corrective (design-validation, 2026-08-04).** This revision applies the design-validation gate's rulings (Engram obs #2478): C-1 terminal-window disambiguation (now spec-ruled in R-ATS-008 rev 2), C-2/C-3 bridge redesign onto the conformance amendment's landed `R-CNF-023`/`R-CNF-024` surfaces, W-1 derived conformance counts, W-2 factory capability declarations, W-3 R-ATS-009's fixture-level premise, W-4 slice-2 blocker in the delivery table, N-1 `client.go` stale prose, N-6 raw-string-strict finish-reason ruling, N-7 pre-stream ordering ruling and the S-ATS-015 empty-`id` home. Validator-confirmed decisions (D2's unquoter necessity — proven by execution; D8; D9; D7's `%w` wrapping; homes arithmetic) stand unchanged.

## Technical Approach

One producer goroutine over a read → `Decoder.Feed` → map → stamp → select-send loop, inside `openaicompat`. Three new production units: `stream.go` (the `Stream` entry point, pre-stream contract, goroutine, close discipline), `chunk.go` (chunk JSON decode with a byte-preserving string unquoter), `stream_state.go` (the mapper state machine: block minting, terminal discipline, protocol-order table, failure construction). The conformance bridge is test-only. No new packages; `go.mod` untouched.

## Architecture Decisions

### D1 — Producer lifecycle
`Stream(ctx, req)`: (1) `req.IsZero()` → `ai.Invalid` direct error; (2) `Translate(req)` — reuses AI-26 validation/refusal, no I/O; (3) `ctx.Err()` → `ai.PreStreamFailure(FailureCategoryCancellation)` (order matches fake_provider.go, S-ATS-007); (4) `newRequest` extended to carry the body (`request.go`'s own doc anticipates this seam) plus `Content-Type: application/json` and `Accept: text/event-stream` (dialect-conventional, fixture-pinned), path segment `chat/completions`; (5) `httpClient.Do`; (6) pre-decode checks (slice 7 only); (7) `make(chan ai.Event)` **unbuffered**, `go run(...)`, return carrier. `run`: `defer close(out)` — the **one** `close(` in the package, reviewer-checkable (S-ATS-019) — plus `defer resp.Body.Close()`; read loop feeds the decoder, every event send is `select { out <- ev; <-ctx.Done(): return }`. The HTTP request uses the caller's `ctx`, so cancellation aborts `Body.Read` and releases transport (AI-20.3). On the sentinel: emit Completion, return early — post-sentinel frames are trivially never decoded (R-ATS-014).
**Rejected**: buffered carrier (fake's `script.Buffer`) — no consumer need at this milestone; unbuffered makes the loss-path race the spec's own.

> **Corrective (design-validation, 2026-08-04) — N-7a, explicit pre-stream ordering ruling.** The `IsZero` → `Translate` → `ctx.Err()` order means a request refused by `Translate` (a reasoning-refusal, R-ART-015) reports the **refusal**, not the cancellation, when both hold — the spec pins only validation-before-cancellation (S-ATS-007) and is silent on where `Translate`'s refusal sits. Ruled acceptable and recorded as a design ruling: `Translate` is part of "validate once, before I/O, before consulting the context" (R-ATS-002) — its refusal is a property of the request, so it outranks a property of the call's context. Destined for `stream.go`'s doc comment.

### D2 — Chunk decode: stdlib structure, hand-owned bytes
`encoding/json.Unmarshal` into a chunk struct whose `delta.content`, `delta.refusal` and `usage` are `json.RawMessage`, then a hand-rolled **byte-preserving unquoter** for `content` (decodes JSON escapes, passes non-escape bytes through raw, no UTF-8 validation).
**Why**: `encoding/json`'s string decode replaces invalid UTF-8 with U+FFFD — it would repair the split-rune fragments R-ATS-009/S-ATS-033 forbid repairing. `RawMessage` preserves bytes verbatim and gives the absent / `null` / `""` trichotomy R-ATS-007 needs for free; unknown delta fields are ignored natively (R-ATS-017, C7 `obfuscation`).
**Rejected**: full hand-rolled chunk parser (body.go's Marshal objection is output-side only; owning all of JSON is unjustified scope) and naive struct decode (corrupts bytes).

> **Corrective (design-validation, 2026-08-04) — W-3, premise now spec-carried.** Spec rev 2 (R-ATS-009) labels the split-rune shape **dialect-conventional with a fixture-pin obligation**: RFC 8259 forbids half a rune inside a legal JSON string, so the shape is expressible only as raw invalid-UTF-8 bytes written directly into committed fixtures — S-ATS-033/034/035 are byte-level robustness pins against proxy/vendor misbehavior, not cited wire behavior. The validator **proved by execution** (go1.26.3) that `encoding/json`'s string decode substitutes U+FFFD on invalid UTF-8 and unpaired surrogates while `RawMessage` preserves bytes: this unquoter is necessary, not defensive. Fixtures MUST carry the exact raw bytes and be read through a byte-preserving path.

### D3 — Skip vs fail boundary (R-ATS-017 vs R-ATS-021)
Invalid JSON → malformed terminal. Valid JSON whose top-level `object` ≠ `"chat.completion.chunk"` (or absent) → skip (S-ATS-066: recognition rides the C1 discriminator). Recognized chunk missing a C1 required field → malformed terminal. Non-default SSE event type → skip before parsing.

### D4 — Block minting and completion deferral (C3, R-ATS-008)
Mapper state: `blockOpen`, `blockClosed`, constant block index `1`, `choiceTerminalSeen`, `finishReason`, `usage`, `outputPreceded`, `sentinelSeen`. First choice-0 `content` string → mint `TextBlockStart` then the delta (S-ATS-030). Terminal chunk → `TextBlockEnd` (if open), record finish reason, but **defer `Completion` to the sentinel** — C4 places the usage chunk between the terminal chunk and `data: [DONE]`, so the completion's usage is only complete at the sentinel (satisfies S-ATS-032: block end precedes completion). No content string anywhere → no block (rule 4, followed as written).

> **Corrective (design-validation, 2026-08-04) — C-1, now spec-ruled (R-ATS-008 rev 2).** The terminal→sentinel window lawfully carries **only delta-less chunks and the usage chunk** (C4); a choice-0 `content` **string** in that window is not rule-3 territory but `R-ATS-020` row 1's *delta after close* and MUST end in a typed malformed-response terminal (S-ATS-074). The state machine implements exactly that split: with `choiceTerminalSeen` set, a chunk whose choice-0 `content` is a JSON string → row-1 malformed terminal via D7; a delta-less chunk, the empty-`choices` usage chunk, or any D3-skippable frame → absorbed, clean close continues toward the sentinel's `Completion` (S-ATS-032 rev 2). The two shapes share no fixture and no outcome. D8/D9 swept for consistency: D8 (close open block before a terminal ErrorEvent) applies unchanged to the row-1 terminal — the block is already closed by the terminal chunk in this window, so D8 is a no-op here and load-bearing only for pre-terminal failures; D9 ([DONE] with no terminal chunk → malformed) is untouched by the disambiguation, which concerns the window **after** a terminal chunk exists. Both were independently validator-confirmed.

### D5 — Finish-reason gate before normalization (R-ATS-010)
Check the raw wire string against C2's five-value allowlist **before** `ai.NormalizeFinishReason`; outside → malformed terminal (S-ATS-039). `NormalizeFinishReason` alone is total and maps unknowns to `FinishReasonUnknown` — that silent tolerance is exactly what R-ATS-010 forbids. Inside the enum, normalization maps `stop/length/tool_calls/function_call/content_filter` to the vocabulary.

> **Corrective (design-validation, 2026-08-04) — N-6, explicit ruling: the gate is raw-string-strict.** The allowlist matches the wire value **byte-exactly** against C2's five lower-case members, with no trimming and no case folding — `"STOP"` is outside the enum and yields the malformed terminal, even though `NormalizeFinishReason` (which trims and lower-cases) would happily map it. Rationale: R-ATS-010 binds the producer to C2's enum as the schema states it; the normalizer's leniency exists for the cross-vendor table (AI-31.1's territory), not for this dialect's own schema-declared constant set — accepting a case variant here would be recording inference as fact (R-ATS-027). Deliberate, validator-confirmed, destined for `doc.go`'s AI-28 rulings.

### D6 — Sequencing
One fresh `ai.Stamper` per stream; the single emit helper stamps immediately before the select-send (fake_provider.go precedent). Sequence runs 1…N contiguously by construction (AI-22.3).

### D7 — Failure construction, no new sentinels (R-ATS-025)
One helper builds every terminal: category + cause + `outputPreceded` → `ai.MidStreamFailure` → `ai.ErrorEvent`, emitted through the same select-send. Pre-handover failures return the error directly (`ai.PreStreamFailure`). Decoder errors: category via `openaicompat.Category(err)`. Missing-sentinel-at-EOF: cause is `fmt.Errorf(...: %w, ErrTruncated)` so `errors.Is(err, ai.ErrMalformedResponse)` and the S-ATS-094 category identity hold. Protocol-order violations: **unexported** errors wrapping `ai.ErrMalformedResponse`. **No new exported sentinel** → S-ART-054's allowlist stays untouched (the stated choice the launch brief asked for). `outputPreceded` = any text event already emitted; `ResponseStart` does not count (S-ATS-050).

> **Corrective (design-validation, 2026-08-04) — N-7b, S-ATS-015's home stated explicitly.** A first chunk whose `id` (or `model`) is present but **empty** surfaces through `ai.NewResponseStart`'s own constructor error — both fields are non-empty-required — and the mapper's identity-establishment path converts that constructor error into this decision's malformed-response terminal (unexported cause wrapping `ai.ErrMalformedResponse`, `outputPreceded == false`). Home: R-ATS-004, slice 1 (`stream_state.go` identity path + D7's builder). The producer never mints, defaults or substitutes an identity to dodge the constructor. Validator confirmation retained: `(*Failure).Is` matches category sentinels, so the `%w` wrapping keeps `errors.Is(err, ai.ErrMalformedResponse)` true through both the decoder-error and protocol-violation paths.

### D8 — Open block at a failure terminal
Emit `TextBlockEnd` before the terminal `ErrorEvent` when a block is open. Spec-silent; chosen so a recorded failure stream still satisfies `ai.CheckStream`'s no-unterminated-block invariant (AI-16). Recorded here, not a wire claim.

### D9 — `[DONE]` with no terminal chunk (spec-silent shape)
Malformed-response terminal. R-ATS-010 forbids a completion while `finish_reason` is null and R-ATS-005 forbids closing with no terminal event; failure is the only lawful exit.

> **Corrective (design-validation, 2026-08-04) — sweep result.** Validator-confirmed genuinely spec-silent and correctly forced; unchanged by R-ATS-008 rev 2's window disambiguation, which governs the window **after** a terminal chunk exists — D9 is the case where none ever does. D8 likewise validator-confirmed (CheckStream sweeps unterminated blocks after its loop, so closing an open block before the terminal ErrorEvent is mandatory, not stylistic); in the disambiguated post-terminal window D8 is a no-op because the terminal chunk already closed the block. The bridge constraint this decision previously imposed is resolved by the amendment's derived counts (see D11) — the former open question is superseded.

### D10 — Usage presence mapping (R-ATS-015/016)
`prompt_tokens` → `Usage.Input`, `completion_tokens` → `Usage.Output`, `ai.Tokens(v)` only when the key is present (including explicit `0`); absent keys stay zero-value `TokenCount`. `usage: null` contributes nothing; last populated usage chunk wins; no cumulative merge (AI-31.2's).

> **Corrective (design-validation, 2026-08-04).** The citation gap this decision previously flagged is closed: spec rev 2's `citations.md` now carries **C8** covering `CompletionUsage`'s field names. The mapping above cites C8 directly; no dialect-conventional label or fixture-pin fallback is needed and the former open question is removed.

### D11 — Conformance bridge (test-only, R-ATS-011)
`_test.go` files only: `renderScript(tb, Script) []byte` → SSE transcript; per-Script `httptest.Server` handler replays one transcript per request in queue order; the factory builds a **real** `*Client` with `Endpoint: srv.URL`, `HTTPClient: srv.Client()`. Event → wire mapping: `ResponseStart` → first chunk's `id`/`model`; `TextDelta` → content chunk (byte-preserving hand escaping — the bridge must not repair encoding, S-ATS-041); `TextBlockStart/End` → **no bytes** (C3: the adapter re-mints); `Completion` → terminal chunk (vocabulary → wire spelling) + usage chunk when any count present + `data: [DONE]`.

> **Corrective (design-validation, 2026-08-04) — C-2/C-3/W-1/W-2, redesigned onto the amendment's landed surfaces.** The whole-registry `agenttest.RunConformance` is not the target: it runs all seventeen cases with required-capability waivers forbidden and an unexported per-case runner, and `Script`/`Step` previously exposed no accessor a renderer could read. The bridge consumes exactly the two surfaces `cachicamas-ai-conformance-lifecycle-amendment` (spec+design landed, worktree `ai-wave-4-conformance`) ADDs:
> - **R-CNF-023** — `agenttest.RunConformanceFor(t *testing.T, f Factory, capability Capability)`: no return value, registry-scoped to the named capability; the three mandatory `*bool` factory declarations and the CAP-O-02 cross-check remain enforced. S-ATS-040 runs `RunConformanceFor(t, f, CapStreamingText)`.
> - **R-CNF-024** — read-only Script introspection: the exported `Script.Steps` field plus `Step.Event() (ai.Event, bool)` and `Step.IsHold() bool`. **No Gate accessor exists.**
>
> The renderer iterates `Script.Steps`; for each step it takes `Event()` from emit steps and renders per the mapping above; on any step where `IsHold()` is true it **fails fast** with a clear `tb.Fatalf` message — gate-driven scripts are outside text-capability scope, and the bridge must not guess at hold semantics it cannot observe. The bridge factory literal declares `Reasoning`, `TokenCounting`, `CacheBoundary` all non-nil **false** (W-2 — otherwise `factoryDefect` fails construction before any case runs). Counts (W-1): the amendment **derives the expected event window per script** (R-CNF-019 derivation is authoritative) — ordering case = **6** events (`ResponseStart, TextBlockStart, TextDelta, TextDelta, TextBlockEnd, Completion`), empty-completion case = **2** (`ResponseStart, Completion`); this design's former 5/2 arithmetic and the open question built on it are superseded and removed. Dependency recorded for slice 2: `cachicamas-ai-conformance-lifecycle-amendment` carrying **R-CNF-023 + R-CNF-024** — the change exists with spec and design landed, so S-ATS-043's named-dependency inspection is dischargeable; if it lands without either surface, slice 2 STOPS per R-ATS-011 rev 2, never worked around.

### D12 — Guard flips (R-ATS-006)
`provider_boundary_test.go`: both tests keep their names' successors with inverted polarity and doc comments citing AI-28/R-ATS-001. `doc.go` line 5 ("Streaming behaviour arrives at AI-26") corrected to AI-28 in place. No S-ART-054 edit (D7).

## Data Flow

    Stream: validate → Translate → ctx check → newRequest(body) → Do → [pre-decode checks, slice 7]
      └─ go run:  Body.Read ─→ Decoder.Feed ─→ Frame ─→ mapper.apply ─→ []Event ─→ Stamper ─→ select{out<-ev | ctx.Done}
                  EOF ─→ Decoder.Finish ─→ mapper.finish ─→ terminal (Completion | ErrorEvent)
                  defer: close(out) once · Body.Close()

## Requirement homes (28/28 — zero homeless)

| Requirements | Home |
| --- | --- |
| R-ATS-001…006 | `stream.go`, `provider_boundary_test.go` flip, `doc.go` (slice 1) |
| R-ATS-007…010 | `chunk.go` + `stream_state.go` (slice 2) |
| R-ATS-011 | bridge `_test.go` (slice 2, amendment-dependent) |
| R-ATS-012…014 | `stream_state.go` sentinel/truncation paths (slice 3) |
| R-ATS-015…016 | D10 usage mapping (slice 4) |
| R-ATS-017…019 | D3 skip paths, keep-alives inert via decoder (slice 5) |
| R-ATS-020…022 | violation table in `stream_state.go` (slice 6) |
| R-ATS-023…024 | pre-decode checks in `stream.go` (slice 7, BLOCKED on AI-32.1) |
| R-ATS-025…028 | D7 construction, import/charter inspections, citation comments (cross-slice, asserted in slice 1 and 6 tests) |

## File Changes

| File | Action |
| --- | --- |
| `backend/agent/src/ai/openaicompat/stream.go` | Create — `Stream`, producer goroutine, emit helper, (slice 7) pre-decode checks incl. named excerpt-bound constant |
| `backend/agent/src/ai/openaicompat/chunk.go` | Create — chunk struct, required-field check, byte-preserving unquoter |
| `backend/agent/src/ai/openaicompat/stream_state.go` | Create — mapper state machine, failure builder |
| `backend/agent/src/ai/openaicompat/request.go` | Modify — body-carrying request + headers |
| `backend/agent/src/ai/openaicompat/provider_boundary_test.go` | Modify — S-APC-030/031 flips |
| `backend/agent/src/ai/openaicompat/doc.go` | Modify — stale AI-26 prose; AI-28 rulings D8/D9/N-6/N-7a |
| `backend/agent/src/ai/openaicompat/client.go` | Modify — line 61-62's stale "acquires streaming behaviour at AI-26" comment corrected to AI-28 (**hygiene beyond spec** — S-ATS-023 scopes only `doc.go`; 1 line, mechanical, disclosed per this package's correction practice) |
| `*_test.go` per node + bridge + committed `data: [DONE]` fixture | Create |
| `src/ai`, `src/agenttest`, `errors.go`, `go.mod` | Untouched (R-ATS-025/026, S-ATS-042) |

## Testing Strategy

Strict TDD, `-race -count=1`, RED first per slice. Fixtures are raw SSE byte transcripts driven **both** through `httptest.Server` and directly through `Decoder.Feed`+mapper (the two seams D1/D4 separate on purpose). Vacuous-pass catalogue applied: negative controls (S-ATS-057/060), content after split points (S-ATS-033/035), skip-vs-fail asserted different in one test (S-ATS-082), goroutine-leak sampling with tolerance, 18 `[inspection]` scenarios discharged by reviewer-checkable structure (one `close(`, doc citations).

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary; in-process HTTP client against loopback test servers only.

## Migration / Rollout

No migration. Delivery per proposal: auto-chain, feature-branch-chain, budget 5000. AI-28.1.1/AI-28.1.2 separate PRs by doc 0002.

| Slice | Node | Blocker |
| --- | --- | --- |
| 1 | AI-28.1.1 producer shell | — |
| 2 | AI-28.1.2 text mapping + bridge | **`cachicamas-ai-conformance-lifecycle-amendment` (R-CNF-023 + R-CNF-024) must land first** — STOP per R-ATS-011 rev 2 if absent (W-4) |
| 3–6 | AI-28.2 … AI-28.5 | — |
| 7 | AI-28.6 pre-decode checks (last) | **AI-32.1** — recorded carryover if unlanded |

## Open Questions

> **Corrective (design-validation, 2026-08-04).** All three original open questions are resolved: the amendment count math is superseded by the amendment's derived per-script windows (D11, W-1 — ordering 6, empty-completion 2); D9's ruling is validator-confirmed genuinely spec-silent and correctly forced (recorded in `doc.go`, revisitable only by spec amendment — no longer open); the `CompletionUsage` citation gap is closed by C8 (D10).

- [ ] None blocking. Slice-scheduling watch items live in the delivery table: slice 2's amendment dependency and slice 7's AI-32.1 block.
