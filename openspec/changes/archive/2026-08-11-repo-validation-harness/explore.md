# Exploration: single repo-level validation script for `frontend/` + `backend/`

> **Change**: `repo-validation-harness`
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-10
> **Filename note**: the repo's existing convention is `explore.md` (e.g.
> `openspec/changes/archive/2026-08-09-cachicamas-ai-live-smoke/explore.md`,
> `…/2026-08-08-cachicamas-ai-observability/explore.md`, and 38 other archived
> changes). The generic skill text calls this artifact `exploration.md`; one
> archive (`2026-08-01-frontend-vuln-check/exploration.md`) used the longer
> name and is the only outlier. **This artifact is `explore.md` to match the
> dominant repo convention.** Discrepancy flagged for the orchestrator.

---

## Current State

### What runs today, where

There is **no** CI: `.github/workflows/` does not exist anywhere in the
repo (verified with `glob .github/workflows/**` → 0 matches). The only
existing automation is two human-driven scripts in `scripts/`:

- `scripts/validate-infra.py` — composes + Docker + Dockerfile sanity (PyYAML
  via `uv run --with pyyaml`).
- `scripts/human-run-tail-sampling-verify.sh` — Otel-collector tail-sampling
  smoke test (manual; needs compose stack already up).

Neither runs tests, lint, format, type-check, or vulnerability scans across
`frontend/` + `backend/`.

### Frontend surface (`frontend/package.json`, lines 12–36)

Confirmed scripts:

| Script            | Body                                      | Line | In `verify` (line 13)? |
|-------------------|-------------------------------------------|------|------------------------|
| `lint`            | `eslint "src/**/*.ts*"`                   | 24   | yes (first)            |
| `build.types`     | `tsc --incremental --noEmit`              | 18   | yes                    |
| `fmt.check`       | `prettier --check .`                      | 23   | yes                    |
| `test:ci`         | `vitest --run`                            | 28   | yes (last)             |
| `test.unit`       | `vitest`                                  | 27   | no (interactive)       |
| `test:e2e`        | `playwright test`                         | 30   | **no — and needs stack-up** |
| `vuln-check`      | `pnpm audit --audit-level=high`           | 32   | **no**                 |
| `vuln-check:prod` | `pnpm audit --prod --audit-level=high`    | 33   | **no**                 |
| `vuln-check:ci`   | `pnpm run vuln-check`                     | 34   | **no**                 |

The orchestrator's preliminary audit is correct here: `verify` excludes
`vuln-check` (3 variants) and `test:e2e`.

Engine constraints (`frontend/package.json`, lines 5–9): `node ^18.17.0 ||
^20.3.0 || >=21.0.0`; `pnpm >=11.8.0`; `packageManager: pnpm@11.8.0`.
Confirmed: `pnpm`, `uv`, `jq`, `go1.26.5` are all on the host PATH.

### Frontend ESLint config (`frontend/eslint.config.js`)

- Flat config; one **custom local rule** is wired in:
  `cachicamas/no-inline-button-class` (line 82, severity `warn`).
  Implementation: `frontend/eslint-rules/no-inline-button-class.mjs` (123
  lines). Tests: `frontend/eslint-rules/no-inline-button-class.spec.mjs`.
  Run via `node --test` (per the spec file header, line 13) — **not** via
  vitest (`frontend/vite.config.ts` line 41–43 excludes `eslint-rules/**`).
- Qwik rules from `eslint-plugin-qwik` are active (line 54). The Qwik
  plugin may emit Qwik-specific diagnostics that need to be preserved
  verbatim in the report.

### Frontend Playwright config (`frontend/playwright.config.ts`)

- `webServer` (line 47–56): when `E2E_BASE_URL` is unset, Playwright
  auto-starts `pnpm dev` on port 5173. When `E2E_BASE_URL` is set, Playwright
  does **not** start anything and trusts the caller has a stack at that URL.
- `fullyParallel: false`; `workers: 1` (line 34–35) — playwright is
  intentionally serial because DB writes collide.
- `retries: 0` (line 36) — failures are first-time-fatal.
- Without `E2E_BASE_URL`, the script must have `docker compose up` (Postgres
  + `database_administrator`) running **before** `pnpm test:e2e`. The
  header comment (lines 14–22) is explicit about this. **E2E is therefore
  opt-in for any harness that does not bring its own stack.**

### Backend Makefiles — three, diverged

Read in full:

| Target / variable                  | `database_administrator`     | `workspace_syncer`           | `agent`                     |
|------------------------------------|------------------------------|------------------------------|-----------------------------|
| `help`                             | yes                          | yes                          | yes                         |
| `install` (deps download)          | yes (line 56–59)             | yes (line 44–47)             | **NO — comment lines 55–58: "There is no `install` target on purpose. This module carries ZERO requires…"** |
| `tidy`                             | yes (line 62–64)             | yes (line 50–52)             | yes (line 61–63)            |
| `tools`                            | yes (line 67–75)             | yes (line 55–63)             | yes (line 66–74)            |
| `vuln-check` (`govulncheck`)       | yes (line 78–87)             | yes (line 66–75)             | **NO** — confirmed by grep (`govulncheck` returns 0 matches under `backend/agent/`) |
| `vuln-check/ci`                    | yes (line 90–91)             | yes (line 77–78)             | **NO**                      |
| `build`                            | yes (line 96–99) — produces `./bin/database_administrator` | yes (line 83–87) — `./bin/workspace_syncer` | yes (line 83–85) — **`go build ./...`**, no binary (no `main` yet) |
| `run`                              | yes (line 102–104)           | yes (line 89–91)             | **NO** (no `main` package)  |
| `test`                             | yes (line 109–111)           | yes (line 97–99)             | yes (line 90–92)            |
| `test/cover`                       | yes (line 114–117)           | yes (line 102–105)           | **YES** (line 94–98) — orchestrator's open question is answered: `agent` does NOT differ on `test/cover`; it differs on `install`, `run`, `vuln-check`, `vuln-check/ci`. |
| `test/integration` (boots compose Postgres, gated on `INTEGRATION=1`) | yes (line 143–154) | **NO** | **NO** (no I/O surface yet) |
| `fmt`                              | yes (line 159–162)           | yes (line 110–113)           | yes (line 102–106)          |
| `vet`                              | yes (line 165–167)           | yes (line 116–118)           | yes (line 108–111)          |
| `lint`                             | yes (line 170–176) — `vet` + `golangci-lint run --config=.golangci.yml ./...`; **auto-installs** if missing | yes (line 121–127) — same shape | yes (line 113–120) — same shape |
| `clean`                            | yes (line 181–183)           | yes (line 131–133)           | yes (line 124–127)          |
| `all`                              | `tidy fmt vet lint test build` (line 185–186) | same | same |
| `LOCALBIN`                         | `bin` (line 24)              | `bin` (line 12)              | `bin` (line 27)             |
| `GOLANGCI_LINT_VERSION`            | `v2.9.0` (line 28)           | `v2.9.0` (line 16)           | `v2.9.0` (line 31)          |
| `GOIMPORTS_VERSION`                | `latest` (line 29)           | `latest` (line 17)           | `latest` (line 32)          |
| `GOVULNCHECK_VERSION`              | `v1.1.4` (line 30)           | `v1.1.4` (line 18)           | **N/A**                     |

**The orchestration script CANNOT assume Makefile symmetry.** The
`backend/agent/Makefile` is documented as a deliberate copy of
`database_administrator`'s with an explicit drift-acceptance comment
(lines 1–6), and ADR 0005 (`docs/adr/0005-promote-agent-stack-to-own-module.md`)
is the policy owner. The missing `vuln-check` target is **not a bug**;
it's an unrecorded decision.

### Backend `.golangci.yml` — same linters, divergent configs

Read in full:

| File                                      | Lines | Linters                              | Extra config                                                                                |
|-------------------------------------------|-------|--------------------------------------|---------------------------------------------------------------------------------------------|
| `backend/database_administrator/.golangci.yml` | 30    | `govet`, `errcheck`, `staticcheck`, `unused`, `revive` | `issues-exit-code: 1`, `output: {format: colored-line-number, print-issued-lines: true, print-linter-name: true}` |
| `backend/workspace_syncer/.golangci.yml`      | 19    | same five                            | `issues: { max-issues-per-linter: 50, max-same-issues: 0 }` (no `output` block)             |
| `backend/agent/.golangci.yml`                 | 30    | same five                            | byte-identical to `database_administrator` (verified by inspection — same lines 1–30)       |

The orchestrator's question "same config or divergent?" is answered:
**divergent on `output` / `issues` blocks, identical on linter set and
`run.tests: true` / `run.timeout: 5m`**. `--out-format=json` on
`golangci-lint run` overrides the YAML `output:` block, so a harness can
get a single JSON shape from all three without touching the configs.

### Go version drift (orchestrator audit correction)

Orchestrator said `go 1.26.5`. **Partially wrong — there are TWO versions
in this repo right now:**

| File                                | Declared Go |
|-------------------------------------|-------------|
| `go.work` (line 1)                  | `1.26.5`    |
| `backend/database_administrator/go.mod` (line 3) | `1.26.5` |
| `backend/workspace_syncer/go.mod` (line 3)        | **`1.26.3`** |
| `backend/agent/go.mod` (line 3)                   | **`1.26.3`** |

`go.work` at `1.26.5` and two modules at `1.26.3` is a real inconsistency.
The repo's documented Go version (`openspec/project.md` line 19,
`openspec/AGENTS.md` line 9) is `1.26.3`, which matches two of three
`go.mod` files. The `go.work` `1.26.5` is the floor the toolchain
installed on the host (`go version go1.26.5 darwin/arm64`) and is what
`database_administrator/go.mod` is on.

**For the harness, this matters because:** `govulncheck@v1.1.4` and
`golangci-lint@v2.9.0` are both pinned (Makefile lines 28, 30). The
harness must not silently upgrade either.

### Vendor / install state

- `backend/*/bin/` directories exist but are empty (glob returned 0 files;
  `.gitignore` line 28 excludes `**/bin/`).
- First run of any backend Makefile target triggers `make tools` and/or
  `make vuln-check`, each of which downloads from `raw.githubusercontent.com`
  and `go install`. **Network egress is required on a cold cache.** This
  is a real constraint for offline runs and for CI cache strategy; the
  harness should not pretend otherwise.

### Existing precedent for `scripts/` choice

| Script                                                  | Shebang / runtime                              | Style                                |
|---------------------------------------------------------|-----------------------------------------------|--------------------------------------|
| `scripts/validate-infra.py` (117 lines)                 | `#!/usr/bin/env -S uv run --quiet --no-project --with pyyaml python3` (line 1) | One file, one exit code, structured `OK`/`BAD`/`ERR`/`WARN` per-line prints |
| `scripts/human-run-tail-sampling-verify.sh` (237 lines) | `#!/usr/bin/env bash` + `set -euo pipefail` (line 1, 24) | Pretty-colored `pass`/`fail`/`info`/`warn`/`heading` helpers, per-scenario functions |

Two valid precedents. Either is defensible; the choice should be
justified on its own merits (see Approaches below).

---

## Affected Areas

- `scripts/validate.{sh,py,go}` — **new file**; the validation harness
  itself. Single entry point for the repo.
- `openspec/changes/repo-validation-harness/` — new change folder
  (proposal/spec/design/tasks that follow this exploration).
- `backend/agent/Makefile` — **not modified by this change**, but
  surfaced in the report as a `tool-missing` finding (see Recommendation).
  Adding `vuln-check` to it is **explicitly out of scope** and should be a
  separate change with its own ADR if pursued.
- `backend/database_administrator/Makefile` (lines 137–154) — `test/integration`
  boots Postgres. The harness must skip this unless the operator opts in
  (see Risks).
- `frontend/playwright.config.ts` (lines 47–56) — `test:e2e` is opt-in
  for the same reason.
- `frontend/eslint.config.js` (line 47, 82) — `eslint-rules/**` is ignored
  by ESLint but tested separately via `node --test`. The harness should
  run that test on its own (and surface it as its own finding category).
- `docs/adr/0005-promote-agent-stack-to-own-module.md` — referenced by
  the change as the policy owner of `backend/agent`'s Makefile drift.
  Not modified.
- `.github/workflows/` — **not created by this change.** CI wiring is
  out of scope; this is a local-run harness. Mentioned only so the
  orchestrator doesn't expect it.
- `docker-compose.yaml`, `infra/`, `backend/*/src/` — **untouched**, per
  `openspec/AGENTS.md` hard rules and per `openspec/AGENTS.md` line 86.

---

## Approaches

### A. Python script (`scripts/validate.py`) with `uv` shebang

Mirrors `scripts/validate-infra.py` exactly. Same runtime, same deps
mechanism (uv inline `--with`).

- **Pros**
  - Zero-install on host with `uv` (already on PATH).
  - First-class JSON for the report (the downstream AI consumer needs
    stable, machine-parseable output).
  - Single-file Python is the easiest place to define a `Finding`
    dataclass and produce both a JSON and a Markdown render from the same
    in-memory representation.
  - Mirrors an existing repo convention (validate-infra.py).
  - Easy to test (`pytest`, also via uv); no compile step.
- **Cons**
  - Adds Python as a soft runtime requirement. Mitigated by the `uv`
    shebang — the script fails fast with a clear message if `uv` is
    missing.
  - One more runtime in a Go-heavy repo. Counter: `validate-infra.py`
    already established Python as acceptable for harness-class scripts.
- **Effort**: Medium (1–2 days; main work is the report schema + the
  per-tool JSON parsers).

### B. Bash script (`scripts/validate.sh`)

Mirrors `scripts/human-run-tail-sampling-verify.sh` in shape and color
helpers.

- **Pros**
  - Zero dependencies beyond `bash`, `make`, `pnpm`, `go`, `jq`, `curl`.
  - Same runtime the repo already uses for `human-run-*.sh`.
- **Cons**
  - JSON-in-bash for tool output is fragile. `jq` is available but the
    data-shaping across `golangci-lint`, `eslint`, `prettier`, `tsc`,
    `vitest`, `pnpm audit`, `govulncheck` produces N different parse
    grammars — bash with heredocs and `jq` filters will be ugly.
  - Stable finding IDs (sha256 of normalized content, for dedup) are
    non-trivial in bash.
  - Bash tests are rare and slow.
- **Effort**: Medium–High (more lines for less capability; risk of
  parser rot when tool versions change).

### C. Go script (`scripts/validate.go`, run via `go run`)

Fits the project's language identity. Could be a `main` package in a
small standalone module.

- **Pros**
  - Fastest execution.
  - Same language as the backend; can re-use `internal/` packages if
    they grow.
  - `encoding/json` is rock-solid for the report.
- **Cons**
  - Requires a `go.mod` somewhere — either a new tiny module under
  `scripts/` (which means `go.work` has to learn about it; ADR 0005
  governs the workspace) OR run as a single file with `go run` (no
  `go.mod`, no deps allowed).
  - A new top-level Go file under `scripts/` triggers ADR-0005 / openspec
  review: "new top-level dependency ⇒ ADR first" (`openspec/AGENTS.md`
  line 24). Even without new deps, introducing a new code root costs
  process budget.
  - Hardest to test of the three.
- **Effort**: High.

### D. Root `Makefile`

A `Makefile` at the repo root that calls each module's targets.

- **Pros**
  - Trivial parallelism with `make -j`.
  - Re-uses each module's Makefile targets verbatim — DRY.
- **Cons**
  - Make is a poor report generator. The structured JSON/Markdown
  report would need a separate writer (a separate language anyway).
  - `-j` masks ordering bugs and parallelises target install steps that
    step on each other (`go install` writes to `$GOPATH/bin` from multiple
    goroutines).
  - Output capture from parallel jobs in Make is awkward (one log per
  job, no clean JSONL stream).
- **Effort**: High, and the script would still need a Python or Go
post-processor.

### Recommendation

**Approach A — Python script `scripts/validate.py` with a `uv` shebang,
two output files (JSON + Markdown) from the same in-memory finding list.**

Reasons:

1. **The downstream consumer is an AI agent.** That decides the format
   question: structured JSON is non-negotiable. Approach A makes that
   cheap; B and D require a separate JSON writer anyway.
2. **Python is already in the repo** for harness-class work
   (`validate-infra.py`); the `uv` shebang precedent is set.
3. **`jq` + bash (B) is feasible but ugly.** The seven tools this
   harness wraps produce seven different output shapes; Python's
   `json.loads` + dataclasses keeps each parser small.
4. **Go (C) introduces a new top-level module root** and triggers
   process overhead without buying anything the Python version lacks.
5. **Makefile (D)** is the right tool for the *invocation* layer, not
   the *report* layer. Approach A can call `make` per-module under the
   hood and still get a clean report.

### Report design (the first-class constraint)

The report must be **a stable, dedupable, machine-parseable list of
findings**. Concrete shape:

```jsonc
{
  "schema_version": "1",
  "change": "repo-validation-harness",
  "generated_at": "2026-08-10T13:25:00Z",
  "host": {"go": "go1.26.5", "node": "v20.x", "pnpm": "11.8.0"},
  "summary": {"total": 14, "by_severity": {"high": 3, "medium": 7, "low": 4, "info": 0}},
  "findings": [
    {
      "id": "sha256:...",          // stable over content (path+line+rule+message)
      "category": "lint|typecheck|format|test|vuln|tool-missing|env-missing",
      "tool": "golangci-lint|eslint|prettier|tsc|vitest|playwright|govulncheck|pnpm-audit|make",
      "scope": "backend/database_administrator|backend/workspace_syncer|backend/agent|frontend|repo-root",
      "severity": "critical|high|medium|low|info",
      "rule": "staticcheck|govet|cachicamas/no-inline-button-class|GHSA-...|null",
      "file": "src/interfaces/http/health_handler.go",   // repo-relative
      "line": 42, "column": 5,                            // null if not applicable
      "message": "S1025: the argument is a string literal",
      "evidence": "<raw tool output line(s)>",
      "reproduce": "cd backend/database_administrator && bin/golangci-lint run --config=.golangci.yml ./...",
      "fix_hint": "Replace the literal with a constant or named var."
    }
  ]
}
```

The same in-memory list renders to:

- **`validation-report.json`** — canonical, the AI agent reads this.
- **`validation-report.md`** — for `cat` review by humans; one
  section per scope, findings grouped by severity, each finding has a
  copy-pasteable `reproduce` line.

**Run semantics:** always **run-all-then-aggregate**, never fail-fast.
The whole point of a fix-it report is the AI agent sees the full
picture, not the first error. Per-check timeout default 600s; e2e
1200s (separate flag). Exit code: 0 on no findings, 1 on any finding —
so CI can still gate, but the report is the authoritative output.

**Tool orchestration matrix** (each row is one subprocess):

| Scope                              | Check           | Command                                                                            | JSON output flag                   |
|------------------------------------|-----------------|------------------------------------------------------------------------------------|------------------------------------|
| `backend/database_administrator`   | test            | `make test`                                                                        | (parse stdout)                     |
| `backend/database_administrator`   | test/cover      | `make test/cover`                                                                  | (parse stdout)                     |
| `backend/database_administrator`   | lint            | `make lint`                                                                        | `--out-format=json` on golangci-lint |
| `backend/database_administrator`   | vuln-check      | `make vuln-check`                                                                  | `-json` on govulncheck             |
| `backend/database_administrator`   | test/integration| opt-in (`--include-integration`); `make test/integration`                          | (parse stdout)                     |
| `backend/workspace_syncer`         | test            | `make test`                                                                        | (parse stdout)                     |
| `backend/workspace_syncer`         | test/cover      | `make test/cover`                                                                  | (parse stdout)                     |
| `backend/workspace_syncer`         | lint            | `make lint`                                                                        | `--out-format=json`                |
| `backend/workspace_syncer`         | vuln-check      | `make vuln-check`                                                                  | `-json`                            |
| `backend/agent`                    | test            | `make test`                                                                        | (parse stdout)                     |
| `backend/agent`                    | test/cover      | `make test/cover`                                                                  | (parse stdout)                     |
| `backend/agent`                    | lint            | `make lint`                                                                        | `--out-format=json`                |
| `backend/agent`                    | **vuln-check**  | **`make vuln-check`** — surface as `tool-missing` finding; do NOT call govulncheck directly | n/a                                |
| `frontend`                         | lint            | `pnpm --dir frontend lint -- --format json`                                         | `--format json` (ESLint 9)         |
| `frontend`                         | typecheck       | `pnpm --dir frontend build.types`                                                  | (parse stdout; `--pretty false` if tsc supports) |
| `frontend`                         | format          | `pnpm --dir frontend fmt.check`                                                    | (parse stdout; prettier is line-per-file) |
| `frontend`                         | test            | `pnpm --dir frontend test:ci -- --reporter=json`                                   | vitest `--reporter=json`           |
| `frontend`                         | eslint-rule-test| `cd frontend && node --test eslint-rules/*.spec.mjs`                               | node test runner JSON              |
| `frontend`                         | vuln-check      | `pnpm --dir frontend run vuln-check` (`pnpm audit --json --audit-level=high`)       | `--json` on pnpm audit             |
| `frontend`                         | e2e             | opt-in (`--include-e2e`); requires stack-up per playwright.config.ts               | (parse stdout)                     |
| repo-root                          | go.work sanity  | `go work edit -json` parses; warn if any module's Go directive drifts               | `-json`                             |

**`backend/agent` vuln-check gap resolution:** **option (b) — surface as
a `tool-missing` finding, do not bypass Make.** Reasons:

- The Makefile's missing target is a *policy* gap owned by ADR 0005,
  not a script-level workaround. Adding it behind the harness's back
  would hide the inconsistency from review.
- Option (a) (call `govulncheck` directly) duplicates the install logic
  the other two Makefiles own, and creates two places to keep versions
  in sync.
- Option (c) (add `make vuln-check` to `backend/agent/Makefile`) is a
  real fix and should be a separate sdd-apply with its own ADR if
  pursued — out of scope for this harness change.

The harness's finding for this case:

```json
{
  "category": "tool-missing",
  "tool": "make",
  "scope": "backend/agent",
  "severity": "medium",
  "message": "No `make vuln-check` target in backend/agent/Makefile; govulncheck was not run for this module.",
  "reproduce": "cd backend/agent && make -n vuln-check",
  "fix_hint": "Add a `vuln-check` target mirroring backend/database_administrator/Makefile:78-87 (govulncheck v1.1.4)."
}
```

**Tool-missing vs check-failed:** distinct categories. `tool-missing`
means we couldn't even invoke the tool (binary not installed, target
not defined, network unreachable). `check-failed` means we ran it and
it returned non-zero findings. The severity for `tool-missing` should be
`medium` (we don't know if there are real findings) and `high` for
`check-failed` (we know there are). The AI agent downstream uses these
to triage.

**`make install` for `backend/agent`:** the agent module has zero
requires and no `install` target by design (`Makefile` lines 55–58).
The harness must not report this as `tool-missing` — it's a documented
deliberate absence, not an oversight. **Add an allowlist** in the
harness config for this and any future documented absences.

**Frontend `eslint-rules/`:** these are tested with `node --test`, not
vitest, and are excluded from vitest config (`vite.config.ts` line 41–43).
The harness must run them as a separate check under the `frontend`
scope, not as part of `pnpm test:ci`.

**Parallelism:** v1 runs sequentially. Reasons: predictable exit codes,
cleaner logs, simpler test. Parallelism is a v2 concern (`asyncio` or
`concurrent.futures` in Python; trivially added without changing the
report schema).

---

## Risks

1. **First-run network dependency.** All four backend Makefiles auto-install
   `golangci-lint v2.9.0`, `goimports`, and `govulncheck v1.1.4` on first
   invocation. The harness must declare this in its `--help` and in the
   report header (`host.versions_installed`) so the AI agent can
   distinguish "tool missing" from "network blocked". Severity if it
   fails: `tool-missing` / `high`.

2. **Cold-cache run time.** A clean run touches `go install` for two
   tools × three modules + a full `go test -race ./...` × three modules
   + `pnpm install` (if frontend deps drift) + Playwright browser fetch
   if e2e is on. Realistic floor: 5–10 min, ceiling: 25+ min if
   Playwright is freshly installed. The harness should print elapsed
   time per check and a total; the AI agent needs those to budget
   retries.

3. **Two Go versions in the repo.** `go.work` says `1.26.5`; two modules
   say `1.26.3`. The harness must not pretend these are the same. If
   `go.work` is bumped and the modules lag, `make test` in two modules
   will fail with a toolchain error. The harness should emit a
   `go.work-mismatch` finding on first run if it observes this.

4. **`backend/agent` Makefile drift.** Already partially diverged (no
   `install`, `run`, `vuln-check`, `vuln-check/ci`, `test/integration`).
   The harness must not assume target presence. The current divergence
   is documented; future drift is the real risk. Mitigation: the harness
   enumerates targets at runtime via `make -n <target>` (dry-run) and
   reports `tool-missing` for any expected-but-absent target.

5. **E2E tests requiring a live stack.** `frontend/playwright.config.ts`
   lines 47–56 + 14–22 make this explicit: without `E2E_BASE_URL`, the
   script must have `docker compose up` (Postgres + database_administrator)
   already running, and Playwright then auto-starts `pnpm dev` on 5173.
   Mitigation: e2e is **opt-in** via `--include-e2e` and the harness
   reports a clear "skipped: e2e requires stack" note when run without
   it.

6. **Test flakes under `-race`.** All backend Makefiles run with
   `-race -v`; a race-only failure is a real finding but expensive to
   reproduce. The harness should preserve the full failure output in
   `evidence` and include the package path, test name, and any `-race`
   data-race report header in `message`.

7. **Custom ESLint rule emits warnings, not errors.** The
   `cachicamas/no-inline-button-class` rule is severity `warn`
   (`eslint.config.js` line 82). The harness must preserve severity
   from the tool, not flatten everything to `error`. The AI agent
   downstream decides whether to act on `warn`.

8. **Report drift between JSON and Markdown.** Two renders of the same
   in-memory list — easy to drift if a future change adds a finding
   field. Mitigation: a unit test that runs both renders and asserts
   every field appears in both.

9. **ADR-0005 process trigger.** Adding a new code root under
   `scripts/` is not a top-level dependency, so it does NOT trigger the
   ADR rule (`openspec/AGENTS.md` line 24). But it IS a new top-level
   file in a previously-script-only directory; the proposal should call
   this out so reviewers aren't surprised.

10. **`go.work.sum` changes.** A new dependency in any module forces
    `go.work.sum` updates; the harness does not need to commit them,
    but a future CI wiring will. Note in the design that this change
    is CI-incompatible until a follow-up adds `.github/workflows/`.

---

## Ready for Proposal

**Yes — proceed to `sdd-propose`.** Summary of what the orchestrator
should tell the user:

1. The orchestrator's preliminary audit was **mostly correct** with
   three corrections to note:
   - **Go version is split**: `go.work` says `1.26.5`; `database_administrator`
     is `1.26.5`; `workspace_syncer` and `agent` are still `1.26.3`.
     `go version` on the host confirms `1.26.5` is installed.
   - **`backend/agent` Makefile is missing more than `vuln-check`**: it
     is also missing `install`, `run`, `vuln-check/ci`, and
     `test/integration`. `test/cover` IS present (orchestrator's open
     question is answered).
   - **`.golangci.yml` configs are not byte-identical**: linter set is
     the same, but `workspace_syncer` swaps the `output:` block for an
     `issues:` block. `agent` matches `database_administrator`.

2. The harness is recommended as **Python (`scripts/validate.py`) with a
   `uv` shebang**, mirroring `scripts/validate-infra.py`. Rationale: the
   downstream consumer (an AI agent) needs structured JSON, and Python
   is already an accepted harness runtime in this repo.

3. The harness emits **two renderings of one in-memory finding list**:
   `validation-report.json` (canonical, for the AI agent) and
   `validation-report.md` (human review). The JSON shape is sketched
   above and should be refined in the proposal.

4. **`backend/agent` vuln-check is reported as a `tool-missing` finding
   (option b), not bypassed.** Adding the Makefile target is a separate,
   ADRed change.

5. **CI wiring (`.github/workflows/`) is out of scope.** This change
   produces a local-run harness. CI is a separate change.

6. **E2E and integration tests are opt-in** (`--include-e2e`,
   `--include-integration`). The default run covers everything that
   works against the local checkout alone.

7. The skill-vs-repo filename discrepancy is noted at the top of this
   artifact; the orchestrator can resolve it however it likes. This
   file is `explore.md` to match the dominant convention (39 of the 40
   archived explores in this repo).