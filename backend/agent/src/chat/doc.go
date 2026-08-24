// Package chat is the chat archetype's own package: the Layer 3 position
// (ADR 0009 § D2) `cachicamas_chat` occupies. It was created by CH-01.1 of
// doc 0005 (`chat-archetype-contract` R-CHT-007) carrying no declarations
// beyond this comment.
//
// Layer-2-filling policy milestones (chronological):
//
//   - CH-02 (`cachicamas-chat-conversation`): a Conversation that owns
//     one agent.Harness and one *agent.History, driving one browser
//     turn per Harness.Run call and projecting Layer 2's event stream
//     onto a Go wire vocabulary — split across conversation.go
//     (Conversation, Config, Send, Cancel), wire.go (the wire
//     vocabulary), projection.go (the event projector) and
//     errortext.go (the typed-failure phrase table).
//
//   - CH-03 (`cachicamas-chat-http-surface`): the HTTP+SSE serving
//     surface that `frontend-chat-layer1`'s frozen wire contract
//     drives. Three new files join the package:
//       identity.go      — the `IdentityResolver` port and a
//                          NoopIdentityResolver for tests.
//       registry.go      — a mutex-guarded `map[participantID]*Conversation`
//                          plus a per-turn stream lookup keyed by
//                          turnID; one *Conversation per participant
//                          (D2, mirrors CH-02 R-CHS-001 / R-CCP-001),
//                          409 on concurrent POSTs (S-CHS-001.c).
//       eventsource.go   — channel-driven `fmt.Fprintf` + `http.Flusher`
//                          SSE writer mirroring
//                          database_administrator's `sync_stream_handler.go`;
//                          writes `event: <name>\ndata: <json>\n\n`
//                          bytes-exact against frontend-chat-layer1's
//                          wire (frontend/src/lib/chat-types.ts:158-178).
//       http.go          — Echo v5 handlers mounted by
//                          RegisterRoutes(e *echo.Echo, resolver
//                          IdentityResolver, newConv
//                          ConversationFactory) *Registry; the
//                          composition root (CH-04) calls this with
//                          the production wiring.
//
//     The byte-exact projection invariant and the cancel discriminator
//     (turn.end with `FinishReason` absent for cancelled turns) live
//     here; the wire vocabulary in wire.go stays the frozen Layer-2
//     contract.
//
// Forbidden closure (ADR 0005 § D1 row 3, read as any Layer 3 archetype
// under the ADR 0009 § D2 substitution): the OpenTelemetry SDK and its
// exporters anywhere below the composition root (ADR 0005 § D3, adr:242
// — those belong to backend/agent/src/cmd/chat alone); any
// backend/agent/src/cmd/... package; any Go package of
// github.com/cachicamas/backend/database_administrator or
// github.com/cachicamas/backend/workspace_syncer (network-only backend
// modules, R-07); Layer 2's own test substrate
// (backend/agent/src/agenttest, backend/agent/src/apptest,
// backend/agent/src/layer3handoff); and the vendor adapter subtree
// (backend/agent/src/ai/openaicompat).
//
// Permitted surface for v1 (CH-03 widens this with Echo v5 + its
// measured transitive closure — see engram adr/echo-v5-in-agent-module
// for the recorded ADR; the deny-by-default check 6 keeps anything
// else out):
//   - Layer 2 packages src/agent, src/ai (vendor adapter subtree
//     carved out by the closure above).
//   - OpenTelemetry tracing API (ADR 0005 § D3, adr:240-241).
//   - The `otelslog` bridge (ADR 0005 § D3, adr:243, ✅ at L3 coding)
//     under the ADR 0009 § D2 substitution.
//   - github.com/labstack/echo/v5 v5.2.1 and its measured transitive
//     closure (CH-03 ADR `adr/echo-v5-in-agent-module`).
//
// Both closures are enforced mechanically, not by this comment alone:
// backend/agent/src/agent/import_boundary_test.go's check 6
// (TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault)
// is the deny-by-default enforcement over the per-file AST scan; check
// 7 (TestChatArchetype_ProductionClosure_ImportsOnlyAllowedPrefixes_DenyByDefault)
// is the transitive closure check that admits Echo's measured deps.
//
// Composition root (CH-04): backend/agent/src/cmd/chat/main.go reads
// the environment, installs the OpenTelemetry SDK, constructs the
// production IdentityResolver shim over `database_administrator`'s
// auth middleware (cross-module per R-07 stays network-only, so the
// shim is implemented as an in-process adapter in src/cmd/chat/ alone),
// then calls chat.RegisterRoutes. CH-04 owns that wiring; CH-03 ships
// the surface it mounts.
package chat
