```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:9c00f7ebdbea26cf1894ad8880d8aa3d20e90c604596d883e66a05b6ad49d79d
verdict: pass
blockers: 0
critical_findings: 0
requirements: 116/116
scenarios: 420/420
test_command: make test
test_exit_code: 0
test_output_hash: sha256:9c00f7ebdbea26cf1894ad8880d8aa3d20e90c604596d883e66a05b6ad49d79d
build_command: make lint
build_exit_code: 0
build_output_hash: sha256:66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a
```

## Verification Report — Layer 1 Wave 1 (AI-04 … AI-13)

**Change**: `cachicamas-ai-layer-1` Wave 1 — ten SDD changes verified as one deliverable
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-1`
**Branch / head**: `feat/2026-07-31-cachicamas-ai-layer1-wave-1` @ `e76bab8` (74 commits)
**Charter**: `docs/architecture/milestones/0002-cachicamas-ai-layer-1-task-graph.md` lines 361–850
**Mode**: Strict TDD · hybrid persistence · `exception-ok` single PR, >5000 lines accepted at preflight

**Verdict**: **PASS WITH WARNINGS** — 0 CRITICAL, 4 WARNING, 4 SUGGESTION. Nothing blocks merge or archive. Two of the four warnings are obligations the archive phase must discharge before the deltas are frozen into `openspec/specs/`.

---

## 1. Completeness

| Metric | Value |
| --- | --- |
| Changes verified | 10 / 10 |
| Task checkboxes total | 345 |
| Task checkboxes complete | 345 |
| Task checkboxes incomplete | **0** |
| Delta-spec requirements | 116 |
| Delta-spec scenarios | 420 |
| doc 0002 Wave 1 leaves | 33 (`[leaf]`) + 2 `[decision]` + 1 `[guard]` |
| doc 0002 leaf test-list items with a named, passing covering test | **all** |
| Production Go files | 19 |
| Test Go files | 27 |
| PR diff (three-dot vs `efdedc4`) | 100 files, 33 005 insertions, 4 deletions |

Per-change task state:

| Change | Boxes | Complete | Open |
| --- | --- | --- | --- |
| `cachicamas-ai-validation-errors` | 37 | 37 | 0 |
| `cachicamas-ai-message-roles` | 41 | 41 | 0 |
| `cachicamas-ai-content-parts` | 4 | 4 | 0 |
| `cachicamas-ai-reasoning-content` | 0 (prose-structured) | — | 0 |
| `cachicamas-ai-tool-declarations` | 43 | 43 | 0 |
| `cachicamas-ai-tool-messages` | 13 | 13 | 0 |
| `cachicamas-ai-model-request` | 25 | 25 | 0 |
| `cachicamas-ai-cache-breakpoints` | 66 | 66 | 0 |
| `cachicamas-ai-request-extension-points` | 69 | 69 | 0 |
| `cachicamas-ai-completion-metadata` | 47 | 47 | 0 |

---

## 2. Evidence gate — recorded execution

All commands run from `backend/agent/` in the Wave 1 worktree at head `e76bab8`.

**Tests** — `make test` (= `go test -race -v ./...`)

```text
$ make test
go test -race -v ./...
...
ok  	github.com/cachicamas/backend/agent/src/agenttest	1.491s
ok  	github.com/cachicamas/backend/agent/src/ai	2.369s
exit 0
```

- 176 top-level tests PASS, 745 PASS lines including subtests
- 0 FAIL, 0 SKIP
- `test_output_hash`: `sha256:9c00f7ebdbea26cf1894ad8880d8aa3d20e90c604596d883e66a05b6ad49d79d`

**Lint** — `make lint`

```text
$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
exit 0
```

- `build_output_hash`: `sha256:66d9a3373b26e70b4206ef2aab426698da81f1f718bd3e1c6bb58b06ca3eb38a`

**Module purity** — `backend/agent/go.mod`

```text
module github.com/cachicamas/backend/agent

go 1.26.3
```

Zero `require` lines. Confirmed.

**Dependency closure** — `go list -deps ./src/ai`

43 packages, all Go standard library plus the module's own `src/ai`. Present: `bytes cmp errors io iter math/bits runtime slices strconv strings sync sync/atomic unicode unicode/utf8` and their `internal/*` support. **Absent**: `fmt`, `os`, `net`, `net/http`, `io/fs`, `encoding/json`. Confirmed by inspection and by the guard below.

**Named guards, re-run individually**

```text
$ go test -race -count=1 -v -run 'TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault|TestLayer1_ModuleHasNoDependencies_ZeroRequires|TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage|TestRuleClasses_' ./src/ai/
--- PASS: TestRuleClasses_EverySentinelDeclaredInTheSource_IsInTheRegistry (0.01s)
--- PASS: TestRuleClasses_TheExternalTestMirror_MatchesTheRegistryExactly (0.01s)
--- PASS: TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage (0.04s)
--- PASS: TestLayer1_ModuleHasNoDependencies_ZeroRequires (0.06s)
--- PASS: TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault (0.06s)
ok  	github.com/cachicamas/backend/agent/src/ai	1.654s
```

**AI-00 reverse guard**, run from `backend/database_administrator/`:

```text
$ go test -count=1 -run 'Import|import' ./src/domain/
ok  	github.com/cachicamas/backend/database_administrator/src/domain	0.408s
```

`TestDomainLayer_DoesNotImportAgentModule` and `TestModule_OnlyApplicationAndCmdServerMayImportAgentModule` both pass. Both AI-00 import guards — forward (`backend/agent/src/ai/import_boundary_test.go`) and reverse (`backend/database_administrator/src/domain/imports_test.go`) — hold.

**AI-10.4 dependency-closure guard** — `TestRequestPath_DependencyClosure_ContainsNoNetworkOrFilesystemPackage`, `backend/agent/src/ai/import_boundary_test.go:278`. Uses `go list -deps` **without** `-test` (deliberate and documented at lines 270–277), scoped to `github.com/cachicamas/backend/agent/src/ai`, checking against `net`, `net/http`, `os`, `io/fs`. `fmt` and `encoding/json` are caught transitively through `os`. Passes.

---

## 3. Per-milestone verdict

| Milestone | Change | Charter Goal / Deliverable / Acceptance | Test evidence | Verdict |
| --- | --- | --- | --- | --- |
| **AI-04** | `cachicamas-ai-validation-errors` | Met. Typed sentinels with `errors.Is`/`errors.As`, positional carrier, no content body. `decision.md` closes all four AI-04.1 checklist items with alternatives at full strength. | `validation_test.go` (6 tests), `validation_registry_internal_test.go` (2 go/ast guards). Every AI-04.2 and AI-04.3 test-list item has a named test. | **PASS** |
| **AI-05** | `cachicamas-ai-message-roles` | Met. Closed role vocabulary, `MessageID` unforgeable, ordered content, copy-on-construct and copy-on-read. | `role_test.go` (4), `message_test.go` (6). All nine AI-05.1–05.3 items covered; extreme-input panic test appended. | **PASS** |
| **AI-06** | `cachicamas-ai-content-parts` | Met — the keystone holds. One opaque `Part` value type satisfies readability **and** the seal simultaneously; `decision.md` demonstrates both properties before code, and compares six alternatives at full strength. | `content_part_test.go` (8), `content_part_registry_test.go` (2, go/ast over the `PartKind` constant space), `agenttest/content_part_test.go` (external-package byte-equal round trip), `testdata/handrolled` + `testdata/constructed` compile-failure pair. AI-06.4 `[guard]` closed with **three recorded bites, all dropped** — no `PartKindScratch` survives in the tree. | **PASS** |
| **AI-07** | `cachicamas-ai-reasoning-content` | Met. Reasoning as a part on AI-06's strategy, closed state vocabulary, opaque token with absence distinguishable from empty, byte-exact round trip, redacted and signature-only shapes. | `reasoning_content_test.go` (13). All eleven AI-07.1–07.4 items covered. Two reds honestly labelled "re-observed" with the reason. | **PASS** |
| **AI-08** | `cachicamas-ai-tool-declarations` | Met. Byte-faithful schema bytes, deterministic tool-set order, tool choice cross-validated against the declared set. | `tool_test.go` (3), `tool_set_test.go` (4), `tool_choice_test.go` (5). Sampled `TestNewTool_BrokenConstructionRules_FailWithTheDocumentedSentinels`: eight cases with `want`/`notWant` sentinel discipline mapping exactly onto S-ATD-003/007–013. | **PASS** |
| **AI-09** | `cachicamas-ai-tool-messages` | Met. Tool call with identity/name/exact argument bytes and derived ordinal; tool result with correlation, content and failure indication. | `tool_call_test.go` (7), `tool_result_test.go` (4). All nine AI-09.1–09.3 items covered. Ordinal is derived from position, not stored — matches the design. | **PASS** |
| **AI-10** | `cachicamas-ai-model-request` | Met. Segmented system instruction from birth, ordered messages, tool set + choice, options, validation once before I/O, whole-request round trip, immutability and exported equality. | `request_test.go` (34), `system_instruction_test.go` (8), `request_internal_test.go` (3), `import_boundary_test.go` (1), `agenttest/request_test.go` (2). All nineteen AI-10.1–10.6 items covered. Role/kind table proven in **both directions, twelve cells**, with a hard `len(cases) != 12` assertion. | **PASS** |
| **AI-11** | `cachicamas-ai-cache-breakpoints` | Met. Markers on segments, tools and messages; `MaxCacheBoundaries = 4` documented and enforced; tools → system → messages cascade order; markers advisory by contract. | `cache_boundary_test.go` (29), `agenttest/cache_boundary_test.go` (5, including a blind translator and an aware one). All seven AI-11.1–11.3 items covered plus one appended item (11.1 item 6). | **PASS** |
| **AI-12** | `cachicamas-ai-request-extension-points` | Met. Copy-on-write rebuild reaching all eleven regions, per-request option overrides sharing one rule list with construction, namespaced typed-opaque pass-through inert in validation, read-back determinism. | `request_test.go` (11 derive-related), `request_extension_test.go` (10), `request_internal_test.go` (1). All eight AI-12.1–12.4 items covered plus two appended. Eight bite proofs recorded and reverted. | **PASS** |
| **AI-13** | `cachicamas-ai-completion-metadata` | Met. Seven-value finish-reason vocabulary with refusal and pause-turn from birth; usage with absent distinguishable from zero on every field; cost formula pinned. | `finish_reason_test.go` (7), `usage_test.go` (5). All twelve AI-13.1–13.4 items covered. The exhaustiveness pin scans the whole 0–255 value space against `Validate()` — the strongest pin in the wave. | **PASS** |

---

## 4. doc 0002 traceability (lines 361–850)

Every leaf's test list was implemented or appended-to. No test-list item was substituted, pruned, or silently dropped. Appended items are labelled *(appended)* in their owning leaf and traceable to a discovery event:

| Appended item | Owning leaf | Discovery |
| --- | --- | --- |
| Rule-class registry guard (`T-AIE-12`) | AI-04 | AI-08's `ErrDuplicate` append exposed that the external-test mirror could drift |
| `ErrDuplicate` rule class | AI-04 set, appended by AI-08.2 | Duplicate tool names are decidable from the request and match none of the five original classes |
| `ErrMisplaced` rule class | AI-04 set, appended by AI-10.3 | Well-formed value in a forbidden placement matched none of the six landed classes |
| `agenttest` round-trip pin extension (item 6) | AI-11.1 | AI-10.5's pin never calls `Request.Equal`; it drives hand-written helpers that would stay green on a request that dropped every marker |
| Totality-table row 11 (`provider_extensions`) | AI-12.1 | Re-reading `R-REX-002` while implementing AI-12.3 |
| Deferred extension-value sibling (task 3.10) | AI-12.1 item 3 | Opaque-payload pin needed to cover extension values, not only reasoning tokens |

**`[decision]` nodes.** AI-04.1 (`cachicamas-ai-validation-errors/decision.md`) and AI-06.1 (`cachicamas-ai-content-parts/decision.md`) each close on exactly the four items their charter checklist names, each with the losing alternatives stated at full strength before being rejected. Both exceed the bar.

**`[guard]` node.** AI-06.4 closes on its charter's two items and on `decision.md` § 8's requirement to be recorded failing twice. Three bites are recorded verbatim; all scratch state was dropped (`grep PartKindScratch` over production source returns nothing).

---

## 5. Findings raised during apply — confirmed or refuted

### 5.1 `S-AMR-046` — **CONFIRMED**, cosmetic, archive-owned

`openspec/changes/cachicamas-ai-model-request/specs/ai-model-request/spec.md:247` reads "*Given each of the four permitted cells in turn…*". The count is wrong.

- `R-AMR-011`'s own table (same file, lines 229–234) yields **five** permitted cells: `text`/`user`, `text`/`assistant`, `reasoning`/`assistant`, `tool_call`/`assistant`, `tool_result`/`tool`.
- `backend/agent/src/ai/request.go:30-34` (`rolePermittedKinds`) yields the same five.
- `backend/agent/src/ai/request_test.go:804` labels its five permitted rows "The five permitted cells (S-AMR-046)" and asserts `len(cases) != 12` for the whole table.

The **code and the tests are correct**; only the scenario prose undercounts by one. It is not a merge blocker. It **is** an archive blocker in the narrow sense that `sdd-archive` merges this delta into `openspec/specs/ai-model-request/spec.md`, and a frozen main spec must not carry a statement its own requirement contradicts.

**Owner: `sdd-archive`.** Correct "four permitted cells" → "five permitted cells" in the delta before merging it. The apply phase was right not to edit a spec it did not own.

### 5.2 Region-level exhaustiveness guard deferred to Wave 2 — **deferral is SOUND as scoping; the recording is INCOMPLETE**

The record is at `openspec/changes/cachicamas-ai-request-extension-points/design.md` § 14, and it is unusually honest: it names all three incidents (AI-10.5's `agenttest` walk, AI-11's `Message.Equal`/`toolsEqual`, AI-12.3's `Request.Equal` plus the same two `agenttest` helpers), calls the class "a real, repeating defect class, caught each time by a re-verification gate rather than by a test", and gives three reasons for deferring.

**Judged sound as a scoping decision**, on these grounds:

1. doc 0002 charters **no `[guard]` node under AI-12**. Adding one would itself be a charter deviation, and the wave's discipline elsewhere is to append test cases to existing leaves rather than invent nodes.
2. `PartKind`'s guard is a mechanical port target only in appearance. `PartKind` is a closed constant space parsed out of source with `go/ast`; `requestDraft`'s regions are struct fields in value/presence-flag pairs. A faithful guard needs a pairing heuristic that is a design decision, not a port.
3. AI-12 was already `High` risk under `size-exception` delivery.
4. The eleven-region totality table plus the extended `agenttest` comparators do give today's regions real proof. I verified `requireRequestsEqual` (`backend/agent/src/agenttest/request_test.go:370-443`) reaches all eleven regions including `requireExtensionsEqual`.

**It does not block the wave.** But two things are worth stating plainly:

- The tripwire cannot trip on the failure it guards against. `request_test.go:1745` asserts `len(regions) != 11`. That catches a **row deleted from the table**. It does **not** catch a **field added to `requestDraft`** — which is precisely the failure mode that occurred three times. The guard as landed is one-directional.
- Every one of the three catches depended on someone noticing. AI-11.1 item 6 is explicitly recorded as "the orchestrator's mid-task finding". A control that depends on an agent remembering is not a control; it is luck with good documentation.

**Owner: `sdd-archive`**, to record the obligation in doc 0002 (see § 7) rather than leaving it only in AI-12's `design.md`, which is about to be archived.

### 5.3 AI-10.4's `encoding/json` replacement — **correct for the inputs it is tested on; one real, evidenced divergence found**

`backend/agent/src/ai/json_syntax.go` replaces `encoding/json.Valid` with a hand-rolled RFC 8259 scanner, because `encoding/json` → `fmt` → `os` would break AI-10.4's own guard. The reasoning is sound and documented at the file head.

**What holds.** I read the scanner line by line against RFC 8259 and confirmed: trailing commas rejected in both objects and arrays; leading zeros rejected (`01`, `-01`); bare `.5`, `+1`, `1.`, `1e` rejected; unterminated strings rejected; unescaped control bytes below `0x20` rejected inside strings; `\uXXXX` bounds-checked correctly (`i+4 >= len(data)`); `Infinity`/`NaN` rejected; whitespace limited to RFC 8259 § 2's four bytes; exactly one top-level value with nothing after it. It deliberately does **not** validate surrogate pairing or UTF-8 inside strings — and neither does `encoding/json.Valid`, so that matches.

**The differential test is real evidence.** `json_syntax_differential_internal_test.go` runs 20 000 deterministically seeded generated inputs against `encoding/json.Valid` as an oracle, importing `encoding/json` only from a `_test.go` — which the import guard correctly excludes by omitting `-test`. It passes. This is genuine differential testing, not a token gesture.

**The divergence.** `encoding/json.Valid` caps nesting at 10 000; `isWellFormedJSON` has no depth cap and recurses through `scanJSONValue`. I reproduced this with a verbatim copy of the scanner:

```text
depth    100: json.Valid=true  handrolled=true  AGREE
depth   5000: json.Valid=true  handrolled=true  AGREE
depth  10001: json.Valid=false handrolled=true  *** DISAGREE ***
depth  50000: json.Valid=false handrolled=true  *** DISAGREE ***
depth 2000000: json.Valid=false handrolled=true  *** DISAGREE ***
depth 20000000: runtime: goroutine stack exceeds 1000000000-byte limit
                fatal error: stack overflow
```

At roughly 20 million nesting levels — a ~40 MB byte string — the scanner triggers an **unrecoverable** `fatal error: stack overflow`, which `recover()` cannot catch. `ai.NewToolCall` applies no length or depth bound to argument bytes before calling it (`backend/agent/src/ai/tool_call.go:91`, `:121`).

The differential test does not catch this: `writeJSONishValue` stops offering objects and arrays at `depth >= 4`, so the corpus's maximum nesting is about five. The file's own doc comment at line 28 claims the corpus covers "deeply nested structures" — that claim is not true as written.

**Why it is not blocking.** Layer 1 is a pure in-memory contract with no transport and no untrusted input path until AI-24. `R-AIE-009`'s totality claim is scoped to constructing, matching, extracting from and rendering a **failure value**, not to the JSON scanner, so this does not violate a landed requirement. But it is a real defect with a reproduction, and Wave 2 should close it before any adapter feeds provider bytes into `NewToolCall`.

### 5.4 AI-12 process deviation — **the account is internally consistent and strongly corroborated; the commit history neither confirms nor refutes it**

The self-report is at `openspec/changes/cachicamas-ai-request-extension-points/tasks.md:176`: tasks 1.1/1.1a/1.2 landed as one atomic behaviour-preserving refactor because `func (d requestDraft) rules() []Rule` cannot compile until the draft carries `model`/`messages`. The stated safety net is the pre-existing suite with zero test files edited, and the RED transcripts state that `With` landed as a compiling stub returning `Request{}, nil` and `WithModel`/`WithMessages` as compiling no-ops before real implementation.

**What the commit history can and cannot show.** The entire leaf is one commit, `ad4c942`. Every leaf in this wave is one commit. Commit history therefore **cannot** corroborate intra-leaf RED-first ordering, and I will not claim it does.

**What does corroborate it.** The recorded RED failure strings match `t.Errorf`/`t.Fatalf` format strings in the landed test file verbatim:

- `request_test.go:1479` — ``request.Model() = %q, want "b" — ai.WithModel must win over the constructor's own "a" parameter``
- `request_test.go:1519` — `derived.Messages() carries %d messages in the wrong order, want [first, second] as supplied`
- `request_test.go:1758` — `region %q: the rebuild path did not reach it — the derived request does not observe the supplied change`
- `request_test.go:1432` — `derived.Temperature() = (%v, %t), want (0.99, true)`

All four named test functions exist at the lines the transcripts imply (±1). A fabricated transcript does not reproduce four distinct format strings, with their em-dashes and their argument order, at near-exact line offsets. The recorded line numbers are consistently one lower than the landed `t.Errorf` lines, which is what a transcript captured before a later comment insertion looks like — an artifact of a real recording, not of a reconstruction.

**Conclusion: the account is credible and I found nothing contradicting it.** The transcripts are observed, not reconstructed. The residual honest gap is structural, not a fault of this milestone: one-commit-per-leaf makes intra-leaf TDD ordering unauditable from git for the whole wave.

---

## 6. Vocabulary register integrity

`openspec/specs/ai-contract-vocabulary/spec.md` is live and was appended to correctly.

| Check | Result |
| --- | --- |
| Register amended append-only, never frozen | **Yes.** One amendment by AI-04.1 adding `V-FAIL-16` (**validation rule**) and `V-FAIL-17` (**rule class**), with a full amendment blockquote in § 6, both deferring substance to AI-04 per § 9 rule 5. |
| No existing row renumbered, reworded, reordered or removed | **Confirmed** by diff. |
| Term count updated | **Yes.** 114 → 116; failure-category count 15 → 17; § 10 checklist row 4's range updated `V-FAIL-01 … V-FAIL-15` → `… V-FAIL-17`. |
| Every `V-*` term cited by Wave 1 resolves | **Yes.** 74 distinct `V-*` identifiers cited across the ten changes and the Go source; **zero** unresolved. |
| Any milestone silently invented a term | **No.** AI-05 … AI-13 cite only existing terms; only AI-04 appended, and it recorded why for both terms. |

**Rule-class set and both registry mirrors agree.** All three views carry the same seven classes in the same order:

- `backend/agent/src/ai/validation.go:131` — `ruleClasses = []error{ErrEmpty, ErrNotInVocabulary, ErrOutOfRange, ErrMalformed, ErrUnresolvedReference, ErrDuplicate, ErrMisplaced}`
- The `errors.New` sentinel declarations in the same file, read via `go/ast`
- `backend/agent/src/ai/validation_test.go:97` — the `ai_test`-package mirror, qualified

`validation_registry_internal_test.go` enforces all three bindings mechanically, including order, and binds the AST to the running slice by comparing each element's own text against its declaration's string literal — so the guard proves something about the slice the package actually iterates, not merely that two source files agree. Both guard tests pass.

---

## 7. doc 0002 amendments Wave 1 owes — for `sdd-archive` to make

Wave 1 did not touch `docs/` at all (`git log efdedc4...HEAD -- docs/` is empty). That was correct for apply, but it leaves five amendments outstanding. **I have not made any of them.**

### 7.1 Shipped-milestone counter (line 3) — stale by fourteen

Current:

> **Status:** Not started — **0 of 41** milestones shipped. **AI-00 is the first milestone.** Neither the `backend/agent` module nor a single line of Layer 1 exists on disk; …

This was already stale before Wave 1: Wave 0 (AI-00 … AI-03) was archived on `main` in `ead6ac8` and the counter was never moved. After Wave 1 it should read **14 of 41** (AI-00 … AI-13), and the "Not started" label and the "Neither the `backend/agent` module nor a single line of Layer 1 exists on disk" sentence are both now false — the module exists with 19 production files and 27 test files.

Line 13's authoring constraint ("this document can cite **no shipped code as evidence**, because none exists") is likewise now false and needs re-scoping to "no *unshipped* milestone may cite code".

### 7.2 Layer 1 completion checklist (near line 2271) — three boxes closeable, one partially

All eighteen boxes are currently unchecked. Wave 1 closes:

| # | Item | Disposition |
| --- | --- | --- |
| 3 | Neutral request, message, content and tool contracts are documented and tested | **Check** — AI-05 … AI-10 complete |
| 4 | Every content-part variant readable from another package, no unconstructed value reaches a request | **Check** — AI-06.2, AI-06.3, AI-10.5 complete, proven from `agenttest` |
| 5 | Cache breakpoints expressible; per-request options and escape hatch exist; request rebuildable without mutation | **Check** — AI-11, AI-12 complete |
| 6 | Provider round-trip tokens survive byte-exact through normalization, rebuild, **and the wire** | **Leave open** — AI-07.3 and AI-12.1 done; AI-26.6 and AI-29.2 (the wire) are Wave 2+ |

Items 1 and 2 were closed by Wave 0 and should be checked at the same time if the Wave 0 archive did not.

### 7.3 Traceability spine (near line 2325) — mark the closed rows

The spine's rows name nodes, not status. Wave 1 makes these rows *actually* closed for their Layer 1 half; the archive should mark them rather than leaving forward-looking prose:

| Spine row | Wave 1 status |
| --- | --- |
| **C1** — unconstructed content part passes validation | **Closed.** AI-06.1 → AI-06.3 → AI-06.4 all landed and guarded. |
| **C2** — content unreadable from another package | **Closed for Layer 1.** AI-06.2, AI-07.1, AI-09.1, AI-09.3, AI-10.5 landed. |
| **G4** — cache breakpoints, Layer 1 half | **Closed.** AI-10.2 + AI-11.1 … AI-11.3. AI-26.2 rendering remains. |
| **G5** — tool-call ordinal survives normalization | **Layer 1 half closed.** AI-09.2 landed. AI-18.3, AI-30.5 remain. |
| **G9** — per-request options and escape hatch | **Closed.** AI-12.1 … AI-12.4 landed. AI-26.7 rendering remains. |
| **G12(b)** — reasoning round-trip token | **Layer 1 half closed.** AI-07.2 … AI-07.4 landed; AI-12.1's rebuild pin extends it. |
| **G12(c)** — refusal and pause finish reasons | **Closed.** AI-13.1, AI-13.2 landed with all seven values. |
| Leakage register row 8 | **Layer 1 half closed** by AI-10.2. |
| Leakage register row 9 | **Closed** by AI-11.3. |

The "Named Layer 2 / Layer 3 forward requirements" table already marks **G10** and **G11** as "met"; those are now met in fact rather than in plan (AI-13.3/13.4 and AI-12.1 respectively). No edit required, but the archive may want to cite the landed node.

### 7.4 New Wave 2 obligation to record — the region-exhaustiveness guard

Under doc 0002's living-graph / revert-and-record clause, record a Wave 2 obligation for the deferral in § 5.2. It currently lives only in `cachicamas-ai-request-extension-points/design.md` § 14 and that change's closing checklist, both of which are about to be archived. Suggested placement: as a `[guard]` node under the first Wave 2 milestone that adds a `Request` region, or as an entry in the traceability spine's retired-findings table so it cannot be lost.

Include the specific defect: `request_test.go:1745`'s `len(regions) != 11` is one-directional and cannot fail when a field is added to `requestDraft`.

### 7.5 New Wave 2 obligation to record — the JSON scanner depth cap

Record the § 5.3 divergence: `isWellFormedJSON` needs a nesting-depth cap matching `encoding/json`'s 10 000, and the differential corpus needs a case at depth 10 001. Naturally owned by AI-24 (the first milestone with a transport) or by whichever Wave 2 milestone first feeds provider bytes into `NewToolCall`.

---

## 8. doc 0001 § 9 review checklist, run against Wave 1

**Boundaries**

- [x] No package of `backend/agent` imports another backend module, including from `cmd/` — guarded by `forbiddenPrefixes` (`database_administrator`, `workspace_syncer`, `src/agent`, `src/coding`, `src/cmd`) and passing.
- [x] Layer 1 imports only stdlib — **stronger than required**: zero non-stdlib dependencies, zero `require` lines.
- [ ] **N/A** — Layer 2 performs no I/O. No Layer 2 exists.
- [ ] **N/A** — Layer 3 reaches other modules over HTTP only. No Layer 3 exists.
- [x] Both import guards run, and the reverse guard covers the whole module — verified by direct execution of both.

**Contracts**

- [x] No vendor wire type crosses the Layer 1 method boundary — trivially, there are no vendor types.
- [ ] **N/A** — every event kind constructible. Events are AI-14, Wave 2.
- [x] Anything a provider must receive back byte-identical is carried opaquely — reasoning round-trip tokens, tool-call argument bytes, tool schema bytes and provider-extension values, all byte-exact, all proven from an external package, all proven to survive the rebuild path.
- [x] A new neutral field is genuinely neutral — AI-12.3 lands the escape hatch precisely so provider divergence never widens the neutral vocabulary; `design.md` § 14 records the principle.
- [ ] **N/A** — tool-call deltas remain optional. Deltas are AI-18, Wave 2.

**Streams and concurrency** — **all five N/A.** Nothing in Wave 1 knows what a stream is; doc 0002's own Wave 1 preamble says so. The one partially applicable clause — "run under the race detector" — is satisfied: `make test` is `go test -race -v ./...` and all 176 tests pass under it.

**Observability and safety**

- [x] No span attribute or log field carries prompt, completion, reasoning, tool-argument or tool-result text, a header, or a credential — no telemetry exists yet, but the redaction posture is landed and proven at every formatting surface: `TestViolation_RenderedMessage_CarriesNoCallerContent`, `TestPart_String_CarriesNoPayload`, `TestReasoning_DiagnosticRendering_CarriesNoPayload`, `TestToolCall_Rendering_NeverReproducesItsPayload`, `TestToolResult_Rendering_NeverReproducesItsPayload`, `TestRequest_Formatting_RendersNoRegionPayloadThroughAnyVerb`, `TestSystemInstruction_Formatting_RendersNoSegmentTextThroughAnyVerb`, `TestSegment_MarkedRendering_NamesTheBoundaryAndReproducesNoSecret`, `TestProviderExtension_Formatting_RendersNoPayloadThroughAnyVerbButNamesTheCount`, `TestRequest_Formatting_NamesTheCacheBoundaryCountAndReproducesNoSecret`. Each checks all four `fmt` verbs. This is the strongest single result in the wave.
- [x] Errors are typed and inspectable — AI-04. Mid-stream partial output is **N/A** (AI-19).
- [ ] **N/A** — process-group kill. Nothing spawns processes.

**Process**

- [x] The change fits the review budget, or says why it does not — it does not, by a factor of ~80, and **every one of the ten changes says so explicitly** in a "Review Workload Forecast" section with a "Budget reassessment — trigger 4 fired" subsection giving forecast, actual and the reason for the gap. The session preflight accepted `exception-ok`, single PR, >5000 lines up front. Honest, but see § 10.4.
- [x] Milestone identifiers are appended, never renumbered — doc 0002 untouched; no AI-NN identifier changed. (Delta-spec requirement IDs are a separate issue, § 10.2.)
- [x] Anything deliberately left unsupported is stated explicitly — each delta spec carries an out-of-scope section; AI-10 records the tool-call/result ordering rule as deliberately unlanded and pins the disposition by test; AI-12 § 14 records the region-guard deferral with its reasons.

---

## 9. Strict TDD compliance

| Check | Result | Details |
| --- | --- | --- |
| TDD evidence reported | ✅ | All ten `tasks.md` files carry RED/GREEN transcripts under their leaf sections. |
| All tasks have tests | ✅ | 345/345 boxes complete; every doc 0002 leaf test-list item maps to a named test. |
| RED confirmed (test files exist) | ✅ | 27 test files, all present, all compiled and executed. |
| GREEN confirmed (tests pass now) | ✅ | 176/176 top-level tests pass under `-race`. |
| Transcripts observed, not reconstructed | ✅ | Spot-checked AI-12's four RED transcripts against verbatim `t.Errorf` format strings in the landed source (§ 5.4). AI-07's `tasks.md` explicitly labels two reds "re-observed" with the reason rather than presenting them as first-time reds — the kind of disclosure that raises rather than lowers confidence. AI-06's `tasks.md` discloses which of its transcripts were reconstructed. |
| Triangulation adequate | ✅ | Table-driven throughout with `want`/`notWant` sentinel discipline. Sampled AI-08's construction-rule table (8 cases, each asserting one sentinel matches and two do not) and AI-10's role/kind table (12 cases with a hard count assertion). |
| Bite proofs for guards | ✅ | AI-06.4 recorded failing three times, all scratches dropped. AI-11 one, AI-12 eight, AI-09 three, AI-13 one, AI-10 two. |
| Safety net for modified files | ✅ | AI-12's atomic refactor used the pre-existing suite as an approval test with zero test files edited (`git diff --stat` recorded showing only `request.go`). |

**Assertion quality audit** — ✅ **All assertions verify real behavior.** No tautologies, no orphan empty-collection checks, no type-only assertions standing alone, no ghost loops. Positive findings worth naming:

- Every exhaustiveness guard has an explicit anti-vacuity check (`if len(x) == 0 { t.Fatal("... would pass vacuously") }`) in `import_boundary_test.go`, `role_test.go`, `validation_registry_internal_test.go` and `content_part_registry_test.go`.
- The rule-sentinel tables assert `notWant` as well as `want`, so a test cannot pass on the wrong sentinel.
- `TestFinishReason_AddingAValue_FailsWithoutATableAndAStringForm` scans the full 0–255 value space rather than a hand-list — it discovers what validates instead of asserting what it already knows.
- The redaction tests plant a distinctive sentinel body (`CACHICAMA-SENTINEL`) and assert its absence, rather than asserting the message merely "looks safe".

**Test layer distribution** — Unit: 174 tests across 23 files in `src/ai`. Cross-package integration: 8 tests across 4 files in `src/agenttest` (a genuinely separate package proving the external-reader properties). E2E: none, correctly — Layer 1 has no runnable surface. Coverage tooling is not configured in the `Makefile`; coverage analysis skipped, which is not a failure.

---

## 10. Findings

### CRITICAL — none

No requirement lacks a test. No test fails. No task is open. No guard is absent. No claim made by an apply agent was found to be untrue.

### WARNING

**W1 — `isWellFormedJSON` diverges from its oracle above nesting depth 10 000, and fatally overflows the stack at extreme depth.**
`backend/agent/src/ai/json_syntax.go:55` (`scanJSONValue`, unbounded recursion), reachable from `backend/agent/src/ai/tool_call.go:91` with no length or depth bound on argument bytes. Reproduced: agrees with `encoding/json.Valid` at depth 5 000, disagrees at 10 001 and 2 000 000, and produces an unrecoverable `fatal error: stack overflow` at 20 000 000. The differential test at `json_syntax_differential_internal_test.go:76` caps generated nesting at ~5, so it cannot see this; the file's doc comment at line 28 nonetheless claims the corpus covers "deeply nested structures". **Not blocking** — no untrusted input path exists until AI-24, and `R-AIE-009`'s totality claim is scoped to the failure value rather than the scanner. **Owner: Wave 2.** Fix is a depth cap of 10 000 plus one differential case at 10 001.

**W2 — `S-AMR-046` prose contradicts its own requirement, and will freeze into the main spec unless corrected.**
`openspec/changes/cachicamas-ai-model-request/specs/ai-model-request/spec.md:247` says "four permitted cells"; `R-AMR-011`'s table, `rolePermittedKinds` and the test all say five. Cosmetic in code, but `sdd-archive` merges this delta into `openspec/specs/ai-model-request/spec.md`. **Owner: `sdd-archive`**, before merging.

> **RESOLVED (2026-08-01) — W2 is closed.** `S-AMR-046`'s prose now reads "each of the five permitted cells". One word changed, in one line of one delta spec; the "twelve cells total" clause was already correct and was left alone. No other artifact asserted the wrong count — every other occurrence of "four" in this context is a quotation of the defect itself, kept as audit trail. No code and no test changed.

**W3 — Requirement and scenario ID collision across two capabilities.**
`R-AMR-011` means "Totality: no input causes a panic" in `cachicamas-ai-message-roles/specs/ai-message-roles/spec.md:161` and "Role versus content kind is enforced from a documented table" in `cachicamas-ai-model-request/specs/ai-model-request/spec.md:225`. The `R-AMR-`/`S-AMR-` prefix is shared by AI-05 (001–011) and AI-10 (001–063), so the ranges overlap directly. Within-file uniqueness holds and the archive merges them into different capability files, so nothing breaks mechanically — but every bare citation of `R-AMR-011` or `S-AMR-0NN` is ambiguous, including the ones in `request.go`'s comments. `R-ACP-` is *not* affected: AI-07 uses `R-ARC-` and AI-09 uses `R-ATM-`, so AI-06 holds `R-ACP-` alone. **Owner: `sdd-archive`** — either accept the ambiguity with a recorded note, or renumber AI-10's block to a distinct prefix before freezing. Renumbering after archive is worse.

**W4 — doc 0002 is fourteen milestones stale and Wave 1 owes it five amendments.**
Detailed in § 7. Not a code defect; a bookkeeping debt that compounds. **Owner: `sdd-archive`.**

### SUGGESTION

**S1 — Scenario-level traceability is mechanical for only three of ten milestones.**
115 of 420 scenario IDs are cited in test comments: AI-11 38/40, AI-12 49/54, AI-10 28/63. AI-04, AI-05, AI-06, AI-07, AI-08, AI-09 and AI-13 cite **zero**. The convention was clearly adopted from AI-10 onward. This is a *traceability* gap, not a *coverage* gap — I verified by hand that every doc 0002 leaf test-list item in all ten milestones has a named passing test, and sampled AI-08's 52 scenarios against `tool_test.go`'s table (eight scenarios mapped exactly, including their "AND does not match" clauses via `notWant`). Wave 2 should adopt the `S-*` comment convention uniformly from the first leaf.

**S2 — Closed vocabularies use three different exhaustiveness mechanisms of unequal strength.**
`PartKind` parses the constant space with `go/ast` (strongest — a new constant is seen automatically). `FinishReason` scans the full 0–255 value space against `Validate()` (also strong). `Role`, `CacheRegion` and `ReasoningState` rely on hand-maintained `xFirst`/`xEnd` sentinels (`role.go:68`, `cache_boundary.go:45`, `reasoning_content.go:111`) — `role.go`'s own GoDoc concedes that "moving it is what makes a newly declared member visible to `Roles`". A constant added without moving `roleEnd` is inert rather than broken, so the risk is contained, but this is the same hand-maintained-enumeration family as W-finding 5.2 and worth unifying on the `go/ast` pattern.

**S3 — The branch's merge base is `6a6378f`, not `efdedc4`.**
`main` advanced by two merges (`07f7131`, `efdedc4`, including the Go 1.26.5 / gRPC / x/text bumps) after this branch was cut. A two-dot `git diff efdedc4..HEAD` therefore over-reports the change as 106 files / 33 016 insertions / 81 deletions, six of which — `backend/database_administrator/{Dockerfile,Makefile,README.md,go.mod,go.sum}` and `go.work` — are apparent reversals of `main`'s work, not Wave 1 changes. The true PR diff (three-dot) is **100 files, 33 005 insertions, 4 deletions**. GitHub will compute the three-dot diff, so no reviewer will see the phantom reversals, but the branch should be rebased onto current `main` before the PR so the merge is clean and the Go toolchain line agrees.

**S4 — `sdd-phase-common.md` § E's literal guard lines are absent from eight of ten `tasks.md` files.**
Only `cachicamas-ai-cache-breakpoints` and `cachicamas-ai-request-extension-points` carry the exact `Decision needed before apply: Yes|No` / `Chained PRs recommended: Yes|No` / `400-line budget risk: Low|Medium|High` lines. The other eight carry the *substance* — a "Review Workload Forecast" table plus an explicit "Budget reassessment — trigger 4 fired" section with forecast, actual and the reason for the gap — which is arguably better prose but is not the machine-readable form the shared convention specifies. The session's cached `exception-ok` strategy resolves the decision these lines exist to force, so nothing was actually skipped. Adopt the literal lines in Wave 2 for uniformity.

**S5 — `networkOrFilesystemPackages` matches exact import paths only.**
`import_boundary_test.go:301` checks `net`, `net/http`, `os`, `io/fs` by exact string. A future `net/*` subpackage other than `net/http` (or `os/*`) would not be matched directly — though `os` itself is in almost every such closure, so the practical gap is small. Consider prefix matching with an explicit allow for the I/O-free `net/*` packages (`net/url`, `net/netip`).

---

## 11. Verdict

**PASS WITH WARNINGS.**

Wave 1 delivers all ten milestones against doc 0002 lines 361–850 with no unmet charter obligation, no untested requirement, no open task, and no failing check. The evidence gate is clean: 176 tests pass under `-race`, `golangci-lint` reports 0 issues, `go.mod` carries zero `require` lines, both AI-00 import guards and AI-10.4's dependency-closure guard pass, and `go list -deps ./src/ai` contains no `fmt`, `os`, `net`, `net/http`, `io/fs` or `encoding/json`.

Of the four findings flagged during apply: `S-AMR-046` is **confirmed** as a cosmetic prose defect the archive must fix before freezing; the region-exhaustiveness deferral is **judged sound** as scoping but its recording is incomplete and its landed tripwire is one-directional; the `encoding/json` replacement is **correct for everything it is tested on** but has a real, reproduced divergence above nesting depth 10 000 that its own differential corpus cannot reach; and AI-12's self-reported process deviation is **credible and corroborated** by verbatim format-string matches, with the honest caveat that one-commit-per-leaf makes intra-leaf ordering unauditable from git for the entire wave.

The vocabulary register was amended correctly and append-only; all 74 `V-*` terms Wave 1 cites resolve; no milestone invented one; and the rule-class set agrees across all three of its views under a mechanical `go/ast` guard.

Nothing found blocks merge or archive. The four warnings are: one Wave 2 code obligation (W1) and three bookkeeping obligations the archive phase owns (W2, W3, W4).

**Next**: `sdd-archive` — with the five doc 0002 amendments in § 7 and the W2/W3 spec corrections as preconditions.
