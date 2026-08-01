# Explore — request extension points

> **Change**: `cachicamas-ai-request-extension-points`
> **Milestone**: AI-12 — Add per-request options, the escape hatch, and rebuild
> **Nodes**: AI-12.1 … AI-12.4, all `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-01
> **Worktree**: `cachicamas-worktrees/ai-12` · **Branch**: `feat/ai-12-request-extension-points` (based on Wave 1 head `07d2027`)
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

**Both are running right now, in other worktrees, and neither has landed.** Every dependency on them is stated below as an assumption with the exact file and symbol to re-check. Nothing in this plan invents either surface.

### 6.1 AI-10 is being finished in `../ai-wave-1`

The Wave 1 head this worktree is based on (`07d2027`) carries **AI-10.1 and AI-10.2 only**. Remaining there: AI-10.3 items 2–4, AI-10.4, AI-10.5, AI-10.6.

| Assumption | File · symbol | Why AI-12 depends on it | If it lands differently |
| --- | --- | --- | --- |
| The regions `tools` and `toolChoice` reach the request through **`RequestOption`** constructors (expected `WithTools`, `WithToolChoice`) rather than as new `NewRequest` parameters | `backend/agent/src/ai/request.go` · `WithTools`, `WithToolChoice`, `requestDraft` fields | AI-12.1 item 2's totality is satisfied for those two regions *for free* if they are options | If they become parameters, AI-12.1 must add `WithTools`/`WithToolChoice` itself — one more option each, same pattern as `WithModel` |
| AI-10.3's cross-region rules (role/kind table, orphan results, duplicate call identities) are expressed as `Rule` values inside `NewRequest`'s `FirstFailure(...)` call | `backend/agent/src/ai/request.go` · `NewRequest` | AI-12.1 **extracts that rule slice** into a draft-scoped method so `With` re-runs it unchanged | If a rule closes over the `messages` **parameter** rather than over draft state, the extraction must move that read to `draft.messages` — mechanical, but it is a diff in AI-10.3's code and must be called out in the PR |
| The documented rule **order** is a slice, and appending a row is one line | `backend/agent/src/ai/request.go` · the `FirstFailure` argument list; AI-10 `design.md` § 4 | AI-12.3 appends the extension rule as the **last** row; AI-12.2 item 2 asserts derive-time order equals construction-time order | If the order becomes anything other than a literal slice, AI-12.2's position-equality assertions need a different anchor |
| Equality semantics are **region-wise readback equality with message identity excluded**, and `func (r Request) Equal(other Request) bool` **is exported** (AI-10.6; recorded default: *yes*) | `backend/agent/src/ai/request.go` · `Request.Equal`; AI-10 `design.md` § 11.2 | AI-12.1 item 1 ("the original is observably unmodified, deep comparison before and after") and AI-12.3 item 3 (equality inertness) both name it | If `Equal` is **not** exported, AI-12's tests compare region-by-region locally in `ai_test` and the milestone must not export one of its own — that would be a second equality, which AI-10.6 owns |
| If `Equal` **is** exported, AI-12.3 **extends it** to cover the extension region | `backend/agent/src/ai/request.go` · `Request.Equal` | An `Equal` that ignored extensions would let a rebuild silently drop them and still pass AI-10.5's round trip | Stated as a requirement in `spec.md`, not left to the implementer |
| `Reasoning.Token()` keeps `([]byte, bool)` and its byte-exactness | `backend/agent/src/ai/reasoning_content.go` · `Reasoning.Token` | AI-12.1 item 3 is a **pin** over exactly that property, extended across the rebuild path | Landed and stable at `07d2027`; low risk |

**Rebase discipline**: this worktree branches from `07d2027`. The apply agent re-verifies every row above against the *then-current* `../ai-wave-1` head before writing its first test, and rebases rather than merging.

### 6.2 AI-11 is being planned in `../ai-11`

`cachicamas-ai-cache-breakpoints` defines `V-REQ-23` **cache-boundary marker**, `V-REQ-24` **breakpoint cap** and `V-REQ-25` **invalidation cascade**. Its surface does not exist yet, in any form.

AI-12.1 item 2 names **markers** in its region list. This plan therefore states the requirement and *not* the spelling:

> **Markers are one more region reachable by the rebuild.** Whatever value carries a marker must be replaceable through the rebuild path without a new mechanism invented after the fact.

The likely resolution — and the reason the requirement is satisfiable without knowing AI-11's answer — is AI-10 `design.md` § 12.1, which already records where markers will attach:

> A marker attaches to the things that are already ordered and individually addressable: `Segment`, `Tool` and `Message`. […] `Segment` is a struct with one unexported field, not a string alias. Adding `marked bool` is a field, not a type change, and no signature moves.

If markers are **fields on `Segment`, `Tool` and `Message`**, they are reachable transitively the moment `WithSystemInstruction`, `WithTools` and `WithMessages` exist — AI-12 adds nothing. If AI-11 instead models markers as a **request-level region** (for example an ordinal-keyed list, which the breakpoint cap `V-REQ-24` might argue for), AI-12.1 owes one more `RequestOption` for it.

| Assumption | File · symbol to re-check | Branch |
| --- | --- | --- |
| Markers are carried **on** `Segment`, `Tool` and `Message`, not as a request-level region | `backend/agent/src/ai/system_instruction.go` · `Segment`; `tool.go` · `Tool`; `message.go` · `Message`; AI-11's `design.md` | **Reachable transitively** — AI-12.1 asserts it, adds no option |
| Otherwise: markers are a request-level region | AI-11's `design.md`; whatever `With…` constructor it names | AI-12.1 appends one option and one totality assertion; this is a *discovered case appended* to AI-12.1's test list, never a substitution |
| The breakpoint cap `V-REQ-24` is a `Rule` in the request's rule slice | `backend/agent/src/ai/request.go` · the `FirstFailure` argument list | The rebuild re-runs it automatically — a derived request that exceeds the cap must fail at derive time exactly as at construction |

**Neither sibling's surface is invented here.** Where this plan must name a symbol that does not exist, it names it as an assumption in this section and nowhere else.

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

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| AI-10.3/AI-10.6 land a shape this plan assumed differently | **High** — they are being written concurrently | Medium | § 6.1 states every assumption with its file and symbol; the apply agent re-verifies before its first test and rebases |
| AI-11's marker surface forces a request-level region | Medium | Low | § 6.2 states both branches; the extra option is one constructor and one assertion, appended not substituted |
| The escape hatch becomes a dumping ground for things that should be neutral | Medium | **High** | `V-REQ-26`'s admission test is the gate, and AI-10 `design.md` § 8.2 already lists which candidates are the hatch's. The reverse direction — promoting a namespace to a neutral option — needs the admission test passed and is a new milestone |
| `WithModel` inside `NewRequest` surprises a reader | Medium | Low | § 4.1 — pinned by test, documented on the constructor |
| Extraction of AI-10.3's rule slice produces a merge conflict in `request.go` | **High** | Medium | Expected and cheap: the conflict is in one function. Rebase, do not merge. Engram #2292's rule holds — read siblings' files, write only this milestone's concern, and `request.go` is jointly owned this once |
| Equality drifts if AI-10.6 does not export `Equal` | Medium | Medium | AI-12 must not export a second equality. `spec.md` states the obligation as conditional on AI-10.6's choice |
| The milestone busts the review budget | **Certain** | Medium | Forecast in § 9 and in `tasks.md`; `exception-ok` accepted up front; leaf boundaries are commit boundaries |

## 12. Ready for proposal

**Yes.** Every question this milestone must answer is either decided here (§§ 4, 5), owned by a named sibling with a re-verification anchor (§ 6), or recorded as deliberately unbuilt (§ 9). No blocking unknown remains.
