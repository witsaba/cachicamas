# Explore — content parts: readable and sealed

> **Change**: `cachicamas-ai-content-parts`
> **Milestone**: AI-06 — Define content parts: readable and sealed · **the keystone of Wave 1**
> **Nodes**: AI-06.1 `[decision]` · AI-06.2 `[leaf]` · AI-06.3 `[leaf]` · AI-06.4 `[guard]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Target package**: `backend/agent/src/ai/` (Layer 1), with the cross-package proof in `backend/agent/src/agenttest/`
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 — agent stack v2](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) § 3.1 · [the register](../../../specs/ai-contract-vocabulary/spec.md) · [AI-04's decision](../2026-08-01-cachicamas-ai-validation-errors/decision.md) · [AI-05's design](../2026-08-01-cachicamas-ai-message-roles/design.md) § 4
> **Predecessors**: AI-00 … AI-05
> **Blocks**: AI-07, AI-08, AI-09, AI-10, and — through readability — AI-26

---

## 1. What this milestone is, in one paragraph

AI-06 decides **what a content part is**, and it has to decide two properties that the retired design decided separately and in opposite directions: a part's payload must be readable from another package, and no value that skipped the constructor may be valid. Doc 0002 states the trap in one sentence — *"a part whose payload is readable from another package is usually one that can also be BUILT there, which is exactly the bypass to avoid"* — and rules that **any strategy satisfying one and not the other has failed this milestone**. Every later content variant (AI-07 reasoning, AI-09 tool call and tool result) inherits the answer instead of re-deriving it, which is why the milestone has a `[decision]` node and why that node is the milestone.

## 2. What already exists, and what it settles

### 2.1 The code

`backend/agent/src/ai/` holds `doc.go`, `import_boundary_test.go` (AI-00's guards), `validation.go` (AI-04), and `role.go` + `message.go` (AI-05). `go.mod` carries zero requires and two landed tests hold it there. AI-06 is stdlib-only.

| Landed surface | How AI-06 uses it |
| --- | --- |
| `ErrEmpty`, `ErrNotInVocabulary`, `ErrOutOfRange` | The three rule classes a content part needs. AI-06 appends none |
| `Invalid(rule, at …Step)`, `At`, `AtIndex`, `Path` | The one failure value, and the positions `content[0]` and `content[0].text` |
| `Rule`, `FirstFailure` | The documented rule order inside a constructor |
| `Role`, `roleNames`, `Roles()` | The closed-vocabulary pattern AI-05 wrote to be copied — AI-06's kind vocabulary is its second instance |
| `Message`, `NewMessage`, `Message.Content()` | The contract a part goes into, and the boundary that must reject an unconstructed one |
| `Content interface { isContent() }` | **The open seam AI-05 landed for this milestone to close** |

### 2.2 The seam AI-05 left open, with its dimensions written on it

AI-05's `design.md` § 4.2 is an explicit hand-off table: the seam exposes nothing readable (item 1a open), validates nothing (item 1b open), registers nothing (item 3 open) and declares no accessor (item 4 open). `message.go`'s own GoDoc records the bypass rather than closing it:

```go
type part struct {
	ai.Content        // embedded, nil; promotes the unexported method
	label      string
}
```

AI-05's external tests use exactly that, and `R-AMSG-009`'s second scenario **pins** that `NewMessage` rejects no content element — deliberately, so that AI-06.3 item 1 can fail before it passes. Two consequences for this milestone:

1. AI-06 **supersedes** `R-AMSG-009`'s second scenario. That is the intended disposition, not a regression, and this change records it rather than editing AI-05's artifacts.
2. AI-05's tests build every message from that embedded helper. Closing the seam changes their content helper. Migrating them is part of AI-06's diff and is not optional.

### 2.3 The vocabulary — five terms AI-06 implements rather than invents

| Id | Term | The clause that binds AI-06 |
| --- | --- | --- |
| `V-REQ-04` | content part | "One element of a message's ordered content, carrying exactly one kind of payload. Two properties are **constitutive, not incidental**: its payload is readable from outside the package that owns it, **and** no value that skipped construction — zero value, hand-rolled implementation, literal — can be valid. A definition carrying only one of the two re-creates a shipped defect." |
| `V-REQ-05` | content-part kind | "The discriminator naming which payload a content part carries. The kind set is **closed and exhaustively registered**; every registered kind has a constructible payload." |
| `V-REQ-06` | content-part readability | "An adapter or consumer in another package can read a part's payload back out of a constructed message. Without it, request translation is structurally impossible, not merely inconvenient." |
| `V-REQ-07` | content-part sealing | "A value which did not pass the construction rules cannot be valid, cannot reach a message, and cannot satisfy the part contract by accident." |
| `V-REQ-08` | text content | "The content-part kind carrying model-visible natural-language text. The first subject the part contract is proven against." |

The register already carries every term this milestone needs. **No amendment is required** — see § 6.

### 2.4 The two defects, in their own words

Doc 0001 § 3.1 records both, and the paragraph after the table is the one that decides this milestone's shape:

> C2 deserves emphasis because it is the one that blocked work rather than merely misleading a reader. The reasoning content type avoided the problem by implementing the part interface directly — and its own documentation explained why, in terms that applied verbatim to the three types that actually carried payload data. The wrapper pattern was retained for those three anyway. **The two strategies had to be reconciled**, in the direction the reasoning type already chose.

So the retired design ended with **two strategies for one contract**: a sealed-but-unreadable wrapper for text, tool call and tool result, and a readable-but-unsealed direct implementation for reasoning. C1 is what the second strategy costs (an exported type satisfying the part interface has a zero value that validates); C2 is what the first strategy costs (nothing outside the package can read the payload). This milestone's job is to find the shape that is neither.

## 3. The four tensions

### 3.1 Readability and sealing are the same word for an interface, and different words for a struct

This is the observation the whole milestone turns on, so it is worth stating before any candidate is named.

For an **interface**, "a consumer can read the payload" and "a consumer can supply an implementation" are both statements about *methods*. Exporting a method to make a payload readable is the same act as exporting the surface someone else can implement. Sealing the interface with an unexported method removes the second — and, as AI-05 proved in four lines, does not: an interface can still be *satisfied* by embedding it, producing a value with a nil embedded interface that panics on the first call. So an interface can be readable, or sealed, or embeddable-and-broken, but never cleanly both of the first two.

For a **struct**, the two are different words. Reading is *methods*; constructing is *fields*. A struct with only unexported fields cannot be built from outside with any content at all — the language forbids naming the field, and forbids a positional literal for a type with unexported fields — while its exported methods stay fully callable. The tension doc 0002 names is real, and it is a property of interfaces, not of Go.

### 3.2 The discriminator must not be a second source of truth

`V-REQ-05` calls the kind "the discriminator naming which payload a content part carries". A `Part{Kind: KindText, Text: "", ToolCall: tc}` shape lets the two disagree, and every consumer then has to decide which to believe. Doc 0002 item 2 forecloses it: **the kind is derived from the payload**. Whatever holds the payload has to be the thing that answers `Kind()`.

### 3.3 Accessor shape must survive four kinds, not one

AI-06 lands one kind. AI-07 lands reasoning, AI-09 lands tool call and tool result, and the register names image and audio as deliberately producerless (`V-PRV-07`). The accessor decided here is the one all of them use. Doc 0002 item 4 rules out the obvious shape — *"without a type switch over unexported types"* — which kills "keep the concrete variants unexported and let consumers switch on them", because that is C2 again.

### 3.4 Defense at both boundaries, when the second boundary does not exist

AI-06.3 item 3 asks for rejection "through the request path instead of the message path". **AI-10 does not exist.** Doc 0002's own rules forbid building it speculatively — split trigger 3 says a missing seam becomes a prerequisite node, and the milestone rules say one primary contract per milestone. The honest options are two: express the rule as something AI-10 reuses and *pin the reuse point* so the reuse is mechanical rather than remembered, or invoke the revert-and-record clause and add a node. § 5 resolves it.

## 4. Prior-art scan — how other Go contracts hold a closed, readable sum

Read for shape, not for imitation. None of these carries both of `V-REQ-04`'s properties, which is the point.

| Prior art | Shape | What it teaches AI-06 |
| --- | --- | --- |
| `go/token.Pos`, `time.Month`, `netip.Addr` | Opaque exported struct/scalar; unexported state; exported accessors; package-owned constructors | `netip.Addr` is the closest match in the standard library: fully readable (`Is4`, `String`, `As16`), impossible to build meaningfully from outside, and its **zero value is a documented invalid value** — exactly the shape AI-06.3 item 1 asserts on |
| `database/sql.NullString` and friends | Exported struct, exported fields, a validity flag | The anti-pattern: a flag is forgeable, and the zero value is "valid, empty" rather than "invalid" |
| `errors` wrapping (`Unwrap`, `Is`, `As`) | Interface with exported methods | Implementable from anywhere — correct there, because third-party errors are *supposed* to exist. Content parts are the opposite requirement |
| `go/ast` node types | Exported interface, exported concrete nodes, type switches everywhere | Readable and extensible, sealed against nothing. It is C1's shape |
| `crypto.Hash`, `image.Image` registries | A package-level table keyed by a constant, populated by `init` | The registry idea, minus the `init`-time mutability. AI-04 already ruled that a Layer 1 registry is an **ordered slice, never a map**, "because nothing in this package may let an unordered iteration decide anything" |
| AI-05's `Role` + `roleNames` | Constants from `iota + 1`, a slice indexed by the constant, an enumerator that walks the *constant space* rather than the table | The pattern AI-06's kind vocabulary copies verbatim — including the reason `Roles()` walks constants: a member declared without a table entry must be *visible to the assertion that catches it* |

The standard library's own answer to "readable but not constructible" is unanimous: **an opaque exported value type**. `netip.Addr`, `time.Time`, `token.Pos`, `sql.Rows` — none is an interface, all are readable, none can be assembled by a consumer.

## 5. The four candidate strategies, named here and judged in `decision.md`

Recorded here so that `decision.md` argues against a list that was written before the answer was chosen.

| # | Strategy | Reads? | Seals? |
| --- | --- | --- | --- |
| A | Exported interface `Part` with exported `Kind()`, exported concrete variants (`TextPart`) | yes | no — C1 verbatim |
| B | Unexported wrapper struct, only the discriminator exported | no — C2 verbatim | yes |
| C | **One exported opaque struct `Part`, unexported payload field, exported accessors, package-owned constructors** | yes | yes |
| D | Sealed interface (unexported method) plus package-level accessor functions `TextOf(Content) (string, bool)` | yes | partially — embedding still satisfies it, so every boundary must re-assert |
| E | Generic `Part[T]` | yes | yes, but viral and cannot hold heterogeneous content in one slice |

The scan of § 4 and the observation of § 3.1 both point at **C**, and `decision.md` argues A, B, D and E at full strength before rejecting them.

### 5.1 The AI-06.3 item 3 question, worked

If the answer is C, `Message` holds `[]Part` rather than `[]Content`, and validation of a part is a package-internal function that the message constructor calls. AI-10 lives in the **same package**. So "the same value offered through the request path" is not a second implementation — it is the same function called with a longer position prefix. That makes the first option of § 3.4 real rather than a dodge: expose the rule as `validateContent(prefix Path, content []Part) *Violation`, whose prefix parameter exists for exactly one caller that does not exist yet, and pin the reuse point with a test that calls it with a request-shaped prefix and asserts the position renders as `messages[2].content[0]`. No speculative request type is built. `proposal.md` § "AI-06.3 item 3" states the choice and `tasks.md` records it.

## 6. Vocabulary check — does AI-06 need a register amendment?

Walked term by term against the landed register, because doc 0002's rules require a milestone to cite rather than paraphrase.

| Concept AI-06 introduces | Register term | Amendment needed? |
| --- | --- | --- |
| The part value itself | `V-REQ-04` content part | No |
| The discriminator and its closed set | `V-REQ-05` content-part kind | No |
| "Readable from another package" | `V-REQ-06` | No |
| "Cannot be valid if it skipped construction" | `V-REQ-07` | No |
| The text variant | `V-REQ-08` text content | No |
| The documented length bound on text | `V-FAIL-01` + `ErrOutOfRange`'s existing case (`V-REQ-24`) | No — a documented Layer 1 bound is already a named caller-contract failure class |
| The rule that an unconstructed value fails | AI-04's five classes | **No new sentinel.** See § 6.1 |

**Conclusion: no register amendment.** `V-REQ-04` … `V-REQ-08` cover every noun this milestone uses.

### 6.1 Why an unconstructed part needs no new sentinel

AI-04's `decision.md` § 8 explicitly left AI-06 the question *"whether a rule class for 'an unconstructed value' is needed and must be appended"*. It is not, and the reason is the same one that makes item 2 work: **the kind is derived from the payload, so a part that skipped construction has no kind, and a value with no kind is a value outside a closed vocabulary.** `ErrNotInVocabulary` is exactly that class, and AI-05 already set the precedent — `Role(0)`, the role nobody set, fails with `ErrNotInVocabulary`, not with a sentinel of its own. Adding one would be a second way to say the same fact.

## 7. Forecast — this milestone will overrun the review budget

Stated before the work rather than discovered after it, per AI-05's precedent.

- **Production**: three files (a new part file, a new text file, a modified `message.go`), roughly 200 non-comment lines. Well inside budget.
- **Tests**: five files across two packages, and one of them shells out to the Go toolchain to prove a compile-time property. Roughly 600 lines.
- **Migration**: AI-05's two test files change their content helper and their element type. Mechanical, but it touches ~40 lines across two files.
- **Prose**: six artifacts, of which `decision.md` is the milestone.

Doc 0002's budget is "prefer < 250 changed lines, stop and reassess before 400". **It will be exceeded, and doc 0002 expects it**: AI-06 is the keystone, its charter blocks four milestones plus AI-26, and its acceptance test is a cross-package round trip. The reassessment is recorded in `tasks.md` rather than resolved by cutting tests.

## 8. What this milestone deliberately does not do

- **No reasoning, tool-call or tool-result variant** (AI-07, AI-09). The strategy is proven on text alone, which is `V-REQ-08`'s stated job.
- **No image or audio kind.** `V-PRV-07` and doc 0001 § 8 name them as deliberately producerless.
- **No request type** (AI-10). See § 5.1.
- **No parser for a kind rendering.** `ParseRole` exists because `V-REQ-01` puts provider-role mapping in an adapter; no register clause asks a kind to be parsed, and surface without a citable case is surface that has to be frozen for nothing.
- **No serialization, JSON tags or wire encoding.** That is the adapter's, from AI-24 on.
- **No rule about which role may carry which kind.** Doc 0002 assigns it to AI-10.3.
