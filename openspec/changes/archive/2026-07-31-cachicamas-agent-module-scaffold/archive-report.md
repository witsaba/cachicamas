# Archive report — Agent module scaffold and boundary guards

> **Change**: `cachicamas-agent-module-scaffold`
> **Milestone**: AI-00 of [doc 0002](../../../../docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md#ai-00--create-the-module-and-both-boundary-guards) — Create the module and both boundary guards
> **Nodes**: AI-00.1 `[mechanical]` · AI-00.2 `[mechanical]` · AI-00.3 `[guard]` · AI-00.4 `[guard]`
> **Phase**: archive
> **Status**: **ARCHIVED**
> **Date**: 2026-07-31
> **Pull request**: #95 (`feat/2026-07-31-cachicamas-ai-layer1-wave-0` → `main`)
> **Merge commit**: `a831c06` · **Base**: `origin/main` @ `b6c59e6`
> **Verify verdict**: **PASS** — see [`verify-report.md`](verify-report.md)
> **Canonical spec**: [`openspec/specs/agent-module-scaffold/spec.md`](../../../specs/agent-module-scaffold/spec.md)

---

## 1. Charter acceptance

AI-00's charter states three acceptance clauses. Each was verified against recorded output rather than intent, in `verify-report.md` § 1.

| # | Charter clause | Outcome |
| --- | --- | --- |
| 1 | `make test` and `make lint` are green in the new module | **PASS** — `verify-report.md` §§ 3.1, 3.2 |
| 2 | `make test` unchanged in the other two modules | **PASS**, on the result-set comparison the charter's byte comparison is unsatisfiable by — `verify-report.md` §§ 4.1, 4.2 |
| 3 | Both import directions are mechanically guarded, and both guards are recorded biting | **PASS** — four bites recorded red, then reverted — `verify-report.md` § 5 |

Clause 2 carries a qualification that is not a shortfall. Doc 0002's AI-00.1 check 2 asks that `make test` in the other two modules be "unchanged from its pre-change output", which is unsatisfiable by construction because AI-00.4 *adds two tests* to `database_administrator`. The comparison was therefore made on the **result set** — package, status, and the set of test names — with the expected delta stated in advance: exactly two added tests, both passing, none removed, none newly failing.

---

## 2. What was delivered

Four commits, one per leaf, landed inside PR #95.

| Commit | Leaf | Paths |
| --- | --- | --- |
| `7119776` | AI-00.1 — module skeleton | `backend/agent/go.mod`, `Makefile`, `.golangci.yml`, `.gitignore`, `README.md`; repo-root `go.work` |
| `81c279b` | AI-00.2 — package and test-package layout | `backend/agent/src/ai/doc.go`, `backend/agent/src/agenttest/doc.go`, `backend/agent/src/agenttest/import_compile_test.go` |
| `ad78198` | AI-00.3 — forward guard | `backend/agent/src/ai/import_boundary_test.go` |
| `b2b8cca` | AI-00.4 — reverse guard | `backend/database_administrator/src/domain/imports_test.go` (modified) |

Planning artifacts landed in `bd10927`; the verify report in `f0fd041`.

**Measured properties at merge**

| Property | Result |
| --- | --- |
| `backend/agent/go.mod` | one `module` directive, one `go 1.26.3` directive, **zero requires**, no `replace` |
| `backend/agent/go.sum` | absent, correctly — a module with no requires has no sum file |
| `go.work.sum` | the toolchain generated none, so none was committed |
| `cd backend/agent && make test` | green — 3 tests across `src/ai` and `src/agenttest` |
| `cd backend/agent && make lint` | `0 issues`, `golangci-lint` resolved to exactly `v2.9.0` |
| Zero-dependency invariant after `make tools` | survived — `go install pkg@version` operates outside the current module, verified rather than assumed |
| `database_administrator` result set | 10 `ok` + 3 `[no test files]`, identical to baseline plus exactly the two reverse-guard tests |
| `workspace_syncer` result set | 8 `ok`, identical to baseline, zero added and zero removed |
| `src/tools/tools.go` | byte-identical — SHA-256 `79a1803a1c7e8e930d1ec34d0ead1b101004c4443ea462b96551fd27145f0cb6` |
| `go.mod` files modified at any point, including during the bites | **none** — `go.work` resolves cross-module imports without a `require` or `replace`, so the bites could not have left residue in a manifest they never touched |

**The four guard bites**, each recorded red then reverted:

| # | Bite | What it proved |
| --- | --- | --- |
| 1 | `src/ai` imports `…/database_administrator/src/domain` | The named forbidden-prefix rule fires — and so does the allowlist branch, on `gopkg.in/yaml.v3`, a **transitive** dependency of the scratch import that no rule named |
| 2 | `src/ai` imports `go.opentelemetry.io/otel/sdk/trace` | The ADR 0005 § D3 split is visible in the failure mode itself: SDK and exporter paths fail on a named rule (permanent), API paths fail on the allowlist branch (temporary by design, AI-37 adds one entry) |
| 3 | `src/ai` imports `gopkg.in/yaml.v3`, named by no prefix | The guard failed **on the deny-by-default allowlist rule**, not on a named prefix. This is the load-bearing bite: without it the guard would be a forbidden-substring scan wearing an allowlist's clothes, and AI-24's transport choice could arrive as a quiet `go get` |
| 4 | `src/domain` imports `…/agent/src/ai` | Both reverse-guard halves fired from one violation, as doc 0002's AI-00.4 item 3 anticipates. The module-scope test reported `inspected 13 packages`, so it is not passing vacuously |

---

## 3. Where the contract now lives

The delta spec `specs/agent-module-scaffold/spec.md` was promoted to [`openspec/specs/agent-module-scaffold/spec.md`](../../../specs/agent-module-scaffold/spec.md), which is now the canonical home of the module contract. Change-scoped framing was rewritten into standing contract voice, because the invariants hold for the lifetime of the module rather than for the duration of AI-00, and because AI-24 and AI-37 will each amend the guard's allowlist under their own ADR gate — amending this canonical spec, in the same pull request, not reverting the guard.

**Deltas promoted**

| Kind | Identifiers |
| --- | --- |
| Requirements | `R-AGM-001` … `R-AGM-007` |
| Scenarios | `S-AGM-001` … `S-AGM-067` |
| Non-functional | `NFR-AGM-001` (guard determinism), `NFR-AGM-002` (guard legibility), `NFR-AGM-003` (review budget) |

Two clauses were widened from "at this merge" to "for the lifetime of the module", because the delta wording would otherwise have read as prohibiting later, chartered growth: `R-AGM-004`'s `src/ai` declares-nothing clause is now scoped to AI-00 with AI-04 onward named, and `R-AGM-005`'s empty third-party allowlist group is now scoped to AI-00 with AI-24 and AI-37 named as its only permitted growth path. Neither loosens a guard; both state where the guard's own charter already permits movement.

---

## 4. The two doc 0002 amendments this milestone landed

Doc 0002's living-graph clause requires an amendment to land in the same PR that resumes work. Two landed, in commit `7f4cb05`, both corrections of stated fact; neither added, removed nor renumbered a node.

### 4.1 AI-00.1 check 1 — the "empty pass" is not achievable

The check claimed `make test` and `make lint` "run and pass inside the new module (trivially — there is nothing to test yet, and an empty pass is still evidence the tooling is wired)". Measured on go1.26.3, a module containing no packages fails **both**: `go test ./...` warns `matched no packages` and exits 1, and `go vet ./...` exits 1. A module with a single package holding only a doc comment exits 0.

**Resolution.** AI-00.1 retains `make build` and `make help` — both exit 0 on the empty module — plus the presence of the pinned recipes; the first green `make test` became AI-00.2 check 4. No node added, none renumbered, neither leaf's scope changed.

### 4.2 AI-00.3 test-list item 1 — the flag does not deliver the property claimed for it

The item specified `go list -deps` "so it covers test imports and transitive dependencies". Bare `-deps` covers only the second: over `./src/domain` it reported 2 non-standard entries where `-test -deps` reported 5. A Layer 1 **test file** importing a sibling backend module would pass a bare-`-deps` guard — the exact failure the leaf exists to prevent, and one of the two blind spots ADR 0005 § Guard A names.

**Resolution.** The guard uses `go list -deps -test` and normalizes the three synthesized shapes `-test` introduces (`pkg [pkg.test]`, `pkg_test [pkg.test]`, `pkg.test`); unnormalized, they are measured against the allowlist and the guard fails on its own module. Scope, dependencies and closing condition unchanged.

### 4.3 One design decision recorded without an amendment

The stdlib filter uses the toolchain's own `.Standard` field rather than exact set membership against `go list std`. `go list std` contains 17 `vendor/golang.org/x/...` entries which a "first path segment contains no dot" heuristic misclassifies, while `.Standard` reports `true` for them — correct by construction, with no second subprocess and no maintained set. This changes no stated claim in doc 0002, so it is recorded in `design.md` rather than as an amendment.

---

## 5. Deliberately not done

Each has a named owner, so none is a gap. Verified absent in `verify-report.md` § 8.

- **No dependency of any kind**, including the OpenTelemetry API. The transport arrives at AI-24 behind its own ADR gate; the OTel API at AI-37, pre-authorized by ADR 0005 § D3 and by nothing else.
- **No Layer 1 contract content.** Vocabulary is AI-01, stream lifecycle and carrier AI-02, capability scope AI-03, the first type AI-04. `src/ai/doc.go` states the boundary and stops.
- **No `src/agent/`, `src/coding/` or `src/cmd/`** — not created, not stubbed, not even as an empty directory. Their absence is what makes the forward guard's forbidden-prefix list testable.
- **No `replace` directive** in any module.
- **`database_administrator/src/tools/tools.go` untouched.** S3 is resolved by vacating the name `tools`, not by relocating a standard-idiom file.
- **AI-20.4's signature guard** and **AI-25.2's ambient-authority scan** — this change guarantees only the layout the first depends on.
- **ADR 0005 § D1 row 5 not exercised.** Permitted, unexercised, and priced elsewhere: it breaks `docker compose build` and ADR 0005 makes the fix its own change.
- **No CI.** `.github/workflows/` stays absent; ADR 0005 records this under Consequences.
- **ADR 0005's stale § Migration narrative not amended by this change.** The proposal records the staleness — the ADR assigns the module move to a milestone identifier that no longer exists — and states that silently editing a merged ADR from inside an implementation PR is the wrong shape. The remap was landed separately in the same pull request, as commit `ea7cb8e`, *"docs: remap retired milestone identifiers and correct stale narrative in doc 0001 and ADR 0005"*, which touches no artifact of this change.

---

## 6. Review budget

AI-00 shipped as one PR by explicit user decision, at roughly 520 changed lines against doc 0002's *prefer under 250, reassess before 400*. The reason is recorded in `proposal.md` § "Review budget exception" and restated in `tasks.md` and in the PR description, per doc 0001 § 9: AI-00 is not decomposable into shippable slices, because its four leaves have a strict dependency chain and none of the three intermediate states is mergeable on its own — a PR with `go.mod` and a `Makefile` but no package fails its own acceptance vacuously, and a PR with the package but not the guards merges an unguarded module, which is the failure ADR 0005 exists to prevent. AI-00.3 and AI-00.4 touch disjoint modules and were reviewed as separate commits.

---

## 7. Lifecycle

`explore → proposal → spec → design → tasks → apply → verify → archive` — all phases delivered.

| Phase | File |
| --- | --- |
| Explore | `explore.md` |
| Proposal | `proposal.md` |
| Spec (delta) | `specs/agent-module-scaffold/spec.md` |
| Design | `design.md` |
| Tasks | `tasks.md` |
| Verify | `verify-report.md` |
| Archive | `archive-report.md` (this file) |

**Unblocked by this milestone:** AI-01, AI-02, AI-03 and every Layer 1 milestone thereafter — there was nowhere to put a test until it merged.
