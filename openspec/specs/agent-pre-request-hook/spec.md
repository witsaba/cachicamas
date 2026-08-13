# Spec — The pre-request hook seam (`agent-pre-request-hook`)

> **Change**: `cachicamas-agent-pre-request-hook` · **AG-08** (Layer 2, Wave 2, milestone 8 of 24) of [doc 0003](../../../docs/architecture/milestones/0003-cachicamas-agent-layer-2-task-graph.md#ag-08--add-the-pre-request-hook-seam), `0003:833-900`
> **Nodes**: AG-08.1 `[leaf]` (hook seam) · AG-08.2 `[leaf]` (prefix stability)
> **Format**: Given/When/Then + RFC 2119 per `openspec/config.yaml` `## rules.specs`. Every scenario independently verifiable.
> **IDs**: `R-PRH-NNN` / `S-PRH-NNN`. Distinct from `R-AEV-`/`S-AEV-`, `R-AMT-`/`S-AMT-`, `R-APE-`/`S-APE-`, `R-AGE-`/`S-AGE-`, `R-LSK-`/`S-LSK-`.
> **Scenario count** (AG-04 W9): **6 charter → 7 spec + 2 bites = 9 total** in 7 requirements.

## Coverage

| Charter | Requirements | Spec | Bites |
|---|---|---|---|
| **6 of 6** | 7 (`R-PRH-001`–`007`) | **7** (`S-PRH-001`…`S-PRH-007`) | **2** (`S-PRH-001a`, `S-PRH-001b`) |

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

### R-PRH-002 — Hook invocation between `buildLoopRequest` and `provider.Stream` (D2)

When `opts.PreRequestHook` is non-nil, the system SHALL invoke it with the loop's own `ctx` and the `ai.Request` produced by `buildLoopRequest`, exactly once per `Turn` call, BEFORE `provider.Stream(ctx, req)` is reached. The hook's returned `ai.Request` (and only that value) is what flows to `provider.Stream`.

#### Scenarios

- **S-PRH-001** (cross-references the requirement; see `R-PRH-001`).

### R-PRH-003 — Hook failure aborts before I/O with typed error (D2)

When the hook returns a non-nil error, the system SHALL abort the turn BEFORE `provider.Stream` is called. The system SHALL close `sink` and return `(ai.Message{}, 0, typedErr)` mirroring the existing pre-stream-failure path (`loop.go:140-147`). The typed error SHALL reuse `ai.PreStreamFailure` (`provider_failure.go:34-93`) with a hook-attributing `FailureReport.Category`.

#### Scenarios

- **S-PRH-003** — AG-08.1 #3 failing hook aborts before I/O. Given a hook that returns `(ai.Request{}, errors.New("hook boom"))`, when `Turn` runs, then `len(provider.Requests()) == 0` (no provider call), the sink drains unblocked (channel closed), and the returned error wraps `*ai.PreStreamFailure` with a hook-attributing category — never a half-mutated request sent anyway.

### R-PRH-004 — Hook cannot mutate loop's input in place (R-REX-001 read)

When the hook receives `req` from the loop, the loop's input value SHALL be observably unchanged after `Turn` returns. Per R-REX-001 (`request.go:325-336`), `ai.Request` is a value type whose `With(...)` rebuilds from `r.options`; the hook must not observe side effects on the loop's input. This is a direct read of the substrate's no-mutation promise.

#### Scenarios

- **S-PRH-004** — AG-08.1 #4 mutating hook leaves loop's input unchanged. Given a hook that mutates slice-from-accessor (`msgs := req.Messages(); msgs[0] = mutated`), when `Turn` runs, then the captured request at `provider.Requests()[0]` is byte-equal (via `ai.Request.Equal`, `request.go:555-627`) to the request the skeleton's `buildLoopRequest` produced — the mutation is local to the hook's accessor copy, not propagated.

### R-PRH-005 — Identity default produces byte-identical output to AG-07 skeleton (D4)

When `opts.PreRequestHook` is nil, the system SHALL skip the seam and proceed to `provider.Stream(ctx, req)` unchanged. The captured request SHALL be byte-identical to what AG-07's skeleton produced for identical inputs (AG-07 R-LSK-002 byte-stability preserved).

#### Scenarios

- **S-PRH-002** — AG-08.1 #2 identity default byte-identical to skeleton. Given a zero-value `TurnOptions` (no hook), when `Turn` runs against the same script AG-07's `S-LSK-001` used, then `provider.Requests()[0].Equal(buildLoopRequest(...))` is true — the seam adds zero observable behavior when not installed.

### R-PRH-006 — Prefix stability: byte-stable tools + system regions; message region grows by append

Across successive `Turn` calls with unchanged system material, unchanged tools, and unchanged hook, the system SHALL produce captured requests whose `tools` and `system` regions are byte-equal (`ai.Request.Equal`, `request.go:555-627`) and whose cascade order matches `ai.Request.CacheBoundaries()` (`cache_boundary.go:118-120`, R-ACB-007 contract). The `messages` region SHALL grow strictly by append: the new turn's first N messages are content-equal (via `ai.Message.Equal`, which excludes `MessageID` identity) to the previous turn's N messages.

#### Scenarios

- **S-PRH-005** — AG-08.2 #1 unchanged inputs yield byte-stable prefix across turns. Given two consecutive `Turn` calls with the same system, same tools, same hook, and the second transcript = first transcript + 1 appended message, when both captured requests are compared, then `tools` and `system` regions `Equal` byte-equal, `CacheBoundaries()` returns the same cascade order, and the message region in the second turn has exactly one more message — the first N are `Message.Equal` to the first turn's.

### R-PRH-007 — Hook determinism for identical inputs

For a hook installed once and called twice with byte-equal `ai.Request` inputs, the system SHALL observe the hook's outputs are byte-equal. Combined with R-PRH-006, this closes the prefix-stability property: hook-applied breakpoint markers cannot oscillate between turns.

#### Scenarios

- **S-PRH-006** — AG-08.2 #2 hook deterministic for identical inputs. Given a hook that adds a constant system segment, when the loop calls it twice with byte-equal `req` values, then both hook invocations' outputs are byte-equal (via `Request.Equal`) — hook-applied markers cannot oscillate between turns and invalidate the prefix they exist to cache.

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

- **No edit** to any substrate file in `NFR-PRH-003`. AG-08 is the 5th consecutive extensibility demonstration.
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit.
- **The other three hook points** (pre-compact, post-turn, session-start) — AG-20 widens to the chain composition.
- **Concrete cache-breakpoint placement** — Layer 3 wiring (doc 0004 CO-24.1). Consumes the seam; out of AG-08's scope.
- **Translation interface changes** — AG-07 SUGG 4 stays parked; AG-13 may re-introduce.
- **Tools / permission / retry / cost / context-check** — AG-09, AG-10, AG-11, AG-15, AG-16.
- **Append-only history discipline** — AG-12.1. AG-08 assumes the caller passes transcripts with monotonic append.
- **Value-form `Harness`** — AG-13. AG-08 ships the seam on AG-07's function form only.

## Dependencies

- **Depends on**: AG-07 (`Turn`, `TurnOptions`, `buildLoopRequest`, `provider.Stream` call site — already merged at `93077c07` PR #167).
- **Depends on**: AI-12 (`Request.With(...)` copy-on-write rebuild, `request.go:325-336` — R-REX-001, PR #165).
- **Depends on**: AI-21 (`agenttest.Provider.Requests() []ai.Request`, `fake_provider.go:157-161`).
- **Depends on**: AI-22 (stream kit: `agenttest.Script`, `Emit`, `Hold`, `NewIter`).
- **Closes**: R-12 (G4 Layer 2 half — prefix stability); v2 § 6 seam 1.
- **Parallel with**: AG-09 (tool execution contract + scheduler — no dependency).
- **Blocks**: AG-20 (the four-hook taxonomy registration surface).
- **Layer 3 consumer**: doc 0004 CO-24 (cache-breakpoint placement — out of AG-08's scope).
- **Carry-forwards applied**: AG-07 W1 (unbuffered-sink, S-PRH-007); AG-07 W6 (external posture, NFR-PRH-001); AG-07 W3 (substrate-untouched env-var fix, NFR-PRH-003); AG-05 W1 (bite pattern on R-PRH-002, S-PRH-001a/001b); AG-04 W9 (scenario-count drift discipline); AG-07 W2 (`make test/cover` gate); AG-07 W4 / SUGG 4 (latent — not touched).

## Verification approach

- `cd backend/agent && make test` — full `-race -v ./...` run; all 9 scenarios + bites green; AG-03 boundary guards (`import_boundary_test.go`, `ambient_authority_test.go`) stay green untouched.
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
9. `6 charter → 7 spec + 2 bites = 9 total` scenario count stated identically with the proposal, tasks, apply-progress, and verify-report.

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

All 6 charter Gherkin scenarios are represented; none is reduced. Scenario count stated identically with the proposal (`6 charter → 7 spec + 2 bites = 9 total`).

References: `docs/architecture/0001-cachicamas-agent-stack-v2.md:653` (seam 1); `:406` (cache cascade); `:698` (G4 row); `0003:833-900` (AG-08 charter); `request.go:325-336` (R-REX-001 / `With`); `:555-627` (`Request.Equal`); `cache_boundary.go:118-120` (R-ACB-007 cascade order); `provider_failure.go:34-93` (`PreStreamFailure`); `fake_provider.go:157-161` (`Requests()`).
