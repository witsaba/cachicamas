// Package chat is the chat archetype's own package: the Layer 3 position
// (ADR 0009 § D2) `cachicamas_chat` occupies. It was created by CH-01.1 of
// doc 0005 (`chat-archetype-contract` R-CHT-007) carrying no declarations
// beyond this comment. CH-02 filled it with its first policy: a
// Conversation that owns one agent.Harness and one *agent.History,
// driving one browser turn per Harness.Run call and projecting Layer 2's
// event stream onto a Go wire vocabulary — split across conversation.go
// (Conversation, Config, Send, Cancel), wire.go (the wire vocabulary),
// projection.go (the event projector) and errortext.go (the typed-failure
// phrase table).
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
// Permitted OpenTelemetry surface: the tracing API —
// go.opentelemetry.io/otel/trace, /attribute, /codes and the forced
// closure of otel/trace — and the go.opentelemetry.io/contrib/bridges/otelslog
// bridge (ADR 0005 § D3, adr:243, ✅ at L3 coding), read under the ADR
// 0009 § D2 substitution as this archetype's own grant.
//
// Both closures are enforced mechanically, not by this comment alone:
// backend/agent/src/agent/import_boundary_test.go's check 6
// (TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault)
// is the deny-by-default enforcement.
package chat
