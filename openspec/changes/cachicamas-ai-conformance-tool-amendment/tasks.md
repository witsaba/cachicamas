# Tasks: Tool-call conformance cases assert reconstruction, not delta shape

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~300–350 |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: feature-branch-chain
400-line budget risk: Low

Single PR; base = `cachicamas-ai-conformance-lifecycle-amendment` branch tip (first amendment), not `main` — per R-CNF-026 stacking.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|----------------------|-----------------|-------------------|
| 1 | Helpers + two case conversions, all gates green | PR 1 | `go test -race -count=1 ./src/agenttest/...` (from `backend/agent/`) | `RunConformance(t, FakeFactory())`, all 8 capabilities | Single-commit revert of `conformance_tool_call.go` + `conformance_lifecycle.go` (+ new test files) restores exact lists; no `src/ai` touched |

## Phase 1: Shared helpers — compile-RED → GREEN

- [ ] 1.1 RED: `conformance_lifecycle_test.go` — tests calling not-yet-existing `checkRelativeKindOrder(events, want) error`: pass, dropped-end (S-CTA-009), unexpected-kind, missing-kind, text-delta-in-window (S-CTA-006); permanent negatives (obs #2471 shape 9). Compile-RED.
- [ ] 1.2 GREEN: implement `checkRelativeKindOrder`/`requireRelativeKindOrder` in `conformance_lifecycle.go` (anchored walk, delta-tolerant window). `go test -race -count=1 ./...` GREEN.
- [ ] 1.3 RED: create `conformance_tool_call_test.go` — units for not-yet-existing `reconstructedArguments()`: end-bytes (S-CTA-014), zero-end/delta (S-CTA-015), both-channels-non-doubled (S-CTA-016), no delta-count/boundary/flag surface (S-CTA-017; obs #2471 shape 9). Compile-RED.
- [ ] 1.4 GREEN: add `reconstructedArguments() []byte` to `reconstructedCall` in `conformance_tool_call.go` (end bytes if non-empty else `fromDeltas` concat, never summed). GREEN.

## Phase 2: Convert the two cases — inverted-TDD (assertions before fixtures)

- [ ] 2.1 RED (zero-delta): `conformance_tool_call_test.go` — fake-shaped (S-CTA-001), adapter-shaped delta-carried via `Emit` (S-CTA-002), start-less (S-CTA-003), byte-mutated (S-CTA-004) calling not-yet-existing `checkZeroDeltaCaseWindow`. Zero `fake_*.go`/`stream_kit_*.go` bytes (NFR-CNF-D). Compile-RED.
- [ ] 2.2 GREEN: extract `checkZeroDeltaCaseWindow(events) error` — `checkLifecyclePrefix` + `requireRelativeKindOrder([RS,TCS,TCE,Completion])` + reconstruct block 1 == `{"q":"weather"}`, no finish-reason check (D6). Rewire `toolCallZeroDeltaCase`, drop `requireDrainedKinds`. GREEN (S-CNF-017/018/019, S-CLA-006, S-CTA-001…005).
- [ ] 2.3 RED (mixed): S-CTA-006 (delta-in-window ok, post-start text-delta fails), adapter-shaped fragmented (S-CTA-007), start-less/mismatched-ResponseID (S-CTA-008) calling not-yet-existing `checkMixedCaseWindow`. Compile-RED.
- [ ] 2.4 GREEN: extract `checkMixedCaseWindow(events) error` — `checkLifecyclePrefix` + `requireRelativeKindOrder([RS,TBS,TD,TBE,TCS,TCE,Completion])` + text-survival + reconstruct block 2 + last-event `Completion`/`FinishReasonToolCalls` retained. Rewire `mixedTextAndToolCallCase`, drop `requireDrainedKinds`. GREEN (S-CNF-020, S-CLA-007, S-CTA-006…008).
- [ ] 2.5 Permanent four-outcome guard (S-CTA-011): run each `check*Window` twice (start-bearing/start-less); assert pass/fail per outcome, failure names absent lifecycle event (obs #2471 shape 9).

## Phase 3: Register & doc comments

- [ ] 3.1 Update file-header comments in `conformance_lifecycle.go` and `conformance_tool_call.go`: `checkRelativeKindOrder`/`reconstructedArguments()`, R-CNF-019 boundary, R-CNF-021 register move (S-CTA-005/012/013, inspection).

## Phase 4: Gates

- [ ] 4.1 `go test -race -count=1 ./...` from `backend/agent/` — pass 1, all green incl. `src/ai/openaicompat`.
- [ ] 4.2 Re-run — pass 2 (determinism; cached results are not evidence, obs #2471).
- [ ] 4.3 `gofmt -l` on the 4 touched files — zero output.
- [ ] 4.4 Lint `backend/agent/src/agenttest` — zero new findings.
- [ ] 4.5 `go.mod` — zero new `require` lines.
- [ ] 4.6 NFR-CNF-D byte-check: zero `fake_*.go`/`stream_kit_*.go` diff.
- [ ] 4.7 `src/ai/openaicompat` sibling suite green, untouched.
- [ ] 4.8 First commit banks planning artifacts only via explicit `git add` paths; never `git add -A`.
- [ ] 4.9 Archive-order (R-CNF-026): archive `cachicamas-ai-conformance-lifecycle-amendment` first; archiving this one first, or against the shipped main spec, is a defect (S-CTA-018/019).
