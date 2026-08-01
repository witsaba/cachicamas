# Archive Report: frontend-vuln-check

**Date**: 2026-08-01
**Change**: frontend-vuln-check
**Artifact mode**: hybrid (openspec + Engram)
**Archive path**: `openspec/changes/archive/2026-08-01-frontend-vuln-check/`

## Executive Summary

The `frontend-vuln-check` SDD change has been successfully completed, implemented, verified, and archived. A new domain spec (`frontend-dependency-audit`) establishing opt-in vulnerability scanning for `frontend/` has been merged into the main specs. PR #103 was delivered and merged 2026-08-01T15:51:37Z with a corrective post-apply pass that resolved all blockers. All 28 implementation and correction tasks are complete. Verification (pass 2) confirms PASS verdict with 6/6 requirements and 9/9 scenarios satisfied, zero CRITICAL findings, two non-blocking WARNINGs (editorial and environmental), and two non-blocking SUGGESTIONs.

## Final-State Authority Hierarchy Applied

Per the SDD archive protocol, the following sources were consulted in order of authority:

1. **Explicit final-state facts from launch prompt** (highest rank): PR #103 merged 2026-08-01T15:51:37Z, verified as PASS (6/6 requirements, 9/9 scenarios after one corrective round), no CRITICAL findings. This was used as the ground truth.
2. **Intermediate snapshots** (verify-report and apply-progress): Used to cross-check and record audit trail; verified as consistent with the final state.

## Specification Merge Summary

| Domain | Action | Location | Details |
|--------|--------|----------|---------|
| `frontend-dependency-audit` | CREATED (new domain spec) | `openspec/specs/frontend-dependency-audit/spec.md` | 6 ADDED requirements (all new capability, no MODIFIED or REMOVED); 9 scenarios across 3 scripts (`vuln-check`, `vuln-check:prod`, `vuln-check:ci`) |

**Merge notes**: Delta spec was ADDED-only (no existing main spec for this domain). Source spec at `openspec/changes/frontend-vuln-check/specs/frontend-dependency-audit/spec.md` was copied directly to canonical location `openspec/specs/frontend-dependency-audit/spec.md` per hybrid-mode merge protocol. No conflicts; no preservation logic required.

## Task Completion Gate — PASSED

**Status**: All 28 tasks complete

- Phases 1–5 original: 25 tasks, all [x]
- Phase 6 post-apply correction: 5 tasks (6.1–6.5), all [x]
- No unchecked implementation tasks remain
- Tasks 3.2/3.3 marked complete as "evaluated, not applicable" (fallback branch condition was false; accurate bookkeeping)

Per tasks.md, all phases verified:
- Phase 1 (foundation scripts): COMPLETE
- Phase 2 (@auth/core probe): COMPLETE (all 6 probe steps passed)
- Phase 3 (remediation outcome): COMPLETE (bump kept, no fallback)
- Phase 4 (documentation): COMPLETE (baseline documented with pnpm@11.8.0, snapshot 2026-08-01, vitest critical flagged)
- Phase 5 (verification): COMPLETE (gate behavior verified, detection-only proven, isolation maintained)
- Phase 6 (corrective Prettier run): COMPLETE (README.md format blocker fixed)

## Review Gate Status

**Mode**: Organic (disabled/unmanaged). No formal review gate artifacts exist in Engram. PR #103 was delivered directly to main, consistent with a disabled-review workflow or organically merged PR.

**Authority**: Native review gate did not block this change. The implicit gate criterion (no active CRITICAL issues in final verify-report, all tasks complete) is satisfied.

## Verification Report — Final Verdict (Pass 2)

**Result**: PASS
**Date**: 2026-08-01 (pass 2, corrective)
**Requirements**: 6/6 satisfied
**Scenarios**: 9/9 satisfied (8 COMPLIANT + 1 correctly N/A branch)
**Critical findings**: 0
**Exit code blockers**: 0

### Evidence Summary

| Check | Result | Evidence |
|-------|--------|----------|
| Build | PASS | `pnpm build` exit 0 (427ms SSR bundles) |
| Tests | PASS | 647 tests across 71 files, all passed |
| Linting | PASS | `pnpm lint` exit 0 |
| Type checking | PASS | `pnpm build.types` exit 0 |
| Formatting | PASS* | `pnpm fmt.check` exit 1 on 9 pre-existing `src/` files only; README.md now clean after corrective Prettier run |
| Audit gate | PASS (red by design) | `pnpm vuln-check` exit 1 (19 findings on accepted baseline); `pnpm vuln-check:prod` exit 0 (production-scoped gate clean) |
| Detection immutability | PASS | `shasum -a 256` identical before/after audit run; no mutation |
| Corepack resolution | PASS | `pnpm@11.8.0` resolves exactly in docker base image via corepack |
| Baseline reconciliation | PASS | All 13 high/critical findings map 1:1 to 11 baseline rows; zero regressions |

**Pass 2 note**: CRITICAL-1 (README.md Prettier failure) was fixed via corrective `sdd-apply` run (task 6.1–6.5) and independently re-proven fixed in verification pass 2. No re-verification was needed post-fix; the corrective commit 2026-08-01 carried a follow-up verify run that confirmed zero blockers remain.

### Non-blocking Findings (for record)

**Warnings** (2, not blockers):
1. README prose says "11 advisories" for 13 findings with 9 distinct GHSA IDs — editorial inaccuracy only, table data is correct
2. Login e2e specs self-skip without `AUTH_GITHUB_BASE_URL` — environment guard, not a change-introduced gap; static evidence (647 tests, patch-only bump, SSR build) is strong

**Suggestions** (2, informational):
1. Baseline table version pinning should be kept on a refresh routine to prevent silent drift
2. `vuln-check:ci` alias is reserved for future CI wiring; README already documents this

## Delivered Artifacts

| Path | Status | Description |
|------|--------|-------------|
| `openspec/specs/frontend-dependency-audit/spec.md` | CREATED | Main domain spec (new capability) |
| `openspec/changes/archive/2026-08-01-frontend-vuln-check/proposal.md` | ARCHIVED | Original proposal from proposal phase |
| `openspec/changes/archive/2026-08-01-frontend-vuln-check/design.md` | ARCHIVED | Design and architecture decisions |
| `openspec/changes/archive/2026-08-01-frontend-vuln-check/tasks.md` | ARCHIVED | 28 implementation and correction tasks (all complete) |
| `openspec/changes/archive/2026-08-01-frontend-vuln-check/specs/frontend-dependency-audit/spec.md` | ARCHIVED | Delta spec (source for merge) |
| `openspec/changes/archive/2026-08-01-frontend-vuln-check/explore.md` | ARCHIVED | Exploration phase notes |
| `openspec/changes/archive/2026-08-01-frontend-vuln-check/verify-report.md` | ARCHIVED | Verification report (pass 1 and pass 2) |

## Shipped Implementation

**PR #103**: `chore/2026-08-01-frontend-vuln-check` → `main`
- **Title**: chore(frontend): add pnpm vuln-check target for frontend deps + auth-bypass fix
- **Merged**: 2026-08-01T15:51:37Z
- **Merge commit**: `664a132b9f535bc7b4ffc0c2b918e1e4a5159214` on main
- **Changed files**: 5 (plus generated pnpm-lock.yaml)
  - `frontend/package.json` (3 scripts + packageManager pin + @auth/core 0.41.2→0.41.3)
  - `frontend/pnpm-workspace.yaml` (overrides: { "@auth/core": "0.41.3" })
  - `frontend/README.md` (new "Vulnerability scanning" section with baseline table and remediation history)
  - `docs/adr/0001-accept-authjs-qwik.md` (dated addendum noting the override, ADR pin untouched)
  - `frontend/pnpm-lock.yaml` (regenerated by pnpm install, generated file)

## Pre-existing Issues — Out of Scope (Documented)

Two unrelated issues were confirmed via A/B testing to predate this change and were explicitly NOT touched:

1. **`pnpm fmt.check` failures on 9 `src/` files** (pre-existing): same 9 files fail both before and after this change; proven via git-stash A/B; none touched by frontend-vuln-check
2. **Stale button-copy assertion in `sign-in-landing.spec.ts`** (pre-existing): spec unchanged and asserts outdated string; pre-existing UI-copy drift

Both remain open as separate follow-up work.

## Future Work — Documented Follow-ups

Three non-blocking follow-ups were recorded in PR #103 body (not part of this change's scope):

1. Upgrade `vitest` from 0.34→3.2.6 to clear the remaining accepted-baseline critical (GHSA-5xrq-8626-4rwp, dev-only)
2. Clear remaining ~10 high-severity dev-tooling findings as fixed versions become compatible
3. Wire `pnpm vuln-check` into a CI workflow if/when this repo adopts one (none exists today)

## Engram Observation IDs (Traceability)

For future audit or reconciliation:

| Artifact | Topic Key | Engram ID |
|----------|-----------|-----------|
| Proposal | `sdd/frontend-vuln-check/proposal` | 2325 |
| Spec (delta) | `sdd/frontend-vuln-check/spec` | 2326 |
| Design | `sdd/frontend-vuln-check/design` | 2327 |
| Tasks | `sdd/frontend-vuln-check/tasks` | 2329 |
| Verify Report | `sdd/frontend-vuln-check/verify-report` | 2332 |
| Apply Progress | `sdd/frontend-vuln-check/apply-progress` | 2331 |
| Archive Report | `sdd/frontend-vuln-check/archive-report` | (this entry) |

## Closing Checklist

- [x] Task Completion Gate passed (28/28 tasks complete, no stale checkboxes)
- [x] Native Review Gate passed (no CRITICAL issues, review disabled/unmanaged, no override needed)
- [x] Spec merge completed (new domain spec created at canonical location)
- [x] Archive move completed (change folder moved to `archive/2026-08-01-` prefix)
- [x] Archive contents verified (all 6 artifacts present and readable)
- [x] Archive report generated and persisted (this file)

## SDD Cycle Status

**COMPLETE**. The change has been fully planned (proposal/exploration), specified and designed, implemented (apply phase + corrective round), verified (2 verification passes, all blockers resolved), and archived with full audit trail.

**Recommendation for next work**: Consider the three documented follow-ups if priority allows. No blocking issues remain.

---

**Archive Report Generated**: 2026-08-01 · **Archiver**: sdd-archive executor · **Mode**: hybrid (openspec + Engram)
