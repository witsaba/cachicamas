# Spec — The pre-request hook seam (`agent-pre-request-hook`)

> **Change**: `cachicamas-agent-pre-request-hook` · **AG-08** (Layer 2, Wave 2, milestone 8 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-08--add-the-pre-request-hook-seam), `0003:833-900`
> **Nodes**: AG-08.1 `[leaf]` (hook seam) · AG-08.2 `[leaf]` (prefix stability)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `## rules.specs`. Every scenario independently verifiable.
> **IDs**: `R-PRH-NNN` / `S-PRH-NNN`. Distinct from `R-AEV-`/`S-AEV-`, `R-AMT-`/`S-AMT-`, `R-APE-`/`S-APE-`, `R-AGE-`/`S-AGE-`, `R-LSK-`/`S-LSK-`.
> **Scenario count** (AG-04 W9): **allocated `S-PRH-001` through `S-PRH-011`, plus the lettered bites `S-PRH-001a`, `S-PRH-001b`**. Each milestone that appends records its own additions; this header states the allocated range and never a total, because a total is defended by no test and goes silently false on the next append (`S-LSK-020`/AG-20 treatment). Amended 2026-08-20 (AG-20): four requirements MODIFIED for chain composition, replaced literal "9 total" with allocated range per `S-LSK-020` pattern.

## Coverage

| Charter | Requirements | Spec | Bites |
|---|---|---|---|
| **6 of 6** | 7 (`R-PRH-001`–`007`) | **allocated `S-PRH-001` through `S-PRH-011`, including 4 AG-20 additions** | **2** (`S-PRH-001a`, `S-PRH-001b`) |

Charter → spec: AG-08.1 #1 → `R-PRH-002`/`S-PRH-001` + bites; AG-08.1 #2 → `R-PRH-005`/`S-PRH-002`; AG-08.1 #3 → `R-PRH-003`/`S-PRH-003`; AG-08.1 #4 → `R-PRH-004`/`S-PRH-004`; AG-08.2 #1 → `R-PRH-006`/`S-PRH-005`; AG-08.2 #2 → `R-PRH-007`/`S-PRH-006`. Cross-cuts → `R-PRH-001` (surface), `S-PRH-007` (AG-07 W1 unbuffered-sink).

## Purpose

AG-08 lands **seam 1 of v2 § 6** — the only point where the outgoing request still exists as data, between `buildLoopRequest` (`backend/agent/src/agent/loop.go:132`) and `provider.Stream` (`backend/agent/src/agent/loop.go:140`). It closes **R-12** (G4's Layer 2 half: prompt-cache prefix stability) by giving Layer 2 a single, narrow mutation point that Layer 3's breakpoint placement (doc 0004 CO-24.1) and other consumers will stand on. AG-08 is the first live consumer of Layer 1's AI-12 copy-on-write rebuild (`Request.With(...)`, R-REX-001 at `backend/agent/src/ai/request.go:325-336`) and the first public-surface extension of `TurnOptions` since AG-07. Per D1/D2/D3/D4/D5 in `proposal.md`, the seam is a single callable on `TurnOptions` of type `func(ctx context.Context, req ai.Request) (ai.Request, error)` with a nil = identity default; AG-20 widens to chain composition.

## Requirements

### R-PRH-001 — `TurnOptions.PreRequestHook` surface (D1, D3)

The system SHALL expose `TurnOptions.PreRequestHook func(ctx context.Context, req ai.Request) (ai.Request, error)`. Nil is the identity default. No external caller of the field exists yet (AG-08 is the first user); the field is a non-breaking zero-value extension of `TurnOptions` (`loop.go:46-51`).

#### Scenarios

- **S-PRH-001** — AG-08.1 #1 hook sees + shapes outgoing request. Given a hook that returns `req.With(ai.WithSystemInstruction(<instr with new segment>))`, when `Turn` runs, then the captured request at `provider.Requests()[0]` carries the added system segment — the hook received the fully-assembled request and its return value is what the provider received.
- **S-PRH-001a** — **(bite)** RED-first. Given a hook that returns `req, nil` unchanged, when the test asserts the captured system region DOES contain the hook's marker, then the assertion fails for the right reason (marker absent) — proves the property is non-vacuous.
- **S-PRH-001b** — **(bite)** RED-first. Given a hook that returns `req, nil` unchanged, when the test asserts the captured request IS byte-equal to the skeleton's (no marker), then that equality holds but the system region lacks the marker — proves the property distinguishes "no mutation" from "mutation applied".

### R-PRH-002 — Hook invocation between `buildLoopRequest` and `provider.Stream`, now as a COMPOSED CHAIN whose FINAL output reaches the provider (D2; AG-20 widening)

When `opts.PreRequestHook` is non-nil, the system SHALL invoke it with the loop's own `ctx` and the `ai.Request` produced by `buildLoopRequest`, exactly once per `Turn` call, BEFORE `provider.Stream(ctx, req)` is reached.

**AG-20 discharges the chain-composition promise this requirement's own Purpose recorded** (`spec.md:19`: *"the seam is a single callable on `TurnOptions` … **AG-20 widens to chain composition**"*). The seam is now a **composed sequence**, and the composition rule is normative:

1. `opts.PreRequestHook`, when non-nil, is invoked **first**, on the request `buildLoopRequest` produced.
2. Its returned request is the input to `opts.Hooks.PreRequest[0]`, when that slice is non-empty.
3. For each *i*, element *i*'s returned request is the input to element *i+1*.
4. **The FINAL element's returned request — and only that value — is what flows to `provider.Stream`.**

Every element SHALL be invoked exactly once per `Turn` call that reaches the request build, with the loop's own `ctx`, and the whole sequence SHALL complete before `provider.Stream` is reached. An element that returns a non-nil error aborts the sequence: no later element is invoked and no request reaches the provider (`R-PRH-003` as amended).

(Previously: the requirement closed with *"The hook's returned `ai.Request` (and only that value) is what flows to `provider.Stream`"*, a closed claim about **one** callable's return value. AG-20 composes a chain in which element *n*'s output flows to element *n+1* and only the final output reaches the provider, so the parenthetical was false on the first chained call. The clause is re-scoped to the chain's final output, and the composition order — singular field first, then the chain in registration order — is stated rather than left to inference.)

#### Scenarios

- **S-PRH-001** (cross-references the requirement; see `R-PRH-001`). *(AG-20 update: this scenario's fixture sets a zero-value `Hooks`, so it exercises the one-element case — the singular field alone — and its falsifiable claim is exactly what it was. It stays byte-unchanged and green.)*
- **S-PRH-008** — **AG-20: the chain's final output is what the provider receives, and no intermediate value is.** Given a singular `PreRequestHook` appending marker `A` and a `Hooks.PreRequest` chain of three elements appending `B`, `C` and `D` in registration order, when `Turn` runs, then the request captured at the provider carries all four markers in the order `A, B, C, D`; each element recorded, as its own input, exactly the marker set its predecessor returned; the provider recorded exactly **one** request for that attempt; and no captured request equals any intermediate element's output. Cross-referenced to `R-HKS-002` / `S-HKS-004`.

### R-PRH-003 — Hook failure aborts before I/O with a typed error ATTRIBUTED BY SOURCE NAME (D2; AG-20 widening)

When a hook returns a non-nil error, the system SHALL abort the turn BEFORE `provider.Stream` is called. The system SHALL close `sink` and return `(ai.Message{}, 0, typedErr)` mirroring the existing pre-stream-failure path (`loop.go:140-147`). The typed error SHALL reuse `ai.PreStreamFailure` (`provider_failure.go:34-93`) with a hook-attributing `FailureReport.Category`.

**AG-20 widens the attribution to name WHICH source failed, and the attribution vocabulary is SOURCE NAMES rather than a chain-wide ordinal.** With more than one possible source, "a hook failed" is no longer actionable. The two source names are:

- **`TurnOptions.PreRequestHook`** — the shipped singular field, which runs first;
- **`Hooks.PreRequest[i]`** — the chain element registered at index *i*.

**The singular field is NOT element zero of the chain.** It is a distinct, separately named source that runs *before* element zero. This settles the ambiguity between the proposal's D1 phrasing ("chain element zero") and its own success criteria ("its output feeds element 0") in favour of the latter, and the spec says so rather than leaving a reader to pick.

**A bare ordinal over the composed sequence is FORBIDDEN, and the reason is that it is what stays true under later insertion.** A composed-sequence ordinal renumbers the moment a caller inserts an element earlier in the chain, and a renumbered attribution is a lie that a green suite never catches. A registration index is stable under insertion elsewhere; a composed ordinal is not.

When an element fails, **no later element is invoked**, no request reaches the provider, and the sink drains unblocked.

(Previously: the requirement required only *"a hook-attributing `FailureReport.Category`"*, written when exactly one hook could fail. With a chain, an attribution that names no source identifies nothing.)

#### Scenarios

- **S-PRH-003** — AG-08.1 #3 failing hook aborts before I/O. Given a hook that returns `(ai.Request{}, errors.New("hook boom"))`, when `Turn` runs, then `len(provider.Requests()) == 0` (no provider call), the sink drains unblocked (channel closed), and the returned error wraps `*ai.PreStreamFailure` with a hook-attributing category — never a half-mutated request sent anyway. *(AG-20 update: this scenario's fixture sets a zero-value `Hooks` and installs only the singular field, so the source it exercises is `TurnOptions.PreRequestHook`. It stays byte-unchanged and green.)*
- **S-PRH-009** — **AG-20: the failing element is named, and the name survives the SINGULAR FIELD's own insertion or removal — corrected by `sdd-verify` (MAJOR-3) to name what actually stays stable.** Given a singular hook that succeeds and a chain of three elements whose element at index **1** returns a non-nil error, when `Turn` runs, then the provider recorded **zero** requests; the returned error wraps the typed pre-stream failure; its attribution names `Hooks.PreRequest[1]` and **not** index 2, **not** the singular field, and **not** a bare ordinal over the composed sequence; and the element at index 2 was never invoked. Given the same failing chain element at index 1, run once with **no** singular field registered and once **with** one registered running before the chain, when `Turn` runs in each case, then the recorded attribution string names `Hooks.PreRequest[1]` in **both** — unchanged by whether the singular field exists — because the singular field sits entirely outside `Hooks.PreRequest`'s own indexing. **This does NOT claim** that inserting an additional element INTO the chain ahead of index 1 leaves that element's own attribution unchanged: it does not, by design — `Hooks.PreRequest[i]` is that element's own slice index, so an element inserted ahead of it correctly renumbers it. Cross-referenced to `R-HKS-009` / `S-HKS-022` / `S-HKS-023`.

### R-PRH-004 — Hook cannot mutate loop's input in place (R-REX-001 read)

When the hook receives `req` from the loop, the loop's input value SHALL be observably unchanged after `Turn` returns. Per R-REX-001 (`request.go:325-336`), `ai.Request` is a value type whose `With(...)` rebuilds from `r.options`; the hook must not observe side effects on the loop's input. This is a direct read of the substrate's no-mutation promise.

#### Scenarios

- **S-PRH-004** — AG-08.1 #4 mutating hook leaves loop's input unchanged. Given a hook that mutates slice-from-accessor (`msgs := req.Messages(); msgs[0] = mutated`), when `Turn` runs, then the captured request at `provider.Requests()[0]` is byte-equal (via `ai.Request.Equal`, `request.go:555-627`) to the request the skeleton's `buildLoopRequest` produced — the mutation is local to the hook's accessor copy, not propagated.

### R-PRH-005 — Identity default produces byte-identical output to the AG-07 skeleton — the condition is now NIL SINGULAR **AND** EMPTY CHAIN (D4; AG-20 widening)

When `opts.PreRequestHook` is nil **and** `opts.Hooks.PreRequest` is empty, the system SHALL skip the seam and proceed to `provider.Stream(ctx, req)` unchanged. The captured request SHALL be byte-identical to what AG-07's skeleton produced for identical inputs (AG-07 `R-LSK-002` byte-stability preserved).

**When the singular field is nil and the chain is NON-EMPTY, the seam SHALL NOT be skipped**: the chain runs, starting at element 0, and its final output reaches the provider (`R-PRH-002` as amended). The nil singular field contributes the identity transform and is not invoked.

**Because `Hooks` holds function-typed fields it is NOT comparable**, so the emptiness test SHALL be an explicit predicate over the family slices, never an equality against a zero value — an equality would not compile, and a reviewer reading "when `Hooks` is zero" as an equality check would be reading code that cannot exist.

(Previously: the requirement's condition was *"When `opts.PreRequestHook` is nil, the system SHALL skip the seam"* — a closed condition on the singular field alone. AG-20 adds a second source of pre-request behavior, so the old condition would have the system skip a seam that a caller explicitly registered. The byte-identity guarantee itself is unchanged and now attaches to the "no singular hook **and** empty chain" case, which is exactly what every AG-08 scenario constructs.)

#### Scenarios

- **S-PRH-002** — AG-08.1 #2 identity default byte-identical to skeleton. Given a zero-value `TurnOptions` (no hook), when `Turn` runs against the same script AG-07's `S-LSK-001` used, then `provider.Requests()[0].Equal(buildLoopRequest(...))` is true — the seam adds zero observable behavior when not installed. *(AG-20 update: a zero-value `TurnOptions` now also carries a zero-value `Hooks`, so this scenario exercises the amended condition's true branch exactly as it always did. It stays byte-unchanged and green.)*
- **S-PRH-010** — **AG-20: a nil singular field does NOT skip a non-empty chain.** Given a `TurnOptions` whose `PreRequestHook` is **nil** and whose `Hooks.PreRequest` holds one element appending a marker, when `Turn` runs, then the captured request carries that marker — the seam was **not** skipped; the element received, as its input, a request byte-equal to `buildLoopRequest`'s output; and given the same options with the chain emptied as well, when `Turn` runs, then the captured request is byte-identical to the skeleton's and no element was invoked. Cross-referenced to `R-HKS-012` / `S-HKS-026`.

### R-PRH-006 — Prefix stability: byte-stable tools + system regions; message region grows by append

Across successive `Turn` calls with unchanged system material, unchanged tools, and unchanged hook, the system SHALL produce captured requests whose `tools` and `system` regions are byte-equal (`ai.Request.Equal`, `request.go:555-627`) and whose cascade order matches `ai.Request.CacheBoundaries()` (`cache_boundary.go:118-120`, R-ACB-007 contract). The `messages` region SHALL grow strictly by append: the new turn's first N messages are content-equal (via `ai.Message.Equal`, which excludes `MessageID` identity) to the previous turn's N messages.

#### Scenarios

- **S-PRH-005** — AG-08.2 #1 unchanged inputs yield byte-stable prefix across turns. Given two consecutive `Turn` calls with the same system, same tools, same hook, and the second transcript = first transcript + 1 appended message, when both captured requests are compared, then `tools` and `system` regions `Equal` byte-equal, `CacheBoundaries()` returns the same cascade order, and the message region in the second turn has exactly one more message — the first N are `Message.Equal` to the first turn's.

### R-PRH-007 — Hook determinism for identical inputs, extended over the COMPOSED CHAIN (AG-20 widening)

For a hook installed once and called twice with byte-equal `ai.Request` inputs, the system SHALL observe the hook's outputs are byte-equal. Combined with `R-PRH-006`, this closes the prefix-stability property: hook-applied breakpoint markers cannot oscillate between turns.

**AG-20 extends the claim over the whole composed chain, which is strictly stronger and is stated as such.** For a composition installed once — the singular field plus `Hooks.PreRequest` in registration order — and applied twice to byte-equal `ai.Request` inputs, the system SHALL observe byte-equal outputs from the **composition**. Two facts make the extension sound rather than assumed, and both are checkable:

1. **Composition of deterministic functions is deterministic**, provided the *order* is fixed; `R-HKS-006` fixes it to registration order at every point, deterministically and with no fan-out.
2. **No element observes anything but its input.** Each receives a value type and returns a value type; the loop supplies no shared mutable state, and `R-PRH-004`'s no-in-place-mutation property holds per element.

**Registration order is the whole ordering contract**: deregistration, priority, filtering and conditional registration are **not** in AG-20 (`R-HKS-006`). A milestone adding any of them re-opens this requirement, because a chain whose order can change between turns can oscillate the very prefix this requirement exists to stabilise.

(Previously: the requirement was written for a single installed callable, at a time when a chain did not exist. Left unextended, a chain of individually deterministic elements dispatched in an unpinned order would satisfy every word of it while oscillating the cache prefix between turns.)

#### Scenarios

- **S-PRH-006** — AG-08.2 #2 hook deterministic for identical inputs. Given a hook that adds a constant system segment, when the loop calls it twice with byte-equal `req` values, then both hook invocations' outputs are byte-equal (via `Request.Equal`) — hook-applied markers cannot oscillate between turns and invalidate the prefix they exist to cache. *(AG-20 update: unchanged; its fixture's `Hooks` is zero-valued.)*
- **S-PRH-011** — **AG-20: the composition is deterministic, and the order is what makes it so.** Given a composition of a singular hook plus three chain elements, each adding a constant system segment, when the loop applies the composition twice to byte-equal `ai.Request` inputs, then both compositions' outputs are byte-equal via `Request.Equal`, and the two captured requests' `tools` and `system` regions are byte-equal across two successive turns with unchanged inputs; and given the same four hooks registered in a **different** order, when the composition is applied, then its output differs from the first order's — proving the assertion is sensitive to order rather than accidentally satisfied by commutative fixtures. Cross-referenced to `R-HKS-006` / `S-HKS-016` and to `R-PRH-006` / `S-PRH-005`.

### Cross-cut scenario (AG-07 W1 carry)

- **S-PRH-007** — AG-07 W1 back-pressure: hook path with unbuffered sink + concurrent consumer. Given an unbuffered `sink` (`make(chan *Event)`) and a concurrent consumer goroutine that drains `sink` while `Turn` runs, when `Turn` returns, then `runtime.NumGoroutine()` returns to its baseline (no stranded producer) and the consumer's drain unblocks — proves the hook path supports back-pressure, not just the buffered-sink path AG-07 exercised.

## Non-functional requirements

| ID | Requirement |
|---|---|
| **NFR-PRH-001** | External-package verifiability (AG-07 W6 carry): every scenario verifiable by `cd backend/agent && make test`. Every behavioral test in `package agent_test` or another external package. |
| **NFR-PRH-002** | Determinism, race cleanliness, no-ambient-authority (AG-03.3 carry): every test MUST be deterministic, hermetic, pass under `-race`. Zero `net/http`, `os`, `os/exec`, or environment reads in **non-test sources** added by AG-08 (the `ambient_authority_test.go` guard scans non-test sources only; test files may legitimately use `os.Getenv` / `exec.Command` for substrate-untouched checks). |
| **NFR-PRH-003** | Substrate byte-unchanged (5th consecutive milestone): the 21 files in AG-07 `R-LSK-004` — `event.go`, `event_descriptor.go`, `stream_check.go`, `failure.go`, `sequence.go`, `run_events.go`, `turn_events.go`, `message_text.go`, `message_reasoning.go`, `permission_events.go`, `cost_events.go`, `delegation_events.go`, `compaction_events.go`, `tool_event.go`, `event_registry_test.go`, `doc.go`, `doc_contract_guard_test.go`, `ambient_authority_test.go`, `import_boundary_test.go`, `reconstruction_test.go`, plus `backend/agent/go.mod` and `go.sum` — are byte-unchanged. Zero new envelope / descriptor / validator / register row additions. Verified by the substrate-untouched test using `AG08_BASE_REF` env-var fallback (AG-07 PR #167 W3 fix) + dynamic merge-base. |
| **NFR-PRH-004** | Review budget: single PR under pre-authorised `size:exception` against 1000-line budget; forecast 350–600 lines added. |

## Explicit non-requirements

The list is reproduced in full; one row closes and none is removed or reordered.

- **No edit** to any substrate file in `NFR-PRH-003`. AG-08 is the 5th consecutive extensibility demonstration.
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit.
- **The other three hook points** (pre-compact, post-turn, session-start) — **CLOSED by AG-20**, which lands all three together with the chain composition this list forecast. See [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md) `R-HKS-003`, `R-HKS-004`, `R-HKS-005`. *(The row is back-annotated as closed rather than deleted: it records what AG-08 deliberately did not ship, and that record stays true.)*
- **Concrete cache-breakpoint placement** — Layer 3 wiring (doc 0004 CO-24.1). Consumes the seam; out of AG-08's scope. **Still deferred, and AG-20 does not take it**: the charter's own out-of-scope line is *"Any concrete hook implementation (Layer 3 wires them)"* (`0003:1874`).
- **Translation interface changes** — AG-07 SUGG 4 stays parked; AG-13 may re-introduce.
- **Tools / permission / retry / cost / context-check** — AG-09, AG-10, AG-11, AG-15, AG-16.
- **Append-only history discipline** — AG-12.1. AG-08 assumes the caller passes transcripts with monotonic append.
- **Value-form `Harness`** — AG-13. AG-08 ships the seam on AG-07's function form only.

## Dependencies

Reproduced in full; one row is back-annotated and none is removed.

- **Depends on**: AG-07 (`Turn`, `TurnOptions`, `buildLoopRequest`, `provider.Stream` call site — already merged at `93077c07` PR #167).
- **Depends on**: AI-12 (`Request.With(...)` copy-on-write rebuild, `request.go:325-336` — R-REX-001, PR #165).
- **Depends on**: AI-21 (`agenttest.Provider.Requests() []ai.Request`, `fake_provider.go:157-161`).
- **Depends on**: AI-22 (stream kit: `agenttest.Script`, `Emit`, `Hold`, `NewIter`).
- **Closes**: R-12 (G4 Layer 2 half — prefix stability); v2 § 6 seam 1.
- **Parallel with**: AG-09 (tool execution contract + scheduler — no dependency).
- **Blocks**: AG-20 (the four-hook taxonomy registration surface). **— DISCHARGED.** AG-20 has shipped the taxonomy: one `Hooks` registration surface on the harness, the pre-request seam widened to chain composition with the shipped singular field kept and running first, and the three remaining points. Seam 1 of v2 § 6 widens from a single callable to the full taxonomy. **The shipped field is not removed** — that is AG-23's.
- **Layer 3 consumer**: doc 0004 CO-24 (cache-breakpoint placement — out of AG-08's scope; **still out of AG-20's**).
- **Carry-forwards applied**: AG-07 W1 (unbuffered-sink, S-PRH-007); AG-07 W6 (external posture, NFR-PRH-001); AG-07 W3 (substrate-untouched env-var fix, NFR-PRH-003); AG-05 W1 (bite pattern on R-PRH-002, S-PRH-001a/001b); AG-04 W9 (scenario-count drift discipline — **and AG-20 records that this spec's own header total is the drift instance the discipline warns about; see this delta's promotion obligation**); AG-07 W2 (`make test/cover` gate); AG-07 W4 / SUGG 4 (latent — not touched).

## Verification approach

- `cd backend/agent && make test` — full `-race -v ./...` run; every scenario in the allocated range `S-PRH-001` through `S-PRH-011` plus bites green; AG-03 boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) stay green untouched.
- `cd backend/agent && make lint` — `golangci-lint` clean (cache clean if stale).
- `cd backend/agent && make build` — compile `./bin/database_administrator` (or AG-08's binary) clean.
- `cd backend/agent && make vuln-check` — no vulnerabilities.
- `cd backend/agent && make test/cover` — `loop.go` ≥ 80% line coverage (AG-04 W8 carry; AG-07 hit 85.89%; AG-08 +20–40 lines forecast stays above).
- Substrate-untouched check: NFR-PRH-003 verified by the substrate-untouched test (env-var `AG08_BASE_REF` fallback + dynamic merge-base).

## Evidence discipline

`openspec/config.yaml` `apply.tdd: true`; strict TDD active (`openspec/AGENTS.md`).

- **R-PRH-002 / S-PRH-001 closes only on its recorded bites** (`S-PRH-001a`, `S-PRH-001b`): RED-recorded BEFORE `S-PRH-001` is GREEN. Mirrors AG-05 `S-AMT-071`/`S-AMT-072` and AG-07 `S-LSK-003a`/`S-LSK-003b` bite pattern.
- **AG-07 W1 closed**: `S-PRH-007` adds the unbuffered-sink path AG-07 didn't exercise.
- **AG-07 W3 closed**: substrate-untouched test uses the env-var fallback shipped in AG-07 PR #167.
- **No new event kinds** registered by AG-08 — the every-kind-constructible guard stays at 25 (AG-07 invariant).
- **Scope-fence remains at 25.** AG-08 extends via `NFR-PRH-003`'s 5th consecutive extensibility demonstration, not by editing the registry.
- **Strict TDD skill gap** — `openspec/AGENTS.md` `## Strict TDD is on` is the inline fallback.

## Acceptance criteria

1. Every `S-PRH-001`…`S-PRH-007` has recorded evidence.
2. `cd backend/agent && make test`, `make lint` (after `cache clean`), `make build`, and `make vuln-check` are all green.
3. `backend/agent/go.mod` and `go.sum` byte-unchanged (NFR-PRH-004).
4. The 21 substrate files in `NFR-PRH-003` are byte-unchanged (5th consecutive milestone).
5. `loop.go` line coverage ≥ 80% (AG-04 W8).
6. The every-kind-constructible guard still passes at 25 kinds; AG-03's two boundary guards pass with zero changes.
7. The 6 charter `AG-08.1`/`AG-08.2` Gherkin scenarios are covered; none reduced.
8. Both bites (`S-PRH-001a`, `S-PRH-001b`) RED-recorded with failing output in `apply-progress.md`.
9. Scenario count stated as allocated range `S-PRH-001` through `S-PRH-011` plus bites `S-PRH-001a`, `S-PRH-001b`, per `S-LSK-020`/AG-20 drift-prevention treatment (amended 2026-08-20 archive phase from literal "9 total").

## Traceability

| Requirement | Charter node | Decisions cited | Charter scenario (`0003`) |
|---|---|---|---|
| `R-PRH-001` | AG-08 cross-cut | D1, D3 | surface (TurnOptions field) |
| `R-PRH-002` | AG-08.1 | D2, D5 | `:858-861` (hook sees + shapes) |
| `R-PRH-003` | AG-08.1 | D2 | `:869-871` (failing hook aborts) |
| `R-PRH-004` | AG-08.1 | R-REX-001 | `:873-876` (no mutation) |
| `R-PRH-005` | AG-08.1 | D4 | `:863-866` (identity default) |
| `R-PRH-006` | AG-08.2 | R-ACB-007 | `:888-892` (prefix stability) |
| `R-PRH-007` | AG-08.2 | D1 | `:894-897` (hook deterministic) |

All 6 charter Gherkin scenarios are represented; none is reduced. Scenario count stated as the allocated range `S-PRH-001` through `S-PRH-011` plus bites `S-PRH-001a`, `S-PRH-001b`, per the `S-LSK-020`/AG-20 drift-prevention treatment (amended 2026-08-20; replaces the literal "9 total").

References: `docs/architecture/0001-cachicamas-agent-stack-v2.md:653` (seam 1); `:406` (cache cascade); `:698` (G4 row); `0003:833-900` (AG-08 charter); `request.go:325-336` (R-REX-001 / `With`); `:555-627` (`Request.Equal`); `cache_boundary.go:118-120` (R-ACB-007 cascade order); `provider_failure.go:34-93` (`PreStreamFailure`); `fake_provider.go:157-161` (`Requests()`).
