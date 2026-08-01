# Spec — frontend-dependency-audit

> **Change**: `frontend-vuln-check` · **Phase**: spec (delta, new capability)
> **Canonical spec**: `openspec/specs/frontend-dependency-audit/spec.md` — created by `sdd-archive`
> **Format**: RFC 2119 keywords + Given/When/Then, per `openspec/config.yaml`
> **Date**: 2026-08-01

## Definitions

- **The gate** — the single `frontend/package.json` script that is the authoritative pass/fail vulnerability check.
- **Full tree** — the entire resolved dependency graph, including `devDependencies`.
- **Baseline** — the README-documented set of known, accepted, unremediated findings.

## ADDED Requirements

### Requirement: Opt-in full-tree gating audit

`frontend/` MUST expose one opt-in `package.json` script that is the gate. The gate MUST scan the full tree, MUST NOT be scoped to production dependencies, and MUST exit non-zero when a finding at or above the gate severity exists. A second, non-gating informational script MAY exist (e.g. production-scoped triage); it MUST NOT be described as the gate.

(Supersedes the proposal's two-tier shape, where the production-scoped script was the gate. Per user decision, gate scope is full tree.)

#### Scenario: Gate detects a full-tree finding

- GIVEN a clean install of `frontend/`
- WHEN a maintainer runs the gate script
- THEN it audits the full tree including `devDependencies`
- AND it exits non-zero, reporting each finding with its advisory identifier and severity

#### Scenario: Gate is clean

- GIVEN a resolved tree with no finding at or above the gate severity
- WHEN a maintainer runs the gate script
- THEN it exits zero

#### Scenario: Informational script does not gate

- GIVEN an informational audit script exists
- WHEN it is run
- THEN its exit status MUST NOT be used as the pass/fail signal for the change

### Requirement: Detection only

The audit capability MUST be detection-only. It MUST NOT install, upgrade, downgrade, or otherwise mutate dependencies, `package.json`, or `pnpm-lock.yaml` during its own execution.

#### Scenario: Audit leaves the tree unchanged

- GIVEN a committed `package.json` and `pnpm-lock.yaml`
- WHEN the gate script runs and reports findings
- THEN both files are byte-for-byte unchanged

### Requirement: Isolation from the default workflow

The audit scripts MUST remain opt-in. They MUST NOT be invoked by `verify`, `build`, `test:ci`, or the Docker build, and this change MUST NOT introduce a CI workflow.

#### Scenario: Default loop unaffected

- GIVEN the change is applied
- WHEN `pnpm verify`, `pnpm build`, and the Docker image build run
- THEN their behaviour and exit statuses are unchanged from before the change
- AND no network call to an advisory endpoint occurs

### Requirement: `@auth/core` remediation

`@auth/core` MUST be resolved to `>=0.41.3`, closing GHSA-7rqj-j65f-68wh on the application's only login path. If `@auth/qwik@0.9.2` (pinned by ADR-0001) makes this unresolvable without breaking that pin, the bump MUST be dropped from this change, the finding MUST be recorded in the baseline, and a follow-up change to reconcile ADR-0001 MUST be noted.

#### Scenario: Bump resolves cleanly

- GIVEN `@auth/core@0.41.2` is a direct production dependency
- WHEN the bump is applied and the lockfile regenerated
- THEN the resolved `@auth/core` is `>=0.41.3`
- AND authentication behaviour is unchanged (login e2e passes)

#### Scenario: Bump is blocked by ADR-0001

- GIVEN `@auth/qwik@0.9.2` constrains `@auth/core` below `0.41.3`
- WHEN resolution fails without breaking the ADR-0001 pin
- THEN the bump is dropped from this change
- AND GHSA-7rqj-j65f-68wh is documented as accepted baseline debt with a named follow-up

### Requirement: Documented baseline

`frontend/README.md` MUST carry a "Vulnerability scanning" section stating that the check is detection-only and full-tree, and a "Current baseline" subsection recording the pnpm version, the snapshot date, and each known accepted finding. The `vitest` critical (GHSA-5xrq-8626-4rwp, dev-only) MUST be listed as accepted baseline debt, with the note that it keeps the gate non-zero until a separate upgrade change lands. A "Remediation history" subsection MUST record the `@auth/core` outcome.

#### Scenario: Reviewer reconciles a red gate

- GIVEN the gate exits non-zero after this change
- WHEN a reviewer compares reported findings against "Current baseline"
- THEN every reported finding is either listed there or is a genuine regression

### Requirement: Package manager pinning

`frontend/package.json` MUST pin the package manager via `packageManager`, at a pnpm version containing the `/advisories/bulk` fix, so audit results cannot silently regress. `engines.pnpm` SHOULD state the same minimum.

#### Scenario: Pinned toolchain

- GIVEN `frontend/Dockerfile` activates corepack
- WHEN the image builds
- THEN corepack resolves the pinned pnpm version from `packageManager`
