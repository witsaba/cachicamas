# Explore — the validation error taxonomy

> **Change**: `cachicamas-ai-validation-errors`
> **Milestone**: AI-04 — Define the validation error taxonomy
> **Nodes**: AI-04.1 `[decision]` · AI-04.2 `[leaf]` · AI-04.3 `[leaf]`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Target package**: `backend/agent/src/ai/` (Layer 1)
> **Sources**: [doc 0002 — Layer 1 task graph](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) · [the register](../../../specs/ai-contract-vocabulary/spec.md) · [AI-02's decision](../../../specs/ai-stream-lifecycle/spec.md) · [AI-03's decision](../../../specs/ai-minimum-capabilities/spec.md) · [doc 0001 — agent stack v2](../../../../docs/architecture/0001-cachicamas-agent-stack-v2.md) · [ADR 0005](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md)
> **Predecessors**: AI-00 (module + guards), AI-01 (register), AI-02, AI-03
> **Blocks**: AI-05 … AI-13, AI-19

---

## 1. What this milestone is, in one paragraph

AI-04 is the first milestone of wave 1 and the first milestone in Layer 1 that writes Go which is not a guard. Its deliverable is small and its leverage is large: **one way for every later contract to report that a caller broke a rule.** Nine milestones — AI-05 through AI-13 — will each carry construction and composition rules, and every one of them reports through what AI-04 lands. doc 0002 schedules it first on the strength of a recorded cost: the retired plan defined this taxonomy seventeenth, after ten milestones had each invented their own sentinels, and then spent a milestone rationalizing them.

## 2. What already exists, and what it settles

### 2.1 The code

`backend/agent/src/ai/` holds a package comment and nothing else — no exported declaration exists. `backend/agent/src/agenttest/` holds a compile proof and a comment stating that the first *real* external-readability proof is AI-06.2's, on the first content part that exists. `go.mod` carries zero requires, and two tests hold it there: the deny-by-default import guard and the zero-requires pin. AI-04 is therefore stdlib-only, and `errors` plus `strings` is the whole of what it needs.

### 2.2 The vocabulary — four terms AI-04 implements rather than invents

This is the single most important input to the milestone, and it is easy to get backwards. AI-04 does **not** design a failure taxonomy. AI-01 already recorded one; AI-04 gives it a Go spelling and proves the properties the definitions already claim.

| Id | Term | The clause that binds AI-04 |
| --- | --- | --- |
| `V-FAIL-01` | caller-contract failure | "A violation of a construction or composition rule by the caller. **The caller's bug, knowable without any I/O.** Every rule violation in every Layer 1 contract reports through this one vocabulary." |
| `V-FAIL-02` | validation sentinel | "The stable, matchable value identifying which rule a caller-contract failure violated. **Matchable through at least one layer of wrapping**, so a caller can classify a failure it received indirectly." |
| `V-FAIL-03` | positional context | "The attached information naming *where* a violation occurred — which message, which content index, which tool — **without becoming a second, parallel failure vocabulary**." |
| `V-FAIL-04` | first-failure ordering | "The documented, deterministic order in which rules are checked, so that a value violating several rules reports the same one on every run. **Determinism is the property; whether to report the first violation or all of them is AI-04.1's decision.**" |

Two further rows constrain the milestone without being owned by it. `V-REQ-22` **request validation** requires validation to run "once, before any I/O" — that is what makes the lower-left cell of the register's failure grid empty, and it is why AI-04 never has to think about mid-stream delivery. `V-FAIL-13` **safe metadata**, owned by AI-19, states where the redaction posture starts: "at the first thing in the package that formats caller data — a validation failure — not at the hardening milestone." That sentence is AI-04.2 item 3 and AI-04.3 item 3, written in the register a milestone before the tests exist.

### 2.3 The boundary rule — already stated, already worked

Register § 6.3 states the rule and resolves four cases. AI-04.1 does not restate it in other words; a restatement would be a second definition and `R-AIV-001` forbids it.

> A failure is a caller-contract failure **if and only if it is decidable from the request alone, without contacting a provider.** Everything else is a provider/transport failure. The test is *decidability without I/O*, not *severity*, not *who is to blame*, and not *where in the code it was noticed*.

Already resolved there: an unrecognised model identity (→ provider/transport, because Layer 1 holds no catalog); an empty model identity (→ caller-contract); a tool choice naming a tool absent from the declared set (→ caller-contract); argument bytes malformed for the documented encoding (→ caller-contract); argument bytes that are well-formed but do not satisfy the tool's schema (→ **neither**, because trap 1 puts schema validation above Layer 1).

**What is left for AI-04.1 is not the rule but its application.** doc 0002's closing checklist asks for examples on both sides and at least one borderline case resolved. The register supplies one worked case; the decision has to earn the rest by resolving cases the register did not reach, and the useful ones are those where the rule *looks* like it points the wrong way.

### 2.4 What AI-02 and AI-03 already fixed

AI-02 fixed the delivery split at the handover of the carrier: before handover a failure is returned directly and no stream exists (`V-FAIL-11`). Every caller-contract failure is therefore, by construction, a returned value — AI-04 never touches an event. AI-03 made typed failure delivery a required capability, which means the shape AI-04 lands is not adapter-optional.

## 3. The real design tensions

Four, and three of them are genuinely open. They are the substance of AI-04.1.

### 3.1 Granularity pulls against call-site precision

`V-FAIL-02` requires the sentinel to identify "which rule" was violated. A sentinel per rule *instance* — one exported variable per rule per type — makes `errors.Is` answer the question completely and makes positional context nearly redundant. It also produces a public surface that grows with every milestone from AI-05 to AI-13 and is frozen at AI-40, and it makes the common question ("did something come back empty?") answerable only by enumerating every empty-value sentinel in the package.

The opposite extreme — a single `invalid request` sentinel — makes `errors.Is` carry no information at all, so every consumer type-asserts to learn anything. doc 0002 recommends the middle: one sentinel per **rule class**, reusable across types. The tension is real and it is not resolved by picking the middle; it is resolved by noticing that the middle only works if the *other* axis carries the missing information. Which makes checklist items 2 and 4 one decision, not two.

### 3.2 Determinism pulls against feedback quality

Aggregation is the friendlier answer for a human fixing a large request and it is well supported by the standard library since `errors.Join`. It is also the answer that quietly destroys the property `V-FAIL-04` exists to protect: a joined error answers `errors.Is(err, ErrEmpty)` with `true` when *any* member matches, so a consumer cannot distinguish "the only problem is an empty value" from "one of five problems is an empty value". And the natural implementations of aggregation — collect violations per field, then report — are precisely where a map lands and where the reported order stops being stable.

### 3.3 Positional context pulls against redaction

The obvious way to say *which tool* is to name it. A tool name is caller data. So is a role string, a model identity, and every other value a validation rule looks at. The moment a failure message quotes the offending value it becomes friendlier and it becomes the first leak surface in the package — and AI-04 is the milestone where the package first formats caller data at all. `V-FAIL-13` puts the posture here deliberately, and AI-04.3 item 3 makes it a pin so that a later refactor which "just adds the value to the message for debuggability" fails a test rather than passing review.

The interesting question is whether redaction is achieved by **discipline** (nobody puts caller data in the message) or by **construction** (the type has nowhere to put caller data, and every dynamic component of the message is filtered). Only the second survives nine downstream milestones.

### 3.4 AI-04 has no validating contract of its own

There is no message, no content part, no request. Every type AI-04 could validate belongs to AI-05 … AI-13. This is the trap the milestone has to avoid: writing speculative validators for types that do not exist, which would freeze AI-05's design from outside its own milestone and would be dead code the day AI-05 disagrees.

The honest surface is the part of the taxonomy that is *not* about any particular type: the failure value, the sentinel set, the positional path, and the mechanism that runs an ordered rule list and stops at the first violation. That mechanism is what AI-05 will call, and it is the only thing in the milestone against which AI-04.3's determinism item can be proven at all — a determinism claim needs something that decides *which rule fires*.

## 4. Prior art worth borrowing, and the parts worth refusing

**The standard library's own positional errors.** `fs.PathError`, `net.OpError`, `strconv.NumError` and `encoding/json`'s `UnmarshalTypeError` all follow the same shape: one concrete type, an unwrappable cause, and structured fields naming where the failure occurred. `errors.Is` classifies (`errors.Is(err, fs.ErrNotExist)`), `errors.As` locates. That is exactly the split checklist items 2 and 4 describe, and it is a strong signal that one type plus one sentinel set is the idiomatic answer rather than a compromise.

**The part worth refusing** is equally instructive: every one of those types embeds the offending value in its message — `open /etc/shadow: no such file or directory` names the path, `strconv.Atoi: parsing "hunter2": invalid syntax` quotes the input. That is correct for a general-purpose library whose inputs are not secrets. It is wrong for the type that will format message content, tool arguments, and system instructions. The shape is borrowable; the message policy is not.

**`errors.Join`** is the aggregate answer and is rejected on § 3.2's grounds, not on ignorance of it.

**A sentinel registry.** Nothing in the standard library validates that the error passed to a wrapper is one of a known set. Doing so is unusual, and it is the mechanical form of § 3.3's second option: if the message is rendered from a *registered* sentinel's own text rather than from whatever error was supplied, then no caller-supplied string can reach a message even if a future milestone passes one.

## 5. What the milestone must not do

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Provider status-code mapping, retryability, partial output, the terminal error event | AI-19 | Everything after a valid request leaves the process. AI-04's charter says so, and the register's grid draws the line |
| Any validator for a role, a message, a content part, a tool, a request | AI-05 … AI-13 | Those types do not exist. A validator for them is speculative design imposed from outside the owning milestone |
| The redaction *guard* — a scan proving no fixture, log or span carries a secret | AI-36 | AI-04 owns the posture at the point of formatting; AI-36 owns the adversarial repo-wide proof (`V-FAIL-14`) |
| Aggregate reporting of every violation | — | Decided against in AI-04.1, with the reversal path recorded |
| An external-package proof under `src/agenttest/` | AI-06.2 | `src/agenttest/import_compile_test.go` states in its own comment that the first real external-readability proof is AI-06.2's, on the first content part that exists. AI-04's tests live in the external test package `ai_test`, which already exercises the surface with no access to unexported state |
| Structured logging or an error code enum for the wire | doc 0003 / doc 0004 | Layer 1 returns values; it does not render them |

## 6. The vocabulary gap this exploration found

Two nouns AI-04.1 cannot state its granularity decision without, and neither is in the register.

| Proposed id | Term | Why the register lacks it, and why AI-04 cannot proceed without it |
| --- | --- | --- |
| `V-FAIL-16` | **validation rule** | The word "rule" appears *inside* `V-FAIL-01` ("a violation of a construction or composition rule"), inside `V-FAIL-02` ("which rule a caller-contract failure violated") and inside `V-FAIL-04` ("the order in which rules are checked") without ever being defined. This is `V-STR-23` **backpressure**'s situation exactly, and the register's own amendment history treats it as a defect rather than an acceptable elision |
| `V-FAIL-17` | **rule class** | doc 0002's closing checklist item 2 states the recommended default as "one sentinel per rule **class**, reusable across types, so that 'empty value' is one thing everywhere". The register has no term for the thing a sentinel is one of. Without it the decision cannot say that "a required value is empty" is one class shared by nine milestones while "a value is outside a closed vocabulary" is a different one — and a decision that cannot state its own unit of granularity has not made it |

Both take the next free `V-FAIL` ordinals after `V-FAIL-15`, both are owned by AI-04, and both defer their substance to AI-04 the way `V-PRV-08` defers the discovery mechanism — the register says what a rule class *is*, never which classes exist. Register § 9 rule 2 requires the append to land in this same pull request.

## 7. Size and shape

doc 0002's budget is "prefer less than 250 changed lines; stop and reassess before 400", and the milestone is deliberately small. The expected Go footprint is one production file and one test file in `src/ai/`, with the test file the larger of the two — which is the intended ratio for a milestone whose deliverable is a set of properties nine later milestones depend on. The `[decision]` node produces markdown only.

No split trigger fires: the three nodes are one publicly observable behavior (how a rule violation is reported), they are strictly ordered, and none of them needs a seam that does not exist.

## 8. Open questions carried into the proposal

1. Does the sentinel set need a class for "two values that cannot both hold" (a conflict), or is that a composition rule expressible as one of the other classes? *Carried: only classes with a citable case in the register or doc 0002 are landed; a conflict class has none yet, and the append discipline exists precisely so AI-10 can add one when it meets the case.*
2. Should the failure value carry the rule class as a field in addition to the unwrappable sentinel? *Carried: no — two representations of one fact drift. `errors.Is` is the single query.*
3. Does positional context need to distinguish "this element of a collection" from "this named field"? *Carried: yes, and it is one type with an optional index, not two.*
