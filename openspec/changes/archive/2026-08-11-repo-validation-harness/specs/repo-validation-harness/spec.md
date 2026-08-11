# Delta for `repo-validation-harness`

> **Change**: `repo-validation-harness` · **Status**: ADDED only (no MODIFIED / REMOVED / RENAMED) — `openspec/specs/repo-validation-harness/spec.md` does not yet exist
> **Sources**: [proposal.md](../proposal.md) · [explore.md](../explore.md) · [config.yaml](../../config.yaml)
> **Built on**: `openspec/project.md` (Go 1.26.3, pnpm 11.8.0, Node 18.17+/20.3+/21+, golangci-lint v2.9.0, govulncheck v1.1.4)
> **Conformance**: RFC 2119 + Given/When/Then; every scenario is fixture-based (no live vuln DB, no compose stack)

---

## ADDED Requirements

### R-RVH-001 — Run-all-then-aggregate

> **Purpose.** The downstream consumer is an AI agent that needs the full picture, not the first error. Fail-fast hides every finding after the first one.

The harness MUST execute every enabled check sequentially, MUST continue past a failed check, MUST continue past a `tool-missing` check, and MUST aggregate findings from all checks before deciding exit.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a fixture config with check A (emits a finding) and check B (emits a finding) | the harness runs | the report contains both findings (no silent drop) |
| **S-2** | a fixture config with check A (target absent → `tool-missing`) and check B (emits a finding) | the harness runs | both findings appear (A's `tool-missing` AND B's) |

### R-RVH-002 — Exit contract

> **Purpose.** Consumers must distinguish "checks found problems" from "the harness itself broke". Three distinct exit codes is the minimum for that.

The harness MUST exit 0 when no finding has severity >= the configured failure threshold (default `high`). It MUST exit 1 when any finding reaches the threshold. It MUST exit a distinct non-0/1 code (e.g. 2) for harness-internal errors (malformed config, unwritable output path, invalid CLI flag). The failure code MUST NOT be used for findings.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a fixture config whose checks report zero findings | the harness runs | the process exit code is 0 |
| **S-2** | a fixture config with one check that emits a `high` finding | the harness runs | the exit code is 1 |
| **S-3** | a malformed config file at the configured path | the harness runs | the exit code is 2 (or another distinct non-0/1 value), AND the failure is NOT reported as a `high` finding |

### R-RVH-003 — Missing target is a `tool-missing` finding, not a crash

> **Purpose.** `backend/agent/Makefile` lacks `vuln-check`, `install`, `run`, and `vuln-check/ci`; `backend/workspace_syncer/Makefile` lacks `test/integration`. `agent` does deliberately omit `install` (commented in `backend/agent/Makefile` lines 55–58: "There is no `install` target on purpose. This module carries ZERO requires and has nothing to download. It stays that way until a milestone arrives with an ADR to add a dependency"). The other absences are gaps, not design. The harness MUST surface every absence as a finding — visible, not silently bypassed by calling the underlying tool directly. The allowlist in R-RVH-012 silences only the documented `install` absence; the remaining gaps are exactly the findings the report is meant to expose.

When a check's target is absent (no Makefile target, no binary on PATH, no directory), the harness MUST record a finding with `category="tool-missing"` and `severity="medium"`, MUST continue to the next check, and MUST NOT invoke the underlying tool directly to bypass the gap.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a fixture config mapping `backend/agent` to a `vuln-check` target that does not exist | the harness runs | the report contains a `tool-missing` finding AND subsequent checks still execute |
| **S-2** | the same config; the underlying tool (`govulncheck`) is a shim that records invocations | the harness runs | the shim's invocation log is empty (no direct call from the harness), AND the finding's `reproduce` is the exact `make -n vuln-check` command |

### R-RVH-004 — Stable finding IDs from sha256 of normalized content

> **Purpose.** The AI consumer dedupes and tracks findings across runs. IDs that change between runs defeat that.

A finding's `id` MUST be `sha256({repo-relative-file}|{line-or-empty}|{rule-or-empty}|{message})`, truncated consistently (e.g. first 16 hex chars). Two runs on the same input MUST produce the same ID. Absent fields MUST be normalized to the empty string before hashing.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a finding with `file="src/x.go"`, `line=10`, `rule="staticcheck"`, `message="S1025"` | the harness computes the ID twice on the same input | the two IDs are byte-identical |
| **S-2** | two findings whose messages differ only in a numeric counter | the ID is computed | the two IDs are different |
| **S-3** | two findings where one has `line=null` and the other has `line=0` (after empty-string normalization) | the ID is computed | the IDs are identical when normalization applies; absent-vs-zero is collapsed to the same normalized input |

### R-RVH-005 — Severity preserved verbatim from the underlying tool

> **Purpose.** The custom ESLint rule `cachicamas/no-inline-button-class` is `warn`. Promoting `warn` to `error` would force the AI agent to fix things the rule author marked low-priority.

The harness MUST preserve the severity value emitted by the underlying tool. Severity MUST NOT be promoted (e.g. `warn` → `error`) by the harness. The custom ESLint rule `cachicamas/no-inline-button-class` (severity `warn` in `frontend/eslint.config.js` line 82) MUST stay `warn` in the report.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a fake ESLint rule emitting `severity="warn"` on a fixture file | the harness parses the output | the finding's `severity` is `"warn"` (not `"error"`) |
| **S-2** | a fixture Vue file that triggers `cachicamas/no-inline-button-class` | the harness runs ESLint on it | the finding's `severity` is `"warn"` |

### R-RVH-006 — Opt-in checks for e2e and integration

> **Purpose.** `make test/integration` boots Postgres; Playwright e2e needs the compose stack. Default runs MUST NOT require docker.

The harness MUST skip `make test/integration` and `pnpm test:e2e` by default. These checks MUST run only when the operator passes explicit flags (`--include-integration`, `--include-e2e`). When skipped, the run log MUST record a "skipped" note specifying which check was skipped.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a fixture config that lists an `integration` check | the harness runs without `--include-integration` | the integration check is not in the executed-check list, AND the log line `"skipped: integration"` appears |
| **S-2** | a fixture config that lists an `e2e` check | the harness runs without `--include-e2e` | the e2e check is not in the executed-check list, AND the log line `"skipped: e2e"` appears |
| **S-3** | the same fixture config | the harness runs with `--include-integration` | the integration check IS in the executed-check list (the flag is honored) |

### R-RVH-007 — `go.work-mismatch` is a first-class finding category

> **Purpose.** The repo currently has `go.work` at `1.26.5` while `backend/workspace_syncer/go.mod` and `backend/agent/go.mod` declare `1.26.3`. This drift breaks CI silently.

When `go.work`'s Go version differs from any module's `go.mod` Go directive, the harness MUST emit a finding with `category="go.work-mismatch"` and `severity="medium"`. The scope MUST be the affected module.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a fixture workspace where `go.work` declares `go 1.26.5` and `module/go.mod` declares `go 1.26.3` | the harness runs | the report contains a `go.work-mismatch` finding scoped to the affected module |
| **S-2** | a fixture workspace where `go.work` and all `go.mod` Go directives match | the harness runs | the report contains no `go.work-mismatch` finding |

### R-RVH-008 — `make build` is deliberately NOT a check

> **Purpose.** `go test ./...` and `go vet ./...` already compile every package including `cmd/`; `frontend` `build.types` covers the frontend. `build` adds binary-writing side effects without new signal.

The harness MUST NOT execute `make build` for any module. The check matrix MUST NOT include any entry whose target is `build`. The harness MUST NOT write a binary to `./bin/` as a side effect of any check.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | the harness's check matrix config | the executed-check list is enumerated | no entry has a target named `build` |
| **S-2** | a clean fixture workspace with `./bin/` empty | the harness runs end-to-end | `./bin/` remains empty, AND no check's stdout mentions a build success |

### R-RVH-009 — JSON and Markdown rendered from one in-memory list

> **Purpose.** Two renders of the same data must never disagree. Drift would force the AI consumer to read JSON and ignore Markdown or vice versa.

The harness MUST produce `validation-report.json` and `validation-report.md` from the same in-memory finding list. Any field present in the in-memory list MUST appear in both renders. The harness MUST NOT derive one output from the other (no post-processing).

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a run that emits three findings | both report files are inspected | the three `id` values from `validation-report.json` each appear in `validation-report.md` |
| **S-2** | a run that emits one finding | a new finding is added to the in-memory list and both files are re-rendered | both files contain the new finding (no drift between renders) |

### R-RVH-010 — Finding payload contract

> **Purpose.** The AI consumer must parse findings without guessing. Absent fields are signals, not omissions.

Every finding MUST carry, in this order: `id`, `severity`, `category`, `tool`, `scope`, `rule`, `file`, `line`, `column`, `message`, `evidence`, `reproduce`, `fix_hint`. Fields whose source did not provide a value MUST be present as `null` in JSON, NOT omitted keys. `file` MUST be repo-relative (no leading `/`, no `..`). `reproduce` MUST be a single shell command invokable from the repo root.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | a report loaded from `validation-report.json` | a schema check inspects each finding | all 13 fields are present, AND `null` values are kept (not omitted) |
| **S-2** | an ESLint finding whose source did not provide a column | the report is loaded | the finding's `column` is `null` (not `0`, not omitted) |
| **S-3** | a finding whose source provided an absolute path | the report is loaded | the `file` value is repo-relative (no leading `/`, no `..`) |

### R-RVH-011 — Output paths and deterministic CI-ready output

> **Purpose.** The harness is CI-ready even though CI wiring is out of scope for this change.

The harness MUST write `validation-report.json` and `validation-report.md` to a configurable output directory (default `.`). The JSON MUST parse cleanly under `python3 -m json.tool`. The exit code MUST be deterministic given the same input. The harness MUST NOT require a TTY.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | the harness invoked with `--output-dir=/tmp/foo` | the run completes | `/tmp/foo/validation-report.json` and `/tmp/foo/validation-report.md` both exist, AND `python3 -m json.tool < /tmp/foo/validation-report.json` exits 0 |
| **S-2** | a fixture config and a fixture workspace | the harness runs twice with the same flags | both exit codes are identical |
| **S-3** | the harness invoked with stdin/stdout redirected (no TTY) | the run completes | the harness exits normally with no error about TTY |

### R-RVH-012 — Allowlist for documented absences

> **Purpose.** `tool-missing` must distinguish "the harness found a gap" from "an ADR documented the gap on purpose". `backend/agent/Makefile` lines 55-58 explicitly state no `install` target by design.

The harness MUST support a config allowlist of `(scope, target)` pairs that are documented absences. Allowlisted pairs MUST NOT produce a `tool-missing` finding. The default config MUST allowlist `("backend/agent", "install")` per the `backend/agent/Makefile` comment block.

| Scenario | GIVEN | WHEN | THEN |
|---|---|---|---|
| **S-1** | the harness with the default config | the harness runs against a workspace whose `backend/agent/Makefile` has no `install` target | no `tool-missing` finding appears for `(backend/agent, install)` |
| **S-2** | a config that does NOT allowlist `(backend/agent, install)` | the harness runs against the same workspace | a `tool-missing` finding appears for `(backend/agent, install)` |

---

## Out of scope

- **CI wiring** (`openspec/changes/.../add-github-workflows/`) — separate change.
- **Adding `vuln-check` to `backend/agent/Makefile`** — separate ADRed change (the gap is surfaced as a finding, not silently bridged).
- **Adding `test/integration` to `backend/workspace_syncer/Makefile`** — separate change.
- **`make build` as a check** — explicitly rejected per R-RVH-008.

## Acceptance criteria

1. All 12 requirements hold; each verified by its scenarios.
2. Every scenario is fixture-based; none requires the compose stack or live vulnerability-database contents.
3. The `tool-missing` finding for `backend/agent` `vuln-check` is produced on the real workspace.
4. The `go.work-mismatch` finding is produced on the real workspace (current drift between `go.work` 1.26.5 and `agent`/`workspace_syncer` 1.26.3).
5. `validation-report.json` parses cleanly; `validation-report.md` carries the same `id` values.
6. Default run skips e2e and integration; `--include-*` flags include them.
7. No `backend/*/src/`, `docker-compose.yaml`, `infra/`, or Makefile is modified.
