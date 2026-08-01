# Design — request extension points

> **Change**: `cachicamas-ai-request-extension-points`
> **Milestone**: AI-12 · **Nodes**: AI-12.1 … AI-12.4, all `[leaf]`
> **Phase**: design · **Date**: 2026-08-01
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-request-extension-points/spec.md`
> **Consumes**: AI-10 `design.md` §§ 4, 8, 11, 12 · `backend/agent/src/ai/{request.go, system_instruction.go, validation.go}`
> **Every disposition below is `[provisional]`** and may be changed only by recording the reason **in this file, before writing the test**. Absent a recorded reason, implement what is written.

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

**Consequence to price**: every rule that reads `model` or `messages` must read them from the draft rather than from a parameter, because after § 4 those regions live on the draft. That is the same edit `WithModel`/`WithMessages` require anyway, and it is the one place this milestone touches lines AI-10.3 is writing concurrently. § 13 is the re-verification checklist for it.

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

AI-10 `design.md` § 4 owns the table. AI-12 appends **one row at the end** and moves none:

| # | Rule | Class · position | Owner |
| --- | --- | --- | --- |
| 1 … 4 | model present · messages present · messages constructed · content valid at request depth | AI-04 classes | AI-10.1 |
| 5 | applied system instruction is constructed | `ErrEmpty` at `system` | AI-10.2 |
| 6 … 8 | role/kind table · orphan tool results · duplicate call identities | `ErrMisplaced`, `ErrUnresolvedReference`, `ErrDuplicate` | AI-10.3 |
| 9 | tool choice cross-validated against the tool set | AI-08.3's three rules | AI-10.3 |
| 10 | generation-option bounds (`boundsRule`) | `ErrOutOfRange`, `ErrEmpty` | AI-10.1 |
| 11 | breakpoint cap | `ErrOutOfRange` | AI-11 |
| **12** | **each extension has a namespace, then a non-empty value, in ordinal order** | **`ErrEmpty` at `extensions[i].namespace` / `extensions[i].value`** | **AI-12.3** |

Last, because nothing depends on it and because the region is the newest. The exact ordinals of rows 6–11 are AI-10.3's and AI-11's to settle; AI-12 only requires that its own row is **appended**, so no landed first-failure outcome changes.

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

## 6. Disposition 3 `[provisional]` — last-wins per namespace, in place; an empty value is rejected

**Last-wins, not `ErrDuplicate`.** A namespace is what "the same option" means for this region. `RequestOption`'s GoDoc already says last-wins exists so AI-12 can express an override as re-application; making the escape hatch the one region a hook *cannot* revise would contradict `R-REX-002`.

**Replacement keeps the first ordinal.** The alternative — moving a replaced namespace to the end — makes read-back order depend on how many times a value was revised, which is a determinism surprise (`R-REX-009`) with nothing to recommend it. Read-back order is therefore **first-application order**, stated once and testable in one scenario (`S-REX-030`).

**An empty value is rejected** with `ErrEmpty` at `extensions[i].value`. This is AI-10.2's argument for `NewSystemInstruction` rejecting zero segments, one region over: accepting it would give the package **two spellings of "this provider has nothing extra"** — namespace absent, and namespace present with no bytes — and every later reader would have to know both. One spelling, enforced by a rule.

**A whitespace-only namespace is folded into emptiness** (`NewSegment`'s and `NewRequest`'s treatment of the model identity), because a namespace is *text* that names a provider. **A whitespace-only value is legal**, because a value is *bytes* and the package does not know they are whitespace. The asymmetry is deliberate and is the clearest statement in this file of where interpretation stops.

**No other rule.** No format, no prefix convention, no length bound, no character class, and no registry of recognised namespaces — a registry would make Layer 1 know every provider, which is the coupling the hatch exists to remove.

## 7. Disposition 4 `[provisional]` — extensions participate in equality, structurally

doc 0002's AI-12.3 item 3 says the pass-through is "inert in validation and equality", and states the check as *"two requests differing only in a third provider's namespace **validate** identically"*.

**Read "inert" as *never interpreted*, not as *excluded*.** The reason is mechanical: `R-AMR-015` proves a request round-trips by rebuilding it from what was read and comparing under `R-AMR-016`'s equality. An equality that ignored extensions would let a rebuild **silently drop every extension** and still pass that pin — the exact class of bug this milestone exists to prevent.

So: equality compares namespaces by string equality and values by byte equality, in read-back order, with no per-namespace branch anywhere. Inertness is preserved where doc 0002 asserts it — **validation** — and `S-REX-035`/`S-REX-036` assert exactly that clause, verbatim.

**Dependency on AI-10.6.** If `Request.Equal` is exported (the recorded default), AI-12.3 **extends it** to cover the region. If AI-10.6 declines to export it, AI-12 compares region-by-region inside `ai_test` and **must not export an equality of its own** — a second equality is the failure mode AI-10.6 owns. § 13 row 4.

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

**Markers.** The row is written against whatever AI-11 lands. If markers ride on `Segment`, `Tool` and `Message` — AI-10 `design.md` § 12.1's recorded expectation — the row's `derive` is `WithSystemInstruction`/`WithTools`/`WithMessages` and AI-12 adds no option. If AI-11 models them as a request-level region, the row's `derive` is AI-11's own constructor and AI-12.1 appends one option. **Both are satisfying answers to `R-REX-002`; neither is invented here.**

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
| `backend/agent/src/ai/request.go` | Modify | `WithModel`, `WithMessages`, `Request.With`; `requestDraft` gains `model`, `messages`, `extensions`; the rule list extracted to `draft.rules()`; `appliedNames` gains the extension region |
| `backend/agent/src/ai/request_extension.go` | Create | `ProviderExtension`, `WithProviderExtension`, the two request accessors, the region's rules, `String`/`GoString` |
| `backend/agent/src/ai/request_test.go` | Modify | AI-12.1 and AI-12.2 items, in `ai_test` |
| `backend/agent/src/ai/request_extension_test.go` | Create | AI-12.3 and AI-12.4 items, the two fake translators, the determinism loops |
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

## 13. Re-verification checklist for the apply agent

Run this **before writing the first test**, against the then-current `../ai-wave-1` and `../ai-11` heads. Every row is an assumption this design rests on; none may be assumed landed.

| # | Check | File · symbol | If it differs |
| --- | --- | --- | --- |
| 1 | `tools` and `toolChoice` reach the request as `RequestOption`s | `request.go` · `WithTools`, `WithToolChoice`, `requestDraft` | AI-12.1 adds the missing option(s) — same pattern as `WithModel`; append to AI-12.1's test list |
| 2 | AI-10.3's cross-region rules are `Rule` values in `NewRequest`'s `FirstFailure` list, readable from draft state | `request.go` · `NewRequest` | Move the reads to `draft.*` during the § 2.2 extraction and call it out in the PR body |
| 3 | The rule order is a literal slice and appending is one line | `request.go` · the `FirstFailure` argument list · AI-10 `design.md` § 4 | `R-REX-005`'s position-equality assertions need a different anchor; stop and record before proceeding |
| 4 | `func (r Request) Equal(other Request) bool` is exported | `request.go` · `Request.Equal` · AI-10 `design.md` § 11.2 | If **not** exported: compare region-by-region inside `ai_test`; **do not** export an equality here (§ 7) |
| 5 | If `Equal` is exported, extend it to the extension region | `request.go` · `Request.Equal` | Required by `R-REX-007`; not optional |
| 6 | `Reasoning.Token()` is `([]byte, bool)` and byte-exact | `reasoning_content.go` · `Reasoning.Token` | `R-REX-003`'s pin re-anchors on whatever AI-07 exposes |
| 7 | Markers ride on `Segment`, `Tool`, `Message` rather than on the request | `system_instruction.go` · `Segment`; `tool.go` · `Tool`; `message.go` · `Message`; AI-11 `design.md` | Add one option and one totality row; **append** to AI-12.1's test list, never substitute (§ 8) |
| 8 | The breakpoint cap `V-REQ-24` is a `Rule` in the slice | `request.go` · the `FirstFailure` list | The rebuild re-runs it for free; assert it on the derive path as an appended case |

Rebase onto the advancing Wave 1 head; **never merge**, never push, never `git stash` (the stash stack is shared across worktrees — Engram #2292).

## 14. Open questions

- [ ] **None blocking.** Every unknown is a row in § 13 with a decided branch on both sides.
- [ ] Non-blocking, for AI-24: whether the first real adapter wants `ProviderExtensions()` (whole set) or only `ProviderExtension(ns)` (scoped read). Both ship; if the whole-set read turns out to be an anti-pattern — an adapter iterating namespaces it does not claim — AI-24 may deprecate it. It exists now because `R-AMR-015`'s round trip needs to rebuild the region from what it read.
