```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:120bf27835d3c970543d9ec04e7456c79f6930a7e720d46c91f4e9a6b7ea8bcb
verdict: fail
blockers: 1
critical_findings: 1
requirements: 11/18
scenarios: 68/77
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:1abcb14fb2d561bd2ef623e0ebf91f593c58b0efe9ff56c5970d5e1aea693051
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-ai-observability` (AI-37 — Add the observability boundary, Wave 5 — Harden)
**Branch**: `feat/ai-37-observability` @ `07a5af82` · base `origin/main@ff28240a`
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-37-observability`
**Mode**: Strict TDD · store `hybrid` · `size:exception` pre-granted (size is not a finding)

### Method

Every claim in `apply-progress.md` and `tasks.md` was re-run as a command. Guards were not read for
plausibility — they were **defeated**. All falsification used `go test -overlay` (and, where the guard
shells out to `go list`, `GOFLAGS=-overlay=...` so the subprocess sees the same overlay), so the
worktree was never mutated. `git status --porcelain` was empty in the worktree and in the base
checkout before, during, and after every probe.

---

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 21 |
| Tasks complete | 21 |
| Tasks incomplete | 0 |
| Commits | 8 (`dbbb4e69` … `07a5af82`) |
| Requirements assessed | 18 (16 `R-*` + 2 `NFR-*`) |
| Scenarios assessed | 77 (49 new + 28 restated) |

**OpenSpec artifacts committed**: confirmed by content, not by a clean `git status`. `git diff --name-status origin/main...HEAD`
lists `A openspec/changes/cachicamas-ai-observability/{proposal,design,explore,spec,tasks,apply-progress}.md`
and all three `specs/*/spec.md`. The repo's recorded failure mode (code-only commits) did **not** recur.

---

### Build & Tests Execution

**Build**: PASS
```text
cd backend/agent && make build     # go build -trimpath ./...
exit 0
```

**Tests**: PASS — 9/9 packages `ok`, 0 `FAIL` lines under `-race`
```text
cd backend/agent && make test      # go test -race -v ./...
exit 0
```

**Lint**: `make lint` → `0 issues.` (orchestrator-confirmed; not re-litigated here)
**Coverage**: not run — no coverage target is configured in this module's `Makefile`.

---

### Guard-defeat evidence (the load-bearing half of this report)

Every guard below was defeated by substituting a deliberately defective source file at build time.
The command shape was `go test -overlay=<scratch>/overlay_<n>.json -run <test> ./<pkg>/`.

#### A. Denylist corpus channels — R-AOB-007 / S-AOB-026

Eight leak vectors planted in `startSpan` (overlaying `trace.go`), one per corpus channel:

| # | Channel leaked through | Result |
|---|---|---|
| 1 | attribute **value** | FAIL — `denylist leak: vector "prompt" found in the corpus` |
| 2 | attribute **key** | FAIL — same message |
| 3 | span **name** (`SetName`) | FAIL — same message |
| 4 | span **event name** (`AddEvent`) | FAIL — same message |
| 5 | span **event attribute value** | FAIL — same message |
| 6 | `RecordError` string | FAIL — same message |
| 7 | tool-result text into an attribute | FAIL — `vector "tool-result" found in the corpus` |
| 8 | status **description** (`SetStatus(codes.Error, …)`) | passed at first — see below |

Channel 8's first run passed **not** because the corpus builder misses it, but because
`finalizeSpan` upgrades the status to `codes.Ok` on the success path and the recorder clears the
description on that upgrade — exactly the real OTel API's documented rule (description is used only
for `Error`). A second overlay that removed the `codes.Ok` upgrade left the leak in place and the
test **FAILED** with `vector "prompt" found in the corpus`. **The status-description channel is
scanned; there is no blind spot.** 8/8 channels bite.

#### B. The four non-vacuity guards — R-AOB-008

| Guard | Defeat applied | Result |
|---|---|---|
| 1 — positive control | `sweep.Scan`'s `bytes.Contains` branch made unreachable | FAIL at `ai37_denylist_test.go:182` — `SelfTest: vector "prompt" did not bite its own needle` (a `t.Fatalf`, so no clean scan can run behind a broken control) |
| 2 — corpus non-empty | `tracetest.Provider.record` made a no-op | FAIL at `:227` — `Corpus() is empty — the whole absence claim would pass vacuously` |
| 2b — attribute cardinality | `terminalCompletionAttributes` dropped from `finalizeSpan` | FAIL at `:238` — `recorded attribute count = 11 across 2 span(s), want at least 12` |
| 3 — event coverage | `Recording.Events()` filtered to drop `EventKindToolCallStart` | FAIL at `:251` — `drained recording never contained an event of kind tool_call_start` |
| 4 — recorded overlay RED | reproduced independently (channel A#1 above) | FAIL, exit non-zero, worktree untouched |
| 5 — captured-field count (S-AOB-034) | `writeCorpus` made to skip the status description | FAIL — `Corpus() captured 5 of the 6 content-bearing methods' effects, want all 6` |
| 5b — same | `writeCorpus` made to skip attribute **values** | FAIL — `Corpus() does not contain the SetAttributes sentinel` |

**Corpus is populated at assertion time**: proven empirically, not assumed — defeat 2 shows the test
fails the instant the corpus is empty, and defeat 2b shows it fails at 11 recorded attributes.

**Canary placement**: all six vectors sit in string-valued fields and are rendered through
`attribute.Value.String()`. `attribute.Value.Emit()` is deprecated at otel v1.44.0; `String()` is the
API's own current accessor and is *not* a struct dump — defeat 5b proves the corpus collapses the
moment that accessor is removed from the walk. None of the seven canaries appears as a contiguous
literal anywhere in the tree (`grep -r` over all 7, zero hits).

#### C. The dependency guard — R-AGM-005 / R-AGM-008

Scratch importers overlaid into `backend/agent/src/ai/`, with `GOFLAGS=-overlay=…` so the guard's own
`go list -deps -test` subprocess sees them. **No `go.mod`, `go.sum` or worktree byte was modified.**

| Scratch import | Result |
|---|---|
| root `go.opentelemetry.io/otel` (the global getter § D3 permits) | FAIL — deny-by-default (`import_boundary_test.go:154`) names `go.opentelemetry.io/otel`, **and** `otel/metric`, `otel/metric/embedded`, `otel/baggage`, `otel/internal/baggage`, `otel/propagation`, `otel/internal/global`, `otel/internal/errorhandler`, `go.opentelemetry.io/auto/sdk`, `auto/sdk/internal/telemetry`, `go-logr/logr`, `go-logr/logr/funcr`, `go-logr/stdr` |
| same, closure pin | FAIL — `closure mismatch: want 10 entries […], got 24 non-stdlib entries […]` (`R-AGM-008`) |
| `go.opentelemetry.io/otel/sdk/trace` | FAIL — `:150` `Layer 1 must not import "go.opentelemetry.io/otel/sdk/trace"` / `rule: ADR 0005 § D3: the OTel SDK belongs to a composition root` (7 sdk paths named) |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace` | FAIL — `:150` `…/exporters/otlp/otlptrace/internal/tracetransform` / `rule: ADR 0005 § D3: OTel exporters belong to a composition root` |
| `github.com/cachicamas/backend/database_administrator/src/domain` | FAIL — `:150` `rule: ADR 0005 § D1 row 1: no package of another backend module` |
| `wantExternalClosure` equality vs prefix subset | the root-otel run's `got` set contains `go.opentelemetry.io/otel/semconv/v1.37.0`, which the **prefix** entry `go.opentelemetry.io/otel/semconv` admits but the enumerated closure does not name — this is exactly `S-AGM-070`'s shape, and the pin FAILED on it |

**The five-prefix allowlist does not admit** root `go.opentelemetry.io/otel`, `otel/metric`,
`otel/baggage`, `otel/propagation` or `auto/sdk` — proven empirically above, and by reading
`isAllowed` (`import_boundary_test.go:334-341`): prefixes are `modulePath`, `…/otel/trace`,
`…/otel/attribute`, `…/otel/codes`, `…/otel/semconv`, `github.com/cespare/xxhash/v2`; none is a
prefix of, or equal to, any of those five paths.

#### D. Span-end-exactly-once on every path — R-AOB-003

| Mutation | Rows that failed |
|---|---|
| `finalizeSpan`'s failure branch never calls `End()` | **10** post-handover rows — `completion_build_failure`, `in_band_error`, `malformed_chunk_json`, `applychunk_failure_malformed_identity`, `emit_race_cancellation`, `feed_error_frame_too_large`, `eof_finish_error_truncated`, `eof_incomplete_no_sentinel`, `mid_stream_ctx_cancellation`, `read_error_abrupt_close` — each `span 0 (name "chat") ended 0 time(s), want exactly 1` |
| `endSpanPreHandover` never calls `End()` | **4** pre-handover rows — `pre_handover_retry_exhausted`, `pre_handover_non_retryable`, `pre_handover_cancelled_in_loop`, `pre_handover_content_type_refusal` |
| `finalizeSpan` calls `End()` twice on success | `success_done` — `ended 2 time(s), want exactly 1` |

All 15 rows of the design's terminal table bite, failure and cancellation paths included. This also
**discharges `S-AOB-011`** (the uniform-close falsifier), which `apply` never recorded.

#### E. Other falsifiers

| Claim | Defeat | Result |
|---|---|---|
| `S-AOB-014` — key spelled exactly | `genAISystemKey` renamed to `"genai.system"` | FAIL, and **names the offending key**: `recorded attribute key "genai.system" is not in the twelve-key § D3 allowlist` |
| `S-AOB-019` — streaming span ends at/after the terminal event | `finalizeSpan` called *before* `sendEvent(completion)` | FAIL — `stream.event_count = 4, want 5 (the number of events actually drained)` |
| `S-APC-081` — injected provider proven used, never by a field read | `New` stores `cfg.TracerProvider` then derives the tracer from a fresh `noop` provider | FAIL — 6 AI-37 tests plus every `TestAI37_SpanLifecycle` row |
| pre-existing `parseAllowedNonStdlibPrefixes` bug fix | `*ast.BasicLit` handling reverted | FAIL — `allowedNonStdlibPrefixes = [github.com/cachicamas/backend/agent     ], want […6 entries…]` |

---

### Spec Compliance Matrix

#### `ai-observability-boundary` — R-AOB-001…009, NFR-AOB-001/002

| Requirement | Scenario | Evidence | Result |
|---|---|---|---|
| R-AOB-001 | S-AOB-001 | `import_boundary_test.go:194-205` 10-entry closure; root otel absent; defeat C | COMPLIANT |
| R-AOB-001 | S-AOB-002 | `import_boundary_test.go:113-131` — each entry cites ADR 0005 § D3; the two closure-only entries state the closure reasoning in place | COMPLIANT |
| R-AOB-001 | S-AOB-003 | `import_boundary_test.go:99-107` records the declination and names `auto/sdk` among deliberately-absent paths, but does not state the spec's own reason (*the root's closure contains an SDK-named package*) at the guard site | PARTIAL |
| R-AOB-001 | S-AOB-004 | no process-global telemetry read/write exists (root `otel` is not importable — defeat C); landed `ambient_authority_test.go`; `TestAI37_NoopEquivalence_CorpusFromNoTracerRunIsUntouched` | COMPLIANT |
| R-AOB-002 | S-AOB-005 | defeat C, SDK row — `rule: ADR 0005 § D3: the OTel SDK belongs to a composition root` | COMPLIANT |
| R-AOB-002 | S-AOB-006 | defeat C, exporter row | COMPLIANT |
| R-AOB-002 | S-AOB-007 | `git diff --name-status origin/main...HEAD` — no scratch file; `go.mod` carries exactly 3 requires | COMPLIANT |
| R-AOB-003 | S-AOB-008 | `ai37_span_lifecycle_test.go:191-202` | COMPLIANT |
| R-AOB-003 | S-AOB-009 | span starts at `stream.go:222`, before `retry.Loop` at `:224`; `pre_handover_retry_exhausted` proves one span across a retried request. **No traced retried-then-*succeeded* row exists**, and `tracetest.Span` records no timestamps, so "duration encloses every attempt" is not assertable | PARTIAL |
| R-AOB-003 | S-AOB-010 | 15-row table, `ai37_span_lifecycle_test.go:186-450` | COMPLIANT |
| R-AOB-003 | S-AOB-011 | discharged by defeat D (3 mutations, 15/15 rows) — supplied by verify, not recorded by apply | COMPLIANT (verify evidence) |
| R-AOB-004 | S-AOB-012 | `ai37_attributes_test.go:109-127` subset check | COMPLIANT |
| R-AOB-004 | S-AOB-013 | `:129-228` per-key exact-value subtests | COMPLIANT |
| R-AOB-004 | S-AOB-014 | defeat E row 1 | COMPLIANT |
| R-AOB-004 | S-AOB-015 | **1 of 5 rows.** Only the unretried-success row asserts `retry.count`, and only `== 0`. Hard-wiring `recordRetryCount` to `0` leaves the **whole module suite green**; deleting `recordRetryCount` from `endSpanPreHandover` also leaves the **whole module suite green** | ❌ **UNTESTED** |
| R-AOB-004 | S-AOB-016 | `TestAI37_Attributes_PresenceTyped:234-269`, with a present-key control | COMPLIANT |
| R-AOB-004 | S-AOB-017 | usage chunk carries `reasoning_tokens: 10`; every one of the 12 keys has an exact-value assertion and a 13th key fails the subset check | COMPLIANT |
| R-AOB-004 | S-AOB-018 | `TestAI37_Attributes_CrossVendorSystemEquality:329-366` | COMPLIANT |
| R-AOB-005 | S-AOB-019 | defeat E row 2 — an early span end is detected by the event-count equality | COMPLIANT |
| R-AOB-005 | S-AOB-020 | `ai37_attributes_test.go:170-189` four usage counts vs the drained completion | COMPLIANT |
| R-AOB-005 | S-AOB-021 | `:211-222`, with a non-empty guard | COMPLIANT |
| R-AOB-006 | S-AOB-022 | `TestAI37_Attributes_TerminalFailureOmitsStatusCode:302-304`; note at the omission site, `trace.go:115-121` | COMPLIANT |
| R-AOB-006 | S-AOB-023 | `:305-315` — scans every `INT64` attribute for the bare class value `4` | COMPLIANT |
| R-AOB-006 | S-AOB-024 | follow-up owner `ai-provider-errors` named at `proposal.md:98`, `spec.md:48`, `specs/ai-observability-boundary/spec.md:214` — all committed | COMPLIANT |
| R-AOB-006 | S-AOB-025 | `:317-323` — `error.type` equals `failure.Category().String()` | COMPLIANT |
| R-AOB-007 | S-AOB-026 | defeat A — 8/8 corpus channels | COMPLIANT |
| R-AOB-007 | S-AOB-027 | rendering goes through `attribute.Value.String()`, never `%#v`; defeat B rows 5/5b prove the walk collapses without it. No *pointer-shaped-field* marker is planted — that shape is not constructible in this API surface | PARTIAL |
| R-AOB-007 | S-AOB-028 | no AI-37-specific enumeration test. Covered by landed guards: `TestConfig_SurfaceIsEndpointCredentialAndOptionalClient` (exactly 4 fields, none a content switch), `ambient_authority_test.go` (no `os.Getenv` in non-test sources), no build tag in the `ai` tree | PARTIAL |
| R-AOB-007 | S-AOB-029 | every RED above named the vector and reprinted **no** matched bytes. But the message names the vector **only** — not the span and not the attribute key, which R-AOB-007 requires; `sweep.Scan` returns no such context and `Corpus()` is flattened | PARTIAL |
| R-AOB-008 | S-AOB-030 | defeat B guard 1 | COMPLIANT |
| R-AOB-008 | S-AOB-031 | defeat B guards 2 and 2b | COMPLIANT |
| R-AOB-008 | S-AOB-032 | defeat B guard 3 bites for the 4 asserted kinds. Narrowed from 5 to 4 (deviation 9) — see the deviation verdicts below | PARTIAL |
| R-AOB-008 | S-AOB-033 | apply's recorded overlay RED; independently reproduced (defeat A#1) | COMPLIANT |
| R-AOB-008 | S-AOB-034 | `tracetest_test.go:204-250`; defeat B rows 5/5b | COMPLIANT |
| R-AOB-008 | S-AOB-035 | all 7 canaries assembled at runtime; `grep -r` for each full string over the whole tree returns zero hits; the landed repo-wide `credential_scan_test.go` sweep is green | COMPLIANT |
| R-AOB-009 | S-AOB-036 | `agenttest.RequireSameEvents` element-for-element, with a non-empty guard (`:72-75`) | COMPLIANT |
| R-AOB-009 | S-AOB-037 | **14 of 15** shapes. `pre_handover_cancelled_in_loop` — a cancellation shape — is absent, although the file's own comment claims 15 | PARTIAL |
| R-AOB-009 | S-AOB-038 | `:306-319` over `client.go`, `stream.go`, `trace.go`; verified no other non-test Layer 1 file imports otel | COMPLIANT |
| R-AOB-009 | S-AOB-039 | `:328-355` staged `t.TempDir()` mutation, asserts the scanner flags `span != nil` | COMPLIANT |
| NFR-AOB-001 | — | S-AOB-036/037 are its behavioural evidence; `run` gains no goroutine and no allocation on the carrier path | COMPLIANT |
| NFR-AOB-002 | — | 12 literal `const`s at `trace.go:36-49`, each spelled once, referenced from every site; no key assembled at a recording site | COMPLIANT |

**Twelve-key spelling audit** — `trace.go:37-48` against ADR 0005 § D3 (`docs/adr/0005-…md`, Attribute
allowlist paragraph): `gen_ai.system`, `gen_ai.request.model`, `gen_ai.request.max_tokens`,
`gen_ai.response.finish_reasons`, `gen_ai.usage.input_tokens`, `gen_ai.usage.output_tokens`,
`gen_ai.usage.cache_read_tokens`, `gen_ai.usage.cache_write_tokens`, `http.response.status_code`,
`retry.count`, `stream.event_count`, `error.type` — **12/12 character-exact**. No thirteenth key is
set anywhere (subset check + defeat E row 1).

#### `agent-module-scaffold` — R-AGM-001, R-AGM-003, R-AGM-005 (MODIFIED), R-AGM-008 (ADDED)

| Requirement | Scenario | Evidence | Result |
|---|---|---|---|
| R-AGM-001 | S-AGM-001 | `go.mod` = `otel v1.44.0` + `otel/trace v1.44.0` direct, `xxhash/v2 v2.3.0` indirect, zero `replace`; set-equality pin at `import_boundary_test.go:286-304` | COMPLIANT |
| R-AGM-001 | S-AGM-002 | `git ls-files backend/agent/go.sum` → tracked; `git check-ignore` → exit 1 (not ignored) | COMPLIANT |
| R-AGM-001 | S-AGM-003 | `make build` exit 0; `go.mod` unchanged afterwards | COMPLIANT |
| R-AGM-001 | S-AGM-004 | `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`; defeat C (4 shapes) | COMPLIANT |
| R-AGM-003 | S-AGM-020 | `go.work` `use` block names exactly the three modules. Its `go` directive is `1.26.5`, not the `1.26.3` the scenario restates — **pre-existing**, unchanged by AI-37 (see SUGGESTION 4) | COMPLIANT (for this change) |
| R-AGM-003 | S-AGM-021 | both siblings build independently (apply-recorded; `go.work` untouched by this change) | COMPLIANT |
| R-AGM-003 | S-AGM-022 | `A go.work.sum` in `git diff --name-status origin/main...HEAD` | COMPLIANT |
| R-AGM-003 | S-AGM-023 | `.gitignore` diff removes the `go.work.sum` line and records why; `git check-ignore go.work.sum backend/agent/go.sum` → exit 1; tree clean after a full `-race` run | COMPLIANT |
| R-AGM-003 | S-AGM-068 | workspace-sum file in the diff; require set asserted via `go mod edit -json` (workspace-immune), independently of the `go list` build list | COMPLIANT |
| R-AGM-005 | S-AGM-040 | guard green | COMPLIANT |
| R-AGM-005 | S-AGM-041 | `listNonStdlibDeps:349-351` passes `-deps -test`, pattern `modulePath + "/..."` | COMPLIANT |
| R-AGM-005 | S-AGM-042 | stdlib via `{{if not .Standard}}`; third-party group non-empty and per-entry annotated with ADR 0005 § D3, with closure-only reasoning stated in place for `semconv` and `xxhash` | COMPLIANT |
| R-AGM-005 | S-AGM-043 | all 8 forbidden prefixes present and byte-unchanged vs `origin/main`; no OTel API path among them | COMPLIANT |
| R-AGM-005 | S-AGM-044/045/046 | defeat C — all three bites reproduced independently, with no `go.mod` mutation | COMPLIANT |
| R-AGM-005 | S-AGM-047 | diff carries no scratch file; `go.mod` declares only ADR-authorised requires | COMPLIANT |
| R-AGM-005 | S-AGM-048 | `normalizeListedPackage:389-401` strips ` [pkg.test]` and drops `pkg.test` | COMPLIANT |
| R-AGM-008 | S-AGM-069 | `import_boundary_test.go:306-322` set equality over the closure | COMPLIANT |
| R-AGM-008 | S-AGM-070 | defeat C — `go.opentelemetry.io/otel/semconv/v1.37.0` is prefix-admitted yet closure-rejected; the pin FAILED and named it | COMPLIANT |
| R-AGM-008 | S-AGM-071 | `wantExternalClosure:194-205` contains no sdk/exporter/metric/baggage/propagation/auto path | COMPLIANT |
| R-AGM-008 | S-AGM-072 | `go.opentelemetry.io/otel` is a **direct require** yet its root package is **absent** from the closure — both pins pass, disagreeing by design | COMPLIANT |
| R-AGM-008 | S-AGM-073 | `:166-178` and `:271-285` state the replacement of the zero-require assertion, cite the ADR, and explain equality-vs-prefix | COMPLIANT |

#### `ai-provider-client` — R-APC-001, R-APC-003 (MODIFIED), R-APC-016 (ADDED)

| Requirement | Scenario | Evidence | Result |
|---|---|---|---|
| R-APC-001 | S-APC-001…005 | landed; re-run green under the full `-race` suite | COMPLIANT |
| R-APC-001 | S-APC-081 | `TestAI37_Attributes_TwoAdaptersDiscriminateProviders:373-411`; falsifier proven by defeat E row 3 — no field read anywhere | COMPLIANT |
| R-APC-003 | S-APC-011…014, 016 | landed; re-run green | COMPLIANT |
| R-APC-003 | S-APC-015 | `client_test.go:372-406` — exactly 4 fields, `TracerProvider` an interface, no Timeout/Deadline/Bound/Duration name fragment | COMPLIANT |
| R-APC-016 | S-APC-082 | `TestAI37_Attributes_NoProviderInjected_UsesNoOpDefault:421-441` | COMPLIANT |
| R-APC-016 | S-APC-083 | **structurally unconstructible**: no process-global telemetry registry is reachable from Layer 1 (root `otel` is not imported and is rejected by the guard — defeat C), so no globally installed provider can be observed recording nothing. The shipped test documents this | PARTIAL |
| R-APC-016 | S-APC-084 | injected provider records the span; identity unchanged; used verbatim (no wrapper in `client.go:141-147`) | COMPLIANT |

**Compliance summary**: **68/77 scenarios COMPLIANT**, 8 PARTIAL, 1 UNTESTED.

---

### Deviation verdicts (the three the orchestrator asked for by name)

#### Deviation 9 (C7) — reasoning event-coverage narrowed from 5 kinds to 4

**The landed claim is verified independently and it holds.** `wireDelta` (`chunk.go`) declares exactly
two fields, `content` and `tool_calls` — no `reasoning_content`, no cognate. `decodeChunk` uses a
plain `json.Unmarshal` with no `DisallowUnknownFields`, so an unknown `reasoning_content` key is
dropped before any Go value exists to carry it. `grep -rn "EventKindReasoningDelta"` over the adapter
finds **no producer** — only absence assertions (`reasoning_absence_test.go:110`) and the
openrouter charter guard that forbids one. `decision.md` (AI-29.0) records the verdict with a
five-mechanism table. **This adapter genuinely cannot construct a reasoning event.** The literal
wording of R-AOB-008 item 3 is therefore unsatisfiable without fabricating a wire shape the adapter
does not speak, and the narrowing is the correct resolution.

**But apply's characterisation is overstated.** Apply calls the retained reasoning canary "a strictly
stronger claim". It is not stronger — it is *non-falsifiable*. I could defeat seven other vectors by
overlaying a leak into production code; I could construct **no** overlay that leaks the reasoning
canary, because there is no Go value anywhere in the adapter that holds it. Its absence from the
corpus is guaranteed by construction and therefore proves nothing about the span-recording layer.
The honest statement is: *the reasoning vector is a structurally-guaranteed pass, retained for
documentation value; the span-layer denylist claim for reasoning text rests on the wire-decode drop,
not on the corpus scan.* That distinction should be recorded at archive. **WARNING, not CRITICAL** —
R-AOB-007's substantive prohibition is satisfied; only the evidentiary framing is inflated.

#### Deviation on C5's RED — `git stash` instead of `go test -overlay`

**The RED is non-vacuous.** The captured failure is
`ai37_span_lifecycle_test.go:37:3: unknown field TracerProvider in struct literal of type openaicompat.Config`
— a compile error that can only be produced by a tree in which the field does not exist. That claim
shape ("this field does not exist yet") is genuinely different from the overlay's shape ("this
defect variant should be caught"), and `git stash push -u` / `git stash pop` computes no restore path
by hand, which is precisely the failure mode the standing overlay instruction exists to prevent. The
restore was verified byte-identical by an immediately following green `make test`/`make lint`/`make build`
and a clean `git status`. **Accepted.** The same phase's real behavioural claims (the 15-path table)
were re-proven here by overlay anyway — defeat D.

#### C1/C3 structural merge — dependency and its first real importer in one commit

**The reasoning holds and is empirically re-checked.** Go's module-graph pruning cannot resolve to
the exact three-entry require table until a package in the module actually imports the subpackages;
`go get` with no importer pulls the full graph (`go-logr/*`, `auto/sdk`, `otel/metric`), and
`go mod tidy` with no importer prunes back to zero requires. My own root-otel overlay reproduces the
first half of that exactly: importing the ecosystem root drags 24 non-stdlib packages including
`auto/sdk` and `otel/metric` into the closure. An exact require-set pin and a set-equality closure
pin therefore have nothing to assert against until a real importer exists. `tracetest.go` is the
minimal such importer — of exactly the four allowlisted packages, no more. **Accepted.**

#### `parseAllowedNonStdlibPrefixes` — was the bug real, is the fix correct and in scope?

**Real.** At `origin/main`, `TestOpenRouterAdapter_AllowedNonStdlibPrefixesHoldsExactlyOneEntry`'s own
doc comment promised element resolution "by following const declarations in the same file … **or,
for a literal string, by reading the literal directly**", while the element walk handled only
`*ast.Ident` and silently `continue`d past every `*ast.BasicLit`. `unquoteStringLiteral` already
existed and was already used elsewhere in the same file, so only the dispatch was missing.
**Correct.** The fix adds a `case *ast.BasicLit` that unquotes `token.STRING` values and appends `""`
for anything else, preserving the existing "surface an empty entry so the assertion fails loudly"
discipline. **In scope.** AI-37's five new allowlist entries are literal strings; without the fix the
parser returns one entry and the new set-equality assertion cannot pass. Reverting the fix under
overlay produces `allowedNonStdlibPrefixes = [github.com/cachicamas/backend/agent     ], want […6…]`
— the fix bites.

---

### Milestone subnode coverage (AI-37.1 … AI-37.4)

| Subnode | Coverage | Non-vacuity proven by |
|---|---|---|
| **AI-37.1** — dependency + guard + bite proof | real | defeat C: 4 independent scratch-import shapes, closure-equality failure naming 24 vs 10 packages, `semconv/v1.37.0` prefix-admitted-yet-closure-rejected |
| **AI-37.2** — span seam + attribute mapping | real, one gap | defeat D (15/15 rows), defeat E rows 1–2. **Gap**: `retry.count` (see CRITICAL) |
| **AI-37.3** — denylist absence + non-vacuity guards | real | defeat A (8/8 channels), defeat B (all 4 guards + the captured-field-count guard) |
| **AI-37.4** — no-op equivalence | real, one gap | `RequireSameEvents` element-for-element; nil-check scan with a shipped staged-mutation falsifier. **Gap**: 14 of 15 no-tracer terminal rows |

---

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD evidence reported | ✅ | Inline per-phase RED/GREEN in `tasks.md`, summarised in `apply-progress.md` |
| All tasks have tests | ✅ | 21/21; 4 new AI-37 test files + `tracetest_test.go` + amended guards |
| RED confirmed (test files exist) | ✅ | 9/9 phases; every cited test file present |
| GREEN confirmed (tests pass) | ✅ | `make test` exit 0, 9/9 packages `ok` under `-race` |
| Triangulation adequate | ⚠️ | Strong for span lifecycle (15 rows) and attributes (12 keys). **Single-case for `retry.count`** where the spec names five |
| Safety net for modified files | ✅ | 4 pre-existing zero-require pins updated in the same commit, as each one's own doc comment instructed |

**TDD compliance**: 5/6 checks passed.

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit | 8 | `src/agenttest/tracetest/tracetest_test.go` | `testing`, `reflect` |
| Integration (in-process HTTP) | 13 funcs / 43 subtests | 4 × `src/ai/openaicompat/ai37_*_test.go` | `net/http/httptest`, `tracetest` |
| Build-graph guard | 3 | `src/ai/import_boundary_test.go` | `go list -deps -test`, `go mod edit -json` |
| E2E | 0 | — | not installed / not applicable at Layer 1 |

### Assertion Quality

No tautologies, no orphan empty checks, no ghost loops, no smoke-only tests. Vacuity controls are
present where they matter: `len(eventsPlain) == 0` → `t.Fatal` (`ai37_noop_equivalence_test.go:72`),
`len(events) == 0` → `t.Fatal` (`ai37_attributes_test.go:219`), empty-needle check
(`ai37_denylist_test.go:276-280`), `len(deps) == 0` → `t.Fatal` (`import_boundary_test.go:141`),
present-key control beside the two absence assertions (`ai37_attributes_test.go:266`).

**Assertion quality**: ✅ all assertions verify real behaviour. **0 CRITICAL, 0 WARNING.**

### Quality Metrics

**Linter**: ✅ `make lint` → `0 issues.`
**Build**: ✅ `make build` → exit 0
**Race detector**: ✅ 9/9 packages `ok` under `-race`

---

### Issues Found

#### CRITICAL (1) — blocks archive

**C-1 — `retry.count` is recorded but effectively unasserted; `S-AOB-015` is UNTESTED and `S-AOB-009` is PARTIAL.**

R-AOB-004 states: *"`retry.count` — the number of **retries** … A value MUST be available on **every**
exit of the retry mechanism, not only on the exit that exhausts the budget."* S-AOB-015 names five
rows. The shipped suite asserts **one** — the unretried success, and only that it equals `0`.

Two independent overlay mutations, each run against the **whole module**:

```text
# retry.count hard-wired to 0 on every exit
go test -count=1 -timeout 500s -overlay=<scratch>/overlay_retry0.json ./...
→ exit 0, zero failures

# recordRetryCount deleted from endSpanPreHandover (both pre-handover exits)
go test -count=1 -timeout 500s -overlay=<scratch>/overlay_nopreretry.json ./...
→ exit 0, zero failures
```

Neither the **value** on any retried path nor the **presence** on either pre-handover exit is
detected by any test in the module. The implementation is correct by inspection
(`stream.go:230, 244, 247` and `trace.go:104-106, 122-123`), and `retry.Loop`'s own return value is
proven by `TestLoop_RetryCountOnEveryExit` — but the span-attribute half has no falsifier, so a
regression on any of those four paths ships silently. That is exactly the failure class this
milestone's Wave-5 charter exists to close.

The same missing fixture (a server that fails retryably N times then succeeds, driven under a
recording tracer) also blocks `S-AOB-009`'s "retried more than once before succeeding" row.

**Remediation** (small, additive, test-only): add a fail-then-succeed `httptest` fixture and a
retry-outcome table asserting `retry.count` on all five rows named by S-AOB-015 — unretried success
(`0`), retried-twice-then-success (`2`), non-retryable terminal, budget-exhausted, cancelled. That
one table discharges S-AOB-015 fully and S-AOB-009's traced-retry row.

#### WARNING (6)

**W-1 — `S-AOB-029` failure output names the vector only.** R-AOB-007 requires the failure to name
*"the vector, the span and the attribute key"*. `ai37_denylist_test.go:270` emits only the vector, and
its own message says so. `sweep.Scan` returns no location and `Corpus()` is a flattened byte string,
so the span and key are not recoverable at the failure site. Either narrow the requirement at archive
or make the corpus carry per-span/per-key provenance.

**W-2 — Deviation 9's "strictly stronger" framing is inflated.** The retained reasoning canary is
non-falsifiable by construction (no overlay can make it appear, because no Go value holds it). The
narrowing itself is correct and forced; only the evidentiary claim needs restating. See the deviation
verdict above.

**W-3 — `S-AOB-037` covers 14 of 15 terminal shapes with no tracer.**
`ai37_noop_equivalence_test.go:79-81` claims "every one of the 15 terminal-path shapes"; the table has
14 rows. `pre_handover_cancelled_in_loop` — a cancellation shape the scenario names explicitly — is
absent. Add the row or correct the comment.

**W-4 — `S-AOB-011` shipped with no recorded falsifier.** Apply's Phase 5 record contains no mutation
run for the uniform-close claim, although the scenario requires one. Verify supplied it (defeat D,
three mutations, all 15 rows). The evidence now exists in this report; it should be carried into the
archive record rather than re-discovered.

**W-5 — `S-AOB-009`'s duration clause is not assertable.** `tracetest.Span` records no start or end
timestamp, so "its recorded duration encloses every attempt" cannot be checked with the shipped
recorder. The span-start position (`stream.go:222`, before `retry.Loop` at `:224`) is structurally
correct. Either add timestamps to the recorder or narrow the scenario at archive.

**W-6 — `apply-progress.md`'s diff stat is stale.** It records `29 files changed, 4503 insertions(+)`;
the shipped range is `30 files, 4576 insertions(+), 147 deletions(-)` — its own commit `07a5af82` is
not counted. Cosmetic, but it is the number a reviewer reads first.

#### SUGGESTION (6)

**S-1 — `S-AOB-003`**: add one clause to `import_boundary_test.go:99-107` stating the spec's own
reason for declining the root global getter — *its package closure contains `go.opentelemetry.io/auto/sdk`,
a path literally named for the SDK*. The measurement is recorded in Engram #2702 and in R-AOB-001;
it is not at the guard site, which is where the scenario asks a reader to find it.

**S-2 — `S-AOB-027` / `attribute.Value.Emit`**: the spec's phrasing follows the AI-41 lesson, but
`Emit()` is deprecated at otel v1.44.0 and `String()` is the API's current accessor. The substance
(render through the API, never a struct dump) is satisfied and falsifiable (defeat B rows 5/5b).
Record the accessor substitution at archive so the next reader does not re-litigate it.

**S-3 — `S-AOB-028` and `S-APC-083` are structurally unconstructible** at Layer 1 once the root
global-getter package is excluded: there is no process-global registry to install into, and no
content-capture switch exists to enumerate. Record them as *satisfied by construction* at archive
rather than leaving them looking untested.

**S-4 — pre-existing drift, follow-up not blocker**: canonical `agent-module-scaffold/spec.md:70`
(S-AGM-020) asserts `go.work` declares `go 1.26.3`; the shipped `go.work` declares `go 1.26.5`, and it
is byte-unchanged by AI-37 (`git diff origin/main...HEAD -- go.work` is empty). No test asserts the
directive, which is why it never surfaced. The AI-37 delta restated the scenario verbatim, as a
`MODIFIED` block is required to. Owner: `agent-module-scaffold`.

**S-5 — spec index arithmetic**: `spec.md:18` says "14 requirements touched" while its own breakdown
(9 new + 2 added + 5 modified) sums to 16; it says 29 restated scenarios where the delta files carry
28 (17 + 11). Correct at archive so the canonical counts are right.

**S-6 — `runNoPanic`'s `recover()`** (`ai37_noop_equivalence_test.go:98-106`) only covers the calling
goroutine. A panic inside `run`'s producer goroutine would crash the test binary rather than fail the
row with S-AOB-037's message. The rows still exercise the paths; only the diagnostic would be worse.

---

### Coherence (Design)

| Decision | Followed? | Notes |
|---|---|---|
| AD-1 — otel v1.44.0 measured under `go.work` | ✅ | exact 3-entry require set, verified |
| AD-2 — `go mod edit -json`, workspace-immune | ✅ | `import_boundary_test.go:250-269` |
| AD-3 — embed `noop.*` in the recorder | ✅ | `tracetest.go:66, 172, 204` |
| AD-4 — method-count pin | ✅ | `tracetest_test.go:165-178` (Span 11, Tracer 2, Provider 2) |
| AD-5 — one uniform closing site | ✅ | `finalizeSpan` + `endSpanPreHandover`; defeat D proves both |
| AD-6 — `retry.count` / `stream.event_count` carriers | ⚠️ | carriers implemented; `retry.count` unasserted (C-1) |
| AD-7 — `gen_ai.system` is the dialect | ✅ | `trace.go:62`; `S-AOB-018` |
| AD-8 — constant span name, `SpanKindClient` | ✅ | `trace.go:55, 79` |

---

### Verdict

**FAIL** — one CRITICAL: `S-AOB-015` is UNTESTED and `S-AOB-009` PARTIAL, proven by two whole-module
overlay runs in which `retry.count` was hard-wired to `0` and then deleted from both pre-handover
exits without a single test failing. Everything else in this change is exceptionally well proven:
8/8 denylist corpus channels, all four non-vacuity guards, all 15 span-lifecycle terminal paths, and
all four dependency-guard bites were independently defeated during verification. The remediation is
one additive test table and touches no production code.

---

> **Historical record — superseded.** This report's FAIL verdict, and CRITICAL C-1 specifically, are
> superseded by `verify-report-final.md`. C-1 was fixed test-only in commit `3a61e46d` (the
> remediation round documented in `apply-progress.md`); both Judgment Day correction rounds then
> found and fixed two further CRITICALs (a structurally identical lost-send defect on two terminal
> paths) that this report's own defeat D did not surface, because defeat D asked whether `End()` is
> called, not whether the outcome classification feeding `finalizeSpan` is correct. This file is
> preserved verbatim as the audit-trail record of the pre-remediation state; it is not evidence of
> the change's final state. See `verify-report-final.md` for the archived, superseding verdict.
