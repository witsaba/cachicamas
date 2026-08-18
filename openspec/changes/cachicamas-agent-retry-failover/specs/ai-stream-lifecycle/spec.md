# Delta for `ai-stream-lifecycle` — AG-15 names the Layer 2 consumer of the composed-bound contract

> **Change**: `cachicamas-agent-retry-failover` · **AG-15** (Layer 2, Wave 3), `0003:1444-1525`
> **Modifies**: `ai-stream-lifecycle` ([`../../../../specs/ai-stream-lifecycle/spec.md`](../../../../specs/ai-stream-lifecycle/spec.md)) — `R-AIS-044` (`spec.md:634-650`).
> **Back-annotation only.** The requirement's normative sentences and **both** of its scenarios are reproduced **verbatim** and are not amended: no obligation is added, removed, weakened or re-worded. The only addition is a dated annotation naming the Layer 2 consumer that `S-2` has been waiting for since 2026-08-07. **No Layer 1 file is touched by this change** — `backend/agent/src/ai/internal/retry`'s package documentation is **read** by AG-15's cross-layer test at a repository-relative path and stays byte-unchanged (`R-RUN-012`), and reading a file is not importing a package.
> **Format**: the archive step REPLACES this block in the main spec; **full-block preservation is mandatory**.
> **Ownership**: the Layer 2 half of the contract is owned by [`../agent-retry-failover/spec.md`](../agent-retry-failover/spec.md) (`R-RTY-009` / `S-RTY-009`, with bite `S-RTY-012`). This delta claims nothing on Layer 1's behalf.

## MODIFIED Requirements

### R-AIS-044 (added 2026-08-07) — Composed-bound ceiling (cross-layer contract)

The composed bound **"harness attempts × Layer 1 attempts"** is documented where both layers' readers find it. The Layer 1 multiplier `defaultMaxAttempts = 3` is documented in the helper's package documentation (the file the helper ships in) and referenced verbatim by Layer 2's composed-bound test (per doc 0003 line 718). A reader from either Layer 1 (the helper's documentation) or Layer 2 (the harness test) finds the same number with the same formula.

This requirement carries **no production-code obligation** beyond the documentation's wording. Its purpose is the cross-layer visibility — the contract is binding *as documentation*, not as a runtime check.

**Back-annotation (AG-15, 2026-08-18) — the Layer 2 consumer now exists and is named.** `S-2`'s "Layer 2's harness-attempt test" was a forward reference when this requirement was promoted. It is discharged by `agent-retry-failover`'s `R-RTY-009` / `S-RTY-009`, whose test reads the helper's package documentation as a file, asserts the cited Layer 1 sentences verbatim, and asserts that Layer 2's own retry-policy documentation restates the formula in the same wording. Its third clause — divergence observable as a test failure — is discharged by the bite `S-RTY-012`, which perturbs the cited Layer 1 wording in a scratch tree and records the resulting failure before reverting. The Layer 2 side of the formula evaluates to **3 total harness attempts × 4 wire requests = 12**, where "3" counts total `Turn` invocations rather than retries after the first (`R-RTY-005`); the convention is stated on the Layer 2 side wherever the number appears, precisely because the two layers use different counting conventions. **Nothing in this requirement's obligations changes**, and no Layer 1 file is edited to satisfy it.

#### Scenario: R-AIS-044 / S-1 — Layer 1 multiplier documented in helper's package doc comment *(pin: `R-CNF-019`, `AG-15.2` item 2)*

- **GIVEN** the helper's package documentation file
- **WHEN** a Layer 1 reader opens the file
- **THEN** the documentation names the wire-request count per logical call (i.e. `N+1 = 4` wire requests when retries are exhausted), AND the composed-bound formula "harness attempts × Layer 1 attempts" appears in the same documentation, AND the documentation identifies Layer 2's composed-bound test as the cross-layer consumer

#### Scenario: R-AIS-044 / S-2 — Layer 2 reader sees the same number with the same formula *(pin: `R-CNF-019`, `AG-15.2` item 2; **satisfied by AG-15**, `R-RTY-009` / `S-RTY-009` + bite `S-RTY-012`)*

- **GIVEN** Layer 2's harness-attempt test
- **WHEN** a Layer 2 reader reads the test
- **THEN** the test cites the Layer 1 multiplier verbatim from the helper's package documentation, AND the composed-bound formula matches the helper's wording, AND a divergence between the two layers' wording is observable as a test failure
