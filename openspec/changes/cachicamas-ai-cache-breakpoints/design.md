# Design — cache-boundary markers

> **Change**: `cachicamas-ai-cache-breakpoints`
> **Milestone**: AI-11 · **Nodes**: AI-11.1, AI-11.2, AI-11.3, all `[leaf]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-01
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-cache-breakpoints/spec.md`, the register, AI-04 … AI-10 as landed at `07d2027`
> **Owns**: every Go spelling, the cap's value and its position in the rule order, the validity/equality split, the rendering extension, and the shape of AI-11.3's translator

---

## 1. Technical approach

Three leaves, one mechanism, one new file.

A marker is **one unexported boolean field on each of the three carriers**, placed by a copy-returning method with the same name on all three and read by an accessor whose name AI-10's `design.md` § 12.1 already chose. The request grows **one rule** and **one accessor**; nothing else in the request shape moves, which is exactly what AI-10 promised when it left the seam.

The cap and the cascade share one walk. That is the load-bearing decision of this design: a count and a listing computed by two walks are two things that can disagree, and the symptom of disagreement here is a request that passes validation carrying more breakpoints than the provider accepts.

**Status marks.** Every section is `[planned]` — nothing in this milestone is implemented. Sections that depend on a fact AI-10 has promised but not landed carry `[assumes AI-10]` and cite the row of `explore.md` § 8 that must be re-verified first.

## 2. Architecture decisions

### Decision: a copy-returning method, not a sibling constructor and not a request-level marker set

**Choice**: `func (c Carrier) MarkCacheBoundary() Carrier` on `Segment`, `Tool` and `Message`, plus `func (c Carrier) IsCacheBoundary() bool`.

**Alternatives considered**: marked sibling constructors (`NewMarkedSegment`, …); a request-level set of `(region, index)` pairs applied as a `RequestOption`.

**Rationale**: the three carriers have three different constructor shapes — `NewSegment(text)`, `NewTool(name, description, schema)`, `NewMessage(role, content...)` — so sibling constructors would be three duplications of three different rule sets, and the variadic one is awkward to twin. A method is one spelling on all three and **moves no existing signature at all**. The request-level set is rejected on correctness, not ergonomics: `V-REQ-23` places the mark *on* a segment, a declaration or a message, and an index-based marker held beside the regions silently points at the wrong carrier the moment AI-12 rebuilds a request with a different message list.

The return-a-copy shape is not a stylistic choice either. Every carrier is a value type that a caller may already have shared — `Segments()`, `Tools()` and `Messages()` each hand out copies precisely so two consumers cannot observe each other — and a mutating marker would reintroduce the aliasing those three accessors exist to prevent.

### Decision: `MarkCacheBoundary` on an unconstructed value is a no-op

**Choice**: each method returns the receiver unchanged when the receiver never passed its constructor.

**Alternatives considered**: set the flag anyway; return an error.

**Rationale**: each carrier's zero detector reads its payload — `text == ""`, `Name() == ""`, `ID().IsZero()` — and those three detectors are what `NewSystemInstruction`, `NewToolSet` and `NewRequest` use to reject a value that skipped its constructor. Setting a flag on a zero value would produce a value that is simultaneously unconstructed and marked, and the next reader would have to know which of two questions to ask first. Returning an error was rejected because it would make marking the only fallible operation on a carrier, and `V-REQ-23` says a marker is never a correctness requirement — an advisory mark that can fail is a contradiction on the face of it.

### Decision: `MaxCacheBoundaries = 4`, exported

**Choice**: an exported untyped constant, value 4.

**Alternatives considered**: unexported with an accessor; a larger cap; no cap with adapter-side truncation.

**Rationale**: the cap is the smallest hard cap any target provider enforces, taken as the **intersection rather than the union** for `tool.go:84`'s reason and with its rollback asymmetry — raising a cap later is additive, lowering one is breaking. Exported for `MaxToolNameLen`'s stated reason: *"a caller that generates tool names needs the ceiling before it constructs rather than after it fails"*, and the Layer 2 pre-request hook that places breakpoints (doc 0001 § 6 seam 1) is exactly that caller. Adapter-side truncation is the failure mode `V-REQ-24` names by name.

### Decision: the cap is rule 10, not rule 6 `[assumes AI-10]`

**Choice**: the cap rule runs after every structural and cross-region rule, and before the generation-option bounds. Generation-option bounds move from position 10 to position 11.

**Alternatives considered**: AI-10's `design.md` § 12.1 line — *"inserts into § 4's order as a new row after rule 5"*.

**Rationale**: the count spans three regions, and one of them is the tool set, which AI-10's rule 9 admits. Counting at position 6 would count a tool set that a later rule then rejects, so a caller could be told about a cap they only appear to exceed. Moving it last among the structural rules preserves AI-10's own § 4.1 principle — *structure before parameters* — because the cap is a structural fact of the request and an option bound is a parameter.

This is a **recorded deviation** from a `[provisional]` line in another change's design, taken under that document's own rule (*"may only change it by recording why, before writing the test"*). It edits nothing in `cachicamas-ai-model-request/`. Re-verify `explore.md` § 8 row 3 before implementing: if AI-10.3 landed a different final position, take the last structural slot, whatever its number.

### Decision: markers are outside validity and inside equality

**Choice**: no marker may cause a request to fail or succeed (`R-ACB-005`); marker state is compared by the documented request equality (`R-ACB-004`). The cap is the one rule about markers, and it is about the count.

**Alternatives considered**: markers inside validity (placement rules by role or adjacency); markers outside equality.

**Rationale**: placement rules were rejected because `V-REQ-23` makes a marker advisory and doc 0001 § 3.3 row 9 makes ignoring markers a conformant adapter strategy — a placement a provider cannot use is that provider's business, and *whether a marked prefix is long enough to cache* is decidable only by asking, so it is AI-19's, the same treatment the request already gives an unrecognised model identity and an over-high temperature. Excluding markers from equality was rejected because it makes the whole-request round trip of `R-AMR-015` pass while dropping every marker: no error, no wrong answer, and an input bill roughly ten times larger — `tool_set.go`'s canonical failure mode, one region over.

### Decision: `CacheRegion` is a closed ordered vocabulary, and its numeric order is the cascade

**Choice**: three constants from `iota + 1`, an enumerator walking the constant space, a `String()`, and an exhaustiveness pin.

**Alternatives considered**: three untyped string labels; a comparison written at each call site.

**Rationale**: `role.go`'s file comment calls its own shape *"the pattern every later closed vocabulary in this package reuses"*, and AI-07, AI-08 and AI-13 each reused it. Deviating here would make this the one closed vocabulary in the package with different rules. The specific value the pattern adds: **the cascade order becomes a property of the vocabulary** rather than of a sort comparator someone could get backwards, and the enumerator is what makes `S-ACB-031`'s pin non-decorative — `role.go` records that an enumeration derived from the *name table* rather than the constant space would make a member declared without an entry invisible to every assertion over it.

## 3. Interfaces / contracts

### 3.1 The marker, on each carrier `[planned]`

```go
// system_instruction.go
type Segment struct {
	text          string
	cacheBoundary bool // AI-11.1
}

func (s Segment) MarkCacheBoundary() Segment // no-op on the zero segment
func (s Segment) IsCacheBoundary() bool

// tool.go
type Tool struct {
	name, description string
	schema            []byte
	cacheBoundary     bool // AI-11.1
}

func (t Tool) MarkCacheBoundary() Tool // no-op when Name() == ""
func (t Tool) IsCacheBoundary() bool

// message.go
type Message struct {
	id            MessageID
	role          Role
	content       []Part
	cacheBoundary bool // AI-11.1
}

func (m Message) MarkCacheBoundary() Message // no-op when ID().IsZero()
func (m Message) IsCacheBoundary() bool
```

Three notes the implementer needs:

- **`Segment` stays comparable.** Adding a `bool` keeps `==` defined, which AI-10's `design.md` § 3 relies on for round-trip assertions. A marked segment and its unmarked twin are therefore `!=` under `==`, which is exactly `R-ACB-004` at segment scope.
- **`Tool` and `Message` remain non-comparable** (`[]byte`, `[]Part`), so `S-ACB-006`'s idempotence is asserted field-wise: `slices.Equal` on the payload plus `IsCacheBoundary()`.
- **Marking a `Message` preserves its identity.** `message.go` says *"a copy carries the same identity, because a copy is the same message"*, and the marked value is a copy.

### 3.2 The cascade, the cap, and the one walk `[planned]` `[assumes AI-10]`

New file `backend/agent/src/ai/cache_boundary.go`:

```go
// MaxCacheBoundaries is the documented ceiling on the markers one request may carry.
const MaxCacheBoundaries = 4

// CacheRegion is the closed vocabulary of the three regions the invalidation
// cascade runs through, in cascade order (V-REQ-25).
type CacheRegion uint8

const (
	CacheRegionTools CacheRegion = iota + 1
	CacheRegionSystem
	CacheRegionMessages
)

func CacheRegions() []CacheRegion       // walks the constant space, not the name table
func (r CacheRegion) String() string

// CacheBoundary names one marker: its region and its ordinal within that region.
// Comparable, so a test asserts a whole sequence with slices.Equal.
type CacheBoundary struct {
	region CacheRegion
	index  int
}

func (b CacheBoundary) Region() CacheRegion
func (b CacheBoundary) Index() int
func (b CacheBoundary) String() string

// CacheBoundaries returns the request's markers in tools → system → messages
// order, ascending by ordinal within each region. Fresh slice per call.
func (r Request) CacheBoundaries() []CacheBoundary

// cacheBoundaries is the one walk. The cap rule and the accessor both call it,
// so a count and a listing cannot disagree.
func (d requestDraft) cacheBoundaries(messages []Message) []CacheBoundary

// cacheBoundaryCapRule is rule 10 of NewRequest's documented order.
func (d requestDraft) cacheBoundaryCapRule(messages []Message) Rule
```

`Request.CacheBoundaries()` is `r.options.cacheBoundaries(r.messages)` — the request already stores its draft in `options`, so the frozen request and the in-construction draft feed the identical function. This is AI-06's `decision.md` § 7.2 principle (*one rule set, two callers*) applied to a walk instead of a rule set, and it is why `S-ACB-032` — reported count agrees with enforced count — is true by construction rather than by discipline.

The cap rule:

```go
if len(d.cacheBoundaries(messages)) > MaxCacheBoundaries {
	return Invalid(ErrOutOfRange, At("cacheBoundaries"))
}
```

`"cacheBoundaries"` is 15 bytes of ASCII letters, so `structuralName` renders it unfiltered. **No new sentinel**: `validation.go:81` already cites `V-REQ-24` in `ErrOutOfRange`'s own GoDoc, so AI-04's append rule does not fire and `ruleClasses` and its two registry mirrors are untouched.

### 3.3 The rule order, restated `[assumes AI-10 — `explore.md` § 8 row 3]`

| # | Rule | Class | Position | Owner |
| --- | --- | --- | --- | --- |
| 1 | model identity non-empty | `ErrEmpty` | `model` | AI-10.1 landed |
| 2 | at least one message | `ErrEmpty` | `messages` | AI-10.1 landed |
| 3 | each message constructed | `ErrEmpty` | `messages[i]` | AI-10.1 landed |
| 4 | each message's content, via AI-06's validator | AI-06's | `messages[i].content[j]…` | AI-10.1 landed |
| 5 | applied system instruction constructed | `ErrEmpty` | `system` | AI-10.2 landed |
| 6 | role versus content kind | `ErrMisplaced` | `messages[i].content[j]` | AI-10.3 |
| 7 | tool-call identities unique | `ErrDuplicate` | `messages[i].content[j]` | AI-10.3 |
| 8 | tool results correlate to a call | `ErrUnresolvedReference` | `messages[i].content[j]` | AI-10.3 |
| 9 | tool choice against tool set | AI-08.3's three | AI-08.3's three | AI-10.3 |
| **10** | **marker count within the cap** | **`ErrOutOfRange`** | **`cacheBoundaries`** | **AI-11.2** |
| 11 | generation option bounds | `ErrOutOfRange` / `ErrEmpty` | § 8.1's positions | AI-10.1 landed, renumbered here |

`S-ACB-024` pins the consequence: a request that both breaks an earlier rule and exceeds the cap reports the earlier rule, because `FirstFailure` short-circuits. That is also why inserting this row cannot turn any landed AI-10 test red.

### 3.4 Rendering `[planned]`

| Type | Today | After |
| --- | --- | --- |
| `Segment` unmarked | `"segment"` | `"segment"` — **unchanged**, so `system_instruction_test.go:288` stays green |
| `Segment` marked | — | `"segment(cache boundary)"` |
| `Segment` zero | `"segment(unset)"` | unchanged |
| `Request` with no markers | `"request(model, 2 messages, …)"` | unchanged |
| `Request` with markers | — | `…, 3 cache boundaries)"` |

`system_instruction.go:148` already authorises this: *"AI-11 may extend this to name a cache marker, because a marker is structural rather than payload."* A count and a boolean are facts about the request's shape, which is the same admission test `Request.String()` already applies to its message count.

`Tool` and `Message` define no `String()` today, so `R-ACB-010` binds `Segment` and `Request` only. That is not an omission of this milestone — `R-AMR-017` names exactly the request, the system-instruction and the segment types — and adding one to `Tool` or `Message` would be AI-08's and AI-05's respectively.

## 4. Data flow

```
    caller
      │  NewSegment / NewTool / NewMessage
      ▼
   carrier ──── MarkCacheBoundary() ───► marked copy   (original unchanged)
      │                                      │
      └──────────────┬───────────────────────┘
                     ▼
        NewSystemInstruction / NewToolSet / (messages slice)
                     ▼
                 NewRequest
                     │
                     ├─ rules 1…9   (structure, AI-10)
                     ├─ rule 10 ────► d.cacheBoundaries(messages) ──► len > 4 ? ErrOutOfRange @ cacheBoundaries
                     └─ rule 11  (option bounds)
                     ▼
                  Request  ── CacheBoundaries() ──► d.cacheBoundaries(messages)
                     │                                        │
                     │                              tools → system → messages
                     ▼                                        ▼
                 adapter  ──────── honours ────────────────────┘
                     └──────────── or ignores wholesale (still conformant)
```

The two arrows into `d.cacheBoundaries` are the same function. That is the diagram's only load-bearing detail.

## 5. File changes

| File | Action | Description |
| --- | --- | --- |
| `backend/agent/src/ai/cache_boundary.go` | Create | `MaxCacheBoundaries`, `CacheRegion` + vocabulary, `CacheBoundary`, the shared walk, the cap rule, `Request.CacheBoundaries` |
| `backend/agent/src/ai/system_instruction.go` | Modify | `cacheBoundary` field, `MarkCacheBoundary`, `IsCacheBoundary`, `Segment.String()` extension |
| `backend/agent/src/ai/tool.go` | Modify | `cacheBoundary` field, `MarkCacheBoundary`, `IsCacheBoundary` |
| `backend/agent/src/ai/message.go` | Modify | `cacheBoundary` field, `MarkCacheBoundary`, `IsCacheBoundary` |
| `backend/agent/src/ai/request.go` | Modify | cap rule inserted at position 10; `String()` names the boundary count |
| `backend/agent/src/ai/cache_boundary_test.go` | Create | AI-11.1 and AI-11.2, package `ai_test` |
| `backend/agent/src/agenttest/cache_boundary_test.go` | Create | AI-11.3 — the marker-blind translator, its marker-aware control, and the two usage pins |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **Untouched** | Read and cited. `V-REQ-23` … `V-REQ-25` already exist; this milestone appends no term. |

## 6. AI-11.3 — the marker-blind translator and its control `[planned]`

`R-ACB-008` cannot be proven by an adapter, because no adapter exists until AI-24. It is proven by a **pair** of translators written in `agenttest`, which is the package that already proves cross-package readability for AI-06.

- **The blind translator** reads every region through the exported surface — model, system segments' text, tool declarations' name/description/schema, messages' role and content, tool choice, every generation option — and appends each to a deterministic string. It never calls `IsCacheBoundary` or `CacheBoundaries`. `S-ACB-033` asserts `render(marked) == render(unmarked)`.
- **The aware control** is the same walk plus one marker read. `S-ACB-034` asserts `render(marked) != render(unmarked)`.

The control is the whole point, and its absence is the classic way this test lies: a blind translator that rendered nothing would satisfy `S-ACB-033` perfectly. This is the AI-06 `testdata/handrolled` + `testdata/constructed` lesson — *the control is what stops "does not compile" from passing for a misspelled import* — applied to a rendering instead of a build.

`S-ACB-035` closes the second half: the blind rendering of the marked request is compared field-count-wise against the unmarked twin's, so "identical" cannot be achieved by dropping a region from both.

## 7. Testing strategy

| Layer | What to test | Approach |
| --- | --- | --- |
| Unit, package `ai_test` | Marker placement, copy semantics, zero-value no-op, readback, validity independence, cap, cascade order, rendering | Table-driven over the three carriers where the behaviour is uniform (`t.Run` per carrier); explicit success and failure cases per `errors.Is` + `errors.As` |
| Unit, package `ai` (internal) | None planned | Every property here is observable from outside. If one is not, that is a design defect to record, not an internal test to write. |
| Cross-package, `agenttest` | The advisory contract and its control; the two usage pins | Two renderers over the exported surface only |
| Guard | The cap is enforced with no I/O in the closure | Extend the landed `import_boundary_test.go` pattern rather than writing a second guard |

Conventions, inherited and re-stated because they are where this package's lint traps live:

- **Behavioural tests in `ai_test`**, external package — the consumer a marker exists for is an adapter in a vendor package.
- **stdlib only.** No assertion library. `slices.Equal`, `bytes.Equal`, `errors.Is`, `errors.As`, hand-written table loops.
- **Naming** `Test<Subject>_<Behavior>_<Expectation>`, banner comment citing the leaf ID.
- **Red before green, per item, recorded.** Go's compile-time typing means a red step needs the declaration to exist: land the narrowest thing that **compiles and fails**, record the failure, then implement. A compile error is the state before red, not red.
- **Lint traps already paid for twice this wave**: a file banner comment must be separated from `package ai` by a blank line (revive `package-comments`); literal bidirectional control characters in a test table trip `ST1018`; `fmt.Sprintf("%s", x)` where `String()` exists trips `S1025`. `make lint` runs before every commit — AI-06's commit `6a88fb1` landed with lint failing and had to be repaired in `17f952c`.

## 8. Threat Matrix

**N/A** — no routing, shell commands, subprocesses, VCS/PR automation, executable-file classification, or process integration. This milestone adds one boolean field to three value types, one constant, one closed vocabulary, one pure walk and one validation rule, inside a package whose dependency closure is asserted to contain no network and no filesystem package.

## 9. Migration / rollout

No migration required. Every change is additive and no existing signature moves, so every caller of the AI-10 surface compiles unchanged. The rollback boundary is the leaf commit.

## 10. Open questions — resolved at apply time

Each was an assumption about work landing concurrently, not an undecided design point. The full register with files and symbols is `explore.md` § 8, and `tasks.md`'s "re-verification register — resolved" section carries the complete detail; these are the three that would have changed this document, plus one finding this document did not anticipate at all.

- [x] **`Request.Tools()` is the tool accessor** (`explore.md` § 8 row 1). **Held exactly as assumed**: `func (r Request) Tools() (ToolSet, bool)`, `request.go:357`. § 3.2's walk needed no adaptation.
- [x] **`Request.Equal` is exported and compares regions by readback** (`explore.md` § 8 rows 4–5). **Held for export** (`request.go:426`) — **but the "if it ignores markers, that is a defect in AI-10.6" framing above was too narrow.** `Request.Equal` composes `SystemInstruction.Equal` (picks up a new `Segment` field for free, because `Segment` stays comparable), `Message.Equal` and `toolsEqual` (both hand-rolled, explicit-field comparators that name exactly the fields they compare). Adding `cacheBoundary` to `Tool`/`Message` did not make either see it — not an AI-10.6 defect, since the field did not exist when those comparators were written, but a real gap this milestone had to close in `Message.Equal` and `toolsEqual` (both already-AI-11-owned files). Proven with a genuine RED (`TestRequest_MarkersDifferByOneMarker_AreNotEqualButIdenticalMarkersAreEqual/tool`) before the fix. **A second, independent gap in the same area, found by the orchestrator mid-apply and not anticipated by this document at all**: AI-10.5's whole-request round-trip pin (`agenttest/request_test.go`) never calls `Request.Equal` — it drives hand-written `rebuildFromReadback`/`requireRequestsEqual` helpers that dropped markers silently until AI-11.1 extended them (appended as AI-11.1 item 6).
- [x] **The final structural rule number** (`explore.md` § 8 row 3). **Held, and the cap rule lands exactly where predicted**: the last structural slot, as the 10th positional argument to `NewRequest`'s `FirstFailure(...)` call, immediately before the (now 11th) generation-option bounds rule. § 3.3's table is unchanged in substance from what this document already recorded.
