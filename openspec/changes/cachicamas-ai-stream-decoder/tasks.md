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

- [x] 2a.1 RED `field_grammar_test.go`: colon-in-value splits at first colon only; exactly one leading space stripped (two-space case retains the second); no-space value verbatim; colonless line = field named by whole line, empty value; unrecognized colonless name ignored (S012–016).
- [x] 2a.2 RED: three data lines join with one LF each, no trailing LF; single data line no trailing LF; empty last line preserves interior LF; content-embedded LF survives (S017–020).
- [x] 2a.3 RED: CRLF/LF/lone-CR of the same transcript decode identically; CRLF-only produces no empty frame; mixed terminators accumulate uniformly; CRLF split across chunks reconstructs with no injected blank line (S021–024).
- [x] 2a.4 GREEN `decoder.go`: full first-colon/one-`if`-space rule, data-join-then-trim-at-dispatch, three-terminator handling via retained-tail deferred-final-CR (no boolean flag).
- [x] 2a.5 REFACTOR: consolidate terminator resolution into a helper shared with slice 2b's BOM prefix-wait logic.

### Evidence Log — Slice 2a (AI-27.2 items 1–3)

**Disclosed non-red scenarios (S012–020, 9 of the slice's 13).** Before any 2a production change, `go test -race -run TestFieldGrammar ./src/ai/openaicompat/...` (run against slice 1's committed `decoder.go` immediately after `field_grammar_test.go` was written) already showed these 9 scenarios passing. Slice 1's `splitField` already implemented the complete first-colon/one-`if`-space rule (R-ASD-004), and `processFieldLine`+`dispatch` already implemented the complete data-join-then-trim-at-dispatch rule (R-ASD-005) — there was nothing left for task 2a.4 to make green for these nine.

Falsifiability was demonstrated instead by scratch break-and-revert cycles against `decoder.go`, run one at a time and reverted immediately after observing the failure (`git diff` against the committed file confirmed empty after each):
- `for` loop replacing the one-`if` space-strip → falsified `TestFieldGrammar_TwoLeadingSpacesRetainsSecondSpace` (S013) in isolation; no effect on S012/014–020.
- `bytes.TrimRight(d.data, "\n")` replacing the single-LF strip → falsified S019 and S020 together; no effect on S017/018 (exactly one trailing LF each, so trim-all and strip-one coincide there).
- Removing dispatch's strip entirely → falsified **all nine** of S012–020 at once (every data-bearing frame carries an unstripped trailing LF) — broader than the two targeted cycles above predicted, matching this wave's established pattern of a combined break surfacing more failures than expected.
- `bytes.LastIndexByte` replacing `bytes.IndexByte` for the colon search → falsified S012 in isolation.
- Swapping the colonless fallback's `return line, nil` to `return nil, line` → falsified S015 (after its fixture was strengthened, see below) in isolation; no effect on S016.
- Adding a `default:` case that misroutes unrecognized field names into the data accumulator → falsified S016 in isolation; no effect on S015.

**S015 fixture strengthened before use.** A literal single-line fixture (`"data\n\n"`) cannot distinguish "colonless `data` correctly recognized, contributed an empty value" from "not recognized at all," because dispatch is still unconditional in this slice (R-ASD-008 is Phase 2b's rule) — both readings yield an empty `Data`. Strengthened to `"data\ndata: second\n\n"` (expects `"\nsecond"`), which only survives if the colonless line was genuinely routed to the data accumulator.

**S024 self-referential comparison caught and fixed before commit.** The first draft compared a split decode only to a same-code "unsplit reference" decode. Pre-fix, the CRLF blank line in `"data: x\r\n\r\n"` was misparsed as a one-byte `"\r"` content line by both the split feed and the reference feed identically, so both produced zero frames and the equality check passed for the wrong reason — caught by actually running the test, not by reasoning about it alone. Fixed by asserting the reference decode against a hard-coded `[]Frame{{Event:"message",Data:"x"}}` before comparing the split decode to it, which made the fixture genuinely RED pre-fix.

**Genuine RED (S021–024, R-ASD-006).** Confirmed by an actual pre-2a.4 run: `TestFieldGrammar_ThreeTerminatorStylesDecodeIdentically`'s `/CRLF` and `/lone_CR` sub-tests FAIL (its `/LF` sub-test passes); `TestFieldGrammar_CRLFOnlyProducesNoEmptyFrame` FAILS (0 frames, want 1); `TestFieldGrammar_MixedTerminatorsAccumulateUniformly` FAILS (0 frames, want 1); `TestFieldGrammar_CRLFSplitAcrossChunksReconstructsNoBlankLine` FAILS once its assertion was hard-coded. Root cause: pre-fix `nextLine` located only `'\n'`; a CRLF blank line was returned as a one-byte `"\r"` line (non-empty, no dispatch), and a lone-CR-only transcript never resolved any line (no `'\n'` byte exists anywhere in it).

**GREEN (task 2a.4).** `nextLine` rewritten around `bytes.IndexAny(rest, "\r\n")` for the earliest terminator candidate of either kind, delegating a found CR to a new `resolveCR` helper: 2-byte terminator (CRLF) when the CR's immediate next byte is `'\n'`, 1-byte (lone CR) otherwise, `ok=false` — deferred, never guessed — when no next byte exists yet in the retained buffer. `Feed`'s scan loop itself was not touched, matching slice 1's own forward note that `nextLine` was extracted precisely so this could grow without reshaping the loop.

**REFACTOR (task 2a.5).** `resolveCR` is the "boundary visible but not yet resolvable without one more byte" shape extracted for reuse; its doc comment names slice 2b's BOM prefix-wait check (a fixed 3-byte prefix, not a 1-byte lookahead) as the next grower of the same wait-rather-than-guess idea. No BOM code was written here — that remains slice 2b's own scope.

**Full-suite gate:** `make test` (from `backend/agent/`) → exit 0; `ok .../src/agenttest`, `ok .../src/ai`, `ok .../src/ai/openaicompat`; 494 top-level `--- PASS` lines (481 slice-1 baseline + 13 new `TestFieldGrammar_*` top-level functions), 0 `--- FAIL`, `-race` clean throughout. `make lint` → `golangci-lint run --config=.golangci.yml ./...` → `0 issues.` `go.mod` diff against `feat/ai-27-1-decoder-skeleton`: unchanged, still `module github.com/cachicamas/backend/agent` / `go 1.26.3`, zero `require`, no `go.sum`. AI-00 guards `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` and `TestLayer1_ModuleHasNoDependencies_ZeroRequires`: both PASS, no allowlist edit. AI-25 guard `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources`: PASS — `decoder.go`'s only import remains `"bytes"`.

**Diff size:** `git diff feat/ai-27-1-decoder-skeleton --stat` (2 files, staged individually by path): `decoder.go` +47/-9, `field_grammar_test.go` +366 new → **2 files changed, 413 insertions(+), 9 deletions(-)**. Within this unit's own pre-forecast band (300–800).

## Phase 2b — Slice 2b: AI-27.2 items 4–7 (R-ASD-007…011 · S-ASD-025…040, 084)

- [x] 2b.1 RED: one leading BOM stripped once; two leading BOMs → one stripped, second survives as content; mid-value BOM survives; BOM split across chunks at offsets 1/2 still stripped once (S025–028).
- [x] 2b.2 RED: two blank lines only → zero frames; event-type line + blank/no data → zero frames; blank-framed well-formed frame → exactly one frame (S029–031).
- [x] 2b.3 RED: second typeless frame after a typed one defaults, never inherits; second frame's payload carries none of the first's data; suppressed frame leaves no type residue; **empty-valued `event:` line also defaults, never emits `""`** (S032–034, S084).
- [x] 2b.4 RED: id line (ordinary value + NUL-bearing value) and retry line (digit + non-digit value) each produce frames identical to the same transcript without that line (S035–038).
- [x] 2b.5 GREEN `decoder.go`: `bomChecked` prefix-wait state; empty-data-buffer no-dispatch+reset; post-dispatch reset making an empty-eventType-buffer indistinguishable from never-set; id/retry parsed then discarded, no state carried.
- [x] 2b.6 REFACTOR: fold dispatch-reset into one function called from both the empty-data path and the normal-dispatch path.
- [x] 2b.7 INSPECTION: exactly seven grammar items shipped, each mapped 1:1 to a doc-0002 item (S039); rule recorded that an eighth case forces an explicit `AI-27.2.1`/`.2` living-graph node split, never silent absorption — verified N/A at this milestone (S040).

> **Note for slice 2b's own apply run**: slice 1's naive `default:`-falls-through field-name switch in `decoder.go` already treats an empty-string field name (a comment line's colon-at-position-0 case) and any unrecognized name identically to `id`/`retry` — ignored, no accumulation disturbed. 2b's RED tests for S035–038 may find this already green from slice 1's implementation rather than genuinely red; disclose that plainly rather than reporting a fabricated red, per this milestone's evidence discipline.

### Evidence Log — Slice 2b (AI-27.2 items 4–7)

**Disclosed non-red scenarios (9 of the slice's 15).** Before any 2b production change, `go test -race -run TestFieldGrammar ./src/ai/openaicompat/...` (run immediately after all 2b test functions were written, against slice 2a's committed `decoder.go`) already showed these 9 passing:
- **S032, S033** (second frame defaults/carries no residue) — slice 2a's `dispatch()` already called `d.reset()` unconditionally after every dispatch, so a second frame's event-type and data accumulators were already empty before it started.
- **S035–038** (id/retry ignored) — slice 1's field-name `switch` has no `case "id"`/`case "retry"`; both fall through unrecognized exactly like any invented name, per the disclosure note above.
- **S084** (empty-valued `event:` line defaults) — slice 1's dispatch already tested `len(d.eventType) > 0`, not a "was-assigned" flag, so an empty-valued `event:` line already left the accumulation reading as never-set.

Falsifiability was demonstrated by 5 scratch break-and-revert cycles against `decoder.go` (each applied, run, observed, reverted; `git diff` confirmed empty after each):
1. Routing `id`/`retry` into the data accumulator (`case "data", "id", "retry":`) → falsified **all four** of S035–038 at once.
2. Removing the `d.reset()` call from `dispatch()`'s non-empty-data branch → falsified S032 and S033 together.
3. Guarding the `event:` case to skip reassignment when the value is empty (`if len(value) > 0 { … }`) → falsified S084 in isolation — Event leaked the stale prior value (`"custom"`) instead of defaulting.
4. Changing `stripBOM` to strip every leading BOM occurrence in a loop (over-stripping) → falsified **S026 only**; S025/027/028 stayed green, proving S026's real, distinct job (an over-stripping guard) is independent of S025's job (an under-stripping guard, already proven genuinely RED against the pre-2b baseline below).
5. Changing `stripBOM` to strip a BOM occurrence found anywhere in the buffer, not only at its start (`bytes.Index` instead of `bytes.HasPrefix`) → falsified **S027 only** — a mid-value mark got incorrectly removed.

**A third vacuous-pass trap, found only by running the suite (not predicted by the brief, which flagged only the empty-`event:` case).** `TestFieldGrammar_TwoLeadingBOMsStripsOnlyOneSecondSurvivesAsContent` (S026) passed **before any BOM code existed at all** — with zero BOMs stripped, the "event" field name is corrupted by 6 leftover bytes instead of 3, but corruption-by-any-nonzero-amount collapses to the same "unrecognized, defaults to message" outcome either way, so this fixture alone cannot tell "exactly one stripped" from "none stripped". Cycle 4 above proves what S026 **can** tell: not-more-than-one. S025 (single BOM, hard-coded expected frame, genuinely RED pre-fix — see below) is what proves at-least-one. The two scenarios are complementary by design, not individually self-sufficient, and are documented as such in the test file.

**Genuine RED (S025, S028, S029–031, S034), confirmed by the actual pre-2b run.** `TestFieldGrammar_SingleLeadingBOMStrippedOnce` failed (`Event = "message"`, want `"greet"` — the mark was left as content, corrupting the field name). `TestFieldGrammar_LeadingBOMSplitAcrossChunksStillStrippedOnce` failed both subtests (`Data = []`, want `"hi"` — same corruption, reached via a split-across-chunks feed). `TestFieldGrammar_TwoBlankLinesOnlyYieldZeroFrames` failed (2 frames, want 0). `TestFieldGrammar_EventTypeLineWithNoDataYieldsZeroFrames` failed (1 frame, want 0). `TestFieldGrammar_WellFormedFrameSurroundedByBlankLinesYieldsExactlyOne` failed (4 frames, want 1). `TestFieldGrammar_SuppressedTypeOnlyFrameLeavesNoTypeResidue` failed (2 frames, want 1) — slice 2a's `dispatch()` always yielded a frame, even for an empty data accumulation, so a type-only frame dispatched instead of being suppressed. Root cause for all: slice 2a had no BOM handling at all and no empty-data-accumulation check in `dispatch()`.

**GREEN (task 2b.5).** `decoder.go` gained: a `bomChecked bool` field and `stripBOM()` method (growing `resolveCR`'s "wait for one more byte, never guess" shape to a fixed 3-byte prefix, called from `Feed` right after appending new bytes); `dispatch()` changed to `(Frame, bool)`, returning `false` and resetting state when `d.data` is empty (checked on the raw, pre-trim accumulation, matching § 9.2's own step ordering — the emptiness check precedes the separate trailing-LF-removal step); `Feed`'s blank-line branch updated to only append a frame when `dispatched` is true. `processFieldLine`'s doc comment updated from slice 2a's forward-looking phrasing to a present-tense statement that id/retry are pinned by test, not left as an accident of the fallthrough.

**REFACTOR (task 2b.6).** `dispatch()`'s two `d.reset()` call sites (one per branch) consolidated into a single `defer d.reset()` at the top of the function — one place a future edit could forget to reset, not two. Verified safe: `Frame.Data` is a fresh `append([]byte(nil), …)` copy and `Event` a fresh `string(…)` conversion, both completed before the deferred reset runs, so no aliasing with the reset buffers. Re-ran `TestFieldGrammar_*` (28 functions/32 cases) after the refactor: all still green.

**Lint caught a real simplification.** `golangci-lint`'s `staticcheck` (S1017) flagged `stripBOM`'s `if bytes.HasPrefix(d.buf, bom) { d.buf = d.buf[len(bom):] }` as replaceable by the unconditional `d.buf = bytes.TrimPrefix(d.buf, bom)` (identical semantics — `TrimPrefix` returns the slice unchanged when the prefix does not match). Applied; re-verified `0 issues.` and full suite still green afterward.

**Inspection evidence (task 2b.7).**
- S-ASD-039: `spec.md`'s AI-27.2 section carries exactly seven requirements with grammar content — `R-ASD-004` (split), `005` (join), `006` (terminators), `007` (BOM), `008` (empty-dispatch), `009` (reset/default), `010` (id/retry) — each mapping 1:1 to one of doc 0002's seven grammar items. `R-ASD-011` is the ceiling/governance rule itself, not an eighth item.
- S-ASD-040: no eighth grammar case was discovered during this slice's implementation (BOM, empty-dispatch, reset/default and id/retry all fit cleanly within the four items assigned, `R-ASD-007`…`010`) — the "recorded node split" condition is correctly not triggered. N/A at this milestone, confirmed rather than assumed.

**Full-suite gate:** `go test -race -v -count=1 ./...` (from `backend/agent/`) → exit 0; `ok .../src/agenttest` (2.1s), `ok .../src/ai` (3.7s), `ok .../src/ai/openaicompat` (3.3s). **509** top-level `--- PASS` lines (494 slice-2a baseline + 15 new top-level `TestFieldGrammar_*` functions: 4 BOM + 3 empty-dispatch + 4 reset/default + 4 id/retry), **0** `--- FAIL`, `-race` clean throughout. `make lint` → `0 issues.` (after the `TrimPrefix` fix above). `go.mod` diff against `feat/ai-27-2a-field-grammar-core`: unchanged, still `module github.com/cachicamas/backend/agent` / `go 1.26.3`, zero `require`, no `go.sum`. AI-00 guards `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` and `TestLayer1_ModuleHasNoDependencies_ZeroRequires`: both PASS, no allowlist edit. AI-25 guard `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources`: PASS — `decoder.go`'s only import remains `"bytes"`.

**Diff size:** `git diff feat/ai-27-2a-field-grammar-core --stat` (2 files, staged individually by path): `decoder.go` +89/−13 (net, includes the `TrimPrefix` fix), `field_grammar_test.go` +402 new → **2 files changed, 478 insertions(+), 13 deletions(-)**. Within this unit's own pre-forecast band (300–800).

**Discovered fact for slice 6 (Finish/EOF discipline) — see the refreshed note below.** BOM state (`bomChecked`, a pending 1–2-byte partial mark prefix) needs no special-casing in `Finish()`: a not-yet-resolved partial BOM prefix simply sits in `d.buf`, so slice 6's already-planned "any retained line bytes → `ErrTruncated`" check (6.5) covers it for free via the same `len(d.buf) > 0` test it already needs for an ordinary unterminated line. Design.md's "a 1–2-byte BOM prefix pending at Finish is a partial line → truncation" is therefore already satisfied by the planned `Finish()` shape, not an extra code path slice 6 needs to add.

## Phase 3 — Slice 3: AI-27.3 chunk-boundary re-entrancy (R-ASD-012…014 · S-ASD-041…051)

- [x] 3.1 RED `sweep_transcripts_test.go`: declare `sweepTranscripts` registry (LF-only, CRLF, lone-CR, mixed, BOM-prefixed, multi-line-data) + `maxSweepFixtureBytes = 1024` constant.
- [x] 3.2 RED — **guard bite-proof, its own evidence**: stage one deliberately overweight (>1024-byte) fixture in the registry; run the size-guard assertion in `offset_sweep_test.go` and record it failing loudly — this staged red IS the guard's RED phase, since a guard over only clean fixtures is green from birth (S049 first half).
- [x] 3.3 GREEN: remove the overweight fixture; guard now passes for all real fixtures, `size ≤ 1024` inclusive proven for every registered transcript (S049 second half).
- [x] 3.4 RED `offset_sweep_test.go`: every registered transcript replayed at every offset `0..len`, two-chunk split, deep-equal vs. canonical unsplit decode; split inside a field name; split inside a multi-byte data char; BOM-prefixed transcript split at offsets 1 and 2 (S041–044).
- [x] 3.5 RED: CRLF split exactly between CR and LF (general case + at a frame-terminating blank line) yields no phantom/duplicate dispatch; lone CR as chunk-final byte with no following LF terminates exactly once (S046–048).
- [x] 3.6 GREEN: implement split-state carryover (retained tail, `data`/`eventType` accumulators) proven correct across all offsets by 3.4/3.5.
- [x] 3.7 REFACTOR: extract the deep-equal comparison helper for reuse by slice 5's representative-offset stand-in.
- [x] 3.8 INSPECTION: sweep mismatch path names the offending byte offset in its failure message (S045 — upgrade path: becomes `[test]` only if the message construction is factored into a separately callable pure function, not provided by this design); reviewer trace confirms the multi-megabyte fixture is never enumerated by the sweep (S051).
- [x] 3.9 Cross-slice note: `R-ASD-014`'s remaining fact — the multi-MB fixture's non-exhaustive stand-ins — is discharged in Phase 5 (5.3), once that fixture exists. Its absence here is not a gap.

### Evidence Log — Slice 3 (AI-27.3 chunk-boundary re-entrancy)

**Design-anticipated disclosure, confirmed by the actual pre-3.x run.** design.md's own File Changes table does **not** list `decoder.go` as grown by Slice 3 (it lists 2a/2b/5/6 only) — the milestone's own design predicted this slice ships a proof harness, not new production code, because Slice 2a's `resolveCR` (deferred-final-CR, never guessed) and Slice 2b's `stripBOM` (fixed 3-byte prefix-wait) already carry the cross-feed state this property needs structurally. Confirmed, not assumed: `go test -race -v -run TestOffsetSweep ./src/ai/openaicompat/...` passed **all 7 new top-level functions** (including all 6 sub-tests of `TestOffsetSweep_EveryOffsetMatchesCanonicalDecode`) on the very first run against the unmodified Slice-2b `decoder.go` — zero production changes were needed for 3.4/3.5/3.6.

**Falsifiability demonstrated via 3 scratch break-and-revert cycles against `decoder.go`** (each applied, run, observed, reverted; `git diff --stat` against HEAD confirmed empty after each):

1. **`resolveCR`'s deferred-CR wait defeated** (`if crPos+1 >= len(rest) { return 0, false }` → `return 1, true`, guessing lone-CR instead of waiting for one more byte). Caught by: the general sweep's `"crlf"` and `"mixed-terminators"` entries, both failing at byte offset 13 (`event: greet\r|\ndata:...`, exactly the CR/LF boundary) with `frames[0].Event` corrupted from `"greet"` to `"message"` — the phantom early blank-line dispatch fires a premature `reset()` that wipes the already-set event type before the real data line is even reached; and by `TestOffsetSweep_CRLFSplitBetweenCRAndLFAtFrameTerminatingBlankLine` (S047), which caught it via a different symptom (1 frame dispatched one Feed call too early). **A first draft of the S046 dedicated test ("data: x\r" | "\n\n") passed vacuously under this exact break** — with no second data line after the split, the phantom dispatch's premature reset had nothing observable left to corrupt, so the net result coincidentally matched the correct one. Caught only by actually running it under the break, not by reasoning about the fixture in isolation. Fixed by adding a second data line ("data: first\r" | "\ndata: second\n\n"), which turns the break into an observably wrong 2-frame split instead of 1 joined frame — this is the version shipped.
2. **`stripBOM`'s prefix-wait defeated** (early-return `bomChecked = true` whenever `len(d.buf) < len(bom)`, regardless of whether the partial buffer is a proper BOM prefix). Caught by: the general sweep's `"bom-prefixed"` entry, failing at exactly offsets 0, 1 and 2 (the only offsets whose first chunk is shorter than the 3-byte mark) plus the byte-at-a-time replay; and independently by the pre-existing Slice-2b regression test `TestFieldGrammar_LeadingBOMSplitAcrossChunksStillStrippedOnce` (S-ASD-028), both its `split_at_offset_1` and `split_at_offset_2` subtests failing too — cross-validated by both the old and new suites.
3. **Accumulator carryover defeated** (`d.data = d.data[:0]; d.eventType = d.eventType[:0]` added at the top of `Feed`, simulating "forgot state must survive across `Feed` calls"). Caught **broadly**: all 6 of the general sweep's registry entries failed (every transcript needs carryover somewhere across its offset range), plus `TestOffsetSweep_CRLFSplitBetweenCRAndLFAtFrameTerminatingBlankLine`. This is the strongest single break, proving Task 3.6's "split-state carryover... proven correct across all offsets" claim is backed by comprehensive, not narrow, falsification.

**Guard bite-proof (task 3.2/3.3), its own dedicated cycle.** A deliberately overweight fixture (`bytes.Repeat([]byte("x"), maxSweepFixtureBytes+1)`, 1025 bytes) was staged directly in the `sweepTranscripts` registry; `go test -race -v -run TestOffsetSweep_FixturesWithinSweepBound` failed loudly: `offset_sweep_test.go:32: transcript "OVERWEIGHT-STAGED-FOR-BITE-PROOF" is 1025 bytes, want <= 1024 (S-ASD-049)`, exit 1 — this IS the guard's RED phase; a guard exercised only against clean fixtures would be green from birth and prove nothing. The fixture and its temporary `"bytes"` import were then removed; the guard passed again for the real, clean registry. This also mechanically confirms the off-by-one: 1025 fails, and every real registered fixture (29–73 bytes, all well under the bound) passes.

**Anti-vacuous-pass anchoring (this slice's specific defense against the milestone's #2 known trap — comparing a decode only to a same-code reference).** All 6 registry entries set `want`, not just the spec's required minimum of one: `sweepTemplateFrames()` hard-codes the same 2-frame expectation for the `lf-only`/`crlf`/`lone-cr`/`mixed-terminators` renderings of one logical template (mirroring `field_grammar_test.go`'s own `wantGrammarFrames` three-rendering pattern), and `bom-prefixed`/`multi-line-data` each carry their own literal. `TestOffsetSweep_CanonicalDecodeMatchesHardCodedFrames` asserts every one of them and additionally self-checks that at least one anchor exists (`anchored == 0` → `t.Fatal`), so a future edit cannot silently strip every anchor without the harness itself objecting.

**S-ASD-045 upgrade taken.** The mismatch path's message construction was factored into a separately callable pure function (`sweepMismatch(transcriptName string, offset int, got, want []Frame) (msg string, mismatched bool)`), turning S-ASD-045 from `[inspection]` into a runnable `[test]`: `TestOffsetSweep_MismatchMessageNamesOffendingByteOffset` asserts the constructed message contains both the transcript name and the exact byte offset, and that no message is produced for identical sequences. **Per spec.md's own recorded upgrade condition, the milestone-wide tally shifts to 73 `[test]` / 12 `[inspection]` of 85** — `spec.md` itself is NOT edited by this apply run (out of Slice 3's assigned scope; the formal tally revision belongs to Phase 7's coordination work, task 7.1, which already anticipated exactly this trigger). This paragraph is the disclosure task 7.1 says not to lose.

**Measured, not assumed: the suite does not measurably slow down (design.md's `Split if` into AI-27.3.1/.2 is confirmed NOT triggered).** Three repeated runs of `go test -race -count=1 ./src/ai/openaicompat/...` with the two new sweep files temporarily removed (pre-sweep baseline, non-verbose): package times 2.612s / 2.556s / 2.557s. Three repeated runs with the files restored (post-sweep): 2.577s / 2.567s / 2.556s. The two distributions overlap entirely — no measurable increase. `O(N²)` with `N` ≤ 73 bytes (the largest registered transcript) across 6 transcripts is negligible next to `go test -race`'s own fixed per-process overhead, which dominates wall-clock time for a package this size.

**Full-suite gate:** `go test -race -v -count=1 ./...` (from `backend/agent/`) → exit 0; `ok .../src/agenttest` (2.111s), `ok .../src/ai` (3.419s), `ok .../src/ai/openaicompat` (3.029s). **516** top-level `--- PASS` lines (509 Slice-2b baseline + 7 new top-level `TestOffsetSweep_*` functions), **0** `--- FAIL`, `-race` clean throughout. Full-suite wall clock: `real 4.06s` (verbose `-v`, all packages) vs. pre-sweep baseline `real 3.62s` — the `-v` full-suite number is noisier than the isolated, repeated, non-verbose `openaicompat`-only comparison above, which is the number that actually isolates this slice's cost and shows no measurable change. `make lint` → `go vet ./...` clean, `golangci-lint run --config=.golangci.yml ./...` → `0 issues.` `go.mod`/`go.sum` diff against `feat/ai-27-2b-dispatch-dispositions`: empty — unchanged, zero `require`, no `go.sum`. AI-00 guards `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` and `TestLayer1_ModuleHasNoDependencies_ZeroRequires`: both PASS, no allowlist edit. AI-25 guard `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources`: PASS.

**Inspection evidence (task 3.8, S-ASD-051).** Traced `sweep_transcripts_test.go`: all 6 registered transcripts are 29–73 bytes (`bom-prefixed` 29, `lone-cr` 47, `mixed-terminators` 48, `lf-only` 46, `crlf` 51, `multi-line-data` 73); `grep -nE "Repeat|MB|Mega|1_000_000|1000000"` over both sweep files found no match after the guard bite-proof fixture was removed — the multi-megabyte fixture does not exist anywhere in this milestone's code yet (it is Slice 5's own scope, `bounded_memory_test.go`, not yet created) and is not enumerated by the sweep. Confirmed by trace, not assumed.

**Diff size:** two new files, no modified files (`decoder.go` byte-identical to `feat/ai-27-2b-dispatch-dispositions` — confirmed via `git diff --stat`, empty): `sweep_transcripts_test.go` +119, `offset_sweep_test.go` +324 → **2 files changed, 443 insertions(+), 0 deletions** — within this unit's own pre-forecast band (300–800). Zero production code changed, a first for this milestone's slices — direct evidence that Slices 2a/2b's structural (not flag-based) state carried the re-entrancy property from the start.

**Deviations from design:** None. `sweepTranscripts`/`maxSweepFixtureBytes` live in `sweep_transcripts_test.go` exactly as specified; the size guard executes in `offset_sweep_test.go` as its first act; the bound is the inclusive `<=` relation design.md's Rev 3 fix specified. One precision reconciliation, not a deviation: design.md's Offset-Sweep Harness section paraphrases the per-offset range as `1..len-1`, while `R-ASD-012`'s literal text ("every byte offset from zero through its length") and task 3.4 ("every offset `0..len`") are unambiguous and match each other. This slice implements `0..len` inclusive — a strict superset of design's paraphrase, adding two additional (harmless, always-passing) boundary checks per transcript rather than removing any coverage design called for.

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
>
> **Refreshed by slice 3's apply run**: `offset_sweep_test.go` (same test package, `openaicompat_test`) exports three reusable helpers for 5.3's representative-offset stand-in, built for exactly this reuse (task 3.7): `decodeSplitAtOffset(t, transcript, offset) []Frame` (feeds a transcript in two chunks split at offset), `decodeCanonical(t, transcript) []Frame` (one-Feed reference decode), and `sweepMismatch(name string, offset int, got, want []Frame) (msg string, mismatched bool)` (names both the fixture and the offset on mismatch — the same S-ASD-045-upgraded helper, reusable so 5.3's representative-offset check does not need its own parallel mismatch-message logic). `framesEqual` (field_grammar_test.go) remains the underlying comparison primitive both already share. Call these directly from `bounded_memory_test.go` rather than reimplementing chunk-split-and-compare a second time — they take a `*testing.T`, not `*testing.B` or a raw byte-count, so they work unchanged for a multi-MB `[]byte` at a small, explicit set of representative offsets (5.3's own scope, not the exhaustive `0..len` loop, which stays Slice 3's alone under the size guard).

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
>
> **Refreshed by slice 2b's apply run**: slice 2b added a `bomChecked bool` field and a BOM prefix-wait state (`stripBOM`) that can leave a 1–2-byte partial mark sitting in `d.buf` if the stream ends mid-mark. This needs **no separate check** in 6.5's `Finish()` implementation: a pending partial BOM prefix is just retained bytes in `d.buf`, so the already-planned "any retained line bytes → `ErrTruncated`" rule (the same `len(d.buf) > 0` test an ordinary unterminated line needs) catches it for free. Design.md's "a 1–2-byte BOM prefix pending at Finish is a partial line → truncation" note is therefore already satisfied by the planned shape — do not add BOM-specific code to `Finish()`, and do add a fixture proving it (a lone `"\xEF"` or `"\xEF\xBB"` transcript followed by `Finish()`) so this is pinned by test rather than left as an unverified inference.

## Phase 7 — Outbound coordination notes

- [ ] 7.1 Record the `S-ASD-045` upgrade path in the change's follow-up notes: if a future design factors the sweep's mismatch-message construction into a separately callable pure function, `S-ASD-045` becomes `[test]` and the tally shifts to 73 `[test]` / 12 `[inspection]` — do not lose this when design is revised.
- [ ] 7.2 Flag for correction before archive: `proposal.md` cites AI-31 as the second consumer of the resource-exhaustion concept; doc 0002, the spec, and the design all correctly place it at AI-30.1 item 4 — fix the proposal, do not propagate the AI-31 error.
