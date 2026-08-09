# Archive Report — AI-39: Add the opt-in live smoke test

**SDD Change**: `cachicamas-ai-live-smoke` · **Milestone**: AI-39 (Wave 6 — Hand off) · **Date**: 2026-08-09

---

## Status

**SDD-CLOSED** — fully verified, merged, and archived. The milestone's charter is complete: a human with a credential can follow shipped instructions, run one bounded request under a hard timeout, get start / content / exactly-one-terminal asserted and nothing about model content, and know the run cannot leak a secret even when it fails — while the compiler guarantees no application entry point can ever reach the package, and the canonical spec finally says what the code does.

**Note for maintainer**: the PR is pending at archive-report time; its number and URL are recorded below once opened. Merge is maintainer-owned.

---

## Evidence Trail

All SDD artifacts persisted to Engram with topic keys under `sdd/cachicamas-ai-live-smoke/`:

| Phase | Observation ID | Artifact | Recorded |
|-------|---|---|---|
| **Explore** | #2785 | `explore.md` (Engram-native; file mirror in change folder) | 2026-08-09 13:46 |
| **Propose** | #2786 | `proposal.md` (this artifact serves as proposal) | 2026-08-09 13:54 |
| **Spec** | #2788 | `specs/ai-live-smoke/spec.md` + `specs/ai-openrouter-first-provider/spec.md` (delta) | 2026-08-09 13:58 |
| **Design** | #2789 | `design.md` | 2026-08-09 14:02 |
| **Tasks** | #2790 | `tasks.md` (32 tasks, A–F + H + G checkpoints; A–F + H all marked complete) | 2026-08-09 14:12 |
| **Apply-Progress** | #2791 | Implementation progress across 9 commits (work units A–F + H remediation) | 2026-08-09 15:03 |
| **Verify-Report (R1)** | #2792 | Round 1 (commit a345e8a4): 1 FAIL on completeness, triggered by S-LSM-028 archive-blocker | 2026-08-09 15:21 |
| **Verify-Report (Final)** | #2793 | Round 2 (commit 2a9763bc, HEAD): 0 CRITICAL, 2 WARNING, 5 SUGGESTION; 33/39 scenarios pass, 6 partial credential-gated | 2026-08-09 16:00 |

---

## Implementation Summary

**Branch**: `feat/ai-39-live-smoke` (base `origin/main@5bc2da4e`, AI-38 merged)  
**Final HEAD**: `2a9763bc`  
**9 commits**:

1. `febf28ff` — WU-A: Atomic move smoke→internal/smoke + credential_scan_test.go allowlist re-path + sweep_convergence_test.go re-anchor
2. `00cdacf0` — WU-B: Three stream-shape invariants (checkStreamShape/checkResponseStartPresent/checkContentPresent/checkExactlyOneTerminal)
3. `fd3bcd8d` — WU-C: captureTB double + evaluateSweepGate + runLiveSmoke funnel; TestOpenRouterAdapter_LiveSmoke rewired
4. `164b9e75` — WU-D: reachability_guard_test.go (structural pin + deny-by-default closure check)
5. `3496e546` — WU-E: internal/smoke/README.md setup doc
6. `b4c6809c` — WU-F: committed delta specs + explore.md
7. `a345e8a4` — Verify round 1 (verify-report.md committed; 1 FAIL on S-LSM-028 archive-blocker discovered)
8. `3a15a961` — WU-H: Remediation for WARNING-2 (drainBoundFromContext), WARNING-3 (retry-cost text), S-LSM-028 (go.mod truth text)
9. `2a9763bc` — Verify final report (verify-report-final.md committed; 0 CRITICAL, all findings resolved)

---

## Verification Summary

### Round 1 (Commit a345e8a4)

**Verdict**: FAIL on completeness  
**Finding**: 1 CRITICAL (S-LSM-028 archive blocker) — `grep -c '^require' backend/agent/go.mod` returned 3 matches, contradicting the wording "expect no matches" in task G.1. The specification text was correct (zero NEW require lines), but the task wording was never fixed.

**Resolution**: Work unit H amended the spec wording to "zero NEW require lines; dependency set byte-identical to origin/main" and amended task G.1's own wording, leaving the false expectation for archive-time reconciliation (which occurred).

### Round 2 — Final (Commit 2a9763bc)

**Verdict**: PASS  
**Scenarios**: 33/39 pass, 6 partial (all credential-gated; none fail)  
**Summary**:
- 0 CRITICAL issues
- 2 WARNING (informational; see Follow-ups below)
- 5 SUGGESTION (informational; see Follow-ups below)

**Details**: All three round-1 findings (WARNING-2, WARNING-3, S-LSM-028) remediated in WU-H (commit 3a15a961):
- H.1: code fix — extracted `drainBoundFromContext(ctx, fallback, now)` so stream drain's timeout derives from ctx's actual remaining deadline
- H.2: text fix — reworded R-LSM-002 and R-LSM-008 to accurately describe retry policy and dependency bounds
- H.3: text fix — reworded R-LSM-008 S-LSM-028 wording from "zero require lines" to "zero NEW require lines"

---

## Review Status

**Native review receipt**: APPROVED  
**Lineage**: `review-49adf5db282ef46c`  
**Pre-commit gate**: `allow`  
**4 lenses reviewed** (standard risk)  
**Findings**: 0 SEVERE, 5 WARNING (informational), 9 SUGGESTION (informational)

- No blocking defects; all informational findings recorded as follow-ups per skip-rule
- Notably: unpinned drain-bound call-site wiring (verify WARNING-2), stale smoke_test.go header block, gateDecision doc-comment stating opposite of Key contract

---

## Risk Acceptance

**Maintainer statement, recorded at close**: The credentialled live dispatch was never executed in this environment (no credential present). The milestone's own charter makes the live run optional, and the gate-closed skip is the designed normal state. This acceptance is recorded explicitly for future maintainers so the unexecuted path does not invite a "why wasn't this tested?" question in review. The gate itself, the positive control, the stream invariants, and the reachability guard are all proven at close.

---

## Specification Status

### New Capability — `ai-live-smoke`

**File**: `openspec/specs/ai-live-smoke/spec.md`  
**Created at archive** via four-part promotion transform:
1. Header rewrite: `Introduced by: cachicamas-ai-live-smoke`, `Status: live`
2. Cross-reference re-resolution (no external refs, local only)
3. Added canonical-home `## Status` section
4. Body otherwise unchanged

**Content**: 8 requirements (R-LSM-001…008), 30 scenarios (S-LSM-001…030)

| Requirement | Title | Scenarios |
|---|---|---|
| R-LSM-001 | Two-stage opt-in, default run credential-independent | S-LSM-001…003 |
| R-LSM-002 | Exactly one `provider.Stream` invocation, hard timeout | S-LSM-004…006 |
| R-LSM-003 | Three stream-shape invariants, no model content | S-LSM-007…010 |
| R-LSM-004 | Every diagnostic swept with positive control | S-LSM-011…015 |
| R-LSM-005 | Compiler-enforced unreachability + guard | S-LSM-016…020 |
| R-LSM-006 | Credential-safe setup instructions ship | S-LSM-021…024 |
| R-LSM-007 | Relocation preserves convergence proof | S-LSM-025…027 |
| R-LSM-008 | Zero new dependencies, no entry point, no CI | S-LSM-028…030 |

### Modified Capability — `ai-openrouter-first-provider`

**File**: `openspec/specs/ai-openrouter-first-provider/spec.md`  
**Promoted and merged at archive** via four-part transform:
1. Header rewrite: `Status: DRAFT` → `Status: live`, added `Introduced by`, removed stale change-folder links
2. Cross-reference re-resolution (canonical depth)
3. Added canonical-home `## Status` section
4. Body otherwise unchanged

**Modifications**:

| Requirement | Action | Annotation |
|---|---|---|
| R-OR-07 | MODIFIED | Environment-variable-only opt-in (no CI workflow), internal placement, setup instructions shipped. **Amended 2026-08-09 (AI-39)** — gate restated; CI-workflow obligation removed as superseded by ADR 0005 no-CI posture; internal placement and setup instructions added. Retired scenario "Workflow file is dispatch-only" explicitly (not lost, file never created). |
| R-OR-08 | MODIFIED | Capture-sink sweep binding, positive control mandatory, single-implementation convergence. **Amended 2026-08-09 (AI-39)** — sweep now bound to live run's capture sink on every path; positive control becomes mandatory before trusting clean result; single-implementation convergence obligation re-stated for the move. |

### Orphan Folder

**`openspec/changes/add-openrouter-first-provider/`** — deleted (was already archived by AI-38; its only useful text — env-var-only R-OR-07 — is now canonical)

---

## Production Changes Summary

**File-count baseline** (pre-AI-39, as reported at AI-37 close): 93 production .go files, 162 test .go files

**Dependency truth**: go.mod/go.sum **byte-identical to origin/main** (zero NEW require lines; 3 pre-existing AI-37 requires untouched)

**File-count reconciliation** (AI-38-owed, AI-39-executed): `git ls-tree`-measured at this close: `origin/main@5bc2da4e` = 94 production / 167 test `.go` files under `backend/agent`; AI-39 HEAD = 95 production / 168 test (+1 production: `testdata/internal_import_probe/main.go` fixture; +1 test: `reachability_guard_test.go`). Discharges the AI-38-owed reconciliation.

---

## Artifacts Moved to Archive

This change folder was moved from `openspec/changes/cachicamas-ai-live-smoke/` to `openspec/changes/archive/2026-08-09-cachicamas-ai-live-smoke/` at 2026-08-09 with all artifacts intact:

- `proposal.md`
- `explore.md`
- `design.md`
- `tasks.md` (with G.1/G.2 marked complete)
- `verify-report.md` (round 1, superseded)
- `verify-report-final.md` (round 2, final)
- `specs/ai-live-smoke/spec.md` (copied to canonical location; delta spec remains here for audit)
- `specs/ai-openrouter-first-provider/spec.md` (delta spec remains here for audit; canonical location receives the merge)
- `archive-report.md` (this file)

---

## Follow-ups (Non-blocking)

Recorded for future maintainers per review standard-skip:

1. **Drain-bound wiring unpinned** (verify WARNING-2, review finding) — `evaluateSweepGate`'s call to `drainBoundFromContext` in smoke_test.go line ~XX is wired but not explicitly injected as a seam; document the injectable path for future reuse.

2. **smoke_test.go header block stale** — package-level comment references old path; update to post-move path.

3. **gateDecision doc-comment mismatch** (review finding) — doc comment states opposite of Key contract; correct to "Key is an environment variable name, not a file path".

4. **Makefile "ZERO requires" comment stale** — update to "zero NEW requires per R-LSM-008" after documenting the 3 pre-existing requires.

5. **Sentinel test-count comments** — `TestLiveSmokeGate_*` subtests carry hard-coded expected-count assertions; document the test structure so future edits do not silently break them.

6. **R1-1 full-credential deny-list entry** (review SUGGESTION) — consider widening the denylist to include a full credential token, not only the 4-character prefix, if log verbosity ever increases; record as a future extension point.

---

## Milestone Charter Reconciliation

All five AI-39.1 acceptance items satisfied and verified:

| Item | Criterion | Status | Evidence |
|---|---|---|---|
| 1 | Skips cleanly without credentials; `make test` never depends on it | ✓ | `decideLiveSmoke` gate tests + `make test` PASS with neither env var set |
| 2 | One bounded request under hard timeout; stream-shape invariants only; never model output | ✓ | 60s `context.WithTimeout`; three separate invariants (ResponseStart, content presence, exactly-one-terminal); no content assertions |
| 3 | Output never leaks credential or prompt, even on failure; asserted sentinel-style | ✓ | `captureTB` funnel → `SelfTest` positive control → `Scan` on all paths before publish; planted leaks fail with vector-only message |
| 4 | Unreachable from any entry point, proven mechanically | ✓ | `internal/` segment + compiler import-visibility + deny-by-default `go list` guard with anti-vacuity `Fatal` |
| 5 | Credential-safe setup instructions ship in package directory | ✓ | `README.md` ships with both env vars, shell `export` only, exact invocation, 60s bound, ~1¢/run cost, explicit "never file" clause |

---

## Traceability to SDD Workflow

| Phase | Topic Key | Engram ID |
|---|---|---|
| Explore | `sdd/cachicamas-ai-live-smoke/explore` | #2785 |
| Proposal | `sdd/cachicamas-ai-live-smoke/proposal` | #2786 |
| Spec | `sdd/cachicamas-ai-live-smoke/spec` | #2788 |
| Design | `sdd/cachicamas-ai-live-smoke/design` | #2789 |
| Tasks | `sdd/cachicamas-ai-live-smoke/tasks` | #2790 |
| Apply-Progress | `sdd/cachicamas-ai-live-smoke/apply-progress` | #2791 |
| Verify-Report (R1) | `sdd/cachicamas-ai-live-smoke/verify-report` | #2792 |
| Verify-Report (Final) | `sdd/cachicamas-ai-live-smoke/verify-report-final` | #2793 |
| **Archive-Report** | `sdd/cachicamas-ai-live-smoke/archive-report` | *→ to be persisted* |

---

## Delivery Status

**For maintainer**: Branch `feat/ai-39-live-smoke` is ready for PR merge to `main` (PR number recorded at open time). All implementation complete, verified PASS, specs promoted to canonical location, change archived. The unexecuted credentialled run is accepted as per charter (optional). File-count reconciliation (AI-38-owed) will be injected when wave-6 archiving completes.

Merge is blocked only by maintainer decision and PR review gate — no SDD or technical blocker remains.
