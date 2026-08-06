# Archive Report — `cachicamas-frontend-chat-layer1`

> **Change**: `cachicamas-frontend-chat-layer1`
> **Archive date**: 2026-08-06
> **Archive folder**: `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/`
> **Branch closed**: `feat/frontend-chat-layer1` → `main` (single-PR, `size:exception` APPROVED at orchestrator review gate)
> **HEAD at close**: `6379fe228a7fba0f6fe7bfc562f41722f18ca1f7`
> **Fork point**: `57169dcad481276e0b06dea14bfb479cc2b4c393`
> **Artifact store**: hybrid (OpenSpec file + Engram topic `sdd/cachicamas-frontend-chat-layer1/archive-report`)
> **Strict TDD**: 6/6 compliance per `verify-report.md` §9; carried forward verbatim from orchestrator preflight (`strict_tdd: true`).

---

## §1. Verdict

**CLOSED — PASS.** The change is archived with a fully passing audit trail. `sdd-verify` returned **PASS** (canonical verdict; `requirements: 7/7`, `scenarios: 18/18`, `blockers: 0`, `critical_findings: 0`, `verdict: pass`); `sdd-apply` returned PASS with all 10 implementation tasks executed across 11 commits; all hard constraints from the orchestrator preflight hold (no backend/`openspec/specs/` edits, no new frontend dependencies, no AI attribution in commits, REQ-5 literal present at three call sites, lint and `tsc --noEmit` clean). Two non-blocking WARNINGS documented in `verify-report.md` §7 are orchestrator-accepted (W-1 baseline prettier noise, W-2 scope drift on `routes/chat/route-guard.spec.ts`). The SDD cycle is closed; ready for `git push` / PR creation by the orchestrator.

## §2. Spec sync status

| What | Status |
|---|---|
| Canonical spec location | `openspec/specs/frontend-chat-layer1/spec.md` (149 lines; unchanged at archive; SHA `4bb3d45039da4b38104fd61496503c7eee617254` on disk) |
| Spec authored at | `sdd-spec` phase, 2026-08-06 — Engram observation #2579 |
| Delta spec at `openspec/changes/cachicamas-frontend-chat-layer1/specs/{domain}/spec.md` | **NOT PRESENT** — the proposal §7 named the canonical location directly; the spec was authored at `openspec/specs/frontend-chat-layer1/spec.md` from the start. No delta merge is required. |
| Change folder moved to | `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/` per `openspec-convention.md` §Archive Structure |

**Verdict on sync**: spec is in its canonical location and contains the full binding contract (REQ-1..REQ-7, S-1.a..S-7.c). No sync action was needed at archive time; this report records that fact.

## §3. Final-state facts (handoff from orchestrator)

These six facts outrank any stale snapshot claim in `apply-progress` (Engram #2585) or `verify-report` (Engram #2600):

1. **Apply outcome**: **PASS**. T-01..T-10 executed across 11 commits. T-08 required an in-batch QRL prop fix at commit `104a73b` (`fix(chat): T-10 verify pass — wire ChatInput QRLs cleanly`); `tsc` + `eslint` caught the issue during verification, the fix was committed, and the verification was re-run green. Source: orchestrator launch handoff; cross-checked against `git log feat/frontend-chat-layer1 ^57169dc` and `verify-report.md` §5.

2. **Verify verdict**: **PASS**. All 7 REQs have passing vitest assertions. 78 chat-related tests pass; 773 total project tests pass; 0 regressions vs baseline (694). Source: `verify-report.md` §1 verdict envelope + §4.3 test output (`Test Files 84 passed (84); Tests 773 passed (773)`); cross-checked against Engram #2600.

3. **Two non-blocking warnings documented at verify** (do NOT re-litigate):
   - **W-1**: 9 pre-existing prettier warnings on `frontend/src/components/skills/`, `frontend/src/components/prompts/`, and `frontend/src/routes/settings/` files. Orchestrator-accepted baseline noise — `verify-report.md` §4.1 verified byte-identical to baseline by `git stash --include-untracked` + re-run. **NOT introduced by this change.**
   - **W-2**: `frontend/src/routes/chat/route-guard.spec.ts` was added under T-08 but NOT enumerated in `proposal.md §3` file table. The file follows the established project pattern (`routes/{home,profile,workspaces,ownboarding}/route-guard.spec.ts`); the deviation is naming only, not behavior. 5 structural-wiring assertions; 68 lines.

4. **Backend HTTP gap is intentional and explicit**. Per `docs/architecture/0001-cachicamas-agent-stack-v2.md` §5.2, `cmd/cachicamas` does not exist on disk; per `proposal.md §4 R2`, real end-to-end demo awaits a separate backend PR. D4 vitest-only is the v1 verification posture, user-confirmed at explore review (`proposal.md` §5 D4). Recorded SSE transcript fixture (`frontend/src/components/chat/__fixtures__/single-turn.sse`) pins the wire envelope so the contract survives backend arrival.

5. **Coupling with parallel `add-openrouter-first-provider` PR is independent**. Per `proposal.md §6` and `verify-report.md` §2 check #1: the frontend is vendor-neutral (D7 names events, not the model), so neither ordering blocks the other. The OpenRouter PR is backend-only; it can land before or after this PR with no compatibility cost.

6. **Pattern recorded for future sessions**: Engram #2584 (`workflow/cachicamas-size-exception-pattern`) — when a frontend change hits the 800-line budget due to strict-TDD discipline (forecast ~2,100–3,750 corrected lines per `tasks.md §2`), the user is comfortable approving `size:exception` rather than splitting into chained PRs. Forecast ~3× budget → single PR + exception. The orchestrator's preflight declared `size:exception: APPROVED` before `sdd-apply` ran.

## §4. Deliverable summary

### 4.1 Git topology

| Item | Value |
|---|---|
| Branch | `feat/frontend-chat-layer1` |
| Target | `main` |
| Strategy | `single-pr` (no chain); `size:exception` APPROVED |
| Commits ahead of fork point | 11 |
| Fork point | `57169dcad481276e0b06dea14bfb479cc2b4c393` |

### 4.2 Commits (11, conventional-commits; no AI attribution; no `Co-Authored-By` trailer)

| # | SHA | T-NN | Subject |
|---|---|---|---|
| 1 | `d23b651` | T-01 | `feat(chat): typed chat event contracts (REQ-1, REQ-4, REQ-6)` |
| 2 | `fb7cd93` | T-02 | `feat(chat): recorded SSE transcript fixture (REQ-1, REQ-7)` |
| 3 | `abf511c` | T-03 | `feat(chat): typed SSE client with vitest specs (REQ-1, REQ-2, REQ-4, REQ-5)` |
| 4 | `3d3c2ff` | T-04 | `feat(chat): useChatStream$ hook (REQ-1, REQ-2, REQ-7)` |
| 5 | `79fa8ad` | T-05 | `feat(chat): ChatWindow composes messages + input (REQ-1, REQ-4)` |
| 6 | `b1fe40e` | T-06 | `feat(chat): MessageBubble with DOMPurify-stripped markdown (REQ-6)` |
| 7 | `3f4ab44` | T-07 | `feat(chat): ChatInput with disabled-during-stream (REQ-1, REQ-2)` |
| 8 | `46b168d` | T-08 | `feat(chat): /chat route with auth guard (REQ-3, REQ-5, REQ-6)` |
| 9 | `6afb044` | T-09 | `feat(chat): link from AvatarDropdown (REQ-3 / T-09)` |
| 10 | `104a73b` | T-08 fix | `fix(chat): T-10 verify pass — wire ChatInput QRLs cleanly` |
| 11 | `6379fe2` | T-10 | `chore(chat): end-to-end verification (T-10)` |

All 11 expected commits present in `git log feat/frontend-chat-layer1 ^57169dc`. No unexpected commits. `104a73b` is a substantive `fix(chat):` per conventional-commits (lint+tsc surfaced a `qwik/use-method-usage` regression in chat-input's `useChatStream$()` fallback + a `qwik/valid-lexical-scope` regression in chat-window's destructured QRL; fix required for verification to pass).

### 4.3 Files created (16 total)

| Path | Lines | Source task |
|---|---:|---|
| `frontend/src/lib/chat-types.ts` | 239 | T-01 |
| `frontend/src/lib/chat-api.ts` | 414 | T-03 |
| `frontend/src/lib/chat-api.spec.ts` | 672 | T-01, T-02, T-03 |
| `frontend/src/components/chat/__fixtures__/single-turn.sse` | 18 | T-02 |
| `frontend/src/components/chat/use-chat-stream.ts` | 321 | T-04 |
| `frontend/src/components/chat/use-chat-stream.spec.ts` | 346 | T-04 |
| `frontend/src/components/chat/chat-window.tsx` | 116 | T-05 |
| `frontend/src/components/chat/chat-window.spec.tsx` | 237 | T-05 |
| `frontend/src/components/chat/message-bubble.tsx` | 66 | T-06 |
| `frontend/src/components/chat/message-bubble.spec.tsx` | 182 | T-06 |
| `frontend/src/components/chat/chat-input.tsx` | 115 | T-05 |
| `frontend/src/components/chat/chat-input.spec.tsx` | 202 | T-07 |
| `frontend/src/routes/chat/index.tsx` | 72 | T-08 |
| `frontend/src/routes/chat/layout.tsx` | 49 | T-08 |
| `frontend/src/routes/chat/index.spec.tsx` | 257 | T-08 |
| `frontend/src/routes/chat/route-guard.spec.ts` | 68 | T-08 (W-2: scope drift vs proposal §3) |
| **Total new** | **3,374** | |

### 4.4 Files modified (2 total)

| Path | Δ lines | Source task |
|---|---:|---|
| `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` | +17 | T-09 (one `MenuItem` for Chat) |
| `frontend/src/components/avatar-dropdown/avatar-dropdown.spec.tsx` | +46 (2 new tests + 1 amended) | T-09 |

### 4.5 Verification snapshot

| Metric | Value | Source |
|---|---|---|
| REQs covered | 7/7 | `verify-report.md` §3 |
| Scenarios covered | 18/18 | `verify-report.md` §3 |
| Chat-related tests passing | 78 (verify-audited) | `verify-report.md` §4.3 |
| Total project tests passing | 773 | `verify-report.md` §4.3 |
| Regressions vs baseline (694) | 0 | `verify-report.md` §9 Quality Metrics |
| `pnpm lint` | PASS | `verify-report.md` §4.1 |
| `pnpm build.types` (`tsc --noEmit`) | PASS | `verify-report.md` §4.1 |
| `pnpm test:ci` (vitest `--run`) | PASS (exit 0) | `verify-report.md` §4.3 |
| `pnpm verify` (lint + types + fmt.check + test:ci) | exit 1 (prettier baseline noise only) | `verify-report.md` §4.1 + W-1 |
| `pnpm test:e2e -- --grep chat` | 0 chat e2e authored (expected per D4) | `verify-report.md` §4.2 |
| Strict TDD compliance | 6/6 | `verify-report.md` §9 |
| REQ-5 literal sites in `chat-api.ts` | 3 (lines 27, 103, 107 — header comment, inline note, exported `OFFLINE_LITERAL`) | `verify-report.md` §4.4 |
| AI attribution in commits | 0 | `verify-report.md` §2 check #3 |
| New top-level `frontend/` dependencies | 0 | `verify-report.md` §2 check #2 |
| Backend/`openspec/specs/` edits vs fork point | 0 | `verify-report.md` §2 check #1 |

### 4.6 Prettier baseline (NOT introduced by this change)

9 pre-existing warnings on `frontend/src/components/prompts/`, `frontend/src/components/skills/`, `frontend/src/routes/settings/` files. Identical to baseline by `git stash --include-untracked` round-trip. **Zero chat files** affected. Orchestrator preflight explicitly accepts as expected baseline noise.

## §5. Warnings carried forward (do NOT reopen)

| ID | Description | Carry-forward action |
|---|---|---|
| **W-1** | 9 prettier baseline warnings on skills/prompts/settings | Land in a separate "frontend-baseline-prettier" housekeeping PR — NOT this chat change. `work-unit-commits` forbids touching unrelated files. |
| **W-2** | `routes/chat/route-guard.spec.ts` not enumerated in `proposal.md §3` (W-2 = documentation drift; behavior matches the 5 existing `routes/{home,profile,workspaces,ownboarding}/route-guard.spec.ts` instances) | Optionally amend `proposal.md §3` retroactively OR add a one-line convention note to `openspec/AGENTS.md` documenting that "structural route-guard specs are part of project convention" (per `verify-report.md` S-1). Cosmetic; the file already exists and passes verify. |

## §6. Out-of-scope confirmations (do NOT reopen)

These are intentional exclusions and remain excluded:

- **Backend HTTP surface** (`cmd/cachicamas` + `/api/agent/turns` + `/api/agent/turns/:id/events` + `DELETE /api/agent/turns/:id`) — separate backend PR. Per `docs/architecture/0001-cachicamas-agent-stack-v2.md` §5.2 and `proposal.md §4 R2`. D4 vitest-only is the v1 verification posture. The recorded SSE transcript fixture pins the wire envelope so this PR remains correct when the backend lands.
- **Tool-call rendering, subagent nesting, permission protocol, streaming reasoning deltas, multimodal content** — per `docs/architecture/0001-cachicamas-agent-stack-v2.md` §8 + §7 G1/G7/G12 ("seam now", no v1 implementation).
- **Session persistence across reloads** — v1 is in-memory only; reload = empty chat. Per `proposal.md §4` and `docs/architecture/0001-cachicamas-agent-stack-v2.md` §5.2 (Layer 3 owns append-only records with parent chains).
- **Provider / model / credential selection in the frontend** — backend owns it ("the back is who controls the keys"). The chat's `ChatTurnRequest` (`chat-types.ts:68`) intentionally omits provider/model/credential fields.
- **Cost / usage display, provider/model picker UI, branching UI** — per `docs/architecture/0001-cachicamas-agent-stack-v2.md` §7 G10 + §8. `ChatUsage` is typed in `chat-types.ts:48` as a seam but not surfaced in v1.
- **New top-level frontend dependencies** (no `@ai-sdk`, `openai`, `eventsource-parser`) — existing primitives suffice (`frontend/src/lib/api.ts`, `lib/markdown.ts`, `lib/csrf.ts`, `components/workspace-sync-card/use-sync-status.ts`).
- **Any change to `backend/agent/src/ai/openaicompat/openrouter/` or Layer 1 packages** — parallel OpenRouter change owns that.
- **Modifications to existing specs** (`api-error-envelope`, `frontend-auth`, `frontend-runtime`, `home-page`) — read-only contracts the chat consumes. None modified by this change.
- **The `size:exception` vs split decision** — resolved at `sdd-tasks` review (the orchestrator surfaced the fork to the user; the user picked "Approve size:exception"; `tasks.md §2.1` records the resolution). Do not re-surface.
- **The W-1 prettier baseline noise** — orchestrator-preflight-accepted; belongs to a separate housekeeping PR.
- **The W-2 `route-guard.spec.ts` documentation drift** — file matches project convention; documentation update is optional.

## §7. Cross-references

### 7.1 Architecture

- `docs/architecture/0001-cachicamas-agent-stack-v2.md` §2.2 lines 153–218 — chat as the fourth `CUSTOM` frontend consumer of the `CodingSessionEvent` stream.
- `docs/architecture/0001-cachicamas-agent-stack-v2.md` §4.1 lines 425–427 — retry is a harness concern, not the frontend's.
- `docs/architecture/0001-cachicamas-agent-stack-v2.md` §4.2 lines 450–455 — cancellation tree is harness-owned; frontend signals intent via `DELETE /api/agent/turns/:id`.
- `docs/architecture/0001-cachicamas-agent-stack-v2.md` §4.3 lines 460–487 — event envelope invariants (deltas carry an index, consumers accumulate; typed errors).
- `docs/architecture/0001-cachicamas-agent-stack-v2.md` §5.1 lines 493–506 — Layer 3 ports the chat consumes but defines none of.
- `docs/architecture/0001-cachicamas-agent-stack-v2.md` §5.2 lines 518–535 — `cmd/cachicamas` is the only composition root; the backend HTTP surface is a separate, follow-up change.
- `docs/architecture/0001-cachicamas-agent-stack-v2.md` §8 lines 630–649 — non-goals (TUI deferred, no multimodal, no branching UI).
- AI-13.1 — seven-value `ChatFinishReason` vocabulary.

### 7.2 OpenSpec artifacts (this change)

- `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/explore.md` (24,666 bytes) — frontend primitives survey + backend gap named.
- `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/proposal.md` (25,815 bytes) — D1–D7 decisions, success criteria, D4 vitest-only posture.
- `openspec/specs/frontend-chat-layer1/spec.md` (149 lines, canonical) — REQ-1..REQ-7, S-1.a..S-7.c.
- `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/design.md` (15,320 bytes) — typed `ChatStreamEvent` union, signal graph, cancel race, SSR-cookie handoff.
- `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/tasks.md` (16,659 bytes after checkbox reconciliation) — T-01..T-10 with per-task RED-GREEN-REFACTOR evidence.
- `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/verify-report.md` (32,776 bytes) — independent audit, PASS verdict, 7/7 REQs, 18/18 scenarios, 6/6 strict-TDD compliance.
- `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/archive-report.md` (this file) — closure.

### 7.3 Engram observations

- **#2575** `sdd/cachicamas-frontend-chat-layer1/explore` — explore phase output.
- **#2576** `sdd/cachicamas-frontend-chat-layer1/proposal` — proposal phase output.
- **#2579** `sdd/cachicamas-frontend-chat-layer1/spec` — spec phase output (canonical spec at `openspec/specs/frontend-chat-layer1/spec.md`).
- **#2581** `sdd/cachicamas-frontend-chat-layer1/design` — design phase output.
- **#2582** `sdd/cachicamas-frontend-chat-layer1/tasks` — tasks phase output.
- **#2584** `workflow/cachicamas-size-exception-pattern` — precedent for `size:exception` approval on strict-TDD frontend changes.
- **#2585** `sdd/cachicamas-frontend-chat-layer1/apply-progress` — T-01..T-10 execution evidence (TDD cycles + work units).
- **#2600** `sdd/cachicamas-frontend-chat-layer1/verify-report` — independent audit (PASS).

### 7.4 Existing codebase mirrors

- `frontend/src/lib/api.ts:96-110` — `ApiResult<T>` discriminated union (chat's `ChatTurnError` mirrors verbatim).
- `frontend/src/lib/api.ts:831-881` — `parseSSEResponse` chunk-split discipline.
- `frontend/src/lib/api.ts:896-977` — `subscribeWorkspaceSyncStream` (SSR guard at line 901; `{ withCredentials: true }` at line 907; auto-close on terminal status + null marker).
- `frontend/src/lib/api.sse.spec.ts:75-307` — `FakeES` pattern the chat specs mirror.
- `frontend/src/lib/csrf.ts:28-39` — `stateChangingFetch` for POST/DELETE.
- `frontend/src/lib/markdown.ts:65-67` — `renderSanitizedMarkdown`.
- `frontend/src/components/workspace-sync-card/use-sync-status.ts:65-141` — the canonical "subscribe to a server-pushed event stream" hook pattern (`useVisibleTask$` + `cleanup()`).
- `frontend/src/routes/home/index.tsx:39-51` — `onRequest` guard chain (`setSsrCookieHeader` → `requireAuthRedirect` → `requireOwnboarding`).
- `frontend/src/components/avatar-dropdown/avatar-dropdown.tsx` — chosen navigation component for T-09 (overrides design.md's `components/example/example.tsx` candidate which was not actually a sidebar).

### 7.5 Repo conventions

- `openspec/AGENTS.md` — strict TDD (`openspec/config.yaml` `apply.tdd: true`); vitest is the frontend red-green target. Hexagonal layout for `database_administrator` and `workspace_syncer`; `backend/agent` is layered (not hexagonal) per ADR 0005.
- `openspec/config.yaml` `rules.specs` — RFC 2119 keywords, Given/When/Then scenarios.
- `openspec/changes/cachicamas-frontend-chat-layer1/proposal.md §6` — locked `delivery_strategy = single-pr`, `review_budget_lines = 800`, `strict_tdd = true`.

## §8. Sign-off

| Role | Agent / human | Status |
|---|---|---|
| Orchestrator | (parent agent) | Archive phase complete; ready for `git push` / PR creation per repo delivery policy. The orchestrator handles push and PR — this archive is the close-out. |
| Author (apply) | `sdd-apply-cachicamas-frontend-chat-layer1` sub-agent | T-01..T-10 complete; T-08 QRL prop fix committed at `104a73b`; T-10 verification receipt at `6379fe2`. (Apply self-report superseded by `verify-report` independent audit.) |
| Verifier | `sdd-verify-cachicamas-frontend-chat-layer1` sub-agent | PASS. `requirements: 7/7`, `scenarios: 18/18`, `blockers: 0`, `critical_findings: 0`, `strict_tdd: 6/6`. Two non-blocking WARNINGS documented and orchestrator-accepted. |
| Archivist | `sdd-archive-cachicamas-frontend-chat-layer1` sub-agent (this run) | PASS. Spec sync verified (canonical at `openspec/specs/frontend-chat-layer1/spec.md`); change folder moved to `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/`; `tasks.md` checkboxes reconciled from apply's persisted state per the Task Completion Gate's exceptional mechanical reconciliation clause (proof: orchestrator launch handoff + apply-progress #2585 + verify-report #2600 PASS); this archive-report written to both OpenSpec and Engram. |

**SDD Cycle**: closed. Next change can begin.

---

## §9. Archive folder manifest

```
openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/
├── explore.md         24,666 bytes  (frontend primitives survey + backend gap)
├── proposal.md        25,815 bytes  (D1–D7, success criteria)
├── design.md          15,320 bytes  (typed ChatStreamEvent, signal graph, cancel race)
├── tasks.md           16,659 bytes  (T-01..T-10, all checkboxes [x] after reconciliation)
├── verify-report.md   32,776 bytes  (independent audit PASS; 7/7 REQs, 18/18 scenarios)
└── archive-report.md  (this file)  (closure with final-state facts and lineage)
```

**Active changes directory** (`openspec/changes/` excluding `archive/`): empty — this change is the only active change and has now been moved.

**Canonical spec** (`openspec/specs/frontend-chat-layer1/spec.md`): unchanged at archive; 149 lines; SHA `4bb3d45039da4b38104fd61496503c7eee617254`.

---

**Archive report SHA** (this file): saved to `openspec/changes/archive/2026-08-06-cachicamas-frontend-chat-layer1/archive-report.md` and mirrored to Engram at topic_key `sdd/cachicamas-frontend-chat-layer1/archive-report`, type `architecture`, scope `project`, `capture_prompt: false`.
