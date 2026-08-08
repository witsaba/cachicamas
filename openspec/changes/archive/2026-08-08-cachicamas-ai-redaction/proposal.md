# Proposal: Enforce secret redaction (AI-36)

> **Change**: `cachicamas-ai-redaction` · **Milestone**: AI-36 (doc 0002 lines 2150–2184, amendment 2161, AI-41 close line 22)
> **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1)
> **Strategy**: single-pr with pre-accepted `size:exception` · strict TDD on · stdlib-only · review budget 1000 changed lines
> **Blocks**: AI-37 (observability boundary) · **Unblocked by**: AI-41 (redacted Go-syntax rendering on the failure payload)

## 1. Intent

Redaction in this tree is currently **a property of which surface you reached for**, not a property of the module. Ten renderings are individually proven safe (§ explore.md 1) and each was proven by its own owning milestone, in isolation. Nobody has ever run one adversarial sweep across *every* failure path with a planted sentinel and asserted absence — which is the exact posture the charter exists to end, and the precondition AI-37 needs before it may emit spans at all.

Three concrete gaps make this more than a bookkeeping exercise:

- **Inherited (AI-41 close, doc 0002 line 22)** — the provider-failure payload's redaction is pointer-scoped. A copied value is neither an error nor a Go-syntax stringer, so reflection walks its unexported fields. AI-41 explicitly deferred this to AI-36.
- **Latent (new, explore.md § 2)** — the configured adapter client holds its credential in an *unexported* field and carries no formatting method of its own. Because `reflect.Value.CanInterface()` is false for unexported fields, fmt cannot dispatch to the credential's own redacting methods and falls back to raw reflection. Today the wrapper short-circuits before fmt gets there, so this is latent, not live — and completely untested.
- **Duplicated** — three independent sentinel scanners now exist (conformance case, live-smoke sweep, fixture scan). "The sweep is a reusable helper rather than a one-off" (AI-36.1 test 3) is the charter's own answer to that drift.

## 2. Scope (in)

| # | Work unit | Charter leaf | Shape |
|---|---|---|---|
| WU-1 | One shared, dependency-free sentinel-sweep helper under `agenttest/`; the conformance case and the live-smoke sweep converge on it | AI-36.1 t3 | Prod (test-support) + tests |
| WU-2 | Adversarial sweep: a planted sentinel **credential** and a planted sentinel **prompt body** are absent from every error, wrapped cause, verbose formatting and event metadatum across every failure path the suite can trigger — including a hostile server that echoes both back | AI-36.1 t1/t2 | Tests (+ contingent prod, D-4) |
| WU-3 | Configuration and configured-client redaction proven verb-exhaustively at the adapter level; header-capture redaction proven | AI-36.2 t1/t2 | Tests + contingent ~10 prod lines (D-5) |
| WU-4 | Bounded-summary hygiene: the assertion helpers' own failure output stays sentinel-free, adversarially, with a positive control | AI-36.3 t1 | Tests |
| WU-5 | Fixture scan widened recursively across the whole adapter tree, with the deliberate-plant exclusion converted from a silent category to an enumerated, reviewable allowlist | AI-36.3 t2 | Test-support + tests |
| WU-6 | The AI-41-inherited copied-value gap: closed by structural proof + a guard, not by a new value-receiver method (D-1) | doc 0002 line 22 | Tests + doc-comment |
| WU-7 | Spec deltas (§ 8) | — | Spec |

## 3. Non-goals (out)

- **AI-25.3's credential-attachment proof** — charter Out-of-scope, verbatim. That the credential *reaches* the transport is AI-25's obligation; AI-36 only proves it never reaches a diagnostic.
- **AI-32.5's wire-error-body size bound** — charter Out-of-scope, verbatim. The 8 KiB capture bound is not re-litigated. AI-36 asserts *sentinel absence* on that path, never *bound correctness*.
- **AI-37's observability boundary and all OTel work** — no span, no attribute, no allowlist, no dependency. AI-36 is AI-37's precondition, not its first slice.
- **A general-purpose header-redaction utility.** The only header-capturing diagnostic today uses a 3-name allowlist and never touches the raw header set (explore.md § 10.4). Adding a reusable redactor with zero consumers is speculative generality; if AI-37 needs one, AI-37 owns it.
- **Any new dependency.** `go.mod` and `go.sum` stay **byte-identical**. `TestLayer1_ModuleHasNoDependencies_ZeroRequires` and both import guards stay green.
- **Migrating existing internal-package tests to the external-package fixture convention.** WU-5 makes the exclusion explicit and reviewable; it does not restructure AI-25's or AI-32's test packages.
- **Live-network behavior changes.** The live smoke's skip/fail semantics are untouched; only its scanner's *implementation* is redirected at the shared helper.

## 4. Decisions taken (design phase may override with recorded reasons)

### D-1 — Copied provider-failure value: **prove structurally non-escaping + guard it. Do NOT add a value-receiver method.**

**Decision.** Keep the pointer receivers. Add (a) an adversarial test that constructs the copied-value form directly and records its actual rendering under every verb, and (b) a **structural guard test** asserting no exported function, method, or field anywhere in the module returns or stores the failure payload by value — so the leaking shape is unreachable through the public surface and any future change that makes it reachable fails the build. Plus a doc-comment on the type stating the scoping and why.

**Why not a value-receiver method.** AI-41 recorded the tradeoff at `provider_failure.go:370-380`: a value receiver puts the method in the pointer's method set too and changes typed-nil formatting behavior, re-opening **NFR-AIP-B** (total nil-safety) — a landed, shipped guarantee — to cover a shape no production path produces. Every constructor returns a pointer; the payload reaches callers only as an `error` or through the event's error-payload accessor. Trading a landed totality guarantee for an unreachable copy is the wrong direction.

**Falsifiability hazard the design phase must handle.** The naive adversarial test may go green *for the wrong reason*. Under `%#v`, fmt's `printValue` at depth > 0 renders a **pointer**-typed cause as a hex address, not its message text — so a canary planted in a pointer-shaped cause would not surface even though the mechanism is unsafe. The test must plant the canary in a **value-shaped** cause (or an exported nested field) and must ship a positive control proving the assertion bites, per the repo's existing staged-mutation convention (`sentinel_sweep.go:135-142`).

### D-2 — Sweep helper home: **`backend/agent/src/agenttest/`, confirmed** — as a dependency-free subpackage.

**Confirmed with evidence.** Go's `internal/` rule makes `src/ai/internal/…` importable only by packages rooted at `src/ai/`. `src/agenttest` is a **sibling** of `src/ai`, not nested inside it (ADR 0005 § D2 location table, `docs/adr/0005-promote-agent-stack-to-own-module.md:198-218`), so an internal package would exclude the one package that already owns the analogue. `agenttest` is also ADR 0005 § D2's designated external-consumer-proof location, and every prospective caller (`openaicompat`, `…/openrouter`, `…/smoke`, `…/conformance`) can import it.

**Refinement (new, not in exploration).** The scanner goes in a small **subpackage** of `agenttest` rather than into `agenttest` itself, because `agenttest` imports `testing` package-wide (`conformance_redaction.go:24`) and `openrouter/smoke/sentinel_sweep.go` is a **non-test** file; importing `agenttest` from it would pull `testing`'s flag registration into a non-test build. The subpackage imports `bytes`/`strings` only. If the design phase proves the smoke wrapper is only ever reached from `_test.go` files, it may collapse the subpackage into `agenttest` — that is a reversible, one-import change.

**Composition, not duplication.** One implementation, three call sites: `conformance_redaction.go`'s `scanForSentinel`/`scanTextForSentinel` keep their names and their event-walking/`summarize`-aware logic but delegate their substring core; `openrouter/smoke`'s `Scan`/`BuildDenyList`/`DenyEntry` **keep their exported names** (R-OR-08 and its eight tests bind to them) and become thin adapters over the shared deny-list core; `credential_scan_test.go`'s pattern table stays where it is but reuses the shared "runtime-assembled needle" discipline. Nothing is deleted from the public surface.

### D-3 — Widened fixture scan: **recursive across the adapter tree, with the deliberate-plant exclusion made explicit.**

**Decision.** Replace the single-directory read with a walk over `src/ai/openaicompat/…` (including `openrouter/`, `smoke/`, `conformance/` and their fixture directories) and **replace the blanket "external test packages only" filter** with: sweep every `_test.go` and fixture file; a file is excluded only if it carries an explicit, in-file deliberate-plant marker; and a separate test asserts the set of marker-bearing files matches an enumerated, committed list.

**Why this preserves the deliberate plants without a blind spot.** `stream.go:309-318` records the current exclusion as a *disclosed gap* and names its own successor: *"A package-internal sweep that understands deliberate plants is possible future hardening, owned by a future guard change; recorded here so the gap is disclosed, never silent."* AI-36 **is** that guard change. The blanket filter today means an accidental real credential in any internal-package test file is invisible; the marker converts an invisible category into a reviewable list of named files, so widening the scan *increases* coverage instead of merely relocating the hole. Adding a marker to a new file becomes a visible diff a reviewer must approve.

**Constraints.** The marker string and the allowlist entries must be runtime-assembled, never contiguous literals, or the scan flags its own source (`credential_scan_test.go:39-44`). Existing scenarios `S-ART-013…017` must all still pass; `TestCredentialScan_IgnoresInternalTestPackageFiles` is re-expressed against the marker rule rather than deleted. The design phase must first **locate** every file carrying a deliberate above-threshold plant — that inventory is a design deliverable, not an apply-time discovery.

### D-4 — Non-streaming content-type excerpt: **adversarial test first; contingent, pre-decided remedy.**

**Decision.** No unconditional production change. The excerpt is built solely from the *response* body and its Content-Type header (`stream.go:294-307`), never from the request, so the credential and prompt sentinels cannot reach it unless the server echoes them. WU-2 therefore includes a **hostile-server case**: a stub that echoes the request's Authorization header and request body into a non-streaming response, after which the rendered failure text is swept.

**If it bites** (credential case): the pre-decided remedy is **credential-aware excerpt redaction inside the capture path** — the client already holds the credential, so occurrences are removable before the excerpt is stored. This preserves **R-ATS-023**, which explicitly requires the caller reading the terminal failure's rendered text to see the excerpt, and it is *not* AI-32.5's size bound. ~10 production lines.

**If it bites** (prompt-body case): recorded as a **named residual, not a defect**. The excerpt is the provider's own response; suppressing a server's echo of the caller's own content would defeat R-ATS-023's diagnostic purpose. AI-36.1 test 2 targets content leaking through *our* error paths, not a remote party replaying our request.

### D-5 — Adapter configuration and configured client: **config proven by test; client fix gated on a RED-first empirical run.**

- **Configuration value** (AI-36.2 item 2, "safe to print by construction"): its credential field is **exported**, so `CanInterface()` is true and fmt dispatches to the credential's own value-receiver redacting methods at depth. Safe by construction — but proven today only one layer up, at the wrapper's config. **Decision: test-only**, verb-exhaustive (`%v`, `%s`, `%+v`, `%#v`, `json.Marshal`) with a planted sentinel, at the adapter's own configuration level. No production change expected.
- **Configured client value** (the latent gap): **in scope for AI-36.2 as defense-in-depth, but the fix is gated on empirical confirmation.** The confirmation *is* the strict-TDD RED step: WU-3's first commit adds a verb-exhaustive test that formats a bare, unwrapped configured client built with a sentinel credential and asserts absence. If it fails, the ~4-line redacting pair lands mirroring the wrapper's precedent (`wrapper.go:90-101`). **If it passes unchanged, no production code is written** and the test is kept as a regression pin. Neither outcome blocks the milestone; both are recorded.
- **Mandatory pre-check before adding any exported method** (AI-41 W-2 process gap, doc 0002 line 22): grep for `reflect.TypeOf(` and `NumMethod` across the tree in addition to literal verb occurrences. A reflection-based guard is invisible to a literal grep — this is exactly how `S-AIP-008`'s terminal-exclusivity guard bit AI-41.

### D-6 — Header capture: **test-only, no new utility.** See § 3. Proven by (a) an adversarial test that the rate-limit telemetry rendering never reproduces a header name or value, and (b) a structural test that no diagnostic in the tree captures the full header set — the "opt-out-explicit, never opt-in" property is satisfied by an allowlist, which is opt-in by construction and therefore strictly stronger than the charter asks.

## 5. Approach

Strict TDD throughout (`openspec/AGENTS.md` rule 3): RED asserting for the right reason → GREEN with the minimum code → REFACTOR. Two disciplines are non-negotiable for this milestone specifically:

1. **Every sweep ships a positive control.** An absence assertion that cannot fail is worse than no assertion, because it trains reviewers to trust it. The repo already has the convention (`sentinel_sweep.go:135-142`, `credential_scan_test.go:120-158`, `conformance_redaction.go:47-53`'s purity note): each new sweep gets a staged-mutation or hand-built-leak test proving it bites.
2. **No sweep ever reprints the sentinel it found.** Failure messages name the vector, file, event index or verb — never the matched bytes (`R-CNF-013`, `S-ART-017`, `smoke.Scan`'s own comment).

Work-unit order follows the charter's dependency edges: WU-1 → (WU-2, WU-3, WU-4, WU-5 in any order) → WU-6 → WU-7. WU-6 is last because it is the only unit whose subject is outside the adapter package.

## 6. Size forecast

| Work unit | Production | Test / test-support | Subtotal |
|---|---|---|---|
| WU-1 shared helper + convergence | ~90 | ~180 | ~270 |
| WU-2 adversarial sweep (credential + prompt, incl. hostile server) | 0–10 (D-4) | ~280 | ~285 |
| WU-3 config + client + header proofs | 0–12 (D-5) | ~190 | ~200 |
| WU-4 bounded-summary hygiene | 0 | ~90 | ~90 |
| WU-5 recursive scan + marker allowlist | ~70 (test-support) | ~150 | ~220 |
| WU-6 copied-value structural guard | ~8 (doc-comment) | ~70 | ~78 |
| **Code total** | **~170–190** | **~960** | **~1,140** |
| WU-7 spec deltas | — | — | ~150 |

**Production-vs-test ratio.** ~5.6×, squarely inside this repo's strict-TDD band: AI-34 (+1241/-2 for +15 production), AI-35 (+1301/-62), AI-41 (+257/-12 for +27 production). The forecast is therefore a *central estimate*, not a ceiling — AI-35 overran its accepted 1096-line exception by 168 lines on exactly this mechanism.

**Guard lines** (per `sdd-phase-common.md` § E; `sdd-tasks` re-forecasts authoritatively):

```
Decision needed before apply: Yes
Chained PRs recommended: Yes
400-line budget risk: High
```

The session's 1000-line budget is pre-accepted for this milestone, but ~1,140 code lines **exceeds it**. Two paths, in preference order:

1. **Chain two PRs** — PR #1 = WU-1 + WU-2 (the sweep and its first consumer, ~555 lines, autonomous and independently verifiable); PR #2 = WU-3…WU-7 (~585 lines). Each slice has a clear start, finish, verification and rollback.
2. **Single PR with a raised exception**, if the maintainer prefers one review pass.

**Named scope-trim levers**, ranked, so apply can shed lines without re-opening a decision: (a) collapse D-3's marker/allowlist back to the current external-package-only filter (−~150, but re-opens the disclosed blind spot — least preferred); (b) leave `openrouter/smoke`'s scanner duplicated instead of converging it (−~80, weakens AI-36.1 test 3); (c) drop the client fix if D-5's RED comes back green (−~12, free).

## 7. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| 7.1 | An absence assertion passes vacuously (sentinel never reached the surface under test), producing false confidence — the milestone's characteristic failure mode. | **High** | Positive control mandatory per work unit (§ 5.1). D-1 names the specific fmt-depth mechanism that would cause it. |
| 7.2 | Widened scan breaks the build on deliberate above-threshold plants. | Med | D-3's marker rule; the deliberate-plant inventory is a **design-phase deliverable** before any scan code is written. |
| 7.3 | Adding an exported formatting method collides with a reflection-based guard invisible to grep. | Med | D-5's mandatory `reflect.TypeOf(` / `NumMethod` pre-check. Precedent: AI-41's `S-AIP-008` collision, doc 0002 line 22. |
| 7.4 | The size forecast overruns the accepted exception, as AI-35's did. | **High** | § 6's two-PR chain as the default and three ranked trim levers pre-approved. |
| 7.5 | The shared helper drags `testing` into a non-test build via the smoke package. | Med | D-2's dependency-free subpackage; `make build` and `make lint` catch it immediately. |
| 7.6 | Converging the smoke scanner regresses R-OR-08's eight landed tests. | Low | Exported names preserved verbatim; the change is internal delegation only. |
| 7.7 | Spec-identifier collision if another change lands first. | Low | Maxima verified at propose time (§ 8). `sdd-spec` re-verifies before writing. |
| 7.8 | Fixing the excerpt path (D-4) conflicts with R-ATS-023's visibility requirement. | Low | The pre-decided remedy redacts *the credential within* the excerpt, preserving visibility; the prompt-echo case is a declared residual, not a fix. |

## 8. Capabilities (contract with sdd-spec)

### New capabilities

**None.** Every leaf amends a capability that already exists.

### Modified capabilities

| Capability | Next free ID | What changes |
|---|---|---|
| `ai-provider-conformance-suite` | `R-CNF-020` | The suite's redaction proof is a reusable sweep any adapter and any future failure path inherits, rather than a single registered case; a planted sentinel credential and a planted sentinel content body are absent from every rendering the suite can reach. |
| `ai-provider-client` | `R-APC-015` | Both the adapter's configuration value **and** its configured client value keep the credential out of every formatting rendering, including the Go-syntax one — redaction is a property of the values, not of which one the caller happened to print. |
| `ai-request-translation` | `R-ART-022` (scenarios from `S-ART-090`) | The credential scan's surface extends over the whole adapter tree and its fixtures; files carrying a deliberate credential-shaped plant are excluded only by an explicit in-file declaration enumerated in a committed list, so no file is silently unswept. Amends `R-ART-004` append-only. |
| `ai-stream-testkit` | `R-STK-014` | The assertion helpers' own failure output is proven sentinel-free adversarially, and the proof itself carries a control showing it can fail. |
| `ai-provider-errors` | `R-AIP-017` | The provider-failure payload's redaction scope is stated and pinned: the shape that would render its wrapped cause is unreachable through the published surface, and a guard fails if that ever stops being true. |
| `ai-provider-error-mapping` | `R-AEM-019` *(contingent on D-4)* | A refusal that reproduces a bounded excerpt of the provider's response never reproduces the caller's own credential within it. **Authored only if D-4's hostile-server case bites.** |

> Spec files honor the **no-Go-identifier** rule: behavior-level wording only, naming formatting verbs in prose ("the Go-syntax verb", "the extended verb"), never a Go symbol, method, field, or type name. Every identifier in this proposal is for design/apply consumption, not for spec prose.

## 9. Affected areas

| Area | Impact | Description |
|---|---|---|
| `backend/agent/src/agenttest/<sweep-subpackage>/` | **New** | Dependency-free deny-list/sentinel scanner. ~90 lines + ~180 test. |
| `backend/agent/src/agenttest/conformance_redaction.go` | Modified | `scanForSentinel`/`scanTextForSentinel` delegate their substring core; case registration and event-walking logic unchanged. |
| `backend/agent/src/ai/openaicompat/openrouter/smoke/sentinel_sweep.go` | Modified | `Scan`/`BuildDenyList`/`DenyEntry` keep their exported names, become adapters over the shared core. |
| `backend/agent/src/ai/openaicompat/credential_scan_test.go` | Modified | Recursive walk + marker-based exclusion + enumerated allowlist assertion. |
| `backend/agent/src/ai/openaicompat/a_i-36_*_test.go` | **New** | Adversarial sweep, config/client redaction, header proof, hostile-server case. Milestone-numbered convention, per AI-33/34/35 in this package. |
| `backend/agent/src/ai/openaicompat/client.go` | Modified *(contingent, D-5)* | Redacting formatting pair on the configured client. ~12 lines. |
| `backend/agent/src/ai/openaicompat/capture.go` or `stream.go` | Modified *(contingent, D-4)* | Credential-aware excerpt redaction. ~10 lines. |
| `backend/agent/src/ai/provider_failure.go` | Modified | Doc-comment recording the pointer-scoped redaction posture. ~8 lines, no behavior change. |
| `backend/agent/src/ai/provider_failure_test.go` | Modified | Copied-value rendering test with positive control + structural escape guard. Topical file naming, per the top-level package convention. |
| `openspec/changes/cachicamas-ai-redaction/specs/**` | **New** | Delta specs per § 8. |
| `go.mod` / `go.sum` | **Untouched** | Byte-identical. Layer 1 adds no dependency; AI-37 is the second and last milestone permitted to. |

## 10. Rollback plan

Revert the PR (or the affected slice, if chained). Every unit is independently revertible:

- **WU-1/WU-2** — delete the subpackage; restore the two delegating call sites to their inline bodies. Both are pure substring scanners with no state, so reverting cannot change any assertion's verdict.
- **WU-3 / WU-5** — the contingent production fixes are additive methods and an additive redaction step; removing them restores today's exact behavior. The widened scan reverts to `os.ReadDir(".")` with the external-package filter.
- **WU-6** — a doc-comment and two tests; removal is behavior-neutral by construction.

If a sweep proves wrong in substance (a leak it declared absent is later found), the correct move is **not** to silently narrow the sweep: record the leak, re-open the affected requirement, and amend the spec append-only. Leaving a requirement reading *proven* while its proof was weakened is the one rollback error that reproduces the failure mode this milestone exists to stop.

## 11. Success criteria

- [ ] A planted sentinel **credential** appears in no error string, wrapped cause, verbose or Go-syntax formatting, or event metadatum across every failure path the suite can trigger.
- [ ] A planted sentinel **prompt body** is equally absent from every error, log field and event metadatum; the server-echo residual (D-4) is named in writing, not silently absent.
- [ ] Exactly **one** sentinel-sweep implementation exists in the tree; the conformance case and the live-smoke sweep both reach it; `R-OR-08`'s landed tests stay green with their exported names unchanged.
- [ ] The adapter's configuration value **and** its configured client value are verb-exhaustively proven credential-free; D-5's empirical outcome (fix landed / test kept as pin) is recorded either way.
- [ ] The header-capturing diagnostic is proven never to reproduce a header name or value, and no diagnostic in the tree captures the full header set.
- [ ] Assertion-helper failure output is proven sentinel-free, **with a control proving the assertion can fail**.
- [ ] The fixture scan covers the whole adapter tree; every excluded file is excluded by an explicit marker and appears in a committed, asserted list.
- [ ] The copied provider-failure value's leaking shape is unreachable through the published surface, guarded by a test that fails if that changes.
- [ ] Every new sweep carries a positive control; no sweep reprints a matched sentinel.
- [ ] `cd backend/agent && make test` green under `-race`; `make lint` reports 0 issues; `make build` exits 0; `go.mod`/`go.sum` byte-identical to base; both import guards green.
- [ ] Spec deltas authored per § 8 in behavior-level wording, with **zero** Go identifiers.

## 12. Recommended next step

`sdd-spec` and `sdd-design` (parallel).

- **sdd-spec** authors the five certain deltas (§ 8) and holds `R-AEM-019` pending D-4's empirical outcome.
- **sdd-design** owns four deliverables this proposal deliberately leaves open: (1) the **inventory of files carrying deliberate credential plants**, without which WU-5 cannot be written; (2) the concrete identifiers and package name for the shared helper, including whether the subpackage collapses into `agenttest` (D-2); (3) the exact canary-placement shape that makes WU-6's assertion non-vacuous given fmt's depth behavior (D-1); (4) the hostile-server stub's shape for D-4.

## References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2150-2184`; amendment blockquote `:2161`; AI-41 close (the inherited follow-up) `:22`; authoring constraint `:24-25`.
- **Explore artefact** — `openspec/changes/cachicamas-ai-redaction/explore.md`; Engram `#2679`.
- **Pointer-scoped redaction + the recorded tradeoff** — `backend/agent/src/ai/provider_failure.go:349, 370-380, 381`.
- **Unexported-credential formatting gap** — `backend/agent/src/ai/openaicompat/client.go:43-58, 63-67`; redaction precedent `openaicompat/openrouter/wrapper.go:90-101`.
- **Excerpt interpolated into error text, with its own disclosed scan gap** — `backend/agent/src/ai/openaicompat/stream.go:294-318, 326-328`.
- **Existing sweep analogue** — `backend/agent/src/agenttest/conformance_redaction.go:29-73, 87-156`.
- **Duplicate scanner to converge** — `backend/agent/src/ai/openaicompat/openrouter/smoke/sentinel_sweep.go:74-146`.
- **Fixture scan to widen** — `backend/agent/src/ai/openaicompat/credential_scan_test.go:1-30, 56-90, 180-199`.
- **Bounded-excerpt machinery to protect** — `backend/agent/src/agenttest/stream_kit_diff.go:27, 91-147, 168`.
- **Header allowlist** — `backend/agent/src/ai/openaicompat/retry_metadata.go:53, 69-84`.
- **Verb-exhaustive redaction test pattern** — `openaicompat/credential_test.go:19`; `openrouter/credential_redaction_test.go:24-60, 72`; `ai/content_part_test.go:404-419`.
- **Spec numbering maxima verified at propose time** — `R-CNF-019`, `R-APC-014`, `R-ART-021`/`S-ART-089`, `R-STK-013`, `R-AIP-016`, `R-AEM-018`.
- **No-Go-identifier convention, worked example** — `openspec/specs/ai-provider-errors/spec.md:235-249`.
- **Owning spec for the fixture scan** — `openspec/specs/ai-request-translation/spec.md:107-119` (`R-ART-004`, `S-ART-013…017`).
- **Runner commands** — `backend/agent/Makefile:90-92`.
- **House-style precedent for this artefact** — `openspec/changes/archive/2026-08-07-cachicamas-ai-wave2-carryovers/proposal.md`.
