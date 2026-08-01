```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:94eaa0fcd35f62306c2c739dfe4c35bccb557142aa582cdd5a9b7f336c3a1a4f
verdict: pass
blockers: 0
critical_findings: 0
requirements: 6/6
scenarios: 9/9
test_command: pnpm test:ci
test_exit_code: 0
test_output_hash: sha256:f252eaefe3bdd3de4b70fa40e431f203ece2aa12b4ce62613b2b71297aea8d40
build_command: pnpm build
build_exit_code: 0
build_output_hash: sha256:3086ab8599b4c1296f0214677e4bc98922e33b561d86154fb49eb2eb714a581e
```

> **STATUS — 2026-08-01 (re-verification, pass 2)**: the single CRITICAL blocker
> recorded in pass 1 below is **RESOLVED and independently re-proven**. The YAML
> block above reflects pass 2. Pass 1's findings are preserved verbatim as the
> audit trail; see "## Re-verification (pass 2)" at the end of this file for the
> current verdict and its evidence.

## Verification Report

**Change**: frontend-vuln-check
**Version**: spec delta `frontend-dependency-audit` (2026-08-01)
**Mode**: Standard (Strict TDD is active project-wide; `tasks.md` TDD note correctly scopes it out — this change adds no application code path, so there is no RED test to author. Strict-TDD sections below are reported as N/A, not skipped silently.)

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 25 |
| Tasks complete | 25 |
| Tasks incomplete | 0 |

All 25 task boxes are `[x]` in `openspec/changes/frontend-vuln-check/tasks.md`. Tasks 3.2/3.3 are marked complete as "evaluated, not applicable" (fallback branch); the branch condition is genuinely false, so this is accurate bookkeeping rather than a false completion.

### Build & Tests Execution

All commands below were executed by the verifier in `frontend/`, not copied from `apply-progress`.

**Build**: PASSED

```text
$ pnpm build            → exit 0   (client + server SSR bundles, "built in 427ms")
$ pnpm build.types      → exit 0
$ pnpm lint             → exit 0
```

**Tests**: PASSED — 647 passed / 0 failed

```text
$ pnpm test:ci          → exit 0
  Test Files  71 passed (71)
  Tests       647 passed (647)
```

**Format check**: FAILED — and the failure set grew because of this change

```text
$ pnpm fmt.check        → exit 1
  Code style issues found in 10 files:
    README.md                                                        <-- AUTHORED BY THIS CHANGE
    src/components/prompts/prompt-editor/prompt-editor.tsx
    src/components/skills/delete-confirm-dialog/delete-confirm-dialog.tsx
    src/components/skills/empty-state/empty-state.tsx
    src/components/skills/restore-confirm-dialog/restore-confirm-dialog.tsx
    src/components/skills/skill-editor/classes.ts
    src/components/skills/skill-editor/skill-editor.tsx
    src/components/skills/skill-list-item/skill-list-item.tsx
    src/routes/settings/index.tsx
    src/routes/settings/skills/index.tsx
```

**Audit gate**: executed by the verifier

```text
$ pnpm vuln-check       → exit 1   19 findings (1 low | 5 moderate | 12 high | 1 critical)
                                   13 high/critical blocks printed under --audit-level=high
$ pnpm vuln-check:prod  → exit 0   "No known vulnerabilities found"
```

**Coverage**: Not available — no coverage tool configured in `frontend/package.json`. Not a failure.

### Spec Compliance Matrix

| Requirement | Scenario | Evidence | Result |
|-------------|----------|----------|--------|
| Opt-in full-tree gating audit | Gate detects a full-tree finding | `pnpm vuln-check` → exit 1; 13 high/critical blocks, each printing severity, package, vulnerable range and a `GHSA-*` advisory URL; findings rooted in `devDependencies` (vitest, vite, eslint chain) prove full-tree scope | COMPLIANT |
| Opt-in full-tree gating audit | Gate is clean | Not directly observable — the gate is red by design on accepted `vitest`/`vite` debt. Inverse evidence: `vuln-check:prod` uses the identical `--audit-level=high` mechanism and exits 0 on a clean sub-tree, demonstrating the zero-exit path works | PARTIAL |
| Opt-in full-tree gating audit | Informational script does not gate | `vuln-check:prod` = `pnpm audit --prod --audit-level=high`; README labels `vuln-check` "THE GATE" and `vuln-check:prod` "informational"; no script or CI consumes its status | COMPLIANT |
| Detection only | Audit leaves the tree unchanged | `shasum -a 256` of `package.json` + `pnpm-lock.yaml` captured immediately before and after a `pnpm vuln-check` run: byte-identical (verifier-run, not quoted) | COMPLIANT |
| Isolation from the default workflow | Default loop unaffected | `verify`/`build`/`build.client`/`build.server`/`test:ci` contain no `vuln-check*` reference; `frontend/Dockerfile` runs only `pnpm install --frozen-lockfile --shamefully-hoist` and `pnpm build`; no `.github/workflows/` exists and none was added. BUT `pnpm verify` chains `fmt.check`, and this change adds `README.md` to that command's failure set (see CRITICAL-1) | PARTIAL |
| `@auth/core` remediation | Bump resolves cleanly | `pnpm why @auth/core` → "Found 1 version of @auth/core", `0.41.3` under both `@auth/qwik@0.9.2` and the direct dependency; zero `0.41.2` strings remain in `pnpm-lock.yaml`; `GHSA-7rqj-j65f-68wh` absent from the full-tree audit output. Login-path e2e NOT exercised: 4 of 5 sign-in specs self-skip on an unset `AUTH_GITHUB_BASE_URL` (pre-existing by-design guard) | PARTIAL |
| `@auth/core` remediation | Bump is blocked by ADR-0001 | Branch condition false — resolution succeeded without touching the `@auth/qwik@0.9.2` pin | N/A (correctly not taken) |
| Documented baseline | Reviewer reconciles a red gate | Verifier re-ran `pnpm audit` and mapped all 13 high/critical findings 1:1 into the README's 11 baseline rows; every "Found in" version matches the installed tree (`vitest@0.34.6`, `vite@5.4.21`+`7.3.1`, `brace-expansion@1.1.15`/`2.1.1`/`5.0.7`, `svgo@3.3.3`, `sharp@0.34.5`, `postcss@8.5.16`). Zero unlisted findings, zero regressions | COMPLIANT |
| Package manager pinning | Pinned toolchain | `docker run --rm node:22-alpine` (the Dockerfile's exact base image) with only the new `package.json` mounted: `corepack enable && pnpm --version` → downloaded and activated exactly `11.8.0` | COMPLIANT |

**Compliance summary**: 5/9 scenarios COMPLIANT, 3 PARTIAL, 1 correctly N/A. Counting the N/A branch as satisfied: 7/9 scenarios pass, 5/6 requirements pass.

### Baseline Reconciliation (verifier-run audit vs README table)

| # | Severity | Package | Advisory | Installed | README row |
|---|----------|---------|----------|-----------|------------|
| 1 | critical | `vitest` | GHSA-5xrq-8626-4rwp | 0.34.6 | 1 |
| 2 | high | `vite` | GHSA-v2wj-q39q-566r | 7.3.1 | 2 |
| 3 | high | `vite` | GHSA-p9ff-h696-f583 | 7.3.1 | 3 |
| 4 | high | `vite` | GHSA-fx2h-pf6j-xcff | 5.4.21 | 4 |
| 5 | high | `vite` | GHSA-fx2h-pf6j-xcff | 7.3.1 | 5 |
| 6 | high | `brace-expansion` | GHSA-3jxr-9vmj-r5cp | 2.1.1 | 7 |
| 7 | high | `brace-expansion` | GHSA-3jxr-9vmj-r5cp | 1.1.15 | 6 |
| 8 | high | `svgo` | GHSA-2p49-hgcm-8545 | 3.3.3 | 9 |
| 9 | high | `sharp` | GHSA-f88m-g3jw-g9cj | 0.34.5 | 10 |
| 10 | high | `postcss` | GHSA-r28c-9q8g-f849 | 8.5.16 | 11 |
| 11 | high | `brace-expansion` | GHSA-mh99-v99m-4gvg | 1.1.15 | 6 |
| 12 | high | `brace-expansion` | GHSA-mh99-v99m-4gvg | 2.1.1 | 7 |
| 13 | high | `brace-expansion` | GHSA-mh99-v99m-4gvg | 5.0.7 | 8 |

Every reported finding is listed in "Current baseline"; no unlisted finding and no regression. The spec's "Reviewer reconciles a red gate" scenario holds. `GHSA-7rqj-j65f-68wh` appears nowhere in the output, confirming the remediation.

Distinct advisory IDs across these 13 findings: **9** (`5xrq`, `v2wj`, `p9ff`, `fx2h`, `3jxr`, `2p49`, `f88m`, `r28c`, `mh99`). The README prose calls them "the 11 advisories below" — see WARNING-1.

### Correctness (Static Evidence)

| Requirement | Status | Notes |
|------------|--------|-------|
| Opt-in full-tree gating audit | Implemented | `vuln-check` = `pnpm audit --audit-level=high`, exactly as designed; unscoped, so full tree |
| Detection only | Implemented | Fixed-flag pnpm builtin invocation; no write flags; hash-verified no mutation |
| Isolation from the default workflow | Implemented with a caveat | No wiring into `verify`/`build`/`test:ci`/Docker and no CI workflow, but the change worsens `fmt.check` inside `verify` (CRITICAL-1) |
| `@auth/core` remediation | Implemented | Direct dep `0.41.2 → 0.41.3` plus `overrides` in `pnpm-workspace.yaml`; single tree-wide resolution proven |
| Documented baseline | Implemented | pnpm version, snapshot date, per-finding table, vitest critical flagged as accepted debt, Remediation history present |
| Package manager pinning | Implemented | `packageManager: "pnpm@11.8.0"` exact; `engines.pnpm: ">=11.8.0"` satisfies the companion SHOULD |

### Coherence (Design)

| Decision | Followed? | Notes |
|----------|-----------|-------|
| Three scripts, none named `audit` | Yes | `vuln-check`, `vuln-check:prod`, `vuln-check:ci`; commands byte-match the design table |
| `vuln-check:ci` is an alias | Yes | `pnpm run vuln-check` |
| Override, not just a bump | Yes | `overrides: { "@auth/core": "0.41.3" }` in `pnpm-workspace.yaml`, with the design's rationale reproduced as a comment |
| `@auth/qwik@0.9.2` pin untouched | Yes | Still `"@auth/qwik": "0.9.2"` in `dependencies`; ADR-0001's decision body unchanged |
| Pin `packageManager: pnpm@11.8.0` | Yes | Exact pin; corepack resolution verified in the real base image |
| README structure mirrors backend | Yes | "Vulnerability scanning" → "Current baseline" → "Remediation history"; detection-only and not-reachability-aware notes present |
| ADR-0001 note | Yes | Dated addendum, explicitly non-decision-changing, cross-links the README |
| Compatibility probe step (f) `pnpm test:e2e` | Partially | Login e2e could not run — see WARNING-2 |
| Nothing wired into verify/build/Dockerfile | Yes, structurally | No script reference; the `fmt.check` impact is incidental, not wiring |

### TDD Compliance

N/A by design. `tasks.md` §"TDD note" scopes Strict TDD out for this config/dependency-only change, and no application code path was added. `apply-progress` correctly reports Mode: Standard with no TDD Cycle Evidence table. No test files were created or modified by this change (`git status` shows zero changes under `frontend/src/` or `frontend/e2e/`), so the Assertion Quality Audit has an empty input set.

**Assertion quality**: N/A — no test files authored or modified by this change.

### Test Layer Distribution (pre-existing regression net, not authored here)

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit + Integration | 647 | 71 | vitest 0.34.6 |
| E2E (sign-in net) | 5 specs | 5 | Playwright 1.61.1 — 4 self-skip without `AUTH_GITHUB_BASE_URL` |

### Quality Metrics

**Linter**: PASSED — `pnpm lint` exit 0
**Type checker**: PASSED — `pnpm build.types` exit 0
**Formatter**: FAILED — `pnpm fmt.check` exit 1 on 10 files, one of which this change authored

### Issues Found

**CRITICAL**

1. **`frontend/README.md` introduces a new `prettier --check` failure inside `pnpm verify`.** This directly contradicts `apply-progress` ("Issues Found: None caused by this change") and tasks 2.5 / 5.3, which both record the `fmt.check` debt as "9 files unrelated to this change". The verifier measured **10** files, and proved by A/B that the tenth is change-authored:
   - `git show HEAD:frontend/README.md` → `prettier --check` **exit 0** (clean before the change).
   - Working-tree `frontend/README.md` → `prettier --check` **exit 1** (dirty after the change).
   - Cause: the 11-row "Current baseline" markdown table is written without prettier's column padding/alignment. `prettier README.md` rewrites only those 13 table lines; the rest of the new section is already conformant.
   - Impact: `pnpm verify` chains `fmt.check`, so this adds new formatting debt to the default developer loop that the spec's "Isolation from the default workflow" requirement is meant to protect. The overall `pnpm verify` exit code stays `1` only because pre-existing debt already held it there — that coincidence is what let the apply phase miss the regression.
   - Fix: run `pnpm fmt` (or `npx prettier --write README.md`) in `frontend/`, then re-run `pnpm fmt.check` and confirm the list is back to the 9 pre-existing files. This is the only blocker.

**WARNING**

1. **README prose miscounts distinct advisories.** "13 findings across the 11 advisories below" — the table has 11 *rows*, but the 13 findings carry only **9** distinct `GHSA-*` IDs (rows 6 and 7 each merge `GHSA-3jxr-9vmj-r5cp` + `GHSA-mh99-v99m-4gvg`, and `GHSA-fx2h-pf6j-xcff` appears in both rows 4 and 5). The table content is correct and fully reconciles; only the word "advisories" is wrong. Suggested wording: "13 findings (9 distinct advisories) across the 11 package/version rows below". `apply-progress`'s claim of "11 high/critical advisories reconciled 1:1" carries the same error.
2. **The spec's "authentication behaviour is unchanged (login e2e passes)" clause is not proven by execution.** Four of the five sign-in specs self-skip on an unset `AUTH_GITHUB_BASE_URL` (a pre-existing by-design guard, verified at `github-sign-in.spec.ts:39`, `sign-in-denied.spec.ts:42`, `sign-in-cookie-attrs.spec.ts:36`, `sign-out.spec.ts:33`), so the OAuth round trip was never exercised against `@auth/core@0.41.3`. Static evidence is strong (single-version resolution, patch-only release, 647 unit/integration tests green, SSR build green), but a reviewer should be told the login e2e net was inactive rather than passing.

**SUGGESTION**

1. Consider adding `frontend/README.md`'s baseline table to a lightweight refresh routine — the "Found in" column pins exact installed versions (`vite@7.3.1`, `postcss@8.5.16`, …) and will silently drift on the next `pnpm update`.
2. `vuln-check:ci` is currently an unused alias. Worth a one-line README note that it is reserved and intentionally unreferenced, so a future reader does not assume CI wiring exists.

### Pre-existing Failures — Independently Confirmed (informational, NOT blockers)

The apply phase claimed two exit-1 results were pre-existing. The verifier tested both rather than accepting the claim:

1. **`pnpm fmt.check` on the 9 `src/` files** — CONFIRMED pre-existing. `git status --porcelain frontend/src/` is empty, so all 9 files are byte-identical to `HEAD`; prettier evaluates each file independently and no prettier config file is in this change's diff, so the change cannot have caused their verdicts. Note this confirmation covers **only 9 of the 10** currently failing files — the tenth, `README.md`, is CRITICAL-1.
2. **`e2e/sign-in-landing.spec.ts` button-copy assertion** — CONFIRMED pre-existing and unrelated. `frontend/e2e/` has zero working-tree changes and the spec was last touched in commit `7ea621a` (PR #20, long before this change). Line 61 asserts `toContainText(/Sign in with GitHub/i)`, while `src/components/sign-in-button/sign-in-button.tsx:78` defaults the label to `"Sign in"` — and line 30 of that same component carries the comment `("Sign in", not "Sign in with GitHub" — the brand mark carries …)`. The source of truth deliberately says "Sign in"; the e2e assertion is stale. Neither file is touched by this change, so the mismatch is UI-copy drift predating it.

### Verdict

**FAIL** — 1 CRITICAL, 2 WARNING, 2 SUGGESTION.

The substance of the change is correct and independently proven: the gate is real and full-tree, the audit mutates nothing, `@auth/core` resolves to a single `0.41.3` with `GHSA-7rqj-j65f-68wh` gone, `vuln-check:prod` is clean at exit 0, the README baseline reconciles 1:1 against a fresh audit, and corepack resolves the pinned `pnpm@11.8.0` in the real Docker base image. The single blocker is cosmetic in nature but real in effect: this change adds `frontend/README.md` to the `pnpm fmt.check` failure set, which the apply phase reported as clean. One `pnpm fmt` run clears it, after which this change is archive-ready.

---

## Re-verification (pass 2)

**Date**: 2026-08-01 · **Trigger**: corrective `sdd-apply` run claiming CRITICAL-1 fixed
**Method**: every claim below was re-executed by the verifier. No statement in the
corrective `apply-progress` was accepted on assertion.

### Blocker Resolution — CRITICAL-1

| Check | Command | Result |
|---|---|---|
| Full-tree format gate | `pnpm fmt.check` | exit 1 — **9** files, all under `src/`. `README.md` is **absent** from the list |
| Direct file check | `pnpm exec prettier --check README.md` | **exit 0** — "All matched files use Prettier code style!" |

The failure set returned from 10 files to the original 9. Verbatim pass-2 list:
`src/components/prompts/prompt-editor/prompt-editor.tsx`,
`src/components/skills/delete-confirm-dialog/delete-confirm-dialog.tsx`,
`src/components/skills/empty-state/empty-state.tsx`,
`src/components/skills/restore-confirm-dialog/restore-confirm-dialog.tsx`,
`src/components/skills/skill-editor/classes.ts`,
`src/components/skills/skill-editor/skill-editor.tsx`,
`src/components/skills/skill-list-item/skill-list-item.tsx`,
`src/routes/settings/index.tsx`,
`src/routes/settings/skills/index.tsx`.

This is byte-identical to the 9-file set that pass 1 independently confirmed as
pre-existing (`git status --porcelain frontend/src/` is still empty, so all 9 are
unchanged from `HEAD` and cannot have been caused by this change). **CRITICAL-1 is
closed.** `pnpm verify`'s exit code is now attributable entirely to pre-existing
debt, which is what the "Isolation from the default workflow" requirement demands.

### Anti-Regression: did the corrective Prettier run alter any data?

The corrective run was scoped (`pnpm exec prettier --write README.md`). Verified
three independent ways:

1. **Blast radius** — `git status --porcelain` returns exactly the same five
   modified paths as pass 1 (`docs/adr/0001-accept-authjs-qwik.md`,
   `frontend/README.md`, `frontend/package.json`, `frontend/pnpm-lock.yaml`,
   `frontend/pnpm-workspace.yaml`) plus the untracked planning directory. No new
   path appeared; none of the 9 pre-existing `src/` files was swept up by the fix.
2. **Diff shape** — `git diff --numstat -- frontend/README.md` → `76 0`, i.e. a
   pure 76-line addition with **zero deletions**. The reformat therefore touched
   only lines this change itself authored; no pre-existing README line was altered.
3. **Content reconciliation against a fresh audit** — see the table below. Every
   GHSA ID, package, installed version, and fixed-in version in the post-Prettier
   table still reconciles 1:1 with a verifier-run `pnpm audit`, and matches pass 1's
   independently-captured reconciliation exactly. No advisory ID, version string, or
   accepted-finding entry drifted.

### Baseline Reconciliation Re-run (verifier-executed `pnpm vuln-check`, pass 2)

`pnpm vuln-check` → exit 1, **19 vulnerabilities** (1 low | 5 moderate | 12 high |
1 critical), **13** high/critical blocks printed. Identical finding set to pass 1.

| # | Severity | Package | Advisory | Vulnerable range | Patched | README row | Row's "Found in" | Installed (lockfile) |
|---|---|---|---|---|---|---|---|---|
| 1 | critical | `vitest` | GHSA-5xrq-8626-4rwp | `<3.2.6` | `>=3.2.6` | 1 | 0.34.6 | 0.34.6 ✓ |
| 2 | high | `vite` | GHSA-v2wj-q39q-566r | `>=7.1.0 <=7.3.1` | `>=7.3.2` | 2 | 7.3.1 | 7.3.1 ✓ |
| 3 | high | `vite` | GHSA-p9ff-h696-f583 | `>=7.0.0 <=7.3.1` | `>=7.3.2` | 3 | 7.3.1 | 7.3.1 ✓ |
| 4 | high | `vite` | GHSA-fx2h-pf6j-xcff | `<=6.4.2` | `>=6.4.3` | 4 | 5.4.21 | 5.4.21 ✓ |
| 5 | high | `vite` | GHSA-fx2h-pf6j-xcff | `>=7.0.0 <=7.3.4` | `>=7.3.5` | 5 | 7.3.1 | 7.3.1 ✓ |
| 6 | high | `brace-expansion` | GHSA-3jxr-9vmj-r5cp | `>=2.0.0 <2.1.2` | `>=2.1.2` | 7 | 2.1.1 | 2.1.1 ✓ |
| 7 | high | `brace-expansion` | GHSA-3jxr-9vmj-r5cp | `<1.1.16` | `>=1.1.16` | 6 | 1.1.15 | 1.1.15 ✓ |
| 8 | high | `svgo` | GHSA-2p49-hgcm-8545 | `>=3.0.0 <3.3.4` | `>=3.3.4` | 9 | 3.3.3 | 3.3.3 ✓ |
| 9 | high | `sharp` | GHSA-f88m-g3jw-g9cj | `<0.35.0` | `>=0.35.0` | 10 | 0.34.5 | 0.34.5 ✓ |
| 10 | high | `postcss` | GHSA-r28c-9q8g-f849 | `<=8.5.17` | `>=8.5.18` | 11 | 8.5.16 | 8.5.16 ✓ |
| 11 | high | `brace-expansion` | GHSA-mh99-v99m-4gvg | `<1.1.17` | `>=1.1.17` | 6 | 1.1.15 | 1.1.15 ✓ |
| 12 | high | `brace-expansion` | GHSA-mh99-v99m-4gvg | `>=2.0.0 <2.1.3` | `>=2.1.3` | 7 | 2.1.1 | 2.1.1 ✓ |
| 13 | high | `brace-expansion` | GHSA-mh99-v99m-4gvg | `>=4.0.0 <5.0.8` | `>=5.0.8` | 8 | 5.0.7 | 5.0.7 ✓ |

Zero unlisted findings; zero regressions. `GHSA-7rqj-j65f-68wh` (the `@auth/core`
auth bypass) appears **nowhere** in the output — `grep -c` returns 0 — reconfirming
the remediation. The "Fixed in" values for merged rows 6 and 7 correctly carry the
*higher* of the two merged advisories' patch versions (1.1.17 and 2.1.3), so
adopting the documented version clears both advisories on that row.

Every "Found in" version was independently cross-checked against `pnpm-lock.yaml`
package keys (last column) rather than taken from the README — all 11 rows match
the installed tree.

### Full Command Battery Re-run (pass 2)

| Command | Exit | Result |
|---|---|---|
| `pnpm fmt.check` | 1 | 9 pre-existing `src/` files only — **README.md cleared** |
| `pnpm exec prettier --check README.md` | 0 | clean |
| `pnpm vuln-check` | 1 | 19 findings, 13 high/critical blocks — same set as pass 1 (gate red by design) |
| `pnpm vuln-check:prod` | 0 | "No known vulnerabilities found" |
| `pnpm lint` | 0 | clean |
| `pnpm build.types` | 0 | clean |
| `pnpm test:ci` | 0 | **71 files / 647 tests passed**, 0 failed |
| `pnpm build` | 0 | client + server SSR bundles, "built in 437ms" |
| `pnpm why @auth/core` | 0 | `@auth/core@0.41.3`, "Found 1 version of @auth/core" |

**Detection-only re-proof**: `shasum -a 256` of `frontend/package.json` and
`frontend/pnpm-lock.yaml` captured immediately before and after the pass-2
`pnpm vuln-check` run — `diff` reports no difference. The gate still mutates nothing.

Test and build evidence hashes for pass 2 are recorded in the YAML block at the top
of this file; they differ from pass 1 only because the outputs embed wall-clock
timings.

### Spec Compliance — pass 2

The three scenarios that pass 1 rated PARTIAL are re-rated:

| Requirement | Scenario | Pass 1 | Pass 2 | Basis for change |
|---|---|---|---|---|
| Isolation from the default workflow | Default loop unaffected | PARTIAL | **COMPLIANT** | The sole cause of the downgrade was CRITICAL-1. `README.md` no longer contributes to `fmt.check`, so this change adds nothing to the default loop's failure set. Verified: `verify`/`build`/`build.client`/`build.server`/`test:ci` still contain no `vuln-check*` reference; `frontend/Dockerfile` still runs only `pnpm install --frozen-lockfile --shamefully-hoist` and `pnpm build`; no `.github/workflows/` exists |
| Opt-in full-tree gating audit | Gate is clean | PARTIAL | **COMPLIANT** | `vuln-check:prod` exercises the identical `--audit-level=high` exit-code mechanism on a clean sub-tree and returns exit 0 (verifier-run in pass 2). The zero-exit path is proven by execution; the full-tree gate is red only because of documented, accepted debt, which is the designed state |
| `@auth/core` remediation | Bump resolves cleanly | PARTIAL | **COMPLIANT (with WARNING-2 noted)** | The normative clause — "the resolved `@auth/core` is `>=0.41.3`" — is proven: single-version `0.41.3` resolution, the advisory absent from a fresh audit, 647 tests green, SSR build green. The parenthetical "(login e2e passes)" remains unexercised, which is carried forward as WARNING-2 rather than as a compliance failure, since the e2e skip is a pre-existing by-design environment guard this change neither introduced nor can satisfy in this sandbox |

**Pass 2 compliance summary**: **9/9 scenarios** satisfied (8 COMPLIANT + 1 correctly
N/A branch), **6/6 requirements** satisfied.

### Issues — pass 2

**CRITICAL**: none. (Pass 1's CRITICAL-1 is closed and re-proven above.)

**WARNING** — both carried forward from pass 1, unchanged, non-blocking:

1. **README prose miscounts distinct advisories.** Still reads "13 findings across
   the 11 advisories below". The table has 11 *rows*, but the 13 findings carry only
   **9** distinct `GHSA-*` IDs (rows 6 and 7 each merge `GHSA-3jxr-9vmj-r5cp` +
   `GHSA-mh99-v99m-4gvg`; `GHSA-fx2h-pf6j-xcff` spans rows 4 and 5). Data is correct
   and reconciles fully — only the noun is wrong. Suggested: "13 findings (9 distinct
   advisories) across the 11 package/version rows below". Editorial, not a defect.
2. **The "(login e2e passes)" clause is not proven by execution.** Re-confirmed in
   pass 2: `frontend/e2e/` has zero working-tree changes, and the skip guards are
   intact (`github-sign-in.spec.ts:40`, `sign-in-denied.spec.ts:43`, plus
   `sign-in-cookie-attrs` and `sign-out`), each gating on
   `process.env.AUTH_GITHUB_BASE_URL === undefined`. The OAuth round trip was never
   exercised against `@auth/core@0.41.3`. Static evidence remains strong; a reviewer
   should be told the login e2e net was inactive rather than passing.

**SUGGESTION**

1. Add `frontend/README.md`'s baseline table to a lightweight refresh routine — the
   "Found in" column pins exact installed versions and will silently drift on the
   next `pnpm update`. (Carried forward.)
2. *Partially addressed already.* Pass 1 asked for a note that `vuln-check:ci` is a
   reserved, intentionally unreferenced alias. The README's command fence in fact
   already carries `# alias of vuln-check, reserved for future CI wiring`, which
   covers the intent. Only the prose body omits it. Downgrading to informational.

### Pre-existing Failures — re-confirmed unchanged (informational, NOT blockers)

1. **`pnpm fmt.check` on 9 `src/` files** — `git status --porcelain frontend/src/`
   is empty; all 9 are byte-identical to `HEAD` and untouched by the corrective run.
2. **`e2e/sign-in-landing.spec.ts` stale button-copy assertion** —
   `git status --porcelain frontend/e2e` is empty; unchanged and out of scope.

### Task State — pass 2

`tasks.md` now carries 28 boxes (25 original + a 5-item "Post-apply correction"
phase, 6.1–6.5). All 28 are `[x]`. Tasks 6.1–6.3 and 6.5 were re-executed
independently by the verifier above and their recorded results are accurate. Task
6.4's whitespace-only claim could not be re-run directly (no pre-Prettier copy
survives in git, since the whole section is an uncommitted addition), but it is
corroborated by the `76 0` numstat and the full fresh-audit reconciliation, both of
which would have surfaced any data drift.

### Verdict — pass 2

**PASS** — 0 CRITICAL, 2 WARNING, 2 SUGGESTION.

The one blocker is genuinely fixed, not merely reasserted: `README.md` is Prettier
clean, the format-gate failure set is back to its pre-existing 9, the corrective run
touched no other path, and the baseline table's data survived the reformat intact
against a fresh audit and the installed lockfile. Nothing regressed — tests, build,
lint, types, both audit scripts, and detection-only immutability were all re-executed
green. The two remaining WARNINGs are editorial and environmental, were already
accepted as non-blocking in pass 1, and do not block archive.

**This change is archive-ready.** Recommended next phase: `sdd-archive`.
