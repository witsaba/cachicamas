# Delta — `ai-openrouter-first-provider`

> **Change**: `cachicamas-ai-live-smoke` (AI-39, Wave 6) · **Target**: [`openspec/specs/ai-openrouter-first-provider/spec.md`](../../../../specs/ai-openrouter-first-provider/spec.md)
> **Scope**: `R-OR-07` and `R-OR-08` only — the two requirements AI-38's delta explicitly fenced to AI-39. `R-OR-01`…`R-OR-06`, `R-OR-09`, `R-OR-10` stand unchanged; `R-OR-05`/`R-OR-06` carry AI-38's promoted text and are not reopened here.
> **Basis**: proposal fork **D5** — the orphan `openspec/changes/add-openrouter-first-provider/` folder's env-var-only `R-OR-07` text is adopted as canonical through this delta, then that folder is deleted. The archived workflow-dispatch text is superseded by facts on disk: the repository has no `.github/` directory at all (ADR 0005 § Enforcement) and the shipped gate reads environment variables only.
> **Also required at archive (not spec text, recorded here so it is not lost)**: the canonical file's four-part promotion transform — `Status: DRAFT` → `Status: live`, an `Introduced by:` line, relative links re-resolved from active-change depth to canonical depth (`../../proposal.md` currently 404s), and an added `## Status` canonical-home section.
> **Format**: RFC 2119 + Given/When/Then, per `openspec/config.yaml` `rules.specs`.

---

## MODIFIED Requirements

### R-OR-07 — Live smoke is opt-in via environment variables only, with no CI workflow file

The live smoke (`TestOpenRouterAdapter_LiveSmoke`) SHALL be `t.Skip`-gated unless BOTH
`OPENROUTER_API_KEY` is present in the test process AND `RUN_LIVE_OPENROUTER_SMOKE` equals the exact
literal `1`; the credential alone SHALL NOT be sufficient consent. The skip reason SHALL attribute
which stage closed the gate. `make test` in `backend/agent/` SHALL NOT depend on OpenRouter or on any
network credential. **No CI workflow file is required or created by this spec**: the repository's
established posture is that no `.github/workflows/` directory exists (ADR 0005 § Enforcement; doc 0002
"No CI exists"), so the live smoke is opt-in for human, local runs only. The smoke package SHALL live
under a path containing an `internal` segment beneath `.../openaicompat/openrouter/`, and
credential-safe setup instructions SHALL ship in that package directory. The full live-smoke contract
is stated in the `ai-live-smoke` capability (`R-LSM-001`…`R-LSM-008`); this requirement is its
vendor-side anchor.
(Previously: the gate was stated as `OPENROUTER_API_KEY` alone, and the requirement mandated a
`.github/workflows/agent-openrouter-smoke.yml` `workflow_dispatch`-only file. That file does not
exist, cannot exist under ADR 0005's no-CI posture, and was never created; its scenario is therefore
retired with the clause that required it rather than left as an unsatisfiable obligation. The
`internal` placement and the shipped setup instructions are new, from AI-39.1 items 4 and 5.)

> **Amended 2026-08-09 (AI-39)** by `cachicamas-ai-live-smoke`: the gate is restated as the shipped
> two-stage environment-variable opt-in; the CI-workflow obligation is removed as superseded by facts
> on disk, adopting the text the orphan `add-openrouter-first-provider` change folder already carried;
> `internal` placement and shipped setup instructions are added. The removed
> "Workflow file is dispatch-only" scenario is retired deliberately, not lost — it asserted properties
> of a file this repository will never create.

#### Scenario: Skip path exercised without the env vars

- GIVEN `OPENROUTER_API_KEY` OR `RUN_LIVE_OPENROUTER_SMOKE=1` absent from the test process
- WHEN `make test` runs
- THEN the live smoke is skipped with a reason naming the missing stage
- AND no outbound request is made.

#### Scenario: The credential alone does not open the gate

- GIVEN `OPENROUTER_API_KEY` present and `RUN_LIVE_OPENROUTER_SMOKE` set to any value other than the
  exact literal `1`
- WHEN the gate is evaluated
- THEN it reports closed and the test skips
- AND no outbound request is made.

#### Scenario: No CI workflow file is introduced

- GIVEN the merged repository
- WHEN it is searched for a `.github/workflows/` file triggering the live smoke
- THEN none exists
- AND no scheduled, push, or pull-request trigger of a billing-consuming run exists.

#### Scenario: The package is internal and ships its own setup instructions

- GIVEN the merged smoke package
- WHEN its path and directory contents are read
- THEN the path contains an `internal` segment beneath `.../openaicompat/openrouter/`
- AND setup instructions naming both env vars, shell `export` only, the exact post-move invocation,
  the timeout bound and the per-run cost are present in that directory
- AND no step in them writes the credential to a file inside the repository.

### R-OR-08 — Credential redaction extends to the live smoke's own captured output, proven by a positive control

The live smoke SHALL NOT log the credential, the secret value, or the full prompt, on the success path
or on any failure path. It SHALL route every diagnostic it produces through a single capture sink and
SHALL run the shared sentinel sweep over that sink before any of those bytes reaches the test log. The
deny list SHALL include the literal `OPENROUTER_API_KEY` env-var name, the secret value's 4-character
prefix, and the planted prompt marker. The sweep's positive control (`sweep.SelfTest`) SHALL pass over
that deny list before a clean sweep result is trusted; a clean result obtained without the positive
control SHALL count as no result, per `R-CNF-027` clause 5. A sentinel match SHALL fail the test,
naming only the vector and stating that output was withheld, and SHALL NOT reproduce the matched
bytes. The smoke package's scan SHALL delegate to the module's single sweep implementation and SHALL
be proven equivalent to it, so `S-CNF-080`'s "both reach that one implementation" holds after the
package is relocated.
(Previously: the sweep was required to exist and to catch a planted leak, but nothing bound it to the
live test's own captured output, the failure paths, or the mandatory positive control — so the "even
on failure" clause was an untested intention and a clean result could pass vacuously.)

> **Amended 2026-08-09 (AI-39)** by `cachicamas-ai-live-smoke`: the sweep is now bound to the live
> run's own capture sink on every path, the `sweep.SelfTest` positive control becomes mandatory before
> a clean result is trusted, and the single-implementation convergence obligation is restated so it
> survives the move to `.../openrouter/internal/smoke`.

#### Scenario: Sentinel sweep catches a deliberate leak mutation

- GIVEN a mutated smoke test that calls `t.Logf(key)` with the credential
- WHEN the sentinel sweep runs
- THEN the test fails and names the leak vector without reprinting the credential.

#### Scenario: `openaicompat.Credential` redaction carries through

- GIVEN the wrapper's credential value
- WHEN it is rendered through `String()`, `GoString()`, `MarshalJSON`, or default formatting
- THEN no rendering contains the token.

#### Scenario: The sweep runs over the live run's captured output on every path

- GIVEN a credentialled live run that ends by success, by request failure, by drain failure, or by a
  failed stream-shape invariant
- WHEN the run finishes
- THEN the sweep has run over the captured diagnostics for that path before any of them reached the
  test log
- AND a clean sweep releases the captured diagnostics in full.

#### Scenario: The positive control gates the clean result

- GIVEN the deny list built for a live run
- WHEN `sweep.SelfTest` is forced to fail over it
- THEN the test fails rather than reporting a clean sweep
- AND no clean-sweep claim is made without the control having passed.

#### Scenario: One sweep implementation, reached from both sides after the move

- GIVEN the merged module with the smoke package under `.../openrouter/internal/smoke`
- WHEN the sweep implementations are enumerated and both the smoke-side scan and the suite-side scan
  are run over the same corpus and planted needle
- THEN exactly one implementation exists
- AND both are proven equivalent to it, with every previously published name still published.

---

## ADDED Requirements

None — AI-39's new obligations live in the new `ai-live-smoke` capability (`R-LSM-001`…`R-LSM-008`).

## REMOVED Requirements

None. `R-OR-07`'s CI-workflow clause and its "Workflow file is dispatch-only" scenario are retired
inside the MODIFIED block above, not by removing the requirement.

## RENAMED Requirements

None.

## Traceability

| Delta | Basis |
|---|---|
| `R-OR-07` modified | Fork **D5** (adopt the orphan folder's env-var-only text); AI-39.1 items 1, 4, 5; ADR 0005 § Enforcement; AI-38 delta's explicit scope fence |
| `R-OR-08` modified | AI-39.1 items 2 and 3; fork **D3** (capture sink with a deferred sweep); fork **D2** (`S-CNF-080` re-anchored to the shared core from both sides); `R-CNF-027` clause 5 (mandatory positive control) |
| Canonical header promotion | Explore § "Reconciliation target (a)"; `WAVE-1-ARCHIVE.md` § 2 four-part transform — applied at archive, not here |
| Orphan folder deletion | Fork **D5**; explore § "Reconciliation target (b)" — deleted only after its `R-OR-07` text is adopted above |
