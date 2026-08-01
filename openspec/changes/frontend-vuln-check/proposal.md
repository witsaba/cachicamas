# Proposal: Opt-in dependency vulnerability check for `frontend/`

## Intent

`backend/database_administrator` has an opt-in `make vuln-check` SCA gate (PR #97); `frontend/` has none, so JavaScript dependency risk is invisible and unowned. A live `pnpm audit` already reports 22 findings including a **critical auth bypass in `@auth/core@0.41.2`** (GHSA-7rqj-j65f-68wh, homoglyph email normalization) — a directly pinned production dependency on this app's only login path. We need a repeatable frontend equivalent, and the finding it surfaces must not ship as accepted debt.

## Scope

### In Scope

- Two opt-in `frontend/package.json` scripts: a gating prod-scoped audit and an informational full-tree audit. Neither is wired into `verify`.
- Bump `@auth/core` to `>=0.41.3`, closing the production auth bypass, so the new gate is green on its first run.
- Pin the package manager (`packageManager`) — audit correctness depends on pnpm's April-2026 `/advisories/bulk` fix, and `frontend/Dockerfile:32` already assumes corepack has something to pin.
- `frontend/README.md` "Vulnerability scanning" section mirroring the backend's shape, including "Current baseline" and "Remediation history".

### Out of Scope

- `vitest` critical (GHSA-5xrq-8626-4rwp): dev-only and a `0.34 → 3.2.6` major jump; deferred to its own change.
- The remaining dev-tree findings — documented as baseline, not remediated.
- Any CI workflow (none exists anywhere in this repo) and any auto-fix behaviour.
- Revisiting ADR-0001's `@auth/qwik@0.9.2` pin.

## Capabilities

### New Capabilities

- `frontend-dependency-audit`: opt-in SCA commands, severity/scope gate semantics, exit-code contract, and the documented baseline obligation.

### Modified Capabilities

- None.

## Approach

Two-tier scripts (exploration Approach 1). The gate scopes to production dependencies at high severity so it reflects shippable risk, not dev-tool noise — `pnpm audit` is not reachability-aware, unlike `govulncheck`, so scope and threshold do the filtering the Go tool does statically. Remediating `@auth/core` in the same PR is deliberate: a security gate that ships already red trains the team to ignore it.

## Affected Areas

| Area | Impact | Description |
|------|--------|-------------|
| `frontend/package.json` | Modified | Audit scripts, `@auth/core` bump, `packageManager` pin |
| `frontend/pnpm-lock.yaml` | Modified | Lockfile resolution for the bump |
| `frontend/README.md` | Modified | Vulnerability scanning + baseline sections |
| `docs/adr/0001-accept-authjs-qwik.md` | Modified | Cross-reference the remediation |

## Risks

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| `@auth/qwik@0.9.2` constrains `@auth/core` below `0.41.3` | Med | If unresolvable without breaking ADR-0001, document as baseline debt and split remediation into a follow-up change |
| Auth regression from the bump | Med | Run `pnpm verify` plus the Playwright login e2e before merge |
| Advisory churn makes the gate flaky | Med | Gate is opt-in and not in `verify`; noise stays out of the default loop |
| Baseline table drifts from reality | Low | Section records tool version and snapshot date |

## Rollback Plan

Revert the single PR. The scripts are additive and unreferenced by `verify`, `build`, or the Dockerfile, so removal restores prior behaviour exactly. If only the dependency bump misbehaves, restore `@auth/core@0.41.2` in `package.json`, re-run `pnpm install --lockfile-only`, and record the finding in "Current baseline" instead.

## Dependencies

- Network egress to the npm advisory endpoint at audit time.
- pnpm >= the version containing the `/advisories/bulk` fix.

## Success Criteria

- [ ] The gating audit script exits zero on a clean install after the `@auth/core` bump.
- [ ] The gating script exits non-zero when a production dependency at the gate severity is present.
- [ ] The informational script reports the full tree without failing the gate.
- [ ] `verify`, `build`, and the Docker build are byte-for-byte unaffected in behaviour.
- [ ] `frontend/README.md` documents the commands, the current baseline with date, and the `@auth/core` remediation.
