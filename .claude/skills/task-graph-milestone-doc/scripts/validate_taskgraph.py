#!/usr/bin/env python3
"""Mechanical validator for task-graph milestone documents (ADR 0007 DAG contract).

Checks a milestone document for the structural invariants that a reviewer
should never have to find by hand:

  1. Heading grammar   — `## Wave <id> — <name>`, `### XX-NN — <title>`,
                         `#### XX-NN.p — <title>` (nodes carry a backticked
                         `[type]` tag; heading level must match id depth).
                         One id prefix per document; a heading that carries an
                         id-like token but fails the grammar is an error, not
                         silently skipped.
  2. Unique ids        — no id declared twice.
  3. Edge resolution   — every bare id inside a `Depends on:` / `Blocks:`
                         field resolves to a declared id, or is prefixed by
                         another document's prefix (cross-doc edge, reported
                         as info).
  4. Acyclicity        — the dependency DAG (Depends on + reversed Blocks)
                         has no cycle; a failing topological sort names the
                         edges of the cycle.
  5. Nesting depth     — nodes go at most three levels below a milestone
                         (XX-NN.p.q.r is the floor; XX-NN.p.q.r.s is an error).
  6. Node grammar      — a node is compound or leaf, never both: a node with
                         children must be `[compound]`; a `[compound]` must
                         have children.
  7. Gherkin presence  — every `[leaf]` body contains all three capitalized
                         keywords Given, When, and Then (v2 profile only).
                         Guard bite scenarios are an authoring rule the script
                         does not check.
  8. Wave DAG render   — every wave opens with a ```mermaid block before its
                         first milestone heading (v2 profile only).

Usage:
    validate_taskgraph.py <document.md> [--profile v1|v2] [--quiet]
    validate_taskgraph.py --self-test

Exit codes: 0 = no errors (warnings allowed), 1 = errors, 2 = usage/parse failure.
Profile v2 (default) enforces all checks; v1 skips Gherkin and per-wave mermaid,
so shipped pre-Gherkin documents (docs 0002-0004) can still be audited.
"""

from __future__ import annotations

import re
import sys
from dataclasses import dataclass, field

WAVE_RE = re.compile(r"^##\s+Wave\s+(?P<id>[0-9A-Za-z]+)\s+—\s+(?P<name>.+)$")
MILESTONE_RE = re.compile(r"^###\s+(?P<id>[A-Z]{2}-\d{2})\s+—\s+(?P<title>.+)$")
NODE_RE = re.compile(
    r"^#{4,6}\s+(?P<id>[A-Z]{2}-\d{2}(?:\.\d+)+)\s+—\s+(?P<title>.+)$"
)
TYPE_TAG_RE = re.compile(r"`\[(compound|leaf|guard|decision|mechanical)\]`")
EDGE_MARKER_RE = re.compile(
    r"\*\*\s*(Depends on|Blocks|Parallel with)\s*[:.]?\s*\*\*[:.]?", re.IGNORECASE
)
PARALLEL_CUT_RE = re.compile(r"parallel with", re.IGNORECASE)
ID_TOKEN_RE = re.compile(r"\b([A-Z]{2}-\d{2}(?:\.\d+)*)\b")


def extract_edges(line: str) -> tuple[list[str], list[str]]:
    """Return (depends_on_ids, blocks_ids) from one field line.

    A line may hold several bolded segments (`**Depends on:** … **Blocks:** …`);
    ids belong to the nearest preceding marker, and anything after an unbolded
    "parallel with" inside a segment is prose, not an edge (ADR 0007 D1).
    """
    parts = EDGE_MARKER_RE.split(line)
    depends: list[str] = []
    blocks: list[str] = []
    for kind, segment in zip(parts[1::2], parts[2::2]):
        segment = PARALLEL_CUT_RE.split(segment)[0]
        ids = ID_TOKEN_RE.findall(segment)
        key = kind.lower()
        if key == "depends on":
            depends.extend(ids)
        elif key == "blocks":
            blocks.extend(ids)
    return depends, blocks
# A scenario needs all three capitalized keywords — a lowercase "when" in prose is not Gherkin.
GHERKIN_KEYWORDS = (re.compile(r"\bGiven\b"), re.compile(r"\bWhen\b"), re.compile(r"\bThen\b"))
MERMAID_RE = re.compile(r"^```mermaid\s*$")
UNBOLD_FIELD_RE = re.compile(r"^\s*[-*]?\s*(Depends on|Blocks)\s*:", re.IGNORECASE)


@dataclass
class Section:
    id: str
    title: str
    kind: str  # wave | milestone | node
    line: int
    heading_level: int
    type_tag: str | None = None
    wave: str | None = None
    body: list[str] = field(default_factory=list)
    depends_on: list[str] = field(default_factory=list)
    blocks: list[str] = field(default_factory=list)
    children: list[str] = field(default_factory=list)


@dataclass
class Report:
    errors: list[str] = field(default_factory=list)
    warnings: list[str] = field(default_factory=list)
    infos: list[str] = field(default_factory=list)

    def error(self, line: int, msg: str) -> None:
        self.errors.append(f"ERROR line {line}: {msg}")

    def warn(self, line: int, msg: str) -> None:
        self.warnings.append(f"WARN  line {line}: {msg}")

    def info(self, line: int, msg: str) -> None:
        self.infos.append(f"INFO  line {line}: {msg}")


def parse(lines: list[str], report: Report) -> dict[str, Section]:
    sections: dict[str, Section] = {}
    current: Section | None = None
    current_wave: str | None = None
    in_code = False

    for lineno, raw in enumerate(lines, start=1):
        line = raw.rstrip("\n")
        if line.startswith("```"):
            in_code = not in_code
        if in_code and not line.startswith("```"):
            if current is not None:
                current.body.append(line)
            continue

        wave_m = WAVE_RE.match(line)
        mile_m = MILESTONE_RE.match(line)
        node_m = NODE_RE.match(line)

        if wave_m:
            current_wave = wave_m.group("id")
            current = Section(
                id=f"wave-{current_wave}", title=wave_m.group("name"),
                kind="wave", line=lineno, heading_level=2, wave=current_wave,
            )
            sections[current.id] = current
        elif mile_m or node_m:
            m = mile_m or node_m
            sid = m.group("id")
            kind = "milestone" if mile_m else "node"
            level = len(line) - len(line.lstrip("#"))
            sec = Section(
                id=sid, title=m.group("title"), kind=kind, line=lineno,
                heading_level=level, wave=current_wave,
            )
            if sid in sections:
                report.error(lineno, f"duplicate id {sid} (first at line {sections[sid].line})")
                current = sec  # keep body attribution local; the duplicate stays unregistered
                continue
            expected_level = 3 + sid.count(".")
            if level != expected_level and expected_level <= 6:
                report.error(lineno, f"{sid} sits at heading level {level}; its depth requires level {expected_level} (ADR 0007 D2)")
            tag = TYPE_TAG_RE.search(m.group("title"))
            if tag:
                sec.type_tag = tag.group(1)
            sections[sid] = sec
            if kind == "node":
                parent_id = sid.rsplit(".", 1)[0]
                parent = sections.get(parent_id)
                if parent is None:
                    report.error(lineno, f"node {sid} has no declared parent {parent_id}")
                else:
                    parent.children.append(sid)
            current = sec
        elif line.lstrip().startswith("#") and ID_TOKEN_RE.search(line):
            report.error(lineno, f"heading carries an id-like token but does not match the heading grammar: {line.strip()!r}")
        elif current is not None:
            current.body.append(line)
            if current.kind != "wave":
                depends, blocks = extract_edges(line)
                if not (depends or blocks) and UNBOLD_FIELD_RE.match(line) and ID_TOKEN_RE.search(line):
                    report.warn(lineno, f"unbolded edge field ignored (contract shape is `- **Depends on:** …`): {line.strip()!r}")
                current.depends_on.extend(depends)
                current.blocks.extend(blocks)

    return sections


def check_prefix(sections: dict[str, Section], report: Report) -> str | None:
    prefixes = {s.id[:2]: s for s in sections.values() if s.kind in ("milestone", "node")}
    if len(prefixes) > 1:
        detail = ", ".join(sorted(prefixes))
        report.error(1, f"more than one id prefix declared in headings: {detail}")
        return None
    return next(iter(prefixes), None)


def check_edges(sections: dict[str, Section], prefix: str | None, report: Report) -> list[tuple[str, str]]:
    """Return the internal edge list as (from, to) meaning `from` depends on `to`."""
    edges: list[tuple[str, str]] = []
    for sec in sections.values():
        if sec.kind == "wave":
            continue
        for target in sec.depends_on + sec.blocks:
            if target.count(".") > 3:
                report.error(sec.line, f"edge id {target} exceeds three dotted segments (ADR 0007 D2 item 4)")
        for target in sec.depends_on:
            if target in sections:
                edges.append((sec.id, target))
            elif prefix and target.startswith(prefix):
                report.error(sec.line, f"{sec.id} depends on undeclared id {target}")
            else:
                report.info(sec.line, f"{sec.id} depends on cross-document id {target}")
        for target in sec.blocks:
            if target in sections:
                edges.append((target, sec.id))
            elif prefix and target.startswith(prefix):
                report.error(sec.line, f"{sec.id} blocks undeclared id {target}")
            else:
                report.info(sec.line, f"{sec.id} blocks cross-document id {target}")
    return edges


def check_acyclic(edges: list[tuple[str, str]], report: Report) -> None:
    adjacency: dict[str, set[str]] = {}
    indegree: dict[str, int] = {}
    nodes: set[str] = set()
    for frm, to in edges:
        nodes.update((frm, to))
        if to not in adjacency.setdefault(frm, set()):
            adjacency[frm].add(to)
            indegree[to] = indegree.get(to, 0) + 1
        indegree.setdefault(frm, indegree.get(frm, 0))

    queue = sorted(n for n in nodes if indegree.get(n, 0) == 0)
    seen = 0
    indeg = dict(indegree)
    while queue:
        node = queue.pop()
        seen += 1
        for nxt in adjacency.get(node, ()):  # noqa: B007
            indeg[nxt] -= 1
            if indeg[nxt] == 0:
                queue.append(nxt)
    if seen != len(nodes):
        cyclic = sorted(n for n in nodes if indeg.get(n, 0) > 0)
        involved = [f"{a}->{b}" for a, b in edges if a in cyclic and b in cyclic]
        report.error(1, f"dependency graph has a cycle among: {', '.join(cyclic)} (edges: {', '.join(involved)})")


def check_depth(sections: dict[str, Section], report: Report) -> None:
    for sec in sections.values():
        if sec.kind != "node":
            continue
        depth = sec.id.count(".")
        if depth > 3:
            report.error(sec.line, f"{sec.id} nests {depth} levels below its milestone; maximum is 3")


def check_node_grammar(sections: dict[str, Section], report: Report) -> None:
    for sec in sections.values():
        if sec.kind != "node":
            continue
        if sec.type_tag is None:
            report.error(sec.line, f"{sec.id} heading carries no backticked `[type]` tag")
            continue
        if sec.children and sec.type_tag != "compound":
            report.error(sec.line, f"{sec.id} is `[{sec.type_tag}]` but has children {sec.children}; a node with children must be `[compound]`")
        if not sec.children and sec.type_tag == "compound":
            report.error(sec.line, f"{sec.id} is `[compound]` but declares no children")


def check_gherkin(sections: dict[str, Section], report: Report) -> None:
    for sec in sections.values():
        if sec.kind == "node" and sec.type_tag == "leaf":
            text = "\n".join(sec.body)
            if not all(kw.search(text) for kw in GHERKIN_KEYWORDS):
                report.error(sec.line, f"leaf {sec.id} has no Given/When/Then scenario in its body (all three capitalized keywords required)")


def check_wave_mermaid(sections: dict[str, Section], lines: list[str], report: Report) -> None:
    """Each wave must render its DAG tree before its first milestone heading."""
    waves = sorted(
        (s for s in sections.values() if s.kind == "wave"), key=lambda s: s.line
    )
    for i, wave in enumerate(waves):
        end = waves[i + 1].line - 1 if i + 1 < len(waves) else len(lines)
        milestones_in_wave = [
            s.line for s in sections.values()
            if s.kind == "milestone" and wave.line < s.line <= end
        ]
        render_deadline = min(milestones_in_wave) if milestones_in_wave else end
        block = lines[wave.line: render_deadline]
        if not any(MERMAID_RE.match(line.rstrip("\n")) for line in block):
            report.error(wave.line, f"wave {wave.wave} does not open with a mermaid DAG tree (required before its first milestone)")


def validate(text: str, profile: str = "v2") -> Report:
    report = Report()
    lines = text.splitlines()
    sections = parse(lines, report)
    if not any(s.kind == "milestone" for s in sections.values()):
        report.error(1, "no milestone headings (`### XX-NN — …`) found")
        return report
    prefix = check_prefix(sections, report)
    edges = check_edges(sections, prefix, report)
    check_acyclic(edges, report)
    check_depth(sections, report)
    check_node_grammar(sections, report)
    if profile == "v2":
        check_gherkin(sections, report)
        check_wave_mermaid(sections, lines, report)
    return report


# --- self-test ------------------------------------------------------------

GOOD_DOC = """
## Wave 0 — Foundations

```mermaid
flowchart TB
  XX01[XX-01]
```

### XX-01 — Lay the foundation

- **SDD change:** `demo-foundation`
- **Depends on:** nothing.

#### XX-01.1 — walking skeleton `[leaf]`

- **Scenario:** Given an empty project, When the skeleton runs, Then it exits zero.
- **Depends on:** nothing.

#### XX-01.2 — widen the path `[compound]`

- **Depends on:** XX-01.1

##### XX-01.2.1 — error path `[leaf]`

- **Scenario:** Given a bad input, When the skeleton runs, Then it reports a typed error.
- **Depends on:** XX-01.1
"""

CYCLIC_DOC = """
## Wave 1 — Loop

```mermaid
flowchart TB
  YY01[YY-01]
```

### YY-01 — First

- **Depends on:** YY-02

### YY-02 — Second

- **Depends on:** YY-01
"""


def self_test() -> int:
    good = validate(GOOD_DOC)
    assert not good.errors, f"good doc should pass, got: {good.errors}"

    cyclic = validate(CYCLIC_DOC)
    assert any("cycle" in e for e in cyclic.errors), f"cycle not detected: {cyclic.errors}"

    no_gherkin = validate(GOOD_DOC.replace("Given an empty project, When the skeleton runs, Then it exits zero.", "add a mutex"))
    assert any("Given/When/Then" in e for e in no_gherkin.errors), "missing Gherkin not detected"

    prose = validate(GOOD_DOC.replace("Given an empty project, When the skeleton runs, Then it exits zero.", "run this check when idle, given time; that is all then."))
    assert any("XX-01.1" in e and "Given/When/Then" in e for e in prose.errors), "lowercase prose accepted as Gherkin"

    typo = validate(GOOD_DOC + "\n#### XX-01.9 - typo hyphen `[leaf]`\n")
    assert any("heading grammar" in e for e in typo.errors), "malformed heading not reported"

    wrong_level = validate(GOOD_DOC.replace("##### XX-01.2.1 — error path `[leaf]`", "#### XX-01.2.1 — error path `[leaf]`"))
    assert any("heading level" in e for e in wrong_level.errors), "heading level vs depth not enforced"

    late_render = validate(GOOD_DOC.replace("```mermaid\nflowchart TB\n  XX01[XX-01]\n```\n\n### XX-01", "### XX-01") + "\n```mermaid\nflowchart TB\n  X[after]\n```\n")
    assert any("does not open with" in e for e in late_render.errors), "late mermaid accepted as wave render"
    v1 = validate(GOOD_DOC.replace("Given an empty project, When the skeleton runs, Then it exits zero.", "add a mutex"), profile="v1")
    assert not v1.errors, f"v1 profile must not enforce Gherkin: {v1.errors}"

    too_deep = validate(GOOD_DOC + "\n###### XX-01.2.1.1.1 — too deep `[leaf]`\n\n- **Scenario:** Given, When, Then ok.\n- **Depends on:** XX-01.1\n")
    assert any("maximum is 3" in e for e in too_deep.errors), f"depth not enforced: {too_deep.errors}"

    no_mermaid = validate(GOOD_DOC.replace("```mermaid\nflowchart TB\n  XX01[XX-01]\n```", ""))
    assert any("mermaid" in e for e in no_mermaid.errors), "missing wave mermaid not detected"

    bad_type = validate(GOOD_DOC.replace("#### XX-01.2 — widen the path `[compound]`", "#### XX-01.2 — widen the path `[leaf]`"))
    assert any("must be `[compound]`" in e for e in bad_type.errors), "leaf-with-children not detected"

    print("self-test: all assertions passed")
    return 0


def main(argv: list[str]) -> int:
    import argparse

    parser = argparse.ArgumentParser(description=__doc__, formatter_class=argparse.RawDescriptionHelpFormatter)
    parser.add_argument("document", nargs="?", help="milestone document to validate")
    parser.add_argument("--profile", choices=("v1", "v2"), default="v2")
    parser.add_argument("--quiet", action="store_true", help="suppress INFO lines")
    parser.add_argument("--self-test", action="store_true", dest="self_test")
    opts = parser.parse_args(argv)

    if opts.self_test:
        return self_test()
    if not opts.document:
        parser.print_usage()
        return 2
    try:
        with open(opts.document, encoding="utf-8") as fh:
            text = fh.read()
    except OSError as exc:
        print(f"cannot read {opts.document}: {exc}")
        return 2
    report = validate(text, profile=opts.profile)
    shown = report.errors + report.warnings + ([] if opts.quiet else report.infos)
    for line in shown:
        print(line)
    print(f"\n{len(report.errors)} error(s), {len(report.warnings)} warning(s), profile {opts.profile}")
    return 1 if report.errors else 0


if __name__ == "__main__":
    sys.exit(main(sys.argv[1:]))
