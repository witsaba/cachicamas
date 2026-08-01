# Design — request extension points

> **Change**: `cachicamas-ai-request-extension-points`
> **Milestone**: AI-12 · **Nodes**: AI-12.1 … AI-12.4, all `[leaf]`
> **Phase**: design · **Date**: 2026-08-01
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-request-extension-points/spec.md`
> **Consumes**: AI-10 `design.md` §§ 4, 8, 11, 12 · `backend/agent/src/ai/{request.go, system_instruction.go, validation.go}`
> **Every disposition below is `[provisional]`** and may be changed only by recording the reason **in this file, before writing the test**. Absent a recorded reason, implement what is written.
> **Re-verified 2026-08-01** against finished Wave 1 head `1c4171e` (AI-04 … AI-10 complete and green). § 13 is now a **resolved** register: six AI-10 rows read from the landed source, two AI-11 rows still open with both branches decided. Every disposition survived; §§ 2.2, 3, 5, 7, 8 and 10 carry the corrections the reading produced. `explore.md` § 6 holds the evidence.

---

## 1. Technical approach — one draft, one rule slice, two seeds

AI-10 already put every optional region into an unexported accumulator, `requestDraft`, and made `RequestOption` a total, last-wins mutation of it. AI-12's whole approach is to finish that idea:

```
                 model, messages, opts…            ┐
NewRequest ─────────────────────────────────────►  │
                                                   ├──►  requestDraft  ──►  draft.rules()  ──►  FirstFailure  ──►  Request
                 r.options (frozen) + opts…        │          (every region)     (one slice, one order)
r.With ─────────────────────────────────────────►  ┘
```

Two doors, one draft type, one rule slice, one order. That is what makes `R-REX-005` — "derive-time validation is construction's validation" — a **structural fact** rather than a discipline a reviewer must re-check on every future rule. It is AI-06's "one rule set, two callers" at request scope, and it is the reason § 4 widens `RequestOption` rather than giving the derive path a vocabulary of its own.

The three deliverables then fall out as small additions rather than as mechanisms:

| Deliverable | What it actually is |
| --- | --- |
| Copy-on-write rebuild (`V-REQ-29`) | The second seed. `Request` is a value and the draft is copied, so "copy-on-write" needs no bookkeeping — it is Go's assignment plus the clones `NewRequest` already performs |
| Per-request override (`V-REQ-27`) | The landed last-wins rule, reached through the second seed. **No new symbol at all** |
| Escape hatch (`V-REQ-28`) | One more optional region on the draft, one more option constructor, two accessors, one rule |

## 2. The Go shapes

### 2.1 Added to `request.go`

```go
// The two options that make the required regions reachable (§ 4).
func WithModel(model string) RequestOption
func WithMessages(messages ...Message) RequestOption

// The rebuild (V-REQ-29).
func (r Request) With(opts ...RequestOption) (Request, error)
```

`With` is the signature AI-10 `design.md` § 12.2 recorded. Its body is three steps: copy `r`'s draft, apply the options in order, validate and freeze — the same three `NewRequest` performs after seeding from its parameters.

### 2.2 Extracted in `request.go` — the refactor AI-12.1 must land first

`NewRequest` today passes an argument list of `Rule` closures directly to `FirstFailure`. AI-12.1 moves that list to a method on the draft:

```go
func (d requestDraft) rules() []Rule
```

`NewRequest` and `With` both call `FirstFailure(d.rules()...)`. Nothing else changes about the rules; the order is preserved exactly, and the extraction is behaviour-preserving by construction.

`FirstFailure(rules ...Rule) error` (`validation.go:381`) is variadic, so `FirstFailure(d.rules()...)` is a drop-in for the landed argument list. **Read at `1c4171e`**: that list is a *variadic argument list*, not a `[]Rule` literal — nine inline `func() *Violation` literals plus one method call, `draft.boundsRule()`. The extraction is still one mechanical move.

**Consequence, now measured rather than assumed** (`explore.md` § 6.1.1). Seven of the ten landed rules read a `NewRequest` **parameter** and must be re-pointed at draft state: rule 1 reads `model`; rules 2, 3, 4 and 6 iterate `messages`; rules 7 and 8 pass `messages` into a free function. Only rules 5, 9 and 10 already read `draft`.

The three cross-region free functions — `duplicateToolCallRule([]Message)`, `unresolvedToolResultRule([]Message)`, `anyToolCallHasID([]Message, string)` (`request.go:71`, `:100`, `:117`) — take their messages as a **parameter** and need **no edit**. Only their two call sites move to `d.messages`. The whole diff is confined to `NewRequest`'s body, and AI-10 has landed, so nothing is jointly owned any more.

### 2.2.1 Extract the freeze too — the symmetry that closes a landed trap

`Request` stores the system region **twice**: `Request.system` / `Request.hasSystem` at the top level (`request.go:17–18`) *and* `requestDraft.system` / `hasSystem` inside `Request.options` (`request.go:299–300`). `NewRequest` sets both from one draft on the way out, and `Request.SystemInstruction()` reads the **top-level** pair.

`With`'s seed is `r.options`. A `With` that applies options and freezes without re-deriving the top-level pair would **silently revert the system region** — a bug no test in the current suite would catch, because no current test derives a request.

The fix is symmetry rather than vigilance. Extract the freeze alongside the rules:

```go
func (d requestDraft) freeze() Request
```

Both doors then read identically:

```go
if err := FirstFailure(d.rules()...); err != nil {
    return Request{}, err
}
return d.freeze(), nil
```

One draft, **one rule slice, one freeze**, two seeds. This is an addition to AI-12.1's ordered work discovered by re-verification (`explore.md` § 6.1.6), and it is the reason `S-REX-009`'s system-instruction leg must be asserted through the derive path and not only through construction.

### 2.3 New file `request_extension.go`

```go
// ProviderExtension is one namespaced, opaque, provider-specific value (V-REQ-28).
//
// Sealed by unexported fields and by the absence of an exported constructor: a
// consumer reads one, never builds one. It is produced only by a request.
type ProviderExtension struct {
    namespace string
    value     []byte
}

func (e ProviderExtension) Namespace() string          // caller data — never rendered
func (e ProviderExtension) Value() []byte              // fresh copy per call
func (e ProviderExtension) IsZero() bool
func (e ProviderExtension) String() string             // "extension", never the namespace
func (e ProviderExtension) GoString() string

// WithProviderExtension attaches a provider-specific value under a namespace.
// Total, never validating, last-wins per namespace — the RequestOption contract.
func WithProviderExtension(namespace string, value []byte) RequestOption

func (r Request) ProviderExtension(namespace string) (ProviderExtension, bool)
func (r Request) ProviderExtensions() []ProviderExtension   // ordered, fresh slice per call
```

`requestDraft` gains one field: `extensions []ProviderExtension`. **A slice, never a map** — `validation.go`'s `ruleClasses` states the rule for the whole package (*"nothing in this package may let an unordered iteration decide anything, and a registry is where that temptation is strongest"*), and this region is the next-strongest temptation. `ProviderExtension(namespace)` is a linear scan; the region holds a handful of entries and a map here would buy nothing but nondeterminism.

## 3. The widened rule order

AI-10 `design.md` § 4 owns the table. AI-12 appends **one row at the end** and moves none.

**Read from the landed `FirstFailure` argument list at `1c4171e`** (`request.go:180–234`) — this corrects the order previously recorded here:

| # | Rule | Class · position | Reads | Owner |
| --- | --- | --- | --- | --- |
| 1 | model identity present, whitespace folded | `ErrEmpty` at `model` | param | AI-10.1 |
| 2 | at least one message | `ErrEmpty` at `messages` | param | AI-10.1 |
| 3 | every message constructed (`ID().IsZero()`) | `ErrEmpty` at `messages[i]` | param | AI-10.1 |
| 4 | content valid at request depth (`validateContent`) | AI-06's classes | param | AI-10.1 |
| 5 | applied system instruction is constructed | `ErrEmpty` at `system` | draft | AI-10.2 |
| 6 | role/kind table (`roleAllowsKind`) | `ErrMisplaced` at `messages[i].content[j]` | param | AI-10.3 |
| 7 | **duplicate tool-call identities** | `ErrDuplicate` | param | AI-10.3 |
| 8 | **orphan tool results** | `ErrUnresolvedReference` | param | AI-10.3 |
| 9 | tool choice cross-validated against the tool set | AI-08.3's three rules | draft | AI-10.3 |
| 10 | generation-option bounds (`boundsRule`) | `ErrOutOfRange`, `ErrEmpty` | draft | AI-10.1 |
| *(11)* | *breakpoint cap — **does not exist at `1c4171e`*** | `ErrOutOfRange` | — | AI-11, unlanded |
| **11 or 12** | **each extension has a namespace, then a non-empty value, in ordinal order** | **`ErrEmpty` at `extensions[i].namespace` / `extensions[i].value`** | draft | **AI-12.3** |

**Correction 1 — rows 7 and 8 were recorded in the wrong order.** This table previously listed "orphan tool results · duplicate call identities". The landed order is the reverse, and deliberately: `duplicateToolCallRule`'s GoDoc (`request.go:62–63`) states that identity uniqueness is a *precondition* of `unresolvedToolResultRule`, because "a rule that resolves by a non-unique key is not a rule — and that is why this rule is checked first". Anything in AI-12.2 that asserts *position* rather than *class* must anchor on this order.

**Correction 2 — there is no row 11 today.** AI-11's cap is not in the list, so AI-12's extension rule appends as row **11** at present and becomes row **12** only if AI-11 lands its cap first. Both are fine: AI-12's sole requirement is that its own row is **appended**, so no landed first-failure outcome changes. `R-REX-005`'s scenarios assert class-and-position equivalence between the two doors, never an absolute ordinal, so neither ordering needs a spec edit.

## 4. Disposition 1 `[provisional]` — `WithModel` and `WithMessages`

**Decision.** The required regions become reachable through `RequestOption`. `NewRequest` keeps its `(model, messages, opts…)` signature and seeds the draft from its parameters **before** applying options.

**Alternatives rejected** — `explore.md` § 4 carries the full table. In one line each: `With(model, messages, opts…)` forces every hook to restate the whole transcript to change one option; a second option type duplicates the generation options and turns `R-REX-005` back into a claim about two code paths.

**The hazard, and the pin.** `NewRequest("a", msgs, WithModel("b"))` yields model `"b"`. This is last-wins applied uniformly, not a new rule — a reader who knows `RequestOption`'s documented last-wins predicts it correctly. It is pinned by a test (`S-REX-007`'s sibling in `tasks.md`) in AI-10.3's `S-AMR-050` style: **a deliberate disposition that is not asserted is indistinguishable from an oversight**, and the next reader would close it by reflex.

**Why not reject it at runtime.** Rejecting would need either a new rule class ("you used the wrong door" is not a caller-contract failure about a *value*) or two option types. Both cost more than the documentation the hazard costs.

## 5. Disposition 2 `[provisional]` — the payload is `[]byte` behind a sealed type

**Decision.** A provider extension carries `[]byte`. The container is a sealed value type with no exported constructor.

`explore.md` § 5 carries the comparison. The two clauses that decide it:

- **`any` is forbidden by doc 0001 § 8's contract checklist** — *"No vendor wire type crosses the Layer 1 method boundary."* An `any` is precisely a hole through which one does, and the hole is invisible at the call site.
- **An encoding-named alias is forbidden twice** — Layer 1 carries zero requires until AI-24, and naming an encoding is an invitation to parse the thing the register calls opaque.

`[]byte` is also the shape AI-07 chose for the reasoning round-trip token, for the identical reason. Reusing it gives the package **one** opacity discipline instead of two.

**"Typed" is about the container, not the payload.** `ProviderExtension` is a real type with a namespace and rules, so a consumer cannot confuse it with an unrelated `[]byte` and cannot forge one. The bytes inside stay uninterpreted.

**Copy-out per call**, matching `Message.Content`, `ToolSet.Tools` and `Request.StopSequences`: a consumer that rewrites what it received must not be able to rewrite the request, and two consumers must not be able to observe each other.

### 5.1 The AI-10.4 constraint this disposition must survive — confirmed explicitly

AI-10.4 landed `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` (`import_boundary_test.go:278`). It runs `go list -deps` — deliberately **without** `-test`, so it is a claim about the production closure only — over `github.com/cachicamas/backend/agent/src/ai`, and fails if that closure contains `net`, `net/http`, `os` or `io/fs`.

The closure measured at `1c4171e` is: `errors`, `sync`, `sync/atomic`, `io`, `iter`, `unicode`, `unicode/utf8`, `bytes`, `cmp`, `slices`, `strconv`, `strings`, `math/bits`, plus runtime internals. **No `fmt`. No `encoding/json`.**

`encoding/json` is banned from non-test files in this package as a *consequence* of that guard rather than by a rule naming it: `encoding/json` imports `fmt`, which imports `os`, so importing it turns the guard red. `json_syntax.go:5–6` records the measurement, and it is why the package hand-wrote `isWellFormedJSON` rather than calling `encoding/json.Valid`.

**This design assumes no serialization dependency, and that is now verified rather than intended.** Stated explicitly, because the constraint is the reason two things in this file are shaped as they are:

1. **The payload is bytes precisely so that no encoder is ever needed to carry it.** `explore.md` § 5 rejected a `json.RawMessage`-shaped alias on two grounds — zero requires, and opacity. The dependency-closure guard promotes the first from a preference to a hard constraint: a `ProviderExtension` that named an encoding could not be carried without importing one, and importing one would turn a landed AI-10.4 guard red inside AI-12's own commit.
2. **Every rendering in `request_extension.go` must be built with `strings.Builder` and `strconv`, never `fmt.Sprintf`.** `Request.String` (`request.go:584`) is written that way for exactly this reason, and `ProviderExtension.String` / `GoString` follow it. A single `fmt` call in a non-test file of this package fails AI-10.4's guard.

Everything `ProviderExtension` needs — `bytes.Equal`, `slices.Clone`, `strings.Builder`, `strings.TrimSpace`, `strconv.Itoa` — is **already in the closure**. AI-12 adds no import to the package's production surface, `go.mod` stays at zero requires, and the guard stays green without being weakened.

## 6. Disposition 3 `[provisional]` — last-wins per namespace, in place; an empty value is rejected

**Last-wins, not `ErrDuplicate`.** A namespace is what "the same option" means for this region. `RequestOption`'s GoDoc already says last-wins exists so AI-12 can express an override as re-application; making the escape hatch the one region a hook *cannot* revise would contradict `R-REX-002`.

**Replacement keeps the first ordinal.** The alternative — moving a replaced namespace to the end — makes read-back order depend on how many times a value was revised, which is a determinism surprise (`R-REX-009`) with nothing to recommend it. Read-back order is therefore **first-application order**, stated once and testable in one scenario (`S-REX-030`).

**An empty value is rejected** with `ErrEmpty` at `extensions[i].value`. This is AI-10.2's argument for `NewSystemInstruction` rejecting zero segments, one region over: accepting it would give the package **two spellings of "this provider has nothing extra"** — namespace absent, and namespace present with no bytes — and every later reader would have to know both. One spelling, enforced by a rule.

**A whitespace-only namespace is folded into emptiness** (`NewSegment`'s and `NewRequest`'s treatment of the model identity), because a namespace is *text* that names a provider. **A whitespace-only value is legal**, because a value is *bytes* and the package does not know they are whitespace. The asymmetry is deliberate and is the clearest statement in this file of where interpretation stops.

**No other rule.** No format, no prefix convention, no length bound, no character class, and no registry of recognised namespaces — a registry would make Layer 1 know every provider, which is the coupling the hatch exists to remove.

## 7. Disposition 4 `[provisional]` — extensions participate in equality, structurally

doc 0002's AI-12.3 item 3 says the pass-through is "inert in validation and equality", and states the check as *"two requests differing only in a third provider's namespace **validate** identically"*.

**Read "inert" as *never interpreted*, not as *excluded*.** An equality that ignored extensions would let a rebuild **silently drop every extension** and still satisfy the round-trip property `R-AMR-015` pins — the exact class of bug this milestone exists to prevent.

So: equality compares namespaces by string equality and values by byte equality, in read-back order, with no per-namespace branch anywhere. Inertness is preserved where doc 0002 asserts it — **validation** — and `S-REX-035`/`S-REX-036` assert exactly that clause, verbatim.

**Dependency on AI-10.6 — resolved.** `func (r Request) Equal(other Request) bool` **is exported** (`request.go:426`), the recorded default. AI-12.3 therefore **extends it**; the alternative branch — compare region-by-region inside `ai_test` and export no equality of AI-12's own — is **closed, not taken**, and is retained here only so a later reader can see it was considered rather than missed. A second equality remains forbidden either way.

### 7.1 Exactly what AI-12.3 adds to `Request.Equal`

`Equal` returns `false` at the first difference across nine regions, in this order (`request.go:426–490`): `model` (the unexported field), messages via `slices.EqualFunc(…, Message.Equal)`, then system, tools and tool choice each as *presence flag first, then value*, then the four generation options, then `return true`. Tools are compared by the **free function** `toolsEqual` (`request.go:499`) rather than by a new method on a type AI-08 owns.

AI-12.3 appends one block immediately before `return true`:

```go
if !slices.EqualFunc(r.ProviderExtensions(), other.ProviderExtensions(), providerExtensionsEqual) {
    return false
}
```

with `providerExtensionsEqual(a, b ProviderExtension) bool` a **free function** on `toolsEqual`'s precedent — `a.Namespace() == b.Namespace() && bytes.Equal(a.Value(), b.Value())` — and **not** an exported `ProviderExtension.Equal`. Two properties make this one line rather than a flag branch: the region carries **no presence flag** (it is a slice, so absent and empty are one fact), and it is already in first-application order, so `slices.EqualFunc` is the whole comparison.

It reads through `ProviderExtensions()` and not `r.options.extensions` because `Equal`'s own GoDoc requires it: every comparison goes through the exported surface "so this method proves the documented equality is reachable from outside, not only from here".

### 7.2 The larger hole re-verification found, and the obligation it adds

The previous version of this section justified extending `Equal` by claiming that otherwise a dropped extension "would still pass AI-10.5's round-trip pin". **That claim is false, and the truth is worse** (`explore.md` § 6.1.4).

`backend/agent/src/agenttest/request_test.go` **never calls `Request.Equal`**. `TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest` (`:137`) rebuilds through its own `rebuildFromReadback` (`:235`) and compares through its own `requireRequestsEqual` (`:329`) — two hand-written walks over a fixed nine-region list. The moment a tenth region exists, the pin has a **two-sided** blind spot: the rebuild does not re-apply what it does not know about, and the comparison does not notice what the rebuild dropped. The pin then goes green on a request that lost its extensions.

`agenttest` guards its *kind* set (`TestPartKindReaders_EveryRegisteredKind_HasAReader`, `:111`) but has no region-level equivalent, which is precisely why a new region slips through unnoticed.

**Obligation, appended to AI-12.3** — extending `Request.Equal` is necessary and **not sufficient**. AI-12.3 must also:

1. add the extension region to `rebuildFromReadback`, re-applying it through `WithProviderExtension` in read-back order, exactly as the tool set and each generation option are re-applied;
2. add the extension region to `requireRequestsEqual`, compared namespace-by-namespace with `bytes.Equal` on values, exactly as tool schemas are compared.

This is an **appended** obligation. It adds `backend/agent/src/agenttest/request_test.go` to § 10's file table and one item to AI-12.3's list; it substitutes for nothing and prunes nothing.

## 8. Disposition 5 `[provisional]` — totality is proven by enumeration

`R-REX-002` requires the region set to be walked, not sampled. The implementation is a table in the test:

```go
// AI-12.1 — every region is reachable by the rebuild path.
regions := []struct {
    name    string
    derive  func(ai.Request) (ai.Request, error)  // applies a change through With
    changed func(ai.Request) bool                  // observes it on the derived request
}{ /* model, system, messages, tools, toolChoice, each option, markers, extensions */ }
```

A region added by a later milestone without a rebuild path does not silently pass: the table is the checklist, and the test's failure message names the missing region. This is the cheap half of AI-06.4's guard idea without adding a `[guard]` node the charter does not carry (`explore.md` § 9).

### 8.1 The complete region list the rebuild must be total over

Enumerated from the landed `Request` at `1c4171e` — the struct's fields (`request.go:14–20`), `requestDraft`'s fields (`request.go:286–307`) and every exported accessor. **Nine regions exist today; two are added by the two milestones this one sits beside.** The § 8 table has one row per line of this list — *not* one row for "generation options", because a table that groups four options into one row is a table a fifth option can be added to without failing.

| # | Region | Accessor | `derive` — how the rebuild reaches it | Reachable today? |
| --- | --- | --- | --- | --- |
| 1 | model identity | `Model() string` | `WithModel` | **No — `NewRequest` parameter.** AI-12.1 adds it |
| 2 | messages | `Messages() []Message` | `WithMessages` | **No — `NewRequest` parameter.** AI-12.1 adds it |
| 3 | system instruction | `SystemInstruction() (SystemInstruction, bool)` | `WithSystemInstruction` (`system_instruction.go:121`) | Yes |
| 4 | tool set | `Tools() (ToolSet, bool)` | `WithTools` (`request.go:334`) | Yes |
| 5 | tool choice | `ToolChoice() (ToolChoice, bool)` | `WithToolChoice` (`request.go:344`) | Yes |
| 6 | max output tokens | `MaxOutputTokens() (int, bool)` | `WithMaxOutputTokens` | Yes |
| 7 | temperature | `Temperature() (float64, bool)` | `WithTemperature` | Yes |
| 8 | top-p | `TopP() (float64, bool)` | `WithTopP` | Yes |
| 9 | stop sequences | `StopSequences() ([]string, bool)` | `WithStopSequences` | Yes |
| 10 | cache-boundary markers | AI-11's | branch-dependent — § 13 row 7 | **Unlanded** |
| 11 | provider extensions | `ProviderExtension(ns)` / `ProviderExtensions()` | `WithProviderExtension` | AI-12.3 adds it |

Seven of the nine landed regions are already option-reachable; **exactly two are not**, and they are exactly the two `NewRequest` takes as parameters. That is the whole of § 4's case, now read rather than predicted.

Region 3 carries the § 2.2.1 trap: it is stored twice on `Request`, so its totality row must observe it through `SystemInstruction()` on the **derived** request, which is the accessor that reads the top-level pair a careless `With` would fail to set.

### 8.2 Markers

The row is written against whatever AI-11 lands, and both branches are decided in `explore.md` § 6.2 row 7. If markers ride on `Segment`, `Tool` and `Message` — AI-10 `design.md` § 12.1's recorded expectation, and the direction `Segment.String()`'s GoDoc (`system_instruction.go:158`) hints at — the row's `derive` is `WithSystemInstruction`/`WithTools`/`WithMessages`, its `changed` reads the marker off the derived value, and AI-12 adds no option. If AI-11 models them as a request-level region, the row's `derive` is AI-11's own constructor wrapped in one new `RequestOption` that AI-12.1 appends. **Both are satisfying answers to `R-REX-002`; neither is invented here.** No marker surface exists at `1c4171e`: `Segment` is `struct{ text string }`, and `Tool` and `Message` carry no marker field.

## 9. Data flow

```
caller ──► NewRequest(model, messages, opts…)
                 │  seed draft from parameters
                 │  apply opts in order (last-wins)
                 ▼
            requestDraft ──► rules() ──► FirstFailure ──► Request (frozen, immutable)
                 ▲                                            │
                 │  seed draft from r.options                 │
hook ──► r.With(opts…) ◄─────────────────────────────────────┘
                 │  apply opts in order (last-wins)
                 ▼
            requestDraft ──► rules() ──► FirstFailure ──► Request'  (r is untouched)

adapter A ──► r.ProviderExtension("A")  ──► value bytes, byte-exact
adapter B ──► r.ProviderExtension("B")  ──► absent — A's value is unreachable
```

## 10. File changes

| File | Action | Description |
| --- | --- | --- |
| `backend/agent/src/ai/request.go` | Modify | `WithModel`, `WithMessages`, `Request.With`; `requestDraft` gains `model`, `messages`, `extensions`; the rule list extracted to `draft.rules()` **and the freeze to `draft.freeze()`** (§ 2.2.1); `Request.Equal` gains the extension block (§ 7.1); `appliedNames` gains the extension region |
| `backend/agent/src/ai/request_extension.go` | Create | `ProviderExtension`, `WithProviderExtension`, the two request accessors, `providerExtensionsEqual`, the region's rules, `String`/`GoString` — **`strings.Builder`/`strconv` only, never `fmt`** (§ 5.1) |
| `backend/agent/src/ai/request_test.go` | Modify | AI-12.1 and AI-12.2 items, in `ai_test` |
| `backend/agent/src/ai/request_extension_test.go` | Create | AI-12.3 and AI-12.4 items, the two fake translators, the determinism loops |
| `backend/agent/src/agenttest/request_test.go` | **Modify** *(added by re-verification)* | AI-10.5's round-trip pin walks a fixed nine-region list through its own `rebuildFromReadback` and `requireRequestsEqual` and never calls `Request.Equal`. Both gain the extension region, or the pin silently narrows (§ 7.2). **AI-12.3 owns this edit** |
| `backend/agent/src/ai/validation.go` and both mirrors | **None** | No rule class appended — `explore.md` § 8 walks the fit |
| `openspec/specs/ai-contract-vocabulary/spec.md` | **None** | `V-REQ-27` … `V-REQ-29` already exist and already name AI-12 |
| `backend/agent/go.mod`, `go.work` | **None** | Zero requires preserved |

## 11. Testing strategy

| Layer | What | How |
| --- | --- | --- |
| Behavioral | Every `R-REX-*` scenario | External package `ai_test`, for AI-06's reason: the consumer this contract exists for is an adapter in a vendor package |
| Structural | Totality over the region set | The § 8 table, whose rows are the checklist |
| Adapter acceptance | "survives to its adapter, invisible to another" | Two **fake** translator funcs in `ai_test`, each namespace-scoped. No production adapter, no network, no vendor package (`explore.md` § 5.2) |
| Determinism | Read-back order and content | Repeat-read loops (100 iterations) plus a same-inputs-two-requests comparison; `-race` is already on via `make test` |
| Redaction | `R-REX-010` | Four fmt verbs × the two secret-bearing fields, one table test — `R-AMR-017`'s shape |

Conventions, unchanged from the wave: `Test<Subject>_<Behavior>_<Expectation>`; a banner comment citing the leaf ID; stdlib only, so comparisons are `slices.Equal`, `bytes.Equal`, `errors.Is`, `errors.As` and hand-written table loops; red before green per item, recorded. Go's compile-time typing means a red step needs the declaration to exist — land the narrowest thing that **compiles and fails**, record it, then implement. A compile error is the state before red, not red.

Lint traps already paid for twice this wave and re-paid here: a file banner comment must be separated from `package ai` by a blank line (revive `package-comments`); literal bidirectional control characters in a test table trip `ST1018`, so such fixtures are built from escapes; `fmt.Sprintf("%s", x)` where `String()` exists trips `S1025`. **`make lint` runs before every commit** — AI-06.2 landed with it failing, and that is the recorded reason this line is here.

## 12. Threat matrix

**N/A** — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. This change adds pure in-memory value types to a package whose no-I/O property is asserted mechanically by AI-10.4's dependency-closure guard, which this change does not weaken.

## 13. Re-verification register — **resolved 2026-08-01**

This was an eight-row checklist of assumptions about surfaces that had not landed. **AI-10 has now landed** and this worktree is rebased onto Wave 1 head `1c4171e`. Rows 1–6 are **resolved facts**, each with the signature that was read. Rows 7–8 depend on **AI-11**, which is being implemented right now in `../ai-11` and cannot be read from here; they stay **unresolved**, with both branches and the exact edit each implies, so the apply agent executes a decision rather than repeating the analysis.

The evidence, at length, is `explore.md` § 6.

### 13.1 Rows 1–6 — resolved against the landed AI-10 surface

| # | Check | Verdict | The signature that was read | What changed in the plan |
| --- | --- | --- | --- | --- |
| 1 | `tools` / `toolChoice` reach the request as `RequestOption`s | **Matches** | `func WithTools(tools ToolSet) RequestOption` (`request.go:334`); `func WithToolChoice(choice ToolChoice) RequestOption` (`request.go:344`); `requestDraft` holds `tools ToolSet / hasTools bool`, `toolChoice ToolChoice / hasToolChoice bool` (`request.go:302–306`) | **Nothing.** Both regions are option-reachable for free; AI-12 adds no constructor for them. Rows 4 and 5 of the § 8.1 region table are satisfied without work |
| 2 | AI-10.3's cross-region rules read draft state | **Differs — branch taken** | They close over the **`messages` parameter**. Seven of ten rules read a parameter (1: `model`; 2, 3, 4, 6: `messages`; 7, 8: `messages` passed to a free function). Only 5, 9, 10 read `draft` | § 2.2 now **measures** the extraction instead of calling it mechanical. The three free functions `duplicateToolCallRule` / `unresolvedToolResultRule` / `anyToolCallHasID` (`request.go:71`, `:100`, `:117`) take `[]Message` as a parameter and need **no edit** — only their two call sites move to `d.messages`. The diff is confined to `NewRequest`'s body, and with AI-10 landed nothing is jointly owned |
| 3 | The rule order is a literal slice; appending is one line | **Differs in form, holds in substance** | A **variadic argument list**, not a `[]Rule` literal: nine inline `func() *Violation` literals plus `draft.boundsRule()`, passed to `FirstFailure(rules ...Rule) error` (`validation.go:381`) | § 3's table **corrected twice**: rows 7 and 8 were recorded in the wrong order (landed order is *duplicate identities* then *orphan results*, because uniqueness is a precondition of resolution — `request.go:62–63`), and there is **no row 11**, so AI-12's rule appends as row 11 today or row 12 if AI-11 lands its cap first. `FirstFailure(d.rules()...)` is still a drop-in; no anchor is lost. `R-REX-005` asserts class-and-position equivalence between doors, never an absolute ordinal, so the spec needs no edit |
| 4 | `Request.Equal` is **exported** | **Matches — recorded default confirmed** | `func (r Request) Equal(other Request) bool` (`request.go:426`) | The conditional branch is **closed, not taken**. § 7 keeps it as a closed branch rather than deleting it. AI-12 extends `Equal` and exports no second equality. **No test-list item is removed**: `tasks.md` 3.6 keeps its wording and records which side it took |
| 5 | If exported, extend `Equal` to the extension region | **Confirmed — and the recorded rationale was wrong** | `Equal` walks nine regions in a fixed order and returns at the first difference (`request.go:426–490`); tools compare through the **free function** `toolsEqual` (`:499`) | § 7.1 now states the exact addition: one `slices.EqualFunc(r.ProviderExtensions(), other.ProviderExtensions(), providerExtensionsEqual)` block before `return true`, with a **free** comparator on `toolsEqual`'s precedent, **no** presence flag, and the read through the exported accessor as `Equal`'s GoDoc demands. **And § 7.2 adds an obligation**: the rationale claimed a dropped extension "would still pass AI-10.5's round trip" — but that pin never calls `Equal`. It walks its own nine-region list through `rebuildFromReadback` (`agenttest/request_test.go:235`) and `requireRequestsEqual` (`:329`). Extending `Equal` is **necessary and not sufficient**; both `agenttest` helpers must gain the region too |
| 6 | `Reasoning.Token()` is `([]byte, bool)` and byte-exact | **Matches** | `func (r Reasoning) Token() ([]byte, bool) { return []byte(r.token), r.hasToken }` (`reasoning_content.go:306`); backing field is `token string`, converted per call, so every read is a fresh byte-identical slice; `MaxReasoningTokenLen = 1 << 20` (`:67`) | **Nothing.** `R-REX-003` and `S-REX-012` re-anchor on this exact signature, unchanged |

**Two facts verified outside the eight rows, both of which changed the plan:**

- **The central decision survives.** `func NewRequest(model string, messages []Message, opts ...RequestOption) (Request, error)` (`request.go:173`), and `requestDraft` (`request.go:286–307`) carries **neither** a `model` nor a `messages` field. Since `RequestOption` is `func(*requestDraft)` (`:279`), an option provably cannot reach either region. § 4's `WithModel`/`WithMessages` decision stands, and its motivating claim is now compiler-checkable rather than expected.
- **The system region is stored twice** — `Request.system`/`hasSystem` (`request.go:17–18`) *and* `requestDraft.system`/`hasSystem` (`:299–300`), with `SystemInstruction()` reading the top-level pair. A `With` that freezes without re-deriving it silently reverts the region. § 2.2.1 adds `draft.freeze()` so both seeds share one freeze expression.

### 13.2 Rows 7–8 — **unresolved**; AI-11 is landing concurrently

**Verified absent at `1c4171e`**: no marker field on `Segment` (`system_instruction.go:13`, `struct{ text string }`), `Tool` (`tool.go:20`) or `Message` (`message.go:112`); no breakpoint-cap rule in the `FirstFailure` list. Two forward references exist in landed code, both weak and both pointing at branch **A**: `Segment.String()`'s GoDoc says *"AI-11 may extend this to name a cache marker, because a marker is structural rather than payload"* (`system_instruction.go:158`), and `ErrOutOfRange`'s GoDoc already names `V-REQ-24` (`validation.go:82`), so the cap's rule class exists and **neither AI-11 nor AI-12 appends one**.

| # | Check | Status | **Branch A** — and its exact edit | **Branch B** — and its exact edit |
| --- | --- | --- | --- | --- |
| 7 | Markers ride on `Segment` / `Tool` / `Message` rather than on the request | **Unresolved** | **Markers on the values.** They are reachable **transitively** through `WithSystemInstruction`, `WithTools` and `WithMessages`. **AI-12 adds no option and no production line.** Edit: § 8's totality table gets its markers row with `derive` = those three options and `changed` = a marker read on the derived value; `tasks.md` 1.7 records "transitive" and `S-REX-010` is satisfied as written | **A request-level marker region.** Edit: AI-12.1 appends **one `RequestOption`** wrapping whatever constructor AI-11 names, **one row** to § 8's table, and **one item** *(appended)* to AI-12.1's test list. `R-REX-002`'s marker clause already admits both spellings, so the delta spec needs no edit either way |
| 8 | The breakpoint cap `V-REQ-24` is a `Rule` in the request's rule list | **Unresolved** | **The cap is in the list.** The rebuild **re-runs it for free** — zero AI-12 production code. Edit: AI-12.2 appends one derive-path assertion *(appended)* — a derivation that pushes the request past the cap fails with `ErrOutOfRange` at the position construction reports. AI-11's row lands before AI-12's, so AI-12's extension rule becomes row 12 | **The cap is enforced elsewhere** (inside a marker constructor, or at a boundary the request never re-runs). Then the rebuild does **not** re-run it. Edit: AI-12 **records the gap** in `tasks.md` for the Wave 1 verify phase and asserts nothing. It must **not** add the rule itself — `V-REQ-24` belongs to AI-11, and adding it here would put a second owner on one register term |

**Decision procedure for the apply agent** — mechanical, no judgement required:

1. `grep` the landed `system_instruction.go`, `tool.go` and `message.go` for a marker field. **Present ⇒ row 7 branch A. Absent, with a request-level constructor in AI-11's surface ⇒ branch B.**
2. Read `NewRequest`'s `FirstFailure` argument list for a cap rule. **Present ⇒ row 8 branch A. Absent ⇒ branch B.**
3. The two rows are **independent**: A on one does not imply A on the other.
4. Neither branch ever removes a planned item. In every case the test list only grows.

### 13.3 Standing discipline

Rebase onto the Wave 1 head; **never merge**, never push, never `git stash` (the stash stack is shared across worktrees — Engram #2292). Do not edit `openspec/specs/ai-contract-vocabulary/spec.md`, `openspec/changes/cachicamas-ai-model-request/` or `openspec/changes/cachicamas-ai-cache-breakpoints/`.

## 14. Open questions

- [ ] **None blocking.** Every unknown is a row in § 13 with a decided branch on both sides. After the 2026-08-01 re-verification, six of the eight rows are no longer unknowns at all; the two that remain are AI-11's, and both branches carry their exact edit.
- [ ] Recorded for the **Wave 1 verify phase**, not for this change: `S-AMR-046` in AI-10's delta spec says "four permitted cells" where `R-AMR-011`'s table and the landed `rolePermittedKinds` both yield **five**. Real, cosmetic, and out of AI-12's scope to fix — `tasks.md` § 0.5 carries it.
- [ ] Non-blocking, for AI-24: whether the first real adapter wants `ProviderExtensions()` (whole set) or only `ProviderExtension(ns)` (scoped read). Both ship; if the whole-set read turns out to be an anti-pattern — an adapter iterating namespaces it does not claim — AI-24 may deprecate it. It exists now because `R-AMR-015`'s round trip needs to rebuild the region from what it read.
