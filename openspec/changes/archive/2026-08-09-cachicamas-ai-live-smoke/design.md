# Design — AI-39: Add the opt-in live smoke test

> **Change**: `cachicamas-ai-live-smoke` · **Proposal**: [`proposal.md`](./proposal.md) (Engram `#2786`)
> Paths below are relative to `backend/agent/` unless prefixed `openspec/`.

## Technical Approach

Approach A from the proposal, with one refinement: every live-path diagnostic becomes an
**error-returning value routed through one capture funnel**, so the sweep covers failure paths by
construction. The move to `openrouter/internal/smoke` lands with its two lockstep repairs in one
commit; guards follow the AI-00.3 pattern verbatim.

## Verified code facts (line evidence)

| Fact | Evidence |
|---|---|
| Assertion today: `len(events)==0` Fatal + `hasAnyChunk` disjunction; exactly-one-terminal explicitly declined | `smoke_test.go:339-360` (`t.Logf("... not an assertion")` at 360) |
| `decideLiveSmoke`: key first, run-flag literal `"1"`, `Key` never populated even on proceed | `smoke_test.go:132-151, 258-260` |
| `smoke.Scan` delegates to `sweep.Scan`; error names vector only | `sentinel_sweep.go:143-149` |
| `sweep.SelfTest` fails on an entry that cannot bite (incl. empty needle) | `sweep/sweep.go:85-93`; `Scan` skips empty needles at 61-63 |
| Convergence pin imports smoke, compares `scanTextForSentinel` vs `smoke.Scan` | `sweep_convergence_test.go:16, 29-32, 48-50` |
| `scanTextForSentinel` already delegates to `sweep.Scan` | `conformance_redaction.go:108-111` |
| S-CNF-080 asks "both reach that one implementation" + published names kept | `openspec/specs/ai-provider-conformance-suite/spec.md:431` |
| Allowlist path-keyed at `openrouter/smoke/...`; stale path comment | `credential_scan_test.go:84-85, 10` |
| AI-00.3 guard: deny-by-default + anti-vacuity `Fatal` | `import_boundary_test.go:134-164` (Fatal at 141-144) |
| `ai.CheckStream` enforces nothing-after-terminal; `StreamReport.Terminated()` reports presence | `stream_check.go:31, 85-117` |
| `CheckContiguity` exported, error-returning; `RequireValidStream` is a `Fatalf` wrapper | `stream_kit_ordering.go:21-33, 47` |
| `DrainAndRecord(tb testing.TB, ...)` returns after `Fatalf` | `stream_kit_record.go:63-73` |
| `internal/` precedent | `src/ai/internal/retry/{doc,retry,retry_test}.go` |
| Old-path doc citations to re-path | `sweep/sweep.go:8,17`; `openaicompat/stream.go:533`; `credential_scan_test.go:10` |

## Architecture Decisions

### D-placement — `openrouter/internal/smoke` (D1 confirmed)

**Choice**: `src/ai/openaicompat/openrouter/internal/smoke/`.
**Rejected**: `ai/internal/smoke` (admits all of `src/ai/...`, incl. future doc-0004 reachable code); no move (forfeits the compiler half of item 4).
**Rationale**: importers restricted to the `.../openrouter/` subtree, which contains no entry point; the compiler alone discharges "unreachable from any application entry point".

### D2 — Re-anchor S-CNF-080 to `sweep.Scan` from both sides (confirmed, not weaker)

**Choice**: two half-pins, each a **direct same-corpus, two-sided comparison** against `sweep.Scan`, each keeping the positive-control and clean-corpus subtests:
- agenttest half: rewrite `sweep_convergence_test.go` — drop the smoke import; compare `scanTextForSentinel` vs `sweep.Scan` over the identical corpus/needle. Test name kept.
- smoke half: new test in `sentinel_sweep_test.go` comparing `smoke.Scan` vs `sweep.Scan` (importable — `agenttest/sweep` is not internal) over the same canary-corpus construction, with a comment naming its counterpart.

**Why not weaker**: the current pin proves A≡B over two corpora; the re-anchored pins prove A≡core and B≡core over the same corpus construction — transitively A≡B with the same corpus coverage, plus a divergence now names *which* consumer left the core. S-CNF-080's "previously published names" (the sweep API names) are all preserved; **scenario text is unchanged, so the conditional `ai-provider-conformance-suite` delta is NOT engaged** — `sdd-spec` must not touch it.
**Rejected**: exporting a new agenttest symbol (widens the AI-40 freeze surface); keeping the direct pin (blocks the move).

### D3 — Capture funnel: buffer sink + error returns + local capturing TB

**Choice**: restructure the live path so diagnostics cannot reach `t` before the sweep:

```go
var sink bytes.Buffer
key := os.Getenv(liveSmokeEnvVarName)
deny := smoke.BuildDenyList(liveSmokeEnvVarName, key[:min(4, len(key))], liveSmokePromptMarker)
if err := sweep.SelfTest(deny); err != nil { t.Fatal(err) } // positive control BEFORE spending money
runErr := runLiveSmoke(ctx, &sink)          // every failure path returns error; rendering goes into sink
if runErr != nil { fmt.Fprintf(&sink, "live smoke failed: %v\n", runErr) }
if scanErr := smoke.Scan(sink.Bytes(), deny); scanErr != nil {
    t.Fatalf("%v — captured output withheld", scanErr) // vector name only
}
t.Logf("%s", sink.String())                 // clean sweep releases the buffer
if runErr != nil { t.Fatal("live smoke failed; swept diagnostics above") }
```

- `runLiveSmoke` funnels provider construction, request build, `Stream`, drain, and shape checks into error returns. Shape checks use the **error-returning forms** — `ai.CheckStream` + `report.Terminated()` + `agenttest.CheckContiguity` — not `RequireValidStream`, whose `Fatalf` would publish pre-sweep (justified deviation from proposal step D; same underlying checks, one delegation point).
- TB-taking helpers (`DrainAndRecord`) receive a local `captureTB` (`struct { testing.TB; sink *bytes.Buffer; failed bool }` overriding `Logf/Errorf/Fatalf/Helper`); verified safe: `DrainAndRecord` returns after `Fatalf`.
- Three invariants, separately asserted, none reading model content: (1) ≥1 `EventKindResponseStart`; (2) ≥1 content event (`TextDelta`/`ToolCallStart`/`ToolCallDelta`); (3) exactly one terminal = `CheckStream` violation-free (at-most-one + placement) AND `Terminated() == true`.
- Key shorter than 4 bytes: prefix is the whole key — never an empty needle (`SelfTest` would correctly fail one).

**Honesty — NOT covered** (recorded in the test doc comment and README): runtime panics (stack printed directly to stderr); the test runner's own banner lines; `t.Skip` reason on the gate-closed path (deliberate — carries the env-var name as an operator hint, no live run occurred).
**Rejected**: stdout/stderr interception (racy under `-race`/`t.Parallel`, captures foreign output); scanning after `t.Errorf` (bytes already published); subprocess self-exec (covers panics but propagates the credential into a child env, brittle binary/`-run` discovery, poor unit-testability).

### D4 — Guard shape (confirmed, three pieces)

New `internal/smoke/reachability_guard_test.go` (package `smoke_test`) + fixture:
1. **Structural pin**: `go list` the smoke import path — `Fatal` if unresolvable (rename protection), assert the path contains `/internal/`.
2. **Closure check, deny-by-default**: `go list -test -f '{{.ImportPath}} {{join .Deps " "}}'` over `modulePath/...`, normalized per `normalizeListedPackage`; any package whose transitive deps contain the smoke path must sit under `.../openaicompat/openrouter/`. Anti-vacuity `Fatal`s: empty enumeration; smoke path absent everywhere. Doc comment states the zero-composition-root scope plainly (no `t.Skip` placeholder).
3. **Compile-negative fixture**: `src/ai/openaicompat/testdata/internal_import_probe/main.go` (outside the subtree, ignored by `./...`, no credential-shaped bytes) importing the smoke path; the guard runs `go build` on it and asserts failure **with stderr containing `use of internal package`** — failure for the right reason. Precedent: `src/ai/testdata/{handrolled,constructed}` compile fixtures.

### Gate, bound, and doc (confirmed defaults)

`decideLiveSmoke` semantics preserved byte-for-byte (no reason to change). Request shape and 60 s bound unchanged. Setup doc: `internal/smoke/README.md` — quick path (two `export`s + exact post-move `go test -run TestOpenRouterAdapter_LiveSmoke ./src/ai/openaicompat/openrouter/internal/smoke/ -v`), cost/timeout table (~1¢, 60 s), what is asserted / not asserted, sweep-withholds-output-on-leak note, and a callout: **never write the credential to a repository file**. No 20+-char credential-shaped literals in the doc.

## File Changes

| File | Action |
|---|---|
| `.../openrouter/smoke/{3 files}` → `.../openrouter/internal/smoke/` | Move (`git mv`) |
| `internal/smoke/smoke_test.go` | Modify — funnel, `captureTB`, three invariants |
| `internal/smoke/sentinel_sweep_test.go` | Modify — smoke-side convergence half-pin |
| `internal/smoke/reachability_guard_test.go` | Create — D4 pieces 1-3 |
| `internal/smoke/README.md` | Create — setup doc |
| `src/ai/openaicompat/testdata/internal_import_probe/main.go` | Create — compile-negative fixture |
| `src/agenttest/sweep_convergence_test.go` | Modify — re-anchor to `sweep.Scan`, drop smoke import |
| `src/ai/openaicompat/credential_scan_test.go` | Modify — allowlist lines 84-85 re-pathed; line 10 comment |
| `src/agenttest/sweep/sweep.go:8,17`, `src/ai/openaicompat/stream.go:533` | Modify — path citations |
| `openspec/changes/.../specs/` | Create — `ai-live-smoke` spec; `ai-openrouter-first-provider` delta |
| `openspec/specs/ai-openrouter-first-provider/spec.md`; orphan `openspec/changes/add-openrouter-first-provider/` | At archive — promote; delete |

## Testing Strategy (Strict TDD; `make test` = `go test -race -v ./...` from `backend/agent/`)

| AI-39.1 item | RED → GREEN |
|---|---|
| 2 | Unit tests over synthetic `[]ai.Event` for the shape helper (missing start / no content / zero terminals / event-after-terminal) reference the absent helper — compile-fail RED (repo convention) → implement |
| 3 | Unit tests for `captureTB` (records, marks failed, forwards nothing) and the sweep gate (planted marker → withheld + vector-only; clean → released; broken deny entry → `SelfTest` propagates) → implement, then rewire the live body |
| 4 | Guard RED proven by overlay-defeat (`go test -overlay` pointing the guard at a wrong path → anti-vacuity `Fatal`; synthetic importer → closure check bites) per repo discipline → land guard |
| 1, 5 | Gate tests already green (regression pins); README verified by running its exact invocation without env vars → SKIP |
| D2 | Both half-pins are regression pins over existing behavior (documented, as the original pin's header did) |

## Migration / Commit Ordering (every commit green)

1. **A**: `git mv` + allowlist re-path + both convergence half-pins + doc-comment citations — one commit (mandatory companions).
2. **B**: item 2 shape helper (RED→GREEN). 3. **C**: item 3 funnel (RED→GREEN), consumes B. 4. **D**: item 4 guards + fixture. 5. **E**: README. 6. **F**: delta specs. Archive: promotion transform, orphan deletion, doc-0002 amendment, AI-38's `find`-measured file-count reconciliation.

## Threat Matrix

Skill matrix: all five rows **N/A** — no routing, VCS/PR automation, executable-doc classification, or commit/push state change; the only subprocesses are read-only Go-toolchain invocations (`go list`, `go build` on an in-repo fixture), the pattern `import_boundary_test.go` already ships.

Change-specific:

| Threat | Breaks | Mitigation |
|---|---|---|
| Move without both repairs | compile (`sweep_convergence_test.go:16`) or credential scan (allowlist 84-85) | Commit A is atomic; `make test` before/after |
| Re-anchored pin drifts corpora apart | S-CNF-080 transitivity thins | Same canary construction both halves; cross-referencing comments |
| New exported agenttest symbol | AI-40 surface freeze | D2 exports nothing; guard lives in the smoke package |
| Guard passes vacuously after a future rename | item 4 silently unproven | Structural pin `Fatal`s when the path stops resolving; enumeration-empty `Fatal` |
| Fixture fails to compile for the wrong reason | false unreachability proof | Assert `use of internal package` in stderr |
| Sink withholds diagnostics a maintainer needs | debuggability | Sweep failure states output was withheld; clean sweep releases the full buffer |
| Empty/short key → empty needle | `SelfTest` hard-fails a live run | `key[:min(4,len(key))]`; SelfTest runs before dispatch (no spend on a broken sweep) |

## Open Questions

- [ ] Credentialled live run availability during apply — all other items are provable without one; if unexercised, verify records it honestly (proposal risk accepted).
