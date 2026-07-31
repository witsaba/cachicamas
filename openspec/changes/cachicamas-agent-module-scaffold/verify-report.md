# Verify report — Agent module scaffold and boundary guards

> **Change**: `cachicamas-agent-module-scaffold`
> **Milestone**: AI-00 of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards)
> **Phase**: verify
> **Status**: **PASS**
> **Date**: 2026-07-31
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Base**: `origin/main` @ `b6c59e6`
> **Mode**: Strict TDD (guard leaves closed on recorded red before green; mechanical leaves on recorded Check evidence)
> **Toolchain**: go1.26.3 darwin/arm64, golangci-lint v2.9.0 (pinned)

---

## 1. Charter acceptance

AI-00's charter states three acceptance clauses. Each is verified below against recorded output, not against intent.

| # | Charter clause | Verdict | Evidence |
| --- | --- | --- | --- |
| 1 | `make test` and `make lint` are green in the new module | **PASS** | § 3.1, § 3.2 |
| 2 | `make test` unchanged in the other two modules | **PASS**, with the qualification in § 4.2 | § 4.1, § 4.2 |
| 3 | Both import directions are mechanically guarded, and both guards are recorded biting | **PASS** — four bites recorded | § 5 |

---

## 2. Deliverable inventory

Every artifact the charter names, and nothing it does not.

| Path | Present | Note |
| --- | :---: | --- |
| `backend/agent/go.mod` | ✅ | module path and `go 1.26.3` as chartered; **zero requires** |
| `backend/agent/go.sum` | ✅ absent | correct — a module with no requires has no sum file |
| `backend/agent/Makefile` | ✅ | `test` and `lint` bodies pinned; see § 3.3 |
| `backend/agent/.golangci.yml` | ✅ | byte-identical to `database_administrator`'s |
| `backend/agent/README.md` | ✅ | all four required elements; see § 6 |
| `backend/agent/.gitignore` | ✅ | ignores `bin/` and coverage artifacts |
| `backend/agent/src/ai/` | ✅ | package documentation only, no declarations |
| `backend/agent/src/agenttest/` | ✅ | direct sibling of `src/ai/`; external test compiles |
| `go.work` (repo root) | ✅ | lists all three modules |
| forward guard | ✅ | `backend/agent/src/ai/import_boundary_test.go` |
| reverse guard | ✅ | `backend/database_administrator/src/domain/imports_test.go` |
| `src/agent/`, `src/coding/`, `src/cmd/` | ✅ absent | deliberately not created — AI-00.2 check 3 |

---

## 3. The new module

### 3.1 `make test`

```
$ cd backend/agent && make test
go test -race -v ./...
=== RUN   TestLayer1Package_IsImportableFromAnExternalPackage_Compiles
--- PASS: TestLayer1Package_IsImportableFromAnExternalPackage_Compiles (0.00s)
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.507s
=== RUN   TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault
--- PASS: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.04s)
=== RUN   TestLayer1_ModuleHasNoDependencies_ZeroRequires
--- PASS: TestLayer1_ModuleHasNoDependencies_ZeroRequires (0.04s)
ok  	github.com/cachicamas/backend/agent/src/ai	1.502s
```

### 3.2 `make lint`

```
$ cd backend/agent && make lint
go vet ./...
>> golangci-lint not found, installing...
>> installing golangci-lint v2.9.0
golangci/golangci-lint info found version: 2.9.0 for v2.9.0/darwin/arm64
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
```

The pinned version resolved to exactly `v2.9.0`, matching `database_administrator`. `gofmt -l .` reports no files.

### 3.3 The zero-dependency invariant survived the tooling install

This is worth an explicit check rather than an assumption: `make lint` runs `make tools`, which performs a `go install` of `goimports` and downloads four `golang.org/x/...` modules. None of that may reach `go.mod`.

```
$ cat backend/agent/go.mod
module github.com/cachicamas/backend/agent

go 1.26.3

$ ls backend/agent/go.sum
ls: go.sum: No such file or directory

$ git status --porcelain
(empty — bin/ is ignored)
```

`go install pkg@version` operates outside the current module, so it cannot mutate `go.mod`. Verified rather than assumed, because the guard's whole value rests on it.

---

## 4. Non-regression of the existing modules

### 4.1 Result sets

Both modules were baselined **before** the first file of this change was created (`go.work` did not yet exist), then re-run after all four phases.

| Module | Baseline | After | Verdict |
| --- | --- | --- | --- |
| `database_administrator` | 10 `ok`, 3 `[no test files]` | 10 `ok`, 3 `[no test files]` | identical |
| `workspace_syncer` | 8 `ok` | 8 `ok` | identical |

No package changed status, none disappeared, none was added. The only textual differences are timings and `(cached)` markers.

### 4.2 A qualification the plan required, and the reason it is not a failure

Doc 0002's AI-00.1 check 2 asks that `make test` in the other two modules be "unchanged from its pre-change output". Taken as a byte comparison this is unsatisfiable by construction: AI-00.4 **adds two tests** to `database_administrator`, so its output must change.

The comparison is therefore made on the **result set** — package, status, and the set of test names — and the expected delta is stated in advance rather than discovered: exactly two added tests, both passing, none removed, none newly failing.

```
+ TestDomainLayer_DoesNotImportAgentModule                    PASS
+ TestModule_OnlyApplicationAndCmdServerMayImportAgentModule  PASS
```

`workspace_syncer`'s result set is unchanged in the strict sense: zero added, zero removed.

### 4.3 The untouched-file preconditions

| Precondition | Verdict |
| --- | --- |
| `database_administrator/go.mod` gains no `replace` | **PASS** — file unmodified; `git diff` empty |
| `workspace_syncer/go.mod` gains no `replace` | **PASS** — file unmodified |
| `src/tools/tools.go` byte-identical | **PASS** — SHA-256 `79a1803a1c7e8e930d1ec34d0ead1b101004c4443ea462b96551fd27145f0cb6`, matching the pre-change hash |
| Changed files under `backend/database_administrator/` | exactly one: `src/domain/imports_test.go` |
| Changed files under `backend/workspace_syncer/` | none |

Notably **no `go.mod` was modified at any point**, including during the four bite proofs. `go.work` resolves cross-module imports without a `require` or `replace`, so the transient `go.mod` edits the tasks anticipated proved unnecessary. This is a stronger result than planned: the bites could not have left residue in a manifest they never touched.

---

## 5. Guard bite proofs

A guard that has only ever passed is indistinguishable from a broken one. Four bites recorded, each reverted immediately.

### 5.1 Forward guard — bite 1, sibling backend module

Scratch import of `…/database_administrator/src/domain` in `src/ai`:

```
--- FAIL: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault
    import_boundary_test.go:118: Layer 1 must not import "github.com/cachicamas/backend/database_administrator/src/domain"
          rule: ADR 0005 § D1 row 1: no package of another backend module
    import_boundary_test.go:122: Layer 1 must not import "gopkg.in/yaml.v3"
          rule: deny-by-default allowlist (ADR 0005 § D1 row 1) …
```

The second line is unplanned and valuable: `yaml.v3` is a **transitive** dependency of the scratch import, caught without being named. That is the transitivity blind spot ADR 0005 § Guard A requires closed, demonstrated rather than asserted.

### 5.2 Forward guard — bite 2, the OpenTelemetry SDK

Scratch import of `go.opentelemetry.io/otel/sdk/trace`:

```
    import_boundary_test.go:118: Layer 1 must not import "go.opentelemetry.io/otel/sdk"
          rule: ADR 0005 § D3: the OTel SDK belongs to a composition root
```

The § D3 split is visible in the failure mode itself, and this is the check most likely to be got wrong by a later maintainer:

- `otel/sdk/*`, `otel/exporters/*` → fail on a **named forbidden rule**. Permanent. No milestone unblocks them below the composition root.
- `otel/attribute`, `otel/codes`, `otel/trace` → fail on the **allowlist branch**. Temporary by design: AI-37 adds one allowlist entry and they pass, exactly as § D3 pre-authorises.

Two different failure branches for two different meanings is the property being verified.

### 5.3 Forward guard — bite 3, the deny-by-default proof

Scratch import of `gopkg.in/yaml.v3`, a module **no forbidden prefix names**:

```
    import_boundary_test.go:122: Layer 1 must not import "gopkg.in/yaml.v3"
          rule: deny-by-default allowlist (ADR 0005 § D1 row 1) — this path is neither
          the Go standard library nor a package of github.com/cachicamas/backend/agent.
```

**Line 122, not line 118.** This is the load-bearing detail. Failing on the allowlist branch proves the guard denies by default; had it failed on a named prefix, the guard would be a forbidden-substring scan wearing an allowlist's clothes, and AI-24's transport choice could arrive as a quiet `go get`. This bite is the one that would have to be rewritten if it ever passes.

### 5.4 Reverse guard — the bite, both halves at once

Scratch import `_ "github.com/cachicamas/backend/agent/src/ai"` in `src/domain`:

```
--- FAIL: TestDomainLayer_DoesNotImportAgentModule
    imports_test.go:87: the domain package must not import the agent module, by any path;
          found: "github.com/cachicamas/backend/agent/src/ai"
          rule: ADR 0005 § D1 row 6 …
--- FAIL: TestModule_OnlyApplicationAndCmdServerMayImportAgentModule
    imports_test.go:146: package ".../src/domain" must not import the agent module …
          rule: ADR 0005 § D1 row 5 — only [.../src/application .../src/cmd/server] may import …
```

Both halves fired, as doc 0002's AI-00.4 item 3 anticipates from a single `src/domain` violation. The module-scope test reports `inspected 13 packages`, so it is not passing vacuously.

---

## 6. Spec conformance

All requirements in `specs/agent-module-scaffold/spec.md` verified. Selected clauses where the verdict needed judgement rather than a command:

| Requirement | Verdict | Note |
| --- | --- | --- |
| R-AGM-002 — `test`/`lint` bodies pinned and load-bearing | **PASS** | Both recipes carry a comment naming them as the evidence gate for docs 0002/0003/0004, so a later divergence is visible |
| S-AGM-032 — `src/ai` has a package comment and **no declarations** | **PASS** | `go doc ./src/ai` shows the comment only |
| S-AGM-036 — the external test is a compile proof, not a behavior test | **PASS** | Blank import plus one empty test function; the file states in its own comment that AI-06.2 is the leaf that proves real external readability |
| S-AGM-030 — the three sibling layers are absent | **PASS** | `src/` holds exactly `ai` and `agenttest` |
| S-AGM-048 — the OTel **API** paths are absent from the forbidden table | **PASS** | Confirmed by inspection and by bite 2's branch behaviour (§ 5.2) |

---

## 7. Graph amendments landed with this change

Doc 0002's living-graph clause requires an amendment to land in the same PR that resumes work. Two landed, both corrections of stated fact, neither adding or renumbering a node.

### 7.1 AI-00.1 check 1 — the "empty pass" is not achievable

The check claims `make test` and `make lint` "run and pass inside the new module (trivially — there is nothing to test yet, and an empty pass is still evidence the tooling is wired)".

Measured on go1.26.3, a module containing no packages fails **both**:

```
$ go test ./...
go: warning: "./..." matched no packages
no packages to test          → exit 1

$ go vet ./...               → exit 1
```

A module with a single package holding only a doc comment exits 0. So the gate is achievable only once a package exists — which is AI-00.2's scope, not AI-00.1's.

**Resolution.** AI-00.1 retains `make build` and `make help` (both exit 0 on the empty module) plus the presence of the pinned recipes; the first green `make test` becomes AI-00.2 check 4. No node added, none renumbered, neither leaf's scope changed.

### 7.2 AI-00.3 test-list item 1 — the flag does not deliver the property claimed for it

The item specifies `go list -deps` "so it covers test imports and transitive dependencies". Bare `-deps` covers only the second.

```
$ go list      -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/domain   → 2
$ go list -test -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/domain   → 5
```

The three additional entries are the external test package, the synthesized test binary, and their closure. A Layer 1 **test file** importing a sibling backend module would pass a bare-`-deps` guard — the exact failure the leaf exists to prevent, and one of the two blind spots ADR 0005 § Guard A names.

**Resolution.** The guard uses `go list -deps -test` and normalizes the three synthesized shapes `-test` introduces (`pkg [pkg.test]`, `pkg_test [pkg.test]`, `pkg.test`); unnormalized, they are measured against the allowlist and the guard fails on its own module. Scope, dependencies and closing condition unchanged.

### 7.3 One design decision recorded without an amendment

The stdlib filter uses the toolchain's own `.Standard` field rather than exact set membership against `go list std`. `go list std` contains 17 `vendor/golang.org/x/...` entries which a "first path segment contains no dot" heuristic misclassifies — but `.Standard` reports `true` for them, so it is correct by construction and needs no second subprocess or maintained set. This changes no stated claim in doc 0002, so it is recorded in `design.md` rather than as an amendment.

---

## 8. Out-of-scope confirmations

Verified **not** done, each deliberate:

- No dependency of any kind, including OpenTelemetry. `go.mod` carries zero requires.
- No `replace` directive in any module.
- `database_administrator/src/tools/tools.go` untouched — S3 is resolved by vacating the name, not relocating the file.
- No `src/agent/`, `src/coding/` or `src/cmd/`, not even empty.
- ADR 0005 § D1 row 5 is **not** exercised. Permitted, unexercised, and priced elsewhere.
- ADR 0005 § Migration's stale cross-reference to milestone "AI-39" is recorded in `proposal.md` and **not** fixed here; amending the ADR is its own change.

---

## 9. Verdict

**PASS.** AI-00's charter is satisfied, both guards are proven biting, the two existing modules are unperturbed, and the two places where implementation disproved the plan are recorded as dated amendments rather than silent edits.

**Ready for archive** once the wave's PR merges.
