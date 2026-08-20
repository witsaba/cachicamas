# Delta for `agent-tool-scheduler` — the seam is installed in `executeCall` with ZERO type assertions, and seam 3 survives intact

> **Change**: `cachicamas-agent-delegation-readiness` · **AG-19** (Layer 2, Wave 5), `0003:1793-1862`
> **Modifies**: `agent-tool-scheduler` ([`../../../../specs/agent-tool-scheduler/spec.md`](../../../../specs/agent-tool-scheduler/spec.md)) — `R-TLS-002` (`spec.md:33-40`) and the non-requirement row at `spec.md:193`.
> **BACK-ANNOTATION PLUS ONE NEW SCENARIO. `R-TLS-002` is UNAMENDED in substance** — the requirement text ships word-for-word and its source guard passes **unchanged**. What it gains is the statement that AG-19 installs something onto the tool's context *without* touching the seam this requirement protects.
> **Why this delta is not optional.** AG-19 modifies `scheduler.go` — three lines inside `executeCall` — and `R-TLS-002`'s guard **scans `scheduler.go` as raw bytes** for type assertions (`scheduler_test.go:616-650`). A reviewer meeting a `scheduler.go` diff in a milestone about re-entrancy will reasonably ask whether seam 3 was overloaded. The answer is no, and this delta is where that answer is recorded and made checkable rather than asserted.
> **The one thing this delta must prevent being read into it**: AG-19 does **not** carry the seam in `PolicySlot`. Overloading `PolicySlot` was considered and **ruled out entirely** — its single documented meaning is the permission slot (`scheduler.go:466-471`), and the source guard exists to keep that meaning single.
> **Ownership**: the seam is owned by [`../agent-delegation-readiness/spec.md`](../agent-delegation-readiness/spec.md) (`R-DEL-001`, `R-DEL-003`). This delta owns only the scheduler contract's audit statement.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-TLS-002`'s requirement text and `S-TLS-002` / `S-TLS-002a` | **Byte-unchanged.** The guard passes with zero change to its own source, and the byte-equality assertion on the forwarded policy value is untouched |
| `R-TLS-003`'s `EffectClass` vocabulary, `R-TLS-004`'s concurrency policy | **Untouched.** AG-19 adds no effect class and changes no scheduling rule; its siblings scenario uses ordinary read-class calls |
| `R-TLS-014` / `R-CAN-006`'s detach reporting and the wind-down bound | **Untouched and deliberately not re-litigated.** AG-19 asserts its hosted call is **not** detached rather than asserting what a detach looks like; the detach path stays owned here |
| The sink-ownership flag rule — exactly one close per sink, by exactly one owner | **Confirmed.** Each child harness owns and closes its own sink; the parent's scheduler never sees it |
| `Tool.Run`'s signature (`tool.go:182-186`) | **Byte-unchanged**, and that is the reason the seam rides the context at all: changing the signature would break every existing implementation and is a change AG-19's dependency set does not authorise |

## MODIFIED Requirements

### R-TLS-002 — `PolicySlot` is opaque to the scheduler (D3, seam 3)

The system SHALL define `PolicySlot` as a named type over `any` (`type PolicySlot any`). The scheduler MUST NOT type-assert, type-switch, or read the value of `PolicySlot`; it MUST forward the exact bytes/identity of the injected value to the tool's `Run`. This is enforced by (a) source guard test scanning `scheduler.go` for any type assertion on `PolicySlot`, and (b) scripted-tool byte-equality assertion via `bytes.Equal`.

**Back-annotation (AG-19) — a second per-call value now rides the tool's context, and seam 3 is untouched by it.** AG-19 installs a publishing seam onto the context handed to `tool.Run`, inside `executeCall` and around the wind-down call. Five constraints keep this requirement literally true, and each is checkable rather than asserted:

- **`PolicySlot` is not the carrier.** The seam has its own unexported context key and its own type. Reusing `PolicySlot` would overload a seam whose single documented meaning is the permission slot (`scheduler.go:466-471`) — considered and rejected outright.
- **`scheduler.go` gains ZERO type assertions of any kind.** The seam's single assertion lives in the seam's own file, on the seam's own type; even the `context.WithValue` call is placed behind an unexported installer so the scheduler source names neither the key nor the type. The source guard passes with **zero change to its own source**.
- **The forwarded policy value is untouched.** `executeCall` still hands `PolicySlot(call.ID())` through unread and unmodified; `S-TLS-002`'s byte-equality assertion holds.
- **The scheduler learns nothing about delegation.** It installs an opaque value and revokes it; it never inspects what a tool publishes, never counts children, and holds no delegation state.
- **The addition is per-call and bounded.** The seam exists for exactly one `tool.Run` frame and is revoked on **every** exit path — normal return, detached return and re-panic — so no scheduler-owned state outlives the call (`R-DEL-003`).

(Previously: the requirement stated the opacity rule against a scheduler that installed nothing on the tool's context at all, so a reader meeting AG-19's `scheduler.go` diff had no recorded basis for deciding whether seam 3 had been overloaded.)

#### Scenarios

- **S-TLS-002** — AG-09.1 #2 policy slot passes through opaquely. Given a caller injecting `PolicySlot(sandboxBytes)` (e.g., a `[]byte` payload a Layer 3 sandbox would interpret), when the scripted tool's `Run` records the received `policy` value, then `bytes.Equal(recorded, injected)` is true byte-for-byte and the scheduler source contains zero type assertions on `PolicySlot`. *(AG-19 update: the assertion is unchanged and now also runs against a scheduler that installs a delegation seam on the same call's context — the two per-call values are independent, and this scenario proves the policy one is still forwarded byte-exact.)*
- **S-TLS-002a** — **(bite)** RED-first. Given a scheduler implementation that strips the type tag (`policy = PolicySlot(underlying)`), when the byte-equality assertion runs, then it FAILS for the right reason (tag stripped) — proves the property is non-vacuous. RED-recorded BEFORE `S-TLS-002` is GREEN.
- **S-TLS-020** — **AG-19: the seam rides beside `PolicySlot`, not inside it, and the guard proves it.** Given the merged AG-19 change, when `scheduler.go` is read as raw bytes, then it contains **no** type assertion of any kind added by this change and **no** occurrence of the seam's context key or interface type; when the `PolicySlot` source guard runs, then it passes with its own source byte-unchanged; and when a tool records both the `policy` value it received and the seam it obtained from its context, then the policy value is byte-identical to the injected one and the seam is a distinct value obtained through its own accessor. Cross-referenced to `R-DEL-001` / `S-DEL-003`.

## MODIFIED Explicit non-requirements

The list is reproduced only where AG-19 touches it; every other row is unchanged and none is removed.

- **Subagent tool** — was: *"v1 non-goal per doc 0003 § 8."* **STILL A V1 NON-GOAL, and AG-19 is not its delivery — this row is REAFFIRMED rather than closed.** AG-19 installs a publishing seam in `executeCall` and proves the harness is re-entrant, but it ships no subagent tool, no subagent configuration and no depth limit; those stay post-v1 on the substrate AG-19 proved (`0003:1803`). The enforcement is structural: the seam names no subagent concept, its concrete type and installer are unexported so no code outside the package can mint one, and every subagent concept lives in `package agent_test`, which production code cannot import. **A reader must not take AG-19's `executeCall` diff as this row closing.** What the scheduler gained is a per-call opaque context value and its revocation — nothing that knows what a subagent is.
