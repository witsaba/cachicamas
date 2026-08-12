# Spec delta — `agent-module-scaffold` (amended by AG-03)

> **Change**: `cachicamas-agent-package-scaffold` · **Milestone**: AG-03 (Layer 2, Wave 1)
> **Amends**: `openspec/specs/agent-module-scaffold/spec.md` (AI-00's live spec, last amended 2026-08-08 by AI-37)
> **Promotion discipline**: each `MODIFIED` block below restates its **entire** requirement, including every scenario whose wording did not change, so no landed proof is lost when the archive replaces the requirement in the main spec. No requirement and no scenario not named below changes.

## Why this spec is amended

Creating `backend/agent/src/agent/` falsifies two properties this spec currently asserts:

1. **`S-AGM-035`** asserts unconditionally that `backend/agent/src/agent` does not exist. `R-AGM-004`'s own prose already authorises this change — *"MUST NOT contain `src/agent/` … **until docs 0003 and 0004 create them**"* — so the scenario is amended to state the post-AG-03 invariant while `src/coding/` and `src/cmd/` remain forbidden. This amendment is unconditional.
2. **`S-AGM-041`** asserts that the guard's `go list` pattern *"covers every package of the module"*. Verified against the shipped guard: `layer1Pattern = modulePath + "/..."` (`backend/agent/src/ai/import_boundary_test.go:52`) does scan the whole module, and that is exactly why the recorded self-reference hazard fires (`:82-92`). AG-03's design chooses between **narrowing the scanned roots** — which makes the current wording false — and **exempting the pattern's own members from the prefix match** — which leaves it true. The amendment below is written to hold under **either** mechanism while still asserting the property the scenario exists to protect (complete Layer 1 coverage, no silent narrowing), so a later mechanism choice does not require a second amendment and a later over-narrowing is a spec violation rather than an edit.

`R-AGM-008`'s closure pin is unaffected either way: `TestLayer1_DependencySet_ExactRequiresAndClosure` filters `modulePath`-prefixed entries out of its closure before comparing (`import_boundary_test.go:328-329`), and `src/agent` adds no external dependency. No scenario of `R-AGM-008` changes.

---

## MODIFIED Requirements

### R-AGM-004 — Package and test-package layout

*(MODIFIED 2026-08-12 by AG-03 — restates the entire requirement, including every unchanged scenario, so no landed proof is lost at archive. Previously: `src/agent/` was forbidden unconditionally alongside `src/coding/` and `src/cmd/`; doc 0003's AG-03 is the milestone this requirement already named as the one that creates it.)*

The module MUST contain `src/ai/` as its Layer 1 package. At AI-00 that package carried package documentation stating the layer boundary and the import rule **and nothing else** — no type, no constant, no function; from AI-04 onward it carries the Layer 1 contracts. The module MUST contain `src/agenttest/` as a **direct sibling** of `src/ai/`, holding an external-package test that imports `github.com/cachicamas/backend/agent/src/ai` and compiles.

From AG-03 onward the module MUST contain `src/agent/` as its Layer 2 package, created by doc 0003's AG-03 exactly as this requirement anticipated. Its contract is owned by `agent-package-scaffold`, not by this spec; this requirement asserts only its existence, its position as a direct sibling of `src/ai/` and `src/agenttest/`, and that its creation did not disturb the layout `R-AGM-004` guarantees.

The module MUST NOT contain `src/coding/` or `src/cmd/`, in any form, including as an empty directory or a directory holding only a placeholder file, until doc 0004 creates them. Creating either of them earlier would make the forward guard's forbidden-prefix list untestable, because a prefix that matches an existing package can no longer be proven forbidden **by absence**. `src/agent/` is the first prefix of that list to name a package that now exists; its row therefore MUST remain provable by a deliberate import bite instead (`R-AGM-005`, `S-AGM-041`), and MUST NOT be deleted.

#### Scenarios

- **S-AGM-030** — Given the repository at AI-00's merge, when `backend/agent/src/` is listed, then it contains exactly two entries, `ai` and `agenttest`, and both are directories.
- **S-AGM-031** — Given `backend/agent/src/ai/` at AI-00's merge, when its Go declarations are enumerated (for example with `go doc github.com/cachicamas/backend/agent/src/ai`), then the package exports nothing and declares nothing beyond the package clause and its documentation comment.
- **S-AGM-032** — Given `backend/agent/src/ai/`'s package documentation, when it is read, then it states the Layer 1 boundary and the import rule, and cites ADR 0005 § D1.
- **S-AGM-033** — Given `backend/agent/src/agenttest/`, when its test files are read, then at least one declares an external test package (`package agenttest_test`) AND imports `github.com/cachicamas/backend/agent/src/ai`.
- **S-AGM-034** — Given the repository, when `cd backend/agent && go test ./src/agenttest/...` runs, then it compiles and exits 0, proving `src/ai` is importable from outside its own package.
- **S-AGM-035** — *(AMENDED 2026-08-12 by AG-03)* Given the repository from AG-03's merge onward, when `test -e backend/agent/src/agent`, `test -e backend/agent/src/coding` and `test -e backend/agent/src/cmd` each run, then the first exits **zero** — `src/agent` exists, is a directory, and is a direct sibling of `src/ai` and `src/agenttest` — while the other two exit **non-zero**: neither `src/coding` nor `src/cmd` exists in any form, including as an empty directory or a directory holding only a placeholder file. (Before AG-03's merge all three exited non-zero; `R-AGM-004`'s prose named docs 0003 and 0004 as the milestones that change this, and AG-03 is doc 0003's.)
- **S-AGM-036** — Given `backend/agent/src/agenttest/` and `backend/agent/src/ai/`, when their parent directories are compared, then both resolve to `backend/agent/src`, so `../ai` from any file in `agenttest` resolves to the Layer 1 package directory. This is the layout invariant AI-20.4's signature guard depends on.

### R-AGM-005 — Forward guard: Layer 1 purity

*(MODIFIED 2026-08-12 by AG-03 — restates the entire requirement, including every unchanged scenario, so no landed proof is lost at archive. `S-AGM-041` is amended because AG-03 may narrow the guard's scanned roots to resolve the self-reference hazard the guard's own source recorded at `:82-92`; the amended wording asserts complete Layer 1 coverage independently of which mechanism design selects. Previously amended 2026-08-08 by AI-37, which made the third-party allowlist group non-empty.)*

The module MUST carry an executable guard at `backend/agent/src/ai/import_boundary_test.go` that fails when any package of `backend/agent` — including its test packages and its transitive dependencies — depends on anything outside an explicit allowlist.

The guard MUST use `go list -deps -test`. Bare `go list -deps` is insufficient: it does not report test-only imports, and a direct-imports listing reports neither test imports nor transitive dependencies ([ADR 0005 § Guard A](../../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement), finding S6).

The guard's scanned set MUST cover **every Layer 1 package of the module** — `src/ai/…`, `src/agenttest/…` and `src/handoff` — and MUST use fully-qualified module-path patterns rather than relative patterns resolved from an assumed working directory. Because `go list` emits the scanned pattern's **own members**, and because Layer 2's package path is itself a forbidden prefix, the guard MUST distinguish a member of its own scanned set from a genuine import violation. Whether that is achieved by narrowing the scanned roots to Layer 1's packages or by exempting the pattern's own members from the prefix match is an implementation choice; the guard's source MUST record which mechanism it uses and why. Narrowing that drops coverage of any Layer 1 package is a violation of this requirement, not an implementation detail.

The allowlist MUST be **deny-by-default**: an import path that is neither a member of the Go standard library set nor a package of this module fails the guard *even when no rule names it*. The allowlist MUST be expressed as three groups — the standard library, this module's own packages, and a third-party group whose only permitted growth path is a milestone with its own ADR.

**The third-party group is no longer empty.** At AI-37 it admits exactly the OpenTelemetry **API** paths this module actually imports, plus the packages transitively required in order to import them at all. Every entry MUST carry, in the guard's own source, the ADR clause authorising it — ADR 0005 § D3 for each OpenTelemetry path. An entry that is not an OpenTelemetry module and is admitted only because an authorised path cannot be imported without it MUST record that reasoning in place: authorising an import path necessarily authorises its closure, and § D3's "any additional OTel module requires its own ADR" clause is not engaged by a non-OpenTelemetry transitive requirement. That reasoning is bounded by `R-AGM-008`'s set-equality pin, which is what stops it becoming a blank cheque.

In addition to the allowlist, the guard MUST carry a named forbidden-prefix list covering: `github.com/cachicamas/backend/database_administrator`, `github.com/cachicamas/backend/workspace_syncer`, `github.com/cachicamas/backend/agent/src/agent`, `github.com/cachicamas/backend/agent/src/coding`, `github.com/cachicamas/backend/agent/src/cmd`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters`, and `go.opentelemetry.io/contrib/bridges/otelslog`. **AI-37 adds no forbidden prefix and removes none; AG-03 adds none and removes none**; that table stays exactly as written. The OpenTelemetry **API** paths of ADR 0005 § D3 MUST NOT appear in the forbidden list.

A guard that has only ever been green is unproven. This guard closed on **bite proof**: it was shown to fail against three separate deliberate violations, with the failing output recorded, before it landed green. Any amendment to its allowlist MUST re-prove the same property — which at AI-37 is discharged by `R-AOB-002`'s two recorded bites. Any amendment to its **scanned set** MUST likewise re-prove that a genuine violation still bites — which at AG-03 is discharged by `agent-package-scaffold`'s `S-AGP-062`.

#### Scenarios

- **S-AGM-040** — Given the repository with no violation present, when `cd backend/agent && go test ./src/ai/...` runs, then the guard test passes.
- **S-AGM-041** — *(AMENDED 2026-08-12 by AG-03)* Given the guard's source, when its `go list` invocation and its scanned-root definition are read, then it passes both `-deps` and `-test`; its pattern or patterns are fully-qualified module paths, never relative; the resulting scanned set covers every Layer 1 package of the module — `src/ai/…`, `src/agenttest/…` and `src/handoff` — with none omitted; and, where the scanned set can contain the guard's own pattern members, the source states in place which mechanism prevents a member of that set from being reported as an import violation and why that mechanism was chosen. (Previously: *"then it passes both `-deps` and `-test`, and its pattern covers every package of the module."* That wording was true of `layer1Pattern = modulePath + "/..."` and is the wording AG-03 may falsify by narrowing the roots; the property it protected — complete Layer 1 coverage with no silent narrowing — is restated above in a form independent of the mechanism AG-03's design selects.)
- **S-AGM-042** — Given the guard's source, when its allowlist is read, then the standard-library group is derived from the toolchain rather than from a "path segment contains no dot" heuristic, AND the third-party group is present, non-empty, and annotated per entry with the ADR clause that authorises it and — for any entry admitted only as the closure of an authorised path — with that reasoning stated in place.
- **S-AGM-043** — Given the guard's source, when its forbidden-prefix list is read, then it names all eight prefixes above, unchanged from before this milestone, AND contains no OpenTelemetry API path (`go.opentelemetry.io/otel`, `…/otel/trace`, `…/otel/attribute`, `…/otel/codes`, `…/otel/metric`).
- **S-AGM-044** — **(bite 1)** Given a scratch file in `backend/agent/src/ai/` that imports `github.com/cachicamas/backend/database_administrator/src/domain`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. The failing output is recorded; the scratch file is then removed.
- **S-AGM-045** — **(bite 2)** Given a scratch file in `backend/agent/src/ai/` that imports `go.opentelemetry.io/otel/sdk/trace`, when `go test ./src/ai/...` runs, then the guard FAILS and its message names the offending import path. Recorded, then removed. This bite MUST be re-proven at AI-37, because AI-37 is the milestone that makes the neighbouring API paths admissible and therefore the one at which an over-broad prefix would first go unnoticed.
- **S-AGM-046** — **(bite 3)** Given a scratch file in `backend/agent/src/ai/` that imports an arbitrary third-party module named by no forbidden prefix and by no allowlist entry, when `go test ./src/ai/...` runs, then the guard FAILS **on the deny-by-default allowlist rule**, not on a named prefix. This is what proves the allowlist is still deny-by-default after it stopped being empty. Recorded, then removed together with any require entry its presence forced into `go.mod`.
- **S-AGM-047** — Given the diff of any change that touches the guard, when it is inspected, then no scratch violation file appears in it AND `backend/agent/go.mod` declares no require entry that an ADR does not authorise.
- **S-AGM-048** — Given the guard's source, when its handling of `go list -deps -test` output is read, then it normalizes the synthesized entries the toolchain emits — the ` [<pkg>.test]` test-variant suffix and the `<pkg>.test` binary — so that neither is mistaken for a real import path.

---

## Unchanged

`R-AGM-001`, `R-AGM-002`, `R-AGM-003`, `R-AGM-006`, `R-AGM-007`, `R-AGM-008`, all three non-functional requirements, and every scenario not restated above are unaffected by this change. In particular `S-AGM-069`…`S-AGM-073` continue to hold: `src/agent/` adds no external dependency, and the closure pin filters `modulePath`-prefixed entries before comparing.

At archive time the spec's own header note MUST gain an "Amended 2026-08-12 (AG-03)" entry recording the two amended scenarios and the reason, in the same shape as the existing AI-37 note, and the acceptance-criteria section MUST record which scenarios were re-verified at AG-03's close.
