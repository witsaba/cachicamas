```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:d531b2e01e82a289a46bb8db3247ae74829f7865e24aa2c65b5f948c537a0433
verdict: fail
blockers: 0
critical_findings: 0
requirements: 8/11
scenarios: 48/56
test_command: cd backend/agent && go test -count=1 -race ./...
test_exit_code: 0
test_output_hash: sha256:d12231ef64dc7738e35c6733beb58707dac5a3f922bd504b5a976f1f09a97aa7
build_command: cd backend/agent && go vet ./... && ./bin/golangci-lint run --config=.golangci.yml ./...
build_exit_code: 0
build_output_hash: sha256:e92606b0bf483111dff0a120c315ea165821348f31365020e2468a0059095c47
```

# Verify report — Agent package scaffold and boundary guards

> **Change**: `cachicamas-agent-package-scaffold`
> **Milestone**: AG-03 (Layer 2, Wave 1) of doc 0003 — nodes AG-03.1 `[mechanical]`, AG-03.2 `[guard]`, AG-03.3 `[guard]`
> **Phase**: verify (pass 1)
> **Status**: **0 CRITICAL, 5 WARNING, 6 SUGGESTION**. Human-readable verdict: **PASS WITH WARNINGS**. Strict-envelope verdict: **`fail`** — § 1.2 explains why that word carries no blocker here.
> **Date**: 2026-08-12
> **Worktree**: `.claude/worktrees/agent-layer2-wave1-ag03` · **Branch**: `feat/agent-layer2-wave1-ag03`
> **Mode**: Strict TDD (`openspec/config.yaml` `apply.tdd: true`). Guard-leaf convention: the guard's logic and its test are the same artifact, so RED means a recorded bite against a deliberate violation, not a test against absent production code.
> **Artifact store**: openspec only (Engram unavailable this session).

---

## 1. Method

### 1.1 Nothing was carried forward

Every claim in `apply-progress.md` that could be settled by a command was re-derived by running that command against the current bytes. In particular:

- The OpenTelemetry allowlist-omission argument was re-measured from scratch with `go list`, not read.
- **All seven required bites were physically re-planted, re-run, and re-cleaned during this verification**, rather than accepted from the apply agent's transcript. Two extra violation shapes not among the required seven (an aliased `os` import, a dot-import of `os`) were also planted, to exercise branches the shipped suite never executes.
- The Layer 1 narrowing was proved non-lossy by a set diff of the old module-wide scanned set against the new narrowed one.
- The tree was checksummed before and after every plant; `doc.go` and `src/ai/import_boundary_test.go` both return to their exact baseline MD5s (§ 6).

### 1.2 Why the envelope says `fail` while the prose says PASS WITH WARNINGS

`gentle-ai sdd-verify-validate` binds the schema's `verdict` field to evidence *completeness*, not to blocker count. Three scenarios are honestly PARTIAL (`S-AGP-022`, `S-AGP-023`, `S-AGP-040`) and one non-functional requirement is PARTIAL (`NFR-AGP-003`), so the counts are 8/11 and 48/56 and the validator will not admit `pass` against them. Declaring 11/11 and 56/56 to buy the word `pass` would be inflating the ledger, so it was not done.

`blockers: 0` and `critical_findings: 0` are the substantive result. **Nothing in this change blocks archive on correctness grounds.** One WARNING (W1) does need an editorial action *at archive*, before the spec is promoted verbatim.

---

## 2. Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 33 |
| Tasks complete | 31 (Phases 0–4) |
| Tasks incomplete | 2 (`5.1`, `5.2` — Phase 5 spec promotion, by design an archive-phase action) |

Phase 5's two unchecked boxes are **not** a CRITICAL: `tasks.md` itself titles the phase "already drafted, apply at archive", and the orchestrator scoped apply to Phases 0–4. See W3 for the bookkeeping error this exposes in `apply-progress.md`.

---

## 3. Build & tests execution

**Tests**: ✅ 12 packages `ok`, 0 failing, 0 data races. Fresh, uncached, `-race`.

```text
$ cd backend/agent && go test -count=1 -race ./...
ok  	github.com/cachicamas/backend/agent/src/agent	1.555s
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.543s
ok  	github.com/cachicamas/backend/agent/src/agenttest/sweep	1.995s
ok  	github.com/cachicamas/backend/agent/src/agenttest/tracetest	2.230s
ok  	github.com/cachicamas/backend/agent/src/ai	4.637s
ok  	github.com/cachicamas/backend/agent/src/ai/internal/retry	2.760s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	178.173s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest	3.049s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter	3.671s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance	6.747s
?   	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance/fixtures	[no test files]
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke	3.068s
ok  	github.com/cachicamas/backend/agent/src/handoff	2.549s
```

**Build / static**: ✅ `go vet ./...` silent (0 bytes of output, exit 0); `golangci-lint` v2.9.0 pinned → `0 issues.`

**Coverage**: ➖ Not measured. Every file this change adds is a `_test.go` guard, and `src/agent` has zero production statements by contract (`R-AGP-001`). A coverage number here would be meaningless, not informative.

### 3.1 The Layer 2 package's nine tests, verbose

```text
$ cd backend/agent && go test -count=1 -race -v ./src/agent/...
--- PASS: TestLayer2DocContract_MatchesTheCommittedTable (0.00s)
--- PASS: TestLayer2Agent_NonTestSourcesCarryNoForbiddenCallSite (0.00s)
    ambient_authority_test.go:256: ambient-authority scan inspected 1 non-test source file(s)
--- PASS: TestLayer2Agent_ForbiddenSetIsPackageScopedDenyByDefault (0.00s)
--- PASS: TestLayer2Agent_FailsOnStagedMutation (0.00s)
--- PASS: TestLayer2Agent_FileSelectionIsUniform (0.00s)   [4 subtests]
--- PASS: TestLayer2Agent_TestSourcesStayGreenEvenWithForbiddenCalls (0.00s)
--- PASS: TestLayer2_ProductionClosure_ImportsOnlyLayer1AndStdlib_DenyByDefault (0.02s)
--- PASS: TestLayer2_TestClosure_AdmitsOnlyTheTestSubstrateBeyondProduction (0.09s)
--- PASS: TestLayer2_ProductionClosure_ContainsNoNetworkOrFilesystemPackage (0.02s)
ok  	github.com/cachicamas/backend/agent/src/agent	1.564s
```

---

## 4. The fresh closure measurement, re-derived verbatim (`R-AGP-004`)

Re-run 2026-08-12 during verification, from `backend/agent/`. `apply-progress.md` records these commands but compresses their output (see W2); the verbatim output is preserved here so the archive carries it.

```text
$ go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai
github.com/cachicamas/backend/agent/src/ai

$ go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/ai
github.com/cachicamas/backend/agent/src/ai
github.com/cachicamas/backend/agent/src/ai [github.com/cachicamas/backend/agent/src/ai.test]
github.com/cachicamas/backend/agent/src/agenttest/sweep
github.com/cachicamas/backend/agent/src/agenttest [github.com/cachicamas/backend/agent/src/ai.test]
github.com/cachicamas/backend/agent/src/ai_test [github.com/cachicamas/backend/agent/src/ai.test]
github.com/cachicamas/backend/agent/src/ai.test

$ go list -deps -f '{{.ImportPath}}' ./src/ai | grep -E '^(net|net/http|os|io/fs)$'
(no output — grep exit 1)

$ go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' ./src/agenttest
github.com/cachicamas/backend/agent/src/agenttest/sweep
github.com/cachicamas/backend/agent/src/ai
github.com/cachicamas/backend/agent/src/agenttest
```

And the two closures the Layer 2 guard actually asserts:

```text
$ go list -deps -f '{{if not .Standard}}{{.ImportPath}}{{end}}' .../agent/src/agent/...
github.com/cachicamas/backend/agent/src/agent

$ go list -deps -test -f '{{if not .Standard}}{{.ImportPath}}{{end}}' .../agent/src/agent/...
github.com/cachicamas/backend/agent/src/agent
github.com/cachicamas/backend/agent/src/agent_test [github.com/cachicamas/backend/agent/src/agent.test]
github.com/cachicamas/backend/agent/src/agent.test
```

**Verdict on the apply agent's OTel deviation: sound, and independently confirmed.** Scoped to `src/ai` itself, the production closure contains exactly one package — `src/ai` — and no third-party path of any kind. The test closure adds only own-module packages. `src/agenttest`'s bare closure is likewise OTel-free. So neither Layer 2's production closure nor its widened test closure can reach OpenTelemetry, and the five `otel/*` + `xxhash/v2` entries design.md's Interfaces table proposed would have been **admitted-but-unreachable** — precisely the speculative grant `R-AGP-003` forbids. Omitting them is the stricter, correct reading of `R-AGP-003`'s "only" and of `S-AGP-024`.

This is not a corner cut. It is empirically load-bearing in both directions:

- **Positive control** — the *only* way to make those paths appear is to import the forbidden vendor subtree, which I did (§ 5, bite 5): all ten OTel/xxhash paths surfaced at once, every one reported under the deny-by-default rule.
- **Negative control** — `S-AGP-026`'s green half (a *test* file importing `src/agenttest`) passes with **zero** OTel allowlist entries (§ 5, bite 6′). The one scenario design.md feared the omission would break, does not break.

The all-paths listing contains `syscall` (dragged in by `time`) but none of `net`, `net/http`, `os`, `io/fs` — so `S-AGP-042`'s blocking precondition is false, confirmed by command rather than asserted.

---

## 5. Bite re-proof — every red re-planted and re-run during verification

All seven required bites plus two extra branch probes. Each was planted as a real file, run, and removed; § 6 proves the tree returned to baseline.

| # | Scenario | Guard file | Plant | Observed red (verbatim excerpt) |
|---|---|---|---|---|
| 1 | `S-AGP-013` | `doc_contract_guard_test.go` | changed L2C-02's text in `doc.go`, table untouched | `row 2 = id "L2C-02" text "…no process spawn EXCEPT verify-scratch…", want id "L2C-02" text "…no process spawn (ADR 0005…" — doc.go has drifted from the committed table` |
| 2 | `S-AGP-014` | `doc_contract_guard_test.go` | appended an unregistered `L2C-04` row | `found 4 of 3 rows … found rows: […{id:L2C-04 text:SCRATCH APPEND…}] want rows: [… 3 entries]` |
| 3 | `S-AGP-027` | `import_boundary_test.go` | `src/coding/doc.go` + `import _ ".../src/coding"` | `must not import ".../src/coding"  rule: ADR 0005 § D1 row 2: Layer 2 must not import Layer 3` |
| 4 | `S-AGP-028` | `import_boundary_test.go` | `import _ "net/http"` | check 3 only: `imports "net/http"  rule: … no network package` (checks 1–2 stayed green — the recorded point of check 3) |
| 5 | `S-AGP-029` | `import_boundary_test.go` | `import _ ".../src/ai/openaicompat"` | `must not import ".../src/ai/openaicompat"  rule: AG-03.2: the vendor adapter subtree is denied by name…` |
| 6 | `S-AGP-026` red | `import_boundary_test.go` | production `import _ ".../src/agenttest"` | `must not import ".../src/agenttest"  rule: deny-by-default allowlist (ADR 0005 § D1 row 2)…` |
| 6′ | `S-AGP-026` green | `import_boundary_test.go` | *test-file* `import _ ".../src/agenttest"` | all three checks **PASS** — production never sees it, test closure admits it |
| 7 | `S-AGP-055` | `ambient_authority_test.go` | **aliased** `osalias "os"` + `osalias.Getenv(…)` | `zz_verify_scratch.go:8: call osalias.Getenv reaches forbidden package "os" (ADR 0005 § D1 row 2 / AG-00: …)` |
| 7′ | `S-AGP-053` dot | `ambient_authority_test.go` | `import . "os"` | `zz_verify_scratch.go:3: dot-import of forbidden package "os" (…)` |
| 8 | `S-AGP-062` | `src/ai/import_boundary_test.go` | **production** `package ai` file importing `.../src/agent` | `Layer 1 must not import ".../src/agent"  rule: ADR 0005 § D1 row 1: Layer 1 must not import Layer 2` |

Three observations worth recording:

1. **Bite 8 is stronger than the apply agent's.** `apply-progress.md` planted `package ai_test` (an external test file). I planted a **production** `package ai` file, which is the harder case for a narrowed pattern: the violation must enter `src/ai/...`'s own dependency closure rather than only its test closure. It still bites, naming the path and the exact `.../src/agent` rule string. **The Layer 1 fix is a fix, not a silencing.**
2. **Bite 5 is a clean ordering proof.** The `openaicompat` path itself is reported under the *deny-by-name* rule string, while its ten OTel/xxhash transitives are reported under *deny-by-default* — exactly the split `S-AGP-029` demands, and direct evidence that forbidden-before-allowlist ordering carves the vendor subtree back out from under the admitted `.../src/ai` prefix.
3. **Bites 7 and 7′ exercise branches no shipped test executes** (see W5). Both work correctly; neither is protected against regression.

---

## 6. Tree integrity after verification

```text
$ git status --short
 M backend/agent/src/ai/import_boundary_test.go
?? backend/agent/src/agent/
?? openspec/changes/cachicamas-agent-package-scaffold/

$ md5 backend/agent/src/agent/doc.go backend/agent/src/ai/import_boundary_test.go
fa7bd7a08e9d0f972efbbddb2a4abdeb   (baseline: fa7bd7a08e9d0f972efbbddb2a4abdeb) ✅
24b0eaab6e779db7de7f729b21ced431   (baseline: 24b0eaab6e779db7de7f729b21ced431) ✅

$ ls backend/agent/src/
agent  agenttest  ai  handoff

$ git diff --stat -- backend/agent/go.mod backend/agent/go.sum backend/agent/Makefile backend/agent/.golangci.yml
(empty — all four byte-unchanged)

$ git diff --name-only -- backend/agent/src/ai/
backend/agent/src/ai/import_boundary_test.go        (exactly one entry)
```

No scratch file survives. `src/coding` and `src/cmd` do not exist. Every bite left the tree exactly as it found it.

---

## 7. Spec compliance matrix — `agent-package-scaffold` (40 scenarios)

| Req | Scenario | Evidence | Result |
|-----|----------|----------|--------|
| R-AGP-001 | S-AGP-001 | package clauses in `src/agent/*.go` → `agent`, `agent_test`×3 | ✅ COMPLIANT |
| R-AGP-001 | S-AGP-002 | `go doc -all …/src/agent` → doc comment only, zero declarations | ✅ COMPLIANT |
| R-AGP-001 | S-AGP-003 | `test -e src/coding` → 1; `test -e src/cmd` → 1 | ✅ COMPLIANT |
| R-AGP-001 | S-AGP-004 | § 3 — 12 packages `ok`, lint `0 issues.` | ✅ COMPLIANT |
| R-AGP-001 | S-AGP-005 | § 6 — four files byte-unchanged, no `require` added | ✅ COMPLIANT |
| R-AGP-002 | S-AGP-010 | `doc.go:16-18` — L2C-01 imports + ADR 0005 § D1 row 2, L2C-02 no-I/O, L2C-03 event stream | ✅ COMPLIANT |
| R-AGP-002 | S-AGP-011 | `docGoPath` uses `runtime.Caller(0)`; `os.ReadFile` + `regexp`; never imports the package | ✅ COMPLIANT |
| R-AGP-002 | S-AGP-012 | `TestLayer2DocContract_MatchesTheCommittedTable` PASS; ordered per-index diff | ✅ COMPLIANT |
| R-AGP-002 | S-AGP-013 | § 5 bite 1 — re-reproduced; names row, found text, expected text | ✅ COMPLIANT |
| R-AGP-002 | S-AGP-014 | § 5 bite 2 — re-reproduced; closed comparison proven | ✅ COMPLIANT |
| R-AGP-002 | S-AGP-015 | `doc_contract_guard_test.go:1-22` — "Substitution, not a match", names `doc_matrix_guard_test.go`, states same-PR discipline | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-020 | forward guard PASS on clean tree | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-021 | `-deps` + `-test`; `layer2Pattern` fully qualified; `{{if not .Standard}}`; `normalizeListedPackage`; zero-result fatal in all three checks | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-022 | forbidden-first ✅; comment names both cases but records that the ordering does **not** apply to `net/http` — see W1 | ⚠️ PARTIAL |
| R-AGP-003 | S-AGP-023 | `forbiddenPrefixes` names 8 of the 9 prefixes, each with its rule; `net/http` lives in `networkOrFilesystemPackages` with its own rule — see W1 | ⚠️ PARTIAL |
| R-AGP-003 | S-AGP-024 | allowlist has **only** own-module entries; no SDK/exporter/otelslog path; no entry absent from § 4's measurement | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-025 | two tests, two allowlists; `allowedProductionPrefixes` has no `agenttest` — proven by § 5 bite 6 | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-026 | § 5 bites 6 / 6′ — both halves re-reproduced live | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-027 | § 5 bite 3 | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-028 | § 5 bite 4 | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-029 | § 5 bite 5 — deny-by-name rule string fires for the path itself | ✅ COMPLIANT |
| R-AGP-003 | S-AGP-030 | § 6 — clean tree, no scratch file, no `require` | ✅ COMPLIANT |
| R-AGP-004 | S-AGP-040 | commands + date recorded in `apply-progress.md`; output compressed, not verbatim — see W2 (verbatim now supplied in § 4) | ⚠️ PARTIAL |
| R-AGP-004 | S-AGP-041 | every allowlist entry appears in § 4's measurement; no measured entry is admitted unnamed (`openaicompat` carved out first, proven by bite 5) | ✅ COMPLIANT |
| R-AGP-004 | S-AGP-042 | precondition false, verified by command: the all-paths listing contains none of `net`/`net/http`/`os`/`io/fs` | ✅ COMPLIANT |
| R-AGP-004 | S-AGP-043 | `apply-progress.md` § Phase 2 states the doc-0003 match **and** the OTel divergence explicitly | ✅ COMPLIANT |
| R-AGP-005 | S-AGP-050 | PASS + `ambient-authority scan inspected 1 non-test source file(s)`; fatal on zero | ✅ COMPLIANT |
| R-AGP-005 | S-AGP-051 | `parser.ParseFile` + `ast.Inspect` over `*ast.CallExpr` / `*ast.SelectorExpr`; not path or text matching | ✅ COMPLIANT |
| R-AGP-005 | S-AGP-052 | set is exactly `{os, os/exec, syscall, io/ioutil}`, pinned by `…_ForbiddenSetIsPackageScopedDenyByDefault`; package comment states why a closure scan cannot express the rule | ✅ COMPLIANT |
| R-AGP-005 | S-AGP-053 | alias → `localNameToPackage[imp.Name.Name]`; dot → flagged in its own right. **Both branches executed live** (§ 5 bites 7, 7′) | ✅ COMPLIANT |
| R-AGP-005 | S-AGP-054 | `isLayer2SourceFile` suffix rule + `…_FileSelectionIsUniform` asserts the guard's own file is excluded by the rule, not by name | ✅ COMPLIANT |
| R-AGP-005 | S-AGP-055 | § 5 bite 7 — names file, line 8, package, rule; permanent staged-mutation equivalent also PASS | ✅ COMPLIANT |
| R-AGP-005 | S-AGP-056 | package comment § "Recorded limitation" — no type info, shadowing example, `go/types`/`x/tools` unauthorised | ✅ COMPLIANT |
| R-AGP-006 | S-AGP-060 | `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` PASS, uncached, with `src/agent` present | ✅ COMPLIANT |
| R-AGP-006 | S-AGP-061 | `import_boundary_test.go:118-120` — `src/agent` row present with its exact § D1 row 1 rule string; `coding`/`cmd` rows byte-unchanged | ✅ COMPLIANT |
| R-AGP-006 | S-AGP-062 | § 5 bite 8 — **production**-file import, still bites, names path and the Layer-2 rule | ✅ COMPLIANT |
| R-AGP-006 | S-AGP-063 | set diff (§ 8) — narrowed set = old set **minus exactly the three `src/agent` entries**; 40 non-stdlib entries, non-zero | ✅ COMPLIANT |
| R-AGP-006 | S-AGP-064 | `TestLayer1_DependencySet_ExactRequiresAndClosure` PASS; `wantGoModRequires` / `wantExternalClosure` do not appear in the diff at all | ✅ COMPLIANT |
| R-AGP-006 | S-AGP-065 | `import_boundary_test.go:94-117` — `FIXED (AD-1, 2026-08-12, AG-03)` records mechanism, rejected alternative with its reason, and that the row was preserved | ✅ COMPLIANT |
| R-AGP-006 | S-AGP-066 | changed-file set under `src/ai/` is exactly one entry; no `require`; no forbidden row removed | ✅ COMPLIANT |

**AGP compliance: 37/40 COMPLIANT, 3 PARTIAL, 0 UNTESTED, 0 FAILING.**

### 7.1 Non-functional requirements

| NFR | Status | Notes |
|-----|--------|-------|
| NFR-AGP-001 — guard determinism | ✅ | All guards green under `-race`; `go list` patterns fully qualified; doc guard resolves via `runtime.Caller(0)`. See S5 on the ambient guard's `"."` root. |
| NFR-AGP-002 — guard legibility | ✅ | Every one of the ten reds in § 5 named both the offending path/row **and** the rule that rejected it. Verified by reading the actual failure output, not the source. |
| NFR-AGP-003 — review budget | ⚠️ PARTIAL | 1005 authored lines measured exactly (§ 9) — 5 over the pre-authorised 1000-line ceiling. The PR does not exist yet, so its required rationale text is unwritten. See W4. |

---

## 8. Proof that the Layer 1 narrowing lost no coverage (`S-AGP-063`, `S-AGM-041`)

The single most load-bearing risk in AD-1 is that narrowing the scanned roots quietly drops a Layer 1 package. Settled by set difference, not by reading. Both listings use `go list -deps -test` with the non-standard-library filter, with the bracketed test-variant suffix stripped and the result sorted and deduplicated:

```text
old pattern:  github.com/cachicamas/backend/agent/...                              → 43 entries
new patterns: …/src/ai/...  …/src/agenttest/...  …/src/handoff/...                 → 40 entries

difference (old minus new), complete:
  github.com/cachicamas/backend/agent/src/agent
  github.com/cachicamas/backend/agent/src/agent.test
  github.com/cachicamas/backend/agent/src/agent_test
```

The narrowed pattern set removes **exactly the three synthesized `src/agent` entries and nothing else**. The remaining 40 are identical. Every Layer 1 package the old pattern covered is still covered — confirmed present in the narrowed list:

```text
…/src/ai            …/src/ai_test           …/src/handoff         …/src/handoff_test
…/src/agenttest     …/src/agenttest_test    …/src/agenttest/sweep (+_test)   …/src/agenttest/tracetest (+_test)
```

This closes both `S-AGP-063` and the amended `S-AGM-041`'s "with none omitted" clause with direct evidence. Note that `modulePath + "/src/handoff/..."` does match the `src/handoff` package itself — confirmed above — so the spec's singular `src/handoff` wording is satisfied by the `/...` form.

---

## 9. Coherence — design decisions vs shipped code

| Decision | Followed? | Notes |
|----------|-----------|-------|
| AD-1 — narrow the L1 scanned roots | ✅ Yes | `layer1Patterns` exactly as specified; `listNonStdlibDeps` variadic; both call sites updated; vacuous-pass message joins the slice; `KNOWN HAZARD` rewritten as a `FIXED` note; `src/agent` row byte-unchanged. Proven non-lossy in § 8. |
| AD-2 — zero OTel grant to Layer 2 | ⚠️ Deviated, correctly | design.md's Interfaces table listed five OTel/xxhash entries "as `src/ai`'s measured forced closure". § 4 disproves that premise by measurement; the entries are omitted. This makes the guard **stricter**, is recorded in the guard's own package comment (`import_boundary_test.go:28-63`) and in `apply-progress.md`, and is the correct reading of `R-AGP-003`'s "only". Not a corner cut. |
| AD-3 — three checks, two closures | ✅ Yes | Three separate tests; production and test asserted independently; check 3 uses `listAllProductionDeps` (standard library included, no `-test`). Independence proven by § 5 bites 6 / 6′. |
| AD-4 — `L2C-NN` tab grammar, committed table | ✅ Yes | Pinned row regexp; two-field tab split; count-then-ordered diff; `runtime.Caller(0)` resolution; fatal on an unresolvable file and on a malformed matched row. |
| AD-5 — verbatim AI-25.2 retarget | ✅ Yes, plus one addition | Forbidden set, alias/dot/blank handling, uniform `_test.go` exclusion, staged-mutation bite, recorded limitation — all present. The added vacuous-pass fence is an improvement on the precedent, required by `S-AGP-050`'s literal wording. |
| AD-6 — fresh measurement is a gate | ✅ Yes | Run before the allowlist was written; the one divergence it found (AD-2's premise) changed the allowlist actually committed rather than being absorbed. Re-derived independently in § 4. |
| Review budget — `size:exception` at 1000 | ⚠️ 1005 | Measured: `ambient_authority_test.go` 380 + `import_boundary_test.go` 400 + `doc_contract_guard_test.go` 123 + `doc.go` 19 = 922, plus the L1 diff (56 insertions, 27 deletions) = **1005**. Apply flagged this rather than rounding down. See W4. |

The two undesigned implementation extensions both check out:

- **`isAllowed`'s `_test` trim** is genuinely load-bearing, not a workaround. `go list -deps -test` over `layer2Pattern` emits `…/src/agent_test` (§ 4), and the prefix matcher rejects the raw form — `…/src/agent_test` is neither equal to nor prefixed by `…/src/agent/`. Without the trim, check 2 fails on the guard's own external test package, exactly the failure `apply-progress.md` reports catching at task 2.5. L1's guard is immune only because its single allowlist entry is the bare module path, which subsumes that shape by accident. See S2 for the one narrow hardening this leaves on the table.
- **The ambient guard's vacuous-pass fence** is required by `S-AGP-050`'s "reports having inspected at least one source file", which the lifted openrouter precedent does not satisfy. Verified live: the log line fires with count 1 on the clean tree and count 2 during a bite.

---

## 10. Amendment fidelity — `agent-module-scaffold` delta

| Scenario | What the amendment claims | What the shipped code does | Result |
|----------|---------------------------|----------------------------|--------|
| `S-AGM-035` | `src/agent` exists (exit zero), a direct sibling of `src/ai` and `src/agenttest`; `src/coding` and `src/cmd` exit non-zero | `src/` contains exactly `agent`, `agenttest`, `ai`, `handoff`; `test -e src/coding` → 1; `test -e src/cmd` → 1 | ✅ Faithful |
| `S-AGM-041` | `-deps` **and** `-test`; fully-qualified patterns, never relative; the scanned set covers `src/ai/…`, `src/agenttest/…` and `src/handoff` with none omitted; and the source states in place which mechanism prevents a pattern member being reported as an import violation, and why | `listNonStdlibDeps` passes both flags; all three patterns are built from `modulePath`; § 8 proves none omitted; `import_boundary_test.go:94-117` states narrowing as the chosen mechanism, states why, and states the rejected alternative with its reason | ✅ Faithful |

Both restatements are complete `MODIFIED` blocks per the delta's own promotion discipline. `R-AGM-005`'s eight forbidden prefixes are unchanged — they do not appear in the diff at all — and its claim that AG-03's re-bite obligation "is discharged by `S-AGP-062`" is now backed by § 5 bite 8. `R-AGM-008`'s closure pin is unaffected: `TestLayer1_DependencySet_ExactRequiresAndClosure` passes with both tables untouched, because the pin filters `modulePath`-prefixed entries before comparing and `src/agent` contributes no external package.

Five of the delta's sixteen restated scenarios are historical statements not re-derivable at AG-03 — `S-AGM-030` and `S-AGM-031` are scoped to "AI-00's merge", and `S-AGM-044`, `S-AGM-045`, `S-AGM-046` are AI-37's recorded bites whose AG-03 obligation the delta explicitly discharges through `S-AGP-062`. They are carried, not regressed. The other eleven were re-verified this pass.

---

## 11. Explore-phase risk closure

| Original risk | Closed by | Verified how |
|---|---|---|
| 1. Byte-suffix convention underspecified | `R-AGP-002`'s recorded substitution + AD-4 | Read `doc_contract_guard_test.go:1-22`: it states the substitution, names `doc_matrix_guard_test.go` as the copied precedent, and states the same-PR amendment discipline. Both directions of the closed comparison were made to bite (§ 5 bites 1–2). **Closed.** |
| 2. L1 self-reference fix soundness | AD-1 | Set diff proves zero coverage loss (§ 8); a production-file re-bite proves it still fails on a genuine violation (§ 5 bite 8). **Closed — this is a fix, not a silencing.** |
| 3. OTel scope decision | AD-2, corrected by AD-6's measurement | Re-measured independently (§ 4); positive and negative controls both run. **Closed, and the shipped answer is stricter than the design's.** |
| 4. Fresh network-measurement gate | `R-AGP-004` / AD-6 | Three commands re-run verbatim this pass; the production closure contains none of `net`, `net/http`, `os`, `io/fs`; doc 0003's 2026-08-11 claim matches. **Closed.** |
| 5. L1 spec amendment correctness | `agent-module-scaffold` delta | § 10 — both amended scenarios are true against the shipped code. **Closed.** |

No explore-phase CRITICAL remains open.

---

## 12. TDD compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD evidence reported | ✅ | `apply-progress.md` § "TDD Cycle Evidence" — four rows, one per guard leaf |
| All tasks have tests | ✅ | 4/4 guard leaves have their guard file; the mechanical leaf (`doc.go`) is spec-exempt from red-green |
| RED confirmed (bites are real) | ✅ | **7/7 required bites re-planted and re-run by verify**, not accepted from the transcript (§ 5) |
| GREEN confirmed (tests pass) | ✅ | 12/12 packages `ok`, fresh, `-race`, uncached |
| Triangulation adequate | ✅ | doc-contract 2 shapes (edit, append); forward guard 4 shapes (app-layer, stdlib-network, vendor-by-name, deny-by-default); ambient 1 required shape + 2 extra branch probes run by verify |
| Safety net for modified files | ✅ | Task 0.4 recorded a pre-existence GREEN baseline before the L1 edit; the `(cached)` false-green was caught and re-run uncached |

**TDD compliance: 6/6 checks passed.**

### 12.1 Test layer distribution

| Layer | Tests | Files | Tools |
|-------|-------|-------|-------|
| Unit (guard) | 9 top-level (+4 subtests) in `src/agent`; 3 in `src/ai` re-verified | 3 new, 1 modified | `go test -race` |
| Integration | 0 | — | not applicable — no `main`, no server, zero production statements |
| E2E | 0 | — | not installed |

### 12.2 Assertion quality audit

Every test file created or modified by this change was read line by line against the banned-pattern list.

- No tautology. No assertion that never calls the code under test. **No ghost loop** — the two loops over the dependency list are fenced by an explicit zero-result fatal, and the loop over directory entries in the ambient guard is fenced by the `inspected == 0` fatal. This is the specific failure mode a guard-shaped test would otherwise hide, and all three guards fence it.
- **No orphan empty check.** `TestLayer2Agent_TestSourcesStayGreenEvenWithForbiddenCalls` asserts the violation list is empty, but has a companion with the same setup — `TestLayer2Agent_FailsOnStagedMutation` — asserting it is non-empty. That is exactly the required pairing.
- No smoke test, no CSS or implementation-detail coupling, no mocks at all — the subprocess under test is the real `go` toolchain.
- `TestLayer2Agent_FileSelectionIsUniform` is table-driven with variance in the expected value (`true` and `false`), not four assertions of the same shape.

**Assertion quality: ✅ All assertions verify real behavior. 0 CRITICAL, 0 WARNING.**

The one caveat is coverage of code paths, not assertion quality — see W5.

---

## 13. Issues found

### CRITICAL — none

No finding blocks correctness. Every protective property `R-AGP-001` through `R-AGP-006` claims was proven by executing a violation against it during this verification pass, not by reading the guard's source.

### WARNING (5)

**W1 — `net/http` is not in `forbiddenPrefixes`, so `S-AGP-022` and `S-AGP-023` are literally false against the shipped guard. Resolve this *before* the spec is promoted verbatim at archive.**

`R-AGP-003` requires a forbidden-prefix list "covering at minimum: … and `net/http`", justified by "an allowlist-first pass would admit both". That justification is factually wrong, and the shipped code says so in place: the dependency lister filters `.Standard` **before** either allowlist runs, so `net/http` never reaches `forbiddenPrefixes` at all and a row for it there would be unreachable dead code. Design AD-3 identified this and routed `net/http` to `networkOrFilesystemPackages` and check 3 instead. **The implementation is more correct than its spec**, and the protective property is fully delivered and bite-proven — § 5 bite 4 names `net/http` and its rule.

The problem is durability. This spec's header states it is "promoted verbatim at archive". Promoting it as written puts two scenarios into a live `openspec/specs/` capability that are false against the code they describe, guarded by nothing — the exact owning-spec staleness mechanism this repository already tracks.

*Recommended action at archive*: amend `R-AGP-003`'s prose and `S-AGP-022` / `S-AGP-023` to describe the two-table mechanism the code ships — a forbidden-prefix table for non-standard-library denials, an exact-path table for standard-library denials, and an ordering rationale scoped to the vendor subtree alone — then promote. Do **not** "fix" the code by adding a dead `net/http` row.

**W2 — `S-AGP-040`'s "full output" is recorded compressed, not verbatim.**

`apply-progress.md` records all three `go list` commands and the date, but paraphrases their output: brace shorthand for the OTel paths, `(+ conformancetest)` elisions, and "stdlib only (bytes, context, … various internal/*)". The scenario requires "their full output". The substance was re-derived and confirmed correct this pass, and the verbatim output is now preserved in § 4, which closes the gap for the archive record.

**W3 — `apply-progress.md` contradicts itself on task count, and its test census is off by one.**

Its Status section says "33/33 tasks in `tasks.md` marked `[x]`"; the actual count is **31** checked and 2 unchecked. Its own closing line says "Phase 5 … intentionally untouched", which contradicts the 33/33 claim. Separately, its Test Summary claims "12 top-level" guard test functions; the enumeration it gives sums to 11 (1 doc-contract + 3 forward + 5 ambient + 2 L1), and the verbose run shows 9 top-level in `src/agent`. Neither error changes any verdict, but an evidence log whose arithmetic is wrong should be corrected before it becomes the archive record.

**W4 — the change is 1005 authored lines, 5 over the pre-authorised 1000-line `size:exception` ceiling, and `NFR-AGP-003`'s required PR rationale does not exist yet.**

Measured independently rather than taken from the report: 922 lines across the four new files plus 56 insertions and 27 deletions on the L1 guard = 1005. Apply flagged the overage explicitly rather than rounding it down, which is the right behaviour. It still needs the driver's acknowledgement, and `NFR-AGP-003`'s "the pull request description MUST state why the change does not fit the default budget" is unsatisfiable until the PR is opened.

**W5 — the ambient guard's alias and dot-import branches have no permanent covering test.**

`scanFileForAmbientAuthority` has four import-resolution branches: blank (skip), dot (flag in its own right), named alias (bind the local identifier), and default (bind the package's default name). The only shipped fixture, in `TestLayer2Agent_FailsOnStagedMutation`, plants a plain `import "os"` and therefore exercises the **default** branch alone. I planted an aliased import and a dot import during this pass and both behave correctly (§ 5 bites 7 and 7′) — but nothing in the shipped suite protects them from regression, and `S-AGP-053` is the scenario that exists to keep them alive. Two more staged-mutation fixtures mirroring the existing one would close this for roughly twenty lines.

### SUGGESTION (6)

**S1 — `S-AGP-026`'s proof is transient; the ambient guard's equivalent is permanent.** No test file in `src/agent` imports `src/agenttest`, so `allowedTestPrefixes`'s substrate entry is currently unexercised configuration, and the production/test independence claim survives only as a deleted scratch file in a transcript. `TestLayer2Agent_FailsOnStagedMutation` shows the better pattern. AG-04, which will import the substrate for real, closes this naturally.

**S2 — `isAllowed`'s `_test` trim bypasses the forbidden check on the trimmed base.** `matchForbidden` runs on the raw path only. A path such as `…/src/ai/openaicompat_test` matches no forbidden row — it is neither equal to nor prefixed by `…/src/ai/openaicompat/` — then trims to an allowlisted base and is admitted. Unreachable today: verified that `go list -deps -test` synthesizes the `<pkg>_test` shape only for the named pattern's own members, never for a dependency, so only `…/src/agent`-rooted paths can take it. Re-running `matchForbidden` on the trimmed base inside that branch is a two-line hardening.

**S3 — checks 1 and 2 currently scan one and two entries respectively.** Structurally correct, bite-proven, and fenced against the zero case — but their green is near-trivial until Layer 2 imports something. Worth remembering when reading a future green: the bites, not the pass, are what make these two guards mean anything today.

**S4 — `…/src/ai` is admitted before use.** Layer 2's measured closure does not reach it. `S-AGP-024`'s before-use clause is satisfied, because `src/ai` does appear in `R-AGP-004`'s measurement and ADR 0005 § D1 row 2 authorises it outright — but a one-line "admitted ahead of AG-04's first real use" note on that entry would match the clause's spirit as well as its letter.

**S5 — the ambient guard resolves its scan root from the process working directory, not from the test's own source location.** Safe in practice: `go test` pins the working directory to the package source directory, so it cannot depend on where `make test` is invoked, and this matches the AI-25.2 precedent it lifts verbatim. But `NFR-AGP-001`'s literal "file paths are resolved from the test's own location" is met only by the doc guard, which uses `runtime.Caller(0)`. One line, and both guards would resolve identically.

**S6 — `syscall` is in the production closure but deliberately absent from check 3's list.** The all-paths listing of `./src/ai` shows `syscall` arriving via `time`. It is in the ambient guard's forbidden **call-site** set but not in `networkOrFilesystemPackages`, which is correct — it is unavoidable standard library and a closure row for it could never pass — but the omission is currently silent. A source note would stop a later reader "fixing" it.

---

## 14. Verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 5 WARNING, 6 SUGGESTION.

Every guard this change ships was proven to bite during this verification pass, not merely reported as having bitten. The Layer 1 fix is demonstrably a fix and not a silencing: a production-file import of Layer 2 still fails the narrowed guard with its exact rule string, and the narrowing removes exactly the three synthesized `src/agent` entries and no Layer 1 package. The apply agent's OpenTelemetry omission is empirically correct and makes the guard stricter than its own design specified. The tree is clean, `go.mod`, `go.sum`, `Makefile` and `.golangci.yml` are byte-unchanged, and tests, vet and lint are all green.

**Recommended next phase: `sdd-archive`**, with one mandatory editorial action first:

> **W1 must be resolved before `agent-package-scaffold/spec.md` is promoted verbatim.** Amend `R-AGP-003`, `S-AGP-022` and `S-AGP-023` to describe the two-table mechanism the code actually ships. Promoting the current wording would land two knowingly-false scenarios into a live capability spec on day one.

W2, W3 and W4 are record-quality and delivery-budget items, not correctness blockers. W5 and all six suggestions are improvements a follow-up milestone can absorb.
