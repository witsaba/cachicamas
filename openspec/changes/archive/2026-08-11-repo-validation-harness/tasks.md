# Tasks: Repo-level validation harness

## Review Workload Forecast

Estimated: ~900 lines (validate.py ~600 + tests ~300 + config + project.md). Single PR with `size:exception`. Delivery: exception-ok. Chain: size-exception.

Decision needed before apply: Yes
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High

### Decision: single PR with `size:exception`

Purely additive (`git rm scripts/*`); no Makefile/src/compose touched. PR1 can't run end-to-end alone, leaving reviews incomplete. TDD preserved (`openspec/AGENTS.md` line 47). ~30 min one PR vs ~45 min two.

## Orchestrator Constraints

1. Strict TDD: RED before GREEN; each RED names the failing test class+method in `scripts/test_validate.py`.
2. Skill: `test-driven-development` (NOT `go-testing`; harness is Python). Stdlib + `pyyaml` via `uv run --with`; ADR rule not triggered.
3. Conventional commits only.

## Phase 0 — Skeleton & fixtures

- [x] **0.1** Create `scripts/testdata/{fixtures,mock-tools}/` + `scripts/validate-config.yaml` (allowlist `(backend/agent, install)`, `fail_on: high`, `timeout_check: 600`). SKELETON.
- [x] **0.2** Add `uv 0.11.17` to `openspec/project.md` Tooling Versions (open question §1). DOC.
- [x] **0.3** Shim `scripts/testdata/mock-tools/record_invocations.py` for R-RVH-003 S-2. SKELETON.

## Phase 1 — RED: pure functions

- [x] **1.1** RED `TestFindingPayload` (R-RVH-010: 13 fields, null preserved, file repo-relative). First fail: `test_all_13_fields_present`.
- [x] **1.2** RED `TestFindingId` (R-RVH-004: deterministic, differ-on-message, normalize-null-line). First fail: `test_deterministic_for_same_input`.
- [x] **1.3** RED `TestSeverity` (R-RVH-005: `warn` stays `warn`, cachicamas preserved). First fail: `test_eslint_warn_stays_warn_in_severity_field`.
- [x] **1.4** RED `TestExit` (R-RVH-002: exit 0/1/2). First fail: `test_zero_when_no_findings_above_threshold`.
- [x] **1.5** RED `TestParsers` (10 parsers). First fail: `test_golangci`.

## Phase 2 — RED: aggregator, renderers, opt-in, allowlist, go.work, build

- [x] **2.1** RED `TestAggregator` + `TestToolMissing` (R-RVH-001/003). First fail: `test_continues_past_failed_check`.
- [x] **2.2** RED `TestOptIn` + `TestAllowlist` (R-RVH-006/012). First fail: `test_integration_skipped_without_flag`.
- [x] **2.3** RED `TestRenderers` (R-RVH-009: parity JSON↔MD). First fail: `test_same_ids_in_both_outputs`.
- [x] **2.4** RED `TestGoWorkMismatch` + `TestNoBuildCheck` (R-RVH-007/008). First fail: `test_emits_finding_on_drift`.

## Phase 3 — GREEN: CLI, subprocess runner, main

- [x] **3.1** GREEN pure functions: `Finding`, `compute_finding_id`, `severity_ladder`, `compute_exit`. Passes Phase 1 RED.
- [x] **3.2** GREEN parsers (10). Passes `TestParsers`.
- [x] **3.3** GREEN aggregator + tool-missing + opt-in + allowlist + renderers + go.work-mismatch + build meta-check. Passes Phase 2 RED.
- [x] **3.4** GREEN CLI (8 flags per design.md); subprocess runner (`timeout=N`, `TimeoutExpired → timeout/medium`); preflight (binary, `make -n <target>`); `main()`; output-dir. Covers R-RVH-011.

## Phase 4 — Live integration

- [x] **4.1** Live run `scripts/validate.py` (no `--include-*`). Assert: `backend/agent vuln-check` → tool-missing (acceptance #3); `go.work-mismatch` (go.work 1.26.5 vs go.mod 1.26.3) (acceptance #4); `./bin/` unchanged (R-RVH-008 S-2). VERIFY.

## Phase 5 — Verify

- [x] **5.1** `python3 -m unittest test_validate` — all 44 tests pass (29 spec + 10 parsers + 5 bug-fix). VERIFY.
- [x] **5.2** Checklist: no Makefile/src/compose/infra modified; `uv` row in `project.md`; report JSON parses; conventional commits only. VERIFY.

## Coverage Matrix (29 scenarios → 29 tests)

- R-RVH-001 S-1,S-2: TestAggregator.{test_continues_past_failed_check, test_includes_tool_missing_with_real_finding}
- R-RVH-002 S-1..S-3: TestExit.{test_zero_when_no_findings_above_threshold, test_one_when_finding_above_threshold, test_two_on_harness_internal_error}
- R-RVH-003 S-1,S-2: TestToolMissing.{test_absent_target_emits_finding, test_does_not_invoke_underlying_tool}
- R-RVH-004 S-1..S-3: TestFindingId.{test_deterministic_for_same_input, test_differs_when_message_differs, test_normalizes_null_line}
- R-RVH-005 S-1,S-2: TestSeverity.{test_eslint_warn_stays_warn_in_severity_field, test_cachicamas_no_inline_button_class_severity_preserved}
- R-RVH-006 S-1..S-3: TestOptIn.{test_integration_skipped_without_flag, test_e2e_skipped_without_flag, test_integration_included_with_flag}
- R-RVH-007 S-1,S-2: TestGoWorkMismatch.{test_emits_finding_on_drift, test_no_finding_on_match}
- R-RVH-008 S-1,S-2: TestNoBuildCheck.{test_check_matrix_has_no_build_target, test_bin_dir_unchanged_after_dry_run}
- R-RVH-009 S-1,S-2: TestRenderers.{test_same_ids_in_both_outputs, test_adding_finding_updates_both}
- R-RVH-010 S-1..S-3: TestFindingPayload.{test_all_13_fields_present, test_null_preserved_not_omitted, test_file_is_repo_relative}
- R-RVH-011 S-1..S-3: TestCIReady.{test_output_dir_creates_files, test_deterministic_exit_across_runs, test_runs_without_tty}
- R-RVH-012 S-1,S-2: TestAllowlist.{test_default_allowlist_silences_documented_absence, test_removing_allowlist_surfaces_finding}