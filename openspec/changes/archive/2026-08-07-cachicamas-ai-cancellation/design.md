# Design: AI-33 Cancellation and Goroutine Cleanup

## Identity

| Field | Value |
|---|---|
| Change / milestone | `cachicamas-ai-cancellation` / AI-33, Wave 5 Harden |
| Branch / worktree | `feat/ai-33-cancellation` / `cachicamas-worktrees/ai-33` |
| Base | `main` @ `e9a8054` |
| Inputs | `proposal.md` §§4–14; `specs/ai-stream-lifecycle/spec.md` R-AIS-033–038; `explore.md` §§2–6; charter doc 0002:1987–2037 |
| Runtime | Layered `backend/agent`; `cd backend/agent && make test` |

## Outcome

This design proves AI-33 over real HTTP transport with one surgical production change: drain the response body in `run()`'s defer chain before its existing close. Tests remain strict-TDD, reuse shipped `agenttest` helpers, and leave the conformance suite unchanged.

## Architecture overview

```mermaid
flowchart LR
  C[Client / run] --> H[httptest HTTP body]
  C --> T[a_i-33_1..5 tests]
  T --> D[DrainAndRecord]
  T --> L[RequireNoGoroutineLeak]
  T -. regression pins .-> S[conformance cancellation R-CNF-011/012 unchanged]
  C -->|defer close(out); drain; close body| H
```

## Where the drain goes

At `stream.go:344–345`, retain the unique `close(out)` defer and add one statement to the body-close defer chain: `io.Copy(io.Discard, resp.Body)` before `resp.Body.Close()` (errors ignored). This mirrors `capture.go:117–122`, preserves the single-producer model (R-ATS-003), adds no helper or dependency, and applies to completion, error, and cancellation returns.

## Per-subnode design

| Node / file | GIVEN → WHEN → THEN; fixtures and posture | Production / forecast |
|---|---|---|
| AI-33.1 / `a_i-33_1_test.go`, internal | Cancelled context before `Do`/first byte → call `Stream` against `httptest.NewServer` using `contextBeforeFirstFrameServer` (`stream_failure_test.go:391–405`) → typed pre-stream cancellation, nil channel, no spawned goroutine. Run text and tool request variants inside `RequireNoGoroutineLeak`; use `DrainAndRecord` only when a channel exists. | None; ~150 lines |
| AI-33.4 / `a_i-33_4_test.go`, internal | Real short text/tool transcript ending `[DONE]` via `bridgeServeTranscripts` (`bridge_test.go:141–158`) → drain, then cancel (including concurrent final receive) → exactly one terminal, clean close, no panic/leak. Use `DrainAndRecord` and 50-repeat leak helper. | None; ~150 lines |
| AI-33.2 / `a_i-33_2_test.go`, internal | First text/tool frame from `newDripHandler` (`timeout_test.go:20–39`), stall, cancel → close within `DefaultDrainTimeout + safety`, body released, next same-transport request succeeds, no leak. | RED first. If read loop fails to unblock, append a living-graph decision node; only then choose a `Body.Close()` watcher or ctx-aware wrapper, without a persistent second producer. ~200 lines plus any bounded fix |
| AI-33.3 / `a_i-33_3_test.go`, internal | Truly-abandoned consumer (R-CNF-012 wording verbatim), real text/tool transcript via `bridgeServeTranscripts`, no reads, cancel while send is blocked → bare close, intact landed events, no completion/error, no leak. Do not test abandoned-never-cancelled or slow-but-alive consumers. | None; ~200 lines |
| AI-33.5 / `a_i-33_5_test.go`, external | Full serial wrapper exercises completion, terminal error, pre-header, between-frame, abandoned-then-cancelled, and after-completion paths for text/tool streams → every scenario through `RequireNoGoroutineLeak`, bounded drains where readable, and unchanged `go.mod`. | Add defer drain plus wrapper; ~300 lines, split 33.5a/33.5b if >400 |

## AI-33.5 wrapper and isolation

The external-package suite mirrors `bridge_test.go`: one non-parallel test function invokes named scenario functions from 33.1–33.4 for both stream kinds through `RequireNoGoroutineLeak`. No AI-33 file calls `t.Parallel()`. The helper's `tb.Setenv` mechanically rejects parallel ancestors; independent files therefore coexist safely, while the wrapper guarantees serial full-package ordering.

## Testing and regression contract

All behavior leaves are RED-first and use `make test` (`go test -race -v ./...`) plus `make lint`. `DrainAndRecord` (`stream_kit_record.go:63`) supplies bounded closure; `RequireNoGoroutineLeak` (`stream_kit_leak.go:107`) supplies 50 repeats and amplitude tolerance. Existing `conformance_cancellation.go` (`R-CNF-011/012`) is unchanged and must remain green. No threat matrix applies: this is HTTP body/process integration, not routing, shell, subprocess, VCS, or executable classification.

## Risk register

| Risk | Mitigation |
|---|---|
| R1: AI-33.2 RED requires read-loop change | Empirical RED gate; append decision under living-graph clause and recalculate leak posture. |
| R2: AI-33.3 wording drift | Keep R-CNF-012 truly-abandoned wording verbatim. |
| R3: dependency added | Reuse stdlib-only kit; diff `backend/agent/go.mod`. |
| R4: parallel leak checks | Serial wrapper and no `t.Parallel()`. |
| R5: missing tool coverage | Require text + tool variant in every node. |
| R6: drain perturbs scripted behavior | Run full conformance suite before 33.5 merge. |

## Work units, acceptance, rollout

Independent PRs land in proposal order: 33.1 → 33.4 → 33.2 → 33.3 → 33.5. Tests stay with the behavior they prove; 33.5 may chain drain implementation (33.5a) before the full leak wrapper (33.5b). Acceptance maps one-to-one to R-AIS-033–038 and proposal §12: bounded close, exact single close, text/tool coverage, race-clean leak checks, green `make test`/`make lint`, unchanged dependencies, and separate rollback-ready PRs. No migration or rollout flag is required.

## Open questions

None. AI-33.3 scope is bound to “truly-abandoned consumer”; AI-33.2's implementation choice is intentionally deferred until its RED result.
