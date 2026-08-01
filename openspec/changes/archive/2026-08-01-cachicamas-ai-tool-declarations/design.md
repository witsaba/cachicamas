# Design — tool declarations

> **Change**: `cachicamas-ai-tool-declarations`
> **Milestone**: AI-08 · **Nodes**: AI-08.1 `[leaf]`, AI-08.2 `[leaf]`, AI-08.3 `[leaf]`
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-tool-declarations/spec.md`
> **Inherits**: AI-04's taxonomy · AI-05's closed-vocabulary pattern (`cachicamas-ai-message-roles/design.md` § 3)

---

## 1. What this document owns

Every Go spelling this milestone lands, the exact tool-name rule and its justification, the documented order of every rule list, and the one extension this milestone makes to AI-05's closed-vocabulary pattern. `spec.md` says what must be true; this file says what is written.

It does **not** own: what a tool does, how a name resolves, whether a schema is well formed, or how any of this attaches to a request. Those are Layers 2 and 3, nobody, nobody, and AI-10 respectively.

---

## 2. The surface

Three files, split on the leaf boundary.

```go
// tool.go — AI-08.1

type Tool struct{ /* name, description string; schema []byte */ }

func NewTool(name, description string, schema []byte) (Tool, error)

func (t Tool) Name() string
func (t Tool) Description() string
func (t Tool) Schema() []byte

const MaxToolNameLen = 64
```

```go
// tool_set.go — AI-08.2

type ToolSet struct{ /* tools []Tool */ }

func NewToolSet(tools ...Tool) (ToolSet, error)

func (s ToolSet) Tools() []Tool
func (s ToolSet) Len() int
func (s ToolSet) Declares(name string) bool
```

```go
// tool_choice.go — AI-08.3

type ToolChoiceMode uint8

const (
	ToolChoiceAuto ToolChoiceMode = iota + 1
	ToolChoiceNone
	ToolChoiceRequired
	ToolChoiceSpecific
)

func ToolChoiceModes() []ToolChoiceMode
func (m ToolChoiceMode) String() string
func ParseToolChoiceMode(name string) (ToolChoiceMode, error)

type ToolChoice struct{ /* mode ToolChoiceMode; name string */ }

func NewToolChoice(mode ToolChoiceMode) (ToolChoice, error)
func NewNamedToolChoice(name string) (ToolChoice, error)

func (c ToolChoice) Mode() ToolChoiceMode
func (c ToolChoice) Name() (string, bool)
func (c ToolChoice) ValidateAgainst(tools ToolSet) error
```

### 2.1 Why each name

- **`Tool`, not `ToolDeclaration`.** The register's term is "tool declaration", but the package is `ai` and the type is used as `ai.Tool`. `ai.ToolDeclaration` stutters against a package whose whole job is declarations of contracts; the GoDoc's first line cites `V-REQ-12` by name, which is what keeps the term traceable.
- **`Schema()`, not `InputSchema()`.** `V-REQ-13`'s term is "schema bytes". A vendor calls the field `input_schema`; that spelling belongs to an adapter.
- **`ToolSet`, not `Tools`.** The register's term, and a plural type name would collide with `ToolSet.Tools()`.
- **`Declares`, not `Contains` or `Has`.** It answers the question cross-validation asks — "does this request declare a tool by that name?" — and `V-REQ-15`'s own wording is "validated against the **declared** tool set".
- **`ToolChoiceMode` and `ToolChoice` are two types.** § 5 defends the split; the short version is that the vocabulary must stay a payload-free integer for AI-05's pattern to apply at all.
- **`NewNamedToolChoice`, not `NewSpecificToolChoice`.** The distinguishing fact at the call site is that this constructor takes a name; "specific" is the member's rendering and belongs on the constant.
- **`MaxToolNameLen` is exported.** A caller that generates tool names — a plugin host, a Layer 3 tool source — needs the ceiling before it constructs, not after it fails. It is the only constant here a caller could want.

### 2.2 What is deliberately not on the surface

| Omitted | Why |
| --- | --- |
| `Tool.String`, `ToolChoice.String` | No citable case, and both would render caller data. `ToolChoiceMode.String` exists because the pattern requires a rendering; a `ToolChoiceMode` is an integer and carries none |
| `Tool.Equal`, `ToolSet.Equal` | AI-10.6 item 3 owns "the documented equality", defined over a request's regions |
| `ToolSet.Get(name)`, `ToolSet.Index(name)` | `Declares` is what cross-validation needs. A lookup returning the declaration invites a consumer to resolve a name, which is `V-OUT-04`'s |
| `MarshalJSON` / `UnmarshalJSON` anywhere | Wire shapes are confined to adapters (`V-PRV-15`). A marshaller here is a second rendering path with a second set of rules — and on `Tool` specifically it would be the exact mechanism `R-ATD-002` forbids |
| A schema **syntax** check | § 8 |
| `ToolSet.Add`, `ToolSet.Without` | Copy-on-write rebuilding is AI-12's, at request scope |

---

## 3. The tool-name rule

### 3.1 The rule

A tool name is legal exactly when all three hold:

1. It is **1 to 64 bytes** long.
2. Its **first byte** is an ASCII letter (`A`–`Z`, `a`–`z`) or `_`.
3. **Every byte** is an ASCII letter, an ASCII digit, `_`, or `-`.

Bytes, not runes, and deliberately: the alphabet is ASCII-only, so any multi-byte sequence fails rule 3 on its first byte, and a byte-wise scan cannot disagree with a rune-wise one about a string that contains no legal multi-byte character. `validation.go`'s own `structuralName` filter uses the same byte-wise shape for the same reason.

### 3.2 Why the intersection, and which half is verified

The rule is the **intersection** of what real providers accept, not the union, and not "anything non-empty".

The verified anchor: Anthropic's published constraint on a tool `name` is **"Must match the regex `^[a-zA-Z0-9_-]{1,64}$`"** — read from [platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools](https://platform.claude.com/docs/en/agents-and-tools/tool-use/define-tools) on 2026-07-31 for this change. That is rules 1 and 3 exactly. Other major vendors publish the same alphabet and the same 64-byte ceiling.

Rule 2 — the leading-character restriction — is a **deliberate narrowing** and is not from that source. It is added because at least one major vendor additionally requires a function name to begin with a letter or an underscore, and because a name that is a valid identifier in every plausible target is worth more than the handful of names rule 2 excludes (`-foo`, `9foo`, `-`). A reviewer should read rule 2 as a choice this change made, not as a constraint it inherited; § 3.4 records why the choice is safe to revisit.

The reason the intersection wins is specific to *names* rather than to validation generally, and it is worth stating because "be liberal in what you accept" would otherwise be the obvious default: **a tool name is not merely sent, it comes back.** The model answers with a tool call naming the tool, and Layer 2 resolves that name to an implementation. If Layer 1 accepted a name some adapter had to rewrite on the way out, that adapter would have to un-rewrite it on the way in — a bidirectional mapping AI-26 would own, AI-30 would have to invert, and every future adapter would have to reimplement. The alternative to rejecting the name here is therefore not "the provider decides"; it is either a network round trip ending in a 400, or a silent adapter-local mangling that breaks call correlation. Doc 0002's phrase for the seam is exactly this: *caught client-side*.

### 3.3 The rules, and their classes

```go
func NewTool(name, description string, schema []byte) (Tool, error) {
	if err := FirstFailure(
		toolNameRules(name)...,          // 1. empty  2. too long  3. bad shape
		func() *Violation {              // 4. empty schema
			if len(schema) == 0 {
				return Invalid(ErrEmpty, At("schema"))
			}
			return nil
		},
	); err != nil {
		return Tool{}, err
	}
	// ...
}
```

The documented order is: **empty name → over-long name → malformed name → empty schema**. Three classes, and the split between rules 2 and 3 is the part worth defending:

| Violation | Class | Why not the other one |
| --- | --- | --- |
| Empty name | `ErrEmpty` | "You gave me nothing" is a different fact from "you gave me something wrong" — `ParseRole` already draws this line |
| Name of 65+ bytes | `ErrOutOfRange` | A documented *bound*, and the fix is "shorten it". Rendering it as malformed would tell a caller to change the spelling of a perfectly well-spelled name |
| Name with `.`, a space, a leading digit | `ErrMalformed` | Not well-formed for its documented encoding, and the fix is "spell it differently". Rendering it as out-of-range would be nonsense for a three-character name |

`V-FAIL-17` is the authority for the split: a rule class is "the kind of thing a rule checks, independent of what it checks it on". Length and shape are different kinds. Doc 0002's AI-08.1 item 3 names one item ("a name outside the documented character rules"); the length case is therefore a **discovered case appended** to the leaf's list under the living-graph clause's rule 3, and `tasks.md` records the append.

The description has **no rule**. Every provider a v1 adapter could target treats a tool's description as optional — strongly recommended in at least one, and recommendation is not a caller contract. `R-ATD-001`'s second scenario pins the emptiness as legal so a later reader cannot mistake the absence of a rule for an oversight.

### 3.4 The asymmetry that makes this safe

Loosening the rule later is additive: a name that was rejected becomes accepted, and nothing that worked stops working. Tightening it is breaking: a name a caller shipped stops constructing. Starting at the intersection keeps the cheap move available; starting at the union does not. That is the whole argument for landing rule 2 despite it being a narrowing rather than an inherited constraint.

---

## 4. Byte fidelity

Two mechanisms, and they are not one:

```go
func NewTool(name, description string, schema []byte) (Tool, error) {
	// ... rules ...
	return Tool{name: name, description: description, schema: slices.Clone(schema)}, nil
}

func (t Tool) Schema() []byte { return slices.Clone(t.schema) }
```

This is `message.go`'s copy-in/copy-out discipline applied to bytes, and its reasoning transfers verbatim: a constructor that clones and a reader that does not passes every construction test and fails the moment two consumers hold the same declaration.

**What is deliberately absent is the interesting part.** There is no `json.Compact`, no `json.Indent`, no unmarshal-then-remarshal, no key sort, no whitespace trim, and no `bytes.TrimSpace`. Each is a one-line "improvement" that would pass a naive round-trip test — because a marshal round trip is idempotent *after the first pass* — and would silently destroy `V-REQ-25`'s cache prefix for every caller whose schema was not already in the canonical form.

The test therefore cannot use canonical JSON as its fixture. `R-ATD-002`'s fixture has object keys in non-alphabetical order and irregular internal whitespace, so a normalizing implementation produces different bytes on the first read, not on the thousandth:

```go
schema := []byte(`{"type":"object", "properties":{"zulu":{"type":"string"},"alpha":{"type":"number"}},   "required":["zulu"]}`)
```

Assert with `bytes.Equal`, then assert the returned slice does not alias the stored one by mutating it and re-reading.

---

## 5. The closed-vocabulary pattern, extended for a payload

AI-05's `design.md` § 3 states the pattern as four rules and says the three later vocabularies "each substitute their own noun". Three of tool choice's four members substitute cleanly. The fourth carries a name, and this section is the record of how that was absorbed.

### 5.1 The extension, in one sentence

**The vocabulary stays a payload-free integer; the payload lives in a separate value type carrying a member plus its payload; and the table row widens by one column recording whether a member takes a payload.**

That last clause is the whole extension, and AI-05 pre-authorised it: "Rule 2's table may carry more than a name — AI-13's finish reasons may want a retryability disposition beside the rendering — and that is a widening of the row, not a change of shape."

### 5.2 What that looks like

```go
// The vocabulary. Rules 1 and 3 of AI-05's pattern are untouched.
const (
	ToolChoiceAuto ToolChoiceMode = iota + 1
	ToolChoiceNone
	ToolChoiceRequired
	ToolChoiceSpecific

	toolChoiceFirst = ToolChoiceAuto
	toolChoiceEnd   = ToolChoiceSpecific + 1
)

// The table. Rule 2's row, widened by one column.
type toolChoiceEntry struct {
	name      string
	needsName bool // the arity column — the extension
}

var toolChoiceModes = []toolChoiceEntry{
	ToolChoiceAuto:     {name: "auto"},
	ToolChoiceNone:     {name: "none"},
	ToolChoiceRequired: {name: "required"},
	ToolChoiceSpecific: {name: "specific", needsName: true},
}
```

Rules 1 and 3 are untouched, which matters because AI-05 calls them "the load-bearing ones" with no degree of freedom. `iota + 1` still makes a `ToolChoiceMode` nobody set fail exactly like a wild one. `ToolChoiceModes()` still walks `toolChoiceFirst … toolChoiceEnd` — the **constant space**, not the table — which is the rule that makes the pin bite on a member declared without an entry.

### 5.3 The two constructors, and why two

```go
func NewToolChoice(mode ToolChoiceMode) (ToolChoice, error)   // the three payload-free members
func NewNamedToolChoice(name string) (ToolChoice, error)      // the payload-carrying member
```

Two constructors because there are two arities, and the arity column is the single source of which one a member takes. `NewToolChoice(ToolChoiceSpecific)` does not silently produce a nameless "specific" choice; it fails with `Invalid(ErrEmpty, At("name"))` — the member requires a name and none was supplied, which is exactly what the empty class means.

The rules, in documented order:

```
NewToolChoice(mode):        1. mode is a member         -> ErrNotInVocabulary at "toolChoice"
                            2. mode does not need a name-> ErrEmpty           at "name"

NewNamedToolChoice(name):   1. name is non-empty        -> ErrEmpty           at "name"
                            2. name satisfies the name rule (length, then shape)
                                                        -> ErrOutOfRange / ErrMalformed at "name"
```

`NewNamedToolChoice` reuses `toolNameRules` verbatim. A choice naming a syntactically impossible tool can never resolve against any set, so failing it at construction rather than at cross-validation gives the caller the precise fix — and it keeps one name rule in the package rather than two that could drift.

### 5.4 The pin, and how it stays mechanical

```go
// for every m in ToolChoiceModes():
//   1. m.String() is non-empty, lowercase, and not the diagnostic form
//   2. ParseToolChoiceMode(m.String()) == m
//   3. NewToolChoice(m) succeeds  <=>  the member does not need a name
//   4. when the member does need a name, NewNamedToolChoice(...) yields exactly m
```

Step 3 is the one the arity column makes possible. It asserts a biconditional against the table rather than against a hard-coded list of members, so a fifth member added tomorrow is covered by whichever branch its own row selects.

**The honest limit, recorded rather than discovered later.** Step 4 maps "needs a name" to one specific constructor. With one payload-carrying member that is exact. If a second one ever lands — a hypothetical "any of these tools" — the arity column stops being a boolean and step 4 needs a per-member constructor mapping. That is a widening of the same row and the same shape, and this paragraph is here so the milestone that meets the case reads it instead of re-deriving it.

### 5.5 What AI-13 should take from this

`V-MET-08` unknown finish reason is a member reached by a *failed* lookup, which AI-05 already flagged. Nothing here changes that. What AI-13 can take is the general move: **when a member needs something a bare integer cannot carry, widen the table row and put the payload in a separate value type — never widen the vocabulary type itself.** Widening the vocabulary type is what costs `iota + 1`, the indexed table, and the pin, all three at once.

---

## 6. The tool set

```go
type ToolSet struct{ tools []Tool }

func NewToolSet(tools ...Tool) (ToolSet, error) {
	for i, tool := range tools {
		if tool.Name() == "" {
			return ToolSet{}, Invalid(ErrEmpty, AtIndex("tools", i))
		}
		for j := 0; j < i; j++ {
			if tools[j].Name() == tool.Name() {
				return ToolSet{}, Invalid(ErrDuplicate /* see § 6.1 */, AtIndex("tools", i))
			}
		}
	}
	return ToolSet{tools: slices.Clone(tools)}, nil
}
```

Three decisions in that body.

**A slice, and a linear scan.** Not a map for membership, not a map for duplicate detection. AI-04's `ruleClasses` established the rule and its reason — "nothing in this package may let an unordered iteration decide anything, and a registry is where that temptation is strongest" — and a tool set is a registry. The scan is `O(n²)` over a collection whose realistic size is single or low double digits, and it buys the property that the *reported* duplicate is deterministic: always the lowest index whose name repeats an earlier one.

**The zero value is the empty set.** `ToolSet{}` has a nil slice; `Tools()` returns an empty slice, `Len()` returns 0, `Declares` returns false. That is the correct behavior rather than a convenience, because `R-ATD-006` makes the empty set legal — unlike a vocabulary, where the zero value must be invalid, here the zero value *is* a member of the legal domain.

**An unconstructed `Tool` is rejected at the collection boundary.** A `Tool{}` that skipped `NewTool` has an empty name, which no constructed `Tool` can have, so the emptiness check is a sound detector without a separate `constructed` flag. This is AI-06.3 item 1's shape applied one contract over: a value that skipped its constructor must not reach the wire. It is a **discovered case appended** to AI-08.2's test list; `tasks.md` records the append.

### 6.1 The duplicate sentinel

`ErrNotInVocabulary` is wrong for a duplicate and is written above only to mark the decision point. The five landed classes are: empty, not-in-vocabulary, out-of-range, malformed, unresolved-reference. A duplicate name is none of them cleanly — which is a real finding, not a gap to paper over.

The choice is **`ErrDuplicate`, appended by this milestone, positioned at the second occurrence.** The alternative readings and why they lose:

- `ErrOutOfRange` — a duplicate is not a bound.
- `ErrNotInVocabulary` — the set is not a closed vocabulary.
- `ErrMalformed` — **every name in the set is well-formed.** What fails is uniqueness *across* the set, which is not a property any single value has, so no inspection of any one declaration finds it. A consumer told "malformed" goes looking for the badly spelled name, and there is none.

The last of those is the one that decides it, and it decides it against the class this milestone first shipped. The two classes name the two different fixes a consumer must make — `ErrMalformed` says *spell it differently*, `ErrDuplicate` says *remove one of them* — and `errors.Is` is the only place that difference is readable. Collapsing them makes the taxonomy answer a question the consumer did not ask, which is `V-FAIL-06`'s recorded failure mode one class at a time.

Appending is what AI-04's decision record asks for, not an exception to it. `decision.md` § 3.5 makes the set "closed and appended, never invented": a milestone that meets a violation no landed class describes appends a class **in the pull request that needs it**, rather than defining a local sentinel. It forecast this exact class — "a 'two values conflict' class is plausible and has no citable case yet" — and named the citable case as the thing it was waiting for. AI-08.2 item 1 *is* that case. The rule was never "do not grow the set"; it was "do not grow it in anticipation".

The GoDoc on `NewToolSet` states this reading, because a reader who hits the failure will want the reasoning where the failure is, not in a change directory.

> **Amended 2026-07-31** — this section originally chose `ErrMalformed`, reading the *set as a whole* as not well-formed and citing AI-04's append rule as the reason not to add a class. That inverted the rule: § 3.5 says a milestone appends a class in the pull request that needs it, and AI-08 was that pull request. The class was appended and `NewToolSet` moved to it in a follow-up commit, with the assertion taken red first. The rejected-alternative list is kept and `ErrMalformed` moved into it, because the argument that defeats it is the substance of the correction and deleting it would leave the next reader to re-derive it. `decision.md` § 3.5, `design.md` § 3.1 and the AI-04 delta spec carry the matching amendments; no register row was needed, because `V-FAIL-17` states that which rule classes exist is AI-04's and not the register's.

---

## 7. Cross-validation

```go
func (c ToolChoice) ValidateAgainst(tools ToolSet) error
```

A method on the choice rather than a package function, because the choice is the thing being validated and the set is what it is validated *against* — `V-REQ-15`'s own wording.

The documented order, and the reason it is that order:

1. **The mode is a member** → `ErrNotInVocabulary` at `toolChoice`. This is what rejects the zero `ToolChoice`.
2. **The mode is not `none` and the set is empty** → `ErrEmpty` at `tools`.
3. **The mode is `specific` and the set declares no tool of that name** → `ErrUnresolvedReference` at `toolChoice.name`.

Rule 2 precedes rule 3 by choice, and it is the only ordering decision here with two defensible answers. With an empty set and a `specific` choice, both rules fire. Reporting rule 3 would say "the tool you named is not declared", which is true and unhelpful — no tool is declared. Reporting rule 2 says "you have declared no tools at all", which is the more fundamental fact and names the thing the caller must fix. `R-ATD-010`'s scenario `S-ATD-044` pins the choice, so a later reordering is a test failure rather than a silent behavior change.

Rule 2 is the milestone's most valuable single line: it is the combination every provider rejects, and catching it here costs a comparison instead of a network round trip. `none` against an empty set is explicitly legal — "do not call a tool" when there are no tools is coherent.

`ValidateAgainst` is **not** the only place these rules will run. AI-10.3 re-runs them at the request boundary, which is why they live on a method a request can call rather than being inlined into a constructor.

---

## 8. The schema-syntax question, recorded rather than decided

Should non-empty schema bytes be required to be syntactically well-formed JSON?

**For.** A malformed schema is a 400 from every provider. Catching it client-side is the same argument that justifies cross-validation rule 2, and `encoding/json`'s `json.Valid` is standard library and allocation-free.

**Against, and this is what carries.** Doc 0002 asks for syntactic validation explicitly on **AI-09.1's argument bytes** ("argument bytes that are not syntactically well-formed for the documented encoding") and pointedly does not on **AI-08.1's schema bytes**, whose only listed rule is emptiness. That asymmetry has a defensible reading: argument bytes are produced by a *model* and reassembled from a *stream*, so a malformed reassembly is a real and recurring defect class that AI-30 must catch; schema bytes are supplied by the caller as a literal in their own source. And `V-REQ-13` states Layer 1's role as "transports it" without qualification, while the register's trap-1 worked example draws the line at bytes-versus-meaning rather than at bytes-versus-syntax.

**Decision: not landed here.** Adding it would also make a future non-JSON schema dialect a breaking change, and the milestone that meets a real case for it can add it additively. This paragraph exists so that the absence reads as a decision rather than an oversight — which is doc 0002's own standard for a disposition ("decided here and asserted, rather than left to the first adapter to discover"). `R-ATD-004`'s scenarios pin the current behavior, so adding the rule later is a deliberate spec change.

---

## 9. How a red step is taken in a statically typed language

AI-04's convention, restated because every item follows it:

1. Write the test.
2. Add the **narrowest declaration** that makes it compile and makes the assertion fail — a constructor returning the zero value, an accessor returning nil. Never one that could pass.
3. Run, and record the failing output. **A compile error is not the red step**; it is the state before it.
4. Implement minimally, run, record green.
5. Refactor while green.

Three items are at particular risk of a stub that passes for the wrong reason, and are handled explicitly:

- **AI-08.1 item 2** (byte fidelity). A stub `Schema()` returning the stored slice passes trivially. The red step returns `nil`, and the assertion is on byte-equality *and* non-aliasing, so neither half can be satisfied by accident.
- **AI-08.2 item 3** (order determinism). A stub returning the stored slice passes trivially, and worse, an *incorrect* map-backed implementation passes a self-comparison. The red step returns `nil`; the assertion compares against the **caller's** order across many reads.
- **AI-08.3 item 2** (cross-validation). A stub returning `nil` passes the happy path. The red step returns `nil` and the first assertion is on the failing case.

The pin (AI-08.3's last item) is exempt from red-first by doc 0002's leaf anatomy and is still fully mechanical. Its bite proof is a scratch member declared without a table entry, its failure recorded, then removed.

---

## 10. File layout

| File | Contents | Forecast |
| --- | --- | --- |
| `src/ai/tool.go` | File documentation for the declaration and the name rule; `Tool`; `NewTool`; `MaxToolNameLen`; the three accessors; `toolNameRules` | ~120 lines including GoDoc |
| `src/ai/tool_test.go` | `package ai_test`. AI-08.1's items | ~200 lines |
| `src/ai/tool_set.go` | File documentation for the set and the determinism requirement; `ToolSet`; `NewToolSet`; `Tools`; `Len`; `Declares` | ~95 lines including GoDoc |
| `src/ai/tool_set_test.go` | `package ai_test`. AI-08.2's items | ~200 lines |
| `src/ai/tool_choice.go` | File documentation for the vocabulary extension; `ToolChoiceMode` and its constants; `toolChoiceModes`; `ToolChoiceModes`; `String`; `ParseToolChoiceMode`; `ToolChoice`; both constructors; `Mode`; `Name`; `ValidateAgainst` | ~175 lines including GoDoc |
| `src/ai/tool_choice_test.go` | `package ai_test`. AI-08.3's items and the pin | ~240 lines |

**Three production files, split on the leaf boundary.** Doc 0002's split trigger 4 calls that boundary the PR-chain boundary, so the commits fall where a reviewer would cut them.

**Nothing else is touched.** `validation.go`, `role.go` and `message.go` are read-only — AI-06 and AI-13 are being built concurrently against neighbouring files, and the file sets are disjoint by construction. `doc.go` is not touched: it already records trap 1, and a milestone's own contract documentation belongs on its declarations. `src/agenttest/` is not touched: its own comment reserves the first cross-package readability proof for AI-06.2, and `ai_test` is already an external package.

**The lint gotcha, recorded because it costs a run.** revive's `package-comments` rule treats a comment block attached directly above `package ai` as the package comment and rejects a second one. Every file here separates its banner from the `package` clause with a blank line — the shape `validation.go` and `role.go` already use.

---

## 11. Test plan

Eleven functions, in the order they are written. Each carries a banner comment citing its leaf ID.

| # | Leaf item | Test function | What makes it fail if the behavior regresses |
| --- | --- | --- | --- |
| 1 | AI-08.1.1 | `TestTool_NameDescriptionAndSchema_ReadBackExactlyFromAnExternalPackage` | Constructs and reads all three back; covers the empty description as legal. The walking skeleton |
| 2 | AI-08.1.2 | `TestTool_SchemaBytes_PassThroughByteIdentically` | Non-alphabetical keys and irregular whitespace; `bytes.Equal`; plus mutation of the caller's buffer and of the returned slice |
| 3 | AI-08.1.3 | `TestNewTool_BrokenConstructionRules_FailWithTheDocumentedSentinels` | Empty name, over-long name, bad characters, bad leading character, empty schema; asserts the class, the position, that no *other* class matches, the first-failure order, and that no offered value reaches the message |
| 4 | AI-08.2.1 | `TestNewToolSet_DuplicateNames_FailWithASentinelAtTheSecondOccurrence` | Two same-named declarations; asserts the class and the index; repeated runs report the same index. Also covers the appended unconstructed-tool case |
| 5 | AI-08.2.2 | `TestNewToolSet_EmptySet_IsLegal` | No arguments, an empty slice, a nil slice, and the zero `ToolSet` all yield an empty, readable set |
| 6 | AI-08.2.3 | `TestToolSet_Iteration_YieldsTheCallersOrderEveryTime` | 64 declarations, 100 reads, compared against the caller's order; plus copy-in and copy-out |
| 7 | AI-08.3.1 | `TestToolChoice_EachVocabularyMember_IsConstructibleAndReadsBack` | All four members; the named one carries its name; the payload-free constructor rejects the payload-carrying member; rendering round-trips; parsing is exact |
| 8 | AI-08.3.2 | `TestToolChoice_NamingAnUndeclaredTool_FailsWithUnresolvedReference` | Names a tool absent from a non-empty set; asserts the class and the position; the declared case succeeds |
| 9 | AI-08.3.3 | `TestToolChoice_AgainstAnEmptyToolSet_OnlyNoneIsLegal` | Auto, required and specific each fail with `ErrEmpty` at `tools`; none succeeds; the both-rules-violated case reports the first in the documented order on every run |
| 10 | AI-08.3.4 *(pin)* | `TestToolChoiceMode_DeclaredVocabulary_IsExhaustivelyTabulated` | Enumerates the constant space; asserts rendering, parse round trip, and the arity biconditional. Bites on a member declared without an entry |
| 11 | AI-08.3.5 *(appended, pin)* | `TestToolContract_NoInputCausesAPanic` | Zero values, nil slices, huge names, non-UTF-8 bytes through every exported entry point |

---

## 12. What this design deliberately does not do

- **No encoding on any type here.** Wire shapes are confined to adapters (`V-PRV-15`), and on `Tool` a marshaller is the exact mechanism `R-ATD-002` forbids.
- **No schema syntax check.** § 8, recorded with both sides.
- **No lookup that returns a declaration.** `Declares` answers membership; returning the declaration invites name resolution, which is `V-OUT-04`'s.
- **No equality and no rebuild.** AI-10.6 and AI-12 own them, at request scope.
- **No exported sentinel, error type or error variable.** `NFR-ATD-B`. Everything reports through AI-04's five landed classes.
- **No content part, message or request type.** `R-ATD-012`, and the reason the file set is disjoint from two concurrent milestones.
