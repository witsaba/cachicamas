# Tasks: AI-25 — Provider configuration and client construction

Package: `backend/agent/src/ai/openaicompat/`. Runner: `make test` (`go test -race -v ./...`) from `backend/agent/`. Strict TDD active. Chain: tracker `feat/2026-08-03-cachicamas-ai-layer1-wave-4` ← **A** ← **B** ← **C** (linear feature-branch-chain; B/C are graph-parallel but stack for clean diffs).

## Review Workload Forecast

| Field | Value |
|---|---|
| Estimated changed lines (naive) | Slice A ~950–1,050 · Slice B ~350 · Slice C ~100–120 · Total ~1,400–1,520 |
| Estimated changed lines (corrected, 2–4× repo-confirmed undershoot) | Slice A ~1,900–4,200 · Slice B ~700–1,400 · Slice C ~200–480 · Total ~2,700–5,400 (design's own reconfirmed range) |
| 5,000-line budget risk | **High** — corrected total straddles the 5,000-line session budget; Slice A alone (now carrying `request.go`, `provider_boundary_test.go`, and the Makefile fix moved in from Slice C / added per coordinator ruling) is the single slice most likely to approach the budget on its own, though not certain to exceed it standalone |
| Chained PRs recommended | Yes |
| Suggested split | PR A (AI-25.1, injected construction) → PR B (AI-25.2, guard) → PR C (AI-25.3, viability) |
| Delivery strategy | auto-chain |
| Chain strategy | feature-branch-chain |

Decision needed before apply: No
Chained PRs recommended: Yes
Chain strategy: feature-branch-chain
400-line budget risk: High

**Pre-declared fission trigger** (design, authoritative): if Slice A alone risks breaching budget, split at its own seam — **A1** = items 1–4 (doc, client, credential, request.go + their tests) targets the tracker; **A2** = item 5 (`endpoint.go` + join table) targets A1's branch; **B** then retargets to A2's branch. `newRequest` may use bare `JoinPath` in A1 until A2 lands the empty-segment filter.

**Proposal-vs-design lag note**: `proposal.md`'s slice table (naive 840/350/150) is stale — it placed credential-attachment code in Slice C. Design moved `request.go` to Slice A (S-APC-003's stub-transport observation needs it there) and added the Makefile fix to Slice A. The figures above reflect design's authoritative placement.

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|---|---|---|---|---|---|
| A | Injected construction: types, validation, defaults, timeout absence, bounds, path-join, no-false-satisfaction, credential opacity, totality, comment fixes | PR A (base: tracker) | `cd backend/agent && go test -race -run 'TestNew|TestCredential|TestEndpoint|TestTimeout|TestProviderBoundary' ./src/ai/openaicompat/...` | `make test` full suite green from `backend/agent/` | delete `src/ai/openaicompat/`; revert `ai/doc.go`, `ai/import_boundary_test.go`, `Makefile` comment edits — independent of B/C |
| B | No-ambient-authority guard + 4 recorded bite proofs | PR B (base: PR A branch) | `cd backend/agent && go test -race -run TestAmbientAuthority ./src/ai/openaicompat/...` | N/A — static call-site scan, no network/process harness beyond `make test` | delete `ambient_authority_test.go`; guard is additive, no production-code dependency |
| C | Test-server viability probe, proxy-immune | PR C (base: PR B branch) | `cd backend/agent && go test -race -run TestViability ./src/ai/openaicompat/...` | `httptest.NewServer` probe itself is the runtime harness (real HTTP round trip) | delete `viability_test.go` |

## Evidence Classes (do not conflate)

1. **TDD RED** — every runnable scenario, ordinary red→green→refactor, recorded per task below.
2. **Guard bite proofs (4)** — the guard is green-from-birth; its staged violations ARE its RED phase. Tasks B.4–B.7.
3. **Mutation proofs (5)** — break a finished, GREEN implementation to prove the finished test catches it; distinct from TDD RED. Tasks A.4.3, A.4.4, A.4.5, A.5.3, A.6.3.

`*(review)*`-marked scenarios have no red phase; discharged by a recorded confirmation naming what was read, tracked separately below.

---

## Phase A.0 — Foundation (Slice A / fission bucket A1)

- [x] A.0.1 Create `ai/openaicompat/doc.go`: state (a) no whole-request cap + why, (b) injection-only credential/no ambient authority, (c) proxy resolution deliberately unset + why, (d) guard-scope boundary at an injected client. *Review*: discharges `S-APC-065, S-APC-066, S-APC-067, S-APC-068` (NFR-APC-D). No red phase.
- [x] A.0.2 Create `client.go` skeleton: `Config`, `Client` types; unexported constants `defaultDialTimeout=10s`, `defaultTLSHandshakeTimeout=10s`, `defaultResponseHeaderTimeout=60s`, `defaultIdleConnTimeout=90s`.
- [x] A.0.3 Create `credential.go`: unexported `Credential.token`; unexported `bearer()`. **Deviation**: `String()`/`GoString()` were deliberately deferred to A.8 rather than built here, so A.8.1 is a genuine pre-implementation RED instead of a pin-style confirmation — see A.8 note below.

## Phase A.1 — R-APC-001 injected value is used (fission bucket A1)

- [x] A.1.1 RED `client_test.go`: stub `RoundTripper`, drive one outbound request, assert scheme/host/path from injected endpoint, credential in headers, no cross-adapter leakage between two differently-stubbed clients. Covers `S-APC-001…004`.
- [x] A.1.2 GREEN: implement `request.go` `newRequest` (join + Bearer attach) and wire `New`/`httpClient.Do` so the stub observes traffic.
- [x] A.1.3 REFACTOR: shared `stubTransport` type and `driveOneRequest` helper extracted in `client_test.go`, reused by every later construction test in this slice (A.2, A.3, A.9).

## Phase A.2 — R-APC-002 typed construction faults (fission bucket A1)

- [x] A.2.1 RED: table test — malformed endpoints (empty/whitespace/no-scheme/unsupported-scheme/unparseable/control-char) and empty credential → `errors.Is`/`errors.As` position; zero round trips via stub; sentinel-set unchanged; no usable adapter on failure. Covers `S-APC-006…010`.
- [x] A.2.2 GREEN: implement validation in `New` using `ai.Invalid(ai.ErrMalformed/ai.ErrEmpty, ai.At(...))`. Zero new sentinels. **Decision recorded**: an empty endpoint is grouped under the malformed-endpoint bucket (`ErrMalformed`, not `ErrEmpty`) per S-APC-006's literal table membership and per R-APC-002's text, which names only "a malformed endpoint" and "an empty credential" as the two fault shapes — design's architecture-decision table row is looser than the scenario text and was not treated as overriding it.

## Phase A.3 — R-APC-003 / R-APC-009 safe defaults, no mutation, no proxy vector (fission bucket A1)

- [x] A.3.1 RED: no-default-endpoint failure; `http.DefaultClient`/`DefaultTransport` snapshot unchanged on both paths; shared-client-reuse-across-two-adapters unaffected; construction surface enumerates only endpoint/credential/optional client; adapter-built client identity ≠ default client/transport; adapter-built transport `Proxy == nil`. Covers `S-APC-011…013, S-APC-015, S-APC-016, S-APC-036`.
- [x] A.3.2 GREEN: nil-client branch builds a fresh bounded transport; injected client used verbatim, never mutated; adapter-built transport leaves `Proxy` unset.
- [x] A.3.3 **Mutation proof #1** (`S-APC-005`): staged "store injected client without using it" (`New` always called `newDefaultHTTPClient()`, ignoring `cfg.HTTPClient`); recorded red `make test` output — 3 tests failed with a fast, safe DNS failure against the `.invalid`-TLD test hosts (no hang, no real network reach); reverted; re-ran green.
- [x] A.3.4 **Mutation proof #2** (`S-APC-014`): staged "assign a bound onto `http.DefaultTransport`" (`ResponseHeaderTimeout = 60s`, chosen because it is zero-valued on the real default and therefore a genuine observable change — `IdleConnTimeout` was tried first and found already 90s on `http.DefaultTransport`, a no-op); recorded red against the snapshot test; reverted; re-ran green.
- [x] A.3.5 **Mutation proof #5** (`S-APC-037`, formally under `R-APC-009`/AI-25.2 in the spec but exercised here per design's testing-strategy table): staged "route the adapter-built client through `http.DefaultTransport`"; recorded red against **both** `S-APC-016` (fresh-identity) and `S-APC-036` (nil-proxy); reverted; re-ran green.

## Phase A.4 — R-APC-004 no whole-request cap, behavioral proof (SERIAL — no `t.Parallel()`)

- [x] A.4.1 RED `timeout_test.go`: drip handler (k=5 chunks, delay d=200ms); control `http.Client{Timeout:2d}` must die mid-read, asserting shape only (`net.Error.Timeout()==true`, <k chunks — never exact duration); adapter-built client reads all k chunks to EOF; caller context with no deadline imposes none internally. Covers `S-APC-017…019`. **No `t.Parallel()`** — this pair must stay serial; do not let a later change add it. **Note**: this test passed immediately on first run (no red) because A.3's `newDefaultHTTPClient` already left `Client.Timeout` at zero and `newRequest` never wrapped the context — the genuine falsifiability proof for this requirement is A.4.3's mutation proof.
- [x] A.4.2 GREEN: adapter-built client leaves `Client.Timeout` zero; passthrough of caller context with no internal `context.WithTimeout`. Already satisfied by A.3's implementation; no further code change needed.
- [x] A.4.3 **Mutation proof #3** (`S-APC-020`): staged internal `context.WithTimeout(ctx, 300ms)` in `newRequest`; recorded red — the adapter leg failed with "context deadline exceeded" after ~300ms instead of reading all 5 chunks; reverted; re-ran green.

## Phase A.5 — R-APC-005 connect/idle bounds exist (structural)

- [x] A.5.1 RED: assert `DialContext` non-nil (from `defaultDialTimeout>0`); `TLSHandshakeTimeout`/`ResponseHeaderTimeout`/`IdleConnTimeout` equal the named constants (10s/10s/60s/90s). Covers `S-APC-021, S-APC-022`. **Note**: passed immediately (already wired in A.3); genuine falsifiability proof is A.5.3.
- [x] A.5.2 GREEN: four bound constants already wired into the adapter-built transport by A.3.2; confirmed, no further change needed.
- [x] A.5.3 **Mutation proof #4** (`S-APC-023`): staged "leave connect and time-to-headers bounds unset" (dropped `DialContext` and `ResponseHeaderTimeout` from the built transport); recorded red on both assertions; reverted; re-ran green.

## Phase A.6 — R-APC-006 endpoint join (fission bucket A2)

- [x] A.6.1 RED `endpoint_test.go`: trailing-slash base, sub-path base **without** trailing slash (the RFC 3986 footgun), doubled interior separators, empty relative path, query-carrying base, absolute-URL-shaped relative segment, two-requests-non-destructive-on-stored-base. Covers `S-APC-024…029`.
- [x] A.6.2 GREEN: `endpoint.go` join helper — filter empty segments, then `(*url.URL).JoinPath`; wired into `newRequest` (refactored off the bare `JoinPath` call used since A.1, as pre-authorized by this file's own fission note).

## Phase A.7 — R-APC-007 no stubbed streaming, no false interface satisfaction

- [x] A.7.1 RED `provider_boundary_test.go`: no streaming entry point exists; **run-time** `_, ok := any(&Client{}).(ai.ModelProvider)` expects `ok==false` (deliberately not the compile-time `var _` idiom); AI-20 signature guard runs unmodified and passes. Covers `S-APC-030…032`. **Note**: passed immediately — `*Client` never had a `Stream` method, so this is a negative-property pin rather than an ordinary red→green pair.
- [x] A.7.2 GREEN: confirmed `*Client` exposes no `Stream`-shaped method (`reflect` method-set check); confirmed the existing AI-20 guard in `src/agenttest` is unaffected (re-ran, still green).

## Phase A.8 — R-APC-014 credential opacity (type-shape only)

- [x] A.8.1 RED `credential_test.go`: `%v/%s/%+v/%#v` and `json.Marshal` output never contain the raw token. Covers `S-APC-053`. **Genuinely red**: `String()`/`GoString()` were deferred from A.0.3 (see that task's note) specifically so this RED was real — all four format verbs leaked the token before this task's GREEN (json.Marshal was already safe, since `token` has no exported counterpart).
- [x] A.8.2 GREEN: implemented `Credential.String()`/`GoString()` returning `<redacted>`; no Marshal methods added (`encoding/json` already omits the unexported field).
- [x] A.8.3 *Review*: confirmed the spec (`specs/ai-provider-client/spec.md`, R-APC-014 section) states attachment-only scope naming AI-32.5/AI-36.1 as wire-level owners. Covers `S-APC-054, S-APC-055`. No red phase. **Partial**: the viability probe's own documentation does not exist yet (Slice C, out of scope for this apply batch) — its half of this confirmation is deferred to Slice C's C.2.1.

## Phase A.9 — NFR-APC-F totality / nil-client contract

- [x] A.9.1 RED: table of extreme inputs (empty endpoint, empty credential, whitespace-only endpoint, both empty) through `New` — none panics (deferred `recover()` per case), each returns a typed failure naming the offending position; valid endpoint+credential with **no** HTTP client supplied succeeds, returns a usable adapter, no error. Covers `S-APC-072, S-APC-073`. **Note**: passed immediately — no gap found.
- [x] A.9.2 GREEN: totality already satisfied by A.2.2's validation and A.3.2's nil-client branch; no gap found, no further change needed.
- [x] A.9.3 *Review*: confirmed `doc.go`'s "An absent HTTP client is not a fault" section states an absent client as selecting the adapter's own bounded client, not an omission or a fault. Covers `S-APC-074`. No red phase.

## Phase A.10 — Comment corrections and one scope lag (coordinator-ruled in scope)

- [x] A.10.1 Modified `ai/doc.go` (verified lines 5–9, 46–48, 71–72 before editing): subpackage adapters, "one milestone" (AI-37) may add a dependency, "concrete vendor adapters … arrive from AI-25 onward" — `S-AGM-032`'s claims stay verbatim (confirmed by inspection: boundary/import-rule/ADR-0005-§D1 paragraphs untouched). Discharges `S-APC-062, S-APC-063` review scenarios.
- [x] A.10.2 Modified `ai/import_boundary_test.go` (verified lines 96–104 and 134 before editing): dropped the AI-24 allowlist-entrant bullet + "two milestones"→"one" in the doc comment; dropped "the AI-24 transport or" from the guard's own failure-message string literal at line 134. Discharges `S-APC-061`.
- [x] A.10.3 Modified `backend/agent/Makefile` (verified lines 56–59 before editing): corrected the same stale "AI-24 selects a transport (its own ADR gate)" claim, per coordinator ruling as a fourth instance of the identical staleness. **Scope-lag record**: spec rev 2's `NFR-APC-C` text still enumerates only **three** comment sites (the two `doc.go` claims and `import_boundary_test.go`); this Makefile correction is a coordinator-ruled addition beyond that literal text, implemented per the ruling, not as an unrequested change.
- [x] A.10.4 *Review*: re-ran the full `src/ai` suite (the S-AGM-030/S-AGM-032-equivalent checks live there, no dedicated IDs found under those exact labels in this codebase's current test files) and confirmed it passes verbatim; confirmed via `git diff` that every edit under `src/ai/` (doc.go, import_boundary_test.go) is comment- or message-string-only — no declaration, signature or behaviour changed; confirmed (by construction — these are the only Slice-A edits to pre-existing files, and `make test` from `backend/agent/` passed in full, exercising AI-00…AI-23 unmodified) that reverting this change in isolation leaves AI-00…AI-23 green. Covers `S-APC-058, S-APC-059, S-APC-060, S-APC-064`. No red phase (regression + reading confirmation).

## Phase A.11 — Slice-A closure

- [x] A.11.1 Confirmed `go.mod` still declares zero `require` directives (only `module`/`go` lines; no `go.sum`); both AI-00 import guards (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires`) pass with `openaicompat` admitted by the existing module-prefix rule; `allowedNonStdlibPrefixes` slice value itself is byte-for-byte unmodified in the diff (only its surrounding comment changed). Covers `S-APC-056, S-APC-057`.
- [x] A.11.2 Ran full `make test` from `backend/agent/`: **PASS**, exit 0, 466 `--- PASS` lines, 0 `--- FAIL` lines, across `src/agenttest`, `src/ai`, `src/ai/openaicompat`. Ran `make lint`: 5 issues found on first run (3 `errcheck` unchecked-error, 2 `staticcheck` false-positives on deliberate test assertions), all fixed with `nolint` + explanatory comments where the flagged pattern was the test's actual point; **0 issues** on re-run. Re-ran `make test` twice more with `-count=1` after the lint fixes: identical `ok` results both times, no flakiness. PR-A description evidence assembled in the apply-progress artifact: all 5 mutation-proof reds (A.3.3, A.3.4, A.3.5, A.4.3, A.5.3) plus final green.
- [x] A.11.3 Red/green/refactor evidence recorded per task above (this file) and in the Evidence Log table below; every Slice-A `*(review)*` scenario carries a recorded confirmation.

---

## Phase B.0 — R-APC-008 guard implementation (green from birth)

- [x] B.0.1 Create `ambient_authority_test.go`: `go/parser`+`go/ast`+`go/token` call-site scan over this package's non-test `.go` files; forbidden set `{os, os/exec, syscall, io/ioutil}` resolved by local identifier (alias-aware via `file.Imports`); dot-import of a forbidden path is an independent violation; failure message names file, line, package. Base case: passes clean on landed sources with no violation. Covers `S-APC-033, S-APC-034, S-APC-035`.

## Phase B.1 — R-APC-011 / R-APC-012 scope and limitation, documented

- [x] B.1.1 Document the non-test-only scan scope in the guard's own comment, citing the in-repo precedent (`import_boundary_test.go`'s "-test would also pull in `testing`, which imports `os`"). *Review*: `S-APC-043`. No red phase.
- [x] B.1.2 Add tests: guard stays green while adapter tests use `httptest`; guard's own source is exempt as meta-tooling. Covers `S-APC-044, S-APC-045`.
- [x] B.1.3 Document the local-shadow false-positive limitation as an accepted, reversible non-requirement scoped to this change. *Review*: `S-APC-046, S-APC-047`. No red phase.
- [x] B.1.4 *Review*: confirm adapter documentation states the ambient-authority guarantee is scoped to the adapter's own sources and its own client, and that an injected client's transport is its injector's responsibility. Covers `S-APC-068` (cross-ref A.0.1, closed formally here since B closes the guard node).

## Phase B.2 — R-APC-010 four recorded bite proofs (non-negotiable, own tasks each)

- [x] B.2.1 **Bite proof 1/4 — plain environment read**: stage `os.Getenv(...)` in real non-test adapter source (e.g. `client.go`); run `make test`; guard fails naming that file and package; **record the red output verbatim**; revert. Covers `S-APC-038`.
- [x] B.2.2 **Bite proof 2/4 — aliased import**: stage `osx "os"` aliased call; run; guard fails (alias resolved via local-identifier map, not bypassed); record red; revert. Covers `S-APC-039`.
- [x] B.2.3 **Bite proof 3/4 — process spawn**: stage `exec.Command(...)`; run; guard fails naming `os/exec`; record red; revert. Covers `S-APC-040`.
- [x] B.2.4 **Bite proof 4/4 — dot-import**: stage `import . "os"` with no call site; run; guard fails on the dot-import itself (independent violation, no `go/types` needed); record red; revert. Covers `S-APC-041`.
- [x] B.2.5 After all four scratch violations are dropped, re-run the suite; record final green alongside the four recorded reds in the same evidence log. Covers `S-APC-042`.

## Phase B.3 — Slice-B closure

- [x] B.3.1 Run `make test` green + `make lint` clean.
- [x] B.3.2 Assemble PR-B description carrying all four bite-proof red runs (B.2.1–B.2.4) and the final green run (B.2.5) verbatim. Covers `S-APC-076` (NFR-APC-G).

---

## Phase C.0 — R-APC-013 test-server viability

- [x] C.0.1 RED `viability_test.go`: `httptest.NewServer` handler records `Authorization` header + path; construct adapter with **nil** `HTTPClient` (adapter-built path); drive one request; assert exactly one request observed, Bearer credential present in dialect shape, path equals `R-APC-006`'s joined path, a second construction with a different credential carries the second value. Covers `S-APC-048…051`. **Note**: passed immediately (no red) — see Evidence Log for what provides falsifiability instead.
- [x] C.0.2 GREEN: wire the probe against `New`/`newRequest`; confirm it passes.

## Phase C.1 — R-APC-013 proxy immunity (SERIAL — no `t.Parallel()`)

- [x] C.1.1 RED: `t.Setenv("HTTP_PROXY", <dead-address>)`; assert the probe still reaches the local server unaffected — the adapter-built client's unset proxy resolver (`R-APC-009`) makes this trustworthy. Covers `S-APC-052`. **Must not add `t.Parallel()`** — `t.Setenv` panics under parallel execution; Go runs a package's non-parallel tests strictly sequentially, so this and Phase A.4's timing pair queue rather than overlap, and `t.Setenv`'s automatic cleanup restores the environment before the next serial test runs. **Note**: passed immediately (no red) — see Evidence Log.
- [x] C.1.2 GREEN: confirm the adapter-built transport's unset `Proxy` (from A.3.2) already satisfies this without further code change.

## Phase C.2 — R-APC-014 probe-doc review

- [x] C.2.1 *Review*: confirm the viability probe's own documentation states attachment-only scope, naming AI-32.5/AI-36.1 as wire-level owners (cross-ref A.8.3). Covers the probe-specific portion of `S-APC-054, S-APC-055`. No red phase. **Discharges the A.8.3 deferral**: the viability-probe-doc half of that confirmation, explicitly left open in Slice A, is closed here.

## Phase C.3 — Slice-C closure

- [x] C.3.1 Run `make test` green + `make lint` clean; record evidence.

---

## Phase D — Cross-cutting determinism and final coverage sweep

- [x] D.1 Run the whole milestone's test set repeatedly under `-race`; confirm identical results and no race-detector report. Covers `S-APC-069`.
- [x] D.2 *Review*: confirm Phase A.4's timing pair declares no parallelism, asserts the control's failure **shape** rather than duration, and uses a wide ratio margin. Covers `S-APC-070`. No red phase. **Completes the Slice-A-partial record**: full confirmation below.
- [x] D.3 Confirm no single test's wall-clock runtime approaches the 60s time-to-headers bound — bounds are asserted structurally (Phase A.5), never waited out. Covers `S-APC-071`.
- [x] D.4 Final evidence walk: every runnable scenario in this file carries recorded red output, recorded green output and a refactor note; every `*(review)*` scenario carries a recorded confirmation naming what was read; Slice-B's PR carries all four bite-proof reds plus final green. Covers `S-APC-075, S-APC-076` (NFR-APC-G).

---

**Milestone AI-25 status**: all tasks in this file (A.0–A.11, B.0–B.3, C.0–C.3, D) are now `[x]`. See the Requirement/NFR Coverage Map below — all 14 requirements and all 7 NFRs are discharged. AI-25 is complete.

---

## Requirement / NFR Coverage Map

| ID | Discharged by |
|---|---|
| R-APC-001 | Phase A.1 |
| R-APC-002 | Phase A.2 |
| R-APC-003 | Phase A.3 (+ mutation proofs #1, #2) |
| R-APC-004 | Phase A.4 (+ mutation proof #3) |
| R-APC-005 | Phase A.5 (+ mutation proof #4) |
| R-APC-006 | Phase A.6 |
| R-APC-007 | Phase A.7 |
| R-APC-008 | Phase B.0 |
| R-APC-009 | Phase A.3 (`S-APC-036`) + mutation proof #5 (`S-APC-037`) — tested in Slice A per design, though numbered under AI-25.2 in the spec |
| R-APC-010 | Phase B.2 (4 bite proofs) |
| R-APC-011 | Phase B.1.1–B.1.2 |
| R-APC-012 | Phase B.1.3 |
| R-APC-013 | Phase C.0, C.1 |
| R-APC-014 | Phase A.8, C.2 |
| NFR-APC-A | Phase A.11.1 |
| NFR-APC-B | Phase A.10.4 |
| NFR-APC-C | Phase A.10 (+ scope-lag record at A.10.3) |
| NFR-APC-D | Phase A.0.1, B.1.4 |
| NFR-APC-E | Phase D.1–D.3 |
| NFR-APC-F | Phase A.9 |
| NFR-APC-G | Phase A.11.3, B.3.2, C.3.1, D.4 |

All 14 requirements and all 7 NFRs are mapped. No gap found.

## Work-Unit Commit Sequence (per `work-unit-commits`)

**Slice A**: (1) `feat(openaicompat): scaffold package doc and client/credential types` · (2) `feat(openaicompat): construct client with injected-value observation via stub transport` · (3) `feat(openaicompat): fail construction on malformed endpoint or empty credential` · (4) `feat(openaicompat): build safe-default bounded client without mutating shared transports` (includes mutation proofs #1/#2/#5 staged-then-reverted) · (5) `feat(openaicompat): prove absence of whole-request timeout behaviorally` (+ mutation proof #3) · (6) `feat(openaicompat): wire connect/idle bounds structurally` (+ mutation proof #4) · (7) `feat(openaicompat): join endpoint paths without dropping or doubling segments` (A2 fission point) · (8) `test(openaicompat): assert no stubbed streaming and no AI-20 satisfaction` · (9) `feat(openaicompat): keep credential opaque across format and serialization` · (10) `test(openaicompat): assert totality across extreme construction inputs` · (11) `docs(ai): correct stale AI-24 transport/dependency claims across doc.go, import_boundary_test.go, Makefile` · (12) `chore(openaicompat): record slice-A red/green evidence`.

**Slice B**: (1) `feat(openaicompat): add call-site ambient-authority guard over adapter sources` · (2) `test(openaicompat): prove guard bites on plain and aliased environment reads` · (3) `test(openaicompat): prove guard bites on process spawn and dot-import` · (4) `chore(openaicompat): record all four bite-proof reds and final green`.

**Slice C**: (1) `test(openaicompat): prove viability against local test server with credential attachment` · (2) `test(openaicompat): prove proxy-environment immunity of adapter-built client` · (3) `chore(openaicompat): record slice-C evidence and close milestone documentation review`.

## Evidence Log — Slice A (populated during sdd-apply)

TDD Cycle Evidence (runnable scenarios, RED → GREEN → REFACTOR):

| Task | Scenario(s) | Red recorded | Green recorded | Refactor note |
|---|---|---|---|---|
| A.1.1/A.1.2 | S-APC-001…004 | compile failure: `undefined: New` / `c.newRequest undefined` | `go test -run 'TestNew_InjectedClientObservesOutboundRequest\|TestNew_TwoAdaptersDoNotShareStubbedClients'` → PASS | Shared `stubTransport` + `driveOneRequest` extracted, reused by every later test |
| A.2.1/A.2.2 | S-APC-006…010 | 7/7 subtests failed (`errors.Is` false / `New() error = nil`) before validation | same run → all subtests PASS | none needed |
| A.3.1/A.3.2 | S-APC-011…013,015,016,036 | nil-pointer panic (`c.httpClient` nil) before nil-client branch existed | `TestNew_DoesNotMutateProcessWideDefaults`, `..._SharedInjectedClientNotMutated...`, `TestConfig_Surface...`, `..._AdapterBuiltClientIsNeither...`, `..._AdapterBuiltTransportLeavesProxyUnset` → all PASS | none needed |
| A.4.1/A.4.2 | S-APC-017…019 | **N/A — passed on first run** (A.3's `newDefaultHTTPClient` already left `Client.Timeout` zero); genuine falsifiability is A.4.3's mutation proof | `TestTimeout_NoWholeRequestCapOnAdapterBuiltClient` → PASS (1.21s total, control 0.40s + adapter 0.81s) | none needed |
| A.5.1/A.5.2 | S-APC-021,022 | **N/A — passed on first run** (bounds already wired in A.3.2); genuine falsifiability is A.5.3's mutation proof | `TestNew_AdapterBuiltTransportBoundsExist` → PASS | none needed |
| A.6.1/A.6.2 | S-APC-024…029 | compile failure: `undefined: joinRequestPath` (4 call sites) | `TestJoinRequestPath` (7 subtests) + `..._AbsoluteURLShapedSegment...` + `..._NonDestructiveOnStoredBase` → all PASS | `newRequest` rewired from bare `c.base.JoinPath` to `joinRequestPath(c.base, ...)` |
| A.7.1/A.7.2 | S-APC-030…032 | **N/A — passed on first run** (`*Client` never had a `Stream` method; negative-property pin) | `TestClient_HasNoStreamingEntryPoint`, `..._DoesNotSatisfyModelProviderAtRuntime` → PASS; re-ran `src/agenttest`'s `TestModelProviderInterface_SignatureGuard` (+3 sibling tests) → PASS unmodified | none needed |
| A.8.1/A.8.2 | S-APC-053 | **genuinely red**: all 4 format verbs (`%v`,`%s`,`%+v`,`%#v`) leaked the raw token (`String`/`GoString` deliberately not yet implemented) | `TestCredential_NeverRendersRawToken` → PASS after adding `String()`/`GoString()` | none needed |
| A.9.1/A.9.2 | S-APC-072,073 | **N/A — passed on first run** (totality already satisfied by A.2+A.3); table covers empty endpoint, empty credential, whitespace-only endpoint, both empty, plus the no-client-supplied success case | `TestNew_TotalityAcrossExtremeInputs` (4 subtests) + `TestNew_SucceedsWithNoHTTPClientSupplied` → all PASS, no panic in any case | none needed |

Mutation-detection proofs (Slice A, all 5 — distinct from ordinary TDD RED; each staged against a **finished, green** implementation, then reverted):

| # | Scenario | Staged mutation | Recorded red `make test` output (verbatim excerpt) | Reverted, re-run green |
|---|---|---|---|---|
| 1 | S-APC-005 | `New` ignored `cfg.HTTPClient`, always called `newDefaultHTTPClient()` | `--- FAIL: TestNew_InjectedClientObservesOutboundRequest` / `--- FAIL: TestNew_SharedInjectedClientNotMutatedAcrossTwoConstructions` / `--- FAIL: TestNew_TwoAdaptersDoNotShareStubbedClients`, each: `httpClient.Do() error = Post "http://one.invalid/...": dial tcp: lookup one.invalid: no such host, want nil` | Yes — full suite green after revert |
| 2 | S-APC-014 | `newDefaultHTTPClient` set `http.DefaultTransport.(*http.Transport).ResponseHeaderTimeout = 60s` (chosen over `IdleConnTimeout`, which was already 90s on the real default and would have been a no-op) | `--- FAIL: TestNew_DoesNotMutateProcessWideDefaults`: `http.DefaultTransport observably changed: before={... responseHeaderTimeout:0 ...} after={... responseHeaderTimeout:60000000000 ...}` | Yes — green after revert |
| 3 | S-APC-020 | `newRequest` wrapped `ctx` in `context.WithTimeout(ctx, 300ms)` | `--- FAIL: TestTimeout_NoWholeRequestCapOnAdapterBuiltClient/adapter-built_client_reads_the_full_stream_to_completion`: `reading body: context deadline exceeded, want nil` | Yes — green after revert (1.21s total, both subtests pass) |
| 4 | S-APC-023 | `newDefaultHTTPClient` dropped `DialContext` and `ResponseHeaderTimeout` from the built transport | `--- FAIL: TestNew_AdapterBuiltTransportBoundsExist`: `DialContext is nil, want non-nil` and `ResponseHeaderTimeout = 0s, want 1m0s` | Yes — green after revert |
| 5 | S-APC-037 | `newDefaultHTTPClient` returned `&http.Client{Transport: http.DefaultTransport}` | `--- FAIL: TestNew_AdapterBuiltTransportLeavesProxyUnset`: `Proxy resolver is set, want nil` / `--- FAIL: TestNew_AdapterBuiltClientIsNeitherDefaultClientNorDefaultTransport`: `adapter-built client's transport is http.DefaultTransport itself` | Yes — green after revert |

Review-obligation confirmations (Slice A `*(review)*` scenarios — no red phase):

| Task | Scenario(s) | Recorded confirmation |
|---|---|---|
| A.0.1 | S-APC-065…068 | `openaicompat/doc.go` read in full: states (a) no whole-request cap + why (streaming-footgun paragraph), (b) injection-only credential/no ambient authority, (c) proxy resolution deliberately unset + why, (d) the guard-scope boundary at an injected client. All four present. |
| A.8.3 | S-APC-054,055 | Spec's R-APC-014 section (`specs/ai-provider-client/spec.md`) read: states attachment-only scope, names AI-32.5 and AI-36.1 as wire-level owners, and permits `S-APC-053`'s type-shape assertion as the one exception. Viability-probe-doc half deferred to Slice C (`viability_test.go` does not exist yet). |
| A.9.3 | S-APC-074 | `openaicompat/doc.go`'s "An absent HTTP client is not a fault" section read: states absence selects the adapter's own bounded client, "never treated as an omission." |
| A.10.4 | S-APC-058,059,060,064 | `git diff` of `src/ai/doc.go` and `src/ai/import_boundary_test.go` read in full: every changed line is a `//` comment or an error-message string literal; no declaration, signature, struct value or control-flow changed. `backend/agent/src/` listed: exactly two entries (`agenttest`, `ai`) — `openaicompat` is a subdirectory of `ai`, not a third top-level entry. Full `make test` (which exercises AI-00…AI-25 together, since this repo has no isolated AI-00…AI-23-only target) passed with 0 failures, which is the available evidence that AI-00…AI-23 remain green with this change applied; no separate isolated-revert run was performed within this apply batch. |
| D.2 (partial, Slice-A-relevant) | S-APC-070 | `timeout_test.go` read: the timing pair declares no `t.Parallel()` anywhere in its call tree, asserts the control's failure via `net.Error`+`Timeout()` shape and a chunk-count bound rather than any duration, and uses a cap (2d=400ms) comfortably inside a handler span (4d=800ms) that does not depend on machine speed. |

Determinism (Slice A portion of NFR-APC-E, D.1/D.3): `go test -race -count=1 ./...` run twice after the final lint fix — identical `ok` results both times (`src/agenttest`, `src/ai`, `src/ai/openaicompat`), no flakiness. Longest single test observed: the timing pair at ~1.2s combined; no test approached the 60s `defaultResponseHeaderTimeout` bound. Full milestone-wide D.1/D.4 (spanning Slices B/C) is out of scope for this apply batch.

## Evidence Log — Slice B (populated during sdd-apply)

**Guard evidence-class note** (per this milestone's Evidence Classes, applied as framed rather than as a deviation): AI-25.2 is a `[guard]` node, green from birth over clean code. Ordinary TDD red→green does not apply to Phase B.0/B.1's tests: `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources` (S-APC-033), `TestAmbientAuthority_ForbiddenSetIsPackageScopedDenyByDefault` (S-APC-034), `TestAmbientAuthority_IsAdapterSourceFile` (S-APC-045) and `TestAmbientAuthority_TestSourcesStayGreenEvenWithForbiddenCalls` (S-APC-044) all passed on first run, written together with their implementation in one commit (`bdda353`) — this is Evidence Class 1's stated reading, not an invented false red. This guard's falsifiability comes entirely from Phase B.2's four recorded bite proofs below, which ARE its RED phase (Evidence Class 2).

Bite-proof evidence (Phase B.2, all four — `R-APC-010`, non-negotiable). Each was staged one at a time directly in `client.go` (a real, non-test, already-landed adapter source file), run via `make test` (full suite), the exact `--- FAIL` line and guard message recorded verbatim below, then reverted via `git checkout -- backend/agent/src/ai/openaicompat/client.go` and confirmed clean (`git status --short` empty) before the next proof was staged. None of these edits were committed — they exist only in this recorded evidence.

| # | Scenario | Staged violation (real, non-test adapter source) | Recorded red `make test` output (verbatim) | Reverted, re-run green |
|---|---|---|---|---|
| 1/4 | S-APC-038 (plain environment read) | Added `"os"` to `client.go`'s import block plus `func scratchAmbientAuthorityBiteProof1() string { return os.Getenv("OPENAICOMPAT_SCRATCH_BITE_PROOF_1") }` | `ambient_authority_test.go:257: ambient authority: client.go:18: call os.Getenv reaches forbidden package "os" (R-APC-008: no environment variable read, no filesystem path touched)` / `--- FAIL: TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources (0.00s)` — exactly 1 `--- FAIL` in the whole suite, exit code 2 | Yes — `git checkout --` reverted `client.go`; `git status --short` empty; full suite green after |
| 2/4 | S-APC-039 (aliased-import environment read) | `osx "os"` in place of `"os"`, plus `func scratchAmbientAuthorityBiteProof2() string { return osx.Getenv("OPENAICOMPAT_SCRATCH_BITE_PROOF_2") }` | `ambient_authority_test.go:257: ambient authority: client.go:19: call osx.Getenv reaches forbidden package "os" (R-APC-008: no environment variable read, no filesystem path touched)` — the alias resolved back to `"os"` via the local-identifier map, not bypassed; exactly 1 `--- FAIL`, exit code 2 | Yes — reverted, confirmed clean, green after |
| 3/4 | S-APC-040 (process spawn) | `"os/exec"` import plus `func scratchAmbientAuthorityBiteProof3() *exec.Cmd { return exec.Command("echo", "scratch-bite-proof-3") }` | `ambient_authority_test.go:257: ambient authority: client.go:19: call exec.Command reaches forbidden package "os/exec" (R-APC-008: no process spawned)` — exactly 1 `--- FAIL`, exit code 2 | Yes — reverted, confirmed clean, green after |
| 4/4 | S-APC-041 (dot-import environment read) | `. "os"` import plus `func scratchAmbientAuthorityBiteProof4() string { return Getenv("OPENAICOMPAT_SCRATCH_BITE_PROOF_4") }` (bare call, no selector — invisible to the call-site scan by design) | `ambient_authority_test.go:257: ambient authority: client.go:7: dot-import of forbidden package "os" (R-APC-008: no environment variable read, no filesystem path touched)` — the guard flagged the **import declaration itself** (line 7 — the `. "os"` line), never resolving the bare `Getenv` call; exactly 1 `--- FAIL`, exit code 2 | Yes — reverted, confirmed clean, green after |

Final green (Phase B.2.5, S-APC-042): all four scratch violations dropped and confirmed absent (`git diff --stat` empty on `client.go`); `go test -race -v -count=1 ./...` from `backend/agent/`: **exit 0**, **470** `--- PASS` (466 Slice-A baseline + 4 new top-level Slice-B tests), **0** `--- FAIL`, across `src/agenttest`, `src/ai`, `src/ai/openaicompat`.

Review-obligation confirmations (Slice B `*(review)*` scenarios — no red phase):

| Task | Scenario(s) | Recorded confirmation |
|---|---|---|
| B.1.1 | S-APC-043 | `ambient_authority_test.go`'s file-level comment read in full: states the non-test-only scan scope and cites the in-repo precedent verbatim — `import_boundary_test.go`'s own words, "-test would also pull in \"testing\", which imports \"os\" itself... a guard that scanned test imports could never pass" — rather than leaving the reason to be rediscovered. |
| B.1.3 | S-APC-046, S-APC-047 | Same file's "Recorded limitation: a local shadow can false-positive" section read: states the local-shadow false-positive explicitly, names the exact cost of closing it (a non-stdlib type-checking dependency — `go/types` at minimum, realistically `golang.org/x/tools/go/analysis` — spending the zero-module-requires invariant, NFR-APC-A), and states the record is scoped to this change and reversible by a later ADR, not a permanent prohibition. Confirmed no such local shadow exists in this package's own sources today (all four bite proofs used the real, unshadowed package names). |
| B.1.4 | S-APC-068 | `openaicompat/doc.go`'s "The no-ambient-authority guarantee is scoped, not absolute" section (landed in Slice A) re-read here: confirmed it states the guarantee covers only this package's own sources and the client it builds for itself, and that an injected client's transport remains its injector's responsibility. Closed formally in Slice B, since this is the guard node that discharges it (cross-ref A.0.1, which first stated the same four documentation obligations). |

Determinism and closure (NFR-APC-E, B.3.1): `go test -race -v -count=1 ./...` run again after `make lint` — identical `ok` results, no flakiness, no race-detector report. `make lint`: **0 issues** (`go vet ./...` clean; `bin/golangci-lint run --config=.golangci.yml ./...` — "0 issues."; no new `nolint` needed for `ambient_authority_test.go`). `go.mod` unchanged: still declares zero `require` directives. Both AI-00 guards (`TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestLayer1_ModuleHasNoDependencies_ZeroRequires`) re-ran green; `allowedNonStdlibPrefixes` is untouched by this slice (no file outside `openaicompat/` and `tasks.md` was edited). Diff vs `feat/ai-25a-injected-construction` (chain base, commit `37ffea2`): one new file, `ambient_authority_test.go`, 334 lines added, 0 removed — well under the design's ~700–1,400 corrected forecast for Slice B and the 5,000-line session budget.

## Evidence Log — Slice C (populated during sdd-apply)

**Header-shape resolution (task preface, R-APC-013)**: AI-24's merged `decision.md` was read at apply time, as instructed. § 4's credential-handling-boundary axis states: "A bearer token is attached as a single `Authorization: Bearer <token>` header at request-construction time, entirely inside code this repo owns and can audit." § 11's credential boundary states the same shape under "Receives": "An opaque bearer-token value, supplied at construction through injected configuration." Both sections settle the literal header form — `Authorization: Bearer <token>` — which is also exactly what `request.go` already implements (`Credential.bearer()` returns `"Bearer " + c.token`, attached via `req.Header.Set("Authorization", ...)`). Merged fact and landed Slice-A code agree; `viability_test.go`'s assertions are pinned against this confirmed literal, not a placeholder, and its file-level doc comment records the same citation.

**Evidence-class disclosure, per this milestone's own discipline (do not fabricate a red)**: both C.0.1 and C.1.1 passed on first run — no genuine red phase occurred for either. This is the same disclosed pattern already recorded for A.4.1, A.5.1, A.7.1 and A.9.1: the behaviour each scenario exercises was already fully implemented and independently proven correct in Slice A (Bearer attachment: Phase A.1; endpoint joining: Phase A.6; adapter-built client construction, bounds and unset `Proxy`: Phases A.3/A.5), so composing them into one real-network, real-server round trip for the first time did not surface a gap. What provides falsifiability instead:

- For C.0.1 (S-APC-048…051): Phase A.1's stub-transport tests (`TestNew_InjectedClientObservesOutboundRequest`, `TestNew_TwoAdaptersDoNotShareStubbedClients`) already proved credential attachment and cross-construction non-leakage at the unit level; this test composes the same proven behaviour end-to-end against a real socket instead of a stub, and would fail if any of that composition regressed — confirmed directly, see the scratch verification below.
- For C.1.1 (S-APC-052): Slice A's mutation proof #5 (`S-APC-037`, Phase A.3.5) already staged "route the adapter-built client through `http.DefaultTransport`" and recorded a red against the structural nil-proxy and fresh-identity assertions. This test is a second, independent, **behavioral** line of defense for the identical regression class — it would also fail if that same mutation were re-staged, because a `DefaultTransport`-routed client would consult `HTTP_PROXY` and fail to reach the dead proxy address's non-existent listener instead of reaching the real local server directly.

**Scratch verification (not committed, confirms falsifiability, reverted after use)**: to confirm the disclosure above is not merely asserted, `request.go`'s `req.Header.Set("Authorization", c.credential.bearer())` line was replaced with `_ = c.credential.bearer()` (header never attached), and `go test -race -v -run TestViability ./src/ai/openaicompat/...` was run. All three viability assertions failed, exactly as expected:

```
viability_test.go:217: Authorization header = "", want "Bearer viability-proxy-token" — the request must still carry the credential despite the poisoned proxy environment (S-APC-052)
--- FAIL: TestViability_AdapterBuiltClientIgnoresProxyEnvironment (0.00s)
viability_test.go:151: Authorization header = "", want "Bearer viability-token-one" (S-APC-049)
viability_test.go:175: Authorization header = "", want "Bearer viability-token-two" — attachment derives from the injected credential, not a hard-coded value (S-APC-051)
--- FAIL: TestViability_RequestReachesLocalServerWithCredentialAttached (0.00s)
    --- FAIL: TestViability_RequestReachesLocalServerWithCredentialAttached/first_construction,_first_credential (0.00s)
    --- FAIL: TestViability_RequestReachesLocalServerWithCredentialAttached/second_construction,_different_credential_carries_the_second_value (0.00s)
FAIL	github.com/cachicamas/backend/agent/src/ai/openaicompat	0.539s
```

All three top-level assertions failed (exit code 1), not merely the two anticipated in the disclosure above — the proxy-immunity test's own credential check (S-APC-052) shares the same underlying attachment path and caught the same break independently, which is itself further evidence the two tests are not redundant with each other. `git checkout -- backend/agent/src/ai/openaicompat/request.go` reverted the break; the full suite was re-confirmed green (`go test -race -v -count=1 ./...` → `ok` across all three packages) before proceeding.

Runnable-scenario evidence:

| Task | Scenario(s) | Red recorded | Green recorded | Refactor note |
|---|---|---|---|---|
| C.0.1/C.0.2 | S-APC-048…051 | **N/A — passed on first run** (Slice A's Phase A.1/A.6/A.3/A.5 already proved the composed behaviour); falsifiability confirmed instead by the scratch verification above | `go test -race -v -run TestViability ./src/ai/openaicompat/...` → `TestViability_RequestReachesLocalServerWithCredentialAttached` PASS (both subtests) | none needed |
| C.1.1/C.1.2 | S-APC-052 | **N/A — passed on first run** (Slice A's mutation proof #5, `S-APC-037`, already proved the identical regression class structurally); this test is an independent behavioral proof of the same property | `go test -race -v -run TestViability ./src/ai/openaicompat/...` → `TestViability_AdapterBuiltClientIgnoresProxyEnvironment` PASS | none needed |

Review-obligation confirmation (Slice C `*(review)*` scenario — no red phase):

| Task | Scenario(s) | Recorded confirmation |
|---|---|---|
| C.2.1 | S-APC-054, S-APC-055 (probe-specific portion) | `viability_test.go`'s file-level doc comment read in full: its "Scope: attachment only, not a wire-level secrecy guarantee (R-APC-014)" section states that nothing in the file asserts wire-level redaction, provider-returned-content non-disclosure, or log safety; names AI-32.5 as owning the wire-error-body bound and AI-36.1 as owning the exhaustive sentinel sweep; and states explicitly that this file does not duplicate or extend `credential_test.go`'s `S-APC-053` type-shape assertion. This discharges the A.8.3 deferral recorded in Slice A's evidence log (the viability-probe-doc half of that confirmation, left open because `viability_test.go` did not exist yet). |

Determinism (Slice C portion of NFR-APC-E, D.1/D.3): `go test -race -v -count=1 ./...` run 3 times total after Slice C's changes landed (once immediately after each commit, once at final closure) — identical `ok` results every time across `src/agenttest`, `src/ai`, `src/ai/openaicompat`, no flakiness, no race-detector report. Longest single package run observed: `src/ai` at ~3.5s; `src/ai/openaicompat` (carrying every Slice A/B/C test, including the new real-network round trips) at ~2.6–3.0s — no test anywhere near the 60s `defaultResponseHeaderTimeout` bound.

Diff vs `feat/ai-25b-ambient-authority-guard` (chain base): one file, `viability_test.go`, 219 lines added, 0 removed, across 2 commits — within the design's ~200–480 corrected forecast for Slice C and comfortably under the 5,000-line session budget.

## Evidence Log — Phase D (cross-cutting closure, populated during sdd-apply)

**D.1 (S-APC-069)**: the whole milestone's test set (`src/agenttest`, `src/ai`, `src/ai/openaicompat`) was run repeatedly under `-race -count=1` across this session — twice as the Slice-A/B baseline (470 `--- PASS`, 0 `--- FAIL`), then after each Slice-C commit and twice more at final closure (472 `--- PASS`, 0 `--- FAIL` throughout Slice C, since Slice C added exactly 2 new top-level tests to the 470-test baseline). Every run: identical `ok` results, exit 0, no race-detector report.

**D.2 (S-APC-070)**: `timeout_test.go` re-read in full at Slice-C closure to complete the record Slice A's evidence log marked partial. Confirmed: (a) no `t.Parallel()` call appears anywhere in `TestTimeout_NoWholeRequestCapOnAdapterBuiltClient` or either of its two `t.Run` subtests; (b) the control subtest asserts failure **shape** only — a `net.Error` type assertion, `Timeout() == true`, and a chunk count strictly fewer than `chunks` — never an exact duration; (c) the ratio margin is wide: `interval = 200ms`, `chunks = 5`, so the control's cap (`2 * interval = 400ms`) is half the handler's total span (`(chunks-1) * interval = 800ms`) — a 2× margin, not a tight one, so ordinary scheduling jitter on a loaded machine cannot flip the outcome.

**D.3 (S-APC-071)**: confirmed via the determinism runs above — no single test's observed wall-clock duration (longest package total ~3.5s; longest individual test well under 1s outside the deliberate ~1.2s timing pair) approaches the 60s `defaultResponseHeaderTimeout` bound. Every bound this milestone asserts is asserted structurally (Phase A.5's `TestNew_AdapterBuiltTransportBoundsExist`), never waited out.

**D.4 (S-APC-075, S-APC-076, final evidence walk)**: every runnable scenario across Slices A, B and C carries a recorded red-or-disclosed-N/A entry, a recorded green entry and a refactor note in this file's three Evidence Log sections above. Every `*(review)*` scenario across all three slices carries a recorded confirmation naming what was read (Slice A: A.0.1, A.8.3 partial, A.9.3, A.10.4, D.2 partial; Slice B: B.1.1, B.1.3, B.1.4; Slice C: C.2.1, closing A.8.3's deferral; D.2 fully closed here). Slice B's evidence log carries all four bite-proof reds (B.2.1–B.2.4) plus the final green (B.2.5) verbatim, as `S-APC-076` requires. No gap found across the full walk.

**Final milestone-wide evidence**: `make test` (`go test -race -v -count=1 ./...`) from `backend/agent/`: **PASS**, exit 0, **472** `--- PASS`, **0** `--- FAIL`, across `src/agenttest`, `src/ai`, `src/ai/openaicompat`. `make lint`: **0 issues** (`go vet ./...` clean; `golangci-lint run --config=.golangci.yml ./...` — "0 issues."). `go.mod`: unchanged, zero `require` directives (`module github.com/cachicamas/backend/agent` / `go 1.26.3` only). Both AI-00 guards re-ran green with no allowlist edit. Slice B's ambient-authority guard (`TestAmbientAuthority_*`, 4 tests) re-ran green, unaffected by `viability_test.go`'s addition (excluded from the guard's scan as a `_test.go` file, confirmed by `TestAmbientAuthority_IsAdapterSourceFile` still passing). Total diff vs `feat/ai-25b-ambient-authority-guard`: 219 lines added, 0 removed, 1 file (`viability_test.go`), across 2 work-unit commits plus this closure commit.
