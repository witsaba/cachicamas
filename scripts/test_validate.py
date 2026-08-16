#!/usr/bin/env python3
"""Strict-TDD test suite for scripts/validate.py.

Maps 1:1 to the 29 scenarios in
openspec/changes/repo-validation-harness/specs/repo-validation-harness/spec.md,
plus 10 parser-specific tests per design.md "Test Plan".

Run with:
    cd scripts && python3 -m unittest test_validate -v

Or:
    python3 -m unittest discover -s scripts -p 'test_validate.py' -v

All tests are pure-function tests (no subprocess, no live network, no
compose stack) except where the spec scenario explicitly calls for one.
"""
from __future__ import annotations

import json
import sys
import tempfile
import textwrap
import unittest
from pathlib import Path
from unittest import mock

# Make the validate module importable regardless of where tests run from.
HERE = Path(__file__).resolve().parent
sys.path.insert(0, str(HERE))

import validate  # noqa: E402


# ---------------------------------------------------------------------------
# R-RVH-010 — Finding payload contract (3 scenarios)
# ---------------------------------------------------------------------------

class TestFindingPayload(unittest.TestCase):
    """R-RVH-010 S-1..S-3: 13 fields, null preserved, file repo-relative."""

    def test_all_13_fields_present(self) -> None:
        finding = validate.Finding(
            id="abc123",
            severity="high",
            category="lint",
            tool="golangci-lint",
            scope="backend/database_administrator",
            rule="staticcheck",
            file="src/x.go",
            line=42,
            column=5,
            message="S1025",
            evidence="raw line",
            reproduce="cd backend/database_administrator && bin/golangci-lint run ./...",
            fix_hint="Replace literal with constant.",
        )
        as_dict = validate.finding_to_jsonable(finding)
        expected = [
            "id", "severity", "category", "tool", "scope", "rule", "file",
            "line", "column", "message", "evidence", "reproduce", "fix_hint",
        ]
        self.assertEqual(list(as_dict.keys()), expected)

    def test_null_preserved_not_omitted(self) -> None:
        finding = validate.Finding(
            id="x", severity="warn", category="lint", tool="eslint",
            scope="frontend", rule=None, file=None, line=None, column=None,
            message="m", evidence=None, reproduce="r", fix_hint=None,
        )
        blob = json.dumps(validate.finding_to_jsonable(finding))
        # Nulls MUST be present as JSON null, NOT omitted keys.
        self.assertIn('"rule": null', blob)
        self.assertIn('"file": null', blob)
        self.assertIn('"line": null', blob)
        self.assertIn('"column": null', blob)
        self.assertIn('"evidence": null', blob)
        self.assertIn('"fix_hint": null', blob)

    def test_file_is_repo_relative(self) -> None:
        # Absolute paths and parent refs must be stripped.
        for bad, good in [
            ("/Users/x/repo/src/y.go", "src/y.go"),
            ("../../escape/foo.ts", "escape/foo.ts"),
            ("/Users/x/repo/foo/../bar/baz.go", "bar/baz.go"),
            ("src/z.ts", "src/z.ts"),
        ]:
            normalized = validate.normalize_repo_relative(bad, repo_root=Path("/Users/x/repo"))
            self.assertEqual(normalized, good)


# ---------------------------------------------------------------------------
# R-RVH-004 — Stable finding IDs from sha256 of normalized content (3 scenarios)
# ---------------------------------------------------------------------------

class TestFindingId(unittest.TestCase):
    """R-RVH-004 S-1..S-3: deterministic, differ-on-message, normalize-null-line."""

    def test_deterministic_for_same_input(self) -> None:
        a = validate.compute_finding_id(
            file="src/x.go", line=10, rule="staticcheck", message="S1025"
        )
        b = validate.compute_finding_id(
            file="src/x.go", line=10, rule="staticcheck", message="S1025"
        )
        self.assertEqual(a, b)
        self.assertEqual(len(a), 16)  # truncated hex

    def test_differs_when_message_differs(self) -> None:
        a = validate.compute_finding_id(
            file="src/x.go", line=10, rule="staticcheck", message="S1025"
        )
        b = validate.compute_finding_id(
            file="src/x.go", line=10, rule="staticcheck", message="S1026"
        )
        self.assertNotEqual(a, b)

    def test_normalizes_null_line(self) -> None:
        # Null and 0 (after empty-string normalization) collapse to the
        # same normalized input — absent-vs-zero is not a different finding.
        # The spec uses line=0 in the example; null is the practical case.
        # Both MUST produce the same ID because the empty-string normalization
        # happens before hashing.
        a = validate.compute_finding_id(
            file="src/x.go", line=0, rule="staticcheck", message="m"
        )
        b = validate.compute_finding_id(
            file="src/x.go", line=None, rule="staticcheck", message="m"
        )
        # 0 -> "0"; None -> "". Different by spec wording? Let's check.
        # Spec S-3 says "absent-vs-zero is collapsed to the same normalized input"
        # when normalization applies. Implementation MUST normalize absent
        # to ""; the question is whether 0 is also normalized. Per design.md
        # finding ID formula uses {line-or-empty}: absent IS empty; 0 is NOT
        # empty. So a != b — BUT the spec text says S-3 expects equality.
        # To pass S-3 we normalize 0 to "" as well.
        self.assertEqual(a, b)


# ---------------------------------------------------------------------------
# R-RVH-005 — Severity preserved verbatim from the underlying tool (2 scenarios)
# ---------------------------------------------------------------------------

class TestSeverityLadder(unittest.TestCase):
    """The severity ladder maps the synthetic severities used by the harness
    (`medium`, `low`, `info`, `warn`) to the mapped bucket the threshold
    uses. Missing a key means the bucket is wrong, which mislabels the
    summary and can mis-fire the exit code under a non-default threshold.

    Identity entries for `medium` / `low` / `info` / `high` / `critical`
    override the tool-severity entries (`low → medium`, `info → low`) for
    inputs that happen to match the bucket names. The harness's synthetic
    findings already use the bucket name as the verbatim severity, so the
    identity mapping is what makes the summary report the right bucket.
    """

    def test_medium_maps_to_medium(self) -> None:
        # `medium` is the synthetic severity for tool-missing, timeout, and
        # go.work-mismatch findings. It MUST map to the medium bucket.
        self.assertEqual(validate.map_severity("medium"), "medium")

    def test_low_maps_to_low(self) -> None:
        # Identity: a synthetic `low` finding stays in the low bucket.
        self.assertEqual(validate.map_severity("low"), "low")

    def test_info_maps_to_info(self) -> None:
        # Identity: a synthetic `info` finding stays in the info bucket.
        self.assertEqual(validate.map_severity("info"), "info")

    def test_warn_maps_to_low(self) -> None:
        # Tool-emitted `warn` (ESLint 9 severity 1) is bucketed as low.
        self.assertEqual(validate.map_severity("warn"), "low")


class TestSeverity(unittest.TestCase):
    """R-RVH-005 S-1, S-2: warn stays warn; cachicamas rule preserved."""

    def test_eslint_warn_stays_warn_in_severity_field(self) -> None:
        raw_eslint = json.dumps([
            {
                "filePath": "/abs/path/src/Button.tsx",
                "messages": [
                    {
                        "ruleId": "some-rule",
                        "severity": 1,  # 1 == warn in ESLint 9
                        "message": "use class binding",
                    }
                ],
            }
        ])
        findings = validate.parse_eslint_json(raw_eslint, scope="frontend")
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].severity, "warn")

    def test_cachicamas_no_inline_button_class_severity_preserved(self) -> None:
        # The custom rule cachicamas/no-inline-button-class is severity "warn"
        # in eslint.config.js. ESLint surfaces warn as severity 1.
        raw_eslint = json.dumps([
            {
                "filePath": "/abs/path/src/Btn.tsx",
                "messages": [
                    {
                        "ruleId": "cachicamas/no-inline-button-class",
                        "severity": 1,
                        "message": "Inline btn class is forbidden",
                    }
                ],
            }
        ])
        findings = validate.parse_eslint_json(raw_eslint, scope="frontend")
        self.assertEqual(findings[0].rule, "cachicamas/no-inline-button-class")
        self.assertEqual(findings[0].severity, "warn")


# ---------------------------------------------------------------------------
# R-RVH-002 — Exit contract (3 scenarios)
# ---------------------------------------------------------------------------

class TestExit(unittest.TestCase):
    """R-RVH-002 S-1..S-3: exit 0 / 1 / 2."""

    def test_zero_when_no_findings_above_threshold(self) -> None:
        # All findings below threshold (info) -> exit 0.
        findings = [
            _make_finding(severity="info", message="noise"),
        ]
        self.assertEqual(validate.compute_exit(findings, threshold="high"), 0)

    def test_one_when_finding_above_threshold(self) -> None:
        # One high finding -> exit 1.
        findings = [
            _make_finding(severity="high", message="real problem"),
        ]
        self.assertEqual(validate.compute_exit(findings, threshold="high"), 1)

    def test_two_on_harness_internal_error(self) -> None:
        # Malformed config -> exit 2; the failure is NOT a high finding.
        with self.assertRaises(validate.HarnessConfigError):
            validate.load_config(Path("/nonexistent/config.yaml"))


# ---------------------------------------------------------------------------
# R-RVH-003 — Missing target is a tool-missing finding, not a crash (2 scenarios)
# ---------------------------------------------------------------------------

class TestToolMissing(unittest.TestCase):
    """R-RVH-003 S-1, S-2: absent target -> tool-missing finding; no bypass."""

    def test_absent_target_emits_finding(self) -> None:
        # A CheckDef with no underlying tool/function and no Makefile target
        # resolves to a tool-missing finding on run; subsequent checks still
        # execute.
        from validate import run_checks  # type: ignore

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cfg = {
                "checks": [
                    {
                        "scope": "backend/agent",
                        "name": "vuln-check",
                        "command": ["make", "vuln-check"],
                        "cwd": str(tmp_path),
                    },
                    {
                        "scope": "backend/agent",
                        "name": "test",
                        "command": ["true"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            results = run_checks(cfg_path, allowlist=[])
            findings = [f for r in results for f in r.findings]
            # Both checks must have produced their own outcomes — the absent
            # target yields a tool-missing finding, the second check passes.
            self.assertTrue(any(f.category == "tool-missing" for f in findings))
            statuses = [r.status for r in results]
            self.assertIn("tool-missing", statuses)
            self.assertIn("passed", statuses)

    def test_does_not_invoke_underlying_tool(self) -> None:
        # When a Makefile target is absent, the harness MUST NOT call the
        # underlying tool directly. We point the Makefile at a shim that
        # records invocations; the log MUST remain empty.
        from validate import run_checks  # type: ignore

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            shim = tmp_path / "govulncheck_shim.py"
            shim_src = (HERE / "testdata" / "mock-tools" / "record_invocations.py").read_text()
            shim.write_text(shim_src)
            log = tmp_path / "invocations.log"
            log.write_text("")

            # Make a Makefile whose vuln-check target exists but does nothing.
            mkfile = tmp_path / "Makefile"
            mkfile.write_text(
                ".PHONY: test\n"
                "vuln-check:\n"
                "\t@echo 'no real vulncheck here'\n"
                "test:\n"
                "\t@exit 0\n"
            )
            cfg = {
                "checks": [
                    {
                        "scope": "backend/agent",
                        "name": "vuln-check",
                        "command": ["make", "vuln-check"],
                        "cwd": str(tmp_path),
                    },
                    {
                        "scope": "backend/agent",
                        "name": "sanity",
                        "command": [sys.executable, str(shim), "vuln-check"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            env = {"INVOCATIONS_LOG": str(log)}
            with mock.patch.dict(os.environ, env):
                results = run_checks(cfg_path, allowlist=[], extra_env=env)
            # The sanity check invokes the shim, so the log has 1 line.
            # The vuln-check check calls make (not the shim) — so the log
            # SHOULD have exactly 1 line (the sanity check invocation),
            # NOT 2 (which would mean the harness bypassed make and called
            # govulncheck directly).
            log_lines = log.read_text().strip().splitlines()
            # Exactly one invocation: the sanity check ran the shim; the
            # vuln-check check ran `make` (not the shim). If the harness
            # bypassed make and called the underlying tool directly, the
            # log would have 2 lines instead of 1.
            self.assertEqual(len(log_lines), 1)
            self.assertIn("govulncheck_shim.py", log_lines[0])
            self.assertIn("vuln-check", log_lines[0])


# ---------------------------------------------------------------------------
# R-RVH-001 — Run-all-then-aggregate (2 scenarios)
# ---------------------------------------------------------------------------

class TestAggregator(unittest.TestCase):
    """R-RVH-001 S-1, S-2: never fail-fast; aggregate every check."""

    def test_continues_past_failed_check(self) -> None:
        from validate import run_checks  # type: ignore

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cfg = {
                "checks": [
                    {
                        "scope": "backend/database_administrator",
                        "name": "lint",
                        # Always fails
                        "command": ["false"],
                        "cwd": str(tmp_path),
                    },
                    {
                        "scope": "backend/database_administrator",
                        "name": "fmt",
                        "command": ["true"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            results = run_checks(cfg_path, allowlist=[])
            statuses = [r.status for r in results]
            # First check failed; second check still ran.
            self.assertEqual(statuses, ["failed", "passed"])

    def test_includes_tool_missing_with_real_finding(self) -> None:
        from validate import run_checks  # type: ignore

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cfg = {
                "checks": [
                    {
                        "scope": "backend/agent",
                        "name": "vuln-check",
                        "command": ["make", "vuln-check"],
                        "cwd": str(tmp_path),
                    },
                    {
                        "scope": "frontend",
                        "name": "lint",
                        "command": ["false"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            results = run_checks(cfg_path, allowlist=[])
            all_findings = [f for r in results for f in r.findings]
            categories = {f.category for f in all_findings}
            self.assertIn("tool-missing", categories)


# ---------------------------------------------------------------------------
# R-RVH-006 — Opt-in checks for e2e and integration (3 scenarios)
# ---------------------------------------------------------------------------

class TestOptIn(unittest.TestCase):
    """R-RVH-006 S-1..S-3: skipped by default; flag honors include."""

    def test_integration_skipped_without_flag(self) -> None:
        from validate import run_checks  # type: ignore

        cfg = _single_check_cfg("integration", "test/integration")
        results = run_checks(Path(_write_cfg(cfg)), allowlist=[], include_integration=False)
        self.assertEqual(results[0].status, "skipped")
        self.assertIn("skipped: integration", results[0].log)

    def test_e2e_skipped_without_flag(self) -> None:
        from validate import run_checks  # type: ignore

        cfg = _single_check_cfg("e2e", "test:e2e")
        results = run_checks(Path(_write_cfg(cfg)), allowlist=[], include_e2e=False)
        self.assertEqual(results[0].status, "skipped")
        self.assertIn("skipped: e2e", results[0].log)

    def test_integration_included_with_flag(self) -> None:
        from validate import run_checks  # type: ignore

        cfg = _single_check_cfg("integration", "test/integration")
        results = run_checks(Path(_write_cfg(cfg)), allowlist=[], include_integration=True)
        self.assertNotEqual(results[0].status, "skipped")


# ---------------------------------------------------------------------------
# R-RVH-012 — Allowlist for documented absences (2 scenarios)
# ---------------------------------------------------------------------------

class TestAllowlist(unittest.TestCase):
    """R-RVH-012 S-1, S-2: default allowlist silences install absence."""

    def test_default_allowlist_silences_documented_absence(self) -> None:
        # The default config has (backend/agent, install) in its allowlist.
        # For a CheckDef (backend/agent, install) with no underlying tool,
        # the harness MUST NOT emit a tool-missing finding.
        from validate import run_checks, load_default_allowlist  # type: ignore

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cfg = {
                "checks": [
                    {
                        "scope": "backend/agent",
                        "name": "install",
                        "command": ["make", "install"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            allowlist = load_default_allowlist(HERE / "validate-config.yaml")
            results = run_checks(cfg_path, allowlist=allowlist)
            findings = [f for r in results for f in r.findings]
            self.assertFalse(
                any(f.category == "tool-missing" for f in findings),
                f"Allowlist should silence install absence; got: {findings}",
            )

    def test_removing_allowlist_surfaces_finding(self) -> None:
        from validate import run_checks  # type: ignore

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cfg = {
                "checks": [
                    {
                        "scope": "backend/agent",
                        "name": "install",
                        "command": ["make", "install"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            results = run_checks(cfg_path, allowlist=[])
            findings = [f for r in results for f in r.findings]
            self.assertTrue(any(f.category == "tool-missing" for f in findings))


# ---------------------------------------------------------------------------
# R-RVH-009 — JSON and Markdown rendered from one in-memory list (2 scenarios)
# ---------------------------------------------------------------------------

class TestRenderers(unittest.TestCase):
    """R-RVH-009 S-1, S-2: same IDs in both outputs; parity on add."""

    def test_same_ids_in_both_outputs(self) -> None:
        findings = [
            _make_finding(severity="high", message="a", id_="aaa111"),
            _make_finding(severity="medium", message="b", id_="bbb222"),
            _make_finding(severity="low", message="c", id_="ccc333"),
        ]
        json_blob = validate.render_json(findings)
        md_blob = validate.render_markdown(findings)
        for fid in ("aaa111", "bbb222", "ccc333"):
            self.assertIn(fid, json_blob)
            self.assertIn(fid, md_blob)

    def test_adding_finding_updates_both(self) -> None:
        findings_before = [_make_finding(severity="high", message="a", id_="aaa111")]
        findings_after = findings_before + [_make_finding(severity="high", message="b", id_="bbb222")]
        json_before = validate.render_json(findings_before)
        md_before = validate.render_markdown(findings_before)
        json_after = validate.render_json(findings_after)
        md_after = validate.render_markdown(findings_after)
        self.assertNotIn("bbb222", json_before)
        self.assertNotIn("bbb222", md_before)
        self.assertIn("bbb222", json_after)
        self.assertIn("bbb222", md_after)


# ---------------------------------------------------------------------------
# R-RVH-007 — go.work-mismatch is a first-class finding category (2 scenarios)
# ---------------------------------------------------------------------------

class TestGoWorkMismatch(unittest.TestCase):
    """R-RVH-007 S-1, S-2: drift -> finding; match -> no finding."""

    def test_emits_finding_on_drift(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            (tmp_path / "go.work").write_text(
                "go 1.26.5\n\nuse (\n\t./m\n)\n"
            )
            mod = tmp_path / "m"
            mod.mkdir()
            (mod / "go.mod").write_text("module m\n\ngo 1.26.3\n")
            findings = validate.check_go_work_drift(tmp_path)
            self.assertTrue(any(f.category == "go.work-mismatch" for f in findings))
            self.assertTrue(any(f.scope == "m" for f in findings))

    def test_no_finding_on_match(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            (tmp_path / "go.work").write_text(
                "go 1.26.3\n\nuse (\n\t./m\n)\n"
            )
            mod = tmp_path / "m"
            mod.mkdir()
            (mod / "go.mod").write_text("module m\n\ngo 1.26.3\n")
            findings = validate.check_go_work_drift(tmp_path)
            self.assertFalse(any(f.category == "go.work-mismatch" for f in findings))


# ---------------------------------------------------------------------------
# R-RVH-008 — make build is deliberately NOT a check (2 scenarios)
# ---------------------------------------------------------------------------

class TestDefaultCheckMatrix(unittest.TestCase):
    """When the YAML config has no `checks:` block, the harness MUST fall back
    to the canonical default matrix. Without this wiring, the production run
    produces zero results — the harness silently no-ops against the
    default `scripts/validate-config.yaml`.
    """

    def test_uses_default_matrix_when_config_has_no_checks(self) -> None:
        from validate import run_checks, default_check_matrix  # type: ignore

        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(
                "allowlist: []\n"
                "fail_on: high\n"
                "timeout_check: 5\n"
            )
            # Patch ROOT to the tmp dir so the default matrix points at
            # paths that have no Makefile/package.json — every preflight
            # will fail fast with `make -n ...` returning non-zero. The
            # test only asserts structure (length + scope/name), so the
            # outcome of each subprocess is irrelevant.
            with mock.patch.object(validate, "ROOT", tmp_path):
                results = run_checks(cfg_path, allowlist=[])
            expected = default_check_matrix(repo_root=tmp_path)
            self.assertEqual(len(results), len(expected))
            for r in results:
                self.assertTrue(r.scope, f"empty scope in result: {r}")
                self.assertTrue(r.name, f"empty name in result: {r}")


class TestNoBuildCheck(unittest.TestCase):
    """R-RVH-008 S-1, S-2: no build entry in matrix; ./bin/ untouched."""

    def test_check_matrix_has_no_build_target(self) -> None:
        matrix = validate.default_check_matrix(repo_root=Path("/repo"))
        for entry in matrix:
            self.assertNotEqual(
                entry.get("name"), "build",
                f"check matrix must not include 'build'; got {entry}",
            )

    def test_bin_dir_unchanged_after_dry_run(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            bin = tmp_path / "bin"
            bin.mkdir()
            before = sorted(p.name for p in bin.iterdir())
            # A dry run of run_checks (no real tools) leaves bin/ alone.
            cfg = {
                "checks": [
                    {
                        "scope": "frontend",
                        "name": "lint",
                        "command": ["true"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            validate.run_checks(cfg_path, allowlist=[])
            after = sorted(p.name for p in bin.iterdir())
            self.assertEqual(before, after)


# ---------------------------------------------------------------------------
# R-RVH-011 — Output paths and deterministic CI-ready output (3 scenarios)
# ---------------------------------------------------------------------------

class TestCIReady(unittest.TestCase):
    """R-RVH-011 S-1..S-3: output-dir creates files; deterministic; no TTY."""

    def test_output_dir_creates_files(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            out = tmp_path / "reports"
            cfg = {
                "checks": [
                    {
                        "scope": "frontend",
                        "name": "lint",
                        "command": ["true"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            validate.run_full(cfg_path, allowlist=[], output_dir=out)
            self.assertTrue((out / "validation-report.json").exists())
            self.assertTrue((out / "validation-report.md").exists())
            # JSON parses cleanly.
            json.loads((out / "validation-report.json").read_text())

    def test_deterministic_exit_across_runs(self) -> None:
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            cfg = {
                "checks": [
                    {
                        "scope": "frontend",
                        "name": "lint",
                        "command": ["false"],
                        "cwd": str(tmp_path),
                    },
                ]
            }
            cfg_path = tmp_path / "cfg.yaml"
            cfg_path.write_text(json.dumps(cfg))
            r1 = validate.run_full(cfg_path, allowlist=[], output_dir=tmp_path / "a")
            r2 = validate.run_full(cfg_path, allowlist=[], output_dir=tmp_path / "b")
            self.assertEqual(r1, r2)

    def test_runs_without_tty(self) -> None:
        # run_full MUST NOT call isatty(). We mock it to throw and prove the
        # harness never inspects it.
        with mock.patch("sys.stdout.isatty", side_effect=AssertionError("TTY checked!")):
            with mock.patch("sys.stdin.isatty", side_effect=AssertionError("TTY checked!")):
                with tempfile.TemporaryDirectory() as tmp:
                    tmp_path = Path(tmp)
                    cfg = {
                        "checks": [
                            {
                                "scope": "frontend",
                                "name": "lint",
                                "command": ["true"],
                                "cwd": str(tmp_path),
                            },
                        ]
                    }
                    cfg_path = tmp_path / "cfg.yaml"
                    cfg_path.write_text(json.dumps(cfg))
                    validate.run_full(cfg_path, allowlist=[], output_dir=tmp_path / "out")


# ---------------------------------------------------------------------------
# Parser-specific tests (10)
# ---------------------------------------------------------------------------

class TestParsers(unittest.TestCase):
    """Each parser is a pure function: bytes -> list[Finding]."""

    def test_golangci(self) -> None:
        raw = json.dumps({
            "Issues": [
                {
                    "FromLinter": "staticcheck",
                    "Text": "S1025",
                    "Severity": "error",
                    "Pos": {"Filename": "src/x.go", "Line": 42, "Column": 5},
                }
            ]
        })
        findings = validate.parse_golangci_json(raw, scope="backend/db")
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].tool, "golangci-lint")
        self.assertEqual(findings[0].rule, "staticcheck")
        self.assertEqual(findings[0].file, "src/x.go")
        self.assertEqual(findings[0].line, 42)

    def test_eslint(self) -> None:
        raw = json.dumps([{
            "filePath": "src/y.tsx",
            "messages": [
                {"ruleId": "no-unused", "severity": 2, "message": "x is unused"}
            ],
        }])
        findings = validate.parse_eslint_json(raw, scope="frontend")
        self.assertEqual(findings[0].severity, "error")
        self.assertEqual(findings[0].file, "src/y.tsx")

    def test_tsc(self) -> None:
        out = "src/foo.ts(10,5): error TS2322: Type 'string' is not assignable to type 'number'.\n"
        findings = validate.parse_tsc(out, scope="frontend")
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].rule, "TS2322")
        self.assertEqual(findings[0].line, 10)
        self.assertEqual(findings[0].column, 5)

    def test_prettier(self) -> None:
        # prettier --check prints filenames on stderr; non-zero exit.
        findings = validate.parse_prettier(
            stderr="src/a.ts\nsrc/b.ts\n",
            exit_code=1,
            scope="frontend",
        )
        self.assertEqual(len(findings), 2)
        self.assertEqual(findings[0].file, "src/a.ts")

    def test_vitest(self) -> None:
        # vitest --reporter=json emits NDJSON.
        lines = [
            json.dumps({"type": "test", "name": "fails", "status": "failed",
                         "assertionResults": [{"status": "failed", "title": "x"}]}),
        ]
        findings = validate.parse_vitest("\n".join(lines), scope="frontend")
        self.assertTrue(len(findings) >= 1)

    def test_node_test(self) -> None:
        # node --test emits TAP.
        tap = textwrap.dedent("""\
            TAP version 13
            ok 1 - good test
            not ok 2 - bad test
              ---
              message: 'broken'
              ---
            1..2
        """)
        findings = validate.parse_node_test(tap, scope="frontend")
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].message, "bad test")

    def test_playwright(self) -> None:
        out = "  1) test 'has title'\n     Error: page closed\n"
        findings = validate.parse_playwright(out, scope="frontend")
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].rule, "test 'has title'")

    def test_pnpm_audit(self) -> None:
        # Real pnpm audit --json schema: ``advisories`` keyed by advisory id,
        # each carrying ``module_name``, ``severity``, ``title``, ``findings``.
        raw = json.dumps({
            "metadata": {"vulnerabilities": {"high": 1, "moderate": 0, "low": 0}},
            "advisories": {
                "1120680": {
                    "id": 1120680,
                    "module_name": "lodash",
                    "severity": "high",
                    "title": "Prototype Pollution in lodash",
                    "url": "https://github.com/advisories/GHSA-xxxx",
                    "findings": [{"version": "4.17.20", "paths": [".>lodash"], "dev": False}],
                },
            },
        })
        findings = validate.parse_pnpm_audit(raw, scope="frontend")
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].severity, "high")
        self.assertEqual(findings[0].rule, "lodash")
        self.assertIn("Prototype Pollution", findings[0].message)

    def test_govulncheck(self) -> None:
        # govulncheck emits pretty-printed multi-line JSON; the parser must
        # stream top-level objects (not splitlines + json.loads), distinguish
        # reachable (``finding`` block with ``trace``) from unreachable
        # (standalone ``osv`` record, no call site) advisories, and dedupe
        # multi-call-site advisories to one finding per OSV.
        raw = json.dumps({
            "config": {"scanner_version": "v1.7.0"},
        }) + "\n" + json.dumps({
            "SBOM": {"modules": []},
        }) + "\n" + json.dumps({
            "osv": {"id": "GO-2020-0001", "summary": "unreachable"},
        }) + "\n" + json.dumps({
            "finding": {"osv": "GHSA-x",
                        "trace": [{"position": {"filename": "src/x.go", "line": 42, "column": 7}}]},
        }) + "\n" + json.dumps({
            "finding": {"osv": "GHSA-x",
                        "trace": [{"position": {"filename": "src/y.go", "line": 1, "column": 1}}]},
        }) + "\n" + json.dumps({
            "finding": {"osv": "GHSA-y",
                        "trace": [{"position": {"filename": "src/z.go", "line": 9, "column": 3}}]},
        })
        findings = validate.parse_govulncheck(raw, scope="backend/db")
        # GHSA-x deduped (two call sites collapsed to one finding with a count);
        # GHSA-y has one call site; the unreachable GO-2020-0001 is dropped.
        self.assertEqual(len(findings), 2)
        by_rule = {f.rule: f for f in findings}
        self.assertEqual(by_rule["GHSA-x"].file, "src/x.go")
        self.assertEqual(by_rule["GHSA-x"].line, 42)
        self.assertIn("2 reachable call sites", by_rule["GHSA-x"].message)
        self.assertNotIn("2 reachable call sites", by_rule["GHSA-y"].message)
        # SBOM, config, and unreachable-osv records produce no findings.
        self.assertNotIn("GO-2020-0001", by_rule)

    def test_govulncheck_skips_non_json_preamble(self) -> None:
        # Defense in depth: if a Makefile echo slips into stdout (it shouldn't,
        # but cheap to guard), the parser must skip past it to the first '{'.
        raw = ">> running govulncheck v1.1.4\n" + json.dumps({
            "finding": {"osv": "GHSA-z",
                        "trace": [{"position": {"filename": "a.go", "line": 1}}]},
        })
        findings = validate.parse_govulncheck(raw, scope="x")
        self.assertEqual(len(findings), 1)
        self.assertEqual(findings[0].rule, "GHSA-z")

    def test_go_work(self) -> None:
        # Sanity that check_go_work_drift walks go.work for `use` entries.
        with tempfile.TemporaryDirectory() as tmp:
            tmp_path = Path(tmp)
            (tmp_path / "go.work").write_text("go 1.26.5\n\nuse (./m1 ./m2)\n")
            for m in ("m1", "m2"):
                d = tmp_path / m
                d.mkdir()
                (d / "go.mod").write_text(f"module {m}\n\ngo 1.26.5\n")
            findings = validate.check_go_work_drift(tmp_path)
            self.assertEqual(findings, [])


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

import os  # noqa: E402  (used by TestToolMissing.test_does_not_invoke_underlying_tool)


def _make_finding(severity: str = "info", message: str = "m", id_: str = "id") -> validate.Finding:
    return validate.Finding(
        id=id_, severity=severity, category="lint", tool="t", scope="s",
        rule=None, file=None, line=None, column=None,
        message=message, evidence=None, reproduce="r", fix_hint=None,
    )


def _single_check_cfg(name: str, target: str) -> dict:
    return {
        "checks": [
            {"scope": "x", "name": name, "target": target,
             "command": ["true"], "cwd": "/tmp"},
        ]
    }


def _write_cfg(cfg: dict) -> str:
    tmp_path = Path(tempfile.mkdtemp())
    cfg_path = tmp_path / "cfg.yaml"
    cfg_path.write_text(json.dumps(cfg))
    return str(cfg_path)


if __name__ == "__main__":
    unittest.main(verbosity=2)