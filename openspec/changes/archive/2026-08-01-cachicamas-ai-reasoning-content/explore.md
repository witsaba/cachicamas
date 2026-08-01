# Explore — reasoning content and its round-trip token

> **Change**: `cachicamas-ai-reasoning-content`
> **Milestone**: AI-07 — Define reasoning content with its round-trip token
> **Nodes**: AI-07.1 `[leaf]` · AI-07.2 `[leaf]` · AI-07.3 `[leaf]` · AI-07.4 `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Binding inputs**: [doc 0002 § AI-07](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [doc 0001 § 3.2, § 3.3 row 2, § 6 seam 11](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [the register](../../../specs/ai-contract-vocabulary/spec.md) `V-REQ-09`, `V-REQ-10`, `V-REQ-11` · [AI-06's decision](../2026-08-01-cachicamas-ai-content-parts/decision.md)

---

## 1. What this milestone is, in one paragraph

Add the **second** content-part kind. AI-06 decided the shape of a content part once, for every variant, and wrote § 9 of its `decision.md` as a table "so that AI-07, AI-09 and AI-10 can cite this row instead of re-deriving the answer". This milestone cites it. What is genuinely new is the payload: a **state**, optional **text**, and an **opaque round-trip token** that cachicamas stores and returns byte-identically and never parses. Doc 0001 puts that token in the strongest terms the document uses about a field: *"This is not optional metadata; it is a correctness requirement, and the storage has to exist before the adapter that produces it."* At least one provider signs reasoning blocks cryptographically; a signature returned inexactly fails multi-turn extended thinking with tool use — on the **second** turn of a tool-using conversation, which is why it is invisible until it is expensive.

## 2. What already exists, and what it settles

### 2.1 The landed code

| File | What AI-07 takes from it |
| --- | --- |
| `src/ai/content_part.go` | `Part`, `PartKind` and its declared constant space, `partKindNames`, `partPayload`, `Part.validate`, `under`, `firstViolation`. Adding a kind is three appends here plus one new file |
| `src/ai/text_content.go` | The one-file-per-kind layout, and the precedent for every question this milestone will ask: emptiness before bound, a documented sanity bound that is not a provider limit, "stored byte for byte, nothing normalizes" |
| `src/ai/validation.go` | Five rule classes and one `*Violation` type. AI-07 adds none — § 6 |
| `src/ai/content_part_registry_test.go` | AI-06.4's guard. Declaring `PartKindReasoning` without completing the five-step procedure makes it fail, which is a red step this milestone gets for free |
| `src/ai/message.go` | `NewMessage`, `Message.Content()` and their copy semantics — AI-07.3 item 2's subject |

### 2.2 The three register terms AI-07 implements rather than invents

- `V-REQ-09` **reasoning content** — "the content-part kind carrying a model's intermediate reasoning: a state, optional text, and an opaque round-trip token. Distinct from text content at every layer — reasoning is never rendered as, merged into, or substituted for text."
- `V-REQ-10` **reasoning state** — "the value distinguishing the shapes reasoning can take: full text, redacted, signature-only, or a provider that emitted no reasoning text at all. Exists so that *no reasoning text* is a recorded state rather than an empty string."
- `V-REQ-11` **round-trip token** — "an opaque provider-supplied value attached to reasoning content, which cachicamas stores and returns **byte-identically** and never parses, reformats, re-encodes, or interprets."

### 2.3 The defect this milestone must not re-create

Doc 0001 § 3.1 records that the retired Layer 1 shipped **two strategies for one contract**: a sealed-but-unreadable wrapper for three payload-carrying kinds, and a readable-but-unsealed direct implementation **for reasoning** — the two "had to be reconciled". Reasoning was the kind that broke the pattern last time. The single most important property of this change is therefore negative: **it introduces no second strategy.** Everything below is AI-06's shape with a different payload in it.

---

## 3. The four questions this milestone actually has to answer

AI-06 answered the shape. These four are AI-07's own, and `design.md` resolves each.

### 3.1 How is a token's **absence** distinguished from an **empty** token?

Doc 0002 AI-07.2 item 1 calls it "the distinction a naive design collapses and an adapter cannot recover". The naive design is one `[]byte` field where `nil` and `[]byte{}` are both "nothing to see". The distinction is load-bearing on the wire: a provider that returned `"signature": ""` and a provider that returned no `signature` field at all are two different facts, and the return trip must reproduce the one that happened. AI-06's `decision.md` § 6.2 already pointed at the answer — *"the `ok` bool is the same distinction `V-REQ-11` will need for 'absent is not empty'"* — so the accessor shape is settled and what remains is the storage and the door.

### 3.2 Is the reasoning **state** supplied, or derived?

AI-06's `decision.md` § 5 made the kind **derived from the payload** so that "the two cannot disagree, because there is only one of them". The same question arrives one level down: a supplied state is a second source of truth about the same payload, and a payload whose state says *redacted* while carrying plaintext is a state that needs a rule class to report — and AI-04's five do not name a contradiction between two fields. **Deriving the state costs nothing and removes the rule.** The only bit that cannot be derived from the bytes is redaction itself: "the provider withheld the plaintext" is a provider fact, not a shape of the data, so it needs a door of its own rather than a flag a caller sets.

### 3.3 What is the closed state vocabulary, given that the register names four shapes and doc 0002 names three?

`V-REQ-10` lists "full text, redacted, signature-only, or a provider that emitted no reasoning text at all". Doc 0002's AI-07.1 item 3 requires three constructible states: "with text, redacted, and token-only". The two agree, and AI-07.4 item 2 is the sentence that reconciles them: *"A reasoning part with a token but no text is constructible and valid — **the signature-only shape, and** the 'this provider emitted no reasoning text' state."* One shape, described twice. The vocabulary is three members.

### 3.4 What may this package do to a token, and what must it not?

Nothing, past a documented sanity bound. Doc 0002 AI-07.2 item 2: "nothing interprets, validates, normalizes or length-caps the token beyond a documented sanity bound — proven by constructing tokens that are not valid UTF-8, not valid JSON, and not printable." The bound is the same device `MaxTextLen` already is: it makes an unbounded value decidable from the request alone, and it is deliberately not a provider limit, because Layer 1 holds no model catalog (`V-OUT-14`).

---

## 4. Prior art — how the token is usually got wrong

| Design | Where it breaks |
| --- | --- |
| Store the signature as a `string` field on a struct with exported fields | The zero value is a valid reasoning block with an empty signature — doc 0001's **C1**, one milestone after it was closed |
| Store it in a `map[string]any` of provider metadata | Nothing types it, nothing bounds it, and a round trip through a generic map is where a `[]byte` becomes a base64 `string` and back — the exact re-encoding `V-REQ-11` forbids |
| Drop it on the neutral hop and re-request | The signature is over content the provider generated; it cannot be re-derived. This is the failure that shows up on turn two |
| Keep the caller's `[]byte` by reference | Aliasing. The caller reuses its decode buffer and the stored token changes with nobody writing to the part. AI-08 hit exactly this on schema bytes |
| Normalize "for safety" — trim, validate UTF-8, cap the length | Each is a byte the provider will not accept back. The token is not text and this package cannot know what a byte means to a signer |

## 5. The budget forecast

Doc 0002's rule is "prefer less than 250 changed lines; stop and reassess before 400", and the forecast belongs here rather than in the retrospective.

| Slice | Forecast |
| --- | --- |
| `reasoning_content.go` — the state vocabulary, the payload, two doors, three accessors, the rules | ~200 lines, of which ~120 are GoDoc |
| `reasoning_content_test.go` — four leaves, twelve test-list items | ~400 lines |
| `content_part.go`, `content_part_registry_test.go` — the appends the five-step procedure requires | ~30 lines |
| SDD artifacts | ~700 prose |

**It will overrun, and the overrun is in the tests.** Four of the twelve test-list items are byte-class properties over a value this package promises never to touch, and a property of that kind is proven by enumerating classes, not by asserting once. Split trigger 1 does not fire: the four leaves are one contract, and the token has no meaning without the state it hangs on. `tasks.md` records the reassessment against the actual figure.

## 6. Vocabulary check — does AI-07 need a register amendment?

**No.** `V-REQ-09`, `V-REQ-10` and `V-REQ-11` define reasoning content, its state and its token, and each names AI-07 as owner. Nothing below introduces a Layer 1 noun the register lacks.

**No new rule class either.** The rules this milestone needs are *a required value is empty* (`ErrEmpty`) and *a value outside a documented bound* (`ErrOutOfRange`), both landed by AI-04. The one violation that would have needed a class the five do not name — a state that disagrees with the payload it describes — is removed by § 3.2's answer rather than reported: it is not a state the type can be in. Two candidate terms were considered and rejected: *"token presence"* (`V-REQ-11`'s "attached to reasoning content" already carries it, and the register's scope rule excludes Go shapes) and *"redaction"*, which is `V-REQ-10`'s "redacted" wearing a noun.

## 7. What this milestone deliberately does not do

| Not done | Owner |
| --- | --- |
| Survival of the token through request rebuild | AI-12.1 |
| Survival of the token through the wire, in either direction | AI-26.6, AI-29.2 |
| Reasoning **block** and **delta** events on a stream | AI-17 |
| Any interpretation of a token — parsing, verifying, decrypting, counting | nobody; `V-REQ-11` forbids it at every layer |
| Reasoning **token counts** in usage | AI-13.3, a different meaning of the word "token" |
| Whether an assistant message may carry reasoning and a user message may not | AI-10.3 |
| Tool call and tool result kinds | AI-09, in flight concurrently |
