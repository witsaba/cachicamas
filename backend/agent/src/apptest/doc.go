// Package apptest is AG-23's packaged test kit: an importable scripting
// substrate — shipped as ordinary production source, never declared
// inside a Go test-only compilation unit — that a genuinely third-party
// consumer package needs to build and drive an [agent.Harness] — a
// scriptable [agent.PermissionPolicy] ([ScriptedPermissionPolicy]), a
// scriptable [agent.Tool] ([ScriptedTool]), and a drain helper
// ([DrainAndCheck]) that delegates validation wholesale to
// [agent.CheckStream].
//
// It is named by the same discipline as [agenttest] — for the layer
// that consumes it: agenttest is Layer 1's own external-consumer kit,
// for the agent layer; apptest is the same role one layer up, for the
// application layer. It cannot be built by widening agenttest itself:
// any [agent.PermissionPolicy] or [agent.Tool] implementation
// necessarily references Layer 2 types, which Layer 1's own import
// guard denies by name, and agenttest is separately frozen
// byte-unchanged by a shipped scope fence.
//
// This package performs no process, network, storage or wall-clock I/O
// of any kind — proven mechanically, not merely documented, by the
// import-boundary guard's fifth check (agent-package-scaffold's
// R-AGP-003). See this change's decision.md for the frozen v1 surface
// this kit scripts against.
package apptest
