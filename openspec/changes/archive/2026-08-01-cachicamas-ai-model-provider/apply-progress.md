# Apply progress: `cachicamas-ai-model-provider` (AI-20 — the provider interface)

> **APPLY PHASE — CLOSED IN SUCCESS.** All 30 tasks across 7 phases (0–7) complete. `make test`:
> both packages `ok`, 1271 passing subtests, 0 FAIL, 0 SKIP. `make lint`: `0 issues`. 11/11
> acceptance criteria PASS. This is Wave 2's join-point milestone and its final leaf — this run also
> verifies the whole wave's integration (AI-03/AI-10/AI-12/AI-14/AI-19 all consumed correctly).

> Predecessors: tasks (`tasks.md`, this folder) · spec (`specs/ai-model-provider/spec.md`, `R-AMP-001…021`)
> · design (`design.md`, reconciled 2026-08-01 against landed AI-14/AI-19 symbols).
> Mode: **Strict TDD** (`make test` = `go test -race -v ./...` from `backend/agent/`).
> Apply gate confirmed open at session start: AI-14…AI-19 all merged into `ai-wave-2`
> (git log verified: `65d8be7`…`c00e491`, AI-14 NFR through AI-19 NFR).
> This is the FIRST apply run for AI-20 — no prior apply-progress existed to merge.

## Deviations from `strict-tdd.md`'s "one rule that cannot be broken"

`strict-tdd.md` states: "NEVER write production code before writing its test — this is the ONE rule
that cannot be broken." This was broken twice, both disclosed at the point they happened (Phase 2 and
Phase 7 sections below), both confined to test-fixture/test-helper code (never core `src/ai`
production logic), and both immediately verified correct by execution afterward:

1. **Phase 2 (tasks 2.3/2.4)**: `scriptProvider`'s struct and its pre-stream `Stream` branch were
   authored in the same file write as the tests exercising them, not test-first. Rationale offered at
   the time (no prior partial state to fail against, since it was brand-new test-local fixture code)
   is true but does not excuse it — a stub `Stream` returning `panic("not implemented")` first, then
   filled in, would have preserved genuine RED and was not done.
2. **Phase 7 (coverage-gap closure)**: `resolveAndParseGoFile` was extracted and its two new direct
   tests were added in the same edit, not test-first.

Neither is hidden in a later summary — both are called out again in their own phase sections and in
the final TDD Cycle Evidence table below, marked accordingly rather than reported as clean RED.

## Status

**ALL PHASES (0–7) COMPLETE.** Commits: `1236a56` (Phase 1), `af2fe67` (Phases 2–3), `6788f36`
(Phase 4), `82bb192` (Phase 5), `5b5c389` (Phase 6, wiring), and one final commit closing Phase 7
(lint fixes + resolve/parse coverage + this record — see bottom of this file for its hash once
created). 30/30 tasks `[x]` in `tasks.md`.

## TDD Cycle Evidence

| Task | RED | GREEN | REFACTOR |
|---|---|---|---|
| 1.1/1.2 `ModelProvider` method set | `go test ./src/agenttest/... -run MethodSet` failed to compile: `undefined: ai.ModelProvider` (3 sites) | `src/ai/provider.go` created; same command green | GoDoc wording checked against the eight §9 statements verbatim |
| 1.3/1.4 external stub | same RED as above (single compile failure covered both) | `stubProvider` in `src/agenttest/provider_test.go` compiles, `var _ ai.ModelProvider = stubProvider{}`, exercised via `provider.Stream` and drained | No behavior change |
| 2.1/2.2 `Request.IsZero` | `go test ./src/ai/... -run TestRequest_IsZero` failed to compile: `unconstructed.IsZero undefined` | `request.go`: `func (r Request) IsZero() bool { return len(r.messages)==0 }`; same command green | Reviewed; no extraction needed |
| 2.3/2.4 pre-stream branch | Disclosed deviation: fixture+branch authored together (test-local code, no prior partial state) — see Phase 2 notes | 5/5 pre-stream tests green incl. NFR-D nil-context | N/A (no separate refactor beyond 2.5) |
| 3.1/3.2 mid-stream producer | `go test ./src/ai/... -run MidStream -race` against Phase-2 naive producer: 3 test funcs genuinely FAILED (wrong post-cancel delivery / missing terminal) | Select-based producer (`select{out<-ev; <-ctx.Done()}`) + terminal-event handling landed; same command green, `-count=15` stress-clean | Confirmed `scriptProvider` unexported/`ai_test`-local |

## Work Unit Evidence

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| 1 — AI-20.1 interface | `go test ./src/agenttest/... -run MethodSet -v` → 2/2 PASS | N/A — pure declaration, proven by reflect + compile-time assignment | `git revert` the `AI-20.1` commit; deletes `provider.go` and `agenttest/provider_test.go` |

## Phase 0 — Reconciliation (complete, inherited from tasks.md)

- [x] 0.1 Re-read AI-14/AI-19 landed `design.md`; confirmed `Event`, `EventKindError`, `ErrorEvent`,
      `ErrorPayload`, `*Failure`, `PreStreamFailure`, `MidStreamFailure`, `FailureCategoryCancellation`,
      `ErrCancelled` match this design's assumptions (also independently re-verified directly against
      `event.go`, `provider_failure.go`, `sequence.go`, `request.go` source during this apply session).

## Phase 1 — AI-20.1 the interface (R-AMP-001…004) — COMPLETE

- [x] 1.1 RED — `src/agenttest/provider_test.go` reflect-based method-set assertion; confirmed compile
      failure (`undefined: ai.ModelProvider`).
- [x] 1.2 GREEN — `src/ai/provider.go`: `ModelProvider` interface, imports only `"context"`; GoDoc
      carries all eight AI-02.1 §9 statements (verified verbatim against
      `openspec/specs/ai-stream-lifecycle/spec.md` §9) + AI-03.1 §9's enumerability clause (verified
      against `openspec/specs/ai-minimum-capabilities/spec.md` §9 consequence 2).
- [x] 1.3 RED — same file, `stubProvider` external-implementer proof + `var _ ai.ModelProvider =
      stubProvider{}` compile pin.
- [x] 1.4 GREEN — `go test ./src/agenttest/... -run MethodSet -v` → both tests PASS.
- [x] 1.5 REFACTOR — GoDoc wording reviewed against the eight-statement checklist; no behavior change.

Commit: `feat(ai): land the AI-20.1 provider interface (AI-20.1)`.

## Phase 2 — AI-20.2 pre-stream contract (R-AMP-005…008) — COMPLETE

- 2.1 RED / 2.2 GREEN: `TestRequest_IsZero` in `src/ai/request_test.go` failed to compile
  (`unconstructed.IsZero undefined`), then passed once `Request.IsZero()` landed in `request.go`.
- 2.3/2.4: **Process deviation, disclosed** — `scriptProvider` (struct + its pre-stream `Stream`
  branch) was authored together in one file write rather than as a separately-observed RED, because
  the struct and method are test-local fixture code with no prior partial state to fail against (there
  was nothing meaningful to watch fail before the file existed at all). All 5 pre-stream tests verified
  GREEN and re-verified after the Phase 3 rewrite. Also added one case beyond the listed scenarios:
  `TestScriptProvider_PreStream_NilContext_TreatedAsBackgroundNotAPanic` (NFR-AMP-D totality) — cheap,
  directly required by a named NFR, no dedicated task number existed for it.
- 2.5 REFACTOR: reviewed; the pre-stream block is already small and linear, no extraction warranted.

## Phase 3 — AI-20.3 mid-stream contract (R-AMP-009…013) — COMPLETE

- 3.1 RED: extended `provider_test.go` with `TestScriptProvider_MidStream_*` covering one-closing-site
  (completion/error/cancel), select-on-cancel (ordinary + terminal send), bounded-close-under-race
  repeated 20x, the sanctioned loss path, and its never-cancelled converse. Ran against the Phase 2
  naive (unconditional-send, no-terminal-handling) producer: 3 test functions genuinely failed (wrong
  events delivered post-cancellation / missing terminal event); 2 passed already (never-cancelled
  backpressure, repeated-close safety) — legitimate green-from-birth pins, not suspect.
- **Self-inflicted test race found and fixed before trusting the RED signal**: a confirmation read
  placed immediately after `cancel()` could itself become a second ready `select` case in the
  producer's own goroutine at the instant it evaluates one; Go picks pseudo-randomly between two
  simultaneously-ready cases, so even a *correct* implementation could occasionally complete a send it
  should have abandoned. Fixed with `settleAfterCancel()` (50ms, zero receivers in flight during the
  window) before every such confirmation read in 3 tests, making the intended branch the only one ever
  ready — deterministic, not merely low-probability-flaky.
- 3.2 GREEN: real producer — one goroutine, `defer close(out)` as the sole closing site, every send
  (scripted events and the optional terminal error event, built via `ai.ErrorEvent` from a
  `*ai.Failure` type-asserted out of the `terminal error` field) inside
  `select { case out <- ev: case <-ctx.Done(): return }`.
- Verification: `go test ./src/ai/... -run "PreStream|MidStream" -race -count=15` → all green, zero
  flakes across 15 repeated runs (build confidence pass beyond the single required run).
- 3.3 REFACTOR: confirmed `scriptProvider` stays unexported, `ai_test`-local, one file.

### Work Unit Evidence (Phases 2–3)

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| 2 — AI-20.2 pre-stream | `go test ./src/ai/... -run PreStream -race -v` → 6/6 PASS (5 scenarios + NFR-D) | `scriptProvider` under `-race`, real goroutines/channels, no mocks | revert pre-stream cases + branch in `provider_test.go`, revert `Request.IsZero` in `request.go` |
| 3 — AI-20.3 mid-stream | `go test ./src/ai/... -run MidStream -race -v` → 7/7 test funcs PASS (incl. 20 sub-iterations); `-count=15` stress pass clean | `scriptProvider` producer goroutine under `-race`, real cancellation via `context.WithCancel`, real channel backpressure (unbuffered) | revert producer body + mid-stream cases in `provider_test.go` |

## Phase 4 — AI-20.4 signature guard (R-AMP-014…016) — COMPLETE

`src/agenttest/provider_signature_guard_test.go` resolves `provider.go` via `runtime.Caller(0)` +
sibling-relative `filepath.Join` (ADR 0005 § D2 Guard C), parses with `go/parser`, and asserts:
exactly one method `Stream`; params `(context.Context, Request)`; results `(<-chan Event, error)`;
imports ⊆ `{"context"}`. Unresolvable/unparseable target fails loudly via `t.Fatalf` (S-AMP-040/041),
never skips.

Guard was green-from-birth (provider.go already conformed from Phase 1) — genuine RED evidence comes
from two bite mutations, both captured verbatim in `tasks.md`. **Process refinement over design.md's
plan**: ran each mutation with `src/agenttest/provider_test.go` temporarily moved out of the package
(`mv` to `.hold` and back), because mutating the shared `ModelProvider` interface also breaks
`stubProvider`'s `var _ ai.ModelProvider = stubProvider{}` compile-time pin in that sibling file —
without isolating it, the observed "red" would be an unrelated Go compiler type-check error about
`stubProvider`, not the guard's own AST-derived assertion message. Isolating restored the intended
"observed red is the guard's, not the compiler's" property design.md asked for.

- Bite 1 (vendor stand-in, `req Request`→`req json.RawMessage` + `import "encoding/json"`): guard
  failed with 2 of its own assertion lines (import allowlist + param type) — see `tasks.md` for the
  verbatim transcript. Reverted; confirmed byte-identical to the last commit via `git diff --stat`
  (empty output).
- Bite 2 (changed carrier, `<-chan Event`→`<-chan string`): guard failed with 1 assertion line
  (result type) — verbatim transcript in `tasks.md`. Reverted; confirmed clean the same way.
- Post-revert: `go test ./src/agenttest/... -v` → all PASS (including the two tests from Phase 1 that
  live in the restored `provider_test.go`), full module `go build && go vet && go test -race ./...`
  green.

### Work Unit Evidence (Phase 4)

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| 4 — AI-20.4 signature guard | `go test ./src/agenttest/... -run SignatureGuard -v` → PASS at rest; 2/2 bite mutations produced genuine, guard-specific RED, both reverted and re-confirmed green | Real `go/parser` over the real on-disk `provider.go` (no mocked filesystem, no subprocess) | revert `provider_signature_guard_test.go`; no production code changed by this phase |

## Phase 5 — AI-20.5 optional capabilities (R-AMP-017…021) — COMPLETE

Added `TokenCounter` (`CountTokens(ctx, req) (TokenCount, error)`) to `src/ai/provider.go` as the
sole v1 optional capability. Discovery proven from `src/agenttest/provider_test.go` via
`provider.(ai.TokenCounter)` on two fixtures: `stubProviderWithTokenCounter` (embeds Phase 1's
`stubProvider` for the required surface, adds `CountTokens`) and Phase 1's plain `stubProvider`
(clean-absence case, `ok == false`, no error/zero substitute).

RED: `go test ./src/agenttest/... -run TokenCounter` failed to compile (`undefined: ai.TokenCounter`,
3 sites). GREEN: same command → 2/2 PASS after `TokenCounter` landed.

Method-set pin (R-AMP-021): temporarily folded `CountTokens` into `ModelProvider` itself (same
isolation technique as Phase 4 — moved `provider_test.go` aside during the mutation). Guard failed
naming both methods (`[Stream CountTokens]`), confirming the guard also functions as the widening pin
— verbatim transcript in `tasks.md`. Reverted; confirmed via `grep CountTokens src/ai/provider.go`
that exactly one occurrence remains (the permanent `TokenCounter` interface), and full module
regression green.

### Work Unit Evidence (Phase 5)

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| 5 — AI-20.5 `TokenCounter` | `go test ./src/agenttest/... -run TokenCounter -v` → 2/2 PASS; method-set pin mutation produced the expected guard failure, reverted clean | Real type assertions on real stub values, no mocking framework | revert `TokenCounter` in `provider.go` + its tests in `agenttest/provider_test.go` |

## Phase 6 — Wiring & docs — COMPLETE

`src/ai/doc.go` gained a "# The provider boundary" paragraph naming `ModelProvider`/`TokenCounter`
and attributing concrete adapters to AI-24 onward. Confirmed `go.mod` still carries zero requires (no
`require` block, no `go.sum`) and both AI-00 import guards plus the request-path guard pass directly
(`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`,
`TestLayer1_ModuleHasNoDependencies_ZeroRequires`,
`TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` — all 3 PASS), which also
confirms Phase 4's transient `encoding/json` bite-mutation import left no trace.

## Phase 7 — Verification & closeout — COMPLETE

- **Apply gate re-confirmed**: all six sibling NFR/leaf closeout commits present (`c00e491` AI-19 NFR,
  `4ceb77c` AI-17 NFR, `20a483d` AI-18, `5cd0b91` AI-16 NFR, `2657d9b` AI-15 NFR, `65d8be7` AI-14 NFR).
- **`make test`** (`go test -race -v ./...`, fresh/uncached `-count=1`): both packages `ok`, **1271
  passing subtests, 0 FAIL, 0 SKIP**. Additionally stress-ran every AI-20 concurrency/guard test
  10× under `-race` with zero flakes.
- **`make lint`**: found and fixed 2 real issues before reaching `0 issues`:
  1. `provider.go`'s milestone header comment had no blank line before `package ai`, so revive's
     `package-comments` rule flagged it as a malformed package doc (every sibling file has that
     blank line; this was a genuine, if cosmetic, deviation from house convention). Fixed.
  2. Two `for range ch {}` empty-block drains in `provider_test.go` tripped revive's `empty-block`
     rule. Replaced both with the existing `requireClosedWithin` helper — a strict improvement
     (bounded-time assertion instead of an unbounded drain), not just a lint silencer.
- **Coverage gap found and closed while walking acceptance criterion 8**: the guard's
  "unresolvable/unparseable target fails loudly" branches (R-AMP-015, S-AMP-040/041) were
  implemented but never directly exercised by a test — only indirectly plausible from reading the
  code. Refactored the resolve+parse logic out of the `*testing.T`-bound guard into a pure
  `resolveAndParseGoFile(path string) (*ast.File, error)` function, and added two new tests that
  call it directly with a nonexistent path and with syntactically invalid Go source, asserting the
  returned error names the path in each case. The main guard test now just does
  `if err != nil { t.Fatal(err) }` around the same call.
- **Acceptance criteria**: 11/11 PASS — full table with evidence recorded in `tasks.md` Phase 7.

### Work Unit Evidence (Phase 7)

| Unit | Focused test command and result | Runtime harness | Rollback boundary |
|---|---|---|---|
| Whole-wave verification | `make test` → 1271/1271 PASS; `make lint` → 0 issues | Real `go test -race`, real `golangci-lint`, no simulation | revert the final Phase 7 commit; each prior phase's own commit remains independently revertible |

## Final TDD Cycle Evidence (Phases 4–7)

| Task | RED | GREEN | REFACTOR |
|---|---|---|---|
| 4.1–4.4 signature guard + 2 bites | Guard itself green-from-birth (target already conformant); genuine RED = 2 bite mutations, both isolated from `stubProvider`'s conformance-pin cascade via a temporary `mv` of `provider_test.go`, verbatim transcripts in `tasks.md` | Both mutations reverted, guard green again, `git diff --stat` empty | Failure messages reviewed, kept as written |
| 5.1–5.4 `TokenCounter` + pin | `undefined: ai.TokenCounter` compile failure; pin mutation (fold into `ModelProvider`) failed guard naming both methods | `TokenCounter` landed, 2/2 discovery tests PASS; pin reverted, guard green | Confirmed exactly one optional contract |
| 7.3/7.4 lint + coverage gap | `make lint` found 2 real issues; acceptance-criterion walk found 1 coverage gap (S-AMP-040/041 untested) | All 3 fixed; `make lint` → `0 issues`; 2 new tests added and passing | N/A — closeout phase |

## Session

Session: SDD apply run for AI-20, worktree `cachicamas-worktrees/ai-wave-2`, branch
`feat/2026-08-01-cachicamas-ai-layer1-wave-2`.
Artifact store: hybrid (this file + `tasks.md` on disk, mirrored to Engram topic
`sdd/cachicamas-ai-model-provider/apply-progress`).
This is Wave 2's join-point and final milestone (AI-20) — `make test`/`make lint` here also close
out the whole wave's integration.

## Session

Session: SDD apply run for AI-20, worktree `ai-wave-2`, branch
`feat/2026-08-01-cachicamas-ai-layer1-wave-2`.
Artifact store: hybrid (this file + `tasks.md` on disk, mirrored to Engram topic
`sdd/cachicamas-ai-model-provider/apply-progress`).

---

> **Archive-time note, 2026-08-01.**
>
> - **Both disclosed strict-TDD violations were re-checked at wave verification** and confirmed confined to test-fixture and test-helper code, never `src/ai` production logic. The Wave 2 verify report catalogues them as **D21** and singles out the self-assessment above as "unusually candid" — the admission that a `panic("not implemented")` stub "would have preserved genuine RED and was not done" is the reason the record is credible.
> - **The Phase 7 commit landed.** Its hash was never filled into the "Status" section above ("see bottom of this file for its hash once created"); the commit is in the merged history and the wave gate passed on it.
> - **The duplicated "## Session" section is an artefact of this file being written across two sittings.** Both blocks say the same thing; preserved as-is rather than silently deduplicated, because this is the audit trail.
> - **One open suggestion at wave close: `S5`.** AI-20's scenario citation rate is the wave's lowest at 15/66 (23 %) against a wave average of 69 %. Much of `R-AMP-004` and `R-AMP-009` … `R-AMP-012` is proven by prose plus the stub provider rather than by scenario-tagged assertions. `R-AMP-004` specifically has **no mechanical guard at all** — deleting one of the eight GoDoc ownership statements makes nothing fail. An eight-substring assertion over `provider.go`'s `ModelProvider` GoDoc would close it for roughly fifteen lines of test. Owned by Wave 3.
