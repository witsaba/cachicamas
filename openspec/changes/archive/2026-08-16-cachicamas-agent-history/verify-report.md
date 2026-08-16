```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:6f15ae595248622602be254dbaf32f77cb4c82ad3fa9c351f00d5fdb38656ff4
verdict: pass_with_warnings
blockers: 0
critical_findings: 0
requirements: 9/9
scenarios: 25/25
test_command: cd backend/agent && make test
test_exit_code: 0
test_output_hash: sha256:37dde868a4afd4a4432259a0993a392451471b7526dcd36a9cfe56542e990950
build_command: cd backend/agent && make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

## Verification Report

**Change**: `cachicamas-agent-history` (AG-12 — History and the pairing invariant)
**Version**: `agent-history` v1 (new capability) + two MODIFIED deltas
**Mode**: Strict TDD
**Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/feat-agent-layer2-wave3-ag12`
**HEAD**: `ade633a6` · branch `feat/agent-layer2-wave3-ag12`, 11 commits ahead of `origin/main`

Verification method: every apply-phase claim was re-derived by command. All three
recorded REDs were re-applied to the working tree, observed failing, reverted, and the
tree re-hashed byte-identical to baseline. Working tree was clean before and after
(`git status --porcelain` empty).

### Completeness

| Metric | Value |
|--------|-------|
| Tasks total | 44 |
| Tasks complete | 43 |
| Tasks incomplete | 1 — `12.2` openspec archive move, correctly deferred to `sdd-archive` |

`tasks.md:145` is the sole `[ ]`; it is explicitly scoped out of apply. Not a finding.

### Build & Tests Execution

**Tests**: PASSED — 12/12 packages `ok`, non-cached, race-enabled.

```text
$ cd backend/agent && go test -race -count=1 ./...      # make test body, -count=1 forced
ok  github.com/cachicamas/backend/agent/src/agent                                 1.706s
ok  github.com/cachicamas/backend/agent/src/agenttest                             2.399s
ok  github.com/cachicamas/backend/agent/src/agenttest/sweep                       1.782s
ok  github.com/cachicamas/backend/agent/src/agenttest/tracetest                   1.934s
ok  github.com/cachicamas/backend/agent/src/ai                                    4.438s
ok  github.com/cachicamas/backend/agent/src/ai/internal/retry                     2.442s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat                     170.241s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/conformancetest        3.615s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter             3.051s
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/conformance 6.436s
?   .../openrouter/conformance/fixtures                              [no test files]
ok  github.com/cachicamas/backend/agent/src/ai/openaicompat/openrouter/internal/smoke 2.750s
ok  github.com/cachicamas/backend/agent/src/handoff                               2.552s
FINAL_TEST_EXIT=0
```

Package `src/agent` verbose, non-cached: **193 top-level PASS, 0 FAIL**. All 22 AG-12 test
functions plus the 4 `t.Run` sub-tests of the closed-route audit pass.

**Build**: PASSED — `make build` → `go build -trimpath ./...`, exit 0, no diagnostics.

**Lint**: PASSED — `./bin/golangci-lint cache clean` run first (the repo's recorded
stale-cache trap), then `make lint` → `go vet ./...` + `golangci-lint run` → **`0 issues.`**,
exit 0.

**Vulnerabilities**: PASSED — `make vuln-check` (govulncheck v1.1.4, db 2026-08-14, go1.26.6,
symbol scan) exit 0, **0 `"finding"` entries**. This target is NOT part of `make all` and was
run explicitly. Cleaner than the AG-10/AG-11 precedent's stdlib advisories, consistent with
the go1.26.5→1.26.6 bump.

**Coverage** (`src/agent`, changed file `history.go`):

| Symbol | Line % | Rating |
|---|---|---|
| `NewHistory`, `NewSeededHistory`, `Append`, `CloseTurn`, `Entries`, `Len`, `checkConstructed`, `Entry.ID/Message/Origin`, `commitCloseTurnOp`, `resolveOpenSet` | 100.0% | Excellent |
| `commitAppendOp` | 92.3% | Excellent |
| `SynthesizeOrphans` | 84.2% | Acceptable — uncovered arms are the documented-impossible `ai.NewToolFailure`/`ai.NewMessage` error returns |
| `commit` | 83.3% | Acceptable — uncovered arm is the unreachable `default:` |
| `EntryOrigin.String` | 0.0% | Low — diagnostic renderer, never asserted |
| `Entry.String` | 0.0% | Low — diagnostic renderer, reached only from failure branches |

Package total 72.8% (whole-package figure, dominated by pre-existing files).

### TDD Compliance

| Check | Result | Details |
|-------|--------|---------|
| TDD Evidence reported | PASS | `apply-progress.md:33-54`, 15-row TDD Cycle Evidence table |
| All tasks have tests | PASS | 4 new files; 22 test functions across the three `package agent_test` files |
| RED confirmed (tests exist) | PASS | All named test files exist and compile |
| GREEN confirmed (tests pass) | PASS | 193/193 in `src/agent`, non-cached, `-race` |
| Bites re-proven RED | PASS | 3/3 reproduced by me — see below |
| Triangulation adequate | PASS | Multi-case behavior per requirement; `S-HIS-070` uses N=2 orphans interleaved with matched calls |
| Safety Net for modified files | PASS | Phases 2–9 report running counts; the substrate guards themselves confirm it |
| Honest non-REDs | PASS | Phases 5, 7, 8 record "already-GREEN pin" instead of fabricating a RED — the correct disclosure |

#### Bite reproduction (the core of this verification)

Each bite was re-applied to the real working tree, run, and reverted; SHA-256 of every
touched file was compared against the pre-edit baseline afterwards.

**`S-HIS-012`** — weakened `resolveOpenSet` to admit any tool result once ≥1 call is open
(identity ignored), in `history.go:412-420`:

```text
--- FAIL: TestHistory_OrphanedResult_RejectedTyped (0.00s)
    history_test.go:188: got nil error, want an *ai.Violation matching
      value names something the request does not declare at "messages[1].content[0]"
```

Matches `apply-progress.md:39` verbatim in shape. Reverted; `history.go` sha256 back to
`1e90b78e…`. The fixture is correctly built with a *different* open call present, so the
bite proves identity comparison — not mere call presence — is what rejects.

**`S-HIS-031`** — tested in **two independent halves**, because the design claims two
different probes cover them:

- *Reflection half* — scratch `func (h *History) ScratchAppend(ai.Message)` writing
  `h.entries` directly:
  ```text
  --- FAIL: TestHistoryRouteGuard_SurfaceMatchesExpectedTable (0.03s)
      history_surface_guard_test.go:210: history route guard: surface route "ScratchAppend"
      (method:*History) is not in expectedHistoryRoutes
  ```
- *`go/parser` half* — scratch `func ScratchNewPrefilledHistory([]ai.Message) *History`
  filling `entries` without `commit`:
  ```text
  --- FAIL: TestHistoryRouteGuard_SurfaceMatchesExpectedTable (0.03s)
      history_surface_guard_test.go:210: history route guard: surface route
      "ScratchNewPrefilledHistory" (func) is not in expectedHistoryRoutes
  ```

Both halves genuinely close. Reverted; hash restored.

**`S-HIS-081`** — withheld the `L2C-07` entry from `expectedLayer2ContractRows` while
`doc.go` keeps its row:

```text
--- FAIL: TestLayer2DocContract_MatchesTheCommittedTable (0.00s)
    doc_contract_guard_test.go:127: doc-contract guard: found 7 of 6 rows in ".../doc.go"
    — doc.go must carry exactly one row per committed contract entry…
```

Matches `apply-progress.md:48` verbatim (`found 7 of 6 rows`); the failure message prints
the full found/want row sets, naming the unexpected `L2C-07`. Reverted; hash restored.

**Post-bite integrity**: `git status --porcelain` empty; `history.go`, `doc.go`,
`doc_contract_guard_test.go`, `history_surface_guard_test.go` all sha256-identical to the
pre-verification baseline.

### Independent defeat tests (not derived from the design's own claims)

Five probes were written as throwaway `package agent_test` files, run, then deleted.

| Probe | Question | Result |
|---|---|---|
| Deep no-alias | Mutate `Entry.Message().Content()` elements *and* `ToolCall.Arguments()` bytes, then re-read | **No leak.** `ai.Message.Content()` is `slices.Clone` (`ai/message.go:175`); `ai.ToolCall` is `struct{ id, name, arguments string }` (`ai/tool_call.go:45`) and `Arguments()` re-allocates. `S-HIS-002`'s "any slice it exposes" genuinely holds |
| Zero value, all routes | `new(History)` and `var h agent.History` through `Append`/`CloseTurn`/`SynthesizeOrphans`/`Entries`/`Len` | All three **mutating** routes reject with `history: required value is empty`. Both **read** routes return `Len()=0`, `Entries()=[]` with no error — see WARNING 2 |
| Duplicate result | Append a second result for an already-answered call | Rejected: `messages[2].content[0]: value names something the request does not declare` — the open-set semantics work as designed, with no third rule and no new `ai` class |
| Multi-part all-or-nothing | Message whose part 0 legitimately answers `c1` and part 1 is an orphan | Rejected at `messages[1].content[1]`; **`c1` is still open afterwards** — part 0's pairing effect did not half-commit. This is the strongest confirmation that `resolveOpenSet` computing into a local before assignment is correct |
| Seeded resume path | Seed ending in an open call → `SynthesizeOrphans` → `CloseTurn` | Seeded entry origin `appended`; synthesis closes 1; `CloseTurn` returns nil. The AG-12.2 resume path works end to end through the exact seam it exists to serve |

### Spec Compliance Matrix — `agent-history` (9 requirements, 25 scenarios)

| Req | Scenario | Covering test | Result |
|---|---|---|---|
| R-HIS-001 | S-HIS-001 | `history_test.go > TestHistory_OrderPreserved` | COMPLIANT |
| R-HIS-001 | S-HIS-002 | `history_test.go > TestHistory_ReadDoesNotAliasInternalStorage` | COMPLIANT (test covers only the top-level `[]Entry`; the "any slice it exposes" half was proven at runtime by the deep no-alias probe — see SUGGESTION 1) |
| R-HIS-002 | S-HIS-010 | `history_test.go > TestHistory_OrphanedResult_RejectedTyped` | COMPLIANT |
| R-HIS-002 | S-HIS-011 | `history_test.go > TestHistory_ResultAfterMatchingCall_Accepted` | COMPLIANT |
| R-HIS-002 | S-HIS-012 (bite) | Reproduced by verify — see above | COMPLIANT |
| R-HIS-003 | S-HIS-020 | `history_test.go > TestHistory_UnclosedCallAtTurnClose_RejectedTyped` | COMPLIANT |
| R-HIS-003 | S-HIS-021 | `history_test.go > TestHistory_AllCallsClosed_TurnCloseSucceeds` | COMPLIANT |
| R-HIS-003 | S-HIS-022 | `history_test.go > TestHistory_TwoUnclosedCalls_NamesFirstOffendingPosition` | COMPLIANT — asserts `messages[0].content[0].result`, i.e. the first offender |
| R-HIS-004 | S-HIS-030 | `history_surface_guard_test.go > TestHistory_NoBypass_…` (4 sub-tests, one per mutating row, each also driving `new(History)`) | COMPLIANT |
| R-HIS-004 | S-HIS-031 (bite) | Reproduced by verify, both halves | COMPLIANT |
| R-HIS-005 | S-HIS-040 | `history_test.go > TestHistory_ReadBack_UnmodifiedValuesInOrder` | COMPLIANT |
| R-HIS-005 | S-HIS-041 | `history_test.go > TestHistory_EntryIdentity_StableAcrossReadsAndSeed` | COMPLIANT — "across processes" is structural: `EntryID` is `len(h.entries)+1`, `history.go:390`, with no package-level or atomic state |
| R-HIS-005 | S-HIS-042 | `history_surface_guard_test.go > TestHistoryRouteGuard_SurfaceMatchesExpectedTable` | PARTIAL — see WARNING 3 |
| R-HIS-006 | S-HIS-050 | `history_test.go > TestNewSeededHistory_ValidSeed_Accepted` | COMPLIANT |
| R-HIS-006 | S-HIS-051 | `history_test.go > TestNewSeededHistory_OrphanedResult_RejectedFirstOffendingPosition` | COMPLIANT — result direction only, as the scenario scopes it |
| R-HIS-006 | S-HIS-052 | `history_test.go > TestNewSeededHistory_RejectedSeed_ZeroValueUnusable` | PARTIAL — see WARNING 2 |
| R-HIS-006 | S-HIS-053 | `history_test.go > TestNewSeededHistory_SignatureAcceptsOnlyMessages` | COMPLIANT — `go/ast` signature pin, not a prose claim |
| R-HIS-006 | S-HIS-054 | `history_test.go > TestNewSeededHistory_EndsInOpenCall_AcceptedThenCloseTurnRejects` | COMPLIANT — accepts the open-call seed, then `CloseTurn` rejects at `messages[1].content[0].result`, the identical shape S-HIS-020 produces |
| R-HIS-007 | S-HIS-060 | `history_synthesis_test.go > TestHistory_SynthesizeOrphans_ClosesEveryOrphanDistinguishableByOrigin` | COMPLIANT |
| R-HIS-007 | S-HIS-061 | `history_synthesis_test.go > TestHistory_SynthesizedVsReal_DistinguishableByOriginOnly` | COMPLIANT — content/`Failed()` equality is *proven* as a setup precondition (`:106-111`), then the distinguishing assertions (`:113-118`) read `Origin()` only |
| R-HIS-007 | S-HIS-062 | `history_synthesis_test.go > TestHistory_SynthesizeOrphans_TurnClosesAfterSynthesis` | COMPLIANT |
| R-HIS-008 | S-HIS-070 | `history_synthesis_test.go > TestHistory_SynthesizeOrphans_ClosesExactlyN` | COMPLIANT — N=2 interleaved with two matched calls; per-callID result counts asserted == 1 |
| R-HIS-008 | S-HIS-071 | `history_synthesis_test.go > TestHistory_SynthesizeOrphans_SecondApplicationNoOp` | COMPLIANT — compares ID, Origin and `Message().Equal` entry-for-entry |
| R-HIS-009 | S-HIS-080 | `doc_contract_guard_test.go > TestLayer2DocContract_MatchesTheCommittedTable` | COMPLIANT — table holds 7 rows, `L2C-01`..`L2C-07` in order; guard does count equality then byte-exact per-row diff against `doc.go` |
| R-HIS-009 | S-HIS-081 (bite) | Reproduced by verify | COMPLIANT |

**Compliance summary**: 25/25 scenarios have a passing covering test — 23 COMPLIANT, 2 PARTIAL,
0 UNTESTED, 0 FAILING. All 9 requirements are implemented.

The envelope reports `requirements: 9/9` and `scenarios: 25/25` because every requirement is
implemented and every scenario is discharged by a test that passed at runtime. The two PARTIAL
rows and the `R-HIS-004` residual are recorded as WARNINGs below, not as missing coverage —
they are gaps between a scenario's wording and what its (passing) covering test derives.

### Cross-cut delta compliance

| Delta | Claim | Result |
|---|---|---|
| `agent-event-envelope` `R-AEV-014` / `S-AEV-122` | The amendment makes the scenario TRUE again rather than merely mentioning it | **TRUE.** New wording asserts (a) `L2C-06` present, (b) immediately after `L2C-05`, (c) text references the four families, (d) table holds 7 rows as of AG-12. All four verified against `doc_contract_guard_test.go:61-73`: `L2C-06` is index 5 of 7, immediately after `L2C-05`, text intact. The pre-existing owner test `protocol_events_test.go:90 TestLayer2DocContract_L2C06_ReferencesProtocolFamilies` asserts presence only and never hardcoded 6, and the drift guard uses `len(expectedLayer2ContractRows)` (`:127`), not a literal — so the OLD wording would have gone false with **no test failing**. The delta is genuinely necessary, and the re-scoping ("no scenario of this requirement may be written so that such an append falsifies it") prevents recurrence |
| `agent-package-scaffold` `R-AGP-002` / `S-AGP-012`, `S-AGP-014` | Back-annotation to the seven-row baseline | **TRUE.** Both parentheticals now read "As of AG-12 … seven rows"; `S-AGP-014` cites `S-HIS-081` as the re-proof, which I reproduced. Normative text unaffected, as claimed |

### Correctness (static + runtime evidence)

| Requirement / constraint | Status | Proof |
|---|---|---|
| NFR-HIS-003 — frozen paths byte-unchanged | HELD | `git diff origin/main -- backend/agent/src/ai/ backend/agent/src/agent/loop.go backend/agent/src/agent/scheduler.go backend/agent/go.mod backend/agent/go.sum` → **empty** |
| NFR-HIS-004 — both substrate filters widened by exact filename, byte-in-sync | HELD | Extracted every `HasSuffix(path, "…")` from `filterOutLoopFiles` and `filterOutLoopHookFiles`; `diff` → **IDENTICAL**, 24 entries each, no wildcard/prefix/directory pattern. `TestTurn_SubstrateUntouched` and `TestTurn_PreRequestHook_SubstrateUntouched` (the S-LSK-006 family, `loop_test.go:1136`) both PASS |
| NFR-HIS-001 — external test package only | HELD | All three new test files declare `package agent_test` (`history_test.go:11`, `history_synthesis_test.go:8`, `history_surface_guard_test.go:11`) |
| NFR-HIS-002 — deterministic, hermetic, race-clean | HELD | No network, filesystem (beyond `go/parser` reading the package's own sources for the two structural guards), or environment access in the new tests; full module green under `-race -count=1` |
| Seeded-construction CORRECTED semantics | HELD | `NewSeededHistory` (`history.go:216-224`) runs only `commitAppend` per message; `commitCloseTurnOp` is unreachable from it. Open-call seed accepted, orphaned-result seed rejected, `CloseTurn` owns the unclosed-call rejection — confirmed by `S-HIS-051`/`S-HIS-054` and independently by the seeded-resume probe |
| Exactly one commit path (shipped surface) | HELD | `commit` (`history.go:332`) is the only function assigning `h.entries`/`h.open`; verified by inspection of the whole file and by the 4-route driver audit. See WARNING 1 for the enumeration's residual |
| Doc 0003 status counter | TRUE | `11 of 24` → `12 of 24`; doc declares exactly 24 `### AG-NN` milestone sections. Wave-2 clause "5 of 7 wave-2 milestones shipped" and the AG-07…AG-11 roster are **byte-untouched** — correct, AG-12 is wave 3 (`0003:41` lists Wave 3 = AG-12 · AG-13 · AG-14 · AG-15 · AG-16). Added prose "AG-12 … opens Wave 3 — it stands only on AG-03 and runs beside all of wave 2 — and closes R-07's boundary enforcement" matches the charter verbatim (`0003:1229`, `0003:1236-1237`). Completion checkbox at `:2169` flipped to `[x]`; no sibling checkbox touched |
| Spec promotion transform | CORRECT | `diff` of change-folder vs promoted `agent-history/spec.md` shows exactly two deltas: the promotion-note **Status** line stripped, and the doc-0003 relative link corrected from 5 hops to 3 (correct for `openspec/specs/agent-history/`). Matches the AG-11 precedent |
| `agent-history` per-node coverage table | NUMERICALLY CORRECT | Recounted from the file: AG-12.1 (`R-HIS-001`..`006`) = 16 non-bite + 2 bites; AG-12.2 (`R-HIS-007`..`008`) = 5 + 0; cross-cut (`R-HIS-009`) = 1 + 1. Table states exactly these. Totals: 9 requirements, 25 scenarios (22 + 3 bites) |

### Charter satisfaction — the six Gherkin scenarios at `0003:1247-1292`

Read from the milestone text, not the spec's restatement.

| Charter scenario | Satisfied? | Evidence |
|---|---|---|
| `0003:1252` appends keep order and orphans are rejected typed | YES | Order: `TestHistory_OrderPreserved`. Orphaned result typed: `TestHistory_OrphanedResult_RejectedTyped` (`ai.ErrUnresolvedReference` at `messages[1].content[0]`). "as does committing a state where a call has no result once the turn closes": `TestHistory_UnclosedCallAtTurnClose_RejectedTyped` (`ai.ErrEmpty` at `messages[0].content[0].result`) |
| `0003:1258` the invariant has exactly one commit path | YES for the shipped surface | Driver-map audit over every `routeMutating` row + set-equal surface diff; both bite halves reproduced. Residual in WARNING 1 |
| `0003:1263` history exposes read-only views for loop and session | YES | `Entries()` yields unmodified `ai.Message` values in order with ordinal `EntryID`; identities stable across reads and across a re-seeded history. Note: one view (`Entries()`/`Len()`) serves both consumers rather than two distinct views — see SUGGESTION 2 |
| `0003:1268` seeded construction validates like appends do | YES | Valid seed accepted with identical read-back and ordinal identities; a seed with an unmatched pair (orphaned result) rejected typed at the first offending entry. The open-call reading is the documented `design.md:214` reconciliation carried as normative prose, and `S-HIS-054` pins that `CloseTurn` still rejects |
| `0003:1281` interruption synthesizes results for orphaned calls | YES | Every orphaned call gains a matching result; origin `synthesized` vs `appended`; distinguishable from a byte-identical real `NewToolFailure` result with identical `Failed()`, by origin alone |
| `0003:1286` synthesis is idempotent and total | YES | First pass over N=2 orphans returns 2 and each callID has exactly 1 result; second pass returns 0 with an entry-for-entry identical transcript |

### Coherence (Design)

| Decision (`design.md`) | Followed? | Notes |
|---|---|---|
| Opaque struct, one `commit` primitive | Yes | `history.go:332` |
| Seed MAY end with open calls; only an unresolved result rejects at seed time | Yes | `history.go:216-224`, `S-HIS-054` |
| Duplicate result → `ErrUnresolvedReference` via open-set semantics, no third rule | Yes | Confirmed by probe: `messages[2].content[0]: value names something the request does not declare` |
| `EntryID` = 1-based `uint32` ordinal | Yes | `history.go:114`, `:390` |
| Synthesized result via `ai.NewToolFailure` + fixed unexported content, origin on the envelope | Yes | `history.go:251`, `:283-291` |
| Zero-value `History` rejected inside `commit` | Yes | `checkConstructed` at `history.go:320`, called from `commit` and from `SynthesizeOrphans`' early exit |
| Explicit `CloseTurn()`; no auto-detection | Yes | `history.go:242` |
| Closed-route guard: reflection + `go/parser`, set-equal both directions | Yes, with a residual | Both probes proven live; WARNING 1 records the shape neither probe sees |
| All four new files in both substrate filters | Yes | Plus two pre-existing files (`doc.go`, `doc_contract_guard_test.go`) added as an apply-phase correction, honestly recorded |
| `ai/validation.go` not extended, no new rule class | Yes | Frozen-path diff empty; only `ai.ErrUnresolvedReference` and `ai.ErrEmpty` used |

**Design deviations**: none. Every deviation from the plan (`apply-progress.md:77-91`) is a
*discovery* the apply phase recorded rather than absorbed, and each checks out.

### Assertion Quality

Audited all three new test files (1125 lines).

- No tautologies, no orphan empty-collection assertions, no ghost loops, no smoke tests.
- No assertion without a production call.
- Loop-based assertions (`history_test.go:129`, `:194`, `:363`, `:502`, `:535`,
  `:549`; `history_synthesis_test.go:48`, `:191`, `:232`) are all preceded by a `t.Fatalf`
  length check against a non-zero expected count, so no loop can vacuously pass.
- `history_synthesis_test.go:106-111` reads `Content()` and `Failed()` — correctly, as
  *setup-validity* `t.Fatalf`s, explicitly labelled, before the distinguishing assertions
  which read `Origin()` only. This satisfies rather than violates `R-HIS-007`'s prohibition.
- No mocks. Mock/assertion ratio: 0.

**Assertion quality**: All assertions verify real behavior. 0 CRITICAL, 0 WARNING.

### Test Layer Distribution

| Layer | Tests | Files |
|---|---|---|
| Unit | 22 top-level + 4 sub-tests | 3 (`history_test.go`, `history_synthesis_test.go`, `history_surface_guard_test.go`) |
| Structural guard | 3 of the above (`SignatureAcceptsOnlyMessages`, `SurfaceMatchesExpectedTable`, `MatchesTheCommittedTable`) | — |
| Integration / E2E | 0 | — |

Unit-only is correct here: `History` is pure in-memory with no I/O seam and wires into
nothing in AG-12.

### Issues Found

**CRITICAL**: **None (0).**

**WARNING (3)**

1. **`R-HIS-004`'s enumeration is closed over two shapes, not over every public mutating
   route.** The requirement states absolutely: *"a public mutating route that the enumeration
   does not name MUST be observable as a failure of the suite, never as an unaudited door"*
   (`spec.md:94`). I constructed one that is **not** observable. Scratch-adding to
   `history.go`:
   ```go
   type ScratchMutator struct{ H *History }
   func (m ScratchMutator) Force(message ai.Message) {
       m.H.entries = append(m.H.entries, Entry{id: EntryID(len(m.H.entries)+1), message: message, origin: EntryOriginAppended})
   }
   ```
   `TestHistoryRouteGuard_SurfaceMatchesExpectedTable` and `TestHistory_NoBypass_…` both
   **PASS**. `actualHistoryRoutes` (`history_surface_guard_test.go:165-185`) reflects only the
   method sets of `*History` and `Entry`, and `packageLevelHistoryRoutes` (`:104-134`) skips
   any `FuncDecl` with `fn.Recv != nil` (`:125`) — so a method on a *third* exported type is
   invisible to both probes, and `funcSignatureMentionsHistory` never inspects struct field
   types. Proven end-to-end from `package agent_test`: `agent.ScratchMutator{H: h}.Force(orphan)`
   committed a previously-rejected orphaned result (`Len()` 0 → 1) and `CloseTurn()` then
   returned **nil**. Reverted; tree clean.
   *Assessment*: the **shipped** surface has no such type and is fully enumerated and audited,
   so the pairing invariant is genuinely enforced today. This is a gap in the guard's future
   closure, not a live bypass. `S-HIS-031` itself is discharged. Suggested (non-blocking)
   hardening for a later milestone: extend the `go/parser` probe to also flag exported struct
   types whose field types mention `History`.

2. **The zero value behaves as an empty valid transcript through the read routes, and
   `S-HIS-052`'s test enshrines that.** `S-HIS-052` says the returned value *"used through any
   public **read** or append route … is not usable as a history — the zero value never behaves
   as an empty valid transcript"* (`spec.md:132`). Verified by probe: `new(agent.History).Len()`
   → `0` with no error; `Entries()` → `[]` with no error; same for `var h agent.History`. The
   covering test at `history_test.go:408-410` asserts `zero.Len() == 0` as the **wanted**
   outcome, i.e. it certifies the behavior the scenario reads against. The mutating routes
   (`Append`, `CloseTurn`, `SynthesizeOrphans`) all correctly reject with
   `history: required value is empty`, which is where the real risk lives, so the practical
   exposure is low. Either the implementation should make reads on an unconstructed history
   distinguishable, or `S-HIS-052` should be narrowed to the mutating routes — a spec/code
   disagreement that should be resolved rather than carried silently.

3. **`S-HIS-042` is proven partly by declaration rather than derivation.** The check is
   `if r.kind == "method:Entry" && r.class != routeReadOnly` (`history_surface_guard_test.go:220-224`)
   — it asserts the *authored* `class` field of the expectation table, not that the methods are
   actually read-only. Additionally, `reflect.TypeOf(agent.Entry{})` (`:175`) enumerates only
   the **value** method set: a scratch `func (e *Entry) ScratchSetOrigin(EntryOrigin)` is
   invisible to the guard (probe run: guard **PASSED**). The substance still holds — `Entry` is
   returned by value with unexported fields, so neither a value- nor a pointer-receiver method
   can reach committed storage — but the scenario's wording ("exposes no route to set an entry
   identity, an origin discriminator, or a committed Layer 1 value") is broader than what the
   guard derives.

**SUGGESTION (4)**

1. `S-HIS-002` says *"including any slice it exposes"*; `TestHistory_ReadDoesNotAliasInternalStorage`
   (`history_test.go:144-163`) mutates only the top-level `[]Entry`. I proved the deeper property
   holds at runtime (mutating `Content()` elements and `Arguments()` bytes leaks nothing), but no
   committed test does. A three-line extension would make the scenario self-proving.

2. `R-HIS-005` asks for read-only **views** (plural) "serving two consumers"; the implementation
   ships one view, `Entries()`, whose `Entry` carries both the Layer 1 value and the identity.
   That satisfies both consumers and matches `design.md`, but the plural in the charter
   (`0003:1263-1266`) and requirement is not literally reflected. Worth a one-line note in the
   spec rather than a code change.

3. `EntryOrigin.String()` (`history.go:97`) and `Entry.String()` (`history.go:144`) are at
   **0% coverage**. `Entry.String()` carries a stated V-FAIL-13 obligation — *"never a byte of
   its message content"* — that nothing asserts, and neither the `entry(unset)` branch nor the
   `entryorigin(N)` non-member branch is exercised. Cheap, high-value test.

4. `commitAppendOp`'s rule 1 (`message.ID().IsZero()` → `ErrEmpty` at `messages[i]`,
   `history.go:376-379`) has no covering test — no scenario appends a zero `ai.Message`. It is
   correct defensive code but currently unproven.

**Recorded context (explicitly NOT a finding, per the verify launch)**

- Diff size: `git diff origin/main --stat` = **3020 insertions + 7 deletions = 3027 changed
  lines**, 20 files. `size:exception` is recorded and pre-authorized; `apply-progress.md:6-26`
  reports the overrun prominently with a per-component breakdown rather than absorbing it.
  Note the apply record's figure (2923) predates its own `apply-progress.md` commit; the
  current number is 3027.
- `NFR-HIS-005` requires the **pull request description** to state why the change does not fit
  the default budget. No PR exists yet — this is a carry-forward obligation for delivery, not
  an apply or verify defect.
- Commit hygiene: all 11 commits are conventional (`feat|fix|docs|chore|test` + `(agent|architecture|openspec)` scope), and a case-insensitive scan for `co-authored`, `claude`, `anthropic`, `generated with`, `copilot`, `opus`, `sonnet` and the robot emoji across every subject, body, author and committer field returns **nothing**. Four subjects run 73–79 bytes; the repository has no commitlint config and merged `main` history routinely carries 87–113-character subjects, so this is within convention.

### Verdict

**PASS WITH WARNINGS** — 0 CRITICAL, 3 WARNING, 4 SUGGESTION.

All 9 requirements are implemented, all 25 scenarios have passing covering tests, all three
recorded REDs were independently reproduced and reverted byte-clean, the four frozen paths are
untouched, both substrate filters are byte-in-sync, and `make test` / `make lint` /
`make build` / `make vuln-check` are all green. The three warnings are residuals in the
*proof machinery* and one spec/code disagreement about zero-value reads; none is a live defect
in the shipped transcript store, and none blocks archive.
