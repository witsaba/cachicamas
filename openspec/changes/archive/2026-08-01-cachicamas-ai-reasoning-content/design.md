# Design — reasoning content and its round-trip token

> **Change**: `cachicamas-ai-reasoning-content`
> **Milestone**: AI-07 · **Nodes**: AI-07.1 … AI-07.4
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-reasoning-content/spec.md`
> **Inherited whole**: [AI-06's `decision.md`](../2026-08-01-cachicamas-ai-content-parts/decision.md) — especially § 6 (accessor shape), § 7 (validation), § 8 (the five steps) and **§ 9 (the inheritance table)**

---

## 1. What this document owns

AI-06 chose the content-part strategy and demonstrated both of its constitutive properties against it. **This milestone does not re-open any of that.** `decision.md` § 9 is a table written so that "AI-07, AI-09 and AI-10 can cite this row instead of re-deriving the answer", and every row is cited here:

| Question | Answer, decided by AI-06 | Where AI-07 uses it |
| --- | --- | --- |
| What type does a variant produce? | `ai.Part` | `NewReasoning` returns one |
| How is the variant distinguished? | By its payload type, which answers `kind()` | `Reasoning.kind()` returns `PartKindReasoning` |
| How does a consumer discover it? | `p.Kind()` against a closed exported vocabulary | unchanged |
| How does a consumer read it? | `p.<Kind>() (T, bool)` | `p.Reasoning() (Reasoning, bool)` — the exact spelling § 6.1 wrote |
| Where do the rules live? | On the payload's `validate(at Path)` | `Reasoning.validate` |
| Which sentinels? | AI-04's five; a variant appends one only for a case they do not cover | none appended — § 4.4 |
| Where does it register? | `partKindNames`, plus the guarded GoDoc list | three appends to `content_part.go` |
| What proves it? | An external round trip, a zero-value rejection, a guard entry with three legs | § 6 |
| May a payload type be exported? | Yes, when it carries structure — opaque, unexported fields, package-owned constructor | `Reasoning` is exactly that |

What is left for this document is the payload: its fields, its doors, its state derivation, and its rules.

---

## 2. The surface

```go
// reasoning_content.go

const MaxReasoningTokenLen = 1 << 20

type ReasoningState uint8

const (
	ReasoningStateText ReasoningState = iota + 1
	ReasoningStateRedacted
	ReasoningStateTokenOnly

	reasoningStateFirst = ReasoningStateText
	reasoningStateEnd   = ReasoningStateTokenOnly + 1
)

func ReasoningStates() []ReasoningState
func (s ReasoningState) String() string

type Reasoning struct {
	text     string
	token    string
	hasToken bool
	redacted bool
}

func (r Reasoning) State() ReasoningState
func (r Reasoning) Text() string
func (r Reasoning) Token() ([]byte, bool)

func NewReasoning(text string, token []byte) (Part, error)
func NewRedactedReasoning(token []byte) (Part, error)
func (p Part) Reasoning() (Reasoning, bool)

// unexported
var reasoningStateNames = []string{…}
func reasoningStateName(s ReasoningState) string
func (Reasoning) kind() PartKind
func (r Reasoning) validate(at Path) *Violation
```

```go
// content_part.go — three appends, nothing renumbered

//   - reasoning — …                      (the documented list)
PartKindReasoning                          (the constant, after PartKindText)
partKindEnd = PartKindReasoning + 1        (moved past it, per step 1)
PartKindReasoning: "reasoning",            (the table entry)
```

### 2.1 Why each name

| Name | Why |
| --- | --- |
| `Reasoning` | `V-REQ-09` is "reasoning content"; the package is `ai`, so `ai.Reasoning` reads as the register's term. `decision.md` § 6.1 already spelled it, in the signature it wrote for this milestone |
| `ReasoningState`, `ReasoningStateText`, … | `V-REQ-10` is "reasoning state". Constants carry their type as a prefix, exactly as `RoleUser` and `PartKindText` do |
| `ReasoningStateTokenOnly` | Names the shape by what it carries rather than by what a single provider calls it. "SignatureOnly" would bake one vendor's word for the token into a neutral vocabulary, and `V-REQ-10`'s fourth shape — "a provider that emitted no reasoning text at all" — is the same state under a different vendor's word |
| `NewReasoning` | The kind's constructor, named for the kind. `decision.md` § 8 step 3's `New<Kind>` |
| `NewRedactedReasoning` | The second door, named for the state only it can build. § 3.2 is why there are two |
| `Token() ([]byte, bool)` | Bytes, not a string, because the value is opaque and a consumer must not be invited to print it. The bool is `V-REQ-11`'s absence, in the shape `decision.md` § 6.2 anticipated |
| `MaxReasoningTokenLen` | `Len` and not `Size`, because it bounds a byte count exactly, matching `MaxTextLen` |

### 2.2 What is deliberately not on the surface

| Absent | Why |
| --- | --- |
| `ParseReasoningState` | No register clause asks a state to be parsed. `ParseRole` exists because `V-REQ-01` puts provider-role mapping in an adapter; nothing says the same of a state |
| A state parameter on `NewReasoning` | § 3.2. It would make a contradiction expressible, and reporting one needs a rule class AI-04 does not have |
| `Reasoning.Text() (string, bool)` | The state already records whether there is text, and a second predicate would be a second way to ask one question — the argument AI-06's `design.md` § 2.2 used against `Part.IsZero()` |
| `Reasoning.HasToken() bool` | Same reason: `Token()`'s second result is that answer, and a value plus its presence belong in one call |
| A `MaxReasoningTextLen` constant | Reasoning text is model-visible text and takes the package's existing bound, `MaxTextLen`. A second name for one number is a second thing to keep in agreement |
| Any exported field, on any type | `decision.md` § 3.3 bypasses 2 and 3, inherited unchanged |

### 2.3 The concession, stated openly

A caller cannot **set** the state; it is derived. So an adapter that has read a provider's block cannot say "this is redacted" by assigning a value — it must call the door that builds redacted parts. That is one more name to know, in exchange for a contradiction that cannot be written, and it is the identical trade `Part.Kind()` already made one milestone ago.

---

## 3. The three decisions this milestone actually makes

### 3.1 Absence versus an empty token: a presence bool, never a nil convention in storage

`Reasoning` stores the token **and** a `hasToken` flag. The flag is the storage; `nil` is only the spelling at the door.

- **At the door**, `token == nil` means *absent* and any non-nil slice — including one of length zero — means *present*. One rule, both doors, documented on both.
- **In storage**, presence is a `bool`, so it cannot be lost by anything that touches the bytes. A design that inferred presence from `len(token) == 0`, or from the slice being nil after a clone, would collapse the distinction in exactly the place `V-REQ-11` needs it kept — and `slices.Clone` of an empty slice is precisely the kind of subtlety that decides such a thing by accident.
- **On the read path**, `Token() ([]byte, bool)` returns the bool. `decision.md` § 6.2 named this shape for this requirement by name.

### 3.2 The state is derived, and redaction is a door

```go
func (r Reasoning) State() ReasoningState {
	switch {
	case r.redacted:
		return ReasoningStateRedacted
	case r.text != "":
		return ReasoningStateText
	default:
		return ReasoningStateTokenOnly
	}
}
```

No stored state field, for `decision.md` § 5's three reasons, one level down:

1. **The state and the payload cannot disagree**, because there is only one of them. A part labelled *redacted* carrying plaintext is not a state this type can be in — so it is not a rule anything has to report, and `explore.md` § 6 records that this is what keeps the change free of a sentinel append.
2. **`State()` is total.** Every reachable field combination maps to a member of the vocabulary, so no constructed part can report a non-member.
3. **Redaction is the one bit that is not derivable.** "The provider withheld the plaintext" is a fact about the provider, not a shape of the bytes: a redacted blob and a signature are both opaque bytes with no text beside them. So it is a `bool` set by one door and by nothing else, rather than a value a caller passes.

`NewReasoning` therefore derives on **text presence alone**: a non-empty text yields the *with text* state, an empty text yields *token-only*. Model-visibility is a rule rather than part of the derivation — see § 4.2.

### 3.3 The token is stored as a `string`

A `[]byte` field would be the obvious shape and is the wrong one, for two reasons that only appear later:

1. **Aliasing.** A stored slice shares its backing array with the caller's buffer, and the value changes with nobody writing to the part. `Reasoning` is also handed **out** by value, so the same hazard exists on the read path; a defensive clone in the constructor alone fixes half of it. AI-08 hit this on schema bytes. A `string` is immutable, so both halves are closed by the type rather than by two remembered clones.
2. **Comparability.** `content_part.go` documents that `Part` is "a value, and safe to copy… Equality with `==` is defined and compares payloads". A payload containing a slice makes `==` **panic** at runtime for that kind. A landed property of `Part` must not regress because a second kind arrived.

A Go string is a byte container, not a text type: `textPayload` already relies on that, and its GoDoc records that a text part "may hold … bytes that are not well-formed UTF-8". The conversions at the boundary — `string(token)` in and `[]byte(r.token)` out — each copy, which is why a consumer can overwrite what an accessor returned without reaching the part.

**This is the destination, not the starting point.** § 7 records the red step: the token is first stored as the caller's slice, the aliasing test is recorded failing, and the storage is then changed. Taking that red is doc 0002's instruction and not a formality — it is the difference between a property that is proven and one that is asserted.

---

## 4. Validation

### 4.1 The rules, in their documented order

`Reasoning.validate(at Path)`, called by both doors and by the message boundary — one rule set, two entry points, `decision.md` § 7.2:

1. the *with text* state carries at least one non-whitespace character, else `ErrEmpty` at `…text`;
2. the text is at most `MaxTextLen` bytes, else `ErrOutOfRange` at `…text`;
3. a state that carries no text carries a token, else `ErrEmpty` at `…token`;
4. the token, when present, is at most `MaxReasoningTokenLen` bytes, else `ErrOutOfRange` at `…token`.

Text before token because text is the model-visible half, and within each, emptiness before bound — "you gave me nothing" and "you gave me too much" are different facts with different fixes, and a caller fixing one at a time must make progress in a predictable direction. That ordering is `text_content.go`'s, restated so a reader does not have to fetch it.

### 4.2 Why model-visibility is a rule and not part of the derivation

`NewReasoning("   ", token)` could plausibly mean "no reasoning text", and deriving *token-only* from whitespace would make it so. It is rejected instead, with `ErrEmpty` at `text`, for `text_content.go`'s reason: **the emptiness rule inspects the text, it does not rewrite it.** Deriving *token-only* from whitespace would store a payload whose state says there is no text while `Text()` returns three spaces — a disagreement re-introduced through the back door of a convenience. The empty string is the only way to say "no text", and it is unambiguous.

### 4.3 Rule 3, and what "carries nothing" means

A reasoning part with no text and no token holds a state and nothing else. It is not a shape any provider produces and it is not one an adapter can render, so it is a caller-contract failure rather than a valid empty. It reports `ErrEmpty` at `token` — the position names the value that is missing and could have been supplied, which is what makes the failure actionable. This is the rule the AI-06.4 guard's third leg trips for this kind: the zero `Reasoning`, assembled without a door, is exactly this shape.

A **present but zero-length** token satisfies rule 3. That is deliberate and is `R-ARC-005` in one line: the provider sent an empty token, which is a fact this package records rather than corrects.

### 4.4 No new rule class, and the one that was nearly needed

The five landed classes cover every rule above. The violation that would have needed a sixth — *a state that contradicts the payload it describes* — is designed out by § 3.2 rather than reported, so `validation.go` is not touched and the register is unamended.

Stated plainly because AI-04's append rule cuts both ways: had the design kept a supplied state, the honest response would have been to **append a class in this pull request**, not to stretch `ErrNotInVocabulary` (the state *is* in the vocabulary) or `ErrMalformed` (nothing is malformed for its encoding). Removing the state that needs the class is better than either, and it is available here only because the state is genuinely derivable.

### 4.5 Redaction of the failure

No rule above reproduces a byte of the text or of the token: a position is built from names this package wrote, and `Violation.Error` renders only a registered class's own text. `Part.String()` already names the kind and never the payload, inherited unchanged — which matters more for this kind than for text, because a token can be a credential-shaped blob and a reasoning trace is the most sensitive text a model produces.

---

## 5. The five-step procedure, walked

`decision.md` § 8, applied literally. Each step names what AI-06.4 asserts about it.

| # | Step | This change |
| --- | --- | --- |
| 1 | Declare the constant at the end of the `PartKind` block and move `partKindEnd` | `PartKindReasoning` appended **after** `PartKindText`; `partKindEnd = PartKindReasoning + 1` |
| 2 | Add the payload type with `kind()` and `validate(at Path)` | `Reasoning`, in `reasoning_content.go` |
| 3 | Add the exported constructor whose rules are the payload's | `NewReasoning`, plus `NewRedactedReasoning` for the state it cannot express |
| 4 | Add the exported accessor `func (p Part) <Kind>() (T, bool)` | `Part.Reasoning` |
| 5 | Add the name to `partKindNames` and to the documented list | `"reasoning"`, in both |

**On concurrency.** AI-09 adds two more kinds to the same three places. Every edit here is an **append at the end of a list**; no existing constant, table entry or documentation line is renumbered, reordered or reworded. The guard cross-checks the declared constant space against `PartKinds()` and `partKindNames` in both directions, so a merge that loses an entry or leaves `partKindEnd` behind fails the build.

---

## 6. File layout and test plan

| File | Node | Package | Content |
| --- | --- | --- | --- |
| `src/ai/reasoning_content.go` | AI-07.1 … AI-07.4 | `ai` | The state vocabulary, `Reasoning`, its rules, both doors, `Part.Reasoning` |
| `src/ai/content_part.go` | AI-07.1 | `ai` | *three appends*, § 5 |
| `src/ai/reasoning_content_test.go` | all four | `ai_test` | Every test-list item |
| `src/ai/content_part_registry_test.go` | AI-07.1 | `ai` | *one append* — the witness with three legs |

One file per kind is AI-06's layout, inherited: the payload type, its rules, its constructors and its accessor live together, so adding a kind is one new file plus three lines.

| Node | Test | Item |
| --- | --- | --- |
| AI-07.1 | `TestReasoning_ConstructedAndReadBack_UsesTheContentPartStrategy` | 1 |
| AI-07.1 | `TestReasoning_AndText_AreStructurallyDistinct` | 2 |
| AI-07.1 | `TestReasoningStates_Vocabulary_IsClosedStableAndConstructible` | 3 |
| AI-07.2 | `TestReasoning_AbsentToken_IsDistinguishableFromAnEmptyToken` | 1 |
| AI-07.2 | `TestReasoning_OpaqueToken_IsNeitherInterpretedNorNormalized` | 2 |
| AI-07.3 | `TestReasoning_TokenThroughAMessage_RoundTripsByteIdentical` | 1 |
| AI-07.3 | `TestReasoning_TokenAcrossCopies_IsUnaffectedByTheCallersSlice` | 2 |
| AI-07.4 | `TestReasoning_RedactedVariant_ReplaysItsPayloadVerbatim` | 1 |
| AI-07.4 | `TestReasoning_SignatureOnlyVariant_IsConstructibleAndValid` | 2 |
| AI-07.4 | `TestReasoning_RedactedAndSignatureOnly_AreConfusableWithNothing` | 3 |
| — | `TestNewReasoning_RuleViolations_FailWithTheDocumentedSentinels` | the rules of § 4.1 |

Every name follows `Test<Subject>_<Behavior>_<Expectation>`; every scenario banner cites its leaf ID.

**Where the tests live.** `ai_test`, the external test package, so every assertion is written against exactly the surface a consumer sees. AI-06 put its cross-package round trip in `src/agenttest/` because that package's own comment reserved the *first* such proof by name; the reservation is spent, and a second copy of the same loop in a second package would prove nothing new.

---

## 7. How each red step is taken

AI-05's rule, inherited through AI-06: **a compile error is the state before red, not red.** Each item lands the narrowest thing that compiles and fails.

| Item | The stub that makes it fail rather than not build |
| --- | --- |
| AI-07.1.1 | `PartKindReasoning` declared, `Reasoning` with no fields, `NewReasoning` returning the zero `Part`. The test reads a payload that was never stored — **and AI-06.4's guard fails in the same run**, because a declared constant with no witness is the first thing it checks |
| AI-07.1.2 | `Part.Reasoning` returning `Reasoning{}, true` unconditionally: it answers yes for a text part |
| AI-07.1.3 | `ReasoningStates()` over a vocabulary of one member |
| AI-07.2.1 | The token stored, presence inferred from `len(token) == 0` — the collapse, made concrete |
| AI-07.2.2 | The rules present, the bound absent, so the over-bound case is accepted |
| AI-07.3.1 | Already green if 07.2 was done right; taken as a **pin** and marked as one, per doc 0002's leaf anatomy |
| AI-07.3.2 | **The token stored as the caller's slice.** The red is a real aliasing bug, taken honestly rather than pre-empted by a clone — § 3.3 |
| AI-07.4.1 | `NewRedactedReasoning` absent; its state derives to *token-only*, so the two shapes are indistinguishable |

---

## 8. What this design deliberately does not do

- **No second strategy.** Doc 0001 § 3.1 records that the retired Layer 1's reasoning type "implemented the interface directly" while three other payloads kept a wrapper, and that the two "had to be reconciled". Reasoning is the kind that broke the pattern last time; this design's most important property is that it is unremarkable.
- **No `init()`, no map, no reflection in production.** Inherited from AI-05 and AI-06 for their stated reasons.
- **No token interpretation of any kind** — not a UTF-8 check, not a base64 probe, not a length heuristic, not a "looks like JSON" branch.
- **No change to `validation.go`, `role.go`, `message.go`, `text_content.go`, or any tool or finish-reason file.** AI-09 is in flight in another worktree; the only shared file is `content_part.go`, and § 5 records how it is shared.
