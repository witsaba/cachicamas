# Tasks: AI-28 `cachicamas-ai-provider-text-stream` (translate the response lifecycle and text)

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | 1800–3600+ across 7 slice PRs (repo precedent runs ×2–4 over naive estimates: AI-21 forecast 900–1400, landed 2241) |
| 400-line budget risk | High milestone-wide; Medium–High for slices 1–2 (pre-declared `Split if` candidates), Low–Medium for slices 3–7 |
| Chained PRs recommended | Yes |
| Suggested split | 7 slices = 7 nodes (doc 0002: node boundary = PR-chain boundary); slice1 base, slice2→1, slice3→2, slice4→3, slice5→4, slice6→5, slice7→6 |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| 1 | Producer shell: `Stream`, cancellation, guard flips (R-ATS-001…006) | PR 1 (base=tracker) | `go test -race -count=1 -run 'TestClient_HasNoStreamingEntryPoint|TestClient_DoesNotSatisfyModelProviderAtRuntime|TestStream' ./...` | `httptest.Server` transcript replay, real loopback HTTP | Revert `stream.go` + guard-flip commit; `*Client` returns to AI-25 shape |
| 2 | Text mapping + conformance bridge (R-ATS-007…011) — BLOCKED, see gate | PR 2 → PR1 | `-run 'TestChunk|TestStreamState|TestConformanceBridge'` | `agenttest.RunConformanceFor(CapStreamingText)` over bridge's `httptest.Server` | Revert `chunk.go`+`stream_state.go`+bridge; slice 1 unaffected |
| 3 | Terminal discipline (R-ATS-012…014) | PR 3 → PR2 | `-run 'TestTerminal|TestTruncation'` | `[DONE]`-terminated fixture replay | Revert terminal-handling diff only |
| 4 | Absent-vs-zero usage (R-ATS-015…016) | PR 4 → PR3 | `-run TestUsage` | Usage-chunk fixture replay | Revert usage-mapping diff only |
| 5 | Unknown/delta-less tolerance (R-ATS-017…019) | PR 5 → PR4 | `-run 'TestTolerance|TestKeepAlive'` | Unknown-frame fixture replay | Revert tolerance diff only |
| 6 | Protocol-order violations + charter inspections (R-ATS-020…028) | PR 6 → PR5 | `-run 'TestProtocolViolation|TestCharter'` | 5-row table-driven fixture replay | Revert violation-table diff only |
| 7 | Pre-decode checks (R-ATS-023…024) — BLOCKED, see gate | PR 7 → PR6 | `-run TestPreDecode` | Non-stream-content-type + failure-status fixture replay | Revert `stream.go` pre-decode addition; carryover if AI-32.1 absent |

## Slice 1 — AI-28.1.1 Producer shell

- [x] 1.1 RED: flip `TestClient_HasNoStreamingEntryPoint`/`TestClient_DoesNotSatisfyModelProviderAtRuntime` in `provider_boundary_test.go` to post-AI-28 polarity (S-ATS-020…023, S-APC-030/031); run first — this captures the old guards failing because `Stream` doesn't exist yet.
  Evidence: flipped both assertions + doc comments (cite AI-28/R-ATS-001/R-ATS-006/S-ATS-020…021) in place, same test names ("successors"). RED confirmed via `go test -race -count=1 -v -run 'TestClient_HasNoStreamingEntryPoint|TestClient_DoesNotSatisfyModelProviderAtRuntime' ./src/ai/openaicompat/...`: both FAIL — `provider_boundary_test.go:22: *Client exposes no Stream method — AI-28 must land it (R-ATS-001, S-ATS-020)` and `provider_boundary_test.go:42: *Client does not satisfy ai.ModelProvider — AI-28 must land Stream (R-ATS-001, S-ATS-021)`. Both green after 1.4 (see 1.5).
- [x] 1.2 Correct stale AI-26 prose: `doc.go:5` and `client.go:61-62` → AI-28 (R-ATS-006).
  Evidence: both corrected in place ("Streaming behaviour arrives at AI-28 (see stream.go)…" / "…it acquires streaming behaviour at AI-28."). Disclosed hygiene beyond this task's named two files: `request.go`'s `newRequest` doc comment also said "AI-26 is the seam that will drive this method with a real request body" — corrected to AI-28 as part of 1.4's edit to that same function (the comment directly describes the signature 1.4 changes), not a separate file-wide rewrite.
- [x] 1.3 RED: write failing tests for R-ATS-001…005 (S-ATS-001…019): entry point/interface, validate→Translate→ctx.Err ordering (N-7a), minimal-transcript drain sequenced 1…N closed once, byte-exact response identity incl. S-ATS-015 empty-`id` malformed path, cancellation/select-send/single-close (AI-20.3).
  Evidence: created `stream_test.go` (11 top-level tests + 6 subtests, internal `package openaicompat`) covering S-ATS-001…002 (S-ATS-003 is agenttest's own pre-existing, unmodified `TestModelProviderInterface_SignatureGuard` — proof is its continued pass in 1.5's full-suite run, not a new test), S-ATS-004…007, S-ATS-008…011, S-ATS-012…015 (triangulated 3 ways: empty id / empty model / absent id key), S-ATS-016…018 (S-ATS-019 is `[inspection]`, discharged by grep in 1.6, not a test). RED confirmed via `go vet ./src/ai/openaicompat/...`: compile failure, `c.Stream undefined (type *Client has no field or method Stream)` at every one of the 10 call sites — reference to not-yet-existing production code per strict-tdd.md's sanctioned RED form.
- [x] 1.4 GREEN: create `stream.go` — `Stream` entry point, producer goroutine, one `close(out)` site, select-send, `ai.NewResponseStart` identity path converting constructor error to D7 malformed-response terminal (N-7b).
  Evidence: created `stream.go` (`Stream`, `run`, `applyFrame`/`buildCompletion` on a slice-1-scoped `producerState`, single `emit` helper (D6), `emitFailure` (D7), `errMalformedIdentity`/`errIncompleteStream` unexported causes wrapping `ai.ErrMalformedResponse`). Extended `request.go`'s `newRequest` with a `body []byte` parameter (before the variadic `segments`) plus conditional `Content-Type`/`Accept` headers when body is non-nil; updated its doc comment (AI-26→AI-28). Updated the three existing call sites to pass `nil`: `client_test.go` (`driveOneRequest`), `viability_test.go` (`driveOneViabilityRequest`), and `timeout_test.go` (found via `grep -rn '\.newRequest(' src/ai/openaicompat/*.go` after the first `go vet` pass caught only 2 of 3 — safety-net discipline: fixed rather than left broken). All new tests green on first GREEN attempt (see 1.5).
- [x] 1.5 Evidence: `go test -race -count=1 ./...` from `backend/agent/`, run twice; confirm 1.1/1.3 scenarios and both flipped guards green.
  Ran 4 total full-suite passes across this slice (2 before the lint fix below, 2 after) — all green, all packages: `agenttest`, `ai`, `ai/openaicompat`. `openaicompat` top-level test count: baseline 137 (pre-slice-1, captured before any edit) → 148 post-slice-1 (+11 new top-level tests; +17 incl. subtests) — all real assertions against a real `httptest.Server`-driven `Stream` (HTTP request/response, SSE decode loop, `ai.NewResponseStart`/`ai.NewCompletion` construction), none vacuous. Both flipped guards (1.1) pass. Goroutine-leak and carrier-close-discipline tests (S-ATS-005/016/018) pass under `-race`.
- [x] 1.6 `go.mod` zero-requires check + `go vet`/lint clean.
  `grep -c '^require' go.mod` = 0 (file unchanged: `module`+`go` directives only). `go vet ./...` clean. `make lint` (`golangci-lint`) initially reported 1 issue: `stream.go`'s file-header comment was directly attached to `package openaicompat` with no blank line, so revive's `package-comments` rule read it as a second, malformed package comment (doc.go already carries the canonical one). Fixed by inserting a blank line before `package openaicompat` in both `stream.go` and `stream_test.go` (matching the rest of the package's own convention, e.g. `src/ai/sequence.go`). Re-ran: `make lint` → `0 issues`.

## Slice 2 — AI-28.1.2 Text mapping + conformance bridge — BLOCKED

> GATE (R-ATS-011 STOP rule): verify `cachicamas-ai-conformance-lifecycle-amendment` has landed **both** R-CNF-023 (capability-scoped entry point) and R-CNF-024 (read-only `Script`/`Step` introspection) before 2.4. If absent: STOP, record a carryover naming the amendment as blocker — no `agenttest` edit, no fork, no substitute entry point.

- [ ] 2.1 RED: tests in `chunk_test.go` for R-ATS-007/009/010 (S-ATS-024…027, S-ATS-033…039): byte-exact `TextDelta` incl. null/absent/empty content, raw-byte split-rune dialect-conventional fixtures (content placed after split point per non-vacuity discipline), raw-string-strict finish-reason gate.
- [ ] 2.2 RED: tests in `stream_state_test.go` for R-ATS-008 (S-ATS-028…032): block mint at index 1, open-before-first-delta, close-before-Completion, terminal→sentinel window where delta-less/usage-only chunks are the clean-close shape (S-ATS-032; content-string contrast deferred to slice 6's S-ATS-074).
- [ ] 2.3 GREEN: create `chunk.go` (`RawMessage` + byte-preserving unquoter, D2) and `stream_state.go` mapper (block minting D4, finish-reason gate D5).
- [ ] 2.4 RED: bridge tests for R-ATS-011 (S-ATS-040…043) using `agenttest.RunConformanceFor(t, f, CapStreamingText)` (R-CNF-023) and `Script.Steps`/`Step.Event()`/`Step.IsHold()` (R-CNF-024), failing fast (`tb.Fatalf`) on any hold step; bridge factory declares Reasoning/TokenCounting/CacheBoundary non-nil `false` (W-2).
- [ ] 2.5 GREEN: implement bridge renderer + `httptest.Server` transcript replay, test-only, confined to `_test.go` files of `openaicompat`.
- [ ] 2.6 Evidence: `-race -count=1` twice; confirm zero files under `src/agenttest/` modified (S-ATS-042); `go.mod` zero-requires + lint.

## Slice 3 — AI-28.2 Terminal discipline

- [ ] 3.1 RED: tests for R-ATS-012…014 (S-ATS-044…054): `data: [DONE]` clean termination (dialect-conventional, fixture-pinned), close-without-sentinel → typed error with `PartialOutput()` flag (D8 block-close-before-ErrorEvent), post-sentinel/pre-EOF frames ignored.
- [ ] 3.2 GREEN: implement sentinel/truncation/post-terminal handling in `stream_state.go`/`stream.go`.
- [ ] 3.3 Evidence: `-race -count=1` twice; `go.mod` + lint.

## Slice 4 — AI-28.3 Absent-vs-zero fidelity

- [ ] 4.1 RED: tests for R-ATS-015…016 (S-ATS-055…062): absent vs reported-zero `TokenCount`, `usage:null` non-overwrite, presence positively asserted with negative controls (S-ATS-057/060 proving the assertion can fail).
- [ ] 4.2 GREEN: implement usage mapping in `stream_state.go` (D10, cites C8).
- [ ] 4.3 Evidence: `-race -count=1` twice; `go.mod` + lint.

## Slice 5 — AI-28.4 Unknown and delta-less tolerance

- [ ] 5.1 RED: tests for R-ATS-017…019 (S-ATS-063…073): unrecognised SSE event type / non-chunk `object` skip, undeclared delta fields incl. `obfuscation` (C7) tolerated, zero-delta/no-content normalization with no min-count enforcement (S-ATS-070 inspection), keep-alives inert anywhere.
- [ ] 5.2 GREEN: implement skip/tolerance paths in `chunk.go`/`stream_state.go`.
- [ ] 5.3 Evidence: `-race -count=1` twice; `go.mod` + lint.

## Slice 6 — AI-28.5 Protocol-order violations

> Depends only on 28.1.1 and is file-disjoint from slices 3–5; MAY be developed on a branch parallel to slice 2 if preferred. This plan keeps the linear chain (PR6→PR5) per the proposal's delivery table for review predictability.

- [ ] 6.1 RED: table-driven tests for R-ATS-020…022 (S-ATS-074…085): 5-row violation table incl. row-1 contrast vs S-ATS-032 (S-ATS-074), malformed-known-type vs unknown-type distinguished in one test (S-ATS-082), `recover()` no-panic probe (S-ATS-083), bound/existence-check inspection (S-ATS-085).
- [ ] 6.2 GREEN: implement violation table + D7 failure builder in `stream_state.go`.
- [ ] 6.3 RED+GREEN: cross-slice charter/citation scenarios R-ATS-025…028 (S-ATS-094…107): sentinel reuse via `errors.Is`, delivery-path handover, closed 9-member `FailureCategory` vocabulary, `go.mod` byte-identical/stdlib-only imports, no reasoning/tool-call paths, C1…C8 citation resolution (S-ATS-102…104), C6 no-error-frame path (S-ATS-105…107).
- [ ] 6.4 Evidence: `-race -count=1` twice; `go.mod` + lint.

## Slice 7 — AI-28.6 Pre-decode checks — BLOCKED on AI-32.1

> GATE: verify AI-32.1 (failure-status taxonomy) landed before 7.1. If unlanded when slices 1–6 are green: record a carryover naming AI-32.1 as blocker; do not author a substitute taxonomy.

- [ ] 7.1 RED: tests for R-ATS-023…024 (S-ATS-086…093): non-stream content type refused pre-decode with a bounded, named-constant excerpt (case/parameter-tolerant match, credential-scan guard S-ATS-089), missing content-type refused, failure statuses routed pre-decode (zero content events first).
- [ ] 7.2 GREEN: implement pre-decode checks in `stream.go`.
- [ ] 7.3 Evidence: `-race -count=1` twice; `go.mod` + lint.

## Milestone close

- [ ] 8.1 Full suite `go test -race -count=1 ./...` from `backend/agent/`, run twice.
- [ ] 8.2 Confirm 28/28 requirements and 107/107 scenarios (89 [test] / 18 [inspection]) discharged; `go.mod` unchanged; AI-00 import guards pass; AI-28.6 landed or recorded as a named carryover.
