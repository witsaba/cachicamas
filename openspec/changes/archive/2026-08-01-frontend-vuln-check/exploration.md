# Exploration: opt-in SCA (dependency vulnerability check) for `frontend/`

## Current State

- `frontend/` is a standalone pnpm workspace: its own `pnpm-workspace.yaml` (`allowBuilds`) and `.npmrc` (`shamefully-hoist=true`). There is no repo-root `package.json`/`pnpm-workspace.yaml`.
- `frontend/package.json` declares no `packageManager` field, despite `frontend/Dockerfile:32` assuming corepack pins to one — corepack has nothing to actually pin to today.
- `verify = pnpm lint && pnpm build.types && pnpm fmt.check && pnpm test:ci` has no network egress today. No `.github/workflows` exist anywhere in the repo.
- Production `dependencies` (6): `@auth/core@0.41.2`, `@auth/qwik@0.9.2`, `@panva/hkdf@1.2.1`, `marked`, `postgres`, `zod`. `devDependencies` (20) include eslint/prettier/tailwind/typescript/vite/vitest/@vitest-ui/playwright.
- Orchestrator-confirmed live `pnpm audit`: 22 findings (1 low, 6 moderate, 13 high, 2 critical). The two criticals:
  - `vitest <3.2.6` — dev-only, arbitrary file read/execute via the Vitest UI server (GHSA-5xrq-8626-4rwp).
  - `@auth/core >=0.1.0 <0.41.3` — production dependency, installed exactly `0.41.2`, auth bypass via homoglyph email normalization (GHSA-7rqj-j65f-68wh). Pinned directly in `dependencies` (not just transitively via `@auth/qwik`), so remediation is a plain version bump — out of scope for this change.
- `docs/adr/0001-accept-authjs-qwik.md` pins `@auth/qwik@0.9.2` for pre-1.0 API-stability reasons and predates this effort; does not reference the `@auth/core` CVE.
- No existing frontend security/audit tooling anywhere in `frontend/src` — greenfield for this ecosystem.

## Backend precedent (`backend/database_administrator`, PR #97, merged)

- `Makefile`: pins `GOVULNCHECK_VERSION := v1.1.4`; `make vuln-check` auto-installs into `./bin` if missing, then runs `govulncheck ./...`; `make vuln-check/ci` is an explicit alias reserved for future CI wiring. Neither target is part of `all: tidy fmt vet lint test build`.
- `README.md` has a "Vulnerability scanning" section explaining `govulncheck` is **reachability-aware** (only reports advisories whose vulnerable symbols are call-reachable), plus a "Current baseline" subsection (tool version + DB snapshot date + finding count) and a "Remediation history" subsection.
- Detection-only, no auto-fix.

## Key ecosystem difference vs backend

`pnpm audit` is NOT reachability-aware like `govulncheck` — it flags every vulnerable version range in the resolved tree regardless of call-reachability. This is why the frontend surfaces 22 raw findings versus the backend's "zero reachable" state, and why `--prod` scoping plus an explicit severity gate matter more here than on the Go side.

## Affected Areas

- `frontend/package.json` — new `scripts` entries only (no runtime code changes).
- `frontend/README.md` — new "Vulnerability scanning" section mirroring the backend's.
- Possibly `engines`/`packageManager` in `frontend/package.json` — to pin a minimum pnpm version so this script can't silently regress (correctness depends on pnpm's `/advisories/bulk` fix, PR #11268, merged 2026-04-15).
- `docs/adr/0001-accept-authjs-qwik.md` — candidate cross-reference (not required now; relevant once `@auth/core` is remediated).

## Approaches Considered

1. **Two-tier `package.json` scripts, opt-in, not in `verify`** — `"audit": "pnpm audit --prod --audit-level=high"` (gating, prod-scoped) + `"audit:full": "pnpm audit --audit-level=moderate"` (informational, full tree).
   - Pros: mirrors backend's opt-in shape; `--prod` keeps the gate focused on shippable risk (excludes the dev-only vitest critical); no new tooling to install (pnpm audit is built-in).
   - Cons: two scripts to document; severity threshold (`high` vs `critical`) is a judgment call for spec/design.
   - Effort: Low.
2. **Single full-tree script** — `"audit": "pnpm audit --audit-level=high"`.
   - Pros: simplest, one command.
   - Cons: conflates dev-only noise with production risk in one gate; weaker signal than the backend's reachability-aware model.
   - Effort: Low.
3. **Wire into a new CI workflow now** — first `.github/workflows/*` in the repo.
   - Pros: closes the loop immediately.
   - Cons: no CI exists anywhere in-repo (backend didn't get one either); disproportionate to precedent; unresolved network-egress-in-CI policy.
   - Effort: Medium-High.

## Recommendation

Approach 1 — two-tier opt-in scripts (`audit` prod/high-gated, `audit:full` informational), not added to `verify`, documented in `frontend/README.md` with the same "Current baseline"/"Remediation history" shape as the backend. No new CI workflow. Design should also decide whether to pin a minimum pnpm version given the audit's correctness depends on pnpm's `/advisories/bulk` fix.

## Risks

- Enabling this immediately surfaces a real, actionable, production-scoped critical (`@auth/core@0.41.2`) — proposal/design must decide: fix now, or document-as-baseline-and-defer (mirroring the backend's original "5 findings, remediated later" pattern).
- `pnpm audit` noise floor is high (not reachability-aware); a loose `--audit-level` risks constant red / alert fatigue.
- No pnpm version pin exists (`packageManager`/`engines.pnpm` missing), so a contributor on a pre-fix pnpm could silently reintroduce the audit-endpoint breakage.

## Ready for Proposal

Yes.
