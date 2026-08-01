# Decision — one content-part strategy

> **Change**: `cachicamas-ai-content-parts`
> **Milestone**: AI-06 — Define content parts: readable and sealed
> **Node**: **AI-06.1 `[decision]`** — the keystone of Wave 1
> **Phase**: decision
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Closes**: doc 0002 AI-06.1's four-item checklist
> **Binding inputs**: [doc 0002 § AI-06](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 § 3.1 **C1**, **C2**](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [the register](../../specs/ai-contract-vocabulary/spec.md) `V-REQ-04` … `V-REQ-08` · [AI-04's decision](../cachicamas-ai-validation-errors/decision.md) · [AI-05's design § 4](../cachicamas-ai-message-roles/design.md)
> **Status**: decided. Every later content variant — AI-07 reasoning, AI-09 tool call and tool result — inherits this document instead of re-deriving it.

---

## 1. What is being decided, and what is not

Four questions, from doc 0002's closing checklist:

1. **One strategy** that simultaneously satisfies **(a)** an adapter in another package can read a part's payload out of a constructed message, and **(b)** no value that skipped the constructor — zero value, hand-rolled implementation, struct literal — can validate. Both properties demonstrated against the strategy **before any code exists**.
2. The relationship between discriminator and payload: the kind is **derived from** the payload rather than set alongside it.
3. The procedure for adding a kind later — registration, accessor, construction rule, and the guard that fails when one of the three is missing.
4. The accessor shape, decided **once for all variants**: how a caller discovers a part's kind and obtains the typed payload **without a type switch over unexported types**.

Not decided here: what reasoning content is (AI-07), what a tool call carries (AI-09), where the request runs its single validation pass (AI-10), or how any of it renders on a wire (AI-24 onward). Decided here and inherited by all of them: **the shape a content part has.**

The decision is written before the code, and § 3 is the demonstration doc 0002's item 1 requires. Nothing below depends on an implementation detail discovered while writing one.

---

## 2. The observation the decision turns on

Doc 0002 states the tension as a property of the problem:

> The two properties are in tension: a part whose payload is readable from another package is usually one that can also be BUILT there, which is exactly the bypass to avoid.

**It is a property of interfaces, not of the problem.**

For an interface, *readable* and *implementable* are the same word: both are **methods**. Every method exported to make a payload readable is a method someone else can supply. Sealing with an unexported method removes implementation and does not remove satisfaction — AI-05 landed the proof in four lines, and its own GoDoc records it:

```go
type part struct {
	ai.Content        // embedded, nil; promotes the unexported method
	label      string
}
```

That value satisfies `ai.Content`, carries a nil embedded interface, and panics on the first method call. So an interface seam can be readable, or sealed, or satisfiable-and-broken — never cleanly the first two at once.

For a **struct**, they are different words. Reading is **methods**. Constructing is **fields**. Go forbids naming an unexported field in a composite literal from another package, and forbids a positional literal for any type that has one. A struct whose every field is unexported is therefore **externally constructible only as its zero value**, while every exported method on it stays callable from anywhere.

The tension dissolves the moment the seam stops being an interface. That is the whole decision; everything below is its consequences.

---

## 3. Decision 1 — the strategy: one exported opaque value type

### 3.1 The shape

```go
// package ai

type Part struct{ payload partPayload }          // one unexported field

type partPayload interface {                     // unexported; unexported methods
	kind() PartKind
	validate(at Path) *Violation
}

type PartKind uint8                              // closed vocabulary, iota + 1
const PartKindText PartKind = iota + 1

type textPayload struct{ text string }           // unexported concrete payload

func NewText(text string) (Part, error)          // the only door in
func (p Part) Kind() PartKind                    // derived: p.payload.kind()
func (p Part) Text() (string, bool)              // the exported payload accessor
```

`Message` holds `[]Part`. The `Content` interface AI-05 landed as an explicitly open seam is **removed** in the same change — that removal is the closure AI-05's `design.md` § 4.2 handed to this milestone.

### 3.2 Property (a) — readable from another package, demonstrated

An adapter in `package anthropic` — or, today, a test in `package agenttest_test` — writes this and nothing else:

```go
part, err := ai.NewText("hello")                 // constructed here
msg, err := ai.NewMessage(ai.RoleUser, part)     // placed in a message

for _, got := range msg.Content() {              // read back out
	switch got.Kind() {
	case ai.PartKindText:
		text, ok := got.Text()                   // the exact bytes, no re-encoding
		_ = text
		_ = ok
	}
}
```

Every identifier on that path is exported: `Part`, `NewText`, `Message.Content`, `Part.Kind`, `PartKindText`, `Part.Text`. Nothing is asserted against an unexported type; nothing needs a helper inside `package ai`. `Text()` returns the stored `string` by value, so the bytes a caller reads are the bytes it supplied — `V-REQ-06` satisfied, **C2 structurally impossible**, and AI-26's request translation writes exactly the loop above.

### 3.3 Property (b) — sealed, demonstrated bypass by bypass

Doc 0002 names three bypasses. Each is worked below against the § 3.1 shape, and the mechanism that stops it is named.

| # | Bypass attempt, written from another package | Outcome | Mechanism |
| --- | --- | --- | --- |
| 1 | `var p ai.Part` — the zero value | Compiles; **validation rejects it** | `p.payload` is nil, so `p.Kind()` is `PartKind(0)`, which is not a member of the closed kind vocabulary. `ErrNotInVocabulary` at `content[i]` |
| 2 | `ai.Part{payload: somethingIWrote}` | **Compile error**: `unknown field payload in struct literal` / `implicit assignment to unexported field` | The single field is unexported. Go forbids naming it from another package |
| 3 | `ai.Part{myPayload}` — positional literal | **Compile error**: `implicit assignment of unexported field 'payload'` | Same language rule, positional form |
| 4 | `type mine struct{ ai.Part }` — the AI-05 embedding bypass | **Compile error at the message boundary**: `cannot use mine{} (value of type mine) as ai.Part value` | `Message` holds `[]Part`, a concrete type. Embedding produces a *different* type. There is no interface left to satisfy |
| 5 | `type mine struct{ ai.Part }`, then offering `mine{}.Part` | Compiles; **validation rejects it** | The promoted field is the zero `Part`, which is bypass 1 |
| 6 | Implementing the part contract | **There is nothing to implement.** `Part` is a struct; `partPayload` is unexported with unexported methods | An unexported interface with unexported methods cannot be named, embedded or implemented outside `package ai` |
| 7 | Reaching a payload by reflection and writing it | `reflect` refuses to `Set` an unexported field (`CanSet` is false) | The language's own rule, not a check this package wrote |
| 8 | Copying a valid part and mutating it | `Part` is a value; `textPayload` is a value struct holding a `string`; a `string` is immutable | Copying yields an equal, equally valid part. There is no mutable state to reach |

**Two mechanisms, in complementary halves.** The compiler prevents bypasses 2, 3, 4, 6 and 7 — nothing external can *assemble* a part. Validation rejects bypasses 1 and 5 — the one value external code can still produce, the zero value, is not a member of the kind vocabulary. Doc 0002's AI-06.3 item 2 asks for "either the compiler or validation, whichever the strategy chose"; this strategy chose **both, with a clean boundary between them**, and AI-06.3 proves each half separately.

`V-REQ-07`'s three clauses, checked: *cannot be valid* — bypass 1 and 5; *cannot reach a message* — bypass 4 makes the type unofferable, and 1/5 make the offer fail; *cannot satisfy the part contract by accident* — bypass 6, there is no contract to satisfy.

### 3.4 Why both properties hold at once, stated as one sentence

**A struct can export behavior without exporting construction; an interface cannot.** Readability is delivered by exported methods on `Part`; sealing is delivered by the unexported field those methods read. The two surfaces are disjoint, so widening one does not widen the other — which is the exact failure mode that produced **C1** and **C2** as separate defects of one contract.

---

## 4. The alternatives, at full strength

Each is stated as its proponent would state it, then judged. Three of the five were shipped by the retired design or by AI-05, so none is a straw man.

### 4.1 Alternative A — exported interface with exported concrete variants

```go
type Part interface{ Kind() PartKind }
type TextPart struct{ Text string }
func (TextPart) Kind() PartKind { return PartKindText }
```

**At full strength.** This is how `go/ast`, `database/sql/driver` and most idiomatic Go sum types are written. It is maximally readable: `p.(TextPart).Text` needs no accessor and no `ok` dance. Each variant carries exactly its own fields, so a text part costs one string and a tool call costs no unused text field. Type switches are the language's native tool for a sum, and `go vet`-adjacent linters can even check them for exhaustiveness. Adding a kind is one new type and one method — no registry, no guard, no AI-06.4. A reviewer who has written Go for a decade will reach for this first, and be right about every project that does not have `V-REQ-07`.

**Why it loses.** `TextPart{}` is a valid `Part` whose payload is the empty string, and `TextPart{Text: strings.Repeat("x", 1<<30)}` is a valid `Part` that skipped the length rule. That is **doc 0001 C1, verbatim**: *"the exported text value type satisfies the part interface directly, so its zero value is a valid part that passes message validation and bypasses every construction rule."* The repairs all fail:

- *Unexport the fields.* Then the payload is unreadable — C2 — unless accessors are added, at which point the variant is an opaque struct and the interface is doing nothing but permitting hand-rolled implementations.
- *Seal the interface with an unexported method.* Embedding still satisfies it (§ 2), so every boundary must type-assert against the exported concrete types anyway — and their zero values are still valid. The seal moves the defect, it does not remove it.
- *Validate at the message boundary.* The zero `TextPart` is indistinguishable from a legitimately-constructed part carrying an empty string, because there is no field that records "this passed the rules" that a literal cannot also set.

This alternative is the reason the milestone exists. It is rejected.

### 4.2 Alternative B — unexported wrapper, only the discriminator exported

The retired design's shape for text, tool call and tool result.

**At full strength.** It is perfectly sealed and needs no runtime check to be so: nothing outside the package can name the wrapper type, so nothing outside can build one. It is the smallest possible exported surface — one interface and one enum — which means the smallest possible frozen API at the v1 freeze (AI-38). Every construction rule is unavoidable because the constructor is the only exported symbol that yields a value. If sealing were the only requirement, this would win outright.

**Why it loses.** `V-REQ-06`. Doc 0001 records the cost in the strongest language in the document: *"Content cannot be read back out of a request from another package, which makes request translation structurally impossible"* — and doc 0002 marks AI-06 a **hard blocker on AI-26** because of it. An adapter cannot render what it cannot read. Rejected by the register.

### 4.3 Alternative D — sealed interface plus package-level accessor functions

```go
type Content interface{ isContent() }
func TextOf(c Content) (string, bool)
```

**At full strength.** This is the minimal change from what AI-05 already landed: the seam stays exactly as it is, one file is added, and no existing test moves. It reads well at a call site (`if text, ok := ai.TextOf(c); ok {…}`), it keeps every variant's concrete type unexported so no zero value of an exported type is a part, and it is genuinely extensible — AI-07 and AI-09 each add one function. It also *does* satisfy both properties in a narrow reading: payloads are readable through the functions, and unexported concrete types cannot be built outside.

**Why it loses.** Three reasons, in increasing order of weight.

1. **The seam stays satisfiable.** `Message` still holds `[]Content`, so the embedding bypass still compiles, still produces a nil-embedded value, and still has to be rejected at every boundary by a type assertion. The compiler is doing none of the work; a boundary that forgets the assertion is a C1 in the shape of an omission. Compare bypass 4 in § 3.3, where the same attempt does not compile.
2. **It grows a second vocabulary.** Every kind adds a package-level function whose name has to be kept in sync with a kind constant and a concrete type, with nothing binding the three. AI-06.4's guard would have to scan free functions by name convention.
3. **It reads the payload away from the value.** `ai.TextOf(c)` and `c.Kind()` are two different call shapes for one question. Doc 0002 item 4 asks for *one* accessor shape for all variants; a method on the value is that shape, a package function is a second one.

Rejected — but it is the closest runner-up, and if `Part` ever needed to be an interface for a reason not yet visible, this is the shape it would take.

### 4.4 Alternative E — generic `Part[T]`

**At full strength.** It is the only candidate with **compile-time** payload typing: `p.Payload()` returns `T`, with no `ok` bool and no possibility of asking a text part for a tool call. The discriminator becomes redundant, so item 2's disagreement risk disappears by construction rather than by discipline. Go 1.26 has had generics for years; this is not exotic.

**Why it loses.** A message holds *heterogeneous* content — `V-REQ-02` is "one role plus ordered content", and that content is a text part next to a tool call. `[]Part[T]` cannot hold both. The only repairs are an erasing interface (which is Alternative A or D wearing a type parameter) or making `Message` itself generic — which AI-05's `design.md` § 4.4 already rejected as **viral**: "AI-10's request, AI-14's events and the provider interface would all carry the parameter". Rejected for the same reason, one milestone later.

### 4.5 Alternative F — exported struct with exported fields and a validity flag

**At full strength.** The simplest thing that could possibly work, and the shape of `database/sql.NullString`. Fully readable with no accessor at all. A `valid bool` unexported field means a literal cannot claim validity.

**Why it loses.** The exported payload fields make `Part{Text: "…"}` writable from anywhere; only the flag stops it validating, so the seal rests entirely on one boolean that the type's own package must remember to set. Worse, it makes the *payload* mutable after construction: a consumer holding a part can rewrite its text, and `V-REQ-07`'s "cannot be valid" says nothing about a part that was valid and then changed. Rejected.

### 4.6 The comparison, collected

| | A — exported iface + variants | B — unexported wrapper | **C — opaque struct** | D — sealed iface + funcs | E — generic | F — flag |
| --- | --- | --- | --- | --- | --- | --- |
| `V-REQ-06` readable | yes | **no (C2)** | **yes** | yes | yes | yes |
| `V-REQ-07` sealed | **no (C1)** | yes | **yes** | at each boundary, by assertion | yes | by one boolean |
| Kind derived from payload | by method, per variant | yes | **yes** | yes | redundant | no |
| Heterogeneous content in one slice | yes | yes | **yes** | yes | **no** | yes |
| Compiler blocks hand-rolling | no | yes | **yes** | no | yes | no |
| One accessor shape for all kinds | type switch | none | **method on `Part`** | free functions | `Payload()` | fields |
| Later variants add | a type | a type | **a payload + 3 edits** | a type + a func | — | a field |

---

## 5. Decision 2 — the kind is derived from the payload

`Part` stores **no kind field**. `Kind()` is:

```go
func (p Part) Kind() PartKind {
	if p.payload == nil {
		return 0
	}
	return p.payload.kind()
}
```

Three consequences, each of which is the point:

1. **The two cannot disagree**, because there is only one of them. A `Part{kind: PartKindText, payload: toolCallPayload{…}}` is not a state this type can be in, so no consumer ever has to decide which to believe and no test has to assert they agree — though AI-06.2 item 2 asserts it anyway, as a pin against a future refactor that adds the field back.
2. **`kind()` is a method on the payload**, so declaring a payload type without giving it a kind is a compile error, not a registration someone forgot.
3. **A part with no payload has no kind**, which is what makes the zero value fail the vocabulary rule rather than needing a rule of its own. See § 7.

The trade is that `Kind()` costs an interface dispatch instead of a field read. `Part` is one interface word wide either way, and nothing in Layer 1 reads a kind in a loop that matters.

---

## 6. Decision 3 — the accessor shape, once, for every variant

### 6.1 The shape

Two exported members, and the second is repeated once per kind:

```go
func (p Part) Kind() PartKind          // discovers the kind
func (p Part) Text() (string, bool)    // AI-06: yields the typed payload
```

and, inherited unchanged by the milestones that follow:

```go
func (p Part) Reasoning() (Reasoning, bool)    // AI-07
func (p Part) ToolCall() (ToolCall, bool)      // AI-09
func (p Part) ToolResult() (ToolResult, bool)  // AI-09
```

A consumer switches on `Kind()` — an exported, comparable value from a closed exported vocabulary — and calls the matching accessor. The second result reports whether the part actually carries that payload, so an accessor called on the wrong kind is a `false`, never a panic and never a zero value that reads as data.

**There is no type switch, over unexported types or any other kind**, because there is only one type. Doc 0002 item 4's requirement is satisfied by removing the construct it constrains.

### 6.2 Why `(T, bool)` rather than the alternatives

| Alternative | Why not |
| --- | --- |
| `Text() string` alone | A text part and a tool-call part would both answer `""`, and a consumer could not tell "empty" from "not this kind". The `ok` bool is the same distinction `V-REQ-11` will need for "absent is not empty" |
| `Text() (string, error)` | Asking a tool-call part for text is not a caller-contract failure; it is a question with the answer "no". AI-04's taxonomy is for broken rules, and diluting it here would cost the property that a `*Violation` always means the caller has something to fix |
| `MustText() string` panicking | A panic reachable by asking the wrong question of a valid value |
| A free function `ai.TextOf(p)` | § 4.3 reason 3: two call shapes for one question |
| `Payload() any` plus a caller-side type assertion | The assertion target has to be exported for the caller to name it, which puts an exported payload type back in the API — and its zero value back in play |
| Generic `PayloadOf[T](p Part) (T, bool)` | Same exported-target problem, plus it moves the kind vocabulary into the type system where the guard cannot scan it |

### 6.3 Payload types stay unexported until a variant needs otherwise

`textPayload` is unexported and `Text()` returns a `string`, so AI-06 exports no payload type at all. AI-07 and AI-09 carry structured payloads and will export the payload *value* types (`Reasoning`, `ToolCall`) — as opaque structs with the same discipline, built only by `NewReasoning` / `NewToolCall`. The rule they inherit: **an exported payload type is an opaque value type with unexported fields, or it is a defect.** Exporting a payload type is not a widening of the seal, because the seal is on `Part`, whose field remains unexported.

---

## 7. Decision 4 — validation, and why no new sentinel

### 7.1 The rules, in their documented order

Message construction (`V-FAIL-04`, order is contract):

1. the role is a member of the role vocabulary — `ErrNotInVocabulary` at `role` *(AI-05)*
2. the content is not empty — `ErrEmpty` at `content` *(AI-05)*
3. **each content element is a constructed part, in index order** — at `content[i]` *(AI-06)*

Rule 3, per element, again in order:

1. the part has a payload — else `ErrNotInVocabulary` at `content[i]`
2. the payload's kind is registered — else `ErrNotInVocabulary` at `content[i]`
3. the payload satisfies its own rules — the kind's own class, at `content[i].<field>`

And the text payload's own rules, which are **the same code the constructor runs**:

1. the text has at least one non-whitespace character — else `ErrEmpty` at `…text`
2. the text is at most `MaxTextLen` bytes — else `ErrOutOfRange` at `…text`

### 7.2 One rule set, two callers

`NewText` and the validation path are not two implementations of the text rules; they are one function called from two places. The constructor builds the payload, runs the rules, and returns the zero `Part` on failure. The validator runs the same rules on a payload that reached a message some other way — which, from outside the package, is impossible, and from inside is exactly the mistake a future variant author will make.

That is why AI-06.4 can demand a **constructor with rules** and a **validation path** as two separate legs without demanding two rule sets: the legs are two entry points, and the guard proves each entry point is wired, not that each has its own copy of the rules. A milestone that let them drift would have re-created C1 from the other direction — a constructor that checks and a boundary that does not.

### 7.3 No new sentinel: an unconstructed part is a vocabulary failure

AI-04's `decision.md` § 8 left this open: *"AI-06 … still decides the part strategy; whether a rule class for 'an unconstructed value' is needed and must be appended."*

**It is not needed.** Because the kind is derived from the payload (§ 5), a part that skipped construction **has no kind**, and a value with no kind is a value outside a closed vocabulary — which is precisely what `ErrNotInVocabulary` names. AI-05 set the precedent one milestone earlier: `Role(0)`, the role nobody set, fails with `ErrNotInVocabulary` rather than with a sentinel of its own, and `role.go` records why (`iota + 1` makes "a field nobody set" indistinguishable from a wild value at the boundary). `PartKind` uses `iota + 1` for the same reason and inherits the same answer.

Adding `ErrUnconstructed` would give the package two ways to say one fact, and would put the *first* new rule class since AI-04 into the register for a case the existing set already covers. The register is unamended.

### 7.4 The position of an element failure

`AtIndex("content", i)`, exactly as AI-04's `AtIndex` GoDoc anticipated: *"This is how a position says which message, which content part, or which tool: by ordinal, not by name."* A text rule failure renders as `content[0].text`; an unconstructed element renders as `content[0]`. Neither reproduces a byte of the caller's text — the redaction posture costs nothing here because a position is built from names this package wrote.

---

## 8. Decision 5 — the procedure for adding a kind, and the guard that enforces it

Doc 0002 item 3 asks for the procedure and for the guard that fails when one of its parts is missing. The procedure is five steps; AI-06.4 mechanizes steps 1–5 as three legs plus a documentation claim.

| # | Step | What AI-06.4 asserts |
| --- | --- | --- |
| 1 | Declare the constant at the end of the `PartKind` block and move `partKindEnd` past it | The guard enumerates the **declared constant space**, not the name table, so a constant declared without a table entry is visible to the assertion that catches it — AI-05's `role.go` rule 2, inherited |
| 2 | Add the unexported payload type with `kind()` and `validate(at Path)` | Compile-enforced by `partPayload`; a payload without them cannot be stored |
| 3 | Add the exported constructor `New<Kind>` whose rules are the payload's | **Leg 1 — a constructor with rules**: the guard calls the kind's witness with an input the rules must reject and requires a `*Violation` carrying a registered rule class |
| 4 | Add the exported accessor `func (p Part) <Kind>() (T, bool)` | **Leg 2 — a payload accessor**: the guard reads a valid part of the kind and requires `ok`, and reads a part of a *different* kind through the same accessor and requires `!ok` |
| 5 | Add the name to `partKindNames` and to the guarded kind list in the `PartKind` GoDoc | **Leg 3 — a validation path**: the guard assembles a part of the kind that skipped its rules and requires message construction to reject it. **Plus the documentation claim**: an AST scan compares the GoDoc list to the table |

**Where the witnesses live.** In the guard's own test file, not in production. The registry that production carries is `partKindNames` — a slice indexed by the constant, exactly like AI-05's `roleNames`, and for AI-04's stated reason that a Layer 1 registry is never a map. Putting four closures per kind into production code so that a test can call them would be production carrying test scaffolding; putting them in the guard, keyed by the declared constant, gets the same bite — a kind declared without a witness entry fails the guard's first assertion — at no production cost.

**The bite proof** AI-06.4 requires: a scratch kind is declared with a table entry and a witness entry missing one leg, the guard is recorded failing, and the scratch kind is dropped. Recorded in `tasks.md`, twice: once for a missing leg, once for a documentation list that drifted from the table.

---

## 9. What every later variant inherits, in one table

Written so that AI-07, AI-09 and AI-10 can cite this row instead of re-deriving the answer.

| Question | Answer, decided here |
| --- | --- |
| What type does a variant produce? | `ai.Part`. There is exactly one part type in Layer 1 |
| How is the variant distinguished? | By its unexported payload type, which answers `kind()` |
| How does a consumer discover it? | `p.Kind()`, against an exported closed vocabulary |
| How does a consumer read it? | `p.<Kind>() (T, bool)` — a method on `Part`, one per kind |
| Where do the rules live? | On the payload's `validate(at Path)`, called by both `New<Kind>` and the message boundary |
| Which sentinels? | AI-04's five. No variant appends one without a case the five do not cover |
| Where does it register? | `partKindNames`, indexed by the constant, plus the guarded GoDoc list |
| What proves it? | An external-package round trip, a zero-value rejection, and a guard entry with three legs |
| May a payload type be exported? | Yes, when it carries structure — as an opaque value type with unexported fields and a package-owned constructor. Never with exported payload fields |

---

## 10. Conceded costs, collected

The strategy is not free, and a reviewer should see the bill before the code.

| Cost | Detail | Why it is acceptable |
| --- | --- | --- |
| **No type-level narrowing** | Every kind is the same Go type, so no signature can say "this function takes a text part". A caller narrows at runtime with `ok` | Layer 1's consumers are adapters that must handle *all* kinds of a message's content in one loop. A signature narrowed to one kind would be a signature the loop cannot call. This is the single largest concession and it is paid to Alternative A |
| **Exhaustiveness is not compiler-checked** | Adding a kind does not break a `switch` that forgot it | Go checks no sum's exhaustiveness for any of the six candidates. That is precisely why AI-06.4 is a `[guard]` and not a comment |
| **`Part` accumulates accessors** | After AI-09 it carries `Kind` plus four accessors, and after any image/audio producer, more | They are one-line methods with one GoDoc paragraph each, grouped in the file of the kind they read. The alternative is one exported type per kind, each with a zero value that is C1 |
| **An interface dispatch per `Kind()`** | Instead of a struct field read | Immeasurable at Layer 1's call rates, and it is what buys § 5's "cannot disagree" |
| **Three edits per new kind** | Constant, payload, constructor, accessor, registry line, GoDoc line | The guard makes the coupling mechanical rather than remembered, which is the only reason a five-step procedure is safe to hand to a later milestone |
| **AI-05's tests must migrate** | Their content helper is the embedding bypass this decision closes | It is the intended disposition of `R-AMSG-009`'s second scenario, which AI-05 wrote as a deliberate placeholder. The migration is ~40 lines and is in this change's diff |
| **`Content` is removed from the exported surface** | A name AI-05 exported one milestone ago disappears | Nothing outside the module imports it; ADR 0005's v1 freeze is AI-38, and doc 0002 explicitly permits a milestone to refine later names. Leaving it as an alias would leave the embedding bypass alive, which is the defect |

---

## 11. Register amendment required by this decision

**None.** `V-REQ-04` … `V-REQ-08` already define every noun used above, including both constitutive properties of a content part and the closed, exhaustively registered kind set. § 7.3 records why no rule class is appended to AI-04's five.

If a reviewer believes a term is missing, the two candidates considered and rejected were *"opaque value type"* (a Go implementation shape, not a Layer 1 concept — the register's own scope rule excludes it) and *"kind registration"* (`V-REQ-05`'s "closed and exhaustively registered" already carries it).

---

## 12. What this decision does not settle

- Whether a given **role** may carry a given kind — AI-10.3.
- Where the request's **single validation pass** runs — `V-REQ-22`, AI-10.4. This decision gives it the function to call and pins the call's position prefix; it does not decide where AI-10 calls it from.
- The **shape of structured payloads** — AI-07 and AI-09 decide what reasoning and a tool call carry, under § 6.3's rule.
- Any **wire representation** — AI-24 onward.
- Whether image and audio ever gain producers — `V-PRV-07` and doc 0001 § 8 keep them producerless.
