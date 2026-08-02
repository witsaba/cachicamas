# Exploration: cachicamas-ai-event-envelope (AI-14 — event envelope + per-stream sequencing)

> Persisted from Engram observation `#2144` (topic `sdd/cachicamas-ai-event-envelope/explore`).
> The explore executor had no file-write tool; this file is the on-disk copy of that artifact.

## Current State

- Package `backend/agent/src/ai` (Layer 1) currently has NO event/stream/sequence file. AI-14 is greenfield in this package: existing files are cache_boundary, content_part(+registry/internal tests), doc.go, finish_reason, json_syntax, message, reasoning_content, request(+extension), role, system_instruction, text_content, tool(+call/choice/result/set), usage, validation.
- AI-02 (`openspec/specs/ai-stream-lifecycle/spec.md`, live, shipped) already decided and AI-14 MUST NOT reopen: carrier = receive-only channel of events (§3); ownership = exactly one producer goroutine, one closing site (§4); cancellation/abandonment (§5); bounded buffer starting 64, backpressure = waiting-never-dropping, one sanctioned loss path (§6); failure delivery boundary = handover not first event (§7). §10 states verbatim what AI-14 inherits vs owns: "The producer stamps sequence and owns the stream's state... §4's single-sender rule is the structural reason AI-14.2's 'two concurrent streams each start at 1' is achievable without coordination — and the reason AI-14.3's guard against a package-global has something to guard." NOT inherited (AI-14 owns): event kinds, payloads, sequence semantics, ordering invariants.
- Register (`openspec/specs/ai-contract-vocabulary/spec.md` §4) already defines: V-STR-10 event, V-STR-11 event kind (derived from payload, closed, every kind constructible), V-STR-12 payload (payload-less event does not exist — C4 if reintroduced), V-STR-13 sequence (1-based, contiguous, PER STREAM, assigned by producer, explicitly NOT a process-wide counter — this is C3's fix), V-STR-14 block, V-STR-15 block index, V-STR-16 delta (index + only new fragment, never a snapshot), V-STR-17 ordering invariant, V-STR-18 terminal event (exactly one per stream; its two instances V-STR-20 completion event owned by AI-15 and V-FAIL-10 terminal error event owned by AI-19 — NEITHER exists yet).
- AI-04 (`backend/agent/src/ai/validation.go`, shipped) already provides everything AI-14.1.2 needs: `Violation` type, `Invalid(rule error, at ...Step) *Violation`, `FirstFailure`, closed sentinel set (ErrEmpty, ErrNotInVocabulary, ErrOutOfRange, ErrMalformed, ErrUnresolvedReference, ErrDuplicate, ErrMisplaced), closed and append-only (R-AIE-003, must append not stretch). `content_part.go` already uses `ErrNotInVocabulary` for "no payload" — direct precedent for the event's payload-less rejection; no new sentinel needed.
- `content_part.go` is the structural precedent for the envelope shape: PartKind is derived from payload via `Part.Kind()` calling `p.payload.kind()`, never stored; closed kind vocabulary is an unexported `partPayload` interface (sealing, only this package can implement); `partKindNames` is a slice not a map (AI-04's "no unordered iteration decides anything"). `content_part_registry_test.go` (AI-06.4) is the exhaustiveness-guard template for AI-14.1.3: AST-parses the package's own const declarations and cross-checks them against the registry and a witness table with per-kind legs (constructor-with-rules, accessor, validation-path).
- CRITICAL FINDING for AI-14.3: `message.go` already carries a package-level mutable counter, `var lastMessageID atomic.Uint64` (used by `mintMessageID()`). Its own doc comment (lines 82-96) explicitly discusses defect C3 and explains why this counter is NOT that defect: MessageID (V-REQ-03) only needs "two messages are distinguishable" (satisfied by any process-wide monotonic counter), while V-STR-13 sequence needs the much stronger "first event of every stream carries 1, contiguous" property that a process-global counter structurally cannot give. AI-14.3's "no package-level mutable sequence state" guard MUST NOT be a blanket "no package-level counter" scan — that would false-positive on the already-shipped, deliberately-justified `lastMessageID`. The guard must be scoped specifically to sequence state, not counters generally.
- `import_boundary_test.go` (AI-00.3) and `content_part_registry_test.go` (AI-06.4) are this package's two existing "mechanical AST scan" precedents; both use a forbidden-first/allowlist shape with each entry carrying the rule it enforces. AI-14.3's guard should follow this idiom: scan `src/ai`'s whole non-test source (not just new files, per the audit-upstream lesson) for package-level `var` of integer/`sync/atomic.*` type, fail on any found except an explicit, reasoned allowlist entry for `lastMessageID`.
- Engram tool limitation: `mem_search`/`mem_get_observation`/`mem_context` were NOT exposed to the explore executor session (only `mem_save` was available), so the prior Wave 1 learning docs (`learning/sdd-rework-patterns`, `learning/sdd-spec-inconsistency-patterns`) and the AI-02/AI-04 prior-session notes could not be retrieved programmatically. All AI-02/AI-04 grounding in this exploration instead comes from direct reading of the shipped spec/code files, which independently corroborates (but may not fully cover) the two learning docs' content. The orchestrator should re-pull those two learning docs before sdd-design if full coverage matters.
- File-write tool was unavailable in the explore executor session (Read/WebFetch/WebSearch/mem_save/codegraph_explore/Grep/Glob only). This file is the persisted copy required by that note.

## Affected Areas

- `backend/agent/src/ai/` (new files expected: an event/envelope file, a sequence/stamping file, an ordering-invariant checker file, plus their tests) — this is where AI-14 lands; no existing file needs modification except possibly `doc.go`'s package comment.
- `backend/agent/src/ai/validation.go` — read-only dependency; AI-14.1.2 reuses `ErrNotInVocabulary` via `Invalid()`/`FirstFailure()`, adds no new sentinel.
- `backend/agent/src/ai/content_part.go`, `content_part_registry_test.go` — structural and test-shape precedent to mirror, not modify.
- `backend/agent/src/ai/message.go` — read-only; its `lastMessageID` counter is the precedent AI-14.3's guard must explicitly distinguish from and allowlist.
- `backend/agent/src/ai/import_boundary_test.go` — precedent for the AST-scan guard idiom AI-14.3 should follow.
- `openspec/specs/ai-stream-lifecycle/spec.md` §10 and `openspec/specs/ai-contract-vocabulary/spec.md` §4 — normative sources AI-14's spec must cite by `V-*` identifier, per register discipline (append not invent).

## Approaches

### Envelope shape

1. **Interface-derived-kind envelope (mirrors `content_part.go`)** — one exported `Event` struct with an unexported `payload` field implementing an unexported `kind()`-bearing interface; `Event.Kind()` asks the payload. RECOMMENDED.
   - Pros: exact proven precedent in this package; satisfies charter test 1 (kind cannot disagree with payload) and test 4 (readable externally without a type switch over unexported types) by construction; reuses AI-04 sentinels and the `Part`/`partPayload` sealing idiom reviewers already know.
   - Cons: none material; requires writing a per-kind witness-table guard test (AI-14.1.3) mirroring AI-06.4's ~600-line test, which is real but bounded effort.
   - Effort: Medium.
2. **Tagged struct with explicit `Kind` field set by each constructor** — REJECTED. Directly contradicts charter test 1 ("a caller cannot set a kind that disagrees with what the event carries"); a caller-settable field can disagree with the payload by construction.
3. **Interface-per-event-kind (distinct concrete types satisfying an `Event` interface)** — REJECTED. Charter test 4 requires reading kind/sequence/payload "without a type switch over unexported types"; per-kind concrete types force exactly that at every consumer.

### Sequencing

- **A. Per-stream stamping value type, instantiated once per (future) producer/stream, no atomics required** — RECOMMENDED. AI-02 §4's single-sender-per-stream rule means only one goroutine ever calls it per stream, so plain (non-atomic) state suffices for that stream; different streams get independent instances, so the `-race` test (AI-14.2 item 2) is satisfied trivially by having zero shared state between streams.
  - Pros: structurally excludes defect C3 (no state to be global); directly satisfies AI-14.2 items 1, 2, 4 ("no residual process state").
  - Cons: exact type name/exported-ness is undecided pending AI-20 (provider interface, not yet built) — a real open question for sdd-design, not a blocker.
  - Effort: Low-Medium.
- **B. Package-global atomic counter** — REJECTED. This is literally defect C3, the reason the milestone exists; excluded explicitly by the charter and by AI-14.3's guard.
- **C. Caller-supplied sequence number, unchecked** — REJECTED. Contradicts V-STR-13 ("assigned by the producer"); cannot structurally guarantee the 1-based/contiguous invariant, since nothing stops two different callers from independently supplying the same or non-contiguous numbers.

### AI-14.3 guard

- **A. Blanket "no package-level mutable state" AST scan** — REJECTED. False-positives on the already-shipped, already-justified `lastMessageID` in `message.go`.
- **B. Scoped scan with an explicit, reasoned allowlist** (mirrors `import_boundary_test.go`'s forbidden-prefix/allowlist shape) — RECOMMENDED. Scan the whole `src/ai` non-test source for package-level `var` of integer or `sync/atomic.*` type; fail on any found except an explicit, commented allowlist entry for `lastMessageID` (with rationale echoing message.go's own comment).
  - Pros: matches the codebase's own established idiom for this exact kind of guard; audits the whole package rather than only new files, catching a stray global reintroduced anywhere later.
  - Cons: needs careful AST-type matching to avoid missing an obfuscated counter (e.g. wrapped in a small unexported struct) — a real design risk, see below.
  - Effort: Medium.
- **C. Scan restricted to only this milestone's new files** — REJECTED. Does not audit upstream/future code; a global added later elsewhere in the package would be invisible to a file-scoped guard. This is the same category of risk the Wave 1 rework notes flagged: a dependency/state-purity guard must audit the whole surface, not just the milestone under test.

## Recommendation

Adopt the `content_part.go`-mirrored envelope shape (kind derived from an unexported sealed payload interface, AI-04 sentinels reused, no new vocabulary invented locally), a per-stream (non-global, non-atomic, single-writer) sequence stamper for AI-14.2, and a package-wide AST-scan guard with an explicit, reasoned allowlist entry for `lastMessageID` for AI-14.3. All three choices are directly grounded in shipped, reviewed precedent already in this exact package, which minimizes both design risk and reviewer surprise. Defer the exact type names and exported-ness of the sequence stamper to sdd-design, since AI-20 (its eventual caller) has not landed.

## Risks

- Engram `mem_search`/`mem_get_observation` were unavailable to the explore executor; the two Wave 1 learning docs (rework-patterns, spec-inconsistency-patterns) were not retrieved. Direct code reading independently surfaced the `lastMessageID`/C3 distinction, but full coverage of all eight rework episodes and nine spec defects is not confirmed.
- No file-write tool was available to the explore executor; this file was persisted afterwards by sdd-propose.
- AI-14.4's ordering-invariant checker needs to state "exactly one terminal event, nothing follows it" in terms of a closed EventKind vocabulary, but the terminal event's two instances (completion event V-STR-20, terminal error event V-FAIL-10) belong to AI-15 and AI-19, neither of which has landed. sdd-design must decide how AI-14 expresses "terminal-ness" as a structural property of its own closed kind set without inventing AI-15/AI-19's payloads prematurely.
- The AST-based guard for AI-14.3 risks under-matching if a future contributor wraps a counter in an unexported struct or renames the type; sdd-design/sdd-tasks should specify the exact AST match rule (e.g. match on underlying type `int*`/`uint*`/`atomic.*` recursively through struct fields, not just top-level var type) so the guard's bite proof ("a scratch package-level counter fails the scan") is robust to at least one layer of indirection.

## Ready for Proposal

Yes. The envelope shape, sequencing approach, and guard design are each grounded in existing shipped precedent in this package, and the two Wave 1 tool-availability gaps (Engram search, file write) are documented but do not block proposing AI-14's scope.
