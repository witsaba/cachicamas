# Tasks — Layer 1 contract vocabulary

> **Change**: `cachicamas-ai-contract-vocabulary`
> **Milestone**: AI-01 · **Node**: AI-01.1 — The vocabulary `[decision]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-contract-vocabulary/spec.md`, `design.md`
> **Forecast**: **1 PR**, documentation only, zero Go
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Depends on**: AI-00 (`cachicamas-agent-module-scaffold`) merged
> **Blocks**: AI-02, AI-03, and every contract milestone AI-04 … AI-40

---

## Node type and what it means for this task list

AI-01.1 is a **`[decision]` leaf**. Per doc 0002's node grammar:

> **Decision leaf** — `[decision]` — A recorded choice with a closing checklist. **No production code.** Closes when: the decision artifact answers every listed question and is merged.

Consequences that shape this file:

- There is **no test list**, because a decision leaf has none. Behavior leaves carry test lists; this one carries a closing checklist.
- There is **no red-green-refactor cycle**, and this is not a TDD exemption being taken quietly — `openspec/config.yaml` sets `apply.tdd: true` for **Go service code**, and this change writes none.
- There is **no `make test` evidence gate**. doc 0002's global evidence gate binds behavior and guard leaves; a decision leaf closes on its merged artifact.
- The whole milestone is **one phase with one node**, so the usual PR-chain forecast is degenerate: one PR.

---

## Phase AI-01.1 — The vocabulary `[decision]`

The six tasks below are AI-01.1's closing checklist, one task per item, in the checklist's own order. The node closes when all six are struck and the artifact is merged.

**Deliverable of the whole phase:** `openspec/changes/cachicamas-ai-contract-vocabulary/decision.md`.

---

### T-AIV-1 — Request-side terms defined

- [x] Define every request-side term, each with one definition, one owning milestone, and its doc 0001 provenance where the term derives from a recorded section or closes a `C`/`G` identifier.

**Required by the checklist:** role, message, content part and its kinds, tool declaration, tool choice, tool call, tool result, system-instruction segment, cache-boundary marker, generation option, provider escape hatch.

**Also defined** (required by AI-01's acceptance criterion, not by the checklist): message identity, content-part readability, content-part sealing, text content, reasoning content, reasoning state, round-trip token, schema bytes, tool set, argument bytes, normalized request, model identity, request validation, breakpoint cap, invalidation cascade, per-request option override, request rebuild.

**Evidence:** `decision.md` § 3, category `V-REQ`. Every checklist-named term resolves to exactly one row (`S-AIV-008`).

---

### T-AIV-2 — Stream-side terms defined

- [x] Define every stream-side term, with the container nouns preceding the content nouns so that `sequence` is defined **after** and **as a property of** `stream` — C3 encoded as document order (`design.md` § 3.3).

**Required by the checklist:** event, event kind, payload, sequence, stream, terminal event, delta, block index, call ordinal.

**Also defined:** carrier, producer, consumer, stream ownership, cancellation, abandonment, bounded buffer, sanctioned loss path, block, ordering invariant, response-start event, completion event.

**Special case handled:** `call ordinal` is grouped stream-side per the checklist, owned by AI-09 per the concept's origin, with AI-18.3 and AI-30.5 recorded as restatements rather than second definitions (`S-AIV-003`).

**Evidence:** `decision.md` § 4, category `V-STR`.

---

### T-AIV-3 — Metadata terms defined

- [x] Define the terminal metadata a response carries, including every value of the closed finish-reason vocabulary, and state the absence-versus-zero distinction as a definitional property rather than an implementation note.

**Required by the checklist:** finish reason, usage, token-count field, absence versus zero.

**Also defined:** the seven finish-reason values (natural stop, length, tool calls, content filter, refusal, pause-turn, unknown) and cost formula.

**Evidence:** `decision.md` § 5, category `V-MET`. Refusal and pause-turn are present from birth, closing `G12(c)` at the vocabulary level before AI-13 implements them.

---

### T-AIV-4 — Failure terms defined and separated

- [x] Define the failure vocabulary with **two orthogonal separations** stated as such: the owner split (caller-contract, AI-04 · provider/transport, AI-19) and the delivery split (pre-stream · mid-stream).

**Required by the checklist:** caller-contract failure (AI-04's territory) versus provider/transport failure (AI-19's), plus the pre-stream versus mid-stream delivery split.

**Also defined:** validation sentinel, positional context, first-failure ordering, failure category, retryability, partial output, partial-output discriminator, terminal error event, safe metadata, redaction, Layer 1 retry policy.

**Additional obligations:**
- The boundary between the two owners is stated as a **rule a reader can apply to a new case**, not as two examples (`S-AIV-010`).
- At least one **borderline case is worked**, which is what AI-04.1's own closing checklist will need (`S-AIV-012`).
- The delivery split records that AI-02.1 item 5 decides the observable shape and AI-19.5 implements it as one vocabulary over two paths (`S-AIV-011`).

**Evidence:** `decision.md` § 6, category `V-FAIL`.

---

### T-AIV-5 — Excluded terms listed with their owners

- [x] List every term Layer 1 deliberately does not own, each with a named owner, and — where the exclusion is easy to misread — the Layer 1 neighbour it is confused with. **No excluded term is defined** (`S-AIV-015`).

**Required by the checklist:** agent turn, transcript, session, tool execution, permission, compaction, cost, price, frontend.

**Also listed** (appended for completeness, since a charter from AI-02 … AI-40 may reach for any of them): loop termination, retry and failover policy above Layer 1, delegation and subagent, hook, provider catalog and credential resolution, skill and prompt and project instruction, sandbox, tool source.

**Evidence:** `decision.md` § 8, category `V-OUT`. All nine checklist terms present with a non-Layer-1 owner (`S-AIV-013`); `transcript` explicitly distinguished from `message` (`S-AIV-014`).

---

### T-AIV-6 — The two wording traps restated verbatim

- [x] Quote both wording traps from doc 0002's *Layer boundary* section **character for character**, each with the record that it has already caused one wrong decision, and each with the operational consequence for the register's own rows.

**Trap 1** — "Layer 1 does not know what a tool is" is too broad.
**Trap 2** — "Provider swap is a config change" applies only after adapters exist.

**Additional obligation:** the register's tool-related rows must be consistent with both halves of trap 1 — Layer 1 owns the transport representation; Layer 1 does not execute, resolve names, or own application behavior (`S-AIV-030`).

**Evidence:** `decision.md` § 2, quoted as block quotations. Verified by character comparison against doc 0002's *Layer boundary* paragraph (`S-AIV-029`).

---

## Verification pass (closes the milestone)

Run after T-AIV-1 … T-AIV-6. Every check is inspection; nothing executes.

- [x] **V-1** — Every term appears in exactly one row; recurrence is a cross-reference, never a second definition (`R-AIV-002`).
- [x] **V-2** — Every non-excluded row names exactly one owning milestone, and that milestone's charter in doc 0002 covers the term (`R-AIV-003`).
- [x] **V-3** — Every `AI-NN` identifier in `decision.md` that attributes ownership or provenance resolves against a doc 0002 milestone heading, and none is a retired identifier from doc 0001 or ADR 0005 (`R-AIV-008`, `S-AIV-018`). The single permitted exception is the retired-identifier warning in `decision.md` § 1, where each retired number is named beside its current replacement. Translation table: `design.md` § 4.2.
- [x] **V-4** — No Go type, field, method, interface, or package identifier appears anywhere in the change (`R-AIV-009`).
- [x] **V-5** — The change's diff contains only markdown under `openspec/changes/cachicamas-ai-contract-vocabulary/` (`R-AIV-010`, `S-AIV-021`).
- [x] **V-6** — Term identifiers are unique within their category and contiguous from 01 (`R-AIV-004`).
- [x] **V-7** — **AI-02's charter is walked noun by noun** and every domain noun resolves to a register row: carrier, ownership, cancellation, buffering, failure-delivery split (`S-AIV-026`).
- [x] **V-8** — **AI-03's charter is walked noun by noun** and every domain noun resolves: capability, required capability, optional capability, capability discovery, capability record (`S-AIV-027`).
- [x] **V-9** — Spot-check three further charters (AI-06, AI-13, AI-19) for unresolvable domain nouns; any finding is closed by amendment under `R-AIV-011`, not by inventing the term downstream (`S-AIV-028`).

---

## Review focus

For the reviewer, in priority order — the first three are where a defect is expensive and the rest are where it is cheap to catch:

1. **Owner mis-assignment.** A term owned by a milestone whose charter does not cover it is the shape of C4. Read the owner column against doc 0002's charters, not against plausibility.
2. **Retired identifiers.** doc 0001 § 3.1 says C4 is fixed by "AI-18", and AI-18 exists today as a *different* milestone. A copied citation points at a real, wrong milestone. Check every `AI-NN`.
3. **Over-reach.** Apply `design.md` § 5's test: if a sentence were deleted, would a later milestone have fewer options? If yes and the milestone is not AI-01, it is over-reach. Highest risk in the AI-02, AI-03, AI-04 and AI-06 rows.
4. **Trap fidelity.** Both traps must be verbatim. Paraphrase is how trap 1 became too broad in the first place.
5. **Leaked Go identifiers.** Term names are noun phrases with spaces; a single-token camel-case name is the anomaly to look for.
6. **Silent duplication.** Two rows describing the same concept under different names — for instance a stream-side row that re-defines `call ordinal` instead of cross-referencing AI-09.

---

## PR forecast and review budget

| PR | Content | Forecast | Depends on |
| --- | --- | --- | --- |
| 1 | six markdown artifacts under the change directory | ~1,400 lines of prose, **0 Go** | AI-00 merged |

doc 0002's review budget — "prefer less than 250 changed lines; stop and reassess before 400" — is a **code** budget, expressed in the same document that requires each milestone's SDD to carry proposal, spec, design and tasks artifacts. A decision leaf's diff is entirely those artifacts. No chaining applies: splitting a vocabulary across pull requests would produce exactly the partial-definition state the milestone exists to prevent.

---

## Acceptance criteria for the milestone

1. All six closing-checklist items (T-AIV-1 … T-AIV-6) are answered in `decision.md`.
2. The verification pass V-1 … V-9 is recorded as complete.
3. `spec.md`'s `R-AIV-001` … `R-AIV-013` hold.
4. The change adds six markdown files and modifies nothing else.
5. AI-02's and AI-03's SDD agents can be handed the register and write their charters from it without inventing a term.

## Next

- **AI-02** — `cachicamas-ai-stream-lifecycle`: carrier, ownership, cancellation, buffering, failure delivery. Consumes `V-STR-01` … `V-STR-09` and `V-FAIL-11`/`V-FAIL-12`.
- **AI-03** — `cachicamas-ai-minimum-capabilities`: the capability matrix and discovery mechanism. Consumes `V-PRV-06` … `V-PRV-09`.
