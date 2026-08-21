# Spec — Layer 2's observability boundary (`agent-observability-boundary`)

> **Change**: `cachicamas-agent-observability` · **AG-22** (Layer 2, Wave 6) of [doc 0003](../../../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-22--add-the-observability-boundary), charter `0003:2043-2094`
> **Nodes**: AG-22.1 `[decision]` the Layer 2 attribute vocabulary (`0003:2064-2069`) · AG-22.2 `[leaf]` spans within the boundary (`0003:2071-2093`)
> **Status**: **new capability**. This file is the normative text; per the AG-14 / AG-19 / AG-20 / AG-21 precedent it is promoted to `openspec/specs/agent-observability-boundary/spec.md` at archive. Two cross-cut deltas ship beside it under this change's `specs/` tree.
> **Governing ADR**: [ADR 0005 § D3](../../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) — the import table (`0005:237-244`), the Layer **1** attribute allowlist (`0005:246-250`) and the absolute content denylist (`0005:252-255`). § D3's allowlist is scoped *"for Layer 1 spans"* in its own words; this capability is the § D3 **extension** that decides Layer 2's, per `0003:2067`.
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `rules.specs`. Every scenario is independently verifiable by `cd backend/agent && go test -race -count=1 ./...`.
> **IDs**: requirements `R-AGO-0NN`, scenarios `S-AGO-0NN` (bites carry the same `S-AGO-` prefix and are marked **(bite)**), non-functional `NFR-AGO-0NN`. **Append-only.**
> **Allocated ranges**: `R-AGO-001` … `R-AGO-099`; `S-AGO-001` … `S-AGO-199`; `NFR-AGO-001` … `NFR-AGO-099`. This capability's range is reserved so a later milestone extending Layer 2's observability appends without collision. The header states **ranges and never totals** — a total is defended by no test and goes silently false on the next append (`S-LSK-020`, and `agent-concurrency-hardening/spec.md:8`).
> **Prefix verification**: `AGO` was re-verified collision-free in this phase across `openspec/specs/`, `openspec/changes/` **and** `openspec/changes/archive/`: the only pre-existing hit for `R-AGO-`/`S-AGO-` is this change's own `proposal.md:53`. `AOB` belongs to Layer 1 (`ai-observability-boundary`) and MUST NOT be reused.
> **Evidence gate**: `cd backend/agent && go test -race -count=1 ./...` with the wall-clock duration recorded (a `(cached)` run is not evidence), plus `make lint` (after `golangci-lint cache clean`), `make build` and `make vuln-check`. `make all` MUST NOT be run — its fmt step rewrites committed files. No CI exists.
> **Sources**: charter `0003:2043-2094`; this change's `proposal.md` and `design.md` decisions **D-A … D-G** and the design's *Addendum — closure measured by command* (`design.md:141-189`), all binding.

> **Note on length.** The `sdd-spec` skill sets a 650-word budget. This spec exceeds it deliberately, on the recorded precedent of `agent-v1-scope/spec.md:348-350` and `agent-concurrency-hardening/spec.md:13`: a decided attribute vocabulary with a per-attribute value-equality mapping, an absolute denylist with a per-field enumeration, four span lifetimes and an absence claim with its own defeat test do not compress without dropping content `openspec/config.yaml` requires be independently verifiable.

## Purpose

Layer 2's runs, turns, tool executions and compactions are observable today only through the in-process event stream. This capability defines what Layer 2 may export about itself, and — more load-bearing — what it may never export.

It has four halves that fail in four different ways: a **dependency** boundary that fails loudly at build time; a **vocabulary** contract that fails as a breaking change for every telemetry consumer if a key is renamed or a value stops matching the event it mirrors; a **lifetime** contract that fails as a leak of exactly the shape AG-21 removed; and an **absence** claim that fails **silently** if its proof is vacuous. Each half is specified with the falsifier that makes it non-vacuous, not merely with the property it asserts.

**Register.** These requirements state behaviour. Attribute **keys** are spelled exactly, because they are external semantic-convention names rather than language identifiers (the `R-AOB-004` precedent). **Event accessor names also appear**, deliberately: the charter's acceptance clause is *"values equal to the corresponding events'"* (`0003:2080`), and a value-equality claim that does not name the accessor it must equal is prose, not a checkable mapping.

## Coverage — the charter's own acceptance, traced

| Charter clause | Requirements | Scenarios |
|---|---|---|
| AG-22.1 item 1 — span names and attributes decided, recorded as a § D3 extension, denylist restated not weakened (`0003:2067`) | `R-AGO-002`, `R-AGO-003`, `R-AGO-004` | `S-AGO-010`…`S-AGO-018`, `S-AGO-020`…`S-AGO-023` |
| AG-22.1 item 2 — what is deliberately **not** recorded, each named with its reason (`0003:2068`) | `R-AGO-005` | `S-AGO-030`, `S-AGO-031` |
| AG-22.2 scenario 1 — spans nest correctly with delegation trees preserved, attributes from the decided vocabulary with values equal to the events' (`0003:2076-2080`) | `R-AGO-002`, `R-AGO-006` | `S-AGO-014`…`S-AGO-018`, `S-AGO-040`…`S-AGO-044` |
| AG-22.2 scenario 2 — the denylist proven by absence over a full-featured run (`0003:2082-2085`) | `R-AGO-004`, `R-AGO-008` | `S-AGO-020`…`S-AGO-024`, `S-AGO-070`…`S-AGO-075` |
| AG-22.2 scenario 3 — no tracer, no difference, nothing panics (`0003:2087-2090`) | `R-AGO-001`, `R-AGO-009` | `S-AGO-001`…`S-AGO-004`, `S-AGO-080`…`S-AGO-083` |
| Charter Goal — *"through the OTel **API** only"* (`0003:2049`) | `R-AGO-010` | `S-AGO-090`…`S-AGO-094` |
| Span lifetime — the detached arm, decided rather than implicit (proposal open question 1) | `R-AGO-007` | `S-AGO-050`…`S-AGO-058` |

The columns name IDs rather than counts, for the reason the **Allocated ranges** header line records.

---

## The denylist, quoted from ADR 0005 § D3 (`0005:252-255`), absolute

> **Attribute denylist, absolute:** any prompt, completion, reasoning, tool-argument or tool-result text; any HTTP header; any credential; any raw provider response body.

It is restated here **verbatim and unweakened**. It applies to Layer 2 spans without exception and without carve-out. No requirement, scenario or note in this capability may narrow it.

---

## Requirements

### R-AGO-001 — A tracer provider reaches Layer 2 only by injection, and its absence is answered structurally

Layer 2 MUST obtain a tracer provider **only by injection**, through one exported optional field on the harness value. Layer 2 MUST NOT read, write or otherwise consult any process-global telemetry registry, getter or setter — the tracing instance of Layer 1's standing rule that configuration arrives only by injection (`R-AOB-001`, `R-APC-008`).

When no provider is injected, the boundary MUST substitute the tracing API's **own no-op provider**. The substitution MUST make every recording site unconditional, so a zero-value harness stays inert and nothing panics. A harness that never had a provider MUST behave exactly as one that has a recording one, per `R-AGO-009`.

Below the harness, a tracer MUST be acquired **ambiently from the span already on the context**, not from a second injected field and not from a global. A call entered directly with a context carrying no recording span therefore records nothing and MUST still complete normally.

The injected field is an OpenTelemetry **API** provider. It is **not a telemetry sink**: it registers nothing, exports nothing, and reaches no SDK. That distinction is what keeps `agent-v1-scope`'s `S-AGS-066` true after this milestone, and it is amended there in the same pull request rather than left to be re-derived.

#### Scenarios

- **S-AGO-001** — Given the shipped Layer 2 package, when its sources are scanned for any read or write of process-global telemetry state, then none exists, and every tracer used by a traced run is traceable to an injected value or to a span already on the context.
- **S-AGO-002** — Given a harness value with no tracer provider set, when a full run is driven to its terminal, then it completes normally, nothing panics, and no span reaches any provider registered in any process-global telemetry state.
- **S-AGO-003** — Given a turn or a tool schedule entered directly with a context carrying no recording span, when it is driven to completion, then it records no span and completes with the same outcome it produces without this change.
- **S-AGO-004** — **(bite)** Given a deliberately mutated implementation that acquires its tracer from the ecosystem's process-global getter instead of the injected field, when `S-AGO-001` runs, then it FAILS and names the global-getter access — the injection claim is asserted, not assumed. The red output is recorded, the mutation dropped, the suite re-run green.

### R-AGO-002 — Four span families, a closed attribute vocabulary, and every value equal to the exact event accessor it mirrors

Layer 2 MUST record exactly four span families, and the vocabulary below MUST be recorded as an **extension of ADR 0005 § D3's table** — appended to ADR 0005 immediately after § D3's Layer 1 allowlist, not as a sibling ADR and not as a replacement (`0003:2067`). Span names follow the OpenTelemetry GenAI semantic conventions where they exist; where no convention equivalent exists, names and keys are minimal inventions under the `cachicamas.` namespace, which is the charter's own *"inventing nothing gratuitous where they do not"* carve-out exercised rather than violated.

| Span family | Span name |
|---|---|
| run | `invoke_agent` |
| turn | `turn` |
| tool execution | `execute_tool {name}` |
| compaction | `compact` |

The attribute key set recorded on any Layer 2 span MUST be a **subset** of the keys below. No key outside this set MAY appear, under any name, on any path. Each key MUST be spelled **exactly** as written here and MUST be a fixed literal spelled in exactly one place, never assembled at a recording site (`NFR-AGO-002`). Each recorded value MUST **equal** the value of the named accessor on the corresponding event of that same bracket — this is the charter's *"values equal to the corresponding events'"* clause (`0003:2080`) made checkable:

| Span | Key | Go type | MUST equal — exact event accessor |
|---|---|---|---|
| run | `gen_ai.operation.name` | string | the constant `"invoke_agent"` — convention-required on an agent span; mirrors no event |
| run | `cachicamas.run.id` | string | `Event.Run()` on the run bracket's events (`event.go:481`) |
| run | `cachicamas.run.parent_id` *(iff delegated)* | string | `Event.Parent()` (`event.go:492`), whose second result reports presence |
| run | `cachicamas.run.outcome` | string | `RunEnd.Outcome().String()` (`run_events.go:202`, `:130`) |
| run | `error.type` *(iff outcome is the failed member)* | string | `RunEnd.Failure()` → `Failure.Category().String()` (`run_events.go:206`, `failure.go:44`) |
| turn | `cachicamas.run.id` | string | `Event.Run()` (`event.go:481`) |
| turn | `cachicamas.turn.id` | string | `Event.Turn()` (`event.go:486`), whose second result reports presence |
| turn | `cachicamas.turn.outcome` | string | `TurnEnd.Outcome().String()` (`turn_events.go:202`, `:124`) |
| turn | `error.type` *(iff outcome is the aborted member)* | string | `TurnEnd.Failure()` → `Failure.Category().String()` (`turn_events.go:206`, `failure.go:44`) |
| tool | `gen_ai.tool.name` | string | `ToolStart.Name()` (`tool_event.go:128`) |
| tool | `gen_ai.tool.call.id` | string | `ToolStart.CallID()` (`tool_event.go:121`) |
| tool | `cachicamas.tool.ordinal` | int64 | `ToolStart.Ordinal()` (`tool_event.go:125`, a `uint32`), widened losslessly to the API's integer attribute type |
| tool | `cachicamas.tool.outcome` | string | the tool-end event's `Outcome().String()` (`tool_event.go:250`; the success, result-failure and execution-failure arms at `:331`, `:406`, `:491`) |
| tool | `cachicamas.tool.detached` *(iff true)* | bool | the scheduler's own detached arm — see `R-AGO-007` |
| tool | `error.type` *(iff execution failure)* | string | `ToolEndExecutionFailure.Failure()` → `Failure.Category().String()` (`tool_event.go:488`, `failure.go:44`) |
| compaction | `cachicamas.run.id` | string | `Event.Run()` (`event.go:481`) |
| compaction | `cachicamas.turn.id` | string | `Event.Turn()` (`event.go:486`) — the compaction bracket's own minted turn |
| compaction | `cachicamas.compaction.id` | string | `CompactionStarted.CompactionID()` (`compaction_events.go:121`) |
| compaction | `cachicamas.compaction.summary_id` *(iff finished)* | string | `CompactionFinished.SummaryID()` (`compaction_events.go:203`) |
| compaction | `cachicamas.turn.outcome` | string | the closing `TurnEnd.Outcome().String()` (`turn_events.go:202`) |
| compaction | `error.type` *(iff the fail arm)* | string | `CompactionFailed.Failure()` → `Failure.Category().String()` (`compaction_events.go:278`, `failure.go:44`) |

A **presence-typed** attribute — every row marked *iff* — whose underlying value is absent MUST be **omitted**, never recorded as an empty string, a zero or a `false` standing in for absence.

Span status MUST be the error status exactly when the bracket's own outcome is its failure member — run failed, turn aborted, tool execution failure (including the detached arm), compaction fail arm — and the ok status otherwise. The status **description** MUST carry the failure **category name only**, never a wrapped error's text (`R-AGO-004`).

The compaction bracket MUST carry **exactly one** span, of the compaction family. It MUST NOT additionally open a turn span with a compaction child: one bracket, one span.

#### Scenarios

- **S-AGO-010** — Given the merged change, when ADR 0005 is read, then the table above appears as a subsection of § D3 immediately after § D3's Layer 1 allowlist, introduced as an extension of that table, and no sibling ADR carries it.
- **S-AGO-011** — Given that ADR subsection, when its denylist statement is compared byte-for-byte against `0005:252-255`, then it is identical and carries no exception, carve-out or qualifier.
- **S-AGO-012** — Given a traced run driven against the in-memory recording tracer, when the recorded span names are enumerated, then every one belongs to the four families above and the tool spans' names carry the tool name from `ToolStart.Name()`.
- **S-AGO-013** — Given that same recording, when the union of recorded attribute keys is compared against the key set above, then it is a subset of it and contains no other key.
- **S-AGO-014** — Given that same recording, when each present attribute value is compared against the value of its named accessor on the corresponding event drained from the stream, then every comparison is exact, row by row, per the table above.
- **S-AGO-015** — **(bite)** Given a deliberately mutated mapping that records one key under an ad-hoc equivalent spelling, when `S-AGO-013` runs, then it FAILS and names the offending key. Recorded, then reverted.
- **S-AGO-016** — **(bite)** Given a deliberately mutated mapping that records a correct key with a value read from somewhere other than its named accessor, when `S-AGO-014` runs, then it FAILS and names the diverging key — the equality claim detects divergence rather than assuming it.
- **S-AGO-017** — Given a non-delegated run, a turn whose outcome is not the aborted member, a tool call that succeeded, and a compaction that failed, when each span's keys are enumerated, then `cachicamas.run.parent_id`, the turn's `error.type`, the tool's `cachicamas.tool.detached` and `cachicamas.compaction.summary_id` are each **absent** — not recorded as an empty string, a zero or a `false`.
- **S-AGO-018** — Given a run that failed, a turn that aborted, a tool call that failed in execution and a compaction that took its fail arm, when each span's status is inspected, then it is the error status and its description is exactly the failure category name — with no wrapped error text, no message and no payload anywhere in it; and given their successful counterparts, then each status is the ok status.
- **S-AGO-019** — Given a run that compacts, when the spans covering the compaction bracket are enumerated, then exactly one span covers it, it belongs to the compaction family, and no turn span was opened for that bracket.

### R-AGO-003 — Layer 1's request-span keys are NOT re-recorded at Layer 2, and that deferral is stated on the record

The eleven keys ADR 0005 § D3 allowlists for **Layer 1 spans** and that Layer 1 records on its request span — `gen_ai.system`, `gen_ai.request.model`, `gen_ai.request.max_tokens`, `gen_ai.response.finish_reasons`, the four `gen_ai.usage.*` counts, `http.response.status_code`, `retry.count` and `stream.event_count` — MUST NOT appear on any Layer 2 span. They are owned by AI-37's request span (`ai-observability-boundary/R-AOB-004`), and re-recording them at Layer 2 would duplicate one truth in two places that can disagree.

`error.type` is the **one shared key**. It is the standard error key reused on a different span per the convention — not a re-record — and Layer 2 records it only from its own brackets' typed failures, per `R-AGO-002`.

The deferral MUST be stated in the ADR extension itself, naming AI-37 as the owner, so a later contributor inherits a decision rather than a silence and does not re-add them.

#### Scenarios

- **S-AGO-020** — Given a traced Layer 2 run, when the union of Layer 2 span attribute keys is intersected with the eleven Layer 1 request-span keys named above, then the intersection is empty.
- **S-AGO-021** — Given the same recording, when `error.type` is looked for, then it appears only on Layer 2 spans whose own bracket carried a typed failure, and its value is that failure's category name.
- **S-AGO-022** — Given the ADR extension, when it is read, then it names each of the eleven keys as deliberately not re-recorded at Layer 2 and names AI-37 as their owner.
- **S-AGO-023** — Given a traced run in which Layer 1's own request span is also recorded, when the two spans' key sets are compared, then Layer 1's carries its own keys, Layer 2's carries none of them, and neither has been widened to cover the other.

### R-AGO-004 — The content denylist is absolute, enumerated per field, and no denied accessor's value ever reaches telemetry

No span name, no span status description, no attribute key, no attribute value, no span event name, no span event attribute value and no recorded error string MAY carry any prompt, completion, reasoning, tool-argument or tool-result text, any HTTP header name or value, any credential, or any raw provider response body.

The denylist is **absolute at Layer 2**. No content-capture opt-in MAY exist: no configuration input, no field, no flag, no build tag and no environment-derived switch MAY enable recording any denied category.

Because a blanket prohibition is unfalsifiable at review time, the accessors whose values MUST NEVER reach any attribute, span name, status description or span event are **enumerated per field**:

| Denied accessor / value | Where |
|---|---|
| `ToolStart.Arguments()` | `tool_event.go:131` |
| `ToolEndSuccess.Result()` | `tool_event.go:328` |
| `ToolEndResultFailure.Result()` | `tool_event.go:403` |
| `Failure.Unwrap()`, and every `Failure` projection other than `Category().String()` | `failure.go:87`, `failure.go:44` |
| The compaction instruction — `CompactionRequest.Instruction` | `context_strategy.go:93` |
| The compaction summary's content (only the opaque summary identity is recorded) | see `R-AGO-002`'s `cachicamas.compaction.summary_id` row |
| Every message-text and reasoning delta payload | the per-delta events of `R-AGO-005` item 1 |
| Every permission argument byte — `PermissionDecisionRequired.Arguments()` and `PermissionDecisionMade.ModifiedArguments()` | `permission_events.go:173`, `permission_events.go:278` |

The tracing API's error-recording call MUST NOT be made anywhere in Layer 2 production code: it would carry a wrapped error's text into a span event, which the first row of the table above forbids by a different route (AI-37's rule, `R-AOB-007`).

Failure output from the absence scan MUST name the vector. It MUST NOT reprint the bytes it matched.

#### Scenarios

- **S-AGO-024** — Given a full-featured run — one that used tools, reasoning, a permission decision, a compaction and a failure arm — where each denied category is represented by a **distinct, runtime-assembled** marker (assembled by concatenation at test time, so no denied marker appears as a contiguous literal in any source file), when the corpus built from every span name, status description, attribute key and value, span event name and span event attribute value is scanned, then none of the markers is found.
- **S-AGO-025** — Given that same corpus, when every recorded attribute value is rendered, then each is rendered through the tracing API's own value accessor rather than a language-level structure dump — so a marker planted in a non-string-shaped field cannot render as an opaque address and pass (the AI-41 lesson, `S-AOB-027`).
- **S-AGO-026** — Given the shipped Layer 2 production sources, when each accessor in the per-field table above is traced to its uses, then no use flows into a span name, attribute, status description or span event; and when the tracing API's error-recording call is searched for, then it appears in no production source.
- **S-AGO-027** — Given the shipped Layer 2 package, when every configuration input, exported field, flag, build tag and environment-derived switch reachable from it is enumerated, then none enables recording any denied category — the ambient-authority guard (`R-AGP-005`) admits no environment read, so there is no switch for a test to drive.
- **S-AGO-028** — **(bite)** Given a scratch defeat that records `ToolStart.Arguments()` as a span attribute, when `S-AGO-024` runs, then it FAILS and its message names the tool-argument vector **without reprinting the matched bytes**. The red output is recorded, the defeat reverted, and the suite re-run green.
- **S-AGO-029** — Given the absence test's own source, when the shared sweep scans this repository's sources, then it does not flag that source — every needle is assembled at runtime.

### R-AGO-005 — Seven things are deliberately not recorded, each named with its reason

AG-22.1 item 2 requires that what is deliberately **not** recorded be stated, each with its reason (`0003:2068`). The list is closed at the seven below **for AG-22**; a later milestone that records one of them amends this requirement rather than adding an eighth silently.

| Not recorded | Reason |
|---|---|
| 1. Per-delta events — message-text and reasoning deltas, and tool-progress payloads (`tool_event.go:143` onward) | Raw model and tool output: **denylisted content**, and additionally unbounded in cardinality |
| 2. Hook timings | A concrete telemetry hook is Layer 3's (`agent-hook-taxonomy/spec.md:328`); a timing hook would additionally require a wall clock `hooks.go` is forbidden to carry (`R-HKS-008`, enforced at `hooks_test.go:2365-2373`) |
| 3. Permission argument content, and any permission span or permission attribute at all | Permission arguments are tool-argument text: **denylisted**. The gate runs before the tool span opens (`R-AGO-007`), so there is no span for them to land on either |
| 4. Usage and cost — the cost-turn and cost-session payloads | Owned by Layer 1's `gen_ai.usage.*` on the request span (AI-37, `R-AOB-005`); re-recording would duplicate. Deferred, stated on the record — this is `R-AGO-003` applied to a Layer 2 event |
| 5. The compaction instruction and the compaction summary content | The instruction is a prompt and the summary is a completion: both **denylisted**. Only the opaque summary identity is recorded |
| 6. Every `Failure` projection beyond `Category().String()` | `Unwrap()`'s error text is **denylisted**; the delivery, retryability and partial-output projections are safe but gratuitous — no consumer named by this milestone needs them |
| 7. The subagent identity and the compaction bracket's start/end turn identifiers | Opaque and safe, but derivable from the event stream by envelope join. Minimality: nesting is proven **structurally** (`R-AGO-006`), never by an attribute |

#### Scenarios

- **S-AGO-030** — Given the ADR extension, when its not-recorded list is read, then each of the seven items above appears with its own stated reason, and no item is listed without one.
- **S-AGO-031** — Given a traced full-featured run, when the recorded attribute keys and span names are enumerated, then no key or name carries any of the seven — in particular no permission span exists, no delta payload appears, and no `Failure` projection other than the category name appears.

### R-AGO-006 — Spans nest along the existing context chain, and a delegation tree is preserved

Span context MUST thread the **existing** context chain from the run through the turn, the scheduler and the tool invocation. No signature change, no context re-rooting and no context value-stripping MAY be introduced to carry it.

The nesting MUST hold: the turn span's parent is its run's span; the tool-execution span's parent is its turn's span; the compaction span's parent is the span of the bracket that requested it. Layer 1's own request span, when recorded, MUST nest under the turn span by the same mechanism — it receives the span-carrying context unchanged.

**Delegation.** When a tool invocation drives a child run, that child run's span's parent MUST be the delegating tool's span, so a delegation tree reads back as a span tree. This capability MUST state honestly that **no production delegating tool ships**: AG-19 kept every subagent concept in test fixtures, so the claim is proven against those fixtures — the delegating-tool, nested-run and siblings fixtures — with the child harness wired to the same recording provider. The proof surface is the fixtures, and this spec says so rather than implying a production surface exists.

The recording double MUST record the parent linkage it reads from the context. Parenting MUST be established from the context rather than from the tracer instance, so nesting holds across tracer instances and across a parent and child harness.

#### Scenarios

- **S-AGO-040** — Given a traced run that takes at least two turns and executes at least one tool call, when the recorded spans' parent chain is walked, then each turn span's parent is the run span, each tool span's parent is its own turn's span, and the run span has no parent.
- **S-AGO-041** — Given a traced run that compacts, when the compaction span's parent is read, then it is the span of the bracket that requested the compaction.
- **S-AGO-042** — Given the delegating-tool fixture driving a child run under the same recording provider, when the child run span's parent is read, then it is the delegating tool's span — and when two sibling child runs are driven, then each names the same delegating tool span as its parent and neither names the other.
- **S-AGO-043** — Given the recording double, when a span is started from a context carrying no span, then the recorded parent is empty and the span reads back as a root.
- **S-AGO-044** — **(bite)** Given the recording double before it records parent linkage, when `S-AGO-040` runs, then it FAILS because no parent is recoverable — the nesting proof is driven RED first against the substrate gap rather than written after the fact. The red output is recorded.

### R-AGO-007 — A span opens iff its bracket's start event is emitted, and ends exactly once on every exit — including the detached arm

A span MUST be opened **iff** its bracket's Start event is emitted. An exit that emits no Start event MUST get no span: the harness's pre-identity refusals, a turn's pre-emission refusals, a tool call gated out before its start event exists, and the compaction entry's refusals each record nothing.

Once opened, the closing obligation MUST be discharged **uniformly for all exits** by a finalizer registered at open, never at individual return sites. Every span MUST end **exactly once** on every terminal path without exception: normal completion, every typed failure arm, every cancellation shape, a panic unwinding through the frame, and the detached arm below. A span that ends more than once, or that is left unended on any path, MUST fail the suite.

**The detached arm is decided, not implicit.** When the scheduler's bounded wind-down expires with a tool's goroutine still running, the tool-execution span MUST end **at that bound**, on the scheduler's own detached arm — with the error status, `error.type` set to that arm's typed cancellation category, and `cachicamas.tool.detached` set to `true`. The still-running goroutine MUST hold **no span handle**: it never ends, mutates or annotates that span. Any span opened by work that goroutine subsequently hosts belongs to that work's own lifecycle and is ended by that lifecycle's own finalizer, so no orphan is constructible.

Handing the span to the detached goroutine to end "when it really finishes" is **rejected**: it recreates the unbounded-lifetime shape AG-21 removed (`R-CNH-005`) and makes exactly-once unassertable.

A test that drives a detached fixture MUST release and join the detached goroutine **before** asserting exactly-once, so the assertion is not racing the very goroutine the bound abandoned.

#### Scenarios

- **S-AGO-050** — Given a traced run driven to normal completion, when the recording is inspected, then for each of the four families the number of spans started equals the number ended, and no span is left unended.
- **S-AGO-051** — Given a table of run exits — normal terminal, each typed failure arm, and cancellation — when each is exercised under tracing, then in every row every started span ended exactly once.
- **S-AGO-052** — Given a table of tool-execution exits — each of the outcome arms, a start-constructor failure, a panic re-raised from the tool, an execution error, and the detached arm — when each is exercised under tracing, then in every row the tool span ended exactly once.
- **S-AGO-053** — Given a table of compaction exits — the success tail and each failure arm — when each is exercised under tracing, then in every row the compaction span ended exactly once.
- **S-AGO-054** — Given a run refused before it emits any event, a turn refused before its start event, a tool call gated out by the permission gate before its start event exists, and a compaction refused before its started event, when each is exercised under tracing, then **no span** was started for that bracket.
- **S-AGO-055** — Given the detached fixture driven so the wind-down bound expires with the tool still running, when the detached goroutine has been released and joined and the recording is then inspected, then the tool span ended exactly once, at the bound, with the error status, `cachicamas.tool.detached` present and `true`, and `error.type` equal to that arm's typed cancellation category.
- **S-AGO-056** — Given that same fixture, when the detached goroutine's own code is read, then it holds no span handle and calls no span method; and when the recording is inspected after the goroutine finally returns, then the span's end count is still exactly one.
- **S-AGO-057** — **(bite)** Given a deliberately mutated implementation that ends one family's span at a single return site instead of through its finalizer, when `S-AGO-051`, `S-AGO-052` or `S-AGO-053` runs, then at least one row FAILS — an unclosed path is detected rather than assumed closed. Recorded, then reverted.
- **S-AGO-058** — Given a tool whose invocation panics, when the panic unwinds through the traced frame and is re-raised, then the tool span ended exactly once and the panic reaches the same handler it reaches without this change.

### R-AGO-008 — *(non-negotiable)* The absence proof ships its non-vacuity guards, and the needles are proven to bite first

An absence assertion is the easiest test in this repository to write vacuously. `R-AGO-004`'s proof MUST therefore ship all of the following, and the positive control is **not interchangeable** with the corpus guards: a self-test proves the *needles* bite and proves nothing about whether the *corpus* was ever populated.

1. **Positive control** — the shared sweep's self-test MUST run **before** every clean scan and MUST fail the suite if any needle fails to bite its own synthetic corpus.
2. **Exactly-once precondition** — `R-AGO-007`'s exactly-once assertion MUST hold on the same run, so the corpus is not scanned mid-flight.
3. **Corpus-non-empty floor** — the corpus length MUST be asserted greater than zero, the recorded span count at least one, and the recorded attribute count at least the cardinality of the vocabulary the driven run is expected to exercise. The scanner returns *not found* for an empty corpus by construction, so without this floor the whole claim passes on a run that recorded nothing.
4. **Event-kind coverage** — the drained event stream MUST be asserted to contain every event kind the driven run structurally produces — tool, reasoning, permission, compaction and failure kinds among them — so absence is asserted over a run that **used** what it denies rather than over a run that used none.
5. **Every needle non-empty** — each assembled marker MUST be asserted non-empty before it is scanned for; an empty needle matches nothing and would pass silently.
6. **Recorded defeat** — a scratch defeat MUST be planted, watched FAIL naming the vector, and reverted, per `S-AGO-028`.

#### Scenarios

- **S-AGO-070** — Given the absence test, when it runs, then the sweep's self-test runs before the clean scan; and given a needle deliberately made not to bite, when the self-test runs, then the suite FAILS.
- **S-AGO-071** — Given a traced run that recorded nothing, when the absence test runs against it, then the **corpus-non-empty floor** FAILS; and given the real full-featured run, then the corpus length is greater than zero, the span count is at least one, and the attribute count is at least the expected cardinality.
- **S-AGO-072** — Given the drained stream of the absence run, when its event kinds are enumerated, then every kind that run structurally produces is present; and given a run that omitted one of them, when this guard runs, then it FAILS and names the missing kind.
- **S-AGO-073** — Given the assembled needles, when each is measured before scanning, then each is non-empty; and given one deliberately emptied, when the guard runs, then it FAILS.
- **S-AGO-074** — Given the absence run, when `R-AGO-007`'s exactly-once assertion is applied to it, then it holds — the corpus was scanned after every span closed.
- **S-AGO-075** — Given this change's recorded evidence, when it is read, then it contains the defeat of `S-AGO-028` as a recorded RED naming the vector, followed by the reverted GREEN, with the wall-clock duration of an uncached `-count=1` run recorded beside each.

### R-AGO-009 — With no tracer configured, behaviour is identical and nothing panics

WHEN no tracer provider is injected, a run MUST behave **identically**: the drained event sequence of the untraced run MUST equal, element for element and value for value, the drained event sequence of the same scripted run with a recording tracer configured. Nothing MUST panic on any path.

The equality claim MUST carry its own **non-vacuity floor**: the traced arm MUST be asserted to have started at least one span. Without it, the comparison passes trivially when instrumentation records nothing at all — which is the state the claim is meant to distinguish itself from.

#### Scenarios

- **S-AGO-080** — Given one scripted run driven twice — once with no tracer configured, once with the recording tracer — when the two drained event sequences are compared, then they are equal element for element, in the same order, with the same values.
- **S-AGO-081** — Given that same pair, when the traced arm's recording is inspected, then it started at least one span — the equality of `S-AGO-080` is not vacuous.
- **S-AGO-082** — Given a table of terminal paths run with **no** tracer configured — normal completion, each typed failure arm, cancellation, the detached tool arm, a panicking tool, and a compaction failure — when each is exercised, then none panics and each produces the same outcome it produces without this change.
- **S-AGO-083** — **(bite)** Given instrumentation deliberately removed from one family, when `S-AGO-081`'s floor runs against a run that exercises only that family, then it FAILS — the floor detects an unrecording arm rather than assuming one records.

### R-AGO-010 — Layer 2 reaches the OpenTelemetry ecosystem through the API only, over a freshly measured closure

Layer 2's production closure MUST admit only OpenTelemetry **API** paths, and that set MUST be exactly what the design selects out of what ADR 0005 § D3's table (`0005:237-244`) authorises for Layer 2 — a **strict subset** of it. Layer 2 MUST NOT import the ecosystem's root global-getter package, even though § D3's table marks it permitted, and MUST NOT import the metric module, which § D3 permits and this milestone does not select.

The narrowing MUST be recorded rather than inferred, and it MUST be recorded as a **measurement**: the root package's own dependency closure contains a package whose path is literally an auto-instrumentation SDK, and `S-AGP-024` forbids any allowlist entry naming an SDK path — so the global getter is structurally inadmissible under the shipped guard family, not merely inelegant.

The allowlist admitting these paths MUST be amended in the **same commit** as the first production import that needs it, never as a follow-up, and every entry MUST carry its authorising clause in the guard's own source per `agent-package-scaffold`'s `R-AGP-003`. An entry that is **not** a § D3 table row MUST cite the forced-closure clause with its measurement, never a § D3 row it does not have.

A **fresh** closure measurement MUST be taken over Layer 2's own package pattern before and after the change and diffed; nothing beyond the measured set MAY be admitted. An ADR authorises a path, not its closure.

This change MUST add no `require` entry to the module and MUST change no file under Layer 1. Both are already enforced by a shipped test that asserts those diffs empty against the resolved base ref (`hooks_test.go:2386`, `TestHooks_S_AIV_032_ExclusionCheckedAgainstShippedCode`), so a requirement here that implied touching either would be self-falsifying.

#### Scenarios

- **S-AGO-090** — Given the shipped module, when every telemetry import in Layer 2's production sources is enumerated, then each is an OpenTelemetry API path, the ecosystem's root global-getter package is **not** among them, and no metric, SDK, exporter or `otelslog` path is among them.
- **S-AGO-091** — Given this change's recorded evidence, when it is read, then it contains the exact dependency-listing command, its full output and its date, taken **in this worktree** both before and after the production edits, together with the diff of the two — and the design's own pre-implementation measurement (`design.md:141-189`) is cited as the earlier half rather than as the whole.
- **S-AGO-092** — Given that measurement and the allowlist actually written, when they are compared, then every measured non-standard-library package is covered by an allowlist entry and no entry admits a path the measurement does not show.
- **S-AGO-093** — Given the merged diff, when `backend/agent/go.mod`, `backend/agent/go.sum` and every path under `backend/agent/src/ai/` are inspected, then each is byte-unchanged and `TestHooks_S_AIV_032_ExclusionCheckedAgainstShippedCode` passes.
- **S-AGO-094** — **(bite)** Given a scratch production file in Layer 2 importing a package under the ecosystem's SDK path, when the forward import guard runs, then it FAILS naming the offending path and the forbidden-prefix rule that rejected it — proving the widened allowlist did not go over-broad. Recorded, then removed; the merged diff contains no scratch file.

---

## Non-functional requirements

### NFR-AGO-001 — Telemetry costs nothing when unwired

Recording MUST impose no observable behavioural cost when no provider is injected: no additional goroutine, no additional channel, and no change in event ordering or delivery that a consumer can observe. This is the property that makes the tracing API safe below a composition root at all (ADR 0005 § D3), and `R-AGO-009`'s equality proof is its behavioural evidence.

### NFR-AGO-002 — Key literacy

Each attribute key and each span name MUST be spelled in exactly one place and referenced from every recording site, so a rename is a single reviewable edit rather than a search. A recording site MUST NOT assemble a key from parts. The tool span's name is the one composed value, and its composition MUST happen in exactly one place.

### NFR-AGO-003 — Substrate and boundary preservation

The ten substrate files named in `openspec/AGENTS.md` MUST remain byte-unchanged, including `import_boundary_test.go`'s **behaviour**, with exactly two authorised amendments, each recorded rather than silent: its data tables, per `R-AGO-010`; and its exact-path-table direct-import-edge matching mechanism together with its direct-import family scan, per `agent-package-scaffold`'s `R-AGP-003` as amended by this same change. No other behavioural amendment to this file is authorised — in particular, no amendment MAY widen a check's tolerance (for example, exempting an importer class the check previously caught) without the corresponding spec text naming that exemption and the evidence that bounds it. Production sources added by this change MUST NOT import process, filesystem, environment or network facilities; the ambient-authority guard (`R-AGP-005`) and both import-boundary guards MUST pass. No `go.mod` or `go.sum` edit. No edit under `backend/agent/src/ai/`.

### NFR-AGO-004 — Test isolation

Every test added by this capability MUST be safe under `t.Parallel()` and `-race`, which the injection rule of `R-AGO-001` makes achievable: each test owns its own recording provider and no process-global telemetry state is touched. Evidence MUST come from an uncached `-count=1` run; a `(cached)` result is not evidence.

---

## Out of scope at AG-22, with the owner of each

| Item | Owner |
|---|---|
| Exporters, the OTel **SDK**, dashboards and provider registration | The composition root — charter Out-of-scope verbatim (`0003:2053`), per § D3; already denied below `cmd/` by `R-AGP-003`'s forbidden-prefix table |
| Metrics of any kind, and any metric instrument | Non-goal — § D3 permits the path, this milestone does not select it; the charter's deliverable is spans |
| Trace-context propagation across a process boundary, and header injection | Needs a module the measured subset excludes; no charter scenario names it |
| Re-recording Layer 1's request-span keys at Layer 2 | AI-37, per `R-AGO-003` |
| A concrete telemetry **hook** | Layer 3 — `agent-hook-taxonomy/spec.md:328`, doc 0004 CO-24.1 / CO-24.2 |
| A production delegating tool, on which delegation nesting could be proven without fixtures | Not this milestone — AG-19 recorded the fixtures as the surface (`R-AGO-006` states it) |
| Any OpenTelemetry module beyond § D3's table | Each requires **its own ADR** (§ D3's closing blockquote, `0005:257-261`) |
| Every name, receiver, package and file layout implied above | **Design phase** — this spec pins behaviour only |

---

## Acceptance criteria

The capability holds when:

1. Every scenario in this file has recorded evidence.
2. The ADR 0005 § D3 extension is landed with the four span families, the attribute table, the not-recorded list and the denylist restated verbatim.
3. Every bite — `S-AGO-004`, `S-AGO-015`, `S-AGO-016`, `S-AGO-028`, `S-AGO-044`, `S-AGO-057`, `S-AGO-083`, `S-AGO-094` — is recorded RED against real scanned source, then reverted and re-run green.
4. `cd backend/agent && go test -race -count=1 ./...` is green with its wall-clock duration recorded, and `make lint`, `make build` and `make vuln-check` are green with a clean working tree.
5. `backend/agent/go.mod` and `go.sum` are byte-unchanged and no file under `backend/agent/src/ai/` is changed.
6. The `agent-package-scaffold` and `agent-v1-scope` deltas shipping beside this file are promoted in the same pull request.
