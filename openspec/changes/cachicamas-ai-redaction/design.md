# Design: Enforce secret redaction (AI-36)

> Change `cachicamas-ai-redaction` · builds on proposal rulings D-1…D-6 (Engram #2681) and explore (#2679).
> Delivery: **single PR**, `size:exception` accepted, budget **1600 changed lines** — the proposal's two-PR chain
> recommendation is **overruled by the maintainer** (session ruling; recorded per proposal § 4 preamble).
> Verification: `make test` (`-race`), `make lint`, `make build` from `backend/agent/`. `go.mod`/`go.sum` byte-identical.

## Technical Approach

One dependency-free scanner package becomes the single sweep implementation; every existing scanner delegates to it.
Every absence assertion ships a positive control built from the same mechanism. The two empirical unknowns (D-4, D-5)
are RED-first tests with both outcomes pre-designed. The deliberate-plant blanket exclusion becomes a marker plus a
committed, falsifiable allowlist over the inventory below.

## Architecture Decisions

### AD-1 — Sweep helper: package `backend/agent/src/agenttest/sweep`, does NOT collapse into `agenttest`

**Choice**: New package `sweep` (path `github.com/cachicamas/backend/agent/src/agenttest/sweep`), importing
`bytes` + `fmt` only. Exported surface:

```go
type Entry struct { Vector string; Needle []byte }
// Scan reports the first matching entry's Vector only — never the needle.
func Scan(corpus []byte, deny []Entry) (vector string, found bool)
// SelfTest plants each needle into a synthetic in-memory corpus and
// returns an error (naming the vector only) if Scan fails to detect it.
func SelfTest(deny []Entry) error
```

`Scan` returns detection facts, not a formatted error, so **each call site keeps its own message text verbatim**:

| Call site | Migration |
|---|---|
| `openrouter/smoke/sentinel_sweep.go` | `type DenyEntry = sweep.Entry` (type alias — literals in the eight R-OR-08 tests compile unchanged; verified no `%T`/reflection on `DenyEntry` anywhere in `smoke/`). `BuildDenyList` body unchanged. `Scan(captured, denyList) error` keeps its exact signature and its exact `"sentinel sweep detected credential leak: vector=%q"` message; internally calls `sweep.Scan`. |
| `agenttest/conformance_redaction.go` | `scanTextForSentinel` delegates to `sweep.Scan` with a one-entry deny list; `scanForSentinel` keeps its event-walking/`summarize` logic and message text, delegating the substring core. |
| `openaicompat/credential_scan_test.go` | Regex classes stay local (different mechanism: pattern classes, not fixed needles); it adopts the shared runtime-assembly discipline and the WU-5 walker (AD-4). |

**Collapse question closed**: `smoke.Scan` is exported from a **non-test** file, so delegation executes in the
non-test build — the collapse-into-`agenttest` condition (only-reached-from-`_test.go`) is false. Subpackage stands.
**Alternatives rejected**: `src/ai/internal/…` (excludes sibling `agenttest`, per Go `internal/` rule);
folding into `agenttest` (drags `testing`'s flag registration into `smoke`'s non-test build).
**Import legality verified**: `import_boundary_test.go` allowlists the whole module path; `src/agenttest` is not a
forbidden prefix (only `src/agent`, `src/coding`, `src/cmd` are).

### AD-2 — Positive controls: `sweep.SelfTest`, one line per call site

**Choice**: The control is part of the package, not hand-rolled per site. `SelfTest` builds, in memory, a corpus of
benign padding plus each needle (bytes taken from the deny list itself — already runtime-assembled, so no new
literal), runs `Scan`, and reports any entry the sweep failed to bite on. Every sweeping test calls
`SelfTest(deny)` before scanning the real corpus. Nothing is written to fixtures; teardown is garbage collection.
`sweep`'s own tests add the staged-mutation proof (precedent `sentinel_sweep.go:135-142`).
**Alternative rejected**: per-site staged mutations (three hand-rolled variants — the drift this change exists to end).

### AD-3 — WU-6 copied-value guard: value-shaped canary + AST escape guard (D-1)

The `%#v` hazard: fmt at depth > 0 renders a **pointer**-typed field as a hex address — a canary inside
`errors.New(...)` (a `*errorString`) never surfaces, and the test would go green while proving nothing.
**Concrete construction** (lands in `src/ai/provider_failure_test.go`, package `ai_test`):

```go
type valueShapedCause struct{ Detail string }          // stored in the error
func (c valueShapedCause) Error() string { return c.Detail } // interface AS A VALUE

failure, _ := ai.MidStreamFailure(ai.FailureReport{
    Category: ai.FailureCategoryUnavailable,
    Cause:    valueShapedCause{Detail: canary},        // canary runtime-assembled
}, false)
copied := *failure                                     // the AI-41-deferred shape
```

Three assertions: (1) **mechanism pin** — `fmt.Sprintf("%#v", copied)` (and `%v`, `%+v`) DOES contain the canary,
proving the copied-value leak is real and the structural guard is load-bearing; if someone later adds a value-receiver
method, this pin fails and forces the NFR-AIP-B tradeoff to be re-argued in the open. (2) **depth-hazard contrast** —
the same rendering with a pointer-shaped cause (`errors.New`) does NOT surface the canary, documenting in-test why the
value-shaped plant is mandatory. (3) **escape guard** — a `go/parser` walk over every non-test `.go` file in the
module flags any exported function/method signature or exported struct field carrying `Failure`/`ai.Failure` **by
value** (no `*`). Its positive control parses an in-memory synthetic source declaring `func Leak() ai.Failure` and
requires a flag — the guard is defeated ⇒ the control fails. Plus the D-1 doc-comment extension on
`provider_failure.go` (the AI-41 comment at :370-380 already states the posture; add the "pinned by AI-36" sentence).
No production method is added — the S-AIP-008 terminal-exclusivity guard (`provider_failure_test.go:200-201`) is
untouched by construction.

### AD-4 — WU-5 marker + allowlist over the inventory (D-3)

**Marker**: a comment line `// credential-scan:deliberate-plant — <reason>` added to each inventoried file. The
scanner assembles its detection needle at runtime (`"credential-scan:" + "deliberate-plant"`) — a contiguous literal
in the scanner would self-exempt the scanner's file and desynchronize the allowlist assertion.
**Walker**: `filepath.WalkDir` from `src/ai/openaicompat/`, sweeping every `_test.go` and fixture file (all packages —
the `externalTestPackageClause` filter is deleted).
**Committed allowlist**: a string slice of the six relative paths in `credential_scan_test.go`. Assertions:

1. Pattern match + no marker → fail, naming file and class (never the bytes).
2. Marker present but file not in allowlist → fail (a new plant is a reviewable diff).
3. **Falsifiability**: every allowlist entry must exist, carry the marker, AND still match ≥1 credential pattern —
   a stale entry (plant removed, file renamed) fails the build.
4. `S-ART-013…017` re-expressed; `TestCredentialScan_IgnoresInternalTestPackageFiles` inverts into
   "an internal-package file WITHOUT a marker IS swept" — the exact coverage widening D-3 promises.

**Rule for new AI-36 tests**: all new sentinels are runtime-assembled (concatenation), so no new file ever joins the
allowlist.

## Deliberate-plant inventory (design deliverable, blocker for tasks)

Every file under `backend/agent/src/ai/openaicompat/…` whose raw bytes match the scan's above-threshold classes
(`sk-` + 20+ `[A-Za-z0-9_-]`, `Bearer ` + 20+ `[A-Za-z0-9._-]`), verified by grep of the actual worktree:

| # | File (relative to `openaicompat/`) | Package | Line(s) | What it plants |
|---|---|---|---|---|
| 1 | `credential_test.go` | `openaicompat` | 22 | `sk-super-secret-token-value` — Credential.String/GoString redaction proof (AI-25, S-APC-053) |
| 2 | `capture_proof_test.go` | `openaicompat` | 13 (comment), 209 | `sk-AEM060-planted-in-body-only` — S-AEM-060 body-only capture proof |
| 3 | `viability_test.go` | `openaicompat` | 215 | `Bearer viability-proxy-token` (21-char bearer class) — proxy Authorization assertion |
| 4 | `openrouter/credential_redaction_test.go` | `openrouter` | 27, 75, 115 | `sk-super-secret-openrouter-token-shape` ×3 — wrapper redaction proofs |
| 5 | `openrouter/smoke/smoke_test.go` | `smoke_test` | 183 | `sk-prove-stages-of-gate-are-not-collapsed` — gate-stage non-collapse proof |
| 6 | `openrouter/smoke/sentinel_sweep_test.go` | `smoke_test` | 140, 212, 285 | `Bearer sk-aa-bb-cc-dd-ee-ff-gg`, `sk-aa-supersecret-leaked-shape-zz-9999`, `sk-deliberate-leaked-shape-bb-cc-dd-ee` — R-OR-08 bite proofs |

Outside the scan root (recorded so the boundary is explicit, no marker needed): `src/ai/content_part_test.go:252,407`
and `src/ai/provider_failure_test.go:1195` carry above-threshold canaries but sit outside `src/ai/openaicompat/…`,
which is D-3's declared walk root. Fixture directories carry no match.

### AD-5 — D-5 configured-client rendering, both branches

RED test (`openaicompat/a_i-36_2_test.go`, internal package — needs `New`): build `New(Config{Endpoint, Credential:
NewCredential(sentinel)})` with a runtime-assembled sentinel; render `c` and `*c` under `%v`, `%s`, `%+v`, `%#v` plus
`json.Marshal`; `sweep.Scan` the outputs. Expected RED: fmt reflects into the unexported `credential` field
(`CanInterface()` false ⇒ no dispatch to `Credential`'s methods) and prints `token`'s raw bytes.
- **If RED**: land value-receiver `String()`/`GoString()` on `Client` returning a fixed label
  (`"<openaicompat client>"`), mirroring `wrapper.go:90-101`. ~12 lines. fmt's documented panic-recovery covers the
  typed-nil `*Client` value-receiver edge, same as the shipped `redactedProvider` precedent.
- **If GREEN unchanged**: no production code; the test stays as a regression pin; trim lever (c) realizes for free.
  Either outcome is recorded in the task log.

Config-level proof is unconditional test-only (exported `Credential` field ⇒ dispatch works): same verb-exhaustive
shape at the `openaicompat.Config` level. Header proof (D-6): adversarial rendering test on `*RateLimitTelemetry` +
structural test that no diagnostic captures a full `http.Header` (the 3-name allowlist at `retry_metadata.go:69-84`
is the whole capture surface).

### AD-6 — D-4 hostile-server excerpt, both branches

**Stub shape** (in WU-2's `a_i-36_1_test.go`): `httptest.Server` whose handler reads the request and answers
`200` with `Content-Type: application/json` and body
`"auth=" + r.Header.Get("Authorization") + " body=" + <request body>` — a non-streaming success response, which
forces `refuseNonStreamContentType` (`stream.go:334-348`) to capture the echo as the excerpt. The test renders the
returned failure under all verbs AND walks the `errors.Unwrap` chain rendering each hop's `Error()` (R-ATS-023 says
callers read the cause's text, so the chain is in scope), then sweeps with a deny list of {credential sentinel,
`Bearer `+sentinel, prompt sentinel}.
- **Credential bites (expected)**: insertion point is `refuseNonStreamContentType` — it gains the client's
  credential as a parameter (unexported function; its caller is a `*Client` method) and stores
  `redactCredential(captured.bytes(), cred)` into `nonStreamContentType.excerpt`, where `redactCredential` is
  `bytes.ReplaceAll` of `cred.bearer()` then of the raw token with `<redacted>` (~10 lines, all unexported — no
  exported-method blast radius). Excerpt visibility (R-ATS-023) preserved: only token bytes are substituted, the
  rest of the body text stays. AI-32.5's bound untouched: the bound applies at capture; the substitution replaces a
  ≥20-byte token with a 10-byte placeholder and can never grow the excerpt.
- **Prompt-body echo**: named residual, not fixed — the test asserts the credential's absence and *documents* that
  the server's echo of the caller's own content may remain (it is the provider's response; suppressing it defeats
  R-ATS-023). `R-AEM-019` is authored only if the credential case bites.

## Blast-radius report (AI-41 W-2 process gap — reflection guards checked)

Ran `reflect.TypeOf(` / `NumMethod` greps across the whole module tree (not just literal verb greps), per D-5's
mandatory pre-check. Findings for every exported symbol this design introduces:

| New exported symbol | Reflection guards found | Verdict |
|---|---|---|
| `sweep.Entry`, `sweep.Scan`, `sweep.SelfTest` (new package) | None can reference a package that does not exist; `agenttest` guards pin `Step` (`script_introspect_test.go:176-185`) and `CapabilityRecordEntry` (`conformance_suite_test.go:1257`) — untouched types | **Clear** |
| `Client.String`/`Client.GoString` (contingent, D-5 RED) | `provider_boundary_test.go:28` and `stream_test.go:259` reflect on `&Client{}` but assert only `MethodByName("Stream")` presence/signature — **no method-count pin exists on `Client`**; `client_test.go:372` pins `Config` fields only | **Clear** |
| `smoke.DenyEntry` → type alias of `sweep.Entry` | No `%T` or `reflect.` usage anywhere under `smoke/` | **Clear** |
| WU-6 | Adds **no** method to `Failure`; the S-AIP-008 `exportedMethodNames` guard (`provider_failure_test.go:200-201, 1394-1398`) — the guard that bit AI-41 — compares method sets this change does not alter | **Clear** |

## Data Flow

    test corpus (events / error chains / verb renderings / log buffers / file bytes)
        │
        ├── agenttest scanForSentinel ──┐
        ├── smoke.Scan ─────────────────┼──→ agenttest/sweep.Scan → (vector, found)
        └── new AI-36 sweeps ───────────┘         ▲
                                                  └── sweep.SelfTest (positive control, per call site)
    credential_scan_test walker: WalkDir(openaicompat/…) → marker? → allowlist ✓/✗ → pattern classes

## File Changes

| File (under `backend/agent/`) | Action | WU |
|---|---|---|
| `src/agenttest/sweep/sweep.go` + `sweep_test.go` | Create | 1 |
| `src/agenttest/conformance_redaction.go` | Modify (delegate substring core) | 1 |
| `src/ai/openaicompat/openrouter/smoke/sentinel_sweep.go` | Modify (alias + delegate; names/messages verbatim) | 1 |
| `src/ai/openaicompat/a_i-36_1_test.go` | Create (adversarial sweep + hostile server) | 2 |
| `src/ai/openaicompat/a_i-36_2_test.go` | Create (config/client/header proofs) | 3 |
| `src/ai/openaicompat/client.go` | Modify *(contingent D-5 RED, ~12 lines)* | 3 |
| `src/ai/openaicompat/stream.go` | Modify *(contingent D-4 bite, ~10 lines)* | 2 |
| `src/agenttest/stream_kit_diff_hygiene_test.go` | Create (WU-4 bounded-summary hygiene + control) | 4 |
| `src/ai/openaicompat/credential_scan_test.go` | Modify (walker + marker + allowlist) | 5 |
| 6 inventoried files above | Modify (+marker comment, ~2 lines each) | 5 |
| `src/ai/provider_failure.go` | Modify (doc-comment sentence) | 6 |
| `src/ai/provider_failure_test.go` | Modify (mechanism pin, contrast, AST escape guard) | 6 |
| `openspec/changes/cachicamas-ai-redaction/specs/**` | Create (sibling agent; not this file's scope) | 7 |

## Testing Strategy / Sequencing (single PR, 1600-line budget)

Strict TDD; order: **WU-1 → WU-2 (D-4 empirical) → WU-3 (D-5 empirical) → WU-4 → WU-5 → WU-6 → WU-7** — the two
empirical unknowns resolve earliest so contingent production lines are known before the tail units land.

| WU | Prod | Test | Subtotal |
|---|---|---|---|
| 1 | ~90 | ~180 | ~270 |
| 2 | 0–10 | ~280 | ~290 |
| 3 | 0–12 | ~190 | ~200 |
| 4 | 0 | ~90 | ~90 |
| 5 | ~70 + 12 markers | ~150 | ~230 |
| 6 | ~8 | ~70 | ~80 |
| 7 (spec) | — | — | ~150 |
| **Total** | **~180–192** | **~960** | **~1,310** |

Headroom ~290 under 1600 (absorbs an AI-35-scale overrun of 168). **Pre-approved trim levers, ranked, carried
forward** so apply can shed lines without a new decision: (a) collapse D-3 to the external-only filter (−150, least
preferred — reopens the disclosed blind spot); (b) leave the smoke scanner duplicated (−80, weakens AI-36.1 t3);
(c) drop the client fix if D-5 is green (−12, free).

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration
boundary. The WU-5 walker is a read-only in-repo test scan.

## Migration / Rollout

No migration. Rollback per proposal § 10; every unit independently revertible.

## Overrides of proposal decisions

- **Delivery**: two-PR chain → single PR at 1600 lines — maintainer ruling, not this design's choice.
- **D-2 refinements (within the proposal's granted latitude)**: the subpackage does NOT collapse into `agenttest`
  (evidence: `smoke.Scan` is exported from a non-test file, so the collapse precondition is provably false), and
  `smoke.DenyEntry` becomes a type **alias** of `sweep.Entry` rather than a converted twin (evidence: zero
  `%T`/reflection uses in `smoke/`, so identity is safe and the eight R-OR-08 tests compile unchanged).
  No other proposal decision is overruled.

## Open Questions

- [ ] D-4 and D-5 empirical outcomes — intentionally open; both branches fully designed above, neither blocks tasks.
