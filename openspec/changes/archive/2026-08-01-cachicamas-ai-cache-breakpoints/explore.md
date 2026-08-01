# Exploration — cache-boundary markers

> **Change**: `cachicamas-ai-cache-breakpoints`
> **Milestone**: AI-11 · **Nodes**: AI-11.1, AI-11.2, AI-11.3, all `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-01
> **Worktree**: `cachicamas-worktrees/ai-11` · **Branch**: `feat/ai-11-cache-breakpoints` · **Base**: `07d2027`
> **Depends on**: AI-10 · **Blocks**: AI-24, AI-26.2
> **Closes**: the Layer 1 half of **G4** — doc 0001 § 7 — jointly with AI-10

---

## 1. The concurrency assumption, stated first

**AI-10 is being finished in a sibling worktree while this plan is written.** The base of this branch is `07d2027`, which carries AI-10.1 and AI-10.2 complete, plus the first two commits of AI-10.3 (`ErrMisplaced` appended; message and content order pinned). Landing concurrently in `../ai-wave-1`: the rest of AI-10.3 (items 2–4), AI-10.4, AI-10.5, AI-10.6.

So this plan is written against **two different kinds of fact**:

- **Observed** — read from the tree at `07d2027`. `Request`, `Segment`, `SystemInstruction`, `Message`, `Tool`, `ToolSet`, the AI-04 sentinels.
- **Promised** — stated in `openspec/changes/cachicamas-ai-model-request/design.md` §§ 5–12 as `[provisional]`, and in its `spec.md` as *(planned)*. Nothing promised has been verified against code, because the code does not exist yet.

§ 8 is the register of every promised fact this plan leans on, with the file and symbol to re-check at apply time. **The resuming agent re-verifies § 8 before writing the first test.** A promise that moved is a `revert-and-record` event under doc 0002's living-graph clause, not a reason to improvise.

## 2. What AI-11 is being asked for

doc 0002 lines 713–747, the charter:

| Field | Value |
| --- | --- |
| Goal | Make prompt-cache boundaries expressible on the request. |
| Deliverable | Cache-boundary markers on system segments, tool declarations and messages; a documented cap; the tools → system → messages invalidation ordering readable by an adapter. |
| Acceptance | A request can express a legal breakpoint set, and an adapter can either render it or ignore it whole — markers are **advisory by contract**. |
| Out of scope | Measuring cache hit rates; usage accounting (AI-13.3 already carries cache-read and cache-write token fields, and this milestone adds nothing there). |

Three register terms are already written and are the binding vocabulary. This milestone **implements** them; it does not invent them.

| Term | Register text, compressed | Owner |
| --- | --- | --- |
| `V-REQ-23` **cache-boundary marker** | An **advisory** mark on a system-instruction segment, a tool declaration, or a message, indicating a point at which a provider may cache the preceding prefix. An adapter for an auto-caching provider ignores markers wholesale; one for an explicit provider honours them. **A marker is never a correctness requirement.** | AI-11 |
| `V-REQ-24` **breakpoint cap** | The documented maximum number of markers a request may carry. At least one provider enforces a small hard cap, and exceeding it is a caller-contract failure rather than a silent truncation. | AI-11 |
| `V-REQ-25` **invalidation cascade** | The stated ordering — tools, then system instruction, then messages — in which a change invalidates cached prefixes downstream of it. Readable by an adapter; **Layer 1 never measures a hit rate.** | AI-11 |

Why it cannot wait, in one line from doc 0001 § 3.2: caching is opt-in per breakpoint on at least one provider, cached reads cost roughly a tenth of fresh input, and *a design that cannot express a breakpoint can never obtain that discount — the omission is invisible until the bill arrives.*

## 3. Current state — what exists at `07d2027`

`backend/agent/src/ai/`, 12 504 lines across 35 files, `go.mod` at zero requires.

### 3.1 The three things a marker attaches to

| Type | File | Shape today | Ordered and addressable? |
| --- | --- | --- | --- |
| `Segment` | `system_instruction.go:13` | `struct{ text string }` — comparable, `IsZero()` is `text == ""` | Yes — `SystemInstruction.Segments() []Segment`, index is the ordinal |
| `Tool` | `tool.go:20` | `struct{ name, description string; schema []byte }` — **not** comparable, zero detector is `Name() == ""` | Yes — `ToolSet.Tools() []Tool`, order is the caller's by contract (`V-REQ-14`) |
| `Message` | `message.go:112` | `struct{ id MessageID; role Role; content []Part }` — not comparable, zero detector is `ID().IsZero()` | Yes — `Request.Messages() []Message` |

All three are **value types with unexported fields, constructed through a validating constructor, copied in and copied out**. That uniformity is what makes one marker mechanism work on all three.

### 3.2 The seam AI-10 left, verbatim

`cachicamas-ai-model-request/design.md` § 12.1 — *"AI-11 — cache-boundary markers"*:

> **Nothing in the request shape changes when markers arrive.** A marker attaches to the things that are already ordered and individually addressable: `Segment`, `Tool` and `Message`.
>
> - `Segment` is a struct with one unexported field, not a string alias. Adding `marked bool` is a field, not a type change, and no signature moves.
> - `NewSegment(text string) (Segment, error)` stays; AI-11 adds a sibling constructor or an option, and every existing caller compiles untouched.
> - `Segments()` returns `[]Segment`, so a marker read is `segments[i].IsCacheBoundary()` and needs no new accessor on `SystemInstruction`.
> - The request-level breakpoint **cap** (`V-REQ-24`, `ErrOutOfRange`) inserts into § 4's order as a new row after rule 5. The order is a slice; inserting a row is one line.
> - The tools → system → messages **read order** AI-11.2 needs is already the order § 4 validates in, and already the order the regions are documented in.

`system_instruction.go:148` says the same thing at the point of use: *"AI-11 may extend this to name a cache marker, because a marker is structural rather than payload."*

**The reader's name is therefore already chosen for us**: `IsCacheBoundary() bool`. This plan adopts it rather than inventing a synonym.

### 3.3 No new sentinel is needed

`validation.go:81` already carries the class, and its GoDoc already cites this milestone's term:

```go
// ErrOutOfRange is the class for a value outside a documented bound.
// V-REQ-24: a request exceeding the documented breakpoint cap is a
// caller-contract failure rather than a silent truncation.
ErrOutOfRange = errors.New("value is outside a documented bound")
```

The class fits verbatim — a count crossing a documented ceiling is a value outside a documented bound — so AI-04's append rule does not fire, and **`ruleClasses` and its two registry mirrors are untouched by this milestone.** This is the opposite of AI-08.2 (`ErrDuplicate`) and AI-10.3 (`ErrMisplaced`), and the reason is worth stating: those two met a violation no landed class described; this one meets a class landed *for it*, by name.

### 3.4 What the usage side already carries — the out-of-scope proof

`usage.go:139–143` holds `CacheRead` and `CacheWrite` as `TokenCount` fields, disjoint from `Input`, validated by `Usage.Validate` at `cache_read` and `cache_write`. AI-13.3 landed them. **doc 0001 § 7 G10 records that Layer 1's obligation there is already met.** This milestone adds request-side expression only, and AI-11.3 item 2 pins that as a test rather than as a sentence.

## 4. The three shape questions, and the options for each

### 4.1 How is a marker set?

| Approach | Shape | Pros | Cons | Effort |
| --- | --- | --- | --- | --- |
| **A — sibling constructors** | `NewMarkedSegment(text)`, `NewMarkedTool(...)`, `NewMarkedMessage(...)` | Marker decided at construction; no post-construction state change | Three constructors duplicated, each re-stating its parent's rules; `NewMessage` is variadic, so the marked twin is awkward; the surface doubles | Medium |
| **B — copy-returning method** | `func (s Segment) MarkCacheBoundary() Segment`, same on `Tool` and `Message` | One spelling on all three despite three different constructor shapes; **no existing signature moves at all**; value semantics preserved (returns a copy, mutates nothing); trivially composable — `seg.MarkCacheBoundary()` reads as what it is | A caller could mark an unconstructed value — closed by a documented rule and a test (§ 4.4) | **Low** |
| **C — request-level marker set** | `WithCacheBoundaries(CacheRegionSystem, 0)` on the request | Cap check is local; nothing on the three types changes | Markers drift from what they mark the moment AI-12 rebuilds a request with a different message list; contradicts `V-REQ-23`, which places the mark **on** the segment/declaration/message; an index-based marker cannot survive a re-ordering | Medium |

**Recommendation: B.** It is the only one that adds zero signature churn, and `V-REQ-23` puts the mark on the thing, not beside it. C is rejected on a correctness argument, not an ergonomic one.

### 4.2 How is the cascade order made *readable*?

The test-list item is: *"An adapter can read markers in tools → system → messages order regardless of the order in which they were set."*

| Approach | Pros | Cons |
| --- | --- | --- |
| **D — documentation only.** The adapter loops `Tools()`, then `Segments()`, then `Messages()`, checking `IsCacheBoundary()`. | Zero new surface | The order is not observable, so the test would assert the loop the test itself wrote — a tautology. The item would close on nothing. |
| **E — one enumerator on the request.** `Request.CacheBoundaries() []CacheBoundary`, where a `CacheBoundary` names its region and its ordinal, and the slice is in cascade order. | Cascade order is a property of a returned value, so it is mechanically assertable; **the cap rule and the enumerator share one walk**, so a count and a listing cannot disagree; an adapter gets the whole breakpoint set in one call, which is what AI-26.2 will consume | One small closed vocabulary (`CacheRegion`, three members) and one small value type |

**Recommendation: E.** The decisive argument is not ergonomics — it is that **the cap rule needs a count anyway**, and a count computed by a private walk beside a listing computed by a public one is two walks that can drift. One walk, two callers, which is AI-06's `decision.md` § 7.2 principle applied one region over.

`CacheRegion` follows the package's landed closed-vocabulary pattern (`role.go`'s file comment calls itself *"the pattern every later closed vocabulary in this package reuses"*): constants from `iota + 1`, an enumerator walking the constant space rather than the name table, a `String()`, and an exhaustiveness pin. **The numeric order of the three members is the cascade order** — that is the whole reason the type is ordered rather than a set.

### 4.3 Where does the cap rule sit in the documented rule order?

AI-10's `design.md` § 4 holds a ten-row ordered table, and § 12.1 says the cap inserts *"after rule 5"* (rule 5 = an applied system instruction was constructed).

**That placement does not survive contact with rule 9.** The cap counts markers across **three** regions, and one of them is the tool set, which rule 9 admits. Counting at position 6 would count a tool set that a later rule then rejects, so a request could be told about a cap it only appears to exceed.

**Recommendation: the cap becomes rule 10, and generation-option bounds move to 11.** This keeps AI-10's own § 4.1 principle — *structure before parameters* — because the cap is a structural fact of the request and an option bound is a parameter. It is a recorded deviation from a `[provisional]` line in AI-10's design, and `design.md` § 4 of this change states it with its reason. It does **not** edit AI-10's design.

### 4.4 What does a marker do to validity, and to equality?

Two different questions with two different answers, and conflating them is the trap.

- **Validity: nothing.** AI-11.1 item 2 requires that a marked request and its unmarked twin validate identically. A marker is advisory; it can never be the reason a request is rejected. The **one** exception is the cap — and the cap is a property of the *count*, not of any single marker.
- **Equality: markers participate.** Two requests differing only in their markers are different requests: they ask for different cache behaviour. If AI-10.6's documented equality excluded markers, AI-10.5's round-trip walk would pass while **silently dropping every marker** — a defect with no error, no wrong answer, and an input bill roughly ten times larger, which is precisely the failure mode `tool_set.go` was shaped to avoid.

So AI-11.1 item 2's *"and are otherwise equal"* means: equal in every region's payload, differing only in marker readback. The spec says so explicitly rather than leaving "otherwise" to the reader.

**A marker cannot resurrect an unconstructed value.** `MarkCacheBoundary()` on a zero `Segment`, `Tool` or `Message` returns the receiver unchanged, so the three zero-detectors (`text == ""`, `Name() == ""`, `ID().IsZero()`) stay sound and a marked zero value cannot slip past `NewSystemInstruction`, `NewToolSet` or `NewRequest`. This is a discovered case, appended to AI-11.1's test list.

### 4.5 What is the cap's value?

`MaxCacheBoundaries = 4`, exported.

- **The number.** doc 0001 § 3.2 and doc 0002's AI-11 note both say *"a hard cap on breakpoint count"* and *"a small hard cap"*, without stating it, because doc 0001 is deliberately vendor-neutral in its prose. Four is the cap the opt-in provider enforces. Choosing the smallest cap any target provider enforces is the same intersection-not-union argument `tool.go:84` makes for tool names, and it carries the same rollback asymmetry: **raising a cap later is additive** (a request that was rejected starts being accepted), **lowering one is breaking**.
- **Exported.** Precedent is `MaxToolNameLen` (`tool.go:12`), exported because *"a caller that generates tool names needs the ceiling before it constructs rather than after it fails."* A Layer 2 pre-request hook placing breakpoints is exactly that caller.

## 5. Affected areas

| Path | Impact | What changes |
| --- | --- | --- |
| `backend/agent/src/ai/cache_boundary.go` | **New** | `CacheRegion` vocabulary, `CacheBoundary`, `MaxCacheBoundaries`, `Request.CacheBoundaries()`, the cap rule |
| `backend/agent/src/ai/system_instruction.go` | Modify | `Segment.cacheBoundary` field, `MarkCacheBoundary`, `IsCacheBoundary`, extended `Segment.String()` |
| `backend/agent/src/ai/tool.go` | Modify | `Tool.cacheBoundary` field, `MarkCacheBoundary`, `IsCacheBoundary` |
| `backend/agent/src/ai/message.go` | Modify | `Message.cacheBoundary` field, `MarkCacheBoundary`, `IsCacheBoundary` |
| `backend/agent/src/ai/request.go` | Modify | one rule inserted into the rule slice; `Request.String()` names the boundary count |
| `backend/agent/src/ai/cache_boundary_test.go` | **New** | AI-11.1 and AI-11.2, package `ai_test` |
| `backend/agent/src/agenttest/cache_boundary_test.go` | **New** | AI-11.3 — the marker-blind translator and its control |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **Untouched** | Read and cited only. `V-REQ-23`, `V-REQ-24`, `V-REQ-25` are already written. |

**No file outside `backend/agent/src/` and this change's directory is edited.**

## 6. Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| AI-10.3 item 2 lands `Request.Tools()` with a different name or shape than assumed, so the cascade walk cannot read the tool region | **Medium** | Blocks AI-11.2 entirely | § 8 row 1. Re-verify before the first test. The walk is one function; adapting it to a renamed accessor is a rename, not a redesign. |
| AI-10.6 does not export `Equal`, so AI-11.1 item 2's "otherwise equal" has no comparison to lean on | Medium | Slows AI-11.1 item 2 | § 8 row 4. Fallback: compare region-wise inside this milestone's own test helper, which is what AI-10.5 would have to do anyway. |
| AI-10.6 exports `Equal` **excluding** markers | Low | Silent marker loss on round trip — the expensive defect class | § 8 row 5, and AI-11.1 item 2 is written to fail if it happens. Escalate to the orchestrator rather than patching around it. |
| The cap rule is inserted at a position that changes which failure a landed AI-10 test observes | Low | Red tests in a sibling's landed work | The cap is the last structural rule; every AI-10 test that expects an earlier class still gets it because `FirstFailure` short-circuits. Confirmed by reading the rule slice, re-confirmed by `make test`. |
| Extending `Segment.String()` breaks `system_instruction_test.go:288`, which asserts `"segment"` exactly | **Low** | Red test | Only a **marked** segment renders differently; the unmarked rendering is unchanged, so that assertion stays green. Verified by reading the assertion. |
| Review budget: doc 0002 asks for < 250 changed lines and a reassessment before 400 | **Certain to fire** | Reviewer load | `tasks.md` records the Review Workload Forecast and the reassessment. The wave PR budget is 5000+ and was accepted up front as `exception-ok`. |

## 7. Review budget, forecast before any code

Split trigger 4 (*projected diff pushes the milestone past the review budget*) **fires**, and it is recorded here, before a line of Go, exactly as `explore.md` § 7 did for AI-10 and AI-06.

| Slice | Production | Test | Note |
| --- | --- | --- | --- |
| SDD planning artifacts | — | — | ~1 100 prose lines across five files |
| AI-11.1 markers | ~150 | ~260 | Three types, one mechanism; the weight is contract documentation |
| AI-11.2 cap and cascade | ~170 | ~240 | New file `cache_boundary.go` plus one rule row |
| AI-11.3 advisory semantics | ~10 | ~170 | Almost all test; the marker-blind translator and its control |
| **Total Go** | **~330** | **~670** | **~1 000 changed Go lines** |

Split trigger 1 does not fire — three leaves, one publicly observable behaviour family. The mitigation is the same one AI-10 used and doc 0002 prescribes: **the leaf boundary is the commit boundary**, so the chain is reviewable in slices without rework. Nothing is cut to fit; discovered cases are appended.

## 8. Assumptions about AI-10's final surface — the re-verification register

**The resuming agent runs this table before writing the first test.** Each row names the exact file and symbol to check in the merged tree. A row that fails is a `revert-and-record` event: record the discovered prerequisite in this file and in `design.md`, then proceed against what is actually there.

| # | Assumed | Source of the promise | File · symbol to re-check | If it moved |
| --- | --- | --- | --- | --- |
| 1 | `Request` carries an optional tool set, attached by `WithTools(ToolSet)` and read by `Request.Tools() (ToolSet, bool)` | AI-10 `spec.md` `R-AMR-010`, `S-AMR-038`; `design.md` § 4 rule 9 | `backend/agent/src/ai/request.go` · `requestDraft`, `WithTools`, `Request.Tools` | Adapt the cascade walk's tools leg to the landed accessor. **Blocking for AI-11.2 item 2.** |
| 2 | `Request` carries an optional tool choice, `WithToolChoice` / `Request.ToolChoice()` | AI-10 `spec.md` `R-AMR-010` | `backend/agent/src/ai/request.go` · `WithToolChoice`, `Request.ToolChoice` | No effect — a tool **choice** carries no marker. Listed so its absence is not mistaken for a gap. |
| 3 | The documented rule order is a `[]Rule` passed to `FirstFailure` inside `NewRequest`, with rules 6–9 inserted by AI-10.3 at their numbered positions | AI-10 `design.md` § 4; observed at `07d2027` for rules 1–5, 10 | `backend/agent/src/ai/request.go` · `NewRequest`, the `FirstFailure(...)` argument list | Insert the cap rule at the last structural position, whatever its number turns out to be, and update `design.md` § 4 of this change. |
| 4 | `Request.Equal(other Request) bool` is **exported**, with matching `Message.Equal` and `SystemInstruction.Equal` | AI-10 `design.md` § 11.2 — *"the default recorded here is **yes**"* | `backend/agent/src/ai/request.go` (or a new `equality.go`) · `Request.Equal` | Compare region-wise in this milestone's own test helper. Non-blocking. |
| 5 | That equality **includes** marker state once markers exist | Not promised anywhere — **this milestone's own requirement**, `R-ACB-004` | Same symbol as row 4 | If AI-10.6 landed an equality that ignores markers, AI-11.1 item 2 fails and the fix is in `Equal`, not here. Escalate. |
| 6 | AI-10.5's cross-package round-trip walk lives in `backend/agent/src/agenttest/` and is driven from `PartKinds()` | AI-10 `design.md` § 10 | `backend/agent/src/agenttest/` · the round-trip test file | AI-11.3's marker-blind translator is written independently and does not consume the walk; if the walk exists, extend it to carry a marked request rather than duplicating it. |
| 7 | `ErrMisplaced` and its two registry mirrors are settled, so `validation.go`'s `ruleClasses` is stable | Observed at `07d2027` (`7de9698`) | `backend/agent/src/ai/validation.go` · `ruleClasses`; `validation_registry_internal_test.go`; `validation_test.go` | This milestone appends **no** class, so a further append by AI-10.3 is invisible here. Confirm `ErrOutOfRange` is still registered. |
| 8 | `Segment` stays a comparable struct; `Tool` and `Message` stay non-comparable value types with the zero-detectors named in § 3.1 | Observed at `07d2027` | `system_instruction.go` · `Segment`; `tool.go` · `Tool`; `message.go` · `Message` | Adding `cacheBoundary bool` keeps `Segment` comparable. If a sibling made `Segment` non-comparable, AI-11.1's round-trip assertions switch to field-wise comparison. |

## 9. Deliberately unsupported, and by whom

Named so each is a decision rather than an omission.

| Not supported | Why | Owner |
| --- | --- | --- |
| Measuring cache hit rates | `V-REQ-25`: *"Layer 1 never measures a hit rate."* A hit rate is observed on the response, and Layer 1 reports usage without pricing or aggregating it | Layer 2 / Layer 3 (`V-OUT-07` cost) |
| Any change to the usage record | AI-13.3 already carries `CacheRead` and `CacheWrite`; doc 0001 § 7 G10 records Layer 1's obligation as met | AI-13 (landed) |
| Rendering a marker onto a provider's wire shape | That is an adapter's, and this milestone exists so an adapter *can* | AI-24, AI-26.2 |
| Prefix stability across turns | G4's Layer 2 half — *"L1 places, L2 stabilises"* | Layer 2 |
| Deciding **where** breakpoints should go | Placement policy needs the transcript and the budget; the pre-request hook (seam 1) is where it happens | Layer 2 |
| A marker on a tool **choice**, a content **part**, or a generation option | `V-REQ-23` names exactly three carriers. A part is not a cache prefix boundary on any target provider, and adding a fourth carrier later is additive | Nobody, until a provider needs it |
| Validating that markers are *useful* — that a marked prefix is long enough to be worth caching | Minimum cacheable prefix length is per provider and per model, decidable only by asking. It is a provider failure, the same treatment `request.go` gives temperature's upper bound | AI-19 |

## 10. Recommendation

Implement **B + E**: a copy-returning `MarkCacheBoundary()` on `Segment`, `Tool` and `Message` with an `IsCacheBoundary()` reader, plus one `Request.CacheBoundaries()` enumerator over a three-member ordered `CacheRegion` vocabulary that is both the adapter's read path and the cap rule's count.

Order of work follows doc 0002's leaf order and its walking-skeleton rule: AI-11.1 lands the thinnest end-to-end path (mark a segment, read it back from another package), AI-11.2 widens it to the request-level count and the cascade, AI-11.3 proves the advisory contract. Error paths follow happy paths.

**Ready for proposal: yes**, conditional on § 8 being re-verified at apply time rather than at plan time.
