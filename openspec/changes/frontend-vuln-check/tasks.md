# Tasks: Opt-in dependency vulnerability check for `frontend/`

## Review Workload Forecast

| Field | Value |
|-------|-------|
| Estimated changed lines | ~70-100 authored (package.json, pnpm-workspace.yaml, README, ADR); `pnpm-lock.yaml` regeneration excluded as generated |
| 400-line budget risk | Low |
| Chained PRs recommended | No |
| Suggested split | Single PR |
| Delivery strategy | single-pr |
| Chain strategy | pending |

Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: pending
400-line budget risk: Low

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
|------|------|-----------|-----------------------|------------------|-------------------|
| 1 | Audit scripts + `@auth/core` remediation (or documented fallback) + docs | PR 1 | `pnpm vuln-check`, `pnpm vuln-check:prod` | `pnpm test:e2e` sign-in specs | Revert one PR; scripts/override/docs additive, unreferenced by `verify`/`build`/Docker |

## TDD note

No test code is added. This change is package.json scripts (static pnpm CLI invocations) plus a dependency bump; Strict TDD governs code changes with test coverage, not config/dependency-only changes. `frontend/e2e/` sign-in specs are the pre-existing regression net for the bump — do not author new RED tests; run the existing e2e suite as verification.

## Phase 1: Foundation — scripts & pin (independent of probe outcome)

- [x] 1.1 Add `"vuln-check": "pnpm audit --audit-level=high"` to `frontend/package.json` scripts (Req: Opt-in full-tree gating audit)
- [x] 1.2 Add `"vuln-check:prod": "pnpm audit --prod --audit-level=high"` (informational, non-gating)
- [x] 1.3 Add `"vuln-check:ci": "pnpm run vuln-check"` alias
- [x] 1.4 Add `"packageManager": "pnpm@11.8.0"` to `frontend/package.json` (Req: Package manager pinning). Also added `engines.pnpm: ">=11.8.0"` to satisfy the spec's companion SHOULD clause (advisory only — `.npmrc` has no `engine-strict`, so this cannot break installs).
- [x] 1.5 Confirm `verify`, `build`, `test:ci` scripts and Dockerfile do not reference the new scripts (Req: Isolation from default workflow) — confirmed by inspection: `verify` = `lint && build.types && fmt.check && test:ci`, `build` = `build.types && build.client && build.server`, `test:ci` = `vitest --run`, Dockerfile only runs `pnpm install --frozen-lockfile --shamefully-hoist` and `pnpm build`. None reference `vuln-check*`.

## Phase 2: `@auth/core` compatibility probe (BEFORE Phase 3/4)

- [x] 2.1 `pnpm view @auth/core@0.41.3 version` — confirm the patch exists on npm. PASS: printed `0.41.3`.
- [x] 2.2 Tentatively bump `@auth/core` to `0.41.3` in `frontend/package.json`; add `overrides: { "@auth/core": "0.41.3" }` to `frontend/pnpm-workspace.yaml`. Done.
- [x] 2.3 `pnpm install` — confirm no resolution errors. PASS: `Packages: +1 -1`, `@auth/core 0.41.2 → 0.41.3`, no errors.
- [x] 2.4 `pnpm why @auth/core` — confirm single `0.41.3` resolution, incl. under `@auth/qwik`. PASS: "Found 1 version of @auth/core", listed under both `@auth/qwik@0.9.2` and the direct dependency.
- [x] 2.5 `pnpm verify` and `pnpm test:e2e` (sign-in specs: github-sign-in, sign-in-landing, sign-in-denied, sign-in-cookie-attrs, sign-out). RESULT (see 2.6 for full reasoning): `pnpm lint` PASS, `pnpm build.types` PASS, `pnpm test:ci` PASS (71 files / 647 tests), `pnpm fmt.check` FAILED on 9 files — proven pre-existing via git-stash A/B against the unbumped tree (identical failure list, none of the 9 files touched by this change). e2e: `github-sign-in`/`sign-in-denied`/`sign-in-cookie-attrs`/`sign-out` self-skip (no `AUTH_GITHUB_BASE_URL` / mocks stack in this environment — by-design skip guard in each spec); `sign-in-landing` 1/2 pass, 1/2 fails on a pre-existing button-copy-text assertion ("Sign in with GitHub" vs actual "Sign in") proven identical on `@auth/core@0.41.2` via the same A/B method.
- [x] 2.6 Record pass/fail per step 2.1-2.5 to decide the Phase 3 branch (Req: `@auth/core` remediation). DECISION: **all bump-attributable checks pass** — install/why/lint/build.types/test:ci are clean, and both apparent failures (prettier fmt.check, one e2e button-copy assertion) are confirmed pre-existing/unrelated via side-by-side git-stash reruns against `@auth/core@0.41.2` on the base tree (byte-identical failures). Per the design's fallback criterion ("if unresolvable without breaking the ADR-0001 pin") this is a clean resolve, not a blocked bump — Docker daemon was started and Chromium/Playwright were already installed to make this probe possible; a throwaway local `AUTH_SECRET` (not persisted, not from the repo's real secrets) was exported only to get past the dev server's `MissingSecret` guard for the one unauthenticated-rendering e2e spec that can run without the full mocks stack.

## Phase 3: Finalize remediation outcome (branches on 2.6)

- [x] 3.1 IF all probe steps passed: keep the bump + override; commit regenerated `frontend/pnpm-lock.yaml` (Scenario: Bump resolves cleanly). DONE — bump + override kept; `frontend/pnpm-lock.yaml` reflects the `pnpm install` regeneration (not git-committed by this apply run; commit is the orchestrator's delivery step).
- [x] 3.2 IF any probe step failed: revert Phase 2's bump/override, restore `@auth/core@0.41.2`, `pnpm install` — evaluated per 2.6: condition is false (no probe step failed), so this branch does not execute. Marked complete as "evaluated, not applicable" so no downstream phase mistakes this for pending work.
- [x] 3.3 If fallback taken, note the exact failing step for the ADR-0001 follow-up — evaluated per 2.6: fallback was not taken, so there is no failing step to record. N/A.

## Phase 4: Documentation (content depends on Phase 3 outcome)

- [x] 4.1 Add `## Vulnerability scanning` to `frontend/README.md`: commands, detection-only + full-tree note, opt-in status (Req: Documented baseline)
- [x] 4.2 Add `### Current baseline`: `pnpm@11.8.0 audit`, snapshot date, finding counts from actual apply-time `pnpm audit` output, vitest critical (GHSA-5xrq-8626-4rwp, dev-only, accepted), `@auth/core` entry per Phase 3 outcome. Full-tree run on 2026-08-01 (post-bump) found 19 total (1 low/5 moderate/12 high/1 critical); the gate-relevant 13 high+critical findings across 11 advisories are all listed in the baseline table (dev-only: vitest, vite ×2 versions, brace-expansion ×3 versions, svgo, sharp, postcss). `vuln-check:prod` confirmed clean (exit 0).
- [x] 4.3 Add `### Remediation history` recording the `@auth/core` outcome
- [x] 4.4 Add a note to `docs/adr/0001-accept-authjs-qwik.md`: override detail if bumped, or unresolved-conflict + follow-up if fallback taken. Added as a dated Addendum section.

## Phase 5: Verification

- [x] 5.1 `pnpm vuln-check` — non-zero exit, only baseline-documented findings (Scenario: Gate detects a full-tree finding; Reviewer reconciles a red gate). PASS: exit 1, 19 findings (1 low/5 moderate/12 high/1 critical), all 11 high/critical advisories match the README baseline table 1:1.
- [x] 5.2 `pnpm vuln-check:prod` — exit 0 if bumped, else documents remaining finding (Scenario: Gate is clean). PASS: exit 0, "No known vulnerabilities found".
- [x] 5.3 `pnpm verify` and `pnpm build` — unchanged exit codes/behavior (Scenario: Default loop unaffected). `pnpm build` PASS (exit 0, client + server bundles built). `pnpm verify` exit code unchanged at 1 both before and after this change (pre-existing `fmt.check` debt on 9 unrelated files, proven via git-stash A/B in Phase 2 notes) — lint/build.types/test:ci sub-steps all pass under the new state. **CORRECTED 2026-08-01 (post-verify)**: this evidence undercounted — at the time this task ran, `frontend/README.md`'s new "Current baseline" table had NOT been through Prettier, making it a genuine 10th `fmt.check` failure (change-authored, not pre-existing). See "Post-apply correction" section below for the fix and re-confirmed evidence; the "9 unrelated files" claim is accurate ONLY for the `src/` pre-existing set, not the total failure count at that point in time.
- [x] 5.4 Diff `frontend/package.json`/`pnpm-lock.yaml` before/after a gate run — byte-for-byte unchanged (Req: Detection only). PASS: `shasum -a 256` identical for both files before and after running `pnpm vuln-check`.
- [x] 5.5 Confirm `frontend/Dockerfile` corepack resolves the pinned `pnpm@11.8.0` (Scenario: Pinned toolchain). PASS: `docker run node:22-alpine` (the Dockerfile's exact base image) with only the new `frontend/package.json` mounted — `corepack enable && pnpm --version` downloaded and activated exactly `11.8.0`.

## Post-apply correction (sdd-apply re-run, 2026-08-01)

`sdd-verify` (see verify-report CRITICAL-1) found that `frontend/README.md`'s new "Current baseline" table was authored without Prettier's column padding, so it failed `pnpm fmt.check` as a genuine regression introduced by this change (distinct from the 9 pre-existing, unrelated `src/` failures and the 1 pre-existing, unrelated e2e assertion mismatch — both re-confirmed out of scope here).

- [x] 6.1 Run `pnpm exec prettier --write README.md` (scoped to this file only, so the 9 unrelated `src/` files' formatting is untouched). DONE.
- [x] 6.2 Re-run `pnpm fmt.check` in `frontend/` — confirm `README.md` no longer appears in the failure list. PASS: output now lists exactly the same 9 pre-existing `src/` files as before this change (`prompt-editor.tsx`, `delete-confirm-dialog.tsx`, `empty-state.tsx`, `restore-confirm-dialog.tsx`, `skill-editor/classes.ts`, `skill-editor.tsx`, `skill-list-item.tsx`, `settings/index.tsx`, `settings/skills/index.tsx`); `README.md` is gone.
- [x] 6.3 Run `pnpm exec prettier --check README.md` alone — confirm exit 0. PASS: "All matched files use Prettier code style!"
- [x] 6.4 Diff the baseline table's content (whitespace-stripped) before vs. after the Prettier fix — confirm zero character-level change to any GHSA ID, version, or prose; only column-padding whitespace (and the separator row's dash-count, which carries no data) differs. PASS: whitespace-stripped diff of all 11 data rows is byte-identical; the only diff line is the markdown table separator row's dash-count re-flowing to match new column widths.
- [x] 6.5 Confirm no file other than `frontend/README.md` was touched by this correction. PASS: `git status --porcelain` before and after the fix shows the identical set of modified paths (`docs/adr/0001-accept-authjs-qwik.md`, `frontend/README.md`, `frontend/package.json`, `frontend/pnpm-lock.yaml`, `frontend/pnpm-workspace.yaml`) — no new path appeared.
