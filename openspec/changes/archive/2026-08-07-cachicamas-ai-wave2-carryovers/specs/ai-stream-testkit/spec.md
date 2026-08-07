# Delta for `ai-stream-testkit` — the carryover ledger reads *discharged*, not *assigned*

> **Change**: `cachicamas-ai-wave2-carryovers` · **Milestone**: AI-41 (doc 0002:2233–2257) · **Wave 5 — Harden**
> **Status**: **delta** — amends [`openspec/specs/ai-stream-testkit/spec.md`](../../../../specs/ai-stream-testkit/spec.md) § "Delta status against existing specs", the `W1`/`W2` bullet at line 39, append-only at archive time
> **Type**: Delta — **no `ADDED`, no `MODIFIED`, no `REMOVED`, no `RENAMED` requirement**. This capability's thirteen requirements (`R-STK-001` … `R-STK-013`) are untouched. The amendment is to the spec's carryover ledger, which records ownership rather than behavior
> **Sources**: [proposal.md](../../proposal.md) · [explore.md](../../explore.md) · charter Acceptance (doc 0002:2233–2257)

---

## Why this file exists

`openspec/specs/ai-stream-testkit/spec.md:39` is the **ledger entry** that names AI-41 as the owner of the two Wave-2 carryovers. It is the single line that let `W1` and `W2` survive Waves 2 and 3 in the open: a third silent pass is the failure mode this milestone exists to stop. Discharging the carryovers in code while leaving the ledger reading *assigned* would reproduce that failure mode exactly — and `sdd-verify` would have no artifact to check the ledger against. This delta therefore fixes the target wording in advance, so the archive step is a transcription, not a judgement call.

The amendment MUST be **append-only**, per the discipline the sibling specs already follow (`ai-provider-conformance-suite:242`, `ai-event-envelope:105`): the existing sentences stay verbatim, because the record of what was open and who took it is history worth keeping. Only a dated discharge clause is added.

---

## The amendment

### Current text (line 39, quoted for the archive step's diff)

> - **W1/W2, the two Wave-2 carryovers AI-21 parked** (the `CheckEmit` rule 4 failure-path gap, the missing redacting `GoString()` on the failure payload). **Assigned 2026-08-03 (AI-24):** doc 0002 appends **AI-41 — Discharge the Wave-2 carryovers** (Wave 5) as owner; AI-22 still assigns neither.

### Target text (the exact wording to write at archive)

> - **W1/W2, the two Wave-2 carryovers AI-21 parked** (the `CheckEmit` rule 4 failure-path gap, the missing redacting `GoString()` on the failure payload). **Assigned 2026-08-03 (AI-24):** doc 0002 appends **AI-41 — Discharge the Wave-2 carryovers** (Wave 5) as owner; AI-22 still assigns neither. **Discharged 2026-08-07 (AI-41)** by `cachicamas-ai-wave2-carryovers`: both are closed — the emission boundary's payload-validation rule now carries a direct, attributable failure-path proof (`R-AEE-021`), and the provider-failure payload's redaction is a property of the payload rather than of the caller's formatting verb (`R-AIP-016`). This entry now reads **discharged by AI-41**, not *assigned to AI-41*.

The two pre-existing Go identifiers in the quoted sentence are **retained verbatim** as historical record; the appended clause introduces none, per this project's behavior-only spec rule.

---

## Rules the amendment MUST satisfy

1. The existing sentences MUST be preserved byte-for-byte; only the dated discharge clause is appended.
2. The clause MUST name the date, the milestone, the change name, and both promoted requirement identifiers (`R-AEE-021`, `R-AIP-016`), so a reader can reach the behavior from the ledger in one hop.
3. The clause MUST NOT be written before the code and the two promoted requirements have landed. A ledger reading *discharged* over an unproven carryover is worse than one reading *assigned*.
4. On a revert of either leaf, the clause MUST be removed or re-qualified in the same edit. Leaving it standing after a revert is the single rollback error that reproduces AI-41's own failure mode (proposal § 10).
5. No requirement of this capability is added, modified, removed or renamed; `R-STK-008`'s serial-only rule in particular is untouched.

---

## Acceptance criteria

1. **At archive**: `openspec/specs/ai-stream-testkit/spec.md:39` matches the target text above, including both requirement identifiers.
2. The amendment is append-only — the pre-existing sentences are unchanged in the diff.
3. `R-STK-001` … `R-STK-013` are unchanged; this delta adds no requirement and no scenario, and consumes no `S-STK-*` identifier.
4. The corresponding discharge note lands on `ai-event-envelope`'s "Carried forward" section in the same archive step (see the `ai-event-envelope` delta), so the two records agree.
