# Design — Agent module scaffold and boundary guards

> **Change**: `cachicamas-agent-module-scaffold`
> **Milestone**: AI-00 of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards)
> **Phase**: design
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Inputs**: `explore.md`, `proposal.md`, `specs/agent-module-scaffold/spec.md`
> **Diagrams**: ASCII (project convention — the existing SDD artifacts use no mermaid)
> **Authoring constraint**: this document specifies *mechanisms*, not Go type names. Layer 1 has no types yet; AI-01 names the vocabulary and AI-04 declares the first type.

---

## 1. The exact file tree

```
cachicamas/                                       (repo root)
├── go.work                                       NEW   — 5 lines
├── go.work.sum                                   NEW?  — only if the toolchain writes one
├── .gitignore                                    unchanged — verified not to exclude go.work*
└── backend/
    ├── agent/                                    NEW MODULE
    │   ├── go.mod                                NEW   — 3 lines, zero requires
    │   ├── Makefile                              NEW   — ~120 lines, copy-derived
    │   ├── .golangci.yml                         NEW   — 30 lines, verbatim copy
    │   ├── .gitignore                            NEW   — 12 lines, verbatim copy
    │   ├── README.md                             NEW   — ~60 lines
    │   ├── bin/                                  (ignored; created by `make tools`)
    │   └── src/
    │       ├── ai/
    │       │   ├── doc.go                        NEW   — package clause + doc comment only
    │       │   └── import_boundary_test.go       NEW   — Guard A (forward), ~130 lines
    │       └── agenttest/
    │           ├── doc.go                        NEW   — package clause + why-this-is-a-sibling
    │           └── import_compile_test.go        NEW   — external-package compile proof, ~20 lines
    ├── database_administrator/
    │   ├── go.mod                                UNCHANGED — asserted, no `replace`
    │   └── src/
    │       ├── domain/imports_test.go            MODIFIED  — Guard B, both halves, ~+110 lines
    │       └── tools/tools.go                    UNCHANGED — byte-identical, diff recorded
    └── workspace_syncer/                         UNCHANGED — listed in go.work, no file edited
```

Three directories are **absent by design** and their absence is a chartered property: `backend/agent/src/agent/`, `.../src/coding/`, `.../src/cmd/`. See § 6.

### 1.1 Why `src/agenttest/` sits where it sits

```
backend/agent/src/
        ├── ai/          ←──┐
        └── agenttest/   ───┘  `../ai` resolves here, from any file in agenttest
```

[ADR 0005 § D2](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2) and Guard C: AI-20.4 will land a signature guard that resolves `../ai/provider.go` relative to `runtime.Caller(0)` — that is, relative to *its own source file*, which lives in `agenttest`. Nesting `agenttest` one level deeper, or moving it under `ai/`, breaks that resolution with no compile error and no warning anywhere in the tree. The guard fails loudly when it fails, but a future reorganisation will hit it blind, which is why the constraint is written down in three places (the ADR, doc 0002's AI-00.2 check 2, and `agenttest/doc.go` itself).

At AI-00 `src/ai` exports nothing, so `import_compile_test.go` can only prove *importability*. That is the whole assignment: a blank-identifier import in an external test package (`package agenttest_test`) that compiles. AI-06.2 is the leaf that proves external *readability* of a real type.

---

## 2. `go.mod` and `go.work`

### 2.1 `backend/agent/go.mod`

```
module github.com/cachicamas/backend/agent

go 1.26.3
```

That is the entire file. No `require`, no `replace`, no `toolchain` line, and no `go.sum`. The zero-requires property is not incidental tidiness — it is what makes AI-24's transport selection and AI-37's OpenTelemetry API addition *visible events*. A module that already has a `require` block absorbs a new line without anyone noticing; a module with none cannot.

`go 1.26.3` matches the local toolchain (`go1.26.3 darwin/arm64`) and both existing modules.

### 2.2 `go.work`

```
go 1.26.3

use (
	./backend/agent
	./backend/database_administrator
	./backend/workspace_syncer
)
```

[ADR 0005 § Migration](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#migration) asks for this "for editor and future-CI ergonomics". Two consequences are worth stating rather than discovering:

**It changes how the two existing modules resolve dependencies.** In workspace mode the toolchain computes one build list by running MVS across every listed module, and verifies module hashes against `go.work.sum` rather than the per-module `go.sum`. `backend/agent` has zero requires and therefore contributes nothing to MVS, and the other two pin the same versions of their shared dependencies (`go.opentelemetry.io/otel v1.44.0`, `github.com/labstack/echo/v5 v5.2.1`), so no version bump is expected. *Expected* is not evidence. R-AGM-007 requires a pre-change `make test` baseline in both modules, re-run and compared after `go.work` lands. If the toolchain writes a `go.work.sum`, it is committed (S-AGM-022) — an uncommitted one produces "missing go.sum entry" failures on a fresh clone.

**It must not be gitignored.** The root `.gitignore` was checked and contains no `go.work` pattern. Some Go project templates ignore `go.work` on the reasoning that it is a per-developer file; here it is a chartered deliverable, so S-AGM-023 pins both its presence and the absence of an ignore rule.

**It does not create a dependency.** Listing a module in `go.work` does not import it. Go still refuses a cycle, and the reverse guard still proves that nothing in `database_administrator` names the agent module.

---

## 3. Guard A — the forward guard

**File**: `backend/agent/src/ai/import_boundary_test.go`
**Package**: `ai_test` (external — the guard inspects the module, it does not participate in it)
**Enforces**: [ADR 0005 § D1 row 1](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2), [§ D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary), [§ Guard A](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement)

### 3.1 Why `go list -deps -test`, measured

[ADR 0005 § Guard A](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement) records that the retired guard used `go list -f '{{range .Imports}}'`, which sees neither `.TestImports` / `.XTestImports` nor transitive dependencies — the two blind spots of finding S6. The ADR prescribes moving to `go list -deps` with an allowlist.

`-deps` alone closes the *transitive* blind spot but **not** the test-import one. Measured on this worktree against a package that has tests:

| Command | `testing` in output |
| --- | ---: |
| `go list -deps ./src/domain` | 0 |
| `go list -deps -test ./src/domain` | 1 |

Doc 0002's AI-00.3 test-list item 1 requires the guard to "cover test imports and transitive dependencies". Only `-deps -test` delivers both. This is a refinement of the ADR's wording, not a departure from it: the ADR names the mechanism and the property, and `-test` is what makes the named mechanism deliver the named property.

### 3.2 Normalizing what `-test` adds

`-deps -test` emits synthesized packages that are not import paths. Observed shapes, verbatim:

```
github.com/…/src/domain [github.com/…/src/domain.test]        ← test variant of a real package
github.com/…/src/domain_test [github.com/…/src/domain.test]   ← the external test package
github.com/…/src/domain.test                                  ← the synthesized test binary
```

Normalization, in order:

1. Trim everything from the first ` [` onward — the bracketed suffix names the test binary a variant belongs to, not a dependency.
2. Drop any path with the suffix `.test` — that is the synthesized binary.
3. Drop any path with the suffix `_test` **only when** its prefix is a package of this module — that is our own external test package, which is own-module by definition and would otherwise need a special allowlist entry.
4. Deduplicate.

Skipping step 1 is the subtle failure: the raw line `…/src/ai [….test]` matches neither the stdlib set nor the own-module prefix (because of the trailing bracket), so an un-normalized guard fails on its own module — and the natural "fix" is to loosen the matching, which is how a deny-by-default guard quietly becomes a substring guard.

### 3.3 Deciding "is this the standard library"

Two candidate mechanisms; the design picks the first.

**Chosen — exact set membership from `go list std`.** Run `go list std` once, load the result into a set, and treat membership as the stdlib test.

**Rejected — the "first path segment contains no dot" heuristic.** It is the folklore answer and it is wrong here. Measured: `go list std` contains 17 paths under `vendor/`, such as `vendor/golang.org/x/crypto/chacha20` and `vendor/golang.org/x/net/dns/dnsmessage`, and those paths appear **verbatim** in real `-deps` output (confirmed against `database_administrator/src/interfaces/...`). Their first segment is `vendor`, which has no dot — so the heuristic accidentally admits them for the wrong reason on some toolchains and rejects the dotted remainder on others, depending on exactly where you split. Patching it with a `vendor/` special case layers a second heuristic on the first. Exact-set membership has no such edge, costs one extra subprocess per test run, and cannot drift when the standard library gains or vendors a package.

### 3.4 The three allowlist groups

Deny-by-default means the guard's question is "is this import *permitted*", never "is this import *forbidden*". Three groups answer it:

| Group | Contents at AI-00 | Who may add to it |
| --- | --- | --- |
| **stdlib** | the exact set returned by `go list std` | the Go toolchain |
| **own module** | prefix `github.com/cachicamas/backend/agent/` | any milestone that adds a package to this module |
| **third-party** | **empty** | only a milestone with an ADR: AI-24 (transport, own ADR gate) and AI-37 (OpenTelemetry API, pre-authorized by ADR 0005 § D3 and by nothing else) |

The third group is the load-bearing one and it is *present and empty*, with a comment naming its two permitted growth paths. An empty-but-named list is a different artifact from a missing list: it turns "add a dependency" into "edit a list whose comment tells you which ADR you need", which is exactly the visible, gated event doc 0002's AI-00.3 item 3 asks for.

**The D3 tension, resolved.** ADR 0005 § D3 says Layer 1 *may* import the OpenTelemetry API. Deny-by-default says only stdlib and own-module are allowed. Both are true simultaneously because at AI-00 the module has zero requires, so no `go.opentelemetry.io/...` path can appear in `-deps` output at all. AI-37 adds the API paths to the third group as part of its own change. Until then the D3 split is encoded only where it can be encoded — in the forbidden-prefix list of § 3.5.

### 3.5 The forbidden-prefix list

Deny-by-default already rejects everything not allowlisted, so this list is strictly redundant *today*. It is not decorative:

- It produces a better failure message — "imports the OpenTelemetry SDK, forbidden by ADR 0005 § D3" beats "not in the allowlist".
- It survives a future over-broad allowlist entry. When AI-37 adds `go.opentelemetry.io/otel` to the third group, a careless prefix match would admit `go.opentelemetry.io/otel/sdk/trace` along with it. The forbidden list catches that. Belt and braces, and this is precisely why doc 0002's item 2 requires both coverages to be named.
- It makes the boundary *readable* — the list is documentation that executes.

| Forbidden prefix | Source |
| --- | --- |
| `github.com/cachicamas/backend/database_administrator` | D1 row 1 — sibling backend module |
| `github.com/cachicamas/backend/workspace_syncer` | D1 row 1 — sibling backend module |
| `github.com/cachicamas/backend/agent/src/agent` | D1 row 1 — Layer 2, does not exist yet |
| `github.com/cachicamas/backend/agent/src/coding` | D1 row 1 — Layer 3, does not exist yet |
| `github.com/cachicamas/backend/agent/src/cmd` | D1 row 1 — composition root, does not exist yet |
| `go.opentelemetry.io/otel/sdk` | D3 — SDK is a composition-root concern |
| `go.opentelemetry.io/otel/exporters` | D3 — exporters are a composition-root concern |
| `go.opentelemetry.io/contrib/bridges/otelslog` | D3 — Layer 3 and `cmd` only |

**Not forbidden**, and the guard must be checked for their absence (S-AGM-043): `go.opentelemetry.io/otel`, `…/otel/trace`, `…/otel/attribute`, `…/otel/codes`, `…/otel/metric`. Those are the API paths D3 permits, and a guard that forbids them would make AI-37 impossible without weakening the guard.

Note the ordering trap in the three own-module prefixes: `…/agent/src/agent` is forbidden while `…/agent/` is allowed, so a naive "allow own-module first, then check forbidden" pass admits Layer 2. The guard checks the **forbidden list first**, then the allowlist.

### 3.6 Algorithm

```
TestLayer1_ImportBoundary:

  stdlib  ← set( lines of `go list std` )
  deps    ← lines of `go list -deps -test github.com/cachicamas/backend/agent/...`

  for each raw in deps:
      p ← raw truncated at the first " ["            # test-variant suffix
      skip p if it ends with ".test"                 # synthesized test binary
      skip p if empty

      # 1. forbidden first — a named prefix always wins over any allow rule
      for each (prefix, reason) in FORBIDDEN:
          if p has prefix:
              FAIL "%s imports %q — forbidden by %s"  (package, p, reason)

      # 2. deny-by-default allowlist
      if p ∈ stdlib:                             continue   # group 1
      if p has prefix OWN_MODULE:                continue   # group 2
      if p has any prefix in ALLOWED_THIRD_PARTY: continue  # group 3 — EMPTY at AI-00
      FAIL "%s is not in the Layer 1 allowlist (stdlib + own module). "
           "Adding a dependency requires an ADR — see ADR 0005 § D1/§ D3, AI-24, AI-37."
```

Two properties this shape gives for free, both required by the spec:

- **NFR-AGM-001 (cwd independence)** — the pattern is the fully-qualified module path, not `./...`. `go test` sets the working directory to the package under test, so a relative pattern would silently scope the guard to `src/ai` alone and miss every other package the module later grows.
- **S-AGM-046 (deny-by-default is provable)** — bite 3 imports a third-party module named by *no* forbidden prefix, so it can only fail on the allowlist branch. If the guard were a forbidden-substring scan, bite 3 would pass and the guard would be worthless the day AI-24 runs `go get`.

### 3.7 Bite protocol

Three violations, one at a time, each in a scratch file under `backend/agent/src/ai/`:

| Bite | Scratch import | Must fail on | Recorded as |
| --- | --- | --- | --- |
| 1 | `github.com/cachicamas/backend/database_administrator/src/domain` | forbidden prefix (sibling module) | S-AGM-044 |
| 2 | `go.opentelemetry.io/otel/sdk/trace` | forbidden prefix (D3 SDK) | S-AGM-045 |
| 3 | any third-party module named by no forbidden prefix | **the allowlist branch** | S-AGM-046 |

Bites 1 and 3 need a `require` line in `go.mod` for the import to resolve — bite 1 also needs a `replace ../database_administrator`. Both are transient, live only in the working tree, and are reverted with the scratch file. The closing evidence is threefold: the recorded red output, a `go.mod` with zero requires, and a clean `git status`. S-AGM-047 makes the merged diff the final check.

An alternative that avoids touching `go.mod` for bites 1 and 3 was considered — asserting the guard's decision function directly against a synthetic list of import paths, without running `go list`. It is rejected: it proves the *predicate* bites, not the *guard*, and the two blind spots of finding S6 both lived in the plumbing between `go list` and the predicate, not in the predicate itself.

---

## 4. Guard B — the reverse guard

**File**: `backend/database_administrator/src/domain/imports_test.go` (modified)
**Package**: `domain_test` (already)
**Enforces**: [ADR 0005 § D1 rows 5–7](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2), [§ Guard B](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement)
**Evidence gate**: `make test` in `backend/database_administrator/` — the single documented exception to doc 0002's global gate.

### 4.1 Where it lives, and why not somewhere better-named

The file already exists and already owns the `runGoListDeps` helper. Two alternatives were weighed:

- **A new `src/architecture/` package holding only the guard.** Better name, worse mechanics: a directory containing only `_test.go` files in an external test package has no non-test Go files, and AI-00.1 check 2 requires `go build ./...` to succeed in this module. Making it work means inventing a production file whose only purpose is to keep the build happy.
- **`src/cmd/server/`.** It is `package main`, so a test file works — but `cmd/server` is one of the two packages ADR 0005 § D1 row 5 *permits* to import the agent module. Housing the guard that polices the exemption inside the exemption is a confusing read.

Chosen: keep both halves in `src/domain/imports_test.go`. It needs no new package, cannot break `go build ./...`, and keeps the whole reverse guard in one reviewable file. The placement is admittedly wider than the package that hosts it; a comment says so. If a later change grows a real architecture-test package, moving the module-scope half there is a mechanical rename with no behavior change.

### 4.2 Half one — `src/domain` is transitively clean

Extends the existing `TestDomainLayer_DoesNotImportPgx` pattern with a second forbidden module path. Transitive (`go list -deps`), because domain must not reach the agent module by *any* path, and there is no false-positive risk: nothing imports domain "downward", so domain's dependency closure can never legitimately contain the agent module.

```
TestDomainLayer_DoesNotImportAgentModule:
  deps ← runGoListDeps("github.com/cachicamas/backend/database_administrator/src/domain")
  for each line in deps:
      if line has prefix "github.com/cachicamas/backend/agent":
          FAIL "domain must not import the agent module (ADR 0005 § D1 row 6); found %q"
```

### 4.3 Half two — module scope, direct imports only

This is the half that never existed. It asks a different question from half one: not "what does this package depend on" but "**who names** the agent module".

**It must not be transitive, and that is the most consequential decision in this document.** ADR 0005 § D1 row 5 permits `src/application` and `src/cmd/server` to import the agent module. The moment row 5 is exercised, every package that depends on `application` — `interfaces/http`, `cmd/server`, and anything else — inherits the agent module in its dependency closure. A transitive module-scope guard would then fail on packages that did nothing wrong, and the path of least resistance would be to widen the exemption list until the guard means nothing. Direct-import scanning is the correct instrument for a "who may name this" rule, and it stays correct after row 5 is exercised.

One `go list` call gathers every package's direct imports, including both flavours of test import:

```
go list -f '{{.ImportPath}}|{{join .Imports " "}}|{{join .TestImports " "}}|{{join .XTestImports " "}}' \
        github.com/cachicamas/backend/database_administrator/...
```

Verified on this worktree: 13 lines, zero occurrences of the agent module.

```
TestModule_OnlyApplicationAndCmdServerMayImportAgentModule:

  EXEMPT = { ".../database_administrator/src/application",
             ".../database_administrator/src/cmd/server" }        # ADR 0005 § D1 row 5

  for each line in go list -f '<above>' <module>/...:
      pkg, imports, testImports, xTestImports ← split(line, "|")
      if pkg has any prefix in EXEMPT:  continue
      for each imp in imports ∪ testImports ∪ xTestImports:
          if imp has prefix "github.com/cachicamas/backend/agent":
              FAIL "%s imports %q — only src/application and src/cmd/server may name the "
                   "agent module (ADR 0005 § D1 row 5); this scan is direct-imports-only "
                   "*because* row 5 is permitted"
```

The rule's reason is in the failure message on purpose. A future maintainer who sees the guard fail and reaches for `-deps` needs to read why that is the wrong fix at the moment they are considering it, not three files away.

**Green from birth is the design, not a shortcut.** Today the count is zero, so the test passes on the first run. ADR 0005 § Guard B states the value plainly: it fails on the first accidental import rather than on the first production incident. That is why the leaf still requires a bite (§ 4.4) — a green-from-birth assertion that has never been shown to fail is indistinguishable from a broken one.

### 4.4 Bite protocol

One violation: add `_ "github.com/cachicamas/backend/agent/src/ai"` to a file in `backend/database_administrator/src/domain/`. This requires a temporary `require` and `replace ../agent` in `database_administrator/go.mod` for resolution — both transient, both reverted with the scratch import. `cd backend/database_administrator && make test` must fail; the red output is recorded (S-AGM-055); the violation and the `go.mod` edits are dropped. S-AGM-056 and S-AGM-065 make the merged diff the final check: `database_administrator/go.mod` must be unmodified and must carry no `replace`.

This bite exercises **both** halves at once — a domain import is caught by half one transitively and by half two directly — which is convenient but must not be mistaken for two bites. Doc 0002's AI-00.4 item 3 asks for exactly one.

---

## 5. Build configuration: copied, not shared

Doc 0002's AI-00 charter states the decision and its reason: *"a shared build config would be a fourth thing to own and would couple two modules that must not be coupled."*

The alternatives and why each loses:

| Option | Why not |
| --- | --- |
| A root `Makefile` that dispatches into each module | Creates a fourth build artifact with no module of its own. The three modules then share a failure mode: a typo in the root file breaks all three test gates at once, and the "run `make test` in the module directory" instruction — which is the *only* gate that exists without CI — becomes a lie. |
| A shared `.golangci.yml` at the repo root, included per module | `golangci-lint` resolves config relative to the invocation directory; making three modules point at one file means three relative paths that break the first time a module moves. It also couples the lint posture of a stdlib-only module to that of a Postgres/Echo/OTel service, which will want different linters as it grows. |
| A `replace`-linked internal tooling module | A dependency, which this module is not permitted to have, for the benefit of build config. |

**The drift risk ADR 0005 records.** [§ D2](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2) maps build/lint config to `backend/agent/Makefile` and `backend/agent/.golangci.yml` as *copies* and records the resulting drift risk. Three copies of a `Makefile` will diverge; the only question is whether the divergence is visible.

The mitigation is not synchronization — it is **naming the load-bearing surface**. R-AGM-002 pins two targets by name and body: `test` is `go test -race -v ./...`, and `lint` runs `golangci-lint` at `v2.9.0`. Every evidence gate in doc 0002, doc 0003 and doc 0004 is spelled `make test` in a module directory. If those two targets stay identical across the three modules, drift anywhere else — a `run` target here, an integration target there — is harmless and arguably correct, because the modules genuinely differ. If either target drifts, it is a spec violation with a named requirement to point at, not an unnoticed edit.

**What `backend/agent/Makefile` drops from its source**, and why each is dropped rather than carried:

| Dropped | Reason |
| --- | --- |
| `run` | There is no `main` package until doc 0004's `src/cmd/cachicamas/`. A target that cannot run is worse than an absent one. |
| `install` (`go mod download && go mod verify`) | Zero requires; there is nothing to download or verify. Re-added by whichever of AI-24 / AI-37 lands first. |
| `test/integration` and its compose variables | It boots compose Postgres. Layer 1 performs no I/O and has no container. |
| `MAIN_PKG`, `BINARY_NAME` | No binary. `build` becomes `go build ./...`, keeping the target name stable for when a binary arrives. |

Kept verbatim: `LOCALBIN := bin`, `GOLANGCI_LINT_VERSION := v2.9.0`, the `export PATH` prepend, `GOFLAGS := -trimpath`, and the `help` awk target. `.golangci.yml` and `.gitignore` are byte-for-byte copies (S-AGM-014), which makes them reviewable by diff rather than by reading.

---

## 6. Why three directories must not exist

`backend/agent/src/agent/`, `.../src/coding/` and `.../src/cmd/` are named in Guard A's forbidden-prefix list. They belong to docs 0003 and 0004 and are not created here — not as empty directories, not with a placeholder `doc.go`.

Doc 0002's AI-00.2 check 3 gives the reason and it is a testability argument, not a tidiness one. A forbidden prefix is only meaningful while nothing legitimately matches it. Create `src/agent/` with a `doc.go`, and `github.com/cachicamas/backend/agent/src/agent` is now a real package of this module — which means it matches the **own-module allowlist prefix** `github.com/cachicamas/backend/agent/` as well as the forbidden prefix. The guard's ordering (forbidden first, § 3.5) keeps it correct, but the forbidden rule can no longer be *demonstrated*, because you cannot construct a violation of it without also constructing a legal package. The rule becomes unfalsifiable, which is the state a guard must never reach.

S-AGM-030 and S-AGM-035 pin the absence from two directions: `src/` contains exactly two entries, and each of the three paths fails an existence check.

---

## 7. Evidence protocol

No CI exists (`.github/workflows/` is absent), so "recorded green" means output pasted into the PR description. The change carries nine recorded artifacts:

| # | Artifact | Command | Requirement |
| --- | --- | --- | --- |
| 1 | `database_administrator` pre-change baseline | `cd backend/database_administrator && make test` | S-AGM-060 |
| 2 | `workspace_syncer` pre-change baseline | `cd backend/workspace_syncer && make test` | S-AGM-060 |
| 3 | Forward guard bite 1 (sibling module) — **red** | `cd backend/agent && go test ./src/ai/...` | S-AGM-044 |
| 4 | Forward guard bite 2 (OTel SDK) — **red** | same | S-AGM-045 |
| 5 | Forward guard bite 3 (arbitrary third party) — **red, on the allowlist branch** | same | S-AGM-046 |
| 6 | Reverse guard bite — **red** | `cd backend/database_administrator && make test` | S-AGM-055 |
| 7 | New module green | `cd backend/agent && make test && make lint` | S-AGM-012, S-AGM-013 |
| 8 | Existing modules compared to baseline | `make test` in both | S-AGM-062, S-AGM-063 |
| 9 | `tools.go` untouched | `git diff <base>..HEAD -- backend/database_administrator/src/tools/tools.go` | S-AGM-064 |

Baselines (1 and 2) must be captured **before** the first file of this change is created. Once `go.work` exists, a "pre-change" run is no longer available without stashing, and the whole point of the baseline is to detect exactly the perturbation `go.work` might introduce.

The comparison in artifact 8 is over the **result set** — the list of test names and their outcomes — not over raw bytes. `go test -v` prints elapsed times, so byte equality is impossible and demanding it would make the requirement unverifiable rather than strict. After AI-00.4, `database_administrator`'s result set must equal its baseline **plus exactly** the two reverse-guard tests; `workspace_syncer`'s must equal its baseline exactly.

---

## 8. Diff forecast

| File | Kind | Lines |
| --- | --- | ---: |
| `go.work` | new | 5 |
| `backend/agent/go.mod` | new | 3 |
| `backend/agent/Makefile` | new | ~120 |
| `backend/agent/.golangci.yml` | new | 30 |
| `backend/agent/.gitignore` | new | 12 |
| `backend/agent/README.md` | new | ~60 |
| `backend/agent/src/ai/doc.go` | new | ~25 |
| `backend/agent/src/ai/import_boundary_test.go` | new | ~130 |
| `backend/agent/src/agenttest/doc.go` | new | ~15 |
| `backend/agent/src/agenttest/import_compile_test.go` | new | ~20 |
| `backend/database_administrator/src/domain/imports_test.go` | modified | ~+110 |
| **Total** | | **~530** |

Above doc 0002's budget (prefer < 250, reassess before 400) and shipped as one PR by explicit user decision. The reasoning is in `proposal.md` § "Review budget exception" and is restated in `tasks.md`; doc 0001 § 9 (Process) requires the PR description to carry it.

Two facts moderate what the number implies. About 220 of the 530 lines are configuration copied verbatim or near-verbatim from an existing module and are reviewed by diffing against the source. About 260 are two test files that contain no production code path, cannot be called by anything, and are each proven by a recorded failure before they land. The genuinely novel production surface of this change is 3 lines of `go.mod`, 5 of `go.work`, and two package doc comments.

---

## 9. Open design points deferred to later milestones

| Point | Deferred to | Note |
| --- | --- | --- |
| Adding the OpenTelemetry API to the third allowlist group | AI-37 | Pre-authorized by ADR 0005 § D3 for exactly the API paths in its table and nothing else. |
| Adding a transport / vendor SDK to the third allowlist group | AI-24 | Its own ADR gate. Bite 3 exists to guarantee this cannot happen silently. |
| The `src/agenttest` signature guard resolving `../ai/provider.go` | AI-20.4 | This change guarantees only the sibling layout it depends on. |
| An ambient-authority scan over the adapter package | AI-25.2 | Different guard, different milestone. |
| Extracting the module-scope reverse guard into a dedicated architecture-test package | any later change | A mechanical rename; deliberately not done here (§ 4.1). |
| Adding CI so the three `make test` gates run without a human | unowned | ADR 0005 records it under Consequences; doc 0002 repeats the caveat once, globally. It is not this change's to solve. |
| Amending ADR 0005 § Migration, which assigns the module move to a milestone identifier that no longer exists | separate change | Doc 0002 already records the staleness. Silently editing a merged ADR from inside an implementation PR is the wrong shape. |
