# Proposal: Repo-level validation harness

> **Change**: `repo-validation-harness` · **Builds on**: [`explore.md`](./explore.md)

## Intent

No command today proves "tests + linters + type-checks + vuln scans green" across `frontend/` and three `backend/*` Go modules. Add `scripts/validate.py`: emit a stable report (`validation-report.json` + human `validation-report.md`) consumed by an AI agent fixing every finding.

## Scope

**In scope** — `scripts/validate.py` (uv-shebang Python; stdlib only); two outputs from one in-memory list; orchestration: Go modules `make test`/`test/cover`/`lint`/`vuln-check` (where defined); frontend `pnpm lint`/`build.types`/`fmt.check`/`test:ci`, `node --test eslint-rules/*.spec.mjs`, `pnpm run vuln-check`; repo-root `go work edit -json` drift check; run-all-then-aggregate (never fail-fast); timeout 600s/check (e2e 1200s); exit 0/1; CI-ready contract (deterministic exit, no TTY, valid JSON, stable sha256 finding IDs).

**Out of scope** —
- **CI wiring (`.github/workflows/`)** — *deferred but related.* Go 1.26 + Node/pnpm + compose exceeds 400-line budget; own change once exit codes stabilize. Harness stays CI-ready.
- **Modifying any Makefile** — *deferred but related.* `backend/agent/Makefile` lacks `install`/`run`/`vuln-check`/`vuln-check/ci`; `workspace_syncer/Makefile` lacks `test/integration`. Surface as `tool-missing`/medium findings; fixing is a separate ADRed change.

## Capabilities

**New** — `repo-validation-harness`: checks per scope; finding shape (id, category, tool, scope, severity, rule, file, line, column, message, evidence, reproduce, fix_hint); run semantics (run-all-then-aggregate; exit 0/1); opt-in `--include-integration` and `--include-e2e`; deterministic-output guarantees. **Modified** — none.

## Approach

`scripts/validate.py` with `uv` shebang. Python `json`+`dataclasses` → one in-memory list rendered both ways. Bash needs fragile `jq` over 7 grammars; Go adds a new code root without benefit. Per-tool JSON: `golangci-lint --out-format=json`, `eslint --format json`, `vitest --reporter=json`, `pnpm audit --json`, `govulncheck -json`. Stdout-parse: `tsc`, `prettier --check`, `go test -race -v`. Tool-missing: `backend/agent` missing `vuln-check` → `tool-missing`/medium (NOT bypassed; ADR 0005's gap); `install` allowlisted (Makefile 55–58); `frontend/eslint-rules/` runs as own `node --test` check (excluded from vitest at `vite.config.ts:41-43`).

## Affected areas

`scripts/validate.py` (new, ~400 lines, stdlib); `validation-report.{json,md}` (new artifacts). Makefiles, `docker-compose.yaml`, `infra/`, `backend/*/src/` **untouched**. `.github/workflows/` not created.

## ADR question

"New top-level dependency ⇒ ADR first" (`openspec/AGENTS.md` 24, 90) **not** triggered: rule binds top-level **Go** deps (line 90 names "Go"). `validate.py` adds none. Python deps, if any, per-invocation via `uv run --with` (`validate-infra.py` precedent). v1 stdlib only.

## Risks

Network auto-install → `tool-missing`/high + `host.versions_installed` in header. Cold-cache 5–25 min → per-check elapsed + total. Go drift (`go.work` 1.26.5 vs two `go.mod` 1.26.3) → `go.work-mismatch` finding. `backend/agent` Makefile drift → `make -n <target>` enumeration. JSON/Markdown drift → unit test asserts parity. Custom ESLint `warn` → preserve tool severity. E2E needs live stack → opt-in `--include-e2e` with "skipped" note.

## Rollback

Purely additive. `git rm scripts/validate.py validation-report.{json,md}`. No Makefile, source, compose, or Go module touched.

## Dependencies

`uv` on PATH. Go 1.26.x, `golangci-lint v2.9.0`, `govulncheck v1.1.4`, `goimports`. `pnpm 11.8.0`, Node 18.17+/20.3+/21+. `jq` not required.

## Success criteria

- [ ] `validate.py` runs end-to-end; both reports from one in-memory list.
- [ ] Every check executes; 0 findings = exit 0; ≥1 = exit 1.
- [ ] `backend/agent` missing `vuln-check` → `tool-missing`/medium; `install` allowlisted.
- [ ] Go-version drift → `go.work-mismatch` finding.
- [ ] `--include-integration`/`--include-e2e` honored; "skipped" note when absent.
- [ ] CI-ready: deterministic exit, no TTY, `python3 -m json.tool` parses, stable sha256 IDs.
- [ ] No `backend/*/src/`, `docker-compose.yaml`, `infra/`, or Makefile modified.
