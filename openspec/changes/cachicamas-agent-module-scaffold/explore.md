# Explore — Create the `backend/agent` module and both boundary guards

> **Change**: `cachicamas-agent-module-scaffold`
> **Milestone**: AI-00 (Wave 0 — Found), the first milestone of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards)
> **Phase**: explore
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Decisions this change implements**: [ADR 0005 § D2](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2), [§ D3](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d3--observability-boundary), [§ Enforcement](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement)
> **Blocks**: every other milestone of doc 0002, and the first milestone of docs 0003 and 0004. There is nowhere to put a Layer 1 test until this merges.

---

## 1. Feature shape

AI-00 brings a new Go module into existence and closes both directions of the module boundary before a single line of contract code is written into it.

The module is `backend/agent/` (`github.com/cachicamas/backend/agent`, `go 1.26.3`, **zero requires**). It ships its own build tooling (`Makefile`, `.golangci.yml`, `.gitignore`, `README.md`), one package (`src/ai/`, package documentation only), one external test package (`src/agenttest/`, a compile-proof that `src/ai` is importable from outside), and a repo-root `go.work` that lists all three backend modules.

Two guards land with it:

- **Forward** (`backend/agent/src/ai/import_boundary_test.go`) — Layer 1 may reach nothing but the Go standard library and its own module. Deny-by-default.
- **Reverse** (`backend/database_administrator/src/domain/imports_test.go`) — nothing in `database_administrator` outside `src/application/` and `src/cmd/server` may name the agent module, and `src/domain` may not reach it even transitively.

The module is deliberately empty of behavior. Its whole value is that the *shape* is correct at birth: doc 0002 exists precisely because the previous attempt built seventeen milestones inside the wrong module and then had to unbuild them.

## 2. Current repo state (verified 2026-07-31 on this worktree)

| Fact | Evidence |
| --- | --- |
| `backend/agent/` does not exist | `ls backend/` → `database_administrator`, `workspace_syncer` |
| No `go.work` at the repo root | `ls go.work` → no such file |
| Two Go modules today | `backend/database_administrator/go.mod`, `backend/workspace_syncer/go.mod`, both `go 1.26.3` |
| No CI | `.github/workflows/` absent — matches [ADR 0005 § Enforcement](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement) |
| Local toolchain matches the target | `go version` → `go1.26.3 darwin/arm64` |
| `database_administrator` has 13 Go packages | `go list github.com/cachicamas/backend/database_administrator/... \| wc -l` → 13 |
| The reverse guard has a host file already | `backend/database_administrator/src/domain/imports_test.go`, `package domain_test`, already shells out to `go list -deps` |
| `src/tools/tools.go` is the only file under `src/tools/` | `ls backend/database_administrator/src/tools/` → `tools.go` |
| The root `.gitignore` does not exclude `go.work` | `grep go.work .gitignore` → no match |

There is no `src/tools/agent/` package anywhere on disk. The Layer 1 code the retired plan produced was removed before this branch; this change is not a migration, it is a creation. That is why [ADR 0005 § Migration](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#migration)'s advice about "the `git mv` and the import-path rewrite are separate commits" has no work to govern here — there is nothing to move.

## 3. What `database_administrator` gives us to copy

| Artifact | Path | What AI-00 takes from it |
| --- | --- | --- |
| `Makefile` | `backend/database_administrator/Makefile` | Target names and their bodies: `test` = `go test -race -v ./...`, `lint` = `vet` then `golangci-lint run --config=.golangci.yml ./...`, `fmt`, `build`, `tools`, `tidy`, `clean`, `help`. The `LOCALBIN := bin` convention and `GOLANGCI_LINT_VERSION := v2.9.0` pin. |
| `.golangci.yml` | `backend/database_administrator/.golangci.yml` (30 lines) | Verbatim: `version: "2"`, five linters (`govet`, `errcheck`, `staticcheck`, `unused`, `revive`), `run.tests: true`, 5m timeout. |
| `.gitignore` | `backend/database_administrator/.gitignore` | Verbatim: `bin/`, coverage artifacts, editor noise. |
| `go.mod` shape | `backend/database_administrator/go.mod` | Only the first two lines are relevant — `module …` and `go 1.26.3`. The 15-line `require` block is exactly what `backend/agent` must **not** have. |
| Guard idiom | `src/domain/imports_test.go` | The `runGoListDeps` helper: `exec.Command("go", "list", "-deps", importPath)` with stdout captured into a `strings.Builder`. Both new guards reuse the shape; neither reuses the *assertion*, which is a substring scan and is not deny-by-default. |

Not copied: `Dockerfile`, `.dockerignore`, `scripts/`, the `test/integration` target (it boots compose Postgres — meaningless for a module with no I/O), and `MAIN_PKG`/`BINARY_NAME` (AI-00 creates no `main` package; `src/cmd/cachicamas/` belongs to doc 0004).

## 4. The two guard mechanisms

### 4.1 Forward — Layer 1 purity (AI-00.3)

[ADR 0005 § Guard A](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement) records finding S6: the guard the retired plan shipped used `go list -f '{{range .Imports}}'`, which sees neither test imports nor transitive dependencies. Both blind spots were verified on this worktree rather than taken on trust:

```
$ go list -deps ./src/domain        | grep -c '^testing$'   → 0
$ go list -deps -test ./src/domain  | grep -c '^testing$'   → 1
```

**`go list -deps` alone does not close the test-import blind spot either.** Only `-deps -test` does. Doc 0002's test-list item 1 requires the guard to "cover test imports and transitive dependencies", so the guard command is `go list -deps -test <module pattern>/...`, not bare `-deps`. This is a refinement of the ADR's wording, not a contradiction of it: the ADR names the mechanism (`-deps` plus an allowlist), and `-test` is what makes the named mechanism deliver the property the ADR asks for.

`-deps -test` emits synthesized packages that are not real imports and must be normalized away. Observed shapes:

```
…/src/domain [github.com/…/src/domain.test]      ← test-variant of a real package
…/src/domain_test [github.com/…/src/domain.test] ← the external test package
…/src/domain.test                                ← the synthesized test binary
```

### 4.2 Reverse — nothing reaches back (AI-00.4)

Two assertions, and they must use **different** `go list` modes. This is the finding that most affects the design:

- The `src/domain` assertion is **transitive** (`-deps`): domain must not reach the agent module by any path. Nothing in `database_administrator` imports domain-upward, so there is no false-positive risk.
- The module-scope assertion must be **direct-imports only** (`.Imports` + `.TestImports` + `.XTestImports`). [ADR 0005 § D1 row 5](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) *permits* `src/application` and `src/cmd/server` to import the agent module. The moment row 5 is exercised, every package that depends on `application` — `interfaces/http`, `cmd/server` — inherits the agent module transitively. A transitive module-scope guard would then fail on packages that did nothing wrong, and the natural fix would be to weaken the guard. Direct-import scanning is correct for a "who may *name* this module" rule.

Verified that the direct-import listing is available in one command:

```
go list -f '{{.ImportPath}}|{{join .Imports " "}}|{{join .TestImports " "}}|{{join .XTestImports " "}}' \
  github.com/cachicamas/backend/database_administrator/...
```

Today it returns 13 lines and zero occurrences of the agent module, so the module-scope guard is green from birth. [ADR 0005 § Guard B](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#enforcement) states that this is the point: it fails on the first accidental import rather than on the first production incident.

## 5. Constraints carried in from the ADR and the task graph

1. **Zero requires.** No dependency, not even OpenTelemetry. The OTel API arrives at AI-37 (pre-authorized by ADR 0005 § D3); the transport arrives at AI-24 behind its own ADR gate. `backend/agent/go.mod` is two directives long and there is no `go.sum`.
2. **`src/agenttest/` is a direct sibling of `src/ai/`.** [ADR 0005 § D2](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d2--location-mapping-v2) and Guard C: AI-20.4's signature guard will resolve `../ai/provider.go` from `runtime.Caller(0)`. Any other layout breaks it silently, years from now.
3. **Do not create `src/agent/`, `src/coding/`, `src/cmd/`** — not even empty. Doc 0002 AI-00.2 check 3: creating them makes the forward guard's forbidden-prefix list untestable, because a prefix that matches a package that exists is no longer a *forbidden* prefix.
4. **No `replace` directive** in `database_administrator/go.mod`. [ADR 0005 § Migration](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#migration): a `replace` with no requirer is dead weight, and it disguises the D1 row-5 build cost (the Dockerfile copies only `./backend/database_administrator`; the first cross-module import breaks `docker compose build`).
5. **`database_administrator/src/tools/tools.go` stays byte-identical.** S3 is resolved by vacating the name `tools`, not by relocating a standard-idiom file.
6. **Strict TDD is on** (`openspec/AGENTS.md`). Guard leaves are RED-first by construction — a guard that has never failed is not a guard. Mechanical leaves are exempt from red-green but never from their recorded Check list.
7. **No CI.** "Recorded green" means output pasted into the PR description. After this change there are three module directories in which `make test` must be run by hand.

## 6. Open questions and their resolutions

| # | Question | Resolution | Why |
| --- | --- | --- | --- |
| 1 | Is bare `go list -deps` enough for the forward guard? | **No — use `-deps -test`.** | Measured: `-deps` returns 0 hits for `testing` on a package with tests; `-deps -test` returns 1. Doc 0002 item 1 demands test-import coverage. |
| 2 | How is "stdlib" decided — the exact set from `go list std`, or the "first path segment has no dot" heuristic? | **The exact set from `go list std`.** | The heuristic is wrong here. `go list std` contains 17 vendored paths such as `vendor/golang.org/x/crypto/chacha20`, and those *do* appear verbatim in real `-deps` output (confirmed against `src/interfaces/...`). A dot heuristic would flag them as third-party; a `vendor/` special case would then be a second heuristic layered on the first. Exact-set membership has no such edge. |
| 3 | Deny-by-default says "stdlib + own module only" — but ADR 0005 § D3 says the OTel **API** is *allowed* for Layer 1. Which wins at AI-00? | **Both, expressed as three lists.** The allowlist has a stdlib group, an own-module group, and an explicitly **empty** vendor group whose comment names ADR 0005 § D3, AI-24 and AI-37 as the only ways to add an entry. The D3 split is additionally encoded in the forbidden-prefix list (OTel SDK, exporters, `otelslog`), so that a future over-broad `go.opentelemetry.io/` entry in the vendor group still cannot smuggle the SDK in. | At AI-00 the module has zero requires, so no OTel path can appear at all; making the vendor group empty-but-named is what turns AI-24's transport choice into a visible, ADR-gated event rather than a quiet `go get`. Doc 0002 item 2 requires the D3 split to be *named*, which the forbidden list does, and item 3 requires deny-by-default, which the allowlist does. |
| 4 | Where does the module-scope reverse guard live? `src/domain/imports_test.go`, a new `src/architecture/` package, or `src/cmd/server/`? | **`src/domain/imports_test.go`.** | A new `src/architecture/` directory holding only `_test.go` files would need a non-test file to keep `go build ./...` working — AI-00.1 check 2 requires that build to succeed — so it would mean inventing a production file with no purpose. `src/cmd/server/` is one of the two *permitted* importers, which is a confusing home for the guard that polices them. `domain_test` already owns the `go list` helper, needs no new package, and keeps the whole reverse guard in one reviewable file. If a later change grows a real architecture-test package, moving it is a mechanical rename. |
| 5 | Does the module-scope guard scan transitively? | **No — direct imports only.** | See § 4.2. Row 5 permits `application` to import the agent module; a transitive scan would then indict every consumer of `application`. |
| 6 | Does adding a root `go.work` change what `make test` does in the two existing modules? | **It can, and the acceptance requires proving it does not.** | Workspace mode computes one build list across all listed modules and verifies it against `go.work.sum`. `backend/agent` has zero requires and contributes nothing to MVS, and the other two pin the same versions of their shared dependencies, so no bump is expected — but "expected" is not evidence. A pre-change baseline of `make test` in both modules is captured before `go.work` lands, and compared after. If the toolchain writes a `go.work.sum`, it is committed. |
| 7 | "`make test` … unchanged in the other two" — unchanged how? Its output changes the moment AI-00.4 adds tests. | **Unchanged in the result set, and the comparison point differs per leaf.** After AI-00.1/AI-00.2 both modules must produce the same set of test names with the same outcomes as the baseline (timings excepted). After AI-00.4, `database_administrator`'s output must differ from the baseline **only** by the added reverse-guard tests; `workspace_syncer` must still match exactly. | Doc 0002 places the "unchanged" check in AI-00.1, before the reverse guard exists. Stating the comparison point per leaf is the difference between a checkable claim and an impossible one. |
| 8 | Should `src/agenttest/` hold a real assertion at AI-00? | **No — a compile-proof only.** | AI-00.2 check 2 asks for "an external-package test that imports `src/ai` and compiles". `src/ai` exports nothing yet; AI-01 names the vocabulary and AI-06.2 is the leaf that proves external readability of a real type. Asserting anything more here would be inventing contract surface a milestone earlier than the plan allows. |
| 9 | Does the `Makefile` copy include `build` and `run`? | **`build` yes (parameterized to nothing today), `run` no.** | There is no `main` package until doc 0004's `src/cmd/cachicamas/`. `build` is kept as `go build ./...` so the target name stays stable when a binary arrives; `run` would be a target that cannot run. |

## 7. Risk register (initial)

| # | Risk | Likelihood | Mitigation |
| --- | --- | --- | --- |
| 1 | `go.work` perturbs the two existing modules' build lists | low | Baseline `make test` in both before the change; re-run after; commit `go.work.sum` if generated (open question 6). |
| 2 | Build config drift between three copied `Makefile`s / `.golangci.yml`s | medium (certain over time) | Priced and accepted by ADR 0005 § D2 and doc 0002's charter note. Mitigated by *naming* the load-bearing targets in the spec, so a divergence in `test` or `lint` is a spec violation rather than an unnoticed edit. |
| 3 | The forward guard passes vacuously — a module with one doc-only package has almost no dependency graph | high | This is exactly why AI-00.3 is a `[guard]` leaf and not a `[mechanical]` one: three scratch violations must each be shown to make it fail, with the red output recorded. A guard that has only ever been green is unproven. |
| 4 | The scratch violations get committed by accident | low | They are added, run, recorded, and dropped inside the leaf. The PR diff is the check: no `_scratch` file and no `require` line may appear in it. |
| 5 | The third scratch violation (an arbitrary third-party import) needs a `require` line, which temporarily violates the zero-requires rule | medium | It is transient and never committed. The recorded red output plus a clean `git status` and a final `go.mod` with zero requires are the evidence. |
| 6 | Someone later creates `src/agent/` as an empty dir "to be ready" | medium | AI-00.2 check 3 states the reason in the spec, and the forward guard's forbidden-prefix list is the thing that breaks. |
| 7 | The wave ships as one PR and exceeds doc 0002's review budget | certain (accepted) | Explicit user decision. Doc 0001 § 9 Process requires the PR description to say why it does not fit; `tasks.md` carries the wording. |

## 8. Related paths

Created by this change:

- `backend/agent/{go.mod,Makefile,.golangci.yml,.gitignore,README.md}`
- `backend/agent/src/ai/doc.go`
- `backend/agent/src/ai/import_boundary_test.go` — forward guard
- `backend/agent/src/agenttest/{doc.go,import_compile_test.go}` — external-consumer compile proof
- `go.work` (repo root)

Modified by this change:

- `backend/database_administrator/src/domain/imports_test.go` — reverse guard, both halves

Deliberately untouched:

- `backend/database_administrator/go.mod` (no `replace`)
- `backend/database_administrator/src/tools/tools.go` (byte-identical, diff recorded)
- `backend/workspace_syncer/**` (listed in `go.work`, otherwise unchanged)
- `docker-compose.yaml`, `infra/`, `Dockerfile`s — [ADR 0005 § D1 row 5](../../../docs/adr/0005-promote-agent-stack-to-own-module.md#d1--dependency-rule-v2) prices that work; it is not this change's.

## 9. Skills to load in later phases

- `go-testing` — the two guards are the only Go code in this change and both are tests.
- `test-driven-development` — RED-first is the closing condition for both guard leaves.
- `work-unit-commits` — four leaves, four commits, one PR.
- `cognitive-doc-design` — `backend/agent/README.md` is review-facing and is a chartered deliverable.
