```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:8d825c4b4e158a9aee1add1e1bf6a2cb3a88e8735b7fa24e254814199a44def1
verdict: pass
blockers: 0
critical_findings: 0
requirements: 21/21
scenarios: 89/89
test_command: make test
test_exit_code: 0
test_output_hash: sha256:4609c54a9ad4a00c695e1009bd1ebd35b8a2de7c01e4dfd8113d40aacd8a39d3
build_command: make lint
build_exit_code: 0
build_output_hash: sha256:66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a
```

## Verification Report

**Change**: cachicamas-ai-request-translation (AI-26 — Translate normalized requests to wire requests)
**Version**: change-folder delta, `specs/ai-request-translation/spec.md`
**Mode**: Strict TDD
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4`, branch `feat/ai-26-8-unsupported-policy`
**Base checkout**: `/Users/braejan/workspace/witsaba/repositories/cachicamas` — `git status --short` returned empty (clean), branch `main`. Read-only check, zero writes.
**Sibling worktree**: never read, never referenced, zero commands run there.

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 99 |
| Tasks complete | 99 |
| Tasks incomplete | 0 |

Counted independently from `tasks.md`: `grep -cE '^\s*- \[x\] '` = 99, `- \[ \]` = 0. Matches the apply-progress claim.

Spec counts verified independently, not taken from the artifact header: 21 `### R-ART-nnn` headings (unique IDs 001–021), 6 `### NFR-ART-x` headings (A–F), 89 unique `S-ART-nnn` scenarios — 72 runnable, 17 `*(review)*`. These match the spec's own declared counts exactly.

### Chain and diff measurement

22 commits, 20 files, 4,708 insertions off `feat/ai-25c-test-server-viability` — matches the stated figures exactly.

| Slice | Branch | Changed lines | Within 5,000 budget |
|---|---|---|---|
| 1 | ai-26-1-translation-skeleton | 621 (+621) | Yes |
| 2 | ai-26-2-system-segments | 435 (+429/-6) | Yes |
| 3 | ai-26-3-tools-deterministic | 367 (+340/-27) | Yes |
| 4 | ai-26-4-options-escape-hatch | 628 (+618/-10) | Yes |
| 5 | ai-26-5-messages-content-parts | 626 (+560/-66) | Yes |
| 6 | ai-26-6-tool-results | 685 (+603/-82) | Yes |
| 7 | ai-26-7-reasoning-refusal | 659 (+564/-95) | Yes |
| 8 | ai-26-8-unsupported-policy | 1,261 (+1,260/-1) | Yes |

Every slice is well inside the 5,000-line budget; the largest is 1,261.

### Build & Tests Execution

**Tests**: PASSED — `make test` (`go test -race -v ./...`) from `backend/agent/`, exit code **0**.

```text
ok  github.com/cachicamas/backend/agent/src/agenttest
ok  github.com/cachicamas/backend/agent/src/ai
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat
1805 --- PASS | 0 --- FAIL | 8 --- SKIP | 0 DATA RACE
```

The 8 SKIPs are all pre-existing `agenttest` conformance cases for declared-absent optional capabilities (token counting, cache boundary, reasoning). None is in `openaicompat`, none was introduced by this chain.

`make test` reported packages as `(cached)`, so cached results were not accepted as the evidence. Five additional independent `go test -race -count=1 ./...` invocations were run: every one exit 0, 3/3 packages ok, 0 failures, 0 races, identical results. This is the cross-run evidence for `NFR-ART-F` / `S-ART-089`.

**Build/Lint**: PASSED — `make lint` (`go vet ./...` then `golangci-lint run --config=.golangci.yml ./...`), exit code **0**, output `0 issues.`

**Coverage**: not run — this milestone's contract is byte-exact expectation comparison and mechanical guards, not a coverage threshold; no coverage gate is configured in the Makefile.

### Priority check 1 — Requirement coverage (21/21)

| Req | Discharged by | Evidence seen |
|---|---|---|
| R-ART-001 | `doc.go:66-144` citation gate | Repo, pinned commit and retrieval date all present in source (detail below) |
| R-ART-002 | `translation_test.go` `expectationCases` + `TestExpectationCases_MatchByteExact` | Inline literals compared byte-for-byte; passed |
| R-ART-003 | Same test as the cross-run proof; `TestTranslate_SameProcessDoubleTranslate_IsAPin` | Function name and doc comment both label the in-process check a pin, not the proof |
| R-ART-004 | `credential_scan_test.go` `scanCredentialSurface` | `os.ReadDir` + suffix rule; auto-covers later files; sentinel patterns assembled at runtime so the table never matches itself |
| R-ART-005 | `system.go:39` renders `{"role":"system","content":` | `system_segment_test.go` order/reversed-twin/no-instruction cases |
| R-ART-006 | `cache_marker_test.go` twin walk over `ai.CacheRegions()` | 3 drop rows in `featurePolicy`; twins byte-identical |
| R-ART-007 | `TestMessage_PartKindCoverage` over `ai.PartKinds()` | Distinct fixture per kind + `default:` case that fails on an unexercised kind |
| R-ART-008 | `TestMessage_AdjacentSwap_TranslatesDifferently` | Two hard-coded literals **plus** an inequality check |
| R-ART-009 | `body.go` `appendMessagesField` unconditional per-message loop | No lookahead, no grouping; consecutive same-role cases registered |
| R-ART-010 | `tool.go:70` `append(buf, tool.Schema()...)` | Raw splice, never `appendJSONString`, never re-marshalled |
| R-ART-011 | `tool_test.go` non-alphabetical declaration order + reversed twin | Fresh-process expectation comparison |
| R-ART-012 | `message.go` `appendToolResultObject` emits `{"role":"tool",...}` | Distinct wire message, never nested |
| R-ART-013 | `appendToolCallObject` / `appendToolResultObject` splice `ID()`/`CallID()` unchanged | No generation/derivation/normalization on any path |
| R-ART-014 | `tool_result_test.go` interleaved multi-call cases | Results supplied out of call order, each naming its own id |
| R-ART-015 | `policy.go` `refuseReasoning` + `translation.go` pre-`appendBody` check | Returns `nil` bytes + `PreStreamFailure`; 4 witnesses + 9 registered refusal cases |
| R-ART-016 | `option.go` `appendGenerationOptionFields` presence flags | Absent vs zero distinguished |
| R-ART-017 | `TestExpectationCases_AllCarryUsageOptIn` | Total walk over the whole registry with an explicit empty-registry vacuity guard |
| R-ART-018 | `max_tokens` appended only when the presence flag is set | Absence asserted directly, not as equality to a default |
| R-ART-019 | `option.go:41` `const Namespace = "openaicompat"` | Own namespace merges raw; foreign twin byte-identical |
| R-ART-020 | `feature_inventory_test.go` hybrid derivation | 5 runtime enumerators + `go/ast` scan; counts verified below |
| R-ART-021 | `policy.go` `featurePolicy` (28 rows) + `policy_walk_test.go` | Inventory-driven walk, durable bite proofs |

### Priority check 1 — NFR coverage (6/6)

| NFR | Evidence seen |
|---|---|
| NFR-ART-A | `go.mod` contains only `module` and `go 1.26.3` — zero `require`, no `go.sum` present. Both AI-00 guards run and passed: `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault`, `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` |
| NFR-ART-B | `git diff --name-only` over the chain returns 20 paths, **all** under `src/ai/openaicompat/`. No file in `package ai` was modified. Full suite green |
| NFR-ART-C | No AI-26 test file references `httptest`, `net.Listen` or `net/http`. `TestTranslate_LeavesTheRequestEqualToAnIndependentlyBuiltCopy` proves non-mutation via the exported `Request.Equal` |
| NFR-ART-D | No `testdata/` directory, no `-update` flag, no golden pattern anywhere in the package. Expectations are inline raw-string literals |
| NFR-ART-E | `doc.go:185-211` carries the AI-25-vs-AI-26 taxonomy table with the AI-03 §10.4 citation |
| NFR-ART-F | `make test` exit 0 under `-race`, `make lint` 0 issues, 5 independent invocations identical, per-scenario red/green/review evidence present in `tasks.md` |

### Priority check 2 — THE CITATION GATE

Provenance is recorded **in the source**, not only in a report. `doc.go:66-144`:

- OpenAPI spec repository: `github.com/openai/openai-openapi`, `openapi.yaml`, described as organization-owned.
- **Pinned commit hash**: `d4fb706e6e05d4cc9f1b33ca59b6e4f3e8edd439` (wrapped across two comment lines; 40 hex chars when joined).
- Retrieval date: **2026-08-03**, stated as "Retrieved 2026-08-03."

Each of the four claims is used as cited:

1. **System-instruction placement and role** — cited as a `messages` entry via the `ChatCompletionRequestMessage` discriminated union, never a top-level field. Both `system` and `developer` are recorded as legal members of the union, exactly as the check required. `system.go:39` emits `{"role":"system","content":` and `developer` appears in no non-test source. Deliberate `system`-only emission confirmed.
2. **Tool-call arguments as a JSON-encoded string** — `message.go` `appendToolCallObject` calls `appendJSONString(buf, string(call.Arguments()))`, producing `"arguments":"{}"` (a quoted string), not `"arguments":{}` (a nested object). Confirmed in six expectation literals. The citation is request-side specific (`ChatCompletionRequestAssistantMessage` → `ChatCompletionMessageToolCall.function.arguments` `type: string`), which is what `S-ART-002` demanded.
3. **`tools[].function.parameters` verbatim** — `tool.go:70` appends `tool.Schema()` raw. Cited via `ChatCompletionTool.function` → `FunctionObject` → `FunctionParameters` (`type: object`, `additionalProperties: true`).
4. **No strict role alternation** — established from two sources jointly, as required: the spec's `messages` array imposing no ordering constraint (with the negative check that no normative use of "alternat*" appears in the ~84,000-line spec), plus the Function-calling guide's parallel-function-calling worked example appending consecutive `role:"tool"` messages. AI-24 §6's lone `role:"tool"` row is explicitly recorded as *indicative and not dispositive*, which `S-ART-003` specifically required.

**Citation gate verdict: PASSED.** See WARNING 1 for a separate, ungated claim whose pointer is broken.

### Priority check 3 — Byte fidelity

Confirmed by reading the code. `body.go` `appendBody` hand-assembles the body into a `[]byte` in one fixed source-code order. `encoding/json` is imported for exactly one purpose: `appendJSONString`, which marshals a **single Go string** to get JSON escaping for string leaves. No struct, map, or composite value is ever marshalled, so no `json.Marshal` path produces the body. `Translate` calls only `refuseReasoning` then `appendBody`.

The schema-fidelity test would genuinely fail on a re-encode. `tool_test.go`'s `searchFlightsSchema` is deliberately hostile to `json.Marshal`: it is multi-line and indented, declares `"required"` **before** `"additionalProperties"` (alphabetically inverted), and contains `"additionalProperties":    false` with a four-space run after the colon. The expectation splices that same constant into `want`. A decode/re-encode round trip would compact the whitespace and alphabetically sort the keys, breaking the byte comparison on two independent grounds.

### Priority check 4 — Cross-run determinism

The proof is the fresh-process kind. `TestExpectationCases_MatchByteExact` compares translation output against **inline literals checked into source**, so each `go test` process — with its own independently seeded map hasher — re-verifies against a fixed constant rather than against a value derived in the same process. `TestTranslate_SameProcessDoubleTranslate_IsAPin` is named `_IsAPin` and its doc comment states outright that it is "explicitly NOT the determinism proof", citing Go's per-range-statement map-range re-randomization. `S-ART-012`'s review obligation is discharged in place.

Independently re-run five times (`go test -race -count=1`), identical every time. Structurally, no map is ranged on the translation path: `Messages()`, `Segments()` and `ToolSet.Tools()` are all caller-ordered slices, and `featurePolicy` is a slice with a linear scan rather than a map.

### Priority check 5 — The three coordinator rulings

| Ruling | Recorded in `doc.go` | Reopen trigger |
|---|---|---|
| System role is `system`, never `developer` | Yes, §"System role: system, not developer" | **Yes** — §"Living-graph reopen trigger — not a hedge": reopens if pointed at a first-party OpenAI endpoint whose models require `developer` |
| Reserved namespace is the exported `openaicompat.Namespace = "openaicompat"`, defined by this milestone | Yes, §"Coordinator ruling: the value is openaicompat, and this milestone defines it". Records explicitly that the instruction to read it from AI-25 was itself the defect because AI-25 was never given a task to define one. Constant confirmed exported at `option.go:41` | **No** — see WARNING 2 |
| Reasoning replay refuses | Yes, §"Reasoning replay refuses; Layer 2 must strip first" | **Yes** — reopens if a future vendor update adds a wire field for reasoning |

The reasoning ruling is fully discharged: the doc states plainly that Layer 2 **MUST** strip every reasoning content part before replaying history, that a reasoning-emitting provider's transcript **"fails hard, here, every time"**, and that the duty is routed to **AI-40** (AI-40.2's capability matrix, AI-40.3's compatibility statement) rather than left as an unfixed defect. The capability-vs-validation distinction is recorded with the AI-03 §10.4 citation and the AI-25/AI-26 comparison table: `PreStreamFailure` + `ErrUnsupportedCapability` (AI-19) for capability failure, distinct from AI-04's `Violation` for construction-time faults.

### Priority check 6 — Totality of the unsupported-feature policy

**Derivation is mechanical, and both counts were verified independently rather than trusted.**

Five closed vocabularies — confirmed by `grep` for the enumerator declarations in `package ai`: `PartKinds` (`content_part.go:128`), `ReasoningStates` (`reasoning_content.go:132`), `ToolChoiceModes` (`tool_choice.go:116`), `CacheRegions` (`cache_boundary.go:67`), `Roles` (`role.go:96`). Exactly five. Each body was read: all five iterate `first..end` sentinel bounds over their declared constant space, so a newly declared member is enumerated automatically with no edit. Member counts 4/3/4/3/3 = 17.

Ten `With*` constructors — confirmed by `grep -rn "^func With" *.go | grep -v _test.go` in `package ai`: exactly **10** (`WithProviderExtension`, `WithModel`, `WithMessages`, `WithMaxOutputTokens`, `WithTemperature`, `WithTopP`, `WithStopSequences`, `WithTools`, `WithToolChoice`, `WithSystemInstruction`), across `request.go` (8), `request_extension.go` (1), `system_instruction.go` (1). The production guard hard-asserts this count and fails loudly if it changes.

The scan is a `go/ast` / `go/parser` walk (`requestOptionConstructorNames`), matching an exported top-level func named `With*` whose single result is `RequestOption` — two independent conditions, not one. Reflection is rejected with a substantive reason recorded in `doc.go` and in the test header: `reflect` inspects values already in hand and has no operation enumerating a package's top-level function declarations.

The policy table is total over the inventory: 17 vocabulary rows + 9 unsplit constructors + `WithProviderExtension` split into 2 (own namespace translates, foreign drops, genuinely different dispositions) = **28 rows**, verified by reading `policy.go`. The walk is inventory-driven (`fullInventory` → `unresolvedPolicyFeatures`), never list-driven.

**The bite proofs are durable, and they would genuinely fail if the guard no-opped.** This is the deliberate improvement over AI-27's transient stage-and-revert proof, which AI-27's own verify flagged.

- `TestFeatureInventory_UnaccountedVocabularyMemberFailsNamingIt` — feeds a synthetic candidate set through the **production** `unaccountedFeatures` and asserts `slices.Equal(missing, []string{scratchMember})`. Because the assertion demands a **non-empty** result, a no-opped guard returning `nil` fails the test. Falsification confirmed by inspection.
- `TestFeatureInventory_UnaccountedConstructorFailsNamingIt` — writes a synthetic `WithScratchEleventhOption` into a `t.TempDir()` and runs the **real** `go/ast` scanner against it end to end, then the real comparison. Proves the scanning mechanism itself notices a newly declared constructor, not merely the downstream comparison.
- `TestFeatureInventory_ScanReflectsTheTargetDirectoryNotAFixedList` — three fixtures producing **zero, one and three** names, the last spread across two files and mixed with an unexported func and a wrong-return-type func. A hand-maintained mirror would return the same answer for all three, so this directly kills the "same fixture regardless of case" shape.
- `TestPolicyWalk_UnaccountedFeatureFailsTheWalk` — the same technique at the **walk** layer via the shared `unresolvedPolicyFeatures`.

The factoring that made this possible is real: `requestOptionConstructorNames(t, dir)` takes a directory (production passes `".."`, proofs pass `t.TempDir()`), and `unaccountedFeatures(inventoried, policied []string)` takes plain slices. Production and proof funnel through the identical functions, so they cannot drift apart. No file under `package ai` was staged or reverted at any point, which is why `NFR-ART-B` held continuously rather than only at session end.

### Priority check 7 — Evidence-class honesty

Four disclosures spot-checked against the actual test code. All four hold up; none reads as rationalisation.

1. **`S-ART-020` green on first write** (task 2.1) — claim: slice 1's structure already omits an absent system instruction. Verified: `appendMessagesField` guards on `if system, hasSystem := req.SystemInstruction(); hasSystem`, and tracks a `wrote` flag rather than assuming messages are the sole source. The green is genuinely structural. The companion `S-ART-018/019` reds are recorded as genuine (before `system.go` existed). Honest.
2. **`S-ART-023` staged marker mutation** (task 2.5) — claim: staged `,"scratch_cache_boundary_mutation":true` in `appendMessageObject` and `appendSystemMessageObject`, red observed, reverted. Plausible and falsifying: the twin comparison asserts marked-vs-unmarked byte-identity, so any marker-conditional field breaks it. The disclosure that the system sub-test was *also* found green-on-first-write, and therefore needed its own second mutation, is the kind of admission a fabricated log would omit.
3. **`S-ART-059/060` green on first write** (task 4.5) — claim: opt-in bytes emitted unconditionally since slice 1, so this node owns the *assertion*, not the bytes. Verified against `TestExpectationCases_AllCarryUsageOptIn`: it walks the **complete** shared registry (not a sample), asserts positive containment of `"stream_options":{"include_usage":true}`, and carries an explicit `len(expectationCases) == 0 → t.Fatal` vacuity guard. It would fail if `appendStreamFields` dropped the opt-in. Honest and correctly scoped.
4. **`S-ART-030` adjacent-swap, falsified by a message-order-reversal mutation** (task 5.4/5.6) — this one was checked adversarially, because a full list reversal is a bijection and would leave a pure inequality assertion still passing, i.e. a vacuous proof. It does not apply here: `TestMessage_AdjacentSwap_TranslatesDifferently` anchors against **two hard-coded expectation literals** (`wantForward` and `wantSwapped`) in addition to the inequality check, so a reversal mutation genuinely breaks it by name. The tasks.md claim is accurate.

**Both catalogued vacuous-pass shapes that bit this milestone are fixed, and the fixes held.**

- *Coverage helper building the same fixture regardless of case*: `TestMessage_PartKindCoverage` now builds a genuinely distinct fixture per kind inside its switch (user text / assistant reasoning / assistant tool call / tool result plus its correlating call), and its `default:` branch fails when a kind has a disposition entry but no exercise branch. The shared-fixture shape is gone.
- *Mutation that only disabled a check while reproducing a pre-existing unrelated panic*: task 8.10 explicitly mutated **both** the check (`refuseReasoning` returning `nil`) **and** the render path (`appendMessageContent`/`appendContentPartObject`), so `Translate` genuinely succeeded with content silently gone rather than panicking. 19 sub-tests failed by name. The structural reason this matters is confirmed in source: `appendContentPartObject`'s default case panics on an unreadable kind, so disabling only the check would have reproduced a panic and proven nothing about silent dropping.

### Priority check 8 — The known validation trap

No fixture reintroduces it. `ai.NewRequest`'s `unresolvedToolResultRule` rejects a tool result correlating to no tool call at construction, and the package's `mustRequest`/`mustPart` helpers panic on error, which in a Go subtest would abort the whole test binary. `TestMessage_PartKindCoverage`'s `PartKindToolResult` branch supplies the satisfying tool call as a **second message** and documents exactly why. `policy_walk_test.go`'s purity pin pairs `NewToolCall("call_1", ...)` with `NewToolResult("call_1", ...)`. Runtime evidence is stronger than inspection here: a violating fixture would abort the binary, and the suite completed cleanly with 1,805 passes across five independent invocations, so no such fixture exists anywhere in the package.

### Priority check 9 — Regression and hygiene

- `make test` → exit **0**; 1805 PASS, 0 FAIL, 8 pre-existing SKIP, **0 DATA RACE**.
- `make lint` → exit **0**; `go vet ./...` clean, `golangci-lint run` → `0 issues.`
- `go.mod` → zero `require` directives, no `go.sum`.
- AI-00 guard 1 `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` → PASS.
- AI-00 guard 2 `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage` → PASS.
- AI-25 ambient-authority guard → all four `TestAmbientAuthority_*` PASS. **No allowlist edit occurred, and none was possible**: `scanNonTestSourcesForAmbientAuthority(t, ".")` selects files by a uniform `_test.go` suffix rule with no allowlist or exception list anywhere. `ambient_authority_test.go` is not among the 20 files the chain touched, confirming it was not weakened. AI-26's eight new non-test sources (`body.go`, `doc.go`, `message.go`, `option.go`, `policy.go`, `system.go`, `tool.go`, `translation.go`) are therefore scanned automatically.

**`gofmt` disclosure adjudicated — the claim is TRUE.** `gofmt -l .` flags exactly one file, `src/ai/completion_test.go`. From `git log`, its last touching commit is `2657d9b`, dated **2026-08-01**, subject "test(ai): close out AI-15 NFRs - totality and a clean make lint (AI-15 NFR)". `git log feat/ai-25c-test-server-viability..HEAD -- backend/agent/src/ai/completion_test.go` returns **empty**, and the file does not appear in the chain's 20-file diff. Pre-existing and unrelated, exactly as disclosed. Note `make lint` is the gate and it passes; `gofmt -l` is not part of it.

### Spec Compliance Matrix (summary)

| Group | Scenarios | Result |
|---|---|---|
| Runnable scenarios | 72 | COMPLIANT — covered by the `openaicompat` suite, all green under `-race` across 5 independent runs |
| Review obligations | 17 | COMPLIANT — each discharged by landed prose read directly in `doc.go`, `tool.go`, `system.go`, `translation_test.go` and `tasks.md`, with one qualification (WARNING 1) |
| **Total** | **89/89** | |

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | PASS | TDD Cycle Evidence tables present in apply-progress and per-phase Evidence Logs in `tasks.md` |
| All tasks have tests | PASS | 12 test files across 9 phases; every runnable scenario mapped |
| RED confirmed (tests exist) | PASS | All named test files exist and were read |
| GREEN confirmed (tests pass) | PASS | Re-executed independently: exit 0, 1805 PASS, 0 FAIL |
| Triangulation adequate | PASS | Registry-driven tables; `PartKindCoverage` 4 kinds, refuse witnesses 4, drop witnesses 4, scan fixtures 3 |
| Safety net for modified files | PASS | Full suite re-run per slice; `package ai` never modified |

**Assertion quality**: no tautologies, no orphan empty-collection assertions, no type-only assertions, no ghost loops. Every registry walk that could iterate an empty collection carries an explicit vacuity guard (`expectationCases`, `fullInventory`, credential surface). Assertions compare concrete wire bytes against inline literals.

### Test Layer Distribution

| Layer | Files | Notes |
|-------|-------|-------|
| Unit (pure value comparison) | 12 | Entire AI-26 surface; no client, no test server, no socket |
| Integration | 0 | Deliberate — `NFR-ART-C` forbids it |
| E2E | 0 | Deliberate |

### Issues Found

**CRITICAL**: None.

**WARNING (2)**:

1. **Dangling citation pointer: "Claim 1.0.5" does not exist.** `doc.go:497` and `message.go:222` both attribute the wire-shape claim that `content` is `oneOf[string, array]` — and specifically *not* `oneOf[string, array, null]` — to "Claim 1.0.5 (doc.go's wire-shape provenance section)". That section enumerates exactly four claims, numbered 1, 2, 3 and 4; there is no claim 1.0.5, and claim 1 concerns system-instruction placement and role, saying nothing about `content`'s type union. This claim is load-bearing twice over: it licenses rendering a single content part as a bare JSON string rather than an array, and it licenses **omitting** `content` rather than emitting `null` for an assistant message carrying only tool calls. It is not one of `R-ART-001`'s four gated claims, so the formal citation gate is not breached, and the resulting behaviour is the conservative one (omit, never fabricate a value) and is separately proven byte-exact. But a wire-shape claim resting on an unresolvable pointer reads as cited when it is not, which is precisely the inference-mistaken-for-fact failure mode this milestone exists to prevent. Recommend repointing to a real locator in the pinned OpenAPI spec, or demoting it to an explicitly recorded inference, before archive promotes the spec.

2. **The namespace coordinator ruling carries no reopen trigger.** The other two rulings each carry an explicit living-graph reopen trigger (system role: "reopens when pointed at a first-party endpoint requiring `developer`"; reasoning: "revisited if a future vendor update adds a wire field"). The `Namespace = "openaicompat"` ruling records its rationale in three bullets but names no condition under which it would be revisited — for example a downstream collision with another adapter reserving the same string, or a second OpenAI-compatible adapter in the same binary. Minor and non-blocking, but it is an asymmetry against this milestone's own stated discipline.

**SUGGESTION (4)**:

1. **Credential-scan scope has a narrow residual hole.** `scanCredentialSurface` only scans files whose package clause is `package openaicompat_test`, deliberately excluding internal-package test files (documented, with a real reason: AI-25's `credential_test.go` intentionally embeds a key-shaped literal). Slice 8 added two internal-package test files. They carry no expectations, so `S-ART-014`'s letter holds. But "coverage grows automatically" is now true only for *external* test files; a future internal-package test file with a credential-shaped literal would escape the scan with no edit and no failure.
2. **The vocabulary bite proof does not exercise `discoverVocabularyFeatures`.** It feeds a synthetic string list straight to `unaccountedFeatures`, proving the comparison bites but not that a real new vocabulary member flows into the inventory. The gap is closed structurally — all five enumerators iterate `first..end` sentinel bounds, verified by reading each — and the constructor proof *does* exercise its real scanner end to end. Worth noting only because the two halves have asymmetric proof depth.
3. **Every slice exceeds the shared 400-line PR review budget** (340–1,261 changed lines). The prescribed mitigation, a chained-PR feature branch chain, was used, and all slices are within the 5,000-line budget this milestone was given. Informational.
4. **`mustRequest`/`mustPart` panic semantics are understated.** `translation_test.go`'s comment says a construction failure "go test reports as a failed subtest"; an unrecovered panic actually aborts the whole test binary. Inert today because every fixture is valid, but the comment invites a future contributor to treat a panicking helper as locally contained.

### Verdict

**PASS WITH WARNINGS** — all 21 requirements, all 6 NFRs and all 99 tasks are discharged with runtime evidence independently re-executed; the citation gate, byte-fidelity, cross-run determinism and policy-totality checks each hold under adversarial inspection; two documentation-level warnings remain, neither blocking archive.
