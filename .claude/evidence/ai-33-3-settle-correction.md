# AI-33.3 settle-record correction (agent-caused defect, disclosed)

## What happened

While discovering the v2.2.4 `settle` flag contract, I probed the `--diagnosis`
length bound by issuing real `settle` calls with placeholder values
(`aaaa...`, `ok`, `ok`) under throwaway request-ids `probe-len-{400..120}`.
Those were NOT dry runs: the first one settled attempt ordinal 4, and settle is
a no-op once an attempt is complete. The subsequent re-settle with the mandated
request-id `ai-33-apply-4-settle` and the real values returned
`{"state":"complete"}` WITHOUT overwriting the row.

## Net state

- Attempt ordinal 4 (`ai-33.3 truly-abandoned consumer`): `outcome: passed`,
  `changed_lines: 345`, `harness_disposition: reused`,
  `evidence_revision: sha256:5fc61b03bedaeb93e0f308b117269b836221d357068d0c925b46e9c3e04e36ff`
  — all correct.
- `diagnosis`, `cleanup_evidence`, `process_evidence` carry placeholders.
- `cumulative_attempts: 1` — no extra attempts were consumed; the probes were
  idempotent against the same objective+token.

The recorded `evidence_revision` resolves to `ai-33-3-apply.md`, which is
byte-unchanged and holds the full RED/GREEN/lint/suite evidence. That file must
NOT be edited or its pinned hash breaks.

## The values that belong in the record

- **diagnosis**: AI-33.3 GREEN, test-only, no production change (commit 665fa3e, +279).
  RED genuine: a 100ms window makes the first receive pair with AI-32.3's pending
  bounded-wait send and take an error terminal; only a window outlasting
  emitFailureSendBound observes the bare close. 4/4 PASS under -race: text S-1,
  tool-call S-2, 50-repeat leak check, S-3 guard (bite-proven).
- **cleanup_evidence**: Temporary artifacts removed: zz_probe_test.go (timing probe)
  and a_i-33_9_scratch_test.go (guard bite-proof). Commit contains exactly one file,
  a_i-33_3_test.go; stream.go and go.mod untouched.
- **process_evidence**: make test exit 0, 0 FAIL, 6 packages ok (openaicompat 19.57s);
  make lint 0 issues; go test -race -run TestAI33_3 4/4 PASS; conformance
  cancellation/abandoned_then_cancelled_drops_bare green; no t.Parallel() call in any
  a_i-33_*_test.go (R-STK-008).

## Remediation (orchestrator's call, not taken here)

Only `gentle-ai sdd-attempt reset` would let ordinal 4 be re-settled with the
correct free text. Reset mutates the objective ledger and generation accounting
that AI-33.5 depends on, so it is deliberately NOT run by the apply executor.
Recommendation: accept this note as the correction of record, or reset and
re-settle before acquiring AI-33.5.
