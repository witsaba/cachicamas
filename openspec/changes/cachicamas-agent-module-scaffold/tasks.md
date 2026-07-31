# Tasks — Agent module scaffold and boundary guards

> **Change**: `cachicamas-agent-module-scaffold`
> **Milestone**: AI-00 of [doc 0002](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards)
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-07-31
> **Branch**: `feat/2026-07-31-cachicamas-ai-layer1-wave-0`
> **Inputs**: `explore.md`, `proposal.md`, `specs/agent-module-scaffold/spec.md`, `design.md`
> **Delivery**: **one PR**, four commits — one per leaf of AI-00's graph
> **Forecast**: ~530 changed lines. **Over budget by explicit user decision** — see [Review budget exception](#review-budget-exception)
> **Evidence gate**: `make test` in `backend/agent/`. **AI-00.4 is the one documented exception** — its gate is `make test` in `backend/database_administrator/`

---

## Phase map

The four leaves of AI-00's graph are the four phases. The dependency shape is doc 0002's, unchanged:

```
AI-00.1 module skeleton  [mechanical]
        │
        ▼
AI-00.2 package layout   [mechanical]
        │
        ├──────────────┐
        ▼              ▼
AI-00.3 forward     AI-00.4 reverse        ← PARALLEL, disjoint modules
        guard [guard]        guard [guard]
```

| Phase | Leaf | Type | Module touched | Gate | Commit |
| --- | --- | --- | --- | --- | --- |
| 0 | — | prerequisite | both existing | — | (none — captures baselines) |
| 1 | AI-00.1 | `[mechanical]` | `backend/agent` (new), repo root | recorded Check list | `chore(agent): create the module skeleton and go.work` |
| 2 | AI-00.2 | `[mechanical]` | `backend/agent` | recorded Check list | `chore(agent): add src/ai and its src/agenttest sibling` |
| 3 | AI-00.3 | `[guard]` | `backend/agent` | `make test` in `backend/agent/` | `test(agent): guard Layer 1 import purity` |
| 4 | AI-00.4 | `[guard]` | `backend/database_administrator` | `make test` in `backend/database_administrator/` | `test(database_administrator): guard against reaching into the agent module` |

**Phases 3 and 4 are explicitly parallel.** They touch disjoint modules and share no file. Two people, or two agents, can take them concurrently once phase 2 lands; if they are worked serially the order does not matter.

**Mechanical leaves are exempt from red-green, never from their Check lists** (doc 0002, node grammar). Phases 1 and 2 therefore state a Check list instead of a RED step, and each check names the recorded evidence that closes it. Phases 3 and 4 are guard leaves: RED-first is their closing condition, because a guard that has only ever been green is unproven.

---

## Review budget exception

Doc 0002's milestone rules say *prefer under 250 changed lines; stop and reassess before 400*. This change ships as **one PR** at a forecast ~530 lines, by explicit user decision. [Doc 0001 § 9 (Process)](../../../docs/architecture/0001-cachicamas-agent-stack-v2.md#9-review-checklist) requires the PR description to say why a change does not fit. Paste this into the PR description:

> **Why this exceeds the review budget.** AI-00 is not decomposable into shippable slices. Its four leaves form a strict chain — the module must exist before a package can go in it, and a package must exist before a guard can scan it — and none of the three intermediate states is mergeable on its own. A PR with `go.mod` and a `Makefile` but no package passes its own acceptance vacuously; a PR with the package but neither guard merges an unguarded module, which is the exact failure ADR 0005 exists to prevent.
>
> The line count also overstates the review cost. About 220 lines are build configuration copied from `backend/database_administrator` and reviewable by diffing against the source (`.golangci.yml` and `.gitignore` are byte-identical). About 260 are two test files with no production call path, each proven by a recorded failing run before it lands green. The genuinely novel production surface is 3 lines of `go.mod`, 5 of `go.work`, and two package doc comments.
>
> The PR is four commits, one per leaf. The two guard commits touch disjoint modules and can be reviewed independently of each other.

---

## Phase 0 — Baselines (prerequisite, no commit)

Must run **before the first file of this change is created**. Once `go.work` exists a pre-change run is no longer available without stashing, and detecting exactly the perturbation `go.work` might introduce is the entire purpose of the baseline.

### T-AGM-0-001 — Capture the pre-change `make test` baselines

Run and save the full output of both:

```bash
cd backend/database_administrator && make test 2>&1 | tee /tmp/agm-baseline-dba.txt
cd backend/workspace_syncer      && make test 2>&1 | tee /tmp/agm-baseline-ws.txt
```

**Evidence:** both files saved; the test-name/outcome result set extracted from each for later comparison. Both go in the PR description.
**Closes:** S-AGM-060.
**Strict TDD:** N/A (no code).

### T-AGM-0-002 — Record the untouched-file preconditions

```bash
git -C . rev-parse HEAD                                          # the comparison base
grep -n replace backend/database_administrator/go.mod || echo "no replace (expected)"
sha256sum backend/database_administrator/src/tools/tools.go
grep -n "go.work" .gitignore || echo "go.work not ignored (expected)"
```

**Evidence:** the base SHA (used by S-AGM-064's diff), the confirmed absence of a `replace`, the `tools.go` hash, and the confirmed absence of a `go.work` ignore rule.
**Closes:** groundwork for S-AGM-023, S-AGM-064, S-AGM-065.

---

## Phase 1 — AI-00.1 module skeleton `[mechanical]`

**Commit:** `chore(agent): create the module skeleton and go.work`
**Depends on:** phase 0.
**Out of scope:** any package content (phase 2); any dependency, ever.
**Forecast:** ~230 lines.

### T-AGM-1-001 — Create `backend/agent/go.mod`

Three lines: `module github.com/cachicamas/backend/agent`, blank, `go 1.26.3`. No `require`, no `replace`, no `toolchain`, and no `go.sum`.

**Check (mechanical — no RED step):**
- `backend/agent/go.mod` contains exactly one `module` directive, one `go` directive with value `1.26.3`, and zero `require`/`replace` directives.
- `ls backend/agent/go.sum` reports no such file.

**Evidence command:** `cat backend/agent/go.mod && ls backend/agent/go.sum 2>&1`
**Closes:** S-AGM-001, S-AGM-002.

### T-AGM-1-002 — Copy `.golangci.yml` and `.gitignore` verbatim

```bash
cp backend/database_administrator/.golangci.yml backend/agent/.golangci.yml
cp backend/database_administrator/.gitignore    backend/agent/.gitignore
```

**Check:**
- `diff backend/agent/.golangci.yml backend/database_administrator/.golangci.yml` is empty.
- `backend/agent/.gitignore` ignores `bin/` and the coverage artifacts, so `make tools` never dirties the tree.

**Evidence command:** `diff backend/agent/.golangci.yml backend/database_administrator/.golangci.yml && cat backend/agent/.gitignore`
**Closes:** S-AGM-014, S-AGM-015.

### T-AGM-1-003 — Write `backend/agent/Makefile`

Derive from `backend/database_administrator/Makefile`. Keep `LOCALBIN := bin`, `GOLANGCI_LINT_VERSION := v2.9.0`, the `export PATH` prepend, `GOFLAGS := -trimpath`, and the `help` awk target. Keep the targets `help`, `tools`, `tidy`, `build`, `test`, `test/cover`, `fmt`, `vet`, `lint`, `clean`, `all`.

Two targets are **load-bearing for every evidence gate in docs 0002/0003/0004** and their bodies are pinned by R-AGM-002:

- `test` → `$(GO) test -race -v ./...`
- `lint` → `vet` then `$(GOLANGCI_LINT) run --config=.golangci.yml ./...` at the pinned `v2.9.0`

Drop `run` (no `main` package until doc 0004), `install` (zero requires — nothing to download), `test/integration` and its compose variables (Layer 1 has no I/O and no container), and `MAIN_PKG`/`BINARY_NAME` (`build` becomes `go build ./...`, keeping the target name stable for when a binary arrives).

**Check:**
- The `test` recipe expands to `go test -race -v ./...`.
- The `lint` recipe invokes `golangci-lint run --config=.golangci.yml ./...` and the pinned version resolves to `v2.9.0`.
- No repository-wide shared build or lint config is introduced; each module still owns its own files, and none includes another's.

**Evidence command:** `cd backend/agent && make help && grep -A2 '^test:' Makefile && grep GOLANGCI_LINT_VERSION Makefile`
**Closes:** S-AGM-010, S-AGM-011, S-AGM-017.

### T-AGM-1-004 — Write `backend/agent/README.md`

Must state, per doc 0002's AI-00.1 check 4 and R-AGM-002: the module's one-paragraph purpose; the three-layer contents (`src/ai` Layer 1 ← `src/agent` Layer 2 ← `src/coding` Layer 3 ← `src/cmd/cachicamas`) noting that only Layer 1 exists today; the dependency rule with its ADR 0005 § D1/§ D3 reference; and — explicitly — that `make test` must be run in this directory because no CI exists.

Also record why `src/agenttest/` must stay a direct sibling of `src/ai/`, and why `src/agent/`, `src/coding/` and `src/cmd/` are deliberately absent. Both facts are load-bearing and the README is the first place a newcomer looks.

**Check:** all four required elements present; the layered (not hexagonal) structure is stated, matching `openspec/AGENTS.md` § 2.
**Evidence command:** `cat backend/agent/README.md`
**Closes:** S-AGM-016.

### T-AGM-1-005 — Create the repo-root `go.work`

```
go 1.26.3

use (
	./backend/agent
	./backend/database_administrator
	./backend/workspace_syncer
)
```

If the toolchain writes a `go.work.sum`, commit it. Confirm the root `.gitignore` excludes neither.

**Check:**
- `go.work` names exactly the three module directories and declares `go 1.26.3`.
- `go build ./...` exits 0 from each of the three module directories.
- Any generated `go.work.sum` is staged.
- `git status --porcelain` reports no untracked `go.work*` after a full test run in all three modules.

**Evidence command:**
```bash
cat go.work
for m in agent database_administrator workspace_syncer; do (cd backend/$m && go build ./... && echo "OK $m"); done
git status --porcelain | grep -E 'go\.work' || echo "no untracked go.work* (expected)"
```
**Closes:** S-AGM-003, S-AGM-020, S-AGM-021, S-AGM-022, S-AGM-023.

### T-AGM-1-006 — Prove the existing modules did not move

Re-run `make test` in both existing modules and compare the result set against the phase-0 baselines. At this point no test has been added to either, so the sets must match exactly (timings excepted). This is the check that catches a `go.work`-induced build-list change.

**Check:**
- `database_administrator` result set == baseline.
- `workspace_syncer` result set == baseline.
- `backend/database_administrator/go.mod` still has no `replace`.
- `git diff <base> -- backend/database_administrator/src/tools/tools.go` is empty.

**Evidence command:**
```bash
cd backend/database_administrator && make test 2>&1 | tee /tmp/agm-after-p1-dba.txt
cd backend/workspace_syncer      && make test 2>&1 | tee /tmp/agm-after-p1-ws.txt
git diff <base>..HEAD -- backend/database_administrator/src/tools/tools.go
grep -n replace backend/database_administrator/go.mod || echo "no replace (expected)"
```
**Closes:** S-AGM-061, S-AGM-064, S-AGM-065.

### Phase 1 exit

- [ ] `cd backend/agent && make test` exits 0 (an empty pass is valid evidence that the tooling is wired).
- [ ] `cd backend/agent && make lint` exits 0.
- [ ] `go.mod` has zero requires; no `go.sum`.
- [ ] Both existing modules match their baselines.
- [ ] Recorded outputs for every check above are in the PR description.

---

## Phase 2 — AI-00.2 package and test-package layout `[mechanical]`

**Commit:** `chore(agent): add src/ai and its src/agenttest sibling`
**Depends on:** phase 1.
**Out of scope:** any exported declaration in `src/ai` (AI-01 names the vocabulary, AI-04 declares the first type); either guard (phases 3 and 4).
**Forecast:** ~60 lines.

### T-AGM-2-001 — Create `backend/agent/src/ai/doc.go`

Package documentation **only**: the Layer 1 boundary and the import rule, citing ADR 0005 § D1. No type, no constant, no function, no variable. Doc 0002 is explicit that the contract text grows one milestone at a time, and each milestone's doc paragraph is guarded where it makes a checkable claim — so this paragraph must claim only what the forward guard actually checks.

**Check:**
- `go doc github.com/cachicamas/backend/agent/src/ai` shows the package comment and no declarations.
- The comment states the layer boundary and the import rule and cites ADR 0005 § D1.

**Evidence command:** `cd backend/agent && go doc ./src/ai`
**Closes:** S-AGM-031, S-AGM-032.

### T-AGM-2-002 — Create `backend/agent/src/agenttest/` as a direct sibling

Two files:

- `doc.go` — `package agenttest`, whose comment states *why this directory must remain a direct sibling of `src/ai/`*: AI-20.4's signature guard will resolve `../ai/provider.go` from `runtime.Caller(0)` (ADR 0005 § D2, Guard C), and any other layout breaks it silently.
- `import_compile_test.go` — `package agenttest_test`, importing `github.com/cachicamas/backend/agent/src/ai` (blank identifier is fine — `src/ai` exports nothing yet) plus one trivial test function so the file is a real test. This is a **compile proof**, not a behavior test. AI-06.2 is the leaf that proves external readability of a real type.

**Check:**
- `backend/agent/src/agenttest/` and `backend/agent/src/ai/` share the parent `backend/agent/src`, so `../ai` resolves from any file in `agenttest`.
- At least one file declares `package agenttest_test` and imports the Layer 1 package.
- `go test ./src/agenttest/...` compiles and exits 0.

**Evidence command:** `cd backend/agent && ls src/ && go test -v ./src/agenttest/...`
**Closes:** S-AGM-033, S-AGM-034, S-AGM-036.

### T-AGM-2-003 — Prove the three sibling layers do not exist

Do **not** create `src/agent/`, `src/coding/` or `src/cmd/` — not empty, not with a placeholder. Doc 0002's AI-00.2 check 3: creating any of them makes the forward guard's forbidden-prefix list untestable, because a prefix that also matches a legitimate package of this module can no longer be shown to be forbidden.

**Check:**
- `backend/agent/src/` contains exactly two entries, `ai` and `agenttest`.
- Each of `src/agent`, `src/coding`, `src/cmd` fails an existence check.

**Evidence command:**
```bash
ls backend/agent/src/
for d in agent coding cmd; do test -e backend/agent/src/$d && echo "FAIL: src/$d exists" || echo "OK: src/$d absent"; done
```
**Closes:** S-AGM-030, S-AGM-035.

### Phase 2 exit

- [ ] `cd backend/agent && make test` exits 0, and now runs at least one real (compile-proof) test.
- [ ] `cd backend/agent && make lint` exits 0.
- [ ] `src/` holds exactly `ai` and `agenttest`.
- [ ] `go.mod` still has zero requires.

---

## Phase 3 — AI-00.3 forward guard `[guard]` — **parallel with phase 4**

**Commit:** `test(agent): guard Layer 1 import purity`
**Depends on:** phase 2. **Parallel with:** phase 4 (disjoint module, no shared file).
**Gate:** `make test` in `backend/agent/`.
**Out of scope:** the reverse direction (phase 4); AI-25.2's ambient-authority scan over the adapter package.
**Forecast:** ~130 lines.

> **This leaf does not close on green.** It closes on three recorded reds followed by green. A guard that has only ever passed is indistinguishable from a broken one.

### T-AGM-3-001 — RED: bite 1, an import of a sibling backend module

**RED step.** Before the guard exists, write it, then immediately prove it bites. Add a scratch file `backend/agent/src/ai/scratch_violation_test.go` importing `github.com/cachicamas/backend/database_administrator/src/domain`, with the transient `require` and `replace ../database_administrator` in `go.mod` needed to resolve it.

**Expected red:** the guard FAILS, naming the offending import path and the forbidden-prefix rule that rejected it (sibling backend module, ADR 0005 § D1 row 1).

**GREEN step:** remove the scratch file and revert the `go.mod` edits. The guard passes.

**Evidence command:** `cd backend/agent && go test -v ./src/ai/... 2>&1 | tee /tmp/agm-bite1-red.txt` (red), then the same after removal (green).
**Closes:** S-AGM-044.

### T-AGM-3-002 — RED: bite 2, an import of the OpenTelemetry SDK

Same protocol, importing `go.opentelemetry.io/otel/sdk/trace`.

**Expected red:** the guard FAILS on the forbidden-prefix rule for `go.opentelemetry.io/otel/sdk`, citing ADR 0005 § D3 — the SDK is a composition-root concern.

**Evidence command:** `cd backend/agent && go test -v ./src/ai/... 2>&1 | tee /tmp/agm-bite2-red.txt`
**Closes:** S-AGM-045.

### T-AGM-3-003 — RED: bite 3, an arbitrary third-party import (the deny-by-default proof)

Same protocol, importing any third-party module named by **no** forbidden prefix.

**Expected red — and this one is specific:** the guard must FAIL **on the allowlist branch**, not on a named prefix. That failure is the only proof that the allowlist is deny-by-default, and therefore the only proof that AI-24's transport choice cannot arrive as a quiet `go get`. If bite 3 passes, the guard is a forbidden-substring scan wearing an allowlist's clothes and must be rewritten.

**GREEN step:** remove the scratch file **and** the `require` line its presence forced into `go.mod`.

**Evidence command:** `cd backend/agent && go test -v ./src/ai/... 2>&1 | tee /tmp/agm-bite3-red.txt`, then `cat backend/agent/go.mod` showing zero requires and `git status --porcelain` showing a clean tree.
**Closes:** S-AGM-046, S-AGM-047.

### T-AGM-3-004 — GREEN: the guard itself

`backend/agent/src/ai/import_boundary_test.go`, `package ai_test`. Implement per `design.md` § 3.6:

1. `stdlib` ← the exact set from `go list std` (**not** a "first segment has no dot" heuristic — `go list std` contains 17 `vendor/golang.org/x/...` paths that appear verbatim in real `-deps` output, and the heuristic mishandles them).
2. `deps` ← `go list -deps -test github.com/cachicamas/backend/agent/...`. Both flags are required: `-deps` alone does not report test-only imports (measured: 0 hits for `testing` without `-test`, 1 with it).
3. Normalize: truncate each line at the first ` [`; drop paths ending in `.test`; drop our own `_test` external package; deduplicate.
4. Check the **forbidden-prefix list first** — `…/agent/src/agent` is forbidden while `…/agent/` is allowed, so an allowlist-first pass would admit Layer 2.
5. Then the deny-by-default allowlist: stdlib set, own-module prefix, and a **present-but-empty** third-party group commented with its only two growth paths (AI-24 transport behind its own ADR; AI-37 OTel API pre-authorized by ADR 0005 § D3).

Forbidden prefixes, all eight: both sibling backend modules; `…/agent/src/agent`, `…/agent/src/coding`, `…/agent/src/cmd`; `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters`, `go.opentelemetry.io/contrib/bridges/otelslog`. The OTel **API** paths must **not** appear in this list — forbidding them would make AI-37 impossible without weakening the guard.

Use the fully-qualified module pattern, never `./...`: `go test` sets cwd to the package under test, so a relative pattern would silently scope the guard to `src/ai` alone. Failure messages must name the offending path *and* the rule that rejected it, and the guard must cite the ADR clause it enforces in a comment.

**Check:** `go test ./src/ai/...` passes; the source satisfies S-AGM-041, S-AGM-042, S-AGM-043, S-AGM-048 on inspection.
**Evidence command:** `cd backend/agent && make test`
**Closes:** S-AGM-004, S-AGM-040, S-AGM-041, S-AGM-042, S-AGM-043, S-AGM-048.

### T-AGM-3-005 — REFACTOR

Clean up while green: extract the `go list` runner and the normalizer as unexported test helpers; keep the forbidden table and the three allowlist groups as data, not control flow, so the next milestone edits a table rather than a function. Re-run `make test` and `make lint`.

### Phase 3 exit

- [ ] Three recorded reds in the PR description, one per bite, with bite 3 failing on the allowlist branch.
- [ ] `cd backend/agent && make test` green; `make lint` clean.
- [ ] No scratch file in the diff; `go.mod` has zero requires; `git status` clean.

---

## Phase 4 — AI-00.4 reverse guard `[guard]` — **parallel with phase 3**

**Commit:** `test(database_administrator): guard against reaching into the agent module`
**Depends on:** phase 2. **Parallel with:** phase 3 (disjoint module, no shared file).
**Gate exception:** this leaf's gate is `make test` in **`backend/database_administrator/`**, not in `backend/agent/`. It is the single documented exception to doc 0002's global evidence gate.
**Out of scope:** exercising the permitted ADR 0005 § D1 row-5 import — a non-goal for all of v1, and one that breaks `docker compose build` (ADR 0005 prices it as its own change).
**Forecast:** ~110 lines, all in one existing file.

### T-AGM-4-001 — RED: the bite

**RED step.** Add a scratch import `_ "github.com/cachicamas/backend/agent/src/ai"` to a file in `backend/database_administrator/src/domain/`, with the transient `require` and `replace ../agent` in that module's `go.mod` needed to resolve it.

**Expected red:** `cd backend/database_administrator && make test` FAILS, and the message names both the offending package and the forbidden module path.

**GREEN step:** remove the scratch import **and** both `go.mod` edits. `make test` passes.

Note this single bite exercises both halves at once — a `src/domain` import is caught transitively by half one and directly by half two. Convenient, but it is one bite, which is what doc 0002's AI-00.4 item 3 asks for.

**Evidence command:** `cd backend/database_administrator && make test 2>&1 | tee /tmp/agm-reverse-bite-red.txt`, then `git diff -- backend/database_administrator/go.mod` showing no change.
**Closes:** S-AGM-055, S-AGM-056.

### T-AGM-4-002 — GREEN, half one: extend the existing domain forbidden-prefix test

In `backend/database_administrator/src/domain/imports_test.go`, add `TestDomainLayer_DoesNotImportAgentModule` alongside the existing pgx test, reusing the file's `runGoListDeps` helper. **Transitive** (`go list -deps`) over `…/database_administrator/src/domain`, forbidding the prefix `github.com/cachicamas/backend/agent`. ADR 0005 § D1 row 6: domain must not reach the agent module by any path, and there is no false-positive risk because nothing imports domain downward.

**Check:** the test passes today (green from birth) and the source satisfies S-AGM-051 on inspection.
**Evidence command:** `cd backend/database_administrator && go test -v -run 'TestDomainLayer' ./src/domain/...`
**Closes:** S-AGM-051.

### T-AGM-4-003 — GREEN, half two: the module-scope direct-import test

Same file. `TestModule_OnlyApplicationAndCmdServerMayImportAgentModule`, per `design.md` § 4.3:

```
go list -f '{{.ImportPath}}|{{join .Imports " "}}|{{join .TestImports " "}}|{{join .XTestImports " "}}' \
        github.com/cachicamas/backend/database_administrator/...
```

For every package whose path is not prefixed by `…/src/application` or `…/src/cmd/server`, assert that none of its direct, test, or external-test imports has the prefix `github.com/cachicamas/backend/agent`.

**It must NOT pass `-deps`, and the reason belongs in a comment *and* in the failure message.** ADR 0005 § D1 row 5 permits `src/application` and `src/cmd/server` to import the agent module. The moment row 5 is exercised, every consumer of `application` inherits the agent module transitively, so a transitive guard would fail on packages that did nothing wrong — and the path of least resistance would be to widen the exemption list until the guard means nothing. A future maintainer who sees this fail and reaches for `-deps` must read why that is the wrong fix at the moment they consider it.

**Check:**
- The test inspects every package of the module (13 today) and finds zero naming the agent module.
- The exemption set is exactly the two permitted prefixes and nothing else.
- The source satisfies S-AGM-052 and S-AGM-053 on inspection.

**Evidence command:** `cd backend/database_administrator && go test -v -run 'TestModule_' ./src/domain/...`
**Closes:** S-AGM-050, S-AGM-052, S-AGM-053, S-AGM-054.

### T-AGM-4-004 — Prove non-regression of both existing modules

Re-run `make test` in both and compare against the phase-0 baselines.

**Check:**
- `database_administrator`'s result set == baseline **plus exactly** the two reverse-guard tests, all passing, with no removed or newly failing test.
- `workspace_syncer`'s result set == baseline, exactly.
- The changed-file set under `backend/database_administrator/` is exactly `src/domain/imports_test.go`.
- The changed-file set under `backend/workspace_syncer/` is empty.

**Evidence command:**
```bash
cd backend/database_administrator && make test 2>&1 | tee /tmp/agm-final-dba.txt
cd backend/workspace_syncer      && make test 2>&1 | tee /tmp/agm-final-ws.txt
git diff --name-only <base>..HEAD -- backend/database_administrator/ backend/workspace_syncer/
```
**Closes:** S-AGM-062, S-AGM-063, S-AGM-066, S-AGM-067.

### T-AGM-4-005 — REFACTOR

Clean up while green: keep the two new tests and the pre-existing pgx test sharing one `go list` helper set; add the file-level comment recording that the module-scope test is wider in scope than the `domain` package that hosts it, why that placement was chosen (a dedicated `src/architecture/` package would need a non-test file just to keep `go build ./...` working), and that moving it later is a mechanical rename. Re-run `make test` and `make lint`.

### Phase 4 exit

- [ ] One recorded red in the PR description.
- [ ] `cd backend/database_administrator && make test` green; `make lint` clean.
- [ ] `backend/database_administrator/go.mod` unmodified, still no `replace`.
- [ ] No scratch import in the diff.

---

## PR checklist

Assembled from doc 0001 § 9 (Boundaries + Process), doc 0002's AI-00 charter, and `openspec/AGENTS.md`.

**Evidence recorded in the PR description** (no CI exists — pasted output *is* the gate):

- [ ] Pre-change `make test` baselines for `database_administrator` and `workspace_syncer`.
- [ ] Forward guard bite 1 — red (sibling backend module, forbidden prefix).
- [ ] Forward guard bite 2 — red (OTel SDK, forbidden prefix).
- [ ] Forward guard bite 3 — red **on the allowlist branch** (deny-by-default proof).
- [ ] Reverse guard bite — red.
- [ ] `cd backend/agent && make test` green.
- [ ] `cd backend/agent && make lint` clean.
- [ ] `make test` in both existing modules, compared to baseline.
- [ ] `git diff <base>..HEAD -- backend/database_administrator/src/tools/tools.go` — empty.

**Boundaries:**

- [ ] `backend/agent/go.mod` has zero requires; no `go.sum` exists.
- [ ] No `replace` directive in any module's `go.mod`.
- [ ] `src/agenttest/` is a direct sibling of `src/ai/`.
- [ ] `src/agent/`, `src/coding/`, `src/cmd/` do not exist.
- [ ] The OTel API paths are absent from the forbidden list; the SDK, exporters and `otelslog` are present.
- [ ] The forward guard uses `go list -deps -test`; the module-scope reverse guard uses direct imports only.
- [ ] No scratch violation file or import appears anywhere in the diff.

**Process:**

- [ ] The PR description carries the review-budget exception wording verbatim (doc 0001 § 9).
- [ ] Milestone and node identifiers are appended, never renumbered.
- [ ] What this milestone deliberately leaves unsupported is stated (see `proposal.md` § "Out of scope").
- [ ] Conventional commits, no `Co-Authored-By`, no AI attribution.
- [ ] The known stale ADR 0005 § Migration cross-reference is noted in the PR description and **not** "fixed" in this PR.

---

## Graph amendments

None. AI-00's four leaves are implemented exactly as doc 0002 states them, with no node split, no appended prerequisite, and no renumbering.

Three implementation-level refinements were made inside those leaves and are recorded in `design.md` rather than as graph amendments, because none of them changes a node's scope, its dependencies, or its closing condition:

1. The forward guard uses `go list -deps -test`, not bare `-deps`. Measured: `-deps` alone reports no test-only imports, so bare `-deps` would not satisfy the leaf's own test-list item 1 ("covers test imports and transitive dependencies"). This implements the ADR's stated property with the flag that actually delivers it.
2. The stdlib allowlist is the exact set from `go list std`, not a "first path segment contains no dot" heuristic. Measured: `go list std` contains 17 `vendor/golang.org/x/...` paths that appear verbatim in real `-deps` output, which the heuristic mishandles.
3. The module-scope half of the reverse guard scans direct imports only, while the `src/domain` half stays transitive. ADR 0005 § D1 row 5 permits two packages to import the agent module, so a transitive module-scope scan would produce false positives the moment row 5 is exercised.

If implementation disproves any of the above, doc 0002's [living-graph clause](../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#the-graph-is-alive--the-revert-and-record-clause) applies: revert to green, append the discovered prerequisite as a new node with the next free ordinal, draw the edge, and land the graph amendment **in the same PR** that resumes work.
