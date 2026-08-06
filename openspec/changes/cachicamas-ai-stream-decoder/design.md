# Design: AI-27 — The streaming frame decoder

## Technical Approach

A hand-rolled, push-fed incremental SSE decoder in `backend/agent/src/ai/openaicompat/` (the package AI-25 creates; match its plain-noun file naming). Confirmed against local Go source: `bufio.Scanner` caps tokens at `MaxScanTokenSize = 64 * 1024` and fails with `ErrTooLong` (`/usr/local/go/src/bufio/scan.go:71,82`) — the exact footgun AI-27.5 pins — and both `Scanner` and `bufio.Reader` own an `io.Reader`, conflicting with the charter's "pure incremental function over bytes, no HTTP, no goroutines". **Proposal confirmed: hand-rolled buffer, stdlib only, `go.mod` untouched.** The decoder is framing-only; every semantic fact (what `[DONE]` means, `outputPreceded`, the delivery carrier) belongs to AI-28/AI-32 per the coordinator rulings.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Public shape | `NewDecoder(maxFrameBytes int) *Decoder` (≤0 → `DefaultMaxFrameBytes` = 8 MiB); `Feed(p []byte) ([]Frame, error)`; `Finish() error`. No `io.Reader`, no callbacks, no channels | `io.Writer` impl (hides frames-out); callback per frame (complicates error flow, invites filtering); channel (goroutine-adjacent) | Caller pushes bytes, so tests control exact chunk boundaries — AI-27.3's precondition and the "independently testable forever" guarantee. Slice-return API has **no frame-dropping path**: every dispatched frame is returned unconditionally |
| Frame value | `Frame{Event string; Data []byte}` — `Event` is `"message"` when the event-type buffer is empty (§9.2's own dispatch default); `Data` is a **copy**, never aliasing the internal buffer. Carries NO last-event-id, retry, comments, sequence, or any parsed/JSON view | lossless empty-`Event`; carrying last-event-id | Spec-literal, grounded verbatim in §9.2's dispatch step: create "an event that uses the `MessageEvent` interface, with the event type `message`", then "If the event type buffer has a value other than the empty string, change the type of the newly created event to equal the value of the event type buffer" — an empty buffer (never set, **or set empty by an empty-valued `event:` line**) leaves the default. Upheld by cross-validation on three further grounds: (1) the lost `event: message`-vs-typeless distinction is **unreachable** — the dialect is data-only, `event: message` never appears on the wire, and neither AI-28.2 (terminal recognition keys off **data**) nor AI-30.1 reads `Frame.Event`; (2) `""` is Go's zero value — a `Frame{}` escaping a slicing bug would be indistinguishable from a legitimately typeless frame, while `"message"` is obviously wrong on sight, a robustness argument that cuts against losslessness; (3) the mapping is recoverable in the direction that matters. **Living-graph trigger**: doc 0002 reserves re-derivation of AI-27's nodes on a transport change — a typed-event dialect would make the lost distinction reachable and reopens this decision. Data-copy semantics keep frames valid after later `Feed`s compact the buffer. id/retry are pinned-ignored (below), so carrying them is invented scope |
| Buffer strategy | One growable `[]byte`; each `Feed` appends, scans complete lines forward, retains the unconsumed tail (`d.buf = append(d.buf[:0], d.buf[cursor:]...)`). Separate `data`/`eventType` accumulation buffers per §9.2 | ring buffer; per-line allocation | Simplest structure that gives exact split-point behavior; compaction bounds retained memory to the current frame |
| Three terminators | CR, LF, CRLF-as-one. **A buffer-final CR is never consumed by `Feed`** — it stays in the retained tail until the next byte (LF → one CRLF terminator; other → lone-CR terminator) or `Finish` (resolves it as lone CR) | boolean `pendingCR` flag | Same one-bit of cross-feed state the proposal named, carried **structurally** in the retained tail instead of a flag that can desynchronize — kills the phantom-blank-line bug by shape |
| BOM | `bomChecked bool`; before scanning, if unset: buffer a proper prefix of `EF BB BF` → consume nothing and wait; full BOM → drop 3 bytes; anything else → mark checked. Once per stream | per-chunk stripping | §9.2 strips one leading BOM via UTF-8 decode, once for the whole stream; the wait rule makes a BOM split at offsets 1/2 correct. A 1–2-byte BOM prefix pending at `Finish` is a partial line → truncation |
| Cap accounting | `len(retained tail) + len(data) + len(eventType)` checked **whenever accumulation grows inside the scan loop**. The relation is closed: abort **only when the accumulated size is strictly greater than the cap** (`> cap`, never `>=`) — an exactly-at-cap frame decodes | `>=`; separate large-frame fast path | Doc 0002: "a frame **exceeding** the configured hard cap aborts" — exactly-at-cap is not exceeding. One code path proves both trap directions: 27.5's green (multi-MB and exactly-at-cap frames decode) and red (over-cap aborts) exercise the same counter. Checking inside the loop means one giant `Feed` trips without materializing past cap+chunk |
| Errors (coordinator rulings 1–2) | Two exported sentinels: `ErrFrameTooLarge` (cap) and `ErrTruncated` (mid-frame EOF), **each built with `fmt.Errorf("…: %w", ai.ErrMalformedResponse)`** so `errors.Is(err, ai.ErrMalformedResponse)` is true for either — the repo-native AI-19 matching idiom — while the two decoder sentinels stay distinct from each other under `errors.Is`. Plus `Category(err error) (ai.FailureCategory, bool)` mapping **both** to `FailureCategoryMalformedResponse` as the explicit naming surface. AI-27 **never constructs `ai.Failure`** | bare `errors.New` sentinels reachable only via `Category()` (a second parallel mechanism for a fact AI-19 already has one mechanism for); constructing `MidStreamFailure` here; a tenth category now | Ruling 1: a byte decoder has no `outputPreceded` fact and no carrier — it depends on AI-19.2 (category vocabulary), not the constructors. Ruling 2: cap→`MalformedResponse` is a **recorded deliberate compromise** (an over-cap frame may be well-formed); two distinct sentinels keep `errors.Is` lossless. `Category` makes "names an AI-19 category" structural and test-pinned, not a doc comment |
| Escalation trigger (ruling 3) | Recorded, not acted on: a tenth AI-19 category (resource exhaustion) becomes warranted when doc 0002's per-call accumulation cap (AI-30.1 item 4, "AI-27.5 bounds a single frame; this bounds the sum") lands as a second consumer. The append belongs to the taxonomy owner as its own change | appending from this milestone | R-AIP-005 sanctions append-only, but widening a shipped promoted spec is outside a framing-only charter |
| Poisoning | First terminal error (cap) poisons the decoder: later `Feed`/`Finish` return the same error, zero frames. Frames completed before the trip in the same `Feed` are returned **with** the error | silent reset | A half-consumed frame boundary is unrecoverable; returning pre-trip frames loses nothing |
| `Finish` semantics | Resolve a pending buffer-final CR as a lone-CR terminator first, then: any retained line bytes OR non-empty `data`/`eventType` accumulation → `ErrTruncated`, partial frame **discarded**; otherwise nil | dispatching the pending partial | §9.2: an incomplete event before the final empty line is not dispatched; silent truncation is 27.6's named failure mode |
| id/retry disposition | Decoder tracks neither. Pinned by test as "no observable effect", with **both** NUL-bearing and NUL-free `id` fixtures and digit/non-digit `retry` fixtures present, so any future last-event-id tracking must consciously flip pinned cases | tracking last-event-id | Ignoring is spec-fine; the fixtures pin the disposition by test rather than by accident (doc 0002 27.2 item 7) |
| Fixtures | Inline Go byte/string literals in table-driven tests (Wave 2/3 precedent, `agenttest/conformance_text.go`); multi-MB fixture generated in-test (`bytes.Repeat`), never committed. **No golden-file convention established here** | first repo golden convention under `openaicompat/testdata/` | Verified: zero golden machinery exists repo-wide; SSE fixtures are short and structured-asserted (frames), not rendered output — the golden pattern's home. Establishing a repo-first convention inside a framing milestone is invented scope. **Flag to coordinator**: AI-26 is deciding fixtures in parallel; if AI-38's recorded transcripts later need real files, settle one repo convention once, at that milestone — not twice now |
| AI-27/AI-28 seam | What the shape genuinely forbids is **signalling** sentinel termination: `Finish` has no "ended by sentinel" return and `Feed` no third result, so a recognizing decoder could not report it. The two likelier leaks — *dropping* the sentinel frame (one comparison) and *suppressing post-sentinel frames* (a bool plus a guard — literally AI-28.2 item 3, misplaced a layer down) — fit the existing `Feed` signature unchanged and are locked **by test, not by shape**: the `[DONE]`-yielded-verbatim pin, the post-sentinel-frames-still-yielded pin, and the spec's production-source literal-search scenario. Production sources contain no `[DONE]` literal and import no `encoding/json`; a test pins `data: [DONE]` yielded verbatim as `Frame{Event: "message", Data: []byte("[DONE]")}` | discipline-only comment; claiming shape alone forbids all leaks (overstated — dropping/suppression fit the signature) | AI-28.2 item 2 owns recognition; the honest split is shape for termination-signalling, tests for drop/suppression |

## Data Flow

    AI-28 producer shell (outside AI-27)          tests (offset sweep)
      resp.Body.Read loop ── p []byte ──► Decoder.Feed(p) ──► []Frame{Event, Data}
      io.EOF / io.ErrUnexpectedEOF               │ cap → ErrFrameTooLarge (poisons)
        └────── translate outside ──► Decoder.Finish() ──► nil | ErrTruncated
      AI-28 then: recognizes [DONE], supplies outputPreceded + carrier, wraps
      sentinels via Category() into ai.MidStreamFailure — never AI-27's job.

## File Changes

All paths under `backend/agent/src/ai/openaicompat/` in the worktree. Tests are external (`package openaicompat_test`) — the public API must suffice, which is itself the testability claim.

| File | Action | Slice | Owns |
|---|---|---|---|
| `frame.go` | Create | 1 | `Frame` value + copy semantics |
| `decoder.go` | Create | 1 (grown 2a/2b/5/6) | `Decoder`, `NewDecoder`, `Feed`, `Finish`, scan loop |
| `errors.go` | Create | 5 (grown 6) | `ErrFrameTooLarge`, `ErrTruncated` (each wrapping `ai.ErrMalformedResponse`), `Category`. Plain-noun name per AI-25's convention — snake_case compounds are reserved for test files |
| `doc.go` | Modify | 1 | one paragraph: framing layer, semantic boundary, and that `maxFrameBytes` is accepted here but **enforced from AI-27.5** |
| `decoder_test.go` | Create | 1 | 27.1 skeleton proofs |
| `field_grammar_test.go` | Create | 2a (grown 2b) | 27.2 items 1–7 tables |
| `sweep_transcripts_test.go` | Create | 3 (grown 4) | `sweepTranscripts` registry + `maxSweepFixtureBytes` constant |
| `offset_sweep_test.go` | Create | 3 | 27.3 exhaustive harness; **executes the size guard** as its first act |
| `keepalive_test.go` | Create | 4 | 27.4 comments/unknowns + `[DONE]`-verbatim pin |
| `bounded_memory_test.go` | Create | 5 | 27.5 both trap directions |
| `eof_test.go` | Create | 6 | 27.6 `Finish` discipline |

AI-25's ambient-authority guard scans this package's non-test sources; the new files import only `ai` (and stdlib `errors`/`bytes`-level packages) and pass trivially.

> **Apply-phase amendment (slice 1, recorded here for continuity — not a re-run of design review).**
> Slice 1's apply run found a real collision risk this table did not anticipate: `doc.go` already
> carries AI-25's full package doc comment, and AI-26 is concurrently developing translation in the
> same package in a sibling worktree — a second branch that may independently want to touch the same
> file before either chain converges. Slice 1 therefore did **not** modify `doc.go`. The framing-layer,
> semantic-boundary, and cap-accepted-but-inert content this row describes instead lives as GoDoc on
> the `Decoder` type in `decoder.go` itself. This is a placement change only — every fact this row
> requires is still documented and reviewer-visible, just not in `doc.go`. Later slices should keep
> following this placement rather than reopening `doc.go`.

## Interfaces / Contracts

```go
func NewDecoder(maxFrameBytes int) *Decoder            // <=0 selects DefaultMaxFrameBytes (8 MiB)
func (d *Decoder) Feed(p []byte) ([]Frame, error)      // frames in arrival order; may return frames AND a terminal error
func (d *Decoder) Finish() error                       // nil = clean end; ErrTruncated = pending partial, discarded
type Frame struct{ Event string; Data []byte }         // Event "message" when no event field; Data a copy
func Category(err error) (ai.FailureCategory, bool)    // both sentinels → FailureCategoryMalformedResponse
```

Both sentinels wrap `ai.ErrMalformedResponse` via `%w`, so `errors.Is(err, ai.ErrMalformedResponse)` holds for either while `errors.Is(ErrFrameTooLarge, ErrTruncated)` stays false — consumers keep the repo-native idiom and `Category` remains the explicit naming surface.

`maxFrameBytes` is accepted from slice 1 but **enforced from slice 5** (its RED tests) — an inert parameter inside the unmerged chain. Two mitigations bind: (a) slice 1's `doc.go` paragraph states the cap is accepted here and enforced from AI-27.5, so a slice-1 reviewer is not misled into treating it as live; (b) slice 5's RED list includes the "zero selects `DefaultMaxFrameBytes`" assertion, so the default path is proven, not assumed.

## Offset-Sweep Harness (AI-27.3) — structural exclusion

- `sweepTranscripts` is the **only** input to the sweep. The guard **executes in `offset_sweep_test.go`** as the harness's first act: it asserts every registered fixture is **at most** `maxSweepFixtureBytes` — inclusive, per the spec's bound (`S-ASD-049`): `len(tr) <= maxSweepFixtureBytes`, constant `= 1024`, declared beside the registry in `sweep_transcripts_test.go` and fails the suite loudly otherwise — the multi-MB fixture **cannot** enter even by accident. Second lock: 27.5's fixture is a local variable inside `bounded_memory_test.go`, never registered.
- Per transcript: canonical single-`Feed` parse; then for every offset `1..len-1`, two `Feed`s split at that offset, frames compared deep-equal; plus one byte-at-a-time replay (O(N), catches multi-split). O(N²) with N ≤ 1024, no goroutines — `-race` overhead is memory instrumentation only. **Doc 0002's `Split if` into AI-27.3.1/.2 is not triggered**; revisit only if the suite measurably slows.
- Registry seeds: LF-only, CRLF, lone-CR, mixed-terminator, BOM-prefixed (offsets 1/2 fall inside `EF BB BF` automatically), multi-line data, comment-interleaved (added by slice 4).

## Testing Strategy (RED-first; runner `make test` from `backend/agent/`)

| Node | Failing assertion before implementation |
|---|---|
| 27.1 | `NewDecoder(0)` + one `Feed("event: x\ndata: y\n\n")` yields exactly `[]Frame{{Event:"x", Data:"y"}}`; two frames in one `Feed` arrive in order; a frame's `Data` unchanged after a later `Feed` (copy pin) |
| 27.2a | Tables: first-colon split; exactly ONE leading space stripped (`"data:  a"` → `" a"`); colonless line = name with empty value; `data` lines joined with LF, trailing LF absent at dispatch; each of CRLF/LF/CR terminates — CRLF yields no phantom empty line |
| 27.2b | BOM stripped once (second BOM is content); empty data buffer → zero frames (`"event: x\n\n"` dispatches nothing); after dispatch the next typeless frame is `Event=="message"`, never the prior name; **an empty-valued `event:` line sets the buffer to the empty string, which dispatch treats as never-set — the frame carries `Event=="message"`** (the case where the two candidate defaults visibly diverge); id-with-NUL, id-without-NUL, digit/non-digit retry → no observable effect |
| 27.3 | Sweep harness red until split-state handling exists: every registered transcript, every offset, deep-equal frames vs canonical; oversized-registry guard trips on a deliberately overweight fixture (then removed) |
| 27.4 | `": ping"` between two `data` lines does not disturb accumulation; unknown field ignored; unknown event name yielded; `data: [DONE]` yielded verbatim, decoder state unchanged; **frames arriving after a `[DONE]` frame are still yielded** — post-sentinel suppression is AI-28.2 item 3's, and this pin is one of the test locks the seam decision relies on |
| 27.5 | `NewDecoder(1024)` + 2 KiB frame → frames nil-so-far, `errors.Is(err, ErrFrameTooLarge)`, NOT `ErrTruncated`, and `errors.Is(err, ai.ErrMalformedResponse)`; **exactly-at-cap frame (accumulation == cap) decodes** — the boundary of the strictly-greater relation; **mixed pre-trip case (S-ASD-085): one `Feed` buffer carrying a complete frame followed by an over-cap frame returns the complete frame together with the error**; poisoned thereafter; `NewDecoder(0)` **selects `DefaultMaxFrameBytes`** and a multi-MB frame under it decodes byte-exact; the multi-MB fixture also replayed **split at a small fixed set of representative offsets** and **fed in many small chunks** (its non-exhaustive stand-ins for the sweep it is excluded from); `Category(ErrFrameTooLarge)` = `MalformedResponse, true` |
| 27.6 | `Feed("data: a\n\n")` + `Finish()` → nil; `Feed("data: a")` + `Finish()` → `errors.Is(err, ErrTruncated)`, NOT `ErrFrameTooLarge`, and `errors.Is(err, ai.ErrMalformedResponse)`, no frame ever emitted; trailing `"data: a\r"` + `Finish` resolves CR then reports truncation; `Category(ErrTruncated)` = `MalformedResponse, true` |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. Pure byte parsing; no I/O of any kind.

## Migration / Rollout — chain slices

No migration; additive, zero consumers until AI-28. Feature-branch chain, tracker `feat/2026-08-03-cachicamas-ai-layer1-wave-4`; node boundary = PR-chain boundary (doc 0002). **Proposal's 2a/2b split confirmed**, giving 7 slices:

| Slice | Scope (precise boundary) | Corrected est. | Targets |
|---|---|---|---|
| 1 | 27.1: `frame.go`, `decoder.go` (LF-only happy path, minimal colon split), `doc.go` ¶, `decoder_test.go` | 300–800 | tracker (after AI-25's chain) |
| 2a | 27.2 items 1–3: full first-colon/space rules, multi-line data join + dispatch trim, three terminators incl. deferred-final-CR | 300–800 | slice 1 |
| 2b | 27.2 items 4–7: BOM state, empty-data no-dispatch, buffer reset/default type, id/retry pins | 300–800 | slice 2a |
| 3 | 27.3: registry + size guard + exhaustive harness | 300–800 | slice 2b |
| 4 | 27.4: comments/unknown fields/unknown events + `[DONE]` pin; registers its fixtures into the sweep | 240–640 | slice 3 |
| 5 | 27.5: cap enforcement + poisoning + `errors.go` (`ErrFrameTooLarge`, `Category`) | 240–680 | slice 4 |
| 6 | 27.6: `Finish` + `ErrTruncated` + CR-at-EOF resolution | 180–520 | slice 5 |

Total ≈ 2,000–4,800 at the 5,000 ceiling — chaining mandatory, no `size:exception`. 27.5/27.6 are graph-parallel off 27.1 but stack linearly for clean child diffs. `Decision needed before apply: No` · `Chained PRs recommended: Yes` · `400-line budget risk: High`.

**R-ASD-011 — mirror of the 27.3 ruling**: AI-27.2 sits at doc 0002's 7-item ceiling; a discovered eighth grammar case mandates an explicit living-graph **node** split (AI-27.2.1/.2, amended by the apply phase), never silent absorption. The 2a/2b split above is a PR-review boundary, **not** a node split, and does not satisfy that clause.

Rollback: revert per slice; whole-milestone revert is slice 1's branch point — AI-28/AI-32 have not started.

## Open Questions

- [ ] **Apply-time gate**: `openaicompat/` does not exist in this worktree yet (verified) — slice 1 must branch after AI-25's chain lands on the tracker; if AI-25 slips, AI-27 is blocked, not re-parented. **Resolved for slice 1's own apply run**: AI-25 landed (`feat/ai-25c-test-server-viability`), `openaicompat/` exists with client.go/credential.go/endpoint.go/request.go/doc.go and their tests.
- [ ] Coordinator: settle a single fixture convention at AI-38 (recorded-transcript milestone) rather than letting AI-26 and AI-27 diverge — both currently choose inline literals independently.
