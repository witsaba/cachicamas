# Proposal — translate usage and finish reasons

> **Change**: `cachicamas-ai-provider-completion` · **Milestone**: AI-31 · **Nodes**: AI-31.1, AI-31.2, AI-31.3
> **Phase**: proposal · **Date**: 2026-08-04 · **Project**: cachicamas (witsaba)
> **Wave**: 4 "Connect the vendor" · chain base: the AI-28 chain head (`feat/ai-28-7-pre-decode-checks`)
> **Depends on**: AI-13, AI-28 · **Parallel with**: AI-29, AI-30

---

## Intent

The adapter's terminal event is half-built. `buildCompletion` (`stream_state.go`) carries a finish reason and a
two-field usage record: `prompt_tokens → Usage.Input`, `completion_tokens → Usage.Output`. `usageFromWire`
(`chunk.go`) states the gap in its own comment: "CacheRead, CacheWrite and Reasoning are never touched here:
they stay absent by construction for every transcript." The vendor reports all three
(`prompt_tokens_details.cached_tokens`, `.cache_write_tokens`, `completion_tokens_details.reasoning_tokens`,
**C8**), and AI-13.4's cost formula — `Input + CacheRead + CacheWrite + Output`, with `Input` **exclusive** of
the cache terms — is today true of the neutral type and unproven of this adapter.

AI-31 closes that: every metadata field the wire actually carries reaches the completion event, none is
invented, and the two mappings whose vocabulary the wire **cannot** express are recorded as unreachable rather
than faked.

## Scope

### In scope

1. **Finish-reason reachability, recorded (AI-31.1 items 1–3)** — the five-member wire enum mapped onto the
   neutral vocabulary, with the unreachable neutral values enumerated and their unreachability recorded as a
   cited fact of this dialect (table below).
2. **Stop-sequence disposition (AI-31.1 item 4)** — the wire carries no matched-stop-value field anywhere in
   the chunk, choice or delta schema. Disposition: **recorded deliberate absence**, backed by a new NEGATIVE
   citation, not a silent omission.
3. **Usage detail-field mapping (AI-31.2 item 1)** — `cached_tokens → CacheRead`, `cache_write_tokens →
   CacheWrite`, `reasoning_tokens → Reasoning`; absent stays absent, reported zero stays a reported zero.
4. **AI-13.4 inclusive-versus-exclusive reconciliation (AI-31.2 item 3)** — `Usage.Input` is **exclusive** of
   the cache terms by neutral contract. If the vendor's `prompt_tokens` is inclusive of `cached_tokens`, the
   adapter subtracts before filling `Input`, and the cost formula is verified against a real cache-hit
   transcript. **The inclusivity itself is uncited today** — see the citation gate.
5. **Multi-frame usage merge (AI-31.2 item 2)** — this dialect's schema gives one populated usage chunk
   (**C4**), so the merge is *dialect-conventional*, not schema-given: the landed D10 rule ("last populated
   wins, never fold") is pinned by a fixture proving no double-count when several populated usage frames
   arrive, as some servers sharing this dialect emit.
6. **Never invent, never assume position (AI-31.3)** — usage gathered wherever it appears; an omitted field or
   an unexpected frame position neither invents a value nor crashes; a metadata-only frame stays tolerated.
7. **Charter-boundary delta on `ai-provider-text-stream`** — `R-ATS-026`/`S-ATS-101` currently assert the
   shipped usage handling carries "no per-field taxonomy that AI-31.2 owns". Landing AI-31 makes that
   inspection scenario false; it is amended in scope, never quietly invalidated.

### Out of scope

- **Widening the neutral vocabulary or `ai.Completion`'s field set.** `ai.Completion` carries `reason` and
  `usage` and nothing else; there is no raw-provider-label field, and adding one is an AI-13 change with its
  own proposal.
- **Relaxing the raw-string-strict finish gate (`R-ATS-010`/`S-ATS-039`).** See decision D1.
- **Extending the conformance bridge to `CapCompletionMetadata`.** See decision D3.
- **`total_tokens`.** The neutral record has no term for it and AI-13's formula does not want one; it stays
  undecoded, as `chunk.go` already records.
- **Reasoning *content* (AI-29) and tool-call mapping (AI-30).** `reasoning_tokens` is a count, not content;
  mapping it does not open AI-29's block question.
- **Any new module dependency.** `backend/agent/go.mod` carries zero requires. Stdlib only.

## Capabilities

### New Capabilities

- `ai-provider-completion`: the OpenAI-compatible adapter's terminal metadata mapping — finish-reason
  reachability and its recorded unreachable values, stop-sequence disposition, full usage field mapping with
  AI-13.4 cache semantics, multi-frame merge discipline, and never-invent/position-independence.

### Modified Capabilities

- `ai-provider-text-stream`: `R-ATS-026` / `S-ATS-101` — the charter boundary that reserved the usage field
  taxonomy and the merge for AI-31 must be re-scoped to "as of AI-28" once AI-31 discharges it. No other
  requirement of that capability changes under decision D1.

## The reachability reconciliation

Wire enum (**C2**, five members) → neutral vocabulary (`finish_reason.go`, seven members):

| Wire value | Neutral value | Basis |
| --- | --- | --- |
| `stop` | `FinishReasonStop` | C2 + `NormalizeFinishReason` table |
| `length` | `FinishReasonLength` | idem |
| `tool_calls` | `FinishReasonToolCalls` | idem |
| `function_call` (deprecated) | `FinishReasonToolCalls` | idem |
| `content_filter` | `FinishReasonContentFilter` | idem |

**Unreachable neutral values on this dialect — enumerated, not discovered later:**

| Neutral value | Why unreachable here |
| --- | --- |
| `FinishReasonRefusal` | No wire `finish_reason` value spells refusal. The delta schema does carry a `refusal` field (**C7**), but which `finish_reason` accompanies it is undocumented; mapping its presence to the refusal outcome is inference, which `R-ATS-027` forbids recording as fact. |
| `FinishReasonPauseTurn` | No pause channel exists anywhere in C1/C2/C7. AI-31.1's note about lossy pause-resume is therefore **vacuous for this adapter**: no paused response can arrive, so no skipped block can be replayed wrong. |
| `FinishReasonUnknown` | Unreachable **by design**, not by omission: the landed gate rejects any out-of-enum value as a typed malformed response (`S-ATS-039`), so an unrecognised stop value never becomes a completion at all. |

**Correction to a framing this change inherited.** `content_filter` must **not** map to the refusal-family
value. `finish_reason.go` states the line explicitly at `FinishReasonContentFilter`: "A content filter is a
decision taken *beside* the model… none of which is the right response to a model that declined." Collapsing
them would erase exactly the distinction AI-13.1 paid a vocabulary slot for.

## Three decisions this proposal surfaces, for `sdd-spec` / `sdd-design`

**D1 — AI-31.1 item 2 versus the landed strict gate.** Doc 0002 asks that "a novel vendor stop value maps to
unknown without error, with the raw label preserved". The landed `rawStrictFinishReason` returns
`errUnrecognizedFinishReason` instead, and `S-ATS-039` pins that as a *test*. **Recommended reading**: item 2
is discharged at the **normalizer** level — `ai.NormalizeFinishReason` is already total, crash-free and
round-tripped, which is the bug class the item names — while this dialect's schema-closed enum keeps strict
rejection at the adapter. Adopting the alternative reading means amending a requirement that shipped days ago
and losing a malformed-response detector. Either way it is stated in the spec, not left implicit. Note also
that "the raw label preserved" has **no neutral home**: `ai.Completion` has no such field.

**D2 — the refusal channel.** Either (a) record `FinishReasonRefusal` as unreachable on this dialect in v1, or
(b) map `delta.refusal` presence to it behind a *dialect-conventional* label with a committed transcript
fixture. (b) also forces a second question — whether refusal text becomes a text block or stays skipped by
`R-ATS-017` — which (a) does not open. **Recommended**: (a) for v1, with the obligation named.

**D3 — the conformance bridge.** `bridge_test.go`'s `writeTerminalChunk` renders `reason.String()` straight
onto the wire. Extending `RunConformanceFor` to `CapCompletionMetadata` would emit `"refusal"`,
`"pause_turn"` and `"unknown"` — all rejected by the strict gate — so
`finish_reason/all_seven_values_reachable_drift_guarded` **would fail against the real adapter**. It stays out
of scope, with the obligation recorded against **AI-38.2**'s capability report, where declared-capability
standing is decided. This is the interaction the milestone must not meet for the first time at apply.

## The citation gate — six claims are not yet cited

`R-ATS-027` binds: inference must not be recorded as fact. `citations.md` for this change must add, against the
same pinned `openai-openapi` commit `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439`:

| Claim | Why it is load-bearing |
| --- | --- |
| **C9** — is `prompt_tokens` inclusive of `prompt_tokens_details.cached_tokens`? | Decides whether `Input = prompt_tokens − cached_tokens` or `= prompt_tokens`. Getting it wrong double-counts or under-reports every cached token. |
| **C10** — is `prompt_tokens` inclusive of `cache_write_tokens`? | AI-13.4 requires `Input`, `CacheRead`, `CacheWrite` disjoint. |
| **C11** — is `completion_tokens` inclusive of `reasoning_tokens`? | The neutral record requires `Output` to *contain* `Reasoning`. |
| **C12** — NEGATIVE: no matched-stop-sequence field in chunk, choice or delta | AI-31.1 item 4's recorded-absent disposition rests on it. |
| **C13** — NEGATIVE: no `finish_reason` value for refusal or pause; `delta.refusal` is the only refusal-shaped channel and its companion `finish_reason` is undocumented | D2's basis. |
| **C14** — can more than one populated `usage` frame occur? | Separates what **C4**'s schema gives from what needs a dialect-conventional fixture pin. |

If C9/C10/C11 cannot be cited, the affected field is mapped **without** an arithmetic adjustment and the
uncertainty is recorded — never silently subtracted.

## Approach

**Extend `usageFromWire`, do not rewrite it.** The landed absent-versus-zero mechanism — a `*int64` per field,
present only when the key arrived — is the right shape for seven fields as it was for two. Two optional nested
structs join `wireUsage`; each field keeps the pointer-nil-on-absence discipline. `S-ATS-055 … S-ATS-062` stay
green untouched: their transcripts carry no detail objects, so every new count is absent and `Input` is
unchanged by an absent subtrahend.

**Keep the merge rule where it already lives.** D10's "last populated wins, never fold" is already the
non-double-counting answer for a cumulative-per-frame vendor; AI-31.2 item 2 adds the proof, not the mechanism.

**Finish-reason work is mostly recording, not code.** Under D1 the mapping already exists and is tested; what
AI-31.1 owes is the reachability record, the negative citations, and a table test over the five wire values.

## Delivery (sketch — `sdd-tasks` owns the forecast)

`delivery_strategy: auto-chain` · `chain_strategy: feature-branch-chain` · budget 5000

| Slice | Node | Content |
| --- | --- | --- |
| 0 | precondition | `citations.md` C9 … C14 against the pinned commit. Blocks slices 2–4 if C9–C11 fail |
| 1 | AI-31.1 | Five-value table test, reachability + stop-sequence recorded absences (D1/D2 written down) |
| 2 | AI-31.2a | Detail-object decode → `CacheRead`, `CacheWrite`, `Reasoning`; absent-vs-zero preserved |
| 3 | AI-31.2b | AI-13.4 reconciliation on a real cache-hit transcript; the merge/no-double-count pin |
| 4 | AI-31.3 | Omitted-field and unexpected-position tolerance; `R-ATS-026`/`S-ATS-101` charter delta |

## Affected Areas

| Area | Impact | Description |
| --- | --- | --- |
| `backend/agent/src/ai/openaicompat/chunk.go` | Modified | `wireUsage` gains two nested detail structs; `usageFromWire` gains five field mappings and the cache arithmetic |
| `backend/agent/src/ai/openaicompat/stream_state.go` | Modified | Doc corrections only, unless D1 is resolved the other way |
| `backend/agent/src/ai/openaicompat/usage_test.go` + new `finish_reason_test.go` | New/Modified | Detail-field table, cache-hit transcript, merge pin, five-value mapping table |
| `backend/agent/src/ai/` | Read-only | `Usage`, `TokenCount`, `FinishReason`, `NormalizeFinishReason` consumed as-is |
| `backend/agent/src/agenttest/` | Read-only | Not modified; the `CapCompletionMetadata` obligation is recorded, not discharged |
| `openspec/changes/cachicamas-ai-provider-text-stream/specs/…` | Delta | `R-ATS-026`/`S-ATS-101` re-scoping |
| `backend/agent/go.mod` | Unchanged | Zero requires |

## Risks

| Risk | Likelihood | Mitigation |
| --- | --- | --- |
| C9/C10/C11 cannot be cited and the cache arithmetic becomes inference | **High** | Slice 0 is a precondition; an uncited inclusivity means map raw and record the uncertainty, never subtract on a guess |
| D1 is read the other way at verify, making the landed strict gate look like a regression | **High** | Both readings written into the spec with their costs; the chosen one is a stated ruling, not an omission |
| The `CapCompletionMetadata` conformance case is expected to pass against the real adapter | **High** | D3 records the wire-expressibility gap and routes it to AI-38.2 before anyone extends the bridge |
| `ai-provider-text-stream` is still an unpromoted in-flight change when the delta is written | Medium | Write the delta against the in-flight capability and reconcile at archive; the wave's recorded spec-promotion lesson applies |
| A cache-hit transcript with real `cached_tokens` cannot be obtained | Medium | A committed fixture under an explicit dialect-conventional label, named in the same place, per `R-ATS-027` |
| Landed absent-versus-zero behaviour regresses while extending it | Low | `S-ATS-055 … S-ATS-062` run unchanged on every slice; new fields are additive and absent by default |

## Rollback Plan

Every change is additive inside `openaicompat/` plus one spec delta. `git revert` per slice commit is a clean
boundary; reverting the milestone returns `usageFromWire` to its two-field form, at which point `CacheRead`,
`CacheWrite` and `Reasoning` are absent again for every transcript — the state slice 4 of AI-28 shipped and
tested. `src/ai`, `agenttest` and Waves 1–3 are untouched either way; `go.mod` is unchanged. No consumer
exists yet: AI-33 onwards have not started.

## Dependencies

- **AI-13** (`Usage`, `TokenCount`, `FinishReason`, `NormalizeFinishReason`) — shipped.
- **AI-28** — the mapper state machine, the usage chunk capture and the finish gate this change extends. AI-28.6
  is unblocked (AI-32.1 landed stage 1) and is the current chain head.
- **AI-24 § 13.1** — the usage opt-in trap; `stream_options.include_usage: true` is already emitted by
  `appendStreamFields`, so presence is discharged and this change inherits only the mapping obligation.
- Stdlib only; no ADR triggered. Strict TDD; `make test` (`go test -race -count=1 ./...`) and `make lint` clean.

## Success Criteria

- [ ] Each of the five wire `finish_reason` values maps to its neutral value in a table test, and
      `content_filter` is proven distinct from the refusal outcome.
- [ ] The neutral values unreachable on this dialect are enumerated in the spec with their basis, and no code
      path fabricates one.
- [ ] The matched-stop-value disposition is recorded as a deliberate absence with a NEGATIVE citation.
- [ ] `cached_tokens`, `cache_write_tokens` and `reasoning_tokens` reach the completion event when reported and
      stay absent when not; a reported zero stays distinguishable from absence.
- [ ] AI-13.4's cost formula is verified against a cache-hit transcript, with `Input` exclusive of the cache
      terms — or the missing citation is recorded and no subtraction is performed.
- [ ] Several populated usage frames merge without double-counting; one usage frame behaves exactly as today.
- [ ] An omitted usage field or an unexpectedly positioned metadata frame neither invents a value nor panics.
- [ ] Every new wire-shape statement resolves to a `citations.md` claim or a labelled dialect-conventional pin.
- [ ] `S-ATS-055 … S-ATS-062` still pass unmodified; `R-ATS-026`/`S-ATS-101` carry their delta.
- [ ] `make test` green, `make lint` clean, `go.mod` byte-identical.

## Proposal question round (open — answers may change scope)

1. **D1**: does a novel vendor stop value stay a typed malformed response, or become the unknown outcome? The
   first keeps a detector this wave just built; the second follows doc 0002's wording literally.
2. **D2**: is `FinishReasonRefusal` recorded as unreachable in v1, or mapped from `delta.refusal` behind a
   fixture pin — which also forces a decision about refusal text?
3. **D3**: is the recorded `CapCompletionMetadata` gap acceptable as an AI-38.2 obligation, or must AI-31 make
   all seven values reachable against real transport?
4. If C9–C11 cannot be cited, is mapping the counts raw (no subtraction) with the uncertainty recorded the
   right posture, or should the affected fields stay unmapped until citable?
5. Is `total_tokens` staying undecoded still the wanted answer now that the rest of `CompletionUsage` is mapped?
