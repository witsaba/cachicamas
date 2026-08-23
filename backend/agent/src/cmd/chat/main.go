// Package main is the chat archetype's composition root, created by
// CH-01.1 of doc 0005 — the module's first package main of any kind. It
// is the only package of this archetype permitted to install the
// OpenTelemetry SDK and its exporters: ADR 0005 § D3's cmd/ column
// (adr:242) grants that generically to every composition root and needs
// no ADR 0009 § D2 substitution to reach this one — CH-00's own seam row
// 8 already states it: "CH-04.1 is the only package in this archetype
// permitted to install the OpenTelemetry SDK."
//
// This package is deliberately outside the scanned set of
// backend/agent/src/agent/import_boundary_test.go's check 6
// (TestChatArchetype_ProductionSources_ImportsOnlyAllowedPrefixes_DenyByDefault):
// sweeping it would deny the very import this position exists to grant.
//
// func main below is intentionally empty. CH-04.1 owns its wiring —
// environment reads, the OTel SDK install, and the HTTP listener are all
// out of scope here (0005:240 puts behaviour out of scope for CH-01). A
// binary that runs and does nothing is a recorded decision (proposal
// D-5): the rejected alternative was a fail-fast stub writing to stderr
// and exiting non-zero — more honest to a human running the binary, but
// itself behaviour.
package main

func main() {
	// Intentionally empty. See the package comment above: CH-04.1 wires
	// this composition root — environment reads, the OpenTelemetry SDK
	// install, and the HTTP listener. Nothing runs here until then.
}
