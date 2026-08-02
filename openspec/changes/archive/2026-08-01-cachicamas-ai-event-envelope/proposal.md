# Proposal: Event envelope with per-stream sequencing (AI-14)

## Intent

`backend/agent/src/ai` has no event type, so Wave 2 (AI-15…AI-20) cannot start. The retired design kept the sequence in a process-global atomic counter (defect **C3**): only the first stream in a process began at 1, and a shipped test recorded that gap as expected. Fix: put the counter where the stream is, and state ordering rules a consumer can run.

## Scope

### In Scope

- `Event` envelope: kind **derived** from a sealed unexported payload interface (mirrors `content_part.go`); payload-less event rejected with AI-04's `ErrNotInVocabulary`; AST-checked registry witness table so **C4** cannot return.
- Per-stream sequence stamper: value type, one instance per stream, no atomics (AI-02 §4 single-producer); documented unstamped sentinel rejected at the producer boundary.
- AI-14.3 guard: package-wide AST scan of non-test `src/ai` for package-level mutable **sequence** state, with an explicit allowlist entry for `message.go`'s `lastMessageID` and its recorded C3 rationale.
- AI-14.4 runnable ordering-invariant checker over a recorded stream.
- Spec citing `V-STR-10`…`V-STR-18` by identifier (append, never re-define).

### Out of Scope

- Concrete kinds: response start/completion (AI-15), text/reasoning/tool blocks (AI-16…18), terminal error (AI-19). **AI-14 registers zero production kinds.**
- Gap-detection helpers (AI-22.3), conformance enforcement (AI-23), provider interface (AI-20).
- Reopening AI-02 §3–§7; new AI-04 sentinels; changing `lastMessageID`.

## Capabilities

### New Capabilities
- `ai-event-envelope`: event, kind, payload, per-stream sequence, block/index/delta shape, ordering invariants.

### Modified Capabilities
- None. AI-02 and the register are cited, not changed.

## Approach

Mirror `content_part.go` (derived kind, sealed payload, slice-ordered kind names). Two decisions this proposal settles:

**D1 — invariants are payload-independent.** The checker reads only `(kind, block index, descriptor)`. A descriptor is `{blockRole: none|start|delta|end, cardinality: any|at-most-one, terminal: bool}`. No payload type is referenced, so AI-15/AI-19 land without AI-14 knowing them.

**D2 — for AI-15, read this: the checker generalizes.** "At most one event of kind X" and "kind X is terminal; nothing follows" are **reusable descriptor-driven primitives**, not block-triple-specific logic. AI-15 registers response-start as `at-most-one` and completion as `terminal`; AI-19 registers its error the same way — **no AI-14 change**. Required by AI-15.1 test 2 and AI-15.2 test 2, which delegate to "the AI-14.4 invariants".

Because zero production kinds ship, derivation/sealing/external readability are proved with a test-only payload exposed via `export_test.go` to `package ai_test`.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/ai/` event/sequence/invariant files + tests | New | All AI-14 code |
| `backend/agent/src/ai/validation.go` | Read-only | Reuse `Invalid`/`FirstFailure`/`ErrNotInVocabulary` |
| `backend/agent/src/ai/message.go` | Read-only | Allowlisted `lastMessageID` |
| `backend/agent/src/ai/doc.go` | Possibly modified | Package comment |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| Zero-kind registry makes AI-14.1.3's table vacuous | Med | Test-only witness payload; table becomes real at AI-15 — confirm at sdd-design |
| AI-15 assumes a different invariant surface | Med | D2 stated here; reconcile at spec/design, not apply |
| AST guard under-matches a wrapped counter | Med | Match underlying `int*`/`uint*`/`atomic.*` recursively; bite proof required |

## Rollback Plan

Purely additive greenfield: revert the PR (or delete the new `src/ai` event/sequence/invariant files and restore `doc.go`). No persisted data, no migration, no external consumer — nothing imports the envelope until AI-15.

## Dependencies

- AI-02 (`ai-stream-lifecycle`, shipped) and AI-04 (`ai-validation-errors`, shipped).
- Register `ai-contract-vocabulary` §4 rows `V-STR-10`…`V-STR-18`.
- Blocks AI-15…AI-20.

## Success Criteria

- [ ] Two concurrent streams each start at 1 and are independently contiguous under `-race`; a third, started after both finish, also starts at 1.
- [ ] A mismatched or payload-less envelope cannot be constructed; the never-stamped sentinel is rejected at a stated boundary.
- [ ] The AI-14.3 scan fails on a scratch package-level sequence counter and passes with `lastMessageID` allowlisted.
- [ ] Ordering invariants run as code against a recorded stream and expose at-most-one/terminal as reusable primitives.
- [ ] `make test` green from `backend/agent/`.
