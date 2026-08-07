# Proposal: Discharge the Wave-2 carryovers (AI-41)

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 (doc 0002 lines 2233–2257)
> **Wave**: 5 — Harden · **Module**: `backend/agent/` (layered per ADR 0005 § D1, NOT hexagonal)
> **Strategy**: single-pr · strict TDD on · stdlib-only · review budget 1000 changed lines
> **Package**: `backend/agent/src/ai` (top level) for both leaves — NOT `agenttest`, NOT `openaicompat`

## 1. Intent

Two coverage/redaction gaps were parked by AI-21, re-recorded as `W1`/`W2` in the Wave-2 verify report, and left open through Wave 2 and Wave 3. `openspec/specs/ai-stream-testkit/spec.md:39` names AI-41 as their owner; a third silent pass is the failure mode this milestone exists to stop.

- **W1 (AI-41.1)** — `ai.CheckEmit` has four rules; three have failure-path tests, the fourth (`e.payload.validate(Path{At("event")})`, `event.go:364–366`) has none. `ai-event-envelope/spec.md:322–324` records why: the only payload AI-14 can construct is `export_test.go`'s `WitnessPayload`, whose `validate` returns `nil` unconditionally (`export_test.go:63–68`); every production payload (AI-15…19) validates eagerly in its constructor, so an `Event` carrying an invalid payload is **unconstructible through the public API**. The spec offers two resolutions — give the witness a controllable failure mode, or record rule 4 as deliberately unreachable defence. AI-41.1's "proven directly" wording selects the first.
- **W2 (AI-41.2)** — `*ai.Failure` (`provider_failure.go:320–330`) holds `cause error`, which may wrap raw provider-body or credential-adjacent text. Its `Error()` (`:349–354`) is already redacted and proven with a planted secret (`provider_failure_test.go:1172–1240`). Because `Failure` implements `error`, `%v`/`%s`/`%q` already dispatch safely. **`%#v` is the one leaking verb**: with no `GoString`, `fmt` falls back to reflection over every unexported field, including `cause`. Redaction is therefore currently a property of *which verb the caller reached for*, not a property of the type — the exact posture `event.go:314–319` rejects in its own `GoString` doc comment.

## 2. Scope (in)

- **AI-41.1** — a controllable failure mode on `WitnessPayload` (test-support code in `export_test.go`, not a production surface) plus a new `CheckEmit` rule-4 failure-path test in `event_test.go`.
- **AI-41.2** — a redacting `GoString()` on `*ai.Failure` in `provider_failure.go`, plus an adversarial planted-canary `%#v` test in `provider_failure_test.go`.
- **Spec deltas** — `ai-event-envelope` (new `R-AEE-021`) and `ai-provider-errors` (new `R-AIP-016`).
- **Carryover closure** — the `ai-stream-testkit` carryover line and the `ai-event-envelope` "Carried forward" section amended append-only at archive time to read as *discharged*, not merely *assigned* (charter Acceptance).

## 3. Non-goals (out)

- **Any other redaction or emission-boundary behavior already proven** (charter Out-of-scope, verbatim). `Error()`'s redaction, rules 1–3, `Event.String`/`GoString`, `Part.String` — all already covered; not re-litigated.
- **AI-36's adversarial sweep**, which consumes this milestone's result rather than repeating it (charter Out-of-scope, verbatim).
- **`String()` on `*Failure`** — see **Flag 1**. It would be dead code.
- **`String`/`GoString` on `Violation`** — `validation.go:253–261` is structurally safe under reflection (`rule error` is always a fixed sentinel; `at Path` is positions only). That asymmetry is *why* W2 targets `Failure` alone.
- **Any new dependency.** `go.mod` stays untouched; `import_boundary_test.go` guards stay green.
- **A `a_i-41_*_test.go` file.** That naming convention is confined to `openaicompat/` and its `openrouter` subpackage; the top-level `ai` package uses topical file names. Both leaves extend existing files.

## 4. Approach

**AI-41.1 — additive `rejectWith` field.** `WitnessPayload` gains one field (`rejectWith *Violation`, nil by default) and `validate` returns it. Every existing construction path (`NewWitnessEvent`, `NewTestEvent`) leaves it nil, so all current tests are untouched — a zero-regression additive change. A new test-only constructor supplies a non-nil violation. The new test builds a `Stamper`-stamped witness event and asserts `CheckEmit` returns *that exact violation*, proving rule 4 specifically rather than an earlier rule short-circuiting. Because `KindTestWitness`'s descriptor is `Role: BlockRoleNone` (`export_test.go:79`), rule 3 returns nil on its early-exit branch, so rules 1–3 are structurally satisfied and only rule 4 can fire. Exact identifier names are the design phase's to fix.

**AI-41.2 — `GoString` delegating to the redacted renderer.** `func (f *Failure) GoString() string { return f.Error() }`. Nil-safe for free, because `Error()` already returns `noProviderFailure` for a nil receiver (NFR-AIP-B totality holds without a second nil check). The precedents are in-tree and identical in intent: `Event.GoString` (`event.go:319`), `openaicompat/credential.go:31–43`, `openaicompat/openrouter/wrapper.go:90–101` — the last one added explicitly to stop `%#v` reflecting into an embedded client's unexported credential field.

**Test shape (both leaves).** The adversarial redaction pattern is already in-tree at `content_part_test.go:414–419`: loop over `%v`/`%s`/`%+v`/`%#v`, assert a planted `CANARY` never appears. AI-41.2 reuses that exact shape with the canary planted in the `cause`. Strict TDD per `openspec/AGENTS.md` rule 3: RED (assertion fails for the right reason) → GREEN (minimum code) → REFACTOR.

## 5. Decisions taken (design phase may override)

### Flag 1 — AI-41.2 method shape: **`GoString()` only. No `String()`.**

The charter's two clauses read as compatible, not competing: "a redacting alternate-verb formatting method" is singular, and "matching the pattern its sibling payloads already carry" is satisfied by the *pattern* (delegate to an already-redacted renderer rather than reflect), not by a literal method count.

1. **A `String()` on `*Failure` would be unreachable.** `fmt`'s `handleMethods` consults `error` **before** `Stringer` for `%v`/`%s`/`%q`. `Failure` already implements `error`, so `Error()` wins every dispatch a `String()` could serve. Adding it ships dead code that a reader would have to re-derive as dead.
2. **The siblings' two-method shape exists because they are not errors.** `Event`, `Part`, `Completion` and the other 20 call sites have no `Error()`; they need `String()` to make `%v` safe *and* `GoString()` to make `%#v` safe. `Failure` already has half the pattern, spelled `Error()`.
3. **The governing spec text names the method singular and by name.** `ai-stream-testkit/spec.md:39` records W2 verbatim as "the missing redacting `GoString()` on the failure payload".
4. **Reversible.** Adding `String()` later is additive and breaks nothing; removing a shipped one is a public-surface removal.

### Flag 2 — AI-41.1 witness failure mode: **additive `rejectWith *Violation`, confirmed.**

Considered and rejected: (a) a *new* test-only payload type implementing `eventPayload` with an always-failing `validate` — more code, a second registry entry, and it duplicates `WitnessPayload`'s three other proofs for one new one; (b) recording rule 4 as deliberately unreachable defence — the spec offers it, but AI-41.1's "proven directly" forecloses it. `export_test.go` never reaches a non-test build (S-AEE-013, S-AEE-017), so this is **not** a production-surface change and does not widen any exported API.

### Flag 3 — Parallel safety: **the new rule-4 test may use `t.Parallel()`.**

`export_test.go:29–30` scopes the non-parallel discipline precisely: *a test holding a `RegisterTestKind` registration* must not run in parallel, because `eventRegistry` is one shared package-level slice and a concurrent truncation would corrupt it. The rule-4 test registers nothing — it uses the statically-registered `KindTestWitness` and carries its failure mode per-instance on the payload value, so there is no shared mutable state. The in-tree precedent is exact and on both sides: `TestCheckEmit_PayloadlessEvent_...` (`event_test.go:74`) calls `t.Parallel()`; `TestCheckEmit_BlockScopedEventWithZeroBlockIndex_...` (`event_test.go:187`) does not, and its comment cites the registration as the reason.

## 6. Dependencies

| Dep | What it provides | Status |
| --- | --- | --- |
| **AI-14** (event envelope) | `CheckEmit` (`event.go:339–368`); `WitnessPayload` + `KindTestWitness` (`export_test.go`); `R-AEE-001…020`, `S-AEE-001…070` | Archived |
| **AI-19** (provider errors) | `*Failure` + redacted `Error()` (`provider_failure.go:320–354`); `R-AIP-001…015`, `NFR-AIP-A…E` | Archived `2026-08-01-cachicamas-ai-provider-errors` |
| **AI-04** (`Violation`, `FirstFailure`, `Invalid`, `At`) | The rejection vocabulary both leaves assert against | Archived |

**Unblocks:** AI-36 (adversarial sweep) — consumes this result rather than repeating it.

## 7. Risks

| # | Risk | Likelihood | Mitigation |
|---|------|------------|------------|
| 7.1 | Adding `GoString` changes existing `%#v` output for `*Failure`. | Low | Grep confirms **zero** `%#v` / `GoString` occurrences in `provider_failure_test.go`; no test asserts today's reflected output. `make test` under `-race` re-verifies at apply. |
| 7.2 | The `rejectWith` field changes `WitnessPayload`'s comparability or existing behavior. | Low | Field is a pointer, nil by default; both existing constructors leave it nil, so `validate` still returns nil for every current caller. `WitnessPayload` stays a comparable struct. |
| 7.3 | The rule-4 test passes for the wrong reason (an earlier rule fires). | Med | Assert the returned violation is *the exact planted one* (identity/`errors.Is` against a distinct sentinel), not merely non-nil. `KindTestWitness` being `BlockRoleNone` removes rule 3 from the picture structurally. |
| 7.4 | Spec-identifier collision on `R-AEE-021` / `R-AIP-016` if another change lands first. | Low | Verified at propose time: `R-AEE-020` and `S-AEE-070` are the current maxima; `R-AIP-015` and `S-AIP-055` are the current maxima. `sdd-spec` re-verifies before writing. |
| 7.5 | Archive-time carryover amendment forgotten, reproducing the exact failure mode AI-41 exists to stop. | Med | Named as an explicit success criterion below and as an archive-phase deliverable, not left implicit in the change folder. |
| 7.6 | Charter Flag-1 wording re-read as requiring `String()` at design time. | Low | Flag 1 records the `fmt` dispatch evidence so the design phase overrides with reasons, not by re-reading the same ambiguous sentence. |

## 8. Capabilities (contract with sdd-spec)

### New Capabilities

None. Both leaves amend capabilities that already exist.

### Modified Capabilities

- **`ai-event-envelope`** — new `R-AEE-021`: the emission boundary's payload-validation rule rejects an event whose payload reports its own broken rule, and that rejection is proven directly rather than deferred. Scenarios `S-AEE-071` (rule fires and surfaces the payload's own violation) and `S-AEE-072` (rules 1–3 are satisfied, so only the payload rule can be responsible). The "Carried forward" section (`:322–324`) gains an append-only note recording `W1` discharged.
- **`ai-provider-errors`** — new `R-AIP-016`, appended after `R-AIP-015` and before the `NFR-AIP-*` section: the failure payload's redaction is a property of the type, not of the formatting verb — no formatting verb may reach the wrapped cause. Scenarios `S-AIP-056` (planted secret in the cause is absent from every verb including `%#v`) and `S-AIP-057` (a nil failure formats totally, never panicking — NFR-AIP-B restated for the new method).

> Spec files honor the no-Go-identifier rule (behavior-level wording). The identifiers named in this proposal are for design/apply, not for spec prose.

## 9. Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `backend/agent/src/ai/export_test.go` | Modified | `WitnessPayload` gains `rejectWith *Violation`; `validate` returns it; one new test-only constructor. Test-support code — never reaches a non-test build. ~15–20 lines. |
| `backend/agent/src/ai/event_test.go` | Modified | New rule-4 failure-path test, `t.Parallel()`, asserting the exact planted violation. ~30–45 lines. |
| `backend/agent/src/ai/provider_failure.go` | Modified | `func (f *Failure) GoString() string` delegating to `Error()`, with a doc comment stating the posture. ~6–8 lines. |
| `backend/agent/src/ai/provider_failure_test.go` | Modified | Planted-canary loop over `%v`/`%s`/`%+v`/`%#v` plus nil-receiver totality. ~35–50 lines. |
| `openspec/changes/cachicamas-ai-wave2-carryovers/specs/ai-event-envelope/spec.md` | New (delta) | `R-AEE-021` + `S-AEE-071`/`S-AEE-072`. |
| `openspec/changes/cachicamas-ai-wave2-carryovers/specs/ai-provider-errors/spec.md` | New (delta) | `R-AIP-016` + `S-AIP-056`/`S-AIP-057`. |
| `openspec/specs/ai-stream-testkit/spec.md:39` | Modified (archive) | Carryover line amended append-only to read *discharged*, per charter Acceptance. |
| `openspec/specs/ai-event-envelope/spec.md:322–324` | Modified (archive) | "Carried forward" gains the `W1`-discharged note; `R-AEE-021` promoted into the canonical spec. |
| `openspec/specs/ai-provider-errors/spec.md` | Modified (archive) | `R-AIP-016` promoted into the canonical spec. |
| `docs/architecture/milestones/0002-...md` | Modified (archive) | Top-status updated; amendment blockquote referencing this change, per Engram `#2638`. |
| `go.mod` | **Untouched** | No dependency added. `TestLayer1_ModuleHasNoDependencies_ZeroRequires` stays green. |

**Size forecast:** ~100–130 changed lines total (~25 production/test-support, ~80 test, plus spec deltas). Well under the 400-line default budget and far under the accepted 1000-line exception. **Decision needed before apply: No. Chained PRs recommended: No. 400-line budget risk: Low.** Single PR.

## 10. Rollback Plan

Revert the single PR. Both leaves are additive and independent of each other:

- **AI-41.2 alone** — delete `GoString` from `provider_failure.go`; `%#v` returns to reflected output. No caller depends on the new method's existence (it is reached only through `fmt`).
- **AI-41.1 alone** — remove the `rejectWith` field and the new constructor; `validate` returns to unconditional nil. Every pre-existing test is unaffected either way, because they never set the field.

Spec deltas live in the change folder and are reversible without touching code. If either fix proves wrong in substance, the carryover reverts to *open* and the `ai-stream-testkit` carryover line must be re-amended to say so — silently leaving it reading *discharged* is the one rollback error that would reproduce AI-41's own failure mode.

## 11. Success Criteria

- [ ] `ai.CheckEmit`'s fourth rule has a direct failure-path test that asserts the payload's own violation is what surfaced, with rules 1–3 structurally satisfied.
- [ ] Every pre-existing test in `package ai` passes unchanged — `WitnessPayload`'s new field is nil on every existing construction path.
- [ ] `*ai.Failure` renders redacted under `%#v`; a canary planted in the `cause` appears under **no** verb in `{%v, %s, %+v, %#v}`.
- [ ] A nil `*Failure` formats totally under every verb and never panics (NFR-AIP-B).
- [ ] No `String()` added to `*Failure` (Flag 1), unless the design phase overrides with recorded reasons.
- [ ] `cd backend/agent && make test` green under `-race`; `make lint` clean; both import guards pass; `go.mod` still declares zero requires.
- [ ] Spec deltas authored for `ai-event-envelope` (`R-AEE-021`) and `ai-provider-errors` (`R-AIP-016`).
- [ ] **At archive**: `ai-stream-testkit/spec.md:39` reads as *discharged by AI-41*, not *assigned to AI-41*; `ai-event-envelope`'s "Carried forward" records `W1` closed. This is the charter's Acceptance line and the milestone's whole reason to exist.

## 12. Recommended next step

`sdd-spec` and `sdd-design` (parallel). Spec authors the two delta files in behavior-level wording. Design confirms or overrides Flag 1 (method shape), Flag 2 (witness failure mode), and Flag 3 (parallel safety), and fixes the concrete Go identifiers this proposal deliberately leaves provisional.

## References

- **Charter** — `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md:2233–2257`.
- **Explore artefact** — `openspec/changes/cachicamas-ai-wave2-carryovers/explore.md`; Engram `#2662`.
- **Rule 4 (untested)** — `backend/agent/src/ai/event.go:321–368` (doc comment states the four rules and their order).
- **Rule 1 / 2 / 3 existing failure-path tests** — `event_test.go:73–88`, `sequence_test.go:171–199`, `event_test.go:187–210`.
- **Witness payload + non-parallel discipline** — `backend/agent/src/ai/export_test.go:21–30, 39–68, 98–111`.
- **W1 recorded verbatim, with both resolutions** — `openspec/specs/ai-event-envelope/spec.md:322–324`.
- **W1/W2 assigned to AI-41** — `openspec/specs/ai-stream-testkit/spec.md:39`.
- **`Failure` struct + redacted `Error()`** — `backend/agent/src/ai/provider_failure.go:315–354`.
- **Existing planted-secret `Error()` test** — `backend/agent/src/ai/provider_failure_test.go:1172–1240`.
- **`GoString` posture precedent, stated as a doc comment** — `backend/agent/src/ai/event.go:314–319`.
- **Redaction precedents under `%#v`** — `openaicompat/credential.go:31–43`; `openaicompat/openrouter/wrapper.go:90–101`.
- **Adversarial verb-loop test pattern** — `backend/agent/src/ai/content_part_test.go:404–419`.
- **Structurally-safe sibling (why `Violation` needs neither method)** — `backend/agent/src/ai/validation.go:253–261`.
- **Spec numbering maxima verified at propose time** — `R-AEE-020` / `S-AEE-070`; `R-AIP-015` / `S-AIP-055`.
- **House-style precedent for this artefact** — `openspec/changes/archive/2026-08-07-cachicamas-ai-retry-policy/proposal.md`.
- **Mandatory doc 0002 + Engram + push workflow** — Engram `#2638`.
