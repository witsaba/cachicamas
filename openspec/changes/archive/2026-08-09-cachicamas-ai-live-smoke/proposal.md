# Proposal — AI-39: Add the opt-in live smoke test

> **Change**: `cachicamas-ai-live-smoke`
> **Milestone**: AI-39 (doc 0002, lines 2345–2365) · **Wave**: 6 — Hand off
> **Artifact store**: hybrid (file + Engram) · **Explore**: [`explore.md`](./explore.md) · Engram `#2785`
> **Depends on**: AI-38 (merged, `5bc2da4e`) · **Blocks**: nothing — AI-40 may treat AI-39 as optional
> **Execution mode**: auto — every decision fork below is resolved with a recommendation marked
> *recommended, pending maintainer ratification*, never deferred back as a blocking question.

## Intent

A live smoke test already ships, but it does not yet prove what the milestone charter says it
proves. Of AI-39.1's five acceptance items, one holds and four do not:

- The sentinel sweep is fully built and unit-proven, yet **nothing calls it over the live test's
  own output** — so the "never leaks a credential or the full prompt, even on failure" claim is
  currently an untested intention.
- The package sits at `.../openrouter/smoke`, with **no `internal/` segment**, so it is importable
  by any package in the module and by any external module. The charter's word is "unreachable".
- Its stream-shape assertion is one disjunction (`hasAnyChunk`) where the charter asks for three
  invariants, and it explicitly declines the exactly-one-terminal check in a `t.Logf`.
- No setup instructions ship with it, so the one path a human is meant to follow is undocumented —
  including the part that matters most: never write the credential to a file in the repository.

Two documentation defects, both fenced to AI-39 in writing by AI-38, compound this: the canonical
`ai-openrouter-first-provider` spec still says `Status: DRAFT` with links that 404, and its
`R-OR-07` still requires a `.github/workflows/agent-openrouter-smoke.yml` file that does not
exist and never will, while an orphan change folder already carries the correct env-var-only text.

**Success**: a human with a credential can follow shipped instructions, run one bounded request,
get start / content / exactly-one-terminal asserted and nothing about model content, and know the
run cannot leak a secret even when it fails — while the compiler guarantees no application entry
point can ever reach the package, and the canonical spec finally says what the code does.

## Scope

### In scope

1. **Relocate** `openrouter/smoke/` → `openrouter/internal/smoke/`, making Go's own
   import-visibility rule the primary enforcement of item 4, plus the two lockstep updates the
   move forces (see *Affected areas*: the credential-scan allowlist and the AI-36 convergence pin).
2. **Wire the sweep into the live test** — `BuildDenyList` + `sweep.SelfTest` positive control +
   `Scan` over the live run's own captured output, on the success path *and* every failure path.
3. **Strengthen the stream-shape assertion** to three explicit invariants: a `ResponseStart` is
   present, at least one content event follows, and exactly one terminal exists. No assertion may
   read model content.
4. **Add a module-wide `go list` reachability guard**, AI-00.3 style, deny-by-default with the same
   anti-vacuity `Fatal`, scoped honestly to what it can observe in a module with zero composition
   roots today.
5. **Ship credential-safe setup instructions** in the package directory: both env vars, shell
   `export` only, the exact post-move `go test -run` invocation, the timeout and cost bound, and an
   explicit "never write the credential to a repository file".
6. **Reconcile the canonical spec** (`openspec/specs/ai-openrouter-first-provider/spec.md`): the
   four-part promotion transform on its header, and `R-OR-07` / `R-OR-08` restated over the
   env-var-only gate; then delete the orphan `openspec/changes/add-openrouter-first-provider/`.
7. **Discharge AI-38's owed reporting gap** — run the `find`-measured production/test `.go`
   file-count reconciliation at this change's archive step, with shell access this time.

### Out of scope

- **Charter out-of-scope, verbatim**: a user-facing CLI, production deployment, and any scheduled
  billing-consuming run. No `.github/workflows/` file is created; ADR 0005's no-CI posture holds.
- Anything doc 0004 owns — above all `src/cmd/cachicamas`. This change must not create a
  composition root to make item 4's dependency walk non-vacuous.
- A second vendor, a different default model, or any change to the adapter under test.
- Widening `agenttest`'s exported surface, which AI-40 is about to freeze.
- Re-litigating `R-OR-01`…`R-OR-06` or `R-OR-09`; AI-38 owns their current text.

## Capabilities

### New capabilities

- `ai-live-smoke`: the milestone contract itself — two-stage opt-in gating, the bounded single
  request, the three stream-shape invariants, sweep-over-captured-output including failure paths,
  the compiler-plus-guard unreachability proof, and the credential-safe setup obligation.

### Modified capabilities

- `ai-openrouter-first-provider`: `R-OR-07` restated as env-var-only opt-in with no CI workflow
  file; `R-OR-08` restated over a sweep that actually runs against captured live output with a
  positive control. Delta spec in this change folder; canonical file promoted at archive.

### Conditional — `sdd-spec` MUST NOT assume these

- `ai-provider-conformance-suite`: only if `sdd-design` concludes that re-anchoring S-CNF-080's
  convergence proof (fork D2) changes the scenario's text rather than only its proof location.

## Approach

Approach A from exploration — minimal in-place fix over three proven repo patterns
(`src/ai/internal/retry` placement, `import_boundary_test.go` guard shape, `agenttest/sweep`
positive control) — sequenced so the compile-breaking move lands with its own repairs.

| Step | Work | Item |
|---|---|---|
| A | `git mv` to `internal/smoke`; update the credential-scan allowlist entries and re-anchor the AI-36 convergence pin in the same commit | 4 |
| B | Reachability guard: `go list -deps -test` over the module, deny-by-default, anti-vacuity `Fatal`, plus a compile-level negative proof that a non-`openrouter` package cannot import the path | 4 |
| C | Sweep wiring: capture buffer → `SelfTest` → `Scan`, on success and on every failure path | 3 |
| D | Three explicit stream-shape invariants via `agenttest.RequireValidStream` plus local start/content assertions | 2 |
| E | Setup doc in the package directory, written against the post-move invocation | 5 |
| F | Delta specs; canonical header promotion; orphan folder deleted; file-count reconciliation at archive | reconciliation |

Step A is RED-first in an unusual sense: the move *breaks* two compiles by design, and the
repairs are the deliverable. `sdd-apply` runs `make test` before touching anything, so the
pre-existing baseline is recorded before the move.

## Decision forks — resolved

| # | Fork | Position (*recommended, pending maintainer ratification*) |
|---|---|---|
| D1 | Package placement: `openrouter/internal/smoke` vs `ai/internal/smoke` | **`openrouter/internal/smoke`.** `ai/internal/smoke` would admit every package under `src/ai/...` — including, once doc 0004 lands, anything `src/cmd/cachicamas` may legitimately import through `src/ai`. `openrouter/internal/smoke` restricts importers to the `.../openrouter/` subtree, which contains no entry point and never will, so the compiler alone discharges "unreachable from any application entry point" permanently and without a test. The `ai/internal/retry` precedent justifies the *shape*, not the *depth*: `retry` is production code shared across `openaicompat`; the smoke package is a test-only leaf with exactly one legitimate consumer tree. Neither placement preserves the current `src/agenttest` import — see D2. |
| D2 | The move breaks `src/agenttest/sweep_convergence_test.go`, the S-CNF-080 "both reach it" pin | **Re-anchor convergence to the shared core from both sides.** The smoke package's own test asserts `smoke.Scan` ≡ `sweep.Scan` over one corpus and needle; `agenttest` keeps asserting `scanTextForSentinel` ≡ `sweep.Scan`. Convergence becomes transitive through the single implementation S-CNF-080 already names, every published name is preserved, and no test is deleted. Rejected: exporting `scanTextForSentinel` from `agenttest` (widens a surface AI-40 freezes next), and abandoning the move (forfeits item 4's strongest half). |
| D3 | Item 3 capture mechanism — no repo precedent captures `*testing.T` output | **A purpose-built capture buffer with a deferred sweep, not stdout interception.** The live test routes every diagnostic it produces (request metadata, drain outcome, error renderings, terminal diagnostics) through one `*bytes.Buffer` sink; a single `defer` runs `SelfTest` then `Scan` over that buffer and fails the test on a match, so the failure path is covered by construction rather than by remembering to call the sweep in each branch. Nothing is reported to `t` until the sweep has passed; the sweep's own error names the vector only. Rejected: intercepting `os.Stdout`/`os.Stderr` (racy under `-race` and `t.Parallel`, and captures unrelated packages' output), and scanning after `t.Errorf` (too late — the bytes are already published). |
| D4 | Item 4 proof shape given zero composition roots | **Compiler boundary as the primary proof; `go list` guard as an honest regression detector.** The guard asserts that no non-`internal` package in `backend/agent` has the smoke path in its `go list -deps -test` closure, with the AI-00.3 anti-vacuity `Fatal` when the pattern resolves to zero packages. Its doc comment must state plainly what it does and does not prove today: the module has zero composition roots, so a literal walk from each root is vacuous, and the guard's value is that it fails the day an in-module importer appears — the same day the compiler would refuse an out-of-tree one. No `t.Skip`'d placeholder waiting on doc 0004: a skipped test is not a proof. |
| D5 | R-OR-07 reconciliation direction | **Adopt the orphan folder's env-var-only text as canonical, then delete the orphan folder.** It is the only version that matches the shipped gate and the repository's actual no-CI posture; the archived workflow-dispatch text is superseded by facts on disk. The adoption goes through this change's delta spec and the four-part promotion transform at archive — not an in-place canonical edit — so the audit trail records *which* change promoted the correction. The orphan folder is deleted only after its one useful fact is merged, because an unarchived change folder with no proposal or tasks is itself a defect. |
| D6 | Capability home | **New `ai-live-smoke` capability plus a delta on `ai-openrouter-first-provider`**, following AI-38's precedent (new capability for the milestone contract, delta on the vendor spec). The milestone contract outgrows R-OR-07/08: items 4 and 5 have no home there. |
| D7 | PR boundary for the reconciliation | **One PR.** The doc reconciliation is not independent of the code: item 5's setup instructions and R-OR-07's text both cite the post-move invocation path, so splitting them ships a spec that describes a path the same PR has not yet created. Slicing, if the forecast demands it, should follow the work units in *Approach*, not a code/docs seam. |

## Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/ai/openaicompat/openrouter/smoke/` → `.../internal/smoke/` | Moved | 3 files relocated; import path changes for every consumer |
| `.../internal/smoke/smoke_test.go` | Modified | Capture sink, `SelfTest`+`Scan` defer, three stream-shape invariants |
| `.../internal/smoke/` (new file) | New | Reachability guard test |
| `.../internal/smoke/` (new file) | New | Credential-safe setup instructions |
| `.../internal/smoke/sentinel_sweep_test.go` | Modified | Convergence half-pin against `sweep.Scan` (D2) |
| `backend/agent/src/agenttest/sweep_convergence_test.go` | Modified | Import removed; convergence re-anchored to `sweep.Scan` (D2) |
| `backend/agent/src/ai/openaicompat/credential_scan_test.go` | Modified | Allowlist entries at lines 84–85 re-pathed; stale comment at line 10 |
| `backend/agent/src/agenttest/sweep/sweep.go`, `.../openaicompat/stream.go` | Modified | Doc comments citing the old `openrouter/smoke/` path |
| `openspec/changes/cachicamas-ai-live-smoke/specs/` | New | `ai-live-smoke` full spec; `ai-openrouter-first-provider` delta |
| `openspec/specs/ai-openrouter-first-provider/spec.md` | Modified at archive | Four-part promotion transform; R-OR-07/08 promoted from the delta |
| `openspec/changes/add-openrouter-first-provider/` | Removed | Orphan folder deleted after its R-OR-07 text is adopted |
| `docs/architecture/milestones/0002-...md` | Modified at archive | AI-39 closing amendment, AI-33…AI-38 blockquote pattern |

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| The move breaks a consumer beyond the two found | Med | `sdd-apply` runs `make test` before and immediately after step A; `go build ./...` across `go.work` catches cross-module fallout |
| D2's re-anchored pin is judged a weaker S-CNF-080 proof | Med | `sdd-design` records the argument explicitly; if rejected, fall back to a narrow exported agenttest helper and accept the surface cost, with AI-40 notified |
| The capture sink diverts a diagnostic that a maintainer needs and the sweep then withholds it | Med | Sweep failure reports the vector and the fact that output was withheld; a clean sweep releases the full buffer to `t.Log` |
| Item 4's guard reads as overclaiming | Med | D4's doc comment states the zero-root limitation in the test source, not only in the spec |
| The live path cannot be exercised in this change (no credential at hand) | High | Item 1's skip path, the gate unit tests, the sweep, and the guard are all provable without a credential; the credentialled run is a maintainer action recorded honestly as unexecuted if it does not happen |
| 400-line review budget exceeded | High | `sdd-tasks` owns the binding forecast; indicative estimate 650–1300 authored lines including delta specs |

## Rollback

Confined to the worktree branch `feat/ai-39-live-smoke`. Reverting the branch restores the
pre-existing `openrouter/smoke/` package unchanged — it is committed on `main`, so nothing is lost
by discarding the work. Delta specs never touch `openspec/specs/**` before archive, so an
abandoned change leaves the promoted spec tree exactly as AI-38 left it, including its two known
defects. The steps are independently revertible: E and F do not depend on B, C, or D; A is the
only step with mandatory companions (the allowlist and pin repairs) and they revert with it.

## Dependencies

- AI-38 (merged `5bc2da4e`) — the conformance baseline the live smoke is compared against.
- AI-36's `agenttest/sweep` core, `SelfTest`, and S-CNF-080 — reused, not modified.
- AI-22.3's exported `agenttest.RequireValidStream` and `ai.CheckStream` — reused for the
  exactly-one-terminal invariant.
- A credential is supplied out of band by a human, exported into the shell, never written to a
  file in the repository. `make test` never depends on one.

## Success criteria — traceability to AI-39.1

| AI-39.1 item | Criterion |
|---|---|
| 1 | [ ] Without either env var, the smoke test skips with an attributable reason; `make test` is green with `-race` and makes no outbound request |
| 2 | [ ] With credentials, one request under the 60s bound; `ResponseStart` present, ≥1 content event, exactly one terminal, all three asserted separately; no assertion reads model content |
| 3 | [ ] `SelfTest` passes before any clean-sweep result is trusted; `Scan` runs over the captured buffer on the success path and on every failure path; a deliberate planted leak fails the test naming the vector only |
| 4 | [ ] The package path contains an `internal` segment under `.../openrouter/`; an out-of-tree import fails to compile; the `go list` guard denies by default, fails anti-vacuously, and documents its zero-composition-root scope |
| 5 | [ ] Setup instructions ship in the package directory, name both env vars, use shell `export` only, carry the exact post-move invocation and the cost bound, and require no credential file inside the repository |
| — | [ ] `R-OR-07` / `R-OR-08` canonical text matches the shipped env-var-only gate; the canonical spec header carries `Status: live` and `Introduced by:` with links that resolve; the orphan change folder is gone |
| — | [ ] Production/test `.go` file-count reconciliation, measured with `find` against base, is recorded in this change's archive report, discharging AI-38's owed gap |
| — | [ ] `make lint` clean; `backend/agent/go.mod` still declares zero `require` lines |
