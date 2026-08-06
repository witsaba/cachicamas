# Proposal: AI-25 — Provider configuration and client construction

> **Milestone**: AI-25 (doc 0002, Wave 4) · **3 nodes**: AI-25.1 `[leaf]`, AI-25.2 `[guard]`, AI-25.3 `[leaf]`
> **Depends on**: AI-24 (decided) · **Blocks**: AI-26 … AI-32
> **Scope**: first Go code of Wave 4. Construction only — no request is sent as adapter behavior.

## Intent

AI-24 named a dialect (OpenAI-compatible Chat Completions), a transport (raw `net/http`, `go.mod` stays at zero requires), and a credential boundary (an opaque token arrives by injection; the adapter never reads the environment, the filesystem, or spawns a process). Nothing yet *is* that adapter. AI-26 (request translation), AI-27 (SSE decode) and AI-32 (error mapping) each need a constructed, configured value to build against, and none of them is authorized to invent its own configuration story.

Two defects make this more than scaffolding. First, the naive construction of an `http.Client` for a streaming adapter sets `Client.Timeout`, which — per the stdlib's own words — "includes … reading the response body" and "will interrupt reading of the Response.Body". Every stream longer than that value dies mid-read and surfaces four milestones later as an unexplainable truncation. Second, "the adapter reads no environment variable" is a claim about *call sites*, and doc 0002 mis-specifies its mechanism (see **A1** below). A guard that cannot bite is worse than none, because it is believed.

## Scope

### In Scope

- **AI-25.1 — Injected construction `[leaf]`**, five ordered items:
  1. endpoint, credential and `*http.Client` injected; construction succeeds and each value is demonstrably used, observed through a stub transport rather than by field inspection.
  2. invalid endpoint or empty credential fails **at construction**, with a typed error, before any request exists.
  3. safe defaults: no default endpoint that could silently target production from a test; no mutation of any shared/global client or of `http.DefaultTransport`.
  4. **no whole-request timeout** — connect-phase and pool-idle bounds only, proven behaviorally (see Approach).
  5. path-bearing and trailing-slash base endpoints join to correct request paths, with no doubled and no dropped segment.
- **AI-25.2 — No ambient authority `[guard]`**: a `go/ast` call-site scan over the adapter package's own non-test sources, plus a recorded **bite proof**.
- **AI-25.3 — Test-server viability `[leaf]`**: pointed at a local `httptest.NewServer`, a request reaches it with the credential attached in the dialect's expected header shape. **Attachment only.**
- Two comment-only corrections in `package ai`, routed here by AI-24's proposal ("the comment corrected by the first milestone that opens that package"): `doc.go`'s claim that this package "will own … the concrete vendor adapters", and `import_boundary_test.go`'s stale "AI-24 will add a second allowlist entry" text (verified at lines 96–102 and 134).

### Out of Scope

- **Sending any request as adapter behavior.** AI-25.3's test-server probe is the one sanctioned exception and exists solely to prove construction wired things correctly.
- Catalog files, login flows, credential persistence, and where the token originates (Layer 3).
- Request translation (AI-26), SSE framing (AI-27), event mapping (AI-28 … AI-31), error mapping (AI-32).
- **Secrecy assertions.** The wire-error-body size bound is AI-32.5's; the exhaustive sentinel sweep is AI-36.1's.
- Building an `agenttest.Factory` or running the AI-23 conformance suite against this adapter — a later milestone. AI-25 must only not *foreclose* it.
- A mid-stream inactivity cutoff. No stdlib field provides one; `IdleConnTimeout` is a connection-pool bound, not a read-idle bound. Deferred, unassigned.

## Capabilities

### New Capabilities

- `ai-provider-client`: the adapter's construction contract — what is injected, what is validated and when, which faults are typed and with which taxonomy, the timeout posture and why, endpoint path-joining semantics, the no-ambient-authority guard's scope and mechanism, and the credential-attachment shape.

### Modified Capabilities

- None. The `doc.go` correction follows the precedent set by `ai-fake-provider` NFR-AFP-E and `ai-stream-testkit` NFR-STK-F: the documentation obligation is stated as an NFR in the *new* capability's spec, not as a delta on `agent-module-scaffold`. `agent-module-scaffold` S-AGM-032 (doc.go states the boundary, the import rule, and cites ADR 0005 § D1) must survive the edit unchanged.

## Approach

**Package: a new subpackage `backend/agent/src/ai/openaicompat/`** — not inside `package ai`. The deciding argument is mechanical: AI-25.2 must scan "the adapter's own source files", and that scope is unenforceable inside `ai`'s shared ~60-file package. Verified against the guard: `allowedNonStdlibPrefixes = []string{modulePath}` and `isAllowed` matches by prefix, so any subpackage under `github.com/cachicamas/backend/agent` is already admitted with **zero guard or allowlist edits**. The name is dialect-based, not vendor-branded, because OpenRouter and local compatible servers are in scope.

**Typed faults reuse AI-04's `Violation` taxonomy, not AI-19's `Failure`.** Construction happens before any `Request` value or any I/O exists — precisely AI-04's "decidable from the request alone, without contacting a provider" domain, and explicitly outside AI-19's, whose `DeliveryPath` and `PartialOutput` axes are meaningless pre-request. A malformed endpoint is `ErrMalformed`; a missing credential is `ErrEmpty`. **Zero new sentinels.**

**Timeout absence is proven behaviorally, not by field inspection.** Asserting `client.Timeout == 0` proves nothing — a constructor could impose an equivalent cap internally via `context.WithTimeout`. The proof is a *paired comparison* against one `httptest` handler that writes and flushes several chunks with deliberate delays totalling more than D: (a) a control `http.Client{Timeout: D/2}` against that handler **must** fail mid-read, proving the test genuinely exercises the footgun; (b) the constructed client, same handler and timing, **must** read the stream to completion. Construction sets only `Transport.DialContext` (`net.Dialer.Timeout`), `TLSHandshakeTimeout`, `ResponseHeaderTimeout` and `IdleConnTimeout` — never `Client.Timeout`.

**Path joining uses `(*url.URL).JoinPath`** (Go 1.19+; `go.mod` pins 1.26.3), parsing the endpoint once at construction and storing the `*url.URL`. `ResolveReference` is rejected: RFC 3986 reference-merging treats a non-trailing-slash base's last segment as a document to be *replaced*, so `https://gw/proxy/openai` + `chat/completions` silently yields `https://gw/proxy/chat/completions`. `JoinPath` returns a new URL and does not mutate the receiver, so one stored base is safe to reuse. Table cases: trailing-slash base, non-trailing-slash base, **sub-path base (the merge footgun)**, doubled interior slashes, empty segment, base carrying a query string, and an absolute-URL-shaped string passed as a segment (must stay a literal path component).

**AI-25.2 is a `go/ast` call-site scan — deliberately NOT the AI-00.3 mechanism.** doc 0002 item 1 says "an AST import-and-call scan in the AI-00.3 style"; AI-00.3 is in fact `go list -deps -test`, an import-path transitive-closure scan that says nothing about call sites, and `net/http` transitively imports `os`. Reusing it would either false-positive on legitimate transport use or miss a narrow `os.Getenv`. The living-graph amendment for this is entry **A1** in AI-24's artifact — referenced here, **not duplicated**. Mechanism: `go/parser` + `go/ast` + `go/token`, all stdlib, so **no `go.mod` change**. Per file, build a local-identifier → import-path map from `file.Imports` (so `import osx "os"` resolves correctly); walk `ast.CallExpr`; flag any `*ast.SelectorExpr` whose `X` resolves to `os`, `os/exec`, `syscall` or `io/ioutil` — banning the whole package rather than curating a function list, matching AI-00.3's deny-by-default philosophy. A dot-import of any forbidden path is an independent violation in its own right, since resolving bare identifiers would need `go/types` and in practice `golang.org/x/tools/go/packages` — a non-stdlib dependency that would break the zero-requires invariant. Accepted, flagged limitation: a local shadow of the identifier `os` could theoretically false-positive without type information.

**Scan scope: non-test sources only (`_test.go` excluded).** Decided, with an in-repo citation. `import_boundary_test.go`'s own sibling guard states the reason verbatim: "-test would also pull in `testing`, which imports `os` itself — a guard that scanned test imports could never pass." AI-25's own tests must call `httptest`, and its scratch bite-proof lives in production source precisely so the guard can see it. Scanning tests would make the guard either vacuous or permanently red. The guard's own test file is exempt from its rule — it is meta-tooling, exactly as AI-00.3's guard freely uses `os/exec` to implement the sibling import check.

**Bite proof staging** (guard-leaf closing rule: a guard closes only when shown to bite). Four scratch violations added to a real non-test adapter source, each exercising a distinct guard branch: (1) a plain `os.Getenv` call; (2) an aliased `osx "os"` call, proving the alias map bites; (3) an `os/exec` process spawn; (4) a dot-import of `os`, proving the independent dot-import branch. Each is run individually, the red `make test` output is pasted into the slice-B PR description, then all four are dropped and the suite re-run green. Four rather than one because the alias and dot-import branches are the guard's most fragile code and an untested branch is not a guard.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/ai/openaicompat/` | New | Adapter package: doc, config/client construction, validation, path-join |
| `backend/agent/src/ai/openaicompat/*_test.go` | New | Construction, typed-fault, timeout-absence, path-join table, ambient-authority guard, test-server viability |
| `backend/agent/src/ai/doc.go` | Modified | Correct "will own … the concrete vendor adapters"; correct the stale "two milestones may add a dependency" paragraph. S-AGM-032's claims preserved |
| `backend/agent/src/ai/import_boundary_test.go` | Modified | Comment-only: lines 96–102 and line 134 drop the AI-24 clause |
| `backend/agent/src/ai/validation.go` | Consumed | `Invalid`, `ErrMalformed`, `ErrEmpty`, `At` reused unchanged |
| `backend/agent/go.mod` | Unchanged | Zero requires preserved — stdlib only, verified |
| `openspec/specs/ai-provider-client/spec.md` | New (at archive) | Live contract home |

## Delivery — chained PRs

`delivery_strategy: auto-chain`, `chain_strategy: feature-branch-chain`, `review_budget_lines: 5000`. Naive forecast ≈ **1,340** changed lines; corrected for this repo's confirmed 2–4× undershoot, **≈ 2,700–5,400** — straddling the budget. doc 0002 § Split triggers: "the node boundary is the PR-chain boundary."

| Slice | Nodes | Contents | Naive | Corrected |
|---|---|---|---|---|
| **A** | AI-25.1 | package doc, construction + validation + path-join, all AI-25.1 tests, both `package ai` comment corrections | ~840 | ~1,700–3,400 |
| **B** | AI-25.2 | `go/ast` guard, four bite proofs and their recorded red output | ~350 | ~700–1,400 |
| **C** | AI-25.3 | test-server viability probe | ~150 | ~300–600 |

Chain is linear: tracker `feat/2026-08-03-cachicamas-ai-layer1-wave-4` ← **A** ← **B** ← **C**. AI-25.2 and AI-25.3 are graph-parallel (both depend only on AI-25.1) but stack linearly to keep each child's diff clean. **Pre-declared fission trigger for slice A**: if it exceeds the budget, split at the node's own seam — items 1–4 (construction, typed faults, defaults, timeout posture) in A1, item 5 (path joining) in A2.

`Decision needed before apply: No` · `Chained PRs recommended: Yes` · `400-line budget risk: High`

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| **R1** The paired-comparison timeout test is time-dependent and there is no CI, so it runs on loaded developer machines under `-race` | Med | Use a wide ratio, not tight absolute values (control cap ≈ D/2 against a stream spanning several flushes); assert the *shape* of the control's failure, not its exact duration; no `t.Parallel()` on the timing pair |
| **R2** `net.Dialer.Timeout` and `TLSHandshakeTimeout` semantics were recalled during exploration, not re-fetched (unlike `Client.Timeout`, `ResponseHeaderTimeout`, `IdleConnTimeout` and `URL.JoinPath`, all freshly confirmed) | Low | Confirm both doc strings at design time before the values are wired |
| **R3** `go/ast` without type information can false-positive on a local shadow of `os` | Low | Documented, accepted; closing it needs `golang.org/x/tools`, which would break the zero-requires invariant that AI-24 bought |
| **R4** AI-25.3's expected header shape (`Authorization: Bearer <token>`) comes from the dialect, not from a re-read of AI-24's landed `decision.md` (concurrently being written) | Med | Re-confirm against AI-24's merged artifact before sdd-design locks the assertion |
| **R5** Corrected forecast straddles the 5,000-line budget | Med | Three-slice chain above, with slice A's fission trigger pre-declared |
| **R6** The `doc.go` edit could weaken a claim AI-00 guarded | Low | The edit is narrow and additive-in-precision; S-AGM-032's boundary statement, import rule and ADR 0005 § D1 citation must remain verbatim, checked in verify |
| **R7** The subpackage choice contradicts `doc.go`'s current text, and someone later reads that text as binding | Low | That is exactly why the correction is in scope; `doc.go` already says "this package owns the contract, never a vendor's satisfaction of it", which the subpackage placement honors |

## Rollback Plan

The adapter is additive with no consumer — AI-26 … AI-32 are unstarted — so rollback is deletion, not migration.

1. Delete `backend/agent/src/ai/openaicompat/`, or `git revert` the slice's commit range on its chain branch. Later slices revert independently, in reverse order (C, then B, then A).
2. The two `package ai` comment corrections are comment-only and behavior-free; they may be kept or reverted independently of the Go code.
3. `go.mod` is untouched by design, so no dependency state can be left behind.
4. Re-run `make test` from `backend/agent/` to confirm AI-00 … AI-23 remain green.
5. `openspec/changes/cachicamas-ai-provider-client/` moves aside; no spec is promoted to `openspec/specs/` until archive, so no live contract is orphaned.

## Dependencies

- AI-24's decision artifact: dialect, transport, credential boundary — all settled and given.
- AI-04 (`Violation`, `ErrMalformed`, `ErrEmpty`, `At`) and AI-20 (`ModelProvider`, for eventual satisfaction) — both shipped in-repo.
- Go standard library only: `net/http`, `net`, `net/url`, `net/http/httptest`, `go/ast`, `go/parser`, `go/token`. **No new top-level dependency, so no ADR gate fires and the AI-00.3 forward guard stays green.**

## Success Criteria

- [ ] Construction succeeds with endpoint, credential and `*http.Client` injected, and each injected value is observed in use through a stub transport — not asserted by reading a field.
- [ ] A malformed endpoint and an empty credential each fail at construction with an AI-04 `Violation` carrying the right sentinel and step, before any request exists. **No new sentinel is added.**
- [ ] No default endpoint exists that could reach a real provider from a test, and no shared or global HTTP client or transport is mutated.
- [ ] The paired-comparison test shows the control client dying mid-read and the constructed client completing the same stream — proving the whole-request timeout is behaviorally absent, not merely unset.
- [ ] All seven path-joining cases pass, including the sub-path non-trailing-slash base that `ResolveReference` would silently truncate.
- [ ] The `go/ast` guard is shown to bite against all four scratch violations, each red run recorded in the slice-B PR, then dropped and re-run green.
- [ ] The guard scans the adapter's non-test sources only, and that exclusion is justified in its own doc comment with the in-repo precedent.
- [ ] A request to a local `httptest` server arrives carrying the credential in the dialect's expected header shape.
- [ ] `backend/agent/go.mod` still carries zero `require` directives.
- [ ] `doc.go` no longer claims `package ai` owns the concrete vendor adapters, and `import_boundary_test.go` no longer names AI-24 as a future allowlist entrant, with S-AGM-032's claims intact.
- [ ] `make test` is green from `backend/agent/` for every slice, recorded in that slice's PR.
