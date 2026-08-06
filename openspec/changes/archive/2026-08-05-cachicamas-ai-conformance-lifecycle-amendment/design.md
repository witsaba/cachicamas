# Design: Conformance suite accepts the mandated response lifecycle

## Technical Approach

Every changed case already authors its own `Script`; the fixture, not the checker, decides whether a response began. Each started-response script gains `ai.NewResponseStart(id, model)` as step 0 (plus a terminal where the shipped script omitted one), and every amended assertion is re-derived from the amended script per R-CNF-019. A new pure check plus a permanent guard test (R-CNF-020) makes the lifecycle assertion's bite provable forever. Zero edits outside `backend/agent/src/agenttest/` — `conformance_suite_test.go` needs **no textual change**: all direct callers (lines 570, 576, 687, 699, 748, 878, 965, 1088, 1110, 1146) and both `RunConformance(t, FakeFactory())` drivers (124, 1355) invoke the case functions, whose scripts live inside the cases.

## Per-case derivation table (R-CNF-019 — counts read off each script)

RS=ResponseStart, TBS/TD/TBE=text start/delta/end, TCS/TCE=tool-call start/end, RRBS=redacted reasoning start, RBE=reasoning end, C=Completion, E=Error.

| Case (spec) | Script edit | Derived kind list | N | Amended assertions |
|---|---|---|---|---|
| `text/order_contiguity…` (S-CLA-001/002) | prepend `NewResponseStart`; append `NewCompletion(FinishReasonStop, Usage{})` | RS,TBS,TD,TD,TBE,C | 6 | kinds positional; delta reads move to indices 2,3; `checkLifecyclePrefix` identity equality |
| `text/empty_completion…` (S-CLA-003/004) | prepend RS | RS,C | 2 | Completion read at index 1; identity equality |
| `tool_call/zero_delta…` (S-CLA-006) | prepend RS; append `NewCompletion(FinishReasonToolCalls, Usage{})` | RS,TCS,TCE,C | 4 | kinds positional; end-event args unchanged; **no** finish-reason equality assertion (D6) |
| `tool_call/mixed_text…` (S-CLA-007) | prepend RS (terminal already shipped) | RS,TBS,TD,TBE,TCS,TCE,C | 7 | count 7; last-event finish-reason assertion unchanged (end-indexed) |
| `terminal…/normal_finish` (S-CLA-008) | prepend RS | RS,C | 2 | `rec.Len()==2`, kinds positional |
| `terminal…/mid_stream_failure` (S-CLA-009) | prepend RS; **no completion** — the error IS this path's terminal | RS,E | 2 | kinds positional; error at index 1; `pre_stream_failure` untouched (nil carrier) |
| `cancellation/bounded_close…` (S-CLA-010) | prepend RS; **no terminal** — cancelled stream closes bare (AI-20.3) | window-scoped | — | first received kind becomes `EventKindResponseStart`; post-cancel drain still 0; no count (stated in spec) |
| `reasoning…/redacted` (S-CLA-011) | prepend RS; append `NewCompletion(FinishReasonStop, Usage{})` | RS,RRBS,RBE,C | 4 | reasoning start read at index 1, redacted-bit checks unchanged; plain subtest untouched (R-CNF-021) |
| `cache_boundary/…` (S-CLA-012) | prepend RS | RS,C | 2 | count 2; Completion at index 1 |
| `finish_reason/…` ×7 subtests (S-CLA-013) | prepend RS per subtest | RS,C | 2 each | count 2; Completion at index 1; drift-guard arithmetic untouched |
| `usage/absent_vs_zero` (S-CLA-014) | prepend RS | RS,C | 2 | count 2; Completion at index 1 |

Each case supplies its own non-empty id/model literals; Tier A asserts equality against them. All 12 requirements have homes; the 7 R-CNF-021 register cases change nothing (inspection only).

## Architecture Decisions

| # | Decision | Alternative rejected | Rationale |
|---|---|---|---|
| D1 | Extract `checkLifecyclePrefix(events []ai.Event, wantResponseID, wantServedModel string) error` in new `conformance_lifecycle.go` | testing.TB-coupled helper | Pure-error idiom (`requireFailureCategoryCoverage` precedent) lets guard tests feed synthetic slices; failures name the absent response start / mismatched field (R-CNF-020) |
| D2 | Durable guard in new `conformance_lifecycle_test.go`: `TestLifecyclePrefixGuard_StartlessSlice_FailsNamingAbsentResponseStart` (input: `[NewTextBlockStart(1)]` — start-less and terminal-less) and `TestLifecyclePrefixGuard_MismatchedIdentity_FailsNamingField` (input: `[NewResponseStart("other-id","other-model")]` vs different wanted values; id and model subtests), plus one `_Passes` positive | Staged one-shot mutation | S-CLA-015/016 mandate a permanent artifact; TestSequenceGuard_*_Fails idiom (obs #2471) |
| D3 | Shared `requireDrainedKinds(t, events, want []ai.EventKind)` for exact count + positional kinds | 10 hand-rolled loops | One derivation shape per R-CNF-019; smaller diff |
| D4 | Tier B carries no identity assertion | Calling identity check everywhere | Spec scenarios S-CLA-006…014 mandate positional kinds only; the naming obligation lives on the extracted check (S-CLA-015) used by Tier A + guard |

**Risk-4 verdict (redacted-reasoning + Completion): VALID.** `CheckStream` reads "only (kind, descriptor, block index, sequence) per event: no concrete payload type, no kind special-cased by name (R-AEE-015)" (`stream_check.go:48`). RRBS opens block 1 (`BlockRoleStart`), RBE closes it, Completion (`{Role: BlockRoleNone, Cardinality: AtMostOne, Terminal: true}`, `event.go:169`) only sets `terminalSeen`; nothing follows; the unterminated-block sweep finds block 1 ended. Contiguity holds — the fake stamps 1..4.

**Risk-2 verdict: no charter duplication.** R-CNF-008 *asserts* the drained finish reason equals `FinishReasonToolCalls`; the zero-delta case merely *scripts* that terminal (real-producer plumbing) and asserts its kind positionally — no finish-reason equality assertion is added.

## File Changes

| File | Action |
|---|---|
| `backend/agent/src/agenttest/conformance_{text,tool_call,terminal,cancellation,capabilities}.go` | Modify per table |
| `backend/agent/src/agenttest/conformance_lifecycle.go` + `conformance_lifecycle_test.go` | Create (D1/D2/D3) |
| `openspec/changes/.../specs/ai-provider-conformance-suite/spec.md` | Already written |

**Neutrality proof (R-CNF-022)**: diff surface is exactly the seven agenttest files above plus openspec docs; no `src/ai`, no `openaicompat`, no `go.mod`; verified at apply/verify via `git diff --stat`; both end-to-end drivers stay green untouched.

## Testing Strategy — one PR slice, strict TDD

Single surgical slice (auto-chain; forecast well under 400 authored lines). TDD shape per amended case: RED = amend the **assertions first** and run — the current fixture fails the new count/kind/identity assertion; GREEN = edit the script. Guard helper gets ordinary RED via its two synthetic violating inputs before `checkLifecyclePrefix` exists. Behaviour-neutrality regression: `go test ./... -count=1` in `backend/agent`.

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary.

## Migration / Rollout

No migration; single-commit revert restores shipped counts.

## Open Questions

None.

> **Extension (2026-08-04).** The spec was extended to 14 requirements / 35 S-CLA scenarios (R-CNF-023, R-CNF-024 ADDED; R-CNF-022 widened). The per-case derivation table and D1–D4 plus both risk verdicts stand unchanged. This section adds the two new surfaces.

### D8 — R-CNF-023: scoped entry point `RunConformanceFor`

**New file `backend/agent/src/agenttest/conformance_scoped.go`**:

```go
func RunConformanceFor(t *testing.T, f Factory, capability Capability)
func casesFor(c Capability) []conformanceCase   // unexported filter, fresh slice, registration order
```

- No return value — a scoped run yields **no CapabilityRecord**, so it can never be presented as full-conformance evidence (S-CLA-032); its verdict is the test outcome. `RunConformance` is untouched and remains the complete unscoped gate.
- Body: fail fast via `t.Fatalf` when `!capability.registered()`, naming the value (anti-vacuous guard, obs #2471 shape 1 — an unrecognised scope must not silently run zero cases); then `runConformanceCases(t, f, casesFor(capability))` with the internal record deliberately discarded.
- **Plumbing refactor needed: none.** `runConformanceCases` is already parameterized over an explicit case list; `requireValidFactory`/`factoryDefect` (S-CLA-031), declared-absence skips, the CAP-O-02 cross-check, and panic recovery are reused byte-identical, so no existing behaviour changes. The cross-check still runs in a scoped run — a declaration/discovery contradiction is an R-CNF-002 factory defect, preserved "unchanged" as the spec requires.
- **S-CLA-029 observable — confirmed**: registered case names ARE the executed subtest names (`runOneCase` does `t.Run(c.name, …)`, `conformance_suite.go:475`). Test: `casesFor(CapStreamingText)` returns exactly the two text case names (registry-level observable) plus `RunConformanceFor(t, FakeFactory(), CapStreamingText)` green end-to-end; out-of-scope non-reporting follows from list membership.
- **S-CLA-030**: a genuinely failing subtest marks every ancestor failed (the meta-test constraint documented at `conformance_suite_test.go:622`), so the in-suite test proves the seam (`casesFor` inclusion + delegation to `runConformanceCases`, whose failure propagation existing runner tests already prove) with one scratch-verified end-to-end RED captured in tasks.md — the S-CNF-014 precedent.
- **S-CLA-031**: pure `factoryDefect` precedent — assert the exact undeclared-capability message; the scoped runner reaches it through the identical `runConformanceCases → requireValidFactory` path.
- Tests: new `conformance_scoped_test.go` (white-box, `package agenttest`).

### D9 — R-CNF-024: read-only Script introspection, no `fake_*.go` edit

**New file `backend/agent/src/agenttest/script_introspect.go`** — Go permits method declarations in any file of the package, so `fake_script.go` (and every `fake_*.go`/`stream_kit_*.go`) stays byte-identical; NFR-CNF-D holds verbatim. `Script.Steps` is already an exported field; only per-`Step` reads are missing:

```go
func (s Step) IsHold() bool              // hold/gate step discriminator
func (s Step) Event() (ai.Event, bool)   // emitted event, true — or zero Event, false for hold/gate (and zero) steps
```

- **No-mutation mechanics**: both return value copies; `ai.Event` is an opaque immutable value (unexported payload, no setters), so no pointer, slice, or backing array of internal state escapes. No setter, no rewriting constructor. Deliberately **no `Gate()` accessor** — it would expose a mutation door (release) and is not needed to reconstruct ordered steps (S-CLA-035).
- **Vendor-neutral**: imports only `ai` (already imported); names no adapter, encodes no wire format.
- Tests: new `script_introspect_test.go` in the **external** `package agenttest_test` (existing precedent: `provider_test.go`). S-CLA-033: emit-only script → introspected event list vs `DrainAndRecord` of `NewProvider(script)`, compared kind-for-kind and payload-for-payload via typed accessors — sequence excluded, since scripted events are unstamped (seq 0) and stamping is the producer's job. S-CLA-034: mixed `Emit` + `Hold(Gate)` script, introspection only, no drain (a hold would block).
- **Credential-scan guard — checked, no interaction**: `openaicompat/credential_scan_test.go` `os.ReadDir`s only its own directory and scans only files matching `^package openaicompat_test` (lines 8, 60, 67); agenttest's external tests are outside its scope.

### Widened diff surface (R-CNF-022, re-read and matched)

Diff surface is now exactly **eleven** agenttest files: the five amended `conformance_*.go`, `conformance_lifecycle.go` + `conformance_lifecycle_test.go` (R-CNF-020), `conformance_scoped.go` + `conformance_scoped_test.go` (R-CNF-023), `script_introspect.go` + `script_introspect_test.go` (R-CNF-024) — plus openspec docs. Still no `src/ai`, no `openaicompat`, no `go.mod`. Both surfaces are purely additive: no existing exported symbol or case semantics change.

### RED order extension

Two ordinary REDs, after the amended-case inverted REDs: (1) `conformance_scoped_test.go` written against the not-yet-existing `RunConformanceFor`/`casesFor` (compile RED → implement GREEN); (2) `script_introspect_test.go` against the not-yet-existing `Step.Event`/`Step.IsHold` (compile RED → implement GREEN). Still one PR slice; authored total remains under 400 lines.
