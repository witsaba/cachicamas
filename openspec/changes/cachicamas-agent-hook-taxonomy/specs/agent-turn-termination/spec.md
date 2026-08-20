# Delta for `agent-turn-termination` — the turn-outcome vocabulary gains a CONSUMER and NO member

> **Change**: `cachicamas-agent-hook-taxonomy` · **AG-20** (Layer 2, Wave 5), `0003:1864-1918`
> **Modifies**: `agent-turn-termination` ([`../../../../specs/agent-turn-termination/spec.md`](../../../../specs/agent-turn-termination/spec.md)) — **by APPENDING one requirement**, `R-ATT-010`, alongside `R-ATT-001` (the outcome vocabulary) and `R-ATT-009` (the substrate release). **Both are byte-unchanged.**
> **BACK-ANNOTATION ONLY.** No outcome member is added, removed or renumbered; no `String()` form changes; no constructor signature changes; the failure-iff-aborted rule is untouched. `turn_events.go` and `failure.go` — released for **AG-11 only** — are forbidden again at AG-20 and MUST be byte-unchanged (`R-LSK-008`).
> **Why an ADDED requirement rather than a MODIFIED `R-ATT-001`.** `R-ATT-001` carries the whole outcome vocabulary and its `String()` forms; a `MODIFIED` block must reproduce it in full or the archive step silently drops what it omits, and there is nothing in it to change. `R-ATT-010` is the next free requirement identifier (`R-ATT-001`…`R-ATT-009` are allocated); `S-ATT-015` is the next free scenario identifier (`S-ATT-001`…`S-ATT-014` are allocated, verified in this phase).
> **Ownership**: the post-turn report is owned by [`../agent-hook-taxonomy/spec.md`](../agent-hook-taxonomy/spec.md) (`R-HKS-004`). This delta owns only the outcome vocabulary's audit statement.

## Not modified, and why

| Not modified | Why |
|---|---|
| `R-ATT-001` — the outcome members and their `String()` forms | **Byte-unchanged.** AG-20 **reads** the outcome from the forwarded turn-close payload and adds **no member**. A new member would reopen `turn_events.go`'s `iota` block, its bound check and its `String()` switch, all of which are forbidden here |
| `R-ATT-002`'s failure-iff-aborted rule and `NewTurnEnd`'s signature | **Untouched.** AG-20 constructs no turn-close event; it reads one the forwarder already carries |
| `R-ATT-006` — the partial-output accessor on the typed failure | **Untouched, and `failure.go` is byte-unchanged.** AG-20's stall report is declared in the new hook file, **not** in `failure.go` — a placement declined explicitly, because editing `failure.go` would pass the mechanical substrate filter silently while violating `R-LSK-004`'s prose |
| `R-ATT-009` — AG-11's exact-filename release | **Not extended.** AG-11's release of `turn_events.go` and `failure.go` is scoped to AG-11; both are forbidden again at AG-20 |

## ADDED Requirements

### R-ATT-010 — AG-20 CONSUMES the turn-outcome vocabulary and adds no member, and the consumption is on FOURTEEN enumerated exits

**AG-20 is the outcome vocabulary's first non-stream consumer**, and that is the whole of its interaction with this capability. The per-logical-turn observation report carries the turn's outcome as a payload field, read from the turn-close event the per-attempt forwarder already handles. Three consequences MUST be stated rather than inferred:

1. **No member is added, and the reason is that none is needed.** Every exit AG-20 reports on maps onto an outcome the vocabulary already carries: a completed logical turn reports the finished outcome, and a failed, interrupted or wound-down one reports the aborted outcome, in each case the value the turn's own close event carried. An outcome member meaning "observed by a hook" would be a category error — the report is *about* a turn, it is not a turn state.

2. **The read is a PURE READ downstream of the forwarder, so the outcome the report carries is by construction the outcome the STREAM carries.** They cannot diverge: there is one source. A reviewer checking a report against a recorded stream is checking an identity, not a correspondence, and `S-ATT-015` asserts it that way.

3. **The consumption is total over the logical-turn loop's exits, and the enumeration is normative in `R-HKS-004`.** Fourteen exits are enumerated there with an explicit fires/does-not-fire verdict each, collapsing to four enqueue sites so that no exit can fire twice and no firing exit can be skipped by an early return above it. This capability's contribution is the guarantee the report leans on: **a logical turn that ran has exactly one turn-close event carrying exactly one outcome**, so "the turn's outcome" is well defined for every yes-row and undefined — and therefore not reported — for every no-row.

**This requirement adds no obligation to any producer.** It records what AG-20 reads, so that a later milestone contemplating a new outcome member knows a second consumer exists.

#### Scenarios

- **S-ATT-015** — **AG-20: the reported outcome IS the streamed outcome, and no member was added.** Given six runs driving the finished, failed, interrupted and wound-down logical-turn exits, with a post-turn observer recording every report, when each run completes and its lane has drained, then each report's outcome is **identical** to the outcome carried by that turn's own close event on the recorded stream, compared value-to-value rather than through a rendered string; and when the outcome vocabulary is enumerated through the public surface, then its member set and each member's rendered form are exactly what they were at the merge base, `turn_events.go` and `failure.go` are byte-unchanged, and the turn-close constructor's signature and the failure-iff-aborted rule are unchanged. Cross-referenced to `R-HKS-004` / `S-HKS-010` and `R-HKS-010` / `S-HKS-024`.
