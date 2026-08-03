# Decision — the first provider and its transport

> **Change**: `cachicamas-ai-first-provider-decision`
> **Milestone**: AI-24 — Select first provider and transport
> **Nodes**: AI-24.1 — The provider decision `[decision]` · AI-24.2 — The transport decision `[decision]`
> **Status**: decided
> **Date**: 2026-08-03
> **Project**: cachicamas (witsaba) · **Target package**: none — `[decision]` nodes ship no code
> **Closes**: doc 0002's AI-24.1 (four items) and AI-24.2 (three items) closing checklists
> **Sources**: [doc 0002 — Layer 1 task graph](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md) § AI-24 · [`openspec/AGENTS.md`](../../AGENTS.md) rule 5 (the ADR gate) · [the v1 capability set](../../specs/ai-minimum-capabilities/spec.md) · [cache-boundary markers](../../specs/ai-cache-breakpoints/spec.md) · [ADR 0005 § D1](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2), [§ D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary)
> **Binding predecessor**: [AI-03's decision](../archive/2026-07-31-cachicamas-ai-minimum-capabilities/decision.md) — the closed capability lists, the four-value outcome set, and § 10's totality rule are cited here, never re-decided

> [!IMPORTANT]
> **This artifact decides standings and evidence, not code.** No Go type name, field name, method name, or package identifier belonging to the Layer 1 contract appears here. Naming the standard library package that constitutes the chosen transport (`net/http`), the module manifest, and the vendor's wire-level field names is permitted, because both closing checklists require them (`R-APD-019`).

---

## 1. How to use this document

**If you are writing AI-25:** § 11 is the credential boundary your guard enforces. § 12's AI-25 row states the corrected guard mechanism and why the old one would not have bitten.

**If you are writing AI-26:** § 6 answers all four cross-provider divergences for the chosen vendor. § 12's AI-26 row states the consequence for each of your sub-nodes.

**If you are writing AI-27:** § 10 is your fixture spec, split into what the specification states (§ 10.1) and what only this vendor does (§ 10.2). Encode both; never infer § 10.2 from § 10.1's citation.

**If you are writing AI-28, AI-29, AI-30, AI-31 or AI-32:** § 12 states what you inherit. § 13.1 is a trap that binds AI-28.3 and AI-31.2 particularly, and § 13.2 states exactly what AI-29 inherits and does not inherit.

**If you are reviewing this artifact:** § 14 walks both closing checklists against it, item by item. § 4 and § 8 are where a missing cell or a mis-marked outcome is most expensive.

**Every reference to an existing spec below is by identifier only** — `CAP-R-0N`, `CAP-O-0N`, `V-REQ-24`, and the like resolve to [the v1 capability set](../../specs/ai-minimum-capabilities/spec.md) and [cache-boundary markers](../../specs/ai-cache-breakpoints/spec.md). Nothing in either is redefined or modified here.

---

## 2. What was decided

Both verdicts, before any argument, for the reader who came for one answer.

| Node | Question | Verdict |
| --- | --- | --- |
| AI-24.1 | Which vendor dialect? | **The OpenAI-compatible Chat Completions streaming dialect** — the `/v1/chat/completions` wire shape with `stream: true`, as documented by OpenAI itself and mirrored byte-for-byte by the gateways and self-hosted servers that share it (OpenRouter, Azure OpenAI, and local OpenAI-compatible inference servers among them) |
| AI-24.2 | Which transport? | **Raw `net/http`** — no vendor SDK. `backend/agent/go.mod` carries **zero `require` lines**, command-verifiable (§ 9) |

Rejected, and argued in §§ 4–5: the Anthropic Messages API dialect, native; and, as one candidate class, either vendor's official Go SDK.

**The choice itself was made before this artifact and is not reopened here** (per `proposal.md`'s locked constraints). This document's job is to ground it, price what it costs, and hand each blocked milestone something concrete enough to build against without reopening this decision.

---

## 3. Evidence rules

Five rules bind every section below. Stated once, so no section restates its own standard of proof.

1. **The source-label rule.** Every framing claim in this artifact — inside § 10 or restated anywhere else — is labelled **spec-mandated** (cited to WHATWG HTML Living Standard § 9.2) or **dialect-conventional** (cited to this vendor's documented or observed behavior, with its fixture-pin obligation named). An unlabelled framing claim is a defect.
2. **The grounding rule.** Every § 4 axis cell rests on a cited vendor document, a cited specification clause, or a stated in-repo mechanical fact — never a bare assertion. An empty cell is a defect, never an implied tie.
3. **The fairness rule.** Every rejected alternative's loss is stated as a loss, not folded into a summary verdict. § 5 names what the chosen vendor's choice costs, plainly, in the same document as the win.
4. **The totality rule.** § 8 is total over both closed capability lists. Expected outcomes are drawn only from `satisfied` and `absent` — `failed` and `not exercised` are run results, never predictions, per the v1 capability set §§ 10 and 12. An entry whose basis a later node must still confirm is marked **pending**, not settled.
5. **The deletion test.** If removing a sentence would hand a later milestone more choices, and that milestone is not AI-24, the sentence is cut. Applied sharpest against AI-29.0 in § 7 and § 13.2: this artifact records an indication and explicitly does not decide.

---

## 4. The seven-axis comparison

**Closing-checklist item 1 (AI-24.1).** One vendor named; every rejected alternative recorded, with reasons, across all seven charter axes (doc 0002's own list): capability fit, streaming quality, testability, dependency weight, endpoint configurability, maintenance, and the credential-handling boundary.

Three candidates. **Vendor SDK** is one deliberate class covering either vendor's official Go SDK (OpenAI's and Anthropic's), because the two failure modes it carries — an added `go.mod` dependency and a harder-to-redirect client — hold for both, and treating them as one candidate keeps the product below total rather than doubling a column that argues identically.

| Axis | Chosen: OpenAI-compatible dialect | Anthropic Messages API, native | Vendor SDK (either vendor's official Go SDK) |
| --- | --- | --- | --- |
| **Capability fit** | Clears all five required capabilities (`CAP-R-01…05`): `delta` chunks reconstruct text exactly; `tool_calls[]` carries real `id`, name and argument fragments keyed by `index`; `finish_reason` + `usage` close a normal stream; HTTP-level cancellation is unconditional; failures are typed at both the pre-stream and mid-stream boundary | Also clears all five, structurally: `content_block_delta` events, `tool_use` blocks, `stop_reason` + `usage` in `message_delta`/`message_stop`. This axis does not distinguish the two dialects | Identical to whichever dialect it wraps — an SDK is a transport wrapper, not a third protocol. Adds no capability the raw dialect lacks and removes none |
| **Streaming quality** | Tool-call arguments stream fragmented, keyed by array `index` (§ 7); reasoning, where reported at all, is an opaque `reasoning_tokens` count in `usage`, never a signed block | Deltas carry an explicit `index` and a typed discriminator (`text_delta`, `input_json_delta`); reasoning streams as signed `thinking` blocks with a `signature` field — a richer signal than the chosen dialect's opaque count. **This is a genuine loss, priced in § 5** | Identical wire-level signal to the wrapped dialect; an SDK adds client-side iterator ergonomics, never wire capability |
| **Testability** | Plain SSE-shaped text over HTTP, decodable by any test double; a local `httptest.Server` emits hand-authored frames with no vendor handshake | Equally testable at the wire level — same SSE-shaped framing, same local-server viability. **This axis is a tie between the two dialects**, and does not carry the decision | Weaker for both vendors: an SDK's client construction and streaming iterator typically wrap the HTTP layer, so redirecting it to a local test server depends on whatever base-URL override that SDK version happens to expose, and is not guaranteed by the dialect itself |
| **Dependency weight** | Chosen dialect + `net/http`: zero `go.mod` requires (§ 9, command-verified) | Anthropic dialect + `net/http`: also zero `go.mod` requires — hand-rolling either dialect over the standard library costs the same nothing. **This axis, too, is a tie between the two dialects** | Adds at least one `go.mod` requirement (the SDK module itself, plus its own transitive closure), which is exactly what triggers `openspec/AGENTS.md` rule 5's ADR gate |
| **Endpoint configurability** | The **decisive axis for this choice.** Many independent hosts — OpenAI itself, OpenRouter, Azure OpenAI, and self-hosted OpenAI-compatible inference servers — implement the identical wire shape, so a base-URL change alone can move between them without touching the adapter | Narrower ecosystem: fewer third-party hosts implement a byte-compatible Anthropic Messages dialect, so the same base-URL-only portability is not available to the same degree | Worse than either raw dialect: pointing a vendor SDK at a differently-hosted, non-vendor endpoint risks violating response-parsing assumptions the SDK bakes in for its own vendor's exact behavior |
| **Maintenance** | This repository owns 100% of the wire-level code; upgrades follow this repo's own release cadence, with no upstream SDK version to track and no transitive-dependency advisories to triage | Identical maintenance profile to the chosen dialect — both hand-rolled, both zero-dependency. **Another tie**; this axis does not carry the AI-24.1 verdict | Worse for both: every SDK major-version bump is an upstream surface this repo would have to track, on a release cadence outside this repo's control |
| **Credential-handling boundary** | A bearer token is attached as a single `Authorization: Bearer <token>` header at request-construction time, entirely inside code this repo owns and can audit; nothing about `net/http` requires reading an environment variable | Same shape — an API-key header attached identically, with the same audit property | Worse: an SDK constructor commonly defaults to reading a well-known environment variable when no explicit credential is passed — precisely the ambient-authority risk AI-25.2's guard exists to catch, and one a `net/http`-based adapter has no default to override in the first place |

**Reading the ties honestly.** Three axes — capability fit, testability, dependency weight, maintenance — do not distinguish the chosen dialect from Anthropic's; they are true ties, stated as such rather than manufactured as wins, per the fairness rule. The verdict rests on **endpoint configurability** (the widest reach across vendors sharing one wire shape) and, more decisively still, on **streaming quality and dependency weight together against the vendor-SDK class**, which loses on nearly every axis regardless of which dialect it wraps. Removing any single axis from this table costs at least one candidate a stated basis for its verdict — the vendor-SDK column loses its dependency-weight and testability grounds if either row is removed, and the chosen-versus-Anthropic distinction loses its only decisive ground if endpoint configurability is removed.

---

## 5. What the choice costs

**Closing-checklist item 1's fairness half.** Two contract items lose their first real exercising consumer under this choice. Both remain contract-mandatory regardless of who emits them — the fairness rule forbids treating an orphaned consumer as a deprecated contract.

- **AI-07's reasoning round-trip token.** The chosen dialect signs no reasoning blocks (§ 4, § 7); it reports only an opaque `reasoning_tokens` count inside `usage`. No v1 adapter therefore exercises the signature-preservation path this token exists for. The neutral shape — byte-exact capture, byte-exact replay — stays contract-mandatory, because a future adapter for a dialect that does sign reasoning (Anthropic's `thinking` blocks, for one) needs it unchanged.
- **AI-11's breakpoint cap (`V-REQ-24`).** The chosen vendor caches automatically (§ 6); there is no vendor-side cap to enforce, so the adapter-side cap-enforcement branch has no subject. What is exercised instead is AI-11.3's advisory contract (`R-ACB-008`): cache-boundary markers are dropped whole, and the request remains fully translatable and semantically unchanged. The cap-enforcement code and the neutral marker shape both stay contract-mandatory, for a future adapter whose vendor requires explicit annotation with a real cap.

Neither loss is a defect in this decision; both are priced here, in the same document as the win, rather than discovered later as a surprise.

---

## 6. Four divergences, answered

**Closing-checklist item 2 (AI-24.1).** Each of doc 0002's four documented cross-provider divergences, answered explicitly for the chosen vendor, with the downstream node it drives and the consequence for that node.

| Divergence | Answer for the chosen vendor | Node driven | Consequence |
| --- | --- | --- | --- |
| Cache-breakpoint expression | **Automatic.** No client-supplied breakpoint marker exists in this dialect's wire shape | AI-26.2 | The cap-enforcement branch has no subject; the "dropped whole" branch is taken, exercising AI-11.3's advisory contract in full. `CAP-O-03`'s expected outcome is **absent** (§ 8) |
| Tool-result placement | A tool result is a **distinct `role: "tool"` message**, never a block inside a user-role message and never a nested object | AI-26.5 | AI-26.5 item 1's translation target is fixed: emit one `role: "tool"` message per result, correlated to its call by identity |
| Output-token-limit mandatoriness | **Optional.** The dialect accepts a request with no output-token limit set | AI-26.7 | Item 2's mandatory-default branch is a deliberate no-op for this vendor, recorded as such rather than left unmentioned |
| Tool-call identifier assignment | The vendor **does assign** tool-call identifiers, on the call's **opening delta** | AI-26.5 | Item 2's synthetic-minting branch has no subject for this adapter; conformance exercises only the vendor-assigns branch, marked not-applicable rather than silently skipped |

---

## 7. Two further questions

**Closing-checklist item 3 (AI-24.1).** doc 0002 adds two questions beyond the four divergences, each weighted for the node it drives.

**Does the vendor stream tool-call arguments in fragments or whole?** **In fragments**, keyed by the call's array `index` — each argument delta arrives as a partial JSON string fragment, concatenated in index order to reconstruct the complete argument bytes. This weights **AI-30** toward fragmented accumulation as the primary case; the zero-fragment, whole-call shape (already legal per the Layer 1 contract, AI-18.2) is the edge case for this vendor, not the default.

**Does it sign reasoning blocks?** **No.** The dialect reports no signed reasoning content of any kind — only an opaque `reasoning_tokens` count inside the completion's `usage` object, which is a **count**, not a **block**: it carries no text, no signature, and nothing a `%#v`-style round trip could replay. This is distinguished explicitly from a signed block (which Anthropic's `thinking` blocks are, per § 4) because the two are not degrees of the same thing — one is content with a cryptographic seal, the other is a number.

**This strongly indicates that the first adapter should record a capability absence for `CAP-O-01`** (§ 8) — but per the deletion test (§ 3) and `R-APD-010`, **the emission-versus-absence decision remains AI-29.0's**, not this artifact's. AI-29.0 judges against the exact backend chosen for AI-38/AI-39, and the concrete case that could overturn this indication is stated plainly: **some servers sharing this dialect emit a non-standard `reasoning_content`-style extension field that is not part of the shared dialect itself.** This artifact records the indication as an input to AI-29.0, names AI-29.0 as owner, and neither strikes nor deletes that node (§ 13.2 restates this as a standing rule).

---

## 8. The expected capability report

**Closing-checklist item 4 (AI-24.1).** Total over both closed lists from [the v1 capability set](../../specs/ai-minimum-capabilities/spec.md) — `CAP-R-01…05` and `CAP-O-01…03`, eight entries, no more and no fewer.

**The floor clause, stated first.** Per that document's § 12: *"The five required capabilities are a floor on vendor selection. A candidate vendor that cannot support all five cannot be the first adapter."* This section is a floor confirmation, not merely a table: every required entry below is confirmed cleared before any optional entry is discussed.

**Where standing comes from, and why only two outcomes are legitimate here.** Every row's **Standing** column is copied from [the v1 capability set](../../specs/ai-minimum-capabilities/spec.md) §§ 5–7 — never derived by this decision and never taken from any run. Every **Expected outcome** is drawn only from `satisfied` or `absent`, because `failed` and `not exercised` are that document's run results, never predictions (§§ 10, 12): the distinction is between a **conclusion** (`satisfied`, `absent`) and the **absence of one** (`not exercised`), stated there structurally, not as a stylistic choice made here.

| Id | Standing | Expected outcome | Basis | Confirmation |
| --- | --- | --- | --- | --- |
| `CAP-R-01` | required | `satisfied` | Text arrives as `delta.content` chunks between an implicit block start and the closing `finish_reason`; concatenation reconstructs the text exactly | confirmed |
| `CAP-R-02` | required | `satisfied` | `tool_calls[]` carries a real `id` (assigned by the vendor, § 6), `function.name`, and `function.arguments` fragments keyed by `index` | confirmed |
| `CAP-R-03` | required | `satisfied` | A normally-finished stream closes with `finish_reason` and `usage` — **but `usage` populates in-stream only when the request sets `stream_options.include_usage: true`.** Left unset, `usage` returns empty, which would misread as `CAP-R-03 = absent` — illegal for a required capability, and a disguised adapter defect rather than a vendor limitation. Obligation named again in § 13.1 and assigned to AI-26.7, AI-28.3 and AI-31.2 | confirmed |
| `CAP-R-04` | required | `satisfied` | By construction, as AI-03's decision § 5 records for this capability generally: the caller-owned context cancels the underlying `net/http` request, and cancellation closes the stream within bounded time | confirmed |
| `CAP-R-05` | required | `satisfied` | Pre-stream: HTTP status plus a structured vendor error body distinguish failure categories before any decode (AI-28.6, AI-32). Mid-stream: truncation and malformed-frame detection (AI-27, AI-28.2, AI-28.5) map into AI-19's taxonomy with the partial-output flag intact | confirmed |
| `CAP-O-01` | optional | `absent` | The dialect signs no reasoning blocks; only an opaque `reasoning_tokens` count is reported (§ 7) | **pending AI-29.0's confirmation** — made against the exact backend chosen for AI-38/AI-39, because some servers sharing this dialect emit a non-standard `reasoning_content`-style extension field outside the shared dialect |
| `CAP-O-02` | optional | `absent` | The dialect exposes no dedicated pre-flight token-counting endpoint; `usage` reports post-hoc counts only, which is `CAP-R-03`'s territory, not this capability's | confirmed |
| `CAP-O-03` | optional | `absent` | The vendor caches automatically (§ 6); there is no explicit cache annotation to render, so markers are dropped whole under AI-11.3's advisory contract (`R-ACB-008`) | confirmed |

**Reading the three `absent` rows.** Per AI-03's decision § 6, *"a provider that lacks every one of these is fully conformant."* Three recorded absences are not an under-powered vendor; they are the expected, conformant shape of this specific dialect, stated as predictions this artifact commits to before any adapter exists, so AI-38.2's generated record can be compared entry by entry. **A difference in either direction is a finding**: an unexpected `absent` on an entry expected `satisfied` is a regression, and an unexpected `satisfied` on an entry expected `absent` means the adapter grew a capability nobody reviewed — neither reading is preferred over the other.

---

## 9. Transport: the no-op ADR gate

**Closing-checklist item 1 (AI-24.2).** `net/http` versus a vendor SDK, decided with evidence stated on the charter axes (§ 4), not as a preference: the vendor-SDK class loses on dependency weight, testability and endpoint configurability, and ties on nothing that would recover it.

**The dependency consequence, stated command-verifiably.** `backend/agent/go.mod` declares a module path and a Go version and nothing else. A reviewer can check this directly:

```
$ grep -c '^require' backend/agent/go.mod
0
```

Zero `require` lines, before and after this decision. `net/http` is standard library; choosing it adds no dependency of any kind.

**The ADR gate, recorded as evaluated, not as silent.** `openspec/AGENTS.md` rule 5 states: *"New top-level dependency ⇒ ADR first. Document rationale, alternatives, rollback."* This decision evaluates that gate directly: because the chosen transport adds no `go.mod` dependency, the gate **fires no obligation** — it resolves to a **no-op**, discharged by the same zero-requires fact stated above. A gate that resolved to nothing and a gate nobody applied would be indistinguishable from an artifact that mentions neither; this section exists so they are not.

**The gate transfers, it does not close.** The zero-dependency state this decision confirms holds until **AI-37**, which doc 0002's milestone rules already name as the next boundary: AI-37 adds the OpenTelemetry API, pre-authorized in full by ADR 0005 § D3 and by nothing else. Any dependency beyond that pre-authorization would re-trigger rule 5's gate at AI-37, exactly as it would have here had this transport added one.

**The AI-00.3 forward guard gains no second allowlist entry.** `net/http` is standard library and was already reachable through the guard's stdlib allowance; its one transitive non-stdlib-adjacent import, `os`, is already permitted as a legitimate dependency of the standard HTTP client, not as a new entry this decision adds. The guard's own source comment names AI-24 as the milestone expected to add a second allowlist entry — under this decision, that entry is never added, and the comment is now stale. **Routed, not fixed here:** correcting that comment is Go source and belongs to **AI-25**, the first milestone that reopens that package; this artifact ships no Go and records the routing rather than making the edit.

---

## 10. The streaming framing contract

**Closing-checklist item 2 (AI-24.2).** Named precisely enough that AI-27 encodes fixtures from this section alone, without consulting the vendor. Every claim below carries its label, per the source-label rule (§ 3).

### 10.1 Spec-mandated

Every claim in this subsection is stated by the **WHATWG HTML Living Standard § 9.2** (server-sent events), independently of any one vendor's habits — including the two elements this vendor's dialect never emits, because the decoder is specified once, not per vendor.

| Element | Rule |
| --- | --- |
| Response content type | `text/event-stream` |
| Field grammar | A field's name and value split at the **first colon** on the line; exactly **one leading space** is stripped from the value if present; a line with no colon at all is treated as that line's whole text as the field name, with an empty value |
| Comment lines | A line beginning with `:` is a comment — ignored entirely, disturbing no accumulation state; this is the framing's keep-alive idiom |
| Multi-line `data:` joining | Each `data:` field's value is appended to a per-event data buffer, followed by one line-feed (`\n`); at dispatch, exactly **one trailing line-feed** is removed from the buffer — never more, never a different separator |
| Identifier field disposition | An `id:` field updates the stream's last-event-id state, **unless its value contains a NUL byte**, in which case the field is ignored. **This vendor's dialect never emits this field**; the decoder still implements the rule, because it is specified independently of this vendor |
| Reconnection-time field disposition | A `retry:` field composed entirely of ASCII digits sets the reconnection interval. **This vendor's dialect never emits this field either**; the same independence argument applies |
| Byte-order mark | A leading BOM, if present, is stripped exactly once, at the very start of the stream; a BOM appearing anywhere else is ordinary content |
| Line terminators | All three are accepted: CRLF, a bare LF, and a bare CR |
| Per-event dispatch (the terminal-termination convention) | An event dispatches on a **blank line**. This is the per-event boundary the specification defines; it is distinct from — and must not be confused with — § 10.2's whole-stream sentinel below |

### 10.2 Dialect-only conventions

**No specification states anything in this subsection.** Both entries rest solely on this vendor's documented and observed behavior, and both carry an explicit AI-27 fixture-pin obligation: encoded from this section, never inferred from § 9.2.

| Element | Rule | Fixture-pin obligation |
| --- | --- | --- |
| `data: [DONE]` terminal sentinel | A final frame whose `data:` value is the literal string `[DONE]` marks the end of the **whole stream** — a dialect convention, unrelated to § 10.1's per-event blank-line dispatch | AI-27 must recognize this sentinel by explicit fixture, and must never parse it as event payload |
| Data-only framing | This vendor never sends an `event:` field; every dispatched event therefore carries the specification's default type, **`message`**, because no dialect-specific event-type vocabulary exists on this wire | AI-27 must pin a fixture asserting the default-type behavior explicitly, never assume it from the spec's default alone without a test |

Attributing either row to § 9.2 would be a defect: a decoder built on that attribution conforms to something no specification states, and breaks the first time it meets a genuinely spec-conformant server that sends neither.

---

## 11. The credential boundary

**Closing-checklist item 3 (AI-24.2).** Stated in three parts, so AI-25.2's guard has an enforceable subject.

**Receives.** An opaque bearer-token value, supplied at construction through injected configuration — explicitly **not** a variable name, and explicitly **not** a path. The adapter's own source never resolves either into a value; it only ever holds the value itself.

**Never reads — enumerated as observable behaviors, not as an intention.** The adapter package's own source:

- Never reads an environment variable.
- Never opens a file.
- Never spawns a process.

These three are the mechanically enforceable subject of AI-25.2's guard (§ 12), corrected from an import-path scan to a call-site scan over the adapter's own source files, because the transport this decision names would otherwise make an import-path scan structurally unable to catch a narrow violation of any of the three (§ 12's AI-25 row states the reason in full).

**Origin.** The credential value's origin is **Layer 3's composition root** — entirely outside Layer 1. How that layer obtains, stores or rotates the value is out of this decision's, and out of Layer 1's, scope in its entirety; this artifact states only that the value arrives already resolved.

---

## 12. What each blocked milestone inherits

doc 0002: *"Blocks: AI-25 … AI-32."* Each row is written in that milestone's own terms, sufficient to plan it without reopening this decision.

| Milestone | What it inherits |
| --- | --- |
| **AI-25** | § 11's credential boundary as its guard's enforceable subject, and the **corrected guard mechanism**: a call-site scan over the adapter package's own source files, not an import-path scan — because `net/http` transitively imports `os`, which is already on this module's allowlist as a legitimate stdlib dependency, so an import-path scan would either false-positive on ordinary `net/http` use or miss a narrow `os.Getenv` call inside the adapter entirely. **Obligation, with its failure mode stated**: build the guard as a call-site scan, or ship a guard that cannot bite. Also inherits: plain `net/http` construction, no vendor SDK; and the routing of the AI-00.3 comment correction (§ 9), which is this milestone's Go-source fix to make, not this artifact's |
| **AI-26** | § 6's four divergence answers as its per-node branches, directly: AI-26.2 takes the "dropped whole" branch (no cap-enforcement subject); AI-26.5 translates results into a distinct `role: "tool"` message and takes only the vendor-assigns branch for identifiers (no synthetic-minting subject); AI-26.7's mandatory-default branch is a deliberate no-op |
| **AI-27** | § 10 as its literal fixture specification — both § 10.1 (encode as specified, including the two elements this vendor never emits) and § 10.2 (pin `[DONE]` and the data-only default-type behavior by explicit fixture; never infer either from § 9.2) |
| **AI-28** | § 8's `CAP-R-03` basis as an **obligation, with its failure mode stated**: construct every request with `stream_options.include_usage: true`, and assert that `usage` is actually present rather than silently accepting an empty one — an unset option produces an illegitimate `absent` reading on a required capability, indistinguishable from a vendor limitation unless checked |
| **AI-29** | § 7's indication — no signed reasoning blocks, an opaque token count only — as an **input**, not a ruling. AI-29.0 remains the owner of the emit-versus-absence decision (§ 13.2); this artifact records the concrete case that could overturn the indication and pre-empts nothing |
| **AI-30** | § 7's fragmented tool-call-argument answer, keyed by `index`, as the case to weight as primary; the zero-fragment whole-call shape (already legal under AI-18.2) as the edge case for this vendor, not the default |
| **AI-31** | § 8's `CAP-R-03` basis, the full usage-field-mapping half: **obligation, with its failure mode stated** — map every populated usage field faithfully, and preserve absent-versus-zero fidelity (AI-13.3) for whichever fields this vendor's `usage` object does not populate |
| **AI-32** | § 9's transport decision as what it maps from: raw HTTP status codes and this vendor's own structured error-body shape, with no SDK-level error type standing between the wire and AI-19's taxonomy |

---

## 13. Standing rules and the carryover assignment

Three standing rules. The first two are traps and reservations rather than answers to a checklist item; the third is `R-APD-017`'s home in this artifact.

### 13.1 The usage opt-in trap

Restated as a standing rule because it recurs across three implementation nodes, not because § 8 left it unstated. This vendor emits in-stream `usage` **only when the request sets `stream_options.include_usage: true`.** Left unset, `usage` returns empty on every completion event. That empty result is legal-shaped — nothing about it looks like an error — and would misread as `CAP-R-03 = absent`, which is **illegal for a required capability** (§ 3's totality rule; the v1 capability set § 10). The obligation this creates is assigned to three named nodes: **AI-26.7** must construct every request with the option set; **AI-28.3** must assert `usage`'s presence rather than accept an empty one silently; **AI-31.2** inherits the full mapping obligation once presence is confirmed. An unset option is an adapter defect wearing a vendor limitation's clothes, and this rule is what keeps the two distinguishable.

### 13.2 The AI-29.0 reservation

§ 7's indication is repeated here in its standing-rule form, precisely because a reader arriving at § 13 rather than § 7 must not miss it: this artifact does **not** decide whether the first adapter emits reasoning events. It records that the chosen dialect signs no reasoning blocks and reports only an opaque token count, states that this strongly indicates absence, and names **AI-29.0** as the sole owner of the emit-versus-absence decision — made against the exact backend chosen for AI-38/AI-39, with the concrete overturning case stated (a non-standard `reasoning_content`-style extension some servers sharing this dialect emit). No amendment this decision proposes strikes, deletes, or pre-empts AI-29.0 (§ 4 of the amendment plan, cross-referenced at doc 0002 A5).

### 13.3 The Wave-2 carryovers → AI-41

**`R-APD-017`'s home in this artifact.** Two carryovers, recorded by identifier at [`openspec/specs/ai-stream-testkit/spec.md`](../../specs/ai-stream-testkit/spec.md) as unassigned through Wave 2 and Wave 3:

1. The stream test kit's emission-boundary checker has a fourth rule whose failure path is not exercised by any test.
2. The provider-failure payload has no redacting alternate-verb (`%#v`-shaped) formatting method, unlike its sibling payloads.

**Assigned to an appended milestone, `AI-41` — Discharge the Wave-2 carryovers**, taking the next free milestone ordinal, with two leaves (`AI-41.1` for the first carryover, `AI-41.2` for the second) and a `Blocks: AI-36` edge: AI-36's adversarial redaction sweep must not run against a payload still missing its redacting method, which is exactly the scope overlap that edge exists to prevent. Retrofitting either carryover into an archived Wave-2 or Wave-3 milestone is not proposed; the append-only identifier rule forbids it.

**Scheduled Wave 5, with both reasons stated, not merely asserted.** First: neither carryover blocks any Wave-4 node — AI-24 through AI-32 touch neither the emission-boundary checker's failure path nor `%#v` on failure payloads. Second: Wave 4 already forecasts **20,000–25,000 changed lines** against this project's cached **5,000-line** review budget; absorbing two more, unrelated leaves into Wave 4 buys nothing and costs reviewer focus that wave does not have to spend.

**Cross-referenced, not duplicated.** The append itself — the new `### AI-41` heading, its charter, its two leaves, and the four navigational surfaces it forces to agree on a milestone total of 42 — lands in doc 0002 as amendment **A6**. This section is the artifact's half of the one assignment `R-APD-017` requires; per `S-APD-056`, the assignment obligation belongs here, in `decision.md`, and doc 0002's A6 is its mirror in the graph.

---

## 14. Closing-checklist verification

Both closing checklists, seven items total, each walked against this artifact.

| # | Node | Checklist item (doc 0002's own words) | Where answered | Status |
| --- | --- | --- | --- | --- |
| 1 | AI-24.1 | One vendor named; rejected alternatives recorded with reasons across all seven charter axes | §§ 4–5 | **answered** |
| 2 | AI-24.1 | Four documented cross-provider divergences answered explicitly, each driving a later node | § 6 | **answered** |
| 3 | AI-24.1 | Two further questions — argument fragmentation, reasoning signing | § 7 | **answered** |
| 4 | AI-24.1 | Which optional capabilities this vendor supports, recorded as the expected capability report | § 8 | **answered** |
| 5 | AI-24.2 | `net/http` versus a vendor SDK, decided with evidence; the ADR gate's outcome | § 9 | **answered** |
| 6 | AI-24.2 | The streaming framing named precisely — event and field dialect, terminal sentinel convention | § 10 | **answered** |
| 7 | AI-24.2 | The credential-handling boundary stated: receives, never reads, origin | § 11 | **answered** |

**Milestone acceptance, restated and checked.** doc 0002's AI-24 acceptance clause: *"The decision names one first adapter and explains the rejected alternatives. If the transport adds a `go.mod` dependency, its artifact must include or be promoted to the ADR required by `openspec/AGENTS.md` rule 5 before AI-25 adds that dependency."* One adapter is named (§ 2); rejected alternatives are explained across all seven axes (§ 4); the transport adds no dependency, so the ADR clause's condition did not fire, and § 9 records that discharge rather than its absence.

**Node status.** AI-24.1 and AI-24.2 close on merge of this artifact. Per doc 0002's node grammar, a `[decision]` leaf produces no production code and closes when "the decision artifact answers every listed question and is merged." No `make test` gate applies — this change touches nothing under `backend/`.

**Unblocked by this decision:** AI-25, AI-26, AI-27, AI-28, AI-29, AI-30, AI-31, AI-32 — each per its § 12 row — and, through them in turn, AI-33 through AI-40. AI-41, appended by this decision's own living-graph obligation (§ 13.3), is scheduled in Wave 5 and blocks AI-36 there.
