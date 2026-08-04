```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:69b062192a448dae035016e3095c15e02b0b46dc63a6122f8bf5b463cd6cba3d
verdict: pass
blockers: 0
critical_findings: 0
requirements: 25/25
scenarios: 85/85
test_command: make test
test_exit_code: 0
test_output_hash: sha256:b9fd71875c0985ee4a06db2e208e0777674b19ed77b6a0e9622a32b470a7a5c9
build_command: make build
build_exit_code: 0
build_output_hash: sha256:617ff8b581dc1493b2d85d998e6171f774f2328ec65dc7264829b71cc7238495
```

# Verification Report — AI-27 streaming frame decoder

> **Change**: `cachicamas-ai-stream-decoder` · **Milestone**: AI-27 (nodes AI-27.1 … AI-27.6 + Phase 7)
> **Phase**: verify · **Mode**: Strict TDD · **Store**: hybrid · **Date**: 2026-08-03
> **Worktree**: `/Users/braejan/workspace/witsaba/repositories/cachicamas-worktrees/ai-wave-4-decoder`
> **Branch**: `feat/ai-27-6-eof-discipline` · **Base**: `feat/ai-25c-test-server-viability`
> **Verdict**: **PASS WITH WARNINGS** — 0 blockers, 0 CRITICAL, 2 WARNING, 4 SUGGESTION

---

## Verdict

**PASS WITH WARNINGS.** Every claim the six apply agents made that I could reach by command
reproduced exactly. Nothing was overclaimed. The two warnings are evidence-durability gaps, not
correctness defects: neither blocks archive, and neither can produce wrong decoder behaviour.

| Metric | Value |
|---|---|
| Blockers | **0** |
| CRITICAL | **0** |
| WARNING | **2** |
| SUGGESTION | **4** |
| Requirements verified | **25 / 25** |
| Scenarios verified | **85 / 85** (73 `[test]` + 12 `[inspection]`) |
| Test command | `make test` (`go test -race -v ./...`) from `backend/agent/` |
| Test exit code | **0** |
| Lint command | `make lint` (`go vet` + `golangci-lint v2.9.0`) |
| Lint exit code | **0** |
| Measured diff | **14 files, 4,276 insertions(+), 0 deletions(-)** vs base |

## Isolation preflight

- Base checkout `/Users/braejan/workspace/witsaba/repositories/cachicamas`: `git status --short`
  returned **empty output, exit 0** — clean.
- Verification worktree `cachicamas-worktrees/ai-wave-4-decoder`: clean, on `feat/ai-27-6-eof-discipline`.
- Sibling worktree `cachicamas-worktrees/ai-wave-4`: **never referenced** — no git command, no read, no path.
- Exactly one file written by this phase: `openspec/changes/cachicamas-ai-stream-decoder/verify-report.md`.

---

## 1. Requirement and scenario coverage — 25/25, 85/85

### Tally verified mechanically

Restricted to actual scenario-bullet lines (`^\- \*\*S-ASD-[0-9]{3}\*\*`):

```
85   scenario bullets
12   *[inspection]*
73   *[test]*
25   ^### R-ASD-[0-9]{3} headings
```

Scenario IDs `S-ASD-001…085`: complete, **no gaps, no duplicates** (diffed against `seq 1 85`).
Requirement IDs `R-ASD-001…025`: complete.

The twelve `[inspection]` IDs extracted mechanically — `010, 011, 039, 040, 051, 056, 064, 068,
069, 078, 082, 083` — match the spec's own declared list byte for byte.

**Phase 7's over-count warning independently reproduced.** An unrestricted
`grep -oE '\*\[test\]\*|\*\[inspection\]\*'` over `spec.md` returns **76 / 13**, not 73 / 12: the
Definitions section's own explanatory prose (line 60) reuses the same bold-bracket literals. The
restricted form is the correct one. The apply agent's stated remedy was accurate.

### Runtime evidence

- **74 top-level test functions** across the 7 AI-27 test files.
- **74 / 74 confirmed `--- PASS`** in the fresh `make test` output — checked by matching every
  extracted `func Test…` name against a `^--- PASS: <name> ` line. **0 missing.**
- All **73 `[test]` scenario IDs** are referenced from AI-27 test-file source; the only 6 IDs
  absent from test sources are `010, 011, 039, 040, 069, 083`, all six of which are `[inspection]`.

### Per-requirement discharge

| Req | Scenarios | Discharged by | Confirmed |
|---|---|---|---|
| R-ASD-001 | 001-003 | `decoder_test.go` single-frame, count, multi-byte | ✅ |
| R-ASD-002 | 004-006 | `decoder_test.go` order/duplicate/per-chunk | ✅ |
| R-ASD-003 | 007-011 | `TestDecoder_StartsNoGoroutines` (`runtime.NumGoroutine()` delta), two-decoder purity; 010/011 by inspection | ✅ |
| R-ASD-004 | 012-016 | `field_grammar_test.go` colon/space/colonless table | ✅ |
| R-ASD-005 | 017-020 | multi-line join + trailing-LF strip | ✅ |
| R-ASD-006 | 021-024 | three-terminator equivalence + split CRLF | ✅ |
| R-ASD-007 | 025-028 | BOM once / double BOM / interior BOM / split BOM | ✅ |
| R-ASD-008 | 029-031 | empty-accumulation suppression | ✅ |
| R-ASD-009 | 032-034, 084 | reset + `TestFieldGrammar_EmptyValuedEventLineDispatchesDefaultType` | ✅ |
| R-ASD-010 | 035-038 | id/retry equivalence incl. NUL and non-digit | ✅ |
| R-ASD-011 | 039-040 | inspection — seven grammar items = R-ASD-004…010; no eighth case found | ✅ |
| R-ASD-012 | 041-045 | exhaustive `offset := 0; offset <= len` sweep + byte-at-a-time + `sweepMismatch` | ✅ |
| R-ASD-013 | 046-048 | three dedicated CR/LF split pins | ✅ |
| R-ASD-014 | 049-051 | size guard, representative-offset stand-in, registry trace | ⚠️ W1 |
| R-ASD-015 | 052-053 | comment interleave + comment-only clean end | ✅ |
| R-ASD-016 | 054-056 | unknown field ignored, unknown event yielded, no registry | ✅ |
| R-ASD-017 | 057 | inserted event-type line in data-only dialect | ✅ |
| R-ASD-018 | 058-060 | multi-MB one-chunk / many-chunk / >64 KiB | ✅ |
| R-ASD-019 | 061-064, 085 | cap abort, pre-trip delivery, category, exact-cap | ✅ |
| R-ASD-020 | 065-067 | sentinel distinctness + shared category | ✅ |
| R-ASD-021 | 068-069 | deferral recorded; AI-19 untouched | ✅ |
| R-ASD-022 | 070-072 | clean end / empty stream / comment-only | ✅ |
| R-ASD-023 | 073-076 | four truncation pins + coverage-gap fixture | ✅ |
| R-ASD-024 | 077-078 | fed-not-finished; no `io.EOF` produced | ✅ |
| R-ASD-025 | 079-083 | sentinel verbatim, post-sentinel yield, content-independence, literal scan, `go.mod` | ✅ |

**No scenario is unconfirmed.** `S-ASD-042`, `043` and `044` are discharged by the exhaustive sweep
rather than by dedicated named cases; I verified this is mechanically sound rather than asserted —
the `bom-prefixed` fixture is literally `"\xEF\xBB\xBF" + "event: greet…"`, so offsets 1 and 2 fall
inside the three-byte mark and the loop provably visits them; splits inside the field name `event`
and inside the `café ❤️ 😀` payload are likewise offsets the loop already enumerates. See
SUGGESTION S1 on the durability of that arrangement.

### The twelve `[inspection]` scenarios, re-inspected by me

| ID | Obligation | My independent check |
|---|---|---|
| 010 | No test relies on sleep/timer/wall-clock | `grep` for `time.Sleep\|time.After\|time.NewTimer\|time.Tick\|<-time.\|Eventually\|WaitFor` over all 7 AI-27 test files → **zero matches** |
| 011 | No transport/HTTP/concurrency import in decoder source | `decoder.go` imports **only `"bytes"`**; `frame.go` imports nothing; `errors.go` imports `errors`, `fmt`, `ai` |
| 039 | Exactly seven grammar items | R-ASD-004…010 = 7, one per doc 0002 item |
| 040 | Eighth case ⇒ recorded split | No eighth case present; obligation vacuously satisfied |
| 051 | Multi-MB fixture not in sweep registry | Read `sweep_transcripts_test.go` end to end: **7 literal entries**, largest 73 bytes, no large fixture |
| 056 | No event-type registry/allowlist | `processFieldLine` is a 2-arm switch on `"event"`/`"data"`; no recognition table exists |
| 064 | No provider-failure construction | `ai.Failure`/`ai.PreStreamFailure`/`ai.MidStreamFailure` appear **only in doc-comment prose** (errors.go:49-51, 75); real code uses only `ai.FailureCategory` (type), `ai.FailureCategoryMalformedResponse` (constant), `ai.ErrMalformedResponse` (wrap) |
| 068 | Deferral names append-only, second consumer, owner | `errors.go`:26-33 names R-AIP-005 append-only, AI-30.1 item 4, and the taxonomy owner |
| 069 | AI-19 spec + source unmodified | `git diff --name-only` vs base contains **nothing** outside `openaicompat/` and the SDD change dir |
| 078 | No transport EOF produced/returned | `io.EOF`/`io.ErrUnexpectedEOF` appear only at decoder.go:26-27 in prose; **no file imports `"io"`** |
| 082 | `[DONE]` only in fixtures | 4 occurrences, all in `keepalive_test.go`; **zero** in every non-test source |
| 083 | Module manifest unchanged | `git diff … -- go.mod go.sum` is **0 bytes**; `go.mod` has **zero `require`** |

---

## 2. Charter boundary — AI-27 stayed framing-only ✅

**No transport, HTTP or concurrency import in the decoder's own sources.** Grepping
`decoder.go errors.go frame.go` for `"net/http"|"net"|"io"|"sync"|"time"|"context"|"os"|"bufio"|go func|http.|io.EOF|io.Reader|sync.|time.`
returns exactly three hits — decoder.go lines 26, 27 and 34 — all inside the type doc comment
explaining what the decoder does *not* do. Zero import, zero call site.

**No `ai.Failure` construction anywhere.** Confirmed across all 10 AI-27 files. The dependency is
strictly AI-19.2's vocabulary: `ai.FailureCategoryMalformedResponse` read as a constant and
`ai.ErrMalformedResponse` wrapped with `%w`. Neither constructor is invoked. The `outputPreceded`
fact is never referenced. Correct — a pure byte decoder cannot supply it.

**`[DONE]` sentinel appears only in fixtures.** 4 occurrences, all `keepalive_test.go`
(`const doneSentinel = "[DONE]"` plus 3 prose mentions). Zero in `decoder.go`, `errors.go`,
`frame.go`, `doc.go`, `client.go`, `credential.go`, `endpoint.go`, `request.go`.

**The AI-28.2 lock holds — frames after `[DONE]` are still yielded.** This was the check most worth
doing adversarially, and it passes on substance, not just on existence.
`TestKeepalive_FramesFollowingDoneSentinelStillYieldedNormally` feeds
`"data: [DONE]\n\ndata: alpha\n\ndata: beta\n\n"` and asserts **three** frames *and the literal
payloads* `alpha` and `beta` — not merely a count. A "suppress everything after `[DONE]`"
implementation fails it on content, and a count-only fixture would not have caught a
suppress-but-keep-count regression. Terminal-sentinel recognition was **not** implemented a slice
early. `TestKeepalive_DoneSentinelYieldedVerbatimNoSpecialHandling` additionally asserts `Finish()`
returns `nil` after the sentinel, proving no early completion.

**AI-19's taxonomy source and spec are untouched — proven from the diff.** All 14 changed paths are
status `A` (added); **zero** files are `M`. Nothing outside `backend/agent/src/ai/openaicompat/`
and `openspec/changes/cachicamas-ai-stream-decoder/` appears in the diff at all, so
`src/ai/provider_failure.go` and the `ai-provider-errors` spec are untouched by construction.

**`go.mod` unchanged, zero requires.** Diff is 0 bytes. Full file content is `module …` + `go 1.26.3`
— no `require` block exists.

---

## 3. The two error sentinels ✅

`TestBoundedMemory_ErrorIdentitiesStayDistinctBothReportMalformedResponse` — **PASS at runtime** —
asserts all six required facts directly on the package-level sentinels:

- `errors.Is(ErrFrameTooLarge, ErrTruncated)` → must be false ✓
- `errors.Is(ErrTruncated, ErrFrameTooLarge)` → must be false ✓
- each matches its own identity ✓
- `errors.Is(ErrFrameTooLarge, ai.ErrMalformedResponse)` → must be true ✓
- `errors.Is(ErrTruncated, ai.ErrMalformedResponse)` → must be true ✓
- `Category(...)` returns `ai.FailureCategoryMalformedResponse` for both ✓

Confirmed structurally in `errors.go`: both are built with
`fmt.Errorf("…: %w", ai.ErrMalformedResponse)` from **two separate calls**, which is exactly what
makes them distinct under `errors.Is` while both unwrapping to AI-19's sentinel. The test also
carries negative cases (`Category(nil)` and `Category(errors.New("unrelated"))` must report
`ok == false`), which most implementations omit.

`TestEOF_TruncationErrorCategoryIsMalformedResponseDistinguishableFromCapExceeded` proves the same
facts about the error `Finish()` **actually returns** on a live truncation, not just the bare
sentinel — a genuinely stronger claim.

**Tenth-category deferral is recorded, not acted on.** `failureCategoryNames` in
`src/ai/provider_failure.go` holds exactly **nine** members (authentication, authorization,
rate_limit, unavailable, timeout, cancellation, malformed_response, unsupported_capability,
unknown). No resource-exhaustion member was appended, and the file is not in the diff.

---

## 4. The cap boundary ✅ — the assertions genuinely falsify the opposite implementation

`TestBoundedMemory_ExactlyAtCapDecodesOneByteOverAborts` — **PASS**. I re-derived the arithmetic
rather than trusting the claim:

- `payloadLen = 2048`; `atCap = payloadLen + 1 = 2049` — the **exact peak** accumulated size, since
  `processFieldLine` appends one `'\n'` separator beyond the value.
- `at_cap_decodes` runs with cap `2049` and requires `err == nil`, 1 frame, byte-exact payload.
  Under a `>=` implementation, `2049 >= 2049` is **true**, so the decoder would abort and this
  subtest **would fail**. The assertion is genuinely load-bearing, not decorative.
- `one_under_peak_aborts` runs with cap `2048` and requires `ErrFrameTooLarge`, **0** frames,
  non-match against `ErrTruncated`, match against `ai.ErrMalformedResponse`, and the right category.

`exceedsCap` in decoder.go:436-438 uses `> d.maxFrameBytes` — strictly greater, matching the spec.

**Pre-trip frames are returned together with the error.**
`TestBoundedMemory_CompleteFrameBeforeOverCapFrameReturnedTogetherWithError` — **PASS** — feeds one
chunk holding a complete frame plus a 200-byte over-cap frame against a 64-byte cap, then asserts
`errors.Is(err, ErrFrameTooLarge)` **and** `len(frames) == 1` **and** the returned frame equals
`{Event: "greet", Data: "hello"}`. A drop-on-error implementation fails on the count; a
wrong-content implementation fails on the equality. Both halves are pinned.

The production side matches: `Feed` returns `frames, d.err` (decoder.go:139 and :154), never
`nil, d.err`. decoder.go:129-136 documents precisely why the retained-tail cap check must not run at
an arbitrary cursor — doing so would count a later frame's bytes against the current one and abort
before the complete frame dispatches. That reasoning is correct and the code implements it.

---

## 5. The offset sweep's exclusion ✅ (one durability gap — see W1)

- **Registry**: `sweepTranscripts` in `sweep_transcripts_test.go` holds **7** entries — `lf-only`,
  `crlf`, `lone-cr`, `mixed-terminators`, `bom-prefixed`, `multi-line-data`, `comment-interleaved`
  — every one a small inline literal. The multi-megabyte fixture is **not among them**. Verified by
  reading the whole file, not by grep alone.
- **Inclusive bound**: `const maxSweepFixtureBytes = 1024`, and the guard is
  `if len(tr.transcript) > maxSweepFixtureBytes` → `t.Errorf`. That is `> 1024` fails, i.e. an
  **inclusive `<= 1024`** admission bound, exactly as R-ASD-014 words it. Exactly 1024 is
  admissible; 1025 is not.
- **Both non-exhaustive stand-ins exist and pass**:
  `TestBoundedMemory_MultiMegabyteFixtureAtRepresentativeOffsetsMatchesCanonical` (S-ASD-050,
  representative offsets, 0.44s) and
  `TestBoundedMemory_MultiMegabyteFrameFedInManySmallChunksMatchesSingleChunk` (S-ASD-059, many
  small chunks, 0.83s). Both reuse `decodeCanonical`/`decodeSplitAtOffset`/`sweepMismatch` verbatim
  from the sweep harness rather than reimplementing them.

---

## 6. Evidence-class honesty ✅ — the strongest part of this milestone

**Slices 3 and 4 landed with genuinely zero production changes.** Confirmed mechanically, not from
the report: `git diff --stat` of slice 2b→3 and slice 3→4 restricted to
`decoder.go errors.go frame.go` is **empty in both cases**. Slice 3 touched only
`offset_sweep_test.go`, `sweep_transcripts_test.go`, `tasks.md`; slice 4 only `keepalive_test.go`,
`sweep_transcripts_test.go`, `tasks.md`. The disclosure was accurate.

### Five disclosed non-red claims, each re-derived from the code

The claim under audit: against slice 5's `Finish()` stub
(`if d.err != nil { return d.err }; return nil`), 5 of 12 `TestEOF_*` functions were already green
and were disclosed as such rather than dressed up as red. I traced each through decoder.go:

| Test | Claim | My derivation | Honest? |
|---|---|---|---|
| `…CleanEndAtFrameBoundaryFinishesWithoutError` (S070) | non-red | expects `nil`; stub returns `nil` | ✅ |
| `…EmptyStreamFinishesCleanWithZeroFrames` (S071) | non-red | expects `nil`; stub returns `nil` | ✅ |
| `…CommentOnlyStreamEndingAtLineBoundaryFinishesClean` (S072) | non-red | expects `nil`; stub returns `nil` | ✅ |
| `…FramesFedButNotFinishedExposeOnlyDispatchedFramesNoError` (S077) | non-red, never calls `Finish` | read the body — it genuinely never calls `Finish()`, so it is independent of the stub | ✅ |
| `…TrailingBufferFinalCRWithNothingPendingFinishesClean` | non-red, own falsifiability pin | expects `nil`; stub returns `nil` | ✅ |

All five would pass against a `return nil` stub. The 7 disclosed-red tests all assert
`errors.Is(err, ErrTruncated)`, which a `nil`-returning stub cannot satisfy. **The split is exactly
as reported.** No rationalisation found.

### The break-and-revert claims, re-derived rather than accepted

**Break 1 (CR resolution skipped) is the one worth checking hardest**, and the claim is precisely
correct. Tracing `"data: only\n\n\r"`: `Feed` resolves both LF-terminated lines, dispatches the
frame, then `nextLine` hits the final `\r` and `resolveCR` returns `ok == false` (no following
byte), so `"\r"` stays in `d.buf`. With `pendingFinalCR`, content is empty, `d.buf` is truncated,
all three checks read zero → `nil`. Without it, `len(d.buf) > 0` → wrong `ErrTruncated`. So
`TestEOF_TrailingBufferFinalCRWithNothingPendingFinishesClean` **is** the discriminating fixture.
And its companion `…ResolvesToTruncationWhenContentPending` (`"data: a\r"`) really does report
`ErrTruncated` **either way** — CR-aware (content `"data: a"` folds into `d.data`) or naive
(`d.buf` non-empty) — so it genuinely cannot distinguish the two. The apply agent's surgical claim
is accurate, and the CR-resolution branch is not dead code coincidentally agreeing with a shortcut.

**Break 4 (`len(d.eventType) > 0` arm dropped)** also re-derives correctly: fixture
`"event: custom\n"` leaves `d.data` empty and `d.eventType == "custom"`, so dropping that arm makes
`Finish` return `nil` and falsifies exactly `TestEOF_PendingEventTypeWithNoDataLineReportsTruncation`
— which the apply agent added specifically because it found that none of R-ASD-023's four numbered
scenarios isolates this state. That is a real coverage gap, found and closed **before** GREEN, and
the fixture is genuinely load-bearing.

**Break 3 (folding skipped)** re-derives correctly too: with `"data: a\r"`, skipping
`processFieldLine` while still clearing `d.buf` makes all three checks read zero → wrong `nil`.

### Guard bite proof and leftovers

The sweep guard's bite proof **was** genuinely staged and removed. `tasks.md:198` records the
verbatim failure line — `offset_sweep_test.go:32: transcript "OVERWEIGHT-STAGED-FOR-BITE-PROOF" is
1025 bytes, want <= 1024 (S-ASD-049)`, exit 1 — and I confirmed by grep that no overweight fixture,
no `1025` literal and no leftover `"bytes"` import remains in either sweep file. Removal is clean.
See **W1** for the durability consequence.

**Zero `t.Skip`, `TODO`, `FIXME`, `XXX` or `HACK` in any of the 10 AI-27 files.** The 8 `--- SKIP`
lines in the suite are all pre-existing AI-24 conformance subtests for declared-absent optional
capabilities (`src/ai`, `src/agenttest`); none is in `openaicompat`.

### Assertion quality audit

| File | Tests | Assertion calls |
|---|---|---|
| `decoder_test.go` | 9 | 30 |
| `field_grammar_test.go` | 28 | 95 |
| `offset_sweep_test.go` | 7 | 26 |
| `keepalive_test.go` | 8 | 35 |
| `bounded_memory_test.go` | 10 | 41 |
| `eof_test.go` | 12 | 42 |
| **Total** | **74** | **269** (~3.6 per test) |

**Assertion quality: ✅ All assertions verify real behavior.** Zero tautologies. Zero mocks (the one
`stub` grep hit is prose). Zero ghost loops. Zero smoke-tests. No orphan empty-collection assertion:
every `len(frames) != 0` check sits beside a companion asserting non-empty content. Several tests
carry explicit anti-vacuity guards — `TestOffsetSweep_CanonicalDecodeMatchesHardCodedFrames` fails
outright if no registry entry sets `want`, and
`TestKeepalive_FramesDifferingOnlyInPayloadContentDecodeIdentically` fails with "fixture bug" if its
two payloads are ever made equal. That is a notably high standard.

---

## 7. Regression and hygiene ✅

```
$ go clean -testcache && make test        # from backend/agent/
go test -race -v ./...
ok  	github.com/cachicamas/backend/agent/src/agenttest	2.184s
ok  	github.com/cachicamas/backend/agent/src/ai	3.221s
ok  	github.com/cachicamas/backend/agent/src/ai/openaicompat	3.347s
MAKE_TEST_EXIT=0
```

- **546** `^--- PASS:` top-level — **matches the claimed 546 exactly**
- **0** `--- FAIL:` at any nesting level
- **0** `DATA RACE`, **0** `WARNING:`
- 8 `--- SKIP:` — all pre-existing AI-24 conformance subtests, none in `openaicompat`
- `test_output_hash`: `sha256:b9fd71875c0985ee4a06db2e208e0777674b19ed77b6a0e9622a32b470a7a5c9` (701,589 bytes)

```
$ make lint
go vet ./...
bin/golangci-lint run --config=.golangci.yml ./...
0 issues.
MAKE_LINT_EXIT=0
```

**Guards — all pass, no allowlist edit:**

| Guard | Result |
|---|---|
| `TestLayer1_ImportsOnlyStdlibAndItsOwnPackages_DenyByDefault` | PASS |
| `TestLayer1_ModuleHasNoDependencies_ZeroRequires` | PASS |
| `TestAmbientAuthority_NoForbiddenCallSitesInAdapterSources` | PASS |
| `TestAmbientAuthority_ForbiddenSetIsPackageScopedDenyByDefault` | PASS |
| `TestAmbientAuthority_IsAdapterSourceFile` | PASS |
| `TestAmbientAuthority_TestSourcesStayGreenEvenWithForbiddenCalls` | PASS |

**AI-27's new non-test sources are genuinely scanned.** `isAdapterSourceFile` is
`strings.HasSuffix(name, ".go") && !strings.HasSuffix(name, "_test.go")` — a uniform suffix rule
applied to every entry in the package directory. `decoder.go`, `errors.go` and `frame.go` are
admitted automatically. The guard has **no allowlist mechanism at all**, so "no allowlist edit" holds
in the strongest possible sense; `ambient_authority_test.go` is also absent from the diff.

### TDD Compliance

| Check | Result | Details |
|---|---|---|
| TDD Evidence reported | ✅ | Cycle tables present for all six slices in apply-progress and tasks.md |
| All tasks have tests | ✅ | 56/56 tasks; 74 test functions |
| RED confirmed (test files exist) | ✅ | 7/7 AI-27 test files present |
| GREEN confirmed (tests pass) | ✅ | 74/74 pass on fresh execution |
| Triangulation adequate | ✅ | Table-driven throughout; `field_grammar_test.go` alone carries 28 functions / 95 assertions |
| Safety net for modified files | ✅ | Slices 3-4 modified nothing; slices 5-6 ran the prior full suite first (534 baseline → 546) |
| Non-red disclosure honesty | ✅ | 5/5 spot-checked disclosures re-derived and confirmed truthful |

### Test Layer Distribution

| Layer | Tests | Files | Tools |
|---|---|---|---|
| Unit | 74 | 7 | `go test -race` |
| Integration | 0 | 0 | not required — the decoder is a pure byte function |
| E2E | 0 | 0 | not applicable at Layer 1 |

Test imports are stdlib-minimal (`bytes`, `errors`, `fmt`, `strings`, `runtime`, `testing`) plus the
package under test and `src/ai`. No `net/http`, `sync`, `io`, `os` or `time` anywhere — which is
itself corroborating evidence for S-ASD-007 and S-ASD-010.

**Coverage**: not run. `make test/cover` exists, but coverage is informational under this skill and
never blocking; the 74/74 runtime pass plus the per-scenario mapping is the operative evidence.

---

## 8. Diff size — every slice within the 5,000-line budget ✅

Re-measured independently with `git diff --shortstat` at each boundary:

| Slice | vs | Files | Insertions | Deletions | Total | Budget |
|---|---|---|---|---|---|---|
| ai-27-1 skeleton | ai-25c base | 7 | 1,373 | 0 | 1,373 | ✅ |
| ai-27-2a field grammar | 27-1 | 3 | 444 | 14 | 458 | ✅ |
| ai-27-2b dispositions | 27-2a | 3 | 521 | 20 | 541 | ✅ |
| ai-27-3 offset sweep | 27-2b | 3 | 480 | 9 | 489 | ✅ |
| ai-27-4 keepalives | 27-3 | 3 | 351 | 11 | 362 | ✅ |
| ai-27-5 bounded memory | 27-4 | 4 | 692 | 28 | 720 | ✅ |
| ai-27-6 EOF discipline | 27-5 | 5 | 537 | 40 | 577 | ✅ |
| **Whole chain** | **ai-25c base** | **14** | **4,276** | **0** | **4,276** | ✅ |

15 commits, matching the claim. Largest single slice is 1,373 lines — well inside the session's
`review_budget_lines: 5000`. Every slice targets its immediate predecessor (feature-branch chain),
so no child diff carries a previous slice's content.

---

## 9. Task completion ✅

`tasks.md`: **56 `- [x]`, 0 `- [ ]`**. No task is unchecked. The Coverage line (tasks.md:45) reads
25/25 requirements and 85/85 scenarios at 73 `[test]` / 12 `[inspection]` — consistent with the
mechanically verified spec tally.

**Phase 7.2 verified.** `grep -n "AI-31" proposal.md design.md specs/ai-stream-decoder/spec.md`
finds exactly **one** match: `proposal.md:235`, inside the correction blockquote's own
"previously named **AI-31**" historical note. Not a propagation. `proposal.md` carries two AI-30.1
citations. The correction was genuinely applied, not merely re-flagged.

---

## Issues

### CRITICAL — none

No blocker. Nothing prevents archive.

### WARNING

**W1 — `S-ASD-049`'s second clause has no permanent runnable proof.**
The scenario is marked `[test]` and reads "…every member satisfies size ≤ 1024 bytes … **and a
deliberately overweight fixture trips the guard**." The shipped
`TestOffsetSweep_FixturesWithinSweepBound` only iterates the real, clean registry; it proves the
positive half. The negative half — that the guard *bites* — was demonstrated by a scratch fixture
that was staged, observed failing, and then removed. That proof is real (I confirmed the exact
recorded failure line and confirmed nothing was left behind), but it is **transient**: nothing in
the suite can re-establish it, so a future refactor that silently defanged the comparison would go
unnoticed. This repository already has the durable pattern for exactly this — AI-24 ships five
`TestSequenceGuard_*_Fails` negative tests that construct a violating input in-test and assert the
guard rejects it. Adopting that shape here would make the clause permanently failable.
*Not a blocker: the guard's positive half is tested, the exclusion currently holds, and the
milestone's correctness does not depend on it.*

**W2 — Stale slice-4 comment now contradicts shipped behaviour.**
`keepalive_test.go:86-91` states: "Finish()'s 'no error' is stub-true today — AI-27.6/slice 6 has
not yet implemented truncation detection, so this assertion cannot yet fail." Slice 6 **did**
implement it (`decoder.go:282-299`), so the assertion is now genuinely live and the comment is
false. The comment itself predicted this ("it becomes a live regression guard the moment slice 6's
Finish() implementation starts inspecting d.buf/d.data/d.eventType"), but slice 6 did not return to
update it. A reader of the shipped source is misinformed about the milestone's own state.
*Not a blocker: the test is correct and passing; only the prose is stale.*

### SUGGESTION

**S1 — Sweep-derived scenario coverage is not pinned to the registry's shape.**
`S-ASD-042`, `S-ASD-043` and `S-ASD-044` are discharged entirely by the exhaustive sweep visiting
offsets that happen to fall inside a field name, a multi-byte rune, and the BOM. That is sound
today because the registry contains `bom-prefixed` and `multi-line-data`. But no assertion requires
those entries to remain. Removing either would silently void a scenario's coverage while every test
stayed green — the same vacuous-pass class this milestone otherwise defends against carefully
(`TestOffsetSweep_CanonicalDecodeMatchesHardCodedFrames` already guards the `want` invariant this
way). A small assertion that the registry still contains a BOM-prefixed and a multi-byte entry
would close it.

**S2 — File-level import purity is not package-level import purity.**
`S-ASD-011` says "the shipped decoder source", and at file level it holds cleanly: `decoder.go`
imports only `"bytes"`. But the decoder ships inside the **same Go package** as AI-25's HTTP client,
and `openaicompat` as a package does import `net/http` through `client.go` and `request.go`. Worth
recording explicitly so a later reader — or a later milestone tempted to reach for a transport
helper "already in the package" — does not mistake file-level purity for a package-level boundary.
The charter is satisfied as written; the ambient temptation is real.

**S3 — `tasks.md:208` says "all 6 registered transcripts"; there are now 7.**
Correct when slice 3 wrote it; slice 4 added `comment-interleaved` without updating the slice-3
evidence line. Harmless, but the evidence log is the archive artifact.

**S4 — Consider promoting `S-ASD-051` to `[test]` at archive.**
The "multi-megabyte fixture is not enumerated by the sweep" obligation is currently reviewer-traced,
yet it is now mechanically checkable in one line against `sweepTranscripts` — the same upgrade
condition logic Phase 7 applied to `S-ASD-045`. Optional, and properly an AI-28-era decision.

---

## Design coherence

| Design decision | Implementation | Status |
|---|---|---|
| Poisoning: first terminal error is sticky | `d.err` checked at the head of `Feed` and `Finish` | ✅ |
| Deferred CR held structurally, not via a flag | CR left in `d.buf`; no `pendingCR` field exists | ✅ |
| Offset-sweep harness: canonical + per-offset + byte-at-a-time | all three present | ✅ |
| Finish resolves a pending buffer-final CR first | `pendingFinalCR` called before the truncation check | ✅ |
| Single shared cap counter | `exceedsCap` is the one counter for both call sites | ✅ |

**One recorded deviation** (apply-progress, slice 6): task 6.6's refactor shares CR-resolution logic
*conceptually and by cross-reference* between `pendingFinalCR` and `resolveCR` rather than by literal
function reuse. I confirmed the reasoning is sound rather than a shortcut — `resolveCR` answers
"given a following byte, lone CR or CRLF?" while `pendingFinalCR` answers "no byte is ever coming,
what does this resolve to?". Those are different questions and `resolveCR`'s signature cannot express
the second without changing `Feed`'s well-tested behaviour. Both doc comments cross-reference each
other. **WARNING-level per the decision table, but it breaks no spec** — recorded here as an accepted
deviation rather than an issue.

---

## Conclusion

**PASS WITH WARNINGS.** Not a single apply-agent claim failed independent re-derivation. The 546
passes, the 73/12 tally, the 14-file / 4,276-insertion diff, the zero-production-change slices, the
`errors.Is` distinctness, the exactly-at-cap boundary, the pre-trip frame delivery, the inclusive
1024 sweep bound, the untouched taxonomy and the untouched `go.mod` all reproduced exactly.

The evidence-class discipline is the most credible part of this milestone: the disclosed non-red
tests were genuinely already green, the break-and-revert claims re-derive correctly from the source,
and the one break claimed to be surgically precise — CR resolution — really is. That is unusual and
worth preserving as a pattern.

Both warnings concern the **durability of evidence**, not the correctness of code. Neither blocks
archive.

**Recommended next phase**: `sdd-archive`.
