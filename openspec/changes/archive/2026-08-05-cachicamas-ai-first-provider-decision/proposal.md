# Proposal — the first provider and its transport

> **Change**: `cachicamas-ai-first-provider-decision`
> **Milestone**: AI-24 — Select first provider and transport
> **Nodes**: AI-24.1 — The provider decision `[decision]` · AI-24.2 — The transport decision `[decision]`
> **Status**: proposed
> **Phase**: proposal
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-03
> **Driver**: braejan
> **Scope**: documentation only — `openspec/changes/cachicamas-ai-first-provider-decision/`, plus dated amendments to doc 0002. **Zero Go. Zero `go.mod` change. Zero files under `backend/`.**
> **Predecessor artifact**: Engram `sdd/cachicamas-ai-first-provider-decision/explore`
> **Depends on**: AI-03, AI-11, AI-12, AI-23
> **Blocks**: AI-25 … AI-32
> **Node grammar**: `[decision]` — "a recorded choice with a closing checklist. No production code." It closes when the artifact answers every listed question and is merged.

---

## Intent

Close doc 0002's AI-24.1 and AI-24.2 closing checklists in one merged `decision.md`, so that AI-25 … AI-32 can be planned and built against a **named** vendor dialect, a **named** transport, a **named** streaming framing, and a **named** credential boundary — none of which any of those eight milestones is authorized to invent for itself.

**The choice itself is already made and is not reopened here.** After being shown the alternatives and their costs, the driver selected:

| Axis | Chosen |
| --- | --- |
| Vendor / dialect | **OpenAI-compatible Chat Completions streaming dialect** |
| Transport | **Raw `net/http`** — no vendor SDK, **zero `go.mod` requires** |

This change's job is to *record and ground* that choice against evidence, price what it costs honestly, and hand each blocked milestone something concrete. A decision artifact that only states a verdict cannot be extended, audited, or reversed on grounds anyone can check.

## Locked constraints (inherited, not proposed)

Listed so a reviewer can tell the inherited from the decided.

1. **The verdict is the driver's.** Vendor and transport arrive as input to this change, not as its output. The artifact supplies argument, not the choice.
2. **The five required capabilities are a floor on vendor selection.** `ai-minimum-capabilities` § 12: "A candidate vendor that cannot support all five cannot be the first adapter." This change confirms the floor is cleared; it does not re-decide the floor.
3. **Only `satisfied` and `absent` are legitimate *expectations*.** `ai-minimum-capabilities` § 10 closes the outcome set at four values; § 12 states that `failed` and `not exercised` are run results and never predictions. The expected report obeys this.
4. **The record is total over both closed lists.** One entry per capability across `CAP-R-01 … CAP-R-05` and `CAP-O-01 … CAP-O-03`, standing taken from AI-03's decision and never from a run (§ 10).
5. **Amendments are dated blockquotes, never silent edits.** doc 0002's revert-and-record clause rule 4: `> **Amended YYYY-MM-DD** …` under the touched heading, struck-through text for superseded claims, landing in the same PR.
6. **Identifiers are append-only.** A node ID is never retrofitted into an archived milestone; a new obligation takes the next free ordinal and draws its edges.
7. **No Go identifiers, no production code.** doc 0002's `[decision]` node grammar and the authoring constraint that bound every Wave 0 decision artifact.

## Scope

### In scope — one PR

| Artifact | Content |
| --- | --- |
| `proposal.md` | This file |
| `specs/ai-first-provider-decision/spec.md` | `R-APD-0NN` — each a property of the decision artifact checkable by inspection, without running anything |
| `design.md` | The structure `decision.md` implements, and the evidence rules it applies |
| `tasks.md` | Two leaves, one task per closing-checklist item (4 + 3), plus the doc 0002 amendment block and the verification pass |
| `decision.md` | **The deliverable.** Vendor, transport, framing, credential boundary, the expected capability report, and what AI-25 … AI-32 each inherit |
| doc 0002 | **Amended** — one dated block, six entries (below) |

### The decisions this change commits to

One line each, so a reviewer can accept or reject the substance before reading the argument.

1. **Vendor: the OpenAI-compatible Chat Completions streaming dialect.** Rejected: Anthropic Messages API native, and either vendor's Go SDK — each recorded across all seven charter axes (capability fit, streaming quality, testability, dependency weight, endpoint configurability, maintenance, credential-handling boundary).
2. **Transport: raw `net/http`.** `backend/agent/go.mod` carries **zero `require` lines**, verified by command. `net/http` is standard library, so AI-24.2 adds no dependency and **the ADR gate of `openspec/AGENTS.md` rule 5 resolves to a no-op**. This is the charter's acceptance clause discharging by not firing — recorded as decided, never as forgotten.
3. **The AI-00.3 forward guard gains no second allowlist entry.** Its comment names AI-24 as the milestone expected to add one; under this decision that entry is never added and the guard stays green with zero changes. Recorded as an outcome of the decision.
4. **Four cross-provider divergences, answered.** Caching is **automatic** (no client breakpoint marker) · tool results are a **distinct `role: "tool"` message** · an explicit output-token limit is **optional**, not mandatory · the vendor **does assign** tool-call identifiers, on the call's opening delta.
5. **Two further questions, answered.** Tool-call arguments stream **in fragments** keyed by `index`, which weights AI-30 toward fragmented accumulation as the primary case · the dialect **signs no reasoning blocks** — only an opaque `reasoning_tokens` count inside usage.
6. **Framing: SSE per the WHATWG HTML Living Standard § 9.2, data-only, with a `data: [DONE]` terminal sentinel.** Field grammar, multi-line `data:` LF joining, comment lines, BOM and line-ending handling are cited to the spec. **`[DONE]` and the data-only shape (no `event:` type lines) are dialect conventions, not SSE-spec-mandated** — so AI-27 must pin them by explicit fixture and must never infer them from the spec.
7. **Credential boundary.** The adapter receives an **opaque bearer-token value through injected construction** — never a variable name, never a path. It reads no environment variable, opens no file, spawns no process. The value's origin is Layer 3's composition root, out of Layer 1 entirely.
8. **The expected capability report**, total over both closed lists: `CAP-R-01 … CAP-R-05` expected **satisfied** (the floor, cleared); `CAP-O-01`, `CAP-O-02`, `CAP-O-03` expected **absent**, each with its basis.

### The cost of the choice, priced rather than hidden

Two contract items lose their first real consumer, and the artifact says so plainly:

- **AI-07's reasoning round-trip token** — the dialect exposes no signed reasoning block, so no v1 adapter exercises the signature-preservation path. The neutral shape stays contract-mandatory regardless of who emits it.
- **AI-11's breakpoint cap (`V-REQ-24`)** — automatic caching means there is no vendor cap to enforce, so the adapter-side enforcement scenario has no subject. AI-11.3's advisory contract is exercised instead, by dropping markers whole.

### One trap recorded for the downstream milestones

The dialect emits in-stream `usage` **only when the request sets `stream_options.include_usage: true`**. Unset, usage returns empty — an `absent` outcome disguising an adapter bug rather than a vendor limitation. Recorded as an AI-26.7 / AI-28.3 / AI-31.2 implementation obligation.

### The living-graph amendments this decision forces

Six entries, one dated block, all landing in this PR. **Proposed here, written at apply time.**

| # | Target | Amendment |
| --- | --- | --- |
| 0 | AI-24 charter · milestone rules bullet on `go.mod` | The ADR gate resolved to a no-op: no dependency added, so the module stays at zero requires until AI-37, not until AI-24 |
| 1 | **AI-25.2 item 1** | **Load-bearing correction.** "An AST import-and-call scan in the AI-00.3 style" is wrong: AI-00.3 is an **import-path** scan (`go list -deps -test` against a deny-by-default allowlist), and `net/http` transitively imports `os`. Reusing that mechanism would either false-positive on legitimate use or miss a narrow `os.Getenv` call. AI-25.2 needs genuine **`go/ast` selector-expression scanning scoped to the adapter's own source files** |
| 2 | **AI-26.5 item 2** | Synthetic tool-call-identifier minting has **no subject** for this adapter — the vendor assigns real IDs. The requirement text stays (a future adapter needs it); this adapter's conformance exercises only the vendor-assigns branch, marked not-applicable rather than silently skipped |
| 3 | **AI-26.2 item 3** | The vendor-cap branch has **no subject** — caching is automatic. Item 2's "or dropped whole" is the branch this adapter takes, exercising AI-11.3's advisory contract |
| 4 | **AI-26.7 item 2** | The mandatory-output-limit branch is a **deliberate no-op**, recorded as such per the node's own instruction |
| 5 | **AI-29.0** | Note only: the emit-versus-absence decision is now **strongly indicated toward absence**. AI-29.0 still makes it, against the exact backend chosen for AI-38/AI-39 — some OpenAI-*compatible* servers emit a non-standard `reasoning_content` extension field. **AI-24 does not pre-empt it and does not delete the node** |

### The two Wave-2 carryovers — a recommendation, not a menu

The `CheckEmit` rule 4 failure-path gap and the missing redacting `*Failure.GoString()` are recorded as unassigned at `openspec/specs/ai-stream-testkit/spec.md`, now for a second consecutive wave. They cannot be retrofitted into archived Wave-2 milestones.

**Recommendation: append one new milestone `AI-41 — Discharge the Wave-2 carryovers` with two leaves, scheduled in Wave 5, and carry the append in this PR's amendment block.**

- **`AI-41.1`** — `CheckEmit` rule 4's failure path is exercised: an event the fourth rule itself rejects.
- **`AI-41.2`** — the AI-19 provider-failure payload gains a redacting `GoString()`, matching the pattern `Completion` and `Reasoning` already carry, so `%#v` cannot fall back to reflection over unexported fields.
- **Edges**: `Blocks: AI-36`. AI-36's adversarial sweep would otherwise sweep a type still missing its redacting method; ordering AI-41 first resolves the overlap rather than letting two nodes own one behavior.

**Wave 5, not Wave 4, and stated plainly.** Neither item blocks any Wave-4 node — AI-24 … AI-32 touch neither `CheckEmit`'s failure path nor `%#v` on failures — and Wave 4 already forecasts 20 000–25 000 changed lines against a 5 000-line review budget. Absorbing them into Wave 4 buys nothing and costs review focus. But **the assignment must land now**: a third silent carryover is the failure mode, and appending the nodes in this PR costs roughly twenty-five lines of markdown and zero Go.

### Out of scope

| Excluded | Owner | Why not here |
| --- | --- | --- |
| Any Go code, of any kind | AI-25 onward | `[decision]` node grammar: no production code |
| Adapter construction, configuration, the ambient-authority guard | AI-25 | This decision states the boundary AI-25.2 enforces; it writes no guard |
| Request translation and every wire fixture | AI-26 | This decision names the dialect; it maps nothing |
| The frame decoder and its fixtures | AI-27 | This decision names the framing precisely enough for AI-27 to encode it |
| Whether the adapter emits reasoning | **AI-29.0** | Indicated, explicitly **not** decided. Deleting a downstream decision node is the failure this clause prevents |
| The generated capability report | AI-38.2 | This decision records the **expected** one; AI-38.2 asserts the real one against it |
| Correcting the AI-00.3 guard's code comment | AI-25 | It is Go source. This change ships zero Go; the outcome is recorded here and the comment corrected by the first milestone that opens that package |
| Discharging the two carryovers | AI-41 | Assigned here, worked there |

## Capabilities

> Contract between this proposal and the spec phase.

### New Capabilities

- `ai-first-provider-decision`: the recorded first-provider and transport choice — vendor with rejected alternatives across seven axes, four divergences, two further questions, the expected capability report, the streaming framing contract, and the credential-handling boundary. The system under test is **the artifact**, as it was for AI-01, AI-02 and AI-03.

### Modified Capabilities

- **None.** `ai-minimum-capabilities`, `ai-cache-breakpoints`, `ai-provider-failures` and `ai-stream-testkit` are read and **cited by identifier, never modified**. No requirement of any existing spec changes. The `ai-stream-testkit` carryover note becomes stale on merge; correcting its status line is a documentation-consistency edit tracked under Affected areas, not a requirement delta.

## Approach

1. **Ground every axis, do not assert it.** Each of the seven charter axes gets one row per candidate, so the rejection of Anthropic-native and of both SDKs is checkable rather than asserted. An artifact that records only a verdict cannot be reversed on grounds anyone can inspect.
2. **Price the losses in the same document as the win.** AI-07's signature path and AI-11's cap enforcement lose their first consumer. Stating this in the decision — not in a later post-mortem — is what makes the choice honest and what stops a later reader treating those milestones as dead.
3. **Separate spec-mandated from dialect-conventional.** Every framing claim is labelled with its source: WHATWG § 9.2, or observed vendor behavior. `[DONE]` and the data-only shape are the two that must be fixture-pinned at AI-27, and mislabeling them is how a decoder ends up "conforming" to something no spec says.
4. **Write the expected report in § 10's vocabulary, totally.** Eight entries, standing from AI-03, expected outcome drawn only from `satisfied` / `absent`. AI-38.2 compares entry by entry; a difference in either direction is a finding.
5. **Amend the graph in the same PR, dated.** Six entries, struck-through where superseded. The AI-25.2 correction is flagged first because it is a real defect that would otherwise ship as a guard that cannot bite.
6. **Close by inheritance.** The artifact ends with what AI-25 … AI-32 each receive, in that milestone's own terms, so the acceptance criterion — writable without reopening AI-24 — is checkable from one table.

## Affected areas

| Area | Change | Risk |
| --- | --- | --- |
| `openspec/changes/cachicamas-ai-first-provider-decision/` | Five new markdown files | None — new directory |
| `docs/architecture/milestones/0002-…-task-graph.md` | One dated amendment block, six entries, plus the AI-41 append | Low — append-only, dated, no silent edit |
| `openspec/specs/ai-stream-testkit/spec.md` | One status line: the carryovers are no longer unassigned | Low — prose status, no requirement touched |
| `backend/agent/`, `go.mod`, `go.work` | **None** | — |
| `docker-compose.yaml`, `infra/` | **None** | — |

## Risks

| Risk | Likelihood | Impact | Mitigation |
| --- | --- | --- | --- |
| AI-25.2 is built as an import-path scan, copying AI-00.3 verbatim | **High** if unamended | **High** — a guard that cannot bite is worse than no guard, because it is believed | Amendment 1, flagged as load-bearing, with the `net/http` → `os` transitive-import reason stated |
| The `[DONE]` sentinel or the data-only shape is treated as SSE-spec behavior | Medium | Medium — AI-27's decoder conforms to something no spec says, and breaks on a compliant server | Every framing claim is labelled with its source; the two dialect conventions are named as fixture-pin obligations |
| `stream_options.include_usage` is left unset | Medium | **High** — usage returns empty and `CAP-R-03` reads `absent`, which is illegal for a required capability and hides an adapter bug as a vendor limitation | Recorded as an explicit implementation obligation on AI-26.7 / AI-28.3 / AI-31.2 |
| AI-24 pre-empts AI-29.0 by "obviously" recording absence | Medium | Medium — a decision node is deleted, and the `reasoning_content` extension case goes unchecked against the actual backend | Amendment 5 is a note, not a verdict; the out-of-scope table names AI-29.0 as owner |
| Three expected `absent` outcomes are read as an under-powered vendor | Low | Low | `ai-minimum-capabilities` § 6: an adapter lacking every optional capability is **fully conformant**, with three recorded absences |
| The carryovers pass a third wave | **Medium** | Medium — the compounding failure this proposal exists to stop | AI-41 appended in this PR, with a `Blocks: AI-36` edge that makes silence structurally impossible |
| The zero-dependency outcome reads as an oversight | Medium | Low | Stated as a discharged acceptance clause, in both the charter amendment and the decision body |

## Rollback plan

Additive documentation with one append-only, dated amendment. Rollback is `git revert` of the single commit; nothing is generated from these files, nothing imports them, no build depends on them.

Partial rollback has the shape AI-02 and AI-03 both recorded and the same answer: reverting only the doc 0002 amendments would leave `decision.md` citing node states that no longer match the graph. If the amendment block is rejected in review, the correct move is to reject the whole change and re-propose — not to strip the block.

Post-merge reversal is the expensive direction, and that is the point of the schedule. Once AI-25 builds against the credential boundary, AI-26 against the dialect, and AI-27 against the framing, changing the vendor changes an adapter, a fixture tree, a decoder and a published capability matrix at once.

## Dependencies

- **AI-03** (`cachicamas-ai-minimum-capabilities`) — **hard.** Supplies the closed capability lists, the four-value outcome set, the record's shape, and the rule that only `satisfied` and `absent` are legitimate expectations.
- **AI-11** (`cachicamas-ai-cache-breakpoints`) — **hard.** `CAP-O-03`'s expected outcome turns on AI-11.3's advisory contract.
- **AI-12** — **hard.** AI-26.7's option surface is what the optional-output-limit answer is stated against.
- **AI-23** — **hard.** The conformance suite is the consumer of the expected report.
- **No new Go dependency. No ADR required** — `net/http` is standard library, and `backend/agent/go.mod` stays at zero requires.

## Success criteria

- [ ] All four AI-24.1 closing-checklist items are answered in `decision.md`.
- [ ] All three AI-24.2 closing-checklist items are answered in `decision.md`.
- [ ] One vendor is named, with rejected alternatives recorded across **all seven** charter axes.
- [ ] The four cross-provider divergences and the two further questions each carry an explicit answer and its consequence for the named downstream node.
- [ ] The expected capability report is **total** over both closed lists — eight entries — with every expected outcome drawn only from `satisfied` and `absent`, and the five required capabilities confirmed cleared as a floor on vendor selection.
- [ ] The ADR gate is recorded as **resolved to a no-op**, with the zero-requires fact and the untouched AI-00.3 allowlist stated as outcomes.
- [ ] The streaming framing is named precisely enough for AI-27 to encode fixtures, with each claim labelled spec-mandated or dialect-conventional.
- [ ] The credential boundary states what the adapter receives, what it never reads, and where the value comes from — concretely enough for AI-25.2's guard to enforce.
- [ ] doc 0002 carries one dated amendment block with six entries, struck-through where superseded, plus the AI-41 append and its `Blocks:` edge.
- [ ] The change adds five markdown files and amends two existing files, append-only. **No Go identifier appears anywhere; no file under `backend/` is touched.**

## Notes for the following phases

- **`spec.md`** — the system under test is the artifact. Requirement IDs `R-APD-0NN`, scenario IDs `S-APD-0NN`, every scenario checkable by inspection without running anything. Several requirements must constrain the *argument* rather than the conclusion — the seven-axis comparison, the priced losses, the source label on every framing claim — because a list of verdicts with no reasons passes a spec that checks only verdicts, and cannot be extended consistently.
- **`design.md`** — owns the structure of `decision.md` and its evidence rules: the seven-axis table, the spec-versus-dialect labelling rule, and the totality rule that makes the expected report comparable to AI-38.2's generated one.
- **`tasks.md`** — seven tasks (four for AI-24.1, three for AI-24.2), plus the amendment block and the verification pass. Zero red-green phases: `[decision]` nodes ship no code and are exempt from the evidence gate's `make test` clause, closing instead on a merged artifact that answers every checklist item.
- **`decision.md`** — the deliverable. Ends with the inheritance table for AI-25 … AI-32, because the acceptance criterion is stated in terms of what those milestones can do without reopening this one.
