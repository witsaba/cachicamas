# Exploration: cachicamas-ai-redaction (AI-36)

> **Milestone:** AI-36 — Enforce secret redaction (Wave 5 — Harden)
> **Doc 0002 charter:** `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 2150–2184 (+ amendment blockquote 2161, + AI-41 close amendment line 22)
> **Worktree:** `cachicamas-worktrees/ai-36-redaction`
> **Engram:** `sdd/cachicamas-ai-redaction/explore` (obs #2679)
> **Date:** 2026-08-07

> **Tooling note.** No `.codegraph/` index exists for this worktree and the explore-phase agent had no Bash tool to run `codegraph init`; the entire exploration fell back to Read/Glob/Grep. It also could not run `go test`, so every claim below marked *empirical* is unverified and is handed to propose/design as a RED-first obligation. The apply phase (which has Bash) should initialize CodeGraph in this worktree before large edits.

---

## Charter (verbatim citations)

`docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2150-2184`

- AI-36 goal / deliverable / acceptance at lines 2150–2159.
- Amendment blockquote (AI-41's `Blocks: AI-36` edge) at line 2161.
- AI-36.1 Sentinel sweep at 2163–2170 (depends on AI-32, AI-25; out of scope: AI-25.3 credential-attachment proof, AI-32.5 wire-error-body size bound).
- AI-36.2 Header/config redaction at 2172–2177 (depends on AI-36.1).
- AI-36.3 Failure-output hygiene at 2179–2184 (depends on AI-36.1).
- AI-41 close amendment (the inherited follow-up) at line 22 of the same doc.
- Doc-wide authoring constraint (behavior-only, no Go identifiers) at lines 24–25.

---

## 1. Current redaction surface — what is already redacted vs. not

**Already redacted** (value-receiver or otherwise total, proven by existing tests):

| Surface | Location | Why it is safe |
|---|---|---|
| `ai.FailureCategory.String()` | `src/ai/provider_failure.go:134` | Safe placeholder for out-of-vocabulary values. |
| `*ai.Failure.Error()` | `provider_failure.go:349` | Fixed prefix + category name only; never the cause, RawLabel, StatusClass, RequestID (R-AIP-009; proven S-AIP-028…031). |
| `*ai.Failure.GoString()` | `provider_failure.go:381` | Delegates to `Error()` (AI-41.2, R-AIP-016). **Pointer-receiver only** — see § 2. |
| `ai.Part.String()` / `GoString()` | `src/ai/content_part.go:290,303` | **Value receiver**; renders `part(<kind>)` only, never the payload. Its comment names the credential-shaped risk ("a prefix of a secret is still a secret"). |
| `openaicompat.Credential.String()` / `GoString()` | `openaicompat/credential.go:31,41` | **Value receiver**, always `"<redacted>"`. No `MarshalJSON`/`MarshalText` needed: the `token` field is unexported with no exported counterpart, so `json.Marshal` already yields `{}`. Proven by `credential_test.go:19` `TestCredential_NeverRendersRawToken` (`%v`, `%s`, `%+v`, `%#v`, `json.Marshal`). |
| `openrouter.redactedProvider.String()` / `GoString()` | `openaicompat/openrouter/wrapper.go:90,99` | Value receiver, fixed label `"<openrouter provider>"`, added specifically to stop default `%+v`/`%#v` descending into the embedded `*openaicompat.Client`'s unexported `credential` field. Proven by `credential_redaction_test.go:72`. |
| `capturedBody.Error()` | `openaicompat/capture.go:56` | Fixed text; captured bytes reachable only via the unexported `bytes()` accessor (R-AEM-016). |
| `*RateLimitTelemetry.Error()` | `openaicompat/retry_metadata.go:53` | Fixed text, never a header name or value; capture is a **3-name allowlist** (`captureRateLimitTelemetry`, `retry_metadata.go:69-84`), never the full `http.Header`. |
| `*inBandErrorFrame.Error()` | `openaicompat/stream_failure.go:51` | Fixed text; payload retained in an unexported field, never rendered (R-AEM-010/016). |
| `event.go` types (`Event`, `TextDelta`, `ToolCallDelta`, …) | `src/ai/` | Value-receiver `String()`/`GoString()`, kind + sequence only, payload-free. |
| `openaicompat.Config` (exported `Credential` field) | `openaicompat/client.go:43-58` | Safe *by construction* through `Credential`'s own value-receiver methods — but **only tested at the `openrouter.Config` level** (`credential_redaction_test.go:24-60`), never at the `openaicompat.Config` level. |

**NOT yet redacted / untested — the real gap:**

- **`nonStreamContentType.Error()`** — `openaicompat/stream.go:326-328` — **directly interpolates the captured response-body excerpt into the error string**. This is the one place in the tree that renders raw captured bytes straight into `Error()` text. The size bound is explicitly out of scope (AI-32.5), but **sentinel-absence on this exact path is squarely AI-36.1's charter**. The comment at `stream.go:294-318` discloses that the AI-26.1 credential scan deliberately excludes internal-package test files because this milestone's own tests plant above-threshold credential-shaped sentinels there on purpose — a load-bearing constraint for AI-36.3.2 (see § 5, § 6, § 10).
- **`openaicompat.Client`** — bare struct, no `String()`/`GoString()` at all (confirmed by grep; `client.go:63-67`). Its `credential Credential` field is **unexported**. See § 2.

---

## 2. The value-vs-pointer receiver gap

Grepped every `String()`/`GoString()`/`Format()`/`MarshalJSON` in `backend/agent/src/ai/**` (~40 hits).

**Pointer-receiver (the AI-41-flagged gap, confirmed still open):**

- `*ai.Failure` — `Error()`, `GoString()`, `Unwrap()`, `Is()` all pointer-receiver (`provider_failure.go:349,381,386,402`). A **copied `Failure` value** is neither a `fmt.GoStringer` nor an `error`; `%#v` on it falls back to reflection over every unexported field, `cause` included — exactly the finding the AI-41 close amendment hands to AI-36 (doc 0002 line 22). AI-41's own doc comment at `provider_failure.go:370-380` states this and explains *why* the methods stay pointer-receiver: a value-receiver `GoString` would make fmt dereference a typed-nil `*Failure` to build a value copy, trading NFR-AIP-B's total nil-safety for coverage no call site needs. Every constructor (`PreStreamFailure`, `MidStreamFailure`) returns `*Failure`; **no production path is known to hold a bare `Failure` value**. The design phase must decide between proving no such value escapes and adding narrow additional coverage.
- `*RateLimitTelemetry.Error()`/`Unwrap()` (`retry_metadata.go:53,58`) — same shape in principle, but only ever constructed via `captureRateLimitTelemetry` returning a pointer, and its three fields are plain non-secret strings.
- `*inBandErrorFrame`, `capturedBody` (value receiver — `capture.go:56`), `*nonStreamContentType`, `*malformedToolCallAssembly`, `*AttemptReport` — all unexported/internal types never returned as bare values to a caller who could copy them; pointer-vs-value is moot.

**Value-receiver (already covers copies correctly):** `ai.Part`, `openaicompat.Credential`, `openrouter.redactedProvider`, and `ai.Event`, `ai.Completion`, `ai.TextDelta`, `ai.ToolCallDelta`, `ai.ReasoningDelta`, `ai.ResponseStart`, `ai.ToolResult`, `ai.Reasoning`, `ai.SystemInstruction`, `ai.Segment`, `ai.ProviderExtension` — all payload-free renderings.

**Conclusion.** `*ai.Failure` is the *only* credential/content-adjacent type in the redaction surface with a live, acknowledged, still-open value-copy gap. It is AI-36's inherited, explicitly-named follow-up and deserves its own work unit.

**New finding (*empirical* — needs one quick Go run before it is treated as proven).** `openaicompat.Client` (`client.go:63-67`) has no `String()`/`GoString()`, and its `credential Credential` field is **unexported**. Per Go's documented `fmt`/`reflect` behavior, a value reached by reflecting into an *unexported* struct field cannot have `.Interface()` called on it (`reflect.Value.CanInterface()` is false), so fmt's recursive struct printer cannot invoke `Credential.String()`/`GoString()` on that field and falls back to raw reflection — which, for a string field, prints the raw bytes. `fmt.Sprintf("%+v", client)` or `%#v` on a **bare, unwrapped** `openaicompat.Client` would therefore very likely print the raw bearer token, bypassing `Credential`'s own redaction. **No current production path formats a bare `*openaicompat.Client`** (it is always wrapped in `redactedProvider`, whose `String`/`GoString` short-circuit at depth 0 before reaching the embedded client's fields) — so this is **latent, not an active leak**, but it is **completely untested** (no test in `client_test.go` or elsewhere formats a bare `Client`/`*Client`). Open question for the proposal: is a defense-in-depth `Client.GoString()`/`String()` in scope for AI-36.2, or does "the adapter's configuration value" in the charter mean `Config` only (already safe by construction)?

---

## 3. Credential plumbing (AI-25 work), traced end to end

- `openaicompat.Credential` — opaque wrapper, unexported `token string` — `credential.go:7-9`.
- `openaicompat.Config.Credential` — **exported** field — `client.go:43-58` (line 50).
- `openaicompat.Client.credential` — **unexported** field, copied from `Config.Credential` at construction — `client.go:63-67,90-94`.
- Transport attachment point (the one place the raw token becomes a header value): `Credential.bearer()` — `credential.go:24-26` — called exactly once, at `request.go:35`: `req.Header.Set("Authorization", c.credential.bearer())`.
- `openrouter.Config.Credential` — `wrapper.go:29` — passed straight through to `openaicompat.New` at `wrapper.go:139`.
- No other struct transitively holds the credential; `RateLimitTelemetry`, `capturedBody` and `Failure` never receive it.

---

## 4. Prompt-body plumbing — every path content can reach a diagnostic

- **Bounded-excerpt path (AI-22.2)** — `agenttest/stream_kit_diff.go`: `RequireSameEvents` (line 27), `summarize`/`summaryTable` (91–147, one entry per `ai.EventKind`), `boundedFragment` (168) caps every free-form fragment (text delta, tool-call args, reasoning token, RawLabel) to `summaryRuneCap = 32` runes in a `len=N head=%q…` shape. This is the sanctioned, already-redaction-aware summarization AI-36.3.1 must **reuse**, not reinvent.
- **Wire-error-body path (AI-32.5 out of scope for size, in scope for sentinel-absence per AI-36.1)** — `capture.go:94-125` `captureBody`: 8 KiB cap (`captureLimit`, `capture.go:31`) + `truncationMarker`; retained bytes reachable only via the unexported `bytes()` — **except** `nonStreamContentType.Error()` (`stream.go:326-328`), the one exception that interpolates the excerpt directly. This is the single most load-bearing finding for AI-36.1's scope.
- **In-band error frame path** — `stream_failure.go:72-91` `failureFromErrorFrame` retains `payload []byte` in an unexported field, never rendered by `Error()`.
- **Conformance redaction case (the general shape already built)** — `agenttest/conformance_redaction.go`, registered as case `"redaction/sentinel_absent_from_every_rendering"` (`init()`, line 30): plants a sentinel into `FailureReport.Cause` and `RequestID` (never `RawLabel`, deliberately — header comment lines 1–16), scans every drained event via `scanForSentinel` (line 54) and the suite's own `RequireSameEvents` divergence text via a `capturingTB` double (lines 87–156).

---

## 5. What already exists to reuse

- **`agenttest/conformance_redaction.go`** (AI-23.7, R-CNF-013) — closest analogue to AI-36.1's reusable sweep: `scanForSentinel(events []ai.Event, sentinel string) (bool, string)` (line 54) and `scanTextForSentinel(text, sentinel string) bool` (line 71) are already pure and already redaction-aware (never reprint the sentinel), and exercise both the event-rendering path and the suite's own diff-reporting path. Currently **unexported** and Factory-scoped, not a general-purpose helper.
- **`openrouter/smoke/sentinel_sweep.go`** — a near-duplicate, independently-implemented deny-list scanner (`Scan`, `BuildDenyList`, `DenyEntry`, lines 74–146) for the live smoke test's captured log/stderr buffers. Same "runtime-built needle, never a contiguous literal" trick as `credential_scan_test.go`. Explicitly the "one-off" AI-36.1's test 3 should fold into a shared helper rather than duplicate a third time.
- **`credential_scan_test.go`** (AI-26.1, R-ART-004) — `scanCredentialSurface(dir string)` (line 66): a byte-pattern scan (`os.ReadDir`, **not** `filepath.Walk`) over `package openaicompat_test` files **directly inside one directory only**. Scoped to `src/ai/openaicompat/` alone — does **not** cover `openaicompat/openrouter/`, `openrouter/smoke/`, `openrouter/conformance/`, or its fixtures. AI-36.3.2's charter line is a literal scope-widening of this function, but must not break the deliberate internal-package exclusion (`externalTestPackageClause`, line 60) that `stream.go:309-318` documents.
- **`credential_test.go:19`** and **`credential_redaction_test.go`** — the verb-exhaustive pattern (`%v`, `%s`, `%+v`, `%#v`, `json.Marshal`) AI-36.2's config proof should replicate at the `openaicompat.Config` level.
- **`agenttest/stream_kit_diff.go`** (AI-22.2) — the bounded-excerpt machinery AI-36.3.1 must assert stays adversarially sentinel-free.
- **`ai/provider_failure_test.go:1284,1328`** — `TestFailure_GoString_RedactsLikeError`, `TestFailure_GoString_NilReceiver_TotalByDelegation` — AI-41's own proof and the direct precedent for the value-copy case.

---

## 6. Where a reusable sweep helper should live

- `backend/agent/src/ai/internal/retry/` (AI-35 precedent) is importable only by packages **under `backend/agent/src/ai/…`** (Go's `internal/` visibility rule). `openaicompat`, `openaicompat/openrouter`, `…/smoke`, `…/conformance` all qualify. **`backend/agent/src/agenttest/` does not** — it is a *sibling* of `src/ai/`, not nested inside it (confirmed against ADR 0005 § D2's location table, `docs/adr/0005-promote-agent-stack-to-own-module.md:198-218`), so it is outside a hypothetical `src/ai/internal/…`'s visibility.
- Since `agenttest` already hosts the closest analogue and per ADR 0005 § D2 is explicitly the "Layer 1 external-consumer proof" location — a plain, non-internal, exported package importable by any test package including `openrouter/smoke` — **`backend/agent/src/agenttest/` is the natural home**, generalizing the existing unexported pair into an exported API that both the conformance case and `openrouter/smoke`'s local `Scan`/`BuildDenyList` converge on, superseding the near-duplicate.
- Constraint: `agenttest` code is **not** `_test.go`-suffixed — it is real, importable Go source. The header comment on `conformance_redaction.go:75-86` discusses this Go build-boundary reason explicitly.
- The top-level import-boundary guard (`ai/import_boundary_test.go`, AI-00.3 — deny-by-default allowlist scoped to `github.com/cachicamas/backend/agent/…` + stdlib) does **not** constrain intra-module package layout; it fires only on cross-module or cross-layer imports. So `agenttest` vs. a new internal package is a design choice, not a guard-driven one.

---

## 7. Existing spec capabilities to extend — highest requirement ID today

| Spec file | Highest R-ID today | Next free |
|---|---|---|
| `openspec/specs/ai-provider-errors/spec.md` | `R-AIP-016` (AI-41.2) | `R-AIP-017` |
| `openspec/specs/ai-provider-conformance-suite/spec.md` | `R-CNF-019` (AI-35) | `R-CNF-020` |
| `openspec/specs/ai-stream-testkit/spec.md` | `R-STK-013` | `R-STK-014` |
| `openspec/specs/ai-stream-lifecycle/spec.md` | `R-AIS-044` (AI-35) | `R-AIS-045` |
| `openspec/specs/ai-provider-client/spec.md` | `R-APC-014` | `R-APC-015` |
| `openspec/specs/ai-openrouter-first-provider/spec.md` | `R-OR-10` | `R-OR-11` |
| `openspec/specs/ai-provider-error-mapping/spec.md` | `R-AEM-018` | `R-AEM-019` |

Likely touch points by sub-node:

- **AI-36.1** → `ai-provider-conformance-suite` (extends R-CNF-013's redaction case) and/or `ai-stream-testkit` (if the reusable helper is testkit-level).
- **AI-36.2** → `ai-provider-client` (Config/Credential) and `ai-openrouter-first-provider` (R-OR-08 already covers credential redaction extending to live smoke's logging — read it before adding).
- **AI-36.3** → `ai-stream-testkit` (bounded-excerpt enforcement) plus the spec owning AI-26.1's fixture scan. *(Located at propose time: `R-ART-004` lives in `openspec/specs/ai-request-translation/spec.md:107`, with scenarios `S-ART-013…017`; that spec's maxima are `R-ART-021` / `S-ART-089`.)*
- The `*ai.Failure` value-copy follow-up, if given a requirement, amends `ai-provider-errors` continuing from `R-AIP-016`.

---

## 8. The no-Go-identifier rule — confirmed and quoted

Doc 0002 states the authoring constraint at lines 24–25: *"This document states behaviors and what a test must prove. It never invents Go type names, field names, or signatures… From Wave 1 forward, this document cites shipped Layer 1 code as evidence for completed milestones, never for unshipped ones."*

The openspec spec files follow the same discipline. `openspec/specs/ai-provider-errors/spec.md:235-249` (R-AIP-016, AI-41's own landed requirement) uses **zero Go identifiers**: "the provider-failure payload", "Go-syntax representation", "Go-syntax verb", "the plain, string, extended and Go-syntax verbs" — never `*Failure`, `GoString()`, `%#v`, `%+v`. The AI-33 close amendment (doc 0002 line 16) independently confirms the convention is active and enforced. AI-36's spec amendments must follow the identical pattern.

---

## 9. Test-runner reality (commands only; not run in this phase)

All from `backend/agent/` (module root), per `backend/agent/Makefile`:

| Command | Expands to |
|---|---|
| `make test` | `go test -race -v ./...` |
| `make lint` | `go vet ./...` then `golangci-lint run --config=.golangci.yml ./...` (auto-installs v2.9.0 into `./bin` via `make tools`) |
| `make build` | `go build -trimpath ./...` |
| `make test/cover` | `go test -race -coverprofile=coverage.out -covermode=atomic ./...` |
| `make all` | `tidy fmt vet lint test build` |

The module carries **zero** external dependencies (`go.mod` has no `require`). AI-36 must not add one.

---

## 10. Risks and open questions the proposal must decide

**Evidence-based answer to "does AI-36 need production code changes, or is it purely adversarial tests?" — AI-36 needs at least one small, targeted production change, not purely tests.**

1. The **AI-41-inherited follow-up** (§ 2) is a known, named, still-open gap in production code, not a hypothesis. The AI-41 close text calls it "recorded as a follow-up owned by AI-36" and describes a real leak ("still leaks every unexported field under `%#v`"). AI-36 must either add coverage for the value-copy case or add a structural proof that no such value escapes — either way a proposal-level tradeoff.
2. **`nonStreamContentType.Error()`** (`stream.go:326-328`) is production code that renders raw captured bytes. Whether it needs a code fix or is provably safe as-is (the captured bytes are the *response*, not the *request*) is an open design question the proposal must resolve with a concrete adversarial test first; if the test bites, production code must change.
3. **`openaicompat.Client`'s bare-struct formatting gap** (§ 2) is genuinely open: is a defense-in-depth `String()`/`GoString()` in scope, given "the adapter's configuration value" likely means `Config`? Needs an empirical Go run before it is treated as proven; if proven, it is a ~4-line fix mirroring `redactedProvider`.
4. **AI-36.2 item 1** ("any diagnostic that captures headers redacts credential-bearing ones by default") is **currently satisfied structurally** — the only header-capturing diagnostic, `captureRateLimitTelemetry`, uses a 3-name allowlist and never the raw `http.Header`; no path captures or logs the full request header set. Likely provable by test alone, **unless** the design phase decides to add a general-purpose header redactor for AI-37 to inherit.
5. **Process gap carried over from AI-41** (doc 0002 line 22, "W-2"): a grep-based blast-radius method cannot see reflection-based guards. Any AI-36 change that adds an exported method must also grep for `reflect.TypeOf(<Type>` / `NumMethod` across the tree in addition to literal verb occurrences. This bit AI-41 itself via `S-AIP-008`'s `TerminalExclusivity` guard.
6. **AI-26.1 fixture-check widening** must preserve the documented internal-package exclusion (`stream.go:309-318`) — a naive recursive scan would break the intentional above-threshold credential plants. The design phase must locate the exact files carrying deliberate plants before writing the recursive scan.
7. **Delivery/size** — AI-33/34/35/41 all ran `size:exception` due to strict-TDD test density, and AI-36's charter is explicitly "adversarial by design: every test plants a sentinel and asserts absence" across a dozen distinct failure paths. Expect a similarly test-heavy diff; the session's review budget was pre-raised for this milestone.

---

## Recommendation

Proceed to `sdd-propose` with AI-36 split into at least three work units mirroring the charter's three leaves — AI-36.1 (sentinel sweep + reusable helper in `agenttest`), AI-36.2 (header/config redaction proof + possible client-level fix), AI-36.3 (failure-output hygiene + AI-26.1 scan widening) — plus explicit handling of the AI-41-inherited `*ai.Failure` value-copy follow-up as its own scoped decision. The proposal phase should resolve open questions 2 and 3 with quick empirical Go runs (this exploration had no Bash tool) before committing to a design.
