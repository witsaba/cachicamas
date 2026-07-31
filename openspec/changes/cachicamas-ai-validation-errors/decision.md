# Decision — the validation error taxonomy

> **Change**: `cachicamas-ai-validation-errors`
> **Milestone**: AI-04 · **Node**: AI-04.1 — Taxonomy boundary `[decision]`
> **Phase**: decision (the deliverable of AI-04.1)
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Depends on**: AI-01 (`openspec/specs/ai-contract-vocabulary/spec.md`)
> **Blocks**: AI-05 … AI-13, AI-19
> **Binding vocabulary**: `V-FAIL-01` … `V-FAIL-04`, `V-FAIL-13`, `V-REQ-22`, and — appended by this change — `V-FAIL-16`, `V-FAIL-17`

---

## 1. What is being decided, and what is not

AI-01 already defined the taxonomy: `V-FAIL-01` caller-contract failure, `V-FAIL-02` validation sentinel, `V-FAIL-03` positional context, `V-FAIL-04` first-failure ordering. This node does not re-derive them and does not restate them in other words — `R-AIV-001` forbids a second definition, and a paraphrase is how a definition drifts.

What is open, and what doc 0002's closing checklist asks for, is the **application** of those four definitions:

1. Where exactly the line falls between a caller-contract failure and a provider/transport failure, applied to cases the register did not reach.
2. What a sentinel is one of — the granularity — and what that costs `errors.Is`.
3. Whether a value violating several rules reports one violation or all of them.
4. How positional context attaches to a failure without becoming a second failure vocabulary.

Each is argued below by stating the rejected alternative at its full strength first. A decision that only states its conclusion has not been made; doc 0002 requires the SDD to record *why it chose what it chose*, and the wave-0 milestones set the precedent this artifact follows.

Go spellings are **not** decided here. `V-FAIL-*` are concepts; `design.md` chooses identifiers. That separation is doc 0002's authoring constraint applied one level down: this artifact would read identically if every name in `design.md` changed.

---

## 2. Decision 1 — the boundary, applied

### 2.1 The rule is inherited, not restated

Register § 6.3, quoted rather than paraphrased:

> A failure is a caller-contract failure **if and only if it is decidable from the request alone, without contacting a provider.** Everything else is a provider/transport failure. The test is *decidability without I/O*, not *severity*, not *who is to blame*, and not *where in the code it was noticed*.

Two clauses of that sentence do real work and are routinely dropped when it is recalled from memory. **"From the request alone"** excludes facts that are true of the call but not of the request value. **"Not who is to blame"** excludes the intuition that the caller's mistakes are caller-contract failures — a caller who supplies a revoked credential has made a mistake, and it is a provider/transport failure, because Layer 1 cannot know it without asking.

### 2.2 Examples on both sides

| Case | Side | Why |
| --- | --- | --- |
| A model identity that is empty | caller-contract | Decidable from the request value. Register § 6.3 states it |
| A role outside the closed role vocabulary | caller-contract | `V-REQ-01`: "a value outside the vocabulary is a caller-contract failure, not an unknown passed through" |
| A message with no content | caller-contract | AI-05.2 item 3 already assumes this answer |
| A tool choice naming a tool absent from the declared set | caller-contract | Both halves of the fact are in the request. Register § 6.3 |
| Argument bytes malformed for the documented encoding | caller-contract | The encoding is documented by Layer 1; the bytes are in the request. Register § 6.3 |
| More cache-boundary markers than the documented breakpoint cap | caller-contract | `V-REQ-24` makes the cap a documented Layer 1 constant; see § 2.3 case B |
| A model identity the provider does not recognise | provider/transport | Layer 1 holds no catalog (`V-OUT-14`). Register § 6.3 |
| An expired or revoked credential | provider/transport | Not knowable without asking |
| A connection that fails, a rate limit, a provider-side error, a deadline exceeded on the wire | provider/transport | `V-FAIL-05` enumerates them |
| A stream that dies after emitting output | provider/transport, mid-stream | `V-FAIL-08`, `V-FAIL-12`. Delivered as the terminal error event |

### 2.3 Four borderline cases, resolved

The register resolves case A. Cases B, C and D are this decision's, and each was chosen because the rule *appears* to point the wrong way until the quoted clauses are applied literally.

**Case A — a model identity the provider does not recognise.** Resolved by the register: the request is well-formed, Layer 1 holds no catalog, so the fact is not decidable from the request alone. **Provider/transport** (`V-FAIL-05`), delivered pre-stream (`V-FAIL-11`). Cited, not re-decided.

**Case B — more cache-boundary markers than the provider actually accepts.** This looks like case A wearing a different hat: a limit that lives at the provider is not knowable from the request. The resolution turns on `V-REQ-24` **breakpoint cap**, which defines the cap as "the documented maximum number of cache-boundary markers a request may carry", existing "because at least one provider enforces a small hard cap". The cap is therefore a **Layer 1 constant**, not a provider fact, and a request exceeding it is decidable from the request plus Layer 1's own documentation. **Caller-contract.** The corollary is the part worth writing down: if a provider later enforces a cap *smaller* than the documented one and rejects a request Layer 1 accepted, that rejection is a **provider/transport failure** and not evidence that the boundary was drawn wrong — it is evidence that AI-11's documented cap is stale. The two failures are different failures with different owners, and collapsing them would make AI-11's constant unfalsifiable.

**Case C — a tool result that reports the tool failed.** The intuitive answer is that a failure is a failure. It is neither. `V-REQ-18` states it directly: "A *result that reports failure* is not a `V-FAIL-01` and not a `V-FAIL-05`; it is ordinary content." The case earns its place here because it is the one that proves the taxonomy is not a partition of everything unpleasant: Layer 1 has exactly two failure vocabularies and a third bucket — **content that happens to describe a failure** — which belongs to neither and travels as data. A rule that classified it would have Layer 1 interpreting a tool's semantics, which is trap 1.

**Case D — a context that is already cancelled or past its deadline when validation runs.** This is the sharpest case, because "decidable without I/O" appears to answer it: the deadline can be read with no network at all, so the naive reading says caller-contract. The full clause is "decidable **from the request alone**", and a cancellation signal is not part of the request — `V-REQ-20` enumerates what a normalized request holds (model identity, system-instruction segments, messages, tool set and tool choice, generation options) and a cancellation signal is not among them; `V-STR-06` makes it the caller-owned signal that ends a *stream*. An expired deadline is a property of the call, not of the value. **Provider/transport** (`V-FAIL-05`), delivered pre-stream (`V-FAIL-11`). The generalization: *decidable without I/O* is necessary but not sufficient — the fact must also be a property of the request value. AI-19 inherits this case; it is the reason a pre-stream failure is not synonymous with a caller-contract failure.

### 2.4 What this fixes for the nine milestones downstream

`V-REQ-22` requires validation to run once, before any I/O. Combined with the boundary, that gives AI-05 … AI-13 a rule they can apply without opening this document: **if the check needs anything the request value does not carry, it is not their check.** The empty lower-left cell of the register's grid is that rule's consequence, not a coincidence.

---

## 3. Decision 2 — granularity: one sentinel per rule class

**Decided: one sentinel per rule class, reusable across every type in Layer 1.** doc 0002's recommended default, taken — but the reason it is defensible is not that it is the middle option.

### 3.1 The case for one sentinel per rule instance, at full strength

Give each rule in each type its own exported sentinel: an empty model identity, a message with no content, a role outside the vocabulary, a tool choice naming an undeclared tool, each a distinct value. Then:

- `errors.Is` answers the consumer's real question completely, in one call, with no second step. A consumer that wants to react to exactly one condition writes exactly one comparison.
- Positional context becomes nearly redundant for classification, and is left carrying only indices — a genuine simplification of the harder half of the milestone.
- Every sentinel's name documents its own rule, so the failure is self-describing without a message and without a path.
- Test assertions become maximally specific: the test for AI-05.2 item 3 asserts the exact sentinel for "message has no content" and cannot accidentally pass on a different empty-value failure.
- It is what a reader coming from `errors.Is(err, fs.ErrNotExist)` expects: sentinels that name conditions, not categories.

That case is strong enough that it is worth naming what defeats it, rather than asserting a preference.

### 3.2 Why it loses

**It reproduces the recorded defect.** doc 0002's AI-04 charter opens with it: the retired plan let "ten milestones ... each invent their own sentinels" and then spent a milestone rationalizing them. Per-instance sentinels do not merely resemble that state; they *are* that state, arrived at deliberately rather than by accident.

**The public surface grows without bound and freezes at AI-40.** Nine milestones × several rules each is dozens of exported variables, every one of them permanent from the moment Layer 2 could have matched on it. `V-REQ-28`'s design position — "growing the neutral vocabulary once per provider quirk grows it without bound" — is the same argument one level down.

**The common query becomes an enumeration.** "Did some required value come back empty?" is answerable only by listing every empty-value sentinel in the package, and the list is wrong the day a milestone adds one. That is `errors.Is` degenerating into string matching by another name, which is precisely what `V-FAIL-06` records as the reason a failure classification must be closed.

**And the information is not lost.** The rule instance is the pair *(class, position)*, and the position is carried anyway because `V-FAIL-03` requires it. Per-instance sentinels encode in the type system a fact the failure value already carries as data.

### 3.3 The other extreme, and why it also loses

One sentinel for the whole category — a single "invalid request" value — is the smallest possible surface and makes `errors.Is` trivially correct. It also makes it useless: every consumer must type-assert before it learns anything, and the register's own `V-FAIL-06` reasoning applies — a classification nobody can switch on pushes classification back into message parsing.

### 3.4 The consequence for `errors.Is`, stated

Two orthogonal axes, two different queries, and no overlap:

| Question | Mechanism | Answer |
| --- | --- | --- |
| *Which rule class was violated?* | `errors.Is(err, <sentinel>)` | "A required value was empty", uniformly across every Layer 1 type |
| *Where?* | `errors.As(err, &<failure>)`, then read the position | "The content of message 2", structurally |

A consumer asking the compound question — "is *this particular* value empty?" — writes both. That is the conceded cost and it is stated again in § 6.

### 3.5 What a rule class is, and how the set grows

A **rule class** (`V-FAIL-17`, appended by this change) is the kind of thing a rule checks, independent of what it checks it on. "A required value is empty" is one class whether the value is a model identity, a message's content, or a tool's name.

The set is **closed and appended, never invented** — the same discipline the register applies to itself, for the same reason. A milestone that meets a violation no landed class describes appends a class in the pull request that needs it, rather than defining a local sentinel. This is stated as a rule here because the alternative — an open set — recreates § 3.2's defect one milestone at a time.

Only classes with a citable case in the register or in doc 0002 are landed by AI-04. A "two values conflict" class is plausible and has no citable case yet; AI-10 or AI-12 will meet it or will not, and the append rule is what makes waiting cheap. Which classes exist at merge is `design.md`'s to enumerate and `spec.md`'s to constrain.

---

## 4. Decision 3 — ordered first-failure

**Decided: a value violating several rules reports exactly one violation — the first in a documented order.** doc 0002's recommended default, taken. The order is the order in which a contract lists its rules, and it is data a reviewer can read rather than control flow they must trace.

### 4.1 The case for aggregation, at full strength

- **A caller fixing a large request gets every problem at once.** A normalized request with a bad role, an empty content part and an over-cap marker count takes one edit-run cycle to fix instead of three. On a request assembled programmatically from several sources, three cycles is three *deploys* for a consumer above Layer 2.
- **The standard library supports it directly.** `errors.Join` has existed since Go 1.20; `errors.Is` and `errors.As` both traverse joined errors, so aggregation costs no bespoke machinery and no unusual consumer code.
- **Determinism is achievable without short-circuiting.** `V-FAIL-04` requires a deterministic *order of checking*, not a single reported result; a slice appended to in rule order is exactly as deterministic as a slice with one element.
- **It is strictly more information.** First-failure is recoverable from an aggregate — take the head — while the reverse is not. Choosing the aggregate keeps both behaviors available.
- **The hot-path argument is weak on its own.** `V-REQ-22` puts validation once per request, before I/O, on a path that is about to open a socket. A slice allocation there is not measurable against a network round trip.

That last point is worth conceding openly: the allocation argument doc 0002 offers is the weakest of the reasons to prefer first-failure, and this decision does not rest on it.

### 4.2 Why it loses

**`errors.Is` stops answering the question it exists to answer.** This is the decisive ground and it is a property of `errors.Join`, not of any particular implementation: a joined error matches a sentinel when **any** member matches. `errors.Is(err, <empty-value>)` therefore cannot distinguish "the only problem is an empty value" from "one of five problems is an empty value". Decision 2 spent its whole argument on making that call meaningful across every Layer 1 type; aggregation takes it back. A consumer wanting the true answer must unwrap the join and inspect members, which is the type-assert-before-you-know-anything failure of § 3.3 reappearing.

**The same happens to `errors.As`.** With several members carrying positional context, `errors.As` yields the first match, and "the first match" is an artifact of traversal order rather than a stated contract. `V-FAIL-03`'s "without becoming a second, parallel failure vocabulary" is violated in spirit: consumers would need a way to ask for *all* the positions, which is a second extraction path.

**Determinism becomes a property of the implementation rather than of the shape.** With one reported violation, determinism is structural — the first rule that fails is the one reported, and there is nowhere for a map to hide. With a collected set, determinism depends on every future contributor accumulating in rule order across nine milestones, and the first person who validates a map of fields by ranging over it breaks it silently. `V-FAIL-04` exists because that exact thing happened before. **The property should be impossible to violate, not merely required.**

**Layer 1's caller is not a form.** doc 0002's layering puts Layer 2 above this boundary: code that assembles a request, not a human filling in fields. A caller-contract failure is a **bug**, and the standard library's own convention for bugs is one at a time — `strconv.Atoi` does not report every subsequent parse failure in a string.

### 4.3 The consequence, and the one-way door

First-failure is the restrictive choice, and it is reversible in exactly one direction: an ordered rule mechanism can grow an "every violation" sibling later **without changing the failure type, the sentinel set, or any existing consumer**, because the aggregate would be built from the same values. The reverse — narrowing an aggregate contract to a single failure after AI-40 froze the surface — breaks every consumer that iterated the members.

Taking the restrictive option first is therefore not caution; it is the only ordering that keeps both options open.

---

## 5. Decision 4 — positional context attaches inside the one failure value

**Decided: exactly one concrete caller-contract failure type exists. It always carries a position, the position may be empty, and there is never a second type to look for.**

### 5.1 The case for a separate positional type, at full strength

Wrap a sentinel in a small "where" value and compose: the failure type stays minimal, positional information is optional and pay-as-you-go, and a rule that genuinely has no position (a whole-request rule) constructs nothing extra. Composition is idiomatic Go — `fmt.Errorf("%w", ...)` layered with a typed wrapper — and it lets a future milestone add a second, orthogonal annotation (a phase, a limit) by adding a third wrapper rather than a field.

### 5.2 Why it loses

`V-FAIL-03` names the failure mode in its own definition: positional context must attach "without becoming a second, parallel failure vocabulary". Two wrapper types is two `errors.As` targets, and a consumer that wants both facts must try both and handle every combination — present, absent, and present-but-in-the-other-order. The wrapping **order** becomes observable and therefore becomes contract, which means AI-05 … AI-13 must all wrap identically and nothing enforces it.

One type ends the question. `errors.As` with one target is the only extraction any consumer ever writes, and "no position" is an empty path rather than an absent wrapper — an absence that is a value, which is the same distinction `V-MET-11` (**absence versus zero**) protects one category over.

### 5.3 The position is structural, and that is the redaction decision

A position names *where*, and the tempting spelling of "which tool" is the tool's name. **A tool name is caller data.** So is a role string, a model identity, and a content body. `V-FAIL-13` places the redaction posture exactly here — "at the first thing in the package that formats caller data — a validation failure — not at the hardening milestone" — and AI-04.3 item 3 makes it a pin.

**Decided: a position is built only from structural names and integer indices, and no caller-supplied value ever enters it.** "Which tool" is the tool's index in the ordered tool set — `V-REQ-14` guarantees the set is "ordered and deterministically iterable", so an index identifies a tool unambiguously and the caller, which owns the set, can resolve it to a name in one step.

Two further constraints make this **construction rather than discipline**, which § 3.3 of `explore.md` argued is the only version that survives nine milestones:

1. **Structural names are filtered, and a name that fails the filter is replaced whole.** Not truncated: a prefix of a secret is still a secret. The filter admits the shape of an identifier and nothing else, so a name carrying a content body, a newline, or an argument payload cannot render.
2. **The rendered message never quotes the error it was given.** It renders the text of the *registered* rule class the failure matches. A future milestone that passes a wrapped error carrying a caller string still produces a message built only from the registered class's own words, and the wrapped error remains fully matchable by `errors.Is`.

Constraint 2 is unusual — nothing in the standard library does it — and it is what makes AI-04.3's pin mechanical rather than aspirational. The refactor the pin exists to catch is "let us put the offending value in the message so it is easier to debug", and with these two constraints that refactor cannot be made without changing the type, at which point the pin fails.

### 5.4 The conceded cost

`tools[3]` is less friendly than `tools["read_file"]`, and a human reading the message must count. The trade is deliberate: the friendlier message is unbounded caller data in a string that will be logged, and AI-36's later adversarial proof would then have to find it. The failure carries the index structurally, so a consumer that wants the friendly form can render it against the request it still holds — one line at the call site, and it never reaches a log by default.

---

## 6. Conceded costs, collected

Stated together so a reviewer can weigh the decision as a whole rather than one section at a time.

| Decision | Conceded cost | Why it is acceptable |
| --- | --- | --- |
| Per-class sentinels | The compound question ("is *this* value empty?") takes two calls, not one | Both calls are cheap and neither is a type switch. The alternative is dozens of frozen exported variables |
| Per-class sentinels | The class set must be appended by later milestones, with the discipline that implies | The same discipline the register already runs, and the reason it works is that it is enforced in the pull request that needs it |
| Ordered first-failure | A caller with three violations needs three edit-run cycles | Positional context makes each cycle short, and the aggregate sibling remains addable without breaking anything |
| Ordered first-failure | The order of a contract's rules becomes contract | It is contract either way; this makes it visible and testable rather than implicit in control flow |
| One failure type | A rule with no position still carries an empty position | An empty value is cheaper to reason about than an absent wrapper, and it removes an entire class of consumer branch |
| Structural positions only | Messages are less friendly than the standard library's | The standard library's inputs are not message content, tool arguments and system instructions. Ours are |
| Rendering from the registered class | A rule class that is not registered renders generically | It is the mechanical form of the redaction posture, and an unregistered class is itself a defect the generic text makes visible |

---

## 7. Register amendment this decision requires

Per register § 9 rule 2 and `R-AIV-011`, both nouns are appended to `openspec/specs/ai-contract-vocabulary/spec.md` in this same pull request, with the next free `V-FAIL` ordinals and a dated amendment blockquote under § 6.

| Id | Term | Owner |
| --- | --- | --- |
| `V-FAIL-16` | **validation rule** | AI-04 |
| `V-FAIL-17` | **rule class** | AI-04 |

The justification for each is in `proposal.md` and in `explore.md` § 6. No existing row is renumbered, reworded or removed; the register's term counts are updated in the same edit.

---

## 8. What each blocked milestone inherits

Stated in each milestone's own terms, so doc 0002's acceptance criterion — "every construction and validation rule in AI-05 … AI-13 reports through this taxonomy" — is checkable from one table.

| Milestone | What it receives | What it still decides |
| --- | --- | --- |
| **AI-05** | The failure value, the class set, and the ordered-rule mechanism. Its charter's "fails with an AI-04 sentinel" (items AI-05.1.2, AI-05.2.3) is satisfiable without inventing anything | Which rules a message has, and their order |
| **AI-06** | The same, plus the ruling that a construction bypass is a caller-contract failure decidable from the value | The part strategy; whether a rule class for "an unconstructed value" is needed and must be appended |
| **AI-07**, **AI-08**, **AI-09** | Positional context that already names a content-part index and a tool index | Their own rules and orders |
| **AI-10** | The boundary rule in the form of § 2.4: a check needing anything outside the request value is not AI-10's | Where `V-REQ-22`'s single validation pass runs |
| **AI-11** | Case B: the documented breakpoint cap is a Layer 1 constant, and exceeding it is caller-contract | The cap's value, and its staleness policy |
| **AI-12** | The escape hatch carries opaque values; a malformed namespace is caller-contract, an unrecognised one is its adapter's | Whether a namespace rule needs a new class |
| **AI-13** | Nothing new — finish reasons and usage are read, not validated | — |
| **AI-19** | The complement, exactly: everything this decision assigns to the other side, plus case D (an expired call signal is pre-stream but not caller-contract) and case B's corollary (a provider cap smaller than the documented one) | The failure categories, retryability, partial output, the terminal event |
| **AI-36** | A redaction posture that is already structural at the point of formatting, so its adversarial scan is a proof and not a repair | The repo-wide scan (`V-FAIL-14`) |

---

## 9. Closing-checklist verification

doc 0002's four items for AI-04.1, each against this artifact.

| # | Closing-checklist item | Where answered | Status |
| --- | --- | --- | --- |
| 1 | The line between a caller-contract and a provider/transport failure, with examples on both sides and at least one borderline case resolved | § 2 — the rule quoted from register § 6.3, ten examples split across both sides, and **four** borderline cases: A cited from the register, B, C and D resolved here | **answered** |
| 2 | Granularity decided — one sentinel per rule, per type, or per category — with the consequence for `errors.Is` stated | § 3 — one per **rule class**; the per-instance case argued at full strength in § 3.1 and defeated in § 3.2; the per-category extreme in § 3.3; the `errors.Is` consequence as a two-axis table in § 3.4 | **answered** |
| 3 | Aggregate versus short-circuit decided, with the consequence | § 4 — **ordered first-failure**; the aggregation case argued at full strength in § 4.1, including the concession that doc 0002's allocation argument is its weakest; defeated in § 4.2 on `errors.Is` semantics, `errors.As` traversal order, structural determinism, and the nature of Layer 1's caller; the one-way door in § 4.3 | **answered** |
| 4 | How positional context attaches without becoming a second, parallel error type | § 5 — one concrete type with an always-present, possibly empty position; the composed-wrapper alternative argued in § 5.1 and defeated in § 5.2 on `V-FAIL-03`'s own words; the structural-position and no-quoting constraints in § 5.3 that make redaction construction rather than discipline | **answered** |

**Node type discipline.** AI-04.1 is a `[decision]` leaf and produces no production code. The Go surface it constrains is chosen in `design.md` and landed by AI-04.2 and AI-04.3, whose evidence gate is recorded green `make test` in `backend/agent/`.
