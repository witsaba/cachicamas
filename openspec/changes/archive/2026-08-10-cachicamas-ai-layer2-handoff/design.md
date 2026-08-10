# Design — AI-40: Publish the Layer 2 readiness contract

> **Change**: `cachicamas-ai-layer2-handoff` · **Proposal**: adopted as written (D1–D7 positions binding)
> **On-disk state re-verified this session** (worktree at `b062be74`): doc 0002 items 11 (line 2428), 12 (2429), 14 (2431), 15 (2432), 16 (2433), 17 (2434), 18 (2435) all read `[ ]`; item 6 (2423) `[ ]` by design; Status line 3 reads "41 of 42".

## Technical Approach

Additive, zero-behavior change: two new test-bearing artifacts (`src/handoff/`, `src/ai/example_test.go`), one guard (matrix drift), two doc-comment growths (`ai/doc.go`, `agenttest/doc.go`), one `decision.md`, one doc-0002 reconciliation, one delta-spec fix (D6, owned by sdd-spec; canonical `ai-provider-conformance-suite/spec.md:343` "eight"→"nine" applied at archive). No production symbol beyond `package handoff`'s doc.go. No conformance-suite, adapter, or guard behavior is modified.

## Architecture Decisions

### AD-1 — `src/handoff/` shape (D2)

**Choice**: `doc.go` = `package handoff` + ~5-line comment ("the Layer-2-in-miniature consumer proof; intentionally empty — the test is the deliverable"). `handoff_test.go` = `package handoff_test`, imports exactly {`context`, `errors`, `testing`} + `src/ai` + `src/agenttest`. One driver `TestHandoff_ConsumerProof` with three `t.Run` subtests:

| Subtest | Script (agenttest) | Asserts |
|---|---|---|
| `drains a scripted stream` | `NewProvider(Script{Steps: Emit(NewTextBlockStart/Delta/End) + Emit(NewCompletion(ai.FinishReasonStop, usage))})`; request via `ai.NewText`→`ai.NewMessage`→`ai.NewRequest` | full drain via `for range`; ordered event kinds; `Requests()[0]` echoes the sent request |
| `surfaces a scripted terminal error` | last step `Emit(ai.ErrorEvent(ai.MidStreamFailure(FailureReport{...}, true)))` | error event is last; taxonomy inspected through exported accessors only — `Category()`, `Retryable()`, partial-output discriminator |
| `exits boundedly on cancellation` | `Hold(gate)` mid-script; cancel after gate reached | channel closes within a bounded `select`-with-timeout; no terminal forced through (the sanctioned loss path) |

**Zero-vendor proof**: no new guard. `import_boundary_test.go:52`'s `layer1Pattern = modulePath + "/..."` with `go list -deps -test` sweeps the new package automatically (verified by source read).
**Rejected**: `agenttest_test` (library testing itself), top-level dir outside `src/` (sibling-constraint spirit).

### AD-2 — Four examples (D3)

**Choice**: all four in `src/ai/example_test.go`, `package ai_test`. Each carries a deterministic `// Output:` block — mandatory, because an example without one only compiles and never runs under `make test`.

| Function | Demonstrates | Prints (script-derived only; no timing) |
|---|---|---|
| `ExampleNewRequest` | `NewText` → `NewMessage` → `NewRequest` | model + message count |
| `ExampleModelProvider_streaming` | drive `agenttest.NewProvider`, drain, switch on kind | ordered kind names |
| `ExampleModelProvider_toolCallReconstruction` | scripted `ToolCallStart/Delta/End`; accumulate deltas; End's arguments are authoritative | tool name + reconstructed arguments |
| `ExampleFailure_inspection` | scripted terminal `ErrorEvent` | category name + retryable bit |

**Compile proof first** (per D3): commit-0 file is package clause + imports + one trivial example; `go vet ./src/ai/` proves `ai_test`→`agenttest` is acyclic (legal: an external test package may import a package that imports `ai`). **Fallback if it fails**: split streaming + tool-call into `src/agenttest/example_test.go` (`package agenttest_test`); `doc.go`'s matrix section then cross-links both godoc surfaces.

### AD-3 — `ai/doc.go` additions (D1 + duties)

**Choice**: two GoDoc sections appended after `# The provider boundary`.

1. `# The first adapter's capability record` — nine fixed-grammar rows (grammar pinned for AD-4):
   `//	CAP-R-01 streaming text  required  satisfied — <one-line reason>`
   Header names the generator (`TestOpenRouterAdapter_FullConformance`, `run_for_test.go`) and the committed expectation (`capability_record_test.go`) — never a permanent property. `CAP-O-01`'s row carries the reopen trigger inline: struck verdict (AI-29); a generated `satisfied` is a hard stop needing an ADR (`R-OR-05`, `R-ACR-004`). Followed by the two publication duties as paragraphs: item-6 wire clause naming AI-26.6's refusal and AI-29.2's striking, with the distinction sentence **lifted verbatim from doc 0002 lines 2402/2446** — "The stream half of item 6, already closed by AI-17 (`R-ARE-009`/`R-ARE-010`), is unaffected and is **not** reopened" — and, beside it, the Layer-2-strips-reasoning duty (strip OpenRouter `reasoning_details` on the wire; recorded absence, AI-29).
2. `# The v1 surface freeze` — pointer paragraph citing the decision artifact **by change name** (`cachicamas-ai-layer2-handoff`, archived under `openspec/changes/archive/`) plus doc 0002 § AI-40 — never the pre-archive path, which moves at archive.

`agenttest/doc.go` gains one pointer sentence (examples + handoff proof).

### AD-4 — Matrix drift guard (D4)

**Choice**: new `.../openrouter/conformance/doc_matrix_guard_test.go`, `package openrouter_conformance` — the same package as `capability_record_test.go`, so the unexported `expectedOpenRouterRecord()` is a **direct call** (this is the only placement that can reach it). Mechanism mirrors `provider_signature_guard_test.go`: `runtime.Caller(0)` → `filepath.Join(dir, "..", "..", "..", "doc.go")` resolves `src/ai/doc.go`; unresolvable/unparseable fails loudly naming the path (R-AMP-015 posture), never skips. Parse: regex pinned to AD-3's row grammar (`^//\tCAP-[RO]-\d\d\b`); require **exactly nine rows** (fewer fails — no vacuous pass); compare ID, required/optional, outcome per row against `expectedOpenRouterRecord().Entries()` (already in `Capabilities()` order). Read-only file, no suite behavior touched (Q1 assumption; ratified-against fallback: citing comment, no guard). Bite recorded per repo convention: flip one published outcome, watch the guard name the row, revert.

### AD-5 — `decision.md` outline (D1, Q3)

**Choice**: follows AI-29's artifact (`backend/agent/src/ai/openaicompat/decision.md`) including the `[!IMPORTANT]` no-Go-identifier box — surface enumerated **by capability**; permitted identifiers are landed test names and vendor wire fields only.

§1 How to use (AG-03 author / doc-0003 reader / reviewer) · §2 **The frozen v1 surface** — frozen, by capability: normalized request construction; content + tool vocabulary; cache breakpoints, per-request options, mutation-free rebuild; streaming event contract + per-stream sequencing; provider interface + pre-stream order; typed failure taxonomy + partial-output discriminator; cancellation + bounded backpressure (N=0, one sanctioned loss path); pre-stream retry policy; the testing surface (scripted fake, stream kit, conformance suite); observability boundary. Marked **not frozen/experimental**: live-smoke package (internal, env-gated, unreachable), token counting (discovered-optional, absent on first adapter), the `R-CNF-020`…`026` identifier gap (tracked defect, not surface) · §3 **The eighteen-item walk** — table: item → on-disk checkbox state observed → closing node(s) per the spine (doc 0002 line 2514) → one-line evidence citation (the closing milestone's own close amendment); row 6 records struck `AI-29.2`, wire half published via AI-40.2, AI-17's stream half unaffected; row 18 cites AI-40.1/AI-40.3 · §4 **Never-cancelled posture** — caller owns the context; producer blocks until it ends; abandoning without cancelling is a contract violation; abandoned-then-cancelled tested (AI-23.5, AI-33.3); never-cancelled documented as untestable-to-termination · §5 What Layer 2 inherits (both duties + doc 0003 gate) · §6 Closing-checklist verification (AI-40.3's three items walked, AI-29 § 12 style).

### AD-6 — Doc 0002 edit plan (D5)

| Line | Edit | Evidence cited |
|---|---|---|
| 2428 (item 11) | `[ ]`→`[x]` | AI-33 close amendment (line 16) |
| 2429 (item 12) | `[ ]`→`[x]` | AI-34 close amendment (line 18) |
| 2431 (item 14) | `[ ]`→`[x]` | AI-23 (Wave 3) + AI-38.1 (line 48; PR #140, `5bc2da4e`) |
| 2432 (item 15) | `[ ]`→`[x]` | AI-38 close amendment (line 48) |
| 2433 (item 16) | `[ ]`→`[x]` | AI-36 close (line 24) + suite case AI-23.7 |
| 2434 (item 17) | `[ ]`→`[x]` | AI-39.1 (PR #142, `b062be74`) |
| 2435 (item 18) | `[ ]`→`[x]` | AI-40.1 / AI-40.3 (this change) |
| 2423 (item 6) | **unchanged `[ ]`** | by design (line 2446) |
| 3 (Status) | 41→**42 of 42**; Remaining → none (Layer 1 complete; doc 0003 AG-03 unblocked) | — |

Per-item evidence lives in the new `> Amended 2026-08-10 — AI-40 close` blockquote (AI-38/AI-39 pattern: counter, fresh `find` file counts measured at close, what-delivered, verify-gate evidence, per-record archive link, Engram topic keys), which itemizes each ticked box — checkbox lines stay bare, matching the Wave-2-close precedent.

### AD-7 — Strict-TDD sequence

| WU | RED (explicit failing state) | GREEN |
|---|---|---|
| A handoff | `handoff_test.go` first — compile failure, `package handoff` absent | add `doc.go`; subtests pass against landed agenttest behavior |
| B examples | skeleton file + `go vet` compile gate (cycle → invoke D3 fallback) | fill bodies; `// Output:` blocks are the self-asserting checks run by `make test` |
| C matrix + guard | guard lands **before** the matrix — fails "found 0 of 9 rows" | AD-3 § 1 matrix added; guard green; then one defeat-test + revert (bite recorded) |
| D duties + pointers | passive doc — compile via `make test` | — |
| E decision.md · F doc 0002 | passive docs — structural readback | — |

Gates after each WU: `cd backend/agent && make test` (`-race`), `make lint`, `make build`; `go.mod`/`go.sum` byte-identical to base.

## Data Flow

    handoff_test / examples ──drive──▶ agenttest.Provider ──▶ ai.Event stream ──▶ drain/inspect
    AD-4 guard ──runtime.Caller(0)+regex──▶ ai/doc.go matrix ──compare──▶ expectedOpenRouterRecord()

## File Changes

Proposal's affected-areas table stands unchanged; the D4 file is named `doc_matrix_guard_test.go`. No file moves (sibling constraint untouched).

## Testing Strategy

| Layer | What | Approach |
|---|---|---|
| Consumer proof | drain / scripted error / bounded cancel | AD-1 subtests, `-race`, table-free (three scenarios, distinct shapes) |
| Examples | four run under `make test` | deterministic `// Output:` blocks |
| Guard | matrix cannot drift | AD-4 nine-row parse vs committed expectation + recorded bite |
| Boundary | zero vendor imports | existing AI-00.3 guard, no change |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary is added (the only subprocess use, `go list`, is pre-existing AI-00.3 machinery, unmodified).

## Migration / Rollout

No migration. Rollback per proposal (all-additive; branch revert restores `main`).

## Open Questions

- [ ] Q1–Q4 carried from the proposal with recorded assumptions (D4 boundary, D2 package, capability-granularity, D6 blast radius) — none blocking; each has a documented fallback.
