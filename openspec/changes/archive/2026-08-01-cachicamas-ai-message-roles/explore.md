# Explore — roles and message identity

> **Change**: `cachicamas-ai-message-roles`
> **Milestone**: AI-05 — Define roles and message identity
> **Nodes**: AI-05.1 `[leaf]` · AI-05.2 `[leaf]` · AI-05.3 `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [the register](../../../specs/ai-contract-vocabulary/spec.md) · [AI-04's decision](../2026-08-01-cachicamas-ai-validation-errors/decision.md) · [doc 0001 — agent stack v2](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0005](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Predecessors**: AI-00 … AI-04
> **Blocks**: AI-06 … AI-10

---

## 1. What this milestone is, in one paragraph

AI-05 defines the smallest addressable unit of a transcript: **one role plus ordered content, with a stable identity, that a caller cannot mutate from outside.** It is the first Layer 1 milestone that defines a *domain* contract rather than the machinery around one, and it is the first consumer of AI-04's taxonomy — its charter's "fails with an AI-04 sentinel" appears twice in the test lists. It has three behavior leaves and no `[decision]` node, which means every choice it makes is made in `design.md` and is answerable to a test.

## 2. What already exists, and what it settles

### 2.1 The code

`backend/agent/src/ai/` holds `doc.go` (the package comment), `import_boundary_test.go` (AI-00's two guards) and AI-04's `validation.go` / `validation_test.go`. `go.mod` carries zero requires and two landed tests hold it there. AI-05 is therefore stdlib-only.

AI-04's surface is the whole of what AI-05 needs to report a broken rule:

| AI-04 surface | How AI-05 uses it |
| --- | --- |
| `ErrNotInVocabulary` | A role outside the closed vocabulary (AI-05.1 item 2) |
| `ErrEmpty` | A message with no content, and an empty string offered to the role parser (AI-05.2 item 3) |
| `At("role")`, `At("content")` | The two positions this milestone can be at |
| `Rule`, `FirstFailure` | The documented order in which the two rules are checked |
| `Invalid(rule, at…)` | The one failure value |

AI-04's `tasks.md` § Next already wrote the sentence this milestone has to satisfy: *"`ErrNotInVocabulary` at `At("role")` and `ErrEmpty` at `At("content")`, composed through `FirstFailure`."* If AI-05 needs a sentinel that is not in the landed set, that is news, and the append discipline — not a local `errors.New` — is the response.

### 2.2 The vocabulary — three terms AI-05 implements rather than invents

| Id | Term | The clause that binds AI-05 |
| --- | --- | --- |
| `V-REQ-01` | role | "The **closed**, provider-neutral vocabulary value naming who a message is attributed to. Closed, not advisory: a value outside the vocabulary is a caller-contract failure, not an unknown passed through. **Not a provider's own role string** — an adapter maps between them." |
| `V-REQ-02` | message | "The smallest addressable unit of a transcript: **one role plus ordered content.** Layer 1's unit of attribution and ordering *within* a request. Not a turn, not a transcript, not a history." |
| `V-REQ-03` | message identity | "The **stable handle** by which one message is distinguished from another **within a request**. Identity is Layer 1's; correlating identity across turns or across a persisted session is not." |

Two further rows constrain the milestone without being owned by it. `V-REQ-04` **content part** is the thing a message's ordered content is made of, and it is **AI-06's**, with two constitutive properties AI-05 must not settle. `V-OUT-02` **transcript** names the boundary from the other side: AI-05 owns the unit, "never the collection, its ordering across turns, or its repair."

### 2.3 What the register does *not* settle, and this milestone must

The register defines what a role *is*. It never enumerates the vocabulary. Neither does doc 0002, which is bound by its own authoring constraint not to invent names. So **which roles exist is AI-05's**, and it is the one substantive choice in the milestone that no test list phrases for it. § 3.1 works it.

## 3. The real design tensions

Four. Three are genuinely open; the fourth is the milestone's structural hazard.

### 3.1 Which roles the closed vocabulary contains

Two of the four candidates are not in question. `user` and `assistant` are cited directly: doc 0001 § 3.3 row 5 ("only one provider enforces strict user/assistant alternation"), row 4 ("a block inside a **user-role** message"), and doc 0002 AI-10.1 item 1 ("one **user** text message"). A `tool` role is cited by doc 0002 AI-10.3 item 3, which asks AI-10 to decide "whether a tool-result part may appear in a **non-tool role**" — a question that presupposes a tool role exists.

`system` is the open one, and the evidence points the other way. `V-REQ-19` **system-instruction segment** makes the system instruction its own region of the request, owned by AI-10.2 and **segmented from birth**; doc 0001 § 3.3 row 8 records that the three provider shapes for it are a top-level field, a differently-named top-level field, or a nested object — and that "each adapter renders them into its own shape". Nothing in doc 0001, doc 0002 or the register asks for a system-role *message*. A vocabulary that carries one would give Layer 1 two ways to express one fact, which is the drift `design.md` of AI-04 named as the reason it refused a `Kind` field beside its sentinel.

The counter-argument is real and should be stated: every vendor API a reader has met has a `system` role, so its absence reads as an oversight rather than a decision. The answer is that the neutral vocabulary is not a union of vendor vocabularies — `V-REQ-01` says so in its own last clause — and that adding a constant later is additive while removing one before AI-40's freeze is not free. AI-04 set the precedent explicitly: *"A sixth class for conflicting values is plausible and has no citable case; it is not landed."*

### 3.2 Identity: minted or supplied

`V-REQ-03` calls identity "the stable handle by which one message is distinguished from another." Distinguished implies distinct, and distinctness is not a property a caller-supplied handle has — two messages built from the same caller-supplied string are indistinguishable, and the type has no way to notice.

Supplied identity is the friendlier option for Layer 2, which owns the transcript and may already have keys of its own. It is also the option that pushes uniqueness into a rule someone else has to write, at request scope, in AI-10. Minted identity makes distinctness structural and makes AI-05.2 item 1 a real assertion rather than a tautology: an identity that is *computed on read* — from content, say — would satisfy "comparable" and fail "does not change across reads", and a minted one cannot.

The minting mechanism carries a warning from the retired plan. Defect **C3** was a process-global counter, and its lesson is written into `V-STR-13`: *"the fix is not a smaller counter; it is putting the counter where the stream is."* That lesson applies to a contract that says something about a *value* — "every stream's first event carries 1". Message identity says nothing about its value, only that two messages differ, so the shape of the counter is not observable. The distinction has to be argued, not assumed, or a reviewer is right to flag it.

### 3.3 Copy semantics is where the confusing failures live

doc 0002 attaches an unusual note to AI-05.3: it "is the one whose absence produces the most confusing class of test failure in a streaming package", and it is what lets AI-21.6's request capture assert on history without defensive copying at the call site. The hazard is specific and it is Go's, not the design's: `NewMessage(role, parts...)` called as `NewMessage(role, parts...)` with a slice spread passes the caller's **backing array**, not a copy. Nothing in the language warns about it, the happy-path tests all pass, and the failure appears later as a message whose content changed without anyone writing to it.

Both directions need proving, and they are different mechanisms: construction must copy *in*, and every read must copy *out*. A design that does one and not the other passes half the tests and produces the same confusing failure from the other side.

### 3.4 The content-part problem — the structural hazard

**AI-05 needs ordered content. AI-06, which decides what a content part is, has not happened.** This is deliberate in doc 0002's ordering, and it is the milestone's one real risk: the cheapest way to get AI-05.2's ordering test to compile is to define a content type, and any content type AI-05 defines takes AI-06.1's decision away from it.

AI-06.1's decision is precise, and it is worth quoting so the seam can be checked against it rather than against a paraphrase:

> A single strategy is chosen that simultaneously satisfies: **(a)** an adapter in another package can read a part's payload out of a constructed message, and **(b)** no value that skipped the constructor — zero value, hand-rolled implementation, or struct literal — can validate. The artifact demonstrates both properties against the chosen strategy **before any code exists**.

Two properties, in tension, and the phrase "before any code exists" is the one AI-05 can break by accident. What AI-05 needs from content is much smaller than what AI-06 decides about it:

| AI-05 needs | AI-06 decides |
| --- | --- |
| A named type the message's ordered content is a sequence *of* | What a part carries, and how the kind derives from the payload |
| That two elements can be told apart, so order is observable | How a payload is read from another package (`V-REQ-06`) |
| That a sequence can hold the same element twice | Whether an unconstructed value validates, and by which mechanism (`V-REQ-07`) |
| Nothing about payloads, kinds, accessors or construction rules | The registration procedure and its guard (AI-06.4) |

The right-hand column is untouchable. The left-hand column is satisfiable by a **named marker interface with no readable surface at all** — a type that says "this is one element of a message's content" and says nothing else. § 4 of `design.md` works the shape; the point here is that the two columns do not overlap, which is why this is a seam and not a plan defect. A design that put a payload, a kind, an accessor or a constructor on the left-hand side would be deciding AI-06.1 from outside its milestone, and the revert-and-record clause would be the correct response.

One consequence is worth naming now rather than letting AI-06 discover it: a marker interface whose only method is unexported is **not** a seal. Go lets a type in another package satisfy it by embedding the interface, and AI-05's own external tests do exactly that. That bypass is precisely what AI-06.3 item 2 exists to close, and AI-05 leaves it open on purpose — closing it would be choosing the compile-time answer to a question doc 0002 assigns to AI-06.1.

## 4. Prior art worth borrowing, and the parts worth refusing

**The standard library's closed vocabularies.** `time.Month`, `time.Weekday`, `reflect.Kind` and `http.StateNew` are all the same shape: an integer defined type, `iota`-declared constants, a `String()` method backed by an **indexed table**, and — for `Month` and `Weekday` — a rendering for an out-of-range value that is deliberately ugly (`%!Month(42)`). The shape is worth taking wholesale: it is comparable, it is cheap, the table is a slice rather than a map, and an out-of-range value renders as something no parser will accept back.

**The part worth refusing** is the permissiveness. `time.Month(42)` is a legal value that renders and is never rejected, because `time` has no construction boundary to reject it at. `V-REQ-01` requires the opposite: closed, not advisory. So the table is borrowed and the tolerance is not — the vocabulary is checked at the one place a role enters a contract.

**`database/sql/driver`'s marker interfaces and `ast.Node`.** Both use a method the caller does not implement to name a position in a contract rather than to expose behavior. That is exactly the seam AI-05 needs, and the fact that the standard library reaches for it when a type must be *named* before it is *defined* is the strongest available signal that a marker is not a workaround.

**Opaque identifier types.** `sync.Once`, `netip.Addr` and `time.Time` all keep an unexported field and expose comparison and rendering only. An identity that is a bare `string` or `uint64` is one the caller can forge, compute or reuse; an opaque comparable struct is one that only the constructor can mint, which is the property `V-REQ-03` is asking for without saying so.

## 5. What the milestone must not do

| Excluded | Owner | Why not here |
| --- | --- | --- |
| What a content part **is** — payload, kind, accessors, construction rules, sealing | **AI-06** | The keystone of wave 1. AI-06.1 decides readability and zero-value-invalidity **together, before any code exists**; a payload landed here would make that sentence false |
| Whether a given role may carry a given content kind | **AI-10.3** | Stated as AI-05.1's out-of-scope clause. It is a request-level rule and it needs kinds, which do not exist |
| Whether a tool-result part may appear in a non-tool role | **AI-10.3** | Same rule, named explicitly by doc 0002 |
| Message *collections*, their order, and their repair | **AI-10** / Layer 2 | `V-OUT-02`: Layer 1 owns the unit, never the collection |
| Uniqueness of identity *across* a request | **AI-10** | AI-05 makes two messages distinguishable; that a request holds no duplicate is a composition rule at request scope |
| Rejecting a nil or unconstructed content element | **AI-06.3** | Its item 1 is exactly this assertion, and it needs AI-06.1's strategy to know what "unconstructed" means |
| Equality of two messages | **AI-10.6** item 3 | "the documented equality" is named there, and it is defined over a request's regions |
| A system-role message | **the milestone that meets the case** | § 3.1. No citable case; the vocabulary is append-only and adding a constant is additive |
| Merging consecutive same-role messages | **AI-26.x** (adapter) | doc 0001 § 3.3 row 5 marks it adapter-local |

## 6. Vocabulary check

Every Layer 1 noun this change uses normatively resolves to a register row: `V-REQ-01` role, `V-REQ-02` message, `V-REQ-03` message identity, `V-REQ-04` content part (cited, not defined — AI-06 owns it), `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-16`, `V-FAIL-17` from AI-04, `V-REQ-22` request validation, `V-OUT-02` transcript.

**No append is required**, and the near-misses are worth recording so the next reader does not re-derive them:

- *"copy-on-construct"* is a **property** of `V-REQ-02` message, not a noun with an owner. AI-10.6 states the same property at request scope and calls it immutability without appending a term either. Register § 9 rule 2 governs nouns; a property proven by a test is not one.
- *"role vocabulary"* is `V-REQ-01`'s own word — its definition opens "The closed, provider-neutral **vocabulary** value". Appending a second term for the set a value is drawn from would be the parallel-vocabulary defect `V-FAIL-03` warns about, one category over.
- *"content seam"* is a `design.md` word for how a message holds `V-REQ-04` content parts before AI-06 defines them. It names an implementation position, not a Layer 1 concept, and it will not survive AI-06. A register row for it would outlive the thing it names.

## 7. Size and shape

doc 0002's budget is "prefer less than 250 changed lines; stop and reassess before 400". The expected Go footprint is **two production files and two test files**: the closed vocabulary (AI-05.1) and the message (AI-05.2, AI-05.3). The split is along the leaf boundary, which doc 0002's split trigger 4 also names as the PR-chain boundary, so the commits fall where a reviewer would cut them.

The forecast is that the budget is exceeded, for the same reason AI-04 exceeded it: the deliverable is a set of properties five later milestones stand on, and the tests are the deliverable's larger half. `tasks.md` carries the reassessment rather than a silent overrun.

No split trigger fires on the nodes themselves: three leaves, test lists of 4, 3 and 2 items against a limit of 7; strictly ordered, so two agents could not work them concurrently; and — per § 3.4 — none of them needs a seam that does not exist.

## 8. Open questions carried into the proposal

1. Does the role vocabulary need `system`? *Carried: no, on § 3.1's evidence, with the append path recorded and the counter-argument stated.*
2. Is minted identity opaque, or does it render? *Carried: opaque and comparable, with a rendering for diagnostics only — a caller that parses the rendering is doing what `V-FAIL-06` records as the failure mode of an open classification.*
3. Should `NewMessage` reject a nil content element? *Carried: no. AI-06.3 item 1 is that assertion, and it cannot be written before AI-06.1 defines what an unconstructed part is.*
4. Does the seam need a rendering, a length, or a comparison? *Carried: no. Each is a readable surface, and `V-REQ-06` readability is AI-06's constitutive property to decide.*
