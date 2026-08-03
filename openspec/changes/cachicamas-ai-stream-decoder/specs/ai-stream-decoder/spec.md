# Spec — the streaming frame decoder

> **Change**: `cachicamas-ai-stream-decoder`
> **Milestone**: AI-27 · **Nodes**: AI-27.1 … AI-27.6, all `[leaf]`
> **Phase**: spec (delta — new capability, ADDED only)
> **Canonical spec**: `openspec/specs/ai-stream-decoder/spec.md` — created by `sdd-archive` from this delta
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable.
> **Date**: 2026-08-03 · **Project**: cachicamas (witsaba)
> **Requirement IDs**: `R-ASD-0NN` · **Scenario IDs**: `S-ASD-0NN`
> **Binding predecessors, cited by identifier and never modified**: [`ai-provider-errors`](../../../../specs/ai-provider-errors/spec.md) (AI-19) — the **closed, append-only nine-member failure-category vocabulary** only (AI-19.2), never its constructors · [doc 0002](../../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) §§ AI-27.1 … AI-27.6 · [`proposal.md`](../../proposal.md)
> **Normative framing source**: WHATWG HTML Living Standard **§ 9.2 Server-sent events**, the "interpret the event stream" and "dispatch the event" algorithms.
> **Depends on**: AI-19, AI-24.2, AI-25 · **Blocks**: AI-28, AI-32 · **Parallel with**: AI-26

---

## ADDED Requirements

## Purpose

AI-28 cannot map a wire transcript into normalized events and AI-32 cannot classify a wire failure until something turns a byte stream into frames. This spec constrains what that decoder MUST guarantee: that framing is faithful to § 9.2, that **arbitrary read boundaries do not change the decoded frames**, and that the two structural failures it can actually have — an over-cap frame and an end mid-frame — are typed and never silent.

Three distinctions shape every requirement below and are stated once here.

**The grammar is self-healing; the failures are structural.** § 9.2 has no malformed-line rejection case anywhere: unknown fields are ignored, colonless lines are legal, comments are ignored. AI-27's only genuine decode failures are therefore the size cap (AI-27.5) and the end-mid-frame condition (AI-27.6). Requirements covering AI-27.2, AI-27.3 and AI-27.4 classify and route input; they never produce an error.

**Framing is not meaning.** Whether a frame is a terminal sentinel, what its payload says, which taxonomy member a wire failure belongs to, and how a transport read loop ends are all other milestones' charters. `R-ASD-025` makes a violation of that boundary detectable rather than merely discouraged.

**Naming a category is not constructing a failure.** AI-27's declared dependency is on AI-19.2 — the **category vocabulary** — not on AI-19's constructors. A pure byte decoder cannot answer whether normalized output preceded a failure, nor which carrier handed it over, so it MUST NOT construct provider-failure values. Its own errors *name* a category and stay wrappable; AI-28/AI-32 build the provider failure.

Requirement count: **25** (`R-ASD-001` … `R-ASD-025`). Scenario count: **85** (`S-ASD-001` … `S-ASD-085`) — **73 [test]**, **12 [inspection]**.

*Counted mechanically over the scenario bullets, not estimated. The twelve `[inspection]` scenarios are `S-ASD-010`, `011`, `039`, `040`, `051`, `056`, `064`, `068`, `069`, `078`, `082`, `083`. Evidence rules key off these markings, so a reconciliation that finds a different split is reading a stale revision, not a downgraded test.*

> **Revision note (Phase 7 coordination, applied at Slice 6).** `S-ASD-045` shipped as `[inspection]` with a recorded upgrade condition: it becomes `[test]` once the sweep harness's mismatch-message construction is factored into a separately callable pure function. Slice 3's own implementation of `offset_sweep_test.go` did exactly that (`sweepMismatch`), so the condition was met before this file was updated to say so — `TestOffsetSweep_MismatchMessageNamesOffendingByteOffset` has been exercising the runnable scenario since Slice 3 landed; only this document's own marking lagged until now, exactly as tasks.md's Phase 7 coordination note anticipated.

## Requirement ownership by node

| Node | Requirements |
| --- | --- |
| AI-27.1 — skeleton: one frame | `R-ASD-001` … `R-ASD-003` |
| AI-27.2 — field grammar (7-item ceiling) | `R-ASD-004` … `R-ASD-011` |
| AI-27.3 — chunk-boundary re-entrancy | `R-ASD-012` … `R-ASD-014` |
| AI-27.4 — keep-alives and unknowns | `R-ASD-015` … `R-ASD-017` |
| AI-27.5 — bounded memory | `R-ASD-018` … `R-ASD-021` |
| AI-27.6 — EOF discipline | `R-ASD-022` … `R-ASD-024` |
| Charter boundary | `R-ASD-025` |

## Definitions used by this spec

- **The decoder** — the framing component this milestone ships. It is fed bytes by its caller and yields frames; it never reads from a transport itself.
- **A frame** — one dispatched event: an event-type name and the accumulated data payload, with the dispatch-time trailing separator already removed.
- **The default event type** — the type a frame carries when no type line contributed to it.
- **An input chunk** — one contiguous run of bytes handed to the decoder in a single feeding step. Chunk boundaries are chosen by the caller and carry no meaning.
- **The end-of-input signal** — the caller's explicit statement that no further bytes will arrive. It is a decoder-level signal, not a transport condition.
- **A golden transcript** — one small fixture byte sequence, **at most the sweep fixture bound**, whose canonical unsplit decoding is the reference result.
- **The sweep fixture bound** — 1024 bytes, inclusive. A transcript of exactly 1024 bytes is admissible; 1025 is not.
- **The offset sweep** — replaying a golden transcript once per byte offset `0 … len(transcript)`, split into two input chunks at that offset.
- **The hard cap** — the configured maximum size a single in-progress frame may reach before decoding aborts.
- **Cap-exceeded** and **truncation** — the two, and only two, structural decode failures.
- **Scenario kind** — every scenario below is marked either **[test]** (runnable and failable under `make test` / `go test -race -v ./...`) or **[inspection]** (a reviewer-checkable obligation over the artifact or the shipped source, deterministic but not executed by the suite). Under Strict TDD, every **[test]** scenario MUST be demonstrated failing before the code that satisfies it exists. A scenario that cannot be shown failing first is not a **[test]** scenario and MUST NOT be marked as one.

> **Scenario IDs are allocated, not positional.** `S-ASD-084` and `S-ASD-085` were added by a corrective pass and sit under the requirements they belong to rather than at the end. The ID set `S-ASD-001 … S-ASD-085` is complete with no gaps and no duplicates.

---

## AI-27.1 — Skeleton: one frame `[leaf]`

### R-ASD-001 — A well-formed single frame in one input yields exactly one frame

Given one well-formed frame delivered in a single input chunk, the decoder MUST yield exactly one frame, with its event-type name and its data payload intact and unaltered. It MUST NOT yield a second, empty or duplicate frame, and MUST NOT alter, re-encode, trim or normalize the payload bytes beyond the dispatch-time separator removal § 9.2 mandates.

#### Scenarios

- **S-ASD-001** *[test]* — Given a single frame carrying an explicit event-type line and one data line, when it is fed in one chunk followed by the terminating blank line, then exactly one frame is yielded, its type equals the type line's value and its data equals the data line's value byte for byte.
- **S-ASD-002** *[test]* — Given the same input, when the yielded frames are counted, then the count is exactly one — no trailing empty frame is produced by the terminating blank line.
- **S-ASD-003** *[test]* — Given a frame whose data contains non-ASCII multi-byte content, when it is decoded, then the yielded payload is byte-identical to the input's data bytes.

### R-ASD-002 — Frames are yielded in arrival order

When several frames are present in the input, the decoder MUST yield them in the order they appear in the byte stream. It MUST NOT reorder, coalesce, drop or duplicate frames, regardless of how many frames arrive in one input chunk.

#### Scenarios

- **S-ASD-004** *[test]* — Given three distinct frames in one input chunk, when they are decoded, then exactly three frames are yielded in the same order as their appearance in the bytes.
- **S-ASD-005** *[test]* — Given two frames whose payloads are identical, when they are decoded, then two frames are yielded and neither is coalesced into the other.
- **S-ASD-006** *[test]* — Given frames delivered one per input chunk, when all chunks are fed in order, then the yielded order matches the feeding order.

### R-ASD-003 — The decoder is a pure incremental function over bytes

The decoder MUST be drivable to completion from bytes alone: **no network access, no transport reads, and no concurrency**. It MUST NOT start, own or require goroutines, MUST NOT depend on wall-clock time or timers for correctness, and MUST NOT require any input source other than the byte chunks its caller hands it. Decoding the same byte sequence MUST produce the same frames every time.

#### Scenarios

- **S-ASD-007** *[test]* — Given a complete transcript, when it is decoded in a test that performs no network setup whatsoever, then decoding completes and yields the expected frames.
- **S-ASD-008** *[test]* — Given a complete transcript, when the goroutine count is sampled immediately before feeding and again after the end-of-input signal, then the count is unchanged — the decoder started none.
- **S-ASD-009** *[test]* — Given one transcript decoded twice by two independent decoder instances, when the two frame sequences are compared, then they are identical.
- **S-ASD-010** *[inspection]* — Given the shipped test sources for this package, when a reviewer reads them, then no test relies on a sleep, timer or wall-clock wait to observe a frame — every frame is observed synchronously from a feeding step or the end-of-input signal.
- **S-ASD-011** *[inspection]* — Given the shipped decoder source, when a reviewer reads its imports, then no transport, HTTP or concurrency-primitive import is present.

> **Moved out of the scenario set, deliberately.** "The whole package suite passes under `-race`" was previously carried here as a scenario. It is not one: no test inside a suite can assert that its own suite passes. It is **delivery evidence**, and the change already carries it as such — `make test` green and `make lint` clean in the proposal's success criteria. `S-ASD-010` retains only the half that a reviewer can actually check.

---

## AI-27.2 — Field grammar `[leaf]`

> **Ceiling notice.** This node carries doc 0002's seven grammar items and is **already at that document's ~7-item split ceiling**. A newly discovered eighth grammar case MUST NOT be absorbed silently here; `R-ASD-011` makes that explicit.

### R-ASD-004 — Field lines split at the first colon, with exactly one leading space stripped

A field line MUST be split at the **first** colon: the characters before it are the field name, the characters after it are the value. If the value begins with a single space, **exactly one** space MUST be removed — one `if`, never a loop. A line containing no colon MUST be treated as a field whose name is the whole line and whose value is the empty string. (§ 9.2: "Collect the characters on the line before the first U+003A COLON character…"; "If value starts with a U+0020 SPACE character, remove it from value"; "the whole line as the field name, and the empty string as the field value.")

#### Scenarios

- **S-ASD-012** *[test]* — Given a data line whose value itself contains a colon, when it is decoded, then the split occurs at the first colon only and the remaining colons survive inside the value.
- **S-ASD-013** *[test]* — Given a data line whose value is preceded by **two** spaces, when it is decoded, then exactly one space is removed and the yielded payload retains the second space.
- **S-ASD-014** *[test]* — Given a data line with no space after the colon, when it is decoded, then the value is taken verbatim with nothing removed.
- **S-ASD-015** *[test]* — Given a line consisting of the bare word `data` with no colon, when it is decoded together with a terminating blank line, then it is treated as the data field with an empty value.
- **S-ASD-016** *[test]* — Given a colonless line whose name is not a recognized field, when it is decoded, then it is ignored per `R-ASD-016` and disturbs no accumulation.

### R-ASD-005 — Multi-line data joins with a line feed and loses the trailing one at dispatch

Each data field value MUST be appended to the frame's data accumulation followed by a single line feed. At dispatch, if the accumulation's last character is a line feed, that single character MUST be removed. (§ 9.2: "Append the field value to the data buffer, then append a single U+000A LINE FEED character"; "If the data buffer's last character is a U+000A LINE FEED character, then remove the last character.")

#### Scenarios

- **S-ASD-017** *[test]* — Given a frame with three data lines, when it is decoded, then the yielded payload is the three values joined by exactly one line feed each, with no trailing line feed.
- **S-ASD-018** *[test]* — Given a frame with a single data line, when it is decoded, then the payload has no trailing line feed.
- **S-ASD-019** *[test]* — Given a frame whose last data line has an empty value, when it is decoded, then the payload ends with exactly one line feed — the separator that preceded the empty value survives, and only the final one is removed.
- **S-ASD-020** *[test]* — Given a frame whose data value itself ends in a line feed character within its own bytes, when it is decoded, then only the dispatch-time separator is removed and the content line feed survives.

### R-ASD-006 — Three line terminators, and CRLF is one terminator

Lines MUST be terminated by any of: a carriage return followed by a line feed, a lone line feed, or a lone carriage return. A CR-LF pair MUST be treated as **one** terminator — never as a CR-terminated line followed by an LF-terminated line — **including when the pair is divided across two input chunks**. (§ 9.2: "Lines must be separated by either… CRLF… a single LF… or a single CR.")

#### Scenarios

- **S-ASD-021** *[test]* — Given the same logical transcript rendered three times, once with CRLF, once with LF and once with lone-CR terminators, when each is decoded, then all three yield identical frames.
- **S-ASD-022** *[test]* — Given a transcript terminated exclusively by CRLF, when it is decoded in one chunk, then no empty frame and no empty data line is produced anywhere in the output.
- **S-ASD-023** *[test]* — Given a transcript whose terminators mix CRLF, LF and lone CR within one frame, when it is decoded, then the data lines accumulate as if terminators were uniform.
- **S-ASD-024** *[test]* — Given a CRLF pair divided so the carriage return is the last byte of one input chunk and the line feed is the first byte of the next, when both chunks are fed in order, then the result is identical to the unsplit decoding and no blank line is injected.

### R-ASD-007 — One leading byte-order mark is stripped once for the whole stream

A byte-order mark at the very start of the stream MUST be stripped **once**, as part of decoding the stream — not per input chunk and not per line. A byte-order mark occurring anywhere else MUST be treated as ordinary content. (§ 9.2 by cross-reference: "The UTF-8 decode algorithm strips one leading UTF-8 BOM, if any.")

#### Scenarios

- **S-ASD-025** *[test]* — Given a transcript prefixed by exactly one byte-order mark, when it is decoded, then the resulting frames are identical to the same transcript with no mark.
- **S-ASD-026** *[test]* — Given a transcript prefixed by **two** consecutive byte-order marks, when it is decoded, then exactly one is stripped and the second survives as content of the first field name.
- **S-ASD-027** *[test]* — Given a byte-order mark appearing inside a data value rather than at stream start, when it is decoded, then it survives verbatim in the yielded payload.
- **S-ASD-028** *[test]* — Given a leading byte-order mark whose three bytes are divided across input chunks, when the chunks are fed in order, then the mark is still stripped exactly once.

### R-ASD-008 — An empty data accumulation dispatches nothing

When a dispatch is reached with an empty data accumulation, the decoder MUST yield **no** frame and MUST reset the accumulation and event-type state. (§ 9.2: "If the data buffer is an empty string, set the data buffer and the event type buffer to the empty string and return.")

#### Scenarios

- **S-ASD-029** *[test]* — Given a transcript of two consecutive blank lines and nothing else, when it is decoded, then zero frames are yielded.
- **S-ASD-030** *[test]* — Given a frame consisting of an event-type line followed by a blank line, with no data line at all, when it is decoded, then zero frames are yielded.
- **S-ASD-031** *[test]* — Given a well-formed frame preceded and followed by blank lines, when it is decoded, then exactly one frame is yielded.

### R-ASD-009 — Accumulation state resets after each dispatch

After a dispatch, both the event-type and data accumulations MUST be reset to empty. A subsequent frame carrying no event-type line MUST therefore dispatch with the **default** event type, never with the previous frame's type. A frame that dispatched nothing under `R-ASD-008` MUST likewise leave no residue.

**An empty event-type accumulation is one state, however it was reached.** § 9.2's dispatch tests the accumulation's *emptiness*, not whether a type line was ever seen. An event-type line with an **empty value** therefore sets the accumulation to the empty string and MUST dispatch with the **default** type — identically to a frame that carried no event-type line at all. A decoder that instead records "a type was assigned" and emits the empty string as the frame's type is defective. This is the exact edge the reset rule turns on, so it is pinned rather than left implied.

#### Scenarios

- **S-ASD-032** *[test]* — Given a frame with an explicit event-type line followed by a second frame with none, when both are decoded, then the second frame's type is the default type, not the first frame's.
- **S-ASD-033** *[test]* — Given two consecutive frames each with one data line, when both are decoded, then the second frame's payload contains none of the first frame's data.
- **S-ASD-034** *[test]* — Given an event-type-only frame that dispatches nothing, followed by a data-only frame, when both are decoded, then the single yielded frame carries the default type — the suppressed frame's type left no residue.
- **S-ASD-084** *[test]* — Given a frame carrying an event-type line with an **empty value** and one data line, when it is decoded, then the yielded frame carries the **default** event type — identical to the decoding of the same frame with no event-type line at all, and never an empty-string type.

### R-ASD-010 — The identifier and retry fields are tracked by nothing and observable in nothing

The decoder MUST track **no** last-event-identifier state and **no** retry-interval state, and MUST NOT surface either on a yielded frame. An identifier line MUST be ignored **whether or not** its value contains a NUL character, and a retry line MUST be ignored whether or not its value is all ASCII digits. § 9.2 permits this: ignoring is a legal disposition, and both fields are consumed by a reconnection mechanism this decoder does not implement.

The disposition MUST be pinned **by test rather than by accident** (doc 0002 AI-27.2 item 7): the NUL-bearing and NUL-free identifier cases and the digit and non-digit retry cases MUST each be present as fixtures, so that a future milestone adding identifier tracking has to consciously flip an existing pinned case instead of silently changing behavior nothing asserted.

The observable contract is therefore an **equivalence**: an identifier or retry line changes nothing about the frames yielded. It is deliberately **not** "the identifier state is left unchanged" — there is no such state to observe, and a scenario asserting one could never be shown failing first. (§ 9.2 for the underlying rule this ignoring is compatible with: "If the field value does not contain U+0000 NULL, then set the last event ID buffer to the field value. Otherwise, ignore the field."; retry: "If the field value consists of only ASCII digits… Otherwise, ignore the field.")

#### Scenarios

- **S-ASD-035** *[test]* — Given a frame containing an identifier line with an ordinary value, when it is decoded, then the yielded frame is identical to the decoding of the same frame without the identifier line.
- **S-ASD-036** *[test]* — Given two otherwise identical transcripts differing **only** in that one's identifier value contains a NUL character and the other's does not, when both are decoded, then the two yielded frame sequences are identical — the NUL case is not special-cased into any observable difference.
- **S-ASD-037** *[test]* — Given a retry line whose value contains a non-digit character, when it is decoded, then the yielded frames are identical to the decoding of the same transcript without the retry line.
- **S-ASD-038** *[test]* — Given a retry line whose value is all ASCII digits, when it is decoded, then the yielded frames are identical to the decoding of the same transcript without the retry line — the retry value never becomes frame content and never reaches a frame field.

### R-ASD-011 — An eighth grammar case forces an explicit split, never silent absorption

This node is at doc 0002's ~7-item split ceiling. If an eighth distinct grammar case is discovered while this milestone is open, it MUST be recorded as an explicit fractal split of the node (`AI-27.2.1` / `AI-27.2.2`) in the living graph, with its own requirements. It MUST NOT be absorbed silently into an existing requirement above.

#### Scenarios

- **S-ASD-039** *[inspection]* — Given the shipped requirements for this node, when a reviewer counts the distinct grammar items covered, then the count is exactly seven and each maps to one doc 0002 item.
- **S-ASD-040** *[inspection]* — Given a discovered eighth case, when a reviewer inspects the change, then a recorded node split exists — the absence of one with an eighth case present is a defect.

---

## AI-27.3 — Chunk-boundary re-entrancy `[leaf]`

### R-ASD-012 — Every golden transcript decodes identically at every split offset

For every golden transcript, and for **every** byte offset from zero through its length, splitting the transcript into two input chunks at that offset and feeding them in order MUST yield frames identical to the canonical unsplit decoding — same count, same order, same event types, same payload bytes. The property MUST be proven **mechanically over all offsets**, not by sampled or hand-picked offsets.

#### Scenarios

- **S-ASD-041** *[test]* — Given each golden transcript, when it is replayed once per byte offset split into two chunks, then every replay's frame sequence equals the unsplit reference sequence.
- **S-ASD-042** *[test]* — Given a split landing inside a field **name**, when the two chunks are fed, then the reconstructed field name is correct and the result matches the reference.
- **S-ASD-043** *[test]* — Given a split landing inside a multi-byte character of a data value, when the two chunks are fed, then the yielded payload bytes match the reference exactly.
- **S-ASD-044** *[test]* — Given a byte-order-mark-prefixed transcript split at offsets one and two — inside the three-byte mark itself — when the chunks are fed, then the mark is still stripped once and the result matches the reference.
- **S-ASD-045** *[test]* — Given the sweep harness's mismatch-message construction, factored into the separately callable pure function `sweepMismatch`, when it is called directly with a differing and an identical frame-sequence pair, then the message it produces for the differing pair names both the offending transcript and the exact byte offset, and no message is produced for the identical pair — a bare boolean mismatch would not be sufficient to locate the defect. Upgraded from `[inspection]` to `[test]`: the harness's message construction is factored into a separately callable pure function (`offset_sweep_test.go`'s `sweepMismatch`, extracted by Slice 3), so the construction itself is directly callable and failable without needing a real decoder defect to reach it.

### R-ASD-013 — A split between CR and LF MUST NOT inject a phantom blank line

A decoder that treats carriage return and line feed as independently line-ending turns one split CRLF into two terminators, injecting a spurious blank line at exactly that offset. The decoder MUST NOT do this. This case MUST be pinned by a dedicated scenario in addition to being covered by the sweep, because it is the single offset an example-based test essentially never selects.

#### Scenarios

- **S-ASD-046** *[test]* — Given a transcript containing at least one CRLF terminator, when it is split at the offset exactly between that CR and its LF and both chunks are fed, then the frame sequence equals the unsplit reference and no extra frame boundary appears.
- **S-ASD-047** *[test]* — Given a CRLF split as above **at a frame-terminating blank line**, when both chunks are fed, then exactly one dispatch occurs at that point, not two.
- **S-ASD-048** *[test]* — Given a lone carriage return that is the final byte of an input chunk and is **not** followed by a line feed in the next chunk, when both chunks are fed, then the carriage return terminates its line exactly once.

### R-ASD-014 — The multi-megabyte fixture is excluded from the offset sweep

The offset sweep is quadratic in transcript length. The `R-ASD-018` multi-megabyte fixture MUST be excluded from it; only the small golden transcripts participate. This exclusion is a **scope requirement**, not an optimization note: including the large fixture makes the sweep infeasible under any test budget. The large fixture MUST instead be covered by its own dedicated, non-exhaustive re-entrancy check.

The exclusion MUST be enforced **structurally**, by a guard the sweep itself applies to its own input set, rather than by convention. Every transcript admitted to the sweep MUST satisfy **size ≤ the sweep fixture bound (1024 bytes)** — one relation, one number, stated here so the assertion and the requirement read the same.

#### Scenarios

- **S-ASD-049** *[test]* — Given the golden transcript set that feeds the offset sweep, when each member's size is asserted against the sweep fixture bound, then every member satisfies size ≤ 1024 bytes, so the multi-megabyte fixture provably cannot enter the sweep — and a deliberately overweight fixture trips the guard.
- **S-ASD-050** *[test]* — Given the multi-megabyte fixture, when it is fed split at a small fixed set of representative offsets rather than exhaustively, then each replay matches the unsplit reference.
- **S-ASD-051** *[inspection]* — Given the shipped test sources, when a reviewer traces which fixtures the sweep enumerates, then the multi-megabyte fixture is not among them.

---

## AI-27.4 — Keep-alives and unknowns `[leaf]`

### R-ASD-015 — Comment lines are ignored without disturbing accumulation

A line beginning with a colon is a comment — the keep-alive idiom — and MUST be ignored. Ignoring it MUST NOT dispatch a frame, MUST NOT reset the event-type or data accumulation, and MUST NOT contribute any byte to a payload.

#### Scenarios

- **S-ASD-052** *[test]* — Given a frame whose data lines are interleaved with comment lines, when it is decoded, then the yielded payload is identical to the same frame with the comments removed.
- **S-ASD-053** *[test]* — Given a stream of comment lines only **ending at a line boundary**, when the end-of-input signal is given, then zero frames are yielded and no error is reported. The boundary qualifier is load-bearing: a trailing *unterminated* comment leaves retained line bytes and is a truncation under `R-ASD-023`, not a clean end.

### R-ASD-016 — Unknown field names are ignored; unknown event names are yielded

A field whose name is not one of the recognized field names MUST be ignored, per § 9.2. An **event type name** the decoder does not recognize MUST be yielded on the frame rather than dropped or replaced: § 9.2's parsing model has no known-versus-unknown event-type concept, so no registry, allowlist or recognition mechanism MUST be introduced — faithful compliance already produces this outcome.

#### Scenarios

- **S-ASD-054** *[test]* — Given a frame containing a field with an invented name alongside a valid data line, when it is decoded, then the yielded frame is identical to the same frame without the invented field.
- **S-ASD-055** *[test]* — Given a frame whose event-type line names a type this repository has never seen, when it is decoded, then the yielded frame carries that exact type string.
- **S-ASD-056** *[inspection]* — Given the shipped decoder source, when a reviewer looks for an event-type registry, allowlist or recognition table, then none exists.

### R-ASD-017 — An unexpected event-type line in the data-only dialect has defined behavior

The selected dialect is data-only, so an explicit event-type line is not expected. Its behavior MUST nonetheless be defined and tested rather than left latent: an event-type line MUST be honored exactly as § 9.2 states, setting the frame's type for that frame only, and MUST NOT be treated as an error, dropped, or allowed to leak into a later frame.

#### Scenarios

- **S-ASD-057** *[test]* — Given a data-only transcript into which one explicit event-type line is inserted, when it is decoded, then that frame carries the stated type, the surrounding frames carry the default type, and no error is reported.

---

## AI-27.5 — Bounded memory `[leaf]`

### R-ASD-018 — A single multi-megabyte frame decodes correctly

A frame several megabytes in size, below the configured hard cap, MUST decode correctly and completely. This pins the default-line-limit trap directly: a fixed small scan-token default is the canonical failure mode of this class, and large tool results reach it in practice. The decoder MUST NOT impose any implicit limit below its configured cap.

#### Scenarios

- **S-ASD-058** *[test]* — Given a single frame whose data payload is several megabytes and below the configured cap, when it is decoded, then exactly one frame is yielded and its payload length and content match the input exactly.
- **S-ASD-059** *[test]* — Given the same frame fed in many small input chunks, when all chunks are fed, then the yielded payload is identical to the single-chunk decoding.
- **S-ASD-060** *[test]* — Given a frame larger than sixty-four kibibytes but far below the cap, when it is decoded, then it succeeds — no implicit sub-cap limit exists.

### R-ASD-019 — A frame exceeding the hard cap aborts with a typed error naming the malformed-response category

When a single in-progress frame exceeds the configured hard cap, the decoder MUST abort with a **typed** error rather than continuing to grow its buffer. That error MUST name the AI-19 `malformed_response` category. The decoder MUST NOT construct a provider-failure value; it names a category from AI-19.2's vocabulary and remains wrappable by AI-28/AI-32, which own construction.

**Recorded compromise.** `malformed_response` is a deliberate, reasoned compromise and not a clean fit: an over-cap frame may be perfectly well-formed and merely too large. It is chosen because the nine-member vocabulary has no resource-exhaustion member and widening a shipped, promoted Layer 1 contract from inside a framing-only milestone is out of this milestone's charter.

**Frames completed before the trip are not collateral.** The abort discards the **over-cap frame** and only that. Any frame that already dispatched from the same input — before the cap was reached — MUST still be delivered to the caller **together with** the error, never dropped because an error accompanied it. This condition is reachable in one feeding step: a small complete frame followed by an over-cap one. Left unpinned, a caller writing the idiomatic "return on error" would silently lose delivered content — the same silent-loss failure class `R-ASD-023` forbids on the truncation side, and `S-ASD-075` already pins there.

#### Scenarios

- **S-ASD-061** *[test]* — Given a frame whose accumulated size passes the configured cap, when it is decoded, then decoding aborts with an error and no frame is yielded **for that over-cap frame** — the abort discards the offending frame, and this scenario makes no claim about other frames from the same input.
- **S-ASD-085** *[test]* — Given a single input chunk containing one complete frame followed by an over-cap frame, when it is fed, then the complete frame is returned **together with** the cap-exceeded error — the error accompanies delivered content rather than replacing it.
- **S-ASD-062** *[test]* — Given that abort, when the error's named category is read, then it is the malformed-response category.
- **S-ASD-063** *[test]* — Given a frame that reaches exactly the cap without exceeding it, when it is decoded, then it succeeds — the boundary is inclusive on the success side and the off-by-one is pinned.
- **S-ASD-064** *[inspection]* — Given the shipped decoder source, when a reviewer looks for a provider-failure construction call, then none exists — the decoder names a category and never builds a failure value.

### R-ASD-020 — Cap-exceeded and truncation stay distinguishable by error identity

Because both structural failures collapse onto one category, the decoder MUST expose **two distinct error identities** — one for cap-exceeded, one for truncation — so standard error-identity matching (`errors.Is`) separates them without inspecting message text. Collapsing them into a single identity is a defect, not a simplification.

#### Scenarios

- **S-ASD-065** *[test]* — Given a cap-exceeded error, when it is matched against the truncation identity, then it does not match.
- **S-ASD-066** *[test]* — Given a truncation error, when it is matched against the cap-exceeded identity, then it does not match.
- **S-ASD-067** *[test]* — Given each of the two errors, when each is matched against its own identity, then it matches — and both report the same AI-19 category, proving the distinction lives at the error level, not the category level.

### R-ASD-021 — The tenth-category escalation is recorded, not silently compromised

The change MUST record, as a deliberate deferral rather than an unstated assumption: that AI-19's vocabulary is documented **closed and append-only**, so appending a resource-exhaustion member is the sanctioned path; that the escalation trigger is a **second consumer** of the same concept — AI-30.1's per-call accumulation cap, which bounds the sum where this node bounds one frame; and that widening the shipped Layer 1 contract belongs to the taxonomy owner as its own change, never to this framing-only milestone.

#### Scenarios

- **S-ASD-068** *[inspection]* — Given this change's artifacts, when a reviewer looks for the tenth-category question, then a recorded deferral is present naming the append-only property, the second consumer, and the owning milestone.
- **S-ASD-069** *[inspection]* — Given the AI-19 capability spec and its source file, when a reviewer diffs them against the base branch, then neither is modified by this change.

---

## AI-27.6 — EOF discipline `[leaf]`

### R-ASD-022 — A clean end at a frame boundary finishes without error

When the end-of-input signal arrives with no partial frame pending — no accumulated data, no accumulated event type, no partial line — the decoder MUST finish reporting **no error**. An empty stream MUST likewise finish cleanly with zero frames.

#### Scenarios

- **S-ASD-070** *[test]* — Given a transcript ending with a complete frame and its terminating blank line, when the end-of-input signal is given, then no error is reported and the expected frames were yielded.
- **S-ASD-071** *[test]* — Given no bytes at all, when the end-of-input signal is given, then no error is reported and zero frames were yielded.
- **S-ASD-072** *[test]* — Given a transcript of comment lines ending at a line boundary, when the end-of-input signal is given, then no error is reported.

### R-ASD-023 — An end mid-frame is a typed truncation error and the partial frame is discarded

When the end-of-input signal arrives with a non-empty pending accumulation, the decoder MUST report a **typed truncation error** naming the AI-19 `malformed_response` category, and MUST **discard** the partial frame. It MUST NOT dispatch the partial accumulation as a complete frame. Reporting success on a half-answer is the exact failure mode this requirement exists to prevent.

#### Scenarios

- **S-ASD-073** *[test]* — Given a transcript cut in the middle of a data value, when the end-of-input signal is given, then a truncation error is reported and no frame is yielded for the partial content.
- **S-ASD-074** *[test]* — Given a transcript whose final frame has complete data lines but **no** terminating blank line, when the end-of-input signal is given, then a truncation error is reported and that frame is not dispatched.
- **S-ASD-075** *[test]* — Given a transcript with two complete frames followed by a partial third, when the end-of-input signal is given, then the two complete frames were yielded and the third is absent, and the truncation error is reported.
- **S-ASD-076** *[test]* — Given the truncation error, when its named category is read, then it is the malformed-response category, and it is distinguishable from cap-exceeded per `R-ASD-020`.

### R-ASD-024 — End-of-input is a decoder signal, not a transport condition

The translation of a transport read-loop end into the end-of-input signal is **outside** this milestone: the caller performs it. The decoder MUST NOT interpret, produce or depend on transport end-of-file values, and MUST NOT return them. Its finishing behavior MUST be observable purely from the explicit end-of-input signal.

#### Scenarios

- **S-ASD-077** *[test]* — Given a decoder that has been fed bytes but has not received the end-of-input signal, when the yielded frames so far are inspected, then only frames whose terminating blank line already arrived are present, and no error has been reported.
- **S-ASD-078** *[inspection]* — Given the shipped decoder source, when a reviewer looks for transport end-of-file values in returned errors, then none is produced or returned by the decoder.

---

## Charter boundary

### R-ASD-025 — The decoder is framing-only, and a violation is detectable by inspection

The decoder MUST NOT interpret payload semantics; MUST NOT recognize the dialect's terminal sentinel — a sentinel that appears **nowhere** in § 9.2 and whose recognition doc 0002 assigns to AI-28.2 — and MUST yield it verbatim as ordinary data; MUST perform no transport work of any kind; and MUST map no wire failure into the AI-19 taxonomy, which is AI-32's charter. Any behavior conditioned on the *content* of a payload is a boundary violation.

#### Scenarios

- **S-ASD-079** *[test]* — Given a frame whose data value is the dialect's terminal sentinel string, when it is decoded, then it is yielded as an ordinary frame carrying that exact string as its payload, with no special status, no early finish and no error.
- **S-ASD-080** *[test]* — Given frames following that sentinel frame in the same stream, when they are decoded, then they are yielded normally — the decoder suppresses nothing after the sentinel.
- **S-ASD-081** *[test]* — Given two frames whose payloads differ only in content but not in framing, when both are decoded, then their framing outcomes are identical — no decision depends on payload content.
- **S-ASD-082** *[inspection]* — Given the shipped decoder source, when a reviewer searches for the terminal-sentinel literal, then it appears only in test fixtures and never in decoder logic.
- **S-ASD-083** *[inspection]* — Given the module manifest, when a reviewer diffs it against the base branch, then it is unchanged — the decoder adds no dependency, and any added requirement is a hard blocker rather than a tradeoff.
