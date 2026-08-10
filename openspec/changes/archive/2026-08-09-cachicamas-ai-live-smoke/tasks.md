# Tasks — AI-39: Add the opt-in live smoke test

> All commands run from `backend/agent/` unless stated. `make test` = `go test -race -v ./...`.
> Strict TDD: every behavioral task states its RED proof before its GREEN step.

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~1300–1400 (A ~60, B ~80, C ~210, D ~195, E ~90, F/specs ~710) |
| 400-line budget risk | High |
| Chained PRs recommended | No (maintainer pre-authorized exception) |
| Suggested split | Single PR, six internal work units A–F |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| A | Move `smoke/`→`internal/smoke/` + mandatory companions | PR 1 (single, exception-ok) | `go test ./src/agenttest/... ./src/ai/openaicompat/... -race -v` | N/A — compile/regression pins only, no live call | `git revert` the one atomic commit restores old path + old pins |
| B | Three stream-shape invariants | PR 1 | `go test ./.../internal/smoke/... -run TestStreamShape -race -v` | N/A — synthetic `[]ai.Event`, no network | Revert smoke_test.go to unit A's state |
| C | Capture-sink funnel | PR 1 | `go test ./.../internal/smoke/... -race -v` (gate closed) | `RUN_LIVE_OPENROUTER_SMOKE=1 OPENROUTER_API_KEY=*** go test -run TestOpenRouterAdapter_LiveSmoke ./.../internal/smoke/ -v` (human, out of band) | Revert smoke_test.go to unit B's state |
| D | Reachability guard + fixture | PR 1 | `go test ./.../internal/smoke/... -run TestSmoke -race -v` | N/A — `go build`/`go list` only | Delete guard test + fixture file |
| E | README setup doc | PR 1 | `go test -run TestOpenRouterAdapter_LiveSmoke ./.../internal/smoke/ -v` (gate closed) | Manual: follow README end to end | Delete README.md |
| F | Delta specs commit | PR 1 | N/A (docs) | N/A | Delete `openspec/changes/.../specs/` |

## Work Unit A — Atomic move + mandatory companions

Commit message: `refactor(openrouter): relocate smoke package under internal/ and re-anchor sweep convergence pins`

- [x] A.1 Run `make test` now; record green baseline before the compile-breaking move (threat matrix: move-without-repairs mitigation).
- [x] A.2 `git mv src/ai/openaicompat/openrouter/smoke src/ai/openaicompat/openrouter/internal/smoke` (3 files). RED: `go build ./...` (repo root, go.work) now fails on `src/agenttest/sweep_convergence_test.go:16`'s old import.
- [x] A.3 Re-path `src/ai/openaicompat/credential_scan_test.go:84-85` allowlist entries and the line-10 comment to `openrouter/internal/smoke/...`. RED before fix: `go test ./src/ai/openaicompat/... -run TestCredentialScan -race -v` fails, naming the moved files as carrying the plant marker but absent from the allowlist.
- [x] A.4 Rewrite `src/agenttest/sweep_convergence_test.go`: drop the `smoke` import, compare `scanTextForSentinel` vs `sweep.Scan` directly over the same corpus/needle (D2 agenttest half); keep the test name. GREEN discharges R-LSM-007, S-LSM-025–027. Verify: `go test ./src/agenttest/... -run TestSweepConvergence_SameCorpusAndNeedle_BothConsumersAgree -race -v`.
- [x] A.5 Add the smoke-side convergence half-pin in `internal/smoke/sentinel_sweep_test.go`: `smoke.Scan` vs `sweep.Scan` over the same canary-corpus construction, comment cross-referencing A.4 (regression pin, not RED-first — design D2). Discharges S-LSM-025.
- [x] A.6 Re-path doc-comment citations: `src/agenttest/sweep/sweep.go:8,17`, `src/ai/openaicompat/stream.go:533`. Verify: `grep -rn "openrouter/smoke\"" backend/agent/src` returns nothing (excluding the new `internal/smoke` path).
- [x] A.7 GREEN whole commit: `make test` and `go build ./...` (repo root) both green.

## Work Unit B — Three stream-shape invariants (R-LSM-003)

Commit message: `feat(smoke): assert response-start, content and single-terminal as three separate checks`

- [x] B.1 RED: in `internal/smoke/smoke_test.go`, add table-driven tests over synthetic `[]ai.Event` for a not-yet-existing three-invariant helper — cases: missing start, no content between start/terminal, zero terminals (empty stream), two terminals. Compile-fail RED (repo convention). Verify: `go test ./.../internal/smoke/... -run TestStreamShape -race -v` fails to build.
- [x] B.2 GREEN: implement the helper in `smoke_test.go` — (1) ≥1 `EventKindResponseStart`; (2) ≥1 content event (`TextDelta`/`ToolCallStart`/`ToolCallDelta`); (3) exactly one terminal via `ai.CheckStream` violation-free + `report.Terminated()==true`. Each check independently failing (S-LSM-007); content-presence fails specifically (S-LSM-008); terminal-exclusivity fails on 2 terminals, empty stream fails rather than passing vacuously (S-LSM-009); no comparison to model text/tool-arg bytes/token counts anywhere in the diff (S-LSM-010).
- [x] B.3 Verify: `go test ./.../internal/smoke/... -run TestStreamShape -race -v` green.

## Work Unit C — Capture-sink funnel (R-LSM-004; R-OR-08)

Commit message: `feat(smoke): funnel live-run diagnostics through a swept capture sink`

- [x] C.1 RED: add `captureTB` tests (records, marks failed, forwards nothing to the real `t`) in `smoke_test.go`, referencing the not-yet-existing type — compile-fail RED. Verify: `go test ./.../internal/smoke/... -run TestCaptureTB -v` fails to build.
- [x] C.2 GREEN: implement `captureTB` (`struct{ testing.TB; sink *bytes.Buffer; failed bool }` overriding `Logf/Errorf/Fatalf/Helper`) in `smoke_test.go`.
- [x] C.3 RED: add sweep-gate tests — planted marker in sink fails naming only the vector and states output withheld, no credential/marker reproduced (S-LSM-013); clean sink releases full buffer to `t.Log` (S-LSM-015); positive control forced to fail fails the run rather than reporting clean (S-LSM-014) — referencing not-yet-existing `runLiveSmoke(ctx, &sink) error`. Compile-fail RED.
- [x] C.4 GREEN: implement `runLiveSmoke` funneling provider construction, request build, `Stream`, drain (`agenttest.DrainAndRecord` via `captureTB`), and B's shape checks into error returns; rewire `TestOpenRouterAdapter_LiveSmoke` per design D3: `sweep.SelfTest(deny)` before dispatch (positive control before spend), `runLiveSmoke`, `smoke.Scan(sink.Bytes(), deny)` gate, `t.Logf` release on clean sweep, `t.Fatal` after release on `runErr != nil`. Every failure path (gate-open, request, drain, invariant) is swept before publish (S-LSM-011, S-LSM-012).
- [x] C.5 Verify: `go test ./.../internal/smoke/... -race -v` (no env vars — gate-closed path) green. Runtime harness (human, out of band, when a credential is available): `RUN_LIVE_OPENROUTER_SMOKE=1 OPENROUTER_API_KEY=*** go test -run TestOpenRouterAdapter_LiveSmoke ./src/ai/openaicompat/openrouter/internal/smoke/ -v`.

## Work Unit D — Reachability guard + fixture (R-LSM-005)

Commit message: `test(smoke): add module-wide reachability guard and compile-negative fixture`

- [x] D.1 Create `internal/smoke/reachability_guard_test.go` (package `smoke_test`). Structural pin: `go list` the smoke import path, `Fatal` if unresolvable, assert path contains `/internal/` (S-LSM-016). RED proof (overlay-defeat): `go test -overlay=<overlay pointing the path constant at a wrong/renamed path> ./.../internal/smoke/... -run TestSmokePathContainsInternal` → `Fatal` fires.
- [x] D.2 Closure check, deny-by-default: `go list -test -json` over `modulePath/...` (deviation from the `-f` template sketch — see apply-progress: a `-f '{{.ImportPath}} {{join .Deps " "}}'` template cannot be split back apart unambiguously when `.ImportPath` itself contains a `go list -test`-synthesized embedded space, e.g. `pkg_test [pkg.test]`; JSON keeps the two fields distinct), normalized (`import_boundary_test.go` pattern reused for the `" ["`-suffix strip); non-internal importer must fail naming it (S-LSM-018). Anti-vacuity `Fatal` on zero enumeration or smoke-path-absent-everywhere (S-LSM-019). RED proof (overlay-defeat): overlay narrowing the pattern to zero packages → anti-vacuity `Fatal`; overlay narrowing `allowedSmokeImporterPrefix` (adapted from "add a synthetic importer file" — see apply-progress for why) → closure check bites, naming the real importer.
- [x] D.3 Create `src/ai/openaicompat/testdata/internal_import_probe/main.go` (outside `./...`, no credential-shaped literals) importing the smoke path.
- [x] D.4 GREEN: guard runs `go build` on the fixture, asserts failure with stderr containing `use of internal package` (S-LSM-017 — fails for the right reason).
- [x] D.5 Doc comment states the zero-composition-root scope in the guard's own source, no skipped placeholder (S-LSM-020).
- [x] D.6 Verify: `go test ./src/ai/openaicompat/openrouter/internal/smoke/... -run TestSmoke -race -v` green; re-run the two D.1/D.2 overlay-defeat commands and confirm they fail as expected (not committed as passing state).

## Work Unit E — Setup instructions (R-LSM-006)

Commit message: `docs(smoke): ship credential-safe setup instructions`

- [x] E.1 Create `internal/smoke/README.md`: both env-var names, shell-`export`-only credential step, exact invocation `go test -run TestOpenRouterAdapter_LiveSmoke ./src/ai/openaicompat/openrouter/internal/smoke/ -v`, 60 s bound, ~1¢/run cost, explicit "never write the credential to a repository file", zero 20+-char credential-shaped literals (S-LSM-021, S-LSM-022, S-LSM-023).
- [x] E.2 Verify (stale-path proof, S-LSM-024): run the stated invocation with the gate closed (no env vars) — resolves the package and reports SKIP.
- [x] E.3 Verify README does not trip the recursive credential scan: `go test ./src/ai/openaicompat/... -run TestCredentialScan -race -v`.

## Work Unit F — Delta specs

Commit message: `docs(openspec): add AI-39 live-smoke delta specs`

- [x] F.1 Stage `openspec/changes/cachicamas-ai-live-smoke/{proposal.md,design.md,tasks.md,specs/ai-live-smoke/spec.md,specs/ai-openrouter-first-provider/spec.md}` (already authored in prior SDD phases; no further edits here). Also stages `explore.md` (present in the change directory, not individually named in this task, but part of the same already-authored artifact set).
- [x] F.2 Verify: scenario-ID/requirement-ID prefixes (`R-LSM-`, `S-LSM-`) remain unique across `openspec/specs/` and `openspec/changes/`; delta spec touches only `R-OR-07`/`R-OR-08`.

## Work Unit H (remediation) — Reconcile deadline sharing, retry-cost and dependency-clause text with shipped behavior

Commit message: `fix(smoke): reconcile deadline sharing, retry-cost and dependency-clause text with shipped behavior`

Scoped remediation against exactly three `verify-report.md` findings (WARNING-2, WARNING-3, and the
S-LSM-028 archive-blocker section) — no other scope touched. Full finding→fix mapping and gate
evidence recorded in apply-progress (`sdd/cachicamas-ai-live-smoke/apply-progress`).

- [x] H.1 (WARNING-2, S-LSM-004): code fix, not a spec amendment — extracted `drainBoundFromContext`
      in `internal/smoke/smoke_test.go` so the stream drain's timeout derives from the request
      context's actual remaining time (`ctx.Deadline()` / `time.Until`) instead of an independent
      fresh 60 s timer starting at drain time. RED-first: `TestDrainBoundFromContext` (4 subtests)
      failed to compile before the helper existed; GREEN after. Triangulated with a scratch-only
      overlay-defeat proof (reverting the helper to ignore `ctx` fails 3 of 4 subtests). `README.md`'s
      Timeout row reworded to describe the derived, truly-shared deadline.
- [x] H.2 (WARNING-3, R-LSM-002 / S-LSM-006): text-only — reworded `specs/ai-live-smoke/spec.md`
      R-LSM-002's title/body, S-LSM-004, S-LSM-006, and acceptance criterion 2 from "exactly one
      request" to "exactly one `provider.Stream` invocation, which the adapter's ratified retry policy
      may expand to at most four HTTP attempts" (adapter untouched, per R-LSM-008). `README.md`'s cost
      and "Requests per run" rows reworded to state the ~4x best-case cost bound.
- [x] H.3 (S-LSM-028, R-LSM-008): text-only — reworded `specs/ai-live-smoke/spec.md` R-LSM-008's body,
      S-LSM-028, and acceptance criterion 8 from "zero `require` lines" (false as authored — `go.mod`
      already carries 3 pre-existing, AI-37-introduced lines) to "zero NEW `require` lines; the
      dependency set stays byte-identical to `origin/main`" (confirmed:
      `git diff --stat origin/main..HEAD -- go.mod go.sum` is empty). `go.mod` untouched. G.1's own
      wording is deliberately left unchanged here — its doc-0002-side reconciliation stays an
      archive-time task.
- [x] H.4 Gates: `make test`, `make lint`, `make build` green; `git status --porcelain` clean after
      the commit.

## Archive-time tasks

- [x] G.1 AI-38-owed file-count reconciliation: `find backend/agent/src -type f -name '*.go' ! -name '*_test.go' | wc -l` and `find backend/agent/src -type f -name '*_test.go' | wc -l`, against the pre-AI-38 baseline and the merged AI-39 state; record both counts and the delta in the archive report. Also verify zero `require` lines: `grep -c '^require' backend/agent/go.mod` (expect no matches — R-LSM-008, S-LSM-028).
- [x] G.2 Doc 0002 close amendment: append the AI-39 closing blockquote to `docs/architecture/milestones/0002-...md` (AI-33…AI-38 pattern). Apply the canonical four-part promotion transform to `openspec/specs/ai-openrouter-first-provider/spec.md` (`Status: DRAFT`→`live`, `Introduced by:` line, re-resolved relative links, added `## Status` section); delete orphan `openspec/changes/add-openrouter-first-provider/`. Verify: promoted file's relative links resolve; `ls openspec/changes/add-openrouter-first-provider` reports not found.
