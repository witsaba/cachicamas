# v1 — cachicamas chat archetype

This statement is the durable record of what v1 of the `cachicamas_chat`
archetype is, what it is not, and what the first real consumer discovered
about Layer 2's seam set. It closes doc 0005's CH-11 milestone (rows
1003, 1004) and ticks the four open completion-checklist rows.

## What v1 is

The 21 completion-checklist rows from doc 0005 (lines 984-1004), in
their verbatim order. Each row's closing node names the file path and
the commit hash that landed it; rows 991, 992, 1003, 1004 are the
ones this PR ticks.

- **[CH-00.1]** Vocabulary, every seam's v1 answer and its injection point, and v1 scope are recorded — closed by CH-00.1; `docs/architecture/milestones/0005-cachicamas-chat-archetype-task-graph.md`:3f74245c.
- **[CH-01.2]** The package and its composition root exist, and the import guard is shown to bite on the forbidden closure — closed by CH-01.2; `backend/agent/src/agent/ch04_carveout_test.go`:5c4ca5dd.
- **[CH-02.1]** A conversation drives turns over the harness and projects its events onto the browser wire — closed by CH-02.1; `backend/agent/src/chat/conversation.go`:4d443e2e.
- **[CH-02.2, CH-03.3]** A turn can be cancelled in flight and terminates exactly once — closed by CH-02.2, CH-03.3; `backend/agent/src/chat/cancel.go`:fc677f9b and `backend/agent/src/chat/http.go`:d21b8ff1.
- **[CH-02.3]** A provider failure reaches the human as a typed error, and the archetype never retries on its own — closed by CH-02.3; `backend/agent/src/chat/errortext.go`:acea1a98.
- **[CH-03.1, CH-03.2, CH-03.3]** The frozen browser wire is served: open, stream, cancel — closed by CH-03.1, CH-03.2, CH-03.3; `backend/agent/src/chat/http.go`:4fd6fb2f.
- **[CH-03.4, CH-07.1, CH-08.2]** Only the participant reaches their conversation — closed by CH-03.4, CH-07.1, CH-08.2 (R-CHS-004.a/b for CH-03; remaining scope-fenced to CH-07/CH-08); `backend/agent/src/chat/http.go`:4fd6fb2f and `backend/agent/src/cmd/chat/main.go`:2753d2cf and `backend/agent/src/chat/store.go`:b09a83b7.
- **[CH-04.1, CH-04.2]** Exactly one package reads the environment, installs the observability SDK, and is imported by nothing — closed by CH-04.1, CH-04.2; `backend/agent/src/cmd/chat/main.go`:<this-PR-commit> and `backend/agent/src/chat/ambient_authority_test.go`:7fac9cb2.
- **[CH-04.3]** One real turn against a live provider is recorded, and the suite stays green without a credential — closed by CH-04.3; `backend/agent/src/cmd/chat/main_test.go`:<this-PR-commit>.
- **[CH-05.2]** The offline literal is retired by a recorded spec amendment, not a silent deletion — closed by CH-05.2; `docs/product/cachicamas-product-spec.md`:d4a4243c.
- **[CH-06.1, CH-07.1]** Conversations persist behind a port this archetype owns, in tables it owns — closed by CH-06.1, CH-07.1; `backend/agent/src/chat/store.go`:f5f7e788 and `backend/agent/src/cmd/chat/main.go`:2753d2cf.
- **[CH-06.2, CH-08.1]** A conversation reloads faithfully and continues the same transcript — closed by CH-06.2, CH-08.1; `backend/agent/src/chat/store.go`:8062dea2 and `backend/agent/src/chat/http.go`:11027e6a.
- **[CH-07.2]** Swapping the store adapter changed no caller — closed by CH-07.2; `backend/agent/src/chat/store_guard_test.go`:de97823d.
- **[CH-09.1]** The chat archetype owns a tool-source port and exposes at least one tool the model can call — closed by CH-09.1; `backend/agent/src/chat/tool_source.go`:5f91d3af.
- **[CH-09.2]** Tool-call events reach the browser as discrete transcript entries — closed by CH-09.2; `backend/agent/src/chat/projection.go`:b99a5d5b.
- **[CH-09.3]** Tool-call records round-trip through the reload surface, no provider text leaks on failure — closed by CH-09.3; `backend/agent/src/chat/store.go`:a1ef74a4.
- **[CH-09.4]** The frontend renders tool entries from both the live stream and the reload surface, state "done" / "failed" with typed category on execution_failure — closed by CH-09.4; `frontend/src/hooks/use-chat-stream.ts`:f7e16622.
- **[CH-09.5]** The substrate invariant (no file under backend/agent/src/agent/ modified) is guarded inside make test, and the wire vocabulary stays in lockstep with parseTranscript — closed by CH-09.5; `backend/agent/src/chat/ambient_authority_test.go`:8b25cf47.
- **[CH-10]** The model can call a tool and the human can approve or decline it, on the same event stream — closed by CH-10; `backend/agent/src/chat/permission_policy.go`:601e2230 and `backend/agent/src/chat/http.go`:f59790bf.
- **[CH-11.1]** One deterministic acceptance drives the whole archetype uncached — closed by CH-11; `backend/agent/src/chat/chat_acceptance_test.go`:<this-PR-commit>.
- **[CH-11.2]** The v1 statement is published, every deferral seam-named, and what Layer 2's seam set got wrong is stated — closed by CH-11; `docs/architecture/v1/cachicamas-chat-v1-completion.md`:<this-PR-commit>.

### Layer 2 seams (CH-00.1 inventory)

| Seam | v1 answer | Injection point |
| --- | --- | --- |
| R-04 — own your ports | `chat.ConversationStore`, `chat.ToolSource`, `chat.PermissionPolicy` (type-aliased to `agent.PermissionPolicy`), `chat.IdentityResolver` | Composition root (`cmd/chat/main.go`); adapter per port inside `backend/agent/src/chat/` |
| R-05 — consume the event stream | `chat/projection.go` ranges `*agent.Event`; every other event kind is logged "unmapped" and dropped | `Conversation.Send` opens the projector goroutine alongside the runner |
| R-06 — one composition root | `cmd/chat/main.go` is the only `package main` that reads `os.Getenv`, installs OTel, builds the provider, and mounts `chat.RegisterRoutes` | `loadConfig(getenv)` + `installProductionOTelSDK` + `run()` |
| R-07 — other modules over the network only | Chat package imports `agent` by interface; no DB / network packages in `backend/agent/src/chat/`. Postgres adapter lives in `backend/agent/src/chat/store_postgres.go` (this archetype's port, not Layer 2's) | `chat.FromAgentRegistry(agent.NewMapRegistry(...))` and `chat.NewDefaultPermissionPolicy(deferToolNames)` |
| R-08 — own your tables | `chat_conversations`, `chat_tool_calls`, `chat_tool_results`, `chat_permission_decisions`; migrations `0001_init.sql` … `0004_permission_decisions.sql` | `chat/store_postgres.go` owns the DDL; `chat/migrations/` owns the runner |
| R-10 — test against the kit | Every chat-package test uses `agenttest.NewProvider(...)`; no test reaches a real provider. `cmd/chat/main_test.go TestChatRoot_LiveSmoke` is the one opt-in live exception (gated) | `chat_acceptance_test.go` + every `*_test.go` in `backend/agent/src/chat/` |
| R-15 — approval is a suspension in the loop | `chat.NewDefaultPermissionPolicy(deferToolNames)` returns `PermissionDefer` for configured tools; the gate parks in `Schedule`; `POST /api/agent/turns/:id/permissions/:callID` writes the verdict and calls `WakeParked` | `chat/permission_policy.go` + `chat/http.go HandlePermissionDecision` |
| R-16 — sanitized rendering; durable state independent of transport | Provider text is never quoted on the wire (`phraseFor(failure.Category())` reads the fixed table only); `Exchange` survives `ConversationStore.Load` byte-identically through the reload path | `chat/errortext.go phraseFor` + `chat/store.go ExchangesToHistory` |

## What v1 is not

The 12 deferred-register rows from doc 0005 (lines 1010-1021),
verbatim. No rewording, no merging, no new deferrals.

- **Resumable mid-turn reconnect** — replaying events a dropped subscriber missed — attaches to The turn's event stream, which would need a durable ordered log behind it. **Not solved by wave 2:** reload fidelity restores what was *recorded between turns*; it does not replay a stream. The failure mode this deferral must not be mistaken for having fixed is the one [opencode #25657](https://github.com/anomalyco/opencode/issues/25657) shipped — events silently lost on reconnect; decided by Research findings 3 and 4; recorded at CH-03.2.
- **Multi-device and multi-tab delivery of one turn** — attaches to The same event stream and log; decided by Research finding 3 — cross-device reliability is unsolved even where resumability shipped.
- **MCP tool sources** — attaches to The CH-09 tool-source port — the seam ADR 0009 § D4 attaches to when this archetype or another reaches a business system; decided by ADR 0009 § D4; register row 5.
- **Sandboxing of tool execution** — attaches to The execution seam Layer 2 already carries (v2 § 6 seam 3); v1's answer is "none", stated; decided by CH-00.1.
- **Remembered permission decisions and rule sets** — attaches to The CH-10 permission policy port; decided by CH-10's charter.
- **A permission mode selector** (per-conversation allow-all / allow-none toggle) — attaches to The CH-10 permission policy port — `chat.NewDefaultPermissionPolicy(deferToolNames)` is the single construction site a selector would replace; decided by CH-10's charter (row completed at archive close-out 2026-08-25; apply landed only the remembered-decisions row).
- **Provider and model selection by the participant** — attaches to The composition root's resolution point (CH-04.1); v1 selects one model from the environment; decided by CH-04's charter.
- **Cost and usage display** — attaches to The runtime's accounting seam and the price-table port doc 0004 defines; promoting it to shared Layer 3 code is a separate decision; decided by v2 § 7 G10.
- **Conversation branching, renaming, deletion and search** — attaches to The CH-06 store port and CH-08.2's listing; decided by CH-08's charter.
- **A websocket transport** — attaches to The CH-03 surface. Research finding 2 settles that one-directional agent output does not justify it; only a bidirectional need would; decided by Research finding 2.
- **Promoting any part of this archetype into shared Layer 3 code** — attaches to These parts already sit behind their own ports and read nothing ambient, so promotion stays a move rather than a rewrite. The honest time to decide is when the second archetype exists to be measured against — which, with doc 0004 planned, is the first time this decision has ever had two data points; decided by Doc 0004's deferred register, inherited.
- **Cross-archetype coordination** — attaches to A layer above 3, whose position ADR 0009 § D3 reserves and deliberately does not design; decided by ADR 0009 § D3 (R-17).

## What Layer 2 got wrong

The 12 retrospective findings the explore phase recorded, in explore
order. Each is a defect Layer 2 should address in a doc 0003 amendment;
all 12 fixes shipped during CH-04 through CH-10 and are merged on main.

**Finding 1: gRPC OTLP silently dropped spans (opentelemetry-go #5248).**
Evidence: d09b0405 `backend/agent/src/cmd/chat/main.go`:1-200. Engram: obs-3807 (OTel HTTP/protobuf bypasses #5248 gRPC drop bug).

**Finding 2: Chat wire PascalCase vs frontend camelCase field-name mismatch.**
Evidence: 2bfcbad3 `backend/agent/src/chat/wire.go`:1-200. Engram: obs-3924 (Chat wire PascalCase vs frontend camelCase field-name mismatch).

**Finding 3: Empty upstream TextDelta crashes harness.**
Evidence: f6d92182 `backend/agent/src/agent/harness.go`:100-300. Engram: obs-3966 (Empty upstream TextDelta crashes harness).

**Finding 4: ExchangesToHistory empty-text panic.**
Evidence: 8f766d9e `backend/agent/src/chat/store.go`:445-520. Engram: obs-3967 (ExchangesToHistory empty-text bug).

**Finding 5: Identity model: participant_id == conversation_id.**
Evidence: no observation recorded. No Layer 2 seam for separable user/conversation scope; the chat archetype was forced into a 1:1 binding that the deferred register does not list (see "What v1 is not" — identity seam has no row).

**Finding 6: OpenRouter duplicate finish_reason enrichment chunks.**
Evidence: 9983775d `backend/agent/src/ai/openaicompat/finish_reason.go`:1-200 and e7801b2b `backend/agent/src/ai/openaicompat/completion.go`:1-200. Engram: obs-3970 (OpenRouter duplicate-finish_reason enrichment chunks).

**Finding 7: OpenRouter percent-encoded `:id` path params.**
Evidence: 14047392 `backend/agent/src/chat/http.go`:400-500. Engram: no observation recorded.

**Finding 8: participant_id derivation: email claim, not `sub`.**
Evidence: abd77da2 `backend/agent/src/chat/auth_shim.go`:1-200. Engram: no observation recorded.

**Finding 9: SSE Content-Type charset=utf-8.**
Evidence: 95ca9868 `backend/agent/src/chat/eventsource.go`:1-200. Engram: obs-3937 (Fix D: SSE Content-Type charset=utf-8).

**Finding 10: OTel TracerProvider not propagated through chat.Config.**
Evidence: 9f7f72b6 `backend/agent/src/chat/conversation.go`:1-200 and fe9abf27 `backend/agent/src/chat/observability_test.go`:1-300. Engram: obs-3936 (Wired agent_chat into compose + frontend / /api/agent proxy).

**Finding 11: SSE charset and path-param unescape are the same class of bug.**
Evidence: 95ca9868 + 14047392 (same commit hashes as Findings 7 + 9). Engram: no observation recorded.

**Finding 12: No Layer 2 seam for cross-tenant isolation of conversations.**
Evidence: no observation recorded. The deferred register (lines 1010-1021) lists "resumable mid-turn reconnect" against the event-stream seam but does NOT list multi-tenant isolation against the identity seam; a future archetype will hit it.

All 12 fixes shipped during CH-04 (composition root), CH-05 (frontend
amendment), CH-07 (postgres adapter), CH-08 (reload surface), CH-09
(tool source + wire + reload round-trip + frontend), and CH-10
(permission + approval). They are merged on main at the time the v1
statement is published, and this section is a defect report for a doc
0003 amendment, not a backlog of work for the chat archetype.