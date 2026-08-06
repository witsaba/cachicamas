# Proposal — the streaming frame decoder

> **Change**: `cachicamas-ai-stream-decoder` · **Milestone**: AI-27 · **Nodes**: AI-27.1 … AI-27.6
> **Phase**: proposal · **Date**: 2026-08-03 · **Project**: cachicamas (witsaba)
> **Wave**: 4 "Connect the vendor" — runs **parallel to AI-26** after AI-25
> **Depends on**: AI-19, AI-24.2, AI-25 · **Blocks**: AI-28, AI-32
> **Input**: Engram `sdd/cachicamas-ai-stream-decoder/explore`, `sdd/cachicamas-ai-layer-1/wave-4-preflight`

---

## Intent

AI-28 cannot map a wire transcript into normalized events until something turns a byte stream into
frames, and AI-32 cannot classify a wire failure until something detects one. Both are blocked on a
decoder that does not exist: `backend/agent/src/ai/openaicompat/` is empty and no adapter file
exists anywhere under `src/ai/`.

The framing target is settled: **SSE per WHATWG HTML Living Standard §9.2**, data-only, with
`data: [DONE]` as the OpenAI-compatible dialect's terminal sentinel. Exploration verified against the
live spec that **all seven AI-27.2 grammar items are spec-mandated**, not invented — including the
two that look like implementation detail: the value's leading-space strip is a single `if` (exactly
one space, never a loop), and CRLF is **one** terminator, not a CR-ender followed by an LF-ender.

The decoder must be framing-only. Every semantic decision — what a frame *means*, whether `[DONE]`
ended the stream, which AI-19 category a failure belongs to — is another milestone's charter.

## Scope

### In scope

1. **Skeleton, one frame (27.1)** — a **pure incremental function over bytes**: no HTTP, no
   goroutines, push-fed by the caller so tests control exact chunk boundaries. Frames yielded in
   arrival order with event name and data intact.
2. **Field grammar (27.2)** — the seven spec-mandated items, verbatim-grounded: first-colon split
   with one leading space stripped; colonless line → whole line as name, empty string as value;
   multi-line `data` joined with LF and the trailing LF removed **at dispatch**; CRLF/LF/lone-CR;
   one leading BOM stripped once for the whole stream (the decode step, not per chunk); empty data
   buffer dispatches nothing; buffers reset after dispatch; `id` **ignored entirely** when its value
   contains U+0000 NULL, `retry` ignored unless all-ASCII-digits.
3. **Chunk-boundary re-entrancy (27.3)** — every grammar fixture replayed split at **every** byte
   offset, output identical to the unsplit parse. This is the sweep that catches the phantom
   blank-line bug: a split landing exactly between CR and LF turns one terminator into two, and no
   example-based test ever picks that byte.
4. **Keep-alives and unknowns (27.4)** — comment lines ignored without disturbing accumulation;
   unknown field names ignored; unknown **event** names yielded, not dropped.
5. **Bounded memory (27.5)** — a multi-megabyte frame decodes correctly; a frame past the configured
   hard cap aborts with a typed error instead of growing unbounded.
6. **EOF discipline (27.6)** — clean end at a frame boundary returns no error; a non-empty pending
   buffer at end becomes a typed truncation error and the partial frame is **discarded, never
   dispatched**.

### Out of scope

- **Semantic event mapping (AI-28)** — including **terminal-sentinel recognition**. The decoder
  yields the `[DONE]` frame *verbatim*; doc 0002 assigns recognition to AI-28.2 item 2 and
  post-terminal suppression to AI-28.2 item 3. Baking `[DONE]` into AI-27 would make an
  SSE-generic decoder vendor-specific — a traceability-spine bug by the document's own rule.
- **HTTP concerns of any kind** — status codes, headers, response bodies, `io.EOF` /
  `io.ErrUnexpectedEOF` translation. The producer shell (AI-28.1.1) converts a read-loop end into an
  explicit finish call on the decoder.
- **Wire-failure taxonomy mapping (AI-32)** and the construction of `ai.Failure` values (see below).
- **Any new module dependency.** `backend/agent/go.mod` is at **zero requires**; raw `net/http` plus
  stdlib only. A proposal needing a third-party dependency is a **hard blocker**, not a tradeoff.

### Deliberate non-mechanism

Two items need **no** special machinery, and building one would be over-engineering:

- **AI-27.4 item 2** — the SSE spec has no "known vs unknown event type" concept at the parsing
  layer. Faithful compliance already yields unknown names; a registry or allowlist would be invented
  scope.
- **Malformed-line rejection** — the SSE grammar is **self-healing**: unknown fields ignored,
  colonless lines legal, comments ignored, no rejection case anywhere. AI-27's only genuine decode
  failures are **structural** — 27.5's cap and 27.6's truncation. AI-27.2/.3/.4 classify and route;
  they never return an error.

## Capabilities

### New Capabilities

- `ai-stream-decoder`: incremental SSE framing per WHATWG §9.2 — field grammar, chunk-boundary
  re-entrancy, keep-alives, the frame-size cap, and end-of-stream discipline.

### Modified Capabilities

- **None** under the recommended decision below. `ai-provider-errors` (AI-19) is **cited by
  identifier, never modified**. If the coordinator rules option **(ii)**, this becomes
  `ai-provider-errors`: append a tenth `FailureCategory` member — a shipped-contract amendment that
  belongs in its own change, not absorbed here.

## The open decision — AI-19 has no category for resource exhaustion

`FailureCategory` is a **closed 9-member** vocabulary (`provider_failure.go:53`–`103`):
Authentication, Authorization, RateLimit, Unavailable, Timeout, Cancellation, MalformedResponse,
UnsupportedCapability, Unknown. AI-27.6's truncation maps to `MalformedResponse` exactly — "the
provider's response was not well-formed for its documented transport encoding". **AI-27.5's
cap-exceeded does not**: an over-cap frame may be perfectly well-formed; it is simply too large.
That is a different failure in kind, and the vocabulary has no home for it.

| Option | Consequence |
| --- | --- |
| **(i) Map to `MalformedResponse`** | Imprecise but zero blast radius. The distinction survives as two separate local sentinels, so `errors.Is` still tells cap-exceeded from truncation even though the category collapses. |
| **(ii) Append a tenth category** | `ai-provider-errors` is archived and **promoted** to `openspec/specs/`. R-AIP-005 declares the vocabulary "closed and **append-only**", so appending is the sanctioned path — but it costs an array-pinned name, a sentinel, the exhaustiveness test, AI-23.4's enumeration, and a shipped Layer 1 spec amendment. Widening a shipped contract from inside a framing-only milestone is itself out of AI-27's charter. |
| **(iii) Local vocabulary only, AI-32 chooses** | Cleanest in principle — but **contradicted by literal doc text**. AI-27.5 item 2 says "aborts with a typed **AI-19** error", and both 27.5 and 27.6 declare `Depends on: AI-19.2`. Exploration leaned this way; the doc does not support it as stated. |

**Recommendation: (i), with an escalation trigger recorded rather than a silent compromise.** Map
cap-exceeded to `FailureCategoryMalformedResponse`, keep two distinct local sentinels so no
information is lost at the error level, and escalate the tenth-category question to the milestone
that owns the taxonomy. The escalation is not hypothetical: **AI-30.1 item 4 already plans a second
consumer** of the same concept — "AI-27.5 bounds a single frame; this bounds the sum" (doc 0002,
per-call accumulation cap). Two consumers is a real case for option (ii); one framing milestone is the
wrong place to make it.

### Does AI-27 construct `ai.Failure`? — **No.**

The argument is structural, not stylistic. `MidStreamFailure(report, outputPreceded)` requires
`outputPreceded` — did a normalized output event precede this failure? — and `Delivery` requires
knowing which carrier handed the failure over. **A pure byte decoder can answer neither.** It has no
event stream and no carrier. Both facts are AI-28's.

The reconciliation with the literal charter: **AI-27's dependency is on AI-19.2 — the *category
vocabulary* leaf — not on AI-19's constructors.** So AI-27's typed errors *name* an AI-19
`FailureCategory` and stay wrappable, while AI-28/AI-32 build the `ai.Failure`. That satisfies
"a typed AI-19 error" literally and keeps the decoder framing-only.

## Approach

A **hand-rolled push-fed incremental buffer**, not `bufio`. `bufio.Scanner` is rejected on two
independent grounds: it is pull-based (it owns an `io.Reader`), which conflicts with the charter's
"pure incremental function over bytes — no HTTP, no goroutines" and makes exhaustive split-point
control awkward; and it has no native three-terminator support. Its 64 KB `MaxScanTokenSize` is the
**exact footgun AI-27.5 exists to pin**. `bufio.Reader` shares the ownership problem.

Instead: an internal growable buffer fed by the caller, scanned forward from a cursor by the
decoder's own three-terminator logic, with the cap checked directly against buffer length — so the
*same* code path proves both directions of the trap. The CR/LF fix is one bit of cross-feed state
("pending bare CR, awaiting confirmation the next byte is not LF"), never treating CR as
immediately terminal.

Fixtures follow Wave 2/3 precedent — **inline Go byte literals in table-driven tests**, not a new
`testdata/` subtree (`ai/testdata/` is AI-06-specific compile-pass/fail programs, not a reusable
convention). The multi-megabyte fixture is generated in-test, never committed.

`agenttest/stream_kit_*.go` operates at the `ai.Event`/channel level — one layer above raw bytes.
None of it is reusable here; AI-27.3 needs its own local offset-sweep harness.

**Design owns the shapes.** The public surface (feed/finish, no `io.Reader` ownership) is inferred
from charter language, not literal doc text — `sdd-design` must confirm it, together with the
`ai.Failure` boundary above and the frame value's exact contents.

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/openaicompat/*.go` | New | Decoder, line scanner, frame value, local error sentinels |
| `backend/agent/src/ai/openaicompat/*_test.go` | New | One proof file per AI-27 leaf, plus the offset-sweep harness |
| `backend/agent/src/ai/provider_failure.go` | Read-only | `FailureCategory` cited by identifier; unchanged under option (i) |
| `backend/agent/go.mod` | Unchanged | Zero requires; stdlib only. Any new require is a hard blocker |
| `openspec/specs/ai-provider-errors/spec.md` | Unchanged | Modified **only** if the coordinator rules option (ii) — separate change |
| `docs/architecture/milestones/0002-…md` | Living-graph | Amended by the *apply* phase only if a node splits; not written here |

## Delivery

`delivery_strategy: auto-chain` · `chain_strategy: feature-branch-chain` · budget 5000 ·
tracker `feat/2026-08-03-cachicamas-ai-layer1-wave-4`. Doc 0002: **"the node boundary is the PR-chain
boundary."**

| Slice | Node | Naive | Corrected (×2–4) | Targets |
| --- | --- | --- | --- | --- |
| 1 | AI-27.1 skeleton | 150–200 | 300–800 | tracker |
| 2 | AI-27.2 field grammar | 300–400 | **600–1600** | slice 1 |
| 3 | AI-27.3 re-entrancy | 150–200 | 300–800 | slice 2 |
| 4 | AI-27.4 keep-alives/unknowns | 120–160 | 240–640 | slice 3 |
| 5 | AI-27.5 bounded memory | 120–170 | 240–680 | slice 4 |
| 6 | AI-27.6 EOF discipline | 90–130 | 180–520 | slice 5 |
| | **Total** | **930–1260** | **≈ 2000–4800** | |

The ×2–4 correction is repo-confirmed, not a guess: AI-21 forecast 900–1400 and landed 2241; AI-16
forecast ~290 and landed 1171.

**AI-27.2 must pre-declare an internal split.** It is already at doc 0002's own ~7-item ceiling, and
its corrected estimate alone can reach 1600 — four times the 400-line reassess line. Proposed split
point: **2a** = items 1–3 (colon split, multi-line data, three terminators — the line-and-field
core); **2b** = items 4–7 (BOM, empty-buffer, buffer reset, id/retry disposition — dispatch-time
dispositions). `sdd-tasks` forecasts per node and confirms.

**AI-27.5 and AI-27.6 depend only on AI-27.1** and are parallel-eligible, but stack linearly under
feature-branch-chain to keep every child diff clean.

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| **AI-27.5's multi-MB fixture enters AI-27.3's offset sweep** | **High** | **Load-bearing exclusion, stated as scope, not a footnote.** O(N²) over megabytes is infeasible under any test budget. The sweep admits only the small grammar fixtures (tens–low hundreds of bytes); 27.5 gets its own dedicated non-exhaustive test. With that exclusion, doc 0002's pre-declared `Split if` into AI-27.3.1/.2 is **probably not needed** — no goroutines means `-race` overhead is memory bookkeeping, not scheduling |
| AI-27 alone consumes the wave's whole review budget | **High** | Corrected 4800 sits at the 5000 ceiling. Chaining is **mandatory**, not a preference; a single-PR `size:exception` is not available here |
| An 8th grammar case is discovered mid-leaf | Medium | AI-27.2 is at the 7-item ceiling: the living-graph clause mandates an explicit fractal split (27.2.1/.2), never silent absorption |
| Naive CR/LF handling injects a phantom blank line | Medium | Only manifests when a split lands exactly between CR and LF — which is precisely why the sweep must be **exhaustive**, not sampled. BOM fixtures split at offsets 1 and 2 (inside `EF BB BF`) for the same reason |
| `[DONE]` recognition creeps into the decoder | Medium | Charter-forbidden and fixture-pinned: `[DONE]` appears **nowhere** in the SSE spec. A test asserts it is yielded verbatim as ordinary data |
| The AI-19 category decision is made silently in code | Medium | Blocked on the coordinator's ruling above; `sdd-design` must record it as a decision with its rationale, not an assumption |

## Rollback Plan

Every change is additive and confined to a new package. Nothing in `src/ai` moves; no shipped
signature changes; `go.mod` is untouched, so `src/ai`, `agenttest` and Waves 1–3 compile and pass
identically with or without this change. `git revert` per slice commit is a clean boundary, and
reverting the whole milestone is reverting slice 1's branch point — AI-28 and AI-32 have not started,
so no consumer breaks. If the coordinator rules option (ii), that amendment is a separate change with
its own rollback.

## Dependencies

- AI-24.2 and AI-25 merged into this worktree's branch (`openaicompat/` must exist).
- AI-19 shipped and promoted — depended on for the **category vocabulary** (AI-19.2), not the
  constructors.
- Stdlib only. **No ADR is triggered**; keeping it that way is a constraint, not an outcome.
- Strict TDD; `make test` from `backend/agent/` (`go test -race -v ./...`), `make lint` clean.

## Success Criteria

- [ ] A well-formed single frame in one read yields exactly one frame, name and data intact.
- [ ] All seven §9.2 grammar items pass, each traceable to its spec sentence — including one leading
      space stripped (never two) and `id` ignored whole when its value contains NUL.
- [ ] Every grammar fixture replayed split at **every** byte offset yields output identical to the
      unsplit parse, including a split between CR and LF and inside a 3-byte BOM.
- [ ] Comments do not disturb accumulation; an unknown event name is yielded, not dropped.
- [ ] A multi-megabyte frame decodes correctly; an over-cap frame aborts with a typed error — and the
      multi-MB fixture is provably absent from the offset sweep.
- [ ] Clean end returns no error; end mid-frame returns a typed truncation error and the partial
      frame is **not** dispatched.
- [ ] The decoder yields `data: [DONE]` verbatim and recognizes nothing about it.
- [ ] The AI-19 cap-exceeded category decision is recorded in `design.md` with its rationale.
- [ ] `make test` green, `make lint` clean, AI-00 import guards pass, `go.mod` unchanged.

> **Stale citation corrected (Phase 7 coordination, task 7.2, applied at Slice 6).** This proposal's
> "escalation trigger" paragraph above previously named **AI-31** as the second consumer of the
> resource-exhaustion concept. `design.md` and `specs/ai-stream-decoder/spec.md` always correctly
> placed that second consumer at **AI-30.1 item 4** (doc 0002 line 1819 agrees), so this proposal was
> the only artifact still carrying the stale figure. It has been corrected in place above rather than
> left for archive time, since Slice 6 is this change's own final apply run — no later apply phase
> would otherwise have touched this file again before archive.
