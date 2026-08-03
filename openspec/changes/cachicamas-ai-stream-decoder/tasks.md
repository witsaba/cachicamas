# Tasks: AI-27 — Streaming Frame Decoder (`cachicamas-ai-stream-decoder`)

## Sequencing Gate (read before Phase 1)

`backend/agent/src/ai/openaicompat/` does **not exist yet** — AI-25 creates it and AI-25's Slice A is in flight. Slice 1 cannot branch until AI-25's chain lands on tracker `feat/2026-08-03-cachicamas-ai-layer1-wave-4`. If AI-25 slips, AI-27 is **blocked, not re-parented**.

> **Gate satisfied (recorded by slice 1's apply run).** `backend/agent/src/ai/openaicompat/` exists on
> `feat/ai-25c-test-server-viability` (AI-25's landed chain) with `client.go`, `credential.go`,
> `doc.go`, `endpoint.go`, `request.go` and their tests. Slice 1 branched from that tip
> (`feat/ai-27-1-decoder-skeleton`, base commit `0022733`). See Phase 0's evidence entry below.

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (naive) | 930–1,260 |
| Estimated changed lines (corrected, repo's 2–4x undershoot) | 2,000–4,800 |
| 400-line budget risk | High |
| Chained PRs recommended | Yes |
| Suggested split | 7 slices: 1 → 2a → 2b → 3 → 4 → 5 → 6 |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | 27.1 skeleton: one frame, arrival order, pure-function proof | PR1 → tracker | `go test -race -run TestDecoder ./src/ai/openaicompat/...` | N/A — pure unit tests are the harness; no external process | `frame.go`, `decoder.go` (skeleton), `doc.go` ¶, `decoder_test.go` |
| 2a | 27.2 items 1–3: colon/space split, multi-line data join, 3 terminators | PR2a → PR1 | `go test -race -run TestFieldGrammar ./src/ai/openaicompat/...` | N/A | `field_grammar_test.go` (core cases), `decoder.go` growth |
| 2b | 27.2 items 4–7: BOM, empty-data, reset/default, id/retry, ceiling check | PR2b → PR2a | `go test -race -run TestFieldGrammar ./src/ai/openaicompat/...` | N/A | `field_grammar_test.go` (remaining cases), `decoder.go` growth |
| 3 | 27.3 offset sweep + size guard | PR3 → PR2b | `go test -race -run TestOffsetSweep ./src/ai/openaicompat/...` | N/A | `sweep_transcripts_test.go`, `offset_sweep_test.go` |
| 4 | 27.4 keep-alives/unknowns + `[DONE]` verbatim pin | PR4 → PR3 | `go test -race -run TestKeepalive ./src/ai/openaicompat/...` | N/A | `keepalive_test.go`, sweep registry addition |
| 5 | 27.5 bounded memory + `errors.go` | PR5 → PR4 | `go test -race -run TestBoundedMemory ./src/ai/openaicompat/...` | N/A | `bounded_memory_test.go`, `errors.go`, cap enforcement in `decoder.go` |
| 6 | 27.6 EOF discipline + charter-boundary close | PR6 → PR5 | `go test -race -run TestEOF ./src/ai/openaicompat/...` | N/A | `eof_test.go`, `Finish` in `decoder.go` |

Full-suite gate before each PR: `make test` (from `backend/agent/`) green, `make lint` clean. No runtime/manual harness exists or is needed — this milestone is pure byte parsing with zero I/O (Threat Matrix: N/A).

## Coverage

25/25 requirements (`R-ASD-001…025`) and 85/85 scenarios (`S-ASD-001…085`, 72 `[test]` / 13 `[inspection]`) mapped below. No gap found.

## Phase 0 — Preconditions

- [x] 0.1 Confirm `backend/agent/src/ai/openaicompat/` exists on the tracker post-AI-25 merge before branching slice 1; if absent, STOP and report AI-27 blocked.

## Phase 1 — Slice 1: AI-27.1 skeleton (R-ASD-001…003 · S-ASD-001…011)

- [x] 1.1 RED `decoder_test.go`: one well-formed frame in one chunk → exactly one frame, byte-identical payload, no phantom frame (S001–003).
- [x] 1.2 RED: three frames in one chunk yield in order; identical-payload frames not coalesced; one-frame-per-chunk feeding preserves order (S004–006).
- [x] 1.3 RED: decode completes with no network setup; goroutine count unchanged before/after feeding+finish; two independent decoders on the same bytes yield identical frames (S007–009).
- [x] 1.4 GREEN `frame.go` + `decoder.go`: `Frame{Event,Data}`, `NewDecoder`, `Feed`, minimal LF-only scan/dispatch loop, first-colon split (refined in 2a).
- [x] 1.5 GREEN `doc.go` ¶: framing-layer scope, semantic boundary, cap accepted-but-inert-until-27.5. **Placement deviation (see Evidence Log): the paragraph lives as GoDoc on the `Decoder` type in `decoder.go`, not in `doc.go`.**
- [x] 1.6 REFACTOR: extract scan-loop cursor bookkeeping for slice 2a growth.
- [x] 1.7 INSPECTION (no RED phase — recorded confirmation): test sources contain no sleep/timer/wall-clock wait (S010); decoder source imports no transport/HTTP/concurrency package (S011).

### Evidence Log — Slice 1 (AI-27.1)

**Phase 0 gate.** `git -C <worktree> log --oneline -3 feat/ai-27-1-decoder-skeleton` and `... feat/ai-25c-test-server-viability` both resolve to `0022733` (same tip) — the slice-1 branch starts exactly at AI-25's landed chain. `find backend/agent/src/ai/openaicompat -type f` listed `client.go`, `credential.go`, `doc.go`, `endpoint.go`, `request.go`, `ambient_authority_test.go`, `client_test.go`, `credential_test.go`, `endpoint_test.go`, `provider_boundary_test.go`, `request.go`, `timeout_test.go`, `viability_test.go` before any slice-1 file was added.

**Safety net (pre-existing suite).** `make test` from `backend/agent/` before any slice-1 file existed: `ok github.com/cachicamas/backend/agent/src/agenttest`, `ok .../src/ai`, `ok .../src/ai/openaicompat` — all green, establishing the baseline slice 1 must not regress.

**TDD Cycle Evidence**

| Task | Test file | Layer | Safety Net | RED | GREEN | TRIANGULATE | REFACTOR |
|---|---|---|---|---|---|---|---|
| 1.1 (S001–003) | `decoder_test.go` | Unit | N/A (new file) | ✅ `go test -race -run TestDecoder ./src/ai/openaicompat/...` → `undefined: openaicompat.NewDecoder` (2 call sites) | ✅ passed after `frame.go`+`decoder.go` | ✅ ASCII value + non-ASCII multi-byte value (café/heart/emoji) cases | ✅ folded into 1.6 |
| 1.2 (S004–006) | `decoder_test.go` | Unit | — | ✅ same undefined-symbol failure, now 4 call sites | ✅ passed | ✅ 3-in-one-chunk, identical-payload, one-per-chunk cases | ✅ folded into 1.6 |
| 1.3 (S007–009 + copy pin) | `decoder_test.go` | Unit | — | ✅ same undefined-symbol failure, now 9 call sites, "too many errors" truncation | ✅ passed, 9/9 test functions green | ✅ no-network / goroutine-count / two-independent-decoders / copy-pin cases | ✅ folded into 1.6 |
| 1.6 (refactor) | `decoder.go` | — | ✅ 9/9 green before refactor | N/A (no new behavior) | ✅ re-ran `TestDecoder` after extracting `nextLine`: 9/9 still green | N/A | ✅ cursor bookkeeping extracted to `nextLine(buf, start) (line, next, ok)` |
| 1.7 (S010, S011) | inspection | — | — | N/A (inspection has no red phase) | N/A | N/A | N/A |

**Disclosed non-red note.** None needed for this slice: every `[test]` scenario in S001–009 (plus the copy pin) produced a genuine compiler-level RED (`undefined: openaicompat.NewDecoder`) because `Decoder`/`Frame`/`Feed`/`Finish` did not exist before task 1.4's GREEN — the strongest possible falsifiability class, no fabrication risk. Unlike AI-25's slices, no scratch break-and-revert was needed to manufacture a red; the type genuinely did not exist yet.

**Inspection evidence (task 1.7).**
- S-ASD-010: `grep -n -iE "time\.Sleep|time\.After|time\.Tick|time\.NewTimer|time\.NewTicker|<-time\." decoder_test.go` → no matches. Confirmed: no test observes a frame via sleep/timer/wall-clock wait; every assertion reads `Feed`'s or `Finish`'s direct return value.
- S-ASD-011: `grep -n "^import\|^\t\"" decoder.go frame.go` → `decoder.go` imports only `"bytes"`; `frame.go` has no import block at all. Confirmed: no transport, HTTP, or concurrency-primitive import in decoder source.

**GREEN command (full slice, verbose):** `go test -race -v -run TestDecoder ./src/ai/openaicompat/...` → `PASS`, `ok github.com/cachicamas/backend/agent/src/ai/openaicompat 1.378s` (post-refactor run), 9/9 test functions passed:
`TestDecoder_SingleFrameYieldsExactlyOneFrame`, `TestDecoder_MultiByteDataSurvivesByteExact`, `TestDecoder_ThreeFramesInOneChunkYieldInOrder`, `TestDecoder_IdenticalPayloadFramesAreNotCoalesced`, `TestDecoder_OneFramePerChunkPreservesFeedingOrder`, `TestDecoder_CompletesWithNoNetworkSetup`, `TestDecoder_StartsNoGoroutines`, `TestDecoder_TwoIndependentDecodersYieldIdenticalFrames`, `TestDecoder_FrameDataIsACopyUnaffectedByLaterFeed`.

**Full-suite gate:** `make test` (from `backend/agent/`) → exit 0; per-package summary `ok .../src/agenttest`, `ok .../src/ai`, `ok .../src/ai/openaicompat`; 481 `--- PASS` lines, 0 `--- FAIL` lines, `-race` clean throughout (Makefile's `test` target is `go test -race -v ./...`). `make lint` → `golangci-lint run --config=.golangci.yml ./...` → `0 issues.` `go.mod` diff against `feat/ai-25c-test-server-viability`: unchanged, still `module github.com/cachicamas/backend/agent` / `go 1.26.3`, zero `require` block. AI-00 guard `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` and `TestLayer1_ModuleHasNoDependencies_ZeroRequires`: both PASS, unmodified allowlist. AI-25 guard `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources`: PASS — the new non-test sources (`frame.go`, `decoder.go`) are scanned by it and call into none of `os`/`os/exec`/`syscall`/`io/ioutil`.

**Diff size:** `git diff --cached --stat` (3 new files, staged individually by path): `decoder.go` +201, `decoder_test.go` +244, `frame.go` +32 → **3 files changed, 477 insertions(+), 0 deletions**. Within this unit's own pre-forecast band (300–800, see Suggested Work Units above and `design.md`'s Migration/Rollout table); no `size:exception` needed since the chain strategy already resolved the workload decision (`Decision needed before apply: No`).

**Design deviation — `doc.go` not modified (task 1.5).** `design.md`'s File Changes table lists `doc.go` as `Modify` for slice 1 (one paragraph: framing-layer scope, semantic boundary, cap accepted-but-inert-until-27.5). Slice 1's apply run did **not** modify `doc.go`: it already carries AI-25's complete package doc comment, and AI-26 is concurrently developing translation in the same package in a sibling worktree — a second branch that could independently want to touch the same file before either chain converges, which is exactly the collision this apply run's explicit operating constraints were written to avoid. Editing `doc.go` from this branch would risk two independently-evolving branches touching the identical file ahead of the feature-branch-chain's convergence.

Resolution: the same substantive content (framing layer, semantic boundary, "`maxFrameBytes` accepted here, enforced from AI-27.5") is documented instead as GoDoc on the `Decoder` type and `NewDecoder` function in `decoder.go`. Every fact task 1.5 requires is still shipped and reviewer-visible; only its file location changed. Recorded in `design.md` itself under File Changes ("Apply-phase amendment") for continuity into slices 2a–6, so a later slice does not reopen `doc.go` on the same reasoning.

**Diff vs. sibling worktree files brought into this branch.** This apply run also committed `proposal.md`, `design.md`, and `specs/ai-stream-decoder/spec.md` into this worktree/branch for the first time (verbatim from the sdd-spec/sdd-design phase output; not previously committed to any branch — the sibling worktree that authored them still shows them as untracked). No content was altered except: (a) `proposal.md` gained one trailing blockquote flagging the known-stale AI-31 citation without editing the original text (Phase 7 task 7.2 still owns the actual correction); (b) `design.md` gained the "Apply-phase amendment" note above and marked its first Open Question resolved for this run; (c) `tasks.md` (this file) gained its checkboxes and this Evidence Log. `specs/ai-stream-decoder/spec.md` is unmodified verbatim.

## Phase 2a — Slice 2a: AI-27.2 items 1–3 (R-ASD-004…006 · S-ASD-012…024)

- [ ] 2a.1 RED `field_grammar_test.go`: colon-in-value splits at first colon only; exactly one leading space stripped (two-space case retains the second); no-space value verbatim; colonless line = field named by whole line, empty value; unrecognized colonless name ignored (S012–016).
- [ ] 2a.2 RED: three data lines join with one LF each, no trailing LF; single data line no trailing LF; empty last line preserves interior LF; content-embedded LF survives (S017–020).
- [ ] 2a.3 RED: CRLF/LF/lone-CR of the same transcript decode identically; CRLF-only produces no empty frame; mixed terminators accumulate uniformly; CRLF split across chunks reconstructs with no injected blank line (S021–024).
- [ ] 2a.4 GREEN `decoder.go`: full first-colon/one-`if`-space rule, data-join-then-trim-at-dispatch, three-terminator handling via retained-tail deferred-final-CR (no boolean flag).
- [ ] 2a.5 REFACTOR: consolidate terminator resolution into a helper shared with slice 2b's BOM prefix-wait logic.

## Phase 2b — Slice 2b: AI-27.2 items 4–7 (R-ASD-007…011 · S-ASD-025…040, 084)

- [ ] 2b.1 RED: one leading BOM stripped once; two leading BOMs → one stripped, second survives as content; mid-value BOM survives; BOM split across chunks at offsets 1/2 still stripped once (S025–028).
- [ ] 2b.2 RED: two blank lines only → zero frames; event-type line + blank/no data → zero frames; blank-framed well-formed frame → exactly one frame (S029–031).
- [ ] 2b.3 RED: second typeless frame after a typed one defaults, never inherits; second frame's payload carries none of the first's data; suppressed frame leaves no type residue; **empty-valued `event:` line also defaults, never emits `""`** (S032–034, S084).
- [ ] 2b.4 RED: id line (ordinary value + NUL-bearing value) and retry line (digit + non-digit value) each produce frames identical to the same transcript without that line (S035–038).
- [ ] 2b.5 GREEN `decoder.go`: `bomChecked` prefix-wait state; empty-data-buffer no-dispatch+reset; post-dispatch reset making an empty-eventType-buffer indistinguishable from never-set; id/retry parsed then discarded, no state carried.
- [ ] 2b.6 REFACTOR: fold dispatch-reset into one function called from both the empty-data path and the normal-dispatch path.
- [ ] 2b.7 INSPECTION: exactly seven grammar items shipped, each mapped 1:1 to a doc-0002 item (S039); rule recorded that an eighth case forces an explicit `AI-27.2.1`/`.2` living-graph node split, never silent absorption — verified N/A at this milestone (S040).

> **Note for slice 2b's own apply run**: slice 1's naive `default:`-falls-through field-name switch in `decoder.go` already treats an empty-string field name (a comment line's colon-at-position-0 case) and any unrecognized name identically to `id`/`retry` — ignored, no accumulation disturbed. 2b's RED tests for S035–038 may find this already green from slice 1's implementation rather than genuinely red; disclose that plainly rather than reporting a fabricated red, per this milestone's evidence discipline.

## Phase 3 — Slice 3: AI-27.3 chunk-boundary re-entrancy (R-ASD-012…014 · S-ASD-041…051)

- [ ] 3.1 RED `sweep_transcripts_test.go`: declare `sweepTranscripts` registry (LF-only, CRLF, lone-CR, mixed, BOM-prefixed, multi-line-data) + `maxSweepFixtureBytes = 1024` constant.
- [ ] 3.2 RED — **guard bite-proof, its own evidence**: stage one deliberately overweight (>1024-byte) fixture in the registry; run the size-guard assertion in `offset_sweep_test.go` and record it failing loudly — this staged red IS the guard's RED phase, since a guard over only clean fixtures is green from birth (S049 first half).
- [ ] 3.3 GREEN: remove the overweight fixture; guard now passes for all real fixtures, `size ≤ 1024` inclusive proven for every registered transcript (S049 second half).
- [ ] 3.4 RED `offset_sweep_test.go`: every registered transcript replayed at every offset `0..len`, two-chunk split, deep-equal vs. canonical unsplit decode; split inside a field name; split inside a multi-byte data char; BOM-prefixed transcript split at offsets 1 and 2 (S041–044).
- [ ] 3.5 RED: CRLF split exactly between CR and LF (general case + at a frame-terminating blank line) yields no phantom/duplicate dispatch; lone CR as chunk-final byte with no following LF terminates exactly once (S046–048).
- [ ] 3.6 GREEN: implement split-state carryover (retained tail, `data`/`eventType` accumulators) proven correct across all offsets by 3.4/3.5.
- [ ] 3.7 REFACTOR: extract the deep-equal comparison helper for reuse by slice 5's representative-offset stand-in.
- [ ] 3.8 INSPECTION: sweep mismatch path names the offending byte offset in its failure message (S045 — upgrade path: becomes `[test]` only if the message construction is factored into a separately callable pure function, not provided by this design); reviewer trace confirms the multi-megabyte fixture is never enumerated by the sweep (S051).
- [ ] 3.9 Cross-slice note: `R-ASD-014`'s remaining fact — the multi-MB fixture's non-exhaustive stand-ins — is discharged in Phase 5 (5.3), once that fixture exists. Its absence here is not a gap.

## Phase 4 — Slice 4: AI-27.4 keep-alives and unknowns (R-ASD-015…017 · S-ASD-052…057; R-ASD-025 partial · S-ASD-079…081)

- [ ] 4.1 RED `keepalive_test.go`: comment lines interleaved with data lines leave the payload identical to the comments-removed version; comment-only stream ending at a line boundary yields zero frames, no error (S052–053).
- [ ] 4.2 RED: invented field name ignored without residue; unrecognized event-type name yielded verbatim (S054–055).
- [ ] 4.3 RED: an explicit event-type line inserted into a data-only transcript sets that frame's type only; surrounding frames keep the default; no error (S057).
- [ ] 4.4 RED (charter-boundary pin): `data: [DONE]` yielded as an ordinary frame with the exact string payload, no special status/early finish/error; frames following the sentinel frame are still yielded normally; two frames differing only in payload content decode with identical framing outcomes (S079–081).
- [ ] 4.5 GREEN: comment-line (leading `:`) short-circuit ahead of field dispatch; no event-type registry/allowlist introduced — faithful compliance already yields S055.
- [ ] 4.6 GREEN: register the comment-interleaved fixture into `sweepTranscripts` (grows the slice-3 registry).
- [ ] 4.7 REFACTOR: none required — the comment short-circuit is a one-line guard.
- [ ] 4.8 INSPECTION: no event-type registry/allowlist/recognition table exists in shipped source (S056).

> **Note for slice 4's own apply run**: slice 1's field-line split already resolves a colon-at-position-0 line (a comment) to an empty field name that falls through the `default:` case unrecognized — the same path id/retry/unknown fields take. 4.1/4.5's RED may find comment-ignoring already green from slice 1 rather than genuinely red; disclose plainly if so.

## Phase 5 — Slice 5: AI-27.5 bounded memory (R-ASD-018…021 · S-ASD-058…069, 085)

- [ ] 5.1 RED `bounded_memory_test.go`: multi-megabyte frame below cap decodes byte-exact in one chunk, and fed in many small chunks; a frame above 64 KiB and below cap succeeds — no implicit sub-cap limit (S058–060).
- [ ] 5.2 RED: accumulation strictly greater than cap aborts, no frame for that over-cap frame, `errors.Is(err, ErrFrameTooLarge)` true / `ErrTruncated` false; one chunk carrying a complete frame followed by an over-cap frame returns the complete frame together with the error (S061, S085 — cap-side collateral pin, mirrors the truncation-side S075); exactly-at-cap frame decodes, boundary inclusive (S063); `Category(err)` reports malformed-response and `errors.Is(err, ai.ErrMalformedResponse)` holds (S062).
- [ ] 5.3 RED — **discharges R-ASD-014's remaining stand-in facts**: the multi-MB fixture replayed split at a small fixed set of representative offsets, not exhaustively (S050); confirm it is never registered in `sweepTranscripts`.
- [ ] 5.4 RED: `NewDecoder(0)` selects `DefaultMaxFrameBytes` (8 MiB); a multi-MB frame under that default decodes correctly.
- [ ] 5.5 RED: cap-exceeded error does not match the truncation identity and vice versa; each error matches its own identity while both report the malformed-response category (S065–067).
- [ ] 5.6 GREEN `errors.go`: `ErrFrameTooLarge`, `ErrTruncated`, each wrapping `ai.ErrMalformedResponse` via `%w`; `Category(err) (ai.FailureCategory, bool)`.
- [ ] 5.7 GREEN `decoder.go`: cap check (`> cap`, never `>=`) inside the scan loop against `tail+data+eventType`; poisoning after the first terminal error; pre-trip frames returned together with the error.
- [ ] 5.8 REFACTOR: consolidate the cap-check call site so the under/at-cap and over-cap paths run the identical counter.
- [ ] 5.9 INSPECTION: no provider-failure (`ai.Failure`/`ai.MidStreamFailure`) construction call exists in decoder source (S064); a recorded deferral names the append-only property, the second consumer (AI-30.1 item 4), and the owning milestone for the tenth-category question (S068); diff of `ai-provider-errors` spec and source against base shows no modification (S069).

> **Note for slice 5's own apply run**: `NewDecoder` already stores `maxFrameBytes` (normalized, `<=0` → `DefaultMaxFrameBytes`) as of slice 1 — 5.4's "NewDecoder(0) selects the default" assertion may already read the right stored value and only need a new test, not new production code, to go green. The cap-enforcement check itself (5.7) is genuinely new: slice 1's scan loop has no cap comparison anywhere.

## Phase 6 — Slice 6: AI-27.6 EOF discipline + charter-boundary close (R-ASD-022…024 · S-ASD-070…078; R-ASD-025 close · S-ASD-082…083)

- [ ] 6.1 RED `eof_test.go`: clean end at a frame boundary + terminating blank line finishes no-error; empty stream finishes zero frames, no error; comment-only stream ending at a line boundary finishes clean (S070–072).
- [ ] 6.2 RED: cut mid-data-value at `Finish` reports truncation, no partial frame; complete data lines but no terminating blank line reports truncation, frame not dispatched; two complete frames + partial third → both complete frames yielded, third absent, truncation reported (S073–075).
- [ ] 6.3 RED: truncation error's named category is malformed-response and distinguishable from cap-exceeded per R-ASD-020 (S076).
- [ ] 6.4 RED: frames fed but `Finish` not yet called expose only already-dispatched frames, no error reported (S077).
- [ ] 6.5 GREEN `decoder.go` `Finish()`: resolve a pending buffer-final CR as lone-CR first; any retained line bytes or non-empty accumulator → `ErrTruncated`, partial discarded; else nil.
- [ ] 6.6 REFACTOR: share the CR-resolution helper between `Feed`'s deferred-final-CR path and `Finish`.
- [ ] 6.7 INSPECTION: no transport EOF value (`io.EOF`/`io.ErrUnexpectedEOF`) is produced or returned by the decoder (S078).
- [ ] 6.8 INSPECTION (milestone close): the terminal-sentinel literal appears only in test fixtures, never in decoder logic (S082); `go.mod`/`go.sum` diff against base is unchanged — zero requires held throughout (S083).

> **Note for slice 6's own apply run**: slice 1's `Finish()` is a stub returning `nil` unconditionally and does not yet inspect `d.buf`/`d.data`/`d.eventType`. 6.1's "clean end returns no error" RED tests may already pass against the stub (disclose, don't fabricate); 6.2's truncation-detection tests are genuinely new — the stub has no code path that can return `ErrTruncated` yet.

## Phase 7 — Outbound coordination notes

- [ ] 7.1 Record the `S-ASD-045` upgrade path in the change's follow-up notes: if a future design factors the sweep's mismatch-message construction into a separately callable pure function, `S-ASD-045` becomes `[test]` and the tally shifts to 73 `[test]` / 12 `[inspection]` — do not lose this when design is revised.
- [ ] 7.2 Flag for correction before archive: `proposal.md` cites AI-31 as the second consumer of the resource-exhaustion concept; doc 0002, the spec, and the design all correctly place it at AI-30.1 item 4 — fix the proposal, do not propagate the AI-31 error.
