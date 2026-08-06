# Design: AI-25 — Provider configuration and client construction

## Technical Approach

New subpackage `backend/agent/src/ai/openaicompat/` (dialect-named, guard-scannable as a whole directory; AI-00.3's allowlist admits it by prefix with zero edits). Construction only: validate injected config, store a parsed base URL and an opaque credential, hold an HTTP client with connect/idle bounds and **no whole-request timeout**. Stdlib only; `go.mod` stays at zero requires. Construction faults reuse AI-04's `Violation` (`ErrMalformed`/`ErrEmpty`); zero new sentinels. No `Stream` method until AI-26 — and `*Client`'s non-satisfaction of `ai.ModelProvider` is asserted at run time (R-APC-007), not merely left unstated.

## Architecture Decisions

| Decision | Choice | Rejected | Rationale |
|---|---|---|---|
| Package | `ai/openaicompat/` subpackage | inside `package ai` | AI-25.2 must scan "the adapter's own sources"; unenforceable inside `ai`'s ~60 shared files. Honors `doc.go:71–73` |
| Client injection | `Config.HTTPClient *http.Client`; nil → constructor builds a fresh bounded client | mandatory injection; timeout knobs in `Config` | Bounds are fixed safe defaults, not caller-injectable (settled). Injected client used verbatim, never mutated — its bounds are the injector's. Fresh transport per construction; never touches `http.DefaultTransport`/`DefaultClient` |
| Proxy | adapter-built transport leaves `Proxy` nil; injected clients are used verbatim — never rejected, never defensively re-transported | `http.ProxyFromEnvironment` (what `DefaultTransport` uses) on the adapter-built client; rejecting/rewrapping injected clients whose transport consults the environment | `ProxyFromEnvironment` reads `HTTP_PROXY` env vars — ambient authority the AST guard cannot see (selector resolves to `http`). Scope ruling (ADR 0005): the no-ambient-authority guarantee covers the adapter's own sources and its own constructed client; an injected client's transport behaviour (e.g. `&http.Client{}` → `DefaultTransport` → `ProxyFromEnvironment`) is the injector's responsibility — environment reading belongs to the composition root, and rejecting such clients would break legitimate corporate-proxy deployment. Stated in `openaicompat/doc.go` |
| Credential | `Credential` struct, unexported `token` field; `String()`/`GoString()` return `<redacted>`; no `MarshalJSON`/`MarshalText`; raw value reachable only via unexported `bearer()` used in `request.go` | plain `string`; `func() string` supplier | `fmt` (`%v/%s/%+v/%#v`) and `encoding/json` cannot leak it accidentally; one attachment site helps AI-32.5/AI-36.1. Supplier defers emptiness past construction, breaking fail-early. AI-24: "opaque **bearer-token** value through injected construction" |
| Faults | `ai.Invalid(ai.ErrMalformed, ai.At("endpoint"))`; `ErrEmpty` at `"endpoint"`/`"credential"` | AI-19 `Failure`; new sentinels | Pre-request, pre-I/O = AI-04's domain; `Failure`'s `DeliveryPath`/`PartialOutput` axes are meaningless here |
| Endpoint join | parse once in `New`; `(*url.URL).JoinPath` with empty segments filtered first | `ResolveReference`; `path.Join` on strings | RFC 3986 merge drops a non-trailing-slash base's last segment (`/proxy/openai` + `chat/completions` → `openai` lost). `JoinPath` returns a new URL, never mutates the stored base. Filtering empty segments pins the no-spurious-trailing-slash contract deterministically |
| Guard | `go/parser`+`go/ast`+`go/token` call-site scan of this package's non-test sources | AI-00.3 `go list -deps` reuse; `go/types` resolution | `net/http` transitively imports `os` — import-closure scan can never pass (doc 0002 A1 correction). `go/types` shadow-proofing needs `golang.org/x/tools` — spends zero-requires. Accepted, documented limitation: a local shadow of `os` can false-positive |
| Factory seam | constructor is pure, per-call cheap, no globals | — | `agenttest.Factory.New` closes over a per-case `httptest.Server`; nothing here forecloses it |

## Data Flow

    Config{Endpoint, Credential, HTTPClient?} → New()
      ├─ validate: endpoint parse/abs/http(s) → ErrMalformed·At("endpoint"); empty → ErrEmpty
      │            credential zero → ErrEmpty·At("credential")
      └─ Client{base *url.URL, cred, httpClient}
           └─ newRequest(ctx, segs…) → base.JoinPath + "Authorization: Bearer …" → httpClient.Do

Default client (nil injection): `Client.Timeout` **zero** (streaming footgun — caps body read); `DialContext: (&net.Dialer{Timeout: 10s})` (TCP connect only, dial.go:127); `TLSHandshakeTimeout: 10s` (handshake only, transport.go:184); `ResponseHeaderTimeout: 60s` (time-to-headers, excludes body, transport.go:227); `IdleConnTimeout: 90s` (pooled-connection idle reuse, **not** mid-stream read inactivity, transport.go:221). All four confirmed against Go 1.26 source this session. The four bound values live as unexported package constants (`defaultDialTimeout`, `defaultTLSHandshakeTimeout`, `defaultResponseHeaderTimeout`, `defaultIdleConnTimeout`) so structural tests assert against the same source of truth the constructor wires.

## File Changes

| File (worktree-absolute under `backend/agent/src/`) | Action | Slice | Owns |
|---|---|---|---|
| `ai/openaicompat/doc.go` | Create | A | dialect, boundary, guard-scope statement; explicit ambient-authority scope: the guarantee covers the adapter's own sources and its own constructed client — an injected client's transport behaviour is the injector's responsibility |
| `ai/openaicompat/client.go` | Create | A | `Config`, `Client`, `New`, validation, default client |
| `ai/openaicompat/credential.go` | Create | A | opaque `Credential` |
| `ai/openaicompat/endpoint.go` | Create | A | join helper (empty-segment filter + `JoinPath`) |
| `ai/openaicompat/request.go` | Create | A | `newRequest`: join + Bearer attachment (moved from proposal's slice C — item 1's stub-transport observation needs it) |
| `ai/openaicompat/client_test.go`, `credential_test.go`, `timeout_test.go`, `endpoint_test.go` | Create | A | internal tests (need `c.httpClient`); `client_test.go` also carries the five recorded mutation-detection reds (S-APC-005/014/020/023/037 evidence) |
| `ai/openaicompat/provider_boundary_test.go` | Create | A | run-time assertion that `*Client` does NOT satisfy `ai.ModelProvider` (R-APC-007, S-APC-031) |
| `ai/doc.go` | Modify | A | lines 5–9: adapters live in subpackages (first: `openaicompat`); lines 46–48: only AI-37 may add a dependency; lines 71–72: "Concrete vendor adapters … arrive from AI-24 onward" → arrive at AI-25 (AI-24 is a decision node shipping no adapter) — the "owns the contract, never a vendor's satisfaction of it" clause stays verbatim. Line 38 import rule and S-AGM-032 claims untouched |
| `ai/import_boundary_test.go` | Modify | A | lines 96–104: drop AI-24 bullet, "Two milestones" → one; line 134: drop "the AI-24 transport or" |
| `backend/agent/Makefile` | Modify | A | lines 56–59: same stale "AI-24 selects a transport (its own ADR gate)" claim — corrected alongside NFR-APC-C's three named sites per coordinator ruling (a fourth instance of the identical staleness; comment-only, zero behavioural risk) |
| `ai/openaicompat/ambient_authority_test.go` | Create | B | guard + 4 recorded bite proofs |
| `ai/openaicompat/viability_test.go` | Create | C | real `httptest.NewServer` probe, sent through the **adapter-built** client (proxy-immune) |

## Interfaces / Contracts

```go
func New(cfg Config) (*Client, error)              // Config{Endpoint string; Credential Credential; HTTPClient *http.Client}
func NewCredential(token string) Credential        // Credential.String()/GoString() → "<redacted>"
func (c *Client) newRequest(ctx, segments ...string) (*http.Request, error) // unexported; AI-26's seam
```

Guard algorithm: `os.ReadDir(".")` → keep `*.go`, skip `_test.go` → `parser.ParseFile` → forbidden set {`os`, `os/exec`, `syscall`, `io/ioutil`} → dot-import of a forbidden path = independent violation; else record local name (alias or last element) → path → `ast.Inspect` flags any `CallExpr` whose `SelectorExpr.X` resolves through that map, reported at `fset.Position`. Whole-package ban, no function list. Blank imports are inert (no callable name; unused non-blank imports fail compilation). Guard's own `_test.go` is exempt meta-tooling (in-repo precedent verbatim: "-test would also pull in `testing`, which imports `os` itself").

## Testing Strategy (RED-first; runner `make test` from `backend/agent/`)

| Behavior | Failing assertion before implementation |
|---|---|
| Injection used (25.1.1) | stub `RoundTripper` on injected client records request; scheme/host/path from injected endpoint, `Authorization: Bearer <tok>` present |
| Typed faults (25.1.2) | table: malformed/relative/schemeless/empty endpoint, zero credential → `errors.Is` `ErrMalformed`/`ErrEmpty`, position names the field |
| Safe defaults (25.1.3) | empty endpoint fails (no default); `DefaultTransport`/`DefaultClient` snapshots unchanged; injected client's fields unmutated; fresh transport ≠ `DefaultTransport`; `Proxy == nil` |
| No whole-request timeout (25.1.4) | **paired comparison**, one drip handler (k=5 chunks, flush + interval d=200ms): control `http.Client{Timeout: 2d}` must die mid-read — assert **shape** (`net.Error.Timeout()==true`, fewer than k chunks), never duration; constructed default client reads all k to EOF. Ratio margins (cap between 2d and (k−2)·d), no `t.Parallel()` on the pair. Flakiest test in the milestone; no CI, runs under `-race` on loaded machines |
| Path join (25.1.5) | table: `…/v1`+`chat/completions`→`…/v1/chat/completions`; `…/v1/`+same→same; sub-path base `/proxy/openai`→segment kept; `…/v1//`→collapsed; empty segment→base unchanged, no trailing slash; query base→query preserved; absolute-URL-shaped segment→literal path component, host unchanged |
| Credential opacity (R-APC-014, S-APC-053) | `%v/%s/%+v/%#v` and `json.Marshal` output must not contain the token (type-shape assertion only — wire-level redaction claims stay AI-32.5's/AI-36.1's, per S-APC-054/055) |
| No provider satisfaction (R-APC-007, S-APC-030/031/032) | **run-time** `_, ok := any(&Client{}).(ai.ModelProvider)` reports `ok == false` (S-APC-031) — deliberately NOT the compile-time `var _ ai.ModelProvider = (*Client)(nil)` idiom, which would fail the *build*, not fail as a test; goes green again only when AI-26 flips the expectation. S-APC-030 (no streaming entry point at all) and S-APC-032 (AI-20 signature guard unmodified) ride the same file |
| Connect/idle bounds exist (R-APC-005, S-APC-021/022) | **structural**, not behavioural: adapter-built transport's `TLSHandshakeTimeout`, `ResponseHeaderTimeout`, `IdleConnTimeout` equal their non-zero `default…` constants and `DialContext` is non-nil, wired from `defaultDialTimeout > 0`. The constants (10 s / 10 s / 60 s / 90 s) are the safe values this design records for S-APC-021/022's "within the safe range" clause; the tests assert equality against them. Field reads suffice here because proving a bound *exists* has no equivalent-reimposition escape hatch (unlike the whole-request-cap **absence** proof) — R-APC-005 records the same asymmetry. Behavioural variants rejected: the runner (`go test -race -v ./...`, no `-short`) would make the 60 s header bound a 60-second test (S-APC-071 forbids waiting bounds out), and macOS blackhole dials return `ECONNREFUSED` immediately — a vacuous green |
| Guard (25.2) | green on clean sources; **bite proofs**: four violations staged one at a time in `client.go` — plain `os.Getenv`, aliased `osx "os"` call, `exec.Command`, dot-import of `os` — each red `make test` run recorded in slice-B PR, then dropped, suite green. The staged reds ARE the guard's RED phase (guards are green-from-birth over clean code) |
| Mutation-detection reds (S-APC-005/014/020/023/037) | a different artefact from TDD RED: after each behaviour is GREEN, stage five deliberate breaks in the *finished* implementation — (1) store the injected client without using it (internal default used instead) → injection-observation test must go red (S-APC-005); (2) assign a bound onto the shared `http.DefaultTransport` → global-mutation snapshot test must go red (S-APC-014); (3) wrap the request in an internal `context.WithTimeout` → the constructed-client leg of the paired comparison must go red (S-APC-020); (4) leave the connect and time-to-headers bounds unset on the adapter-built transport → the structural bound assertions must go red (S-APC-023); (5) route the adapter-built client through `http.DefaultTransport` → the fresh-client identity check and/or the nil-proxy assertion must go red (S-APC-037, closing the proxy vector by detection rather than prose). Each red `make test` run recorded in the slice-A PR (same evidence obligation as the guard's bite proofs), reverted, suite re-run green |
| Viability (R-APC-013, S-APC-048…052) | server handler records `Authorization` header and URL path; assert Bearer token (S-APC-049), joined path (S-APC-050), exactly one request (S-APC-048), and a second construction with a different credential carries the second value (S-APC-051). The probe sends through the **adapter-built** client (nil `HTTPClient`): `ProxyFromEnvironment`'s localhost exemption matches the literal string `"localhost"` only, so `httptest`'s `127.0.0.1` URL **is** proxy-eligible and any default-transport client would misroute on a machine with `HTTP_PROXY` set. S-APC-052 sets `HTTP_PROXY` to a dead address via `t.Setenv` and asserts the probe still reaches the server. **Serial-scheduling mechanism**: `t.Setenv` panics under `t.Parallel()`, so S-APC-052 is serial by construction — exactly like the timing pair, which already declares no parallelism. Go runs a package's non-parallel tests strictly sequentially, so the two never overlap and neither blocks the other; they simply queue (≈1.2 s pair + a subsecond probe), and `t.Setenv`'s automatic cleanup restores the environment before the next serial test starts, so the proxy variable cannot leak into the timing pair |

## Threat Matrix

N/A — no routing, shell, subprocess, VCS/PR automation, executable-file classification, or process-integration boundary. The guard parses source statically (no subprocess); production code performs HTTP only.

## Migration / Rollout

No migration; additive with zero consumers. Rollback = delete `openaicompat/` (comment fixes independent). Delivery confirmed as proposed: linear feature-branch chain, tracker `feat/2026-08-03-cachicamas-ai-layer1-wave-4` ← **A** (AI-25.1 + all four comment corrections including the Makefile, `request.go` and `provider_boundary_test.go` included, plus the five recorded mutation-detection reds as PR evidence) ← **B** (AI-25.2) ← **C** (AI-25.3). The proposal's slice table is now stale in one respect: it placed the credential-attachment code in slice C; `request.go` moved to slice A (upheld by cross-validation — S-APC-003's stub-transport observation requires it, an ID unchanged in spec rev 2). B must follow A (guard scans A's sources); C exercises A's constructor; B/C are graph-parallel but stack linearly for clean diffs. Node boundary = chain boundary (doc 0002). Slice-A fission trigger stands: items 1–4 (doc, client, credential, request + tests) → A1; item 5 (`endpoint.go` + table) → A2 — A1's `newRequest` may join with bare `JoinPath` until A2 lands the filter and table. `Decision needed before apply: No` · `Chained PRs recommended: Yes` · `400-line budget risk: High` (session budget 5,000; corrected forecast 2,700–5,400).

## Open Questions

- [ ] Confirm `Authorization: Bearer` against AI-24's **merged** `decision.md` at apply time (proposal line 67 already says "opaque bearer-token"; residual R4).
- [ ] Out-of-scope observation for the coordinator: `ai/doc.go` line 38 still permits "vendor model SDKs" (S-AGM-032-guarded — do not touch here). The Makefile stale comment was absorbed into scope by coordinator ruling (see File Changes); spec rev 2's `NFR-APC-C` still enumerates three sites — the Makefile is the ruled fourth, to be reflected when the spec is next amended.
