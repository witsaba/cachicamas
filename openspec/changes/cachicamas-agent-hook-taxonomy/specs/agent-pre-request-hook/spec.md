# Delta for `agent-pre-request-hook` — AG-20 discharges AG-08's chain-composition promise, and three closed claims are re-scoped rather than left FALSE

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5), `0003:1864-1918`
> **Modifies**: `agent-pre-request-hook` ([`../../../../specs/agent-pre-request-hook/spec.md`](../../../../specs/agent-pre-request-hook/spec.md)) — `R-PRH-002` (`spec.md:33-39`), `R-PRH-003` (`spec.md:41-47`), `R-PRH-005` (`spec.md:57-63`) and `R-PRH-007` (`spec.md:73-79`).
> **This delta amends claims AG-20 GENUINELY FALSIFIES, and each falsifying clause is quoted.** All three are the same repository failure shape — a closed, universally quantified sentence written before the producer that would violate it existed:
> 1. `R-PRH-002`'s *"The hook's returned `ai.Request` (**and only that value**) is what flows to `provider.Stream`"* (`spec.md:35`). Under a chain, element *n*'s output flows to element *n+1*, and only the **final** element's output reaches the provider. Read literally the sentence is false on the first chained call.
> 2. `R-PRH-005`'s *"When `opts.PreRequestHook` is nil, the system SHALL **skip the seam**"* (`spec.md:59`). False the moment the singular field is nil and `Hooks.PreRequest` is non-empty — the seam runs, and it must.
> 3. `R-PRH-003`'s hook-attributing failure category, which names no source once there is more than one possible source.
> **`R-PRH-007` is widened rather than falsified**: determinism was written for one callable and must now hold over the composed chain, which is a strictly stronger claim and is stated as such.
> **`R-PRH-001`, `R-PRH-004` and `R-PRH-006` are UNAMENDED and shown still true** in the table below, opened rather than cited.
> **Ownership**: the chain, the `Hooks` type and the attribution vocabulary are owned by [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md) (`R-HKS-002`, `R-HKS-009`). This delta owns only what the pre-request seam's own contract must now say.
> **Header maintenance obligation at promotion.** The target spec's header carries a **scenario total** (`spec.md:7`: *"6 charter → 7 spec + 2 bites = 9 total"*) and its Coverage table (`spec.md:11-15`) restates it. That total goes **silently false** when this delta's scenarios are promoted. `sdd-archive` MUST, in the same commit that promotes this delta, replace the total with the **allocated ID range** and the named bites — the `S-LSK-020` treatment already applied to `agent-loop-skeleton` — rather than incrementing a number that no test defends. The acceptance line at `spec.md:147` restating the same total MUST be re-scoped in that same commit.
> **Every `file:line` cited below was opened in this worktree during this phase, at `main@2a138b59`.**

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-PRH-001` — the `TurnOptions.PreRequestHook` field and its nil-identity default | **Kept, byte-unchanged, and superseded in prose only.** AG-20 adds a field beside it; it removes nothing and changes no signature. It is **not** marked with Go's `Deprecated:` convention, which would make every existing internal reference a lint finding for no behavioral gain. Removal is **AG-23**'s (`R-HKS-010` consequence 2) |
| `R-PRH-004` — the hook cannot mutate the loop's input in place | **Untouched.** The property is a read of Layer 1's copy-on-write request rebuild (`R-REX-001`), which chaining does not weaken: each element receives a value and returns a value, and no element observes a side effect on another's input |
| `R-PRH-006` — prefix stability across turns | **Untouched.** The claim is about the *shape* of successive captured requests, not about how many callables produced them. A chain of deterministic elements is deterministic (`R-PRH-007` as amended), so the stable-prefix property is unaffected |
| `S-PRH-001`, `S-PRH-001a`, `S-PRH-001b`, `S-PRH-002`, `S-PRH-003`, `S-PRH-004`, `S-PRH-005`, `S-PRH-006`, `S-PRH-007` | **All nine stay byte-unchanged and green.** Every one constructs a `TurnOptions` whose `Hooks` field is zero-valued, so each exercises exactly the single-callable case it always exercised. `R-HKS-002` / `S-HKS-005` asserts that these files are byte-unchanged at the merge-base diff — the compatibility proof is an **unedited suite**, not a claim about one |
| `NFR-PRH-001`…`NFR-PRH-004` | **Untouched.** AG-20 adds no dependency, edits no substrate file (`NFR-HKS-004`), and ships under the same single-PR exception posture |

## MODIFIED Requirements

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

---

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
- **S-PRH-009** — **AG-20: the failing element is named, and the name survives insertion.** Given a singular hook that succeeds and a chain of three elements whose element at index **1** returns a non-nil error, when `Turn` runs, then the provider recorded **zero** requests; the returned error wraps the typed pre-stream failure; its attribution names `Hooks.PreRequest[1]` and **not** index 2, **not** the singular field, and **not** a bare ordinal over the composed sequence; and the element at index 2 was never invoked. Given the same failing element with two **additional** elements inserted **before** it, so its composed position moves, when `Turn` runs, then the recorded attribution string is **unchanged**. Cross-referenced to `R-HKS-009` / `S-HKS-022` / `S-HKS-023`.

---

### R-PRH-005 — Identity default produces byte-identical output to the AG-07 skeleton — the condition is now NIL SINGULAR **AND** EMPTY CHAIN (D4; AG-20 widening)

When `opts.PreRequestHook` is nil **and** `opts.Hooks.PreRequest` is empty, the system SHALL skip the seam and proceed to `provider.Stream(ctx, req)` unchanged. The captured request SHALL be byte-identical to what AG-07's skeleton produced for identical inputs (AG-07 `R-LSK-002` byte-stability preserved).

**When the singular field is nil and the chain is NON-EMPTY, the seam SHALL NOT be skipped**: the chain runs, starting at element 0, and its final output reaches the provider (`R-PRH-002` as amended). The nil singular field contributes the identity transform and is not invoked.

**Because `Hooks` holds function-typed fields it is NOT comparable**, so the emptiness test SHALL be an explicit predicate over the family slices, never an equality against a zero value — an equality would not compile, and a reviewer reading "when `Hooks` is zero" as an equality check would be reading code that cannot exist.

(Previously: the requirement's condition was *"When `opts.PreRequestHook` is nil, the system SHALL skip the seam"* — a closed condition on the singular field alone. AG-20 adds a second source of pre-request behavior, so the old condition would have the system skip a seam that a caller explicitly registered. The byte-identity guarantee itself is unchanged and now attaches to the "no singular hook **and** empty chain" case, which is exactly what every AG-08 scenario constructs.)

#### Scenarios

- **S-PRH-002** — AG-08.1 #2 identity default byte-identical to skeleton. Given a zero-value `TurnOptions` (no hook), when `Turn` runs against the same script AG-07's `S-LSK-001` used, then `provider.Requests()[0].Equal(buildLoopRequest(...))` is true — the seam adds zero observable behavior when not installed. *(AG-20 update: a zero-value `TurnOptions` now also carries a zero-value `Hooks`, so this scenario exercises the amended condition's true branch exactly as it always did. It stays byte-unchanged and green.)*
- **S-PRH-010** — **AG-20: a nil singular field does NOT skip a non-empty chain.** Given a `TurnOptions` whose `PreRequestHook` is **nil** and whose `Hooks.PreRequest` holds one element appending a marker, when `Turn` runs, then the captured request carries that marker — the seam was **not** skipped; the element received, as its input, a request byte-equal to `buildLoopRequest`'s output; and given the same options with the chain emptied as well, when `Turn` runs, then the captured request is byte-identical to the skeleton's and no element was invoked. Cross-referenced to `R-HKS-012` / `S-HKS-026`.

---

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

## MODIFIED Explicit non-requirements

The list is reproduced in full; one row closes and none is removed or reordered.

- **No edit** to any substrate file in `NFR-PRH-003`. AG-08 is the 5th consecutive extensibility demonstration.
- **No new top-level Go deps.** No `go.mod`/`go.sum` edit. No AG-03 guard edit.
- **The other three hook points** (pre-compact, post-turn, session-start) — **CLOSED by AG-20**, which lands all three together with the chain composition this list forecast. See [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md) `R-HKS-003`, `R-HKS-004`, `R-HKS-005`. *(The row is back-annotated as closed rather than deleted: it records what AG-08 deliberately did not ship, and that record stays true.)*
- **Concrete cache-breakpoint placement** — Layer 3 wiring (doc 0004 CO-24.1). Consumes the seam; out of AG-08's scope. **Still deferred, and AG-20 does not take it**: the charter's own out-of-scope line is *"Any concrete hook implementation (Layer 3 wires them)"* (`0003:1874`).
- **Translation interface changes** — AG-07 SUGG 4 stays parked; AG-13 may re-introduce.
- **Tools / permission / retry / cost / context-check** — AG-09, AG-10, AG-11, AG-15, AG-16.
- **Append-only history discipline** — AG-12.1. AG-08 assumes the caller passes transcripts with monotonic append.
- **Value-form `Harness`** — AG-13. AG-08 ships the seam on AG-07's function form only.

## MODIFIED Dependencies

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
