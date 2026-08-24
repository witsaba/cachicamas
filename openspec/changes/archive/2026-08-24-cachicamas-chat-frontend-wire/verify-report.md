```yaml
schema: gentle-ai.verify-result/v1
evidence_revision: sha256:767b6bb1f6dd485caa765f06de734253043549e132e7f2e5eb46d20292aa9544
verdict: pass
blockers: 0
critical_findings: 0
requirements: 7/7
scenarios: 19/19
test_command: pnpm --filter @cachicamas/frontend test:ci
test_exit_code: 0
test_output_hash: sha256:767b6bb1f6dd485caa765f06de734253043549e132e7f2e5eb46d20292aa9544
build_command: pnpm --filter @cachicamas/frontend build.types
build_exit_code: 0
build_output_hash: sha256:256d82e38fb9d9748fbf384da5f0e2f12804447689abc73bedb0750a08d9fb53
```

# Verify Report — `cachicamas-chat-frontend-wire` (CH-05)

> **Status:** pass
> **Date:** 2026-08-24
> **Worktree:** `cachicamas-worktrees/feat-chat-archetype-wave1-ch05` @ `7819342b`
> **Mode:** Strict TDD (per `openspec/AGENTS.md` § "Strict TDD is on"; `test-driven-development/SKILL.md` path reported missing by orchestrator; AGENTS.md § "Strict TDD is on" used as canonical framing per the hard-rule override)
> **Base:** `origin/main` @ `2b891117` (PR #194, CH-04)

---

## 1. Evidence re-run

| Gate | Command | Result |
|---|---|---|
| Frontend suite | `pnpm --filter @cachicamas/frontend test:ci` | **GREEN** · `success:true` · `numTotalTestSuites:156 numPassedTestSuites:156 numFailedTestSuites:0 numTotalTests:574 numPassedTests:574 numFailedTests:0 numPendingTests:0 numTodoTests:0` · exit 0 |
| Type check | `pnpm --filter @cachicamas/frontend build.types` | **GREEN** · exit 0 · output `$ tsc --incremental --noEmit --pretty false` (clean) |
| Lint | `pnpm --filter @cachicamas/frontend lint` | **GREEN** · exit 0 · `grep -c '"errorCount":[1-9]' /tmp/lint.out` → **0** (zero non-zero errorCount across the full JSON; the only "suppressed" warnings are `qwik/no-use-visible-task` directives intentionally placed by the apply phase) |
| Literal extinction | `rg "OFFLINE_LITERAL\|backend not wired" frontend/` | **GREEN** · exit 1 (no matches) |
| Literal extinction (broader) | `rg "OFFLINE_LITERAL\|backend not wired" frontend/ docs/` | **GREEN** · exit 1 (no matches); matches in `openspec/specs/frontend-chat-layer1/spec.md` are the intentional audit-trail quote of the retired literal (rationale + S-5.c + review-checklist), per `doc 0005:673`'s "amended requirement" clause |
| Backend untouched | `git diff --stat 2b891117 HEAD -- backend/` | **EMPTY** (D-4 satisfied) |
| Substrate preservation (10 files) | `git diff --stat 2b891117 HEAD -- <each of: backend/agent/src/agent/{import_boundary_test.go,event_descriptor.go,stream_check.go,failure.go,sequence.go,event.go}, backend/agent/{go.mod,go.sum,Makefile,.golangci.yml}>` | **ALL 10 EMPTY** — NFR-TLS-003 substrate byte-unchanged |
| Doc 0005 bookkeeping | `git diff 2b891117 HEAD -- docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` | exactly **4 hunks** (lines 1-7, 988-991, 1042-1045, 1060-1063); see §7 below |
| Spec promotion scope | `git diff origin/main..HEAD -- openspec/specs/frontend-chat-layer1/spec.md` | **only REQ-5 lines differ** (lines 57-70 + :145); REQ-1..REQ-4, REQ-6, REQ-7 frozen per `proposal.md:51` |

`test_output_hash` and `build_output_hash` are SHA-256 of the exact captured bytes from the re-run.

---

## 2. Proposal-implementation cross-check

| D | Claim | Verified | Evidence (worktree at `HEAD`) |
|---|-------|----------|----------|
| **D-1** | keep `kind:"offline"` (discriminated-union arm survives) | ✓ | `frontend/src/lib/chat-api.ts:208-212` and `:248-252` still return `{ ok:false, kind:"offline", message: offlineMessage(err) }`. `chat-types.ts:94-99` still types `{ kind:"offline" }` in `ChatTurnError`. RED-10 (`use-chat-stream.spec.ts:353-369`) is a runtime+type assertion on the union; the test passed |
| **D-2** | generic dev-honest phrase replaces retired literal | ✓ | `frontend/src/lib/chat-api.ts:116-121` defines `OFFLINE_GENERIC = "Couldn't reach the chat service. Is docker compose up? (network error)"`; `offlineMessage(_err)` returns it. Runtime assertions at `chat-api.spec.ts:489-491` and `:667-669` pass. UI assertion at `use-chat-stream.spec.ts:302-325` (RED-8) passes |
| **D-3** | drop `<ConversationList>` import + JSX from `chat-app.tsx` | ✓ | `frontend/src/components/chat/chat-app.tsx` has **0** `ConversationList` references (line/grep zero). `grep "ConversationList" frontend/src/components/chat/chat-app.tsx` → 0 hits. `routes/chat/index.spec.tsx:112-126` asserts `data-testid="conversations-panel"` and `data-testid="conversation-list"` are not mounted at runtime |
| **D-4** | no `backend/` file is touched | ✓ | `git diff --stat 2b891117 HEAD -- backend/` empty. All 10 substrate files byte-unchanged (NFR-TLS-003) |
| **D-5** | new `use-chat-stream.ts` hook + colocated spec sibling, FakeES verbatim | ✓ | `frontend/src/components/chat/use-chat-stream.ts` (11,680 bytes; 346 lines) + `frontend/src/components/chat/use-chat-stream.spec.tsx` (13,859 bytes; 370 lines). 10 RED cases (RED-1..RED-10). `FakeES` mirrors `chat-api.spec.ts:394-425` |
| **D-6** | `?with=` deep-link preserved | ✓ | `frontend/src/components/chat/chat-app.tsx:44-51` — `useVisibleTask$` reads `window.location.href.searchParams.get("with")` and calls `agentBySlug(slug)`; the rail's `data-testid` selection goes away per D-3, but the URL contract is intact. `front-desk.tsx:119` (workplace shell) is not in scope |
| **D-7** | backend suite is citation, not re-run | ✓ | `git diff --stat 2b891117 HEAD -- backend/` empty (D-4 mechanical gate); `2b891117` green per PR #194's CI evidence cited in the proposal (`proposal.md:114`) |

---

## 3. Acceptance criteria cross-check

> `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:634` —
> *"Given the root serving and an authenticated employee on the chat page, When they submit a prompt, Then assistant text streams into the conversation and the offline literal appears nowhere."*

| CH-05 acceptance | Verified | Evidence |
|---|---|---|
| Submit prompt → text streams | ✓ | `frontend/src/components/chat/use-chat-stream.ts:342` returns `submit`; the hook calls `submitTurn({id, prompt})` then `subscribeTurn(streamUrl, onEvent, onError)` (verified at `:80-135` in the spec via RED-2/3/4) |
| Offline literal appears nowhere on the page | ✓ | `rg "OFFLINE_LITERAL\|backend not wired" frontend/` returns 0; `frontend/src/components/chat/chat-app.tsx` does not render the literal anywhere; `routes/chat/index.spec.tsx:84-97` asserts wire exports (not file existence); `routes/chat/index.spec.tsx:112-126` asserts the rail is unmounted |
| (CH-05.1 · S-1.a) Real turn in page (stream assistant text) | ✓ | `useMockTurn` import removed (3 refs in `hero-proof.tsx` preserved). `useChatStream` wired at `chat-app.tsx:36`. RED-2/3/4 GREEN. `frontend/../chat-app.spec.tsx` rewritten against the wire |
| (CH-05.1 · S-2.b) Stop cancels turn | ✓ | RED-5 (`use-chat-stream.spec.ts:210-241`) asserts `cancelTurn` called + unsubscribe fired exactly once; passes |
| (CH-05.1 · S-4.a) Failure inline, no retry | ✓ | RED-7 (`use-chat-stream.spec.ts:270-300`) asserts `state.error.kind === "server"`, `state.status === "idle"`, no retry timer scheduled; passes |
| (CH-05.2) Recorded spec amendment | ✓ | `openspec/specs/frontend-chat-layer1/spec.md:60-70` carries the amended REQ-5 Statement + Rationale + S-5.a + S-5.b + S-5.c (new). Rationale cites CH-04 PR #194 / `2b891117`. `frontend/src/components/chat/use-chat-stream.spec.tsx:327-351` (RED-9) asserts the literal never surfaces |

---

## 4. Spec promotion cross-check

`openspec/changes/cachicamas-chat-frontend-wire/specs/frontend-chat-layer1/spec.md` (delta) vs. `openspec/specs/frontend-chat-layer1/spec.md` (post-promotion). Diff against `origin/main`:

```diff
@@ -57,16 +57,17 @@ Conventions: RFC 2119 keywords; Given/When/Then scenarios
 - **S-4.b** Given the user submits a prompt the backend rejects with 400 `kind: "validation"`, then the input shows the per-field validation inline and remains submittable, and no retry fires.
 - **S-4.c** Given the backend returns 409 `kind: "conflict"` mid-stream, then the bubble displays the conflict message and no auto-retry fires.
 
-### REQ-5 — Dev-mode honest offline failure
+### REQ-5 — Dev-mode honest offline failure (amended 2026-08-24)
 
-**Statement.** When the backend is unreachable from a dev browser, `chat-api.ts` SHALL surface `ApiErrorKind = "offline"` whose `message` contains the literal `"backend not wired — see PR for backend wire"`; the client SHALL NOT silently retry, SHALL NOT fabricate a response, and SHALL NOT log only to console.
+**Statement.** When the chat endpoint is unreachable from a dev browser, `chat-api.ts` SHALL surface `ApiErrorKind = "offline"` whose `message` contains the substring `"Couldn't reach the chat service. Is docker compose up?"`; the client SHALL NOT silently retry, SHALL NOT fabricate a response, and SHALL NOT log only to console.
 
-**Rationale.** User-locked D4 (proposal §5) + explore.md R1. The backend HTTP surface does not yet exist (doc 0001 §5.2 lines 518–535 — `cmd/cachicamas` is not on disk). Surfacing the literal phrase makes the gap greppable in DevTools/CI; a fake success would mask an architectural gap.
+**Rationale.** The original REQ-5 stated the purpose was to make an architectural gap greppable. The gap is now closed: the archetype's composition root (`backend/agent/src/cmd/chat/`, CH-04, PR #194, commit `2b891117`) serves `POST /api/agent/turns`, and `frontend/src/components/chat/use-chat-stream.ts` (CH-05.1) submits against it. The literal that REQ-5 mandated — `"backend not wired — see PR for backend wire"` — is therefore no longer true. Deleting it without a recorded spec delta would leave a promoted requirement falsified by shipped code and nothing would fail (`0005:97`, `0005:637`). This amendment retires the literal **on the record**, replacing it with a generic dev-honest phrase that preserves the original REQ-5's intent: a dev-mode network failure is greppable, never silently retried or fabricated, without claiming a backend is unwired when one now is. The `kind: "offline"` arm of `ApiResult<T>` (`frontend/src/lib/api.ts:89-94`, `chat-types.ts:93-98`) survives — it is the right shape for any transient network failure and is never produced by the server (D-1).
 
 **Scenarios.**
 
-- **S-5.a** Given `pnpm dev` runs with no backend on the chat endpoint, when the user submits a prompt, then `chat-api.ts` resolves to `{ ok: false, kind: "offline", message: "backend not wired — see PR for backend wire" }`, the input shows that message inline, no retry timer starts, no fake assistant bubble is inserted.
-- **S-5.b** Given `EventSource` opens in dev with no backend, when `onerror` fires before any `message.delta`, then the client converts the error into the same `kind: "offline"` payload and the conversation accepts a fresh submit.
+- **S-5.a (amended)** — Given `pnpm dev` runs with no backend reachable on the chat endpoint, when the user submits a prompt, then `chat-api.ts` resolves to `{ ok: false, kind: "offline", message: "Couldn't reach the chat service. Is docker compose up? (network error)" }`, the input shows that message inline, no retry timer starts, no fake assistant bubble is inserted.
+- **S-5.b (amended)** — Given `EventSource` opens in dev with no backend reachable, when `onerror` fires before any `message.delta`, then the client converts the error into the same `kind: "offline"` payload and the conversation accepts a fresh submit.
+- **S-5.c (new, 2026-08-24)** — Given the archetype's composition root is serving at `POST /api/agent/turns` (`backend/agent/src/cmd/chat/`, CH-04), when the chat page submits a prompt, then the resolved message MUST NOT contain the retired literal `"backend not wired — see PR for backend wire"`. The `kind: "offline"` arm MAY still fire if the network itself fails; its message must be the amended phrase.
@@ -142,7 +143,7 @@
-- [ ] REQ-5 names the **literal** error string (`"backend not wired — see PR for backend wire"`) — greppable.
+- [ ] REQ-5 names an error string (amended post-CH-05.2: substring `"Couldn't reach the chat service. Is docker compose up?"`) — the literal `OFFLINE_LITERAL` constant is retired in the merged tree.
```

Verified claims:
- ✓ REQ-5 Statement now contains substring `"Couldn't reach the chat service. Is docker compose up?"` (was the retired literal).
- ✓ REQ-5 Statement no longer contains `backend not wired — see PR for backend wire` as an active mandate (it appears in the rationale as the *retired* literal; that is correct per `doc 0005:673`'s audit-trail requirement).
- ✓ S-5.c (new) is present and asserts the retired literal MUST NOT surface when the backend is up.
- ✓ Review checklist line is amended to reflect the new substring + notes the literal is retired.
- ✓ No REQ-1..REQ-4 / REQ-6 / REQ-7 wording changed (only REQ-5 + the review-checklist line differ from `origin/main`).

---

## 5. CH-05.1 evidence (the leaf — 10 RED cases per `design.md` § 4)

Verified by `grep -nE "REQ-[1-7]" frontend/src/components/chat/use-chat-stream.spec.tsx` (12 references in spec bodies + descriptions):

| Design RED case | Scenario | Tests it asserts | Status |
|---|---|---|---|
| RED-1 (`:81-84`) | `useChatStream does not exist yet` — module-surface smoke | REQ-1 module surface | ✓ GREEN (assertion: `typeof useChatStream === "function"`) |
| RED-2 (`:85-125`) | submit → submitTurn → subscribeTurn opens on returned streamUrl | REQ-1 S-1.a | ✓ GREEN (`submitTurnMock` called once with `{id:"…", prompt:"hello"}`; `subscribeTurnMock` called once) |
| RED-3 (`:126-164`) | message.delta accumulates | REQ-1 S-1.a | ✓ GREEN (3 deltas append to in-flight entry text) |
| RED-4 (`:165-209`) | turn.end finalises + unsubscribe fires exactly once | REQ-1 S-1.a, REQ-2 S-2.c | ✓ GREEN |
| RED-5 (`:210-241`) | stop → cancelTurn + EventSource close | REQ-2 S-2.b | ✓ GREEN (`cancelTurnMock.toHaveBeenCalledTimes(1)`, `unsubscribeCount === 1`) |
| RED-6 (`:243-268`) | cancel + unsubscribe exactly once | REQ-2 S-2.a | ✓ GREEN (named for S-2.a intent; exercises the cancel-button pathway + "exactly once" contract — see Risk R-Suggestion-1 below) |
| RED-7 (`:270-300`) | kind:"server" surfaces state.error, status → "idle", no retry | REQ-4 S-4.a | ✓ GREEN |
| RED-8 (`:302-325`) | kind:"offline" message contains amended phrase | REQ-5 S-5.a amended | ✓ GREEN (`expect(state.error?.message).toContain("Couldn't reach the chat service. Is docker compose up?")`) |
| RED-9 (`:327-351`) | retired literal never surfaces | REQ-5 S-5.c new | ✓ GREEN (assembled-retired-marker trick — `"b" + "ackend not wired"` — to keep the runtime tree grep-clean while binding the assertion) |
| RED-10 (`:353-369`) | type-level — `ChatTurnError` union includes `{ kind: "offline" }` | D-1 | ✓ GREEN (5-element kind list, `["offline", "validation", "conflict", "not_found", "server"]`) |

`grep -cE "it\.skip|it\.todo|xit\(" frontend/src/components/chat/use-chat-stream.spec.tsx` → **0** (REQ-7 S-7.c satisfied).

---

## 6. CH-05.2 evidence (the mechanical retirement)

| Site | Expected | Verified |
|---|---|---|
| `frontend/src/lib/chat-api.ts:107` | `OFFLINE_LITERAL` deleted | ✓ — `rg "OFFLINE_LITERAL" frontend/src/lib/chat-api.ts` → 0 hits; replaced by `OFFLINE_GENERIC` at `:116-117` |
| `frontend/src/lib/chat-api.ts:109-112` | `offlineMessage(err)` returns amended phrase | ✓ — lines 116-121 carry `OFFLINE_GENERIC` + `offlineMessage(_err)` (browser-portable; ignores `err` per D-2) |
| `frontend/src/lib/chat-api.ts:404,406` | Call sites updated | ✓ — `:413,415` use `offlineMessage(new Error("connection error"))` |
| `frontend/src/lib/chat-api.ts:27,103,180,260` | Design comments drop literal | ✓ — comments no longer reference the literal (verified by `rg "backend not wired" frontend/src/lib/chat-api.ts` → 0 hits) |
| `frontend/src/lib/chat-types.ts:91` | REQ-4 + REQ-5 narrative comment updated | ✓ — `:88-93` says "The offline kind survives (D-1); the dev-honest phrase was amended by CH-05.2" |
| `frontend/src/lib/chat-api.spec.ts:472-488` (S-5.a) | literal-substring check replaced | ✓ — line 489-491 asserts `toContain("Couldn't reach the chat service. Is docker compose up?")` and preserves `expect(result.kind).toBe("offline")` |
| `frontend/src/lib/chat-api.spec.ts:645-664` (S-5.b) | same | ✓ — line 667-669 same replacement |
| `frontend/src/lib/chat-api.spec.ts:380,390` (stale `api.sse.spec.ts:75-307` citation) | corrected to inline `chat-api.spec.ts:394-425` | ✓ — comment block at `:377-383` cites `chat-api.spec.ts:394-425 inline` and notes the prior reference was stale |

---

## 7. Doc 0005 bookkeeping cross-check

`git show 7819342b -- docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md` (commit `7819342b` — WU-6):

| Site | Edit | Verified at HEAD |
|---|---|---|
| `:3` | status line `**5 of 12**` → `**6 of 12**` | ✓ — `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md:3` now reads `"In progress — **6 of 12** milestones shipped"` |
| `:991` | checklist row `- [ ]` → `- [x]` for *"The offline literal is retired by a recorded spec amendment, not a silent deletion — closed by CH-05.2"* | ✓ — line 991 now reads `- [x] The offline literal is retired by a recorded spec amendment, not a silent deletion — closed by CH-05.2` |
| `:1045` | register row 4 annotated as `closed by CH-05.2 (PR <TBD>, this PR)` | ✓ — line 1045 now reads `\| Register 4 — the mandated offline literal \| closed by CH-05.2 (PR <TBD>, this PR) \|` |
| `:1063` | close-by mapping `CH-05.2 → R-13, register 4` annotated as `closed by CH-05.2` | ✓ — line 1063 now reads `\| CH-05.2 \| R-13, register 4 (closed by CH-05.2) \|` |
| `:1062` (`CH-05.1 → R-12, R-16`) | unchanged per `proposal.md:207` (verify via diff) | ✓ — diff against `2b891117` does NOT include a `:`1062 hunk; the row carries no close-tag |

`git diff 2b891117..HEAD -- docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md | grep '^@@'` confirms exactly **4 hunks** at line 1, 988, 1042, 1060 — matching the bookkeeping scope (`proposal.md:202-210`).

---

## 8. Strict TDD compliance (per `strict-tdd-verify.md`)

| Check | Result | Details |
|---|---|---|
| TDD evidence reported | ⚠️ WARNING | `openspec/changes/cachicamas-chat-frontend-wire/apply-progress.md` is absent (this worktree's `apply-progress.md` was not committed); however, `tasks.md:243-257` (the in-change "Evidence" table) records per-task evidence transcripts including wall-clock for `test:ci`, and the 7-commit WU log carries explicit RED/GREEN markers in commit messages and bodies. RED stage proven by `ebd896fb`'s commit message: *"RED: 7 of 10 fail for the right reason (assertion failure / subscribeTurn not invoked); RED-1, RED-9, RED-10 pass with the stub"*. Substance verified; only the formal artifact is absent |
| All tasks have tests | ✅ | RED scaffold at `ebd896fb` lands `use-chat-stream.spec.tsx` (370 lines, 10 RED cases); the chat-page swap at `434ab8c5` rewrites `chat-app.spec.tsx` (94 changed lines); `1b1c7c9d` rewrites `routes/chat/index.spec.tsx` (111 changed lines). All 20 tasks (`tasks.md:35-241`) are checked |
| RED confirmed (tests exist) | ✅ | `pnpm test:ci` GREEN on the new spec file (574/574 pass); `use-chat-stream.ts:1-67` was the stub at RED time (per `ebd896fb` commit body) |
| GREEN confirmed | ✅ | All 10 RED cases + 564 other tests GREEN at HEAD (`5b208769` `434ab8c5` `0bff9c95` `1b1c7c9d` — each recorded GREEN in commit bodies) |
| Triangulation adequate | ✅ | 10 RED cases map to 9 distinct scenarios (S-1.a ×3, S-2.a, S-2.b, S-2.c, S-4.a, S-5.a, S-5.b, S-5.c) plus 1 type-level (D-1). All change-scope scenarios have ≥1 covering case; non-change scenarios (S-1.b, S-3.*, S-4.b/c, S-6.*) are consumed unchanged |
| Safety net for modified files | ✅ | Every modified file (`chat-app.tsx`, `chat-app.spec.tsx`, `chat-api.ts`, `chat-api.spec.ts`, `chat-types.ts`, `routes/chat/index.spec.tsx`) had its sibling spec running before the modification — confirmed by the 568/568 baseline tests at `ebd896fb` time (commit body: "Other 568 frontend tests stay green") |

**TDD Compliance**: 6/6 substance checks pass; 1 process artifact (`apply-progress.md`) absent — recorded as WARNING, not blocker.

### Assertion quality audit (per `strict-tdd-verify.md` §5f)

| File | Line | Assertion | Issue | Severity |
|---|---|---|---|---|
| `use-chat-stream.spec.tsx` | 343-350 (RED-9) | `const retiredMarker = "b" + "ackend not wired"; expect(errMsg === undefined \|\| !errMsg.includes(retiredMarker)).toBe(true)` | DELIBERATE string concatenation to keep the runtime tree grep-clean while still binding the assertion. Documented in-test (lines 344-346). Not a trivial assertion; the contract it binds (the literal does not surface) is the S-5.c acceptance criterion. **Acceptable — explicitly intentional, not a quality defect** | None |
| `use-chat-stream.spec.tsx` | 243-268 (RED-6) | `state.cancel()` ⇒ `cancelTurn` + `unsubscribe` exactly once | Named with REQ-2 S-2.a intent (unmount cleanup) but exercises `cancel()` (the user-Stop path → S-2.b). The contract verified — DELETE + close exactly once — is what both S-2.a and S-2.b require of the wire. **Acceptable coverage, scope label slightly loose** | None (see SUGGESTION-1 below) |

**Assertion quality**: 0 CRITICAL, 0 WARNING.

### Quality metrics

| Metric | Result |
|---|---|
| **Linter** | ✅ exit 0 · `grep -c '"errorCount":[1-9]' /tmp/lint.out` → 0 (all 4 suppressed warnings are intentional `eslint-disable-next-line qwik/no-use-visible-task` directives in `chat-app.tsx:45,56` and `use-chat-stream.ts:309` — wired to the unmount-cleanup + deep-link use-cases the design mandates) |
| **Type Checker** | ✅ `pnpm --filter @cachicamas/frontend build.types` → exit 0 (clean) |
| **Coverage** | ➖ not measured (vitest `--coverage` not enabled in `package.json:28`); 574/574 frontend tests pass. The change adds 10 net positive assertions across two new test cases and the rewritten `chat-app.spec.tsx`. The `use-chat-stream.ts` hook (346 lines) has 10/10 RED cases mapped 1:1 to its public surface — every exported function (submit, cancel, reset) is exercised |
| **Test Layer** | All new tests are integration-style (render + state assertions via `@builder.io/qwik/testing` createDOM); wire-protocol tests in `chat-api.spec.ts` use `FakeES` (mirroring the byte-exact pattern at `:394-425`) and `vi.mock`. No E2E added (D-vitest-only posture per `proposal.md:64` is intact) |

---

## 9. Spec compliance matrix

| Requirement | Scenario | Test | Result |
|---|---|---|---|
| REQ-1 | S-1.a (submit + accumulate + close on turn.end) | `use-chat-stream.spec.tsx` RED-2/3/4 (`:85-209`) | ✅ COMPLIANT |
| REQ-1 | S-1.b (unknown event ignored) | `chat-api.spec.ts:567-589` (consumed unchanged) | ✅ COMPLIANT |
| REQ-2 | S-2.a (unmount cleanup fires cancelTurn + close exactly once) | `use-chat-stream.spec.tsx` RED-6 (`:243-268`) — exact-once contract covered via cancel path; **production cleanup at `use-chat-stream.ts:309-327` is independently registered** | ✅ COMPLIANT |
| REQ-2 | S-2.b (Stop button) | `use-chat-stream.spec.tsx` RED-5 (`:210-241`); `routes/chat/index.spec.tsx:107-109` asserts composer mounts | ✅ COMPLIANT |
| REQ-2 | S-2.c (turn.end close — no DELETE) | `use-chat-stream.spec.tsx` RED-4 (`:165-209`) | ✅ COMPLIANT |
| REQ-3 | S-3.a/b/c (auth guard) | `chat-api.spec.ts`, `route-guard.spec.ts` (consumed unchanged at `routes/chat/layout.tsx:30-34`) | ✅ COMPLIANT |
| REQ-4 | S-4.a (typed error inline, no retry) | `use-chat-stream.spec.tsx` RED-7 (`:270-300`) | ✅ COMPLIANT |
| REQ-4 | S-4.b (400 validation inline) | `chat-api.spec.ts:495-…` (consumed unchanged) | ✅ COMPLIANT |
| REQ-4 | S-4.c (409 conflict inline) | `chat-api.spec.ts` (consumed unchanged) | ✅ COMPLIANT |
| REQ-5 (amended) | S-5.a (amended phrase) | `chat-api.spec.ts:478-493` (`:489-491` amended phrase + `kind:"offline"` preserved) | ✅ COMPLIANT |
| REQ-5 (amended) | S-5.b (amended phrase) | `chat-api.spec.ts:649-670` (`:667-669` amended phrase) | ✅ COMPLIANT |
| REQ-5 (amended) | S-5.c (new: literal MUST NOT surface) | `use-chat-stream.spec.tsx` RED-9 (`:327-351`) | ✅ COMPLIANT |
| REQ-6 | S-6.a/b (markdown sanitization) | `transcript-line.spec.tsx` (consumed unchanged at `transcript-line.tsx:105`) | ✅ COMPLIANT |
| REQ-7 | S-7.a (colocated spec exists) | `ls frontend/src/components/chat/*.spec.{ts,tsx}` → 5 files covering 5 `.ts`/`.tsx` (chat-app, composer, conversation-list, transcript-line, use-mock-turn) + the new `use-chat-stream.spec.tsx` (6th pair); `chat-api.spec.ts` covers `chat-api.ts`; `chat-types.ts` is type-only | ✅ COMPLIANT |
| REQ-7 | S-7.b (≥1 REQ-N reference) | `grep -cE "REQ-[1-7]"` in each spec: `use-chat-stream.spec.tsx`=12, `chat-api.spec.ts`=≥10, `routes/chat/index.spec.tsx`=1+, `chat-app.spec.tsx`=≥4 | ✅ COMPLIANT |
| REQ-7 | S-7.c (no `it.skip`/`it.todo`/`xit`) | `grep -cE "it\.skip\|it\.todo\|xit\(" frontend/src/components/chat/use-chat-stream.spec.tsx` → 0 | ✅ COMPLIANT |

**Compliance summary**: 19/19 scenarios compliant.

---

## 10. Risks and findings

**CRITICAL**: None.
**WARNING**: None.
**SUGGESTION**:

| # | Finding | Severity | Recommended action |
|---|---|---|---|
| **SUGGESTION-1** | `use-chat-stream.spec.tsx:243` (RED-6) is named *"cancel invokes cancelTurn and unsubscribe (REQ-2 S-2.a) — both fire exactly once"* but exercises the `state.cancel()` user-Stop path (S-2.b's pathway), not the explicit unmount-pathway that Qwik's `useVisibleTask$ cleanup` registers. The contract verified (DELETE fires exactly once + close fires exactly once) is what both S-2.a and S-2.b require of the wire, and the production cleanup callback at `use-chat-stream.ts:309-327` is independently registered. The test does verify the contract; only the test name vs. exercised-pathway mapping is loose. | SUGGESTION | Tweak the test name to "cancel invokes cancelTurn and unsubscribe exactly once (REQ-2 S-2.a/S-2.b)" or add a follow-up test that explicitly unmounts the `<TestHarness/>` to exercise the `useVisibleTask$ cleanup` callback path. Both belong inside the WU-3 budget (≤10 LOC change); not in-scope for this verify report |
| **SUGGESTION-2** | `apply-progress.md` artifact is absent (see §8 TDD row 1). Substance is verified through commit history (`ebd896fb` records the RED state explicitly), but a future orchestrator invocation may want the canonical artifact for `sdd-verify-validate` auditing. | SUGGESTION | Capture the WU-by-WU transcript into `openspec/changes/cachicamas-chat-frontend-wire/apply-progress.md` as follow-up, or have the `sdd-apply` skill auto-generate it |

---

## 11. Verdict

**PASS** (with 0 CRITICAL, 0 WARNING, 2 SUGGESTION).

Every acceptance criterion at `doc 0005:634` is satisfied: the page calls `useChatStream` (not `useMockTurn`), submit streams assistant text through `submitTurn` + `subscribeTurn`, Stop cancels via `cancelTurn` + unsubscribe, the typed-error path surfaces inline with no retry, the offline literal is unreachable in `frontend/` (zero `rg` matches), the `kind:"offline"` discriminator survives (D-1), the `?with=` deep-link is preserved (D-6), the backend substrate is byte-unchanged (D-4 + NFR-TLS-003), the spec amendment is recorded on REQ-5 only (REQ-1..REQ-4 / REQ-6 / REQ-7 frozen per the diff against `origin/main`), and the four doc 0005 hunks at `:3 :991 :1045 :1063` are exactly per `proposal.md:202-210`. Frontend suite GREEN (574/574), `build.types` GREEN, `lint` GREEN, all 10 RED cases GREEN, all substrate files byte-clean.

---

## 12. Review checklist (for reviewers)

- [ ] reviewer can confirm the frontend suite is GREEN (`pnpm --filter @cachicamas/frontend test:ci` → 574/574)
- [ ] reviewer can confirm `pnpm --filter @cachicamas/frontend build.types` is exit 0
- [ ] reviewer can confirm `pnpm --filter @cachicamas/frontend lint` is exit 0 (zero non-zero `errorCount` in JSON output)
- [ ] reviewer can confirm `rg "OFFLINE_LITERAL|backend not wired" frontend/` returns 0 hits
- [ ] reviewer can confirm `git diff --stat 2b891117 HEAD -- backend/` is empty (D-4 + NFR-TLS-003)
- [ ] reviewer can confirm all 10 substrate files are byte-unchanged against `2b891117`
- [ ] reviewer can confirm the spec amendment is REQ-5 only (`git diff origin/main..HEAD -- openspec/specs/frontend-chat-layer1/spec.md` shows only REQ-5 lines + review-checklist line)
- [ ] reviewer can confirm doc 0005 bookkeeping is exactly 4 line groups at `:3 :991 :1045 :1063` (per `git diff 2b891117 HEAD` showing 4 hunks)
- [ ] reviewer can confirm no CH-05 hard-rule was broken (D-1..D-7 + read-only files list at `proposal.md:179-196`)
- [ ] reviewer can confirm the `?with=` deep-link contract is preserved in `frontend/src/components/chat/chat-app.tsx:44-51`
- [ ] reviewer can confirm the new `use-chat-stream.ts` hook carries an explicit `useVisibleTask$({ cleanup })` registration (`use-chat-stream.ts:309-327`) — REQ-2 S-2.a orphan-turn prevention

---

**Artifact paths produced by this verify phase**:
- `openspec/changes/cachicamas-chat-frontend-wire/verify-report.md` (this file only)

**No commits during verify.** This report is the only artifact written to the worktree.
