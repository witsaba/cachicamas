# Tasks — cache-boundary markers

> **Change**: `cachicamas-ai-cache-breakpoints`
> **Milestone**: AI-11 · **Nodes**: AI-11.1, AI-11.2, AI-11.3, all `[leaf]`
> **Phase**: tasks
> **Project**: cachicamas (witsaba)
> **Date**: 2026-08-01
> **Worktree**: `cachicamas-worktrees/ai-11` · **Branch**: `feat/ai-11-cache-breakpoints` · **Base**: `07d2027`
> **Inputs**: `explore.md`, `proposal.md`, `specs/ai-cache-breakpoints/spec.md`, `design.md`
> **Depends on**: AI-10 · **Blocks**: AI-24, AI-26.2
> **Evidence gate**: recorded green `make test` **and** clean `make lint` in `backend/agent/` (`make test` is `go test -race -v ./...`)

---

## Entry point for the resuming agent

**Nothing in this milestone is implemented.** Every box below is unchecked. All five planning artifacts are written and committed.

Start here, in this order:

1. **Run `explore.md` § 8 — the re-verification register.** Eight rows, each naming a file and a symbol from AI-10 that this plan assumes. AI-10.3 items 2–4, AI-10.4, AI-10.5 and AI-10.6 were landing concurrently in `../ai-wave-1` while this plan was written, so those rows are promises, not observations. **Do this before writing the first test.** A row that failed is a `revert-and-record` event under doc 0002's living-graph clause: record the discovered prerequisite in `explore.md` § 8 and in `design.md` § 10, then proceed against what is actually in the tree.
2. **Rebase or merge the completed AI-10** into this branch, and confirm `make test` and `make lint` are green **before** adding anything. A red baseline makes every later RED observation ambiguous.
3. Start at **§ Phase AI-11.1 item 1**, the walking skeleton.

Every decision this milestone needs is already made in `design.md` §§ 2–6. A decision may be changed only by recording the reason in `design.md` **before** writing the test; absent a recorded reason, implement what is written.

**This milestone appends no rule class.** `ErrOutOfRange` already cites `V-REQ-24` in its own GoDoc (`validation.go:81`), so `ruleClasses` and both of its registry mirrors stay untouched — unlike AI-08.2 and AI-10.3. If you find yourself editing `validation.go`'s sentinel block, stop and re-read `explore.md` § 3.3.

**Never edit `openspec/specs/ai-contract-vocabulary/spec.md`.** `V-REQ-23`, `V-REQ-24` and `V-REQ-25` are already written there and are this milestone's binding input. If a new term turns out to be needed, report it upward; the orchestrator appends it once.

## Node types and what they close on

| Node | Type | Closes on |
| --- | --- | --- |
| AI-11.1, AI-11.2, AI-11.3 | `[leaf]` | Every test-list item taken red → green → refactored, **in order**, with both outputs recorded here |

Strict TDD is on (`openspec/config.yaml`, `apply.tdd: true`). Go's compile-time typing means a red step needs the declaration to exist, so `design.md` § 7 applies: land the narrowest thing that **compiles and fails**, record the failure, then implement. **A compile error is the state before red, not red.**

Discovered cases are **appended** to the owning leaf's test list, never substituted for a planned one and never pruned to fit the budget. The items marked *(appended)* below were discovered during planning and are already part of the contract.

**Commit per leaf.** Wave 1 recorded that per-leaf commits are what made a mid-milestone agent death cheap; a single end-of-milestone commit loses everything.

---

## Review Workload Forecast

| Field | Value |
| --- | --- |
| Estimated changed lines | ~1 000 Go (~330 production, ~670 test) plus ~1 100 prose in five SDD artifacts |
| 400-line budget risk | High |
| Chained PRs recommended | No |
| Suggested split | Single PR — the wave PR, with the leaf boundary as the commit boundary |
| Delivery strategy | exception-ok |
| Chain strategy | size-exception |

```text
Decision needed before apply: No
Chained PRs recommended: No
Chain strategy: size-exception
400-line budget risk: High
```

### Budget reassessment — split trigger 4 fires, and was forecast before any code

doc 0002's rule is *"prefer less than 250 changed lines; stop and reassess before 400"*, and split trigger 4 fires when a milestone's projected diff, tests included, pushes it past the review budget. **It does.** `explore.md` § 7 recorded it before a line of Go was written. The reassessment:

- **Split trigger 1 does not fire.** Three leaves, one publicly observable behaviour family: a marker's placement, its cap, its advisory status. Splitting them across PRs would ship a marker nobody can count and a cap over nothing.
- **The mitigation is the commit boundary, not a smaller test list.** doc 0002 puts the PR-chain boundary at the node boundary; three leaf commits make the chain reviewable in slices without rework.
- **Production is small; documentation and tests carry the weight.** The exported surface is one constant, one three-member vocabulary, one value type, three method pairs, and one accessor. What is large is the contract documentation, which is where this package puts its reasoning, and the tests.
- **The wave-level PR budget is 5 000+ lines and was accepted up front as `exception-ok`.** A forecast over budget is stated, not obeyed.
- **Nothing is cut to fit.**

### Suggested Work Units

| Unit | Goal | Likely PR | Focused test command | Runtime harness | Rollback boundary |
| --- | --- | --- | --- | --- | --- |
| 1 — AI-11.1 | A marker can be placed on and read from all three carriers, and it changes nothing else | wave PR, commit 1 | `go test -race -run 'TestSegment_Cache\|TestTool_Cache\|TestMessage_Cache\|TestRequest_Marker' ./src/ai/` | N/A — a value-type contract with no runnable process; the cross-package harness is unit 3 | `git revert` commit 1: removes one field and two methods from each of three files |
| 2 — AI-11.2 | The cap is enforced and the cascade order is observable | wave PR, commit 2 | `go test -race -run 'TestRequest_CacheBoundar\|TestCacheRegion' ./src/ai/` | N/A — same reason | `git revert` commit 2: removes `cache_boundary.go` and one rule row |
| 3 — AI-11.3 | The advisory contract holds, with a control proving the assertion is not vacuous | wave PR, commit 3 | `go test -race ./src/agenttest/` | `make test` in `backend/agent/` — the whole-module run is the harness that proves the guards and both AI-00 import boundaries still hold | `git revert` commit 3: removes one test file; no production code |

---

## Phase AI-11.1 — markers on segments, tools and messages `[leaf]`

**Deliverable:** `system_instruction.go`, `tool.go`, `message.go`, `cache_boundary_test.go`.
**Spec:** `R-ACB-001` … `R-ACB-005`, `R-ACB-010`. **Design:** §§ 2, 3.1, 3.4.
**Depends on:** AI-10.

Test list — items 1–3 are doc 0002's, verbatim; 4–5 were appended during planning.

- [ ] **Item 1** — A system segment, a tool declaration and a message can each carry a cache-boundary marker; marker placement round-trips through construction and readback.
  - [ ] 1.1 RED: `TestSegment_MarkCacheBoundary_RoundTripsThroughConstruction` in `ai_test` against a `MarkCacheBoundary` that returns the receiver unchanged. Assert marked, and text byte-equal. `S-ACB-001`.
  - [ ] 1.2 GREEN: add `cacheBoundary bool` to `Segment`; implement `MarkCacheBoundary` and `IsCacheBoundary` (`design.md` § 3.1).
  - [ ] 1.3 RED → GREEN: the same pair on `Tool` — name, description and schema bytes unchanged. `S-ACB-002`.
  - [ ] 1.4 RED → GREEN: the same pair on `Message` — role, ordered content **and identity** unchanged. `S-ACB-003`.
  - [ ] 1.5 RED → GREEN: placement returns a **copy** — the value it was applied to still reports unmarked, on all three carriers. `S-ACB-004`.
  - [ ] 1.6 RED → GREEN: a never-marked carrier reports not-a-boundary; marking twice is idempotent. `S-ACB-005`, `S-ACB-006`. `Tool` and `Message` are non-comparable, so compare field-wise (`design.md` § 3.1).
  - [ ] 1.7 Refactor: the three method pairs are three spellings of one idea — check they read identically and that each GoDoc states its own zero-value rule.

- [ ] **Item 2** — Markers do not participate in validity: a marked request and its unmarked twin validate identically and are otherwise equal.
  - [ ] 2.1 RED → GREEN: a valid request and its marked twin (within the cap) both construct successfully. `S-ACB-016`.
  - [ ] 2.2 RED → GREEN: an **invalid** request and its marked twin fail with the same rule class at the same rendered position. `S-ACB-017`.
  - [ ] 2.3 RED → GREEN: every placement combination across the three carriers, within the cap, constructs — there is no placement rule. `S-ACB-018`.
  - [ ] 2.4 RED → GREEN: a marked segment of one character constructs — minimum cacheable prefix length is not decidable here. `S-ACB-019`.
  - [ ] 2.5 RED → GREEN: two requests identical but for one marker are **not** equal; two with identical markers **are**. `S-ACB-013`, `S-ACB-014`. Uses AI-10.6's `Equal` — see `explore.md` § 8 rows 4–5. If `Equal` was not exported, compare region-wise in a local helper and record it here.
  - [ ] 2.6 RED → GREEN: a marked request read region by region and rebuilt is equal to the original — no marker lost in the round trip. `S-ACB-015`.

- [ ] **Item 3** — A marker is readable from an external package wherever it can be set — the adapter is the only consumer that will ever care.
  - [ ] 3.1 RED → GREEN: an `ai_test` request built from marked segments, marked declarations and marked messages observes every marker at the ordinal it was placed on, through the **existing** region accessors only — no new accessor on `SystemInstruction` or `ToolSet`. `S-ACB-011`.
  - [ ] 3.2 RED → GREEN: reading each region twice observes identical markers, because each read returns a fresh copy. `S-ACB-012`.

- [ ] **Item 4** *(appended)* — A marker cannot make an unconstructed value usable.
  - [ ] 4.1 RED → GREEN: `MarkCacheBoundary` on the zero `Segment`, zero `Tool` and zero `Message` returns the receiver; each is still rejected by its enclosing constructor with `ErrEmpty` at its index. `S-ACB-007` … `S-ACB-009`.
  - [ ] 4.2 RED → GREEN: none of the three marked zero values reports itself a cache boundary. `S-ACB-010`.

- [ ] **Item 5** *(appended)* — Marker state renders as structure and never as payload, and the unmarked rendering is unchanged.
  - [ ] 5.1 RED → GREEN: a marked segment holding a secret renders through the default, string, extended and Go-syntax verbs naming the boundary and reproducing no secret. `S-ACB-038`.
  - [ ] 5.2 RED → GREEN: an unmarked segment's rendering is byte-identical to before — `system_instruction_test.go:288` asserts `"segment"` exactly and must stay green. `S-ACB-039`.

- [ ] **AI-11.1 close:** record green `make test` and clean `make lint`; commit `feat(ai): place cache-boundary markers on segments, tools and messages (AI-11.1)`.

---

## Phase AI-11.2 — cap and ordering invariants `[leaf]`

**Deliverable:** `cache_boundary.go` (new), `request.go`, `cache_boundary_test.go`.
**Spec:** `R-ACB-006`, `R-ACB-007`, `R-ACB-010`. **Design:** §§ 2, 3.2, 3.3, 3.4, 4.
**Depends on:** AI-11.1.

Test list — items 1–2 are doc 0002's, verbatim; item 3 was appended during planning.

- [ ] **Item 1** — WHEN a request's total marker count exceeds the documented cap THEN validation fails before I/O with an AI-04 sentinel naming the excess — the vendor cap is small and hard, and catching it client-side is the point of the seam.
  - [ ] 1.1 RED: `TestRequest_CacheBoundariesOverCap_FailsWithOutOfRange` against a request one marker above `MaxCacheBoundaries`; assert `errors.Is(err, ErrOutOfRange)` and the position renders `cacheBoundaries`. `S-ACB-020`.
  - [ ] 1.2 GREEN: create `cache_boundary.go` with `MaxCacheBoundaries = 4`, the shared walk `requestDraft.cacheBoundaries`, and `cacheBoundaryCapRule`; insert it as rule 10 of `NewRequest`'s slice (`design.md` § 3.3). Verify the landed rule-order table against the tree first — `explore.md` § 8 row 3.
  - [ ] 1.3 RED → GREEN: exactly the cap is valid; no markers at all is valid and reports an empty sequence. `S-ACB-021`, `S-ACB-022`.
  - [ ] 1.4 RED → GREEN: the cap is a **total**, not a per-region budget — exceeding it with markers drawn from all three regions fails identically. `S-ACB-023`.
  - [ ] 1.5 RED → GREEN: a request that both breaks an earlier structural rule and exceeds the cap reports the **earlier** rule, per `V-FAIL-04`. `S-ACB-024`.
  - [ ] 1.6 RED → GREEN: `MaxCacheBoundaries` is readable as a constant without constructing anything. `S-ACB-025`.
  - [ ] 1.7 RED → GREEN: extend the landed `import_boundary_test.go` pattern so the request-validation closure is asserted to contain no network and no filesystem package — do not write a second guard. `S-ACB-026`.

- [ ] **Item 2** — An adapter can read markers **in tools → system → messages order** regardless of the order in which they were set, because that is the order the invalidation cascade runs in.
  - [ ] 2.1 RED: `TestRequest_CacheBoundaries_YieldCascadeOrderRegardlessOfPlacementOrder` — place the message marker first, then the system marker, then the tool marker; assert the returned sequence is tools, system, messages. `S-ACB-027`. **Blocking assumption**: the tools leg needs AI-10.3's tool accessor — `explore.md` § 8 row 1.
  - [ ] 2.2 GREEN: implement `CacheRegion`, its three constants from `iota + 1`, `CacheRegions()`, `String()`, `CacheBoundary` with `Region()`/`Index()`/`String()`, and `Request.CacheBoundaries()` delegating to the shared walk (`design.md` § 3.2).
  - [ ] 2.3 RED → GREEN: two markers in one region appear in ascending ordinal order. `S-ACB-028`.
  - [ ] 2.4 RED → GREEN: indexing a boundary's region accessor at its ordinal yields a carrier that reports it is a boundary. `S-ACB-029`.
  - [ ] 2.5 RED → GREEN: the same request constructed many times in one process yields an identical sequence every time — no map decides anything. `S-ACB-030`.

- [ ] **Item 3** *(appended)* — The listing and the enforced count come from one walk, and the region vocabulary is exhaustive.
  - [ ] 3.1 RED → GREEN *(pin)*: `CacheRegions()` holds exactly three members whose numeric order is tools, system, messages; a member declared without an entry in the enumeration fails the pin. Copy `role_test.go`'s constant-space walk — **not** a walk over the name table. `S-ACB-031`.
  - [ ] 3.2 RED → GREEN: for a request at the cap, the reported sequence length equals the count the cap rule enforced. `S-ACB-032`.
  - [ ] 3.3 RED → GREEN: a request carrying markers renders its boundary count and no payload through all four verbs; a request with no markers renders as before. `S-ACB-040`.
  - [ ] 3.4 Refactor: confirm `Request.CacheBoundaries()` and the cap rule call the **same** function and that neither re-walks a region.

- [ ] **AI-11.2 close:** record green `make test` and clean `make lint`; commit `feat(ai): cap and order the request's cache boundaries (AI-11.2)`.

---

## Phase AI-11.3 — advisory semantics `[leaf]`

**Deliverable:** `backend/agent/src/agenttest/cache_boundary_test.go`.
**Spec:** `R-ACB-008`, `R-ACB-009`. **Design:** § 6.
**Depends on:** AI-11.2.

Test list — items 1–2 are doc 0002's, verbatim; item 3 was appended during planning.

- [ ] **Item 1** — WHEN a translator ignores every marker THEN the request is still fully translatable and semantically unchanged — an adapter for an auto-caching provider is conformant while ignoring them entirely.
  - [ ] 1.1 RED → GREEN: write the **marker-blind** translator in `agenttest` — it reads model, system segment text, declaration name/description/schema, message role and content, tool choice and every generation option through the exported surface, and never calls `IsCacheBoundary` or `CacheBoundaries`. Assert `render(marked) == render(unmarked)`. `S-ACB-033`.
  - [ ] 1.2 RED → GREEN: the blind rendering of the marked request is complete relative to the unmarked twin — no region silently dropped from both. `S-ACB-035`.

- [ ] **Item 2** *(pin)* — The usage-side surface is untouched by this milestone — the cache token fields exist from AI-13.3 and this milestone adds request-side expression only.
  - [ ] 2.1 GREEN-from-birth pin: `Usage`'s cache-read and cache-write counts read and validate exactly as before, at the same positions. `S-ACB-036`.
  - [ ] 2.2 GREEN-from-birth pin: this layer exports no hit-rate, cache-efficiency or cache-statistics accessor. `S-ACB-037`.

- [ ] **Item 3** *(appended)* — The control that proves item 1 is not vacuous.
  - [ ] 3.1 RED → GREEN: the **marker-aware** control — the same walk plus one marker read — renders the marked request and its unmarked twin **differently**. Without it, a blind translator that rendered nothing would satisfy item 1 perfectly. `S-ACB-034`. This is AI-06's `testdata/handrolled` + `testdata/constructed` lesson applied to a rendering.

- [ ] **AI-11.3 close:** record green `make test` and clean `make lint`; commit `test(ai): prove cache-boundary markers are advisory (AI-11.3)`.

---

## Milestone close

- [ ] `make test` green in `backend/agent/` — paste the transcript.
- [ ] `make lint` clean in `backend/agent/` — paste the transcript. **Run it before every commit**, not only at the end.
- [ ] `go.mod` still at **zero requires**.
- [ ] Both AI-00 import guards pass.
- [ ] `openspec/specs/ai-contract-vocabulary/spec.md` unmodified — confirm with `git diff --stat`.
- [ ] `validation.go`'s `ruleClasses` and both registry mirrors unmodified — confirm with `git diff --stat`.
- [ ] Record the **actual** changed-line count against `explore.md` § 7's forecast in the Review Workload Forecast table above.
- [ ] Record any `explore.md` § 8 row that failed re-verification, and what was done instead.
- [ ] Never push, never merge, never open a PR, never `git stash`.
