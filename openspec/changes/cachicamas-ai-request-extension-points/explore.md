# Explore — request extension points

> **Change**: `cachicamas-ai-request-extension-points`
> **Milestone**: AI-12 — Add per-request options, the escape hatch, and rebuild
> **Nodes**: AI-12.1 … AI-12.4, all `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-01
> **Worktree**: `cachicamas-worktrees/ai-12` · **Branch**: `feat/ai-12-request-extension-points` (planned against Wave 1 head `07d2027`; **rebased onto finished Wave 1 head `1c4171e`**, which carries AI-04 … AI-10 complete and green)
> **Re-verified**: 2026-08-01 — § 6 resolved against the landed AI-10 surface; see § 6's status banner
> **Depends on**: AI-10, AI-11 · **Blocks**: AI-24, AI-26.7, Layer 2's pre-request hook (doc 0001 § 6 seam 1)
> **Closes**: gap **G9**; supplies the mechanism **G11** stands on

---

## 1. What this milestone is

AI-12 does not add a contract. It adds three **extension points** to the contract AI-10 landed, so that the request stops being a value you can only build and starts being a value you can *derive*:

| Register term | What it names | Leaf |
| --- | --- | --- |
| `V-REQ-29` **request rebuild** | Deriving a modified request from an existing one such that the original is observably unmodified and the derived request validates independently | AI-12.1 |
| `V-REQ-27` **per-request option override** | A generation option supplied or replaced for one call without rebuilding the caller's own defaults | AI-12.2 |
| `V-REQ-28` **provider escape hatch** | A typed-but-opaque, namespaced pass-through carrying a provider-specific value the neutral vocabulary deliberately does not model | AI-12.3 |

All three terms are **already in the canonical register** (`openspec/specs/ai-contract-vocabulary/spec.md` §3, rows `V-REQ-27` … `V-REQ-29`) and all three name AI-12 as their owner. **This milestone therefore invents no vocabulary and amends no register row.** § 8 records the check.

AI-12.4 is not a fourth concept; it is the property the other three must hold jointly: reading them back twice yields the same thing.

## 2. The design principle, stated before the mechanism

Doc 0001 § 3.3 closes its nine-row leakage register with the sentence this milestone exists to obey:

> **the correct response to leakage is a typed pass-through, not a wider neutral vocabulary.** Every field added to the neutral model to accommodate one provider becomes a field every other adapter must ignore, and the model grows without bound.

Doc 0002's AI-12 charter repeats it as a `**Note:**` and calls it "deliberate and belongs in the SDD". It is not decoration. It is the *acceptance criterion* in disguise: AI-12's acceptance is "a provider-specific value survives to its adapter **without any other adapter needing to know it exists**". A neutral field cannot satisfy that clause, because a neutral field is by definition one every adapter must read or deliberately ignore. Only a namespaced pass-through can.

The corollary a reviewer should hold: **every future request-shaped provider quirk has a default answer, and the answer is not "add a field".** AI-10's `design.md` § 8.2 already named this milestone as the owner of six rejected generation-option candidates — `top_k`, frequency/presence penalties, `seed`, thinking budget, user/metadata identifiers. Those six are the escape hatch's first customers, and none of them may become a neutral option.

The counter-pressure is real and worth naming, so it can be refused on purpose rather than by accident: a pass-through is *less discoverable* than a field. A caller must know a namespace exists. That cost is accepted, because the alternative cost — every adapter, forever, ignoring every other provider's field — is unbounded while this one is a documentation problem.

## 3. What already exists, and what each of it gives this milestone

| Landed | File | What AI-12 consumes |
| --- | --- | --- |
| AI-04 taxonomy | `validation.go` | `Invalid`, `At`, `AtIndex`, `FirstFailure`, `Rule`, seven rule classes, the redaction posture |
| AI-10.1 request core | `request.go` | `Request` (sealed), `NewRequest`, **`RequestOption`** and its seal, **`requestDraft`**, the four generation-option constructors, the `(value, bool)` accessors, `String`/`GoString` |
| AI-10.2 system instruction | `system_instruction.go` | `Segment`, `SystemInstruction`, `WithSystemInstruction`, `Segments()` |
| AI-08 tools | `tool_set.go`, `tool_choice.go` | `ToolSet`, `ToolChoice`, `ValidateAgainst` — the regions AI-10.3 attaches to the request |
| AI-05 / AI-06 / AI-07 | `message.go`, `content_part.go`, `reasoning_content.go` | `Message`, `Part`, `Reasoning.Token()` — the byte-exact round-trip token AI-12.1 item 3 re-pins |

### 3.1 The seam AI-10 cut for this milestone, by name

`request.go`'s `RequestOption` GoDoc is written *at* AI-12:

> An option is total. It never fails, never validates and never allocates a violation, because every rule runs once, in `NewRequest`, in the documented order. […] Applying the same option twice is **last-wins, deliberately: that is what lets AI-12 express a per-request override as re-application rather than as a new mechanism.**

And `NewRequest`'s GoDoc, on rule 4:

> That is defence in depth on purpose. **AI-12 rebuilds requests** and an adapter may one day produce messages by another path […]

AI-10's `design.md` § 12.2 records the intended shapes:

- per-request override — the same `RequestOption` applied again;
- rebuild — `func (r Request) With(opts ...RequestOption) (Request, error)`, "read the frozen request into a `requestDraft`, apply, re-validate through the same rule slice, freeze. No existing signature changes, and validation stays single-sited";
- escape hatch — "one more optional region and therefore one more `RequestOption` — `WithProviderExtension(namespace string, value …)`".

**This exploration confirms all three are still the right shapes, and finds one thing § 12.2 did not price.** See § 4.

## 4. The tension § 12.2 did not price: totality

AI-12.1 item 2 is the load-bearing item of the whole milestone:

> Deriving is **total**: every region — system segments, messages, tools, tool choice, options, markers — is reachable by the rebuild path. A region the hook cannot reach is a region a cache breakpoint or injected context can never be applied to.

`RequestOption` today reaches five regions: `maxOutputTokens`, `temperature`, `topP`, `stopSequences`, `system`. AI-10.3 adds `tools` and `toolChoice`. That leaves **two regions that are `NewRequest` parameters and therefore unreachable by any option**: the model identity and the messages.

> **Verified at `1c4171e`.** Nine regions exist on a landed `Request`, seven reachable by an option and two not. § 6.1.5 carries the proof; the complete enumeration the rebuild must be total over is `design.md` § 8.1. AI-10.3 landed `tools` and `toolChoice` as options exactly as expected, so this paragraph's arithmetic is unchanged and shape **C** stands.

A rebuild that cannot replace the messages is not a rebuild. Injected repository context — the second use doc 0001 § 6 seam 1 names for the pre-request hook — *is* a message-region edit. So totality is not satisfiable without moving the required regions into the option mechanism too.

Three shapes were considered.

| Shape | How it reaches messages | Cost |
| --- | --- | --- |
| **A. `With(model string, messages []Message, opts ...RequestOption)`** | parameters, mirroring `NewRequest` | Every caller restates two regions it does not want to change. A hook that only wants to add a stop sequence must first read and re-pass the whole transcript, which is the exact ceremony the seam exists to remove |
| **B. A second option type for the derive path only** | a parallel `RequestEdit` closure | Two option vocabularies. Generation options would have to exist twice, and AI-12.2 item 2's "the same sentinels as construction" becomes a claim about two code paths rather than a structural fact. Rejected on AI-06's "one rule set, two callers" principle |
| **C. Widen `RequestOption` with `WithModel` and `WithMessages`; both `NewRequest` and `With` seed a draft and apply options over it** | one uniform mechanism | One new hazard, priced in § 4.1. In exchange the draft becomes the single seat of every region, and **`NewRequest` and `With` become the same three lines over two different seeds** |

**C is recommended.** It is the only one under which AI-12.2 item 2 — "there is no second, weaker validation path" — is *structurally* true rather than a discipline the reviewer must check. Under C there is exactly one rule slice, evaluated over exactly one draft type, reached from exactly two seeds: the parameters (`NewRequest`) or the frozen request (`With`).

### 4.1 The hazard C introduces, and why it is acceptable

Under C, `NewRequest("a", msgs, WithModel("b"))` is legal, and last-wins makes the model `"b"`. The required region acquires a second spelling at construction time.

It is acceptable because it is **the landed semantics applied uniformly rather than a new rule**: `RequestOption` already documents last-wins, and a reader who knows that predicts this outcome correctly. The alternative — making `WithModel` illegal inside `NewRequest` — would need either a runtime rule (a second failure class for "you used the wrong door") or two option types, which is shape B.

It must be **pinned by test rather than left to inference**: the disposition is asserted, in AI-10.3's `S-AMR-050` style, so the next reader cannot close it by reflex.

## 5. The escape hatch: what "typed but opaque" has to mean here

Four candidate payload shapes, judged against the register row, the acceptance clause, and the Layer 1 import rule.

| Payload | Byte-exact? | Deterministic read-back? | stdlib-only? | Verdict |
| --- | --- | --- | --- | --- |
| `[]byte` behind a sealed value type | yes, trivially | yes — the region is an ordered slice | yes, no encoder needed | **recommended** |
| `any` | no — equality and copying become interface-shaped | no | yes | **rejected**: an `any` lets a vendor wire type cross the Layer 1 method boundary, which doc 0001 § 8's contract checklist forbids outright |
| `map[string]any` | no | **no** — map iteration is the exact nondeterminism AI-12.4 exists to exclude | yes | **rejected** |
| A `json.RawMessage`-shaped alias | yes | yes | **no** — it names an encoding, and Layer 1 must stay at zero requires until AI-24 chooses a transport | **rejected**: it also breaks opacity, because naming an encoding is an invitation to parse |

`[]byte` is the shape AI-07 already chose for the reasoning round-trip token (`Reasoning.Token() ([]byte, bool)`), for the same reason: the package carries bytes it must never interpret. Reusing that precedent means one opacity discipline in the package instead of two.

**"Typed" is therefore about the container, not the payload**: the namespace and the bytes travel inside a sealed value type with a constructor-free exported surface, so a consumer can neither forge one nor confuse it with a `[]byte` that means something else.

### 5.1 The uniqueness question

Two applications of the same namespace: reject as `ErrDuplicate`, or last-wins?

**Last-wins**, on the landed option semantics. A namespace is what "the same option" means for this region, and last-wins is precisely what lets a pre-request hook *override* an extension a caller set — which is the same capability AI-12.2 gives generation options. Rejecting would make the escape hatch the one region a hook cannot revise, which contradicts AI-12.1 item 2.

The read-back **ordinal** on replacement is a genuine choice with no precedent to inherit, and § 6 of `design.md` decides it.

### 5.2 What "invisible to a different adapter" can even be tested against today

No adapter exists in Layer 1 until AI-24. The acceptance clause is nonetheless testable **without one**, and the test shape matters enough to record here so the apply agent does not invent a production adapter to satisfy it:

Two *fake* translator functions live in the external test package. Each reads only its own namespace through the request's accessor and produces a byte string. The claims are then mechanical:

1. the claiming translator's output contains the value byte-exactly;
2. the foreign translator's output for the request **with** the foreign extension is byte-identical to its output for the request **without** it.

Claim 2 is the acceptance clause. It needs no adapter, no network and no vendor package — only the guarantee that reading is namespace-scoped.

## 6. Concurrency: two siblings this plan sits downstream of

> **Status — resolved 2026-08-01 (Phase 0 re-verification gate).** This section was written while AI-10's second half and AI-11 were both unlanded. **AI-10 is now finished** and this worktree is rebased onto Wave 1 head `1c4171e`, which carries AI-04 … AI-10 complete and green. Every AI-10 row below is now a **read fact** with the exact signature that was read. **AI-11 is still being implemented** in `../ai-11`; its two rows stay explicitly unresolved, with both branches and the exact edit each implies, so the apply agent executes a decision rather than repeating this analysis.

### 6.1 AI-10 — landed; all six rows resolved

Read at Wave 1 head `1c4171e`, in `backend/agent/src/ai/`.

| # | Assumption as recorded | Verdict | The fact that was read |
| --- | --- | --- | --- |
| 1 | `tools` / `toolChoice` arrive as `RequestOption`s, not parameters | **Matches** | `func WithTools(tools ToolSet) RequestOption` (`request.go:334`) and `func WithToolChoice(choice ToolChoice) RequestOption` (`request.go:344`). `requestDraft` carries `tools ToolSet / hasTools bool` and `toolChoice ToolChoice / hasToolChoice bool` (`request.go:302–306`). Both regions are reachable by an option **for free**; AI-12 adds no constructor for them |
| 2 | AI-10.3's cross-region rules read **draft state** | **Differs — branch taken** | They close over the **`messages` parameter**. Of the ten landed rules, **seven** read a parameter: rule 1 reads `model`; rules 2, 3, 4, 6 iterate `messages`; rules 7 and 8 pass `messages` to `duplicateToolCallRule(messages)` / `unresolvedToolResultRule(messages)`. Only rules 5, 9 and 10 read `draft`. See § 6.1.1 — the extraction moves more than "mechanical" suggested, but **less than feared** |
| 3 | The rule order is a **literal slice**; appending is one line | **Differs in form, holds in substance** | It is a **variadic argument list**, not a `[]Rule` literal: nine inline `func() *Violation` literals plus one method call `draft.boundsRule()`, passed to `FirstFailure(rules ...Rule) error` (`validation.go:381`). Appending is still one line, and `FirstFailure(d.rules()...)` is a drop-in for the § 2.2 extraction. The **real order is § 6.1.2**, and it corrects the design's table |
| 4 | `func (r Request) Equal(other Request) bool` **is exported** | **Matches — recorded default confirmed** | `func (r Request) Equal(other Request) bool` at `request.go:426`. The conditional branch ("compare region-by-region in `ai_test`, export no second equality") is **closed, not taken**. It is retained in `design.md` § 7 as a closed branch rather than deleted |
| 5 | If exported, AI-12.3 **must extend it** | **Confirmed, and the stated reason is wrong** | `Equal` walks nine regions (§ 6.1.3) and AI-12.3 must append a tenth block. But the *rationale* — "or a rebuild silently drops extensions and still passes AI-10.5's round-trip pin" — **does not hold**: AI-10.5's pin never calls `Equal`. § 6.1.4 records the real hole, which is larger |
| 6 | `Reasoning.Token()` is `([]byte, bool)` and byte-exact | **Matches** | `func (r Reasoning) Token() ([]byte, bool) { return []byte(r.token), r.hasToken }` (`reasoning_content.go:306`). The field is `token string`, converted per call, so every read is a fresh slice and byte-identical to what was supplied. `MaxReasoningTokenLen = 1 << 20` (`reasoning_content.go:67`). `R-REX-003`'s pin re-anchors on exactly this signature, unchanged |

#### 6.1.1 Row 2 in detail — what `draft.rules()` actually moves

The extraction is bigger than the assumption priced and smaller than the risk row feared, and the difference matters to the merge surface:

- **Bigger**: seven of ten rules change the expression they read, from `model` / `messages` to `d.model` / `d.messages`. The design called this "the same edit `WithModel`/`WithMessages` require anyway" — that remains true, and it is now measured rather than assumed.
- **Smaller**: the three free functions that carry the cross-region logic — `duplicateToolCallRule(messages []Message)`, `unresolvedToolResultRule(messages []Message)`, `anyToolCallHasID(messages []Message, id string)` (`request.go:71`, `:100`, `:117`) — take `[]Message` as a **parameter** and need **no edit at all**. Only their two call sites inside `NewRequest` move to `d.messages`.

So the diff against lines AI-10.3 wrote is confined to `NewRequest`'s body. The risk row's "the conflict is in one function" was correct, and it is now moot: AI-10 has landed, so there is no concurrent writer of `request.go` left.

#### 6.1.2 Row 3 in detail — the real landed rule order

Read directly from `NewRequest`'s `FirstFailure(...)` argument list (`request.go:180–234`):

| # | Rule | Class · position | Reads |
| --- | --- | --- | --- |
| 1 | model identity present (whitespace folded) | `ErrEmpty` at `model` | `model` **param** |
| 2 | at least one message | `ErrEmpty` at `messages` | `messages` **param** |
| 3 | every message constructed (`ID().IsZero()`) | `ErrEmpty` at `messages[i]` | `messages` **param** |
| 4 | content valid at request depth (`validateContent`) | AI-06's classes at `messages[i].content[j]` | `messages` **param** |
| 5 | applied system instruction is constructed | `ErrEmpty` at `system` | `draft` |
| 6 | role/kind table (`roleAllowsKind`) | `ErrMisplaced` at `messages[i].content[j]` | `messages` **param** |
| 7 | **duplicate tool-call identities** | `ErrDuplicate` | `messages` **param** |
| 8 | **orphan tool results** | `ErrUnresolvedReference` | `messages` **param** |
| 9 | tool choice cross-validated (`violationOf(draft.toolChoice.ValidateAgainst(draft.tools))`) | AI-08.3's three rules | `draft` |
| 10 | `draft.boundsRule()` | `ErrOutOfRange`, `ErrEmpty` | `draft` |

**Two corrections fall out.**

1. `design.md` § 3's table listed rows 6…8 as "role/kind table · **orphan tool results · duplicate call identities**". The landed order is the **reverse** for 7 and 8, and deliberately so: `duplicateToolCallRule`'s GoDoc (`request.go:62–63`) states that uniqueness is a *precondition* of `unresolvedToolResultRule`, "a rule that resolves by a non-unique key is not a rule — and that is why this rule is checked first". `design.md` § 3 is corrected.
2. There is **no row 11**. AI-11's breakpoint cap does not exist in the list. AI-12's extension rule therefore appends as row **11** today, and becomes row **12** only if AI-11 lands its cap first. Both orderings satisfy AI-12's only requirement, which is that its own row is *appended* and moves no landed row.

#### 6.1.3 Row 5 in detail — how `Request.Equal` walks regions today

`request.go:426–490`, in order, returning `false` at the first difference:

1. `r.model != other.model` — the unexported field, not `Model()`;
2. `slices.EqualFunc(r.Messages(), other.Messages(), Message.Equal)`;
3. system — presence flags compared first, then `SystemInstruction.Equal` (`system_instruction.go:116`);
4. tools — presence flags, then `slices.EqualFunc(rTools.Tools(), oTools.Tools(), toolsEqual)`, where `toolsEqual` is a **free function** (`request.go:499`) comparing name, description and `bytes.Equal` on the schema;
5. tool choice — presence flags, then `Mode()`, `namesOne` and `Name()`;
6–9. `MaxOutputTokens()`, `Temperature()`, `TopP()`, `StopSequences()` — each presence flag plus value, the last via `slices.Equal`;
then `return true`.

**Exactly what AI-12.3 must add**: one more block immediately before `return true`, comparing the extension region through the exported accessor —

```go
if !slices.EqualFunc(r.ProviderExtensions(), other.ProviderExtensions(), providerExtensionsEqual) {
    return false
}
```

— with `providerExtensionsEqual(a, b ProviderExtension) bool` a **free function** on `toolsEqual`'s precedent (`a.Namespace() == b.Namespace() && bytes.Equal(a.Value(), b.Value())`), not a new exported method on `ProviderExtension`. The region carries **no presence flag** (it is a slice; empty and absent are one fact), so the comparison is one line and needs no flag branch. Reading through `ProviderExtensions()` rather than `r.options.extensions` is required by `Equal`'s own GoDoc, which states that every comparison reads the exported surface so the method proves the documented equality is reachable from outside.

#### 6.1.4 The hole row 5's rationale pointed at, and where it actually is

`design.md` § 7 argued that an `Equal` ignoring extensions would let a rebuild drop them "and still pass AI-10.5's round-trip pin". **That is false as stated, and the true situation is worse.**

`backend/agent/src/agenttest/request_test.go` never calls `Request.Equal`. `TestRequest_WholeRequestRoundTrip_ReconstructsAnEqualRequest` (`:137`) rebuilds through its own `rebuildFromReadback` (`:235`) and compares through its own `requireRequestsEqual` (`:329`) — two hand-written region walks. So the round-trip pin has a **two-sided** blind spot the moment a new region exists:

- `rebuildFromReadback` reads nine regions and re-applies them; it will not re-apply an extension it does not know about, so the rebuilt request silently loses the region;
- `requireRequestsEqual` compares those same nine regions; it will not notice the loss.

The pin then passes on a request that lost its extensions — which is precisely the failure the design named, reached by a different route. Extending `Request.Equal` alone **does not close it**.

`agenttest` guards its *kind* set (`TestPartKindReaders_EveryRegisteredKind_HasAReader`, `:111`) but has **no equivalent region-level guard**, which is exactly why a new region slips through.

**Consequence for AI-12.3**: it must extend `rebuildFromReadback` and `requireRequestsEqual` in `backend/agent/src/agenttest/request_test.go` as well. This adds one file to `design.md` § 10's table and one item to AI-12.3's list. It is an **appended** obligation, not a substitution.

#### 6.1.5 The central decision survives — verified

`func NewRequest(model string, messages []Message, opts ...RequestOption) (Request, error)` (`request.go:173`). `requestDraft` (`request.go:286–307`) carries seven optional regions and **neither a `model` nor a `messages` field**. Since `RequestOption` is `func(*requestDraft)` (`request.go:279`), an option provably cannot reach either region. § 4's shape **C** — widen `RequestOption` with `WithModel` and `WithMessages` — stands unchanged, and the totality argument that motivates it is now a compiler-checkable fact rather than an expectation.

#### 6.1.6 One landed hazard the plan did not price: the system region is stored twice

`Request` carries `system SystemInstruction` and `hasSystem bool` **at the top level** (`request.go:17–18`) *and* inside `options requestDraft` (`request.go:299–300`). `NewRequest` sets both from the same draft on the way out (`request.go:237–243`), and `Request.SystemInstruction()` (`system_instruction.go:127`) reads the **top-level** pair.

`With`'s second seed is `r.options`. A `With` that copies the draft, applies options and freezes **without re-deriving the top-level pair** would silently revert the system region to whatever the composite literal defaults to. The mitigation is symmetry, not vigilance: extract the freeze as well, so both seeds share one expression —

```go
func (d requestDraft) freeze() Request
```

— and `NewRequest` and `With` each become `if err := FirstFailure(d.rules()...); err != nil { return Request{}, err }; return d.freeze(), nil`. One draft, one rule slice, **one freeze**, two seeds. This is an addition to AI-12.1's ordered work, recorded in `design.md` § 2.2.

#### 6.1.7 The AI-10.4 constraint AI-12 must design under

AI-10.4 landed `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` (`import_boundary_test.go:278`), which runs `go list -deps` — deliberately **without** `-test` — over `github.com/cachicamas/backend/agent/src/ai` and fails if the closure contains `net`, `net/http`, `os` or `io/fs`.

The measured closure today is `errors`, `sync`, `sync/atomic`, `io`, `iter`, `unicode`, `unicode/utf8`, `bytes`, `cmp`, `slices`, `strconv`, `strings`, `math/bits`, plus `runtime` internals. **No `fmt`. No `encoding/json`.**

`encoding/json` is banned from non-test files in this package as a **consequence** of that guard rather than by a named rule: `encoding/json` imports `fmt`, which imports `os`, so importing it turns the guard red. `json_syntax.go:5–6` records the measurement and is why the package hand-wrote `isWellFormedJSON` instead of calling `encoding/json.Valid`.

**AI-12 needs nothing new.** `ProviderExtension` requires `bytes` (byte equality), `slices` (clone), `strings` (`Builder`, `TrimSpace`) and `strconv` (`Itoa`) — all four already in the closure. Two concrete obligations follow, recorded in `design.md` § 5: the extension region's renderings must be built with `strings.Builder`/`strconv` exactly as `Request.String()` is, **never** `fmt.Sprintf`; and the "typed but opaque" payload was already chosen so that no encoder is ever needed to carry it, which is now confirmed as a hard constraint rather than a preference.

### 6.2 AI-11 — still unlanded; both rows remain open, both branches decided

`cachicamas-ai-cache-breakpoints` defines `V-REQ-23` **cache-boundary marker**, `V-REQ-24` **breakpoint cap** and `V-REQ-25` **invalidation cascade**. It is being implemented **right now** in `../ai-11` and its result cannot be read from here.

**What was verified**: no marker surface exists at `1c4171e`. `Segment` is `struct{ text string }` (`system_instruction.go:13`) with no marker field; `Tool` carries name, description and schema (`tool.go:20`); `Message` carries id, role and content (`message.go:112`). The `FirstFailure` list holds no cap rule.

**The only two forward references in landed code**, both weak but both pointing the same way:

- `Segment.String()`'s GoDoc (`system_instruction.go:158`): *"AI-11 may extend this to name a cache marker, because a marker is structural rather than payload."* A `String()` that can *name* a marker implies the marker is a field of the value it renders.
- `ErrOutOfRange`'s GoDoc (`validation.go:82`) already names `V-REQ-24` as its motivating case, so the **class the cap needs already exists**. AI-11 appends no rule class, and neither does AI-12.

| # | Row | Status | Branch **A** — and its exact edit | Branch **B** — and its exact edit |
| --- | --- | --- | --- | --- |
| 7 | Where markers live | **Unresolved** | **Markers on `Segment` / `Tool` / `Message`** (the landed hint favours this). They are reachable **transitively** the moment `WithSystemInstruction`, `WithTools` and `WithMessages` exist. **AI-12 adds no option.** The § 8 totality table's markers row gets `derive` = the three region-level options and `changed` = a read of the marker on the derived value. `S-REX-010` is satisfied as written | **A request-level marker region.** AI-12.1 **appends one `RequestOption`** wrapping whatever constructor AI-11 names, **one row** to the § 8 totality table, and **one item** to AI-12.1's test list, marked *(appended)*. `R-REX-002`'s marker clause already admits both, so the delta spec needs no edit |
| 8 | The breakpoint cap `V-REQ-24` | **Unresolved** | **The cap is a `Rule` in the request's rule list.** The rebuild re-runs it **for free** — no AI-12 production code at all. AI-12.2 appends one derive-path assertion: a derivation that pushes the request past the cap fails with `ErrOutOfRange` at the same position construction reports. Ordinal: AI-11's row lands before AI-12's, so AI-12's extension rule becomes row 12 | **The cap is enforced elsewhere** (e.g. inside a marker constructor, or at a boundary the request never re-runs). Then the rebuild does **not** re-run it, and AI-12 must **record that as a gap** in `tasks.md` for the Wave 1 verify phase rather than assert a property that does not hold. AI-12 must not add the rule itself — `V-REQ-24` is AI-11's |

**Decision rule for the apply agent**: read `../ai-11`'s landed `system_instruction.go` / `tool.go` / `message.go` for a marker field, and `request.go`'s `FirstFailure` list for a cap rule. A marker field present ⇒ branch A on row 7. A cap rule present in the list ⇒ branch A on row 8. The two rows are **independent**; A on one does not imply A on the other. Neither branch is a substitution: in every case the test list only grows.

**Neither sibling's surface is invented here.** Where this plan names a symbol that does not exist, it names it in this section and nowhere else.

## 7. Prior art inside the package

Three landed patterns AI-12 reuses rather than re-deriving.

1. **Sealing by unexported field / unexported parameter type** — `Part`'s payload seal (AI-06), `RequestOption`'s `*requestDraft` parameter (AI-10.1). `ProviderExtension` is sealed the same way: unexported fields, no exported constructor, produced only by the request on read.
2. **Ordered slice, never a map** — `ruleClasses` (`validation.go`): *"It is a slice and not a map on purpose. Nothing in this package may let an unordered iteration decide anything, and a registry is where that temptation is strongest."* The extension region is the next-strongest temptation, and AI-12.4 is the test that it was refused.
3. **The exhaustiveness-guard pattern** (AI-06.4, Engram #2301) — parse the package's own source with `go/ast`, cross-check three independent answers, and prove the guard **bites** against a scratch violation. AI-12 has no `[guard]` node, so this is deliberately *not* reproduced; § 9 records why.

## 8. Vocabulary check — no register amendment required

The register already carries every term this milestone needs, with AI-12 named as owner:

- `V-REQ-27` per-request option override · `V-REQ-28` provider escape hatch · `V-REQ-29` request rebuild.

`V-REQ-28`'s row also carries the two clauses this milestone must not violate: *"It survives to its own adapter; every other adapter ignores it without needing to know it exists"*, and trap 2 — *"**Not** a way to use a provider that has no adapter"*.

**No new rule class is needed either.** The only rules AI-12 adds are the extension region's own construction rules, and both are emptiness:

| Candidate rule | Class | Why an existing class fits |
| --- | --- | --- |
| A namespace must be present | `ErrEmpty` | A required value is empty — the class verbatim, and the same fold `NewSegment` applies to whitespace |
| A value must carry at least one byte | `ErrEmpty` | Same. § 6 of `design.md` argues why an empty value is rejected rather than accepted |

No class describes something the seven landed ones do not, so AI-04's append rule is **not** invoked. This is worth stating positively: AI-10.3 appended `ErrMisplaced` and paid for two registry mirrors; AI-12 pays nothing because its rules are ordinary.

**If apply discovers a genuinely new class, it stops and reports it upward.** This change never edits `openspec/specs/ai-contract-vocabulary/spec.md`.

### 8.1 Re-verified at `1c4171e`

The seven landed classes are `ErrEmpty`, `ErrNotInVocabulary`, `ErrOutOfRange`, `ErrMalformed`, `ErrUnresolvedReference`, `ErrDuplicate` and `ErrMisplaced` (`validation.go:70–131`, mirrored in `ruleClasses`). Both of AI-12's rules are `ErrEmpty`. **No class is appended, and no registry mirror is touched.** `ErrOutOfRange`'s GoDoc already names `V-REQ-24`, so AI-11 will not need one either.

### 8.2 A finding in AI-10's own spec — recorded, not fixed

While reading the landed role/kind table this exploration found a discrepancy in **AI-10's** delta spec, `openspec/changes/cachicamas-ai-model-request/specs/ai-model-request/spec.md`:

> **`S-AMR-046`** — *"Given each of the **four** permitted cells in turn, when a request is built holding it, then construction succeeds — so the table is proven in both directions, **twelve cells total**."*

`R-AMR-011`'s own table is 4 kinds × 3 roles = 12 cells, of which **five** are permitted: `text`/user, `text`/assistant, `reasoning`/assistant, `tool_call`/assistant, `tool_result`/tool. The landed implementation agrees — `rolePermittedKinds` (`request.go:30–34`) holds `{Text}`, `{Text, Reasoning, ToolCall}`, `{ToolResult}` = 1 + 3 + 1 = **five** entries.

**The prose is wrong; the table, the "twelve cells total" count and the code are right.** The discrepancy is **real**, cosmetic, and confined to one word of scenario prose. It is recorded here as a **finding for the Wave 1 verify phase**. AI-12 does **not** edit AI-10's artifacts: they belong to `cachicamas-ai-model-request`, which this change is forbidden to touch, and correcting another change's spec from here would put the fix outside the review that owns it.

## 9. Budget, node shapes, and what is deliberately not built

**Split trigger 4 fires**, and it is forecast here before a line of Go exists, per doc 0002's rule. `tasks.md` carries the table; the estimate is ~950–1050 changed lines, against a "prefer < 250, reassess before 400" milestone rule. The wave-level PR budget is 5000+ lines and was accepted up front (`exception-ok`), so the forecast is a statement, not a stop.

The mitigation is the same one AI-10 used and doc 0002 prescribes: **the leaf boundary is the commit boundary**, so the chain is reviewable in four slices without rework. AI-12.2 and AI-12.3 are declared parallel by the charter and are genuinely disjoint — one touches the generation-option draft fields, the other adds a new file — so the two middle slices may be reviewed in either order.

Deliberately not built, each with its owner:

| Not built | Why | Owner |
| --- | --- | --- |
| The pre-request hook itself | Layer 1 supplies rebuildability; the hook is `V-OUT-13` and belongs to Layer 2 | doc 0001 § 6 seam 1 · **G11** |
| Any concrete adapter, or any real provider namespace | Trap 2. The escape hatch is not a way to use a provider that has no adapter | AI-24 … AI-31 |
| A registry of known namespaces, or namespace validation beyond emptiness | Would require Layer 1 to know every provider — the exact coupling the hatch exists to avoid | nobody |
| **Removal** of an option or an extension (an "unset" operation) | There is no consumer, and it would give the package a second spelling of absence, which every landed contract refuses. A caller who wants a request without a region builds one | reopened by AI-24 or AI-26.7 if an adapter needs it |
| Wire-byte determinism | AI-12.4 guarantees only that the **neutral surface** cannot be the source of nondeterminism; wire bytes do not exist yet | **AI-26.1 / AI-26.4** |
| A `[guard]` node scanning for `map` in the extension region | doc 0002 gives AI-12 four `[leaf]` nodes and no guard. Adding one would be scope this milestone's charter does not carry; AI-12.4's behavioral assertion is what the charter asks for | — |
| Serialization of the extension region | Layer 1 is at zero requires until AI-24, and the hatch is designed so no encoder is ever needed to carry it | **AI-26.7** |

## 10. Recommendation

Proceed to proposal with shape **C** (§ 4): one `RequestOption` vocabulary widened to cover every region, one `requestDraft` holding all of them, one rule slice, and two seeds — `NewRequest` from parameters, `With` from a frozen request. The escape hatch is one more optional region on that draft, carrying `[]byte` behind a sealed read-only value type, stored in an ordered slice keyed by namespace with last-wins replacement.

Four leaves, four commits, in charter order. AI-12.2 and AI-12.3 may swap.

## 11. Risks

Re-scored at `1c4171e`, after § 6's resolution.

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| ~~AI-10.3/AI-10.6 land a shape this plan assumed differently~~ | **Closed** | — | AI-10 is landed and all six rows are resolved in § 6.1. Four matched, two differed with their branch taken; none invalidated a decision |
| ~~Extraction of AI-10.3's rule slice produces a merge conflict in `request.go`~~ | **Closed** | — | AI-10 has landed; there is no concurrent writer of `request.go` left. § 6.1.1 measures the extraction: seven of ten rules change one expression each, and the three cross-region free functions need no edit |
| ~~Equality drifts if AI-10.6 does not export `Equal`~~ | **Closed** | — | `Request.Equal` **is** exported (`request.go:426`). AI-12 extends it and exports no second equality |
| **The AI-10.5 round-trip pin silently narrows when the extension region lands** | **Certain if unaddressed** | **High** | **Newly discovered — § 6.1.4.** `agenttest`'s round trip uses its own `rebuildFromReadback` and `requireRequestsEqual`, not `Request.Equal`, and both walk a fixed nine-region list. AI-12.3 must extend both, or the pin passes on a request that lost its extensions |
| **`With` silently reverts the system region** | Medium | **High** | **Newly discovered — § 6.1.6.** The system region is stored twice on `Request`. Mitigated structurally by extracting `draft.freeze()` so both seeds share one freeze expression, rather than by remembering to set two fields |
| AI-11's marker surface forces a request-level region | Medium | Low | § 6.2 row 7 states both branches with the exact edit each implies; the extra option is one constructor and one assertion, appended not substituted |
| AI-11 enforces the breakpoint cap outside the request's rule list | Low-Medium | Low | § 6.2 row 8 branch B: AI-12 records the gap for Wave 1 verify rather than asserting a property that does not hold, and never adds `V-REQ-24`'s rule itself |
| The escape hatch becomes a dumping ground for things that should be neutral | Medium | **High** | `V-REQ-26`'s admission test is the gate, and AI-10 `design.md` § 8.2 already lists which candidates are the hatch's. The reverse direction — promoting a namespace to a neutral option — needs the admission test passed and is a new milestone |
| `WithModel` inside `NewRequest` surprises a reader | Medium | Low | § 4.1 — pinned by test, documented on the constructor |
| The milestone busts the review budget | **Certain** | Medium | Forecast in § 9 and in `tasks.md`; `exception-ok` accepted up front; leaf boundaries are commit boundaries. Re-verification moved the forecast up by ~55 lines and one file — `tasks.md` carries the revision |

## 12. Ready for proposal

**Yes.** Every question this milestone must answer is either decided here (§§ 4, 5), owned by a named sibling with a re-verification anchor (§ 6), or recorded as deliberately unbuilt (§ 9). No blocking unknown remains.
