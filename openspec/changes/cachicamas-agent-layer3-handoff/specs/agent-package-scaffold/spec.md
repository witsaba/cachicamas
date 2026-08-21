# Delta — `agent-package-scaffold` (AG-23)

> **Change**: `cachicamas-agent-layer3-handoff` · **AG-23** · Target: `openspec/specs/agent-package-scaffold/spec.md`
> **Ops**: MODIFIED `R-AGP-003` — AG-23 adds two package trees to the module, so the guard's single scan pattern becomes a **per-check pattern set**, its **test** allowlist gains the two new prefixes, and a **fifth** check lands. Six scenarios (`S-AGP-039`…`S-AGP-044`) are appended. Every existing scenario is reproduced verbatim and unchanged in claim.
> **Decision**: design **AD-2** (which binds proposal `D-3`), binding.

## Why the guard is extended and not cloned

`R-AGP-003`'s allowlist is **deny-by-default**: a path that is neither standard library nor an explicitly
admitted entry fails **even when no rule names it**. The two new trees are non-standard-library package
paths inside this module, so the test closure fails the instant they exist. That is the guard working.

The alternative — a second, self-contained guard inside each new tree — would leave this requirement
byte-unchanged, but it duplicates the allowlist and creates exactly the drift surface a single
deny-by-default list exists to prevent. The charter's own acceptance sentence says vendor-import absence is
proven by *"**the** guard mechanism"*, definite article, and the same milestone one layer down wrote no new
guard.

## The production closure is deliberately NOT widened, and this is the property the placement exists to keep

The production allowlist admits this module's Layer 2 package **as a prefix**, and the guard's prefix match
is path-boundary aware. A kit or a proof **nested** under Layer 2's own tree would therefore be silently
admitted into Layer 2's **production** closure, destroying the separate-closure property `S-AGP-025` and
`S-AGP-026` were built to prove. The kit and the proof are consequently **siblings**, not subpackages, and
this delta widens only the **test** allowlist. `S-AGP-025` and `S-AGP-026` are reproduced unchanged and
`S-AGP-040` re-proves the production half against the new trees specifically.

## Not modified, and why

| Element | Verdict |
|---|---|
| The **production**-closure paragraph and `allowedProductionPrefixes` | **Byte-unchanged.** Checks 1, 3 and 4 keep scanning Layer 2's own tree alone |
| The forbidden-prefix table and its before-the-allowlist ordering | **Byte-unchanged.** AG-23 admits none of its rows and `S-AGP-035` still bites |
| The exact-path standard-library table, its direct-import-edge mechanism, and its vetted-importer lists | **Byte-unchanged.** AG-23 adds no dependency, so no vetted set moves |
| The AG-22 direct-import family scan over Layer 2 **production** files (check 4) | **Byte-unchanged.** Check 5 is a **separate** scan over two **different** trees and does not widen, weaken or replace it |
| `R-AGP-004` (fresh closure re-measurement) | **Not modified.** AG-23 adds no third-party path, so no closure is re-measured |
| `S-AGM-030` — *"`backend/agent/src/` contains exactly two entries"* (`agent-module-scaffold/spec.md:104`) | **Not falsified — checked, not assumed.** Its Given is *"the repository **at AI-00's merge**"*, a historical snapshot. The directory already holds four entries today, so AG-23 is not the milestone that would falsify it, and the same reasoning covers `S-APC-059`, whose Given is scoped to **its own** merged change |

## Header maintenance obligation at promotion

`sdd-archive` MUST add **`S-AGP-039`** … **`S-AGP-044`** wherever this spec's scenario identifiers are
enumerated, as a **range and never a total**. The Acceptance-criteria line already states a range and needs
no edit; no existing `S-AGP-` identifier is renumbered.

## MODIFIED Requirements

### R-AGP-003 — Forward import guard: Layer 2 import purity over both closures

`backend/agent/src/agent/` MUST carry an executable guard that fails when any package of Layer 2 — including its test packages and their transitive dependencies — depends on anything outside an explicit allowlist.

The guard MUST enumerate dependencies via the Go toolchain's own dependency listing with test dependencies included, MUST derive the standard-library set from the toolchain's own classification rather than a path-shape heuristic, and MUST normalize the synthesized test-variant entries the toolchain emits so that neither a test-variant suffix nor a test-binary path is mistaken for a real import. The guard MUST fail loudly rather than pass vacuously when the listing returns zero packages.

The allowlist MUST be **deny-by-default**: a path that is neither standard library nor an explicitly admitted entry fails **even when no rule names it**.

The **production** closure MUST admit only: the standard library; `github.com/cachicamas/backend/agent/src/ai` and its transitive closure as freshly measured under `R-AGP-004`; and such observability API paths as [ADR 0005 § D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary) authorises for Layer 2 **and** design explicitly selects — no others. Every third-party entry MUST carry, in the guard's own source, the ADR clause authorising it; an entry admitted only as the forced closure of an authorised path MUST record that reasoning in place; and an entry that the fresh measurement does not show in the actual closure MUST record why an as-yet-unused path is admitted. The **test** closure MUST additionally admit `github.com/cachicamas/backend/agent/src/agenttest` and its subpackages, and MUST be asserted **separately** from the production closure.

**The observability clause is exercised at AG-22, and the shape of its entries is pinned here rather than left to the guard's source.** Every admitted observability entry MUST be an OpenTelemetry **API** path or the forced closure of one. Each MUST carry exactly one of two citation shapes, and MUST NOT carry the other:

1. an entry that **is** a § D3 table row cites that row; or
2. an entry that is **not** a § D3 table row — including an entry that is not an OpenTelemetry module at all — cites the forced-closure clause of this requirement **together with the measurement that forced it**, and MUST NOT cite a § D3 row it does not have.

An entry whose concrete measured path carries a version suffix MUST be admitted as a **prefix**, never as the exact measured path, so a later dependency bump that moves the version does not fail the guard on a path nobody changed. An observability path that § D3 permits but no design selects — the ecosystem's root global-getter package and its metric module — MUST NOT be admitted, and the declination MUST be recorded in place with its reason.

The allowlist amendment MUST land in the **same commit** as the first production import that requires it. A commit that adds the import without the entry, or the entry without the import, is a violation of this requirement.

The guard MUST maintain **two separate enforcement tables**. The first is a **forbidden-prefix table** for non-standard-library denials, matched **before** the allowlist, covering at minimum: the vendor adapter subtree `github.com/cachicamas/backend/agent/src/ai/openaicompat`; `…/agent/src/coding`; `…/agent/src/cmd`; `github.com/cachicamas/backend/database_administrator`; `github.com/cachicamas/backend/workspace_syncer`; `go.opentelemetry.io/otel/sdk`; `go.opentelemetry.io/otel/exporters`; `go.opentelemetry.io/contrib/bridges/otelslog`. The before-the-allowlist ordering of this table MUST be stated in source with its reason: the vendor subtree lives under an otherwise-admitted Layer 1 prefix, so an allowlist-first pass would admit it. This ordering applies only to the forbidden-prefix table.

Standard-library network/filesystem denial (`net`, `net/http`, `os`, `io/fs`) MUST NOT be attempted via the forbidden-prefix table — the dependency lister filters the standard library out before either the forbidden-prefix table or the allowlist runs, so a row there is unreachable dead code. The guard MUST instead carry a **separate, exact-path table** matched over a dependency listing that includes the standard library, covering exactly `net`, `net/http`, `os`, and `io/fs`. Each entry in this exact-path table MUST be annotated with the rule it enforces, mirroring `ai_test`'s `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` (AI-10.4).

**The exact-path table's matching mechanism (AG-22 correction).** The table MUST NOT be matched by bare presence in the dependency listing once the production allowlist admits an observability API path: `go.opentelemetry.io/otel/trace` legitimately, transitively touches `os` (and, through it, `io/fs` and `syscall`) via an unreachable auto-instrumentation code path, and a bare presence test fails that legitimate case (this requirement's own forced-closure clause already anticipates exactly this shape). The guard MUST instead test **direct import edges**: for each exact-path table entry present in the listing, every package that imports it **directly** MUST appear on that entry's own closed, evidence-cited vetted-importer list, or the guard FAILS naming the unvetted importer and the vetted set. The vetted-importer list MUST NOT exempt an importer merely because it is standard library — an unvetted standard-library importer chosen directly by a Layer 2 production source file (for example `os/exec` or `crypto/tls`) MUST fail exactly as an unvetted third-party importer would. Every vetted entry MUST record, in the guard's own source, either which admitted third-party path forces it (with the forced-closure clause and the measurement, as above) or why it is otherwise inert — a Go-internal package no code outside the standard library can import directly, or a package whose own admitted purpose does not expose the forbidden capability to its callers (for example `fmt`'s internal, non-exposed use of `os.Stdout`).

**The direct-import family scan (AG-22 correction, a fourth check).** Independent of the exact-path table's transitive mechanism above, the guard MUST additionally scan Layer 2's own production source files — directly inside `backend/agent/src/agent/`, no subpackage, no test file — for an import whose path is itself exactly `os`, `net`, `syscall` or `io/fs`, or has one of `os/`, `net/`, `syscall/` as a prefix, and MUST fail naming the offending file, path and family the instant one is found. This check needs no dependency-graph traversal: a package whose own name names a forbidden capability is denied the moment Layer 2 chooses to import it, regardless of which further path its own closure would transitively reach.

**The scan pattern is per check, not per guard (AG-23 extension).** AG-23 adds two **sibling** package trees to this module — the packaged test kit and the consumer proof — that are neither Layer 2 nor Layer 1. The guard's single scan pattern therefore becomes a **pattern set applied per check**, and the split is normative rather than an implementation detail:

| Check | Scanned tree(s) | Allowlist |
|---|---|---|
| 1 — production closure | Layer 2's own tree **alone** | `allowedProductionPrefixes`, **unwidened** |
| 2 — test closure | Layer 2's own tree **and** each new sibling tree | the test allowlist, extended by the two new sibling prefixes |
| 3 — network/filesystem closure | Layer 2's own tree **alone** | unchanged |
| 4 — direct-import family scan | Layer 2's **production** files | unchanged |
| 5 — source scan over the new trees (new) | every file directly inside each new sibling tree, **test files included** | the process, network, system-call and filesystem families **plus the wall clock** |

**The production closure MUST NOT be widened to admit either new tree**, and neither tree may be placed
where an existing production allowlist prefix would admit it. Layer 2's own prefix admits any tree nested
beneath it, so a kit or proof nested under Layer 2 would be silently admitted into the **production**
closure — destroying the separate-closure property `S-AGP-025` and `S-AGP-026` exist to prove. Sibling
placement is therefore a requirement of this guard's design, not a naming preference.

**Check 2's vacuity floor MUST be per pattern, never per set.** The guard's existing zero-packages fatal
MUST run **inside** the loop, once per pattern, and MUST name the pattern that resolved nothing. A mistyped
new pattern resolves zero packages; with a per-set floor the set as a whole is still non-empty and the
mistyped pattern **passes vacuously**, admitting an unswept tree while the guard reports green.

**Check 5 MUST carry its own per-tree vacuity floor**: finding zero files in a tree MUST fail naming that
directory, so a renamed or moved tree cannot silently drop out of the scan.

**Check 5's blind spot MUST be stated rather than left implicit.** Check 5 lists no dependencies — it is a
**zero-hop** scan of the two trees' own source files — so a dependency filter that hides the standard
library cannot blind it. Standard-library-mediated reach is closed **elsewhere and by a different
mechanism**: check 2 pins the two trees' non-standard-library closure to exactly the allowlist, and every
admitted member carries its own I/O guard. The residual floor is the test framework's own unavoidable reach,
which is the same floor check 3's own comment already records; it is **restated, not hidden**. A closure
listing filtered to exclude the standard library MUST NOT be offered as evidence for the zero-I/O claim.

This guard closes on **bite proof**, not on green. An amendment to its allowlist MUST re-prove that a denied neighbour still bites: widening a prefix is exactly the edit that can silence a table nobody edited. A vetted-importer amendment MUST re-prove, with a real scratch import (never `-overlay`, which does not reach an `os/exec`-based guard), that an UNVETTED standard-library importer of the same forbidden path still fails — the failure shape this requirement exists to keep provable. **A pattern-set or allowlist amendment MUST likewise re-prove, per new pattern and per new tree, that the guard still bites** — the per-pattern floor above is what makes that re-proof possible rather than a matter of trust.

(Previously, at AG-23: the requirement described a single scan pattern shared by every check, a test-closure allowlist naming only the Layer 1 substrate, four checks, and a zero-packages fatal whose per-pattern scope was unstated because there was only ever one pattern. AG-23 is the first milestone to add sibling package trees to the module, which the deny-by-default allowlist rejects by construction, and the first for which a mistyped pattern could pass vacuously. Previously, at AG-22: identical normative text without the five AG-22 paragraphs; the observability clause was pre-authorising but unexercised.)

#### Scenarios

- **S-AGP-020** — Given the repository with no violation present, when `cd backend/agent && go test ./src/agent/...` runs, then the forward guard passes.
- **S-AGP-021** — Given the guard's source, when its dependency listing is read, then it requests both transitive dependencies and test dependencies, uses a fully-qualified package pattern rather than a relative one, filters the standard library by the toolchain's own classification, normalizes the synthesized test-variant and test-binary entries, and fails the test when the listing returns zero packages.
- **S-AGP-022** — Given the guard's source, when the matching order and its scope are read, then the forbidden-prefix table is checked before the allowlist AND a comment states that this ordering applies only to the forbidden-prefix table because the vendor adapter subtree lives under an otherwise-admitted Layer 1 prefix, so an allowlist-first pass would admit it.
- **S-AGP-023** — Given the guard's source, when its two denial tables are read, then the forbidden-prefix table names every non-standard-library prefix listed in this requirement, each annotated with the rule it enforces, AND the separate exact-path table covers exactly `net`, `net/http`, `os`, and `io/fs`, matched over a dependency listing that includes the standard library, each entry annotated with the rule it enforces.
- **S-AGP-024** — Given the guard's source, when its allowlist is read, then every non-standard-library, non-own-module entry carries the ADR clause authorising it for Layer 2, no entry names an observability SDK, exporter or `otelslog` path, and any entry absent from the fresh `R-AGP-004` measurement carries its own recorded reason for being admitted before use.
- **S-AGP-025** — Given the guard, when the production closure and the test closure are asserted, then they are two separate assertions with separate allowlists, and the production assertion does not admit `…/src/agenttest` or its subpackages.
- **S-AGP-026** — Given a test file in `backend/agent/src/agent/` importing `…/src/agenttest` and no production file importing it, when the guard runs over both closures, then the production closure passes without admitting the substrate AND the test closure passes admitting it.
- **S-AGP-027** — **(bite 1)** Given a scratch file in `backend/agent/src/agent/` importing an application-layer path (`…/agent/src/coding/…`), when the forward guard runs, then it FAILS and its message names the violating import path and the rule. Recorded, then removed.
- **S-AGP-028** — **(bite 2)** Given a scratch file in `backend/agent/src/agent/` importing `net/http`, when the forward guard runs, then it FAILS and its message names the violating import path. Recorded, then removed.
- **S-AGP-029** — **(bite 3)** Given a scratch file in `backend/agent/src/agent/` importing the vendor adapter subtree `…/src/ai/openaicompat/…`, when the forward guard runs, then it FAILS naming the violating import path AND the recorded reason is the deny-by-name prefix rule, not a network side effect nor the deny-by-default rule.
- **S-AGP-030** — Given the merged diff of this change, when it is inspected, then no scratch violation file appears in it, `git status` reports a clean tree, and `backend/agent/go.mod` gained no `require` entry.
- **S-AGP-031** — **(AG-22)** Given the guard's source after AG-22, when its production allowlist is read, then every observability entry is an OpenTelemetry API path or the forced closure of one; each entry that **is** a § D3 table row cites that row; and each entry that is **not** a § D3 table row cites this requirement's forced-closure clause together with the measurement that forced it, and cites no § D3 row.
- **S-AGP-032** — **(AG-22)** Given that same allowlist, when the entry covering the semantic-conventions package is read, then it is a **prefix** rather than the exact version-suffixed path the measurement returns, and a comment in place states that a later dependency bump moves that version — so an exact-path entry would fail on a path nobody changed.
- **S-AGP-033** — **(AG-22)** Given that same allowlist, when a reader looks for the ecosystem's root global-getter package and its metric module, then neither is admitted, and a note in place records each declination with its reason: the root package's own measured closure contains an auto-instrumentation SDK path, which `S-AGP-024` forbids an entry from naming, and the metric module is permitted by § D3 but selected by no design.
- **S-AGP-034** — **(AG-22)** Given the single commit that introduces Layer 2's first production OpenTelemetry import, when its diff is inspected, then the allowlist amendment is in that same commit; and given the commit's parent, when the forward guard is run against a tree carrying the import without the amendment, then it FAILS naming the unadmitted path and the deny-by-default rule. That red output is recorded — it is what proves the guard was working rather than absent.
- **S-AGP-035** — **(bite 4, AG-22)** Given the widened allowlist and a scratch production file in `backend/agent/src/agent/` importing a package under `go.opentelemetry.io/otel/sdk`, when the forward guard runs, then it FAILS **on the forbidden-prefix rule**, naming the offending path — proving the widened observability prefixes did not go over-broad and did not shadow the denial table. Recorded, then removed; the merged diff contains no scratch file.
- **S-AGP-036** — **(AG-22 correction)** Given the exact-path table's matching mechanism, when a Layer 2 production source directly imports a standard-library package that is not on that forbidden path's own vetted-importer list (for example `os/exec`, reaching `os`), then the guard FAILS naming the unvetted importer and the vetted set, exactly as an unvetted third-party importer would; and given `go.opentelemetry.io/otel/trace`'s own vetted, evidence-cited reach into `os`, then the guard PASSES unchanged.
- **S-AGP-037** — **(bite 5, AG-22 correction)** Given a scratch production file in `backend/agent/src/agent/` importing, one at a time, `os/exec`, `net/http` or `crypto/tls`, when the forward guard runs, then it FAILS naming the offending path for each. Recorded, then removed; the merged diff contains no scratch file.
- **S-AGP-038** — **(AG-22 correction)** Given the guard's source, when its direct-import family scan over Layer 2's own production files is read, then it denies any import exactly matching, or prefixed by, `os`, `net`, `syscall` or `io/fs`, independent of the exact-path table's own transitive mechanism; and given a scratch production file directly importing `syscall`, when the guard runs, then it FAILS naming the file and the family. Recorded, then removed.
- **S-AGP-039** — **(AG-23)** Given the guard's source after AG-23, when each check's scanned pattern set is read, then checks 1, 3 and 4 scan Layer 2's own tree alone, check 2 scans Layer 2's tree **and** each new sibling tree, and check 5 scans only the new sibling trees — and a comment in place states that checks 1, 3 and 4 are deliberately unwidened so Layer 2 production provably cannot import the kit or the proof.
- **S-AGP-040** — **(AG-23)** Given the merged change, when the **production** closure is asserted, then it passes **without** admitting either new sibling tree; and given a scratch **production** file in `backend/agent/src/agent/` importing either one, when the guard runs, then it FAILS naming the unadmitted path and the deny-by-default rule. Recorded, then removed.
- **S-AGP-041** — **(AG-23)** Given the single commit that introduces each new sibling tree, when its diff is inspected, then the test-allowlist entry admitting that tree is in the **same commit** as the tree; and given that commit with the entry withheld, when the guard runs, then check 2 FAILS naming the unadmitted path. That red output is recorded.
- **S-AGP-042** — **(bite, AG-23)** Given check 2's pattern set with one new pattern replaced by a **mistyped** path that resolves zero packages, when the guard runs, then it FAILS naming **that pattern** — proving the vacuity floor is per pattern rather than per set, and that a mistyped pattern cannot silently drop a tree from the sweep. Recorded, then reverted.
- **S-AGP-043** — **(bite, AG-23)** Given a scratch file planted in each new sibling tree importing, one at a time, a wall-clock package and a process or filesystem package — including once as a **test file**, to prove test files are scanned — when check 5 runs, then it FAILS naming the file and the family for each. Recorded, then removed; the merged diff contains no scratch file.
- **S-AGP-044** — **(AG-23)** Given check 5's source, when its vacuity floor is read, then finding zero files in a tree fails naming that directory; and when its comment is read, then it states that check 5 lists no dependencies and is therefore a zero-hop scan that a standard-library-excluding filter cannot blind, that transitive reach is closed by check 2 plus each admitted member's own guard, and that the test framework's own unavoidable reach is the residual floor.
