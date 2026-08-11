#!/usr/bin/env -S uv run --quiet --no-project --with pyyaml python3
"""Repo-level validation harness for cachicamas.

Run with:
    ./scripts/validate.py                   # default run (no e2e / integration)
    ./scripts/validate.py --include-e2e
    ./scripts/validate.py --include-integration
    ./scripts/validate.py --output-dir reports
    ./scripts/validate.py --fail-on medium

Emits:
    validation-report.json    canonical, machine-parseable
    validation-report.md      human review

Exit codes:
    0   no findings above the configured failure threshold
    1   one or more findings at/above the threshold
    2   harness internal error (bad config, unwritable output, bad flag)

No TTY dependency; no live network beyond what the underlying tools do
on first invocation; no `make build` side effect; deterministic output
given the same input.

Spec: openspec/changes/repo-validation-harness/specs/repo-validation-harness/spec.md
Design: openspec/changes/repo-validation-harness/design.md
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
import re
import shutil
import subprocess
import sys
import tempfile
from dataclasses import dataclass, field, asdict
from datetime import datetime, timezone
from pathlib import Path
from typing import Any, Iterable

try:
    import yaml  # noqa: F401  (provided by the uv shebang / --with flag)
except ImportError:
    sys.stderr.write(
        "validate.py needs PyYAML. Run with `uv run scripts/validate.py` so "
        "uv can resolve pyyaml automatically.\n"
    )
    sys.exit(2)


ROOT = Path(__file__).resolve().parent.parent  # scripts/ -> repo root
DEFAULT_CONFIG = ROOT / "scripts" / "validate-config.yaml"


# ---------------------------------------------------------------------------
# Custom errors
# ---------------------------------------------------------------------------


class HarnessConfigError(Exception):
    """Raised when the harness itself is misconfigured (exit 2)."""


class HarnessInternalError(Exception):
    """Raised for any other harness-internal failure (exit 2)."""


# ---------------------------------------------------------------------------
# Data shapes (R-RVH-010: 13 fields, in this order)
# ---------------------------------------------------------------------------


@dataclass
class Finding:
    id: str
    severity: str       # verbatim from tool ("warn", "error", "high", ...)
    category: str       # lint|typecheck|format|test|vuln|tool-missing|go.work-mismatch|timeout
    tool: str           # golangci-lint|eslint|...
    scope: str          # backend/<mod>|frontend|repo-root
    rule: str | None
    file: str | None    # repo-relative
    line: int | None
    column: int | None
    message: str
    evidence: str | None  # raw tool output, truncated 4 KiB
    reproduce: str        # single shell command from repo root
    fix_hint: str | None


@dataclass
class CheckResult:
    scope: str
    name: str
    target: str
    status: str  # passed|failed|skipped|tool-missing|timeout|internal-error
    log: str = ""
    elapsed_seconds: float = 0.0
    findings: list[Finding] = field(default_factory=list)


def finding_to_jsonable(f: Finding) -> dict[str, Any]:
    """Return a dict with all 13 fields; None stays as JSON null."""
    return asdict(f)


# ---------------------------------------------------------------------------
# Path / ID utilities (R-RVH-004, R-RVH-010 S-3)
# ---------------------------------------------------------------------------


_EVIDENCE_MAX = 4096


def truncate_evidence(raw: str | None) -> str | None:
    if raw is None:
        return None
    if len(raw) <= _EVIDENCE_MAX:
        return raw
    return raw[: _EVIDENCE_MAX - 16] + "\n...[truncated]"


def normalize_repo_relative(path: str | Path, repo_root: Path) -> str:
    """Strip leading slash, collapse `..`/`.` segments; never touch the FS.

    The contract (R-RVH-010 S-3): output MUST have no leading `/` and no `..`.
    Real production paths resolve under repo_root via direct prefix match;
    fixtures and partial paths go through lexical collapse only.
    """
    if path is None:
        return ""
    s = str(path)
    rr_str = str(repo_root)
    # 1) Direct prefix match (production case).
    if s.startswith(rr_str + "/"):
        suffix = s[len(rr_str) + 1 :]
    elif s == rr_str:
        return ""
    else:
        suffix = s
    # 2) Lexical cleanup.
    suffix = suffix.lstrip("/")
    parts: list[str] = []
    for part in suffix.split("/"):
        if part == "" or part == ".":
            continue
        if part == "..":
            if parts:
                parts.pop()
            continue
        parts.append(part)
    # 3) If repo_root's parts appear at the tail, strip them.
    rr_parts = Path(rr_str).parts
    if len(parts) >= len(rr_parts) and tuple(parts[-len(rr_parts) :]) == rr_parts:
        parts = parts[: -len(rr_parts)]
    return "/".join(parts)


def compute_finding_id(
    *,
    file: str | None,
    line: int | None,
    rule: str | None,
    message: str,
) -> str:
    """sha256(file|line|rule|message)[:16] with explicit normalization.

    Per spec S-3: "absent-vs-zero is collapsed to the same normalized input".
    We map both None and 0 to the empty string before hashing so they are
    byte-identical.
    """
    file_s = (file or "")
    # Absent line AND 0 both normalize to "" (S-3).
    line_s = "" if (line is None or line == 0) else str(line)
    rule_s = (rule or "")
    msg_s = message or ""
    payload = f"{file_s}|{line_s}|{rule_s}|{msg_s}".encode("utf-8")
    return hashlib.sha256(payload).hexdigest()[:16]


# ---------------------------------------------------------------------------
# Severity ladder (R-RVH-005: verbatim tool severity preserved)
# ---------------------------------------------------------------------------


# Tool severity  ->  our mapped (threshold-comparable) bucket
_SEVERITY_LADDER: dict[str, str] = {
    "0": "info", "off": "info",
    "1": "low", "warn": "low",
    "2": "medium", "error": "medium",
    "moderate": "medium", "MEDIUM": "medium",
    "high": "high", "HIGH": "high",
    "critical": "critical", "CRITICAL": "critical",
    # Identity pass-through for the bucket names themselves: the harness's
    # own synthetic findings (tool-missing, timeout, go.work-mismatch) use
    # the mapped bucket as the verbatim severity so the summary reflects
    # the right bucket without a second hop. These entries override the
    # tool-severity entries above for any input that happens to match
    # (e.g. a tool that emits bare "low" gets bucketed as "low" instead
    # of being promoted to "medium").
    "info": "info",
    "low": "low",
    "medium": "medium",
    "high": "high",
    "critical": "critical",
}


# Threshold ladder for exit code
_THRESHOLD_ORDER = {"info": 0, "low": 1, "medium": 2, "high": 3, "critical": 4}


def map_severity(verbatim: str) -> str:
    """Map a verbatim severity string to the threshold-comparable bucket.

    The verbatim string is preserved on Finding.severity. This map is only
    used internally for threshold comparison.
    """
    return _SEVERITY_LADDER.get(verbatim, "low")


def compute_exit(findings: Iterable[Finding], threshold: str) -> int:
    """Return 0 if no finding >= threshold, else 1. Internal errors raise."""
    t_rank = _THRESHOLD_ORDER.get(threshold)
    if t_rank is None:
        raise HarnessConfigError(f"unknown --fail-on threshold: {threshold!r}")
    for f in findings:
        if _THRESHOLD_ORDER.get(map_severity(f.severity), 1) >= t_rank:
            return 1
    return 0


# ---------------------------------------------------------------------------
# Config loading
# ---------------------------------------------------------------------------


def load_config(path: str | Path) -> dict[str, Any]:
    """Load and minimally validate the harness config YAML.

    Raises HarnessConfigError (caller exits 2) if the file is missing,
    malformed, or doesn't parse as YAML.
    """
    p = Path(path)
    if not p.exists():
        raise HarnessConfigError(f"config not found: {p}")
    try:
        text = p.read_text(encoding="utf-8")
        data = yaml.safe_load(text)
    except yaml.YAMLError as e:
        raise HarnessConfigError(f"malformed YAML in {p}: {e}") from e
    except OSError as e:
        raise HarnessConfigError(f"unreadable config {p}: {e}") from e
    if not isinstance(data, dict):
        raise HarnessConfigError(
            f"config {p} must be a YAML mapping, got {type(data).__name__}"
        )
    # Normalise top-level keys.
    out: dict[str, Any] = {
        "allowlist": data.get("allowlist") or [],
        "fail_on": data.get("fail_on") or "high",
        "timeout_check": int(data.get("timeout_check") or 600),
        "timeout_e2e": int(data.get("timeout_e2e") or 1200),
    }
    if out["fail_on"] not in _THRESHOLD_ORDER:
        raise HarnessConfigError(
            f"invalid fail_on {out['fail_on']!r}; expected one of {sorted(_THRESHOLD_ORDER)}"
        )
    return out


def load_default_allowlist(config_path: Path = DEFAULT_CONFIG) -> list[tuple[str, str]]:
    """Read the default config and return its allowlist as (scope, target) pairs."""
    cfg = load_config(config_path)
    pairs: list[tuple[str, str]] = []
    for entry in cfg["allowlist"]:
        if not (isinstance(entry, dict) and "scope" in entry and "target" in entry):
            raise HarnessConfigError(
                f"allowlist entry must have 'scope' and 'target'; got {entry!r}"
            )
        pairs.append((entry["scope"], entry["target"]))
    return pairs


# ---------------------------------------------------------------------------
# Default check matrix (R-RVH-008: no build target)
# ---------------------------------------------------------------------------


def default_check_matrix(repo_root: Path) -> list[dict[str, Any]]:
    """Return the canonical 21-entry check matrix.

    R-RVH-008 mandates that `build` is NEVER a check. Every entry has the
    keys: scope, name, command, cwd, target, opt_in (None|"integration"|"e2e").
    """
    frontend = repo_root / "frontend"
    mods = {
        "database_administrator": repo_root / "backend" / "database_administrator",
        "workspace_syncer": repo_root / "backend" / "workspace_syncer",
        "agent": repo_root / "backend" / "agent",
    }
    matrix: list[dict[str, Any]] = []
    for scope_name, cwd in mods.items():
        scope = f"backend/{scope_name}"
        matrix.append({
            "scope": scope,
            "name": "test",
            "target": "test",
            "command": ["make", "test"],
            "cwd": str(cwd),
            "opt_in": None,
        })
        matrix.append({
            "scope": scope,
            "name": "test/cover",
            "target": "test/cover",
            "command": ["make", "test/cover"],
            "cwd": str(cwd),
            "opt_in": None,
        })
        matrix.append({
            "scope": scope,
            "name": "lint",
            "target": "lint",
            "command": ["make", "lint"],
            "cwd": str(cwd),
            "opt_in": None,
        })
        # vuln-check target is missing in backend/agent by design.
        if scope_name != "agent":
            matrix.append({
                "scope": scope,
                "name": "vuln-check",
                "target": "vuln-check",
                "command": ["make", "vuln-check"],
                "cwd": str(cwd),
                "opt_in": None,
            })
        else:
            # Surface as tool-missing at runtime; do NOT bypass.
            matrix.append({
                "scope": scope,
                "name": "vuln-check",
                "target": "vuln-check",
                "command": ["make", "vuln-check"],
                "cwd": str(cwd),
                "opt_in": None,
            })
        # test/integration: opt-in (R-RVH-006).
        if scope_name == "database_administrator":
            matrix.append({
                "scope": scope,
                "name": "integration",
                "target": "test/integration",
                "command": ["make", "test/integration"],
                "cwd": str(cwd),
                "opt_in": "integration",
            })
    # Frontend checks (R-RVH-006: e2e opt-in).
    matrix.extend([
        {
            "scope": "frontend",
            "name": "lint",
            "target": "lint",
            "command": ["pnpm", "--dir", str(frontend), "lint"],
            "cwd": str(frontend),
            "opt_in": None,
        },
        {
            "scope": "frontend",
            "name": "typecheck",
            "target": "build.types",
            "command": ["pnpm", "--dir", str(frontend), "build.types"],
            "cwd": str(frontend),
            "opt_in": None,
        },
        {
            "scope": "frontend",
            "name": "format",
            "target": "fmt.check",
            "command": ["pnpm", "--dir", str(frontend), "fmt.check"],
            "cwd": str(frontend),
            "opt_in": None,
        },
        {
            "scope": "frontend",
            "name": "test",
            "target": "test:ci",
            "command": ["pnpm", "--dir", str(frontend), "test:ci"],
            "cwd": str(frontend),
            "opt_in": None,
        },
        {
            "scope": "frontend",
            "name": "eslint-rule-test",
            "target": "node-test",
            "command": ["node", "--test", "eslint-rules/*.spec.mjs"],
            "cwd": str(frontend),
            "opt_in": None,
        },
        {
            "scope": "frontend",
            "name": "vuln-check",
            "target": "vuln-check",
            "command": ["pnpm", "--dir", str(frontend), "run", "vuln-check"],
            "cwd": str(frontend),
            "opt_in": None,
        },
        {
            "scope": "frontend",
            "name": "e2e",
            "target": "test:e2e",
            "command": ["pnpm", "--dir", str(frontend), "test:e2e"],
            "cwd": str(frontend),
            "opt_in": "e2e",
        },
    ])
    # Repo-root go.work drift check (R-RVH-007).
    matrix.append({
        "scope": "repo-root",
        "name": "go.work-drift",
        "target": "go.work",
        "command": [],  # marker: synthetic, not a subprocess
        "cwd": str(repo_root),
        "opt_in": None,
    })
    return matrix


# ---------------------------------------------------------------------------
# Preflight (R-RVH-003: detect absent target, don't crash)
# ---------------------------------------------------------------------------


def _preflight(check: dict[str, Any], allowlist: Iterable[tuple[str, str]]) -> Finding | None:
    """Return a tool-missing Finding if the check can't run; None if OK."""
    cmd = check.get("command") or []
    scope = check["scope"]
    name = check["name"]
    target = check.get("target", name)
    # Synthetic check (no command) -> skip preflight; it's handled inline.
    if not cmd:
        return None
    # Allowlisted absence: documented on purpose.
    if (scope, target) in allowlist:
        return None
    binary = cmd[0]
    cwd = check.get("cwd") or "."
    # Non-make commands: check binary on PATH.
    if binary != "make":
        if shutil.which(binary) is None:
            return _tool_missing_finding(
                scope=scope, name=name, target=target, message=f"binary not on PATH: {binary}",
                reproduce=" ".join(cmd), binary_missing=binary,
            )
        return None
    # make command: dry-run the target.
    preflight = ["make", "-n", target]
    try:
        proc = subprocess.run(
            preflight, cwd=cwd, capture_output=True, text=True, timeout=15,
        )
    except subprocess.TimeoutExpired:
        return _tool_missing_finding(
            scope=scope, name=name, target=target,
            message=f"`make -n {target}` timed out (no Makefile or hung make?)",
            reproduce=" ".join(preflight),
        )
    except FileNotFoundError:
        return _tool_missing_finding(
            scope=scope, name=name, target=target, message="`make` not on PATH",
            reproduce=" ".join(preflight), binary_missing="make",
        )
    # Exit non-zero => absent target (no Makefile, no such target, etc.).
    if proc.returncode != 0:
        return _tool_missing_finding(
            scope=scope, name=name, target=target,
            message=(
                f"`make -n {target}` returned exit {proc.returncode} "
                f"(target absent or Makefile missing)"
            ),
            reproduce=" ".join(preflight),
        )
    return None


def _tool_missing_finding(
    *, scope: str, name: str, target: str, message: str, reproduce: str,
    binary_missing: str | None = None,
) -> Finding:
    fid = compute_finding_id(file=None, line=None, rule=name, message=message)
    return Finding(
        id=fid,
        severity="medium",  # R-RVH-003 synthetic severity
        category="tool-missing",
        tool="make" if not binary_missing else binary_missing,
        scope=scope,
        rule=name,
        file=None,
        line=None,
        column=None,
        message=message,
        evidence=None,
        reproduce=reproduce,
        fix_hint=(
            "Add the target to the module Makefile, or wire the harness to a "
            "different scope if the gap is documented."
        ),
    )


# ---------------------------------------------------------------------------
# Subprocess runner (R-RVH-001 run-all-then-aggregate, timeout, no TTY)
# ---------------------------------------------------------------------------


def _run_subprocess(
    cmd: list[str], cwd: str, timeout: float, extra_env: dict[str, str] | None = None,
) -> tuple[int | None, str, str, str]:
    """Run cmd; return (exit_code_or_None, stdout, stderr, status).

    status is one of: 'ok', 'timeout', 'notfound'.
    """
    env = None
    if extra_env:
        env = {**os.environ, **extra_env}
    try:
        proc = subprocess.run(
            cmd, cwd=cwd, capture_output=True, text=True,
            timeout=timeout, env=env, check=False,
        )
        return proc.returncode, proc.stdout, proc.stderr, "ok"
    except subprocess.TimeoutExpired as e:
        return None, (e.stdout.decode("utf-8", "replace") if isinstance(e.stdout, bytes) else (e.stdout or "")), (e.stderr.decode("utf-8", "replace") if isinstance(e.stderr, bytes) else (e.stderr or "")), "timeout"
    except FileNotFoundError:
        return None, "", "", "notfound"


# ---------------------------------------------------------------------------
# Tool output parsers (R-RVH-005: severity verbatim; R-RVH-010: 13 fields)
# ---------------------------------------------------------------------------


def _finding_from_tool(
    *,
    tool: str,
    scope: str,
    severity: str,
    category: str,
    rule: str | None,
    file: str | None,
    line: int | None,
    column: int | None,
    message: str,
    evidence: str | None,
    reproduce: str,
    fix_hint: str | None = None,
    repo_root: Path = ROOT,
) -> Finding:
    return Finding(
        id=compute_finding_id(file=file, line=line, rule=rule, message=message),
        severity=severity,
        category=category,
        tool=tool,
        scope=scope,
        rule=rule,
        file=normalize_repo_relative(file, repo_root) if file else None,
        line=line,
        column=column,
        message=message,
        evidence=truncate_evidence(evidence),
        reproduce=reproduce,
        fix_hint=fix_hint,
    )


def parse_golangci_json(raw: str, scope: str) -> list[Finding]:
    try:
        doc = json.loads(raw)
        issues = doc.get("Issues") or []
    except (json.JSONDecodeError, AttributeError, TypeError):
        return [_finding_from_tool(
            tool="golangci-lint", scope=scope, severity="medium",
            category="tool-missing", rule=None, file=None, line=None, column=None,
            message="golangci-lint output was not valid JSON",
            evidence=raw, reproduce="see log",
        )]
    out: list[Finding] = []
    for it in issues:
        pos = it.get("Pos") or {}
        out.append(_finding_from_tool(
            tool="golangci-lint", scope=scope,
            severity=str(it.get("Severity") or "error"),
            category="lint",
            rule=it.get("FromLinter"),
            file=pos.get("Filename"),
            line=pos.get("Line"),
            column=pos.get("Column"),
            message=it.get("Text") or "",
            evidence=json.dumps(it, ensure_ascii=False),
            reproduce=f"cd {scope} && bin/golangci-lint run --out-format=json ./...",
        ))
    return out


def parse_eslint_json(raw: str, scope: str) -> list[Finding]:
    """Parse ESLint 9 JSON output.

    Severity 1 == warn, 2 == error. Verbatim preserved on Finding.severity.
    """
    try:
        arr = json.loads(raw)
    except json.JSONDecodeError:
        return [_finding_from_tool(
            tool="eslint", scope=scope, severity="medium",
            category="tool-missing", rule=None, file=None, line=None, column=None,
            message="eslint output was not valid JSON",
            evidence=raw, reproduce="see log",
        )]
    if not isinstance(arr, list):
        return [_finding_from_tool(
            tool="eslint", scope=scope, severity="medium",
            category="tool-missing", rule=None, file=None, line=None, column=None,
            message=f"eslint JSON shape unexpected: {type(arr).__name__}",
            evidence=raw, reproduce="see log",
        )]
    out: list[Finding] = []
    severity_map = {"0": "off", "1": "warn", "2": "error"}
    for file_obj in arr:
        path = file_obj.get("filePath") or ""
        for msg in file_obj.get("messages") or []:
            raw_sev = msg.get("severity")
            sev = severity_map.get(str(raw_sev), str(raw_sev))
            out.append(_finding_from_tool(
                tool="eslint", scope=scope,
                severity=sev,  # R-RVH-005 verbatim: "warn" stays "warn"
                category="lint",
                rule=msg.get("ruleId"),
                file=path,
                line=msg.get("line"),
                column=msg.get("column"),
                message=msg.get("message") or "",
                evidence=json.dumps(msg, ensure_ascii=False),
                reproduce="cd frontend && pnpm lint",
            ))
    return out


_TSC_RE = re.compile(r"^(.+?)\((\d+),(\d+)\): error (TS\d+): (.+)$")


def parse_tsc(out: str, scope: str) -> list[Finding]:
    findings: list[Finding] = []
    for line in out.splitlines():
        m = _TSC_RE.match(line)
        if not m:
            continue
        file, line_no, col, code, msg = m.groups()
        findings.append(_finding_from_tool(
            tool="tsc", scope=scope,
            severity="error", category="typecheck",
            rule=code, file=file, line=int(line_no), column=int(col),
            message=msg, evidence=line,
            reproduce="cd frontend && pnpm build.types",
        ))
    return findings


def parse_prettier(stderr: str, exit_code: int, scope: str) -> list[Finding]:
    if exit_code == 0:
        return []
    out: list[Finding] = []
    for path in (line.strip() for line in stderr.splitlines() if line.strip()):
        out.append(_finding_from_tool(
            tool="prettier", scope=scope,
            severity="warn", category="format",
            rule="prettier", file=path, line=None, column=None,
            message="file would be reformatted",
            evidence=path,
            reproduce=f"cd frontend && pnpm exec prettier --write {path}",
            fix_hint="Run `cd frontend && pnpm fmt` to reformat.",
        ))
    if not out and exit_code != 0:
        out.append(_finding_from_tool(
            tool="prettier", scope=scope,
            severity="medium", category="tool-missing",
            rule=None, file=None, line=None, column=None,
            message=f"prettier exited {exit_code} but produced no file list",
            evidence=stderr,
            reproduce="cd frontend && pnpm fmt.check",
        ))
    return out


def parse_vitest(raw: str, scope: str) -> list[Finding]:
    """Vitest --reporter=json emits NDJSON; we keep test-level failures."""
    findings: list[Finding] = []
    for line in raw.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            doc = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not isinstance(doc, dict):
            continue
        # Vitest JSON shape varies; the field we care about is per-test status.
        status = doc.get("status")
        name = doc.get("name") or ""
        if status and status not in ("failed", "passed", "skipped"):
            continue
        if status == "failed":
            findings.append(_finding_from_tool(
                tool="vitest", scope=scope,
                severity="high", category="test",
                rule=None, file=None, line=None, column=None,
                message=f"test failed: {name}",
                evidence=line,
                reproduce="cd frontend && pnpm test:ci",
            ))
    return findings


_NODE_TEST_FAIL_RE = re.compile(r"^not ok \d+ - (.+)$")


def parse_node_test(tap: str, scope: str) -> list[Finding]:
    findings: list[Finding] = []
    for line in tap.splitlines():
        m = _NODE_TEST_FAIL_RE.match(line)
        if m:
            findings.append(_finding_from_tool(
                tool="node-test", scope=scope,
                severity="high", category="test",
                rule=None, file=None, line=None, column=None,
                message=m.group(1), evidence=line,
                reproduce="cd frontend && node --test eslint-rules/*.spec.mjs",
            ))
    return findings


_PLAYWRIGHT_RE = re.compile(r"^\s*\d+\)\s+(.+)$")


def parse_playwright(out: str, scope: str) -> list[Finding]:
    """Walk Playwright output two-line aware: numbered title, then `Error:`."""
    findings: list[Finding] = []
    pending_title: str | None = None
    for line in out.splitlines():
        title_match = _PLAYWRIGHT_RE.match(line)
        if title_match:
            pending_title = title_match.group(1).strip()
            # Don't emit yet; the next `Error:` line confirms a real failure.
            continue
        if "Error:" in line and pending_title:
            findings.append(_finding_from_tool(
                tool="playwright", scope=scope,
                severity="high", category="test",
                rule=pending_title, file=None, line=None, column=None,
                message=pending_title, evidence=line,
                reproduce="cd frontend && pnpm test:e2e",
            ))
            pending_title = None
            continue
        # Reset the pending title on any other line so we don't pair
        # stale titles with later errors.
        if not line.startswith(" ") and not line.startswith("\t"):
            pending_title = None
    return findings


def parse_pnpm_audit(raw: str, scope: str) -> list[Finding]:
    # pnpm audit --json may append a non-JSON trailer such as
    # "[ELIFECYCLE] Command failed with exit code 1." when vulnerabilities
    # are found. Find the last balanced top-level '}' and parse only the
    # JSON prefix.
    json_text = raw
    end = json_text.rfind("\n}")
    if end != -1:
        json_text = json_text[: end + 2]
    try:
        doc = json.loads(json_text)
    except json.JSONDecodeError:
        return [_finding_from_tool(
            tool="pnpm-audit", scope=scope, severity="medium",
            category="tool-missing", rule=None, file=None, line=None, column=None,
            message="pnpm audit output was not valid JSON",
            evidence=raw, reproduce="see log",
        )]
    # pnpm's JSON schema: { "advisories": { <id>: { module_name, severity, title, findings: [...] } }, "metadata": {...} }
    advisories = doc.get("advisories") or {}
    findings: list[Finding] = []
    for adv_id, info in advisories.items():
        if not isinstance(info, dict):
            continue
        sev = (info.get("severity") or "medium").lower()
        module_name = info.get("module_name") or adv_id
        findings.append(_finding_from_tool(
            tool="pnpm-audit", scope=scope,
            severity=sev, category="vuln",
            rule=module_name, file="package.json", line=None, column=None,
            message=f"{module_name}: {info.get('title', 'vulnerability')}",
            evidence=json.dumps(info, ensure_ascii=False)[:_EVIDENCE_MAX],
            reproduce="cd frontend && pnpm audit --json --audit-level=high",
            fix_hint=f"Update or remove dependency: {module_name}.",
        ))
    return findings


def parse_govulncheck(raw: str, scope: str) -> list[Finding]:
    findings: list[Finding] = []
    for line in raw.splitlines():
        line = line.strip()
        if not line:
            continue
        try:
            doc = json.loads(line)
        except json.JSONDecodeError:
            continue
        if not doc.get("osv"):
            continue
        finding = doc.get("finding") or {}
        trace = finding.get("trace") or []
        file = None
        line_no = None
        if trace:
            entry = trace[0] if isinstance(trace[0], dict) else {}
            file = entry.get("file")
            pos = entry.get("pos") or {}
            line_no = pos.get("line")
        findings.append(_finding_from_tool(
            tool="govulncheck", scope=scope,
            severity="high", category="vuln",
            rule=finding.get("osv"),
            file=file, line=line_no, column=None,
            message=doc.get("message") or "vulnerability found",
            evidence=line,
            reproduce=f"cd {scope} && bin/govulncheck ./...",
            fix_hint=f"See {finding.get('osv')} for remediation.",
        ))
    return findings


# ---------------------------------------------------------------------------
# go.work drift (R-RVH-007)
# ---------------------------------------------------------------------------


_GO_DIRECTIVE_RE = re.compile(r"^\s*go\s+(\d+\.\d+(?:\.\d+)?)\s*$", re.MULTILINE)


def _go_mod_version(go_mod_text: str) -> str | None:
    for line in go_mod_text.splitlines():
        line = line.strip()
        if line.startswith("go "):
            parts = line.split()
            if len(parts) >= 2:
                return parts[1]
    return None


def _go_work_use_paths(go_work_text: str) -> list[str]:
    """Return list of relative paths from a `go.work` `use (...)` block."""
    paths: list[str] = []
    in_use = False
    for line in go_work_text.splitlines():
        stripped = line.strip()
        if stripped.startswith("use ("):
            in_use = True
            continue
        if in_use and stripped == ")":
            in_use = False
            continue
        if in_use and stripped:
            # Strip leading ./ so scope is `backend/agent`, not `./backend/agent`.
            paths.append(stripped.removeprefix("./"))
    return paths


def check_go_work_drift(repo_root: Path) -> list[Finding]:
    """R-RVH-007: emit a finding per module whose `go` directive differs from `go.work`."""
    go_work = repo_root / "go.work"
    if not go_work.exists():
        return []
    text = go_work.read_text(encoding="utf-8")
    m = _GO_DIRECTIVE_RE.search(text)
    if not m:
        return []
    work_version = m.group(1)
    findings: list[Finding] = []
    for rel in _go_work_use_paths(text):
        mod_dir = repo_root / rel
        gm = mod_dir / "go.mod"
        if not gm.exists():
            continue
        mod_version = _go_mod_version(gm.read_text(encoding="utf-8"))
        if not mod_version or mod_version == work_version:
            continue
        msg = f"go.work says {work_version}; {rel}/go.mod says {mod_version}"
        findings.append(_finding_from_tool(
            tool="go", scope=rel, severity="medium",
            category="go.work-mismatch",
            rule=None, file="go.work", line=None, column=None,
            message=msg, evidence=None,
            reproduce=f"go work edit -json | jq -r '.Go'  # expect {work_version}; "
                      f"cat {rel}/go.mod",
            fix_hint=f"Bump {rel}/go.mod to `go {work_version}` or downgrade go.work.",
        ))
    return findings


# ---------------------------------------------------------------------------
# Check runner (R-RVH-001 run-all-then-aggregate; R-RVH-006 opt-in)
# ---------------------------------------------------------------------------


def run_checks(
    cfg_path: str | Path,
    *,
    allowlist: list[tuple[str, str]],
    include_integration: bool = False,
    include_e2e: bool = False,
    extra_env: dict[str, str] | None = None,
    config_overrides: dict[str, Any] | None = None,
) -> list[CheckResult]:
    """Run every check in cfg_path sequentially; never fail-fast."""
    cfg = load_config(cfg_path)
    timeout_check = int(cfg.get("timeout_check") or 600)
    timeout_e2e = int(cfg.get("timeout_e2e") or 1200)
    cfg_text = Path(cfg_path).read_text(encoding="utf-8")
    cfg_doc = yaml.safe_load(cfg_text)
    # If the YAML has no `checks:` key, fall back to the canonical default
    # matrix so the production harness actually runs against the default
    # config. An explicit empty list (`checks: []`) means "operator wants
    # nothing", which we honour.
    checks_raw = (cfg_doc or {}).get("checks")
    if checks_raw is None:
        checks = default_check_matrix(ROOT)
    else:
        checks = checks_raw
    results: list[CheckResult] = []
    for chk in checks:
        scope = chk.get("scope") or ""
        name = chk.get("name") or ""
        target = chk.get("target") or name
        # Opt-in detection: explicit cfg field wins, else infer from name/target.
        opt_in = chk.get("opt_in")
        if opt_in is None:
            if name == "integration" or target == "test/integration":
                opt_in = "integration"
            elif name == "e2e" or target == "test:e2e":
                opt_in = "e2e"
        if opt_in == "integration" and not include_integration:
            results.append(CheckResult(
                scope=scope, name=name, target=target,
                status="skipped", log="skipped: integration",
            ))
            continue
        if opt_in == "e2e" and not include_e2e:
            results.append(CheckResult(
                scope=scope, name=name, target=target,
                status="skipped", log="skipped: e2e",
            ))
            continue
        # Synthetic check (no command): handled inline.
        if not chk.get("command"):
            results.append(_run_synthetic_check(chk, repo_root=ROOT))
            continue
        # Preflight for tool-missing.
        preflight = _preflight(chk, allowlist=allowlist)
        if preflight is not None:
            results.append(CheckResult(
                scope=scope, name=name, target=target,
                status="tool-missing", findings=[preflight],
                log=f"tool-missing: {preflight.message}",
            ))
            continue
        # Run subprocess.
        cmd = list(chk["command"])
        cwd = chk.get("cwd") or "."
        timeout = timeout_e2e if name == "e2e" else timeout_check
        rc, out, err, status = _run_subprocess(cmd, cwd, timeout, extra_env=extra_env)
        if status == "timeout":
            fid = compute_finding_id(file=None, line=None, rule=name,
                                      message=f"timeout after {timeout}s")
            tf = Finding(
                id=fid, severity="medium", category="timeout",
                tool=name, scope=scope, rule=name,
                file=None, line=None, column=None,
                message=f"check timed out after {timeout}s",
                evidence=None, reproduce=" ".join(cmd),
                fix_hint="Increase --timeout-check or split the work.",
            )
            results.append(CheckResult(
                scope=scope, name=name, target=target,
                status="timeout", findings=[tf],
                log=f"timeout after {timeout}s",
            ))
            continue
        if status == "notfound":
            results.append(CheckResult(
                scope=scope, name=name, target=target,
                status="tool-missing",
                findings=[_tool_missing_finding(
                    scope=scope, name=name, target=target,
                    message=f"binary not on PATH: {cmd[0]}",
                    reproduce=" ".join(cmd), binary_missing=cmd[0],
                )],
                log=f"binary missing: {cmd[0]}",
            ))
            continue
        # Parse output by tool.
        findings = _parse_for_check(name, scope, rc, out, err, cmd)
        if rc == 0 and not findings:
            results.append(CheckResult(
                scope=scope, name=name, target=target,
                status="passed", log="ok",
            ))
            continue
        if rc != 0 and not findings:
            # Tool returned non-zero but emitted nothing we could parse.
            results.append(CheckResult(
                scope=scope, name=name, target=target,
                status="failed",
                log=f"exit {rc}; no parseable findings",
            ))
            continue
        results.append(CheckResult(
            scope=scope, name=name, target=target,
            status="failed" if any(f.severity != "medium" or f.category not in
                                    ("tool-missing", "timeout", "go.work-mismatch")
                                    for f in findings) else "passed",
            findings=findings,
        ))
    return results


def _parse_for_check(
    name: str, scope: str, rc: int | None, out: str, err: str, cmd: list[str],
) -> list[Finding]:
    """Dispatch to the right parser based on check name."""
    if name == "lint" and scope == "frontend":
        return parse_eslint_json(out, scope)
    if name == "typecheck":
        return parse_tsc(out, scope)
    if name == "format":
        return parse_prettier(err, rc or 0, scope)
    if name == "test" and scope == "frontend":
        return parse_vitest(out, scope)
    if name == "eslint-rule-test":
        return parse_node_test(out, scope)
    if name == "e2e":
        return parse_playwright(out, scope)
    if name == "vuln-check" and scope == "frontend":
        return parse_pnpm_audit(out, scope)
    if name == "vuln-check":
        return parse_govulncheck(out, scope)
    if name in ("test", "test/cover", "integration"):
        return _parse_go_test(out, scope, name)
    # Generic fallback: parse stdout/stderr line-by-line for FAIL patterns.
    return _parse_go_test(out, scope, name)


def _parse_go_test(out: str, scope: str, name: str) -> list[Finding]:
    findings: list[Finding] = []
    for line in out.splitlines():
        if line.startswith("--- FAIL:") or line.startswith("FAIL\t"):
            msg = line.split(":", 1)[1].strip() if ":" in line else line
            findings.append(_finding_from_tool(
                tool="go-test", scope=scope,
                severity="high", category="test",
                rule=name, file=None, line=None, column=None,
                message=msg, evidence=line,
                reproduce=f"cd {scope} && make {name}",
            ))
        elif "DATA RACE" in line:
            findings.append(_finding_from_tool(
                tool="go-test", scope=scope,
                severity="high", category="test",
                rule="data-race", file=None, line=None, column=None,
                message="data race detected",
                evidence=line,
                reproduce=f"cd {scope} && make test",
            ))
    return findings


def _run_synthetic_check(chk: dict[str, Any], *, repo_root: Path) -> CheckResult:
    """Handle non-subprocess checks (currently just go.work-drift)."""
    scope = chk["scope"]
    name = chk["name"]
    target = chk["target"]
    if name == "go.work-drift":
        findings = check_go_work_drift(repo_root)
        if not findings:
            return CheckResult(scope=scope, name=name, target=target,
                                 status="passed", log="ok")
        return CheckResult(scope=scope, name=name, target=target,
                            status="failed", findings=findings,
                            log=f"{len(findings)} drift finding(s)")
    return CheckResult(scope=scope, name=name, target=target,
                        status="internal-error",
                        log=f"unknown synthetic check: {name}")


# ---------------------------------------------------------------------------
# Renderers (R-RVH-009: both outputs from one in-memory list)
# ---------------------------------------------------------------------------


def render_json(findings: list[Finding], *, host: dict[str, str] | None = None) -> str:
    payload = {
        "schema_version": "1",
        "change": "repo-validation-harness",
        "generated_at": datetime.now(timezone.utc).isoformat(),
        "host": host or {},
        "summary": _summary(findings),
        "findings": [finding_to_jsonable(f) for f in findings],
    }
    return json.dumps(payload, indent=2, ensure_ascii=False, sort_keys=False) + "\n"


def render_markdown(findings: list[Finding], *, host: dict[str, str] | None = None) -> str:
    summary = _summary(findings)
    lines: list[str] = []
    lines.append("# Validation report")
    lines.append("")
    lines.append(f"- generated_at: {datetime.now(timezone.utc).isoformat()}")
    if host:
        lines.append("- host:")
        for k, v in sorted(host.items()):
            lines.append(f"  - {k}: {v}")
    lines.append(f"- total findings: {summary['total']}")
    for sev, n in sorted(summary["by_severity"].items(), key=lambda kv: -kv[1]):
        if n:
            lines.append(f"  - {sev}: {n}")
    lines.append("")
    if not findings:
        lines.append("No findings.")
        return "\n".join(lines) + "\n"
    # Group by scope, then severity.
    by_scope: dict[str, list[Finding]] = {}
    for f in findings:
        by_scope.setdefault(f.scope, []).append(f)
    for scope in sorted(by_scope):
        lines.append(f"## {scope}")
        lines.append("")
        sev_order = ["critical", "high", "medium", "low", "info"]
        items = by_scope[scope]
        items_sorted = sorted(
            items,
            key=lambda f: (sev_order.index(map_severity(f.severity))
                            if map_severity(f.severity) in sev_order else 99,
                            f.id),
        )
        for f in items_sorted:
            lines.append(f"### {f.id} — {f.severity} / {f.category} / {f.tool}")
            if f.rule:
                lines.append(f"- rule: `{f.rule}`")
            if f.file:
                loc = f.file
                if f.line:
                    loc += f":{f.line}"
                    if f.column:
                        loc += f":{f.column}"
                lines.append(f"- location: `{loc}`")
            lines.append(f"- message: {f.message}")
            if f.fix_hint:
                lines.append(f"- fix_hint: {f.fix_hint}")
            lines.append(f"- reproduce: `{f.reproduce}`")
            lines.append("")
    return "\n".join(lines) + "\n"


def _summary(findings: list[Finding]) -> dict[str, Any]:
    by_sev: dict[str, int] = {}
    for f in findings:
        bucket = map_severity(f.severity)
        by_sev[bucket] = by_sev.get(bucket, 0) + 1
    return {"total": len(findings), "by_severity": dict(sorted(by_sev.items()))}


# ---------------------------------------------------------------------------
# Top-level runner (R-RVH-011: output-dir, deterministic, no TTY)
# ---------------------------------------------------------------------------


def _detect_host() -> dict[str, str]:
    """Best-effort host metadata (no TTY). Missing tools become empty."""
    out: dict[str, str] = {}
    for name, args in [
        ("go", ["go", "version"]),
        ("node", ["node", "--version"]),
        ("pnpm", ["pnpm", "--version"]),
        ("make", ["make", "--version"]),
    ]:
        try:
            r = subprocess.run(args, capture_output=True, text=True, timeout=5)
            out[name] = (r.stdout or r.stderr or "").splitlines()[0].strip() if r.returncode == 0 else ""
        except (FileNotFoundError, subprocess.TimeoutExpired, OSError):
            out[name] = ""
    return out


def run_full(
    cfg_path: str | Path,
    *,
    allowlist: list[tuple[str, str]],
    output_dir: Path,
    include_integration: bool = False,
    include_e2e: bool = False,
    extra_env: dict[str, str] | None = None,
) -> int:
    """Run all checks, write reports, return exit code."""
    output_dir = Path(output_dir)
    try:
        output_dir.mkdir(parents=True, exist_ok=True)
        cfg = load_config(cfg_path)
        results = run_checks(
            cfg_path, allowlist=allowlist,
            include_integration=include_integration, include_e2e=include_e2e,
            extra_env=extra_env,
        )
        findings = [f for r in results for f in r.findings]
        host = _detect_host()
        (output_dir / "validation-report.json").write_text(
            render_json(findings, host=host), encoding="utf-8"
        )
        (output_dir / "validation-report.md").write_text(
            render_markdown(findings, host=host), encoding="utf-8"
        )
        return compute_exit(findings, cfg["fail_on"])
    except HarnessConfigError as e:
        sys.stderr.write(f"config error: {e}\n")
        return 2
    except HarnessInternalError as e:
        sys.stderr.write(f"internal error: {e}\n")
        return 2
    except OSError as e:
        sys.stderr.write(f"output error: {e}\n")
        return 2


# ---------------------------------------------------------------------------
# CLI (argparse; R-RVH-011 S-3: never call isatty)
# ---------------------------------------------------------------------------


def _build_argparser() -> argparse.ArgumentParser:
    p = argparse.ArgumentParser(
        prog="validate",
        description="cachicamas repo validation harness",
    )
    p.add_argument("--output-dir", default=".", type=Path,
                   help="where to write validation-report.{json,md} (default: .)")
    p.add_argument("--include-integration", action="store_true",
                   help="run make test/integration (boots Postgres)")
    p.add_argument("--include-e2e", action="store_true",
                   help="run pnpm test:e2e (needs compose stack)")
    p.add_argument("--fail-on",
                   choices=sorted(_THRESHOLD_ORDER.keys()),
                   default="high",
                   help="failure threshold; default: high")
    p.add_argument("--timeout-check", type=int, default=600,
                   help="per-check timeout in seconds (default 600)")
    p.add_argument("--timeout-e2e", type=int, default=1200,
                   help="e2e timeout in seconds (default 1200)")
    p.add_argument("--config", default=str(DEFAULT_CONFIG), type=Path,
                   help=f"config YAML (default: {DEFAULT_CONFIG})")
    p.add_argument("--quiet", action="store_true",
                   help="suppress per-check progress")
    return p


def main(argv: list[str] | None = None) -> int:
    args = _build_argparser().parse_args(argv)
    # Load config (HarnessConfigError -> exit 2).
    try:
        cfg = load_config(args.config)
    except HarnessConfigError as e:
        sys.stderr.write(f"config error: {e}\n")
        return 2
    # CLI flags override config.
    fail_on = args.fail_on if args.fail_on else cfg["fail_on"]
    timeout_check = args.timeout_check if args.timeout_check != 600 else cfg["timeout_check"]
    timeout_e2e = args.timeout_e2e if args.timeout_e2e != 1200 else cfg["timeout_e2e"]
    # Persist overrides into a per-invocation config file the runner can load.
    with tempfile.NamedTemporaryFile("w", suffix=".yaml", delete=False) as f:
        yaml.safe_dump({
            "allowlist": cfg["allowlist"],
            "fail_on": fail_on,
            "timeout_check": timeout_check,
            "timeout_e2e": timeout_e2e,
        }, f)
        cfg_override_path = f.name
    # Run.
    allowlist = load_default_allowlist(args.config)
    exit_code = run_full(
        cfg_override_path,
        allowlist=allowlist,
        output_dir=args.output_dir,
        include_integration=args.include_integration,
        include_e2e=args.include_e2e,
    )
    return exit_code


if __name__ == "__main__":
    sys.exit(main())